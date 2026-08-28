package controller

import (
	"context"
	"testing"

	hcommon "hatchery/common"
	"hatchery/model"
)

// TestCheckInstanceSupportsReinstall 覆盖 checkInstanceSupportsReinstall 全部分支。
func TestCheckInstanceSupportsReinstall(t *testing.T) {
	ctx := context.Background()
	initAgentTypeGuardTestDB(t)

	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"openclaw supports reinstall", model.AgentTypeOpenClaw, false},
		{"empty string supports (legacy openclaw)", "", false},
		{"hermes supports reinstall", model.AgentTypeHermes, false},
		{"lightclawace supports reinstall", model.AgentTypeLightclawACE, false},
		{"unknown type not supports", "unknown_type", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &model.Instance{AgentType: tt.agentType}
			err := checkInstanceSupportsReinstall(ctx, instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkInstanceSupportsReinstall() agentType=%q error = %v, wantErr %v",
					tt.agentType, err, tt.wantErr)
			}
			// 拦截类型时，错误消息应包含 agent 类型的显示名，并建议"请删除后重建"
			if tt.wantErr && err != nil {
				switch tt.agentType {
				case model.AgentTypeHermes:
					if !guardContainsStr(err.Error(), "Hermes") {
						t.Errorf("error should contain 'Hermes', got: %s", err.Error())
					}
				case model.AgentTypeLightclawACE:
					if !guardContainsStr(err.Error(), "LightclawACE") {
						t.Errorf("error should contain 'LightclawACE', got: %s", err.Error())
					}
				}
				if !guardContainsStr(err.Error(), "重装") {
					t.Errorf("error should mention 重装, got: %s", err.Error())
				}
			}
		})
	}
}

// TestCheckInstanceSupportsReinstall_NilInstance nil 实例不应返回错误（预先约定）。
func TestCheckInstanceSupportsReinstall_NilInstance(t *testing.T) {
	if err := checkInstanceSupportsReinstall(context.Background(), nil); err != nil {
		t.Errorf("nil instance should not return error, got %v", err)
	}
}

// TestCheckInstanceSupportsMemoryExtended 覆盖 checkInstanceSupportsMemory 全部分支。
// 仅 OpenClaw 支持记忆功能（Memory TDAI）。
func TestCheckInstanceSupportsMemoryExtended(t *testing.T) {
	ctx := context.Background()
	initAgentTypeGuardTestDB(t)

	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"openclaw supports memory", model.AgentTypeOpenClaw, false},
		{"empty string supports (legacy openclaw)", "", false},
		{"hermes supports memory", model.AgentTypeHermes, false},
		{"lightclawace not supports memory", model.AgentTypeLightclawACE, true},
		{"unknown type not supports", "unknown_type", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &model.Instance{AgentType: tt.agentType}
			err := checkInstanceSupportsMemory(ctx, instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkInstanceSupportsMemory() agentType=%q error = %v, wantErr %v",
					tt.agentType, err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !guardContainsStr(err.Error(), "记忆") {
				t.Errorf("error should mention 记忆, got: %s", err.Error())
			}
		})
	}
}

// TestCheckInstanceSupportsMemory_NilInstance nil 实例不应返回错误。
func TestCheckInstanceSupportsMemory_NilInstance(t *testing.T) {
	if err := checkInstanceSupportsMemory(context.Background(), nil); err != nil {
		t.Errorf("nil instance should not return error, got %v", err)
	}
}

// TestCheckInstanceSupportsBrowserVNC 覆盖 checkInstanceSupportsBrowserVNC 全部分支。
// 仅 OpenClaw 支持云端浏览器。
func TestCheckInstanceSupportsBrowserVNC(t *testing.T) {
	ctx := context.Background()
	initAgentTypeGuardTestDB(t)

	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"openclaw supports browser vnc", model.AgentTypeOpenClaw, false},
		{"empty string supports (legacy openclaw)", "", false},
		{"hermes not supports browser vnc", model.AgentTypeHermes, true},
		{"lightclawace not supports browser vnc", model.AgentTypeLightclawACE, true},
		{"unknown type not supports", "unknown_type", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &model.Instance{AgentType: tt.agentType}
			err := checkInstanceSupportsBrowserVNC(ctx, instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkInstanceSupportsBrowserVNC() agentType=%q error = %v, wantErr %v",
					tt.agentType, err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !guardContainsStr(err.Error(), "云端浏览器") {
				t.Errorf("error should mention 云端浏览器, got: %s", err.Error())
			}
		})
	}
}

// TestCheckInstanceSupportsBrowserVNC_NilInstance nil 实例不应返回错误。
func TestCheckInstanceSupportsBrowserVNC_NilInstance(t *testing.T) {
	if err := checkInstanceSupportsBrowserVNC(context.Background(), nil); err != nil {
		t.Errorf("nil instance should not return error, got %v", err)
	}
}

// TestCheckInstanceSupportsApprove 覆盖 checkInstanceSupportsApprove 全部分支。
// 仅 OpenClaw 需要 approve/approve_device 脚本流程。
func TestCheckInstanceSupportsApprove(t *testing.T) {
	ctx := context.Background()
	initAgentTypeGuardTestDB(t)

	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"openclaw supports approve", model.AgentTypeOpenClaw, false},
		{"empty string supports (legacy openclaw)", "", false},
		{"hermes not supports approve", model.AgentTypeHermes, true},
		{"lightclawace not supports approve", model.AgentTypeLightclawACE, true},
		{"unknown type not supports", "unknown_type", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &model.Instance{AgentType: tt.agentType}
			err := checkInstanceSupportsApprove(ctx, instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkInstanceSupportsApprove() agentType=%q error = %v, wantErr %v",
					tt.agentType, err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !guardContainsStr(err.Error(), "授权") {
				t.Errorf("error should mention 授权, got: %s", err.Error())
			}
		})
	}
}

// TestCheckInstanceSupportsApprove_NilInstance nil 实例不应返回错误。
func TestCheckInstanceSupportsApprove_NilInstance(t *testing.T) {
	if err := checkInstanceSupportsApprove(context.Background(), nil); err != nil {
		t.Errorf("nil instance should not return error, got %v", err)
	}
}

// TestCheckGuardAllNewGuardsAgainstNil 所有 guard 对 nil 实例应返回 nil（预约定）。
// 包括新增的 Reinstall/Memory/BrowserVNC/Approve 四个 guard。
func TestCheckGuardAllNewGuardsAgainstNil(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		check func(context.Context, *model.Instance) *hcommon.RichError
	}{
		{"Reinstall", checkInstanceSupportsReinstall},
		{"Memory", checkInstanceSupportsMemory},
		{"BrowserVNC", checkInstanceSupportsBrowserVNC},
		{"Approve", checkInstanceSupportsApprove},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.check(ctx, nil); err != nil {
				t.Errorf("check%s(nil) should return nil, got %v", tt.name, err)
			}
		})
	}
}

// TestCheckGuardMatrixForHermesAndAce 覆盖 Hermes/ACE 在多个 guard 下的预期行为。
// 此测试是对 final §3.2 能力矩阵的汇总回归测试。
func TestCheckGuardMatrixForHermesAndAce(t *testing.T) {
	ctx := context.Background()

	type guardCase struct {
		name      string
		check     func(context.Context, *model.Instance) *hcommon.RichError
		hermesErr bool // Hermes 是否应拦截
		aceErr    bool // ACE 是否应拦截
	}

	cases := []guardCase{
		// DetailConfig（Model/Channel/Skill/Plugin 任一）：三端均放开（v7）
		{"DetailConfig", checkInstanceSupportsDetailConfig, false, false},
		// Plugin：仅 OpenClaw
		{"Plugin", checkInstanceSupportsPlugin, true, true},
		// Skill：三端放开（v7）
		{"Skill", checkInstanceSupportsSkill, false, false},
		// Model：三端放开（v7）
		{"Model", checkInstanceSupportsModel, false, false},
		// Channel：三端放开（v7）
		{"Channel", checkInstanceSupportsChannel, false, false},
		// Chatbot：OpenClaw + ACE，Hermes 拦截
		{"Chatbot", checkInstanceSupportsChatbot, true, false},
		// Reinstall：三期放开，三端均支持
		{"Reinstall", checkInstanceSupportsReinstall, false, false},
		// Memory：OpenClaw + Hermes 支持，ACE 拦截
		{"Memory", checkInstanceSupportsMemory, false, true},
		{"BrowserVNC", checkInstanceSupportsBrowserVNC, true, true},
		{"Approve", checkInstanceSupportsApprove, true, true},
	}

	hermesInst := &model.Instance{AgentType: model.AgentTypeHermes}
	aceInst := &model.Instance{AgentType: model.AgentTypeLightclawACE}
	openclawInst := &model.Instance{AgentType: model.AgentTypeOpenClaw}

	for _, c := range cases {
		t.Run(c.name+"-hermes", func(t *testing.T) {
			err := c.check(ctx, hermesInst)
			if (err != nil) != c.hermesErr {
				t.Errorf("[%s] hermes: err=%v, wantErr=%v", c.name, err, c.hermesErr)
			}
		})
		t.Run(c.name+"-ace", func(t *testing.T) {
			err := c.check(ctx, aceInst)
			if (err != nil) != c.aceErr {
				t.Errorf("[%s] ace: err=%v, wantErr=%v", c.name, err, c.aceErr)
			}
		})
		t.Run(c.name+"-openclaw-always-ok", func(t *testing.T) {
			if err := c.check(ctx, openclawInst); err != nil {
				t.Errorf("[%s] openclaw should always pass, got err=%v", c.name, err)
			}
		})
	}
}

// ─── verifyReinstallImageMatches ───────────────────────────────────────────

func TestVerifyReinstallImageMatches(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		instType string
		imgType  string
		wantErr  bool
	}{
		{"nil instance", "", "", false},
		{"same type openclaw", model.AgentTypeOpenClaw, model.AgentTypeOpenClaw, false},
		{"same type hermes", model.AgentTypeHermes, model.AgentTypeHermes, false},
		{"same type ace", model.AgentTypeLightclawACE, model.AgentTypeLightclawACE, false},
		{"openclaw + empty img (legacy)", model.AgentTypeOpenClaw, "", false},
		{"hermes + empty img mismatch", model.AgentTypeHermes, "", true},
		{"ace + openclaw img mismatch", model.AgentTypeLightclawACE, model.AgentTypeOpenClaw, true},
		{"hermes + openclaw img mismatch", model.AgentTypeHermes, model.AgentTypeOpenClaw, true},
		// 以下用例覆盖「实例 AgentType 为空 → 归一为 OpenClaw」分支
		{"empty inst + empty img (legacy openclaw both)", "", "", false},
		{"empty inst + openclaw img (legacy upgrade)", "", model.AgentTypeOpenClaw, false},
		{"empty inst + hermes img mismatch", "", model.AgentTypeHermes, true},
		{"empty inst + ace img mismatch", "", model.AgentTypeLightclawACE, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nil instance" {
				err := verifyReinstallImageMatches(ctx, nil, nil)
				if err != nil {
					t.Errorf("nil inputs should return nil, got %v", err)
				}
				// nil img 也应直接通过（覆盖 instance != nil && img == nil 分支）
				inst := &model.Instance{AgentType: model.AgentTypeOpenClaw}
				if err := verifyReinstallImageMatches(ctx, inst, nil); err != nil {
					t.Errorf("nil img should return nil, got %v", err)
				}
				// 同理 instance == nil && img != nil
				img := &model.AIImage{AgentType: model.AgentTypeHermes, ImageId: "img-test"}
				if err := verifyReinstallImageMatches(ctx, nil, img); err != nil {
					t.Errorf("nil instance should return nil, got %v", err)
				}
				return
			}
			inst := &model.Instance{AgentType: tt.instType}
			img := &model.AIImage{AgentType: tt.imgType, ImageId: "img-test"}
			err := verifyReinstallImageMatches(ctx, inst, img)
			if (err != nil) != tt.wantErr {
				t.Errorf("instType=%q imgType=%q err=%v wantErr=%v", tt.instType, tt.imgType, err, tt.wantErr)
			}
		})
	}
}

// TestVerifyReinstallImageMatches_EmptyInstanceNormalized 专门验证「实例 AgentType
// 为空时内部归一化为 openclaw」这一行为：
//  1. 空实例 + openclaw 镜像 → 通过（归一后匹配）；
//  2. 空实例 + 空镜像 → 通过（兼容例外）；
//  3. 空实例 + hermes 镜像 → 拒绝，且错误消息包含正确的类型显示名（OpenClaw）。
//
// 注意：函数不再直接修改 instance.AgentType（使用局部变量归一），避免调用方副作用。
func TestVerifyReinstallImageMatches_EmptyInstanceNormalized(t *testing.T) {
	ctx := context.Background()

	// case 1：空实例 + openclaw 镜像 → 通过
	inst := &model.Instance{AgentType: ""}
	img := &model.AIImage{AgentType: model.AgentTypeOpenClaw, ImageId: "img-1"}
	if err := verifyReinstallImageMatches(ctx, inst, img); err != nil {
		t.Fatalf("empty inst + openclaw img should pass, got err=%v", err)
	}
	// 函数不再修改 instance.AgentType（无副作用）
	if inst.AgentType != "" {
		t.Errorf("expected instance.AgentType unchanged (no mutation), got %q", inst.AgentType)
	}

	// case 2：空实例 + 空镜像 → 通过（兼容例外）
	inst2 := &model.Instance{AgentType: ""}
	img2 := &model.AIImage{AgentType: "", ImageId: "img-2"}
	if err := verifyReinstallImageMatches(ctx, inst2, img2); err != nil {
		t.Fatalf("empty inst + empty img should pass (legacy), got err=%v", err)
	}
	if inst2.AgentType != "" {
		t.Errorf("expected instance.AgentType unchanged (no mutation), got %q", inst2.AgentType)
	}

	// case 3：空实例 + hermes 镜像 → 拒绝，错误消息应包含 OpenClaw（归一后显示名）
	inst3 := &model.Instance{AgentType: ""}
	img3 := &model.AIImage{AgentType: model.AgentTypeHermes, ImageId: "img-3"}
	err := verifyReinstallImageMatches(ctx, inst3, img3)
	if err == nil {
		t.Fatal("empty inst + hermes img should be rejected, got nil")
	}
	if !guardContainsStr(err.Error(), "OpenClaw") {
		t.Errorf("error should mention OpenClaw (normalized inst type), got: %s", err.Error())
	}
	if !guardContainsStr(err.Error(), "Hermes") {
		t.Errorf("error should mention Hermes (img type), got: %s", err.Error())
	}
}

// TestVerifyReinstallImageMatches_ErrorMessage 覆盖错误消息构造分支：
//   - 镜像 AgentType 非空 → 走 GetAgentTypeDisplayName(img.AgentType) 分支
//   - 镜像 AgentType 为空 → 走 "未分类" 分支
func TestVerifyReinstallImageMatches_ErrorMessage(t *testing.T) {
	ctx := context.Background()

	// 镜像非空类型分支：hermes 实例 + openclaw 镜像
	inst := &model.Instance{AgentType: model.AgentTypeHermes}
	img := &model.AIImage{AgentType: model.AgentTypeOpenClaw, ImageId: "img-x"}
	err := verifyReinstallImageMatches(ctx, inst, img)
	if err == nil {
		t.Fatal("hermes inst + openclaw img should be rejected")
	}
	msg := err.Error()
	if !guardContainsStr(msg, "OpenClaw") {
		t.Errorf("error should contain OpenClaw, got: %s", msg)
	}
	if !guardContainsStr(msg, "Hermes") {
		t.Errorf("error should contain Hermes, got: %s", msg)
	}
	if !guardContainsStr(msg, "无法重装") {
		t.Errorf("error should contain 无法重装, got: %s", msg)
	}

	// 镜像空类型分支（"未分类"）：hermes 实例 + 空类型镜像
	inst2 := &model.Instance{AgentType: model.AgentTypeHermes}
	img2 := &model.AIImage{AgentType: "", ImageId: "img-y"}
	err2 := verifyReinstallImageMatches(ctx, inst2, img2)
	if err2 == nil {
		t.Fatal("hermes inst + empty img should be rejected")
	}
	if !guardContainsStr(err2.Error(), "未分类") {
		t.Errorf("error should contain 未分类 for empty img type, got: %s", err2.Error())
	}
}
