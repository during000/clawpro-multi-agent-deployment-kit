package controller

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ─── LookupRuntimeUser ─────────────────────────────────────────────────

func TestLookupRuntimeUser_InstanceNotFound(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.CustomAgentType{}, &model.Instance{})
	defer model.UseDBForTest(db)()

	// 查询不存在的实例 → fallback root
	got := LookupRuntimeUser(context.Background(), "ins-nonexistent")
	if got != "root" {
		t.Errorf("实例不存在时应 fallback 为 root，实际=%q", got)
	}
}

func TestLookupRuntimeUser_RuntimeUserEmpty(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.CustomAgentType{}, &model.Instance{})
	defer model.UseDBForTest(db)()

	// 实例存在但 RuntimeUser 为空 → fallback root
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-empty-ru", RuntimeUser: "",
	}
	db.Create(inst)

	got := LookupRuntimeUser(context.Background(), "ins-empty-ru")
	if got != "root" {
		t.Errorf("RuntimeUser 为空应 fallback 为 root，实际=%q", got)
	}
}

func TestLookupRuntimeUser_WithValidUser(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.CustomAgentType{}, &model.Instance{})
	defer model.UseDBForTest(db)()

	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-with-ru",
		RuntimeUser: "agentuser",
	}
	db.Create(inst)

	got := LookupRuntimeUser(context.Background(), "ins-with-ru")
	if got != "agentuser" {
		t.Errorf("应返回 agentuser，实际=%q", got)
	}
}

// ─── LookupAgentType ────────────────────────────────────────────────────

func TestLookupAgentType_InstanceNotFound(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.CustomAgentType{}, &model.Instance{})
	defer model.UseDBForTest(db)()

	got := LookupAgentType(context.Background(), "ins-nonexistent")
	if got != model.AgentTypeOpenClaw {
		t.Errorf("实例不存在时应 fallback 为 openclaw，实际=%q", got)
	}
}

func TestLookupAgentType_EmptyAgentType(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.CustomAgentType{}, &model.Instance{})
	defer model.UseDBForTest(db)()

	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-empty-at", AgentType: "",
	}
	db.Create(inst)

	got := LookupAgentType(context.Background(), "ins-empty-at")
	if got != model.AgentTypeOpenClaw {
		t.Errorf("AgentType 空应 fallback 为 openclaw，实际=%q", got)
	}
}

func TestLookupAgentType_WithValidType(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.CustomAgentType{}, &model.Instance{})
	defer model.UseDBForTest(db)()

	cases := []struct {
		instanceId string
		agentType  string
	}{
		{"ins-hermes", model.AgentTypeHermes},
		{"ins-ace", model.AgentTypeLightclawACE},
		{"ins-openclaw", model.AgentTypeOpenClaw},
	}
	for _, c := range cases {
		db.Create(&model.Instance{
			Name: "i", InstanceId: c.instanceId, AgentType: c.agentType,
		})
	}

	for _, c := range cases {
		got := LookupAgentType(context.Background(), c.instanceId)
		if got != c.agentType {
			t.Errorf("LookupAgentType(context.Background(), %q)=%q, want %q", c.instanceId, got, c.agentType)
		}
	}
}

// ─── getEffectiveRuntimeUser / homeForUser ─────────────────────────────

func TestGetEffectiveRuntimeUser(t *testing.T) {
	// 覆盖三类典型 RuntimeUser：
	//   - root：旧 OpenClaw 官方镜像 / 显式 root
	//   - agentuser：Hermes / ACE 旧镜像（<= v0.0.11）的运行账户
	//   - ubuntu：Hermes 新镜像（>= v0.0.12）的运行账户
	//   - lightclaw：自定义运行账户兜底
	// 不论哪种，函数都应原样回传，仅空串走 root fallback。
	cases := []struct {
		in, want string
	}{
		{"", "root"},
		{"root", "root"},
		{"agentuser", "agentuser"},
		{"ubuntu", "ubuntu"},
		{"lightclaw", "lightclaw"},
	}
	for _, c := range cases {
		got := getEffectiveRuntimeUser(c.in)
		if got != c.want {
			t.Errorf("getEffectiveRuntimeUser(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}

func TestHomeForUser(t *testing.T) {
	// root → /root 特例；其他用户统一映射到 /home/<user>。
	// 这里同时覆盖 agentuser（旧 Hermes/ACE）和 ubuntu（新 Hermes v0.0.12+）。
	cases := []struct {
		in, want string
	}{
		{"root", "/root"},
		{"agentuser", "/home/agentuser"},
		{"ubuntu", "/home/ubuntu"},
		{"lightclaw", "/home/lightclaw"},
	}
	for _, c := range cases {
		got := homeForUser(c.in)
		if got != c.want {
			t.Errorf("homeForUser(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}
