package controller

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tat "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat/v20201028"
)

// ============================================================================
// 测试用 TAT 客户端 mock
// ============================================================================

type mockTATBatchClient struct {
	runCommandReq     *tat.RunCommandRequest
	runCommandResp    *tat.RunCommandResponse
	runCommandErr     error
	describeTasksReqs []*tat.DescribeInvocationTasksRequest
	describeTasksFn   func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error)
}

func (m *mockTATBatchClient) RunCommand(req *tat.RunCommandRequest) (*tat.RunCommandResponse, error) {
	m.runCommandReq = req
	if m.runCommandErr != nil {
		return nil, m.runCommandErr
	}
	return m.runCommandResp, nil
}

func (m *mockTATBatchClient) DescribeInvocationTasks(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
	m.describeTasksReqs = append(m.describeTasksReqs, req)
	if m.describeTasksFn != nil {
		return m.describeTasksFn(req)
	}
	return &tat.DescribeInvocationTasksResponse{
		Response: &tat.DescribeInvocationTasksResponseParams{},
	}, nil
}

func withMockTAT(m *mockTATBatchClient) func() {
	prev := tatBatchClientFactory
	tatBatchClientFactory = func(ctx context.Context) (tatBatchClient, error) {
		return m, nil
	}
	return func() { tatBatchClientFactory = prev }
}

// 把测试时长加速
func withFastBatchTimings() func() {
	// 不改时间常量，运行时少调用即可；保留占位以便未来调速
	return func() {}
}

// ============================================================================
// RunInlineCommandBatchAsync
// ============================================================================

func TestRunInlineCommandBatchAsync_RawContentEncodedAndNoPreludeInjected(t *testing.T) {
	rawContent := "#!/bin/bash\necho 'hello {{name}}'\n"
	mock := &mockTATBatchClient{
		runCommandResp: &tat.RunCommandResponse{
			Response: &tat.RunCommandResponseParams{
				InvocationId: common.StringPtr("inv-test1234"),
			},
		},
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{
					InvocationTaskSet: []*tat.InvocationTask{
						{
							InvocationTaskId: common.StringPtr("invt-task-a"),
							InstanceId:       common.StringPtr("ins-aaa"),
						},
						{
							InvocationTaskId: common.StringPtr("invt-task-b"),
							InstanceId:       common.StringPtr("ins-bbb"),
						},
					},
				},
			}, nil
		},
	}
	defer withMockTAT(mock)()

	invID, bindings, rerr := RunInlineCommandBatchAsync(
		context.Background(),
		[]string{"ins-aaa", "ins-bbb"},
		rawContent,
		60,
		"root",
		"/root",
		map[string]string{"name": "alice"},
	)
	if rerr != nil {
		t.Fatalf("RunInlineCommandBatchAsync: %v", rerr)
	}
	if invID != "inv-test1234" {
		t.Errorf("invocationId = %q, want inv-test1234", invID)
	}
	if len(bindings) != 2 {
		t.Fatalf("bindings count = %d, want 2", len(bindings))
	}

	// ⚠️ 关键断言（spec.md）：TAT 收到的 Content 字段 base64 解码后逐字等于原文
	if mock.runCommandReq == nil || mock.runCommandReq.Content == nil {
		t.Fatal("RunCommand request not captured")
	}
	gotEncoded := *mock.runCommandReq.Content
	decoded, err := base64.StdEncoding.DecodeString(gotEncoded)
	if err != nil {
		t.Fatalf("decode TAT content: %v", err)
	}
	if string(decoded) != rawContent {
		t.Errorf("TAT Content decoded = %q, want %q (no prelude must be injected)",
			string(decoded), rawContent)
	}
	// 兜底再次确认：不含 tatRuntimePrelude 任何标志性片段
	if strings.Contains(string(decoded), "OPENCLAW_RUNTIME_USER") {
		t.Error("tatRuntimePrelude was injected into TAT Content; spec.md §8.2 violated")
	}

	// Parameters 字段应为 JSON
	if mock.runCommandReq.Parameters == nil ||
		!strings.Contains(*mock.runCommandReq.Parameters, `"name"`) ||
		!strings.Contains(*mock.runCommandReq.Parameters, `"alice"`) {
		t.Errorf("Parameters JSON = %v, want contain name=alice", mock.runCommandReq.Parameters)
	}

	// SaveCommand 必须 false（不污染腾讯云命令库）
	if mock.runCommandReq.SaveCommand == nil || *mock.runCommandReq.SaveCommand {
		t.Errorf("SaveCommand = %v, want false", mock.runCommandReq.SaveCommand)
	}
}

func TestRunInlineCommandBatchAsync_OverLimit(t *testing.T) {
	mock := &mockTATBatchClient{}
	defer withMockTAT(mock)()

	ids := make([]string, TATRunCommandBatchMax+1)
	for i := range ids {
		ids[i] = "ins-x"
	}
	_, _, err := RunInlineCommandBatchAsync(context.Background(), ids,
		"#!/bin/bash\necho ok", 60, "root", "/root", nil)
	if err == nil || !errors.Is(err, ErrTATBatchTooMany) {
		t.Errorf("expected ErrTATBatchTooMany, got %v", err)
	}
	if mock.runCommandReq != nil {
		t.Error("RunCommand should NOT be called when over limit")
	}
}

func TestRunInlineCommandBatchAsync_EmptyInstanceIds(t *testing.T) {
	defer withMockTAT(&mockTATBatchClient{})()
	if _, _, err := RunInlineCommandBatchAsync(context.Background(), nil,
		"echo", 60, "root", "/root", nil); err == nil {
		t.Error("expected error for empty instanceIds")
	}
}

func TestRunInlineCommandBatchAsync_TATError(t *testing.T) {
	mock := &mockTATBatchClient{
		runCommandErr: errors.New("TAT internal error"),
	}
	defer withMockTAT(mock)()
	_, _, err := RunInlineCommandBatchAsync(context.Background(),
		[]string{"ins-a"}, "echo", 60, "root", "/root", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunInlineCommandBatchAsync_NoInvocationID(t *testing.T) {
	mock := &mockTATBatchClient{
		runCommandResp: &tat.RunCommandResponse{
			Response: &tat.RunCommandResponseParams{InvocationId: nil},
		},
	}
	defer withMockTAT(mock)()
	if _, _, err := RunInlineCommandBatchAsync(context.Background(),
		[]string{"ins-a"}, "echo", 60, "root", "/root", nil); err == nil {
		t.Error("expected error when TAT returns no invocation id")
	}
}

// TestFetchInvocationTaskBindings_Paginated 验证 dispatch 上限 200、TAT
// DescribeInvocationTasks 单页 100 上限的场景：函数必须按 Offset 分两页拉，
// 累计 200 个 binding 才算成功；不能因第一页拿到 100 就 return 部分结果。
func TestFetchInvocationTaskBindings_Paginated(t *testing.T) {
	const total = TATRunCommandBatchMax // 200
	calls := 0
	mock := &mockTATBatchClient{
		runCommandResp: &tat.RunCommandResponse{
			Response: &tat.RunCommandResponseParams{
				InvocationId: common.StringPtr("inv-paged"),
			},
		},
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			calls++
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
				idStr := "invt-" + strings.Repeat("0", 4-len(localItoa(i))) + localItoa(i)
				insStr := "ins-" + strings.Repeat("0", 4-len(localItoa(i))) + localItoa(i)
				set = append(set, &tat.InvocationTask{
					InvocationTaskId: common.StringPtr(idStr),
					InstanceId:       common.StringPtr(insStr),
				})
			}
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{
					InvocationTaskSet: set,
				},
			}, nil
		},
	}
	defer withMockTAT(mock)()

	ids := make([]string, total)
	for i := range ids {
		ids[i] = "ins-x"
	}
	invID, bindings, err := RunInlineCommandBatchAsync(context.Background(),
		ids, "echo", 60, "root", "/root", nil)
	if err != nil {
		t.Fatalf("RunInlineCommandBatchAsync: %v", err)
	}
	if invID != "inv-paged" {
		t.Errorf("invID = %q", invID)
	}
	if len(bindings) != total {
		t.Errorf("bindings = %d, want %d", len(bindings), total)
	}
	if calls != 2 {
		t.Errorf("DescribeInvocationTasks calls = %d, want 2 (200/100 split)", calls)
	}
}

// TestFetchInvocationTaskBindings_FirstAttemptPartialThenRetryFills 验证
// 首轮拉到的 binding 数量不足 expected 时，函数 SHALL 进入下一轮重试，
// 模拟 TAT 端 InvocationTask 行延迟可见的场景。
func TestFetchInvocationTaskBindings_FirstAttemptPartialThenRetryFills(t *testing.T) {
	const expected = 50
	round := 0
	mock := &mockTATBatchClient{
		runCommandResp: &tat.RunCommandResponse{
			Response: &tat.RunCommandResponseParams{
				InvocationId: common.StringPtr("inv-late"),
			},
		},
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			// round=0: 首轮第 1 页只返回 10 条（不足 expected=50），第 2 页空 → 进入下一轮重试
			// round>=1: 满 expected
			round++
			if round == 1 {
				set := make([]*tat.InvocationTask, 0, 10)
				for i := 0; i < 10; i++ {
					set = append(set, &tat.InvocationTask{
						InvocationTaskId: common.StringPtr("invt-r1-" + localItoa(i)),
						InstanceId:       common.StringPtr("ins-r1-" + localItoa(i)),
					})
				}
				return &tat.DescribeInvocationTasksResponse{
					Response: &tat.DescribeInvocationTasksResponseParams{InvocationTaskSet: set},
				}, nil
			}
			set := make([]*tat.InvocationTask, 0, expected)
			for i := 0; i < expected; i++ {
				set = append(set, &tat.InvocationTask{
					InvocationTaskId: common.StringPtr("invt-final-" + localItoa(i)),
					InstanceId:       common.StringPtr("ins-final-" + localItoa(i)),
				})
			}
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{InvocationTaskSet: set},
			}, nil
		},
	}
	defer withMockTAT(mock)()

	ids := make([]string, expected)
	for i := range ids {
		ids[i] = "ins-y"
	}
	_, bindings, err := RunInlineCommandBatchAsync(context.Background(),
		ids, "echo", 60, "root", "/root", nil)
	if err != nil {
		t.Fatalf("RunInlineCommandBatchAsync: %v", err)
	}
	if len(bindings) != expected {
		t.Errorf("bindings = %d, want %d (after retry)", len(bindings), expected)
	}
	if round < 2 {
		t.Errorf("expected at least 2 rounds (partial → retry), got %d", round)
	}
}

// ============================================================================
// DescribeInvocationTasksBatch
// ============================================================================

func TestDescribeInvocationTasksBatch_SingleBatch(t *testing.T) {
	encodedOut := base64.StdEncoding.EncodeToString([]byte("hello world\n"))
	mock := &mockTATBatchClient{
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{
					InvocationTaskSet: []*tat.InvocationTask{
						{
							InvocationTaskId: common.StringPtr("invt-1"),
							InstanceId:       common.StringPtr("ins-1"),
							TaskStatus:       common.StringPtr("SUCCESS"),
							TaskResult: &tat.TaskResult{
								Output:   common.StringPtr(encodedOut),
								ExitCode: common.Int64Ptr(0),
							},
							StartTime: common.StringPtr("2026-05-21 10:00:00"),
							EndTime:   common.StringPtr("2026-05-21 10:00:02"),
						},
					},
				},
			}, nil
		},
	}
	defer withMockTAT(mock)()

	got, err := DescribeInvocationTasksBatch(context.Background(),
		[]string{"invt-1", "invt-1" /* 测试去重 */, ""})
	if err != nil {
		t.Fatalf("DescribeInvocationTasksBatch: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	d := got["invt-1"]
	if d.TaskStatus != "SUCCESS" || !d.Finished {
		t.Errorf("status/finished mismatch: %+v", d)
	}
	if d.Stdout != "hello world\n" {
		t.Errorf("stdout decoded = %q, want %q", d.Stdout, "hello world\n")
	}
	if d.ExitCode == nil || *d.ExitCode != 0 {
		t.Errorf("exit_code = %v", d.ExitCode)
	}
	if len(mock.describeTasksReqs) != 1 {
		t.Errorf("DescribeInvocationTasks calls = %d, want 1", len(mock.describeTasksReqs))
	}
}

// TestDescribeInvocationTasksBatch_ErrorInfoPassthrough 回归：TAT 顶层 ErrorInfo
// 字段（DELIVER_FAILED / START_FAILED 时承载启动失败原因）必须透传到 detail，
// 否则前端看到 unreachable + stdout/stderr 都为空，根本不知道具体原因。
func TestDescribeInvocationTasksBatch_ErrorInfoPassthrough(t *testing.T) {
	mock := &mockTATBatchClient{
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{
					InvocationTaskSet: []*tat.InvocationTask{
						{
							InvocationTaskId: common.StringPtr("invt-fail"),
							InstanceId:       common.StringPtr("ins-fail"),
							TaskStatus:       common.StringPtr("START_FAILED"),
							ErrorInfo:        common.StringPtr("user 'dd' does not exist on the instance"),
						},
					},
				},
			}, nil
		},
	}
	defer withMockTAT(mock)()

	got, err := DescribeInvocationTasksBatch(context.Background(), []string{"invt-fail"})
	if err != nil {
		t.Fatalf("DescribeInvocationTasksBatch: %v", err)
	}
	d, ok := got["invt-fail"]
	if !ok {
		t.Fatalf("invt-fail not in result")
	}
	if d.TaskStatus != "START_FAILED" {
		t.Errorf("TaskStatus = %q, want START_FAILED", d.TaskStatus)
	}
	if d.ErrorInfo != "user 'dd' does not exist on the instance" {
		t.Errorf("ErrorInfo = %q, want passthrough from TAT", d.ErrorInfo)
	}
	// 同时确认到达 TAT 终态的判定
	if !d.Finished {
		t.Error("START_FAILED should be terminal")
	}
}

func TestDescribeInvocationTasksBatch_AutoSplit(t *testing.T) {
	// 构造超过单批上限的 ID 集
	total := TATDescribeInvocationTasksBatchMax + 5
	ids := make([]string, total)
	for i := range ids {
		ids[i] = "invt-bulk-" + strings.Repeat("0", 4-len(localItoa(i))) + localItoa(i)
	}
	calls := 0
	mock := &mockTATBatchClient{
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			calls++
			out := &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{
					InvocationTaskSet: []*tat.InvocationTask{},
				},
			}
			for _, idPtr := range req.InvocationTaskIds {
				out.Response.InvocationTaskSet = append(out.Response.InvocationTaskSet,
					&tat.InvocationTask{
						InvocationTaskId: idPtr,
						InstanceId:       common.StringPtr("ins-x"),
						TaskStatus:       common.StringPtr("RUNNING"),
					})
			}
			return out, nil
		},
	}
	defer withMockTAT(mock)()

	got, err := DescribeInvocationTasksBatch(context.Background(), ids)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != total {
		t.Errorf("got %d entries, want %d", len(got), total)
	}
	if calls != 2 {
		t.Errorf("auto-split calls = %d, want 2 (total=%d)", calls, total)
	}
}

func TestDescribeInvocationTasksBatch_MissingTaskNotInResult(t *testing.T) {
	mock := &mockTATBatchClient{
		describeTasksFn: func(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error) {
			// TAT 端只返回其中一条
			return &tat.DescribeInvocationTasksResponse{
				Response: &tat.DescribeInvocationTasksResponseParams{
					InvocationTaskSet: []*tat.InvocationTask{
						{
							InvocationTaskId: common.StringPtr("invt-found"),
							TaskStatus:       common.StringPtr("RUNNING"),
						},
					},
				},
			}, nil
		},
	}
	defer withMockTAT(mock)()

	got, err := DescribeInvocationTasksBatch(context.Background(),
		[]string{"invt-found", "invt-expired"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, ok := got["invt-found"]; !ok {
		t.Error("expected invt-found in result")
	}
	if _, ok := got["invt-expired"]; ok {
		t.Error("expired task should NOT be in result map (caller marks output_expired=true)")
	}
}

func TestDescribeInvocationTasksBatch_EmptyInput(t *testing.T) {
	defer withMockTAT(&mockTATBatchClient{})()
	got, err := DescribeInvocationTasksBatch(context.Background(), nil)
	if err != nil {
		t.Errorf("nil input err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("nil input expected empty map, got %v", got)
	}
}

func TestIsTATTerminalStatus(t *testing.T) {
	cases := map[string]bool{
		"SUCCESS":        true,
		"FAILED":         true,
		"TIMEOUT":        true,
		"DELIVER_FAILED": true,
		"START_FAILED":   true,
		"CANCELED":       true,
		"TERMINATED":     true,
		"PENDING":        false,
		"RUNNING":        false,
		"DELIVERING":     false,
		"":               false,
	}
	for s, want := range cases {
		if got := isTATTerminalStatus(s); got != want {
			t.Errorf("%q = %v, want %v", s, got, want)
		}
	}
}

func TestMapTATTaskStatusToAgentTaskStatus(t *testing.T) {
	cases := map[string]string{
		"SUCCESS":        "success",
		"FAILED":         "failed",
		"TIMEOUT":        "timeout",
		"DELIVER_FAILED": "unreachable",
		"START_FAILED":   "unreachable",
		"TERMINATED":     "unreachable",
		"CANCELED":       "failed",
		"PENDING":        "pending",
		"RUNNING":        "in_progress",
		"DELIVERING":     "in_progress",
		"UNKNOWN_STATE":  "in_progress",
	}
	for tat, want := range cases {
		if got := MapTATTaskStatusToAgentTaskStatus(tat); got != want {
			t.Errorf("%q -> %q, want %q", tat, got, want)
		}
	}
}

func TestParseTATTime(t *testing.T) {
	if _, err := parseTATTime("2026-05-21 10:00:00"); err != nil {
		t.Errorf("standard format failed: %v", err)
	}
	if _, err := parseTATTime("2026-05-21T10:00:00Z"); err != nil {
		t.Errorf("rfc3339 failed: %v", err)
	}
	if _, err := parseTATTime("garbage"); err == nil {
		t.Error("garbage should fail")
	}
}

// localItoa 简易整数转字符串，避免依赖 strconv 增加 import 噪音
func localItoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	buf := make([]byte, 0, 12)
	for i > 0 {
		buf = append([]byte{byte('0' + i%10)}, buf...)
		i /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
