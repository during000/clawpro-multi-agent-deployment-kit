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

// initCLSAgentTestDB 为 cls_agent_task 测试搭建独立的内存 DB，
// 仅 AutoMigrate Instance 表（这些查询只读 instances 一张表）。
func initCLSAgentTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Instance{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	restore := model.UseDBForTest(db)
	return func() { restore() }
}

// TestResetStaleCLSAgentStatus_SkipsDoctorNode 验证 resetStaleCLSAgentStatus
// 不会回退龙虾医生节点（is_doctor_node=true）卡在"安装中/卸载中"的状态。
//
// 背景：龙虾医生节点是临时诊断节点（生命周期 30~60 分钟），不参与 CLS Agent
// 安装/卸载流程。cls_agent_task 的所有查询都加了 `is_doctor_node = false`
// 过滤，避免与 ActivateDoctorSession 内部的 set_model.sh 抢 openclaw.json 文件锁。
func TestResetStaleCLSAgentStatus_SkipsDoctorNode(t *testing.T) {
	cleanup := initCLSAgentTestDB(t)
	defer cleanup()

	ctx := context.Background()
	staleAt := time.Now().Add(-15 * time.Minute) // 早于 10 分钟阈值

	// 普通节点：is_doctor_node=false，状态为"安装中"且超时 → 应被回退
	normalInstalling := model.Instance{
		Name:           "normal-installing",
		InstanceId:     "ins-normal-installing",
		UserID:         1,
		IsDoctorNode:   false,
		CLSAgentStatus: model.CLSAgentInstalling,
	}
	if err := model.DB(ctx).Create(&normalInstalling).Error; err != nil {
		t.Fatalf("create normal installing: %v", err)
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", normalInstalling.ID).
		Update("cls_agent_status_at", staleAt).Error; err != nil {
		t.Fatalf("set stale_at: %v", err)
	}

	// 龙虾医生节点：is_doctor_node=true，状态同样为"安装中"且超时 → 不应被回退
	doctorInstalling := model.Instance{
		Name:           "doctor-installing",
		InstanceId:     "ins-doctor-installing",
		UserID:         1,
		IsDoctorNode:   true,
		CLSAgentStatus: model.CLSAgentInstalling,
	}
	if err := model.DB(ctx).Create(&doctorInstalling).Error; err != nil {
		t.Fatalf("create doctor installing: %v", err)
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", doctorInstalling.ID).
		Update("cls_agent_status_at", staleAt).Error; err != nil {
		t.Fatalf("set stale_at: %v", err)
	}

	// 同样配置一对"卸载中"实例
	normalUninstalling := model.Instance{
		Name:           "normal-uninstalling",
		InstanceId:     "ins-normal-uninstalling",
		UserID:         1,
		IsDoctorNode:   false,
		CLSAgentStatus: model.CLSAgentUninstalling,
	}
	if err := model.DB(ctx).Create(&normalUninstalling).Error; err != nil {
		t.Fatalf("create normal uninstalling: %v", err)
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", normalUninstalling.ID).
		Update("cls_agent_status_at", staleAt).Error; err != nil {
		t.Fatalf("set stale_at: %v", err)
	}

	doctorUninstalling := model.Instance{
		Name:           "doctor-uninstalling",
		InstanceId:     "ins-doctor-uninstalling",
		UserID:         1,
		IsDoctorNode:   true,
		CLSAgentStatus: model.CLSAgentUninstalling,
	}
	if err := model.DB(ctx).Create(&doctorUninstalling).Error; err != nil {
		t.Fatalf("create doctor uninstalling: %v", err)
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", doctorUninstalling.ID).
		Update("cls_agent_status_at", staleAt).Error; err != nil {
		t.Fatalf("set stale_at: %v", err)
	}

	// 执行
	resetStaleCLSAgentStatus(ctx)

	// 验证
	cases := []struct {
		name        string
		instanceID  uint
		wantStatus  int
		description string
	}{
		{
			name:        "normal_installing_should_reset",
			instanceID:  normalInstalling.ID,
			wantStatus:  model.CLSAgentNotInstalled,
			description: "普通节点超时安装中应回退到未安装",
		},
		{
			name:        "doctor_installing_should_NOT_reset",
			instanceID:  doctorInstalling.ID,
			wantStatus:  model.CLSAgentInstalling,
			description: "龙虾医生节点不应被回退（保持安装中状态）",
		},
		{
			name:        "normal_uninstalling_should_reset",
			instanceID:  normalUninstalling.ID,
			wantStatus:  model.CLSAgentInstalled,
			description: "普通节点超时卸载中应回退到已安装",
		},
		{
			name:        "doctor_uninstalling_should_NOT_reset",
			instanceID:  doctorUninstalling.ID,
			wantStatus:  model.CLSAgentUninstalling,
			description: "龙虾医生节点不应被回退（保持卸载中状态）",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got model.Instance
			if err := model.DB(ctx).First(&got, tc.instanceID).Error; err != nil {
				t.Fatalf("query instance %d: %v", tc.instanceID, err)
			}
			if got.CLSAgentStatus != tc.wantStatus {
				t.Errorf("%s: got cls_agent_status=%d, want %d",
					tc.description, got.CLSAgentStatus, tc.wantStatus)
			}
		})
	}
}

// ─── getCLSInterval ───────────────────────────────────────────────

// TestGetCLSInterval_Default 验证未设置环境变量时返回默认值 60。
func TestGetCLSInterval_Default(t *testing.T) {
	t.Setenv("CLS_TASK_INTERVAL", "")
	if got := getCLSInterval(); got != 60 {
		t.Errorf("expected 60, got %d", got)
	}
}

// TestGetCLSInterval_ValidEnv 验证设置有效环境变量时返回对应值。
func TestGetCLSInterval_ValidEnv(t *testing.T) {
	t.Setenv("CLS_TASK_INTERVAL", "30")
	if got := getCLSInterval(); got != 30 {
		t.Errorf("expected 30, got %d", got)
	}
}

// TestGetCLSInterval_InvalidEnv 验证设置无效环境变量时回退到默认值 60。
func TestGetCLSInterval_InvalidEnv(t *testing.T) {
	t.Setenv("CLS_TASK_INTERVAL", "abc")
	if got := getCLSInterval(); got != 60 {
		t.Errorf("expected 60, got %d", got)
	}
}

// TestGetCLSInterval_ZeroEnv 验证设置为 0 时回退到默认值 60（n > 0 校验）。
func TestGetCLSInterval_ZeroEnv(t *testing.T) {
	t.Setenv("CLS_TASK_INTERVAL", "0")
	if got := getCLSInterval(); got != 60 {
		t.Errorf("expected 60, got %d", got)
	}
}

// ─── getCLSBatchLimit ─────────────────────────────────────────────

// TestGetCLSBatchLimit_Default 验证未设置环境变量时返回默认值 50。
func TestGetCLSBatchLimit_Default(t *testing.T) {
	t.Setenv("CLS_BATCH_LIMIT", "")
	if got := getCLSBatchLimit(); got != defaultBatchLimit {
		t.Errorf("expected %d, got %d", defaultBatchLimit, got)
	}
}

// TestGetCLSBatchLimit_ValidEnv 验证设置有效环境变量时返回对应值。
func TestGetCLSBatchLimit_ValidEnv(t *testing.T) {
	t.Setenv("CLS_BATCH_LIMIT", "20")
	if got := getCLSBatchLimit(); got != 20 {
		t.Errorf("expected 20, got %d", got)
	}
}

// TestGetCLSBatchLimit_InvalidEnv 验证设置无效环境变量时回退到默认值。
func TestGetCLSBatchLimit_InvalidEnv(t *testing.T) {
	t.Setenv("CLS_BATCH_LIMIT", "xyz")
	if got := getCLSBatchLimit(); got != defaultBatchLimit {
		t.Errorf("expected %d, got %d", defaultBatchLimit, got)
	}
}

// ─── isCLSUninstallEnabled ────────────────────────────────────────

// TestIsCLSUninstallEnabled_Default 验证未设置环境变量时返回 false。
func TestIsCLSUninstallEnabled_Default(t *testing.T) {
	t.Setenv("ENABLE_CLS_UNINSTALL", "")
	if isCLSUninstallEnabled() {
		t.Error("expected false when env is empty")
	}
}

// TestIsCLSUninstallEnabled_True 验证设置为 "true" 时返回 true。
func TestIsCLSUninstallEnabled_True(t *testing.T) {
	for _, v := range []string{"true", "1", "on", "TRUE", "ON"} {
		t.Run(v, func(t *testing.T) {
			t.Setenv("ENABLE_CLS_UNINSTALL", v)
			if !isCLSUninstallEnabled() {
				t.Errorf("expected true for %q", v)
			}
		})
	}
}

// TestIsCLSUninstallEnabled_False 验证设置为 "false" 等值时返回 false。
func TestIsCLSUninstallEnabled_False(t *testing.T) {
	for _, v := range []string{"false", "0", "off", "no"} {
		t.Run(v, func(t *testing.T) {
			t.Setenv("ENABLE_CLS_UNINSTALL", v)
			if isCLSUninstallEnabled() {
				t.Errorf("expected false for %q", v)
			}
		})
	}
}

// ─── resetStaleCLSAgentStatus（updating 分支） ────────────────────

// TestResetStaleCLSAgentStatus_UpdatingVersion 验证超时的 cls_plugin_version='updating'
// 实例会被回退为 '1.0'，而未超时的实例不受影响。
func TestResetStaleCLSAgentStatus_UpdatingVersion(t *testing.T) {
	cleanup := initCLSAgentTestDB(t)
	defer cleanup()

	ctx := context.Background()
	staleAt := time.Now().Add(-15 * time.Minute) // 早于 10 分钟阈值
	freshAt := time.Now().Add(-1 * time.Minute)  // 未超时

	// 超时的 updating 实例 → 应被回退为 1.0
	staleUpdating := model.Instance{
		Name:             "stale-updating",
		InstanceId:       "ins-stale-updating",
		UserID:           1,
		IsDoctorNode:     false,
		CLSPluginVersion: "updating",
	}
	if err := model.DB(ctx).Create(&staleUpdating).Error; err != nil {
		t.Fatalf("create stale updating: %v", err)
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", staleUpdating.ID).
		Update("cls_agent_status_at", staleAt).Error; err != nil {
		t.Fatalf("set stale_at: %v", err)
	}

	// 未超时的 updating 实例 → 不应被回退
	freshUpdating := model.Instance{
		Name:             "fresh-updating",
		InstanceId:       "ins-fresh-updating",
		UserID:           1,
		IsDoctorNode:     false,
		CLSPluginVersion: "updating",
	}
	if err := model.DB(ctx).Create(&freshUpdating).Error; err != nil {
		t.Fatalf("create fresh updating: %v", err)
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", freshUpdating.ID).
		Update("cls_agent_status_at", freshAt).Error; err != nil {
		t.Fatalf("set fresh_at: %v", err)
	}

	// 龙虾医生节点的 updating 实例 → 不应被回退
	doctorUpdating := model.Instance{
		Name:             "doctor-updating",
		InstanceId:       "ins-doctor-updating",
		UserID:           1,
		IsDoctorNode:     true,
		CLSPluginVersion: "updating",
	}
	if err := model.DB(ctx).Create(&doctorUpdating).Error; err != nil {
		t.Fatalf("create doctor updating: %v", err)
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", doctorUpdating.ID).
		Update("cls_agent_status_at", staleAt).Error; err != nil {
		t.Fatalf("set stale_at: %v", err)
	}

	resetStaleCLSAgentStatus(ctx)

	// 验证超时的普通节点被回退为 1.0
	var got model.Instance
	if err := model.DB(ctx).First(&got, staleUpdating.ID).Error; err != nil {
		t.Fatalf("query stale updating: %v", err)
	}
	if got.CLSPluginVersion != "1.0" {
		t.Errorf("stale updating should be reset to 1.0, got %q", got.CLSPluginVersion)
	}

	// 验证未超时的实例保持 updating
	got = model.Instance{}
	if err := model.DB(ctx).First(&got, freshUpdating.ID).Error; err != nil {
		t.Fatalf("query fresh updating: %v", err)
	}
	if got.CLSPluginVersion != "updating" {
		t.Errorf("fresh updating should remain 'updating', got %q", got.CLSPluginVersion)
	}

	// 验证龙虾医生节点不受影响
	got = model.Instance{}
	if err := model.DB(ctx).First(&got, doctorUpdating.ID).Error; err != nil {
		t.Fatalf("query doctor updating: %v", err)
	}
	if got.CLSPluginVersion != "updating" {
		t.Errorf("doctor updating should remain 'updating', got %q", got.CLSPluginVersion)
	}
}

// TestResetStaleCLSAgentStatus_NullStatusAt 验证 cls_agent_status_at 为 NULL 的中间状态
// 实例（历史遗留）也会被回退。
func TestResetStaleCLSAgentStatus_NullStatusAt(t *testing.T) {
	cleanup := initCLSAgentTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// cls_agent_status_at 为 NULL 的安装中实例 → 应被回退
	nullInstalling := model.Instance{
		Name:           "null-installing",
		InstanceId:     "ins-null-installing",
		UserID:         1,
		IsDoctorNode:   false,
		CLSAgentStatus: model.CLSAgentInstalling,
	}
	if err := model.DB(ctx).Create(&nullInstalling).Error; err != nil {
		t.Fatalf("create null installing: %v", err)
	}
	// 确保 cls_agent_status_at 为 NULL
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", nullInstalling.ID).
		Update("cls_agent_status_at", nil).Error; err != nil {
		t.Fatalf("set null status_at: %v", err)
	}

	resetStaleCLSAgentStatus(ctx)

	var got model.Instance
	if err := model.DB(ctx).First(&got, nullInstalling.ID).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if got.CLSAgentStatus != model.CLSAgentNotInstalled {
		t.Errorf("null status_at installing should be reset to not_installed, got %d", got.CLSAgentStatus)
	}
}

// ─── runCLSAgentInstall（纯 DB 分支） ─────────────────────────────

// initCLSAgentFullTestDB 为需要 SiteConfig 和 GroupConfigBinding 的测试搭建完整 DB。
func initCLSAgentFullTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Instance{},
		&model.SiteConfig{},
		&model.GroupConfigBinding{},
		&model.UserGroup{},
		&model.GroupClosure{},
		&model.UserGroupMember{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	restore := model.UseDBForTest(db)
	return func() { restore() }
}

// TestRunCLSAgentInstall_GroupModeNoInstances 验证分组模式下无对应实例时跳过安装。
func TestRunCLSAgentInstall_GroupModeNoInstances(t *testing.T) {
	cleanup := initCLSAgentFullTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 设置 CLSScopeMode=group
	if err := model.DB(ctx).Create(&model.SiteConfig{
		CLSScopeMode: "group",
		CLSEnabled:   0,
	}).Error; err != nil {
		t.Fatalf("create site config: %v", err)
	}

	// 有一个待安装实例，但 scope 为分组模式且无绑定分组
	if err := model.DB(ctx).Create(&model.Instance{
		Name:           "inst-1",
		InstanceId:     "ins-group-mode-1",
		UserID:         1,
		IsDoctorNode:   false,
		CLSAgentStatus: model.CLSAgentNotInstalled,
	}).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	result := &controller.CLSClawServiceResult{
		MetricTopicId: "topic-metric",
		TopicId:       "topic-id",
		TraceTopicId:  "topic-trace",
	}
	// 分组模式下无匹配实例，应跳过安装，不 panic
	runCLSAgentInstall(ctx, result)

	// 验证实例状态未变化（仍为未安装）
	var inst model.Instance
	if err := model.DB(ctx).Where("instance_id = ?", "ins-group-mode-1").First(&inst).Error; err != nil {
		t.Fatalf("query instance: %v", err)
	}
	if inst.CLSAgentStatus != model.CLSAgentNotInstalled {
		t.Errorf("expected not_installed, got %d", inst.CLSAgentStatus)
	}
}

// TestRunCLSAgentInstall_AllModeNoPendingAfterCooldown 验证全量模式下冷却期内的实例不被捞取。
func TestRunCLSAgentInstall_AllModeNoPendingAfterCooldown(t *testing.T) {
	cleanup := initCLSAgentFullTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 设置 CLSScopeMode=all
	if err := model.DB(ctx).Create(&model.SiteConfig{
		CLSScopeMode: "all",
		CLSEnabled:   0,
	}).Error; err != nil {
		t.Fatalf("create site config: %v", err)
	}

	// 创建一个在冷却期内的待安装实例（cls_agent_status_at 为 1 分钟前，未超过 5 分钟冷却）
	recentAt := time.Now().Add(-1 * time.Minute)
	inst := model.Instance{
		Name:           "inst-cooldown",
		InstanceId:     "ins-cooldown",
		UserID:         1,
		IsDoctorNode:   false,
		CLSAgentStatus: model.CLSAgentNotInstalled,
	}
	if err := model.DB(ctx).Create(&inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", inst.ID).
		Update("cls_agent_status_at", recentAt).Error; err != nil {
		t.Fatalf("set status_at: %v", err)
	}

	result := &controller.CLSClawServiceResult{
		MetricTopicId: "topic-metric",
		TopicId:       "topic-id",
		TraceTopicId:  "topic-trace",
	}
	// 冷却期内无待安装实例，应跳过
	runCLSAgentInstall(ctx, result)
}

// ─── runCLSAgentUninstall（纯 DB 分支） ───────────────────────────

// TestRunCLSAgentUninstall_UpdatesConfigWhenEnabled 验证 CLSEnabled=1 时会更新本地配置。
func TestRunCLSAgentUninstall_UpdatesConfigWhenEnabled(t *testing.T) {
	cleanup := initCLSAgentFullTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 创建 CLSEnabled=1 的配置
	if err := model.DB(ctx).Create(&model.SiteConfig{
		CLSEnabled:   1,
		CLSScopeMode: "all",
	}).Error; err != nil {
		t.Fatalf("create site config: %v", err)
	}

	// 没有待卸载实例，但配置需要被更新
	runCLSAgentUninstall(ctx)

	// 验证 CLSEnabled 已被更新为 0
	config := model.GetSiteConfig(ctx)
	if config.CLSEnabled != 0 {
		t.Errorf("expected CLSEnabled=0 after uninstall, got %d", config.CLSEnabled)
	}
}

// TestResetStaleCLSAgentStatus_StaleUninstalling 验证超时的"卸载中"实例会被回退为"已安装"。
func TestResetStaleCLSAgentStatus_StaleUninstalling(t *testing.T) {
	cleanup := initCLSAgentTestDB(t)
	defer cleanup()

	ctx := context.Background()
	staleAt := time.Now().Add(-15 * time.Minute) // 早于 10 分钟阈值

	// 超时的卸载中实例 → 应被回退为已安装
	staleUninstalling := model.Instance{
		Name:           "stale-uninstalling",
		InstanceId:     "ins-stale-uninstalling",
		UserID:         1,
		IsDoctorNode:   false,
		CLSAgentStatus: model.CLSAgentUninstalling,
	}
	if err := model.DB(ctx).Create(&staleUninstalling).Error; err != nil {
		t.Fatalf("create stale uninstalling: %v", err)
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", staleUninstalling.ID).
		Update("cls_agent_status_at", staleAt).Error; err != nil {
		t.Fatalf("set stale_at: %v", err)
	}

	// 未超时的卸载中实例 → 不应被回退
	freshAt := time.Now().Add(-1 * time.Minute)
	freshUninstalling := model.Instance{
		Name:           "fresh-uninstalling",
		InstanceId:     "ins-fresh-uninstalling",
		UserID:         1,
		IsDoctorNode:   false,
		CLSAgentStatus: model.CLSAgentUninstalling,
	}
	if err := model.DB(ctx).Create(&freshUninstalling).Error; err != nil {
		t.Fatalf("create fresh uninstalling: %v", err)
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", freshUninstalling.ID).
		Update("cls_agent_status_at", freshAt).Error; err != nil {
		t.Fatalf("set fresh_at: %v", err)
	}

	resetStaleCLSAgentStatus(ctx)

	// 验证超时的卸载中实例被回退为已安装
	var got model.Instance
	if err := model.DB(ctx).First(&got, staleUninstalling.ID).Error; err != nil {
		t.Fatalf("query stale uninstalling: %v", err)
	}
	if got.CLSAgentStatus != model.CLSAgentInstalled {
		t.Errorf("stale uninstalling should be reset to installed, got %d", got.CLSAgentStatus)
	}

	// 验证未超时的卸载中实例保持不变
	got = model.Instance{}
	if err := model.DB(ctx).First(&got, freshUninstalling.ID).Error; err != nil {
		t.Fatalf("query fresh uninstalling: %v", err)
	}
	if got.CLSAgentStatus != model.CLSAgentUninstalling {
		t.Errorf("fresh uninstalling should remain uninstalling, got %d", got.CLSAgentStatus)
	}
}

// TestRunCLSAgentInstall_DBClosed_UpdateSiteConfigFails 验证 DB 关闭时 UpdateSiteConfig 失败后安全退出。
func TestRunCLSAgentInstall_DBClosed_UpdateSiteConfigFails(t *testing.T) {
	cleanup := initCLSAgentFullTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 0, CLSScopeMode: "all"})

	// 关闭底层 DB 连接，使后续 DB 操作失败
	if err := model.CloseUnderlyingDBForTest(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	// UpdateSiteConfig 会失败，函数应安全退出，不 panic
	result := &controller.CLSClawServiceResult{
		MetricTopicId: "metric-1",
		TopicId:       "topic-1",
		TraceTopicId:  "trace-1",
	}
	runCLSAgentInstall(ctx, result)
}

// TestRunCLSAgentUninstall_DBClosed_GetSiteConfigFails 验证 DB 关闭时 GetSiteConfig 失败后安全退出。
func TestRunCLSAgentUninstall_DBClosed_GetSiteConfigFails(t *testing.T) {
	cleanup := initCLSAgentFullTestDB(t)
	defer cleanup()

	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 0})

	// 关闭底层 DB 连接，使后续 DB 操作失败
	if err := model.CloseUnderlyingDBForTest(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	// GetSiteConfig 会返回默认值（不会 panic），runCLSAgentUninstall 应安全退出
	runCLSAgentUninstall(ctx)
}

// TestRunCLSAgentUninstall_UpdateSiteConfigFails 验证 CLSEnabled=1 时 UpdateSiteConfig 失败后
// 仍能安全退出（行 216 的 error log 路径）。
func TestRunCLSAgentUninstall_UpdateSiteConfigFails(t *testing.T) {
	cleanup := initCLSAgentFullTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 创建 CLSEnabled=1 的配置
	if err := model.DB(ctx).Create(&model.SiteConfig{
		CLSEnabled:   1,
		CLSScopeMode: "all",
	}).Error; err != nil {
		t.Fatalf("create site config: %v", err)
	}

	// 关闭底层 DB 连接，使 UpdateSiteConfig 失败
	if err := model.CloseUnderlyingDBForTest(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	// UpdateSiteConfig 会失败，函数应安全退出，不 panic
	runCLSAgentUninstall(ctx)
}

// TestRunCLSAgentUninstall_EnvForceUninstall 验证 isCLSUninstallEnabled()=true 时
// 即使无已安装实例也执行卸载（行 243 路径）。
func TestRunCLSAgentUninstall_EnvForceUninstall(t *testing.T) {
	cleanup := initCLSAgentFullTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 设置 CLSEnabled=0，无已安装实例
	if err := model.DB(ctx).Create(&model.SiteConfig{
		CLSEnabled:   0,
		CLSScopeMode: "all",
	}).Error; err != nil {
		t.Fatalf("create site config: %v", err)
	}

	// 设置环境变量强制卸载
	t.Setenv("ENABLE_CLS_UNINSTALL", "true")

	// 无已安装实例，但环境变量强制开启，应触发卸载逻辑（不 panic）
	runCLSAgentUninstall(ctx)
}

// TestResetStaleCLSAgentStatus_UpdatingVersionDBClosed 验证 DB 关闭时重置超时 updating
// 版本失败后安全退出（行 875 路径）。
func TestResetStaleCLSAgentStatus_UpdatingVersionDBClosed(t *testing.T) {
	cleanup := initCLSAgentTestDB(t)
	defer cleanup()

	ctx := context.Background()
	staleAt := time.Now().Add(-15 * time.Minute)

	// 创建超时的 updating 实例
	staleUpdating := model.Instance{
		Name:             "stale-updating-db-closed",
		InstanceId:       "ins-stale-updating-db-closed",
		UserID:           1,
		IsDoctorNode:     false,
		CLSPluginVersion: "updating",
	}
	if err := model.DB(ctx).Create(&staleUpdating).Error; err != nil {
		t.Fatalf("create stale updating: %v", err)
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", staleUpdating.ID).
		Update("cls_agent_status_at", staleAt).Error; err != nil {
		t.Fatalf("set stale_at: %v", err)
	}

	// 关闭底层 DB 连接，使重置操作失败
	if err := model.CloseUnderlyingDBForTest(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	// DB 关闭后 resetStaleCLSAgentStatus 应安全退出，不 panic
	resetStaleCLSAgentStatus(ctx)
}
