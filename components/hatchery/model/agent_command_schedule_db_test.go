package model

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupScheduleDB 内存 SQLite + 迁移 schedule 两表，返回 restore。
func setupScheduleDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetConnMaxIdleTime(0)
	sqlDB.SetConnMaxLifetime(0)
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&AgentCommandSchedule{}, &AgentCommandScheduleRecord{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return UseDBForTest(db)
}

// createSched 创建一行 schedule。gorm 对带 default:true 的 bool 零值(false) 不仅会用 DB
// default，还会把 default 值回写进 struct 字段；故先保存原始意图，建后用原生 SQL 强制回写。
func createSched(t *testing.T, ctx context.Context, s *AgentCommandSchedule) {
	t.Helper()
	enabled, isRunning := s.Enabled, s.IsRunning
	if s.Slug == "" {
		s.Slug = GenerateScheduleSlug()
	}
	if err := DB(ctx).Create(s).Error; err != nil {
		t.Fatalf("create %s: %v", s.Name, err)
	}
	if err := DB(ctx).Exec(
		"UPDATE agent_command_schedules SET enabled = ?, is_running = ? WHERE id = ?",
		b2i(enabled), b2i(isRunning), s.ID,
	).Error; err != nil {
		t.Fatalf("force enabled %s: %v", s.Name, err)
	}
	s.Enabled, s.IsRunning = enabled, isRunning
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestFindDueSchedules(t *testing.T) {
	defer setupScheduleDB(t)()
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	createSched(t, ctx, &AgentCommandSchedule{Name: "due", ScheduleExpr: "rate(d, at=09:00)", Enabled: true, NextRunAt: &past})
	createSched(t, ctx, &AgentCommandSchedule{Name: "notdue", ScheduleExpr: "rate(d, at=09:00)", Enabled: true, NextRunAt: &future})
	createSched(t, ctx, &AgentCommandSchedule{Name: "disabled", ScheduleExpr: "rate(d, at=09:00)", Enabled: false, NextRunAt: &past})

	rows, err := FindDueSchedules(ctx, now, 0)
	if err != nil {
		t.Fatalf("FindDueSchedules: %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "due" {
		t.Fatalf("expected only 'due', got %+v", rows)
	}
}

func TestClaimScheduleRun(t *testing.T) {
	defer setupScheduleDB(t)()
	ctx := context.Background()
	t0 := time.Now().Truncate(time.Second)
	t1 := t0.Add(24 * time.Hour)

	s := &AgentCommandSchedule{Name: "c", ScheduleExpr: "rate(d, at=09:00)", Enabled: true, NextRunAt: &t0}
	createSched(t, ctx, s)

	// 首次抢占成功，next_run_at 推进到 t1
	ok, err := ClaimScheduleRun(ctx, s.ID, t0, &t1)
	if err != nil || !ok {
		t.Fatalf("first claim = (%v,%v), want (true,nil)", ok, err)
	}
	// 用旧 expected 再抢 → 失败（已变）
	ok, _ = ClaimScheduleRun(ctx, s.ID, t0, &t1)
	if ok {
		t.Fatalf("second claim with stale expected should fail")
	}

	// once：next=nil 同时停用
	ok, err = ClaimScheduleRun(ctx, s.ID, t1, nil)
	if err != nil || !ok {
		t.Fatalf("once claim = (%v,%v), want (true,nil)", ok, err)
	}
	var got AgentCommandSchedule
	_ = DB(ctx).First(&got, s.ID).Error
	if got.Enabled {
		t.Fatalf("once claim should disable schedule")
	}
	if got.NextRunAt != nil {
		t.Fatalf("once claim should null next_run_at, got %v", got.NextRunAt)
	}
}

func TestMarkScheduleRunResult_FirstRunAtOnlyOnce(t *testing.T) {
	defer setupScheduleDB(t)()
	ctx := context.Background()
	s := &AgentCommandSchedule{Name: "m", ScheduleExpr: "rate(d, at=09:00)", Enabled: true}
	createSched(t, ctx, s)

	ran1 := time.Now().Truncate(time.Second)
	if err := MarkScheduleRunResult(ctx, s.ID, ran1, "slug1", "", true); err != nil {
		t.Fatalf("mark1: %v", err)
	}
	var after1 AgentCommandSchedule
	_ = DB(ctx).First(&after1, s.ID).Error
	if after1.FirstRunAt == nil || !after1.FirstRunAt.Equal(ran1) {
		t.Fatalf("first_run_at = %v, want %v", after1.FirstRunAt, ran1)
	}
	if !after1.IsRunning || after1.LastDispatchSlug != "slug1" {
		t.Fatalf("unexpected after1: %+v", after1)
	}

	ran2 := ran1.Add(time.Hour)
	if err := MarkScheduleRunResult(ctx, s.ID, ran2, "slug2", "boom", false); err != nil {
		t.Fatalf("mark2: %v", err)
	}
	var after2 AgentCommandSchedule
	_ = DB(ctx).First(&after2, s.ID).Error
	// first_run_at 保持首次值不变
	if after2.FirstRunAt == nil || !after2.FirstRunAt.Equal(ran1) {
		t.Fatalf("first_run_at changed: %v, want %v", after2.FirstRunAt, ran1)
	}
	if after2.LastRunAt == nil || !after2.LastRunAt.Equal(ran2) {
		t.Fatalf("last_run_at = %v, want %v", after2.LastRunAt, ran2)
	}
	if after2.IsRunning || after2.LastError != "boom" {
		t.Fatalf("unexpected after2: %+v", after2)
	}
}

func TestMarkScheduleSkipped(t *testing.T) {
	defer setupScheduleDB(t)()
	ctx := context.Background()
	s := &AgentCommandSchedule{Name: "sk", ScheduleExpr: "rate(d, at=09:00)", Enabled: true, LastDispatchSlug: "keep"}
	createSched(t, ctx, s)

	if err := MarkScheduleSkipped(ctx, s.ID, "skip reason"); err != nil {
		t.Fatalf("MarkScheduleSkipped: %v", err)
	}
	var got AgentCommandSchedule
	_ = DB(ctx).First(&got, s.ID).Error
	if got.LastError != "skip reason" {
		t.Fatalf("last_error = %q, want 'skip reason'", got.LastError)
	}
	if got.LastDispatchSlug != "keep" {
		t.Fatalf("last_dispatch_slug should be unchanged, got %q", got.LastDispatchSlug)
	}
}

func TestFindAndClearScheduleRunning(t *testing.T) {
	defer setupScheduleDB(t)()
	ctx := context.Background()
	r1 := &AgentCommandSchedule{Name: "r1", ScheduleExpr: "rate(d, at=09:00)", Enabled: true, IsRunning: true}
	r2 := &AgentCommandSchedule{Name: "r2", ScheduleExpr: "rate(d, at=09:00)", Enabled: true, IsRunning: true}
	idle := &AgentCommandSchedule{Name: "idle", ScheduleExpr: "rate(d, at=09:00)", Enabled: true, IsRunning: false}
	createSched(t, ctx, r1)
	createSched(t, ctx, r2)
	createSched(t, ctx, idle)

	running, err := FindRunningSchedules(ctx)
	if err != nil {
		t.Fatalf("FindRunningSchedules: %v", err)
	}
	if len(running) != 2 {
		t.Fatalf("expected 2 running, got %d", len(running))
	}

	if err := ClearScheduleRunning(ctx, []uint{r1.ID, r2.ID}); err != nil {
		t.Fatalf("ClearScheduleRunning: %v", err)
	}
	running, _ = FindRunningSchedules(ctx)
	if len(running) != 0 {
		t.Fatalf("expected 0 running after clear, got %d", len(running))
	}
	// 空 ids 应为 no-op
	if err := ClearScheduleRunning(ctx, nil); err != nil {
		t.Fatalf("ClearScheduleRunning(nil): %v", err)
	}
}

// TestScheduleStatusCondition_MatchesStatus 验证 SQL 下推与内存 Status() 完全等价。
func TestScheduleStatusCondition_MatchesStatus(t *testing.T) {
	defer setupScheduleDB(t)()
	ctx := context.Background()
	ran := time.Now().Truncate(time.Second)
	future := ran.Add(time.Hour)
	once := "once(2026-06-30 15:00)"
	every := "every(d, at=09:00)"
	interval := "interval(1m, begin=2026-06-30 15:00)"

	rows := []*AgentCommandSchedule{
		{Name: "completed", ScheduleExpr: once, Enabled: false, LastRunAt: &ran},
		{Name: "completed-enabled", ScheduleExpr: once, Enabled: true, LastRunAt: &ran},
		{Name: "completed-running", ScheduleExpr: once, Enabled: false, LastRunAt: &ran, IsRunning: true},
		{Name: "running", ScheduleExpr: every, Enabled: true, LastRunAt: &ran, IsRunning: true},
		{Name: "paused-rate", ScheduleExpr: every, Enabled: false, LastRunAt: &ran},
		{Name: "paused-once", ScheduleExpr: once, Enabled: false},
		{Name: "pending", ScheduleExpr: every, Enabled: true},
		{Name: "pending-once", ScheduleExpr: once, Enabled: true},
		{Name: "waiting", ScheduleExpr: every, Enabled: true, LastRunAt: &ran},
		// interval 终态（已过 end → next_run_at 置空）：无论是否在执行均为 completed
		{Name: "interval-completed", ScheduleExpr: interval, Enabled: false, LastRunAt: &ran},
		{Name: "interval-completed-running", ScheduleExpr: interval, Enabled: false, LastRunAt: &ran, IsRunning: true},
		// interval 非终态（next_run_at 有值）
		{Name: "interval-running", ScheduleExpr: interval, Enabled: true, NextRunAt: &future, LastRunAt: &ran, IsRunning: true},
		{Name: "interval-paused", ScheduleExpr: interval, Enabled: false, NextRunAt: &future, LastRunAt: &ran},
		{Name: "interval-pending", ScheduleExpr: interval, Enabled: true, NextRunAt: &future},
		{Name: "interval-waiting", ScheduleExpr: interval, Enabled: true, NextRunAt: &future, LastRunAt: &ran},
	}
	for _, s := range rows {
		createSched(t, ctx, s)
	}

	var all []AgentCommandSchedule
	if err := DB(ctx).Find(&all).Error; err != nil {
		t.Fatalf("find all: %v", err)
	}

	for _, status := range []string{
		ScheduleStatusCompleted, ScheduleStatusRunning, ScheduleStatusPaused,
		ScheduleStatusPending, ScheduleStatusWaiting,
	} {
		// 内存口径
		var wantIDs []uint
		for i := range all {
			if all[i].Status() == status {
				wantIDs = append(wantIDs, all[i].ID)
			}
		}
		// SQL 下推口径
		cond, args, ok := ScheduleStatusCondition(status)
		if !ok {
			t.Fatalf("ScheduleStatusCondition(%q) ok=false", status)
		}
		var sqlRows []AgentCommandSchedule
		if err := DB(ctx).Where(cond, args...).Find(&sqlRows).Error; err != nil {
			t.Fatalf("query status %q: %v", status, err)
		}
		var gotIDs []uint
		for i := range sqlRows {
			gotIDs = append(gotIDs, sqlRows[i].ID)
		}
		sort.Slice(wantIDs, func(i, j int) bool { return wantIDs[i] < wantIDs[j] })
		sort.Slice(gotIDs, func(i, j int) bool { return gotIDs[i] < gotIDs[j] })
		if !equalUintSlice(wantIDs, gotIDs) {
			t.Errorf("status %q mismatch: SQL=%v, Status()=%v", status, gotIDs, wantIDs)
		}
	}

	if _, _, ok := ScheduleStatusCondition("bogus"); ok {
		t.Errorf("ScheduleStatusCondition(bogus) should return ok=false")
	}
}

func equalUintSlice(a, b []uint) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestScheduleHelpers(t *testing.T) {
	defer setupScheduleDB(t)()
	ctx := context.Background()
	s := &AgentCommandSchedule{Name: "h", ScheduleExpr: "rate(d, at=09:00)", Enabled: true}
	if err := s.SetInstanceIDs([]uint{7, 8}); err != nil {
		t.Fatalf("SetInstanceIDs: %v", err)
	}
	if err := s.SetParamValues(map[string]string{"k": "v"}); err != nil {
		t.Fatalf("SetParamValues: %v", err)
	}
	createSched(t, ctx, s)

	n, err := CountAgentCommandSchedules(ctx)
	if err != nil || n != 1 {
		t.Fatalf("count = %d, err=%v, want 1", n, err)
	}

	got, err := FindScheduleByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("FindScheduleByID: %v", err)
	}
	if got.ParamValuesMap()["k"] != "v" {
		t.Fatalf("param values = %v, want k=v", got.ParamValuesMap())
	}
	if ids := got.InstanceIDsList(); len(ids) != 2 || ids[0] != 7 {
		t.Fatalf("instance ids = %v, want [7 8]", ids)
	}

	if _, err := FindScheduleByID(ctx, 99999); err == nil {
		t.Fatalf("expected ErrScheduleNotFound for missing id")
	}
}
