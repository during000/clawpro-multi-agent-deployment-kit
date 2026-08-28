package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"hatchery/model"
)

// doAckReq 构造带登录态的 POST /local-agent/commands/ack 请求并直接调 handler。
func doAckReq(t *testing.T, cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/local-agent/commands/ack", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, req)
	return rr
}

func TestHandleLocalAgentAck_UninstallTeamai_Success_SoftDeletesInstance(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	_, instID := seedLocalUserAndInstance(t, "alice", "codebuddy")
	cookie := loginCookie(t, "alice")
	ctx := context.Background()

	// 关联数据：local_instance_skills + local_instance_infos（ack 后应一并软删）
	model.DB(ctx).Create(&model.LocalInstanceSkill{InstanceID: instID, Slug: "skill-x", InstallStatus: model.LocalSkillInstallStatusDistributed})
	model.DB(ctx).Create(&model.LocalInstanceInfo{InstanceID: instID, HostName: "alice-pc"})

	task := model.LocalAgentTask{
		Identifier:  "test-tenant",
		InstanceID:  instID,
		InstanceCID: "local-alice-001",
		Type:        model.LocalAgentTaskTypeUninstallTeamai,
		Cmd:         "teamai uninstall --force --agent codebuddy",
		Status:      model.LocalAgentTaskStatusPending,
	}
	if err := model.DB(ctx).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	rr := doAckReq(t, cookie, `{"id":`+itoaSafe(task.ID)+`,"type":"uninstall_teamai","status":"success"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		RecordID uint   `json:"record_id"`
		Status   string `json:"status"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Status != model.LocalAgentTaskStatusSuccess {
		t.Fatalf("task status=%s", resp.Status)
	}

	// 实例应被软删（gorm 默认查询查不到）
	var inst model.Instance
	err := model.DB(ctx).First(&inst, instID).Error
	if !isRecordNotFound(err) {
		t.Fatalf("instance 应被软删，但查到了: %+v err=%v", inst, err)
	}
	// Unscoped 能查到且 deleted_at 非空
	var softInst model.Instance
	if err := model.DB(ctx).Unscoped().First(&softInst, instID).Error; err != nil {
		t.Fatalf("unscoped 应查到软删实例: %v", err)
	}
	if !softInst.DeletedAt.Valid {
		t.Fatalf("deleted_at 应为非空（已软删）")
	}
	// local_instance_skills 应一并软删
	var skillCnt int64
	model.DB(ctx).Model(&model.LocalInstanceSkill{}).Where("instance_id = ?", instID).Count(&skillCnt)
	if skillCnt != 0 {
		t.Fatalf("local_instance_skills 应被软删，剩余 %d", skillCnt)
	}
	// local_instance_infos 应一并软删
	var infoCnt int64
	model.DB(ctx).Model(&model.LocalInstanceInfo{}).Where("instance_id = ?", instID).Count(&infoCnt)
	if infoCnt != 0 {
		t.Fatalf("local_instance_infos 应被软删，剩余 %d", infoCnt)
	}
}

// TestHandleLocalAgentRemove_SetsDestroyingOnTaskCreate 验证下发卸载任务时实例进入 destroying 中间态。
func TestHandleLocalAgentRemove_SetsDestroyingOnTaskCreate(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	_, instID := seedLocalUserAndInstance(t, "alice", "codebuddy")
	cookie := loginCookie(t, "alice")
	ctx := context.Background()

	rr := doRemoveReq(t, "/local-agent/remove", cookie, `{"instance_id": `+strconv.Itoa(int(instID))+`}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	// 实例应进入 destroying 中间态（current_operation=delete）
	var inst model.Instance
	if err := model.DB(ctx).First(&inst, instID).Error; err != nil {
		t.Fatalf("查实例: %v", err)
	}
	if inst.CurrentOperation != model.LocalAgentOpUninstall {
		t.Fatalf("下发任务后 current_operation 应为 %q，实际 %q", model.LocalAgentOpUninstall, inst.CurrentOperation)
	}
	if inst.LastKnownStatus != model.StatusDestroying {
		t.Fatalf("下发任务后 last_known_status 应为 %q（前端展示销毁中），实际 %q", model.StatusDestroying, inst.LastKnownStatus)
	}
}

// TestHandleLocalAgentAck_UninstallTeamai_Failed_RestoresRunning 验证 ack 失败后实例退出卸载中状态。
func TestHandleLocalAgentAck_UninstallTeamai_Failed_RestoresRunning(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	_, instID := seedLocalUserAndInstance(t, "alice", "codebuddy")
	cookie := loginCookie(t, "alice")
	ctx := context.Background()

	// 先下发任务（实例进入卸载中）
	doRemoveReq(t, "/local-agent/remove", cookie, `{"instance_id": `+strconv.Itoa(int(instID))+`}`)
	var pre model.Instance
	model.DB(ctx).First(&pre, instID)
	if pre.LastKnownStatus != model.StatusDestroying {
		t.Fatalf("下发后应为 destroying，实际 %q", pre.LastKnownStatus)
	}

	task := model.LocalAgentTask{
		Identifier:  "test-tenant",
		InstanceID:  instID,
		InstanceCID: "local-alice-001",
		Type:        model.LocalAgentTaskTypeUninstallTeamai,
		Cmd:         "teamai uninstall --force --agent codebuddy",
		Status:      model.LocalAgentTaskStatusPending,
	}
	model.DB(ctx).Create(&task)

	rr := doAckReq(t, cookie, `{"id":`+itoaSafe(task.ID)+`,"type":"uninstall_teamai","status":"failed","error":"timeout"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	// 失败后：实例退出卸载中，last_known_status 回到 running，current_operation 清空
	var inst model.Instance
	model.DB(ctx).First(&inst, instID)
	if inst.LastKnownStatus != model.StatusRunning {
		t.Fatalf("失败后 last_known_status 应为 running，实际 %q", inst.LastKnownStatus)
	}
	if inst.CurrentOperation != "" {
		t.Fatalf("失败后 current_operation 应清空，实际 %q", inst.CurrentOperation)
	}
	// 实例不应被删
	if err := model.DB(ctx).First(&inst, instID).Error; err != nil {
		t.Fatalf("实例不应被删: %v", err)
	}
}

func TestHandleLocalAgentAck_UninstallTeamai_Failed_KeepsInstance(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	_, instID := seedLocalUserAndInstance(t, "alice", "codebuddy")
	cookie := loginCookie(t, "alice")
	ctx := context.Background()

	task := model.LocalAgentTask{
		Identifier:  "test-tenant",
		InstanceID:  instID,
		InstanceCID: "local-alice-001",
		Type:        model.LocalAgentTaskTypeUninstallTeamai,
		Cmd:         "teamai uninstall --force --agent codebuddy",
		Status:      model.LocalAgentTaskStatusPending,
	}
	model.DB(ctx).Create(&task)

	rr := doAckReq(t, cookie, `{"id":`+itoaSafe(task.ID)+`,"type":"uninstall_teamai","status":"failed","error":"timeout"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Status string `json:"status"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Status != model.LocalAgentTaskStatusFailed {
		t.Fatalf("task status=%s", resp.Status)
	}
	// 实例不应被删
	var inst model.Instance
	if err := model.DB(ctx).First(&inst, instID).Error; err != nil {
		t.Fatalf("instance 不应被删: %v", err)
	}
	// 错误应被记录
	var reloaded model.LocalAgentTask
	model.DB(ctx).First(&reloaded, task.ID)
	if reloaded.Error != "timeout" {
		t.Fatalf("error 未记录: %q", reloaded.Error)
	}
}

func TestHandleLocalAgentAck_UninstallTeamai_ReactivateViaReport(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	localAgentID := "0123456789abcdef"
	instCID := formatLocalInstanceID("codebuddy", localAgentID)
	user, inst := seedLocalUserAndInstanceWithCID(t, "alice", "codebuddy", instCID)
	ctx := context.Background()

	// 先软删实例（模拟 ack success）
	model.DB(ctx).Delete(&inst)

	// 重新 report 同一 local_agent_id → 应恢复 deleted_at（重新激活）
	reportBody := `{"agent_type":"codebuddy","local_agent_id":"` + localAgentID + `","host_name":"alice-pc","status":"running"}`
	req := httptest.NewRequest(http.MethodPost, "/local-agent/report", strings.NewReader(reportBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", loginCookie(t, "alice"))
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("report status=%d body=%s", rr.Code, rr.Body.String())
	}

	// 实例应恢复（gorm 默认查询可查到，deleted_at 为空）
	var reactivated model.Instance
	if err := model.DB(ctx).First(&reactivated, inst.ID).Error; err != nil {
		t.Fatalf("report 后应重新激活实例: %v", err)
	}
	if reactivated.DeletedAt.Valid {
		t.Fatalf("重新激活后 deleted_at 应被清空")
	}
	// name 仅在「为空或等于 instance_id」时才被 host_name 覆盖；本例 name=local-alice 不变，
	// 验证重点是 deleted_at 已恢复（重新激活），不要求 name 变更。
	_ = user
}

func TestHandleLocalAgentAck_UninstallTeamai_ForbiddenForOtherUser(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	// bob 的任务，alice 尝试 ack
	seedLocalUser(t, "alice")
	_, bobInst := seedLocalUserAndInstance(t, "bob", "codebuddy")
	ctx := context.Background()
	task := model.LocalAgentTask{
		Identifier:  "test-tenant",
		InstanceID:  bobInst,
		InstanceCID: "local-bob-001",
		Type:        model.LocalAgentTaskTypeUninstallTeamai,
		Cmd:         "teamai uninstall --force --agent codebuddy",
		Status:      model.LocalAgentTaskStatusPending,
	}
	model.DB(ctx).Create(&task)

	rr := doAckReq(t, loginCookie(t, "alice"), `{"id":`+itoaSafe(task.ID)+`,"type":"uninstall_teamai","status":"success"}`)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404 body=%s", rr.Code, rr.Body.String())
	}
}

func itoaSafe(u uint) string {
	if u == 0 {
		return "0"
	}
	var b []byte
	for u > 0 {
		b = append([]byte{byte('0' + u%10)}, b...)
		u /= 10
	}
	return string(b)
}

func isRecordNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "record not found")
}

var _ = time.Now

// TestHandleLocalAgentReport_KeepsDestroyingDuringUninstall 验证卸载中实例的 report 心跳不覆盖 destroying。
func TestHandleLocalAgentReport_KeepsDestroyingDuringUninstall(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	localAgentID := "0123456789abcdef"
	instCID := formatLocalInstanceID("codebuddy", localAgentID)
	seedLocalUserAndInstanceWithCID(t, "alice", "codebuddy", instCID)
	cookie := loginCookie(t, "alice")
	ctx := context.Background()

	// 下发卸载任务：实例进入 destroying
	doRemoveReq(t, "/local-agent/remove", cookie, `{"instance_id": `+itoaSafe(instanceIDFromCID(t, instCID))+`}`)

	// 模拟 reporter 心跳（report 仍在跑）
	reportBody := `{"agent_type":"codebuddy","local_agent_id":"` + localAgentID + `","host_name":"alice-pc","status":"running"}`
	req := httptest.NewRequest(http.MethodPost, "/local-agent/report", strings.NewReader(reportBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("report status=%d body=%s", rr.Code, rr.Body.String())
	}

	var inst model.Instance
	model.DB(ctx).First(&inst, instanceIDFromCID(t, instCID))
	if inst.LastKnownStatus != model.StatusDestroying {
		t.Fatalf("卸载中 report 不应覆盖 destroying，实际 %q", inst.LastKnownStatus)
	}
}

func instanceIDFromCID(t *testing.T, instCID string) uint {
	t.Helper()
	ctx := context.Background()
	var inst model.Instance
	if err := model.DB(ctx).Where("instance_id = ?", instCID).First(&inst).Error; err != nil {
		t.Fatalf("查实例 by cid: %v", err)
	}
	return inst.ID
}
