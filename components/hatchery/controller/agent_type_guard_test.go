package controller

import (
	"context"
	"testing"

	hcommon "hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initAgentTypeGuardTestDB 初始化 agent_type_guard 测试所需的最小 DB（仅 CustomAgentType 表）。
func initAgentTypeGuardTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.AutoMigrate(&model.CustomAgentType{})
	t.Cleanup(model.UseDBForTest(db))
}

func TestCheckAgentTypeValid(t *testing.T) {
	initAgentTypeGuardTestDB(t)

	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"empty string allowed", "", false},
		{"valid openclaw", "openclaw", false},
		{"valid hermes", "hermes", false},
		{"valid lightclawace", "lightclawace", false},
		{"invalid type", "invalid_type", true},
		{"invalid type with special chars", "open-claw", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkAgentTypeValid(context.Background(), tt.agentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkAgentTypeValid(context.Background(), %q) error = %v, wantErr %v", tt.agentType, err, tt.wantErr)
			}
		})
	}
}

func TestCheckAgentVersionValid(t *testing.T) {
	initAgentTypeGuardTestDB(t)
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"empty version allowed", "", false},
		{"valid date version", "2026.1.1", false},
		{"valid semver", "1.0.0", false},
		{"valid date with month", "2026.12.31", false},
		{"valid short alphanumeric", "abc", false}, // 正则允许纯字母数字
		{"invalid format with dash prefix", "-1.0.0", true},
		{"invalid format with dash suffix", "1.0.0-", true},
		{"invalid format too long", "a" + string(make([]byte, 64)), true}, // 超过 64 字符
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkAgentVersionValid(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkAgentVersionValid(%q) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
		})
	}
}

func TestCheckInstanceSupportsDetailConfig(t *testing.T) {
	initAgentTypeGuardTestDB(t)
	ctx := context.Background()

	// v7：hermes/lightclawace 放开 Model/Channel/Skill 后，DetailConfig 返回 true
	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"openclaw supports detail config", "openclaw", false},
		{"empty string supports (legacy)", "", false},
		{"hermes supports (v7)", "hermes", false},
		{"lightclawace supports (v7)", "lightclawace", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &model.Instance{AgentType: tt.agentType}
			err := checkInstanceSupportsDetailConfig(ctx, instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkInstanceSupportsDetailConfig() agentType=%q error = %v, wantErr %v",
					tt.agentType, err, tt.wantErr)
			}
		})
	}
}

func TestCheckInstanceSupportsChatbot(t *testing.T) {
	initAgentTypeGuardTestDB(t)
	ctx := context.Background()

	// final §3.2：ACE Chatbot 放开；Hermes 仍拦截
	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"openclaw supports chatbot", "openclaw", false},
		{"empty string supports (legacy)", "", false},
		{"hermes not supports chatbot", "hermes", true},
		{"lightclawace supports chatbot", "lightclawace", false}, // final：ACE 放开
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &model.Instance{AgentType: tt.agentType}
			err := checkInstanceSupportsChatbot(ctx, instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkInstanceSupportsChatbot() agentType=%q error = %v, wantErr %v",
					tt.agentType, err, tt.wantErr)
			}
		})
	}
}

func TestCheckInstanceSupportsPlugin(t *testing.T) {
	initAgentTypeGuardTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"openclaw supports plugin", "openclaw", false},
		{"empty string supports (legacy openclaw)", "", false},
		{"hermes not supports plugin", "hermes", true},
		{"lightclawace not supports plugin", "lightclawace", true},
		{"unknown type not supports", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &model.Instance{AgentType: tt.agentType}
			err := checkInstanceSupportsPlugin(ctx, instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkInstanceSupportsPlugin() agentType=%q error = %v, wantErr %v",
					tt.agentType, err, tt.wantErr)
			}
		})
	}
}

func TestCheckInstanceSupportsSkill(t *testing.T) {
	initAgentTypeGuardTestDB(t)
	ctx := context.Background()

	// v7：Skill 矩阵放开
	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"openclaw supports skill", "openclaw", false},
		{"empty string supports (legacy openclaw)", "", false},
		{"hermes supports skill (v7)", "hermes", false},
		{"lightclawace supports skill (v7)", "lightclawace", false},
		{"unknown type not supports", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &model.Instance{AgentType: tt.agentType}
			err := checkInstanceSupportsSkill(ctx, instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkInstanceSupportsSkill() agentType=%q error = %v, wantErr %v",
					tt.agentType, err, tt.wantErr)
			}
		})
	}
}

func TestCheckInstanceSupportsModel(t *testing.T) {
	initAgentTypeGuardTestDB(t)
	ctx := context.Background()

	// v7：Model 矩阵放开
	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"openclaw supports model", "openclaw", false},
		{"empty string supports (legacy openclaw)", "", false},
		{"hermes supports model (v7)", "hermes", false},
		{"lightclawace supports model (v7)", "lightclawace", false},
		{"unknown type not supports", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &model.Instance{AgentType: tt.agentType}
			err := checkInstanceSupportsModel(ctx, instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkInstanceSupportsModel() agentType=%q error = %v, wantErr %v",
					tt.agentType, err, tt.wantErr)
			}
		})
	}
}

func TestCheckInstanceSupportsChannel(t *testing.T) {
	initAgentTypeGuardTestDB(t)
	ctx := context.Background()

	// v7：Channel 矩阵放开（具体 channel 由白名单控制，不在此 guard 层判断）
	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"openclaw supports channel", "openclaw", false},
		{"empty string supports (legacy openclaw)", "", false},
		{"hermes supports channel (v7)", "hermes", false},
		{"lightclawace supports channel (v7)", "lightclawace", false},
		{"unknown type not supports", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &model.Instance{AgentType: tt.agentType}
			err := checkInstanceSupportsChannel(ctx, instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkInstanceSupportsChannel() agentType=%q error = %v, wantErr %v",
					tt.agentType, err, tt.wantErr)
			}
		})
	}
}

func TestCheckInstanceSupportsMemory(t *testing.T) {
	initAgentTypeGuardTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name      string
		agentType string
		wantErr   bool
	}{
		{"openclaw supports memory", "openclaw", false},
		{"empty string supports (legacy openclaw)", "", false},
		{"hermes supports memory", "hermes", false},
		{"lightclawace not supports memory", "lightclawace", true},
		{"unknown type not supports", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &model.Instance{AgentType: tt.agentType}
			err := checkInstanceSupportsMemory(ctx, instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkInstanceSupportsMemory() agentType=%q error = %v, wantErr %v",
					tt.agentType, err, tt.wantErr)
			}
		})
	}

	t.Run("nil instance returns nil", func(t *testing.T) {
		if err := checkInstanceSupportsMemory(ctx, nil); err != nil {
			t.Errorf("checkInstanceSupportsMemory(nil) should return nil, got %v", err)
		}
	})
}

func TestCheckGuardErrorMessages(t *testing.T) {
	initAgentTypeGuardTestDB(t)
	ctx := context.Background()
	instance := &model.Instance{AgentType: "hermes"}

	// v7：Plugin 保持拦截 hermes（未放开），测试其错误信息含类型名
	err := checkInstanceSupportsPlugin(ctx, instance)
	if err == nil {
		t.Fatal("expected error for hermes plugin check")
	}
	if !guardContainsStr(err.Error(), "Hermes") {
		t.Errorf("error should contain type display name 'Hermes', got: %s", err.Error())
	}

	// v7：Chatbot 保持拦截 hermes（未放开）
	err = checkInstanceSupportsChatbot(ctx, instance)
	if err == nil {
		t.Fatal("expected error for hermes chatbot check")
	}
	if !guardContainsStr(err.Error(), "Hermes") {
		t.Errorf("error should contain type display name 'Hermes', got: %s", err.Error())
	}
}

func TestCheckGuardNilInstance(t *testing.T) {
	ctx := context.Background()

	// 所有 guard 函数传入 nil instance 应返回 nil error
	tests := []struct {
		name  string
		check func(context.Context, *model.Instance) *hcommon.RichError
	}{
		{"DetailConfig", checkInstanceSupportsDetailConfig},
		{"Plugin", checkInstanceSupportsPlugin},
		{"Skill", checkInstanceSupportsSkill},
		{"Model", checkInstanceSupportsModel},
		{"Channel", checkInstanceSupportsChannel},
		{"Chatbot", checkInstanceSupportsChatbot},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.check(ctx, nil)
			if err != nil {
				t.Errorf("check%s(nil) should return nil, got %v", tt.name, err)
			}
		})
	}
}

func guardContainsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
