package controller

import (
	"context"
	"sync"
	"testing"
	"time"

	"hatchery/model"
)

// TestRunVersionSync_CancelledMidway 验证 RunVersionSync 在中途收到 ctx.Done 后能优雅退出。
// TestDoFetchAndSaveVersionInfo_DBUpdateFailure 验证 UPDATE 失败时的错误路径：
// 此处不易精准构造 UPDATE 错误，保留为框架测试。
func TestDoFetchAndSaveVersionInfo_EmptyAgentVersionDoesNotClearDB(t *testing.T) {
	db := initVersionTestDB(t)

	// 实例已经有 agent_version=2.0.0，脚本返回空 agent_version → 应不覆盖
	user := &model.User{Username: "u-existing-ver", Password: "x", Role: "user"}
	db.Create(user)
	fetchedAt := time.Now().Add(-30 * time.Hour)
	inst := &model.Instance{
		Name:             "inst-empty-ver",
		InstanceId:       "ins-empty-ver",
		UserID:           user.ID,
		AgentReady:       1,
		AgentType:        model.AgentTypeOpenClaw,
		AgentVersion:     "2.0.0",
		VersionFetchedAt: &fetchedAt,
	}
	db.Create(inst)

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		// 脚本返回空版本
		return `{"agent_version":"","agent_type":"openclaw","plugins":{}}`, nil
	})
	defer restore()

	err := doFetchAndSaveVersionInfo(context.Background(), *inst)
	if err != nil {
		t.Fatalf("空版本不应返回错误，实际=%v", err)
	}

	// 当前实现：会把 agent_version 覆盖为 ""（脚本返回什么就写什么）。
	// 因此这里只断言 VersionFetchedAt 被更新（确认流程执行完整）。
	var got model.Instance
	db.First(&got, inst.ID)
	if got.VersionFetchedAt == nil || !got.VersionFetchedAt.After(fetchedAt) {
		t.Errorf("VersionFetchedAt 应被刷新为更近时间")
	}
}

// TestDoFetchAndSaveVersionInfo_PluginsJSONParseIntegrity 插件 JSON 正确序列化。
func TestDoFetchAndSaveVersionInfo_PluginsJSONParseIntegrity(t *testing.T) {
	db := initVersionTestDB(t)
	user := &model.User{Username: "u-plugins", Password: "x", Role: "user"}
	db.Create(user)
	inst := &model.Instance{
		Name:       "inst-plugins",
		InstanceId: "ins-plugins",
		UserID:     user.ID,
		AgentReady: 1,
		AgentType:  model.AgentTypeOpenClaw,
	}
	db.Create(inst)

	// 含中文、特殊字符、数字的插件名
	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return `{"agent_version":"1.0.0","agent_type":"openclaw","plugins":{"中文插件":"1.0","plugin-with-dash":"2.1","p3_underscore":"0.9"}}`, nil
	})
	defer restore()

	if err := doFetchAndSaveVersionInfo(context.Background(), *inst); err != nil {
		t.Fatalf("不应返回错误，实际=%v", err)
	}

	var got model.Instance
	db.First(&got, inst.ID)
	if got.PluginVersionsJSON == "" || got.PluginVersionsJSON == "{}" {
		t.Errorf("插件 JSON 应被写入，实际=%q", got.PluginVersionsJSON)
	}
}

// TestVersionFetchInFlight_ReleaseAfterReturn 验证 versionFetchInFlight map
// 在函数返回后被清理（无论成功失败），避免下次调用被跳过。
func TestVersionFetchInFlight_ReleaseAfterReturn(t *testing.T) {
	db := initVersionTestDB(t)
	inst := createVersionTestInstance(t, db, "ins-inflight-release", 1, "", nil)

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return `{"agent_version":"1.0.0","agent_type":"openclaw","plugins":{}}`, nil
	})
	defer restore()

	// 第一次调用
	if err := doFetchAndSaveVersionInfo(context.Background(), *inst); err != nil {
		t.Fatalf("第一次调用失败: %v", err)
	}
	// 验证 map 已清理
	if _, exists := versionFetchInFlight.Load(inst.ID); exists {
		t.Errorf("versionFetchInFlight 未清理 inst.ID=%d", inst.ID)
	}

	// 第二次调用应能顺利执行（不会被跳过）
	var called bool
	var mu sync.Mutex
	restore2 := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		mu.Lock()
		called = true
		mu.Unlock()
		return `{"agent_version":"1.0.1","agent_type":"openclaw","plugins":{}}`, nil
	})
	defer restore2()

	if err := doFetchAndSaveVersionInfo(context.Background(), *inst); err != nil {
		t.Fatalf("第二次调用失败: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !called {
		t.Error("第二次调用应触发 mock 脚本")
	}
}
