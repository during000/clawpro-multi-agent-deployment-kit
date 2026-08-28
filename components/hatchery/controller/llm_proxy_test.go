package controller

import (
	"net/http"
	"testing"
)

func TestEnforceMaxTokens(t *testing.T) {
	t.Run("zero_limit_is_noop", func(t *testing.T) {
		req := map[string]interface{}{
			"max_tokens": float64(5000),
		}
		enforceMaxTokens(req, 0)
		if req["max_tokens"].(float64) != 5000 {
			t.Error("zero limit should not cap")
		}
	})

	t.Run("caps_max_tokens_when_exceeds", func(t *testing.T) {
		req := map[string]interface{}{
			"max_tokens": float64(5000),
		}
		enforceMaxTokens(req, 1000)
		if req["max_tokens"].(float64) != 1000 {
			t.Errorf("max_tokens = %v, want 1000", req["max_tokens"])
		}
	})

	t.Run("caps_max_completion_tokens_when_exceeds", func(t *testing.T) {
		req := map[string]interface{}{
			"max_completion_tokens": float64(5000),
		}
		enforceMaxTokens(req, 1000)
		if req["max_completion_tokens"].(float64) != 1000 {
			t.Errorf("max_completion_tokens = %v, want 1000", req["max_completion_tokens"])
		}
	})

	t.Run("caps_both_when_both_exceed", func(t *testing.T) {
		req := map[string]interface{}{
			"max_tokens":            float64(5000),
			"max_completion_tokens": float64(4000),
		}
		enforceMaxTokens(req, 1000)
		if req["max_tokens"].(float64) != 1000 {
			t.Errorf("max_tokens = %v, want 1000", req["max_tokens"])
		}
		if req["max_completion_tokens"].(float64) != 1000 {
			t.Errorf("max_completion_tokens = %v, want 1000", req["max_completion_tokens"])
		}
	})

	t.Run("injects_max_tokens_when_neither_present", func(t *testing.T) {
		req := map[string]interface{}{
			"temperature": float64(0.7),
		}
		enforceMaxTokens(req, 1000)
		if req["max_tokens"].(float64) != 1000 {
			t.Errorf("max_tokens = %v, want 1000", req["max_tokens"])
		}
		if _, ok := req["max_completion_tokens"]; ok {
			t.Error("max_completion_tokens should not be injected")
		}
	})

	t.Run("no_inject_when_max_completion_tokens_present", func(t *testing.T) {
		req := map[string]interface{}{
			"max_completion_tokens": float64(500),
		}
		enforceMaxTokens(req, 1000)
		// under limit, should not change
		if req["max_completion_tokens"].(float64) != 500 {
			t.Errorf("max_completion_tokens = %v, want 500", req["max_completion_tokens"])
		}
		// should not inject max_tokens
		if _, ok := req["max_tokens"]; ok {
			t.Error("max_tokens should not be injected when max_completion_tokens present")
		}
	})

	t.Run("no_inject_when_max_tokens_present", func(t *testing.T) {
		req := map[string]interface{}{
			"max_tokens": float64(500),
		}
		enforceMaxTokens(req, 1000)
		if req["max_tokens"].(float64) != 500 {
			t.Errorf("max_tokens = %v, want 500", req["max_tokens"])
		}
	})

	t.Run("does_not_cap_when_under_limit", func(t *testing.T) {
		req := map[string]interface{}{
			"max_tokens":            float64(500),
			"max_completion_tokens": float64(300),
		}
		enforceMaxTokens(req, 1000)
		if req["max_tokens"].(float64) != 500 {
			t.Error("should not cap under limit")
		}
		if req["max_completion_tokens"].(float64) != 300 {
			t.Error("should not cap under limit")
		}
	})

	t.Run("negative_limit_is_noop", func(t *testing.T) {
		req := map[string]interface{}{
			"max_tokens": float64(5000),
		}
		enforceMaxTokens(req, -1)
		if req["max_tokens"].(float64) != 5000 {
			t.Error("negative limit should not cap")
		}
	})
}

// ─── extractCustomHeaders ────────────────────────────────────────────────

func TestExtractCustomHeaders_NilMap(t *testing.T) {
	r, _ := http.NewRequest("POST", "/v1/chat/completions", nil)
	result := extractCustomHeaders(r, nil)
	if result != nil {
		t.Errorf("nil customHeaders 应返回 nil, 实际=%v", result)
	}
}

func TestExtractCustomHeaders_EmptyMap(t *testing.T) {
	r, _ := http.NewRequest("POST", "/v1/chat/completions", nil)
	result := extractCustomHeaders(r, map[string]string{})
	if result != nil {
		t.Errorf("空 customHeaders 应返回 nil, 实际=%v", result)
	}
}

func TestExtractCustomHeaders_AllMatch(t *testing.T) {
	r, _ := http.NewRequest("POST", "/v1/chat/completions", nil)
	r.Header.Set("X-Api-Key", "sk-123")
	r.Header.Set("X-Request-Id", "req-1")

	customHeaders := map[string]string{"X-Api-Key": "sk-123", "X-Request-Id": "req-1"}
	result := extractCustomHeaders(r, customHeaders)
	if result == nil {
		t.Fatal("期望返回非 nil, 实际 nil")
	}
	// 返回的 key 是 CanonicalHeaderKey 形式
	if result["X-Api-Key"] != "sk-123" {
		t.Errorf("X-Api-Key = %q, want %q", result["X-Api-Key"], "sk-123")
	}
	if result["X-Request-Id"] != "req-1" {
		t.Errorf("X-Request-Id = %q, want %q", result["X-Request-Id"], "req-1")
	}
}

func TestExtractCustomHeaders_ValueMismatch(t *testing.T) {
	r, _ := http.NewRequest("POST", "/v1/chat/completions", nil)
	r.Header.Set("X-Api-Key", "wrong-key")

	customHeaders := map[string]string{"X-Api-Key": "sk-123"}
	result := extractCustomHeaders(r, customHeaders)
	if result != nil {
		t.Errorf("值不匹配时应返回 nil, 实际=%v", result)
	}
}

func TestExtractCustomHeaders_HeaderMissing(t *testing.T) {
	r, _ := http.NewRequest("POST", "/v1/chat/completions", nil)
	// 请求中未设置 X-Api-Key

	customHeaders := map[string]string{"X-Api-Key": "sk-123"}
	result := extractCustomHeaders(r, customHeaders)
	if result != nil {
		t.Errorf("请求头缺失时应返回 nil, 实际=%v", result)
	}
}

func TestExtractCustomHeaders_PartialMatch(t *testing.T) {
	r, _ := http.NewRequest("POST", "/v1/chat/completions", nil)
	r.Header.Set("X-Api-Key", "sk-123")
	// X-Request-Id 缺失

	customHeaders := map[string]string{"X-Api-Key": "sk-123", "X-Request-Id": "req-1"}
	result := extractCustomHeaders(r, customHeaders)
	// 任一不匹配则整个返回 nil
	if result != nil {
		t.Errorf("部分匹配时应返回 nil, 实际=%v", result)
	}
}

func TestExtractCustomHeaders_CanonicalHeaderKey(t *testing.T) {
	r, _ := http.NewRequest("POST", "/v1/chat/completions", nil)
	r.Header.Set("x-api-key", "sk-123") // 小写设置

	customHeaders := map[string]string{"x-api-key": "sk-123"}
	result := extractCustomHeaders(r, customHeaders)
	if result == nil {
		t.Fatal("期望返回非 nil, 实际 nil")
	}
	// http.Header.Get 会自动做 CanonicalHeaderKey 查找
	// extractCustomHeaders 返回时也做了 CanonicalHeaderKey
	_, ok := result[http.CanonicalHeaderKey("x-api-key")]
	if !ok {
		t.Errorf("应包含规范化的 key %q, 实际=%v", http.CanonicalHeaderKey("x-api-key"), result)
	}
}
