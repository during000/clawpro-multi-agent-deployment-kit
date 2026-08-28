package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tat "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat/v20201028"
)

// seedSchedule 在 DB 创建一条 schedule，绕过 gorm default 回写问题强制写 enabled/is_running。
func seedSchedule(t *testing.T, ctx context.Context, s *model.AgentCommandSchedule) {
	t.Helper()
	enabled, running := s.Enabled, s.IsRunning
	if s.Slug == "" {
		s.Slug = model.GenerateScheduleSlug()
	}
	if err := model.DB(ctx).Create(s).Error; err != nil {
		t.Fatalf("seed schedule: %v", err)
	}
	if err := model.DB(ctx).Exec(
		"UPDATE agent_command_schedules SET enabled = ?, is_running = ? WHERE id = ?",
		boolToInt(enabled), boolToInt(running), s.ID,
	).Error; err != nil {
		t.Fatalf("seed schedule enabled: %v", err)
	}
	s.Enabled, s.IsRunning = enabled, running
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// seedCommand 创建一个 agent command 模板，返回其 ID。
func seedCommand(t *testing.T, ctx context.Context, name string, ownerID uint) uint {
	t.Helper()
	cmd := &model.AgentCommand{
		Name: name, Content: "echo hi", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: ownerID,
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5); err != nil {
		t.Fatalf("seed command: %v", err)
	}
	return cmd.ID
}

// ============================================================================
// Create
// ============================================================================

func TestHandleCreateAgentCommandSchedule_Success(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "admin1")
	cmdID := seedCommand(t, ctx, "cmd-a", u.ID)

	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/schedules/create",
		map[string]any{
			"name":         "每天巡检",
			"command_id":   cmdID,
			"instance_ids": []uint{1, 2},
			"schedule":     "every(d, at=02:00)",
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommandSchedule(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["schedule"] != "every(d, at=02:00)" {
		t.Fatalf("unexpected schedule echo: %v", resp["schedule"])
	}
	if resp["status"] != model.ScheduleStatusPending {
		t.Fatalf("new schedule status = %v, want pending", resp["status"])
	}
}

func TestHandleCreateAgentCommandSchedule_Invalid(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "admin1")
	cmdID := seedCommand(t, ctx, "cmd-a", u.ID)

	cases := []struct {
		name string
		body map[string]any
	}{
		{"空名称", map[string]any{"name": "", "command_id": cmdID, "instance_ids": []uint{1}, "schedule": "every(d, at=02:00)"}},
		{"非法表达式", map[string]any{"name": "x", "command_id": cmdID, "instance_ids": []uint{1}, "schedule": "weekly"}},
		{"命令不存在", map[string]any{"name": "x", "command_id": 99999, "instance_ids": []uint{1}, "schedule": "every(d, at=02:00)"}},
		{"无实例", map[string]any{"name": "x", "command_id": cmdID, "instance_ids": []uint{}, "schedule": "every(d, at=02:00)"}},
		{"过期 once", map[string]any{"name": "x", "command_id": cmdID, "instance_ids": []uint{1}, "schedule": "once(2020-01-01 00:00)"}},
	}
	for _, c := range cases {
		req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/schedules/create", c.body, "admin1")
		rr := httptest.NewRecorder()
		HandleCreateAgentCommandSchedule(rr, req)
		if rr.Code < 400 {
			t.Errorf("%s: expected error status, got %d (%s)", c.name, rr.Code, rr.Body.String())
		}
	}
}

// ============================================================================
// List + status 筛选
// ============================================================================

func TestHandleListAgentCommandSchedules_StatusFilter(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "admin1")
	ran := time.Now()
	future := ran.Add(time.Hour)

	// pending（rate，从未触发）
	seedSchedule(t, ctx, &model.AgentCommandSchedule{Name: "p", ScheduleExpr: "every(d, at=09:00)", Enabled: true, NextRunAt: &future, CreatedByUserID: u.ID})
	// running
	seedSchedule(t, ctx, &model.AgentCommandSchedule{Name: "r", ScheduleExpr: "every(d, at=09:00)", Enabled: true, LastRunAt: &ran, IsRunning: true, CreatedByUserID: u.ID})
	// completed（once 已执行）
	seedSchedule(t, ctx, &model.AgentCommandSchedule{Name: "c", ScheduleExpr: "once(2026-06-30 15:00)", Enabled: false, LastRunAt: &ran, CreatedByUserID: u.ID})

	// 不带筛选 → 3 条
	assertScheduleListCount(t, "", 3)
	// 各状态筛选
	assertScheduleListCount(t, "pending", 1)
	assertScheduleListCount(t, "running", 1)
	assertScheduleListCount(t, "completed", 1)
	assertScheduleListCount(t, "waiting", 0)

	// 非法 status → 400
	req := adminSessionReq(t, http.MethodGet, "/admin/agent-commands/schedules?status=bogus", nil, "admin1")
	rr := httptest.NewRecorder()
	HandleListAgentCommandSchedules(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bogus status = %d, want 400", rr.Code)
	}
}

func assertScheduleListCount(t *testing.T, status string, want int) {
	t.Helper()
	path := "/admin/agent-commands/schedules"
	if status != "" {
		path += "?status=" + status
	}
	req := adminSessionReq(t, http.MethodGet, path, nil, "admin1")
	rr := httptest.NewRecorder()
	HandleListAgentCommandSchedules(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status=%q code=%d body=%s", status, rr.Code, rr.Body.String())
	}
	var resp struct {
		Schedules []map[string]any `json:"schedules"`
		Total     int64            `json:"total"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if int(resp.Total) != want || len(resp.Schedules) != want {
		t.Errorf("status=%q total=%d len=%d, want %d", status, resp.Total, len(resp.Schedules), want)
	}
}

// ============================================================================
// Delete
// ============================================================================

func TestHandleDeleteAgentCommandSchedule(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	owner := makeAdminUser(t, ctx, "owner")
	other := makeAdminUser(t, ctx, "other")
	s := &model.AgentCommandSchedule{Name: "d", ScheduleExpr: "every(d, at=09:00)", Enabled: true, CreatedByUserID: owner.ID}
	seedSchedule(t, ctx, s)

	// 非创建者非初始管理员 → 403
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/schedules/delete", map[string]any{"id": s.Slug}, "other")
	rr := httptest.NewRecorder()
	HandleDeleteAgentCommandSchedule(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("other delete = %d, want 403", rr.Code)
	}
	_ = other

	// 创建者删除成功
	req = adminSessionReq(t, http.MethodPost, "/admin/agent-commands/schedules/delete", map[string]any{"id": s.Slug}, "owner")
	rr = httptest.NewRecorder()
	HandleDeleteAgentCommandSchedule(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("owner delete = %d, body=%s", rr.Code, rr.Body.String())
	}

	// not found
	req = adminSessionReq(t, http.MethodPost, "/admin/agent-commands/schedules/delete", map[string]any{"id": "sch-nonexist"}, "owner")
	rr = httptest.NewRecorder()
	HandleDeleteAgentCommandSchedule(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("not found = %d, want 404", rr.Code)
	}
}

// ============================================================================
// Toggle
// ============================================================================

func TestHandleToggleAgentCommandSchedule(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "admin1")
	ran := time.Now()

	// 已完成 once → 启停被 409 拒绝
	done := &model.AgentCommandSchedule{Name: "done", ScheduleExpr: "once(2026-06-30 15:00)", Enabled: false, LastRunAt: &ran, CreatedByUserID: u.ID}
	seedSchedule(t, ctx, done)
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/schedules/toggle", map[string]any{"id": done.Slug, "enabled": true}, "admin1")
	rr := httptest.NewRecorder()
	HandleToggleAgentCommandSchedule(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("toggle completed = %d, want 409 (body=%s)", rr.Code, rr.Body.String())
	}

	// 正常 rate：停用成功
	rate := &model.AgentCommandSchedule{Name: "rate", ScheduleExpr: "every(d, at=09:00)", Enabled: true, CreatedByUserID: u.ID}
	seedSchedule(t, ctx, rate)
	req = adminSessionReq(t, http.MethodPost, "/admin/agent-commands/schedules/toggle", map[string]any{"id": rate.Slug, "enabled": false}, "admin1")
	rr = httptest.NewRecorder()
	HandleToggleAgentCommandSchedule(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("toggle off = %d, body=%s", rr.Code, rr.Body.String())
	}
	// 重新启用：rate 永远能算出下次 → 成功
	req = adminSessionReq(t, http.MethodPost, "/admin/agent-commands/schedules/toggle", map[string]any{"id": rate.Slug, "enabled": true}, "admin1")
	rr = httptest.NewRecorder()
	HandleToggleAgentCommandSchedule(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("toggle on = %d, body=%s", rr.Code, rr.Body.String())
	}

	// not found
	req = adminSessionReq(t, http.MethodPost, "/admin/agent-commands/schedules/toggle", map[string]any{"id": "sch-nope", "enabled": true}, "admin1")
	rr = httptest.NewRecorder()
	HandleToggleAgentCommandSchedule(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("toggle not found = %d, want 404", rr.Code)
	}

	// 非创建者非初始管理员 → 403
	makeAdminUser(t, ctx, "other")
	req = adminSessionReq(t, http.MethodPost, "/admin/agent-commands/schedules/toggle", map[string]any{"id": rate.Slug, "enabled": false}, "other")
	rr = httptest.NewRecorder()
	HandleToggleAgentCommandSchedule(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("toggle by other = %d, want 403", rr.Code)
	}

	// 过期 once（未执行 → paused，可启停）启用时触发时刻已过 → 400
	expired := &model.AgentCommandSchedule{Name: "exp", ScheduleExpr: "once(2020-01-01 00:00)", Enabled: false, CreatedByUserID: u.ID}
	seedSchedule(t, ctx, expired)
	req = adminSessionReq(t, http.MethodPost, "/admin/agent-commands/schedules/toggle", map[string]any{"id": expired.Slug, "enabled": true}, "admin1")
	rr = httptest.NewRecorder()
	HandleToggleAgentCommandSchedule(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("toggle expired once = %d, want 400 (body=%s)", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// Records
// ============================================================================

func TestHandleListAgentCommandScheduleRecords(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "admin1")
	cmdID := seedCommand(t, ctx, "cmd-a", u.ID)
	s := &model.AgentCommandSchedule{Name: "rec", ScheduleExpr: "every(d, at=09:00)", Enabled: true, CreatedByUserID: u.ID}
	seedSchedule(t, ctx, s)

	// seed 一个 dispatch + 一条 record 指向它
	finished := time.Now()
	seedDispatchRow(t, ctx, "slugREC", cmdID, u.ID, "{}", seedDispatchOpts{
		Status: model.AgentDispatchStatusSuccess, TargetCount: 2, SuccessCount: 2, FinishedAt: &finished,
	})
	if err := model.CreateScheduleRecord(ctx, s.ID, "slugREC"); err != nil {
		t.Fatalf("create record: %v", err)
	}

	req := adminSessionReq(t, http.MethodGet, "/admin/agent-commands/schedules/records?schedule_id="+s.Slug, nil, "admin1")
	rr := httptest.NewRecorder()
	HandleListAgentCommandScheduleRecords(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("records = %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Records []map[string]any `json:"records"`
		Total   int64            `json:"total"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Total != 1 || len(resp.Records) != 1 {
		t.Fatalf("records total=%d len=%d, want 1", resp.Total, len(resp.Records))
	}
	if resp.Records[0]["status"] != model.AgentDispatchStatusSuccess {
		t.Fatalf("record status = %v, want success", resp.Records[0]["status"])
	}
}

// ============================================================================
// reconcile：订正 is_running
// ============================================================================

// ============================================================================
// startDispatch：全部离线 → 拒绝
// ============================================================================

func TestStartDispatch_AllOfflineRejected(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "admin1")
	cmdID := seedCommand(t, ctx, "c", u.ID)

	off := &model.Instance{Name: "off", InstanceId: "ins-off", UserID: u.ID, LastCVMState: "STOPPED"}
	if err := model.DB(ctx).Create(off).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	_, err := startDispatch(ctx, startDispatchInput{
		CommandID:           cmdID,
		InstanceIDs:         []uint{off.ID},
		TriggeredByUserID:   u.ID,
		AllowPartialOffline: true,
	})
	if err == nil || !strings.Contains(err.Error(), "all_targets_offline") {
		t.Fatalf("expected all_targets_offline, got %v", err)
	}
}

// ============================================================================
// startDispatch：部分离线 → 在线下发、离线标 unreachable
// ============================================================================

func mockTATSuccess(onlineCVMID string) *mockTATBatchClient {
	return &mockTATBatchClient{
		runCommandResp: &tat.RunCommandResponse{
			Response: &tat.RunCommandResponseParams{InvocationId: common.StringPtr("inv-x")},
		},
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{
					InvocationTaskSet: []*tat.InvocationTask{
						{InvocationTaskId: common.StringPtr("invt-on"), InstanceId: common.StringPtr(onlineCVMID)},
					},
				},
			}, nil
		},
	}
}

func TestStartDispatch_PartialOffline(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	defer withMockTAT(mockTATSuccess("ins-on"))()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "admin1")
	cmdID := seedCommand(t, ctx, "c", u.ID)

	on := &model.Instance{Name: "on", InstanceId: "ins-on", UserID: u.ID, LastCVMState: "RUNNING"}
	off := &model.Instance{Name: "off", InstanceId: "ins-off", UserID: u.ID, LastCVMState: "STOPPED"}
	if err := model.DB(ctx).Create(on).Error; err != nil {
		t.Fatalf("create on: %v", err)
	}
	if err := model.DB(ctx).Create(off).Error; err != nil {
		t.Fatalf("create off: %v", err)
	}

	res, err := startDispatch(ctx, startDispatchInput{
		CommandID:           cmdID,
		InstanceIDs:         []uint{on.ID, off.ID},
		TriggeredByUserID:   u.ID,
		AllowPartialOffline: true,
	})
	if err != nil {
		t.Fatalf("startDispatch: %v", err)
	}
	if agentDispatchAsyncWG != nil {
		agentDispatchAsyncWG.Wait()
	}
	if res.TargetCount != 2 {
		t.Fatalf("target_count = %d, want 2 (含离线)", res.TargetCount)
	}

	// 离线机器 → unreachable 终态
	var offTask model.AgentCommandTask
	_ = model.DB(ctx).Where("dispatch_slug = ? AND instance_id = ?", res.DispatchSlug, off.ID).First(&offTask).Error
	if offTask.Status != model.AgentTaskStatusUnreachable {
		t.Errorf("offline task status = %q, want unreachable", offTask.Status)
	}
	// 在线机器 → 已发出 TAT，in_progress
	var onTask model.AgentCommandTask
	_ = model.DB(ctx).Where("dispatch_slug = ? AND instance_id = ?", res.DispatchSlug, on.ID).First(&onTask).Error
	if onTask.Status != model.AgentTaskStatusInProgress {
		t.Errorf("online task status = %q, want in_progress", onTask.Status)
	}
}

// ============================================================================
// HandleCreateAgentCommandSchedule：本地实例拒绝
// ============================================================================

func TestHandleCreateAgentCommandSchedule_LocalInstance(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "admin1")
	cmdID := seedCommand(t, ctx, "cmd-a", u.ID)

	// 创建一个本地 agent 实例
	localInst := &model.Instance{
		Name:       "local-dev",
		InstanceId: "local-codebuddy-001",
		Source:     model.InstanceSourceLocal,
		UserID:     u.ID,
	}
	if err := model.DB(ctx).Create(localInst).Error; err != nil {
		t.Fatalf("create local instance: %v", err)
	}

	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/schedules/create",
		map[string]any{
			"name":         "本地实例定时任务",
			"command_id":   cmdID,
			"instance_ids": []uint{localInst.ID},
			"schedule":     "every(d, at=02:00)",
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommandSchedule(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "local-dev") {
		t.Fatalf("expected error mentioning local instance name 'local-dev', got body=%s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "local_instance_target_unsupported") {
		t.Fatalf("expected prefix 'local_instance_target_unsupported', got body=%s", rr.Body.String())
	}
}
