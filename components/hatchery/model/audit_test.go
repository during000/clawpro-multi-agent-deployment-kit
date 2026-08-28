package model

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupAuditTestDB 初始化一个干净的内存 SQLite 数据库供审计测试使用。
func setupAuditTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	orig := UseDBForTest(db)
	return func() { orig() }
}

// TestLogAudit_Success：正常写入审计日志，验证所有字段正确持久化。
func TestLogAudit_Success(t *testing.T) {
	cleanup := setupAuditTestDB(t)
	defer cleanup()

	ctx := context.Background()
	startedAt := time.Now().Add(-5 * time.Minute)

	LogAudit(ctx, startedAt, 42, "alice", "agent_bridge_desktop_install", "instance", "ins-001", "success")

	var log AuditLog
	if err := DB(ctx).First(&log).Error; err != nil {
		t.Fatalf("查询审计记录失败: %v", err)
	}

	if log.UserID != 42 {
		t.Errorf("UserID 不匹配：want 42, got %d", log.UserID)
	}
	if log.Username != "alice" {
		t.Errorf("Username 不匹配：want 'alice', got %q", log.Username)
	}
	if log.Action != "agent_bridge_desktop_install" {
		t.Errorf("Action 不匹配：want 'agent_bridge_desktop_install', got %q", log.Action)
	}
	if log.Resource != "instance" {
		t.Errorf("Resource 不匹配：want 'instance', got %q", log.Resource)
	}
	if log.ResourceID != "ins-001" {
		t.Errorf("ResourceID 不匹配：want 'ins-001', got %q", log.ResourceID)
	}
	if log.Status != "success" {
		t.Errorf("Status 不匹配：want 'success', got %q", log.Status)
	}
	// StartedAt 应接近传入的时间
	if log.StartedAt.Sub(startedAt) > time.Second {
		t.Errorf("StartedAt 不匹配：want ~%v, got %v", startedAt, log.StartedAt)
	}
	// CreatedAt 应由 GORM 自动填充
	if log.CreatedAt.IsZero() {
		t.Errorf("CreatedAt 不应为零值")
	}
}

// TestLogAudit_MultipleRecords：连续写入多条审计记录，验证不会互相覆盖。
func TestLogAudit_MultipleRecords(t *testing.T) {
	cleanup := setupAuditTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	LogAudit(ctx, now, 1, "user1", "agent_bridge_desktop_install", "instance", "ins-001", "dispatched")
	LogAudit(ctx, now, 1, "user1", "agent_bridge_desktop_install", "instance", "ins-001", "success")
	LogAudit(ctx, now, 2, "user2", "agent_bridge_desktop_check", "instance", "ins-002", "failed")

	var count int64
	DB(ctx).Model(&AuditLog{}).Count(&count)
	if count != 3 {
		t.Errorf("应有 3 条审计记录，实际 %d", count)
	}

	// 验证同一 resource_id 可以有多条不同 status 的记录（幂等写入）
	var logs []AuditLog
	DB(ctx).Where("resource_id = ?", "ins-001").Find(&logs)
	if len(logs) != 2 {
		t.Errorf("ins-001 应有 2 条记录，实际 %d", len(logs))
	}
}

// TestLogAudit_EmptyFields：空字段写入不会 panic，正常落库。
func TestLogAudit_EmptyFields(t *testing.T) {
	cleanup := setupAuditTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// 所有可选字段为空
	LogAudit(ctx, now, 0, "", "", "", "", "")

	var log AuditLog
	if err := DB(ctx).First(&log).Error; err != nil {
		t.Fatalf("空字段写入失败: %v", err)
	}
	if log.UserID != 0 {
		t.Errorf("UserID 应为 0，实际 %d", log.UserID)
	}
}

// TestLogAudit_ErrorHandling：当数据库不可用时，LogAudit 不 panic（仅记录错误日志）。
// 通过关闭数据库连接模拟错误场景。
func TestLogAudit_ErrorHandling(t *testing.T) {
	cleanup := setupAuditTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 关闭数据库连接，模拟写入失败
	sqlDB, err := DB(ctx).DB()
	if err != nil {
		t.Fatalf("获取 sql.DB 失败: %v", err)
	}
	sqlDB.Close()

	// LogAudit 应该不 panic，即使写入失败
	// 这里主要验证错误处理分支不会导致程序崩溃
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("LogAudit 在数据库不可用时不应 panic，实际 panic: %v", r)
		}
	}()

	LogAudit(ctx, time.Now(), 1, "test", "test_action", "test_resource", "test-id", "failed")
}

func TestAuditLog_TenantScopedQueryIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	for _, indexName := range []string{
		"idx_audit_logs_user_id",
		"idx_audit_logs_identifier_user_id",
		"idx_audit_logs_identifier_username",
		"idx_audit_logs_identifier_resource_id",
	} {
		if !db.Migrator().HasIndex(&AuditLog{}, indexName) {
			t.Errorf("AutoMigrate 未创建索引 %s", indexName)
		}
	}
}
