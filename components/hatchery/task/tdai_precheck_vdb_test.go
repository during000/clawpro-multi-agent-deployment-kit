package task

import (
	"context"
	"errors"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
)

// 集中验证 precheckVDBConnectivity 的 5 种语义路径：
//
//  1. endpoint 为空           → NonRetryable
//  2. 脚本回 reachable=true   → nil（继续）
//  3. 脚本回 reachable=false + 真探测工具 → NonRetryable（关键场景）
//  4. 脚本回 reachable=false + probe=none → nil（保守通过）
//  5. 脚本调用失败/输出非法   → 普通 error（可重试）
func TestPrecheckVDBConnectivity_EmptyEndpoint(t *testing.T) {
	setupTestDB(t)
	err := precheckVDBConnectivity(context.Background(), "inst-x", "", "user", "key")
	if err == nil {
		t.Fatal("空 endpoint 应返回错误")
	}
	var nre *NonRetryableError
	if !errors.As(err, &nre) {
		t.Fatalf("空 endpoint 应返回 NonRetryableError，got %T: %v", err, err)
	}
}

func TestPrecheckVDBConnectivity_Reachable(t *testing.T) {
	setupTestDB(t)
	orig := taskRunScriptFn
	defer func() { taskRunScriptFn = orig }()
	taskRunScriptFn = func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return `[precheck-vdb] OK
{"reachable":true,"host":"10.0.0.49","port":80,"probe":"nc"}`, nil
	}
	if err := precheckVDBConnectivity(context.Background(), "inst-x", "http://10.0.0.49:80", "user", "key"); err != nil {
		t.Fatalf("reachable=true 应返回 nil，got: %v", err)
	}
}

func TestPrecheckVDBConnectivity_Unreachable_NonRetryable(t *testing.T) {
	setupTestDB(t)
	orig := taskRunScriptFn
	defer func() { taskRunScriptFn = orig }()
	// 模拟真实 TAT 行为：脚本退出码非 0 时 output="" + stdout 进 RichError.Detail
	taskRunScriptFn = func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail(`[precheck-vdb] FAIL: 10.0.0.49:80 NOT reachable (probe=nc)
{"reachable":false,"host":"10.0.0.49","port":80,"probe":"nc"}`)
	}
	err := precheckVDBConnectivity(context.Background(), "inst-x", "http://10.0.0.49:80", "user", "key")
	if err == nil {
		t.Fatal("reachable=false 应返回 NonRetryableError")
	}
	var nre *NonRetryableError
	if !errors.As(err, &nre) {
		t.Fatalf("reachable=false 必须是 NonRetryableError（避免无意义重试），got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "网络不通") {
		t.Errorf("错误消息应说明网络不通，got: %v", err)
	}
}

func TestPrecheckVDBConnectivity_NoProbeTool_FailOpen(t *testing.T) {
	setupTestDB(t)
	orig := taskRunScriptFn
	defer func() { taskRunScriptFn = orig }()
	taskRunScriptFn = func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return `[precheck-vdb] WARN: 无 nc 且无 python3
{"reachable":false,"host":"10.0.0.49","port":80,"probe":"none","error":"no_probe_tool"}`,
			hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	}
	if err := precheckVDBConnectivity(context.Background(), "inst-x", "http://10.0.0.49:80", "user", "key"); err != nil {
		t.Fatalf("无探测工具时应保守通过返回 nil，got: %v", err)
	}
}

func TestPrecheckVDBConnectivity_ScriptCallFailed(t *testing.T) {
	setupTestDB(t)
	orig := taskRunScriptFn
	defer func() { taskRunScriptFn = orig }()
	// 模拟 TAT 完全没拿到结果（agent 离线、网络问题等）
	taskRunScriptFn = func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	}
	err := precheckVDBConnectivity(context.Background(), "inst-x", "http://10.0.0.49:80", "user", "key")
	if err == nil {
		t.Fatal("脚本调用失败应返回错误")
	}
	// 必须不是 NonRetryable（TAT 框架问题应可重试）
	var nre *NonRetryableError
	if errors.As(err, &nre) {
		t.Fatalf("TAT 框架问题不应判 NonRetryable: %v", err)
	}
}

func TestPrecheckVDBConnectivity_UnparseableOutput(t *testing.T) {
	setupTestDB(t)
	orig := taskRunScriptFn
	defer func() { taskRunScriptFn = orig }()
	taskRunScriptFn = func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "garbage output without json", nil
	}
	err := precheckVDBConnectivity(context.Background(), "inst-x", "http://10.0.0.49:80", "user", "key")
	if err == nil {
		t.Fatal("无法解析的输出应返回错误（可重试）")
	}
	var nre *NonRetryableError
	if errors.As(err, &nre) {
		t.Fatalf("解析失败不应判 NonRetryable: %v", err)
	}
}

// ========== shouldSkipVDBExportOnDisable 测试 ==========
//
// 与 precheckVDBConnectivity 的语义差异：本函数永不抛错、永不阻断 OFF 流程。
// 所有路径都返回 true/false。
func TestShouldSkipVDBExport_EmptyEndpoint_NoSkip(t *testing.T) {
	setupTestDB(t)
	if shouldSkipVDBExportOnDisable(context.Background(), "inst-x", "", "user", "key") {
		t.Fatal("空 endpoint 应返回 false（让脚本自己决定，不主动 skip）")
	}
}

func TestShouldSkipVDBExport_Reachable_NoSkip(t *testing.T) {
	setupTestDB(t)
	orig := taskRunScriptFn
	defer func() { taskRunScriptFn = orig }()
	taskRunScriptFn = func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return `{"reachable":true,"host":"10.0.0.49","port":80,"probe":"nc"}`, nil
	}
	if shouldSkipVDBExportOnDisable(context.Background(), "inst-x", "http://10.0.0.49:80", "user", "key") {
		t.Fatal("网络通时应返回 false（不跳过 export）")
	}
}

func TestShouldSkipVDBExport_Unreachable_Skip(t *testing.T) {
	setupTestDB(t)
	orig := taskRunScriptFn
	defer func() { taskRunScriptFn = orig }()
	// 真实 TAT 行为：output="" + Detail 装 stdout
	taskRunScriptFn = func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail(`{"reachable":false,"host":"10.0.0.49","port":80,"probe":"nc"}`)
	}
	if !shouldSkipVDBExportOnDisable(context.Background(), "inst-x", "http://10.0.0.49:80", "user", "key") {
		t.Fatal("网络明确不通时应返回 true（跳过 export）")
	}
}

func TestShouldSkipVDBExport_NoProbeTool_NoSkip(t *testing.T) {
	setupTestDB(t)
	orig := taskRunScriptFn
	defer func() { taskRunScriptFn = orig }()
	taskRunScriptFn = func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail(`{"reachable":false,"probe":"none","error":"no_probe_tool"}`)
	}
	if shouldSkipVDBExportOnDisable(context.Background(), "inst-x", "http://10.0.0.49:80", "user", "key") {
		t.Fatal("探测工具不可用时应返回 false（保守按通处理）")
	}
}

func TestShouldSkipVDBExport_TATError_NoSkip(t *testing.T) {
	setupTestDB(t)
	orig := taskRunScriptFn
	defer func() { taskRunScriptFn = orig }()
	taskRunScriptFn = func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	}
	if shouldSkipVDBExportOnDisable(context.Background(), "inst-x", "http://10.0.0.49:80", "user", "key") {
		t.Fatal("TAT 调用失败时应返回 false（保守按通处理，不跳过 export）")
	}
}
