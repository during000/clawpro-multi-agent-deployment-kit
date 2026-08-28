package task

import (
	"context"
	"strings"
	"testing"
	"time"

	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// 验证清理任务的写操作会留下审计日志，且 username 写"龙虾医生服务"。

// initAuditedCleanupTestDB 在 cleanup 测试 DB 基础上额外迁移 AuditLog 表。
func initAuditedCleanupTestDB(t *testing.T) func() {
	t.Helper()
	cleanup := initCleanupTestDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.AuditLog{}); err != nil {
		cleanup()
		t.Fatalf("migrate AuditLog: %v", err)
	}
	return cleanup
}

// findDoctorAuditLogs 查询所有 username="龙虾医生服务" 的审计日志。
func findDoctorAuditLogs(t *testing.T, action string) []model.AuditLog {
	t.Helper()
	var logs []model.AuditLog
	q := model.DB(context.Background()).Where("username = ?", i18n.T(context.Background(), i18n.MsgDoctorServiceUsername))
	if action != "" {
		q = q.Where("action = ?", action)
	}
	if err := q.Find(&logs).Error; err != nil {
		t.Fatalf("查询审计日志失败: %v", err)
	}
	return logs
}

// TestEndDoctorSession_WritesAuditLog 验证 endDoctorSession 调用后产生
// 一条 username="龙虾医生服务" 的审计日志。
func TestEndDoctorSession_WritesAuditLog(t *testing.T) {
	cleanup := initAuditedCleanupTestDB(t)
	defer cleanup()

	session := &model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		Status: model.DoctorStatusActive,
	}
	if err := model.DB(context.Background()).Create(session).Error; err != nil {
		t.Fatalf("创建 session 失败: %v", err)
	}

	endDoctorSession(context.Background(), session)

	// 验证 session.status 被更新
	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusEnding {
		t.Errorf("session status 应为 ending，实际=%s", updated.Status)
	}

	// 验证审计日志
	logs := findDoctorAuditLogs(t, "doctor_session_end_timeout")
	if len(logs) != 1 {
		t.Fatalf("应产生 1 条龙虾医生服务审计日志，实际=%d", len(logs))
	}
	got := logs[0]
	if got.UserID != session.UserID {
		t.Errorf("UserID 应为 %d，实际=%d", session.UserID, got.UserID)
	}
	if got.Resource != "doctor_session" {
		t.Errorf("Resource 应为 doctor_session，实际=%s", got.Resource)
	}
	if !strings.Contains(got.ResourceID, "1") {
		t.Errorf("ResourceID 应包含 session ID 1，实际=%s", got.ResourceID)
	}
	if got.Status != "success" {
		t.Errorf("Status 应为 success，实际=%s", got.Status)
	}
	if got.Username != i18n.T(context.Background(), i18n.MsgDoctorServiceUsername) {
		t.Errorf("Username 应为 %s，实际=%s", i18n.T(context.Background(), i18n.MsgDoctorServiceUsername), got.Username)
	}
}

// TestDestroyDoctorNode_WritesAuditLog 验证 destroyDoctorNode 删除实例
// 时产生 username="龙虾医生服务" 的审计日志。
//
// 这里测试无 CVM 实例的简化路径（InstanceId=""），跳过腾讯云 CVM 调用。
func TestDestroyDoctorNode_WritesAuditLog(t *testing.T) {
	cleanup := initAuditedCleanupTestDB(t)
	defer cleanup()

	doctor := &model.Instance{
		Model: gorm.Model{ID: 200},
		Name:  "doctor-destroy",
		// InstanceId 留空 → destroyDoctorNode 跳过 CVM 调用直接删记录
		InstanceId:   "",
		UserID:       1,
		IsDoctorNode: true,
	}
	if err := model.DB(context.Background()).Create(doctor).Error; err != nil {
		t.Fatalf("创建龙虾医生节点失败: %v", err)
	}

	log := controller.Logger(context.Background())
	destroyDoctorNode(context.Background(), doctor.ID, log)

	// 验证实例记录被软删
	var inst model.Instance
	err := model.DB(context.Background()).First(&inst, doctor.ID).Error
	if err == nil {
		t.Errorf("实例记录应被删除，实际仍存在")
	}

	// 验证审计日志
	logs := findDoctorAuditLogs(t, "doctor_node_destroy")
	if len(logs) != 1 {
		t.Fatalf("应产生 1 条审计日志，实际=%d", len(logs))
	}
	got := logs[0]
	if got.UserID != 1 {
		t.Errorf("UserID 应为 1（即原 instance.UserID），实际=%d", got.UserID)
	}
	if got.Resource != "instance" {
		t.Errorf("Resource 应为 instance，实际=%s", got.Resource)
	}
	if got.Status != "success" {
		t.Errorf("Status 应为 success，实际=%s", got.Status)
	}
	if got.Username != i18n.T(context.Background(), i18n.MsgDoctorServiceUsername) {
		t.Errorf("Username 应为 %s，实际=%s", i18n.T(context.Background(), i18n.MsgDoctorServiceUsername), got.Username)
	}
	// startedAt 应该早于现在但不超过 5 秒
	if time.Since(got.StartedAt) > 5*time.Second {
		t.Errorf("StartedAt 时间不合理: %v", got.StartedAt)
	}
}
