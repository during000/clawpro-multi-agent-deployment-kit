package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ── rewriteCookieDomain 单元测试（纯函数，table-driven） ──────────────────────

func TestOneidUnified_RewriteCookieDomain(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		newDomain string
		want      string
	}{
		{
			name:      "replaces existing Domain attribute",
			raw:       "state_token=abc; Domain=account.tencent.com; Path=/; Secure",
			newDomain: "my.example.com",
			want:      "state_token=abc; Domain=my.example.com; Path=/; Secure",
		},
		{
			name:      "no Domain attribute leaves cookie unchanged",
			raw:       "state_token=abc; Path=/; Secure",
			newDomain: "my.example.com",
			want:      "state_token=abc; Path=/; Secure",
		},
		{
			name:      "strips https:// prefix from newDomain",
			raw:       "session_token=xyz; Domain=old.com; Path=/",
			newDomain: "https://my.example.com",
			want:      "session_token=xyz; Domain=my.example.com; Path=/",
		},
		{
			name:      "strips http:// prefix from newDomain",
			raw:       "session_token=xyz; Domain=old.com; Path=/",
			newDomain: "http://my.example.com",
			want:      "session_token=xyz; Domain=my.example.com; Path=/",
		},
		{
			name:      "case-insensitive Domain matching",
			raw:       "tok=val; DOMAIN=old.com; Path=/",
			newDomain: "new.com",
			want:      "tok=val; Domain=new.com; Path=/",
		},
		{
			name:      "empty raw string",
			raw:       "",
			newDomain: "my.example.com",
			want:      "",
		},
		{
			name:      "multiple attributes no domain",
			raw:       "tok=val; Path=/; HttpOnly; Secure; SameSite=Lax",
			newDomain: "my.example.com",
			want:      "tok=val; Path=/; HttpOnly; Secure; SameSite=Lax",
		},
		{
			name:      "domain at the end",
			raw:       "tok=val; Path=/; Domain=old.tencent.com",
			newDomain: "new.example.com",
			want:      "tok=val; Path=/; Domain=new.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteCookieDomain(tt.raw, tt.newDomain)
			if got != tt.want {
				t.Errorf("rewriteCookieDomain(%q, %q)\n  got  = %q\n  want = %q", tt.raw, tt.newDomain, got, tt.want)
			}
		})
	}
}

// ── filterOneIDCookies 单元测试（纯函数，table-driven） ──────────────────────

func TestOneidUnified_FilterOneIDCookies(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "empty string returns empty",
			raw:  "",
			want: "",
		},
		{
			name: "keeps state_token only",
			raw:  "state_token=abc123; hatchery-session=xyz; other=foo",
			want: "state_token=abc123",
		},
		{
			name: "keeps session_token only",
			raw:  "session_token=def456; hatchery-session=xyz",
			want: "session_token=def456",
		},
		{
			name: "keeps both state_token and session_token",
			raw:  "state_token=abc; session_token=def; other=bar",
			want: "state_token=abc; session_token=def",
		},
		{
			name: "no relevant cookies returns empty",
			raw:  "hatchery-session=xyz; other=foo; tracking=bar",
			want: "",
		},
		{
			name: "handles whitespace in cookie string",
			raw:  " state_token=abc ;  session_token=def ; extra=val ",
			want: "state_token=abc; session_token=def",
		},
		{
			name: "single state_token cookie",
			raw:  "state_token=abc",
			want: "state_token=abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterOneIDCookies(tt.raw)
			if got != tt.want {
				t.Errorf("filterOneIDCookies(%q)\n  got  = %q\n  want = %q", tt.raw, got, tt.want)
			}
		})
	}
}

// ── HandleOneIDAuthnProxy 单元测试 ──────────────────────────────────────────

func TestOneidUnified_HandleOneIDAuthnProxy_NotUnifiedMode(t *testing.T) {
	// Context without OneIDAppID → not unified mode
	req := httptest.NewRequest(http.MethodGet, "/v1/authn/encrypt_setting", nil)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "", // not unified mode
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnProxy(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "未处于统一账号模式") {
		t.Errorf("expected error message about unified mode, got=%s", w.Body.String())
	}
}

func TestOneidUnified_HandleOneIDAuthnProxy_GET(t *testing.T) {
	// Mock OneID backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/authn/encrypt_setting") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Verify cookies are filtered
		cookie := r.Header.Get("Cookie")
		if strings.Contains(cookie, "hatchery-session") {
			t.Errorf("hatchery-session should be filtered out, got Cookie=%s", cookie)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req-123")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"encrypt_type":"sm2"}`))
	}))
	defer backend.Close()

	// Override oneIDAPIBaseURL
	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = backend.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	req := httptest.NewRequest(http.MethodGet, "/v1/authn/encrypt_setting", nil)
	req.Header.Set("Cookie", "state_token=abc; hatchery-session=xyz")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
		Domain:     "https://my.example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnProxy(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type=application/json, got=%s", w.Header().Get("Content-Type"))
	}
	if w.Header().Get("X-Request-Id") != "req-123" {
		t.Errorf("expected X-Request-Id=req-123, got=%s", w.Header().Get("X-Request-Id"))
	}
	if w.Body.String() != `{"encrypt_type":"sm2"}` {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestOneidUnified_HandleOneIDAuthnProxy_POST(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"username":"test"}` {
			t.Errorf("unexpected request body: %s", string(body))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type=application/json, got=%s", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("Set-Cookie", "state_token=newval; Domain=account.tencent.com; Path=/; Secure")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"next":{"type":"CAPTCHA_OPTIONS"}}`))
	}))
	defer backend.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = backend.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	req := httptest.NewRequest(http.MethodPost, "/v1/authn/enterprise", strings.NewReader(`{"username":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
		Domain:     "https://my.example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnProxy(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	// Verify Set-Cookie domain is rewritten
	cookies := w.Header().Values("Set-Cookie")
	if len(cookies) == 0 {
		t.Fatal("expected Set-Cookie header")
	}
	if !strings.Contains(cookies[0], "Domain=my.example.com") {
		t.Errorf("expected Domain=my.example.com in Set-Cookie, got=%s", cookies[0])
	}
	if strings.Contains(cookies[0], "account.tencent.com") {
		t.Errorf("original domain should be replaced, got=%s", cookies[0])
	}
}

func TestOneidUnified_HandleOneIDAuthnProxy_QueryParams(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "foo=bar&baz=1" {
			t.Errorf("expected query params to be forwarded, got=%s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer backend.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = backend.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	req := httptest.NewRequest(http.MethodGet, "/v1/authn/encrypt_setting?foo=bar&baz=1", nil)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnProxy(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDAuthnProxy_BackendError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal"}`))
	}))
	defer backend.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = backend.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	req := httptest.NewRequest(http.MethodGet, "/v1/authn/encrypt_setting", nil)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnProxy(w, req)

	// Backend error should be passed through
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from backend, got %d", w.Code)
	}
}

// ── HandleOneIDAuthnLogin 单元测试 ──────────────────────────────────────────

func TestOneidUnified_HandleOneIDAuthnLogin_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/authn/enterprise", nil)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnLogin(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDAuthnLogin_NotUnifiedMode(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/authn/enterprise", strings.NewReader(`{}`))
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "", // not unified
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnLogin(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDAuthnLogin_OneIDReturnsNextStep(t *testing.T) {
	// OneID returns a "next step" (e.g. CAPTCHA_OPTIONS) → no session should be established
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("Set-Cookie", "state_token=abc; Domain=oneid.com; Path=/")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"next":{"type":"CAPTCHA_OPTIONS"}}`))
	}))
	defer backend.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = backend.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	// Setup Store for session operations
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() { Store = origStore }()

	req := httptest.NewRequest(http.MethodPost, "/oneid/enterprise", strings.NewReader(`{"password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
		Domain:     "https://my.example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnLogin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	// Verify Set-Cookie domain rewrite
	cookies := w.Header().Values("Set-Cookie")
	foundRewritten := false
	for _, c := range cookies {
		if strings.Contains(c, "state_token") && strings.Contains(c, "Domain=my.example.com") {
			foundRewritten = true
		}
	}
	if !foundRewritten {
		t.Errorf("expected rewritten Set-Cookie with Domain=my.example.com, got cookies=%v", cookies)
	}

	// Response should be the raw OneID response (no merged session data)
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, ok := resp["ok"]; ok {
		t.Error("should NOT have merged session data when next step is required")
	}
}

func TestOneidUnified_HandleOneIDAuthnLogin_SuccessWithSession(t *testing.T) {
	// OneID returns success (no next step) → session should be established
	unionId := "uid-123"
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/authn:get_self_v3" {
			// Mock get_self_v3 response
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{"data":{"accountUser":{"name":"loginuser","username":"loginuser","unionId":"%s"}},"errCode":"","errMessage":""}`, unionId)))
			return
		}
		// Login endpoint
		w.Header().Add("Set-Cookie", "session_token=newses; Domain=oneid.com; Path=/")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"user_id":"uid-123"}`))
	}))
	defer backend.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = backend.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	origWorkspaceURL := oneIDWorkspaceBaseURL
	oneIDWorkspaceBaseURL = backend.URL
	defer func() { oneIDWorkspaceBaseURL = origWorkspaceURL }()

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() { Store = origStore }()

	// Setup DB with the user (has one_id_sub matching unionId)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&model.User{}, &model.SiteConfig{}, &model.Instance{})
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	t.Cleanup(useDBForTestWithSafeRestore(db))

	db.Create(&model.User{Username: "loginuser", Role: "user", OneIDSub: &unionId})

	req := httptest.NewRequest(http.MethodPost, "/oneid/enterprise", strings.NewReader(`{"password":"pass"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
		Domain:     "https://my.example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnLogin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	// Response should contain merged session data
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got=%v", resp["ok"])
	}
	if resp["redirect"] != "/" {
		t.Errorf("expected redirect=/, got=%v", resp["redirect"])
	}
	if resp["role"] != "user" {
		t.Errorf("expected role=user, got=%v", resp["role"])
	}
}

func TestOneidUnified_HandleOneIDAuthnLogin_OneIDAuthFailed(t *testing.T) {
	// OneID returns 401 → no session, pass through
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errCode":"INVALID_CREDENTIAL","errMsg":"bad password"}`))
	}))
	defer backend.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = backend.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() { Store = origStore }()

	req := httptest.NewRequest(http.MethodPost, "/oneid/enterprise", strings.NewReader(`{"password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnLogin(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", w.Code, w.Body.String())
	}
}

// ── HandleOneIDPasswordReset 单元测试 ────────────────────────────────────────

func TestOneidUnified_HandleOneIDPasswordReset_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/authn/enterprise/password:reset", nil)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDPasswordReset(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDPasswordReset_NotUnifiedMode(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/authn/enterprise/password:reset", strings.NewReader(`{}`))
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDPasswordReset(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDPasswordReset_Success(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "new_password") {
			t.Errorf("expected body to contain new_password, got=%s", body)
		}
		// Verify OneID cookies are forwarded
		cookie := r.Header.Get("Cookie")
		if !strings.Contains(cookie, "session_token") {
			t.Errorf("expected session_token cookie forwarded, got=%s", cookie)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("Set-Cookie", "session_token=newval; Domain=oneid.com; Path=/")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`)) // no errCode → success
	}))
	defer backend.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = backend.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() { Store = origStore }()

	req := httptest.NewRequest(http.MethodPost, "/v1/authn/enterprise/password:reset",
		strings.NewReader(`{"new_password":"newpass123"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "session_token=oldval; hatchery-session=xyz")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
		Domain:     "https://my.example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDPasswordReset(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	// Verify Set-Cookie domain is rewritten
	cookies := w.Header().Values("Set-Cookie")
	foundRewritten := false
	foundExpired := false
	for _, c := range cookies {
		if strings.Contains(c, "session_token") && strings.Contains(c, "Domain=my.example.com") {
			foundRewritten = true
		}
		if strings.Contains(c, "state_token=") && strings.Contains(c, "Max-Age=0") {
			foundExpired = true
		}
	}
	if !foundRewritten {
		t.Errorf("expected rewritten Set-Cookie, got cookies=%v", cookies)
	}
	if !foundExpired {
		t.Errorf("expected expired state_token cookie, got cookies=%v", cookies)
	}
}

func TestOneidUnified_HandleOneIDPasswordReset_OneIDError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errCode":"INVALID_STATE","errMsg":"state expired"}`))
	}))
	defer backend.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = backend.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() { Store = origStore }()

	req := httptest.NewRequest(http.MethodPost, "/v1/authn/enterprise/password:reset",
		strings.NewReader(`{"new_password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
		Domain:     "https://my.example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDPasswordReset(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (OneID responded 200), got %d", w.Code)
	}
	// Should NOT clear session when errCode is present
	cookies := w.Header().Values("Set-Cookie")
	for _, c := range cookies {
		if strings.Contains(c, "state_token=") && strings.Contains(c, "Max-Age=0") {
			t.Errorf("should NOT expire state_token when errCode present, got=%s", c)
		}
	}
}

// ── OneIDAddRoleUsers 单元测试 ───────────────────────────────────────────────

func TestOneidUnified_OneIDAddRoleUsers_NoGatewayURL(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = ""
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-123",
	})
	err := OneIDAddRoleUsers(ctx, []string{"uid1"})

	wanted := common.I18nError(i18n.MsgOneIDGatewayURLNotConfigured)
	if err == nil || !errors.Is(err, wanted) {
		t.Fatalf("expected GATEWAY_URL error, got=%v", err)
	}
}

func TestOneidUnified_OneIDAddRoleUsers_NoAccountID(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = "http://gateway.test"
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "", // empty
	})
	err := OneIDAddRoleUsers(ctx, []string{"uid1"})
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDAccountIDNotAvailable)) {
		t.Fatalf("expected account_id error, got=%v", err)
	}
}

func TestOneidUnified_OneIDAddRoleUsers_Success(t *testing.T) {
	var receivedBody map[string]interface{}
	var receivedHeaders http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/add-role-users" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		receivedHeaders = r.Header.Clone()
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-456",
		InternalSecret: "my-secret",
	})
	err := OneIDAddRoleUsers(ctx, []string{"uid1", "uid2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify payload
	if receivedBody["account_id"] != "acc-456" {
		t.Errorf("expected account_id=acc-456, got=%v", receivedBody["account_id"])
	}
	if receivedBody["role_id"] != "1400000" {
		t.Errorf("expected role_id=1400000, got=%v", receivedBody["role_id"])
	}
	uids, ok := receivedBody["union_ids"].([]interface{})
	if !ok || len(uids) != 2 {
		t.Errorf("expected 2 union_ids, got=%v", receivedBody["union_ids"])
	}

	// Verify headers
	if receivedHeaders.Get("X-Internal-Tenant") != "acc-456" {
		t.Errorf("expected X-Internal-Tenant=acc-456, got=%s", receivedHeaders.Get("X-Internal-Tenant"))
	}
	if receivedHeaders.Get("X-Internal-Token") == "" {
		t.Error("expected X-Internal-Token to be set")
	}
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type=application/json, got=%s", receivedHeaders.Get("Content-Type"))
	}
}

func TestOneidUnified_OneIDAddRoleUsers_GatewayError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`upstream error`))
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-456",
	})
	err := OneIDAddRoleUsers(ctx, []string{"uid1"})
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("expected gateway 502 error, got=%v", err)
	}
}

// ── OneIDRemoveRoleUsers 单元测试 ────────────────────────────────────────────

func TestOneidUnified_OneIDRemoveRoleUsers_NoGatewayURL(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = ""
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-123",
	})
	err := OneIDRemoveRoleUsers(ctx, []string{"uid1"})
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDGatewayURLNotConfigured)) {
		t.Fatalf("expected GATEWAY_URL error, got=%v", err)
	}
}

func TestOneidUnified_OneIDRemoveRoleUsers_NoAccountID(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = "http://gateway.test"
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "",
	})
	err := OneIDRemoveRoleUsers(ctx, []string{"uid1"})
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDAccountIDNotAvailable)) {
		t.Fatalf("expected account_id error, got=%v", err)
	}
}

func TestOneidUnified_OneIDRemoveRoleUsers_Success(t *testing.T) {
	var receivedBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/remove-role-users" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-789",
		InternalSecret: "secret2",
	})
	err := OneIDRemoveRoleUsers(ctx, []string{"uid3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["account_id"] != "acc-789" {
		t.Errorf("expected account_id=acc-789, got=%v", receivedBody["account_id"])
	}
	if receivedBody["role_id"] != "1400000" {
		t.Errorf("expected role_id=1400000, got=%v", receivedBody["role_id"])
	}
}

func TestOneidUnified_OneIDRemoveRoleUsers_GatewayError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`maintenance`))
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-789",
	})
	err := OneIDRemoveRoleUsers(ctx, []string{"uid1"})
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected gateway 503 error, got=%v", err)
	}
}

// ── OneIDResetPassword 单元测试 ──────────────────────────────────────────────

func TestOneidUnified_OneIDResetPassword_NoGatewayURL(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = ""
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-123",
	})
	err := OneIDResetPassword(ctx, "uid1", "newpass")
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDGatewayURLNotConfigured)) {
		t.Fatalf("expected GATEWAY_URL error, got=%v", err)
	}
}

func TestOneidUnified_OneIDResetPassword_NoAccountID(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = "http://gateway.test"
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "",
	})
	err := OneIDResetPassword(ctx, "uid1", "newpass")
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDAccountIDNotAvailable)) {
		t.Fatalf("expected account_id error, got=%v", err)
	}
}

func TestOneidUnified_OneIDResetPassword_Success(t *testing.T) {
	var receivedBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/reset-password" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-999",
		InternalSecret: "secret3",
	})
	err := OneIDResetPassword(ctx, "union-id-x", "hunter2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["account_id"] != "acc-999" {
		t.Errorf("expected account_id=acc-999, got=%v", receivedBody["account_id"])
	}
	if receivedBody["union_id"] != "union-id-x" {
		t.Errorf("expected union_id=union-id-x, got=%v", receivedBody["union_id"])
	}
	if receivedBody["password"] != "hunter2" {
		t.Errorf("expected password=hunter2, got=%v", receivedBody["password"])
	}
}

func TestOneidUnified_OneIDResetPassword_GatewayError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`forbidden`))
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-999",
	})
	err := OneIDResetPassword(ctx, "uid1", "pass")
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected gateway 403 error, got=%v", err)
	}
}

// ── fetchOneIDDomain 单元测试 ────────────────────────────────────────────────

func TestOneidUnified_FetchOneIDDomain_NoGateway(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = ""
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-123",
	})
	got := fetchOneIDDomain(ctx)
	if got != "" {
		t.Fatalf("expected empty domain when no gateway, got=%s", got)
	}
}

func TestOneidUnified_FetchOneIDDomain_NoTenantID(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = "http://gateway.test"
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "", // empty
	})
	got := fetchOneIDDomain(ctx)
	if got != "" {
		t.Fatalf("expected empty domain when no tenant ID, got=%s", got)
	}
}

func TestOneidUnified_FetchOneIDDomain_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/enterprise" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("tenant") != "ent-123" {
			t.Errorf("expected tenant=ent-123, got=%s", r.URL.Query().Get("tenant"))
		}
		// Verify internal auth headers
		if r.Header.Get("X-Internal-Tenant") != "ent-123" {
			t.Errorf("expected X-Internal-Tenant=ent-123, got=%s", r.Header.Get("X-Internal-Tenant"))
		}
		if r.Header.Get("X-Internal-Token") == "" {
			t.Error("expected X-Internal-Token to be set")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"account": map[string]interface{}{
					"account_id": "ent-123",
					"name":       "Test Corp",
					"domain":     "testcorp.oneid.com",
				},
			},
		})
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	// Setup DB for UpdateSiteConfig
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&model.SiteConfig{})
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	t.Cleanup(useDBForTestWithSafeRestore(db))
	db.Create(&model.SiteConfig{})

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "ent-123",
		InternalSecret: "my-secret",
	})
	got := fetchOneIDDomain(ctx)
	if got != "testcorp.oneid.com" {
		t.Fatalf("expected testcorp.oneid.com, got=%s", got)
	}
}

func TestOneidUnified_FetchOneIDDomain_GatewayError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`error`))
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "ent-123",
	})
	got := fetchOneIDDomain(ctx)
	if got != "" {
		t.Fatalf("expected empty domain on gateway error, got=%s", got)
	}
}

func TestOneidUnified_FetchOneIDDomain_EmptyDomain(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"account": map[string]interface{}{
					"account_id": "ent-123",
					"name":       "No Domain Corp",
					"domain":     "",
				},
			},
		})
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "ent-123",
	})
	got := fetchOneIDDomain(ctx)
	if got != "" {
		t.Fatalf("expected empty when domain field is empty, got=%s", got)
	}
}

func TestOneidUnified_FetchOneIDDomain_APICodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 10001,
			"msg":  "not found",
		})
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "ent-123",
	})
	got := fetchOneIDDomain(ctx)
	if got != "" {
		t.Fatalf("expected empty on API code error, got=%s", got)
	}
}

// ── HandleOneIDLoginName 单元测试 ───────────────────────────────────────────

func TestOneidUnified_LoginName_NotUnifiedMode(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/oneid/login-name?username=alice", nil)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "", // not unified mode
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDLoginName(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "未处于统一账号模式") {
		t.Errorf("expected error about unified mode, got=%s", w.Body.String())
	}
}

func TestOneidUnified_LoginName_MissingUsername(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/oneid/login-name", nil)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDLoginName(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "缺少 username 参数") {
		t.Errorf("expected error about missing username, got=%s", w.Body.String())
	}
}

func TestOneidUnified_LoginName_UserNotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&model.User{})
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	t.Cleanup(useDBForTestWithSafeRestore(db))

	req := httptest.NewRequest(http.MethodGet, "/oneid/login-name?username=nonexistent", nil)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDLoginName(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["error"] == nil || resp["error"] == "" {
		t.Fatalf("expected error field for nonexistent user, got %v", resp)
	}
}

func TestOneidUnified_LoginName_UserFoundWithLoginName(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&model.User{})
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	t.Cleanup(useDBForTestWithSafeRestore(db))

	loginName := "alice_oneid"
	db.Create(&model.User{Username: "alice", OneIDLoginName: &loginName})

	req := httptest.NewRequest(http.MethodGet, "/oneid/login-name?username=alice", nil)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDLoginName(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["username"] != "alice" {
		t.Errorf("expected username=alice, got=%v", resp["username"])
	}
	if resp["oneid_login_name"] != "alice_oneid" {
		t.Errorf("expected oneid_login_name=alice_oneid, got=%v", resp["oneid_login_name"])
	}
}

func TestOneidUnified_LoginName_UserFoundWithNilLoginName(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&model.User{})
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	t.Cleanup(useDBForTestWithSafeRestore(db))

	db.Create(&model.User{Username: "bob", OneIDLoginName: nil})

	req := httptest.NewRequest(http.MethodGet, "/oneid/login-name?username=bob", nil)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDLoginName(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["username"] != "bob" {
		t.Errorf("expected username=bob, got=%v", resp["username"])
	}
	if resp["oneid_login_name"] != "" {
		t.Errorf("expected oneid_login_name to be empty string, got=%v", resp["oneid_login_name"])
	}
}

// ── getOneIDAPIBaseURL / getOneIDWorkspaceBaseURL 单元测试 ──────────────────

func TestOneidUnified_GetOneIDAPIBaseURL_Domestic(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		DefaultLang: "zh",
	})
	got := getOneIDAPIBaseURL(ctx)
	if got != oneIDAPIBaseURL {
		t.Errorf("expected domestic URL %q, got %q", oneIDAPIBaseURL, got)
	}
}

func TestOneidUnified_GetOneIDAPIBaseURL_Overseas(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		DefaultLang: "en",
	})
	got := getOneIDAPIBaseURL(ctx)
	if got != oneIDAPIBaseURLOverseas {
		t.Errorf("expected overseas URL %q, got %q", oneIDAPIBaseURLOverseas, got)
	}
}

func TestOneidUnified_GetOneIDAPIBaseURL_NoLang_FallbackDomestic(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{})
	got := getOneIDAPIBaseURL(ctx)
	if got != oneIDAPIBaseURL {
		t.Errorf("expected domestic URL when lang empty, got %q", got)
	}
}

func TestOneidUnified_GetOneIDWorkspaceBaseURL_Domestic(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		DefaultLang: "zh",
	})
	got := getOneIDWorkspaceBaseURL(ctx)
	if got != oneIDWorkspaceBaseURL {
		t.Errorf("expected domestic workspace URL %q, got %q", oneIDWorkspaceBaseURL, got)
	}
}

func TestOneidUnified_GetOneIDWorkspaceBaseURL_Overseas(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		DefaultLang: "en",
	})
	got := getOneIDWorkspaceBaseURL(ctx)
	if got != oneIDWorkspaceBaseURLOverseas {
		t.Errorf("expected overseas workspace URL %q, got %q", oneIDWorkspaceBaseURLOverseas, got)
	}
}

func TestOneidUnified_GetOneIDWorkspaceBaseURL_NoLang_FallbackDomestic(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{})
	got := getOneIDWorkspaceBaseURL(ctx)
	if got != oneIDWorkspaceBaseURL {
		t.Errorf("expected domestic workspace URL when lang empty, got %q", got)
	}
}

// ── HandleOneIDAuthnProxy overseas URL routing ──────────────────────────────

func TestOneidUnified_HandleOneIDAuthnProxy_OverseasRouting(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer backend.Close()

	origOverseas := oneIDAPIBaseURLOverseas
	oneIDAPIBaseURLOverseas = backend.URL
	defer func() { oneIDAPIBaseURLOverseas = origOverseas }()

	req := httptest.NewRequest(http.MethodGet, "/v1/authn/encrypt_setting", nil)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID:  "app-123",
		DefaultLang: "en",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnProxy(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when overseas routing, got %d, body=%s", w.Code, w.Body.String())
	}
}

// ── generateRandomLoginName 单元测试 ────────────────────────────────────────

func TestOneidUnified_GenerateRandomLoginName_Prefix(t *testing.T) {
	name := generateRandomLoginName()
	if !strings.HasPrefix(name, "user_") {
		t.Errorf("expected prefix 'user_', got=%s", name)
	}
}

func TestOneidUnified_GenerateRandomLoginName_Length(t *testing.T) {
	name := generateRandomLoginName()
	// user_ (5 chars) + 8 hex chars (4 bytes = 8 hex) = 13
	if len(name) != 13 {
		t.Errorf("expected length 13, got=%d (%s)", len(name), name)
	}
}

func TestOneidUnified_GenerateRandomLoginName_DifferentOnSeparateCalls(t *testing.T) {
	// generateRandomLoginName uses time.Now().UnixNano() internally;
	// with a brief pause between calls the output should differ.
	name1 := generateRandomLoginName()
	time.Sleep(time.Millisecond)
	name2 := generateRandomLoginName()
	if name1 == name2 {
		t.Errorf("expected different values on separate calls, got same: %s", name1)
	}
}

// ── validateOneIDUsername 单元测试 ───────────────────────────────────────────

func TestOneidUnified_ValidateUsername(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid cases
		{name: "alphanumeric", input: "admin123", wantErr: false},
		{name: "with at sign", input: "test@email", wantErr: false},
		{name: "with underscore", input: "user_name", wantErr: false},
		{name: "special chars", input: "a!@#$%^&*()", wantErr: false},
		{name: "single char", input: "a", wantErr: false},
		{name: "191 chars", input: strings.Repeat("a", 191), wantErr: false},

		// Invalid cases
		{name: "empty string", input: "", wantErr: true},
		{name: "exceeds 191 chars", input: strings.Repeat("a", 192), wantErr: true},
		{name: "chinese characters", input: "张三", wantErr: true},
		{name: "mixed ascii and chinese", input: "hello你好", wantErr: true},
		{name: "contains space", input: "hello world", wantErr: true},
		{name: "contains tab", input: "hello\tworld", wantErr: true},
		{name: "unicode emoji", input: "user😀", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOneIDUsername(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOneIDUsername(%q) error=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestOneidUnified_GetOneIDAppToken_ForwardAcceptLanguage(t *testing.T) {
	var receivedAL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAL = r.Header.Get("Accept-Language")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token":"test-token-abc","expires_in":3600}`))
	}))
	defer srv.Close()

	clientID := "client-al-test"
	clearOneIDCaches(t, clientID)

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDClientID:      clientID,
		OneIDClientSecret:  "secret",
		OneIDTokenEndpoint: srv.URL,
	})
	ctx = i18n.SetAcceptLanguage(ctx, "en-US")

	token, err := getOneIDAppToken(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "test-token-abc" {
		t.Fatalf("expected token=test-token-abc, got=%s", token)
	}
	if receivedAL != "en-US" {
		t.Errorf("expected Accept-Language=en-US forwarded, got=%q", receivedAL)
	}
}

func TestOneidUnified_OneIDAPICall_ForwardAcceptLanguage(t *testing.T) {
	var receivedAL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAL = r.Header.Get("Accept-Language")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":0,"msg":"ok","data":{}}`))
	}))
	defer srv.Close()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{})
	ctx = i18n.SetAcceptLanguage(ctx, "zh-CN")

	_, err := oneIDAPICall(ctx, http.MethodGet, srv.URL+"/test", "fake-token", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedAL != "zh-CN" {
		t.Errorf("expected Accept-Language=zh-CN forwarded, got=%q", receivedAL)
	}
}

func TestOneidUnified_VerifyOneIDSessionUser_ForwardAcceptLanguage(t *testing.T) {
	var receivedAL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAL = r.Header.Get("Accept-Language")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"accountUser":{"name":"u","username":"u","unionId":"uid-1"}},"errCode":"","errMessage":""}`))
	}))
	defer srv.Close()

	origWorkspaceURL := oneIDWorkspaceBaseURL
	oneIDWorkspaceBaseURL = srv.URL
	defer func() { oneIDWorkspaceBaseURL = origWorkspaceURL }()

	ctx := i18n.SetAcceptLanguage(context.Background(), "ja")

	_, _, err := verifyOneIDSessionUser(ctx, "session-tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedAL != "ja" {
		t.Errorf("expected Accept-Language=ja forwarded, got=%q", receivedAL)
	}
}

func TestOneidUnified_OneIDAddRoleUsers_ForwardAcceptLanguage(t *testing.T) {
	var receivedAL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAL = r.Header.Get("Accept-Language")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-al-1",
	})
	ctx = i18n.SetAcceptLanguage(ctx, "en")

	err := OneIDAddRoleUsers(ctx, []string{"uid1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedAL != "en" {
		t.Errorf("expected Accept-Language=en forwarded, got=%q", receivedAL)
	}
}

func TestOneidUnified_OneIDRemoveRoleUsers_ForwardAcceptLanguage(t *testing.T) {
	var receivedAL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAL = r.Header.Get("Accept-Language")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-al-2",
	})
	ctx = i18n.SetAcceptLanguage(ctx, "zh-CN")

	err := OneIDRemoveRoleUsers(ctx, []string{"uid1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedAL != "zh-CN" {
		t.Errorf("expected Accept-Language=zh-CN forwarded, got=%q", receivedAL)
	}
}

func TestOneidUnified_OneIDResetPassword_ForwardAcceptLanguage(t *testing.T) {
	var receivedAL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAL = r.Header.Get("Accept-Language")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-al-3",
	})
	ctx = i18n.SetAcceptLanguage(ctx, "ko")

	err := OneIDResetPassword(ctx, "uid1", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedAL != "ko" {
		t.Errorf("expected Accept-Language=ko forwarded, got=%q", receivedAL)
	}
}

func TestOneidUnified_OneIDBatchResetPassword_ForwardAcceptLanguage(t *testing.T) {
	var receivedAL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAL = r.Header.Get("Accept-Language")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":0,"msg":"ok","data":{"success_count":1,"fail_count":0,"failures":[]}}`))
	}))
	defer srv.Close()

	origGW := GatewayURL
	GatewayURL = srv.URL
	defer func() { GatewayURL = origGW }()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-al-4",
	})
	ctx = i18n.SetAcceptLanguage(ctx, "en-US")

	result, err := OneIDBatchResetPassword(ctx, []PasswordResetItem{
		{UnionID: "uid1", Password: `{"hash":{"algorithm":"Bcrypt","value":"x"}}`},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SuccessCount != 1 {
		t.Errorf("expected success_count=1, got=%d", result.SuccessCount)
	}
	if receivedAL != "en-US" {
		t.Errorf("expected Accept-Language=en-US forwarded, got=%q", receivedAL)
	}
}

func TestOneidUnified_HandleOneIDAuthnProxy_ForwardAcceptLanguage(t *testing.T) {
	var receivedAL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAL = r.Header.Get("Accept-Language")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = srv.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	req := httptest.NewRequest(http.MethodGet, "/v1/authn/encrypt_setting", nil)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	ctx = i18n.SetAcceptLanguage(ctx, "fr")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnProxy(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if receivedAL != "fr" {
		t.Errorf("expected Accept-Language=fr forwarded, got=%q", receivedAL)
	}
}

func TestOneidUnified_HandleOneIDAuthnProxy_PostBodyReadError(t *testing.T) {
	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = "http://127.0.0.1:1"
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	req := httptest.NewRequest(http.MethodPost, "/v1/authn/enterprise", &errReader{})

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnProxy(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for body read error, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDAuthnProxy_BuildRequestError(t *testing.T) {
	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = "http://127.0.0.1:1/\x00invalid"
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	req := httptest.NewRequest(http.MethodPost, "/v1/authn/enterprise", strings.NewReader(`{}`))
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnProxy(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for build request error, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDAuthnProxy_BackendUnreachable(t *testing.T) {
	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = "http://127.0.0.1:1"
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	req := httptest.NewRequest(http.MethodGet, "/v1/authn/encrypt_setting", nil)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnProxy(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 for unreachable backend, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDPasswordReset_ForwardAcceptLanguage(t *testing.T) {
	var receivedAL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAL = r.Header.Get("Accept-Language")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = srv.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() { Store = origStore }()

	req := httptest.NewRequest(http.MethodPost, "/v1/authn/enterprise/password:reset",
		strings.NewReader(`{"new_password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
		Domain:     "https://my.example.com",
	})
	ctx = i18n.SetAcceptLanguage(ctx, "de")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDPasswordReset(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if receivedAL != "de" {
		t.Errorf("expected Accept-Language=de forwarded, got=%q", receivedAL)
	}
}

func TestOneidUnified_HandleOneIDPasswordReset_BodyReadError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/authn/enterprise/password:reset", &errReader{})
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDPasswordReset(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for body read error, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDPasswordReset_BuildRequestError(t *testing.T) {
	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = "http://127.0.0.1:1/\x00invalid"
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	req := httptest.NewRequest(http.MethodPost, "/v1/authn/enterprise/password:reset",
		strings.NewReader(`{"new_password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDPasswordReset(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for build request error, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDPasswordReset_BackendUnreachable(t *testing.T) {
	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = "http://127.0.0.1:1"
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	req := httptest.NewRequest(http.MethodPost, "/v1/authn/enterprise/password:reset",
		strings.NewReader(`{"new_password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDPasswordReset(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 for unreachable backend, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDAuthnLogin_ForwardAcceptLanguage(t *testing.T) {
	var receivedAL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAL = r.Header.Get("Accept-Language")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"next":{"type":"CAPTCHA_OPTIONS"}}`))
	}))
	defer srv.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = srv.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() { Store = origStore }()

	req := httptest.NewRequest(http.MethodPost, "/oneid/enterprise", strings.NewReader(`{"password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	ctx = i18n.SetAcceptLanguage(ctx, "es")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnLogin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if receivedAL != "es" {
		t.Errorf("expected Accept-Language=es forwarded, got=%q", receivedAL)
	}
}

func TestOneidUnified_HandleOneIDAuthnLogin_BodyReadError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/oneid/enterprise", &errReader{})
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnLogin(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for body read error, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDAuthnLogin_BuildRequestError(t *testing.T) {
	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = "http://127.0.0.1:1/\x00invalid"
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	req := httptest.NewRequest(http.MethodPost, "/oneid/enterprise", strings.NewReader(`{"password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnLogin(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for build request error, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDAuthnLogin_BackendUnreachable(t *testing.T) {
	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = "http://127.0.0.1:1"
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	req := httptest.NewRequest(http.MethodPost, "/oneid/enterprise", strings.NewReader(`{"password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnLogin(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 for unreachable backend, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDAuthnLogin_NoSessionToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"user_id":"uid-1"}`))
	}))
	defer srv.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = srv.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() { Store = origStore }()

	req := httptest.NewRequest(http.MethodPost, "/oneid/enterprise", strings.NewReader(`{"password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnLogin(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 for missing session_token, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDAuthnLogin_VerifySessionFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/authn:get_self_v3" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errCode":"INVALID_TOKEN"}`))
			return
		}
		w.Header().Add("Set-Cookie", "session_token=tok; Domain=oneid.com; Path=/")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"user_id":"uid-1"}`))
	}))
	defer srv.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = srv.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	origWorkspaceURL := oneIDWorkspaceBaseURL
	oneIDWorkspaceBaseURL = srv.URL
	defer func() { oneIDWorkspaceBaseURL = origWorkspaceURL }()

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() { Store = origStore }()

	req := httptest.NewRequest(http.MethodPost, "/oneid/enterprise", strings.NewReader(`{"password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
		Domain:     "https://my.example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnLogin(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 for verify session failed, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDAuthnLogin_EmptyUnionId(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/authn:get_self_v3" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"accountUser":{"name":"u","username":"u","unionId":""}},"errCode":"","errMessage":""}`))
			return
		}
		w.Header().Add("Set-Cookie", "session_token=tok; Domain=oneid.com; Path=/")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"user_id":"uid-1"}`))
	}))
	defer srv.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = srv.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	origWorkspaceURL := oneIDWorkspaceBaseURL
	oneIDWorkspaceBaseURL = srv.URL
	defer func() { oneIDWorkspaceBaseURL = origWorkspaceURL }()

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() { Store = origStore }()

	req := httptest.NewRequest(http.MethodPost, "/oneid/enterprise", strings.NewReader(`{"password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
		Domain:     "https://my.example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnLogin(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 for empty unionId, got %d", w.Code)
	}
}

func TestOneidUnified_HandleOneIDAuthnLogin_LocalUserNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/authn:get_self_v3" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"accountUser":{"name":"u","username":"u","unionId":"nonexistent-uid"}},"errCode":"","errMessage":""}`))
			return
		}
		w.Header().Add("Set-Cookie", "session_token=tok; Domain=oneid.com; Path=/")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"user_id":"uid-1"}`))
	}))
	defer srv.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = srv.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()

	origWorkspaceURL := oneIDWorkspaceBaseURL
	oneIDWorkspaceBaseURL = srv.URL
	defer func() { oneIDWorkspaceBaseURL = origWorkspaceURL }()

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() { Store = origStore }()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&model.User{}, &model.SiteConfig{}, &model.Instance{})
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	t.Cleanup(useDBForTestWithSafeRestore(db))

	req := httptest.NewRequest(http.MethodPost, "/oneid/enterprise", strings.NewReader(`{"password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-123",
		Domain:     "https://my.example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	HandleOneIDAuthnLogin(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for local user not found, got %d", w.Code)
	}
}

// ── setAcceptLanguageHeader 单元测试 ──────────────────────────────────────────

func TestOneidUnified_SetAcceptLanguageHeader_WithLang(t *testing.T) {
	ctx := i18n.SetAcceptLanguage(context.Background(), "en-US")
	req := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	setAcceptLanguageHeader(req, ctx)

	if got := req.Header.Get("Accept-Language"); got != "en-US" {
		t.Errorf("expected Accept-Language=en-US, got=%q", got)
	}
}

func TestOneidUnified_SetAcceptLanguageHeader_NoLang(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	setAcceptLanguageHeader(req, ctx)

	if got := req.Header.Get("Accept-Language"); got != "" {
		t.Errorf("expected empty Accept-Language when no lang in ctx, got=%q", got)
	}
}

func TestOneidUnified_SetAcceptLanguageHeader_EmptyLang(t *testing.T) {
	ctx := i18n.SetAcceptLanguage(context.Background(), "")
	req := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	setAcceptLanguageHeader(req, ctx)

	if got := req.Header.Get("Accept-Language"); got != "" {
		t.Errorf("expected empty Accept-Language when lang is empty string, got=%q", got)
	}
}

// ── setGatewayHeaders 单元测试 ────────────────────────────────────────────────

func TestOneidUnified_SetGatewayHeaders_FullSecret(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-full",
		InternalSecret: "my-hmac-secret",
	})
	ctx = i18n.SetAcceptLanguage(ctx, "zh-CN")
	req := httptest.NewRequest(http.MethodPost, "http://gateway/api/test", nil)

	setGatewayHeaders(req, ctx, "acc-full")

	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("expected Content-Type=application/json, got=%q", got)
	}
	if got := req.Header.Get("X-Internal-Tenant"); got != "acc-full" {
		t.Errorf("expected X-Internal-Tenant=acc-full, got=%q", got)
	}
	if got := req.Header.Get("Accept-Language"); got != "zh-CN" {
		t.Errorf("expected Accept-Language=zh-CN, got=%q", got)
	}
	token := req.Header.Get("X-Internal-Token")
	if token == "" {
		t.Error("expected X-Internal-Token to be set when InternalSecret is present")
	}
	// token format: <timestamp>.<hex-signature>
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		t.Errorf("expected X-Internal-Token format 'ts.sig', got=%q", token)
	}
	if parts[0] == "" || parts[1] == "" {
		t.Errorf("expected non-empty timestamp and signature, got=%q", token)
	}
}

func TestOneidUnified_SetGatewayHeaders_NoSecret(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-nosec",
		InternalSecret: "", // no secret
	})
	req := httptest.NewRequest(http.MethodPost, "http://gateway/api/test", nil)

	setGatewayHeaders(req, ctx, "acc-nosec")

	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("expected Content-Type=application/json, got=%q", got)
	}
	if got := req.Header.Get("X-Internal-Tenant"); got != "acc-nosec" {
		t.Errorf("expected X-Internal-Tenant=acc-nosec, got=%q", got)
	}
	// No Accept-Language set in ctx → should not be present
	if got := req.Header.Get("Accept-Language"); got != "" {
		t.Errorf("expected empty Accept-Language when no lang in ctx, got=%q", got)
	}
	// No InternalSecret → X-Internal-Token should NOT be set
	if got := req.Header.Get("X-Internal-Token"); got != "" {
		t.Errorf("expected empty X-Internal-Token when no secret, got=%q", got)
	}
}

func TestOneidUnified_SetGatewayHeaders_NoLangWithSecret(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-nolang",
		InternalSecret: "sec-1",
	})
	// no SetAcceptLanguage → Accept-Language should be absent
	req := httptest.NewRequest(http.MethodPost, "http://gateway/api/test", nil)

	setGatewayHeaders(req, ctx, "acc-nolang")

	if got := req.Header.Get("Accept-Language"); got != "" {
		t.Errorf("expected empty Accept-Language, got=%q", got)
	}
	if req.Header.Get("X-Internal-Token") == "" {
		t.Error("expected X-Internal-Token to be set when InternalSecret is present")
	}
}

func TestOneidUnified_SetGatewayHeaders_OverwritesExistingContentType(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAccountID: "acc-ow",
	})
	req := httptest.NewRequest(http.MethodPost, "http://gateway/api/test", nil)
	// pre-set a different Content-Type
	req.Header.Set("Content-Type", "text/plain")

	setGatewayHeaders(req, ctx, "acc-ow")

	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("expected Content-Type overwritten to application/json, got=%q", got)
	}
}

// ── forwardProxyHeaders 单元测试 ──────────────────────────────────────────────

func TestOneidUnified_ForwardProxyHeaders_ContentTypeAndCookie(t *testing.T) {
	origReq := httptest.NewRequest(http.MethodPost, "/v1/authn/enterprise", strings.NewReader(`{}`))
	origReq.Header.Set("Content-Type", "application/json")
	origReq.Header.Set("Cookie", "state_token=abc; hatchery-session=xyz; session_token=def")

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-1",
	})
	ctx = i18n.SetAcceptLanguage(ctx, "en")
	origReq = origReq.WithContext(ctx)

	proxyReq := httptest.NewRequest(http.MethodPost, "http://oneid/v1/authn/enterprise", nil)
	forwardProxyHeaders(proxyReq, origReq)

	if got := proxyReq.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("expected Content-Type=application/json, got=%q", got)
	}
	cookie := proxyReq.Header.Get("Cookie")
	if !strings.Contains(cookie, "state_token=abc") {
		t.Errorf("expected state_token in Cookie, got=%q", cookie)
	}
	if !strings.Contains(cookie, "session_token=def") {
		t.Errorf("expected session_token in Cookie, got=%q", cookie)
	}
	if strings.Contains(cookie, "hatchery-session") {
		t.Errorf("hatchery-session should be filtered out, got=%q", cookie)
	}
	if got := proxyReq.Header.Get("Accept-Language"); got != "en" {
		t.Errorf("expected Accept-Language=en, got=%q", got)
	}
}

func TestOneidUnified_ForwardProxyHeaders_NoContentType(t *testing.T) {
	origReq := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{}`))
	// intentionally not setting Content-Type
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-1",
	})
	origReq = origReq.WithContext(ctx)

	proxyReq := httptest.NewRequest(http.MethodPost, "http://oneid/test", nil)
	forwardProxyHeaders(proxyReq, origReq)

	if got := proxyReq.Header.Get("Content-Type"); got != "" {
		t.Errorf("expected empty Content-Type when original has none, got=%q", got)
	}
}

func TestOneidUnified_ForwardProxyHeaders_NoCookies(t *testing.T) {
	origReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	// no Cookie header set
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-1",
	})
	origReq = origReq.WithContext(ctx)

	proxyReq := httptest.NewRequest(http.MethodGet, "http://oneid/test", nil)
	forwardProxyHeaders(proxyReq, origReq)

	if got := proxyReq.Header.Get("Cookie"); got != "" {
		t.Errorf("expected empty Cookie when original has none, got=%q", got)
	}
}

func TestOneidUnified_ForwardProxyHeaders_OnlyIrrelevantCookies(t *testing.T) {
	origReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	origReq.Header.Set("Cookie", "hatchery-session=xyz; tracking=bar")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-1",
	})
	origReq = origReq.WithContext(ctx)

	proxyReq := httptest.NewRequest(http.MethodGet, "http://oneid/test", nil)
	forwardProxyHeaders(proxyReq, origReq)

	if got := proxyReq.Header.Get("Cookie"); got != "" {
		t.Errorf("expected empty Cookie when only irrelevant cookies present, got=%q", got)
	}
}

func TestOneidUnified_ForwardProxyHeaders_NoLangInCtx(t *testing.T) {
	origReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	origReq.Header.Set("Content-Type", "application/json")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-1",
	})
	// no SetAcceptLanguage
	origReq = origReq.WithContext(ctx)

	proxyReq := httptest.NewRequest(http.MethodGet, "http://oneid/test", nil)
	forwardProxyHeaders(proxyReq, origReq)

	if got := proxyReq.Header.Get("Accept-Language"); got != "" {
		t.Errorf("expected empty Accept-Language when no lang in ctx, got=%q", got)
	}
	// Content-Type should still be forwarded
	if got := proxyReq.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("expected Content-Type=application/json, got=%q", got)
	}
}

func TestOneidUnified_ForwardProxyHeaders_DoesNotModifyOriginalRequest(t *testing.T) {
	origReq := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{}`))
	origReq.Header.Set("Content-Type", "application/json")
	origReq.Header.Set("Cookie", "state_token=abc")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "app-1",
	})
	ctx = i18n.SetAcceptLanguage(ctx, "fr")
	origReq = origReq.WithContext(ctx)

	proxyReq := httptest.NewRequest(http.MethodPost, "http://oneid/test", nil)
	forwardProxyHeaders(proxyReq, origReq)

	// Original request should be unchanged
	if got := origReq.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("original Content-Type should be unchanged, got=%q", got)
	}
	if got := origReq.Header.Get("Cookie"); got != "state_token=abc" {
		t.Errorf("original Cookie should be unchanged, got=%q", got)
	}
	// proxyReq should have the forwarded values
	if got := proxyReq.Header.Get("Accept-Language"); got != "fr" {
		t.Errorf("expected proxy Accept-Language=fr, got=%q", got)
	}
}
