package usergroup

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAncestorCoverageDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.UserGroup{}, &model.GroupClosure{}, &model.UserGroupMember{}, &model.User{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

// ── GetAncestorIDs ───────────────────────────────────────

func TestCoverageGetAncestorIDs_ZeroID(t *testing.T) {
	setupAncestorCoverageDB(t)

	ids, err := GetAncestorIDs(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetAncestorIDs(context.Background(), 0): %v", err)
	}
	if ids != nil {
		t.Errorf("expected nil for groupID=0, got %v", ids)
	}
}

func TestCoverageGetAncestorIDs_RootGroup(t *testing.T) {
	setupAncestorCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual"})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})

	ids, err := GetAncestorIDs(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetAncestorIDs: %v", err)
	}
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("expected [1], got %v", ids)
	}
}

func TestCoverageGetAncestorIDs_ChildGroup(t *testing.T) {
	setupAncestorCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{ID: 2, Name: "Child", Source: "manual", ParentID: 1})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 2, DescendantID: 2, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})

	ids, err := GetAncestorIDs(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetAncestorIDs child: %v", err)
	}
	// 含自己，按 depth ASC: [2, 1]
	if len(ids) != 2 {
		t.Errorf("expected 2 ancestors, got %d: %v", len(ids), ids)
	}
	if ids[0] != 2 {
		t.Errorf("ids[0] should be self(2), got %d", ids[0])
	}
}

// ── GetAllAncestorIDs ────────────────────────────────────

func TestCoverageGetAllAncestorIDs_Empty(t *testing.T) {
	setupAncestorCoverageDB(t)

	ids, err := GetAllAncestorIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetAllAncestorIDs nil: %v", err)
	}
	if ids != nil {
		t.Errorf("expected nil, got %v", ids)
	}
}

func TestCoverageGetAllAncestorIDs_MultipleGroups(t *testing.T) {
	setupAncestorCoverageDB(t)

	// Root(1) → A(2), Root(1) → B(3)
	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{ID: 2, Name: "A", Source: "manual", ParentID: 1})
	model.DB(context.Background()).Create(&model.UserGroup{ID: 3, Name: "B", Source: "manual", ParentID: 1})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 2, DescendantID: 2, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 3, DescendantID: 3, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 3, Depth: 1})

	ids, err := GetAllAncestorIDs(context.Background(), []uint{2, 3})
	if err != nil {
		t.Fatalf("GetAllAncestorIDs: %v", err)
	}
	// Union: {1, 2, 3}
	if len(ids) != 3 {
		t.Errorf("expected 3 unique, got %d: %v", len(ids), ids)
	}
}

func TestCoverageGetAllAncestorIDs_Deduplication(t *testing.T) {
	setupAncestorCoverageDB(t)

	// Root(1) → A(2) → C(4), Root(1) → B(3) → C(4) would be complex
	// Simpler: Root(1) → A(2); pass [2, 2] should dedup
	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{ID: 2, Name: "A", Source: "manual", ParentID: 1})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 2, DescendantID: 2, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})

	ids, err := GetAllAncestorIDs(context.Background(), []uint{2, 2})
	if err != nil {
		t.Fatalf("GetAllAncestorIDs dedup: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 (deduped), got %d: %v", len(ids), ids)
	}
}

// ── GetUserAllGroupAndAncestorIDs ────────────────────────

func TestCoverageGetUserAllGroupAndAncestorIDs_NoGroups(t *testing.T) {
	setupAncestorCoverageDB(t)

	ids, err := GetUserAllGroupAndAncestorIDs(context.Background(), 9999)
	if err != nil {
		t.Fatalf("GetUserAllGroupAndAncestorIDs no groups: %v", err)
	}
	if ids != nil {
		t.Errorf("expected nil, got %v", ids)
	}
}

func TestCoverageGetUserAllGroupAndAncestorIDs_WithGroups(t *testing.T) {
	setupAncestorCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{ID: 2, Name: "Child", Source: "manual", ParentID: 1})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 2, DescendantID: 2, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})

	model.DB(context.Background()).Create(&model.User{Username: "ancestor_user", Password: "x"})
	var user model.User
	model.DB(context.Background()).First(&user, "username = ?", "ancestor_user")
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: 2, UserID: user.ID, Source: "manual"})

	ids, err := GetUserAllGroupAndAncestorIDs(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetUserAllGroupAndAncestorIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 (child + root), got %d: %v", len(ids), ids)
	}
}
