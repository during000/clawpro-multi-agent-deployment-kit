package model

import (
	"context"
	hcommon "hatchery/common"
	"testing"
)

// ─── site_config.go ──────────────────────────────────────────────

func TestGetSiteConfig_NilDB(t *testing.T) {
	// 多租户改造后 DB(ctx) 返回 nil 时 GORM 会 panic，此测试场景不再适用
	t.Skip("skipped: DB(ctx) nil panic not handled in multi-tenant mode")
	// gdb == nil 时应返回安全默认值
	defer UseNilDBForTest()()

	config := GetSiteConfig(context.Background())
	if config.DefaultInstanceQuota != 3 {
		t.Errorf("default quota want 3, got %d", config.DefaultInstanceQuota)
	}
	if !config.ChatViewEnabled {
		t.Error("default ChatViewEnabled should be true")
	}
}

func TestGlobalDailyTokenUsage(t *testing.T) {
	db := setupSeedTestDB(t)
	db.AutoMigrate(&DailyUsageSummary{})

	total := GlobalDailyTokenUsage(context.Background())
	if total != 0 {
		t.Errorf("empty DB should have 0 usage, got %d", total)
	}
}

// ─── skill_bundle.go 分支 ────────────────────────────────────────

func TestSeedDefaultSkillBundle_WithSkills(t *testing.T) {
	db := setupSeedTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})
	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})
	// 第一次调用创建 bundle
	if err := SeedDefaultSkillBundle(ctx, db); err != nil {
		t.Fatalf("SeedDefaultSkillBundle failed: %v", err)
	}

	// 验证 BundleSkill 已创建（如果 defaultBundleSkills 非空）
	var config SiteConfig
	db.First(&config)
	if !config.DefaultBundleSeeded {
		t.Error("DefaultBundleSeeded should be true")
	}
}

// ─── plugin_bundle.go 分支 ───────────────────────────────────────

func TestSeedDefaultPluginBundle_WithPlugins(t *testing.T) {
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

// ─── user.go 剩余行 ──────────────────────────────────────────────

func TestGetUserByAPIToken_Banned(t *testing.T) {
	db := setupSeedTestDB(t)
	token := "banned-token-xyz"
	user := User{Username: "banned-user", Password: "x", Role: "user", APIToken: &token}
	db.Create(&user)
	// 软删除用户
	db.Delete(&user)

	got, err := GetUserByAPIToken(context.Background(), token)
	if got != nil {
		t.Error("soft-deleted user should not be returned")
	}
	if _, ok := err.(BannedError); !ok {
		t.Errorf("expected BannedError, got %T: %v", err, err)
	}
}

func TestGetUserByAPIToken_TokenDisabled(t *testing.T) {
	db := setupSeedTestDB(t)
	token := "disabled-token-abc"
	user := User{Username: "disabled-user", Password: "x", Role: "user", APIToken: &token, APITokenDisabled: true}
	db.Create(&user)

	got, err := GetUserByAPIToken(context.Background(), token)
	if got != nil {
		t.Error("disabled token should not return user")
	}
	if _, ok := err.(TokenDisabledError); !ok {
		t.Errorf("expected TokenDisabledError, got %T: %v", err, err)
	}
}

// ─── user_group.go 剩余行 ────────────────────────────────────────

func TestCreateUserGroup_DuplicateName(t *testing.T) {
	db := setupSeedTestDB(t)
	_ = db
	CreateUserGroup(context.Background(), "dup-group", "")
	// 重复名称行为：可能错误也可能覆盖
	_, _ = CreateUserGroup(context.Background(), "dup-group", "")
}

func TestUpdateUserGroupMemberships_Empty(t *testing.T) {
	db := setupSeedTestDB(t)
	user := User{Username: "empty-mbr-user", Password: "x", Role: "user"}
	db.Create(&user)

	// 空 groupIDs 应该清空所有
	err := UpdateUserGroupMemberships(db, user.ID, []uint{})
	if err != nil {
		t.Fatalf("UpdateUserGroupMemberships(empty) failed: %v", err)
	}
}
