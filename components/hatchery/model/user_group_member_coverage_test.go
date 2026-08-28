package model

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupMemberCoverageDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&UserGroup{}, &GroupClosure{}, &UserGroupMember{}, &GroupConfigBinding{}, &User{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	gdb = db
	t.Cleanup(func() { gdb = nil })
	return db
}

func createMemberTestGroup(t *testing.T, name string) *UserGroup {
	t.Helper()
	g, err := CreateUserGroupWithOpts(context.Background(), name, "", 0, GroupSourceManual, "")
	if err != nil {
		t.Fatalf("create group %s: %v", name, err)
	}
	return g
}

func createMemberTestUser(t *testing.T, username string) *User {
	t.Helper()
	u := &User{Username: username, Password: "test"}
	if err := gdb.Create(u).Error; err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	return u
}

// ── GetGroupMembersPaged ─────────────────────────────────

func TestCoverageGetGroupMembersPaged_Basic(t *testing.T) {
	setupMemberCoverageDB(t)

	g := createMemberTestGroup(t, "分页组")
	u1 := createMemberTestUser(t, "paged1")
	u2 := createMemberTestUser(t, "paged2")
	u3 := createMemberTestUser(t, "paged3")

	gdb.Create(&UserGroupMember{UserGroupID: g.ID, UserID: u1.ID, Source: MemberSourceManual})
	gdb.Create(&UserGroupMember{UserGroupID: g.ID, UserID: u2.ID, Source: MemberSourceManual})
	gdb.Create(&UserGroupMember{UserGroupID: g.ID, UserID: u3.ID, Source: MemberSourceManual})

	members, total, err := GetGroupMembersPaged(context.Background(), g.ID, 1, 2)
	if err != nil {
		t.Fatalf("GetGroupMembersPaged: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 on page 1, got %d", len(members))
	}

	members2, _, _ := GetGroupMembersPaged(context.Background(), g.ID, 2, 2)
	if len(members2) != 1 {
		t.Errorf("expected 1 on page 2, got %d", len(members2))
	}
}

func TestCoverageGetGroupMembersPaged_Empty(t *testing.T) {
	setupMemberCoverageDB(t)

	g := createMemberTestGroup(t, "空组")

	members, total, err := GetGroupMembersPaged(context.Background(), g.ID, 1, 10)
	if err != nil {
		t.Fatalf("GetGroupMembersPaged empty: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 total, got %d", total)
	}
	if len(members) != 0 {
		t.Errorf("expected 0 members, got %d", len(members))
	}
}

// ── GetGroupMembersByGroupIDs ────────────────────────────

func TestCoverageGetGroupMembersByGroupIDs(t *testing.T) {
	setupMemberCoverageDB(t)

	g1 := createMemberTestGroup(t, "组A")
	g2 := createMemberTestGroup(t, "组B")
	u1 := createMemberTestUser(t, "multi1")
	u2 := createMemberTestUser(t, "multi2")

	gdb.Create(&UserGroupMember{UserGroupID: g1.ID, UserID: u1.ID, Source: MemberSourceManual})
	gdb.Create(&UserGroupMember{UserGroupID: g1.ID, UserID: u2.ID, Source: MemberSourceManual})
	gdb.Create(&UserGroupMember{UserGroupID: g2.ID, UserID: u1.ID, Source: MemberSourceManual})

	result, err := GetGroupMembersByGroupIDs(context.Background(), []uint{g1.ID, g2.ID})
	if err != nil {
		t.Fatalf("GetGroupMembersByGroupIDs: %v", err)
	}
	if len(result[g1.ID]) != 2 {
		t.Errorf("g1 expected 2 members, got %d", len(result[g1.ID]))
	}
	if len(result[g2.ID]) != 1 {
		t.Errorf("g2 expected 1 member, got %d", len(result[g2.ID]))
	}
}

func TestCoverageGetGroupMembersByGroupIDs_Empty(t *testing.T) {
	setupMemberCoverageDB(t)

	result, err := GetGroupMembersByGroupIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetGroupMembersByGroupIDs nil: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestCoverageGetGroupMembersByGroupIDs_NoMembers(t *testing.T) {
	setupMemberCoverageDB(t)

	g := createMemberTestGroup(t, "无成员")

	result, err := GetGroupMembersByGroupIDs(context.Background(), []uint{g.ID})
	if err != nil {
		t.Fatalf("GetGroupMembersByGroupIDs no members: %v", err)
	}
	if len(result[g.ID]) != 0 {
		t.Errorf("expected 0 members, got %d", len(result[g.ID]))
	}
}

// ── GetGroupMembers ──────────────────────────────────────

func TestCoverageGetGroupMembers(t *testing.T) {
	setupMemberCoverageDB(t)

	g := createMemberTestGroup(t, "GetMembers组")
	u := createMemberTestUser(t, "memberUser")

	gdb.Create(&UserGroupMember{UserGroupID: g.ID, UserID: u.ID, Source: MemberSourceManual})

	members, err := GetGroupMembers(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("GetGroupMembers: %v", err)
	}
	if len(members) != 1 {
		t.Errorf("expected 1, got %d", len(members))
	}
	if members[0].Username != "memberUser" {
		t.Errorf("unexpected username: %s", members[0].Username)
	}
}

func TestCoverageGetGroupMembers_Empty(t *testing.T) {
	setupMemberCoverageDB(t)

	g := createMemberTestGroup(t, "空成员组")

	members, err := GetGroupMembers(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("GetGroupMembers empty: %v", err)
	}
	if len(members) != 0 {
		t.Errorf("expected 0, got %d", len(members))
	}
}

// ── SetGroupMembers ──────────────────────────────────────

func TestCoverageSetGroupMembers(t *testing.T) {
	setupMemberCoverageDB(t)

	g := createMemberTestGroup(t, "SetMembers组")
	u1 := createMemberTestUser(t, "sm1")
	u2 := createMemberTestUser(t, "sm2")

	err := SetGroupMembers(context.Background(), g.ID, []uint{u1.ID, u2.ID})
	if err != nil {
		t.Fatalf("SetGroupMembers: %v", err)
	}

	members, _ := GetGroupMembers(context.Background(), g.ID)
	if len(members) != 2 {
		t.Errorf("expected 2, got %d", len(members))
	}

	// 替换为只有 u1
	err = SetGroupMembers(context.Background(), g.ID, []uint{u1.ID})
	if err != nil {
		t.Fatalf("SetGroupMembers replace: %v", err)
	}
	members, _ = GetGroupMembers(context.Background(), g.ID)
	if len(members) != 1 {
		t.Errorf("expected 1 after replace, got %d", len(members))
	}
}

func TestCoverageSetGroupMembers_ExceedLimit(t *testing.T) {
	setupMemberCoverageDB(t)

	g := createMemberTestGroup(t, "超限组")
	// 构造超量 userIDs
	ids := make([]uint, MaxMembersPerUserGroup+1)
	for i := range ids {
		ids[i] = uint(i + 1)
	}

	err := SetGroupMembers(context.Background(), g.ID, ids)
	if err == nil {
		t.Error("expected error for exceeding member limit")
	}
}

// ── AddGroupMembers ──────────────────────────────────────

func TestCoverageAddGroupMembers(t *testing.T) {
	setupMemberCoverageDB(t)

	g := createMemberTestGroup(t, "AddMembers组")
	u1 := createMemberTestUser(t, "add1")
	u2 := createMemberTestUser(t, "add2")

	err := AddGroupMembers(context.Background(), g.ID, []uint{u1.ID})
	if err != nil {
		t.Fatalf("AddGroupMembers: %v", err)
	}

	// 幂等：再次添加同一用户
	err = AddGroupMembers(context.Background(), g.ID, []uint{u1.ID, u2.ID})
	if err != nil {
		t.Fatalf("AddGroupMembers idempotent: %v", err)
	}

	members, _ := GetGroupMembers(context.Background(), g.ID)
	if len(members) != 2 {
		t.Errorf("expected 2, got %d", len(members))
	}
}

func TestCoverageAddGroupMembers_Empty(t *testing.T) {
	setupMemberCoverageDB(t)

	g := createMemberTestGroup(t, "AddEmpty组")
	err := AddGroupMembers(context.Background(), g.ID, nil)
	if err != nil {
		t.Errorf("AddGroupMembers empty should not error: %v", err)
	}
}

// ── RemoveGroupMembers ───────────────────────────────────

func TestCoverageRemoveGroupMembers(t *testing.T) {
	setupMemberCoverageDB(t)

	g := createMemberTestGroup(t, "Remove组")
	u1 := createMemberTestUser(t, "rm1")
	u2 := createMemberTestUser(t, "rm2")

	AddGroupMembers(context.Background(), g.ID, []uint{u1.ID, u2.ID})

	err := RemoveGroupMembers(context.Background(), g.ID, []uint{u1.ID})
	if err != nil {
		t.Fatalf("RemoveGroupMembers: %v", err)
	}

	members, _ := GetGroupMembers(context.Background(), g.ID)
	if len(members) != 1 {
		t.Errorf("expected 1, got %d", len(members))
	}
}

func TestCoverageRemoveGroupMembers_Empty(t *testing.T) {
	setupMemberCoverageDB(t)

	g := createMemberTestGroup(t, "RemoveEmpty组")
	err := RemoveGroupMembers(context.Background(), g.ID, nil)
	if err != nil {
		t.Errorf("RemoveGroupMembers empty should not error: %v", err)
	}
}

// ── UpdateUserGroupMemberships ───────────────────────────

func TestCoverageUpdateUserGroupMemberships(t *testing.T) {
	db := setupMemberCoverageDB(t)

	g1 := createMemberTestGroup(t, "UGM1")
	g2 := createMemberTestGroup(t, "UGM2")
	u := createMemberTestUser(t, "ugm_user")

	err := db.Transaction(func(tx *gorm.DB) error {
		return UpdateUserGroupMemberships(tx, u.ID, []uint{g1.ID, g2.ID})
	})
	if err != nil {
		t.Fatalf("UpdateUserGroupMemberships: %v", err)
	}

	ids, _ := GetUserGroupIDs(context.Background(), u.ID)
	if len(ids) != 2 {
		t.Errorf("expected 2, got %d", len(ids))
	}

	// 替换为空
	err = db.Transaction(func(tx *gorm.DB) error {
		return UpdateUserGroupMemberships(tx, u.ID, nil)
	})
	if err != nil {
		t.Fatalf("UpdateUserGroupMemberships clear: %v", err)
	}
	ids, _ = GetUserGroupIDs(context.Background(), u.ID)
	if len(ids) != 0 {
		t.Errorf("expected 0 after clear, got %d", len(ids))
	}
}

func TestCoverageUpdateUserGroupMemberships_InvalidGroup(t *testing.T) {
	db := setupMemberCoverageDB(t)

	u := createMemberTestUser(t, "ugm_invalid")

	err := db.Transaction(func(tx *gorm.DB) error {
		return UpdateUserGroupMemberships(tx, u.ID, []uint{9999})
	})
	if err == nil {
		t.Error("expected error for invalid group ID")
	}
}

// ── CountGroupMembers ────────────────────────────────────

func TestCoverageCountGroupMembers(t *testing.T) {
	setupMemberCoverageDB(t)

	g := createMemberTestGroup(t, "CountG")
	u1 := createMemberTestUser(t, "cnt1")
	u2 := createMemberTestUser(t, "cnt2")

	gdb.Create(&UserGroupMember{UserGroupID: g.ID, UserID: u1.ID, Source: MemberSourceManual})
	gdb.Create(&UserGroupMember{UserGroupID: g.ID, UserID: u2.ID, Source: MemberSourceManual})

	count, err := CountGroupMembers(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("CountGroupMembers: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

// ── UpdateUserGroupMembershipsManualOnly ─────────────────

// TestUpdateUserGroupMembershipsManualOnly_AllManual_Replaces
// 全 manual 输入：旧 manual 行被替换；oneid_dept 行保持不动。
func TestUpdateUserGroupMembershipsManualOnly_AllManual_Replaces(t *testing.T) {
	db := setupMemberCoverageDB(t)

	mg1 := createMemberTestGroup(t, "MM1")
	mg2 := createMemberTestGroup(t, "MM2")
	mg3 := createMemberTestGroup(t, "MM3")

	// 准备一个 oneid_dept 组（手工写入，绕过 createMemberTestGroup）
	dept, err := CreateUserGroupWithOpts(context.Background(), "Dept", "", 0, GroupSourceOneIDDept, "D-1")
	if err != nil {
		t.Fatalf("create dept: %v", err)
	}

	u := createMemberTestUser(t, "manual_only_u")

	// 初态：u ∈ {mg1 (manual), dept (oneid_dept)}
	if err := db.Create(&UserGroupMember{UserGroupID: mg1.ID, UserID: u.ID, Source: MemberSourceManual}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&UserGroupMember{UserGroupID: dept.ID, UserID: u.ID, Source: MemberSourceOneIDDept}).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return UpdateUserGroupMembershipsManualOnly(tx, u.ID, []uint{mg2.ID, mg3.ID})
	}); err != nil {
		t.Fatalf("manual-only update: %v", err)
	}

	// mg1 应被清空
	var n int64
	db.Model(&UserGroupMember{}).Where("user_id = ? AND user_group_id = ?", u.ID, mg1.ID).Count(&n)
	if n != 0 {
		t.Errorf("mg1 应被清空，剩余 %d", n)
	}
	// mg2/mg3 应被写入
	db.Model(&UserGroupMember{}).
		Where("user_id = ? AND user_group_id IN ? AND source = ?", u.ID, []uint{mg2.ID, mg3.ID}, MemberSourceManual).
		Count(&n)
	if n != 2 {
		t.Errorf("mg2/mg3 应写入 2 行，实际 %d", n)
	}
	// dept 必须保留
	db.Model(&UserGroupMember{}).
		Where("user_id = ? AND user_group_id = ? AND source = ?", u.ID, dept.ID, MemberSourceOneIDDept).
		Count(&n)
	if n != 1 {
		t.Errorf("oneid_dept 行不应被破坏，实际 %d", n)
	}
}

// TestUpdateUserGroupMembershipsManualOnly_ContainsOneIDDept_FiltersSilently
// 传入的 group_ids 含 oneid_dept 组 → 静默过滤该项，仅对 manual 子集生效；
// 用户已有的 oneid_dept membership 必须保留。
func TestUpdateUserGroupMembershipsManualOnly_ContainsOneIDDept_FiltersSilently(t *testing.T) {
	db := setupMemberCoverageDB(t)

	mg := createMemberTestGroup(t, "MM")
	dept, err := CreateUserGroupWithOpts(context.Background(), "Dept", "", 0, GroupSourceOneIDDept, "D-2")
	if err != nil {
		t.Fatalf("create dept: %v", err)
	}
	u := createMemberTestUser(t, "filter_u")

	// 初态：用户已是 dept 的 oneid_dept 成员（模拟 OneID 同步落库）
	if err := db.Create(&UserGroupMember{
		UserGroupID: dept.ID, UserID: u.ID, Source: MemberSourceOneIDDept,
	}).Error; err != nil {
		t.Fatal(err)
	}

	// 传入 [mg(manual), dept(oneid_dept)] —— dept 应被静默过滤
	if err := db.Transaction(func(tx *gorm.DB) error {
		return UpdateUserGroupMembershipsManualOnly(tx, u.ID, []uint{mg.ID, dept.ID})
	}); err != nil {
		t.Fatalf("应静默过滤而不报错，实际 %v", err)
	}

	// mg 应被写入 manual 行
	var n int64
	db.Model(&UserGroupMember{}).
		Where("user_id = ? AND user_group_id = ? AND source = ?", u.ID, mg.ID, MemberSourceManual).
		Count(&n)
	if n != 1 {
		t.Errorf("manual mg 应写入 1 行，实际 %d", n)
	}
	// dept 不应被写入新行；原 oneid_dept 行必须保留
	db.Model(&UserGroupMember{}).
		Where("user_id = ? AND user_group_id = ? AND source = ?", u.ID, dept.ID, MemberSourceOneIDDept).
		Count(&n)
	if n != 1 {
		t.Errorf("原 oneid_dept membership 应保留，实际 %d", n)
	}
	db.Model(&UserGroupMember{}).
		Where("user_id = ? AND user_group_id = ? AND source = ?", u.ID, dept.ID, MemberSourceManual).
		Count(&n)
	if n != 0 {
		t.Errorf("dept 不应被写入 manual 行，实际 %d", n)
	}
}

// TestUpdateUserGroupMembershipsManualOnly_AllOneIDDept_OnlyClearsManual
// 传入全是 oneid_dept 的 group_ids → 全部静默过滤；行为等价于传 []，
// 仅清空用户的 manual 行，oneid_dept 行保留。
func TestUpdateUserGroupMembershipsManualOnly_AllOneIDDept_OnlyClearsManual(t *testing.T) {
	db := setupMemberCoverageDB(t)

	mg := createMemberTestGroup(t, "MM-keep-or-clear")
	dept, err := CreateUserGroupWithOpts(context.Background(), "Dept", "", 0, GroupSourceOneIDDept, "D-allod")
	if err != nil {
		t.Fatalf("create dept: %v", err)
	}
	u := createMemberTestUser(t, "all_oneid_u")

	// 初态：mg(manual) + dept(oneid_dept)
	if err := db.Create(&UserGroupMember{
		UserGroupID: mg.ID, UserID: u.ID, Source: MemberSourceManual,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&UserGroupMember{
		UserGroupID: dept.ID, UserID: u.ID, Source: MemberSourceOneIDDept,
	}).Error; err != nil {
		t.Fatal(err)
	}

	// 传入只含 oneid_dept 的 [dept]
	if err := db.Transaction(func(tx *gorm.DB) error {
		return UpdateUserGroupMembershipsManualOnly(tx, u.ID, []uint{dept.ID})
	}); err != nil {
		t.Fatalf("应静默过滤而不报错，实际 %v", err)
	}

	// 旧 manual mg 应被清空
	var n int64
	db.Model(&UserGroupMember{}).
		Where("user_id = ? AND source = ?", u.ID, MemberSourceManual).
		Count(&n)
	if n != 0 {
		t.Errorf("manual 行应被清空，实际 %d", n)
	}
	// oneid_dept 必须保留
	db.Model(&UserGroupMember{}).
		Where("user_id = ? AND source = ?", u.ID, MemberSourceOneIDDept).
		Count(&n)
	if n != 1 {
		t.Errorf("oneid_dept 行应保留，实际 %d", n)
	}
}

// TestUpdateUserGroupMembershipsManualOnly_InvalidGroupID_Rejects
// 传入的 group_id 不存在 → ErrInvalidUserGroupID，且不写入。
func TestUpdateUserGroupMembershipsManualOnly_InvalidGroupID_Rejects(t *testing.T) {
	db := setupMemberCoverageDB(t)

	u := createMemberTestUser(t, "invalid_u")

	err := db.Transaction(func(tx *gorm.DB) error {
		return UpdateUserGroupMembershipsManualOnly(tx, u.ID, []uint{99999})
	})
	if err == nil {
		t.Fatal("应返回错误，实际 nil")
	}
}

// TestUpdateUserGroupMembershipsManualOnly_EmptyClearsManualOnly
// 空 group_ids → 仅清空该用户的 manual 行；oneid_dept 保留。
func TestUpdateUserGroupMembershipsManualOnly_EmptyClearsManualOnly(t *testing.T) {
	db := setupMemberCoverageDB(t)

	mg := createMemberTestGroup(t, "MM")
	dept, err := CreateUserGroupWithOpts(context.Background(), "Dept", "", 0, GroupSourceOneIDDept, "D-3")
	if err != nil {
		t.Fatalf("create dept: %v", err)
	}
	u := createMemberTestUser(t, "empty_u")

	if err := db.Create(&UserGroupMember{UserGroupID: mg.ID, UserID: u.ID, Source: MemberSourceManual}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&UserGroupMember{UserGroupID: dept.ID, UserID: u.ID, Source: MemberSourceOneIDDept}).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return UpdateUserGroupMembershipsManualOnly(tx, u.ID, nil)
	}); err != nil {
		t.Fatalf("empty manual-only: %v", err)
	}

	var n int64
	db.Model(&UserGroupMember{}).Where("user_id = ? AND source = ?", u.ID, MemberSourceManual).Count(&n)
	if n != 0 {
		t.Errorf("manual 行应被清空，实际 %d", n)
	}
	db.Model(&UserGroupMember{}).Where("user_id = ? AND source = ?", u.ID, MemberSourceOneIDDept).Count(&n)
	if n != 1 {
		t.Errorf("oneid_dept 行应保留，实际 %d", n)
	}
}

// errorIsOneIDDeptReadonly 用 errors.Is 判定错误链是否含 ErrOneIDDeptReadonly。
func errorIsOneIDDeptReadonly(err error) bool {
	return errors.Is(err, ErrOneIDDeptReadonly)
}
