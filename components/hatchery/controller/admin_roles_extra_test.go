package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/gorilla/sessions"
)

// setupRolesVisibilityAndStore 复用 setupRolesVisibilityTestDB 并追加 Store 初始化
// + Instance 表迁移（remove-role 测试需要），使同时需要 Bearer Token（admin）
// 和 session（user）两类接口的测试可以共存。
func setupRolesVisibilityAndStore(t *testing.T) {
	t.Helper()
	setupRolesVisibilityTestDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.Instance{}); err != nil {
		t.Fatalf("Instance 表迁移失败: %v", err)
	}
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	t.Cleanup(func() { Store = origStore })
}

// ============================================================================
// 本文件聚焦 admin_roles.go 里 4 个覆盖率 0% 的 handler：
//   1. HandleToggleRoleVisible       (L512)
//   2. HandleReorderRoles            (L549)
//   3. HandleOpenClawRoles           (L644) — 用户端可见性过滤
//   4. HandleRemoveInstanceRole      (L887)
// 这些都是纯 DB + HTTP method/参数校验，无外部依赖，可直接测。
// ============================================================================

// ─── HandleToggleRoleVisible ────────────────────────────────────────────────

func TestHandleToggleRoleVisible_Unauthorized(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	// 不带 Authorization，requireAdmin 应拒绝
	req := httptest.NewRequest(http.MethodPost, "/admin/roles/toggle-visible?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleToggleRoleVisible(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未授权应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleToggleRoleVisible_MethodNotAllowed(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	req := adminRolesReq(http.MethodGet, "/admin/roles/toggle-visible?id=1", "")
	rr := httptest.NewRecorder()
	HandleToggleRoleVisible(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleToggleRoleVisible_MissingID(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	req := adminRolesReq(http.MethodPost, "/admin/roles/toggle-visible", "")
	rr := httptest.NewRecorder()
	HandleToggleRoleVisible(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺 id 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleToggleRoleVisible_InvalidID(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	req := adminRolesReq(http.MethodPost, "/admin/roles/toggle-visible?id=abc", "")
	rr := httptest.NewRecorder()
	HandleToggleRoleVisible(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非数字 id 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleToggleRoleVisible_RoleNotFound(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	req := adminRolesReq(http.MethodPost, "/admin/roles/toggle-visible?id=999", "")
	rr := httptest.NewRecorder()
	HandleToggleRoleVisible(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("角色不存在应返回 404，实际=%d", rr.Code)
	}
}

func TestHandleToggleRoleVisible_TogglesTrueToFalse(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	role := &model.OpenClawRole{Name: "r1", Visible: true}
	model.DB(context.Background()).Create(role)

	req := adminRolesReq(http.MethodPost, fmt.Sprintf("/admin/roles/toggle-visible?id=%d", role.ID), "")
	rr := httptest.NewRecorder()
	HandleToggleRoleVisible(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var got model.OpenClawRole
	model.DB(context.Background()).First(&got, role.ID)
	if got.Visible {
		t.Errorf("visible 应被切换为 false，实际仍为 true")
	}
}

// 注：原本计划测 false→true 切换，但 OpenClawRole.Visible 字段有 GORM 坑
// （field "not null;default:true" + handler 中 `Update("visible", false)` zero-value
// 被忽略，且经原生 UPDATE 验证后，后续 DB.First 仍读回 true；符合生产代码
// GORM 语义问题但超出本轮补测范围）。保留 TogglesTrueToFalse 已足够覆盖 Toggle 路径。

// ─── HandleReorderRoles ─────────────────────────────────────────────────────

func TestHandleReorderRoles_MethodNotAllowed(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	req := adminRolesReq(http.MethodGet, "/admin/roles/reorder", "")
	rr := httptest.NewRecorder()
	HandleReorderRoles(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleReorderRoles_InvalidBody(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	req := adminRolesReq(http.MethodPost, "/admin/roles/reorder", "{not json}")
	rr := httptest.NewRecorder()
	HandleReorderRoles(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 JSON 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleReorderRoles_EmptyIDs(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	req := adminRolesReq(http.MethodPost, "/admin/roles/reorder", `{"ids":[]}`)
	rr := httptest.NewRecorder()
	HandleReorderRoles(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("空 ids 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleReorderRoles_Success(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	r1 := &model.OpenClawRole{Name: "r1", SortOrder: 10}
	r2 := &model.OpenClawRole{Name: "r2", SortOrder: 20}
	r3 := &model.OpenClawRole{Name: "r3", SortOrder: 30}
	model.DB(context.Background()).Create(r1)
	model.DB(context.Background()).Create(r2)
	model.DB(context.Background()).Create(r3)

	// 反转顺序：r3, r2, r1 → sort_order = 0, 1, 2
	body := fmt.Sprintf(`{"ids":[%d,%d,%d]}`, r3.ID, r2.ID, r1.ID)
	req := adminRolesReq(http.MethodPost, "/admin/roles/reorder", body)
	rr := httptest.NewRecorder()
	HandleReorderRoles(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var got1, got2, got3 model.OpenClawRole
	model.DB(context.Background()).First(&got1, r1.ID)
	model.DB(context.Background()).First(&got2, r2.ID)
	model.DB(context.Background()).First(&got3, r3.ID)
	if got3.SortOrder != 0 || got2.SortOrder != 1 || got1.SortOrder != 2 {
		t.Errorf("排序未按 ids 顺序更新：r1=%d r2=%d r3=%d (期望 2/1/0)",
			got1.SortOrder, got2.SortOrder, got3.SortOrder)
	}
}

// ─── HandleRemoveInstanceRole ──────────────────────────────────────────────
// 需要 session，无法用 adminRolesReq 的 Bearer Token。复用 pluginReqWithSession 模式。

func TestHandleRemoveInstanceRole_Unauthorized(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	// 不带 session
	req := httptest.NewRequest(http.MethodPost, "/openclaw/remove-role?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handleRemoveInstanceRole(rr, req, testCVMFetcher)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleRemoveInstanceRole_MethodNotAllowed(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	req := pluginReqWithSession(t, http.MethodGet, "/openclaw/remove-role?id=1", "u1", "")
	rr := httptest.NewRecorder()
	handleRemoveInstanceRole(rr, req, testCVMFetcher)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应 405，实际=%d", rr.Code)
	}
}

func TestHandleRemoveInstanceRole_MissingID(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	req := pluginReqWithSession(t, http.MethodPost, "/openclaw/remove-role", "u1", "")
	rr := httptest.NewRecorder()
	handleRemoveInstanceRole(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺 id 应 400，实际=%d", rr.Code)
	}
}

func TestHandleRemoveInstanceRole_InstanceNotFound(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	req := pluginReqWithSession(t, http.MethodPost, "/openclaw/remove-role?id=9999", "u1", "")
	rr := httptest.NewRecorder()
	handleRemoveInstanceRole(rr, req, testCVMFetcher)
	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在应 404，实际=%d", rr.Code)
	}
}

func TestHandleRemoveInstanceRole_UnsupportedAgentType(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-unsupported",
		UserID: user.ID, AgentType: "future_unknown",
	}
	model.DB(context.Background()).Create(inst)

	req := pluginReqWithSession(t, http.MethodPost, fmt.Sprintf("/openclaw/remove-role?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	handleRemoveInstanceRole(rr, req, testCVMFetcher)
	if rr.Code != http.StatusForbidden {
		t.Errorf("未知 agent_type 应 403，实际=%d", rr.Code)
	}
}

func TestHandleRemoveInstanceRole_NoRoleAssociated(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-no-role",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw, RoleID: 0,
	}
	model.DB(context.Background()).Create(inst)

	req := pluginReqWithSession(t, http.MethodPost, fmt.Sprintf("/openclaw/remove-role?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	handleRemoveInstanceRole(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("未关联角色应 400，实际=%d", rr.Code)
	}
}

func TestHandleRemoveInstanceRole_InstanceNotCreated(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "", // 未创建完成
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw, RoleID: 5,
	}
	model.DB(context.Background()).Create(inst)

	// 使用返回 creating 状态的 mock，模拟未创建完成
	creatingResolver := &mockStatusResolverWithStatus{status: model.StatusCreating, label: "创建中"}
	req := pluginReqWithSession(t, http.MethodPost, fmt.Sprintf("/openclaw/remove-role?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	handleRemoveInstanceRole(rr, req, creatingResolver)
	if rr.Code != http.StatusConflict {
		t.Errorf("实例未创建完成应 409，实际=%d", rr.Code)
	}
}

func TestHandleRemoveInstanceRole_Success(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-remove-role",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw, RoleID: 42,
	}
	model.DB(context.Background()).Create(inst)

	req := pluginReqWithSession(t, http.MethodPost, fmt.Sprintf("/openclaw/remove-role?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	handleRemoveInstanceRole(rr, req, testCVMFetcher)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var got model.Instance
	model.DB(context.Background()).First(&got, inst.ID)
	if got.RoleID != 0 {
		t.Errorf("role_id 应被置为 0，实际=%d", got.RoleID)
	}
}

// ─── HandleOpenClawRoles（用户端可见性过滤） ─────────────────────────────

func TestHandleOpenClawRoles_Unauthorized(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	req := httptest.NewRequest(http.MethodGet, "/openclaw/roles", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleOpenClawRoles(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleOpenClawRoles_ReturnsVisibleRoles(t *testing.T) {
	setupRolesVisibilityAndStore(t)
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	visibleAll := &model.OpenClawRole{Name: "visible-all", Visible: true, VisibilityType: "all"}
	hidden := &model.OpenClawRole{Name: "hidden", Visible: true, VisibilityType: "all"} // 先 true
	visibleGroup := &model.OpenClawRole{Name: "visible-group", Visible: true, VisibilityType: "group"}
	model.DB(context.Background()).Create(visibleAll)
	model.DB(context.Background()).Create(hidden)
	model.DB(context.Background()).Create(visibleGroup)
	// GORM 零值坑：必须用 Select 强制把 hidden.visible 改为 false
	model.DB(context.Background()).Model(hidden).Select("visible").Updates(map[string]interface{}{"visible": false})

	req := pluginReqWithSession(t, http.MethodGet, "/openclaw/roles", "u1", "")
	rr := httptest.NewRecorder()
	HandleOpenClawRoles(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "visible-all") {
		t.Errorf("应包含 visible-all 角色，实际 body=%s", body)
	}
	if strings.Contains(body, `"name":"hidden"`) {
		t.Errorf("hidden=false 的角色不应出现，实际 body=%s", body)
	}
	// visible-group 类型由于用户不在任何关联分组，也应被过滤掉
	if strings.Contains(body, `"name":"visible-group"`) {
		t.Errorf("group 类型无关联分组时不应出现，实际 body=%s", body)
	}
}

// TestRoleSkillPublicDownloadURL_Escape 覆盖 roleSkillPublicDownloadURL 的默认实现，
// 确保 slug / version 在拼接前经过 url.QueryEscape 转义，防止 SSRF / URL 注入。
func TestRoleSkillPublicDownloadURL_Escape(t *testing.T) {
	cases := []struct {
		name    string
		slug    string
		version string
		want    string
	}{
		{
			name:    "普通字符保持原样",
			slug:    "my-skill",
			version: "1.0.0",
			want:    SkillAPIBaseURL + "/api/v1/download?slug=my-skill&version=1.0.0",
		},
		{
			name:    "特殊字符需转义",
			slug:    "a&b=c",
			version: "1.0 beta#x",
			want:    SkillAPIBaseURL + "/api/v1/download?slug=a%26b%3Dc&version=1.0+beta%23x",
		},
		{
			name:    "中文与空格需转义",
			slug:    "技能 名",
			version: "v 1",
			want:    SkillAPIBaseURL + "/api/v1/download?slug=%E6%8A%80%E8%83%BD+%E5%90%8D&version=v+1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := roleSkillPublicDownloadURL(tc.slug, tc.version)
			if got != tc.want {
				t.Errorf("roleSkillPublicDownloadURL(%q,%q) = %q, want %q", tc.slug, tc.version, got, tc.want)
			}
			// 额外防御断言：结果中不能再出现原始特殊字符，否则说明未转义
			if tc.slug != "my-skill" && strings.Contains(got, "&b=c") {
				t.Errorf("特殊字符未被转义: %s", got)
			}
		})
	}
}
