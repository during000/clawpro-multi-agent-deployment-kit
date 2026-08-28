package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"
)

func TestHandleOpenClawMemoryTDAIStatus_Unauthorized(t *testing.T) {
	setupMemoryProDB(t)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/memory-tdai-status?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleOpenClawMemoryTDAIStatus(w, req)

	if w.Code != http.StatusUnauthorized && w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 401/403", w.Code)
	}
}

func TestHandleOpenClawMemoryTDAIStatus_MissingID(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "tester", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory-tdai-status", "", user)
	HandleOpenClawMemoryTDAIStatus(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleOpenClawMemoryTDAIStatus_InstanceNotFound(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "tester", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory-tdai-status?id=9999", "", user)
	HandleOpenClawMemoryTDAIStatus(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleOpenClawMemoryTDAIStatus_AgentTypeUnsupported(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "tester", Role: "user"}
	model.DB(context.Background()).Create(&user)

	inst := &model.Instance{
		InstanceId: "ins-ace-status",
		Name:       "ace-test",
		AgentType:  model.AgentTypeLightclawACE,
		UserID:     user.ID,
	}
	model.DB(context.Background()).Create(inst)

	path := fmt.Sprintf("/openclaw/memory-tdai-status?id=%d", inst.ID)
	req, w := makeLoggedInJSONRequest(t, "GET", path, "", user)
	HandleOpenClawMemoryTDAIStatus(w, req)

	if w.Code != 403 {
		t.Fatalf("status = %d, want 403, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "不支持记忆功能") {
		t.Errorf("body should contain '不支持记忆功能', got: %s", w.Body.String())
	}
}

func TestHandleOpenClawMemoryTDAIStatus_NoPluginRow(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "tester", Role: "user"}
	model.DB(context.Background()).Create(&user)

	inst := &model.Instance{
		InstanceId: "ins-no-plugin",
		Name:       "no-plugin",
		AgentType:  model.AgentTypeOpenClaw,
		UserID:     user.ID,
	}
	model.DB(context.Background()).Create(inst)

	path := fmt.Sprintf("/openclaw/memory-tdai-status?id=%d", inst.ID)
	req, w := makeLoggedInJSONRequest(t, "GET", path, "", user)
	HandleOpenClawMemoryTDAIStatus(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	if result["status"] != model.MemoryTDAIPluginStatusNotInstalled {
		t.Errorf("status = %v, want %s", result["status"], model.MemoryTDAIPluginStatusNotInstalled)
	}
	if _, ok := result["last_error"]; ok {
		t.Error("last_error should not be present when empty")
	}
}

func TestHandleOpenClawMemoryTDAIStatus_PluginExists(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "tester", Role: "user"}
	model.DB(context.Background()).Create(&user)

	inst := &model.Instance{
		InstanceId: "ins-with-plugin",
		Name:       "plugin-installed",
		AgentType:  model.AgentTypeOpenClaw,
		UserID:     user.ID,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  inst.InstanceId,
		Status:      model.MemoryTDAIPluginStatusEnabled,
		CurrentPlan: model.MemoryPlanFree,
	})

	path := fmt.Sprintf("/openclaw/memory-tdai-status?id=%d", inst.ID)
	req, w := makeLoggedInJSONRequest(t, "GET", path, "", user)
	HandleOpenClawMemoryTDAIStatus(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	if result["status"] != model.MemoryTDAIPluginStatusEnabled {
		t.Errorf("status = %v, want %s", result["status"], model.MemoryTDAIPluginStatusEnabled)
	}
}

func TestHandleOpenClawMemoryTDAIStatus_WithLastError(t *testing.T) {
	setupMemoryProDB(t)
	user := model.User{Username: "tester", Role: "user"}
	model.DB(context.Background()).Create(&user)

	inst := &model.Instance{
		InstanceId: "ins-err",
		Name:       "with-error",
		AgentType:  model.AgentTypeOpenClaw,
		UserID:     user.ID,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  inst.InstanceId,
		Status:      model.MemoryTDAIPluginStatusFailed,
		CurrentPlan: model.MemoryPlanOff,
		LastError:   "script timeout",
	})

	path := fmt.Sprintf("/openclaw/memory-tdai-status?id=%d", inst.ID)
	req, w := makeLoggedInJSONRequest(t, "GET", path, "", user)
	HandleOpenClawMemoryTDAIStatus(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	result := parseJSON(t, w)
	if result["last_error"] != "script timeout" {
		t.Errorf("last_error = %v, want 'script timeout'", result["last_error"])
	}
}

func TestHandleOpenClawMemoryTDAIStatus_OtherUserInstance(t *testing.T) {
	setupMemoryProDB(t)
	owner := model.User{Username: "owner", Role: "user"}
	model.DB(context.Background()).Create(&owner)
	other := model.User{Username: "other", Role: "user"}
	model.DB(context.Background()).Create(&other)

	inst := &model.Instance{
		InstanceId: "ins-owner",
		Name:       "owner-test",
		AgentType:  model.AgentTypeOpenClaw,
		UserID:     owner.ID,
	}
	model.DB(context.Background()).Create(inst)

	path := fmt.Sprintf("/openclaw/memory-tdai-status?id=%d", inst.ID)
	req, w := makeLoggedInJSONRequest(t, "GET", path, "", other)
	HandleOpenClawMemoryTDAIStatus(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 (实例不存在), body: %s", w.Code, w.Body.String())
	}
}
