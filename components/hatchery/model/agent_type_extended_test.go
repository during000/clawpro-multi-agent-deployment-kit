package model

import (
	"context"
	"testing"
)

// ==================== GetPluginSupportedAgentTypes Tests ====================

func TestGetPluginSupportedAgentTypes(t *testing.T) {
	setupAgentTypeTestDB(t)
	types := GetPluginSupportedAgentTypes(context.Background())

	// 验证返回列表不为空
	if len(types) == 0 {
		t.Fatal("GetPluginSupportedAgentTypes should return at least one type")
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

	// 验证包含 openclaw（支持插件）
	hasOpenclaw := false
	for _, code := range types {
		if code == "openclaw" {
			hasOpenclaw = true
			break
		}
	}
	if !hasOpenclaw {
		t.Error("should contain 'openclaw' which supports plugin")
	}

	// 验证不包含 hermes 和 lightclawace（不支持插件）
	for _, code := range types {
		if code == "hermes" {
			t.Error("should not contain 'hermes' which does not support plugin")
		}
		if code == "lightclawace" {
			t.Error("should not contain 'lightclawace' which does not support plugin")
		}
	}

	// 验证返回的类型数量正确（空字符串 + 支持插件的类型）
	expectedCount := 2 // "" + "openclaw"
	if len(types) != expectedCount {
		t.Errorf("expected %d types, got %d: %v", expectedCount, len(types), types)
	}
}

func TestGetPluginSupportedAgentTypesConsistency(t *testing.T) {
	setupAgentTypeTestDB(t)
	// 验证 GetPluginSupportedAgentTypes 返回的类型与 AgentTypeSupportsPlugin 一致
	supportedTypes := GetPluginSupportedAgentTypes(context.Background())

	// 所有返回的非空类型都应该通过 AgentTypeSupportsPlugin 检查
	for _, code := range supportedTypes {
		if !AgentTypeSupportsPlugin(context.Background(), code) {
			t.Errorf("GetPluginSupportedAgentTypes returned '%s' but AgentTypeSupportsPlugin(context.Background(), '%s') is false", code, code)
		}
	}

	// 所有支持插件的类型都应该在返回列表中
	allTypes := GetAllAgentTypes(context.Background())
	for _, agentType := range allTypes {
		if agentType.SupportsPlugin {
			found := false
			for _, code := range supportedTypes {
				if code == agentType.Code {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("type '%s' supports plugin but not in GetPluginSupportedAgentTypes result", agentType.Code)
			}
		}
	}
}

// ==================== AgentTypeSupportsChannel Tests ====================

func TestAgentTypeSupportsChannelAllTypes(t *testing.T) {
	setupAgentTypeTestDB(t)
	// v7：hermes/lightclawace 放开后支持 channel
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

// ==================== AgentTypeSupportsRole Tests (extended) ====================

func TestAgentTypeSupportsRoleAllTypes(t *testing.T) {
	setupAgentTypeTestDB(t)
	// v7：hermes/lightclawace 放开后支持 role（Soul 靠 LLM 代理注入）
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
			result := AgentTypeSupportsRole(context.Background(), tt.code)
			if result != tt.expected {
				t.Errorf("AgentTypeSupportsRole(context.Background(), %q) = %v, want %v", tt.code, result, tt.expected)
			}
		})
	}
}

// ==================== AgentTypeSupportsSMH Tests (extended) ====================

func TestAgentTypeSupportsSMHAllTypes(t *testing.T) {
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

// ==================== AgentTypeSupportsMemory Tests (extended) ====================

func TestAgentTypeSupportsMemoryAllTypes(t *testing.T) {
	setupAgentTypeTestDB(t)
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

// ==================== GetAgentTypeDisplayName Tests (extended) ====================

func TestGetAgentTypeDisplayNameAllTypes(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		code     string
		expected string
	}{
		{"openclaw", "OpenClaw"},
		{"hermes", "Hermes"},
		{"lightclawace", "LightclawACE"},
		{"", "OpenClaw"},       // 空字符串视为 openclaw
		{"unknown", "unknown"}, // 不存在的类型返回原始 code
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := GetAgentTypeDisplayName(context.Background(), tt.code)
			if result != tt.expected {
				t.Errorf("GetAgentTypeDisplayName(context.Background(), %q) = %q, want %q", tt.code, result, tt.expected)
			}
		})
	}
}

// ==================== ValidateAgentVersion Tests (extended) ====================

func TestValidateAgentVersionEdgeCases(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		name        string
		agentType   string
		version     string
		expectError bool
	}{
		// OpenClaw 边界
		{"openclaw double digit month", "openclaw", "2026.12.1", false},
		{"openclaw double digit day", "openclaw", "2026.1.31", false},
		{"openclaw double digit both", "openclaw", "2026.12.31", false},
		{"openclaw no leading zero", "openclaw", "2026.3.8", false},
		{"openclaw text version", "openclaw", "v2.0", true},
		{"openclaw semver format", "openclaw", "1.2.3", true},

		// Hermes 边界
		{"hermes major only", "hermes", "1", true},
		{"hermes two parts", "hermes", "1.0", true},
		{"hermes valid with high numbers", "hermes", "100.200.300", false},
		{"hermes with prefix", "hermes", "v1.0.0", true},

		// LightclawACE 边界
		{"lightclawace zero version", "lightclawace", "0.0.0", false},
		{"lightclawace with extra parts", "lightclawace", "1.0.0.1", true},
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

// ==================== SortOrder 验证 ====================

func TestAgentTypesSortOrder(t *testing.T) {
	setupAgentTypeTestDB(t)
	types := GetAllAgentTypes(context.Background())
	for i := 1; i < len(types); i++ {
		if types[i].SortOrder <= types[i-1].SortOrder {
			t.Errorf("agent types not properly sorted: [%d].SortOrder=%d <= [%d].SortOrder=%d",
				i, types[i].SortOrder, i-1, types[i-1].SortOrder)
		}
	}
}

// ==================== CanEnableImage Edge Cases ====================

func TestCanEnableImageEdgeCases(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		name      string
		img       AIImage
		canEnable bool
	}{
		{
			name:      "empty version with empty type (fully legacy)",
			img:       AIImage{AgentType: "", AgentVersion: ""},
			canEnable: true,
		},
		{
			name:      "empty version with valid type (incomplete)",
			img:       AIImage{AgentType: "openclaw", AgentVersion: ""},
			canEnable: false,
		},
		{
			name:      "lightclawace with valid version",
			img:       AIImage{AgentType: "lightclawace", AgentVersion: "0.1.1"},
			canEnable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockErr := tt.img.CanEnableImage(context.Background())
			canEnable := blockErr == nil
			if canEnable != tt.canEnable {
				t.Errorf("CanEnableImage() = %v, want %v", canEnable, tt.canEnable)
			}
		})
	}
}

// ==================== final：通道白名单 + Reinstall 能力 ====================

func TestAgentTypeChannelAllowed(t *testing.T) {
	setupAgentTypeTestDB(t)
	// final §3.3：白名单显式化，channel_id 取值对齐 ai_channel.go::predefinedChannels
	// （openclaw-weixin / wecom / wecom_app / feishu / ddingtalk / qqbot / slack）
	tests := []struct {
		name      string
		agentType string
		channel   string
		expected  bool
	}{
		// openclaw 显式白名单（移除 nil 全放行语义）
		{"openclaw-openclaw-weixin", "openclaw", "openclaw-weixin", true},
		{"openclaw-wecom", "openclaw", "wecom", true},
		{"openclaw-wecom_app", "openclaw", "wecom_app", true},
		{"openclaw-feishu", "openclaw", "feishu", true},
		{"openclaw-ddingtalk", "openclaw", "ddingtalk", true},
		{"openclaw-qqbot", "openclaw", "qqbot", true},
		{"openclaw-slack", "openclaw", "slack", true},
		{"openclaw-unknown-channel", "openclaw", "anything", false}, // 未知 channel 现在被拒
		{"openclaw-empty-channel", "openclaw", "", false},
		// 空 agentType 视为 openclaw
		{"empty-openclaw-weixin", "", "openclaw-weixin", true},

		// hermes 白名单
		{"hermes-openclaw-weixin", "hermes", "openclaw-weixin", true},
		{"hermes-wecom", "hermes", "wecom", true},
		{"hermes-feishu", "hermes", "feishu", true},
		{"hermes-ddingtalk", "hermes", "ddingtalk", true},
		{"hermes-qqbot", "hermes", "qqbot", true},
		{"hermes-slack", "hermes", "slack", true},
		{"hermes-wecom_app", "hermes", "wecom_app", false},

		// lightclawace 白名单（无 ddingtalk / wecom_app）
		{"ace-openclaw-weixin", "lightclawace", "openclaw-weixin", true},
		{"ace-wecom", "lightclawace", "wecom", true},
		{"ace-feishu", "lightclawace", "feishu", true},
		{"ace-qqbot", "lightclawace", "qqbot", true},
		{"ace-ddingtalk", "lightclawace", "ddingtalk", false}, // final §2：ACE 不支持钉钉
		{"ace-wecom_app", "lightclawace", "wecom_app", false},
		{"ace-slack", "lightclawace", "slack", false},

		// 未知 agent_type fail-closed
		{"unknown", "unknown", "feishu", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AgentTypeChannelAllowed(context.Background(), tt.agentType, tt.channel)
			if got != tt.expected {
				t.Errorf("AgentTypeChannelAllowed(context.Background(), %q, %q) = %v, want %v",
					tt.agentType, tt.channel, got, tt.expected)
			}
		})
	}
}

func TestSupportedChannelsByAgentType(t *testing.T) {
	setupAgentTypeTestDB(t)
	// final §3.3：所有 agentType 均走显式白名单，不再有 nil 全放行语义
	// openclaw 白名单：openclaw-weixin/wecom/wecom_app/feishu/ddingtalk/msteams/qqbot/dingtalk-connector/slack/discord = 10 个
	openclawCh := SupportedChannelsByAgentType(context.Background(), "openclaw")
	if len(openclawCh) != 12 {
		t.Errorf("openclaw should have 12 channels, got %d: %v", len(openclawCh), openclawCh)
	}
	// 空字符串视为 openclaw
	emptyCh := SupportedChannelsByAgentType(context.Background(), "")
	if len(emptyCh) != 12 {
		t.Errorf("empty should be treated as openclaw (12 channels), got %d: %v", len(emptyCh), emptyCh)
	}
	// unknown 返回 nil
	if got := SupportedChannelsByAgentType(context.Background(), "unknown"); got != nil {
		t.Errorf("unknown should return nil, got %v", got)
	}
	// hermes 返回 10 个（wecom/openclaw-weixin/feishu/ddingtalk/msteams/qqbot/slack/discord/lark/line）
	hermes := SupportedChannelsByAgentType(context.Background(), "hermes")
	if len(hermes) != 10 {
		t.Errorf("hermes should have 10 channels, got %d: %v", len(hermes), hermes)
	}
	// lightclawace 返回 4 个（openclaw-weixin/wecom/feishu/qqbot）
	ace := SupportedChannelsByAgentType(context.Background(), "lightclawace")
	if len(ace) != 4 {
		t.Errorf("lightclawace should have 4 channels, got %d: %v", len(ace), ace)
	}
}

// 三期放开：Hermes / LightclawACE 现均支持重装。
func TestAgentTypeSupportsReinstall(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		code     string
		expected bool
	}{
		{"openclaw", true},
		{"hermes", true},       // 三期放开
		{"lightclawace", true}, // 三期放开
		{"", true},             // 空视为 openclaw
		{"unknown", false},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := AgentTypeSupportsReinstall(context.Background(), tt.code)
			if got != tt.expected {
				t.Errorf("AgentTypeSupportsReinstall(context.Background(), %q) = %v, want %v", tt.code, got, tt.expected)
			}
		})
	}
}
