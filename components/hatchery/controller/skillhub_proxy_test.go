package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	hcommon "hatchery/common"
	"hatchery/model"
	"hatchery/skillhubclient"
)

// ── WithSkillHubProxy 装饰器测试 ──

func setupSkillHubProxyTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.SiteConfig{}, &model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

func TestWithSkillHubProxy_NonGrayscaleCallsLocal(t *testing.T) {
	setupSkillHubProxyTestDB(t)
	// skill_hub_enabled 默认 false，走 local handler

	localCalled := false
	skillhubCalled := false

	local := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		localCalled = true
		w.WriteHeader(http.StatusOK)
	})
	skillhub := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		skillhubCalled = true
	})

	proxy := WithSkillHubProxy(local, skillhub)
	req := httptest.NewRequest(http.MethodGet, "/admin/skills", nil)
	w := httptest.NewRecorder()
	proxy(w, req)

	if !localCalled {
		t.Error("local handler should be called when grayscale is disabled")
	}
	if skillhubCalled {
		t.Error("skillhub handler should NOT be called when grayscale is disabled")
	}
}

func TestWithSkillHubProxy_GrayscaleCallsSkillHub(t *testing.T) {
	setupSkillHubProxyTestDB(t)
	// 启用灰度，确保 SiteConfig 存在且 skill_hub_enabled=true
	model.DB(testCtx()).Create(&model.SiteConfig{SkillHubEnabled: true})

	localCalled := false
	skillhubCalled := false

	local := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		localCalled = true
	})
	skillhub := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		skillhubCalled = true
		w.WriteHeader(http.StatusOK)
	})

	proxy := WithSkillHubProxy(local, skillhub)

	req := httptest.NewRequest(http.MethodGet, "/admin/skills", nil)
	req = req.WithContext(injectTestTenant(testCtx()))
	w := httptest.NewRecorder()
	proxy(w, req)

	if localCalled {
		t.Error("local handler should NOT be called when grayscale is enabled")
	}
	if !skillhubCalled {
		t.Error("skillhub handler should be called when grayscale is enabled")
	}
}

// ── isSkillHubEnabled 测试 ──

func TestIsSkillHubEnabled_DisabledByDefault(t *testing.T) {
	setupSkillHubProxyTestDB(t)
	// 创建一条 SiteConfig（默认 skill_hub_enabled=false）
	model.DB(testCtx()).Create(&model.SiteConfig{})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(injectTestTenant(testCtx()))

	if isSkillHubEnabled(req) {
		t.Error("should be disabled by default")
	}
}

func TestIsSkillHubEnabled_Enabled(t *testing.T) {
	setupSkillHubProxyTestDB(t)
	model.DB(testCtx()).Create(&model.SiteConfig{SkillHubEnabled: true})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(injectTestTenant(testCtx()))

	if !isSkillHubEnabled(req) {
		t.Error("should be enabled when skill_hub_enabled=true")
	}
}

// ── HandleSkillHubStatus 测试 ──

func TestHandleSkillHubStatus_Disabled(t *testing.T) {
	setupSkillHubProxyTestDB(t)
	origToken := AdminToken
	AdminToken = "test-token"
	t.Cleanup(func() { AdminToken = origToken })

	model.DB(testCtx()).Create(&model.SiteConfig{
		SkillHubEnabled: false,
		SkillHubAPIURL:  "https://api.skillhub.cn",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/skillhub-status", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req = req.WithContext(injectTestTenant(testCtx()))
	w := httptest.NewRecorder()

	HandleSkillHubStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, `"enabled":false`) {
		t.Errorf("body should contain enabled:false: %s", body)
	}
}

func TestHandleSkillHubStatus_Enabled(t *testing.T) {
	setupSkillHubProxyTestDB(t)
	origToken := AdminToken
	AdminToken = "test-token"
	t.Cleanup(func() { AdminToken = origToken })

	model.DB(testCtx()).Create(&model.SiteConfig{
		SkillHubEnabled: true,
		SkillHubAPIURL:  "https://api.skillhub.cn",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/skillhub-status", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req = req.WithContext(injectTestTenant(testCtx()))
	w := httptest.NewRecorder()

	HandleSkillHubStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"enabled":true`) {
		t.Errorf("body should contain enabled:true: %s", body)
	}
	// skillhub_url 应该是 api.skillhub.cn 去掉 api. 前缀
	if !strings.Contains(body, `"skillhub_url":"https://skillhub.cn"`) {
		t.Errorf("body should contain skillhub_url without api. prefix: %s", body)
	}
}

func TestHandleSkillHubStatus_NoApiPrefix(t *testing.T) {
	setupSkillHubProxyTestDB(t)
	origToken := AdminToken
	AdminToken = "test-token"
	t.Cleanup(func() { AdminToken = origToken })

	// URL 不含 api. 前缀时，skillhub_url 等于原始 URL
	model.DB(testCtx()).Create(&model.SiteConfig{
		SkillHubEnabled: true,
		SkillHubAPIURL:  "https://skillhub.example.com",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/skillhub-status", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req = req.WithContext(injectTestTenant(testCtx()))
	w := httptest.NewRecorder()

	HandleSkillHubStatus(w, req)

	body := w.Body.String()
	if !strings.Contains(body, `"skillhub_url":"https://skillhub.example.com"`) {
		t.Errorf("body should contain unchanged URL: %s", body)
	}
}

func TestHandleSkillHubStatus_Unauthorized(t *testing.T) {
	setupSkillHubProxyTestDB(t)
	origToken := AdminToken
	AdminToken = "test-token"
	t.Cleanup(func() { AdminToken = origToken })

	// 初始化 Store（requireAdmin → getSession 需要 Store）
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key"))
	t.Cleanup(func() { Store = origStore })

	req := httptest.NewRequest(http.MethodGet, "/admin/skillhub-status", nil)
	// 不设置 Authorization header
	req = req.WithContext(injectTestTenant(testCtx()))
	w := httptest.NewRecorder()

	HandleSkillHubStatus(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// ── getSkillHubAccessToken 测试 ──

func TestGetSkillHubAccessToken_GatewayURLEmpty(t *testing.T) {
	origGateway := GatewayURL
	GatewayURL = ""
	t.Cleanup(func() { GatewayURL = origGateway })

	_, err := getSkillHubAccessToken(testCtx(), testSnap(), "sub", "tid")
	if err == nil {
		t.Error("expected error when GatewayURL is empty")
	}
	if !strings.Contains(err.Error(), "GATEWAY_URL") {
		t.Errorf("error should mention GATEWAY_URL: %v", err)
	}
}

func TestGetSkillHubAccessToken_InternalSecretEmpty(t *testing.T) {
	origGateway := GatewayURL
	GatewayURL = "http://gateway.test"
	t.Cleanup(func() { GatewayURL = origGateway })

	snap := testSnap()
	snap.InternalSecret = ""

	_, err := getSkillHubAccessToken(testCtx(), snap, "sub", "tid")
	if err == nil {
		t.Error("expected error when InternalSecret is empty")
	}
	if !strings.Contains(err.Error(), "internal_secret") {
		t.Errorf("error should mention internal_secret: %v", err)
	}
}

func TestGetSkillHubAccessToken_SuccessViaMockGateway(t *testing.T) {
	// mock Gateway /api/access-token
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/access-token" {
			t.Errorf("path = %q, want /api/access-token", r.URL.Path)
		}
		// 验证 X-Internal-Token 存在
		if r.Header.Get("X-Internal-Token") == "" {
			t.Error("missing X-Internal-Token header")
		}
		// 验证 X-Internal-Tenant
		if r.Header.Get("X-Internal-Tenant") != "tid-123" {
			t.Errorf("X-Internal-Tenant = %q", r.Header.Get("X-Internal-Tenant"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"mock-token-xyz","expires_in":1800}`))
	}))
	defer srv.Close()

	origGateway := GatewayURL
	GatewayURL = srv.URL
	t.Cleanup(func() { GatewayURL = origGateway })

	snap := testSnap()
	snap.InternalSecret = "test-secret"

	// 清缓存
	skillHubTokenMu.Lock()
	skillHubTokenCache = map[string]*skillHubTokenEntry{}
	skillHubTokenMu.Unlock()

	token, err := getSkillHubAccessToken(testCtx(), snap, "sub-123", "tid-123")
	if err != nil {
		t.Fatalf("getSkillHubAccessToken: %v", err)
	}
	if token != "mock-token-xyz" {
		t.Errorf("token = %q, want mock-token-xyz", token)
	}
}

func TestGetSkillHubAccessToken_CacheHit(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte(`{"access_token":"cached-token","expires_in":3600}`))
	}))
	defer srv.Close()

	origGateway := GatewayURL
	GatewayURL = srv.URL
	t.Cleanup(func() { GatewayURL = origGateway })

	snap := testSnap()
	snap.InternalSecret = "test-secret"
	snap.Identifier = "test-cache"

	// 清缓存
	skillHubTokenMu.Lock()
	skillHubTokenCache = map[string]*skillHubTokenEntry{}
	skillHubTokenMu.Unlock()

	// 第一次调用
	token1, _ := getSkillHubAccessToken(testCtx(), snap, "sub", "tid")
	// 第二次调用（应命中缓存）
	token2, _ := getSkillHubAccessToken(testCtx(), snap, "sub", "tid")

	if token1 != token2 {
		t.Errorf("token1 = %q, token2 = %q, should be same (cache)", token1, token2)
	}
	if callCount != 1 {
		t.Errorf("gateway called %d times, want 1 (second should be cache hit)", callCount)
	}
}

func TestGetSkillHubAccessToken_GatewayError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error":"upstream error"}`))
	}))
	defer srv.Close()

	origGateway := GatewayURL
	GatewayURL = srv.URL
	t.Cleanup(func() { GatewayURL = origGateway })

	snap := testSnap()
	snap.InternalSecret = "test-secret"
	snap.Identifier = "test-err"

	skillHubTokenMu.Lock()
	delete(skillHubTokenCache, "test-err:sub:tid")
	skillHubTokenMu.Unlock()

	_, err := getSkillHubAccessToken(testCtx(), snap, "sub", "tid")
	if err == nil {
		t.Error("expected error when gateway returns 502")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("error should contain 502: %v", err)
	}
}

func TestGetSkillHubAccessToken_EmptyToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"access_token":"","expires_in":1800}`))
	}))
	defer srv.Close()

	origGateway := GatewayURL
	GatewayURL = srv.URL
	t.Cleanup(func() { GatewayURL = origGateway })

	snap := testSnap()
	snap.InternalSecret = "test-secret"
	snap.Identifier = "test-empty"

	skillHubTokenMu.Lock()
	delete(skillHubTokenCache, "test-empty:sub:tid")
	skillHubTokenMu.Unlock()

	_, err := getSkillHubAccessToken(testCtx(), snap, "sub", "tid")
	if err == nil {
		t.Error("expected error for empty access_token")
	}
}

// ── getSkillHubOrgID 测试 ──

func TestGetSkillHubOrgID_CacheHit(t *testing.T) {
	// 预填充缓存
	skillHubOrgMu.Lock()
	skillHubOrgCache["test-org-cache"] = &skillhubclient.OrgInfo{OrgID: 42, OrgPublicID: "org-test"}
	skillHubOrgMu.Unlock()
	t.Cleanup(func() {
		skillHubOrgMu.Lock()
		delete(skillHubOrgCache, "test-org-cache")
		skillHubOrgMu.Unlock()
	})

	snap := testSnap()
	snap.Identifier = "test-org-cache"

	orgID, err := getSkillHubOrgID(testCtx(), snap, "http://example.com", "token")
	if err != nil {
		t.Fatalf("getSkillHubOrgID: %v", err)
	}
	if orgID != 42 {
		t.Errorf("orgID = %d, want 42", orgID)
	}
}

func TestGetSkillHubOrgID_FetchViaMockServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/me" {
			t.Errorf("path = %q, want /api/v1/auth/me", r.URL.Path)
		}
		w.Write([]byte(`{"user":{"enterprise":{"orgId":99,"orgPublicId":"org-fetched"}}}`))
	}))
	defer srv.Close()

	snap := testSnap()
	snap.Identifier = "test-org-fetch"

	// 清缓存确保走 fetch
	skillHubOrgMu.Lock()
	delete(skillHubOrgCache, "test-org-fetch")
	skillHubOrgMu.Unlock()
	t.Cleanup(func() {
		skillHubOrgMu.Lock()
		delete(skillHubOrgCache, "test-org-fetch")
		skillHubOrgMu.Unlock()
	})

	orgID, err := getSkillHubOrgID(testCtx(), snap, srv.URL, "token")
	if err != nil {
		t.Fatalf("getSkillHubOrgID: %v", err)
	}
	if orgID != 99 {
		t.Errorf("orgID = %d, want 99", orgID)
	}

	// 验证已缓存
	skillHubOrgMu.Lock()
	cached := skillHubOrgCache["test-org-fetch"]
	skillHubOrgMu.Unlock()
	if cached == nil {
		t.Fatal("OrgInfo not cached after fetch")
	}
	if cached.OrgID != 99 {
		t.Errorf("cached OrgID = %d, want 99", cached.OrgID)
	}
}

// ── skillHubClientOrFail 测试 ──

func TestSkillHubClientOrFail_SkillHubAPIURLEmpty(t *testing.T) {
	setupSkillHubProxyTestDB(t)
	origToken := AdminToken
	AdminToken = "test-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key"))
	t.Cleanup(func() {
		AdminToken = origToken
		Store = origStore
	})

	// 不设置 skill_hub_api_url
	model.DB(testCtx()).Create(&model.SiteConfig{})

	req := httptest.NewRequest(http.MethodGet, "/admin/skills", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req = req.WithContext(injectTestTenant(testCtx()))
	w := httptest.NewRecorder()

	client := skillHubClientOrFail(w, req)
	if client != nil {
		t.Error("client should be nil when SkillHubAPIURL is empty")
	}
	if w.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", w.Code)
	}
}

// ── HandleAdminSkillsViaSkillHub 测试 ──

func TestHandleAdminSkillsViaSkillHub_Unauthorized(t *testing.T) {
	setupSkillHubProxyTestDB(t)
	origToken := AdminToken
	AdminToken = "test-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key"))
	t.Cleanup(func() {
		AdminToken = origToken
		Store = origStore
	})

	model.DB(testCtx()).Create(&model.SiteConfig{SkillHubEnabled: true, SkillHubAPIURL: "https://api.skillhub.cn"})

	req := httptest.NewRequest(http.MethodGet, "/admin/skills", nil)
	// 不设置 Authorization header
	req = req.WithContext(injectTestTenant(testCtx()))
	w := httptest.NewRecorder()

	HandleAdminSkillsViaSkillHub(w, req)

	if w.Code != http.StatusUnauthorized && w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 401 or 403", w.Code)
	}
}

func TestHandleAdminSkillsViaSkillHub_Success(t *testing.T) {
	setupSkillHubProxyTestDB(t)
	origToken := AdminToken
	AdminToken = "test-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key"))
	t.Cleanup(func() {
		AdminToken = origToken
		Store = origStore
	})

	model.DB(testCtx()).Create(&model.SiteConfig{
		SkillHubEnabled: true,
		SkillHubAPIURL:  "https://api.skillhub.cn",
	})

	// 创建有 OneIDSub 的用户
	user := model.User{Username: "admin", Role: "admin", OneIDSub: strPtr("test-sub")}
	model.DB(testCtx()).Create(&user)

	// mock Gateway + SkillHub
	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/access-token" {
			w.Write([]byte(`{"access_token":"test-token","expires_in":1800}`))
			return
		}
		if r.URL.Path == "/api/v1/auth/me" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"user":{"enterprise":{"orgId":17,"orgPublicId":"org-test"}}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"total":1,"items":[{"id":1,"display_name":"Test Skill","slug":"test-skill","version":"1.0.0","created_at":"2026-07-15T10:00:00Z","updated_at":"2026-07-15T12:00:00Z"}]}`))
	}))
	defer mockSrv.Close()

	origGateway := GatewayURL
	GatewayURL = mockSrv.URL
	t.Cleanup(func() { GatewayURL = origGateway })

	model.DB(testCtx()).Model(&model.SiteConfig{}).Where("1=1").Update("skill_hub_api_url", mockSrv.URL)

	skillHubTokenMu.Lock()
	skillHubTokenCache = map[string]*skillHubTokenEntry{}
	skillHubTokenMu.Unlock()
	skillHubOrgMu.Lock()
	skillHubOrgCache = map[string]*skillhubclient.OrgInfo{}
	skillHubOrgMu.Unlock()

	// 先保存 session 到一个 recorder，再从 recorder 提取 cookie 用于后续请求
	saveReq := httptest.NewRequest(http.MethodGet, "/admin/skills", nil)
	saveReq = saveReq.WithContext(injectTestTenant(testCtx()))
	saveW := httptest.NewRecorder()
	sess, _ := Store.Get(saveReq, "hatchery-session")
	sess.Values["username"] = "admin"
	sess.Values["role"] = "admin"
	sess.Values["identifier"] = "test-tenant"
	sess.Save(saveReq, saveW)

	// 从 saveW 提取 Set-Cookie
	cookies := saveW.Result().Header.Values("Set-Cookie")

	req := httptest.NewRequest(http.MethodGet, "/admin/skills?page=1&page_size=20", nil)
	// 不设 Authorization header，只靠 session 鉴权
	for _, c := range cookies {
		req.Header.Add("Cookie", c)
	}
	req = req.WithContext(injectTestTenant(testCtx()))
	w := httptest.NewRecorder()

	HandleAdminSkillsViaSkillHub(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, `"skills"`) {
		t.Errorf("body should contain 'skills' key: %s", body)
	}
	if !strings.Contains(body, `"test-skill"`) {
		t.Errorf("body should contain test-skill: %s", body)
	}
}

func TestHandleAdminSkillsViaSkillHub_APIError(t *testing.T) {
	setupSkillHubProxyTestDB(t)
	origToken := AdminToken
	AdminToken = "test-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key"))
	t.Cleanup(func() {
		AdminToken = origToken
		Store = origStore
	})

	model.DB(testCtx()).Create(&model.SiteConfig{
		SkillHubEnabled: true,
		SkillHubAPIURL:  "https://api.skillhub.cn",
	})

	user := model.User{Username: "admin", Role: "admin", OneIDSub: strPtr("test-sub")}
	model.DB(testCtx()).Create(&user)

	// Gateway 不可达
	origGateway := GatewayURL
	GatewayURL = "http://127.0.0.1:1"
	t.Cleanup(func() { GatewayURL = origGateway })

	skillHubTokenMu.Lock()
	skillHubTokenCache = map[string]*skillHubTokenEntry{}
	skillHubTokenMu.Unlock()
	skillHubOrgMu.Lock()
	skillHubOrgCache = map[string]*skillhubclient.OrgInfo{}
	skillHubOrgMu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/admin/skills", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Hatchery-OpenAPI", "1")
	ctx := injectTestTenant(testCtx())
	req = req.WithContext(ctx)

	sess, _ := Store.Get(req, "hatchery-session")
	sess.Values["username"] = "admin"
	sess.Values["role"] = "admin"
	sess.Save(req, httptest.NewRecorder())

	w := httptest.NewRecorder()
	HandleAdminSkillsViaSkillHub(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502, body = %s", w.Code, w.Body.String())
	}
}

// ── 测试辅助函数 ──

func testCtx() context.Context {
	return context.Background()
}

func testSnap() hcommon.TenantSnapshot {
	return hcommon.TenantSnapshot{
		Identifier:     "test-tenant",
		OneIDAccountID: "test-tid",
		InternalSecret: "test-secret",
	}
}

func injectTestTenant(ctx context.Context) context.Context {
	return hcommon.InjectTenant(ctx, testSnap())
}
