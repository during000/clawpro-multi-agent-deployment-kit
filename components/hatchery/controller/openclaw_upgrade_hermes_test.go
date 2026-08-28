// openclaw_upgrade_hermes_test.go
//
// 针对本次一键升级 Hermes 化改造新增的 3 个函数补齐单元测试：
//   - redetectAndPersistRuntimeUser：以 root 探测 → 与 DB 对比 → 一致 no-op / 不一致回写 / 全失败超时
//   - runHermesUpgradePostHooks   ：waitForOpenclawReady 二次兜底 + cleanupUpgradeTemp
//   - runOpenClawUpgradePostHooks ：5 项私有补丁串行执行、失败仅 warn 不阻断
//
// 测试策略：
//   - 全部走 setupUpgradeExtraEnv（内存 SQLite + mock runScriptFn），零真实 TAT 调用；
//   - 通过替换 runScriptFn 的闭包，为不同 scriptName 返回不同的桩数据/错误；
//   - Hermes hook 失败分支借助 agent_type 未注册 check_ready 让 ResolveScript 直接返回 error，
//     从而秒级触达 waitForOpenclawReady 的 error 分支，避免真的等 5min。
package controller

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"hatchery/model"
)

// ============================================================================
// redetectAndPersistRuntimeUser
// ============================================================================

// TestRedetectAndPersistRuntimeUser_UnknownAgentType 验证：
// 当 agentType 未注册 detect_install（例如 DeepSeekTUI）时，函数应记录 Warn 并直接返回 nil，
// 不阻断升级流程；也不会调用 runScriptFn。
func TestRedetectAndPersistRuntimeUser_UnknownAgentType(t *testing.T) {
	setupUpgradeExtraEnv(t)

	var callCount int32
	origRunScript := runScriptFn
	runScriptFn = func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		atomic.AddInt32(&callCount, 1)
		return "", nil
	}
	t.Cleanup(func() { runScriptFn = origRunScript })

	inst := &model.Instance{
		InstanceId:  "ins-detect-unknown",
		AgentType:   model.AgentTypeDeepSeekTUI, // 未注册 detect_install
		RuntimeUser: "root",
		RuntimeHome: "/root",
	}
	if err := redetectAndPersistRuntimeUser(context.Background(), inst, 100*time.Millisecond); err != nil {
		t.Fatalf("expected nil error for unknown agent type, got %v", err)
	}
	if got := atomic.LoadInt32(&callCount); got != 0 {
		t.Errorf("未注册脚本时不应调用 runScriptFn，实际调用=%d", got)
	}
}

// TestRedetectAndPersistRuntimeUser_ConsistentWithDB 验证：
// 探测结果与 DB 完全一致时，函数立即返回 nil，且不进行 DB 更新（内存对象也保持不变）。
func TestRedetectAndPersistRuntimeUser_ConsistentWithDB(t *testing.T) {
	setupUpgradeExtraEnv(t)

	var callCount int32
	origRunScript := runScriptFn
	runScriptFn = func(ctx context.Context, _, scriptName string, _ uint64, runtimeUser string, _ func(string), _ map[string]string) (string, error) {
		atomic.AddInt32(&callCount, 1)
		// runtimeUser 应为空（内部 fallback root）
		if runtimeUser != "" {
			t.Errorf("expected runtimeUser=\"\" (root fallback), got %q", runtimeUser)
		}
		if !strings.Contains(scriptName, "detect_") {
			t.Errorf("expected detect_* script, got %q", scriptName)
		}
		return `{"runtime_user":"ubuntu","runtime_home":"/home/ubuntu"}`, nil
	}
	t.Cleanup(func() { runScriptFn = origRunScript })

	// DB 里已经预置一条记录，字段与探测结果保持一致
	db := upgradeExtraTestDB
	inst := &model.Instance{
		InstanceId:  "ins-detect-consistent",
		AgentType:   model.AgentTypeHermes,
		RuntimeUser: "ubuntu",
		RuntimeHome: "/home/ubuntu",
	}
	if err := db.Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	if err := redetectAndPersistRuntimeUser(context.Background(), inst, 5*time.Second); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Errorf("expected 1 script call, got %d", got)
	}
	if inst.RuntimeUser != "ubuntu" || inst.RuntimeHome != "/home/ubuntu" {
		t.Errorf("内存对象不应被修改：runtime_user=%q runtime_home=%q", inst.RuntimeUser, inst.RuntimeHome)
	}
}

// TestRedetectAndPersistRuntimeUser_MismatchWritesBack 验证：
// 探测结果与 DB 不一致时，函数应把探测值回写 DB，并同步内存对象；返回 nil。
func TestRedetectAndPersistRuntimeUser_MismatchWritesBack(t *testing.T) {
	setupUpgradeExtraEnv(t)

	origRunScript := runScriptFn
	runScriptFn = func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		// 返回一个与 DB 不同的结果（新镜像默认用户从 agentuser 切到 ubuntu）
		return `{"runtime_user":"ubuntu","runtime_home":"/home/ubuntu"}`, nil
	}
	t.Cleanup(func() { runScriptFn = origRunScript })

	db := upgradeExtraTestDB
	inst := &model.Instance{
		InstanceId:  "ins-detect-mismatch",
		AgentType:   model.AgentTypeHermes,
		RuntimeUser: "agentuser", // 旧值
		RuntimeHome: "/home/agentuser",
	}
	if err := db.Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	if err := redetectAndPersistRuntimeUser(context.Background(), inst, 5*time.Second); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	// 内存对象应被同步
	if inst.RuntimeUser != "ubuntu" || inst.RuntimeHome != "/home/ubuntu" {
		t.Errorf("内存对象未同步：runtime_user=%q runtime_home=%q", inst.RuntimeUser, inst.RuntimeHome)
	}

	// DB 应被回写
	var got model.Instance
	if err := db.Where("instance_id = ?", "ins-detect-mismatch").First(&got).Error; err != nil {
		t.Fatalf("query updated instance: %v", err)
	}
	if got.RuntimeUser != "ubuntu" || got.RuntimeHome != "/home/ubuntu" {
		t.Errorf("DB 未回写：runtime_user=%q runtime_home=%q", got.RuntimeUser, got.RuntimeHome)
	}
}

// TestRedetectAndPersistRuntimeUser_TimeoutOnAllFailures 验证：
// 脚本连续失败 / 返回不合法 JSON 时，超过 timeout 后应返回错误（含 "重探" 或 i18n key 描述）。
// 用极短 timeout（10ms）+ 立即返回 error 让循环秒级触达 deadline 分支。
func TestRedetectAndPersistRuntimeUser_PersistMissingRowFails(t *testing.T) {
	setupUpgradeExtraEnv(t)

	origRunScript := runScriptFn
	runScriptFn = func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
		return `{"runtime_user":"ubuntu","runtime_home":"/home/ubuntu"}`, nil
	}
	t.Cleanup(func() { runScriptFn = origRunScript })

	inst := &model.Instance{
		InstanceId:  "ins-detect-missing-row",
		AgentType:   model.AgentTypeHermes,
		RuntimeUser: "agentuser",
		RuntimeHome: "/home/agentuser",
	}
	if err := redetectAndPersistRuntimeUser(context.Background(), inst, time.Second); err == nil {
		t.Fatal("runtime_user 写入未命中实例时应返回错误")
	}
	if inst.RuntimeUser != "agentuser" || inst.RuntimeHome != "/home/agentuser" {
		t.Fatalf("持久化失败时不应修改内存对象，got user=%q home=%q", inst.RuntimeUser, inst.RuntimeHome)
	}
}

func TestRedetectAndPersistRuntimeUser_TimeoutOnAllFailures(t *testing.T) {
	setupUpgradeExtraEnv(t)

	var callCount int32
	origRunScript := runScriptFn
	runScriptFn = func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		atomic.AddInt32(&callCount, 1)
		return "", errors.New("simulated tat failure")
	}
	t.Cleanup(func() { runScriptFn = origRunScript })

	inst := &model.Instance{
		InstanceId:  "ins-detect-timeout",
		AgentType:   model.AgentTypeHermes,
		RuntimeUser: "agentuser",
		RuntimeHome: "/home/agentuser",
	}
	// 10ms → 单次调用后 time.Now().After(deadline) 立即为真，直接返回错误
	err := redetectAndPersistRuntimeUser(context.Background(), inst, 10*time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if got := atomic.LoadInt32(&callCount); got < 1 {
		t.Errorf("expected at least 1 script call, got %d", got)
	}
}

// TestRedetectAndPersistRuntimeUser_EmptyRuntimeUserThenSuccess 验证：
// 首次探测返回 runtime_user="" 视为未就绪继续等待；deadline 触达后仍报错。
// 用于覆盖 "解析成功但值为空/unknown" 的分支。
func TestRedetectAndPersistRuntimeUser_EmptyRuntimeUserThenTimeout(t *testing.T) {
	setupUpgradeExtraEnv(t)

	origRunScript := runScriptFn
	runScriptFn = func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		return `{"runtime_user":"","runtime_home":""}`, nil
	}
	t.Cleanup(func() { runScriptFn = origRunScript })

	inst := &model.Instance{
		InstanceId:  "ins-detect-empty",
		AgentType:   model.AgentTypeHermes,
		RuntimeUser: "agentuser",
		RuntimeHome: "/home/agentuser",
	}
	err := redetectAndPersistRuntimeUser(context.Background(), inst, 10*time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout error when runtime_user is empty, got nil")
	}
}

// TestRedetectAndPersistRuntimeUser_NilInstance 防御性测试：
// instance 为 nil 时应直接返回 nil，不 panic。
func TestRedetectAndPersistRuntimeUser_NilInstance(t *testing.T) {
	setupUpgradeExtraEnv(t)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("nil instance 不应 panic，实际=%v", r)
		}
	}()
	if err := redetectAndPersistRuntimeUser(context.Background(), nil, time.Second); err != nil {
		t.Errorf("nil instance expected nil error, got %v", err)
	}
}

func TestIsRetryableRestoreDispatchError(t *testing.T) {
	if !isRetryableRestoreDispatchError(ErrTATCommandDispatchFailed) {
		t.Fatal("TAT dispatch failure must be retryable")
	}
	if !isRetryableRestoreDispatchError(ErrTATCommandStartFailed) {
		t.Fatal("TAT start failure must be retryable")
	}
	if isRetryableRestoreDispatchError(errors.New("restore script failed")) {
		t.Fatal("restore script failure must not be retryable")
	}
}

// ============================================================================
// runHermesUpgradePostHooks
// ============================================================================

// TestRunHermesUpgradePostHooks_Success 验证：
// waitForOpenclawReady 探测到 ready:true 时，hook 立即返回 nil；cleanupUpgradeTemp 亦被下发。
func TestRunHermesUpgradePostHooks_Success(t *testing.T) {
	setupUpgradeExtraEnv(t)

	var (
		readyCalls   int32
		cleanupCalls int32
	)
	origRunScript := runScriptFn
	runScriptFn = func(ctx context.Context, _, scriptName string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		switch {
		case strings.HasPrefix(scriptName, "check_hermes_ready") || strings.HasPrefix(scriptName, "check_openclaw_ready"):
			atomic.AddInt32(&readyCalls, 1)
			return `{"ready": true}`, nil
		case strings.HasPrefix(scriptName, "cleanup_upgrade_temp"):
			atomic.AddInt32(&cleanupCalls, 1)
			return "", nil
		default:
			return "", nil
		}
	}
	t.Cleanup(func() { runScriptFn = origRunScript })

	inst := &model.Instance{
		InstanceId:  "ins-hermes-hook-ok",
		AgentType:   model.AgentTypeHermes,
		RuntimeUser: "ubuntu",
		RuntimeHome: "/home/ubuntu",
	}
	if err := runHermesUpgradePostHooks(context.Background(), inst); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if got := atomic.LoadInt32(&readyCalls); got < 1 {
		t.Errorf("expected at least 1 ready call, got %d", got)
	}
	if got := atomic.LoadInt32(&cleanupCalls); got != 1 {
		t.Errorf("expected exactly 1 cleanup call, got %d", got)
	}
}

// TestRunHermesUpgradePostHooks_ReadyResolveFail_WarnOnly 验证：
// waitForOpenclawReady 内 ResolveScript 失败（例如 agentType 不在 check_ready 白名单）时，
// hook 应 warn 不阻断（返回 nil），与 OpenClaw 后置语义对齐。agent_ready 由 AgentChecker 兜底。
// 用未注册 check_ready 的 DeepSeekTUI 直接触达 ResolveScript error 分支，避免真的 sleep 5min。
func TestRunHermesUpgradePostHooks_ReadyResolveFail_WarnOnly(t *testing.T) {
	setupUpgradeExtraEnv(t)

	origRunScript := runScriptFn
	runScriptFn = func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		// 该分支下不应被调用（ResolveScript 已失败），一旦被调用返回 error 让上层继续快速失败
		return "", errors.New("should not be called")
	}
	t.Cleanup(func() { runScriptFn = origRunScript })

	inst := &model.Instance{
		InstanceId:  "ins-hermes-hook-noready",
		AgentType:   model.AgentTypeDeepSeekTUI, // 未注册 check_ready → ResolveScript 直接 error
		RuntimeUser: "root",
		RuntimeHome: "/root",
	}
	err := runHermesUpgradePostHooks(context.Background(), inst)
	if err != nil {
		t.Fatalf("expected nil (warn-only, not blocking upgrade), got %v", err)
	}
}

// ============================================================================
// runOpenClawUpgradePostHooks
// ============================================================================

// TestRunOpenClawUpgradePostHooks_AllSteps 验证：
// 5 项后置补丁按顺序被下发；即使其中若干项失败，函数也不返回错误、不 panic。
// 通过 scriptName 分类计数，断言核心 3 个同步脚本各被调用 1 次；approveDeviceAfterUpgrade
// 走 approveDeviceAfterUpgradeFn 桩替换（同步调用，无需 goroutine 等待）。
func TestRunOpenClawUpgradePostHooks_AllSteps(t *testing.T) {
	setupUpgradeExtraEnv(t)

	var (
		syncGwPortCalls int32
		pluginFixCalls  int32
		compatCalls     int32
		cleanupCalls    int32
		approveCalls    int32
		approveDone     = make(chan struct{})
	)
	origRunScript := runScriptFn
	runScriptFn = func(ctx context.Context, _, scriptName string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		switch scriptName {
		case "sync_gateway_port.sh":
			atomic.AddInt32(&syncGwPortCalls, 1)
			// 模拟失败：应仅 warn，不阻断
			return "", errors.New("simulated port sync failure")
		case "restore_plugin_node_modules.sh":
			atomic.AddInt32(&pluginFixCalls, 1)
			return "ok", nil
		case "compat_installs_json.sh", "compat_plugins.sh":
			atomic.AddInt32(&compatCalls, 1)
			return "ok", nil
		case "cleanup_upgrade_temp.sh":
			atomic.AddInt32(&cleanupCalls, 1)
			return "ok", nil
		default:
			return "", nil
		}
	}
	origApprove := approveDeviceAfterUpgradeFn
	approveDeviceAfterUpgradeFn = func(_ context.Context, _ *model.Instance) {
		atomic.AddInt32(&approveCalls, 1)
		close(approveDone)
	}
	t.Cleanup(func() {
		runScriptFn = origRunScript
		approveDeviceAfterUpgradeFn = origApprove
	})

	inst := &model.Instance{
		InstanceId:  "ins-openclaw-hooks",
		AgentType:   model.AgentTypeOpenClaw,
		RuntimeUser: "root",
		RuntimeHome: "/root",
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("runOpenClawUpgradePostHooks 不应 panic，实际=%v", r)
		}
	}()
	runOpenClawUpgradePostHooks(context.Background(), inst)

	// approveDeviceAfterUpgrade 已改为同步调用，runOpenClawUpgradePostHooks 返回前 approveDone 已 close
	select {
	case <-approveDone:
	case <-time.After(2 * time.Second):
		t.Fatalf("approveDeviceAfterUpgradeFn 未在 2s 内被调用")
	}

	if got := atomic.LoadInt32(&syncGwPortCalls); got != 1 {
		t.Errorf("sync_gateway_port.sh 期望被调用 1 次（失败），实际=%d", got)
	}
	if got := atomic.LoadInt32(&pluginFixCalls); got != 1 {
		t.Errorf("restore_plugin_node_modules.sh 期望被调用 1 次，实际=%d", got)
	}
	if got := atomic.LoadInt32(&compatCalls); got < 1 {
		t.Errorf("compat_* 脚本期望至少调用 1 次，实际=%d", got)
	}
	if got := atomic.LoadInt32(&cleanupCalls); got != 1 {
		t.Errorf("cleanup_upgrade_temp.sh 期望被调用 1 次，实际=%d", got)
	}
	if got := atomic.LoadInt32(&approveCalls); got != 1 {
		t.Errorf("approveDeviceAfterUpgradeFn 期望被调用 1 次，实际=%d", got)
	}
}

// TestRunOpenClawUpgradePostHooks_NilContextSafe 防御性测试：
// context.Background 场景下不 panic；approveDeviceAfterUpgrade 已改为同步，无需异步等待。
func TestRunOpenClawUpgradePostHooks_NilContextSafe(t *testing.T) {
	setupUpgradeExtraEnv(t)

	origRunScript := runScriptFn
	runScriptFn = func(ctx context.Context, _, scriptName string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		if strings.HasPrefix(scriptName, "check_") {
			return `{"ready": true}`, nil
		}
		return "", nil
	}
	approveDone := make(chan struct{})
	origApprove := approveDeviceAfterUpgradeFn
	approveDeviceAfterUpgradeFn = func(_ context.Context, _ *model.Instance) {
		close(approveDone)
	}
	t.Cleanup(func() {
		runScriptFn = origRunScript
		approveDeviceAfterUpgradeFn = origApprove
	})

	inst := &model.Instance{
		InstanceId: "ins-openclaw-hooks-bg",
		AgentType:  model.AgentTypeOpenClaw,
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("runOpenClawUpgradePostHooks 在 context.Background 下不应 panic，实际=%v", r)
		}
	}()
	runOpenClawUpgradePostHooks(context.Background(), inst)

	select {
	case <-approveDone:
	case <-time.After(2 * time.Second):
		t.Fatalf("approveDeviceAfterUpgradeFn 未在 2s 内被调用")
	}
}
