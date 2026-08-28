package task

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	hcommon "hatchery/common"
	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"

	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

// ─── processDoctorEnding 测试 ────────────────────────────────────────────────

func TestProcessDoctorEnding_有ending会话_无回滚(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	snap := saveTaskFns()
	defer snap.restore()

	cleanedUp := false
	controller.CleanupDoctorSessionFn =
		func(_ context.Context, s *model.DoctorSession) {
			cleanedUp = true
			model.DB(context.Background()).Model(s).
				Update("status", model.DoctorStatusEnded)
		}

	session := model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 1,
		Status:           model.DoctorStatusEnding,
		HasSnapshot:      false,
	}
	model.DB(context.Background()).Create(&session)

	processDoctorEnding(context.Background())

	if !cleanedUp {
		t.Error("CleanupDoctorSession 未被调用")
	}

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusEnded {
		t.Errorf("status = %s, want ended", updated.Status)
	}
}

func TestProcessDoctorEnding_无ending会话(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	snap := saveTaskFns()
	defer snap.restore()

	cleanedUp := false
	controller.CleanupDoctorSessionFn =
		func(_ context.Context, s *model.DoctorSession) {
			cleanedUp = true
		}

	// 只有 active 会话，无 ending
	session := model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 1,
		Status:           model.DoctorStatusActive,
	}
	model.DB(context.Background()).Create(&session)

	processDoctorEnding(context.Background())

	if cleanedUp {
		t.Error("不应该有 ending 会话被清理")
	}
}

func TestProcessDoctorEnding_回滚请求(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	snap := saveTaskFns()
	defer snap.restore()

	controller.CleanupDoctorSessionFn =
		func(_ context.Context, s *model.DoctorSession) {
			model.DB(context.Background()).Model(s).
				Update("status", model.DoctorStatusEnded)
		}

	session := model.DoctorSession{
		UserID:            1,
		TargetInstanceID:  1,
		Status:            model.DoctorStatusEnding,
		RollbackRequested: true,
		HasSnapshot:       true,
		SnapshotFileKey:   "doctor/snapshots/test.tar.gz",
	}
	model.DB(context.Background()).Create(&session)

	// 回滚会尝试 BuildCommonSMHDownloadURL 和 RunScript，
	// 这里会失败（因为没有真实 SMH），但不影响清理流程
	processDoctorEnding(context.Background())

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusEnded {
		t.Errorf("status = %s, want ended", updated.Status)
	}
}

// ─── processDoctorActivate 补充测试 ─────────────────────────────────────────

func TestActivateDoctorSessions_DoctorInstance不存在则标记失败(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	nonExistID := uint(9999)
	session := model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 1,
		DoctorInstanceID: &nonExistID,
		Status:           model.DoctorStatusCreating,
	}
	model.DB(context.Background()).Create(&session)

	processDoctorActivate(context.Background())

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusFailed {
		t.Errorf("status = %s, want failed (实例不存在应标记失败)",
			updated.Status)
	}
}

// ─── destroyDoctorNode 测试 ─────────────────────────────────────────────────

func TestDestroyDoctorNode_实例不存在(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	log := slog.Default()
	// 实例 ID 9999 不存在，不应 panic
	destroyDoctorNode(context.Background(), 9999, log)
}

func TestDestroyDoctorNode_CVM客户端创建失败不删记录(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	log := slog.Default()

	doctorInst := model.Instance{
		Model:        gorm.Model{ID: 500},
		Name:         "doctor-destroy",
		InstanceId:   "ins-doctor-destroy",
		UserID:       1,
		IsDoctorNode: true,
	}
	model.DB(context.Background()).Create(&doctorInst)

	origNewCVM := controller.NewCVMClient
	controller.NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return nil, hcommon.I18nRichError(fmt.Errorf("mock CVM client error"), i18n.MsgOperationFailed)
	}
	defer func() { controller.NewCVMClient = origNewCVM }()

	destroyDoctorNode(context.Background(), 500, log)

	// 实例记录应该仍然存在（销毁失败不删除）
	var inst model.Instance
	err := model.DB(context.Background()).First(&inst, 500).Error
	if err != nil {
		t.Error("CVM 客户端创建失败时，实例记录不应被删除")
	}
}

func TestDestroyDoctorNode_无CVMInstanceId直接删记录(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	log := slog.Default()

	doctorInst := model.Instance{
		Model:        gorm.Model{ID: 501},
		Name:         "doctor-no-cvm",
		InstanceId:   "", // 无 CVM 实例 ID
		UserID:       1,
		IsDoctorNode: true,
	}
	model.DB(context.Background()).Create(&doctorInst)

	destroyDoctorNode(context.Background(), 501, log)

	// 无 CVM 实例，直接删除记录
	var inst model.Instance
	err := model.DB(context.Background()).First(&inst, 501).Error
	if err == nil {
		t.Error("无 CVM 实例时应直接删除实例记录")
	}
}

// ─── cleanupOrphanedDoctorNodes 补充测试 ────────────────────────────────────

func TestCleanupOrphanedDoctorNodes_发现残留节点则清理(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	log := slog.Default()

	doctorInstID := uint(600)
	doctorInst := model.Instance{
		Model:        gorm.Model{ID: doctorInstID},
		Name:         "doctor-orphan",
		InstanceId:   "", // 无 CVM，直接删记录
		UserID:       1,
		IsDoctorNode: true,
	}
	model.DB(context.Background()).Create(&doctorInst)

	session := model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusFailed,
	}
	model.DB(context.Background()).Create(&session)

	cleanupOrphanedDoctorNodes(context.Background(), log)

	// 实例记录应被删除
	var inst model.Instance
	err := model.DB(context.Background()).First(&inst, doctorInstID).Error
	if err == nil {
		t.Error("残留实例记录应被删除")
	}
}

func TestCleanupOrphanedDoctorNodes_实例已不存在则删session(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	log := slog.Default()

	nonExistID := uint(9998)
	session := model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 1,
		DoctorInstanceID: &nonExistID,
		Status:           model.DoctorStatusEnded,
	}
	model.DB(context.Background()).Create(&session)

	cleanupOrphanedDoctorNodes(context.Background(), log)

	// session 应被软删除
	var updated model.DoctorSession
	err := model.DB(context.Background()).
		First(&updated, session.ID).Error
	if err == nil {
		t.Error("实例已不存在时 session 应被软删除")
	}
}

// ─── processDoctorActivate 补充：CVM RUNNING 但 RunScript 失败 ────────────
