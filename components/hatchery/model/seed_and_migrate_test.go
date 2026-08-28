package model

import (
	"context"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"github.com/glebarez/sqlite"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupSeedMigrateTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	// :memory: 必须单连接，否则并发连接会导致 no such table
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxIdleTime(0)
	sqlDB.SetConnMaxLifetime(0)

	if err := db.AutoMigrate(allModels...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	t.Cleanup(UseDBForTestWithDriver(db, "sqlite"))
}

// TestRunAllSeeds_NilClient 验证 cvmClientFn 为 nil 时不 panic，正常执行其他 Seed。
func TestRunAllSeeds_NilClient(t *testing.T) {
	setupSeedMigrateTestDB(t)
	RunAllSeeds(context.Background(), nil)
}

// TestRunAllSeeds_WithFailingClient 验证 CVM 客户端创建失败时跳过镜像初始化，其他 Seed 正常。
func TestRunAllSeeds_WithFailingClient(t *testing.T) {
	setupSeedMigrateTestDB(t)

	failClient := func(ctx context.Context) (*cvm.Client, error) {
		return nil, hcommon.I18nError(i18n.MsgCreateCVMClientFailed)
	}
	RunAllSeeds(context.Background(), failClient)

	// 验证其他 Seed 正常执行（如 channels 被创建）
	var count int64
	DB(context.Background()).Model(&AIChannel{}).Count(&count)
	if count == 0 {
		t.Error("RunAllSeeds 应创建预置渠道")
	}
}

// TestRunAllSeeds_Idempotent 验证多次调用幂等。
func TestRunAllSeeds_Idempotent(t *testing.T) {
	setupSeedMigrateTestDB(t)

	RunAllSeeds(context.Background(), nil)
	RunAllSeeds(context.Background(), nil)

	var count int64
	DB(context.Background()).Model(&AIChannel{}).Count(&count)
	if count == 0 {
		t.Error("幂等执行后仍应有预置渠道")
	}
}

// TestSeedAvailableImages_FailingClient 验证客户端创建失败时安全返回。
func TestSeedAvailableImages_FailingClient(t *testing.T) {
	setupSeedMigrateTestDB(t)

	failClient := func(ctx context.Context) (*cvm.Client, error) {
		return nil, hcommon.I18nError(i18n.MsgCreateCVMClientFailed)
	}
	SeedAvailableImages(context.Background(), failClient)
}

func TestCandidateAgentVersionFromHistory(t *testing.T) {
	candidate := hcommon.CandidateImage{
		ImageId:      "img-test",
		ImageName:    "Test Image",
		AgentType:    AgentTypeOpenClaw,
		AgentVersion: "2026.1.1",
	}

	agentType, agentVersion := candidateAgentVersionFromHistory(candidate, nil)
	if agentType != AgentTypeOpenClaw || agentVersion != "2026.1.1" {
		t.Fatalf("fallback = (%s,%s), want (%s,2026.1.1)", agentType, agentVersion, AgentTypeOpenClaw)
	}

	agentType, agentVersion = candidateAgentVersionFromHistory(candidate, map[string]ImageHistory{
		"img-test": {ImageId: "img-test", AgentType: AgentTypeOpenClaw, AgentVersion: "2026.5.25"},
	})
	if agentType != AgentTypeOpenClaw || agentVersion != "2026.5.25" {
		t.Fatalf("history override = (%s,%s), want (%s,2026.5.25)", agentType, agentVersion, AgentTypeOpenClaw)
	}
}

func TestEnableDefaultImagesByBuiltinTypesFillsMissingOnly(t *testing.T) {
	setupSeedMigrateTestDB(t)
	const testBuiltinA = "testbuiltin-a"
	const testBuiltinB = "testbuiltin-b"
	agentTypesMap[testBuiltinA] = &AgentType{Code: testBuiltinA, Name: "TestBuiltinA", IsBuiltin: true}
	agentTypesMap[testBuiltinB] = &AgentType{Code: testBuiltinB, Name: "TestBuiltinB", IsBuiltin: true}
	defer delete(agentTypesMap, testBuiltinA)
	defer delete(agentTypesMap, testBuiltinB)

	if err := DB(context.Background()).Create(&SiteConfig{DefaultAgentType: AgentTypeOpenClaw, DisabledAgentTypes: `[]`}).Error; err != nil {
		t.Fatalf("create site config: %v", err)
	}

	images := []AIImage{
		{ImageId: "img-openclaw-legacy", ImageName: "legacy openclaw", AgentType: "", Enabled: true},
		{ImageId: "img-openclaw-new", ImageName: "new openclaw", AgentType: AgentTypeOpenClaw},
		{ImageId: "img-hermes-old", ImageName: "old hermes", AgentType: AgentTypeHermes, Enabled: true},
		{ImageId: "img-hermes-new", ImageName: "new hermes", AgentType: AgentTypeHermes},
		{ImageId: "img-ace", ImageName: "ace", AgentType: AgentTypeLightclawACE},
		{ImageId: "img-test-a", ImageName: "test a", AgentType: testBuiltinA},
		{ImageId: "img-test-b", ImageName: "test b", AgentType: testBuiltinB},
	}
	for _, img := range images {
		if err := DB(context.Background()).Create(&img).Error; err != nil {
			t.Fatalf("create image %s: %v", img.ImageId, err)
		}
	}

	enableDefaultImagesByBuiltinTypes(context.Background(), []AIImage{
		{ImageId: "img-openclaw-new", AgentType: AgentTypeOpenClaw},
		{ImageId: "img-hermes-new", AgentType: AgentTypeHermes},
		{ImageId: "img-ace", AgentType: AgentTypeLightclawACE},
		{ImageId: "img-test-a", AgentType: testBuiltinA},
		{ImageId: "img-test-b", AgentType: testBuiltinB},
	})

	assertEnabled := func(imageID string, want bool) {
		t.Helper()
		var img AIImage
		if err := DB(context.Background()).Where("image_id = ?", imageID).First(&img).Error; err != nil {
			t.Fatalf("query image %s: %v", imageID, err)
		}
		if img.Enabled != want {
			t.Fatalf("image %s enabled=%v, want %v", imageID, img.Enabled, want)
		}
	}
	assertEnabled("img-openclaw-legacy", true)
	assertEnabled("img-openclaw-new", false)
	assertEnabled("img-hermes-old", true)
	assertEnabled("img-hermes-new", false)
	assertEnabled("img-ace", true)
	assertEnabled("img-test-a", true)
	assertEnabled("img-test-b", true)

	disabled := map[string]bool{}
	for _, agentType := range GetDisabledAgentTypes(context.Background()) {
		disabled[agentType] = true
	}
	for _, agentType := range []string{AgentTypeLightclawACE, testBuiltinA, testBuiltinB} {
		if !disabled[agentType] {
			t.Fatalf("auto-enabled missing agent type %s should be disabled, got %v", agentType, disabled)
		}
	}
	if disabled[AgentTypeOpenClaw] || disabled[AgentTypeHermes] {
		t.Fatalf("existing enabled/default types should not be disabled, got %v", disabled)
	}
}

func TestLatestOfficialImageHistories(t *testing.T) {
	setupSeedMigrateTestDB(t)
	candidate := hcommon.CandidateImages[0]
	older := ImageHistory{ImageId: candidate.ImageId, AgentType: candidate.AgentType, AgentVersion: "2026.1.1", PublishedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	newer := ImageHistory{ImageId: candidate.ImageId, AgentType: candidate.AgentType, AgentVersion: "2026.5.25", PublishedAt: time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)}
	deleted := ImageHistory{ImageId: candidate.ImageId, AgentType: candidate.AgentType, AgentVersion: "2099.1.1", PublishedAt: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)}
	if err := DBGlobal(context.Background()).Create(&older).Error; err != nil {
		t.Fatalf("create older history: %v", err)
	}
	if err := DBGlobal(context.Background()).Create(&newer).Error; err != nil {
		t.Fatalf("create newer history: %v", err)
	}
	if err := DBGlobal(context.Background()).Create(&deleted).Error; err != nil {
		t.Fatalf("create deleted history: %v", err)
	}
	if err := DBGlobal(context.Background()).Delete(&deleted).Error; err != nil {
		t.Fatalf("soft delete history: %v", err)
	}

	histories, err := LatestOfficialImageHistories(context.Background())
	if err != nil {
		t.Fatalf("latestOfficialImageHistories: %v", err)
	}
	latest, ok := histories[candidate.ImageId]
	if !ok {
		t.Fatalf("missing latest history for %s", candidate.ImageId)
	}
	if latest.AgentVersion != "2026.5.25" {
		t.Fatalf("latest version = %s, want 2026.5.25", latest.AgentVersion)
	}
}

// TestStrVal 验证指针解引用。
func TestStrVal(t *testing.T) {
	if strVal(nil) != "" {
		t.Error("nil should return empty string")
	}
	s := "hello"
	if strVal(&s) != "hello" {
		t.Error("should return pointed value")
	}
}

// TestInt64Val 验证指针解引用。
func TestInt64Val(t *testing.T) {
	if int64Val(nil) != 0 {
		t.Error("nil should return 0")
	}
	n := int64(42)
	if int64Val(&n) != 42 {
		t.Error("should return pointed value")
	}
}
