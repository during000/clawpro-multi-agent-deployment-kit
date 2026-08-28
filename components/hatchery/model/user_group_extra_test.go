package model

import (
	"context"
	"testing"
)

// ── ValidateGroupName 测试 ──────────────────────────────────────────────────

func TestExtraValidateGroupName_Empty(t *testing.T) {
	err := ValidateGroupName("")
	if err == nil {
		t.Error("空名称应报错")
	}
}

func TestExtraValidateGroupName_TooLong(t *testing.T) {
	name := make([]byte, 200)
	for i := range name {
		name[i] = 'a'
	}
	err := ValidateGroupName(string(name))
	if err == nil {
		t.Error("超长名称应报错")
	}
}

func TestExtraValidateGroupName_Valid(t *testing.T) {
	err := ValidateGroupName("研发组")
	if err != nil {
		t.Errorf("合法名称不应报错: %v", err)
	}
}

func TestExtraValidateGroupName_WithSlash(t *testing.T) {
	err := ValidateGroupName("研发/前端")
	if err == nil {
		t.Error("含斜杠应报错")
	}
}

// ── GroupByID 测试 ──────────────────────────────────────────────────────────

func TestExtraGroupByID_Exists(t *testing.T) {
	setupGroupClosureExtraDB(t)
	g := &UserGroup{Name: "TestGroup", Source: "manual"}
	gdb.Create(g)

	got, err := GroupByID(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("GroupByID err: %v", err)
	}
	if got.Name != "TestGroup" {
		t.Errorf("期望 TestGroup，实际=%s", got.Name)
	}
}

func TestExtraGroupByID_NotFound(t *testing.T) {
	setupGroupClosureExtraDB(t)

	_, err := GroupByID(context.Background(), 99999)
	if err == nil {
		t.Error("期望错误（不存在）")
	}
}

// ── GetGroupsByIDs 测试 ─────────────────────────────────────────────────────

func TestExtraGetGroupsByIDs_Multiple(t *testing.T) {
	setupGroupClosureExtraDB(t)
	g1 := &UserGroup{Name: "Group1", Source: "manual"}
	g2 := &UserGroup{Name: "Group2", Source: "manual"}
	gdb.Create(g1)
	gdb.Create(g2)

	groups, err := GetGroupsByIDs(context.Background(), []uint{g1.ID, g2.ID})
	if err != nil {
		t.Fatalf("GetGroupsByIDs err: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("期望 2 个，实际=%d", len(groups))
	}
}

func TestExtraGetGroupsByIDs_Empty(t *testing.T) {
	setupGroupClosureExtraDB(t)

	groups, err := GetGroupsByIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetGroupsByIDs err: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("期望空，实际=%d", len(groups))
	}
}

// （GetUngroupedUsersFiltered 测试已删除，该函数已废弃）

// ── CountGroupMembers 测试 ──────────────────────────────────────────────────

func TestExtraCountGroupMembers_WithMembers(t *testing.T) {
	setupGroupClosureExtraDB(t)

	g := &UserGroup{Name: "Group1", Source: "manual"}
	gdb.Create(g)
	u1 := &User{Username: "u1", Password: "hash", Role: "user"}
	u2 := &User{Username: "u2", Password: "hash", Role: "user"}
	gdb.Create(u1)
	gdb.Create(u2)
	gdb.Create(&UserGroupMember{UserGroupID: g.ID, UserID: u1.ID, Source: "manual"})
	gdb.Create(&UserGroupMember{UserGroupID: g.ID, UserID: u2.ID, Source: "manual"})

	count, err := CountGroupMembers(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("CountGroupMembers err: %v", err)
	}
	if count != 2 {
		t.Errorf("期望 2，实际=%d", count)
	}
}

func TestExtraCountGroupMembers_Empty(t *testing.T) {
	setupGroupClosureExtraDB(t)

	g := &UserGroup{Name: "EmptyGroup", Source: "manual"}
	gdb.Create(g)

	count, err := CountGroupMembers(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("CountGroupMembers err: %v", err)
	}
	if count != 0 {
		t.Errorf("期望 0，实际=%d", count)
	}
}

// ── CountGroupMembersBatch 测试 ─────────────────────────────────────────────

func TestExtraCountGroupMembersBatch(t *testing.T) {
	setupGroupClosureExtraDB(t)

	g1 := &UserGroup{Name: "G1", Source: "manual"}
	g2 := &UserGroup{Name: "G2", Source: "manual"}
	gdb.Create(g1)
	gdb.Create(g2)

	u := &User{Username: "batchuser", Password: "hash", Role: "user"}
	gdb.Create(u)
	gdb.Create(&UserGroupMember{UserGroupID: g1.ID, UserID: u.ID, Source: "manual"})

	counts, err := CountGroupMembersBatch(gdb, []uint{g1.ID, g2.ID})
	if err != nil {
		t.Fatalf("CountGroupMembersBatch err: %v", err)
	}
	if counts[g1.ID] != 1 {
		t.Errorf("g1 期望 1，实际=%d", counts[g1.ID])
	}
	if counts[g2.ID] != 0 {
		t.Errorf("g2 期望 0，实际=%d", counts[g2.ID])
	}
}

// ── GetUserGroupsByUserID 测试 ──────────────────────────────────────────────

func TestExtraGetUserGroupsByUserID(t *testing.T) {
	setupGroupClosureExtraDB(t)

	u := &User{Username: "multigroup", Password: "hash", Role: "user"}
	gdb.Create(u)
	g1 := &UserGroup{Name: "G1", Source: "manual"}
	g2 := &UserGroup{Name: "G2", Source: "manual"}
	gdb.Create(g1)
	gdb.Create(g2)
	gdb.Create(&UserGroupMember{UserGroupID: g1.ID, UserID: u.ID, Source: "manual"})
	gdb.Create(&UserGroupMember{UserGroupID: g2.ID, UserID: u.ID, Source: "manual"})

	groups, err := GetUserGroupsByUserID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetUserGroupsByUserID err: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("期望 2 个组，实际=%d", len(groups))
	}
}

func TestExtraGetUserGroupsByUserID_NoGroups(t *testing.T) {
	setupGroupClosureExtraDB(t)

	u := &User{Username: "loner", Password: "hash", Role: "user"}
	gdb.Create(u)

	groups, err := GetUserGroupsByUserID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetUserGroupsByUserID err: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("期望 0 个组，实际=%d", len(groups))
	}
}

// ── GetUserGroupIDs 测试 ────────────────────────────────────────────────────

func TestExtraGetUserGroupIDs(t *testing.T) {
	setupGroupClosureExtraDB(t)

	u := &User{Username: "giduser", Password: "hash", Role: "user"}
	gdb.Create(u)
	g := &UserGroup{Name: "GID Group", Source: "manual"}
	gdb.Create(g)
	gdb.Create(&UserGroupMember{UserGroupID: g.ID, UserID: u.ID, Source: "manual"})

	ids, err := GetUserGroupIDs(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetUserGroupIDs err: %v", err)
	}
	if len(ids) != 1 || ids[0] != g.ID {
		t.Errorf("期望 [%d]，实际=%v", g.ID, ids)
	}
}

// ── AssertEditableByAdmin 测试 ──────────────────────────────────────────────

func TestExtraAssertEditableByAdmin_Manual(t *testing.T) {
	g := &UserGroup{Source: GroupSourceManual}
	if err := AssertEditableByAdmin(g); err != nil {
		t.Errorf("manual 组应可编辑: %v", err)
	}
}

func TestExtraAssertEditableByAdmin_OneIDDept(t *testing.T) {
	g := &UserGroup{Source: GroupSourceOneIDDept}
	if err := AssertEditableByAdmin(g); err == nil {
		t.Error("oneid_dept 组不应可编辑")
	}
}

// ── deduplicateUintSlice 测试 ───────────────────────────────────────────────

func TestExtraDeduplicateUintSlice(t *testing.T) {
	result := deduplicateUintSlice([]uint{1, 2, 2, 3, 1, 4})
	if len(result) != 4 {
		t.Errorf("期望 4 个唯一值，实际=%d", len(result))
	}
}

func TestExtraDeduplicateUintSlice_Empty(t *testing.T) {
	result := deduplicateUintSlice(nil)
	if len(result) != 0 {
		t.Errorf("期望空，实际=%d", len(result))
	}
}

// ── GetGroupsByNames 测试 ───────────────────────────────────────────────────

func TestExtraGetGroupsByNames(t *testing.T) {
	setupGroupClosureExtraDB(t)

	gdb.Create(&UserGroup{Name: "Alpha", Source: "manual"})
	gdb.Create(&UserGroup{Name: "Beta", Source: "manual"})

	groups, err := GetGroupsByNames(context.Background(), []string{"Alpha", "Beta"})
	if err != nil {
		t.Fatalf("GetGroupsByNames err: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("期望 2，实际=%d", len(groups))
	}
}

func TestExtraGetGroupsByNames_NotFound(t *testing.T) {
	setupGroupClosureExtraDB(t)

	groups, err := GetGroupsByNames(context.Background(), []string{"nonexist"})
	if err != nil {
		t.Fatalf("GetGroupsByNames err: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("期望 0，实际=%d", len(groups))
	}
}

// ── ListUserGroupsExt 测试 ──────────────────────────────────────────────────

func TestExtraListUserGroupsExt_Pagination(t *testing.T) {
	setupGroupClosureExtraDB(t)

	for i := 0; i < 5; i++ {
		gdb.Create(&UserGroup{Name: "Group" + string(rune('A'+i)), Source: "manual"})
	}

	groups, total, err := ListUserGroupsExt(context.Background(), ListUserGroupsOpts{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("ListUserGroupsExt err: %v", err)
	}
	if total != 5 {
		t.Errorf("期望 total=5，实际=%d", total)
	}
	if len(groups) != 2 {
		t.Errorf("期望 page size=2，实际=%d", len(groups))
	}
}

func TestExtraListUserGroupsExt_WithQuery(t *testing.T) {
	setupGroupClosureExtraDB(t)

	gdb.Create(&UserGroup{Name: "研发组", Source: "manual"})
	gdb.Create(&UserGroup{Name: "产品组", Source: "manual"})

	groups, total, err := ListUserGroupsExt(context.Background(), ListUserGroupsOpts{
		Page: 1, PageSize: 20, Query: "研发",
	})
	if err != nil {
		t.Fatalf("ListUserGroupsExt err: %v", err)
	}
	if total != 1 {
		t.Errorf("期望 1，实际=%d", total)
	}
	if len(groups) != 1 || groups[0].Name != "研发组" {
		t.Errorf("期望 研发组，实际=%v", groups)
	}
}
