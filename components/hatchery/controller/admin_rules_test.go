package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// setupRulesTestDB 只挂载 rule 相关表 + 依赖表；对齐 setupSkillsFullTestDB 风格。
// 独立于 skill 侧 helper 是为了避免 rule 加表 / 加索引反过来污染现有 skill 测试。
func setupRulesTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&model.SiteConfig{},
		&model.SMHSpace{},
		&model.User{},
		&model.Instance{},
		&model.EnterpriseRule{},
		&model.RuleDistributionTask{},
		&model.RuleDistributionRecord{},
		&model.LocalInstanceRule{},
		&model.RuleVisibilityGroup{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		// project_visibility.go 查询 project_config_bindings（rule 类型可见性）
		&model.ProjectConfigBinding{},
		&model.Project{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	origDB := model.UseDBForTestWithDriver(db, "sqlite")
	db.Create(&model.SiteConfig{SMHEnabled: 1})

	origToken := AdminToken
	AdminToken = "test-admin-token"

	if Store == nil {
		Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	}

	var wg sync.WaitGroup
	skillDistributeWG = &wg

	t.Cleanup(func() {
		AdminToken = origToken
		wg.Wait()
		skillDistributeWG = nil
		origDB()
	})
}

// createTestRule 创建一个测试用 rule，返回创建结果。
func createTestRule(t *testing.T, slug, version, ruleType string) *model.EnterpriseRule {
	t.Helper()
	r := &model.EnterpriseRule{
		Slug:    slug,
		Name:    slug + " name",
		Type:    ruleType,
		Source:  model.EnterpriseRuleSourceEnterprise,
		Version: version,
		COSKey:  buildRuleCOSKey(slug, version),
	}
	if err := r.ParseVersion(); err != nil {
		t.Fatalf("parse version %s: %v", version, err)
	}
	if err := model.DB(context.Background()).Create(r).Error; err != nil {
		t.Fatalf("create test rule: %v", err)
	}
	return r
}

// buildMultipartRuleRequest 组装 /admin/rules/create 需要的 multipart 请求。
// filenameOverride 为空时按 "rule.md" 命名。
func buildMultipartRuleRequest(t *testing.T,
	fields map[string]string, fileContent []byte, filename string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for k, v := range fields {
		writer.WriteField(k, v)
	}
	if fileContent != nil {
		fw, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		fw.Write(fileContent)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/admin/rules/create", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// adminFormPost 用于表单编码的 delete 接口。
func adminFormPost(url, form string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// ────────────────────────────────────────────────────────────────────────────
// helpers 单元测试
// ────────────────────────────────────────────────────────────────────────────

func TestBuildRuleCOSKey(t *testing.T) {
	got := buildRuleCOSKey("coding-standards", "1.2.0")
	want := "enterprise-rules/coding-standards/1.2.0.md"
	if got != want {
		t.Fatalf("buildRuleCOSKey = %q, want %q", got, want)
	}
}

func TestValidateRuleType(t *testing.T) {
	if err := validateRuleType("prompt"); err != nil {
		t.Errorf("prompt should be valid: %v", err)
	}
	if err := validateRuleType("rule"); err != nil {
		t.Errorf("rule should be valid: %v", err)
	}
	if err := validateRuleType(""); err == nil {
		t.Errorf("empty type should be invalid")
	}
	if err := validateRuleType("unknown"); err == nil {
		t.Errorf("unknown type should be invalid")
	}
}

func TestValidateRuleFileContent(t *testing.T) {
	if err := validateRuleFileContent(nil); err == nil {
		t.Errorf("nil content should be invalid")
	}
	if err := validateRuleFileContent([]byte{}); err == nil {
		t.Errorf("empty content should be invalid")
	}
	if err := validateRuleFileContent([]byte("hello # md")); err != nil {
		t.Errorf("valid utf8 md should pass: %v", err)
	}
	// 含 \x00 应被拒
	if err := validateRuleFileContent([]byte{'a', 0, 'b'}); err == nil {
		t.Errorf("content with \\x00 should be invalid")
	}
	// 非 UTF-8 应被拒（\xff 不是 UTF-8 起始字节）
	if err := validateRuleFileContent([]byte{0xff, 0xfe, 0xfd}); err == nil {
		t.Errorf("non-UTF8 should be invalid")
	}
}

func TestSHA256Hex(t *testing.T) {
	// 空串的 sha256 = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	got := sha256Hex(nil)
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Fatalf("sha256Hex(nil) = %q, want %q", got, want)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// HandleAdminRules（列表）—— early-return + 空数据分支
// ────────────────────────────────────────────────────────────────────────────

func TestHandleAdminRules_EmptyList(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	HandleAdminRules(w, adminJSONGet("/admin/rules"))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Rules []map[string]interface{} `json:"rules"`
		Total int64                    `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 0 || len(resp.Rules) != 0 {
		t.Fatalf("want empty, got total=%d len=%d", resp.Total, len(resp.Rules))
	}
}

func TestHandleAdminRules_TypeFilter(t *testing.T) {
	setupRulesTestDB(t)
	createTestRule(t, "p1", "1.0.0", model.EnterpriseRuleTypePrompt)
	createTestRule(t, "r1", "1.0.0", model.EnterpriseRuleTypeRule)
	createTestRule(t, "r2", "1.0.0", model.EnterpriseRuleTypeRule)

	// 不带筛选 → 3 条
	w := httptest.NewRecorder()
	HandleAdminRules(w, adminJSONGet("/admin/rules"))
	var resp struct{ Total int64 }
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 3 {
		t.Fatalf("no filter: want 3, got %d", resp.Total)
	}

	// type=prompt → 1 条
	w = httptest.NewRecorder()
	HandleAdminRules(w, adminJSONGet("/admin/rules?type=prompt"))
	var resp2 struct{ Total int64 }
	json.Unmarshal(w.Body.Bytes(), &resp2)
	if resp2.Total != 1 {
		t.Fatalf("type=prompt: want 1, got %d", resp2.Total)
	}

	// type=rule → 2 条
	w = httptest.NewRecorder()
	HandleAdminRules(w, adminJSONGet("/admin/rules?type=rule"))
	var resp3 struct{ Total int64 }
	json.Unmarshal(w.Body.Bytes(), &resp3)
	if resp3.Total != 2 {
		t.Fatalf("type=rule: want 2, got %d", resp3.Total)
	}
}

func TestHandleAdminRules_LatestVersionOnly(t *testing.T) {
	setupRulesTestDB(t)
	// 同 slug 三个版本，列表只返回最新
	createTestRule(t, "coding", "1.0.0", model.EnterpriseRuleTypeRule)
	createTestRule(t, "coding", "1.1.0", model.EnterpriseRuleTypeRule)
	latest := createTestRule(t, "coding", "2.0.0", model.EnterpriseRuleTypeRule)

	w := httptest.NewRecorder()
	HandleAdminRules(w, adminJSONGet("/admin/rules"))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp struct {
		Rules []struct {
			ID      uint   `json:"id"`
			Version string `json:"version"`
		} `json:"rules"`
		Total int64 `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 || len(resp.Rules) != 1 {
		t.Fatalf("want 1 latest, got total=%d len=%d", resp.Total, len(resp.Rules))
	}
	if resp.Rules[0].ID != latest.ID || resp.Rules[0].Version != "2.0.0" {
		t.Fatalf("want latest id=%d ver=2.0.0, got id=%d ver=%s",
			latest.ID, resp.Rules[0].ID, resp.Rules[0].Version)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// HandleAdminRuleDetail
// ────────────────────────────────────────────────────────────────────────────

func TestHandleAdminRuleDetail_MissingSlug(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	HandleAdminRuleDetail(w, adminJSONGet("/admin/rules/detail"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestHandleAdminRuleDetail_NotFoundLatest(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	HandleAdminRuleDetail(w, adminJSONGet("/admin/rules/detail?slug=no-such"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestHandleAdminRuleDetail_NotFoundSpecificVersion(t *testing.T) {
	setupRulesTestDB(t)
	createTestRule(t, "s1", "1.0.0", model.EnterpriseRuleTypeRule)

	w := httptest.NewRecorder()
	HandleAdminRuleDetail(w, adminJSONGet("/admin/rules/detail?slug=s1&version=9.9.9"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestHandleAdminRuleDetail_ReturnsAllVersions(t *testing.T) {
	setupRulesTestDB(t)
	createTestRule(t, "s1", "1.0.0", model.EnterpriseRuleTypeRule)
	createTestRule(t, "s1", "1.1.0", model.EnterpriseRuleTypeRule)
	createTestRule(t, "s1", "2.0.0", model.EnterpriseRuleTypeRule)

	// version=latest → 拿到 2.0.0
	w := httptest.NewRecorder()
	HandleAdminRuleDetail(w, adminJSONGet("/admin/rules/detail?slug=s1&version=latest"))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp struct {
		Rule struct {
			Version string `json:"version"`
		} `json:"rule"`
		Versions []string `json:"versions"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Rule.Version != "2.0.0" {
		t.Errorf("latest: want 2.0.0, got %s", resp.Rule.Version)
	}
	if len(resp.Versions) != 3 {
		t.Errorf("versions: want 3 items, got %v", resp.Versions)
	}
	// 版本按降序排
	if resp.Versions[0] != "2.0.0" || resp.Versions[2] != "1.0.0" {
		t.Errorf("versions order: %v", resp.Versions)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// HandleCreateRule —— early-return 分支
// ────────────────────────────────────────────────────────────────────────────

func TestHandleCreateRule_MissingRequiredFields(t *testing.T) {
	setupRulesTestDB(t)
	req := buildMultipartRuleRequest(t, map[string]string{
		"slug": "", "name": "n", "version": "1.0.0", "type": "rule",
	}, []byte("hello"), "rule.md")

	w := httptest.NewRecorder()
	HandleCreateRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (empty slug), got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleCreateRule_InvalidSlug(t *testing.T) {
	setupRulesTestDB(t)
	req := buildMultipartRuleRequest(t, map[string]string{
		"slug": "INVALID_UPPER", "name": "n", "version": "1.0.0", "type": "rule",
	}, []byte("x"), "rule.md")
	w := httptest.NewRecorder()
	HandleCreateRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (invalid slug), got %d", w.Code)
	}
}

func TestHandleCreateRule_MissingType(t *testing.T) {
	setupRulesTestDB(t)
	req := buildMultipartRuleRequest(t, map[string]string{
		"slug": "abc", "name": "n", "version": "1.0.0",
	}, []byte("x"), "rule.md")
	w := httptest.NewRecorder()
	HandleCreateRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (missing type), got %d", w.Code)
	}
}

func TestHandleCreateRule_InvalidType(t *testing.T) {
	setupRulesTestDB(t)
	req := buildMultipartRuleRequest(t, map[string]string{
		"slug": "abc", "name": "n", "version": "1.0.0", "type": "unknown",
	}, []byte("x"), "rule.md")
	w := httptest.NewRecorder()
	HandleCreateRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (invalid type), got %d", w.Code)
	}
}

func TestHandleCreateRule_TypeMismatchWithFirstVersion(t *testing.T) {
	setupRulesTestDB(t)
	// 首版 type=rule
	createTestRule(t, "abc", "1.0.0", model.EnterpriseRuleTypeRule)

	// 后续版本改成 prompt → 拒绝
	req := buildMultipartRuleRequest(t, map[string]string{
		"slug": "abc", "name": "n", "version": "1.1.0", "type": "prompt",
	}, []byte("x"), "rule.md")
	w := httptest.NewRecorder()
	HandleCreateRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (type mismatch), got %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "type") {
		t.Errorf("body should mention type: %s", w.Body.String())
	}
}

func TestHandleCreateRule_VersionNotIncremented(t *testing.T) {
	setupRulesTestDB(t)
	createTestRule(t, "abc", "2.0.0", model.EnterpriseRuleTypeRule)

	// 尝试插 1.0.0 → 拒绝（新版本必须大于当前最高）
	req := buildMultipartRuleRequest(t, map[string]string{
		"slug": "abc", "name": "n", "version": "1.0.0", "type": "rule",
	}, []byte("x"), "rule.md")
	w := httptest.NewRecorder()
	HandleCreateRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (version not incremented), got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleCreateRule_VersionAlreadyExists(t *testing.T) {
	setupRulesTestDB(t)
	createTestRule(t, "abc", "1.0.0", model.EnterpriseRuleTypeRule)
	createTestRule(t, "abc", "2.0.0", model.EnterpriseRuleTypeRule)

	// 尝试插重复的 2.0.0 → 版本递增校验先命中，仍 400
	req := buildMultipartRuleRequest(t, map[string]string{
		"slug": "abc", "name": "n", "version": "2.0.0", "type": "rule",
	}, []byte("x"), "rule.md")
	w := httptest.NewRecorder()
	HandleCreateRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestHandleCreateRule_MissingFile(t *testing.T) {
	setupRulesTestDB(t)
	req := buildMultipartRuleRequest(t, map[string]string{
		"slug": "abc", "name": "n", "version": "1.0.0", "type": "rule",
	}, nil, "")
	w := httptest.NewRecorder()
	HandleCreateRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (missing file), got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleCreateRule_FileNotMD(t *testing.T) {
	setupRulesTestDB(t)
	req := buildMultipartRuleRequest(t, map[string]string{
		"slug": "abc", "name": "n", "version": "1.0.0", "type": "rule",
	}, []byte("hello"), "rule.txt")
	w := httptest.NewRecorder()
	HandleCreateRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (not md), got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleCreateRule_FileTooLarge(t *testing.T) {
	setupRulesTestDB(t)
	// 构造 > 1 MiB 的 body 触发 ParseMultipartForm 上限
	big := make([]byte, maxRuleUploadSize+2048)
	for i := range big {
		big[i] = 'x'
	}
	req := buildMultipartRuleRequest(t, map[string]string{
		"slug": "abc", "name": "n", "version": "1.0.0", "type": "rule",
	}, big, "rule.md")
	w := httptest.NewRecorder()
	HandleCreateRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (file too large), got %d", w.Code)
	}
}

func TestHandleCreateRule_FileContentInvalid(t *testing.T) {
	setupRulesTestDB(t)
	req := buildMultipartRuleRequest(t, map[string]string{
		"slug": "abc", "name": "n", "version": "1.0.0", "type": "rule",
	}, []byte{0xff, 0xfe, 0xfd}, "rule.md") // 非 UTF-8
	w := httptest.NewRecorder()
	HandleCreateRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (invalid content), got %d, body=%s", w.Code, w.Body.String())
	}
}

// ────────────────────────────────────────────────────────────────────────────
// HandleDeleteRule
// ────────────────────────────────────────────────────────────────────────────

func TestHandleDeleteRule_MissingSlug(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	HandleDeleteRule(w, adminFormPost("/admin/rules/delete", ""))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestHandleDeleteRule_SingleVersionNotFound(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	HandleDeleteRule(w, adminFormPost("/admin/rules/delete", "slug=no-such&version=1.0.0"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestHandleDeleteRule_AllVersionsNotFound(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	HandleDeleteRule(w, adminFormPost("/admin/rules/delete", "slug=no-such"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestHandleDeleteRule_RunningTaskBlocks(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "abc", "1.0.0", model.EnterpriseRuleTypeRule)

	// 建一个 running task
	if err := model.DB(context.Background()).Create(&model.RuleDistributionTask{
		RuleID: rule.ID, Slug: "abc", RuleType: "rule", Version: "1.0.0",
		Status: "running", Type: model.RuleTaskTypeDistribute, Total: 1,
	}).Error; err != nil {
		t.Fatalf("seed task: %v", err)
	}

	w := httptest.NewRecorder()
	HandleDeleteRule(w, adminFormPost("/admin/rules/delete", "slug=abc"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (running task blocks), got %d, body=%s", w.Code, w.Body.String())
	}
	// 记录未删
	var count int64
	model.DB(context.Background()).Model(&model.EnterpriseRule{}).Where("slug = ?", "abc").Count(&count)
	if count != 1 {
		t.Errorf("rule should not be deleted while task running, got count=%d", count)
	}
}

func TestHandleDeleteRule_AllVersions_Success(t *testing.T) {
	setupRulesTestDB(t)
	createTestRule(t, "abc", "1.0.0", model.EnterpriseRuleTypeRule)
	createTestRule(t, "abc", "1.1.0", model.EnterpriseRuleTypeRule)
	createTestRule(t, "abc", "2.0.0", model.EnterpriseRuleTypeRule)

	w := httptest.NewRecorder()
	HandleDeleteRule(w, adminFormPost("/admin/rules/delete", "slug=abc"))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		OK           bool `json:"ok"`
		DeletedRules int  `json:"deleted_rules"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.OK || resp.DeletedRules != 3 {
		t.Fatalf("want ok=true deleted=3, got %+v", resp)
	}

	// 全部软删
	var alive int64
	model.DB(context.Background()).Model(&model.EnterpriseRule{}).Where("slug = ?", "abc").Count(&alive)
	if alive != 0 {
		t.Errorf("want 0 alive after delete-all, got %d", alive)
	}
	// Unscoped 能查到 3 条
	var withDeleted int64
	model.DB(context.Background()).Unscoped().Model(&model.EnterpriseRule{}).Where("slug = ?", "abc").Count(&withDeleted)
	if withDeleted != 3 {
		t.Errorf("want 3 total including deleted, got %d", withDeleted)
	}
}

func TestHandleDeleteRule_SingleVersion_Success(t *testing.T) {
	setupRulesTestDB(t)
	createTestRule(t, "abc", "1.0.0", model.EnterpriseRuleTypeRule)
	createTestRule(t, "abc", "2.0.0", model.EnterpriseRuleTypeRule)

	w := httptest.NewRecorder()
	HandleDeleteRule(w, adminFormPost("/admin/rules/delete", "slug=abc&version=1.0.0"))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d, body=%s", w.Code, w.Body.String())
	}

	// 1.0.0 被删，2.0.0 保留
	var alive int64
	model.DB(context.Background()).Model(&model.EnterpriseRule{}).Where("slug = ?", "abc").Count(&alive)
	if alive != 1 {
		t.Errorf("want 1 alive, got %d", alive)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// HandleAdminRuleFiles
// ────────────────────────────────────────────────────────────────────────────

func TestHandleAdminRuleFiles_MissingSlug(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	HandleAdminRuleFiles(w, adminJSONGet("/admin/rules/files"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestHandleAdminRuleFiles_NotFound(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	HandleAdminRuleFiles(w, adminJSONGet("/admin/rules/files?slug=no-such"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestHandleAdminRuleFiles_ReturnsVersions(t *testing.T) {
	setupRulesTestDB(t)
	// 3 个版本，SMH 未配置：download_url 会为空但接口本身不失败（走 warn 日志）
	createTestRule(t, "abc", "1.0.0", model.EnterpriseRuleTypeRule)
	createTestRule(t, "abc", "1.1.0", model.EnterpriseRuleTypeRule)
	createTestRule(t, "abc", "2.0.0", model.EnterpriseRuleTypeRule)

	w := httptest.NewRecorder()
	HandleAdminRuleFiles(w, adminJSONGet("/admin/rules/files?slug=abc"))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Slug     string `json:"slug"`
		Versions []struct {
			Version string `json:"version"`
		} `json:"versions"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Slug != "abc" {
		t.Errorf("slug: want abc, got %s", resp.Slug)
	}
	if len(resp.Versions) != 3 {
		t.Fatalf("versions: want 3, got %d", len(resp.Versions))
	}
	// 版本按降序
	if resp.Versions[0].Version != "2.0.0" || resp.Versions[2].Version != "1.0.0" {
		t.Errorf("version order: %v", resp.Versions)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// visibility + changelog（追加：对齐 skill 侧特性）
// ────────────────────────────────────────────────────────────────────────────

// createRuleTestGroup 在规范测试库里插入一个用户分组，返回主键。
func createRuleTestGroup(t *testing.T, name string) uint {
	t.Helper()
	g := &model.UserGroup{Name: name, ParentID: 0, Source: "manual"}
	if err := model.DB(context.Background()).Create(g).Error; err != nil {
		t.Fatalf("create group %s: %v", name, err)
	}
	return g.ID
}

// TestHandleAdminRules_VisibilityGroupsInResponse 列表响应含 visibility_groups 字段
func TestHandleAdminRules_VisibilityGroupsInResponse(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "vis1", "1.0.0", model.EnterpriseRuleTypeRule)
	// 手工设置为 group 可见
	gid := createRuleTestGroup(t, "team-A")
	tx := model.DB(context.Background())
	tx.Model(&model.EnterpriseRule{}).Where("id = ?", rule.ID).Update("visibility_type", model.VisibilityGroup)
	tx.Create(&model.RuleVisibilityGroup{RuleID: rule.ID, GroupID: gid})

	w := httptest.NewRecorder()
	HandleAdminRules(w, adminJSONGet("/admin/rules"))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp struct {
		Rules []struct {
			ID               uint `json:"id"`
			VisibilityGroups []struct {
				GroupID   uint   `json:"group_id"`
				GroupName string `json:"group_name"`
			} `json:"visibility_groups"`
		} `json:"rules"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Rules) != 1 {
		t.Fatalf("want 1 rule, got %d", len(resp.Rules))
	}
	vg := resp.Rules[0].VisibilityGroups
	if len(vg) != 1 || vg[0].GroupID != gid || vg[0].GroupName != "team-A" {
		t.Fatalf("want [{gid=%d, name=team-A}], got %+v", gid, vg)
	}
}

// TestHandleAdminRules_GroupIDFilter 按 group_id 筛选
func TestHandleAdminRules_GroupIDFilter(t *testing.T) {
	setupRulesTestDB(t)
	// 三条 rule：r1 挂 team-A、r2 挂 team-B、r3 全局
	r1 := createTestRule(t, "r1", "1.0.0", model.EnterpriseRuleTypeRule)
	r2 := createTestRule(t, "r2", "1.0.0", model.EnterpriseRuleTypeRule)
	createTestRule(t, "r3", "1.0.0", model.EnterpriseRuleTypeRule)
	gA := createRuleTestGroup(t, "team-A")
	gB := createRuleTestGroup(t, "team-B")
	tx := model.DB(context.Background())
	tx.Model(&model.EnterpriseRule{}).Where("id = ?", r1.ID).Update("visibility_type", model.VisibilityGroup)
	tx.Model(&model.EnterpriseRule{}).Where("id = ?", r2.ID).Update("visibility_type", model.VisibilityGroup)
	tx.Create(&model.RuleVisibilityGroup{RuleID: r1.ID, GroupID: gA})
	tx.Create(&model.RuleVisibilityGroup{RuleID: r2.ID, GroupID: gB})

	// 只查 group_id=team-A → 返回 r1
	w := httptest.NewRecorder()
	HandleAdminRules(w, adminJSONGet("/admin/rules?group_id="+uintStr(gA)))
	var resp struct {
		Rules []struct {
			ID uint `json:"id"`
		} `json:"rules"`
		Total int64 `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 || resp.Rules[0].ID != r1.ID {
		t.Fatalf("group_id=%d: want 1 rule id=%d, got total=%d rules=%+v",
			gA, r1.ID, resp.Total, resp.Rules)
	}

	// visibility_type=all + group_id=team-A → r1 + r3
	w = httptest.NewRecorder()
	HandleAdminRules(w, adminJSONGet("/admin/rules?visibility_type=all&group_id="+uintStr(gA)))
	var resp2 struct{ Total int64 }
	json.Unmarshal(w.Body.Bytes(), &resp2)
	if resp2.Total != 2 {
		t.Fatalf("visibility_type=all + group_id: want 2, got %d", resp2.Total)
	}
}

func TestHandleAdminRules_GroupAndProjectFiltersUseUnion(t *testing.T) {
	setupRulesTestDB(t)
	groupID := createRuleTestGroup(t, "项目与分组联合筛选组")
	project := model.Project{Name: "项目与分组联合筛选项目"}
	if err := model.DB(context.Background()).Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	groupRule := createTestRule(t, "union-group-rule", "1.0.0", model.EnterpriseRuleTypeRule)
	projectRule := createTestRule(t, "union-project-rule", "1.0.0", model.EnterpriseRuleTypeRule)
	tx := model.DB(context.Background())
	tx.Model(&model.EnterpriseRule{}).Where("id = ?", groupRule.ID).Update("visibility_type", model.VisibilityGroup)
	if err := tx.Create(&model.RuleVisibilityGroup{RuleID: groupRule.ID, GroupID: groupID}).Error; err != nil {
		t.Fatalf("create group binding: %v", err)
	}
	if err := tx.Create(&model.ProjectConfigBinding{ProjectID: project.ID, ConfigType: model.ProjectConfigTypeRule, ConfigKey: projectRule.Slug}).Error; err != nil {
		t.Fatalf("create project binding: %v", err)
	}

	w := httptest.NewRecorder()
	HandleAdminRules(w, adminJSONGet(fmt.Sprintf("/admin/rules?group_id=%d&project_id=%d", groupID, project.ID)))
	var resp struct {
		Total int64 `json:"total"`
		Rules []struct {
			Slug string `json:"slug"`
		} `json:"rules"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Fatalf("分组和项目筛选应返回并集 2 条，实际=%d body=%s", resp.Total, w.Body.String())
	}
	slugs := map[string]bool{}
	for _, rule := range resp.Rules {
		slugs[rule.Slug] = true
	}
	if !slugs[groupRule.Slug] || !slugs[projectRule.Slug] {
		t.Fatalf("筛选结果应同时包含分组和项目命中项，实际=%v", slugs)
	}
}

// TestHandleAdminRuleDetail_VisibilityGroups 详情返回 visibility_groups
func TestHandleAdminRuleDetail_VisibilityGroups(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "d1", "1.0.0", model.EnterpriseRuleTypeRule)
	gid := createRuleTestGroup(t, "team-Q")
	tx := model.DB(context.Background())
	tx.Model(&model.EnterpriseRule{}).Where("id = ?", rule.ID).Update("visibility_type", model.VisibilityGroup)
	tx.Create(&model.RuleVisibilityGroup{RuleID: rule.ID, GroupID: gid})

	w := httptest.NewRecorder()
	HandleAdminRuleDetail(w, adminJSONGet("/admin/rules/detail?slug=d1"))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Rule struct {
			VisibilityType string `json:"visibility_type"`
		} `json:"rule"`
		VisibilityGroups []struct {
			GroupID   uint   `json:"group_id"`
			GroupName string `json:"group_name"`
		} `json:"visibility_groups"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Rule.VisibilityType != model.VisibilityGroup {
		t.Errorf("visibility_type: want group, got %s", resp.Rule.VisibilityType)
	}
	if len(resp.VisibilityGroups) != 1 || resp.VisibilityGroups[0].GroupID != gid ||
		resp.VisibilityGroups[0].GroupName != "team-Q" {
		t.Errorf("visibility_groups: want [{%d, team-Q}], got %+v", gid, resp.VisibilityGroups)
	}
}

// TestHandleCreateRule_ChangelogPersisted create 时 changelog 字段落库
func TestHandleCreateRule_ChangelogPersisted(t *testing.T) {
	setupRulesTestDB(t)
	// 首版
	createTestRule(t, "cl1", "1.0.0", model.EnterpriseRuleTypeRule)

	// 手动通过 DB 修改一个已存在 rule 的 changelog（模拟 create 成功后的效果不需 SMH）
	// 这里更严格的做法是让 handler 走通，但 handler 依赖 SMH。作为 fallback，
	// 我们至少验证 model 层字段读写没问题（回归防止 gorm tag 打错）。
	if err := model.DB(context.Background()).
		Model(&model.EnterpriseRule{}).
		Where("slug = ? AND version = ?", "cl1", "1.0.0").
		Update("changelog", "首版：基础规则").Error; err != nil {
		t.Fatalf("update changelog: %v", err)
	}

	w := httptest.NewRecorder()
	HandleAdminRuleDetail(w, adminJSONGet("/admin/rules/detail?slug=cl1"))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp struct {
		Rule struct {
			Changelog string `json:"changelog"`
		} `json:"rule"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Rule.Changelog != "首版：基础规则" {
		t.Errorf("changelog: want '首版：基础规则', got %q", resp.Rule.Changelog)
	}
}

// TestHandleCreateRule_InvalidVisibilityType 传非法 visibility_type 应 400
func TestHandleCreateRule_InvalidVisibilityType(t *testing.T) {
	setupRulesTestDB(t)
	req := buildMultipartRuleRequest(t, map[string]string{
		"slug": "abc", "name": "n", "version": "1.0.0", "type": "rule",
		"visibility_type": "unknown-vt",
	}, []byte("hello"), "rule.md")
	w := httptest.NewRecorder()
	HandleCreateRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (invalid visibility_type), got %d, body=%s", w.Code, w.Body.String())
	}
}

// TestHandleCreateRule_VisibilityGroupWithoutGroupIDs group 类型必须传 group_ids
func TestHandleCreateRule_VisibilityGroupWithoutGroupIDs(t *testing.T) {
	setupRulesTestDB(t)
	req := buildMultipartRuleRequest(t, map[string]string{
		"slug": "abc", "name": "n", "version": "1.0.0", "type": "rule",
		"visibility_type": "group",
		// 故意不传 group_ids
	}, []byte("hello"), "rule.md")
	w := httptest.NewRecorder()
	HandleCreateRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (group type without group_ids), got %d, body=%s", w.Code, w.Body.String())
	}
}

// TestHandleCreateRule_VisibilityGroupInvalidGroupID group_ids 中包含不存在的 gid
func TestHandleCreateRule_VisibilityGroupInvalidGroupID(t *testing.T) {
	setupRulesTestDB(t)
	req := buildMultipartRuleRequest(t, map[string]string{
		"slug": "abc", "name": "n", "version": "1.0.0", "type": "rule",
		"visibility_type": "group",
		"group_ids":       "9999", // 不存在
	}, []byte("hello"), "rule.md")
	w := httptest.NewRecorder()
	HandleCreateRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (nonexistent group_id), got %d, body=%s", w.Code, w.Body.String())
	}
}

// TestHandleDeleteRule_CleanupVisibility delete 应清理 rule_visibility_groups
func TestHandleDeleteRule_CleanupVisibility(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "delvis", "1.0.0", model.EnterpriseRuleTypeRule)
	gid := createRuleTestGroup(t, "team-Del")
	tx := model.DB(context.Background())
	tx.Model(&model.EnterpriseRule{}).Where("id = ?", rule.ID).Update("visibility_type", model.VisibilityGroup)
	tx.Create(&model.RuleVisibilityGroup{RuleID: rule.ID, GroupID: gid})

	// 确认关联存在
	var before int64
	tx.Model(&model.RuleVisibilityGroup{}).Where("rule_id = ?", rule.ID).Count(&before)
	if before != 1 {
		t.Fatalf("precondition: want 1 assoc, got %d", before)
	}

	w := httptest.NewRecorder()
	HandleDeleteRule(w, adminFormPost("/admin/rules/delete", "slug=delvis"))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d, body=%s", w.Code, w.Body.String())
	}

	// 关联应被清理
	var after int64
	tx.Model(&model.RuleVisibilityGroup{}).Where("rule_id = ?", rule.ID).Count(&after)
	if after != 0 {
		t.Errorf("want 0 assoc after delete, got %d", after)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// HandleAdminRuleUpdate 测试
// ────────────────────────────────────────────────────────────────────────────

func TestHandleAdminRuleUpdate_MissingParams(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	HandleAdminRuleUpdate(w, adminFormPost("/admin/rules/update", "slug=myslug"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
	w = httptest.NewRecorder()
	HandleAdminRuleUpdate(w, adminFormPost("/admin/rules/update", "version=1.0.0"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

func TestHandleAdminRuleUpdate_NotFound(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	HandleAdminRuleUpdate(w, adminFormPost("/admin/rules/update", "slug=nonexistent&version=1.0.0"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

func TestHandleAdminRuleUpdate_Name(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "update-name", "1.0.0", "rule")

	w := httptest.NewRecorder()
	HandleAdminRuleUpdate(w, adminFormPost("/admin/rules/update",
		"slug=update-name&version=1.0.0&name=NewName"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	var updated model.EnterpriseRule
	model.DB(context.Background()).First(&updated, rule.ID)
	if updated.Name != "NewName" {
		t.Errorf("name 应更新为 NewName，实际=%s", updated.Name)
	}
}

func TestHandleAdminRuleUpdate_Description(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "update-desc", "1.0.0", "rule")

	w := httptest.NewRecorder()
	HandleAdminRuleUpdate(w, adminFormPost("/admin/rules/update",
		"slug=update-desc&version=1.0.0&description=NewDesc"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	var updated model.EnterpriseRule
	model.DB(context.Background()).First(&updated, rule.ID)
	if updated.Description != "NewDesc" {
		t.Errorf("description 应更新为 NewDesc，实际=%s", updated.Description)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// HandleAdminRuleTasks 测试
// ────────────────────────────────────────────────────────────────────────────

func TestHandleAdminRuleTasks_MissingParams(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/rules/tasks", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminRuleTasks(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

func TestHandleAdminRuleTasks_SlugNotFound(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/rules/tasks?slug=nonexistent", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminRuleTasks(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

func TestHandleAdminRuleTasks_ReturnsTasks(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "task-rule", "1.0.0", "rule")

	task := model.RuleDistributionTask{
		RuleID: rule.ID, Slug: rule.Slug, RuleType: rule.Type,
		Version: rule.Version, OperatorID: 1,
		Total: 2, Status: "completed", Type: "distribute",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	// 加两条 record
	for i := 0; i < 2; i++ {
		rec := model.RuleDistributionRecord{
			TaskID: task.ID, RuleID: rule.ID, InstanceID: uint(i + 1),
			InstanceCID: "local-test-" + string(rune('a'+i)),
			Version:     rule.Version, Status: model.RuleRecordStatusSuccess, Type: "distribute",
		}
		if err := model.DB(context.Background()).Create(&rec).Error; err != nil {
			t.Fatalf("create record: %v", err)
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/rules/tasks?slug=task-rule", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminRuleTasks(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Tasks []model.RuleDistributionTask `json:"tasks"`
		Total int                          `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if resp.Total != 1 {
		t.Fatalf("期望 1 个 task，实际 total=%d", resp.Total)
	}
	if len(resp.Tasks) != 1 {
		t.Fatalf("期望 1 个 task，实际 len=%d", len(resp.Tasks))
	}
	if resp.Tasks[0].Success != 2 {
		t.Errorf("success 应=2，实际=%d", resp.Tasks[0].Success)
	}

	// 验证 records 明细已返回（与 /admin/skills/tasks 对齐）
	var rawResp struct {
		Tasks []struct {
			ID      uint `json:"id"`
			Records []struct {
				InstanceID    uint   `json:"instance_id"`
				CVMInstanceID string `json:"cvm_instance_id"`
				Status        string `json:"status"`
			} `json:"records"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rawResp); err != nil {
		t.Fatalf("decode raw: %v body=%s", err, w.Body.String())
	}
	if len(rawResp.Tasks) != 1 {
		t.Fatalf("期望 1 个 task，实际 len=%d", len(rawResp.Tasks))
	}
	if len(rawResp.Tasks[0].Records) != 2 {
		t.Fatalf("期望 2 条 records，实际=%d", len(rawResp.Tasks[0].Records))
	}
	if rawResp.Tasks[0].Records[0].CVMInstanceID == "" {
		t.Errorf("record 的 cvm_instance_id 不应为空")
	}
}

func TestHandleAdminRuleTasks_ByBatchID(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "batch-rule", "1.0.0", "rule")

	task := model.RuleDistributionTask{
		RuleID: rule.ID, Slug: rule.Slug, RuleType: rule.Type,
		Version: "1.0.0", OperatorID: 1,
		Total: 1, Status: "running", Type: "distribute",
		BatchID: "batch-001",
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/rules/tasks?batch_id=batch-001", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminRuleTasks(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if resp.Total != 1 {
		t.Fatalf("期望 1 个 task，实际 total=%d", resp.Total)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// HandleAdminRuleInstances 测试
// ────────────────────────────────────────────────────────────────────────────

func TestHandleAdminRuleInstances_MissingSlug(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/rules/instances", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminRuleInstances(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

func TestHandleAdminRuleInstances_SlugNotFound(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/rules/instances?slug=nonexistent", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminRuleInstances(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

func TestHandleAdminRuleInstances_ReturnsRecords(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "inst-rule", "1.0.0", "rule")

	// 建用户和实例
	user := &model.User{Username: "u-inst", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst-rule-box", InstanceId: "local-inst-rule-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(inst)

	task := model.RuleDistributionTask{
		RuleID: rule.ID, Slug: rule.Slug, RuleType: rule.Type,
		Version: rule.Version, OperatorID: 1,
		Total: 1, Status: "completed", Type: "distribute",
	}
	model.DB(context.Background()).Create(&task)

	rec := model.RuleDistributionRecord{
		TaskID: task.ID, RuleID: rule.ID, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Version: rule.Version,
		Status: model.RuleRecordStatusSuccess, Type: "distribute",
	}
	model.DB(context.Background()).Create(&rec)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/rules/instances?slug=inst-rule", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminRuleInstances(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Slug  string `json:"slug"`
		Total int    `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if resp.Total != 1 {
		t.Fatalf("期望 1 条记录，实际 total=%d body=%s", resp.Total, w.Body.String())
	}
	if resp.Slug != "inst-rule" {
		t.Errorf("slug 应=inst-rule，实际=%s", resp.Slug)
	}
}

func TestHandleAdminRuleInstances_StatusFilter(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "filter-rule", "1.0.0", "rule")

	user := &model.User{Username: "u-filter", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "filter-box", InstanceId: "local-filter-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(inst)

	task := model.RuleDistributionTask{
		RuleID: rule.ID, Slug: rule.Slug, RuleType: rule.Type,
		Version: rule.Version, OperatorID: 1,
		Total: 2, Status: "completed", Type: "distribute",
	}
	model.DB(context.Background()).Create(&task)

	// 一条 success，一条 failed
	for _, s := range []string{model.RuleRecordStatusSuccess, model.RuleRecordStatusFailed} {
		rec := model.RuleDistributionRecord{
			TaskID: task.ID, RuleID: rule.ID, InstanceID: inst.ID,
			InstanceCID: inst.InstanceId, Version: rule.Version,
			Status: s, Type: "distribute",
		}
		model.DB(context.Background()).Create(&rec)
	}

	// 新模型下状态由 SQL CASE 推导：success record → installed，failed record → failed。
	// 同一实例的最新一条 record 决定当前状态，这里两条 record 同实例，
	// MAX(id) 取 failed 那条 → 该实例 status=failed。
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/admin/rules/instances?slug=filter-rule&status=failed", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminRuleInstances(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if resp.Total != 1 {
		t.Fatalf("status=failed 过滤后应返回 1 条，实际 total=%d body=%s", resp.Total, w.Body.String())
	}
}

// TestHandleAdminRuleInstances_GroupIDFilter 验证按 group_id 筛选实例。
// 重点覆盖：同一用户跨多个组时，结果不能重复（半连接实现，与 skill/mcp 的 JOIN 老 bug 无关）。
func TestHandleAdminRuleInstances_GroupIDFilter(t *testing.T) {
	setupRulesTestDB(t)
	createTestRule(t, "grp-rule", "1.0.0", "rule")

	tx := model.DB(context.Background())

	gA := createRuleTestGroup(t, "grp-A")
	gB := createRuleTestGroup(t, "grp-B")

	// 用户 uA 仅属于 gA
	uA := &model.User{Username: "u-grp-a", Password: "x", Role: "user"}
	tx.Create(uA)
	tx.Create(&model.UserGroupMember{UserID: uA.ID, UserGroupID: gA})
	instA := &model.Instance{
		Name: "grp-box-a", InstanceId: "local-grp-a-001",
		UserID: uA.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	tx.Create(instA)

	// 用户 uCross 同时属于 gA 和 gB —— 这是去重的关键用例
	uCross := &model.User{Username: "u-grp-cross", Password: "x", Role: "user"}
	tx.Create(uCross)
	tx.Create(&model.UserGroupMember{UserID: uCross.ID, UserGroupID: gA})
	tx.Create(&model.UserGroupMember{UserID: uCross.ID, UserGroupID: gB})
	instCross := &model.Instance{
		Name: "grp-box-cross", InstanceId: "local-grp-cross-001",
		UserID: uCross.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	tx.Create(instCross)

	// 用户 uB 仅属于 gB
	uB := &model.User{Username: "u-grp-b", Password: "x", Role: "user"}
	tx.Create(uB)
	tx.Create(&model.UserGroupMember{UserID: uB.ID, UserGroupID: gB})
	instB := &model.Instance{
		Name: "grp-box-b", InstanceId: "local-grp-b-001",
		UserID: uB.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	tx.Create(instB)

	// 仅查 group_id=gA → 应返回 uA 与 uCross 两个实例（注意 uCross 只算一次）
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/admin/rules/instances?slug=grp-rule&group_id="+uintStr(gA), nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminRuleInstances(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	var respA struct {
		Total     int `json:"total"`
		Instances []struct {
			InstanceCID string `json:"cvm_instance_id"`
		} `json:"instances"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &respA); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if respA.Total != 2 {
		t.Fatalf("group_id=gA 应返回 2 条（uCross 不能重复），实际 total=%d body=%s",
			respA.Total, w.Body.String())
	}

	// 同时查 group_id=gA,gB（前端跨组多选）→ 应返回 3 个实例，且 uCross 仅一次。
	// 半连接实现下 uCross 只匹配一次 user_id，不会被 JOIN 扇出成多行。
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet,
		"/admin/rules/instances?slug=grp-rule&group_id="+uintStr(gA)+","+uintStr(gB), nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminRuleInstances(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	var respAB struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &respAB); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if respAB.Total != 3 {
		t.Fatalf("group_id=gA,gB 应返回 3 条（跨组用户不得重复），实际 total=%d body=%s",
			respAB.Total, w.Body.String())
	}
}

// TestHandleAdminRuleInstances_GroupIDUngrouped 验证 group_id=0 筛选未分组用户实例。
func TestHandleAdminRuleInstances_GroupIDUngrouped(t *testing.T) {
	setupRulesTestDB(t)
	createTestRule(t, "ungrp-rule", "1.0.0", "rule")

	tx := model.DB(context.Background())

	// 未分组用户
	uNoGroup := &model.User{Username: "u-no-group", Password: "x", Role: "user"}
	tx.Create(uNoGroup)
	instNoGroup := &model.Instance{
		Name: "ungrp-box", InstanceId: "local-ungrp-001",
		UserID: uNoGroup.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	tx.Create(instNoGroup)

	// 已分组用户
	gA := createRuleTestGroup(t, "ungrp-A")
	uGrouped := &model.User{Username: "u-grouped", Password: "x", Role: "user"}
	tx.Create(uGrouped)
	tx.Create(&model.UserGroupMember{UserID: uGrouped.ID, UserGroupID: gA})
	tx.Create(&model.Instance{
		Name: "ungrp-box-grouped", InstanceId: "local-ungrp-002",
		UserID: uGrouped.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	})

	// group_id=0 → 仅返回未分组用户的实例
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/admin/rules/instances?slug=ungrp-rule&group_id=0", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminRuleInstances(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if resp.Total != 1 {
		t.Fatalf("group_id=0 应仅返回未分组用户实例 1 条，实际 total=%d body=%s",
			resp.Total, w.Body.String())
	}
}

// TestHandleAdminRuleInstances_NeverInstalled_ReturnsUninstalled 验证从未下发过的
// 本地 agent 实例，在 status=uninstalled 筛选下能正确返回（对齐 skill 版行为）。
func TestHandleAdminRuleInstances_NeverInstalled_ReturnsUninstalled(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "never-rule", "1.0.0", "rule")

	// 本地实例，但从未建过任何 rule_distribution_records
	user := &model.User{Username: "u-never", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "never-box", InstanceId: "local-never-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(inst)

	// 另一个本地实例建了 success record（用于对照）
	user2 := &model.User{Username: "u-never2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user2)
	inst2 := &model.Instance{
		Name: "never-box-2", InstanceId: "local-never-002",
		UserID: user2.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(inst2)
	task := model.RuleDistributionTask{
		RuleID: rule.ID, Slug: rule.Slug, RuleType: rule.Type,
		Version: rule.Version, OperatorID: 1,
		Total: 1, Status: "completed", Type: "distribute",
	}
	model.DB(context.Background()).Create(&task)
	model.DB(context.Background()).Create(&model.RuleDistributionRecord{
		TaskID: task.ID, RuleID: rule.ID, InstanceID: inst2.ID,
		InstanceCID: inst2.InstanceId, Version: rule.Version,
		Status: model.RuleRecordStatusSuccess, Type: "distribute",
	})
	// inst2 真正装着：补写 local_instance_rules 事实快照，使其状态为 installed
	now := time.Now()
	model.DB(context.Background()).Create(&model.LocalInstanceRule{
		InstanceID: inst2.ID, Slug: rule.Slug, Version: rule.Version,
		DisplayName: rule.Name, RuleType: rule.Type,
		Source: model.LocalRuleSourceEnterprise, InstalledAt: &now, LastSeenAt: &now,
	})

	// status=uninstalled 应只命中从未安装的 inst（total=1）
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/admin/rules/instances?slug=never-rule&status=uninstalled", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminRuleInstances(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Total     int `json:"total"`
		Instances []struct {
			InstanceID uint   `json:"instance_id"`
			Status     string `json:"status"`
		} `json:"instances"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if resp.Total != 1 {
		t.Fatalf("status=uninstalled 应返回 1 条，实际 total=%d body=%s", resp.Total, w.Body.String())
	}
	if resp.Instances[0].InstanceID != inst.ID {
		t.Errorf("期望返回从未安装的实例 id=%d，实际 id=%d", inst.ID, resp.Instances[0].InstanceID)
	}
	if resp.Instances[0].Status != "uninstalled" {
		t.Errorf("期望 status=uninstalled，实际=%s", resp.Instances[0].Status)
	}
}

// TestHandleAdminRuleInstances_MultiStatusFilter 验证逗号分隔多状态筛选，
// 特别是 status=uninstalled,failed 能同时命中从未安装与安装失败的实例。
func TestHandleAdminRuleInstances_MultiStatusFilter(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "multi-rule", "1.0.0", "rule")

	user := &model.User{Username: "u-multi", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// instA：从未安装（无 record）
	instA := &model.Instance{
		Name: "multi-a", InstanceId: "local-multi-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(instA)

	// instB：安装失败（failed record）
	instB := &model.Instance{
		Name: "multi-b", InstanceId: "local-multi-002",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(instB)

	// instC：安装成功（installed record，补写 local_instance_rules 事实快照，
	// 使其推导为 installed，不应命中 uninstalled,failed）
	instC := &model.Instance{
		Name: "multi-c", InstanceId: "local-multi-003",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(instC)

	task := model.RuleDistributionTask{
		RuleID: rule.ID, Slug: rule.Slug, RuleType: rule.Type,
		Version: rule.Version, OperatorID: 1,
		Total: 2, Status: "completed", Type: "distribute",
	}
	model.DB(context.Background()).Create(&task)
	for _, pair := range []struct {
		inst *model.Instance
		st   string
	}{
		{instB, model.RuleRecordStatusFailed},
		{instC, model.RuleRecordStatusSuccess},
	} {
		model.DB(context.Background()).Create(&model.RuleDistributionRecord{
			TaskID: task.ID, RuleID: rule.ID, InstanceID: pair.inst.ID,
			InstanceCID: pair.inst.InstanceId, Version: rule.Version,
			Status: pair.st, Type: "distribute",
		})
	}
	// instC 真正装着：补写 local_instance_rules 事实快照
	nowC := time.Now()
	model.DB(context.Background()).Create(&model.LocalInstanceRule{
		InstanceID: instC.ID, Slug: rule.Slug, Version: rule.Version,
		DisplayName: rule.Name, RuleType: rule.Type,
		Source: model.LocalRuleSourceEnterprise, InstalledAt: &nowC, LastSeenAt: &nowC,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/admin/rules/instances?slug=multi-rule&status=uninstalled,failed", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminRuleInstances(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if resp.Total != 2 {
		t.Fatalf("status=uninstalled,failed 应返回 2 条，实际 total=%d body=%s", resp.Total, w.Body.String())
	}
}

// TestHandleAdminRuleInstances_LocalManualUninstall_ReturnsUninstalled 验证本地 agent
// 实例在 records 显示已安装、但 local_instance_rules 快照已消失（用户本地手动卸载）时，
// 被改判为 uninstalled（对齐 skill 版的本地事实校验分支）。
func TestHandleAdminRuleInstances_LocalManualUninstall_ReturnsUninstalled(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "manual-rule", "1.0.0", "rule")

	user := &model.User{Username: "u-manual", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "manual-box", InstanceId: "local-manual-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(inst)

	task := model.RuleDistributionTask{
		RuleID: rule.ID, Slug: rule.Slug, RuleType: rule.Type,
		Version: rule.Version, OperatorID: 1,
		Total: 1, Status: "completed", Type: "distribute",
	}
	model.DB(context.Background()).Create(&task)
	// records 显示 success（装上过），但没有对应的 local_instance_rules 行
	model.DB(context.Background()).Create(&model.RuleDistributionRecord{
		TaskID: task.ID, RuleID: rule.ID, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Version: rule.Version,
		Status: model.RuleRecordStatusSuccess, Type: "distribute",
	})

	// 不写 local_instance_rules → 应改判为 uninstalled
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/admin/rules/instances?slug=manual-rule&status=uninstalled", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminRuleInstances(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if resp.Total != 1 {
		t.Fatalf("本地手动卸载应改判 uninstalled，返回 1 条，实际 total=%d body=%s", resp.Total, w.Body.String())
	}
}

// TestHandleAdminRuleInstances_MultiScopeLIR_NoDuplicate 回归用例：
// 同一 slug 在同一本地实例上可同时存在于 user / project 两个 scope 的 local_instance_rules
// （见 LocalInstanceRule 唯一约束 (scope, instance_id, workspace_path, slug)）。
// BuildRuleInstanceQuery 之前的 JOIN 按 (instance_id, slug) 直接 LEFT JOIN，会把这个实例
// 扇出成多行，导致 /admin/rules/instances 返回重复数据（与 skill/mcp 的 JOIN 扇出老 bug 同类）。
// 修复后改用 MAX(id) 聚合子查询，每个 (instance_id, slug) 至多匹配一行 lir，应只返回 1 条实例。
func TestHandleAdminRuleInstances_MultiScopeLIR_NoDuplicate(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "multiscope-rule", "1.0.0", "rule")

	user := &model.User{Username: "u-multiscope", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "multiscope-box", InstanceId: "local-multiscope-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(inst)

	task := model.RuleDistributionTask{
		RuleID: rule.ID, Slug: rule.Slug, RuleType: rule.Type,
		Version: rule.Version, OperatorID: 1,
		Total: 1, Status: "completed", Type: "distribute",
	}
	model.DB(context.Background()).Create(&task)
	model.DB(context.Background()).Create(&model.RuleDistributionRecord{
		TaskID: task.ID, RuleID: rule.ID, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Version: rule.Version,
		Status: model.RuleRecordStatusSuccess, Type: "distribute",
	})
	// 同一 slug 在同一实例上写两行不同 scope 的本地事实快照（user + project）。
	now := time.Now()
	for _, sc := range []string{model.LocalSkillScopeUser, model.LocalSkillScopeWorkspace} {
		wp := ""
		if sc == model.LocalSkillScopeWorkspace {
			wp = "/home/alex/proj1"
		}
		model.DB(context.Background()).Create(&model.LocalInstanceRule{
			InstanceID:    inst.ID,
			Slug:          rule.Slug,
			Version:       rule.Version,
			DisplayName:   rule.Name,
			RuleType:      rule.Type,
			Source:        model.LocalRuleSourceEnterprise,
			Scope:         sc,
			WorkspacePath: wp,
			InstallStatus: model.LocalSkillInstallStatusDistributed,
			InstalledAt:   &now,
			LastSeenAt:    &now,
		})
	}

	// 全量查询：无论状态筛选，同一实例不得重复出现。
	for _, q := range []string{
		"/admin/rules/instances?slug=multiscope-rule",
		"/admin/rules/instances?slug=multiscope-rule&status=installed",
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, q, nil)
		req.Header.Set("Authorization", "Bearer test-admin-token")
		HandleAdminRuleInstances(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Total     int `json:"total"`
			Instances []struct {
				InstanceID uint   `json:"instance_id"`
				Status     string `json:"status"`
			} `json:"instances"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v body=%s", err, w.Body.String())
		}
		if resp.Total != 1 {
			t.Fatalf("query=%s 同一实例不应重复，应返回 1 条，实际 total=%d body=%s", q, resp.Total, w.Body.String())
		}
		if resp.Instances[0].InstanceID != inst.ID {
			t.Errorf("query=%s 期望实例 id=%d，实际 id=%d", q, inst.ID, resp.Instances[0].InstanceID)
		}
		if resp.Instances[0].Status != "installed" {
			t.Errorf("query=%s 期望 status=installed（有 success record + lir 事实快照），实际=%s", q, resp.Instances[0].Status)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// HandleDistributeRule 测试
// ────────────────────────────────────────────────────────────────────────────

func TestHandleDistributeRule_MissingSlug(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/distribute",
		strings.NewReader(`{"instance_ids":[1]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleDistributeRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

func TestHandleDistributeRule_EmptyInstances(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/distribute",
		strings.NewReader(`{"slug":"test","instance_ids":[]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleDistributeRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

func TestHandleDistributeRule_NoValidInstances(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/distribute",
		strings.NewReader(`{"slug":"test","instance_ids":[999]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleDistributeRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDistributeRule_Success(t *testing.T) {
	setupRulesTestDB(t)
	createTestRule(t, "dist-rule", "1.0.0", "rule")

	user := &model.User{Username: "u-dist", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "dist-box", InstanceId: "local-dist-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(inst)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/distribute",
		strings.NewReader(fmt.Sprintf(`{"slug":"dist-rule","instance_ids":[%d]}`, inst.ID)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleDistributeRule(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		OK      bool   `json:"ok"`
		TaskID  uint   `json:"task_id"`
		Slug    string `json:"slug"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if !resp.OK {
		t.Fatal("期望 ok=true")
	}
	if resp.TaskID == 0 {
		t.Fatal("期望 task_id > 0")
	}
	if resp.Slug != "dist-rule" {
		t.Errorf("slug 应=dist-rule，实际=%s", resp.Slug)
	}

	// 验证 task 和 record 已创建
	var task model.RuleDistributionTask
	if model.DB(context.Background()).First(&task, resp.TaskID).Error != nil {
		t.Fatal("task 未创建")
	}
	if task.Status != "completed" {
		t.Errorf("task status 应=completed，实际=%s", task.Status)
	}
	var count int64
	model.DB(context.Background()).Model(&model.RuleDistributionRecord{}).
		Where("task_id = ?", task.ID).Count(&count)
	if count != 1 {
		t.Errorf("期望 1 条 record，实际=%d", count)
	}
}

// TestHandleDistributeRule_LongSlugLockKey
// slug 很长时，原始 lock key 拼上 tenant identifier 前缀会超过 MySQL GET_LOCK 的 64 字符上限。
// 改动后用 sha256(slug) 前 16 位 hex 代替原始 slug，固定长度避免超长。
// SQLite 下 AcquireLock 返回空壳锁，本测试主要验证哈希路径不破坏正常下发流程。
func TestHandleDistributeRule_LongSlugLockKey(t *testing.T) {
	setupRulesTestDB(t)
	longSlug := "frontend-react-rules-1-4-0-frontend-react-rules-1-4-0-extra-long-suffix"
	createTestRule(t, longSlug, "2.0.0", "rule")

	user := &model.User{Username: "u-dist-long", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "dist-box-long", InstanceId: "local-dist-long-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(inst)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/distribute",
		strings.NewReader(fmt.Sprintf(`{"slug":%q,"version":"2.0.0","instance_ids":[%d]}`, longSlug, inst.ID)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleDistributeRule(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("长 slug 期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		OK     bool   `json:"ok"`
		TaskID uint   `json:"task_id"`
		Slug   string `json:"slug"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if !resp.OK || resp.TaskID == 0 {
		t.Fatalf("长 slug 下发应成功，resp=%+v", resp)
	}
	if resp.Slug != longSlug {
		t.Errorf("slug 应=%s，实际=%s", longSlug, resp.Slug)
	}
}

// TestHandleDistributeRule_SupersedePendingPrevious
// 若上一次同 slug 的 distribute 仍有 pending 记录（reporter 还没拉走），
// 新一次下发应把上一次任务判失败、其 pending 记录置 failed，再创建本次新任务的 pending。
func TestHandleDistributeRule_SupersedePendingPrevious(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "sup-rule", "1.0.0", "rule")

	user := &model.User{Username: "u-sup", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "sup-box", InstanceId: "local-sup-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(inst)

	// 模拟上一次下发：reporter 还没拉走，task 已 completed 但 record 仍 pending
	prevTask := model.RuleDistributionTask{
		RuleID: rule.ID, Slug: rule.Slug, RuleType: rule.Type, Version: rule.Version,
		OperatorID: user.ID, Total: 1, Status: "completed", Type: model.RuleTaskTypeDistribute,
	}
	model.DB(context.Background()).Create(&prevTask)
	prevRec := model.RuleDistributionRecord{
		TaskID: prevTask.ID, RuleID: rule.ID, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Version: rule.Version,
		Status: model.RuleRecordStatusPending, Type: model.RuleTaskTypeDistribute,
	}
	model.DB(context.Background()).Create(&prevRec)

	// 发起新一次下发（同 slug 同实例）
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/distribute",
		strings.NewReader(fmt.Sprintf(`{"slug":"sup-rule","instance_ids":[%d]}`, inst.ID)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleDistributeRule(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		TaskID uint `json:"task_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if resp.TaskID == 0 || resp.TaskID == prevTask.ID {
		t.Fatalf("新任务应已创建且不同于上一次，task_id=%d prev=%d", resp.TaskID, prevTask.ID)
	}

	// 上一次任务本身不应被改（task 可能给多个本地 agent 下发，其它可能已成功）
	var gotPrev model.RuleDistributionTask
	if model.DB(context.Background()).First(&gotPrev, prevTask.ID).Error != nil {
		t.Fatal("上一次任务不存在")
	}
	if gotPrev.Status != "completed" {
		t.Errorf("上一次任务不应被改，应 completed，实际=%s", gotPrev.Status)
	}
	// 上一次任务的 failed 计数应累加本次判失败的 pending 数量（此处为 1）
	if gotPrev.Failed != 1 {
		t.Errorf("上一次任务 failed 计数应为 1（累加 pending 数），实际=%d", gotPrev.Failed)
	}

	// 上一次 record 应 failed（原因：已下发新的版本）
	var gotPrevRec model.RuleDistributionRecord
	if model.DB(context.Background()).First(&gotPrevRec, prevRec.ID).Error != nil {
		t.Fatal("上一次 record 不存在")
	}
	if gotPrevRec.Status != model.RuleRecordStatusFailed {
		t.Errorf("上一次 record 应 failed，实际=%s", gotPrevRec.Status)
	}
	if gotPrevRec.Error != "已下发新的版本" {
		t.Errorf("上一次 record error 应为「已下发新的版本」，实际=%s", gotPrevRec.Error)
	}

	// 新任务应有 1 条 pending record
	var newPending int64
	model.DB(context.Background()).Model(&model.RuleDistributionRecord{}).
		Where("task_id = ? AND status = ?", resp.TaskID, model.RuleRecordStatusPending).Count(&newPending)
	if newPending != 1 {
		t.Errorf("新任务应剩 1 条 pending record，实际=%d", newPending)
	}
}

// TestHandleDistributeRule_NoSupersedeWhenPreviousDone
// 上一次下发已完成（无 pending 记录），新一次下发不应误判上一次失败。
func TestHandleDistributeRule_NoSupersedeWhenPreviousDone(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "nodup-rule", "1.0.0", "rule")

	user := &model.User{Username: "u-nodup", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "nodup-box", InstanceId: "local-nodup-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(inst)

	// 上一次下发：record 已 success（reporter 已 ack），无 pending
	prevTask := model.RuleDistributionTask{
		RuleID: rule.ID, Slug: rule.Slug, RuleType: rule.Type, Version: rule.Version,
		OperatorID: user.ID, Total: 1, Success: 1, Status: "completed", Type: model.RuleTaskTypeDistribute,
	}
	model.DB(context.Background()).Create(&prevTask)
	prevRec := model.RuleDistributionRecord{
		TaskID: prevTask.ID, RuleID: rule.ID, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Version: rule.Version,
		Status: model.RuleRecordStatusSuccess, Type: model.RuleTaskTypeDistribute,
	}
	model.DB(context.Background()).Create(&prevRec)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/distribute",
		strings.NewReader(fmt.Sprintf(`{"slug":"nodup-rule","instance_ids":[%d]}`, inst.ID)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleDistributeRule(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	var gotPrev model.RuleDistributionTask
	if model.DB(context.Background()).First(&gotPrev, prevTask.ID).Error != nil {
		t.Fatal("上一次任务不存在")
	}
	if gotPrev.Status != "completed" {
		t.Errorf("上一次任务不应被改，应 completed，实际=%s", gotPrev.Status)
	}
}

// TestHandleDistributeRule_SupersedeOnlySameInstance
// 核心规则：只有本次请求涉及的「相同 instance_id」的 pending 记录才被置失败。
// 上一次任务给实例 A、B 都下发了且都 pending；本次只给实例 A 下发——
// 则 A 被置失败、上一次任务 failed+1，而 B 的 pending 必须保留（不被本次误判）。
func TestHandleDistributeRule_SupersedeOnlySameInstance(t *testing.T) {
	setupRulesTestDB(t)
	rule := createTestRule(t, "sup-multi-rule", "1.0.0", "rule")

	user := &model.User{Username: "u-sup-multi", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instA := &model.Instance{
		Name: "sup-multi-a", InstanceId: "local-sup-multi-a",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	instB := &model.Instance{
		Name: "sup-multi-b", InstanceId: "local-sup-multi-b",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(instA)
	model.DB(context.Background()).Create(instB)

	// 上一次任务给 A、B 都下发，且都 pending（reporter 还没拉走）
	prevTask := model.RuleDistributionTask{
		RuleID: rule.ID, Slug: rule.Slug, RuleType: rule.Type, Version: rule.Version,
		OperatorID: user.ID, Total: 2, Status: "completed", Type: model.RuleTaskTypeDistribute,
	}
	model.DB(context.Background()).Create(&prevTask)
	recA := model.RuleDistributionRecord{
		TaskID: prevTask.ID, RuleID: rule.ID, InstanceID: instA.ID,
		InstanceCID: instA.InstanceId, Version: rule.Version,
		Status: model.RuleRecordStatusPending, Type: model.RuleTaskTypeDistribute,
	}
	recB := model.RuleDistributionRecord{
		TaskID: prevTask.ID, RuleID: rule.ID, InstanceID: instB.ID,
		InstanceCID: instB.InstanceId, Version: rule.Version,
		Status: model.RuleRecordStatusPending, Type: model.RuleTaskTypeDistribute,
	}
	model.DB(context.Background()).Create(&recA)
	model.DB(context.Background()).Create(&recB)

	// 本次只给 A 下发
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/distribute",
		strings.NewReader(fmt.Sprintf(`{"slug":"sup-multi-rule","instance_ids":[%d]}`, instA.ID)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleDistributeRule(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	// 上一次任务 failed 计数应只累加本次涉及的实例数（仅 A = 1）
	var gotPrev model.RuleDistributionTask
	if model.DB(context.Background()).First(&gotPrev, prevTask.ID).Error != nil {
		t.Fatal("上一次任务不存在")
	}
	if gotPrev.Failed != 1 {
		t.Errorf("上一次任务 failed 计数应为 1，实际=%d", gotPrev.Failed)
	}

	// A 的 pending 应被置 failed
	var gotA model.RuleDistributionRecord
	if model.DB(context.Background()).First(&gotA, recA.ID).Error != nil {
		t.Fatal("A record 不存在")
	}
	if gotA.Status != model.RuleRecordStatusFailed {
		t.Errorf("A 的 pending 应被置 failed，实际=%s", gotA.Status)
	}
	if gotA.Error != "已下发新的版本" {
		t.Errorf("A record error 应为「已下发新的版本」，实际=%s", gotA.Error)
	}

	// B 的 pending 必须保留（本次没涉及 B，不能被误判）
	var gotB model.RuleDistributionRecord
	if model.DB(context.Background()).First(&gotB, recB.ID).Error != nil {
		t.Fatal("B record 不存在")
	}
	if gotB.Status != model.RuleRecordStatusPending {
		t.Errorf("B 的 pending 不应被本次误判，应保持 pending，实际=%s", gotB.Status)
	}

	// 新任务只给 A 创建 1 条 pending record
	var resp struct {
		TaskID uint `json:"task_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if resp.TaskID == 0 || resp.TaskID == prevTask.ID {
		t.Fatalf("新任务应已创建且不同于上一次，task_id=%d prev=%d", resp.TaskID, prevTask.ID)
	}
	var newPending int64
	model.DB(context.Background()).Model(&model.RuleDistributionRecord{}).
		Where("task_id = ? AND status = ?", resp.TaskID, model.RuleRecordStatusPending).Count(&newPending)
	if newPending != 1 {
		t.Errorf("新任务应只有 1 条 pending record（仅 A），实际=%d", newPending)
	}
}

func TestHandleDistributeRule_NotFound(t *testing.T) {
	setupRulesTestDB(t)
	// 先创建一个 local 实例，确保能通过实例过滤
	user := &model.User{Username: "u-nf", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "nf-box", InstanceId: "local-nf-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(inst)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/distribute",
		strings.NewReader(fmt.Sprintf(`{"slug":"nonexistent","instance_ids":[%d]}`, inst.ID)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleDistributeRule(w, req)
	// distributeRuleBatch 采用批量模式：rule 不存在时不会直接返回 error，
	// 而是返回 200 + ok:true，但 task_id=0、version="" 表示未成功提交。
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("期望 ok=true，实际=%v", resp["ok"])
	}
	if resp["task_id"].(float64) != 0 {
		t.Errorf("期望 task_id=0（rule 不存在未提交），实际=%v", resp["task_id"])
	}
	if resp["version"] != "" {
		t.Errorf("期望 version 为空，实际=%v", resp["version"])
	}
}

// ────────────────────────────────────────────────────────────────────────────
// HandleUninstallRule 测试
// ────────────────────────────────────────────────────────────────────────────

func TestHandleUninstallRule_MissingSlug(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/uninstall",
		strings.NewReader(`{"instance_ids":[1]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUninstallRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

func TestHandleUninstallRule_EmptyInstances(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/uninstall",
		strings.NewReader(`{"slug":"test","instance_ids":[]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUninstallRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

func TestHandleUninstallRule_NotFound(t *testing.T) {
	setupRulesTestDB(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/uninstall",
		strings.NewReader(`{"slug":"nonexistent","instance_ids":[1]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUninstallRule(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

func TestHandleUninstallRule_Success(t *testing.T) {
	setupRulesTestDB(t)
	createTestRule(t, "uninst-rule", "1.0.0", "rule")

	user := &model.User{Username: "u-uninst", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "uninst-box", InstanceId: "local-uninst-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(inst)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/uninstall",
		strings.NewReader(fmt.Sprintf(`{"slug":"uninst-rule","instance_ids":[%d]}`, inst.ID)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUninstallRule(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		OK      bool   `json:"ok"`
		TaskID  uint   `json:"task_id"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if !resp.OK {
		t.Fatal("期望 ok=true")
	}
	if resp.TaskID == 0 {
		t.Fatal("期望 task_id > 0")
	}

	var task model.RuleDistributionTask
	if model.DB(context.Background()).First(&task, resp.TaskID).Error != nil {
		t.Fatal("task 未创建")
	}
	if task.Type != model.RuleTaskTypeUninstall {
		t.Errorf("type 应=uninstall，实际=%s", task.Type)
	}
}

func TestHandleUninstallRule_NoValidInstances(t *testing.T) {
	setupRulesTestDB(t)
	createTestRule(t, "noinst-rule", "1.0.0", "rule")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/uninstall",
		strings.NewReader(`{"slug":"noinst-rule","instance_ids":[999]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUninstallRule(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d body=%s", w.Code, w.Body.String())
	}
}
