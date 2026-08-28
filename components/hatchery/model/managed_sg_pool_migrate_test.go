package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestAutoMigrate_ManagedSGPool_AddRuleSetIDOnExistingTable 复现生产环境的一次崩溃：
// SQLite 库里已有上一版 sg-managed-sharding schema 的 managed_sg_pool 表（没有 rule_set_id 列），
// 新版 GORM 模型把 rule_set_id 标为 NOT NULL；AutoMigrate 生成的
// `ALTER TABLE managed_sg_pool ADD rule_set_id integer NOT NULL`
// 会被 SQLite 拒绝，报：
//
//	Cannot add a NOT NULL column with default value NULL
//
// 修复方式：给 RuleSetID 打 `default:0` tag，让 GORM 把 ALTER 改写成
// `ADD rule_set_id integer NOT NULL DEFAULT 0`，SQLite 接受。
//
// 本测试用一个内存 SQLite 数据库模拟"老 schema 已有数据"的场景，
// 再跑 AutoMigrate 验证能否补上 rule_set_id 列且不报错。
func TestAutoMigrate_ManagedSGPool_AddRuleSetIDOnExistingTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// Step 1: 伪造一个"老 schema" 的 managed_sg_pool 表（缺 rule_set_id）
	// 只保留本次迁移最关心的列即可。
	ddl := `CREATE TABLE managed_sg_pool (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		identifier TEXT DEFAULT '',
		sg_id TEXT NOT NULL,
		status TEXT DEFAULT 'ACTIVE',
		cvm_count INTEGER DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME
	)`
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("create old schema: %v", err)
	}

	// Step 2: 插入一行老数据
	if err := db.Exec(
		`INSERT INTO managed_sg_pool (identifier, sg_id, status, cvm_count) VALUES (?, ?, ?, ?)`,
		"tenant-1", "sg-legacy", "ACTIVE", 5,
	).Error; err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	// Step 3: 跑 AutoMigrate —— 这一步是之前崩溃的地方
	if err := db.AutoMigrate(&ManagedSGPool{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	// Step 4: 确认老行仍存在，且新列 rule_set_id 默认填 0
	var row ManagedSGPool
	if err := db.Where("sg_id = ?", "sg-legacy").First(&row).Error; err != nil {
		t.Fatalf("legacy row missing after migrate: %v", err)
	}
	if row.RuleSetID != 0 {
		t.Errorf("expected rule_set_id=0 for legacy row, got %d", row.RuleSetID)
	}
	if row.CVMCount != 5 {
		t.Errorf("expected cvm_count=5 preserved, got %d", row.CVMCount)
	}

	// Step 5: 新插入带 rule_set_id 的行，验证可写
	nr := ManagedSGPool{
		Identifier: "tenant-1",
		SGID:       "sg-new",
		RuleSetID:  42,
		Status:     SGStatusActive,
	}
	if err := db.Create(&nr).Error; err != nil {
		t.Fatalf("insert new row: %v", err)
	}
	var check ManagedSGPool
	if err := db.Where("sg_id = ?", "sg-new").First(&check).Error; err != nil {
		t.Fatalf("new row missing: %v", err)
	}
	if check.RuleSetID != 42 {
		t.Errorf("expected new row rule_set_id=42, got %d", check.RuleSetID)
	}
}
