package model

import (
	"context"
	"os"
	"testing"

	"hatchery/common"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupUserGroupTestDB 创建只含 UserGroup 相关表的临时 SQLite DB。
func setupUserGroupTestDB(t *testing.T) (db *gorm.DB, cleanup func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "hatchery_ug_test_*.db")
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
	registerIdentifierCallbacks(gdb)

	migrateDB := gdb.WithContext(common.WithSkipIdentifier(context.Background()))
	if err := migrateDB.AutoMigrate(
		&UserGroup{},
		&UserGroupMember{},
	); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("auto migrate: %v", err)
	}

	return testDB, func() {
		sqlDB, _ := gdb.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
		os.Remove(tmpFile.Name() + "-wal")
		os.Remove(tmpFile.Name() + "-shm")
		gdb = origDB
	}
}

// ---- GetUserGroupName / GetUserGroupNameWithDB ----

func TestGetUserGroupName_HappyPath(t *testing.T) {
	_, cleanup := setupUserGroupTestDB(t)
	defer cleanup()
	ctx := common.WithSkipIdentifier(context.Background())

	g := &UserGroup{Name: "DevTeam", Source: GroupSourceOneIDDept}
	if err := DB(ctx).Create(g).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	got := GetUserGroupName(ctx, g.ID)
	if got != "DevTeam" {
		t.Errorf("got=%q, want=DevTeam", got)
	}
}

func TestGetUserGroupName_NotFound(t *testing.T) {
	_, cleanup := setupUserGroupTestDB(t)
	defer cleanup()

	got := GetUserGroupName(common.WithSkipIdentifier(context.Background()), 99999)
	if got != "" {
		t.Errorf("not found should return empty string, got=%q", got)
	}
}

func TestGetUserGroupName_GroupIDZero(t *testing.T) {
	_, cleanup := setupUserGroupTestDB(t)
	defer cleanup()

	got := GetUserGroupName(common.WithSkipIdentifier(context.Background()), 0)
	if got != "" {
		t.Errorf("groupID=0 should return empty string, got=%q", got)
	}
}

func TestGetUserGroupNameWithDB_HappyPath(t *testing.T) {
	db, cleanup := setupUserGroupTestDB(t)
	defer cleanup()
	ctx := common.WithSkipIdentifier(context.Background())
	tx := db.WithContext(ctx)

	g := &UserGroup{Name: "OpsTeam", Source: GroupSourceOneIDDept}
	if err := tx.Create(g).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	got, err := GetUserGroupNameWithDB(tx, g.ID)
	if err != nil {
		t.Fatalf("GetUserGroupNameWithDB: %v", err)
	}
	if got != "OpsTeam" {
		t.Errorf("got=%q, want=OpsTeam", got)
	}
	got, err = GetUserGroupNameWithDB(tx, g.ID+1000)
	if err != nil || got != "" {
		t.Errorf("missing group: got=%q err=%v, want empty nil", got, err)
	}
}

func TestGetUserGroupNameWithDB_ReturnsDatabaseError(t *testing.T) {
	db, cleanup := setupUserGroupTestDB(t)
	defer cleanup()
	tx := db.WithContext(common.WithSkipIdentifier(context.Background()))
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql DB: %v", err)
	}
	if got, err := GetUserGroupNameWithDB(tx, 1); err == nil || got != "" {
		t.Fatalf("closed DB got=%q err=%v, want empty and non-nil error", got, err)
	}
}

// ---- GetUserGroupIDs / GetUserGroupIDsWithDB ----

func TestGetUserGroupIDs_HappyPath(t *testing.T) {
	_, cleanup := setupUserGroupTestDB(t)
	defer cleanup()
	ctx := common.WithSkipIdentifier(context.Background())

	g1 := &UserGroup{Name: "G1", Source: GroupSourceOneIDDept}
	g2 := &UserGroup{Name: "G2", Source: GroupSourceManual}
	DB(ctx).Create(g1)
	DB(ctx).Create(g2)

	DB(ctx).Create(&UserGroupMember{UserGroupID: g1.ID, UserID: 1, Source: MemberSourceOneIDDept})
	DB(ctx).Create(&UserGroupMember{UserGroupID: g2.ID, UserID: 1, Source: MemberSourceManual})

	ids, err := GetUserGroupIDs(ctx, 1)
	if err != nil {
		t.Fatalf("GetUserGroupIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 group IDs, got %d: %v", len(ids), ids)
	}
}

func TestGetUserGroupIDs_NoGroups(t *testing.T) {
	_, cleanup := setupUserGroupTestDB(t)
	defer cleanup()

	ids, err := GetUserGroupIDs(common.WithSkipIdentifier(context.Background()), 99999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 group IDs, got %d: %v", len(ids), ids)
	}
}

func TestGetUserGroupIDsWithDB_HappyPath(t *testing.T) {
	db, cleanup := setupUserGroupTestDB(t)
	defer cleanup()
	ctx := common.WithSkipIdentifier(context.Background())
	tx := db.WithContext(ctx)

	g := &UserGroup{Name: "G1", Source: GroupSourceOneIDDept}
	tx.Create(g)
	tx.Create(&UserGroupMember{UserGroupID: g.ID, UserID: 42, Source: MemberSourceOneIDDept})

	ids, err := GetUserGroupIDsWithDB(tx, 42)
	if err != nil {
		t.Fatalf("GetUserGroupIDsWithDB: %v", err)
	}
	if len(ids) != 1 || ids[0] != g.ID {
		t.Errorf("expected [%d], got %v", g.ID, ids)
	}
}
