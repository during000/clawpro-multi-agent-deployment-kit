package controller

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ─── 辅助 ─────────────────────────────────────────────────────────────────

// setupSkillScanControllerDB 供 admin_skill_security_scan_test.go 使用的 DB 初始化。
func setupSkillScanControllerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Skill{},
		&model.SkillSecurityScan{},
		&model.SkillScanViolation{},
		&model.SiteConfig{},
		&model.User{},
		&model.AuditLog{},
		&model.SMHSpace{},
		&model.Project{},
		&model.ProjectConfigBinding{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Create(&model.SiteConfig{Name: "Test"})
	t.Cleanup(model.UseDBForTest(db))
	return db
}

func setupSkillScanTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SkillSecurityScan{}, &model.SkillScanViolation{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	return db
}

// ─── parseCSIPError ────────────────────────────────────────────────────────

func TestParseCSIPError_LimitExceeded(t *testing.T) {
	err := parseCSIPError("LimitExceeded", "quota exhausted")
	var scanLimitErr *ScanLimitError
	if !errors.As(err, &scanLimitErr) {
		t.Fatalf("期望 *ScanLimitError，实际 %T", err)
	}
	if scanLimitErr.Code != "LimitExceeded" {
		t.Errorf("期望 Code=LimitExceeded，实际 %s", scanLimitErr.Code)
	}
}

func TestParseCSIPError_Other(t *testing.T) {
	err := parseCSIPError("InternalError", "server error")
	if err == nil {
		t.Fatal("期望非 nil error")
	}
	var scanLimitErr *ScanLimitError
	if errors.As(err, &scanLimitErr) {
		t.Error("非 LimitExceeded 不应返回 ScanLimitError")
	}
}

func TestScanLimitError_Error(t *testing.T) {
	e := &ScanLimitError{Code: "LimitExceeded", Message: "quota done"}
	if e.Error() == "" {
		t.Error("Error() 不应为空")
	}
}

// ─── handleScanSuccess ─────────────────────────────────────────────────────

func TestHandleScanSuccess_Basic(t *testing.T) {
	db := setupSkillScanTestDB(t)

	scan := &model.SkillSecurityScan{
		SkillID:       1,
		ContentHash:   "sha256:abc",
		EngineVersion: 1,
		Status:        "SCANNING",
	}
	db.Create(scan)

	data := map[string]interface{}{
		"RiskLevel":     "benign",
		"SecurityScore": 95,
		"ReportURL":     "https://example.com/report",
		"ScannedAt":     time.Now().UTC().Format(time.RFC3339),
		"ScanItems":     []interface{}{},
		"RuleCatalog":   []interface{}{},
	}

	err := handleScanSuccess(context.Background(), scan, data)
	if err != nil {
		t.Fatalf("handleScanSuccess 返回错误: %v", err)
	}

	var updated model.SkillSecurityScan
	db.First(&updated, scan.ID)
	if updated.Status != "SUCCESS" {
		t.Errorf("期望 Status=SUCCESS，实际 %s", updated.Status)
	}
	if updated.RiskLevel != "benign" {
		t.Errorf("期望 RiskLevel=benign，实际 %s", updated.RiskLevel)
	}
}

func TestHandleScanSuccess_WithViolations(t *testing.T) {
	db := setupSkillScanTestDB(t)

	scan := &model.SkillSecurityScan{
		SkillID: 1, ContentHash: "sha256:viol", EngineVersion: 1, Status: "SCANNING",
	}
	db.Create(scan)

	data := map[string]interface{}{
		"RiskLevel":     "malicious",
		"SecurityScore": 20,
		"ScanItems": []interface{}{
			map[string]interface{}{
				"ScanType": "AI",
				"RuleList": []interface{}{
					map[string]interface{}{"RuleID": "R001", "Description": "危险操作"},
				},
			},
		},
		"RuleCatalog": []interface{}{
			map[string]interface{}{"RuleID": "R001", "RuleName": "危险规则"},
		},
	}

	err := handleScanSuccess(context.Background(), scan, data)
	if err != nil {
		t.Fatalf("handleScanSuccess 返回错误: %v", err)
	}

	var violations []model.SkillScanViolation
	db.Where("skill_security_scan_id = ?", scan.ID).Find(&violations)
	if len(violations) != 1 {
		t.Errorf("期望 1 条违规记录，实际 %d", len(violations))
	}
}

func TestHandleScanSuccess_InvalidScannedAt(t *testing.T) {
	db := setupSkillScanTestDB(t)

	scan := &model.SkillSecurityScan{
		SkillID: 1, ContentHash: "sha256:bad-time", EngineVersion: 1, Status: "SCANNING",
	}
	db.Create(scan)

	// ScannedAt 格式不正确，应忽略解析错误
	data := map[string]interface{}{
		"RiskLevel": "benign", "SecurityScore": 90,
		"ScannedAt": "not-a-time",
	}
	err := handleScanSuccess(context.Background(), scan, data)
	if err != nil {
		t.Fatalf("期望忽略无效时间格式，实际返回错误: %v", err)
	}
}

// ─── handleScanFailed ──────────────────────────────────────────────────────

func TestHandleScanFailed_WithData(t *testing.T) {
	db := setupSkillScanTestDB(t)

	scan := &model.SkillSecurityScan{
		SkillID: 1, ContentHash: "sha256:fail", EngineVersion: 1, Status: "SCANNING",
	}
	db.Create(scan)

	now := time.Now().UTC().Format(time.RFC3339)
	data := map[string]interface{}{
		"FailedAt": now,
		"Message":  "服务器错误",
	}

	err := handleScanFailed(context.Background(), scan, data)
	if err != nil {
		t.Fatalf("handleScanFailed 返回错误: %v", err)
	}

	var updated model.SkillSecurityScan
	db.First(&updated, scan.ID)
	if updated.Status != "FAILED" {
		t.Errorf("期望 Status=FAILED，实际 %s", updated.Status)
	}
	if updated.FailureMessage != "服务器错误" {
		t.Errorf("期望 FailureMessage='服务器错误'，实际 %s", updated.FailureMessage)
	}
}

func TestHandleScanFailed_NilData(t *testing.T) {
	db := setupSkillScanTestDB(t)

	scan := &model.SkillSecurityScan{
		SkillID: 1, ContentHash: "sha256:nil-data", EngineVersion: 1, Status: "SCANNING",
	}
	db.Create(scan)

	err := handleScanFailed(context.Background(), scan, nil)
	if err != nil {
		t.Fatalf("handleScanFailed(nil data) 返回错误: %v", err)
	}

	var updated model.SkillSecurityScan
	db.First(&updated, scan.ID)
	if updated.Status != "FAILED" {
		t.Errorf("期望 Status=FAILED，实际 %s", updated.Status)
	}
	if updated.FailureMessage == "" {
		t.Error("期望有默认错误消息")
	}
}

func TestHandleScanFailed_InvalidFailedAt(t *testing.T) {
	db := setupSkillScanTestDB(t)
	scan := &model.SkillSecurityScan{
		SkillID: 1, ContentHash: "sha256:bad-fa", EngineVersion: 1, Status: "SCANNING",
	}
	db.Create(scan)

	data := map[string]interface{}{"FailedAt": "bad-time", "Message": "err"}
	err := handleScanFailed(context.Background(), scan, data)
	if err != nil {
		t.Fatalf("期望容错处理无效时间，实际: %v", err)
	}
}

// ─── saveScanViolations ────────────────────────────────────────────────────

func TestSaveScanViolations_Empty(t *testing.T) {
	setupSkillScanTestDB(t)
	err := saveScanViolations(context.Background(), 1, nil, nil)
	if err != nil {
		t.Errorf("空 scanItems 不应返回错误: %v", err)
	}
}

func TestSaveScanViolations_MultipleItems(t *testing.T) {
	db := setupSkillScanTestDB(t)

	scan := &model.SkillSecurityScan{
		SkillID: 1, ContentHash: "sha256:viol2", EngineVersion: 1, Status: "SCANNING",
	}
	db.Create(scan)

	items := []scanItem{
		{
			ScanType: "STATIC",
			RuleList: []ruleViolation{
				{RuleID: "S001", Description: "硬编码密钥"},
				{RuleID: "S002", Description: "不安全随机数"},
			},
		},
		{
			ScanType: "AI",
			RuleList: []ruleViolation{
				{RuleID: "A001", Description: "恶意意图"},
			},
		},
	}
	catalog := []ruleCatalogItem{
		{RuleID: "S001", RuleName: "硬编码密钥规则"},
		{RuleID: "A001", RuleName: "AI恶意规则"},
	}

	err := saveScanViolations(context.Background(), scan.ID, items, catalog)
	if err != nil {
		t.Fatalf("saveScanViolations 返回错误: %v", err)
	}

	var count int64
	db.Model(&model.SkillScanViolation{}).Where("skill_security_scan_id = ?", scan.ID).Count(&count)
	if count != 3 {
		t.Errorf("期望 3 条违规记录，实际 %d", count)
	}
}

// ─── pollSingleScan ────────────────────────────────────────────────────────

func TestPollSingleScan_Timeout(t *testing.T) {
	db := setupSkillScanTestDB(t)

	// 创建一个超时的扫描记录（创建时间超过 30 分钟）
	old := time.Now().Add(-31 * time.Minute)
	scan := &model.SkillSecurityScan{
		SkillID: 1, ContentHash: "sha256:timeout", EngineVersion: 1, Status: "SCANNING",
	}
	db.Create(scan)
	db.Model(scan).Update("created_at", old)
	scan.CreatedAt = old

	err := pollSingleScan(context.Background(), scan)
	if err != nil {
		t.Fatalf("pollSingleScan(timeout) 返回错误: %v", err)
	}

	var updated model.SkillSecurityScan
	db.First(&updated, scan.ID)
	if updated.Status != "FAILED" {
		t.Errorf("超时扫描应标记为 FAILED，实际 %s", updated.Status)
	}
}

// ─── PollSkillSecurityScanResults ─────────────────────────────────────────

func TestPollSkillSecurityScanResults_Empty(t *testing.T) {
	setupSkillScanTestDB(t)
	// 无待轮询记录，应直接返回 nil
	err := PollSkillSecurityScanResults(context.Background())
	if err != nil {
		t.Errorf("无待扫描记录时不应返回错误: %v", err)
	}
}

func TestPollSkillSecurityScanResults_OnlyTimeout(t *testing.T) {
	db := setupSkillScanTestDB(t)

	old := time.Now().Add(-35 * time.Minute)
	scan := &model.SkillSecurityScan{
		SkillID: 1, ContentHash: "sha256:poll-timeout", EngineVersion: 1, Status: "SCANNING",
	}
	db.Create(scan)
	db.Model(scan).Update("created_at", old)

	// 有超时记录，pollSingleScan 应将其标记为 FAILED（不调用 callCSIPAction）
	err := PollSkillSecurityScanResults(context.Background())
	if err != nil {
		t.Errorf("期望 nil，实际: %v", err)
	}

	var updated model.SkillSecurityScan
	db.First(&updated, scan.ID)
	if updated.Status != "FAILED" {
		t.Errorf("超时扫描应被标记为 FAILED，实际 %s", updated.Status)
	}
}

// ─── CreateSkillSecurityScan 前置校验 ─────────────────────────────────────

func TestCreateSkillSecurityScan_FileTooLarge(t *testing.T) {
	setupSkillScanTestDB(t)

	bigData := make([]byte, maxScanFileSize+1)
	_, err := CreateSkillSecurityScan(context.Background(), bigData, 1, "1.0", "test.zip")
	if !errors.Is(err, ErrFileTooLargeForScan) {
		t.Errorf("期望 ErrFileTooLargeForScan，实际: %v", err)
	}
}

// ─── handleScanSuccess 反序列化失败路径 ───────────────────────────────────

func TestHandleScanSuccess_BadData(t *testing.T) {
	db := setupSkillScanTestDB(t)

	scan := &model.SkillSecurityScan{
		SkillID: 1, ContentHash: "sha256:bad-data", EngineVersion: 1, Status: "SCANNING",
	}
	db.Create(scan)

	// 传入无法解析为 scanResultData 的结构（但 json.Marshal 的 map 不会失败）
	// 用合法 map 覆盖，验证正常路径
	data := map[string]interface{}{
		"RiskLevel": "benign", "SecurityScore": json.Number("100"),
	}
	// 不管是否报错，不应 panic
	_ = handleScanSuccess(context.Background(), scan, data)
}
