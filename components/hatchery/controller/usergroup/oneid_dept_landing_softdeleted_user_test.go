// oneid_dept_landing_softdeleted_user_test.go
//
// 回归：OneID 上游 Suspended/Disabled/LockedOut 的用户，本地走 model.DB(context.Background()).Delete
// 软删（deleted_at != NULL），但仍存在于组织结构内。SyncOneIDMemberships 必须
// 把这些用户也纳入 desired map，否则 step 6 差集会把他们的 oneid_dept membership
// 全部误删 —— 表现为"OneID 上仍归属某部门、ClawPro 上变成未分组"。
//
// 修复点：oneid_dept_landing.go 中 sub→user_id 映射查询加 Unscoped()。

package usergroup

import (
	"context"
	"encoding/json"
	"testing"

	"hatchery/model"
)

// TestSyncOneIDMemberships_SoftDeletedUserKeepsMembership
// 场景：alice 已经在 ClawPro 上是 oneid_dept 成员（D1），后被 OneID 标禁用
// → 本地软删；OneID 拉回的 profile 仍含 D1。再跑一轮 SyncOneIDMemberships，
// alice 的 (oneid_dept, D1) membership 必须保留，不能被 step 6 删除。
func TestSyncOneIDMemberships_SoftDeletedUserKeepsMembership(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(
		&model.OneIDDepartmentRecord{},
		&model.OneIDUserProfile{},
		&model.User{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 1) 部门 + landing
	depts := []model.OneIDDepartmentRecord{
		{DepartmentID: "D0", DepartmentName: "总部", DepartmentParentID: ""},
		{DepartmentID: "D1", DepartmentName: "研发", DepartmentParentID: "D0"},
	}
	for _, d := range depts {
		if err := model.DB(context.Background()).Create(&d).Error; err != nil {
			t.Fatalf("seed dept %s: %v", d.DepartmentID, err)
		}
	}
	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("landing: %v", err)
	}
	var g1 model.UserGroup
	if err := model.DB(context.Background()).Where("source_ref = ?", "D1").First(&g1).Error; err != nil {
		t.Fatalf("find D1: %v", err)
	}

	// 2) alice + profile
	sub := "sub-alice-disabled"
	alice := model.User{Username: "alice", Role: "user", OneIDSub: &sub}
	if err := model.DB(context.Background()).Create(&alice).Error; err != nil {
		t.Fatalf("create alice: %v", err)
	}
	deptsJSON, _ := json.Marshal([]model.OneIDDepartment{
		{DepartmentID: "D1", DepartmentName: "研发", IsMainDepartment: true},
	})
	if err := model.DB(context.Background()).Create(&model.OneIDUserProfile{
		OneIDSub:        sub,
		MainDeptID:      "D1",
		MainDeptName:    "研发",
		DepartmentsJSON: string(deptsJSON),
	}).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}

	// 3) 第一轮同步：alice 应该有 1 条 oneid_dept membership
	if _, err := SyncOneIDMemberships(context.Background()); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	var n int64
	model.DB(context.Background()).Model(&model.UserGroupMember{}).
		Where("user_id = ? AND source = ?", alice.ID, model.MemberSourceOneIDDept).
		Count(&n)
	if n != 1 {
		t.Fatalf("第一轮同步期望 1 条 oneid_dept membership，实际 %d", n)
	}

	// 4) alice 在 OneID 被禁用 → 本地软删
	if err := model.DB(context.Background()).Delete(&alice).Error; err != nil {
		t.Fatalf("soft-delete alice: %v", err)
	}
	// 确认确实软删了（默认查不到，Unscoped 能查到）
	var existsDefault int64
	model.DB(context.Background()).Model(&model.User{}).Where("id = ?", alice.ID).Count(&existsDefault)
	if existsDefault != 0 {
		t.Fatalf("alice 应已被软删，默认查询不应命中，实际 %d", existsDefault)
	}
	var existsUnscoped int64
	model.DB(context.Background()).Unscoped().Model(&model.User{}).Where("id = ?", alice.ID).Count(&existsUnscoped)
	if existsUnscoped != 1 {
		t.Fatalf("alice 软删后 Unscoped 应能查到，实际 %d", existsUnscoped)
	}

	// 5) 再跑一轮同步：alice 的 (oneid_dept, D1) 必须保留
	if _, err := SyncOneIDMemberships(context.Background()); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	model.DB(context.Background()).Model(&model.UserGroupMember{}).
		Where("user_id = ? AND source = ?", alice.ID, model.MemberSourceOneIDDept).
		Count(&n)
	if n != 1 {
		t.Fatalf("禁用后再同步期望仍保留 1 条 oneid_dept membership，实际 %d（修复前会被 step 6 差集误删）", n)
	}
	var remaining model.UserGroupMember
	if err := model.DB(context.Background()).Where("user_id = ? AND source = ?", alice.ID, model.MemberSourceOneIDDept).
		First(&remaining).Error; err != nil {
		t.Fatalf("查 alice 剩余 membership: %v", err)
	}
	if remaining.UserGroupID != g1.ID {
		t.Fatalf("剩余 membership 应指向 D1(%d)，实际 %d", g1.ID, remaining.UserGroupID)
	}
	if !remaining.IsMain {
		t.Errorf("alice 在 D1 是 IsMainDepartment，IsMain 应为 true")
	}
}

// TestSyncOneIDMemberships_FirstTimeOnSoftDeletedUser
// 场景：alice 在第一次跑同步前就已经处于本地软删态（OneID 那边新拉过来的 profile
// 含 D1）。修复前 subToUserID 跳过 alice → desired 不含 alice 的 D1 → 一条
// membership 都不写入；修复后应正常写入。
func TestSyncOneIDMemberships_FirstTimeOnSoftDeletedUser(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(
		&model.OneIDDepartmentRecord{},
		&model.OneIDUserProfile{},
		&model.User{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 1) 部门 + landing
	depts := []model.OneIDDepartmentRecord{
		{DepartmentID: "D0", DepartmentName: "总部", DepartmentParentID: ""},
		{DepartmentID: "D1", DepartmentName: "研发", DepartmentParentID: "D0"},
	}
	for _, d := range depts {
		if err := model.DB(context.Background()).Create(&d).Error; err != nil {
			t.Fatalf("seed dept %s: %v", d.DepartmentID, err)
		}
	}
	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("landing: %v", err)
	}
	var g1 model.UserGroup
	if err := model.DB(context.Background()).Where("source_ref = ?", "D1").First(&g1).Error; err != nil {
		t.Fatalf("find D1: %v", err)
	}

	// 2) 直接造一个软删用户 + 含 D1 的 profile（模拟之前已被禁用、现在才第一次跑同步）
	sub := "sub-bob-pre-disabled"
	bob := model.User{Username: "bob", Role: "user", OneIDSub: &sub}
	if err := model.DB(context.Background()).Create(&bob).Error; err != nil {
		t.Fatalf("create bob: %v", err)
	}
	if err := model.DB(context.Background()).Delete(&bob).Error; err != nil {
		t.Fatalf("soft-delete bob: %v", err)
	}
	deptsJSON, _ := json.Marshal([]model.OneIDDepartment{
		{DepartmentID: "D1", DepartmentName: "研发", IsMainDepartment: true},
	})
	if err := model.DB(context.Background()).Create(&model.OneIDUserProfile{
		OneIDSub:        sub,
		MainDeptID:      "D1",
		MainDeptName:    "研发",
		DepartmentsJSON: string(deptsJSON),
	}).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}

	// 3) 第一轮同步：bob 必须被纳入；oneid_dept membership 应该被建出
	if _, err := SyncOneIDMemberships(context.Background()); err != nil {
		t.Fatalf("sync: %v", err)
	}
	var n int64
	model.DB(context.Background()).Model(&model.UserGroupMember{}).
		Where("user_id = ? AND source = ?", bob.ID, model.MemberSourceOneIDDept).
		Count(&n)
	if n != 1 {
		t.Fatalf("软删用户首次同步期望写入 1 条 membership，实际 %d（修复前是 0）", n)
	}
	var got model.UserGroupMember
	if err := model.DB(context.Background()).Where("user_id = ? AND source = ?", bob.ID, model.MemberSourceOneIDDept).
		First(&got).Error; err != nil {
		t.Fatalf("查 bob membership: %v", err)
	}
	if got.UserGroupID != g1.ID {
		t.Fatalf("membership 应指向 D1(%d)，实际 %d", g1.ID, got.UserGroupID)
	}
}
