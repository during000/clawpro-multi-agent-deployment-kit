package provider

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	hcommon "hatchery/common"
)

// ctxWithSSRFEnabled 返回一个启用了 SSRF 安全策略的 context。
func ctxWithSSRFEnabled() context.Context {
	return hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{SecurityPolicies: []string{"SSRF"}})
}

func TestIsDisallowedIP(t *testing.T) {
	cases := []struct {
		ip      string
		want    bool
		comment string
	}{
		{"127.0.0.1", true, "loopback"},
		{"127.0.0.53", true, "loopback range"},
		{"0.0.0.0", true, "unspecified"},
		{"10.0.0.1", true, "rfc1918 10/8"},
		{"172.16.0.1", true, "rfc1918 172.16/12"},
		{"172.31.255.254", true, "rfc1918 172.16/12 upper"},
		{"192.168.1.1", true, "rfc1918 192.168/16"},
		{"169.254.169.254", true, "AWS/Tencent metadata"},
		{"100.100.100.200", true, "Alibaba metadata"},
		{"169.254.0.1", true, "link-local"},
		{"100.64.0.1", true, "CGNAT"},
		{"100.127.255.254", true, "CGNAT upper"},
		{"224.0.0.1", true, "multicast"},
		{"255.255.255.255", true, "broadcast"},
		{"240.0.0.1", true, "reserved"},
		{"::1", true, "ipv6 loopback"},
		{"fc00::1", true, "ipv6 ULA"},
		{"fd12:3456:789a::1", true, "ipv6 ULA"},
		{"fe80::1", true, "ipv6 link-local"},
		{"ff02::1", true, "ipv6 multicast"},
		{"fd00:ec2::254", true, "AWS IMDSv6"},

		// Public addresses should pass.
		{"8.8.8.8", false, "google dns"},
		{"1.1.1.1", false, "cloudflare dns"},
		{"99.83.190.102", false, "public"},
		{"2606:4700:4700::1111", false, "cloudflare ipv6"},
		// CGNAT boundary just below — 100.63 belongs to AT&T public space.
		{"100.63.255.254", false, "below CGNAT"},
		{"100.128.0.1", false, "above CGNAT"},
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("invalid test ip %q", c.ip)
		}
		if got := isDisallowedIP(ip); got != c.want {
			t.Errorf("isDisallowedIP(%s) = %v, want %v (%s)", c.ip, got, c.want, c.comment)
		}
	}
}

func TestValidateOutboundURL_BlockedSchemes(t *testing.T) {
	for _, raw := range []string{
		"file:///etc/passwd",
		"gopher://127.0.0.1:11211/_stats",
		"ftp://example.com/",
		"ldap://internal/",
	} {
		err := validateOutboundURL(context.Background(), raw)
		if err == nil {
			t.Errorf("expected SSRF block for %q, got nil", raw)
			continue
		}
		if !errors.Is(err, errSSRFBlocked) {
			t.Errorf("expected errSSRFBlocked for %q, got %v", raw, err)
		}
	}
}

func TestValidateOutboundURL_BlockedLiteralIPs(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1/",
		"https://10.0.0.5:8443/v1/models",
		"http://169.254.169.254/latest/meta-data/",
		"http://[::1]/",
		"http://[fc00::1]/",
		"https://192.168.0.1/",
		"http://100.64.1.2/",
	} {
		err := validateOutboundURL(context.Background(), raw)
		if err == nil {
			t.Errorf("expected SSRF block for %q, got nil", raw)
			continue
		}
		if !errors.Is(err, errSSRFBlocked) {
			t.Errorf("expected errSSRFBlocked for %q, got %v", raw, err)
		}
	}
}

func TestValidateOutboundURL_AllowsPublicLiteralIP(t *testing.T) {
	if err := validateOutboundURL(context.Background(), "https://1.1.1.1/"); err != nil {
		t.Fatalf("expected public IP to be allowed, got %v", err)
	}
}

func TestValidateOutboundURL_EmptyHost(t *testing.T) {
	err := validateOutboundURL(context.Background(), "http:///path")
	if err == nil || !errors.Is(err, errSSRFBlocked) {
		t.Fatalf("expected errSSRFBlocked for empty host, got %v", err)
	}
}

func TestOpenAICheckConnectivityWithChat_SSRFBlocked(t *testing.T) {
	ctx := ctxWithSSRFEnabled()
	p := &OpenAIProvider{}
	for _, base := range []string{
		"http://127.0.0.1:8080",
		"http://10.0.0.1",
		"http://localhost",
		"file:///etc/passwd",
	} {
		_, err := p.CheckConnectivityWithChat(ctx, "sk", base, "gpt-4o-mini")
		if err == nil {
			t.Errorf("base=%q: expected SSRF block, got nil", base)
			continue
		}
		if !errors.Is(err, ErrNetworkUnreachable) {
			t.Errorf("base=%q: err = %v, want ErrNetworkUnreachable", base, err)
		}
		var ce *ConnectivityError
		if !errors.As(err, &ce) || ce.Cause == nil || !errors.Is(ce.Cause, errSSRFBlocked) {
			t.Errorf("base=%q: expected ConnectivityError wrapping errSSRFBlocked, got %v", base, err)
		}
	}
}

func TestAnthropicCheckConnectivityWithChat_SSRFBlocked(t *testing.T) {
	ctx := ctxWithSSRFEnabled()
	p := &AnthropicProvider{}
	for _, base := range []string{
		"http://127.0.0.1:9090",
		"http://192.168.1.1",
		"http://[::1]:8080",
	} {
		_, err := p.CheckConnectivityWithChat(ctx, "sk", base, "claude")
		if err == nil {
			t.Errorf("base=%q: expected SSRF block, got nil", base)
			continue
		}
		if !errors.Is(err, ErrNetworkUnreachable) {
			t.Errorf("base=%q: err = %v, want ErrNetworkUnreachable", base, err)
		}
		var ce *ConnectivityError
		if !errors.As(err, &ce) {
			t.Errorf("base=%q: err is not *ConnectivityError: %v", base, err)
			continue
		}
		if ce.Cause == nil || (!errors.Is(ce.Cause, errSSRFBlocked) && !strings.Contains(ce.Cause.Error(), "ssrf")) {
			t.Errorf("base=%q: expected SSRF cause, got %v", base, ce.Cause)
		}
	}
}

// ---------------------------------------------------------------------------
// Additional coverage for isDisallowedIP edge cases
// ---------------------------------------------------------------------------

// TestIsDisallowedIP_Nil ensures a nil IP is treated as disallowed so that
// callers cannot bypass validation by passing a parse failure through.
func TestIsDisallowedIP_Nil(t *testing.T) {
	if !isDisallowedIP(nil) {
		t.Fatalf("isDisallowedIP(nil) = false, want true")
	}
}

// TestIsDisallowedIP_IPv4MappedIPv6 ensures that IPv4 addresses tunnelled
// through IPv6 syntax (::ffff:a.b.c.d) are normalised back to IPv4 and
// rejected if the underlying v4 address is internal. This is a classic
// SSRF bypass technique.
func TestIsDisallowedIP_IPv4MappedIPv6(t *testing.T) {
	cases := []struct {
		ip   string
		want bool
	}{
		{"::ffff:127.0.0.1", true},
		{"::ffff:10.0.0.1", true},
		{"::ffff:169.254.169.254", true},
		{"::ffff:192.168.1.1", true},
		{"::ffff:8.8.8.8", false},
		{"::ffff:1.1.1.1", false},
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("invalid test ip %q", c.ip)
		}
		if got := isDisallowedIP(ip); got != c.want {
			t.Errorf("isDisallowedIP(%s) = %v, want %v", c.ip, got, c.want)
		}
	}
}

// TestIsDisallowedIP_RangeBoundaries verifies the edges of every numeric
// range so we catch off-by-one errors in the bitmask arithmetic.
func TestIsDisallowedIP_RangeBoundaries(t *testing.T) {
	cases := []struct {
		ip      string
		want    bool
		comment string
	}{
		// 172.16/12 boundaries — IsPrivate handles this.
		{"172.15.255.255", false, "just below 172.16/12"},
		{"172.16.0.0", true, "172.16/12 lower"},
		{"172.31.255.255", true, "172.16/12 upper"},
		{"172.32.0.0", false, "just above 172.16/12"},

		// 192.168/16 boundaries.
		{"192.167.255.255", false, "just below 192.168/16"},
		{"192.168.0.0", true, "192.168/16 lower"},
		{"192.168.255.255", true, "192.168/16 upper"},
		{"192.169.0.0", false, "just above 192.168/16"},

		// 10/8 boundaries.
		{"9.255.255.255", false, "just below 10/8"},
		{"10.0.0.0", true, "10/8 lower"},
		{"10.255.255.255", true, "10/8 upper"},
		{"11.0.0.0", false, "just above 10/8"},

		// CGNAT 100.64/10 boundaries.
		{"100.63.255.255", false, "just below 100.64/10"},
		{"100.64.0.0", true, "100.64/10 lower"},
		{"100.127.255.255", true, "100.64/10 upper"},
		{"100.128.0.0", false, "just above 100.64/10"},

		// 192.0.0/24 vs sibling 192.0.1/24 (the latter is public).
		{"192.0.0.255", true, "192.0.0/24 upper"},
		{"192.0.1.0", false, "just above 192.0.0/24"},

		// 198.18/15 benchmarking boundaries.
		{"198.17.255.255", false, "just below 198.18/15"},
		{"198.18.0.0", true, "198.18/15 lower"},
		{"198.19.255.255", true, "198.18/15 upper"},
		{"198.20.0.0", false, "just above 198.18/15"},

		// 240/4 reserved boundaries.
		{"239.255.255.255", true, "still multicast 224/4"},
		{"240.0.0.0", true, "240/4 lower"},
		{"254.255.255.255", true, "240/4 upper interior"},

		// Documentation prefixes.
		{"192.0.2.0", true, "TEST-NET-1 lower"},
		{"192.0.2.255", true, "TEST-NET-1 upper"},
		{"198.51.100.128", true, "TEST-NET-2"},
		{"203.0.113.7", true, "TEST-NET-3"},
		{"203.0.114.7", false, "just outside TEST-NET-3"},
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("invalid test ip %q", c.ip)
		}
		if got := isDisallowedIP(ip); got != c.want {
			t.Errorf("isDisallowedIP(%s) = %v, want %v (%s)", c.ip, got, c.want, c.comment)
		}
	}
}

// TestIsDisallowedIP_IPv6Boundaries checks specific IPv6 ranges in addition
// to the basic sanity cases already covered by TestIsDisallowedIP.
func TestIsDisallowedIP_IPv6Boundaries(t *testing.T) {
	cases := []struct {
		ip      string
		want    bool
		comment string
	}{
		// ULA fc00::/7 covers fc00:: through fdff::.
		{"fc00::", true, "ULA lower"},
		{"fdff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", true, "ULA upper"},
		// fe00:: is outside ULA but is reserved/unassigned — Go's IsPrivate
		// returns false; we don't currently reject it. Document via false.
		{"fe00::1", false, "outside ULA, currently allowed"},
		// fe80::/10 link-local handled by IsLinkLocalUnicast.
		{"fe80::1", true, "link-local lower"},
		{"febf:ffff:ffff:ffff::1", true, "link-local upper"},
		// 2001:db8::/32 documentation.
		{"2001:db8::", true, "TEST docs lower"},
		{"2001:db8:ffff:ffff:ffff:ffff:ffff:ffff", true, "TEST docs upper"},
		{"2001:db9::", false, "just outside TEST docs"},
		// Multicast ff00::/8.
		{"ff00::1", true, "multicast"},
		{"ff15::1234", true, "site-local multicast"},
		// Public IPv6.
		{"2606:4700::1", false, "cloudflare"},
		{"2400:cb00::", false, "public"},
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("invalid test ip %q", c.ip)
		}
		if got := isDisallowedIP(ip); got != c.want {
			t.Errorf("isDisallowedIP(%s) = %v, want %v (%s)", c.ip, got, c.want, c.comment)
		}
	}
}

// ---------------------------------------------------------------------------
// validateOutboundURL cases
// ---------------------------------------------------------------------------

// TestValidateOutboundURL_AlwaysBlocksInternal verifies that validateOutboundURL
// always validates URLs (there is no longer a global guard; the decision to call
// validateOutboundURL is made by the caller via context-based SSRF policy).
func TestValidateOutboundURL_AlwaysBlocksInternal(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1/",
		"http://169.254.169.254/",
		"file:///etc/passwd",
	} {
		if err := validateOutboundURL(context.Background(), raw); err == nil {
			t.Errorf("validateOutboundURL(%q) = nil, want error", raw)
		}
	}
	// Malformed URLs should also be blocked.
	for _, raw := range []string{
		"not a url at all",
		"",
	} {
		if err := validateOutboundURL(context.Background(), raw); err == nil {
			t.Errorf("validateOutboundURL(%q) = nil, want error", raw)
		}
	}
}

// TestValidateOutboundURL_RejectsCustomSchemes covers a mixed bag of
// non-https schemes that have all been used in real-world SSRF exploits.
func TestValidateOutboundURL_RejectsCustomSchemes(t *testing.T) {
	for _, raw := range []string{
		"dict://1.1.1.1/",
		"jar:https://1.1.1.1/",
		"netdoc://1.1.1.1/",
		"sftp://example.com/",
		"data:text/plain,hello",
		"javascript:alert(1)",
	} {
		err := validateOutboundURL(context.Background(), raw)
		if err == nil || !errors.Is(err, errSSRFBlocked) {
			t.Errorf("expected errSSRFBlocked for %q, got %v", raw, err)
		}
	}
}

// TestValidateOutboundURL_RejectsMalformedURL ensures that url.Parse errors
// are surfaced as SSRF blocks rather than swallowed.
func TestValidateOutboundURL_RejectsMalformedURL(t *testing.T) {
	for _, raw := range []string{
		"https://exa mple.com/", // space in host
		"https://[::1",          // unterminated bracket
		"https://%ZZ",           // invalid percent escape
	} {
		err := validateOutboundURL(context.Background(), raw)
		if err == nil || !errors.Is(err, errSSRFBlocked) {
			t.Errorf("expected errSSRFBlocked for malformed %q, got %v", raw, err)
		}
	}
}

// TestValidateOutboundURL_BlocksIPv4MappedIPv6Literal closes the bypass
// where an attacker submits ::ffff:127.0.0.1 to slip past v4-only filters.
func TestValidateOutboundURL_BlocksIPv4MappedIPv6Literal(t *testing.T) {
	for _, raw := range []string{
		"https://[::ffff:127.0.0.1]/",
		"https://[::ffff:10.0.0.1]/",
		"https://[::ffff:169.254.169.254]/",
	} {
		err := validateOutboundURL(context.Background(), raw)
		if err == nil || !errors.Is(err, errSSRFBlocked) {
			t.Errorf("expected errSSRFBlocked for %q, got %v", raw, err)
		}
	}
}

// TestValidateOutboundURL_AllowsPublicWithPortAndPath exercises a non-trivial
// URL shape (path, query, port) to make sure the parser branch only inspects
// the host.
func TestValidateOutboundURL_AllowsPublicWithPortAndPath(t *testing.T) {
	if err := validateOutboundURL(context.Background(), "https://1.1.1.1:8443/v1/models?x=1"); err != nil {
		t.Fatalf("expected public URL to be allowed, got %v", err)
	}
}

// TestValidateOutboundURL_LookupFailureIsTolerated ensures that DNS errors
// are NOT treated as SSRF blocks (the dial layer will surface a clearer
// network error). Using a TLD that cannot resolve.
func TestValidateOutboundURL_LookupFailureIsTolerated(t *testing.T) {
	// The .invalid TLD is reserved by RFC 2606 and must never resolve.
	err := validateOutboundURL(context.Background(), "https://nonexistent.invalid/")
	if err != nil {
		t.Fatalf("DNS failure should be tolerated by validateOutboundURL, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// ssrfSafeDialContext direct tests
// ---------------------------------------------------------------------------

// TestSSRFSafeDialContext_RejectsLiteralInternalIP verifies that the dial
// hook short-circuits before the OS-level connect when handed an internal
// literal IP, defeating DNS-rebinding even when validateOutboundURL has
// already passed.
func TestSSRFSafeDialContext_RejectsLiteralInternalIP(t *testing.T) {
	dialer := &net.Dialer{Timeout: time.Second}
	dial := ssrfSafeDialContext(dialer, true)

	for _, addr := range []string{
		"127.0.0.1:80",
		"10.1.2.3:443",
		"169.254.169.254:80",
		"[::1]:80",
		"[fc00::1]:443",
	} {
		conn, err := dial(context.Background(), "tcp", addr)
		if conn != nil {
			conn.Close()
			t.Errorf("expected dial to fail for %q, got a connection", addr)
		}
		if err == nil {
			t.Errorf("expected error for %q, got nil", addr)
			continue
		}
		if !errors.Is(err, errSSRFBlocked) {
			t.Errorf("dial(%q): err = %v, want wrapping errSSRFBlocked", addr, err)
		}
	}
}

// TestSSRFSafeDialContext_BadAddrFormat verifies that a malformed addr is
// reported through the underlying SplitHostPort error, not silently dropped.
func TestSSRFSafeDialContext_BadAddrFormat(t *testing.T) {
	dialer := &net.Dialer{Timeout: time.Second}
	dial := ssrfSafeDialContext(dialer, true)

	conn, err := dial(context.Background(), "tcp", "not-an-address")
	if conn != nil {
		conn.Close()
	}
	if err == nil {
		t.Fatalf("expected error for malformed addr, got nil")
	}
}

// TestSSRFSafeDialContext_DisabledGuardDelegates verifies that with the
// guard turned off the wrapper transparently delegates to the underlying
// dialer. We trigger a deterministic failure (port 1 + tiny timeout) so the
// test cannot be flaky on environments that disable raw-socket privileges.
func TestSSRFSafeDialContext_DisabledGuardDelegates(t *testing.T) {
	dialer := &net.Dialer{Timeout: 200 * time.Millisecond}
	dial := ssrfSafeDialContext(dialer, false)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	conn, err := dial(ctx, "tcp", "127.0.0.1:1")
	if conn != nil {
		conn.Close()
	}
	// We expect *some* error from the underlying dialer (connection refused
	// or context deadline) — and crucially NOT an SSRF block.
	if err == nil {
		t.Fatalf("expected dial error from underlying dialer, got nil")
	}
	if errors.Is(err, errSSRFBlocked) {
		t.Fatalf("guard disabled but got SSRF block: %v", err)
	}
}

// ---------------------------------------------------------------------------
// newSSRFSafeHTTPClient tests
// ---------------------------------------------------------------------------

// TestNewSSRFSafeHTTPClient_HasTimeout verifies that the constructor wires
// the timeout through (defence-in-depth: a long-running internal handler
// must not be able to keep the probe goroutine pinned indefinitely).
func TestNewSSRFSafeHTTPClient_HasTimeout(t *testing.T) {
	c := newSSRFSafeHTTPClient(7*time.Second, false)
	if c.Timeout != 7*time.Second {
		t.Fatalf("timeout = %v, want 7s", c.Timeout)
	}
	if c.Transport == nil {
		t.Fatalf("transport should not be nil")
	}
	if c.CheckRedirect == nil {
		t.Fatalf("CheckRedirect should not be nil")
	}
}

// TestNewSSRFSafeHTTPClient_RejectsInternalDestination verifies that an
// end-to-end Do() call against a literal internal IP is blocked at the dial
// layer, not just at validateOutboundURL.
func TestNewSSRFSafeHTTPClient_RejectsInternalDestination(t *testing.T) {
	c := newSSRFSafeHTTPClient(2*time.Second, true)
	req, err := http.NewRequest("GET", "https://10.0.0.1/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := c.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil {
		t.Fatalf("expected dial-layer SSRF block, got nil")
	}
	if !strings.Contains(err.Error(), errSSRFBlocked.Error()) &&
		!errors.Is(err, errSSRFBlocked) {
		t.Fatalf("expected SSRF error, got %v", err)
	}
}

// TestNewSSRFSafeHTTPClient_CheckRedirect_BlocksInternal directly drives
// the CheckRedirect closure with a synthetic redirect target, isolating
// the redirect-validation behaviour from network I/O.
func TestNewSSRFSafeHTTPClient_CheckRedirect_BlocksInternal(t *testing.T) {
	c := newSSRFSafeHTTPClient(time.Second, true)

	internal, _ := http.NewRequest("GET", "https://127.0.0.1/", nil)
	if err := c.CheckRedirect(internal, nil); err == nil || !errors.Is(err, errSSRFBlocked) {
		t.Fatalf("CheckRedirect to internal: err = %v, want errSSRFBlocked", err)
	}

	metadata, _ := http.NewRequest("GET", "https://169.254.169.254/", nil)
	if err := c.CheckRedirect(metadata, nil); err == nil || !errors.Is(err, errSSRFBlocked) {
		t.Fatalf("CheckRedirect to metadata: err = %v, want errSSRFBlocked", err)
	}
}

// TestNewSSRFSafeHTTPClient_CheckRedirect_AllowsPublic verifies that a
// public redirect target is permitted (otherwise legitimate providers that
// 30x to a regional endpoint would break).
func TestNewSSRFSafeHTTPClient_CheckRedirect_AllowsPublic(t *testing.T) {
	c := newSSRFSafeHTTPClient(time.Second, true)

	pub, _ := http.NewRequest("GET", "https://1.1.1.1/v1/models", nil)
	if err := c.CheckRedirect(pub, nil); err != nil {
		t.Fatalf("CheckRedirect to public IP: err = %v, want nil", err)
	}
}

// TestNewSSRFSafeHTTPClient_CheckRedirect_DisabledGuardPasses verifies the
// short-circuit branch when the guard has been turned off (the production
// default has it on; this branch only fires from inside the test binary).
func TestNewSSRFSafeHTTPClient_CheckRedirect_DisabledGuardPasses(t *testing.T) {
	c := newSSRFSafeHTTPClient(time.Second, false)
	r, _ := http.NewRequest("GET", "http://127.0.0.1/", nil)
	if err := c.CheckRedirect(r, nil); err != nil {
		t.Fatalf("guard disabled CheckRedirect should be a no-op, got %v", err)
	}
}
