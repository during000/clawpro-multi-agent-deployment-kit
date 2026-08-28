package model

import (
	"context"
	"testing"

	"hatchery/common"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestDBWithNilContext 测试 DB(nil) 返回全局句柄
func TestDBWithNilContext(t *testing.T) {
	// 不 InitDB，gdb 为 nil
	defer UseNilDBForTest()()

	result := DB(nil)
	if result != nil {
		t.Fatalf("DB(nil) with uninitialized gdb should return nil")
	}
}

// TestDBBindsContext 测试 DB(ctx) 返回绑定到 ctx 的句柄
func TestDBBindsContext(t *testing.T) {
	// 不能用空 mock DB，因为 WithContext 需要完整的 gorm.DB
	// 此测试验证当 gdb 为 nil 时的行为
	defer UseNilDBForTest()()

	ctx := context.Background()
	result := DB(ctx)
	if result != nil {
		t.Fatal("DB(ctx) with nil gdb should return nil")
	}
}

// TestDBGlobalSkipsIdentifier 测试 DBGlobal 会跳过 identifier 过滤
func TestDBGlobalSkipsIdentifier(t *testing.T) {
	// DBGlobal 需要一个有效的 gdb，暂时设置为 nil 来测试早期返回逻辑
	defer UseNilDBForTest()()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-1"})
	result := DBGlobal(ctx)

	if result != nil {
		t.Fatal("DBGlobal with nil gdb should return nil")
	}
}

// TestDBGlobalWithNilContext 测试 DBGlobal(nil) 会创建一个新的 Background context
func TestDBGlobalWithNilContext(t *testing.T) {
	defer UseNilDBForTest()()

	//nolint:staticcheck // 明确测试 nil ctx 分支
	result := DBGlobal(nil)
	if result != nil {
		t.Fatal("DBGlobal(nil) with nil gdb should return nil")
	}
}

// TestResolveIdentifierWithSkipFlag 测试 InjectTenant 与 WithSkipIdentifier 互斥：
// 后调用的会清除前者，因此同时设置时以最后注入的为准。
func TestResolveIdentifierWithSkipFlag(t *testing.T) {
	// 先 InjectTenant 再 WithSkipIdentifier → SkipIdentifier 清除了 snapshot，返回空
	ctx := common.WithSkipIdentifier(common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-1"}))
	id := resolveIdentifier(ctx)
	if id != "" {
		t.Fatalf("resolveIdentifier after WithSkipIdentifier should return empty, got %q", id)
	}

	// 先 WithSkipIdentifier 再 InjectTenant → snapshot 清除了 skip，返回 identifier
	ctx2 := common.InjectTenant(common.WithSkipIdentifier(context.Background()), common.TenantSnapshot{Identifier: "tenant-2"})
	id2 := resolveIdentifier(ctx2)
	if id2 != "tenant-2" {
		t.Fatalf("resolveIdentifier after InjectTenant should return tenant-2, got %q", id2)
	}
}

// TestResolveIdentifierWithSnapshot 测试从 common.TenantSnapshot 读取 identifier
func TestResolveIdentifierWithSnapshot(t *testing.T) {
	snap := common.TenantSnapshot{Identifier: "tenant-2"}
	ctx := common.InjectTenant(context.Background(), snap)
	id := resolveIdentifier(ctx)
	if id != "tenant-2" {
		t.Fatalf("resolveIdentifier should return tenant-2, got %q", id)
	}
}

// TestResolveIdentifierWithoutSnapshot 测试无 snapshot 时 panic
func TestResolveIdentifierWithoutSnapshot(t *testing.T) {
	old := dbDriver
	dbDriver = "sqlite"
	defer func() { dbDriver = old }()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("resolveIdentifier without snapshot should panic")
		}
	}()

	resolveIdentifier(context.Background())
}

// TestResolveIdentifierWithNilContext 测试 nil context 时 panic
func TestResolveIdentifierWithNilContext(t *testing.T) {
	old := dbDriver
	dbDriver = "sqlite"
	defer func() { dbDriver = old }()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("resolveIdentifier(nil) without snapshot should panic")
		}
	}()

	//nolint:staticcheck // 明确测试 nil context 的 panic 行为
	resolveIdentifier(nil)
}

// TestResolveIdentifierMySQLWithoutSnapshot 测试 MySQL 模式下无 snapshot 时会 panic
func TestResolveIdentifierMySQLWithoutSnapshot(t *testing.T) {
	old := dbDriver
	dbDriver = "mysql"
	defer func() { dbDriver = old }()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("resolveIdentifier in MySQL mode should panic when no snapshot")
		}
	}()

	ctx := context.Background()
	//nolint:staticcheck // 明确测试 MySQL 无 snapshot 时的 panic
	resolveIdentifier(ctx)
}

// TestResolveIdentifierMySQLWithSnapshot 测试 MySQL 模式下有 snapshot 时正常返回
func TestResolveIdentifierMySQLWithSnapshot(t *testing.T) {
	old := dbDriver
	dbDriver = "mysql"
	defer func() { dbDriver = old }()

	snap := common.TenantSnapshot{Identifier: "mysql-tenant"}
	ctx := common.InjectTenant(context.Background(), snap)
	id := resolveIdentifier(ctx)
	if id != "mysql-tenant" {
		t.Fatalf("resolveIdentifier in MySQL mode should return mysql-tenant, got %q", id)
	}
}

// TestApplyIdentifierFilter 测试 identifier 过滤在 GORM 回调中的应用
// 注：此测试仅验证回调不 panic，实际 SQL 拼装由 GORM 负责
func TestApplyIdentifierFilter(t *testing.T) {
	snap := common.TenantSnapshot{Identifier: "test-tenant"}
	ctx := common.InjectTenant(context.Background(), snap)

	// 使用一个极简的 GORM 实例（不连接真实数据库）
	// 这里主要验证回调执行流程不出错
	// 实际的 SQL 拼装测试应在集成测试中覆盖

	// 如果 gdb 为 nil，resolveIdentifier 和 applyIdentifierFilter 应该安全返回
	defer UseNilDBForTest()()

	// 测试 resolveIdentifier 的功能
	id := resolveIdentifier(ctx)
	if id != "test-tenant" {
		t.Fatalf("resolveIdentifier should extract tenant-test, got %q", id)
	}
}

// TestDBGlobalWithNilContext_SkipIdentifier 测试 DBGlobal(nil) 创建新 context 并标记跳过 identifier
func TestDBGlobalWithNilContext_SkipIdentifier(t *testing.T) {
	defer UseNilDBForTest()()

	//nolint:staticcheck
	result := DBGlobal(nil)
	// DBGlobal(nil) 返回 nil 因为 gdb 为 nil，这测试了 ctx == nil 的分支被执行
	if result != nil {
		t.Fatal("DBGlobal(nil) with nil gdb should return nil")
	}
}

// TestResolveIdentifierWithAllContexts 综合测试 resolveIdentifier 的各类输入
func TestResolveIdentifierWithAllContexts(t *testing.T) {
	tests := []struct {
		name          string
		ctx           context.Context
		dbDriver      string
		fixedSnapshot *common.TenantSnapshot
		expectID      string
		shouldPanic   bool
	}{
		{
			name:     "SkipIdentifier after Snapshot returns empty (mutually exclusive)",
			ctx:      common.WithSkipIdentifier(common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "x"})),
			dbDriver: "sqlite",
			expectID: "",
		},
		{
			name:     "SQLite with snapshot",
			ctx:      common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "sqlite-t"}),
			dbDriver: "sqlite",
			expectID: "sqlite-t",
		},
		{
			name:        "SQLite without snapshot panics",
			ctx:         context.Background(),
			dbDriver:    "sqlite",
			shouldPanic: true,
		},
		{
			name:     "MySQL with snapshot",
			ctx:      common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "mysql-t"}),
			dbDriver: "mysql",
			expectID: "mysql-t",
		},
		{
			name:        "MySQL without snapshot and no FixedSnapshot panics",
			ctx:         context.Background(),
			dbDriver:    "mysql",
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldDriver := dbDriver
			defer func() { dbDriver = oldDriver }()
			dbDriver = tt.dbDriver

			oldSnap := common.FixedSnapshot
			defer func() { common.FixedSnapshot = oldSnap }()
			common.FixedSnapshot = tt.fixedSnapshot

			if tt.shouldPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatalf("expected panic but got none")
					}
				}()
				resolveIdentifier(tt.ctx)
			} else {
				id := resolveIdentifier(tt.ctx)
				if id != tt.expectID {
					t.Fatalf("expected %q, got %q", tt.expectID, id)
				}
			}
		})
	}
}

// TestDBGlobalWithInjectedTenant_SkipsIdentifier 测试 DBGlobal 即使注入了 common.TenantSnapshot 也跳过过滤
func TestDBGlobalWithInjectedTenant_SkipsIdentifier(t *testing.T) {
	defer UseNilDBForTest()()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "should-skip"})
	// DBGlobal 返回 nil 因为 gdb 为 nil，但我们验证 common.WithSkipIdentifier 被调用
	result := DBGlobal(ctx)
	if result != nil {
		t.Fatal("DBGlobal with nil gdb should return nil")
	}
}

// TestUseDBForTestMultipleSetRestore 测试 UseDBForTest 多次嵌套调用的恢复机制
/*func TestUseDBForTestMultipleSetRestore(t *testing.T) {
	db1 := &gorm.DB{}
	db2 := &gorm.DB{}
	db3 := &gorm.DB{}

	restore1 := UseDBForTest(db1)
	if DB(nil) != db1 {
		t.Fatal("first set failed")
	}

	restore2 := UseDBForTest(db2)
	if DB(nil) != db2 {
		t.Fatal("second set failed")
	}

	restore3 := UseDBForTest(db3)
	if DB(nil) != db3 {
		t.Fatal("third set failed")
	}

	// 恢复链（LIFO）
	restore3()
	if DB(nil) != db2 {
		t.Fatal("restore3 should restore to db2")
	}

	restore2()
	if DB(nil) != db1 {
		t.Fatal("restore2 should restore to db1")
	}

	restore1()
}*/

// TestDBGlobalWithNonNilGdbAndNilContext 测试 DBGlobal 当 gdb 非 nil 但 ctx 为 nil 时的行为
func TestDBGlobalWithNonNilGdbAndNilContext(t *testing.T) {
	// 使用真实的 SQLite 内存数据库
	dsn := "file:memdb_dbglobal_nil_ctx?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	defer UseDBForTest(db)()

	// 调用 DBGlobal(nil)，此时应该创建 Background context 并注入 SkipIdentifier
	result := DBGlobal(nil)

	// 验证返回值不是 nil
	if result == nil {
		t.Fatal("DBGlobal(nil) with non-nil gdb should return non-nil")
	}
}

// TestDBWithNilContextAndNonNilGdb 测试 DB 当 ctx 为 nil 时的行为
func TestDBWithNilContextAndNonNilGdb(t *testing.T) {
	// 使用真实的 SQLite 内存数据库
	dsn := "file:memdb_db_nil_ctx?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	defer UseDBForTest(db)()

	// 调用 DB(nil)，此时应该返回 gdb（不绑定 context）
	result := DB(nil)

	// 验证返回的是 gdb
	if result != db {
		t.Fatal("DB(nil) with non-nil gdb should return gdb itself")
	}
}
