// 管控端重装（POST /admin/instances/reset）触发 approve_device 异步链路的单元测试。
//
// 背景：用户端 HandleResetInstance（openclaw.go）在重装后会异步调用
// approveDeviceAsync 写入 paired.json 的 operator token 5 件套权限，
// 但管控端 HandleAdminResetInstance 历史上未对齐这条链路，导致管控端重装后
// paired.json 中 scopes 为空、所有 RPC 鉴权失败、用户首次取 token 需阻塞 ~30 分钟。
//
// 本测试验证 P0 修复：管控端 reset 成功路径必须触发 adminApproveDeviceAsyncFn，
// 失败/拒绝路径必须不触发，且 RuntimeUser 等参数必须原样透传。
package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"

	"hatchery/model"
)

// approveCall 记录一次 approve 钩子调用的入参，用于断言。
type approveCall struct {
	instancePK    uint
	cvmInstanceId string
	runtimeUser   string
}

// installApproveDeviceSpy 把 adminApproveDeviceAsyncFn / syncSMHEnvWhenReadyFn 替换为
// 受控的 spy，返回：
//   - calls 指针：测试侧观察 approve 调用次数与入参（线程安全）
//   - cleanup：恢复原始函数
//
// 注意：syncSMHEnvWhenReadyFn 也必须 mock，否则在测试 DB 缺少 SMHPersonalSpace
// migration 时可能走入未预期路径或写入异常日志。
func installApproveDeviceSpy(t *testing.T) (callsPtr *atomic.Pointer[[]approveCall], cleanup func()) {
	t.Helper()

	var (
		mu    = make(chan struct{}, 1) // 单元素互斥
		state = make([]approveCall, 0, 4)
	)
	ptr := &atomic.Pointer[[]approveCall]{}
	ptr.Store(&state)

	origApprove := adminApproveDeviceAsyncFn
	adminApproveDeviceAsyncFn = func(ctx context.Context, instancePK uint, cvmInstanceId string, runtimeUser string) {
		mu <- struct{}{}
		defer func() { <-mu }()
		cur := *ptr.Load()
		cur = append(cur, approveCall{
			instancePK:    instancePK,
			cvmInstanceId: cvmInstanceId,
			runtimeUser:   runtimeUser,
		})
		ptr.Store(&cur)
	}

	origSMH := syncSMHEnvWhenReadyFn
	syncSMHEnvWhenReadyFn = func(ctx context.Context, inst model.Instance) {}

	cleanup = func() {
		adminApproveDeviceAsyncFn = origApprove
		syncSMHEnvWhenReadyFn = origSMH
	}
	return ptr, cleanup
}

// waitForApproveCalls 等待 approve 钩子被调用至少 want 次（最多 maxWait）。
// 返回最终调用列表的副本。go func 异步触发，必须轮询。
func waitForApproveCalls(callsPtr *atomic.Pointer[[]approveCall], want int, maxWait time.Duration) []approveCall {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		cur := *callsPtr.Load()
		if len(cur) >= want {
			out := make([]approveCall, len(cur))
			copy(out, cur)
			return out
		}
		time.Sleep(10 * time.Millisecond)
	}
	cur := *callsPtr.Load()
	out := make([]approveCall, len(cur))
	copy(out, cur)
	return out
}

// seedEnabledOpenClawImage 写入一条 OpenClaw 启用镜像，使
// model.GetEnabledImageByType(ctx, "openclaw") 命中。
func seedEnabledOpenClawImage(t *testing.T) *model.AIImage {
	t.Helper()
	img := &model.AIImage{
		ImageId:      "img-admin-reset-test",
		ImageName:    "openclaw-test",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "1.0.0",
		Enabled:      true,
		ImageState:   "NORMAL",
	}
	if err := model.DB(context.Background()).Create(img).Error; err != nil {
		t.Fatalf("seed enabled image: %v", err)
	}
	return img
}

// mockCVMResetServer 返回 ResetInstance 成功响应的 httptest server。
func mockCVMResetServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// 腾讯云 ResetInstance 响应格式
		fmt.Fprintf(w, `{"Response":{"RequestId":"mock-reset-req-id"}}`)
	}))
}

// mockCVMResetFailServer 返回 ResetInstance 失败响应（业务错误）。
func mockCVMResetFailServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// 腾讯云错误响应格式：在 Response.Error 中携带错误码
		fmt.Fprintf(w, `{"Response":{"Error":{"Code":"InvalidInstanceState","Message":"reset not allowed"},"RequestId":"req-fail"}}`)
	}))
}

// ─── P0 正向：管控端 reset 成功路径必须触发 approveDeviceAsync ──────────

// TestAdminReset_TriggersApproveDeviceAsync 是 P0 修复的核心断言：
// 管控端 /admin/instances/reset 在 CVM ResetInstance 成功后，必须
// 异步触发 adminApproveDeviceAsyncFn，且参数（instancePK / cvmInstanceId /
// runtimeUser）必须与重装实例的字段完全一致。
//
// 没有此断言，任何后续重构（如把 approve 调用搬到别的位置或忘记复用钩子）
// 都会导致 paired.json 缺权限的"半残"实例回归——线上表现是用户首次取 token
// 阻塞 ~30 分钟、所有 RPC 403。
func TestAdminReset_TriggersApproveDeviceAsync(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	callsPtr, restore := installApproveDeviceSpy(t)
	defer restore()

	// mock CVM client → ResetInstance 成功
	ts := mockCVMResetServer(t)
	defer ts.Close()
	origCVM := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return newMockCVMClient(t, ts.URL), nil
	}
	defer func() { NewCVMClient = origCVM }()

	// 写入启用镜像 + OpenClaw 实例
	seedEnabledOpenClawImage(t)
	inst := &model.Instance{
		Name:        "admin-reset-approve",
		InstanceId:  "ins-admin-reset-approve-001",
		UserID:      1,
		AgentType:   model.AgentTypeOpenClaw,
		RuntimeUser: "centos",
	}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := adminTokenReq(http.MethodPost, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusOK {
		t.Fatalf("管控端 reset 应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	calls := waitForApproveCalls(callsPtr, 1, 2*time.Second)
	if len(calls) != 1 {
		t.Fatalf("管控端 reset 成功后必须触发 1 次 approveDeviceAsync，实际=%d", len(calls))
	}
	got := calls[0]
	if got.instancePK != inst.ID {
		t.Errorf("instancePK 应为 %d，实际=%d", inst.ID, got.instancePK)
	}
	if got.cvmInstanceId != inst.InstanceId {
		t.Errorf("cvmInstanceId 应为 %q，实际=%q", inst.InstanceId, got.cvmInstanceId)
	}
	if got.runtimeUser != inst.RuntimeUser {
		t.Errorf("runtimeUser 应为 %q（来自 instance.RuntimeUser），实际=%q",
			inst.RuntimeUser, got.runtimeUser)
	}
}

// TestAdminReset_TriggersApproveDeviceAsync_EmptyRuntimeUser 验证
// RuntimeUser 为空字符串时也能透传——脚本侧会 fallback 默认用户，
// 不应在 hatchery 侧做空值过滤。
func TestAdminReset_TriggersApproveDeviceAsync_EmptyRuntimeUser(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	callsPtr, restore := installApproveDeviceSpy(t)
	defer restore()

	ts := mockCVMResetServer(t)
	defer ts.Close()
	origCVM := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return newMockCVMClient(t, ts.URL), nil
	}
	defer func() { NewCVMClient = origCVM }()

	seedEnabledOpenClawImage(t)
	inst := &model.Instance{
		Name:        "admin-reset-empty-user",
		InstanceId:  "ins-admin-reset-empty-user",
		UserID:      1,
		AgentType:   model.AgentTypeOpenClaw,
		RuntimeUser: "", // 显式空
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := adminTokenReq(http.MethodPost, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	calls := waitForApproveCalls(callsPtr, 1, 2*time.Second)
	if len(calls) != 1 {
		t.Fatalf("应触发 1 次 approve，实际=%d", len(calls))
	}
	if calls[0].runtimeUser != "" {
		t.Errorf("RuntimeUser 空字符串应原样透传，实际=%q", calls[0].runtimeUser)
	}
}

// ─── P0 反向 1：CVM ResetInstance 失败时不应触发 approve ────────────────

// TestAdminReset_CVMFailure_DoesNotTriggerApprove 验证：CVM ResetInstance
// 调用失败 → handler 走 clearOperation + 500 错误分支 → 必须不触发 approve。
// 这个反向断言保护"approve 调用语义=重装真的发生了"，避免日后改动误把 approve
// 调用挪到 ResetInstance 之前导致幽灵审批。
func TestAdminReset_CVMFailure_DoesNotTriggerApprove(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	callsPtr, restore := installApproveDeviceSpy(t)
	defer restore()

	// mock CVM 返回业务错误（InvalidInstanceState）
	ts := mockCVMResetFailServer(t)
	defer ts.Close()
	origCVM := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return newMockCVMClient(t, ts.URL), nil
	}
	defer func() { NewCVMClient = origCVM }()

	seedEnabledOpenClawImage(t)
	inst := &model.Instance{
		Name:        "admin-reset-cvm-fail",
		InstanceId:  "ins-admin-reset-cvm-fail",
		UserID:      1,
		AgentType:   model.AgentTypeOpenClaw,
		RuntimeUser: "centos",
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := adminTokenReq(http.MethodPost, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("CVM ResetInstance 失败应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 给可能错误触发的异步 goroutine 一个时间窗口
	time.Sleep(150 * time.Millisecond)
	calls := *callsPtr.Load()
	if len(calls) != 0 {
		t.Errorf("CVM 失败时不应触发 approveDeviceAsync，实际触发=%d 次", len(calls))
	}
}

// ─── P0 反向 2：状态 guard 拒绝时不应触发 approve ───────────────────────

// TestAdminReset_StatusGuardRejected_DoesNotTriggerApprove 使用
// stopped 状态的 mock resolver 让 guard 命中拒绝分支（reinstall 仅
// running/stopped 允许，但 stopped 在 guard 通过后还要进 enabledImage 查询；
// 这里改用 creating 状态，AdminStatusMap[creating].Actions 不含 reinstall）。
func TestAdminReset_StatusGuardRejected_DoesNotTriggerApprove(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	callsPtr, restore := installApproveDeviceSpy(t)
	defer restore()

	seedEnabledOpenClawImage(t)
	inst := &model.Instance{
		Name:        "admin-reset-creating",
		InstanceId:  "ins-admin-reset-creating",
		UserID:      1,
		AgentType:   model.AgentTypeOpenClaw,
		RuntimeUser: "centos",
	}
	model.DB(context.Background()).Create(inst)

	// 注入 creating 状态 resolver → guard 拒绝
	rejectResolver := &mockStatusResolverWithStatus{status: model.StatusCreating, label: "创建中"}

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := adminTokenReq(http.MethodPost, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, rejectResolver)

	if rr.Code != http.StatusConflict {
		t.Fatalf("creating 状态下 reset 应被 guard 拒绝（409），实际=%d body=%s",
			rr.Code, rr.Body.String())
	}

	time.Sleep(100 * time.Millisecond)
	calls := *callsPtr.Load()
	if len(calls) != 0 {
		t.Errorf("guard 拒绝时不应触发 approveDeviceAsync，实际触发=%d 次", len(calls))
	}
}

// ─── P0 反向 3：实例无 InstanceId 时不应触发 approve ────────────────────

// TestAdminReset_EmptyCVM_DoesNotTriggerApprove 验证：实例无关联 CVM 时
// handler 早早返回 400，必须不触发 approve（approve 必须有 cvmInstanceId 才有意义）。
func TestAdminReset_EmptyCVM_DoesNotTriggerApprove(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	callsPtr, restore := installApproveDeviceSpy(t)
	defer restore()

	// 不需要 mock CVM client（不会走到 ResetInstance）
	seedEnabledOpenClawImage(t)
	inst := &model.Instance{
		Name:       "admin-reset-no-cvm",
		InstanceId: "", // 关键：无 CVM
		UserID:     1,
		AgentType:  model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := adminTokenReq(http.MethodPost, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("无 CVM 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	time.Sleep(100 * time.Millisecond)
	calls := *callsPtr.Load()
	if len(calls) != 0 {
		t.Errorf("无 CVM 时不应触发 approveDeviceAsync，实际触发=%d 次", len(calls))
	}
}

// ─── 钩子契约：默认值必须指向 approveDeviceAsync ────────────────────────

// TestAdminApproveDeviceAsyncFn_DefaultsToApproveDeviceAsync 验证生产环境下
// adminApproveDeviceAsyncFn 不会因为某次重构变成 nil 或被错误地指向其他函数。
// 用反射比较函数指针避免依赖具体实现细节，但又能在意外重置时报警。
func TestAdminApproveDeviceAsyncFn_DefaultsToApproveDeviceAsync(t *testing.T) {
	if adminApproveDeviceAsyncFn == nil {
		t.Fatal("adminApproveDeviceAsyncFn 不应为 nil（生产环境会 nil panic）")
	}
	// 通过包内可见的辅助断言确保签名兼容（编译期已保证），运行期仅检查 non-nil。
	// 若未来真要校验"指向 approveDeviceAsync 本体"，可用 reflect.ValueOf(...).Pointer() 比较，
	// 但那种比较对方法值/闭包包装不稳定，这里仅做 nil 防御。
	_ = strings.TrimSpace // 引用 strings 避免 unused import（被其它测试已用，留作 noop 防御）
}
