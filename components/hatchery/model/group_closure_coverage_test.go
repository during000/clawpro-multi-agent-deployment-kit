package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupClosureCoverageDB 创建内存 SQLite 数据库，用于 closure 覆盖测试。
func setupClosureCoverageDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&UserGroup{}, &GroupClosure{}, &UserGroupMember{}, &GroupConfigBinding{}, &User{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	gdb = db
	t.Cleanup(func() { gdb = nil })
	return db
}

// ── closureInsertForNewChildTx ────────────────────────

func TestCoverageClosureInsertForNewChild_RootGroup(t *testing.T) {
	db := setupClosureCoverageDB(t)

	// 自指行
	err := db.Transaction(func(tx *gorm.DB) error {
		return closureInsertForNewChildTx(tx, 100, 0)
	})
	if err != nil {
		t.Fatalf("closureInsertForNewChildTx root: %v", err)
	}

	var rows []GroupClosure
	db.Where("descendant_id = ?", 100).Find(&rows)
	if len(rows) != 1 {
		t.Errorf("root group: expected 1 row, got %d", len(rows))
	}
	if rows[0].AncestorID != 100 || rows[0].Depth != 0 {
		t.Errorf("unexpected row: %+v", rows[0])
	}
}

func TestCoverageClosureInsertForNewChild_WithParent(t *testing.T) {
	db := setupClosureCoverageDB(t)

	// 先建父组的 closure
	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})

	err := db.Transaction(func(tx *gorm.DB) error {
		return closureInsertForNewChildTx(tx, 2, 1)
	})
	if err != nil {
		t.Fatalf("closureInsertForNewChildTx child: %v", err)
	}

	var rows []GroupClosure
	db.Where("descendant_id = ?", 2).Find(&rows)
	// 应有：自指(2,2,0) + 继承(1,2,1)
	if len(rows) != 2 {
		t.Errorf("child: expected 2 rows, got %d", len(rows))
	}
}

func TestCoverageClosureInsertForNewChild_DeepHierarchy(t *testing.T) {
	db := setupClosureCoverageDB(t)

	// 建三层：1 → 2 → 3
	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	db.Transaction(func(tx *gorm.DB) error {
		return closureInsertForNewChildTx(tx, 2, 1)
	})
	db.Transaction(func(tx *gorm.DB) error {
		return closureInsertForNewChildTx(tx, 3, 2)
	})

	var rows []GroupClosure
	db.Where("descendant_id = ?", 3).Find(&rows)
	// 应有：自指(3,3,0) + (2,3,1) + (1,3,2)
	if len(rows) != 3 {
		t.Errorf("deep hierarchy: expected 3 rows, got %d", len(rows))
	}
}

// ── closureMoveSubtreeTx ──────────────────────────────

func TestCoverageClosureMoveSubtree_ToNewParent(t *testing.T) {
	db := setupClosureCoverageDB(t)

	// A(1) → B(2), C(3) 独立根
	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	db.Create(&GroupClosure{AncestorID: 2, DescendantID: 2, Depth: 0})
	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})
	db.Create(&GroupClosure{AncestorID: 3, DescendantID: 3, Depth: 0})

	// 把 B(2) 从 A(1) 移到 C(3) 下
	err := db.Transaction(func(tx *gorm.DB) error {
		return closureMoveSubtreeTx(tx, 2, 3)
	})
	if err != nil {
		t.Fatalf("closureMoveSubtreeTx: %v", err)
	}

	// B 的祖先应是 C 而不是 A
	var rows []GroupClosure
	db.Where("descendant_id = ? AND ancestor_id <> ?", 2, 2).Find(&rows)
	if len(rows) != 1 {
		t.Errorf("expected 1 ancestor for B, got %d", len(rows))
		return
	}
	if rows[0].AncestorID != 3 {
		t.Errorf("B's ancestor should be 3, got %d", rows[0].AncestorID)
	}
}

func TestCoverageClosureMoveSubtree_ToRoot(t *testing.T) {
	db := setupClosureCoverageDB(t)

	// A(1) → B(2)
	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	db.Create(&GroupClosure{AncestorID: 2, DescendantID: 2, Depth: 0})
	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})

	// 把 B(2) 移到根（newParentID=0）
	err := db.Transaction(func(tx *gorm.DB) error {
		return closureMoveSubtreeTx(tx, 2, 0)
	})
	if err != nil {
		t.Fatalf("closureMoveSubtreeTx to root: %v", err)
	}

	// B 应只有自指行
	var rows []GroupClosure
	db.Where("descendant_id = ?", 2).Find(&rows)
	if len(rows) != 1 {
		t.Errorf("expected 1 row (self), got %d", len(rows))
	}
}

// ── closureDeleteNodeTx ───────────────────────────────

func TestCoverageClosureDeleteNode(t *testing.T) {
	db := setupClosureCoverageDB(t)

	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	db.Create(&GroupClosure{AncestorID: 2, DescendantID: 2, Depth: 0})
	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})

	err := db.Transaction(func(tx *gorm.DB) error {
		return closureDeleteNodeTx(tx, 2)
	})
	if err != nil {
		t.Fatalf("closureDeleteNodeTx: %v", err)
	}

	var count int64
	db.Model(&GroupClosure{}).Where("ancestor_id = ? OR descendant_id = ?", 2, 2).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 rows for deleted node, got %d", count)
	}
}

// ── closureIsDescendantTx ─────────────────────────────

func TestCoverageClosureIsDescendant_True(t *testing.T) {
	db := setupClosureCoverageDB(t)

	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})
	db.Create(&GroupClosure{AncestorID: 2, DescendantID: 2, Depth: 0})

	ok, err := closureIsDescendantTx(db, 1, 2)
	if err != nil {
		t.Fatalf("closureIsDescendantTx: %v", err)
	}
	if !ok {
		t.Error("expected 2 is descendant of 1")
	}
}

func TestCoverageClosureIsDescendant_False(t *testing.T) {
	db := setupClosureCoverageDB(t)

	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	db.Create(&GroupClosure{AncestorID: 2, DescendantID: 2, Depth: 0})

	ok, err := closureIsDescendantTx(db, 1, 2)
	if err != nil {
		t.Fatalf("closureIsDescendantTx: %v", err)
	}
	if ok {
		t.Error("expected 2 is NOT descendant of 1")
	}
}

func TestCoverageClosureIsDescendant_Self(t *testing.T) {
	db := setupClosureCoverageDB(t)

	db.Create(&GroupClosure{AncestorID: 5, DescendantID: 5, Depth: 0})

	ok, err := closureIsDescendantTx(db, 5, 5)
	if err != nil {
		t.Fatalf("closureIsDescendantTx: %v", err)
	}
	if !ok {
		t.Error("node should be descendant of itself")
	}
}

// ── closureMaxRelativeDepthTx ─────────────────────────

func TestCoverageClosureMaxRelativeDepth_NoDescendants(t *testing.T) {
	db := setupClosureCoverageDB(t)

	db.Create(&GroupClosure{AncestorID: 10, DescendantID: 10, Depth: 0})

	depth, err := closureMaxRelativeDepthTx(db, 10)
	if err != nil {
		t.Fatalf("closureMaxRelativeDepthTx: %v", err)
	}
	if depth != 0 {
		t.Errorf("expected 0, got %d", depth)
	}
}

func TestCoverageClosureMaxRelativeDepth_WithDescendants(t *testing.T) {
	db := setupClosureCoverageDB(t)

	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})
	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 3, Depth: 2})

	depth, err := closureMaxRelativeDepthTx(db, 1)
	if err != nil {
		t.Fatalf("closureMaxRelativeDepthTx: %v", err)
	}
	if depth != 2 {
		t.Errorf("expected 2, got %d", depth)
	}
}

// ── closureDescendantsDB ──────────────────────────────

func TestCoverageClosureDescendantsDB_EmptyResult(t *testing.T) {
	db := setupClosureCoverageDB(t)

	ids, err := closureDescendantsDB(db, 999, true)
	if err != nil {
		t.Fatalf("closureDescendantsDB: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected empty, got %v", ids)
	}
}

// ── ReconcileClosure 额外用例 ──────────────────────────

func TestCoverageReconcileClosure_NilDB(t *testing.T) {
	origDB := gdb
	gdb = nil
	defer func() { gdb = origDB }()

	// 不应 panic
	ReconcileClosure(context.Background())
}

func TestCoverageReconcileClosure_ExtraRows(t *testing.T) {
	setupClosureCoverageDB(t)

	// 建一个根组
	gdb.Create(&UserGroup{ID: 1, Name: "Root", Source: GroupSourceManual})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	// 插入一条多余行
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 99, Depth: 1})

	ReconcileClosure(context.Background())

	// 重建后应只有 1 条自指行
	var count int64
	gdb.Model(&GroupClosure{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 row after reconcile, got %d", count)
	}
}
