// 本文件集中验证：在 P0+P1+P2 整理阶段为多个接口入口加上 rejectLocalOrWrite
// 后，本地 agent 实例（Source=local）能被早期拒绝、不会再走到调 CVM API 的代码路径。
//
// 设计意图：
//   - 单测仅断言「入口拒绝」这一行为，不进入业务逻辑；
//   - 期望状态码：默认 400；HandleBrowserStatus 例外（高频轮询接口走 200+unsupported）；
//   - 测试数据使用 Source=local + InstanceId="local-xxx-001"（host CID 形态），
//     一旦 reject 失效，下游会被 CVM SDK 客户端校验报「实例ID不合要求」，从错误
//     消息也可以兜底识别。
package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"strconv"

	"hatchery/model"
)

// localRejectInst 在内存 DB 里建一个标准的本地 agent 实例并返回。
// 直接复用 initInstanceIDParamTestDB / initBatchUpgradeTestDB 的 DB 注入。
func localRejectInst(t *testing.T, username, instanceID string) (*model.User, *model.Instance) {
	t.Helper()
	user := &model.User{Username: username, Password: "test", Role: "user"}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	inst := &model.Instance{
		Name:       "local-" + username,
		InstanceId: instanceID,
		Source:     model.InstanceSourceLocal,
		UserID:     user.ID,
	}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	return user, inst
}

// assertLocalRejected 统一的断言：本地实例被入口拒绝，不会因 CVM ID 格式报错。
func assertLocalRejected(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()
	if w.Code != expectedStatus {
		t.Fatalf("期望 %d，实际=%d body=%s", expectedStatus, w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "实例ID") && strings.Contains(w.Body.String(), "不合要求") {
		t.Errorf("本地实例不应触发 CVM 客户端 ID 校验报错，body=%s", w.Body.String())
	}
}

// ─── 用户端接口 ──────────────────────────────────────────────────────────────

// TestHandleApprove_LocalInstance 飞书设备配对 approve 不适用本地 agent。
func TestHandleApprove_LocalInstance(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()
	_, inst := localRejectInst(t, "ap-local", "local-codebuddy-001")

	form := "id=" + strconv.FormatUint(uint64(inst.ID), 10)
	req := userInstanceIDReqWithSession(t, http.MethodPost, "/openclaw/approve", "ap-local")
	req.Body = io.NopCloser(strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	HandleApprove(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

// TestHandleRetryInstance_LocalInstance 重试创建对本地 agent 没有意义。
func TestHandleRetryInstance_LocalInstance(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()
	_, inst := localRejectInst(t, "rt-local", "local-codebuddy-001")

	form := "id=" + strconv.FormatUint(uint64(inst.ID), 10)
	req := userInstanceIDReqWithSession(t, http.MethodPost, "/openclaw/retry", "rt-local")
	req.Body = io.NopCloser(strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	HandleRetryInstance(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

// TestHandleGetEnv_LocalInstance 读 env 不走 TAT，对本地 agent 拒绝。
func TestHandleGetEnv_LocalInstance(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()
	_, inst := localRejectInst(t, "ge-local", "local-codebuddy-001")

	req := userInstanceIDReqWithSession(t, http.MethodGet, "/openclaw/env?id="+strconv.FormatUint(uint64(inst.ID), 10), "ge-local")
	w := httptest.NewRecorder()
	HandleGetEnv(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

// TestHandleGetWSUrl_LocalInstance 即使绕过 ins- 前缀校验，DB 命中本地实例也应被拒绝。
// 注：这里 instance_id 必须以 ins- 开头才能通过前缀校验进入 DB 查询，但本地实例的
// instance_id 实际是 "local-xxx" 不会以 ins- 开头，所以走到 DB 时永远 ErrRecordNotFound
// → 走 403 "未找到或无权限"分支。这反过来说明：rejectLocalOrWrite 是**第二道**防线，
// 兜底而非主拦截。这里我们用一个特殊的"伪装本地实例"（instance_id = "ins-local0001"）
// 验证第二道防线生效。
func TestHandleGetWSUrl_LocalInstance(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()
	user := &model.User{Username: "ws-local", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)
	// instance_id 故意伪装成 ins- 前缀以通过第一道校验
	inst := &model.Instance{
		Name: "fake-cvm", InstanceId: "ins-local0001",
		Source: model.InstanceSourceLocal, UserID: user.ID,
	}
	model.DB(context.Background()).Create(inst)

	body, _ := json.Marshal(map[string]string{"instance_id": "ins-local0001"})
	req := userInstanceIDReqWithSession(t, http.MethodPost, "/openclaw/ws-url", "ws-local")
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	HandleGetWSUrl(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

// TestHandleBrowserStatus_LocalInstance 高频轮询接口走 200 + unsupported=true。
func TestHandleBrowserStatus_LocalInstance(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()
	_, inst := localRejectInst(t, "bs-local", "local-codebuddy-001")

	req := userInstanceIDReqWithSession(t, http.MethodGet, "/openclaw/browser-status?id="+strconv.FormatUint(uint64(inst.ID), 10), "bs-local")
	w := httptest.NewRecorder()
	HandleBrowserStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("BrowserStatus 对本地实例应返回 200（保持轮询稳定），实际=%d body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	data, _ := resp["data"].(map[string]interface{})
	if data == nil {
		t.Fatalf("响应缺少 data 字段，body=%s", resp)
	}
	if unsupported, ok := data["unsupported"].(bool); !ok || !unsupported {
		t.Errorf("期望 data.unsupported=true，实际=%v", data["unsupported"])
	}
	if active, ok := data["ai_active"].(bool); !ok || active {
		t.Errorf("期望 data.ai_active=false，实际=%v", data["ai_active"])
	}
}

// TestHandleBrowserVNCAccess_LocalInstance 浏览器 VNC 访问对本地 agent 拒绝。
func TestHandleBrowserVNCAccess_LocalInstance(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()
	_, inst := localRejectInst(t, "bv-local", "local-codebuddy-001")

	form := "id=" + strconv.FormatUint(uint64(inst.ID), 10)
	req := userInstanceIDReqWithSession(t, http.MethodPost, "/openclaw/browser-vnc-access", "bv-local")
	req.Body = io.NopCloser(strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	HandleBrowserVNCAccess(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

// TestHandleRetryFailedSkills_LocalInstance Skill 失败重试对本地 agent 拒绝
// （本地实例 skill 走 local_instance_skills 表，失败重试由 reporter 负责）。
func TestHandleRetryFailedSkills_LocalInstance(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()
	_, inst := localRejectInst(t, "rf-local", "local-codebuddy-001")

	form := "id=" + strconv.FormatUint(uint64(inst.ID), 10)
	req := userInstanceIDReqWithSession(t, http.MethodPost, "/openclaw/retry-failed-skills", "rf-local")
	req.Body = io.NopCloser(strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	HandleRetryFailedSkills(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

// TestHandleCancelFailedSkills_LocalInstance 同上。
func TestHandleCancelFailedSkills_LocalInstance(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()
	_, inst := localRejectInst(t, "cf-local", "local-codebuddy-001")

	form := "id=" + strconv.FormatUint(uint64(inst.ID), 10)
	req := userInstanceIDReqWithSession(t, http.MethodPost, "/openclaw/cancel-failed-skills", "cf-local")
	req.Body = io.NopCloser(strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	HandleCancelFailedSkills(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

// TestHandleDoctorStart_LocalInstance 龙虾医生针对 CVM 节点，对本地 agent 拒绝。
func TestHandleDoctorStart_LocalInstance(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()
	_, inst := localRejectInst(t, "ds-local", "local-codebuddy-001")

	form := "id=" + strconv.FormatUint(uint64(inst.ID), 10)
	req := userInstanceIDReqWithSession(t, http.MethodPost, "/openclaw/doctor/start", "ds-local")
	req.Body = io.NopCloser(strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	HandleDoctorStart(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

// ─── 管控端接口 ──────────────────────────────────────────────────────────────

// TestHandleAdminDeleteInstance_LocalInstance_Single 单删本地实例：admin 路径
// 不调 CVM TerminateInstances；本地实例走 user 路径删，admin 单删拒绝。
func TestHandleAdminDeleteInstance_LocalInstance_Single(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "ad-local", "local-codebuddy-001")

	form := "id=" + strconv.FormatUint(uint64(inst.ID), 10)
	req := adminFormReq(http.MethodPost, "/admin/instances/delete", form)

	w := httptest.NewRecorder()
	HandleAdminDeleteInstance(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

// TestHandleAdminDeleteInstance_LocalInstance_Batch 批量删除遇到本地实例整体拒绝。
func TestHandleAdminDeleteInstance_LocalInstance_Batch(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "adb-local", "local-codebuddy-001")

	body, _ := json.Marshal(map[string]interface{}{"ids": []uint{inst.ID}})
	req := adminJSONReq(http.MethodPost, "/admin/instances/delete", body)

	w := httptest.NewRecorder()
	HandleAdminDeleteInstance(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

// TestHandleAdminBatchUpgrade_LocalInstance 批量升级遇到本地实例整体拒绝。
func TestHandleAdminBatchUpgrade_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "bu-local", "local-codebuddy-001")

	body, _ := json.Marshal(map[string]interface{}{"ids": []uint{inst.ID}})
	req := adminJSONReq(http.MethodPost, "/admin/instances/batch-upgrade", body)

	w := httptest.NewRecorder()
	HandleAdminBatchUpgrade(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

// TestHandleAdminDetectInstall_LocalInstance 同上。
func TestHandleAdminDetectInstall_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "di-local", "local-codebuddy-001")

	body, _ := json.Marshal(map[string]interface{}{"ids": []uint64{uint64(inst.ID)}})
	req := adminJSONReq(http.MethodPost, "/admin/instances/detect-install", body)

	w := httptest.NewRecorder()
	HandleAdminDetectInstall(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

// TestHandleAdminRefreshInstanceVersion_LocalInstance admin 刷新版本对本地 agent 拒绝。
func TestHandleAdminRefreshInstanceVersion_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "rv-local", "local-codebuddy-001")

	form := "id=" + strconv.FormatUint(uint64(inst.ID), 10)
	req := adminFormReq(http.MethodPost, "/admin/instances/refresh-version", form)

	w := httptest.NewRecorder()
	HandleAdminRefreshInstanceVersion(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

// TestHandleModifyInstancesCamRole_LocalInstance CamRole 批量绑定遇到本地实例拒绝。
func TestHandleModifyInstancesCamRole_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, _ = localRejectInst(t, "cr-local", "local-codebuddy-001")

	body, _ := json.Marshal(map[string]interface{}{"instance_ids": []string{"local-codebuddy-001"}})
	req := adminJSONReq(http.MethodPost, "/admin/instances/cam-role", body)

	w := httptest.NewRecorder()
	HandleModifyInstancesCamRole(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

// ─── 管控端动作类接口（13 个）：补齐 P0+P1 漏网 ─────────────────────────────────
//
// 这一批是 2026-06-29 补加的：start/stop/reboot/reset/terminal/status/channels/
// set-model/add-model/switch-primary-model/del-model/set-channel/del-channel
// 之前都没有 source guard，本地 agent 实例（instance_id="local-xxx-yyy"）会落到
// CVM/TAT API 才被参数校验拒绝，错误体不友好。
//
// 入口处统一用 rejectLocalOrWrite 拦截，返 400 MsgLocalInstanceUnsupportedOp。
// 测试只断言「入口被拒绝」，不进入业务逻辑。

// adminInstanceIDForm 是管控端 13 个 handler 的通用 form 构造器
func adminInstanceIDForm(t *testing.T, method, path string, instanceID uint) *http.Request {
	t.Helper()
	form := "id=" + strconv.FormatUint(uint64(instanceID), 10)
	return adminFormReq(method, path, form)
}

func TestHandleAdminStartInstance_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "as-local", "local-codebuddy-001")
	req := adminInstanceIDForm(t, http.MethodPost, "/admin/instances/start", inst.ID)
	w := httptest.NewRecorder()
	HandleAdminStartInstance(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

func TestHandleAdminStopInstance_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "asp-local", "local-codebuddy-001")
	req := adminInstanceIDForm(t, http.MethodPost, "/admin/instances/stop", inst.ID)
	w := httptest.NewRecorder()
	HandleAdminStopInstance(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

func TestHandleAdminRebootInstance_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "arb-local", "local-codebuddy-001")
	req := adminInstanceIDForm(t, http.MethodPost, "/admin/instances/reboot", inst.ID)
	w := httptest.NewRecorder()
	HandleAdminRebootInstance(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

func TestHandleAdminResetInstance_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "ars-local", "local-codebuddy-001")
	req := adminInstanceIDForm(t, http.MethodPost, "/admin/instances/reset", inst.ID)
	w := httptest.NewRecorder()
	HandleAdminResetInstance(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

func TestHandleAdminInstanceTerminal_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "att-local", "local-codebuddy-001")
	req := adminInstanceIDForm(t, http.MethodPost, "/admin/instances/terminal-url", inst.ID)
	w := httptest.NewRecorder()
	HandleAdminInstanceTerminal(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

func TestHandleAdminInstanceStatus_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "ast-local", "local-codebuddy-001")
	req := adminInstanceIDForm(t, http.MethodGet,
		"/admin/instances/status?id="+strconv.FormatUint(uint64(inst.ID), 10), inst.ID)
	w := httptest.NewRecorder()
	HandleAdminInstanceStatus(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

func TestHandleAdminInstanceChannels_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "ach-local", "local-codebuddy-001")
	req := adminInstanceIDForm(t, http.MethodGet,
		"/admin/instances/channels?id="+strconv.FormatUint(uint64(inst.ID), 10), inst.ID)
	w := httptest.NewRecorder()
	HandleAdminInstanceChannels(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
}

func TestHandleAdminSetModel_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "asm-local", "local-codebuddy-001")
	req := adminInstanceIDForm(t, http.MethodPost, "/admin/instances/set-model", inst.ID)
	w := httptest.NewRecorder()
	HandleAdminSetModel(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
	if !strings.Contains(w.Body.String(), "不支持此操作") {
		t.Errorf("期望错误体包含「不支持此操作」，实际=%s", w.Body.String())
	}
}

func TestHandleAdminAddModel_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "aam-local", "local-codebuddy-001")
	req := adminInstanceIDForm(t, http.MethodPost, "/admin/instances/add-model", inst.ID)
	w := httptest.NewRecorder()
	HandleAdminAddModel(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
	if !strings.Contains(w.Body.String(), "不支持此操作") {
		t.Errorf("期望错误体包含「不支持此操作」，实际=%s", w.Body.String())
	}
}

func TestHandleAdminSwitchPrimaryModel_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "asp2-local", "local-codebuddy-001")
	req := adminInstanceIDForm(t, http.MethodPost, "/admin/instances/switch-primary-model", inst.ID)
	w := httptest.NewRecorder()
	HandleAdminSwitchPrimaryModel(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
	if !strings.Contains(w.Body.String(), "不支持此操作") {
		t.Errorf("期望错误体包含「不支持此操作」，实际=%s", w.Body.String())
	}
}

func TestHandleAdminDelModel_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "adm-local", "local-codebuddy-001")
	req := adminInstanceIDForm(t, http.MethodPost, "/admin/instances/del-model", inst.ID)
	w := httptest.NewRecorder()
	HandleAdminDelModel(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
	if !strings.Contains(w.Body.String(), "不支持此操作") {
		t.Errorf("期望错误体包含「不支持此操作」，实际=%s", w.Body.String())
	}
}

func TestHandleAdminSetChannel_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "asc-local", "local-codebuddy-001")
	req := adminInstanceIDForm(t, http.MethodPost, "/admin/instances/set-channel", inst.ID)
	w := httptest.NewRecorder()
	HandleAdminSetChannel(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
	if !strings.Contains(w.Body.String(), "不支持此操作") {
		t.Errorf("期望错误体包含「不支持此操作」，实际=%s", w.Body.String())
	}
}

func TestHandleAdminDelChannel_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "adc-local", "local-codebuddy-001")
	req := adminInstanceIDForm(t, http.MethodPost, "/admin/instances/del-channel", inst.ID)
	w := httptest.NewRecorder()
	HandleAdminDelChannel(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
	if !strings.Contains(w.Body.String(), "不支持此操作") {
		t.Errorf("期望错误体包含「不支持此操作」，实际=%s", w.Body.String())
	}
}

// TestHandleAdminRestartGateway_LocalInstance 验证 restart-gateway 接口拒绝本地 agent 实例。
// 本地 agent 不走 CVM/TAT，restart_gateway 脚本无法下发，应在 adminRestartGatewayOne
// 入口被 source guard 拒绝，返回 400 + 「不支持此操作」。
func TestHandleAdminRestartGateway_LocalInstance(t *testing.T) {
	initBatchUpgradeTestDB(t)
	_, inst := localRejectInst(t, "arg-local", "local-codebuddy-001")
	req := adminInstanceIDForm(t, http.MethodPost, "/admin/instances/restart-gateway", inst.ID)
	w := httptest.NewRecorder()
	HandleAdminRestartGateway(w, req)
	assertLocalRejected(t, w, http.StatusBadRequest)
	if !strings.Contains(w.Body.String(), "不支持此操作") {
		t.Errorf("期望错误体包含「不支持此操作」，实际=%s", w.Body.String())
	}
}
