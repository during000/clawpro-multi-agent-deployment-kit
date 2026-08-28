package model

import (
	"context"
	hcommon "hatchery/common"
	"os"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupModelMiscTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	f, err := os.CreateTemp("", "test-misc-*.db")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	f.Close()
	db, err := gorm.Open(sqlite.Open(f.Name()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}
	db.AutoMigrate(
		&AIImage{},
		&AIModel{},
		&AIChannel{},
		&Plugin{},
		&PluginBundle{},
		&BundlePlugin{},
		&PluginCategory{},
		&Skill{},
		&SkillBundle{},
		&BundleSkill{},
		&SkillCategory{},
		&SkillVisibilityGroup{},
		&SkillBundleVisibilityGroup{},
		&ModelVisibilityGroup{},
		&RoleVisibilityGroup{},
		&OpenClawRole{},
		&OpenClawRoleSkill{},
		&User{},
		&UserGroup{},
		&UserGroupMember{},
		&SiteConfig{},
	)
	orig := UseDBForTest(db)
	dbDriver = "sqlite"
	t.Cleanup(func() {
		// 先关闭底层 sql.DB 连接，释放 connectionOpener goroutine，
		// 再恢复全局 gdb，最后删除临时文件。
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		orig()
		os.Remove(f.Name())
	})
	return db
}

// ─── ai_image.go ─────────────────────────────────────────────────

func TestGetEnabledImagesMap_Cov(t *testing.T) {
	db := setupModelMiscTestDB(t)
	db.Create(&AIImage{ImageId: "img-1", AgentType: "openclaw", Enabled: true})
	db.Create(&AIImage{ImageId: "img-2", AgentType: "hermes", Enabled: false})

	m, err := GetEnabledImagesMap(context.Background())
	if err != nil {
		t.Fatalf("GetEnabledImagesMap failed: %v", err)
	}
	if len(m) != 1 {
		t.Errorf("expected 1 enabled image, got %d", len(m))
	}
	if _, ok := m["openclaw"]; !ok {
		t.Error("expected openclaw image in map")
	}
}

// ─── ai_model.go ─────────────────────────────────────────────────

func TestSeedModels_FirstRun(t *testing.T) {
	db := setupModelMiscTestDB(t)

	if err := SeedModels(db); err != nil {
		t.Fatalf("SeedModels failed: %v", err)
	}

	var count int64
	db.Model(&AIModel{}).Count(&count)
	if count == 0 {
		t.Error("should create at least one model")
	}
}

func TestSeedModels_Idempotent(t *testing.T) {
	db := setupModelMiscTestDB(t)
	SeedModels(db)

	if err := SeedModels(db); err != nil {
		t.Fatalf("SeedModels(2nd) failed: %v", err)
	}
}

// ─── plugin.go ───────────────────────────────────────────────────

func TestLatestVersionPluginIDs(t *testing.T) {
	db := setupModelMiscTestDB(t)
	db.Create(&Plugin{
		Name: "test-plugin", Slug: "test-plugin",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
	})
	db.Create(&Plugin{
		Name: "test-plugin v2", Slug: "test-plugin",
		VersionMajor: 2, VersionMinor: 0, VersionPatch: 0,
	})

	sub := LatestVersionPluginIDs(context.Background())
	var ids []uint
	db.Raw("SELECT id FROM plugins WHERE id IN (?)", sub).Scan(&ids)
	if len(ids) != 1 {
		t.Errorf("expected 1 latest version, got %d", len(ids))
	}
}

// ─── model_visibility.go ─────────────────────────────────────────

func strPtr(s string) *string { return &s }

func TestGetModelVisibilityGroupIDs_MiscCov(t *testing.T) {
	setupModelMiscTestDB(t)

	ids, err := GetModelVisibilityGroupIDs(context.Background(), []uint{})
	if err != nil {
		t.Fatalf("GetModelVisibilityGroupIDs failed: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0, got %d", len(ids))
	}
}

func TestIsGroupUsedByModelVisibility_MiscCov(t *testing.T) {
	setupModelMiscTestDB(t)

	used, err := IsGroupUsedByModelVisibility(context.Background(), 999)
	if err != nil {
		t.Fatalf("IsGroupUsedByModelVisibility failed: %v", err)
	}
	if used {
		t.Error("nonexistent group should not be in use")
	}
}

// ─── skill_visibility.go ─────────────────────────────────────────

func TestCopySkillVisibility_NoSourceSkill(t *testing.T) {
	db := setupModelMiscTestDB(t)
	toSkill := Skill{Slug: "to-skill", Name: "Target", VersionMajor: 2}
	db.Create(&toSkill)

	// 源技能不存在（slug 不匹配），应正常返回不 panic
	err := CopySkillVisibility(db, "nonexistent-slug", toSkill.ID)
	if err != nil {
		t.Errorf("CopySkillVisibility with no source should not error: %v", err)
	}
}

// ─── ai_channel.go ───────────────────────────────────────────────

func TestSeedChannels_Cov2(t *testing.T) {
	db := setupModelMiscTestDB(t)

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	// 再次 seed（测试幂等）
	if err := SeedChannels(ctx, db); err != nil {
		t.Fatalf("SeedChannels failed: %v", err)
	}
	if err := SeedChannels(ctx, db); err != nil {
		t.Fatalf("SeedChannels(2nd) failed: %v", err)
	}
}

// ─── skill_category.go ───────────────────────────────────────────

func TestSeedCategories_Coverage(t *testing.T) {
	db := setupModelMiscTestDB(t)

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
		t.Error("should have at least one category")
	}
}

// ─── plugin_category.go ──────────────────────────────────────────

func TestSeedPluginCategories_Coverage2(t *testing.T) {
	db := setupModelMiscTestDB(t)

	if err := SeedPluginCategories(db); err != nil {
		t.Fatalf("SeedPluginCategories failed: %v", err)
	}
	if err := SeedPluginCategories(db); err != nil {
		t.Fatalf("SeedPluginCategories(2nd) failed: %v", err)
	}
}

// ─── site_config.go ──────────────────────────────────────────────

func TestGetDefaultAgentType_SetAndGet(t *testing.T) {
	db := setupModelMiscTestDB(t)
	db.Create(&SiteConfig{Name: "Test", DefaultAgentType: "openclaw"})

	at := GetDefaultAgentType(context.Background())
	if at != "openclaw" {
		t.Errorf("expected openclaw, got %q", at)
	}
}

func TestSetDefaultAgentType_Coverage(t *testing.T) {
	db := setupModelMiscTestDB(t)
	db.Create(&SiteConfig{Name: "Test"})

	if err := SetDefaultAgentType(context.Background(), "hermes"); err != nil {
		t.Fatalf("SetDefaultAgentType failed: %v", err)
	}

	at := GetDefaultAgentType(context.Background())
	if at != "hermes" {
		t.Errorf("expected hermes, got %q", at)
	}
}

// ─── user.go ─────────────────────────────────────────────────────

func TestGenerateAPIToken(t *testing.T) {
	db := setupModelMiscTestDB(t)
	user := User{Username: "tok-user", Password: "x", Role: "user"}
	db.Create(&user)

	token, err := GenerateAPIToken(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GenerateAPIToken failed: %v", err)
	}
	if token == "" {
		t.Error("should generate a token")
	}
}

func TestRevokeAPIToken(t *testing.T) {
	db := setupModelMiscTestDB(t)
	user := User{Username: "rev-user", Password: "x", Role: "user", APIToken: strPtr("tok-123")}
	db.Create(&user)

	if err := RevokeAPIToken(context.Background(), user.ID); err != nil {
		t.Fatalf("RevokeAPIToken failed: %v", err)
	}
}

func TestGetUserByAPIToken(t *testing.T) {
	db := setupModelMiscTestDB(t)
	user := User{Username: "bt-user", Password: "x", Role: "user", APIToken: strPtr("test-token-xyz")}
	db.Create(&user)

	got, err := GetUserByAPIToken(context.Background(), "test-token-xyz")
	if err != nil {
		t.Fatalf("GetUserByAPIToken failed: %v", err)
	}
	if got == nil || got.Username != "bt-user" {
		t.Errorf("unexpected user: %v", got)
	}

	// 不存在的 token
	got, err = GetUserByAPIToken(context.Background(), "")
	if err != nil || got != nil {
		t.Error("empty token should return nil, nil")
	}
	got, err = GetUserByAPIToken(context.Background(), "nonexistent")
	if err != nil || got != nil {
		t.Error("nonexistent token should return nil, nil")
	}
}

func TestMaskAPIToken(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"hk-short", "hk-short"}, // too short after prefix
		{"hk-12345678901234567890", "hk-1234****7890"},
	}
	for _, c := range cases {
		got := MaskAPIToken(c.in)
		if got != c.want {
			t.Errorf("MaskAPIToken(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSetAPITokenDisabled(t *testing.T) {
	db := setupModelMiscTestDB(t)
	user := User{Username: "dis-user", Password: "x", Role: "user", APIToken: strPtr("dis-tok")}
	db.Create(&user)

	if err := SetAPITokenDisabled(context.Background(), user.ID, true); err != nil {
		t.Fatalf("SetAPITokenDisabled failed: %v", err)
	}
}

func TestUpdateAPITokenLastUsed(t *testing.T) {
	db := setupModelMiscTestDB(t)
	user := User{Username: "lu-user", Password: "x", Role: "user"}
	db.Create(&user)

	if err := UpdateAPITokenLastUsed(context.Background(), user.ID); err != nil {
		t.Fatalf("UpdateAPITokenLastUsed failed: %v", err)
	}
}

// ─── instance.go / tdai_job.go ──────────────────────────────────

// ─── db.go CloseDB / CloseUnderlyingDBForTest ─────────────────────

func TestCloseDB_WhenGDBNil(t *testing.T) {
	orig := UseDBForTest(nil)
	defer orig()
	// Should not panic
	CloseDB()
}

func TestCloseUnderlyingDBForTest_WhenGDBNil(t *testing.T) {
	orig := UseDBForTest(nil)
	defer orig()
	// gdb == nil → should return nil immediately
	if err := CloseUnderlyingDBForTest(); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// ─── user.go BatchEnsureAPITokens / UserDailyTotalTokenUsage ─────

func TestBatchEnsureAPITokens_MixedUsers(t *testing.T) {
	db := setupModelMiscTestDB(t)

	// Migrate DailyUsageSummary if not already
	db.AutoMigrate(&DailyUsageSummary{})

	tok := "existing-token"
	// user with existing token
	u1 := User{Username: "bt-u1", Password: "x", Role: "user", APIToken: &tok}
	db.Create(&u1)
	// user without token
	u2 := User{Username: "bt-u2", Password: "x", Role: "user"}
	db.Create(&u2)

	results, err := BatchEnsureAPITokens(context.Background())
	if err != nil {
		t.Fatalf("BatchEnsureAPITokens: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Both should have non-empty tokens
	for _, r := range results {
		if r.Token == "" {
			t.Errorf("user %s should have a token", r.Username)
		}
	}
}

func TestUserDailyTotalTokenUsage_Basic(t *testing.T) {
	db := setupModelMiscTestDB(t)
	db.AutoMigrate(&DailyUsageSummary{})

	user := User{Username: "usage-u1", Password: "x", Role: "user"}
	db.Create(&user)

	today := LocalToday()
	db.Create(&DailyUsageSummary{
		UserID:      user.ID,
		Date:        today,
		TotalTokens: 1500,
	})

	total := UserDailyTotalTokenUsage(context.Background(), user.ID)
	if total != 1500 {
		t.Errorf("expected 1500 tokens, got %d", total)
	}
}

func TestUserDailyTotalTokenUsage_NoUsage(t *testing.T) {
	db := setupModelMiscTestDB(t)
	db.AutoMigrate(&DailyUsageSummary{})

	total := UserDailyTotalTokenUsage(context.Background(), 9999)
	if total != 0 {
		t.Errorf("expected 0 tokens for nonexistent user, got %d", total)
	}
}

// ─── instance.go CleanupStaleCreatingInstances ────────────────────

func TestCleanupStaleCreatingInstances_Basic(t *testing.T) {
	db := setupModelMiscTestDB(t)
	db.AutoMigrate(&Instance{})

	// Create a stale creating instance (instance_id = '', created long ago)
	inst := Instance{Name: "stale", InstanceId: ""}
	db.Create(&inst)
	// Backdate created_at to 2 hours ago
	db.Model(&inst).Update("created_at", time.Now().Add(-2*time.Hour))

	affected, err := CleanupStaleCreatingInstances(context.Background(), 1*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleCreatingInstances: %v", err)
	}
	if affected == 0 {
		t.Error("expected at least 1 stale instance to be cleaned")
	}
}

func TestCleanupStaleCreatingInstances_NoneStale(t *testing.T) {
	db := setupModelMiscTestDB(t)
	db.AutoMigrate(&Instance{})

	// Real instance (has instance_id), should not be cleaned
	db.Create(&Instance{Name: "real", InstanceId: "ins-123"})

	affected, err := CleanupStaleCreatingInstances(context.Background(), 1*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleCreatingInstances: %v", err)
	}
	if affected != 0 {
		t.Errorf("expected 0 cleaned, got %d", affected)
	}
}

// ─── model_visibility.go IsGroupUsedByModelVisibility / GetModelsAssociatedWithGroup ───

func TestIsGroupUsedByModelVisibility_WithData(t *testing.T) {
	db := setupModelMiscTestDB(t)

	model1 := AIModel{ModelName: "m1", Enabled: true}
	db.Create(&model1)
	db.Create(&ModelVisibilityGroup{AIModelID: model1.ID, GroupID: 10})

	used, err := IsGroupUsedByModelVisibility(context.Background(), 10)
	if err != nil {
		t.Fatalf("IsGroupUsedByModelVisibility: %v", err)
	}
	if !used {
		t.Error("group 10 should be in use")
	}
}

func TestGetModelsAssociatedWithGroup_Basic(t *testing.T) {
	db := setupModelMiscTestDB(t)

	m1 := AIModel{ModelName: "ma1", Enabled: true}
	m2 := AIModel{ModelName: "ma2", Enabled: true}
	db.Create(&m1)
	db.Create(&m2)
	db.Create(&ModelVisibilityGroup{AIModelID: m1.ID, GroupID: 20})
	db.Create(&ModelVisibilityGroup{AIModelID: m2.ID, GroupID: 20})

	ids, err := GetModelsAssociatedWithGroup(context.Background(), 20)
	if err != nil {
		t.Fatalf("GetModelsAssociatedWithGroup: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 model IDs, got %d", len(ids))
	}
}

func TestGetModelsAssociatedWithGroup_Empty(t *testing.T) {
	setupModelMiscTestDB(t)

	ids, err := GetModelsAssociatedWithGroup(context.Background(), 9999)
	if err != nil {
		t.Fatalf("GetModelsAssociatedWithGroup: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 model IDs, got %d", len(ids))
	}
}

// ─── skill_bundle_visibility.go IsGroupUsedBySkillBundleVisibility ─

func TestIsGroupUsedBySkillBundleVisibility_WithData(t *testing.T) {
	db := setupModelMiscTestDB(t)

	bundle := SkillBundle{Name: "test-bundle", Enabled: true}
	db.Create(&bundle)
	db.Create(&SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: 30})

	used, err := IsGroupUsedBySkillBundleVisibility(context.Background(), 30)
	if err != nil {
		t.Fatalf("IsGroupUsedBySkillBundleVisibility: %v", err)
	}
	if !used {
		t.Error("group 30 should be in use by skill bundle visibility")
	}
}

func TestIsGroupUsedBySkillBundleVisibility_NotUsed2(t *testing.T) {
	setupModelMiscTestDB(t)

	used, err := IsGroupUsedBySkillBundleVisibility(context.Background(), 9999)
	if err != nil {
		t.Fatalf("IsGroupUsedBySkillBundleVisibility: %v", err)
	}
	if used {
		t.Error("nonexistent group should not be in use")
	}
}

// ─── role_visibility.go IsGroupUsedByRoleVisibility ──────────────

func TestIsGroupUsedByRoleVisibility_WithData(t *testing.T) {
	db := setupModelMiscTestDB(t)

	role := OpenClawRole{Name: "test-role", Visible: true}
	db.Create(&role)
	db.Create(&RoleVisibilityGroup{OpenClawRoleID: role.ID, GroupID: 40})

	used, err := IsGroupUsedByRoleVisibility(context.Background(), 40)
	if err != nil {
		t.Fatalf("IsGroupUsedByRoleVisibility: %v", err)
	}
	if !used {
		t.Error("group 40 should be in use by role visibility")
	}
}

func TestIsGroupUsedByRoleVisibility_NotUsed2(t *testing.T) {
	setupModelMiscTestDB(t)

	used, err := IsGroupUsedByRoleVisibility(context.Background(), 9999)
	if err != nil {
		t.Fatalf("IsGroupUsedByRoleVisibility: %v", err)
	}
	if used {
		t.Error("nonexistent group should not be in use")
	}
}

// ─── skill_visibility.go IsGroupUsedBySkillVisibility ────────────

func TestIsGroupUsedBySkillVisibility_WithData(t *testing.T) {
	db := setupModelMiscTestDB(t)
	db.AutoMigrate(&SkillVisibilityGroup{})

	skill := Skill{Slug: "vis-skill", Name: "Vis", VersionMajor: 1}
	db.Create(&skill)
	db.Create(&SkillVisibilityGroup{SkillID: skill.ID, GroupID: 50})

	used, err := IsGroupUsedBySkillVisibility(context.Background(), 50)
	if err != nil {
		t.Fatalf("IsGroupUsedBySkillVisibility: %v", err)
	}
	if !used {
		t.Error("group 50 should be in use by skill visibility")
	}
}

func TestIsGroupUsedBySkillVisibility_NotUsed2(t *testing.T) {
	setupModelMiscTestDB(t)

	used, err := IsGroupUsedBySkillVisibility(context.Background(), 9999)
	if err != nil {
		t.Fatalf("IsGroupUsedBySkillVisibility: %v", err)
	}
	if used {
		t.Error("nonexistent group should not be in use")
	}
}
