package controller

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initOneidSyncTestDB 初始化内存 SQLite，迁移 OneID 同步所需的表。
func initOneidSyncTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.SiteConfig{},
		&model.OneIDUserProfile{},
		&model.OneIDDepartmentRecord{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	db.Create(&model.SiteConfig{})
}

// ─── isOneIDUserDisabled 单元测试 ───────────────────────────────────────────

func TestIsOneIDUserDisabled(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{"Suspended", true},
		{"Disabled", true},
		{"LockedOut", true},
		{"Active", false},
		{"Inactive", false},
		{"", false},
		{"UnknownStatus", false},
	}
	for _, tc := range cases {
		got := isOneIDUserDisabled(tc.status)
		if got != tc.want {
			t.Errorf("isOneIDUserDisabled(%q)=%v, want %v", tc.status, got, tc.want)
		}
	}
}

// ─── ensureLocalUser 单元测试 ────────────────────────────────────────────────

func TestEnsureLocalUser_CreateNewUser(t *testing.T) {
	initOneidSyncTestDB(t)

	u := gwUserInfo{
		UnionID: "union-001",
		Name:    "张三",
		Email:   "zhangsan@test.com",
		Status:  "Active",
	}
	result := ensureLocalUser(context.Background(), u, "enterprise-1")
	if !result.Created {
		t.Fatal("期望创建新用户，但 Created=false")
	}
	if result.Disabled {
		t.Fatal("Active 用户不应被禁用")
	}

	// 验证数据库中确实存在该用户
	var user model.User
	if err := model.DB(context.Background()).Where("one_id_sub = ?", "union-001").First(&user).Error; err != nil {
		t.Fatalf("数据库中未找到新创建的用户: %v", err)
	}
	if user.Username != "张三" {
		t.Errorf("期望 username=张三，实际=%s", user.Username)
	}
}

func TestEnsureLocalUser_SkipCreateForDisabledUser(t *testing.T) {
	initOneidSyncTestDB(t)

	// OneID 中已停用的用户不应创建本地用户
	u := gwUserInfo{
		UnionID: "union-disabled-new",
		Name:    "被停用的新用户",
		Status:  "Suspended",
	}
	result := ensureLocalUser(context.Background(), u, "enterprise-1")
	if result.Created {
		t.Fatal("Suspended 状态的新用户不应被创建")
	}
	if result.Disabled {
		t.Fatal("未创建的用户不应标记为 Disabled")
	}

	// 数据库中不应存在
	var count int64
	model.DB(context.Background()).Model(&model.User{}).Where("one_id_sub = ?", "union-disabled-new").Count(&count)
	if count != 0 {
		t.Fatalf("Suspended 新用户不应被创建，但数据库中存在 %d 条记录", count)
	}
}

func TestEnsureLocalUser_DisableExistingUser(t *testing.T) {
	initOneidSyncTestDB(t)

	// 先创建一个正常用户
	sub := "union-to-disable"
	model.DB(context.Background()).Create(&model.User{
		Username: "正常用户",
		OneIDSub: &sub,
	})

	// 同步时发现该用户在 OneID 中已停用
	u := gwUserInfo{
		UnionID: "union-to-disable",
		Name:    "正常用户",
		Status:  "Suspended",
	}
	result := ensureLocalUser(context.Background(), u, "enterprise-1")
	if !result.Disabled {
		t.Fatal("Suspended 用户应被标记为 Disabled")
	}
	if result.UserID == 0 {
		t.Fatal("Disabled 用户的 UserID 应非零")
	}
	if result.Username == "" {
		t.Fatal("Disabled 用户的 Username 应非空")
	}

	// 验证 deleted_at 已设置
	var user model.User
	model.DB(context.Background()).Unscoped().Where("one_id_sub = ?", "union-to-disable").First(&user)
	if !user.DeletedAt.Valid {
		t.Fatal("被禁用的用户应设置 deleted_at")
	}
}

func TestEnsureLocalUser_AlreadyDisabledUser(t *testing.T) {
	initOneidSyncTestDB(t)

	// 创建一个已经被软删除的用户
	sub := "union-already-disabled"
	user := model.User{
		Username: "已禁用用户",
		OneIDSub: &sub,
	}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Delete(&user) // 软删除

	// 同步时 OneID 状态仍是 Suspended
	u := gwUserInfo{
		UnionID: "union-already-disabled",
		Name:    "已禁用用户",
		Status:  "Disabled",
	}
	result := ensureLocalUser(context.Background(), u, "enterprise-1")
	// 无论是新禁用还是已禁用，都应返回 Disabled=true
	if !result.Disabled {
		t.Fatal("已禁用的用户应返回 Disabled=true")
	}
	if result.UserID == 0 {
		t.Fatal("已禁用的用户 UserID 应非零")
	}
}

func TestEnsureLocalUser_ReEnableUser(t *testing.T) {
	initOneidSyncTestDB(t)

	// 创建一个已被软删除的用户
	sub := "union-re-enable"
	user := model.User{
		Username: "待恢复用户",
		OneIDSub: &sub,
	}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Delete(&user) // 软删除

	// 同步时发现 OneID 已恢复为 Active
	u := gwUserInfo{
		UnionID: "union-re-enable",
		Name:    "待恢复用户",
		Status:  "Active",
	}
	result := ensureLocalUser(context.Background(), u, "enterprise-1")
	if result.Disabled {
		t.Fatal("Active 用户不应被标记为 Disabled")
	}

	// 验证 deleted_at 已被清除
	var restored model.User
	model.DB(context.Background()).Where("one_id_sub = ?", "union-re-enable").First(&restored)
	if restored.DeletedAt.Valid {
		t.Fatal("恢复的用户 deleted_at 应为 null")
	}
}

func TestEnsureLocalUser_SyncUsername(t *testing.T) {
	initOneidSyncTestDB(t)

	// 创建用户，名字与 OneID 不同
	sub := "union-sync-name"
	model.DB(context.Background()).Create(&model.User{
		Username: "旧名字",
		OneIDSub: &sub,
	})

	u := gwUserInfo{
		UnionID: "union-sync-name",
		Name:    "新名字",
		Status:  "Active",
	}
	ensureLocalUser(context.Background(), u, "enterprise-1")

	var user model.User
	model.DB(context.Background()).Where("one_id_sub = ?", "union-sync-name").First(&user)
	if user.Username != "新名字" {
		t.Errorf("期望 username 同步为'新名字'，实际=%s", user.Username)
	}
}

func TestEnsureLocalUser_SkipSyncUsernameVariant(t *testing.T) {
	initOneidSyncTestDB(t)

	// 创建用户，名字是 OneID 姓名的变体（带后缀）
	sub := "union-variant"
	model.DB(context.Background()).Create(&model.User{
		Username: "李四_8779",
		OneIDSub: &sub,
	})

	u := gwUserInfo{
		UnionID: "union-variant",
		Name:    "李四",
		Status:  "Active",
	}
	ensureLocalUser(context.Background(), u, "enterprise-1")

	// 变体用户名不应被修改
	var user model.User
	model.DB(context.Background()).Where("one_id_sub = ?", "union-variant").First(&user)
	if user.Username != "李四_8779" {
		t.Errorf("变体用户名不应被修改，期望'李四_8779'，实际=%s", user.Username)
	}
}

func TestEnsureLocalUser_DisabledStatus(t *testing.T) {
	initOneidSyncTestDB(t)

	// "Disabled" 状态也应被视为禁用（新增的映射）
	sub := "union-disabled-status"
	model.DB(context.Background()).Create(&model.User{
		Username: "DisabledUser",
		OneIDSub: &sub,
	})

	u := gwUserInfo{
		UnionID: "union-disabled-status",
		Name:    "DisabledUser",
		Status:  "Disabled",
	}
	result := ensureLocalUser(context.Background(), u, "enterprise-1")
	if !result.Disabled {
		t.Fatal("Disabled 状态应被识别为禁用")
	}
}

func TestEnsureLocalUser_LockedOutStatus(t *testing.T) {
	initOneidSyncTestDB(t)

	sub := "union-lockedout"
	model.DB(context.Background()).Create(&model.User{
		Username: "LockedUser",
		OneIDSub: &sub,
	})

	u := gwUserInfo{
		UnionID: "union-lockedout",
		Name:    "LockedUser",
		Status:  "LockedOut",
	}
	result := ensureLocalUser(context.Background(), u, "enterprise-1")
	if !result.Disabled {
		t.Fatal("LockedOut 状态应被识别为禁用")
	}
}

// ─── ensureLocalUserResult 结构体字段测试 ─────────────────────────────────────

func TestEnsureLocalUserResult_HasUserIDAndUsername(t *testing.T) {
	initOneidSyncTestDB(t)

	sub := "union-result-fields"
	model.DB(context.Background()).Create(&model.User{
		Username: "结果字段测试",
		OneIDSub: &sub,
	})

	u := gwUserInfo{
		UnionID: "union-result-fields",
		Name:    "结果字段测试",
		Status:  "Suspended",
	}
	result := ensureLocalUser(context.Background(), u, "enterprise-1")
	if result.UserID == 0 {
		t.Error("UserID 应非零")
	}
	if result.Username != "结果字段测试" {
		t.Errorf("Username 期望'结果字段测试'，实际=%s", result.Username)
	}
}

// ─── tryHardDeleteUser 单元测试 ─────────────────────────────────────────────

func TestTryHardDeleteUser_NoResources(t *testing.T) {
	initOneidSyncTestDB(t)

	// 创建一个无实例、无 VPC 的用户
	user := model.User{Username: "no-resources", VpcId: ""}
	model.DB(context.Background()).Create(&user)

	ok := tryHardDeleteUser(context.Background(), &user)
	if !ok {
		t.Fatal("无资源用户应被硬删除，返回 true")
	}

	// 验证用户已被物理删除（Unscoped 也查不到）
	var count int64
	model.DB(context.Background()).Unscoped().Model(&model.User{}).Where("id = ?", user.ID).Count(&count)
	if count != 0 {
		t.Fatalf("硬删除后用户仍存在，count=%d", count)
	}
}

func TestTryHardDeleteUser_HasInstances(t *testing.T) {
	initOneidSyncTestDB(t)

	user := model.User{Username: "has-instances", VpcId: ""}
	model.DB(context.Background()).Create(&user)
	// 创建一个实例
	model.DB(context.Background()).Create(&model.Instance{UserID: user.ID, Name: "test-inst"})

	ok := tryHardDeleteUser(context.Background(), &user)
	if ok {
		t.Fatal("有实例的用户不应被硬删除")
	}

	// 验证用户仍然存在
	var count int64
	model.DB(context.Background()).Model(&model.User{}).Where("id = ?", user.ID).Count(&count)
	if count != 1 {
		t.Fatalf("用户应仍存在，count=%d", count)
	}
}

// ─── buildAffectedUser 单元测试 ─────────────────────────────────────────────

func TestBuildAffectedUser_NoInstances(t *testing.T) {
	initOneidSyncTestDB(t)

	user := model.User{Username: "affected-no-inst", VpcId: ""}
	model.DB(context.Background()).Create(&user)

	au := buildAffectedUser(context.Background(), &user, "disabled")
	if au.Username != "affected-no-inst" {
		t.Errorf("Username 期望 'affected-no-inst'，实际=%s", au.Username)
	}
	if au.InstanceCount != 0 {
		t.Errorf("InstanceCount 期望 0，实际=%d", au.InstanceCount)
	}
	if au.Action != "disabled" {
		t.Errorf("Action 期望 'disabled'，实际=%s", au.Action)
	}
	if au.VpcId != nil {
		t.Errorf("VpcId 期望 nil，实际=%v", au.VpcId)
	}
}

func TestBuildAffectedUser_WithInstances(t *testing.T) {
	initOneidSyncTestDB(t)

	user := model.User{Username: "affected-with-inst", VpcId: ""}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{UserID: user.ID, Name: "inst-1"})
	model.DB(context.Background()).Create(&model.Instance{UserID: user.ID, Name: "inst-2"})

	au := buildAffectedUser(context.Background(), &user, "hard_deleted")
	if au.InstanceCount != 2 {
		t.Errorf("InstanceCount 期望 2，实际=%d", au.InstanceCount)
	}
	if au.Action != "hard_deleted" {
		t.Errorf("Action 期望 'hard_deleted'，实际=%s", au.Action)
	}
}

// ─── syncResult / syncAffectedUser 结构体测试 ────────────────────────────────

func TestSyncResult_EmptyAffectedUsers(t *testing.T) {
	sr := &syncResult{}
	if sr.AffectedUsers != nil {
		t.Error("初始 AffectedUsers 应为 nil")
	}
}

func TestSyncResult_AppendAffectedUsers(t *testing.T) {
	sr := &syncResult{}
	sr.AffectedUsers = append(sr.AffectedUsers, syncAffectedUser{
		Username:      "test-user",
		InstanceCount: 3,
		Action:        "disabled",
	})
	if len(sr.AffectedUsers) != 1 {
		t.Fatalf("期望 1 条记录，实际=%d", len(sr.AffectedUsers))
	}
	if sr.AffectedUsers[0].Action != "disabled" {
		t.Errorf("Action 期望 'disabled'，实际=%s", sr.AffectedUsers[0].Action)
	}
}

func TestSyncAffectedUser_VpcFields(t *testing.T) {
	vpcId := "vpc-12345"
	au := syncAffectedUser{
		Username:        "vpc-user",
		InstanceCount:   0,
		Action:          "hard_deleted",
		VpcId:           &vpcId,
		VpcHasResources: false,
	}
	if au.VpcId == nil || *au.VpcId != "vpc-12345" {
		t.Errorf("VpcId 期望 vpc-12345，实际=%v", au.VpcId)
	}
	if au.VpcHasResources {
		t.Error("VpcHasResources 应为 false")
	}
}

// ─── cleanupOutOfScopeUsers 单元测试 ────────────────────────────────────────

func TestCleanupOutOfScopeUsers_SkipWhenSyncIncomplete(t *testing.T) {
	initOneidSyncTestDB(t)

	sub := "union-incomplete"
	model.DB(context.Background()).Create(&model.User{Username: "incomplete-user", OneIDSub: &sub})

	// syncIncomplete=true 时应跳过所有清理操作
	result := cleanupOutOfScopeUsers(context.Background(), map[string]bool{"other": true}, true)
	if result.disabled != 0 || result.hardDeleted != 0 || len(result.affectedUsers) != 0 {
		t.Fatal("syncIncomplete=true 时不应有任何清理操作")
	}

	// 验证用户仍存在
	var count int64
	model.DB(context.Background()).Model(&model.User{}).Where("one_id_sub = ?", sub).Count(&count)
	if count != 1 {
		t.Fatalf("用户应仍存在，count=%d", count)
	}
}

func TestCleanupOutOfScopeUsers_SkipWhenNoSyncedSubs(t *testing.T) {
	initOneidSyncTestDB(t)

	sub := "union-empty-sync"
	model.DB(context.Background()).Create(&model.User{Username: "empty-sync-user", OneIDSub: &sub})

	// syncedSubs 为空时应跳过
	result := cleanupOutOfScopeUsers(context.Background(), map[string]bool{}, false)
	if result.disabled != 0 || result.hardDeleted != 0 {
		t.Fatal("syncedSubs 为空时不应有清理操作")
	}
}

func TestCleanupOutOfScopeUsers_HardDeleteNoResources(t *testing.T) {
	initOneidSyncTestDB(t)

	sub1 := "union-in-scope"
	sub2 := "union-out-scope"
	model.DB(context.Background()).Create(&model.User{Username: "in-scope", OneIDSub: &sub1})
	model.DB(context.Background()).Create(&model.User{Username: "out-scope", OneIDSub: &sub2})

	// sub1 在同步集合中，sub2 不在 → sub2 应被硬删除
	result := cleanupOutOfScopeUsers(context.Background(), map[string]bool{sub1: true}, false)
	if result.hardDeleted != 1 {
		t.Errorf("期望硬删除 1 个用户，实际=%d", result.hardDeleted)
	}
	if len(result.affectedUsers) != 1 {
		t.Fatalf("期望 1 个 affectedUser，实际=%d", len(result.affectedUsers))
	}
	if result.affectedUsers[0].Action != "hard_deleted" {
		t.Errorf("Action 期望 hard_deleted，实际=%s", result.affectedUsers[0].Action)
	}

	// 验证 sub2 已被物理删除
	var count int64
	model.DB(context.Background()).Unscoped().Model(&model.User{}).Where("one_id_sub = ?", sub2).Count(&count)
	if count != 0 {
		t.Fatalf("不在范围的无资源用户应被硬删除，但仍存在 count=%d", count)
	}
}

func TestCleanupOutOfScopeUsers_SoftDeleteHasInstances(t *testing.T) {
	initOneidSyncTestDB(t)

	sub1 := "union-keep"
	sub2 := "union-has-inst"
	model.DB(context.Background()).Create(&model.User{Username: "keep-user", OneIDSub: &sub1})
	user2 := model.User{Username: "inst-user", OneIDSub: &sub2}
	model.DB(context.Background()).Create(&user2)
	model.DB(context.Background()).Create(&model.Instance{UserID: user2.ID, Name: "blocking-inst"})

	// sub2 不在同步集合中，有实例 → 应被软删除
	result := cleanupOutOfScopeUsers(context.Background(), map[string]bool{sub1: true}, false)
	if result.disabled != 1 {
		t.Errorf("期望软删除 1 个用户，实际=%d", result.disabled)
	}
	if result.hardDeleted != 0 {
		t.Errorf("有实例的用户不应被硬删除，hardDeleted=%d", result.hardDeleted)
	}

	// 验证用户被软删除
	var user model.User
	model.DB(context.Background()).Unscoped().Where("one_id_sub = ?", sub2).First(&user)
	if !user.DeletedAt.Valid {
		t.Fatal("有实例的用户应被软删除")
	}

	// affectedUsers 应包含 disabled 记录
	if len(result.affectedUsers) != 1 || result.affectedUsers[0].Action != "disabled" {
		t.Errorf("affectedUsers 应含 1 条 disabled 记录，实际=%+v", result.affectedUsers)
	}
}

func TestCleanupOutOfScopeUsers_AlreadySoftDeleted_NoResources(t *testing.T) {
	initOneidSyncTestDB(t)

	sub1 := "union-scope-ok"
	sub2 := "union-already-soft"
	model.DB(context.Background()).Create(&model.User{Username: "scope-ok", OneIDSub: &sub1})
	user2 := model.User{Username: "already-soft", OneIDSub: &sub2}
	model.DB(context.Background()).Create(&user2)
	model.DB(context.Background()).Delete(&user2) // 已经软删除，无实例

	// sub2 已软删除且无资源 → 应被硬删除（不受安全阈值限制）
	result := cleanupOutOfScopeUsers(context.Background(), map[string]bool{sub1: true}, false)
	if result.hardDeleted != 1 {
		t.Errorf("已软删除且无资源的用户应被硬删除，hardDeleted=%d", result.hardDeleted)
	}

	var count int64
	model.DB(context.Background()).Unscoped().Model(&model.User{}).Where("one_id_sub = ?", sub2).Count(&count)
	if count != 0 {
		t.Fatalf("已软删除且无资源的用户应被物理删除，count=%d", count)
	}
}

func TestCleanupOutOfScopeUsers_AlreadySoftDeleted_HasResources(t *testing.T) {
	initOneidSyncTestDB(t)

	sub1 := "union-scope-ok2"
	sub2 := "union-soft-with-inst"
	model.DB(context.Background()).Create(&model.User{Username: "scope-ok2", OneIDSub: &sub1})
	user2 := model.User{Username: "soft-with-inst", OneIDSub: &sub2}
	model.DB(context.Background()).Create(&user2)
	model.DB(context.Background()).Create(&model.Instance{UserID: user2.ID, Name: "inst-blocking"})
	model.DB(context.Background()).Delete(&user2) // 已经软删除，但有实例

	// sub2 已软删除但有资源 → 维持禁用状态
	result := cleanupOutOfScopeUsers(context.Background(), map[string]bool{sub1: true}, false)
	if result.hardDeleted != 0 {
		t.Errorf("已软删除但有资源的用户不应被硬删除，hardDeleted=%d", result.hardDeleted)
	}
	if len(result.affectedUsers) != 1 || result.affectedUsers[0].Action != "disabled" {
		t.Errorf("已软删除有资源用户应报告 disabled，实际=%+v", result.affectedUsers)
	}
}

func TestCleanupOutOfScopeUsers_MajorityRemoved(t *testing.T) {
	initOneidSyncTestDB(t)

	// 3 个用户，只有 1 个在同步集合 → 2/3 被清理，无安全阈值阻挡
	sub1 := "union-stay"
	sub2 := "union-go-1"
	sub3 := "union-go-2"
	model.DB(context.Background()).Create(&model.User{Username: "stay", OneIDSub: &sub1})
	model.DB(context.Background()).Create(&model.User{Username: "go1", OneIDSub: &sub2})
	model.DB(context.Background()).Create(&model.User{Username: "go2", OneIDSub: &sub3})

	result := cleanupOutOfScopeUsers(context.Background(), map[string]bool{sub1: true}, false)
	if result.hardDeleted != 2 {
		t.Errorf("期望硬删除 2 个无资源用户，实际=%d", result.hardDeleted)
	}

	// sub1 仍在
	var count int64
	model.DB(context.Background()).Model(&model.User{}).Where("one_id_sub = ?", sub1).Count(&count)
	if count != 1 {
		t.Fatal("在范围内的用户不应被删除")
	}
}

func TestCleanupOutOfScopeUsers_InScopeUsersUntouched(t *testing.T) {
	initOneidSyncTestDB(t)

	sub1 := "union-untouched-1"
	sub2 := "union-untouched-2"
	model.DB(context.Background()).Create(&model.User{Username: "untouched1", OneIDSub: &sub1})
	model.DB(context.Background()).Create(&model.User{Username: "untouched2", OneIDSub: &sub2})

	// 两个都在同步集合中 → 都不应被处理
	result := cleanupOutOfScopeUsers(context.Background(), map[string]bool{sub1: true, sub2: true}, false)
	if result.disabled != 0 || result.hardDeleted != 0 || len(result.affectedUsers) != 0 {
		t.Fatal("在范围内的用户不应被处理")
	}
}
