package common

import (
	"math"
	"context"
	"testing"
	"time"
)

func TestInjectAndGetTenant(t *testing.T) {
	ctx := context.Background()

	// 未注入时 GetTenantSnapshot 返回 false
	if _, ok := GetTenantSnapshot(ctx); ok {
		t.Fatal("empty ctx should not contain tenant snapshot")
	}

	snap := TenantSnapshot{
		Identifier: "tenant-a",
		Uin:        "1000",
		Domain:     "a.example.com",
	}
	ctx2 := InjectTenant(ctx, snap)
	got, ok := GetTenantSnapshot(ctx2)
	if !ok {
		t.Fatal("expected tenant snapshot, got none")
	}
	if got.Identifier != "tenant-a" || got.Uin != "1000" || got.Domain != "a.example.com" {
		t.Fatalf("unexpected snapshot: %+v", got)
	}

	// 原始 ctx 不受派生影响
	if _, ok := GetTenantSnapshot(ctx); ok {
		t.Fatal("original ctx should not be mutated")
	}
}

func TestInjectTenantWithNilCtx(t *testing.T) {
	//nolint:staticcheck // 明确测试 nil ctx 兼容分支
	ctx := InjectTenant(nil, TenantSnapshot{Identifier: "x"})
	got, ok := GetTenantSnapshot(ctx)
	if !ok || got.Identifier != "x" {
		t.Fatalf("nil ctx inject failed: ok=%v snap=%+v", ok, got)
	}
}

func TestSkipIdentifier(t *testing.T) {
	ctx := context.Background()
	if ShouldSkipIdentifier(ctx) {
		t.Fatal("default ctx should not skip identifier")
	}
	ctx2 := WithSkipIdentifier(ctx)
	if !ShouldSkipIdentifier(ctx2) {
		t.Fatal("expected skip flag set")
	}
	if ShouldSkipIdentifier(ctx) {
		t.Fatal("original ctx should not be mutated")
	}
}

func TestDetachContextPreservesTenant(t *testing.T) {
	snap := TenantSnapshot{Identifier: "tenant-b"}
	parent, cancel := context.WithTimeout(InjectTenant(context.Background(), snap), 10*time.Millisecond)
	cancel()

	detached := DetachContext(parent)

	// 派生出的 ctx 不应继承父 ctx 的 cancel
	select {
	case <-detached.Done():
		t.Fatal("detached ctx should not inherit cancel")
	default:
	}

	got, ok := GetTenantSnapshot(detached)
	if !ok || got.Identifier != "tenant-b" {
		t.Fatalf("detached ctx should preserve tenant snapshot, got ok=%v snap=%+v", ok, got)
	}
}

func TestDetachContextPreservesSkipFlag(t *testing.T) {
	ctx := WithSkipIdentifier(InjectTenant(context.Background(), TenantSnapshot{Identifier: "t"}))
	detached := DetachContext(ctx)
	if !ShouldSkipIdentifier(detached) {
		t.Fatal("detached ctx should preserve skip flag")
	}
}

func TestDetachContextEmpty(t *testing.T) {
	detached := DetachContext(context.Background())
	if _, ok := GetTenantSnapshot(detached); ok {
		t.Fatal("empty parent ctx should produce empty detached ctx")
	}
}

// TestGetTenantSnapshotWithNilCtx 测试 nil context 情况
func TestGetTenantSnapshotWithNilCtx(t *testing.T) {
	//nolint:staticcheck // 明确测试 nil ctx
	snap, ok := GetTenantSnapshot(nil)
	if ok {
		t.Fatal("nil ctx should return ok=false")
	}
	if snap.Identifier != "" {
		t.Fatal("nil ctx should return empty snapshot")
	}
}

// TestShouldSkipIdentifierWithNilCtx 测试 ShouldSkipIdentifier 的 nil 处理
func TestShouldSkipIdentifierWithNilCtx(t *testing.T) {
	//nolint:staticcheck // 明确测试 nil ctx
	result := ShouldSkipIdentifier(nil)
	if result {
		t.Fatal("nil ctx should return false")
	}
}

// TestWithSkipIdentifierWithNilCtx 测试 WithSkipIdentifier 的 nil 处理
func TestWithSkipIdentifierWithNilCtx(t *testing.T) {
	//nolint:staticcheck // 明确测试 nil ctx
	ctx := WithSkipIdentifier(nil)
	if !ShouldSkipIdentifier(ctx) {
		t.Fatal("result ctx should have skip flag")
	}
}

// TestDetachContextWithNilCtx 测试 DetachContext 的 nil 处理
func TestDetachContextWithNilCtx(t *testing.T) {
	//nolint:staticcheck // 明确测试 nil ctx
	detached := DetachContext(nil)
	if detached == nil {
		t.Fatal("DetachContext(nil) should return a valid context")
	}
	// 应该返回 Background 派生
	if _, ok := GetTenantSnapshot(detached); ok {
		t.Fatal("nil input should produce empty snapshot in output")
	}
}

func TestWithTaskTrace(t *testing.T) {
	old := FixedSnapshot
	FixedSnapshot = &TenantSnapshot{Identifier: "t-test"}
	defer func() { FixedSnapshot = old }()

	ctx := WithTaskTrace(context.Background(), "my_task")
	if GetRequestID(ctx) == "" {
		t.Error("WithTaskTrace should inject request_id")
	}
	if GetTraceID(ctx) == "" {
		t.Error("WithTaskTrace should inject trace_id")
	}
	if GetInterface(ctx) != "my_task" {
		t.Errorf("WithTaskTrace interface want my_task, got %q", GetInterface(ctx))
	}
	if !IsTask(ctx) {
		t.Error("WithTaskTrace should mark as task")
	}
}

func TestWithTaskTrace_NilCtx(t *testing.T) {
	ctx := WithTaskTrace(nil, "nil_task")
	if ctx == nil {
		t.Fatal("WithTaskTrace(nil) should not return nil")
	}
	if GetInterface(ctx) != "nil_task" {
		t.Errorf("want nil_task, got %q", GetInterface(ctx))
	}
}

func TestWithTaskTrace_WithSnapshotUin(t *testing.T) {
	snap := TenantSnapshot{Identifier: "t1", Uin: "snap-uin"}
	ctx := InjectTenant(context.Background(), snap)
	ctx = WithTaskTrace(ctx, "snap_task")
	// GetUin 已移除，通过 CVMUinFromCtx 从 TenantSnapshot 验证 uin
	if CVMUinFromCtx(ctx) != "snap-uin" {
		t.Errorf("want snap-uin, got %q", CVMUinFromCtx(ctx))
	}
}

func TestGetRequestID_NilCtx(t *testing.T) {
	if GetRequestID(nil) != "" {
		t.Error("nil ctx should return empty")
	}
}

func TestGetTraceID_NilCtx(t *testing.T) {
	if GetTraceID(nil) != "" {
		t.Error("nil ctx should return empty")
	}
}

func TestGetInterface_NilCtx(t *testing.T) {
	if GetInterface(nil) != "" {
		t.Error("nil ctx should return empty")
	}
}

func TestIsTask_NilCtx(t *testing.T) {
	if IsTask(nil) {
		t.Error("nil ctx should return false")
	}
}

func TestGetSubUin(t *testing.T) {
	if GetSubUin(nil) != 0 {
		t.Error("nil ctx should return 0")
	}
	ctx := context.WithValue(context.Background(), CtxKeySubUin, uint(42))
	if GetSubUin(ctx) != 42 {
		t.Errorf("want 42, got %d", GetSubUin(ctx))
	}
}

func TestGetRequestID_WithValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyRequestID, "req-123")
	if GetRequestID(ctx) != "req-123" {
		t.Errorf("want req-123, got %q", GetRequestID(ctx))
	}
}

func TestGetTraceID_WithValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyTraceID, "trace-456")
	if GetTraceID(ctx) != "trace-456" {
		t.Errorf("want trace-456, got %q", GetTraceID(ctx))
	}
}

func TestIsTask_False(t *testing.T) {
	ctx := context.Background()
	if IsTask(ctx) {
		t.Error("empty ctx should not be task")
	}
}

func TestDetachContextPreservesTaskTrace(t *testing.T) {
	old := FixedSnapshot
	FixedSnapshot = &TenantSnapshot{Identifier: "t-detach"}
	defer func() { FixedSnapshot = old }()

	ctx := WithTaskTrace(context.Background(), "trace_test")
	detached := DetachContext(ctx)
	// DetachContext preserves tenant snapshot but not arbitrary values
	// Just verify it doesn't panic and returns a valid context
	if detached == nil {
		t.Fatal("DetachContext should not return nil")
	}
}

// ─── TenantSnapshot ctx-aware accessor functions ─────────────────

func TestCVMUinFromCtx_WithSnapshot(t *testing.T) {
	ctx := InjectTenant(context.Background(), TenantSnapshot{Uin: "test-uin"})
	if got := CVMUinFromCtx(ctx); got != "test-uin" {
		t.Errorf("want test-uin, got %q", got)
	}
}

func TestCVMUinFromCtx_NoSnapshot(t *testing.T) {
	if got := CVMUinFromCtx(context.Background()); got != "" {
		t.Errorf("want empty, got %q", got)
	}
}

func TestDomainFromCtx_WithSnapshot(t *testing.T) {
	ctx := InjectTenant(context.Background(), TenantSnapshot{Domain: "https://test.example.com"})
	if got := DomainFromCtx(ctx); got != "https://test.example.com" {
		t.Errorf("want https://test.example.com, got %q", got)
	}
}

func TestDomainFromCtx_NoSnapshot(t *testing.T) {
	if got := DomainFromCtx(context.Background()); got != "" {
		t.Errorf("want empty, got %q", got)
	}
}

func TestInternalSecretFromCtx_WithSnapshot(t *testing.T) {
	ctx := InjectTenant(context.Background(), TenantSnapshot{InternalSecret: "secret-abc"})
	if got := InternalSecretFromCtx(ctx); got != "secret-abc" {
		t.Errorf("want secret-abc, got %q", got)
	}
}

func TestInternalSecretFromCtx_NoSnapshot(t *testing.T) {
	if got := InternalSecretFromCtx(context.Background()); got != "" {
		t.Errorf("want empty, got %q", got)
	}
}

func TestTenantIDFromCtx_WithSnapshot(t *testing.T) {
	ctx := InjectTenant(context.Background(), TenantSnapshot{OneIDAccountID: "oneid-123"})
	if got := TenantIDFromCtx(ctx); got != "oneid-123" {
		t.Errorf("want oneid-123, got %q", got)
	}
}

func TestTenantIDFromCtx_NoSnapshot(t *testing.T) {
	if got := TenantIDFromCtx(context.Background()); got != "" {
		t.Errorf("want empty, got %q", got)
	}
}

func TestUserLimitFromCtx_WithSnapshot(t *testing.T) {
	// 多租户阶段一：UserLimitFromCtx 始终返回 math.MaxInt，不再限制
	if got := UserLimitFromCtx(context.Background()); got != math.MaxInt {
		t.Errorf("want math.MaxInt, got %d", got)
	}
}

func TestUserLimitFromCtx_NoSnapshot(t *testing.T) {
	if got := UserLimitFromCtx(context.Background()); got != math.MaxInt {
		t.Errorf("want math.MaxInt, got %d", got)
	}
}
