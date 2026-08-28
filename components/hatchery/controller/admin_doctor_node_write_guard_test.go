package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"hatchery/model"
)

// 9 个管控端写操作 handler 在收到龙虾医生节点 ID 时应返回错误：
// terminal-url / stop / start / reboot / reset / delete /
// refresh-version / batch-upgrade / detect-install
//
// 龙虾医生节点会出现在 admin 实例列表里（可见），但所有写操作必须被拒绝，
// 避免 admin 误中断诊断会话。

// createAdminDoctorNode 在 DB 中创建一个龙虾医生节点。
func createAdminDoctorNode(t *testing.T, name string) *model.Instance {
	t.Helper()
	doctor := &model.Instance{
		Name:         name,
		InstanceId:   "ins-" + name,
		UserID:       1,
		AgentType:    model.AgentTypeOpenClaw,
		IsDoctorNode: true,
		LastCVMState: "RUNNING",
	}
	if err := model.DB(context.Background()).Create(doctor).Error; err != nil {
		t.Fatalf("创建龙虾医生节点失败: %v", err)
	}
	return doctor
}

// assertAdminDoctorRejected 验证响应是 400 + 包含"龙虾医生"关键字。
func assertAdminDoctorRejected(t *testing.T, rr *httptest.ResponseRecorder, handlerName string) {
	t.Helper()
	if rr.Code != http.StatusBadRequest {
		t.Errorf("[%s] 龙虾医生节点应返回 400，实际=%d body=%s",
			handlerName, rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "龙虾医生") {
		t.Errorf("[%s] 错误信息应包含\"龙虾医生\"，实际=%s",
			handlerName, rr.Body.String())
	}
}

// TestHandleAdminInstanceTerminal_RejectsDoctorNode terminal-url 应拒绝龙虾医生节点。
func TestHandleAdminInstanceTerminal_RejectsDoctorNode(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()
	doctor := createAdminDoctorNode(t, "doctor-terminal")

	req := adminTokenReq(http.MethodPost, "/admin/instances/terminal-url",
		fmt.Sprintf("id=%d", doctor.ID))
	rr := httptest.NewRecorder()
	handleAdminInstanceTerminal(rr, req, testCVMFetcher)

	assertAdminDoctorRejected(t, rr, "AdminInstanceTerminal")
}

// TestHandleAdminStopInstance_RejectsDoctorNode stop 应拒绝龙虾医生节点。
func TestHandleAdminStopInstance_RejectsDoctorNode(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()
	doctor := createAdminDoctorNode(t, "doctor-stop")

	req := adminTokenReq(http.MethodPost, "/admin/instances/stop",
		fmt.Sprintf("id=%d", doctor.ID))
	rr := httptest.NewRecorder()
	handleAdminStopInstance(rr, req, testCVMFetcher)

	assertAdminDoctorRejected(t, rr, "AdminStopInstance")
}

// TestHandleAdminStartInstance_RejectsDoctorNode start 应拒绝龙虾医生节点。
func TestHandleAdminStartInstance_RejectsDoctorNode(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()
	doctor := createAdminDoctorNode(t, "doctor-start")

	req := adminTokenReq(http.MethodPost, "/admin/instances/start",
		fmt.Sprintf("id=%d", doctor.ID))
	rr := httptest.NewRecorder()
	handleAdminStartInstance(rr, req, testCVMFetcher)

	assertAdminDoctorRejected(t, rr, "AdminStartInstance")
}

// TestHandleAdminRebootInstance_RejectsDoctorNode reboot 应拒绝龙虾医生节点。
func TestHandleAdminRebootInstance_RejectsDoctorNode(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()
	doctor := createAdminDoctorNode(t, "doctor-reboot")

	req := adminTokenReq(http.MethodPost, "/admin/instances/reboot",
		fmt.Sprintf("id=%d", doctor.ID))
	rr := httptest.NewRecorder()
	handleAdminRebootInstance(rr, req, testCVMFetcher)

	assertAdminDoctorRejected(t, rr, "AdminRebootInstance")
}

// TestHandleAdminResetInstance_RejectsDoctorNode reset 应拒绝龙虾医生节点。
func TestHandleAdminResetInstance_RejectsDoctorNode(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()
	doctor := createAdminDoctorNode(t, "doctor-reset")

	req := adminTokenReq(http.MethodPost, "/admin/instances/reset",
		fmt.Sprintf("id=%d", doctor.ID))
	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, testCVMFetcher)

	assertAdminDoctorRejected(t, rr, "AdminResetInstance")
}

// TestHandleAdminDeleteInstance_RejectsDoctorNode delete 应拒绝龙虾医生节点。
func TestHandleAdminDeleteInstance_RejectsDoctorNode(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()
	doctor := createAdminDoctorNode(t, "doctor-delete")

	req := adminTokenReq(http.MethodPost, "/admin/instances/delete",
		fmt.Sprintf("id=%d", doctor.ID))
	rr := httptest.NewRecorder()
	handleAdminDeleteInstance(rr, req, testCVMFetcher)

	assertAdminDoctorRejected(t, rr, "AdminDeleteInstance")
}

// TestHandleAdminRefreshInstanceVersion_RejectsDoctorNode refresh-version 应拒绝龙虾医生节点。
func TestHandleAdminRefreshInstanceVersion_RejectsDoctorNode(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()
	doctor := createAdminDoctorNode(t, "doctor-refresh")

	req := adminTokenReq(http.MethodPost, "/admin/instances/refresh-version",
		fmt.Sprintf("id=%d", doctor.ID))
	rr := httptest.NewRecorder()
	HandleAdminRefreshInstanceVersion(rr, req)

	assertAdminDoctorRejected(t, rr, "AdminRefreshInstanceVersion")
}

// TestHandleAdminBatchUpgrade_RejectsDoctorNode batch-upgrade 在列表里出现龙虾医生节点应整体拒绝。
func TestHandleAdminBatchUpgrade_RejectsDoctorNode(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	// 创建一个普通实例 + 一个龙虾医生节点
	normal := &model.Instance{
		Name: "normal", InstanceId: "ins-normal",
		UserID: 1, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(normal)
	doctor := createAdminDoctorNode(t, "doctor-batch-upgrade")

	body, _ := json.Marshal(map[string]interface{}{
		"ids": []uint{normal.ID, doctor.ID},
	})
	req := adminTokenReqJSON(http.MethodPost, "/admin/instances/batch-upgrade", body)
	rr := httptest.NewRecorder()
	HandleAdminBatchUpgrade(rr, req)

	assertAdminDoctorRejected(t, rr, "AdminBatchUpgrade")
	// 错误信息应包含具体 ID
	if !strings.Contains(rr.Body.String(), fmt.Sprintf("%d", doctor.ID)) {
		t.Errorf("[AdminBatchUpgrade] 错误信息应包含龙虾医生节点 ID %d，实际=%s",
			doctor.ID, rr.Body.String())
	}
}

// TestHandleAdminDetectInstall_RejectsDoctorNode detect-install 在列表里出现龙虾医生节点应整体拒绝。
func TestHandleAdminDetectInstall_RejectsDoctorNode(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	normal := &model.Instance{
		Name: "normal", InstanceId: "ins-normal",
		UserID: 1, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(normal)
	doctor := createAdminDoctorNode(t, "doctor-detect-install")

	body, _ := json.Marshal(map[string]interface{}{
		"ids": []uint{normal.ID, doctor.ID},
	})
	req := adminTokenReqJSON(http.MethodPost, "/admin/instances/detect-install", body)
	rr := httptest.NewRecorder()
	HandleAdminDetectInstall(rr, req)

	assertAdminDoctorRejected(t, rr, "AdminDetectInstall")
	if !strings.Contains(rr.Body.String(), fmt.Sprintf("%d", doctor.ID)) {
		t.Errorf("[AdminDetectInstall] 错误信息应包含龙虾医生节点 ID %d，实际=%s",
			doctor.ID, rr.Body.String())
	}
}

// TestHandleAdminBatchUpgrade_AllNormalNotRejected 全部为普通实例时不应被本守卫拒绝
// （后续可能因其他原因失败，但不应是 400 "龙虾医生"）。
func TestHandleAdminBatchUpgrade_AllNormalNotRejected(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	normal := &model.Instance{
		Name: "normal", InstanceId: "",
		UserID: 1, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(normal)

	body, _ := json.Marshal(map[string]interface{}{
		"ids": []uint{normal.ID},
	})
	req := adminTokenReqJSON(http.MethodPost, "/admin/instances/batch-upgrade", body)
	rr := httptest.NewRecorder()
	HandleAdminBatchUpgrade(rr, req)

	// 不应该是因为"龙虾医生"被拒绝
	if strings.Contains(rr.Body.String(), "龙虾医生") {
		t.Errorf("[AdminBatchUpgrade] 全普通实例不应触发龙虾医生拒绝，实际=%s",
			rr.Body.String())
	}
}

// 防止 url import 未使用（adminTokenReq 已用，但保留以备 form 编码扩展）
var _ = url.Values{}
