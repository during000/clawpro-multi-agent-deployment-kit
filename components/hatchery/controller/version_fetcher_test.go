package controller

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ─── 测试辅助函数 ─────────────────────────────────────────────────────────────

// initVersionTestDB 初始化内存 SQLite 数据库，迁移版本同步所需的表。
func initVersionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	return db
}

// mockRunScript 替换 runScriptFn 为 mock 函数，返回恢复函数。
func mockRunScript(fn func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error)) func() {
	orig := runScriptFn
	runScriptFn = fn
	return func() { runScriptFn = orig }
}

// createVersionTestInstance 创建测试用实例。
func createVersionTestInstance(t *testing.T, db *gorm.DB, instanceId string, agentReady int, agentVersion string, fetchedAt *time.Time) *model.Instance {
	t.Helper()
	user := &model.User{Username: "testuser-" + instanceId, Password: "x", Role: "user"}
	db.Create(user)
	inst := &model.Instance{
		Name:             "inst-" + instanceId,
		InstanceId:       instanceId,
		UserID:           user.ID,
		AgentReady:       agentReady,
		AgentVersion:     agentVersion,
		VersionFetchedAt: fetchedAt,
	}
	if err := db.Create(inst).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	return inst
}

// ─── doFetchAndSaveVersionInfo 单元测试 ──────────────────────────────────────

func TestDoFetchAndSaveVersionInfo_EmptyInstanceId(t *testing.T) {
	db := initVersionTestDB(t)
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	db.Create(user)
	inst := model.Instance{Name: "empty-id", InstanceId: "", UserID: user.ID}
	db.Create(&inst)

	// InstanceId 为空时应直接返回 nil，不调用脚本
	err := doFetchAndSaveVersionInfo(context.Background(), inst)
	if err != nil {
		t.Errorf("InstanceId 为空时应返回 nil，实际=%v", err)
	}
}

func TestDoFetchAndSaveVersionInfo_ScriptError(t *testing.T) {
	db := initVersionTestDB(t)
	inst := createVersionTestInstance(t, db, "ins-err-001", 1, "", nil)

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	})
	defer restore()

	err := doFetchAndSaveVersionInfo(context.Background(), *inst)
	if err == nil {
		t.Error("脚本执行失败时应返回错误")
	}
}

func TestDoFetchAndSaveVersionInfo_NoJSONInOutput(t *testing.T) {
	db := initVersionTestDB(t)
	inst := createVersionTestInstance(t, db, "ins-nojson-001", 1, "", nil)

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "some log output\nno json here\n", nil
	})
	defer restore()

	err := doFetchAndSaveVersionInfo(context.Background(), *inst)
	if err != nil {
		t.Errorf("输出中无 JSON 时应返回 nil（静默跳过），实际=%v", err)
	}

	// 验证 DB 中版本信息未被更新
	var updated model.Instance
	db.First(&updated, inst.ID)
	if updated.AgentVersion != "" {
		t.Errorf("无 JSON 输出时 AgentVersion 不应被更新，实际=%s", updated.AgentVersion)
	}
}

func TestDoFetchAndSaveVersionInfo_InvalidJSON(t *testing.T) {
	db := initVersionTestDB(t)
	inst := createVersionTestInstance(t, db, "ins-badjson-001", 1, "", nil)

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "{invalid json}", nil
	})
	defer restore()

	err := doFetchAndSaveVersionInfo(context.Background(), *inst)
	if err == nil {
		t.Error("JSON 解析失败时应返回错误")
	}
}

func TestDoFetchAndSaveVersionInfo_Success(t *testing.T) {
	db := initVersionTestDB(t)
	inst := createVersionTestInstance(t, db, "ins-ok-001", 1, "", nil)

	mockOutput := `checking version...
{"agent_version":"1.2.3","agent_type":"openclaw","plugins":{"codegen":"0.5.1","chat":"1.0.0"}}`

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		if scriptName == "detect_agent_type.sh" {
			return "openclaw\n", nil
		}
		if instanceId != "ins-ok-001" {
			t.Errorf("期望 instanceId=ins-ok-001，实际=%s", instanceId)
		}
		if scriptName != "get_version_info.sh" {
			t.Errorf("期望 scriptName=get_version_info.sh，实际=%s", scriptName)
		}
		if timeout != 30 {
			t.Errorf("期望 timeout=30，实际=%d", timeout)
		}
		return mockOutput, nil
	})
	defer restore()

	err := doFetchAndSaveVersionInfo(context.Background(), *inst)
	if err != nil {
		t.Fatalf("成功场景不应返回错误，实际=%v", err)
	}

	// 验证 DB 中版本信息已正确写入
	var updated model.Instance
	db.First(&updated, inst.ID)
	if updated.AgentVersion != "1.2.3" {
		t.Errorf("期望 AgentVersion=1.2.3，实际=%s", updated.AgentVersion)
	}
	if updated.AgentType != "openclaw" {
		t.Errorf("期望 AgentType=openclaw，实际=%s", updated.AgentType)
	}
	if updated.VersionFetchedAt == nil {
		t.Error("VersionFetchedAt 应被设置")
	}

	// 验证插件版本 JSON
	var plugins map[string]string
	if err := json.Unmarshal([]byte(updated.PluginVersionsJSON), &plugins); err != nil {
		t.Fatalf("解析 PluginVersionsJSON 失败: %v", err)
	}
	if plugins["codegen"] != "0.5.1" {
		t.Errorf("期望 codegen=0.5.1，实际=%s", plugins["codegen"])
	}
	if plugins["chat"] != "1.0.0" {
		t.Errorf("期望 chat=1.0.0，实际=%s", plugins["chat"])
	}
}

func TestDoFetchAndSaveVersionInfo_NoPlugins(t *testing.T) {
	db := initVersionTestDB(t)
	inst := createVersionTestInstance(t, db, "ins-noplugin-001", 1, "", nil)

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return `{"agent_version":"2.0.0","agent_type":"lightclaw","plugins":{}}`, nil
	})
	defer restore()

	err := doFetchAndSaveVersionInfo(context.Background(), *inst)
	if err != nil {
		t.Fatalf("不应返回错误，实际=%v", err)
	}

	var updated model.Instance
	db.First(&updated, inst.ID)
	if updated.AgentVersion != "2.0.0" {
		t.Errorf("期望 AgentVersion=2.0.0，实际=%s", updated.AgentVersion)
	}
	// final 修复后：agent_type 不再由版本同步覆盖（防止 hermes/ace 实例被误写为 openclaw），
	// 实例保留创建时写入的 agent_type。此测试实例默认 agent_type="openclaw"（DB 列默认值）。
	if updated.AgentType != "openclaw" {
		t.Errorf("期望 AgentType 保持 openclaw（不被脚本返回值覆盖），实际=%s", updated.AgentType)
	}
	if updated.PluginVersionsJSON != "{}" {
		t.Errorf("期望 PluginVersionsJSON={}，实际=%s", updated.PluginVersionsJSON)
	}
}

func TestDoFetchAndSaveVersionInfo_MultiLineOutputPicksLastJSON(t *testing.T) {
	db := initVersionTestDB(t)
	inst := createVersionTestInstance(t, db, "ins-multi-001", 1, "", nil)

	// 模拟输出中有多行 JSON，应取最后一个
	mockOutput := `{invalid first line}
some log
{"agent_version":"old","agent_type":"x","plugins":{}}
more log
{"agent_version":"3.0.0","agent_type":"openclaw","plugins":{"tool":"1.1"}}`

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return mockOutput, nil
	})
	defer restore()

	err := doFetchAndSaveVersionInfo(context.Background(), *inst)
	if err != nil {
		t.Fatalf("不应返回错误，实际=%v", err)
	}

	var updated model.Instance
	db.First(&updated, inst.ID)
	if updated.AgentVersion != "3.0.0" {
		t.Errorf("应取最后一个 JSON 行，期望 AgentVersion=3.0.0，实际=%s", updated.AgentVersion)
	}
}

// ─── 并发去重保护测试 ─────────────────────────────────────────────────────────

func TestDoFetchAndSaveVersionInfo_ConcurrentDedup(t *testing.T) {
	db := initVersionTestDB(t)
	inst := createVersionTestInstance(t, db, "ins-dedup-001", 1, "", nil)

	var callCount int
	var mu sync.Mutex

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		// 模拟脚本执行耗时
		time.Sleep(100 * time.Millisecond)
		return `{"agent_version":"1.0.0","agent_type":"openclaw","plugins":{}}`, nil
	})
	defer restore()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = doFetchAndSaveVersionInfo(context.Background(), *inst)
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if callCount != 2 {
		t.Errorf("并发去重应只执行 1 轮拉取（detect+version 共 2 次脚本调用），实际=%d", callCount)
	}
}

// ─── FetchAndSaveVersionInfoSync 测试 ────────────────────────────────────────

func TestFetchAndSaveVersionInfoSync_DelegatesToDoFetch(t *testing.T) {
	db := initVersionTestDB(t)
	inst := createVersionTestInstance(t, db, "ins-sync-001", 1, "", nil)

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return `{"agent_version":"4.0.0","agent_type":"openclaw","plugins":{}}`, nil
	})
	defer restore()

	err := FetchAndSaveVersionInfoSync(context.Background(), *inst)
	if err != nil {
		t.Fatalf("不应返回错误，实际=%v", err)
	}

	var updated model.Instance
	db.First(&updated, inst.ID)
	if updated.AgentVersion != "4.0.0" {
		t.Errorf("期望 AgentVersion=4.0.0，实际=%s", updated.AgentVersion)
	}
}

// ─── versionInfoResult JSON 解析测试 ─────────────────────────────────────────

func TestVersionInfoResult_JSONParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantVer  string
		wantType string
		wantPlug int
	}{
		{
			name:     "完整字段",
			input:    `{"agent_version":"1.2.3","agent_type":"openclaw","plugins":{"a":"1.0","b":"2.0"}}`,
			wantVer:  "1.2.3",
			wantType: "openclaw",
			wantPlug: 2,
		},
		{
			name:     "空插件",
			input:    `{"agent_version":"2.0.0","agent_type":"lightclaw","plugins":{}}`,
			wantVer:  "2.0.0",
			wantType: "lightclaw",
			wantPlug: 0,
		},
		{
			name:     "无插件字段",
			input:    `{"agent_version":"3.0.0","agent_type":"openclaw"}`,
			wantVer:  "3.0.0",
			wantType: "openclaw",
			wantPlug: 0,
		},
		{
			name:     "空版本",
			input:    `{"agent_version":"","agent_type":"","plugins":{}}`,
			wantVer:  "",
			wantType: "",
			wantPlug: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result versionInfoResult
			if err := json.Unmarshal([]byte(tt.input), &result); err != nil {
				t.Fatalf("JSON 解析失败: %v", err)
			}
			if result.AgentVersion != tt.wantVer {
				t.Errorf("AgentVersion: 期望=%s，实际=%s", tt.wantVer, result.AgentVersion)
			}
			if result.AgentType != tt.wantType {
				t.Errorf("AgentType: 期望=%s，实际=%s", tt.wantType, result.AgentType)
			}
			if len(result.Plugins) != tt.wantPlug {
				t.Errorf("Plugins 数量: 期望=%d，实际=%d", tt.wantPlug, len(result.Plugins))
			}
		})
	}
}

// ─── AgentType 保护测试（防止回归） ──────────────────────────────────────────

// TestDoFetchAndSaveVersionInfo_DoesNotOverrideAgentType 验证版本探测**不会**覆盖
// 实例原有的 agent_type。
//
// 背景：get_version_info.sh 当前硬编码返回 "openclaw"，如果我们盲目把脚本返回的
// agent_type 写回 DB，会把 hermes/ace 实例的 agent_type 误覆盖为 openclaw，
// 导致后续所有 ResolveScript 走错分支、前端显示错类型。
//
// 契约：agent_type 是实例创建时确定的不变量，只应由 CreateInstance 写入。
// 版本同步任务只同步 agent_version 和插件版本信息。
func TestDoFetchAndSaveVersionInfo_DoesNotOverrideAgentType(t *testing.T) {
	cases := []struct {
		name             string
		dbAgentType      string // 实例 DB 中已有的 agent_type
		scriptReturnType string // 脚本返回的 agent_type
	}{
		{"ace_not_overridden_by_openclaw", "lightclawace", "openclaw"},
		{"hermes_not_overridden_by_openclaw", "hermes", "openclaw"},
		{"openclaw_not_overridden_by_empty", "openclaw", ""},
		{"ace_not_overridden_by_empty", "lightclawace", ""},
		{"openclaw_stays_openclaw", "openclaw", "openclaw"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := initVersionTestDB(t)

			// 先创建实例，手动设置 agent_type
			user := &model.User{Username: "testuser-" + tc.name, Password: "x", Role: "user"}
			db.Create(user)
			inst := &model.Instance{
				Name:       "inst-" + tc.name,
				InstanceId: "ins-" + tc.name,
				UserID:     user.ID,
				AgentReady: 1,
				AgentType:  tc.dbAgentType,
			}
			if err := db.Create(inst).Error; err != nil {
				t.Fatalf("创建实例失败: %v", err)
			}

			// Mock 脚本：让它返回一个有问题的 agent_type
			restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
				return `{"agent_version":"1.2.3","agent_type":"` + tc.scriptReturnType + `","plugins":{"p1":"0.1.0"}}`, nil
			})
			defer restore()

			// 执行版本同步
			if err := doFetchAndSaveVersionInfo(context.Background(), *inst); err != nil {
				t.Fatalf("doFetchAndSaveVersionInfo 失败: %v", err)
			}

			// 校验：agent_type **不应被覆盖**
			var updated model.Instance
			if err := db.First(&updated, inst.ID).Error; err != nil {
				t.Fatalf("查询实例失败: %v", err)
			}
			if updated.AgentType != tc.dbAgentType {
				t.Errorf("agent_type 被意外覆盖：期望保持 %q，实际 %q（脚本返回 %q）",
					tc.dbAgentType, updated.AgentType, tc.scriptReturnType)
			}

			// 但是 agent_version 应该被正确同步
			if updated.AgentVersion != "1.2.3" {
				t.Errorf("agent_version 未正确同步：期望 1.2.3，实际 %q", updated.AgentVersion)
			}

			// 插件版本也应被同步
			if updated.PluginVersionsJSON == "" || updated.PluginVersionsJSON == "{}" {
				t.Errorf("插件版本未同步：%q", updated.PluginVersionsJSON)
			}
		})
	}
}

// ─── 按 agent_type 分派脚本测试（防止 ACE/Hermes 实例跑 openclaw 脚本） ──────

// TestDoFetchAndSaveVersionInfo_DispatchesScriptByAgentType 验证不同 agent_type
// 的实例会使用不同的 get_version_info 脚本。
//
// 背景：原来 version_fetcher 硬编码 "get_version_info.sh"，导致 ACE/Hermes 实例
// 都跑 openclaw 专属脚本，永远拿不到真实版本。修复后应按 scriptResolveTable 分派。
func TestDoFetchAndSaveVersionInfo_DispatchesScriptByAgentType(t *testing.T) {
	cases := []struct {
		name       string
		agentType  string
		wantScript string
	}{
		{"openclaw", "openclaw", "get_version_info.sh"},
		{"hermes", "hermes", "get_version_info_hermes.sh"},
		{"ace", "lightclawace", "get_version_info_ace.sh"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := initVersionTestDB(t)

			user := &model.User{Username: "u-" + tc.name, Password: "x", Role: "user"}
			db.Create(user)
			inst := &model.Instance{
				Name:        "inst-" + tc.name,
				InstanceId:  "ins-" + tc.name,
				UserID:      user.ID,
				AgentReady:  1,
				AgentType:   tc.agentType,
				RuntimeUser: "agentuser",
			}
			db.Create(inst)

			var actualScript string
			var actualRuntimeUser string
			restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, _ uint64, runtimeUser string, _ func(chunk string), _ map[string]string) (string, error) {
				actualScript = scriptName
				actualRuntimeUser = runtimeUser
				return `{"agent_version":"1.0.0","agent_type":"` + tc.agentType + `","plugins":{}}`, nil
			})
			defer restore()

			if err := doFetchAndSaveVersionInfo(context.Background(), *inst); err != nil {
				t.Fatalf("doFetchAndSaveVersionInfo 失败: %v", err)
			}

			if actualScript != tc.wantScript {
				t.Errorf("脚本分派错误: agent_type=%s 期望脚本=%s 实际=%s",
					tc.agentType, tc.wantScript, actualScript)
			}
			// 校验 runtime_user 也被正确透传
			if actualRuntimeUser != "agentuser" {
				t.Errorf("runtime_user 未透传: 期望 agentuser 实际 %q", actualRuntimeUser)
			}
		})
	}
}

// TestDoFetchAndSaveVersionInfo_UnknownAgentTypeSkips 验证未注册的 agent_type
// （既不是内置也不是自定义类型）会被 GetAgentRuntimeType=="" 守卫直接跳过，不触发任何脚本调用。
func TestDoFetchAndSaveVersionInfo_UnknownAgentTypeSkips(t *testing.T) {
	db := initVersionTestDB(t)

	user := &model.User{Username: "u-unknown", Password: "x", Role: "user"}
	db.Create(user)
	inst := &model.Instance{
		Name:       "inst-unknown",
		InstanceId: "ins-unknown",
		UserID:     user.ID,
		AgentReady: 1,
		AgentType:  "some_future_type",
	}
	db.Create(inst)

	called := false
	restore := mockRunScript(func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		called = true
		return "unknown\n", nil
	})
	defer restore()

	if err := doFetchAndSaveVersionInfo(context.Background(), *inst); err != nil {
		t.Errorf("未知 agent_type 应静默跳过而非返回错误，实际 err=%v", err)
	}
	if called {
		t.Error("未注册的 agent_type 应在入口被跳过，不触发任何脚本调用")
	}
}

func TestDoFetchAndSaveVersionInfo_DetectRepairsAgentType(t *testing.T) {
	db := initVersionTestDB(t)

	user := &model.User{Username: "u-repair", Password: "x", Role: "user"}
	db.Create(user)
	inst := &model.Instance{
		Name:       "inst-repair",
		InstanceId: "ins-repair",
		UserID:     user.ID,
		AgentReady: 1,
		AgentType:  model.AgentTypeOpenClaw,
	}
	db.Create(inst)

	var scripts []string
	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		scripts = append(scripts, scriptName)
		switch scriptName {
		case "detect_agent_type.sh":
			return "hermes\n", nil
		case "get_version_info_hermes.sh":
			return `{"agent_version":"0.9.1","agent_type":"hermes","plugins":{}}`, nil
		default:
			return "", nil
		}
	})
	defer restore()

	if err := doFetchAndSaveVersionInfo(context.Background(), *inst); err != nil {
		t.Fatalf("doFetchAndSaveVersionInfo 失败: %v", err)
	}

	if len(scripts) != 2 || scripts[0] != "detect_agent_type.sh" || scripts[1] != "get_version_info_hermes.sh" {
		t.Fatalf("脚本调用顺序错误，实际=%v", scripts)
	}

	var updated model.Instance
	if err := db.First(&updated, inst.ID).Error; err != nil {
		t.Fatalf("查询实例失败: %v", err)
	}
	if updated.AgentType != model.AgentTypeHermes {
		t.Fatalf("agent_type 应被纠正为 hermes，实际=%s", updated.AgentType)
	}
	if updated.AgentVersion != "0.9.1" {
		t.Fatalf("agent_version 应同步，实际=%s", updated.AgentVersion)
	}
}
