package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// setupPluginUpgradeDB 初始化插件升级测试所需的数据库。
func setupPluginUpgradeDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试 DB 失败: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&model.User{},
		&model.Instance{},
		&model.MemoryTDAIPlugin{},
	); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	restore := model.UseDBForTest(db)
	if Store == nil {
		Store = sessions.NewCookieStore([]byte("test-secret-key-32bytes-padding!"))
	}
	origAdminToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() {
		restore()
		AdminToken = origAdminToken
	})
}

func makePluginUpgradeRequest(method, path, body string) (*http.Request, *httptest.ResponseRecorder) {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req, httptest.NewRecorder()
}

// ========== versionLessThan 测试 ==========

func TestVersionLessThan(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"0.3.0", "0.3.3", true},
		{"0.3.3", "0.3.3", false},
		{"0.3.4", "0.3.3", false},
		{"0.2.9", "0.3.0", true},
		{"1.0.0", "0.9.9", false},
		{"", "0.3.3", true},
		{"0.3.3", "", false},
		{"v0.3.0", "0.3.3", true},
		{"0.3.0", "v0.3.3", true},
		{"0.3.3-beta", "0.3.3", false},
		{"0.3.2-rc1", "0.3.3", true},
	}

	for _, tt := range tests {
		got := versionLessThan(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("versionLessThan(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

// ========== Candidates 接口测试 ==========

func TestCandidates_Unauthorized(t *testing.T) {
	setupPluginUpgradeDB(t)

	req := httptest.NewRequest("GET", "/admin/memory/plugin-upgrade/candidates", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	HandleAdminMemoryPluginUpgradeCandidates(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCandidates_NoProInstances(t *testing.T) {
	setupPluginUpgradeDB(t)

	req, w := makePluginUpgradeRequest("GET", "/admin/memory/plugin-upgrade/candidates", "")
	HandleAdminMemoryPluginUpgradeCandidates(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["total"].(float64) != 0 {
		t.Errorf("expected total=0, got %v", resp["total"])
	}
	if resp["min_version"] != model.DefaultMemoryTDAIMinVersion {
		t.Errorf("expected min_version=%s, got %v", model.DefaultMemoryTDAIMinVersion, resp["min_version"])
	}
}

func TestCandidates_SkipsSwitchingInstances(t *testing.T) {
	setupPluginUpgradeDB(t)

	// 创建一个正在切换中的 Pro 实例
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "ins-switching",
		CurrentPlan:  model.MemoryPlanPro,
		SwitchStatus: "SWITCHING_TO_OFF",
	})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-switching", AgentType: "openclaw"})

	req, w := makePluginUpgradeRequest("GET", "/admin/memory/plugin-upgrade/candidates", "")
	HandleAdminMemoryPluginUpgradeCandidates(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	// 正在切换中的实例不应出现在候选列表
	if resp["total"].(float64) != 0 {
		t.Errorf("expected total=0 (switching instance excluded), got %v", resp["total"])
	}
}

// ========== Execute 接口测试 ==========

func TestExecute_Unauthorized(t *testing.T) {
	setupPluginUpgradeDB(t)

	req := httptest.NewRequest("POST", "/admin/memory/plugin-upgrade/execute", strings.NewReader(`{"instance_ids":["ins-test"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	HandleAdminMemoryPluginUpgradeExecute(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestExecute_MethodNotAllowed(t *testing.T) {
	setupPluginUpgradeDB(t)

	req, w := makePluginUpgradeRequest("GET", "/admin/memory/plugin-upgrade/execute", "")
	HandleAdminMemoryPluginUpgradeExecute(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestExecute_EmptyInstanceIDs(t *testing.T) {
	setupPluginUpgradeDB(t)

	req, w := makePluginUpgradeRequest("POST", "/admin/memory/plugin-upgrade/execute", `{"instance_ids":[]}`)
	HandleAdminMemoryPluginUpgradeExecute(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestExecute_InvalidJSON(t *testing.T) {
	setupPluginUpgradeDB(t)

	req, w := makePluginUpgradeRequest("POST", "/admin/memory/plugin-upgrade/execute", `{bad json}`)
	HandleAdminMemoryPluginUpgradeExecute(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestExecute_InstanceNotFound(t *testing.T) {
	setupPluginUpgradeDB(t)

	req, w := makePluginUpgradeRequest("POST", "/admin/memory/plugin-upgrade/execute", `{"instance_ids":["ins-nonexistent"]}`)
	HandleAdminMemoryPluginUpgradeExecute(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	results := resp["results"].([]any)
	first := results[0].(map[string]any)
	if first["status"] != "failed" {
		t.Errorf("expected status=failed, got %v", first["status"])
	}
	if first["message"] != "实例不存在" {
		t.Errorf("expected message=实例不存在, got %v", first["message"])
	}
}

func TestExecute_NotProInstance(t *testing.T) {
	setupPluginUpgradeDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "ins-free",
		CurrentPlan: "FREE",
	})

	req, w := makePluginUpgradeRequest("POST", "/admin/memory/plugin-upgrade/execute", `{"instance_ids":["ins-free"]}`)
	HandleAdminMemoryPluginUpgradeExecute(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	results := resp["results"].([]any)
	first := results[0].(map[string]any)
	if first["status"] != "failed" {
		t.Errorf("expected status=failed, got %v", first["status"])
	}
	if first["message"] != "实例当前非 Pro 版" {
		t.Errorf("expected message about non-Pro, got %v", first["message"])
	}
}

func TestExecute_SwitchingInstance(t *testing.T) {
	setupPluginUpgradeDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "ins-switching",
		CurrentPlan:  model.MemoryPlanPro,
		SwitchStatus: "SWITCHING_TO_OFF",
	})

	req, w := makePluginUpgradeRequest("POST", "/admin/memory/plugin-upgrade/execute", `{"instance_ids":["ins-switching"]}`)
	HandleAdminMemoryPluginUpgradeExecute(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	results := resp["results"].([]any)
	first := results[0].(map[string]any)
	if first["status"] != "failed" {
		t.Errorf("expected status=failed, got %v", first["status"])
	}
}

func TestExecute_ValidProInstance(t *testing.T) {
	setupPluginUpgradeDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "ins-pro-ok",
		CurrentPlan:  model.MemoryPlanPro,
		SwitchStatus: "",
	})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-pro-ok", AgentType: "openclaw"})

	// 备份并 mock EnsureMemoryPlugin 以避免真实 TAT 调用
	origEnsure := ensureMemoryPluginFn
	ensureMemoryPluginFn = func(ctx context.Context, instanceID string) error {
		return nil
	}
	origRunInline := runInlineScriptFn
	runInlineScriptFn = func(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error) {
		return "0.3.3", nil
	}
	t.Cleanup(func() {
		ensureMemoryPluginFn = origEnsure
		runInlineScriptFn = origRunInline
	})

	req, w := makePluginUpgradeRequest("POST", "/admin/memory/plugin-upgrade/execute",
		`{"instance_ids":["ins-pro-ok"]}`)
	HandleAdminMemoryPluginUpgradeExecute(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["submitted"].(float64) != 1 {
		t.Errorf("expected submitted=1, got %v", resp["submitted"])
	}
	results := resp["results"].([]any)
	first := results[0].(map[string]any)
	if first["status"] != "submitted" {
		t.Errorf("expected status=submitted, got %v", first["status"])
	}
}

func TestExecute_MixedInstances(t *testing.T) {
	setupPluginUpgradeDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "ins-pro-1",
		CurrentPlan:  model.MemoryPlanPro,
		SwitchStatus: "",
	})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-pro-1", AgentType: "openclaw"})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "ins-free-1",
		CurrentPlan: "FREE",
	})

	origEnsure := ensureMemoryPluginFn
	ensureMemoryPluginFn = func(ctx context.Context, instanceID string) error { return nil }
	origRunInline := runInlineScriptFn
	runInlineScriptFn = func(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error) {
		return "0.3.3", nil
	}
	t.Cleanup(func() {
		ensureMemoryPluginFn = origEnsure
		runInlineScriptFn = origRunInline
	})

	req, w := makePluginUpgradeRequest("POST", "/admin/memory/plugin-upgrade/execute",
		`{"instance_ids":["ins-pro-1","ins-free-1","ins-nonexistent"]}`)
	HandleAdminMemoryPluginUpgradeExecute(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["submitted"].(float64) != 1 {
		t.Errorf("expected submitted=1, got %v", resp["submitted"])
	}
	results := resp["results"].([]any)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

// ========== Candidates 接口 — 带 mock TAT 的完整流程测试 ==========

func TestCandidates_WithMockTAT_FiltersCorrectly(t *testing.T) {
	setupPluginUpgradeDB(t)

	// 创建测试数据：
	// ins-low: 版本低 + offload 未开 → 应返回
	// ins-ok-no-offload: 版本够 + offload 未开 → 应返回
	// ins-ok-offload: 版本够 + offload 已开 → 不应返回
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-low", CurrentPlan: model.MemoryPlanPro, SwitchStatus: ""})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-ok-no-offload", CurrentPlan: model.MemoryPlanPro, SwitchStatus: ""})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-ok-offload", CurrentPlan: model.MemoryPlanPro, SwitchStatus: ""})

	// 创建对应的 Instance 行（用于 JOIN 过滤 agent_type + 获取名称）
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-low", Name: "test-low", AgentType: "openclaw"})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ok-no-offload", Name: "test-ok-no-offload", AgentType: "openclaw"})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ok-offload", Name: "test-ok-offload", AgentType: "openclaw"})

	// Mock TAT 返回
	origRunInline := runInlineScriptFn
	runInlineScriptFn = func(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error) {
		switch instanceId {
		case "ins-low":
			return `{"version":"0.2.0","offload":false}`, nil
		case "ins-ok-no-offload":
			return `{"version":"0.3.3","offload":false}`, nil
		case "ins-ok-offload":
			return `{"version":"0.3.3","offload":true}`, nil
		}
		return "", nil
	}
	t.Cleanup(func() { runInlineScriptFn = origRunInline })

	req, w := makePluginUpgradeRequest("GET", "/admin/memory/plugin-upgrade/candidates", "")
	HandleAdminMemoryPluginUpgradeCandidates(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	total := int(resp["total"].(float64))
	if total != 2 {
		t.Errorf("expected total=2, got %d", total)
	}

	instances := resp["instances"].([]any)
	ids := make(map[string]bool)
	for _, inst := range instances {
		m := inst.(map[string]any)
		ids[m["instance_id"].(string)] = true
	}
	if !ids["ins-low"] {
		t.Error("expected ins-low in candidates")
	}
	if !ids["ins-ok-no-offload"] {
		t.Error("expected ins-ok-no-offload in candidates")
	}
	if ids["ins-ok-offload"] {
		t.Error("ins-ok-offload should NOT be in candidates")
	}

	// 验证 DB 回写
	var plugin model.MemoryTDAIPlugin
	model.DB(context.Background()).Where("instance_id = ?", "ins-low").First(&plugin)
	if plugin.MemoryPluginVersion != "0.2.0" {
		t.Errorf("expected version 0.2.0 written to DB, got %s", plugin.MemoryPluginVersion)
	}
}

func TestCandidates_TATFailure_SkipsInstance(t *testing.T) {
	setupPluginUpgradeDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-offline", CurrentPlan: model.MemoryPlanPro, SwitchStatus: ""})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-offline", AgentType: "openclaw"})

	origRunInline := runInlineScriptFn
	runInlineScriptFn = func(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error) {
		return "", fmt.Errorf("TAT agent offline")
	}
	t.Cleanup(func() { runInlineScriptFn = origRunInline })

	req, w := makePluginUpgradeRequest("GET", "/admin/memory/plugin-upgrade/candidates", "")
	HandleAdminMemoryPluginUpgradeCandidates(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["total"].(float64) != 0 {
		t.Errorf("expected total=0 (offline instance skipped), got %v", resp["total"])
	}
}

// ========== doPluginUpgrade 直接测试 ==========

func TestDoPluginUpgrade_FullSuccess(t *testing.T) {
	setupPluginUpgradeDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-upgrade", CurrentPlan: model.MemoryPlanPro})

	origResolve := resolveMemoryPluginRootFn
	resolveMemoryPluginRootFn = func(ctx context.Context, instanceID string) (string, error) {
		return "$HOME/.openclaw/npm/node_modules/@tencentdb-agent-memory/memory-tencentdb", nil
	}
	origEnsure := ensureMemoryPluginFn
	ensureMemoryPluginFn = func(ctx context.Context, instanceID string) error { return nil }
	origRunInline := runInlineScriptFn
	runInlineScriptFn = func(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error) {
		if strings.Contains(scriptContent, "setup-offload.sh") {
			return "offload enabled", nil
		}
		if strings.Contains(scriptContent, "openclaw-gateway") {
			return "openclaw-gateway restarted", nil
		}
		return "0.3.3", nil
	}
	t.Cleanup(func() {
		resolveMemoryPluginRootFn = origResolve
		ensureMemoryPluginFn = origEnsure
		runInlineScriptFn = origRunInline
	})

	doPluginUpgrade(context.Background(), slog.Default(), "ins-upgrade", "https://memory.tdai.tencentyun.com", "3205597606")

	// 等一下让 goroutine 内的 DB 写入完成
	time.Sleep(100 * time.Millisecond)

	var plugin model.MemoryTDAIPlugin
	model.DB(context.Background()).Where("instance_id = ?", "ins-upgrade").First(&plugin)
	if plugin.MemoryPluginVersion != "0.3.3" {
		t.Errorf("expected version=0.3.3, got %s", plugin.MemoryPluginVersion)
	}
	if plugin.OffloadEnabled == nil || !*plugin.OffloadEnabled {
		t.Errorf("expected offload_enabled=true")
	}
}

func TestDoPluginUpgrade_EnsureFails(t *testing.T) {
	setupPluginUpgradeDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-fail", CurrentPlan: model.MemoryPlanPro})

	origEnsure := ensureMemoryPluginFn
	ensureMemoryPluginFn = func(ctx context.Context, instanceID string) error {
		return fmt.Errorf("TAT timeout")
	}
	origRunInline := runInlineScriptFn
	runInlineScriptFn = func(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error) {
		t.Error("should not call RunInlineScript when ensure fails")
		return "", nil
	}
	t.Cleanup(func() {
		ensureMemoryPluginFn = origEnsure
		runInlineScriptFn = origRunInline
	})

	doPluginUpgrade(context.Background(), slog.Default(), "ins-fail", "https://memory.tdai.tencentyun.com", "3205597606")

	// DB 不应被更新
	var plugin model.MemoryTDAIPlugin
	model.DB(context.Background()).Where("instance_id = ?", "ins-fail").First(&plugin)
	if plugin.MemoryPluginVersion != "" {
		t.Errorf("expected empty version (not updated), got %s", plugin.MemoryPluginVersion)
	}
}

func TestDoPluginUpgrade_OffloadFails_StillRestarts(t *testing.T) {
	setupPluginUpgradeDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-offload-fail", CurrentPlan: model.MemoryPlanPro})

	var mu sync.Mutex
	var calledScripts []string

	origResolve := resolveMemoryPluginRootFn
	resolveMemoryPluginRootFn = func(ctx context.Context, instanceID string) (string, error) {
		return "$HOME/.openclaw/npm/node_modules/@tencentdb-agent-memory/memory-tencentdb", nil
	}
	origEnsure := ensureMemoryPluginFn
	ensureMemoryPluginFn = func(ctx context.Context, instanceID string) error { return nil }
	origRunInline := runInlineScriptFn
	runInlineScriptFn = func(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error) {
		mu.Lock()
		calledScripts = append(calledScripts, scriptContent)
		mu.Unlock()
		if strings.Contains(scriptContent, "setup-offload.sh") {
			return "", fmt.Errorf("offload script failed")
		}
		if strings.Contains(scriptContent, "openclaw-gateway") {
			return "openclaw-gateway restarted", nil
		}
		return "0.3.3", nil // version query
	}
	t.Cleanup(func() {
		resolveMemoryPluginRootFn = origResolve
		ensureMemoryPluginFn = origEnsure
		runInlineScriptFn = origRunInline
	})

	doPluginUpgrade(context.Background(), slog.Default(), "ins-offload-fail", "https://memory.tdai.tencentyun.com", "3205597606")

	// 验证重启被调用了（即使 offload 失败）
	mu.Lock()
	defer mu.Unlock()
	restartCalled := false
	for _, s := range calledScripts {
		if strings.Contains(s, "openclaw-gateway") {
			restartCalled = true
		}
	}
	if !restartCalled {
		t.Error("expected openclaw restart to be called even when offload fails")
	}

	// offload_enabled 不应被更新
	var plugin model.MemoryTDAIPlugin
	model.DB(context.Background()).Where("instance_id = ?", "ins-offload-fail").First(&plugin)
	if plugin.OffloadEnabled != nil && *plugin.OffloadEnabled {
		t.Error("offload_enabled should not be true when offload failed")
	}
}

func TestDoPluginUpgrade_EmptyOffloadURL(t *testing.T) {
	setupPluginUpgradeDB(t)

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-no-url", CurrentPlan: model.MemoryPlanPro})

	origResolve := resolveMemoryPluginRootFn
	resolveMemoryPluginRootFn = func(ctx context.Context, instanceID string) (string, error) {
		return "$HOME/.openclaw/npm/node_modules/@tencentdb-agent-memory/memory-tencentdb", nil
	}
	origEnsure := ensureMemoryPluginFn
	ensureMemoryPluginFn = func(ctx context.Context, instanceID string) error { return nil }
	origRunInline := runInlineScriptFn
	var scripts []string
	runInlineScriptFn = func(ctx context.Context, instanceId string, scriptContent string, timeout uint64) (string, error) {
		scripts = append(scripts, scriptContent)
		if strings.Contains(scriptContent, "openclaw-gateway") {
			return "restarted", nil
		}
		return "0.3.3", nil
	}
	t.Cleanup(func() {
		resolveMemoryPluginRootFn = origResolve
		ensureMemoryPluginFn = origEnsure
		runInlineScriptFn = origRunInline
	})

	doPluginUpgrade(context.Background(), slog.Default(), "ins-no-url", "", "3205597606")

	// offload 不应被调用
	for _, s := range scripts {
		if strings.Contains(s, "setup-offload.sh") {
			t.Error("should not call offload script when URL is empty")
		}
	}
	// 重启应该被调用
	restartCalled := false
	for _, s := range scripts {
		if strings.Contains(s, "openclaw-gateway") {
			restartCalled = true
		}
	}
	if !restartCalled {
		t.Error("expected restart to be called")
	}
}
