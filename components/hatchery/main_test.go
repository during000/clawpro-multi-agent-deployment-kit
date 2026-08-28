package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/controller"
	"hatchery/i18n"
)

func TestDeprecatedFlags_TestFixedFlagValues(t *testing.T) {
	deprecatedFlagsMap := deprecatedFlags()
	if deprecatedFlagsMap["disable-ui"] != true {
		t.Fatal("--disable-ui should be true")
	}
	if deprecatedFlagsMap["user-limit"] != 1000 {
		t.Fatal("--user-limit should be 1000")
	}
}

// ─── applyFlags 测试 ─────────────────────────────────────────────────────────

func TestApplyFlags_LangEn(t *testing.T) {
	applyFlags("en")
	if !i18n.IsOverseas() {
		t.Fatalf("IsOverseas() = false, want true when lang=en")
	}
	// Restore
	i18n.SetDefaultLang("zh")
}

// ─── pprofAuthMiddleware 测试 ─────────────────────────────────────────────────

func TestPprofAuthMiddleware_NoToken_Forbidden(t *testing.T) {
	origToken := controller.AdminToken
	controller.AdminToken = ""
	defer func() { controller.AdminToken = origToken }()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("inner handler should not be called")
	})

	handler := pprofAuthMiddleware(inner)
	req := httptest.NewRequest("GET", "/debug/pprof/goroutine", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestPprofAuthMiddleware_BadToken_Unauthorized(t *testing.T) {
	origToken := controller.AdminToken
	controller.AdminToken = "valid-token"
	defer func() { controller.AdminToken = origToken }()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("inner handler should not be called")
	})

	handler := pprofAuthMiddleware(inner)
	req := httptest.NewRequest("GET", "/debug/pprof/heap", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestPprofAuthMiddleware_ValidToken_PassThrough(t *testing.T) {
	origToken := controller.AdminToken
	controller.AdminToken = "my-secret"
	defer func() { controller.AdminToken = origToken }()

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := pprofAuthMiddleware(inner)
	req := httptest.NewRequest("GET", "/debug/pprof/goroutine", nil)
	req.Header.Set("Authorization", "Bearer my-secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("inner handler should be called with valid token")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestPprofAuthMiddleware_NonPprofPath_PassThrough(t *testing.T) {
	origToken := controller.AdminToken
	controller.AdminToken = ""
	defer func() { controller.AdminToken = origToken }()

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := pprofAuthMiddleware(inner)
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("non-pprof paths should pass through regardless of token")
	}
}
