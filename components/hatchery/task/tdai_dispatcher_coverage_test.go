package task

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"hatchery/model"
)

// --- runTdaiDispatcherEntry 测试 ---

func TestRunTdaiDispatcherEntry_Normal(t *testing.T) {
	setupTestDB(t)
	// 不应 panic
	runTdaiDispatcherEntry(context.Background())
}

// 环境变量禁用时，init() 不注册 → runTdaiDispatcherEntry 仍可直接调用
func TestRunTdaiDispatcherEntry_EnvDisable(t *testing.T) {
	os.Setenv("DISABLE_TDAI_TASK_ENGINE", "true")
	defer os.Unsetenv("DISABLE_TDAI_TASK_ENGINE")
	// 直接调用入口函数应不 panic
	setupTestDB(t)
	runTdaiDispatcherEntry(context.Background())
}

// --- dispatchPendingJobs ---

// 数据库无 PENDING 任务时，dispatch 应直接返回。
func TestDispatchPendingJobs_NoJobs(t *testing.T) {
	setupTestDB(t)
	dispatchPendingJobs(context.Background(), "test-host")
	// 不应 panic，不应报错
}

// PENDING 任务被成功抢占后执行（即使 handler 失败）。
func TestDispatchPendingJobs_PicksAndExecutes(t *testing.T) {
	setupTestDB(t)

	// 构造一个可执行的 job：SWITCH_TO_OFF，且已在 OFF 状态 → 直接 success
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-disp", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-disp",
		CurrentPlan: model.MemoryPlanOff,
	})
	job, err := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-disp", "inst-disp", "{}", "u", "")
	if err != nil {
		t.Fatalf("submit job: %v", err)
	}
	// 使 run_at 已到达
	model.DB(context.Background()).Model(job).Update("run_at", time.Now().Add(-time.Second))

	dispatchPendingJobs(context.Background(), "test-host")

	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	if updated.State != model.TdaiJobStateSucceeded {
		t.Errorf("job should succeed, got state %q, error: %s", updated.State, updated.LastError)
	}
	if updated.LeaseOwner != "test-host" {
		t.Errorf("lease_owner = %q, want test-host", updated.LeaseOwner)
	}
}

// 未来 run_at 的任务不应被捞起
func TestDispatchPendingJobs_FutureJobSkipped(t *testing.T) {
	setupTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-future", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-future",
		CurrentPlan: model.MemoryPlanOff,
	})
	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-future", "inst-future", "{}", "u", "")
	// 将 run_at 设置为未来
	model.DB(context.Background()).Model(job).Update("run_at", time.Now().Add(time.Hour))

	dispatchPendingJobs(context.Background(), "test-host")

	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	if updated.State != model.TdaiJobStatePending {
		t.Errorf("future job should stay PENDING, got %q", updated.State)
	}
	if updated.LeaseOwner == "test-host" {
		t.Error("lease_owner 不应被设置（未被抢占）")
	}
}

// --- executeJob ---

// job 不存在 → 应静默返回不 panic
func TestExecuteJob_JobNotFound(t *testing.T) {
	setupTestDB(t)
	executeJob(context.Background(), 999999)
	// 不应 panic
}

// SWITCH_TO_PRO 任务最终失败时应触发 rollbackProMemSpace
func TestMarkJobFailed_SwitchToProRollback(t *testing.T) {
	setupTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-pro-final-fail", AgentType: model.AgentTypeOpenClaw})
	plugin := &model.MemoryTDAIPlugin{
		InstanceID:   "inst-pro-final-fail",
		CurrentPlan:  model.MemoryPlanOff, // 未到 PRO，应触发 rollback
		PoolID:       "space-final-fail",
		DatabaseName: "db-final-fail",
		SwitchStatus: model.MemorySwitchStatusSwitchingToPro,
	}
	model.DB(context.Background()).Create(plugin)

	// SWITCH_TO_PRO job 达到 max attempts
	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToPro, "switch:inst-pro-final-fail", "inst-pro-final-fail", "{}", "u", "")
	model.DB(context.Background()).Model(job).Update("attempt", model.TdaiJobDefaultMaxAttempts)
	job.Attempt = model.TdaiJobDefaultMaxAttempts

	// 不设置 MEMORY_API_SECRET_ID → rollback 里的 SDK 初始化失败，但函数不应 panic
	os.Unsetenv("MEMORY_API_SECRET_ID")

	markJobFailed(job, errors.New("SWITCH_TO_PRO final failure"))

	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	if updated.State != model.TdaiJobStateFailed {
		t.Errorf("job should be FAILED, got %q", updated.State)
	}
	// switch_status 应该被回滚到 None
	var updatedPlugin model.MemoryTDAIPlugin
	model.DB(context.Background()).Where("instance_id = ?", "inst-pro-final-fail").First(&updatedPlugin)
	if updatedPlugin.SwitchStatus != model.MemorySwitchStatusNone {
		t.Errorf("switch_status should be rolled back, got %q", updatedPlugin.SwitchStatus)
	}
}

// 验证退避上限 180s：job.Attempt=5 时计算值应被封顶
func TestMarkJobFailed_BackoffCappedAt180s(t *testing.T) {
	setupTestDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "inst-backoff", CurrentPlan: model.MemoryPlanOff})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-backoff", "inst-backoff", "{}", "u", "")
	// attempt=5：理论退避 5 * 6^4 = 6480s，应被封顶到 180s
	model.DB(context.Background()).Model(job).Update("attempt", 5)
	job.Attempt = 5

	beforeMark := time.Now()
	markJobFailed(job, errors.New("tempfail"))

	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	if updated.State != model.TdaiJobStatePending {
		t.Fatalf("attempt 5 of 10 should retry, got %q", updated.State)
	}
	// next_run 距离现在应该约 180s（允许 10s 容差）
	diff := updated.RunAt.Sub(beforeMark)
	if diff > 190*time.Second {
		t.Errorf("backoff not capped: %v > 180s", diff)
	}
	if diff < 170*time.Second {
		t.Errorf("backoff too short: %v < 170s", diff)
	}
}
