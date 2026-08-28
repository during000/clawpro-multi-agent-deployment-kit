package model

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var aiModelTestDBCounter int64

// setupAIModelTestDB creates an isolated SQLite memory database for testing.
func setupAIModelTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	id := atomic.AddInt64(&aiModelTestDBCounter, 1)
	dsn := fmt.Sprintf("file:aiModelTest%d?mode=memory&cache=shared", id)
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite mem: %v", err)
	}
	if err := gdb.AutoMigrate(&AIModel{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return gdb
}

// TestSeedModels_CreatesBuiltinModel verifies that SeedModels creates builtin model on first call.
func TestSeedModels_CreatesBuiltinModel(t *testing.T) {
	gdb := setupAIModelTestDB(t)

	// First call should create the builtin model
	err := gdb.Transaction(func(tx *gorm.DB) error {
		return SeedModels(tx)
	})
	if err != nil {
		t.Fatalf("SeedModels: %v", err)
	}

	var count int64
	if err := gdb.Model(&AIModel{}).Where("provider = ? AND model_id = ?", BuiltinModelProvider, BuiltinModelID).Count(&count).Error; err != nil {
		t.Fatalf("count builtin model: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 builtin model, got %d", count)
	}
}

// TestSeedModels_IdempotentCall verifies that calling SeedModels twice doesn't create duplicates.
func TestSeedModels_IdempotentCall(t *testing.T) {
	gdb := setupAIModelTestDB(t)

	// First call
	err := gdb.Transaction(func(tx *gorm.DB) error {
		return SeedModels(tx)
	})
	if err != nil {
		t.Fatalf("SeedModels first call: %v", err)
	}

	// Second call should not create duplicate
	err = gdb.Transaction(func(tx *gorm.DB) error {
		return SeedModels(tx)
	})
	if err != nil {
		t.Fatalf("SeedModels second call: %v", err)
	}

	var count int64
	if err := gdb.Model(&AIModel{}).Where("provider = ? AND model_id = ?", BuiltinModelProvider, BuiltinModelID).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 builtin model after two calls, got %d", count)
	}
}

// TestSeedModels_BuiltinModelDisabledByDefault verifies that builtin model
// is enabled but invisible by default (after enabled/visible decoupling).
func TestSeedModels_BuiltinModelDisabledByDefault(t *testing.T) {
	gdb := setupAIModelTestDB(t)

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return SeedModels(tx)
	})
	if err != nil {
		t.Fatalf("SeedModels: %v", err)
	}

	var model AIModel
	if err := gdb.Where("provider = ? AND model_id = ?", BuiltinModelProvider, BuiltinModelID).First(&model).Error; err != nil {
		t.Fatalf("find builtin model: %v", err)
	}
	// 占位记录 Enabled 仍保留旧默认 false，需管理员主动打开"开放自定义模型"
	if model.Enabled {
		t.Fatalf("builtin model.Enabled should be false by default")
	}
	if model.Visible {
		t.Fatalf("builtin model.Visible should be false by default")
	}
}

// TestSeedModels_WithClosedDatabase verifies error handling when database operations fail.
func TestSeedModels_WithClosedDatabase(t *testing.T) {
	gdb := setupAIModelTestDB(t)

	// Get the underlying sqlDB and close it to simulate database error
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("get sqlDB: %v", err)
	}
	sqlDB.Close()

	// Now attempt to seed models on the closed database
	// This should fail during the count or create operation
	err = gdb.Transaction(func(tx *gorm.DB) error {
		return SeedModels(tx)
	})
	if err == nil {
		t.Fatalf("expected error with closed database, got nil")
	}
}

// TestGetInputTypes_JSONArray tests JSON array format parsing
func TestGetInputTypes_JSONArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "JSON array with text and image",
			input:    `["text","image"]`,
			expected: []string{"text", "image"},
		},
		{
			name:     "JSON array with single element",
			input:    `["text"]`,
			expected: []string{"text"},
		},
		{
			name:     "Empty JSON array returns empty",
			input:    `[]`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := AIModel{InputTypes: tt.input}
			result := model.GetInputTypes()
			if !sliceEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestGetInputTypes_CommaSeparated tests comma-separated format parsing
func TestGetInputTypes_CommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple comma-separated",
			input:    "text,image",
			expected: []string{"text", "image"},
		},
		{
			name:     "Comma-separated with spaces",
			input:    " text , image , video ",
			expected: []string{"text", "image", "video"},
		},
		{
			name:     "Single value",
			input:    "text",
			expected: []string{"text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := AIModel{InputTypes: tt.input}
			result := model.GetInputTypes()
			if !sliceEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestGetInputTypes_EdgeCases tests edge cases and defaults
func TestGetInputTypes_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Empty string defaults to text",
			input:    "",
			expected: []string{"text"},
		},
		{
			name:     "Whitespace only defaults to text",
			input:    "   ",
			expected: []string{"text"},
		},
		{
			name:     "Invalid JSON falls back to comma-separated",
			input:    "[invalid]",
			expected: []string{"[invalid]"},
		},
		{
			name:     "Empty string after stripping spaces defaults to text",
			input:    "  ",
			expected: []string{"text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := AIModel{InputTypes: tt.input}
			result := model.GetInputTypes()
			if !sliceEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDisplayName tests the DisplayName method formatting
func TestDisplayName(t *testing.T) {
	tests := []struct {
		name           string
		provider       string
		modelID        string
		expectedResult string
	}{
		{
			name:           "OpenAI GPT-4",
			provider:       "openai",
			modelID:        "gpt-4",
			expectedResult: "openai/gpt-4",
		},
		{
			name:           "Anthropic Claude",
			provider:       "anthropic",
			modelID:        "claude-3-opus",
			expectedResult: "anthropic/claude-3-opus",
		},
		{
			name:           "Custom model",
			provider:       "hatchery",
			modelID:        "custom",
			expectedResult: "hatchery/custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := AIModel{Provider: tt.provider, ModelID: tt.modelID}
			result := model.DisplayName()
			if result != tt.expectedResult {
				t.Errorf("got %q, want %q", result, tt.expectedResult)
			}
		})
	}
}

// Helper function to compare two string slices
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestSlugifyModelID 验证白名单 slug 化逻辑：
// 只保留 [a-z0-9._-]，其他字符替换为 -，合并连续 -，去除首尾 -。
func TestSlugifyModelID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// 普通模型（无变化）
		{name: "普通模型无变化", input: "deepseek-v3.2", expected: "deepseek-v3.2"},
		{name: "纯小写字母数字", input: "deepseek-chat", expected: "deepseek-chat"},
		{name: "含下划线", input: "gpt_4o", expected: "gpt_4o"},
		{name: "含点号", input: "glm-5.1", expected: "glm-5.1"},

		// 大写转小写
		{name: "大写转小写", input: "GLM-5.1", expected: "glm-5.1"},
		{name: "混合大小写", input: "DeepSeek-V3.2", expected: "deepseek-v3.2"},

		// 含 / 的模型
		{name: "含斜杠", input: "minimax/minimax-m2.5", expected: "minimax-minimax-m2.5"},
		{name: "含斜杠和大写", input: "ZhiJia/GLM-5.1-Plus", expected: "zhijia-glm-5.1-plus"},
		{name: "openrouter 格式", input: "openrouter/deepseek-v3", expected: "openrouter-deepseek-v3"},

		// 含 : 的模型
		{name: "含冒号", input: "minimax-m2.5:free", expected: "minimax-m2.5-free"},
		{name: "含冒号和版本", input: "model:v1.0", expected: "model-v1.0"},

		// 含 / 和 : 的模型
		{name: "含斜杠和冒号", input: "minimax/minimax-m2.5:free", expected: "minimax-minimax-m2.5-free"},
		{name: "复杂组合", input: "Provider/Model:Version", expected: "provider-model-version"},

		// 边界情况
		{name: "空字符串", input: "", expected: ""},
		{name: "只有特殊字符", input: "///:::", expected: ""},
		{name: "首尾特殊字符", input: "/model/", expected: "model"},
		{name: "连续特殊字符", input: "a//b::c", expected: "a-b-c"},
		{name: "含空格", input: "my model", expected: "my-model"},
		{name: "含中文", input: "模型abc", expected: "abc"},

		// 存量兼容验证
		{name: "tc-code-latest 无变化", input: "tc-code-latest", expected: "tc-code-latest"},
		{name: "qwen-max 无变化", input: "qwen-max", expected: "qwen-max"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SlugifyModelID(tt.input)
			if result != tt.expected {
				t.Errorf("SlugifyModelID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSlugifyModelID_Idempotent 验证 slug 化是幂等的（对已 slug 化的值再次调用结果不变）。
func TestSlugifyModelID_Idempotent(t *testing.T) {
	inputs := []string{
		"deepseek-v3.2",
		"minimax/minimax-m2.5:free",
		"ZhiJia/GLM-5.1-Plus",
		"openrouter/deepseek-v3",
		"tc-code-latest",
	}
	for _, input := range inputs {
		first := SlugifyModelID(input)
		second := SlugifyModelID(first)
		if first != second {
			t.Errorf("SlugifyModelID 非幂等: input=%q, first=%q, second=%q", input, first, second)
		}
	}
}

// TestAIModel_MarshalJSON_LegacyEnabledMappedToVisible 验证 AIModel.MarshalJSON
// 的核心兼容契约：对外大写 `Enabled` 字段值映射为内部 `Visible` 真实值。
//
// 旧版 React 管控台无法改动，它读 model.Enabled 渲染"用户可见"开关，
// 后端必须保证 JSON 输出的大写 Enabled = 内部 Visible，否则切换可见性
// 会出现 UI 与 DB 不一致。
func TestAIModel_MarshalJSON_LegacyEnabledMappedToVisible(t *testing.T) {
	cases := []struct {
		name       string
		enabled    bool
		visible    bool
		wantBigE   bool // 对外 "Enabled" 字段（值应等于 visible，兼容旧前端）
		wantEnStat bool // 对外 "EnabledStatus" 字段（值应等于真实 enabled，新前端读）
	}{
		{name: "enabled=T,visible=T", enabled: true, visible: true, wantBigE: true, wantEnStat: true},
		{name: "enabled=T,visible=F → 大写Enabled应为F（旧前端隐藏）", enabled: true, visible: false, wantBigE: false, wantEnStat: true},
		{name: "enabled=F,visible=T → 大写Enabled应为T（旧前端展示）", enabled: false, visible: true, wantBigE: true, wantEnStat: false},
		{name: "enabled=F,visible=F", enabled: false, visible: false, wantBigE: false, wantEnStat: false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			m := AIModel{
				Provider: "openai", ModelID: "gpt-4",
				Enabled: tt.enabled, Visible: tt.visible,
			}
			data, err := json.Marshal(m)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var parsed map[string]interface{}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			gotBigE, _ := parsed["Enabled"].(bool)
			gotEnStat, _ := parsed["EnabledStatus"].(bool)
			if gotBigE != tt.wantBigE {
				t.Errorf("Enabled = %v, want %v (raw=%s)", gotBigE, tt.wantBigE, string(data))
			}
			if gotEnStat != tt.wantEnStat {
				t.Errorf("EnabledStatus = %v, want %v (raw=%s)", gotEnStat, tt.wantEnStat, string(data))
			}
			// Visible 字段已被 MarshalJSON 显式隐藏（json:"-"），不应出现在输出中。
			if _, ok := parsed["Visible"]; ok {
				t.Errorf("Visible 字段不应被输出, raw=%s", string(data))
			}
			// 确保旧的字段名 enabled（小写）不再出现，防止回归。
			if _, ok := parsed["enabled"]; ok {
				t.Errorf("不应再输出小写 enabled 字段, raw=%s", string(data))
			}
		})
	}
}

// TestAIModel_MarshalJSON_PreservesOtherFields 确保 MarshalJSON 不影响
// 其他字段输出，避免重构 MarshalJSON 时回归。
func TestAIModel_MarshalJSON_PreservesOtherFields(t *testing.T) {
	m := AIModel{
		Provider:       "openai",
		ModelID:        "gpt-4",
		ModelName:      "GPT-4",
		URL:            "https://api.openai.com/v1",
		ModelType:      "openai-completions",
		ContextLen:     128000,
		QuotaDay:       1000,
		Enabled:        true,
		Visible:        true,
		VisibilityType: "group",
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	// 大写字段
	for _, k := range []string{"Provider", "ModelID", "ModelName", "URL", "ModelType", "ContextLen", "QuotaDay"} {
		if _, ok := parsed[k]; !ok {
			t.Errorf("缺少字段 %q, raw=%s", k, string(data))
		}
	}
	// VisibilityType 不再带 json tag，按 Go 默认大写字段名输出
	if v, _ := parsed["VisibilityType"].(string); v != "group" {
		t.Errorf("VisibilityType = %v, want group, raw=%s", parsed["VisibilityType"], string(data))
	}
	if _, ok := parsed["visibility_type"]; ok {
		t.Errorf("不应再输出小写 visibility_type 字段, raw=%s", string(data))
	}
	// APIKey 不应出现（有 json:"-"）
	if _, ok := parsed["APIKey"]; ok {
		t.Errorf("APIKey 不应被序列化, raw=%s", string(data))
	}
	// Visible 也不应出现（MarshalJSON 里用 json:"-" 显式隐藏）
	if _, ok := parsed["Visible"]; ok {
		t.Errorf("Visible 不应被序列化, raw=%s", string(data))
	}
}

// TestIsCustomModelEnabled_OnlyVisible 验证 IsCustomModelEnabled
// 仅判断 Visible 字段。hatchery/custom 是内置占位记录，
// 其 Enabled 字段对功能开放开关无影响。
func TestIsCustomModelEnabled_OnlyVisible(t *testing.T) {
	cases := []struct {
		name    string
		enabled bool
		visible bool
		want    bool
	}{
		{name: "both true → enabled", enabled: true, visible: true, want: true},
		{name: "enabled only → disabled", enabled: true, visible: false, want: false},
		{name: "visible only → enabled", enabled: false, visible: true, want: true},
		{name: "both false → disabled", enabled: false, visible: false, want: false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			gdb := setupAIModelTestDB(t)
			restore := UseDBForTest(gdb)
			defer restore()

			// 创建占位记录
			rec := AIModel{
				Provider: BuiltinModelProvider, ModelID: BuiltinModelID,
				Enabled: tt.enabled, Visible: tt.visible,
			}
			if err := gdb.Create(&rec).Error; err != nil {
				t.Fatalf("create: %v", err)
			}

			got := IsCustomModelEnabled(context.Background())
			if got != tt.want {
				t.Errorf("IsCustomModelEnabled = %v, want %v (enabled=%v visible=%v)",
					got, tt.want, tt.enabled, tt.visible)
			}
		})
	}
}

// TestIsCustomModelEnabled_RecordMissing 验证占位记录不存在时返回 false。
func TestIsCustomModelEnabled_RecordMissing(t *testing.T) {
	gdb := setupAIModelTestDB(t)
	restore := UseDBForTest(gdb)
	defer restore()

	if got := IsCustomModelEnabled(context.Background()); got {
		t.Errorf("无占位记录时应返回 false, 实际=%v", got)
	}
}

// TestAIModel_MarshalJSON_OutputKeySet 是本次 MarshalJSON 改造的"快照式契约测试"：
// 锁定对外 JSON 的字段名集合，未来任何对 AIModel 字段、json tag 或 MarshalJSON
// 的修改如果不慎增删字段，都能立刻被这个测试发现。
//
// 重点契约：
//   - 必须包含 Enabled / EnabledStatus（兼容字段 + 真实启用状态字段）；
//   - 必须不包含 Visible（已显式删除）；
//   - 必须不包含 APIKey（json:"-"）；
//   - 必须不包含小写遗留字段 enabled / visibility_type。
func TestAIModel_MarshalJSON_OutputKeySet(t *testing.T) {
	m := AIModel{
		Provider: "openai", ModelID: "gpt-4", ModelName: "GPT-4",
		URL: "https://api.openai.com/v1", ModelType: "openai-completions",
		APIKey:  "sk-secret-should-be-hidden",
		Enabled: true, Visible: true, VisibilityType: "all",
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	mustHave := []string{
		"ID", "CreatedAt", "UpdatedAt", "DeletedAt", "Identifier",
		"Provider", "ModelID", "ModelName", "URL", "ModelType",
		"InputTypes", "ContextLen", "MaxTokens", "CustomHTTPHeaders", "QuotaDay",
		"Enabled", "EnabledStatus", "VisibilityType",
	}
	for _, k := range mustHave {
		if _, ok := parsed[k]; !ok {
			t.Errorf("缺少必要字段 %q, raw=%s", k, string(data))
		}
	}

	mustNotHave := []string{
		"Visible",         // 已从 map 中删除
		"APIKey",          // json:"-"
		"enabled",         // 旧的小写字段名
		"visibility_type", // 旧的小写 json tag
		"EnabledLower",    // 历史上的奇怪名字，确保彻底清除
		"enabled_admin",   // 注释里曾承诺过的另一个名字，确保未实现
	}
	for _, k := range mustNotHave {
		if _, ok := parsed[k]; ok {
			t.Errorf("不应输出字段 %q, raw=%s", k, string(data))
		}
	}
}

// TestAIModel_MarshalJSON_EmbeddedInOtherStruct 验证当 AIModel 被嵌入其他
// 结构体时，AIModel.MarshalJSON 仍然生效（Visible 仍被剥离）。
//
// 这是 controller 层 modelWithVisibility 类型依赖的关键契约：内嵌 AIModel
// 的结构体在序列化时，会通过接口提升调用 AIModel.MarshalJSON，从而保留
// Enabled = Visible 的兼容映射并隐藏原始 Visible 字段。
func TestAIModel_MarshalJSON_EmbeddedInOtherStruct(t *testing.T) {
	type wrapper struct {
		AIModel
	}
	w := wrapper{
		AIModel: AIModel{
			Provider: "anthropic", ModelID: "claude-3",
			Enabled: false, Visible: true,
		},
	}
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	// 内嵌场景下 AIModel.MarshalJSON 通过接口提升生效：
	if bigE, _ := parsed["Enabled"].(bool); bigE != true { // = Visible
		t.Errorf("嵌入场景下 Enabled = %v, want true (=Visible), raw=%s", bigE, string(data))
	}
	if enStat, _ := parsed["EnabledStatus"].(bool); enStat != false { // = 真实 Enabled
		t.Errorf("嵌入场景下 EnabledStatus = %v, want false, raw=%s", enStat, string(data))
	}
	if _, ok := parsed["Visible"]; ok {
		t.Errorf("嵌入场景下 Visible 仍不应输出, raw=%s", string(data))
	}
}
