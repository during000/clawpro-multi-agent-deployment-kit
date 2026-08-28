package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"hatchery/common"
	"hatchery/model"
)

// setupGetConfigTest 统一初始化：内存 DB + local agent 表迁移 + 测试 session Store。
// 返回即装即用，每个测试第一行调用。
func setupGetConfigTest(t *testing.T) {
	t.Helper()
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	t.Cleanup(func() { Store = origStore })
}

// getConfigReq 构造带 session 登录的 GET /local-agent/get-config 请求。
// identifier 用于向 ctx 注入租户快照，模拟生产环境 middleware 的租户隔离（identifier 回调据此过滤）。
func getConfigReq(t *testing.T, username, identifier, query string) *http.Request {
	t.Helper()
	url := "/local-agent/get-config"
	if query != "" {
		url += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	req = req.WithContext(common.InjectTenant(req.Context(), common.TenantSnapshot{Identifier: identifier}))
	sess, _ := Store.Get(req, "hatchery-session")
	sess.Values["username"] = username
	rr := httptest.NewRecorder()
	if err := sess.Save(req, rr); err != nil {
		t.Fatalf("session save: %v", err)
	}
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

// getConfigResp 解析 get-config 响应体为 map。
func getConfigResp(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode resp: %v body=%s", err, rr.Body.String())
	}
	return body
}

// seedLocalAgentUser 创建测试用户并打开两层白名单（feature_allowlist + SiteConfig）。
func seedLocalAgentUser(t *testing.T, username, identifier string) model.User {
	t.Helper()
	ctx := context.Background()
	user := model.User{Username: username, Role: "user", Identifier: identifier}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := model.DB(ctx).Create(&model.FeatureAllowlist{
		Type:       model.FeatureAllowlistTypeLocalAgent,
		Identifier: identifier,
		Note:       "test",
	}).Error; err != nil {
		t.Fatalf("create allowlist: %v", err)
	}
	return user
}

// seedCLSCredential 写入该 identifier 的 cls 凭据（按租户隔离）。
func seedCLSCredential(t *testing.T, identifier, secretID, secretKey string) {
	t.Helper()
	ctx := context.Background()
	if err := model.DB(ctx).Create(&model.LocalAgentCLSCredential{
		Identifier: identifier,
		ConfigType: "cls",
		SecretID:   secretID,
		SecretKey:  secretKey,
	}).Error; err != nil {
		t.Fatalf("create cls credential: %v", err)
	}
}

// UT1: 未登录
func TestGetConfig_Unauthorized(t *testing.T) {
	setupGetConfigTest(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/local-agent/get-config?config_type=cls", nil)
	req.Header.Set("Accept", "application/json")
	HandleLocalAgentGetConfig(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("应 401，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// UT2: 白名单未放行（feature_allowlist 有记录但 identifier 不在）
func TestGetConfig_AllowlistBlocks(t *testing.T) {
	setupGetConfigTest(t)
	ctx := context.Background()
	if err := model.DB(ctx).Create(&model.FeatureAllowlist{
		Type: model.FeatureAllowlistTypeLocalAgent, Identifier: "allowed-tenant", Note: "pilot",
	}).Error; err != nil {
		t.Fatalf("create allowlist: %v", err)
	}
	user := model.User{Username: "blocked-user", Role: "user", Identifier: "blocked-tenant"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	rr := httptest.NewRecorder()
	HandleLocalAgentGetConfig(rr, getConfigReq(t, "blocked-user", "blocked-tenant", "config_type=cls"))
	if rr.Code != http.StatusForbidden {
		t.Errorf("应 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "未开放") {
		t.Errorf("响应应含「未开放」，实际=%s", rr.Body.String())
	}
}

// UT3: SiteConfig.LocalAgentEnabled=false
func TestGetConfig_SiteConfigDisabled(t *testing.T) {
	setupGetConfigTest(t)
	disableLocalAgentSiteConfig(t)
	ctx := context.Background()
	user := model.User{Username: "u", Role: "user", Identifier: "tenant-a"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := model.DB(ctx).Create(&model.FeatureAllowlist{
		Type: model.FeatureAllowlistTypeLocalAgent, Identifier: "tenant-a", Note: "test",
	}).Error; err != nil {
		t.Fatalf("create allowlist: %v", err)
	}
	rr := httptest.NewRecorder()
	HandleLocalAgentGetConfig(rr, getConfigReq(t, "u", "tenant-a", "config_type=cls"))
	if rr.Code != http.StatusForbidden {
		t.Errorf("应 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// UT4: config_type 缺失（非必传，返回全量=cls）
func TestGetConfig_MissingConfigType(t *testing.T) {
	setupGetConfigTest(t)
	seedLocalAgentUser(t, "u", "tenant-a")
	seedCLSCredential(t, "tenant-a", "AKID-test", "secret-test")
	origRegion := CVMRegion
	CVMRegion = "ap-guangzhou"
	t.Cleanup(func() { CVMRegion = origRegion })
	// mock CLS 开通校验通过，且 topic_id 实时返回
	orig := localAgentCheckCLSClawServiceOpened
	localAgentCheckCLSClawServiceOpened = func(ctx context.Context) (*CLSClawServiceResult, error) {
		return &CLSClawServiceResult{TraceTopicId: "clawpro-topic-xxx"}, nil
	}
	t.Cleanup(func() { localAgentCheckCLSClawServiceOpened = orig })

	rr := httptest.NewRecorder()
	HandleLocalAgentGetConfig(rr, getConfigReq(t, "u", "tenant-a", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("config_type 缺失应返回全量(200)，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	body := getConfigResp(t, rr)
	cls, ok := body["cls"].(map[string]any)
	if !ok {
		t.Fatalf("全量响应应含 cls 对象，实际=%v", body)
	}
	if cls["topic_id"] != "clawpro-topic-xxx" {
		t.Errorf("cls.topic_id = %v, want clawpro-topic-xxx", cls["topic_id"])
	}
}

// UT5: config_type 非法
func TestGetConfig_UnsupportedConfigType(t *testing.T) {
	setupGetConfigTest(t)
	seedLocalAgentUser(t, "u", "tenant-a")
	rr := httptest.NewRecorder()
	HandleLocalAgentGetConfig(rr, getConfigReq(t, "u", "tenant-a", "config_type=smh"))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "config_type 无效") {
		t.Errorf("响应应含「config_type 无效」，实际=%s", rr.Body.String())
	}
}

// UT6: CLS 服务未开通（CheckCLSClawServiceOpened 返回 nil）
func TestGetConfig_CLSNotOpened(t *testing.T) {
	setupGetConfigTest(t)
	seedLocalAgentUser(t, "u", "tenant-a")
	orig := localAgentCheckCLSClawServiceOpened
	localAgentCheckCLSClawServiceOpened = func(ctx context.Context) (*CLSClawServiceResult, error) {
		return nil, nil
	}
	t.Cleanup(func() { localAgentCheckCLSClawServiceOpened = orig })

	rr := httptest.NewRecorder()
	HandleLocalAgentGetConfig(rr, getConfigReq(t, "u", "tenant-a", "config_type=cls"))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "CLS 服务未开启") {
		t.Errorf("响应应含「CLS 服务未开启」，实际=%s", rr.Body.String())
	}
}

// UT7: CLS 返回 err
func TestGetConfig_CLSError(t *testing.T) {
	setupGetConfigTest(t)
	seedLocalAgentUser(t, "u", "tenant-a")
	orig := localAgentCheckCLSClawServiceOpened
	localAgentCheckCLSClawServiceOpened = func(ctx context.Context) (*CLSClawServiceResult, error) {
		return nil, context.DeadlineExceeded
	}
	t.Cleanup(func() { localAgentCheckCLSClawServiceOpened = orig })

	rr := httptest.NewRecorder()
	HandleLocalAgentGetConfig(rr, getConfigReq(t, "u", "tenant-a", "config_type=cls"))
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// UT8: 凭据表无数据
func TestGetConfig_CredentialNotReady(t *testing.T) {
	setupGetConfigTest(t)
	seedLocalAgentUser(t, "u", "tenant-a")
	orig := localAgentCheckCLSClawServiceOpened
	localAgentCheckCLSClawServiceOpened = func(ctx context.Context) (*CLSClawServiceResult, error) {
		return &CLSClawServiceResult{TraceTopicId: "topic-xxx"}, nil
	}
	t.Cleanup(func() { localAgentCheckCLSClawServiceOpened = orig })

	rr := httptest.NewRecorder()
	HandleLocalAgentGetConfig(rr, getConfigReq(t, "u", "tenant-a", "config_type=cls"))
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "CLS 凭据未配置") {
		t.Errorf("响应应含「CLS 凭据未配置」，实际=%s", rr.Body.String())
	}
}

// UT9: 正常路径
func TestGetConfig_OK(t *testing.T) {
	setupGetConfigTest(t)
	user := seedLocalAgentUser(t, "u", "tenant-a")
	seedCLSCredential(t, "tenant-a", "AKID-test", "secret-test")
	origRegion := CVMRegion
	CVMRegion = "ap-guangzhou"
	t.Cleanup(func() { CVMRegion = origRegion })
	// mock CLS 开通校验通过，且 topic_id 实时返回
	orig := localAgentCheckCLSClawServiceOpened
	localAgentCheckCLSClawServiceOpened = func(ctx context.Context) (*CLSClawServiceResult, error) {
		return &CLSClawServiceResult{TraceTopicId: "clawpro-topic-xxx"}, nil
	}
	t.Cleanup(func() { localAgentCheckCLSClawServiceOpened = orig })

	rr := httptest.NewRecorder()
	HandleLocalAgentGetConfig(rr, getConfigReq(t, "u", "tenant-a", "config_type=cls"))
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	body := getConfigResp(t, rr)
	cls, ok := body["cls"].(map[string]any)
	if !ok {
		t.Fatalf("响应应含 cls 对象，实际=%v", body)
	}
	want := map[string]any{
		"endpoint":    "ap-guangzhou.cls.tencentcs.com",
		"topic_id":    "clawpro-topic-xxx",
		"secret_id":   "AKID-test",
		"secret_key":  "secret-test",
		"user_id":     float64(user.ID),
		"user_name":   "u",
		"install_cmd": localAgentCLSInstallCmd,
		"run_cmd":     localAgentCLSRunCmd,
		"update_cmd":  localAgentCLSUpdateCmd,
		"uninstall_cmd": localAgentCLSUninstallCmd,
	}
	for k, v := range want {
		if cls[k] != v {
			t.Errorf("cls.%s = %v, want %v", k, cls[k], v)
		}
	}
}
