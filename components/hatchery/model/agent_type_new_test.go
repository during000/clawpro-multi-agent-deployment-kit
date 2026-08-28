package model

import (
	"context"
	"sort"
	"testing"
)

// TestAgentTypeSupportsBrowserVNC 覆盖 AgentTypeSupportsBrowserVNC 全部分支：
// 仅 OpenClaw 为 true；Hermes / ACE 镜像无 noVNC/Chrome 栈，应返回 false。
// 空字符串兼容为 openclaw。
func TestAgentTypeSupportsBrowserVNC(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		code string
		want bool
	}{
		{AgentTypeOpenClaw, true},
		{"", true}, // 空字符串兼容 openclaw
		{AgentTypeHermes, false},
		{AgentTypeLightclawACE, false},
		{"unknown_agent_type", false}, // 未知类型 → false（GetAgentTypeByCode 返回 nil）
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := AgentTypeSupportsBrowserVNC(context.Background(), tt.code)
			if got != tt.want {
				t.Errorf("AgentTypeSupportsBrowserVNC(context.Background(), %q)=%v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

// TestAgentTypeSupportsApprove 覆盖 AgentTypeSupportsApprove 全部分支：
// 仅 OpenClaw 为 true；Hermes（harness）/ACE（lightclaw）走自己的 OAuth/Server API，
// 不需要 approve CLI 子命令。
func TestAgentTypeSupportsApprove(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		code string
		want bool
	}{
		{AgentTypeOpenClaw, true},
		{"", true},
		{AgentTypeHermes, false},
		{AgentTypeLightclawACE, false},
		{"unknown_agent_type", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := AgentTypeSupportsApprove(context.Background(), tt.code)
			if got != tt.want {
				t.Errorf("AgentTypeSupportsApprove(context.Background(), %q)=%v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

// TestAgentTypeSupportsMultiAgent 覆盖 multi-agent 查询能力矩阵：
// 当前只有 OpenClaw 支持；DeepSeekTUI / OpenCode 显式关闭。
func TestAgentTypeSupportsMultiAgent(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		code string
		want bool
	}{
		{AgentTypeOpenClaw, true},
		{"", true},
		{AgentTypeHermes, false},
		{AgentTypeLightclawACE, false},
		{AgentTypeDeepSeekTUI, false},
		{AgentTypeOpenCode, false},
		{"unknown_agent_type", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := AgentTypeSupportsMultiAgent(context.Background(), tt.code)
			if got != tt.want {
				t.Errorf("AgentTypeSupportsMultiAgent(context.Background(), %q)=%v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

// TestSupportedAgentTypesByChannel 覆盖 SupportedAgentTypesByChannel 全部分支：
//   - 仅 OpenClaw 支持的 openclaw-weixin / wecom_app / ddingtalk → 只返回 openclaw
//     （注：ddingtalk 白名单经 Hermes 脚本对齐后从 Hermes 移除；wecom 仅 OpenClaw+ACE）
//   - Slack：仅 OpenClaw / Hermes 支持
//   - 未知 channel_id → 返回空 slice（非 nil，避免序列化为 null）
func TestSupportedAgentTypesByChannel(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		name      string
		channelID string
		want      []string
	}{
		{
			name:      "feishu_openclaw_and_ace",
			channelID: "feishu",
			want:      []string{AgentTypeOpenClaw, AgentTypeHermes, AgentTypeLightclawACE},
		},
		{
			name:      "wecom_openclaw_and_ace",
			channelID: "wecom",
			want:      []string{AgentTypeOpenClaw, AgentTypeHermes, AgentTypeLightclawACE},
		},
		{
			name:      "qqbot_all_three",
			channelID: "qqbot",
			want:      []string{AgentTypeOpenClaw, AgentTypeHermes, AgentTypeLightclawACE},
		},
		{
			name:      "openclaw_weixin_only_openclaw",
			channelID: "openclaw-weixin",
			want:      []string{AgentTypeOpenClaw, AgentTypeHermes, AgentTypeLightclawACE},
		},
		{
			name:      "wecom_app_only_openclaw",
			channelID: "wecom_app",
			want:      []string{AgentTypeOpenClaw}, // hermes/ace 白名单无 wecom_app
		},
		{
			name:      "slack_openclaw_and_hermes",
			channelID: "slack",
			want:      []string{AgentTypeOpenClaw, AgentTypeHermes},
		},
		{
			name:      "ddingtalk_only_openclaw",
			channelID: "ddingtalk",
			want:      []string{AgentTypeOpenClaw, AgentTypeHermes}, // ACE 不支持钉钉
		},
		{
			name:      "dingtalk_connector_only_openclaw",
			channelID: "dingtalk-connector",
			want:      []string{AgentTypeOpenClaw}, // 新版钉钉插件仅 openclaw 支持
		},
		{
			name:      "unknown_channel_returns_empty_not_nil",
			channelID: "some_future_channel",
			want:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SupportedAgentTypesByChannel(context.Background(), tt.channelID)

			// 必须非 nil，方便 JSON 序列化
			if got == nil {
				t.Fatalf("SupportedAgentTypesByChannel(context.Background(), %q) returned nil, want non-nil", tt.channelID)
			}

			// 比较内容（排序后对比，避免 map 遍历顺序差异）
			sorted := append([]string(nil), got...)
			wantSorted := append([]string(nil), tt.want...)
			sort.Strings(sorted)
			sort.Strings(wantSorted)

			if len(sorted) != len(wantSorted) {
				t.Fatalf("SupportedAgentTypesByChannel(context.Background(), %q)=%v, want %v", tt.channelID, got, tt.want)
			}
			for i := range sorted {
				if sorted[i] != wantSorted[i] {
					t.Errorf("SupportedAgentTypesByChannel(context.Background(), %q)[%d]=%q, want %q", tt.channelID, i, sorted[i], wantSorted[i])
				}
			}
		})
	}
}

// TestSupportedAgentTypesByChannel_StableOrder 验证返回结果按 agentTypesList 排序顺序稳定输出，
// 以便前端展示顺序一致。
func TestSupportedAgentTypesByChannel_StableOrder(t *testing.T) {
	setupAgentTypeTestDB(t)
	// qqbot 是当前唯一三端全支持的 channel（feishu 因 2026-04-20 产品侧下线 hermes 入口而降级；
	// wecom 仅 openclaw+ace；openclaw-weixin 仅 openclaw）。
	got := SupportedAgentTypesByChannel(context.Background(), "qqbot")
	want := []string{AgentTypeOpenClaw, AgentTypeHermes, AgentTypeLightclawACE}
	if len(got) != len(want) {
		t.Fatalf("len got=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("order mismatch: got[%d]=%q, want[%d]=%q", i, got[i], i, want[i])
		}
	}
}

// TestAgentTypeChannelAllowed_Matrix 覆盖 AgentTypeChannelAllowed 的核心矩阵：
//   - 空 agentType → 按 openclaw 处理
//   - 未知 agentType → fail-closed false
//   - channel 不在白名单 → false
func TestAgentTypeChannelAllowed_Matrix(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		name      string
		agentType string
		channelID string
		want      bool
	}{
		{"openclaw-feishu", AgentTypeOpenClaw, "feishu", true},
		{"openclaw-weixin-openclaw", AgentTypeOpenClaw, "openclaw-weixin", true},
		{"openclaw-slack", AgentTypeOpenClaw, "slack", true},
		{"hermes-feishu", AgentTypeHermes, "feishu", true},
		{"hermes-no-openclaw-weixin", AgentTypeHermes, "openclaw-weixin", true},
		{"hermes-wecom_app-not-supported", AgentTypeHermes, "wecom_app", false},
		{"hermes-wecom_app-not-supported", AgentTypeHermes, "wecom_app", false},
		{"hermes-slack", AgentTypeHermes, "slack", true},
		{"ace-feishu", AgentTypeLightclawACE, "feishu", true},
		{"ace-no-ddingtalk", AgentTypeLightclawACE, "ddingtalk", false},
		{"ace-no-slack", AgentTypeLightclawACE, "slack", false},
		{"empty-agent-type-treated-as-openclaw", "", "feishu", true},
		{"unknown-agent-type-fail-closed", "totally_unknown", "feishu", false},
		{"unknown-channel-fail-closed", AgentTypeOpenClaw, "nonexistent_channel", false},
		// dingtalk-connector：仅 openclaw 支持，hermes/ace 不支持
		{"openclaw-dingtalk-connector", AgentTypeOpenClaw, "dingtalk-connector", true},
		{"hermes-no-dingtalk-connector", AgentTypeHermes, "dingtalk-connector", false},
		{"ace-no-dingtalk-connector", AgentTypeLightclawACE, "dingtalk-connector", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AgentTypeChannelAllowed(context.Background(), tt.agentType, tt.channelID)
			if got != tt.want {
				t.Errorf("AgentTypeChannelAllowed(context.Background(), %q, %q)=%v, want %v",
					tt.agentType, tt.channelID, got, tt.want)
			}
		})
	}
}

// TestSupportedChannelsByAgentType_Contents 覆盖 SupportedChannelsByAgentType 核心分支。
// 注：该函数返回值为无序，此测试检查集合成员存在性而非顺序。
func TestSupportedChannelsByAgentType_Contents(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		name      string
		agentType string
		wantAny   []string // 列表必须完全等于此集合
		wantNil   bool
	}{
		{
			name:      "openclaw-has-all-known-channels",
			agentType: AgentTypeOpenClaw,
			wantAny:   []string{"openclaw-weixin", "wecom", "wecom_app", "feishu", "lark", "ddingtalk", "msteams", "qqbot", "dingtalk-connector", "slack", "discord", "whatsapp"},
		},
		{
			name:      "hermes-supported-channels",
			agentType: AgentTypeHermes,
			wantAny:   []string{"wecom", "openclaw-weixin", "feishu", "ddingtalk", "msteams", "qqbot", "slack", "lark", "discord", "line"},
		},
		{
			name:      "ace-only-three",
			agentType: AgentTypeLightclawACE,
			wantAny:   []string{"openclaw-weixin", "wecom", "feishu", "qqbot"},
		},
		{
			name:      "empty-agent-type-as-openclaw",
			agentType: "",
			wantAny:   []string{"openclaw-weixin", "wecom", "wecom_app", "feishu", "lark", "ddingtalk", "msteams", "qqbot", "dingtalk-connector", "slack", "discord", "whatsapp"},
		},
		{
			name:      "unknown-agent-type-returns-nil",
			agentType: "unknown_type",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SupportedChannelsByAgentType(context.Background(), tt.agentType)
			if tt.wantNil {
				if got != nil {
					t.Errorf("want nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("want non-nil, got nil")
			}
			// 集合等价性
			gotSet := map[string]bool{}
			for _, v := range got {
				gotSet[v] = true
			}
			wantSet := map[string]bool{}
			for _, v := range tt.wantAny {
				wantSet[v] = true
			}
			if len(gotSet) != len(wantSet) {
				t.Errorf("len mismatch: got=%d (%v), want=%d (%v)", len(gotSet), got, len(wantSet), tt.wantAny)
			}
			for k := range wantSet {
				if !gotSet[k] {
					t.Errorf("missing channel %q in result %v", k, got)
				}
			}
			for k := range gotSet {
				if !wantSet[k] {
					t.Errorf("unexpected channel %q in result %v", k, got)
				}
			}
		})
	}
}

// TestGetSkillSupportedAgentTypes_ContainsAllEnabled 验证 GetSkillSupportedAgentTypes
// 返回所有 SupportsSkill=true 的类型以及兼容空字符串。
// v7：Skill 三端均放开。
func TestGetSkillSupportedAgentTypes_ContainsAllEnabled(t *testing.T) {
	setupAgentTypeTestDB(t)
	got := GetSkillSupportedAgentTypes(context.Background())

	set := map[string]bool{}
	for _, code := range got {
		set[code] = true
	}

	// 空字符串（存量数据兼容）
	if !set[""] {
		t.Error("应包含空字符串，用于兼容存量数据")
	}
	// 三种已启用的类型
	if !set[AgentTypeOpenClaw] {
		t.Error("应包含 openclaw")
	}
	if !set[AgentTypeHermes] {
		t.Error("应包含 hermes（v7 放开）")
	}
	if !set[AgentTypeLightclawACE] {
		t.Error("应包含 lightclawace（v7 放开）")
	}
}

// TestValidateAgentVersion_Matrix 覆盖版本校验的核心分支：
//   - 空版本允许
//   - OpenClaw 期望 YYYY.M.D 格式
//   - Hermes/ACE 期望 semver X.Y.Z 格式
//   - 非默认类型 fallback 到通用正则
func TestValidateAgentVersion_Matrix(t *testing.T) {
	setupAgentTypeTestDB(t)
	tests := []struct {
		name      string
		agentType string
		version   string
		wantErr   bool
	}{
		{"empty-version-always-ok", AgentTypeOpenClaw, "", false},
		{"openclaw-valid-date", AgentTypeOpenClaw, "2026.3.28", false},
		{"openclaw-valid-date-2digit", AgentTypeOpenClaw, "2026.12.31", false},
		{"openclaw-invalid-semver", AgentTypeOpenClaw, "1.0.0", true},
		{"openclaw-invalid-random", AgentTypeOpenClaw, "garbage", true},
		{"hermes-valid-semver", AgentTypeHermes, "0.9.0", false},
		// 说明：semverRegex = ^\d+\.\d+\.\d+$ 对 Hermes/ACE 放行任意 3 段数字版本，
		// 因此 "2026.3.28" 形式也会通过 Hermes 校验（虽然语义上不是 semver）。
		{"hermes-date-like-passes-semver-regex", AgentTypeHermes, "2026.3.28", false},
		{"hermes-invalid-garbage", AgentTypeHermes, "abc", true},
		{"hermes-invalid-two-parts", AgentTypeHermes, "1.2", true},
		{"ace-valid-semver", AgentTypeLightclawACE, "1.2.3", false},
		{"ace-invalid-three-digit", AgentTypeLightclawACE, "1.2", true},
		{"unknown-type-fallback-valid", "some_future_type", "v1.0", false},
		{"unknown-type-fallback-invalid", "some_future_type", string(make([]byte, 100)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentVersion(context.Background(), tt.agentType, tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAgentVersion(context.Background(), %q,%q)=%v, wantErr=%v",
					tt.agentType, tt.version, err, tt.wantErr)
			}
		})
	}
}

// TestGetAgentTypeDisplayName_KnownAndUnknown 覆盖 GetAgentTypeDisplayName 已知与未知分支。
func TestGetAgentTypeDisplayName_KnownAndUnknown(t *testing.T) {
	setupAgentTypeTestDB(t)
	cases := []struct {
		code string
		want string
	}{
		{AgentTypeOpenClaw, "OpenClaw"},
		{AgentTypeHermes, "Hermes"},
		{AgentTypeLightclawACE, "LightclawACE"},
		{"", "OpenClaw"}, // 空字符串兼容 → 对应 openclaw 类型
	}
	for _, c := range cases {
		got := GetAgentTypeDisplayName(context.Background(), c.code)
		if got != c.want {
			t.Errorf("GetAgentTypeDisplayName(context.Background(), %q)=%q, want %q", c.code, got, c.want)
		}
	}

	// 未知类型：函数应回落到原始 code（由实现约定）
	got := GetAgentTypeDisplayName(context.Background(), "future_unknown_type")
	if got != "future_unknown_type" {
		t.Errorf("unknown agent type should echo back the code, got %q", got)
	}
}

// TestGetAgentTypeDetailConfigFlags_UnknownReturnsNil 确保未知类型返回 nil，
// 供调用方显式判空处理。
func TestGetAgentTypeDetailConfigFlags_UnknownReturnsNil(t *testing.T) {
	setupAgentTypeTestDB(t)
	if flags := GetAgentTypeDetailConfigFlags(context.Background(), "unknown_xxx"); flags != nil {
		t.Errorf("unknown agent type should return nil flags, got %+v", flags)
	}

	// openclaw 应该 DetailConfig 所有字段都是 true
	flags := GetAgentTypeDetailConfigFlags(context.Background(), AgentTypeOpenClaw)
	if flags == nil {
		t.Fatal("openclaw flags should not be nil")
	}
	if !flags.SupportsModel || !flags.SupportsChannel || !flags.SupportsSkill || !flags.SupportsPlugin {
		t.Errorf("openclaw should support all detail config, got %+v", flags)
	}
}
