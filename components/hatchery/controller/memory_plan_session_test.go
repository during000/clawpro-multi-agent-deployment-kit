package controller

import (
	"context"
	"fmt"
	"testing"

	"hatchery/model"
)

func createMemoryTestUser(t *testing.T, username, role string) model.User {
	t.Helper()
	user := model.User{Username: username, Password: "x", Role: role}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func TestUserPlanSwitch_SuccessWithSession(t *testing.T) {
	setupMemoryProDB(t)
	user := createMemoryTestUser(t, "alice", "user")
	if err := model.DB(context.Background()).Create(&model.Instance{Name: "demo", InstanceId: "ins-user-free", UserID: user.ID}).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/plan/switch", `{"instance_id":"ins-user-free","target_plan":"free"}`, user)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	plugin := model.GetMemoryTDAIPlugin(context.Background(), "ins-user-free")
	if plugin == nil {
		t.Fatal("expected plugin row")
	}
	if plugin.DesiredPlan != model.MemoryPlanFree {
		t.Fatalf("desired_plan = %q, want FREE", plugin.DesiredPlan)
	}
	if plugin.SwitchStatus != model.MemorySwitchStatusSwitchingToFree {
		t.Fatalf("switch_status = %q, want SWITCHING_TO_FREE", plugin.SwitchStatus)
	}
	if plugin.LastTaskID == 0 {
		t.Fatal("expected last_task_id to be set")
	}

	var job model.TdaiJob
	if err := model.DB(context.Background()).First(&job, plugin.LastTaskID).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	if job.JobType != model.TdaiJobTypeSwitchToFree {
		t.Fatalf("job_type = %q, want SWITCH_TO_FREE", job.JobType)
	}
	if job.Operator != user.Username {
		t.Fatalf("operator = %q, want %q", job.Operator, user.Username)
	}
}

func TestUserPlanSwitch_ConflictWhenSwitching(t *testing.T) {
	setupMemoryProDB(t)
	user := createMemoryTestUser(t, "alice", "user")
	model.DB(context.Background()).Create(&model.Instance{Name: "demo", InstanceId: "ins-busy-user", UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-busy-user", SwitchStatus: model.MemorySwitchStatusSwitchingToPro})

	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/plan/switch", `{"instance_id":"ins-busy-user","target_plan":"free"}`, user)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != 409 {
		t.Fatalf("status = %d, want 409, body: %s", w.Code, w.Body.String())
	}
}

func TestUserPlanSwitch_ForbiddenForOtherUser(t *testing.T) {
	setupMemoryProDB(t)
	owner := createMemoryTestUser(t, "owner", "user")
	visitor := createMemoryTestUser(t, "visitor", "user")
	model.DB(context.Background()).Create(&model.Instance{Name: "demo", InstanceId: "ins-owner-only", UserID: owner.ID})

	req, w := makeLoggedInJSONRequest(t, "POST", "/openclaw/memory/plan/switch", `{"instance_id":"ins-owner-only","target_plan":"free"}`, visitor)
	HandleMemoryPlanSwitch(w, req)
	if w.Code != 403 {
		t.Fatalf("status = %d, want 403, body: %s", w.Code, w.Body.String())
	}
}

func TestMemoryConfig_SuccessWithSession(t *testing.T) {
	setupMemoryProDB(t)
	user := createMemoryTestUser(t, "alice", "user")
	model.DB(context.Background()).Create(&model.Instance{Name: "demo", InstanceId: "ins-config-ok", UserID: user.ID})
	job, err := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:ins-config-ok", "ins-config-ok", "{}", user.Username, "")
	if err != nil {
		t.Fatalf("submit job: %v", err)
	}
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "ins-config-ok",
		CurrentPlan:  model.MemoryPlanFree,
		DesiredPlan:  model.MemoryPlanFree,
		SwitchStatus: model.MemorySwitchStatusNone,
		LastTaskID:   job.ID,
		Status:       model.MemoryTDAIPluginStatusEnabled,
	})

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/config?instance_id=ins-config-ok", "", user)
	HandleMemoryConfig(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	resp := parseJSON(t, w)
	if resp["current_plan"] != model.MemoryPlanFree {
		t.Fatalf("current_plan = %v, want FREE", resp["current_plan"])
	}
	lastTask := resp["last_task"].(map[string]any)
	if lastTask["job_type"] != model.TdaiJobTypeSwitchToFree {
		t.Fatalf("job_type = %v, want SWITCH_TO_FREE", lastTask["job_type"])
	}
}

func TestMemoryTaskDetail_SuccessWithSession(t *testing.T) {
	setupMemoryProDB(t)
	user := createMemoryTestUser(t, "alice", "user")
	model.DB(context.Background()).Create(&model.Instance{Name: "demo", InstanceId: "ins-task-ok", UserID: user.ID})
	job, err := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToPro, "switch:ins-task-ok", "ins-task-ok", "{}", user.Username, "")
	if err != nil {
		t.Fatalf("submit job: %v", err)
	}

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/task?task_id="+fmt.Sprintf("%d", job.ID), "", user)
	req.URL.RawQuery = "task_id=" + fmt.Sprintf("%d", job.ID)
	HandleMemoryTaskDetail(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	resp := parseJSON(t, w)
	if resp["job_type"] != job.JobType {
		t.Fatalf("job_type = %v, want %s", resp["job_type"], job.JobType)
	}
}

func TestLibraryDetail_InvalidSubType_WithSession(t *testing.T) {
	setupMemoryProDB(t)
	user := createMemoryTestUser(t, "alice", "user")
	model.DB(context.Background()).Create(&model.Instance{Name: "demo", InstanceId: "ins-lib-sub", UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-lib-sub", CurrentPlan: model.MemoryPlanFree})

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/library/detail?instance_id=ins-lib-sub&type=memory&sub_type=bad", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400, body: %s", w.Code, w.Body.String())
	}
}

func TestLibraryDetail_OffPlan_WithSession(t *testing.T) {
	setupMemoryProDB(t)
	user := createMemoryTestUser(t, "alice", "user")
	model.DB(context.Background()).Create(&model.Instance{Name: "demo", InstanceId: "ins-lib-off", UserID: user.ID})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: "ins-lib-off", CurrentPlan: model.MemoryPlanOff})

	req, w := makeLoggedInJSONRequest(t, "GET", "/openclaw/memory/library/detail?instance_id=ins-lib-off&type=persona", "", user)
	HandleMemoryLibraryDetail(w, req)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400, body: %s", w.Code, w.Body.String())
	}
}
