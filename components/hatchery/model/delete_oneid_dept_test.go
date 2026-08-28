package model

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupDeleteOneIDDeptTestDB 为 DeleteUserGroupForOneIDDept 的兜底测试准备 schema。
// 需要 UserGroup / UserGroupMember / GroupClosure / Instance + GroupConfigBinding
// （CanDeleteUserGroup 的统一绑定表，虽然此处不会直接触发，但保留与现网一致）。
func setupDeleteOneIDDeptTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&UserGroup{},
		&UserGroupMember{},
		&GroupClosure{},
		&GroupConfigBinding{},
		&Instance{},
		&Tag{},
		&TagVisibilityGroup{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	gdb = db
}

// TestDeleteUserGroupForOneIDDept_BlockedByDirectInstance 模拟并发窗口：
// 上层 landing 已通过 CanDeleteUserGroup 预检（预检时 instances 表还没有 agent），
// 但在事务执行前另一个请求给该组创建了 agent。此时事务内兜底检查必须拦住，
// 避免把 instance.group_id 变孤儿。
func TestDeleteUserGroupForOneIDDept_BlockedByDirectInstance(t *testing.T) {
	setupDeleteOneIDDeptTestDB(t)

	g := UserGroup{Name: "dept", FullPath: "dept", Source: GroupSourceOneIDDept, SourceRef: "D1"}
	if err := gdb.Create(&g).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	// 写 closure 自指，模拟真实组（否则 closureDeleteNodeTx 也没行可删，但不影响本断言）
	if err := gdb.Create(&GroupClosure{AncestorID: g.ID, DescendantID: g.ID, Depth: 0}).Error; err != nil {
		t.Fatalf("seed closure: %v", err)
	}
	// "并发窗口"里被加进来的 agent
	if err := gdb.Create(&Instance{Name: "late-agent", InstanceId: "ins-xyz", GroupID: g.ID}).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	err := DeleteUserGroupForOneIDDept(context.Background(), g.ID)
	if !errors.Is(err, ErrGroupHasDependencies) {
		t.Fatalf("期望 ErrGroupHasDependencies，实际 %v", err)
	}

	// 组必须还在（软删不应发生）
	var count int64
	gdb.Model(&UserGroup{}).Where("id = ?", g.ID).Count(&count)
	if count != 1 {
		t.Fatalf("分组不应被删，实际 count=%d", count)
	}
}

// TestDeleteUserGroupForOneIDDept_Success 无任何阻塞时正常删除。
func TestDeleteUserGroupForOneIDDept_Success(t *testing.T) {
	setupDeleteOneIDDeptTestDB(t)

	g := UserGroup{Name: "dept", FullPath: "dept", Source: GroupSourceOneIDDept, SourceRef: "D1"}
	if err := gdb.Create(&g).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := gdb.Create(&GroupClosure{AncestorID: g.ID, DescendantID: g.ID, Depth: 0}).Error; err != nil {
		t.Fatalf("seed closure: %v", err)
	}
	// 添加 CLS scope 绑定，验证删除时一并清理
	gdb.Create(&GroupConfigBinding{GroupID: g.ID, ConfigType: ConfigTypeCLSCollectScope, ConfigKey: CLSCollectScopeKey})

	if err := DeleteUserGroupForOneIDDept(context.Background(), g.ID); err != nil {
		t.Fatalf("无阻塞时应成功删除，实际 %v", err)
	}
	// 软删：默认 gdb.Find 查不到
	var count int64
	gdb.Model(&UserGroup{}).Where("id = ?", g.ID).Count(&count)
	if count != 0 {
		t.Fatalf("分组应被软删，实际 count=%d", count)
	}
	// 验证 CLS scope 绑定也被清理
	var bindingCount int64
	gdb.Model(&GroupConfigBinding{}).Where("group_id = ? AND config_type = ?", g.ID, ConfigTypeCLSCollectScope).Count(&bindingCount)
	if bindingCount != 0 {
		t.Errorf("分组删除后 CLS scope 绑定应被清理，实际 count=%d", bindingCount)
	}
}
