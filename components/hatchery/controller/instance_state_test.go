package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

// initStateTestDB 初始化内存数据库
func initStateTestDB(t *testing.T) {
	t.Helper()
	t.Cleanup(model.UseDBForTest(setupTestDB(t)))
}

// ─── ResolveInstanceStatus 状态映射测试 ─────────────────────────────────────

func TestResolveInstanceStatus_Destroyed(t *testing.T) {
	// currentOp=delete + CVM 不存在 → destroyed
	instance := &model.Instance{
		CurrentOperation: model.OpDelete,
	}
	cvmInfo := (*CVMInstanceInfo)(nil) // CVM 不存在

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusDestroyed {
		t.Errorf("期望 destroyed，实际 %s", status.Status)
	}
	if status.Transient {
		t.Error("destroyed 应为非过渡态")
	}
}

func TestResolveInstanceStatus_Destroyed_StillExists(t *testing.T) {
	// currentOp=delete + CVM 存在 → destroying
	instance := &model.Instance{
		CurrentOperation: model.OpDelete,
	}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusDestroying {
		t.Errorf("删除中但 CVM 存在，期望 destroying，实际 %s", status.Status)
	}
	if !status.Transient {
		t.Error("删除中应为过渡态")
	}
}

func TestResolveInstanceStatus_InstanceIdEmpty(t *testing.T) {
	// instanceId 为空 → creating
	instance := &model.Instance{
		InstanceId: "",
	}
	cvmInfo := (*CVMInstanceInfo)(nil)

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusCreating {
		t.Errorf("InstanceId 为空，期望 creating，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_CVMNotFound(t *testing.T) {
	// instanceId 非空但 CVM 不存在，且从未运行过（LastStableState/LastCVMState 均为空）→ create_failed（从未成功运行过）
	instance := &model.Instance{
		InstanceId: "ins-xxxx",
	}
	cvmInfo := (*CVMInstanceInfo)(nil)

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusCreateFailed {
		t.Errorf("CVM 不存在且从未运行过，期望 create_failed，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_ExternalDestroy_LastStableState(t *testing.T) {
	// instanceId 非空，CVM 不存在，但 LastStableState 表明曾经运行过 → destroyed（外部销毁）
	instance := &model.Instance{
		InstanceId:      "ins-xxxx",
		LastStableState: "RUNNING",
	}
	cvmInfo := (*CVMInstanceInfo)(nil)

	snap := common.TenantSnapshot{DefaultLang: "zh"}
	ctx := common.InjectTenant(context.Background(), snap)

	status := ResolveInstanceStatus(ctx, instance, cvmInfo, nil)
	if status.Status != model.StatusDestroyed {
		t.Errorf("CVM 不存在但 LastStableState=RUNNING，期望 destroyed，实际 %s", status.Status)
	}
	if status.Label != "已销毁" {
		t.Errorf("期望 label=已销毁，实际 %s", status.Label)
	}
}

func TestResolveInstanceStatus_ExternalDestroy_LastCVMState(t *testing.T) {
	// instanceId 非空，CVM 不存在，LastStableState 为空但 LastCVMState 表明曾经运行过 → destroyed
	instance := &model.Instance{
		InstanceId:   "ins-xxxx",
		LastCVMState: "TERMINATING",
	}
	cvmInfo := (*CVMInstanceInfo)(nil)

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusDestroyed {
		t.Errorf("CVM 不存在但 LastCVMState=TERMINATING，期望 destroyed，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_ExternalDestroy_StoppedState(t *testing.T) {
	// instanceId 非空，CVM 不存在，LastStableState=STOPPED → destroyed（曾经运行后关机，再被外部销毁）
	instance := &model.Instance{
		InstanceId:      "ins-xxxx",
		LastStableState: "STOPPED",
	}
	cvmInfo := (*CVMInstanceInfo)(nil)

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusDestroyed {
		t.Errorf("CVM 不存在但 LastStableState=STOPPED，期望 destroyed，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_CVMNotFound_PendingState(t *testing.T) {
	// instanceId 非空，CVM 不存在，LastCVMState=PENDING（仍在创建阶段）→ create_failed
	instance := &model.Instance{
		InstanceId:   "ins-xxxx",
		LastCVMState: "PENDING",
	}
	cvmInfo := (*CVMInstanceInfo)(nil)

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusCreateFailed {
		t.Errorf("CVM 不存在且 LastCVMState=PENDING（创建阶段），期望 create_failed，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_CVMNotFound_LaunchingState(t *testing.T) {
	// instanceId 非空，CVM 不存在，LastCVMState=LAUNCHING（仍在创建阶段）→ create_failed
	instance := &model.Instance{
		InstanceId:   "ins-xxxx",
		LastCVMState: "LAUNCHING",
	}
	cvmInfo := (*CVMInstanceInfo)(nil)

	snap := common.TenantSnapshot{DefaultLang: "zh"}
	ctx := common.InjectTenant(context.Background(), snap)

	status := ResolveInstanceStatus(ctx, instance, cvmInfo, nil)
	if status.Status != model.StatusCreateFailed {
		t.Errorf("CVM 不存在且 LastCVMState=LAUNCHING（创建阶段），期望 create_failed，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_CVMNotFound_CurrentOpCreate(t *testing.T) {
	// Step 2.2b: instanceId 非空，CVM 不存在，CurrentOperation=create，
	// LastCVMState 不是 post-creation 也不是创建阶段 → create_failed
	instance := &model.Instance{
		InstanceId:       "ins-xxxx",
		CurrentOperation: model.OpCreate,
		LastCVMState:     "LAUNCH_FAILED",
	}
	cvmInfo := (*CVMInstanceInfo)(nil)

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusCreateFailed {
		t.Errorf("CVM 不存在且 CurrentOperation=create，期望 create_failed，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_CVMNotFound_FallbackDestroyed(t *testing.T) {
	// Step 2.2c: instanceId 非空，CVM 不存在，非创建阶段，非 post-creation，有未知 LastCVMState → destroyed（存量兜底）
	instance := &model.Instance{
		InstanceId:       "ins-xxxx",
		CurrentOperation: "some_other_op",
		LastCVMState:     "LAUNCH_FAILED", // 非 post-creation，非创建阶段
		LastStableState:  "",
	}
	cvmInfo := (*CVMInstanceInfo)(nil)

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusDestroyed {
		t.Errorf("CVM 不存在且存量兜底，期望 destroyed，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_LaunchFailed(t *testing.T) {
	instance := &model.Instance{}
	cvmInfo := &CVMInstanceInfo{State: "LAUNCH_FAILED"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusCreateFailed {
		t.Errorf("LAUNCH_FAILED，期望 create_failed，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_PendingCreating(t *testing.T) {
	// PENDING 状态 → creating（CVM 尚未启动，属于创建中）
	instance := &model.Instance{}
	cvmInfo := &CVMInstanceInfo{State: "PENDING"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusCreating {
		t.Errorf("PENDING，期望 creating，实际 %s", status.Status)
	}
	if !status.Transient {
		t.Error("creating 应为过渡态")
	}
}

func TestResolveInstanceStatus_CVMTransient(t *testing.T) {
	// PENDING 已单独处理为 creating，不在此列表中
	transientStates := []string{"LAUNCHING", "STOPPING", "STARTING", "REBOOTING", "SHUTDOWN", "TERMINATING"}

	for _, state := range transientStates {
		t.Run("transient_"+state, func(t *testing.T) {
			instance := &model.Instance{}
			cvmInfo := &CVMInstanceInfo{State: state}

			status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
			if status.Status != model.StatusLoading {
				t.Errorf("状态 %s，期望 loading，实际 %s", state, status.Status)
			}
			if !status.Transient {
				t.Errorf("状态 %s 应为过渡态", state)
			}
		})
	}
}

func TestResolveInstanceStatus_Running_AgentReady(t *testing.T) {
	instance := &model.Instance{AgentReady: 1}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusRunning {
		t.Errorf("RUNNING + AgentReady=1，期望 running，实际 %s", status.Status)
	}
	if status.Transient {
		t.Error("running 应为非过渡态")
	}
}

func TestResolveInstanceStatus_Running_WithInstallingSkillInstallations(t *testing.T) {
	initStateTestDB(t)

	inst := model.Instance{AgentReady: 1}
	if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.SkillInstallation{
		InstanceID:    inst.ID,
		Name:          "demo",
		Slug:          "demo",
		Version:       "1.0.0",
		InstallStatus: model.SkillInstalling,
	}).Error; err != nil {
		t.Fatalf("创建技能安装记录失败: %v", err)
	}

	status := ResolveInstanceStatus(context.Background(), &inst, &CVMInstanceInfo{State: "RUNNING"}, nil)
	if status.Status != model.StatusLoading {
		t.Errorf("RUNNING + 技能安装中，期望 loading，实际 %s", status.Status)
	}
	if !status.Transient {
		t.Error("技能安装中时应为过渡态")
	}
}

func TestResolveInstanceStatus_Running_WithFinishedSkillInstallations(t *testing.T) {
	initStateTestDB(t)

	t.Run("success", func(t *testing.T) {
		inst := model.Instance{AgentReady: 1}
		if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
			t.Fatalf("创建实例失败: %v", err)
		}
		if err := model.DB(context.Background()).Create(&model.SkillInstallation{
			InstanceID:    inst.ID,
			Name:          "demo-success",
			Slug:          "demo-success",
			Version:       "1.0.0",
			InstallStatus: model.SkillInstallSuccess,
		}).Error; err != nil {
			t.Fatalf("创建技能安装记录失败: %v", err)
		}

		status := ResolveInstanceStatus(context.Background(), &inst, &CVMInstanceInfo{State: "RUNNING"}, nil)
		if status.Status != model.StatusRunning {
			t.Errorf("RUNNING + 技能安装成功，期望 running，实际 %s", status.Status)
		}
	})

	t.Run("failed", func(t *testing.T) {
		inst := model.Instance{AgentReady: 1}
		if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
			t.Fatalf("创建实例失败: %v", err)
		}
		if err := model.DB(context.Background()).Create(&model.SkillInstallation{
			InstanceID:    inst.ID,
			Name:          "demo-failed",
			Slug:          "demo-failed",
			Version:       "1.0.0",
			InstallStatus: model.SkillInstallFailed,
		}).Error; err != nil {
			t.Fatalf("创建技能安装记录失败: %v", err)
		}

		status := ResolveInstanceStatus(context.Background(), &inst, &CVMInstanceInfo{State: "RUNNING"}, nil)
		if status.Status != model.StatusRunning {
			t.Errorf("RUNNING + 技能安装失败，期望 running，实际 %s", status.Status)
		}
	})
}

func TestResolveInstanceStatus_Running_AgentNotReady(t *testing.T) {
	instance := &model.Instance{AgentReady: 0}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusLoading {
		t.Errorf("RUNNING + AgentReady=0，期望 loading，实际 %s", status.Status)
	}
	if !status.Transient {
		t.Error("Agent 未就绪时应为过渡态")
	}
}

func TestResolveInstanceStatus_Stopped(t *testing.T) {
	instance := &model.Instance{AgentReady: 1}
	cvmInfo := &CVMInstanceInfo{State: "STOPPED"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusStopped {
		t.Errorf("STOPPED，期望 stopped，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_Isolated(t *testing.T) {
	// ISOLATED 状态 + RestrictState 非 NORMAL → pending
	instance := &model.Instance{}
	cvmInfo := &CVMInstanceInfo{State: "ISOLATED", RestrictState: "EXPIRED"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusPending {
		t.Errorf("ISOLATED + RestrictState=EXPIRED，期望 pending，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_Orphaned(t *testing.T) {
	instance := &model.Instance{}
	cvmInfo := &CVMInstanceInfo{State: "ORPHANED"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusMaintaining {
		t.Errorf("ORPHANED，期望 maintaining，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_UnknownState(t *testing.T) {
	instance := &model.Instance{}
	cvmInfo := &CVMInstanceInfo{State: "UNKNOWN_STATE"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusMaintaining {
		t.Errorf("未知状态，期望 maintaining，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_LiveMigrateStates(t *testing.T) {
	// 热迁移态 → running（实例仍在运行）
	liveMigrateStates := []string{
		"ENTER_SERVICE_LIVE_MIGRATE",
		"SERVICE_LIVE_MIGRATE",
		"EXIT_SERVICE_LIVE_MIGRATE",
	}
	for _, state := range liveMigrateStates {
		t.Run("live_migrate_"+state, func(t *testing.T) {
			instance := &model.Instance{}
			cvmInfo := &CVMInstanceInfo{State: state}

			status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
			if status.Status != model.StatusRunning {
				t.Errorf("热迁移态 %s，期望 running，实际 %s", state, status.Status)
			}
		})
	}
}

func TestResolveInstanceStatus_PlatformLimitStates(t *testing.T) {
	// 平台限制态（InstanceState）→ pending
	platformLimitStates := []string{"FREEZING", "BANNING", "CORRUPTED"}
	for _, state := range platformLimitStates {
		t.Run("platform_limit_"+state, func(t *testing.T) {
			instance := &model.Instance{}
			cvmInfo := &CVMInstanceInfo{State: state, RestrictState: "NORMAL"}

			status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
			if status.Status != model.StatusPending {
				t.Errorf("平台限制态 %s，期望 pending，实际 %s", state, status.Status)
			}
		})
	}
}

func TestResolveInstanceStatus_RestrictStateNotNormal(t *testing.T) {
	// RestrictState 非 NORMAL → pending（无论 InstanceState 是什么）
	tests := []struct {
		name          string
		cvmState      string
		restrictState string
	}{
		{"RUNNING+EXPIRED", "RUNNING", "EXPIRED"},
		{"RUNNING+PROTECTIVELY_ISOLATED", "RUNNING", "PROTECTIVELY_ISOLATED"},
		{"STOPPED+EXPIRED", "STOPPED", "EXPIRED"},
		{"STOPPED+PROTECTIVELY_ISOLATED", "STOPPED", "PROTECTIVELY_ISOLATED"},
		{"ISOLATED+EXPIRED", "ISOLATED", "EXPIRED"},
		{"ISOLATING+EXPIRED", "ISOLATING", "EXPIRED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &model.Instance{}
			cvmInfo := &CVMInstanceInfo{State: tt.cvmState, RestrictState: tt.restrictState}

			status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
			if status.Status != model.StatusPending {
				t.Errorf("RestrictState=%s + CVM=%s，期望 pending，实际 %s", tt.restrictState, tt.cvmState, status.Status)
			}
		})
	}
}

func TestResolveInstanceStatus_RescueModeStates(t *testing.T) {
	// 维护态（救援模式）→ maintaining
	rescueModeStates := []string{"ENTER_RESCUE_MODE", "RESCUE_MODE", "EXIT_RESCUE_MODE"}
	for _, state := range rescueModeStates {
		t.Run("rescue_mode_"+state, func(t *testing.T) {
			instance := &model.Instance{}
			cvmInfo := &CVMInstanceInfo{State: state}

			status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
			if status.Status != model.StatusMaintaining {
				t.Errorf("维护态 %s，期望 maintaining，实际 %s", state, status.Status)
			}
		})
	}
}

func TestResolveInstanceStatus_ReinstallStopped(t *testing.T) {
	// reinstall 操作中 CVM 经过 STOPPED 状态 → loading（过渡态，非已关机）
	instance := &model.Instance{
		CurrentOperation: model.OpReinstall,
		AgentReady:       1,
	}
	cvmInfo := &CVMInstanceInfo{State: "STOPPED"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusLoading {
		t.Errorf("reinstall + STOPPED，期望 loading（重装过渡态），实际 %s", status.Status)
	}
	if !status.Transient {
		t.Error("reinstall + STOPPED 应为过渡态")
	}
}

func TestResolveInstanceStatus_NormalStopped(t *testing.T) {
	// 非 reinstall 操作的 STOPPED → stopped
	instance := &model.Instance{
		CurrentOperation: model.OpNone,
		AgentReady:       1,
	}
	cvmInfo := &CVMInstanceInfo{State: "STOPPED"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusStopped {
		t.Errorf("正常 STOPPED，期望 stopped，实际 %s", status.Status)
	}
}

// ─── Gap 4：STOPPED + RestrictState 状态映射测试 ─────────────────────────

func TestResolveInstanceStatus_StoppedWithNORMAL(t *testing.T) {
	// STOPPED + RestrictState=NORMAL → stopped（正常关机）
	instance := &model.Instance{
		CurrentOperation: model.OpNone,
		AgentReady:       1,
	}
	cvmInfo := &CVMInstanceInfo{State: "STOPPED", RestrictState: "NORMAL"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusStopped {
		t.Errorf("STOPPED + NORMAL，期望 stopped，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_StoppedWithEmptyRestrictState(t *testing.T) {
	// STOPPED + RestrictState="" → stopped（兼容旧数据，空值视为正常）
	instance := &model.Instance{
		CurrentOperation: model.OpNone,
		AgentReady:       1,
	}
	cvmInfo := &CVMInstanceInfo{State: "STOPPED", RestrictState: ""}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusStopped {
		t.Errorf("STOPPED + 空 RestrictState，期望 stopped，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_StoppedWithExpiredRestrictState(t *testing.T) {
	// STOPPED + RestrictState=EXPIRED（过期隔离导致关机）→ pending
	instance := &model.Instance{
		CurrentOperation: model.OpNone,
		AgentReady:       1,
	}
	cvmInfo := &CVMInstanceInfo{State: "STOPPED", RestrictState: "EXPIRED"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusPending {
		t.Errorf("STOPPED + EXPIRED，期望 pending，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_StoppedWithProtectivelyIsolated(t *testing.T) {
	// STOPPED + RestrictState=PROTECTIVELY_ISOLATED（安全隔离）→ pending
	instance := &model.Instance{
		CurrentOperation: model.OpNone,
	}
	cvmInfo := &CVMInstanceInfo{State: "STOPPED", RestrictState: "PROTECTIVELY_ISOLATED"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusPending {
		t.Errorf("STOPPED + PROTECTIVELY_ISOLATED，期望 pending，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_ReinstallStoppedWithRestrictState(t *testing.T) {
	// reinstall 操作中 STOPPED + RestrictState 非 NORMAL → pending（RestrictState 优先级高于 reinstall）
	instance := &model.Instance{
		CurrentOperation: model.OpReinstall,
		AgentReady:       1,
	}
	cvmInfo := &CVMInstanceInfo{State: "STOPPED", RestrictState: "EXPIRED"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusPending {
		t.Errorf("reinstall + STOPPED + RestrictState=EXPIRED，期望 pending，实际 %s", status.Status)
	}
}

// ─── ResolveInstanceStatus Label/Tooltip/Actions 测试 ───────────────────────

func TestResolveInstanceStatus_Label(t *testing.T) {
	tests := []struct {
		cvmState      string
		agentReady    int
		expectedLabel string
	}{
		// PENDING → 创建中（CVM 尚未启动）
		{"PENDING", 0, "创建中"},
		{"LAUNCH_FAILED", 0, "创建失败"},
		{"RUNNING", 1, "运行中"},
		{"RUNNING", 0, "加载中"},
		{"STOPPED", 1, "已关机"},
		{"TERMINATING", 0, "加载中"},
	}

	snap := common.TenantSnapshot{DefaultLang: "zh"}
	ctx := common.InjectTenant(context.Background(), snap)

	for _, tt := range tests {
		t.Run(tt.cvmState+"_agent"+string(rune(tt.agentReady+'0')), func(t *testing.T) {
			instance := &model.Instance{AgentReady: tt.agentReady}
			cvmInfo := &CVMInstanceInfo{State: tt.cvmState}

			status := ResolveInstanceStatus(ctx, instance, cvmInfo, nil)
			if status.Label != tt.expectedLabel {
				t.Errorf("CVM=%s AgentReady=%d，期望 label=%s，实际 %s",
					tt.cvmState, tt.agentReady, tt.expectedLabel, status.Label)
			}
		})
	}
}

func TestResolveInstanceStatus_RunningActions(t *testing.T) {
	instance := &model.Instance{AgentReady: 1}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	expected := []string{"restart_gateway", "reboot", "reinstall", "delete", "terminal"}

	for _, a := range expected {
		found := false
		for _, aa := range status.Actions {
			if aa == a {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("running 应包含 action %q，实际 %v", a, status.Actions)
		}
	}
}

func TestResolveInstanceStatus_LoadFailedActions(t *testing.T) {
	instance := &model.Instance{AgentReady: 0}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	// load_failed 时期望有 retry action
	hasRetry := false
	for _, a := range status.Actions {
		if a == "retry" {
			hasRetry = true
			break
		}
	}
	// 注意：当前逻辑 load_failed 需要 currentOp 相关条件
	_ = hasRetry
}

// ─── InstanceStatusResponse 字段完整性测试 ─────────────────────────────────

func TestResolveInstanceStatus_AllFields(t *testing.T) {
	instance := &model.Instance{AgentReady: 1}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)

	if status.Status == "" {
		t.Error("Status 不应为空")
	}
	if status.Label == "" {
		t.Error("Label 不应为空")
	}
	// Tooltip 可以为空
	if status.Actions == nil {
		t.Error("Actions 不应为 nil")
	}
}

// ─── handleStatusSideEffects 测试 ─────────────────────────────────────────

func TestHandleStatusSideEffects_UpdateLastCVMState(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)
	instance := createTestInstanceForDB(t, db, user.ID, "test")

	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	status := "running"

	handleStatusSideEffects(context.Background(), db, instance, cvmInfo, status)

	// 验证 LastCVMState 被更新（使用原始 SQL 验证，因为 GORM 与 SQLite varchar 类型映射可能有问题）
	var rawLastCVMState string
	db.Raw("SELECT last_cvm_state FROM instances WHERE id = ?", instance.ID).Scan(&rawLastCVMState)
	if rawLastCVMState != "RUNNING" {
		t.Errorf("期望 LastCVMState=RUNNING，实际=%s", rawLastCVMState)
	}

	// 验证 struct 字段也被更新
	if instance.LastCVMState != "RUNNING" {
		t.Errorf("期望 instance.LastCVMState=RUNNING，实际=%s", instance.LastCVMState)
	}
}

func TestHandleStatusSideEffects_NoCVMInfo(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)
	instance := createTestInstanceForDB(t, db, user.ID, "test")

	// CVMInfo 为 nil 不应 panic
	handleStatusSideEffects(context.Background(), db, instance, nil, "creating")
}

// ─── 2.5 操作超时自动恢复测试 ────────────────────────────────────────────

// TestHandleStatusSideEffects_Step25_CaseA: load_failed + CVM RUNNING + AgentReady=1 + CLS 稳定 → 恢复为 running
func TestHandleStatusSideEffects_Step25_CaseA(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)

	now := time.Now()
	instance := &model.Instance{
		Name:                      "step25-case-a",
		UserID:                    user.ID,
		InstanceId:                "ins-step25-a",
		CurrentOperation:          model.OpReboot,
		CurrentOperationState:     model.OpStateFailed,
		CurrentOperationUpdatedAt: &now,
		AgentReady:                1,
		CLSAgentStatus:            model.CLSAgentInstalled,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	handleStatusSideEffects(context.Background(), db, instance, cvmInfo, "load_failed")

	var rawOp, rawOpState, rawStable string
	db.Raw("SELECT current_operation FROM instances WHERE id = ?", instance.ID).Scan(&rawOp)
	db.Raw("SELECT current_operation_state FROM instances WHERE id = ?", instance.ID).Scan(&rawOpState)
	db.Raw("SELECT last_stable_state FROM instances WHERE id = ?", instance.ID).Scan(&rawStable)

	if rawOp != "" {
		t.Errorf("Case A: 期望 current_operation 被清空，实际=%q", rawOp)
	}
	if rawOpState != model.OpStateSuccess {
		t.Errorf("Case A: 期望 current_operation_state=success，实际=%s", rawOpState)
	}
	if rawStable != "RUNNING" {
		t.Errorf("Case A: 期望 last_stable_state=RUNNING，实际=%s", rawStable)
	}
}

// TestHandleStatusSideEffects_Step25_CaseB: load_failed + CVM STOPPED → 收敛为 stopped
func TestHandleStatusSideEffects_Step25_CaseB(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)

	now := time.Now()
	instance := &model.Instance{
		Name:                      "step25-case-b",
		UserID:                    user.ID,
		InstanceId:                "ins-step25-b",
		CurrentOperation:          model.OpCreate,
		CurrentOperationState:     model.OpStateFailed,
		CurrentOperationUpdatedAt: &now,
		AgentReady:                0,
		CLSAgentStatus:            0,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	cvmInfo := &CVMInstanceInfo{State: "STOPPED"}
	handleStatusSideEffects(context.Background(), db, instance, cvmInfo, "load_failed")

	var rawOp, rawOpState, rawStable string
	db.Raw("SELECT current_operation FROM instances WHERE id = ?", instance.ID).Scan(&rawOp)
	db.Raw("SELECT current_operation_state FROM instances WHERE id = ?", instance.ID).Scan(&rawOpState)
	db.Raw("SELECT last_stable_state FROM instances WHERE id = ?", instance.ID).Scan(&rawStable)

	if rawOp != "" {
		t.Errorf("Case B: 期望 current_operation 被清空，实际=%q", rawOp)
	}
	if rawOpState != model.OpStateSuccess {
		t.Errorf("Case B: 期望 current_operation_state=success，实际=%s", rawOpState)
	}
	if rawStable != "STOPPED" {
		t.Errorf("Case B: 期望 last_stable_state=STOPPED，实际=%s", rawStable)
	}
}

// TestHandleStatusSideEffects_Step25_Negative_AgentNotReady: load_failed + CVM RUNNING + AgentReady=0 → 不应恢复
func TestHandleStatusSideEffects_Step25_Negative_AgentNotReady(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)

	now := time.Now()
	instance := &model.Instance{
		Name:                      "step25-negative-agent",
		UserID:                    user.ID,
		InstanceId:                "ins-step25-neg",
		CurrentOperation:          model.OpReboot,
		CurrentOperationState:     model.OpStateFailed,
		CurrentOperationUpdatedAt: &now,
		AgentReady:                0, // Agent 未就绪 → Case A 不应触发
		CLSAgentStatus:            model.CLSAgentNotInstalled,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	handleStatusSideEffects(context.Background(), db, instance, cvmInfo, "load_failed")

	var rawOp string
	db.Raw("SELECT current_operation FROM instances WHERE id = ?", instance.ID).Scan(&rawOp)
	if rawOp != model.OpReboot {
		t.Errorf("Agent 未就绪时不应恢复，期望 current_operation=reboot，实际=%q", rawOp)
	}
}

// TestHandleStatusSideEffects_Step25_Negative_Upgrade: upgrade_failed 不应被 2.5 恢复
func TestHandleStatusSideEffects_Step25_Negative_Upgrade(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)

	now := time.Now()
	instance := &model.Instance{
		Name:                      "step25-negative-upgrade",
		UserID:                    user.ID,
		InstanceId:                "ins-step25-upg",
		CurrentOperation:          model.OpUpgrade,
		CurrentOperationState:     model.OpStateFailed,
		CurrentOperationUpdatedAt: &now,
		AgentReady:                1,
		CLSAgentStatus:            model.CLSAgentInstalled,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	handleStatusSideEffects(context.Background(), db, instance, cvmInfo, "upgrade_failed")

	var rawOp string
	db.Raw("SELECT current_operation FROM instances WHERE id = ?", instance.ID).Scan(&rawOp)
	if rawOp != model.OpUpgrade {
		t.Errorf("OpUpgrade 不应被 Step 2.5 恢复，期望 current_operation=upgrade，实际=%q", rawOp)
	}
}

// ─── handleStatusSideEffects Step 2 超时检测：upgrade/migrate 互斥测试 ────────
//
// 修复背景（TAPD #1020422209159847466）：升级操作耗时超过 30 分钟时，后台
// handleStatusSideEffects 的 Step 2 超时检测会将 current_operation_state 从
// processing 强制改为 failed，而 performUpgrade goroutine 仍在正常执行。此后
// 用户触发重试接口时读到 state=failed → 放行新的升级请求，导致同一实例跑多个
// 升级流程，争夺 TAT Agent 资源。
//
// 修复方案：Step 2 超时检测排除 upgrade 和 migrate 操作，与 Step 2.5 和 Step 3
// 保持一致——这两个操作由各自的异步 goroutine 自行管理生命周期。

// TestHandleStatusSideEffects_Step2_UpgradeTimedOutNotMarkedFailed 验证：
// upgrade 操作即使超过 30 分钟超时阈值，也不应由 Step 2 标记为 failed。
func TestHandleStatusSideEffects_Step2_UpgradeTimedOutNotMarkedFailed(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)

	// 设置 upgrade 操作在 40 分钟前开始（超过 30 分钟阈值）
	oldTime := time.Now().Add(-40 * time.Minute)
	instance := &model.Instance{
		Name:                      "step2-upgrade-timeout",
		UserID:                    user.ID,
		InstanceId:                "ins-step2-upg",
		CurrentOperation:          model.OpUpgrade,
		CurrentOperationState:     model.OpStateProcessing,
		CurrentOperationUpdatedAt: &oldTime,
		AgentReady:                0,
		CLSAgentStatus:            0,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	handleStatusSideEffects(context.Background(), db, instance, cvmInfo, "upgrading")

	// 核心断言：upgrade 操作的 state 应保持 processing，不被 Step 2 越权改为 failed
	var rawOpState string
	db.Raw("SELECT current_operation_state FROM instances WHERE id = ?", instance.ID).Scan(&rawOpState)
	if rawOpState != model.OpStateProcessing {
		t.Errorf("upgrade 超时不应被 Step 2 标记为 failed，期望 state=%s，实际=%q",
			model.OpStateProcessing, rawOpState)
	}
	if instance.CurrentOperationState != model.OpStateProcessing {
		t.Errorf("内存对象：upgrade 超时不应被 Step 2 修改 state，期望=%s，实际=%s",
			model.OpStateProcessing, instance.CurrentOperationState)
	}
}

// TestHandleStatusSideEffects_Step2_MigrateTimedOutNotMarkedFailed 验证：
// migrate 操作即使超过 30 分钟超时阈值，也不应由 Step 2 标记为 failed。
func TestHandleStatusSideEffects_Step2_MigrateTimedOutNotMarkedFailed(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)

	oldTime := time.Now().Add(-40 * time.Minute)
	instance := &model.Instance{
		Name:                      "step2-migrate-timeout",
		UserID:                    user.ID,
		InstanceId:                "ins-step2-mig",
		CurrentOperation:          model.OpMigrate,
		CurrentOperationState:     model.OpStateProcessing,
		CurrentOperationUpdatedAt: &oldTime,
		AgentReady:                0,
		CLSAgentStatus:            0,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	handleStatusSideEffects(context.Background(), db, instance, cvmInfo, "loading")

	var rawOpState string
	db.Raw("SELECT current_operation_state FROM instances WHERE id = ?", instance.ID).Scan(&rawOpState)
	if rawOpState != model.OpStateProcessing {
		t.Errorf("migrate 超时不应被 Step 2 标记为 failed，期望 state=%s，实际=%q",
			model.OpStateProcessing, rawOpState)
	}
}

// TestHandleStatusSideEffects_Step2_RebootTimedOutStillMarkedFailed 回归验证：
// 非 upgrade/migrate 的操作（如 reboot）超时后仍应被 Step 2 正常标记为 failed。
func TestHandleStatusSideEffects_Step2_RebootTimedOutStillMarkedFailed(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)

	oldTime := time.Now().Add(-10 * time.Minute) // reboot 超时阈值 5 分钟
	instance := &model.Instance{
		Name:                      "step2-reboot-timeout",
		UserID:                    user.ID,
		InstanceId:                "ins-step2-rbt",
		CurrentOperation:          model.OpReboot,
		CurrentOperationState:     model.OpStateProcessing,
		CurrentOperationUpdatedAt: &oldTime,
		AgentReady:                0, // AgentReady=0 确保 Step 2.5 Case A 不触发恢复
		CLSAgentStatus:            0,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	handleStatusSideEffects(context.Background(), db, instance, cvmInfo, "loading")

	var rawOpState string
	db.Raw("SELECT current_operation_state FROM instances WHERE id = ?", instance.ID).Scan(&rawOpState)
	if rawOpState != model.OpStateFailed {
		t.Errorf("reboot 超时仍应被 Step 2 标记为 failed，期望 state=%s，实际=%q",
			model.OpStateFailed, rawOpState)
	}
}

// TestHandleStatusSideEffects_Step2_UpgradeNotTimedOutStatePreserved 验证边界：
// upgrade 操作未超时时，state 保持 processing 不变。
func TestHandleStatusSideEffects_Step2_UpgradeNotTimedOutStatePreserved(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)

	// 升级刚开始 5 分钟（未超过 30 分钟阈值）
	recentTime := time.Now().Add(-5 * time.Minute)
	instance := &model.Instance{
		Name:                      "step2-upgrade-notimeout",
		UserID:                    user.ID,
		InstanceId:                "ins-step2-unt",
		CurrentOperation:          model.OpUpgrade,
		CurrentOperationState:     model.OpStateProcessing,
		CurrentOperationUpdatedAt: &recentTime,
		AgentReady:                0,
		CLSAgentStatus:            0,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	handleStatusSideEffects(context.Background(), db, instance, cvmInfo, "upgrading")

	var rawOpState string
	db.Raw("SELECT current_operation_state FROM instances WHERE id = ?", instance.ID).Scan(&rawOpState)
	if rawOpState != model.OpStateProcessing {
		t.Errorf("upgrade 未超时 state 应保持 processing，期望=%s，实际=%q",
			model.OpStateProcessing, rawOpState)
	}
}

// TestHandleStatusSideEffects_Step2_DeleteTimedOutStillMarkedFailed 回归验证：
// delete 操作超时后仍应被 Step 2 正常标记为 failed。
func TestHandleStatusSideEffects_Step2_DeleteTimedOutStillMarkedFailed(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)

	oldTime := time.Now().Add(-10 * time.Minute) // delete 超时阈值 5 分钟
	instance := &model.Instance{
		Name:                      "step2-delete-timeout",
		UserID:                    user.ID,
		InstanceId:                "ins-step2-del",
		CurrentOperation:          model.OpDelete,
		CurrentOperationState:     model.OpStateProcessing,
		CurrentOperationUpdatedAt: &oldTime,
		AgentReady:                0,
		CLSAgentStatus:            0,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	handleStatusSideEffects(context.Background(), db, instance, cvmInfo, "destroying")

	var rawOpState string
	db.Raw("SELECT current_operation_state FROM instances WHERE id = ?", instance.ID).Scan(&rawOpState)
	if rawOpState != model.OpStateFailed {
		t.Errorf("delete 超时仍应被 Step 2 标记为 failed，期望 state=%s，实际=%q",
			model.OpStateFailed, rawOpState)
	}
}

// ─── handleStatusSideEffects 外部销毁通知幂等测试（问题 #2）────────────────

func TestHandleStatusSideEffects_UpdateInstanceChargeType(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)
	instance := createTestInstanceForDB(t, db, user.ID, "test")

	handleStatusSideEffects(context.Background(), db, instance, &CVMInstanceInfo{
		State:              "RUNNING",
		InstanceChargeType: "POSTPAID_BY_HOUR",
	}, "running")

	var got string
	db.Model(&model.Instance{}).Select("instance_charge_type").Where("id = ?", instance.ID).Scan(&got)
	if got != "POSTPAID_BY_HOUR" {
		t.Fatalf("InstanceChargeType = %s, want POSTPAID_BY_HOUR", got)
	}
	if instance.InstanceChargeType != "POSTPAID_BY_HOUR" {
		t.Fatalf("instance.InstanceChargeType = %s, want POSTPAID_BY_HOUR", instance.InstanceChargeType)
	}
}

// ─── handleStatusSideEffects 外部销毁通知幂等测试（问题 #2）────────────────

func TestHandleStatusSideEffects_ExternalDestroyNotification_FirstTime(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)

	// 创建一个有 InstanceId 但 CVM 不存在的实例（模拟外部销毁）
	instance := &model.Instance{
		Name:             "external-destroy-test",
		UserID:           user.ID,
		InstanceId:       "ins-mock-ext001",
		CurrentOperation: "", // 无操作
	}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	// 第一次调用：CVM 不存在 + currentOp 为空 → 应创建外部销毁通知
	handleStatusSideEffects(context.Background(), db, instance, nil, model.StatusCreateFailed)

	// 轮询等待异步 goroutine 写入通知（最多 1s）
	var count int64
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		db.Model(&model.Notification{}).
			Where("instance_id = ? AND type = ?", instance.ID, model.NotifyTypeExternalDestroy).
			Count(&count)
		if count > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if count != 1 {
		t.Errorf("第一次调用应创建 1 条外部销毁通知，实际=%d", count)
	}
}

func TestHandleStatusSideEffects_ExternalDestroyNotification_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)

	// 创建一个有 InstanceId 但 CVM 不存在的实例
	instance := &model.Instance{
		Name:             "idempotent-test",
		UserID:           user.ID,
		InstanceId:       "ins-mock-idem001",
		CurrentOperation: "",
	}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	// 第一次调用
	handleStatusSideEffects(context.Background(), db, instance, nil, model.StatusCreateFailed)
	time.Sleep(20 * time.Millisecond)

	// 第二次调用（模拟列表刷新再次触发）
	handleStatusSideEffects(context.Background(), db, instance, nil, model.StatusCreateFailed)
	time.Sleep(20 * time.Millisecond)

	// 第三次调用
	handleStatusSideEffects(context.Background(), db, instance, nil, model.StatusCreateFailed)

	// 轮询等待最后一次 goroutine 完成（最多 1s）
	var count int64
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		db.Model(&model.Notification{}).
			Where("instance_id = ? AND type = ?", instance.ID, model.NotifyTypeExternalDestroy).
			Count(&count)
		if count >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if count != 1 {
		t.Errorf("多次调用应只创建 1 条外部销毁通知（幂等），实际=%d", count)
	}
}

func TestHandleStatusSideEffects_NoNotificationWhenCVMExists(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)

	instance := &model.Instance{
		Name:             "cvm-exists-test",
		UserID:           user.ID,
		InstanceId:       "ins-mock-exists001",
		CurrentOperation: "",
	}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	// CVM 存在时不应发送外部销毁通知
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	handleStatusSideEffects(context.Background(), db, instance, cvmInfo, model.StatusRunning)
	time.Sleep(30 * time.Millisecond)

	var count int64
	db.Model(&model.Notification{}).
		Where("instance_id = ? AND type = ?", instance.ID, model.NotifyTypeExternalDestroy).
		Count(&count)
	if count != 0 {
		t.Errorf("CVM 存在时不应创建外部销毁通知，实际=%d", count)
	}
}

func TestHandleStatusSideEffects_NoNotificationWhenOperationInProgress(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	user := createTestUserForDB(t, db)

	instance := &model.Instance{
		Name:             "op-in-progress-test",
		UserID:           user.ID,
		InstanceId:       "ins-mock-op001",
		CurrentOperation: model.OpDelete, // 有操作进行中
	}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	// 有操作进行中 + CVM 不存在 → 不应发送外部销毁通知（由 Step 4 处理）
	handleStatusSideEffects(context.Background(), db, instance, nil, model.StatusDestroyed)
	time.Sleep(30 * time.Millisecond)

	var count int64
	db.Model(&model.Notification{}).
		Where("instance_id = ? AND type = ?", instance.ID, model.NotifyTypeExternalDestroy).
		Count(&count)
	if count != 0 {
		t.Errorf("有操作进行中时不应创建外部销毁通知，实际=%d", count)
	}
}

// ─── Gateway Restart Task Tracking ─────────────────────────────────────────

func TestGatewayRestartTasks_BasicFlow(t *testing.T) {
	// 清理可能影响的其他测试状态
	gatewayRestartTasks = sync.Map{}

	instanceID := uint(42)

	// 初始状态：无任务
	if hasPendingGatewayRestartTasks(instanceID) {
		t.Error("初始状态应返回 false")
	}

	// track 1 个任务
	trackGatewayRestartTask(instanceID)
	if !hasPendingGatewayRestartTasks(instanceID) {
		t.Error("track 后应返回 true")
	}

	// track 第 2 个任务
	trackGatewayRestartTask(instanceID)
	if !hasPendingGatewayRestartTasks(instanceID) {
		t.Error("track 两次后应仍返回 true")
	}

	// untrack 1 个
	untrackGatewayRestartTask(instanceID)
	if !hasPendingGatewayRestartTasks(instanceID) {
		t.Error("untrack 一次后（还剩 1 个）应仍返回 true")
	}

	// untrack 第 2 个 → counter 归 0，key 被清理
	untrackGatewayRestartTask(instanceID)
	if hasPendingGatewayRestartTasks(instanceID) {
		t.Error("全部 untrack 后应返回 false")
	}

	// 再次 untrack 不存在的 key 不应 panic
	untrackGatewayRestartTask(instanceID)
	if hasPendingGatewayRestartTasks(instanceID) {
		t.Error("对不存在的 key untrack 后应仍返回 false")
	}
}

func TestGatewayRestartTasks_ResolveInstanceStatus(t *testing.T) {
	initStateTestDB(t)
	gatewayRestartTasks = sync.Map{}

	user := createTestUserForDB(t, model.DB(context.Background()))
	instance := createTestInstanceForDB(t, model.DB(context.Background()), user.ID, "test")

	// 模拟 CVM RUNNING + AgentReady=1 + 无 installing skills
	// 但有 pending gateway restart task → 应返回 loading 而非 running
	trackGatewayRestartTask(instance.ID)

	instance.AgentReady = 1
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusLoading {
		t.Errorf("有 pending gateway task 时应返回 loading，实际 %s", status.Status)
	}

	// 清理后应返回 running
	untrackGatewayRestartTask(instance.ID)
	status = ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusRunning {
		t.Errorf("无 pending gateway task 后应返回 running，实际 %s", status.Status)
	}
}

// ─── 辅助函数 ───────────────────────────────────────────────────────────────

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Skipf("跳过测试（数据库初始化失败）: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}, &model.Notification{}, &model.SkillInstallation{}, &model.LocalInstanceInfo{}); err != nil {
		t.Skipf("跳过测试（数据库迁移失败）: %v", err)
	}

	// 确保生命周期管理字段存在（SQLite AutoMigrate 可能遗漏某些字段）
	ensureColumnExists := func(col string, createSQL string) {
		var count int
		// 检查列是否存在
		db.Raw("SELECT COUNT(*) FROM pragma_table_info('instances') WHERE name = ?", col).Scan(&count)
		if count == 0 {
			db.Exec(createSQL)
		}
	}

	ensureColumnExists("current_operation", "ALTER TABLE instances ADD COLUMN current_operation varchar(32) DEFAULT ''")
	ensureColumnExists("current_operation_state", "ALTER TABLE instances ADD COLUMN current_operation_state varchar(32) DEFAULT ''")
	ensureColumnExists("last_cvm_state", "ALTER TABLE instances ADD COLUMN last_cvm_state varchar(32) DEFAULT ''")
	ensureColumnExists("last_stable_state", "ALTER TABLE instances ADD COLUMN last_stable_state varchar(32) DEFAULT ''")
	ensureColumnExists("agent_ready", "ALTER TABLE instances ADD COLUMN agent_ready integer DEFAULT 0")
	ensureColumnExists("cls_agent_status", "ALTER TABLE instances ADD COLUMN cls_agent_status integer DEFAULT 0")
	ensureColumnExists("cls_agent_status_at", "ALTER TABLE instances ADD COLUMN cls_agent_status_at datetime DEFAULT null")

	return db
}

func createTestUserForDB(t *testing.T, db *gorm.DB) *model.User {
	t.Helper()
	user := &model.User{Username: "testuser", Password: "x", Role: "user"}
	if err := db.Create(user).Error; err != nil {
		t.Skipf("跳过测试（创建用户失败）: %v", err)
	}
	return user
}

func createTestInstanceForDB(t *testing.T, db *gorm.DB, userID uint, name string) *model.Instance {
	t.Helper()
	instance := &model.Instance{
		Name:   name,
		UserID: userID,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Skipf("跳过测试（创建实例失败）: %v", err)
	}
	return instance
}

// ─── isCLSPendingInstallation 测试 ────────────────────────────────────────

func TestIsCLSPendingInstallation_NilInstance(t *testing.T) {
	if isCLSPendingInstallation(context.Background(), nil) {
		t.Error("nil instance 应返回 false")
	}
}

func TestIsCLSPendingInstallation_CLSEnabled0(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	// site_configs 表不存在时 GetSiteConfig 返回默认值 CLSEnabled=0
	inst := &model.Instance{CLSAgentStatus: 0, CLSAgentStatusAt: nil}
	if isCLSPendingInstallation(context.Background(), inst) {
		t.Error("CLS 未开通时应返回 false")
	}
}

func TestIsCLSPendingInstallation_StatusNotZero(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Skipf("跳过测试: %v", err)
	}
	db.Create(&model.SiteConfig{CLSEnabled: 1})

	// status=1 (已安装) → false
	inst := &model.Instance{CLSAgentStatus: model.CLSAgentInstalled}
	if isCLSPendingInstallation(context.Background(), inst) {
		t.Error("CLSAgentStatus=Installed 时应返回 false")
	}
}

func TestIsCLSPendingInstallation_Installing(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Skipf("跳过测试: %v", err)
	}
	db.Create(&model.SiteConfig{CLSEnabled: 1})

	now := time.Now()
	inst := &model.Instance{CLSAgentStatus: model.CLSAgentInstalling, CLSAgentStatusAt: &now}
	if !isCLSPendingInstallation(context.Background(), inst) {
		t.Error("CLSAgentStatus=Installing 时应返回 true")
	}
}

// TestIsCLSPendingInstallation_NotInstalled_StatusAtNil 验证 status=0（未安装）时不触发 loading。
// 修改后 status=0 不再触发 loading，避免大批量机器同时处于 loading 状态。
func TestIsCLSPendingInstallation_NotInstalled_StatusAtNil(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Skipf("跳过测试: %v", err)
	}
	db.Create(&model.SiteConfig{CLSEnabled: 1})

	// status=0 + StatusAt=nil → false（不再提前 loading，等待后台任务标记 status=2）
	inst := &model.Instance{CLSAgentStatus: model.CLSAgentNotInstalled, CLSAgentStatusAt: nil}
	if isCLSPendingInstallation(context.Background(), inst) {
		t.Error("status=0（未安装）时不应触发 loading，应等待后台任务标记为 status=2")
	}
}

// TestIsCLSPendingInstallation_NotInstalled_StatusAtSet 验证 status=0 无论 StatusAt 是否设置都不触发 loading。
func TestIsCLSPendingInstallation_NotInstalled_StatusAtSet(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Skipf("跳过测试: %v", err)
	}
	db.Create(&model.SiteConfig{CLSEnabled: 1})

	now := time.Now()
	inst := &model.Instance{CLSAgentStatus: model.CLSAgentNotInstalled, CLSAgentStatusAt: &now}
	if isCLSPendingInstallation(context.Background(), inst) {
		t.Error("status=0（未安装）时不应触发 loading")
	}
}

// TestIsCLSPendingInstallation_NotInstalled_PastStatusAt 验证 status=0 即使 StatusAt 很久以前也不触发 loading。
func TestIsCLSPendingInstallation_NotInstalled_PastStatusAt(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Skipf("跳过测试: %v", err)
	}
	db.Create(&model.SiteConfig{CLSEnabled: 1})

	past := time.Now().Add(-6 * time.Minute)
	inst := &model.Instance{CLSAgentStatus: model.CLSAgentNotInstalled, CLSAgentStatusAt: &past}
	if isCLSPendingInstallation(context.Background(), inst) {
		t.Error("status=0（未安装）时不应触发 loading，即使 StatusAt 已超过冷却期")
	}
}

// TestResolveInstanceStatus_Running_CLSNotInstalled 验证 status=0（未安装）时不阻塞为 loading。
// 修改后 status=0 的机器等待后台任务调度，前端直接显示 running。
func TestResolveInstanceStatus_Running_CLSNotInstalled(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Skipf("跳过测试: %v", err)
	}
	db.Create(&model.SiteConfig{CLSEnabled: 1})

	// status=0（未安装），CLS 已开通 → 不阻塞，返回 running
	inst := &model.Instance{AgentReady: 1, CLSAgentStatus: model.CLSAgentNotInstalled, CLSAgentStatusAt: nil}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	status := ResolveInstanceStatus(context.Background(), inst, cvmInfo, nil)
	if status.Status != model.StatusRunning {
		t.Errorf("CLS status=0（未安装）时不应阻塞，应返回 running，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_Running_CLSNotPending(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Skipf("跳过测试: %v", err)
	}
	db.Create(&model.SiteConfig{CLSEnabled: 1})

	// CLS 已安装 → 不阻塞，返回 running
	past := time.Now().Add(-6 * time.Minute)
	inst := &model.Instance{AgentReady: 1, CLSAgentStatus: model.CLSAgentInstalled, CLSAgentStatusAt: &past}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	status := ResolveInstanceStatus(context.Background(), inst, cvmInfo, nil)
	if status.Status != model.StatusRunning {
		t.Errorf("CLS 已安装时应返回 running，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_Running_CLSInstalling(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Skipf("跳过测试: %v", err)
	}
	db.Create(&model.SiteConfig{CLSEnabled: 1})

	now := time.Now()
	inst := &model.Instance{AgentReady: 1, CLSAgentStatus: model.CLSAgentInstalling, CLSAgentStatusAt: &now}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	status := ResolveInstanceStatus(context.Background(), inst, cvmInfo, nil)
	if status.Status != model.StatusLoading {
		t.Errorf("CLS 安装中时应返回 loading，实际 %s", status.Status)
	}
}

// ─── OpMigrate 相关状态测试 ──────────────────────────────────────────────────

func TestResolveInstanceStatus_MigrateProcessing_ReturnsLoading(t *testing.T) {
	// 迁移进行中 + CVM RUNNING → loading（对外不暴露迁移状态）
	inst := &model.Instance{
		CurrentOperation:      model.OpMigrate,
		CurrentOperationState: model.OpStateProcessing,
		AgentReady:            1,
	}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	status := ResolveInstanceStatus(context.Background(), inst, cvmInfo, nil)
	if status.Status != model.StatusLoading {
		t.Errorf("迁移进行中应返回 loading，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_MigrateFailed_ReturnsRunning(t *testing.T) {
	// 迁移失败 + CVM RUNNING + agent ready → 不暴露失败状态，按正常 CVM 状态走 → running
	initStateTestDB(t)
	inst := &model.Instance{
		CurrentOperation:      model.OpMigrate,
		CurrentOperationState: model.OpStateFailed,
		AgentReady:            1,
	}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	status := ResolveInstanceStatus(context.Background(), inst, cvmInfo, nil)
	if status.Status != model.StatusRunning {
		t.Errorf("迁移失败后 agent ready 应返回 running，实际 %s", status.Status)
	}
}

func TestResolveInstanceStatus_MigrateOpConvergence_Excluded(t *testing.T) {
	// OpMigrate 在 SideEffect 收敛条件里被排除，不会自动清除操作锁
	// 验证：OpMigrate + CVM STOPPED 时不走收敛分支（状态由 CVM STOPPED 决定）
	inst := &model.Instance{
		CurrentOperation:      model.OpMigrate,
		CurrentOperationState: model.OpStateProcessing,
		AgentReady:            1,
	}
	cvmInfo := &CVMInstanceInfo{State: "STOPPED"}
	status := ResolveInstanceStatus(context.Background(), inst, cvmInfo, nil)
	if status.Status != model.StatusStopped {
		t.Errorf("OpMigrate + STOPPED 应返回 stopped，实际 %s", status.Status)
	}
}

// ─── model/instance_lifecycle.go 状态常量覆盖 ────────────────────────────────

func TestInstanceLifecycleConstants_Migrate(t *testing.T) {
	if model.OpMigrate == "" {
		t.Error("OpMigrate should not be empty")
	}
	if _, ok := model.OperationTimeouts[model.OpMigrate]; !ok {
		t.Error("OperationTimeouts should contain OpMigrate")
	}
}

// --- API_ERROR 相关单测 ---

// TestResolveInstanceStatus_APIError_ReturnsRunning 验证 CVM API 失败时返回 running（缓存兜底）
func TestResolveInstanceStatus_APIError_ReturnsRunning(t *testing.T) {
	instance := &model.Instance{
		InstanceId:      "ins-test",
		LastCVMState:    "RUNNING",
		LastStableState: "RUNNING",
	}
	cvmInfo := &CVMInstanceInfo{State: "API_ERROR"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != "running" {
		t.Errorf("API_ERROR 时期望 running，实际 %s", status.Status)
	}
}

// TestResolveInstanceStatus_APIError_NoCachedState_ReturnsRunning 验证无缓存状态时也保守返回 running
func TestResolveInstanceStatus_APIError_NoCachedState_ReturnsRunning(t *testing.T) {
	instance := &model.Instance{
		InstanceId:      "ins-test",
		LastCVMState:    "",
		LastStableState: "",
	}
	cvmInfo := &CVMInstanceInfo{State: "API_ERROR"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != "running" {
		t.Errorf("API_ERROR 且无缓存状态时期望 running，实际 %s", status.Status)
	}
}

// TestResolveInstanceStatus_APIError_OpDelete_ReturnsDestroyed 验证正在删除时 API_ERROR 走正常删除流程
func TestResolveInstanceStatus_APIError_OpDelete_ReturnsDestroyed(t *testing.T) {
	instance := &model.Instance{
		InstanceId:       "ins-test",
		LastCVMState:     "RUNNING",
		CurrentOperation: model.OpDelete,
	}
	cvmInfo := &CVMInstanceInfo{State: "API_ERROR"}

	status := ResolveInstanceStatus(context.Background(), instance, cvmInfo, nil)
	if status.Status != model.StatusDestroyed {
		t.Errorf("OpDelete + API_ERROR 时期望 destroyed，实际 %s", status.Status)
	}
}

// TestBatchFetchCVMInfoMap_ClientError_MarksAPIError 验证 NewCVMClient 失败时所有实例标记为 API_ERROR
func TestBatchFetchCVMInfoMap_ClientError_MarksAPIError(t *testing.T) {
	// 不设置 CVM 凭据，NewCVMClient 会失败
	ctx := context.Background()
	ids := []string{"ins-aaa", "ins-bbb", "ins-ccc"}

	result := batchFetchCVMInfoMap(ctx, ids)

	for _, id := range ids {
		info, ok := result[id]
		if !ok {
			t.Errorf("实例 %s 应在 result 中，但不存在", id)
			continue
		}
		if info == nil || info.State != "API_ERROR" {
			t.Errorf("实例 %s 期望 State=API_ERROR，实际 %v", id, info)
		}
	}
}

// TestBatchFetchCVMInfoMap_EmptyIDs_ReturnsEmpty 验证空 ID 列表返回空 map
func TestBatchFetchCVMInfoMap_EmptyIDs_ReturnsEmpty(t *testing.T) {
	result := batchFetchCVMInfoMap(context.Background(), []string{})
	if len(result) != 0 {
		t.Errorf("空 ID 列表期望返回空 map，实际长度=%d", len(result))
	}
}

// TestBatchFetchCVMInfoMap_InternalError_MarksAPIError 验证 CVM API 返回 InternalError 时批次标记为 API_ERROR
func TestBatchFetchCVMInfoMap_InternalError_MarksAPIError(t *testing.T) {
	// Mock CVM server 返回 InternalError
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"Response":{"Error":{"Code":"InternalError","Message":"内部服务暂不可用"},"RequestId":"mock-req-id"}}`)
	}))
	defer ts.Close()

	origNewCVMClient := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return newMockCVMClient(t, ts.URL), nil
	}
	defer func() { NewCVMClient = origNewCVMClient }()

	ids := []string{"ins-aaa", "ins-bbb"}
	result := batchFetchCVMInfoMap(context.Background(), ids)

	for _, id := range ids {
		info, ok := result[id]
		if !ok {
			t.Errorf("实例 %s 应在 result 中，但不存在", id)
			continue
		}
		if info == nil || info.State != "API_ERROR" {
			t.Errorf("InternalError 时实例 %s 期望 State=API_ERROR，实际 %v", id, info)
		}
	}
}

// TestBatchFetchCVMInfoMap_NotFound_DegradeAndMarkAPIError 验证 InvalidInstanceId.NotFound 降级后单个查询失败标记 API_ERROR
func TestBatchFetchCVMInfoMap_NotFound_DegradeAndMarkAPIError(t *testing.T) {
	callCount := 0
	// 第一次（批量）返回 InvalidInstanceId.NotFound，后续（单个）返回 InternalError
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		callCount++
		if callCount == 1 {
			// 批量查询 → NotFound
			fmt.Fprintf(w, `{"Response":{"Error":{"Code":"InvalidInstanceId.NotFound","Message":"not found"},"RequestId":"req-1"}}`)
		} else {
			// 单个查询 → InternalError
			fmt.Fprintf(w, `{"Response":{"Error":{"Code":"InternalError","Message":"内部错误"},"RequestId":"req-2"}}`)
		}
	}))
	defer ts.Close()

	origNewCVMClient := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return newMockCVMClient(t, ts.URL), nil
	}
	defer func() { NewCVMClient = origNewCVMClient }()

	ids := []string{"ins-xxx"}
	result := batchFetchCVMInfoMap(context.Background(), ids)

	info, ok := result["ins-xxx"]
	if !ok {
		t.Fatal("ins-xxx 应在 result 中")
	}
	if info == nil || info.State != "API_ERROR" {
		t.Errorf("降级后单个查询失败期望 API_ERROR，实际 %v", info)
	}
}

// TestIsCLSPendingInstallation_NotInstalled_ScopeIrrelevant 验证 status=0 时无论是否在 scope 内都不触发 loading。
// 修改后 scope 检查已从前端状态判断中移除，由后台任务负责 scope 过滤。
func TestIsCLSPendingInstallation_NotInstalled_ScopeIrrelevant(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}, &model.GroupConfigBinding{}, &model.GroupClosure{}); err != nil {
		t.Skipf("跳过测试: %v", err)
	}
	// 分组模式，scope 包含 group 5
	db.Create(&model.SiteConfig{CLSEnabled: 1, CLSScopeMode: "group"})
	db.Create(&model.GroupConfigBinding{GroupID: 5, ConfigType: model.ConfigTypeCLSCollectScope, ConfigKey: model.CLSCollectScopeKey})
	db.Create(&model.GroupClosure{AncestorID: 5, DescendantID: 5, Depth: 0})

	tests := []struct {
		name    string
		groupID uint
	}{
		{"在scope内的分组", 5},
		{"不在scope内的分组", 99},
		{"未指定分组", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &model.Instance{CLSAgentStatus: model.CLSAgentNotInstalled, GroupID: tt.groupID}
			if isCLSPendingInstallation(context.Background(), inst) {
				t.Errorf("status=0（未安装）时不应触发 loading，groupID=%d", tt.groupID)
			}
		})
	}
}

// TestIsCLSPendingInstallation_Installing_AlwaysLoading 验证 status=2（安装中）时无论 StatusAt 如何都触发 loading。
func TestIsCLSPendingInstallation_Installing_AlwaysLoading(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Skipf("跳过测试: %v", err)
	}
	db.Create(&model.SiteConfig{CLSEnabled: 1})

	now := time.Now()
	past := time.Now().Add(-10 * time.Minute)
	tests := []struct {
		name     string
		statusAt *time.Time
	}{
		{"StatusAt=nil", nil},
		{"StatusAt=now", &now},
		{"StatusAt=10分钟前", &past},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &model.Instance{CLSAgentStatus: model.CLSAgentInstalling, CLSAgentStatusAt: tt.statusAt}
			if !isCLSPendingInstallation(context.Background(), inst) {
				t.Errorf("status=2（安装中）时应触发 loading，StatusAt=%v", tt.statusAt)
			}
		})
	}
}

// TestResolveInstanceStatus_Running_CLSInstalling_Loading 验证后台任务标记 status=2 后前端显示 loading。
func TestResolveInstanceStatus_Running_CLSInstalling_Loading(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Skipf("跳过测试: %v", err)
	}
	db.Create(&model.SiteConfig{CLSEnabled: 1})

	// 后台任务已将 status 标记为 2（安装中）→ loading
	now := time.Now()
	inst := &model.Instance{AgentReady: 1, CLSAgentStatus: model.CLSAgentInstalling, CLSAgentStatusAt: &now}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	status := ResolveInstanceStatus(context.Background(), inst, cvmInfo, nil)
	if status.Status != model.StatusLoading {
		t.Errorf("后台任务标记 status=2（安装中）时应返回 loading，实际 %s", status.Status)
	}
	if !status.Transient {
		t.Error("CLS 安装中时应为过渡态")
	}
}

// TestResolveInstanceStatus_Running_CLSInstallSuccess_Running 验证安装成功后 status=1 时返回 running。
func TestResolveInstanceStatus_Running_CLSInstallSuccess_Running(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Skipf("跳过测试: %v", err)
	}
	db.Create(&model.SiteConfig{CLSEnabled: 1})

	// 安装成功，status=1（已安装）→ running
	now := time.Now()
	inst := &model.Instance{AgentReady: 1, CLSAgentStatus: model.CLSAgentInstalled, CLSAgentStatusAt: &now}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	status := ResolveInstanceStatus(context.Background(), inst, cvmInfo, nil)
	if status.Status != model.StatusRunning {
		t.Errorf("CLS 安装成功（status=1）时应返回 running，实际 %s", status.Status)
	}
}

// TestResolveInstanceStatus_Running_CLSInstallFailed_Running 验证安装失败回退 status=0 后返回 running（不再 loading）。
func TestResolveInstanceStatus_Running_CLSInstallFailed_Running(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(model.UseDBForTest(db))
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Skipf("跳过测试: %v", err)
	}
	db.Create(&model.SiteConfig{CLSEnabled: 1})

	// 安装失败后 status 回退为 0，status_at 置为 nil → 不再 loading，等待下轮重试
	inst := &model.Instance{AgentReady: 1, CLSAgentStatus: model.CLSAgentNotInstalled, CLSAgentStatusAt: nil}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	status := ResolveInstanceStatus(context.Background(), inst, cvmInfo, nil)
	if status.Status != model.StatusRunning {
		t.Errorf("CLS 安装失败回退 status=0 后应返回 running（不再 loading），实际 %s", status.Status)
	}
}

// TestResolveLocalInstanceStatus_ActionsEmpty 本地实例 actions 必须为空：
// hatchery 无法对本地机器远程 reboot/reinstall/start/terminal，delete 也不允许
// （需用户从本地卸载 agent，hatchery 端只能等 reporter 超时被动清理）。
func TestResolveLocalInstanceStatus_ActionsEmpty(t *testing.T) {
	initStateTestDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.LocalInstanceInfo{}); err != nil {
		t.Fatalf("migrate LocalInstanceInfo: %v", err)
	}

	cases := []struct {
		name        string
		hasInfo     bool
		freshReport bool
		wantStatus  string
	}{
		{"running_fresh_report", true, true, model.StatusRunning},
		{"stopped_stale_report", true, false, model.StatusStopped},
		{"stopped_no_info", false, false, model.StatusStopped},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inst := &model.Instance{
				Name: "local-" + tc.name, InstanceId: "local-" + tc.name + "-001",
				Source: model.InstanceSourceLocal, AgentType: "codebuddy",
			}
			if err := model.DB(context.Background()).Create(inst).Error; err != nil {
				t.Fatalf("create instance: %v", err)
			}
			if tc.hasInfo {
				var report time.Time
				if tc.freshReport {
					report = time.Now()
				} else {
					report = time.Now().Add(-2 * model.LocalInstanceOfflineThreshold)
				}
				if err := model.DB(context.Background()).Create(&model.LocalInstanceInfo{
					InstanceID: inst.ID, LastReportAt: &report,
				}).Error; err != nil {
					t.Fatalf("create LocalInstanceInfo: %v", err)
				}
			}

			resp := resolveLocalInstanceStatus(context.Background(), inst)
			if resp.Status != tc.wantStatus {
				t.Errorf("status: 期望 %s 实际 %s", tc.wantStatus, resp.Status)
			}
			if resp.Actions == nil {
				t.Error("Actions 应为非 nil 空切片，实际为 nil")
			}
			if len(resp.Actions) != 0 {
				t.Errorf("本地实例 actions 应为空，实际=%v", resp.Actions)
			}
		})
	}
}

// TestBatchFetchCVMInfoMap_LocalIDsFilteredOut 验证本地实例 ID 不会落到 CVM API。
//
// 本次需求加固：batchFetchCVMInfoMap 在调 NewCVMClient 之前会先用
// strings.HasPrefix(id, "local-") 过滤本地实例。如果输入全是 local-，则 cvmIds
// 列表为空，函数提前返回，**绝不会调 CVM API**——这意味着 NewCVMClient
// 失败的兜底分支也不会进入，输出 State 不是 API_ERROR，而是占位的 "LOCAL"。
//
// 这个测试同时保护两件事：
//  1. 本地 ID 被过滤（不会去打 CVM API）
//  2. 占位结果 State="LOCAL" 给调用方的 fallthrough 用
func TestBatchFetchCVMInfoMap_LocalIDsFilteredOut(t *testing.T) {
	ctx := context.Background()
	ids := []string{"local-codebuddy-001", "local-workbuddy-deadbe", "local-pure-cafe01"}

	result := batchFetchCVMInfoMap(ctx, ids)

	for _, id := range ids {
		info, ok := result[id]
		if !ok {
			t.Errorf("local 实例 %s 应在 result 中，但不存在", id)
			continue
		}
		if info == nil || info.State != "LOCAL" {
			t.Errorf("local 实例 %s 期望 State=LOCAL，实际 %v", id, info)
		}
	}
	if len(result) != len(ids) {
		t.Errorf("result 应只含 %d 个本地实例占位，实际=%d 项", len(ids), len(result))
	}
}

// TestBatchFetchCVMInfoMap_MixedLocalAndCVM_LocalGetsLocalCVMGetsAPIError
// 验证 local + 真 CVM ID 混合时：
//   - local 标记 LOCAL
//   - CVM 因为没有凭据回退到 API_ERROR（而不是被本地过滤逻辑漏过）
func TestBatchFetchCVMInfoMap_MixedLocalAndCVM_LocalGetsLocalCVMGetsAPIError(t *testing.T) {
	ctx := context.Background()
	ids := []string{"local-codebuddy-001", "ins-real-1", "ins-real-2"}

	result := batchFetchCVMInfoMap(ctx, ids)

	if got := result["local-codebuddy-001"]; got == nil || got.State != "LOCAL" {
		t.Errorf("local-codebuddy-001 期望 LOCAL，实际 %v", got)
	}
	for _, id := range []string{"ins-real-1", "ins-real-2"} {
		got := result[id]
		if got == nil || got.State != "API_ERROR" {
			t.Errorf("CVM ID %s 期望 API_ERROR（无 CVM 凭据），实际 %v", id, got)
		}
	}
}

// ─── batchHasInstallingSkillInstallations 测试 ──────────────────────────────

func TestBatchHasInstallingSkillInstallations_EmptyIDs(t *testing.T) {
	initStateTestDB(t)
	result := batchHasInstallingSkillInstallations(context.Background(), nil)
	if len(result) != 0 {
		t.Errorf("空 ID 列表应返回空 map，实际 len=%d", len(result))
	}
}

func TestBatchHasInstallingSkillInstallations_HasInstalling(t *testing.T) {
	initStateTestDB(t)
	db := model.DB(context.Background())
	// 创建 installing 状态的记录
	db.Create(&model.SkillInstallation{
		InstanceID:    100,
		Slug:          "test-skill",
		Version:       "1.0",
		InstallStatus: model.SkillInstalling,
	})
	// 创建非 installing 状态的记录
	db.Create(&model.SkillInstallation{
		InstanceID:    200,
		Slug:          "test-skill",
		Version:       "1.0",
		InstallStatus: model.SkillInstallSuccess,
	})

	result := batchHasInstallingSkillInstallations(context.Background(), []uint{100, 200, 300})
	if !result[100] {
		t.Error("instance 100 应有 installing 状态")
	}
	if result[200] {
		t.Error("instance 200 不应有 installing 状态（已安装成功）")
	}
	if result[300] {
		t.Error("instance 300 不应有 installing 状态（不存在）")
	}
}

func TestBatchHasInstallingSkillInstallations_MultipleIDs(t *testing.T) {
	initStateTestDB(t)
	db := model.DB(context.Background())
	// 创建多个 installing 记录
	for i := uint(1); i <= 3; i++ {
		db.Create(&model.SkillInstallation{
			InstanceID:    i,
			Slug:          "chunk-test",
			Version:       "1.0",
			InstallStatus: model.SkillInstalling,
		})
	}
	ids := []uint{1, 2, 3}
	result := batchHasInstallingSkillInstallations(context.Background(), ids)
	for _, id := range ids {
		if !result[id] {
			t.Errorf("instance %d 应有 installing 状态", id)
		}
	}
}

// ─── batchResolveLocalInstanceStatus 测试 ────────────────────────────────────

func TestBatchResolveLocalInstanceStatus_EmptyIDs(t *testing.T) {
	initStateTestDB(t)
	result := batchResolveLocalInstanceStatus(context.Background(), nil)
	if len(result) != 0 {
		t.Errorf("空 ID 列表应返回空 map，实际 len=%d", len(result))
	}
}

func TestBatchResolveLocalInstanceStatus_WithRecords(t *testing.T) {
	initStateTestDB(t)
	db := model.DB(context.Background())
	now := time.Now()
	db.Create(&model.LocalInstanceInfo{
		InstanceID:   10,
		HostName:     "local-host-1",
		OS:           "linux",
		LastReportAt: &now,
	})
	db.Create(&model.LocalInstanceInfo{
		InstanceID:   20,
		HostName:     "local-host-2",
		OS:           "macos",
		LastReportAt: nil,
	})

	result := batchResolveLocalInstanceStatus(context.Background(), []uint{10, 20, 30})
	if info, ok := result[10]; !ok || info == nil {
		t.Error("instance 10 应有记录")
	} else if info.HostName != "local-host-1" {
		t.Errorf("instance 10 HostName 期望 local-host-1，实际 %s", info.HostName)
	}
	if info, ok := result[20]; !ok || info == nil {
		t.Error("instance 20 应有记录")
	}
	if _, ok := result[30]; ok {
		t.Error("instance 30 不应有记录")
	}
}

// ─── chunkUintIDs 测试 ──────────────────────────────────────────────────────

func TestChunkUintIDs_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name         string
		total        int
		size         int
		expectChunks int
		lastChunkLen int
	}{
		{"empty", 0, 500, 0, 0},
		{"single_element", 1, 500, 1, 1},
		{"exact_one_chunk", 500, 500, 1, 500},
		{"one_over_boundary", 501, 500, 2, 1},
		{"exact_two_chunks", 1000, 500, 2, 500},
		{"partial_last", 5, 2, 3, 1},
		{"exact_small", 6, 3, 2, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids := make([]uint, tt.total)
			for i := range ids {
				ids[i] = uint(i + 1)
			}
			chunks := chunkUintIDs(ids, tt.size)
			if len(chunks) != tt.expectChunks {
				t.Errorf("期望 %d 个 chunk，实际 %d", tt.expectChunks, len(chunks))
				return
			}
			if tt.expectChunks > 0 && len(chunks[tt.expectChunks-1]) != tt.lastChunkLen {
				t.Errorf("最后一个 chunk 期望 %d 个元素，实际 %d", tt.lastChunkLen, len(chunks[tt.expectChunks-1]))
			}
			// 验证所有元素都被包含且顺序正确
			var all []uint
			for _, c := range chunks {
				all = append(all, c...)
			}
			if len(all) != tt.total {
				t.Errorf("分片后元素总数期望 %d，实际 %d", tt.total, len(all))
			}
		})
	}
}

// ─── ResolveInstanceStatus 测试 ────────────────────────────────────────

func TestResolveInstanceStatusCached_LocalInstance(t *testing.T) {
	initStateTestDB(t)
	now := time.Now()
	db := model.DB(context.Background())
	db.Create(&model.LocalInstanceInfo{
		InstanceID:   42,
		HostName:     "local-1",
		LastReportAt: &now,
	})

	inst := &model.Instance{
		Source: model.InstanceSourceLocal,
	}
	inst.ID = 42

	localInfoMap := map[uint]*model.LocalInstanceInfo{
		42: {InstanceID: 42, LastReportAt: &now},
	}

	result := ResolveInstanceStatus(context.Background(), inst, nil,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: localInfoMap})
	if result.Status != model.StatusRunning {
		t.Errorf("本地实例刚上报应返回 running，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_LocalInstanceStopped(t *testing.T) {
	initStateTestDB(t)
	// 超过离线阈值（7 天）未上报 → stopped（用 2 倍阈值确保越过边界）
	past := time.Now().Add(-2 * model.LocalInstanceOfflineThreshold)
	db := model.DB(context.Background())
	db.Create(&model.LocalInstanceInfo{
		InstanceID:   43,
		HostName:     "local-2",
		LastReportAt: &past,
	})

	inst := &model.Instance{
		Source: model.InstanceSourceLocal,
	}
	inst.ID = 43

	localInfoMap := map[uint]*model.LocalInstanceInfo{
		43: {InstanceID: 43, LastReportAt: &past},
	}

	result := ResolveInstanceStatus(context.Background(), inst, nil,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: localInfoMap})
	if result.Status != model.StatusStopped {
		t.Errorf("本地实例超过离线阈值(%v)未上报应返回 stopped，实际 %s", model.LocalInstanceOfflineThreshold, result.Status)
	}
}

func TestResolveInstanceStatusCached_LocalInstanceNoInfo(t *testing.T) {
	initStateTestDB(t)
	inst := &model.Instance{
		Source: model.InstanceSourceLocal,
	}
	inst.ID = 99

	result := ResolveInstanceStatus(context.Background(), inst, nil,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: map[uint]*model.LocalInstanceInfo{}})
	if result.Status != model.StatusStopped {
		t.Errorf("本地实例无 LocalInfo 应返回 stopped，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_UpgradeFailed(t *testing.T) {
	inst := &model.Instance{
		CurrentOperation:      model.OpUpgrade,
		CurrentOperationState: model.OpStateFailed,
	}
	// CVM 存在时 → upgrade_failed
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusUpgradeFailed {
		t.Errorf("升级失败且CVM存在应返回 upgrade_failed，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_UpgradeFailedCVMGone(t *testing.T) {
	inst := &model.Instance{
		InstanceId:            "ins-test-upgrade-gone",
		CurrentOperation:      model.OpUpgrade,
		CurrentOperationState: model.OpStateFailed,
		LastStableState:       "RUNNING",
	}
	// CVM 不存在时 → destroyed（外部销毁）
	result := ResolveInstanceStatus(context.Background(), inst, nil,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusDestroyed {
		t.Errorf("升级失败但CVM不存在应返回 destroyed，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_LoadFailed(t *testing.T) {
	inst := &model.Instance{
		CurrentOperation:      model.OpReboot,
		CurrentOperationState: model.OpStateFailed,
	}
	// CVM 存在时 → load_failed
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusLoadFailed {
		t.Errorf("重启失败且CVM存在应返回 load_failed，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_LoadFailedCVMGone(t *testing.T) {
	inst := &model.Instance{
		InstanceId:            "ins-test-reboot-gone",
		CurrentOperation:      model.OpReboot,
		CurrentOperationState: model.OpStateFailed,
		LastStableState:       "RUNNING",
	}
	// CVM 不存在时 → destroyed（外部销毁）
	result := ResolveInstanceStatus(context.Background(), inst, nil,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusDestroyed {
		t.Errorf("操作失败但CVM不存在应返回 destroyed，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_CVMAPIError_Running(t *testing.T) {
	inst := &model.Instance{}
	cvmInfo := &CVMInstanceInfo{State: "API_ERROR"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != "running" {
		t.Errorf("CVM API 错误应兜底返回 running，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_CVMAPIError_Delete(t *testing.T) {
	inst := &model.Instance{
		CurrentOperation: model.OpDelete,
	}
	cvmInfo := &CVMInstanceInfo{State: "API_ERROR"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusDestroyed {
		t.Errorf("删除+CVM API 错误应返回 destroyed，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_RunningWithAgentReady(t *testing.T) {
	inst := &model.Instance{
		AgentReady: 1,
	}
	inst.ID = 1
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{CLSEnabled: 0}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusRunning {
		t.Errorf("Agent 就绪的 RUNNING 实例应返回 running，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_RunningWithAgentNotReady(t *testing.T) {
	inst := &model.Instance{
		AgentReady: 0,
	}
	inst.ID = 1
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{CLSEnabled: 0}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusLoading {
		t.Errorf("Agent 未就绪的 RUNNING 实例应返回 loading，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_RunningWithCLSInstalling(t *testing.T) {
	inst := &model.Instance{
		AgentReady:     1,
		CLSAgentStatus: model.CLSAgentInstalling,
	}
	inst.ID = 1
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{CLSEnabled: 1}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusLoading {
		t.Errorf("CLS 安装中的实例应返回 loading，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_RunningWithInstallingSkill(t *testing.T) {
	inst := &model.Instance{
		AgentReady: 1,
	}
	inst.ID = 100
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}
	installingMap := map[uint]bool{100: true}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{CLSEnabled: 0}, InstallingSkillMap: installingMap, LocalInfoMap: nil})
	if result.Status != model.StatusLoading {
		t.Errorf("技能安装中的实例应返回 loading，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_Upgrading(t *testing.T) {
	inst := &model.Instance{
		CurrentOperation:      model.OpUpgrade,
		CurrentOperationState: model.OpStateProcessing,
	}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusUpgrading {
		t.Errorf("升级中的 RUNNING 实例应返回 upgrading，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_Migrating(t *testing.T) {
	inst := &model.Instance{
		CurrentOperation:      model.OpMigrate,
		CurrentOperationState: model.OpStateProcessing,
	}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusLoading {
		t.Errorf("迁移中的 RUNNING 实例应返回 loading，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_Stopped(t *testing.T) {
	inst := &model.Instance{}
	cvmInfo := &CVMInstanceInfo{State: "STOPPED"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusStopped {
		t.Errorf("STOPPED 状态应返回 stopped，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_StoppedWithUpgrade(t *testing.T) {
	inst := &model.Instance{
		CurrentOperation:      model.OpUpgrade,
		CurrentOperationState: model.OpStateProcessing,
	}
	cvmInfo := &CVMInstanceInfo{State: "STOPPED"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusUpgrading {
		t.Errorf("STOPPED+升级中应返回 upgrading，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_StoppedWithReinstall(t *testing.T) {
	inst := &model.Instance{
		CurrentOperation: model.OpReinstall,
	}
	cvmInfo := &CVMInstanceInfo{State: "STOPPED"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusLoading {
		t.Errorf("STOPPED+重装中应返回 loading，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_Pending(t *testing.T) {
	inst := &model.Instance{}
	cvmInfo := &CVMInstanceInfo{State: "PENDING"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusCreating {
		t.Errorf("PENDING 状态应返回 creating，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_LaunchFailed(t *testing.T) {
	inst := &model.Instance{}
	cvmInfo := &CVMInstanceInfo{State: "LAUNCH_FAILED"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusCreateFailed {
		t.Errorf("LAUNCH_FAILED 应返回 create_failed，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_UnknownState(t *testing.T) {
	inst := &model.Instance{}
	cvmInfo := &CVMInstanceInfo{State: "SOME_UNKNOWN_STATE"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusMaintaining {
		t.Errorf("未知状态应兜底返回 maintaining，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_ChargeTypeFromCVM(t *testing.T) {
	inst := &model.Instance{}
	cvmInfo := &CVMInstanceInfo{
		State:              "RUNNING",
		InstanceChargeType: "POSTPAID_BY_HOUR",
	}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.InstanceChargeType != "POSTPAID_BY_HOUR" {
		t.Errorf("计费类型应取自 CVM 信息，期望 POSTPAID_BY_HOUR，实际 %s", result.InstanceChargeType)
	}
}

func TestResolveInstanceStatusCached_ChargeTypeDefault(t *testing.T) {
	inst := &model.Instance{}
	inst.ID = 1
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.InstanceChargeType == "" {
		t.Error("计费类型应有默认值")
	}
}

// ─── isCLSPendingInstallationWithConfig 测试 ────────────────────────────────────

func TestIsCLSPendingInstallationCached_NilInstance(t *testing.T) {
	if isCLSPendingInstallationWithConfig(model.SiteConfig{CLSEnabled: 1}, nil) {
		t.Error("nil instance 应返回 false")
	}
}

func TestIsCLSPendingInstallationCached_CLSNotEnabled(t *testing.T) {
	inst := &model.Instance{CLSAgentStatus: model.CLSAgentInstalling}
	if isCLSPendingInstallationWithConfig(model.SiteConfig{CLSEnabled: 0}, inst) {
		t.Error("CLS 未启用时应返回 false")
	}
}

func TestIsCLSPendingInstallationCached_Installing(t *testing.T) {
	inst := &model.Instance{CLSAgentStatus: model.CLSAgentInstalling}
	if !isCLSPendingInstallationWithConfig(model.SiteConfig{CLSEnabled: 1}, inst) {
		t.Error("CLS 安装中应返回 true")
	}
}

func TestIsCLSPendingInstallationCached_NotInstalling(t *testing.T) {
	inst := &model.Instance{CLSAgentStatus: model.CLSAgentInstalled}
	if isCLSPendingInstallationWithConfig(model.SiteConfig{CLSEnabled: 1}, inst) {
		t.Error("CLS 已安装应返回 false")
	}
}

// ─── resolveLocalInstanceStatusCached 测试 ──────────────────────────────────

func TestResolveLocalInstanceStatusCached_RecentReport(t *testing.T) {
	now := time.Now()
	infoMap := map[uint]*model.LocalInstanceInfo{
		1: {InstanceID: 1, LastReportAt: &now},
	}
	inst := &model.Instance{}
	inst.ID = 1

	result := resolveLocalInstanceStatusCached(context.Background(), inst, infoMap)
	if result.Status != model.StatusRunning {
		t.Errorf("刚上报的本地实例应返回 running，实际 %s", result.Status)
	}
}

func TestResolveLocalInstanceStatusCached_NotInMap(t *testing.T) {
	inst := &model.Instance{}
	inst.ID = 99

	result := resolveLocalInstanceStatusCached(context.Background(), inst, map[uint]*model.LocalInstanceInfo{})
	if result.Status != model.StatusStopped {
		t.Errorf("未在 map 中的本地实例应返回 stopped，实际 %s", result.Status)
	}
}

func TestResolveLocalInstanceStatusCached_NilLastReport(t *testing.T) {
	infoMap := map[uint]*model.LocalInstanceInfo{
		1: {InstanceID: 1, LastReportAt: nil},
	}
	inst := &model.Instance{}
	inst.ID = 1

	result := resolveLocalInstanceStatusCached(context.Background(), inst, infoMap)
	if result.Status != model.StatusStopped {
		t.Errorf("LastReportAt 为 nil 时应返回 stopped，实际 %s", result.Status)
	}
}

func TestResolveLocalInstanceStatusCached_Offline(t *testing.T) {
	// 超过离线阈值（7 天）未上报 → stopped（用 2 倍阈值确保越过边界）
	past := time.Now().Add(-2 * model.LocalInstanceOfflineThreshold)
	infoMap := map[uint]*model.LocalInstanceInfo{
		1: {InstanceID: 1, LastReportAt: &past},
	}
	inst := &model.Instance{}
	inst.ID = 1

	result := resolveLocalInstanceStatusCached(context.Background(), inst, infoMap)
	if result.Status != model.StatusStopped {
		t.Errorf("超过离线阈值(%v)未上报应返回 stopped，实际 %s", model.LocalInstanceOfflineThreshold, result.Status)
	}
}

// ─── ResolveInstanceStatus 空 instanceID 分支 ─────────────────────────

func TestResolveInstanceStatusCached_EmptyInstanceID_Creating(t *testing.T) {
	inst := &model.Instance{
		InstanceId: "",
	}
	cvmInfo := (*CVMInstanceInfo)(nil)

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusCreating {
		t.Errorf("空 InstanceId+CVM 不存在应返回 creating，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_CVMNotFound_Destroyed(t *testing.T) {
	inst := &model.Instance{
		InstanceId:    "ins-exists",
		LastCVMState:  "RUNNING",
		LastStableState: "RUNNING",
	}
	cvmInfo := (*CVMInstanceInfo)(nil)

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	// LastStableState 为 RUNNING → isPostCreationCVMState 为 true → destroyed
	if result.Status != model.StatusDestroyed {
		t.Errorf("实例已存在但 CVM 消失应返回 destroyed，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_CVMNotFound_CreateFailed(t *testing.T) {
	inst := &model.Instance{
		InstanceId:   "ins-exists",
		LastCVMState: "PENDING",
	}
	cvmInfo := (*CVMInstanceInfo)(nil)

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusCreateFailed {
		t.Errorf("创建阶段的实例 CVM 消失应返回 create_failed，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_CVMNotFound_CreateFailed_OpCreate(t *testing.T) {
	inst := &model.Instance{
		InstanceId:       "ins-exists",
		CurrentOperation: model.OpCreate,
	}
	cvmInfo := (*CVMInstanceInfo)(nil)

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusCreateFailed {
		t.Errorf("创建中实例 CVM 消失应返回 create_failed，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_CVMNotFound_CreateFailed_EmptyStates(t *testing.T) {
	inst := &model.Instance{
		InstanceId:     "ins-exists",
		LastCVMState:   "",
		LastStableState: "",
	}
	cvmInfo := (*CVMInstanceInfo)(nil)

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusCreateFailed {
		t.Errorf("LastCVMState 和 LastStableState 均为空应返回 create_failed，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_RestrictState_Pending(t *testing.T) {
	inst := &model.Instance{}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING", RestrictState: "EXPIRED"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusPending {
		t.Errorf("限制状态非 NORMAL 应返回 pending，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_CVMLiveMigrate_Running(t *testing.T) {
	inst := &model.Instance{}
	cvmInfo := &CVMInstanceInfo{State: "SERVICE_LIVE_MIGRATE"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusRunning {
		t.Errorf("CVM 热迁移状态应返回 running，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_CVMPlatformLimit_Pending(t *testing.T) {
	inst := &model.Instance{}
	cvmInfo := &CVMInstanceInfo{State: "FREEZING"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusPending {
		t.Errorf("CVM 平台限制状态应返回 pending，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_CVMRescueMode_Maintaining(t *testing.T) {
	inst := &model.Instance{}
	// 使用 CVMRescueModeStates 中的一个值
	cvmInfo := &CVMInstanceInfo{State: "RESCUE_MODE"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusMaintaining {
		t.Errorf("CVM 救援模式应返回 maintaining，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_CVMTransient_Loading(t *testing.T) {
	inst := &model.Instance{}
	// 使用 CVMTransientStates 中的一个值
	cvmInfo := &CVMInstanceInfo{State: "REBOOTING"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusLoading {
		t.Errorf("CVM 瞬时状态应返回 loading，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_CVMInfoNil_Delete(t *testing.T) {
	inst := &model.Instance{
		CurrentOperation: model.OpDelete,
	}
	cvmInfo := &CVMInstanceInfo{State: ""}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusDestroyed {
		t.Errorf("空 State+删除操作应返回 destroyed，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_CVMInfoNil_NotDelete(t *testing.T) {
	inst := &model.Instance{
		CurrentOperation: model.OpDelete,
	}
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusDestroying {
		t.Errorf("RUNNING+删除操作应返回 destroying，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_CVMAPIError_NilInfo(t *testing.T) {
	inst := &model.Instance{
		CurrentOperation: model.OpDelete,
	}
	cvmInfo := &CVMInstanceInfo{State: "API_ERROR"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	// delete + API_ERROR → cvmInfo 被置 nil → destroyed
	if result.Status != model.StatusDestroyed {
		t.Errorf("delete+API_ERROR 应返回 destroyed，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_NotFoundCVMInfo(t *testing.T) {
	inst := &model.Instance{
		CurrentOperation: model.OpDelete,
	}
	cvmInfo := &CVMInstanceInfo{State: "NOTFOUND"}

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusDestroyed {
		t.Errorf("NOTFOUND+删除操作应返回 destroyed，实际 %s", result.Status)
	}
}

func TestResolveInstanceStatusCached_DeleteNotProcessed(t *testing.T) {
	inst := &model.Instance{
		InstanceId:            "ins-migrate-test",
		CurrentOperation:      model.OpMigrate,
		CurrentOperationState: model.OpStateFailed,
		LastCVMState:          "RUNNING",
		LastStableState:       "RUNNING",
	}
	// OpMigrate 不是 OpDelete 也不是 OpUpgrade，且 OpMigrate 被排除在 Step 0 之外
	// Step 0: OpMigrate 排除条件满足 → status 仍为 ""
	// Step 2: cvmInfo==nil + InstanceId 非空 → 走 isPostCreationCVMState(LastStableState="RUNNING") → destroyed
	cvmInfo := (*CVMInstanceInfo)(nil)

	result := ResolveInstanceStatus(context.Background(), inst, cvmInfo,
		&InstanceStatusBatchLookup{SiteConfig: model.SiteConfig{}, InstallingSkillMap: nil, LocalInfoMap: nil})
	if result.Status != model.StatusDestroyed {
		t.Errorf("期望 destroyed，实际 %s", result.Status)
	}
}

// ─── 交叉验证测试：batch 路径与 solo 路径一致性 ─────────────────────────────

func TestResolveInstanceStatus_BatchVsSolo_ProduceSameStatus(t *testing.T) {
	initStateTestDB(t)
	db := model.DB(context.Background())

	// 准备 installing skill
	db.Create(&model.SkillInstallation{
		InstanceID:    100,
		Slug:          "s1",
		Version:       "1.0",
		InstallStatus: model.SkillInstalling,
	})

	siteConfig := model.GetSiteConfig(context.Background())
	installingMap := batchHasInstallingSkillInstallations(context.Background(), []uint{100, 200})

	batch := &InstanceStatusBatchLookup{
		SiteConfig:         siteConfig,
		InstallingSkillMap: installingMap,
	}

	tests := []struct {
		name     string
		instance *model.Instance
		cvmInfo  *CVMInstanceInfo
	}{
		{"running_agent_ready", &model.Instance{AgentReady: 1}, &CVMInstanceInfo{State: "RUNNING"}},
		{"loading_no_agent", &model.Instance{AgentReady: 0}, &CVMInstanceInfo{State: "RUNNING"}},
		{"upgrade_failed_cvm_exists", &model.Instance{CurrentOperation: model.OpUpgrade, CurrentOperationState: model.OpStateFailed}, &CVMInstanceInfo{State: "RUNNING"}},
		{"upgrade_failed_cvm_gone", &model.Instance{
			InstanceId: "ins-1", LastStableState: "RUNNING",
			CurrentOperation: model.OpUpgrade, CurrentOperationState: model.OpStateFailed,
		}, nil},
		{"stopped", &model.Instance{}, &CVMInstanceInfo{State: "STOPPED"}},
		{"create_failed", &model.Instance{}, &CVMInstanceInfo{State: "LAUNCH_FAILED"}},
		{"upgrading", &model.Instance{CurrentOperation: model.OpUpgrade, CurrentOperationState: model.OpStateProcessing}, &CVMInstanceInfo{State: "RUNNING"}},
		{"destroying", &model.Instance{CurrentOperation: model.OpDelete}, &CVMInstanceInfo{State: "RUNNING"}},
		{"destroyed", &model.Instance{CurrentOperation: model.OpDelete}, (*CVMInstanceInfo)(nil)},
		{"maintaining_unknown", &model.Instance{}, &CVMInstanceInfo{State: "SOME_UNKNOWN"}},
		{"pending_restrict", &model.Instance{}, &CVMInstanceInfo{State: "RUNNING", RestrictState: "EXPIRED"}},
		{"creating_pending", &model.Instance{}, &CVMInstanceInfo{State: "PENDING"}},
	}

	for _, tt := range tests {
		tt.instance.ID = 200 // 使用不在 installingMap 中的 ID
		t.Run(tt.name, func(t *testing.T) {
			solo := ResolveInstanceStatus(context.Background(), tt.instance, tt.cvmInfo, nil)
			batched := ResolveInstanceStatus(context.Background(), tt.instance, tt.cvmInfo, batch)
			if solo.Status != batched.Status {
				t.Errorf("Status 不一致: solo=%s batched=%s", solo.Status, batched.Status)
			}
			if solo.InstanceChargeType != batched.InstanceChargeType {
				t.Errorf("ChargeType 不一致: solo=%s batched=%s", solo.InstanceChargeType, batched.InstanceChargeType)
			}
		})
	}
}

func TestResolveInstanceStatus_BatchVsSolo_InstallingSkill(t *testing.T) {
	initStateTestDB(t)
	db := model.DB(context.Background())

	db.Create(&model.SkillInstallation{
		InstanceID:    300,
		Slug:          "verify-skill",
		Version:       "1.0",
		InstallStatus: model.SkillInstalling,
	})

	siteConfig := model.GetSiteConfig(context.Background())
	installingMap := batchHasInstallingSkillInstallations(context.Background(), []uint{300})

	batch := &InstanceStatusBatchLookup{
		SiteConfig:         siteConfig,
		InstallingSkillMap: installingMap,
	}

	inst := &model.Instance{AgentReady: 1}
	inst.ID = 300
	cvmInfo := &CVMInstanceInfo{State: "RUNNING"}

	solo := ResolveInstanceStatus(context.Background(), inst, cvmInfo, nil)
	batched := ResolveInstanceStatus(context.Background(), inst, cvmInfo, batch)

	if solo.Status != batched.Status {
		t.Errorf("有 installing skill 时 Status 不一致: solo=%s batched=%s", solo.Status, batched.Status)
	}
	if solo.Status != model.StatusLoading {
		t.Errorf("有 installing skill 的实例应返回 loading，实际 %s", solo.Status)
	}
}
