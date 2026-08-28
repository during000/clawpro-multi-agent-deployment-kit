// audit_integration_test.go
//
// LogAudit 集成测试：验证通过 WithAudit / WithCloudProxyAudit 中间件触发后，
// 审计记录正确写入数据库，覆盖以下场景：
//   - WithAudit POST 已注册路径 → 审计记录落库（action/resource/status 正确）
//   - WithAudit POST 已注册路径 + X-Audit-Failed → status="failed"
//   - WithAudit POST 已注册路径 + 带 id 参数 → resource_id 正确
//   - WithAudit POST 已注册路径 + 带登录用户 → user_id/username 正确
//   - WithAudit GET 方法 → 不产生审计记录
//   - WithAudit POST 未注册路径 → 不产生审计记录
//   - WithCloudProxyAudit POST → 审计记录落库（action 含 cloud_proxy_ 前缀）
//   - WithCloudProxyAudit POST + X-Audit-Failed → status="failed"
//   - WithCloudProxyAudit 非 POST → 不产生审计记录
//   - 并发多次 LogAudit 写入 → 所有记录均正确落库

package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initAuditIntegrationTestDB 初始化一个干净的内存 SQLite 数据库供审计集成测试使用。
// 迁移 User + SiteConfig + AuditLog 表，并设置 session store。
func initAuditIntegrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 DB 失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取 sql.DB 失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.User{}, &model.SiteConfig{}, &model.AuditLog{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	restoreSafe := useDBForTestWithSafeRestore(db)

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-audit-integ-secret-32bytes!"))

	t.Cleanup(func() {
		// 等待异步 goroutine 完成写入
		time.Sleep(100 * time.Millisecond)
		Store = origStore
		restoreSafe()
	})
	return db
}

// waitAuditAsync 等待异步 goroutine 中的 LogAudit 完成写入。
func waitAuditAsync() {
	time.Sleep(100 * time.Millisecond)
}

// ============================================================================
// WithAudit 中间件集成测试
// ============================================================================

// TestWithAudit_Integration_PostRegisteredPath_AuditRecordCreated：
// POST 已注册路径 → 审计记录正确落库，验证 action/resource/status 字段。
func TestWithAudit_Integration_PostRegisteredPath_AuditRecordCreated(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	handler := WithAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// /admin/user-groups/create 在 auditRules 中注册为 {"user_group_create", "user_group"}
	r := httptest.NewRequest(http.MethodPost, "/admin/user-groups/create", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	waitAuditAsync()

	// 验证审计记录已落库
	var log model.AuditLog
	if err := db.Where("action = ?", "user_group_create").First(&log).Error; err != nil {
		t.Fatalf("审计记录未写入数据库: %v", err)
	}
	if log.Action != "user_group_create" {
		t.Errorf("Action 不匹配：期望 'user_group_create'，实际 %q", log.Action)
	}
	if log.Resource != "user_group" {
		t.Errorf("Resource 不匹配：期望 'user_group'，实际 %q", log.Resource)
	}
	if log.Status != "success" {
		t.Errorf("Status 不匹配：期望 'success'，实际 %q", log.Status)
	}
}

// TestWithAudit_Integration_XAuditFailed_StatusFailed：
// POST 已注册路径 + handler 设置 X-Audit-Failed → status="failed" 落库。
func TestWithAudit_Integration_XAuditFailed_StatusFailed(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	handler := WithAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Audit-Failed", "1")
		w.WriteHeader(http.StatusBadRequest)
	}))

	r := httptest.NewRequest(http.MethodPost, "/admin/models/create", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	waitAuditAsync()

	var log model.AuditLog
	if err := db.Where("action = ?", "model_create").First(&log).Error; err != nil {
		t.Fatalf("审计记录未写入数据库: %v", err)
	}
	if log.Status != "failed" {
		t.Errorf("Status 不匹配：期望 'failed'（X-Audit-Failed 已设置），实际 %q", log.Status)
	}
	if log.Resource != "ai_model" {
		t.Errorf("Resource 不匹配：期望 'ai_model'，实际 %q", log.Resource)
	}
}

// TestWithAudit_Integration_WithResourceID：
// POST 已注册路径 + URL 带 id 参数 → resource_id 正确记录。
func TestWithAudit_Integration_WithResourceID(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	handler := WithAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/admin/models/delete?id=model-abc-123", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	waitAuditAsync()

	var log model.AuditLog
	if err := db.Where("action = ?", "model_delete").First(&log).Error; err != nil {
		t.Fatalf("审计记录未写入数据库: %v", err)
	}
	if log.ResourceID != "model-abc-123" {
		t.Errorf("ResourceID 不匹配：期望 'model-abc-123'，实际 %q", log.ResourceID)
	}
}

// TestWithAudit_Integration_WithAuthenticatedUser：
// POST 已注册路径 + 带登录用户 session → user_id/username 正确记录。
func TestWithAudit_Integration_WithAuthenticatedUser(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	// 种入用户
	user := &model.User{Username: "audit-test-user"}
	db.Create(user)

	handler := WithAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 先构造一个请求，设置 session 中的 username（RequestUser 通过 session["username"] 查 DB）
	r := httptest.NewRequest(http.MethodPost, "/admin/skills/create?id=skill-001", nil)
	w := httptest.NewRecorder()

	// 创建 session 并写入 username（session 名称为 "hatchery-session"）
	session, _ := Store.Get(r, "hatchery-session")
	session.Values["username"] = "audit-test-user"
	session.Save(r, w)

	// 重新构造请求，带上 cookie
	cookies := w.Result().Cookies()
	r2 := httptest.NewRequest(http.MethodPost, "/admin/skills/create?id=skill-001", nil)
	for _, c := range cookies {
		r2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, r2)

	waitAuditAsync()

	var log model.AuditLog
	if err := db.Where("action = ?", "skill_create").First(&log).Error; err != nil {
		t.Fatalf("审计记录未写入数据库: %v", err)
	}
	if log.UserID != user.ID {
		t.Errorf("UserID 不匹配：期望 %d，实际 %d", user.ID, log.UserID)
	}
	if log.Username != "audit-test-user" {
		t.Errorf("Username 不匹配：期望 'audit-test-user'，实际 %q", log.Username)
	}
	if log.ResourceID != "skill-001" {
		t.Errorf("ResourceID 不匹配：期望 'skill-001'，实际 %q", log.ResourceID)
	}
}

// TestWithAudit_Integration_GetMethod_NoAuditRecord：
// GET 方法 → 不产生审计记录。
func TestWithAudit_Integration_GetMethod_NoAuditRecord(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	handler := WithAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/admin/user-groups/create", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	waitAuditAsync()

	var count int64
	db.Model(&model.AuditLog{}).Count(&count)
	if count != 0 {
		t.Errorf("GET 方法不应产生审计记录，实际有 %d 条", count)
	}
}

// TestWithAudit_Integration_UnregisteredPath_NoAuditRecord：
// POST 未注册路径 → 不产生审计记录。
func TestWithAudit_Integration_UnregisteredPath_NoAuditRecord(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	handler := WithAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/admin/nonexistent-path-xyz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	waitAuditAsync()

	var count int64
	db.Model(&model.AuditLog{}).Count(&count)
	if count != 0 {
		t.Errorf("未注册路径不应产生审计记录，实际有 %d 条", count)
	}
}

// TestWithAudit_Integration_PutMethod_AuditRecordCreated：
// PUT 方法也应触发审计（WithAudit 支持 POST/PUT/DELETE）。
func TestWithAudit_Integration_PutMethod_AuditRecordCreated(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	handler := WithAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPut, "/admin/models/update?id=model-put-001", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	waitAuditAsync()

	var log model.AuditLog
	if err := db.Where("action = ?", "model_update").First(&log).Error; err != nil {
		t.Fatalf("PUT 方法应触发审计记录: %v", err)
	}
	if log.ResourceID != "model-put-001" {
		t.Errorf("ResourceID 不匹配：期望 'model-put-001'，实际 %q", log.ResourceID)
	}
}

// TestWithAudit_Integration_DeleteMethod_AuditRecordCreated：
// DELETE 方法也应触发审计。
func TestWithAudit_Integration_DeleteMethod_AuditRecordCreated(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	handler := WithAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodDelete, "/admin/channels/delete?id=ch-del-001", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	waitAuditAsync()

	var log model.AuditLog
	if err := db.Where("action = ?", "channel_delete").First(&log).Error; err != nil {
		t.Fatalf("DELETE 方法应触发审计记录: %v", err)
	}
	if log.ResourceID != "ch-del-001" {
		t.Errorf("ResourceID 不匹配：期望 'ch-del-001'，实际 %q", log.ResourceID)
	}
	if log.Resource != "ai_channel" {
		t.Errorf("Resource 不匹配：期望 'ai_channel'，实际 %q", log.Resource)
	}
}

// ============================================================================
// WithCloudProxyAudit 中间件集成测试
// ============================================================================

// TestWithCloudProxyAudit_Integration_PostSuccess_AuditRecordCreated：
// POST /admin/cloud/mutate/cvm + X-TC-Action → 审计记录正确落库。
func TestWithCloudProxyAudit_Integration_PostSuccess_AuditRecordCreated(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	// 确保 FixedSnapshot 非 nil
	origSnap := common.FixedSnapshot
	if origSnap == nil {
		common.FixedSnapshot = &common.TenantSnapshot{}
	}
	t.Cleanup(func() { common.FixedSnapshot = origSnap })

	handler := WithCloudProxyAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/admin/cloud/mutate/cvm", nil)
	r.Header.Set("X-TC-Action", "RunInstances")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	waitAuditAsync()

	var log model.AuditLog
	if err := db.Where("action = ?", "cloud_proxy_RunInstances").First(&log).Error; err != nil {
		t.Fatalf("审计记录未写入数据库: %v", err)
	}
	if log.Action != "cloud_proxy_RunInstances" {
		t.Errorf("Action 不匹配：期望 'cloud_proxy_RunInstances'，实际 %q", log.Action)
	}
	if log.Resource != "cloud_cvm" {
		t.Errorf("Resource 不匹配：期望 'cloud_cvm'，实际 %q", log.Resource)
	}
	if log.ResourceID != "RunInstances" {
		t.Errorf("ResourceID 不匹配：期望 'RunInstances'，实际 %q", log.ResourceID)
	}
	if log.Status != "success" {
		t.Errorf("Status 不匹配：期望 'success'，实际 %q", log.Status)
	}
}

// TestWithCloudProxyAudit_Integration_XAuditFailed_StatusFailed：
// POST + X-Audit-Failed → status="failed" 落库。
func TestWithCloudProxyAudit_Integration_XAuditFailed_StatusFailed(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	origSnap := common.FixedSnapshot
	if origSnap == nil {
		common.FixedSnapshot = &common.TenantSnapshot{}
	}
	t.Cleanup(func() { common.FixedSnapshot = origSnap })

	handler := WithCloudProxyAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Audit-Failed", "1")
		w.WriteHeader(http.StatusInternalServerError)
	}))

	r := httptest.NewRequest(http.MethodPost, "/admin/cloud/mutate/vpc", nil)
	r.Header.Set("X-TC-Action", "CreateVpc")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	waitAuditAsync()

	var log model.AuditLog
	if err := db.Where("action = ?", "cloud_proxy_CreateVpc").First(&log).Error; err != nil {
		t.Fatalf("审计记录未写入数据库: %v", err)
	}
	if log.Status != "failed" {
		t.Errorf("Status 不匹配：期望 'failed'，实际 %q", log.Status)
	}
	if log.Resource != "cloud_vpc" {
		t.Errorf("Resource 不匹配：期望 'cloud_vpc'，实际 %q", log.Resource)
	}
}

// TestWithCloudProxyAudit_Integration_GetMethod_NoAuditRecord：
// GET 方法 → 不产生审计记录。
func TestWithCloudProxyAudit_Integration_GetMethod_NoAuditRecord(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	origSnap := common.FixedSnapshot
	if origSnap == nil {
		common.FixedSnapshot = &common.TenantSnapshot{}
	}
	t.Cleanup(func() { common.FixedSnapshot = origSnap })

	handler := WithCloudProxyAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/admin/cloud/mutate/cvm", nil)
	r.Header.Set("X-TC-Action", "DescribeInstances")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	waitAuditAsync()

	var count int64
	db.Model(&model.AuditLog{}).Count(&count)
	if count != 0 {
		t.Errorf("GET 方法不应产生审计记录，实际有 %d 条", count)
	}
}

// TestWithCloudProxyAudit_Integration_ActionFromQuery：
// X-TC-Action 为空时从 URL Query 的 Action 参数获取。
func TestWithCloudProxyAudit_Integration_ActionFromQuery(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	origSnap := common.FixedSnapshot
	if origSnap == nil {
		common.FixedSnapshot = &common.TenantSnapshot{}
	}
	t.Cleanup(func() { common.FixedSnapshot = origSnap })

	handler := WithCloudProxyAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 不设置 X-TC-Action header，而是通过 URL Query 传递
	r := httptest.NewRequest(http.MethodPost, "/admin/cloud/mutate/cbs?Action=CreateDisks", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	waitAuditAsync()

	var log model.AuditLog
	if err := db.Where("action = ?", "cloud_proxy_CreateDisks").First(&log).Error; err != nil {
		t.Fatalf("审计记录未写入数据库: %v", err)
	}
	if log.Resource != "cloud_cbs" {
		t.Errorf("Resource 不匹配：期望 'cloud_cbs'，实际 %q", log.Resource)
	}
}

// ============================================================================
// LogAudit 并发写入集成测试
// ============================================================================

// TestLogAudit_Integration_ConcurrentWrites：
// 模拟多个 goroutine 并发调用 LogAudit，验证所有记录均正确落库不丢失。
func TestLogAudit_Integration_ConcurrentWrites(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	ctx := context.Background()
	const concurrency = 20

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			model.LogAudit(ctx, time.Now(), uint(idx), "user-concurrent",
				"agent_bridge_concurrent_test", "instance", "ins-concurrent", "success")
		}(i)
	}

	wg.Wait()
	// 额外等待确保所有写入完成
	time.Sleep(50 * time.Millisecond)

	var count int64
	db.Model(&model.AuditLog{}).Where("action = ?", "agent_bridge_concurrent_test").Count(&count)
	if count != concurrency {
		t.Errorf("并发写入应产生 %d 条记录，实际 %d 条", concurrency, count)
	}
}

// TestLogAudit_Integration_StartedAtPreserved：
// 验证通过 WithAudit 中间件触发时，started_at 记录的是 handler 开始执行的时间。
func TestLogAudit_Integration_StartedAtPreserved(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	beforeTest := time.Now()

	handler := WithAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟 handler 执行耗时
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/admin/images/delete?id=img-time-001", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	waitAuditAsync()

	var log model.AuditLog
	if err := db.Where("action = ?", "image_delete").First(&log).Error; err != nil {
		t.Fatalf("审计记录未写入数据库: %v", err)
	}

	// started_at 应在 beforeTest 之后（handler 开始时记录）
	if log.StartedAt.Before(beforeTest) {
		t.Errorf("StartedAt 应在测试开始之后：beforeTest=%v, startedAt=%v", beforeTest, log.StartedAt)
	}
	// started_at 应在当前时间之前
	if log.StartedAt.After(time.Now()) {
		t.Errorf("StartedAt 不应在未来：startedAt=%v", log.StartedAt)
	}
}

// TestLogAudit_Integration_MultiplePathsMultipleRecords：
// 连续触发多个不同路径的审计，验证每个路径都独立产生正确的记录。
func TestLogAudit_Integration_MultiplePathsMultipleRecords(t *testing.T) {
	db := initAuditIntegrationTestDB(t)

	handler := WithAudit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	paths := []struct {
		path           string
		expectedAction string
	}{
		{"/admin/user-groups/create", "user_group_create"},
		{"/admin/models/create", "model_create"},
		{"/admin/channels/add", "channel_add"},
		{"/admin/skills/create", "skill_create"},
		{"/admin/roles/create", "role_create"},
	}

	for _, p := range paths {
		r := httptest.NewRequest(http.MethodPost, p.path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}

	waitAuditAsync()

	// 验证每个路径都产生了对应的审计记录
	for _, p := range paths {
		var log model.AuditLog
		if err := db.Where("action = ?", p.expectedAction).First(&log).Error; err != nil {
			t.Errorf("路径 %s 应产生 action=%q 的审计记录，但未找到: %v", p.path, p.expectedAction, err)
		}
	}

	// 验证总记录数
	var count int64
	db.Model(&model.AuditLog{}).Count(&count)
	if count != int64(len(paths)) {
		t.Errorf("应有 %d 条审计记录，实际 %d 条", len(paths), count)
	}
}
