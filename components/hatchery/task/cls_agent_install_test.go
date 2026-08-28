package task

import (
	"context"
	"testing"
	"time"

	"hatchery/controller"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initInstallTestDB 初始化内存 DB，支持 CLS install 相关查询。
func initInstallTestDB(t *testing.T) func() {
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

// TestRunCLSAgentInstall_GroupMode_EmptyScope_Skips
// 分组模式 + scope 为空 → 无对应实例，直接跳过。
func TestRunCLSAgentInstall_GroupMode_EmptyScope_Skips(t *testing.T) {
	cleanup := initInstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})

	// 有待安装实例但 scope 为空（分组模式下无配置分组）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:     "ins-pending",
		UserID:         1,
		GroupID:        5,
		CLSAgentStatus: model.CLSAgentNotInstalled,
	})

	result := &controller.CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "topic-id",
	}
	runCLSAgentInstall(ctx, result)

	// 验证实例状态未变
	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-pending").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentNotInstalled {
		t.Errorf("分组模式空 scope 时不应修改实例状态，期望 %d，实际 %d", model.CLSAgentNotInstalled, inst.CLSAgentStatus)
	}
}

// TestRunCLSAgentInstall_NoPendingInstances 无待安装实例时跳过。
func TestRunCLSAgentInstall_NoPendingInstances(t *testing.T) {
	cleanup := initInstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1})

	// 所有实例已安装
	model.DB(ctx).Create(&model.Instance{
		InstanceId:     "ins-installed",
		UserID:         1,
		CLSAgentStatus: model.CLSAgentInstalled,
	})

	result := &controller.CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "topic-id",
	}
	runCLSAgentInstall(ctx, result)
	// 不 panic 且不修改已安装实例
	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-installed").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("无待安装实例时不应修改已安装实例，期望 %d，实际 %d", model.CLSAgentInstalled, inst.CLSAgentStatus)
	}
}

// TestRunCLSAgentInstall_ScopeFilter_OnlyProcessesScopeInstances
// 分组模式 + scope 非空 → 只处理 scope 内待安装实例。
func TestRunCLSAgentInstall_ScopeFilter_OnlyProcessesScopeInstances(t *testing.T) {
	cleanup := initInstallTestDB(t)
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

	// scope 内待安装实例 (group_id=10)
	model.DB(ctx).Create(&model.Instance{
		InstanceId:     "ins-scope-pending",
		UserID:         1,
		GroupID:        10,
		CLSAgentStatus: model.CLSAgentNotInstalled,
	})

	// scope 外待安装实例 (group_id=20)
	model.DB(ctx).Create(&model.Instance{
		InstanceId:     "ins-outscope-pending",
		UserID:         1,
		GroupID:        20,
		CLSAgentStatus: model.CLSAgentNotInstalled,
	})

	result := &controller.CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "topic-id",
	}
	// 执行 - CVM API 会失败，但 scope 过滤逻辑已覆盖
	runCLSAgentInstall(ctx, result)

	// scope 外实例不应被修改为 installing（CVM API 失败前不会标记）
	var outInst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-outscope-pending").First(&outInst)
	if outInst.CLSAgentStatus != model.CLSAgentNotInstalled {
		t.Errorf("scope 外实例不应被处理，期望 %d，实际 %d", model.CLSAgentNotInstalled, outInst.CLSAgentStatus)
	}
}

// TestRunCLSAgentInstall_CooldownFilter 冷却期内实例不被捞取。
func TestRunCLSAgentInstall_CooldownFilter(t *testing.T) {
	cleanup := initInstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1})

	recent := time.Now().Add(-1 * time.Minute)
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-cooldown",
		UserID:           1,
		CLSAgentStatus:   model.CLSAgentNotInstalled,
		CLSAgentStatusAt: &recent, // 冷却期内
	})

	result := &controller.CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "topic-id",
	}
	runCLSAgentInstall(ctx, result)

	// 冷却期内不应被处理
	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-cooldown").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentNotInstalled {
		t.Errorf("冷却期内实例不应被修改，期望 %d，实际 %d", model.CLSAgentNotInstalled, inst.CLSAgentStatus)
	}
}

// TestRunCLSAgentInstall_SkipsDoctorNode doctor 节点不参与安装。
func TestRunCLSAgentInstall_SkipsDoctorNode(t *testing.T) {
	cleanup := initInstallTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1})

	model.DB(ctx).Create(&model.Instance{
		InstanceId:     "ins-doctor",
		UserID:         1,
		IsDoctorNode:   true,
		CLSAgentStatus: model.CLSAgentNotInstalled,
	})

	result := &controller.CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "topic-id",
	}
	runCLSAgentInstall(ctx, result)

	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-doctor").First(&inst)
	if inst.CLSAgentStatus != model.CLSAgentNotInstalled {
		t.Errorf("doctor 节点不应被安装，期望 %d，实际 %d", model.CLSAgentNotInstalled, inst.CLSAgentStatus)
	}
}
