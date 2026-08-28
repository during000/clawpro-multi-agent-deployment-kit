package controller

import (
	"context"
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

// initBrowserVNCHandlerTestDB 为 Browser VNC Handler 测试初始化 DB。
func initBrowserVNCHandlerTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{}, &model.Instance{}, &model.SiteConfig{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	return func() {
		origDB()
		Store = origStore
	}
}

// browserVNCReqWithSession 构造带 session 的请求。
func browserVNCReqWithSession(t *testing.T, method, path, username string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
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

// ─── HandleBrowserVNCAccess ──────────────────────────────────────────────

func TestHandleBrowserVNCAccess_Unauthorized(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleBrowserVNCAccess(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleBrowserVNCAccess_InstanceNotFound(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := browserVNCReqWithSession(t, http.MethodGet, "/openclaw/browser-vnc-access?id=999", "u1")
	rr := httptest.NewRecorder()
	HandleBrowserVNCAccess(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleBrowserVNCAccess_HermesForbidden(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "hermes-inst", InstanceId: "ins-hermes-vnc",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	req := browserVNCReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/browser-vnc-access?id=%d", inst.ID), "u1")
	rr := httptest.NewRecorder()
	HandleBrowserVNCAccess(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("hermes 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "云端浏览器") {
		t.Errorf("错误消息应包含 '云端浏览器'，实际=%s", rr.Body.String())
	}
}

func TestHandleBrowserVNCAccess_AceForbidden(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "ace-inst", InstanceId: "ins-ace-vnc",
		UserID: user.ID, AgentType: model.AgentTypeLightclawACE,
	}
	model.DB(context.Background()).Create(inst)

	req := browserVNCReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/browser-vnc-access?id=%d", inst.ID), "u1")
	rr := httptest.NewRecorder()
	HandleBrowserVNCAccess(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("ace 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── HandleBrowserStatus（关键：轮询接口不能 403）────────────────────────

func TestHandleBrowserStatus_HermesReturns200WithUnsupported(t *testing.T) {
	// 关键行为：hermes/ace 不能返回 403，要返回 200 + unsupported=true
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "hermes", InstanceId: "ins-hermes-st",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	req := browserVNCReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/browser-status?id=%d", inst.ID), "u1")
	rr := httptest.NewRecorder()
	HandleBrowserStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("hermes 应返回 200（不能 403），实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v body=%s", err, rr.Body.String())
	}
	// 响应格式：{"ok":true,"data":{...}}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data 字段应为对象，实际=%v", resp)
	}
	if !data["unsupported"].(bool) {
		t.Errorf("unsupported 应为 true，实际=%v", data["unsupported"])
	}
	if data["ai_active"].(bool) {
		t.Error("ai_active 应为 false")
	}
	if data["takeover"].(bool) {
		t.Error("takeover 应为 false")
	}
	if at, _ := data["agent_type"].(string); at != model.AgentTypeHermes {
		t.Errorf("agent_type 应为 hermes，实际=%q", at)
	}
}

func TestHandleBrowserStatus_AceReturns200WithUnsupported(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "ace", InstanceId: "ins-ace-st",
		UserID: user.ID, AgentType: model.AgentTypeLightclawACE,
	}
	model.DB(context.Background()).Create(inst)

	req := browserVNCReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/browser-status?id=%d", inst.ID), "u1")
	rr := httptest.NewRecorder()
	HandleBrowserStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ace 应返回 200，实际=%d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "unsupported") {
		t.Errorf("响应应包含 unsupported 字段，实际=%s", rr.Body.String())
	}
}

func TestHandleBrowserStatus_InstanceNotFound(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := browserVNCReqWithSession(t, http.MethodGet, "/openclaw/browser-status?id=999", "u1")
	rr := httptest.NewRecorder()
	HandleBrowserStatus(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应返回 400，实际=%d", rr.Code)
	}
}

// ─── HandleBrowserTakeover ───────────────────────────────────────────────

func TestHandleBrowserTakeover_HermesForbidden(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "h", InstanceId: "ins-h-tk",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	req := browserVNCReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/browser-takeover?id=%d&action=start", inst.ID), "u1")
	rr := httptest.NewRecorder()
	handleBrowserTakeover(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("hermes 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleBrowserTakeover_AceForbidden(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "a", InstanceId: "ins-a-tk",
		UserID: user.ID, AgentType: model.AgentTypeLightclawACE,
	}
	model.DB(context.Background()).Create(inst)

	req := browserVNCReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/browser-takeover?id=%d&action=stop", inst.ID), "u1")
	rr := httptest.NewRecorder()
	handleBrowserTakeover(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("ace 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleBrowserTakeover_InstanceNotFound(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := browserVNCReqWithSession(t, http.MethodPost, "/openclaw/browser-takeover?id=999", "u1")
	rr := httptest.NewRecorder()
	handleBrowserTakeover(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应返回 400，实际=%d", rr.Code)
	}
}

// ─── HandleBrowserVNCCheck ───────────────────────────────────────────────

func TestHandleBrowserVNCCheck_HermesForbidden(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "h", InstanceId: "ins-h-chk",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	req := browserVNCReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/browser-vnc-check?id=%d", inst.ID), "u1")
	rr := httptest.NewRecorder()
	HandleBrowserVNCCheck(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("hermes 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleBrowserVNCCheck_AceForbidden(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "a", InstanceId: "ins-a-chk",
		UserID: user.ID, AgentType: model.AgentTypeLightclawACE,
	}
	model.DB(context.Background()).Create(inst)

	req := browserVNCReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/browser-vnc-check?id=%d", inst.ID), "u1")
	rr := httptest.NewRecorder()
	HandleBrowserVNCCheck(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("ace 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleBrowserVNCCheck_InstanceNotFound(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := browserVNCReqWithSession(t, http.MethodGet, "/openclaw/browser-vnc-check?id=999", "u1")
	rr := httptest.NewRecorder()
	HandleBrowserVNCCheck(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应返回 400，实际=%d", rr.Code)
	}
}

// ─── HandleBrowserVNCInstall ─────────────────────────────────────────────

func TestHandleBrowserVNCInstall_HermesForbidden(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "h", InstanceId: "ins-h-inst",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	req := browserVNCReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/browser-vnc-install?id=%d", inst.ID), "u1")
	rr := httptest.NewRecorder()
	handleBrowserVNCInstall(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("hermes 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleBrowserVNCInstall_AceForbidden(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "a", InstanceId: "ins-a-inst",
		UserID: user.ID, AgentType: model.AgentTypeLightclawACE,
	}
	model.DB(context.Background()).Create(inst)

	req := browserVNCReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/browser-vnc-install?id=%d", inst.ID), "u1")
	rr := httptest.NewRecorder()
	handleBrowserVNCInstall(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("ace 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleBrowserVNCInstall_Unauthorized(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-vnc-install", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handleBrowserVNCInstall(rr, req, testCVMFetcher)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleBrowserVNCInstall_InstanceNotFound(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := browserVNCReqWithSession(t, http.MethodPost, "/openclaw/browser-vnc-install?id=999", "u1")
	rr := httptest.NewRecorder()
	handleBrowserVNCInstall(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应返回 400，实际=%d", rr.Code)
	}
}
