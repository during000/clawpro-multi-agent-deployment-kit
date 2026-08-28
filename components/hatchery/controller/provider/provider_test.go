package provider

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestGetProvider_AnthropicMessages verifies that GetProvider returns AnthropicProvider for anthropic-messages.
func TestGetProvider_AnthropicMessages(t *testing.T) {
	provider := GetProvider("anthropic-messages")
	if provider == nil {
		t.Fatalf("GetProvider returned nil for anthropic-messages")
	}
	if _, ok := provider.(*AnthropicProvider); !ok {
		t.Fatalf("GetProvider returned %T, expected *AnthropicProvider", provider)
	}
}

// TestGetProvider_OpenAICompletions verifies that GetProvider returns OpenAIProvider for openai-completions.
func TestGetProvider_OpenAICompletions(t *testing.T) {
	provider := GetProvider("openai-completions")
	if provider == nil {
		t.Fatalf("GetProvider returned nil for openai-completions")
	}
	if _, ok := provider.(*OpenAIProvider); !ok {
		t.Fatalf("GetProvider returned %T, expected *OpenAIProvider", provider)
	}
}

// TestGetProvider_DefaultUnknown verifies that GetProvider returns OpenAIProvider for unknown types.
func TestGetProvider_DefaultUnknown(t *testing.T) {
	provider := GetProvider("unknown-provider")
	if provider == nil {
		t.Fatalf("GetProvider returned nil for unknown provider")
	}
	if _, ok := provider.(*OpenAIProvider); !ok {
		t.Fatalf("GetProvider returned %T for unknown type, expected default *OpenAIProvider", provider)
	}
}

// TestGetProvider_EmptyString verifies that GetProvider returns OpenAIProvider for empty string.
func TestGetProvider_EmptyString(t *testing.T) {
	provider := GetProvider("")
	if provider == nil {
		t.Fatalf("GetProvider returned nil for empty string")
	}
	if _, ok := provider.(*OpenAIProvider); !ok {
		t.Fatalf("GetProvider returned %T for empty string, expected default *OpenAIProvider", provider)
	}
}

func TestConnectivityError_Error(t *testing.T) {
	cause := errors.New("dial tcp timeout")
	tests := []struct {
		name string
		err  *ConnectivityError
		want string
	}{
		{
			name: "transport cause without status",
			err: &ConnectivityError{
				Kind:  ErrNetworkUnreachable,
				Cause: cause,
			},
			want: "network unreachable: dial tcp timeout",
		},
		{
			name: "status with snippet",
			err: &ConnectivityError{
				Kind:       ErrInvalidAPIKey,
				StatusCode: http.StatusUnauthorized,
				Snippet:    "bad key",
			},
			want: "invalid api key (status 401): bad key",
		},
		{
			name: "status without snippet",
			err: &ConnectivityError{
				Kind:       ErrForbidden,
				StatusCode: http.StatusForbidden,
			},
			want: "forbidden (status 403)",
		},
		{
			name: "kind only",
			err: &ConnectivityError{
				Kind: ErrRateLimited,
			},
			want: "rate limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConnectivityError_Unwrap(t *testing.T) {
	err := &ConnectivityError{Kind: ErrInvalidAPIKey}
	if !errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("errors.Is should match ErrInvalidAPIKey")
	}
	if errors.Is(err, ErrForbidden) {
		t.Fatalf("errors.Is should not match ErrForbidden")
	}
}

func TestClassifyHTTPStatus(t *testing.T) {
	tests := []struct {
		status int
		want   error
	}{
		{status: http.StatusOK, want: nil},
		{status: http.StatusNoContent, want: nil},
		{status: http.StatusFound, want: nil},
		{status: http.StatusUnauthorized, want: ErrInvalidAPIKey},
		{status: http.StatusForbidden, want: ErrForbidden},
		{status: http.StatusTooManyRequests, want: ErrRateLimited},
		{status: http.StatusBadRequest, want: ErrUpstreamClient},
		{status: http.StatusNotFound, want: ErrUpstreamClient},
		{status: http.StatusInternalServerError, want: ErrUpstreamServer},
		{status: http.StatusBadGateway, want: ErrUpstreamServer},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			got := classifyHTTPStatus(tt.status)
			if !errors.Is(got, tt.want) {
				t.Fatalf("classifyHTTPStatus(%d) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestReadBodySnippet_TrimsCollapsesAndDrains(t *testing.T) {
	r := strings.NewReader(" \nalpha\r\nbeta\n ")

	if got := readBodySnippet(r, 1024); got != "alpha  beta" {
		t.Fatalf("readBodySnippet() = %q, want %q", got, "alpha  beta")
	}

	remaining, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll remaining body: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("body should be drained, remaining %q", remaining)
	}
}

func TestReadBodySnippet_LimitDefaultAndDrains(t *testing.T) {
	r := strings.NewReader("abcdef")
	if got := readBodySnippet(r, 3); got != "abc" {
		t.Fatalf("readBodySnippet() = %q, want %q", got, "abc")
	}
	remaining, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll remaining body: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("body should be drained after limit, remaining %q", remaining)
	}

	r = strings.NewReader(strings.Repeat("a", 600))
	got := readBodySnippet(r, 0)
	if got != strings.Repeat("a", 512) {
		t.Fatalf("readBodySnippet() length = %d, want 512", len(got))
	}
	remaining, err = io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll remaining body: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("body should be drained after default limit, remaining length %d", len(remaining))
	}
}

// ─── applyCustomHeaders ─────────────────────────────────────────────────

func TestApplyCustomHeaders_SetsNewHeaders(t *testing.T) {
	req, _ := http.NewRequest("POST", "https://example.com", nil)
	headers := map[string]string{"X-Custom": "value1", "X-Request-Id": "req-123"}
	applyCustomHeaders(req, headers)

	if v := req.Header.Get("X-Custom"); v != "value1" {
		t.Errorf("X-Custom = %q, want %q", v, "value1")
	}
	if v := req.Header.Get("X-Request-Id"); v != "req-123" {
		t.Errorf("X-Request-Id = %q, want %q", v, "req-123")
	}
}

func TestApplyCustomHeaders_DoesNotOverwriteExisting(t *testing.T) {
	req, _ := http.NewRequest("POST", "https://example.com", nil)
	req.Header.Set("Content-Type", "application/json")
	// 试图覆盖 Content-Type，但已存在，不应覆盖
	headers := map[string]string{"Content-Type": "text/plain"}
	applyCustomHeaders(req, headers)

	if v := req.Header.Get("Content-Type"); v != "application/json" {
		t.Errorf("已有请求头不应被覆盖: Content-Type = %q, want %q", v, "application/json")
	}
}

func TestApplyCustomHeaders_NilMap(t *testing.T) {
	req, _ := http.NewRequest("POST", "https://example.com", nil)
	applyCustomHeaders(req, nil)
	// 不应 panic
	if len(req.Header) != 0 {
		t.Errorf("nil headers 不应设置任何请求头, 实际=%v", req.Header)
	}
}

func TestApplyCustomHeaders_EmptyMap(t *testing.T) {
	req, _ := http.NewRequest("POST", "https://example.com", nil)
	applyCustomHeaders(req, map[string]string{})
	if len(req.Header) != 0 {
		t.Errorf("空 map 不应设置任何请求头, 实际=%v", req.Header)
	}
}
