package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupCustomAgentTypeTestDB(t *testing.T) func() {
	t.Helper()

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := testDB.AutoMigrate(&CustomAgentType{}, &Instance{}, &AIImage{}, &SiteConfig{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	restore := UseDBForTest(testDB)
	return restore
}

func TestCustomAgentTypeCreateValidation(t *testing.T) {
	cleanup := setupCustomAgentTypeTestDB(t)
	defer cleanup()

	if _, err := CreateCustomAgentType(context.Background(), "", ""); err == nil {
		t.Fatal("empty name should fail")
	}
	if _, err := CreateCustomAgentType(context.Background(), "abcdefghijklmnopqrstu", ""); err == nil {
		t.Fatal("name longer than 20 runes should fail")
	}
	if _, err := CreateCustomAgentType(context.Background(), " custom-a", ""); err == nil {
		t.Fatal("name with leading spaces should fail")
	}
	if _, err := CreateCustomAgentType(context.Background(), "custom-a ", ""); err == nil {
		t.Fatal("name with trailing spaces should fail")
	}
	if _, err := CreateCustomAgentType(context.Background(), AgentTypeOpenClaw, ""); err == nil {
		t.Fatal("builtin name should fail")
	}
	if _, err := CreateCustomAgentType(context.Background(), "custom-a", "bad-runtime"); err == nil {
		t.Fatal("invalid compatible_with should fail")
	}
	if _, err := CreateCustomAgentType(context.Background(), "custom-a", " openclaw"); err == nil {
		t.Fatal("compatible_with with leading spaces should fail")
	}
	if _, err := CreateCustomAgentType(context.Background(), "custom-a", AgentTypeOpenClaw); err != nil {
		t.Fatalf("create custom agent type: %v", err)
	}
	if _, err := CreateCustomAgentType(context.Background(), "custom-a", AgentTypeHermes); err == nil {
		t.Fatal("duplicate name should fail")
	}
}

func TestCustomAgentTypeCapabilityAndRuntime(t *testing.T) {
	cleanup := setupCustomAgentTypeTestDB(t)
	defer cleanup()

	if _, err := CreateCustomAgentType(context.Background(), "oc-custom", AgentTypeOpenClaw); err != nil {
		t.Fatalf("create openclaw-compatible type: %v", err)
	}
	if _, err := CreateCustomAgentType(context.Background(), "minimal-custom", ""); err != nil {
		t.Fatalf("create minimal type: %v", err)
	}

	oc := GetAgentTypeByCode(context.Background(), "oc-custom")
	if oc == nil {
		t.Fatal("expected oc-custom agent type")
	}
	if oc.IsBuiltin || oc.CompatibleWith != AgentTypeOpenClaw {
		t.Fatalf("unexpected custom metadata: %+v", oc)
	}
	if !AgentTypeSupportsPlugin(context.Background(), "oc-custom") || !AgentTypeSupportsModel(context.Background(), "oc-custom") {
		t.Fatalf("openclaw-compatible custom type should inherit openclaw capabilities: %+v", oc)
	}
	if got := GetAgentRuntimeType(context.Background(), "oc-custom"); got != AgentTypeOpenClaw {
		t.Fatalf("runtime type = %q, want %q", got, AgentTypeOpenClaw)
	}
	if !AgentTypeSupportsMCP(context.Background(), "oc-custom") {
		t.Fatal("openclaw-compatible custom type should support MCP")
	}
	if !AgentTypeChannelAllowed(context.Background(), "oc-custom", "wecom_app") {
		t.Fatal("openclaw-compatible custom type should inherit openclaw channel whitelist")
	}

	minimal := GetAgentTypeByCode(context.Background(), "minimal-custom")
	if minimal == nil {
		t.Fatal("expected minimal-custom agent type")
	}
	if AgentTypeSupportsModel(context.Background(), "minimal-custom") || AgentTypeSupportsChannel(context.Background(), "minimal-custom") || AgentTypeSupportsPlugin(context.Background(), "minimal-custom") {
		t.Fatalf("minimal custom type should not support agent-specific features: %+v", minimal)
	}
	if got := GetAgentRuntimeType(context.Background(), "minimal-custom"); got != "" {
		t.Fatalf("minimal runtime type = %q, want empty", got)
	}
}

func TestCustomAgentTypeDeleteGuards(t *testing.T) {
	cleanup := setupCustomAgentTypeTestDB(t)
	defer cleanup()

	if _, err := CreateCustomAgentType(context.Background(), "delete-me", ""); err != nil {
		t.Fatalf("create custom type: %v", err)
	}
	if err := DeleteCustomAgentType(context.Background(), AgentTypeOpenClaw); err == nil {
		t.Fatal("deleting builtin type should fail")
	}
	if err := DB(context.Background()).Create(&AIImage{ImageId: "img-enabled", AgentType: "delete-me", Enabled: true}).Error; err != nil {
		t.Fatalf("create enabled image: %v", err)
	}
	if err := DeleteCustomAgentType(context.Background(), "delete-me"); err == nil {
		t.Fatal("deleting enabled type with enabled image should fail")
	}
	if err := DB(context.Background()).Model(&AIImage{}).Where("image_id = ?", "img-enabled").Update("enabled", false).Error; err != nil {
		t.Fatalf("disable image: %v", err)
	}

	if err := DB(context.Background()).Create(&SiteConfig{DefaultAgentType: "delete-me"}).Error; err != nil {
		t.Fatalf("create site config: %v", err)
	}
	if err := DeleteCustomAgentType(context.Background(), "delete-me"); err == nil {
		t.Fatal("deleting default type should fail")
	}
	if err := DB(context.Background()).Where("default_agent_type = ?", "delete-me").Delete(&SiteConfig{}).Error; err != nil {
		t.Fatalf("delete site config: %v", err)
	}
	if err := DB(context.Background()).Create(&SiteConfig{DefaultAgentType: AgentTypeOpenClaw}).Error; err != nil {
		t.Fatalf("create site config: %v", err)
	}

	if err := DB(context.Background()).Create(&Instance{Name: "inst", AgentType: "delete-me"}).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if err := DeleteCustomAgentType(context.Background(), "delete-me"); err == nil {
		t.Fatal("deleting type with instances should fail")
	}
	if err := DB(context.Background()).Where("agent_type = ?", "delete-me").Delete(&Instance{}).Error; err != nil {
		t.Fatalf("delete instance: %v", err)
	}
	if err := DB(context.Background()).Create(&AIImage{ImageId: "img-custom", AgentType: "delete-me", Enabled: false}).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}
	if err := DeleteCustomAgentType(context.Background(), "delete-me"); err != nil {
		t.Fatalf("delete disabled custom type: %v", err)
	}
	var imageCount int64
	if err := DB(context.Background()).Model(&AIImage{}).Where("agent_type = ?", "delete-me").Count(&imageCount).Error; err != nil {
		t.Fatalf("count images: %v", err)
	}
	if imageCount != 0 {
		t.Fatalf("images should be automatically deleted, got %d", imageCount)
	}
	if IsCustomAgentType(context.Background(), "delete-me") {
		t.Fatal("deleted custom type should not be valid")
	}
	if _, err := CreateCustomAgentType(context.Background(), "delete-me", AgentTypeOpenClaw); err != nil {
		t.Fatalf("deleted custom type should be reusable: %v", err)
	}
}

func TestCustomAgentTypeSupportedLists(t *testing.T) {
	cleanup := setupCustomAgentTypeTestDB(t)
	defer cleanup()

	if _, err := CreateCustomAgentType(context.Background(), "oc-custom", AgentTypeOpenClaw); err != nil {
		t.Fatalf("create custom type: %v", err)
	}
	plugins := GetPluginSupportedAgentTypes(context.Background())
	channels := SupportedAgentTypesByChannel(context.Background(), "wecom_app")
	mcpTypes := GetMCPSupportedAgentTypes(context.Background())
	for name, list := range map[string][]string{"plugins": plugins, "channels": channels, "mcp": mcpTypes} {
		found := false
		for _, code := range list {
			if code == "oc-custom" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s list should contain oc-custom, got %v", name, list)
		}
	}
}

// TestCustomAgentTypeImageNoLegacyFallback 验证自定义 Agent 类型查不到精确匹配镜像时，
// 不回退到空 agent_type 的 legacy 镜像；内置类型仍保持原有 legacy fallback 行为。
func TestCustomAgentTypeImageNoLegacyFallback(t *testing.T) {
	cleanup := setupCustomAgentTypeTestDB(t)
	defer cleanup()

	if _, err := CreateCustomAgentType(context.Background(), "oc-custom", AgentTypeOpenClaw); err != nil {
		t.Fatalf("create custom type: %v", err)
	}
	if err := DB(context.Background()).Create(&AIImage{ImageId: "img-legacy", Enabled: true}).Error; err != nil {
		t.Fatalf("create legacy image: %v", err)
	}

	// 内置类型应能回退到 legacy 镜像。
	if img, err := GetEnabledImageByType(context.Background(), AgentTypeOpenClaw); err != nil || img == nil {
		t.Fatalf("openclaw should fall back to legacy image, got img=%v err=%v", img, err)
	}

	// 自定义类型不回退 legacy。
	if img, err := GetEnabledImageByType(context.Background(), "oc-custom"); err != nil || img != nil {
		t.Fatalf("custom type should not fall back to legacy image, got img=%v err=%v", img, err)
	}
}

func TestGetInstanceCountsByType(t *testing.T) {
	cleanup := setupCustomAgentTypeTestDB(t)
	defer cleanup()

	instances := []Instance{
		{Name: "legacy", AgentType: ""},
		{Name: "openclaw", AgentType: AgentTypeOpenClaw},
		{Name: "hermes", AgentType: AgentTypeHermes},
	}
	for _, inst := range instances {
		if err := DB(context.Background()).Create(&inst).Error; err != nil {
			t.Fatalf("create instance: %v", err)
		}
	}
	if err := DB(context.Background()).Where("name = ?", "hermes").Delete(&Instance{}).Error; err != nil {
		t.Fatalf("delete hermes instance: %v", err)
	}

	counts, err := GetInstanceCountsByType(context.Background())
	if err != nil {
		t.Fatalf("GetInstanceCountsByType: %v", err)
	}
	if counts[AgentTypeOpenClaw] != 2 {
		t.Fatalf("openclaw count = %d, want 2", counts[AgentTypeOpenClaw])
	}
	if counts[AgentTypeHermes] != 0 {
		t.Fatalf("hermes count = %d, want 0", counts[AgentTypeHermes])
	}
}

func TestCustomAgentTypeDeleteDBErrorBranches(t *testing.T) {
	cleanup := setupCustomAgentTypeTestDB(t)
	defer cleanup()
	if _, err := CreateCustomAgentType(context.Background(), "broken", ""); err != nil {
		t.Fatalf("create custom type: %v", err)
	}
	if err := DB(context.Background()).Migrator().DropTable(&AIImage{}); err != nil {
		t.Fatalf("drop ai_images: %v", err)
	}
	if err := DeleteCustomAgentType(context.Background(), "broken"); err == nil {
		t.Fatal("expected enabled image count error")
	}
}

func TestDeleteCustomAgentTypeRemovesDisabledEntry(t *testing.T) {
	cleanup := setupCustomAgentTypeTestDB(t)
	defer cleanup()
	if _, err := CreateCustomAgentType(context.Background(), "disabled-custom", ""); err != nil {
		t.Fatalf("create custom type: %v", err)
	}
	if err := DB(context.Background()).Create(&SiteConfig{DefaultAgentType: AgentTypeOpenClaw, DisabledAgentTypes: `["disabled-custom","hermes"]`}).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}
	if err := DeleteCustomAgentType(context.Background(), "disabled-custom"); err != nil {
		t.Fatalf("delete custom type: %v", err)
	}
	config := GetSiteConfig(context.Background())
	if config.DisabledAgentTypes != `["hermes"]` {
		t.Fatalf("disabled_agent_types = %s, want [\"hermes\"]", config.DisabledAgentTypes)
	}
}

func TestCustomAgentTypeDeleteImageCleanupError(t *testing.T) {
	cleanup := setupCustomAgentTypeTestDB(t)
	defer cleanup()
	if _, err := CreateCustomAgentType(context.Background(), "broken-cleanup", ""); err != nil {
		t.Fatalf("create custom type: %v", err)
	}
	if err := DB(context.Background()).Create(&SiteConfig{DefaultAgentType: AgentTypeOpenClaw, DisabledAgentTypes: `["broken-cleanup"]`}).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}
	if err := DB(context.Background()).Migrator().DropTable(&AIImage{}); err != nil {
		t.Fatalf("drop ai_images: %v", err)
	}
	if err := DeleteCustomAgentType(context.Background(), "broken-cleanup"); err == nil {
		t.Fatal("expected image cleanup error")
	}
}

func TestGetInstanceCountsByTypeError(t *testing.T) {
	cleanup := setupCustomAgentTypeTestDB(t)
	defer cleanup()
	if err := DB(context.Background()).Migrator().DropTable(&Instance{}); err != nil {
		t.Fatalf("drop instances: %v", err)
	}
	if _, err := GetInstanceCountsByType(context.Background()); err == nil {
		t.Fatal("expected instance count error")
	}
}
