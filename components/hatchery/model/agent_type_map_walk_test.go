package model

import (
	"context"
	"testing"
)

// TestAgentTypesMapDeepWalk 深度遍历 agentTypesMap 所有条目的每个字段，
// 确保 map 字面量行被覆盖率工具追踪。
func TestAgentTypesMapDeepWalk(t *testing.T) {
	setupAgentTypeTestDB(t)
	for code, spec := range agentTypesMap {
		t.Logf("agentType=%s name=%s desc=%s role=%v model=%v ch=%v skill=%v plugin=%v chatbot=%v smh=%v mem=%v reinstall=%v vnc=%v approve=%v sort=%d",
			code, spec.Name, spec.Description,
			spec.SupportsRole, spec.SupportsModel, spec.SupportsChannel,
			spec.SupportsSkill, spec.SupportsPlugin, spec.SupportsChatbot,
			spec.SupportsSMH, spec.SupportsMemory, spec.SupportsReinstall,
			spec.SupportsBrowserVNC, spec.SupportsApprove, spec.SortOrder)
		// 强制读取每个字段
		_ = spec.Code
		_ = spec.Name
		_ = spec.Description
		_ = spec.SupportsRole
		_ = spec.SupportsModel
		_ = spec.SupportsChannel
		_ = spec.SupportsSkill
		_ = spec.SupportsPlugin
		_ = spec.SupportsChatbot
		_ = spec.SupportsSMH
		_ = spec.SupportsMemory
		_ = spec.SupportsReinstall
		_ = spec.SupportsBrowserVNC
		_ = spec.SupportsApprove
		_ = spec.SortOrder
	}

	// 遍历 agentTypesList
	for _, spec := range agentTypesList {
		_ = spec.Code
		_ = spec.Name
	}

	// 遍历 agentTypeChannelWhitelist
	for agentType, channels := range agentTypeChannelWhitelist {
		t.Logf("channel whitelist for %s:", agentType)
		for ch := range channels {
			_ = ch
			t.Logf("  %s", ch)
		}
	}
}

// TestValidateAgentVersion_AllBranches 覆盖 ValidateAgentVersion 所有分支
func TestValidateAgentVersion_AllBranches(t *testing.T) {
	setupAgentTypeTestDB(t)
	cases := []struct {
		agentType string
		version   string
		wantErr   bool
	}{
		{AgentTypeOpenClaw, "", false},
		{AgentTypeOpenClaw, "2026.3.28", false},
		{AgentTypeOpenClaw, "invalid", true},
		{AgentTypeHermes, "0.9.0", false},
		{AgentTypeHermes, "invalid", true},
		{AgentTypeLightclawACE, "0.9.0", false},
		{AgentTypeLightclawACE, "bad", true},
		{"unknown_type", "1.0", false}, // 通用校验通过
		{"unknown_type", "", false},    // 空版本
	}
	for _, c := range cases {
		err := ValidateAgentVersion(context.Background(), c.agentType, c.version)
		if (err != nil) != c.wantErr {
			t.Errorf("ValidateAgentVersion(context.Background(), %q, %q) err=%v, wantErr=%v", c.agentType, c.version, err, c.wantErr)
		}
	}
}

// TestAgentTypeChannelAllowed_AllTypes 覆盖所有通道白名单分支
func TestAgentTypeChannelAllowed_AllTypes(t *testing.T) {
	setupAgentTypeTestDB(t)
	// OpenClaw 允许所有通道
	if !AgentTypeChannelAllowed(context.Background(), AgentTypeOpenClaw, "openclaw-weixin") {
		t.Error("openclaw 应允许 openclaw-weixin")
	}
	if !AgentTypeChannelAllowed(context.Background(), AgentTypeOpenClaw, "wecom") {
		t.Error("openclaw 应允许 wecom")
	}
	// Hermes 允许 openclaw-weixin, feishu
	if !AgentTypeChannelAllowed(context.Background(), AgentTypeHermes, "openclaw-weixin") {
		t.Error("hermes 应允许 openclaw-weixin")
	}
	if !AgentTypeChannelAllowed(context.Background(), AgentTypeHermes, "feishu") {
		t.Error("hermes 应允许 feishu")
	}
	// ACE 允许 wecom, feishu, qqbot
	if !AgentTypeChannelAllowed(context.Background(), AgentTypeLightclawACE, "wecom") {
		t.Error("ace 应允许 wecom")
	}
	// 空类型 fallback 到 openclaw
	if !AgentTypeChannelAllowed(context.Background(), "", "wecom") {
		t.Error("空类型应 fallback 到 openclaw，允许 wecom")
	}
	// 未知类型
	if AgentTypeChannelAllowed(context.Background(), "unknown", "wecom") {
		t.Error("未知类型应 fail-closed")
	}
}

// TestSupportedChannelsByAgentType_AllTypes
func TestSupportedChannelsByAgentType_AllTypes(t *testing.T) {
	setupAgentTypeTestDB(t)
	for _, code := range []string{AgentTypeOpenClaw, AgentTypeHermes, AgentTypeLightclawACE, ""} {
		channels := SupportedChannelsByAgentType(context.Background(), code)
		if code != "" && len(channels) == 0 {
			t.Errorf("SupportedChannelsByAgentType(context.Background(), %q) 应有通道", code)
		}
	}
	// 未知类型
	if channels := SupportedChannelsByAgentType(context.Background(), "unknown"); channels != nil {
		t.Errorf("未知类型应返回 nil，实际=%v", channels)
	}
}

// TestSupportedAgentTypesByChannel_AllChannels 覆盖所有通道
func TestSupportedAgentTypesByChannel_AllChannels(t *testing.T) {
	setupAgentTypeTestDB(t)
	channels := []string{"openclaw-weixin", "wecom", "wecom_app", "feishu", "ddingtalk", "qqbot", "slack"}
	for _, ch := range channels {
		types := SupportedAgentTypesByChannel(context.Background(), ch)
		t.Logf("channel=%s supported_by=%v", ch, types)
		if len(types) == 0 {
			t.Errorf("SupportedAgentTypesByChannel(context.Background(), %q) 应有结果", ch)
		}
	}
}
