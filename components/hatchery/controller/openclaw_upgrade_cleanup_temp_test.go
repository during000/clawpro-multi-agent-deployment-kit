// openclaw_upgrade_cleanup_temp_test.go
//
// 针对 cleanupUpgradeTemp 的单元测试。
// 该函数是 performUpgrade 数据恢复后的"收尾清理步骤"：
//   - 通过 TAT 下发 cleanup_upgrade_temp.sh，清理 /tmp 残留压缩包 + ~/.openclaw/upgrades/ 旧快照
//   - 失败不阻断升级主流程（仅 Warn 日志），不返回错误
//
// 覆盖以下分支（确保新增代码行覆盖率 ≥ 60%，实际可达 100%）：
//   - runScriptFn 成功     → 走 else 分支，记录"清理完成"
//   - runScriptFn 失败     → 走 if 分支，记录 Warn，函数本身不返回错误、不 panic
//   - 调用参数透传正确性   → scriptName / timeout / runtimeUser / params 均按约定下发
//   - context.Background() 防御性冒烟（避免 Logger 兜底缺失被回归引入）
package controller

import (
	"context"
	"sync/atomic"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// TestCleanupUpgradeTemp_Success 验证：runScriptFn 成功返回时，
// cleanupUpgradeTemp 走"清理完成"分支，函数正常返回；runScriptFn 被调用 1 次。
func TestCleanupUpgradeTemp_Success(t *testing.T) {
	cleanup := setupUpgradeExtraEnv(t)
	defer cleanup()

	var callCount int32
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		atomic.AddInt32(&callCount, 1)
		return "ok", nil
	}

	inst := &model.Instance{InstanceId: "ins-cleanup-ok", RuntimeUser: "agentuser"}
	// 不应 panic，无返回值（失败也吞）
	cleanupUpgradeTemp(context.Background(), inst)

	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Errorf("期望 runScriptFn 被调用 1 次，实际=%d", got)
	}
}

// TestCleanupUpgradeTemp_Fail 验证：runScriptFn 失败时，cleanupUpgradeTemp 不 panic、
// 不返回错误（仅记录 Warn）；runScriptFn 仍被调用 1 次。
// 这是"失败不阻断升级主流程"的核心保证 —— 收尾清理失败不应让整个升级被判定失败。
func TestCleanupUpgradeTemp_Fail(t *testing.T) {
	cleanup := setupUpgradeExtraEnv(t)
	defer cleanup()

	var callCount int32
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		atomic.AddInt32(&callCount, 1)
		return "", hcommon.I18nError(i18n.MsgTATCommandDispatchFailed)
	}

	inst := &model.Instance{InstanceId: "ins-cleanup-fail", RuntimeUser: "agentuser"}
	// 失败时不应 panic，函数正常返回
	cleanupUpgradeTemp(context.Background(), inst)

	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Errorf("期望 runScriptFn 被调用 1 次（失败也只调用 1 次），实际=%d", got)
	}
}

// TestCleanupUpgradeTemp_ScriptParams 验证：cleanupUpgradeTemp 下发的参数与设计一致：
//   - scriptName  = cleanup_upgrade_temp.sh
//   - timeout     = 120s（清理动作轻量，无需长超时）
//   - runtimeUser = instance.RuntimeUser（以 runtimeUser 身份执行，不进 root 白名单）
//   - params      = nil（脚本无需入参，固定保留最近 3 个快照）
//   - instanceId  = instance.InstanceId
func TestCleanupUpgradeTemp_ScriptParams(t *testing.T) {
	cleanup := setupUpgradeExtraEnv(t)
	defer cleanup()

	var (
		gotInstanceId  string
		gotScriptName  string
		gotTimeout     uint64
		gotRuntimeUser string
		gotParams      map[string]string
		gotParamsSeen  bool
	)
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		gotInstanceId = instanceId
		gotScriptName = scriptName
		gotTimeout = timeout
		gotRuntimeUser = runtimeUser
		gotParams = params
		gotParamsSeen = true
		return "ok", nil
	}

	inst := &model.Instance{InstanceId: "ins-cleanup-params", RuntimeUser: "agentuser"}
	cleanupUpgradeTemp(context.Background(), inst)

	if !gotParamsSeen {
		t.Fatalf("runScriptFn 未被调用")
	}
	if gotInstanceId != "ins-cleanup-params" {
		t.Errorf("instanceId 不一致：want=ins-cleanup-params, got=%s", gotInstanceId)
	}
	if gotScriptName != "cleanup_upgrade_temp.sh" {
		t.Errorf("scriptName 不一致：want=cleanup_upgrade_temp.sh, got=%s", gotScriptName)
	}
	if gotTimeout != 120 {
		t.Errorf("timeout 不一致：want=120, got=%d", gotTimeout)
	}
	if gotRuntimeUser != "agentuser" {
		t.Errorf("runtimeUser 不一致：want=agentuser, got=%s", gotRuntimeUser)
	}
	if gotParams != nil {
		t.Errorf("params 期望为 nil（脚本无需入参），实际=%v", gotParams)
	}
}

// TestCleanupUpgradeTemp_NilContextSafe 防御性测试：context.Background() 是合法值，
// 不会因 ctx 内无 logger 而 panic（Logger(ctx) 必须有兜底）。
// 仅作为冒烟测试存在，避免后续重构中误把 ctx 必传字段化导致 cleanupUpgradeTemp 在升级流程中 panic。
func TestCleanupUpgradeTemp_NilContextSafe(t *testing.T) {
	cleanup := setupUpgradeExtraEnv(t)
	defer cleanup()

	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", nil
	}

	inst := &model.Instance{InstanceId: "ins-cleanup-bg", RuntimeUser: "agentuser"}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("cleanupUpgradeTemp 在 context.Background 下不应 panic，实际 recover=%v", r)
		}
	}()
	cleanupUpgradeTemp(context.Background(), inst)
}
