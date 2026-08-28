package controller

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"hatchery/model"
)

// --- HandleMemoryPlanSwitch HTTP 测试 ---

func TestHandleMemoryPlanSwitch_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/plan/switch", "", user)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != 405 {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestHandleMemoryPlanSwitch_InvalidJSON(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/plan/switch", `not-json`, user)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestHandleMemoryPlanSwitch_EmptyInstanceID(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/plan/switch",
		`{"instance_id":"","target_plan":"free"}`, user)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestHandleMemoryPlanSwitch_InstanceNotFound(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/plan/switch",
		`{"instance_id":"ins-gone","target_plan":"free"}`, user)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestHandleMemoryPlanSwitch_NotOwnerForbidden(t *testing.T) {
	setupMemoryProDB(t)
	owner := model.User{Username: "owner", Role: "user"}
	model.DB(context.Background()).Create(&owner)
	other := model.User{Username: "other", Role: "user"}
	model.DB(context.Background()).Create(&other)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-owned", AgentType: model.AgentTypeOpenClaw, UserID: owner.ID})

	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/plan/switch",
		`{"instance_id":"ins-owned","target_plan":"free"}`, other)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != 403 {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestHandleMemoryPlanSwitch_AgentTypeUnsupported(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ace-switch2", AgentType: "lightclawace", UserID: user.ID})
	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/plan/switch",
		`{"instance_id":"ins-ace-switch2","target_plan":"free"}`, user)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != 403 {
		t.Fatalf("status = %d, want 403, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "不支持记忆功能") {
		t.Errorf("body 应包含 '不支持记忆功能'")
	}
}

func TestHandleMemoryPlanSwitch_InvalidTargetPlan(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ok-switch", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})

	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/plan/switch",
		`{"instance_id":"ins-ok-switch","target_plan":"invalid"}`, user)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestHandleMemoryPlanSwitch_InProgress(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-in-progress", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "ins-in-progress",
		CurrentPlan:  model.MemoryPlanOff,
		SwitchStatus: model.MemorySwitchStatusSwitchingToFree,
	})

	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/plan/switch",
		`{"instance_id":"ins-in-progress","target_plan":"free"}`, user)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != 409 {
		t.Fatalf("status = %d, want 409, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleMemoryPlanSwitch_ProToFree(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-p2f-h", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "ins-p2f-h",
		CurrentPlan: model.MemoryPlanPro,
		PoolID:      "space-xxx",
	})

	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/plan/switch",
		`{"instance_id":"ins-p2f-h","target_plan":"free"}`, user)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleMemoryPlanSwitch_Success(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ok-sub", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})

	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/plan/switch",
		`{"instance_id":"ins-ok-sub","target_plan":"off"}`, user)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	if result["task_id"] == nil {
		t.Error("response 应含 task_id")
	}
	// plugin 行应被更新
	plugin := model.GetMemoryTDAIPlugin(context.Background(), "ins-ok-sub")
	if plugin.DesiredPlan != model.MemoryPlanOff {
		t.Errorf("desired_plan = %q, want OFF", plugin.DesiredPlan)
	}
}

// --- HandleMemoryConfig HTTP 测试 ---

func TestHandleMemoryConfig_MethodNotAllowed(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/config", "", user)
	HandleMemoryConfig(w, req)
	if w.Code != 405 {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestHandleMemoryConfig_MissingInstanceID(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/config", "", user)
	HandleMemoryConfig(w, req)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestHandleMemoryConfig_InstanceNotFound(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/config?instance_id=ins-gone", "", user)
	HandleMemoryConfig(w, req)
	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestHandleMemoryConfig_NotOwnerForbidden(t *testing.T) {
	setupMemoryProDB(t)
	owner := model.User{Username: "owner", Role: "user"}
	model.DB(context.Background()).Create(&owner)
	other := model.User{Username: "other", Role: "user"}
	model.DB(context.Background()).Create(&other)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-cfg-owned", AgentType: model.AgentTypeOpenClaw, UserID: owner.ID})

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/config?instance_id=ins-cfg-owned", "", other)
	HandleMemoryConfig(w, req)
	if w.Code != 403 {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestHandleMemoryConfig_AgentTypeUnsupported(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-ace-cfg2", AgentType: "lightclawace", UserID: user.ID})
	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/config?instance_id=ins-ace-cfg2", "", user)
	HandleMemoryConfig(w, req)
	if w.Code != 403 {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestHandleMemoryConfig_Success(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-cfg-ok", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "ins-cfg-ok",
		CurrentPlan: model.MemoryPlanFree,
		DesiredPlan: model.MemoryPlanFree,
	})

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/config?instance_id=ins-cfg-ok", "", user)
	HandleMemoryConfig(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	if result["current_plan"] != "FREE" {
		t.Errorf("current_plan = %v, want FREE", result["current_plan"])
	}
}

// --- HandleMemoryTaskDetail HTTP 测试 ---

func TestHandleMemoryTaskDetail_MissingID(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/task", "", user)
	HandleMemoryTaskDetail(w, req)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestHandleMemoryTaskDetail_TaskNotFound(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/task?task_id=99999", "", user)
	HandleMemoryTaskDetail(w, req)
	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestHandleMemoryTaskDetail_Success(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-task-ok", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})
	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:ins-task-ok", "ins-task-ok", "{}", "u1", "")

	path := fmt.Sprintf("/openclaw/memory/task?task_id=%d", job.ID)
	req, w := makeLoggedInJSONRequest(t, "GET", path, "", user)
	HandleMemoryTaskDetail(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
}

// task 关联的实例已被其他用户拥有 → 禁止访问
func TestHandleMemoryTaskDetail_NotOwner(t *testing.T) {
	setupMemoryProDB(t)
	owner := model.User{Username: "owner", Role: "user"}
	model.DB(context.Background()).Create(&owner)
	other := model.User{Username: "other", Role: "user"}
	model.DB(context.Background()).Create(&other)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-task-owned", AgentType: model.AgentTypeOpenClaw, UserID: owner.ID})
	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:ins-task-owned", "ins-task-owned", "{}", "owner", "")

	path := fmt.Sprintf("/openclaw/memory/task?task_id=%d", job.ID)
	req, w := makeLoggedInJSONRequest(t, "GET", path, "", other)
	HandleMemoryTaskDetail(w, req)
	if w.Code != 403 {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

// task 关联的实例是 lightclawace 类型 → 禁止访问
func TestHandleMemoryTaskDetail_AgentTypeUnsupported(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-task-ace2", AgentType: "lightclawace", UserID: user.ID})
	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:ins-task-ace2", "ins-task-ace2", "{}", "u1", "")
	path := fmt.Sprintf("/openclaw/memory/task?task_id=%d", job.ID)
	req, w := makeLoggedInJSONRequest(t, "GET", path, "", user)
	HandleMemoryTaskDetail(w, req)
	if w.Code != 403 {
		t.Fatalf("status = %d, want 403, body: %s", w.Code, w.Body.String())
	}
}

// --- HandleMemoryLibraryDetail HTTP 测试 ---

func TestHandleMemoryLibraryDetail_MissingInstanceID(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/library/detail?type=persona", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestHandleMemoryLibraryDetail_InvalidType(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-lib-ok", AgentType: model.AgentTypeOpenClaw, UserID: user.ID})

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/library/detail?instance_id=ins-lib-ok&type=invalid", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleMemoryLibraryDetail_AgentTypeUnsupported(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-lib-ace", AgentType: "lightclawace", UserID: user.ID})
	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/library/detail?instance_id=ins-lib-ace&type=persona", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != 403 {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestHandleMemoryLibraryDetail_InstanceNotFound(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "u1", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/library/detail?instance_id=ins-gone&type=persona", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
