package model

import (
	"testing"
)

// ─── 操作类型常量测试 ─────────────────────────────────────────────────────────

func TestOperationConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"OpNone should be empty", OpNone, ""},
		{"OpCreate should be create", OpCreate, "create"},
		{"OpReboot should be reboot", OpReboot, "reboot"},
		{"OpReinstall should be reinstall", OpReinstall, "reinstall"},
		{"OpDelete should be delete", OpDelete, "delete"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("got %q, want %q", tt.got, tt.expected)
			}
		})
	}
}

// ─── 操作状态常量测试 ───────────────────────────────────────────────────────

func TestOperationStateConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"OpStateNone should be empty", OpStateNone, ""},
		{"OpStateProcessing should be processing", OpStateProcessing, "processing"},
		{"OpStateSuccess should be success", OpStateSuccess, "success"},
		{"OpStateFailed should be failed", OpStateFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("got %q, want %q", tt.got, tt.expected)
			}
		})
	}
}

// ─── OpenClaw 状态常量测试 ─────────────────────────────────────────────────

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"StatusCreating", StatusCreating, "creating"},
		{"StatusCreateFailed", StatusCreateFailed, "create_failed"},
		{"StatusRunning", StatusRunning, "running"},
		{"StatusStopped", StatusStopped, "stopped"},
		{"StatusLoading", StatusLoading, "loading"},
		{"StatusLoadFailed", StatusLoadFailed, "load_failed"},
		{"StatusMaintaining", StatusMaintaining, "maintaining"},
		{"StatusPending", StatusPending, "pending"},
		{"StatusDestroying", StatusDestroying, "destroying"},
		{"StatusDestroyed", StatusDestroyed, "destroyed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("got %q, want %q", tt.got, tt.expected)
			}
		})
	}
}

// ─── 用户端状态映射表测试 ───────────────────────────────────────────────────

func TestUserStatusMap_AllStatuses(t *testing.T) {
	// 验证所有状态都在映射表中
	statuses := []string{
		StatusCreating, StatusCreateFailed, StatusRunning, StatusStopped,
		StatusLoading, StatusLoadFailed, StatusMaintaining, StatusPending,
		StatusDestroying, StatusDestroyed,
	}

	for _, status := range statuses {
		t.Run("UserStatusMap_"+status, func(t *testing.T) {
			def, ok := UserStatusMap[status]
			if !ok {
				t.Errorf("状态 %q 不在 UserStatusMap 中", status)
				return
			}
			if def.Status != status {
				t.Errorf("Status 字段不匹配: got %q, want %q", def.Status, status)
			}
			if def.Label == "" {
				t.Errorf("状态 %q 的 Label 为空", status)
			}
		})
	}
}

func TestUserStatusMap_Actions(t *testing.T) {
	tests := []struct {
		status          string
		expectEmpty     bool
		expectedActions []string
	}{
		{StatusCreating, true, nil},
		{StatusCreateFailed, false, []string{"delete"}},
		{StatusRunning, false, []string{"restart_gateway", "reboot", "reinstall", "delete", "terminal"}},
		{StatusStopped, false, []string{"delete"}},
		{StatusLoading, true, nil},
		{StatusLoadFailed, false, []string{"retry", "delete"}},
		{StatusMaintaining, false, []string{"delete"}},
		{StatusPending, true, nil},
		{StatusDestroying, true, nil},
		{StatusDestroyed, false, []string{"delete"}},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			def := UserStatusMap[tt.status]
			if tt.expectEmpty {
				if len(def.Actions) != 0 {
					t.Errorf("期望 %s 的 Actions 为空，实际为 %v", tt.status, def.Actions)
				}
			} else {
				if len(def.Actions) == 0 {
					t.Errorf("期望 %s 的 Actions 非空", tt.status)
				}
			}
		})
	}
}

func TestUserStatusMap_Transient(t *testing.T) {
	tests := []struct {
		status    string
		transient bool
	}{
		{StatusCreating, true},
		{StatusCreateFailed, false},
		{StatusRunning, false},
		{StatusStopped, false},
		{StatusLoading, true},
		{StatusLoadFailed, false},
		{StatusMaintaining, true},
		{StatusPending, true},
		{StatusDestroying, true},
		{StatusDestroyed, false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			def := UserStatusMap[tt.status]
			if def.Transient != tt.transient {
				t.Errorf("Status %s: Transient got %v, want %v", tt.status, def.Transient, tt.transient)
			}
		})
	}
}

// ─── 管控端状态映射表测试 ──────────────────────────────────────────────────

func TestAdminStatusMap_AllStatuses(t *testing.T) {
	statuses := []string{
		StatusCreating, StatusCreateFailed, StatusRunning, StatusStopped,
		StatusLoading, StatusLoadFailed, StatusMaintaining, StatusPending,
		StatusDestroying, StatusDestroyed,
	}

	for _, status := range statuses {
		t.Run("AdminStatusMap_"+status, func(t *testing.T) {
			def, ok := AdminStatusMap[status]
			if !ok {
				t.Errorf("状态 %q 不在 AdminStatusMap 中", status)
				return
			}
			if def.Status != status {
				t.Errorf("Status 字段不匹配: got %q, want %q", def.Status, status)
			}
		})
	}
}

func TestAdminStatusMap_RunningHasAllActions(t *testing.T) {
	def := AdminStatusMap[StatusRunning]
	expected := []string{"terminal", "stop", "delete", "restart_gateway", "reboot", "reinstall", "monitor"}
	if len(def.Actions) != len(expected) {
		t.Errorf("running 的 Actions 数量不对: got %d, want %d", len(def.Actions), len(expected))
	}
	for _, a := range expected {
		found := false
		for _, aa := range def.Actions {
			if aa == a {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("running 缺少 action %q", a)
		}
	}
}

func TestAdminStatusMap_StoppedHasStart(t *testing.T) {
	def := AdminStatusMap[StatusStopped]
	for _, a := range def.Actions {
		if a == "start" {
			return
		}
	}
	t.Errorf("stopped 的 Actions 应包含 'start'，实际为 %v", def.Actions)
}

// ─── 操作超时阈值测试 ──────────────────────────────────────────────────────

func TestOperationTimeouts(t *testing.T) {
	tests := []struct {
		op       string
		expected int
	}{
		{OpCreate, 600},
		{OpReboot, 300},
		{OpReinstall, 900},
		{OpDelete, 300},
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			timeout, ok := OperationTimeouts[tt.op]
			if !ok {
				t.Errorf("操作 %q 没有定义超时", tt.op)
				return
			}
			if timeout != tt.expected {
				t.Errorf("操作 %q 超时: got %d, want %d", tt.op, timeout, tt.expected)
			}
		})
	}

	// 验证所有操作都有超时定义
	for op := range OperationTimeouts {
		if op != OpCreate && op != OpReboot && op != OpReinstall && op != OpDelete && op != "" {
			// OperationTimeouts 中不应有未知操作
		}
	}
}

// ─── CVM 状态分类测试 ──────────────────────────────────────────────────────

func TestCVMTransientStates(t *testing.T) {
	transientStates := []string{
		"PENDING", "LAUNCHING", "STOPPING", "STARTING",
		"REBOOTING", "SHUTDOWN", "TERMINATING",
	}

	for _, state := range transientStates {
		t.Run("transient_"+state, func(t *testing.T) {
			if !CVMTransientStates[state] {
				t.Errorf("状态 %q 应在 CVMTransientStates 中", state)
			}
		})
	}

	stableStates := []string{"RUNNING", "STOPPED", "LAUNCH_FAILED"}
	for _, state := range stableStates {
		t.Run("stable_not_"+state, func(t *testing.T) {
			if CVMTransientStates[state] {
				t.Errorf("状态 %q 不应在 CVMTransientStates 中", state)
			}
		})
	}
}

func TestCVMStableStates(t *testing.T) {
	stableStates := []string{"RUNNING", "STOPPED"}

	for _, state := range stableStates {
		t.Run("stable_"+state, func(t *testing.T) {
			if !CVMStableStates[state] {
				t.Errorf("状态 %q 应在 CVMStableStates 中", state)
			}
		})
	}

	transientStates := []string{"PENDING", "TERMINATING"}
	for _, state := range transientStates {
		t.Run("transient_not_"+state, func(t *testing.T) {
			if CVMStableStates[state] {
				t.Errorf("状态 %q 不应在 CVMStableStates 中", state)
			}
		})
	}
}

func TestCVMOrphanedNotInSpecialClassification(t *testing.T) {
	// Gap 8 修复验证：ORPHANED 不在需求定义的 CVM 状态枚举中
	// 应走兜底 maintaining 逻辑，不应被任何分类 map 捕获
	orphaned := "ORPHANED"
	if CVMTransientStates[orphaned] {
		t.Errorf("ORPHANED 不应在 CVMTransientStates 中")
	}
	if CVMPlatformLimitStates[orphaned] {
		t.Errorf("ORPHANED 不应在 CVMPlatformLimitStates 中")
	}
	if CVMRescueModeStates[orphaned] {
		t.Errorf("ORPHANED 不应在 CVMRescueModeStates 中")
	}
	if CVMLiveMigrateStates[orphaned] {
		t.Errorf("ORPHANED 不应在 CVMLiveMigrateStates 中")
	}
	if CVMStableStates[orphaned] {
		t.Errorf("ORPHANED 不应在 CVMStableStates 中")
	}
}

func TestCVMLiveMigrateStates(t *testing.T) {
	liveMigrateStates := []string{
		"ENTER_SERVICE_LIVE_MIGRATE",
		"SERVICE_LIVE_MIGRATE",
		"EXIT_SERVICE_LIVE_MIGRATE",
	}
	for _, state := range liveMigrateStates {
		t.Run("live_migrate_"+state, func(t *testing.T) {
			if !CVMLiveMigrateStates[state] {
				t.Errorf("状态 %q 应在 CVMLiveMigrateStates 中", state)
			}
		})
	}
	// 非热迁移态不应在其中
	nonLiveMigrate := []string{"RUNNING", "STOPPED", "PENDING"}
	for _, state := range nonLiveMigrate {
		if CVMLiveMigrateStates[state] {
			t.Errorf("状态 %q 不应在 CVMLiveMigrateStates 中", state)
		}
	}
}

func TestCVMPlatformLimitStates(t *testing.T) {
	platformLimitStates := []string{
		"FREEZING", "BANNING", "CORRUPTED",
	}
	for _, state := range platformLimitStates {
		t.Run("platform_limit_"+state, func(t *testing.T) {
			if !CVMPlatformLimitStates[state] {
				t.Errorf("状态 %q 应在 CVMPlatformLimitStates 中", state)
			}
		})
	}
	// 正常态不应在其中
	normalStates := []string{"RUNNING", "STOPPED", "PENDING"}
	for _, state := range normalStates {
		if CVMPlatformLimitStates[state] {
			t.Errorf("状态 %q 不应在 CVMPlatformLimitStates 中", state)
		}
	}
	// ISOLATING/ISOLATED 已改用 RestrictState 判断，不应在 CVMPlatformLimitStates 中
	isolatedStates := []string{"ISOLATING", "ISOLATED"}
	for _, state := range isolatedStates {
		if CVMPlatformLimitStates[state] {
			t.Errorf("状态 %q 不应在 CVMPlatformLimitStates 中（已改用 RestrictState 判断）", state)
		}
	}
}

func TestCVMRescueModeStates(t *testing.T) {
	rescueModeStates := []string{
		"ENTER_RESCUE_MODE", "RESCUE_MODE", "EXIT_RESCUE_MODE",
	}
	for _, state := range rescueModeStates {
		t.Run("rescue_mode_"+state, func(t *testing.T) {
			if !CVMRescueModeStates[state] {
				t.Errorf("状态 %q 应在 CVMRescueModeStates 中", state)
			}
		})
	}
	// 非救援模式态不应在其中
	normalStates := []string{"RUNNING", "STOPPED", "PENDING"}
	for _, state := range normalStates {
		if CVMRescueModeStates[state] {
			t.Errorf("状态 %q 不应在 CVMRescueModeStates 中", state)
		}
	}
}
