package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// setupCLSTestEnv 初始化 CLS handler 测试所需的内存数据库和全局状态。
func setupCLSTestEnv(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("数据库初始化失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.SiteConfig{},
		&model.GroupConfigBinding{},
		&model.GroupClosure{},
		&model.UserGroupMember{},
		&model.UserGroup{},
		&model.Instance{},
		&model.User{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	restore := model.UseDBForTest(db)

	origToken := AdminToken
	AdminToken = "test-admin-token"

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	t.Cleanup(func() {
		restore()
		AdminToken = origToken
		Store = origStore
	})
}

// clsAdminReq 创建带 admin Bearer Token 的 JSON 请求。
func clsAdminReq(method, path, body string) (*http.Request, *httptest.ResponseRecorder) {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req, httptest.NewRecorder()
}

// TestHandleAdminCloseClsService_DeleteResourcesFalse 覆盖 shouldDeleteResources 返回 false 的路径（517行）：
// 发送 delete_resources=false 的请求，触发 shouldDeleteResources 返回 false，
// 然后继续执行到 newCLSClient（失败，返回 500）。
func TestHandleAdminCloseClsService_DeleteResourcesFalse(t *testing.T) {
	setupCLSTestEnv(t)

	req, w := clsAdminReq("POST", "/admin/cls/close", `{"delete_resources":false}`)
	HandleAdminCloseClsService(w, req)

	// 期望 500（newCLSClient 失败）
	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500（newCLSClient 失败），实际 %d，body: %s", w.Code, w.Body.String())
	}
}

// 发送无请求体的请求，触发 newCLSClient 失败（返回 500）。
func TestHandleAdminCloseClsService_NoBody(t *testing.T) {
	setupCLSTestEnv(t)

	req, w := clsAdminReq("POST", "/admin/cls/close", "")
	HandleAdminCloseClsService(w, req)

	// 期望 500（newCLSClient 失败，因为没有 CLS 配置）
	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500（newCLSClient 失败），实际 %d，body: %s", w.Code, w.Body.String())
	}
}

// TestHandleAdminCloseClsService_InvalidJSON 覆盖 HandleAdminCloseClsService 的 JSON 解析失败路径（539-545行）：
// 发送无效 JSON 请求体，触发 slog.Warn，然后继续执行到 newCLSClient（失败，返回 500）。
func TestHandleAdminCloseClsService_InvalidJSON(t *testing.T) {
	setupCLSTestEnv(t)

	body := `{invalid json}`
	r := httptest.NewRequest("POST", "/admin/cls/close", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer test-admin-token")
	r.ContentLength = int64(len(body))
	w := httptest.NewRecorder()
	HandleAdminCloseClsService(w, r)

	// 期望 500（newCLSClient 失败）
	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500（newCLSClient 失败），实际 %d，body: %s", w.Code, w.Body.String())
	}
}

// 当请求体为无效 JSON 时，slog.Warn 被调用，req.GroupIDs 被清空，
// scope_type 推断为 "all"，继续执行到 newCLSCommonClient（失败，返回 500）。
func TestHandleAdminOpenClsService_InvalidJSON(t *testing.T) {
	setupCLSTestEnv(t)

	// 发送无效 JSON 请求体（触发 json.Decode 失败）
	body := `{invalid json}`
	r := httptest.NewRequest("POST", "/admin/cls/open", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer test-admin-token")
	r.ContentLength = int64(len(body))
	w := httptest.NewRecorder()
	HandleAdminOpenClsService(w, r)

	// JSON 解析失败后，scope_type 推断为 "all"，继续执行到 newCLSCommonClient 失败，返回 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500（newCLSCommonClient 失败），实际 %d，body: %s", w.Code, w.Body.String())
	}
}

// TestHandleAdminOpenClsService_NoBody 覆盖空 body 兼容旧调用行为：
// 空 body 时 scope_type 推断为 "all"，继续执行到 newCLSCommonClient（失败，返回 500）。
func TestHandleAdminOpenClsService_NoBody(t *testing.T) {
	setupCLSTestEnv(t)

	req, w := clsAdminReq("POST", "/admin/cls/open", "")
	HandleAdminOpenClsService(w, req)

	// 空 body 时 scope_type 推断为 "all"，继续执行到 newCLSCommonClient 失败，返回 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500（newCLSCommonClient 失败），实际 %d，body: %s", w.Code, w.Body.String())
	}
}

// TestHandleAdminOpenClsService_OnlyGroupIDs 覆盖只传 group_ids 不传 scope_type 的兼容行为：
// group_ids 非空时 scope_type 自动推断为 "group"，继续执行到 group_ids 校验（分组不存在，返回 400）。
func TestHandleAdminOpenClsService_OnlyGroupIDs(t *testing.T) {
	setupCLSTestEnv(t)

	// 只传 group_ids，不传 scope_type
	req, w := clsAdminReq("POST", "/admin/cls/open", `{"group_ids":[99999]}`)
	HandleAdminOpenClsService(w, req)

	// scope_type 推断为 "group"，group_ids 中分组不存在，返回 400
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400（分组不存在），实际 %d，body: %s", w.Code, w.Body.String())
	}
}

// TestHandleAdminOpenClsService_TooManyGroupIDs 覆盖分组数量超过上限的路径（354-356行）：
// 临时将 maxScopeGroupIDs 设为 1，发送 2 个分组 ID 的请求，触发 400 错误。
func TestHandleAdminOpenClsService_TooManyGroupIDs(t *testing.T) {
	setupCLSTestEnv(t)

	// 临时修改 maxScopeGroupIDs 为 1
	orig := maxScopeGroupIDs
	maxScopeGroupIDs = 1
	t.Cleanup(func() { maxScopeGroupIDs = orig })

	req, w := clsAdminReq("POST", "/admin/cls/open", `{"scope_type":"group","group_ids":[1,2]}`)
	HandleAdminOpenClsService(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400（分组数量超过上限），实际 %d，body: %s", w.Code, w.Body.String())
	}
}

// 当 scope_type 不是 "all" 或 "group" 时，返回 400 错误。
func TestHandleAdminOpenClsService_InvalidScopeType(t *testing.T) {
	setupCLSTestEnv(t)

	req, w := clsAdminReq("POST", "/admin/cls/open", `{"scope_type":"invalid"}`)
	HandleAdminOpenClsService(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d，body: %s", w.Code, w.Body.String())
	}
}

// TestHandleAdminOpenClsService_AllModeClearsGroupIDs 覆盖 all 模式清空 group_ids 路径（348-350行）：
// 当 scope_type 为 "all" 时，group_ids 被清空，继续执行到 newCLSCommonClient（会失败，返回 500）。
func TestHandleAdminOpenClsService_AllModeClearsGroupIDs(t *testing.T) {
	setupCLSTestEnv(t)

	// scope_type=all 时，group_ids 被清空，继续执行到 newCLSCommonClient
	// 由于没有 CLS 配置，newCLSCommonClient 会失败，返回 500
	req, w := clsAdminReq("POST", "/admin/cls/open", `{"scope_type":"all","group_ids":[1,2,3]}`)
	HandleAdminOpenClsService(w, req)

	// 期望 500（newCLSCommonClient 失败）或 400（其他校验失败）
	if w.Code == http.StatusBadRequest {
		t.Errorf("scope_type=all 时不应返回 400（group_ids 应被清空），实际 body: %s", w.Body.String())
	}
}

// TestHandleAdminOpenClsService_InvalidGroupIDs 覆盖 group_ids 存在性校验失败路径（359-363行）：
// 当 group_ids 中包含不存在的分组 ID 时，返回 400 错误。
func TestHandleAdminOpenClsService_InvalidGroupIDs(t *testing.T) {
	setupCLSTestEnv(t)

	// group_ids 中包含不存在的分组 ID（99999）
	req, w := clsAdminReq("POST", "/admin/cls/open", `{"scope_type":"group","group_ids":[99999]}`)
	HandleAdminOpenClsService(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400（分组不存在），实际 %d，body: %s", w.Code, w.Body.String())
	}
}
