package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
)

// Usage holds token counts returned by the upstream provider.
type Usage struct {
	PromptTokens           int
	CompletionTokens       int
	TotalTokens            int
	PromptCacheReadTokens  int
	PromptCacheWriteTokens int
}

// CompletionResult holds a non-streaming response body and parsed usage.
type CompletionResult struct {
	Body  []byte
	Usage *Usage
}

// StreamResult 封装流式响应的结果，包含 usage 统计和 tool_call 检测信息。
type StreamResult struct {
	Usage        *Usage
	HasToolCalls bool // LLM 响应中是否包含 tool_calls（用于 Agent Loop 宽限期判断）
}

var (
	// ErrNetworkUnreachable means the request never reached the upstream
	// (DNS failure, dial error, TLS handshake failure, timeout, ctx cancel…).
	ErrNetworkUnreachable = errors.New("network unreachable")

	// ErrInvalidAPIKey means the upstream rejected the credentials (HTTP 401).
	ErrInvalidAPIKey = errors.New("invalid api key")

	// ErrForbidden means the upstream returned HTTP 403 (key valid but the
	// account/region/model is not permitted).
	ErrForbidden = errors.New("forbidden")

	// ErrRateLimited means the upstream returned HTTP 429.
	ErrRateLimited = errors.New("rate limited")

	// ErrUpstreamServer means the upstream returned a 5xx status.
	ErrUpstreamServer = errors.New("upstream server error")

	// ErrUpstreamClient means the upstream returned a non-401/403/429 4xx
	// status (bad request, not found, etc).
	ErrUpstreamClient = errors.New("upstream client error")
)

type ConnectivityError struct {
	Kind       error  // one of the Err* sentinels above
	StatusCode int    // HTTP status code; 0 when the request never completed
	Snippet    string // truncated upstream response body, may be empty
	Cause      error  // the original transport error, may be nil
}

// Error renders a human readable message describing the failure.
func (e *ConnectivityError) Error() string {
	switch {
	case e.StatusCode == 0 && e.Cause != nil:
		return fmt.Sprintf("%s: %v", e.Kind.Error(), e.Cause)
	case e.Snippet != "":
		return fmt.Sprintf("%s (status %d): %s", e.Kind.Error(), e.StatusCode, e.Snippet)
	case e.StatusCode != 0:
		return fmt.Sprintf("%s (status %d)", e.Kind.Error(), e.StatusCode)
	default:
		return e.Kind.Error()
	}
}

// Unwrap returns the sentinel kind so errors.Is(err, ErrInvalidAPIKey) works.
func (e *ConnectivityError) Unwrap() error { return e.Kind }

// classifyHTTPStatus maps an HTTP status code (>=400) to the matching sentinel.
// Returns nil for 2xx/3xx, signalling success.
func classifyHTTPStatus(status int) error {
	switch {
	case status >= 200 && status < 400:
		return nil
	case status == http.StatusUnauthorized:
		return ErrInvalidAPIKey
	case status == http.StatusForbidden:
		return ErrForbidden
	case status == http.StatusTooManyRequests:
		return ErrRateLimited
	case status >= 500:
		return ErrUpstreamServer
	default:
		return ErrUpstreamClient
	}
}

// Provider abstracts an LLM provider's chat completion API.
type Provider interface {
	// ChatCompletion sends a non-streaming request and returns the raw
	// OpenAI-compatible JSON response body plus parsed provider metadata.
	// customHeaders are additional HTTP headers to include in the upstream request.
	ChatCompletion(ctx context.Context, apiKey, apiBase, model string, reqBody []byte, customHeaders map[string]string) (result *CompletionResult, statusCode int, err error)

	// ChatCompletionStream sends a streaming request and writes OpenAI-compatible
	// SSE chunks to w. It returns the StreamResult (usage + hasToolCalls) when available.
	// customHeaders are additional HTTP headers to include in the upstream request.
	ChatCompletionStream(ctx context.Context, apiKey, apiBase, model string, reqBody []byte, w http.ResponseWriter, flusher http.Flusher, customHeaders map[string]string) (result *StreamResult, statusCode int, err error)

	// CheckConnectivityWithChat probes the upstream provider by issuing a
	// minimal chat completion request (one user message, max_tokens=1).
	// This is more expensive than CheckConnectivity but works on providers
	// that don't expose a list-models endpoint, and additionally validates
	// that the specified model is actually invocable with the supplied key.
	//
	// On success it returns the round-trip latency and a nil error. On
	// failure it returns a *ConnectivityError wrapping one of the Err*
	// sentinels; callers can use errors.Is to classify the failure.
	CheckConnectivityWithChat(ctx context.Context, apiKey, apiBase, model string) (latency time.Duration, err error)
}

// applyCustomHeaders 将用户自定义的 HTTP 头设置到请求上。
// 不会覆盖已有的标准头（如 Content-Type、Authorization）。
func applyCustomHeaders(req *http.Request, headers map[string]string) {
	for k, v := range headers {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		} else {
			slog.WarnContext(req.Context(), "请求头已经存在，覆盖自定义请求头", slog.String("header", k))
		}
	}
}

func GetProvider(modelType string) Provider {
	switch modelType {
	case "anthropic-messages":
		return &AnthropicProvider{}
	default:
		return &OpenAIProvider{}
	}
}

// readBodySnippet reads up to maxBytes from r and returns it as a single-line
// trimmed string, suitable for embedding into a ConnectivityError.Snippet.
// The body is fully drained so the underlying connection can be reused.
func readBodySnippet(r io.Reader, maxBytes int64) string {
	if maxBytes <= 0 {
		maxBytes = 512
	}
	buf, _ := io.ReadAll(io.LimitReader(r, maxBytes))
	// Drain anything that exceeds maxBytes to allow connection reuse.
	_, _ = io.Copy(io.Discard, r)
	s := strings.TrimSpace(string(buf))
	// Collapse newlines so the message stays single-line in logs.
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// ---------------------------------------------------------------------------
// SSRF 漏洞防护
// ---------------------------------------------------------------------------

var errSSRFBlocked = errors.New("目标地址不允许访问")

// ipv4/ipv6 元数据 IP 地址
var (
	ipv4MetadataAddrs = map[string]struct{}{
		"169.254.169.254": {}, // AWS / Azure / GCP / Alibaba / Tencent Cloud
		"100.100.100.200": {}, // Alibaba Cloud (legacy)
	}
	ipv6MetadataAddrs = map[string]struct{}{
		"fd00:ec2::254": {}, // AWS IMDSv6
	}
)

// isDisallowedIP 如果返回 true 则代表该 IP 地址不能作为目的地址
// 不能作为目的地址的有:
// 1. 回环地址
// 2. 私有地址
// 3. IPv6 链路本地单播地址
// 4. IPv6 链路本地多播地址
// 5. 端口本地多播地址
// 6. 多播地址
// 7. 未指定地址如 0.0.0.0 等
// 8. 广播地址
// 9. 100.64.0.0/10
// 10. 0.0.0.0/8
// 11. 192.0.0.0/24
// 12. 192.0.2.0/24, 198.51.100.0/24, 203.0.113.0/24
// 13. 198.18.0.0/15
// 14. 240.0.0.0/4
// 15. IPv4/IPv6 元数据地址
// 16. fc00::/7
// 17. 2001:db8::/32
func isDisallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		if v4.Equal(net.IPv4bcast) {
			return true
		}
		// 100.64.0.0/10
		if v4[0] == 100 && (v4[1]&0xc0) == 64 {
			return true
		}
		// 0.0.0.0/8
		if v4[0] == 0 {
			return true
		}
		// 192.0.0.0/24
		if v4[0] == 192 && v4[1] == 0 && v4[2] == 0 {
			return true
		}
		// 192.0.2.0/24, 198.51.100.0/24, 203.0.113.0/24.
		if (v4[0] == 192 && v4[1] == 0 && v4[2] == 2) ||
			(v4[0] == 198 && v4[1] == 51 && v4[2] == 100) ||
			(v4[0] == 203 && v4[1] == 0 && v4[2] == 113) {
			return true
		}
		// 198.18.0.0/15
		if v4[0] == 198 && (v4[1] == 18 || v4[1] == 19) {
			return true
		}
		// 240.0.0.0/4
		if v4[0] >= 240 {
			return true
		}
		if _, ok := ipv4MetadataAddrs[v4.String()]; ok {
			return true
		}
	} else {
		if len(ip) == net.IPv6len {
			// fc00::/7
			if ip[0]&0xfe == 0xfc {
				return true
			}
			// 2001:db8::/32
			if ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0x0d && ip[3] == 0xb8 {
				return true
			}
		}
		if _, ok := ipv6MetadataAddrs[ip.String()]; ok {
			return true
		}
	}
	return false
}

// validateOutBoundURL 解析 URL 地址，保证其为合法目标地址
func validateOutboundURL(ctx context.Context, raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return hcommon.I18nRichError(errSSRFBlocked, i18n.MsgSSRFURLParseError, err)
	}

	// URL scheme 必须为 HTTP 或者 HTTPS
	scheme := strings.ToLower(u.Scheme)
	if scheme != "https" && scheme != "http" {
		return hcommon.I18nRichError(errSSRFBlocked, i18n.MsgSSRFSchemeNotAllowed, u.Scheme)
	}

	// Host 不能为空
	host := u.Hostname()
	if host == "" {
		return hcommon.I18nRichError(errSSRFBlocked, i18n.MsgSSRFHostEmpty)
	}

	// 如果 Host 是 IP 字面量则用 isDisallowedIP 检查该 IP 是否可以访问
	if ip := net.ParseIP(host); ip != nil {
		if isDisallowedIP(ip) {
			return hcommon.I18nRichError(errSSRFBlocked, i18n.MsgSSRFInternalAddress, ip.String())
		}
		return nil
	}

	// 如果不是 IP 字面量则解析出 IP 然后再用 isDisallowedIP 检查该 IP 是否可以访问
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		// 忽略解析错误 后续 Dial 可以给出更详细错误信息
		return nil
	}
	if len(addrs) == 0 {
		return hcommon.I18nRichError(errSSRFBlocked, i18n.MsgSSRFNoMappedIP, host)
	}
	for _, a := range addrs {
		if isDisallowedIP(a.IP) {
			return hcommon.I18nRichError(errSSRFBlocked, i18n.MsgSSRFResolvedToInternal, host, a.IP.String())
		}
	}
	return nil
}

func ssrfSafeDialContext(base *net.Dialer, enableSSRF bool) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !enableSSRF {
			return base.DialContext(ctx, network, addr)
		}
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		if ip := net.ParseIP(host); ip != nil {
			if isDisallowedIP(ip) {
				return nil, hcommon.I18nRichError(errSSRFBlocked, i18n.MsgSSRFInternalAddress, ip.String())
			}
			return base.DialContext(ctx, network, addr)
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, a := range ips {
			if isDisallowedIP(a.IP) {
				return nil, hcommon.I18nRichError(errSSRFBlocked, i18n.MsgSSRFResolvedToInternal, host, a.IP.String())
			}
			conn, err := base.DialContext(ctx, network, net.JoinHostPort(a.IP.String(), port))
			if err == nil {
				return conn, nil
			}
		}
		return nil, hcommon.I18nRichError(errSSRFBlocked, i18n.MsgSSRFCannotResolve, host)
	}
}

func newSSRFSafeHTTPClient(timeout time.Duration, enableSSRF bool) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:       http.ProxyFromEnvironment,
		DialContext: ssrfSafeDialContext(dialer, enableSSRF),
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		// 检查重定向后的目标 URL 是否合法
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !enableSSRF {
				return nil
			}
			return validateOutboundURL(req.Context(), req.URL.String())
		},
	}
}
