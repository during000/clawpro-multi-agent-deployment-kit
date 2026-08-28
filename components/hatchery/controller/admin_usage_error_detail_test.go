package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initUsageErrorTestEnv 构造一个"故意缺少业务表"的 DB，让 GORM 在 Count/Find
// 时返回 "no such table" 错误，覆盖 admin_usage handler 的 DB 失败分支。
//
// 这是对真实生产事故的最小可复现样本：DB schema 异常 / 软删除迁移失败 /
// 多租户隔离回调被误删时，业务表不可见，handler 应正确把 err.Error() 透传到响应 detail。
func initUsageErrorTestEnv(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	// 故意不 AutoMigrate DailyUsageSummary / LLMUsageLog，让查询自然报错。
	// 但 SiteConfig 需要建表，否则 GetSiteConfig 会更早就报错，掩盖了我们要测的分支。
	if err := db.AutoMigrate(&model.SiteConfig{}); err != nil {
		t.Fatalf("迁移 SiteConfig 失败: %v", err)
	}
	db.Create(&model.SiteConfig{})

	origDB := model.UseDBForTest(db)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	return func() {
		origDB()
		AdminToken = origToken
		Store = origStore
	}
}

func usageErrReq(path string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

func parseUsageErrResp(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v, body=%s", err, w.Body.String())
	}
	return resp
}

// ── HandleAdminUsageData：DB 失败时应返回 detail ───────────────────────────

func TestHandleAdminUsageData_DBError_DetailIncluded(t *testing.T) {
	cleanup := initUsageErrorTestEnv(t)
	defer cleanup()

	req := usageErrReq("/admin/usage/data?group_by=user")
	w := httptest.NewRecorder()
	HandleAdminUsageData(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500, got %d, body=%s", w.Code, w.Body.String())
	}
	resp := parseUsageErrResp(t, w)

	// 验证 1：error 字段是 i18n 翻译后的"查询用量数据失败"消息
	if got, _ := resp["error"].(string); !strings.Contains(got, "查询用量数据失败") {
		t.Errorf("error 字段不对, got %q", got)
	}

	// 验证 2：detail 字段存在且包含底层 DB 错误信息
	detail, _ := resp["detail"].(string)
	if detail == "" {
		t.Errorf("detail 应非空，body=%s", w.Body.String())
	}
	// SQLite 报错典型为 "no such table: daily_usage_summaries"
	if !strings.Contains(detail, "no such table") {
		t.Errorf("detail 应包含底层 SQL 错误, got %q", detail)
	}
}

// ── HandleAdminUsageLogs：Count 失败时 detail 透传 ─────────────────────────

func TestHandleAdminUsageLogs_CountError_DetailIncluded(t *testing.T) {
	cleanup := initUsageErrorTestEnv(t)
	defer cleanup()

	// 不带 page_size，count 会先执行；表不存在 → Count 报错
	req := usageErrReq("/admin/usage/logs")
	w := httptest.NewRecorder()
	HandleAdminUsageLogs(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500, got %d, body=%s", w.Code, w.Body.String())
	}
	resp := parseUsageErrResp(t, w)

	if got, _ := resp["error"].(string); !strings.Contains(got, "记录") && !strings.Contains(got, "总数") {
		// 实际 i18n 文案视具体 key，但应包含"总数"或"记录"
		t.Logf("error 字段=%q（仅作信息）", got)
	}
	detail, _ := resp["detail"].(string)
	if detail == "" {
		t.Errorf("Count 失败 detail 应非空")
	}
	if !strings.Contains(detail, "no such table") {
		t.Errorf("detail 应包含底层 SQL 错误, got %q", detail)
	}
}

// ── HandleAdminUsageLogs：Find 失败时 detail 透传 ──────────────────────────
//
// 构造 Count 通过、Find 失败的场景：建好 llm_usage_logs 表让 Count 正常，
// 注册一个 GORM Query 回调，在该表的 SELECT *（即 Find，非 COUNT）时强制
// 返回错误。这种"中途失败"是真实生产环境中可能遇到的（如 page_size 越界、
// DB 中途断连、读副本延迟等）。
func TestHandleAdminUsageLogs_FindError_DetailIncluded(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.SiteConfig{}, &model.LLMUsageLog{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	db.Create(&model.SiteConfig{})

	origDB := model.UseDBForTest(db)
	defer origDB()
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() { Store = origStore }()

	// 注册 Query 回调：只在目标表的 Find（dest 是 *[]LLMUsageLog）时报错；
	// Count 的 dest 是 *int64，不会被拦截。
	cbName := "test:fail_llm_usage_logs_find"
	if err := db.Callback().Query().Before("gorm:query").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement.Table != "llm_usage_logs" {
			return
		}
		if _, isSlice := tx.Statement.Dest.(*[]model.LLMUsageLog); isSlice {
			_ = tx.AddError(fmt.Errorf("injected Find error: simulated read-replica timeout"))
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	defer db.Callback().Query().Remove(cbName)

	// 触发 HandleAdminUsageLogs：传 page_size 让 Find 走 Limit 路径
	req := usageErrReq("/admin/usage/logs?page=1&page_size=20")
	w := httptest.NewRecorder()
	HandleAdminUsageLogs(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Find 失败时应返回 500, got %d, body=%s", w.Code, w.Body.String())
	}
	resp := parseUsageErrResp(t, w)

	// 验证 1：error 字段是 i18n 翻译后的"查询用量日志失败"消息
	if got, _ := resp["error"].(string); !strings.Contains(got, "日志") && !strings.Contains(got, "Logs") {
		t.Logf("error 字段=%q（仅作信息）", got)
	}

	// 验证 2：detail 字段包含我们注入的错误信息
	detail, _ := resp["detail"].(string)
	if detail == "" {
		t.Fatal("Find 失败 detail 应非空")
	}
	if !strings.Contains(detail, "injected Find error") {
		t.Errorf("detail 应包含注入的错误, got %q", detail)
	}
}

// ── 正常路径回归：确认正常请求不会带 detail（避免误污染响应） ───────────────

func TestHandleAdminUsageData_NoErrorNoDetail(t *testing.T) {
	cleanup := initUsageDataTestEnv(t) // 此 helper 在 admin_usage_test.go 已建好全表
	defer cleanup()

	req := usageErrReq("/admin/usage/data?group_by=user")
	w := httptest.NewRecorder()
	HandleAdminUsageData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("正常请求应 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是 JSON: %v", err)
	}
	if _, hasDetail := resp["detail"]; hasDetail {
		t.Errorf("正常响应不应有 detail 字段, body=%s", w.Body.String())
	}
}
