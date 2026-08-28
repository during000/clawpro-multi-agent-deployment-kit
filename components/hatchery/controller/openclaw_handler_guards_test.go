package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initOpenClawHandlerTestDB 初始化 openclaw handler 测试 DB。
func initOpenClawHandlerTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{}, &model.Instance{}, &model.SiteConfig{},
		&model.InstanceAdjustment{},
		&model.AuditLog{}, &model.Notification{}, &model.InstanceModel{},
		&model.AIModel{}, &model.AIChannel{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	return func() {
		// 等待异步 goroutine 完成（如 installSkillsAsync、UpdateAPITokenLastUsed 等）
		time.Sleep(100 * time.Millisecond)
		origDB()
		Store = origStore
	}
}

// openclawReqWithSession 构造带 session 的请求。
func openclawReqWithSession(t *testing.T, method, path, username, body string) *http.Request {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username

	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

// ─── HandleApprove: agent_type guard ────────────────────────────────────

func TestHandleApprove_MethodNotAllowed(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/approve", nil)
	rr := httptest.NewRecorder()
	HandleApprove(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleApprove_Unauthorized(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/approve", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleApprove(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleApprove_HermesForbidden(t *testing.T) {
	// 覆盖源码 1842-1848: checkInstanceSupportsApprove 对 Hermes 返回 403
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u-hermes", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "h", InstanceId: "ins-h-apv",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("code", "test-code")
	req := openclawReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/approve?id=%d", inst.ID), "u-hermes", form.Encode())
	rr := httptest.NewRecorder()
	HandleApprove(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Hermes 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleApprove_AceForbidden(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u-ace", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "a", InstanceId: "ins-a-apv",
		UserID: user.ID, AgentType: model.AgentTypeLightclawACE,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("code", "test-code")
	req := openclawReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/approve?id=%d", inst.ID), "u-ace", form.Encode())
	rr := httptest.NewRecorder()
	HandleApprove(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("ACE 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleApprove_InstanceNotFound(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := openclawReqWithSession(t, http.MethodPost, "/openclaw/approve?id=999", "u1", "")
	rr := httptest.NewRecorder()
	HandleApprove(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在应返回 400，实际=%d", rr.Code)
	}
}

// ─── HandleResetInstance: agent_type guard ─────────────────────────────

func TestHandleResetInstance_MethodNotAllowed(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/reset", nil)
	rr := httptest.NewRecorder()
	handleResetInstance(rr, req, testCVMFetcher)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleResetInstance_HermesForbidden(t *testing.T) {
	// Hermes 支持重装，无启用镜像时应返回 500
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "h", InstanceId: "ins-h-reset-user",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	req := openclawReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/reset?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Hermes 无镜像应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleResetInstance_AceForbidden(t *testing.T) {
	// ACE 支持重装，无启用镜像时应返回 500
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "a", InstanceId: "ins-a-reset-user",
		UserID: user.ID, AgentType: model.AgentTypeLightclawACE,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	req := openclawReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/reset?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ACE 无镜像应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── HandleServiceStatus: agent_type 分派 ──────────────────────────────

func TestHandleServiceStatus_Unauthorized(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/service-status?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleServiceStatus(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleServiceStatus_UnknownAgentType(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-svc-unk",
		UserID: user.ID, AgentType: "future_unknown_type",
	}
	model.DB(context.Background()).Create(inst)

	req := openclawReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/service-status?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleServiceStatus(rr, req)

	// ResolveScript 失败 → 400
	if rr.Code != http.StatusBadRequest {
		t.Errorf("未知 agent_type 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "check_service") {
		t.Errorf("错误应含 check_service，实际=%s", rr.Body.String())
	}
}
