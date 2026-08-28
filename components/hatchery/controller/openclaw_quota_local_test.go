package controller

import (
	"context"
	"testing"

	"hatchery/model"
)

// TestQuotaCount_ExcludesLocalInstance 验证配额计数查询排除本地 agent 实例。
// HandleCreate 的全局/分组配额检查使用同一过滤条件：
//
//	user_id = ? AND is_doctor_node = false AND current_operation != 'delete' AND source != 'local'
//
// 所以本地 agent 实例不会消耗用户的 CVM 实例配额。
func TestQuotaCount_ExcludesLocalInstance(t *testing.T) {
	setupSkillInstancesDB(t)

	ctx := context.Background()
	user := model.User{Username: "quota-test", InstanceQuota: 5}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	// 各类实例：
	//  cvm-running  → 算入配额
	//  cvm-deleting → current_operation=delete，不算
	//  doctor-node  → is_doctor_node=true，不算
	//  local-agent  → source=local，不算（本次 fix）
	type seed struct {
		name      string
		source    string
		isDoctor  bool
		currentOp string
	}
	rows := []seed{
		{"cvm-running", "", false, ""},
		{"cvm-deleting", "", false, model.OpDelete},
		{"doctor-node", "", true, ""},
		{"local-agent-1", model.InstanceSourceLocal, false, ""},
		{"local-agent-2", model.InstanceSourceLocal, false, ""},
	}
	for _, s := range rows {
		inst := model.Instance{
			Name: s.name, InstanceId: s.name,
			UserID: user.ID, Source: s.source, IsDoctorNode: s.isDoctor,
			CurrentOperation: s.currentOp,
		}
		if err := model.DB(ctx).Create(&inst).Error; err != nil {
			t.Fatalf("create %s: %v", s.name, err)
		}
	}

	// 全局配额查询（openclaw.go:1355）
	var globalCount int64
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("user_id = ? AND is_doctor_node = ? AND current_operation != ? AND source != ?",
			user.ID, false, model.OpDelete, model.InstanceSourceLocal).
		Count(&globalCount).Error; err != nil {
		t.Fatalf("count global: %v", err)
	}
	if globalCount != 1 {
		t.Errorf("全局配额计数应=1（仅 cvm-running），实际=%d", globalCount)
	}

	// 反向断言：不带 source 过滤会把本地 agent 也算进去（旧 bug 的样子）
	var leakedCount int64
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("user_id = ? AND is_doctor_node = ? AND current_operation != ?",
			user.ID, false, model.OpDelete).
		Count(&leakedCount).Error; err != nil {
		t.Fatalf("count leaked: %v", err)
	}
	if leakedCount != 3 {
		t.Errorf("反向断言（不排除本地）期望=3（cvm-running + 2 local），实际=%d", leakedCount)
	}
}
