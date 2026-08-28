package task

import (
	"context"
	"fmt"
	"testing"
	"time"

	"hatchery/controller"
	"hatchery/model"
)

// withFastDetectRetry 把 detectAndSaveRuntimeUser 的重试间隔缩短到 1ms，避免单测阻塞。
func withFastDetectRetry(t *testing.T) {
	t.Helper()
	origInterval := detectRuntimeUserRetryInterval
	detectRuntimeUserRetryInterval = time.Millisecond
	t.Cleanup(func() { detectRuntimeUserRetryInterval = origInterval })
}

// TestDetectAndSaveRuntimeUser_HermesFallbackToAgentuser 验证 Hermes 实例
// 在探测脚本执行失败时，不再写入兜底值（避免写入可能错误的硬编码值），
// runtime_user 保持为空，后续 ensureRuntimeUser 调用时再重试探测。
func TestDetectAndSaveRuntimeUser_HermesFallbackToAgentuser(t *testing.T) {
	withFastDetectRetry(t)
	db := setupRuntimeUserTestDB(t)
	user := &model.User{Username: "u-hermes", Password: "x", Role: "user"}
	db.Create(user)
	inst := &model.Instance{
		Name:       "hermes-inst",
		UserID:     user.ID,
		InstanceId: "ins-hermes-detect",
		AgentType:  model.AgentTypeHermes,
	}
	db.Create(inst)

	// 让 controller.LoadScript 返回 error → RunScript 在第一步就失败
	origLoader := controller.LoadScript
	controller.LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: script not found: %s", name)
	}
	defer func() { controller.LoadScript = origLoader }()

	detectAndSaveRuntimeUser(context.Background(), inst.ID, inst.InstanceId, model.AgentTypeHermes)

	// 验证：探测失败时不写入兜底值，runtime_user 保持为空
	var got model.Instance
	db.First(&got, inst.ID)
	if got.RuntimeUser != "" {
		t.Errorf("Hermes 探测失败时 runtime_user 不应被写入，实际=%q", got.RuntimeUser)
	}
	if got.RuntimeHome != "" {
		t.Errorf("Hermes 探测失败时 runtime_home 不应被写入，实际=%q", got.RuntimeHome)
	}
}

// TestDetectAndSaveRuntimeUser_AceFallbackToAgentuser 验证 LightclawACE 实例
// 在探测脚本执行失败时，同样不写入兜底值，runtime_user 保持为空。
func TestDetectAndSaveRuntimeUser_AceFallbackToAgentuser(t *testing.T) {
	withFastDetectRetry(t)
	db := setupRuntimeUserTestDB(t)
	user := &model.User{Username: "u-ace", Password: "x", Role: "user"}
	db.Create(user)
	inst := &model.Instance{
		Name:       "ace-inst",
		UserID:     user.ID,
		InstanceId: "ins-ace-detect",
		AgentType:  model.AgentTypeLightclawACE,
	}
	db.Create(inst)

	origLoader := controller.LoadScript
	controller.LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: script not found: %s", name)
	}
	defer func() { controller.LoadScript = origLoader }()

	detectAndSaveRuntimeUser(context.Background(), inst.ID, inst.InstanceId, model.AgentTypeLightclawACE)

	// 验证：探测失败时不写入兜底值，runtime_user 保持为空
	var got model.Instance
	db.First(&got, inst.ID)
	if got.RuntimeUser != "" {
		t.Errorf("ACE 探测失败时 runtime_user 不应被写入，实际=%q", got.RuntimeUser)
	}
	if got.RuntimeHome != "" {
		t.Errorf("ACE 探测失败时 runtime_home 不应被写入，实际=%q", got.RuntimeHome)
	}
}

// TestDetectAndSaveRuntimeUser_OpenClawNoFallback 验证 OpenClaw 实例
// 脚本失败时**不会**走 agentuser 兜底（保持 runtime_user 为空）。
func TestDetectAndSaveRuntimeUser_OpenClawNoFallback(t *testing.T) {
	withFastDetectRetry(t)
	db := setupRuntimeUserTestDB(t)
	user := &model.User{Username: "u-oc", Password: "x", Role: "user"}
	db.Create(user)
	inst := &model.Instance{
		Name:       "oc-inst",
		UserID:     user.ID,
		InstanceId: "ins-oc-detect",
		AgentType:  model.AgentTypeOpenClaw,
	}
	db.Create(inst)

	origLoader := controller.LoadScript
	controller.LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: script not found")
	}
	defer func() { controller.LoadScript = origLoader }()

	detectAndSaveRuntimeUser(context.Background(), inst.ID, inst.InstanceId, model.AgentTypeOpenClaw)

	// 验证：OpenClaw 不走兜底逻辑，runtime_user 保持空
	var got model.Instance
	db.First(&got, inst.ID)
	if got.RuntimeUser != "" {
		t.Errorf("OpenClaw 脚本失败时 runtime_user 不应被写入，实际=%q", got.RuntimeUser)
	}
}

// TestDetectAndSaveRuntimeUser_RetryParamsZero 验证 max_attempts=0 时
// 直接进入失败分支（lastErr 为 nil 但 output 为空 → 解析失败），DB 不会被写入。
func TestDetectAndSaveRuntimeUser_RetryParamsZero(t *testing.T) {
	origAttempts := detectRuntimeUserMaxAttempts
	detectRuntimeUserMaxAttempts = 0
	t.Cleanup(func() { detectRuntimeUserMaxAttempts = origAttempts })

	db := setupRuntimeUserTestDB(t)
	user := &model.User{Username: "u-zero", Password: "x", Role: "user"}
	db.Create(user)
	inst := &model.Instance{
		Name:       "zero-attempt-inst",
		UserID:     user.ID,
		InstanceId: "ins-zero-detect",
		AgentType:  model.AgentTypeHermes,
	}
	db.Create(inst)

	// 即便 LoadScript 不 mock，max_attempts=0 时循环不会执行
	detectAndSaveRuntimeUser(context.Background(), inst.ID, inst.InstanceId, model.AgentTypeHermes)

	// 验证：循环不执行 → output 为空 → JSON 解析失败 → 不写入 DB
	var got model.Instance
	db.First(&got, inst.ID)
	if got.RuntimeUser != "" {
		t.Errorf("max_attempts=0 时不应写入 DB，实际=%q", got.RuntimeUser)
	}
}
