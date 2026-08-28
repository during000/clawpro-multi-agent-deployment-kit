package task

import (
	"context"
	"testing"
	"time"

	"hatchery/model"
)

func TestRecoverInterruptedRoleSyncTasks_NoRecords(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()
	recoverInterruptedRoleSyncTasks(context.Background())
	// no panic = pass
}

func TestRecoverInterruptedRoleSyncTasks_UpdatingRecord(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 插入 updating record（soul=pending，skill=running）
	model.DB(ctx).Create(&model.RoleDistributionRecord{
		InstanceID:  1,
		InstanceCID: "ins-1",
		RoleID:      1,
		Version:     "1.0",
		Status:      model.RoleRecordStatusUpdating,
		SoulStatus:  model.RoleSubStatusPending,
		SkillStatus: model.RoleSubStatusRunning,
	})
	// 插入 updating instance
	model.DB(ctx).Create(&model.Instance{
		Name: "agent-1", InstanceId: "ins-1", RoleID: 1,
		RoleSyncStatus: model.RoleSyncStatusUpdating,
	})

	recoverInterruptedRoleSyncTasks(ctx)

	var rec model.RoleDistributionRecord
	model.DB(ctx).First(&rec)
	if rec.Status != model.RoleRecordStatusFailed {
		t.Errorf("期望 status=failed，实际=%s", rec.Status)
	}
	if rec.SoulStatus != model.RoleSubStatusFailed {
		t.Errorf("期望 soul_status=failed，实际=%s", rec.SoulStatus)
	}
	if rec.SoulError != "服务重启中断" {
		t.Errorf("期望 soul_error=服务重启中断，实际=%s", rec.SoulError)
	}
	if rec.SkillStatus != model.RoleSubStatusFailed {
		t.Errorf("期望 skill_status=failed，实际=%s", rec.SkillStatus)
	}
	if rec.SkillError != "服务重启中断" {
		t.Errorf("期望 skill_error=服务重启中断，实际=%s", rec.SkillError)
	}

	var inst model.Instance
	model.DB(ctx).First(&inst)
	if inst.RoleSyncStatus != model.RoleSyncStatusFailed {
		t.Errorf("期望 role_sync_status=failed，实际=%s", inst.RoleSyncStatus)
	}
}

func TestRecoverInterruptedRoleSyncTasks_AlreadyFinalized(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 已 finalized 的 record 不应被修改
	model.DB(ctx).Create(&model.RoleDistributionRecord{
		InstanceID: 1, RoleID: 1, Version: "1.0",
		Status:      model.RoleRecordStatusUpdated,
		SoulStatus:  model.RoleSubStatusSuccess,
		SkillStatus: model.RoleSubStatusSuccess,
	})

	recoverInterruptedRoleSyncTasks(ctx)

	var rec model.RoleDistributionRecord
	model.DB(ctx).First(&rec)
	if rec.Status != model.RoleRecordStatusUpdated {
		t.Errorf("已 finalized 的 record 不应被改，实际=%s", rec.Status)
	}
}

func TestRecoverInterruptedRoleSyncTasks_PreserveExistingError(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()
	ctx := context.Background()

	model.DB(ctx).Create(&model.RoleDistributionRecord{
		InstanceID: 1, RoleID: 1, Version: "1.0",
		Status:      model.RoleRecordStatusUpdating,
		SoulStatus:  model.RoleSubStatusPending,
		SoulError:   "原有错误",
		SkillStatus: model.RoleSubStatusPending,
	})

	recoverInterruptedRoleSyncTasks(ctx)

	var rec model.RoleDistributionRecord
	model.DB(ctx).First(&rec)
	if rec.SoulError != "原有错误" {
		t.Errorf("已有错误不应被覆盖，实际=%s", rec.SoulError)
	}
}

func TestRecoverInterruptedRoleSyncTasks_SoulAlreadyFailed(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()
	ctx := context.Background()

	model.DB(ctx).Create(&model.RoleDistributionRecord{
		InstanceID: 1, RoleID: 1, Version: "1.0",
		Status:      model.RoleRecordStatusUpdating,
		SoulStatus:  model.RoleSubStatusFailed,
		SoulError:   "soul fail",
		SkillStatus: model.RoleSubStatusPending,
	})

	recoverInterruptedRoleSyncTasks(ctx)

	var rec model.RoleDistributionRecord
	model.DB(ctx).First(&rec)
	// soul 已是 failed，不应被改成 "服务重启中断"
	if rec.SoulStatus != model.RoleSubStatusFailed {
		t.Errorf("期望 soul_status=failed，实际=%s", rec.SoulStatus)
	}
	if rec.SoulError != "soul fail" {
		t.Errorf("期望 soul_error=soul fail，实际=%s", rec.SoulError)
	}
	// skill 仍为 pending 应被改 failed
	if rec.SkillStatus != model.RoleSubStatusFailed {
		t.Errorf("期望 skill_status=failed，实际=%s", rec.SkillStatus)
	}
}

func TestRunRoleSyncRefresh_NoInstances(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()
	runRoleSyncRefresh(context.Background())
}

func TestRunRoleSyncRefresh_WithInstances(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now()
	model.DB(ctx).Create(&model.Instance{Name: "a1", InstanceId: "ins-1", RoleSyncStatus: model.RoleSyncStatusUpdating})
	model.DB(ctx).Create(&model.Instance{Name: "a2", InstanceId: "ins-2", RoleSyncStatus: model.RoleSyncStatusUpdated})

	// 创建一条 updating record（差集空 + soul=success → 应收敛到 updated）
	model.DB(ctx).Create(&model.RoleDistributionRecord{
		InstanceID: 1, RoleID: 1, Version: "1.0",
		Status:      model.RoleRecordStatusUpdating,
		SoulStatus:  model.RoleSubStatusSuccess,
		SkillStatus: model.RoleSubStatusPending,
		SoulSetAt:   &now,
	})

	runRoleSyncRefresh(ctx)

	var rec model.RoleDistributionRecord
	model.DB(ctx).First(&rec)
	if rec.Status != model.RoleRecordStatusUpdated {
		t.Errorf("期望 record status=updated，实际=%s", rec.Status)
	}
}
