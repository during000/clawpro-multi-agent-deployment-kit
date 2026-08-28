package controller

import (
	"context"
	"errors"
	"fmt"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"strings"
	"sync"
	"testing"
)

// TestRegisterInlineScript_BasicRoundTrip 注册后可查到，未注册时查不到。
func TestRegisterInlineScript_BasicRoundTrip(t *testing.T) {
	// 清理 registry，避免串扰
	inlineScriptMu.Lock()
	inlineScriptRegistry = make(map[string]string)
	inlineScriptMu.Unlock()

	name := "inline_test_script_abc"
	if _, ok := LookupInlineScript(name); ok {
		t.Fatalf("未注册时应查不到，却查到了 name=%s", name)
	}

	content := "#!/bin/bash\necho hello\n"
	RegisterInlineScript(name, content)

	got, ok := LookupInlineScript(name)
	if !ok {
		t.Fatalf("注册后应查得到，却查不到 name=%s", name)
	}
	if got != content {
		t.Errorf("内容不一致: got=%q want=%q", got, content)
	}

	UnregisterInlineScript(name)
	if _, ok := LookupInlineScript(name); ok {
		t.Errorf("注销后仍能查到 name=%s", name)
	}
}

// TestRegisterInlineScript_Overwrite 重复注册应覆盖旧值。
func TestRegisterInlineScript_Overwrite(t *testing.T) {
	inlineScriptMu.Lock()
	inlineScriptRegistry = make(map[string]string)
	inlineScriptMu.Unlock()

	name := "overwrite_script"
	RegisterInlineScript(name, "v1")
	RegisterInlineScript(name, "v2")

	got, ok := LookupInlineScript(name)
	if !ok {
		t.Fatal("应查到")
	}
	if got != "v2" {
		t.Errorf("覆盖后应为 v2，实际=%q", got)
	}

	UnregisterInlineScript(name)
}

// TestUnregisterInlineScript_Idempotent 注销一个不存在的 key 不应 panic。
func TestUnregisterInlineScript_Idempotent(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("注销不存在的脚本不应 panic，但触发了 panic: %v", r)
		}
	}()
	UnregisterInlineScript("never_registered_script_" + t.Name())
}

// TestInlineScriptRegistry_ConcurrentSafety 并发读写不 panic、不数据竞争。
func TestInlineScriptRegistry_ConcurrentSafety(t *testing.T) {
	inlineScriptMu.Lock()
	inlineScriptRegistry = make(map[string]string)
	inlineScriptMu.Unlock()

	const N = 50
	var wg sync.WaitGroup
	wg.Add(3)

	// 并发注册
	go func() {
		defer wg.Done()
		for i := 0; i < N; i++ {
			RegisterInlineScript(fmt.Sprintf("concurrent_%d", i), fmt.Sprintf("content_%d", i))
		}
	}()
	// 并发查找
	go func() {
		defer wg.Done()
		for i := 0; i < N; i++ {
			_, _ = LookupInlineScript(fmt.Sprintf("concurrent_%d", i))
		}
	}()
	// 并发注销
	go func() {
		defer wg.Done()
		for i := 0; i < N; i++ {
			UnregisterInlineScript(fmt.Sprintf("concurrent_%d", i))
		}
	}()

	wg.Wait()
}

// TestLookupInlineScript_EmptyStringContent 空字符串内容也是合法的注册。
func TestLookupInlineScript_EmptyStringContent(t *testing.T) {
	inlineScriptMu.Lock()
	inlineScriptRegistry = make(map[string]string)
	inlineScriptMu.Unlock()

	name := "empty_content_script"
	RegisterInlineScript(name, "")

	got, ok := LookupInlineScript(name)
	if !ok {
		t.Fatal("即使 content 为空字符串，注册后 Lookup 也应返回 ok=true")
	}
	if got != "" {
		t.Errorf("内容应为空字符串，实际=%q", got)
	}

	UnregisterInlineScript(name)
}

// TestResolveScript_AllFeaturesOpenClaw 覆盖 scriptResolveTable 中所有 feature 的 openclaw 分支，
// 保证新增 feature 后测试会提醒人维护此表。
func TestResolveScript_AllFeaturesOpenClaw(t *testing.T) {
	wants := map[string]string{
		"set_model":              "set_model.sh",
		"add_skill":              "add_skill.sh",
		"batch_install_skills":   "batch_install_skills_from_smh.sh",
		"install_skill_from_smh": "install_skill_from_smh.sh",
		"restart_gateway":        "restart_gateway.sh",
		"set_channel":            "set_channel.sh",
		"del_channel":            "del_channel.sh",
		"list_channels":          "list_channels.sh",
		"check_ready":            "check_openclaw_ready.sh",
		"weixin_bot_creator":     "weixin_bot_creator.sh",
		"feishu_bot_creator":     "feishu_bot_creator.sh",
		"qq_bot_creator":         "qq_bot_creator.sh",
		"init_smh_env":           "init_smh_env.sh",
		"remove_smh_env":         "remove_smh_env.sh",
		"set_smh_token":          "set_smh_token.sh",
		"get_version_info":       "get_version_info.sh",
		"list_skills":            "list_skills.sh",
		"set_env":                "set_env.sh",
		"get_env":                "get_env.sh",
		"check_service":          "check_service.sh",
		"detect_install":         "detect_openclaw_install.sh",
	}

	for feature, want := range wants {
		t.Run(feature, func(t *testing.T) {
			got, err := ResolveScript(context.Background(), feature, "openclaw")
			if err != nil {
				t.Fatalf("ResolveScript(context.Background(), %q, openclaw) unexpected err=%v", feature, err)
			}
			if got != want {
				t.Errorf("ResolveScript(context.Background(), %q, openclaw)=%q, want %q", feature, got, want)
			}
		})
	}
}

// TestResolveScript_AllFeaturesHermes 覆盖 Hermes 在 scriptResolveTable 中的各 feature。
// Hermes 不支持 weixin_bot_creator / qq_bot_creator，需要验证 fail-closed。
func TestResolveScript_AllFeaturesHermes(t *testing.T) {
	supported := map[string]string{
		"set_model":              "set_model_hermes.sh",
		"add_skill":              "add_skill_hermes.sh",
		"batch_install_skills":   "batch_install_skills_from_smh_hermes.sh",
		"install_skill_from_smh": "install_skill_from_smh_hermes.sh",
		"restart_gateway":        "restart_gateway_hermes.sh",
		"set_channel":            "set_channel_hermes.sh",
		"del_channel":            "del_channel_hermes.sh",
		"list_channels":          "list_channels_hermes.sh",
		"check_ready":            "check_hermes_ready.sh",
		"feishu_bot_creator":     "feishu_bot_creator_hermes.sh",
		"init_smh_env":           "init_smh_env.sh",
		"remove_smh_env":         "remove_smh_env.sh",
		"set_smh_token":          "set_smh_token.sh",
		"get_version_info":       "get_version_info_hermes.sh",
		"list_skills":            "list_skills_hermes.sh",
		"set_env":                "set_env_hermes.sh",
		"get_env":                "get_env_hermes.sh",
		"check_service":          "check_service_hermes.sh",
		"detect_install":         "detect_hermes_install.sh",
	}

	for feature, want := range supported {
		t.Run("hermes-supported-"+feature, func(t *testing.T) {
			got, err := ResolveScript(context.Background(), feature, "hermes")
			if err != nil {
				t.Fatalf("ResolveScript(context.Background(), %q, hermes) unexpected err=%v", feature, err)
			}
			if got != want {
				t.Errorf("ResolveScript(context.Background(), %q, hermes)=%q, want %q", feature, got, want)
			}
		})
	}

	// Hermes 不支持 qq_bot_creator
	for _, feature := range []string{"qq_bot_creator"} {
		t.Run("hermes-unsupported-"+feature, func(t *testing.T) {
			got, err := ResolveScript(context.Background(), feature, "hermes")
			if err == nil {
				t.Errorf("ResolveScript(context.Background(), %q, hermes)=%q, want error", feature, got)
			}
		})
	}
}

// TestResolveScript_AllFeaturesACE 覆盖 LightclawACE 在 scriptResolveTable 中的各 feature。
// ACE 不支持 weixin_bot_creator / qq_bot_creator，需要验证 fail-closed。
func TestResolveScript_AllFeaturesACE(t *testing.T) {
	supported := map[string]string{
		"set_model":              "set_model_ace.sh",
		"add_skill":              "add_skill_ace.sh",
		"batch_install_skills":   "batch_install_skills_from_smh_ace.sh",
		"install_skill_from_smh": "install_skill_from_smh_ace.sh",
		"restart_gateway":        "restart_gateway_ace.sh",
		"set_channel":            "set_channel_ace.sh",
		"del_channel":            "del_channel_ace.sh",
		"list_channels":          "list_channels_ace.sh",
		"check_ready":            "check_ace_ready.sh",
		"feishu_bot_creator":     "feishu_bot_creator_ace.sh",
		"init_smh_env":           "init_smh_env.sh",
		"remove_smh_env":         "remove_smh_env.sh",
		"set_smh_token":          "set_smh_token.sh",
		"get_version_info":       "get_version_info_ace.sh",
		"list_skills":            "list_skills_ace.sh",
		"set_env":                "set_env_ace.sh",
		"get_env":                "get_env_ace.sh",
		"check_service":          "check_service_ace.sh",
		"detect_install":         "detect_ace_install.sh",
	}

	for feature, want := range supported {
		t.Run("ace-supported-"+feature, func(t *testing.T) {
			got, err := ResolveScript(context.Background(), feature, "lightclawace")
			if err != nil {
				t.Fatalf("ResolveScript(context.Background(), %q, ace) unexpected err=%v", feature, err)
			}
			if got != want {
				t.Errorf("ResolveScript(context.Background(), %q, ace)=%q, want %q", feature, got, want)
			}
		})
	}

	// ACE 不支持 qq_bot_creator
	for _, feature := range []string{"qq_bot_creator"} {
		t.Run("ace-unsupported-"+feature, func(t *testing.T) {
			if _, err := ResolveScript(context.Background(), feature, "lightclawace"); err == nil {
				t.Errorf("ResolveScript(context.Background(), %q, ace) want error, got nil", feature)
			}
		})
	}
}

// TestResolveScript_ErrorMessageSemantics 验证 error 类型的语义：
//   - 未知 feature 错误应包含 "unknown feature"
//   - 不支持该 agent_type 错误应包含 "not supported"
func TestResolveScript_ErrorMessageSemantics(t *testing.T) {
	ctx := context.Background()

	_, err := ResolveScript(context.Background(), "totally_unknown_feature_xxx", "openclaw")
	wanted := hcommon.I18nError(i18n.MsgUnknownFeature, "totally_unknown_feature_xxx")
	if !errors.Is(err, wanted) {
		t.Errorf("expected '%s' error, got %s", wanted.ErrorMessage(ctx), hcommon.ErrorMessageWithCtx(ctx, err))
	}

	_, err = ResolveScript(context.Background(), "qq_bot_creator", "hermes")
	wanted = hcommon.I18nError(i18n.MsgFeatureNotSupportedForAgentType, "qq_bot_creator", "hermes")
	if !errors.Is(err, wanted) {
		t.Errorf("expected '%s' error, got %s", wanted.ErrorMessage(ctx), hcommon.ErrorMessageWithCtx(ctx, err))
	}

	// 未知 agentType 也走 not supported 分支
	_, err = ResolveScript(context.Background(), "set_model", "totally_future_type")
	wanted = hcommon.I18nError(i18n.MsgFeatureNotSupportedForAgentType, "set_model", "totally_future_type")
	if !errors.Is(err, wanted) {
		t.Errorf("expected '%s' error for unknown agent, got %s", wanted.ErrorMessage(ctx), hcommon.ErrorMessageWithCtx(ctx, err))
	}
}

// TestExpandIncludes_EmptyContent 空内容应直接返回空内容，不触发 loader。
func TestExpandIncludes_EmptyContent(t *testing.T) {
	got, err := ExpandIncludes("", func(name string) (string, error) {
		t.Fatalf("loader should not be called for empty content")
		return "", nil
	})
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// TestExpandIncludes_DirectiveNotAtLineStart 非行首的 `# %INCLUDE% ...` 不应被展开（regex 约束）。
func TestExpandIncludes_DirectiveNotAtLineStart(t *testing.T) {
	// include 指令前有空格 → 不匹配 multiline ^
	content := "#!/bin/bash\n    # %INCLUDE% lib_qr_render.sh\necho x\n"
	got, err := ExpandIncludes(content, func(name string) (string, error) {
		t.Fatalf("loader should not be called; include is indented")
		return "", nil
	})
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	if got != content {
		t.Errorf("indented directive should not be expanded, got=%q", got)
	}
}

// TestExpandIncludes_WithCarriageReturn 验证指令末尾允许 \s* 空白。
func TestExpandIncludes_WithTrailingSpaces(t *testing.T) {
	called := false
	loader := func(name string) (string, error) {
		called = true
		return "body", nil
	}
	// 注意正则是 `^# %INCLUDE% (\S+)\s*$`：末尾允许任意空白
	content := "# %INCLUDE% lib_foo.sh   \n"
	got, err := ExpandIncludes(content, loader)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !called {
		t.Fatal("loader should be called for valid include directive")
	}
	if !strings.Contains(got, "body") {
		t.Errorf("body should be inlined, got=%q", got)
	}
}

// TestExpandIncludes_DepthLimitExactlyFour 深度限制为 > 4 触发错误；4 层嵌套应允许。
func TestExpandIncludes_DepthLimitExactlyFour(t *testing.T) {
	// 构造 4 层链式依赖：root -> lib_a -> lib_b -> lib_c -> lib_d (leaf)
	loader := func(name string) (string, error) {
		switch name {
		case "lib_a.sh":
			return "# %INCLUDE% lib_b.sh\n", nil
		case "lib_b.sh":
			return "# %INCLUDE% lib_c.sh\n", nil
		case "lib_c.sh":
			return "# %INCLUDE% lib_d.sh\n", nil
		case "lib_d.sh":
			return "LEAF\n", nil
		}
		return "", fmt.Errorf("not found: %s", name)
	}

	content := "# %INCLUDE% lib_a.sh\n"
	got, err := ExpandIncludes(content, loader)
	if err != nil {
		t.Fatalf("4 层嵌套不应触发深度限制，实际 err=%v", err)
	}
	if !strings.Contains(got, "LEAF") {
		t.Errorf("最内层 LEAF 标识应被内联，got=%q", got)
	}
}
