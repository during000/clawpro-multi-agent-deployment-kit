package controller

import (
	"context"
	"errors"
	"sync"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func createExecutorTestTask(t *testing.T, db *gorm.DB, recordCount int) (model.SkillDistributionTask, []model.SkillDistributionRecord) {
	t.Helper()
	task := model.SkillDistributionTask{Slug: "executor", Source: model.SkillSourcePublic, Type: model.TaskTypeDistribute, Total: recordCount, Status: model.TaskStatusRunning}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	records := make([]model.SkillDistributionRecord, recordCount)
	for i := range records {
		records[i] = model.SkillDistributionRecord{
			TaskID: task.ID, InstanceID: uint(i + 1), InstanceCID: "ins",
			Version: "1.0.0", Type: model.TaskTypeDistribute, Status: model.RecordStatusPending,
		}
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create records: %v", err)
	}
	return task, records
}

func TestExecuteSkillTask_PersistsTerminalStatesAndReturnsError(t *testing.T) {
	db := setupSkillInstancesDB(t)
	task, records := createExecutorTestTask(t, db, 2)
	executionErr := errors.New("agent failed")
	completeCalls := 0

	err := executeSkillTask(SkillTaskConfig{
		Ctx: context.Background(), Task: task, Records: records,
		OnFailed: func(context.Context, model.SkillDistributionRecord) string {
			return model.RecordStatusUpgradeFailed
		},
		OnComplete: func(_ context.Context, success, failed int) {
			completeCalls++
			if success != 1 || failed != 1 {
				t.Fatalf("OnComplete counts = %d/%d", success, failed)
			}
		},
	}, func(_ context.Context, record model.SkillDistributionRecord) error {
		if record.ID == records[1].ID {
			return executionErr
		}
		return nil
	})
	if !errors.Is(err, executionErr) {
		t.Fatalf("executeSkillTask error = %v", err)
	}
	if completeCalls != 1 {
		t.Fatalf("OnComplete calls = %d", completeCalls)
	}

	var updatedTask model.SkillDistributionTask
	if err := db.First(&updatedTask, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updatedTask.Status != model.TaskStatusCompleted || updatedTask.Success != 1 || updatedTask.Failed != 1 {
		t.Fatalf("task = %+v", updatedTask)
	}
	var updatedRecords []model.SkillDistributionRecord
	if err := db.Where("task_id = ?", task.ID).Order("id ASC").Find(&updatedRecords).Error; err != nil {
		t.Fatal(err)
	}
	if updatedRecords[0].Status != model.RecordStatusSuccess || updatedRecords[1].Status != model.RecordStatusUpgradeFailed || updatedRecords[1].Error != executionErr.Error() {
		t.Fatalf("records = %+v", updatedRecords)
	}
}

func TestExecuteSkillTask_RecordPersistenceFailureDoesNotReportSuccess(t *testing.T) {
	db := setupSkillInstancesDB(t)
	task, records := createExecutorTestTask(t, db, 1)
	injectedErr := errors.New("injected record update failure")
	callbackName := "test:fail_first_record_status_update"
	failedOnce := false
	if err := db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "skill_distribution_records" && !failedOnce {
			failedOnce = true
			_ = tx.AddError(injectedErr)
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Update().Remove(callbackName) })

	successCalls := 0
	err := executeSkillTask(SkillTaskConfig{
		Ctx: context.Background(), Task: task, Records: records,
		OnSuccess: func(context.Context, model.SkillDistributionRecord) {
			successCalls++
		},
	}, func(context.Context, model.SkillDistributionRecord) error {
		return nil
	})
	if !errors.Is(err, injectedErr) {
		t.Fatalf("executeSkillTask error=%v, want %v", err, injectedErr)
	}
	if successCalls != 0 {
		t.Fatalf("OnSuccess calls=%d, want 0", successCalls)
	}

	var updatedTask model.SkillDistributionTask
	if err := db.First(&updatedTask, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updatedTask.Status != model.TaskStatusCompleted || updatedTask.Success != 0 || updatedTask.Failed != 1 {
		t.Fatalf("task=%+v", updatedTask)
	}
	var updatedRecord model.SkillDistributionRecord
	if err := db.First(&updatedRecord, records[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if updatedRecord.Status != model.RecordStatusFailed || updatedRecord.Error != injectedErr.Error() {
		t.Fatalf("record=%+v", updatedRecord)
	}
}

func TestExecuteSkillTask_ReturnsPersistenceError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SiteConfig{}, &model.SkillDistributionTask{}, &model.SkillDistributionRecord{}); err != nil {
		t.Fatal(err)
	}
	restore := model.UseDBForTestWithDriver(db, "sqlite")
	t.Cleanup(restore)
	task, records := createExecutorTestTask(t, db, 1)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}

	if err := executeSkillTask(SkillTaskConfig{Ctx: context.Background(), Task: task, Records: records}, func(context.Context, model.SkillDistributionRecord) error {
		return nil
	}); err == nil {
		t.Fatal("closed database should return persistence error")
	}
}

func TestExecuteSkillTaskAsync_ReturnsBeforeBackgroundCompletion(t *testing.T) {
	db := setupSkillInstancesDB(t)
	task, records := createExecutorTestTask(t, db, 1)
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseTask := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(func() {
		releaseTask()
		skillDistributeWG.Wait()
	})

	executeSkillTaskAsync(SkillTaskConfig{Ctx: context.Background(), Task: task, Records: records}, func(context.Context, model.SkillDistributionRecord) error {
		close(started)
		<-release
		return nil
	})
	<-started

	var pending model.SkillDistributionRecord
	if err := db.First(&pending, records[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if pending.Status != model.RecordStatusPending {
		t.Fatalf("record completed before executor release: %+v", pending)
	}
	releaseTask()
	skillDistributeWG.Wait()

	if err := db.First(&pending, records[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if pending.Status != model.RecordStatusSuccess {
		t.Fatalf("record status = %s", pending.Status)
	}
}

func TestRunSkillDistributeTask_AdminAsyncRegression(t *testing.T) {
	db := setupSkillInstancesDB(t)
	task, records := createExecutorTestTask(t, db, 1)
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseTask := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(func() {
		releaseTask()
		skillDistributeWG.Wait()
	})
	deps := skillOperationTestDependencies(nil)
	mockSkillOperationExecution(&deps, func(string, map[string]string) error {
		close(started)
		<-release
		return nil
	})

	item := skillTaskItem{
		Source: model.SkillSourcePublic, Slug: "executor", Version: "1.0.0",
		DownloadURL: "https://example.test/executor.zip",
	}
	info := map[uint]skillInstanceInfo{
		records[0].InstanceID: {
			ID: records[0].InstanceID, InstanceId: records[0].InstanceCID,
			RuntimeUser: "agent", AgentType: model.AgentTypeOpenClaw,
		},
	}
	runSkillDistributeTask(context.Background(), item, task, records, nil, info, deps.execution)
	<-started

	var pending model.SkillDistributionRecord
	if err := db.First(&pending, records[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if pending.Status != model.RecordStatusPending {
		t.Fatalf("admin wrapper blocked until completion: %+v", pending)
	}
	releaseTask()
	skillDistributeWG.Wait()
	if err := db.First(&pending, records[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if pending.Status != model.RecordStatusSuccess {
		t.Fatalf("record status = %s", pending.Status)
	}
}

func TestRunSkillUninstallTask_AdminAsyncRegression(t *testing.T) {
	db := setupSkillInstancesDB(t)
	task, records := createExecutorTestTask(t, db, 1)
	task.Type = model.TaskTypeUninstall
	records[0].Type = model.TaskTypeUninstall
	if err := db.Model(&task).Update("type", task.Type).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&records[0]).Update("type", records[0].Type).Error; err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseTask := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(func() {
		releaseTask()
		skillDistributeWG.Wait()
	})
	deps := skillOperationTestDependencies(nil)
	mockSkillOperationExecution(&deps, func(string, map[string]string) error {
		close(started)
		<-release
		return nil
	})

	item := skillTaskItem{Source: model.SkillSourcePublic, Slug: "executor", Version: "1.0.0"}
	info := map[uint]skillInstanceInfo{
		records[0].InstanceID: {
			ID: records[0].InstanceID, InstanceId: records[0].InstanceCID,
			RuntimeUser: "agent", AgentType: model.AgentTypeOpenClaw,
		},
	}
	runSkillUninstallTask(context.Background(), item, task, records, nil, info, deps.execution)
	<-started

	var pending model.SkillDistributionRecord
	if err := db.First(&pending, records[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if pending.Status != model.RecordStatusPending {
		t.Fatalf("admin wrapper blocked until completion: %+v", pending)
	}
	releaseTask()
	skillDistributeWG.Wait()
	if err := db.First(&pending, records[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if pending.Status != model.RecordStatusSuccess {
		t.Fatalf("record status = %s", pending.Status)
	}
}
