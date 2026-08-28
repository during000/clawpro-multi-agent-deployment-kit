package controller

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"
)

// =====================================================================
// memory_plan.go 中有两类接口：
//
//  1. 用户端接口（requireLogin 认证）：
//     - HandleMemoryPlanSwitch
//     - HandleMemoryConfig
//     - HandleMemoryTaskDetail
//     - HandleMemoryLibraryDetail
//     这些接口在 AdminToken 模式下 user.ID=0，getLoginUser 会视为未登录。
//     测试仅覆盖 MethodNotAllowed + Unauthorized 分支。
//     参数校验和业务逻辑通过 admin 版接口（HandleAdminMemoryPlanSwitch 等）
//     和 model/task 层单测间接覆盖。
//
//  2. 管控端接口（requireAdmin 认证）：
//     - HandleAdminMemoryTDAIConfig
//     - HandleAdminUpdateMemoryTDAIConfig
//     这些已在 admin_memory_pro_test.go 中测试，此处补充遗漏的测试。
// =====================================================================

// --- HandleMemoryPlanSwitch ---

func TestUserPlanSwitch_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	req := httptest.NewRequest("GET", "/openclaw/memory/plan/switch", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleMemoryPlanSwitch(w, req)
	if w.Code != 405 {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestUserPlanSwitch_Unauthorized(t *testing.T) {
	setupMemoryProDB(t)
	req := httptest.NewRequest("POST", "/openclaw/memory/plan/switch",
		strings.NewReader(`{"instance_id":"ins-001","target_plan":"free"}`))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleMemoryPlanSwitch(w, req)
	// requireLogin 未认证 → 401/403
	if w.Code != 401 && w.Code != 403 {
		t.Fatalf("status = %d, want 401 or 403", w.Code)
	}
}

// --- HandleMemoryConfig ---

func TestMemoryConfig_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	req := httptest.NewRequest("POST", "/openclaw/memory/config?instance_id=ins-001", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleMemoryConfig(w, req)
	if w.Code != 405 {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestMemoryConfig_Unauthorized(t *testing.T) {
	setupMemoryProDB(t)
	req := httptest.NewRequest("GET", "/openclaw/memory/config?instance_id=ins-001", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleMemoryConfig(w, req)
	if w.Code != 401 {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// --- HandleMemoryTaskDetail ---

func TestMemoryTaskDetail_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	req := httptest.NewRequest("POST", "/openclaw/memory/task?task_id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleMemoryTaskDetail(w, req)
	if w.Code != 405 {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestMemoryTaskDetail_Unauthorized(t *testing.T) {
	setupMemoryProDB(t)
	req := httptest.NewRequest("GET", "/openclaw/memory/task?task_id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleMemoryTaskDetail(w, req)
	if w.Code != 401 {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// --- HandleMemoryLibraryDetail ---

func TestLibraryDetail_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	req := httptest.NewRequest("POST", "/openclaw/memory/library/detail", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleMemoryLibraryDetail(w, req)
	if w.Code != 405 {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestLibraryDetail_Unauthorized(t *testing.T) {
	setupMemoryProDB(t)
	req := httptest.NewRequest("GET",
		"/openclaw/memory/library/detail?instance_id=ins-001&type=persona", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleMemoryLibraryDetail(w, req)
	if w.Code != 401 {
		t.Fatalf("status = %d, want 401 (unauthorized)", w.Code)
	}
}

// --- HandleAdminMemoryTDAIConfig ---

func TestMemoryTDAIConfig_GET(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("GET", "/admin/memory-tdai/config", "")
	HandleAdminMemoryTDAIConfig(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	config := result["config"].(map[string]any)
	if config["memory_tdai_enable"] != false {
		t.Errorf("memory_tdai_enable = %v, want false", config["memory_tdai_enable"])
	}
}

func TestMemoryTDAIConfig_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("POST", "/admin/memory-tdai/config", "")
	HandleAdminMemoryTDAIConfig(w, req)
	if w.Code != 405 {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

// --- HandleAdminUpdateMemoryTDAIConfig ---

func TestUpdateMemoryTDAIConfig_Success(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("PUT", "/admin/memory-tdai/config", `{"memory_tdai_enable":true}`)
	HandleAdminUpdateMemoryTDAIConfig(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	if result["memory_tdai_enable"] != true {
		t.Errorf("got %v, want true", result["memory_tdai_enable"])
	}
}

func TestUpdateMemoryTDAIConfig_MissingField(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("PUT", "/admin/memory-tdai/config", `{}`)
	HandleAdminUpdateMemoryTDAIConfig(w, req)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestUpdateMemoryTDAIConfig_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := makeAdminRequest("GET", "/admin/memory-tdai/config", "")
	HandleAdminUpdateMemoryTDAIConfig(w, req)
	if w.Code != 405 {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

// --- HandleAdminMemoryOverview (admin) ---

func TestMemoryOverview_NoAuth(t *testing.T) {
	setupMemoryProDB(t)
	req := httptest.NewRequest("GET", "/admin/memory/overview", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleAdminMemoryOverview(w, req)
	// requireAdmin 无认证 → 401
	if w.Code != 401 {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestMemoryOverview_NoSiteConfig(t *testing.T) {
	// 即使配置中 Pro 未开通（SDK 调用失败），也应正常返回 plan_stats
	setupMemoryProDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 创建一些 plugin 行 + 对应 Instance 行（agent_type=openclaw 才会被统计）
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ov-1", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ov-2", AgentType: model.AgentTypeOpenClaw})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-ov-1", CurrentPlan: model.MemoryPlanFree})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-ov-2", CurrentPlan: model.MemoryPlanOff})

	req, w := makeAdminRequest("GET", "/admin/memory/overview", "")
	HandleAdminMemoryOverview(w, req)

	// SDK 调用会失败（无真实凭证），但 handler 不应 500，应 fallback 返回空的 pro_capacity
	if w.Code != 200 {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	planStats := result["plan_stats"].(map[string]any)
	if planStats["FREE"].(float64) != 1 {
		t.Errorf("FREE count = %v, want 1", planStats["FREE"])
	}
}
