// openclaw_upgrade_plugin_fix_test.go
//
// 针对 fixPluginNodeModules 的单元测试。
// 该函数是 performUpgrade 数据恢复后的"补丁步骤"：
//   - 通过 TAT 下发 restore_plugin_node_modules.sh，修复 extensions/*/node_modules 与 openclaw 软链
//   - 失败不阻断升级主流程（仅 Warn 日志），不返回错误
//
// 覆盖以下分支：
//   - runScriptFn 成功     → 走 else 分支，记录"修复完成"
//   - runScriptFn 失败     → 走 if 分支，记录 Warn，函数本身不返回错误、不 panic
//   - 调用参数透传正确性   → scriptName / timeout / runtimeUser / params 均按约定下发
package controller

import (
	"context"
	"sync/atomic"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// TestFixPluginNodeModules_Success 验证：runScriptFn 成功返回时，
// fixPluginNodeModules 走"修复完成"分支，函数正常返回；runScriptFn 被调用 1 次。
func TestFixPluginNodeModules_Success(t *testing.T) {
	cleanup := setupUpgradeExtraEnv(t)
	defer cleanup()

	var callCount int32
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		atomic.AddInt32(&callCount, 1)
		return "ok", nil
	}

	inst := &model.Instance{InstanceId: "ins-plugin-fix-ok", RuntimeUser: "agentuser"}
	// 不应 panic，无返回值（失败也吞）
	fixPluginNodeModules(context.Background(), inst)

	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Errorf("期望 runScriptFn 被调用 1 次，实际=%d", got)
	}
}

// TestFixPluginNodeModules_Fail 验证：runScriptFn 失败时，fixPluginNodeModules 不 panic、
// 不返回错误（仅记录 Warn）；runScriptFn 仍被调用 1 次。
// 这是"失败不阻断升级主流程"的核心保证。
func TestFixPluginNodeModules_Fail(t *testing.T) {
	cleanup := setupUpgradeExtraEnv(t)
	defer cleanup()

	var callCount int32
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		atomic.AddInt32(&callCount, 1)
		return "", hcommon.I18nError(i18n.MsgTATCommandDispatchFailed)
	}

	inst := &model.Instance{InstanceId: "ins-plugin-fix-fail", RuntimeUser: "agentuser"}
	// 失败时不应 panic，函数正常返回
	fixPluginNodeModules(context.Background(), inst)

	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Errorf("期望 runScriptFn 被调用 1 次（失败也只调用 1 次），实际=%d", got)
	}
}

// TestFixPluginNodeModules_ScriptParams 验证：fixPluginNodeModules 下发的参数与设计一致：
//   - scriptName = restore_plugin_node_modules.sh
//   - timeout    = 600s
//   - runtimeUser = instance.RuntimeUser（以 runtimeUser 身份执行，不进 root 白名单）
//   - params     = 非 nil 空 map（与代码实现一致，避免 nil map 在下游解引用 panic）
//   - instanceId = instance.InstanceId
func TestFixPluginNodeModules_ScriptParams(t *testing.T) {
	cleanup := setupUpgradeExtraEnv(t)
	defer cleanup()

	var (
		gotInstanceId  string
		gotScriptName  string
		gotTimeout     uint64
		gotRuntimeUser string
		gotParams      map[string]string
		gotParamsNil   = true
	)
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		gotInstanceId = instanceId
		gotScriptName = scriptName
		gotTimeout = timeout
		gotRuntimeUser = runtimeUser
		gotParams = params
		if params != nil {
			gotParamsNil = false
		}
		return "ok", nil
	}

	inst := &model.Instance{InstanceId: "ins-plugin-fix-params", RuntimeUser: "agentuser"}
	fixPluginNodeModules(context.Background(), inst)

	if gotInstanceId != "ins-plugin-fix-params" {
		t.Errorf("instanceId 不一致：want=ins-plugin-fix-params, got=%s", gotInstanceId)
	}
	if gotScriptName != "restore_plugin_node_modules.sh" {
		t.Errorf("scriptName 不一致：want=restore_plugin_node_modules.sh, got=%s", gotScriptName)
	}
	if gotTimeout != 600 {
		t.Errorf("timeout 不一致：want=600, got=%d", gotTimeout)
	}
	if gotRuntimeUser != "agentuser" {
		t.Errorf("runtimeUser 不一致：want=agentuser, got=%s", gotRuntimeUser)
	}
	if gotParamsNil {
		t.Errorf("params 期望非 nil（空 map），实际=nil")
	}
	if len(gotParams) != 0 {
		t.Errorf("params 期望为空 map，实际=%v", gotParams)
	}
}

// TestFixPluginNodeModules_NilContextSafe 防御性测试：context.Background() 是合法值，
// 不会因 ctx 内无 logger 而 panic（Logger(ctx) 必须有兜底）。
// 仅作为冒烟测试存在，避免后续重构中误把 ctx 必传字段化导致 fixPluginNodeModules 在升级流程中 panic。
func TestFixPluginNodeModules_NilContextSafe(t *testing.T) {
	cleanup := setupUpgradeExtraEnv(t)
	defer cleanup()

	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", nil
	}

	inst := &model.Instance{InstanceId: "ins-plugin-fix-bg", RuntimeUser: "agentuser"}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("fixPluginNodeModules 在 context.Background 下不应 panic，实际 recover=%v", r)
		}
	}()
	fixPluginNodeModules(context.Background(), inst)
}
