package task

import (
	"context"
	"os"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupRecoverTestDB 创建临时 SQLite 数据库，并将 model.DB(context.Background()) 替换为测试库。
// 返回 cleanup 函数，测试结束后调用以恢复原始 DB 并删除临时文件。
func setupRecoverTestDB(t *testing.T) func() {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "hatchery_recover_test_*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()

	dsn := tmpFile.Name() + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("open test db: %v", err)
	}

	if err := testDB.AutoMigrate(
		&model.SkillDistributionTask{},
		&model.SkillDistributionRecord{},
		&model.PluginDistributionTask{},
		&model.PluginDistributionRecord{},
		&model.SkillInstallation{},
		&model.PluginInstallation{},
		&model.RoleDistributionRecord{},
		&model.Instance{},
	); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("auto migrate: %v", err)
	}

	origDB := model.UseDBForTest(testDB)
	return func() {
		origDB()
		sqlDB, _ := testDB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
	}
}

// ─── recoverInterruptedSkillTasks 测试 ───────────────────────────────────────

// TestrecoverInterruptedSkillTasks_NoTasks 验证没有 running 任务时函数正常返回，不报错。
func TestRecoverInterruptedSkillTasks_NoTasks(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	// 不插入任何数据，直接调用
	recoverInterruptedSkillTasks(context.Background())
	// 无 panic / 无 fatal 即通过
}

// TestrecoverInterruptedSkillTasks_AllPending 验证：running 任务下所有 pending 记录被标记为 failed，
// 任务状态更新为 completed，success/failed 计数正确。
func TestRecoverInterruptedSkillTasks_AllPending(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	task := model.SkillDistributionTask{
		SkillID: 1,
		Version: "1.0.0",
		Total:   3,
		Status:  "running",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	// 3 条 pending 记录
	for i := 1; i <= 3; i++ {
		rec := model.SkillDistributionRecord{
			TaskID:     task.ID,
			SkillID:    1,
			InstanceID: uint(i),
			Status:     "pending",
		}
		if err := model.DB(context.Background()).Create(&rec).Error; err != nil {
			t.Fatalf("create record: %v", err)
		}
	}

	recoverInterruptedSkillTasks(context.Background())

	// 验证所有 pending 记录已变为 failed
	var pendingCount int64
	model.DB(context.Background()).Model(&model.SkillDistributionRecord{}).
		Where("task_id = ? AND status = ?", task.ID, "pending").Count(&pendingCount)
	if pendingCount != 0 {
		t.Errorf("期望 pending=0，实际=%d", pendingCount)
	}

	var failedCount int64
	model.DB(context.Background()).Model(&model.SkillDistributionRecord{}).
		Where("task_id = ? AND status = ?", task.ID, "failed").Count(&failedCount)
	if failedCount != 3 {
		t.Errorf("期望 failed=3，实际=%d", failedCount)
	}

	// 验证任务状态
	var updatedTask model.SkillDistributionTask
	model.DB(context.Background()).First(&updatedTask, task.ID)
	if updatedTask.Status != "completed" {
		t.Errorf("期望 task.Status=completed，实际=%s", updatedTask.Status)
	}
	if updatedTask.Success != 0 {
		t.Errorf("期望 task.Success=0，实际=%d", updatedTask.Success)
	}
	if updatedTask.Failed != 3 {
		t.Errorf("期望 task.Failed=3，实际=%d", updatedTask.Failed)
	}
}

// TestrecoverInterruptedSkillTasks_MixedRecords 验证：running 任务下有 success/failed/pending 混合记录，
// 只有 pending 被改为 failed，success 计数正确。
func TestRecoverInterruptedSkillTasks_MixedRecords(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	task := model.SkillDistributionTask{
		SkillID: 2,
		Version: "1.0.0",
		Total:   4,
		Status:  "running",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	statuses := []string{"success", "success", "failed", "pending"}
	for i, s := range statuses {
		rec := model.SkillDistributionRecord{
			TaskID:     task.ID,
			SkillID:    2,
			InstanceID: uint(i + 1),
			Status:     s,
		}
		if err := model.DB(context.Background()).Create(&rec).Error; err != nil {
			t.Fatalf("create record: %v", err)
		}
	}

	recoverInterruptedSkillTasks(context.Background())

	var updatedTask model.SkillDistributionTask
	model.DB(context.Background()).First(&updatedTask, task.ID)

	if updatedTask.Status != "completed" {
		t.Errorf("期望 task.Status=completed，实际=%s", updatedTask.Status)
	}
	// success 仍为 2，failed = 原 1 + 恢复的 1 = 2
	if updatedTask.Success != 2 {
		t.Errorf("期望 task.Success=2，实际=%d", updatedTask.Success)
	}
	if updatedTask.Failed != 2 {
		t.Errorf("期望 task.Failed=2，实际=%d", updatedTask.Failed)
	}
}

// TestrecoverInterruptedSkillTasks_AlreadyCompleted 验证：已 completed 的任务不会被二次处理。
func TestRecoverInterruptedSkillTasks_AlreadyCompleted(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	task := model.SkillDistributionTask{
		SkillID: 3,
		Version: "1.0.0",
		Total:   1,
		Success: 1,
		Status:  "completed",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	recoverInterruptedSkillTasks(context.Background())

	var updatedTask model.SkillDistributionTask
	model.DB(context.Background()).First(&updatedTask, task.ID)
	// 状态不应改变
	if updatedTask.Status != "completed" {
		t.Errorf("completed 任务不应被修改，实际 status=%s", updatedTask.Status)
	}
	if updatedTask.Success != 1 {
		t.Errorf("completed 任务 success 不应改变，实际=%d", updatedTask.Success)
	}
}

// ─── recoverInterruptedPluginTasks 测试 ──────────────────────────────────────

// TestrecoverInterruptedPluginTasks_NoTasks 验证没有 running 任务时函数正常返回。
func TestRecoverInterruptedPluginTasks_NoTasks(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	recoverInterruptedPluginTasks(context.Background())
}

// TestrecoverInterruptedPluginTasks_AllPending 验证：running 任务下所有 pending 记录被标记为 failed，
// 任务状态更新为 completed，计数正确。
func TestRecoverInterruptedPluginTasks_AllPending(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	task := model.PluginDistributionTask{
		PluginDBID: 1,
		Version:    "1.0.0",
		Total:      2,
		Status:     "running",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	for i := 1; i <= 2; i++ {
		rec := model.PluginDistributionRecord{
			TaskID:     task.ID,
			PluginDBID: 1,
			InstanceID: uint(i),
			Status:     "pending",
		}
		if err := model.DB(context.Background()).Create(&rec).Error; err != nil {
			t.Fatalf("create record: %v", err)
		}
	}

	recoverInterruptedPluginTasks(context.Background())

	var pendingCount int64
	model.DB(context.Background()).Model(&model.PluginDistributionRecord{}).
		Where("task_id = ? AND status = ?", task.ID, "pending").Count(&pendingCount)
	if pendingCount != 0 {
		t.Errorf("期望 pending=0，实际=%d", pendingCount)
	}

	var updatedTask model.PluginDistributionTask
	model.DB(context.Background()).First(&updatedTask, task.ID)
	if updatedTask.Status != "completed" {
		t.Errorf("期望 task.Status=completed，实际=%s", updatedTask.Status)
	}
	if updatedTask.Failed != 2 {
		t.Errorf("期望 task.Failed=2，实际=%d", updatedTask.Failed)
	}
	if updatedTask.Success != 0 {
		t.Errorf("期望 task.Success=0，实际=%d", updatedTask.Success)
	}
}

// TestrecoverInterruptedPluginTasks_MixedRecords 验证：running 任务下有 success/pending 混合记录，
// 只有 pending 被改为 failed，success 计数正确。
func TestRecoverInterruptedPluginTasks_MixedRecords(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	task := model.PluginDistributionTask{
		PluginDBID: 2,
		Version:    "1.0.0",
		Total:      3,
		Status:     "running",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	statuses := []string{"success", "pending", "pending"}
	for i, s := range statuses {
		rec := model.PluginDistributionRecord{
			TaskID:     task.ID,
			PluginDBID: 2,
			InstanceID: uint(i + 1),
			Status:     s,
		}
		if err := model.DB(context.Background()).Create(&rec).Error; err != nil {
			t.Fatalf("create record: %v", err)
		}
	}

	recoverInterruptedPluginTasks(context.Background())

	var updatedTask model.PluginDistributionTask
	model.DB(context.Background()).First(&updatedTask, task.ID)

	if updatedTask.Status != "completed" {
		t.Errorf("期望 task.Status=completed，实际=%s", updatedTask.Status)
	}
	if updatedTask.Success != 1 {
		t.Errorf("期望 task.Success=1，实际=%d", updatedTask.Success)
	}
	if updatedTask.Failed != 2 {
		t.Errorf("期望 task.Failed=2，实际=%d", updatedTask.Failed)
	}
}

// TestrecoverInterruptedPluginTasks_AlreadyCompleted 验证：已 completed 的任务不会被二次处理。
func TestRecoverInterruptedPluginTasks_AlreadyCompleted(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	task := model.PluginDistributionTask{
		PluginDBID: 3,
		Version:    "1.0.0",
		Total:      1,
		Success:    1,
		Status:     "completed",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	recoverInterruptedPluginTasks(context.Background())

	var updatedTask model.PluginDistributionTask
	model.DB(context.Background()).First(&updatedTask, task.ID)
	if updatedTask.Status != "completed" {
		t.Errorf("completed 任务不应被修改，实际 status=%s", updatedTask.Status)
	}
	if updatedTask.Success != 1 {
		t.Errorf("completed 任务 success 不应改变，实际=%d", updatedTask.Success)
	}
}

// TestrecoverInterruptedPluginTasks_ErrorMessageSet 验证：恢复后 pending 记录的 error 字段被设置为"服务重启中断"。
func TestRecoverInterruptedPluginTasks_ErrorMessageSet(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	task := model.PluginDistributionTask{
		PluginDBID: 4,
		Version:    "1.0.0",
		Total:      1,
		Status:     "running",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	rec := model.PluginDistributionRecord{
		TaskID:     task.ID,
		PluginDBID: 4,
		InstanceID: 1,
		Status:     "pending",
	}
	if err := model.DB(context.Background()).Create(&rec).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	recoverInterruptedPluginTasks(context.Background())

	var updatedRec model.PluginDistributionRecord
	model.DB(context.Background()).First(&updatedRec, rec.ID)
	if updatedRec.Error != "服务重启中断" {
		t.Errorf("期望 error='服务重启中断'，实际='%s'", updatedRec.Error)
	}
}

// TestrecoverInterruptedSkillTasks_ErrorMessageSet 验证：恢复后 pending 记录的 error 字段被设置为"服务重启中断"。
func TestRecoverInterruptedSkillTasks_ErrorMessageSet(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	task := model.SkillDistributionTask{
		SkillID: 10,
		Version: "1.0.0",
		Total:   1,
		Status:  "running",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	rec := model.SkillDistributionRecord{
		TaskID:     task.ID,
		SkillID:    10,
		InstanceID: 1,
		Status:     "pending",
	}
	if err := model.DB(context.Background()).Create(&rec).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	recoverInterruptedSkillTasks(context.Background())

	var updatedRec model.SkillDistributionRecord
	model.DB(context.Background()).First(&updatedRec, rec.ID)
	if updatedRec.Error != "服务重启中断" {
		t.Errorf("期望 error='服务重启中断'，实际='%s'", updatedRec.Error)
	}
}

// ─── 错误路径测试（DB 不可用时的 early-return / continue 分支）────────────────

// setupBrokenDB 创建一个底层连接已关闭的 *gorm.DB，用于触发 DB 错误分支。
// 返回的 cleanup 会恢复 model.DB(context.Background())。
func setupBrokenDB(t *testing.T) func() {
	t.Helper()

	// 先建一个正常的内存 DB，然后立即关闭底层连接
	tmpFile, err := os.CreateTemp("", "hatchery_broken_test_*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()

	dsn := tmpFile.Name() + "?_pragma=journal_mode(WAL)"
	brokenDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("open broken db: %v", err)
	}

	// 关闭底层连接，使后续所有 DB 操作都返回错误
	sqlDB, _ := brokenDB.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}

	origDB := model.UseDBForTest(brokenDB)
	return func() {
		origDB()
		os.Remove(tmpFile.Name())
	}
}

// TestrecoverInterruptedSkillTasks_DBError 验证：DB 不可用时，函数触发 early-return 错误分支，不 panic。
func TestRecoverInterruptedSkillTasks_DBError(t *testing.T) {
	cleanup := setupBrokenDB(t)
	defer cleanup()

	// DB 已关闭，Find 会返回错误 → 触发 early-return 分支
	recoverInterruptedSkillTasks(context.Background())
	// 不 panic 即通过
}

// TestrecoverInterruptedPluginTasks_DBError 验证：DB 不可用时，函数触发 early-return 错误分支，不 panic。
func TestRecoverInterruptedPluginTasks_DBError(t *testing.T) {
	cleanup := setupBrokenDB(t)
	defer cleanup()

	recoverInterruptedPluginTasks(context.Background())
}

// TestrecoverInterruptedSkillTasks_UpdateRecordError 验证：更新 pending 记录失败时，
// 触发 continue 分支（跳过该任务），函数不 panic。
func TestRecoverInterruptedSkillTasks_UpdateRecordError(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	// 插入一个 running 任务，但不创建 skill_distribution_records 表对应的记录，
	// 通过删除表来触发 Updates 错误
	task := model.SkillDistributionTask{
		SkillID: 20,
		Version: "1.0.0",
		Total:   1,
		Status:  "running",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	// 删除 skill_distribution_records 表，使 Updates 操作失败
	if err := model.DB(context.Background()).Exec("DROP TABLE skill_distribution_records").Error; err != nil {
		t.Fatalf("drop table: %v", err)
	}

	// Updates 会失败 → 触发 continue 分支
	recoverInterruptedSkillTasks(context.Background())
	// 不 panic 即通过
}

// TestrecoverInterruptedPluginTasks_UpdateRecordError 验证：更新 pending 记录失败时，
// 触发 continue 分支，函数不 panic。
func TestRecoverInterruptedPluginTasks_UpdateRecordError(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	task := model.PluginDistributionTask{
		PluginDBID: 10,
		Version:    "1.0.0",
		Total:      1,
		Status:     "running",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	// 删除 plugin_distribution_records 表，使 Updates 操作失败
	if err := model.DB(context.Background()).Exec("DROP TABLE plugin_distribution_records").Error; err != nil {
		t.Fatalf("drop table: %v", err)
	}

	recoverInterruptedPluginTasks(context.Background())
}

// TestrecoverInterruptedSkillTasks_UpdateTaskError 验证：更新任务状态失败时，
// 触发 continue 分支，函数不 panic。
func TestRecoverInterruptedSkillTasks_UpdateTaskError(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	task := model.SkillDistributionTask{
		SkillID: 30,
		Version: "1.0.0",
		Total:   1,
		Status:  "running",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	rec := model.SkillDistributionRecord{
		TaskID:     task.ID,
		SkillID:    30,
		InstanceID: 1,
		Status:     "pending",
	}
	if err := model.DB(context.Background()).Create(&rec).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	// 删除 skill_distribution_tasks 表，使 task.Updates 失败
	if err := model.DB(context.Background()).Exec("DROP TABLE skill_distribution_tasks").Error; err != nil {
		t.Fatalf("drop table: %v", err)
	}

	recoverInterruptedSkillTasks(context.Background())
}

// TestrecoverInterruptedPluginTasks_UpdateTaskError 验证：更新任务状态失败时，
// 触发 continue 分支，函数不 panic。
func TestRecoverInterruptedPluginTasks_UpdateTaskError(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	task := model.PluginDistributionTask{
		PluginDBID: 20,
		Version:    "1.0.0",
		Total:      1,
		Status:     "running",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	rec := model.PluginDistributionRecord{
		TaskID:     task.ID,
		PluginDBID: 20,
		InstanceID: 1,
		Status:     "pending",
	}
	if err := model.DB(context.Background()).Create(&rec).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	// 删除 plugin_distribution_tasks 表，使 task.Updates 失败
	if err := model.DB(context.Background()).Exec("DROP TABLE plugin_distribution_tasks").Error; err != nil {
		t.Fatalf("drop table: %v", err)
	}

	recoverInterruptedPluginTasks(context.Background())
}

// TestrecoverInterruptedSkillTasks_UpdateTaskErrorViaTrigger 验证：records 更新成功但 task.Updates 失败时，
// 触发 continue 分支，函数不 panic。
// 通过 SQLite BEFORE UPDATE 触发器让 tasks 表的 UPDATE 抛出错误来模拟。
func TestRecoverInterruptedSkillTasks_UpdateTaskErrorViaTrigger(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	task := model.SkillDistributionTask{
		SkillID: 40,
		Version: "1.0.0",
		Total:   1,
		Status:  "running",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	rec := model.SkillDistributionRecord{
		TaskID:     task.ID,
		SkillID:    40,
		InstanceID: 1,
		Status:     "pending",
	}
	if err := model.DB(context.Background()).Create(&rec).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	// 创建触发器：当 skill_distribution_tasks 的 status 被更新为 'completed' 时抛出错误
	triggerSQL := `CREATE TRIGGER block_task_update
		BEFORE UPDATE OF status ON skill_distribution_tasks
		FOR EACH ROW WHEN NEW.status = 'completed'
		BEGIN
			SELECT RAISE(ABORT, 'blocked by test trigger');
		END`
	if err := model.DB(context.Background()).Exec(triggerSQL).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	// records 更新成功，但 task.Updates 会因触发器失败 → 触发 continue 分支
	recoverInterruptedSkillTasks(context.Background())
	// 不 panic 即通过
}

// TestrecoverInterruptedPluginTasks_UpdateTaskErrorViaTrigger 验证：records 更新成功但 task.Updates 失败时，
// 触发 continue 分支，函数不 panic。
func TestRecoverInterruptedPluginTasks_UpdateTaskErrorViaTrigger(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	task := model.PluginDistributionTask{
		PluginDBID: 30,
		Version:    "1.0.0",
		Total:      1,
		Status:     "running",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	rec := model.PluginDistributionRecord{
		TaskID:     task.ID,
		PluginDBID: 30,
		InstanceID: 1,
		Status:     "pending",
	}
	if err := model.DB(context.Background()).Create(&rec).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	// 创建触发器：当 plugin_distribution_tasks 的 status 被更新为 'completed' 时抛出错误
	triggerSQL := `CREATE TRIGGER block_plugin_task_update
		BEFORE UPDATE OF status ON plugin_distribution_tasks
		FOR EACH ROW WHEN NEW.status = 'completed'
		BEGIN
			SELECT RAISE(ABORT, 'blocked by test trigger');
		END`
	if err := model.DB(context.Background()).Exec(triggerSQL).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	recoverInterruptedPluginTasks(context.Background())
}

// TestrecoverInterruptedSkillTasks_CountErrorAfterUpdate 验证：records 更新成功但 Count 查询失败时，
// 触发 Count 错误分支（slog.Error），函数继续执行不 panic。
// 通过 GORM Callback 在 Updates 之后重命名 records 表，使 Count 操作失败。
func TestRecoverInterruptedSkillTasks_CountErrorAfterUpdate(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	task := model.SkillDistributionTask{
		SkillID: 50,
		Version: "1.0.0",
		Total:   1,
		Status:  "running",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	rec := model.SkillDistributionRecord{
		TaskID:     task.ID,
		SkillID:    50,
		InstanceID: 1,
		Status:     "pending",
	}
	if err := model.DB(context.Background()).Create(&rec).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	// 注册 AFTER UPDATE callback：records 表更新后立即重命名，使后续 Count 失败
	called := false
	model.DB(context.Background()).Callback().Update().After("gorm:update").Register("test:rename_records_after_update", func(db *gorm.DB) {
		if called {
			return
		}
		if db.Statement != nil && db.Statement.Table == "skill_distribution_records" {
			called = true
			// 重命名 records 表，使后续 Count 查询失败
			db.Exec("ALTER TABLE skill_distribution_records RENAME TO skill_distribution_records_bak")
		}
	})
	defer model.DB(context.Background()).Callback().Update().Remove("test:rename_records_after_update")

	// records 更新成功后，records 表被重命名 → Count 失败 → slog.Error 分支被覆盖
	// task.Updates 仍然成功（tasks 表未受影响）
	recoverInterruptedSkillTasks(context.Background())
}

// ─── RecoverInterrupted* 集成测试（已迁移到 model 包）────────────────────────

// TestRecoverInterruptedTasks_Integration 验证 model.RecoverInterrupted* 在有
// running 任务时正确将 pending 记录标为 failed 并将任务标为 completed。
func TestRecoverInterruptedTasks_Integration(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	// 构造一个 running 的 skill 任务 + 一个 pending 记录
	skillTask := model.SkillDistributionTask{
		SkillID: 100,
		Version: "1.0.0",
		Total:   1,
		Status:  "running",
	}
	if err := model.DB(context.Background()).Create(&skillTask).Error; err != nil {
		t.Fatalf("create skill task: %v", err)
	}
	skillRec := model.SkillDistributionRecord{
		TaskID:     skillTask.ID,
		SkillID:    100,
		InstanceID: 1,
		Status:     "pending",
	}
	if err := model.DB(context.Background()).Create(&skillRec).Error; err != nil {
		t.Fatalf("create skill record: %v", err)
	}

	// 构造一个 running 的 plugin 任务 + 一个 pending 记录
	pluginTask := model.PluginDistributionTask{
		PluginDBID: 200,
		Version:    "1.0.0",
		Total:      1,
		Status:     "running",
	}
	if err := model.DB(context.Background()).Create(&pluginTask).Error; err != nil {
		t.Fatalf("create plugin task: %v", err)
	}
	pluginRec := model.PluginDistributionRecord{
		TaskID:     pluginTask.ID,
		PluginDBID: 200,
		InstanceID: 1,
		Status:     "pending",
	}
	if err := model.DB(context.Background()).Create(&pluginRec).Error; err != nil {
		t.Fatalf("create plugin record: %v", err)
	}

	recoverInterruptedSkillTasks(context.Background())
	recoverInterruptedPluginTasks(context.Background())

	// skill 任务已恢复
	var updatedSkillTask model.SkillDistributionTask
	model.DB(context.Background()).First(&updatedSkillTask, skillTask.ID)
	if updatedSkillTask.Status != "completed" {
		t.Errorf("期望 skillTask.Status=completed，实际=%s", updatedSkillTask.Status)
	}
	if updatedSkillTask.Failed != 1 {
		t.Errorf("期望 skillTask.Failed=1，实际=%d", updatedSkillTask.Failed)
	}

	// plugin 任务已恢复
	var updatedPluginTask model.PluginDistributionTask
	model.DB(context.Background()).First(&updatedPluginTask, pluginTask.ID)
	if updatedPluginTask.Status != "completed" {
		t.Errorf("期望 pluginTask.Status=completed，实际=%s", updatedPluginTask.Status)
	}
	if updatedPluginTask.Failed != 1 {
		t.Errorf("期望 pluginTask.Failed=1，实际=%d", updatedPluginTask.Failed)
	}
}

// TestRecoverInterruptedTasks_NoData 验证无 running 任务时不报错。
func TestRecoverInterruptedTasks_NoData(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	// 不插入任何 running 任务，执行不应 panic
	recoverInterruptedSkillTasks(context.Background())
	recoverInterruptedPluginTasks(context.Background())
}

// ─── recoverInterruptedSkillInitTasks 测试 ───────────────────────────────

// TestRecoverInterruptedSkillInitTasks_NoRecords 验证无中间状态记录时函数正常返回。
func TestRecoverInterruptedSkillInitTasks_NoRecords(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	recoverInterruptedSkillInitTasks(context.Background())
}

// TestRecoverInterruptedSkillInitTasks_InstallingToFailed 验证 Installing 状态的记录被标记为 Failed。
func TestRecoverInterruptedSkillInitTasks_InstallingToFailed(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	records := []model.SkillInstallation{
		{InstanceID: 1, Name: "s1", Slug: "skill-1", Version: "1.0.0", InstallStatus: model.SkillInstalling},
		{InstanceID: 1, Name: "s2", Slug: "skill-2", Version: "1.0.0", InstallStatus: model.SkillInstalling},
	}
	for _, r := range records {
		if err := model.DB(context.Background()).Create(&r).Error; err != nil {
			t.Fatalf("create record: %v", err)
		}
	}

	recoverInterruptedSkillInitTasks(context.Background())

	var installations []model.SkillInstallation
	model.DB(context.Background()).Where("instance_id = ?", 1).Find(&installations)
	for _, inst := range installations {
		if inst.InstallStatus != model.SkillInstallFailed {
			t.Errorf("slug=%s: 期望 install_status=%d(Failed), 实际=%d", inst.Slug, model.SkillInstallFailed, inst.InstallStatus)
		}
		if inst.ErrorMessage != "服务重启中断" {
			t.Errorf("slug=%s: 期望 error_message='服务重启中断', 实际='%s'", inst.Slug, inst.ErrorMessage)
		}
	}
}

// TestRecoverInterruptedSkillInitTasks_NoneToFailed 验证 None 状态的记录也被标记为 Failed。
func TestRecoverInterruptedSkillInitTasks_NoneToFailed(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	rec := model.SkillInstallation{
		InstanceID: 2, Name: "s3", Slug: "skill-3", Version: "1.0.0",
		InstallStatus: model.SkillInstallNone,
	}
	if err := model.DB(context.Background()).Create(&rec).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	recoverInterruptedSkillInitTasks(context.Background())

	var updated model.SkillInstallation
	model.DB(context.Background()).First(&updated, rec.ID)
	if updated.InstallStatus != model.SkillInstallFailed {
		t.Errorf("期望 install_status=%d(Failed), 实际=%d", model.SkillInstallFailed, updated.InstallStatus)
	}
	if updated.ErrorMessage != "服务重启中断" {
		t.Errorf("期望 error_message='服务重启中断', 实际='%s'", updated.ErrorMessage)
	}
}

// TestRecoverInterruptedSkillInitTasks_SuccessNotAffected 验证已成功的记录不受影响。
func TestRecoverInterruptedSkillInitTasks_SuccessNotAffected(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	records := []model.SkillInstallation{
		{InstanceID: 3, Name: "ok", Slug: "ok-skill", Version: "1.0.0", InstallStatus: model.SkillInstallSuccess},
		{InstanceID: 3, Name: "fail", Slug: "fail-skill", Version: "1.0.0", InstallStatus: model.SkillInstallFailed, ErrorMessage: "原始错误"},
		{InstanceID: 3, Name: "cancel", Slug: "cancel-skill", Version: "1.0.0", InstallStatus: model.SkillInstallCancelled},
		{InstanceID: 3, Name: "pending", Slug: "pending-skill", Version: "1.0.0", InstallStatus: model.SkillInstallNone},
	}
	for i := range records {
		if err := model.DB(context.Background()).Create(&records[i]).Error; err != nil {
			t.Fatalf("create record: %v", err)
		}
	}

	recoverInterruptedSkillInitTasks(context.Background())

	// Success 不受影响
	var success model.SkillInstallation
	model.DB(context.Background()).Where("slug = ?", "ok-skill").First(&success)
	if success.InstallStatus != model.SkillInstallSuccess {
		t.Errorf("ok-skill 应保持 Success, 实际=%d", success.InstallStatus)
	}

	// 已 Failed 的保持 Failed（error_message 会被覆盖因为满足 IN 条件...不，它不满足）
	var failed model.SkillInstallation
	model.DB(context.Background()).Where("slug = ?", "fail-skill").First(&failed)
	if failed.InstallStatus != model.SkillInstallFailed {
		t.Errorf("fail-skill 应保持 Failed, 实际=%d", failed.InstallStatus)
	}
	if failed.ErrorMessage != "原始错误" {
		t.Errorf("fail-skill error_message 不应变, 实际='%s'", failed.ErrorMessage)
	}

	// Cancelled 不受影响
	var cancelled model.SkillInstallation
	model.DB(context.Background()).Where("slug = ?", "cancel-skill").First(&cancelled)
	if cancelled.InstallStatus != model.SkillInstallCancelled {
		t.Errorf("cancel-skill 应保持 Cancelled, 实际=%d", cancelled.InstallStatus)
	}

	// None 被恢复为 Failed
	var pending model.SkillInstallation
	model.DB(context.Background()).Where("slug = ?", "pending-skill").First(&pending)
	if pending.InstallStatus != model.SkillInstallFailed {
		t.Errorf("pending-skill 应变为 Failed, 实际=%d", pending.InstallStatus)
	}
}

// TestRecoverInterruptedSkillInitTasks_MixedStatuses 验证 None + Installing 混合状态全部变为 Failed。
func TestRecoverInterruptedSkillInitTasks_MixedStatuses(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	records := []model.SkillInstallation{
		{InstanceID: 4, Name: "a", Slug: "a", Version: "1.0", InstallStatus: model.SkillInstallNone},
		{InstanceID: 4, Name: "b", Slug: "b", Version: "1.0", InstallStatus: model.SkillInstalling},
		{InstanceID: 5, Name: "c", Slug: "c", Version: "1.0", InstallStatus: model.SkillInstalling},
	}
	for i := range records {
		if err := model.DB(context.Background()).Create(&records[i]).Error; err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	recoverInterruptedSkillInitTasks(context.Background())

	var count int64
	model.DB(context.Background()).Model(&model.SkillInstallation{}).
		Where("install_status = ?", model.SkillInstallFailed).Count(&count)
	if count != 3 {
		t.Errorf("期望 3 条 Failed 记录, 实际=%d", count)
	}

	// 确认无 None 和 Installing 残留
	var remaining int64
	model.DB(context.Background()).Model(&model.SkillInstallation{}).
		Where("install_status IN ?", []int{model.SkillInstallNone, model.SkillInstalling}).Count(&remaining)
	if remaining != 0 {
		t.Errorf("不应有 None/Installing 残留, 实际=%d", remaining)
	}
}

// ─── recoverInterruptedPluginInitTasks 测试 ──────────────────────────────

// TestRecoverInterruptedPluginInitTasks_NoRecords 验证无中间状态记录时函数正常返回。
func TestRecoverInterruptedPluginInitTasks_NoRecords(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	recoverInterruptedPluginInitTasks(context.Background())
}

// TestRecoverInterruptedPluginInitTasks_InstallingToFailed 验证 Installing 状态被标记为 Failed。
func TestRecoverInterruptedPluginInitTasks_InstallingToFailed(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	records := []model.PluginInstallation{
		{InstanceID: 10, Name: "p1", Slug: "plugin-1", Version: "1.0.0", InstallStatus: model.PluginInstalling, InstallMode: "npm"},
		{InstanceID: 10, Name: "p2", Slug: "plugin-2", Version: "1.0.0", InstallStatus: model.PluginInstalling, InstallMode: "smh"},
	}
	for i := range records {
		if err := model.DB(context.Background()).Create(&records[i]).Error; err != nil {
			t.Fatalf("create record: %v", err)
		}
	}

	recoverInterruptedPluginInitTasks(context.Background())

	var installations []model.PluginInstallation
	model.DB(context.Background()).Where("instance_id = ?", 10).Find(&installations)
	for _, inst := range installations {
		if inst.InstallStatus != model.PluginInstallFailed {
			t.Errorf("slug=%s: 期望 install_status=%d(Failed), 实际=%d", inst.Slug, model.PluginInstallFailed, inst.InstallStatus)
		}
		if inst.ErrorMessage != "服务重启中断" {
			t.Errorf("slug=%s: 期望 error_message='服务重启中断', 实际='%s'", inst.Slug, inst.ErrorMessage)
		}
	}
}

// TestRecoverInterruptedPluginInitTasks_NoneToFailed 验证 None 状态被标记为 Failed。
func TestRecoverInterruptedPluginInitTasks_NoneToFailed(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	rec := model.PluginInstallation{
		InstanceID: 11, Name: "p3", Slug: "plugin-3", Version: "1.0.0",
		InstallStatus: model.PluginInstallNone, InstallMode: "npm",
	}
	if err := model.DB(context.Background()).Create(&rec).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	recoverInterruptedPluginInitTasks(context.Background())

	var updated model.PluginInstallation
	model.DB(context.Background()).First(&updated, rec.ID)
	if updated.InstallStatus != model.PluginInstallFailed {
		t.Errorf("期望 install_status=%d(Failed), 实际=%d", model.PluginInstallFailed, updated.InstallStatus)
	}
	if updated.ErrorMessage != "服务重启中断" {
		t.Errorf("期望 error_message='服务重启中断', 实际='%s'", updated.ErrorMessage)
	}
}

// TestRecoverInterruptedPluginInitTasks_SuccessNotAffected 验证已成功/已取消的记录不受影响。
func TestRecoverInterruptedPluginInitTasks_SuccessNotAffected(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	records := []model.PluginInstallation{
		{InstanceID: 12, Name: "ok", Slug: "ok-plugin", Version: "1.0.0", InstallStatus: model.PluginInstallSuccess, InstallMode: "npm"},
		{InstanceID: 12, Name: "cancel", Slug: "cancel-plugin", Version: "1.0.0", InstallStatus: model.PluginInstallCancelled, InstallMode: "npm"},
		{InstanceID: 12, Name: "pending", Slug: "pending-plugin", Version: "1.0.0", InstallStatus: model.PluginInstallNone, InstallMode: "npm"},
	}
	for i := range records {
		if err := model.DB(context.Background()).Create(&records[i]).Error; err != nil {
			t.Fatalf("create record: %v", err)
		}
	}

	recoverInterruptedPluginInitTasks(context.Background())

	var success model.PluginInstallation
	model.DB(context.Background()).Where("slug = ?", "ok-plugin").First(&success)
	if success.InstallStatus != model.PluginInstallSuccess {
		t.Errorf("ok-plugin 应保持 Success, 实际=%d", success.InstallStatus)
	}

	var cancelled model.PluginInstallation
	model.DB(context.Background()).Where("slug = ?", "cancel-plugin").First(&cancelled)
	if cancelled.InstallStatus != model.PluginInstallCancelled {
		t.Errorf("cancel-plugin 应保持 Cancelled, 实际=%d", cancelled.InstallStatus)
	}

	var pending model.PluginInstallation
	model.DB(context.Background()).Where("slug = ?", "pending-plugin").First(&pending)
	if pending.InstallStatus != model.PluginInstallFailed {
		t.Errorf("pending-plugin 应变为 Failed, 实际=%d", pending.InstallStatus)
	}
}

// TestRecoverInterruptedPluginInitTasks_DBError 验证 DB 错误时不 panic。
func TestRecoverInterruptedPluginInitTasks_DBError(t *testing.T) {
	cleanup := setupBrokenDB(t)
	defer cleanup()

	recoverInterruptedPluginInitTasks(context.Background())
}

// TestRecoverInterruptedSkillInitTasks_DBError 验证 DB 错误时不 panic。
func TestRecoverInterruptedSkillInitTasks_DBError(t *testing.T) {
	cleanup := setupBrokenDB(t)
	defer cleanup()

	recoverInterruptedSkillInitTasks(context.Background())
}

// ─── recoverInterruptedUpgradeAndMigrate 测试 ───────────────────────────────

// setupRecoverInstanceTestDB 创建带 User+Instance 表的临时 SQLite 数据库。
func setupRecoverInstanceTestDB(t *testing.T) func() {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "hatchery_recover_instance_test_*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()

	dsn := tmpFile.Name() + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("open test db: %v", err)
	}

	if err := testDB.AutoMigrate(&model.User{}, &model.Instance{}); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("auto migrate: %v", err)
	}

	// 确保 Instance 生命周期管理字段存在（SQLite AutoMigrate 可能遗漏）
	for _, col := range []string{"current_operation", "current_operation_state", "current_operation_updated_at"} {
		var count int
		testDB.Raw("SELECT COUNT(*) FROM pragma_table_info('instances') WHERE name = ?", col).Scan(&count)
		if count == 0 {
			switch col {
			case "current_operation":
				testDB.Exec("ALTER TABLE instances ADD COLUMN current_operation varchar(32) DEFAULT ''")
			case "current_operation_state":
				testDB.Exec("ALTER TABLE instances ADD COLUMN current_operation_state varchar(32) DEFAULT ''")
			case "current_operation_updated_at":
				testDB.Exec("ALTER TABLE instances ADD COLUMN current_operation_updated_at datetime DEFAULT null")
			}
		}
	}

	// 创建测试用户（Instance 依赖外键）
	user := model.User{Username: "test", Password: "x", Role: "user"}
	if err := testDB.Create(&user).Error; err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("create test user: %v", err)
	}

	origDB := model.UseDBForTest(testDB)
	return func() {
		origDB()
		sqlDB, _ := testDB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
	}
}

// createTestRecoveryInstance 在测试 DB 中创建一个 Instance 记录。
func createTestRecoveryInstance(t *testing.T, userID uint, op string, state string) *model.Instance {
	t.Helper()
	now := time.Now()
	inst := &model.Instance{
		Name:                      "test-inst",
		UserID:                    userID,
		CurrentOperation:          op,
		CurrentOperationState:     state,
		CurrentOperationUpdatedAt: &now,
	}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("创建测试实例失败: %v", err)
	}
	return inst
}

// TestRecoverInterruptedUpgradeAndMigrate_NoProcessingInstances 验证无 processing 实例时正常返回。
func TestRecoverInterruptedUpgradeAndMigrate_NoProcessingInstances(t *testing.T) {
	cleanup := setupRecoverInstanceTestDB(t)
	defer cleanup()

	// 不创建任何 processing 实例，函数不应报错
	recoverInterruptedUpgradeAndMigrate(context.Background())
}

// TestRecoverInterruptedUpgradeAndMigrate_UpgradeProcessing 验证 upgrade processing 被恢复为 failed。
func TestRecoverInterruptedUpgradeAndMigrate_UpgradeProcessing(t *testing.T) {
	cleanup := setupRecoverInstanceTestDB(t)
	defer cleanup()

	createTestRecoveryInstance(t, 1, model.OpUpgrade, model.OpStateProcessing)

	recoverInterruptedUpgradeAndMigrate(context.Background())

	var instance model.Instance
	if err := model.DB(context.Background()).First(&instance).Error; err != nil {
		t.Fatalf("查询实例失败: %v", err)
	}
	if instance.CurrentOperationState != model.OpStateFailed {
		t.Errorf("期望 state=failed，实际=%s", instance.CurrentOperationState)
	}
	if instance.CurrentOperation != model.OpUpgrade {
		t.Errorf("期望 operation=upgrade 不变，实际=%s", instance.CurrentOperation)
	}
	if instance.CurrentOperationUpdatedAt == nil {
		t.Error("期望 CurrentOperationUpdatedAt 被更新，实际为 nil")
	}
}

// TestRecoverInterruptedUpgradeAndMigrate_MigrateProcessing 验证 migrate processing 被恢复为 failed。
func TestRecoverInterruptedUpgradeAndMigrate_MigrateProcessing(t *testing.T) {
	cleanup := setupRecoverInstanceTestDB(t)
	defer cleanup()

	createTestRecoveryInstance(t, 1, model.OpMigrate, model.OpStateProcessing)

	recoverInterruptedUpgradeAndMigrate(context.Background())

	var instance model.Instance
	if err := model.DB(context.Background()).First(&instance).Error; err != nil {
		t.Fatalf("查询实例失败: %v", err)
	}
	if instance.CurrentOperationState != model.OpStateFailed {
		t.Errorf("期望 state=failed，实际=%s", instance.CurrentOperationState)
	}
}

// TestRecoverInterruptedUpgradeAndMigrate_OtherOpNotAffected 验证非 upgrade/migrate 的 processing 不受影响。
func TestRecoverInterruptedUpgradeAndMigrate_OtherOpNotAffected(t *testing.T) {
	cleanup := setupRecoverInstanceTestDB(t)
	defer cleanup()

	createTestRecoveryInstance(t, 1, model.OpCreate, model.OpStateProcessing)

	recoverInterruptedUpgradeAndMigrate(context.Background())

	var instance model.Instance
	if err := model.DB(context.Background()).First(&instance).Error; err != nil {
		t.Fatalf("查询实例失败: %v", err)
	}
	if instance.CurrentOperationState != model.OpStateProcessing {
		t.Errorf("期望 state=processing（不受影响），实际=%s", instance.CurrentOperationState)
	}
}

// TestRecoverInterruptedUpgradeAndMigrate_UpgradeFailedNotAffected 验证 upgrade failed 不受影响。
func TestRecoverInterruptedUpgradeAndMigrate_UpgradeFailedNotAffected(t *testing.T) {
	cleanup := setupRecoverInstanceTestDB(t)
	defer cleanup()

	createTestRecoveryInstance(t, 1, model.OpUpgrade, model.OpStateFailed)

	recoverInterruptedUpgradeAndMigrate(context.Background())

	var instance model.Instance
	if err := model.DB(context.Background()).First(&instance).Error; err != nil {
		t.Fatalf("查询实例失败: %v", err)
	}
	if instance.CurrentOperationState != model.OpStateFailed {
		t.Errorf("期望 state=failed（不受影响），实际=%s", instance.CurrentOperationState)
	}
}

// TestRecoverInterruptedUpgradeAndMigrate_UpgradeSuccessNotAffected 验证 upgrade success 不受影响。
func TestRecoverInterruptedUpgradeAndMigrate_UpgradeSuccessNotAffected(t *testing.T) {
	cleanup := setupRecoverInstanceTestDB(t)
	defer cleanup()

	createTestRecoveryInstance(t, 1, model.OpUpgrade, model.OpStateSuccess)

	recoverInterruptedUpgradeAndMigrate(context.Background())

	var instance model.Instance
	if err := model.DB(context.Background()).First(&instance).Error; err != nil {
		t.Fatalf("查询实例失败: %v", err)
	}
	if instance.CurrentOperationState != model.OpStateSuccess {
		t.Errorf("期望 state=success（不受影响），实际=%s", instance.CurrentOperationState)
	}
}

// TestRecoverInterruptedUpgradeAndMigrate_MixedInstances 验证混合场景：upgrade+migrate processing 都被恢复，其他不受影响。
func TestRecoverInterruptedUpgradeAndMigrate_MixedInstances(t *testing.T) {
	cleanup := setupRecoverInstanceTestDB(t)
	defer cleanup()

	// upgrade processing → 应变为 failed
	createTestRecoveryInstance(t, 1, model.OpUpgrade, model.OpStateProcessing)
	// migrate processing → 应变为 failed
	createTestRecoveryInstance(t, 1, model.OpMigrate, model.OpStateProcessing)
	// reboot failed → 不受影响
	createTestRecoveryInstance(t, 1, model.OpReboot, model.OpStateFailed)
	// create processing → 不受影响（非 upgrade/migrate）
	createTestRecoveryInstance(t, 1, model.OpCreate, model.OpStateProcessing)

	recoverInterruptedUpgradeAndMigrate(context.Background())

	var instances []model.Instance
	if err := model.DB(context.Background()).Find(&instances).Error; err != nil {
		t.Fatalf("查询实例失败: %v", err)
	}

	if len(instances) != 4 {
		t.Fatalf("期望 4 个实例，实际=%d", len(instances))
	}

	for _, inst := range instances {
		switch inst.CurrentOperation {
		case model.OpUpgrade, model.OpMigrate:
			if inst.CurrentOperationState != model.OpStateFailed {
				t.Errorf("实例 %d (%s): 期望 state=failed，实际=%s",
					inst.ID, inst.CurrentOperation, inst.CurrentOperationState)
			}
		case model.OpReboot:
			if inst.CurrentOperationState != model.OpStateFailed {
				t.Errorf("实例 %d (reboot): 期望 state=failed 不变，实际=%s",
					inst.ID, inst.CurrentOperationState)
			}
		case model.OpCreate:
			if inst.CurrentOperationState != model.OpStateProcessing {
				t.Errorf("实例 %d (create): 期望 state=processing 不变，实际=%s",
					inst.ID, inst.CurrentOperationState)
			}
		}
	}
}
