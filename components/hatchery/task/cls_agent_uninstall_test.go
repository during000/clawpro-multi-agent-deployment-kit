package task

import (
	"context"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initUninstallTestDB 初始化 CLS uninstall 测试的内存 DB。
func initUninstallTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Instance{},
		&model.SiteConfig{},
		&model.GroupConfigBinding{},
		&model.GroupClosure{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	restore := model.UseDBForTest(db)
	return func() { restore() }
}

// TestRunCLSAgentUninstall_ClearsConfigAndScope 验证 runCLSAgentUninstall 在 CLSEnabled=1 时
// 会更新配置为 cls_enabled=0 并清空采集范围。
func TestRunCLSAgentUninstall_ClearsConfigAndScope(t *testing.T) {
	cleanup := initUninstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})

	// 配置 scope
	model.DB(ctx).Create(&model.GroupConfigBinding{
		GroupID:    10,
		ConfigType: model.ConfigTypeCLSCollectScope,
		ConfigKey:  model.CLSCollectScopeKey,
	})

	// 无已安装实例 → 执行完 config 更新后直接跳过
	runCLSAgentUninstall(ctx)

	// 验证配置已更新
	config := model.GetSiteConfig(ctx)
	if config.CLSEnabled != 0 {
		t.Errorf("期望 CLSEnabled=0，实际 %d", config.CLSEnabled)
	}
	if config.CLSScopeMode != "all" {
		t.Errorf("期望 CLSScopeMode=all，实际 %s", config.CLSScopeMode)
	}

	// 验证 scope 已清空
	ids, err := model.GetCLSCollectScopeGroupIDs(ctx)
	if err != nil {
		t.Fatalf("查询 scope 失败: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("期望 scope 为空，实际 %v", ids)
	}
}

// TestRunCLSAgentUninstall_NoPendingInstances 无已安装实例时跳过。
func TestRunCLSAgentUninstall_NoPendingInstances(t *testing.T) {
	cleanup := initUninstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 0})

	// 所有实例状态为未安装
	model.DB(ctx).Create(&model.Instance{
		InstanceId:     "ins-not-installed",
		UserID:         1,
		CLSAgentStatus: model.CLSAgentNotInstalled,
	})

	runCLSAgentUninstall(ctx)
	// 不 panic 即通过
}

// TestRunCLSAgentUninstall_HasPendingInstances_CVMFails
// 有待卸载实例但 CVM API 不可用时安全退出。
func TestRunCLSAgentUninstall_HasPendingInstances_CVMFails(t *testing.T) {
	cleanup := initUninstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 0})

	past := time.Now().Add(-6 * time.Minute)
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-to-uninstall",
		UserID:           1,
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSAgentStatusAt: &past,
	})

	// CVM API 会失败，但不应 panic
	runCLSAgentUninstall(ctx)

	// 实例状态不应被修改（CVM API 失败了）
	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-to-uninstall").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("CVM API 失败时实例不应被修改，期望 %d，实际 %d", model.CLSAgentInstalled, inst.CLSAgentStatus)
	}
}

// TestRunCLSAgentUninstall_SkipsDoctorNode doctor 节点不参与卸载。
func TestRunCLSAgentUninstall_SkipsDoctorNode(t *testing.T) {
	cleanup := initUninstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 0})

	past := time.Now().Add(-6 * time.Minute)
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-doctor-installed",
		UserID:           1,
		IsDoctorNode:     true,
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSAgentStatusAt: &past,
	})

	runCLSAgentUninstall(ctx)

	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-doctor-installed").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("doctor 节点不应被卸载，期望 %d，实际 %d", model.CLSAgentInstalled, inst.CLSAgentStatus)
	}
}

// TestRunCLSAgentUninstall_CooldownFilter 冷却期内实例不被捞取。
func TestRunCLSAgentUninstall_CooldownFilter(t *testing.T) {
	cleanup := initUninstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 0})

	recent := time.Now().Add(-1 * time.Minute)
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-cooldown-installed",
		UserID:           1,
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSAgentStatusAt: &recent,
	})

	runCLSAgentUninstall(ctx)

	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-cooldown-installed").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("冷却期内实例不应被处理，期望 %d，实际 %d", model.CLSAgentInstalled, inst.CLSAgentStatus)
	}
}
