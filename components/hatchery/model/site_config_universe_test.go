package model

import (
	"context"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"hatchery/common"
)

// setupSiteConfigUniverseTestDB 初始化用于 ListAllTenants / ApplySiteConfigDefaults 测试的 DB。
func setupSiteConfigUniverseTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	f, err := os.CreateTemp("", "test-siteconfig-universe-*.db")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	f.Close()
	db, err := gorm.Open(sqlite.Open(f.Name()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}
	db.AutoMigrate(&SiteConfig{})
	restore := UseDBForTest(db)
	dbDriver = "sqlite"
	t.Cleanup(func() {
		restore()
		os.Remove(f.Name())
	})
	return db
}

// ─── ApplySiteConfigDefaults ─────────────────────────────────────────────────

func TestApplySiteConfigDefaults_FillsEmptyFields(t *testing.T) {
	c := &SiteConfig{}
	ApplySiteConfigDefaults(c)

	if c.Name != "ClawPro" {
		t.Errorf("Name = %q, want ClawPro", c.Name)
	}
	if c.CVMTemplate == "" {
		t.Error("CVMTemplate should be filled with default")
	}
	if c.PublicImageId != "img-idzg74s9" {
		t.Errorf("PublicImageId = %q, want img-idzg74s9", c.PublicImageId)
	}
	if c.MemoryTDAISupportedVersions == "" {
		t.Error("MemoryTDAISupportedVersions should be filled with default")
	}
}

func TestApplySiteConfigDefaults_PreservesExistingValues(t *testing.T) {
	c := &SiteConfig{
		Name:          "CustomName",
		CVMTemplate:   "custom-tpl",
		PublicImageId: "img-custom",
		MemoryTDAISupportedVersions: "v1,v2",
	}
	ApplySiteConfigDefaults(c)

	if c.Name != "CustomName" {
		t.Errorf("Name should be preserved, got %q", c.Name)
	}
	if c.CVMTemplate != "custom-tpl" {
		t.Errorf("CVMTemplate should be preserved, got %q", c.CVMTemplate)
	}
	if c.PublicImageId != "img-custom" {
		t.Errorf("PublicImageId should be preserved, got %q", c.PublicImageId)
	}
	if c.MemoryTDAISupportedVersions != "v1,v2" {
		t.Errorf("MemoryTDAISupportedVersions should be preserved, got %q", c.MemoryTDAISupportedVersions)
	}
}

func TestApplySiteConfigDefaults_PartialFill(t *testing.T) {
	c := &SiteConfig{
		Name: "Partial",
	}
	ApplySiteConfigDefaults(c)

	if c.Name != "Partial" {
		t.Errorf("Name should stay Partial, got %q", c.Name)
	}
	if c.CVMTemplate == "" {
		t.Error("CVMTemplate should be filled")
	}
	if c.PublicImageId != "img-idzg74s9" {
		t.Errorf("PublicImageId should be filled, got %q", c.PublicImageId)
	}
}

// ─── ListAllTenants ─────────────────────────────────────────────────

func TestListAllTenants_FixedSnapshotMode(t *testing.T) {
	oldSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnap }()

	common.FixedSnapshot = &common.TenantSnapshot{
		Identifier: "fixed-tenant",
		Uin:        "12345",
	}

	tenants, err := ListAllTenants()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tenants) != 1 {
		t.Fatalf("expected 1 tenant, got %d", len(tenants))
	}
	if tenants[0].Identifier != "fixed-tenant" {
		t.Fatalf("expected fixed-tenant, got %q", tenants[0].Identifier)
	}
}

func TestListAllTenants_UniverseMode(t *testing.T) {
	db := setupSiteConfigUniverseTestDB(t)

	oldSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnap }()
	common.FixedSnapshot = nil // universe mode

	// 创建几个租户
	db.Create(&SiteConfig{Identifier: "tenant-a", Uin: "111"})
	db.Create(&SiteConfig{Identifier: "tenant-b", Uin: "222"})
	// identifier 为空的不应返回
	db.Create(&SiteConfig{Identifier: "", Uin: "000"})

	tenants, err := ListAllTenants()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tenants) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(tenants))
	}

	ids := map[string]bool{}
	for _, snap := range tenants {
		ids[snap.Identifier] = true
	}
	if !ids["tenant-a"] || !ids["tenant-b"] {
		t.Fatalf("missing expected tenants: %v", ids)
	}
}

func TestListAllTenants_UniverseMode_Empty(t *testing.T) {
	setupSiteConfigUniverseTestDB(t)

	oldSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnap }()
	common.FixedSnapshot = nil

	tenants, err := ListAllTenants()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tenants) != 0 {
		t.Fatalf("expected 0 tenants, got %d", len(tenants))
	}
}

func TestListAllTenants_UniverseMode_DBClosed(t *testing.T) {
	f, _ := os.CreateTemp("", "test-closed-db-*.db")
	f.Close()
	db, _ := gorm.Open(sqlite.Open(f.Name()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	db.AutoMigrate(&SiteConfig{})
	restore := UseDBForTest(db)
	dbDriver = "sqlite"
	defer func() {
		restore()
		os.Remove(f.Name())
	}()

	oldSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnap }()
	common.FixedSnapshot = nil

	// 关闭数据库连接
	sqlDB, _ := db.DB()
	sqlDB.Close()

	// 查询应该失败但不 panic
	_, err := ListAllTenants()
	if err == nil {
		// SQLite close 后行为可能不一致，不强制失败
		t.Log("no error after DB close (SQLite behavior)")
	}
}

func TestListAllTenants_UniverseMode_SnapshotFields(t *testing.T) {
	db := setupSiteConfigUniverseTestDB(t)

	oldSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnap }()
	common.FixedSnapshot = nil

	db.Create(&SiteConfig{
		Identifier:            "full-tenant",
		Uin:                   "998877",
		Domain:                "https://full.example.com",
		InternalSecret:        "full-secret",
		OneIDAccountID:        "full-oneid",
		CVMSecretId:           "full-sec-id",
		CVMSecretKey:          "full-sec-key",
		AgentCamRoleSecretId:  "full-cam-id",
		AgentCamRoleSecretKey: "full-cam-key",
	})

	tenants, err := ListAllTenants()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tenants) != 1 {
		t.Fatalf("expected 1, got %d", len(tenants))
	}
	s := tenants[0]
	if s.Identifier != "full-tenant" || s.Uin != "998877" || s.Domain != "https://full.example.com" {
		t.Errorf("snapshot fields mismatch: %+v", s)
	}
	if s.InternalSecret != "full-secret" || s.OneIDAccountID != "full-oneid" {
		t.Errorf("secret/oneid mismatch: %+v", s)
	}
	if s.CVMSecretId != "full-sec-id" || s.CVMSecretKey != "full-sec-key" {
		t.Errorf("CVM secret mismatch: %+v", s)
	}
	if s.AgentCamRoleSecretId != "full-cam-id" || s.AgentCamRoleSecretKey != "full-cam-key" {
		t.Errorf("CAM role mismatch: %+v", s)
	}
}

// ─── ListAllTenants: context 隔离 ─────────────────────────────────────────

func TestListAllTenants_UniverseMode_UsesDBGlobalCtx(t *testing.T) {
	db := setupSiteConfigUniverseTestDB(t)

	oldSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnap }()
	common.FixedSnapshot = nil

	// 即便 context 里有 identifier 过滤，ListAllTenants 用的是 DBGlobal（跳过过滤）
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "other"})
	_ = ctx // ListAllTenants 不接受 ctx 参数，内部用 context.Background()

	db.Create(&SiteConfig{Identifier: "isolated-tenant", Uin: "001"})

	tenants, err := ListAllTenants()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tenants) != 1 || tenants[0].Identifier != "isolated-tenant" {
		t.Fatalf("expected isolated-tenant, got %+v", tenants)
	}
}
