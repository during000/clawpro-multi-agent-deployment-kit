package controller

import (
	"context"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"strings"
	"testing"
)

// withRunScriptMock 替换 runScriptForPathFn，测试结束后恢复。
func withRunScriptMock(t *testing.T,
	runner func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error),
) {
	t.Helper()
	origRunner := runScriptForPathFn
	runScriptForPathFn = runner
	t.Cleanup(func() {
		runScriptForPathFn = origRunner
	})
}

// TestResolveMemoryPluginRoot_Priority1_Projects 验证 5.28+ npm/projects 路径优先返回。
func TestResolveMemoryPluginRoot_Priority1_Projects(t *testing.T) {
	expected := "$HOME/.openclaw/npm/projects/tencentdb-agent-memory-memory-tencentdb-abc123/node_modules/@tencentdb-agent-memory/memory-tencentdb"
	withRunScriptMock(t,
		func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return expected + "\n", nil
		},
	)

	root, err := resolveMemoryPluginRootImpl(context.Background(), "ins-test001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != expected {
		t.Errorf("expected %q, got %q", expected, root)
	}
}

// TestResolveMemoryPluginRoot_Priority2_Npm 验证 5.2~5.7 npm/node_modules 路径。
func TestResolveMemoryPluginRoot_Priority2_Npm(t *testing.T) {
	expected := "$HOME/.openclaw/npm/node_modules/@tencentdb-agent-memory/memory-tencentdb"
	withRunScriptMock(t,
		func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return expected, nil
		},
	)

	root, err := resolveMemoryPluginRootImpl(context.Background(), "ins-test002")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != expected {
		t.Errorf("expected %q, got %q", expected, root)
	}
}

// TestResolveMemoryPluginRoot_Priority3_Extensions 验证 ≤5.1 extensions 路径。
func TestResolveMemoryPluginRoot_Priority3_Extensions(t *testing.T) {
	expected := "$HOME/.openclaw/extensions/memory-tencentdb"
	withRunScriptMock(t,
		func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return expected, nil
		},
	)

	root, err := resolveMemoryPluginRootImpl(context.Background(), "ins-test003")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != expected {
		t.Errorf("expected %q, got %q", expected, root)
	}
}

// TestResolveMemoryPluginRoot_EmptyOutput_AllPathsNotExist 验证三个路径都不存在时返回错误。
func TestResolveMemoryPluginRoot_EmptyOutput_AllPathsNotExist(t *testing.T) {
	withRunScriptMock(t,
		func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return "", nil
		},
	)

	_, err := resolveMemoryPluginRootImpl(context.Background(), "ins-test004")
	if err == nil {
		t.Fatal("expected error when all paths not exist, got nil")
	}
	if !strings.Contains(err.Error(), "三个路径均不存在") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestResolveMemoryPluginRoot_TATFailure_Fallback 验证 TAT 探测失败时 fallback 到旧路径。
func TestResolveMemoryPluginRoot_TATFailure_Fallback(t *testing.T) {
	withRunScriptMock(t,
		func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return "", hcommon.I18nError(i18n.MsgOperationFailed)
		},
	)

	root, err := resolveMemoryPluginRootImpl(context.Background(), "ins-test005")
	if err != nil {
		t.Fatalf("expected fallback (no error), got error: %v", err)
	}
	expected := "$HOME/.openclaw/extensions/memory-tencentdb"
	if root != expected {
		t.Errorf("expected fallback %q, got %q", expected, root)
	}
}

// TestResolveMemoryPluginRoot_TrimSpace 验证输出中的空白字符被正确去除。
func TestResolveMemoryPluginRoot_TrimSpace(t *testing.T) {
	expected := "$HOME/.openclaw/npm/node_modules/@tencentdb-agent-memory/memory-tencentdb"
	withRunScriptMock(t,
		func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return "  " + expected + "  \n", nil
		},
	)

	root, err := resolveMemoryPluginRootImpl(context.Background(), "ins-test006")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != expected {
		t.Errorf("expected %q, got %q", expected, root)
	}
}

// TestResolveMemoryPluginRoot_ParamsPassedCorrectly 验证 params 传递了正确的参数。
func TestResolveMemoryPluginRoot_ParamsPassedCorrectly(t *testing.T) {
	var capturedParams map[string]string
	withRunScriptMock(t,
		func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			capturedParams = params
			return "$HOME/.openclaw/extensions/memory-tencentdb", nil
		},
	)

	_, _ = resolveMemoryPluginRootImpl(context.Background(), "ins-test007")

	if capturedParams == nil {
		t.Fatal("params should not be nil")
	}
	if capturedParams["memory_plugin_id"] != "memory-tencentdb" {
		t.Errorf("expected memory_plugin_id=%q, got %q", "memory-tencentdb", capturedParams["memory_plugin_id"])
	}
	if capturedParams["memory_npm_pkg"] != "@tencentdb-agent-memory/memory-tencentdb" {
		t.Errorf("expected memory_npm_pkg=%q, got %q", "@tencentdb-agent-memory/memory-tencentdb", capturedParams["memory_npm_pkg"])
	}
}

// TestResolveMemoryPluginRoot_ScriptName 验证调用的是正确的脚本文件名。
func TestResolveMemoryPluginRoot_ScriptName(t *testing.T) {
	var capturedScript string
	withRunScriptMock(t,
		func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			capturedScript = scriptName
			return "$HOME/.openclaw/extensions/memory-tencentdb", nil
		},
	)

	_, _ = resolveMemoryPluginRootImpl(context.Background(), "ins-test012")

	if capturedScript != "resolve_memory_plugin_root.sh" {
		t.Errorf("expected script %q, got %q", "resolve_memory_plugin_root.sh", capturedScript)
	}
}

// TestResolveMemoryPluginRoot_Timeout 验证传入正确的超时参数。
func TestResolveMemoryPluginRoot_Timeout(t *testing.T) {
	var capturedTimeout uint64
	withRunScriptMock(t,
		func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			capturedTimeout = timeout
			return "$HOME/.openclaw/extensions/memory-tencentdb", nil
		},
	)

	_, _ = resolveMemoryPluginRootImpl(context.Background(), "ins-test009")

	if capturedTimeout != 60 {
		t.Errorf("expected timeout 60, got %d", capturedTimeout)
	}
}

// TestFallbackMemoryPluginRoot 验证 fallback 路径格式正确。
func TestFallbackMemoryPluginRoot(t *testing.T) {
	result := fallbackMemoryPluginRoot(context.Background(), "ins-test010")
	expected := "$HOME/.openclaw/extensions/memory-tencentdb"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestResolveMemoryPluginRoot_PublicAPI 验证公开接口调用到 resolveMemoryPluginRootFn。
func TestResolveMemoryPluginRoot_PublicAPI(t *testing.T) {
	orig := resolveMemoryPluginRootFn
	called := false
	resolveMemoryPluginRootFn = func(ctx context.Context, instanceID string) (string, error) {
		called = true
		return "$HOME/.openclaw/extensions/memory-tencentdb", nil
	}
	t.Cleanup(func() { resolveMemoryPluginRootFn = orig })

	root, err := ResolveMemoryPluginRoot(context.Background(), "ins-test011")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("resolveMemoryPluginRootFn was not called")
	}
	if root != "$HOME/.openclaw/extensions/memory-tencentdb" {
		t.Errorf("unexpected root: %q", root)
	}
}
