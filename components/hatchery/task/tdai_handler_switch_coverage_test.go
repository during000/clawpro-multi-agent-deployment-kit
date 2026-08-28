package task

import (
	"context"
	"testing"

	"hatchery/model"
)

// --- handleSwitchToFree 成功路径（覆盖 61-67 行：plan update + log） ---

// OFF → FREE 成功路径：需要 checkInstanceRunning + doEnablePlugin 都通过。
// 因为无法 mock TAT，只测"已在 FREE 时的幂等跳过后 DB 状态写入"能覆盖 update 逻辑。
// 补充：让 handleSwitchToFree 走到 already-in-target 分支时，验证 DB 写入的字段完整性。
func TestHandleSwitchToFree_AlreadyFree_VerifyDBUpdates(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-free-db", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "inst-free-db",
		CurrentPlan:  model.MemoryPlanFree,
		SwitchStatus: model.MemorySwitchStatusSwitchingToFree,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-free-db", "inst-free-db", "{}", "u", "")
	if err := handleSwitchToFree(job); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	plugin := model.GetMemoryTDAIPlugin(context.Background(), "inst-free-db")
	if plugin.CurrentPlan != model.MemoryPlanFree {
		t.Errorf("current_plan = %q, want FREE", plugin.CurrentPlan)
	}
	if plugin.SwitchStatus != model.MemorySwitchStatusNone {
		t.Errorf("switch_status = %q, want empty", plugin.SwitchStatus)
	}
	if plugin.LastTaskID != job.ID {
		t.Errorf("last_task_id = %d, want %d", plugin.LastTaskID, job.ID)
	}
}

// --- handleSwitchToOff 成功路径（覆盖 152-158：plan update + log） ---

func TestHandleSwitchToOff_AlreadyOff_VerifyDBUpdates(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-off-db", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "inst-off-db",
		CurrentPlan:  model.MemoryPlanOff,
		SwitchStatus: model.MemorySwitchStatusSwitchingToOff,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-off-db", "inst-off-db", "{}", "u", "")
	if err := handleSwitchToOff(job); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	plugin := model.GetMemoryTDAIPlugin(context.Background(), "inst-off-db")
	if plugin.CurrentPlan != model.MemoryPlanOff {
		t.Errorf("current_plan = %q, want OFF", plugin.CurrentPlan)
	}
	if plugin.SwitchStatus != model.MemorySwitchStatusNone {
		t.Errorf("switch_status = %q, want empty", plugin.SwitchStatus)
	}
	if plugin.LastTaskID != job.ID {
		t.Errorf("last_task_id = %d, want %d", plugin.LastTaskID, job.ID)
	}
}

// --- handleSwitchToOff: FREE→OFF 走到 checkInstanceRunning 失败（覆盖 126 行前后） ---

func TestHandleSwitchToOff_FreeToOff_CVMNotReady(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-free-off-cvm", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-free-off-cvm",
		CurrentPlan: model.MemoryPlanFree,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-free-off-cvm", "inst-free-off-cvm", "{}", "u", "")
	err := handleSwitchToOff(job)
	// 应在 checkInstanceRunning 处失败（无 CVM 凭证）
	if err == nil {
		t.Fatal("expected error from checkInstanceRunning")
	}
}

// --- handleSwitchToPro: OFF → PRO 走到 SDK 初始化失败（覆盖 52-53） ---

func TestHandleSwitchToPro_SDKInitFailed(t *testing.T) {
	setupMemoryProTestDB(t)
	defer mockEnsureMemoryPlugin(t)()
	// 确保无凭证
	t.Setenv("MEMORY_API_SECRET_ID", "")

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-pro-sdk-fail", AgentType: model.AgentTypeOpenClaw, LastCVMState: "RUNNING", AgentReady: 1})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-pro-sdk-fail",
		CurrentPlan: model.MemoryPlanOff,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToPro, "switch:inst-pro-sdk-fail", "inst-pro-sdk-fail", "{}", "u", "")
	err := handleSwitchToPro(job)
	// 会在 wait_instance_ready 或 allocate_database 处失败
	if err == nil {
		t.Fatal("expected error")
	}
}

// runSwitchPro 的 embeddingModel 默认值分支（204 行）需要真实 TAT 环境，
// 无法在纯单测中覆盖，交由集成测试验证。
