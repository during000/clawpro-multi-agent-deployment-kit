package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ── 测试辅助 ────────────────────────────────────────────────────────

func setupUserMcpTestDB(t *testing.T) {
	t.Helper()

	// 将探测器 HTTP 超时设为极短，使异步探测 goroutine 快速失败退出
	origClient := mcpProber.client
	mcpProber.client = &http.Client{Timeout: 1 * time.Millisecond}
	t.Cleanup(func() { mcpProber.client = origClient })

	// 启用 mcpAsyncWG，让异步探测 goroutine 在 cleanup 时可被等待，
	// 避免 -race 时后台 goroutine 与 DB 恢复操作产生数据竞态。
	var wg sync.WaitGroup
	mcpAsyncWG = &wg

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.McpServer{},
		&model.McpVersion{},
		&model.McpInstallation{},
		&model.Instance{},
		&model.InstanceAdjustment{},
		&model.User{},
		&model.SiteConfig{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.GroupConfigBinding{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	unlock := model.UseDBForTest(db)
	t.Cleanup(func() {
		wg.Wait()
		mcpAsyncWG = nil
		unlock()
		useDBForTestWithSafeRestore(db)()
	})
	db.Create(&model.SiteConfig{})
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
}

func userMcpReqWithSession(t *testing.T, method, path, body, username string) *http.Request {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username

	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

func createTestUserAndInstance(t *testing.T, agentType, agentVersion, state string) (model.User, model.Instance) {
	t.Helper()
	user := model.User{Username: "testuser", Password: "pass"}
	model.DB(nil).Create(&user)

	agentReady := 0
	if state == "RUNNING" {
		agentReady = 1
	}
	instance := model.Instance{
		UserID:          user.ID,
		Name:            "test-instance",
		InstanceId:      "ins-test001",
		AgentType:       agentType,
		AgentVersion:    agentVersion,
		LastStableState: state,
		LastCVMState:    state,
		AgentReady:      agentReady,
		RuntimeUser:     "ubuntu",
	}
	model.DB(nil).Create(&instance)
	return user, instance
}

func createTestMcpServer(t *testing.T) model.McpServer {
	t.Helper()
	server := model.McpServer{
		ServiceID:      "test-mcp",
		Name:           "Test MCP",
		Description:    "A test MCP server",
		TransportType:  "streamable-http",
		VisibilityType: "all",
	}
	model.DB(nil).Create(&server)

	version := model.McpVersion{
		MCPID:         server.ID,
		Version:       "1.0.0",
		TransportType: "streamable-http",
		ConfigJSON:    `{"transportType":"streamable-http","url":"http://10.0.0.1/mcp","headers":{"Authorization":"Bearer <api_key>"}}`,
	}
	model.DB(nil).Create(&version)
	model.DB(nil).Model(&server).Update("latest_version_id", version.ID)

	return server
}

// ── checkInstanceSupportsMcp 测试 ─────────────────────────────────────────

func TestCheckInstanceSupportsMcp_OpenclawOK(t *testing.T) {
	inst := &model.Instance{AgentType: "openclaw", AgentVersion: "2026.4.1"}
	err := checkInstanceSupportsMcp(nil, inst)
	if err != nil {
		t.Errorf("openclaw 2026.4.1 应支持 MCP, got error: %v", err)
	}
}

func TestCheckInstanceSupportsMcp_OpenclawMinVersion(t *testing.T) {
	inst := &model.Instance{AgentType: "openclaw", AgentVersion: "2026.3.28"}
	err := checkInstanceSupportsMcp(nil, inst)
	if err != nil {
		t.Errorf("openclaw 2026.3.28 应支持 MCP, got error: %v", err)
	}
}

func TestCheckInstanceSupportsMcp_VersionTooLow(t *testing.T) {
	inst := &model.Instance{AgentType: "openclaw", AgentVersion: "2026.3.27"}
	err := checkInstanceSupportsMcp(nil, inst)
	if err == nil {
		t.Error("openclaw 2026.3.27 不应支持 MCP, expected error")
	}
}

func TestCheckInstanceSupportsMcp_EmptyVersion(t *testing.T) {
	inst := &model.Instance{AgentType: "openclaw", AgentVersion: ""}
	err := checkInstanceSupportsMcp(nil, inst)
	if err == nil {
		t.Error("空版本号不应支持 MCP, expected error")
	}
}

func TestCheckInstanceSupportsMcp_WrongAgentType(t *testing.T) {
	inst := &model.Instance{AgentType: "hermes", AgentVersion: "2026.4.1"}
	err := checkInstanceSupportsMcp(nil, inst)
	if err == nil {
		t.Error("hermes 类型不应支持 MCP, expected error")
	}
}

func TestCheckInstanceSupportsMcp_NilInstance(t *testing.T) {
	err := checkInstanceSupportsMcp(nil, nil)
	if err != nil {
		t.Errorf("nil instance 应返回 nil, got: %v", err)
	}
}

// ── HandleUserMcpAvailable 测试 ──────────────────────────────────────────

func TestHandleUserMcpAvailable_Unauthorized(t *testing.T) {
	setupUserMcpTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/mcp/available?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleUserMcpAvailable(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleUserMcpAvailable_InstanceNotFound(t *testing.T) {
	setupUserMcpTestDB(t)
	model.DB(nil).Create(&model.User{Username: "testuser", Password: "pass"})

	req := userMcpReqWithSession(t, http.MethodGet, "/openclaw/mcp/available?id=999", "", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAvailable(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleUserMcpAvailable_VersionTooLow(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.3.1", "RUNNING")

	req := userMcpReqWithSession(t, http.MethodGet, fmt.Sprintf("/openclaw/mcp/available?id=%d", instance.ID), "", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAvailable(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("版本过低应返回 403，实际=%d", rr.Code)
	}
}

func TestHandleUserMcpAvailable_Success(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	createTestMcpServer(t)

	req := userMcpReqWithSession(t, http.MethodGet, fmt.Sprintf("/openclaw/mcp/available?id=%d", instance.ID), "", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAvailable(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["total"].(float64) != 1 {
		t.Errorf("应返回 1 个可选 MCP，实际=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	item := items[0].(map[string]interface{})
	if item["service_id"] != "test-mcp" {
		t.Errorf("service_id 不匹配: %v", item["service_id"])
	}
	if item["config_json"] == nil || item["config_json"] == "" {
		t.Error("config_json 不应为空")
	}
}

func TestHandleUserMcpAvailable_ExcludesInstalled(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	server := createTestMcpServer(t)

	// 预安装
	model.DB(nil).Create(&model.McpInstallation{
		InstanceID: instance.ID,
		MCPID:      server.ID,
		ServiceID:  server.ServiceID,
		Name:       server.Name,
	})

	req := userMcpReqWithSession(t, http.MethodGet, fmt.Sprintf("/openclaw/mcp/available?id=%d", instance.ID), "", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAvailable(rr, req)

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["total"].(float64) != 0 {
		t.Errorf("已安装的 MCP 应被排除，实际 total=%v", resp["total"])
	}
}

func TestHandleUserMcpAvailable_VisibilityGroupFiltered(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")

	// 创建 group 可见性的 MCP
	server := model.McpServer{
		ServiceID:      "group-mcp",
		Name:           "Group MCP",
		TransportType:  "sse",
		VisibilityType: "group",
	}
	model.DB(nil).Create(&server)

	req := userMcpReqWithSession(t, http.MethodGet, fmt.Sprintf("/openclaw/mcp/available?id=%d", instance.ID), "", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAvailable(rr, req)

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	// 用户不在任何组，group 类型的 MCP 不可见
	if resp["total"].(float64) != 0 {
		t.Errorf("group 类型 MCP 应被过滤, total=%v", resp["total"])
	}
}

func TestHandleUserMcpAdd_MethodNotAllowed(t *testing.T) {
	setupUserMcpTestDB(t)
	model.DB(nil).Create(&model.User{Username: "testuser", Password: "pass"})

	req := userMcpReqWithSession(t, http.MethodGet, "/openclaw/mcp/add", "", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAdd(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleUserMcpAdd_InvalidBody(t *testing.T) {
	setupUserMcpTestDB(t)
	model.DB(nil).Create(&model.User{Username: "testuser", Password: "pass"})

	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/add", "not json at all{{{", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAdd(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 body 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleUserMcpAdd_VisibilityGroupForbidden(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")

	// 创建 group 可见性的 MCP
	server := model.McpServer{
		ServiceID:      "group-mcp",
		Name:           "Group MCP",
		TransportType:  "sse",
		VisibilityType: "group",
	}
	model.DB(nil).Create(&server)
	version := model.McpVersion{MCPID: server.ID, Version: "1.0.0", TransportType: "sse", ConfigJSON: `{}`}
	model.DB(nil).Create(&version)
	model.DB(nil).Model(&server).Update("latest_version_id", version.ID)

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", nil
	})
	defer restore()

	body := fmt.Sprintf(`{"instance_ids":[%d],"service_id":"group-mcp","config_json":"{\"url\":\"http://10.0.0.1/mcp\"}"}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/add", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAdd(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("无权 MCP 应返回 403，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUserMcpRefreshStatus_MethodNotAllowed(t *testing.T) {
	setupUserMcpTestDB(t)
	model.DB(nil).Create(&model.User{Username: "testuser", Password: "pass"})

	req := userMcpReqWithSession(t, http.MethodGet, "/openclaw/mcp/refresh-status", "", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpRefreshStatus(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleUserMcpUpdateConfig_MethodNotAllowed(t *testing.T) {
	setupUserMcpTestDB(t)
	model.DB(nil).Create(&model.User{Username: "testuser", Password: "pass"})

	req := userMcpReqWithSession(t, http.MethodGet, "/openclaw/mcp/update-config", "", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpUpdateConfig(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleUserMcpDelete_MethodNotAllowed(t *testing.T) {
	setupUserMcpTestDB(t)
	model.DB(nil).Create(&model.User{Username: "testuser", Password: "pass"})

	req := userMcpReqWithSession(t, http.MethodGet, "/openclaw/mcp/delete", "", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpDelete(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleUserMcpToggle_MethodNotAllowed(t *testing.T) {
	setupUserMcpTestDB(t)
	model.DB(nil).Create(&model.User{Username: "testuser", Password: "pass"})

	req := userMcpReqWithSession(t, http.MethodGet, "/openclaw/mcp/toggle", "", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpToggle(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestIsPrivateIP_DomainResolvesToPrivate(t *testing.T) {
	// localhost 应该解析为 127.0.0.1 → private
	if !isPrivateIP("localhost") {
		t.Error("localhost 应被视为内网地址")
	}
}

func TestHandleUserMcpAvailable_NoLatestVersion(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")

	// 创建没有版本的 MCP
	server := model.McpServer{
		ServiceID:      "no-version-mcp",
		Name:           "No Version",
		TransportType:  "stdio",
		VisibilityType: "all",
	}
	model.DB(nil).Create(&server)

	req := userMcpReqWithSession(t, http.MethodGet, fmt.Sprintf("/openclaw/mcp/available?id=%d", instance.ID), "", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAvailable(rr, req)

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("应返回 1 个, got %d", len(items))
	}
	item := items[0].(map[string]interface{})
	// transport_type 应从 server 本身取
	if item["transport_type"] != "stdio" {
		t.Errorf("transport_type 应为 stdio, got %v", item["transport_type"])
	}
}

func TestHandleUserMcpSearch(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	createTestMcpServer(t)

	req := userMcpReqWithSession(t, http.MethodGet, fmt.Sprintf("/openclaw/mcp/available?id=%d&q=nonexist", instance.ID), "", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAvailable(rr, req)

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["total"].(float64) != 0 {
		t.Errorf("搜索不到应返回 0, got %v", resp["total"])
	}
}

// ── HandleUserMcpAdd 测试 ────────────────────────────────────────────────

func TestHandleUserMcpAdd_Unauthorized(t *testing.T) {
	setupUserMcpTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/openclaw/mcp/add", strings.NewReader(`{}`))
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleUserMcpAdd(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleUserMcpAdd_MissingParams(t *testing.T) {
	setupUserMcpTestDB(t)
	model.DB(nil).Create(&model.User{Username: "testuser", Password: "pass"})

	cases := []struct {
		name string
		body string
	}{
		{"empty instance_ids", `{"instance_ids":[],"service_id":"x","config_json":"{}"}`},
		{"missing service_id", `{"instance_ids":[1],"service_id":"","config_json":"{}"}`},
		{"missing config_json", `{"instance_ids":[1],"service_id":"x","config_json":""}`},
		{"invalid json", `{"instance_ids":[1],"service_id":"x","config_json":"not json"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/add", tc.body, "testuser")
			rr := httptest.NewRecorder()
			HandleUserMcpAdd(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("%s: 应返回 400，实际=%d, body=%s", tc.name, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandleUserMcpAdd_McpNotFound(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")

	body := fmt.Sprintf(`{"instance_ids":[%d],"service_id":"nonexist","config_json":"{\"url\":\"http://10.0.0.1/mcp\"}"}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/add", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAdd(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("MCP 不存在应返回 404，实际=%d", rr.Code)
	}
}

func TestHandleUserMcpAdd_InstanceNotRunning(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "STOPPED")
	createTestMcpServer(t)

	body := fmt.Sprintf(`{"instance_ids":[%d],"service_id":"test-mcp","config_json":"{\"url\":\"http://10.0.0.1/mcp\"}"}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/add", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAdd(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	item := items[0].(map[string]interface{})
	if item["status"] != "skipped" {
		t.Errorf("未运行实例应 skipped, got %v", item["status"])
	}
}

func TestHandleUserMcpAdd_TooManyInstances(t *testing.T) {
	setupUserMcpTestDB(t)
	model.DB(nil).Create(&model.User{Username: "testuser", Password: "pass"})

	ids := make([]string, 51)
	for i := range ids {
		ids[i] = fmt.Sprintf("%d", i+1)
	}
	body := fmt.Sprintf(`{"instance_ids":[%s],"service_id":"x","config_json":"{}"}`, strings.Join(ids, ","))
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/add", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAdd(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("超 50 个实例应返回 400，实际=%d", rr.Code)
	}
}

// ── HandleUserMcpList 测试 ───────────────────────────────────────────────

func TestHandleUserMcpList_Success(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	server := createTestMcpServer(t)

	model.DB(nil).Create(&model.McpInstallation{
		InstanceID:       instance.ID,
		MCPID:            server.ID,
		ServiceID:        server.ServiceID,
		Name:             server.Name,
		Version:          "1.0.0",
		InstallStatus:    model.McpInstallSuccess,
		ConfigJSON:       `{"url":"http://10.0.0.1/mcp"}`,
		Source:           "user",
		ConnectionStatus: "connected",
		ToolsJSON:        `["tool1","tool2"]`,
	})

	req := userMcpReqWithSession(t, http.MethodGet, fmt.Sprintf("/openclaw/mcp/list?id=%d", instance.ID), "", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["total"].(float64) != 1 {
		t.Errorf("应返回 1 条，实际=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	item := items[0].(map[string]interface{})
	if item["service_id"] != "test-mcp" {
		t.Errorf("service_id 不匹配: %v", item["service_id"])
	}
	tools := item["tools"].([]interface{})
	if len(tools) != 2 {
		t.Errorf("tools 应有 2 个，实际=%d", len(tools))
	}
}

func TestHandleUserMcpList_IncludesFailed(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	server := createTestMcpServer(t)

	model.DB(nil).Create(&model.McpInstallation{
		InstanceID:    instance.ID,
		MCPID:         server.ID,
		ServiceID:     server.ServiceID,
		Name:          server.Name,
		InstallStatus: model.McpInstallFailed,
		ErrorMessage:  "配置下发失败: timeout",
		ConfigJSON:    `{}`,
		Source:        "user",
	})

	req := userMcpReqWithSession(t, http.MethodGet, fmt.Sprintf("/openclaw/mcp/list?id=%d", instance.ID), "", "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpList(rr, req)

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["total"].(float64) != 1 {
		t.Errorf("Failed 状态也应返回，total=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	item := items[0].(map[string]interface{})
	if int(item["install_status"].(float64)) != model.McpInstallFailed {
		t.Errorf("install_status 应为 Failed(3), got %v", item["install_status"])
	}
}

// ── HandleUserMcpRefreshStatus 测试 ──────────────────────────────────────

func TestHandleUserMcpRefreshStatus_Unauthorized(t *testing.T) {
	setupUserMcpTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/openclaw/mcp/refresh-status", strings.NewReader(`{}`))
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleUserMcpRefreshStatus(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleUserMcpRefreshStatus_EmptyList(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")

	body := fmt.Sprintf(`{"id":%d}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/refresh-status", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpRefreshStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d", rr.Code)
	}
}

// ── HandleUserMcpUpdateConfig 测试 ───────────────────────────────────────

func TestHandleUserMcpUpdateConfig_Unauthorized(t *testing.T) {
	setupUserMcpTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/openclaw/mcp/update-config", strings.NewReader(`{}`))
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleUserMcpUpdateConfig(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleUserMcpUpdateConfig_MissingParams(t *testing.T) {
	setupUserMcpTestDB(t)
	model.DB(nil).Create(&model.User{Username: "testuser", Password: "pass"})

	cases := []struct {
		name string
		body string
	}{
		{"missing service_id", `{"id":1,"service_id":"","config_json":"{}","restart":false}`},
		{"missing config_json", `{"id":1,"service_id":"x","config_json":"","restart":false}`},
		{"invalid config_json", `{"id":1,"service_id":"x","config_json":"not json","restart":false}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/update-config", tc.body, "testuser")
			rr := httptest.NewRecorder()
			HandleUserMcpUpdateConfig(rr, req)
			if rr.Code != http.StatusBadRequest && rr.Code != http.StatusForbidden {
				t.Errorf("%s: 应返回 400/403，实际=%d, body=%s", tc.name, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandleUserMcpUpdateConfig_VersionTooLow(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.3.1", "RUNNING")

	body := fmt.Sprintf(`{"id":%d,"service_id":"x","config_json":"{}","restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/update-config", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpUpdateConfig(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("版本过低应返回 403，实际=%d", rr.Code)
	}
}

// ── HandleUserMcpDelete 测试 ─────────────────────────────────────────────

func TestHandleUserMcpDelete_Unauthorized(t *testing.T) {
	setupUserMcpTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/openclaw/mcp/delete", strings.NewReader(`{}`))
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleUserMcpDelete(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleUserMcpDelete_MissingServiceID(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")

	body := fmt.Sprintf(`{"id":%d,"service_id":"","restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/delete", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpDelete(rr, req)
	// 会走到 requireInstanceRunning（可能 500），或 400
	if rr.Code == http.StatusOK {
		t.Errorf("空 service_id 不应返回 200")
	}
}

// ── HandleUserMcpToggle 测试 ─────────────────────────────────────────────

func TestHandleUserMcpToggle_Unauthorized(t *testing.T) {
	setupUserMcpTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/openclaw/mcp/toggle", strings.NewReader(`{}`))
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleUserMcpToggle(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleUserMcpToggle_VersionTooLow(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "hermes", "2026.4.1", "RUNNING")

	body := fmt.Sprintf(`{"id":%d,"service_id":"x","disabled":true,"restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/toggle", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpToggle(rr, req)
	// hermes 类型不支持 MCP → 409 (via writeAgentGuardError) 或 500 (requireInstanceRunning hits CVM)
	if rr.Code == http.StatusOK {
		t.Errorf("hermes 类型不应成功")
	}
}

// ── MCP Probe 测试 ───────────────────────────────────────────────────────

func TestContainsPlaceholder(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{`{"url":"https://example.com","headers":{"Authorization":"Bearer <api_key>"}}`, true},
		{`{"url":"https://example.com","key":"<base_url>"}`, true},
		{`{"url":"https://example.com"}`, false},
		{`{"url":"<>"}`, false},    // 空尖括号不算
		{`{"url":"<123>"}`, false}, // 数字开头不算
		{`{"url":"<a>"}`, true},
		{`{"url":"<_private>"}`, true},
	}

	for _, tc := range cases {
		got := containsPlaceholder(tc.input)
		if got != tc.expected {
			t.Errorf("containsPlaceholder(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestIsPrivateIP(t *testing.T) {
	cases := []struct {
		host     string
		expected bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"127.0.0.1:8080", true},
		{"8.8.8.8:443", false},
	}

	for _, tc := range cases {
		got := isPrivateIP(tc.host)
		if got != tc.expected {
			t.Errorf("isPrivateIP(%q) = %v, want %v", tc.host, got, tc.expected)
		}
	}
}

func TestProbeOne_Placeholder(t *testing.T) {
	input := McpProbeInput{
		ServiceID:     "test",
		TransportType: "streamable-http",
		ConfigJSON:    `{"url":"https://example.com","headers":{"Authorization":"Bearer <api_key>"}}`,
	}
	result := mcpProber.probeOne(context.Background(), input)
	if result.ConnectionStatus != "unconfigured" {
		t.Errorf("含占位符应返回 unconfigured, got %s", result.ConnectionStatus)
	}
}

func TestProbeOne_STDIO(t *testing.T) {
	input := McpProbeInput{
		ServiceID:     "test",
		TransportType: "stdio",
		ConfigJSON:    `{"command":"node","args":["server.js"]}`,
	}
	result := mcpProber.probeOne(context.Background(), input)
	if result.ConnectionStatus != "unsupported" {
		t.Errorf("STDIO 应返回 unsupported, got %s", result.ConnectionStatus)
	}
}

func TestProbeOne_InvalidJSON(t *testing.T) {
	input := McpProbeInput{
		ServiceID:     "test",
		TransportType: "streamable-http",
		ConfigJSON:    `not json`,
	}
	result := mcpProber.probeOne(context.Background(), input)
	if result.ConnectionStatus != "failed" {
		t.Errorf("非法 JSON 应返回 failed, got %s", result.ConnectionStatus)
	}
}

func TestProbeOne_MissingURL(t *testing.T) {
	input := McpProbeInput{
		ServiceID:     "test",
		TransportType: "streamable-http",
		ConfigJSON:    `{"headers":{}}`,
	}
	result := mcpProber.probeOne(context.Background(), input)
	if result.ConnectionStatus != "failed" {
		t.Errorf("缺少 url 应返回 failed, got %s", result.ConnectionStatus)
	}
}

func TestProbeOne_PrivateIP(t *testing.T) {
	input := McpProbeInput{
		ServiceID:     "test",
		TransportType: "streamable-http",
		ConfigJSON:    `{"url":"http://192.168.1.1:8080/mcp"}`,
	}
	result := mcpProber.probeOne(context.Background(), input)
	if result.ConnectionStatus != "failed" {
		t.Errorf("内网地址应返回 failed, got %s", result.ConnectionStatus)
	}
	if !strings.Contains(result.Error, "内网") {
		t.Errorf("错误信息应包含'内网', got %s", result.Error)
	}
}

func TestProbeOne_InvalidURL(t *testing.T) {
	input := McpProbeInput{
		ServiceID:     "test",
		TransportType: "streamable-http",
		ConfigJSON:    `{"url":"://bad-url"}`,
	}
	result := mcpProber.probeOne(context.Background(), input)
	if result.ConnectionStatus != "failed" {
		t.Errorf("无效 URL 应返回 failed, got %s", result.ConnectionStatus)
	}
}

func TestMcpProber_TryAcquireInstance(t *testing.T) {
	// 首次获取应成功
	if !mcpProber.TryAcquireInstance(999) {
		t.Error("首次获取应成功")
	}
	// 重复获取应失败
	if mcpProber.TryAcquireInstance(999) {
		t.Error("重复获取应失败")
	}
	// 释放后再获取应成功
	mcpProber.ReleaseInstance(999)
	if !mcpProber.TryAcquireInstance(999) {
		t.Error("释放后获取应成功")
	}
	mcpProber.ReleaseInstance(999)
}

func TestParseToolsList(t *testing.T) {
	resp := map[string]interface{}{
		"result": map[string]interface{}{
			"tools": []interface{}{
				map[string]interface{}{"name": "tool_a", "description": "desc a"},
				map[string]interface{}{"name": "tool_b", "description": "desc b"},
			},
		},
	}
	tools := parseToolsList(resp)
	if len(tools) != 2 {
		t.Fatalf("应解析出 2 个工具, got %d", len(tools))
	}
	if tools[0] != "tool_a" || tools[1] != "tool_b" {
		t.Errorf("工具名不匹配: %v", tools)
	}
}

func TestParseToolsList_Empty(t *testing.T) {
	tools := parseToolsList(map[string]interface{}{})
	if len(tools) != 0 {
		t.Errorf("空响应应返回空列表, got %v", tools)
	}
}

func TestExtractHeaders(t *testing.T) {
	config := map[string]interface{}{
		"url": "https://example.com",
		"headers": map[string]interface{}{
			"Authorization": "Bearer token",
			"X-Custom":      "value",
		},
	}
	headers := extractHeaders(config)
	if headers["Authorization"] != "Bearer token" {
		t.Errorf("Authorization 不匹配: %v", headers["Authorization"])
	}
	if headers["X-Custom"] != "value" {
		t.Errorf("X-Custom 不匹配: %v", headers["X-Custom"])
	}
}

func TestExtractHeaders_NoHeaders(t *testing.T) {
	config := map[string]interface{}{"url": "https://example.com"}
	headers := extractHeaders(config)
	if len(headers) != 0 {
		t.Errorf("无 headers 应返回空 map, got %v", headers)
	}
}

func TestIsPrivateAddr(t *testing.T) {
	cases := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"10.255.0.1", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"172.32.0.1", false},
		{"192.168.0.1", true},
		{"169.254.1.1", true},
		{"8.8.8.8", false},
		{"::1", true},
	}

	for _, tc := range cases {
		ip := net.ParseIP(tc.ip)
		if ip == nil {
			t.Fatalf("无法解析 IP: %s", tc.ip)
		}
		got := isPrivateAddr(ip)
		if got != tc.expected {
			t.Errorf("isPrivateAddr(%s) = %v, want %v", tc.ip, got, tc.expected)
		}
	}
}

// ── MCP Probe 集成测试（mock HTTP server）───────────────────────────────

func TestProbeStreamableHTTP_Success(t *testing.T) {
	// 模拟 MCP 服务器：响应 initialize + tools/list
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		method, _ := req["method"].(string)
		callCount++

		switch method {
		case "initialize":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]interface{}{},
					"serverInfo":      map[string]interface{}{"name": "test-mcp", "version": "1.0.0"},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusOK)
		case "tools/list":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]interface{}{
					"tools": []interface{}{
						map[string]interface{}{"name": "tool_a", "description": "Tool A"},
						map[string]interface{}{"name": "tool_b", "description": "Tool B"},
					},
				},
			})
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	tools, err := mcpProber.probeStreamableHTTP(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("probeStreamableHTTP 失败: %v", err)
	}
	if len(tools) != 2 {
		t.Errorf("应返回 2 个工具, got %d: %v", len(tools), tools)
	}
	if tools[0] != "tool_a" || tools[1] != "tool_b" {
		t.Errorf("工具名不匹配: %v", tools)
	}
}

func TestProbeStreamableHTTP_InitializeFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := mcpProber.probeStreamableHTTP(context.Background(), server.URL, nil)
	if err == nil {
		t.Error("服务器 500 应返回错误")
	}
}

func TestProbeStreamableHTTP_InitializeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"error":   map[string]interface{}{"code": -32600, "message": "Invalid Request"},
		})
	}))
	defer server.Close()

	_, err := mcpProber.probeStreamableHTTP(context.Background(), server.URL, nil)
	if err == nil {
		t.Error("initialize 返回 error 应失败")
	}
}

func TestProbeStreamableHTTP_WithHeaders(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": map[string]interface{}{}, "serverInfo": map[string]interface{}{"name": "test"}},
		})
	}))
	defer server.Close()

	headers := map[string]string{"Authorization": "Bearer test-token"}
	mcpProber.probeStreamableHTTP(context.Background(), server.URL, headers)
	if receivedAuth != "Bearer test-token" {
		t.Errorf("headers 未正确传递, got: %s", receivedAuth)
	}
}

func TestProbe_BatchWithContext(t *testing.T) {
	inputs := []McpProbeInput{
		{ServiceID: "stdio-mcp", TransportType: "stdio", ConfigJSON: `{"command":"node"}`},
		{ServiceID: "placeholder-mcp", TransportType: "sse", ConfigJSON: `{"url":"http://x.com","headers":{"Authorization":"Bearer <key>"}}`},
		{ServiceID: "invalid-mcp", TransportType: "streamable-http", ConfigJSON: `{"url":"http://203.0.113.1:9999/mcp"}`},
	}

	results := mcpProber.Probe(context.Background(), inputs)
	if len(results) != 3 {
		t.Fatalf("应返回 3 个结果, got %d", len(results))
	}

	// stdio → unsupported
	if results[0].ConnectionStatus != "unsupported" {
		t.Errorf("stdio 应 unsupported, got %s", results[0].ConnectionStatus)
	}
	// placeholder → unconfigured
	if results[1].ConnectionStatus != "unconfigured" {
		t.Errorf("placeholder 应 unconfigured, got %s", results[1].ConnectionStatus)
	}
	// 不可达公网地址 → failed（不是因为 SSRF）
	if results[2].ConnectionStatus != "failed" {
		t.Errorf("不可达地址应 failed, got %s", results[2].ConnectionStatus)
	}
}

func TestDoJSONRPC_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type 应为 application/json")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "result": "ok"})
	}))
	defer server.Close()

	resp, err := mcpProber.doJSONRPC(context.Background(), server.URL, nil, map[string]interface{}{"method": "test"})
	if err != nil {
		t.Fatalf("doJSONRPC 失败: %v", err)
	}
	if resp["result"] != "ok" {
		t.Errorf("响应不匹配: %v", resp)
	}
}

func TestDoJSONRPC_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := mcpProber.doJSONRPC(context.Background(), server.URL, nil, map[string]interface{}{"method": "test"})
	if err == nil {
		t.Error("503 应返回错误")
	}
}

func TestDoJSONRPC_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	_, err := mcpProber.doJSONRPC(context.Background(), server.URL, nil, map[string]interface{}{"method": "test"})
	if err == nil {
		t.Error("非 JSON 响应应返回错误")
	}
}

func TestProbeOne_ConnectSuccess(t *testing.T) {
	// probeOne 会做 SSRF 检查（127.0.0.1 会被拒绝），所以用外部公网地址测试不现实。
	// 这里验证 probeOne 对合法公网 URL 的 JSON 解析和路由逻辑：
	// 使用一个不可达但非内网的 IP（TEST-NET-1: 192.0.2.x），验证它走到 HTTP 请求阶段。
	input := McpProbeInput{
		ServiceID:     "live-mcp",
		TransportType: "streamable-http",
		ConfigJSON:    `{"url":"http://203.0.113.1:9999/mcp"}`, // TEST-NET-3, 公网但不可达
	}
	result := mcpProber.probeOne(context.Background(), input)
	// 非内网，应该尝试连接（然后超时/连接拒绝 → failed）
	if result.ConnectionStatus != "failed" {
		t.Errorf("不可达公网地址应 failed, got %s", result.ConnectionStatus)
	}
	// 不应因 SSRF 拒绝
	if strings.Contains(result.Error, "内网") {
		t.Errorf("公网地址不应被 SSRF 拒绝: %s", result.Error)
	}
}

func TestProbeOne_ConnectFailed(t *testing.T) {
	input := McpProbeInput{
		ServiceID:     "unreachable",
		TransportType: "streamable-http",
		ConfigJSON:    `{"url":"http://192.0.2.1:9999/mcp"}`, // TEST-NET, unreachable
	}
	result := mcpProber.probeOne(context.Background(), input)
	if result.ConnectionStatus != "failed" {
		t.Errorf("不可达地址应 failed, got %s", result.ConnectionStatus)
	}
}

// ── 带 mock RunScript 的 Handler 完整路径测试 ────────────────────────────

func TestHandleUserMcpAdd_FullSuccess(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	createTestMcpServer(t)

	// mock RunScript 返回成功
	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", nil
	})
	defer restore()

	body := fmt.Sprintf(`{"instance_ids":[%d],"service_id":"test-mcp","config_json":"{\"url\":\"http://10.0.0.1/mcp\"}"}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/add", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAdd(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if int(resp["success"].(float64)) != 1 {
		t.Errorf("应成功 1 个, got %v", resp["success"])
	}

	// 验证 DB
	var installation model.McpInstallation
	model.DB(nil).Where("instance_id = ? AND service_id = ?", instance.ID, "test-mcp").First(&installation)
	if installation.InstallStatus != model.McpInstallSuccess {
		t.Errorf("install_status 应为 Success, got %d", installation.InstallStatus)
	}
	if installation.Source != "user" {
		t.Errorf("source 应为 user, got %s", installation.Source)
	}
	if installation.ConfigJSON != `{"url":"http://10.0.0.1/mcp"}` {
		t.Errorf("config_json 不匹配: %s", installation.ConfigJSON)
	}
}

func TestHandleUserMcpAdd_TATFailed(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	createTestMcpServer(t)

	// mock RunScript 返回失败
	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATInvocationTimeout)
	})
	defer restore()

	body := fmt.Sprintf(`{"instance_ids":[%d],"service_id":"test-mcp","config_json":"{\"url\":\"http://10.0.0.1/mcp\"}"}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/add", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpAdd(rr, req)

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if int(resp["failed"].(float64)) != 1 {
		t.Errorf("应失败 1 个, got %v", resp["failed"])
	}

	// 验证 DB 标记 Failed
	var installation model.McpInstallation
	model.DB(nil).Where("instance_id = ? AND service_id = ?", instance.ID, "test-mcp").First(&installation)
	if installation.InstallStatus != model.McpInstallFailed {
		t.Errorf("install_status 应为 Failed, got %d", installation.InstallStatus)
	}
}

func TestHandleUserMcpUpdateConfig_Success(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	server := createTestMcpServer(t)

	// 创建安装记录
	model.DB(nil).Create(&model.McpInstallation{
		InstanceID:    instance.ID,
		MCPID:         server.ID,
		ServiceID:     server.ServiceID,
		Name:          server.Name,
		Version:       "1.0.0",
		InstallStatus: model.McpInstallSuccess,
		ConfigJSON:    `{"url":"http://10.0.0.2/old"}`,
		Source:        "user",
	})

	// mock RunScript 成功
	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", nil
	})
	defer restore()

	body := fmt.Sprintf(`{"id":%d,"service_id":"test-mcp","config_json":"{\"url\":\"http://10.0.0.3/new\"}","restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/update-config", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpUpdateConfig(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["ok"] != true {
		t.Errorf("ok 应为 true, got %v", resp["ok"])
	}

	// 验证 DB 更新
	var installation model.McpInstallation
	model.DB(nil).Where("instance_id = ? AND service_id = ?", instance.ID, "test-mcp").First(&installation)
	if installation.ConfigJSON != `{"url":"http://10.0.0.3/new"}` {
		t.Errorf("config_json 不匹配: %s", installation.ConfigJSON)
	}
}

func TestHandleUserMcpUpdateConfig_NotInstalled(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")

	body := fmt.Sprintf(`{"id":%d,"service_id":"nonexist","config_json":"{\"url\":\"http://10.0.0.1/mcp\"}","restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/update-config", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpUpdateConfig(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("未安装 MCP 应返回 404，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUserMcpUpdateConfig_InstanceNotRunning(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "STOPPED")

	body := fmt.Sprintf(`{"id":%d,"service_id":"test-mcp","config_json":"{\"url\":\"http://10.0.0.1/mcp\"}","restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/update-config", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpUpdateConfig(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("实例未运行应返回 409，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUserMcpUpdateConfig_TATFailed(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	server := createTestMcpServer(t)

	model.DB(nil).Create(&model.McpInstallation{
		InstanceID:    instance.ID,
		MCPID:         server.ID,
		ServiceID:     server.ServiceID,
		Name:          server.Name,
		Version:       "1.0.0",
		InstallStatus: model.McpInstallSuccess,
		ConfigJSON:    `{"url":"http://10.0.0.2/old"}`,
		Source:        "user",
	})

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATInvocationTimeout)
	})
	defer restore()

	body := fmt.Sprintf(`{"id":%d,"service_id":"test-mcp","config_json":"{\"url\":\"http://10.0.0.3/new\"}","restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/update-config", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpUpdateConfig(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["ok"] != false {
		t.Errorf("TAT 失败时 ok 应为 false, got %v", resp["ok"])
	}

	// DB 应标记失败
	var installation model.McpInstallation
	model.DB(nil).Where("instance_id = ? AND service_id = ?", instance.ID, "test-mcp").First(&installation)
	if installation.InstallStatus != model.McpInstallFailed {
		t.Errorf("install_status 应为 Failed, got %d", installation.InstallStatus)
	}
}

func TestHandleUserMcpDelete_Success(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	server := createTestMcpServer(t)

	model.DB(nil).Create(&model.McpInstallation{
		InstanceID:    instance.ID,
		MCPID:         server.ID,
		ServiceID:     server.ServiceID,
		Name:          server.Name,
		Version:       "1.0.0",
		InstallStatus: model.McpInstallSuccess,
		ConfigJSON:    `{"url":"http://10.0.0.1/mcp"}`,
		Source:        "user",
	})

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", nil
	})
	defer restore()

	body := fmt.Sprintf(`{"id":%d,"service_id":"test-mcp","restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/delete", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpDelete(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["ok"] != true {
		t.Errorf("ok 应为 true, got %v", resp["ok"])
	}

	// 验证 DB 记录已删除
	var count int64
	model.DB(nil).Model(&model.McpInstallation{}).Where("instance_id = ? AND service_id = ?", instance.ID, "test-mcp").Count(&count)
	if count != 0 {
		t.Errorf("安装记录应已删除，但仍有 %d 条", count)
	}
}

func TestHandleUserMcpDelete_TATFailed(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	server := createTestMcpServer(t)

	model.DB(nil).Create(&model.McpInstallation{
		InstanceID:    instance.ID,
		MCPID:         server.ID,
		ServiceID:     server.ServiceID,
		Name:          server.Name,
		Version:       "1.0.0",
		InstallStatus: model.McpInstallSuccess,
		ConfigJSON:    `{"url":"http://10.0.0.1/mcp"}`,
		Source:        "user",
	})

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	})
	defer restore()

	body := fmt.Sprintf(`{"id":%d,"service_id":"test-mcp","restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/delete", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpDelete(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["ok"] != false {
		t.Errorf("TAT 失败时 ok 应为 false, got %v", resp["ok"])
	}

	// 验证 DB 记录未删除，标记为 Failed
	var installation model.McpInstallation
	model.DB(nil).Where("instance_id = ? AND service_id = ?", instance.ID, "test-mcp").First(&installation)
	if installation.InstallStatus != model.McpInstallFailed {
		t.Errorf("install_status 应为 Failed, got %d", installation.InstallStatus)
	}
}

func TestHandleUserMcpDelete_NotInstalled(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")

	body := fmt.Sprintf(`{"id":%d,"service_id":"nonexist","restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/delete", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpDelete(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("未安装 MCP 应返回 404，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUserMcpDelete_InstanceNotRunning(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "STOPPED")

	body := fmt.Sprintf(`{"id":%d,"service_id":"test-mcp","restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/delete", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpDelete(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("实例未运行应返回 409，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUserMcpDelete_WithRestart(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	server := createTestMcpServer(t)

	model.DB(nil).Create(&model.McpInstallation{
		InstanceID:    instance.ID,
		MCPID:         server.ID,
		ServiceID:     server.ServiceID,
		Name:          server.Name,
		Version:       "1.0.0",
		InstallStatus: model.McpInstallSuccess,
		ConfigJSON:    `{"url":"http://10.0.0.1/mcp"}`,
		Source:        "user",
	})

	var scripts []string
	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		scripts = append(scripts, scriptName)
		return "", nil
	})
	defer restore()

	body := fmt.Sprintf(`{"id":%d,"service_id":"test-mcp","restart":true}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/delete", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpDelete(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["restarted"] != true {
		t.Errorf("restarted 应为 true, got %v", resp["restarted"])
	}

	// 验证调用了 mcp_del.sh 和 restart_gateway.sh
	if len(scripts) != 2 {
		t.Fatalf("应调用 2 个脚本, 实际=%d: %v", len(scripts), scripts)
	}
	if scripts[0] != "mcp_del.sh" || scripts[1] != "restart_gateway.sh" {
		t.Errorf("脚本顺序不对: %v", scripts)
	}
}

func TestHandleUserMcpToggle_Success(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	server := createTestMcpServer(t)

	model.DB(nil).Create(&model.McpInstallation{
		InstanceID:    instance.ID,
		MCPID:         server.ID,
		ServiceID:     server.ServiceID,
		Name:          server.Name,
		Version:       "1.0.0",
		InstallStatus: model.McpInstallSuccess,
		ConfigJSON:    `{"url":"http://10.0.0.1/mcp"}`,
		Source:        "user",
	})

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", nil
	})
	defer restore()

	// 禁用
	body := fmt.Sprintf(`{"id":%d,"service_id":"test-mcp","disabled":true,"restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/toggle", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpToggle(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["ok"] != true {
		t.Errorf("ok 应为 true, got %v", resp["ok"])
	}

	// 验证 config_json 包含 disabled: true
	var installation model.McpInstallation
	model.DB(nil).Where("instance_id = ? AND service_id = ?", instance.ID, "test-mcp").First(&installation)
	var configMap map[string]interface{}
	json.Unmarshal([]byte(installation.ConfigJSON), &configMap)
	if configMap["disabled"] != true {
		t.Errorf("disabled 应为 true, config_json=%s", installation.ConfigJSON)
	}
}

func TestHandleUserMcpToggle_Enable(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	server := createTestMcpServer(t)

	model.DB(nil).Create(&model.McpInstallation{
		InstanceID:    instance.ID,
		MCPID:         server.ID,
		ServiceID:     server.ServiceID,
		Name:          server.Name,
		Version:       "1.0.0",
		InstallStatus: model.McpInstallSuccess,
		ConfigJSON:    `{"url":"http://10.0.0.1/mcp","disabled":true}`,
		Source:        "user",
	})

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", nil
	})
	defer restore()

	// 启用（disabled=false）
	body := fmt.Sprintf(`{"id":%d,"service_id":"test-mcp","disabled":false,"restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/toggle", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpToggle(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	// 验证 config_json 不含 disabled
	var installation model.McpInstallation
	model.DB(nil).Where("instance_id = ? AND service_id = ?", instance.ID, "test-mcp").First(&installation)
	var configMap map[string]interface{}
	json.Unmarshal([]byte(installation.ConfigJSON), &configMap)
	if _, has := configMap["disabled"]; has {
		t.Errorf("启用后不应有 disabled 字段, config_json=%s", installation.ConfigJSON)
	}
}

func TestHandleUserMcpToggle_InstanceNotRunning(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "STOPPED")

	body := fmt.Sprintf(`{"id":%d,"service_id":"test-mcp","disabled":true,"restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/toggle", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpToggle(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("实例未运行应返回 409，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUserMcpToggle_NotInstalled(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")

	body := fmt.Sprintf(`{"id":%d,"service_id":"nonexist","disabled":true,"restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/toggle", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpToggle(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("未安装 MCP 应返回 404，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUserMcpToggle_TATFailed(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	server := createTestMcpServer(t)

	model.DB(nil).Create(&model.McpInstallation{
		InstanceID:    instance.ID,
		MCPID:         server.ID,
		ServiceID:     server.ServiceID,
		Name:          server.Name,
		Version:       "1.0.0",
		InstallStatus: model.McpInstallSuccess,
		ConfigJSON:    `{"url":"http://10.0.0.1/mcp"}`,
		Source:        "user",
	})

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	})
	defer restore()

	body := fmt.Sprintf(`{"id":%d,"service_id":"test-mcp","disabled":true,"restart":false}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/toggle", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpToggle(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["ok"] != false {
		t.Errorf("TAT 失败时 ok 应为 false, got %v", resp["ok"])
	}

	// 验证 DB 标记失败
	var installation model.McpInstallation
	model.DB(nil).Where("instance_id = ? AND service_id = ?", instance.ID, "test-mcp").First(&installation)
	if installation.InstallStatus != model.McpInstallFailed {
		t.Errorf("install_status 应为 Failed, got %d", installation.InstallStatus)
	}
}

func TestHandleUserMcpRefreshStatus_Success(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	server := createTestMcpServer(t)

	// 安装一个 stdio 类型的 MCP（probeOne 会返回 unsupported，不需要网络）
	model.DB(nil).Create(&model.McpInstallation{
		InstanceID:    instance.ID,
		MCPID:         server.ID,
		ServiceID:     server.ServiceID,
		Name:          server.Name,
		Version:       "1.0.0",
		InstallStatus: model.McpInstallSuccess,
		ConfigJSON:    `{"command":"npx","args":["-y","@mcp/test"]}`,
		Source:        "user",
	})

	// 修改 server transport_type 为 stdio（探测会返回 unsupported，无需网络）
	model.DB(nil).Model(&model.McpServer{}).Where("id = ?", server.ID).Update("transport_type", "stdio")

	body := fmt.Sprintf(`{"id":%d}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/refresh-status", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpRefreshStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("应有 1 个探测结果, got %d", len(items))
	}
	item := items[0].(map[string]interface{})
	if item["connection_status"] != "unsupported" {
		t.Errorf("stdio 类型应为 unsupported, got %v", item["connection_status"])
	}
}

func TestHandleUserMcpRefreshStatus_DedupLock(t *testing.T) {
	setupUserMcpTestDB(t)
	_, instance := createTestUserAndInstance(t, "openclaw", "2026.4.1", "RUNNING")
	createTestMcpServer(t)

	// 手动锁住实例
	mcpProber.TryAcquireInstance(instance.ID)
	defer mcpProber.ReleaseInstance(instance.ID)

	body := fmt.Sprintf(`{"id":%d}`, instance.ID)
	req := userMcpReqWithSession(t, http.MethodPost, "/openclaw/mcp/refresh-status", body, "testuser")
	rr := httptest.NewRecorder()
	HandleUserMcpRefreshStatus(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("重复刷新应返回 409，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}
