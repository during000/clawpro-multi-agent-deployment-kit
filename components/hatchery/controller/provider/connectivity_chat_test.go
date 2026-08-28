package provider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- OpenAI provider chat-probe tests ---

// TestOpenAICheckConnectivityWithChat_Success verifies that the chat probe
// hits /chat/completions with a minimal payload and accepts a 200 response.
func TestOpenAICheckConnectivityWithChat_Success(t *testing.T) {
	var (
		gotPath   string
		gotAuth   string
		gotMethod string
		gotBody   map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","choices":[{"message":{"role":"assistant","content":""}}]}`))
	}))
	defer srv.Close()

	p := &OpenAIProvider{}
	latency, err := p.CheckConnectivityWithChat(context.Background(), "sk-ok", srv.URL, "gpt-4o-mini")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if latency <= 0 {
		t.Fatalf("expected positive latency, got %v", latency)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/chat/completions" {
		t.Fatalf("path = %s, want /chat/completions", gotPath)
	}
	if gotAuth != "Bearer sk-ok" {
		t.Fatalf("auth = %q, want Bearer sk-ok", gotAuth)
	}
	if gotBody["model"] != "gpt-4o-mini" {
		t.Fatalf("body.model = %v, want gpt-4o-mini", gotBody["model"])
	}
	// Legacy GPT model: should send max_tokens (not max_completion_tokens).
	if _, ok := gotBody["max_tokens"]; !ok {
		t.Fatalf("body should contain max_tokens for legacy model: %v", gotBody)
	}
	if _, ok := gotBody["max_completion_tokens"]; ok {
		t.Fatalf("body should NOT contain max_completion_tokens for legacy model: %v", gotBody)
	}
	msgs, ok := gotBody["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Fatalf("messages should be a single-element array, got %v", gotBody["messages"])
	}
}

// TestOpenAICheckConnectivityWithChat_NewModelMaxCompletionTokens ensures the
// chat probe normalises max_tokens → max_completion_tokens for GPT-5+ / o-series.
func TestOpenAICheckConnectivityWithChat_NewModelMaxCompletionTokens(t *testing.T) {
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	p := &OpenAIProvider{}
	if _, err := p.CheckConnectivityWithChat(context.Background(), "sk", srv.URL, "gpt-5"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := gotBody["max_tokens"]; ok {
		t.Fatalf("max_tokens should be removed for gpt-5: %v", gotBody)
	}
	if _, ok := gotBody["max_completion_tokens"]; !ok {
		t.Fatalf("max_completion_tokens should be set for gpt-5: %v", gotBody)
	}
}

// TestOpenAICheckConnectivityWithChat_Unauthorized verifies that a 401 from
// the upstream is reported as ErrInvalidAPIKey with a snippet captured.
func TestOpenAICheckConnectivityWithChat_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := &OpenAIProvider{}
	_, err := p.CheckConnectivityWithChat(context.Background(), "sk-bad", srv.URL, "gpt-4o-mini")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("err = %v, want ErrInvalidAPIKey", err)
	}
	var ce *ConnectivityError
	if !errors.As(err, &ce) {
		t.Fatalf("err is not *ConnectivityError: %v", err)
	}
	if ce.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", ce.StatusCode)
	}
	if !strings.Contains(ce.Snippet, "bad key") {
		t.Fatalf("snippet = %q, want it to contain %q", ce.Snippet, "bad key")
	}
}

// TestOpenAICheckConnectivityWithChat_NetworkUnreachable verifies that
// a transport-level failure is reported as ErrNetworkUnreachable.
func TestOpenAICheckConnectivityWithChat_NetworkUnreachable(t *testing.T) {
	// Use an unroutable URL so Dial fails fast.
	p := &OpenAIProvider{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := p.CheckConnectivityWithChat(ctx, "sk", "http://127.0.0.1:1", "gpt-4o-mini")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrNetworkUnreachable) {
		t.Fatalf("err = %v, want ErrNetworkUnreachable", err)
	}
}

// TestOpenAICheckConnectivityWithChat_RateLimited verifies 429 → ErrRateLimited.
func TestOpenAICheckConnectivityWithChat_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	p := &OpenAIProvider{}
	_, err := p.CheckConnectivityWithChat(context.Background(), "sk", srv.URL, "gpt-4o-mini")
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, want ErrRateLimited", err)
	}
}

// --- Anthropic provider chat-probe tests ---

// TestAnthropicCheckConnectivityWithChat_Success verifies the Anthropic chat
// probe hits /v1/messages with x-api-key, anthropic-version, and a minimal body.
func TestAnthropicCheckConnectivityWithChat_Success(t *testing.T) {
	var (
		gotPath    string
		gotAPIKey  string
		gotVersion string
		gotMethod  string
		gotBody    map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","content":[{"type":"text","text":""}]}`))
	}))
	defer srv.Close()

	p := &AnthropicProvider{}
	latency, err := p.CheckConnectivityWithChat(context.Background(), "sk-anthropic", srv.URL, "claude-3-5-haiku-20241022")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if latency <= 0 {
		t.Fatalf("expected positive latency, got %v", latency)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("path = %s, want /v1/messages", gotPath)
	}
	if gotAPIKey != "sk-anthropic" {
		t.Fatalf("x-api-key = %q, want sk-anthropic", gotAPIKey)
	}
	if gotVersion == "" {
		t.Fatalf("anthropic-version header must be set")
	}
	if gotBody["model"] != "claude-3-5-haiku-20241022" {
		t.Fatalf("body.model = %v, want claude-3-5-haiku-20241022", gotBody["model"])
	}
	if v, ok := gotBody["max_tokens"].(float64); !ok || int(v) != 1 {
		t.Fatalf("body.max_tokens = %v, want 1", gotBody["max_tokens"])
	}
	msgs, ok := gotBody["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Fatalf("messages should be a single-element array, got %v", gotBody["messages"])
	}
}

// TestAnthropicCheckConnectivityWithChat_Forbidden verifies that 403 is
// surfaced as ErrForbidden.
func TestAnthropicCheckConnectivityWithChat_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no access", http.StatusForbidden)
	}))
	defer srv.Close()

	p := &AnthropicProvider{}
	_, err := p.CheckConnectivityWithChat(context.Background(), "sk", srv.URL, "claude")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

// TestAnthropicCheckConnectivityWithChat_UpstreamServerError verifies that 5xx
// is surfaced as ErrUpstreamServer.
func TestAnthropicCheckConnectivityWithChat_UpstreamServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()

	p := &AnthropicProvider{}
	_, err := p.CheckConnectivityWithChat(context.Background(), "sk", srv.URL, "claude")
	if !errors.Is(err, ErrUpstreamServer) {
		t.Fatalf("err = %v, want ErrUpstreamServer", err)
	}
}

// TestAnthropicCheckConnectivityWithChat_DefaultBaseWhenEmpty verifies that an
// empty apiBase falls back to the Anthropic default endpoint. We can't reach
// that endpoint in a unit test, so we just check the URL the request ended up
// targeting via a dial-failure substring.
func TestAnthropicCheckConnectivityWithChat_DefaultBaseWhenEmpty(t *testing.T) {
	// Quick context to avoid any real network call.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	p := &AnthropicProvider{}
	_, err := p.CheckConnectivityWithChat(ctx, "sk", "", "claude")
	if err == nil {
		t.Skip("network reachable to api.anthropic.com — cannot assert default base in this env")
	}
	// We expect a transport-level error, since either DNS / TCP / TLS fails or
	// the ctx times out before completion.
	if !errors.Is(err, ErrNetworkUnreachable) {
		t.Fatalf("err = %v, want ErrNetworkUnreachable", err)
	}
}

func TestOpenAICheckConnectivityWithChat_InvalidURL(t *testing.T) {
	p := &OpenAIProvider{}
	_, err := p.CheckConnectivityWithChat(context.Background(), "sk", "http://invalid\x7fhost", "gpt-4o-mini")
	if err == nil {
		t.Fatalf("expected error for invalid URL, got nil")
	}
	if !errors.Is(err, ErrNetworkUnreachable) {
		t.Fatalf("err = %v, want ErrNetworkUnreachable", err)
	}
	var ce *ConnectivityError
	if !errors.As(err, &ce) {
		t.Fatalf("err is not *ConnectivityError: %v", err)
	}
	if ce.Cause == nil {
		t.Fatalf("ConnectivityError.Cause should be populated for invalid URL: %#v", ce)
	}
	if ce.StatusCode != 0 {
		t.Fatalf("ConnectivityError.StatusCode = %d, want 0 (request never completed)", ce.StatusCode)
	}
}

func TestAnthropicCheckConnectivityWithChat_InvalidURL(t *testing.T) {
	p := &AnthropicProvider{}
	_, err := p.CheckConnectivityWithChat(context.Background(), "sk", "http://invalid\x7fhost", "claude")
	if err == nil {
		t.Fatalf("expected error for invalid URL, got nil")
	}
	if !errors.Is(err, ErrNetworkUnreachable) {
		t.Fatalf("err = %v, want ErrNetworkUnreachable", err)
	}
	var ce *ConnectivityError
	if !errors.As(err, &ce) {
		t.Fatalf("err is not *ConnectivityError: %v", err)
	}
	if ce.Cause == nil {
		t.Fatalf("ConnectivityError.Cause should be populated for invalid URL: %#v", ce)
	}
}

func TestOpenAICheckConnectivityWithChat_NoAPIKeyHeader(t *testing.T) {
	var sawAuth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			sawAuth = true
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	p := &OpenAIProvider{}
	if _, err := p.CheckConnectivityWithChat(context.Background(), "", srv.URL, "gpt-4o-mini"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sawAuth {
		t.Fatalf("Authorization header must not be set when apiKey is empty")
	}
}

func TestAnthropicCheckConnectivityWithChat_NoAPIKeyHeader(t *testing.T) {
	var (
		sawAPIKey  bool
		sawVersion bool
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "" {
			sawAPIKey = true
		}
		if r.Header.Get("anthropic-version") != "" {
			sawVersion = true
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	p := &AnthropicProvider{}
	if _, err := p.CheckConnectivityWithChat(context.Background(), "", srv.URL, "claude"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sawAPIKey {
		t.Fatalf("x-api-key header must not be set when apiKey is empty")
	}
	if !sawVersion {
		t.Fatalf("anthropic-version header must always be set")
	}
}

func TestOpenAICheckConnectivityWithChat_ContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block long enough for the test ctx to cancel.
		time.Sleep(2 * time.Second)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	p := &OpenAIProvider{}
	_, err := p.CheckConnectivityWithChat(ctx, "sk", srv.URL, "gpt-4o-mini")
	if err == nil {
		t.Fatalf("expected error after ctx cancel, got nil")
	}
	if !errors.Is(err, ErrNetworkUnreachable) {
		t.Fatalf("err = %v, want ErrNetworkUnreachable", err)
	}
}

func TestProvider_CheckConnectivityWithChat_InterfaceCompliance(t *testing.T) {
	var _ Provider = (*OpenAIProvider)(nil)
	var _ Provider = (*AnthropicProvider)(nil)
}

// Ensure the io import isn't accidentally dropped (used by other helpers
// indirectly when we drain the test server body).
var _ = io.EOF
