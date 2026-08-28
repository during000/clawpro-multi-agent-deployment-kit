package task

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	hcommon "hatchery/common"
	"hatchery/model"
)

func setupDefaultBundleSMHSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.SiteConfig{},
		&model.SkillBundle{},
		&model.BundleSkill{},
		&model.OpenClawRoleSkill{},
		&model.SMHSpace{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	return db
}

func setupSMHReady(t *testing.T, db *gorm.DB, lang string) context.Context {
	t.Helper()

	db.Create(&model.SiteConfig{Name: "Test", SMHEnabled: 1})

	expiredAt := time.Now().Add(24 * time.Hour).Unix()
	db.Create(&model.SMHSpace{
		SpaceTag:            "common",
		SpaceId:             "common-space-id",
		AdminToken:          "fake-admin-token",
		AdminTokenExpiredAt: expiredAt,
	})

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: lang,
	})

	return ctx
}

func TestRunDefaultBundleSMHSync_ChineseLang_NoBundle(t *testing.T) {
	db := setupDefaultBundleSMHSyncTestDB(t)
	ctx := setupSMHReady(t, db, "zh")
	runDefaultBundleSMHSync(ctx)

	var config model.SiteConfig
	if err := db.First(&config).Error; err != nil {
		t.Fatal(err)
	}
	if config.SMHEnabled != 1 {
		t.Errorf("SMHEnabled 应保持为 1，实际 %d", config.SMHEnabled)
	}

	var space model.SMHSpace
	if err := db.Where("space_tag = ?", "common").First(&space).Error; err != nil {
		t.Fatal(err)
	}
	if space.AdminToken != "fake-admin-token" {
		t.Errorf("AdminToken 应未被修改，实际 %s", space.AdminToken)
	}

	var count int64
	db.Model(&model.SkillBundle{}).Count(&count)
	if count != 0 {
		t.Errorf("不应创建技能包，实际 %d 条", count)
	}
}

func TestRunDefaultBundleSMHSync_EnglishLang_NoBundle(t *testing.T) {
	db := setupDefaultBundleSMHSyncTestDB(t)
	ctx := setupSMHReady(t, db, "en")
	runDefaultBundleSMHSync(ctx)

	var space model.SMHSpace
	if err := db.Where("space_tag = ?", "common").First(&space).Error; err != nil {
		t.Fatal(err)
	}
	if space.AdminToken != "fake-admin-token" {
		t.Errorf("AdminToken 应未被修改，实际 %s", space.AdminToken)
	}

	var count int64
	db.Model(&model.SkillBundle{}).Count(&count)
	if count != 0 {
		t.Errorf("不应创建技能包，实际 %d 条", count)
	}
}

func TestRunDefaultBundleSMHSync_EmptyLang_NoBundle(t *testing.T) {
	db := setupDefaultBundleSMHSyncTestDB(t)
	ctx := setupSMHReady(t, db, "")
	runDefaultBundleSMHSync(ctx)

	var count int64
	db.Model(&model.SkillBundle{}).Count(&count)
	if count != 0 {
		t.Errorf("不应创建技能包，实际 %d 条", count)
	}
}

func TestRunDefaultBundleSMHSync_BundleExists_StorageFails(t *testing.T) {
	db := setupDefaultBundleSMHSyncTestDB(t)
	bundle := model.SkillBundle{Name: model.DefaultBundleName, Enabled: true}
	db.Create(&bundle)

	db.Create(&model.BundleSkill{
		SkillBundleID: bundle.ID,
		Slug:          "test-skill",
		Version:       "1.0",
		CosZipKey:     "",
	})
	db.Create(&model.OpenClawRoleSkill{
		Slug:      "test-role-skill",
		Version:   "1.0",
		Source:    "public",
		CosZipKey: "",
	})

	ctx := setupSMHReady(t, db, "zh")
	runDefaultBundleSMHSync(ctx)

	var b model.SkillBundle
	if err := model.DB(ctx).Where("name = ?", model.DefaultBundleName).First(&b).Error; err != nil {
		t.Fatal(err)
	}
	if b.Name != model.DefaultBundleName {
		t.Errorf("应查到中文技能包，实际 %s", b.Name)
	}

	var skill model.BundleSkill
	if err := model.DB(ctx).Where("slug = ?", "test-skill").First(&skill).Error; err != nil {
		t.Fatal(err)
	}
	if skill.CosZipKey != "" {
		t.Errorf("BundleSkill.cos_zip_key 应保持为空（storage 客户端失败不会执行同步），实际 %s", skill.CosZipKey)
	}

	var roleSkill model.OpenClawRoleSkill
	if err := model.DB(ctx).Where("slug = ?", "test-role-skill").First(&roleSkill).Error; err != nil {
		t.Fatal(err)
	}
	if roleSkill.CosZipKey != "" {
		t.Errorf("OpenClawRoleSkill.cos_zip_key 应保持为空（storage 客户端失败不会执行同步），实际 %s", roleSkill.CosZipKey)
	}
}

func TestRunDefaultBundleSMHSync_BundleExists_EnStorageFails(t *testing.T) {
	db := setupDefaultBundleSMHSyncTestDB(t)
	bundle := model.SkillBundle{Name: model.DefaultBundleNameEn, Enabled: true}
	db.Create(&bundle)

	db.Create(&model.BundleSkill{
		SkillBundleID: bundle.ID,
		Slug:          "test-skill-en",
		Version:       "1.0",
		CosZipKey:     "",
	})

	ctx := setupSMHReady(t, db, "en")
	runDefaultBundleSMHSync(ctx)

	var b model.SkillBundle
	if err := model.DB(ctx).Where("name = ?", model.DefaultBundleNameEn).First(&b).Error; err != nil {
		t.Fatal(err)
	}
	if b.Name != model.DefaultBundleNameEn {
		t.Errorf("应查到英文技能包，实际 %s", b.Name)
	}

	var skill model.BundleSkill
	if err := model.DB(ctx).Where("slug = ?", "test-skill-en").First(&skill).Error; err != nil {
		t.Fatal(err)
	}
	if skill.CosZipKey != "" {
		t.Errorf("BundleSkill.cos_zip_key 应保持为空，实际 %s", skill.CosZipKey)
	}
}
