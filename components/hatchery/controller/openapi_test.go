package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestWithOpenAPI_InjectsHeaderAndCallsHandler verifies WithOpenAPI injects a marker and calls the handler.
func TestWithOpenAPI_InjectsHeaderAndCallsHandler(t *testing.T) {
	called := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		called = true
		// Verify the header was injected
		if r.Header.Get(openAPIHeader) != "1" {
			t.Fatalf("expected openAPIHeader to be injected")
		}
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler := WithOpenAPI(handler)
	wrappedHandler(w, req)

	if !called {
		t.Fatalf("expected handler to be called")
	}
	// Verify the header was cleaned up after handler
	if req.Header.Get(openAPIHeader) != "" {
		t.Fatalf("expected openAPIHeader to be cleaned up after handler")
	}
}

// TestWithOpenAPI_SetAcceptJsonWhenBearerTokenPresent verifies Accept header is set to JSON when Bearer token exists.
func TestWithOpenAPI_SetAcceptJsonWhenBearerTokenPresent(t *testing.T) {
	handlerCalled := false
	var capturedRequest *http.Request
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		capturedRequest = r
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	wrappedHandler := WithOpenAPI(handler)
	wrappedHandler(w, req)

	if !handlerCalled {
		t.Fatalf("expected handler to be called")
	}
	if capturedRequest.Header.Get("Accept") != "application/json" {
		t.Fatalf("expected Accept header to be set to application/json, got %q", capturedRequest.Header.Get("Accept"))
	}
}

// TestWithOpenAPI_DoNotOverrideAcceptIfAlreadySet verifies Accept header is not overridden if already contains json.
func TestWithOpenAPI_DoNotOverrideAcceptIfAlreadySet(t *testing.T) {
	var capturedRequest *http.Request
	handler := func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Accept", "application/json;charset=utf-8")
	w := httptest.NewRecorder()

	wrappedHandler := WithOpenAPI(handler)
	wrappedHandler(w, req)

	if capturedRequest.Header.Get("Accept") != "application/json;charset=utf-8" {
		t.Fatalf("expected Accept header to remain unchanged")
	}
}

// TestWithOpenAPI_NoAcceptChangeWithoutBearerToken verifies Accept is not set without Bearer token.
func TestWithOpenAPI_NoAcceptChangeWithoutBearerToken(t *testing.T) {
	var capturedRequest *http.Request
	handler := func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler := WithOpenAPI(handler)
	wrappedHandler(w, req)

	if capturedRequest.Header.Get("Accept") != "" {
		t.Fatalf("expected Accept header not to be set without Bearer token")
	}
}

// TestIsOpenAPIRequest_True verifies isOpenAPIRequest returns true when header is set.
func TestIsOpenAPIRequest_True(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(openAPIHeader, "1")
	if !isOpenAPIRequest(req) {
		t.Fatalf("expected isOpenAPIRequest to return true")
	}
}

// TestIsOpenAPIRequest_False verifies isOpenAPIRequest returns false when header is not set.
func TestIsOpenAPIRequest_False(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	if isOpenAPIRequest(req) {
		t.Fatalf("expected isOpenAPIRequest to return false")
	}
}

// TestHasBearerToken_True verifies hasBearerToken returns true when Bearer token is present.
func TestHasBearerToken_True(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer my-token")
	if !hasBearerToken(req) {
		t.Fatalf("expected hasBearerToken to return true")
	}
}

// TestHasBearerToken_False verifies hasBearerToken returns false when no Bearer token.
func TestHasBearerToken_False(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	if hasBearerToken(req) {
		t.Fatalf("expected hasBearerToken to return false")
	}

	req.Header.Set("Authorization", "Basic xyz")
	if hasBearerToken(req) {
		t.Fatalf("expected hasBearerToken to return false for non-Bearer auth")
	}
}

// TestIsAdminTokenRequest_TrueWhenMatches verifies isAdminTokenRequest returns true when token matches.
func TestIsAdminTokenRequest_TrueWhenMatches(t *testing.T) {
	originalToken := AdminToken
	defer func() { AdminToken = originalToken }()

	AdminToken = "secret-admin-token"
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer secret-admin-token")
	if !isAdminTokenRequest(req) {
		t.Fatalf("expected isAdminTokenRequest to return true")
	}
}

// TestIsAdminTokenRequest_FalseWhenNotMatches verifies isAdminTokenRequest returns false when token doesn't match.
func TestIsAdminTokenRequest_FalseWhenNotMatches(t *testing.T) {
	originalToken := AdminToken
	defer func() { AdminToken = originalToken }()

	AdminToken = "secret-admin-token"
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	if isAdminTokenRequest(req) {
		t.Fatalf("expected isAdminTokenRequest to return false")
	}
}

// TestIsAdminTokenRequest_FalseWhenAdminTokenEmpty verifies isAdminTokenRequest returns false when AdminToken is empty.
func TestIsAdminTokenRequest_FalseWhenAdminTokenEmpty(t *testing.T) {
	originalToken := AdminToken
	defer func() { AdminToken = originalToken }()

	AdminToken = ""
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer any-token")
	if isAdminTokenRequest(req) {
		t.Fatalf("expected isAdminTokenRequest to return false when AdminToken is empty")
	}
}

// TestIsAdminTokenRequest_FalseWhenNoAuthHeader verifies isAdminTokenRequest returns false when no auth header.
func TestIsAdminTokenRequest_FalseWhenNoAuthHeader(t *testing.T) {
	originalToken := AdminToken
	defer func() { AdminToken = originalToken }()

	AdminToken = "secret-admin-token"
	req := httptest.NewRequest("GET", "/test", nil)
	if isAdminTokenRequest(req) {
		t.Fatalf("expected isAdminTokenRequest to return false when no auth header")
	}
}

// TestIsAdminTokenRequest_FalseWhenNonBearerAuth verifies isAdminTokenRequest returns false for non-Bearer auth.
func TestIsAdminTokenRequest_FalseWhenNonBearerAuth(t *testing.T) {
	originalToken := AdminToken
	defer func() { AdminToken = originalToken }()

	AdminToken = "secret-admin-token"
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic secret-admin-token")
	if isAdminTokenRequest(req) {
		t.Fatalf("expected isAdminTokenRequest to return false for non-Bearer auth")
	}
}
