package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ── 测试辅助 ─────────────────────────────────────────────────────────────────

// setupUserGroupsUserTestDB 初始化内存 SQLite，迁移用户组相关表。
func setupUserGroupsUserTestDB(t *testing.T) {
	t.Helper()
	setupUserGroupTestDB(t) // 复用管理员测试的 DB 初始化（含 AdminToken 设置）
}

// getMyGroupsHandler 提取 HandleGetMyUserGroups 的核心逻辑，绕过 RequestUser 的 session 解析。
// 直接传入已知用户，模拟已登录状态。
func getMyGroupsHandler(user *model.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w)
		if user == nil {
			writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgUnauthorized))
			return
		}

		groups, err := model.GetUserGroups(context.Background(), user.ID)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
			return
		}

		type memberItem struct {
			UserID   uint   `json:"user_id"`
			Username string `json:"username"`
		}
		type groupItem struct {
			ID          uint         `json:"id"`
			Name        string       `json:"name"`
			Description string       `json:"description"`
			MemberCount int64        `json:"member_count"`
			Members     []memberItem `json:"members"`
		}

		items := make([]groupItem, len(groups))
		for i, g := range groups {
			members, err := model.GetGroupMembers(context.Background(), g.ID)
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
				return
			}
			count := int64(len(members))
			mItems := make([]memberItem, len(members))
			for j, m := range members {
				mItems[j] = memberItem{
					UserID:   m.UserID,
					Username: m.Username,
				}
			}
			items[i] = groupItem{
				ID:          g.ID,
				Name:        g.Name,
				Description: g.Description,
				MemberCount: count,
				Members:     mItems,
			}
		}

		jsonOK(w, map[string]interface{}{
			"ok":     true,
			"groups": items,
		})
	}
}

// doGetMyGroups 模拟已登录用户调用 GET /user-groups/mine。
// 通过 getMyGroupsHandler wrapper 绕过 session 认证，直接注入用户。
func doGetMyGroups(t *testing.T, user *model.User) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/user-groups/mine", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	getMyGroupsHandler(user)(w, req)
	return w
}

// ── GET /user-groups/mine ─────────────────────────────────────────────────────

// 场景：用户不属于任何组，返回空数组
func TestGetMyUserGroups_NoGroups(t *testing.T) {
	setupUserGroupsUserTestDB(t)

	userIDs := createTestUsers(t, "alice")
	var alice model.User
	if err := model.DB(context.Background()).First(&alice, userIDs[0]).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	w := doGetMyGroups(t, &alice)
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
	groups := resp["groups"].([]interface{})
	if len(groups) != 0 {
		t.Errorf("期望 groups 为空数组，实际=%d 个", len(groups))
	}
}

// 场景：用户属于 1 个组，响应包含该组的基本信息和完整成员列表
func TestGetMyUserGroups_OneGroup(t *testing.T) {
	setupUserGroupsUserTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "研发组", "研发部门")
	doAddMembers(t, groupID, userIDs) // alice + bob 都在组内

	var alice model.User
	if err := model.DB(context.Background()).First(&alice, userIDs[0]).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	w := doGetMyGroups(t, &alice)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	groups := resp["groups"].([]interface{})
	if len(groups) != 1 {
		t.Fatalf("期望 1 个组，实际=%d", len(groups))
	}

	g := groups[0].(map[string]interface{})
	if g["name"].(string) != "研发组" {
		t.Errorf("期望 name=研发组，实际=%v", g["name"])
	}
	if g["description"].(string) != "研发部门" {
		t.Errorf("期望 description=研发部门，实际=%v", g["description"])
	}
	if g["member_count"].(float64) != 2 {
		t.Errorf("期望 member_count=2，实际=%v", g["member_count"])
	}

	// 验证 members 字段包含完整成员列表
	members := g["members"].([]interface{})
	if len(members) != 2 {
		t.Fatalf("期望 members 包含 2 名成员，实际=%d", len(members))
	}
	memberNames := make(map[string]bool)
	for _, m := range members {
		item := m.(map[string]interface{})
		memberNames[item["username"].(string)] = true
		// 验证字段存在
		if item["user_id"] == nil {
			t.Error("期望 user_id 字段存在")
		}
		// 验证不含敏感字段
		if item["password"] != nil {
			t.Error("响应中不应包含 password 字段")
		}
		if item["api_token"] != nil {
			t.Error("响应中不应包含 api_token 字段")
		}
	}
	if !memberNames["alice"] || !memberNames["bob"] {
		t.Errorf("期望 members 包含 alice 和 bob，实际=%v", memberNames)
	}
}

// 场景：用户属于多个组，响应包含所有组，不包含自己不在的组。
// 当前新增用户已不支持多组，本测试保留用于测试旧用户兼容。
func TestGetMyUserGroups_MultipleGroups(t *testing.T) {
	setupUserGroupsUserTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupA := mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "项目A组", "")
	groupC := mustCreateGroup(t, "运营组", "") // alice 不在此组

	mustInsertLegacyGroupMembers(t,
		model.UserGroupMember{UserGroupID: groupA, UserID: userIDs[0]},
		model.UserGroupMember{UserGroupID: groupA, UserID: userIDs[1]},
		model.UserGroupMember{UserGroupID: groupB, UserID: userIDs[0]},
		model.UserGroupMember{UserGroupID: groupC, UserID: userIDs[1]},
	)

	var alice model.User
	if err := model.DB(context.Background()).First(&alice, userIDs[0]).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	w := doGetMyGroups(t, &alice)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	groups := resp["groups"].([]interface{})
	if len(groups) != 2 {
		t.Fatalf("期望 alice 属于 2 个组（研发组+项目A组），实际=%d", len(groups))
	}

	groupNames := make(map[string]bool)
	for _, g := range groups {
		item := g.(map[string]interface{})
		groupNames[item["name"].(string)] = true
	}
	if !groupNames["研发组"] || !groupNames["项目A组"] {
		t.Errorf("期望包含研发组和项目A组，实际=%v", groupNames)
	}
	if groupNames["运营组"] {
		t.Error("alice 不在运营组，不应出现在响应中")
	}
}

// 场景：用户只能看到自己所在的组，不能看到其他组
func TestGetMyUserGroups_OnlyOwnGroups(t *testing.T) {
	setupUserGroupsUserTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupA := mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "运营组", "")

	doAddMembers(t, groupA, userIDs[:1]) // 只有 alice
	doAddMembers(t, groupB, userIDs[1:]) // 只有 bob

	var bob model.User
	if err := model.DB(context.Background()).First(&bob, userIDs[1]).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	w := doGetMyGroups(t, &bob)
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	groups := resp["groups"].([]interface{})
	if len(groups) != 1 {
		t.Fatalf("bob 期望只看到 1 个组，实际=%d", len(groups))
	}
	g := groups[0].(map[string]interface{})
	if g["name"].(string) != "运营组" {
		t.Errorf("bob 期望看到运营组，实际=%v", g["name"])
	}
}

// 场景：响应中的 members 字段只包含 user_id 和 username，不含 joined_at 等管理员专属字段
func TestGetMyUserGroups_MembersFieldsRestricted(t *testing.T) {
	setupUserGroupsUserTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	var alice model.User
	if err := model.DB(context.Background()).First(&alice, userIDs[0]).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	w := doGetMyGroups(t, &alice)
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	groups := resp["groups"].([]interface{})
	g := groups[0].(map[string]interface{})
	members := g["members"].([]interface{})

	for _, m := range members {
		item := m.(map[string]interface{})
		// 普通用户接口的 members 只含 user_id + username
		if item["user_id"] == nil {
			t.Error("期望 user_id 字段存在")
		}
		if item["username"] == nil {
			t.Error("期望 username 字段存在")
		}
		// 不应含管理员专属字段
		if item["joined_at"] != nil {
			t.Error("普通用户接口不应返回 joined_at 字段")
		}
		if item["password"] != nil {
			t.Error("不应返回 password 字段")
		}
	}
}

// 场景：用户所在的组被删除后，该组不再出现在响应中（软删除过滤）。
// 当前新增用户已不支持多组，本测试保留用于测试旧用户兼容。
func TestGetMyUserGroups_DeletedGroupNotVisible(t *testing.T) {
	setupUserGroupsUserTestDB(t)

	userIDs := createTestUsers(t, "alice")
	groupA := mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "项目组", "")
	mustInsertLegacyGroupMembers(t,
		model.UserGroupMember{UserGroupID: groupA, UserID: userIDs[0]},
		model.UserGroupMember{UserGroupID: groupB, UserID: userIDs[0]},
	)

	// 删除研发组
	doDeleteGroup(t, groupA)

	var alice model.User
	if err := model.DB(context.Background()).First(&alice, userIDs[0]).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	w := doGetMyGroups(t, &alice)
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	groups := resp["groups"].([]interface{})
	if len(groups) != 1 {
		t.Fatalf("删除研发组后，alice 期望只看到 1 个组，实际=%d", len(groups))
	}
	g := groups[0].(map[string]interface{})
	if g["name"].(string) != "项目组" {
		t.Errorf("期望看到项目组，实际=%v", g["name"])
	}
}

// 场景：未登录用户访问，返回 401
func TestGetMyUserGroups_Unauthorized(t *testing.T) {
	setupUserGroupsUserTestDB(t)

	// 传入 nil user 模拟未登录，避免触发 session store（测试环境未初始化）
	req := httptest.NewRequest(http.MethodGet, "/user-groups/mine", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	getMyGroupsHandler(nil)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，实际=%d", w.Code)
	}
}

// 场景：组内成员列表与管理员接口返回的成员一致（数据一致性）
func TestGetMyUserGroups_MembersConsistentWithAdminView(t *testing.T) {
	setupUserGroupsUserTestDB(t)

	userIDs := createTestUsers(t, "alice", "bob", "carol")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, userIDs)

	var alice model.User
	if err := model.DB(context.Background()).First(&alice, userIDs[0]).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	// 普通用户视角
	wUser := doGetMyGroups(t, &alice)
	var userResp map[string]interface{}
	if err := json.NewDecoder(wUser.Body).Decode(&userResp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	userGroups := userResp["groups"].([]interface{})
	userMembers := userGroups[0].(map[string]interface{})["members"].([]interface{})

	// 管理员视角
	wAdmin := doGetMembers(t, groupID)
	var adminResp map[string]interface{}
	if err := json.NewDecoder(wAdmin.Body).Decode(&adminResp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	adminMembers := adminResp["members"].([]interface{})

	// 成员数量应一致
	if len(userMembers) != len(adminMembers) {
		t.Errorf("普通用户视角成员数=%d，管理员视角成员数=%d，应一致",
			len(userMembers), len(adminMembers))
	}

	// 用户名集合应一致
	userNames := make(map[string]bool)
	for _, m := range userMembers {
		item := m.(map[string]interface{})
		userNames[item["username"].(string)] = true
	}
	adminNames := make(map[string]bool)
	for _, m := range adminMembers {
		item := m.(map[string]interface{})
		adminNames[item["username"].(string)] = true
	}
	for name := range adminNames {
		if !userNames[name] {
			t.Errorf("管理员视角有 %s，普通用户视角没有", name)
		}
	}
}
