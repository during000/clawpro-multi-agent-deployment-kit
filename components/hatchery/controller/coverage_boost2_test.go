package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ─── access_log.go 覆盖 ────────────────────────────────────────────

func TestInjectRequestContext(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	// X-Request-ID header 作为 trace_id 的来源
	r.Header.Set("X-Request-ID", "req-abc")

	r2 := NewRequestContext(r)
	ctx := r2.Context()

	// request_id 总是新生成的 UUID
	if common.GetRequestID(ctx) == "" {
		t.Error("request_id should be generated")
	}
	// trace_id 来自 X-Request-ID header
	if common.GetTraceID(ctx) != "req-abc" {
		t.Errorf("trace_id want req-abc, got %q", common.GetTraceID(ctx))
	}
	if common.GetInterface(ctx) != "/test/path" {
		t.Errorf("interface want /test/path, got %q", common.GetInterface(ctx))
	}
}

func TestInjectRequestContext_GeneratesIDs(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1", nil)
	r2 := NewRequestContext(r)
	ctx := r2.Context()

	if common.GetRequestID(ctx) == "" {
		t.Error("should generate request_id")
	}
	// 无 header 时 trace_id fallback 到 request_id
	if common.GetTraceID(ctx) != common.GetRequestID(ctx) {
		t.Error("trace_id should fallback to request_id")
	}
}

func TestInjectRequestContext_WithTenantUin(t *testing.T) {
	snap := common.TenantSnapshot{Identifier: "t1", Uin: "uin-from-snap"}
	ctx := common.InjectTenant(context.Background(), snap)
	r := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)

	r2 := NewRequestContext(r)
	// GetUin 已移除，通过 CVMUinFromCtx 从 TenantSnapshot 验证 uin
	if common.CVMUinFromCtx(r2.Context()) != "uin-from-snap" {
		t.Errorf("want uin-from-snap, got %q", common.CVMUinFromCtx(r2.Context()))
	}
}

func TestResolveUinFromCtx_Fallback(t *testing.T) {
	// ctx 无 TenantSnapshot 时，resolveUinFromCtx 返回空串（不兜底 FixedSnapshot）
	uin := resolveUinFromCtx(context.Background())
	if uin != "" {
		t.Errorf("ctx 无 snapshot 时 want empty, got %q", uin)
	}
}

func TestInjectSubUin(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r2 := InjectSubUin(r, 42)
	if common.GetSubUin(r2.Context()) != 42 {
		t.Errorf("want 42, got %d", common.GetSubUin(r2.Context()))
	}
}

// ─── identifier_middleware.go 覆盖 ──────────────────────────────────

func TestIsTenantAgnosticPath_Cov(t *testing.T) {
	if !isTenantAgnosticPath("/health") {
		t.Error("/health should be tenant agnostic")
	}
	if isTenantAgnosticPath("/api/v1/test") {
		t.Error("/api/v1/test should not be tenant agnostic")
	}
}

func TestIdentifierMiddleware_WithSnapshot_Cov(t *testing.T) {
	oldSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnap }()
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "test-id", Uin: "uin-test"}

	var gotSnap common.TenantSnapshot
	handler := IdentifierMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snap, ok := common.GetTenantSnapshot(r.Context())
		if ok {
			gotSnap = snap
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(w, r)

	if gotSnap.Identifier != "test-id" {
		t.Errorf("want test-id, got %q", gotSnap.Identifier)
	}
}

func TestIdentifierMiddleware_HealthBypass_Cov(t *testing.T) {
	oldSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnap }()
	common.FixedSnapshot = nil

	called := false
	handler := IdentifierMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	handler.ServeHTTP(w, r)

	if !called {
		t.Error("/health should bypass middleware")
	}
}

// ─── email.go 覆盖 ──────────────────────────────────────────────────

func TestSendEmail_NoURL(t *testing.T) {
	setupMemoryProDB(t)
	oldURL := EmailAPIURL
	defer func() { EmailAPIURL = oldURL }()
	EmailAPIURL = ""

	err := sendEmail(context.Background(), "to@x.com", 1, "ap-guangzhou", "", nil)
	if err == nil {
		t.Error("should fail when apiURL is empty")
	}
}

// ─── sts.go 覆盖 ────────────────────────────────────────────────────

func TestRefreshSTSCredentials_NoCreds(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.SiteConfig{})
	db.Create(&model.SiteConfig{Name: "Test"}) // no CVMSecretId/Key
	t.Cleanup(model.UseDBForTest(db))
	err := RefreshSTSCredentials(context.Background())
	if err == nil {
		t.Error("should fail when CVMSecretId/Key not configured")
	}
}

func TestRunSTSRefreshOnce_NoCreds(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.SiteConfig{})
	db.Create(&model.SiteConfig{Name: "Test"})
	t.Cleanup(model.UseDBForTest(db))
	// Should not panic, just log error
	_ = RefreshSTSCredentials(context.Background())
}

func TestGetCredential_NoCreds(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.SiteConfig{})
	db.Create(&model.SiteConfig{Name: "Test"})
	t.Cleanup(model.UseDBForTest(db))
	_, err := getCredential(context.Background())
	if err == nil {
		t.Error("should fail when credentials not configured")
	}
}

func TestGetCredential_PermanentCreds(t *testing.T) {
	origSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = origSnap }()
	common.FixedSnapshot = &common.TenantSnapshot{
		Identifier:     origSnap.Identifier,
		Domain:         origSnap.Domain,
		Uin:            "", // no UIN = use permanent credentials
		InternalSecret: origSnap.InternalSecret,
	}

	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.SiteConfig{})
	db.Create(&model.SiteConfig{Name: "Test", CVMSecretId: "id-123", CVMSecretKey: "key-456"})
	t.Cleanup(model.UseDBForTest(db))
	cred, err := getCredential(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred == nil {
		t.Fatal("credential should not be nil")
	}
}

func TestGetCredential_WithUinAndSnapshot(t *testing.T) {
	origSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = origSnap }()
	common.FixedSnapshot = &common.TenantSnapshot{
		Identifier:     origSnap.Identifier,
		Domain:         origSnap.Domain,
		Uin:            "",
		InternalSecret: origSnap.InternalSecret,
	}

	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.SiteConfig{})
	db.Create(&model.SiteConfig{Name: "Test", CVMSecretId: "id-x", CVMSecretKey: "key-x"})
	t.Cleanup(model.UseDBForTest(db))
	// With snapshot uin set → tries STS path → fails due to no STS credentials
	snap := common.TenantSnapshot{Identifier: "t1", Uin: "uin-snap"}
	ctx := common.InjectTenant(context.Background(), snap)

	_, err := getCredential(ctx)
	if err == nil {
		t.Error("should fail when STS credentials not available")
	}
}

// ─── openclaw_smh.go 覆盖通过 tat.go 简单函数 ───────────────────────

// ─── access_log.go 日志函数覆盖 ──────────────────────────────────────

func TestLogRcvRequest_Cov(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	// 不 panic 即为通过
	LogRcvRequest(context.Background(), r, []byte(`{"key":"value"}`))
}

func TestLogSendResponse_Cov(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	LogSendResponse(context.Background(), r, 200, nil, []byte(`ok`), 10*time.Millisecond)
	LogSendResponse(context.Background(), r, 400, nil, []byte(`bad`), 5*time.Millisecond)
	LogSendResponse(context.Background(), r, 500, nil, []byte(`err`), 1*time.Millisecond)
}

func TestLogCallHTTPAPI_Cov(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	LogCallHTTPAPI(context.Background(), r,
		"host", "1.2.3.4", "GET", "/api",
		nil, nil, "/action", "ap-beijing",
		200, nil, []byte("ok"),
		5*time.Millisecond, "http", true)
	LogCallHTTPAPI(context.Background(), r,
		"host", "1.2.3.4", "GET", "/api",
		nil, nil, "/action", "ap-beijing",
		500, nil, []byte("fail"),
		5*time.Millisecond, "http", false)
}

func TestLogUncaughtException_Cov(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	LogUncaughtException(context.Background(), r, "RuntimeError", 500, "unexpected", "stack trace")
}

func TestLogLLMStream_Cov(t *testing.T) {
	// LogLLMStream 只记录日志，不 panic 即为通过
	LogLLMStream(context.Background(), 200, nil, 100*time.Millisecond, nil)
	LogLLMStream(context.Background(), 500, nil, 50*time.Millisecond, nil)
	LogLLMStream(context.Background(), 400, nil, 10*time.Millisecond, nil)
}

func TestLogCallSDKAPI_Cov(t *testing.T) {
	logCallSDKAPI(context.Background(), "cvm", "RunInstances",
		map[string]string{"key": "val"}, map[string]string{"resp": "ok"},
		5*time.Millisecond, true, nil)
	logCallSDKAPI(context.Background(), "cvm", "RunInstances",
		nil, nil, 5*time.Millisecond, false, nil)
}

func TestLogCallDBAPI_Cov(t *testing.T) {
	logCallDBAPI(context.Background(), "SELECT", "users", "SELECT * FROM users",
		nil, 5, 2*time.Millisecond, true, nil)
}

func TestResolveUinFromCtx_NilSnapshot(t *testing.T) {
	origSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = origSnap }()
	common.FixedSnapshot = nil
	uin := resolveUinFromCtx(context.Background())
	if uin != "" {
		t.Errorf("want empty uin when snapshot is nil, got %q", uin)
	}
}

// ─── admin_notices.go 覆盖 ────────────────────────────────────────

func TestBuildConfigSteps_WithTenantID(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.SiteConfig{}, &model.User{}, &model.AIModel{}, &model.AIChannel{})
	db.Create(&model.SiteConfig{})
	t.Cleanup(model.UseDBForTest(db))

	// 注入一个带 TenantID 的 context（让 common.TenantIDFromCtx 返回非空，走 hasEnabledModel/hasEnabledChannel 分支）
	snap := common.TenantSnapshot{Identifier: "test-tenant", OneIDAccountID: "oneid-123"}
	ctx := common.InjectTenant(context.Background(), snap)
	steps := buildConfigSteps(ctx)
	if len(steps) == 0 {
		t.Error("buildConfigSteps should return non-empty steps")
	}
}

func TestHasEnterpriseUsers_Cov(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.User{})
	t.Cleanup(model.UseDBForTest(db))

	// 无用户
	if hasEnterpriseUsers(context.Background()) {
		t.Error("should be false with no users")
	}

	// 创建 1 个用户（count=1，不超过 1，仍为 false）
	db.Create(&model.User{Username: "admin", Password: "x", Role: "admin"})
	if hasEnterpriseUsers(context.Background()) {
		t.Error("should be false with only 1 user (not > 1)")
	}

	// 创建第 2 个用户（count=2 > 1，才为 true）
	db.Create(&model.User{Username: "ent1", Password: "x", Role: "enterprise"})
	if !hasEnterpriseUsers(context.Background()) {
		t.Error("should be true when more than 1 user exists")
	}
}

func TestHasEnabledModel_Cov(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.AIModel{})
	t.Cleanup(model.UseDBForTest(db))

	if hasEnabledModel(context.Background()) {
		t.Error("should be false with no models")
	}
	db.Create(&model.AIModel{ModelID: "m1", Enabled: true, Visible: true, ModelType: "openai-completions"})
	if !hasEnabledModel(context.Background()) {
		t.Error("should be true with enabled model")
	}
}

func TestHasEnabledChannel_Cov(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.AIChannel{})
	t.Cleanup(model.UseDBForTest(db))

	if hasEnabledChannel(context.Background()) {
		t.Error("should be false with no channels")
	}
	enabled := true
	db.Create(&model.AIChannel{Name: "ch1", ChannelID: "openai", Enabled: &enabled})
	if !hasEnabledChannel(context.Background()) {
		t.Error("should be true with enabled channel")
	}
}

// ─── tenant_init.go - coverage note ──────────────────────────────
// backfillSiteConfig is in the main package; covered via tenant_init_test.go.

// ─── session_identifier.go 覆盖 ──────────────────────────────────

func TestCurrentIdentifierFromCtx_FallbackToFixedSnapshot(t *testing.T) {
	// ctx 无 TenantSnapshot 时，currentIdentifierFromCtx 返回空串（不兜底 FixedSnapshot）
	id := currentIdentifierFromCtx(context.Background())
	if id != "" {
		t.Errorf("ctx 无 snapshot 时 want empty, got %q", id)
	}
}

// ─── audit.go WithAudit / WithCloudProxyAudit 覆盖 ─────────────────

func setupAuditTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.AutoMigrate(&model.AuditLog{}, &model.User{}, &model.SiteConfig{})
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-audit-secret-key-32bytes!!!"))
	t.Cleanup(func() { Store = origStore })
	return model.UseDBForTest(db)
}

// TestWithAudit_PostPath 覆盖 audit.go line 195（goroutine 调用 LogAudit）
func TestWithAudit_PostPath(t *testing.T) {
	cleanup := setupAuditTestDB(t)
	defer cleanup()

	called := false
	handler := WithAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	// 使用已注册的审计路径（/admin/user-groups/create 在 auditRules 中）
	r := httptest.NewRequest(http.MethodPost, "/admin/user-groups/create", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if !called {
		t.Error("handler should have been called")
	}
	// 等待 goroutine 完成（line 195 async）
	time.Sleep(10 * time.Millisecond)
}

// TestWithAudit_GetPath 验证 GET 方法直接 pass-through
func TestWithAudit_GetPath(t *testing.T) {
	called := false
	handler := WithAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/admin/users/create", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if !called {
		t.Error("handler should have been called for GET")
	}
}

// TestWithAudit_UnknownPath 验证未注册路径直接 pass-through
func TestWithAudit_UnknownPath(t *testing.T) {
	called := false
	handler := WithAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/admin/unknown-path", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if !called {
		t.Error("handler should have been called for unknown path")
	}
}

// TestWithCloudProxyAudit_PostPath 覆盖 audit.go line 240（goroutine 调用 LogAudit）
func TestWithCloudProxyAudit_PostPath(t *testing.T) {
	cleanup := setupAuditTestDB(t)
	defer cleanup()

	// 确保 FixedSnapshot 非 nil
	origSnap := common.FixedSnapshot
	if origSnap == nil {
		common.FixedSnapshot = &common.TenantSnapshot{}
		defer func() { common.FixedSnapshot = origSnap }()
	}

	called := false
	handler := WithCloudProxyAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/admin/cloud/mutate/cvm", nil)
	r.Header.Set("X-TC-Action", "RunInstances")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if !called {
		t.Error("handler should have been called")
	}
	// 等待 goroutine 完成（line 240 async）
	time.Sleep(10 * time.Millisecond)
}

// ─── tencent_clients.go GetCVMClient/GetVPCClient/GetSTSClient/GetCLSClient/GetTagClient ─

func setupTencentClientsTestDB(t *testing.T) func() {
	t.Helper()
	origSnap := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{
		Uin: "", // use permanent creds
	}
	t.Cleanup(func() { common.FixedSnapshot = origSnap })

	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.SiteConfig{})
	db.Create(&model.SiteConfig{Name: "Test", CVMSecretId: "id-abc", CVMSecretKey: "key-xyz"})
	return model.UseDBForTest(db)
}

func TestGetCVMClient_WithCreds(t *testing.T) {
	cleanup := setupTencentClientsTestDB(t)
	defer cleanup()

	cli, err := GetCVMClient(context.Background())
	if err != nil {
		t.Fatalf("GetCVMClient: %v", err)
	}
	if cli == nil {
		t.Error("client should not be nil")
	}
}

func TestGetVPCClient_WithCreds(t *testing.T) {
	cleanup := setupTencentClientsTestDB(t)
	defer cleanup()

	cli, err := GetVPCClient(context.Background())
	if err != nil {
		t.Fatalf("GetVPCClient: %v", err)
	}
	if cli == nil {
		t.Error("client should not be nil")
	}
}

func TestGetSTSClient_WithCreds(t *testing.T) {
	cleanup := setupTencentClientsTestDB(t)
	defer cleanup()

	cli, err := GetSTSClient(context.Background())
	if err != nil {
		t.Fatalf("GetSTSClient: %v", err)
	}
	if cli == nil {
		t.Error("client should not be nil")
	}
}

func TestGetCLSClient_WithCreds(t *testing.T) {
	cleanup := setupTencentClientsTestDB(t)
	defer cleanup()

	cli, err := GetCLSClient(context.Background())
	if err != nil {
		t.Fatalf("GetCLSClient: %v", err)
	}
	if cli == nil {
		t.Error("client should not be nil")
	}
}

func TestGetTagClient_WithCreds(t *testing.T) {
	cleanup := setupTencentClientsTestDB(t)
	defer cleanup()

	cli, err := GetTagClient(context.Background())
	if err != nil {
		t.Fatalf("GetTagClient: %v", err)
	}
	if cli == nil {
		t.Error("client should not be nil")
	}
}
