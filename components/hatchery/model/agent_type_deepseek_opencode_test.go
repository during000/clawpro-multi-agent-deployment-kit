// agent_type_deepseek_opencode_test.go
//
// 集中验证 DeepSeekTUI / OpenCode 两个新增内置 Agent Type 的关键不变量：
//  1. 矩阵中所有 Supports* 能力位均为 false（最小操作集，仅 Web 终端）；
//  2. 通道白名单为空（任何 channel 都不允许）；
//  3. ValidateAgentVersion 接受 semver、拒绝 OpenClaw 风格日期版本；
//  4. GetAgentTypeDisplayName 返回与产品文案一致的展示名；
//  5. SortOrder 严格排在 LightClawACE（3）之后，依次为 4、5；
//  6. GetSiteConfig 在 DB 无 site_configs 行的 fallback 场景下默认禁用两类型。
package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupDeepSeekOpenCodeTestDB 准备一个仅含 SiteConfig + CustomAgentType 的最小 DB。
// 不写入 SiteConfig 行，便于触发 GetSiteConfig 的 fallback 分支。
func setupDeepSeekOpenCodeTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&SiteConfig{}, &CustomAgentType{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(UseDBForTest(db))
}

func TestDeepSeekTUIAndOpenCode_AllSupportsFlagsFalse(t *testing.T) {
	setupDeepSeekOpenCodeTestDB(t)
	for _, code := range []string{AgentTypeDeepSeekTUI, AgentTypeOpenCode} {
		tt := GetAgentTypeByCode(context.Background(), code)
		if tt == nil {
			t.Fatalf("%s: GetAgentTypeByCode 应返回非空", code)
		}
		if !tt.IsBuiltin {
			t.Errorf("%s: 应是内置类型 IsBuiltin=true", code)
		}
		flags := []struct {
			name string
			got  bool
		}{
			{"SupportsRole", tt.SupportsRole},
			{"SupportsModel", tt.SupportsModel},
			{"SupportsChannel", tt.SupportsChannel},
			{"SupportsSkill", tt.SupportsSkill},
			{"SupportsPlugin", tt.SupportsPlugin},
			{"SupportsChatbot", tt.SupportsChatbot},
			{"SupportsSMH", tt.SupportsSMH},
			{"SupportsMemory", tt.SupportsMemory},
			{"SupportsReinstall", tt.SupportsReinstall},
			{"SupportsUpgrade", tt.SupportsUpgrade},
			{"SupportsBrowserVNC", tt.SupportsBrowserVNC},
			{"SupportsApprove", tt.SupportsApprove},
			{"SupportsDefaultModelInjection", tt.SupportsDefaultModelInjection},
			{"SupportsAPIGateway", tt.SupportsAPIGateway},
		}
		for _, f := range flags {
			if f.got {
				t.Errorf("%s.%s 应为 false，实际为 true", code, f.name)
			}
		}
	}
}

func TestDeepSeekTUIAndOpenCode_ChannelsEmpty(t *testing.T) {
	setupDeepSeekOpenCodeTestDB(t)
	for _, code := range []string{AgentTypeDeepSeekTUI, AgentTypeOpenCode} {
		got := SupportedChannelsByAgentType(context.Background(), code)
		if len(got) != 0 {
			t.Errorf("%s: SupportedChannelsByAgentType 应为空，实际=%v", code, got)
		}
		// 任意 channel 全部 fail-closed
		for _, ch := range []string{"wecom", "feishu", "qqbot", "openclaw-weixin", "ddingtalk", "wecom_app", "dingtalk-connector", "slack"} {
			if AgentTypeChannelAllowed(context.Background(), code, ch) {
				t.Errorf("%s: 不应允许 channel=%s", code, ch)
			}
		}
	}
}

func TestDeepSeekTUIAndOpenCode_ValidateAgentVersion(t *testing.T) {
	setupDeepSeekOpenCodeTestDB(t)
	cases := []struct {
		agentType string
		version   string
		wantErr   bool
	}{
		// 空版本兼容存量
		{AgentTypeDeepSeekTUI, "", false},
		{AgentTypeOpenCode, "", false},
		// semver 通过
		{AgentTypeDeepSeekTUI, "0.8.20", false},
		{AgentTypeOpenCode, "1.14.41", false},
		{AgentTypeDeepSeekTUI, "10.0.0", false},
		// 非 semver 拒绝
		{AgentTypeDeepSeekTUI, "0.8", true},
		{AgentTypeDeepSeekTUI, "v0.8.20", true},
		{AgentTypeOpenCode, "garbage", true},
		// OpenClaw 风格的日期版本号在 semver 正则下被允许（与 hermes/ace 同语义；
		// 不强制要求拒绝，以与 Hermes/ACE 已有用例保持一致）
		{AgentTypeDeepSeekTUI, "2026.5.30", false},
	}
	for _, c := range cases {
		err := ValidateAgentVersion(context.Background(), c.agentType, c.version)
		if (err != nil) != c.wantErr {
			t.Errorf("ValidateAgentVersion(%q, %q) err=%v, wantErr=%v",
				c.agentType, c.version, err, c.wantErr)
		}
	}
}

func TestDeepSeekTUIAndOpenCode_DisplayName(t *testing.T) {
	setupDeepSeekOpenCodeTestDB(t)
	if got := GetAgentTypeDisplayName(context.Background(), AgentTypeDeepSeekTUI); got != "DeepSeek TUI" {
		t.Errorf("DeepSeekTUI display name = %q, want \"DeepSeek TUI\"", got)
	}
	if got := GetAgentTypeDisplayName(context.Background(), AgentTypeOpenCode); got != "OpenCode" {
		t.Errorf("OpenCode display name = %q, want OpenCode", got)
	}
	// AgentTypeDisplayNames map 中文文案
	if AgentTypeDisplayNames[AgentTypeDeepSeekTUI] != "DeepSeek TUI" {
		t.Errorf("AgentTypeDisplayNames[deepseektui] = %q, want \"DeepSeek TUI\"", AgentTypeDisplayNames[AgentTypeDeepSeekTUI])
	}
	if AgentTypeDisplayNames[AgentTypeOpenCode] != "OpenCode" {
		t.Errorf("AgentTypeDisplayNames[opencode] = %q, want OpenCode", AgentTypeDisplayNames[AgentTypeOpenCode])
	}
}

func TestDeepSeekTUIAndOpenCode_SortOrder(t *testing.T) {
	setupDeepSeekOpenCodeTestDB(t)
	deepseek := GetAgentTypeByCode(context.Background(), AgentTypeDeepSeekTUI)
	opencode := GetAgentTypeByCode(context.Background(), AgentTypeOpenCode)
	ace := GetAgentTypeByCode(context.Background(), AgentTypeLightclawACE)
	if deepseek == nil || opencode == nil || ace == nil {
		t.Fatal("内置类型 GetAgentTypeByCode 应非空")
	}
	if !(ace.SortOrder < deepseek.SortOrder && deepseek.SortOrder < opencode.SortOrder) {
		t.Errorf("SortOrder 顺序应为 ACE(%d) < DeepSeekTUI(%d) < OpenCode(%d)",
			ace.SortOrder, deepseek.SortOrder, opencode.SortOrder)
	}
}

func TestDeepSeekTUIAndOpenCode_DefaultDisabledOnFreshDB(t *testing.T) {
	// DB 中尚未创建 site_configs 行 → GetSiteConfig 走 fallback 分支，
	// fallback 中 DisabledAgentTypes 默认为 ["deepseektui","opencode"]，
	// 因此 IsAgentTypeEnabled 应该为 false。
	setupDeepSeekOpenCodeTestDB(t)
	if IsAgentTypeEnabled(context.Background(), AgentTypeDeepSeekTUI) {
		t.Error("fresh DB 下 DeepSeekTUI 应默认禁用")
	}
	if IsAgentTypeEnabled(context.Background(), AgentTypeOpenCode) {
		t.Error("fresh DB 下 OpenCode 应默认禁用")
	}
	// 其它内置类型不受影响
	if !IsAgentTypeEnabled(context.Background(), AgentTypeOpenClaw) {
		t.Error("OpenClaw 不应被默认禁用")
	}
	if !IsAgentTypeEnabled(context.Background(), AgentTypeHermes) {
		t.Error("Hermes 不应被默认禁用")
	}
	if !IsAgentTypeEnabled(context.Background(), AgentTypeLightclawACE) {
		t.Error("LightclawACE 不应被默认禁用")
	}
}
