package controller

import (
	"context"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSafeRuntimeUserForEnv(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"root", "root", "root"},
		{"lowercase", "lightclaw", "lightclaw"},
		{"digits", "user123", "user123"},
		{"underscore", "claw_user", "claw_user"},
		{"hyphen_mid", "claw-user", "claw-user"},
		{"mixed_case", "LightClaw", "LightClaw"},
		{"max_len_32", strings.Repeat("a", 32), strings.Repeat("a", 32)},
		{"empty", "", ""},
		{"too_long_33", strings.Repeat("a", 33), ""},
		{"leading_hyphen", "-rf", ""},
		{"space", "user name", ""},
		{"semicolon", "user;rm -rf /", ""},
		{"backtick", "user`whoami`", ""},
		{"dollar", "user$USER", ""},
		{"paren", "user$(whoami)", ""},
		{"pipe", "user|cat", ""},
		{"newline", "user\nls", ""},
		{"dot", "user.name", ""},
		{"slash", "user/path", ""},
		{"quote_single", "user'", ""},
		{"quote_double", "user\"", ""},
		{"equal", "user=1", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := safeRuntimeUserForEnv(c.input)
			if got != c.want {
				t.Errorf("safeRuntimeUserForEnv(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

func TestGetEffectiveRuntimeUser_NonEmpty(t *testing.T) {
	if got := getEffectiveRuntimeUser("ubuntu"); got != "ubuntu" {
		t.Errorf("got %q, want ubuntu", got)
	}
}

func TestGetEffectiveRuntimeUser_Empty(t *testing.T) {
	if got := getEffectiveRuntimeUser(""); got != "root" {
		t.Errorf("got %q, want root (fallback)", got)
	}
}

func TestHomeForUser_Root(t *testing.T) {
	if got := homeForUser("root"); got != "/root" {
		t.Errorf("got %q, want /root", got)
	}
}

func TestHomeForUser_Regular(t *testing.T) {
	if got := homeForUser("ubuntu"); got != "/home/ubuntu" {
		t.Errorf("got %q, want /home/ubuntu", got)
	}
}

func TestHomeForUser_Custom(t *testing.T) {
	if got := homeForUser("deploy"); got != "/home/deploy" {
		t.Errorf("got %q, want /home/deploy", got)
	}
}

func TestGetDefaultTATRunIdentity_WithUser(t *testing.T) {
	user, workdir := getDefaultTATRunIdentity("ubuntu")
	if user != "ubuntu" {
		t.Errorf("user = %q, want ubuntu", user)
	}
	if workdir != "/home/ubuntu" {
		t.Errorf("workdir = %q, want /home/ubuntu", workdir)
	}
}

func TestGetDefaultTATRunIdentity_FallbackRoot(t *testing.T) {
	user, workdir := getDefaultTATRunIdentity("")
	if user != "root" {
		t.Errorf("user = %q, want root", user)
	}
	if workdir != "/root" {
		t.Errorf("workdir = %q, want /root", workdir)
	}
}

func TestGetTATRunIdentity_NormalScript(t *testing.T) {
	user, workdir := getTATRunIdentity("memory_tdai_switch_free.sh", "ubuntu")
	if user != "ubuntu" {
		t.Errorf("user = %q, want ubuntu", user)
	}
	if workdir != "/home/ubuntu" {
		t.Errorf("workdir = %q, want /home/ubuntu", workdir)
	}
}

func TestGetTATRunIdentity_RootRequiredScript(t *testing.T) {
	user, workdir := getTATRunIdentity("cls_agent_installer.sh", "ubuntu")
	if user != "root" {
		t.Errorf("user = %q, want root (forced)", user)
	}
	if workdir != "/root" {
		t.Errorf("workdir = %q, want /root (forced)", workdir)
	}
}

func TestGetTATRunIdentity_RootRequiredWithPath(t *testing.T) {
	user, _ := getTATRunIdentity("/some/path/restore_post_reinstall.sh", "deploy")
	if user != "root" {
		t.Errorf("user = %q, want root (path prefix should be stripped)", user)
	}
}

func TestGetTATRunIdentity_EmptyUser(t *testing.T) {
	user, workdir := getTATRunIdentity("memory_tdai_disable.sh", "")
	if user != "root" {
		t.Errorf("user = %q, want root (fallback)", user)
	}
	if workdir != "/root" {
		t.Errorf("workdir = %q, want /root", workdir)
	}
}

func TestGetTATRunIdentity_ProScript(t *testing.T) {
	user, workdir := getTATRunIdentity("memory_tdai_switch_pro.sh", "openclaw")
	if user != "openclaw" {
		t.Errorf("user = %q, want openclaw", user)
	}
	if workdir != "/home/openclaw" {
		t.Errorf("workdir = %q, want /home/openclaw", workdir)
	}
}

func TestGetTATRunIdentity_RootRequiredWhitelist(t *testing.T) {
	rootScripts := []string{
		"cls_agent_installer.sh",
		"cls_agent_uninstaller.sh",
		"restore_post_reinstall.sh",
	}

	for _, s := range rootScripts {
		t.Run(s, func(t *testing.T) {
			user, workdir := getTATRunIdentity(s, "lightclaw")
			if user != "root" {
				t.Errorf("script %s should force runUser=root, got %s", s, user)
			}
			if workdir != "/root" {
				t.Errorf("script %s should force workdir=/root, got %s", s, workdir)
			}
		})
	}
}

func TestGetTATRunIdentity_NonWhitelistKeepsRuntimeUser(t *testing.T) {
	user, workdir := getTATRunIdentity("list_channels.sh", "lightclaw")
	if user != "lightclaw" {
		t.Errorf("non-whitelist script should use runtimeUser, got %s", user)
	}
	if workdir != "/home/lightclaw" {
		t.Errorf("non-whitelist script workdir should be /home/lightclaw, got %s", workdir)
	}

	user2, workdir2 := getTATRunIdentity("list_channels.sh", "")
	if user2 != "root" || workdir2 != "/root" {
		t.Errorf("empty runtimeUser should fallback to root, got (%s, %s)", user2, workdir2)
	}
}

func TestLookupRuntimeUser_Found(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.CustomAgentType{}, &model.Instance{})
	t.Cleanup(model.UseDBForTest(db))
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-lu-001", RuntimeUser: "ubuntu"})

	got := LookupRuntimeUser(context.Background(), "ins-lu-001")
	if got != "ubuntu" {
		t.Errorf("got %q, want ubuntu", got)
	}
}

func TestLookupRuntimeUser_EmptyRuntimeUser(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.CustomAgentType{}, &model.Instance{})
	t.Cleanup(model.UseDBForTest(db))
	model.DB(context.Background()).Create(&model.Instance{InstanceId: "ins-lu-002", RuntimeUser: ""})

	got := LookupRuntimeUser(context.Background(), "ins-lu-002")
	if got != "root" {
		t.Errorf("got %q, want root (fallback)", got)
	}
}

func TestLookupRuntimeUser_NotFound(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.CustomAgentType{}, &model.Instance{})
	t.Cleanup(model.UseDBForTest(db))
	got := LookupRuntimeUser(context.Background(), "ins-nonexist")
	if got != "root" {
		t.Errorf("got %q, want root (fallback for not found)", got)
	}
}

func TestRootRequiredScripts(t *testing.T) {
	expected := []string{
		"cls_agent_installer.sh",
		"cls_agent_uninstaller.sh",
		"restore_post_reinstall.sh",
	}
	for _, script := range expected {
		if _, ok := rootRequiredTATScripts[script]; !ok {
			t.Errorf("%q should be in rootRequiredTATScripts", script)
		}
	}
}
