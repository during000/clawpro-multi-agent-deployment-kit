package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"
)

// 5 个用户端写操作 handler（reboot/reset/delete/upgrade/upgrade-retry）
// 在收到龙虾医生节点 ID 时应返回 400 + "龙虾医生节点不允许该操作"。

// createDoctorNodeForUser 在 DB 中创建一个属于 user 的龙虾医生节点。
func createDoctorNodeForUser(t *testing.T, user *model.User, name string) *model.Instance {
	t.Helper()
	doctor := &model.Instance{
		Name:         name,
		InstanceId:   "ins-" + name,
		UserID:       user.ID,
		AgentType:    model.AgentTypeOpenClaw,
		IsDoctorNode: true,
		LastCVMState: "RUNNING",
	}
	if err := model.DB(context.Background()).Create(doctor).Error; err != nil {
		t.Fatalf("创建龙虾医生节点失败: %v", err)
	}
	return doctor
}

// assertDoctorRejected 验证响应是 400 + 包含"龙虾医生"关键字。
func assertDoctorRejected(t *testing.T, rr *httptest.ResponseRecorder, handlerName string) {
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

// TestHandleDeleteInstance_RejectsDoctorNode delete 应拒绝龙虾医生节点。
func TestHandleDeleteInstance_RejectsDoctorNode(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	doctor := createDoctorNodeForUser(t, user, "doctor-del")

	form := fmt.Sprintf("id=%d", doctor.ID)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/delete", "u1", form)
	rr := httptest.NewRecorder()

	HandleDeleteInstance(rr, req)

	assertDoctorRejected(t, rr, "DeleteInstance")
}

// TestHandleRebootInstance_RejectsDoctorNode reboot 应拒绝龙虾医生节点。
func TestHandleRebootInstance_RejectsDoctorNode(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	doctor := createDoctorNodeForUser(t, user, "doctor-reboot")

	form := fmt.Sprintf("id=%d", doctor.ID)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reboot", "u1", form)
	rr := httptest.NewRecorder()

	handleRebootInstance(rr, req, testCVMFetcher)

	assertDoctorRejected(t, rr, "RebootInstance")
}

// TestHandleResetInstance_RejectsDoctorNode reset 应拒绝龙虾医生节点。
func TestHandleResetInstance_RejectsDoctorNode(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	doctor := createDoctorNodeForUser(t, user, "doctor-reset")

	form := fmt.Sprintf("id=%d", doctor.ID)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/reset", "u1", form)
	rr := httptest.NewRecorder()

	handleResetInstance(rr, req, testCVMFetcher)

	assertDoctorRejected(t, rr, "ResetInstance")
}

// TestHandleUpgrade_RejectsDoctorNode upgrade 应拒绝龙虾医生节点。
func TestHandleUpgrade_RejectsDoctorNode(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	doctor := createDoctorNodeForUser(t, user, "doctor-upgrade")

	form := fmt.Sprintf("id=%d", doctor.ID)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/upgrade", "u1", form)
	rr := httptest.NewRecorder()

	handleUpgrade(rr, req, testCVMFetcher)

	assertDoctorRejected(t, rr, "Upgrade")
}

// TestHandleUpgradeRetry_RejectsDoctorNode upgrade-retry 应拒绝龙虾医生节点。
func TestHandleUpgradeRetry_RejectsDoctorNode(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	t.Cleanup(cleanup)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	doctor := createDoctorNodeForUser(t, user, "doctor-upgrade-retry")

	form := fmt.Sprintf("id=%d", doctor.ID)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/upgrade/retry", "u1", form)
	rr := httptest.NewRecorder()


	HandleUpgradeRetry(rr, req)

	assertDoctorRejected(t, rr, "UpgradeRetry")
}
