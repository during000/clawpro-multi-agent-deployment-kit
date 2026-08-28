package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
)

// TestI18nSentinel_BasicBehavior 验证 i18nSentinel 的基础 error 接口与 errors.Is 等价性。
func TestI18nSentinel_BasicBehavior(t *testing.T) {
	// errors.Is 通过指针相等性工作（单例 sentinel）。
	if !errors.Is(ErrMethodNotAllowed, ErrMethodNotAllowed) {
		t.Errorf("errors.Is(self, self) should be true")
	}

	// 不同 sentinel 不相等。
	if errors.Is(ErrMethodNotAllowed, ErrUnauthorized) {
		t.Errorf("errors.Is(MethodNotAllowed, Unauthorized) should be false")
	}
}

// TestErrorMessageCtx_SentinelTranslation 验证 errorMessageCtx 按请求 ctx 翻译 sentinel。
func TestErrorMessageCtx_SentinelTranslation(t *testing.T) {
	// 中文 ctx（无 Printer，fallback）
	cnCtx := context.Background()
	if got := hcommon.ErrorMessageWithCtx(cnCtx, ErrMethodNotAllowed); !strings.Contains(got, "请求方法不允许") {
		t.Errorf("CN errorMessageCtx = %q, want 请求方法不允许", got)
	}

	// 英文 ctx：通过 Middleware 包一层 handler 注入 Printer
	enCtx := buildEnglishCtx(t)
	if got := hcommon.ErrorMessageWithCtx(enCtx, ErrMethodNotAllowed); !strings.Contains(got, "Method not allowed") {
		t.Errorf("EN errorMessageCtx for ErrMethodNotAllowed = %q, want \"Method not allowed\"", got)
	}
	if got := hcommon.ErrorMessageWithCtx(enCtx, ErrUnauthorized); !strings.Contains(got, "Not logged in") {
		t.Errorf("EN errorMessageCtx for ErrUnauthorized = %q, want \"Not logged in\"", got)
	}
	if got := hcommon.ErrorMessageWithCtx(enCtx, ErrInstanceNotFound); !strings.Contains(got, "Instance not found") {
		t.Errorf("EN errorMessageCtx for ErrInstanceNotFound = %q, want \"Instance not found\"", got)
	}
}

// TestErrorMessageCtx_NonSentinel 验证非 sentinel 错误（普通 error / RichError）行为不变。
func TestErrorMessageCtx_NonSentinel(t *testing.T) {
	enCtx := buildEnglishCtx(t)

	plain := errors.New("plain error")
	if got := hcommon.ErrorMessageWithCtx(enCtx, plain); got != "plain error" {
		t.Errorf("plain error msg = %q, want \"plain error\"", got)
	}

	rich := hcommon.I18nError(i18n.MsgTATTimeout)
	if got := hcommon.ErrorMessageWithCtx(enCtx, rich); got != "TAT timeout" {
		t.Errorf("RichError msg = %q, want \"TAT timeout\"", got)
	}
}

// buildEnglishCtx 构造一个携带 English Printer 的 context（通过 Middleware 注入）。
func buildEnglishCtx(t *testing.T) context.Context {
	t.Helper()
	var captured context.Context
	handler := I18nMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Context()
	}))
	req := httptest.NewRequest(http.MethodGet, "/?lang=en", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if captured == nil {
		t.Fatal("middleware did not invoke handler")
	}
	return captured
}
