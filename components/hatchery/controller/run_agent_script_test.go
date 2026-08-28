package controller

import (
	"context"
	"errors"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// withAgentScriptRunner 把 agentScriptRunner 替换为 stub，并通过 Cleanup 自动还原。
// 避免测试之间的交叉污染 + 避免真正触发 TAT。
func withAgentScriptRunner(t *testing.T, stub func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error)) {
	t.Helper()
	orig := agentScriptRunner
	agentScriptRunner = stub
	t.Cleanup(func() { agentScriptRunner = orig })
}

// TestRunAgentScript_ResolveSuccess 验证正常路径：Resolve 出脚本名并透传到 runner。
func TestRunAgentScript_ResolveSuccess(t *testing.T) {
	var gotScript, gotInstanceID, gotRuntimeUser string
	var gotTimeout uint64
	var gotParams map[string]string

	withAgentScriptRunner(t, func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		gotInstanceID = instanceId
		gotScript = scriptName
		gotTimeout = timeout
		gotRuntimeUser = runtimeUser
		gotParams = params
		return "stdout-ok", nil
	})

	inst := &model.Instance{
		InstanceId:  "ins-xxx",
		AgentType:   model.AgentTypeHermes,
		RuntimeUser: "ubuntu",
	}
	params := map[string]string{"k": "v"}

	out, err := RunAgentScript(context.Background(), inst, "set_model", 60, nil, params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if out != "stdout-ok" {
		t.Errorf("output mismatch: got %q", out)
	}
	// Hermes 的 set_model 应分派到 set_model_hermes.sh
	if gotScript != "set_model_hermes.sh" {
		t.Errorf("scriptName dispatch wrong: got %q, want set_model_hermes.sh", gotScript)
	}
	if gotInstanceID != "ins-xxx" || gotRuntimeUser != "ubuntu" || gotTimeout != 60 {
		t.Errorf("runner args wrong: id=%q user=%q timeout=%d", gotInstanceID, gotRuntimeUser, gotTimeout)
	}
	if gotParams["k"] != "v" {
		t.Errorf("params not propagated: %#v", gotParams)
	}
}

// TestRunAgentScript_ResolveFailed 验证 Resolve 阶段失败会包装成 ErrScriptResolveFailed，
// 且**不会**调用 runner（fail-fast）。
func TestRunAgentScript_ResolveFailed(t *testing.T) {
	called := false
	withAgentScriptRunner(t, func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		called = true
		return "", nil
	})

	inst := &model.Instance{
		InstanceId: "ins-xxx",
		AgentType:  model.AgentTypeHermes,
	}
	// qq_bot_creator 仅 openclaw 支持，hermes 不支持 → ResolveScript 返回 "not supported"
	// （weixin_bot_creator 在新版 scriptResolveTable 中 Hermes 已注册 weixin_bot_creator_hermes.sh，不能再用作 fail-closed 用例）
	_, err := RunAgentScript(context.Background(), inst, "qq_bot_creator", 60, nil, nil)
	if err == nil {
		t.Fatal("expected resolve error, got nil")
	}
	if !errors.Is(err, ErrScriptResolveFailed) {
		t.Errorf("expected ErrScriptResolveFailed, got: %v", err)
	}
	if errors.Is(err, ErrScriptRunFailed) {
		t.Errorf("resolve error should not be classified as ErrScriptRunFailed: %v", err)
	}
	if called {
		t.Error("runner should NOT be invoked when resolve fails")
	}
}

// TestRunAgentScript_RunFailed 验证 runner 错误会被 ErrScriptRunFailed 包装，
// 且原始错误（含 RichError）可通过 errors.Unwrap 取回。
func TestRunAgentScript_RunFailed(t *testing.T) {
	rawErr := hcommon.I18nError(i18n.MsgTATFailed)
	withAgentScriptRunner(t, func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "", rawErr
	})

	inst := &model.Instance{
		InstanceId: "ins-xxx",
		AgentType:  model.AgentTypeOpenClaw,
	}
	_, err := RunAgentScript(context.Background(), inst, "set_model", 60, nil, nil)
	if err == nil {
		t.Fatal("expected run error, got nil")
	}
	if !errors.Is(err, ErrScriptRunFailed) {
		t.Errorf("expected ErrScriptRunFailed, got: %v", err)
	}
	if errors.Is(err, ErrScriptResolveFailed) {
		t.Errorf("run error should not be classified as ErrScriptResolveFailed: %v", err)
	}
	// 原始 RichError 应可通过 errors.As / Unwrap 恢复
	if !errors.Is(err, rawErr) {
		t.Errorf("raw error should be retrievable via errors.Is: got %v, want wraps %v", err, rawErr)
	}
}

// TestRunAgentScript_NilInstance 防御性：传 nil 不应 panic，应返回 ErrScriptResolveFailed。
func TestRunAgentScript_NilInstance(t *testing.T) {
	withAgentScriptRunner(t, func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		t.Error("runner must not be called when instance is nil")
		return "", nil
	})

	_, err := RunAgentScript(context.Background(), nil, "set_model", 60, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil instance, got nil")
	}
	if !errors.Is(err, ErrScriptResolveFailed) {
		t.Errorf("nil instance should produce ErrScriptResolveFailed, got: %v", err)
	}
}

// TestRunAgentScript_EmptyAgentTypeFallback 空 agent_type 应按 ResolveScript 语义 fallback 到 openclaw。
func TestRunAgentScript_EmptyAgentTypeFallback(t *testing.T) {
	var gotScript string
	withAgentScriptRunner(t, func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		gotScript = scriptName
		return "ok", nil
	})

	inst := &model.Instance{InstanceId: "ins-legacy", AgentType: ""}
	if _, err := RunAgentScript(context.Background(), inst, "set_model", 60, nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotScript != "set_model.sh" {
		t.Errorf("empty agent_type should fallback to openclaw/set_model.sh, got %q", gotScript)
	}
}
