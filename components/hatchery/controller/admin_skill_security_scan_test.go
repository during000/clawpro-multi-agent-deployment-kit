package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/gorilla/sessions"
)

func setupAdminSkillScanHandlerTest(t *testing.T) {
	t.Helper()
	setupSkillScanControllerDB(t)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("skill-scan-test-secret-key"))
	t.Cleanup(func() { Store = origStore })
}

func newSkillScanAdminRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

func TestHandleSkillScanConfigGetAndPost(t *testing.T) {
	setupAdminSkillScanHandlerTest(t)
	defer withAdminToken("test-admin-token")()

	postReq := newSkillScanAdminRequest(http.MethodPost, "/admin/skills/scan-config", `{"skill_scan_default_enabled":true}`)
	postW := httptest.NewRecorder()
	HandleSkillScanConfigRouter(postW, postReq)
	if postW.Code != http.StatusOK {
		t.Fatalf("POST status = %d, body = %s", postW.Code, postW.Body.String())
	}
	var postResp map[string]interface{}
	if err := json.Unmarshal(postW.Body.Bytes(), &postResp); err != nil {
		t.Fatalf("decode post resp: %v", err)
	}
	if postResp["ok"] != true || postResp["skill_scan_default_enabled"] != true {
		t.Fatalf("post response = %#v", postResp)
	}

	getReq := newSkillScanAdminRequest(http.MethodGet, "/admin/skills/scan-config", "")
	getW := httptest.NewRecorder()
	HandleSkillScanConfigRouter(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body = %s", getW.Code, getW.Body.String())
	}
	var getResp map[string]interface{}
	if err := json.Unmarshal(getW.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get resp: %v", err)
	}
	if getResp["skill_scan_default_enabled"] != true {
		t.Fatalf("get response = %#v, want enabled=true", getResp)
	}

	postFalseReq := newSkillScanAdminRequest(http.MethodPost, "/admin/skills/scan-config", `{"skill_scan_default_enabled":false}`)
	postFalseW := httptest.NewRecorder()
	HandleSkillScanConfigRouter(postFalseW, postFalseReq)
	if postFalseW.Code != http.StatusOK {
		t.Fatalf("POST false status = %d, body = %s", postFalseW.Code, postFalseW.Body.String())
	}
	if config := model.GetSiteConfig(context.Background()); config.SkillScanDefaultEnabled {
		t.Fatalf("SkillScanDefaultEnabled = true, want false after POST false")
	}
}

func TestHandleSkillScanConfigInvalidJSON(t *testing.T) {
	setupAdminSkillScanHandlerTest(t)
	defer withAdminToken("test-admin-token")()

	req := newSkillScanAdminRequest(http.MethodPost, "/admin/skills/scan-config", `{bad json`)
	w := httptest.NewRecorder()
	HandleSkillScanConfigRouter(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s; want 400", w.Code, w.Body.String())
	}
}

func TestHandleSkillScanConfigUnauthorized(t *testing.T) {
	setupAdminSkillScanHandlerTest(t)
	defer withAdminToken("test-admin-token")()

	req := httptest.NewRequest(http.MethodGet, "/admin/skills/scan-config", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleSkillScanConfigRouter(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s; want 401", w.Code, w.Body.String())
	}
}

func TestHandleSkillScanConfigMethodNotAllowed(t *testing.T) {
	setupAdminSkillScanHandlerTest(t)
	defer withAdminToken("test-admin-token")()

	req := newSkillScanAdminRequest(http.MethodPut, "/admin/skills/scan-config", `{}`)
	w := httptest.NewRecorder()
	HandleSkillScanConfigRouter(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestHandleSkillScanTriggerInvalidJSON(t *testing.T) {
	setupAdminSkillScanHandlerTest(t)
	defer withAdminToken("test-admin-token")()

	req := newSkillScanAdminRequest(http.MethodPost, "/admin/skills/scan-trigger", `{bad json`)
	w := httptest.NewRecorder()
	HandleSkillScanTrigger(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s; want 400", w.Code, w.Body.String())
	}
}

func TestHandleSkillScanTriggerValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		seed      func(t *testing.T) uint
		wantCode  int
		wantError string
	}{
		{
			name:      "missing skill_id",
			body:      `{}`,
			wantCode:  http.StatusBadRequest,
			wantError: "skill_id",
		},
		{
			name:      "skill not found",
			body:      `{"skill_id":999}`,
			wantCode:  http.StatusNotFound,
			wantError: "技能不存在",
		},
		{
			name: "file too large",
			seed: func(t *testing.T) uint {
				return createSkillScanTriggerSkill(t, "too-large", int64(maxScanFileSize)+1)
			},
			wantCode:  http.StatusBadRequest,
			wantError: "7MB",
		},
		{
			name: "already scanning",
			seed: func(t *testing.T) uint {
				skillID := createSkillScanTriggerSkill(t, "already-scanning", 1024)
				scan := model.SkillSecurityScan{SkillID: skillID, SkillVersion: "1.0.0", ContentHash: "sha256:in-progress", EngineVersion: 1, Status: "SCANNING"}
				if err := model.DB(context.Background()).Create(&scan).Error; err != nil {
					t.Fatalf("create existing scan: %v", err)
				}
				return skillID
			},
			wantCode:  http.StatusConflict,
			wantError: "正在进行",
		},
		{
			name: "download url generation fails",
			seed: func(t *testing.T) uint {
				// 创建合法大小且不设 COSZipKey 的 skill，覆盖 cosZipKey 回退路径
				skill := model.Skill{
					Slug: "download-fail", Name: "download-fail",
					Description: "test", Version: "1.0.0", FileSize: 1024,
				}
				if err := skill.ParseVersion(); err != nil {
					t.Fatalf("parse version: %v", err)
				}
				if err := model.DB(context.Background()).Create(&skill).Error; err != nil {
					t.Fatalf("create skill: %v", err)
				}
				return skill.ID
			},
			wantCode:  http.StatusInternalServerError,
			wantError: "下载链接失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupAdminSkillScanHandlerTest(t)
			defer withAdminToken("test-admin-token")()

			body := tt.body
			if tt.seed != nil {
				skillID := tt.seed(t)
				body = `{"skill_id":` + strconv.FormatUint(uint64(skillID), 10) + `}`
			}
			req := newSkillScanAdminRequest(http.MethodPost, "/admin/skills/scan-trigger", body)
			w := httptest.NewRecorder()
			HandleSkillScanTrigger(w, req)
			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, body = %s; want %d", w.Code, w.Body.String(), tt.wantCode)
			}
			if tt.wantError != "" && !strings.Contains(w.Body.String(), tt.wantError) {
				t.Fatalf("body = %s, want containing %q", w.Body.String(), tt.wantError)
			}
		})
	}
}

func TestHandleSkillScanTriggerUnauthorized(t *testing.T) {
	setupAdminSkillScanHandlerTest(t)
	defer withAdminToken("test-admin-token")()

	req := httptest.NewRequest(http.MethodPost, "/admin/skills/scan-trigger", strings.NewReader(`{"skill_id":1}`))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleSkillScanTrigger(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s; want 401", w.Code, w.Body.String())
	}
}

func createSkillScanTriggerSkill(t *testing.T, slug string, fileSize int64) uint {
	t.Helper()
	skill := model.Skill{
		Slug:        slug,
		Name:        slug,
		Description: "test skill",
		Version:     "1.0.0",
		FileSize:    fileSize,
		COSZipKey:   slug + ".zip",
	}
	if err := skill.ParseVersion(); err != nil {
		t.Fatalf("parse version: %v", err)
	}
	if err := model.DB(context.Background()).Create(&skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}
	return skill.ID
}

func TestHandleSetSkillScanConfigUnauthorized(t *testing.T) {
	setupAdminSkillScanHandlerTest(t)
	defer withAdminToken("test-admin-token")()

	req := httptest.NewRequest(http.MethodPost, "/admin/skills/scan-config", strings.NewReader(`{"skill_scan_default_enabled":true}`))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header
	w := httptest.NewRecorder()
	HandleSetSkillScanConfig(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s; want 401", w.Code, w.Body.String())
	}
}

func TestHandleSkillScanConfigRouterDispatchesCorrectly(t *testing.T) {
	setupAdminSkillScanHandlerTest(t)
	defer withAdminToken("test-admin-token")()

	// DELETE should return 405
	req := newSkillScanAdminRequest(http.MethodDelete, "/admin/skills/scan-config", "")
	w := httptest.NewRecorder()
	HandleSkillScanConfigRouter(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE status = %d, want 405", w.Code)
	}
}

func TestBuildScanStatusRespScanningIncludesCreatedAt(t *testing.T) {
	now := time.Now().UTC()
	resp := buildScanStatusResp(&model.SkillSecurityScan{Status: "SCANNING", CreatedAt: now})
	if resp.ScanStatus != "scanning" {
		t.Fatalf("ScanStatus = %q, want scanning", resp.ScanStatus)
	}
	if resp.CreatedAt == nil || !resp.CreatedAt.Equal(now) {
		t.Fatalf("CreatedAt = %v, want %v", resp.CreatedAt, now)
	}
	if resp.SecurityScore != nil {
		t.Fatalf("SecurityScore should be nil for scanning, got %v", resp.SecurityScore)
	}
}

func TestBuildScanDetailRespScanningMinimalFields(t *testing.T) {
	now := time.Now().UTC()
	resp := buildScanDetailResp(&model.SkillSecurityScan{ID: 42, Status: "SCANNING", CreatedAt: now})
	if resp.ScanStatus != "scanning" || resp.ScanID != 42 {
		t.Fatalf("scanning detail resp = %#v", resp)
	}
	if resp.RiskDescription != "" || resp.ScanItems != nil {
		t.Fatalf("scanning should not have risk_description or scan_items")
	}
}

func TestBuildScanDetailRespNilScan(t *testing.T) {
	resp := buildScanDetailResp(nil)
	if resp.ScanStatus != "not_scanned" || resp.ScanID != 0 {
		t.Fatalf("nil detail resp = %#v", resp)
	}
}

func TestBuildScanDetailRespFailedStatus(t *testing.T) {
	now := time.Now().UTC()
	resp := buildScanDetailResp(&model.SkillSecurityScan{
		ID:             77,
		Status:         "FAILED",
		FailedAt:       &now,
		FailureMessage: "scan engine error",
		CreatedAt:      now,
	})
	if resp.ScanStatus != "not_scanned" {
		t.Fatalf("FAILED scan should map to not_scanned, got %q", resp.ScanStatus)
	}
	if resp.ScanID != 77 {
		t.Fatalf("ScanID = %d, want 77", resp.ScanID)
	}
}

// ── HandleAdminSkillDetail / HandleAdminSkillFiles 安全扫描字段覆盖 ──

func setupAdminSkillHandlerTestDB(t *testing.T) {
	t.Helper()
	setupSkillScanControllerDB(t)
	// 额外创建 HandleAdminSkillDetail/Files 所需的表
	db := model.DB(context.Background())
	db.AutoMigrate(
		&model.SkillCategory{},
		&model.SkillCategoryMapping{},
		&model.SkillVisibilityGroup{},
		&model.UserGroup{},
	)
	// 确保 SiteConfig 启用 SMH
	db.Model(&model.SiteConfig{}).Where("1 = 1").Update("smh_enabled", 1)
}

func TestHandleAdminSkillDetailIncludesSecurityScan(t *testing.T) {
	setupAdminSkillHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()
	ctx := context.Background()

	// 创建技能
	skill := model.Skill{
		Slug: "scan-detail-test", Name: "Scan Detail Test",
		Version: "1.0.0", Description: "test desc",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
	}
	if err := model.DB(ctx).Create(&skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}

	// 创建扫描记录
	scannedAt := time.Now().UTC()
	scan := model.SkillSecurityScan{
		SkillID:       skill.ID,
		SkillVersion:  "1.0.0",
		ContentHash:   "sha256:detail-test",
		EngineVersion: 1,
		Status:        "SUCCESS",
		RiskLevel:     "suspicious",
		SecurityScore: 65,
		ScannedAt:     &scannedAt,
		ReportURL:     "https://example.com/report",
	}
	if err := model.DB(ctx).Create(&scan).Error; err != nil {
		t.Fatalf("create scan: %v", err)
	}

	req := newSkillScanAdminRequest(http.MethodGet,
		"/admin/skills/detail?slug=scan-detail-test", "")
	w := httptest.NewRecorder()
	HandleAdminSkillDetail(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	skillData, ok := resp["skill"].(map[string]interface{})
	if !ok {
		t.Fatalf("response missing 'skill' field: %s", w.Body.String())
	}
	secScan, ok := skillData["security_scan"].(map[string]interface{})
	if !ok {
		t.Fatalf("response missing 'security_scan' field in skill: %s", w.Body.String())
	}
	if secScan["scan_status"] != "suspicious" {
		t.Fatalf("scan_status = %v, want suspicious", secScan["scan_status"])
	}
}

func TestHandleAdminSkillFilesIncludesSecurityScan(t *testing.T) {
	setupAdminSkillHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()
	ctx := context.Background()

	// 创建技能（两个版本）
	skills := []model.Skill{
		{Slug: "scan-files-test", Name: "Scan Files", Version: "1.1.0", Description: "latest",
			VersionMajor: 1, VersionMinor: 1, VersionPatch: 0, FileList: `["main.py"]`},
		{Slug: "scan-files-test", Name: "Scan Files", Version: "1.0.0", Description: "older",
			VersionMajor: 1, VersionMinor: 0, VersionPatch: 0, FileList: `["old.py"]`},
	}
	for i := range skills {
		if err := model.DB(ctx).Create(&skills[i]).Error; err != nil {
			t.Fatalf("create skill: %v", err)
		}
	}

	// 仅为最新版本创建扫描记录
	scan := model.SkillSecurityScan{
		SkillID:       skills[0].ID,
		SkillVersion:  "1.1.0",
		ContentHash:   "sha256:files-test",
		EngineVersion: 1,
		Status:        "SCANNING",
	}
	if err := model.DB(ctx).Create(&scan).Error; err != nil {
		t.Fatalf("create scan: %v", err)
	}

	req := newSkillScanAdminRequest(http.MethodGet,
		"/admin/skills/files?slug=scan-files-test", "")
	w := httptest.NewRecorder()
	HandleAdminSkillFiles(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	versions, ok := resp["versions"].([]interface{})
	if !ok || len(versions) == 0 {
		t.Fatalf("response missing versions: %s", w.Body.String())
	}
	// 最新版本应包含 security_scan
	firstVer := versions[0].(map[string]interface{})
	secScan := firstVer["security_scan"]
	if secScan == nil {
		t.Fatalf("latest version should have security_scan field: %s", w.Body.String())
	}
	secScanMap := secScan.(map[string]interface{})
	if secScanMap["scan_status"] != "scanning" {
		t.Fatalf("scan_status = %v, want scanning", secScanMap["scan_status"])
	}

	// 老版本也应有 security_scan 字段（无扫描记录时为 not_scanned）
	if len(versions) > 1 {
		secondVer := versions[1].(map[string]interface{})
		secScan2 := secondVer["security_scan"]
		if secScan2 == nil {
			t.Fatalf("older version should also have security_scan field")
		}
		secScanMap2 := secScan2.(map[string]interface{})
		if secScanMap2["scan_status"] != "not_scanned" {
			t.Fatalf("older version scan_status = %v, want not_scanned", secScanMap2["scan_status"])
		}
	}
}
