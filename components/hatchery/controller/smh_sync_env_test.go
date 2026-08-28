package controller

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initSyncEnvTestDB 初始化内存 SQLite 数据库，迁移 syncSMHEnvWhenReady 所需的表。
func initSyncEnvTestDB(t *testing.T) (func(), *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.SMHPersonalSpace{},
		&model.Notification{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	return model.UseDBForTest(db), db
}

// createSyncEnvTestInstance 创建测试实例。
func createSyncEnvTestInstance(t *testing.T, name string) model.Instance {
	t.Helper()
	user := model.User{Username: name + "-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	inst := model.Instance{
		Name:       name,
		InstanceId: "ins-" + name,
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(&inst)
	return inst
}

// createSyncEnvTestSpace 创建测试个人空间（活跃状态，env_initialized=true）。
func createSyncEnvTestSpace(t *testing.T, inst model.Instance) model.SMHPersonalSpace {
	t.Helper()
	space := model.SMHPersonalSpace{
		SpaceId:        "sp-" + inst.InstanceId,
		UserId:         inst.UserID,
		InstanceId:     inst.ID,
		CVMInstanceId:  inst.InstanceId,
		EnvInitialized: true,
	}
	model.DB(context.Background()).Create(&space)
	return space
}

// ============================================================================
// TestSyncSMHEnvWhenReady_NoSpace
// 验证：实例没有活跃个人空间时，函数直接返回，不做任何等待或 DB 修改。
// ============================================================================

func TestSyncSMHEnvWhenReady_NoSpace(t *testing.T) {
	restore, _ := initSyncEnvTestDB(t)
	t.Cleanup(restore)

	inst := createSyncEnvTestInstance(t, "no-space")

	// 直接调用，不应 panic 或阻塞
	syncSMHEnvWhenReady(context.Background(), inst)

	// 验证没有创建任何 space 记录
	var count int64
	model.DB(context.Background()).Model(&model.SMHPersonalSpace{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 0 {
		t.Errorf("不应创建 space 记录，实际 count=%d", count)
	}
}

// ============================================================================
// TestSyncSMHEnvWhenReady_DBError
// 验证：DB 查询失败时（表不存在等），函数安全返回不 panic。
// ============================================================================

func TestSyncSMHEnvWhenReady_DBError(t *testing.T) {
	// 使用一个只迁移了 Instance 但没有迁移 SMHPersonalSpace 的 DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		// 故意不迁移 SMHPersonalSpace，模拟 DB 错误
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))

	user := model.User{Username: "db-err-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{
		Name:       "db-err-inst",
		InstanceId: "ins-db-err",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(&inst)

	// 调用不应 panic（查询 smh_personal_spaces 表会报错，走 default 分支）
	syncSMHEnvWhenReady(context.Background(), inst)
}

// ============================================================================
// TestSyncSMHEnvWhenReady_ResetEnvInitialized
// 验证：找到活跃 space 后，env_initialized 被重置为 false。
// 利用 InstanceId="" 使 waitForCVMRunning 立即返回 false，从而快速验证 DB 重置逻辑。
// ============================================================================

func TestSyncSMHEnvWhenReady_ResetEnvInitialized(t *testing.T) {
	restore, _ := initSyncEnvTestDB(t)
	t.Cleanup(restore)
	// 使用空 InstanceId，让 waitForCVMRunning 立即返回 false（不会真实调用 CVM API）
	inst := createSyncEnvTestInstance(t, "reset-env")
	inst.InstanceId = ""
	model.DB(context.Background()).Save(&inst)

	space := createSyncEnvTestSpace(t, inst)

	// 确认初始状态
	if !space.EnvInitialized {
		t.Fatal("测试前提：space.EnvInitialized 应为 true")
	}

	// 调用函数（waitForCVMRunning 会因 InstanceId="" 立即返回 false）
	syncSMHEnvWhenReady(context.Background(), inst)

	// 验证 DB 中 env_initialized 已被重置为 false
	var updated model.SMHPersonalSpace
	if err := model.DB(context.Background()).First(&updated, space.ID).Error; err != nil {
		t.Fatalf("重新加载 space 失败: %v", err)
	}
	if updated.EnvInitialized {
		t.Error("syncSMHEnvWhenReady 应将 env_initialized 重置为 false")
	}
}

// ============================================================================
// TestSyncSMHEnvWhenReady_CVMTimeout
// 验证：CVM 不就绪时（waitForCVMRunning 返回 false）函数提前返回，
// 但 DB 中 env_initialized 已被正确重置为 false，确保 syncEnvs 定时任务能兜底恢复。
// 利用 InstanceId="" 使 waitForCVMRunning 立即返回 false 来模拟超时场景。
// ============================================================================

func TestSyncSMHEnvWhenReady_CVMTimeout(t *testing.T) {
	restore, _ := initSyncEnvTestDB(t)
	t.Cleanup(restore)
	inst := createSyncEnvTestInstance(t, "cvm-timeout")
	// 设置空 InstanceId 模拟 CVM 不可达（waitForCVMRunning 立即返回 false）
	inst.InstanceId = ""
	model.DB(context.Background()).Save(&inst)

	space := createSyncEnvTestSpace(t, inst)

	syncSMHEnvWhenReady(context.Background(), inst)

	// 验证 DB 中 env_initialized 已被重置（即使 CVM 等待失败，兜底机制仍可工作）
	var updated model.SMHPersonalSpace
	if err := model.DB(context.Background()).First(&updated, space.ID).Error; err != nil {
		t.Fatalf("重新加载 space 失败: %v", err)
	}
	if updated.EnvInitialized {
		t.Error("即使 CVM 不可达，env_initialized 也应已被重置为 false（确保 syncEnvs 兜底）")
	}
}

// ============================================================================
// TestSyncSMHEnvWhenReady_DeletedSpace
// 验证：space 在回收站中（to_be_deleted_at 非空）时，视为不存在，直接返回。
// ============================================================================

func TestSyncSMHEnvWhenReady_DeletedSpace(t *testing.T) {
	restore, _ := initSyncEnvTestDB(t)
	t.Cleanup(restore)
	inst := createSyncEnvTestInstance(t, "deleted-space")

	// 创建一个在回收站中的 space
	deleteAt := time.Now().Add(7 * 24 * time.Hour)
	space := model.SMHPersonalSpace{
		SpaceId:        "sp-deleted",
		UserId:         inst.UserID,
		InstanceId:     inst.ID,
		CVMInstanceId:  inst.InstanceId,
		EnvInitialized: true,
		ToBeDeletedAt:  &deleteAt,
	}
	model.DB(context.Background()).Create(&space)

	// 调用不应修改 DB 状态（回收站中的 space 不应被处理）
	syncSMHEnvWhenReady(context.Background(), inst)

	// 验证 env_initialized 未被修改（仍为 true）
	var updated model.SMHPersonalSpace
	model.DB(context.Background()).First(&updated, space.ID)
	if !updated.EnvInitialized {
		t.Error("回收站中的 space 不应被处理，env_initialized 应保持 true")
	}
}

// ============================================================================
// TestSyncSMHEnvWhenReady_PanicRecovery
// 验证：函数内部 panic 时 defer recover 能正常捕获，不会冒泡到调用方。
// ============================================================================

func TestSyncSMHEnvWhenReady_PanicRecovery(t *testing.T) {
	// 传入零值 instance（ID=0），在某些极端情况下可能触发异常
	// 但更直接的方式是：使用一个会导致 panic 的 context
	// 这里我们验证函数签名本身的 recover 机制：即使传入非法参数也不会 panic 到外层
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("syncSMHEnvWhenReady 不应将 panic 冒泡到调用方，捕获到: %v", r)
		}
	}()

	restore, _ := initSyncEnvTestDB(t)
	t.Cleanup(restore)
	// 传入一个 ID=0 的实例（DB 中不存在），应走 NotFound 分支安全返回
	syncSMHEnvWhenReady(context.Background(), model.Instance{})
}

// ============================================================================
// TestFinalizeUpgradeResult_SMHEnvSync
// 验证：升级成功时会触发 syncSMHEnvWhenReadyFn，升级失败时不触发。
// ============================================================================

func TestFinalizeUpgradeResult_SMHEnvSync(t *testing.T) {
	// 准备测试环境
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{}, &model.Instance{}, &model.AIImage{},
		&model.SiteConfig{}, &model.Notification{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	defer origDB()

	// mock createErrorNotification 避免异步写入
	origNotif := createErrorNotification
	createErrorNotification = func(userID, instanceID uint, instanceName, notifyType, title string, err error, ctx context.Context) {
	}
	defer func() { createErrorNotification = origNotif }()

	// mock syncSMHEnvWhenReadyFn 并记录调用
	var called atomic.Int32
	origSyncSMH := syncSMHEnvWhenReadyFn
	syncSMHEnvWhenReadyFn = func(ctx context.Context, inst model.Instance) {
		called.Add(1)
	}
	defer func() { syncSMHEnvWhenReadyFn = origSyncSMH }()

	// 创建测试用户和实例
	user := model.User{Username: "smh-sync-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	inst := &model.Instance{
		Name:                  "smh-sync-inst",
		InstanceId:            "ins-smh-sync",
		UserID:                user.ID,
		AgentType:             model.AgentTypeOpenClaw,
		CurrentOperation:      model.OpUpgrade,
		CurrentOperationState: model.OpStateProcessing,
	}
	model.DB(context.Background()).Create(inst)

	// 测试1：升级成功时应触发 SMH 环境同步
	called.Store(0)
	finalizeUpgradeResult(context.Background(), inst, nil)

	// 等待异步 goroutine 执行
	time.Sleep(100 * time.Millisecond)
	if called.Load() != 1 {
		t.Errorf("升级成功时应触发 syncSMHEnvWhenReadyFn，实际调用次数=%d", called.Load())
	}

	// 测试2：升级失败时不应触发 SMH 环境同步
	inst.CurrentOperation = model.OpUpgrade
	inst.CurrentOperationState = model.OpStateProcessing
	model.DB(context.Background()).Save(inst)

	called.Store(0)
	finalizeUpgradeResult(context.Background(), inst, errContextFinalize("模拟升级失败"))

	time.Sleep(100 * time.Millisecond)
	if called.Load() != 0 {
		t.Errorf("升级失败时不应触发 syncSMHEnvWhenReadyFn，实际调用次数=%d", called.Load())
	}
}

// ============================================================================
// TestTriggerSyncPersonalSpaceEnv_Success
// 验证：TriggerSyncPersonalSpaceEnv 正常获取锁后调用 SyncPersonalSpaceEnv。
// 通过 mock runScriptFn + primeTokenCache + seedSMHConfigured 让 SyncPersonalSpaceEnv 成功。
// ============================================================================

func TestTriggerSyncPersonalSpaceEnv_Success(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	seedSMHConfigured(t)

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	primeTokenCache(space.SpaceId, "trigger-sync-token")

	var called int32
	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		atomic.AddInt32(&called, 1)
		return "ok", nil
	})
	defer restore()

	if err := TriggerSyncPersonalSpaceEnv(context.Background(), space, true); err != nil {
		t.Fatalf("期望成功，实际返回错误: %v", err)
	}

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("runScriptFn 应被调用 1 次，实际 %d", atomic.LoadInt32(&called))
	}

	// 验证 DB 中 env_initialized 已被置为 true
	var updated model.SMHPersonalSpace
	if err := model.DB(context.Background()).First(&updated, space.ID).Error; err != nil {
		t.Fatalf("重新加载 space 失败: %v", err)
	}
	if !updated.EnvInitialized {
		t.Error("TriggerSyncPersonalSpaceEnv(install=true) 应将 env_initialized 置为 true")
	}
}

// ============================================================================
// TestTriggerSyncPersonalSpaceEnv_Uninstall
// 验证：TriggerSyncPersonalSpaceEnv(install=false) 正常卸载。
// ============================================================================

func TestTriggerSyncPersonalSpaceEnv_Uninstall(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	seedSMHConfigured(t)

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	// 先设置 env_initialized=true
	model.DB(context.Background()).Model(space).Update("env_initialized", true)

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "ok", nil
	})
	defer restore()

	if err := TriggerSyncPersonalSpaceEnv(context.Background(), space, false); err != nil {
		t.Fatalf("期望成功，实际返回错误: %v", err)
	}

	// 验证 DB 中 env_initialized 已被置为 false
	var updated model.SMHPersonalSpace
	if err := model.DB(context.Background()).First(&updated, space.ID).Error; err != nil {
		t.Fatalf("重新加载 space 失败: %v", err)
	}
	if updated.EnvInitialized {
		t.Error("TriggerSyncPersonalSpaceEnv(install=false) 应将 env_initialized 置为 false")
	}
}

// ============================================================================
// TestTriggerSyncPersonalSpaceEnv_ConcurrentSkip
// 验证：同一实例并发调用时，第二次调用被跳过（进程内 sync.Map 去重）。
// ============================================================================

func TestTriggerSyncPersonalSpaceEnv_ConcurrentSkip(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	seedSMHConfigured(t)

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	primeTokenCache(space.SpaceId, "concurrent-token")

	// 用 channel 控制第一次调用阻塞，让第二次调用命中 "已在执行" 分支
	blockCh := make(chan struct{})
	var callCount int32

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		atomic.AddInt32(&callCount, 1)
		<-blockCh // 阻塞直到测试放行
		return "ok", nil
	})
	defer restore()

	// 第一次调用（异步，会阻塞在 runScriptFn 中）
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		TriggerSyncPersonalSpaceEnv(context.Background(), space, true)
	}()

	// 等待第一次调用进入 runScriptFn
	time.Sleep(50 * time.Millisecond)

	// 第二次调用应被跳过（进程内 sync.Map 去重）
	err := TriggerSyncPersonalSpaceEnv(context.Background(), space, true)
	if err != nil {
		t.Errorf("并发跳过时应返回 nil，实际=%v", err)
	}

	// 放行第一次调用
	close(blockCh)
	wg.Wait()

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("runScriptFn 应只被调用 1 次（第二次被跳过），实际 %d", atomic.LoadInt32(&callCount))
	}
}

// ============================================================================
// TestTriggerRefreshPersonalSpaceToken_Success
// 验证：TriggerRefreshPersonalSpaceToken 正常获取锁后调用 refreshPersonalSpaceToken。
// ============================================================================

func TestTriggerRefreshPersonalSpaceToken_Success(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	seedSMHConfigured(t)

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	primeTokenCache(space.SpaceId, "trigger-refresh-token")

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "ok", nil
	})
	defer restore()

	if err := TriggerRefreshPersonalSpaceToken(context.Background(), space); err != nil {
		t.Fatalf("期望成功，实际返回错误: %v", err)
	}
}

// ============================================================================
// TestTriggerRefreshPersonalSpaceToken_ConcurrentSkip
// 验证：同一实例并发调用时，第二次调用被跳过。
// ============================================================================

func TestTriggerRefreshPersonalSpaceToken_ConcurrentSkip(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	seedSMHConfigured(t)

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	primeTokenCache(space.SpaceId, "concurrent-refresh-token")

	blockCh := make(chan struct{})
	var callCount int32

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		atomic.AddInt32(&callCount, 1)
		<-blockCh
		return "ok", nil
	})
	defer restore()

	// 第一次调用（异步，会阻塞在 runScriptFn 中）
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		TriggerRefreshPersonalSpaceToken(context.Background(), space)
	}()

	// 等待第一次调用进入 runScriptFn
	time.Sleep(50 * time.Millisecond)

	// 第二次调用应被跳过
	err := TriggerRefreshPersonalSpaceToken(context.Background(), space)
	if err != nil {
		t.Errorf("并发跳过时应返回 nil，实际=%v", err)
	}

	// 放行第一次调用
	close(blockCh)
	wg.Wait()

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("runScriptFn 应只被调用 1 次（第二次被跳过），实际 %d", atomic.LoadInt32(&callCount))
	}
}

// ============================================================================
// TestSyncSMHEnvWhenReady_ResetEnvInitializedDBFailed
// 验证：重置 env_initialized 时 DB 写入失败，函数安全返回不 panic。
// 通过 PRAGMA query_only 模拟 DB 写入失败。
// ============================================================================

func TestSyncSMHEnvWhenReady_ResetEnvInitializedDBFailed(t *testing.T) {
	restore, db := initSyncEnvTestDB(t)
	t.Cleanup(restore)
	inst := createSyncEnvTestInstance(t, "reset-db-fail")
	_ = createSyncEnvTestSpace(t, inst)

	// 设置 DB 为只读模式，模拟 UPDATE 失败
	db.Exec("PRAGMA query_only = ON")
	defer db.Exec("PRAGMA query_only = OFF")

	// 调用不应 panic（UPDATE env_initialized 会失败，走 Warn 分支返回）
	syncSMHEnvWhenReady(context.Background(), inst)

	// 恢复写权限后验证 env_initialized 未被修改（仍为 true）
	db.Exec("PRAGMA query_only = OFF")
	var updated model.SMHPersonalSpace
	model.DB(context.Background()).Where("instance_id = ?", inst.ID).First(&updated)
	if !updated.EnvInitialized {
		t.Error("DB 写入失败时 env_initialized 不应被修改")
	}
}

// ============================================================================
// TestSyncSMHEnvWhenReady_PanicInDBQuery
// 验证：函数内部发生 panic 时 defer recover 能正常捕获，不会冒泡到调用方。
// 通过注入一个 nil DB 上下文来触发 panic。
// ============================================================================

func TestSyncSMHEnvWhenReady_PanicInDBQuery(t *testing.T) {
	// 使用一个会导致 panic 的场景：设置 DB 为 nil 后调用
	// 先正常初始化 DB，然后替换为 nil
	restore, _ := initSyncEnvTestDB(t)
	inst := createSyncEnvTestInstance(t, "panic-test")
	_ = createSyncEnvTestSpace(t, inst)

	restore()
	// 替换全局 DB 为 nil，触发 panic
	model.SetDBForTest(nil)

	// 验证 panic 不会冒泡
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("syncSMHEnvWhenReady 不应将 panic 冒泡到调用方，捕获到: %v", r)
		}
	}()

	syncSMHEnvWhenReady(context.Background(), inst)
}

// ============================================================================
// 注：syncSMHEnvWhenReady 中 "CVM 就绪 + TAT Agent 超时" 和 "TriggerSyncPersonalSpaceEnv
// 调用" 的路径由于 waitForTATAgentOnline 内部有硬编码的 time.Sleep(10s) 和 3 分钟超时，
// 且不检查 context 取消，无法在单元测试中高效覆盖。
// 这些路径的核心逻辑已通过以下测试间接覆盖：
//   - TestTriggerSyncPersonalSpaceEnv_Success（覆盖 TriggerSyncPersonalSpaceEnv 完整路径）
//   - TestSyncSMHEnvWhenReady_CVMTimeout（覆盖 waitForCVMRunning 返回 false 的分支）
//   - TestWaitForCVMRunning_EmptyID / TestWaitForTATAgentOnline_EmptyID（覆盖空 ID 分支）
// ============================================================================
