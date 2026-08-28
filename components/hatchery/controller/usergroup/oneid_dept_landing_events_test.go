package usergroup

import (
	"context"
	"encoding/json"
	"testing"

	"hatchery/model"
)

// TestLandOneIDDepartmentsToGroups_ChangedParentGroupIDs 覆盖 v6.13：
// 同步发现某 oneid_dept 分组的父节点发生了切换时，LandDepartmentsResult
// 必须在 ChangedParentGroupIDs 里记这个组的 ID。仅改名不换父不应记录。
//
// 场景：
//
//	第一轮：总部 / 运营组 / 二组（D12 挂在 D1 下）
//	第二轮：OneID 把 D12 的父改成 D2（后台组）→ D12 换父
//	                同时 D1 改名为 "运营一组"（只改名不换父）→ D1 不应记入
func TestLandOneIDDepartmentsToGroups_ChangedParentGroupIDs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(&model.OneIDDepartmentRecord{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	initial := []model.OneIDDepartmentRecord{
		{DepartmentID: "D0", DepartmentName: "总部", DepartmentParentID: ""},
		{DepartmentID: "D1", DepartmentName: "运营组", DepartmentParentID: "D0"},
		{DepartmentID: "D2", DepartmentName: "后台组", DepartmentParentID: "D0"},
		{DepartmentID: "D12", DepartmentName: "二组", DepartmentParentID: "D1"},
	}
	for _, d := range initial {
		if err := model.DB(context.Background()).Create(&d).Error; err != nil {
			t.Fatalf("seed dept %s: %v", d.DepartmentID, err)
		}
	}
	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("first landing: %v", err)
	}

	// 找到 D1 / D12 本地 group id
	var g1, g12 model.UserGroup
	if err := model.DB(context.Background()).Where("source_ref = ?", "D1").First(&g1).Error; err != nil {
		t.Fatalf("find D1: %v", err)
	}
	if err := model.DB(context.Background()).Where("source_ref = ?", "D12").First(&g12).Error; err != nil {
		t.Fatalf("find D12: %v", err)
	}

	// 改动：D12 换父到 D2、D1 改名（不换父）
	if err := model.DB(context.Background()).Model(&model.OneIDDepartmentRecord{}).
		Where("department_id = ?", "D12").
		Update("department_parent_id", "D2").Error; err != nil {
		t.Fatalf("reparent D12: %v", err)
	}
	if err := model.DB(context.Background()).Model(&model.OneIDDepartmentRecord{}).
		Where("department_id = ?", "D1").
		Update("department_name", "运营一组").Error; err != nil {
		t.Fatalf("rename D1: %v", err)
	}

	res, err := LandOneIDDepartmentsToGroups(context.Background())
	if err != nil {
		t.Fatalf("second landing: %v", err)
	}
	if len(res.LandingFailures) != 0 {
		t.Fatalf("期望无 landing failure，实际 %+v", res.LandingFailures)
	}

	// 断言：ChangedParentGroupIDs 仅含 D12（只有它换父）
	if len(res.ChangedParentGroupIDs) != 1 || res.ChangedParentGroupIDs[0] != g12.ID {
		t.Fatalf("期望 ChangedParentGroupIDs=[%d]，实际 %v", g12.ID, res.ChangedParentGroupIDs)
	}
	// D1 只改名不换父 → 不应出现
	for _, id := range res.ChangedParentGroupIDs {
		if id == g1.ID {
			t.Fatalf("D1 只改名不换父，不应计入 ChangedParentGroupIDs")
		}
	}
}

// TestSyncOneIDMemberships_MovedUsers 覆盖 v6.13：
// 用户在 OneID 侧被调出某部门后，SyncOneIDMemberships 必须把对应
// (user_id, from_group_id) 记入 MembershipsResult.MovedUsers。
//
// 场景：
//
//	alice 初始属于 D1、D2 两个 oneid_dept 组
//	OneID 把 alice 调出 D2（DepartmentsJSON 只剩 D1）
//	→ MovedUsers 应含 {UserID: alice, FromGroupID: D2 对应的 group id}
func TestSyncOneIDMemberships_MovedUsers(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(
		&model.OneIDDepartmentRecord{},
		&model.OneIDUserProfile{},
		&model.User{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 1) 建部门 + landing
	depts := []model.OneIDDepartmentRecord{
		{DepartmentID: "D0", DepartmentName: "总部", DepartmentParentID: ""},
		{DepartmentID: "D1", DepartmentName: "研发", DepartmentParentID: "D0"},
		{DepartmentID: "D2", DepartmentName: "运营", DepartmentParentID: "D0"},
	}
	for _, d := range depts {
		if err := model.DB(context.Background()).Create(&d).Error; err != nil {
			t.Fatalf("seed dept %s: %v", d.DepartmentID, err)
		}
	}
	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("landing: %v", err)
	}
	var g1, g2 model.UserGroup
	if err := model.DB(context.Background()).Where("source_ref = ?", "D1").First(&g1).Error; err != nil {
		t.Fatalf("find D1: %v", err)
	}
	if err := model.DB(context.Background()).Where("source_ref = ?", "D2").First(&g2).Error; err != nil {
		t.Fatalf("find D2: %v", err)
	}

	// 2) 创建 alice（带 OneID sub）+ profile 初始归属 D1 + D2
	sub := "sub-alice"
	alice := model.User{Username: "alice", Role: "user", OneIDSub: &sub}
	if err := model.DB(context.Background()).Create(&alice).Error; err != nil {
		t.Fatalf("create alice: %v", err)
	}

	initialDepts, _ := json.Marshal([]model.OneIDDepartment{
		{DepartmentID: "D1", DepartmentName: "研发", IsMainDepartment: true},
		{DepartmentID: "D2", DepartmentName: "运营", IsMainDepartment: false},
	})
	profile := model.OneIDUserProfile{
		OneIDSub:        sub,
		MainDeptID:      "D1",
		MainDeptName:    "研发",
		DepartmentsJSON: string(initialDepts),
	}
	if err := model.DB(context.Background()).Create(&profile).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}

	// 3) 第一轮 SyncOneIDMemberships：应新建两条成员、无 MovedUsers
	res1, err := SyncOneIDMemberships(context.Background())
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if len(res1.MovedUsers) != 0 {
		t.Fatalf("首次同步不应有 MovedUsers，实际 %+v", res1.MovedUsers)
	}
	var count int64
	model.DB(context.Background()).Model(&model.UserGroupMember{}).Where("user_id = ?", alice.ID).Count(&count)
	if count != 2 {
		t.Fatalf("首次同步期望 2 条成员，实际 %d", count)
	}

	// 4) 改 profile 只剩 D1（模拟 OneID 把 alice 调出 D2）
	newDepts, _ := json.Marshal([]model.OneIDDepartment{
		{DepartmentID: "D1", DepartmentName: "研发", IsMainDepartment: true},
	})
	if err := model.DB(context.Background()).Model(&model.OneIDUserProfile{}).
		Where("one_id_sub = ?", sub).
		Update("departments_json", string(newDepts)).Error; err != nil {
		t.Fatalf("update profile: %v", err)
	}

	// 5) 第二轮同步：应记录 MovedUsers = [{alice, g2}]
	res2, err := SyncOneIDMemberships(context.Background())
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if len(res2.MovedUsers) != 1 {
		t.Fatalf("期望 1 条 MovedUsers，实际 %d: %+v", len(res2.MovedUsers), res2.MovedUsers)
	}
	mv := res2.MovedUsers[0]
	if mv.UserID != alice.ID || mv.FromGroupID != g2.ID {
		t.Fatalf("MovedUsers 内容错：期望 {alice=%d, from=%d(D2)}，实际 %+v",
			alice.ID, g2.ID, mv)
	}

	// 6) 断言成员行已被删：alice 仅剩 D1 一条
	model.DB(context.Background()).Model(&model.UserGroupMember{}).Where("user_id = ?", alice.ID).Count(&count)
	if count != 1 {
		t.Fatalf("同步后期望 1 条成员，实际 %d", count)
	}
	var remaining model.UserGroupMember
	model.DB(context.Background()).Where("user_id = ?", alice.ID).First(&remaining)
	if remaining.UserGroupID != g1.ID {
		t.Fatalf("剩下的成员应是 D1(%d)，实际 %d", g1.ID, remaining.UserGroupID)
	}
}

// TestSyncOneIDMemberships_NoChange 没有调整时 MovedUsers 为空。
func TestSyncOneIDMemberships_NoChange(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(
		&model.OneIDDepartmentRecord{},
		&model.OneIDUserProfile{},
		&model.User{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	if err := model.DB(context.Background()).Create(&model.OneIDDepartmentRecord{
		DepartmentID: "D1", DepartmentName: "研发", DepartmentParentID: "",
	}).Error; err != nil {
		t.Fatalf("seed dept: %v", err)
	}
	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("landing: %v", err)
	}

	sub := "sub-u1"
	if err := model.DB(context.Background()).Create(&model.User{Username: "u1", Role: "user", OneIDSub: &sub}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	depts, _ := json.Marshal([]model.OneIDDepartment{
		{DepartmentID: "D1", DepartmentName: "研发", IsMainDepartment: true},
	})
	if err := model.DB(context.Background()).Create(&model.OneIDUserProfile{
		OneIDSub: sub, MainDeptID: "D1", DepartmentsJSON: string(depts),
	}).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}

	// 第一轮新建成员
	if _, err := SyncOneIDMemberships(context.Background()); err != nil {
		t.Fatalf("first: %v", err)
	}
	// 第二轮未改 profile → 无 MovedUsers
	res, err := SyncOneIDMemberships(context.Background())
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if len(res.MovedUsers) != 0 {
		t.Fatalf("未改动时 MovedUsers 应为空，实际 %+v", res.MovedUsers)
	}
}
