package model

import (
	"context"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupAIImageTestDB creates a temporary SQLite database for AI image testing.
func setupAIImageTestDB(t *testing.T) (cleanup func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "ai_image_test_*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()

	dsn := tmpFile.Name() + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("open test db: %v", err)
	}

	origDB := gdb
	gdb = testDB

	if err := gdb.AutoMigrate(&AIImage{}, &CustomAgentType{}); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("auto migrate: %v", err)
	}

	return func() {
		sqlDB, _ := gdb.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
		os.Remove(tmpFile.Name() + "-wal")
		os.Remove(tmpFile.Name() + "-shm")
		gdb = origDB
	}
}

func TestAIImageCanEnableImage(t *testing.T) {
	cleanup := setupAIImageTestDB(t)
	defer cleanup()

	if _, err := CreateCustomAgentType(context.Background(), "my-custom", ""); err != nil {
		t.Fatalf("create custom agent type: %v", err)
	}

	tests := []struct {
		name       string
		image      AIImage
		wantEnable bool
		wantReason string
	}{
		{
			name:       "legacy image (empty type and version) can enable",
			image:      AIImage{AgentType: "", AgentVersion: ""},
			wantEnable: true,
		},
		{
			name:       "valid openclaw image",
			image:      AIImage{AgentType: "openclaw", AgentVersion: "2026.1.1"},
			wantEnable: true,
		},
		{
			name:       "valid hermes image",
			image:      AIImage{AgentType: "hermes", AgentVersion: "1.0.0"},
			wantEnable: true,
		},
		{
			name:       "valid lightclawace image",
			image:      AIImage{AgentType: "lightclawace", AgentVersion: "2.0.0"},
			wantEnable: true,
		},
		{
			name:       "invalid agent type",
			image:      AIImage{AgentType: "invalid_type", AgentVersion: "1.0.0"},
			wantEnable: false,
			wantReason: "无效的智能体类型",
		},
		{
			name:       "type set but version empty",
			image:      AIImage{AgentType: "openclaw", AgentVersion: ""},
			wantEnable: false,
			wantReason: "请先设置 Agent 版本后再启用",
		},
		{
			name:       "hermes type but version empty",
			image:      AIImage{AgentType: "hermes", AgentVersion: ""},
			wantEnable: false,
			wantReason: "请先设置 Agent 版本后再启用",
		},
		{
			name:       "custom type with empty version can enable",
			image:      AIImage{AgentType: "my-custom", AgentVersion: ""},
			wantEnable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockErr := tt.image.CanEnableImage(context.Background())
			canEnable := blockErr == nil
			reason := ""
			if blockErr != nil {
				reason = blockErr.Error()
			}
			if canEnable != tt.wantEnable {
				t.Errorf("CanEnableImage() canEnable = %v, want %v", canEnable, tt.wantEnable)
			}
			if !tt.wantEnable && tt.wantReason != "" {
				if !containsString(reason, tt.wantReason) {
					t.Errorf("CanEnableImage() reason = %q, want to contain %q", reason, tt.wantReason)
				}
			}
		})
	}
}

func TestAIImageIsLegacyImage(t *testing.T) {
	cleanup := setupAIImageTestDB(t)
	defer cleanup()

	if _, err := CreateCustomAgentType(context.Background(), "my-custom", ""); err != nil {
		t.Fatalf("create custom agent type: %v", err)
	}

	tests := []struct {
		name       string
		image      AIImage
		wantLegacy bool
	}{
		{
			name:       "empty type and version - legacy",
			image:      AIImage{AgentType: "", AgentVersion: ""},
			wantLegacy: true,
		},
		{
			name:       "type set but version empty - legacy",
			image:      AIImage{AgentType: "openclaw", AgentVersion: ""},
			wantLegacy: true,
		},
		{
			name:       "type empty but version set - legacy",
			image:      AIImage{AgentType: "", AgentVersion: "1.0.0"},
			wantLegacy: true,
		},
		{
			name:       "both type and version set - not legacy",
			image:      AIImage{AgentType: "openclaw", AgentVersion: "2026.1.1"},
			wantLegacy: false,
		},
		{
			name:       "hermes with version - not legacy",
			image:      AIImage{AgentType: "hermes", AgentVersion: "1.0.0"},
			wantLegacy: false,
		},
		{
			name:       "custom type with empty version - not legacy",
			image:      AIImage{AgentType: "my-custom", AgentVersion: ""},
			wantLegacy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isLegacy := tt.image.IsLegacyImage(context.Background())
			if isLegacy != tt.wantLegacy {
				t.Errorf("IsLegacyImage() = %v, want %v", isLegacy, tt.wantLegacy)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ==================== gdb Tests with SQLite Memory ====================

func TestGetEnabledImageByType(t *testing.T) {
	cleanup := setupAIImageTestDB(t)
	defer cleanup()

	// Test 1: No images - should return nil, nil
	img, err := GetEnabledImageByType(context.Background(), "openclaw")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if img != nil {
		t.Errorf("expected nil image, got %+v", img)
	}

	// Test 2: Create enabled image for openclaw
	openclawImg := AIImage{
		ImageId:      "img-openclaw-001",
		ImageName:    "OpenClaw Image",
		AgentType:    "openclaw",
		AgentVersion: "2026.1.1",
		Enabled:      true,
	}
	if err := gdb.Create(&openclawImg).Error; err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	// Should find the enabled openclaw image
	img, err = GetEnabledImageByType(context.Background(), "openclaw")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if img == nil {
		t.Fatalf("expected image, got nil")
	}
	if img.ImageId != "img-openclaw-001" {
		t.Errorf("expected image id %q, got %q", "img-openclaw-001", img.ImageId)
	}

	// Test 3: Create hermes image
	hermesImg := AIImage{
		ImageId:      "img-hermes-001",
		ImageName:    "Hermes Image",
		AgentType:    "hermes",
		AgentVersion: "1.0.0",
		Enabled:      true,
	}
	if err := gdb.Create(&hermesImg).Error; err != nil {
		t.Fatalf("failed to create hermes image: %v", err)
	}

	// Should find hermes image
	img, err = GetEnabledImageByType(context.Background(), "hermes")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if img == nil || img.ImageId != "img-hermes-001" {
		t.Errorf("expected hermes image, got %+v", img)
	}

	// Test 4: Empty type should fallback to openclaw
	img, err = GetEnabledImageByType(context.Background(), "")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if img == nil || img.AgentType != "openclaw" {
		t.Errorf("expected openclaw image for empty type, got %+v", img)
	}

	// Test 5: Fallback to legacy image (empty agent_type)
	// First disable all existing images
	gdb.Model(&AIImage{}).Where("1=1").Update("enabled", false)

	legacyImg := AIImage{
		ImageId:      "img-legacy-001",
		ImageName:    "Legacy Image",
		AgentType:    "", // legacy: empty type
		AgentVersion: "",
		Enabled:      true,
	}
	if err := gdb.Create(&legacyImg).Error; err != nil {
		t.Fatalf("failed to create legacy image: %v", err)
	}

	// 内置类型查不到精确匹配时，应回退到 legacy 空类型镜像。
	img, err = GetEnabledImageByType(context.Background(), "lightclawace")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if img == nil || img.ImageId != "img-legacy-001" {
		t.Errorf("expected legacy image fallback for lightclawace, got %+v", img)
	}
	img, err = GetEnabledImageByType(context.Background(), "openclaw")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if img == nil || img.ImageId != "img-legacy-001" {
		t.Errorf("expected legacy image fallback for openclaw, got %+v", img)
	}
}

func TestGetEnabledImagesMap(t *testing.T) {
	cleanup := setupAIImageTestDB(t)
	defer cleanup()

	// Test 1: Empty gdb
	result, err := GetEnabledImagesMap(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}

	// Test 2: Create multiple enabled images
	images := []AIImage{
		{ImageId: "img-001", ImageName: "OpenClaw", AgentType: "openclaw", AgentVersion: "1.0", Enabled: true},
		{ImageId: "img-002", ImageName: "Hermes", AgentType: "hermes", AgentVersion: "1.0", Enabled: true},
		{ImageId: "img-003", ImageName: "Disabled", AgentType: "lightclawace", AgentVersion: "1.0", Enabled: false}, // disabled
		{ImageId: "img-004", ImageName: "Legacy", AgentType: "", AgentVersion: "", Enabled: true},                   // legacy (empty type)
	}

	for _, img := range images {
		if err := gdb.Create(&img).Error; err != nil {
			t.Fatalf("failed to create image: %v", err)
		}
	}

	result, err = GetEnabledImagesMap(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Should only have openclaw and hermes (legacy has empty type, disabled is not enabled)
	if len(result) != 2 {
		t.Errorf("expected 2 images in map, got %d: %v", len(result), result)
	}
	if _, ok := result["openclaw"]; !ok {
		t.Error("expected openclaw in map")
	}
	if _, ok := result["hermes"]; !ok {
		t.Error("expected hermes in map")
	}
	if _, ok := result["lightclawace"]; ok {
		t.Error("disabled image should not be in map")
	}
}

func TestGetEnabledImageCountByType(t *testing.T) {
	cleanup := setupAIImageTestDB(t)
	defer cleanup()

	// Test 1: No images
	count, err := GetEnabledImageCountByType(context.Background(), "openclaw")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}

	// Test 2: Create images
	images := []AIImage{
		{ImageId: "img-001", AgentType: "openclaw", Enabled: true},
		{ImageId: "img-002", AgentType: "openclaw", Enabled: true},
		{ImageId: "img-003", AgentType: "openclaw", Enabled: false}, // disabled
		{ImageId: "img-004", AgentType: "hermes", Enabled: true},
	}

	for _, img := range images {
		if err := gdb.Create(&img).Error; err != nil {
			t.Fatalf("failed to create image: %v", err)
		}
	}

	// openclaw should have 2 enabled
	count, err = GetEnabledImageCountByType(context.Background(), "openclaw")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	// hermes should have 1
	count, err = GetEnabledImageCountByType(context.Background(), "hermes")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// lightclawace should have 0
	count, err = GetEnabledImageCountByType(context.Background(), "lightclawace")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}
}

func TestGetImageStatsByType(t *testing.T) {
	cleanup := setupAIImageTestDB(t)
	defer cleanup()

	// Test 1: Empty gdb
	stats, err := GetImageStatsByType(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected empty stats, got %v", stats)
	}

	// Test 2: Create images of various types
	images := []AIImage{
		{ImageId: "img-001", AgentType: "openclaw", Enabled: true},
		{ImageId: "img-002", AgentType: "openclaw", Enabled: false},
		{ImageId: "img-003", AgentType: "openclaw", Enabled: true},
		{ImageId: "img-004", AgentType: "hermes", Enabled: true},
		{ImageId: "img-005", AgentType: "lightclawace", Enabled: false},
		{ImageId: "img-006", AgentType: "", Enabled: true}, // legacy
	}

	for _, img := range images {
		if err := gdb.Create(&img).Error; err != nil {
			t.Fatalf("failed to create image: %v", err)
		}
	}

	stats, err = GetImageStatsByType(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Check counts
	if stats["openclaw"] != 3 {
		t.Errorf("expected openclaw count 3, got %d", stats["openclaw"])
	}
	if stats["hermes"] != 1 {
		t.Errorf("expected hermes count 1, got %d", stats["hermes"])
	}
	if stats["lightclawace"] != 1 {
		t.Errorf("expected lightclawace count 1, got %d", stats["lightclawace"])
	}
	if stats[""] != 1 {
		t.Errorf("expected legacy (empty type) count 1, got %d", stats[""])
	}
}

// ==================== GetEnabledImage Tests ====================

func TestGetEnabledImage(t *testing.T) {
	cleanup := setupAIImageTestDB(t)
	defer cleanup()

	// Test 1: No images - should return nil
	img := GetEnabledImage(context.Background())
	if img != nil {
		t.Errorf("expected nil image when no images exist, got %+v", img)
	}

	// Test 2: Only disabled images - should return nil
	disabledImg := AIImage{
		ImageId:   "img-disabled-001",
		ImageName: "Disabled Image",
		Enabled:   false,
	}
	if err := gdb.Create(&disabledImg).Error; err != nil {
		t.Fatalf("failed to create disabled image: %v", err)
	}

	img = GetEnabledImage(context.Background())
	if img != nil {
		t.Errorf("expected nil image when no enabled images, got %+v", img)
	}

	// Test 3: One enabled image - should return it
	enabledImg := AIImage{
		ImageId:      "img-enabled-001",
		ImageName:    "Enabled Image",
		AgentType:    "openclaw",
		AgentVersion: "2026.1.1",
		Enabled:      true,
	}
	if err := gdb.Create(&enabledImg).Error; err != nil {
		t.Fatalf("failed to create enabled image: %v", err)
	}

	img = GetEnabledImage(context.Background())
	if img == nil {
		t.Fatal("expected enabled image, got nil")
	}
	if img.ImageId != "img-enabled-001" {
		t.Errorf("expected image id %q, got %q", "img-enabled-001", img.ImageId)
	}

	// Test 4: Multiple enabled images - should return first one found
	enabledImg2 := AIImage{
		ImageId:      "img-enabled-002",
		ImageName:    "Enabled Image 2",
		AgentType:    "hermes",
		AgentVersion: "1.0.0",
		Enabled:      true,
	}
	if err := gdb.Create(&enabledImg2).Error; err != nil {
		t.Fatalf("failed to create second enabled image: %v", err)
	}

	img = GetEnabledImage(context.Background())
	if img == nil {
		t.Fatal("expected enabled image, got nil")
	}
	// Should return one of the enabled images
	if !img.Enabled {
		t.Error("expected returned image to be enabled")
	}
}

func TestGetEnabledImagesMap_EmptyAgentTypeMapsToOpenclaw(t *testing.T) {
	cleanup := setupAIImageTestDB(t)
	defer cleanup()

	// 创建空 agent_type 的启用镜像（存量数据）
	gdb.Create(&AIImage{ImageId: "img-legacy", ImageName: "Legacy Image", Enabled: true, AgentType: ""})

	result, err := GetEnabledImagesMap(context.Background())
	if err != nil {
		t.Fatalf("GetEnabledImagesMap failed: %v", err)
	}

	// 空 agent_type 应映射为 openclaw
	img, ok := result[AgentTypeOpenClaw]
	if !ok {
		t.Fatal("expected openclaw key in map, got none")
	}
	if img.ImageId != "img-legacy" {
		t.Errorf("expected img-legacy, got %s", img.ImageId)
	}

	// 不应有空 key
	if _, ok := result[""]; ok {
		t.Error("should not have empty string key in map")
	}
}

func TestGetEnabledImagesMap_MultipleTypes(t *testing.T) {
	cleanup := setupAIImageTestDB(t)
	defer cleanup()

	gdb.Create(&AIImage{ImageId: "img-oc", ImageName: "OC Image", Enabled: true, AgentType: "openclaw"})
	gdb.Create(&AIImage{ImageId: "img-hermes", ImageName: "Hermes Image", Enabled: true, AgentType: "hermes"})
	gdb.Create(&AIImage{ImageId: "img-disabled", ImageName: "Disabled", Enabled: false, AgentType: "lightclawace"})

	result, err := GetEnabledImagesMap(context.Background())
	if err != nil {
		t.Fatalf("GetEnabledImagesMap failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 types in map, got %d", len(result))
	}
	if result["openclaw"].ImageId != "img-oc" {
		t.Errorf("openclaw image mismatch: %s", result["openclaw"].ImageId)
	}
	if result["hermes"].ImageId != "img-hermes" {
		t.Errorf("hermes image mismatch: %s", result["hermes"].ImageId)
	}
	if _, ok := result["lightclawace"]; ok {
		t.Error("disabled image should not be in map")
	}
}

func TestGetEnabledImagesMap_Empty(t *testing.T) {
	cleanup := setupAIImageTestDB(t)
	defer cleanup()

	result, err := GetEnabledImagesMap(context.Background())
	if err != nil {
		t.Fatalf("GetEnabledImagesMap failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}
