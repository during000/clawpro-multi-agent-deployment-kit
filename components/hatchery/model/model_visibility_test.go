package model

import (
	"context"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupVisibilityTestDB creates a temporary SQLite database for visibility tests.
func setupVisibilityTestDB(t *testing.T) (cleanup func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "visibility_test_*.db")
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

	if err := gdb.AutoMigrate(&AIModel{}, &ModelVisibilityGroup{}, &UserGroupMember{}); err != nil {
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

// ─── IsModelVisibleToUser Tests ───

func TestIsModelVisibleToUser_AllType(t *testing.T) {
	cleanup := setupVisibilityTestDB(t)
	defer cleanup()

	m := &AIModel{VisibilityType: "all"}
	m.ID = 1

	visible, err := IsModelVisibleToUser(context.Background(), m, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !visible {
		t.Error("expected visible=true for all type")
	}
}

func TestIsModelVisibleToUser_EmptyType(t *testing.T) {
	cleanup := setupVisibilityTestDB(t)
	defer cleanup()

	m := &AIModel{VisibilityType: ""}
	m.ID = 1

	visible, err := IsModelVisibleToUser(context.Background(), m, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !visible {
		t.Error("expected visible=true for empty type (defaults to all)")
	}
}

func TestIsModelVisibleToUser_GroupType_InGroup(t *testing.T) {
	cleanup := setupVisibilityTestDB(t)
	defer cleanup()

	// Create model and visibility association
	m := AIModel{VisibilityType: "group"}
	gdb.Create(&m)

	gdb.Create(&ModelVisibilityGroup{AIModelID: m.ID, GroupID: 1})

	// user 42 belongs to groups 1 and 3
	gdb.Create(&UserGroupMember{UserID: 42, UserGroupID: 1})
	gdb.Create(&UserGroupMember{UserID: 42, UserGroupID: 3})

	visible, err := IsModelVisibleToUser(context.Background(), &m, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !visible {
		t.Error("expected visible=true when user is in the group")
	}
}

func TestIsModelVisibleToUser_GroupType_NotInGroup(t *testing.T) {
	cleanup := setupVisibilityTestDB(t)
	defer cleanup()

	m := AIModel{VisibilityType: "group"}
	gdb.Create(&m)

	gdb.Create(&ModelVisibilityGroup{AIModelID: m.ID, GroupID: 1})
	gdb.Create(&ModelVisibilityGroup{AIModelID: m.ID, GroupID: 3})

	// user 42 belongs to group 5
	gdb.Create(&UserGroupMember{UserID: 42, UserGroupID: 5})

	visible, err := IsModelVisibleToUser(context.Background(), &m, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if visible {
		t.Error("expected visible=false when user is NOT in the group")
	}
}

func TestIsModelVisibleToUser_GroupType_NoGroups(t *testing.T) {
	cleanup := setupVisibilityTestDB(t)
	defer cleanup()

	// user belongs to group 1
	gdb.Create(&UserGroupMember{UserID: 42, UserGroupID: 1})

	m := AIModel{VisibilityType: "group"}
	gdb.Create(&m)
	// No visibility groups created for this model

	visible, err := IsModelVisibleToUser(context.Background(), &m, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if visible {
		t.Error("expected visible=false when model has no associated groups")
	}
}

func TestIsModelVisibleToUser_UserNoGroup(t *testing.T) {
	cleanup := setupVisibilityTestDB(t)
	defer cleanup()

	m := AIModel{VisibilityType: "group"}
	gdb.Create(&m)

	gdb.Create(&ModelVisibilityGroup{AIModelID: m.ID, GroupID: 1})

	visible, err := IsModelVisibleToUser(context.Background(), &m, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if visible {
		t.Error("expected visible=false when user has no groups")
	}
}

func TestIsModelVisibleToUser_MultiGroup_Union(t *testing.T) {
	cleanup := setupVisibilityTestDB(t)
	defer cleanup()

	m := AIModel{VisibilityType: "group"}
	gdb.Create(&m)

	// Model associated with groups 1, 2, 3
	gdb.Create(&ModelVisibilityGroup{AIModelID: m.ID, GroupID: 1})
	gdb.Create(&ModelVisibilityGroup{AIModelID: m.ID, GroupID: 2})
	gdb.Create(&ModelVisibilityGroup{AIModelID: m.ID, GroupID: 3})

	// user 42 belongs to group 3 only
	gdb.Create(&UserGroupMember{UserID: 42, UserGroupID: 3})

	visible, err := IsModelVisibleToUser(context.Background(), &m, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !visible {
		t.Error("expected visible=true when user is in one of multiple groups (union)")
	}
}

// TestIsModelVisibleToUser_GetUserGroupIDsError 验证用户不属于任何分组时模型不可见。
// 注意：GetUserGroupIDs 是普通函数，无法 mock 错误路径；
// 此用例改为验证用户无分组时返回 (false, nil) 的正常逻辑。
func TestIsModelVisibleToUser_GetUserGroupIDsError(t *testing.T) {
	cleanup := setupVisibilityTestDB(t)
	defer cleanup()

	m := AIModel{VisibilityType: "group"}
	gdb.Create(&m)
	gdb.Create(&ModelVisibilityGroup{AIModelID: m.ID, GroupID: 1})
	// 用户 42 不属于任何分组

	visible, err := IsModelVisibleToUser(context.Background(), &m, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if visible {
		t.Error("expected visible=false when user has no groups")
	}
}

// ─── GetModelVisibilityGroupIDs Tests ───

func TestGetModelVisibilityGroupIDs(t *testing.T) {
	cleanup := setupVisibilityTestDB(t)
	defer cleanup()

	// Create associations
	gdb.Create(&ModelVisibilityGroup{AIModelID: 10, GroupID: 1})
	gdb.Create(&ModelVisibilityGroup{AIModelID: 10, GroupID: 3})
	gdb.Create(&ModelVisibilityGroup{AIModelID: 20, GroupID: 5})

	result, err := GetModelVisibilityGroupIDs(context.Background(), []uint{10, 20, 30})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check model 10
	if len(result[10]) != 2 {
		t.Errorf("model 10: expected 2 groups, got %d", len(result[10]))
	}
	// Check model 20
	if len(result[20]) != 1 {
		t.Errorf("model 20: expected 1 group, got %d", len(result[20]))
	}
	// Check model 30 (no associations)
	if len(result[30]) != 0 {
		t.Errorf("model 30: expected 0 groups, got %d", len(result[30]))
	}
}

func TestGetModelVisibilityGroupIDs_Empty(t *testing.T) {
	cleanup := setupVisibilityTestDB(t)
	defer cleanup()

	result, err := GetModelVisibilityGroupIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// ─── Cleanup Functions Tests ───

func TestCleanupVisibilityByGroupID(t *testing.T) {
	cleanup := setupVisibilityTestDB(t)
	defer cleanup()

	// Create associations for different groups
	gdb.Create(&ModelVisibilityGroup{AIModelID: 1, GroupID: 10})
	gdb.Create(&ModelVisibilityGroup{AIModelID: 2, GroupID: 10})
	gdb.Create(&ModelVisibilityGroup{AIModelID: 1, GroupID: 20}) // should not be deleted

	err := CleanupVisibilityByGroupID(gdb, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// group 10 associations should be gone
	var count int64
	gdb.Model(&ModelVisibilityGroup{}).Where("group_id = ?", 10).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 records for group 10, got %d", count)
	}

	// group 20 association should remain
	gdb.Model(&ModelVisibilityGroup{}).Where("group_id = ?", 20).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 record for group 20, got %d", count)
	}
}

func TestCleanupVisibilityByModelID(t *testing.T) {
	cleanup := setupVisibilityTestDB(t)
	defer cleanup()

	// Create associations for different models
	gdb.Create(&ModelVisibilityGroup{AIModelID: 100, GroupID: 1})
	gdb.Create(&ModelVisibilityGroup{AIModelID: 100, GroupID: 2})
	gdb.Create(&ModelVisibilityGroup{AIModelID: 200, GroupID: 1}) // should not be deleted

	err := CleanupVisibilityByModelID(gdb, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// model 100 associations should be gone
	var count int64
	gdb.Model(&ModelVisibilityGroup{}).Where("ai_model_id = ?", 100).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 records for model 100, got %d", count)
	}

	// model 200 association should remain
	gdb.Model(&ModelVisibilityGroup{}).Where("ai_model_id = ?", 200).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 record for model 200, got %d", count)
	}
}
