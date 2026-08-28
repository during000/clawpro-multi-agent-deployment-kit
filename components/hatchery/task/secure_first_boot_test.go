package task

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ─── 工具函数 ─────────────────────────────────────────────────────────────

// mockSecureFirstBootRunScript 替换 secureFirstBootRunScriptFn，返回 cleanup。
// 回调 fn 决定第 N 次调用返回什么；调用次数由调用方维护。
func mockSecureFirstBootRunScript(fn func(attempt int, instanceId, scriptName, runtimeUser string) (string, error)) (cleanup func(), calls *int32) {
	var counter int32
	orig := secureFirstBootRunScriptFn
	secureFirstBootRunScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		n := atomic.AddInt32(&counter, 1)
		return fn(int(n), instanceId, scriptName, runtimeUser)
	}
	return func() { secureFirstBootRunScriptFn = orig }, &counter
}

// withFastRetry 把重试间隔临时缩到 1ms，避免单测因为 5s 间隔变慢。
func withFastRetry(t *testing.T) {
	t.Helper()
	origInterval := secureFirstBootRetryInterval
	secureFirstBootRetryInterval = time.Millisecond
	t.Cleanup(func() { secureFirstBootRetryInterval = origInterval })
}

// ─── secureFirstBootAsync 本体覆盖 ────────────────────────────────

// TestSecureFirstBootAsync_SuccessFirstTry 首次 RunScript 即成功，不应重试。
func TestSecureFirstBootAsync_SuccessFirstTry(t *testing.T) {
	withFastRetry(t)

	cleanup, calls := mockSecureFirstBootRunScript(func(attempt int, instanceId, scriptName, runtimeUser string) (string, error) {
		if scriptName != "secure_first_boot.sh" {
			t.Errorf("脚本名应为 secure_first_boot.sh，实际 %s", scriptName)
		}
		if instanceId != "ins-rotate-1" {
			t.Errorf("instanceId 透传错误，实际 %s", instanceId)
		}
		if runtimeUser != "ubuntu" {
			t.Errorf("runtimeUser 透传错误，实际 %s", runtimeUser)
		}
		return "ok", nil
	})
	defer cleanup()

	inst := model.Instance{
		InstanceId:  "ins-rotate-1",
		AgentType:   model.AgentTypeOpenClaw,
		RuntimeUser: "ubuntu",
	}
	secureFirstBootAsync(context.Background(), inst)

	if got := atomic.LoadInt32(calls); got != 1 {
		t.Errorf("应只调用 1 次，实际 %d", got)
	}
}

// TestSecureFirstBootAsync_RetryThenSuccess 第一次失败、第二次成功，应停止重试。
func TestSecureFirstBootAsync_RetryThenSuccess(t *testing.T) {
	withFastRetry(t)

	cleanup, calls := mockSecureFirstBootRunScript(func(attempt int, instanceId, scriptName, runtimeUser string) (string, error) {
		if attempt == 1 {
			return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
		}
		return "ok", nil
	})
	defer cleanup()

	inst := model.Instance{InstanceId: "ins-rotate-2", AgentType: model.AgentTypeOpenClaw}
	secureFirstBootAsync(context.Background(), inst)

	if got := atomic.LoadInt32(calls); got != 2 {
		t.Errorf("应调用 2 次（第 1 次失败、第 2 次成功），实际 %d", got)
	}
}

// TestSecureFirstBootAsync_AllFailed 所有重试均失败：函数正常返回（仅日志），不 panic、不阻塞。
func TestSecureFirstBootAsync_AllFailed(t *testing.T) {
	withFastRetry(t)

	cleanup, calls := mockSecureFirstBootRunScript(func(attempt int, instanceId, scriptName, runtimeUser string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	})
	defer cleanup()

	inst := model.Instance{InstanceId: "ins-rotate-3", AgentType: model.AgentTypeOpenClaw}
	secureFirstBootAsync(context.Background(), inst)

	// 应恰好用满 secureFirstBootMaxAttempts 次
	if got := atomic.LoadInt32(calls); int(got) != secureFirstBootMaxAttempts {
		t.Errorf("应调用 %d 次，实际 %d", secureFirstBootMaxAttempts, got)
	}
}

// TestSecureFirstBootAsync_PanicRecovered RunScript 内部 panic 时，recover 兜底，函数正常返回。
func TestSecureFirstBootAsync_PanicRecovered(t *testing.T) {
	withFastRetry(t)

	cleanup, _ := mockSecureFirstBootRunScript(func(attempt int, instanceId, scriptName, runtimeUser string) (string, error) {
		panic("simulated panic")
	})
	defer cleanup()

	inst := model.Instance{InstanceId: "ins-rotate-4", AgentType: model.AgentTypeOpenClaw}
	// 不应 panic 到测试外部
	secureFirstBootAsync(context.Background(), inst)
}

// TestSecureFirstBootAsync_EmptyRuntimeUser RuntimeUser 为空时仍能跑（脚本侧有 fallback）。
func TestSecureFirstBootAsync_EmptyRuntimeUser(t *testing.T) {
	withFastRetry(t)

	cleanup, calls := mockSecureFirstBootRunScript(func(attempt int, instanceId, scriptName, runtimeUser string) (string, error) {
		if runtimeUser != "" {
			t.Errorf("应透传空 runtimeUser，实际 %q", runtimeUser)
		}
		return "ok", nil
	})
	defer cleanup()

	inst := model.Instance{InstanceId: "ins-rotate-5", AgentType: model.AgentTypeOpenClaw, RuntimeUser: ""}
	secureFirstBootAsync(context.Background(), inst)

	if got := atomic.LoadInt32(calls); got != 1 {
		t.Errorf("应只调用 1 次，实际 %d", got)
	}
}

// ─── checkAllAgents 触发分支覆盖 ─────────────────────────────

// secureFirstBootTrigger 用于测试期间记录 secureFirstBootAsyncFn 的调用情况。
type secureFirstBootTrigger struct {
	mu     sync.Mutex
	calls  []model.Instance
	doneCh chan struct{} // 每次调用都向其发信号，便于同步等待
}

func newSecureFirstBootTrigger() *secureFirstBootTrigger {
	return &secureFirstBootTrigger{doneCh: make(chan struct{}, 8)}
}

func (r *secureFirstBootTrigger) hook(ctx context.Context, inst model.Instance) {
	r.mu.Lock()
	r.calls = append(r.calls, inst)
	r.mu.Unlock()
	select {
	case r.doneCh <- struct{}{}:
	default:
	}
}

func (r *secureFirstBootTrigger) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

// installSecureFirstBootTriggerMock 替换 secureFirstBootAsyncFn，返回 cleanup。
func installSecureFirstBootTriggerMock(t *testing.T) *secureFirstBootTrigger {
	t.Helper()
	r := newSecureFirstBootTrigger()
	orig := secureFirstBootAsyncFn
	secureFirstBootAsyncFn = r.hook
	t.Cleanup(func() { secureFirstBootAsyncFn = orig })
	return r
}

// waitForCalls 等待至多 timeout，直到 trigger 收到 n 次调用；返回最终实际次数。
func (r *secureFirstBootTrigger) waitForCalls(n int, timeout time.Duration) int {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if r.count() >= n {
			return r.count()
		}
		time.Sleep(10 * time.Millisecond)
	}
	return r.count()
}

// TestCheckAllAgents_SecureFirstBootTriggered_CustomImage 自定义镜像（非候选 imgId）+ OpCreate + openclaw → 触发。
func TestCheckAllAgents_SecureFirstBootTriggered_CustomImage(t *testing.T) {
	db := setupAgentTestDB(t)
	user := createAgentTestUser(t, db)
	trigger := installSecureFirstBootTriggerMock(t)

	inst := &model.Instance{
		Name:             "openclaw-custom-img",
		UserID:           user.ID,
		InstanceId:       "ins-rot-trig-1",
		AgentType:        model.AgentTypeOpenClaw,
		AgentReady:       0,
		LastCVMState:     "RUNNING",
		CurrentOperation: model.OpCreate,
		ImgId:            "img-custom-xxx", // 非候选镜像 → 走自定义镜像分支
	}
	if err := db.Create(inst).Error; err != nil {
		t.Skipf("跳过测试（创建实例失败）: %v", err)
	}
	// SQLite 对 varchar 字段在 Create 时有时丢失，原始 SQL 兜底（与 base_test 同款）
	db.Exec("UPDATE instances SET last_cvm_state = 'RUNNING', current_operation = ?, img_id = ?, agent_type = ? WHERE id = ?",
		model.OpCreate, "img-custom-xxx", model.AgentTypeOpenClaw, inst.ID)

	agentCheckers = nil
	RegisterAgentChecker(&mockChecker{
		name:   "mock",
		result: map[string]CheckResult{"ins-rot-trig-1": CheckReady},
	})

	checkAllAgents(context.Background())

	if got := trigger.waitForCalls(1, time.Second); got != 1 {
		t.Errorf("自定义镜像应触发 1 次 secure_first_boot，实际 %d", got)
	}
}

// TestCheckAllAgents_SecureFirstBootSkipped_OfficialImage 官方候选镜像 → 不触发（由镜像内置脚本接管）。
func TestCheckAllAgents_SecureFirstBootSkipped_OfficialImage(t *testing.T) {
	db := setupAgentTestDB(t)
	user := createAgentTestUser(t, db)
	trigger := installSecureFirstBootTriggerMock(t)

	// 用 common.CandidateImages 里第一个 openclaw 镜像 ID
	officialImgId := "img-idzg74s9"
	if !hcommon.IsCandidateImage(officialImgId) {
		t.Skipf("用例前提：%s 应为候选镜像", officialImgId)
	}

	inst := &model.Instance{
		Name:             "openclaw-official-img",
		UserID:           user.ID,
		InstanceId:       "ins-rot-skip-official",
		AgentType:        model.AgentTypeOpenClaw,
		AgentReady:       0,
		LastCVMState:     "RUNNING",
		CurrentOperation: model.OpCreate,
		ImgId:            officialImgId,
	}
	if err := db.Create(inst).Error; err != nil {
		t.Skipf("跳过测试（创建实例失败）: %v", err)
	}
	db.Exec("UPDATE instances SET last_cvm_state = 'RUNNING', current_operation = ?, img_id = ?, agent_type = ? WHERE id = ?",
		model.OpCreate, officialImgId, model.AgentTypeOpenClaw, inst.ID)

	agentCheckers = nil
	RegisterAgentChecker(&mockChecker{
		name:   "mock",
		result: map[string]CheckResult{"ins-rot-skip-official": CheckReady},
	})

	checkAllAgents(context.Background())

	// 给 goroutine 一点时间发车（不应该发，但保险起见小睡再断言）
	time.Sleep(80 * time.Millisecond)
	if got := trigger.count(); got != 0 {
		t.Errorf("官方候选镜像不应触发 secure_first_boot，实际触发 %d 次", got)
	}
}

// TestCheckAllAgents_SecureFirstBootSkipped_NotOpenClaw 非 openclaw 类型 → 不触发。
func TestCheckAllAgents_SecureFirstBootSkipped_NotOpenClaw(t *testing.T) {
	db := setupAgentTestDB(t)
	user := createAgentTestUser(t, db)
	trigger := installSecureFirstBootTriggerMock(t)

	inst := &model.Instance{
		Name:             "hermes-custom",
		UserID:           user.ID,
		InstanceId:       "ins-rot-skip-hermes",
		AgentType:        model.AgentTypeHermes,
		AgentReady:       0,
		LastCVMState:     "RUNNING",
		CurrentOperation: model.OpCreate,
		ImgId:            "img-custom-yyy",
	}
	if err := db.Create(inst).Error; err != nil {
		t.Skipf("跳过测试（创建实例失败）: %v", err)
	}
	db.Exec("UPDATE instances SET last_cvm_state = 'RUNNING', current_operation = ?, img_id = ?, agent_type = ? WHERE id = ?",
		model.OpCreate, "img-custom-yyy", model.AgentTypeHermes, inst.ID)

	agentCheckers = nil
	RegisterAgentChecker(&mockChecker{
		name:   "mock",
		result: map[string]CheckResult{"ins-rot-skip-hermes": CheckReady},
	})

	checkAllAgents(context.Background())

	time.Sleep(80 * time.Millisecond)
	if got := trigger.count(); got != 0 {
		t.Errorf("非 openclaw 类型不应触发 secure_first_boot，实际触发 %d 次", got)
	}
}

// TestCheckAllAgents_SecureFirstBootSkipped_NotOpCreate 非创建操作（如 reboot）→ 不触发。
func TestCheckAllAgents_SecureFirstBootSkipped_NotOpCreate(t *testing.T) {
	db := setupAgentTestDB(t)
	user := createAgentTestUser(t, db)
	trigger := installSecureFirstBootTriggerMock(t)

	inst := &model.Instance{
		Name:             "openclaw-reboot",
		UserID:           user.ID,
		InstanceId:       "ins-rot-skip-reboot",
		AgentType:        model.AgentTypeOpenClaw,
		AgentReady:       0,
		LastCVMState:     "RUNNING",
		CurrentOperation: model.OpReboot,
		ImgId:            "img-custom-zzz",
	}
	if err := db.Create(inst).Error; err != nil {
		t.Skipf("跳过测试（创建实例失败）: %v", err)
	}
	db.Exec("UPDATE instances SET last_cvm_state = 'RUNNING', current_operation = ?, img_id = ?, agent_type = ? WHERE id = ?",
		model.OpReboot, "img-custom-zzz", model.AgentTypeOpenClaw, inst.ID)

	agentCheckers = nil
	RegisterAgentChecker(&mockChecker{
		name:   "mock",
		result: map[string]CheckResult{"ins-rot-skip-reboot": CheckReady},
	})

	checkAllAgents(context.Background())

	time.Sleep(80 * time.Millisecond)
	if got := trigger.count(); got != 0 {
		t.Errorf("非 OpCreate 不应触发 secure_first_boot，实际触发 %d 次", got)
	}
}
