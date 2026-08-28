package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tat "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat/v20201028"
)

// ============================================================================
// 状态写入 helper 单测
// ============================================================================

func TestSetInvocationStatus_Terminal(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	inv := &model.AgentCommandInvocation{
		DispatchSlug: "task-set1", BatchIndex: 1,
		Status: model.AgentInvocationStatusPending,
	}
	if err := model.DB(ctx).Create(inv).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := setInvocationStatus(ctx, inv.ID, model.AgentInvocationStatusSuccess, "inv-abc"); err != nil {
		t.Fatalf("setInvocationStatus: %v", err)
	}
	var got model.AgentCommandInvocation
	if err := model.DB(ctx).First(&got, inv.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Status != model.AgentInvocationStatusSuccess ||
		got.TATInvocationID != "inv-abc" || got.FinishedAt == nil {
		t.Errorf("got %+v", got)
	}
}

func TestSetTaskTATIDAndStatus(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	tk := &model.AgentCommandTask{
		DispatchSlug: "task-set2", InstanceID: 1,
		Status: model.AgentTaskStatusPending,
	}
	_ = model.DB(ctx).Create(tk).Error

	if err := setTaskTATID(ctx, tk.ID, "invt-xxx"); err != nil {
		t.Fatalf("setTaskTATID: %v", err)
	}
	if err := setTaskStatus(ctx, tk.ID, model.AgentTaskStatusInProgress); err != nil {
		t.Fatalf("setTaskStatus: %v", err)
	}
	var got model.AgentCommandTask
	_ = model.DB(ctx).First(&got, tk.ID).Error
	if got.TATInvocationTaskID != "invt-xxx" || got.Status != model.AgentTaskStatusInProgress {
		t.Errorf("got tat=%q status=%q", got.TATInvocationTaskID, got.Status)
	}

	exit := 7
	if err := setTaskFailed(ctx, tk.ID, model.AgentTaskStatusFailed, &exit); err != nil {
		t.Fatalf("setTaskFailed: %v", err)
	}
	_ = model.DB(ctx).First(&got, tk.ID).Error
	if got.Status != model.AgentTaskStatusFailed || got.ExitCode == nil || *got.ExitCode != 7 ||
		got.FinishedAt == nil {
		t.Errorf("after fail: %+v", got)
	}
}

func TestUpdateInvocationCountsFromTasks(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	inv := &model.AgentCommandInvocation{DispatchSlug: "task-counts", TargetCount: 3}
	_ = model.DB(ctx).Create(inv).Error
	for _, st := range []string{
		model.AgentTaskStatusSuccess,
		model.AgentTaskStatusFailed,
		model.AgentTaskStatusSuccess,
	} {
		_ = model.DB(ctx).Create(&model.AgentCommandTask{
			InvocationID: inv.ID, Status: st,
		}).Error
	}
	updateInvocationCountsFromTasks(ctx, inv.ID)
	var got model.AgentCommandInvocation
	_ = model.DB(ctx).First(&got, inv.ID).Error
	if got.SuccessCount != 2 || got.FailedCount != 1 ||
		got.Status != model.AgentInvocationStatusPartial {
		t.Errorf("got success=%d failed=%d status=%s", got.SuccessCount, got.FailedCount, got.Status)
	}
}

func TestApplyTATDetailToTask(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	tk := &model.AgentCommandTask{
		DispatchSlug:        "task-apply",
		TATInvocationTaskID: "invt-1",
		Status:              model.AgentTaskStatusInProgress,
	}
	_ = model.DB(ctx).Create(tk).Error

	start := time.Now()
	end := start.Add(2 * time.Second)
	exit := int64(0)
	d := InvocationTaskDetail{
		InvocationTaskID: "invt-1",
		TaskStatus:       "SUCCESS",
		ExitCode:         &exit,
		StartTime:        &start,
		EndTime:          &end,
	}
	applyTATDetailToTask(ctx, tk, d)
	var got model.AgentCommandTask
	_ = model.DB(ctx).First(&got, tk.ID).Error
	if got.Status != model.AgentTaskStatusSuccess {
		t.Errorf("status = %s, want success", got.Status)
	}
	if got.ExitCode == nil || *got.ExitCode != 0 {
		t.Errorf("exit_code = %v", got.ExitCode)
	}
	if got.ElapsedMs == nil || *got.ElapsedMs == 0 {
		t.Errorf("elapsed_ms not set: %v", got.ElapsedMs)
	}
}

func TestMarkDispatchAllFailed(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	plan := &dispatchPlan{dispatchSlug: "task-fail"}
	for _, isTest := range []bool{true, false, false} {
		_ = model.DB(ctx).Create(&model.AgentCommandInvocation{
			DispatchSlug: "task-fail",
			IsTestRun:    isTest,
			Status:       model.AgentInvocationStatusPending,
		}).Error
	}
	markDispatchAllFailed(ctx, plan)
	var got []model.AgentCommandInvocation
	_ = model.DB(ctx).Where("dispatch_slug = ?", "task-fail").Find(&got).Error
	for _, r := range got {
		if r.Status != model.AgentInvocationStatusFailed {
			t.Errorf("invocation test=%v still %s", r.IsTestRun, r.Status)
		}
	}
}

// ============================================================================
// 预创建事务
// ============================================================================

func TestPreCreateInvocationsAndTasks(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	plan := &dispatchPlan{
		dispatchSlug:    "task-pre",
		commandID:       1,
		commandSnapshot: `{"a":1}`,
		paramValuesJSON: `{}`,
		triggeredByUID:  1,
		testFirst:       true,
		testInstanceID:  100,
		prodBatches:     [][]uint{{200, 201, 202}, {203}},
		instancesByID: map[uint]*model.Instance{
			100: {Name: "test-machine", InstanceId: "ins-100"},
			200: {Name: "prod-1", InstanceId: "ins-200"},
			201: {Name: "prod-2", InstanceId: "ins-201"},
			202: {Name: "prod-3", InstanceId: "ins-202"},
			203: {Name: "prod-4", InstanceId: "ins-203"},
		},
		startedAt: time.Now(),
	}
	if err := preCreateInvocationsAndTasks(model.DB(ctx), plan); err != nil {
		t.Fatalf("preCreate: %v", err)
	}

	if plan.testInvocationID == 0 || plan.testTaskID == 0 {
		t.Errorf("test fields not filled: invID=%d taskID=%d", plan.testInvocationID, plan.testTaskID)
	}
	if len(plan.prodInvocations) != 2 || plan.prodInvocations[0] == 0 {
		t.Errorf("prodInvocations not filled: %v", plan.prodInvocations)
	}
	if len(plan.prodTaskIDs) != 2 || len(plan.prodTaskIDs[0]) != 3 || len(plan.prodTaskIDs[1]) != 1 {
		t.Errorf("prodTaskIDs structure wrong: %v", plan.prodTaskIDs)
	}

	// 验证 task 元信息已冗余
	var tasks []model.AgentCommandTask
	_ = model.DB(ctx).Where("dispatch_slug = ?", "task-pre").Find(&tasks).Error
	if len(tasks) != 5 { // 1 test + 3 + 1 = 5
		t.Errorf("task count = %d, want 5", len(tasks))
	}
	var hasTest int
	for _, tk := range tasks {
		if tk.IsTestTarget {
			hasTest++
		}
		if tk.AgentName == "" || tk.CVMInstanceID == "" {
			t.Errorf("task %d missing agent meta", tk.ID)
		}
	}
	if hasTest != 1 {
		t.Errorf("test_target count = %d, want 1", hasTest)
	}
}

// ============================================================================
// runDispatchAsync：测试机失败 → 整 dispatch failed
// ============================================================================

// TestRunDispatchAsync_TestFirst_NoAutoChain 测试机成功后，runDispatchAsync 必须**不**自动衔接生产批次。
//
// 改造前的 bug：测试机 success → 立即 for loop 跑 prodBatches。新行为：测试机成功就 return，
// dispatch 状态聚合到 awaiting_confirmation，等用户调 continue。
func TestRunDispatchAsync_TestFirst_NoAutoChain(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()

	mock := &mockTATBatchClient{
		runCommandResp: &tat.RunCommandResponse{
			Response: &tat.RunCommandResponseParams{
				InvocationId: common.StringPtr("inv-test-success"),
			},
		},
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{
					InvocationTaskSet: []*tat.InvocationTask{
						{
							InvocationTaskId: common.StringPtr("invt-test-ok"),
							InstanceId:       common.StringPtr("ins-test"),
							TaskStatus:       common.StringPtr("SUCCESS"),
							TaskResult: &tat.TaskResult{
								ExitCode: common.Int64Ptr(0),
							},
							StartTime: common.StringPtr("2026-05-21 10:00:00"),
							EndTime:   common.StringPtr("2026-05-21 10:00:01"),
						},
					},
				},
			}, nil
		},
	}
	defer withMockTAT(mock)()

	plan := &dispatchPlan{
		dispatchSlug:    "task-noauto",
		commandID:       1,
		commandSnapshot: "{}", paramValuesJSON: "{}", triggeredByUID: 1,
		testFirst:      true,
		testInstanceID: 100,
		prodBatches:    [][]uint{{200, 201}},
		instancesByID: map[uint]*model.Instance{
			100: {Name: "tm", InstanceId: "ins-test"},
			200: {Name: "p1", InstanceId: "ins-prod1"},
			201: {Name: "p2", InstanceId: "ins-prod2"},
		},
		startedAt: time.Now(),
	}
	if err := preCreateInvocationsAndTasks(model.DB(ctx), plan); err != nil {
		t.Fatalf("pre: %v", err)
	}

	runDispatchAsync(ctx, plan, "echo test", 5, "root", "/root", nil)

	// 校验：测试 invocation 已 success
	var testInv model.AgentCommandInvocation
	_ = model.DB(ctx).Where("dispatch_slug = ? AND is_test_run = ?", "task-noauto", true).
		First(&testInv).Error
	if testInv.Status != model.AgentInvocationStatusSuccess {
		t.Errorf("test invocation status = %s, want success", testInv.Status)
	}

	// 校验：生产 invocation 必须仍是 pending，且没有 tat_invocation_id
	var prodInvs []model.AgentCommandInvocation
	_ = model.DB(ctx).Where("dispatch_slug = ? AND is_test_run = ?", "task-noauto", false).
		Find(&prodInvs).Error
	if len(prodInvs) != 1 {
		t.Fatalf("prod invocation count = %d, want 1", len(prodInvs))
	}
	if prodInvs[0].Status != model.AgentInvocationStatusPending {
		t.Errorf("prod invocation status = %s, want pending (no auto-chain)", prodInvs[0].Status)
	}
	if prodInvs[0].TATInvocationID != "" {
		t.Errorf("prod invocation tat_invocation_id = %q, want empty (RunCommand should not be called)",
			prodInvs[0].TATInvocationID)
	}

	// 校验：dispatch 整体聚合 = awaiting_confirmation
	all, _ := model.FindInvocationsByDispatchSlug(ctx, "task-noauto")
	anyInProgress := false
	for i := range all {
		if !all[i].IsTerminal() {
			anyInProgress = true
		}
	}
	got := aggregateDispatchStatus(all, anyInProgress, testInv.SuccessCount, 0)
	if got != model.AgentDispatchStatusAwaitingConfirmation {
		t.Errorf("dispatch aggregate = %s, want awaiting_confirmation", got)
	}
}

func TestRunDispatchAsync_TestRunFailure(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()

	mock := &mockTATBatchClient{
		runCommandResp: &tat.RunCommandResponse{
			Response: &tat.RunCommandResponseParams{
				InvocationId: common.StringPtr("inv-test-fail"),
			},
		},
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{
					InvocationTaskSet: []*tat.InvocationTask{
						{
							InvocationTaskId: common.StringPtr("invt-test"),
							InstanceId:       common.StringPtr("ins-test"),
							TaskStatus:       common.StringPtr("FAILED"),
							TaskResult: &tat.TaskResult{
								ExitCode: common.Int64Ptr(1),
							},
							StartTime: common.StringPtr("2026-05-21 10:00:00"),
							EndTime:   common.StringPtr("2026-05-21 10:00:01"),
						},
					},
				},
			}, nil
		},
	}
	defer withMockTAT(mock)()

	plan := &dispatchPlan{
		dispatchSlug:    "task-asyncfail",
		commandID:       1,
		commandSnapshot: "{}", paramValuesJSON: "{}", triggeredByUID: 1,
		testFirst:      true,
		testInstanceID: 100,
		prodBatches:    [][]uint{{200}},
		instancesByID: map[uint]*model.Instance{
			100: {Name: "tm", InstanceId: "ins-test"},
			200: {Name: "p1", InstanceId: "ins-prod1"},
		},
		startedAt: time.Now(),
	}
	if err := preCreateInvocationsAndTasks(model.DB(ctx), plan); err != nil {
		t.Fatalf("pre: %v", err)
	}

	runDispatchAsync(ctx, plan, "echo test", 5, "root", "/root", nil)

	var invs []model.AgentCommandInvocation
	_ = model.DB(ctx).Where("dispatch_slug = ?", "task-asyncfail").
		Order("batch_index asc").Find(&invs).Error
	if len(invs) != 2 {
		t.Fatalf("invs count = %d, want 2", len(invs))
	}
	for _, inv := range invs {
		if inv.Status != model.AgentInvocationStatusFailed {
			t.Errorf("inv batch=%d test=%v status=%s, want failed",
				inv.BatchIndex, inv.IsTestRun, inv.Status)
		}
	}
	// 生产 task 应保持 pending（"未触发"）
	var tasks []model.AgentCommandTask
	_ = model.DB(ctx).Where("dispatch_slug = ?", "task-asyncfail").Find(&tasks).Error
	for _, tk := range tasks {
		if tk.IsTestTarget {
			continue
		}
		if tk.Status != model.AgentTaskStatusPending {
			t.Errorf("non-test task status=%s, want pending", tk.Status)
		}
	}
}

// TestRunAgentCommandPollerOnce_UsesCtxSnapshot 回归：poller 不再剥
// TenantSnapshot。曾用 InjectTenant({Identifier:identifier}) stub 替换 ctx，
// 导致下游 getCredential 拿不到 Uin → 走永久凭证路径，STS-only 部署里
// TAT 调用静默失败，task 永远停在 in_progress。
//
// 验证：poller 用 ctx 直接调 TAT mock，task 状态从 in_progress → success，
// 同时 invocation 聚合也跟着更新。
func TestRunAgentCommandPollerOnce_UsesCtxSnapshot(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()

	// 准备 1 个 invocation + 1 个 task（TAT id 已写回，status=in_progress）
	inv := &model.AgentCommandInvocation{
		DispatchSlug:    "task-poll-1",
		BatchIndex:      1,
		Status:          model.AgentInvocationStatusInProgress,
		TATInvocationID: "inv-12345",
		TargetCount:     1,
	}
	if err := model.DB(ctx).Create(inv).Error; err != nil {
		t.Fatalf("seed inv: %v", err)
	}
	tk := &model.AgentCommandTask{
		InvocationID:        inv.ID,
		DispatchSlug:        "task-poll-1",
		InstanceID:          1,
		CVMInstanceID:       "ins-aaa",
		TATInvocationTaskID: "invt-task-1",
		Status:              model.AgentTaskStatusInProgress,
	}
	if err := model.DB(ctx).Create(tk).Error; err != nil {
		t.Fatalf("seed task: %v", err)
	}

	// Mock TAT 返回 SUCCESS
	mock := &mockTATBatchClient{
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			zero := int64(0)
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{
					InvocationTaskSet: []*tat.InvocationTask{
						{
							InvocationTaskId: common.StringPtr("invt-task-1"),
							InstanceId:       common.StringPtr("ins-aaa"),
							TaskStatus:       common.StringPtr("SUCCESS"),
							TaskResult: &tat.TaskResult{
								ExitCode: &zero,
							},
						},
					},
				},
			}, nil
		},
	}
	defer withMockTAT(mock)()

	RunAgentCommandPollerOnce(ctx)

	// task 应被更新为 success
	var gotTask model.AgentCommandTask
	if err := model.DB(ctx).First(&gotTask, tk.ID).Error; err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if gotTask.Status != model.AgentTaskStatusSuccess {
		t.Errorf("task status=%s, want success", gotTask.Status)
	}
	if gotTask.ExitCode == nil || *gotTask.ExitCode != 0 {
		t.Errorf("task exit_code=%v, want 0", gotTask.ExitCode)
	}

	// invocation 聚合也应同步刷新
	var gotInv model.AgentCommandInvocation
	if err := model.DB(ctx).First(&gotInv, inv.ID).Error; err != nil {
		t.Fatalf("reload inv: %v", err)
	}
	if gotInv.Status != model.AgentInvocationStatusSuccess {
		t.Errorf("inv status=%s, want success", gotInv.Status)
	}
	if gotInv.SuccessCount != 1 {
		t.Errorf("inv success_count=%d, want 1", gotInv.SuccessCount)
	}
}

// TestHandleListAgentCommandTasks_AutoHealStaleAggregate 回归：list handler
// 自愈逻辑——所有 task 已终态但 invocation.status 还停在 in_progress 时，
// 列表查询应自动重算聚合并返回正确的 dispatch 状态，无需用户先点 detail。
func TestHandleListAgentCommandTasks_AutoHealStaleAggregate(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner")
	cmd := &model.AgentCommand{
		Name: "n", Content: "df -h", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: u.ID,
	}
	_ = model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5)

	snap := `{"name":"n","content":"df -h"}`
	dID := seedDispatchRow(t, ctx, "task-heal-1", cmd.ID, u.ID, snap, seedDispatchOpts{
		Status: model.AgentDispatchStatusInProgress, TargetCount: 2,
	})
	// invocation 故意停在 in_progress + 0/0，模拟 poller 写回 task 但聚合
	// 没跟上的"卡死"状态。
	inv := &model.AgentCommandInvocation{
		DispatchID:   dID,
		DispatchSlug: "task-heal-1",
		BatchIndex:   1,
		Status:       model.AgentInvocationStatusInProgress,
		TargetCount:  2,
		SuccessCount: 0,
		FailedCount:  0,
	}
	if err := model.DB(ctx).Create(inv).Error; err != nil {
		t.Fatalf("seed inv: %v", err)
	}
	// 但 task 已全部终态：1 success + 1 failed
	for _, st := range []string{model.AgentTaskStatusSuccess, model.AgentTaskStatusFailed} {
		if err := model.DB(ctx).Create(&model.AgentCommandTask{
			DispatchID: dID, InvocationID: inv.ID, DispatchSlug: "task-heal-1",
			Status: st,
		}).Error; err != nil {
			t.Fatalf("seed task: %v", err)
		}
	}

	req := adminSessionReq(t, "GET", "/admin/agent-commands/tasks", nil, "owner")
	rr := httptest.NewRecorder()
	HandleListAgentCommandTasks(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp agentTaskListResp
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Tasks) != 1 {
		t.Fatalf("tasks len=%d, want 1; body=%s", len(resp.Tasks), rr.Body.String())
	}
	got := resp.Tasks[0]
	// 1 success + 1 failed → dispatch 应该是 partial（聚合规则）
	if got.Status != "partial" {
		t.Errorf("status=%s, want partial（self-heal 后）", got.Status)
	}
	if got.SuccessCount != 1 || got.FailedCount != 1 {
		t.Errorf("counts: success=%d failed=%d, want 1/1", got.SuccessCount, got.FailedCount)
	}

	// 副作用：DB 中的 invocation 聚合也应该被刷新
	var refreshed model.AgentCommandInvocation
	_ = model.DB(ctx).First(&refreshed, inv.ID).Error
	if refreshed.Status != model.AgentInvocationStatusPartial ||
		refreshed.SuccessCount != 1 || refreshed.FailedCount != 1 {
		t.Errorf("DB invocation 未被自愈刷新: %+v", refreshed)
	}
}

// ============================================================================
// HandleListAgentCommandTasks
// ============================================================================

func TestHandleListAgentCommandTasks_Aggregation(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner")
	cmd := &model.AgentCommand{
		Name: "n", Content: "echo", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: u.ID,
	}
	_ = model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5)
	snap := `{"name":"n","content":"echo"}`

	// dispatch 1: test+prod，6 个 success
	d1 := seedDispatchRow(t, ctx, "task-list1", cmd.ID, u.ID, snap, seedDispatchOpts{
		Status: model.AgentDispatchStatusSuccess, TestFirst: true,
		TargetCount: 6, SuccessCount: 6,
	})
	// dispatch 2: 仅 prod，3 个 failed
	d2 := seedDispatchRow(t, ctx, "task-list2", cmd.ID, u.ID, snap, seedDispatchOpts{
		Status: model.AgentDispatchStatusFailed,
		TargetCount: 3, FailedCount: 3,
	})

	rows := []*model.AgentCommandInvocation{
		{
			DispatchID: d1, DispatchSlug: "task-list1", IsTestRun: true,
			Status: model.AgentInvocationStatusSuccess, BatchIndex: 0,
			TargetCount: 1, SuccessCount: 1,
		},
		{
			DispatchID: d1, DispatchSlug: "task-list1", BatchIndex: 1,
			Status: model.AgentInvocationStatusSuccess,
			TargetCount: 5, SuccessCount: 5,
		},
		{
			DispatchID: d2, DispatchSlug: "task-list2", BatchIndex: 1,
			Status: model.AgentInvocationStatusFailed,
			TargetCount: 3, FailedCount: 3,
		},
	}
	for _, r := range rows {
		_ = model.DB(ctx).Create(r).Error
	}
	req := adminSessionReq(t, "GET", "/admin/agent-commands/tasks", nil, "owner")
	rr := httptest.NewRecorder()
	HandleListAgentCommandTasks(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var resp agentTaskListResp
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("Total = %d, want 2 dispatches", resp.Total)
	}
	for _, item := range resp.Tasks {
		if item.DispatchSlug == "task-list1" {
			if item.SuccessCount != 6 || item.TargetCount != 6 {
				t.Errorf("task-list1 success=%d target=%d", item.SuccessCount, item.TargetCount)
			}
			if !item.TestFirst {
				t.Error("task-list1 should have test_first=true")
			}
			if item.InvocationCount != 2 {
				t.Errorf("invocation_count=%d, want 2", item.InvocationCount)
			}
		}
	}
}

// TestHandleListAgentCommandTasks_PaginationTotal 回归：page_size 小于真实
// dispatch 数时，total 必须如实反映 dispatch 总数。v2 数据模型下分页直接基于
// dispatch 表，total 由 SELECT COUNT(*) 获得，不再受任何内存截断的影响。
func TestHandleListAgentCommandTasks_PaginationTotal(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner")
	cmd := &model.AgentCommand{
		Name: "n", Content: "df -h", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: u.ID,
	}
	_ = model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5)
	snap := `{"name":"n","content":"df -h"}`
	const dispatchCount = 8
	for i := 0; i < dispatchCount; i++ {
		slug := fmt.Sprintf("task-page-%d", i)
		dID := seedDispatchRow(t, ctx, slug, cmd.ID, u.ID, snap, seedDispatchOpts{
			Status: model.AgentDispatchStatusSuccess, TargetCount: 1, SuccessCount: 1,
		})
		inv := &model.AgentCommandInvocation{
			DispatchID:   dID,
			DispatchSlug: slug,
			BatchIndex:   1,
			Status:       model.AgentInvocationStatusSuccess,
			TargetCount:  1, SuccessCount: 1,
		}
		if err := model.DB(ctx).Create(inv).Error; err != nil {
			t.Fatalf("seed inv %d: %v", i, err)
		}
	}

	req := adminSessionReq(t, "GET", "/admin/agent-commands/tasks?page=1&page_size=1", nil, "owner")
	rr := httptest.NewRecorder()
	HandleListAgentCommandTasks(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var resp agentTaskListResp
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if resp.Total != int64(dispatchCount) {
		t.Errorf("Total = %d, want %d (page_size=1 不应截断 total)", resp.Total, dispatchCount)
	}
	if len(resp.Tasks) != 1 {
		t.Errorf("Tasks len = %d, want 1 (page_size=1 这一页正好返回一条)", len(resp.Tasks))
	}
}

// TestHandleListAgentCommandTasks_QSearch 校验 q 参数支持的 4 个字段：
// dispatch_slug / 命令名 / 命令内容 / 操作人用户名。
func TestHandleListAgentCommandTasks_QSearch(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	alice := makeAdminUser(t, ctx, "alice_ops")
	bob := makeAdminUser(t, ctx, "bob_dev")
	cmd := &model.AgentCommand{
		Name: "查看磁盘", Content: "df -h", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: alice.ID,
	}
	_ = model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5)
	snapDisk := `{"name":"查看磁盘","content":"df -h"}`
	snapPing := `{"name":"网络检查","content":"ping baidu.com"}`

	// alice → 查看磁盘
	dAlice := seedDispatchRow(t, ctx, "task-q-alice-disk", cmd.ID, alice.ID, snapDisk, seedDispatchOpts{
		Status: model.AgentDispatchStatusSuccess, TargetCount: 1, SuccessCount: 1,
	})
	// bob → 网络检查
	dBob := seedDispatchRow(t, ctx, "task-q-bob-ping", cmd.ID, bob.ID, snapPing, seedDispatchOpts{
		Status: model.AgentDispatchStatusSuccess, TargetCount: 1, SuccessCount: 1,
	})

	rows := []*model.AgentCommandInvocation{
		{DispatchID: dAlice, DispatchSlug: "task-q-alice-disk", BatchIndex: 1,
			Status: model.AgentInvocationStatusSuccess, TargetCount: 1, SuccessCount: 1},
		{DispatchID: dBob, DispatchSlug: "task-q-bob-ping", BatchIndex: 1,
			Status: model.AgentInvocationStatusSuccess, TargetCount: 1, SuccessCount: 1},
	}
	for _, r := range rows {
		if err := model.DB(ctx).Create(r).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	cases := []struct {
		name     string
		q        string
		wantSlug string
	}{
		{"by_slug", "alice-disk", "task-q-alice-disk"},
		{"by_command_name", "网络", "task-q-bob-ping"},
		{"by_command_content", "df", "task-q-alice-disk"},
		{"by_username_alice", "alice_ops", "task-q-alice-disk"},
		{"by_username_bob", "bob_dev", "task-q-bob-ping"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := adminSessionReq(t, "GET",
				"/admin/agent-commands/tasks?q="+c.q, nil, "alice_ops")
			rr := httptest.NewRecorder()
			HandleListAgentCommandTasks(rr, req)
			if rr.Code != 200 {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			var resp agentTaskListResp
			_ = json.Unmarshal(rr.Body.Bytes(), &resp)
			if resp.Total != 1 || len(resp.Tasks) != 1 {
				t.Fatalf("Total=%d Tasks=%d, want exactly 1; body=%s",
					resp.Total, len(resp.Tasks), rr.Body.String())
			}
			if resp.Tasks[0].DispatchSlug != c.wantSlug {
				t.Errorf("got slug=%s, want %s", resp.Tasks[0].DispatchSlug, c.wantSlug)
			}
		})
	}
}

// TestHandleListAgentCommandTasks_StatusFilter 校验 status 过滤直接走 dispatch.status。
//
// v2 数据模型：dispatch.status 是显式持久字段，可直接 WHERE 下推到 SQL，
// partial / awaiting_confirmation / cancelled 等"聚合状态"也是真值。
func TestHandleListAgentCommandTasks_StatusFilter(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner")
	cmd := &model.AgentCommand{
		Name: "n", Content: "echo", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: u.ID,
	}
	_ = model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5)
	snap := `{"name":"n","content":"echo"}`

	// 3 个 dispatch
	dSuccess := seedDispatchRow(t, ctx, "task-st-success", cmd.ID, u.ID, snap, seedDispatchOpts{
		Status: model.AgentDispatchStatusSuccess, TargetCount: 2, SuccessCount: 2,
	})
	dFailed := seedDispatchRow(t, ctx, "task-st-failed", cmd.ID, u.ID, snap, seedDispatchOpts{
		Status: model.AgentDispatchStatusFailed, TargetCount: 2, FailedCount: 2,
	})
	dPartial := seedDispatchRow(t, ctx, "task-st-partial", cmd.ID, u.ID, snap, seedDispatchOpts{
		Status: model.AgentDispatchStatusPartial, TestFirst: true,
		TargetCount: 3, SuccessCount: 2, FailedCount: 1,
	})
	rows := []*model.AgentCommandInvocation{
		{DispatchID: dSuccess, DispatchSlug: "task-st-success", BatchIndex: 1,
			Status: model.AgentInvocationStatusSuccess, TargetCount: 2, SuccessCount: 2},
		{DispatchID: dFailed, DispatchSlug: "task-st-failed", BatchIndex: 1,
			Status: model.AgentInvocationStatusFailed, TargetCount: 2, FailedCount: 2},
		{DispatchID: dPartial, DispatchSlug: "task-st-partial", IsTestRun: true, BatchIndex: 0,
			Status: model.AgentInvocationStatusSuccess, TargetCount: 1, SuccessCount: 1},
		{DispatchID: dPartial, DispatchSlug: "task-st-partial", BatchIndex: 1,
			Status: model.AgentInvocationStatusFailed, TargetCount: 2, SuccessCount: 1, FailedCount: 1},
	}
	for _, r := range rows {
		if err := model.DB(ctx).Create(r).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	cases := []struct {
		status     string
		wantTotal  int64
		wantSlugs  []string // 严格匹配集合
	}{
		{"success", 1, []string{"task-st-success"}},
		{"failed", 1, []string{"task-st-failed"}},
		{"partial", 1, []string{"task-st-partial"}},
	}
	for _, c := range cases {
		t.Run("status_"+c.status, func(t *testing.T) {
			req := adminSessionReq(t, "GET",
				"/admin/agent-commands/tasks?status="+c.status, nil, "owner")
			rr := httptest.NewRecorder()
			HandleListAgentCommandTasks(rr, req)
			if rr.Code != 200 {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			var resp agentTaskListResp
			_ = json.Unmarshal(rr.Body.Bytes(), &resp)
			if resp.Total != c.wantTotal {
				t.Errorf("Total=%d, want %d; body=%s", resp.Total, c.wantTotal, rr.Body.String())
			}
			gotSlugs := make([]string, 0, len(resp.Tasks))
			for _, t := range resp.Tasks {
				gotSlugs = append(gotSlugs, t.DispatchSlug)
			}
			if len(gotSlugs) != len(c.wantSlugs) {
				t.Fatalf("slug count=%d, want %d; got=%v", len(gotSlugs), len(c.wantSlugs), gotSlugs)
			}
			for _, want := range c.wantSlugs {
				found := false
				for _, got := range gotSlugs {
					if got == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("missing slug %s; got=%v", want, gotSlugs)
				}
			}
		})
	}
}

// ============================================================================
// HandleAgentCommandTaskDetail
// ============================================================================

func TestHandleAgentCommandTaskDetail_NotFound(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "u1")
	req := adminSessionReq(t, "GET",
		"/admin/agent-commands/tasks/detail?dispatch_slug=task-nonexistent",
		nil, "u1")
	rr := httptest.NewRecorder()
	HandleAgentCommandTaskDetail(rr, req)
	if rr.Code != 404 {
		t.Errorf("status = %d, want 404; body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAgentCommandTaskDetail_MissingSlug(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "u1")
	req := adminSessionReq(t, "GET", "/admin/agent-commands/tasks/detail", nil, "u1")
	rr := httptest.NewRecorder()
	HandleAgentCommandTaskDetail(rr, req)
	if rr.Code != 400 || !strings.Contains(rr.Body.String(), "dispatch_slug_required") {
		t.Errorf("got %d %s", rr.Code, rr.Body.String())
	}
}

func TestHandleAgentCommandTaskDetail_HappyPath(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner")
	snap := `{"name":"清理日志","content":"echo done","type":"SHELL","timeout_sec":60}`
	dID := seedDispatchRow(t, ctx, "task-detail1", 1, u.ID, snap, seedDispatchOpts{
		Status: model.AgentDispatchStatusSuccess, TargetCount: 2, SuccessCount: 2,
	})
	inv := &model.AgentCommandInvocation{
		DispatchID: dID, DispatchSlug: "task-detail1", BatchIndex: 1,
		Status:          model.AgentInvocationStatusSuccess,
		TargetCount:     2, SuccessCount: 2,
		TATInvocationID: "inv-xyz",
	}
	_ = model.DB(ctx).Create(inv).Error
	for _, tk := range []*model.AgentCommandTask{
		{
			DispatchID: dID, InvocationID: inv.ID, DispatchSlug: "task-detail1", InstanceID: 1,
			TATInvocationTaskID: "invt-1", AgentName: "Alice", CVMInstanceID: "ins-1",
			Status: model.AgentTaskStatusSuccess,
		},
		{
			DispatchID: dID, InvocationID: inv.ID, DispatchSlug: "task-detail1", InstanceID: 2,
			TATInvocationTaskID: "invt-2", AgentName: "Bob", CVMInstanceID: "ins-2",
			Status: model.AgentTaskStatusSuccess,
		},
	} {
		_ = model.DB(ctx).Create(tk).Error
	}

	mock := &mockTATBatchClient{
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{
					InvocationTaskSet: []*tat.InvocationTask{
						{
							InvocationTaskId: common.StringPtr("invt-1"),
							TaskStatus:       common.StringPtr("SUCCESS"),
							TaskResult:       &tat.TaskResult{Output: common.StringPtr("ZG9uZQ==")},
						},
						{
							InvocationTaskId: common.StringPtr("invt-2"),
							TaskStatus:       common.StringPtr("SUCCESS"),
							TaskResult:       &tat.TaskResult{Output: common.StringPtr("ZG9uZQ==")},
						},
					},
				},
			}, nil
		},
	}
	defer withMockTAT(mock)()

	req := adminSessionReq(t, "GET",
		"/admin/agent-commands/tasks/detail?dispatch_slug=task-detail1&with_output=true",
		nil, "owner")
	rr := httptest.NewRecorder()
	HandleAgentCommandTaskDetail(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var resp agentTaskDetailResp
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.DispatchSlug != "task-detail1" || len(resp.Invocations) != 1 ||
		len(resp.Invocations[0].Tasks) != 2 {
		t.Fatalf("structure wrong: %+v", resp)
	}
	for _, td := range resp.Invocations[0].Tasks {
		if td.Stdout == nil || *td.Stdout != "done" {
			t.Errorf("stdout = %v", td.Stdout)
		}
		// 正常 SUCCESS 路径下 ErrorInfo 必须为空
		if td.ErrorInfo != "" {
			t.Errorf("error_info = %q, want empty for SUCCESS task", td.ErrorInfo)
		}
	}
}

// TestHandleAgentCommandTaskDetail_StartFailedErrorInfo 回归：TAT 返回
// START_FAILED + ErrorInfo 时，detail 接口必须把 error_info 透传给前端。
//
// 复现真实场景：用户填了不存在的 run_user="dd"，TAT 在 agent 上 setuid 失败，
// 状态映射成 unreachable；前端看到 stdout/stderr 都为空，必须依赖 error_info 定位。
func TestHandleAgentCommandTaskDetail_StartFailedErrorInfo(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner")
	snap := `{"name":"bad","content":"ddd","type":"SHELL","timeout_sec":60,"run_user":"dd","workdir":"dd"}`
	dID := seedDispatchRow(t, ctx, "task-startfail", 1, u.ID, snap, seedDispatchOpts{
		Status: model.AgentDispatchStatusFailed, TargetCount: 1, FailedCount: 1,
	})
	inv := &model.AgentCommandInvocation{
		DispatchID: dID, DispatchSlug: "task-startfail", BatchIndex: 1,
		Status:          model.AgentInvocationStatusFailed,
		TargetCount:     1, FailedCount: 1,
		TATInvocationID: "inv-startfail",
	}
	_ = model.DB(ctx).Create(inv).Error
	tk := &model.AgentCommandTask{
		DispatchID: dID, InvocationID: inv.ID, DispatchSlug: "task-startfail", InstanceID: 1,
		TATInvocationTaskID: "invt-startfail", AgentName: "ag", CVMInstanceID: "ins-1",
		Status: model.AgentTaskStatusUnreachable,
	}
	_ = model.DB(ctx).Create(tk).Error

	mock := &mockTATBatchClient{
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{
					InvocationTaskSet: []*tat.InvocationTask{
						{
							InvocationTaskId: common.StringPtr("invt-startfail"),
							TaskStatus:       common.StringPtr("START_FAILED"),
							ErrorInfo:        common.StringPtr("user 'dd' does not exist"),
						},
					},
				},
			}, nil
		},
	}
	defer withMockTAT(mock)()

	req := adminSessionReq(t, "GET",
		"/admin/agent-commands/tasks/detail?dispatch_slug=task-startfail&with_output=true",
		nil, "owner")
	rr := httptest.NewRecorder()
	HandleAgentCommandTaskDetail(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var resp agentTaskDetailResp
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Invocations) != 1 || len(resp.Invocations[0].Tasks) != 1 {
		t.Fatalf("structure wrong: %+v", resp)
	}
	td := resp.Invocations[0].Tasks[0]
	if td.Status != "unreachable" {
		t.Errorf("status = %q, want unreachable", td.Status)
	}
	if td.ErrorInfo != "user 'dd' does not exist" {
		t.Errorf("error_info = %q, want passthrough from TAT", td.ErrorInfo)
	}
}

// ============================================================================
// HandleDispatchAgentCommand happy path (使用 poller 推进至终态)
// ============================================================================

func TestHandleDispatchAgentCommand_HappyPath(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner")
	cmd := &model.AgentCommand{
		Name: "echo", Content: "echo hi", Type: "SHELL", TimeoutSec: 5,
		ParamsJSON: "[]", CreatedByUserID: u.ID, RunUser: "root", Workdir: "/root",
	}
	_ = model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5)

	ins := &model.Instance{
		Name: "ins-a", InstanceId: "ins-aaa",
		LastCVMState: "RUNNING", UserID: u.ID,
	}
	_ = model.DB(ctx).Create(ins).Error

	mock := &mockTATBatchClient{
		runCommandResp: &tat.RunCommandResponse{
			Response: &tat.RunCommandResponseParams{
				InvocationId: common.StringPtr("inv-happy"),
			},
		},
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{
					InvocationTaskSet: []*tat.InvocationTask{
						{
							InvocationTaskId: common.StringPtr("invt-happy"),
							InstanceId:       common.StringPtr("ins-aaa"),
							TaskStatus:       common.StringPtr("SUCCESS"),
							TaskResult: &tat.TaskResult{
								ExitCode: common.Int64Ptr(0),
							},
							StartTime: common.StringPtr("2026-05-21 10:00:00"),
							EndTime:   common.StringPtr("2026-05-21 10:00:01"),
						},
					},
				},
			}, nil
		},
	}
	defer withMockTAT(mock)()

	req := adminSessionReq(t, "POST", "/admin/agent-commands/dispatch",
		map[string]any{
			"command_id":   cmd.ID,
			"instance_ids": []uint{ins.ID},
		}, "owner")
	rr := httptest.NewRecorder()
	HandleDispatchAgentCommand(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var resp agentDispatchResp
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if !strings.HasPrefix(resp.DispatchSlug, model.AgentCommandDispatchSlugPrefix) {
		t.Errorf("dispatch_slug = %q", resp.DispatchSlug)
	}

	// 等异步派发 goroutine 跑完。docs/testing.md 强要求异步任务不得用
	// sleep-poll 等待；这里 initAgentCommandsTestDB 已把 agentDispatchAsyncWG
	// 注入，goroutine 启动时 Add(1)、退出时 Done()，主动等待即可消除 race。
	agentDispatchAsyncWG.Wait()
	// 失败时显式诊断：清晰区分「异步 goroutine 没跑完」和「poller 没把 task 推到终态」
	{
		var preTasks []model.AgentCommandTask
		_ = model.DB(ctx).Where("dispatch_slug = ?", resp.DispatchSlug).Find(&preTasks).Error
		if len(preTasks) != 1 || preTasks[0].TATInvocationTaskID == "" {
			t.Fatalf("dispatch async finished but task TAT id missing; tasks=%+v", preTasks)
		}
	}
	// 模拟 poller 单次 tick → 把 task 推进到终态
	RunAgentCommandPollerOnce(ctx)

	var tasks []model.AgentCommandTask
	_ = model.DB(ctx).Where("dispatch_slug = ?", resp.DispatchSlug).Find(&tasks).Error
	if len(tasks) != 1 {
		t.Fatalf("tasks count = %d, want 1", len(tasks))
	}
	if tasks[0].Status != model.AgentTaskStatusSuccess {
		t.Errorf("task final status = %s, want success", tasks[0].Status)
	}
}

// ============================================================================
// 工具函数补充覆盖
// ============================================================================

func TestIsInstanceRunning(t *testing.T) {
	cases := []struct {
		name string
		ins  *model.Instance
		want bool
	}{
		{"nil", nil, false},
		{"running cvm state", &model.Instance{LastCVMState: "RUNNING"}, true},
		{"stopped cvm state", &model.Instance{LastCVMState: "STOPPED"}, false},
		{"running stable state", &model.Instance{LastStableState: "running"}, true},
		{"unknown", &model.Instance{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isInstanceRunning(c.ins); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestTrimRunes(t *testing.T) {
	if trimRunes("hi", 10) != "hi" {
		t.Error("short should pass")
	}
	if got := trimRunes("hello world", 5); got != "hello" {
		t.Errorf("got %q, want hello", got)
	}
}

func TestIndexInstancesByID(t *testing.T) {
	ins := []model.Instance{{Name: "a"}, {Name: "b"}}
	ins[0].ID = 1
	ins[1].ID = 2
	got := indexInstancesByID(ins)
	if got[1].Name != "a" || got[2].Name != "b" {
		t.Errorf("index wrong: %v", got)
	}
}

func TestGenerateUniqueDispatchSlug(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	s, err := generateUniqueDispatchSlug(ctx, 5)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.HasPrefix(s, model.AgentCommandDispatchSlugPrefix) {
		t.Errorf("prefix wrong: %q", s)
	}
}

func TestMapTATToAgentStatusFromDetail(t *testing.T) {
	if got := mapTATToAgentStatusFromDetail(InvocationTaskDetail{}); got != model.AgentTaskStatusInProgress {
		t.Errorf("empty status = %s, want in_progress", got)
	}
	if got := mapTATToAgentStatusFromDetail(InvocationTaskDetail{TaskStatus: "SUCCESS"}); got != model.AgentTaskStatusSuccess {
		t.Errorf("SUCCESS mapping wrong: %s", got)
	}
}

func TestRunUserOrDefault_WorkdirOrDefault(t *testing.T) {
	if runUserOrDefault("") != "root" {
		t.Error("empty -> root")
	}
	if runUserOrDefault("alice") != "alice" {
		t.Error("non-empty pass through")
	}
	if workdirOrDefault("") != "/root" {
		t.Error("empty -> /root")
	}
	if workdirOrDefault("/tmp") != "/tmp" {
		t.Error("non-empty pass through")
	}
}
