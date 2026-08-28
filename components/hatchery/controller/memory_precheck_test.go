package controller

import (
	"context"
	"fmt"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupPrecheckDB 初始化预检测试所需的最小 DB。
func setupPrecheckDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试 DB 失败: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&model.Instance{},
		&model.MemoryTDAIPlugin{},
	); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	restore := model.UseDBForTest(db)
	return func() { restore() }
}

// mockPrecheckRunScript 替换 precheckRunScriptFn 并返回还原函数。
func mockPrecheckRunScript(fn func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error)) func() {
	orig := precheckRunScriptFn
	precheckRunScriptFn = fn
	return func() { precheckRunScriptFn = orig }
}

// --- probeSingleInstance 测试 ---

func TestProbeSingleInstance_Reachable(t *testing.T) {
	cleanup := setupPrecheckDB(t)
	defer cleanup()

	restore := mockPrecheckRunScript(func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return `{"reachable":true,"host":"10.0.0.18","port":80,"probe":"curl","http_code":"200"}`, nil
	})
	defer restore()

	result := probeSingleInstance(context.Background(), "ins-test", "http://10.0.0.18:80", "testuser", "testkey", "vdb-xxx")
	if !result.Reachable {
		t.Fatalf("expected reachable=true, got false")
	}
	if result.Reason != "" {
		t.Errorf("expected empty reason, got %q", result.Reason)
	}
	if result.VDBInstanceID != "vdb-xxx" {
		t.Errorf("expected VDBInstanceID=vdb-xxx, got %q", result.VDBInstanceID)
	}
}

func TestProbeSingleInstance_Unreachable(t *testing.T) {
	cleanup := setupPrecheckDB(t)
	defer cleanup()

	restore := mockPrecheckRunScript(func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		// TAT 脚本退出码非 0 时，output 为空，stderr/stdout 在 RichError.Detail
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail(`{"reachable":false,"host":"10.0.0.18","port":80,"probe":"curl","http_code":"000"}`)
	})
	defer restore()

	result := probeSingleInstance(context.Background(), "ins-test", "http://10.0.0.18:80", "testuser", "testkey", "vdb-p6cytw9z")
	if result.Reachable {
		t.Fatalf("expected reachable=false, got true")
	}
	if result.Reason != "network_unreachable" {
		t.Errorf("expected reason=network_unreachable, got %q", result.Reason)
	}
	if !strings.Contains(result.Message, "ins-test") {
		t.Errorf("message should contain instance_id, got %q", result.Message)
	}
	if !strings.Contains(result.Message, "vdb-p6cytw9z") {
		t.Errorf("message should contain vdb_instance_id, got %q", result.Message)
	}
	if result.VDBInstanceID != "vdb-p6cytw9z" {
		t.Errorf("expected VDBInstanceID=vdb-p6cytw9z, got %q", result.VDBInstanceID)
	}
}

func TestProbeSingleInstance_NoProbeToolSkipped(t *testing.T) {
	cleanup := setupPrecheckDB(t)
	defer cleanup()

	restore := mockPrecheckRunScript(func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	})
	defer restore()

	result := probeSingleInstance(context.Background(), "ins-test", "http://10.0.0.18:80", "testuser", "testkey", "vdb-xxx")
	if !result.Reachable {
		t.Fatalf("no probe tool should be treated as reachable (conservative)")
	}
	if !result.Skipped {
		t.Errorf("expected skipped=true")
	}
}

func TestProbeSingleInstance_TATError(t *testing.T) {
	cleanup := setupPrecheckDB(t)
	defer cleanup()

	restore := mockPrecheckRunScript(func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", fmt.Errorf("TAT 调用超时")
	})
	defer restore()

	result := probeSingleInstance(context.Background(), "ins-test", "http://10.0.0.18:80", "testuser", "testkey", "vdb-xxx")
	if !result.Reachable {
		t.Fatalf("TAT error should be treated as reachable (conservative)")
	}
	if !result.Skipped {
		t.Errorf("expected skipped=true on TAT error")
	}
}

func TestProbeSingleInstance_ParamsPassthrough(t *testing.T) {
	cleanup := setupPrecheckDB(t)
	defer cleanup()

	var capturedParams map[string]string
	restore := mockPrecheckRunScript(func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		capturedParams = params
		return `{"reachable":true,"host":"10.0.0.18","port":80,"probe":"curl","http_code":"200"}`, nil
	})
	defer restore()

	probeSingleInstance(context.Background(), "ins-aaa", "http://10.0.0.18:80", "myuser", "mykey", "vdb-xxx")

	if capturedParams["vdb_endpoint"] != "http://10.0.0.18:80" {
		t.Errorf("vdb_endpoint = %q", capturedParams["vdb_endpoint"])
	}
	if capturedParams["vdb_username"] != "myuser" {
		t.Errorf("vdb_username = %q", capturedParams["vdb_username"])
	}
	if capturedParams["vdb_api_key"] != "mykey" {
		t.Errorf("vdb_api_key = %q", capturedParams["vdb_api_key"])
	}
	if capturedParams["timeout_sec"] != "5" {
		t.Errorf("timeout_sec = %q", capturedParams["timeout_sec"])
	}
}

// --- PrecheckBatchForProSwitch 测试 ---

func TestPrecheckBatch_EmptyList(t *testing.T) {
	results := PrecheckBatchForProSwitch(context.Background(), nil)
	if len(results) != 0 {
		t.Errorf("expected 0 results for nil input, got %d", len(results))
	}
}

func TestPrecheckBatch_MixedResults(t *testing.T) {
	// mock RunScript: ins-good → reachable, ins-bad → unreachable
	restore := mockPrecheckRunScript(func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		if instanceID == "ins-good" {
			return `{"reachable":true,"host":"10.0.0.18","port":80,"probe":"curl","http_code":"200"}`, nil
		}
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail(`{"reachable":false,"host":"10.0.0.18","port":80,"probe":"curl","http_code":"000"}`)
	})
	defer restore()

	// mock getVDBPoolPrecheckTarget — 用 DB fallback 路径
	cleanup := setupPrecheckDB(t)
	defer cleanup()
	// 插入一个已有 PRO 实例作为 fallback 凭证
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:      "ins-existing-pro",
		CurrentPlan:     model.MemoryPlanPro,
		Endpoint:        "http://10.0.0.18:80",
		VdbUsername:     "testaccount",
		ApiKeySecretRef: "testpassword",
		PoolID:          "vdb-p6cytw9z",
	})

	results := PrecheckBatchForProSwitch(context.Background(), []string{"ins-good", "ins-bad"})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if !results["ins-good"].Reachable {
		t.Errorf("ins-good should be reachable")
	}
	if results["ins-bad"].Reachable {
		t.Errorf("ins-bad should be unreachable")
	}
	if results["ins-bad"].Reason != "network_unreachable" {
		t.Errorf("ins-bad reason = %q, want network_unreachable", results["ins-bad"].Reason)
	}
	if results["ins-bad"].VDBInstanceID != "vdb-p6cytw9z" {
		t.Errorf("ins-bad VDBInstanceID = %q, want vdb-p6cytw9z", results["ins-bad"].VDBInstanceID)
	}
}

func TestPrecheckBatch_NoPoolEndpoint_SkipsAll(t *testing.T) {
	// 无任何 PRO 实例，getVDBPoolPrecheckTarget fallback 也拿不到凭证
	cleanup := setupPrecheckDB(t)
	defer cleanup()

	// mock RunScript 不应该被调用
	called := false
	restore := mockPrecheckRunScript(func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		called = true
		return "", nil
	})
	defer restore()

	results := PrecheckBatchForProSwitch(context.Background(), []string{"ins-a", "ins-b"})
	if called {
		t.Errorf("RunScript should NOT be called when no pool endpoint available")
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, id := range []string{"ins-a", "ins-b"} {
		if !results[id].Reachable {
			t.Errorf("%s should be reachable (skip mode)", id)
		}
		if !results[id].Skipped {
			t.Errorf("%s should be skipped", id)
		}
	}
}

// --- getVDBPoolPrecheckTargetFallback 测试 ---

func TestGetVDBPoolPrecheckTargetFallback_HasProInstance(t *testing.T) {
	cleanup := setupPrecheckDB(t)
	defer cleanup()
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:      "ins-pro-1",
		CurrentPlan:     model.MemoryPlanPro,
		Endpoint:        "http://10.0.0.18:80",
		VdbUsername:     "user1",
		ApiKeySecretRef: "key1",
		PoolID:          "mp-abc",
	})

	ep, user, pass, vdbID, err := getVDBPoolPrecheckTargetFallback(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep != "http://10.0.0.18:80" {
		t.Errorf("endpoint = %q", ep)
	}
	if user != "user1" {
		t.Errorf("user = %q", user)
	}
	if pass != "key1" {
		t.Errorf("pass = %q", pass)
	}
	if vdbID != "mp-abc" {
		t.Errorf("vdbID = %q", vdbID)
	}
}

func TestGetVDBPoolPrecheckTargetFallback_NoProInstance(t *testing.T) {
	cleanup := setupPrecheckDB(t)
	defer cleanup()

	_, _, _, _, err := getVDBPoolPrecheckTargetFallback(context.Background())
	if err == nil {
		t.Fatalf("expected error when no PRO instance, got nil")
	}
}

// --- HandleAdminMemoryPlanSwitch 集成：预检拒绝 ---

func TestBatchSwitch_PrecheckRejectsUnreachable(t *testing.T) {
	setupMemoryProDB(t)
	AdminToken = "test-admin-token"
	defer func() { AdminToken = "" }()

	// 创建实例 + plugin
	model.DB(context.Background()).Create(&model.Instance{Name: "test-precheck", InstanceId: "ins-precheck", AgentType: "openclaw"})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-precheck", CurrentPlan: model.MemoryPlanOff})

	// mock RunScript: 返回不通
	restore := mockPrecheckRunScript(func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail(`{"reachable":false,"host":"10.0.0.18","port":80,"probe":"curl","http_code":"000"}`)
	})
	defer restore()

	// 插入一个 PRO 实例用于 fallback 凭证
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-pro-existing", CurrentPlan: model.MemoryPlanPro, Endpoint: "http://10.0.0.18:80", VdbUsername: "u", ApiKeySecretRef: "k", PoolID: "vdb-test"})

	req, w := makeAdminRequest("POST", "/admin/memory/plan/switch",
		`{"instance_ids":["ins-precheck"],"target_plan":"pro"}`)
	HandleAdminMemoryPlanSwitch(w, req)

	// 全部 rejected 时返回 HTTP 422
	if w.Code != 422 {
		t.Fatalf("status = %d, want 422, body: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	// 顶层应有 error 字段
	if result["error"] == nil || result["error"] == "" {
		t.Errorf("top-level error should not be empty")
	}
	results := result["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	item := results[0].(map[string]any)
	if item["status"] != "rejected" {
		t.Errorf("status = %v, want rejected", item["status"])
	}
	if item["reason"] != "network_unreachable" {
		t.Errorf("reason = %v, want network_unreachable", item["reason"])
	}
	if item["message"] == nil || item["message"] == "" {
		t.Errorf("message should not be empty")
	}
	// 验证不创建任务
	var jobCount int64
	model.DB(context.Background()).Model(&model.TdaiJob{}).Where("instance_id = ?", "ins-precheck").Count(&jobCount)
	if jobCount != 0 {
		t.Errorf("no job should be created for rejected instance, got %d", jobCount)
	}
}

func TestBatchSwitch_TargetNotPro_NoPrecheck(t *testing.T) {
	setupMemoryProDB(t)
	AdminToken = "test-admin-token"
	defer func() { AdminToken = "" }()

	model.DB(context.Background()).Create(&model.Instance{Name: "test-off", InstanceId: "ins-off", AgentType: "openclaw"})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-off", CurrentPlan: model.MemoryPlanFree})

	// mock: 不应被调用
	called := false
	restore := mockPrecheckRunScript(func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		called = true
		return "", nil
	})
	defer restore()

	req, w := makeAdminRequest("POST", "/admin/memory/plan/switch",
		`{"instance_ids":["ins-off"],"target_plan":"off"}`)
	HandleAdminMemoryPlanSwitch(w, req)

	if called {
		t.Errorf("RunScript should NOT be called for target_plan=off")
	}
	if w.Code != 200 {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}
