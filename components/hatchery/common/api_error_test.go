package common

import (
	"errors"
	"hatchery/i18n"
	"testing"
)

// TestNewRichError_ExistingI18nPrefix 覆盖 newRichError 中当嵌套 RichError
// 已有 i18nPrefix 时的分支（行 129-133）：创建新切片、prepend prefix、赋值。
func TestNewRichError_ExistingI18nPrefix(t *testing.T) {
	inner := I18nError(i18n.MsgABCredsNotConfigured)
	inner = inner.WithI18nPrefix(i18n.MsgOperationFailed)
	wrapped := inner.WithPrefix("layer2")

	// 验证 i18nPrefix 被 prepend：["layer2", "layer1"]
	if len(wrapped.i18nPrefix) != 2 {
		t.Fatalf("expected 2 i18nPrefix entries, got %d", len(wrapped.i18nPrefix))
	}
	if prefix, ok := wrapped.i18nPrefix[0].(string); !ok || prefix != "layer2" {
		t.Errorf("i18nPrefix[0] = %q, want %q", wrapped.i18nPrefix[0], "layer2")
	}
	if prefix, ok := wrapped.i18nPrefix[1].(i18n.KeyAndArgs); !ok || prefix.Key != i18n.MsgOperationFailed {
		t.Errorf("i18nPrefix[1] = %q, want %q", wrapped.i18nPrefix[1], "layer1")
	}
	if wrapped.i18nMessage.Key != i18n.MsgABCredsNotConfigured {
		t.Errorf("i18nKey = %q, want %q", wrapped.i18nMessage.Key, i18n.MsgABCredsNotConfigured)
	}
}

func TestNewRichError_NilI18nPrefix(t *testing.T) {
	inner := I18nError(i18n.MsgOperationFailed)
	wrapped := inner.WithI18nPrefix(i18n.MsgOperationFailed)

	if len(wrapped.i18nPrefix) != 1 {
		t.Fatalf("expected 1 i18nPrefix entry, got %d", len(wrapped.i18nPrefix))
	}
	if prefix, ok := wrapped.i18nPrefix[0].(i18n.KeyAndArgs); !ok || prefix.Key != i18n.MsgOperationFailed {
		t.Errorf("i18nPrefix[0] = %q, want %q", wrapped.i18nPrefix[0], i18n.MsgOperationFailed)
	}
}

// TestRichErrorWithInternalError_WithError 覆盖行 192-198：
// RichErrorWithInternalError.withError 返回 *RichErrorWithInternalErrors，
// 其 internalErrs 包含两个 error。
func TestRichErrorWithInternalError_WithError(t *testing.T) {
	err1 := errors.New("first error")
	err2 := errors.New("second error")

	rich := I18nError(i18n.MsgOperationFailed).WithCause(err1)
	result := rich.WithCause(err2)

	// 验证 internalErrs 包含两个 error
	errs := result.Unwrap()
	if len(errs) != 2 {
		t.Fatalf("expected 2 internal errors, got %d", len(errs))
	}
	if errs[0].Error() != "first error" {
		t.Errorf("errs[0] = %q, want %q", errs[0].Error(), "first error")
	}
	if errs[1].Error() != "second error" {
		t.Errorf("errs[1] = %q, want %q", errs[1].Error(), "second error")
	}
}

// TestRichErrorWithInternalErrors_Unwrap 覆盖行 206-208：
// RichErrorWithInternalErrors.Unwrap() 返回 internalErrs 切片。
func TestRichErrorWithInternalErrors_Unwrap(t *testing.T) {
	err1 := errors.New("e1")
	err2 := errors.New("e2")

	rich := I18nError(i18n.MsgOperationFailed).WithCause(err1).WithCause(err2)
	errs := rich.Unwrap()

	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(errs))
	}
	if !errors.Is(errs[0], err1) {
		t.Errorf("errs[0] mismatch: got %v, want %v", errs[0], err1)
	}
	if !errors.Is(errs[1], err2) {
		t.Errorf("errs[1] mismatch: got %v, want %v", errs[1], err2)
	}
}

// TestRichErrorWithInternalErrors_WithError 覆盖行 210-213：
// RichErrorWithInternalErrors.withError 追加 error 到 internalErrs 并返回自身。
func TestRichErrorWithInternalErrors_WithError(t *testing.T) {
	err1 := errors.New("first")
	err2 := errors.New("second")
	err3 := errors.New("third")

	// 构造一个含两个 error 的 RichErrorWithInternalErrors
	rich := I18nError(i18n.MsgOperationFailed).WithCause(err1).WithCause(err2)

	// 再次调用 withError（这次调用的是 RichErrorWithInternalErrors.withError）
	rich = rich.WithCause(err3)

	// 验证 internalErrs 现在有 3 个
	errs := rich.Unwrap()
	if len(errs) != 3 {
		t.Fatalf("expected 3 internal errors, got %d", len(errs))
	}
	if errs[2].Error() != "third" {
		t.Errorf("errs[2] = %q, want %q", errs[2].Error(), "third")
	}

	// 验证 errors.Is 可以找到追加的 error
	if !errors.Is(rich, err3) {
		t.Error("errors.Is should find err3 in the wrapped errors")
	}
}
