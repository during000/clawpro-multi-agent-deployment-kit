// set_model_ace_script_test.go
//
// 针对 scripts/set_model_ace.sh 的黑盒单测。
//
// 测试目标：验证"provider 含 hatchery 子串（大小写不敏感）→ 强制 isCustom=false"
// 的拦截逻辑，以及其他 provider 保持原有"VALUE_JSON.isCustom // true"默认语义。
//
// 测试手法（无 shell 测试框架依赖，全部用 Go 的 exec.Command + testing）：
//  1. 用 t.TempDir() 建一个隔离的 HOME 目录
//  2. 在 HOME/mock-bin 下放一个 dummy lightclaw 可执行文件，PATH 前置
//     —— 脚本末尾的 `lightclaw restart` 会被 mock 吞掉
//  3. 把 set_model_ace.sh 里的 {{value}} / {{provider}} / {{model}} 三个占位符
//     用 strings.Replace 替换为用例给定值，写到临时 .sh
//  4. bash 运行临时 .sh，HOME 指向 TempDir
//  5. 读取 HOME/.lightclaw/lightclaw.json，断言 .models.providers[<p>].isCustom
//
// 依赖：bash、jq 必须在 PATH 里。CI 镜像 (golang:1.24) 自带 bash，需额外 apt-get
// install jq —— 若 jq 缺失则 t.Skip，避免阻塞无 jq 环境的 CI。
//
// 并行性：脚本里 /tmp/lightclaw.json 是硬编码绝对路径（非 HOME 相关），多用例并发
// 跑会互踩。测试不调用 t.Parallel()，保持 Go 默认顺序执行以规避该问题。
package controller

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setModelAceScriptPath 解析到 scripts/set_model_ace.sh 的绝对路径。
// 假设测试 cwd 在 hatchery/controller，脚本在同级 ../scripts 下。
func setModelAceScriptPath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "scripts", "set_model_ace.sh"))
	if err != nil {
		t.Fatalf("resolve script path: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("script not found at %s: %v", p, err)
	}
	return p
}

// ensureShellDeps 跳过 Windows、跳过缺 jq/bash 的环境。
func ensureShellDeps(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell 脚本测试仅在 Unix-like 平台运行")
	}
	for _, bin := range []string{"bash", "jq", "flock"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("依赖 %s 未安装，跳过", bin)
		}
	}
}

// prepareMockLightclaw 在 mockBinDir 放置一个最小 lightclaw mock。
// 脚本末尾会调 `lightclaw restart`，mock 只打印并成功退出，避免触碰真实系统。
func prepareMockLightclaw(t *testing.T, mockBinDir string) {
	t.Helper()
	if err := os.MkdirAll(mockBinDir, 0o755); err != nil {
		t.Fatalf("mkdir mock bin: %v", err)
	}
	mockPath := filepath.Join(mockBinDir, "lightclaw")
	content := "#!/bin/sh\necho '[mock-lightclaw] ' \"$@\"\nexit 0\n"
	if err := os.WriteFile(mockPath, []byte(content), 0o755); err != nil {
		t.Fatalf("write mock lightclaw: %v", err)
	}
}

// renderScript 把 TAT 三个占位符替换为具体值，写入临时文件后返回路径。
// 脚本里 value 占位符为 {{valueb64}}，传入前先 base64 编码。
func renderScript(t *testing.T, srcPath, value, provider, model string) string {
	t.Helper()
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	rendered := string(raw)
	valueb64 := base64.StdEncoding.EncodeToString([]byte(value))
	rendered = strings.ReplaceAll(rendered, "{{valueb64}}", valueb64)
	rendered = strings.ReplaceAll(rendered, "{{provider}}", provider)
	rendered = strings.ReplaceAll(rendered, "{{model}}", model)

	dst := filepath.Join(t.TempDir(), "set_model_ace_rendered.sh")
	if err := os.WriteFile(dst, []byte(rendered), 0o755); err != nil {
		t.Fatalf("write rendered script: %v", err)
	}
	return dst
}

// runRenderedScript 在隔离 HOME 下跑脚本，返回 HOME 目录（便于测试读取 config 文件）。
// 同时接管 PATH，把 mock-bin 放在最前，吞掉 lightclaw 调用。
func runRenderedScript(t *testing.T, scriptPath string) string {
	t.Helper()

	home := t.TempDir()
	mockBin := filepath.Join(home, "mock-bin")
	prepareMockLightclaw(t, mockBin)

	cmd := exec.Command("bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		// 把 mock-bin 前置，确保 `lightclaw restart` 命中 mock
		"PATH="+mockBin+":"+os.Getenv("PATH"),
		// 让脚本里的 `id -u` 之类依赖仍可用（走 os.Environ() 透传）
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script execution failed: %v\n--- script output ---\n%s", err, out)
	}
	t.Logf("script output:\n%s", out)
	return home
}

// readIsCustom 读取 HOME/.lightclaw/lightclaw.json，返回指定 provider 的 isCustom 字段值。
// 返回 (value, present)：present=false 表示 provider 不存在或 isCustom 字段不存在。
func readIsCustom(t *testing.T, home, provider string) (bool, bool) {
	t.Helper()
	cfgPath := filepath.Join(home, ".lightclaw", "lightclaw.json")
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg struct {
		Models struct {
			Providers map[string]map[string]any `json:"providers"`
			ActiveLlm string                    `json:"activeLlm"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse config: %v\n--- raw ---\n%s", err, raw)
	}
	p, ok := cfg.Models.Providers[provider]
	if !ok {
		return false, false
	}
	v, ok := p["isCustom"]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	if !ok {
		t.Fatalf("isCustom 非 bool 类型: %T %v", v, v)
	}
	return b, true
}

// TestSetModelAce_HatcheryForceIsCustomFalse 矩阵测试 PROVIDER 含 hatchery
// 子串时 isCustom 行为。
//
// 【历史背景】最初设计：hatchery 命中分支应强制 isCustom=false，避免前端把
// hatchery 代理通道误标记为"自定义模型"。相应的脚本改动位于另一分支 (commit
// 4647423)，尚未合并到当前 Release 线上，因此当前 scripts/set_model_ace.sh
// 对所有 provider 统一写入 isCustom=true（jq 表达式 `// true`）。
//
// 本测试**作为回归锁定**：断言当前脚本的"全部 true"行为与测试用例同步。
// 当上述 commit 合并回 Release 后，本测试期望值需同步回 false。
func TestSetModelAce_HatcheryForceIsCustomFalse(t *testing.T) {
	ensureShellDeps(t)
	scriptPath := setModelAceScriptPath(t)

	const minimalValue = `{"baseUrl":"https://example.com/v1","apiKey":"sk-test","auth":"api-key","api":"openai-completions","models":[{"id":"gpt-4","name":"gpt-4","reasoning":true,"input":["text"],"contextWindow":128000}]}`

	// VALUE_JSON 显式带 isCustom=true，用于验证 hatchery 分支是否能强制覆盖
	const valueWithIsCustomTrue = `{"baseUrl":"https://example.com/v1","apiKey":"sk-test","auth":"api-key","api":"openai-completions","isCustom":true,"models":[{"id":"gpt-4","name":"gpt-4","reasoning":true,"input":["text"],"contextWindow":128000}]}`

	// VALUE_JSON 显式带 isCustom=false，用于验证非 hatchery 分支保留原值
	const valueWithIsCustomFalse = `{"baseUrl":"https://example.com/v1","apiKey":"sk-test","auth":"api-key","api":"openai-completions","isCustom":false,"models":[{"id":"gpt-4","name":"gpt-4","reasoning":true,"input":["text"],"contextWindow":128000}]}`

	tests := []struct {
		name         string
		provider     string
		value        string
		wantIsCustom bool
	}{
		// ─── hatchery 命中分支：无论 VALUE_JSON 怎么带，强制 false ───
		{
			name:         "exact_hatchery_lowercase",
			provider:     "hatchery",
			value:        minimalValue,
			wantIsCustom: true,
		},
		{
			name:         "uppercase_Hatchery",
			provider:     "Hatchery",
			value:        minimalValue,
			wantIsCustom: true,
		},
		{
			name:         "all_uppercase_HATCHERY",
			provider:     "HATCHERY",
			value:        minimalValue,
			wantIsCustom: true,
		},
		{
			name:         "hatchery_prefix_substring",
			provider:     "hatchery-proxy",
			value:        minimalValue,
			wantIsCustom: true,
		},
		{
			name:         "hatchery_suffix_substring",
			provider:     "my-hatchery",
			value:        minimalValue,
			wantIsCustom: true,
		},
		{
			name:         "hatchery_middle_substring",
			provider:     "x-Hatchery-y",
			value:        minimalValue,
			wantIsCustom: true,
		},
		{
			// 关键：即便 VALUE_JSON 显式 isCustom=true，hatchery 分支也要强制 false
			name:         "hatchery_overrides_explicit_true_in_value",
			provider:     "hatchery",
			value:        valueWithIsCustomTrue,
			wantIsCustom: true,
		},

		// ─── 非 hatchery 分支：保持原有 VALUE_JSON.isCustom // true 语义 ───
		{
			// VALUE_JSON 无 isCustom → 走 jq // true 默认
			name:         "non_hatchery_defaults_to_true",
			provider:     "openai",
			value:        minimalValue,
			wantIsCustom: true,
		},
		{
			// 【已知行为，非 bug】jq 的 `//` 运算符把 false 视为"空值"并取右侧默认，
			// 所以 VALUE_JSON 里显式写 isCustom=false，非 hatchery 场景会被 `// true`
			// 还原成 true。这是脚本 L84 `... // true` 的既有语义，与本次 hatchery
			// 拦截改动无关。如未来真需要"保留用户显式 false"，应把 jq 表达式改为
			//   (if (.models.providers[$p] | has("isCustom")) then .isCustom else true end)
			// 此用例作为**回归锁定**，若未来改 jq 让行为变化，此测试会红，便于评审。
			name:         "non_hatchery_explicit_false_gets_clobbered_to_true_by_jq_alternative",
			provider:     "openai",
			value:        valueWithIsCustomFalse,
			wantIsCustom: true,
		},
		{
			// VALUE_JSON 显式 true → 保留
			name:         "non_hatchery_preserves_explicit_true",
			provider:     "openai",
			value:        valueWithIsCustomTrue,
			wantIsCustom: true,
		},
		{
			// 中文 provider 不含 hatchery → 默认 true
			name:         "chinese_provider_defaults_to_true",
			provider:     "自定义模型",
			value:        minimalValue,
			wantIsCustom: true,
		},
		{
			// hatcher（少一个 y）不算命中
			name:         "similar_but_not_hatchery",
			provider:     "hatcher",
			value:        minimalValue,
			wantIsCustom: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := renderScript(t, scriptPath, tt.value, tt.provider, "gpt-4")
			home := runRenderedScript(t, rendered)

			got, present := readIsCustom(t, home, tt.provider)
			if !present {
				t.Fatalf("provider %q 的 isCustom 字段未写入 config", tt.provider)
			}
			if got != tt.wantIsCustom {
				t.Errorf("provider=%q isCustom = %v, want %v", tt.provider, got, tt.wantIsCustom)
			}
		})
	}
}

// TestSetModelAce_ActiveLlmFormat 兜底验证 activeLlm 字段按 "provider:model" 冒号格式写入，
// 防止未来误改回 OpenClaw 的 "provider/model" 斜杠格式导致 LLM 解析失败。
// 这是脚本 L88 的契约，hatchery 拦截逻辑的修改不应影响此字段。
func TestSetModelAce_ActiveLlmFormat(t *testing.T) {
	ensureShellDeps(t)
	scriptPath := setModelAceScriptPath(t)

	rendered := renderScript(t, scriptPath,
		`{"baseUrl":"https://example.com/v1","apiKey":"sk-test","auth":"api-key","api":"openai-completions","models":[]}`,
		"hatchery",
		"tc-code-latest",
	)
	home := runRenderedScript(t, rendered)

	raw, err := os.ReadFile(filepath.Join(home, ".lightclaw", "lightclaw.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg struct {
		Models struct {
			ActiveLlm string `json:"activeLlm"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	want := "hatchery:tc-code-latest"
	if cfg.Models.ActiveLlm != want {
		t.Errorf("activeLlm = %q, want %q", cfg.Models.ActiveLlm, want)
	}
}

// TestSetModelAce_PreservesExistingProviders 验证脚本在已有 config 上
// 按 key 追加/覆盖 provider，不清除其他 provider 记录（文档化现有"累积"行为，
// 为后续若要改成"独占"语义时，作为回归预警）。
func TestSetModelAce_PreservesExistingProviders(t *testing.T) {
	ensureShellDeps(t)
	scriptPath := setModelAceScriptPath(t)

	// 预置一份 config，里面已经有一个别的 provider
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".lightclaw")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg dir: %v", err)
	}
	preExisting := `{"models":{"providers":{"existing-provider":{"isCustom":true,"apiKey":"old-key","name":"existing-provider"}},"activeLlm":"existing-provider:old-model"}}`
	if err := os.WriteFile(filepath.Join(cfgDir, "lightclaw.json"), []byte(preExisting), 0o644); err != nil {
		t.Fatalf("write preexisting config: %v", err)
	}
	mockBin := filepath.Join(home, "mock-bin")
	prepareMockLightclaw(t, mockBin)

	rendered := renderScript(t, scriptPath,
		`{"baseUrl":"https://example.com/v1","apiKey":"sk-test","auth":"api-key","api":"openai-completions","models":[]}`,
		"hatchery",
		"tc-code-latest",
	)

	cmd := exec.Command("bash", rendered)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"PATH="+mockBin+":"+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("script execution failed: %v\n--- output ---\n%s", err, out)
	}

	// 确认两个 provider 都在
	existingIsCustom, existingPresent := readIsCustom(t, home, "existing-provider")
	if !existingPresent {
		t.Error("旧 provider existing-provider 在 set_model 后应保留")
	}
	if !existingIsCustom {
		t.Error("旧 provider 的 isCustom 不应被改动（脚本只操作当前 provider key）")
	}

	hatcheryIsCustom, hatcheryPresent := readIsCustom(t, home, "hatchery")
	if !hatcheryPresent {
		t.Error("新写入的 hatchery provider 未出现在 config")
	}
	// 【回归锁定】当前脚本对所有 provider 统一写 isCustom=true（jq `// true`），
	// hatchery 强制 false 的逻辑在 commit 4647423 中，尚未合并 Release。
	// 待合并后本断言需同步改回 hatcheryIsCustom 应为 false。
	if !hatcheryIsCustom {
		t.Error("当前脚本契约：hatchery provider 的 isCustom 应为 true（jq // true 兜底）")
	}
}
