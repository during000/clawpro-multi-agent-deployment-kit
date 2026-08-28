package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tat "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat/v20201028"

	"hatchery/model"
)

// TestHandleDispatchAgentCommand_Max200_NonTestFirst 验证 dispatch 上限提升到 200
// 后，单次选满 200 台 Agent（无测试机）能成功下发：1 行 prod invocation + 200
// 行 task；异步 binding 通过分页拉取。
func TestHandleDispatchAgentCommand_Max200_NonTestFirst(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner200")
	cmd := &model.AgentCommand{
		Name: "echo200", Content: "echo hi", Type: "SHELL", TimeoutSec: 5,
		ParamsJSON: "[]", CreatedByUserID: u.ID, RunUser: "root", Workdir: "/root",
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5); err != nil {
		t.Fatalf("create command: %v", err)
	}

	const total = 200
	insIDs := make([]uint, 0, total)
	cvmIDs := make([]string, 0, total)
	for i := 0; i < total; i++ {
		cvmID := fmt.Sprintf("ins-%04d", i)
		ins := &model.Instance{
			Name: cvmID, InstanceId: cvmID,
			LastCVMState: "RUNNING", UserID: u.ID,
		}
		if err := model.DB(ctx).Create(ins).Error; err != nil {
			t.Fatalf("seed instance %d: %v", i, err)
		}
		insIDs = append(insIDs, ins.ID)
		cvmIDs = append(cvmIDs, cvmID)
	}

	mock := &mockTATBatchClient{
		runCommandResp: &tat.RunCommandResponse{
			Response: &tat.RunCommandResponseParams{
				InvocationId: common.StringPtr("inv-200-prod"),
			},
		},
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			offset := uint64(0)
			if req.Offset != nil {
				offset = *req.Offset
			}
			limit := uint64(TATDescribeInvocationTasksBatchMax)
			if req.Limit != nil {
				limit = *req.Limit
			}
			start := int(offset)
			end := start + int(limit)
			if start >= total {
				return &tat.DescribeInvocationTasksResponse{
					Response: &tat.DescribeInvocationTasksResponseParams{},
				}, nil
			}
			if end > total {
				end = total
			}
			set := make([]*tat.InvocationTask, 0, end-start)
			for i := start; i < end; i++ {
				set = append(set, &tat.InvocationTask{
					InvocationTaskId: common.StringPtr(fmt.Sprintf("invt-%04d", i)),
					InstanceId:       common.StringPtr(cvmIDs[i]),
				})
			}
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{InvocationTaskSet: set},
			}, nil
		},
	}
	defer withMockTAT(mock)()

	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/dispatch",
		map[string]any{
			"command_id":   cmd.ID,
			"instance_ids": insIDs,
		}, "owner200")
	rr := httptest.NewRecorder()
	HandleDispatchAgentCommand(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var resp agentDispatchResp
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.HasPrefix(resp.DispatchSlug, model.AgentCommandDispatchSlugPrefix) {
		t.Errorf("dispatch_slug = %q", resp.DispatchSlug)
	}

	agentDispatchAsyncWG.Wait()

	var invs []model.AgentCommandInvocation
	_ = model.DB(ctx).Where("dispatch_slug = ?", resp.DispatchSlug).Find(&invs).Error
	if len(invs) != 1 {
		t.Fatalf("invocation count = %d, want 1 (single prod batch with 200 targets)", len(invs))
	}
	if invs[0].IsTestRun {
		t.Errorf("invocation should not be test run")
	}
	if invs[0].TATInvocationID == "" {
		t.Errorf("invocation TAT id missing")
	}

	var tasks []model.AgentCommandTask
	_ = model.DB(ctx).Where("dispatch_slug = ?", resp.DispatchSlug).Find(&tasks).Error
	if len(tasks) != total {
		t.Fatalf("task count = %d, want %d", len(tasks), total)
	}
	missingBinding := 0
	for _, tk := range tasks {
		if tk.TATInvocationTaskID == "" {
			missingBinding++
		}
	}
	if missingBinding > 0 {
		t.Errorf("%d/%d tasks missing TAT binding (paginated fetch incomplete)",
			missingBinding, total)
	}
}

// TestHandleDispatchAgentCommand_Max200_TestFirst 验证 200 台 + 测试机模式：
// 1 行测试 invocation + 1 行生产 invocation（199 台）+ 200 行 task。
// 仅校验同步预创建产物，不走异步派发（异步路径已由 _NonTestFirst 覆盖）。
func TestHandleDispatchAgentCommand_Max200_TestFirst(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner200tf")
	cmd := &model.AgentCommand{
		Name: "echo200tf", Content: "echo hi", Type: "SHELL", TimeoutSec: 5,
		ParamsJSON: "[]", CreatedByUserID: u.ID, RunUser: "root", Workdir: "/root",
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5); err != nil {
		t.Fatalf("create command: %v", err)
	}

	const total = 200
	insIDs := make([]uint, 0, total)
	for i := 0; i < total; i++ {
		ins := &model.Instance{
			Name:         fmt.Sprintf("tf-ins-%04d", i),
			InstanceId:   fmt.Sprintf("tf-ins-%04d", i),
			LastCVMState: "RUNNING",
			UserID:       u.ID,
		}
		if err := model.DB(ctx).Create(ins).Error; err != nil {
			t.Fatalf("seed instance %d: %v", i, err)
		}
		insIDs = append(insIDs, ins.ID)
	}

	// 测试机阶段会异步发 RunCommand 后等终态：mock 直接返回 SUCCESS，让异步路径秒结束。
	mock := &mockTATBatchClient{
		runCommandResp: &tat.RunCommandResponse{
			Response: &tat.RunCommandResponseParams{
				InvocationId: common.StringPtr("inv-200-test"),
			},
		},
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{
					InvocationTaskSet: []*tat.InvocationTask{
						{
							InvocationTaskId: common.StringPtr("invt-tf-test"),
							InstanceId:       common.StringPtr("tf-ins-0000"),
							TaskStatus:       common.StringPtr("SUCCESS"),
							TaskResult: &tat.TaskResult{
								ExitCode: common.Int64Ptr(0),
							},
							StartTime: common.StringPtr("2026-05-29 10:00:00"),
							EndTime:   common.StringPtr("2026-05-29 10:00:01"),
						},
					},
				},
			}, nil
		},
	}
	defer withMockTAT(mock)()

	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/dispatch",
		map[string]any{
			"command_id":              cmd.ID,
			"instance_ids":            insIDs,
			"test_first":              true,
			"test_target_instance_id": insIDs[0],
		}, "owner200tf")
	rr := httptest.NewRecorder()
	HandleDispatchAgentCommand(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var resp agentDispatchResp
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// 不等异步：直接看预创建结果（事务结束即可见）
	var invs []model.AgentCommandInvocation
	_ = model.DB(ctx).Where("dispatch_slug = ?", resp.DispatchSlug).
		Order("batch_index ASC").Find(&invs).Error
	if len(invs) != 2 {
		t.Fatalf("invocation count = %d, want 2 (1 test + 1 prod)", len(invs))
	}
	if !invs[0].IsTestRun || invs[0].TargetCount != 1 {
		t.Errorf("test invocation mismatch: is_test=%v target=%d",
			invs[0].IsTestRun, invs[0].TargetCount)
	}
	if invs[1].IsTestRun || invs[1].TargetCount != total-1 {
		t.Errorf("prod invocation mismatch: is_test=%v target=%d (want %d)",
			invs[1].IsTestRun, invs[1].TargetCount, total-1)
	}

	var tasks []model.AgentCommandTask
	_ = model.DB(ctx).Where("dispatch_slug = ?", resp.DispatchSlug).Find(&tasks).Error
	if len(tasks) != total {
		t.Errorf("task count = %d, want %d (1 test + 199 prod)", len(tasks), total)
	}

	agentDispatchAsyncWG.Wait()
}

// TestHandleDispatchAgentCommand_201Rejected 验证常量调到 200 后，201 台 dispatch
// 仍被拒绝。等价于既有 too_many_targets 用例，但显式断言文案中包含 200。
func TestHandleDispatchAgentCommand_201Rejected(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner201")
	cmd := &model.AgentCommand{
		Name: "echo201", Content: "echo hi", Type: "SHELL", TimeoutSec: 5,
		ParamsJSON: "[]", CreatedByUserID: u.ID, RunUser: "root", Workdir: "/root",
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5); err != nil {
		t.Fatalf("create command: %v", err)
	}

	ids := make([]uint, model.AgentDispatchMaxTargets+1)
	for i := range ids {
		ids[i] = uint(i + 1)
	}
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/dispatch",
		map[string]any{"command_id": cmd.ID, "instance_ids": ids}, "owner201")
	rr := httptest.NewRecorder()
	HandleDispatchAgentCommand(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "too_many_targets") {
		t.Errorf("body missing too_many_targets: %s", body)
	}
	if !strings.Contains(body, "200") {
		t.Errorf("body should mention new limit 200: %s", body)
	}
}
