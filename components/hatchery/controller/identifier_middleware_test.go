package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/common"
)

// TestIdentifierMiddlewareWithTenantAgnosticPath 测试租户无关路径直接放行
func TestIdentifierMiddlewareWithTenantAgnosticPath(t *testing.T) {
	middleware := IdentifierMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 ctx 中没有 TenantSnapshot（租户无关路径不注入）
		snap, ok := common.GetTenantSnapshot(r.Context())
		if ok && snap.Identifier != "" {
			t.Fatalf("Tenant-agnostic path should not have tenant snapshot")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

// TestIdentifierMiddlewareWithFixedSnapshot 测试使用 FixedSnapshot 注入租户
func TestIdentifierMiddlewareWithFixedSnapshot(t *testing.T) {
	snap := &common.TenantSnapshot{
		Identifier: "test-tenant",
		Uin:        "12345",
		Domain:     "test.example.com",
	}

	// 保存原始 FixedSnapshot
	oldSnapshot := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnapshot }()

	// 设置 FixedSnapshot
	common.FixedSnapshot = snap

	middleware := IdentifierMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 ctx 中被注入了正确的 TenantSnapshot
		injectedSnap, ok := common.GetTenantSnapshot(r.Context())
		if !ok {
			t.Fatal("tenant snapshot should be injected")
		}
		if injectedSnap.Identifier != "test-tenant" {
			t.Fatalf("expected identifier 'test-tenant', got %q", injectedSnap.Identifier)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/admin/users", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

// TestIdentifierMiddlewareWithoutFixedSnapshot 测试未配置 FixedSnapshot 时的行为
// 注意：FixedSnapshot 为 nil 时调用非白名单路径会 panic（生产环境中 main.go 始终构造 FixedSnapshot）
// 这里只测试白名单路径（如 /health）在 nil 时能正常放行
func TestIdentifierMiddlewareWithoutFixedSnapshot(t *testing.T) {
	// 保存原始 FixedSnapshot
	oldSnapshot := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnapshot }()

	// 清空 FixedSnapshot
	common.FixedSnapshot = nil

	middleware := IdentifierMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// /health 是白名单路径，直接放行，不会走到 *FixedSnapshot
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

// TestIdentifierMiddlewareWithNilFixedSnapshot 测试 FixedSnapshot 为 nil 时的行为（universe 模式）。
// universe 模式下，未注册域名的请求应该返回 404。
func TestIdentifierMiddlewareWithNilFixedSnapshot(t *testing.T) {
	// 保存原始 FixedSnapshot
	oldSnapshot := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnapshot }()

	// 设置 FixedSnapshot 为 nil（模拟 universe 模式）
	common.FixedSnapshot = nil

	middleware := IdentifierMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 未注册域名 → 404
	req := httptest.NewRequest("GET", "/admin/users", nil)
	req.Host = "unknown.example.com"
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for unknown domain in universe mode, got %d", w.Code)
	}

	// 租户无关路径（/tenants/init）→ 直接放行
	req2 := httptest.NewRequest("POST", "/tenants/init", nil)
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200 for tenant-agnostic path, got %d", w2.Code)
	}
}

// TestIsTenantAgnosticPath 测试租户无关路径判断
func TestIsTenantAgnosticPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/health", true},
		{"/admin/users", false},
		{"/api/models", false},
		{"/login", false},
		{"/", false},
	}

	for _, tt := range tests {
		result := isTenantAgnosticPath(tt.path)
		if result != tt.expected {
			t.Fatalf("isTenantAgnosticPath(%q) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

// TestIdentifierMiddlewarePreservesOriginalRequest 测试中间件不修改原始请求
func TestIdentifierMiddlewarePreservesOriginalRequest(t *testing.T) {
	snap := &common.TenantSnapshot{
		Identifier: "test-tenant",
	}

	oldSnapshot := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnapshot }()
	common.FixedSnapshot = snap

	middleware := IdentifierMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证原始请求属性未被修改
		if r.Method != "POST" {
			t.Fatalf("expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/admin/test" {
			t.Fatalf("expected path /admin/test, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/admin/test", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

// TestIdentifierMiddlewareExtractsCorrectSnapshot 测试 middleware 正确提取 TenantSnapshot
func TestIdentifierMiddlewareExtractsCorrectSnapshot(t *testing.T) {
	snap := &common.TenantSnapshot{
		Identifier:     "extracted-tenant",
		Uin:            "555666",
		Domain:         "extracted.example.com",
		InternalSecret: "test-secret",
	}

	oldSnapshot := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnapshot }()
	common.FixedSnapshot = snap

	middleware := IdentifierMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		injectedSnap, ok := common.GetTenantSnapshot(r.Context())
		if !ok {
			t.Fatal("snapshot should be found in context")
		}
		if injectedSnap.Identifier != "extracted-tenant" {
			t.Fatalf("expected identifier 'extracted-tenant', got %q", injectedSnap.Identifier)
		}
		if injectedSnap.Uin != "555666" {
			t.Fatalf("expected UIN '555666', got %q", injectedSnap.Uin)
		}
		if injectedSnap.Domain != "extracted.example.com" {
			t.Fatalf("expected domain, got %q", injectedSnap.Domain)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/admin/config", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("handler should succeed")
	}
}
