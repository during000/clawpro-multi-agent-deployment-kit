package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"hatchery/model"
)

type agentTaskFixture struct {
	User          *model.User
	Instance      *model.Instance
	Project       model.Project
	WorkspacePath string
	Cookie        string
}

func seedAgentTaskFixture(t *testing.T, username string) agentTaskFixture {
	t.Helper()
	localAgentID := "0123456789abcdef"
	instanceCID := formatLocalInstanceID("codebuddy", localAgentID)
	user, inst := seedLocalUserAndInstanceWithCID(t, username, "codebuddy", instanceCID)
	project := model.Project{Identifier: "test-tenant", Name: "协作项目", CreatedBy: user.ID}
	if err := model.DB(context.Background()).Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.ProjectMember{
		Identifier: "test-tenant", ProjectID: project.ID, UserID: user.ID, CreatedBy: user.ID,
	}).Error; err != nil {
		t.Fatalf("create project member: %v", err)
	}
	workspacePath := "/Users/alice/code/repo-x"
	if err := model.DB(context.Background()).Create(&model.LocalAgentScopeBinding{
		Identifier: "test-tenant", InstanceID: inst.ID, Scope: model.LocalAgentScopeWorkspace,
		ScopeKey: workspacePath, ScopeName: "repo-x", IDEType: "codebuddy", ProjectID: project.ID,
	}).Error; err != nil {
		t.Fatalf("create workspace binding: %v", err)
	}
	return agentTaskFixture{
		User: user, Instance: inst, Project: project, WorkspacePath: workspacePath, Cookie: loginCookie(t, username),
	}
}

func doAgentTaskCreate(t *testing.T, cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/agent-tasks/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rr := httptest.NewRecorder()
	HandleAgentTaskCreate(rr, req)
	return rr
}

func TestHandleAgentTaskCreate_Success(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	fixture := seedAgentTaskFixture(t, "alice")
	body := `{"instance_id":` + strconv.Itoa(int(fixture.Instance.ID)) +
		`,"project_id":` + strconv.Itoa(int(fixture.Project.ID)) +
		`,"workspace_path":"` + fixture.WorkspacePath + `","prompt":"修复登录失败问题"}`
	rr := doAgentTaskCreate(t, fixture.Cookie, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Task agentTaskResponse `json:"task"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Task.ID == 0 || resp.Task.Status != model.LocalAgentTaskStatusPending {
		t.Fatalf("unexpected task: %+v", resp.Task)
	}
	var stored model.LocalAgentTask
	if err := model.DB(context.Background()).First(&stored, resp.Task.ID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if stored.Type != model.LocalAgentTaskTypeExecuteAgent || stored.AgentType != "codebuddy" || stored.ProjectID != fixture.Project.ID {
		t.Fatalf("unexpected stored task: %+v", stored)
	}
}

func TestHandleAgentTaskCreate_IMateExecutor(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	fixture := seedAgentTaskFixture(t, "alice")
	body, err := json.Marshal(map[string]any{
		"instance_id": fixture.Instance.ID, "project_id": fixture.Project.ID,
		"workspace_path": fixture.WorkspacePath, "prompt": "分析项目风险",
		"executor": "imate", "target_agent_id": "agent-openclaw-1", "imate_project_id": "imate-project-1",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	rr := doAgentTaskCreate(t, fixture.Cookie, string(body))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Task agentTaskResponse `json:"task"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Task.Executor != "imate" || resp.Task.TargetAgentID != "agent-openclaw-1" || resp.Task.IMateProjectID != "imate-project-1" {
		t.Fatalf("unexpected iMate task: %+v", resp.Task)
	}
}

func TestHandleAgentTaskCreate_RejectsUnboundWorkspace(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	fixture := seedAgentTaskFixture(t, "alice")
	body := `{"instance_id":` + strconv.Itoa(int(fixture.Instance.ID)) +
		`,"project_id":` + strconv.Itoa(int(fixture.Project.ID)) +
		`,"workspace_path":"/tmp/not-reported","prompt":"do work"}`
	rr := doAgentTaskCreate(t, fixture.Cookie, body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=400 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAgentTaskCreate_RejectsOtherUsersInstance(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	alice := seedAgentTaskFixture(t, "alice")
	seedLocalUser(t, "bob")
	body := `{"instance_id":` + strconv.Itoa(int(alice.Instance.ID)) +
		`,"project_id":` + strconv.Itoa(int(alice.Project.ID)) +
		`,"workspace_path":"` + alice.WorkspacePath + `","prompt":"do work"}`
	rr := doAgentTaskCreate(t, loginCookie(t, "bob"), body)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=404 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAgentTaskCreate_Validation(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	fixture := seedAgentTaskFixture(t, "alice")
	validPrefix := `{"instance_id":` + strconv.Itoa(int(fixture.Instance.ID)) + `,`
	tests := []struct {
		name   string
		method string
		body   string
		want   int
	}{
		{name: "method", method: http.MethodGet, body: `{}`, want: http.StatusMethodNotAllowed},
		{name: "invalid json", method: http.MethodPost, body: `{`, want: http.StatusBadRequest},
		{name: "missing instance", method: http.MethodPost, body: `{"workspace_path":"x","prompt":"x"}`, want: http.StatusBadRequest},
		{name: "missing workspace", method: http.MethodPost, body: validPrefix + `"prompt":"x"}`, want: http.StatusBadRequest},
		{name: "missing prompt", method: http.MethodPost, body: validPrefix + `"workspace_path":"` + fixture.WorkspacePath + `"}`, want: http.StatusBadRequest},
		{name: "imate missing target", method: http.MethodPost, body: validPrefix + `"workspace_path":"` + fixture.WorkspacePath + `","prompt":"x","executor":"imate","imate_project_id":"p"}`, want: http.StatusBadRequest},
		{name: "imate missing project", method: http.MethodPost, body: validPrefix + `"workspace_path":"` + fixture.WorkspacePath + `","prompt":"x","executor":"imate","target_agent_id":"a"}`, want: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/agent-tasks/create", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Cookie", fixture.Cookie)
			rr := httptest.NewRecorder()
			HandleAgentTaskCreate(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("status=%d want=%d body=%s", rr.Code, tt.want, rr.Body.String())
			}
		})
	}

	for name, payload := range map[string]map[string]any{
		"long workspace": {
			"instance_id": fixture.Instance.ID, "workspace_path": strings.Repeat("x", 513), "prompt": "x",
		},
		"long prompt": {
			"instance_id": fixture.Instance.ID, "workspace_path": fixture.WorkspacePath,
			"prompt": strings.Repeat("x", maxAgentTaskPromptRunes+1),
		},
	} {
		t.Run(name, func(t *testing.T) {
			body, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			rr := doAgentTaskCreate(t, fixture.Cookie, string(body))
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want=400 body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandleAgentTaskCreate_DerivesProjectFromWorkspace(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	fixture := seedAgentTaskFixture(t, "alice")
	body := `{"instance_id":` + strconv.Itoa(int(fixture.Instance.ID)) +
		`,"workspace_path":"` + fixture.WorkspacePath + `","prompt":"do work"}`
	rr := doAgentTaskCreate(t, fixture.Cookie, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Task agentTaskResponse `json:"task"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Task.ProjectID != fixture.Project.ID {
		t.Fatalf("project_id=%d want=%d", resp.Task.ProjectID, fixture.Project.ID)
	}
}

func TestHandleLocalAgentSync_ExecuteAgentTask(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	fixture := seedAgentTaskFixture(t, "alice")
	task := model.LocalAgentTask{
		Identifier: "test-tenant", InstanceID: fixture.Instance.ID, InstanceCID: fixture.Instance.InstanceId,
		Type: model.LocalAgentTaskTypeExecuteAgent, Status: model.LocalAgentTaskStatusPending,
		OperatorID: fixture.User.ID, ProjectID: fixture.Project.ID, WorkspacePath: fixture.WorkspacePath,
		AgentType: "codebuddy", Prompt: "修复登录失败问题",
		Cmd: `{"agent_type":"codebuddy","project_id":1,"workspace_path":"/Users/alice/code/repo-x","prompt":"修复登录失败问题","executor":"imate","target_agent_id":"agent-openclaw-1","imate_project_id":"imate-project-1"}`,
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/local-agent/sync",
		strings.NewReader(`{"agent_type":"codebuddy","local_agent_id":"0123456789abcdef","status":"running"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", fixture.Cookie)
	rr := httptest.NewRecorder()
	HandleLocalAgentSync(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Commands []map[string]any `json:"commands"`
		Cmds     []map[string]any `json:"cmds"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for name, list := range map[string][]map[string]any{"commands": resp.Commands, "cmds": resp.Cmds} {
		var found map[string]any
		for _, item := range list {
			if item["type"] == model.LocalAgentTaskTypeExecuteAgent {
				found = item
				break
			}
		}
		if found == nil || found["prompt"] != "修复登录失败问题" || found["workspace_path"] != fixture.WorkspacePath || found["executor"] != "imate" || found["target_agent_id"] != "agent-openclaw-1" {
			t.Fatalf("%s missing execution task: %+v", name, list)
		}
	}
}

func TestHandleLocalAgentAck_ExecuteAgentTask_ProgressAndSuccess(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	fixture := seedAgentTaskFixture(t, "alice")
	task := model.LocalAgentTask{
		Identifier: "test-tenant", InstanceID: fixture.Instance.ID, InstanceCID: fixture.Instance.InstanceId,
		Type: model.LocalAgentTaskTypeExecuteAgent, Status: model.LocalAgentTaskStatusPending,
		OperatorID: fixture.User.ID, ProjectID: fixture.Project.ID, WorkspacePath: fixture.WorkspacePath,
		AgentType: "codebuddy", Prompt: "修复登录失败问题",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	running := doAckReq(t, fixture.Cookie, `{"id":`+itoaSafe(task.ID)+`,"type":"execute_agent_task","status":"running","result":"开始分析\n","session_id":"session-1"}`)
	if running.Code != http.StatusOK {
		t.Fatalf("running status=%d body=%s", running.Code, running.Body.String())
	}
	success := doAckReq(t, fixture.Cookie, `{"id":`+itoaSafe(task.ID)+`,"type":"execute_agent_task","status":"success","result":"开始分析\n修复完成"}`)
	if success.Code != http.StatusOK {
		t.Fatalf("success status=%d body=%s", success.Code, success.Body.String())
	}
	var stored model.LocalAgentTask
	if err := model.DB(context.Background()).First(&stored, task.ID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if stored.Status != model.LocalAgentTaskStatusSuccess || stored.Result != "开始分析\n修复完成" || stored.SessionID != "session-1" {
		t.Fatalf("unexpected task after ack: %+v", stored)
	}
	if stored.StartedAt == nil || stored.FinishedAt == nil {
		t.Fatalf("timestamps not set: %+v", stored)
	}

	// 终态幂等：后续 ACK 不得覆盖结果。
	idempotent := doAckReq(t, fixture.Cookie, `{"id":`+itoaSafe(task.ID)+`,"type":"execute_agent_task","status":"failed","result":"不应写入","error":"late"}`)
	if idempotent.Code != http.StatusOK {
		t.Fatalf("idempotent status=%d body=%s", idempotent.Code, idempotent.Body.String())
	}
	if err := model.DB(context.Background()).First(&stored, task.ID).Error; err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if stored.Status != model.LocalAgentTaskStatusSuccess || strings.Contains(stored.Result, "不应写入") {
		t.Fatalf("terminal task was overwritten: %+v", stored)
	}
}

func TestHandleLocalAgentAck_ExecuteAgentTask_RejectsInvalidStatusAndOversizedResult(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	fixture := seedAgentTaskFixture(t, "alice")
	task := model.LocalAgentTask{
		Identifier: "test-tenant", InstanceID: fixture.Instance.ID,
		Type: model.LocalAgentTaskTypeExecuteAgent, Status: model.LocalAgentTaskStatusPending,
		OperatorID: fixture.User.ID,
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	invalid := doAckReq(t, fixture.Cookie, `{"id":`+itoaSafe(task.ID)+`,"type":"execute_agent_task","status":"unknown"}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid status=%d want=400 body=%s", invalid.Code, invalid.Body.String())
	}
	oversizedBody, err := json.Marshal(map[string]any{
		"id": task.ID, "type": "execute_agent_task", "status": "running",
		"result": strings.Repeat("x", maxAgentTaskResultBytes+1),
	})
	if err != nil {
		t.Fatalf("marshal oversized ack: %v", err)
	}
	oversized := doAckReq(t, fixture.Cookie, string(oversizedBody))
	if oversized.Code != http.StatusBadRequest {
		t.Fatalf("oversized status=%d want=400 body=%s", oversized.Code, oversized.Body.String())
	}
}

func TestHandleLocalAgentAck_ExecuteAgentTask_FailedAndNotFound(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	fixture := seedAgentTaskFixture(t, "alice")
	task := model.LocalAgentTask{
		Identifier: "test-tenant", InstanceID: fixture.Instance.ID,
		Type: model.LocalAgentTaskTypeExecuteAgent, Status: model.LocalAgentTaskStatusPending,
		OperatorID: fixture.User.ID,
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	failed := doAckReq(t, fixture.Cookie, `{"id":`+itoaSafe(task.ID)+`,"type":"execute_agent_task","status":"failed","result":"最后输出","error":"exit 1"}`)
	if failed.Code != http.StatusOK {
		t.Fatalf("failed status=%d body=%s", failed.Code, failed.Body.String())
	}
	var stored model.LocalAgentTask
	if err := model.DB(context.Background()).First(&stored, task.ID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if stored.Status != model.LocalAgentTaskStatusFailed || stored.Error != "exit 1" || stored.Result != "最后输出" {
		t.Fatalf("unexpected failed task: %+v", stored)
	}
	notFound := doAckReq(t, fixture.Cookie, `{"id":999999,"type":"execute_agent_task","status":"running"}`)
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("not found status=%d want=404 body=%s", notFound.Code, notFound.Body.String())
	}
}

func TestHandleAgentTasks_OnlyReturnsCurrentUserTasks(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	alice := seedAgentTaskFixture(t, "alice")
	bob, bobInst := seedLocalUserAndInstanceWithCID(t, "bob", "codebuddy", formatLocalInstanceID("codebuddy", "fedcba9876543210"))
	for _, task := range []model.LocalAgentTask{
		{Identifier: "test-tenant", InstanceID: alice.Instance.ID, Type: model.LocalAgentTaskTypeExecuteAgent, OperatorID: alice.User.ID, ProjectID: alice.Project.ID, Status: model.LocalAgentTaskStatusPending},
		{Identifier: "test-tenant", InstanceID: bobInst.ID, Type: model.LocalAgentTaskTypeExecuteAgent, OperatorID: bob.ID, ProjectID: alice.Project.ID, Status: model.LocalAgentTaskStatusPending},
	} {
		if err := model.DB(context.Background()).Create(&task).Error; err != nil {
			t.Fatalf("create task: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/agent-tasks?project_id="+strconv.Itoa(int(alice.Project.ID))+"&status=pending", nil)
	req.Header.Set("Cookie", alice.Cookie)
	rr := httptest.NewRecorder()
	HandleAgentTasks(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Tasks []agentTaskResponse `json:"tasks"`
		Total int                 `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Total != 1 || len(resp.Tasks) != 1 || resp.Tasks[0].InstanceID != alice.Instance.ID {
		t.Fatalf("unexpected list response: %+v", resp)
	}
}

func TestHandleAgentTasks_Validation(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	fixture := seedAgentTaskFixture(t, "alice")
	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "method", method: http.MethodPost, path: "/agent-tasks", want: http.StatusMethodNotAllowed},
		{name: "id", method: http.MethodGet, path: "/agent-tasks?id=bad", want: http.StatusBadRequest},
		{name: "project", method: http.MethodGet, path: "/agent-tasks?project_id=bad", want: http.StatusBadRequest},
		{name: "status", method: http.MethodGet, path: "/agent-tasks?status=bad", want: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Cookie", fixture.Cookie)
			rr := httptest.NewRecorder()
			HandleAgentTasks(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("status=%d want=%d body=%s", rr.Code, tt.want, rr.Body.String())
			}
		})
	}
}

func TestIsAgentTaskStatus(t *testing.T) {
	for _, status := range []string{
		model.LocalAgentTaskStatusPending,
		model.LocalAgentTaskStatusRunning,
		model.LocalAgentTaskStatusSuccess,
		model.LocalAgentTaskStatusFailed,
		model.LocalAgentTaskStatusCancelled,
	} {
		if !isAgentTaskStatus(status) {
			t.Fatalf("status %q should be valid", status)
		}
	}
	if isAgentTaskStatus("unknown") {
		t.Fatal("unknown status should be invalid")
	}
}
