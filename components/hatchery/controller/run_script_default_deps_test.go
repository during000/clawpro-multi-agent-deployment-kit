package controller

import (
	"context"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
)

// 本文件覆盖一批「默认 TAT 脚本执行依赖」包装函数的成功 / 失败两条分支。
//
// 这些包装函数（defaultPerformMigrationImportDeps.RunRestoreScript、
// defaultMigrationModelsDependencies.RunScript、
// defaultLightClawHandlerDependencies.RunScript、precheckRunScriptFn）
// 本质上是对底层 agentScriptRunner（默认绑定真实 RunScript）的薄封装：
//
//	output, rerr := agentScriptRunner(...)
//	if rerr != nil { return ..., rerr }
//	return output, nil
//
// 生产路径无法在单测中让真实 RunScript 成功（需要凭据 + 在线 Agent + TAT 网络），
// 因此这里复用既有测试桩 withAgentScriptRunner 直接替换 agentScriptRunner，
// 分别注入成功与失败结果，覆盖 if 分支与最终 return 分支。

// ─── openclaw_migration.go: defaultPerformMigrationImportDeps.RunRestoreScript ───

func TestDefaultRunRestoreScript_Success(t *testing.T) {
	withAgentScriptRunner(t, func(_ context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return "RESTORE_DONE:1", nil
	})

	err := defaultPerformMigrationImportDeps.RunRestoreScript(
		context.Background(), "ins-restore", "restore_migration.sh", 600, "root", nil)
	if err != nil {
		t.Fatalf("RunScript 成功时应返回 nil，实际=%v", err)
	}
}

func TestDefaultRunRestoreScript_Error(t *testing.T) {
	withAgentScriptRunner(t, func(_ context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	})

	err := defaultPerformMigrationImportDeps.RunRestoreScript(
		context.Background(), "ins-restore", "restore_migration.sh", 600, "root", nil)
	if err == nil {
		t.Fatal("RunScript 失败时应返回错误，实际=nil")
	}
}

// onOutput 透传：确认包装函数把回调原样交给底层 runner。
func TestDefaultRunRestoreScript_OnOutputPassthrough(t *testing.T) {
	var gotCallback bool
	withAgentScriptRunner(t, func(_ context.Context, _, _ string, _ uint64, _ string, onOutput func(chunk string), _ map[string]string) (string, error) {
		if onOutput != nil {
			onOutput("PROGRESS:1")
		}
		return "ok", nil
	})

	called := false
	err := defaultPerformMigrationImportDeps.RunRestoreScript(
		context.Background(), "ins-restore", "restore_migration.sh", 600, "root",
		func(string) { called = true })
	if err != nil {
		t.Fatalf("期望成功，实际=%v", err)
	}
	gotCallback = called
	if !gotCallback {
		t.Error("onOutput 回调应被底层 runner 调用")
	}
}

// ─── openclaw_migration.go: defaultMigrationModelsDependencies.RunScript ───

func TestDefaultMigrationModelsDeps_RunScript_Success(t *testing.T) {
	withAgentScriptRunner(t, func(_ context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return "MODELS:[]", nil
	})

	out, err := defaultMigrationModelsDependencies{}.RunScript(
		context.Background(), "ins-models", "extract_migration_models.sh", 120, "root")
	if err != nil {
		t.Fatalf("期望成功，实际=%v", err)
	}
	if out != "MODELS:[]" {
		t.Errorf("输出应原样透传，实际=%q", out)
	}
}

func TestDefaultMigrationModelsDeps_RunScript_Error(t *testing.T) {
	withAgentScriptRunner(t, func(_ context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return "partial-output", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	})

	out, err := defaultMigrationModelsDependencies{}.RunScript(
		context.Background(), "ins-models", "extract_migration_models.sh", 120, "root")
	if err == nil {
		t.Fatal("RunScript 失败时应返回错误，实际=nil")
	}
	// 失败时仍应把已有 output 一并返回，便于上层解析 stderr/stdout。
	if out != "partial-output" {
		t.Errorf("失败时也应透传 output，实际=%q", out)
	}
}

// ─── lightclaw.go: defaultLightClawHandlerDependencies.RunScript ───

func TestDefaultLightClawDeps_RunScript_Success(t *testing.T) {
	withAgentScriptRunner(t, func(_ context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return `{"code":0}`, nil
	})

	out, err := defaultLightClawHandlerDependencies{}.RunScript(
		context.Background(), "ins-lc", "lightclaw_token.sh", 30, "root", nil, nil)
	if err != nil {
		t.Fatalf("期望成功，实际=%v", err)
	}
	if out != `{"code":0}` {
		t.Errorf("输出应原样透传，实际=%q", out)
	}
}

func TestDefaultLightClawDeps_RunScript_Error(t *testing.T) {
	withAgentScriptRunner(t, func(_ context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	})

	_, err := defaultLightClawHandlerDependencies{}.RunScript(
		context.Background(), "ins-lc", "lightclaw_token.sh", 30, "root", nil, nil)
	if err == nil {
		t.Fatal("RunScript 失败时应返回错误，实际=nil")
	}
}

// ─── memory_precheck.go: precheckRunScriptFn 默认实现 ───
//
// 注意：其他用例通过 mockPrecheckRunScript 整体替换 precheckRunScriptFn，
// 因此其默认函数体从不执行。这里不替换 precheckRunScriptFn 本身，而是替换它内部
// 依赖的 agentScriptRunner，从而真正执行默认体并覆盖成功/失败两条分支。

func TestPrecheckRunScriptFnDefault_Success(t *testing.T) {
	withAgentScriptRunner(t, func(_ context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return `{"reachable":true}`, nil
	})

	out, err := precheckRunScriptFn(
		context.Background(), "ins-pc", "precheck_vdb.sh", 10, "root", nil,
		map[string]string{"vdb_endpoint": "http://10.0.0.18:80"})
	if err != nil {
		t.Fatalf("期望成功，实际=%v", err)
	}
	if out != `{"reachable":true}` {
		t.Errorf("输出应原样透传，实际=%q", out)
	}
}

func TestPrecheckRunScriptFnDefault_Error(t *testing.T) {
	withAgentScriptRunner(t, func(_ context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return "stderr-detail", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	})

	out, err := precheckRunScriptFn(
		context.Background(), "ins-pc", "precheck_vdb.sh", 10, "root", nil, nil)
	if err == nil {
		t.Fatal("RunScript 失败时应返回错误，实际=nil")
	}
	if out != "stderr-detail" {
		t.Errorf("失败时也应透传 output，实际=%q", out)
	}
}
