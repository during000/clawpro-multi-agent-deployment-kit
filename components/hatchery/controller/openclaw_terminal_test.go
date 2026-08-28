package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"hatchery/model"
)

// ==================== HandleInstanceTerminal Tests ====================

func TestHandleInstanceTerminal_Unauthenticated(t *testing.T) {
	cleanup := initOpenclawTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/terminal-url", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	handleInstanceTerminal(rr, req, testCVMFetcher)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleInstanceTerminal_MethodNotAllowed(t *testing.T) {
	cleanup := initOpenclawTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := userOpenclawReqWithSession(t, http.MethodGet, "/openclaw/terminal-url", "testuser")
	rr := httptest.NewRecorder()

	handleInstanceTerminal(rr, req, testCVMFetcher)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleInstanceTerminal_OpenClawTerminalDisabled(t *testing.T) {
	cleanup := initOpenclawTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建 SiteConfig, TerminalEnabled=false
	config := model.SiteConfig{ID: 1, TerminalEnabled: false}
	model.DB(context.Background()).Create(&config)

	proxyToken := "sk-test-openclaw"
	inst := model.Instance{
		Name:       "test-inst",
		InstanceId: "ins-test-001",
		UserID:     user.ID,
		AgentType:  "openclaw",
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	form := url.Values{"id": {uintToString(inst.ID)}}
	req := userTerminalReqWithSession(t, http.MethodPost, "/openclaw/terminal-url", "testuser", form.Encode())
	rr := httptest.NewRecorder()

	handleInstanceTerminal(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for openclaw with terminal disabled, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if errMsg, ok := resp["error"].(string); ok {
		if !strings.Contains(errMsg, "终端功能未开启") {
			t.Errorf("expected error about terminal disabled, got %q", errMsg)
		}
	}
}

func TestHandleInstanceTerminal_EmptyAgentTypeTerminalDisabled(t *testing.T) {
	cleanup := initOpenclawTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	config := model.SiteConfig{ID: 1, TerminalEnabled: false}
	model.DB(context.Background()).Create(&config)

	proxyToken := "sk-test-empty"
	inst := model.Instance{
		Name:       "legacy-inst",
		InstanceId: "ins-legacy-001",
		UserID:     user.ID,
		AgentType:  "", // 空类型应视为 openclaw
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	form := url.Values{"id": {uintToString(inst.ID)}}
	req := userTerminalReqWithSession(t, http.MethodPost, "/openclaw/terminal-url", "testuser", form.Encode())
	rr := httptest.NewRecorder()

	handleInstanceTerminal(rr, req, testCVMFetcher)

	// 空类型视为 openclaw，终端关闭应返回 403
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for empty agent_type with terminal disabled, got %d", rr.Code)
	}
}

// final §3.2 决策：终端由站点级 config.TerminalEnabled 统一控制，三端一视同仁。
// 原 v7 下"Hermes/ACE 不受开关限制"的语义已作废。
func TestHandleInstanceTerminal_HermesBlockedWhenDisabled(t *testing.T) {
	cleanup := initOpenclawTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 终端关闭：三端均应返回 403
	config := model.SiteConfig{ID: 1, TerminalEnabled: false}
	model.DB(context.Background()).Create(&config)

	proxyToken := "sk-test-hermes"
	inst := model.Instance{
		Name:       "hermes-inst",
		InstanceId: "ins-hermes-001",
		UserID:     user.ID,
		AgentType:  "hermes",
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	form := url.Values{"id": {uintToString(inst.ID)}}
	req := userTerminalReqWithSession(t, http.MethodPost, "/openclaw/terminal-url", "testuser", form.Encode())
	rr := httptest.NewRecorder()

	handleInstanceTerminal(rr, req, testCVMFetcher)

	// final §3.2：终端关闭时，Hermes 与 OpenClaw 同样被拦
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 (Hermes blocked by disabled terminal switch, same as OpenClaw), got %d", rr.Code)
	}
}

func TestHandleInstanceTerminal_LightclawACEBlockedWhenDisabled(t *testing.T) {
	cleanup := initOpenclawTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	config := model.SiteConfig{ID: 1, TerminalEnabled: false}
	model.DB(context.Background()).Create(&config)

	proxyToken := "sk-test-lightclaw"
	inst := model.Instance{
		Name:       "lightclaw-inst",
		InstanceId: "ins-lightclaw-001",
		UserID:     user.ID,
		AgentType:  "lightclawace",
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	form := url.Values{"id": {uintToString(inst.ID)}}
	req := userTerminalReqWithSession(t, http.MethodPost, "/openclaw/terminal-url", "testuser", form.Encode())
	rr := httptest.NewRecorder()

	handleInstanceTerminal(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 (LightclawACE blocked by disabled terminal switch), got %d", rr.Code)
	}
}

func TestHandleInstanceTerminal_NoInstanceId(t *testing.T) {
	cleanup := initOpenclawTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	config := model.SiteConfig{ID: 1, TerminalEnabled: true}
	model.DB(context.Background()).Create(&config)

	proxyToken := "sk-test-noid"
	inst := model.Instance{
		Name:       "no-cvm-inst",
		InstanceId: "", // 无关联 CVM
		UserID:     user.ID,
		AgentType:  "openclaw",
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	form := url.Values{"id": {uintToString(inst.ID)}}
	req := userTerminalReqWithSession(t, http.MethodPost, "/openclaw/terminal-url", "testuser", form.Encode())
	rr := httptest.NewRecorder()

	handleInstanceTerminal(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for instance without CVM, got %d", rr.Code)
	}
}

// ==================== HandleResetInstance agent_type 镜像查询 Tests ====================

func TestHandleResetInstance_Unauthenticated(t *testing.T) {
	cleanup := initOpenclawTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/reset?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	handleResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ==================== HandleSetGatewayUi agent_type dispatch Tests ====================

func TestHandleSetGatewayUi_Unauthenticated(t *testing.T) {
	cleanup := initOpenclawTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/gateway-ui?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	handleSetGatewayUi(rr, req, testCVMFetcher)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ==================== HandleCheckGatewayAccess agent_type 校验 Tests ====================

func TestHandleCheckGatewayAccess_Unauthenticated(t *testing.T) {
	cleanup := initOpenclawTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/check-gateway-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleCheckGatewayAccess(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ==================== Helpers ====================

func userTerminalReqWithSession(t *testing.T, method, path string, username string, body string) *http.Request {
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

func uintToString(id uint) string {
	return strings.TrimRight(strings.TrimRight(
		strings.Replace(
			strings.Replace(
				func() string { return fmt.Sprintf("%d", id) }(),
				" ", "", -1),
			"\n", "", -1),
		" "), "\n")
}
