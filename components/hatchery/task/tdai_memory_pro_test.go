package task

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupMemoryProTestDB 创建测试数据库（扩展版，包含 Instance 表）。
func setupMemoryProTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试 DB 失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	if err := db.AutoMigrate(
		&model.MemoryTDAIPlugin{},
		&model.TdaiJob{},
		&model.SiteConfig{},
		&model.Instance{},
	); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	db.Create(&model.SiteConfig{
		MemoryTDAIEnable:            false,
		MemoryTDAISupportedVersions: "[]",
	})
}

// --- NonRetryableError ---

func TestNonRetryableError_Message(t *testing.T) {
	err := NewNonRetryableError("test message")
	if err.Error() != "test message" {
		t.Errorf("got %q, want 'test message'", err.Error())
	}
}

func TestNonRetryableError_TypeAssertion(t *testing.T) {
	err := NewNonRetryableError("non-retryable")
	var nre *NonRetryableError
	if !errors.As(err, &nre) {
		t.Fatal("should be assertable to *NonRetryableError")
	}
}

// --- markJobFailed ---

func TestMarkJobFailed_RetryWithBackoff(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-retry-bf",
		CurrentPlan: model.MemoryPlanOff,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-retry-bf", "inst-retry-bf", "{}", "u", "")
	// 模拟第 1 次尝试
	model.DB(context.Background()).Model(job).Update("attempt", 1)
	job.Attempt = 1

	markJobFailed(job, errors.New("temporary error"))

	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	if updated.State != model.TdaiJobStatePending {
		t.Fatalf("attempt 1 of 3: should retry, got state %s", updated.State)
	}
	if updated.RunAt.Before(time.Now()) {
		t.Error("run_at should be in the future for backoff")
	}
}

func TestMarkJobFailed_MaxAttemptsExhausted(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-maxretry",
		CurrentPlan: model.MemoryPlanOff,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-maxretry", "inst-maxretry", "{}", "u", "")
	model.DB(context.Background()).Model(job).Update("attempt", model.TdaiJobDefaultMaxAttempts)
	job.Attempt = model.TdaiJobDefaultMaxAttempts

	markJobFailed(job, errors.New("persistent error"))

	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	if updated.State != model.TdaiJobStateFailed {
		t.Fatalf("max attempts: should be FAILED, got %s", updated.State)
	}
	if updated.FinishedAt == nil {
		t.Error("finished_at should be set")
	}
}

func TestMarkJobFailed_NonRetryable(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-nonretry",
		CurrentPlan: model.MemoryPlanOff,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-nonretry", "inst-nonretry", "{}", "u", "")
	model.DB(context.Background()).Model(job).Update("attempt", 1)
	job.Attempt = 1

	nre := NewNonRetryableError("state conflict")
	markJobFailed(job, nre)

	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	if updated.State != model.TdaiJobStateFailed {
		t.Fatalf("NonRetryable: should be FAILED immediately, got %s", updated.State)
	}
}

func TestMarkJobFailed_RollbacksSwitchStatus(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "inst-rollback",
		CurrentPlan:  model.MemoryPlanOff,
		SwitchStatus: model.MemorySwitchStatusSwitchingToFree,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-rollback", "inst-rollback", "{}", "u", "")
	model.DB(context.Background()).Model(job).Update("attempt", model.TdaiJobDefaultMaxAttempts)
	job.Attempt = model.TdaiJobDefaultMaxAttempts

	markJobFailed(job, errors.New("final failure"))

	var plugin model.MemoryTDAIPlugin
	model.DB(context.Background()).Where("instance_id = ?", "inst-rollback").First(&plugin)
	if plugin.SwitchStatus != model.MemorySwitchStatusNone {
		t.Errorf("switch_status should be rolled back to empty, got %q", plugin.SwitchStatus)
	}
}

// --- markJobSucceeded ---

func TestMarkJobSucceeded(t *testing.T) {
	setupMemoryProTestDB(t)

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-succ", "inst-succ", "{}", "u", "")
	markJobSucceeded(job)

	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	if updated.State != model.TdaiJobStateSucceeded {
		t.Fatalf("expected SUCCEEDED, got %s", updated.State)
	}
	if updated.Progress != 100 {
		t.Errorf("progress = %d, want 100", updated.Progress)
	}
	if updated.FinishedAt == nil {
		t.Error("finished_at should be set")
	}
}

// --- updateStep ---

func TestUpdateStep(t *testing.T) {
	setupMemoryProTestDB(t)

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-step", "inst-step", "{}", "u", "")
	updateStep(job, "validate_request", 25)

	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	if updated.CurrentStep != "validate_request" {
		t.Errorf("current_step = %q, want validate_request", updated.CurrentStep)
	}
	if updated.Progress != 25 {
		t.Errorf("progress = %d, want 25", updated.Progress)
	}
}

// --- getPluginByInstanceID ---

func TestGetPluginByInstanceID_Found(t *testing.T) {
	setupMemoryProTestDB(t)
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-gpbi",
		CurrentPlan: model.MemoryPlanFree,
	})

	plugin, err := getPluginByInstanceID(context.Background(), "inst-gpbi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plugin.CurrentPlan != model.MemoryPlanFree {
		t.Errorf("current_plan = %q, want FREE", plugin.CurrentPlan)
	}
}

func TestGetPluginByInstanceID_NotFound(t *testing.T) {
	setupMemoryProTestDB(t)

	_, err := getPluginByInstanceID(context.Background(), "inst-nonexist")
	if err == nil {
		t.Fatal("expected error for nonexistent instance")
	}
}

// --- checkInstanceRunning ---

func TestCheckInstanceRunning_Running(t *testing.T) {
	setupMemoryProTestDB(t)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-running", LastCVMState: "RUNNING"})

	err := checkInstanceRunning(context.Background(), "inst-running")
	if err != nil {
		t.Fatalf("RUNNING instance should pass, got: %v", err)
	}
}

func TestCheckInstanceRunning_NotRunning(t *testing.T) {
	setupMemoryProTestDB(t)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-stopped", LastCVMState: "STOPPED"})

	err := checkInstanceRunning(context.Background(), "inst-stopped")
	if err == nil {
		t.Fatal("STOPPED instance should fail")
	}
}

func TestCheckInstanceRunning_NotFound(t *testing.T) {
	setupMemoryProTestDB(t)

	err := checkInstanceRunning(context.Background(), "inst-gone")
	if err == nil {
		t.Fatal("nonexistent instance should fail")
	}
}

// --- executeJob (unknown type) ---

func TestExecuteJob_UnknownType(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-unk",
		CurrentPlan: model.MemoryPlanOff,
	})

	job := model.TdaiJob{
		JobType:     "UNKNOWN_TYPE",
		BizKey:      "switch:inst-unk",
		InstanceID:  "inst-unk",
		State:       model.TdaiJobStatePending,
		RunAt:       time.Now(),
		MaxAttempts: 3,
	}
	model.DB(context.Background()).Create(&job)

	executeJob(context.Background(), job.ID)

	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	// 未知类型会报错，但因 attempt 还未到 max，可能重试
	if updated.LastError == "" {
		t.Error("last_error should contain unknown type message")
	}
}

// --- Dispatcher entry ---

func TestRunTdaiDispatcherEntry_DisabledByEnv(t *testing.T) {
	os.Setenv("DISABLE_TDAI_TASK_ENGINE", "true")
	defer os.Unsetenv("DISABLE_TDAI_TASK_ENGINE")

	setupTestDB(t)
	// 调用不应 panic
	runTdaiDispatcherEntry(context.Background())
}

// --- handleSwitchToFree 补充测试 ---

func TestHandleSwitchToFree_FromOff_NeedsCVM(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-off-to-free",
		Status:      model.MemoryTDAIPluginStatusNotInstalled,
		CurrentPlan: model.MemoryPlanOff,
		DesiredPlan: model.MemoryPlanFree,
	})
	// 没有 Instance 行 → checkInstanceRunning 应失败

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-off-to-free", "inst-off-to-free", "{}", "u", "")
	err := handleSwitchToFree(job)
	if err == nil {
		t.Fatal("should fail because CVM instance not found")
	}
}

// --- handleSwitchToOff 补充测试 ---

func TestHandleSwitchToOff_FromFree_NeedsCVM(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-free-to-off",
		Status:      model.MemoryTDAIPluginStatusEnabled,
		CurrentPlan: model.MemoryPlanFree,
		DesiredPlan: model.MemoryPlanOff,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-free-to-off", "inst-free-to-off", "{}", "u", "")
	err := handleSwitchToOff(job)
	if err == nil {
		t.Fatal("should fail because CVM instance not found")
	}
}

// --- handleSwitchToPro validation ---

func TestHandleSwitchToPro_PluginNotFound(t *testing.T) {
	setupMemoryProTestDB(t)

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToPro, "switch:inst-noplugin", "inst-noplugin", "{}", "u", "")
	err := handleSwitchToPro(job)
	if err == nil {
		t.Fatal("should fail because plugin row not found")
	}
}

func TestHandleSwitchToPro_AlreadyProIncompleteBinding(t *testing.T) {
	setupMemoryProTestDB(t)

	// 已在 PRO 但 PoolID 为空 → 绑定不完整，应返回 NonRetryableError
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-pro-incomplete", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-pro-incomplete",
		CurrentPlan: model.MemoryPlanPro,
		PoolID:      "",
		Endpoint:    "",
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToPro, "switch:inst-pro-incomplete", "inst-pro-incomplete", "{}", "u", "")
	err := handleSwitchToPro(job)
	if err == nil {
		t.Fatal("should fail for incomplete PRO binding")
	}
	var nre *NonRetryableError
	if !errors.As(err, &nre) {
		t.Fatalf("expected NonRetryableError, got %T: %v", err, err)
	}
}

func TestHandleSwitchToPro_NeedsCVM(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-off-to-pro",
		CurrentPlan: model.MemoryPlanOff,
	})
	// 没有 Instance 行 → checkInstanceRunning 应失败

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToPro, "switch:inst-off-to-pro", "inst-off-to-pro", "{}", "u", "")
	err := handleSwitchToPro(job)
	if err == nil {
		t.Fatal("should fail because CVM instance not found")
	}
}

// --- safeTdaiDispatch ---

func TestSafeTdaiDispatch_NoPanic(t *testing.T) {
	setupMemoryProTestDB(t)

	// 不应 panic，即使没有待调度任务
	safeTdaiDispatch(context.Background(), "test-host")
}

func TestSafeTdaiDispatch_ConcurrencyGuard(t *testing.T) {
	setupMemoryProTestDB(t)

	// 模拟已在运行
	tdaiDispatcherRunning.Store("", true)
	defer tdaiDispatcherRunning.Store("", false)

	// 应直接返回（CAS 失败）
	safeTdaiDispatch(context.Background(), "test-host")
}
