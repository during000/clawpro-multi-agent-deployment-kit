package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initInstancesByUserGroupTestDB 准备包含 user_groups / closure / members / instances
// 四张表的内存 DB，并打开 admin token 鉴权旁路。
func initInstancesByUserGroupTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.GroupClosure{},
		&model.Instance{},
		&model.SiteConfig{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	unlock := model.UseDBForTest(db)
	db.Create(&model.SiteConfig{})
	AdminToken = "test-admin-token"
	t.Cleanup(func() { unlock() })
}

// seedClosureSelf 为 closure 表写入自指行。测试里手工摆 closure 比走
// model.ClosureXxx 一套 API 更可控。
func seedClosureSelf(t *testing.T, id uint) {
	t.Helper()
	if err := model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: id, DescendantID: id, Depth: 0}).Error; err != nil {
		t.Fatalf("seed closure self(%d): %v", id, err)
	}
}

// seedClosureEdge 在 closure 表上写入 (ancestor → descendant, depth) 行。
func seedClosureEdge(t *testing.T, ancestor, descendant uint, depth int) {
	t.Helper()
	if err := model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: ancestor, DescendantID: descendant, Depth: depth}).Error; err != nil {
		t.Fatalf("seed closure edge(%d->%d,depth=%d): %v", ancestor, descendant, depth, err)
	}
}

// mustSeedInstance 插入一条 instance 行，返回其 ID。
func mustSeedInstance(t *testing.T, name, cvmID string, userID, groupID uint) uint {
	t.Helper()
	inst := model.Instance{
		Name:       name,
		InstanceId: cvmID,
		UserID:     userID,
		GroupID:    groupID,
	}
	if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
		t.Fatalf("create instance %s: %v", name, err)
	}
	return inst.ID
}

// mustSeedMember 插入一条 user_group_members 行。
func mustSeedMember(t *testing.T, groupID, userID uint) {
	t.Helper()
	if err := model.DB(context.Background()).Create(&model.UserGroupMember{
		UserGroupID: groupID, UserID: userID, Source: model.MemberSourceManual,
	}).Error; err != nil {
		t.Fatalf("seed member g=%d u=%d: %v", groupID, userID, err)
	}
}

// callByUserGroup 发起请求并解析响应。返回 status + instances 列表。
func callByUserGroup(t *testing.T, body map[string]interface{}) (int, []map[string]interface{}) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/by-user-group", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminInstancesByUserGroup(w, req)

	if w.Code != http.StatusOK {
		return w.Code, nil
	}
	var resp struct {
		OK        bool                     `json:"ok"`
		Instances []map[string]interface{} `json:"instances"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode resp: %v, body=%s", err, w.Body.String())
	}
	return w.Code, resp.Instances
}

// TestInstancesByUserGroup_BothConditions 复现需求里的例子：
//
//	group1(root, user1,user2)
//	 ├ group2(user1,user3)
//	 └ group3(user4)
//	    ├ group4(user5,user6)
//	    └ group5(user6)
//
// 查询：{"user_group_ids": [{1,1},{1,2}], "group_ids": [3]}
// 命中 pair：(1,1),(1,2),(4,3),(5,4),(6,4),(6,5)
func TestInstancesByUserGroup_BothConditions(t *testing.T) {
	initInstancesByUserGroupTestDB(t)

	// 5 个分组 + closure
	for _, gid := range []uint{1, 2, 3, 4, 5} {
		seedClosureSelf(t, gid)
	}
	// 层级: 1→2,3；3→4,5
	seedClosureEdge(t, 1, 2, 1)
	seedClosureEdge(t, 1, 3, 1)
	seedClosureEdge(t, 1, 4, 2)
	seedClosureEdge(t, 1, 5, 2)
	seedClosureEdge(t, 3, 4, 1)
	seedClosureEdge(t, 3, 5, 1)

	// 成员
	mustSeedMember(t, 1, 1)
	mustSeedMember(t, 1, 2)
	mustSeedMember(t, 2, 1)
	mustSeedMember(t, 2, 3)
	mustSeedMember(t, 3, 4)
	mustSeedMember(t, 4, 5)
	mustSeedMember(t, 4, 6)
	mustSeedMember(t, 5, 6)

	// 为每个"命中"的 (user, group) 对各插 1 条 instance；再插几条不该命中的干扰数据
	wantPairs := map[[2]uint]string{
		{1, 1}: "inst-u1-g1",
		{1, 2}: "inst-u1-g2",
		{4, 3}: "inst-u4-g3",
		{5, 4}: "inst-u5-g4",
		{6, 4}: "inst-u6-g4",
		{6, 5}: "inst-u6-g5",
	}
	for p, name := range wantPairs {
		mustSeedInstance(t, name, "ins-"+name, p[0], p[1])
	}
	// 干扰：user2 在 group1 里，但未被查询条件指定 user_id → 不该命中
	mustSeedInstance(t, "noise-u2-g1", "ins-noise-u2-g1", 2, 1)
	// 干扰：user3 在 group2 里，但条件只指定 user=1 + group=2 → user3 不应命中 (3,2)
	mustSeedInstance(t, "noise-u3-g2", "ins-noise-u3-g2", 3, 2)
	// group_ids=3 的子树包含 group 3,4,5，所以 user5 在 group5 的实例也会命中
	//（group_ids 不依赖 user_group_members，按 group_id IN subtree 查所有实例）
	mustSeedInstance(t, "noise-u5-g5", "ins-noise-u5-g5", 5, 5)

	status, items := callByUserGroup(t, map[string]interface{}{
		"user_group_ids": []map[string]uint{{"user_id": 1, "group_id": 1}, {"user_id": 1, "group_id": 2}},
		"group_ids":      []uint{3},
	})
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	if len(items) != len(wantPairs)+1 {
		// wantPairs 的 6 条 + noise-u5-g5（group_ids=3 子树含 group 5，按 group_id IN subtree 命中）
		t.Fatalf("期望 %d 条 instance，实际 %d: %+v", len(wantPairs)+1, len(items), items)
	}
	// 按 name 映射核对 (user, group) 对
	gotNames := map[string]struct{}{}
	for _, it := range items {
		name, _ := it["name"].(string)
		gotNames[name] = struct{}{}
		// 每条必须带齐字段
		for _, key := range []string{"id", "instance_id", "name", "user_id", "user_name", "group_id", "group_full_path", "status", "created_at"} {
			if _, ok := it[key]; !ok {
				t.Errorf("响应缺字段 %q: %+v", key, it)
			}
		}
	}
	for _, name := range wantPairs {
		if _, ok := gotNames[name]; !ok {
			t.Errorf("缺少命中项: %s", name)
		}
	}
	// 干扰项必须都不在（noise-u5-g5 现在会命中：group_ids=3 子树含 group 5，
	// 按 group_id IN subtree 查所有实例，不依赖 user_group_members）
	for _, bad := range []string{"noise-u2-g1", "noise-u3-g2"} {
		if _, ok := gotNames[bad]; ok {
			t.Errorf("误命中干扰项: %s", bad)
		}
	}
	// noise-u5-g5 应命中（group 5 在 group 3 的子树中）
	if _, ok := gotNames["noise-u5-g5"]; !ok {
		t.Errorf("应命中 noise-u5-g5（group 5 在 group_ids=3 子树中）")
	}
}

// TestInstancesByUserGroup_OnlyUserGroupIDs 只传条件 1（user_group_ids）。
func TestInstancesByUserGroup_OnlyUserGroupIDs(t *testing.T) {
	initInstancesByUserGroupTestDB(t)
	seedClosureSelf(t, 1)
	mustSeedInstance(t, "hit", "ins-hit", 1, 1)
	mustSeedInstance(t, "miss", "ins-miss", 2, 1)

	status, items := callByUserGroup(t, map[string]interface{}{
		"user_group_ids": []map[string]uint{{"user_id": 1, "group_id": 1}},
	})
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	if len(items) != 1 || items[0]["name"] != "hit" {
		t.Fatalf("期望 1 条 hit，实际 %+v", items)
	}
}

// TestInstancesByUserGroup_OnlyGroupIDs 只传条件 2。
// group_ids 子树场景：返回 group_id IN subtree 的所有实例，不依赖 user_group_members。
func TestInstancesByUserGroup_OnlyGroupIDs(t *testing.T) {
	initInstancesByUserGroupTestDB(t)
	// 1→2
	for _, gid := range []uint{1, 2} {
		seedClosureSelf(t, gid)
	}
	seedClosureEdge(t, 1, 2, 1)
	mustSeedMember(t, 1, 10)
	mustSeedMember(t, 2, 20)
	mustSeedInstance(t, "u10-g1", "ins-u10-g1", 10, 1) // 命中（group_id=1 在子树中）
	mustSeedInstance(t, "u20-g2", "ins-u20-g2", 20, 2) // 命中（group_id=2 在子树中）
	mustSeedInstance(t, "u10-g2", "ins-u10-g2", 10, 2) // 命中（group_id=2 在子树中，u10 是否是 g2 成员不影响）
	mustSeedInstance(t, "u20-g1", "ins-u20-g1", 20, 1) // 命中（group_id=1 在子树中，u20 是否是 g1 成员不影响）

	status, items := callByUserGroup(t, map[string]interface{}{
		"group_ids": []uint{1},
	})
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	got := map[string]struct{}{}
	for _, it := range items {
		got[it["name"].(string)] = struct{}{}
	}
	for _, want := range []string{"u10-g1", "u20-g2", "u10-g2", "u20-g1"} {
		if _, ok := got[want]; !ok {
			t.Errorf("应命中 %s，实际 %v", want, got)
		}
	}
	if len(items) != 4 {
		t.Errorf("期望 4 条，实际 %d: %v", len(items), got)
	}
}

// TestInstancesByUserGroup_Empty 两者都不传 → 400。
func TestInstancesByUserGroup_Empty(t *testing.T) {
	initInstancesByUserGroupTestDB(t)
	status, _ := callByUserGroup(t, map[string]interface{}{})
	if status != http.StatusBadRequest {
		t.Fatalf("期望 400，实际 %d", status)
	}
}

// TestInstancesByUserGroup_NoHits 条件正确但没命中任何 instance 时返回空数组（非 nil）。
func TestInstancesByUserGroup_NoHits(t *testing.T) {
	initInstancesByUserGroupTestDB(t)
	seedClosureSelf(t, 1)

	status, items := callByUserGroup(t, map[string]interface{}{
		"user_group_ids": []map[string]uint{{"user_id": 99, "group_id": 1}},
	})
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	if items == nil {
		t.Fatal("期望空数组，不是 nil")
	}
	if len(items) != 0 {
		t.Fatalf("期望 0 条，实际 %+v", items)
	}
}

// TestInstancesByUserGroup_UnionDedup 条件 1 和条件 2 展开有重复对时不应出现重复结果。
func TestInstancesByUserGroup_UnionDedup(t *testing.T) {
	initInstancesByUserGroupTestDB(t)
	seedClosureSelf(t, 1)
	mustSeedMember(t, 1, 7)
	mustSeedInstance(t, "only-one", "ins-only-one", 7, 1)

	status, items := callByUserGroup(t, map[string]interface{}{
		"user_group_ids": []map[string]uint{{"user_id": 7, "group_id": 1}}, // 与 group_ids 展开出的 (7,1) 重复
		"group_ids":      []uint{1},
	})
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	if len(items) != 1 {
		t.Fatalf("期望 1 条（不去重会变 2），实际 %d: %+v", len(items), items)
	}
}

// TestInstancesByUserGroup_CreatedAtFormat created_at 字段是 UTC RFC3339 串。
func TestInstancesByUserGroup_CreatedAtFormat(t *testing.T) {
	initInstancesByUserGroupTestDB(t)
	seedClosureSelf(t, 1)
	mustSeedInstance(t, "x", "ins-x", 1, 1)

	_, items := callByUserGroup(t, map[string]interface{}{
		"user_group_ids": []map[string]uint{{"user_id": 1, "group_id": 1}},
	})
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %+v", items)
	}
	s, ok := items[0]["created_at"].(string)
	if !ok || s == "" {
		t.Fatalf("created_at 应为非空字符串: %+v", items[0])
	}
	if _, err := time.Parse("2006-01-02T15:04:05Z", s); err != nil {
		t.Fatalf("created_at 解析失败 (%q): %v", s, err)
	}
}

// TestInstancesByUserGroup_GroupFullPath 验证响应里 group_full_path 字段
// 能正确回填所属分组的 full_path；group_id=0（未指定分组）的理论上不会命中 pair
// 查询，但单独验证 fetchGroupFullPathMap 对 0 不产生错误键即可。
func TestInstancesByUserGroup_GroupFullPath(t *testing.T) {
	initInstancesByUserGroupTestDB(t)

	// 真实 UserGroup 行（带 full_path），closure 只需自指以便 pair 查询
	g1 := model.UserGroup{Name: "研发中心", FullPath: "研发中心", Source: model.GroupSourceManual}
	g2 := model.UserGroup{Name: "后端组", FullPath: "研发中心/后端组", Source: model.GroupSourceManual}
	if err := model.DB(context.Background()).Create(&g1).Error; err != nil {
		t.Fatalf("create g1: %v", err)
	}
	if err := model.DB(context.Background()).Create(&g2).Error; err != nil {
		t.Fatalf("create g2: %v", err)
	}
	seedClosureSelf(t, g1.ID)
	seedClosureSelf(t, g2.ID)

	mustSeedInstance(t, "inst-a", "ins-a", 1, g1.ID)
	mustSeedInstance(t, "inst-b", "ins-b", 1, g2.ID)

	status, items := callByUserGroup(t, map[string]interface{}{
		"user_group_ids": []map[string]uint{
			{"user_id": 1, "group_id": g1.ID},
			{"user_id": 1, "group_id": g2.ID},
		},
	})
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	if len(items) != 2 {
		t.Fatalf("期望 2 条命中，实际 %+v", items)
	}
	byName := map[string]map[string]interface{}{}
	for _, it := range items {
		byName[it["name"].(string)] = it
	}
	if got := byName["inst-a"]["group_full_path"]; got != "研发中心" {
		t.Errorf("inst-a group_full_path 期望 '研发中心'，实际 %v", got)
	}
	if got := byName["inst-b"]["group_full_path"]; got != "研发中心/后端组" {
		t.Errorf("inst-b group_full_path 期望 '研发中心/后端组'，实际 %v", got)
	}
}

// TestInstancesByUserGroup_UserName 验证响应里每条实例都带 user_name（所属用户的用户名）。
// 同时覆盖"用户已被软删"的场景：Unscoped 查询应仍然能拿到 username（历史 agent 不应显示空）。
func TestInstancesByUserGroup_UserName(t *testing.T) {
	initInstancesByUserGroupTestDB(t)

	alice := model.User{Username: "alice", Role: "user"}
	bob := model.User{Username: "bob", Role: "user"}
	if err := model.DB(context.Background()).Create(&alice).Error; err != nil {
		t.Fatalf("create alice: %v", err)
	}
	if err := model.DB(context.Background()).Create(&bob).Error; err != nil {
		t.Fatalf("create bob: %v", err)
	}
	// 软删 bob，验证 Unscoped 仍能读到 username
	if err := model.DB(context.Background()).Delete(&bob).Error; err != nil {
		t.Fatalf("soft delete bob: %v", err)
	}

	g := model.UserGroup{Name: "研发组", FullPath: "研发组", Source: model.GroupSourceManual}
	if err := model.DB(context.Background()).Create(&g).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	seedClosureSelf(t, g.ID)

	mustSeedInstance(t, "inst-alice", "ins-a", alice.ID, g.ID)
	mustSeedInstance(t, "inst-bob-deleted-user", "ins-b", bob.ID, g.ID)

	status, items := callByUserGroup(t, map[string]interface{}{
		"user_group_ids": []map[string]uint{
			{"user_id": alice.ID, "group_id": g.ID},
			{"user_id": bob.ID, "group_id": g.ID},
		},
	})
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	if len(items) != 2 {
		t.Fatalf("期望 2 条，实际 %+v", items)
	}
	byName := map[string]map[string]interface{}{}
	for _, it := range items {
		byName[it["name"].(string)] = it
	}

	a := byName["inst-alice"]
	if a == nil {
		t.Fatalf("缺少 inst-alice: %+v", items)
	}
	if got := a["user_id"].(float64); uint(got) != alice.ID {
		t.Errorf("inst-alice user_id 期望 %d，实际 %v", alice.ID, got)
	}
	if got := a["user_name"]; got != "alice" {
		t.Errorf("inst-alice user_name 期望 alice，实际 %v", got)
	}

	b := byName["inst-bob-deleted-user"]
	if b == nil {
		t.Fatalf("缺少 inst-bob-deleted-user: %+v", items)
	}
	if got := b["user_id"].(float64); uint(got) != bob.ID {
		t.Errorf("bob user_id 期望 %d，实际 %v", bob.ID, got)
	}
	// 软删用户仍应能读出 username（Unscoped 查询）
	if got := b["user_name"]; got != "bob" {
		t.Errorf("软删用户 user_name 期望 bob，实际 %v", got)
	}
}
