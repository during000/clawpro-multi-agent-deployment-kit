package task

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"hatchery/controller"
	"hatchery/model"
)

// setupMockMemoryAPIServer 启动一个本地 HTTP server 模拟腾讯云 Agent Memory API。
// handler 函数可根据 Action 返回不同响应。返回 cleanup 函数。
func setupMockMemoryAPIServer(t *testing.T, handler http.HandlerFunc) func() {
	t.Helper()
	server := httptest.NewServer(handler)

	// 腾讯云 SDK 使用 endpoint 拼成 https://<endpoint>/，需指向本地 test server 的地址
	// 去掉 http:// 前缀，这样 SDK 会按默认 https 拼回去 —— 但 httptest.NewServer 是 http，
	// 所以 SDK 会尝试 https://127.0.0.1:xxx 并失败（TLS 握手错误）。
	// 实际测试路径：我们要让 SDK 调用返回错误，所以把 endpoint 设为 server.Listener.Addr().String() 即可
	// SDK 使用 POST + HTTPS，会在 TLS 阶段失败，同样达到"调用出错"目的。
	origID := os.Getenv("MEMORY_API_SECRET_ID")
	origKey := os.Getenv("MEMORY_API_SECRET_KEY")
	origEP := os.Getenv("MEMORY_API_ENDPOINT")
	origRegion := os.Getenv("MEMORY_API_REGION")

	os.Setenv("MEMORY_API_SECRET_ID", "mock-id")
	os.Setenv("MEMORY_API_SECRET_KEY", "mock-key")
	os.Setenv("MEMORY_API_ENDPOINT", server.Listener.Addr().String())
	os.Setenv("MEMORY_API_REGION", "ap-test")

	return func() {
		server.Close()
		setEnvOrUnset("MEMORY_API_SECRET_ID", origID)
		setEnvOrUnset("MEMORY_API_SECRET_KEY", origKey)
		setEnvOrUnset("MEMORY_API_ENDPOINT", origEP)
		setEnvOrUnset("MEMORY_API_REGION", origRegion)
	}
}

func setEnvOrUnset(key, value string) {
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
}

// --- handleSwitchToPro 更多分支 ---

// CVM RUNNING + 无 PoolID 场景 → 进入 allocate_database，SDK 调用失败
func TestHandleSwitchToPro_AllocateDatabaseSDKFails(t *testing.T) {
	setupMemoryProTestDB(t)
	defer mockEnsureMemoryPlugin(t)()
	// 使用会 TLS 握手失败的 mock endpoint（httptest.NewServer 是 http，SDK 用 https）
	defer setupMockMemoryAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})()

	model.DB(context.Background()).Create(&model.Instance{
		InstanceId:   "inst-pro-sdk-fail",
		AgentType:    model.AgentTypeOpenClaw,
		LastCVMState: "RUNNING",
	})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:  "inst-pro-sdk-fail",
		CurrentPlan: model.MemoryPlanOff,
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToPro, "switch:inst-pro-sdk-fail", "inst-pro-sdk-fail", "{}", "u", "")
	err := handleSwitchToPro(job)
	if err == nil {
		t.Fatal("SDK 调用失败时应返回错误")
	}
	// 不应是 NonRetryable（SDK 网络错误是可重试的）
	if _, ok := err.(*NonRetryableError); ok {
		t.Errorf("SDK 网络错误不应被判为 NonRetryable: %v", err)
	}
}

// PoolID 存在但远端查询失败 → 回退到重新创建路径（继续失败）
func TestHandleSwitchToPro_ReusePoolIDRemoteGone(t *testing.T) {
	setupMemoryProTestDB(t)
	defer mockEnsureMemoryPlugin(t)()
	defer setupMockMemoryAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})()

	model.DB(context.Background()).Create(&model.Instance{
		InstanceId:   "inst-pro-pool-exist",
		AgentType:    model.AgentTypeOpenClaw,
		LastCVMState: "RUNNING",
	})
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "inst-pro-pool-exist",
		CurrentPlan:  model.MemoryPlanOff,
		PoolID:       "space-old",
		Endpoint:     "http://10.0.0.1:80", // 非 "http://:0"，触发幂等检查路径
		DatabaseName: "db-old",
	})

	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToPro, "switch:inst-pro-pool-exist", "inst-pro-pool-exist", "{}", "u", "")
	err := handleSwitchToPro(job)
	if err == nil {
		t.Fatal("远端查询失败时应返回错误（进入重新创建路径后仍失败）")
	}
}

// handleDeleteMemSpace：SDK 调用失败
func TestHandleDeleteMemSpace_SDKCallFails(t *testing.T) {
	setupMemoryProTestDB(t)
	defer setupMockMemoryAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})()

	plugin := &model.MemoryTDAIPlugin{
		InstanceID:   "inst-del-sdk-fail",
		CurrentPlan:  model.MemoryPlanPro,
		PoolID:       "space-xxx",
		DatabaseName: "db-xxx",
	}
	err := handleDeleteMemSpace(context.Background(), plugin)
	if err == nil {
		t.Fatal("SDK 调用失败时 handleDeleteMemSpace 应返回错误")
	}
}

// rollbackProMemSpace：SDK 调用失败 → 保留本地绑定（已记录 slog.Error）
func TestRollbackProMemSpace_SDKCallFails(t *testing.T) {
	setupMemoryProTestDB(t)
	defer setupMockMemoryAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})()

	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID:   "inst-rb-sdk-call-fail",
		CurrentPlan:  model.MemoryPlanOff, // 未切 PRO → 触发 rollback
		PoolID:       "space-rb",
		DatabaseName: "db-rb",
	})

	rollbackProMemSpace(context.Background(), "inst-rb-sdk-call-fail")

	// SDK 调用失败 → 本地绑定应保留
	plugin := model.GetMemoryTDAIPlugin(context.Background(), "inst-rb-sdk-call-fail")
	if plugin.PoolID != "space-rb" {
		t.Errorf("SDK 调用失败时本地绑定应保留，pool_id = %q", plugin.PoolID)
	}
}

// --- runSwitchPro: embeddingModel 默认值分支 ---
// 这个测试比较难：runSwitchPro 调 RunScript。我们用 LoadScript mock 让它失败，
// 只是为了验证 embeddingModel 为空时走到默认值赋值逻辑（覆盖率）
func TestRunSwitchPro_EmbeddingModelFallback(t *testing.T) {
	setupMemoryProTestDB(t)
	defer setupLoadScriptMock(t)()

	// 直接调用 runSwitchPro（它会在 RunScript 阶段失败）
	err := runSwitchPro(1, "inst-test", "OFF", "http://10.0.0.1:80", "db", "key", "user", "")
	if err == nil {
		t.Fatal("RunScript 失败应返回错误")
	}
}

func TestRunSwitchPro_WithEmbeddingModel(t *testing.T) {
	setupMemoryProTestDB(t)
	defer setupLoadScriptMock(t)()

	err := runSwitchPro(1, "inst-test", "FREE", "http://10.0.0.1:80", "db", "key", "user", "bge-large")
	if err == nil {
		t.Fatal("RunScript 失败应返回错误")
	}
}

// 防御性测试：确保 controller 包 ABI 不变
var _ = controller.LoadScript
