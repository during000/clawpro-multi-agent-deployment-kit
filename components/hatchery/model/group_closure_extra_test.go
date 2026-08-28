package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupGroupClosureExtraDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.AutoMigrate(&UserGroup{}, &GroupClosure{}, &UserGroupMember{}, &User{}, &Instance{}, &GroupConfigBinding{})
	gdb = db
	t.Cleanup(func() { gdb = nil })
	return db
}

// seedClosureTree 创建 root -> child -> grandchild 树
func seedClosureTree(t *testing.T) (root, child, grandchild *UserGroup) {
	t.Helper()
	r := &UserGroup{Name: "Root", Source: "manual"}
	gdb.Create(r)
	c := &UserGroup{Name: "Child", Source: "manual", ParentID: r.ID}
	gdb.Create(c)
	g := &UserGroup{Name: "Grandchild", Source: "manual", ParentID: c.ID}
	gdb.Create(g)

	closures := []GroupClosure{
		{AncestorID: r.ID, DescendantID: r.ID, Depth: 0},
		{AncestorID: c.ID, DescendantID: c.ID, Depth: 0},
		{AncestorID: g.ID, DescendantID: g.ID, Depth: 0},
		{AncestorID: r.ID, DescendantID: c.ID, Depth: 1},
		{AncestorID: r.ID, DescendantID: g.ID, Depth: 2},
		{AncestorID: c.ID, DescendantID: g.ID, Depth: 1},
	}
	for _, cl := range closures {
		gdb.Create(&cl)
	}
	return r, c, g
}

// ── ClosureDescendants 测试 ─────────────────────────────────────────────────

func TestExtraClosureDescendants_IncludeSelf(t *testing.T) {
	setupGroupClosureExtraDB(t)
	root, child, grandchild := seedClosureTree(t)

	ids, err := ClosureDescendants(context.Background(), root.ID, true)
	if err != nil {
		t.Fatalf("ClosureDescendants err: %v", err)
	}
	if len(ids) != 3 {
		t.Errorf("期望 3 个后代（含自身），实际=%d, ids=%v", len(ids), ids)
	}

	// 检查包含所有三个
	m := make(map[uint]bool)
	for _, id := range ids {
		m[id] = true
	}
	if !m[root.ID] || !m[child.ID] || !m[grandchild.ID] {
		t.Errorf("缺少后代，ids=%v", ids)
	}
}

func TestExtraClosureDescendants_ExcludeSelf(t *testing.T) {
	setupGroupClosureExtraDB(t)
	root, _, _ := seedClosureTree(t)

	ids, err := ClosureDescendants(context.Background(), root.ID, false)
	if err != nil {
		t.Fatalf("ClosureDescendants err: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("期望 2 个后代（不含自身），实际=%d", len(ids))
	}
	for _, id := range ids {
		if id == root.ID {
			t.Error("不应包含自身")
		}
	}
}

func TestExtraClosureDescendants_LeafNode(t *testing.T) {
	setupGroupClosureExtraDB(t)
	_, _, grandchild := seedClosureTree(t)

	ids, err := ClosureDescendants(context.Background(), grandchild.ID, false)
	if err != nil {
		t.Fatalf("ClosureDescendants err: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("叶子节点不应有后代，实际=%d", len(ids))
	}
}

func TestExtraClosureDescendants_NonExistentNode(t *testing.T) {
	setupGroupClosureExtraDB(t)

	ids, err := ClosureDescendants(context.Background(), 99999, true)
	if err != nil {
		t.Fatalf("ClosureDescendants err: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("不存在的节点不应有后代，实际=%d", len(ids))
	}
}

// ── ClosureAncestors 测试 ───────────────────────────────────────────────────

func TestExtraClosureAncestors_IncludeSelf(t *testing.T) {
	setupGroupClosureExtraDB(t)
	root, child, grandchild := seedClosureTree(t)

	ids, err := ClosureAncestors(context.Background(), grandchild.ID, true)
	if err != nil {
		t.Fatalf("ClosureAncestors err: %v", err)
	}
	if len(ids) != 3 {
		t.Errorf("期望 3 个祖先（含自身），实际=%d, ids=%v", len(ids), ids)
	}
	// 第一个应该是自身（depth=0），最后是根
	if ids[0] != grandchild.ID {
		t.Errorf("第一个应为自身 %d，实际=%d", grandchild.ID, ids[0])
	}
	_ = child
	_ = root
}

func TestExtraClosureAncestors_ExcludeSelf(t *testing.T) {
	setupGroupClosureExtraDB(t)
	_, _, grandchild := seedClosureTree(t)

	ids, err := ClosureAncestors(context.Background(), grandchild.ID, false)
	if err != nil {
		t.Fatalf("ClosureAncestors err: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("期望 2 个祖先（不含自身），实际=%d", len(ids))
	}
	for _, id := range ids {
		if id == grandchild.ID {
			t.Error("不应包含自身")
		}
	}
}

func TestExtraClosureAncestors_RootNode(t *testing.T) {
	setupGroupClosureExtraDB(t)
	root, _, _ := seedClosureTree(t)

	ids, err := ClosureAncestors(context.Background(), root.ID, false)
	if err != nil {
		t.Fatalf("ClosureAncestors err: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("根节点不应有祖先，实际=%d", len(ids))
	}
}

// ── ReconcileClosure 测试 ───────────────────────────────────────────────────

func TestExtraReconcileClosure_RebuildFromParentID(t *testing.T) {
	setupGroupClosureExtraDB(t)

	// 创建分组但不创建 closure（模拟脏数据）
	root := &UserGroup{Name: "Root", Source: "manual"}
	gdb.Create(root)
	child := &UserGroup{Name: "Child", Source: "manual", ParentID: root.ID}
	gdb.Create(child)

	// 调用 reconcile 重建
	ReconcileClosure(context.Background())

	// 验证 closure 已重建
	ids, err := ClosureAncestors(context.Background(), child.ID, true)
	if err != nil {
		t.Fatalf("ClosureAncestors err: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("reconcile 后期望 2 个祖先（含自身），实际=%d", len(ids))
	}
}

// ── closureMaxRelativeDepthTx 测试 ──────────────────────────────────────────

func TestExtraClosureMaxRelativeDepth(t *testing.T) {
	setupGroupClosureExtraDB(t)
	root, _, _ := seedClosureTree(t)

	var maxDepth int
	gdb.Table("group_closure").Select("COALESCE(MAX(depth),0)").
		Where("ancestor_id = ?", root.ID).Scan(&maxDepth)
	if maxDepth != 2 {
		t.Errorf("期望最大深度 2，实际=%d", maxDepth)
	}
}

// ── closureIsDescendantTx 测试 ──────────────────────────────────────────────

func TestExtraClosureIsDescendant_True(t *testing.T) {
	setupGroupClosureExtraDB(t)
	root, _, grandchild := seedClosureTree(t)

	is, err := closureIsDescendantTx(gdb, root.ID, grandchild.ID)
	if err != nil {
		t.Fatalf("closureIsDescendantTx err: %v", err)
	}
	if !is {
		t.Error("grandchild 应为 root 的后代")
	}
}

func TestExtraClosureIsDescendant_False(t *testing.T) {
	setupGroupClosureExtraDB(t)
	_, child, grandchild := seedClosureTree(t)

	is, err := closureIsDescendantTx(gdb, grandchild.ID, child.ID)
	if err != nil {
		t.Fatalf("closureIsDescendantTx err: %v", err)
	}
	if is {
		t.Error("child 不应为 grandchild 的后代")
	}
}

func TestExtraClosureIsDescendant_Self(t *testing.T) {
	setupGroupClosureExtraDB(t)
	root, _, _ := seedClosureTree(t)

	is, err := closureIsDescendantTx(gdb, root.ID, root.ID)
	if err != nil {
		t.Fatalf("closureIsDescendantTx err: %v", err)
	}
	if !is {
		t.Error("节点应该是自身的后代（depth=0）")
	}
}
