package controller

import (
	"context"
	"strings"
	"testing"
	"time"

	"hatchery/model"
)

func waitForCondition(t *testing.T, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func TestHandleStatusSideEffects_DeleteMissingCVM_CleansMemoryArtifacts(t *testing.T) {
	setupMemoryProDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.SkillInstallation{}, &model.Notification{}, &model.SMHPersonalSpace{}); err != nil {
		t.Fatalf("migrate extra tables: %v", err)
	}

	inst := model.Instance{Name: "demo", InstanceId: "ins-deleting", UserID: 1, CurrentOperation: model.OpDelete}
	if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{InstanceID: inst.InstanceId, CurrentPlan: model.MemoryPlanFree}).Error; err != nil {
		t.Fatalf("create plugin: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.SkillInstallation{InstanceID: inst.ID, Name: "s1", Slug: "s1", Version: "1.0.0", InstallStatus: model.SkillInstalling}).Error; err != nil {
		t.Fatalf("create skill installation: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.SMHPersonalSpace{SpaceId: "space-1", UserId: 1, InstanceId: inst.ID, UserName: "u", InstanceName: inst.Name, CVMInstanceId: inst.InstanceId}).Error; err != nil {
		t.Fatalf("create personal space: %v", err)
	}

	handleStatusSideEffects(context.Background(), model.DB(context.Background()), &inst, nil, model.StatusDestroyed)

	// 等待异步 goroutine 完成所有清理工作（最后一步是标记个人空间待删除）
	waitForCondition(t, func() bool {
		var space model.SMHPersonalSpace
		if err := model.DB(context.Background()).Where("instance_id = ?", inst.ID).First(&space).Error; err != nil {
			return false
		}
		return space.ToBeDeletedAt != nil
	})

	var instCount int64
	model.DB(context.Background()).Model(&model.Instance{}).Where("id = ?", inst.ID).Count(&instCount)
	if instCount != 0 {
		t.Fatalf("instance should be deleted, got %d", instCount)
	}

	var pluginCount int64
	model.DB(context.Background()).Model(&model.MemoryTDAIPlugin{}).Where("instance_id = ?", inst.InstanceId).Count(&pluginCount)
	if pluginCount != 0 {
		t.Fatalf("plugin row should be deleted, got %d", pluginCount)
	}

	var skillCount int64
	model.DB(context.Background()).Model(&model.SkillInstallation{}).Where("instance_id = ?", inst.ID).Count(&skillCount)
	if skillCount != 0 {
		t.Fatalf("skill installations should be deleted, got %d", skillCount)
	}

	var space model.SMHPersonalSpace
	if err := model.DB(context.Background()).Where("instance_id = ?", inst.ID).First(&space).Error; err != nil {
		t.Fatalf("load personal space: %v", err)
	}
	if space.ToBeDeletedAt == nil {
		t.Fatal("personal space should be marked to be deleted")
	}
}

func TestInstanceDestroyMessage(t *testing.T) {
	msg := instanceDestroyMessage(context.Background(), "我的龙虾")
	if !strings.Contains(msg, "我的龙虾") {
		t.Fatalf("message should contain instance name, got %q", msg)
	}
}

func TestIsPostCreationCVMState(t *testing.T) {
	if !isPostCreationCVMState("RUNNING") {
		t.Fatal("RUNNING should be treated as post creation state")
	}
	if isPostCreationCVMState("PENDING") {
		t.Fatal("PENDING should not be treated as post creation state")
	}
}

func TestCVMInfoState(t *testing.T) {
	if got := cvmInfoState(nil); got != "" {
		t.Fatalf("nil cvmInfo should return empty state, got %q", got)
	}
	if got := cvmInfoState(&CVMInstanceInfo{State: "RUNNING"}); got != "RUNNING" {
		t.Fatalf("state = %q, want RUNNING", got)
	}
}
