package controller

import (
	"context"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupCosCommonTestDB 初始化一个内存 DB，供 cos.go 里涉及 SiteConfig / SMHSpace 的测试使用。
// 返回 cleanup 闭包，恢复 model.DB 状态。
func setupCosCommonTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 DB 失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.SiteConfig{},
		&model.SMHSpace{},
		// 同时迁移其他测试常用表，避免其他测试遗留的异步 goroutine 在
		// 本测试期间访问 model.DB 时触发 no-such-table 错误。
		&model.User{},
		&model.Instance{},
		&model.Notification{},
		&model.AIImage{},
	); err != nil {
		t.Fatalf("迁移表失败: %v", err)
	}
	origDB := model.UseDBForTest(db)

	return func() {
		// 给可能泄漏的异步 goroutine 充分时间完成
		time.Sleep(50 * time.Millisecond)
		origDB()
	}
}

// seedSMHFullyConfigured 写入完整的 SiteConfig + common/skillhub 两个 SMHSpace，
// 使 SMHConfig.IsConfigured() 返回 true。
func seedSMHFullyConfigured(t *testing.T) {
	t.Helper()
	if err := model.DB(context.Background()).Create(&model.SiteConfig{
		SMHEnabled:       1,
		SMHEndpoint:      "https://smh.example.com",
		SMHLibraryId:     "lib-test",
		SMHLibrarySecret: "secret-test",
	}).Error; err != nil {
		t.Fatalf("写入 SiteConfig 失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.SMHSpace{
		SpaceTag: "common", SpaceId: "sp-common", LibraryId: "lib-test", Purpose: "common",
	}).Error; err != nil {
		t.Fatalf("写入 common space 失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.SMHSpace{
		SpaceTag: "skillhub", SpaceId: "sp-skillhub", LibraryId: "lib-test", Purpose: "skillhub",
	}).Error; err != nil {
		t.Fatalf("写入 skillhub space 失败: %v", err)
	}
}

// ============================================================================
// FindLatestSMHCommonBackup 前置校验分支测试
// ============================================================================

func TestUpgradeBackupPrefix(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name      string
		agentType string
		want      string
	}{
		{name: "openclaw", agentType: model.AgentTypeOpenClaw, want: "openclaw-state-"},
		{name: "hermes", agentType: model.AgentTypeHermes, want: "hermes-state-"},
		{name: "legacy empty", agentType: "", want: "openclaw-state-"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := upgradeBackupPrefix(ctx, tt.agentType); got != tt.want {
				t.Fatalf("upgradeBackupPrefix(%q)=%q, want %q", tt.agentType, got, tt.want)
			}
		})
	}
}

// TestFindLatestSMHCommonBackup_NotConfigured 覆盖 "SMH 未配置" 分支。
func TestFindLatestSMHCommonBackup_NotConfigured(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()

	// 未 seed 任何 SMH 配置 → IsConfigured() 为 false
	fileKey, found, err := FindLatestSMHCommonBackup(context.Background(), "ins-1", "")
	if err == nil {
		t.Error("未配置时应返回 error")
	}
	if found {
		t.Error("未配置时 found 应为 false")
	}
	if fileKey != "" {
		t.Errorf("未配置时 fileKey 应为空，实际=%q", fileKey)
	}
}

// TestFindLatestSMHCommonBackup_EmptyInstanceId 覆盖 "instanceId 为空" 分支。
func TestFindLatestSMHCommonBackup_EmptyInstanceId(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	fileKey, found, err := FindLatestSMHCommonBackup(context.Background(), "", "")
	if err == nil {
		t.Error("instanceId 为空时应返回 error")
	}
	if found {
		t.Error("instanceId 为空时 found 应为 false")
	}
	if fileKey != "" {
		t.Errorf("instanceId 为空时 fileKey 应为空，实际=%q", fileKey)
	}
}

// TestFindLatestSMHCommonBackup_TokenNotAvailable 覆盖 "token 不可用" 分支。
// 完整 seed 了 SMH 配置使 IsConfigured()=true，但 DB 中无 token 且无法连接 SMH 自愈。
func TestFindLatestSMHCommonBackup_TokenNotAvailable(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	// DB 中无 token，自愈会因无法连接 SMH 而失败
	fileKey, found, err := FindLatestSMHCommonBackup(context.Background(), "ins-valid", "")
	if err == nil {
		t.Error("token 不可用时应返回 error")
	}
	if found {
		t.Error("token 不可用时 found 应为 false")
	}
	if fileKey != "" {
		t.Errorf("token 不可用时 fileKey 应为空，实际=%q", fileKey)
	}
}

// ============================================================================
// DB-based SMH Token 管理测试
// ============================================================================

// TestGetSMHSpaceRecord_Found 验证 GetSMHSpaceRecord 能读到完整记录。
func TestGetSMHSpaceRecord_Found(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	space, found := model.GetSMHSpaceRecord(context.Background(), "common")
	if !found {
		t.Fatal("应找到 common space")
	}
	if space.SpaceId != "sp-common" {
		t.Errorf("SpaceId=%q, want sp-common", space.SpaceId)
	}
	if space.AdminToken != "" {
		t.Error("初始 AdminToken 应为空")
	}
}

// TestGetSMHSpaceRecord_NotFound 验证不存在的 spaceTag 返回 found=false。
func TestGetSMHSpaceRecord_NotFound(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()

	_, found := model.GetSMHSpaceRecord(context.Background(), "nonexistent")
	if found {
		t.Error("不存在的 spaceTag 应返回 found=false")
	}
}

// TestUpdateSMHSpaceToken_Admin 验证更新 admin token。
func TestUpdateSMHSpaceToken_Admin(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	expiredAt := time.Now().Add(24 * time.Hour).Unix()
	if err := model.UpdateSMHSpaceToken(context.Background(), "common", true, "admin-tok-123", expiredAt); err != nil {
		t.Fatalf("UpdateSMHSpaceToken 失败: %v", err)
	}

	space, _ := model.GetSMHSpaceRecord(context.Background(), "common")
	if space.AdminToken != "admin-tok-123" {
		t.Errorf("AdminToken=%q, want admin-tok-123", space.AdminToken)
	}
	if space.AdminTokenExpiredAt != expiredAt {
		t.Errorf("AdminTokenExpiredAt=%d, want %d", space.AdminTokenExpiredAt, expiredAt)
	}
	// read token 不应被影响
	if space.ReadToken != "" {
		t.Errorf("ReadToken 应保持空，实际=%q", space.ReadToken)
	}
}

// TestUpdateSMHSpaceToken_Read 验证更新 read token。
func TestUpdateSMHSpaceToken_Read(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	expiredAt := time.Now().Add(24 * time.Hour).Unix()
	if err := model.UpdateSMHSpaceToken(context.Background(), "skillhub", false, "read-tok-456", expiredAt); err != nil {
		t.Fatalf("UpdateSMHSpaceToken 失败: %v", err)
	}

	space, _ := model.GetSMHSpaceRecord(context.Background(), "skillhub")
	if space.ReadToken != "read-tok-456" {
		t.Errorf("ReadToken=%q, want read-tok-456", space.ReadToken)
	}
}

// TestGetSpaceToken_FromDB_Valid 验证从 DB 读取有效 admin token。
func TestGetSpaceToken_FromDB_Valid(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	expiredAt := time.Now().Add(24 * time.Hour).Unix()
	model.UpdateSMHSpaceToken(context.Background(), "common", true, "valid-admin-token", expiredAt)

	token, err := getSpaceToken(context.Background(), "common", true)
	if err != nil {
		t.Fatalf("应成功: %v", err)
	}
	if token != "valid-admin-token" {
		t.Errorf("token=%q, want valid-admin-token", token)
	}
}

// TestGetSpaceToken_FromDB_Read 验证从 DB 读取有效 read token。
func TestGetSpaceToken_FromDB_Read(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	expiredAt := time.Now().Add(24 * time.Hour).Unix()
	model.UpdateSMHSpaceToken(context.Background(), "skillhub", false, "valid-read-token", expiredAt)

	token, err := getSpaceToken(context.Background(), "skillhub", false)
	if err != nil {
		t.Fatalf("应成功: %v", err)
	}
	if token != "valid-read-token" {
		t.Errorf("token=%q, want valid-read-token", token)
	}
}

// TestGetSpaceToken_SpaceNotFound 验证 space 不存在时返回错误。
func TestGetSpaceToken_SpaceNotFound(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()

	_, err := getSpaceToken(context.Background(), "nonexistent", true)
	if err == nil {
		t.Error("space 不存在时应返回 error")
	}
}

// TestGetSpaceToken_ExpiredTriggersSelfHeal 验证过期 token 触发自愈（无 SMH 服务应失败）。
func TestGetSpaceToken_ExpiredTriggersSelfHeal(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	expiredAt := time.Now().Add(-1 * time.Hour).Unix()
	model.UpdateSMHSpaceToken(context.Background(), "common", true, "expired-token", expiredAt)

	_, err := getSpaceToken(context.Background(), "common", true)
	if err == nil {
		t.Error("过期 token 且无法自愈时应返回 error")
	}
}

// TestGetCommonSpaceToken_FromDB 验证公共入口函数。
func TestGetCommonSpaceToken_FromDB(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	expiredAt := time.Now().Add(24 * time.Hour).Unix()
	model.UpdateSMHSpaceToken(context.Background(), "common", true, "common-admin", expiredAt)

	token, err := GetCommonSpaceToken(context.Background())
	if err != nil {
		t.Fatalf("应成功: %v", err)
	}
	if token != "common-admin" {
		t.Errorf("token=%q, want common-admin", token)
	}
}

// TestGetCommonSpaceReadToken_FromDB 验证只读 token 入口函数。
func TestGetCommonSpaceReadToken_FromDB(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	expiredAt := time.Now().Add(24 * time.Hour).Unix()
	model.UpdateSMHSpaceToken(context.Background(), "common", false, "common-read", expiredAt)

	token, err := GetCommonSpaceReadToken(context.Background())
	if err != nil {
		t.Fatalf("应成功: %v", err)
	}
	if token != "common-read" {
		t.Errorf("token=%q, want common-read", token)
	}
}

// TestGetSkillhubSpaceReadToken_FromDB 验证 skillhub 只读 token 入口函数。
func TestGetSkillhubSpaceReadToken_FromDB(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	expiredAt := time.Now().Add(24 * time.Hour).Unix()
	model.UpdateSMHSpaceToken(context.Background(), "skillhub", false, "skillhub-read", expiredAt)

	token, err := GetSkillhubSpaceReadToken(context.Background())
	if err != nil {
		t.Fatalf("应成功: %v", err)
	}
	if token != "skillhub-read" {
		t.Errorf("token=%q, want skillhub-read", token)
	}
}

// TestNewSMHAPIClient 验证 newSMHAPIClient 创建非 nil client。
func TestNewSMHAPIClient(t *testing.T) {
	c := newSMHAPIClient("https://smh.example.com")
	if c == nil {
		t.Fatal("newSMHAPIClient 不应返回 nil")
	}
}

// TestNewSMHClientConfig_NotConfigured 验证未配置时返回错误。
func TestNewSMHClientConfig_NotConfigured(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()

	_, _, err := newSMHClientConfig(context.Background())
	if err == nil {
		t.Error("未配置时应返回 error")
	}
}

// TestNewSMHClientConfig_Configured 验证配置后成功创建。
func TestNewSMHClientConfig_Configured(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	apiClient, cfg, err := newSMHClientConfig(context.Background())
	if err != nil {
		t.Fatalf("应成功: %v", err)
	}
	if apiClient == nil || cfg == nil {
		t.Error("返回值不应为 nil")
	}
}

// TestGetStorageClient_ReturnsClient 验证 getStorageClient 返回正确客户端。
func TestGetStorageClient_ReturnsClient(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	sc, err := getStorageClient(context.Background())
	if err != nil {
		t.Fatalf("应成功: %v", err)
	}
	if sc == nil {
		t.Fatal("StorageClient 不应为 nil")
	}
}

// TestGetCommonStorageClient_ReturnsClient 验证 GetCommonStorageClient 返回正确客户端。
func TestGetCommonStorageClient_ReturnsClient(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	sc, err := GetCommonStorageClient(context.Background())
	if err != nil {
		t.Fatalf("应成功: %v", err)
	}
	if sc == nil {
		t.Fatal("StorageClient 不应为 nil")
	}
}

// TestInitSMHTokenRefresher_FailsGracefully 验证 InitSMHTokenRefresher 在无 SMH 服务时优雅失败。
func TestInitSMHTokenRefresher_FailsGracefully(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	// 调用后应不 panic，token 刷新失败只记日志
	InitSMHTokenRefresher(context.Background(), model.GetSMHConfig(context.Background()))

	// DB 中的 token 应仍为空（因为 CreateToken 会失败）
	space, _ := model.GetSMHSpaceRecord(context.Background(), "common")
	if space.AdminToken != "" {
		t.Errorf("刷新失败时 AdminToken 应保持空，实际=%q", space.AdminToken)
	}
}

// ============================================================================
// DeleteSMHCommonDirectory 前置校验分支测试
// ============================================================================

// TestDeleteSMHCommonDirectory_NotConfigured 覆盖 "SMH 未配置" 分支。
func TestDeleteSMHCommonDirectory_NotConfigured(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()

	err := DeleteSMHCommonDirectory(context.Background(), "backups/ins-1")
	if err == nil {
		t.Error("未配置时应返回 error")
	}
}

// TestDeleteSMHCommonDirectory_TokenNotAvailable 覆盖 "token 不可用" 分支。
func TestDeleteSMHCommonDirectory_TokenNotAvailable(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)

	err := DeleteSMHCommonDirectory(context.Background(), "backups/ins-1")
	if err == nil {
		t.Error("token 不可用时应返回 error")
	}
}
