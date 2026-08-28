package model

import (
	"context"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupBundleVisibilityTestDB creates a temporary SQLite database for skill bundle visibility tests.
func setupBundleVisibilityTestDB(t *testing.T) (cleanup func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "bundle_visibility_test_*.db")
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

	if err := gdb.AutoMigrate(&SkillBundle{}, &SkillBundleVisibilityGroup{}, &UserGroup{}, &UserGroupMember{}); err != nil {
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

// ─── IsSkillBundleVisibleToUser Tests ───

func TestIsSkillBundleVisibleToUser_AllType(t *testing.T) {
	cleanup := setupBundleVisibilityTestDB(t)
	defer cleanup()

	bundle := SkillBundle{Name: "全局包", VisibilityType: "all", Enabled: true}
	gdb.Create(&bundle)

	visible, err := IsSkillBundleVisibleToUser(context.Background(), &bundle, 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !visible {
		t.Error("all-type bundle should be visible to any user")
	}
}

func TestIsSkillBundleVisibleToUser_GroupType_Match(t *testing.T) {
	cleanup := setupBundleVisibilityTestDB(t)
	defer cleanup()

	group := UserGroup{Name: "研发"}
	gdb.Create(&group)
	gdb.Create(&UserGroupMember{UserGroupID: group.ID, UserID: 1})

	bundle := SkillBundle{Name: "研发包", VisibilityType: "group", Enabled: true}
	gdb.Create(&bundle)
	gdb.Create(&SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: group.ID})

	visible, err := IsSkillBundleVisibleToUser(context.Background(), &bundle, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !visible {
		t.Error("user in matching group should see the bundle")
	}
}

func TestIsSkillBundleVisibleToUser_GroupType_NoMatch(t *testing.T) {
	cleanup := setupBundleVisibilityTestDB(t)
	defer cleanup()

	groupA := UserGroup{Name: "研发"}
	gdb.Create(&groupA)
	groupB := UserGroup{Name: "市场"}
	gdb.Create(&groupB)
	gdb.Create(&UserGroupMember{UserGroupID: groupA.ID, UserID: 1})

	bundle := SkillBundle{Name: "市场包", VisibilityType: "group", Enabled: true}
	gdb.Create(&bundle)
	gdb.Create(&SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: groupB.ID})

	visible, err := IsSkillBundleVisibleToUser(context.Background(), &bundle, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if visible {
		t.Error("user not in matching group should NOT see the bundle")
	}
}

func TestIsSkillBundleVisibleToUser_GroupType_UserNoGroups(t *testing.T) {
	cleanup := setupBundleVisibilityTestDB(t)
	defer cleanup()

	group := UserGroup{Name: "研发"}
	gdb.Create(&group)

	bundle := SkillBundle{Name: "研发包", VisibilityType: "group", Enabled: true}
	gdb.Create(&bundle)
	gdb.Create(&SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: group.ID})

	visible, err := IsSkillBundleVisibleToUser(context.Background(), &bundle, 999) // user 999 has no groups
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if visible {
		t.Error("user with no groups should NOT see group-type bundle")
	}
}

// ─── GetSkillBundleVisibilityGroupIDs Tests ───

func TestGetSkillBundleVisibilityGroupIDs_Empty(t *testing.T) {
	cleanup := setupBundleVisibilityTestDB(t)
	defer cleanup()

	result, err := GetSkillBundleVisibilityGroupIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestGetSkillBundleVisibilityGroupIDs_Batch(t *testing.T) {
	cleanup := setupBundleVisibilityTestDB(t)
	defer cleanup()

	bundle1 := SkillBundle{Name: "包1", VisibilityType: "group"}
	bundle2 := SkillBundle{Name: "包2", VisibilityType: "group"}
	gdb.Create(&bundle1)
	gdb.Create(&bundle2)

	gdb.Create(&SkillBundleVisibilityGroup{SkillBundleID: bundle1.ID, GroupID: 10})
	gdb.Create(&SkillBundleVisibilityGroup{SkillBundleID: bundle1.ID, GroupID: 20})
	gdb.Create(&SkillBundleVisibilityGroup{SkillBundleID: bundle2.ID, GroupID: 30})

	result, err := GetSkillBundleVisibilityGroupIDs(context.Background(), []uint{bundle1.ID, bundle2.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result[bundle1.ID]) != 2 {
		t.Errorf("bundle1 expected 2 groups, got %d", len(result[bundle1.ID]))
	}
	if len(result[bundle2.ID]) != 1 {
		t.Errorf("bundle2 expected 1 group, got %d", len(result[bundle2.ID]))
	}
}

// ─── SetSkillBundleVisibility Tests ───

func TestSetSkillBundleVisibility_AllToGroup(t *testing.T) {
	cleanup := setupBundleVisibilityTestDB(t)
	defer cleanup()

	bundle := SkillBundle{Name: "测试包", VisibilityType: "all"}
	gdb.Create(&bundle)

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return SetSkillBundleVisibility(tx, bundle.ID, "group", []uint{10, 20})
	})
	if err != nil {
		t.Fatalf("SetSkillBundleVisibility failed: %v", err)
	}

	// Verify
	var updated SkillBundle
	gdb.First(&updated, bundle.ID)
	if updated.VisibilityType != "group" {
		t.Errorf("expected visibility_type=group, got %s", updated.VisibilityType)
	}
	var count int64
	gdb.Model(&SkillBundleVisibilityGroup{}).Where("skill_bundle_id = ?", bundle.ID).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 visibility groups, got %d", count)
	}
}

func TestSetSkillBundleVisibility_GroupToAll(t *testing.T) {
	cleanup := setupBundleVisibilityTestDB(t)
	defer cleanup()

	bundle := SkillBundle{Name: "测试包", VisibilityType: "group"}
	gdb.Create(&bundle)
	gdb.Create(&SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: 10})

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return SetSkillBundleVisibility(tx, bundle.ID, "all", nil)
	})
	if err != nil {
		t.Fatalf("SetSkillBundleVisibility failed: %v", err)
	}

	// Verify associations cleared
	var count int64
	gdb.Model(&SkillBundleVisibilityGroup{}).Where("skill_bundle_id = ?", bundle.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 visibility groups after switching to all, got %d", count)
	}
}

// ─── Cleanup Tests ───

func TestCleanupSkillBundleVisibilityByBundleID(t *testing.T) {
	cleanup := setupBundleVisibilityTestDB(t)
	defer cleanup()

	bundle := SkillBundle{Name: "测试包", VisibilityType: "group"}
	gdb.Create(&bundle)
	gdb.Create(&SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: 10})
	gdb.Create(&SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: 20})

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return CleanupSkillBundleVisibilityByBundleID(tx, bundle.ID)
	})
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	var count int64
	gdb.Model(&SkillBundleVisibilityGroup{}).Where("skill_bundle_id = ?", bundle.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 records after cleanup, got %d", count)
	}
}

// ─── IsGroupUsedBySkillBundleVisibility Tests ───

func TestIsGroupUsedBySkillBundleVisibility_Used(t *testing.T) {
	cleanup := setupBundleVisibilityTestDB(t)
	defer cleanup()

	gdb.Create(&SkillBundleVisibilityGroup{SkillBundleID: 1, GroupID: 10})

	used, err := IsGroupUsedBySkillBundleVisibility(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !used {
		t.Error("group 10 should be reported as used")
	}
}

func TestIsGroupUsedBySkillBundleVisibility_NotUsed(t *testing.T) {
	cleanup := setupBundleVisibilityTestDB(t)
	defer cleanup()

	used, err := IsGroupUsedBySkillBundleVisibility(context.Background(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used {
		t.Error("group 999 should NOT be reported as used")
	}
}
