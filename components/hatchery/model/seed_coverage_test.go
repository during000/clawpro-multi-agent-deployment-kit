package model

import (
	"context"
	hcommon "hatchery/common"
	"os"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	f, err := os.CreateTemp("", "test-seed-*.db")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	f.Close()
	db, err := gorm.Open(sqlite.Open(f.Name()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}
	db.AutoMigrate(
		&SiteConfig{},
		&SkillBundle{},
		&BundleSkill{},
		&PluginBundle{},
		&BundlePlugin{},
		&OpenClawRole{},
		&OpenClawRoleSkill{},
		&SkillCategory{},
		&PluginCategory{},
		&AIChannel{},
		&AIModel{},
		&UserGroup{},
		&UserGroupMember{},
		&GroupClosure{},
		&GroupConfigBinding{},
		&Instance{},
		&User{},
		&RoleVisibilityGroup{},
		&SkillVisibilityGroup{},
		&SkillBundleVisibilityGroup{},
		&ModelVisibilityGroup{},
		&Tag{},
		&TagVisibilityGroup{},
	)
	orig := UseDBForTest(db)
	dbDriver = "sqlite"
	t.Cleanup(func() {
		orig()
		os.Remove(f.Name())
	})
	return db
}

func TestSeedDefaultSkillBundle_FirstRun(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})
	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})
	if err := SeedDefaultSkillBundle(ctx, db); err != nil {
		t.Fatalf("SeedDefaultSkillBundle failed: %v", err)
	}

	// 验证标记已设置
	var config SiteConfig
	db.First(&config)
	if !config.DefaultBundleSeeded {
		t.Error("DefaultBundleSeeded should be true")
	}

	// 验证技能包已创建
	var count int64
	db.Model(&SkillBundle{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 bundle, got %d", count)
	}
}

func TestSeedDefaultSkillBundle_Idempotent(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test", DefaultBundleSeeded: true})
	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})
	if err := SeedDefaultSkillBundle(ctx, db); err != nil {
		t.Fatalf("SeedDefaultSkillBundle should skip: %v", err)
	}

	var count int64
	db.Model(&SkillBundle{}).Count(&count)
	if count != 0 {
		t.Errorf("should not create bundle when already seeded, got %d", count)
	}
}

func TestSeedDefaultSkillBundle_ExistingBundle(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})
	db.Create(&SkillBundle{Name: DefaultBundleName, Enabled: true})
	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})
	if err := SeedDefaultSkillBundle(ctx, db); err != nil {
		t.Fatalf("SeedDefaultSkillBundle failed: %v", err)
	}

	var config SiteConfig
	db.First(&config)
	if !config.DefaultBundleSeeded {
		t.Error("should set DefaultBundleSeeded flag for existing bundle")
	}
}

func TestSeedDefaultPluginBundle_FirstRun_Cov(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})

	if err := SeedDefaultPluginBundle(db); err != nil {
		t.Fatalf("SeedDefaultPluginBundle failed: %v", err)
	}

	var config SiteConfig
	db.First(&config)
	if !config.DefaultPluginBundleSeeded {
		t.Error("DefaultPluginBundleSeeded should be true")
	}
}

func TestSeedDefaultPluginBundle_Idempotent_Cov(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test", DefaultPluginBundleSeeded: true})

	if err := SeedDefaultPluginBundle(db); err != nil {
		t.Fatalf("SeedDefaultPluginBundle should skip: %v", err)
	}
}

func TestSeedDefaultRoles_FirstRun(t *testing.T) {
	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})

	if err := SeedDefaultRoles(ctx, db); err != nil {
		t.Fatalf("SeedDefaultRoles failed: %v", err)
	}

	// 验证标记已设置
	var config SiteConfig
	db.First(&config)
	if !config.DefaultRolesSeeded {
		t.Error("DefaultRolesSeeded should be true")
	}
}

func TestSeedDefaultRoles_Idempotent(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test", DefaultRolesSeeded: true})

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	if err := SeedDefaultRoles(ctx, db); err != nil {
		t.Fatalf("SeedDefaultRoles should skip: %v", err)
	}

	var count int64
	db.Model(&OpenClawRole{}).Count(&count)
	if count != 0 {
		t.Errorf("should not create roles when already seeded, got %d", count)
	}
}

func TestSeedDefaultRoles_ExistingRoles(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})
	db.Create(&OpenClawRole{Name: "已有角色", Visible: true})

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	if err := SeedDefaultRoles(ctx, db); err != nil {
		t.Fatalf("SeedDefaultRoles failed: %v", err)
	}

	// 应该只设置标记不创建新角色
	var config SiteConfig
	db.First(&config)
	if !config.DefaultRolesSeeded {
		t.Error("should set flag for existing roles")
	}
}

func TestSeedDefaultRoleSkills(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	SeedDefaultRoles(ctx, db)

	// 第二次调用不应 panic
	SeedDefaultRoleSkills(ctx, db)
}

func TestSeedCategories_Cov(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	if err := SeedCategories(ctx, db); err != nil {
		t.Fatalf("SeedCategories failed: %v", err)
	}

	var count int64
	db.Model(&SkillCategory{}).Count(&count)
	if count == 0 {
		t.Error("should create categories")
	}

	// 幂等性
	if err := SeedCategories(ctx, db); err != nil {
		t.Fatalf("SeedCategories(2nd) failed: %v", err)
	}
}

func TestSeedPluginCategories_Coverage(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})

	if err := SeedPluginCategories(db); err != nil {
		t.Fatalf("SeedPluginCategories failed: %v", err)
	}

	var count int64
	db.Model(&PluginCategory{}).Count(&count)
	if count == 0 {
		t.Error("should create plugin categories")
	}

	// 幂等性
	if err := SeedPluginCategories(db); err != nil {
		t.Fatalf("SeedPluginCategories(2nd) failed: %v", err)
	}
}

func TestSeedChannels_Cov(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	if err := SeedChannels(ctx, db); err != nil {
		t.Fatalf("SeedChannels failed: %v", err)
	}
}

func TestSeedModels_Cov(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})

	if err := SeedModels(db); err != nil {
		t.Fatalf("SeedModels failed: %v", err)
	}
}

func TestGetSiteConfig_Empty(t *testing.T) {
	db := setupSeedTestDB(t)
	_ = db // just set up the global DB

	config := GetSiteConfig(context.Background())
	if config.Name != "" {
		t.Errorf("empty DB should return default config, got name=%q", config.Name)
	}
}

func TestGetSiteConfig_WithData(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "MyApp"})

	config := GetSiteConfig(context.Background())
	if config.Name != "MyApp" {
		t.Errorf("expected MyApp, got %q", config.Name)
	}
}

func TestUpdateSiteConfig(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Original"})

	UpdateSiteConfig(context.Background(), map[string]interface{}{
		"name": "Updated",
	})

	var got SiteConfig
	db.First(&got)
	if got.Name != "Updated" {
		t.Errorf("expected Updated, got %q", got.Name)
	}
}

// ─── SeedDefaultSkillBundle error paths ─────────────────────────

func TestSeedDefaultSkillBundle_NoSiteConfig(t *testing.T) {
	// DB 没有 SiteConfig，使 tx.First(&config) 失败
	gdb, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	gdb.AutoMigrate(&SkillBundle{}, &BundleSkill{})
	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})
	// 不迁移 SiteConfig
	err := SeedDefaultSkillBundle(ctx, gdb)
	if err == nil {
		t.Error("expected error when site_configs table is missing")
	}
}

func TestSeedDefaultSkillBundle_CreateBundleFails(t *testing.T) {
	// 迁移 SiteConfig 和 SkillBundle，但不迁移 BundleSkill，使 Create(&bundle) 之后的 Create(&skill) 失败
	gdb, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	gdb.AutoMigrate(&SiteConfig{})
	gdb.Create(&SiteConfig{Name: "test"})
	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})
	// 不迁移 SkillBundle 表，使 Create 失败
	err := SeedDefaultSkillBundle(ctx, gdb)
	if err == nil {
		t.Error("expected error when skill_bundles table is missing")
	}
}

// ─── SeedDefaultPluginBundle error paths ────────────────────────

func TestSeedDefaultPluginBundle_NoSiteConfig(t *testing.T) {
	gdb, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	gdb.AutoMigrate(&PluginBundle{})
	// 没有 site_configs 表
	err := SeedDefaultPluginBundle(gdb)
	if err == nil {
		t.Error("expected error when site_configs table is missing")
	}
}

func TestSeedDefaultPluginBundle_CreateBundleFails(t *testing.T) {
	// SiteConfig 存在但 plugin_bundles 表不存在
	gdb, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	gdb.AutoMigrate(&SiteConfig{})
	gdb.Create(&SiteConfig{Name: "test"})
	err := SeedDefaultPluginBundle(gdb)
	if err == nil {
		t.Error("expected error when plugin_bundles table is missing")
	}
}

// ─── SeedDefaultRoles error paths ───────────────────────────────

func TestSeedDefaultRoles_NoSiteConfig(t *testing.T) {
	gdb, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	gdb.AutoMigrate(&OpenClawRole{}, &OpenClawRoleSkill{})

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	// 没有 site_configs 表
	err := SeedDefaultRoles(ctx, gdb)
	if err == nil {
		t.Error("expected error when site_configs table is missing")
	}
}

func TestSeedDefaultRoles_CountFails(t *testing.T) {
	gdb, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	gdb.AutoMigrate(&SiteConfig{})
	gdb.Create(&SiteConfig{Name: "test"})

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	// open_claw_roles 表不存在，使 Count 失败
	err := SeedDefaultRoles(ctx, gdb)
	if err == nil {
		t.Error("expected error when open_claw_roles table is missing")
	}
}

func TestSeedDefaultRoles_WithRolesJSON(t *testing.T) {
	// 注入一个包含角色数据的 JSON，覆盖 SeedDefaultRoles 中的创建角色循环
	origJSON := DefaultRolesJSON
	defer func() {
		DefaultRolesJSON = origJSON
		defaultRolesOnce = sync.Once{}
		defaultRolesJSON = nil
	}()

	DefaultRolesJSON = []byte(`[
		{"name":"测试角色","description":"desc","soul":"soul","visible":true,"skills":[
			{"name":"skill1","slug":"test-skill-1","version":"1.0.0","source":"public"}
		]}
	]`)
	// 重置 once，让 getDefaultRoles() 重新加载
	defaultRolesOnce = sync.Once{}
	defaultRolesJSON = nil

	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	if err := SeedDefaultRoles(ctx, db); err != nil {
		t.Fatalf("SeedDefaultRoles with roles JSON: %v", err)
	}

	var roleCount int64
	db.Model(&OpenClawRole{}).Count(&roleCount)
	if roleCount == 0 {
		t.Error("expected at least 1 role to be created")
	}

	var skillCount int64
	db.Model(&OpenClawRoleSkill{}).Count(&skillCount)
	if skillCount == 0 {
		t.Error("expected at least 1 role skill to be created")
	}
}

func TestSeedDefaultRoleSkills_WithRolesJSON(t *testing.T) {
	// 注入角色 JSON，覆盖 seedDefaultRoleSkillsTx 中的技能同步逻辑
	origJSON := DefaultRolesJSON
	defer func() {
		DefaultRolesJSON = origJSON
		defaultRolesOnce = sync.Once{}
		defaultRolesJSON = nil
	}()

	DefaultRolesJSON = []byte(`[
		{"name":"同步角色","description":"d","soul":"s","visible":true,"skills":[
			{"name":"s1","slug":"sync-skill","version":"1.0.0","source":"public"}
		]}
	]`)
	defaultRolesOnce = sync.Once{}
	defaultRolesJSON = nil

	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	// 先 Seed 角色
	if err := SeedDefaultRoles(ctx, db); err != nil {
		t.Fatalf("SeedDefaultRoles: %v", err)
	}

	// 再 Sync（应不重复创建）
	SeedDefaultRoleSkills(ctx, db)

	var count int64
	db.Model(&OpenClawRoleSkill{}).Count(&count)
	if count == 0 {
		t.Error("expected skills after sync")
	}
}

// TestSeedDefaultRoleSkills_RemovesObsolete 覆盖 line 213：删除已从配置中移除的多余预置技能
func TestSeedDefaultRoleSkills_RemovesObsolete(t *testing.T) {
	origJSON := DefaultRolesJSON
	defer func() {
		DefaultRolesJSON = origJSON
		defaultRolesOnce = sync.Once{}
		defaultRolesJSON = nil
	}()

	// 第一版配置：角色有技能 A 和 B
	DefaultRolesJSON = []byte(`[
		{"name":"移除测试角色","description":"d","soul":"s","visible":true,"skills":[
			{"name":"a","slug":"skill-a","version":"1.0.0","source":"public"},
			{"name":"b","slug":"skill-b","version":"1.0.0","source":"public"}
		]}
	]`)
	defaultRolesOnce = sync.Once{}
	defaultRolesJSON = nil

	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	// Seed 角色（有两个技能）
	if err := SeedDefaultRoles(ctx, db); err != nil {
		t.Fatalf("SeedDefaultRoles: %v", err)
	}

	// 切换到第二版配置：只保留技能 A（移除了 B）
	DefaultRolesJSON = []byte(`[
		{"name":"移除测试角色","description":"d","soul":"s","visible":true,"skills":[
			{"name":"a","slug":"skill-a","version":"1.0.0","source":"public"}
		]}
	]`)
	defaultRolesOnce = sync.Once{}
	defaultRolesJSON = nil

	// Sync 应删除 skill-b
	SeedDefaultRoleSkills(ctx, db)

	var count int64
	db.Model(&OpenClawRoleSkill{}).Where("slug = ?", "skill-b").Count(&count)
	if count != 0 {
		t.Errorf("obsolete skill-b should have been removed, got count=%d", count)
	}
}

// TestSeedDefaultRoleSkills_AddsNewSkills 覆盖 line 244：补齐缺失的预置技能
func TestSeedDefaultRoleSkills_AddsNewSkills(t *testing.T) {
	origJSON := DefaultRolesJSON
	defer func() {
		DefaultRolesJSON = origJSON
		defaultRolesOnce = sync.Once{}
		defaultRolesJSON = nil
	}()

	// 第一版：只有技能 A
	DefaultRolesJSON = []byte(`[
		{"name":"补齐测试角色","description":"d","soul":"s","visible":true,"skills":[
			{"name":"a","slug":"add-skill-a","version":"1.0.0","source":"public"}
		]}
	]`)
	defaultRolesOnce = sync.Once{}
	defaultRolesJSON = nil

	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	// Seed 角色
	if err := SeedDefaultRoles(ctx, db); err != nil {
		t.Fatalf("SeedDefaultRoles: %v", err)
	}

	// 第二版：新增技能 B
	DefaultRolesJSON = []byte(`[
		{"name":"补齐测试角色","description":"d","soul":"s","visible":true,"skills":[
			{"name":"a","slug":"add-skill-a","version":"1.0.0","source":"public"},
			{"name":"b","slug":"add-skill-b","version":"1.0.0","source":"public"}
		]}
	]`)
	defaultRolesOnce = sync.Once{}
	defaultRolesJSON = nil

	// Sync 应补齐 add-skill-b
	SeedDefaultRoleSkills(ctx, db)

	var count int64
	db.Model(&OpenClawRoleSkill{}).Where("slug = ?", "add-skill-b").Count(&count)
	if count == 0 {
		t.Error("new skill add-skill-b should have been added by sync")
	}
}

// TestSeedDefaultRoleSkills_RoleNotInDB 覆盖 line 184：角色不在 DB 时跳过（continue）
func TestSeedDefaultRoleSkills_RoleNotInDB(t *testing.T) {
	origJSON := DefaultRolesJSON
	defer func() {
		DefaultRolesJSON = origJSON
		defaultRolesOnce = sync.Once{}
		defaultRolesJSON = nil
	}()

	DefaultRolesJSON = []byte(`[
		{"name":"不存在角色","description":"d","soul":"s","visible":true,"skills":[
			{"name":"x","slug":"x-skill","version":"1.0.0","source":"public"}
		]}
	]`)
	defaultRolesOnce = sync.Once{}
	defaultRolesJSON = nil

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})
	// 不 Seed 角色，直接 Sync（角色不存在，应跳过，不 panic）
	SeedDefaultRoleSkills(ctx, db) // should not panic
}

// TestSeedCategories_CreateFails 覆盖 model/skill_category.go lines 64-65
// 用不完整的表触发 Create 失败
func TestSeedCategories_CreateFails(t *testing.T) {
	// 只迁移 SiteConfig，不迁移 SkillCategory 表
	gdb, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	gdb.AutoMigrate(&SiteConfig{})
	gdb.Create(&SiteConfig{Name: "test"})

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	// skill_categories 表不存在，Create 会失败
	err := SeedCategories(ctx, gdb)
	if err == nil {
		t.Error("expected error when skill_categories table is missing")
	}
}

// TestSeedPluginCategories_CreateFails 覆盖 model/plugin_category.go lines 65-66
func TestSeedPluginCategories_CreateFails(t *testing.T) {
	gdb, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	gdb.AutoMigrate(&SiteConfig{})
	gdb.Create(&SiteConfig{Name: "test"})
	// plugin_categories 表不存在，Create 会失败
	err := SeedPluginCategories(gdb)
	if err == nil {
		t.Error("expected error when plugin_categories table is missing")
	}
}
