package task

import (
	"context"
	"os"
	"testing"

	"hatchery/model"
)

// --- handleDeleteMemSpace ---

// handleDeleteMemSpace 在 PoolID 为空时应直接返回 nil，不调用 SDK。
func TestHandleDeleteMemSpace_EmptyPoolID(t *testing.T) {
	setupMemoryProTestDB(t)

	plugin := &model.MemoryTDAIPlugin{
		InstanceID:  "inst-empty-pool",
		CurrentPlan: model.MemoryPlanPro,
		PoolID:      "",
	}
	err := handleDeleteMemSpace(context.Background(), plugin)
	if err != nil {
		t.Fatalf("empty PoolID should be noop, got err: %v", err)
	}
}

// handleDeleteMemSpace 在 SDK 初始化失败时应返回错误（不 panic）。
// 通过清空 SiteConfig 的 CVMSecretId / CVMSecretKey，触发 getCredential 失败。
func TestHandleDeleteMemSpace_SDKInitFailed(t *testing.T) {
	setupMemoryProTestDB(t)

	// 确保 MEMORY_API_SECRET_ID 未设置（走 getCredential 分支）
	origEnvID := os.Getenv("MEMORY_API_SECRET_ID")
	os.Unsetenv("MEMORY_API_SECRET_ID")
	defer func() {
		if origEnvID != "" {
			os.Setenv("MEMORY_API_SECRET_ID", origEnvID)
		}
	}()

	plugin := &model.MemoryTDAIPlugin{
		InstanceID:  "inst-sdk-fail",
		CurrentPlan: model.MemoryPlanPro,
		PoolID:      "space-xxx",
	}
	err := handleDeleteMemSpace(context.Background(), plugin)
	if err == nil {
		t.Fatal("SDK init failed should return error")
	}
}

// --- rollbackProMemSpace ---

// 插件行不存在 → 直接返回，不做任何操作。
func TestRollbackProMemSpace_PluginNotFound(t *testing.T) {
	setupMemoryProTestDB(t)
	// 不应 panic，也不应写库
	rollbackProMemSpace(context.Background(), "inst-nonexistent")
}

// pool_id 为空 → 直接返回，不做任何操作。
func TestRollbackProMemSpace_EmptyPoolID(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-empty",
		CurrentPlan: model.MemoryPlanOff,
		PoolID:      "",
	})
	rollbackProMemSpace(context.Background(), "inst-empty")
	// 确认 plugin 行没变
	plugin := model.GetMemoryTDAIPlugin(context.Background(), "inst-empty")
	if plugin.PoolID != "" {
		t.Errorf("pool_id should remain empty, got %q", plugin.PoolID)
	}
}

// current_plan 已是 PRO 说明切换实际成功 → 不应回滚。
func TestRollbackProMemSpace_AlreadyInPro(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "inst-already-pro",
		CurrentPlan:  model.MemoryPlanPro,
		PoolID:       "space-keep",
		DatabaseName: "db-keep",
		Endpoint:     "http://10.0.0.1:80",
	})
	rollbackProMemSpace(context.Background(), "inst-already-pro")
	// 绑定信息应保留
	plugin := model.GetMemoryTDAIPlugin(context.Background(), "inst-already-pro")
	if plugin.PoolID != "space-keep" {
		t.Errorf("pool_id should be kept, got %q", plugin.PoolID)
	}
	if plugin.Endpoint != "http://10.0.0.1:80" {
		t.Errorf("endpoint should be kept")
	}
}

// SDK 初始化失败 → 保留本地绑定信息（不清空），避免泄漏未清理的记录。
func TestRollbackProMemSpace_SDKInitFailed(t *testing.T) {
	setupMemoryProTestDB(t)
	origEnvID := os.Getenv("MEMORY_API_SECRET_ID")
	os.Unsetenv("MEMORY_API_SECRET_ID")
	defer func() {
		if origEnvID != "" {
			os.Setenv("MEMORY_API_SECRET_ID", origEnvID)
		}
	}()

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "inst-rb-sdk-fail",
		CurrentPlan:  model.MemoryPlanOff, // 未到 PRO，应触发回滚路径
		PoolID:       "space-fail",
		DatabaseName: "db-fail",
		Endpoint:     "http://10.0.0.2:80",
	})
	rollbackProMemSpace(context.Background(), "inst-rb-sdk-fail")

	// SDK 初始化失败 → 本地绑定信息应保留（等人工清理）
	plugin := model.GetMemoryTDAIPlugin(context.Background(), "inst-rb-sdk-fail")
	if plugin.PoolID != "space-fail" {
		t.Errorf("pool_id should be kept when SDK init fails, got %q", plugin.PoolID)
	}
}

// --- newAgentMemoryClient ---

// 无凭证配置 → 应返回错误（SDK 客户端初始化失败）。
func TestNewAgentMemoryClient_NoCredentials(t *testing.T) {
	setupMemoryProTestDB(t)
	origEnvID := os.Getenv("MEMORY_API_SECRET_ID")
	os.Unsetenv("MEMORY_API_SECRET_ID")
	defer func() {
		if origEnvID != "" {
			os.Setenv("MEMORY_API_SECRET_ID", origEnvID)
		}
	}()

	_, err := newAgentMemoryClientFn(context.Background())
	if err == nil {
		t.Fatal("expected error when no credentials configured")
	}
}

// --- handleSwitchToPro 更多边界 ---

// 已在 PRO 且有完整绑定 → 应走重发配置路径（isRedelivery），不应返回 NonRetryableError
// 但会在 checkInstanceRunning 处失败（CVM 未配置），返回可重试错误。
func TestHandleSwitchToPro_RedeliveryPath(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-redeliver", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "inst-redeliver",
		CurrentPlan:  model.MemoryPlanPro,
		PoolID:       "space-keep",
		Endpoint:     "http://10.0.0.1:80",
		DatabaseName: "db-keep",
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToPro, "switch:inst-redeliver", "inst-redeliver", "{}", "u", "")
	err := handleSwitchToPro(job)
	if err == nil {
		t.Fatal("should fail due to CVM not ready, but not NonRetryable (redelivery path)")
	}
	// 不应是 NonRetryableError（redelivery 走到 wait_instance_ready 失败）
	if _, ok := err.(*NonRetryableError); ok {
		t.Errorf("should not be NonRetryableError: %v", err)
	}
}

// checkInstanceSupportsMemoryTask 不支持的 agent_type → 应返回 NonRetryableError
func TestHandleSwitchToPro_UnsupportedAgentType(t *testing.T) {
	setupMemoryProTestDB(t)

	// lightclawace 类型：不支持记忆
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-ace-pro", AgentType: "lightclawace"})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-ace-pro",
		CurrentPlan: model.MemoryPlanOff,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToPro, "switch:inst-ace-pro", "inst-ace-pro", "{}", "u", "")
	err := handleSwitchToPro(job)
	if err == nil {
		t.Fatal("should fail for unsupported agent_type")
	}
	if _, ok := err.(*NonRetryableError); !ok {
		t.Errorf("expected NonRetryableError, got %T: %v", err, err)
	}
}

// --- handleSwitchToFree / handleSwitchToOff 已终态跳过路径 ---

// 实例已在 FREE → handleSwitchToFree 应幂等跳过，不报错
func TestHandleSwitchToFree_AlreadyFreeSkip(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-already-free", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "inst-already-free",
		CurrentPlan:  model.MemoryPlanFree,
		SwitchStatus: model.MemorySwitchStatusSwitchingToFree,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-already-free", "inst-already-free", "{}", "u", "")
	err := handleSwitchToFree(job)
	if err != nil {
		t.Fatalf("已在 FREE 应幂等跳过，got err: %v", err)
	}
	// switch_status 应被清空
	plugin := model.GetMemoryTDAIPlugin(context.Background(), "inst-already-free")
	if plugin.SwitchStatus != model.MemorySwitchStatusNone {
		t.Errorf("switch_status should be cleared, got %q", plugin.SwitchStatus)
	}
}

// 实例已在 PRO → handleSwitchToFree 应返回 NonRetryableError
// （和 tdai_dispatcher_test.go 里的轻量版区别：这里用 setupMemoryProTestDB 带 Instance 表）
func TestHandleSwitchToFree_FromProBlocked_FullSetup(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-pro-to-free", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-pro-to-free",
		CurrentPlan: model.MemoryPlanPro,
		PoolID:      "space-xxx",
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-pro-to-free", "inst-pro-to-free", "{}", "u", "")
	err := handleSwitchToFree(job)
	if err == nil {
		t.Fatal("Pro → Free 应被拦截")
	}
	if _, ok := err.(*NonRetryableError); !ok {
		t.Errorf("expected NonRetryableError, got %T: %v", err, err)
	}
}

// 实例已在 OFF → handleSwitchToOff 应幂等跳过，不报错
func TestHandleSwitchToOff_AlreadyOffSkip(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-already-off", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "inst-already-off",
		CurrentPlan:  model.MemoryPlanOff,
		SwitchStatus: model.MemorySwitchStatusSwitchingToOff,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-already-off", "inst-already-off", "{}", "u", "")
	err := handleSwitchToOff(job)
	if err != nil {
		t.Fatalf("已在 OFF 应幂等跳过，got err: %v", err)
	}
	plugin := model.GetMemoryTDAIPlugin(context.Background(), "inst-already-off")
	if plugin.SwitchStatus != model.MemorySwitchStatusNone {
		t.Errorf("switch_status should be cleared, got %q", plugin.SwitchStatus)
	}
}

// 任务执行器保险：handleSwitchToOff agent_type 不支持 → NonRetryableError
func TestHandleSwitchToOff_UnsupportedAgentType(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-ace-off", AgentType: "lightclawace"})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-ace-off",
		CurrentPlan: model.MemoryPlanFree,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-ace-off", "inst-ace-off", "{}", "u", "")
	err := handleSwitchToOff(job)
	if err == nil {
		t.Fatal("unsupported agent_type should fail")
	}
	if _, ok := err.(*NonRetryableError); !ok {
		t.Errorf("expected NonRetryableError, got %T: %v", err, err)
	}
}
