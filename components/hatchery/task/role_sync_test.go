package task

import (
	"context"
	"testing"

	"hatchery/model"
)

// TestRunRoleSyncRefresh_NoUpdatingInstances 验证没有 updating 实例时函数正常返回。
func TestRunRoleSyncRefresh_NoUpdatingInstances(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	// 不插入任何数据，直接调用
	runRoleSyncRefresh(context.Background())
	// 无 panic 即通过
}

// TestRunRoleSyncRefresh_WithUpdatingInstances 验证有 updating 实例时函数逐个调 refresh。
func TestRunRoleSyncRefresh_WithUpdatingInstances(t *testing.T) {
	cleanup := setupRecoverTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 插入 2 个 updating 实例和 1 个 updated 实例
	model.DB(ctx).Create(&model.Instance{
		Identifier:     "test",
		Name:           "agent-1",
		InstanceId:     "ins-1",
		RoleID:         1,
		RoleSyncStatus: model.RoleSyncStatusUpdating,
	})
	model.DB(ctx).Create(&model.Instance{
		Identifier:     "test",
		Name:           "agent-2",
		InstanceId:     "ins-2",
		RoleID:         1,
		RoleSyncStatus: model.RoleSyncStatusUpdating,
	})
	model.DB(ctx).Create(&model.Instance{
		Identifier:     "test",
		Name:           "agent-3",
		InstanceId:     "ins-3",
		RoleID:         1,
		RoleSyncStatus: model.RoleSyncStatusUpdated,
	})

	// 调用 runRoleSyncRefresh（内部会调 refreshRoleRecord，无 record 时是 no-op）
	runRoleSyncRefresh(ctx)

	// 实例状态不应改变（因为没有 record，refresh 是 no-op）
	var inst1 model.Instance
	model.DB(ctx).First(&inst1, 1)
	if inst1.RoleSyncStatus != model.RoleSyncStatusUpdating {
		t.Errorf("实例1状态应仍为 updating，实际=%s", inst1.RoleSyncStatus)
	}

	var inst3 model.Instance
	model.DB(ctx).First(&inst3, 3)
	if inst3.RoleSyncStatus != model.RoleSyncStatusUpdated {
		t.Errorf("实例3状态应仍为 updated，实际=%s", inst3.RoleSyncStatus)
	}
}
