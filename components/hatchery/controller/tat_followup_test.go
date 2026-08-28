package controller

import (
	"bytes"
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

	tcCommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tcprofile "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tat "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat/v20201028"
)

// ────────────────────────────────────────────────────────────────────
// asyncFollowupInvocation 测试
//
// 测试策略（参考项目里 newMockCVMClient + httptest.Server 模式）：
//   1. 启动 httptest.Server 模拟腾讯云 TAT 后端
//   2. mock NewTATClient 让真实 SDK client 的 endpoint 指向 mock server
//   3. 缩短 asyncFollowupInterval / asyncFollowupTotalTimeout，使测试在毫秒级完成
// ────────────────────────────────────────────────────────────────────

// newMockTATClientPointingTo 构造一个把 HTTP 调用劫持到 mock server 的真实 TAT client。
func newMockTATClientPointingTo(t *testing.T, serverURL string) *tat.Client {
	t.Helper()
	endpoint := strings.TrimPrefix(serverURL, "http://")
	credential := tcCommon.NewCredential("fake-id", "fake-key")
	cpf := tcprofile.NewClientProfile()
	cpf.HttpProfile.Endpoint = endpoint
	cpf.HttpProfile.Scheme = "HTTP"
	client, err := tat.NewClient(credential, "ap-guangzhou", cpf)
	if err != nil {
		t.Fatalf("创建 mock TAT client 失败: %v", err)
	}
	return client
}

// installFastFollowupTimings 把 followup 节拍缩到毫秒级，t.Cleanup 自动还原。
func installFastFollowupTimings(t *testing.T, interval, total time.Duration) {
	t.Helper()
	origInterval, origTotal := asyncFollowupInterval, asyncFollowupTotalTimeout
	asyncFollowupInterval = interval
	asyncFollowupTotalTimeout = total
	t.Cleanup(func() {
		asyncFollowupInterval = origInterval
		asyncFollowupTotalTimeout = origTotal
	})
}

// installMockTATClient 替换 NewTATClient 为返回指向 mock server 的 client。
func installMockTATClient(t *testing.T, serverURL string) {
	t.Helper()
	orig := NewTATClient
	NewTATClient = func(ctx context.Context) (*tat.Client, error) {
		return newMockTATClientPointingTo(t, serverURL), nil
	}
	t.Cleanup(func() { NewTATClient = orig })
}

func sptr(s string) *string { return &s }
func uptr(v uint64) *uint64 { return &v }

// makeTATFinishedRespJSON 构造一个"任务已结束"的 TAT API 响应。
func makeTATFinishedRespJSON(status, output, errorInfo string) string {
	resp := tat.DescribeInvocationTasksResponse{
		Response: &tat.DescribeInvocationTasksResponseParams{
			TotalCount: uptr(1),
			InvocationTaskSet: []*tat.InvocationTask{{
				TaskStatus: sptr(status),
				EndTime:    sptr("2026-06-24T10:00:00Z"),
				TaskResult: &tat.TaskResult{
					Output: sptr(base64.StdEncoding.EncodeToString([]byte(output))),
				},
				ErrorInfo: sptr(errorInfo),
			}},
			RequestId: sptr("mock-req-id"),
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

// makeTATRunningRespJSON 构造一个"任务仍在运行"（EndTime 为空）的响应。
func makeTATRunningRespJSON() string {
	resp := tat.DescribeInvocationTasksResponse{
		Response: &tat.DescribeInvocationTasksResponseParams{
			TotalCount: uptr(1),
			InvocationTaskSet: []*tat.InvocationTask{{
				TaskStatus: sptr("RUNNING"),
			}},
			RequestId: sptr("mock-req-id"),
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

// makeTATEmptySetRespJSON 构造一个"任务集合为空"的响应。
func makeTATEmptySetRespJSON() string {
	resp := tat.DescribeInvocationTasksResponse{
		Response: &tat.DescribeInvocationTasksResponseParams{
			TotalCount:        uptr(0),
			InvocationTaskSet: []*tat.InvocationTask{},
			RequestId:         sptr("mock-req-id"),
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func logHas(buf *bytes.Buffer, sub string) bool {
	return strings.Contains(buf.String(), sub)
}

// ─── 用例 1：拿到最终状态后输出日志并退出 ────────────────────────────

func TestAsyncFollowupInvocation_FinalStatusLogged(t *testing.T) {
	var callCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			fmt.Fprint(w, makeTATRunningRespJSON())
			return
		}
		fmt.Fprint(w, makeTATFinishedRespJSON("SUCCESS", "[{\"name\":\"foo\"}]", ""))
	}))
	defer ts.Close()

	installMockTATClient(t, ts.URL)
	installFastFollowupTimings(t, 5*time.Millisecond, 500*time.Millisecond)
	buf, restore := captureSlog(t)
	defer restore()

	asyncFollowupInvocation(context.Background(), "inv-final-success", "list_skills.sh")

	if got := atomic.LoadInt32(&callCount); got < 2 {
		t.Errorf("应至少轮询 2 次, 实际=%d", got)
	}
	if !logHas(buf, "续查到最终状态") {
		t.Errorf("日志应包含'续查到最终状态'，实际=%s", buf.String())
	}
	if !logHas(buf, "SUCCESS") {
		t.Errorf("日志应包含 SUCCESS，实际=%s", buf.String())
	}
	if !logHas(buf, "inv-final-success") {
		t.Errorf("日志应包含 invocation id, 实际=%s", buf.String())
	}
}

// ─── 用例 2：失败状态 + ErrorInfo 透传到日志 ──────────────────────────

func TestAsyncFollowupInvocation_DeliverFailedWithErrorInfo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, makeTATFinishedRespJSON("DELIVER_FAILED", "", "agent offline: ins-xxx"))
	}))
	defer ts.Close()

	installMockTATClient(t, ts.URL)
	installFastFollowupTimings(t, 1*time.Millisecond, 100*time.Millisecond)
	buf, restore := captureSlog(t)
	defer restore()

	asyncFollowupInvocation(context.Background(), "inv-deliver-failed", "list_skills.sh")

	if !logHas(buf, "DELIVER_FAILED") {
		t.Errorf("应记录 DELIVER_FAILED，实际=%s", buf.String())
	}
	if !logHas(buf, "agent offline") {
		t.Errorf("ErrorInfo 应出现在日志，实际=%s", buf.String())
	}
}

// ─── 用例 3：API 调用错误时会继续重试 ────────────────────────────────

func TestAsyncFollowupInvocation_APIErrorRetried(t *testing.T) {
	var callCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		if n <= 2 {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, makeTATFinishedRespJSON("SUCCESS", "ok", ""))
	}))
	defer ts.Close()

	installMockTATClient(t, ts.URL)
	installFastFollowupTimings(t, 1*time.Millisecond, 500*time.Millisecond)
	buf, restore := captureSlog(t)
	defer restore()

	asyncFollowupInvocation(context.Background(), "inv-retry", "list_skills.sh")

	if got := atomic.LoadInt32(&callCount); got < 3 {
		t.Errorf("应至少重试 3 次, 实际=%d", got)
	}
	if !logHas(buf, "查询失败") {
		t.Errorf("应有查询失败日志，实际=%s", buf.String())
	}
	if !logHas(buf, "SUCCESS") {
		t.Errorf("最终应记录 SUCCESS, 实际=%s", buf.String())
	}
}

// ─── 用例 4：超时未拿到结果时输出"放弃追踪"日志 ──────────────────────

func TestAsyncFollowupInvocation_DeadlineExceeded(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, makeTATRunningRespJSON())
	}))
	defer ts.Close()

	installMockTATClient(t, ts.URL)
	installFastFollowupTimings(t, 5*time.Millisecond, 30*time.Millisecond)
	buf, restore := captureSlog(t)
	defer restore()

	asyncFollowupInvocation(context.Background(), "inv-deadline", "list_skills.sh")

	if !logHas(buf, "放弃追踪") {
		t.Errorf("应输出'放弃追踪'日志，实际=%s", buf.String())
	}
}

// ─── 用例 5：响应中 InvocationTaskSet 为空时继续轮询 ──────────────────

func TestAsyncFollowupInvocation_EmptyTaskSetIgnored(t *testing.T) {
	var callCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			fmt.Fprint(w, makeTATEmptySetRespJSON())
			return
		}
		fmt.Fprint(w, makeTATFinishedRespJSON("SUCCESS", "", ""))
	}))
	defer ts.Close()

	installMockTATClient(t, ts.URL)
	installFastFollowupTimings(t, 1*time.Millisecond, 200*time.Millisecond)
	buf, restore := captureSlog(t)
	defer restore()

	asyncFollowupInvocation(context.Background(), "inv-empty-set", "list_skills.sh")

	if got := atomic.LoadInt32(&callCount); got < 2 {
		t.Errorf("空 TaskSet 应继续轮询, 实际调用=%d", got)
	}
	if !logHas(buf, "SUCCESS") {
		t.Errorf("应最终拿到 SUCCESS, 实际=%s", buf.String())
	}
}

// ─── 用例 6：NewTATClient 失败时早退 ──────────────────────────────────

func TestAsyncFollowupInvocation_ClientCreationFails(t *testing.T) {
	origFactory := NewTATClient
	NewTATClient = func(ctx context.Context) (*tat.Client, error) {
		return nil, fmt.Errorf("mock credential not configured")
	}
	defer func() { NewTATClient = origFactory }()

	installFastFollowupTimings(t, 1*time.Millisecond, 100*time.Millisecond)
	buf, restore := captureSlog(t)
	defer restore()

	asyncFollowupInvocation(context.Background(), "inv-client-fail", "list_skills.sh")

	if !logHas(buf, "创建 TAT client 失败") {
		t.Errorf("应输出创建失败日志, 实际=%s", buf.String())
	}
	if !logHas(buf, "mock credential not configured") {
		t.Errorf("应包含底层错误信息, 实际=%s", buf.String())
	}
	if logHas(buf, "续查到最终状态") {
		t.Errorf("client 创建失败时不应走到 followup 主循环")
	}
}

// ─── 用例 7：base64 解码失败时 Output 留空，仍然记录日志 ─────────────

func TestAsyncFollowupInvocation_InvalidBase64Output(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := tat.DescribeInvocationTasksResponse{
			Response: &tat.DescribeInvocationTasksResponseParams{
				InvocationTaskSet: []*tat.InvocationTask{{
					TaskStatus: sptr("FAILED"),
					EndTime:    sptr("2026-06-24T10:00:00Z"),
					TaskResult: &tat.TaskResult{Output: sptr("!!not-base64!!")},
				}},
				RequestId: sptr("mock-req"),
			},
		}
		b, _ := json.Marshal(resp)
		w.Write(b)
	}))
	defer ts.Close()

	installMockTATClient(t, ts.URL)
	installFastFollowupTimings(t, 1*time.Millisecond, 100*time.Millisecond)
	buf, restore := captureSlog(t)
	defer restore()

	asyncFollowupInvocation(context.Background(), "inv-bad-b64", "list_skills.sh")

	if !logHas(buf, "FAILED") {
		t.Errorf("应记录 FAILED 状态，实际=%s", buf.String())
	}
}
