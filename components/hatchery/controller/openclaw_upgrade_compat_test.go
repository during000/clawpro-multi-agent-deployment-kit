package controller

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ============================================================================
// runCompatScripts 单元测试
// ============================================================================

// TestRunCompatScripts_AllSuccess 验证：两个兼容脚本均执行成功时，runScriptFn 被调用 2 次，无错误日志。
func TestRunCompatScripts_AllSuccess(t *testing.T) {
	setupUpgradeExtraEnv(t)

	var callCount int32
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		atomic.AddInt32(&callCount, 1)
		return "ok", nil
	}

	inst := &model.Instance{InstanceId: "ins-compat-ok", RuntimeUser: "agentuser"}
	runCompatScripts(context.Background(), inst)

	if got := atomic.LoadInt32(&callCount); got != 2 {
		t.Errorf("期望 runScriptFn 被调用 2 次（两个兼容脚本），实际=%d", got)
	}
}

// TestRunCompatScripts_AllFail 验证：两个兼容脚本均失败时，runScriptFn 仍被调用 2 次（失败不中断循环）。
func TestRunCompatScripts_AllFail(t *testing.T) {
	setupUpgradeExtraEnv(t)

	var callCount int32
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		atomic.AddInt32(&callCount, 1)
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	}

	inst := &model.Instance{InstanceId: "ins-compat-fail", RuntimeUser: "agentuser"}
	runCompatScripts(context.Background(), inst)

	if got := atomic.LoadInt32(&callCount); got != 2 {
		t.Errorf("脚本失败不应中断循环，期望调用 2 次，实际=%d", got)
	}
}

// TestRunCompatScripts_PartialFail 验证：第一个脚本失败、第二个成功，两个脚本均被执行。
func TestRunCompatScripts_PartialFail(t *testing.T) {
	setupUpgradeExtraEnv(t)

	var calledScripts []string
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		calledScripts = append(calledScripts, scriptName)
		if strings.Contains(scriptName, "compat_installs_json") {
			return "", hcommon.I18nError(i18n.MsgOperationFailed)
		}
		return "ok", nil
	}

	inst := &model.Instance{InstanceId: "ins-compat-partial", RuntimeUser: "agentuser"}
	runCompatScripts(context.Background(), inst)

	if len(calledScripts) != 2 {
		t.Errorf("期望执行 2 个脚本，实际=%d: %v", len(calledScripts), calledScripts)
	}
}

// TestRunCompatScripts_ScriptNames 验证：runCompatScripts 依次执行 compat_installs_json.sh 和 compat_plugins.sh。
func TestRunCompatScripts_ScriptNames(t *testing.T) {
	setupUpgradeExtraEnv(t)

	var calledScripts []string
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		calledScripts = append(calledScripts, scriptName)
		return "", nil
	}

	inst := &model.Instance{InstanceId: "ins-compat-names", RuntimeUser: "agentuser"}
	runCompatScripts(context.Background(), inst)

	wantScripts := []string{"compat_installs_json.sh", "compat_plugins.sh"}
	for i, want := range wantScripts {
		if i >= len(calledScripts) {
			t.Errorf("第 %d 个脚本未被调用，期望=%s", i+1, want)
			continue
		}
		if calledScripts[i] != want {
			t.Errorf("第 %d 个脚本名称错误，期望=%s，实际=%s", i+1, want, calledScripts[i])
		}
	}
}

// ============================================================================
// approveDeviceAfterUpgrade 单元测试
//
// 注意：本函数自 v0.x.x 起在 approve_device.sh 之前会先调用 waitForOpenclawReady，
// 后者通过 ResolveScript("check_ready", agentType) + runScriptFn 探测就绪状态。
// 所以测试里 mock runScriptFn 时需要按 scriptName 区分：
//   - check_ready.sh / check_ready_*.sh → 返回 `{"ready": true}` 让 wait 立即通过
//   - approve_device.sh → 按用例意图返回成功 / 失败
// ============================================================================

// fakeReadyOutput 返回一段尾行为 `{"ready": true}` 的 stdout，
// waitForOpenclawReady 内部会从输出最后一行 JSON 中提取 ready。
const fakeReadyOutput = `some leading log
{"ready": true}`

// fakeNotReadyOutput 用于让 waitForOpenclawReady 第一次循环判定为 not ready，
// 配合 ctx cancel 在第二次循环 select 入口快速返回，避免 30s 等待。
const fakeNotReadyOutput = `boot in progress
{"ready": false, "reason": "gateway_not_active"}`

// TestApproveDeviceAfterUpgrade_Success 验证：waitForOpenclawReady + approve_device.sh
// 都成功时，approveDeviceAfterUpgrade 正常返回，且确实下发了 approve_device.sh。
func TestApproveDeviceAfterUpgrade_Success(t *testing.T) {
	setupUpgradeExtraEnv(t)

	var (
		mu              sync.Mutex
		approveCalled   bool
		checkReadyCalls int
	)
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		// waitForOpenclawReady 探测：返回 ready=true 让其立即跳出循环。
		// 注意：ResolveScript("check_ready", AgentTypeOpenClaw) 解析后的真实脚本名是
		// "check_openclaw_ready.sh"，必须显式匹配该名字，否则 mock 不命中会让 wait 循环
		// 跑满 5 分钟超时，导致 go test 包级 timeout 被触发。
		if scriptName == "check_openclaw_ready.sh" {
			checkReadyCalls++
			return fakeReadyOutput, nil
		}
		if scriptName == "approve_device.sh" {
			approveCalled = true
			return "approved", nil
		}
		return "", nil
	}

	inst := &model.Instance{InstanceId: "ins-approve-ok", AgentType: model.AgentTypeOpenClaw, RuntimeUser: "agentuser"}
	approveDeviceAfterUpgrade(context.Background(), inst)

	mu.Lock()
	defer mu.Unlock()
	if checkReadyCalls < 1 {
		t.Errorf("期望 waitForOpenclawReady 至少触发 1 次 check_ready 调用，实际=%d", checkReadyCalls)
	}
	if !approveCalled {
		t.Error("waitForOpenclawReady 成功后应调用 approve_device.sh")
	}
}

// TestApproveDeviceAfterUpgrade_ApproveFail 验证：waitForOpenclawReady 通过后 approve_device.sh
// 执行失败时，approveDeviceAfterUpgrade 不 panic、不 return error（仅 Warn 日志），且仍会继续
// 执行 imageModel 自愈步骤（这里 syncInstanceModelsToCVM 因测试 DB 无 site config 也会失败，
// 但同样仅 Warn，不影响函数返回）。
func TestApproveDeviceAfterUpgrade_ApproveFail(t *testing.T) {
	setupUpgradeExtraEnv(t)

	var approveCalled bool
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		// 显式匹配 ResolveScript("check_ready", AgentTypeOpenClaw) 解析出的真实脚本名
		if scriptName == "check_openclaw_ready.sh" {
			return fakeReadyOutput, nil
		}
		if scriptName == "approve_device.sh" {
			approveCalled = true
			return "", hcommon.I18nError(i18n.MsgOperationFailed)
		}
		return "", nil
	}

	inst := &model.Instance{InstanceId: "ins-approve-fail", AgentType: model.AgentTypeOpenClaw, RuntimeUser: "agentuser"}
	// 失败时不应 panic，函数正常返回
	approveDeviceAfterUpgrade(context.Background(), inst)

	if !approveCalled {
		t.Error("waitForOpenclawReady 成功后应调用 approve_device.sh（即使最终失败）")
	}
}

// TestApproveDeviceAfterUpgrade_WaitOpenclawReadyTimeout 验证：waitForOpenclawReady 失败
// （ctx 提前 cancel 模拟超时）时，approveDeviceAfterUpgrade 跳过 approve_device.sh，
// 但仍继续执行 imageModel 自愈（这是 v0.x.x hotfix 的关键策略：wait 超时 ≠ 升级失败）。
func TestApproveDeviceAfterUpgrade_WaitOpenclawReadyTimeout(t *testing.T) {
	setupUpgradeExtraEnv(t)

	var (
		mu            sync.Mutex
		approveCalled bool
	)
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		// 让 check_ready 始终判定 not ready，触发 wait 内部循环。
		// 必须显式匹配 ResolveScript 解析后的脚本名 "check_openclaw_ready.sh"。
		if scriptName == "check_openclaw_ready.sh" {
			return fakeNotReadyOutput, nil
		}
		if scriptName == "approve_device.sh" {
			approveCalled = true
			return "approved", nil
		}
		return "", nil
	}

	// ctx 立即 cancel：第一次循环 attempt==1 不 select 直接跑 runScriptFn（not ready），
	// 第二次循环 attempt>1 进入 select(<-ctx.Done()) 立即返回 err，wait 整体 < 1ms 失败。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	inst := &model.Instance{InstanceId: "ins-wait-timeout", AgentType: model.AgentTypeOpenClaw, RuntimeUser: "agentuser"}
	approveDeviceAfterUpgrade(ctx, inst)

	mu.Lock()
	defer mu.Unlock()
	if approveCalled {
		t.Error("waitForOpenclawReady 失败时不应再调用 approve_device.sh")
	}
}
