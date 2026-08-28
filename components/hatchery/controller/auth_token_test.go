package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"
)

func TestGetUserFromToken_AdminToken(t *testing.T) {
	setupMemoryProDB(t)
	orig := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = orig }()

	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")

	user, err := getUserFromToken(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("expected admin user")
	}
	if user.Role != "admin" {
		t.Fatalf("role = %q, want admin", user.Role)
	}
}

func TestGetUserFromToken_UserTokenRequiresOpenAPI(t *testing.T) {
	setupMemoryProDB(t)
	token := "hk-user-token-001"
	user := model.User{Username: "alice", Password: "x", Role: "user", APIToken: &token}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	req := httptest.NewRequest("GET", "/not-openapi", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	got, err := getUserFromToken(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil user on non-openapi route, got %+v", got)
	}
}

func TestGetUserFromToken_UserTokenOpenAPI(t *testing.T) {
	setupMemoryProDB(t)
	token := "hk-user-token-002"
	user := model.User{Username: "bob", Password: "x", Role: "user", APIToken: &token}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	req := httptest.NewRequest("GET", "/openapi", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(openAPIHeader, "1")

	got, err := getUserFromToken(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.Username != "bob" {
		t.Fatalf("expected bob, got %+v", got)
	}
}

func TestGetUserFromToken_DisabledTokenOpenAPI(t *testing.T) {
	setupMemoryProDB(t)
	token := "hk-user-token-003"
	user := model.User{Username: "charlie", Password: "x", Role: "user", APIToken: &token, APITokenDisabled: true}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	req := httptest.NewRequest("GET", "/openapi", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(openAPIHeader, "1")

	got, err := getUserFromToken(req)
	if got != nil {
		t.Fatalf("expected nil user, got %+v", got)
	}
	if !errors.Is(err, model.TokenDisabledError{}) {
		t.Fatalf("expected TokenDisabledError, got %v", err)
	}
}

func TestGetUserFromToken_InvalidTokenOpenAPI(t *testing.T) {
	setupMemoryProDB(t)

	req := httptest.NewRequest("GET", "/openapi", nil)
	req.Header.Set("Authorization", "Bearer hk-invalid")
	req.Header.Set(openAPIHeader, "1")

	got, err := getUserFromToken(req)
	if got != nil {
		t.Fatalf("expected nil user, got %+v", got)
	}
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestWithOpenAPI_SetsInternalHeaderAndJSONAccept(t *testing.T) {
	setupMemoryProDB(t)

	var sawOpenAPI bool
	var sawJSONAccept bool
	h := WithOpenAPI(func(w http.ResponseWriter, r *http.Request) {
		sawOpenAPI = isOpenAPIRequest(r)
		sawJSONAccept = strings.Contains(r.Header.Get("Accept"), "application/json")
		jsonOK(w, map[string]any{"ok": true})
	})

	req := httptest.NewRequest("GET", "/api/demo", nil)
	req.Header.Set("Authorization", "Bearer hk-some-user-token")
	w := httptest.NewRecorder()
	h(w, req)

	if !sawOpenAPI {
		t.Fatal("handler should observe openapi marker")
	}
	if !sawJSONAccept {
		t.Fatal("handler should observe application/json accept")
	}
	if req.Header.Get(openAPIHeader) != "" {
		t.Fatalf("openapi marker should be cleaned after handler, got %q", req.Header.Get(openAPIHeader))
	}
}

func TestIsAdminTokenRequest(t *testing.T) {
	orig := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = orig }()

	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	if !isAdminTokenRequest(req) {
		t.Fatal("expected admin token request")
	}
}
