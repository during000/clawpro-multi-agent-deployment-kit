package model

import (
	"context"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupSkillVisibilityTestDB creates a temporary SQLite database for skill visibility tests.
func setupSkillVisibilityTestDB(t *testing.T) (cleanup func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "skill_visibility_test_*.db")
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

	if err := gdb.AutoMigrate(&Skill{}, &SkillVisibilityGroup{}, &UserGroup{}, &UserGroupMember{}); err != nil {
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

// ─── IsSkillVisibleToUser Tests ───

func TestIsSkillVisibleToUser_AllType(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	s := &Skill{VisibilityType: "all"}
	s.ID = 1

	visible, err := IsSkillVisibleToUser(context.Background(), s, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !visible {
		t.Error("expected visible=true for all type")
	}
}

func TestIsSkillVisibleToUser_EmptyType(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	s := &Skill{VisibilityType: ""}
	s.ID = 1

	visible, err := IsSkillVisibleToUser(context.Background(), s, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !visible {
		t.Error("expected visible=true for empty type (defaults to all)")
	}
}

func TestIsSkillVisibleToUser_GroupType_InGroup(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	// Create user group and membership (real gdb records, no mock needed)
	group := UserGroup{Name: "dev-team"}
	gdb.Create(&group)
	gdb.Create(&UserGroupMember{UserGroupID: group.ID, UserID: 42})

	s := Skill{Slug: "test-skill", Name: "test", Version: "1.0.0", VisibilityType: "group"}
	gdb.Create(&s)
	gdb.Create(&SkillVisibilityGroup{SkillID: s.ID, GroupID: group.ID})

	visible, err := IsSkillVisibleToUser(context.Background(), &s, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !visible {
		t.Error("expected visible=true when user is in the group")
	}
}

func TestIsSkillVisibleToUser_GroupType_NotInGroup(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	// User 42 is in group "other", but skill requires "dev"
	groupDev := UserGroup{Name: "dev"}
	gdb.Create(&groupDev)
	groupOther := UserGroup{Name: "other"}
	gdb.Create(&groupOther)
	gdb.Create(&UserGroupMember{UserGroupID: groupOther.ID, UserID: 42})

	s := Skill{Slug: "test-skill", Name: "test", Version: "1.0.0", VisibilityType: "group"}
	gdb.Create(&s)
	gdb.Create(&SkillVisibilityGroup{SkillID: s.ID, GroupID: groupDev.ID})

	visible, err := IsSkillVisibleToUser(context.Background(), &s, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if visible {
		t.Error("expected visible=false when user is NOT in the group")
	}
}

func TestIsSkillVisibleToUser_UserNoGroup(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	// User 42 has no group memberships
	group := UserGroup{Name: "dev"}
	gdb.Create(&group)

	s := Skill{Slug: "test-skill", Name: "test", Version: "1.0.0", VisibilityType: "group"}
	gdb.Create(&s)
	gdb.Create(&SkillVisibilityGroup{SkillID: s.ID, GroupID: group.ID})

	visible, err := IsSkillVisibleToUser(context.Background(), &s, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if visible {
		t.Error("expected visible=false when user has no groups")
	}
}

func TestIsSkillVisibleToUser_MultiGroup_Union(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	// User 42 is in group3 only; skill requires groups 1,2,3
	group1 := UserGroup{Name: "g1"}
	gdb.Create(&group1)
	group2 := UserGroup{Name: "g2"}
	gdb.Create(&group2)
	group3 := UserGroup{Name: "g3"}
	gdb.Create(&group3)
	gdb.Create(&UserGroupMember{UserGroupID: group3.ID, UserID: 42})

	s := Skill{Slug: "test-skill", Name: "test", Version: "1.0.0", VisibilityType: "group"}
	gdb.Create(&s)
	gdb.Create(&SkillVisibilityGroup{SkillID: s.ID, GroupID: group1.ID})
	gdb.Create(&SkillVisibilityGroup{SkillID: s.ID, GroupID: group2.ID})
	gdb.Create(&SkillVisibilityGroup{SkillID: s.ID, GroupID: group3.ID})

	visible, err := IsSkillVisibleToUser(context.Background(), &s, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !visible {
		t.Error("expected visible=true when user is in one of multiple groups (union)")
	}
}

// ─── GetSkillVisibilityGroupIDs Tests ───

func TestGetSkillVisibilityGroupIDs(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	gdb.Create(&SkillVisibilityGroup{SkillID: 10, GroupID: 1})
	gdb.Create(&SkillVisibilityGroup{SkillID: 10, GroupID: 3})
	gdb.Create(&SkillVisibilityGroup{SkillID: 20, GroupID: 5})

	result, err := GetSkillVisibilityGroupIDs(context.Background(), []uint{10, 20, 30})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result[10]) != 2 {
		t.Errorf("skill 10: expected 2 groups, got %d", len(result[10]))
	}
	if len(result[20]) != 1 {
		t.Errorf("skill 20: expected 1 group, got %d", len(result[20]))
	}
	if len(result[30]) != 0 {
		t.Errorf("skill 30: expected 0 groups, got %d", len(result[30]))
	}
}

func TestGetSkillVisibilityGroupIDs_Empty(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	result, err := GetSkillVisibilityGroupIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// ─── SetSkillVisibility Tests ───

func TestSetSkillVisibility_AllToGroup(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	s := Skill{Slug: "test-skill", Name: "test", Version: "1.0.0", VisibilityType: "all"}
	gdb.Create(&s)

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return SetSkillVisibility(tx, s.ID, "group", []uint{1, 3})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var updated Skill
	gdb.First(&updated, s.ID)
	if updated.VisibilityType != "group" {
		t.Errorf("expected visibility_type=group, got %s", updated.VisibilityType)
	}

	var count int64
	gdb.Model(&SkillVisibilityGroup{}).Where("skill_id = ?", s.ID).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 associations, got %d", count)
	}
}

func TestSetSkillVisibility_GroupToAll(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	s := Skill{Slug: "test-skill", Name: "test", Version: "1.0.0", VisibilityType: "group"}
	gdb.Create(&s)
	gdb.Create(&SkillVisibilityGroup{SkillID: s.ID, GroupID: 1})
	gdb.Create(&SkillVisibilityGroup{SkillID: s.ID, GroupID: 3})

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return SetSkillVisibility(tx, s.ID, "all", nil)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var updated Skill
	gdb.First(&updated, s.ID)
	if updated.VisibilityType != "all" {
		t.Errorf("expected visibility_type=all, got %s", updated.VisibilityType)
	}

	var count int64
	gdb.Model(&SkillVisibilityGroup{}).Where("skill_id = ?", s.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 associations, got %d", count)
	}
}

// ─── CopySkillVisibility Tests ───

func TestCopySkillVisibility_FromGroup(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	old := Skill{Slug: "my-skill", Name: "test", Version: "1.0.0", VersionMajor: 1, VisibilityType: "group"}
	gdb.Create(&old)
	gdb.Create(&SkillVisibilityGroup{SkillID: old.ID, GroupID: 10})
	gdb.Create(&SkillVisibilityGroup{SkillID: old.ID, GroupID: 20})

	newSkill := Skill{Slug: "my-skill", Name: "test", Version: "2.0.0", VersionMajor: 2, VisibilityType: "all"}
	gdb.Create(&newSkill)

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return CopySkillVisibility(tx, "my-skill", newSkill.ID)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var updated Skill
	gdb.First(&updated, newSkill.ID)
	if updated.VisibilityType != "group" {
		t.Errorf("expected visibility_type=group, got %s", updated.VisibilityType)
	}

	var count int64
	gdb.Model(&SkillVisibilityGroup{}).Where("skill_id = ?", newSkill.ID).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 associations, got %d", count)
	}
}

func TestCopySkillVisibility_FromAll(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	old := Skill{Slug: "my-skill", Name: "test", Version: "1.0.0", VersionMajor: 1, VisibilityType: "all"}
	gdb.Create(&old)

	newSkill := Skill{Slug: "my-skill", Name: "test", Version: "2.0.0", VersionMajor: 2, VisibilityType: "all"}
	gdb.Create(&newSkill)

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return CopySkillVisibility(tx, "my-skill", newSkill.ID)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var updated Skill
	gdb.First(&updated, newSkill.ID)
	if updated.VisibilityType != "all" {
		t.Errorf("expected visibility_type=all, got %s", updated.VisibilityType)
	}

	var count int64
	gdb.Model(&SkillVisibilityGroup{}).Where("skill_id = ?", newSkill.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 associations, got %d", count)
	}
}

func TestCopySkillVisibility_NoOldVersion(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	newSkill := Skill{Slug: "brand-new-skill", Name: "test", Version: "1.0.0", VersionMajor: 1, VisibilityType: "all"}
	gdb.Create(&newSkill)

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return CopySkillVisibility(tx, "brand-new-skill", newSkill.ID)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var updated Skill
	gdb.First(&updated, newSkill.ID)
	if updated.VisibilityType != "all" {
		t.Errorf("expected visibility_type=all, got %s", updated.VisibilityType)
	}
}

// ─── Cleanup Functions Tests ───

func TestCleanupSkillVisibilityByGroupID(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	gdb.Create(&SkillVisibilityGroup{SkillID: 1, GroupID: 10})
	gdb.Create(&SkillVisibilityGroup{SkillID: 2, GroupID: 10})
	gdb.Create(&SkillVisibilityGroup{SkillID: 1, GroupID: 20})

	err := CleanupSkillVisibilityByGroupID(gdb, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int64
	gdb.Model(&SkillVisibilityGroup{}).Where("group_id = ?", 10).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 records for group 10, got %d", count)
	}

	gdb.Model(&SkillVisibilityGroup{}).Where("group_id = ?", 20).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 record for group 20, got %d", count)
	}
}

func TestCleanupSkillVisibilityBySkillID(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	gdb.Create(&SkillVisibilityGroup{SkillID: 100, GroupID: 1})
	gdb.Create(&SkillVisibilityGroup{SkillID: 100, GroupID: 2})
	gdb.Create(&SkillVisibilityGroup{SkillID: 200, GroupID: 1})

	err := CleanupSkillVisibilityBySkillID(gdb, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int64
	gdb.Model(&SkillVisibilityGroup{}).Where("skill_id = ?", 100).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 records for skill 100, got %d", count)
	}

	gdb.Model(&SkillVisibilityGroup{}).Where("skill_id = ?", 200).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 record for skill 200, got %d", count)
	}
}

// ─── IsGroupUsedBySkillVisibility Tests ───

func TestIsGroupUsedBySkillVisibility_Used(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	gdb.Create(&SkillVisibilityGroup{SkillID: 1, GroupID: 10})

	used, err := IsGroupUsedBySkillVisibility(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !used {
		t.Error("expected used=true when group is referenced")
	}
}

func TestIsGroupUsedBySkillVisibility_NotUsed(t *testing.T) {
	cleanup := setupSkillVisibilityTestDB(t)
	defer cleanup()

	used, err := IsGroupUsedBySkillVisibility(context.Background(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used {
		t.Error("expected used=false when group is not referenced")
	}
}
