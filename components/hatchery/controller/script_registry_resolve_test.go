package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

func TestResolveScript(t *testing.T) {
	tests := []struct {
		name      string
		feature   string
		agentType string
		want      string
		wantErr   bool
	}{
		// 基线：openclaw 全量
		{"openclaw-set_model", "set_model", model.AgentTypeOpenClaw, "set_model.sh", false},
		{"openclaw-add_skill", "add_skill", model.AgentTypeOpenClaw, "add_skill.sh", false},
		{"openclaw-qq_bot_creator", "qq_bot_creator", model.AgentTypeOpenClaw, "qq_bot_creator.sh", false},
		{"openclaw-restart_gateway", "restart_gateway", model.AgentTypeOpenClaw, "restart_gateway.sh", false},

		// Hermes 对齐
		{"hermes-set_model", "set_model", model.AgentTypeHermes, "set_model_hermes.sh", false},
		{"hermes-feishu_bot", "feishu_bot_creator", model.AgentTypeHermes, "feishu_bot_creator_hermes.sh", false},
		{"hermes-restart_gateway", "restart_gateway", model.AgentTypeHermes, "restart_gateway_hermes.sh", false},

		// ACE 对齐
		{"ace-install_skill_from_smh", "install_skill_from_smh", model.AgentTypeLightclawACE, "install_skill_from_smh_ace.sh", false},
		{"ace-restart_gateway", "restart_gateway", model.AgentTypeLightclawACE, "restart_gateway_ace.sh", false},

		// fail-closed 场景：feature 存在但 agentType 不在其中
		// final §8.2：ACE 飞书已有 wrapper，放开
		{"ace-feishu-supported", "feishu_bot_creator", model.AgentTypeLightclawACE, "feishu_bot_creator_ace.sh", false},
		{"hermes-qqbot-not-supported", "qq_bot_creator", model.AgentTypeHermes, "", true},
		{"ace-qqbot-not-supported", "qq_bot_creator", model.AgentTypeLightclawACE, "", true},

		// 未知 feature
		{"unknown-feature", "nonexistent", model.AgentTypeOpenClaw, "", true},

		// 未知 agentType
		{"unknown-agent", "set_model", "unknown", "", true},

		// 空 agentType 视为 openclaw
		{"empty-agent", "set_model", "", "set_model.sh", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveScript(context.Background(), tt.feature, tt.agentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveScript(context.Background(), %q, %q) error = %v, wantErr %v", tt.feature, tt.agentType, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveScript(context.Background(), %q, %q) = %q, want %q", tt.feature, tt.agentType, got, tt.want)
			}
		})
	}
}

func TestExpandIncludes_NoDirective(t *testing.T) {
	content := "#!/bin/bash\necho hello\n"
	got, err := ExpandIncludes(content, nil) // loader 不会被调用
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != content {
		t.Errorf("content should be unchanged when no include, got %q", got)
	}
}

func TestExpandIncludes_SingleInclude(t *testing.T) {
	libBody := "echo 'from lib'"
	loader := func(name string) (string, error) {
		if name == "lib_qr_render.sh" {
			return libBody, nil
		}
		return "", fmt.Errorf("not found: %s", name)
	}
	content := "#!/bin/bash\n# %INCLUDE% lib_qr_render.sh\necho done\n"
	got, err := ExpandIncludes(content, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, libBody) {
		t.Errorf("expected lib body inlined, got: %q", got)
	}
	if !strings.Contains(got, "echo done") {
		t.Errorf("original content after include should be preserved, got: %q", got)
	}
}

func TestExpandIncludes_InvalidName_DirectoryTraversal(t *testing.T) {
	ctx := context.Background()

	// 安全性：禁止目录穿越
	loader := func(name string) (string, error) {
		t.Fatalf("loader should not be called for invalid name, got: %s", name)
		return "", nil
	}
	content := "# %INCLUDE% ../etc/passwd\n"
	_, err := ExpandIncludes(content, loader)
	if err == nil {
		t.Fatal("expected error for directory traversal attempt")
	}

	var re *hcommon.RichError
	if !errors.As(err, &re) {
		t.Fatalf("expected RichError, got: %T", err)
	}

	wanted := hcommon.I18nError(i18n.MsgInvalidIncludeName, "../etc/passwd")
	if !errors.Is(re, wanted) {
		t.Errorf("expected '%s' error, got: %s", wanted.ErrorMessage(ctx), re.ErrorMessage(ctx))
	}
}

func TestExpandIncludes_InvalidName_NonLibPrefix(t *testing.T) {
	// 白名单强制：必须 lib_ 前缀
	content := "# %INCLUDE% helper.sh\n"
	_, err := ExpandIncludes(content, nil)
	if err == nil {
		t.Fatal("expected error for non-lib_ prefix")
	}
}

func TestExpandIncludes_LoaderFailure(t *testing.T) {
	loader := func(name string) (string, error) {
		return "", fmt.Errorf("disk error")
	}
	content := "# %INCLUDE% lib_missing.sh\n"
	_, err := ExpandIncludes(content, loader)
	if err == nil {
		t.Fatal("expected error for loader failure")
	}
	if !strings.Contains(err.Error(), "lib_missing.sh") {
		t.Errorf("error should mention lib name, got: %v", err)
	}
}

func TestExpandIncludes_CircularReference(t *testing.T) {
	ctx := context.Background()

	// 深度保护：> 4 报错
	loader := func(name string) (string, error) {
		return "# %INCLUDE% lib_self.sh\n", nil // 自引用
	}
	content := "# %INCLUDE% lib_self.sh\n"
	_, err := ExpandIncludes(content, loader)
	if err == nil {
		t.Fatal("expected depth exceeded error for circular reference")
	}

	var re *hcommon.RichError
	if !errors.As(err, &re) {
		t.Fatalf("expected RichError, got: %T", err)
	}

	wanted := hcommon.I18nError(i18n.MsgIncludeDepthExceeded)
	if !errors.Is(err, wanted) {
		t.Errorf("expected %s, got: %s", wanted.ErrorMessage(ctx), re.ErrorMessage(ctx))
	}
}

func TestExpandIncludes_NestedIncludes(t *testing.T) {
	// 合法嵌套：lib_a 内包含 lib_b
	loader := func(name string) (string, error) {
		switch name {
		case "lib_a.sh":
			return "# level A\n# %INCLUDE% lib_b.sh\n", nil
		case "lib_b.sh":
			return "# level B content\n", nil
		}
		return "", fmt.Errorf("not found")
	}
	content := "#!/bin/bash\n# %INCLUDE% lib_a.sh\necho end\n"
	got, err := ExpandIncludes(content, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "level A") || !strings.Contains(got, "level B content") {
		t.Errorf("expected both lib_a and lib_b inlined, got: %q", got)
	}
}

func TestExpandIncludes_MultipleIncludesSameScript(t *testing.T) {
	// 同一脚本内多个 include：验证倒序替换正确性（offset 不乱）
	loader := func(name string) (string, error) {
		return "LIB_" + strings.TrimSuffix(strings.TrimPrefix(name, "lib_"), ".sh"), nil
	}
	content := "# %INCLUDE% lib_one.sh\nmiddle\n# %INCLUDE% lib_two.sh\n"
	got, err := ExpandIncludes(content, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "LIB_one") || !strings.Contains(got, "LIB_two") {
		t.Errorf("both libs should be inlined, got: %q", got)
	}
	// 验证 middle 顺序保留
	idxOne := strings.Index(got, "LIB_one")
	idxMiddle := strings.Index(got, "middle")
	idxTwo := strings.Index(got, "LIB_two")
	if !(idxOne < idxMiddle && idxMiddle < idxTwo) {
		t.Errorf("order broken: one=%d, middle=%d, two=%d", idxOne, idxMiddle, idxTwo)
	}
}
