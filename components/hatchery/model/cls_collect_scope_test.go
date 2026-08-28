package model

import (
	"context"
	"os"
	"sort"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupCLSScopeTestDB 创建临时 SQLite 数据库并 migrate 相关表。
func setupCLSScopeTestDB(t *testing.T) (cleanup func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "cls_scope_test_*.db")
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

	if err := testDB.AutoMigrate(
		&GroupConfigBinding{},
		&GroupClosure{},
		&UserGroupMember{},
		&UserGroup{},
		&Instance{},
		&SiteConfig{},
	); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("auto migrate: %v", err)
	}

	return func() {
		sqlDB, _ := testDB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
		os.Remove(tmpFile.Name() + "-wal")
		os.Remove(tmpFile.Name() + "-shm")
		gdb = origDB
	}
}

// ─── GetCLSCollectScopeGroupIDs ─────────────────────────────────────

func TestGetCLSCollectScopeGroupIDs_Empty(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	ids, err := GetCLSCollectScopeGroupIDs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected empty, got %v", ids)
	}
}

func TestGetCLSCollectScopeGroupIDs_WithData(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{10, 20, 30})

	ids, err := GetCLSCollectScopeGroupIDs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 ids, got %d", len(ids))
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if ids[0] != 10 || ids[1] != 20 || ids[2] != 30 {
		t.Errorf("unexpected ids: %v", ids)
	}
}

// ─── SetCLSCollectScope ─────────────────────────────────────

func TestSetCLSCollectScope_EmptyToNonEmpty(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	if err := SetCLSCollectScope(context.Background(), []uint{1, 2, 3}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 3 {
		t.Errorf("expected 3 ids, got %d: %v", len(ids), ids)
	}
}

func TestSetCLSCollectScope_ReplaceExisting(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1, 2, 3})
	if err := SetCLSCollectScope(context.Background(), []uint{4, 5}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(ids))
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if ids[0] != 4 || ids[1] != 5 {
		t.Errorf("unexpected ids: %v", ids)
	}
}

func TestSetCLSCollectScope_ClearWithEmpty(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1, 2, 3})
	if err := SetCLSCollectScope(context.Background(), []uint{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 0 {
		t.Errorf("expected empty after clear, got %v", ids)
	}
}

func TestSetCLSCollectScope_NilSlice(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1, 2})
	if err := SetCLSCollectScope(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 0 {
		t.Errorf("expected empty after nil set, got %v", ids)
	}
}

// ─── AddCLSCollectScopeGroups ─────────────────────────────────────

func TestAddCLSCollectScopeGroups_Empty(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	if err := AddCLSCollectScopeGroups(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 0 {
		t.Errorf("expected empty, got %v", ids)
	}
}

func TestAddCLSCollectScopeGroups_NewGroups(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	if err := AddCLSCollectScopeGroups(context.Background(), []uint{10, 20}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 2 {
		t.Errorf("expected 2, got %d", len(ids))
	}
}

func TestAddCLSCollectScopeGroups_Idempotent(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	AddCLSCollectScopeGroups(context.Background(), []uint{10, 20})
	if err := AddCLSCollectScopeGroups(context.Background(), []uint{20, 30}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 3 {
		t.Fatalf("expected 3 (idempotent), got %d: %v", len(ids), ids)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if ids[0] != 10 || ids[1] != 20 || ids[2] != 30 {
		t.Errorf("unexpected ids: %v", ids)
	}
}

// ─── RemoveCLSCollectScopeGroups ─────────────────────────────────────

func TestRemoveCLSCollectScopeGroups_Empty(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	if err := RemoveCLSCollectScopeGroups(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveCLSCollectScopeGroups_RemoveExisting(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1, 2, 3, 4})
	if err := RemoveCLSCollectScopeGroups(context.Background(), []uint{2, 4}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 2 {
		t.Fatalf("expected 2, got %d", len(ids))
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if ids[0] != 1 || ids[1] != 3 {
		t.Errorf("unexpected remaining ids: %v", ids)
	}
}

func TestRemoveCLSCollectScopeGroups_NonExisting(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1, 2})
	if err := RemoveCLSCollectScopeGroups(context.Background(), []uint{99, 100}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 2 {
		t.Errorf("should not affect existing, got %v", ids)
	}
}

// ─── ClearCLSCollectScope ─────────────────────────────────────

func TestClearCLSCollectScope(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1, 2, 3})
	if err := ClearCLSCollectScope(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 0 {
		t.Errorf("expected empty after clear, got %v", ids)
	}
}

func TestClearCLSCollectScope_AlreadyEmpty(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	if err := ClearCLSCollectScope(context.Background()); err != nil {
		t.Fatalf("unexpected error on empty scope: %v", err)
	}
}

// ─── DiffCLSCollectScope ─────────────────────────────────────

func TestDiffCLSCollectScope_EmptyToNew(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	added, removed, err := DiffCLSCollectScope(context.Background(), []uint{1, 2, 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("expected no removed, got %v", removed)
	}
	sort.Slice(added, func(i, j int) bool { return added[i] < added[j] })
	if len(added) != 3 || added[0] != 1 || added[1] != 2 || added[2] != 3 {
		t.Errorf("expected added=[1,2,3], got %v", added)
	}
}

func TestDiffCLSCollectScope_PartialOverlap(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1, 2, 3})

	added, removed, err := DiffCLSCollectScope(context.Background(), []uint{2, 3, 4, 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Slice(added, func(i, j int) bool { return added[i] < added[j] })
	sort.Slice(removed, func(i, j int) bool { return removed[i] < removed[j] })

	if len(added) != 2 || added[0] != 4 || added[1] != 5 {
		t.Errorf("expected added=[4,5], got %v", added)
	}
	if len(removed) != 1 || removed[0] != 1 {
		t.Errorf("expected removed=[1], got %v", removed)
	}
}

func TestDiffCLSCollectScope_ClearAll(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1, 2, 3})

	added, removed, err := DiffCLSCollectScope(context.Background(), []uint{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(added) != 0 {
		t.Errorf("expected no added, got %v", added)
	}
	sort.Slice(removed, func(i, j int) bool { return removed[i] < removed[j] })
	if len(removed) != 3 || removed[0] != 1 || removed[1] != 2 || removed[2] != 3 {
		t.Errorf("expected removed=[1,2,3], got %v", removed)
	}
}

func TestDiffCLSCollectScope_NoChange(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1, 2, 3})

	added, removed, err := DiffCLSCollectScope(context.Background(), []uint{1, 2, 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(added) != 0 {
		t.Errorf("expected no added, got %v", added)
	}
	if len(removed) != 0 {
		t.Errorf("expected no removed, got %v", removed)
	}
}

// ─── DiffAndSetCLSCollectScope（事务版）─────────────────────────────

func TestDiffAndSetCLSCollectScope_EmptyToNew(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	added, removed, err := DiffAndSetCLSCollectScope(context.Background(), []uint{1, 2, 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("expected no removed, got %v", removed)
	}
	sort.Slice(added, func(i, j int) bool { return added[i] < added[j] })
	if len(added) != 3 || added[0] != 1 || added[1] != 2 || added[2] != 3 {
		t.Errorf("expected added=[1,2,3], got %v", added)
	}

	// 验证数据库已更新
	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Errorf("expected DB=[1,2,3], got %v", ids)
	}
}

func TestDiffAndSetCLSCollectScope_PartialOverlap(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1, 2, 3})

	added, removed, err := DiffAndSetCLSCollectScope(context.Background(), []uint{2, 3, 4, 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Slice(added, func(i, j int) bool { return added[i] < added[j] })
	sort.Slice(removed, func(i, j int) bool { return removed[i] < removed[j] })

	if len(added) != 2 || added[0] != 4 || added[1] != 5 {
		t.Errorf("expected added=[4,5], got %v", added)
	}
	if len(removed) != 1 || removed[0] != 1 {
		t.Errorf("expected removed=[1], got %v", removed)
	}

	// 验证数据库已更新
	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) != 4 || ids[0] != 2 || ids[1] != 3 || ids[2] != 4 || ids[3] != 5 {
		t.Errorf("expected DB=[2,3,4,5], got %v", ids)
	}
}

func TestDiffAndSetCLSCollectScope_ClearAll(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1, 2, 3})

	added, removed, err := DiffAndSetCLSCollectScope(context.Background(), []uint{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(added) != 0 {
		t.Errorf("expected no added, got %v", added)
	}
	sort.Slice(removed, func(i, j int) bool { return removed[i] < removed[j] })
	if len(removed) != 3 || removed[0] != 1 || removed[1] != 2 || removed[2] != 3 {
		t.Errorf("expected removed=[1,2,3], got %v", removed)
	}

	// 验证数据库已清空
	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 0 {
		t.Errorf("expected empty DB, got %v", ids)
	}
}

func TestDiffAndSetCLSCollectScope_NoChange(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1, 2, 3})

	added, removed, err := DiffAndSetCLSCollectScope(context.Background(), []uint{1, 2, 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(added) != 0 {
		t.Errorf("expected no added, got %v", added)
	}
	if len(removed) != 0 {
		t.Errorf("expected no removed, got %v", removed)
	}

	// 验证数据库不变
	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Errorf("expected DB=[1,2,3], got %v", ids)
	}
}

func TestDiffAndSetCLSCollectScope_DuplicateInput(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// 输入包含重复 ID
	added, removed, err := DiffAndSetCLSCollectScope(context.Background(), []uint{1, 1, 2, 2, 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("expected no removed, got %v", removed)
	}
	sort.Slice(added, func(i, j int) bool { return added[i] < added[j] })
	if len(added) != 3 || added[0] != 1 || added[1] != 2 || added[2] != 3 {
		t.Errorf("expected added=[1,2,3] (deduped), got %v", added)
	}

	// 验证数据库中只有 3 条（而非 5 条）
	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 3 {
		t.Errorf("expected 3 records (deduped), got %d: %v", len(ids), ids)
	}
}

// ─── ExpandGroupIDsWithDescendants ─────────────────────────────────────

func TestExpandGroupIDsWithDescendants_NoClosure(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// 没有 closure 数据，结果应至少包含原始 IDs
	result, err := ExpandGroupIDsWithDescendants(context.Background(), []uint{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	if len(result) != 2 || result[0] != 1 || result[1] != 2 {
		t.Errorf("expected [1,2], got %v", result)
	}
}

func TestExpandGroupIDsWithDescendants_WithClosure(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// 构造 closure: group 1 -> {1, 10, 11}, group 2 -> {2, 20}
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 10, Depth: 1})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 11, Depth: 2})
	gdb.Create(&GroupClosure{AncestorID: 2, DescendantID: 2, Depth: 0})
	gdb.Create(&GroupClosure{AncestorID: 2, DescendantID: 20, Depth: 1})

	result, err := ExpandGroupIDsWithDescendants(context.Background(), []uint{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	expected := []uint{1, 2, 10, 11, 20}
	if len(result) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("result[%d]=%d, want %d", i, result[i], v)
		}
	}
}

func TestExpandGroupIDsWithDescendants_Empty(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	result, err := ExpandGroupIDsWithDescendants(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// ─── GetCLSCollectScopeCVMInstanceIDs ─────────────────────────────────────

func TestGetCLSCollectScopeCVMInstanceIDs_EmptyScope(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	ids, scopeSet, err := GetCLSCollectScopeCVMInstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scopeSet {
		t.Errorf("expected scopeSet=false for empty scope, got true")
	}
	if ids != nil {
		t.Errorf("expected nil for empty scope, got %v", ids)
	}
}

func TestGetCLSCollectScopeCVMInstanceIDs_WithData(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// 设置 scope 包含 group 1
	SetCLSCollectScope(context.Background(), []uint{1})
	// closure: group 1 -> {1}
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	// 用户 100 属于 group 1（membership 关联）
	gdb.Create(&UserGroupMember{UserGroupID: 1, UserID: 100})
	// 实例属于用户 100
	gdb.Create(&Instance{InstanceId: "ins-aaa", UserID: 100, GroupID: 1})
	gdb.Create(&Instance{InstanceId: "ins-bbb", UserID: 100, GroupID: 1})
	// 实例属于用户 200，不在 scope 分组中
	gdb.Create(&Instance{InstanceId: "ins-ccc", UserID: 200, GroupID: 99})

	ids, scopeSet, err := GetCLSCollectScopeCVMInstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !scopeSet {
		t.Errorf("expected scopeSet=true, got false")
	}
	sort.Strings(ids)
	if len(ids) != 2 {
		t.Fatalf("expected 2 instances, got %d: %v", len(ids), ids)
	}
	if ids[0] != "ins-aaa" || ids[1] != "ins-bbb" {
		t.Errorf("unexpected ids: %v", ids)
	}
}

// ─── IsInstanceGroupInCLSCollectScope ─────────────────────────────────────

func TestIsInstanceGroupInCLSCollectScope_EmptyScope(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// scope 为空 = 全量模式，任何分组都命中
	gdb.Create(&SiteConfig{CLSEnabled: 1})
	hit, err := IsInstanceGroupInCLSCollectScope(context.Background(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hit {
		t.Error("expected true (全量模式), got false")
	}
}

func TestIsInstanceGroupInCLSCollectScope_GroupInScope(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	gdb.Create(&SiteConfig{CLSEnabled: 1})
	SetCLSCollectScope(context.Background(), []uint{1})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 10, Depth: 1})

	hit, err := IsInstanceGroupInCLSCollectScope(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hit {
		t.Error("expected true for group in scope subtree, got false")
	}
}

func TestIsInstanceGroupInCLSCollectScope_GroupNotInScope(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})

	hit, err := IsInstanceGroupInCLSCollectScope(context.Background(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hit {
		t.Error("expected false for group not in scope, got true")
	}
}

func TestIsInstanceGroupInCLSCollectScope_GroupInChildScope(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// scope 只配了 group 1，但通过 closure 展开到子孙 group 11
	gdb.Create(&SiteConfig{CLSEnabled: 1})
	SetCLSCollectScope(context.Background(), []uint{1})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 11, Depth: 1})

	hit, err := IsInstanceGroupInCLSCollectScope(context.Background(), 11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hit {
		t.Error("expected true for group in child scope, got false")
	}
}

func TestIsInstanceGroupInCLSCollectScope_ZeroGroupID(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})

	// group_id=0 表示未指定分组，按分组模式下不命中
	hit, err := IsInstanceGroupInCLSCollectScope(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hit {
		t.Error("expected false for group_id=0 in group mode, got true")
	}
}

// ─── GetGroupsCVMInstanceIDs ─────────────────────────────────────

func TestGetGroupsCVMInstanceIDs_Empty(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	ids, err := GetGroupsCVMInstanceIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected empty, got %v", ids)
	}
}

func TestGetGroupsCVMInstanceIDs_WithData(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// 实例通过 group_id 关联到分组
	gdb.Create(&Instance{InstanceId: "ins-001", UserID: 100, GroupID: 1})
	gdb.Create(&Instance{InstanceId: "ins-002", UserID: 100, GroupID: 1})
	gdb.Create(&Instance{InstanceId: "ins-003", UserID: 200, GroupID: 1})
	gdb.Create(&Instance{InstanceId: "ins-004", UserID: 300, GroupID: 1})
	// 不属于分组 1 的实例
	gdb.Create(&Instance{InstanceId: "ins-005", UserID: 100, GroupID: 99})

	ids, err := GetGroupsCVMInstanceIDs(context.Background(), []uint{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(ids)
	if len(ids) != 4 {
		t.Fatalf("expected 4, got %d: %v", len(ids), ids)
	}
	if ids[0] != "ins-001" || ids[1] != "ins-002" || ids[2] != "ins-003" || ids[3] != "ins-004" {
		t.Errorf("unexpected ids: %v", ids)
	}
}

func TestGetGroupsCVMInstanceIDs_MultipleGroups(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	gdb.Create(&Instance{InstanceId: "ins-g1", UserID: 100, GroupID: 1})
	gdb.Create(&Instance{InstanceId: "ins-g2", UserID: 200, GroupID: 2})

	ids, err := GetGroupsCVMInstanceIDs(context.Background(), []uint{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(ids)
	if len(ids) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(ids), ids)
	}
	if ids[0] != "ins-g1" || ids[1] != "ins-g2" {
		t.Errorf("unexpected ids: %v", ids)
	}
}

func TestGetGroupsCVMInstanceIDs_DeduplicatesAcrossGroups(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// 每个实例有独立的 group_id，查多个分组时结果去重
	gdb.Create(&Instance{InstanceId: "ins-g1", UserID: 100, GroupID: 1})
	gdb.Create(&Instance{InstanceId: "ins-g2", UserID: 100, GroupID: 2})

	ids, err := GetGroupsCVMInstanceIDs(context.Background(), []uint{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(ids)
	if len(ids) != 2 {
		t.Errorf("期望 2 个实例，got %v", ids)
	}
}

// ─── RemoveCLSCollectScopeGroups 集成场景 ─────────────────────────────

func TestRemoveCLSCollectScopeGroups_PartialRemove(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{10, 20, 30, 40})

	// 移除中间两个
	if err := RemoveCLSCollectScopeGroups(context.Background(), []uint{20, 30}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) != 2 || ids[0] != 10 || ids[1] != 40 {
		t.Errorf("期望 [10,40]，got %v", ids)
	}
}

func TestRemoveCLSCollectScopeGroups_RemoveAll(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{1, 2})
	if err := RemoveCLSCollectScopeGroups(context.Background(), []uint{1, 2}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 0 {
		t.Errorf("期望清空，got %v", ids)
	}
}

// ─── RemoveThenAdd（硬删除后重新添加）───────────────────────────────

func TestRemoveThenAdd_HardDeleteAndReinsert(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// 先添加
	SetCLSCollectScope(context.Background(), []uint{1, 2, 3})
	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 3 {
		t.Fatalf("期望 3，got %d", len(ids))
	}

	// 硬删除移除 group 2
	RemoveCLSCollectScopeGroups(context.Background(), []uint{2})
	ids, _ = GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 2 {
		t.Fatalf("移除后期望 2，got %d", len(ids))
	}

	// 验证记录已被硬删除
	var allCount int64
	gdb.Model(&GroupConfigBinding{}).
		Where("config_type = ? AND config_key = ?", ConfigTypeCLSCollectScope, CLSCollectScopeKey).
		Count(&allCount)
	if allCount != 2 {
		t.Errorf("硬删除后应只剩 2 条记录，got %d", allCount)
	}

	// 重新添加 group 2
	AddCLSCollectScopeGroups(context.Background(), []uint{2})
	ids, _ = GetCLSCollectScopeGroupIDs(context.Background())
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Errorf("重新添加后期望 [1,2,3]，got %v", ids)
	}
}

func TestClearThenSet_HardDeleteAndReinsert(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	SetCLSCollectScope(context.Background(), []uint{10, 20})

	// 清空（硬删除）
	ClearCLSCollectScope(context.Background())
	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	if len(ids) != 0 {
		t.Fatalf("清空后期望 0，got %d", len(ids))
	}

	// 重新设置
	SetCLSCollectScope(context.Background(), []uint{10, 30})
	ids, _ = GetCLSCollectScopeGroupIDs(context.Background())
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) != 2 || ids[0] != 10 || ids[1] != 30 {
		t.Errorf("重新设置后期望 [10,30]，got %v", ids)
	}
}

func TestSetCLSCollectScope_DeleteAndReplace(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// 初始 [1, 2, 3]
	SetCLSCollectScope(context.Background(), []uint{1, 2, 3})

	// 替换为 [2, 4]：group 1, 3 被删除，group 4 新建
	SetCLSCollectScope(context.Background(), []uint{2, 4})
	ids, _ := GetCLSCollectScopeGroupIDs(context.Background())
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) != 2 || ids[0] != 2 || ids[1] != 4 {
		t.Errorf("期望 [2,4]，got %v", ids)
	}

	// 再替换为 [1, 2, 3, 4]
	SetCLSCollectScope(context.Background(), []uint{1, 2, 3, 4})
	ids, _ = GetCLSCollectScopeGroupIDs(context.Background())
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) != 4 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 || ids[3] != 4 {
		t.Errorf("期望 [1,2,3,4]，got %v", ids)
	}
}

func TestIsInstanceGroupInCLSCollectScope_DirectScopeGroup(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// scope 包含 group 1
	gdb.Create(&SiteConfig{CLSEnabled: 1})
	SetCLSCollectScope(context.Background(), []uint{1})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})

	// 实例直接属于 scope 内的分组
	hit, err := IsInstanceGroupInCLSCollectScope(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hit {
		t.Error("实例的 group_id 在 scope 中应命中")
	}
}

func TestIsInstanceGroupInCLSCollectScope_DeepNesting(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// scope 只配了 root(1)，实例属于三层子孙 group 111
	gdb.Create(&SiteConfig{CLSEnabled: 1})
	SetCLSCollectScope(context.Background(), []uint{1})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 11, Depth: 1})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 111, Depth: 2})

	hit, err := IsInstanceGroupInCLSCollectScope(context.Background(), 111)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hit {
		t.Error("三层子孙分组中的实例应命中 scope")
	}
}

func TestIsInstanceGroupInCLSCollectScope_GroupModeEmptyScope(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// 设置分组模式但不添加任何 scope 分组
	gdb.Create(&SiteConfig{CLSScopeMode: "group", CLSEnabled: 1})

	// 分组模式下 scope 为空 → 不命中任何实例
	hit, err := IsInstanceGroupInCLSCollectScope(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hit {
		t.Error("分组模式下 scope 为空时应不命中任何实例")
	}
}

func TestIsInstanceGroupInCLSCollectScope_GroupModeWithScope(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	gdb.Create(&SiteConfig{CLSScopeMode: "group", CLSEnabled: 1})
	SetCLSCollectScope(context.Background(), []uint{10})
	gdb.Create(&GroupClosure{AncestorID: 10, DescendantID: 10, Depth: 0})
	gdb.Create(&GroupClosure{AncestorID: 10, DescendantID: 11, Depth: 1})

	// group 11 在 scope 子树中
	hit, err := IsInstanceGroupInCLSCollectScope(context.Background(), 11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hit {
		t.Error("分组模式下 scope 子树内的分组应命中")
	}

	// group 99 不在 scope 中
	hit, err = IsInstanceGroupInCLSCollectScope(context.Background(), 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hit {
		t.Error("分组模式下 scope 外的分组应不命中")
	}
}

func TestGetCLSCollectScopeCVMInstanceIDs_ScopeWithNoClosureEntries(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	gdb.Create(&SiteConfig{CLSScopeMode: "group", CLSEnabled: 1})
	// 设置 scope 分组 50，但 closure 表中没有 group 50 的条目
	SetCLSCollectScope(context.Background(), []uint{50})

	ids, scopeSet, err := GetCLSCollectScopeCVMInstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !scopeSet {
		t.Error("expected scopeSet=true")
	}
	// ExpandGroupIDsWithDescendants 会将原始 groupIDs 也包含在结果中，所以不会为空
	// 但因为没有实例属于 group 50，最终 ids 应为空
	if len(ids) != 0 {
		t.Errorf("expected empty instance IDs, got %v", ids)
	}
}

func TestGetCVMInstanceIDsInGroups_EmptyInput(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	ids, err := GetCVMInstanceIDsInGroups(context.Background(), []uint{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected empty, got %v", ids)
	}
}

func TestGetCVMInstanceIDsInGroups_NoMatchingInstances(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// 查询一个不存在实例的分组
	ids, err := GetCVMInstanceIDsInGroups(context.Background(), []uint{999})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected empty, got %v", ids)
	}
}

func TestGetCVMInstanceIDsInGroups_WithData(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	gdb.Create(&Instance{InstanceId: "ins-m1", UserID: 1, GroupID: 5})
	gdb.Create(&Instance{InstanceId: "ins-m2", UserID: 2, GroupID: 5})
	gdb.Create(&Instance{InstanceId: "ins-m3", UserID: 3, GroupID: 6})

	ids, err := GetCVMInstanceIDsInGroups(context.Background(), []uint{5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(ids)
	if len(ids) != 2 || ids[0] != "ins-m1" || ids[1] != "ins-m2" {
		t.Errorf("expected [ins-m1, ins-m2], got %v", ids)
	}
}

func TestExpandGroupIDsWithDescendants_EmptyInput(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	result, err := ExpandGroupIDsWithDescendants(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestExpandGroupIDsWithDescendants_NoClosureEntries(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	// closure 表为空，但原始 groupIDs 仍应包含在结果中
	result, err := ExpandGroupIDsWithDescendants(context.Background(), []uint{7, 8})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	if len(result) != 2 || result[0] != 7 || result[1] != 8 {
		t.Errorf("expected [7,8], got %v", result)
	}
}

func TestExpandGroupIDsWithDescendants_MultiLevelClosure(t *testing.T) {
	cleanup := setupCLSScopeTestDB(t)
	defer cleanup()

	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})
	gdb.Create(&GroupClosure{AncestorID: 1, DescendantID: 3, Depth: 2})

	result, err := ExpandGroupIDsWithDescendants(context.Background(), []uint{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	if len(result) != 3 || result[0] != 1 || result[1] != 2 || result[2] != 3 {
		t.Errorf("expected [1,2,3], got %v", result)
	}
}
