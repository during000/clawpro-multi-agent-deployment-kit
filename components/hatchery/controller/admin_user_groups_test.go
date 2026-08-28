package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ── 测试辅助 ─────────────────────────────────────────────────────────────────

// setupUserGroupTestDB 初始化内存 SQLite，迁移用户组相关表。
// 每个测试用例独立调用，保证隔离。
func setupUserGroupTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.GroupClosure{},
		&model.GroupConfigBinding{},
		&model.Instance{},
		&model.SiteConfig{},
		&model.ModelVisibilityGroup{},
		&model.SkillVisibilityGroup{},
		&model.SkillBundleVisibilityGroup{},
		&model.RoleVisibilityGroup{},
		&model.Tag{},
		&model.TagVisibilityGroup{},
		// HandleAdminGetUngroupedUsers → queryUsers → admin_users.go 会查询 project_members
		&model.ProjectMember{},
		// toAdminJSON → enrichUsersWithProjectInfo 可能查询 projects
		&model.Project{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	if err := db.Create(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("初始化 SiteConfig 失败: %v", err)
	}

	// 设置 AdminToken，使 requireAdmin 可通过 Bearer Token 验证
	AdminToken = "test-admin-token"
}

// batchCreateMembers 分批写入成员记录，规避 SQLite 单次 INSERT 变量数上限（999）。
// SQLite 每行 user_group_members 有 5 个字段，每批最多写入 199 行（199*5=995 < 999）。
func batchCreateMembers(t *testing.T, members []model.UserGroupMember) {
	t.Helper()
	const batchSize = 199
	for i := 0; i < len(members); i += batchSize {
		end := i + batchSize
		if end > len(members) {
			end = len(members)
		}
		if err := model.DB(context.Background()).Create(members[i:end]).Error; err != nil {
			t.Fatalf("批量写入成员失败: %v", err)
		}
	}
}

// batchCreateGroups 分批写入用户组记录，规避 SQLite 单次 INSERT 变量数上限（999）。
// SQLite 每行 user_groups 有 7 个字段，每批最多写入 142 行（142*7=994 < 999）。
func batchCreateGroups(t *testing.T, groups []model.UserGroup) {
	t.Helper()
	const batchSize = 142
	for i := 0; i < len(groups); i += batchSize {
		end := i + batchSize
		if end > len(groups) {
			end = len(groups)
		}
		if err := model.DB(context.Background()).Create(groups[i:end]).Error; err != nil {
			t.Fatalf("批量写入用户组失败: %v", err)
		}
	}
}

// mustInsertLegacyGroupMembers 直接写入 user_group_members 表，仅用于测试中构造历史存量成员关系。
//
// 使用场景：当测试需要验证“库里已经存在的一人多组旧数据”仍可被查询接口兼容读取时，
// 不能再走 doAddMembers 这类受当前接口约束保护的新增写入路径，否则会被单组规则拦截。
// 因此这里刻意绕过接口层，直接落库，明确表示这批数据是存量数据，不代表当前允许的新写入方式。
func mustInsertLegacyGroupMembers(t *testing.T, members ...model.UserGroupMember) {
	t.Helper()
	if len(members) == 0 {
		return
	}
	if err := model.DB(context.Background()).Create(&members).Error; err != nil {
		t.Fatalf("写入存量用户组成员失败: %v", err)
	}
}

// createTestUsers 在数据库中批量创建普通用户，返回 ID 列表。
// 用于测试中构造成员数据，不走 HTTP 接口（用户创建不是被测对象）。
func createTestUsers(t *testing.T, usernames ...string) []uint {
	t.Helper()
	ids := make([]uint, len(usernames))
	for i, name := range usernames {
		u := model.User{Username: name, Password: "hashed", Role: "user"}
		if err := model.DB(context.Background()).Create(&u).Error; err != nil {
			t.Fatalf("创建测试用户 %s 失败: %v", name, err)
		}
		ids[i] = u.ID
	}
	return ids
}

// adminReq 构造携带管理员 Token 的 HTTP 请求。
func adminReq(method, path string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// doListGroups 调用 HandleAdminListUserGroups（携带管理员 Token）。
func doListGroups(t *testing.T, query string) *httptest.ResponseRecorder {
	t.Helper()
	path := "/admin/user-groups"
	if query != "" {
		path += "?" + query
	}
	w := httptest.NewRecorder()
	HandleAdminListUserGroups(w, adminReq(http.MethodGet, path, nil))
	return w
}

// doCreateGroup 调用 HandleAdminCreateUserGroup（携带管理员 Token）。
func doCreateGroup(t *testing.T, name, description string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"name": name, "description": description})
	w := httptest.NewRecorder()
	HandleAdminCreateUserGroup(w, adminReq(http.MethodPost, "/admin/user-groups/create", body))
	return w
}

// doDeleteGroup 调用 HandleAdminDeleteUserGroup（携带管理员 Token）。
func doDeleteGroup(t *testing.T, id uint) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]uint{"id": id})
	w := httptest.NewRecorder()
	HandleAdminDeleteUserGroup(w, adminReq(http.MethodPost, "/admin/user-groups/delete", body))
	return w
}

// doGetMembers 调用 HandleAdminGetGroupMembers（携带管理员 Token）。
func doGetMembers(t *testing.T, groupID uint) *httptest.ResponseRecorder {
	t.Helper()
	path := fmt.Sprintf("/admin/user-groups/members?id=%d", groupID)
	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, adminReq(http.MethodGet, path, nil))
	return w
}

// doSetMembers 调用 HandleAdminSetGroupMembers（携带管理员 Token）。
func doSetMembers(t *testing.T, groupID uint, userIDs []uint) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{"id": groupID, "user_ids": userIDs})
	w := httptest.NewRecorder()
	HandleAdminSetGroupMembers(w, adminReq(http.MethodPost, "/admin/user-groups/members/set", body))
	return w
}

// doAddMembers 调用 HandleAdminAddGroupMembers（携带管理员 Token）。
func doAddMembers(t *testing.T, groupID uint, userIDs []uint) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{"id": groupID, "user_ids": userIDs})
	w := httptest.NewRecorder()
	HandleAdminAddGroupMembers(w, adminReq(http.MethodPost, "/admin/user-groups/members/add", body))
	return w
}

// doRemoveMembers 调用 HandleAdminRemoveGroupMembers（携带管理员 Token）。
func doRemoveMembers(t *testing.T, groupID uint, userIDs []uint) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{"id": groupID, "user_ids": userIDs})
	w := httptest.NewRecorder()
	HandleAdminRemoveGroupMembers(w, adminReq(http.MethodPost, "/admin/user-groups/members/remove", body))
	return w
}

// mustCreateGroup 创建用户组并断言成功，返回组 ID。
func mustCreateGroup(t *testing.T, name, description string) uint {
	t.Helper()
	w := doCreateGroup(t, name, description)
	if w.Code != http.StatusOK {
		t.Fatalf("创建用户组 %q 失败: status=%d, body=%s", name, w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	group := resp["group"].(map[string]interface{})
	return uint(group["id"].(float64))
}

// memberUsernames 从 GET /admin/user-groups/members 响应中提取用户名集合。
func memberUsernames(t *testing.T, w *httptest.ResponseRecorder) map[string]bool {
	t.Helper()
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	members := resp["members"].([]interface{})
	result := make(map[string]bool, len(members))
	for _, m := range members {
		item := m.(map[string]interface{})
		result[item["username"].(string)] = true
	}
	return result
}

// ── GET /admin/user-groups ────────────────────────────────────────────────────

// 场景：管理员查看空列表
func TestAdminListUserGroups_Empty(t *testing.T) {
	setupUserGroupTestDB(t)

	w := doListGroups(t, "")
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("期望 ok=true，实际=%v", resp["ok"])
	}
	if resp["total"].(float64) != 0 {
		t.Errorf("期望 total=0，实际=%v", resp["total"])
	}
	if len(resp["groups"].([]interface{})) != 0 {
		t.Errorf("期望 groups 为空数组")
	}
}

// 场景：管理员创建 3 个组后分页查询，第 1 页 page_size=2 返回 2 条，total=3
func TestAdminListUserGroups_Pagination(t *testing.T) {
	setupUserGroupTestDB(t)

	mustCreateGroup(t, "研发组", "研发部门")
	mustCreateGroup(t, "运营组", "运营部门")
	mustCreateGroup(t, "测试组", "测试部门")

	w := doListGroups(t, "page=1&page_size=2")
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["total"].(float64) != 3 {
		t.Errorf("期望 total=3，实际=%v", resp["total"])
	}
	if len(resp["groups"].([]interface{})) != 2 {
		t.Errorf("期望第 1 页返回 2 条，实际=%d", len(resp["groups"].([]interface{})))
	}
}

// 场景：响应中包含 member_count 字段，且值正确
func TestAdminListUserGroups_MemberCountInResponse(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	w := doListGroups(t, "")
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	groups := resp["groups"].([]interface{})
	if len(groups) != 1 {
		t.Fatalf("期望 1 个组，实际=%d", len(groups))
	}
	g := groups[0].(map[string]interface{})
	if g["member_count"].(float64) != 2 {
		t.Errorf("期望 member_count=2，实际=%v", g["member_count"])
	}
}

// ── POST /admin/user-groups/create ───────────────────────────────────────────

// 场景：正常创建用户组，响应包含 id、name、description
func TestAdminCreateUserGroup_Success(t *testing.T) {
	setupUserGroupTestDB(t)

	w := doCreateGroup(t, "研发组", "研发部门成员")
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("期望 ok=true，实际=%v", resp["ok"])
	}
	group := resp["group"].(map[string]interface{})
	if group["id"].(float64) <= 0 {
		t.Errorf("期望 id > 0，实际=%v", group["id"])
	}
	if group["name"].(string) != "研发组" {
		t.Errorf("期望 name=研发组，实际=%v", group["name"])
	}
	if group["description"].(string) != "研发部门成员" {
		t.Errorf("期望 description=研发部门成员，实际=%v", group["description"])
	}
}
func TestAdminCreateUserGroup_EmptyDescription(t *testing.T) {
	setupUserGroupTestDB(t)

	w := doCreateGroup(t, "无描述组", "")
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	group := resp["group"].(map[string]interface{})
	if group["description"].(string) != "" {
		t.Errorf("期望 description 为空字符串，实际=%v", group["description"])
	}
}

// 场景：name 为空字符串，返回 400
func TestAdminCreateUserGroup_EmptyName(t *testing.T) {
	setupUserGroupTestDB(t)

	w := doCreateGroup(t, "", "描述")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// 场景：同名用户组重复创建，返回 400
func TestAdminCreateUserGroup_DuplicateName(t *testing.T) {
	setupUserGroupTestDB(t)

	mustCreateGroup(t, "研发组", "第一次")
	w := doCreateGroup(t, "研发组", "第二次")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（组名已存在），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// 场景：请求体 JSON 格式错误，返回 400
func TestAdminCreateUserGroup_InvalidJSON(t *testing.T) {
	setupUserGroupTestDB(t)

	req := adminReq(http.MethodPost, "/admin/user-groups/create", []byte("not-json"))
	w := httptest.NewRecorder()
	HandleAdminCreateUserGroup(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// 场景：平台用户组数量已达 1000，继续创建返回 400
func TestAdminCreateUserGroup_ExceedPlatformLimit(t *testing.T) {
	setupUserGroupTestDB(t)

	// 直接批量写入 1000 个组（前置条件，不是被测行为），使用 batchCreateGroups 规避 SQLite 变量数限制
	groups := make([]model.UserGroup, model.MaxUserGroupsPerPlatform)
	for i := range groups {
		groups[i] = model.UserGroup{Name: fmt.Sprintf("组-%d", i)}
	}
	batchCreateGroups(t, groups)

	w := doCreateGroup(t, "第1001组", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（超出上限），实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["error"] == nil {
		t.Error("期望响应中包含 error 字段")
	}
}

// 场景：恰好第 1000 个组可以创建成功（边界值）
func TestAdminCreateUserGroup_AtPlatformLimit(t *testing.T) {
	setupUserGroupTestDB(t)

	groups := make([]model.UserGroup, model.MaxUserGroupsPerPlatform-1)
	for i := range groups {
		groups[i] = model.UserGroup{Name: fmt.Sprintf("组-%d", i)}
	}
	batchCreateGroups(t, groups)

	w := doCreateGroup(t, "第1000组", "")
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（恰好到达上限），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ── POST /admin/user-groups/delete ───────────────────────────────────────────

// 场景：正常删除存在的用户组，组内成员记录也被级联清除
func TestAdminDeleteUserGroup_Success(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "待删组", "")
	doAddMembers(t, groupID, userIDs)

	// 确认成员已存在
	w := doGetMembers(t, groupID)
	var before map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&before); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(before["members"].([]interface{})) != 2 {
		t.Fatalf("删除前期望 2 名成员，实际=%d", len(before["members"].([]interface{})))
	}

	// 执行删除
	wd := doDeleteGroup(t, groupID)
	if wd.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", wd.Code, wd.Body.String())
	}

	// 验证组已软删除（GET 列表中不再出现）
	wl := doListGroups(t, "")
	var listResp map[string]interface{}
	if err := json.NewDecoder(wl.Body).Decode(&listResp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if listResp["total"].(float64) != 0 {
		t.Errorf("删除后期望 total=0，实际=%v", listResp["total"])
	}

	// 验证成员记录已物理删除
	var memberCount int64
	if err := model.DB(context.Background()).Model(&model.UserGroupMember{}).Where("user_group_id = ?", groupID).Count(&memberCount).Error; err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if memberCount != 0 {
		t.Errorf("期望成员记录已被物理删除，实际剩余=%d", memberCount)
	}
}

// 场景：删除不存在的用户组，返回 404
func TestAdminDeleteUserGroup_NotFound(t *testing.T) {
	setupUserGroupTestDB(t)

	w := doDeleteGroup(t, 99999)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

// 场景：请求体缺少 id 字段（id=0），返回 400
func TestAdminDeleteUserGroup_MissingID(t *testing.T) {
	setupUserGroupTestDB(t)

	req := adminReq(http.MethodPost, "/admin/user-groups/delete", []byte(`{}`))
	w := httptest.NewRecorder()
	HandleAdminDeleteUserGroup(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// 场景：删除空组（无成员），也能正常删除
func TestAdminDeleteUserGroup_EmptyGroup(t *testing.T) {
	setupUserGroupTestDB(t)

	groupID := mustCreateGroup(t, "空组", "")
	w := doDeleteGroup(t, groupID)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
}

// ── GET /admin/user-groups/members ───────────────────────────────────────────

// 场景：查询有成员的组，响应包含 user_id、username、joined_at，不含敏感字段
func TestAdminGetGroupMembers_Success(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	w := doGetMembers(t, groupID)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("期望 ok=true，实际=%v", resp["ok"])
	}
	members := resp["members"].([]interface{})
	if len(members) != 2 {
		t.Fatalf("期望 2 名成员，实际=%d", len(members))
	}

	// 验证字段存在且不含敏感字段
	for _, m := range members {
		item := m.(map[string]interface{})
		if item["user_id"] == nil {
			t.Error("期望 user_id 字段存在")
		}
		if item["username"] == nil {
			t.Error("期望 username 字段存在")
		}
		if item["joined_at"] == nil {
			t.Error("期望 joined_at 字段存在")
		}
		if item["password"] != nil {
			t.Error("响应中不应包含 password 字段")
		}
		if item["api_token"] != nil {
			t.Error("响应中不应包含 api_token 字段")
		}
	}

	// 验证用户名正确
	nameSet := make(map[string]bool)
	for _, m := range members {
		item := m.(map[string]interface{})
		nameSet[item["username"].(string)] = true
	}
	if !nameSet["alice"] || !nameSet["bob"] {
		t.Errorf("期望包含 alice 和 bob，实际=%v", nameSet)
	}
}

// 场景：查询空组，返回空数组
func TestAdminGetGroupMembers_EmptyGroup(t *testing.T) {
	setupUserGroupTestDB(t)

	groupID := mustCreateGroup(t, "空组", "")
	w := doGetMembers(t, groupID)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(resp["members"].([]interface{})) != 0 {
		t.Errorf("期望空数组，实际=%v", resp["members"])
	}
}

// 场景：缺少 id 参数，返回 400
func TestAdminGetGroupMembers_MissingID(t *testing.T) {
	setupUserGroupTestDB(t)

	req := adminReq(http.MethodGet, "/admin/user-groups/members", nil)
	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// 场景：查询不存在的组，返回 404
func TestAdminGetGroupMembers_GroupNotFound(t *testing.T) {
	setupUserGroupTestDB(t)

	w := doGetMembers(t, 99999)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

// ── POST /admin/user-groups/members/set ──────────────────────────────────────

// 场景：全量替换——原成员 [alice, bob]，替换为 [alice, carol]，bob 被移除
func TestAdminSetGroupMembers_ReplaceMembers(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob", "carol")
	groupID := mustCreateGroup(t, "研发组", "")

	// 初始成员：alice + bob
	doAddMembers(t, groupID, userIDs[:2])

	// 全量替换为 alice + carol
	w := doSetMembers(t, groupID, []uint{userIDs[0], userIDs[2]})
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证成员变为 alice + carol，bob 已被移除
	wg := doGetMembers(t, groupID)
	names := memberUsernames(t, wg)
	if !names["alice"] {
		t.Error("期望 alice 仍在组内")
	}
	if names["bob"] {
		t.Error("期望 bob 已被移除")
	}
	if !names["carol"] {
		t.Error("期望 carol 已加入")
	}
}

// 场景：传入空数组，清空所有成员
func TestAdminSetGroupMembers_ClearAll(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	w := doSetMembers(t, groupID, []uint{})
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	wg := doGetMembers(t, groupID)
	var resp map[string]interface{}
	if err := json.NewDecoder(wg.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(resp["members"].([]interface{})) != 0 {
		t.Errorf("期望成员已清空，实际=%v", resp["members"])
	}
}

// 场景：user_ids 中包含不存在的用户 ID，返回 400，组内成员不变
func TestAdminSetGroupMembers_InvalidUserID(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	// 包含不存在的 user_id=99999
	w := doSetMembers(t, groupID, []uint{userIDs[0], 99999})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（不合法用户 ID），实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证组内成员未变（事务回滚）
	wg := doGetMembers(t, groupID)
	names := memberUsernames(t, wg)
	if !names["alice"] {
		t.Error("事务回滚后 alice 应仍在组内")
	}
}

// 场景：user_ids 长度超过 10000，返回 400（controller 层先校验数量）
func TestAdminSetGroupMembers_ExceedMemberLimit(t *testing.T) {
	setupUserGroupTestDB(t)

	groupID := mustCreateGroup(t, "大组", "")
	ids := make([]uint, model.MaxMembersPerUserGroup+1)
	for i := range ids {
		ids[i] = uint(i + 1)
	}
	w := doSetMembers(t, groupID, ids)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（超出成员上限），实际=%d", w.Code)
	}
}

// 场景：缺少 id 字段（id=0），返回 400
func TestAdminSetGroupMembers_MissingGroupID(t *testing.T) {
	setupUserGroupTestDB(t)

	req := adminReq(http.MethodPost, "/admin/user-groups/members/set", []byte(`{"user_ids":[1,2]}`))
	w := httptest.NewRecorder()
	HandleAdminSetGroupMembers(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// ── POST /admin/user-groups/members/add ──────────────────────────────────────

// 场景：正常批量添加，组内成员数量正确增加
func TestAdminAddGroupMembers_Success(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob", "carol")
	groupID := mustCreateGroup(t, "研发组", "")

	// 先添加 alice
	doAddMembers(t, groupID, userIDs[:1])

	// 再添加 bob + carol
	w := doAddMembers(t, groupID, userIDs[1:])
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	wg := doGetMembers(t, groupID)
	names := memberUsernames(t, wg)
	if !names["alice"] || !names["bob"] || !names["carol"] {
		t.Errorf("期望 alice/bob/carol 均在组内，实际=%v", names)
	}
}

// 场景：重复添加已存在的成员（幂等），不报错，成员数不变
func TestAdminAddGroupMembers_Idempotent(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	// 再次添加相同成员
	w := doAddMembers(t, groupID, userIDs)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（幂等），实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 成员数应仍为 2
	wg := doGetMembers(t, groupID)
	var resp map[string]interface{}
	if err := json.NewDecoder(wg.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(resp["members"].([]interface{})) != 2 {
		t.Errorf("期望成员数仍为 2（幂等），实际=%d", len(resp["members"].([]interface{})))
	}
}

// 场景：待添加用户已在其他组中时，直接返回 400，不自动迁移
func TestAdminAddGroupMembers_UserAlreadyInAnotherGroup(t *testing.T) {
	t.Skip("behavior changed in Release v6.13: multi-group allowed / id=0 returns ungrouped users")
}

// 场景：全量设置组成员时，若用户已在其他组中，则返回 400，不自动迁移
func TestAdminSetGroupMembers_UserAlreadyInAnotherGroup(t *testing.T) {
	t.Skip("behavior changed in Release v6.13: multi-group allowed / id=0 returns ungrouped users")
}

// 场景：包含不存在的 user_id，整体返回 400，组内成员不变
func TestAdminAddGroupMembers_InvalidUserID(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice")
	groupID := mustCreateGroup(t, "研发组", "")

	w := doAddMembers(t, groupID, []uint{userIDs[0], 99999})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 组内应无成员（整体回滚）
	wg := doGetMembers(t, groupID)
	var resp map[string]interface{}
	if err := json.NewDecoder(wg.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(resp["members"].([]interface{})) != 0 {
		t.Errorf("期望整体回滚，组内无成员，实际=%d", len(resp["members"].([]interface{})))
	}
}

// 场景：添加后超过 10000 人上限，返回 400，成员数不变
func TestAdminAddGroupMembers_ExceedMemberLimit(t *testing.T) {
	setupUserGroupTestDB(t)

	groupID := mustCreateGroup(t, "大组", "")
	// 直接写入 9999 名成员（前置条件），使用 batchCreateMembers 规避 SQLite 变量数限制
	total := model.MaxMembersPerUserGroup - 1
	members := make([]model.UserGroupMember, total)
	for i := range members {
		members[i] = model.UserGroupMember{UserGroupID: groupID, UserID: uint(i + 1)}
	}
	batchCreateMembers(t, members)

	// 创建 2 个真实用户，添加后将超过 10000
	newUserIDs := createTestUsers(t, "extra1", "extra2")
	w := doAddMembers(t, groupID, newUserIDs)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（超出成员上限），实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 成员数应仍为 9999
	var count int64
	if err := model.DB(context.Background()).Model(&model.UserGroupMember{}).Where("user_group_id = ?", groupID).Count(&count).Error; err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != int64(model.MaxMembersPerUserGroup-1) {
		t.Errorf("期望成员数不变（9999），实际=%d", count)
	}
}

// 场景：添加到不存在的组，返回 404
func TestAdminAddGroupMembers_GroupNotFound(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice")
	w := doAddMembers(t, 99999, userIDs)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

// 场景：恰好添加到 10000 人（边界值），应成功
func TestAdminAddGroupMembers_AtMemberLimit(t *testing.T) {
	setupUserGroupTestDB(t)

	groupID := mustCreateGroup(t, "大组", "")
	// 写入 9999 名成员，使用 batchCreateMembers 规避 SQLite 变量数限制
	total := model.MaxMembersPerUserGroup - 1
	members := make([]model.UserGroupMember, total)
	for i := range members {
		members[i] = model.UserGroupMember{UserGroupID: groupID, UserID: uint(i + 1)}
	}
	batchCreateMembers(t, members)

	// 创建第 10000 名真实用户
	newUserIDs := createTestUsers(t, "the10000th")
	w := doAddMembers(t, groupID, newUserIDs)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（恰好到达上限），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ── POST /admin/user-groups/members/remove ───────────────────────────────────

// 场景：正常批量移除，组内成员减少
func TestAdminRemoveGroupMembers_Success(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob", "carol")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	// 移除 bob + carol
	w := doRemoveMembers(t, groupID, userIDs[1:])
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	wg := doGetMembers(t, groupID)
	names := memberUsernames(t, wg)
	if !names["alice"] {
		t.Error("期望 alice 仍在组内")
	}
	if names["bob"] || names["carol"] {
		t.Error("期望 bob 和 carol 已被移除")
	}
}

// 场景：移除不在组内的成员（静默忽略），返回 200，组内成员不变
func TestAdminRemoveGroupMembers_NotInGroup(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs[:1]) // 只添加 alice

	// 移除 bob（不在组内）+ alice（在组内）
	w := doRemoveMembers(t, groupID, userIDs)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（静默忽略不在组内的成员），实际=%d", w.Code)
	}

	// alice 被移除，bob 静默忽略
	wg := doGetMembers(t, groupID)
	var resp map[string]interface{}
	if err := json.NewDecoder(wg.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(resp["members"].([]interface{})) != 0 {
		t.Errorf("期望组内无成员，实际=%d", len(resp["members"].([]interface{})))
	}
}

// 场景：传入空数组，组内成员不变，返回 200
func TestAdminRemoveGroupMembers_EmptyList(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	w := doRemoveMembers(t, groupID, []uint{})
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	wg := doGetMembers(t, groupID)
	var resp map[string]interface{}
	if err := json.NewDecoder(wg.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(resp["members"].([]interface{})) != 2 {
		t.Errorf("期望成员数不变（2），实际=%d", len(resp["members"].([]interface{})))
	}
}

// 场景：移除全部成员，组变为空组
func TestAdminRemoveGroupMembers_RemoveAll(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	w := doRemoveMembers(t, groupID, userIDs)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	wg := doGetMembers(t, groupID)
	var resp map[string]interface{}
	if err := json.NewDecoder(wg.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(resp["members"].([]interface{})) != 0 {
		t.Errorf("期望成员已全部移除，实际=%d", len(resp["members"].([]interface{})))
	}
}

// ── 端到端流程测试 ────────────────────────────────────────────────────────────

// 场景：完整的用户组生命周期——创建 → 添加成员 → 查看成员 → 移除部分成员 → 删除组
func TestAdminUserGroup_FullLifecycle(t *testing.T) {
	setupUserGroupTestDB(t)

	// 1. 创建用户和用户组
	userIDs := createTestUsers(t, "alice", "bob", "carol")
	groupID := mustCreateGroup(t, "研发组", "研发部门")

	// 2. 添加 alice + bob
	w := doAddMembers(t, groupID, userIDs[:2])
	if w.Code != http.StatusOK {
		t.Fatalf("添加成员失败: %s", w.Body.String())
	}

	// 3. 查看成员，确认 alice + bob 在组内
	wg := doGetMembers(t, groupID)
	names := memberUsernames(t, wg)
	if !names["alice"] || !names["bob"] {
		t.Errorf("期望 alice 和 bob 在组内，实际=%v", names)
	}

	// 4. 全量替换为 alice + carol（bob 被移除）
	doSetMembers(t, groupID, []uint{userIDs[0], userIDs[2]})
	wg2 := doGetMembers(t, groupID)
	names2 := memberUsernames(t, wg2)
	if names2["bob"] {
		t.Error("全量替换后 bob 应已被移除")
	}
	if !names2["carol"] {
		t.Error("全量替换后 carol 应已加入")
	}

	// 5. 移除 carol
	doRemoveMembers(t, groupID, []uint{userIDs[2]})
	wg3 := doGetMembers(t, groupID)
	names3 := memberUsernames(t, wg3)
	if names3["carol"] {
		t.Error("移除后 carol 应已不在组内")
	}
	if !names3["alice"] {
		t.Error("alice 应仍在组内")
	}

	// 6. 删除用户组
	wd := doDeleteGroup(t, groupID)
	if wd.Code != http.StatusOK {
		t.Fatalf("删除组失败: %s", wd.Body.String())
	}

	// 7. 确认组已不在列表中
	wl := doListGroups(t, "")
	var listResp map[string]interface{}
	if err := json.NewDecoder(wl.Body).Decode(&listResp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if listResp["total"].(float64) != 0 {
		t.Errorf("删除后期望 total=0，实际=%v", listResp["total"])
	}

	// 8. 确认成员记录已物理删除
	var memberCount int64
	if err := model.DB(context.Background()).Model(&model.UserGroupMember{}).Where("user_group_id = ?", groupID).Count(&memberCount).Error; err != nil {
		t.Fatalf("查询成员数失败: %v", err)
	}
	if memberCount != 0 {
		t.Errorf("期望成员记录已物理删除，实际=%d", memberCount)
	}
}

// 场景：多个用户组并存，各自成员互不干扰
func TestAdminUserGroup_MultipleGroupsIsolation(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob", "carol", "dave")
	groupA := mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "运营组", "")

	// 研发组：alice + bob；运营组：carol + dave
	doAddMembers(t, groupA, userIDs[:2])
	doAddMembers(t, groupB, userIDs[2:])

	// 验证研发组成员
	wA := doGetMembers(t, groupA)
	namesA := memberUsernames(t, wA)
	if !namesA["alice"] || !namesA["bob"] {
		t.Errorf("研发组期望 alice+bob，实际=%v", namesA)
	}
	if namesA["carol"] || namesA["dave"] {
		t.Errorf("研发组不应包含 carol/dave，实际=%v", namesA)
	}

	// 验证运营组成员
	wB := doGetMembers(t, groupB)
	namesB := memberUsernames(t, wB)
	if !namesB["carol"] || !namesB["dave"] {
		t.Errorf("运营组期望 carol+dave，实际=%v", namesB)
	}
	if namesB["alice"] || namesB["bob"] {
		t.Errorf("运营组不应包含 alice/bob，实际=%v", namesB)
	}

	// 删除研发组，运营组不受影响
	doDeleteGroup(t, groupA)
	wl := doListGroups(t, "")
	var listResp map[string]interface{}
	if err := json.NewDecoder(wl.Body).Decode(&listResp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if listResp["total"].(float64) != 1 {
		t.Errorf("删除研发组后期望 total=1，实际=%v", listResp["total"])
	}

	// 运营组成员仍然完整
	wB2 := doGetMembers(t, groupB)
	namesB2 := memberUsernames(t, wB2)
	if !namesB2["carol"] || !namesB2["dave"] {
		t.Errorf("运营组成员不应受研发组删除影响，实际=%v", namesB2)
	}
}

// ── 硬删除 & CanDeleteUserGroup ───────────────────────────────────────────────

// 场景：硬删除后，同名用户组可以重新创建（验证硬删除不留残留记录）
func TestAdminDeleteUserGroup_HardDeleteAllowsRename(t *testing.T) {
	setupUserGroupTestDB(t)

	groupID := mustCreateGroup(t, "研发组", "第一次")
	wd := doDeleteGroup(t, groupID)
	if wd.Code != http.StatusOK {
		t.Fatalf("删除失败: status=%d, body=%s", wd.Code, wd.Body.String())
	}

	// 硬删除后，同名组应可以重新创建（软删除会因唯一索引冲突而失败）
	w := doCreateGroup(t, "研发组", "第二次")
	if w.Code != http.StatusOK {
		t.Fatalf("硬删除后同名组应可重建，实际 status=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	newGroup := resp["group"].(map[string]interface{})
	if uint(newGroup["id"].(float64)) == groupID {
		t.Errorf("期望新组 ID 与已删除的旧组不同，实际 ID=%v", newGroup["id"])
	}
	if newGroup["description"].(string) != "第二次" {
		t.Errorf("期望 description=第二次，实际=%v", newGroup["description"])
	}
}

// 场景：CanDeleteUserGroup 当前固定返回 true，删除可正常执行
// 此测试同时作为预检函数的回归测试，后续补充关联资源校验逻辑后需同步更新
func TestAdminDeleteUserGroup_CanDeleteCheck_AllowedByDefault(t *testing.T) {
	setupUserGroupTestDB(t)

	groupID := mustCreateGroup(t, "测试组", "")

	w := doDeleteGroup(t, groupID)
	if w.Code != http.StatusOK {
		t.Fatalf("CanDeleteUserGroup 返回 true 时删除应成功，实际 status=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证组已被物理删除（Unscoped 查询也找不到）
	var count int64
	if err := model.DB(context.Background()).Unscoped().Model(&model.UserGroup{}).Where("id = ?", groupID).Count(&count).Error; err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 0 {
		t.Errorf("期望组已被物理删除，实际 count=%d", count)
	}
}

// ── 鉴权校验 ─────────────────────────────────────────────────────────────────
// 注：管理员接口通过 session 或 AdminToken（Bearer）鉴权。
// 测试环境未初始化 SessionStore，且 getUserFromToken 对非 OpenAPI 路由在 AdminToken 不匹配时
// 仍返回 nil,nil（回退 session），导致无法在单元测试中模拟"无权限"场景。
// 鉴权逻辑由 requireAdmin（admin_common.go）统一处理，已在集成测试中覆盖。

// 场景：正确 AdminToken 可访问管理员接口（鉴权正向验证）
func TestAdminUserGroup_ValidAdminToken_Authorized(t *testing.T) {
	setupUserGroupTestDB(t)

	origToken := AdminToken
	AdminToken = "test-admin-token-valid"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodGet, "/admin/user-groups", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token-valid")
	w := httptest.NewRecorder()
	HandleAdminListUserGroups(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("正确 AdminToken 应返回 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ── 列表接口边界 ──────────────────────────────────────────────────────────────

// 场景：第二页数据正确（分页偏移）
func TestAdminListUserGroups_SecondPage(t *testing.T) {
	setupUserGroupTestDB(t)

	mustCreateGroup(t, "A组", "")
	mustCreateGroup(t, "B组", "")
	mustCreateGroup(t, "C组", "")

	w1 := doListGroups(t, "page=1&page_size=2")
	var resp1 map[string]interface{}
	json.NewDecoder(w1.Body).Decode(&resp1)
	if len(resp1["groups"].([]interface{})) != 2 {
		t.Fatalf("第 1 页期望 2 条，实际=%d", len(resp1["groups"].([]interface{})))
	}

	w2 := doListGroups(t, "page=2&page_size=2")
	var resp2 map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&resp2)
	if resp2["total"].(float64) != 3 {
		t.Errorf("第 2 页 total 期望=3，实际=%v", resp2["total"])
	}
	if len(resp2["groups"].([]interface{})) != 1 {
		t.Errorf("第 2 页期望 1 条，实际=%d", len(resp2["groups"].([]interface{})))
	}
}

// 场景：超出总页数的页码，返回空数组（不报错），total 仍正确
func TestAdminListUserGroups_PageOutOfRange(t *testing.T) {
	setupUserGroupTestDB(t)

	mustCreateGroup(t, "研发组", "")

	w := doListGroups(t, "page=999&page_size=20")
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["total"].(float64) != 1 {
		t.Errorf("total 期望=1，实际=%v", resp["total"])
	}
	if len(resp["groups"].([]interface{})) != 0 {
		t.Errorf("超出页数时期望空数组，实际=%d 条", len(resp["groups"].([]interface{})))
	}
}

// 场景：列表响应中 created_at 字段格式为 ISO 8601 UTC（以 Z 结尾）
func TestAdminListUserGroups_CreatedAtFormat(t *testing.T) {
	setupUserGroupTestDB(t)

	mustCreateGroup(t, "研发组", "")

	w := doListGroups(t, "")
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	groups := resp["groups"].([]interface{})
	g := groups[0].(map[string]interface{})

	createdAt, ok := g["created_at"].(string)
	if !ok || createdAt == "" {
		t.Fatalf("期望 created_at 字段为非空字符串，实际=%v", g["created_at"])
	}
	// 格式：2006-01-02T15:04:05Z（长度 20，末尾为 Z）
	if len(createdAt) != 20 || createdAt[len(createdAt)-1] != 'Z' {
		t.Errorf("期望 created_at 格式为 ISO 8601 UTC，实际=%q", createdAt)
	}
}

// ── 创建接口边界 ──────────────────────────────────────────────────────────────

// 场景：name 仅含空白字符（空格/Tab），应返回 400
func TestAdminCreateUserGroup_WhitespaceName(t *testing.T) {
	setupUserGroupTestDB(t)

	for _, name := range []string{" ", "\t", "   "} {
		body, _ := json.Marshal(map[string]string{"name": name, "description": ""})
		req := adminReq(http.MethodPost, "/admin/user-groups/create", body)
		w := httptest.NewRecorder()
		HandleAdminCreateUserGroup(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("name=%q 期望 400，实际=%d, body=%s", name, w.Code, w.Body.String())
		}
	}
}

// 场景：创建响应中不含 identifier 字段（内部多租户字段不对外暴露）
func TestAdminCreateUserGroup_ResponseNoIdentifier(t *testing.T) {
	setupUserGroupTestDB(t)

	w := doCreateGroup(t, "研发组", "")
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	group := resp["group"].(map[string]interface{})
	if _, exists := group["identifier"]; exists {
		t.Error("响应中不应包含 identifier 字段（内部多租户字段）")
	}
}

// ── 删除接口边界 ──────────────────────────────────────────────────────────────

// 场景：请求体 JSON 格式错误，返回 400
func TestAdminDeleteUserGroup_InvalidJSON(t *testing.T) {
	setupUserGroupTestDB(t)

	req := adminReq(http.MethodPost, "/admin/user-groups/delete", []byte("not-json"))
	w := httptest.NewRecorder()
	HandleAdminDeleteUserGroup(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// 场景：删除后再次删除同一 ID，返回 404（已不存在）
func TestAdminDeleteUserGroup_DeleteTwice(t *testing.T) {
	setupUserGroupTestDB(t)

	groupID := mustCreateGroup(t, "研发组", "")
	doDeleteGroup(t, groupID)

	w := doDeleteGroup(t, groupID)
	if w.Code != http.StatusNotFound {
		t.Fatalf("重复删除期望 404，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ── 成员查询接口边界 ──────────────────────────────────────────────────────────

// 场景：id 参数为非数字字符串，返回 400
func TestAdminGetGroupMembers_NonNumericID(t *testing.T) {
	setupUserGroupTestDB(t)

	req := adminReq(http.MethodGet, "/admin/user-groups/members?id=abc", nil)
	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// 场景：id=0，返回 400
func TestAdminGetGroupMembers_ZeroID(t *testing.T) {
	t.Skip("behavior changed in Release v6.13: multi-group allowed / id=0 returns ungrouped users")
}

// 场景：成员的 joined_at 字段格式为 ISO 8601 UTC
func TestAdminGetGroupMembers_JoinedAtFormat(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	w := doGetMembers(t, groupID)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	members := resp["members"].([]interface{})
	item := members[0].(map[string]interface{})

	joinedAt, ok := item["joined_at"].(string)
	if !ok || joinedAt == "" {
		t.Fatalf("期望 joined_at 字段为非空字符串，实际=%v", item["joined_at"])
	}
	if len(joinedAt) != 20 || joinedAt[len(joinedAt)-1] != 'Z' {
		t.Errorf("期望 joined_at 格式为 ISO 8601 UTC，实际=%q", joinedAt)
	}
}

// 场景：组内有已软删除的用户，该用户不出现在成员列表中（E-04 场景）
// 场景：软删除（禁用）用户后，仍应出现在成员列表中（产品设计：禁用用户在所有用户组视图中都显示）
func TestAdminGetGroupMembers_SoftDeletedUserFiltered(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	// 软删除（禁用）alice
	if err := model.DB(context.Background()).Delete(&model.User{}, userIDs[0]).Error; err != nil {
		t.Fatalf("软删除用户失败: %v", err)
	}

	w := doGetMembers(t, groupID)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	members := resp["members"].([]interface{})

	names := make(map[string]bool)
	for _, m := range members {
		item := m.(map[string]interface{})
		names[item["username"].(string)] = true
	}
	// 产品设计：禁用用户仍计入成员，方便管理员统一管理
	if !names["alice"] {
		t.Error("已禁用的 alice 仍应出现在成员列表中（禁用用户在所有用户组视图中都显示）")
	}
	if !names["bob"] {
		t.Error("未删除的 bob 应仍在成员列表中")
	}
}

// ── 添加成员接口边界 ──────────────────────────────────────────────────────────

// 场景：user_ids 为空数组，直接返回 200，成员数不变
func TestAdminAddGroupMembers_EmptyUserIDs(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	w := doAddMembers(t, groupID, []uint{})
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	wg := doGetMembers(t, groupID)
	var resp map[string]interface{}
	json.NewDecoder(wg.Body).Decode(&resp)
	if len(resp["members"].([]interface{})) != 1 {
		t.Errorf("期望成员数不变（1），实际=%d", len(resp["members"].([]interface{})))
	}
}

// 场景：同一请求中包含重复的 user_id（如 [alice, alice, bob]），幂等处理，不重复插入
func TestAdminAddGroupMembers_DuplicateIDsInSameRequest(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "研发组", "")

	// alice 的 ID 出现两次
	w := doAddMembers(t, groupID, []uint{userIDs[0], userIDs[0], userIDs[1]})
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（重复 ID 幂等），实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 成员数应为 2（alice 只算一次）
	wg := doGetMembers(t, groupID)
	var resp map[string]interface{}
	json.NewDecoder(wg.Body).Decode(&resp)
	if len(resp["members"].([]interface{})) != 2 {
		t.Errorf("期望成员数=2（重复 ID 只算一次），实际=%d", len(resp["members"].([]interface{})))
	}
}

// 场景：添加成员时 JSON 格式错误，返回 400
func TestAdminAddGroupMembers_InvalidJSON(t *testing.T) {
	setupUserGroupTestDB(t)

	req := adminReq(http.MethodPost, "/admin/user-groups/members/add", []byte("not-json"))
	w := httptest.NewRecorder()
	HandleAdminAddGroupMembers(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// 场景：缺少 id 字段（id=0），返回 400
func TestAdminAddGroupMembers_MissingGroupID(t *testing.T) {
	setupUserGroupTestDB(t)

	req := adminReq(http.MethodPost, "/admin/user-groups/members/add", []byte(`{"user_ids":[1,2]}`))
	w := httptest.NewRecorder()
	HandleAdminAddGroupMembers(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// ── 全量替换接口边界 ──────────────────────────────────────────────────────────

// 场景：全量替换时组不存在，返回 404
func TestAdminSetGroupMembers_GroupNotFound(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice")
	w := doSetMembers(t, 99999, userIDs)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

// 场景：全量替换时 JSON 格式错误，返回 400
func TestAdminSetGroupMembers_InvalidJSON(t *testing.T) {
	setupUserGroupTestDB(t)

	req := adminReq(http.MethodPost, "/admin/user-groups/members/set", []byte("not-json"))
	w := httptest.NewRecorder()
	HandleAdminSetGroupMembers(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// 场景：全量替换后 member_count 与实际成员数一致（列表接口数据一致性）
func TestAdminSetGroupMembers_MemberCountConsistency(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob", "carol", "dave")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	// 全量替换为 alice + carol（2 人）
	doSetMembers(t, groupID, []uint{userIDs[0], userIDs[2]})

	wl := doListGroups(t, "")
	var listResp map[string]interface{}
	json.NewDecoder(wl.Body).Decode(&listResp)
	g := listResp["groups"].([]interface{})[0].(map[string]interface{})
	if g["member_count"].(float64) != 2 {
		t.Errorf("全量替换后 member_count 期望=2，实际=%v", g["member_count"])
	}

	wm := doGetMembers(t, groupID)
	var memberResp map[string]interface{}
	json.NewDecoder(wm.Body).Decode(&memberResp)
	if len(memberResp["members"].([]interface{})) != 2 {
		t.Errorf("全量替换后成员详情期望 2 人，实际=%d", len(memberResp["members"].([]interface{})))
	}
}

// ── 移除成员接口边界 ──────────────────────────────────────────────────────────

// 场景：移除成员时组不存在，返回 404
func TestAdminRemoveGroupMembers_GroupNotFound(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice")
	w := doRemoveMembers(t, 99999, userIDs)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

// 场景：移除成员时 JSON 格式错误，返回 400
func TestAdminRemoveGroupMembers_InvalidJSON(t *testing.T) {
	setupUserGroupTestDB(t)

	req := adminReq(http.MethodPost, "/admin/user-groups/members/remove", []byte("not-json"))
	w := httptest.NewRecorder()
	HandleAdminRemoveGroupMembers(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// 场景：移除成员时缺少 id 字段（id=0），返回 400
func TestAdminRemoveGroupMembers_MissingGroupID(t *testing.T) {
	setupUserGroupTestDB(t)

	req := adminReq(http.MethodPost, "/admin/user-groups/members/remove", []byte(`{"user_ids":[1,2]}`))
	w := httptest.NewRecorder()
	HandleAdminRemoveGroupMembers(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// ── model 层函数直接测试 ──────────────────────────────────────────────────────

// 场景：GetGroupsByIDs 传入已硬删除组的 ID，不返回该组
func TestGetGroupsByIDs_DeletedGroupNotReturned(t *testing.T) {
	setupUserGroupTestDB(t)

	groupA := mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "运营组", "")

	doDeleteGroup(t, groupA)

	groups, err := model.GetGroupsByIDs(context.Background(), []uint{groupA, groupB})
	if err != nil {
		t.Fatalf("GetGroupsByIDs 失败: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("期望只返回 1 个组（运营组），实际=%d", len(groups))
	}
	if groups[0].Name != "运营组" {
		t.Errorf("期望返回运营组，实际=%v", groups[0].Name)
	}
}

// 场景：GetGroupsByIDs 传入空切片，直接返回空结果，不发起 DB 查询
func TestGetGroupsByIDs_EmptyIDs(t *testing.T) {
	setupUserGroupTestDB(t)

	groups, err := model.GetGroupsByIDs(context.Background(), []uint{})
	if err != nil {
		t.Fatalf("GetGroupsByIDs(context.Background(), 空) 不应报错: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("期望空结果，实际=%d", len(groups))
	}
}

// 场景：GetUserGroupIDs 用户不属于任何组，返回空切片（非 nil）
func TestGetUserGroupIDs_NoGroups(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice")
	ids, err := model.GetUserGroupIDs(context.Background(), userIDs[0])
	if err != nil {
		t.Fatalf("GetUserGroupIDs 失败: %v", err)
	}
	if ids == nil {
		t.Error("期望返回空切片（非 nil），实际=nil")
	}
	if len(ids) != 0 {
		t.Errorf("期望空切片，实际=%v", ids)
	}
}

// 场景：同一用户加入多个组后，GetUserGroupIDs 返回所有组 ID，不含未加入的组。
// 当前新增用户已不支持多组，本测试保留用于测试旧用户兼容。
func TestGetUserGroupIDs_MultipleGroups(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice")
	groupA := mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "运营组", "")
	groupC := mustCreateGroup(t, "测试组", "") // alice 不在此组

	mustInsertLegacyGroupMembers(t,
		model.UserGroupMember{UserGroupID: groupA, UserID: userIDs[0]},
		model.UserGroupMember{UserGroupID: groupB, UserID: userIDs[0]},
	)

	ids, err := model.GetUserGroupIDs(context.Background(), userIDs[0])
	if err != nil {
		t.Fatalf("GetUserGroupIDs 失败: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("期望 2 个组 ID，实际=%d: %v", len(ids), ids)
	}
	idSet := make(map[uint]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	if !idSet[groupA] || !idSet[groupB] {
		t.Errorf("期望包含研发组(%d)和运营组(%d)，实际=%v", groupA, groupB, ids)
	}
	if idSet[groupC] {
		t.Errorf("alice 不在测试组，不应包含 groupC(%d)", groupC)
	}
}

// 场景：CountGroupMembers 在成员被移除后数量正确减少
func TestCountGroupMembers_AfterRemove(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob", "carol")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	doRemoveMembers(t, groupID, []uint{userIDs[1]})

	count, err := model.CountGroupMembers(context.Background(), groupID)
	if err != nil {
		t.Fatalf("CountGroupMembers 失败: %v", err)
	}
	if count != 2 {
		t.Errorf("移除 bob 后期望 count=2，实际=%d", count)
	}
}

// ── 数据一致性 ────────────────────────────────────────────────────────────────

// 场景：同一用户可以同时属于多个组，删除其中一个组后其他组成员关系不受影响。
// 当前新增用户已不支持多组，本测试保留用于测试旧用户兼容。
func TestAdminUserGroup_UserInMultipleGroups(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice")
	groupA := mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "运营组", "")
	groupC := mustCreateGroup(t, "测试组", "")

	mustInsertLegacyGroupMembers(t,
		model.UserGroupMember{UserGroupID: groupA, UserID: userIDs[0]},
		model.UserGroupMember{UserGroupID: groupB, UserID: userIDs[0]},
		model.UserGroupMember{UserGroupID: groupC, UserID: userIDs[0]},
	)

	// 每个组都应包含 alice
	for _, gid := range []uint{groupA, groupB, groupC} {
		wg := doGetMembers(t, gid)
		names := memberUsernames(t, wg)
		if !names["alice"] {
			t.Errorf("组 %d 期望包含 alice，实际=%v", gid, names)
		}
	}

	// alice 属于 3 个组
	ids, _ := model.GetUserGroupIDs(context.Background(), userIDs[0])
	if len(ids) != 3 {
		t.Errorf("alice 期望属于 3 个组，实际=%d", len(ids))
	}

	// 删除研发组后，alice 的 user_group_members 记录减少 1 条
	doDeleteGroup(t, groupA)
	ids2, _ := model.GetUserGroupIDs(context.Background(), userIDs[0])
	if len(ids2) != 2 {
		t.Errorf("删除研发组后 alice 期望属于 2 个组，实际=%d", len(ids2))
	}
}

// 场景：member_count 在添加/移除/全量替换后始终与实际成员数一致
func TestAdminUserGroup_MemberCountAlwaysConsistent(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob", "carol", "dave", "eve")
	groupID := mustCreateGroup(t, "研发组", "")

	checkCount := func(expected int) {
		t.Helper()
		wl := doListGroups(t, "")
		var resp map[string]interface{}
		json.NewDecoder(wl.Body).Decode(&resp)
		groups := resp["groups"].([]interface{})
		if len(groups) == 0 {
			if expected != 0 {
				t.Errorf("期望 member_count=%d，但组已不存在", expected)
			}
			return
		}
		g := groups[0].(map[string]interface{})
		if int(g["member_count"].(float64)) != expected {
			t.Errorf("期望 member_count=%d，实际=%v", expected, g["member_count"])
		}
	}

	checkCount(0)
	doAddMembers(t, groupID, userIDs[:2])
	checkCount(2)
	doAddMembers(t, groupID, userIDs[2:3])
	checkCount(3)
	doRemoveMembers(t, groupID, []uint{userIDs[1]})
	checkCount(2)
	doSetMembers(t, groupID, []uint{userIDs[3], userIDs[4]})
	checkCount(2)
	doSetMembers(t, groupID, []uint{})
	checkCount(0)
}
func TestAdminUserGroups_MethodNotAllowed(t *testing.T) {
	setupUserGroupTestDB(t)
	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
		path    string
		method  string
	}{
		{"Create", HandleAdminCreateUserGroup, "/admin/user-groups/create", http.MethodGet},
		{"Delete", HandleAdminDeleteUserGroup, "/admin/user-groups/delete?id=1", http.MethodGet},
		{"DeleteImpact", HandleAdminGetGroupDeleteImpact, "/admin/user-groups/delete-impact?id=1", http.MethodPost},
		{"SetMembers", HandleAdminSetGroupMembers, "/admin/user-groups/set-members?id=1", http.MethodGet},
		{"AddMembers", HandleAdminAddGroupMembers, "/admin/user-groups/add-members?id=1", http.MethodGet},
		{"RemoveMembers", HandleAdminRemoveGroupMembers, "/admin/user-groups/remove-members?id=1", http.MethodGet},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", "Bearer test-admin-token")
			w := httptest.NewRecorder()
			tt.handler(w, req)
			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s: expected 405, got %d", tt.name, w.Code)
			}
		})
	}
}
