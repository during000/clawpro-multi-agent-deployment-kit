package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupAgentTypeTestDB 为纯能力函数（GetAgentTypeByCode / GetAgentRuntimeType /
// AgentTypeSupports* 等）提供必要的内存 DB 和 CustomAgentType 表。
// 这些函数会查询自定义 Agent 类型表，因此即使测试只校验内置类型行为，也需要先建表。
func setupAgentTypeTestDB(t *testing.T) {
	t.Helper()

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := testDB.AutoMigrate(&CustomAgentType{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	t.Cleanup(UseDBForTest(testDB))
}

func TestIsValidAgentType(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		name      string
		agentType string
		expected  bool
	}{
		{"valid openclaw", "openclaw", true},
		{"valid hermes", "hermes", true},
		{"valid lightclawace", "lightclawace", true},
		{"invalid type", "unknown", false},
		{"empty string", "", false},
		{"sql injection attempt", "openclaw'; DROP TABLE--", false},
		{"case sensitive", "OpenClaw", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidAgentType(context.Background(), tt.agentType)
			if result != tt.expected {
				t.Errorf("IsValidAgentType(context.Background(), %q) = %v, want %v", tt.agentType, result, tt.expected)
			}
		})
	}
}

func TestIsValidAgentVersion(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{"valid semver", "1.0.0", true},
		{"valid date version", "2026.4.2", true},
		{"valid with suffix", "1.0.0-beta", true},
		{"empty allowed", "", true},
		{"single char", "1", true},
		{"too long", string(make([]byte, 100)), false},
		{"invalid start", "-1.0.0", false},
		{"script injection", "1.0<script>", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidAgentVersion(tt.version)
			if result != tt.expected {
				t.Errorf("IsValidAgentVersion(%q) = %v, want %v", tt.version, result, tt.expected)
			}
		})
	}
}

func TestAgentTypeSupportsRole(t *testing.T) {
	setupAgentTypeTestDB(t)
	// v7：矩阵放开后 hermes/lightclawace 均支持 role（Soul 靠 LLM 代理注入）
	tests := []struct {
		code     string
		expected bool
	}{
		{"openclaw", true},
		{"hermes", true},
		{"lightclawace", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := AgentTypeSupportsRole(context.Background(), tt.code)
			if result != tt.expected {
				t.Errorf("AgentTypeSupportsRole(context.Background(), %s) = %v, want %v", tt.code, result, tt.expected)
			}
		})
	}
}

func TestAgentTypeSupportsDetailConfig(t *testing.T) {
	setupAgentTypeTestDB(t)
	// v7：hermes/lightclawace 支持 Model/Channel/Skill，DetailConfig 返回 true
	if !AgentTypeSupportsDetailConfig(context.Background(), "openclaw") {
		t.Error("OpenClaw should support detail config")
	}
	if !AgentTypeSupportsDetailConfig(context.Background(), "hermes") {
		t.Error("v7: Hermes should now support detail config (Model/Channel/Skill enabled)")
	}
	if !AgentTypeSupportsDetailConfig(context.Background(), "lightclawace") {
		t.Error("v7: LightclawACE should now support detail config (Model/Channel/Skill enabled)")
	}
	if AgentTypeSupportsDetailConfig(context.Background(), "unknown") {
		t.Error("unknown should not support detail config")
	}
}

func TestGetAgentTypeDetailConfigFlags(t *testing.T) {
	setupAgentTypeTestDB(t)
	// 测试 OpenClaw - 支持所有配置
	flags := GetAgentTypeDetailConfigFlags(context.Background(), "openclaw")
	if flags == nil {
		t.Fatal("flags should not be nil for openclaw")
	}
	if !flags.SupportsModel || !flags.SupportsChannel || !flags.SupportsSkill || !flags.SupportsPlugin {
		t.Error("OpenClaw should support all detail configs")
	}

	// v7：Hermes 支持 Model/Channel/Skill，不支持 Plugin
	flags = GetAgentTypeDetailConfigFlags(context.Background(), "hermes")
	if flags == nil {
		t.Fatal("flags should not be nil for hermes")
	}
	if !flags.SupportsModel || !flags.SupportsChannel || !flags.SupportsSkill {
		t.Error("v7: Hermes should support Model/Channel/Skill")
	}
	if flags.SupportsPlugin {
		t.Error("Hermes should NOT support Plugin")
	}

	// 测试不存在的类型
	flags = GetAgentTypeDetailConfigFlags(context.Background(), "nonexistent")
	if flags != nil {
		t.Error("expected nil for nonexistent agent type")
	}
}

func TestAgentTypeSupportsChatbot(t *testing.T) {
	setupAgentTypeTestDB(t)
	result := AgentTypeSupportsChatbot(context.Background(), "openclaw")
	if !result {
		t.Error("OpenClaw should support chatbot")
	}

	result = AgentTypeSupportsChatbot(context.Background(), "hermes")
	if result {
		t.Error("Hermes should not support chatbot")
	}
}

func TestGetAllAgentTypes(t *testing.T) {
	setupAgentTypeTestDB(t)
	types := GetAllAgentTypes(context.Background())
	if len(types) != 5 {
		t.Errorf("expected 5 agent types, got %d", len(types))
	}

	// 验证排序
	if types[0].Code != "openclaw" {
		t.Error("first type should be openclaw")
	}
	if types[1].Code != "hermes" {
		t.Error("second type should be hermes")
	}
	if types[2].Code != "lightclawace" {
		t.Error("third type should be lightclawace")
	}
	if types[3].Code != "deepseektui" {
		t.Error("fourth type should be deepseektui")
	}
	if types[4].Code != "opencode" {
		t.Error("fifth type should be opencode")
	}
}

func TestGetAgentTypeByCode(t *testing.T) {
	setupAgentTypeTestDB(t)
	// 测试存在的类型
	t1 := GetAgentTypeByCode(context.Background(), "openclaw")
	if t1 == nil || t1.Name != "OpenClaw" {
		t.Error("should return OpenClaw type")
	}

	// 测试不存在的类型
	t2 := GetAgentTypeByCode(context.Background(), "nonexistent")
	if t2 != nil {
		t.Error("should return nil for nonexistent type")
	}

	// 测试空字符串应视为 openclaw（兼容存量数据）
	t3 := GetAgentTypeByCode(context.Background(), "")
	if t3 == nil || t3.Code != AgentTypeOpenClaw {
		t.Error("empty string should be treated as openclaw")
	}
}

func TestGetAgentTypeDisplayName(t *testing.T) {
	setupAgentTypeTestDB(t)
	name := GetAgentTypeDisplayName(context.Background(), "openclaw")
	if name != "OpenClaw" {
		t.Errorf("expected 'OpenClaw', got '%s'", name)
	}

	name = GetAgentTypeDisplayName(context.Background(), "hermes")
	if name != "Hermes" {
		t.Errorf("expected 'Hermes', got '%s'", name)
	}

	// 不存在的类型返回原始 code
	name = GetAgentTypeDisplayName(context.Background(), "unknown")
	if name != "unknown" {
		t.Errorf("expected 'unknown', got '%s'", name)
	}
}

func TestAgentTypeSupportsSkill(t *testing.T) {
	setupAgentTypeTestDB(t)
	if !AgentTypeSupportsSkill(context.Background(), "openclaw") {
		t.Error("OpenClaw should support skill")
	}
	// v7：hermes/lightclawace 放开后支持 skill
	if !AgentTypeSupportsSkill(context.Background(), "hermes") {
		t.Error("v7: Hermes should support skill")
	}
	if !AgentTypeSupportsSkill(context.Background(), "lightclawace") {
		t.Error("v7: LightclawACE should support skill")
	}
	// 空字符串应视为 openclaw，支持技能
	if !AgentTypeSupportsSkill(context.Background(), "") {
		t.Error("Empty string (legacy openclaw) should support skill")
	}
}

func TestAgentTypeSupportsPlugin(t *testing.T) {
	setupAgentTypeTestDB(t)
	if !AgentTypeSupportsPlugin(context.Background(), "openclaw") {
		t.Error("OpenClaw should support plugin")
	}
	// Plugin 保持 false（Hermes/ACE 无 plugin 体系）
	if AgentTypeSupportsPlugin(context.Background(), "hermes") {
		t.Error("Hermes should not support plugin")
	}
	if AgentTypeSupportsPlugin(context.Background(), "lightclawace") {
		t.Error("LightclawACE should not support plugin")
	}
	if AgentTypeSupportsPlugin(context.Background(), "unknown") {
		t.Error("Unknown type should not support plugin")
	}
	// 空字符串应视为 openclaw，支持插件
	if !AgentTypeSupportsPlugin(context.Background(), "") {
		t.Error("Empty string (legacy openclaw) should support plugin")
	}
}

func TestAgentTypeSupportsModel(t *testing.T) {
	setupAgentTypeTestDB(t)
	if !AgentTypeSupportsModel(context.Background(), "openclaw") {
		t.Error("OpenClaw should support model")
	}
	// v7：hermes/lightclawace 放开后支持 model
	if !AgentTypeSupportsModel(context.Background(), "hermes") {
		t.Error("v7: Hermes should support model")
	}
	if !AgentTypeSupportsModel(context.Background(), "lightclawace") {
		t.Error("v7: LightclawACE should support model")
	}
	if AgentTypeSupportsModel(context.Background(), "unknown") {
		t.Error("Unknown type should not support model")
	}
	// 空字符串应视为 openclaw，支持模型
	if !AgentTypeSupportsModel(context.Background(), "") {
		t.Error("Empty string (legacy openclaw) should support model")
	}
}

func TestValidateAgentVersion(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		name        string
		agentType   string
		version     string
		expectError bool
	}{
		// OpenClaw 格式 YYYY.M.D
		{"openclaw valid", "openclaw", "2026.3.28", false},
		{"openclaw valid short", "openclaw", "2026.1.1", false},
		{"openclaw invalid semver", "openclaw", "1.0.0", true},

		// Hermes 格式 X.Y.Z
		{"hermes valid", "hermes", "0.9.0", false},
		{"hermes valid release", "hermes", "1.0.0", false},
		{"hermes date format", "hermes", "2026.3.28", false}, // semver regex allows any X.Y.Z

		// LightclawACE 格式 X.Y.Z
		{"lightclawace valid", "lightclawace", "0.1.1", false},
		{"lightclawace date format", "lightclawace", "2026.3.28", false}, // semver regex allows any X.Y.Z
		{"lightclawace invalid", "lightclawace", "not-a-version", true},

		// 空版本允许
		{"empty version openclaw", "openclaw", "", false},
		{"empty version hermes", "hermes", "", false},

		// 未知类型 - 走 default 分支
		{"unknown type valid version", "unknown", "1.0.0", false},
		{"unknown type invalid version", "unknown", "-invalid-", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentVersion(context.Background(), tt.agentType, tt.version)
			hasError := err != nil
			if hasError != tt.expectError {
				t.Errorf("ValidateAgentVersion(context.Background(), %s, %s) error=%v, expectError=%v",
					tt.agentType, tt.version, err, tt.expectError)
			}
		})
	}
}

func TestCanEnableImage(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		name      string
		img       AIImage
		canEnable bool
		reason    string
	}{
		{
			name:      "normal image can enable",
			img:       AIImage{AgentType: "openclaw", AgentVersion: "2026.1.1"},
			canEnable: true,
		},
		{
			name:      "no version non-legacy cannot enable",
			img:       AIImage{AgentType: "hermes", AgentVersion: ""},
			canEnable: false,
			reason:    "请先设置 Agent 版本后再启用",
		},
		{
			name:      "legacy image (no type) can enable",
			img:       AIImage{AgentType: "", AgentVersion: ""},
			canEnable: true,
		},
		{
			name:      "invalid type cannot enable",
			img:       AIImage{AgentType: "invalid_type", AgentVersion: "1.0.0"},
			canEnable: false,
			reason:    "无效的智能体类型: invalid_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockErr := tt.img.CanEnableImage(context.Background())
			canEnable := blockErr == nil
			reason := ""
			if blockErr != nil {
				reason = blockErr.Error()
			}
			if canEnable != tt.canEnable {
				t.Errorf("CanEnableImage() = %v, want %v", canEnable, tt.canEnable)
			}
			if !tt.canEnable && reason != tt.reason {
				t.Errorf("reason = %q, want %q", reason, tt.reason)
			}
		})
	}
}

func TestIsLegacyImage(t *testing.T) {
	setupAgentTypeTestDB(t)
	// Legacy: no type
	img1 := AIImage{AgentType: "", AgentVersion: ""}
	if !img1.IsLegacyImage(context.Background()) {
		t.Error("image with empty type should be legacy")
	}

	// Legacy: type but no version
	img2 := AIImage{AgentType: "openclaw", AgentVersion: ""}
	if !img2.IsLegacyImage(context.Background()) {
		t.Error("image with type but no version should be legacy")
	}

	// Not legacy
	img3 := AIImage{AgentType: "openclaw", AgentVersion: "2026.1.1"}
	if img3.IsLegacyImage(context.Background()) {
		t.Error("image with type and version should not be legacy")
	}
}

func TestGetAllAgentTypesMap(t *testing.T) {
	setupAgentTypeTestDB(t)
	typesMap := GetAllAgentTypesMap(context.Background())

	// 验证返回的 map 包含所有类型
	if len(typesMap) != 5 {
		t.Errorf("expected 5 agent types in map, got %d", len(typesMap))
	}

	// 验证每个类型都存在
	expectedTypes := []string{"openclaw", "hermes", "lightclawace", "deepseektui", "opencode"}
	for _, code := range expectedTypes {
		if _, exists := typesMap[code]; !exists {
			t.Errorf("expected type %s to exist in map", code)
		}
	}

	// 验证 OpenClaw 配置正确
	openclaw := typesMap["openclaw"]
	if openclaw == nil {
		t.Fatal("openclaw should exist")
	}
	if openclaw.Name != "OpenClaw" {
		t.Errorf("openclaw name should be 'OpenClaw', got '%s'", openclaw.Name)
	}
	if !openclaw.SupportsSkill {
		t.Error("openclaw should support skill")
	}
}

func TestGetSkillSupportedAgentTypes(t *testing.T) {
	setupAgentTypeTestDB(t)
	types := GetSkillSupportedAgentTypes(context.Background())

	// 验证返回列表不为空
	if len(types) == 0 {
		t.Fatal("GetSkillSupportedAgentTypes should return at least one type")
	}

	// 验证包含空字符串（兼容存量数据）
	hasEmpty := false
	for _, code := range types {
		if code == "" {
			hasEmpty = true
			break
		}
	}
	if !hasEmpty {
		t.Error("should contain empty string for legacy data compatibility")
	}

	// v7：hermes/lightclawace/openclaw 均支持技能
	for _, want := range []string{"openclaw", "hermes", "lightclawace"} {
		found := false
		for _, code := range types {
			if code == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("v7: should contain '%s' which supports skill", want)
		}
	}

	// v7：空字符串 + 3 种类型 = 4
	expectedCount := 4
	if len(types) != expectedCount {
		t.Errorf("expected %d types, got %d: %v", expectedCount, len(types), types)
	}
}

func TestGetSkillSupportedAgentTypesConsistency(t *testing.T) {
	setupAgentTypeTestDB(t)
	// 验证 GetSkillSupportedAgentTypes 返回的类型与 AgentTypeSupportsSkill 一致
	supportedTypes := GetSkillSupportedAgentTypes(context.Background())

	// 所有返回的类型都应该通过 AgentTypeSupportsSkill 检查
	for _, code := range supportedTypes {
		if !AgentTypeSupportsSkill(context.Background(), code) {
			t.Errorf("GetSkillSupportedAgentTypes returned '%s' but AgentTypeSupportsSkill(context.Background(), '%s') is false", code, code)
		}
	}

	// 所有支持技能的类型都应该在返回列表中
	allTypes := GetAllAgentTypes(context.Background())
	for _, agentType := range allTypes {
		if agentType.SupportsSkill {
			found := false
			for _, code := range supportedTypes {
				if code == agentType.Code {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("type '%s' supports skill but not in GetSkillSupportedAgentTypes result", agentType.Code)
			}
		}
	}
}

func TestAgentTypeSupportsSkillAllTypes(t *testing.T) {
	setupAgentTypeTestDB(t)
	// v7：矩阵放开，openclaw/hermes/lightclawace 均支持
	tests := []struct {
		code     string
		expected bool
	}{
		{"openclaw", true},
		{"hermes", true},
		{"lightclawace", true},
		{"", true},         // 空字符串视为 openclaw
		{"unknown", false}, // 未知类型不支持
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := AgentTypeSupportsSkill(context.Background(), tt.code)
			if result != tt.expected {
				t.Errorf("AgentTypeSupportsSkill(context.Background(), %q) = %v, want %v", tt.code, result, tt.expected)
			}
		})
	}
}

// TestAgentTypeSupportsSkillByMap 覆盖 AgentTypeSupportsSkillByMap 所有核心路径：
//   - 已知支持技能的类型（openclaw/hermes/lightclawace）→ true
//   - 空字符串经 NormalizeAgentType 回退到 openclaw → true
//   - map 中存在但不支持技能的类型 → false
//   - 不在 map 中的未知类型 → false
//   - 空 map / nil map → false（安全降级）
//   - 与 AgentTypeSupportsSkill(ctx) 对预加载全量 map 的结果一致（回归防护）
func TestAgentTypeSupportsSkillByMap(t *testing.T) {
	// 构造一个与内置 agentTypesMap 一致的测试 map（纯内存，无需 DB）
	skillTypes := map[string]*AgentType{
		"openclaw":     {Code: "openclaw", SupportsSkill: true},
		"hermes":       {Code: "hermes", SupportsSkill: true},
		"lightclawace": {Code: "lightclawace", SupportsSkill: true},
		"noskill":      {Code: "noskill", SupportsSkill: false}, // 不支持技能的虚构类型
	}

	tests := []struct {
		name     string
		code     string
		allTypes map[string]*AgentType
		want     bool
	}{
		{"openclaw 支持", "openclaw", skillTypes, true},
		{"hermes 支持", "hermes", skillTypes, true},
		{"lightclawace 支持", "lightclawace", skillTypes, true},
		{"空字符串回退 openclaw", "", skillTypes, true},
		{"不支持技能的类型", "noskill", skillTypes, false},
		{"未知类型不在 map 中", "unknown_type", skillTypes, false},
		{"空 map 安全降级", "openclaw", map[string]*AgentType{}, false},
		{"nil map 安全降级", "openclaw", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AgentTypeSupportsSkillByMap(tt.code, tt.allTypes)
			if got != tt.want {
				t.Errorf("AgentTypeSupportsSkillByMap(%q, map) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

// TestAgentTypeSupportsSkillByMap_ConsistentWithDB 验证 ByMap 变体对全量预加载 map
// 的结果与 AgentTypeSupportsSkill(ctx) 完全一致，确保 N+1 修复未改变语义。
func TestAgentTypeSupportsSkillByMap_ConsistentWithDB(t *testing.T) {
	setupAgentTypeTestDB(t)
	allTypes := GetAllAgentTypesMap(context.Background())

	codes := []string{"openclaw", "hermes", "lightclawace", "", "nonexistent"}
	for _, code := range codes {
		want := AgentTypeSupportsSkill(context.Background(), code)
		got := AgentTypeSupportsSkillByMap(code, allTypes)
		if got != want {
			t.Errorf("code=%q: ByMap=%v vs DB=%v (不一致)", code, got, want)
		}
	}
}

func TestAgentTypeSupportsChatbotAllTypes(t *testing.T) {
	setupAgentTypeTestDB(t)
	// final §3.2：ACE Chatbot 放开（注：Chatbot handler 未实现前用户无入口触发）
	tests := []struct {
		code     string
		expected bool
	}{
		{"openclaw", true},
		{"hermes", false},
		{"lightclawace", true},
		{"", true}, // 空字符串视为 openclaw
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := AgentTypeSupportsChatbot(context.Background(), tt.code)
			if result != tt.expected {
				t.Errorf("AgentTypeSupportsChatbot(context.Background(), %q) = %v, want %v", tt.code, result, tt.expected)
			}
		})
	}
}

func TestAgentTypeSupportsDetailConfigAllTypes(t *testing.T) {
	setupAgentTypeTestDB(t)
	// v7：hermes/lightclawace 放开后 DetailConfig=true
	tests := []struct {
		code     string
		expected bool
	}{
		{"openclaw", true},
		{"hermes", true},
		{"lightclawace", true},
		{"", true}, // 空字符串视为 openclaw
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := AgentTypeSupportsDetailConfig(context.Background(), tt.code)
			if result != tt.expected {
				t.Errorf("AgentTypeSupportsDetailConfig(context.Background(), %q) = %v, want %v", tt.code, result, tt.expected)
			}
		})
	}
}

func TestAgentTypeSupportsSMH(t *testing.T) {
	setupAgentTypeTestDB(t)
	// final §2 / §3.2：三端 SMH 均放开
	tests := []struct {
		code     string
		expected bool
	}{
		{"openclaw", true},
		{"hermes", true},
		{"lightclawace", true},
		{"", true}, // 空字符串视为 openclaw
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := AgentTypeSupportsSMH(context.Background(), tt.code)
			if result != tt.expected {
				t.Errorf("AgentTypeSupportsSMH(context.Background(), %q) = %v, want %v", tt.code, result, tt.expected)
			}
		})
	}
}

// Note: TestGetPluginSupportedAgentTypes 和 TestGetPluginSupportedAgentTypesConsistency
// 已迁移至 agent_type_extended_test.go，避免 duplicate declaration（v7 订正）。

func TestAgentTypeSupportsChannel(t *testing.T) {
	setupAgentTypeTestDB(t)
	// v7：hermes/lightclawace 放开后支持 channel（具体 channel 由白名单控制）
	tests := []struct {
		code     string
		expected bool
	}{
		{"openclaw", true},
		{"hermes", true},
		{"lightclawace", true},
		{"", true}, // 空字符串视为 openclaw
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := AgentTypeSupportsChannel(context.Background(), tt.code)
			if result != tt.expected {
				t.Errorf("AgentTypeSupportsChannel(context.Background(), %q) = %v, want %v", tt.code, result, tt.expected)
			}
		})
	}
}

func TestAgentTypeSupportsMemory(t *testing.T) {
	setupAgentTypeTestDB(t)
	// Memory：openclaw 和 hermes 均支持
	tests := []struct {
		code     string
		expected bool
	}{
		{"openclaw", true},
		{"hermes", true},
		{"lightclawace", false},
		{"", true}, // 空字符串视为 openclaw
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := AgentTypeSupportsMemory(context.Background(), tt.code)
			if result != tt.expected {
				t.Errorf("AgentTypeSupportsMemory(context.Background(), %q) = %v, want %v", tt.code, result, tt.expected)
			}
		})
	}
}

func TestGetMemorySupportedAgentTypes(t *testing.T) {
	setupAgentTypeTestDB(t)
	types := GetMemorySupportedAgentTypes(context.Background())

	if len(types) == 0 {
		t.Fatal("GetMemorySupportedAgentTypes should return at least one type")
	}

	// 必须包含空字符串（兼容存量）
	hasEmpty := false
	for _, code := range types {
		if code == "" {
			hasEmpty = true
			break
		}
	}
	if !hasEmpty {
		t.Error("should contain empty string for legacy data compatibility")
	}

	// 必须包含 openclaw
	hasOpenclaw := false
	for _, code := range types {
		if code == "openclaw" {
			hasOpenclaw = true
			break
		}
	}
	if !hasOpenclaw {
		t.Error("should contain 'openclaw' which supports memory")
	}

	// 不得包含 lightclawace
	for _, code := range types {
		if code == "lightclawace" {
			t.Errorf("should not contain %q which does not support memory", code)
		}
	}

	// 必须包含 hermes（已适配记忆功能）
	hasHermes := false
	for _, code := range types {
		if code == "hermes" {
			hasHermes = true
			break
		}
	}
	if !hasHermes {
		t.Error("should contain 'hermes' which now supports memory")
	}
}

func TestGetSMHSupportedAgentTypes(t *testing.T) {
	setupAgentTypeTestDB(t)
	types := GetSMHSupportedAgentTypes(context.Background())

	if len(types) == 0 {
		t.Fatal("GetSMHSupportedAgentTypes should return at least one type")
	}

	// 必须包含空字符串（兼容存量）
	hasEmpty := false
	for _, code := range types {
		if code == "" {
			hasEmpty = true
			break
		}
	}
	if !hasEmpty {
		t.Error("should contain empty string for legacy data compatibility")
	}

	// 三端都支持 SMH
	expected := []string{"openclaw", "hermes", "lightclawace"}
	for _, want := range expected {
		found := false
		for _, code := range types {
			if code == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("should contain %q which supports SMH", want)
		}
	}
}
