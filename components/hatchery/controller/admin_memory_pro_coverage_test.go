package controller

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"hatchery/model"
)

// --- HandleAdminMemoryOverview ---

// plan_stats 应只统计支持记忆的 agent_type，并以 instances 表为主口径，
// 没有 plugin 行的实例视为 OFF。
func TestAdminMemoryOverview_PlanStatsFiltersAgentType(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// openclaw 实例：1 FREE + 1 PRO + 2 OFF（一条有 OFF plugin 行，一条没有 plugin 行）
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ow-free", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-ow-free", CurrentPlan: model.MemoryPlanFree})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ow-pro", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-ow-pro", CurrentPlan: model.MemoryPlanPro})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ow-off1", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-ow-off1", CurrentPlan: model.MemoryPlanOff})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ow-off2", AgentType: model.AgentTypeOpenClaw}) // 无 plugin 行

	// hermes 实例：现在支持记忆，应被统计
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-hm-free", AgentType: "hermes"})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-hm-free", CurrentPlan: model.MemoryPlanFree})
	req, w := makeAdminRequest("GET", "/admin/memory/overview", "")
	HandleAdminMemoryOverview(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)

	if total, _ := result["total_instances"].(float64); int(total) != 5 {
		t.Errorf("total_instances = %v, want 5", result["total_instances"])
	}
	stats, _ := result["plan_stats"].(map[string]any)
	if f, _ := stats["FREE"].(float64); int(f) != 2 {
		t.Errorf("FREE = %v, want 2", stats["FREE"])
	}
	if p, _ := stats["PRO"].(float64); int(p) != 1 {
		t.Errorf("PRO = %v, want 1", stats["PRO"])
	}
	// OFF：1 条明确 OFF + 1 条无 plugin 行 = 2
	if o, _ := stats["OFF"].(float64); int(o) != 2 {
		t.Errorf("OFF = %v, want 2 (1 plugin row + 1 no-plugin)", stats["OFF"])
	}
}

// 已软删的实例不应被统计。
func TestAdminMemoryOverview_FiltersSoftDeleted(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 活实例
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-alive", AgentType: model.AgentTypeOpenClaw})
	// 软删实例 + 残留 plugin 行
	inst := &model.Instance{InstanceId: "ins-deleted", AgentType: model.AgentTypeOpenClaw}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Delete(inst) // 软删
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-deleted", CurrentPlan: model.MemoryPlanPro})

	req, w := makeAdminRequest("GET", "/admin/memory/overview", "")
	HandleAdminMemoryOverview(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	if total, _ := result["total_instances"].(float64); int(total) != 1 {
		t.Errorf("total_instances = %v, want 1 (软删的应排除)", result["total_instances"])
	}
	stats, _ := result["plan_stats"].(map[string]any)
	if p, _ := stats["PRO"].(float64); int(p) != 0 {
		t.Errorf("软删实例的 PRO 不应被统计，got PRO = %v", stats["PRO"])
	}
}

// --- HandleAdminMemoryInstances 列表接口 ---

func TestAdminMemoryInstances_FiltersAgentType(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-list-openclaw", Name: "openclaw-1", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-list-hermes", Name: "hermes-1", AgentType: "hermes"})

	req, w := makeAdminRequest("GET", "/admin/memory/instances", "")
	HandleAdminMemoryInstances(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	items, _ := result["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items len = %d, want 2 (openclaw + hermes both support memory)", len(items))
	}
}

func TestAdminMemoryInstances_KeywordFilter(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-alpha", Name: "alpha-name", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-beta", Name: "beta-name", AgentType: model.AgentTypeOpenClaw})

	req, w := makeAdminRequest("GET", "/admin/memory/instances?keyword=alpha", "")
	HandleAdminMemoryInstances(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	result := parseJSON(t, w)
	items, _ := result["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
}

func TestAdminMemoryInstances_PlanFilter(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-pf-free", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-pf-free", CurrentPlan: model.MemoryPlanFree})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-pf-off", AgentType: model.AgentTypeOpenClaw})

	req, w := makeAdminRequest("GET", "/admin/memory/instances?plan=free", "")
	HandleAdminMemoryInstances(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	result := parseJSON(t, w)
	items, _ := result["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	row, _ := items[0].(map[string]any)
	if row["current_plan"] != "FREE" {
		t.Errorf("current_plan = %v, want FREE", row["current_plan"])
	}
}

func TestAdminMemoryInstances_Pagination(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 创建 15 个实例
	for i := 0; i < 15; i++ {
		model.DB(context.Background()).Create(&model.Instance{
			InstanceId: fmt.Sprintf("ins-page-%02d", i),
			Name:       fmt.Sprintf("page-%02d", i),
			AgentType:  model.AgentTypeOpenClaw,
		})
	}

	// 默认分页 10 条
	req, w := makeAdminRequest("GET", "/admin/memory/instances", "")
	HandleAdminMemoryInstances(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	result := parseJSON(t, w)
	if items, _ := result["items"].([]any); len(items) != 10 {
		t.Errorf("default page_size: got %d items, want 10", len(items))
	}
	if total, _ := result["total"].(float64); int(total) != 15 {
		t.Errorf("total = %v, want 15", total)
	}

	// page=2 应返回剩下 5 条
	req, w = makeAdminRequest("GET", "/admin/memory/instances?page=2&page_size=10", "")
	HandleAdminMemoryInstances(w, req)
	result = parseJSON(t, w)
	if items, _ := result["items"].([]any); len(items) != 5 {
		t.Errorf("page 2: got %d items, want 5", len(items))
	}
}

// --- HandleAdminMemoryPlanSwitch 更多边界 ---

func TestBatchSwitch_UnsupportedAgentType(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ace-switch", Name: "ace", AgentType: "lightclawace"})
	req, w := makeAdminRequest("POST", "/admin/memory/plan/switch",
		`{"instance_ids":["ins-ace-switch"],"target_plan":"free"}`)
	HandleAdminMemoryPlanSwitch(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200 (单条错误不影响批量)", w.Code)
	}
	result := parseJSON(t, w)
	results, _ := result["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	row, _ := results[0].(map[string]any)
	errMsg, _ := row["error"].(string)
	if !strings.Contains(errMsg, "不支持记忆功能") {
		t.Errorf("error should mention 不支持记忆功能, got: %s", errMsg)
	}
}

// PRO→FREE 在接口层应被拦截（需先切 OFF）。
func TestBatchSwitch_ProToFreeBlocked(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-p2f", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "ins-p2f",
		CurrentPlan: model.MemoryPlanPro,
		PoolID:      "space-xxx",
	})

	req, w := makeAdminRequest("POST", "/admin/memory/plan/switch",
		`{"instance_ids":["ins-p2f"],"target_plan":"free"}`)
	HandleAdminMemoryPlanSwitch(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	result := parseJSON(t, w)
	results, _ := result["results"].([]any)
	row, _ := results[0].(map[string]any)
	errMsg, _ := row["error"].(string)
	if !strings.Contains(errMsg, "Pro") || !strings.Contains(errMsg, "Free") {
		t.Errorf("error should mention Pro→Free 限制, got: %s", errMsg)
	}
}

// 切换时已有 SwitchStatus（正在切换）应被拦截。
func TestBatchSwitch_SwitchingInProgress(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-switching", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "ins-switching",
		CurrentPlan:  model.MemoryPlanOff,
		SwitchStatus: model.MemorySwitchStatusSwitchingToFree,
	})

	req, w := makeAdminRequest("POST", "/admin/memory/plan/switch",
		`{"instance_ids":["ins-switching"],"target_plan":"free"}`)
	HandleAdminMemoryPlanSwitch(w, req)

	result := parseJSON(t, w)
	results, _ := result["results"].([]any)
	row, _ := results[0].(map[string]any)
	errMsg, _ := row["error"].(string)
	if !strings.Contains(errMsg, "进行中") {
		t.Errorf("error should mention 进行中的切换, got: %s", errMsg)
	}
}

// 正常 OFF→FREE：应成功提交 job，plugin 行被更新 switch_status。
func TestBatchSwitch_SuccessSubmitsJob(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ok", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "ins-ok",
		CurrentPlan: model.MemoryPlanOff,
	})

	req, w := makeAdminRequest("POST", "/admin/memory/plan/switch",
		`{"instance_ids":["ins-ok"],"target_plan":"free"}`)
	HandleAdminMemoryPlanSwitch(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	results, _ := result["results"].([]any)
	row, _ := results[0].(map[string]any)
	if row["task_id"] == nil {
		t.Fatalf("should have task_id, got: %v", row)
	}
	// plugin 行 switch_status 应被设置
	plugin := model.GetMemoryTDAIPlugin(context.Background(), "ins-ok")
	if plugin.SwitchStatus != model.MemorySwitchStatusSwitchingToFree {
		t.Errorf("switch_status = %q, want SWITCHING_TO_FREE", plugin.SwitchStatus)
	}
}

// --- releaseProMemSpaceForInstance 快速分支 ---

// agent_type 不支持 → 直接返回 true，不调用 SDK
func TestReleaseProMemSpaceForInstance_UnsupportedAgentType(t *testing.T) {
	setupMemoryProDB(t)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ace-rel", AgentType: "lightclawace"})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID: "ins-ace-rel", CurrentPlan: model.MemoryPlanPro,
		PoolID: "space-not-release",
	})

	result := ReleaseProMemSpaceForInstance(context.Background(), "ins-ace-rel")
	if !result {
		t.Error("应直接放行（agent_type 不支持记忆）")
	}
	// plugin 行应保持不变（不清绑定信息）
	plugin := model.GetMemoryTDAIPlugin(context.Background(), "ins-ace-rel")
	if plugin.PoolID != "space-not-release" {
		t.Errorf("pool_id 应保持不变，got %q", plugin.PoolID)
	}
}

// plugin 行不存在 → 返回 true
func TestReleaseProMemSpaceForInstance_NoPlugin(t *testing.T) {
	setupMemoryProDB(t)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-no-plugin-rel", AgentType: model.AgentTypeOpenClaw})

	result := ReleaseProMemSpaceForInstance(context.Background(), "ins-no-plugin-rel")
	if !result {
		t.Error("plugin 行不存在应直接放行")
	}
}

// PoolID 为空 → 返回 true
func TestReleaseProMemSpaceForInstance_EmptyPoolID(t *testing.T) {
	setupMemoryProDB(t)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-empty-pool-rel", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "ins-empty-pool-rel",
		CurrentPlan: model.MemoryPlanOff,
		PoolID:      "",
	})

	if !ReleaseProMemSpaceForInstance(context.Background(), "ins-empty-pool-rel") {
		t.Error("PoolID 为空应直接放行")
	}
}

// SDK 初始化失败 → 返回 false（保留本地绑定信息以便后续清理）
func TestReleaseProMemSpaceForInstance_SDKInitFailed(t *testing.T) {
	setupMemoryProDB(t)
	origEnvID := os.Getenv("MEMORY_API_SECRET_ID")
	os.Unsetenv("MEMORY_API_SECRET_ID")
	defer func() {
		if origEnvID != "" {
			os.Setenv("MEMORY_API_SECRET_ID", origEnvID)
		}
	}()

	// current_plan != PRO + 有残留 pool_id：跳过导出直接尝试释放（SDK 初始化失败）
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-sdkf-rel", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "ins-sdkf-rel",
		CurrentPlan:  model.MemoryPlanOff,
		PoolID:       "space-sdkf",
		DatabaseName: "db-sdkf",
	})

	// requireExport=false 跳过导出
	result := ReleaseProMemSpaceForMissingInstance(context.Background(), "ins-sdkf-rel")
	if result {
		t.Error("SDK 初始化失败应返回 false")
	}
	// 本地绑定信息应保留
	plugin := model.GetMemoryTDAIPlugin(context.Background(), "ins-sdkf-rel")
	if plugin.PoolID != "space-sdkf" {
		t.Errorf("pool_id 应保留，got %q", plugin.PoolID)
	}
}

// --- HandleAdminMemoryProActivate 参数校验 ---

func TestAdminMemoryProActivate_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("GET", "/admin/memory/pro/activate", "")
	HandleAdminMemoryProActivate(w, req)
	if w.Code != 405 {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestAdminMemoryProActivate_InvalidBody(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("POST", "/admin/memory/pro/activate", `not-json`)
	HandleAdminMemoryProActivate(w, req)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestAdminMemoryProActivate_NegativeLimit(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("POST", "/admin/memory/pro/activate", `{"memory_limit":0}`)
	HandleAdminMemoryProActivate(w, req)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// --- HandleAdminMemoryProRelease 参数校验 ---

func TestAdminMemoryProRelease_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("GET", "/admin/memory/pro/release", "")
	HandleAdminMemoryProRelease(w, req)
	if w.Code != 405 {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}
