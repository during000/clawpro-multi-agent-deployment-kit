package model

import (
	"context"
	"testing"
	"time"

	"hatchery/common"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupSkillSecurityScanTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	registerIdentifierCallbacks(db)
	// AutoMigrate 需要跳过 identifier 检查（DDL 不涉及数据隔离）
	migrateCtx := common.WithSkipIdentifier(context.Background())
	if err := db.WithContext(migrateCtx).AutoMigrate(&Skill{}, &SkillSecurityScan{}, &SkillScanViolation{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	oldDriver := dbDriver
	dbDriver = "sqlite"
	restoreDB := UseDBForTest(db)
	t.Cleanup(func() {
		restoreDB()
		dbDriver = oldDriver
	})
}

func scanTenantCtx(identifier string) context.Context {
	return common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: identifier})
}

func TestGetLatestSkillSecurityScanReturnsNewestInTenant(t *testing.T) {
	setupSkillSecurityScanTestDB(t)
	ctx := scanTenantCtx("tenant-latest")

	skill := Skill{Slug: "scan-skill", Name: "Scan Skill", Version: "1.0.0"}
	if err := DB(ctx).Create(&skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}
	oldCreated := time.Now().Add(-time.Hour)
	newCreated := time.Now()
	scans := []SkillSecurityScan{
		{SkillID: skill.ID, SkillVersion: "1.0.0", ContentHash: "sha256:old", EngineVersion: 1, Status: "SUCCESS", RiskLevel: "benign", CreatedAt: oldCreated},
		{SkillID: skill.ID, SkillVersion: "1.0.0", ContentHash: "sha256:new", EngineVersion: 1, Status: "SCANNING", CreatedAt: newCreated},
	}
	if err := DB(ctx).Create(&scans).Error; err != nil {
		t.Fatalf("create scans: %v", err)
	}

	got, err := GetLatestSkillSecurityScan(ctx, skill.ID)
	if err != nil {
		t.Fatalf("GetLatestSkillSecurityScan error: %v", err)
	}
	if got == nil || got.ContentHash != "sha256:new" || got.Status != "SCANNING" {
		t.Fatalf("latest scan = %#v, want newest SCANNING scan", got)
	}
}

func TestSkillSecurityScanQueriesRespectTenantIsolation(t *testing.T) {
	setupSkillSecurityScanTestDB(t)
	ctxA := scanTenantCtx("tenant-a")
	ctxB := scanTenantCtx("tenant-b")

	scanA := SkillSecurityScan{SkillID: 1, SkillVersion: "1.0.0", ContentHash: "sha256:same", EngineVersion: 7, Status: "SUCCESS", RiskLevel: "benign"}
	scanB := SkillSecurityScan{SkillID: 2, SkillVersion: "1.0.0", ContentHash: "sha256:same", EngineVersion: 7, Status: "SCANNING"}
	if err := DB(ctxA).Create(&scanA).Error; err != nil {
		t.Fatalf("create tenant A scan: %v", err)
	}
	if err := DB(ctxB).Create(&scanB).Error; err != nil {
		t.Fatalf("create tenant B scan with same hash+engine: %v", err)
	}

	gotA, err := GetSkillSecurityScanByHash(ctxA, "sha256:same", 7)
	if err != nil {
		t.Fatalf("tenant A hash query error: %v", err)
	}
	if gotA == nil || gotA.SkillID != 1 || gotA.Identifier != "tenant-a" {
		t.Fatalf("tenant A got %#v, want tenant-a skill 1", gotA)
	}

	gotB, err := GetSkillSecurityScanByHash(ctxB, "sha256:same", 7)
	if err != nil {
		t.Fatalf("tenant B hash query error: %v", err)
	}
	if gotB == nil || gotB.SkillID != 2 || gotB.Identifier != "tenant-b" {
		t.Fatalf("tenant B got %#v, want tenant-b skill 2", gotB)
	}
}

func TestGetPendingScanRecordsFiltersScanningAndTenant(t *testing.T) {
	setupSkillSecurityScanTestDB(t)
	ctxA := scanTenantCtx("tenant-pending-a")
	ctxB := scanTenantCtx("tenant-pending-b")

	if err := DB(ctxA).Create(&[]SkillSecurityScan{
		{SkillID: 1, SkillVersion: "1.0.0", ContentHash: "sha256:a1", EngineVersion: 1, Status: "SCANNING"},
		{SkillID: 2, SkillVersion: "1.0.0", ContentHash: "sha256:a2", EngineVersion: 1, Status: "SUCCESS"},
	}).Error; err != nil {
		t.Fatalf("create tenant A scans: %v", err)
	}
	if err := DB(ctxB).Create(&SkillSecurityScan{SkillID: 3, SkillVersion: "1.0.0", ContentHash: "sha256:b1", EngineVersion: 1, Status: "SCANNING"}).Error; err != nil {
		t.Fatalf("create tenant B scan: %v", err)
	}

	got, err := GetPendingScanRecords(ctxA)
	if err != nil {
		t.Fatalf("GetPendingScanRecords error: %v", err)
	}
	if len(got) != 1 || got[0].SkillID != 1 || got[0].Identifier != "tenant-pending-a" {
		t.Fatalf("pending scans = %#v, want only tenant A SCANNING record", got)
	}
}

func TestGetScanViolationsReturnsOrderedViolations(t *testing.T) {
	setupSkillSecurityScanTestDB(t)
	ctx := scanTenantCtx("tenant-violations")
	scan := SkillSecurityScan{SkillID: 1, SkillVersion: "1.0.0", ContentHash: "sha256:v", EngineVersion: 1, Status: "SUCCESS"}
	if err := DB(ctx).Create(&scan).Error; err != nil {
		t.Fatalf("create scan: %v", err)
	}
	violations := []SkillScanViolation{
		{SkillSecurityScanID: scan.ID, RuleID: "R2", RuleName: "Rule 2", ScanType: "STATIC", Description: "second"},
		{SkillSecurityScanID: scan.ID, RuleID: "R1", RuleName: "Rule 1", ScanType: "AI", Description: "first"},
	}
	if err := DB(ctx).Create(&violations).Error; err != nil {
		t.Fatalf("create violations: %v", err)
	}

	got, err := GetScanViolations(ctx, scan.ID)
	if err != nil {
		t.Fatalf("GetScanViolations error: %v", err)
	}
	if len(got) != 2 || got[0].RuleID != "R1" || got[1].RuleID != "R2" {
		t.Fatalf("violations = %#v, want ordered by rule_id", got)
	}
}

func TestGetSkillsSecurityStatusReturnsLatestPerSkill(t *testing.T) {
	setupSkillSecurityScanTestDB(t)
	ctx := scanTenantCtx("tenant-batch")

	oldCreated := time.Now().Add(-time.Hour)
	newCreated := time.Now()
	if err := DB(ctx).Create(&[]SkillSecurityScan{
		{SkillID: 10, SkillVersion: "1.0.0", ContentHash: "sha256:10-old", EngineVersion: 1, Status: "SUCCESS", RiskLevel: "benign", CreatedAt: oldCreated},
		{SkillID: 10, SkillVersion: "1.0.1", ContentHash: "sha256:10-new", EngineVersion: 1, Status: "SUCCESS", RiskLevel: "malicious", CreatedAt: newCreated},
		{SkillID: 20, SkillVersion: "1.0.0", ContentHash: "sha256:20", EngineVersion: 1, Status: "SCANNING"},
	}).Error; err != nil {
		t.Fatalf("create scans: %v", err)
	}

	got, err := GetSkillsSecurityStatus(ctx, []uint{10, 20, 30})
	if err != nil {
		t.Fatalf("GetSkillsSecurityStatus error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(status map) = %d, want 2", len(got))
	}
	if got[10] == nil || got[10].ContentHash != "sha256:10-new" || got[10].RiskLevel != "malicious" {
		t.Fatalf("skill 10 status = %#v, want latest malicious scan", got[10])
	}
	if got[20] == nil || got[20].Status != "SCANNING" {
		t.Fatalf("skill 20 status = %#v, want scanning", got[20])
	}
	if got[30] != nil {
		t.Fatalf("skill 30 status = %#v, want nil/missing", got[30])
	}
}

func TestGetSkillsSecurityStatusEmptyInput(t *testing.T) {
	setupSkillSecurityScanTestDB(t)
	got, err := GetSkillsSecurityStatus(scanTenantCtx("tenant-empty"), nil)
	if err != nil {
		t.Fatalf("GetSkillsSecurityStatus error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("empty input returned %#v, want empty map", got)
	}
}

func TestGetLatestSkillSecurityScanNotFound(t *testing.T) {
	setupSkillSecurityScanTestDB(t)
	ctx := scanTenantCtx("tenant-notfound")

	got, err := GetLatestSkillSecurityScan(ctx, 99999)
	if err != nil {
		t.Fatalf("GetLatestSkillSecurityScan error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for non-existent skill, got %#v", got)
	}
}

func TestGetSkillSecurityScanByHashNotFound(t *testing.T) {
	setupSkillSecurityScanTestDB(t)
	ctx := scanTenantCtx("tenant-hash-notfound")

	got, err := GetSkillSecurityScanByHash(ctx, "sha256:nonexistent", 999)
	if err != nil {
		t.Fatalf("GetSkillSecurityScanByHash error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for non-existent hash, got %#v", got)
	}
}

func TestGetPendingScanRecordsReturnsEmpty(t *testing.T) {
	setupSkillSecurityScanTestDB(t)
	ctx := scanTenantCtx("tenant-no-pending")

	got, err := GetPendingScanRecords(ctx)
	if err != nil {
		t.Fatalf("GetPendingScanRecords error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %d records", len(got))
	}
}

func TestGetScanViolationsEmpty(t *testing.T) {
	setupSkillSecurityScanTestDB(t)
	ctx := scanTenantCtx("tenant-no-violations")

	got, err := GetScanViolations(ctx, 99999)
	if err != nil {
		t.Fatalf("GetScanViolations error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %d records", len(got))
	}
}
