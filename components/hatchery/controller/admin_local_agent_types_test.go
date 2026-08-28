package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"

	"github.com/gorilla/sessions"
)

// TestHandleAdminLocalAgentTypes_Happy
// 正常路径：返 codebuddy + workbuddy + claude，按 localAgentTypes 数组顺序输出。
func TestHandleAdminLocalAgentTypes_Happy(t *testing.T) {
	setupSkillInstancesDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })

	rr := httptest.NewRecorder()
	HandleAdminLocalAgentTypes(rr, adminJSONGet("/admin/local-agent-types"))
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		LocalAgentTypes []LocalAgentTypeResponse `json:"local_agent_types"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	expectedCount := len(localAgentTypes)
	if len(resp.LocalAgentTypes) != expectedCount {
		t.Fatalf("期望 %d 条，实际=%d body=%s", expectedCount, len(resp.LocalAgentTypes), rr.Body.String())
	}
	// 顺序断言：与 localAgentTypes 数组顺序一致
	if resp.LocalAgentTypes[0].Code != "codebuddy" {
		t.Errorf("第 1 条应是 codebuddy，实际=%s", resp.LocalAgentTypes[0].Code)
	}
	if resp.LocalAgentTypes[1].Code != "workbuddy" {
		t.Errorf("第 2 条应是 workbuddy，实际=%s", resp.LocalAgentTypes[1].Code)
	}
	if resp.LocalAgentTypes[2].Code != "claude" {
		t.Errorf("第 3 条应是 claude，实际=%s", resp.LocalAgentTypes[2].Code)
	}
	// 字段完整性
	for i, item := range resp.LocalAgentTypes {
		if item.Name == "" || item.Description == "" {
			t.Errorf("第 %d 条 name/description 不应为空: %+v", i, item)
		}
	}
}

// TestHandleAdminLocalAgentTypes_MethodNotAllowed
func TestHandleAdminLocalAgentTypes_MethodNotAllowed(t *testing.T) {
	setupSkillInstancesDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/local-agent-types", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminLocalAgentTypes(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("应 405，实际=%d", rr.Code)
	}
}

// TestLocalAgentTypeWhitelist_DerivedFromLocalAgentTypes
// 守护测试：reporter 校验白名单必须与 /admin/local-agent-types 列表完全同源，
// 防止未来扩 type 时漏改一处导致管控页显示但 reporter 端拒。
func TestLocalAgentTypeWhitelist_DerivedFromLocalAgentTypes(t *testing.T) {
	if len(localAgentTypeWhitelist) != len(localAgentTypes) {
		t.Fatalf("白名单条数=%d 与 localAgentTypes 数组条数=%d 不一致",
			len(localAgentTypeWhitelist), len(localAgentTypes))
	}
	for _, item := range localAgentTypes {
		if !localAgentTypeWhitelist[item.Code] {
			t.Errorf("localAgentTypes 里有 %q 但白名单 map 没收录", item.Code)
		}
	}
}

// adminSessionReqForLocalAgentTypes 构造以某个 tenant admin session 登录的请求。
// 与 admin-token bearer 分开：白名单守卫仅对非 admin-token 的 tenant admin 生效。
func adminSessionReqForLocalAgentTypes(t *testing.T, username string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/admin/local-agent-types", nil)
	req.Header.Set("Accept", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username
	rr := httptest.NewRecorder()
	if err := session.Save(req, rr); err != nil {
		t.Fatalf("save session: %v", err)
	}
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

// setupLocalAgentTypesWhitelistTest 在 setupSkillInstancesDB 之上补上 session Store
// 与 tenant admin 用户，返回 cleanup。
func setupLocalAgentTypesWhitelistTest(t *testing.T, tenantID string) {
	t.Helper()
	setupSkillInstancesDB(t)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	t.Cleanup(func() { Store = origStore })

	user := model.User{
		Username:   "tenant-admin-" + tenantID,
		Role:       "admin",
		Identifier: tenantID,
	}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("create tenant admin: %v", err)
	}
}

// TestHandleAdminLocalAgentTypes_Whitelist_TenantNotAllowed
// 白名单表非空、当前 tenant admin 未命中 → 返 200 + 空数组（静默降级，不 403）。
func TestHandleAdminLocalAgentTypes_Whitelist_TenantNotAllowed(t *testing.T) {
	setupLocalAgentTypesWhitelistTest(t, "blocked-tenant")

	// 白名单只放行 allowed-tenant，当前 tenant='blocked-tenant' 未命中
	if err := model.DB(context.Background()).Create(&model.FeatureAllowlist{
		Type: model.FeatureAllowlistTypeLocalAgent, Identifier: "allowed-tenant",
	}).Error; err != nil {
		t.Fatalf("create allowlist: %v", err)
	}

	rr := httptest.NewRecorder()
	req := adminSessionReqForLocalAgentTypes(t, "tenant-admin-blocked-tenant")
	HandleAdminLocalAgentTypes(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("未命中白名单应仍 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		LocalAgentTypes []LocalAgentTypeResponse `json:"local_agent_types"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if len(resp.LocalAgentTypes) != 0 {
		t.Errorf("未命中白名单应返回空数组，实际=%d 条 body=%s",
			len(resp.LocalAgentTypes), rr.Body.String())
	}
}

// TestHandleAdminLocalAgentTypes_Whitelist_TenantAllowed
// 白名单命中 → 正常返回全量。
func TestHandleAdminLocalAgentTypes_Whitelist_TenantAllowed(t *testing.T) {
	setupLocalAgentTypesWhitelistTest(t, "allowed-tenant")

	if err := model.DB(context.Background()).Create(&model.FeatureAllowlist{
		Type: model.FeatureAllowlistTypeLocalAgent, Identifier: "allowed-tenant",
		Note: "pilot",
	}).Error; err != nil {
		t.Fatalf("create allowlist: %v", err)
	}

	rr := httptest.NewRecorder()
	req := adminSessionReqForLocalAgentTypes(t, "tenant-admin-allowed-tenant")
	HandleAdminLocalAgentTypes(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("命中白名单应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		LocalAgentTypes []LocalAgentTypeResponse `json:"local_agent_types"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if len(resp.LocalAgentTypes) != len(localAgentTypes) {
		t.Errorf("命中白名单应返全量 %d 条，实际=%d", len(localAgentTypes), len(resp.LocalAgentTypes))
	}
}

// TestHandleAdminLocalAgentTypes_Whitelist_EmptyTableAllOpen
// 白名单表空 → IsFeatureAllowed 语义"全开"→ 正常返回全量。
func TestHandleAdminLocalAgentTypes_Whitelist_EmptyTableAllOpen(t *testing.T) {
	setupLocalAgentTypesWhitelistTest(t, "any-tenant")
	// 故意不建 FeatureAllowlist 记录

	rr := httptest.NewRecorder()
	req := adminSessionReqForLocalAgentTypes(t, "tenant-admin-any-tenant")
	HandleAdminLocalAgentTypes(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("表空应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		LocalAgentTypes []LocalAgentTypeResponse `json:"local_agent_types"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if len(resp.LocalAgentTypes) != len(localAgentTypes) {
		t.Errorf("表空应全开返全量 %d 条，实际=%d", len(localAgentTypes), len(resp.LocalAgentTypes))
	}
}

// TestHandleAdminLocalAgentTypes_Whitelist_AdminTokenBypasses
// AdminToken 场景（isAdminTokenRequest=true）应绕过白名单，即使当前 tenant 未在白名单，
// 也应返回全量。守护 admin-token 运维路径不被白名单误伤。
func TestHandleAdminLocalAgentTypes_Whitelist_AdminTokenBypasses(t *testing.T) {
	setupSkillInstancesDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })

	// 白名单只放行 other-tenant，非 AdminToken 场景应被拦下
	if err := model.DB(context.Background()).Create(&model.FeatureAllowlist{
		Type: model.FeatureAllowlistTypeLocalAgent, Identifier: "other-tenant",
	}).Error; err != nil {
		t.Fatalf("create allowlist: %v", err)
	}

	rr := httptest.NewRecorder()
	HandleAdminLocalAgentTypes(rr, adminJSONGet("/admin/local-agent-types"))
	if rr.Code != http.StatusOK {
		t.Fatalf("AdminToken 应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		LocalAgentTypes []LocalAgentTypeResponse `json:"local_agent_types"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if len(resp.LocalAgentTypes) != len(localAgentTypes) {
		t.Errorf("AdminToken 应绕过白名单返全量 %d 条，实际=%d",
			len(localAgentTypes), len(resp.LocalAgentTypes))
	}
}
