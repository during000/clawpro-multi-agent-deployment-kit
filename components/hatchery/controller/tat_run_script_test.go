package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	hcommon "hatchery/common"

	tat "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat/v20201028"
)

// ────────────────────────────────────────────────────────────────────
// RunScript 完整链路单测
//
// 关键依赖与 mock 策略：
//   ① LoadScript           → 包级 var，注入一个返回简单脚本的 mock
//   ② NewTATClient         → 包级 var，返回指向 httptest.Server 的真实 SDK client
//   ③ httptest.Server      → 按 X-TC-Action 路由 3 个 TAT API：
//                            - DescribeAutomationAgentStatus  (checkAgentOnline)
//                            - RunCommand                     (下发命令)
//                            - DescribeInvocationTasks        (轮询任务结果)
//   ④ runScriptPollInterval → 包级 var，缩短到 1ms，避免每轮真等 2s
//
// 这样可以稳定、快速地覆盖 RunScript 的全部 status switch 分支。
// ────────────────────────────────────────────────────────────────────

// runScriptMockServer 启动一个按 X-TC-Action 分发响应的 TAT mock server。
// invocationStates 描述 DescribeInvocationTasks 每次调用应返回的响应：
//   - 第 i 次调用返回 invocationStates[min(i, len-1)]
// agentStatus 控制 DescribeAutomationAgentStatus 返回值（默认 "Online"）。
type runScriptMockServer struct {
	server                *httptest.Server
	describeCallCount     int32
	invocationStates      []*tat.InvocationTask
	agentStatus           string
	runCommandInvocation  string
	runCommandShouldError bool
}

func newRunScriptMockServer(t *testing.T) *runScriptMockServer {
	t.Helper()
	m := &runScriptMockServer{
		agentStatus:          "Online",
		runCommandInvocation: "inv-mock-12345",
	}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.Header.Get("X-TC-Action")
		w.Header().Set("Content-Type", "application/json")
		switch action {
		case "DescribeAutomationAgentStatus":
			fmt.Fprint(w, m.makeAgentStatusResp())
		case "RunCommand":
			if m.runCommandShouldError {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			fmt.Fprint(w, m.makeRunCommandResp())
		case "DescribeInvocationTasks":
			n := int(atomic.AddInt32(&m.describeCallCount, 1)) - 1
			task := m.invocationStates[len(m.invocationStates)-1]
			if n < len(m.invocationStates) {
				task = m.invocationStates[n]
			}
			fmt.Fprint(w, m.makeDescribeTasksResp(task))
		default:
			http.Error(w, "unknown action: "+action, http.StatusBadRequest)
		}
	}))
	t.Cleanup(func() { m.server.Close() })
	return m
}

func (m *runScriptMockServer) makeAgentStatusResp() string {
	resp := tat.DescribeAutomationAgentStatusResponse{
		Response: &tat.DescribeAutomationAgentStatusResponseParams{
			AutomationAgentSet: []*tat.AutomationAgentInfo{{
				AgentStatus: sptr(m.agentStatus),
			}},
			RequestId: sptr("mock-req-agent"),
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func (m *runScriptMockServer) makeRunCommandResp() string {
	resp := tat.RunCommandResponse{
		Response: &tat.RunCommandResponseParams{
			InvocationId: sptr(m.runCommandInvocation),
			RequestId:    sptr("mock-req-run"),
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func (m *runScriptMockServer) makeDescribeTasksResp(task *tat.InvocationTask) string {
	resp := tat.DescribeInvocationTasksResponse{
		Response: &tat.DescribeInvocationTasksResponseParams{
			TotalCount:        uptr(1),
			InvocationTaskSet: []*tat.InvocationTask{task},
			RequestId:         sptr("mock-req-desc"),
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

// b64 是用于构造 task.TaskResult.Output 的辅助。
func b64(s string) *string {
	encoded := base64.StdEncoding.EncodeToString([]byte(s))
	return &encoded
}

// finishedTask 构造一个已结束的 task。
func finishedTask(status, output, errorInfo string) *tat.InvocationTask {
	t := &tat.InvocationTask{
		TaskStatus: sptr(status),
		EndTime:    sptr("2026-06-24T10:00:00Z"),
	}
	if output != "" {
		t.TaskResult = &tat.TaskResult{Output: b64(output)}
	}
	if errorInfo != "" {
		t.ErrorInfo = sptr(errorInfo)
	}
	return t
}

// runningTask 构造一个未结束的 task。
func runningTask() *tat.InvocationTask {
	return &tat.InvocationTask{TaskStatus: sptr("RUNNING")}
}

// setupRunScriptEnv 注入所有 mock：LoadScript、NewTATClient、缩短 poll 间隔。
func setupRunScriptEnv(t *testing.T, mockServerURL string) {
	t.Helper()

	// 1. mock LoadScript
	origLoadScript := LoadScript
	LoadScript = func(name string) (string, error) {
		return "#!/bin/bash\necho hello\n", nil
	}
	t.Cleanup(func() { LoadScript = origLoadScript })

	// 2. mock NewTATClient 指向 mock server
	installMockTATClient(t, mockServerURL)

	// 3. 缩短 RunScript 主循环的 poll 间隔到 1ms
	origInterval := runScriptPollInterval
	runScriptPollInterval = 1 * time.Millisecond
	t.Cleanup(func() { runScriptPollInterval = origInterval })

	// 4. 缩短 deadline buffer 到 ms 级，使 WaitResultTimeout 路径可以快速触发
	origBuffer := runScriptDeadlineBuffer
	runScriptDeadlineBuffer = 50 * time.Millisecond
	t.Cleanup(func() { runScriptDeadlineBuffer = origBuffer })

	// 5. 缩短 followup 节拍（RunScript WaitResultTimeout 路径会启动后台 goroutine）
	installFastFollowupTimings(t, 1*time.Millisecond, 50*time.Millisecond)
}

// detailStr 从 error（通常是 *RichError）中提取 detail。
func detailStr(err error) string {
	if err == nil {
		return ""
	}
	return hcommon.ErrorDetailWithCtx(context.Background(), err)
}

// ─── 用例 1：SUCCESS 路径，返回完整 output ───────────────────────────

func TestRunScript_Success(t *testing.T) {
	m := newRunScriptMockServer(t)
	m.invocationStates = []*tat.InvocationTask{
		runningTask(),
		finishedTask("SUCCESS", "  result-payload\n", ""),
	}
	setupRunScriptEnv(t, m.server.URL)

	out, rerr := RunScript(context.Background(), "ins-test", "fake.sh", 5, "root", nil, nil)
	if rerr != nil {
		t.Fatalf("不应返回错误: %v, detail=%s", rerr, detailStr(rerr))
	}
	if out != "result-payload" {
		t.Errorf("output 应被 trim, got %q", out)
	}
}

// ─── 用例 2：FAILED 路径，stdout 进入 detail ────────────────────────

func TestRunScript_Failed_DetailFromStdout(t *testing.T) {
	m := newRunScriptMockServer(t)
	m.invocationStates = []*tat.InvocationTask{
		finishedTask("FAILED", "  script crashed: exit 1\n", "ignored-tat-info"),
	}
	setupRunScriptEnv(t, m.server.URL)

	out, rerr := RunScript(context.Background(), "ins-test", "fake.sh", 5, "root", nil, nil)
	if rerr == nil {
		t.Fatal("FAILED 应返回错误")
	}
	if out != "" {
		t.Errorf("出错时 output 应为空, got %q", out)
	}
	if detail := detailStr(rerr); detail != "script crashed: exit 1" {
		t.Errorf("FAILED detail 应为 stdout, got %q", detail)
	}
}

// ─── 用例 3：TIMEOUT 路径 ─────────────────────────────────────────

func TestRunScript_Timeout_DetailFromStdout(t *testing.T) {
	m := newRunScriptMockServer(t)
	m.invocationStates = []*tat.InvocationTask{
		finishedTask("TIMEOUT", "killed before completion", ""),
	}
	setupRunScriptEnv(t, m.server.URL)

	_, rerr := RunScript(context.Background(), "ins-test", "fake.sh", 5, "root", nil, nil)
	if rerr == nil {
		t.Fatal("TIMEOUT 应返回错误")
	}
	if detail := detailStr(rerr); detail != "killed before completion" {
		t.Errorf("TIMEOUT detail 应为 stdout, got %q", detail)
	}
}

// ─── 用例 4：DELIVER_FAILED 路径 - 关键新逻辑：ErrorInfo 进入 detail ─

func TestRunScript_DeliverFailed_DetailFromTATErrorInfo(t *testing.T) {
	m := newRunScriptMockServer(t)
	m.invocationStates = []*tat.InvocationTask{
		finishedTask("DELIVER_FAILED", "", "agent offline: instance ins-xxx not connected"),
	}
	setupRunScriptEnv(t, m.server.URL)

	_, rerr := RunScript(context.Background(), "ins-test", "fake.sh", 5, "root", nil, nil)
	if rerr == nil {
		t.Fatal("DELIVER_FAILED 应返回错误")
	}
	if detail := detailStr(rerr); detail != "agent offline: instance ins-xxx not connected" {
		t.Errorf("DELIVER_FAILED detail 应来自 ErrorInfo, got %q", detail)
	}
}

// ─── 用例 5：START_FAILED 路径 - 关键新逻辑：ErrorInfo 进入 detail ──

func TestRunScript_StartFailed_DetailFromTATErrorInfo(t *testing.T) {
	m := newRunScriptMockServer(t)
	m.invocationStates = []*tat.InvocationTask{
		finishedTask("START_FAILED", "", "user 'nobody' not found on instance"),
	}
	setupRunScriptEnv(t, m.server.URL)

	_, rerr := RunScript(context.Background(), "ins-test", "fake.sh", 5, "root", nil, nil)
	if rerr == nil {
		t.Fatal("START_FAILED 应返回错误")
	}
	if detail := detailStr(rerr); detail != "user 'nobody' not found on instance" {
		t.Errorf("START_FAILED detail 应来自 ErrorInfo, got %q", detail)
	}
}

// ─── 用例 6：未知 status 视为已结束但无错误 ──────────────────────

func TestRunScript_UnknownStatus(t *testing.T) {
	m := newRunScriptMockServer(t)
	m.invocationStates = []*tat.InvocationTask{
		finishedTask("WEIRD_STATUS", "  some output  ", ""),
	}
	setupRunScriptEnv(t, m.server.URL)

	out, rerr := RunScript(context.Background(), "ins-test", "fake.sh", 5, "root", nil, nil)
	if rerr != nil {
		t.Errorf("未知 status 不应返回错误, got %v", rerr)
	}
	if out != "some output" {
		t.Errorf("未知 status output 应 trim, got %q", out)
	}
}

// ─── 用例 7：WaitResultTimeout 路径 - 关键新逻辑：detail + 异步 followup ─

func TestRunScript_WaitResultTimeout_DetailIncluded(t *testing.T) {
	m := newRunScriptMockServer(t)
	// 永远返回 RUNNING（无 EndTime），让主循环熬到 deadline。
	m.invocationStates = []*tat.InvocationTask{runningTask()}
	setupRunScriptEnv(t, m.server.URL)
	// 配合 setupRunScriptEnv：runScriptDeadlineBuffer=50ms、poll=1ms、timeout=0
	// → deadline ≈ now+50ms，能在 ~50ms 内可靠触发 WaitResultTimeout 分支。

	_, rerr := RunScript(context.Background(), "ins-test", "fake.sh", 0, "root", nil, nil)
	if rerr == nil {
		t.Fatal("WaitResultTimeout 应返回错误")
	}
	detail := detailStr(rerr)
	if !strings.Contains(detail, "invocation_id=inv-mock-12345") {
		t.Errorf("detail 应包含 invocation_id, got %q", detail)
	}
	if !strings.Contains(detail, "timeout=") {
		t.Errorf("detail 应包含 timeout, got %q", detail)
	}
	// 至少轮询了一次（说明真的走到主循环而不是被 deadline=now 直接跳出）
	if atomic.LoadInt32(&m.describeCallCount) == 0 {
		t.Error("应至少轮询一次 DescribeInvocationTasks")
	}
	// 给后台 followup goroutine 一点时间结束（也用了 fast timings，<= 50ms）
	time.Sleep(100 * time.Millisecond)
}

// ─── 用例 8：LoadScript 失败 ────────────────────────────────────

func TestRunScript_LoadScriptFails(t *testing.T) {
	origLoadScript := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: script not found")
	}
	defer func() { LoadScript = origLoadScript }()

	_, rerr := RunScript(context.Background(), "ins-test", "missing.sh", 5, "root", nil, nil)
	if rerr == nil {
		t.Fatal("LoadScript 失败时应返回错误")
	}
	if !strings.Contains(rerr.Error(), "missing.sh") &&
		!strings.Contains(detailStr(rerr), "missing.sh") {
		t.Errorf("错误应包含脚本名 missing.sh, got msg=%q detail=%q",
			rerr.Error(), detailStr(rerr))
	}
}

// ─── 用例 9：DescribeInvocationTasks 调用失败 ───────────────────

func TestRunScript_DescribeInvocationTasksFails(t *testing.T) {
	// mock server 对 DescribeInvocationTasks 永远返回 500
	var ranCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.Header.Get("X-TC-Action")
		w.Header().Set("Content-Type", "application/json")
		switch action {
		case "DescribeAutomationAgentStatus":
			resp := tat.DescribeAutomationAgentStatusResponse{
				Response: &tat.DescribeAutomationAgentStatusResponseParams{
					AutomationAgentSet: []*tat.AutomationAgentInfo{{AgentStatus: sptr("Online")}},
					RequestId:          sptr("r"),
				},
			}
			b, _ := json.Marshal(resp)
			w.Write(b)
		case "RunCommand":
			resp := tat.RunCommandResponse{
				Response: &tat.RunCommandResponseParams{
					InvocationId: sptr("inv-x"),
					RequestId:    sptr("r"),
				},
			}
			b, _ := json.Marshal(resp)
			w.Write(b)
		case "DescribeInvocationTasks":
			atomic.AddInt32(&ranCount, 1)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}))
	defer ts.Close()

	// 注入 mock
	origLoadScript := LoadScript
	LoadScript = func(name string) (string, error) { return "#!/bin/bash\necho ok\n", nil }
	defer func() { LoadScript = origLoadScript }()

	installMockTATClient(t, ts.URL)
	origInterval := runScriptPollInterval
	runScriptPollInterval = 1 * time.Millisecond
	defer func() { runScriptPollInterval = origInterval }()

	_, rerr := RunScript(context.Background(), "ins-test", "fake.sh", 5, "root", nil, nil)
	if rerr == nil {
		t.Fatal("DescribeInvocationTasks 失败时应返回错误")
	}
	if atomic.LoadInt32(&ranCount) == 0 {
		t.Error("应至少调用一次 DescribeInvocationTasks")
	}
}

// ─── 用例 10：onOutput 回调被触发（验证流式输出能力未破坏） ────────

func TestRunScript_OnOutputCallback(t *testing.T) {
	m := newRunScriptMockServer(t)
	m.invocationStates = []*tat.InvocationTask{
		// 第 1 次：partial 输出 + 未结束
		{
			TaskStatus: sptr("RUNNING"),
			TaskResult: &tat.TaskResult{Output: b64("partial-1\n")},
		},
		// 第 2 次：结束 + 完整输出
		finishedTask("SUCCESS", "partial-1\nfinal-line", ""),
	}
	setupRunScriptEnv(t, m.server.URL)

	var captured []string
	onOutput := func(chunk string) {
		captured = append(captured, chunk)
	}
	out, rerr := RunScript(context.Background(), "ins-test", "fake.sh", 5, "root", onOutput, nil)
	if rerr != nil {
		t.Fatalf("不应返回错误: %v", rerr)
	}
	if out != "partial-1\nfinal-line" {
		t.Errorf("最终 output 不对, got %q", out)
	}
	if len(captured) < 2 {
		t.Errorf("onOutput 应被调用至少 2 次, 实际=%d", len(captured))
	}
}
