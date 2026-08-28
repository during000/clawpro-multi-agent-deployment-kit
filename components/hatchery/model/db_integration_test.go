package model

import (
	"context"
	"os"
	"testing"
	"time"

	"hatchery/common"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB creates a temporary SQLite database for testing.
// When identifier is non-empty, identifier callbacks are registered to simulate multi-tenant mode.
// Tests should now use context with common.InjectTenant(ctx, identifier) instead of currentIdentifier.
func setupTestDB(t *testing.T, identifier string) (cleanup func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "hatchery_test_*.db")
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

	// Save and restore global state
	origDB := gdb

	gdb = testDB

	// Always register identifier callbacks (they're safe with SQLite, just won't filter for empty identifier)
	registerIdentifierCallbacks(gdb)

	// AutoMigrate 用 SkipIdentifier ctx，避免回调在无 TenantSnapshot 时 panic
	migrateDB := gdb.WithContext(common.WithSkipIdentifier(context.Background()))
	if err := migrateDB.AutoMigrate(
		&User{}, &SiteConfig{}, &Instance{}, &AIModel{}, &AIChannel{},
		&LLMUsageLog{}, &DailyUsageSummary{}, &AIImage{}, &AuditLog{},
		&SessionBlacklist{}, &SkillCategory{}, &SkillCategoryMapping{},
		&Skill{}, &SkillDistributionTask{}, &SkillDistributionRecord{},
		&SMHSpace{}, &SMHPersonalSpace{}, &Notification{}, &SkillBundle{}, &BundleSkill{},
		&PublicSkill{}, &SkillInstallation{}, &LocalInstanceSkill{}, &MemoryTDAIPlugin{},
	); err != nil {
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

// ---------- Test 1: IsInitialAdmin query frequency ----------

func TestIsInitialAdmin(t *testing.T) {
	cleanup := setupTestDB(t, "")
	defer cleanup()

	// 使用 SkipIdentifier ctx，测试不关心多租户隔离，只测 IsInitialAdmin 逻辑
	ctx := common.WithSkipIdentifier(context.Background())

	// Create two admins
	admin1 := User{Username: "admin1", Password: "x", Role: "admin"}
	admin2 := User{Username: "admin2", Password: "x", Role: "admin"}
	gdb.WithContext(ctx).Create(&admin1)
	gdb.WithContext(ctx).Create(&admin2)

	// The first-created admin should be the initial admin
	if !admin1.IsInitialAdmin(ctx) {
		t.Error("admin1 should be initial admin")
	}
	if admin2.IsInitialAdmin(ctx) {
		t.Error("admin2 should NOT be initial admin")
	}

	// A regular user should not be initial admin
	user := User{Username: "user1", Password: "x", Role: "user"}
	gdb.WithContext(ctx).Create(&user)
	if user.IsInitialAdmin(ctx) {
		t.Error("regular user should NOT be initial admin")
	}
}

// ---------- Test 2: Unscoped + identifier callback interaction ----------

func TestUnscopedWithIdentifierCallback(t *testing.T) {
	cleanup := setupTestDB(t, "tenant-A")
	defer cleanup()

	// Create a context with TenantSnapshot
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-A"})

	// Create a user in tenant-A
	userA := User{Username: "alice", Password: "x", Role: "user"}
	gdb.WithContext(ctx).Create(&userA)

	// Verify identifier was auto-filled
	var check User
	gdb.WithContext(ctx).First(&check, userA.ID)
	if check.Identifier != "tenant-A" {
		t.Fatalf("expected identifier='tenant-A', got %q", check.Identifier)
	}

	// Manually insert a user with different identifier using raw SQL to bypass callbacks
	gdb.Exec("INSERT INTO users (identifier, username, password, role, instance_quota, token_quota_day, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		"tenant-B", "bob", "x", "user", 1, -1, time.Now(), time.Now())

	// Normal query should only see tenant-A
	var users []User
	gdb.WithContext(ctx).Find(&users)
	if len(users) != 1 || users[0].Username != "alice" {
		t.Errorf("normal query: expected only alice, got %d users: %v", len(users), users)
	}

	// Unscoped query should STILL apply identifier filter (callback is on Query, not on soft-delete scope)
	var unscopedUsers []User
	gdb.WithContext(ctx).Unscoped().Find(&unscopedUsers)
	// Unscoped removes soft-delete condition but our identifier callback should still fire
	allAreOurs := true
	for _, u := range unscopedUsers {
		if u.Identifier != "tenant-A" {
			allAreOurs = false
			break
		}
	}
	if !allAreOurs {
		t.Errorf("Unscoped query leaked cross-tenant data: got users %+v", unscopedUsers)
	}
	t.Logf("Unscoped query returned %d users (all tenant-A: %v)", len(unscopedUsers), allAreOurs)
}

// ---------- Test 3: UpsertDailyUsage with SQLite (identifier='') ----------

func TestUpsertDailyUsage_SQLite(t *testing.T) {
	cleanup := setupTestDB(t, "")
	defer cleanup()

	// SkipIdentifier ctx：此测试仅验证 UpsertDailyUsage 累加逻辑，不关心多租户隔离
	ctx := common.WithSkipIdentifier(context.Background())

	// Create prerequisite records
	user := User{Username: "testuser", Password: "x", Role: "user"}
	gdb.WithContext(ctx).Create(&user)
	instance := Instance{UserID: user.ID, InstanceId: "ins-test"}
	gdb.WithContext(ctx).Create(&instance)
	aiModel := AIModel{ModelName: "gpt-4", ModelID: "gpt-4", Provider: "openai", Enabled: true}
	gdb.WithContext(ctx).Create(&aiModel)

	// First upsert
	UpsertDailyUsage(ctx, user.ID, instance.ID, aiModel.ID, 100, 50, 150, 0, 0, 0)

	var summary DailyUsageSummary
	gdb.WithContext(ctx).First(&summary)
	if summary.TotalTokens != 150 {
		t.Errorf("first upsert: expected total_tokens=150, got %d", summary.TotalTokens)
	}
	if summary.RequestCount != 1 {
		t.Errorf("first upsert: expected request_count=1, got %d", summary.RequestCount)
	}

	// Second upsert should accumulate
	UpsertDailyUsage(ctx, user.ID, instance.ID, aiModel.ID, 200, 100, 300, 0, 0, 0)

	var summary2 DailyUsageSummary
	gdb.WithContext(ctx).First(&summary2)
	if summary2.TotalTokens != 450 {
		t.Errorf("second upsert: expected total_tokens=450, got %d", summary2.TotalTokens)
	}
	if summary2.RequestCount != 2 {
		t.Errorf("second upsert: expected request_count=2, got %d", summary2.RequestCount)
	}
}

// ---------- Test 4: UpsertDailyUsage with identifier (multi-tenant) ----------

func TestUpsertDailyUsage_WithIdentifier(t *testing.T) {
	cleanup := setupTestDB(t, "tenant-X")
	defer cleanup()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-X"})

	user := User{Username: "testuser", Password: "x", Role: "user"}
	gdb.WithContext(ctx).Create(&user)
	instance := Instance{UserID: user.ID, InstanceId: "ins-test"}
	gdb.WithContext(ctx).Create(&instance)
	aiModel := AIModel{ModelName: "gpt-4", ModelID: "gpt-4", Provider: "openai", Enabled: true}
	gdb.WithContext(ctx).Create(&aiModel)

	UpsertDailyUsage(ctx, user.ID, instance.ID, aiModel.ID, 100, 50, 150, 0, 0, 0)

	var summary DailyUsageSummary
	gdb.WithContext(ctx).First(&summary)
	if summary.Identifier != "tenant-X" {
		t.Errorf("expected identifier='tenant-X', got %q", summary.Identifier)
	}
	if summary.TotalTokens != 150 {
		t.Errorf("expected total_tokens=150, got %d", summary.TotalTokens)
	}

	// Second upsert should still accumulate correctly with identifier in the conflict key
	UpsertDailyUsage(ctx, user.ID, instance.ID, aiModel.ID, 50, 25, 75, 0, 0, 0)
	gdb.WithContext(ctx).First(&summary)
	if summary.TotalTokens != 225 {
		t.Errorf("after second upsert: expected total_tokens=225, got %d", summary.TotalTokens)
	}
}

// ---------- Test 5: SiteConfig uniqueIndex on identifier (single row per tenant) ----------

func TestSiteConfig_SinglePerTenant(t *testing.T) {
	cleanup := setupTestDB(t, "")
	defer cleanup()

	// SkipIdentifier：此测试只验证 unique constraint，不涉及多租户过滤
	ctx := common.WithSkipIdentifier(context.Background())

	// Create first config
	gdb.WithContext(ctx).Create(&SiteConfig{Name: "Site1"})

	// Second create with same identifier ('') should fail
	err := gdb.WithContext(ctx).Create(&SiteConfig{Name: "Site2"}).Error
	if err == nil {
		t.Error("expected unique constraint error on second SiteConfig with same identifier, but got nil")
	}
}

// ---------- Test 6: identifier auto-fill on Create ----------

func TestIdentifierAutoFillOnCreate(t *testing.T) {
	cleanup := setupTestDB(t, "my-tenant")
	defer cleanup()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "my-tenant"})

	// Create various models and verify identifier is auto-populated
	skill := Skill{Slug: "test-skill", Name: "Test", Version: "1.0.0"}
	gdb.WithContext(ctx).Create(&skill)

	var check Skill
	gdb.WithContext(ctx).First(&check, skill.ID)
	if check.Identifier != "my-tenant" {
		t.Errorf("Skill.Identifier: expected 'my-tenant', got %q", check.Identifier)
	}

	cat := SkillCategory{Name: "Test Category"}
	gdb.WithContext(ctx).Create(&cat)

	var checkCat SkillCategory
	gdb.WithContext(ctx).First(&checkCat, cat.ID)
	if checkCat.Identifier != "my-tenant" {
		t.Errorf("SkillCategory.Identifier: expected 'my-tenant', got %q", checkCat.Identifier)
	}

	img := AIImage{ImageId: "img-test", ImageName: "Test Image"}
	gdb.WithContext(ctx).Create(&img)

	var checkImg AIImage
	gdb.WithContext(ctx).First(&checkImg, img.ID)
	if checkImg.Identifier != "my-tenant" {
		t.Errorf("AIImage.Identifier: expected 'my-tenant', got %q", checkImg.Identifier)
	}
}

// ---------- Test 7: Cross-tenant isolation on Update/Delete ----------

func TestCrossTenantIsolation_UpdateDelete(t *testing.T) {
	cleanup := setupTestDB(t, "tenant-1")
	defer cleanup()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-1"})

	// Create a user in tenant-1
	user1 := User{Username: "user1", Password: "x", Role: "user"}
	gdb.WithContext(ctx).Create(&user1)

	// Manually insert a user in tenant-2 via raw SQL
	gdb.WithContext(common.WithSkipIdentifier(context.Background())).Exec("INSERT INTO users (identifier, username, password, role, instance_quota, token_quota_day, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		"tenant-2", "user2", "x", "user", 1, -1, time.Now(), time.Now())

	// Try to update user2 via GORM (should be blocked by identifier filter)
	result := gdb.WithContext(ctx).Model(&User{}).Where("username = ?", "user2").Update("role", "admin")
	if result.RowsAffected != 0 {
		t.Error("update should not affect cross-tenant user, but RowsAffected > 0")
	}

	// Try to delete user2 via GORM (should be blocked by identifier filter)
	result = gdb.WithContext(ctx).Where("username = ?", "user2").Delete(&User{})
	if result.RowsAffected != 0 {
		t.Error("delete should not affect cross-tenant user, but RowsAffected > 0")
	}

	// Verify user2 still exists via raw SQL
	skipCtx := common.WithSkipIdentifier(context.Background())
	var count int64
	gdb.WithContext(skipCtx).Raw("SELECT COUNT(*) FROM users WHERE username = ? AND identifier = ?", "user2", "tenant-2").Scan(&count)
	if count != 1 {
		t.Errorf("user2 should still exist in tenant-2, got count=%d", count)
	}
}

// ---------- Test 8: UpdateSiteConfig with zero values (map-based Updates) ----------
// Verifies that GORM Updates(map) does NOT skip zero values like 0 or "".
// This is critical because HandleUpdateSMHConfig removed .Select(columns) and now
// relies on UpdateSiteConfig(map). If zero values were silently dropped, setting
// smh_enabled=0 would be ignored.

func TestUpdateSiteConfig_ZeroValues(t *testing.T) {
	cleanup := setupTestDB(t, "")
	defer cleanup()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: ""})

	// Create initial config with non-zero values
	gdb.WithContext(ctx).Create(&SiteConfig{Name: "TestSite", SMHEnabled: 1, SkillHub: "https://example.com", ChatViewEnabled: true})

	// Update to zero values via map
	err := UpdateSiteConfig(ctx, map[string]interface{}{
		"smh_enabled":       0,
		"skill_hub":         "",
		"chat_view_enabled": false,
	})
	if err != nil {
		t.Fatalf("UpdateSiteConfig error: %v", err)
	}

	config := GetSiteConfig(ctx)
	if config.SMHEnabled != 0 {
		t.Errorf("expected SMHEnabled=0, got %d (zero value was silently dropped!)", config.SMHEnabled)
	}
	if config.SkillHub != "" {
		t.Errorf("expected SkillHub='', got %q (empty string was silently dropped!)", config.SkillHub)
	}
	if config.ChatViewEnabled {
		t.Error("expected ChatViewEnabled=false after zero-value update")
	}
}

func TestGetSiteConfig_DefaultChatViewEnabled(t *testing.T) {
	cleanup := setupTestDB(t, "")
	defer cleanup()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: ""})

	config := GetSiteConfig(ctx)
	if !config.ChatViewEnabled {
		t.Fatal("expected ChatViewEnabled=true by default when site config row is missing")
	}
}

func TestSiteConfig_SavePersistsFalseWithTrueDefault(t *testing.T) {
	cleanup := setupTestDB(t, "")
	defer cleanup()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: ""})

	config := SiteConfig{Name: "TestSite", ChatViewEnabled: true}
	if err := gdb.WithContext(ctx).Create(&config).Error; err != nil {
		t.Fatalf("create site config: %v", err)
	}

	config.ChatViewEnabled = false
	if err := gdb.WithContext(ctx).Save(&config).Error; err != nil {
		t.Fatalf("save site config: %v", err)
	}

	reloaded := GetSiteConfig(ctx)
	if reloaded.ChatViewEnabled {
		t.Fatal("expected ChatViewEnabled=false after Save, got true")
	}
}

func TestSaveSelectedFields_RejectsUnknownField(t *testing.T) {
	cleanup := setupTestDB(t, "")
	defer cleanup()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: ""})

	cfg := SiteConfig{Name: "before"}
	if err := gdb.WithContext(ctx).Create(&cfg).Error; err != nil {
		t.Fatalf("create site config: %v", err)
	}

	cfg.Name = "after"
	err := SaveSelectedFields(ctx, &cfg, "Nmae")
	if err == nil {
		t.Fatal("expected unknown field error, got nil")
	}
	if got := err.Error(); got != `unknown SiteConfig field "Nmae"` {
		t.Fatalf("unexpected error: %s", got)
	}

	stored := GetSiteConfig(ctx)
	if stored.Name != "before" {
		t.Fatalf("unknown field update should not persist Name, got %q", stored.Name)
	}
}

func TestSaveSelectedFields_UpdatesOnlySelectedFields(t *testing.T) {
	cleanup := setupTestDB(t, "")
	defer cleanup()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: ""})

	cfg := SiteConfig{Name: "before", SkillHub: "https://skillhub.before.example.com", TerminalEnabled: true}
	if err := gdb.WithContext(ctx).Create(&cfg).Error; err != nil {
		t.Fatalf("create site config: %v", err)
	}

	cfg.Name = "after"
	cfg.SkillHub = "https://skillhub.after.example.com"
	cfg.TerminalEnabled = false
	if err := SaveSelectedFields(ctx, &cfg, "Name", "TerminalEnabled"); err != nil {
		t.Fatalf("SaveSelectedFields: %v", err)
	}

	stored := GetSiteConfig(ctx)
	if stored.Name != "after" {
		t.Fatalf("Name = %q, want after", stored.Name)
	}
	if stored.TerminalEnabled {
		t.Fatal("TerminalEnabled = true, want false")
	}
	if stored.SkillHub != "https://skillhub.before.example.com" {
		t.Fatalf("SkillHub should be preserved, got %q", stored.SkillHub)
	}
}

// ---------- Test 9: SeedDefaultSkillBundle with transaction and identifier ----------
// Verifies that tx.Create inside gdb.Transaction inherits identifier callbacks.

func TestTransactionInheritsIdentifierCallbacks(t *testing.T) {
	cleanup := setupTestDB(t, "txn-tenant")
	defer cleanup()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "txn-tenant"})

	// Simulate what SeedDefaultSkillBundle does: create inside a transaction
	err := gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		bundle := SkillBundle{Name: "test-bundle", SkillCount: 1, Enabled: true}
		if err := tx.Create(&bundle).Error; err != nil {
			return err
		}
		skill := BundleSkill{SkillBundleID: bundle.ID, Name: "s1", Slug: "s1", Version: "1.0.0", Source: "public"}
		return tx.Create(&skill).Error
	})
	if err != nil {
		t.Fatalf("transaction error: %v", err)
	}

	// Verify identifier was filled via callback even inside transaction
	var bundle SkillBundle
	gdb.WithContext(ctx).First(&bundle)
	if bundle.Identifier != "txn-tenant" {
		t.Errorf("SkillBundle.Identifier: expected 'txn-tenant', got %q", bundle.Identifier)
	}

	var skill BundleSkill
	gdb.WithContext(ctx).First(&skill)
	if skill.Identifier != "txn-tenant" {
		t.Errorf("BundleSkill.Identifier: expected 'txn-tenant', got %q", skill.Identifier)
	}
}

// ---------- Test 10: CAS update on SiteConfig (openclaw.go ensureEgressRulesCore) ----------
// The old code used Where("id = 1 AND auto_created_security_group_id = ''"),
// new code uses Where("auto_created_security_group_id = ''").
// Verify that CAS (compare-and-swap) still works correctly without id=1.

func TestSiteConfig_CASUpdate(t *testing.T) {
	cleanup := setupTestDB(t, "")
	defer cleanup()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: ""})

	gdb.WithContext(ctx).Create(&SiteConfig{Name: "CAS Test"})

	// CAS: only update if field is empty
	result := gdb.WithContext(ctx).Model(&SiteConfig{}).
		Where("auto_created_security_group_id = ''").
		Update("auto_created_security_group_id", "sg-12345")
	if result.RowsAffected != 1 {
		t.Errorf("CAS update should affect 1 row, got %d", result.RowsAffected)
	}

	// Second CAS should fail (field already set)
	result = gdb.WithContext(ctx).Model(&SiteConfig{}).
		Where("auto_created_security_group_id = ''").
		Update("auto_created_security_group_id", "sg-99999")
	if result.RowsAffected != 0 {
		t.Errorf("second CAS should affect 0 rows, got %d", result.RowsAffected)
	}

	// Verify original value preserved
	config := GetSiteConfig(ctx)
	if config.AutoCreatedSecurityGroupId != "sg-12345" {
		t.Errorf("expected sg-12345, got %q", config.AutoCreatedSecurityGroupId)
	}
}

// ---------- Test 11: SMHPersonalSpace — 联合唯一索引 (identifier + instance_id) ----------
// 验证改造后的联合唯一索引：同一 identifier 下 instance_id 不可重复，不同 identifier 可重复。

func TestSMHPersonalSpace_CompositeUniqueIndex(t *testing.T) {
	cleanup := setupTestDB(t, "tenant-A")
	defer cleanup()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-A"})

	// 创建第一条记录
	space1 := SMHPersonalSpace{SpaceId: "sp-001", UserId: 1, InstanceId: 100, UserName: "alice", InstanceName: "ins-1"}
	if err := gdb.WithContext(ctx).Create(&space1).Error; err != nil {
		t.Fatalf("create space1: %v", err)
	}

	// 同一 identifier 下重复 instance_id 应失败
	space2 := SMHPersonalSpace{SpaceId: "sp-002", UserId: 2, InstanceId: 100, UserName: "bob", InstanceName: "ins-1-dup"}
	err := gdb.WithContext(ctx).Create(&space2).Error
	if err == nil {
		t.Error("expected unique constraint error for duplicate instance_id under same identifier, got nil")
	}

	// 不同 identifier 下相同 instance_id 应成功
	// 切换到 tenant-B，通过 GORM 正常创建（回调自动填充 tenant-B）
	ctxB := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-B"})
	space3 := SMHPersonalSpace{SpaceId: "sp-003", UserId: 3, InstanceId: 100, UserName: "charlie", InstanceName: "ins-1-other-tenant"}
	if err := DB(ctxB).Create(&space3).Error; err != nil {
		t.Errorf("different identifier with same instance_id should succeed, got: %v", err)
	}
}

// ---------- Test 12: SMHPersonalSpace — identifier 自动填充与跨租户隔离 ----------

func TestSMHPersonalSpace_IdentifierAutoFillAndIsolation(t *testing.T) {
	cleanup := setupTestDB(t, "tenant-X")
	defer cleanup()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-X"})

	// 创建记录，验证 identifier 自动填充
	space := SMHPersonalSpace{SpaceId: "sp-100", UserId: 1, InstanceId: 200, UserName: "user1", InstanceName: "ins-200"}
	if err := gdb.WithContext(ctx).Create(&space).Error; err != nil {
		t.Fatalf("create tenant-X space: %v", err)
	}

	var check SMHPersonalSpace
	gdb.WithContext(ctx).First(&check, space.ID)
	if check.Identifier != "tenant-X" {
		t.Errorf("expected Identifier='tenant-X', got %q", check.Identifier)
	}

	// 切换到另一个租户，通过 GORM 正常创建记录（回调会自动填充 tenant-Y）
	ctxY := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-Y"})
	spaceY := SMHPersonalSpace{SpaceId: "sp-200", UserId: 2, InstanceId: 300, UserName: "user2", InstanceName: "ins-300"}
	if err := DB(ctxY).Create(&spaceY).Error; err != nil {
		t.Fatalf("create tenant-Y space: %v", err)
	}

	// 在 tenant-Y 视角下只能看到自己的数据
	var spacesY []SMHPersonalSpace
	DB(ctxY).Find(&spacesY)
	if len(spacesY) != 1 || spacesY[0].SpaceId != "sp-200" {
		t.Errorf("tenant-Y query: expected only sp-200, got %d spaces", len(spacesY))
	}

	// 切回 tenant-X，验证只能看到自己的数据
	ctxX := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-X"})
	var spacesX []SMHPersonalSpace
	DB(ctxX).Find(&spacesX)
	if len(spacesX) != 1 {
		t.Errorf("expected 1 space (tenant-X only), got %d", len(spacesX))
	}
	for _, s := range spacesX {
		if s.Identifier != "tenant-X" {
			t.Errorf("leaked cross-tenant data: identifier=%q", s.Identifier)
		}
	}
}

// ---------- Test 13: SMHPersonalSpace — CreatePersonalSpace / HasPersonalSpace / EnvInitialized ----------

func TestSMHPersonalSpace_CRUDFunctions(t *testing.T) {
	cleanup := setupTestDB(t, "crud-tenant")
	defer cleanup()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "crud-tenant"})

	// HasPersonalSpace 应返回 false
	has, err := HasPersonalSpace(ctx, 999)
	if err != nil {
		t.Fatalf("HasPersonalSpace error: %v", err)
	}
	if has {
		t.Error("expected no personal space for instance 999")
	}

	// CreatePersonalSpace
	space := &SMHPersonalSpace{
		SpaceId:      "sp-test",
		UserId:       1,
		InstanceId:   999,
		UserName:     "testuser",
		InstanceName: "ins-test",
	}
	if err := CreatePersonalSpace(ctx, space); err != nil {
		t.Fatalf("CreatePersonalSpace error: %v", err)
	}
	if space.ID == 0 {
		t.Error("expected non-zero ID after create")
	}

	// HasPersonalSpace 应返回 true
	has, err = HasPersonalSpace(ctx, 999)
	if err != nil {
		t.Fatalf("HasPersonalSpace error: %v", err)
	}
	if !has {
		t.Error("expected personal space to exist for instance 999")
	}

	// env_initialized — 默认应为 false
	var checkSpace SMHPersonalSpace
	if err := gdb.WithContext(ctx).First(&checkSpace, space.ID).Error; err != nil {
		t.Fatalf("query space error: %v", err)
	}
	if checkSpace.EnvInitialized {
		t.Error("expected env_initialized=false for new space")
	}

	// 更新 env_initialized 为 true 后再验证
	gdb.WithContext(ctx).Model(space).Update("env_initialized", true)
	if err := gdb.WithContext(ctx).First(&checkSpace, space.ID).Error; err != nil {
		t.Fatalf("query space after update error: %v", err)
	}
	if !checkSpace.EnvInitialized {
		t.Error("expected env_initialized=true after update")
	}

	// 多租户隔离验证：切换到另一个租户，查询不应看到当前租户的数据
	ctxOther := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "other-tenant"})
	var otherSpaces []SMHPersonalSpace
	if err := DB(ctxOther).Find(&otherSpaces).Error; err != nil {
		t.Fatalf("query other-tenant error: %v", err)
	}
	if len(otherSpaces) > 0 {
		t.Error("other-tenant should not see cross-tenant data")
	}
}

// ---------- Test 14: HandleUpdateSMHConfig 改造验证 — UpdateSiteConfig 替代硬编码 ----------
// 验证 controller 层改造后，通过 UpdateSiteConfig(map) 更新 SMH 配置字段的正确性：
// 1. smh_enabled=0 这种零值不会被丢弃
// 2. 布尔字段 smh_auto_provision_on_create 的 true→false 更新正确生效
// 3. 多租户模式下 UpdateSiteConfig 只影响当前租户

func TestUpdateSiteConfig_SMHFields(t *testing.T) {
	cleanup := setupTestDB(t, "smh-tenant")
	defer cleanup()

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "smh-tenant"})

	// 创建初始配置
	gdb.WithContext(ctx).Create(&SiteConfig{
		Name: "SMH Test", SMHEnabled: 1,
		SMHLibraryId: "lib-old", SMHEndpoint: "https://old.example.com",
		SMHAutoProvisionOnCreate: true,
	})

	// 模拟 HandleUpdateSMHConfig 的行为：通过 map 更新多个字段
	updates := map[string]interface{}{
		"smh_enabled":                  0,
		"smh_library_id":               "lib-new",
		"smh_endpoint":                 "https://new.example.com",
		"smh_auto_provision_on_create": false,
	}
	if err := UpdateSiteConfig(ctx, updates); err != nil {
		t.Fatalf("UpdateSiteConfig error: %v", err)
	}

	config := GetSiteConfig(ctx)
	if config.SMHEnabled != 0 {
		t.Errorf("expected SMHEnabled=0, got %d", config.SMHEnabled)
	}
	if config.SMHLibraryId != "lib-new" {
		t.Errorf("expected SMHLibraryId='lib-new', got %q", config.SMHLibraryId)
	}
	if config.SMHEndpoint != "https://new.example.com" {
		t.Errorf("expected SMHEndpoint='https://new.example.com', got %q", config.SMHEndpoint)
	}
	if config.SMHAutoProvisionOnCreate {
		t.Error("expected SMHAutoProvisionOnCreate=false, got true (bool zero value was silently dropped!)")
	}

	// 多租户隔离验证：切换到另一个租户，创建独立配置
	ctxOtherSmh := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "other-smh-tenant"})
	DB(ctxOtherSmh).Create(&SiteConfig{Name: "Other Tenant", SMHEnabled: 0})

	// 更新 other-smh-tenant 的配置 - 需要设置当前context进行更新
	// 为简化起见，直接用 DB(ctxOtherSmh).Model(...).Updates(...)
	if err := DB(ctxOtherSmh).Model(&SiteConfig{}).Updates(map[string]interface{}{"smh_enabled": 1}).Error; err != nil {
		t.Fatalf("UpdateSiteConfig (other-tenant) error: %v", err)
	}

	// 切回原租户，验证配置未被影响
	ctxSmhTenant := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "smh-tenant"})
	var origConfig SiteConfig
	DB(ctxSmhTenant).First(&origConfig)
	if origConfig.SMHEnabled != 0 {
		t.Errorf("cross-tenant UpdateSiteConfig leaked: expected SMHEnabled=0, got %d", origConfig.SMHEnabled)
	}
}

func TestUpsertDailyUsageCacheTokens(t *testing.T) {
	cleanup := setupTestDB(t, "")
	defer cleanup()
	ctx := common.WithSkipIdentifier(context.Background())
	user := &User{Username: "cache-user", Email: "cache@example.com"}
	gdb.WithContext(ctx).Create(user)
	instance := &Instance{Name: "cache-inst", UserID: user.ID}
	gdb.WithContext(ctx).Create(instance)
	aiModel := &AIModel{ModelID: "cache-model", Provider: "test"}
	gdb.WithContext(ctx).Create(aiModel)

	UpsertDailyUsage(ctx, user.ID, instance.ID, aiModel.ID, 100, 50, 150, 0, 30, 12)
	UpsertDailyUsage(ctx, user.ID, instance.ID, aiModel.ID, 10, 5, 15, 0, 3, 1)

	var summary DailyUsageSummary
	gdb.WithContext(ctx).First(&summary)
	if summary.PromptCacheReadTokens != 33 || summary.PromptCacheWriteTokens != 13 {
		t.Fatalf("cache tokens = read %d write %d, want read 33 write 13", summary.PromptCacheReadTokens, summary.PromptCacheWriteTokens)
	}
}
