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
	"gorm.io/gorm/logger"
)

// setupScheduleTestDB 创建临时 SQLite 库并替换 model.DB，返回 cleanup。
// 仅迁移 runner / reconcile 编排所需的三张表——派发链路被 triggerScheduleDispatch seam 替换，
// 故无需 command / instance / dispatch-task 等下游表。
func setupScheduleTestDB(t *testing.T) func() {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "hatchery_schedule_test_*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()

	dsn := tmpFile.Name() + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("open test db: %v", err)
	}
	if err := testDB.AutoMigrate(
		&model.AgentCommandSchedule{},
		&model.AgentCommandScheduleRecord{},
		&model.AgentCommandDispatch{},
	); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("auto migrate: %v", err)
	}

	origDB := model.UseDBForTest(testDB)
	return func() {
		origDB()
		if sqlDB, _ := testDB.DB(); sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
	}
}

// seedSchedule 写入一条 schedule。enabled / is_running 走 gorm 默认有零值回写问题，
// 这里用一条 UPDATE 强制落库，确保种子状态准确。
func seedSchedule(t *testing.T, ctx context.Context, s *model.AgentCommandSchedule) {
	t.Helper()
	enabled, running := s.Enabled, s.IsRunning
	if s.Slug == "" {
		s.Slug = model.GenerateScheduleSlug()
	}
	if err := model.DB(ctx).Create(s).Error; err != nil {
		t.Fatalf("seed schedule: %v", err)
	}
	if err := model.DB(ctx).Exec(
		"UPDATE agent_command_schedules SET enabled = ?, is_running = ? WHERE id = ?",
		b2i(enabled), b2i(running), s.ID,
	).Error; err != nil {
		t.Fatalf("seed schedule state: %v", err)
	}
	s.Enabled, s.IsRunning = enabled, running
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// withStubDispatch 临时替换 triggerScheduleDispatch seam，返回恢复函数。
func withStubDispatch(fn func(ctx context.Context, s *model.AgentCommandSchedule) (string, error)) func() {
	prev := triggerScheduleDispatch
	triggerScheduleDispatch = fn
	return func() { triggerScheduleDispatch = prev }
}

// ============================================================================
// runner：到期 → 通过 seam 触发，推进 next_run_at + 记录
// ============================================================================

func TestRunAgentCommandSchedule_TriggersDueSchedule(t *testing.T) {
	defer setupScheduleTestDB(t)()
	ctx := context.Background()

	var gotID uint
	calls := 0
	defer withStubDispatch(func(_ context.Context, s *model.AgentCommandSchedule) (string, error) {
		calls++
		gotID = s.ID
		return "disp-fake", nil
	})()

	past := time.Now().Add(-time.Minute)
	s := &model.AgentCommandSchedule{
		Name: "due", ScheduleExpr: "every(d, at=09:00)", Enabled: true,
		CommandID: 1, NextRunAt: &past, CreatedByUserID: 1,
	}
	seedSchedule(t, ctx, s)

	runAgentCommandSchedule(ctx)

	if calls != 1 || gotID != s.ID {
		t.Fatalf("dispatch seam calls = %d (gotID=%d), want 1 for schedule %d", calls, gotID, s.ID)
	}
	var after model.AgentCommandSchedule
	_ = model.DB(ctx).First(&after, s.ID).Error
	if !after.IsRunning {
		t.Errorf("is_running = false, want true")
	}
	if after.LastDispatchSlug != "disp-fake" {
		t.Errorf("last_dispatch_slug = %q, want disp-fake", after.LastDispatchSlug)
	}
	if after.NextRunAt == nil || !after.NextRunAt.After(time.Now()) {
		t.Errorf("next_run_at not advanced to future: %v", after.NextRunAt)
	}
	if _, total, _ := model.ListScheduleRecords(ctx, s.ID, 1, 10); total != 1 {
		t.Errorf("schedule records total = %d, want 1", total)
	}
}

// ============================================================================
// runner：串行保护——上一轮 dispatch 未终态 → 跳过本次（不触发，但推进 next）
// ============================================================================

func TestRunAgentCommandSchedule_SerialProtectionSkips(t *testing.T) {
	defer setupScheduleTestDB(t)()
	ctx := context.Background()

	// 上一轮 dispatch 仍在执行中（非终态）
	if err := model.DB(ctx).Create(&model.AgentCommandDispatch{
		Slug: "live", Status: model.AgentDispatchStatusInProgress,
	}).Error; err != nil {
		t.Fatalf("seed dispatch: %v", err)
	}

	defer withStubDispatch(func(_ context.Context, _ *model.AgentCommandSchedule) (string, error) {
		t.Fatalf("dispatch seam should NOT be called when previous run is not terminal")
		return "", nil
	})()

	past := time.Now().Add(-time.Minute)
	s := &model.AgentCommandSchedule{
		Name: "busy", ScheduleExpr: "every(d, at=09:00)", Enabled: true,
		CommandID: 1, NextRunAt: &past, LastDispatchSlug: "live", CreatedByUserID: 1,
	}
	seedSchedule(t, ctx, s)

	runAgentCommandSchedule(ctx)

	var after model.AgentCommandSchedule
	_ = model.DB(ctx).First(&after, s.ID).Error
	// next_run_at 已被 CAS 推进（不补跑），但本次未触发
	if after.NextRunAt == nil || !after.NextRunAt.After(time.Now()) {
		t.Errorf("next_run_at should be advanced even when skipped: %v", after.NextRunAt)
	}
	if after.LastError == "" {
		t.Errorf("expected last_error to note the skip, got empty")
	}
	if _, total, _ := model.ListScheduleRecords(ctx, s.ID, 1, 10); total != 0 {
		t.Errorf("schedule records total = %d, want 0 (skipped)", total)
	}
}

// ============================================================================
// runner：触发失败 → 记录 last_error，不写执行记录，is_running 保持 false
// ============================================================================

func TestRunAgentCommandSchedule_DispatchErrorRecorded(t *testing.T) {
	defer setupScheduleTestDB(t)()
	ctx := context.Background()

	defer withStubDispatch(func(_ context.Context, _ *model.AgentCommandSchedule) (string, error) {
		return "", errors.New("all_targets_offline: 所有 Agent 均未运行，无法下发")
	})()

	past := time.Now().Add(-time.Minute)
	s := &model.AgentCommandSchedule{
		Name: "fail", ScheduleExpr: "every(d, at=09:00)", Enabled: true,
		CommandID: 1, NextRunAt: &past, CreatedByUserID: 1,
	}
	seedSchedule(t, ctx, s)

	runAgentCommandSchedule(ctx)

	var after model.AgentCommandSchedule
	_ = model.DB(ctx).First(&after, s.ID).Error
	if after.IsRunning {
		t.Errorf("is_running = true, want false on dispatch error")
	}
	if after.LastError == "" {
		t.Errorf("expected last_error recorded on dispatch failure")
	}
	if _, total, _ := model.ListScheduleRecords(ctx, s.ID, 1, 10); total != 0 {
		t.Errorf("schedule records total = %d, want 0 on failure", total)
	}
}

// ============================================================================
// reconcile：按 last_dispatch_slug 对应 dispatch 是否终态订正 is_running
// ============================================================================

func TestRunAgentCommandScheduleReconcile(t *testing.T) {
	defer setupScheduleTestDB(t)()
	ctx := context.Background()
	fin := time.Now()

	// 终态 / 非终态 dispatch
	if err := model.DB(ctx).Create(&model.AgentCommandDispatch{
		Slug: "term", Status: model.AgentDispatchStatusSuccess, FinishedAt: &fin,
	}).Error; err != nil {
		t.Fatalf("seed dispatch term: %v", err)
	}
	if err := model.DB(ctx).Create(&model.AgentCommandDispatch{
		Slug: "live", Status: model.AgentDispatchStatusInProgress,
	}).Error; err != nil {
		t.Fatalf("seed dispatch live: %v", err)
	}

	done := &model.AgentCommandSchedule{Name: "done", ScheduleExpr: "every(d, at=09:00)", Enabled: true, IsRunning: true, LastDispatchSlug: "term", CreatedByUserID: 1}
	live := &model.AgentCommandSchedule{Name: "live", ScheduleExpr: "every(d, at=09:00)", Enabled: true, IsRunning: true, LastDispatchSlug: "live", CreatedByUserID: 1}
	noslug := &model.AgentCommandSchedule{Name: "noslug", ScheduleExpr: "every(d, at=09:00)", Enabled: true, IsRunning: true, CreatedByUserID: 1}
	seedSchedule(t, ctx, done)
	seedSchedule(t, ctx, live)
	seedSchedule(t, ctx, noslug)

	runAgentCommandScheduleReconcile(ctx)

	assertRunning := func(id uint, want bool) {
		t.Helper()
		var s model.AgentCommandSchedule
		_ = model.DB(ctx).First(&s, id).Error
		if s.IsRunning != want {
			t.Errorf("schedule %d is_running = %v, want %v", id, s.IsRunning, want)
		}
	}
	assertRunning(done.ID, false)   // 对应 dispatch 终态 → 订正 false
	assertRunning(noslug.ID, false) // 无 slug → 订正 false
	assertRunning(live.ID, true)    // 对应 dispatch 非终态 → 保持 true
}
