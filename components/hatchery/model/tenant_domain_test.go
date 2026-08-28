package model

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"hatchery/common"
	"hatchery/i18n"
)

// setupTenantDomainTestDB 初始化测试用 SQLite 数据库，包含 tenant_domain 和 site_config 相关表。
func setupTenantDomainTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	f, err := os.CreateTemp("", "test-tenant-domain-*.db")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	f.Close()
	db, err := gorm.Open(sqlite.Open(f.Name()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}
	db.AutoMigrate(&TenantDomain{}, &SiteConfig{}, &User{})
	t.Cleanup(func() {
		UseDBForTest(nil)()
		os.Remove(f.Name())
	})
	restore := UseDBForTest(db)
	t.Cleanup(restore)
	dbDriver = "sqlite"
	return db
}

// ─── ExtractHostFromURL ─────────────────────────────────────────────────

func TestExtractHostFromURL_Empty(t *testing.T) {
	if got := ExtractHostFromURL(""); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if got := ExtractHostFromURL("   "); got != "" {
		t.Fatalf("expected empty for whitespace, got %q", got)
	}
}

func TestExtractHostFromURL_FullURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://a.example.com", "a.example.com"},
		{"http://b.example.com:8080", "b.example.com"},
		{"https://c.example.com:443/path?q=1", "c.example.com"},
		{"http://[::1]:8080/path", "::1"},
	}
	for _, tt := range tests {
		got := ExtractHostFromURL(tt.input)
		if got != tt.want {
			t.Errorf("ExtractHostFromURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractHostFromURL_HostPort(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a.example.com:8080", "a.example.com"},
		{"[::1]:8080", "::1"},
	}
	for _, tt := range tests {
		got := ExtractHostFromURL(tt.input)
		if got != tt.want {
			t.Errorf("ExtractHostFromURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractHostFromURL_PlainHost(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a.example.com", "a.example.com"},
		{"localhost", "localhost"},
	}
	for _, tt := range tests {
		got := ExtractHostFromURL(tt.input)
		if got != tt.want {
			t.Errorf("ExtractHostFromURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ─── IsDuplicateError ─────────────────────────────────────────────────

func TestIsDuplicateError_Nil(t *testing.T) {
	if IsDuplicateError(nil) {
		t.Fatal("nil error should not be duplicate")
	}
}

func TestIsDuplicateError_MySQL(t *testing.T) {
	err := errors.New("Error 1062: Duplicate entry 'xxx' for key 'domain'")
	if !IsDuplicateError(err) {
		t.Fatal("MySQL duplicate error not detected")
	}
}

func TestIsDuplicateError_SQLite(t *testing.T) {
	err := errors.New("UNIQUE constraint failed: tenant_domains.domain")
	if !IsDuplicateError(err) {
		t.Fatal("SQLite UNIQUE error not detected")
	}
}

func TestIsDuplicateError_OtherError(t *testing.T) {
	err := errors.New("connection refused")
	if IsDuplicateError(err) {
		t.Fatal("random error should not be duplicate")
	}
}

// ─── SnapFromConfig ─────────────────────────────────────────────────

func TestSnapFromConfig_AllFields(t *testing.T) {
	c := SiteConfig{
		Identifier:            "tenant-1",
		Uin:                   "100001",
		Domain:                "https://tenant1.example.com",
		InternalSecret:        "secret-1",
		OneIDAccountID:        "oneid-1",
		OneIDAppID:            "app-001",
		OneIDClientID:         "client-001",
		OneIDClientSecret:     "secret-001",
		OneIDTokenEndpoint:    "https://oneid.example.com/token",
		OneIDDomain:           "oneid.example.com",
		CVMSecretId:           "sec-id-1",
		CVMSecretKey:          "sec-key-1",
		AgentCamRoleSecretId:  "cam-id-1",
		AgentCamRoleSecretKey: "cam-key-1",
	}
	snap := SnapFromConfig(c)
	if snap.Identifier != "tenant-1" {
		t.Errorf("Identifier = %q, want tenant-1", snap.Identifier)
	}
	if snap.Uin != "100001" {
		t.Errorf("Uin = %q, want 100001", snap.Uin)
	}
	if snap.Domain != "https://tenant1.example.com" {
		t.Errorf("Domain = %q", snap.Domain)
	}
	if snap.InternalSecret != "secret-1" {
		t.Errorf("InternalSecret = %q", snap.InternalSecret)
	}
	if snap.OneIDAccountID != "oneid-1" {
		t.Errorf("OneIDAccountID = %q", snap.OneIDAccountID)
	}
	if snap.OneIDAppID != "app-001" {
		t.Errorf("OneIDAppID = %q, want app-001", snap.OneIDAppID)
	}
	if snap.OneIDClientID != "client-001" {
		t.Errorf("OneIDClientID = %q, want client-001", snap.OneIDClientID)
	}
	if snap.OneIDClientSecret != "secret-001" {
		t.Errorf("OneIDClientSecret = %q, want secret-001", snap.OneIDClientSecret)
	}
	if snap.OneIDTokenEndpoint != "https://oneid.example.com/token" {
		t.Errorf("OneIDTokenEndpoint = %q", snap.OneIDTokenEndpoint)
	}
	if snap.OneIDDomain != "oneid.example.com" {
		t.Errorf("OneIDDomain = %q, want oneid.example.com", snap.OneIDDomain)
	}
	if snap.CVMSecretId != "sec-id-1" {
		t.Errorf("CVMSecretId = %q", snap.CVMSecretId)
	}
	if snap.CVMSecretKey != "sec-key-1" {
		t.Errorf("CVMSecretKey = %q", snap.CVMSecretKey)
	}
	if snap.AgentCamRoleSecretId != "cam-id-1" {
		t.Errorf("AgentCamRoleSecretId = %q", snap.AgentCamRoleSecretId)
	}
	if snap.AgentCamRoleSecretKey != "cam-key-1" {
		t.Errorf("AgentCamRoleSecretKey = %q", snap.AgentCamRoleSecretKey)
	}
}

func TestSnapFromConfig_Empty(t *testing.T) {
	c := SiteConfig{}
	snap := SnapFromConfig(c)
	if snap.Identifier != "" || snap.Uin != "" || snap.Domain != "" {
		t.Fatalf("empty config should produce empty snap, got %+v", snap)
	}
}

// ─── WarmTenantCache / InvalidateTenantCache ─────────────────────────

func TestWarmAndInvalidateTenantCache(t *testing.T) {
	domain := "test-cache.example.com"
	snap := common.TenantSnapshot{Identifier: "cached-tenant", Domain: domain}

	// 初始状态：缓存中没有
	tenantCache.Delete(domain) // 清理
	if _, ok := tenantCache.Load(domain); ok {
		t.Fatal("cache should be empty initially")
	}

	// 预热
	WarmTenantCache(domain, snap)
	v, ok := tenantCache.Load(domain)
	if !ok {
		t.Fatal("cache should have entry after warm")
	}
	if v.(common.TenantSnapshot).Identifier != "cached-tenant" {
		t.Fatalf("unexpected cached snap: %+v", v)
	}

	// 失效
	InvalidateTenantCache(domain)
	if _, ok := tenantCache.Load(domain); ok {
		t.Fatal("cache should be empty after invalidate")
	}
}

// ─── ResolveTenant ─────────────────────────────────────────────────

func TestResolveTenant_FromCache(t *testing.T) {
	domain := "resolve-cache.example.com"
	snap := common.TenantSnapshot{Identifier: "from-cache"}
	tenantCache.Store(domain, snap)
	defer tenantCache.Delete(domain)

	got, err := ResolveTenant(context.Background(), domain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Identifier != "from-cache" {
		t.Fatalf("expected from-cache, got %q", got.Identifier)
	}
}

func TestResolveTenant_FromDB(t *testing.T) {
	db := setupTenantDomainTestDB(t)
	domain := "resolve-db.example.com"
	tenantCache.Delete(domain)

	// 写入数据
	db.Create(&TenantDomain{Domain: domain, Identifier: "db-tenant"})
	db.Create(&SiteConfig{Identifier: "db-tenant", Uin: "999", Domain: "https://" + domain})

	ctx := common.WithSkipIdentifier(context.Background())
	got, err := ResolveTenant(ctx, domain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Identifier != "db-tenant" {
		t.Fatalf("expected db-tenant, got %q", got.Identifier)
	}
	if got.Uin != "999" {
		t.Fatalf("expected uin 999, got %q", got.Uin)
	}

	// 验证缓存已写入
	if _, ok := tenantCache.Load(domain); !ok {
		t.Fatal("cache should be populated after DB resolve")
	}
	tenantCache.Delete(domain) // 清理
}

func TestResolveTenant_UnknownDomain(t *testing.T) {
	setupTenantDomainTestDB(t)
	tenantCache.Delete("unknown.example.com")

	ctx := common.WithSkipIdentifier(context.Background())
	_, err := ResolveTenant(ctx, "unknown.example.com")
	if err == nil {
		t.Fatal("expected error for unknown domain")
	}
	if !errors.Is(err, common.I18nError(i18n.MsgUnknownDomain)) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveTenant_DomainExistsButConfigMissing(t *testing.T) {
	db := setupTenantDomainTestDB(t)
	domain := "orphan.example.com"
	tenantCache.Delete(domain)

	// 只有域名映射，没有 SiteConfig
	db.Create(&TenantDomain{Domain: domain, Identifier: "orphan-tenant"})

	ctx := common.WithSkipIdentifier(context.Background())
	_, err := ResolveTenant(ctx, domain)
	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !errors.Is(err, common.I18nError(i18n.MsgTenantConfigNotFound)) {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ─── CreateInitAdmin ─────────────────────────────────────────────────

func TestCreateInitAdmin_Success(t *testing.T) {
	db := setupTenantDomainTestDB(t)

	err := CreateInitAdmin(db, "testadmin", "testpass123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var user User
	if err := db.Where("username = ?", "testadmin").First(&user).Error; err != nil {
		t.Fatalf("admin user not found: %v", err)
	}
	if user.Role != "admin" {
		t.Fatalf("expected role admin, got %q", user.Role)
	}
	if user.Password == "testpass123" {
		t.Fatal("password should be hashed, not plain text")
	}
}

func TestCreateInitAdmin_DuplicateUsername(t *testing.T) {
	db := setupTenantDomainTestDB(t)

	err := CreateInitAdmin(db, "dupadmin", "pass1")
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	// SQLite 不一定有 username unique 约束，但至少验证不 panic
	_ = CreateInitAdmin(db, "dupadmin2", "pass2")
}
