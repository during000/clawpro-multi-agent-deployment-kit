package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupClosureTestDB 创建内存 SQLite 数据库，用于 closure 测试。
func setupClosureTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&UserGroup{},
		&UserGroupMember{},
		&GroupClosure{},
		&GroupConfigBinding{},
		&AIModel{},
		&ModelVisibilityGroup{},
		&Skill{},
		&SkillVisibilityGroup{},
		&SkillBundle{},
		&SkillBundleVisibilityGroup{},
		&OpenClawRole{},
		&RoleVisibilityGroup{},
		&Instance{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	gdb = db
}

// ── ClosureDescendants ───────────────────────────────────────────────────────

// TestClosureDescendants_RootWithChildren 根组有子孙时，IncludeSelf=true 返回全部。
func TestClosureDescendants_RootWithChildren(t *testing.T) {
	setupClosureTestDB(t)

	// 手动插入分组和 closure
	groups := []UserGroup{
		{ID: 1, Name: "根组"},
		{ID: 2, Name: "子组"},
		{ID: 3, Name: "孙组"},
	}
	for _, g := range groups {
		gdb.Create(&g)
	}
	closures := []GroupClosure{
		{AncestorID: 1, DescendantID: 1, Depth: 0},
		{AncestorID: 2, DescendantID: 2, Depth: 0},
		{AncestorID: 3, DescendantID: 3, Depth: 0},
		{AncestorID: 1, DescendantID: 2, Depth: 1},
		{AncestorID: 2, DescendantID: 3, Depth: 1},
		{AncestorID: 1, DescendantID: 3, Depth: 2},
	}
	for _, c := range closures {
		gdb.Create(&c)
	}

	// includeSelf=true：1, 2, 3
	ids, err := ClosureDescendants(context.Background(), 1, true)
	if err != nil {
		t.Fatalf("ClosureDescendants: %v", err)
	}
	if len(ids) != 3 {
		t.Errorf("expected 3 descendants (incl. self), got %d: %v", len(ids), ids)
	}
}

// TestClosureDescendants_ExcludeSelf includeSelf=false 不包含自身。
func TestClosureDescendants_ExcludeSelf(t *testing.T) {
	setupClosureTestDB(t)

	gdb.Create(&UserGroup{ID: 10, Name: "A"})
	gdb.Create(&UserGroup{ID: 11, Name: "B"})
	gdb.Create(&GroupClosure{AncestorID: 10, DescendantID: 10, Depth: 0})
	gdb.Create(&GroupClosure{AncestorID: 10, DescendantID: 11, Depth: 1})
	gdb.Create(&GroupClosure{AncestorID: 11, DescendantID: 11, Depth: 0})

	ids, err := ClosureDescendants(context.Background(), 10, false)
	if err != nil {
		t.Fatalf("ClosureDescendants: %v", err)
	}
	if len(ids) != 1 || ids[0] != 11 {
		t.Errorf("expected [11], got %v", ids)
	}
}

// TestClosureDescendants_LeafNode 叶子节点无子孙，includeSelf=false 返回空。
func TestClosureDescendants_LeafNode(t *testing.T) {
	setupClosureTestDB(t)

	gdb.Create(&UserGroup{ID: 20, Name: "叶子"})
	gdb.Create(&GroupClosure{AncestorID: 20, DescendantID: 20, Depth: 0})

	ids, err := ClosureDescendants(context.Background(), 20, false)
	if err != nil {
		t.Fatalf("ClosureDescendants: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected empty, got %v", ids)
	}
}

// TestClosureDescendants_LeafNodeIncludeSelf 叶子节点，includeSelf=true 只返回自己。
func TestClosureDescendants_LeafNodeIncludeSelf(t *testing.T) {
	setupClosureTestDB(t)

	gdb.Create(&UserGroup{ID: 30, Name: "叶子2"})
	gdb.Create(&GroupClosure{AncestorID: 30, DescendantID: 30, Depth: 0})

	ids, err := ClosureDescendants(context.Background(), 30, true)
	if err != nil {
		t.Fatalf("ClosureDescendants: %v", err)
	}
	if len(ids) != 1 || ids[0] != 30 {
		t.Errorf("expected [30], got %v", ids)
	}
}

// ── ClosureAncestors ─────────────────────────────────────────────────────────

// TestClosureAncestors_IncludeSelf 祖先链含自身，深度从小到大。
func TestClosureAncestors_IncludeSelf(t *testing.T) {
	setupClosureTestDB(t)

	gdb.Create(&UserGroup{ID: 1, Name: "Root"})
	gdb.Create(&UserGroup{ID: 2, Name: "Child"})
	gdb.Create(&UserGroup{ID: 3, Name: "GrandChild"})
	closures := []GroupClosure{
		{AncestorID: 1, DescendantID: 1, Depth: 0},
		{AncestorID: 2, DescendantID: 2, Depth: 0},
		{AncestorID: 3, DescendantID: 3, Depth: 0},
		{AncestorID: 1, DescendantID: 2, Depth: 1},
		{AncestorID: 2, DescendantID: 3, Depth: 1},
		{AncestorID: 1, DescendantID: 3, Depth: 2},
	}
	for _, c := range closures {
		gdb.Create(&c)
	}

	ancestors, err := ClosureAncestors(context.Background(), 3, true)
	if err != nil {
		t.Fatalf("ClosureAncestors: %v", err)
	}
	if len(ancestors) != 3 {
		t.Errorf("expected 3, got %d: %v", len(ancestors), ancestors)
	}
	// 按 depth 升序：[3, 2, 1]（depth=0, depth=1, depth=2 对 descendant_id=3 而言）
	if ancestors[0] != 3 {
		t.Errorf("ancestors[0] should be self(3), got %d", ancestors[0])
	}
}

// TestClosureAncestors_ExcludeSelf 排除自身后只含父辈。
func TestClosureAncestors_ExcludeSelf(t *testing.T) {
	setupClosureTestDB(t)

	gdb.Create(&UserGroup{ID: 40, Name: "Parent"})
	gdb.Create(&UserGroup{ID: 41, Name: "Child"})
	gdb.Create(&GroupClosure{AncestorID: 40, DescendantID: 40, Depth: 0})
	gdb.Create(&GroupClosure{AncestorID: 41, DescendantID: 41, Depth: 0})
	gdb.Create(&GroupClosure{AncestorID: 40, DescendantID: 41, Depth: 1})

	ancestors, err := ClosureAncestors(context.Background(), 41, false)
	if err != nil {
		t.Fatalf("ClosureAncestors: %v", err)
	}
	if len(ancestors) != 1 || ancestors[0] != 40 {
		t.Errorf("expected [40], got %v", ancestors)
	}
}

// TestClosureAncestors_RootGroup 根组无祖先，includeSelf=false 返回空。
func TestClosureAncestors_RootGroup(t *testing.T) {
	setupClosureTestDB(t)

	gdb.Create(&UserGroup{ID: 50, Name: "OnlyRoot"})
	gdb.Create(&GroupClosure{AncestorID: 50, DescendantID: 50, Depth: 0})

	ancestors, err := ClosureAncestors(context.Background(), 50, false)
	if err != nil {
		t.Fatalf("ClosureAncestors: %v", err)
	}
	if len(ancestors) != 0 {
		t.Errorf("expected empty, got %v", ancestors)
	}
}

// ── 通过 CreateUserGroupWithOpts 测试 closureInsertForNewChildTx ────────────

// TestCreateUserGroupWithOpts_RootGroup 创建根组后 closure 只有自指行。
func TestCreateUserGroupWithOpts_RootGroup(t *testing.T) {
	setupClosureTestDB(t)

	g, err := CreateUserGroupWithOpts(context.Background(), "根组A", "desc", 0, MemberSourceManual, "")
	if err != nil {
		t.Fatalf("CreateUserGroupWithOpts: %v", err)
	}

	var rows []GroupClosure
	gdb.Where("descendant_id = ?", g.ID).Find(&rows)
	if len(rows) != 1 {
		t.Errorf("root group should have 1 closure row, got %d", len(rows))
	}
	if rows[0].AncestorID != g.ID || rows[0].Depth != 0 {
		t.Errorf("unexpected closure row: %+v", rows[0])
	}
}

// TestCreateUserGroupWithOpts_ChildInheritsAncestors 子组继承父的祖先关系。
func TestCreateUserGroupWithOpts_ChildInheritsAncestors(t *testing.T) {
	setupClosureTestDB(t)

	root, err := CreateUserGroupWithOpts(context.Background(), "Root", "", 0, MemberSourceManual, "")
	if err != nil {
		t.Fatalf("创建根组: %v", err)
	}
	child, err := CreateUserGroupWithOpts(context.Background(), "Child", "", root.ID, MemberSourceManual, "")
	if err != nil {
		t.Fatalf("创建子组: %v", err)
	}
	grandchild, err := CreateUserGroupWithOpts(context.Background(), "GrandChild", "", child.ID, MemberSourceManual, "")
	if err != nil {
		t.Fatalf("创建孙组: %v", err)
	}

	// 孙组应有 3 条 closure 行（自指 + 子 + 根）
	var rows []GroupClosure
	gdb.Where("descendant_id = ?", grandchild.ID).Find(&rows)
	if len(rows) != 3 {
		t.Errorf("grandchild should have 3 closure rows, got %d", len(rows))
	}
}

// TestCreateUserGroupWithOpts_DepthField 子组的 Depth 字段等于父的 Depth+1。
func TestCreateUserGroupWithOpts_DepthField(t *testing.T) {
	setupClosureTestDB(t)

	root, _ := CreateUserGroupWithOpts(context.Background(), "RootDep", "", 0, MemberSourceManual, "")
	child, _ := CreateUserGroupWithOpts(context.Background(), "ChildDep", "", root.ID, MemberSourceManual, "")

	if root.Depth != 0 {
		t.Errorf("root depth should be 0, got %d", root.Depth)
	}
	if child.Depth != 1 {
		t.Errorf("child depth should be 1, got %d", child.Depth)
	}
}

// ── DeleteUserGroup 测试 closureDeleteNodeTx ─────────────────────────────────

// TestDeleteUserGroup_CleansClosure 删除叶子组后 closure 行被清理。
func TestDeleteUserGroup_CleansClosure(t *testing.T) {
	setupClosureTestDB(t)

	root, _ := CreateUserGroupWithOpts(context.Background(), "RootDel", "", 0, MemberSourceManual, "")
	leaf, _ := CreateUserGroupWithOpts(context.Background(), "LeafDel", "", root.ID, MemberSourceManual, "")

	if err := DeleteUserGroup(context.Background(), leaf.ID); err != nil {
		t.Fatalf("DeleteUserGroup: %v", err)
	}

	var count int64
	gdb.Model(&GroupClosure{}).Where("ancestor_id = ? OR descendant_id = ?", leaf.ID, leaf.ID).Count(&count)
	if count != 0 {
		t.Errorf("closure rows for deleted group should be 0, got %d", count)
	}
}

// ── ReconcileClosure ─────────────────────────────────────────────────────────

// TestReconcileClosure_ConsistentData 数据一致时不触发重建。
func TestReconcileClosure_ConsistentData(t *testing.T) {
	setupClosureTestDB(t)

	// 通过正常 API 创建，保证 closure 一致
	root, _ := CreateUserGroupWithOpts(context.Background(), "RecRoot", "", 0, MemberSourceManual, "")
	_, _ = CreateUserGroupWithOpts(context.Background(), "RecChild", "", root.ID, MemberSourceManual, "")

	var before int64
	gdb.Model(&GroupClosure{}).Count(&before)

	// 调用 ReconcileClosure，不应改变数量
	ReconcileClosure(context.Background())

	var after int64
	gdb.Model(&GroupClosure{}).Count(&after)
	if before != after {
		t.Errorf("consistent DB: closure row count should not change (before=%d, after=%d)", before, after)
	}
}

// TestReconcileClosure_FixesInconsistency 数据不一致时触发重建。
func TestReconcileClosure_FixesInconsistency(t *testing.T) {
	setupClosureTestDB(t)

	root, _ := CreateUserGroupWithOpts(context.Background(), "FixRoot", "", 0, MemberSourceManual, "")
	child, _ := CreateUserGroupWithOpts(context.Background(), "FixChild", "", root.ID, MemberSourceManual, "")

	// 手动删除一条 closure 行，制造不一致
	gdb.Where("ancestor_id = ? AND descendant_id = ?", root.ID, child.ID).Delete(&GroupClosure{})

	var before int64
	gdb.Model(&GroupClosure{}).Count(&before)

	ReconcileClosure(context.Background())

	var after int64
	gdb.Model(&GroupClosure{}).Count(&after)
	if after <= before {
		t.Errorf("expected rebuild to add rows (before=%d, after=%d)", before, after)
	}
}

// TestReconcileClosure_EmptyDB 空数据库不 panic。
func TestReconcileClosure_EmptyDB(t *testing.T) {
	setupClosureTestDB(t)

	// 不应 panic
	ReconcileClosure(context.Background())

	var count int64
	gdb.Model(&GroupClosure{}).Count(&count)
	if count != 0 {
		t.Errorf("empty DB should have 0 closure rows after reconcile, got %d", count)
	}
}

// TestReconcileClosure_OnlyMinimalDelete 验证不一致仅多了一条脏行时，
// 增量修复只 DELETE 那一行，不 TRUNCATE 全表（保留所有正确行）。
//
// 这是本次重构的核心目标：旧实现哪怕只差一行也会先 DELETE 整张表再 rebuild，
// 风险包括"事务期间并发查询读到空表"、"rebuild 失败丢历史"。新实现只动差异行。
func TestReconcileClosure_OnlyMinimalDelete(t *testing.T) {
	setupClosureTestDB(t)

	root, _ := CreateUserGroupWithOpts(context.Background(), "MinDelRoot", "", 0, MemberSourceManual, "")
	child, _ := CreateUserGroupWithOpts(context.Background(), "MinDelChild", "", root.ID, MemberSourceManual, "")
	_ = child

	// 在正确 closure 之外塞一条脏行：99 → 99 自指（user_groups 里没有 id=99）
	if err := gdb.Create(&GroupClosure{AncestorID: 99, DescendantID: 99, Depth: 0}).Error; err != nil {
		t.Fatalf("seed dirty row: %v", err)
	}

	// 记下当前所有正确行的 (a,d,depth)
	var before []GroupClosure
	gdb.Order("ancestor_id, descendant_id").Find(&before)

	ReconcileClosure(context.Background())

	// 脏行应被删除
	var dirty int64
	gdb.Model(&GroupClosure{}).Where("ancestor_id = ? OR descendant_id = ?", 99, 99).Count(&dirty)
	if dirty != 0 {
		t.Errorf("脏行未被清理，剩余 %d 行", dirty)
	}

	// 所有原本正确的行必须仍在（不是被 TRUNCATE 后 rebuild 出来的）
	for _, b := range before {
		if b.AncestorID == 99 || b.DescendantID == 99 {
			continue
		}
		var n int64
		gdb.Model(&GroupClosure{}).
			Where("ancestor_id = ? AND descendant_id = ? AND depth = ?", b.AncestorID, b.DescendantID, b.Depth).
			Count(&n)
		if n != 1 {
			t.Errorf("正确行 (a=%d, d=%d, depth=%d) 不应被清理（旧实现会 TRUNCATE 全表）",
				b.AncestorID, b.DescendantID, b.Depth)
		}
	}
}

// TestReconcileClosure_DepthMismatch_FixedInPlace 验证同 (a,d) 但 depth 不一致时，
// 同时进 toDelete + toInsert，最终修正为正确 depth。
func TestReconcileClosure_DepthMismatch_FixedInPlace(t *testing.T) {
	setupClosureTestDB(t)

	root, _ := CreateUserGroupWithOpts(context.Background(), "DepthRoot", "", 0, MemberSourceManual, "")
	child, _ := CreateUserGroupWithOpts(context.Background(), "DepthChild", "", root.ID, MemberSourceManual, "")

	// 把 (root, child, 1) 改成 (root, child, 5) — 模拟历史脏数据
	if err := gdb.Model(&GroupClosure{}).
		Where("ancestor_id = ? AND descendant_id = ?", root.ID, child.ID).
		Update("depth", 5).Error; err != nil {
		t.Fatalf("seed bad depth: %v", err)
	}

	ReconcileClosure(context.Background())

	var got GroupClosure
	if err := gdb.Where("ancestor_id = ? AND descendant_id = ?", root.ID, child.ID).
		First(&got).Error; err != nil {
		t.Fatalf("查 (root,child) closure: %v", err)
	}
	if got.Depth != 1 {
		t.Errorf("depth 应被修正为 1，实际 %d", got.Depth)
	}

	// 不应留下重复行（同 a/d 应该只有 1 条）
	var n int64
	gdb.Model(&GroupClosure{}).
		Where("ancestor_id = ? AND descendant_id = ?", root.ID, child.ID).Count(&n)
	if n != 1 {
		t.Errorf("(root,child) 应只剩 1 行，实际 %d", n)
	}
}

// TestReconcileClosure_NoOpWhenConsistent_NoWrite 验证一致状态下不发生写操作。
//
// 通过比较 reconcile 前后所有 closure 行的内容（包含 depth）完全相等来判定。
// 旧实现走"length match + 内容快速匹配"短路，新实现走"完整 diff 短路"，
// 行为应一致：consistent 数据 → reconcile 不应改任何行。
func TestReconcileClosure_NoOpWhenConsistent_NoWrite(t *testing.T) {
	setupClosureTestDB(t)

	root, _ := CreateUserGroupWithOpts(context.Background(), "NoopRoot", "", 0, MemberSourceManual, "")
	c1, _ := CreateUserGroupWithOpts(context.Background(), "NoopC1", "", root.ID, MemberSourceManual, "")
	_, _ = CreateUserGroupWithOpts(context.Background(), "NoopGC", "", c1.ID, MemberSourceManual, "")

	var before, after []GroupClosure
	gdb.Order("ancestor_id, descendant_id, depth").Find(&before)

	ReconcileClosure(context.Background())

	gdb.Order("ancestor_id, descendant_id, depth").Find(&after)
	if len(before) != len(after) {
		t.Fatalf("一致状态下不应改变行数，before=%d after=%d", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("行 %d 内容被改动: before=%+v after=%+v", i, before[i], after[i])
		}
	}
}

// ── UpdateUserGroupExt 测试 closureMoveSubtreeTx ──────────────────────────────

// TestUpdateUserGroupExt_Reparent 换父后 closure 关系正确更新。
func TestUpdateUserGroupExt_Reparent(t *testing.T) {
	setupClosureTestDB(t)

	// groupA(root) → groupB
	// groupC(root)
	// 目标：把 groupB 移到 groupC 下面
	groupA, _ := CreateUserGroupWithOpts(context.Background(), "GroupA", "", 0, MemberSourceManual, "")
	groupB, _ := CreateUserGroupWithOpts(context.Background(), "GroupB", "", groupA.ID, MemberSourceManual, "")
	groupC, _ := CreateUserGroupWithOpts(context.Background(), "GroupC", "", 0, MemberSourceManual, "")

	// 验证 groupB 当前的祖先链包含 groupA
	var rowsBefore []GroupClosure
	gdb.Where("descendant_id = ?", groupB.ID).Find(&rowsBefore)
	foundA := false
	for _, r := range rowsBefore {
		if r.AncestorID == groupA.ID {
			foundA = true
		}
	}
	if !foundA {
		t.Error("before reparent: groupB should have groupA as ancestor")
	}

	// 换父
	_, err := UpdateUserGroupExt(context.Background(), groupB.ID, UpdateGroupOpts{NewParentIDPtr: &groupC.ID})
	if err != nil {
		t.Fatalf("UpdateUserGroupExt reparent: %v", err)
	}

	// 验证 groupB 现在的祖先链包含 groupC，不包含 groupA
	var rowsAfter []GroupClosure
	gdb.Where("descendant_id = ?", groupB.ID).Find(&rowsAfter)
	foundC, foundAAfter := false, false
	for _, r := range rowsAfter {
		if r.AncestorID == groupC.ID {
			foundC = true
		}
		if r.AncestorID == groupA.ID {
			foundAAfter = true
		}
	}
	if !foundC {
		t.Error("after reparent: groupB should have groupC as ancestor")
	}
	if foundAAfter {
		t.Error("after reparent: groupB should NOT have groupA as ancestor")
	}
}

// TestUpdateUserGroupExt_ReparentToRoot 换父到根级别（parentID=0）。
func TestUpdateUserGroupExt_ReparentToRoot(t *testing.T) {
	setupClosureTestDB(t)

	parent, _ := CreateUserGroupWithOpts(context.Background(), "MoveParent", "", 0, MemberSourceManual, "")
	child, _ := CreateUserGroupWithOpts(context.Background(), "MoveChild", "", parent.ID, MemberSourceManual, "")

	// 移动为根组（parentID = 0）
	zero := uint(0)
	_, err := UpdateUserGroupExt(context.Background(), child.ID, UpdateGroupOpts{NewParentIDPtr: &zero})
	if err != nil {
		t.Fatalf("UpdateUserGroupExt to root: %v", err)
	}

	// child 应该只有自指行
	var rows []GroupClosure
	gdb.Where("descendant_id = ?", child.ID).Find(&rows)
	if len(rows) != 1 {
		t.Errorf("after reparent to root: expected 1 closure row, got %d", len(rows))
	}
	if rows[0].AncestorID != child.ID {
		t.Errorf("after reparent to root: only row should be self-ref, got %+v", rows[0])
	}
}
