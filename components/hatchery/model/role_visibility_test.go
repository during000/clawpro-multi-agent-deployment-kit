package model

import (
	"context"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupRoleVisibilityTestDB creates a temporary SQLite database for role visibility tests.
func setupRoleVisibilityTestDB(t *testing.T) (cleanup func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "role_visibility_test_*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()

	dsn := tmpFile.Name() + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("open test db: %v", err)
	}

	origDB := gdb
	gdb = testDB

	if err := gdb.AutoMigrate(&OpenClawRole{}, &RoleVisibilityGroup{}, &UserGroup{}, &UserGroupMember{}); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("auto migrate: %v", err)
	}

	return func() {
		sqlDB, _ := gdb.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
		gdb = origDB
	}
}

// ─── IsRoleVisibleToUser Tests ───

func TestIsRoleVisibleToUser_AllType(t *testing.T) {
	cleanup := setupRoleVisibilityTestDB(t)
	defer cleanup()

	role := OpenClawRole{Name: "通用助手", VisibilityType: "all", Visible: true}
	gdb.Create(&role)

	visible, err := IsRoleVisibleToUser(context.Background(), &role, 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !visible {
		t.Error("all-type role should be visible to any user")
	}
}

func TestIsRoleVisibleToUser_GroupType_Match(t *testing.T) {
	cleanup := setupRoleVisibilityTestDB(t)
	defer cleanup()

	group := UserGroup{Name: "研发"}
	gdb.Create(&group)
	gdb.Create(&UserGroupMember{UserGroupID: group.ID, UserID: 1})

	role := OpenClawRole{Name: "代码助手", VisibilityType: "group", Visible: true}
	gdb.Create(&role)
	gdb.Create(&RoleVisibilityGroup{OpenClawRoleID: role.ID, GroupID: group.ID})

	visible, err := IsRoleVisibleToUser(context.Background(), &role, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !visible {
		t.Error("user in matching group should see the role")
	}
}

func TestIsRoleVisibleToUser_GroupType_NoMatch(t *testing.T) {
	cleanup := setupRoleVisibilityTestDB(t)
	defer cleanup()

	groupA := UserGroup{Name: "研发"}
	gdb.Create(&groupA)
	groupB := UserGroup{Name: "市场"}
	gdb.Create(&groupB)
	gdb.Create(&UserGroupMember{UserGroupID: groupA.ID, UserID: 1})

	role := OpenClawRole{Name: "营销助手", VisibilityType: "group", Visible: true}
	gdb.Create(&role)
	gdb.Create(&RoleVisibilityGroup{OpenClawRoleID: role.ID, GroupID: groupB.ID})

	visible, err := IsRoleVisibleToUser(context.Background(), &role, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if visible {
		t.Error("user not in matching group should NOT see the role")
	}
}

func TestIsRoleVisibleToUser_GroupType_UserNoGroups(t *testing.T) {
	cleanup := setupRoleVisibilityTestDB(t)
	defer cleanup()

	group := UserGroup{Name: "研发"}
	gdb.Create(&group)

	role := OpenClawRole{Name: "代码助手", VisibilityType: "group", Visible: true}
	gdb.Create(&role)
	gdb.Create(&RoleVisibilityGroup{OpenClawRoleID: role.ID, GroupID: group.ID})

	visible, err := IsRoleVisibleToUser(context.Background(), &role, 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if visible {
		t.Error("user with no groups should NOT see group-type role")
	}
}

// ─── GetRoleVisibilityGroupIDs Tests ───

func TestGetRoleVisibilityGroupIDs_Empty(t *testing.T) {
	cleanup := setupRoleVisibilityTestDB(t)
	defer cleanup()

	result, err := GetRoleVisibilityGroupIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestGetRoleVisibilityGroupIDs_Batch(t *testing.T) {
	cleanup := setupRoleVisibilityTestDB(t)
	defer cleanup()

	role1 := OpenClawRole{Name: "角色1", VisibilityType: "group"}
	role2 := OpenClawRole{Name: "角色2", VisibilityType: "group"}
	gdb.Create(&role1)
	gdb.Create(&role2)

	gdb.Create(&RoleVisibilityGroup{OpenClawRoleID: role1.ID, GroupID: 10})
	gdb.Create(&RoleVisibilityGroup{OpenClawRoleID: role1.ID, GroupID: 20})
	gdb.Create(&RoleVisibilityGroup{OpenClawRoleID: role2.ID, GroupID: 30})

	result, err := GetRoleVisibilityGroupIDs(context.Background(), []uint{role1.ID, role2.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result[role1.ID]) != 2 {
		t.Errorf("role1 expected 2 groups, got %d", len(result[role1.ID]))
	}
	if len(result[role2.ID]) != 1 {
		t.Errorf("role2 expected 1 group, got %d", len(result[role2.ID]))
	}
}

// ─── SetRoleVisibility Tests ───

func TestSetRoleVisibility_AllToGroup(t *testing.T) {
	cleanup := setupRoleVisibilityTestDB(t)
	defer cleanup()

	role := OpenClawRole{Name: "测试角色", VisibilityType: "all"}
	gdb.Create(&role)

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return SetRoleVisibility(tx, role.ID, "group", []uint{10, 20})
	})
	if err != nil {
		t.Fatalf("SetRoleVisibility failed: %v", err)
	}

	var updated OpenClawRole
	gdb.First(&updated, role.ID)
	if updated.VisibilityType != "group" {
		t.Errorf("expected visibility_type=group, got %s", updated.VisibilityType)
	}
	var count int64
	gdb.Model(&RoleVisibilityGroup{}).Where("open_claw_role_id = ?", role.ID).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 visibility groups, got %d", count)
	}
}

func TestSetRoleVisibility_GroupToAll(t *testing.T) {
	cleanup := setupRoleVisibilityTestDB(t)
	defer cleanup()

	role := OpenClawRole{Name: "测试角色", VisibilityType: "group"}
	gdb.Create(&role)
	gdb.Create(&RoleVisibilityGroup{OpenClawRoleID: role.ID, GroupID: 10})

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return SetRoleVisibility(tx, role.ID, "all", nil)
	})
	if err != nil {
		t.Fatalf("SetRoleVisibility failed: %v", err)
	}

	var count int64
	gdb.Model(&RoleVisibilityGroup{}).Where("open_claw_role_id = ?", role.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 visibility groups after switching to all, got %d", count)
	}
}

// ─── Cleanup Tests ───

func TestCleanupRoleVisibilityByRoleID(t *testing.T) {
	cleanup := setupRoleVisibilityTestDB(t)
	defer cleanup()

	role := OpenClawRole{Name: "测试角色", VisibilityType: "group"}
	gdb.Create(&role)
	gdb.Create(&RoleVisibilityGroup{OpenClawRoleID: role.ID, GroupID: 10})
	gdb.Create(&RoleVisibilityGroup{OpenClawRoleID: role.ID, GroupID: 20})

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return CleanupRoleVisibilityByRoleID(tx, role.ID)
	})
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	var count int64
	gdb.Model(&RoleVisibilityGroup{}).Where("open_claw_role_id = ?", role.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 records after cleanup, got %d", count)
	}
}

// ─── IsGroupUsedByRoleVisibility Tests ───

func TestIsGroupUsedByRoleVisibility_Used(t *testing.T) {
	cleanup := setupRoleVisibilityTestDB(t)
	defer cleanup()

	gdb.Create(&RoleVisibilityGroup{OpenClawRoleID: 1, GroupID: 10})

	used, err := IsGroupUsedByRoleVisibility(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !used {
		t.Error("group 10 should be reported as used")
	}
}

func TestIsGroupUsedByRoleVisibility_NotUsed(t *testing.T) {
	cleanup := setupRoleVisibilityTestDB(t)
	defer cleanup()

	used, err := IsGroupUsedByRoleVisibility(context.Background(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used {
		t.Error("group 999 should NOT be reported as used")
	}
}
