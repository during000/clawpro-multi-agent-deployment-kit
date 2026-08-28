package model

import (
	"context"
	"testing"

	"hatchery/common"
)

// TestLocalAgentCLSCredential_TenantIsolation 验证 LocalAgentCLSCredential 的租户隔离：
// 凭据按 identifier 隔离，不同租户互不可见（依赖 GORM identifier 回调）。
func TestLocalAgentCLSCredential_TenantIsolation(t *testing.T) {
	cleanup := setupTestDB(t, "")
	defer cleanup()

	ctxSkip := common.WithSkipIdentifier(context.Background())
	if err := gdb.WithContext(ctxSkip).AutoMigrate(&LocalAgentCLSCredential{}); err != nil {
		t.Fatalf("auto migrate LocalAgentCLSCredential: %v", err)
	}

	// 两个租户各写一份 cls 凭据
	if err := gdb.WithContext(ctxSkip).Create(&LocalAgentCLSCredential{
		Identifier: "tenant-a", ConfigType: "cls", SecretID: "AKID-a", SecretKey: "key-a",
	}).Error; err != nil {
		t.Fatalf("create tenant-a: %v", err)
	}
	if err := gdb.WithContext(ctxSkip).Create(&LocalAgentCLSCredential{
		Identifier: "tenant-b", ConfigType: "cls", SecretID: "AKID-b", SecretKey: "key-b",
	}).Error; err != nil {
		t.Fatalf("create tenant-b: %v", err)
	}

	// 租户 A 的 ctx 只能看到 tenant-a 的凭据
	ctxA := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-a"})
	var credA LocalAgentCLSCredential
	if err := gdb.WithContext(ctxA).Where("config_type = ?", "cls").First(&credA).Error; err != nil {
		t.Fatalf("tenant-a 应查到自己的凭据: %v", err)
	}
	if credA.SecretID != "AKID-a" {
		t.Errorf("tenant-a 查到 SecretID=%s, want AKID-a", credA.SecretID)
	}

	// 租户 B 的 ctx 只能看到 tenant-b 的凭据
	ctxB := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-b"})
	var credB LocalAgentCLSCredential
	if err := gdb.WithContext(ctxB).Where("config_type = ?", "cls").First(&credB).Error; err != nil {
		t.Fatalf("tenant-b 应查到自己的凭据: %v", err)
	}
	if credB.SecretID != "AKID-b" {
		t.Errorf("tenant-b 查到 SecretID=%s, want AKID-b", credB.SecretID)
	}

	// 租户 C（无凭据）的 ctx 查不到 → 隔离生效
	ctxC := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-c"})
	var credC LocalAgentCLSCredential
	if err := gdb.WithContext(ctxC).Where("config_type = ?", "cls").First(&credC).Error; err == nil {
		t.Errorf("tenant-c 不应查到任何凭据，但查到了 SecretID=%s", credC.SecretID)
	}
}

// TestLocalAgentCLSCredential_UniqueIndex 验证 (identifier, config_type) 唯一索引。
func TestLocalAgentCLSCredential_UniqueIndex(t *testing.T) {
	cleanup := setupTestDB(t, "")
	defer cleanup()

	ctxSkip := common.WithSkipIdentifier(context.Background())
	if err := gdb.WithContext(ctxSkip).AutoMigrate(&LocalAgentCLSCredential{}); err != nil {
		t.Fatalf("auto migrate LocalAgentCLSCredential: %v", err)
	}
	if err := gdb.WithContext(ctxSkip).Create(&LocalAgentCLSCredential{
		Identifier: "tenant-a", ConfigType: "cls", SecretID: "AKID-a", SecretKey: "key-a",
	}).Error; err != nil {
		t.Fatalf("create first: %v", err)
	}
	// 同 identifier + config_type 重复插入应失败
	err := gdb.WithContext(ctxSkip).Create(&LocalAgentCLSCredential{
		Identifier: "tenant-a", ConfigType: "cls", SecretID: "AKID-a2", SecretKey: "key-a2",
	}).Error
	if err == nil {
		t.Error("同 (identifier, config_type) 重复插入应失败（唯一索引）")
	}
}
