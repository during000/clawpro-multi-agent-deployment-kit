package model

import (
	"context"
	"testing"

	"hatchery/common"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 本文件覆盖：closure 相关 helper 在多租户(identifier)隔离下的行为。
//
// 背景：原实现里 closureInsertForNewChildTx / closureMoveSubtreeTx /
// closureDeleteNodeTx / closureMaxRelativeDepthTx 全部用 tx.Exec / tx.Raw 走
// 原生 SQL —— GORM identifier 回调只挂在 Create/Query/Update/Delete/Row 钩子上，
// Exec/Raw 一律不触发，于是 SQL 里没显式 WHERE identifier=? 的话，MySQL 模式下
// user_groups.id 跨租户重复时会读/写到别的租户的数据。
//
// 重构后把这些函数全部改用 GORM 接口（Where/Find/Delete/Create/Scan），由全局
// 回调自动注入 identifier。本文件就是回归这个隔离行为：构造两个 identifier 同
// 时种数据 + id 完全重叠 → 调一边只能动一边。

// setupClosureTenantDB 建一个**注册了 identifier 回调**的内存 SQLite。
// closure 业务测试默认不注册回调（见 setupClosureTestDB），这里我们要显式开启。
func setupClosureTenantDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// 注册 identifier 回调
	registerIdentifierCallbacks(db)
	// AutoMigrate 用 SkipIdentifier ctx 避免回调在无 TenantSnapshot 时 panic
	if err := db.WithContext(common.WithSkipIdentifier(context.Background())).
		AutoMigrate(&UserGroup{}, &GroupClosure{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	gdb = db
}

// ctxA / ctxB 两个不同 identifier 的 ctx，用于隔离测试。
func ctxA() context.Context {
	return common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-A"})
}
func ctxB() context.Context {
	return common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-B"})
}

// seedClosureRowRaw 直接通过 SkipIdentifier 路径插入指定 identifier 的 closure 行。
// 用 Create 而非 Exec —— Create 在 SkipIdentifier ctx 下不会被回调覆盖 identifier
// 字段，从而精确控制每行属于哪个租户。
func seedClosureRowRaw(t *testing.T, identifier string, ancestor, descendant uint, depth int) {
	t.Helper()
	row := GroupClosure{
		Identifier:   identifier,
		AncestorID:   ancestor,
		DescendantID: descendant,
		Depth:        depth,
	}
	skipCtx := common.WithSkipIdentifier(context.Background())
	if err := gdb.WithContext(skipCtx).Create(&row).Error; err != nil {
		t.Fatalf("seed closure (%s, a=%d, d=%d): %v", identifier, ancestor, descendant, err)
	}
}

// countClosureForTenant 不走回调直接数指定 identifier 的行数。
func countClosureForTenant(t *testing.T, identifier string) int64 {
	t.Helper()
	var n int64
	skipCtx := common.WithSkipIdentifier(context.Background())
	if err := gdb.WithContext(skipCtx).Model(&GroupClosure{}).
		Where("identifier = ?", identifier).Count(&n).Error; err != nil {
		t.Fatalf("count tenant=%s: %v", identifier, err)
	}
	return n
}

// findClosureForTenant 拉指定 identifier 的所有行，按 (a,d) 排序，便于断言。
func findClosureForTenant(t *testing.T, identifier string) []GroupClosure {
	t.Helper()
	skipCtx := common.WithSkipIdentifier(context.Background())
	var rows []GroupClosure
	if err := gdb.WithContext(skipCtx).
		Where("identifier = ?", identifier).
		Order("ancestor_id, descendant_id").Find(&rows).Error; err != nil {
		t.Fatalf("find tenant=%s: %v", identifier, err)
	}
	return rows
}

// ── closureInsertForNewChildTx：跨租户隔离 ───────────────────────────────────

// TestClosureInsertForNewChildTx_TenantIsolation
//
// 场景：A 租户和 B 租户都有 id=1 (root) 和 id=2 (root 自指)。
// 现在 A 下要新建 id=5（parent=1），跑 closureInsertForNewChildTx。
// 期望：只继承 A 租户 id=1 的祖先链；B 的 closure 完全不动。
//
// 旧实现（INSERT...SELECT 不带 identifier 过滤）会同时把 B 的 (id=1, id=1, 0)
// 拉过来写一行 (B.id=1, A.id=5, 1) — 跨租户污染。
func TestClosureInsertForNewChildTx_TenantIsolation(t *testing.T) {
	setupClosureTenantDB(t)

	// A 租户：id=1 自指；id=2 是 1 的子组（已在；id=2 自指 + (1,2,1)）
	seedClosureRowRaw(t, "tenant-A", 1, 1, 0)
	seedClosureRowRaw(t, "tenant-A", 2, 2, 0)
	seedClosureRowRaw(t, "tenant-A", 1, 2, 1)
	// B 租户：id=1 自指（id 与 A 重叠）
	seedClosureRowRaw(t, "tenant-B", 1, 1, 0)

	bBefore := findClosureForTenant(t, "tenant-B")

	// 在 A 租户下，新建 id=5 挂在 parent=1 下面
	tx := gdb.WithContext(ctxA())
	if err := closureInsertForNewChildTx(tx, 5, 1); err != nil {
		t.Fatalf("closureInsertForNewChildTx: %v", err)
	}

	// A 租户应该新增：(5,5,0) + 继承 (1,5,1) = 2 行
	aAfter := findClosureForTenant(t, "tenant-A")
	wantA := map[[3]uint]int{
		{1, 1, 0}: 0, {2, 2, 0}: 0, {1, 2, 1}: 1, // 原有
		{5, 5, 0}: 0, {1, 5, 1}: 1, // 新增
	}
	if len(aAfter) != len(wantA) {
		t.Errorf("A 租户期望 %d 行，实际 %d (%+v)", len(wantA), len(aAfter), aAfter)
	}
	for _, r := range aAfter {
		k := [3]uint{r.AncestorID, r.DescendantID, uint(r.Depth)}
		if _, ok := wantA[k]; !ok {
			t.Errorf("A 租户出现意外行 %+v", r)
		}
	}

	// B 租户必须毫发无伤
	bAfter := findClosureForTenant(t, "tenant-B")
	if len(bAfter) != len(bBefore) {
		t.Fatalf("B 租户行数被改动: before=%d after=%d (%+v)", len(bBefore), len(bAfter), bAfter)
	}
	for i := range bAfter {
		if bAfter[i] != bBefore[i] {
			t.Errorf("B 租户行 %d 被改动: before=%+v after=%+v", i, bBefore[i], bAfter[i])
		}
	}
}

// ── closureMoveSubtreeTx：跨租户隔离 ─────────────────────────────────────────

// TestClosureMoveSubtreeTx_TenantIsolation
//
// 两租户都有同样 (1→2) 结构，A 下把 id=2 移到 0（变根组），B 不应被波及。
//
// 旧实现的两段 Exec 都没 identifier 过滤，子树枚举 / 祖先 JOIN 会跨租户取行 →
// B 也会被错误删/插。
func TestClosureMoveSubtreeTx_TenantIsolation(t *testing.T) {
	setupClosureTenantDB(t)

	// 两租户都种相同结构：id=1 root, id=2 child of 1
	for _, tt := range []string{"tenant-A", "tenant-B"} {
		seedClosureRowRaw(t, tt, 1, 1, 0)
		seedClosureRowRaw(t, tt, 2, 2, 0)
		seedClosureRowRaw(t, tt, 1, 2, 1)
	}
	bBefore := findClosureForTenant(t, "tenant-B")

	// A 租户：把 id=2 移到根（newParentID=0）
	tx := gdb.WithContext(ctxA())
	if err := closureMoveSubtreeTx(tx, 2, 0); err != nil {
		t.Fatalf("move: %v", err)
	}

	// A：(1,2,1) 应被删除；(2,2,0) 保留；(1,1,0) 保留
	aAfter := findClosureForTenant(t, "tenant-A")
	wantA := map[[3]uint]bool{{1, 1, 0}: true, {2, 2, 0}: true}
	if len(aAfter) != len(wantA) {
		t.Errorf("A 期望 %d 行，实际 %d (%+v)", len(wantA), len(aAfter), aAfter)
	}
	for _, r := range aAfter {
		k := [3]uint{r.AncestorID, r.DescendantID, uint(r.Depth)}
		if !wantA[k] {
			t.Errorf("A 残留意外行 %+v（旧 (1,2,1) 应被删）", r)
		}
	}

	// B：原封不动
	bAfter := findClosureForTenant(t, "tenant-B")
	if len(bAfter) != len(bBefore) {
		t.Fatalf("B 行数被改动: before=%d after=%d (%+v)", len(bBefore), len(bAfter), bAfter)
	}
}

// ── closureDeleteNodeTx：跨租户隔离 ──────────────────────────────────────────

// TestClosureDeleteNodeTx_TenantIsolation
// A 租户删 id=2 的节点不应影响 B 租户的 id=2。
func TestClosureDeleteNodeTx_TenantIsolation(t *testing.T) {
	setupClosureTenantDB(t)

	for _, tt := range []string{"tenant-A", "tenant-B"} {
		seedClosureRowRaw(t, tt, 1, 1, 0)
		seedClosureRowRaw(t, tt, 2, 2, 0)
		seedClosureRowRaw(t, tt, 1, 2, 1)
	}

	tx := gdb.WithContext(ctxA())
	if err := closureDeleteNodeTx(tx, 2); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if got := countClosureForTenant(t, "tenant-A"); got != 1 {
		t.Errorf("A 删 id=2 后期望 1 行（只剩 1,1,0），实际 %d", got)
	}
	if got := countClosureForTenant(t, "tenant-B"); got != 3 {
		t.Errorf("B 应不受影响保留 3 行，实际 %d", got)
	}
}

// ── closureMaxRelativeDepthTx：跨租户隔离 ────────────────────────────────────

// TestClosureMaxRelativeDepthTx_TenantIsolation
//
// A 租户 root=1 子树深度 1；B 租户 root=1 子树深度 5（特意制造很大）。
// 在 A 的 ctx 下查询应得 1 而不是 5。
func TestClosureMaxRelativeDepthTx_TenantIsolation(t *testing.T) {
	setupClosureTenantDB(t)

	// A: 1 → 2 (depth 1)
	seedClosureRowRaw(t, "tenant-A", 1, 1, 0)
	seedClosureRowRaw(t, "tenant-A", 2, 2, 0)
	seedClosureRowRaw(t, "tenant-A", 1, 2, 1)
	// B: 1 → 99 (depth 5) —— 故意大数字，跨租户读取必爆掉
	seedClosureRowRaw(t, "tenant-B", 1, 1, 0)
	seedClosureRowRaw(t, "tenant-B", 99, 99, 0)
	seedClosureRowRaw(t, "tenant-B", 1, 99, 5)

	tx := gdb.WithContext(ctxA())
	got, err := closureMaxRelativeDepthTx(tx, 1)
	if err != nil {
		t.Fatalf("max depth: %v", err)
	}
	if got != 1 {
		t.Errorf("A 租户 root=1 max depth 期望 1，实际 %d（旧实现会读到 B 的 5 → 5）", got)
	}
}

// ── ReconcileClosure：跨租户隔离 ─────────────────────────────────────────────

// TestReconcileClosure_TenantIsolation
//
// A 租户 closure 缺失一行需修复；B 租户 closure 完全一致 → reconcile 后
// A 修复了，B 一行都没动。
func TestReconcileClosure_TenantIsolation(t *testing.T) {
	setupClosureTenantDB(t)

	skipCtx := common.WithSkipIdentifier(context.Background())

	// A: user_groups id=1 root, id=2 child of 1； closure 缺 (1,2,1)
	gdb.WithContext(skipCtx).Create(&UserGroup{
		Identifier: "tenant-A", Name: "A-root", Source: GroupSourceManual,
	})
	gdb.WithContext(skipCtx).Create(&UserGroup{
		Identifier: "tenant-A", Name: "A-child", ParentID: 1, Source: GroupSourceManual,
	})
	seedClosureRowRaw(t, "tenant-A", 1, 1, 0)
	seedClosureRowRaw(t, "tenant-A", 2, 2, 0)
	// 故意不写 (1,2,1)，让 reconcile 修

	// B: 同样 id=1 root + id=2 child；closure 完整
	gdb.WithContext(skipCtx).Create(&UserGroup{
		Identifier: "tenant-B", Name: "B-root", Source: GroupSourceManual,
	})
	gdb.WithContext(skipCtx).Create(&UserGroup{
		Identifier: "tenant-B", Name: "B-child", ParentID: 1, Source: GroupSourceManual,
	})
	seedClosureRowRaw(t, "tenant-B", 1, 1, 0)
	seedClosureRowRaw(t, "tenant-B", 2, 2, 0)
	seedClosureRowRaw(t, "tenant-B", 1, 2, 1)
	bBefore := findClosureForTenant(t, "tenant-B")

	// 在 A 的 ctx 下跑 reconcile
	ReconcileClosure(ctxA())

	// A：必须补出 (1,2,1)
	var n int64
	gdb.WithContext(skipCtx).Model(&GroupClosure{}).
		Where("identifier = ? AND ancestor_id = ? AND descendant_id = ?", "tenant-A", 1, 2).
		Count(&n)
	if n != 1 {
		t.Errorf("A 租户期望 reconcile 补出 (1,2,1)，实际命中 %d 行", n)
	}

	// B：原封不动
	bAfter := findClosureForTenant(t, "tenant-B")
	if len(bAfter) != len(bBefore) {
		t.Fatalf("B 行数被改动: before=%d after=%d", len(bBefore), len(bAfter))
	}
	for i := range bAfter {
		if bAfter[i] != bBefore[i] {
			t.Errorf("B 行 %d 被改动: before=%+v after=%+v", i, bBefore[i], bAfter[i])
		}
	}
}

// ctxB 当前未在测试中使用，但保留以便后续扩展（如双向跑 reconcile）。
var _ = ctxB
