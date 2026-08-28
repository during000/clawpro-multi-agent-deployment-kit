package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"
)

// ── 辅助函数 ──────────────────────────────────────────────────────────────────

// doUpdateGroup 调用 HandleAdminUpdateUserGroup（携带管理员 Token）。
func doUpdateGroup(t *testing.T, id uint, name, description string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{"id": id, "name": name, "description": description})
	w := httptest.NewRecorder()
	HandleAdminUpdateUserGroup(w, adminReq(http.MethodPost, "/admin/user-groups/update", body))
	return w
}

// doGetMembersPaged 调用 HandleAdminGetGroupMembers（支持分页参数）。
func doGetMembersPaged(t *testing.T, groupID uint, page, pageSize int) *httptest.ResponseRecorder {
	t.Helper()
	path := fmt.Sprintf("/admin/user-groups/members?id=%d&page=%d&page_size=%d", groupID, page, pageSize)
	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, adminReq(http.MethodGet, path, nil))
	return w
}

// doGetUngroupedUsers 调用 HandleAdmin 并传入 ungrouped=1 过滤参数。
func doGetUngroupedUsers(t *testing.T, page, pageSize int) *httptest.ResponseRecorder {
	t.Helper()
	path := fmt.Sprintf("/admin/users?ungrouped=1&page=%d&page_size=%d", page, pageSize)
	w := httptest.NewRecorder()
	HandleAdmin(w, adminReq(http.MethodGet, path, nil))
	return w
}

// doGetGroupsByUsers 调用 HandleAdminGetGroupsByUsers（批量模式）。
func doGetGroupsByUsers(t *testing.T, userIDs ...uint) *httptest.ResponseRecorder {
	t.Helper()
	parts := make([]string, len(userIDs))
	for i, id := range userIDs {
		parts[i] = fmt.Sprintf("%d", id)
	}
	path := "/admin/user-groups/groups-by-users?user_ids=" + strings.Join(parts, ",")
	w := httptest.NewRecorder()
	HandleAdminGetGroupsByUsers(w, adminReq(http.MethodGet, path, nil))
	return w
}

// doCreateUserJSON 通过 JSON 请求体调用 HandleCreateUser（携带管理员 Token）。
func doCreateUserJSON(t *testing.T, body map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/admin/users/create", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleCreateUser(w, req)
	return w
}

// doUpdateUserJSON 通过 JSON 请求体调用 HandleUpdateUser（携带管理员 Token）。
func doUpdateUserJSON(t *testing.T, userID uint, body map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	path := fmt.Sprintf("/admin/users/update?id=%d", userID)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleUpdateUser(w, req)
	return w
}

// doDeleteUserJSON 调用 HandleDeleteUser（携带管理员 Token）。
func doDeleteUserJSON(t *testing.T, userID uint) *httptest.ResponseRecorder {
	t.Helper()
	path := fmt.Sprintf("/admin/users/delete?id=%d", userID)
	req := httptest.NewRequest(http.MethodPost, path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleDeleteUser(w, req)
	return w
}

// doHardDeleteUserJSON 调用硬删除接口（物理删除用户及其成员记录）
func doHardDeleteUserJSON(t *testing.T, userID uint) *httptest.ResponseRecorder {
	t.Helper()
	path := fmt.Sprintf("/admin/users/hard-delete?id=%d", userID)
	req := httptest.NewRequest(http.MethodPost, path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleHardDeleteUser(w, req)
	return w
}

// setupUserGroupTestDBWithSiteConfig 初始化测试 DB，并确保 SiteConfig 存在（HandleCreateUser 依赖）。
// 复用 setupUserGroupTestDB，它已经创建了 SiteConfig。
func setupUserGroupTestDBWithSiteConfig(t *testing.T) {
	t.Helper()
	setupUserGroupTestDB(t)
	// 迁移 Instance 表（HandleDeleteUser 中 stopUserInstances 会查询）
	if err := model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.Instance{}); err != nil {
		t.Fatalf("迁移 Instance 表失败: %v", err)
	}
	// 迁移 OneIDDepartmentRecord 表（queryUsers 中 BuildFullDeptMap 会查询）
	if err := model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.OneIDDepartmentRecord{}); err != nil {
		t.Fatalf("迁移 OneIDDepartmentRecord 表失败: %v", err)
	}
}

// ── POST /admin/user-groups/update ───────────────────────────────────────────

// 场景：正常修改用户组名称和描述
func TestAdminUpdateUserGroup_Success(t *testing.T) {
	setupUserGroupTestDB(t)

	groupID := mustCreateGroup(t, "研发组", "研发部门")

	w := doUpdateGroup(t, groupID, "研发一组", "研发部门核心成员")
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
	if group["name"].(string) != "研发一组" {
		t.Errorf("期望 name=研发一组，实际=%v", group["name"])
	}
	if group["description"].(string) != "研发部门核心成员" {
		t.Errorf("期望 description=研发部门核心成员，实际=%v", group["description"])
	}
	if uint(group["id"].(float64)) != groupID {
		t.Errorf("期望 id=%d，实际=%v", groupID, group["id"])
	}
}

// 场景：只修改描述，名称不变
func TestAdminUpdateUserGroup_OnlyDescription(t *testing.T) {
	setupUserGroupTestDB(t)

	groupID := mustCreateGroup(t, "研发组", "旧描述")

	w := doUpdateGroup(t, groupID, "研发组", "新描述")
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	group := resp["group"].(map[string]interface{})
	if group["name"].(string) != "研发组" {
		t.Errorf("名称应保持不变，实际=%v", group["name"])
	}
	if group["description"].(string) != "新描述" {
		t.Errorf("期望 description=新描述，实际=%v", group["description"])
	}
}

// 场景：修改为已存在的名称，返回 400
func TestAdminUpdateUserGroup_DuplicateName(t *testing.T) {
	setupUserGroupTestDB(t)

	mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "运营组", "")

	// 将运营组改名为研发组（已存在），应返回 400
	w := doUpdateGroup(t, groupB, "研发组", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（名称重复），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// 场景：用户组 ID 不存在，返回 404
func TestAdminUpdateUserGroup_NotFound(t *testing.T) {
	setupUserGroupTestDB(t)

	w := doUpdateGroup(t, 99999, "新名称", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404（用户组不存在），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// 场景：名称为空，返回 400
func TestAdminUpdateUserGroup_EmptyName(t *testing.T) {
	setupUserGroupTestDB(t)

	groupID := mustCreateGroup(t, "研发组", "")

	w := doUpdateGroup(t, groupID, "", "描述")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（名称为空），实际=%d", w.Code)
	}
}

// 场景：ID 为 0，返回 400
func TestAdminUpdateUserGroup_ZeroID(t *testing.T) {
	setupUserGroupTestDB(t)

	w := doUpdateGroup(t, 0, "新名称", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（ID 为 0），实际=%d", w.Code)
	}
}

// 场景：修改后数据库中的记录确实已更新
func TestAdminUpdateUserGroup_DBUpdated(t *testing.T) {
	setupUserGroupTestDB(t)

	groupID := mustCreateGroup(t, "研发组", "旧描述")
	doUpdateGroup(t, groupID, "研发一组", "新描述")

	var group model.UserGroup
	if err := model.DB(context.Background()).First(&group, groupID).Error; err != nil {
		t.Fatalf("查询用户组失败: %v", err)
	}
	if group.Name != "研发一组" {
		t.Errorf("数据库中 name 期望=研发一组，实际=%s", group.Name)
	}
	if group.Description != "新描述" {
		t.Errorf("数据库中 description 期望=新描述，实际=%s", group.Description)
	}
}

// 场景：非 POST 方法返回 405
func TestAdminUpdateUserGroup_MethodNotAllowed(t *testing.T) {
	setupUserGroupTestDB(t)

	req := adminReq(http.MethodGet, "/admin/user-groups/update", nil)
	w := httptest.NewRecorder()
	HandleAdminUpdateUserGroup(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("期望 405，实际=%d", w.Code)
	}
}

// ── GET /admin/user-groups/members（分页）────────────────────────────────────

// 场景：成员列表分页，第 1 页返回正确数量，total 正确
func TestAdminGetGroupMembers_Pagination(t *testing.T) {
	setupUserGroupTestDB(t)

	// 创建 5 个用户
	userIDs := createTestUsers(t, "u1", "u2", "u3", "u4", "u5")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	// 第 1 页，每页 2 条
	w := doGetMembersPaged(t, groupID, 1, 2)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["total"].(float64) != 5 {
		t.Errorf("期望 total=5，实际=%v", resp["total"])
	}
	members := resp["members"].([]interface{})
	if len(members) != 2 {
		t.Errorf("期望第 1 页返回 2 条，实际=%d", len(members))
	}
}

// 场景：第 2 页返回剩余成员
func TestAdminGetGroupMembers_Page2(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "u1", "u2", "u3", "u4", "u5")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	// 第 2 页，每页 2 条，应返回 2 条（第 3、4 条）
	w2 := doGetMembersPaged(t, groupID, 2, 2)
	var resp2 map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&resp2)
	if len(resp2["members"].([]interface{})) != 2 {
		t.Errorf("期望第 2 页返回 2 条，实际=%d", len(resp2["members"].([]interface{})))
	}

	// 第 3 页，每页 2 条，应返回 1 条（最后 1 条）
	w3 := doGetMembersPaged(t, groupID, 3, 2)
	var resp3 map[string]interface{}
	json.NewDecoder(w3.Body).Decode(&resp3)
	if len(resp3["members"].([]interface{})) != 1 {
		t.Errorf("期望第 3 页返回 1 条，实际=%d", len(resp3["members"].([]interface{})))
	}
}

// 场景：响应中包含 total 字段
func TestAdminGetGroupMembers_HasTotalField(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob", "carol")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	w := doGetMembers(t, groupID) // 不带分页参数，使用默认值
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if _, ok := resp["total"]; !ok {
		t.Error("期望响应中包含 total 字段")
	}
	if resp["total"].(float64) != 3 {
		t.Errorf("期望 total=3，实际=%v", resp["total"])
	}
}

// 场景：空组分页查询，total=0，members=[]
func TestAdminGetGroupMembers_EmptyGroupPaged(t *testing.T) {
	setupUserGroupTestDB(t)

	groupID := mustCreateGroup(t, "空组", "")

	w := doGetMembersPaged(t, groupID, 1, 20)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["total"].(float64) != 0 {
		t.Errorf("期望 total=0，实际=%v", resp["total"])
	}
	if len(resp["members"].([]interface{})) != 0 {
		t.Errorf("期望 members 为空数组，实际=%d 条", len(resp["members"].([]interface{})))
	}
}

// 场景：分页超出范围，返回空 members，total 不变
func TestAdminGetGroupMembers_PageOutOfRange(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	// 第 100 页，每页 20 条，超出范围
	w := doGetMembersPaged(t, groupID, 100, 20)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["total"].(float64) != 2 {
		t.Errorf("期望 total=2，实际=%v", resp["total"])
	}
	if len(resp["members"].([]interface{})) != 0 {
		t.Errorf("超出范围的页应返回空 members，实际=%d 条", len(resp["members"].([]interface{})))
	}
}

// ── GET /admin/user-groups/ungrouped-users ────────────────────────────────────

// 场景：所有用户都未分组，返回全部用户
func TestAdminGetUngroupedUsers_AllUngrouped(t *testing.T) {
	setupUserGroupTestDB(t)

	createTestUsers(t, "alice", "bob", "carol")

	w := doGetUngroupedUsers(t, 1, 20)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["total"].(float64) != 3 {
		t.Errorf("期望 total=3，实际=%v", resp["total"])
	}
	users := resp["users"].([]interface{})
	if len(users) != 3 {
		t.Errorf("期望 3 个未分组用户，实际=%d", len(users))
	}
}

// 场景：部分用户已分组，只返回未分组的用户
func TestAdminGetUngroupedUsers_PartialGrouped(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob", "carol", "dave")
	groupID := mustCreateGroup(t, "研发组", "")
	// alice 和 bob 加入用户组
	doAddMembers(t, groupID, userIDs[:2])

	w := doGetUngroupedUsers(t, 1, 20)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["total"].(float64) != 2 {
		t.Errorf("期望 total=2（carol+dave），实际=%v", resp["total"])
	}
	users := resp["users"].([]interface{})
	if len(users) != 2 {
		t.Errorf("期望 2 个未分组用户，实际=%d", len(users))
	}
	// 验证返回的是 carol 和 dave，不含 alice 和 bob
	names := make(map[string]bool)
	for _, u := range users {
		item := u.(map[string]interface{})
		names[item["Username"].(string)] = true
	}
	if names["alice"] || names["bob"] {
		t.Error("已分组的 alice/bob 不应出现在未分组列表中")
	}
	if !names["carol"] || !names["dave"] {
		t.Errorf("未分组的 carol/dave 应出现在列表中，实际=%v", names)
	}
}

// 场景：所有用户都已分组，返回空列表
func TestAdminGetUngroupedUsers_AllGrouped(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	w := doGetUngroupedUsers(t, 1, 20)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["total"].(float64) != 0 {
		t.Errorf("期望 total=0，实际=%v", resp["total"])
	}
	if len(resp["users"].([]interface{})) != 0 {
		t.Errorf("期望 users 为空数组，实际=%d 条", len(resp["users"].([]interface{})))
	}
}

// 场景：没有任何用户，返回空列表
func TestAdminGetUngroupedUsers_NoUsers(t *testing.T) {
	setupUserGroupTestDB(t)

	w := doGetUngroupedUsers(t, 1, 20)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["total"].(float64) != 0 {
		t.Errorf("期望 total=0，实际=%v", resp["total"])
	}
}

// 场景：分页查询未分组用户，total 正确，每页数量正确
func TestAdminGetUngroupedUsers_Pagination(t *testing.T) {
	setupUserGroupTestDB(t)

	createTestUsers(t, "u1", "u2", "u3", "u4", "u5")

	// 第 1 页，每页 2 条
	w := doGetUngroupedUsers(t, 1, 2)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["total"].(float64) != 5 {
		t.Errorf("期望 total=5，实际=%v", resp["total"])
	}
	if len(resp["users"].([]interface{})) != 2 {
		t.Errorf("期望第 1 页返回 2 条，实际=%d", len(resp["users"].([]interface{})))
	}
}

// 场景：响应中包含 ID、Username、CreatedAt 字段
func TestAdminGetUngroupedUsers_ResponseFields(t *testing.T) {
	setupUserGroupTestDB(t)

	createTestUsers(t, "alice")

	w := doGetUngroupedUsers(t, 1, 20)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	users := resp["users"].([]interface{})
	if len(users) == 0 {
		t.Fatal("期望至少 1 个未分组用户")
	}
	item := users[0].(map[string]interface{})
	if item["ID"] == nil {
		t.Error("期望 ID 字段存在")
	}
	if item["Username"] == nil {
		t.Error("期望 Username 字段存在")
	}
	if item["CreatedAt"] == nil {
		t.Error("期望 CreatedAt 字段存在")
	}
}

// 场景：用户被移出用户组后，重新出现在未分组列表中
func TestAdminGetUngroupedUsers_AfterRemoveMember(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	// 此时 alice 已分组，未分组列表应为空
	w1 := doGetUngroupedUsers(t, 1, 20)
	var resp1 map[string]interface{}
	json.NewDecoder(w1.Body).Decode(&resp1)
	if resp1["total"].(float64) != 0 {
		t.Errorf("alice 已分组，期望 total=0，实际=%v", resp1["total"])
	}

	// 将 alice 从组中移除
	doRemoveMembers(t, groupID, userIDs)

	// 移除后 alice 应重新出现在未分组列表中
	w2 := doGetUngroupedUsers(t, 1, 20)
	var resp2 map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&resp2)
	if resp2["total"].(float64) != 1 {
		t.Errorf("alice 被移出组后，期望 total=1，实际=%v", resp2["total"])
	}
}

// ── GET /admin/user-groups/groups-by-users ───────────────────────────────────

// 场景：批量查询，存量多组用户仍可正确返回 data map。
// 当前新增用户已不支持多组，本测试保留用于测试旧用户兼容。
func TestAdminGetGroupsByUsers_MultipleGroups(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupA := mustCreateGroup(t, "研发组", "研发部门")
	groupB := mustCreateGroup(t, "项目A组", "项目A")
	mustCreateGroup(t, "运营组", "") // alice 不在此组

	mustInsertLegacyGroupMembers(t,
		model.UserGroupMember{UserGroupID: groupA, UserID: userIDs[0]},
		model.UserGroupMember{UserGroupID: groupB, UserID: userIDs[0]},
		model.UserGroupMember{UserGroupID: groupB, UserID: userIDs[1]},
	)

	w := doGetGroupsByUsers(t, userIDs[0], userIDs[1])
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["ok"] != true {
		t.Errorf("期望 ok=true，实际=%v", resp["ok"])
	}
	data := resp["data"].(map[string]interface{})

	// alice 应属于研发组和项目A组
	aliceKey := fmt.Sprintf("%d", userIDs[0])
	aliceGroups := data[aliceKey].([]interface{})
	if len(aliceGroups) != 2 {
		t.Fatalf("期望 alice 属于 2 个组，实际=%d", len(aliceGroups))
	}
	groupNames := make(map[string]bool)
	for _, g := range aliceGroups {
		item := g.(map[string]interface{})
		groupNames[item["name"].(string)] = true
		if item["id"] == nil {
			t.Error("期望 id 字段存在")
		}
	}
	if !groupNames["研发组"] || !groupNames["项目A组"] {
		t.Errorf("期望包含研发组和项目A组，实际=%v", groupNames)
	}

	// bob 应只属于项目A组
	bobKey := fmt.Sprintf("%d", userIDs[1])
	bobGroups := data[bobKey].([]interface{})
	if len(bobGroups) != 1 {
		t.Fatalf("期望 bob 属于 1 个组，实际=%d", len(bobGroups))
	}
}

// 场景：用户不属于任何组，返回空数组
func TestAdminGetGroupsByUsers_NoGroups(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice")

	w := doGetGroupsByUsers(t, userIDs[0])
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	data := resp["data"].(map[string]interface{})
	aliceKey := fmt.Sprintf("%d", userIDs[0])
	aliceGroups := data[aliceKey].([]interface{})
	if len(aliceGroups) != 0 {
		t.Errorf("期望 alice 的组列表为空数组，实际=%d 个", len(aliceGroups))
	}
}

// 场景：user_ids 参数缺失，返回 400
func TestAdminGetGroupsByUsers_MissingUserIDs(t *testing.T) {
	setupUserGroupTestDB(t)

	req := adminReq(http.MethodGet, "/admin/user-groups/groups-by-users", nil)
	w := httptest.NewRecorder()
	HandleAdminGetGroupsByUsers(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// 场景：user_ids 中含格式错误的 ID，返回 400
func TestAdminGetGroupsByUsers_InvalidUserID(t *testing.T) {
	setupUserGroupTestDB(t)

	req := adminReq(http.MethodGet, "/admin/user-groups/groups-by-users?user_ids=abc", nil)
	w := httptest.NewRecorder()
	HandleAdminGetGroupsByUsers(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// 场景：user_ids 超过 100 个，返回 400
func TestAdminGetGroupsByUsers_TooManyUserIDs(t *testing.T) {
	setupUserGroupTestDB(t)

	parts := make([]string, 101)
	for i := range parts {
		parts[i] = fmt.Sprintf("%d", i+1)
	}
	path := "/admin/user-groups/groups-by-users?user_ids=" + strings.Join(parts, ",")
	req := adminReq(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	HandleAdminGetGroupsByUsers(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（超过 100 个），实际=%d", w.Code)
	}
}

// 场景：用户被移出组后，该组不再出现在响应中。
// 当前新增用户已不支持多组，本测试保留用于测试旧用户兼容。
func TestAdminGetGroupsByUsers_AfterRemove(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice")
	groupA := mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "项目组", "")
	mustInsertLegacyGroupMembers(t,
		model.UserGroupMember{UserGroupID: groupA, UserID: userIDs[0]},
		model.UserGroupMember{UserGroupID: groupB, UserID: userIDs[0]},
	)

	// 从研发组移除 alice
	doRemoveMembers(t, groupA, userIDs)

	w := doGetGroupsByUsers(t, userIDs[0])
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	data := resp["data"].(map[string]interface{})
	aliceKey := fmt.Sprintf("%d", userIDs[0])
	aliceGroups := data[aliceKey].([]interface{})
	if len(aliceGroups) != 1 {
		t.Fatalf("移除后期望 alice 只属于 1 个组，实际=%d", len(aliceGroups))
	}
	g := aliceGroups[0].(map[string]interface{})
	if g["name"].(string) != "项目组" {
		t.Errorf("期望看到项目组，实际=%v", g["name"])
	}
}

// ── HandleCreateUser + group_ids ─────────────────────────────────────────────

// 场景：创建用户时指定 1 个 group_id，用户自动加入对应用户组
func TestHandleCreateUser_WithGroupIDs(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	groupA := mustCreateGroup(t, "研发组", "")

	w := doCreateUserJSON(t, map[string]interface{}{
		"username":  "alice",
		"password":  "password123",
		"group_ids": []uint{groupA},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Errorf("期望 ok=true，实际=%v", resp["ok"])
	}
	userID := uint(resp["id"].(float64))

	// 验证用户已加入一个组
	groupIDs, err := model.GetUserGroupIDs(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserGroupIDs 失败: %v", err)
	}
	if len(groupIDs) != 1 || groupIDs[0] != groupA {
		t.Errorf("期望用户只属于研发组，实际=%v", groupIDs)
	}
}

// 场景：创建用户时指定多个 group_ids，返回 400
func TestHandleCreateUser_WithMultipleGroupIDs(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	groupA := mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "项目组", "")

	w := doCreateUserJSON(t, map[string]interface{}{
		"username":  "alice",
		"password":  "password123",
		"group_ids": []uint{groupA, groupB},
	})
	// v6.13：允许用户一次属于多个分组，新增用户时也生效
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（v6.13 允许多分组），实际=%d, body=%s", w.Code, w.Body.String())
	}
	// 解析 id 后验证归属
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	newID, ok := resp["id"].(float64)
	if !ok || newID <= 0 {
		t.Fatalf("响应应含 id 字段，实际=%v", resp)
	}
	gotIDs, err := model.GetUserGroupIDs(context.Background(), uint(newID))
	if err != nil {
		t.Fatalf("GetUserGroupIDs 失败: %v", err)
	}
	if len(gotIDs) != 2 {
		t.Fatalf("期望新用户属于 2 个组，实际=%d: %v", len(gotIDs), gotIDs)
	}
	set := map[uint]struct{}{gotIDs[0]: {}, gotIDs[1]: {}}
	if _, ok := set[groupA]; !ok {
		t.Errorf("应包含 研发组(id=%d)，实际=%v", groupA, gotIDs)
	}
	if _, ok := set[groupB]; !ok {
		t.Errorf("应包含 项目组(id=%d)，实际=%v", groupB, gotIDs)
	}
}

// 场景：创建用户时不传 group_ids，用户不属于任何组
func TestHandleCreateUser_WithoutGroupIDs(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	mustCreateGroup(t, "研发组", "")

	w := doCreateUserJSON(t, map[string]interface{}{
		"username": "alice",
		"password": "password123",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	userID := uint(resp["id"].(float64))

	groupIDs, _ := model.GetUserGroupIDs(context.Background(), userID)
	if len(groupIDs) != 0 {
		t.Errorf("不传 group_ids 时，用户不应属于任何组，实际=%d 个", len(groupIDs))
	}
}

// 场景：创建用户时传空数组 group_ids=[]，用户不属于任何组
func TestHandleCreateUser_WithEmptyGroupIDs(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	mustCreateGroup(t, "研发组", "")

	w := doCreateUserJSON(t, map[string]interface{}{
		"username":  "alice",
		"password":  "password123",
		"group_ids": []uint{},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	userID := uint(resp["id"].(float64))

	groupIDs, _ := model.GetUserGroupIDs(context.Background(), userID)
	if len(groupIDs) != 0 {
		t.Errorf("传空数组时，用户不应属于任何组，实际=%d 个", len(groupIDs))
	}
}

// 场景：创建用户时指定不存在的 group_id，返回错误（用户已创建但用户组归属设置失败）
func TestHandleCreateUser_WithInvalidGroupID(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	w := doCreateUserJSON(t, map[string]interface{}{
		"username":  "alice",
		"password":  "password123",
		"group_ids": []uint{99999},
	})
	// 用户已创建但用户组归属设置失败，返回 400
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（不合法的 group_id），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ── HandleUpdateUser + group_ids ─────────────────────────────────────────────

// 场景：修改用户时指定 group_ids，全量替换用户组归属
func TestHandleUpdateUser_WithGroupIDs(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	userIDs := createTestUsers(t, "alice")
	groupA := mustCreateGroup(t, "研发组", "")
	groupC := mustCreateGroup(t, "运营组", "")

	// 先将 alice 加入研发组
	doAddMembers(t, groupA, userIDs)

	// 修改用户，将 group_ids 改为只有运营组
	w := doUpdateUserJSON(t, userIDs[0], map[string]interface{}{
		"group_ids": []uint{groupC},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证 alice 现在只属于运营组
	groupIDs, err := model.GetUserGroupIDs(context.Background(), userIDs[0])
	if err != nil {
		t.Fatalf("GetUserGroupIDs 失败: %v", err)
	}
	if len(groupIDs) != 1 {
		t.Errorf("期望 alice 只属于 1 个组（运营组），实际=%d 个", len(groupIDs))
	}
	if groupIDs[0] != groupC {
		t.Errorf("期望 alice 属于运营组（id=%d），实际=%d", groupC, groupIDs[0])
	}
}

// 场景：修改用户时指定多个 group_ids，应允许（v6.13 起支持多分组归属）。
func TestHandleUpdateUser_WithMultipleGroupIDs(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	userIDs := createTestUsers(t, "alice")
	groupA := mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "项目组", "")

	w := doUpdateUserJSON(t, userIDs[0], map[string]interface{}{
		"group_ids": []uint{groupA, groupB},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（v6.13 允许多分组），实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证 alice 同时属于两个组
	gotIDs, err := model.GetUserGroupIDs(context.Background(), userIDs[0])
	if err != nil {
		t.Fatalf("GetUserGroupIDs 失败: %v", err)
	}
	if len(gotIDs) != 2 {
		t.Fatalf("期望 alice 属于 2 个组，实际=%d 个: %v", len(gotIDs), gotIDs)
	}
	gotSet := map[uint]struct{}{gotIDs[0]: {}, gotIDs[1]: {}}
	if _, ok := gotSet[groupA]; !ok {
		t.Errorf("alice 应包含 研发组(id=%d)，实际=%v", groupA, gotIDs)
	}
	if _, ok := gotSet[groupB]; !ok {
		t.Errorf("alice 应包含 项目组(id=%d)，实际=%v", groupB, gotIDs)
	}
}

// 场景：修改用户时传空数组 group_ids=[]，清除所有用户组归属
func TestHandleUpdateUser_WithEmptyGroupIDs(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	userIDs := createTestUsers(t, "alice")
	groupA := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupA, userIDs)

	// 传空数组，清除所有用户组归属
	w := doUpdateUserJSON(t, userIDs[0], map[string]interface{}{
		"group_ids": []uint{},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	groupIDs, _ := model.GetUserGroupIDs(context.Background(), userIDs[0])
	if len(groupIDs) != 0 {
		t.Errorf("传空数组后，alice 不应属于任何组，实际=%d 个", len(groupIDs))
	}
}

// 场景：修改用户时不传 group_ids，用户组归属不变
func TestHandleUpdateUser_WithoutGroupIDs(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	userIDs := createTestUsers(t, "alice")
	groupA := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupA, userIDs)

	// 只修改 email，不传 group_ids
	w := doUpdateUserJSON(t, userIDs[0], map[string]interface{}{
		"email": "alice@example.com",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 用户组归属应保持不变
	groupIDs, _ := model.GetUserGroupIDs(context.Background(), userIDs[0])
	if len(groupIDs) != 1 {
		t.Errorf("不传 group_ids 时，用户组归属应保持不变，实际=%d 个", len(groupIDs))
	}
}

// 场景：存量多组用户修改其他字段时，不传 group_ids，应保持原有多组归属且不报错。
// 当前新增用户已不支持多组，本测试保留用于测试旧用户兼容。
func TestHandleUpdateUser_WithoutGroupIDs_KeepsLegacyMultipleGroups(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	userIDs := createTestUsers(t, "alice")
	groupA := mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "项目组", "")
	mustInsertLegacyGroupMembers(t,
		model.UserGroupMember{UserGroupID: groupA, UserID: userIDs[0]},
		model.UserGroupMember{UserGroupID: groupB, UserID: userIDs[0]},
	)

	w := doUpdateUserJSON(t, userIDs[0], map[string]interface{}{
		"email": "alice@example.com",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	groupIDs, err := model.GetUserGroupIDs(context.Background(), userIDs[0])
	if err != nil {
		t.Fatalf("GetUserGroupIDs 失败: %v", err)
	}
	if len(groupIDs) != 2 {
		t.Fatalf("不传 group_ids 时，应保留存量多组归属，实际=%v", groupIDs)
	}

	var user model.User
	if err := model.DB(context.Background()).First(&user, userIDs[0]).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("期望 email=alice@example.com，实际=%s", user.Email)
	}
}

// 场景：修改用户时指定不存在的 group_id，返回 400
func TestHandleUpdateUser_WithInvalidGroupID(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	userIDs := createTestUsers(t, "alice")

	w := doUpdateUserJSON(t, userIDs[0], map[string]interface{}{
		"group_ids": []uint{99999},
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（不合法的 group_id），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// 场景：同时修改用户属性和用户组归属
func TestHandleUpdateUser_UpdateBothAttributesAndGroups(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	userIDs := createTestUsers(t, "alice")
	groupA := mustCreateGroup(t, "研发组", "")

	w := doUpdateUserJSON(t, userIDs[0], map[string]interface{}{
		"email":     "alice@example.com",
		"group_ids": []uint{groupA},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证用户组归属已更新
	groupIDs, _ := model.GetUserGroupIDs(context.Background(), userIDs[0])
	if len(groupIDs) != 1 || groupIDs[0] != groupA {
		t.Errorf("期望 alice 属于研发组，实际=%v", groupIDs)
	}

	// 验证 email 已更新
	var user model.User
	model.DB(context.Background()).First(&user, userIDs[0])
	if user.Email != "alice@example.com" {
		t.Errorf("期望 email=alice@example.com，实际=%s", user.Email)
	}
}

// ── HandleDeleteUser 解绑用户组 ───────────────────────────────────────────────

// 场景：硬删除用户时，自动解绑该用户的所有用户组成员关系
func TestHandleDeleteUser_UnbindsGroupMembership(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupA := mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "项目组", "")
	doAddMembers(t, groupA, userIDs[:1]) // alice 在研发组
	doAddMembers(t, groupB, userIDs[:1]) // alice 在项目组
	doAddMembers(t, groupA, userIDs[1:]) // bob 在研发组

	// 硬删除 alice（物理删除，同步清理成员记录）
	w := doHardDeleteUserJSON(t, userIDs[0])
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证 alice 的用户组成员记录已被物理删除
	var memberCount int64
	model.DB(context.Background()).Model(&model.UserGroupMember{}).Where("user_id = ?", userIDs[0]).Count(&memberCount)
	if memberCount != 0 {
		t.Errorf("删除用户后，期望用户组成员记录已清除，实际=%d 条", memberCount)
	}

	// 验证 bob 的成员关系不受影响
	var bobMemberCount int64
	model.DB(context.Background()).Model(&model.UserGroupMember{}).Where("user_id = ?", userIDs[1]).Count(&bobMemberCount)
	if bobMemberCount != 1 {
		t.Errorf("bob 的成员关系不应受影响，期望=1，实际=%d", bobMemberCount)
	}
}

// 场景：硬删除用户后，用户组的成员列表不再包含该用户
func TestHandleDeleteUser_MemberListUpdated(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	// 硬删除 alice（物理删除，成员记录同步清理）
	doHardDeleteUserJSON(t, userIDs[0])

	// 查询组内成员，不应包含 alice
	wMembers := doGetMembers(t, groupID)
	var resp map[string]interface{}
	json.NewDecoder(wMembers.Body).Decode(&resp)
	members := resp["members"].([]interface{})

	names := make(map[string]bool)
	for _, m := range members {
		item := m.(map[string]interface{})
		names[item["username"].(string)] = true
	}
	if names["alice"] {
		t.Error("删除 alice 后，成员列表不应包含 alice")
	}
	if !names["bob"] {
		t.Error("bob 应仍在成员列表中")
	}
}

// 场景：删除不属于任何组的用户，正常成功（无成员记录可删除）
func TestHandleDeleteUser_UserNotInAnyGroup(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	userIDs := createTestUsers(t, "alice")

	w := doDeleteUserJSON(t, userIDs[0])
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// 场景：删除用户后，未分组用户列表不再包含该用户
func TestHandleDeleteUser_RemovedFromUngroupedList(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	userIDs := createTestUsers(t, "alice", "bob")

	// 删除前，alice 和 bob 都在未分组列表中
	w1 := doGetUngroupedUsers(t, 1, 20)
	var resp1 map[string]interface{}
	json.NewDecoder(w1.Body).Decode(&resp1)
	if resp1["total"].(float64) != 2 {
		t.Errorf("删除前期望 total=2，实际=%v", resp1["total"])
	}

	// 禁用 alice（软删除）
	doDeleteUserJSON(t, userIDs[0])

	// 禁用后，未分组列表仍包含 alice（禁用用户仍可在用户组中展示和计数）
	w2 := doGetUngroupedUsers(t, 1, 20)
	var resp2 map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&resp2)
	if resp2["total"].(float64) != 2 {
		t.Errorf("禁用 alice 后期望 total=2（禁用用户仍计入未分组），实际=%v", resp2["total"])
	}
}

// ── 综合场景 ──────────────────────────────────────────────────────────────────

// 场景：完整流程：创建用户并加组 → 查询用户所在组 → 修改用户组归属 → 删除用户解绑
func TestUserGroupMembership_FullFlow(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	groupA := mustCreateGroup(t, "研发组", "")
	groupC := mustCreateGroup(t, "运营组", "")

	// 1. 创建用户时加入研发组
	w := doCreateUserJSON(t, map[string]interface{}{
		"username":  "alice",
		"password":  "password123",
		"group_ids": []uint{groupA},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("创建用户失败: status=%d, body=%s", w.Code, w.Body.String())
	}
	var createResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&createResp)
	userID := uint(createResp["id"].(float64))

	// 2. 查询 alice 所在的组，应只包含研发组
	wGroups := doGetGroupsByUsers(t, userID)
	var groupsResp map[string]interface{}
	json.NewDecoder(wGroups.Body).Decode(&groupsResp)
	data := groupsResp["data"].(map[string]interface{})
	aliceKey := fmt.Sprintf("%d", userID)
	groups := data[aliceKey].([]interface{})
	if len(groups) != 1 {
		t.Fatalf("期望 alice 属于 1 个组，实际=%d", len(groups))
	}

	// 3. 修改用户组归属：改为只属于运营组
	wUpdate := doUpdateUserJSON(t, userID, map[string]interface{}{
		"group_ids": []uint{groupC},
	})
	if wUpdate.Code != http.StatusOK {
		t.Fatalf("修改用户组归属失败: status=%d", wUpdate.Code)
	}

	// 4. 再次查询，应只属于运营组
	wGroups2 := doGetGroupsByUsers(t, userID)
	var groupsResp2 map[string]interface{}
	json.NewDecoder(wGroups2.Body).Decode(&groupsResp2)
	data2 := groupsResp2["data"].(map[string]interface{})
	groups2 := data2[aliceKey].([]interface{})
	if len(groups2) != 1 {
		t.Fatalf("修改后期望 alice 只属于 1 个组，实际=%d", len(groups2))
	}
	g := groups2[0].(map[string]interface{})
	if g["name"].(string) != "运营组" {
		t.Errorf("期望 alice 属于运营组，实际=%v", g["name"])
	}

	// 5. 硬删除用户，解绑用户组（物理删除，同步清理成员记录）
	wDelete := doHardDeleteUserJSON(t, userID)
	if wDelete.Code != http.StatusOK {
		t.Fatalf("删除用户失败: status=%d", wDelete.Code)
	}

	// 6. 验证成员记录已清除
	var memberCount int64
	model.DB(context.Background()).Model(&model.UserGroupMember{}).Where("user_id = ?", userID).Count(&memberCount)
	if memberCount != 0 {
		t.Errorf("删除用户后，期望成员记录已清除，实际=%d 条", memberCount)
	}
}

// 场景：修改用户组名称后，用户所在组的查询结果反映新名称
func TestAdminUpdateUserGroup_NameReflectedInUserQuery(t *testing.T) {
	setupUserGroupTestDB(t)

	userIDs := createTestUsers(t, "alice")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	// 修改用户组名称
	doUpdateGroup(t, groupID, "研发一组", "")

	// 查询 alice 所在的组，应看到新名称
	w := doGetGroupsByUsers(t, userIDs[0])
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	aliceKey := fmt.Sprintf("%d", userIDs[0])
	groups := data[aliceKey].([]interface{})
	if len(groups) != 1 {
		t.Fatalf("期望 1 个组，实际=%d", len(groups))
	}
	g := groups[0].(map[string]interface{})
	if g["name"].(string) != "研发一组" {
		t.Errorf("期望看到新名称研发一组，实际=%v", g["name"])
	}
}

// 场景：未分组用户加入组后，不再出现在未分组列表中；修改用户组归属为空后，重新出现
func TestAdminGetUngroupedUsers_DynamicMembership(t *testing.T) {
	setupUserGroupTestDBWithSiteConfig(t)

	userIDs := createTestUsers(t, "alice")
	groupID := mustCreateGroup(t, "研发组", "")

	// 初始：alice 未分组
	w1 := doGetUngroupedUsers(t, 1, 20)
	var resp1 map[string]interface{}
	json.NewDecoder(w1.Body).Decode(&resp1)
	if resp1["total"].(float64) != 1 {
		t.Errorf("初始期望 total=1，实际=%v", resp1["total"])
	}

	// 将 alice 加入研发组
	doAddMembers(t, groupID, userIDs)

	// 加入后：alice 不再是未分组用户
	w2 := doGetUngroupedUsers(t, 1, 20)
	var resp2 map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&resp2)
	if resp2["total"].(float64) != 0 {
		t.Errorf("加入组后期望 total=0，实际=%v", resp2["total"])
	}

	// 通过 UpdateUser 将 alice 的 group_ids 设为空
	doUpdateUserJSON(t, userIDs[0], map[string]interface{}{
		"group_ids": []uint{},
	})

	// 清空后：alice 重新出现在未分组列表中
	w3 := doGetUngroupedUsers(t, 1, 20)
	var resp3 map[string]interface{}
	json.NewDecoder(w3.Body).Decode(&resp3)
	if resp3["total"].(float64) != 1 {
		t.Errorf("清空用户组后期望 total=1，实际=%v", resp3["total"])
	}
}
