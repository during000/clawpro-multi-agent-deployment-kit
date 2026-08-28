package task

import (
	"context"
	"os"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建临时 SQLite 测试数据库。
func setupTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试 DB 失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	oldSnap := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: ""}
	t.Cleanup(func() { common.FixedSnapshot = oldSnap })
	if err := db.AutoMigrate(
		&model.MemoryTDAIPlugin{},
		&model.TdaiJob{},
		&model.SiteConfig{},
		&model.Instance{},
	); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	// Seed 默认配置
	db.Create(&model.SiteConfig{
		MemoryTDAIEnable:            false,
		MemoryTDAISupportedVersions: "[]",
	})
}

// --- SubmitJob 测试 ---

func TestSubmitJob_Basic(t *testing.T) {
	setupTestDB(t)

	job, err := model.SubmitJob(
		context.Background(),
		model.TdaiJobTypeSwitchToFree,
		"switch:inst-001",
		"inst-001",
		"{}",
		"test-user",
		"trace-001",
	)
	if err != nil {
		t.Fatalf("SubmitJob 失败: %v", err)
	}
	if job.ID == 0 {
		t.Fatal("job.ID 应不为 0")
	}
	if job.State != model.TdaiJobStatePending {
		t.Fatalf("期望 PENDING，得到 %s", job.State)
	}
	if job.JobType != model.TdaiJobTypeSwitchToFree {
		t.Fatalf("期望 SWITCH_TO_FREE，得到 %s", job.JobType)
	}
}

func TestSubmitJob_Idempotent(t *testing.T) {
	setupTestDB(t)

	job1, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-002", "inst-002", "{}", "u", "t")
	job2, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-002", "inst-002", "{}", "u", "t")

	if job1.ID != job2.ID {
		t.Fatalf("幂等失败：期望返回相同 job，得到 %d vs %d", job1.ID, job2.ID)
	}
}

func TestSubmitJob_AllowAfterCompleted(t *testing.T) {
	setupTestDB(t)

	job1, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-003", "inst-003", "{}", "u", "t")
	// 手动标记成功
	now := time.Now()
	model.DB(context.Background()).Model(job1).Updates(map[string]any{
		"state":       model.TdaiJobStateSucceeded,
		"finished_at": &now,
	})

	// 同 biz_key 应可再次提交
	job2, err := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-003", "inst-003", "{}", "u", "t")
	if err != nil {
		t.Fatalf("已完成后重新提交失败: %v", err)
	}
	if job2.ID == job1.ID {
		t.Fatal("应创建新任务")
	}
}

// --- CancelJob / RetryJob 测试 ---

func TestCancelJob(t *testing.T) {
	setupTestDB(t)

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-cancel", "inst-cancel", "{}", "u", "t")
	if err := model.CancelJob(context.Background(), job.ID); err != nil {
		t.Fatalf("CancelJob 失败: %v", err)
	}
	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	if updated.State != model.TdaiJobStateCanceled {
		t.Fatalf("期望 CANCELED，得到 %s", updated.State)
	}
}

func TestRetryJob(t *testing.T) {
	setupTestDB(t)

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-retry", "inst-retry", "{}", "u", "t")
	// 手动标记失败
	model.DB(context.Background()).Model(job).Update("state", model.TdaiJobStateFailed)

	if err := model.RetryJob(context.Background(), job.ID); err != nil {
		t.Fatalf("RetryJob 失败: %v", err)
	}
	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	if updated.State != model.TdaiJobStatePending {
		t.Fatalf("期望 PENDING，得到 %s", updated.State)
	}
}

// --- Handler 测试（不依赖真实脚本执行） ---

func TestHandleSwitchToFree_AlreadyFree(t *testing.T) {
	setupTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-free", AgentType: model.AgentTypeOpenClaw})
	// 创建 plugin 行，current_plan=FREE
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-free",
		Status:      model.MemoryTDAIPluginStatusEnabled,
		CurrentPlan: model.MemoryPlanFree,
		DesiredPlan: model.MemoryPlanFree,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-free", "inst-free", "{}", "u", "t")
	err := handleSwitchToFree(job)
	if err != nil {
		t.Fatalf("已在 FREE 应直接成功，得到错误: %v", err)
	}
}

func TestHandleSwitchToFree_FromProBlocked(t *testing.T) {
	setupTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-pro", AgentType: model.AgentTypeOpenClaw})

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-pro",
		Status:      model.MemoryTDAIPluginStatusEnabled,
		CurrentPlan: model.MemoryPlanPro,
		DesiredPlan: model.MemoryPlanPro,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-pro", "inst-pro", "{}", "u", "t")
	err := handleSwitchToFree(job)
	if err == nil {
		t.Fatal("从 Pro 切 Free 应报错")
	}
}

func TestHandleSwitchToOff_AlreadyOff(t *testing.T) {
	setupTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-off", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-off",
		Status:      model.MemoryTDAIPluginStatusNotInstalled,
		CurrentPlan: model.MemoryPlanOff,
		DesiredPlan: model.MemoryPlanOff,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-off", "inst-off", "{}", "u", "t")
	err := handleSwitchToOff(job)
	if err != nil {
		t.Fatalf("已在 OFF 应直接成功，得到错误: %v", err)
	}
}

// --- Dispatcher 调度测试 ---

func TestDispatcher_PicksUpPendingJob(t *testing.T) {
	setupTestDB(t)

	// 创建 plugin 行，current_plan 已经是 OFF，提交 SWITCH_TO_OFF 任务
	// handler 会发现"已在 OFF"直接成功，不调用脚本
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-dispatch", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-dispatch",
		Status:      model.MemoryTDAIPluginStatusNotInstalled,
		CurrentPlan: model.MemoryPlanOff,
		DesiredPlan: model.MemoryPlanOff,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-dispatch", "inst-dispatch", "{}", "u", "t")

	hostname, _ := os.Hostname()
	dispatchPendingJobs(context.Background(), hostname)

	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	if updated.State != model.TdaiJobStateSucceeded {
		t.Fatalf("期望 SUCCEEDED，得到 %s（last_error: %s）", updated.State, updated.LastError)
	}
	if updated.Attempt != 1 {
		t.Fatalf("期望 attempt=1，得到 %d", updated.Attempt)
	}
	if updated.Progress != 100 {
		t.Fatalf("期望 progress=100，得到 %d", updated.Progress)
	}
}

// TestDispatcher_EmptyQueueShortCircuits 验证无 PENDING 任务时
// dispatchPendingJobs 走"先扫再锁"的快速路径——不进入持锁流程。
// SQLite 模式下 AcquireLock 是空操作，这里主要确认空队列下不抛错、不修改任何数据。
func TestDispatcher_EmptyQueueShortCircuits(t *testing.T) {
	setupTestDB(t)

	hostname, _ := os.Hostname()
	// 没有任何 PENDING 任务，预扫返回空，函数应安静返回。
	dispatchPendingJobs(context.Background(), hostname)

	var count int64
	model.DB(context.Background()).Model(&model.TdaiJob{}).Count(&count)
	if count != 0 {
		t.Fatalf("空队列预扫不应产生任何 job，得到 %d", count)
	}
}

// TestDispatcher_OnlyFutureJob_ShortCircuits 验证只有未来 run_at 任务时
// 预扫返回空，同样走快速路径。
func TestDispatcher_OnlyFutureJob_ShortCircuits(t *testing.T) {
	setupTestDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-only-future",
		CurrentPlan: model.MemoryPlanOff,
	})
	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-only-future", "inst-only-future", "{}", "u", "t")
	model.DB(context.Background()).Model(job).Update("run_at", time.Now().Add(1*time.Hour))

	hostname, _ := os.Hostname()
	dispatchPendingJobs(context.Background(), hostname)

	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	if updated.State != model.TdaiJobStatePending {
		t.Fatalf("未来 run_at 任务不应被抢占，得到 state=%s", updated.State)
	}
	if updated.Attempt != 0 {
		t.Fatalf("未来 run_at 任务不应增加 attempt，得到 %d", updated.Attempt)
	}
}

func TestDispatcher_FutureRunAtNotPickedUp(t *testing.T) {
	setupTestDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-future",
		Status:      model.MemoryTDAIPluginStatusNotInstalled,
		CurrentPlan: model.MemoryPlanOff,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-future", "inst-future", "{}", "u", "t")
	// 把 run_at 设到未来
	model.DB(context.Background()).Model(job).Update("run_at", time.Now().Add(1*time.Hour))

	hostname, _ := os.Hostname()
	dispatchPendingJobs(context.Background(), hostname)

	var updated model.TdaiJob
	model.DB(context.Background()).First(&updated, job.ID)
	if updated.State != model.TdaiJobStatePending {
		t.Fatalf("run_at 在未来，不应被调度，得到 %s", updated.State)
	}
}
