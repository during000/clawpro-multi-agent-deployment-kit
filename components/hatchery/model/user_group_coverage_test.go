package model

import (
	"context"
	"testing"
)

func TestCreateUserGroup(t *testing.T) {
	db := setupSeedTestDB(t)
	_ = db

	g, err := CreateUserGroup(context.Background(), "测试组", "描述")
	if err != nil {
		t.Fatalf("CreateUserGroup failed: %v", err)
	}
	if g.ID == 0 {
		t.Error("should have an ID")
	}
}

func TestUpdateUserGroup(t *testing.T) {
	db := setupSeedTestDB(t)
	_ = db

	g, _ := CreateUserGroup(context.Background(), "原名", "")
	got, err := UpdateUserGroup(context.Background(), g.ID, "新名", "新描述")
	if err != nil {
		t.Fatalf("UpdateUserGroup failed: %v", err)
	}
	if got.Name != "新名" {
		t.Errorf("want 新名, got %q", got.Name)
	}
}

func TestDeleteUserGroup(t *testing.T) {
	db := setupSeedTestDB(t)
	_ = db

	g, _ := CreateUserGroup(context.Background(), "待删除", "")
	if err := DeleteUserGroup(context.Background(), g.ID); err != nil {
		t.Fatalf("DeleteUserGroup failed: %v", err)
	}
}

func TestListUserGroups(t *testing.T) {
	db := setupSeedTestDB(t)
	_ = db

	CreateUserGroup(context.Background(), "g1", "")
	CreateUserGroup(context.Background(), "g2", "")

	groups, total, err := ListUserGroups(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("ListUserGroups failed: %v", err)
	}
	if total < 2 {
		t.Errorf("expected at least 2, got %d", total)
	}
	if len(groups) < 2 {
		t.Errorf("expected at least 2 groups, got %d", len(groups))
	}
}

func TestCountGroupMembers(t *testing.T) {
	db := setupSeedTestDB(t)
	_ = db

	g, _ := CreateUserGroup(context.Background(), "cnt-group", "")
	count, err := CountGroupMembers(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("CountGroupMembers failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestCountGroupMembersBatch(t *testing.T) {
	db := setupSeedTestDB(t)
	g, _ := CreateUserGroup(context.Background(), "batch-group", "")

	m, err := CountGroupMembersBatch(db, []uint{g.ID})
	if err != nil {
		t.Fatalf("CountGroupMembersBatch failed: %v", err)
	}
	if m[g.ID] != 0 {
		t.Errorf("expected 0, got %d", m[g.ID])
	}
}

func TestGetGroupMembers(t *testing.T) {
	db := setupSeedTestDB(t)
	_ = db

	g, _ := CreateUserGroup(context.Background(), "members-group", "")
	members, err := GetGroupMembers(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("GetGroupMembers failed: %v", err)
	}
	if len(members) != 0 {
		t.Errorf("expected 0, got %d", len(members))
	}
}

func TestGetGroupMembersByGroupIDs(t *testing.T) {
	db := setupSeedTestDB(t)
	_ = db

	g, _ := CreateUserGroup(context.Background(), "multi-group", "")
	m, err := GetGroupMembersByGroupIDs(context.Background(), []uint{g.ID})
	if err != nil {
		t.Fatalf("GetGroupMembersByGroupIDs failed: %v", err)
	}
	if len(m) == 0 {
		// empty map is fine
	}
}

func TestGetGroupMembersPaged(t *testing.T) {
	db := setupSeedTestDB(t)
	_ = db

	g, _ := CreateUserGroup(context.Background(), "paged-group", "")
	members, total, err := GetGroupMembersPaged(context.Background(), g.ID, 1, 10)
	if err != nil {
		t.Fatalf("GetGroupMembersPaged failed: %v", err)
	}
	if total != 0 || len(members) != 0 {
		t.Errorf("expected empty, got total=%d", total)
	}
}

func TestGetUngroupedUsers(t *testing.T) {
	t.Skip("GetUngroupedUsers has been removed")
}

func TestUpdateUserGroupMemberships(t *testing.T) {
	db := setupSeedTestDB(t)
	user := User{Username: "ugm-user", Password: "x", Role: "user"}
	db.Create(&user)
	g, _ := CreateUserGroup(context.Background(), "ugm-group", "")

	err := UpdateUserGroupMemberships(db, user.ID, []uint{g.ID})
	if err != nil {
		t.Fatalf("UpdateUserGroupMemberships failed: %v", err)
	}

	ids, _ := GetUserGroupIDs(context.Background(), user.ID)
	if len(ids) != 1 || ids[0] != g.ID {
		t.Errorf("expected group %d, got %v", g.ID, ids)
	}
}

func TestSetGroupMembers(t *testing.T) {
	db := setupSeedTestDB(t)
	user := User{Username: "sgm-user", Password: "x", Role: "user"}
	db.Create(&user)
	g, _ := CreateUserGroup(context.Background(), "sgm-group", "")

	if err := SetGroupMembers(context.Background(), g.ID, []uint{user.ID}); err != nil {
		t.Fatalf("SetGroupMembers failed: %v", err)
	}

	count, _ := CountGroupMembers(context.Background(), g.ID)
	if count != 1 {
		t.Errorf("expected 1 member, got %d", count)
	}
}

func TestAddGroupMembers(t *testing.T) {
	db := setupSeedTestDB(t)
	user := User{Username: "agm-user", Password: "x", Role: "user"}
	db.Create(&user)
	g, _ := CreateUserGroup(context.Background(), "agm-group", "")

	if err := AddGroupMembers(context.Background(), g.ID, []uint{user.ID}); err != nil {
		t.Fatalf("AddGroupMembers failed: %v", err)
	}

	count, _ := CountGroupMembers(context.Background(), g.ID)
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestRemoveGroupMembers(t *testing.T) {
	db := setupSeedTestDB(t)
	user := User{Username: "rgm-user", Password: "x", Role: "user"}
	db.Create(&user)
	g, _ := CreateUserGroup(context.Background(), "rgm-group", "")
	AddGroupMembers(context.Background(), g.ID, []uint{user.ID})

	if err := RemoveGroupMembers(context.Background(), g.ID, []uint{user.ID}); err != nil {
		t.Fatalf("RemoveGroupMembers failed: %v", err)
	}

	count, _ := CountGroupMembers(context.Background(), g.ID)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestGetGroupsByNames(t *testing.T) {
	db := setupSeedTestDB(t)
	_ = db

	CreateUserGroup(context.Background(), "name-group-1", "")

	groups, err := GetGroupsByNames(context.Background(), []string{"name-group-1", "nonexistent"})
	if err != nil {
		t.Fatalf("GetGroupsByNames failed: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("expected 1, got %d", len(groups))
	}
}

func TestGetGroupsByIDs(t *testing.T) {
	db := setupSeedTestDB(t)
	_ = db

	g, _ := CreateUserGroup(context.Background(), "ids-group", "")
	groups, err := GetGroupsByIDs(context.Background(), []uint{g.ID, 99999})
	if err != nil {
		t.Fatalf("GetGroupsByIDs failed: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("expected 1, got %d", len(groups))
	}
}

func TestGetUserGroupsByUserID(t *testing.T) {
	db := setupSeedTestDB(t)
	user := User{Username: "ug-user", Password: "x", Role: "user"}
	db.Create(&user)
	g, _ := CreateUserGroup(context.Background(), "ug-group", "")
	AddGroupMembers(context.Background(), g.ID, []uint{user.ID})

	groups, err := GetUserGroupsByUserID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetUserGroupsByUserID failed: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("expected 1, got %d", len(groups))
	}
}

func TestGetUserGroupsByUserIDs(t *testing.T) {
	db := setupSeedTestDB(t)
	user := User{Username: "ugids-user", Password: "x", Role: "user"}
	db.Create(&user)
	g, _ := CreateUserGroup(context.Background(), "ugids-group", "")
	AddGroupMembers(context.Background(), g.ID, []uint{user.ID})

	m, err := GetUserGroupsByUserIDs(context.Background(), []uint{user.ID})
	if err != nil {
		t.Fatalf("GetUserGroupsByUserIDs failed: %v", err)
	}
	if len(m[user.ID]) != 1 {
		t.Errorf("expected 1 group for user, got %d", len(m[user.ID]))
	}
}

func TestCanDeleteUserGroup(t *testing.T) {
	db := setupSeedTestDB(t)
	_ = db

	g, _ := CreateUserGroup(context.Background(), "del-group", "")
	can, err := CanDeleteUserGroup(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("CanDeleteUserGroup failed: %v", err)
	}
	if !can {
		t.Error("empty group should be deletable")
	}
}

func TestGetUserGroups(t *testing.T) {
	db := setupSeedTestDB(t)
	user := User{Username: "getug-user", Password: "x", Role: "user"}
	db.Create(&user)
	g, _ := CreateUserGroup(context.Background(), "getug-group", "")
	AddGroupMembers(context.Background(), g.ID, []uint{user.ID})

	groups, err := GetUserGroups(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetUserGroups failed: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("expected 1, got %d", len(groups))
	}
}

// TestGetGroupsCVMInstanceIDs_Coverage_WithData 覆盖 model/user_group.go GetGroupsCVMInstanceIDs
func TestGetGroupsCVMInstanceIDs_Coverage_WithData(t *testing.T) {
	db := setupSeedTestDB(t)
	db.AutoMigrate(&Instance{})

	grp, _ := CreateUserGroup(context.Background(), "cvm-test-grp", "")

	// 创建实例，通过 group_id 关联到分组（不依赖用户分组成员关系）
	db.Create(&Instance{Name: "test-inst", UserID: 1, InstanceId: "ins-test-xyz", GroupID: grp.ID})

	ids, err := GetGroupsCVMInstanceIDs(context.Background(), []uint{grp.ID})
	if err != nil {
		t.Fatalf("GetGroupsCVMInstanceIDs: %v", err)
	}
	if len(ids) == 0 {
		t.Error("expected at least 1 instance id")
	}
}

func TestGetGroupsCVMInstanceIDs_Coverage_Empty(t *testing.T) {
	setupSeedTestDB(t)

	ids, err := GetGroupsCVMInstanceIDs(context.Background(), []uint{})
	if err != nil {
		t.Fatalf("GetGroupsCVMInstanceIDs(empty): %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0, got %d", len(ids))
	}
}

// TestGetGroupMembers_Basic 覆盖 model/user_group.go line 194
func TestGetGroupMembers_Basic(t *testing.T) {
	db := setupSeedTestDB(t)

	user := User{Username: "ggm-user", Password: "x", Role: "user"}
	db.Create(&user)
	grp, _ := CreateUserGroup(context.Background(), "ggm-grp", "")
	AddGroupMembers(context.Background(), grp.ID, []uint{user.ID})

	members, err := GetGroupMembers(context.Background(), grp.ID)
	if err != nil {
		t.Fatalf("GetGroupMembers: %v", err)
	}
	if len(members) == 0 {
		t.Error("expected members")
	}
}

// TestGetGroupMembersPaged_Basic 覆盖 model/user_group.go line 281-302
func TestGetGroupMembersPaged_Basic(t *testing.T) {
	db := setupSeedTestDB(t)

	user := User{Username: "gmpp-user", Password: "x", Role: "user"}
	db.Create(&user)
	grp, _ := CreateUserGroup(context.Background(), "gmpp-grp", "")
	AddGroupMembers(context.Background(), grp.ID, []uint{user.ID})

	members, total, err := GetGroupMembersPaged(context.Background(), grp.ID, 1, 10)
	if err != nil {
		t.Fatalf("GetGroupMembersPaged: %v", err)
	}
	if total == 0 || len(members) == 0 {
		t.Error("expected members")
	}
}

// TestDeleteUserGroup_WithCLSScopeBinding 验证删除含 CLS 采集范围绑定的分组时，绑定被清理。
func TestDeleteUserGroup_WithCLSScopeBinding(t *testing.T) {
	db := setupSeedTestDB(t)
	_ = db

	g, err := CreateUserGroup(context.Background(), "cls-scope-grp", "")
	if err != nil {
		t.Fatalf("CreateUserGroup failed: %v", err)
	}
	// 添加 CLS scope 绑定
	SetCLSCollectScope(context.Background(), []uint{g.ID})

	// 确认绑定存在
	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) == 0 {
		t.Fatal("scope 绑定应存在")
	}

	// 删除分组
	if err := DeleteUserGroup(context.Background(), g.ID); err != nil {
		t.Fatalf("DeleteUserGroup failed: %v", err)
	}

	// 验证绑定被清理
	ids, _ = GetCLSCollectScopeGroupIDs(context.Background())
	for _, id := range ids {
		if id == g.ID {
			t.Errorf("分组删除后其 CLS scope 绑定应被清理")
		}
	}
}
