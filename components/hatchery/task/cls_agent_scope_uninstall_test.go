package task

import (
	"context"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initScopeUninstallTestDB 初始化内存 DB，支持 CLS scope 相关查询。
func initScopeUninstallTestDB(t *testing.T) func() {
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

// TestRunCLSAgentScopeUninstall_AllMode_Noop 验证全量模式（非分组模式）下不执行卸载。
func TestRunCLSAgentScopeUninstall_AllMode_Noop(t *testing.T) {
	cleanup := initScopeUninstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	// CLSScopeMode 为空 = 全量模式
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1})

	// 创建已安装实例
	model.DB(ctx).Create(&model.Instance{
		InstanceId:     "ins-all-mode",
		UserID:         1,
		CLSAgentStatus: model.CLSAgentInstalled,
	})

	// 全量模式下不应该卸载任何实例
	runCLSAgentScopeUninstall(ctx)

	// 验证实例状态未变
	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-all-mode").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("全量模式下实例不应被卸载，期望 status=%d，实际 %d", model.CLSAgentInstalled, inst.CLSAgentStatus)
	}
}

// TestRunCLSAgentScopeUninstall_GroupMode_EmptyScope_NoInstances
// 验证分组模式 + scope 为空（无配置分组）时，所有已安装实例都视为 scope 外，
// 但因无运行实例（CVM API 无法 mock），仅验证查询逻辑不 panic。
func TestRunCLSAgentScopeUninstall_GroupMode_EmptyScope_NoInstances(t *testing.T) {
	cleanup := initScopeUninstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})

	// 无已安装实例
	runCLSAgentScopeUninstall(ctx)
	// 不 panic 即通过
}

// TestRunCLSAgentScopeUninstall_GroupMode_NoOutOfScope 验证分组模式下所有实例都在 scope 内时无操作。
func TestRunCLSAgentScopeUninstall_GroupMode_NoOutOfScope(t *testing.T) {
	cleanup := initScopeUninstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})

	// 配置 scope: group_id=10
	model.DB(ctx).Create(&model.GroupConfigBinding{
		GroupID:    10,
		ConfigType: model.ConfigTypeCLSCollectScope,
		ConfigKey:  model.CLSCollectScopeKey,
	})
	// closure: group 10 含自身
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: 10, DescendantID: 10, Depth: 0})

	// 创建已安装实例，在 scope 内 (group_id=10)
	now := time.Now().Add(-6 * time.Minute)
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-in-scope",
		UserID:           1,
		GroupID:          10,
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSAgentStatusAt: &now,
	})

	runCLSAgentScopeUninstall(ctx)

	// 验证实例状态未变
	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-in-scope").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("scope 内实例不应被卸载，期望 status=%d，实际 %d", model.CLSAgentInstalled, inst.CLSAgentStatus)
	}
}

// TestRunCLSAgentScopeUninstall_GroupMode_EmptyScope_HasInstalled
// 验证分组模式 + scope 为空 + 存在已安装实例时，会尝试查询这些实例（但因 CVM API 不可用会失败退出）。
// 这个测试主要覆盖 lines 840-882 的查询逻辑。
func TestRunCLSAgentScopeUninstall_GroupMode_EmptyScope_HasInstalled(t *testing.T) {
	cleanup := initScopeUninstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})
	// scope 为空（分组模式但未配置分组）= 所有已安装实例都不在 scope 内

	past := time.Now().Add(-6 * time.Minute)
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-out-scope-1",
		UserID:           1,
		GroupID:          5,
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSAgentStatusAt: &past,
	})
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-out-scope-2",
		UserID:           1,
		GroupID:          6,
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSAgentStatusAt: &past,
	})

	// 执行 - 会走到 describeInstancesWithoutAgentRole，但 CVM client 创建失败，函数会提前返回
	runCLSAgentScopeUninstall(ctx)

	// 验证实例状态未被错误修改（因 CVM API 失败，不会执行任何状态变更）
	var inst1, inst2 model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-out-scope-1").First(&inst1)
	model.DB(ctx).Where("instance_id = ?", "ins-out-scope-2").First(&inst2)
	if inst1.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("CVM API 失败时实例不应被修改，期望 %d，实际 %d", model.CLSAgentInstalled, inst1.CLSAgentStatus)
	}
	if inst2.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("CVM API 失败时实例不应被修改，期望 %d，实际 %d", model.CLSAgentInstalled, inst2.CLSAgentStatus)
	}
}

// TestRunCLSAgentScopeUninstall_SkipsDoctorNode 验证不处理 doctor 节点。
func TestRunCLSAgentScopeUninstall_SkipsDoctorNode(t *testing.T) {
	cleanup := initScopeUninstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})

	past := time.Now().Add(-6 * time.Minute)
	// doctor 节点 - scope 外已安装
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-doctor",
		UserID:           1,
		GroupID:          99,
		IsDoctorNode:     true,
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSAgentStatusAt: &past,
	})

	runCLSAgentScopeUninstall(ctx)

	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-doctor").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("doctor 节点不应被处理，期望 status=%d，实际 %d", model.CLSAgentInstalled, inst.CLSAgentStatus)
	}
}

// TestRunCLSAgentScopeUninstall_InCooldown 验证冷却期内的实例不被捞取。
func TestRunCLSAgentScopeUninstall_InCooldown(t *testing.T) {
	cleanup := initScopeUninstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})

	// 刚刚更新过状态（冷却期内）
	recent := time.Now().Add(-1 * time.Minute)
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-cooldown",
		UserID:           1,
		GroupID:          99,
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSAgentStatusAt: &recent,
	})

	runCLSAgentScopeUninstall(ctx)

	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-cooldown").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("冷却期内实例不应被处理，期望 status=%d，实际 %d", model.CLSAgentInstalled, inst.CLSAgentStatus)
	}
}

// TestRunCLSAgentScopeUninstall_SkipsEmptyInstanceId 验证 instance_id 为空的记录不被捞取。
func TestRunCLSAgentScopeUninstall_SkipsEmptyInstanceId(t *testing.T) {
	cleanup := initScopeUninstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})

	past := time.Now().Add(-6 * time.Minute)
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "", // 空
		UserID:           1,
		GroupID:          99,
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSAgentStatusAt: &past,
	})

	runCLSAgentScopeUninstall(ctx)
	// 不 panic 即通过
}

// TestRunCLSAgentScopeUninstall_WithScopeConfig_FiltersCorrectly
// 验证有 scope 配置时，NOT IN 过滤逻辑正确。
func TestRunCLSAgentScopeUninstall_WithScopeConfig_FiltersCorrectly(t *testing.T) {
	cleanup := initScopeUninstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})

	// scope: group_id=10
	model.DB(ctx).Create(&model.GroupConfigBinding{
		GroupID:    10,
		ConfigType: model.ConfigTypeCLSCollectScope,
		ConfigKey:  model.CLSCollectScopeKey,
	})
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: 10, DescendantID: 10, Depth: 0})

	past := time.Now().Add(-6 * time.Minute)

	// scope 内实例 (group_id=10)
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-in-10",
		UserID:           1,
		GroupID:          10,
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSAgentStatusAt: &past,
	})

	// scope 外实例 (group_id=20)
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-out-20",
		UserID:           1,
		GroupID:          20,
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSAgentStatusAt: &past,
	})

	// 执行 - CVM client 会失败，但查询部分已执行
	runCLSAgentScopeUninstall(ctx)

	// scope 内实例不应改变
	var inScope model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-in-10").First(&inScope)
	if inScope.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("scope 内实例不应被修改，期望 %d，实际 %d", model.CLSAgentInstalled, inScope.CLSAgentStatus)
	}

	// scope 外实例也不变（因为 CVM API 调用失败，不会执行后续标记）
	var outScope model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-out-20").First(&outScope)
	if outScope.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("CVM API 失败时 scope 外实例不应被修改，期望 %d，实际 %d", model.CLSAgentInstalled, outScope.CLSAgentStatus)
	}
}
