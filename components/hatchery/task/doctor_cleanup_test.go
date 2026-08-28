package task

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"hatchery/controller"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// taskFnSnapshot 保存导出函数变量的快照，用于测试后恢复。
type taskFnSnapshot struct {
	getMtime          func(context.Context, string) controller.DoctorMtimeResult
	refreshSTS        func(context.Context)
	cleanupSession    func(context.Context, *model.DoctorSession)
	deleteSMHFile     func(context.Context, string) error
	startProbeChecker func(context.Context, model.DoctorSession, string, *slog.Logger)
}

func saveTaskFns() taskFnSnapshot {
	return taskFnSnapshot{
		getMtime:          controller.GetDoctorSessionMtimeFn,
		refreshSTS:        controller.RefreshDoctorSTSFn,
		cleanupSession:    controller.CleanupDoctorSessionFn,
		deleteSMHFile:     controller.DeleteSMHCommonFileFn,
		startProbeChecker: startProbeCheckerFn,
	}
}

func (s taskFnSnapshot) restore() {
	controller.GetDoctorSessionMtimeFn = s.getMtime
	controller.RefreshDoctorSTSFn = s.refreshSTS
	controller.CleanupDoctorSessionFn = s.cleanupSession
	controller.DeleteSMHCommonFileFn = s.deleteSMHFile
	startProbeCheckerFn = s.startProbeChecker
}

// ─── 辅助 ───────────────────────────────────────────────────────────────────

func initCleanupTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(
		sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Instance{},
		&model.DoctorSession{},
		&model.DoctorAuthorization{},
		&model.SiteConfig{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	origDB := model.UseDBForTest(db)

	db.Create(&model.SiteConfig{DoctorEnabled: true})
	db.Create(&model.User{
		Username: "testuser", Password: "x", Role: "user",
	})
	db.Create(&model.Instance{
		Name: "target", InstanceId: "ins-target", UserID: 1,
	})

	return func() { origDB() }
}

// ─── 测试 ───────────────────────────────────────────────────────────────────

func TestCleanupDoctorSessions_Active超时自动结束(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	doctorInstID := uint(100)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-timeout", InstanceId: "ins-doctor-timeout",
		UserID: 1, IsDoctorNode: true,
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		STSExpiredAt:     time.Now().Unix() + 7200,
	}
	model.DB(context.Background()).Create(&session)

	snap := saveTaskFns()
	defer snap.restore()

	// mtime 13 小时前 → 超过 12 小时超时
	controller.GetDoctorSessionMtimeFn =
		func(context.Context, string) controller.DoctorMtimeResult {
			return controller.DoctorMtimeResult{Mtime: time.Now().Add(-13 * time.Hour)}
		}
	controller.RefreshDoctorSTSFn = func(context.Context) {}
	controller.CleanupDoctorSessionFn =
		func(ctx context.Context, s *model.DoctorSession) {
			model.DB(context.Background()).Model(s).Update("status", model.DoctorStatusEnded)
		}

	cleanupDoctorSessions(context.Background())

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusEnding {
		t.Errorf("status = %s, want ending", updated.Status)
	}
}

func TestCleanupDoctorSessions_目标实例已删除则立即结束(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	// 创建一个已软删的目标实例
	targetInstID := uint(500)
	db := model.DB(context.Background())
	db.Create(&model.Instance{
		Model: gorm.Model{ID: targetInstID},
		Name:  "target-deleted", InstanceId: "ins-target-deleted",
		UserID: 1,
	})
	db.Delete(&model.Instance{}, targetInstID) // 软删

	// 创建龙虾医生 CVM
	doctorInstID := uint(501)
	db.Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-target-gone", InstanceId: "ins-doctor-target-gone",
		UserID: 1, IsDoctorNode: true,
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: targetInstID,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		STSExpiredAt:     time.Now().Unix() + 7200,
	}
	db.Create(&session)

	snap := saveTaskFns()
	defer snap.restore()

	// 即使 mtime 未超时，也应该被立即结束
	mtimeCalled := false
	controller.GetDoctorSessionMtimeFn =
		func(context.Context, string) controller.DoctorMtimeResult {
			mtimeCalled = true
			return controller.DoctorMtimeResult{Mtime: time.Now().Add(-1 * time.Minute)}
		}
	controller.RefreshDoctorSTSFn = func(context.Context) {}

	cleanupDoctorSessions(context.Background())

	var updated model.DoctorSession
	db.First(&updated, session.ID)
	if updated.Status != model.DoctorStatusEnding {
		t.Errorf("status = %s, want ending", updated.Status)
	}
	// 目标已删，不应再走 mtime 检查
	if mtimeCalled {
		t.Errorf("GetDoctorSessionMtimeFn 不应被调用，目标已删应提前跳出")
	}
}

func TestCleanupDoctorSessions_目标实例不存在则立即结束(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	// 不创建 target，TargetInstanceID 指向一个不存在的 ID
	doctorInstID := uint(601)
	db := model.DB(context.Background())
	db.Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-no-target", InstanceId: "ins-doctor-no-target",
		UserID: 1, IsDoctorNode: true,
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 9999, // 不存在的 ID
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		STSExpiredAt:     time.Now().Unix() + 7200,
	}
	db.Create(&session)

	snap := saveTaskFns()
	defer snap.restore()

	controller.GetDoctorSessionMtimeFn =
		func(context.Context, string) controller.DoctorMtimeResult {
			return controller.DoctorMtimeResult{Mtime: time.Now()}
		}
	controller.RefreshDoctorSTSFn = func(context.Context) {}

	cleanupDoctorSessions(context.Background())

	var updated model.DoctorSession
	db.First(&updated, session.ID)
	if updated.Status != model.DoctorStatusEnding {
		t.Errorf("status = %s, want ending", updated.Status)
	}
}

func TestCleanupDoctorSessions_Active未超时则跳过(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	doctorInstID := uint(101)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-active", InstanceId: "ins-doctor-active",
		UserID: 1, IsDoctorNode: true,
	})

	activatedAt := time.Now().Add(-13 * time.Hour)
	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		ActivatedAt:      &activatedAt,
		STSExpiredAt:     time.Now().Unix() + 7200,
	}
	model.DB(context.Background()).Create(&session)

	snap := saveTaskFns()
	defer snap.restore()

	// mtime 1 分钟前 → 未超时
	controller.GetDoctorSessionMtimeFn =
		func(context.Context, string) controller.DoctorMtimeResult {
			return controller.DoctorMtimeResult{Mtime: time.Now().Add(-1 * time.Minute)}
		}
	controller.RefreshDoctorSTSFn = func(context.Context) {}

	cleanupDoctorSessions(context.Background())

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusActive {
		t.Errorf("status = %s, want active", updated.Status)
	}
}

// TestCleanupDoctorSessions_探测持续失败但超12小时仍强制结束 覆盖
// TAPD bug 1020422209160782882：mtime 探测持续失败（如 TAT 掉线）
// 不应导致会话永久卡在 active，创建超过 12h 后必须被兜底强制结束。
func TestCleanupDoctorSessions_探测持续失败但超12小时仍强制结束(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	doctorInstID := uint(105)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-stuck", InstanceId: "ins-doctor-stuck",
		UserID: 1, IsDoctorNode: true,
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		STSExpiredAt:     time.Now().Unix() + 7200,
	}
	model.DB(context.Background()).Create(&session)
	// 会话创建时间已超过 12 小时兜底超时
	model.DB(context.Background()).Model(&session).
		Update("created_at", time.Now().Add(-13*time.Hour))

	snap := saveTaskFns()
	defer snap.restore()

	controller.GetDoctorSessionMtimeFn =
		func(context.Context, string) controller.DoctorMtimeResult {
			return controller.DoctorMtimeResult{Err: fmt.Errorf("mock TAT offline")}
		}
	controller.RefreshDoctorSTSFn = func(context.Context) {}

	// 直接测试 runProbeChecker：用极短间隔快速完成 3 次探测
	done := make(chan struct{})
	go func() {
		defer close(done)
		runProbeChecker(context.Background(), session.ID,
			"ins-doctor-stuck", slog.Default(),
			10*time.Millisecond, 3, 12*time.Hour)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runProbeChecker 超时未退出")
	}

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusEnding {
		t.Errorf("status = %s, want ending（探测持续失败但超12h应被兜底强制结束）",
			updated.Status)
	}
}

// TestCleanupDoctorSessions_DoctorInstanceID为空但超12小时仍强制结束 覆盖
// DoctorInstanceID 缺失（如激活流程异常中断）时的兜底出口。
func TestCleanupDoctorSessions_DoctorInstanceID为空但超12小时仍强制结束(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: nil,
		Status:           model.DoctorStatusActive,
	}
	model.DB(context.Background()).Create(&session)
	model.DB(context.Background()).Model(&session).
		Update("created_at", time.Now().Add(-13*time.Hour))

	snap := saveTaskFns()
	defer snap.restore()
	controller.RefreshDoctorSTSFn = func(context.Context) {}

	cleanupDoctorSessions(context.Background())

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusEnding {
		t.Errorf("status = %s, want ending（DoctorInstanceID 为空但超12h应被兜底强制结束）",
			updated.Status)
	}
}

// TestCleanupDoctorSessions_关联实例不存在但超12小时仍强制结束 覆盖
// 关联的龙虾医生 Instance 记录已不存在（如被异常删除）时的兜底出口。
func TestCleanupDoctorSessions_关联实例不存在但超12小时仍强制结束(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	missingInstID := uint(9999) // 不存在的 Instance ID
	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &missingInstID,
		Status:           model.DoctorStatusActive,
	}
	model.DB(context.Background()).Create(&session)
	model.DB(context.Background()).Model(&session).
		Update("created_at", time.Now().Add(-13*time.Hour))

	snap := saveTaskFns()
	defer snap.restore()
	controller.RefreshDoctorSTSFn = func(context.Context) {}

	cleanupDoctorSessions(context.Background())

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusEnding {
		t.Errorf("status = %s, want ending（关联实例不存在但超12h应被兜底强制结束）",
			updated.Status)
	}
}

// TestCleanupDoctorSessions_探测失败但未超12小时不应被误伤 回归测试：
// 兜底逻辑不应缩短用户可用时长，未超 12h 时探测失败仍走原有 continue 逻辑等待重试。
func TestCleanupDoctorSessions_探测失败但未超12小时不应被误伤(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	doctorInstID := uint(106)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-fresh", InstanceId: "ins-doctor-fresh",
		UserID: 1, IsDoctorNode: true,
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		STSExpiredAt:     time.Now().Unix() + 7200,
	}
	// CreatedAt 保持默认（刚创建，远未超过 12h）
	model.DB(context.Background()).Create(&session)

	snap := saveTaskFns()
	defer snap.restore()

	controller.GetDoctorSessionMtimeFn =
		func(context.Context, string) controller.DoctorMtimeResult {
			return controller.DoctorMtimeResult{Err: fmt.Errorf("transient error")}
		}
	controller.RefreshDoctorSTSFn = func(context.Context) {}
	// 阻止 startProbeChecker 泄漏后台协程（本测试直接调用 runProbeChecker 验证）
	startProbeCheckerFn = func(context.Context, model.DoctorSession, string, *slog.Logger) {}

	// 验证1：cleanupDoctorSessions 同步调用不会误杀
	cleanupDoctorSessions(context.Background())

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusActive {
		t.Errorf("status = %s, want active（未超12h时探测失败不应被兜底逻辑误伤提前结束）",
			updated.Status)
	}

	// 验证2：runProbeChecker 固定次数探测后也不会结束
	done := make(chan struct{})
	go func() {
		defer close(done)
		runProbeChecker(context.Background(), session.ID,
			"ins-doctor-fresh", slog.Default(),
			10*time.Millisecond, 3, 12*time.Hour)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runProbeChecker 超时未退出")
	}

	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusActive {
		t.Errorf("status = %s, want active（runProbeChecker 未超12h不应结束）",
			updated.Status)
	}
}

// TestRunProbeChecker_探测恢复成功则退出 验证协程在探测恢复后正常退出不结束会话。
func TestRunProbeChecker_探测恢复成功则退出(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	doctorInstID := uint(107)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-recover", InstanceId: "ins-doctor-recover",
		UserID: 1, IsDoctorNode: true,
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		STSExpiredAt:     time.Now().Unix() + 7200,
	}
	model.DB(context.Background()).Create(&session)
	model.DB(context.Background()).Model(&session).
		Update("created_at", time.Now().Add(-13*time.Hour))

	snap := saveTaskFns()
	defer snap.restore()

	// 前 2 次探测失败，第 3 次恢复
	callCount := 0
	controller.GetDoctorSessionMtimeFn =
		func(context.Context, string) controller.DoctorMtimeResult {
			callCount++
			if callCount <= 2 {
				return controller.DoctorMtimeResult{Err: fmt.Errorf("transient")}
			}
			return controller.DoctorMtimeResult{}
		}
	controller.RefreshDoctorSTSFn = func(context.Context) {}

	done := make(chan struct{})
	go func() {
		defer close(done)
		runProbeChecker(context.Background(), session.ID,
			"ins-doctor-recover", slog.Default(),
			10*time.Millisecond, 5, 12*time.Hour)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runProbeChecker 超时未退出")
	}

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusActive {
		t.Errorf("status = %s, want active（探测恢复后不应结束会话）",
			updated.Status)
	}
}

// TestRunProbeChecker_Context取消则退出 验证协程响应 context 取消信号及时退出，
// 不因 time.Sleep 阻塞而泄漏 goroutine。
func TestRunProbeChecker_Context取消则退出(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	doctorInstID := uint(108)
	model.DB(context.Background()).Create(&model.Instance{
		Model:     gorm.Model{ID: doctorInstID},
		Name:      "doctor-cancel", InstanceId: "ins-doctor-cancel",
		UserID:    1, IsDoctorNode: true,
	})

	session := model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		STSExpiredAt:     time.Now().Unix() + 7200,
	}
	model.DB(context.Background()).Create(&session)

	snap := saveTaskFns()
	defer snap.restore()

	controller.GetDoctorSessionMtimeFn =
		func(context.Context, string) controller.DoctorMtimeResult {
			return controller.DoctorMtimeResult{Err: fmt.Errorf("always fail")}
		}
	controller.RefreshDoctorSTSFn = func(context.Context) {}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		runProbeChecker(ctx, session.ID, "ins-doctor-cancel",
			slog.Default(), 10*time.Millisecond, 5, 12*time.Hour)
	}()

	// 等待至少一次探测完成后取消 context
	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runProbeChecker 应在 context 取消后退出")
	}

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusActive {
		t.Errorf("status = %s, want active（context 取消不应结束会话）",
			updated.Status)
	}
}

func TestCleanupDoctorSessions_获取失败则启动验证协程(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	doctorInstID := uint(102)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-mz", InstanceId: "ins-doctor-mz",
		UserID: 1, IsDoctorNode: true,
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		STSExpiredAt:     time.Now().Unix() + 7200,
	}
	model.DB(context.Background()).Create(&session)

	snap := saveTaskFns()
	defer snap.restore()

	controller.GetDoctorSessionMtimeFn =
		func(context.Context, string) controller.DoctorMtimeResult {
			return controller.DoctorMtimeResult{Err: fmt.Errorf("mock error")}
		}
	controller.RefreshDoctorSTSFn = func(context.Context) {}
	// mock startProbeChecker 防止后台协程泄漏，同时验证被调用
	probeStarted := false
	startProbeCheckerFn = func(_ context.Context, _ model.DoctorSession, _ string, _ *slog.Logger) {
		probeStarted = true
	}

	cleanupDoctorSessions(context.Background())

	if !probeStarted {
		t.Error("探测失败应启动验证协程")
	}

	// mtime 探测失败时启动后台验证协程继续探测，不会立即结束会话
	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusActive {
		t.Errorf("status = %s, want active（探测失败后应保持 active 等待验证协程）",
			updated.Status)
	}
}

func TestCleanupDoctorSessions_无文件超时则结束(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	doctorInstID := uint(103)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-nofile", InstanceId: "ins-doctor-nofile",
		UserID: 1, IsDoctorNode: true,
	})

	activatedAt := time.Now().Add(-13 * time.Hour)
	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		ActivatedAt:      &activatedAt,
		STSExpiredAt:     time.Now().Unix() + 7200,
	}
	// 模拟激活时间是 13 小时前，但 updated_at 最近刚被技术字段刷新。
	model.DB(context.Background()).Create(&session)
	model.DB(context.Background()).Model(&session).
		UpdateColumn("updated_at", time.Now())

	snap := saveTaskFns()
	defer snap.restore()

	// 无文件
	controller.GetDoctorSessionMtimeFn =
		func(context.Context, string) controller.DoctorMtimeResult {
			return controller.DoctorMtimeResult{NoFiles: true}
		}
	controller.RefreshDoctorSTSFn = func(context.Context) {}

	cleanupDoctorSessions(context.Background())

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusEnding {
		t.Errorf("status = %s, want ending (无文件超时应结束)",
			updated.Status)
	}
}

func TestCleanupDoctorSessions_无文件未超时则跳过(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	doctorInstID := uint(104)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-nofile2", InstanceId: "ins-doctor-nofile2",
		UserID: 1, IsDoctorNode: true,
	})

	activatedAt := time.Now().Add(-1 * time.Minute)
	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		ActivatedAt:      &activatedAt,
		STSExpiredAt:     time.Now().Unix() + 7200,
	}
	model.DB(context.Background()).Create(&session)

	snap := saveTaskFns()
	defer snap.restore()

	controller.GetDoctorSessionMtimeFn =
		func(context.Context, string) controller.DoctorMtimeResult {
			return controller.DoctorMtimeResult{NoFiles: true}
		}
	controller.RefreshDoctorSTSFn = func(context.Context) {}

	cleanupDoctorSessions(context.Background())

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusActive {
		t.Errorf("status = %s, want active (无文件未超时应跳过)",
			updated.Status)
	}
}

func TestCleanupDoctorSessions_无文件存量会话按创建时间超时(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	doctorInstID := uint(106)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-nofile-legacy", InstanceId: "ins-doctor-nofile-legacy",
		UserID: 1, IsDoctorNode: true,
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		ActivatedAt:      nil,
		STSExpiredAt:     time.Now().Unix() + 7200,
	}
	model.DB(context.Background()).Create(&session)
	model.DB(context.Background()).Model(&session).
		UpdateColumn("created_at", time.Now().Add(-13*time.Hour))
	model.DB(context.Background()).Model(&session).
		UpdateColumn("updated_at", time.Now())

	snap := saveTaskFns()
	defer snap.restore()

	controller.GetDoctorSessionMtimeFn =
		func(context.Context, string) controller.DoctorMtimeResult {
			return controller.DoctorMtimeResult{NoFiles: true}
		}
	controller.RefreshDoctorSTSFn = func(context.Context) {}

	cleanupDoctorSessions(context.Background())

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusEnding {
		t.Errorf("status = %s, want ending (存量会话应回退 created_at)",
			updated.Status)
	}
}

func TestCleanupDoctorSessions_RefreshSTS被调用(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	called := false
	snap := saveTaskFns()
	defer snap.restore()

	controller.RefreshDoctorSTSFn =
		func(context.Context) { called = true }

	cleanupDoctorSessions(context.Background())

	if !called {
		t.Error("RefreshDoctorSTS 未被调用")
	}
}

// ─── cleanupOrphanedSnapshots 测试 ──────────────────────────────────────────

func TestCleanupOrphanedSnapshots_目标实例已删除则清理快照(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	// 创建目标实例并软删除
	targetInst := model.Instance{
		Model: gorm.Model{ID: 200},
		Name:  "target-deleted", InstanceId: "ins-target-del",
		UserID: 1,
	}
	model.DB(context.Background()).Create(&targetInst)
	model.DB(context.Background()).Delete(&targetInst) // 软删除

	// 创建有快照但未清理的 session
	session := model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 200,
		Status:           model.DoctorStatusEnded,
		HasSnapshot:      true,
		SnapshotFileKey:  "doctor/snapshots/test-key.tar.gz",
		SnapshotDeleted:  false,
	}
	model.DB(context.Background()).Create(&session)

	deleted := false
	snap := saveTaskFns()
	defer snap.restore()
	controller.DeleteSMHCommonFileFn =
		func(_ context.Context, fileKey string) error {
			if fileKey == "doctor/snapshots/test-key.tar.gz" {
				deleted = true
			}
			return nil
		}

	ctx := context.Background()
	log := slog.Default()
	cleanupOrphanedSnapshots(ctx, log)

	if !deleted {
		t.Error("DeleteSMHCommonFile 未被调用")
	}

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if !updated.SnapshotDeleted {
		t.Error("snapshot_deleted 应为 true")
	}
}

func TestCleanupOrphanedSnapshots_目标实例未删除则跳过(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	// 目标实例存在（未删除），ID=1 由 initCleanupTestDB 创建
	session := model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 1,
		Status:           model.DoctorStatusEnded,
		HasSnapshot:      true,
		SnapshotFileKey:  "doctor/snapshots/should-not-delete.tar.gz",
		SnapshotDeleted:  false,
	}
	model.DB(context.Background()).Create(&session)

	deleted := false
	snap := saveTaskFns()
	defer snap.restore()
	controller.DeleteSMHCommonFileFn =
		func(_ context.Context, fileKey string) error {
			deleted = true
			return nil
		}

	ctx := context.Background()
	log := slog.Default()
	cleanupOrphanedSnapshots(ctx, log)

	if deleted {
		t.Error("目标实例未删除时不应调用 DeleteSMHCommonFile")
	}

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.SnapshotDeleted {
		t.Error("snapshot_deleted 应保持 false")
	}
}

func TestCleanupOrphanedSnapshots_已标记删除则跳过(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	// 目标实例已删除
	targetInst := model.Instance{
		Model: gorm.Model{ID: 201},
		Name:  "target-del2", InstanceId: "ins-target-del2",
		UserID: 1,
	}
	model.DB(context.Background()).Create(&targetInst)
	model.DB(context.Background()).Delete(&targetInst)

	// session 快照已标记为已删除
	session := model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 201,
		Status:           model.DoctorStatusEnded,
		HasSnapshot:      true,
		SnapshotFileKey:  "doctor/snapshots/already-deleted.tar.gz",
		SnapshotDeleted:  true,
		SessionsDeleted:  true,
	}
	model.DB(context.Background()).Create(&session)

	deleted := false
	snap := saveTaskFns()
	defer snap.restore()
	controller.DeleteSMHCommonFileFn =
		func(_ context.Context, fileKey string) error {
			deleted = true
			return nil
		}

	ctx := context.Background()
	log := slog.Default()
	cleanupOrphanedSnapshots(ctx, log)

	if deleted {
		t.Error("已标记 snapshot_deleted=true 时不应再调用 DeleteSMHCommonFile")
	}
}

func TestCleanupOrphanedSnapshots_删除失败则不标记(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	// 目标实例已删除
	targetInst := model.Instance{
		Model: gorm.Model{ID: 202},
		Name:  "target-del3", InstanceId: "ins-target-del3",
		UserID: 1,
	}
	model.DB(context.Background()).Create(&targetInst)
	model.DB(context.Background()).Delete(&targetInst)

	session := model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 202,
		Status:           model.DoctorStatusEnded,
		HasSnapshot:      true,
		SnapshotFileKey:  "doctor/snapshots/fail-delete.tar.gz",
		SnapshotDeleted:  false,
	}
	model.DB(context.Background()).Create(&session)

	snap := saveTaskFns()
	defer snap.restore()
	controller.DeleteSMHCommonFileFn =
		func(_ context.Context, fileKey string) error {
			return fmt.Errorf("SMH unavailable")
		}

	ctx := context.Background()
	log := slog.Default()
	cleanupOrphanedSnapshots(ctx, log)

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.SnapshotDeleted {
		t.Error("删除失败时 snapshot_deleted 应保持 false，等下次重试")
	}
}

func TestCleanupOrphanedSnapshots_目标实例硬删除也清理(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	// TargetInstanceID=999 在数据库中完全不存在
	session := model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 999,
		Status:           model.DoctorStatusEnded,
		HasSnapshot:      true,
		SnapshotFileKey:  "doctor/snapshots/hard-deleted.tar.gz",
		SnapshotDeleted:  false,
	}
	model.DB(context.Background()).Create(&session)

	deleted := false
	snap := saveTaskFns()
	defer snap.restore()
	controller.DeleteSMHCommonFileFn =
		func(_ context.Context, fileKey string) error {
			deleted = true
			return nil
		}

	ctx := context.Background()
	log := slog.Default()
	cleanupOrphanedSnapshots(ctx, log)

	if !deleted {
		t.Error("目标实例不存在时也应清理快照")
	}

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if !updated.SnapshotDeleted {
		t.Error("snapshot_deleted 应为 true")
	}
}

// ─── cleanupOrphanedSessionFiles 测试 ───────────────────────────────────────

func TestCleanupOrphanedSessionFiles_目标实例已删除则清理(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	// 创建目标实例并软删除
	targetInst := model.Instance{
		Model: gorm.Model{ID: 300},
		Name:  "target-del-sess", InstanceId: "ins-target-del-sess",
		UserID: 1,
	}
	model.DB(context.Background()).Create(&targetInst)
	model.DB(context.Background()).Delete(&targetInst)

	session := model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 300,
		Status:           model.DoctorStatusEnded,
		HasSnapshot:      false,
		SessionsDeleted:  false,
	}
	model.DB(context.Background()).Create(&session)

	deletedKey := ""
	snap := saveTaskFns()
	defer snap.restore()
	controller.DeleteSMHCommonFileFn =
		func(_ context.Context, fileKey string) error {
			deletedKey = fileKey
			return nil
		}

	ctx := context.Background()
	log := slog.Default()
	cleanupOrphanedSessionFiles(ctx, log)

	expectedKey := "doctor-sessions/1/300/sessions.tar.gz"
	if deletedKey != expectedKey {
		t.Errorf("DeleteSMHCommonFile 参数 = %q, want %q", deletedKey, expectedKey)
	}

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if !updated.SessionsDeleted {
		t.Error("sessions_deleted 应为 true")
	}
}

func TestCleanupOrphanedSessionFiles_目标实例未删除则跳过(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	// 目标实例存在（ID=1 由 initCleanupTestDB 创建）
	session := model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 1,
		Status:           model.DoctorStatusEnded,
		HasSnapshot:      false,
		SessionsDeleted:  false,
	}
	model.DB(context.Background()).Create(&session)

	deleted := false
	snap := saveTaskFns()
	defer snap.restore()
	controller.DeleteSMHCommonFileFn =
		func(_ context.Context, fileKey string) error {
			deleted = true
			return nil
		}

	ctx := context.Background()
	log := slog.Default()
	cleanupOrphanedSessionFiles(ctx, log)

	if deleted {
		t.Error("目标实例未删除时不应调用 DeleteSMHCommonFile")
	}
}

func TestCleanupOrphanedSessionFiles_已标记删除则跳过(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	// 目标实例已删除
	targetInst := model.Instance{
		Model: gorm.Model{ID: 301},
		Name:  "target-del-sess2", InstanceId: "ins-target-del-sess2",
		UserID: 1,
	}
	model.DB(context.Background()).Create(&targetInst)
	model.DB(context.Background()).Delete(&targetInst)

	session := model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 301,
		Status:           model.DoctorStatusEnded,
		HasSnapshot:      false,
		SessionsDeleted:  true, // 已标记
	}
	model.DB(context.Background()).Create(&session)

	deleted := false
	snap := saveTaskFns()
	defer snap.restore()
	controller.DeleteSMHCommonFileFn =
		func(_ context.Context, fileKey string) error {
			deleted = true
			return nil
		}

	ctx := context.Background()
	log := slog.Default()
	cleanupOrphanedSessionFiles(ctx, log)

	if deleted {
		t.Error("已标记 sessions_deleted=true 时不应再调用 DeleteSMHCommonFile")
	}
}
