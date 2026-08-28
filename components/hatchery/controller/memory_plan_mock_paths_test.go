package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

func withResolveMemoryPluginRootMock(t *testing.T, fn func(ctx context.Context, instanceID string) (string, error)) func() {
	t.Helper()
	orig := resolveMemoryPluginRootFn
	resolveMemoryPluginRootFn = fn
	return func() { resolveMemoryPluginRootFn = orig }
}

func withEnsureMemoryPluginMock(t *testing.T, fn func(ctx context.Context, instanceID string) error) func() {
	t.Helper()
	orig := ensureMemoryPluginFn
	ensureMemoryPluginFn = fn
	return func() { ensureMemoryPluginFn = orig }
}

func withRunScriptForMemoryPlanMock(t *testing.T, fn func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error)) func() {
	t.Helper()
	orig := runScriptForMemoryPlanFn
	runScriptForMemoryPlanFn = fn
	return func() { runScriptForMemoryPlanFn = orig }
}

func withRunInlineScriptMock(t *testing.T, fn func(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error)) func() {
	t.Helper()
	orig := runInlineScriptFn
	runInlineScriptFn = func(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error) {
		return fn(ctx, instanceId, scriptContent, timeout)
	}
	return func() { runInlineScriptFn = orig }
}

func TestEnsureMemoryPlugin_DelegatesToMock(t *testing.T) {
	setupMemoryProDB(t)

	wantErr := errors.New("mock ensure error")
	var gotInstanceID string
	defer withEnsureMemoryPluginMock(t, func(ctx context.Context, instanceID string) error {
		gotInstanceID = instanceID
		return wantErr
	})()

	err := ensureMemoryPlugin(context.Background(), "ins-mock-delegate")
	if !errors.Is(err, wantErr) {
		t.Fatalf("ensureMemoryPlugin error = %v, want %v", err, wantErr)
	}
	if gotInstanceID != "ins-mock-delegate" {
		t.Fatalf("instanceID = %q, want %q", gotInstanceID, "ins-mock-delegate")
	}
}

func TestEnsureMemoryPluginImpl_UsesRunScriptWrapper(t *testing.T) {
	setupMemoryProDB(t)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ensure-impl", AgentType: model.AgentTypeOpenClaw})

	wantErr := errors.New("ensure script failed")
	defer withRunScriptForMemoryPlanMock(t, func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		if instanceID != "ins-ensure-impl" {
			t.Fatalf("instanceID = %q, want ins-ensure-impl", instanceID)
		}
		if scriptName != "ensure_memory_plugin.sh" {
			t.Fatalf("scriptName = %q, want ensure_memory_plugin.sh", scriptName)
		}
		if params["plugin"] != model.DefaultMemoryTDAIPluginName {
			t.Fatalf("plugin param = %q, want %q", params["plugin"], model.DefaultMemoryTDAIPluginName)
		}
		return "", hcommon.I18nRichError(wantErr, i18n.MsgOperationFailed)
	})()

	err := ensureMemoryPluginImpl(context.Background(), "ins-ensure-impl")
	if !errors.Is(err, wantErr) {
		t.Fatalf("ensureMemoryPluginImpl error = %v, want %v", err, wantErr)
	}
}

func TestHandleMemoryLibraryDetail_FreeEnsurePluginFailed(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u-free-ensure-fail", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-free-ensure-fail", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-free-ensure-fail", CurrentPlan: model.MemoryPlanFree})

	defer withResolveMemoryPluginRootMock(t, func(ctx context.Context, instanceID string) (string, error) {
		return "$HOME/.openclaw/npm/node_modules/@tencentdb-agent-memory/memory-tencentdb", nil
	})()
	defer withEnsureMemoryPluginMock(t, func(ctx context.Context, instanceID string) error { return errors.New("plugin not ready") })()
	defer withRunInlineScriptMock(t, func(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error) {
		t.Fatal("runInlineScriptFn should not be called when ensureMemoryPlugin fails")
		return "", nil
	})()

	req, w := makeLoggedInJSONRequest(t, http.MethodGet,
		"/openclaw/memory/library/detail?instance_id=ins-free-ensure-fail&type=memory", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "记忆插件未就绪") {
		t.Fatalf("body should mention plugin not ready, got: %s", w.Body.String())
	}
}

func TestHandleMemoryLibraryDetail_FreeReadScriptFailed(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u-free-read-fail", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-free-read-fail", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-free-read-fail", CurrentPlan: model.MemoryPlanFree})

	defer withResolveMemoryPluginRootMock(t, func(ctx context.Context, instanceID string) (string, error) {
		return "$HOME/.openclaw/npm/node_modules/@tencentdb-agent-memory/memory-tencentdb", nil
	})()
	defer withEnsureMemoryPluginMock(t, func(ctx context.Context, instanceID string) error { return nil })()
	defer withRunInlineScriptMock(t, func(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error) {
		return "", errors.New("tat exec failed")
	})()

	req, w := makeLoggedInJSONRequest(t, http.MethodGet,
		"/openclaw/memory/library/detail?instance_id=ins-free-read-fail&type=memory&page=1&page_size=10", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "读取本地记忆数据失败") {
		t.Fatalf("body should mention local read failure, got: %s", w.Body.String())
	}
}

func TestHandleMemoryLibraryDetail_FreeLargeOutput(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u-free-large", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-free-large", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-free-large", CurrentPlan: model.MemoryPlanFree})

	defer withResolveMemoryPluginRootMock(t, func(ctx context.Context, instanceID string) (string, error) {
		return "$HOME/.openclaw/npm/node_modules/@tencentdb-agent-memory/memory-tencentdb", nil
	})()
	defer withEnsureMemoryPluginMock(t, func(ctx context.Context, instanceID string) error { return nil })()
	defer withRunInlineScriptMock(t, func(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error) {
		return strings.Repeat("x", 24*1024), nil
	})()

	req, w := makeLoggedInJSONRequest(t, http.MethodGet,
		"/openclaw/memory/library/detail?instance_id=ins-free-large&type=conversation&page=1&page_size=10", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "24KB") {
		t.Fatalf("body should mention TAT 24KB limit, got: %s", w.Body.String())
	}
}

func TestHandleMemoryLibraryDetail_FreeSceneSuccess(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u-scene-ok", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-scene-ok", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-scene-ok", CurrentPlan: model.MemoryPlanFree})

	defer withResolveMemoryPluginRootMock(t, func(ctx context.Context, instanceID string) (string, error) {
		return "$HOME/.openclaw/npm/node_modules/@tencentdb-agent-memory/memory-tencentdb", nil
	})()
	defer withEnsureMemoryPluginMock(t, func(ctx context.Context, instanceID string) error { return nil })()
	call := 0
	defer withRunInlineScriptMock(t, func(ctx context.Context, instanceID, script string, timeout uint64) (string, error) {
		call++
		switch call {
		case 1:
			return `{"level":"L2","total":2,"data":[{"fileName":"a.md","summary":"sa","heat":1,"created":"ca","updated":"ua"},{"fileName":"b.md","summary":"sb","heat":2,"created":"cb","updated":"ub"}]}`, nil
		case 2:
			return `{"body":"body-a"}`, nil
		case 3:
			return `{"body":"body-b"}`, nil
		default:
			return "", fmt.Errorf("unexpected call %d", call)
		}
	})()

	req, w := makeLoggedInJSONRequest(t, http.MethodGet,
		"/openclaw/memory/library/detail?instance_id=ins-scene-ok&type=scene", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"total_count":2`) || !strings.Contains(body, `"body":"body-a"`) || !strings.Contains(body, `"body":"body-b"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestHandleMemoryLibraryDetail_FreeSceneFallbacks(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u-scene-fallback", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-scene-fallback", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-scene-fallback", CurrentPlan: model.MemoryPlanFree})

	defer withResolveMemoryPluginRootMock(t, func(ctx context.Context, instanceID string) (string, error) {
		return "$HOME/.openclaw/npm/node_modules/@tencentdb-agent-memory/memory-tencentdb", nil
	})()
	defer withEnsureMemoryPluginMock(t, func(ctx context.Context, instanceID string) error { return nil })()
	call := 0
	defer withRunInlineScriptMock(t, func(ctx context.Context, instanceID, script string, timeout uint64) (string, error) {
		call++
		switch call {
		case 1:
			return `{"level":"L2","total":3,"data":[{"fileName":"err.md","summary":"se","heat":1,"created":"ce","updated":"ue"},{"fileName":"big.md","summary":"sb","heat":2,"created":"cb","updated":"ub"},{"fileName":"bad.json","summary":"sc","heat":3,"created":"cc","updated":"uc"}]}`, nil
		case 2:
			return "", errors.New("read failed")
		case 3:
			return strings.Repeat("x", 24*1024), nil
		case 4:
			return `not-json`, nil
		default:
			return "", fmt.Errorf("unexpected call %d", call)
		}
	})()

	req, w := makeLoggedInJSONRequest(t, http.MethodGet,
		"/openclaw/memory/library/detail?instance_id=ins-scene-fallback&type=scene", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"读取失败", "内容过大，暂不支持在线查看", "解析失败"} {
		if !strings.Contains(body, want) {
			t.Fatalf("response should contain %q, got: %s", want, body)
		}
	}
}
