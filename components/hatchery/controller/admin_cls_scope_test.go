package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ─── 测试辅助 ─────────────────────────────────────────────────────────

// setupCLSScopeTestEnv 初始化 CLS scope handler 测试所需的内存数据库和全局状态。
func setupCLSScopeTestEnv(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("数据库初始化失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.SiteConfig{},
		&model.GroupConfigBinding{},
		&model.GroupClosure{},
		&model.UserGroupMember{},
		&model.UserGroup{},
		&model.Instance{},
		&model.User{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	restore := model.UseDBForTest(db)

	origToken := AdminToken
	AdminToken = "test-admin-token"

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	t.Cleanup(func() {
		restore()
		AdminToken = origToken
		Store = origStore
	})
}

// clsScopeAdminReq 创建带 admin Bearer Token 的 JSON 请求。
func clsScopeAdminReq(method, path, body string) (*http.Request, *httptest.ResponseRecorder) {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req, httptest.NewRecorder()
}

// parseCLSScopeResp 解析 JSON 响应。
func parseCLSScopeResp(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("响应 JSON 解析失败: %v, body=%s", err, w.Body.String())
	}
	return result
}

// ─── HandleAdminGetCLSScope 测试 ─────────────────────────────────────

func TestHandleAdminGetCLSScope_Unauthorized(t *testing.T) {
	setupCLSScopeTestEnv(t)

	req := httptest.NewRequest("GET", "/admin/cls/scope", nil)
	req.Header.Set("Accept", "application/json")
	// 不设置 Authorization header
	w := httptest.NewRecorder()

	HandleAdminGetCLSScope(w, req)

	if w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Errorf("未授权请求期望 401/403，实际 %d", w.Code)
	}
}

func TestHandleAdminGetCLSScope_CLSDisabled_EmptyScope(t *testing.T) {
	setupCLSScopeTestEnv(t)

	// SiteConfig CLSEnabled = 0 (默认)
	model.DB(context.Background()).Create(&model.SiteConfig{})

	req, w := clsScopeAdminReq("GET", "/admin/cls/scope", "")
	HandleAdminGetCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSScopeResp(t, w)
	if result["scope_type"] != "off" {
		t.Errorf("CLS 未开启时 scope_type 应为 off，实际 %v", result["scope_type"])
	}
	if result["cls_enabled"] != false {
		t.Errorf("CLS 未开启时 cls_enabled 应为 false，实际 %v", result["cls_enabled"])
	}
}

func TestHandleAdminGetCLSScope_CLSEnabled_NoScope(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})

	req, w := clsScopeAdminReq("GET", "/admin/cls/scope", "")
	HandleAdminGetCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSScopeResp(t, w)
	if result["scope_type"] != "all" {
		t.Errorf("CLS 开启但无 scope 时 scope_type 应为 all，实际 %v", result["scope_type"])
	}
	if result["cls_enabled"] != true {
		t.Errorf("CLS 开启时 cls_enabled 应为 true，实际 %v", result["cls_enabled"])
	}
}

func TestHandleAdminGetCLSScope_WithGroupScope(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "研发组", FullPath: "根组/研发组", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "测试组", FullPath: "根组/测试组", Source: "manual"})

	// 获取创建后的 group IDs
	var groups []model.UserGroup
	model.DB(context.Background()).Find(&groups)
	if len(groups) < 2 {
		t.Fatalf("expected at least 2 groups, got %d", len(groups))
	}

	model.SetCLSCollectScope(context.Background(), []uint{groups[0].ID, groups[1].ID})

	req, w := clsScopeAdminReq("GET", "/admin/cls/scope", "")
	HandleAdminGetCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSScopeResp(t, w)
	if result["scope_type"] != "group" {
		t.Errorf("有 scope 时 scope_type 应为 group，实际 %v", result["scope_type"])
	}
	groupIDs, ok := result["group_ids"].([]interface{})
	if !ok {
		t.Fatalf("group_ids 应为数组，实际 %T", result["group_ids"])
	}
	if len(groupIDs) != 2 {
		t.Errorf("期望 2 个 group_ids，实际 %d", len(groupIDs))
	}
}

// ─── HandleAdminUpdateCLSScope 测试 ─────────────────────────────────────

func TestHandleAdminUpdateCLSScope_CLSNotEnabled(t *testing.T) {
	setupCLSScopeTestEnv(t)

	// CLS 未开启
	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 0})

	req, w := clsScopeAdminReq("POST", "/admin/cls/scope", `{"group_ids": [1]}`)
	HandleAdminUpdateCLSScope(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CLS 未开启时期望 400，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleAdminUpdateCLSScope_InvalidJSON(t *testing.T) {
	setupCLSScopeTestEnv(t)
	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})

	req, w := clsScopeAdminReq("POST", "/admin/cls/scope", `{invalid json`)
	HandleAdminUpdateCLSScope(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("无效 JSON 期望 400，实际 %d", w.Code)
	}
}

func TestHandleAdminUpdateCLSScope_InvalidGroupIDs(t *testing.T) {
	setupCLSScopeTestEnv(t)
	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})

	// 引用不存在的 group ID
	req, w := clsScopeAdminReq("POST", "/admin/cls/scope", `{"scope_type": "group", "group_ids": [9999]}`)
	HandleAdminUpdateCLSScope(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("不存在的 group ID 期望 400，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleAdminUpdateCLSScope_TooManyGroups(t *testing.T) {
	setupCLSScopeTestEnv(t)
	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})

	// 构造超过 maxScopeGroupIDs 的请求
	ids := make([]uint, maxScopeGroupIDs+1)
	for i := range ids {
		ids[i] = uint(i + 1)
	}
	bodyBytes, _ := json.Marshal(map[string]interface{}{"scope_type": "group", "group_ids": ids})
	req, w := clsScopeAdminReq("POST", "/admin/cls/scope", string(bodyBytes))
	HandleAdminUpdateCLSScope(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("超过分组数量限制期望 400，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleAdminUpdateCLSScope_EmptyToGroup(t *testing.T) {
	setupCLSScopeTestEnv(t)
	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "组A", FullPath: "组A", Source: "manual"})

	var group model.UserGroup
	model.DB(context.Background()).First(&group)

	body := `{"scope_type": "group", "group_ids": [` + uintToStr(group.ID) + `]}`
	req, w := clsScopeAdminReq("POST", "/admin/cls/scope", body)
	HandleAdminUpdateCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSScopeResp(t, w)
	if result["ok"] != true {
		t.Errorf("期望 ok=true，实际 %v", result["ok"])
	}

	// 验证 scope 已更新
	ids, _ := model.GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 1 || ids[0] != group.ID {
		t.Errorf("scope 未正确更新，got %v", ids)
	}
}

func TestHandleAdminUpdateCLSScope_GroupToEmpty(t *testing.T) {
	setupCLSScopeTestEnv(t)
	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "组A", FullPath: "组A", Source: "manual"})

	var group model.UserGroup
	model.DB(context.Background()).First(&group)
	model.SetCLSCollectScope(context.Background(), []uint{group.ID})

	req, w := clsScopeAdminReq("POST", "/admin/cls/scope", `{"scope_type": "group", "group_ids": []}`)
	HandleAdminUpdateCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSScopeResp(t, w)
	removedGroups, ok := result["removed_groups"].([]interface{})
	if !ok {
		t.Fatalf("removed_groups 应为数组")
	}
	if len(removedGroups) != 1 {
		t.Errorf("期望 1 个 removed_group，实际 %d", len(removedGroups))
	}

	// 验证 scope 已清空
	ids, _ := model.GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 0 {
		t.Errorf("scope 应已清空，got %v", ids)
	}
}

func TestHandleAdminUpdateCLSScope_DiffAddAndRemove(t *testing.T) {
	setupCLSScopeTestEnv(t)
	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "组A", FullPath: "组A", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "组B", FullPath: "组B", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "组C", FullPath: "组C", Source: "manual"})

	var groups []model.UserGroup
	model.DB(context.Background()).Find(&groups)
	if len(groups) < 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}

	// 初始 scope = [A, B]
	model.SetCLSCollectScope(context.Background(), []uint{groups[0].ID, groups[1].ID})

	// 更新为 [B, C]
	body := `{"scope_type": "group", "group_ids": [` + uintToStr(groups[1].ID) + `,` + uintToStr(groups[2].ID) + `]}`
	req, w := clsScopeAdminReq("POST", "/admin/cls/scope", body)
	HandleAdminUpdateCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSScopeResp(t, w)

	// A 被移除, C 被新增
	addedGroups, _ := result["added_groups"].([]interface{})
	removedGroups, _ := result["removed_groups"].([]interface{})

	if len(addedGroups) != 1 {
		t.Errorf("期望 1 个 added_group，实际 %d: %v", len(addedGroups), addedGroups)
	}
	if len(removedGroups) != 1 {
		t.Errorf("期望 1 个 removed_group，实际 %d: %v", len(removedGroups), removedGroups)
	}
}

func TestHandleAdminUpdateCLSScope_MarkInstances(t *testing.T) {
	setupCLSScopeTestEnv(t)
	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "组A", FullPath: "组A", Source: "manual"})

	var group model.UserGroup
	model.DB(context.Background()).First(&group)

	// closure: group -> 自身
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})

	// 用户在 group 中
	user := model.User{Username: "testuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: group.ID, UserID: user.ID})

	// 用户有两个实例（通过 GroupID 关联分组）：一个已安装（活跃状态不应被覆盖），一个已跳过（应被标记为待安装）
	model.DB(context.Background()).Create(&model.Instance{
		InstanceId:     "ins-test-001",
		UserID:         user.ID,
		GroupID:        group.ID,
		CLSAgentStatus: model.CLSAgentInstalled,
	})
	model.DB(context.Background()).Create(&model.Instance{
		InstanceId:     "ins-test-002",
		UserID:         user.ID,
		GroupID:        group.ID,
		CLSAgentStatus: model.CLSAgentSkipped,
	})

	// 设置 scope 到包含该分组
	body := `{"scope_type": "group", "group_ids": [` + uintToStr(group.ID) + `]}`
	req, w := clsScopeAdminReq("POST", "/admin/cls/scope", body)
	HandleAdminUpdateCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}

	// 验证已安装的实例状态不被覆盖（markInstancesCLSStatusSafe 跳过活跃状态）
	var inst1 model.Instance
	model.DB(context.Background()).Where("instance_id = ?", "ins-test-001").First(&inst1)
	if inst1.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("已安装实例不应被覆盖，期望 %d (已安装)，实际 %d", model.CLSAgentInstalled, inst1.CLSAgentStatus)
	}

	// 验证已跳过的实例被标记为待安装
	var inst2 model.Instance
	model.DB(context.Background()).Where("instance_id = ?", "ins-test-002").First(&inst2)
	if inst2.CLSAgentStatus != model.CLSAgentNotInstalled {
		t.Errorf("已跳过实例应被标记待安装，期望 %d (待安装)，实际 %d", model.CLSAgentNotInstalled, inst2.CLSAgentStatus)
	}
}

// ─── validateGroupIDs 测试 ─────────────────────────────────────

func TestValidateGroupIDs_Empty(t *testing.T) {
	setupCLSScopeTestEnv(t)

	if err := validateGroupIDs(context.Background(), nil); err != nil {
		t.Errorf("空列表应返回 nil，got %v", err)
	}
}

func TestValidateGroupIDs_AllExist(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.UserGroup{Name: "A", FullPath: "A", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "B", FullPath: "B", Source: "manual"})

	var groups []model.UserGroup
	model.DB(context.Background()).Find(&groups)

	ids := make([]uint, len(groups))
	for i, g := range groups {
		ids[i] = g.ID
	}

	if err := validateGroupIDs(context.Background(), ids); err != nil {
		t.Errorf("所有 group 存在时应返回 nil，got %v", err)
	}
}

func TestValidateGroupIDs_SomeMissing(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.UserGroup{Name: "A", FullPath: "A", Source: "manual"})

	err := validateGroupIDs(context.Background(), []uint{1, 9999})
	if err == nil {
		t.Error("部分 group 不存在时应返回错误")
	}
}

func TestValidateGroupIDs_DuplicateIDs(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.UserGroup{Name: "A", FullPath: "A", Source: "manual"})

	var group model.UserGroup
	model.DB(context.Background()).First(&group)

	// 输入包含重复 ID，不应误判为缺失
	if err := validateGroupIDs(context.Background(), []uint{group.ID, group.ID, group.ID}); err != nil {
		t.Errorf("重复 ID 不应导致校验失败，got %v", err)
	}
}

// ─── expandAndGetCVMIDs 测试 ─────────────────────────────────────

func TestExpandAndGetCVMIDs_Empty(t *testing.T) {
	setupCLSScopeTestEnv(t)

	ids, err := expandAndGetCVMIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ids != nil {
		t.Errorf("expected nil, got %v", ids)
	}
}

func TestExpandAndGetCVMIDs_WithData(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.UserGroup{Name: "组A", FullPath: "组A", Source: "manual"})
	var group model.UserGroup
	model.DB(context.Background()).First(&group)

	// closure
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})
	// member
	user := model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: group.ID, UserID: user.ID})
	// instance（通过 GroupID 关联分组）
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-expand-001", UserID: user.ID, GroupID: group.ID})

	ids, err := expandAndGetCVMIDs(context.Background(), []uint{group.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "ins-expand-001" {
		t.Errorf("expected [ins-expand-001], got %v", ids)
	}
}

// ─── getExclusiveRemovedInstances 测试 ─────────────────────────────────────

func TestGetExclusiveRemovedInstances_NoRemaining(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.UserGroup{Name: "组A", FullPath: "组A", Source: "manual"})
	var group model.UserGroup
	model.DB(context.Background()).First(&group)

	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})
	user := model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: group.ID, UserID: user.ID})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-excl-001", UserID: user.ID, GroupID: group.ID})

	ids, err := getExclusiveRemovedInstances(context.Background(), []uint{group.ID}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "ins-excl-001" {
		t.Errorf("无剩余 scope 时所有实例应为独占，got %v", ids)
	}
}

func TestGetExclusiveRemovedInstances_WithOverlap(t *testing.T) {
	setupCLSScopeTestEnv(t)

	// 创建两个组
	model.DB(context.Background()).Create(&model.UserGroup{Name: "组A", FullPath: "组A", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "组B", FullPath: "组B", Source: "manual"})
	var groups []model.UserGroup
	model.DB(context.Background()).Find(&groups)
	groupA, groupB := groups[0], groups[1]

	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: groupA.ID, DescendantID: groupA.ID, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: groupB.ID, DescendantID: groupB.ID, Depth: 0})

	// user1 同时在 A 和 B，实例归属 B（GroupID=B）
	user1 := model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user1)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: groupA.ID, UserID: user1.ID})
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: groupB.ID, UserID: user1.ID})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-shared", UserID: user1.ID, GroupID: groupB.ID})

	// user2 只在 A，实例归属 A（GroupID=A）
	user2 := model.User{Username: "u2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user2)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: groupA.ID, UserID: user2.ID})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-exclusive", UserID: user2.ID, GroupID: groupA.ID})

	// 移除 A，B 仍在 scope
	ids, err := getExclusiveRemovedInstances(context.Background(), []uint{groupA.ID}, []uint{groupB.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ins-shared 的 GroupID=B（仍在 scope 中），不应被移除; ins-exclusive 的 GroupID=A（已从 scope 移除），应被移除
	if len(ids) != 1 || ids[0] != "ins-exclusive" {
		t.Errorf("期望只有 ins-exclusive 为独占，got %v", ids)
	}
}

// ─── markInstancesCLSStatus 测试 ─────────────────────────────────────

func TestMarkInstancesCLSStatus_Empty(t *testing.T) {
	setupCLSScopeTestEnv(t)

	if err := markInstancesCLSStatus(context.Background(), nil, model.CLSAgentNotInstalled); err != nil {
		t.Errorf("空列表应返回 nil，got %v", err)
	}
}

func TestMarkInstancesCLSStatus_UpdatesCorrectly(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-mark-001", CLSAgentStatus: model.CLSAgentInstalled})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-mark-002", CLSAgentStatus: model.CLSAgentInstalled})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-mark-003", CLSAgentStatus: model.CLSAgentInstalled})

	if err := markInstancesCLSStatus(context.Background(), []string{"ins-mark-001", "ins-mark-002"}, model.CLSAgentNotInstalled); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var inst1, inst2, inst3 model.Instance
	model.DB(context.Background()).Where("instance_id = ?", "ins-mark-001").First(&inst1)
	model.DB(context.Background()).Where("instance_id = ?", "ins-mark-002").First(&inst2)
	model.DB(context.Background()).Where("instance_id = ?", "ins-mark-003").First(&inst3)

	if inst1.CLSAgentStatus != model.CLSAgentNotInstalled {
		t.Errorf("ins-mark-001 期望状态 %d，实际 %d", model.CLSAgentNotInstalled, inst1.CLSAgentStatus)
	}
	if inst2.CLSAgentStatus != model.CLSAgentNotInstalled {
		t.Errorf("ins-mark-002 期望状态 %d，实际 %d", model.CLSAgentNotInstalled, inst2.CLSAgentStatus)
	}
	if inst3.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("ins-mark-003 应不受影响，期望 %d，实际 %d", model.CLSAgentInstalled, inst3.CLSAgentStatus)
	}
}

// ─── markInstancesCLSStatusSafe 测试 ─────────────────────────────────────

func TestMarkInstancesCLSStatusSafe_Empty(t *testing.T) {
	setupCLSScopeTestEnv(t)

	if err := markInstancesCLSStatusSafe(context.Background(), nil, model.CLSAgentNotInstalled); err != nil {
		t.Errorf("空列表应返回 nil，got %v", err)
	}
}

func TestMarkInstancesCLSStatusSafe_SkipsActiveStates(t *testing.T) {
	setupCLSScopeTestEnv(t)

	// 创建各种状态的实例
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-safe-not", CLSAgentStatus: model.CLSAgentNotInstalled}) // 未安装 -> 应更新
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-safe-inst", CLSAgentStatus: model.CLSAgentInstalled})   // 已安装 -> 应跳过
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-safe-ing", CLSAgentStatus: model.CLSAgentInstalling})   // 安装中 -> 应跳过
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-safe-un", CLSAgentStatus: model.CLSAgentUninstalling})  // 卸载中 -> 应跳过
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-safe-skip", CLSAgentStatus: model.CLSAgentSkipped})     // 已跳过 -> 应更新

	allIDs := []string{"ins-safe-not", "ins-safe-inst", "ins-safe-ing", "ins-safe-un", "ins-safe-skip"}
	if err := markInstancesCLSStatusSafe(context.Background(), allIDs, model.CLSAgentNotInstalled); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	check := func(id string, expected int) {
		var inst model.Instance
		model.DB(context.Background()).Where("instance_id = ?", id).First(&inst)
		if inst.CLSAgentStatus != expected {
			t.Errorf("%s 期望状态 %d，实际 %d", id, expected, inst.CLSAgentStatus)
		}
	}

	check("ins-safe-not", model.CLSAgentNotInstalled)  // 未安装 -> 更新为未安装（无变化）
	check("ins-safe-inst", model.CLSAgentInstalled)    // 已安装 -> 跳过，保持不变
	check("ins-safe-ing", model.CLSAgentInstalling)    // 安装中 -> 跳过，保持不变
	check("ins-safe-un", model.CLSAgentUninstalling)   // 卸载中 -> 跳过，保持不变
	check("ins-safe-skip", model.CLSAgentNotInstalled) // 已跳过 -> 更新为未安装
}

// ─── inheritCLSScopeForNewInstance 测试 ─────────────────────────────

func TestInheritCLSScopeForNewInstance_CLSDisabled(t *testing.T) {
	setupCLSScopeTestEnv(t)

	// CLS 未开启
	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 0})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-inherit-off", GroupID: 1, CLSAgentStatus: model.CLSAgentSkipped})

	if err := inheritCLSScopeForNewInstance(context.Background(), 1, "ins-inherit-off"); err != nil {
		t.Fatalf("CLS 未开启时不应返回错误，got %v", err)
	}

	var inst model.Instance
	model.DB(context.Background()).Where("instance_id = ?", "ins-inherit-off").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentSkipped {
		t.Errorf("CLS 未开启时实例状态不应变更，期望 %d，实际 %d", model.CLSAgentSkipped, inst.CLSAgentStatus)
	}
}

func TestInheritCLSScopeForNewInstance_GroupNotInScope(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})

	// 创建分组和 scope
	model.DB(context.Background()).Create(&model.UserGroup{Name: "组A", FullPath: "组A", Source: "manual"})
	var group model.UserGroup
	model.DB(context.Background()).First(&group)
	model.SetCLSCollectScope(context.Background(), []uint{group.ID})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})

	// 实例的 group_id 不在 scope 中
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-inherit-out", GroupID: 999, CLSAgentStatus: model.CLSAgentSkipped})

	if err := inheritCLSScopeForNewInstance(context.Background(), 999, "ins-inherit-out"); err != nil {
		t.Fatalf("分组不在 scope 时不应返回错误，got %v", err)
	}

	var inst model.Instance
	model.DB(context.Background()).Where("instance_id = ?", "ins-inherit-out").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentSkipped {
		t.Errorf("分组不在 scope 中时实例状态不应变更，期望 %d，实际 %d", model.CLSAgentSkipped, inst.CLSAgentStatus)
	}
}

func TestInheritCLSScopeForNewInstance_GroupInScope(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})

	model.DB(context.Background()).Create(&model.UserGroup{Name: "组A", FullPath: "组A", Source: "manual"})
	var group model.UserGroup
	model.DB(context.Background()).First(&group)
	model.SetCLSCollectScope(context.Background(), []uint{group.ID})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-inherit-in", GroupID: group.ID, CLSAgentStatus: model.CLSAgentSkipped})

	if err := inheritCLSScopeForNewInstance(context.Background(), group.ID, "ins-inherit-in"); err != nil {
		t.Fatalf("分组在 scope 中时不应返回错误，got %v", err)
	}

	var inst model.Instance
	model.DB(context.Background()).Where("instance_id = ?", "ins-inherit-in").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentNotInstalled {
		t.Errorf("分组在 scope 中时实例应标记待安装，期望 %d，实际 %d", model.CLSAgentNotInstalled, inst.CLSAgentStatus)
	}
}

func TestInheritCLSScopeForNewInstance_EmptyScope_AllMode(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})
	// 不设置 scope = 全量模式

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-inherit-all", GroupID: 5, CLSAgentStatus: model.CLSAgentSkipped})

	if err := inheritCLSScopeForNewInstance(context.Background(), 5, "ins-inherit-all"); err != nil {
		t.Fatalf("全量模式下不应返回错误，got %v", err)
	}

	var inst model.Instance
	model.DB(context.Background()).Where("instance_id = ?", "ins-inherit-all").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentNotInstalled {
		t.Errorf("全量模式下任何实例应标记待安装，期望 %d，实际 %d", model.CLSAgentNotInstalled, inst.CLSAgentStatus)
	}
}

func TestInheritCLSScopeForNewInstance_GroupInChildScope(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})

	// 创建父子分组
	model.DB(context.Background()).Create(&model.UserGroup{Name: "父组", FullPath: "父组", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "子组", FullPath: "父组/子组", Source: "manual"})
	var groups []model.UserGroup
	model.DB(context.Background()).Find(&groups)
	parent, child := groups[0], groups[1]

	// scope 只配了父组
	model.SetCLSCollectScope(context.Background(), []uint{parent.ID})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: parent.ID, DescendantID: parent.ID, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: parent.ID, DescendantID: child.ID, Depth: 1})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: child.ID, DescendantID: child.ID, Depth: 0})

	// 实例属于子组
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-inherit-child", GroupID: child.ID, CLSAgentStatus: model.CLSAgentSkipped})

	if err := inheritCLSScopeForNewInstance(context.Background(), child.ID, "ins-inherit-child"); err != nil {
		t.Fatalf("子组实例继承时不应返回错误，got %v", err)
	}

	var inst model.Instance
	model.DB(context.Background()).Where("instance_id = ?", "ins-inherit-child").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentNotInstalled {
		t.Errorf("子组实例应继承父组 scope，期望 %d，实际 %d", model.CLSAgentNotInstalled, inst.CLSAgentStatus)
	}
}

// ─── HandleAdminGetCLSScope 响应增强测试 ─────────────────────────────

func TestHandleAdminGetCLSScope_InstanceCountFields(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})

	// 创建父子分组
	model.DB(context.Background()).Create(&model.UserGroup{Name: "研发组", FullPath: "根组/研发组", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "后端组", FullPath: "根组/研发组/后端组", Source: "manual"})
	var groups []model.UserGroup
	model.DB(context.Background()).Find(&groups)
	parent, child := groups[0], groups[1]

	// closure
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: parent.ID, DescendantID: parent.ID, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: parent.ID, DescendantID: child.ID, Depth: 1})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: child.ID, DescendantID: child.ID, Depth: 0})

	// 用户和实例（通过 GroupID 关联分组）
	user1 := model.User{Username: "dev1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user1)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: parent.ID, UserID: user1.ID})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-dev1-001", UserID: user1.ID, GroupID: parent.ID})

	user2 := model.User{Username: "dev2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user2)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: child.ID, UserID: user2.ID})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-dev2-001", UserID: user2.ID, GroupID: child.ID})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-dev2-002", UserID: user2.ID, GroupID: child.ID})

	// scope 只配父组
	model.SetCLSCollectScope(context.Background(), []uint{parent.ID})

	req, w := clsScopeAdminReq("GET", "/admin/cls/scope", "")
	HandleAdminGetCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSScopeResp(t, w)

	// 验证 total_instance_count
	totalCount, ok := result["total_instance_count"].(float64)
	if !ok {
		t.Fatalf("total_instance_count 应为数字，实际 %T: %v", result["total_instance_count"], result["total_instance_count"])
	}
	if int(totalCount) != 3 {
		t.Errorf("total_instance_count 期望 3，实际 %d", int(totalCount))
	}

	// 验证 groups 中有 instance_count 和 descendant_count
	groupInfos, ok := result["groups"].([]interface{})
	if !ok || len(groupInfos) != 1 {
		t.Fatalf("groups 应为长度 1 的数组，实际 %v", result["groups"])
	}
	groupInfo := groupInfos[0].(map[string]interface{})

	instanceCount, ok := groupInfo["instance_count"].(float64)
	if !ok {
		t.Fatalf("instance_count 应为数字，实际 %T", groupInfo["instance_count"])
	}
	if int(instanceCount) != 3 {
		t.Errorf("instance_count 期望 3（父组+子组），实际 %d", int(instanceCount))
	}

	descCount, ok := groupInfo["descendant_count"].(float64)
	if !ok {
		t.Fatalf("descendant_count 应为数字，实际 %T", groupInfo["descendant_count"])
	}
	if int(descCount) != 1 {
		t.Errorf("descendant_count 期望 1（子组），实际 %d", int(descCount))
	}
}

func TestHandleAdminGetCLSScope_MultipleGroupsInstanceCount(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})

	model.DB(context.Background()).Create(&model.UserGroup{Name: "组A", FullPath: "组A", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "组B", FullPath: "组B", Source: "manual"})
	var groups []model.UserGroup
	model.DB(context.Background()).Find(&groups)
	groupA, groupB := groups[0], groups[1]

	// closure（各自独立，无父子关系）
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: groupA.ID, DescendantID: groupA.ID, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: groupB.ID, DescendantID: groupB.ID, Depth: 0})

	// 用户1 在组A（实例通过 GroupID 关联）
	user1 := model.User{Username: "ua", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user1)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: groupA.ID, UserID: user1.ID})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-a1", UserID: user1.ID, GroupID: groupA.ID})

	// 用户2 在组B（实例通过 GroupID 关联）
	user2 := model.User{Username: "ub", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user2)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: groupB.ID, UserID: user2.ID})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-b1", UserID: user2.ID, GroupID: groupB.ID})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-b2", UserID: user2.ID, GroupID: groupB.ID})

	model.SetCLSCollectScope(context.Background(), []uint{groupA.ID, groupB.ID})

	req, w := clsScopeAdminReq("GET", "/admin/cls/scope", "")
	HandleAdminGetCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSScopeResp(t, w)

	totalCount := int(result["total_instance_count"].(float64))
	if totalCount != 3 {
		t.Errorf("total_instance_count 期望 3，实际 %d", totalCount)
	}

	groupInfos := result["groups"].([]interface{})
	if len(groupInfos) != 2 {
		t.Fatalf("期望 2 个 group，实际 %d", len(groupInfos))
	}

	// 收集各组 instance_count
	var counts []int
	for _, gi := range groupInfos {
		g := gi.(map[string]interface{})
		counts = append(counts, int(g["instance_count"].(float64)))
	}
	// 1 + 2 = 3
	total := 0
	for _, c := range counts {
		total += c
	}
	if total != 3 {
		t.Errorf("各组 instance_count 之和期望 3，实际 %d", total)
	}
}

func TestHandleAdminGetCLSScope_NoInstancesGroup(t *testing.T) {
	setupCLSScopeTestEnv(t)

	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "空组", FullPath: "空组", Source: "manual"})
	var group model.UserGroup
	model.DB(context.Background()).First(&group)

	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})
	model.SetCLSCollectScope(context.Background(), []uint{group.ID})

	req, w := clsScopeAdminReq("GET", "/admin/cls/scope", "")
	HandleAdminGetCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSScopeResp(t, w)

	totalCount := int(result["total_instance_count"].(float64))
	if totalCount != 0 {
		t.Errorf("空组 total_instance_count 期望 0，实际 %d", totalCount)
	}

	groupInfos := result["groups"].([]interface{})
	groupInfo := groupInfos[0].(map[string]interface{})
	instCount := int(groupInfo["instance_count"].(float64))
	if instCount != 0 {
		t.Errorf("空组 instance_count 期望 0，实际 %d", instCount)
	}
}

func TestHandleAdminUpdateCLSScope_InvalidScopeType(t *testing.T) {
	setupCLSScopeTestEnv(t)
	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1})

	req, w := clsScopeAdminReq("POST", "/admin/cls/scope", `{"scope_type": "invalid", "group_ids": []}`)
	HandleAdminUpdateCLSScope(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleAdminUpdateCLSScope_AllModeClearsGroups(t *testing.T) {
	setupCLSScopeTestEnv(t)
	model.DB(context.Background()).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "组B", FullPath: "组B", Source: "manual"})

	var group model.UserGroup
	model.DB(context.Background()).First(&group)
	model.SetCLSCollectScope(context.Background(), []uint{group.ID})

	// 切换到全量模式时 group_ids 应被忽略
	body := `{"scope_type": "all", "group_ids": [` + uintToStr(group.ID) + `]}`
	req, w := clsScopeAdminReq("POST", "/admin/cls/scope", body)
	HandleAdminUpdateCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSScopeResp(t, w)
	if result["ok"] != true {
		t.Errorf("期望 ok=true")
	}

	// scope 应被清空（all 模式，group_ids 被忽略为 nil → DiffAndSet 传空）
	ids, _ := model.GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 0 {
		t.Errorf("全量模式下 scope 应清空，got %v", ids)
	}
}

func TestHandleAdminUpdateCLSScope_RemovedInstanceCount(t *testing.T) {
	setupCLSScopeTestEnv(t)
	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})
	model.DB(ctx).Create(&model.UserGroup{Name: "组C", FullPath: "组C", Source: "manual"})
	model.DB(ctx).Create(&model.UserGroup{Name: "组D", FullPath: "组D", Source: "manual"})

	var groups []model.UserGroup
	model.DB(ctx).Find(&groups)
	if len(groups) < 2 {
		t.Fatalf("需要至少 2 个分组")
	}
	groupC, groupD := groups[0], groups[1]

	// 设置初始 scope 为 [C, D]
	model.SetCLSCollectScope(ctx, []uint{groupC.ID, groupD.ID})
	// closure
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: groupC.ID, DescendantID: groupC.ID, Depth: 0})
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: groupD.ID, DescendantID: groupD.ID, Depth: 0})
	// 实例属于组 D
	model.DB(ctx).Create(&model.Instance{InstanceId: "ins-rm1", UserID: 1, GroupID: groupD.ID, CLSAgentStatus: model.CLSAgentInstalled})
	model.DB(ctx).Create(&model.Instance{InstanceId: "ins-rm2", UserID: 2, GroupID: groupD.ID, CLSAgentStatus: model.CLSAgentInstalled})

	// 更新 scope 为只保留 C → 移除 D
	body := `{"scope_type": "group", "group_ids": [` + uintToStr(groupC.ID) + `]}`
	req, w := clsScopeAdminReq("POST", "/admin/cls/scope", body)
	HandleAdminUpdateCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSScopeResp(t, w)
	removedCount := int(result["removed_instance_count"].(float64))
	if removedCount != 2 {
		t.Errorf("期望移除 2 个实例，实际 %d", removedCount)
	}
}

// TestHandleAdminUpdateCLSScope_RemoveAllGroups 验证移除所有分组时（切换到全量模式），
// 被移除分组的独占实例数量被正确统计（覆盖 getExclusiveRemovedInstances 中 remainingGroupIDs 为空的路径）。
func TestHandleAdminUpdateCLSScope_RemoveAllGroups(t *testing.T) {
	setupCLSScopeTestEnv(t)
	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})
	model.DB(ctx).Create(&model.UserGroup{Name: "组E", FullPath: "组E", Source: "manual"})

	var grp model.UserGroup
	model.DB(ctx).Where("name = ?", "组E").First(&grp)

	// 设置初始 scope 为 [E]
	model.SetCLSCollectScope(ctx, []uint{grp.ID})
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: grp.ID, DescendantID: grp.ID, Depth: 0})
	// 组 E 有一个已安装实例
	model.DB(ctx).Create(&model.Instance{InstanceId: "ins-e1", UserID: 1, GroupID: grp.ID, CLSAgentStatus: model.CLSAgentInstalled})

	// 切换到全量模式（group_ids 为空），移除所有分组
	body := `{"scope_type": "all", "group_ids": []}`
	req, w := clsScopeAdminReq("POST", "/admin/cls/scope", body)
	HandleAdminUpdateCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSScopeResp(t, w)
	// 全量模式下 group_ids 被忽略，removed 应包含原来的分组 E
	removedCount := int(result["removed_instance_count"].(float64))
	if removedCount != 1 {
		t.Errorf("期望移除 1 个实例，实际 %d", removedCount)
	}
}

// TestInheritCLSScopeForNewInstance_InScope 验证新实例命中 CLS 采集范围时被标记为待安装。
func TestInheritCLSScopeForNewInstance_InScope(t *testing.T) {
	setupCLSScopeTestEnv(t)
	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})
	model.DB(ctx).Create(&model.UserGroup{Name: "组F", FullPath: "组F", Source: "manual"})

	var grp model.UserGroup
	model.DB(ctx).Where("name = ?", "组F").First(&grp)

	// scope 包含组 F
	model.SetCLSCollectScope(ctx, []uint{grp.ID})
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: grp.ID, DescendantID: grp.ID, Depth: 0})
	// 创建新实例
	model.DB(ctx).Create(&model.Instance{InstanceId: "ins-f1", UserID: 1, GroupID: grp.ID, CLSAgentStatus: model.CLSAgentSkipped})

	err := inheritCLSScopeForNewInstance(ctx, grp.ID, "ins-f1")
	if err != nil {
		t.Fatalf("命中 scope 时期望 nil error，实际 %v", err)
	}

	// 验证实例被标记为待安装（跳过活跃状态，CLSAgentSkipped 不是活跃状态，应被更新）
	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-f1").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentNotInstalled {
		t.Errorf("命中 scope 的实例应被标记为待安装，期望 %d，实际 %d", model.CLSAgentNotInstalled, inst.CLSAgentStatus)
	}
}

// TestHandleAdminGetCLSScope_WithAllInstallStats 验证 buildInstallStats 覆盖所有状态分支
// （installing/uninstalling/skipped），确保统计数据正确。
func TestHandleAdminGetCLSScope_WithAllInstallStats(t *testing.T) {
	setupCLSScopeTestEnv(t)
	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})
	model.DB(ctx).Create(&model.UserGroup{Name: "统计组", FullPath: "统计组", Source: "manual"})

	var grp model.UserGroup
	model.DB(ctx).Where("name = ?", "统计组").First(&grp)

	model.SetCLSCollectScope(ctx, []uint{grp.ID})
	// closure：分组包含自身
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: grp.ID, DescendantID: grp.ID, Depth: 0})

	// 创建覆盖所有状态的实例
	model.DB(ctx).Create(&model.Instance{InstanceId: "stat-ins-1", UserID: 1, GroupID: grp.ID, CLSAgentStatus: model.CLSAgentNotInstalled})
	model.DB(ctx).Create(&model.Instance{InstanceId: "stat-ins-2", UserID: 1, GroupID: grp.ID, CLSAgentStatus: model.CLSAgentInstalling})
	model.DB(ctx).Create(&model.Instance{InstanceId: "stat-ins-3", UserID: 1, GroupID: grp.ID, CLSAgentStatus: model.CLSAgentInstalled})
	model.DB(ctx).Create(&model.Instance{InstanceId: "stat-ins-4", UserID: 1, GroupID: grp.ID, CLSAgentStatus: model.CLSAgentUninstalling})
	model.DB(ctx).Create(&model.Instance{InstanceId: "stat-ins-5", UserID: 1, GroupID: grp.ID, CLSAgentStatus: model.CLSAgentSkipped})
	// 创建一个未知状态的实例，覆盖 buildInstallStats 中的 default 分支
	model.DB(ctx).Create(&model.Instance{InstanceId: "stat-ins-6", UserID: 1, GroupID: grp.ID, CLSAgentStatus: 99})

	req, w := clsScopeAdminReq("GET", "/admin/cls/scope", "")
	HandleAdminGetCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSScopeResp(t, w)
	if result["scope_type"] != "group" {
		t.Errorf("有 scope 时 scope_type 应为 group，实际 %v", result["scope_type"])
	}
	totalStats, ok := result["total_install_stats"].(map[string]interface{})
	if !ok {
		t.Fatalf("total_install_stats 应为 map，实际 %T", result["total_install_stats"])
	}
	// 验证各状态统计
	// 注意：状态为99（未知）的实例会被 default 分支归类为 not_installed，所以 not_installed = 2
	if int(totalStats["not_installed"].(float64)) != 2 {
		t.Errorf("not_installed 期望 2（含1个未知状态实例），实际 %v", totalStats["not_installed"])
	}
	if int(totalStats["installing"].(float64)) != 1 {
		t.Errorf("installing 期望 1，实际 %v", totalStats["installing"])
	}
	if int(totalStats["installed"].(float64)) != 1 {
		t.Errorf("installed 期望 1，实际 %v", totalStats["installed"])
	}
	if int(totalStats["uninstalling"].(float64)) != 1 {
		t.Errorf("uninstalling 期望 1，实际 %v", totalStats["uninstalling"])
	}
	if int(totalStats["skipped"].(float64)) != 1 {
		t.Errorf("skipped 期望 1，实际 %v", totalStats["skipped"])
	}
	if int(result["total_instance_count"].(float64)) != 6 {
		t.Errorf("total_instance_count 期望 6，实际 %v", result["total_instance_count"])
	}
}

// TestHandleAdminUpdateCLSScope_RemovedGroupWithInstalledInstance 验证移除分组时，
// 已安装实例被正确统计（覆盖 391-395 行的 slog.Info 多行参数路径）。
func TestHandleAdminUpdateCLSScope_RemovedGroupWithInstalledInstance(t *testing.T) {
	setupCLSScopeTestEnv(t)
	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})
	model.DB(ctx).Create(&model.UserGroup{Name: "移除组H", FullPath: "移除组H", Source: "manual"})
	model.DB(ctx).Create(&model.UserGroup{Name: "保留组I", FullPath: "保留组I", Source: "manual"})

	var grpH, grpI model.UserGroup
	model.DB(ctx).Where("name = ?", "移除组H").First(&grpH)
	model.DB(ctx).Where("name = ?", "保留组I").First(&grpI)

	// 初始 scope = [H, I]
	model.SetCLSCollectScope(ctx, []uint{grpH.ID, grpI.ID})
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: grpH.ID, DescendantID: grpH.ID, Depth: 0})
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: grpI.ID, DescendantID: grpI.ID, Depth: 0})

	// 组 H 有一个已安装实例（独占，不属于组 I）
	model.DB(ctx).Create(&model.Instance{InstanceId: "ins-h1", UserID: 1, GroupID: grpH.ID, CLSAgentStatus: model.CLSAgentInstalled})
	// 组 I 有一个实例
	model.DB(ctx).Create(&model.Instance{InstanceId: "ins-i1", UserID: 1, GroupID: grpI.ID, CLSAgentStatus: model.CLSAgentInstalled})

	// 更新 scope 为 [I]，移除 H
	body := `{"scope_type": "group", "group_ids": [` + uintToStr(grpI.ID) + `]}`
	req, w := clsScopeAdminReq("POST", "/admin/cls/scope", body)
	HandleAdminUpdateCLSScope(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSScopeResp(t, w)
	removedCount := int(result["removed_instance_count"].(float64))
	if removedCount != 1 {
		t.Errorf("期望移除 1 个独占实例，实际 %d", removedCount)
	}
}
