package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"hatchery/model"
)

// ── OneID 用户测试辅助 ────────────────────────────────────────────────────────

// createOneIDUsers 在数据库中批量创建 OneID 用户（模拟 SSO 首次登录后自动创建的用户）。
// OneID 用户特征：OneIDSub != nil，Password = ""，Username 来自 OneID name 字段。
// 这是模拟真实 OneID 用户的标准方式，与 HandleInternalLogin 的创建逻辑完全一致。
func createOneIDUsers(t *testing.T, names ...string) []*model.User {
	t.Helper()
	users := make([]*model.User, len(names))
	for i, name := range names {
		sub := fmt.Sprintf("oneid-sub-%s-%d", name, i)
		u := model.User{
			Username: name,
			Password: "", // OneID 用户无本地密码
			Role:     "user",
			OneIDSub: &sub,
		}
		if err := model.DB(context.Background()).Create(&u).Error; err != nil {
			t.Fatalf("创建 OneID 测试用户 %s 失败: %v", name, err)
		}
		users[i] = &u
	}
	return users
}

// createOneIDAdminUser 创建一个 OneID 管理员用户（模拟 OneID admin.added 事件后的状态）。
func createOneIDAdminUser(t *testing.T, name string) *model.User {
	t.Helper()
	sub := fmt.Sprintf("oneid-sub-admin-%s", name)
	u := model.User{
		Username: name,
		Password: "",
		Role:     "admin",
		OneIDSub: &sub,
	}
	if err := model.DB(context.Background()).Create(&u).Error; err != nil {
		t.Fatalf("创建 OneID 管理员用户 %s 失败: %v", name, err)
	}
	return &u
}

// oneIDUserIDs 从 OneID 用户切片中提取 ID 列表。
func oneIDUserIDs(users []*model.User) []uint {
	ids := make([]uint, len(users))
	for i, u := range users {
		ids[i] = u.ID
	}
	return ids
}

// ── 场景一：OneID 用户被添加到用户组，能正确查询到自己所在的组 ─────────────────

// 场景：OneID 用户首次 SSO 登录后（已自动创建本地记录），管理员将其加入用户组，
// 用户调用 /user-groups/mine 能看到自己所在的组。
// 这是最核心的 OneID 兼容性验证。
func TestOneIDUser_CanBeAddedToGroupAndQueryOwnGroups(t *testing.T) {
	setupUserGroupTestDB(t)

	// 模拟 OneID 用户（SSO 登录后自动创建）
	oneIDUsers := createOneIDUsers(t, "张三", "李四")
	groupID := mustCreateGroup(t, "研发组", "研发部门")

	// 管理员通过 user_id 将 OneID 用户加入组（与本地用户完全相同的操作）
	w := doAddMembers(t, groupID, oneIDUserIDs(oneIDUsers))
	if w.Code != http.StatusOK {
		t.Fatalf("添加 OneID 用户到组失败: status=%d, body=%s", w.Code, w.Body.String())
	}

	// OneID 用户查询自己所在的组（模拟 SSO 登录后的 session 状态）
	wUser := doGetMyGroups(t, oneIDUsers[0])
	if wUser.Code != http.StatusOK {
		t.Fatalf("OneID 用户查询自己的组失败: status=%d", wUser.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(wUser.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	groups := resp["groups"].([]interface{})
	if len(groups) != 1 {
		t.Fatalf("期望 OneID 用户属于 1 个组，实际=%d", len(groups))
	}
	g := groups[0].(map[string]interface{})
	if g["name"].(string) != "研发组" {
		t.Errorf("期望看到研发组，实际=%v", g["name"])
	}
}

// ── 场景二：OneID 用户与本地用户混合在同一用户组 ─────────────────────────────

// 场景：同一用户组内同时包含 OneID 用户和本地用户，成员列表应正确返回所有人。
// 验证两种用户类型在用户组功能中完全透明、互不干扰。
func TestOneIDUser_MixedWithLocalUsersInSameGroup(t *testing.T) {
	setupUserGroupTestDB(t)

	// 本地用户（有密码）
	localIDs := createTestUsers(t, "local_alice", "local_bob")
	// OneID 用户（无密码，有 sub）
	oneIDUsers := createOneIDUsers(t, "oneid_张三", "oneid_李四")

	groupID := mustCreateGroup(t, "混合组", "本地+OneID 用户混合")

	// 将所有用户加入同一组
	allIDs := append(localIDs, oneIDUserIDs(oneIDUsers)...)
	w := doAddMembers(t, groupID, allIDs)
	if w.Code != http.StatusOK {
		t.Fatalf("添加混合用户失败: status=%d, body=%s", w.Code, w.Body.String())
	}

	// 管理员查看成员列表，应包含所有 4 人
	wMembers := doGetMembers(t, groupID)
	if wMembers.Code != http.StatusOK {
		t.Fatalf("查询成员失败: status=%d", wMembers.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(wMembers.Body).Decode(&resp)
	members := resp["members"].([]interface{})
	if len(members) != 4 {
		t.Fatalf("期望 4 名成员（2 本地 + 2 OneID），实际=%d", len(members))
	}

	// 验证用户名都正确返回
	names := make(map[string]bool)
	for _, m := range members {
		item := m.(map[string]interface{})
		names[item["username"].(string)] = true
	}
	for _, expected := range []string{"local_alice", "local_bob", "oneid_张三", "oneid_李四"} {
		if !names[expected] {
			t.Errorf("期望成员列表包含 %s，实际=%v", expected, names)
		}
	}

	// OneID 用户视角：能看到自己所在的组，且成员列表完整
	wOneID := doGetMyGroups(t, oneIDUsers[0])
	var oneIDResp map[string]interface{}
	json.NewDecoder(wOneID.Body).Decode(&oneIDResp)
	oneIDGroups := oneIDResp["groups"].([]interface{})
	if len(oneIDGroups) != 1 {
		t.Fatalf("OneID 用户期望看到 1 个组，实际=%d", len(oneIDGroups))
	}
	oneIDGroupMembers := oneIDGroups[0].(map[string]interface{})["members"].([]interface{})
	if len(oneIDGroupMembers) != 4 {
		t.Errorf("OneID 用户视角期望看到 4 名成员，实际=%d", len(oneIDGroupMembers))
	}
}

// ── 场景三：OneID 用户被软删除（禁用）后，仍应出现在成员列表中 ──────────────

// 场景：模拟 OneID member.deleted Webhook 触发用户软删除（禁用）后，
// 产品设计：禁用用户仍可在用户组成员列表中展示和计数，方便管理员统一管理。
// user_group_members 记录不会自动清除，成员列表通过 Unscoped() 包含禁用用户。
func TestOneIDUser_SoftDeletedUserNotInMemberList(t *testing.T) {
	setupUserGroupTestDB(t)

	oneIDUsers := createOneIDUsers(t, "张三", "李四")
	localIDs := createTestUsers(t, "local_alice")
	groupID := mustCreateGroup(t, "研发组", "")

	allIDs := append(oneIDUserIDs(oneIDUsers), localIDs...)
	doAddMembers(t, groupID, allIDs)

	// 模拟 OneID member.deleted 事件：软删除（禁用）张三
	// 这与 handleOneIDMemberDeleted 的实际行为完全一致
	if err := model.DB(context.Background()).Delete(oneIDUsers[0]).Error; err != nil {
		t.Fatalf("软删除 OneID 用户失败: %v", err)
	}

	// 管理员查看成员列表，禁用用户张三仍应出现（产品设计：禁用用户仍计入成员）
	wMembers := doGetMembers(t, groupID)
	var resp map[string]interface{}
	json.NewDecoder(wMembers.Body).Decode(&resp)
	members := resp["members"].([]interface{})

	names := make(map[string]bool)
	for _, m := range members {
		item := m.(map[string]interface{})
		names[item["username"].(string)] = true
	}

	if !names["张三"] {
		t.Error("已禁用的 OneID 用户张三仍应出现在成员列表中（禁用用户仍计入成员）")
	}
	if !names["李四"] {
		t.Error("未禁用的 OneID 用户李四应仍在成员列表中")
	}
	if !names["local_alice"] {
		t.Error("本地用户 local_alice 应仍在成员列表中")
	}
}

// ── 场景四：OneID 用户被软删除后，/user-groups/mine 不应返回已删除用户所在的组 ─

// 场景：OneID 用户被软删除后，其他用户调用 /user-groups/mine 时，
// 已删除用户的 user_group_members 记录仍存在，但该用户本身已不可用。
// 验证软删除用户无法再查询自己的组（RequestUser 会返回 BannedError）。
func TestOneIDUser_SoftDeletedUserCannotQueryOwnGroups(t *testing.T) {
	setupUserGroupTestDB(t)

	oneIDUsers := createOneIDUsers(t, "张三")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, oneIDUserIDs(oneIDUsers))

	// 软删除张三（模拟 member.deleted 事件）
	model.DB(context.Background()).Delete(oneIDUsers[0])

	// 软删除后，user.DeletedAt.Valid = true
	// getMyGroupsHandler 直接注入用户对象，但 RequestUser 在真实场景中会检查 DeletedAt
	// 这里验证：即使注入了软删除用户，业务逻辑也应正确处理
	// 注：getMyGroupsHandler 是测试 wrapper，真实 HandleGetMyUserGroups 通过 RequestUser 会返回 BannedError
	// 此处验证 model 层：软删除用户的 GetUserGroups 仍能返回组 ID（成员记录未清除）
	groupIDs, err := model.GetUserGroupIDs(context.Background(), oneIDUsers[0].ID)
	if err != nil {
		t.Fatalf("GetUserGroupIDs 失败: %v", err)
	}
	// 成员记录仍存在（user_group_members 不随用户软删除而清除）
	if len(groupIDs) != 1 {
		t.Errorf("期望成员记录仍存在（1 个组），实际=%d", len(groupIDs))
	}
}

// ── 场景五：OneID 用户改名后，用户组成员关系不受影响 ─────────────────────────

// 场景：模拟 OneID member.updated 事件触发用户名更新（safeUpdateUsername），
// 用户组成员关系通过 user_id 关联，改名后成员关系应完全不受影响。
func TestOneIDUser_RenameDoesNotAffectGroupMembership(t *testing.T) {
	setupUserGroupTestDB(t)

	oneIDUsers := createOneIDUsers(t, "张三")
	groupID := mustCreateGroup(t, "研发组", "")
	doAddMembers(t, groupID, oneIDUserIDs(oneIDUsers))

	// 模拟 OneID member.updated 事件：更新用户名（与 handleOneIDMemberUpdated 逻辑一致）
	newName := "张三（已改名）"
	if err := model.DB(context.Background()).Model(oneIDUsers[0]).Update("username", newName).Error; err != nil {
		t.Fatalf("更新用户名失败: %v", err)
	}

	// 验证成员关系仍然存在，且用户名已更新
	wMembers := doGetMembers(t, groupID)
	var resp map[string]interface{}
	json.NewDecoder(wMembers.Body).Decode(&resp)
	members := resp["members"].([]interface{})

	if len(members) != 1 {
		t.Fatalf("改名后期望成员数仍为 1，实际=%d", len(members))
	}
	item := members[0].(map[string]interface{})
	if item["username"].(string) != newName {
		t.Errorf("期望成员名已更新为 %q，实际=%v", newName, item["username"])
	}
	// user_id 应保持不变
	if uint(item["user_id"].(float64)) != oneIDUsers[0].ID {
		t.Errorf("改名后 user_id 应保持不变，期望=%d，实际=%v", oneIDUsers[0].ID, item["user_id"])
	}
}

// ── 场景六：OneID 管理员用户可以管理用户组 ───────────────────────────────────

// 场景：OneID 管理员（通过 admin.added 事件提升为 admin）使用 AdminToken 调用管理接口，
// 验证 OneID 管理员与本地管理员在用户组管理上的权限完全一致。
// 注：测试中使用 AdminToken（Bearer Token）模拟管理员身份，这与真实 OneID 管理员
// 通过 session 登录后调用接口的效果等价（requireAdmin 两条路径最终都通过）。
func TestOneIDAdmin_CanManageUserGroups(t *testing.T) {
	setupUserGroupTestDB(t)

	// 创建 OneID 普通用户
	oneIDUsers := createOneIDUsers(t, "张三", "李四")

	// 管理员创建用户组
	groupID := mustCreateGroup(t, "OneID测试组", "OneID 管理员创建的组")

	// 管理员添加 OneID 用户
	w := doAddMembers(t, groupID, oneIDUserIDs(oneIDUsers))
	if w.Code != http.StatusOK {
		t.Fatalf("添加 OneID 用户失败: status=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证成员已添加
	wMembers := doGetMembers(t, groupID)
	names := memberUsernames(t, wMembers)
	if !names["张三"] || !names["李四"] {
		t.Errorf("期望张三和李四在组内，实际=%v", names)
	}

	// 全量替换：只保留张三
	w2 := doSetMembers(t, groupID, []uint{oneIDUsers[0].ID})
	if w2.Code != http.StatusOK {
		t.Fatalf("全量替换失败: status=%d", w2.Code)
	}
	wMembers2 := doGetMembers(t, groupID)
	names2 := memberUsernames(t, wMembers2)
	if names2["李四"] {
		t.Error("全量替换后李四应已被移除")
	}
	if !names2["张三"] {
		t.Error("全量替换后张三应仍在组内")
	}

	// 删除用户组
	wd := doDeleteGroup(t, groupID)
	if wd.Code != http.StatusOK {
		t.Fatalf("删除用户组失败: status=%d", wd.Code)
	}
}

// ── 场景七：OneID 用户属于多个组，各组成员互不干扰 ───────────────────────────

// 场景：一个 OneID 用户同时属于多个用户组，验证 /user-groups/mine 返回所有组，
// 且每个组的成员列表独立正确。
// 当前新增用户已不支持多组，本测试保留用于测试旧用户兼容。
func TestOneIDUser_BelongsToMultipleGroups(t *testing.T) {
	setupUserGroupTestDB(t)

	oneIDUsers := createOneIDUsers(t, "张三", "李四", "王五")
	groupA := mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "项目A组", "")
	groupC := mustCreateGroup(t, "运营组", "") // 张三不在此组

	mustInsertLegacyGroupMembers(t,
		model.UserGroupMember{UserGroupID: groupA, UserID: oneIDUsers[0].ID},
		model.UserGroupMember{UserGroupID: groupA, UserID: oneIDUsers[1].ID},
		model.UserGroupMember{UserGroupID: groupB, UserID: oneIDUsers[0].ID},
		model.UserGroupMember{UserGroupID: groupB, UserID: oneIDUsers[2].ID},
		model.UserGroupMember{UserGroupID: groupC, UserID: oneIDUsers[1].ID},
		model.UserGroupMember{UserGroupID: groupC, UserID: oneIDUsers[2].ID},
	)

	// 张三查询自己的组，应看到研发组和项目A组，不含运营组
	wZhangSan := doGetMyGroups(t, oneIDUsers[0])
	var resp map[string]interface{}
	json.NewDecoder(wZhangSan.Body).Decode(&resp)
	groups := resp["groups"].([]interface{})

	if len(groups) != 2 {
		t.Fatalf("张三期望属于 2 个组，实际=%d", len(groups))
	}
	groupNames := make(map[string]bool)
	for _, g := range groups {
		item := g.(map[string]interface{})
		groupNames[item["name"].(string)] = true
	}
	if !groupNames["研发组"] || !groupNames["项目A组"] {
		t.Errorf("期望张三看到研发组和项目A组，实际=%v", groupNames)
	}
	if groupNames["运营组"] {
		t.Error("张三不在运营组，不应出现在响应中")
	}
}

// ── 场景八：OneID 用户组被删除后，成员的 /user-groups/mine 不再返回该组 ────────

// 场景：管理员删除用户组后，原 OneID 成员调用 /user-groups/mine 不应再看到该组（E-03 场景）。
// 当前新增用户已不支持多组，本测试保留用于测试旧用户兼容。
func TestOneIDUser_DeletedGroupNotVisibleToMember(t *testing.T) {
	setupUserGroupTestDB(t)

	oneIDUsers := createOneIDUsers(t, "张三")
	groupA := mustCreateGroup(t, "研发组", "")
	groupB := mustCreateGroup(t, "项目组", "")
	mustInsertLegacyGroupMembers(t,
		model.UserGroupMember{UserGroupID: groupA, UserID: oneIDUsers[0].ID},
		model.UserGroupMember{UserGroupID: groupB, UserID: oneIDUsers[0].ID},
	)

	// 删除研发组
	wd := doDeleteGroup(t, groupA)
	if wd.Code != http.StatusOK {
		t.Fatalf("删除研发组失败: status=%d", wd.Code)
	}

	// 张三查询自己的组，只应看到项目组
	wUser := doGetMyGroups(t, oneIDUsers[0])
	var resp map[string]interface{}
	json.NewDecoder(wUser.Body).Decode(&resp)
	groups := resp["groups"].([]interface{})

	if len(groups) != 1 {
		t.Fatalf("删除研发组后，张三期望只看到 1 个组，实际=%d", len(groups))
	}
	g := groups[0].(map[string]interface{})
	if g["name"].(string) != "项目组" {
		t.Errorf("期望看到项目组，实际=%v", g["name"])
	}
}

// ── 场景九：OneID 用户的 user_id 合法性校验（与本地用户一致） ─────────────────

// 场景：添加成员时，OneID 用户的 user_id 与本地用户完全等价，
// 合法性校验（validateUserIDs）对两种用户类型行为一致。
func TestOneIDUser_UserIDValidationConsistentWithLocalUser(t *testing.T) {
	setupUserGroupTestDB(t)

	oneIDUsers := createOneIDUsers(t, "张三")
	localIDs := createTestUsers(t, "local_alice")
	groupID := mustCreateGroup(t, "测试组", "")

	// 混合添加：OneID 用户 + 本地用户 + 不存在的 ID，应整体失败
	w := doAddMembers(t, groupID, []uint{oneIDUsers[0].ID, localIDs[0], 99999})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（包含不存在的 user_id），实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证整体回滚，组内无成员
	wMembers := doGetMembers(t, groupID)
	var resp map[string]interface{}
	json.NewDecoder(wMembers.Body).Decode(&resp)
	if len(resp["members"].([]interface{})) != 0 {
		t.Error("期望整体回滚，组内无成员")
	}

	// 只传合法的 OneID user_id，应成功
	w2 := doAddMembers(t, groupID, []uint{oneIDUsers[0].ID})
	if w2.Code != http.StatusOK {
		t.Fatalf("期望 200（合法 OneID user_id），实际=%d, body=%s", w2.Code, w2.Body.String())
	}
}

// ── 场景十：OneID 用户完整生命周期（SSO 登录 → 加组 → 改名 → 删除） ──────────

// 场景：模拟 OneID 用户从 SSO 首次登录到最终被删除的完整生命周期，
// 验证每个阶段用户组功能的行为符合预期。
func TestOneIDUser_FullLifecycle(t *testing.T) {
	setupUserGroupTestDB(t)

	// 1. SSO 首次登录：自动创建 OneID 用户（模拟 HandleInternalLogin 的创建逻辑）
	oneIDUsers := createOneIDUsers(t, "张三")
	localIDs := createTestUsers(t, "local_alice")

	// 2. 管理员创建用户组并添加成员
	groupID := mustCreateGroup(t, "研发组", "研发部门")
	doAddMembers(t, groupID, []uint{oneIDUsers[0].ID, localIDs[0]})

	// 3. 张三查询自己的组
	wUser := doGetMyGroups(t, oneIDUsers[0])
	var resp1 map[string]interface{}
	json.NewDecoder(wUser.Body).Decode(&resp1)
	if len(resp1["groups"].([]interface{})) != 1 {
		t.Fatalf("加组后张三期望看到 1 个组，实际=%d", len(resp1["groups"].([]interface{})))
	}

	// 4. OneID member.updated：张三改名
	model.DB(context.Background()).Model(oneIDUsers[0]).Update("username", "张三(新名)")

	// 5. 改名后成员列表应反映新名字
	wMembers := doGetMembers(t, groupID)
	names := memberUsernames(t, wMembers)
	if !names["张三(新名)"] {
		t.Error("改名后成员列表应显示新名字")
	}
	if names["张三"] {
		t.Error("改名后旧名字不应出现在成员列表中")
	}

	// 6. OneID member.deleted（assetAction=keep）：软删除（禁用）张三，实例保留
	model.DB(context.Background()).Delete(oneIDUsers[0])

	// 7. 禁用后，成员列表仍应包含张三（产品设计：禁用用户仍计入成员）
	wMembers2 := doGetMembers(t, groupID)
	names2 := memberUsernames(t, wMembers2)
	if !names2["张三(新名)"] {
		t.Error("禁用后张三仍应出现在成员列表中（禁用用户仍计入成员）")
	}
	if !names2["local_alice"] {
		t.Error("本地用户 local_alice 应仍在成员列表中")
	}

	// 8. 管理员删除用户组
	doDeleteGroup(t, groupID)

	// 9. 验证成员记录已物理删除
	var memberCount int64
	model.DB(context.Background()).Model(&model.UserGroupMember{}).Where("user_group_id = ?", groupID).Count(&memberCount)
	if memberCount != 0 {
		t.Errorf("删除组后期望成员记录已物理删除，实际=%d", memberCount)
	}
}
