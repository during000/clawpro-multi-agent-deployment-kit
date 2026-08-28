package task

import (
	"context"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ─── 辅助：初始化 recover 测试数据库 ────────────────────────────────────────

func setupRecoverHermesTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Instance{}, &model.SiteConfig{}, &model.CustomAgentType{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	return db
}

// mockRecoverScripts 替换 recoverResolveScriptFn 和 recoverRunScriptFn，返回 cleanup 函数。
func mockRecoverScripts(
	resolveFn func(ctx context.Context, feature, agentType string) (string, error),
	runFn func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error),
) func() {
	origResolve := recoverResolveScriptFn
	origRun := recoverRunScriptFn
	recoverResolveScriptFn = resolveFn
	recoverRunScriptFn = runFn
	return func() {
		recoverResolveScriptFn = origResolve
		recoverRunScriptFn = origRun
	}
}

// ─── 测试用例 ────────────────────────────────────────────────────────────────

// TestRecoverHermesRuntimeUser_NoInstances 无符合条件的实例时直接返回。
func TestRecoverHermesRuntimeUser_NoInstances(t *testing.T) {
	setupRecoverHermesTestDB(t)
	// 不应 panic
	runRecoverHermesRuntimeUser(context.Background())
}

// TestRecoverHermesRuntimeUser_NonHermesSkipped 非 hermes 类型的实例不应被处理。
func TestRecoverHermesRuntimeUser_NonHermesSkipped(t *testing.T) {
	db := setupRecoverHermesTestDB(t)
	db.Create(&model.Instance{
		Name:            "openclaw-inst",
		InstanceId:      "ins-001",
		AgentReady:      1,
		AgentType:       "openclaw",
		LastStableState: "RUNNING",
		RuntimeUser:     "root",
		RuntimeHome:     "/root",
	})

	runRecoverHermesRuntimeUser(context.Background())
	// 无 hermes 实例，不应触发任何探测
}

// TestRecoverHermesRuntimeUser_DataConsistent 探测结果与 DB 一致时不更新。
func TestRecoverHermesRuntimeUser_DataConsistent(t *testing.T) {
	db := setupRecoverHermesTestDB(t)
	db.Create(&model.Instance{
		Name:            "hermes-ok",
		InstanceId:      "ins-hermes-1",
		AgentReady:      1,
		AgentType:       model.AgentTypeHermes,
		LastStableState: "RUNNING",
		RuntimeUser:     "agentuser",
		RuntimeHome:     "/home/agentuser",
	})

	cleanup := mockRecoverScripts(
		func(ctx context.Context, feature, agentType string) (string, error) {
			return "detect_hermes_install.sh", nil
		},
		func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return `{"runtime_user":"agentuser","runtime_home":"/home/agentuser"}`, nil
		},
	)
	defer cleanup()

	runRecoverHermesRuntimeUser(context.Background())

	// 验证 DB 未变
	var inst model.Instance
	db.First(&inst, "instance_id = ?", "ins-hermes-1")
	if inst.RuntimeUser != "agentuser" || inst.RuntimeHome != "/home/agentuser" {
		t.Errorf("数据不应变化，got user=%q home=%q", inst.RuntimeUser, inst.RuntimeHome)
	}
}

// TestRecoverHermesRuntimeUser_DataInconsistent_Fixed 探测结果与 DB 不一致时更新。
func TestRecoverHermesRuntimeUser_DataInconsistent_Fixed(t *testing.T) {
	db := setupRecoverHermesTestDB(t)
	db.Create(&model.Instance{
		Name:            "hermes-dirty",
		InstanceId:      "ins-hermes-2",
		AgentReady:      1,
		AgentType:       model.AgentTypeHermes,
		LastStableState: "RUNNING",
		RuntimeUser:     "root",  // 脏数据
		RuntimeHome:     "/root", // 脏数据
	})

	cleanup := mockRecoverScripts(
		func(ctx context.Context, feature, agentType string) (string, error) {
			return "detect_hermes_install.sh", nil
		},
		func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return `{"runtime_user":"agentuser","runtime_home":"/home/agentuser"}`, nil
		},
	)
	defer cleanup()

	runRecoverHermesRuntimeUser(context.Background())

	// 验证 DB 已更新
	var inst model.Instance
	db.First(&inst, "instance_id = ?", "ins-hermes-2")
	if inst.RuntimeUser != "agentuser" {
		t.Errorf("runtime_user 应更新为 agentuser，got %q", inst.RuntimeUser)
	}
	if inst.RuntimeHome != "/home/agentuser" {
		t.Errorf("runtime_home 应更新为 /home/agentuser，got %q", inst.RuntimeHome)
	}
}

// TestRecoverHermesRuntimeUser_ResolveScriptFailed 解析脚本失败时不 panic。
func TestRecoverHermesRuntimeUser_ResolveScriptFailed(t *testing.T) {
	db := setupRecoverHermesTestDB(t)
	db.Create(&model.Instance{
		Name:            "hermes-resolve-fail",
		InstanceId:      "ins-hermes-3",
		AgentReady:      1,
		AgentType:       model.AgentTypeHermes,
		LastStableState: "RUNNING",
		RuntimeUser:     "root",
		RuntimeHome:     "/root",
	})

	cleanup := mockRecoverScripts(
		func(ctx context.Context, feature, agentType string) (string, error) {
			return "", hcommon.I18nError(i18n.MsgRoleNotFound)
		},
		func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			t.Fatal("RunScript 不应被调用")
			return "", nil
		},
	)
	defer cleanup()

	runRecoverHermesRuntimeUser(context.Background())

	// DB 不应变化
	var inst model.Instance
	db.First(&inst, "instance_id = ?", "ins-hermes-3")
	if inst.RuntimeUser != "root" {
		t.Errorf("探测失败不应修改 DB，got user=%q", inst.RuntimeUser)
	}
}

// TestRecoverHermesRuntimeUser_RunScriptFailed 脚本执行失败时不更新 DB。
func TestRecoverHermesRuntimeUser_RunScriptFailed(t *testing.T) {
	db := setupRecoverHermesTestDB(t)
	db.Create(&model.Instance{
		Name:            "hermes-run-fail",
		InstanceId:      "ins-hermes-4",
		AgentReady:      1,
		AgentType:       model.AgentTypeHermes,
		LastStableState: "RUNNING",
		RuntimeUser:     "root",
		RuntimeHome:     "/root",
	})

	cleanup := mockRecoverScripts(
		func(ctx context.Context, feature, agentType string) (string, error) {
			return "detect_hermes_install.sh", nil
		},
		func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return "", hcommon.I18nError(i18n.MsgTATAgentNotReady)
		},
	)
	defer cleanup()

	runRecoverHermesRuntimeUser(context.Background())

	var inst model.Instance
	db.First(&inst, "instance_id = ?", "ins-hermes-4")
	if inst.RuntimeUser != "root" {
		t.Errorf("脚本失败不应修改 DB，got user=%q", inst.RuntimeUser)
	}
}

// TestRecoverHermesRuntimeUser_InvalidJSON 脚本返回非法 JSON 时不更新 DB。
func TestRecoverHermesRuntimeUser_InvalidJSON(t *testing.T) {
	db := setupRecoverHermesTestDB(t)
	db.Create(&model.Instance{
		Name:            "hermes-bad-json",
		InstanceId:      "ins-hermes-5",
		AgentReady:      1,
		AgentType:       model.AgentTypeHermes,
		LastStableState: "RUNNING",
		RuntimeUser:     "root",
		RuntimeHome:     "/root",
	})

	cleanup := mockRecoverScripts(
		func(ctx context.Context, feature, agentType string) (string, error) {
			return "detect_hermes_install.sh", nil
		},
		func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return "not a json", nil
		},
	)
	defer cleanup()

	runRecoverHermesRuntimeUser(context.Background())

	var inst model.Instance
	db.First(&inst, "instance_id = ?", "ins-hermes-5")
	if inst.RuntimeUser != "root" {
		t.Errorf("JSON 解析失败不应修改 DB，got user=%q", inst.RuntimeUser)
	}
}

// TestRecoverHermesRuntimeUser_EmptyRuntimeUser 探测返回空 runtime_user 时不更新 DB。
func TestRecoverHermesRuntimeUser_EmptyRuntimeUser(t *testing.T) {
	db := setupRecoverHermesTestDB(t)
	db.Create(&model.Instance{
		Name:            "hermes-empty-user",
		InstanceId:      "ins-hermes-6",
		AgentReady:      1,
		AgentType:       model.AgentTypeHermes,
		LastStableState: "RUNNING",
		RuntimeUser:     "root",
		RuntimeHome:     "/root",
	})

	cleanup := mockRecoverScripts(
		func(ctx context.Context, feature, agentType string) (string, error) {
			return "detect_hermes_install.sh", nil
		},
		func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return `{"runtime_user":"","runtime_home":""}`, nil
		},
	)
	defer cleanup()

	runRecoverHermesRuntimeUser(context.Background())

	var inst model.Instance
	db.First(&inst, "instance_id = ?", "ins-hermes-6")
	if inst.RuntimeUser != "root" {
		t.Errorf("空 runtime_user 不应修改 DB，got user=%q", inst.RuntimeUser)
	}
}

// TestRecoverHermesRuntimeUser_UnknownRuntimeUser 探测返回 "unknown" 时不更新 DB。
func TestRecoverHermesRuntimeUser_UnknownRuntimeUser(t *testing.T) {
	db := setupRecoverHermesTestDB(t)
	db.Create(&model.Instance{
		Name:            "hermes-unknown",
		InstanceId:      "ins-hermes-7",
		AgentReady:      1,
		AgentType:       model.AgentTypeHermes,
		LastStableState: "RUNNING",
		RuntimeUser:     "root",
		RuntimeHome:     "/root",
	})

	cleanup := mockRecoverScripts(
		func(ctx context.Context, feature, agentType string) (string, error) {
			return "detect_hermes_install.sh", nil
		},
		func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return `{"runtime_user":"unknown","runtime_home":""}`, nil
		},
	)
	defer cleanup()

	runRecoverHermesRuntimeUser(context.Background())

	var inst model.Instance
	db.First(&inst, "instance_id = ?", "ins-hermes-7")
	if inst.RuntimeUser != "root" {
		t.Errorf("unknown runtime_user 不应修改 DB，got user=%q", inst.RuntimeUser)
	}
}

// TestRecoverHermesRuntimeUser_ContextCanceled 上下文取消时提前退出。
func TestRecoverHermesRuntimeUser_ContextCanceled(t *testing.T) {
	db := setupRecoverHermesTestDB(t)
	for i := 0; i < 3; i++ {
		db.Create(&model.Instance{
			Name:            fmt.Sprintf("hermes-ctx-%d", i),
			InstanceId:      fmt.Sprintf("ins-ctx-%d", i),
			AgentReady:      1,
			AgentType:       model.AgentTypeHermes,
			LastStableState: "RUNNING",
			RuntimeUser:     "root",
			RuntimeHome:     "/root",
		})
	}

	cleanup := mockRecoverScripts(
		func(ctx context.Context, feature, agentType string) (string, error) {
			return "detect_hermes_install.sh", nil
		},
		func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return `{"runtime_user":"agentuser","runtime_home":"/home/agentuser"}`, nil
		},
	)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	// 不应 panic
	runRecoverHermesRuntimeUser(ctx)
}

// TestRecoverHermesRuntimeUser_NotRunningSkipped 非 RUNNING 状态的实例不应被处理。
func TestRecoverHermesRuntimeUser_NotRunningSkipped(t *testing.T) {
	db := setupRecoverHermesTestDB(t)
	db.Create(&model.Instance{
		Name:            "hermes-stopped",
		InstanceId:      "ins-hermes-stopped",
		AgentReady:      1,
		AgentType:       model.AgentTypeHermes,
		LastStableState: "STOPPED",
		RuntimeUser:     "root",
		RuntimeHome:     "/root",
	})

	called := false
	cleanup := mockRecoverScripts(
		func(ctx context.Context, feature, agentType string) (string, error) {
			called = true
			return "detect_hermes_install.sh", nil
		},
		func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return `{"runtime_user":"agentuser","runtime_home":"/home/agentuser"}`, nil
		},
	)
	defer cleanup()

	runRecoverHermesRuntimeUser(context.Background())

	if called {
		t.Error("STOPPED 状态的实例不应触发探测")
	}
}

// TestRecoverOneInstance_FixReturnsTrue 验证 recoverOneInstance 修复后返回 recoverFixed。
func TestRecoverOneInstance_FixReturnsTrue(t *testing.T) {
	db := setupRecoverHermesTestDB(t)
	db.Create(&model.Instance{
		Name:            "hermes-fix",
		InstanceId:      "ins-fix",
		AgentReady:      1,
		AgentType:       model.AgentTypeHermes,
		LastStableState: "RUNNING",
		RuntimeUser:     "wrong-user",
		RuntimeHome:     "/home/wrong-user",
	})

	cleanup := mockRecoverScripts(
		func(ctx context.Context, feature, agentType string) (string, error) {
			return "detect_hermes_install.sh", nil
		},
		func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
			return `{"runtime_user":"correct-user","runtime_home":"/home/correct-user"}`, nil
		},
	)
	defer cleanup()

	var inst model.Instance
	db.First(&inst, "instance_id = ?", "ins-fix")

	result := recoverOneInstance(context.Background(), inst)
	if result != recoverFixed {
		t.Errorf("数据不一致应返回 recoverFixed，got %v", result)
	}

	// 验证 DB 已更新
	db.First(&inst, "instance_id = ?", "ins-fix")
	if inst.RuntimeUser != "correct-user" {
		t.Errorf("expected correct-user, got %q", inst.RuntimeUser)
	}
}
