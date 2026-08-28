package task

import (
	"context"
	"testing"

	"hatchery/model"
)

// ============================================================================
// 本文件聚焦 agent_checker.go 里已有 57.1% 覆盖率的 detectAndSaveRuntimeUser
// 的剩余分支（成功解析 JSON 路径 无法测，因为 RunScript 未暴露 DI 接口；
// 但可以测 "script output 解析失败"、"runtime_user=unknown" 这两个 warn-only 分支）。
// ============================================================================

// TestDetectAndSaveRuntimeUser_InvalidJson 验证脚本执行成功但 output 非法 JSON
// 时，函数 warn-only return（不更新 DB）。这条路径要 LoadScript 成功 +
// RunScript 成功 + output 返回非法 JSON，无法直接通过 LoadScript mock 实现，
// 因此该场景目前不可测（需要 DI RunScript）。
//
// 当前仅补一条"ResolveScript 失败"的分支（L319-323，unknown agent_type），
// 这是唯一不依赖 RunScript 就能命中的早期 return 路径。
func TestDetectAndSaveRuntimeUser_UnknownAgentType_NoOp(t *testing.T) {
	db := setupRuntimeUserTestDB(t)
	user := &model.User{Username: "u-unknown", Password: "x", Role: "user"}
	db.Create(user)
	inst := &model.Instance{
		Name:        "unknown-inst",
		UserID:      user.ID,
		InstanceId:  "ins-unknown-agent",
		AgentType:   "some_future_unknown_type",
		RuntimeUser: "", // 未设置
	}
	db.Create(inst)

	// ResolveScript("detect_install", "some_future_unknown_type") 将失败 →
	// 早期 return，不触碰 DB
	detectAndSaveRuntimeUser(context.Background(), inst.ID, inst.InstanceId, "some_future_unknown_type")

	var got model.Instance
	db.First(&got, inst.ID)
	if got.RuntimeUser != "" {
		t.Errorf("未知 agent_type ResolveScript 失败时不应写 runtime_user，实际=%q", got.RuntimeUser)
	}
}
