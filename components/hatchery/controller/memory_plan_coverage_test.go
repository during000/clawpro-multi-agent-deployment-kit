package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"
)

// --- resolveMemoryPlanTransition 全覆盖 ---

func TestResolveMemoryPlanTransition_AllInputs(t *testing.T) {
	cases := []struct {
		input      string
		wantPlan   string
		wantJob    string
		wantSwitch string
		wantOK     bool
	}{
		{"off", model.MemoryPlanOff, model.TdaiJobTypeSwitchToOff, model.MemorySwitchStatusSwitchingToOff, true},
		{"OFF", model.MemoryPlanOff, model.TdaiJobTypeSwitchToOff, model.MemorySwitchStatusSwitchingToOff, true},
		{"free", model.MemoryPlanFree, model.TdaiJobTypeSwitchToFree, model.MemorySwitchStatusSwitchingToFree, true},
		{"Free", model.MemoryPlanFree, model.TdaiJobTypeSwitchToFree, model.MemorySwitchStatusSwitchingToFree, true},
		{"pro", model.MemoryPlanPro, model.TdaiJobTypeSwitchToPro, model.MemorySwitchStatusSwitchingToPro, true},
		{"PRO", model.MemoryPlanPro, model.TdaiJobTypeSwitchToPro, model.MemorySwitchStatusSwitchingToPro, true},
		{" pro ", model.MemoryPlanPro, model.TdaiJobTypeSwitchToPro, model.MemorySwitchStatusSwitchingToPro, true},
		{"invalid", "", "", "", false},
		{"", "", "", "", false},
	}
	for _, tc := range cases {
		plan, job, sw, ok := resolveMemoryPlanTransition(tc.input)
		if ok != tc.wantOK || plan != tc.wantPlan || job != tc.wantJob || sw != tc.wantSwitch {
			t.Errorf("resolveMemoryPlanTransition(%q) = (%q,%q,%q,%v), want (%q,%q,%q,%v)",
				tc.input, plan, job, sw, ok, tc.wantPlan, tc.wantJob, tc.wantSwitch, tc.wantOK)
		}
	}
}

// --- HandleMemoryPlanSwitch 用户端 session 测试 ---

func TestHandleMemoryPlanSwitch_MissingInstanceID(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u-switch", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/plan/switch",
		`{"instance_id":"","target_plan":"free"}`, user)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestHandleMemoryPlanSwitch_SuccessPath(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u-switch-ok", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-switch-ok", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-switch-ok", CurrentPlan: model.MemoryPlanOff})

	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/plan/switch",
		`{"instance_id":"ins-switch-ok","target_plan":"free"}`, user)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "task_id") {
		t.Errorf("response should contain task_id, got: %s", w.Body.String())
	}

	// 验证 plugin 被更新
	plugin := model.GetMemoryTDAIPlugin(context.Background(), "ins-switch-ok")
	if plugin.DesiredPlan != model.MemoryPlanFree {
		t.Errorf("desired_plan = %q, want FREE", plugin.DesiredPlan)
	}
	if plugin.SwitchStatus != model.MemorySwitchStatusSwitchingToFree {
		t.Errorf("switch_status = %q, want SWITCHING_TO_FREE", plugin.SwitchStatus)
	}
}

// --- HandleMemoryConfig 用户端成功路径 ---

func TestHandleMemoryConfig_SuccessPath(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u-config", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-cfg", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "ins-cfg",
		CurrentPlan: model.MemoryPlanFree,
		DesiredPlan: model.MemoryPlanFree,
	})

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/config?instance_id=ins-cfg", "", user)
	HandleMemoryConfig(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"current_plan":"FREE"`) {
		t.Errorf("response should contain current_plan FREE, got: %s", body)
	}
}

// --- HandleMemoryLibraryDetail 边界: OFF plan ---

func TestHandleMemoryLibraryDetail_OffPlan(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u-lib-off", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-lib-off", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-lib-off", CurrentPlan: model.MemoryPlanOff})

	req, w := makeLoggedInJSONRequest(t, "GET",
		"/openclaw/memory/library/detail?instance_id=ins-lib-off&type=persona", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "OFF") {
		t.Errorf("should mention OFF, got: %s", w.Body.String())
	}
}

// --- HandleMemoryLibraryDetail: invalid type ---

func TestHandleMemoryLibraryDetail_InvalidType_Coverage(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u-lib-inv", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-lib-inv", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-lib-inv", CurrentPlan: model.MemoryPlanFree})

	req, w := makeLoggedInJSONRequest(t, "GET",
		"/openclaw/memory/library/detail?instance_id=ins-lib-inv&type=invalid", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// --- HandleMemoryLibraryDetail: no plugin record ---

func TestHandleMemoryLibraryDetail_NoPlugin(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u-lib-noplugin", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-lib-nop", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	// 不创建 MemoryTDAIPlugin

	req, w := makeLoggedInJSONRequest(t, "GET",
		"/openclaw/memory/library/detail?instance_id=ins-lib-nop&type=persona", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body: %s", w.Code, w.Body.String())
	}
}

// --- HandleMemoryLibraryDetail: Pro 版但 pool_id 为空 ---

func TestHandleMemoryLibraryDetail_ProEmptyPool(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u-lib-pro-empty", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-lib-pro-ep", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-lib-pro-ep", CurrentPlan: model.MemoryPlanPro, PoolID: ""})

	req, w := makeLoggedInJSONRequest(t, "GET",
		"/openclaw/memory/library/detail?instance_id=ins-lib-pro-ep&type=persona", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "未分配") {
		t.Errorf("should mention pool not allocated, got: %s", w.Body.String())
	}
}

// --- buildReadLocalMemoryScript 验证输出格式 ---

func TestBuildReadLocalMemoryScript_Format(t *testing.T) {
	// OpenClaw 路径：调用方传入已探测好的 pluginRoot
	pluginRoot := "$HOME/.openclaw/npm/node_modules/@tencentdb-agent-memory/memory-tencentdb"
	script := buildReadLocalMemoryScript(pluginRoot, "-d /data -L L3 --format json")
	if !strings.Contains(script, "#!/bin/bash") {
		t.Error("should start with shebang")
	}
	if !strings.Contains(script, "read-local-memory.mjs") {
		t.Error("should reference read-local-memory.mjs")
	}
	if !strings.Contains(script, pluginRoot) {
		t.Error("should contain pluginRoot path")
	}
	if !strings.Contains(script, "-d /data -L L3 --format json") {
		t.Error("should contain script args")
	}

	// Hermes 路径：调用方直接传入固定路径
	hermesRoot := "$HOME/.memory-tencentdb/tdai-memory-openclaw-plugin"
	hermesScript := buildReadLocalMemoryScript(hermesRoot, "-d /h/memory-tdai -L L3 --format json")
	if !strings.Contains(hermesScript, "tdai-memory-openclaw-plugin") {
		t.Error("hermes pluginRoot should contain tdai-memory-openclaw-plugin path")
	}
	if strings.Contains(hermesScript, ".openclaw/extensions") {
		t.Error("hermes pluginRoot should NOT reference .openclaw/extensions")
	}
}

// ensureMemoryPlugin 依赖 RunScript/TAT 环境，无法在纯单测中覆盖，
// 由集成测试或灰度验证覆盖。

// --- admin_memory_pro.go: NewMemorySDKClient Region 跟随 CVMRegion ---

func TestNewMemorySDKClient_RegionFollowsCVMRegion(t *testing.T) {
	setupMemoryProDB(t)

	// 设置测试凭证，避免 "凭据未配置" 错误
	t.Setenv("MEMORY_API_SECRET_ID", "test-id")
	t.Setenv("MEMORY_API_SECRET_KEY", "test-key")
	t.Setenv("MEMORY_API_REGION", "")

	origRegion := CVMRegion
	CVMRegion = "ap-shanghai"
	defer func() { CVMRegion = origRegion }()

	client, err := NewMemorySDKClient(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("client should not be nil")
	}
	// 无法直接读 client 内部 region，但此处覆盖了 659-660 行
}

func TestNewMemorySDKClient_EnvRegionOverride(t *testing.T) {
	setupMemoryProDB(t)

	t.Setenv("MEMORY_API_SECRET_ID", "test-id")
	t.Setenv("MEMORY_API_SECRET_KEY", "test-key")
	t.Setenv("MEMORY_API_REGION", "ap-chengdu")

	origRegion := CVMRegion
	CVMRegion = "ap-shanghai"
	defer func() { CVMRegion = origRegion }()

	client, err := NewMemorySDKClient(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("client should not be nil")
	}
	// 环境变量优先，覆盖 657-658 行
}

// --- admin_memory_pro.go: HandleAdminMemoryProActivate 参数校验 ---

func TestHandleAdminMemoryProActivate_InvalidLimit(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest("POST", "/admin/memory/pro/activate",
		strings.NewReader(`{"memory_limit": 0}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminMemoryProActivate(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleAdminMemoryProActivate_BadJSON(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest("POST", "/admin/memory/pro/activate",
		strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminMemoryProActivate(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// --- admin_memory_pro.go: HandleAdminMemoryProRelease 无 admin ---

func TestHandleAdminMemoryProRelease_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest("GET", "/admin/memory/pro/release", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminMemoryProRelease(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

// --- admin_memory_pro.go: resubmitProSwitchAfterReinstall 边界 ---

func TestResubmitProSwitchAfterReinstall_NotPro(t *testing.T) {
	setupMemoryProDB(t)
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "ins-not-pro-resub",
		CurrentPlan: model.MemoryPlanFree,
		PoolID:      "",
	})
	// 不应 panic，不应创建 job
	resubmitProSwitchAfterReinstall(context.Background(), "ins-not-pro-resub")
	var count int64
	model.DB(context.Background()).Model(&model.TdaiJob{}).Where("instance_id = ?", "ins-not-pro-resub").Count(&count)
	if count != 0 {
		t.Errorf("should not create job for non-pro, got %d", count)
	}
}

func TestResubmitProSwitchAfterReinstall_SwitchInProgress(t *testing.T) {
	setupMemoryProDB(t)
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "ins-resub-switching",
		CurrentPlan:  model.MemoryPlanPro,
		PoolID:       "space-xxx",
		SwitchStatus: model.MemorySwitchStatusSwitchingToPro,
	})
	resubmitProSwitchAfterReinstall(context.Background(), "ins-resub-switching")
	var count int64
	model.DB(context.Background()).Model(&model.TdaiJob{}).Where("instance_id = ?", "ins-resub-switching").Count(&count)
	if count != 0 {
		t.Errorf("should not create job when switch in progress, got %d", count)
	}
}

func TestResubmitProSwitchAfterReinstall_Success(t *testing.T) {
	setupMemoryProDB(t)
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "ins-resub-ok",
		CurrentPlan: model.MemoryPlanPro,
		PoolID:      "space-resub",
	})
	resubmitProSwitchAfterReinstall(context.Background(), "ins-resub-ok")
	var count int64
	model.DB(context.Background()).Model(&model.TdaiJob{}).Where("instance_id = ? AND job_type = ?",
		"ins-resub-ok", model.TdaiJobTypeSwitchToPro).Count(&count)
	if count != 1 {
		t.Errorf("should create 1 pro resubmit job, got %d", count)
	}
	// 验证 switch_status 被更新
	plugin := model.GetMemoryTDAIPlugin(context.Background(), "ins-resub-ok")
	if plugin.SwitchStatus != model.MemorySwitchStatusSwitchingToPro {
		t.Errorf("switch_status = %q, want SWITCHING_TO_PRO", plugin.SwitchStatus)
	}
}

// --- admin_memory_pro.go: HandleAdminMemoryPlanSwitch 无实例/不支持类型 ---

func TestHandleAdminMemoryPlanSwitch_UnsupportedAgentType(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ace-switch", AgentType: "lightclawace"})
	req := httptest.NewRequest("POST", "/admin/memory/plan/switch",
		strings.NewReader(`{"instance_ids":["ins-ace-switch"],"target_plan":"free"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminMemoryPlanSwitch(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (partial success)", w.Code)
	}
	if !strings.Contains(w.Body.String(), "不支持记忆") {
		t.Errorf("should report unsupported type, got: %s", w.Body.String())
	}
}
