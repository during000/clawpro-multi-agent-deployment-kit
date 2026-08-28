package model

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupMigrateTestDB 准备 user_groups + group_closure 两张表的内存 SQLite。
func setupMigrateTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试 DB 失败: %v", err)
	}
	if err := db.AutoMigrate(&UserGroup{}, &GroupClosure{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	gdb = db
}

// countClosure 返回 group_closure 行数（含通过 INSERT 写入的）。
func countClosure(t *testing.T) int64 {
	t.Helper()
	var n int64
	if err := gdb.Table("group_closure").Count(&n).Error; err != nil {
		t.Fatalf("count closure 失败: %v", err)
	}
	return n
}

// TestMigrateUserGroupClosureAndFullPath_NoGroups 没有任何 user_groups 行 → no-op。
func TestMigrateUserGroupClosureAndFullPath_NoGroups(t *testing.T) {
	setupMigrateTestDB(t)

	MigrateUserGroupClosureAndFullPath(gdb)

	if got := countClosure(t); got != 0 {
		t.Errorf("没有分组时不应写入 closure，实际 %d 行", got)
	}
}

// TestMigrateUserGroupClosureAndFullPath_ExistingClosure 闭包已有数据 → no-op。
func TestMigrateUserGroupClosureAndFullPath_ExistingClosure(t *testing.T) {
	setupMigrateTestDB(t)

	// 先插一个分组 + 一条已存在的闭包行（模拟之前已迁移过）
	if err := gdb.Create(&UserGroup{Name: "存量组", Source: GroupSourceManual}).Error; err != nil {
		t.Fatalf("插入分组失败: %v", err)
	}
	if err := gdb.Create(&GroupClosure{
		AncestorID: 999, DescendantID: 999, Depth: 0,
	}).Error; err != nil {
		t.Fatalf("插入 closure 失败: %v", err)
	}
	beforeRows := countClosure(t)

	MigrateUserGroupClosureAndFullPath(gdb)

	if got := countClosure(t); got != beforeRows {
		t.Errorf("已有 closure 时不应再写入，期望 %d 实际 %d", beforeRows, got)
	}
	// full_path 也不应被改写
	var g UserGroup
	if err := gdb.First(&g).Error; err != nil {
		t.Fatal(err)
	}
	if g.FullPath != "" {
		t.Errorf("已有 closure 时不应回填 full_path，实际 %q", g.FullPath)
	}
}

// TestMigrateUserGroupClosureAndFullPath_HappyPath 多个根组、闭包为空 → 写入自指行 + 回填 full_path。
func TestMigrateUserGroupClosureAndFullPath_HappyPath(t *testing.T) {
	setupMigrateTestDB(t)

	groups := []UserGroup{
		{Name: "研发中心", Source: GroupSourceManual},
		{Name: "运营组", Source: GroupSourceManual},
		{Name: "测试组", Source: GroupSourceManual},
	}
	for i := range groups {
		if err := gdb.Create(&groups[i]).Error; err != nil {
			t.Fatalf("插入分组失败: %v", err)
		}
	}

	MigrateUserGroupClosureAndFullPath(gdb)

	// 闭包应有 3 行自指
	if got := countClosure(t); got != 3 {
		t.Errorf("期望 3 行自指闭包，实际 %d", got)
	}
	var rows []GroupClosure
	if err := gdb.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.AncestorID != r.DescendantID {
			t.Errorf("应为自指 (a=d)，实际 a=%d d=%d", r.AncestorID, r.DescendantID)
		}
		if r.Depth != 0 {
			t.Errorf("自指 depth 应为 0，实际 %d", r.Depth)
		}
	}

	// full_path 全部回填为 name
	var fresh []UserGroup
	if err := gdb.Order("id ASC").Find(&fresh).Error; err != nil {
		t.Fatal(err)
	}
	wantPaths := []string{"研发中心", "运营组", "测试组"}
	for i, g := range fresh {
		if g.FullPath != wantPaths[i] {
			t.Errorf("g[%d].full_path 期望 %q 实际 %q", i, wantPaths[i], g.FullPath)
		}
	}
}

// TestMigrateUserGroupClosureAndFullPath_OnlySoftDeleted 已废弃：
// user_groups 现在不再使用 GORM 软删，"仅有软删行"场景不再存在。
// 保留空 case 占位以避免误以为漏测；如需新增"仅含某种状态行 → no-op"
// 的回归，参考 _NoGroups 即可。
func TestMigrateUserGroupClosureAndFullPath_OnlySoftDeleted(t *testing.T) {
	t.Skip("user_groups 已移除 deleted_at（GORM 软删），此场景不再适用")
}

// TestMigrateUserGroupClosureAndFullPath_LivePlusSoftDeleted 已废弃：
// 同上，user_groups 没有软删态，gdb.Delete 即物理删除。
func TestMigrateUserGroupClosureAndFullPath_LivePlusSoftDeleted(t *testing.T) {
	t.Skip("user_groups 已移除 deleted_at（GORM 软删），此场景不再适用")
}

// TestMigrateUserGroupClosureAndFullPath_FullPathNotEmpty 已有 full_path 的行不被覆盖。
func TestMigrateUserGroupClosureAndFullPath_FullPathNotEmpty(t *testing.T) {
	setupMigrateTestDB(t)

	g := UserGroup{
		Name:     "已有路径",
		FullPath: "自定义/路径",
		Source:   GroupSourceManual,
	}
	if err := gdb.Create(&g).Error; err != nil {
		t.Fatal(err)
	}

	MigrateUserGroupClosureAndFullPath(gdb)

	var fresh UserGroup
	if err := gdb.First(&fresh, g.ID).Error; err != nil {
		t.Fatal(err)
	}
	if fresh.FullPath != "自定义/路径" {
		t.Errorf("已有 full_path 不应被覆盖，实际 %q", fresh.FullPath)
	}
}

// TestMigrateUserGroupClosureAndFullPath_DropsLegacyIndexOnSQLite 验证迁移
// 函数在 SQLite 后端下顺带删除 0414 遗留的 idx_ug_identifier_name 唯一索引。
//
// 背景：该索引会让同名不同父的子组写入触发 "UNIQUE constraint failed:
// user_groups.identifier, user_groups.name" —— 对应线上 OneID 同步报错。
//
// 场景：
//  1. seed 一个老库：建好 user_groups + 索引 + 一行存量 + 空 closure
//  2. 跑迁移
//  3. 之后写 (parent=2, name='测试2') 应成功（之前会冲突）
func TestMigrateUserGroupClosureAndFullPath_DropsLegacyIndexOnSQLite(t *testing.T) {
	setupMigrateTestDB(t)

	// 模拟历史库：人工建一个 (identifier, name) 全局唯一索引
	if err := gdb.Exec("CREATE UNIQUE INDEX idx_ug_identifier_name ON user_groups(identifier, name)").Error; err != nil {
		t.Fatalf("seed legacy index 失败: %v", err)
	}

	// 一行存量分组（让迁移条件成立: groupCount > 0 && closureCount == 0）
	if err := gdb.Create(&UserGroup{Name: "测试2", ParentID: 1, Source: GroupSourceManual}).Error; err != nil {
		t.Fatalf("create first: %v", err)
	}

	// 验证迁移前同名不同父冲突
	if err := gdb.Create(&UserGroup{Name: "测试2", ParentID: 2, Source: GroupSourceManual}).Error; err == nil {
		t.Fatal("迁移前预期同名不同父冲突，实际无错误")
	}

	// 触发迁移
	MigrateUserGroupClosureAndFullPath(gdb)

	// 迁移后再写应成功
	if err := gdb.Create(&UserGroup{Name: "测试2", ParentID: 2, Source: GroupSourceManual}).Error; err != nil {
		t.Errorf("迁移后同名不同父仍冲突: %v", err)
	}
}

// TestMigrateUserGroupClosureAndFullPath_DropsLegacyIndex_EvenWhenGateSkips
// 当 closure 已经有数据(gate 短路 closure/full_path backfill)时，索引清理仍需发生。
// 这是测试环境的真实场景：之前已经跑过迁移、closure 已有数据，但旧索引仍残留。
func TestMigrateUserGroupClosureAndFullPath_DropsLegacyIndex_EvenWhenGateSkips(t *testing.T) {
	setupMigrateTestDB(t)

	// seed 老索引
	if err := gdb.Exec("CREATE UNIQUE INDEX idx_ug_identifier_name ON user_groups(identifier, name)").Error; err != nil {
		t.Fatalf("seed legacy index 失败: %v", err)
	}
	// 写一行分组 + 写一行 closure 让 gate 短路（closureCount > 0）
	g := UserGroup{Name: "测试2", ParentID: 1, Source: GroupSourceManual}
	if err := gdb.Create(&g).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Create(&GroupClosure{AncestorID: g.ID, DescendantID: g.ID, Depth: 0}).Error; err != nil {
		t.Fatal(err)
	}

	// 验证迁移前同名不同父冲突
	if err := gdb.Create(&UserGroup{Name: "测试2", ParentID: 2, Source: GroupSourceManual}).Error; err == nil {
		t.Fatal("迁移前预期同名不同父冲突，实际无错误")
	}

	// 跑迁移：closure gate 应短路 backfill，但 SQLite 索引清理应仍执行
	MigrateUserGroupClosureAndFullPath(gdb)

	// 现在再写应成功
	if err := gdb.Create(&UserGroup{Name: "测试2", ParentID: 2, Source: GroupSourceManual}).Error; err != nil {
		t.Errorf("迁移后同名不同父仍冲突: %v", err)
	}
}

// TestMigrateUserGroupClosureAndFullPath_NilDB DB 为 nil → 安全跳过。
func TestMigrateUserGroupClosureAndFullPath_NilDB(t *testing.T) {
	orig := gdb
	defer func() { gdb = orig }()
	gdb = nil

	// 不应 panic
	done := make(chan struct{})
	go func() {
		defer close(done)
		MigrateUserGroupClosureAndFullPath(gdb)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("gdb = nil 时迁移应立即返回，疑似阻塞")
	}
}
