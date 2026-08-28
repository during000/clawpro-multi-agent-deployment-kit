// set_model_script_test.go
//
// 针对 scripts/set_model.sh 的黑盒回归单测。
//
// 测试目标：
//  1. 【核心回归】同一 modelId 在多个 provider 通道下并存时，新增其中一个 provider
//     不应误删其他合法 provider（修复 bug：custom-deepseek-v3.1 入库时
//     hatchery-deepseek-v3.1 被误删导致 fallback 悬空引用）。
//  2. 旧版"裸 provider key"（hatchery / zhipu / qcloudlkeap）应被正确清理或迁移。
//
// 测试手法：与 set_model_ace_script_test.go 一致——
//   - t.TempDir() 隔离 HOME
//   - mock systemctl（避免脚本调 user systemd 失败）
//   - 渲染脚本占位符后 bash 运行
//   - 读取 $HOME/.openclaw/openclaw.json 校验 providers 状态
package controller

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setModelScriptPath 定位 scripts/set_model.sh 绝对路径。
func setModelScriptPath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "scripts", "set_model.sh"))
	if err != nil {
		t.Fatalf("resolve script path: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("script not found at %s: %v", p, err)
	}
	return p
}

// prepareSetModelMockBin 在 mockBinDir 放置 systemctl/flock 等 mock，
// 保证脚本能完整跑完 jq 逻辑而不真正调系统服务。
// flock 用真实命令，systemctl 走 mock。
func prepareSetModelMockBin(t *testing.T, mockBinDir string) {
	t.Helper()
	if err := os.MkdirAll(mockBinDir, 0o755); err != nil {
		t.Fatalf("mkdir mock bin: %v", err)
	}
	// systemctl mock：吞掉 restart 请求
	mockSystemctl := filepath.Join(mockBinDir, "systemctl")
	if err := os.WriteFile(mockSystemctl,
		[]byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$HOME/systemctl.log\"\necho '[mock-systemctl] '\"$@\"\nexit 0\n"),
		0o755); err != nil {
		t.Fatalf("write mock systemctl: %v", err)
	}
}

// renderSetModelScript 把 set_model.sh 的占位符替换为具体值，写入临时文件。
// valueb64/fallbacksb64 由 caller 提供原始 JSON 串，本函数负责 base64 编码。
func renderSetModelScript(t *testing.T, srcPath, valueJSON, providerKey, modelSlug, primaryRef, fallbacksJSON string) string {
	t.Helper()
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	rendered := string(raw)
	valueb64 := base64.StdEncoding.EncodeToString([]byte(valueJSON))
	fallbacksb64 := base64.StdEncoding.EncodeToString([]byte(fallbacksJSON))
	rendered = strings.ReplaceAll(rendered, "{{valueb64}}", valueb64)
	rendered = strings.ReplaceAll(rendered, "{{fallbacksb64}}", fallbacksb64)
	rendered = strings.ReplaceAll(rendered, "{{provider}}", providerKey)
	rendered = strings.ReplaceAll(rendered, "{{model}}", modelSlug)
	rendered = strings.ReplaceAll(rendered, "{{primary}}", primaryRef)
	rendered = strings.ReplaceAll(rendered, "{{imageprimary}}", "")
	rendered = strings.ReplaceAll(rendered, "{{imagefallbacksb64}}", base64.StdEncoding.EncodeToString([]byte("[]")))

	dst := filepath.Join(t.TempDir(), "set_model_rendered.sh")
	if err := os.WriteFile(dst, []byte(rendered), 0o755); err != nil {
		t.Fatalf("write rendered script: %v", err)
	}
	return dst
}

// runSetModelScript 在隔离 HOME 下执行渲染后的脚本，返回 HOME 路径。
func runSetModelScript(t *testing.T, scriptPath, home string) {
	t.Helper()
	mockBin := filepath.Join(home, "mock-bin")
	prepareSetModelMockBin(t, mockBin)
	cmd := exec.Command("bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		// mock-bin 前置：systemctl 走 mock；flock/jq/base64 走系统真实路径
		"PATH="+mockBin+":"+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script execution failed: %v\n--- output ---\n%s", err, out)
	}
	t.Logf("script output:\n%s", out)
}

// readOpenclawConfig 读取 $HOME/.openclaw/openclaw.json，反序列化为 map。
func readOpenclawConfig(t *testing.T, home string) map[string]any {
	t.Helper()
	cfgPath := filepath.Join(home, ".openclaw", "openclaw.json")
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read openclaw.json: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse openclaw.json: %v\n--- raw ---\n%s", err, raw)
	}
	return cfg
}

// getProvidersKeys 从配置中提取 models.providers 的 key 列表。
func getProvidersKeys(t *testing.T, cfg map[string]any) []string {
	t.Helper()
	models, ok := cfg["models"].(map[string]any)
	if !ok {
		t.Fatalf("models 字段缺失或类型错误: %v", cfg["models"])
	}
	providers, ok := models["providers"].(map[string]any)
	if !ok {
		t.Fatalf("providers 字段缺失或类型错误: %v", models["providers"])
	}
	keys := make([]string, 0, len(providers))
	for k := range providers {
		keys = append(keys, k)
	}
	return keys
}

// containsString 简单包含判断。
func containsString(s []string, target string) bool {
	for _, v := range s {
		if v == target {
			return true
		}
	}
	return false
}

// TestSetModelScript_CoexistSameModelIDDifferentProvider 是本次 bug 的核心回归用例：
// 当 hatchery-deepseek-v3.1 已存在时，新增 custom-deepseek-v3.1（同 modelId 不同 provider）
// 不应导致 hatchery-deepseek-v3.1 被误删——新方案允许同 modelId 多 provider 并存。
func TestSetModelScript_CoexistSameModelIDDifferentProvider(t *testing.T) {
	ensureShellDeps(t) // 复用 set_model_ace_script_test.go 中的依赖检查
	scriptPath := setModelScriptPath(t)

	home := t.TempDir()
	cfgDir := filepath.Join(home, ".openclaw")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir openclaw cfg dir: %v", err)
	}

	// 预置：hatchery-deepseek-v3.1 已写入 providers，并作为 fallback ref 存在
	preExisting := `{
        "models": {
            "providers": {
                "hatchery-deepseek-v3.1": {
                    "baseUrl": "https://hatchery-proxy.example.com/v1",
                    "apiKey": "sk-hatchery",
                    "auth": "api-key",
                    "api": "openai-completions",
                    "models": [{"id": "deepseek-v3.1", "name": "deepseek-v3.1"}]
                },
                "hatchery-other-model": {
                    "baseUrl": "https://hatchery-proxy.example.com/v1",
                    "apiKey": "sk-hatchery",
                    "auth": "api-key",
                    "api": "openai-completions",
                    "models": [{"id": "other-model", "name": "other-model"}]
                }
            }
        },
        "agents": {
            "defaults": {
                "model": {
                    "primary": "hatchery-other-model/other-model",
                    "fallbacks": ["hatchery-deepseek-v3.1/deepseek-v3.1"]
                }
            }
        }
    }`
	if err := os.WriteFile(filepath.Join(cfgDir, "openclaw.json"), []byte(preExisting), 0o644); err != nil {
		t.Fatalf("write preexisting config: %v", err)
	}

	// 现在执行 set_model.sh，新增 custom-deepseek-v3.1（用户侧自定义模型）
	// fallbacks 同时包含旧的 hatchery-deepseek-v3.1 和新的 custom-deepseek-v3.1
	customValue := `{
        "baseUrl": "https://api.example.com/v1",
        "apiKey": "sk-custom",
        "auth": "api-key",
        "api": "openai-completions",
        "models": [{"id": "deepseek-v3.1", "name": "deepseek-v3.1"}]
    }`
	primaryRef := "hatchery-other-model/other-model"
	fallbacksJSON := `["hatchery-deepseek-v3.1/deepseek-v3.1","custom-deepseek-v3.1/deepseek-v3.1"]`

	rendered := renderSetModelScript(t, scriptPath,
		customValue, "custom-deepseek-v3.1", "deepseek-v3.1",
		primaryRef, fallbacksJSON)
	runSetModelScript(t, rendered, home)

	cfg := readOpenclawConfig(t, home)
	keys := getProvidersKeys(t, cfg)

	// 核心断言 1：custom-deepseek-v3.1 已写入
	if !containsString(keys, "custom-deepseek-v3.1") {
		t.Errorf("新增的 custom-deepseek-v3.1 未写入 providers，实际 keys=%v", keys)
	}
	// 核心断言 2：hatchery-deepseek-v3.1 不应被误删（这是 bug 修复点）
	if !containsString(keys, "hatchery-deepseek-v3.1") {
		t.Errorf("【BUG 回归】hatchery-deepseek-v3.1 被误删！实际 keys=%v", keys)
	}
	// 核心断言 3：其他无关 provider 不受影响
	if !containsString(keys, "hatchery-other-model") {
		t.Errorf("无关 provider hatchery-other-model 被误删，实际 keys=%v", keys)
	}

	// 核心断言 4：fallback ref 与 providers key 一一对应（无悬空引用）
	agents := cfg["agents"].(map[string]any)
	defaults := agents["defaults"].(map[string]any)
	modelObj := defaults["model"].(map[string]any)
	fallbacks := modelObj["fallbacks"].([]any)
	for _, f := range fallbacks {
		ref := f.(string)
		idx := strings.Index(ref, "/")
		if idx <= 0 {
			t.Errorf("fallback ref 格式错误：%q", ref)
			continue
		}
		key := ref[:idx]
		if !containsString(keys, key) {
			t.Errorf("fallback ref %q 引用的 providerKey %q 不存在于 providers，悬空引用！keys=%v",
				ref, key, keys)
		}
	}
}

// TestSetModelScript_LegacyBareKeyMigration 验证旧版"裸 provider key"
// （如 "hatchery"）应被正确清理——新格式 key 写入后，旧裸 key 应消失。
func TestSetModelScript_LegacyBareKeyMigration(t *testing.T) {
	ensureShellDeps(t)
	scriptPath := setModelScriptPath(t)

	home := t.TempDir()
	cfgDir := filepath.Join(home, ".openclaw")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir openclaw cfg dir: %v", err)
	}

	// 预置：旧格式裸 key "hatchery" 持有 deepseek-v3.1
	preExisting := `{
        "models": {
            "providers": {
                "hatchery": {
                    "baseUrl": "https://old.example.com/v1",
                    "apiKey": "sk-old",
                    "auth": "api-key",
                    "api": "openai-completions",
                    "models": [{"id": "deepseek-v3.1", "name": "deepseek-v3.1"}]
                }
            }
        },
        "agents": {"defaults": {"model": {"primary": "hatchery/deepseek-v3.1", "fallbacks": []}}}
    }`
	if err := os.WriteFile(filepath.Join(cfgDir, "openclaw.json"), []byte(preExisting), 0o644); err != nil {
		t.Fatalf("write preexisting config: %v", err)
	}

	newValue := `{
        "baseUrl": "https://new.example.com/v1",
        "apiKey": "sk-new",
        "auth": "api-key",
        "api": "openai-completions",
        "models": [{"id": "deepseek-v3.1", "name": "deepseek-v3.1"}]
    }`
	rendered := renderSetModelScript(t, scriptPath,
		newValue, "hatchery-deepseek-v3.1", "deepseek-v3.1",
		"hatchery-deepseek-v3.1/deepseek-v3.1", "[]")
	runSetModelScript(t, rendered, home)

	cfg := readOpenclawConfig(t, home)
	keys := getProvidersKeys(t, cfg)

	// 新格式 key 应存在
	if !containsString(keys, "hatchery-deepseek-v3.1") {
		t.Errorf("新格式 key hatchery-deepseek-v3.1 未写入，实际 keys=%v", keys)
	}
	// 旧裸 key "hatchery" 应被清理（持有相同 modelId）
	if containsString(keys, "hatchery") {
		t.Errorf("旧裸 key hatchery 未被清理，实际 keys=%v", keys)
	}
}

// TestSetModelScript_LegacyBareKeyDifferentModelID_Preserved 验证：
// 旧裸 key 持有的 modelId 与本次写入的 modelId 不同时，不应被误清理。
// 这保证清理逻辑只针对"同 modelId 的旧裸 key"，不会误删其他 modelId 的旧 key。
func TestSetModelScript_LegacyBareKeyDifferentModelID_Preserved(t *testing.T) {
	ensureShellDeps(t)
	scriptPath := setModelScriptPath(t)

	home := t.TempDir()
	cfgDir := filepath.Join(home, ".openclaw")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir openclaw cfg dir: %v", err)
	}

	// 预置：旧裸 key "hatchery" 持有 OTHER 模型；本次写入 deepseek-v3.1
	preExisting := `{
        "models": {
            "providers": {
                "hatchery": {
                    "baseUrl": "https://old.example.com/v1",
                    "apiKey": "sk-old",
                    "auth": "api-key",
                    "api": "openai-completions",
                    "models": [{"id": "qwen-max", "name": "qwen-max"}]
                }
            }
        },
        "agents": {"defaults": {"model": {"primary": "hatchery/qwen-max", "fallbacks": []}}}
    }`
	if err := os.WriteFile(filepath.Join(cfgDir, "openclaw.json"), []byte(preExisting), 0o644); err != nil {
		t.Fatalf("write preexisting config: %v", err)
	}

	newValue := `{
        "baseUrl": "https://new.example.com/v1",
        "apiKey": "sk-new",
        "auth": "api-key",
        "api": "openai-completions",
        "models": [{"id": "deepseek-v3.1", "name": "deepseek-v3.1"}]
    }`
	rendered := renderSetModelScript(t, scriptPath,
		newValue, "hatchery-deepseek-v3.1", "deepseek-v3.1",
		"hatchery-deepseek-v3.1/deepseek-v3.1", "[]")
	runSetModelScript(t, rendered, home)

	cfg := readOpenclawConfig(t, home)
	keys := getProvidersKeys(t, cfg)

	// 新 key 写入
	if !containsString(keys, "hatchery-deepseek-v3.1") {
		t.Errorf("新 key 未写入，实际 keys=%v", keys)
	}
	// 旧裸 key "hatchery" 持有的是 qwen-max（不同 modelId），不应被清理
	if !containsString(keys, "hatchery") {
		t.Errorf("旧裸 key hatchery（持不同 modelId qwen-max）被误清理，实际 keys=%v", keys)
	}
}

// TestSetModelScript_ProviderPrefixChange_Rename 验证：
// 当模型的 Provider 字段从 "hatchery" 变为 "deepseek" 后（如版本升级），
// 新增其他模型时，存量 primary 的 provider key 应从 "hatchery-deepseek-chat"
// 被正确 rename 为 "deepseek-deepseek-chat"，使 primary ref 能找到对应的 provider 配置。
func TestSetModelScript_ProviderPrefixChange_Rename(t *testing.T) {
	ensureShellDeps(t)
	scriptPath := setModelScriptPath(t)

	home := t.TempDir()
	cfgDir := filepath.Join(home, ".openclaw")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir openclaw cfg dir: %v", err)
	}

	// 预置：旧版本写入的 provider key 是 "hatchery-deepseek-chat"（Provider="hatchery" 时生成）
	preExisting := `{
        "models": {
            "providers": {
                "hatchery-deepseek-chat": {
                    "baseUrl": "http://proxy.example.com/v1",
                    "apiKey": "sk-proxy",
                    "auth": "api-key",
                    "api": "openai-completions",
                    "models": [{"id": "deepseek-chat", "name": "deepseek-chat"}]
                },
                "custom-deepseek-v3.1": {
                    "baseUrl": "https://api.example.com/v1",
                    "apiKey": "sk-custom",
                    "auth": "api-key",
                    "api": "openai-completions",
                    "models": [{"id": "deepseek-v3.1", "name": "deepseek-v3.1"}]
                }
            }
        },
        "agents": {
            "defaults": {
                "model": {
                    "primary": "hatchery-deepseek-chat/deepseek-chat",
                    "fallbacks": ["custom-deepseek-v3.1/DeepSeek-V3.1"]
                }
            }
        }
    }`
	if err := os.WriteFile(filepath.Join(cfgDir, "openclaw.json"), []byte(preExisting), 0o644); err != nil {
		t.Fatalf("write preexisting config: %v", err)
	}

	// 新版本中 deepseek-chat 的 Provider 变为 "deepseek"，resolveBindingRef 生成的 ref 变为
	// "deepseek-deepseek-chat/deepseek-chat"。此时通过 add-model 新增 nvidia 模型，
	// set_model.sh 收到的 primary 是新格式 ref。
	newValue := `{
        "baseUrl": "http://proxy.example.com/v1",
        "apiKey": "sk-proxy",
        "auth": "api-key",
        "api": "openai-completions",
        "models": [{"id": "nvidia-nemotron", "name": "nvidia-nemotron"}]
    }`
	// primary 使用新格式 ref（Provider="deepseek" 生成的 key）
	primaryRef := "deepseek-deepseek-chat/deepseek-chat"
	fallbacksJSON := `["custom-deepseek-v3.1/DeepSeek-V3.1","hatchery-nvidia-nemotron/nvidia-nemotron"]`

	rendered := renderSetModelScript(t, scriptPath,
		newValue, "hatchery-nvidia-nemotron", "nvidia-nemotron",
		primaryRef, fallbacksJSON)
	runSetModelScript(t, rendered, home)

	cfg := readOpenclawConfig(t, home)
	keys := getProvidersKeys(t, cfg)

	// 核心断言 1：旧 key "hatchery-deepseek-chat" 应被 rename 为 "deepseek-deepseek-chat"
	if containsString(keys, "hatchery-deepseek-chat") {
		t.Errorf("旧 key hatchery-deepseek-chat 未被迁移，实际 keys=%v", keys)
	}
	if !containsString(keys, "deepseek-deepseek-chat") {
		t.Errorf("新 key deepseek-deepseek-chat 未出现（应从 hatchery-deepseek-chat rename），实际 keys=%v", keys)
	}

	// 核心断言 2：新增的 nvidia 模型 provider 已写入
	if !containsString(keys, "hatchery-nvidia-nemotron") {
		t.Errorf("新增的 hatchery-nvidia-nemotron 未写入，实际 keys=%v", keys)
	}

	// 核心断言 3：custom-deepseek-v3.1 不受影响
	if !containsString(keys, "custom-deepseek-v3.1") {
		t.Errorf("custom-deepseek-v3.1 被误删，实际 keys=%v", keys)
	}

	// 核心断言 4：primary 指向正确
	agents := cfg["agents"].(map[string]any)
	defaults := agents["defaults"].(map[string]any)
	modelObj := defaults["model"].(map[string]any)
	if modelObj["primary"] != "deepseek-deepseek-chat/deepseek-chat" {
		t.Errorf("primary 应为 deepseek-deepseek-chat/deepseek-chat，实际=%v", modelObj["primary"])
	}

	// 核心断言 5：rename 后的 provider 内容正确（保留原始配置）
	models := cfg["models"].(map[string]any)
	providers := models["providers"].(map[string]any)
	renamed := providers["deepseek-deepseek-chat"].(map[string]any)
	renamedModels := renamed["models"].([]any)
	if len(renamedModels) != 1 {
		t.Fatalf("rename 后 models 数量应为 1，实际=%d", len(renamedModels))
	}
	firstModel := renamedModels[0].(map[string]any)
	if firstModel["id"] != "deepseek-chat" {
		t.Errorf("rename 后 model id 应为 deepseek-chat，实际=%v", firstModel["id"])
	}
}

func TestSetModelScript_BatchWritesAllProvidersAndRestartsOnce(t *testing.T) {
	for _, bin := range []string{"bash", "base64", "jq"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("依赖 %s 未安装，跳过", bin)
		}
	}

	scriptPath := setModelScriptPath(t)
	home := t.TempDir()
	mockBin := filepath.Join(home, "mock-bin")
	if err := os.MkdirAll(mockBin, 0o755); err != nil {
		t.Fatalf("mkdir mock bin: %v", err)
	}
	mockFlock := filepath.Join(mockBin, "flock")
	if err := os.WriteFile(mockFlock, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write mock flock: %v", err)
	}

	batchValue := `{
        "mode": "batch",
        "providers": [
            {
                "provider": "hatchery-primary",
                "model": "primary",
                "value": {
                    "baseUrl": "https://proxy.example.com/v1",
                    "apiKey": "sk-primary",
                    "auth": "api-key",
                    "api": "openai-completions",
                    "models": [{"id": "primary", "name": "primary"}]
                }
            },
            {
                "provider": "hatchery-fallback",
                "model": "fallback",
                "value": {
                    "baseUrl": "https://proxy.example.com/v1",
                    "apiKey": "sk-fallback",
                    "auth": "api-key",
                    "api": "openai-completions",
                    "models": [{"id": "fallback", "name": "fallback"}]
                }
            }
        ]
    }`
	rendered := renderSetModelScript(
		t,
		scriptPath,
		batchValue,
		"hatchery-primary",
		"primary",
		"hatchery-primary/primary",
		`["hatchery-fallback/fallback"]`,
	)
	runSetModelScript(t, rendered, home)

	cfg := readOpenclawConfig(t, home)
	keys := getProvidersKeys(t, cfg)
	for _, want := range []string{"hatchery-primary", "hatchery-fallback"} {
		if !containsString(keys, want) {
			t.Fatalf("batch provider %q 未写入，实际 keys=%v", want, keys)
		}
	}
	agents := cfg["agents"].(map[string]any)
	defaults := agents["defaults"].(map[string]any)
	modelObj := defaults["model"].(map[string]any)
	if modelObj["primary"] != "hatchery-primary/primary" {
		t.Fatalf("primary=%v, want hatchery-primary/primary", modelObj["primary"])
	}
	fallbacks := modelObj["fallbacks"].([]any)
	if len(fallbacks) != 1 || fallbacks[0] != "hatchery-fallback/fallback" {
		t.Fatalf("fallbacks=%v, want [hatchery-fallback/fallback]", fallbacks)
	}

	restartLog, err := os.ReadFile(filepath.Join(home, "systemctl.log"))
	if err != nil {
		t.Fatalf("read systemctl log: %v", err)
	}
	restarts := strings.FieldsFunc(strings.TrimSpace(string(restartLog)), func(r rune) bool { return r == '\n' })
	if len(restarts) != 1 {
		t.Fatalf("gateway restart 次数=%d, want 1, log=%q", len(restarts), restartLog)
	}
}
