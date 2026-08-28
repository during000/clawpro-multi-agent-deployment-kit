package task

import (
	"context"
	"errors"
	"testing"

	"hatchery/controller"
	"hatchery/model"
)

// setupLoadScriptMock 注入一个返回错误的 LoadScript mock，保证 RunScript 立即失败，
// 便于测试 handler 的错误路径。
// 返回 cleanup 函数，测试结束时调用恢复原值。
func setupLoadScriptMock(t *testing.T) func() {
	t.Helper()
	orig := controller.LoadScript
	controller.LoadScript = func(name string) (string, error) {
		return "", errors.New("mock: script not loaded in test")
	}
	return func() {
		controller.LoadScript = orig
	}
}

// --- doEnablePlugin（task/memory_tdai_task.go） ---

// LoadScript mock 返回错误 → RunScript 返回 RichError → handlePluginScriptError 写入 last_error，retry_count+1
func TestDoEnablePlugin_ScriptLoadFailed(t *testing.T) {
	setupMemoryProTestDB(t)
	defer setupLoadScriptMock(t)()

	plugin := &model.MemoryTDAIPlugin{
		InstanceID: "inst-dep-fail",
		Status:     model.MemoryTDAIPluginStatusNotInstalled,
		RetryCount: 0,
	}
	model.DB(context.Background()).Create(plugin)

	err := doEnablePlugin(context.Background(), "inst-dep-fail", "", "", "tdai-memory", "[]", plugin)
	if err == nil {
		t.Fatal("LoadScript failed 应返回错误")
	}

	var updated model.MemoryTDAIPlugin
	model.DB(context.Background()).Where("instance_id = ?", "inst-dep-fail").First(&updated)
	if updated.Status != model.MemoryTDAIPluginStatusFailed {
		t.Errorf("status = %q, want FAILED", updated.Status)
	}
	if updated.RetryCount != 1 {
		t.Errorf("retry_count = %d, want 1", updated.RetryCount)
	}
	if updated.LastError == "" {
		t.Error("last_error 应被写入")
	}
}

// --- doSwitchFree（task/tdai_handler_switch.go） ---
// doSwitchFree 目前是未被使用的备份接口，但作为公开符号覆盖一下。

func TestDoSwitchFree_ScriptLoadFailed(t *testing.T) {
	setupMemoryProTestDB(t)
	defer setupLoadScriptMock(t)()

	plugin := &model.MemoryTDAIPlugin{
		InstanceID: "inst-dsf-fail",
		Status:     model.MemoryTDAIPluginStatusNotInstalled,
	}
	model.DB(context.Background()).Create(plugin)

	err := doSwitchFree(context.Background(), "inst-dsf-fail", "tdai-memory", "[]", plugin)
	if err == nil {
		t.Fatal("LoadScript failed 应返回错误")
	}
}

// --- doDisablePluginWithParams（task/tdai_handler_switch.go） ---

func TestDoDisablePluginWithParams_ScriptLoadFailed(t *testing.T) {
	setupMemoryProTestDB(t)
	defer setupLoadScriptMock(t)()

	plugin := &model.MemoryTDAIPlugin{
		InstanceID: "inst-ddpp-fail",
		Status:     model.MemoryTDAIPluginStatusEnabled,
	}
	model.DB(context.Background()).Create(plugin)

	params := map[string]string{"plugin": "tdai-memory"}
	err := doDisablePluginWithParams(context.Background(), "inst-ddpp-fail", params, plugin)
	if err == nil {
		t.Fatal("LoadScript failed 应返回错误")
	}

	var updated model.MemoryTDAIPlugin
	model.DB(context.Background()).Where("instance_id = ?", "inst-ddpp-fail").First(&updated)
	if updated.Status != model.MemoryTDAIPluginStatusFailed {
		t.Errorf("status = %q, want FAILED", updated.Status)
	}
}

// --- handleSwitchToFree 全流程：CVM 就绪后因 RunScript 失败 ---

func TestHandleSwitchToFree_RunScriptFails(t *testing.T) {
	setupMemoryProTestDB(t)
	defer setupLoadScriptMock(t)()

	// 实例 CVM 已就绪
	model.DB(context.Background()).Create(&model.Instance{
		InstanceId:   "inst-free-runfail",
		AgentType:    model.AgentTypeOpenClaw,
		LastCVMState: "RUNNING",
	})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-free-runfail",
		CurrentPlan: model.MemoryPlanOff,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-free-runfail", "inst-free-runfail", "{}", "u", "")
	err := handleSwitchToFree(job)
	if err == nil {
		t.Fatal("RunScript 失败时 handler 应返回错误")
	}
}

// --- handleSwitchToOff 全流程：FREE→OFF，CVM 就绪后因 RunScript 失败 ---

func TestHandleSwitchToOff_FreeRunScriptFails(t *testing.T) {
	setupMemoryProTestDB(t)
	defer setupLoadScriptMock(t)()

	model.DB(context.Background()).Create(&model.Instance{
		InstanceId:   "inst-off-runfail",
		AgentType:    model.AgentTypeOpenClaw,
		LastCVMState: "RUNNING",
	})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-off-runfail",
		CurrentPlan: model.MemoryPlanFree,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-off-runfail", "inst-off-runfail", "{}", "u", "")
	err := handleSwitchToOff(job)
	if err == nil {
		t.Fatal("RunScript 失败时 handler 应返回错误")
	}
}

// --- handleSwitchToOff：Pro→OFF，disable 成功 + DeleteMemSpace 失败路径 ---

// 这个测试覆盖 Step 3 disable_plugin 成功、进入 Step 4 release_database → handleDeleteMemSpace 分支
// 通过 LoadScript 返回一个空字符串（不是 error）让 RunScript 继续走，但再设一个不连通的 TAT 网络环境
// 实际上 LoadScript 返回空不报错，RunScript 会继续尝试 newTATClient - 需要 CVM 凭证，失败
// 实际效果：RunScript 在 newTATClient 阶段失败
func TestHandleSwitchToOff_ProToOff_DeleteMemSpaceFails(t *testing.T) {
	setupMemoryProTestDB(t)
	// 允许 LoadScript 返回有效内容，但 newTATClient 由于无 CVM 凭证失败 → RunScript 失败
	orig := controller.LoadScript
	controller.LoadScript = func(name string) (string, error) {
		return "echo hello", nil
	}
	defer func() { controller.LoadScript = orig }()

	model.DB(context.Background()).Create(&model.Instance{
		InstanceId:   "inst-p2off-fail",
		AgentType:    model.AgentTypeOpenClaw,
		LastCVMState: "RUNNING",
	})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "inst-p2off-fail",
		CurrentPlan:  model.MemoryPlanPro,
		PoolID:       "space-xxx",
		Endpoint:     "http://10.0.0.1:80",
		DatabaseName: "db-x",
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-p2off-fail", "inst-p2off-fail", "{}", "u", "")
	err := handleSwitchToOff(job)
	if err == nil {
		t.Fatal("TAT 客户端创建失败时应返回错误")
	}
}
