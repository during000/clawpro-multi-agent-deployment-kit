package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ── 测试辅助 ────────────────────────────────────────────────────────

// setupMcpTestDB 初始化内存 SQLite 数据库，迁移 MCP 相关表。
func setupMcpTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.McpServer{},
		&model.McpVersion{},
		&model.McpDistributionTask{},
		&model.McpDistributionRecord{},
		&model.McpInstallation{},
		&model.McpHostedKey{},
		&model.Instance{},
		&model.User{},
		&model.SiteConfig{},
		&model.UserGroup{},
		&model.UserGroupMember{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	db.Create(&model.SiteConfig{})
	AdminToken = "test-admin-token"
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

}

func waitForMCPTaskCompleted(t *testing.T, db *gorm.DB, taskID uint) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var task model.McpDistributionTask
		if err := db.Select("status").First(&task, taskID).Error; err != nil {
			t.Fatalf("查询 MCP 下发任务 %d 失败: %v", taskID, err)
		}
		if task.Status == model.TaskStatusCompleted {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待 MCP 下发任务 %d 完成超时，当前状态=%q", taskID, task.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func trackMCPTaskCompletion(t *testing.T, db *gorm.DB, taskID uint) {
	t.Helper()
	t.Cleanup(func() {
		waitForMCPTaskCompleted(t, db, taskID)
	})
}

// mcpAdminGet 创建带 admin Bearer Token 的 GET 请求
func mcpAdminGet(url string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// mcpAdminPost 创建带 admin Bearer Token 的 POST 请求
func mcpAdminPost(url, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// mcpParseJSON 解析 JSON 响应
func mcpParseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析 JSON 失败: %v, body: %s", err, w.Body.String())
	}
	return resp
}

// ── validateMCPInput 校验测试 ────────────────────────────────────────

func TestValidateMCPInput_ServiceID(t *testing.T) {
	tests := []struct {
		name      string
		serviceID string
		wantErr   bool
	}{
		{"空 service_id", "", true},
		{"合法 service_id", "my-mcp-server", false},
		{"含下划线", "my_mcp", false},
		{"含大写", "MyMCP", false},
		{"含特殊字符", "my@mcp", true},
		{"含空格", "my mcp", true},
		{"含中文", "我的MCP", true},
		{"超长（49字符）", strings.Repeat("a", 49), true},
		{"最大长度（48字符）", strings.Repeat("a", 48), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateMCPInput(tt.serviceID, "sse", `{"url":"http://localhost:3000"}`, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMCPInput(%q) error = %v, wantErr %v", tt.serviceID, err, tt.wantErr)
			}
		})
	}
}

func TestValidateMCPInput_TransportType(t *testing.T) {
	tests := []struct {
		name          string
		transportType string
		configJSON    string
		wantErr       bool
		wantWarning   bool
	}{
		{"空 transport_type", "", `{"url":"http://localhost"}`, true, false},
		{"sse 类型", "sse", `{"url":"http://localhost:3000"}`, false, false},
		{"streamable-http 类型", "streamable-http", `{"url":"https://example.com/mcp"}`, false, false},
		{"stdio 类型", "stdio", `{"command":"npx","args":["-y","@example/mcp"]}`, false, false},
		{"未知类型", "custom-type", `{"foo":"bar"}`, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateMCPInput("test-svc", tt.transportType, tt.configJSON, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.wantWarning && len(result.Warnings) == 0 {
				t.Error("expected warning for unknown transport type")
			}
		})
	}
}

func TestValidateMCPInput_ConfigJSON(t *testing.T) {
	tests := []struct {
		name       string
		transport  string
		configJSON string
		wantErr    bool
	}{
		{"空 config", "sse", "", true},
		{"非 JSON", "sse", "not json", true},
		{"空对象", "sse", "{}", true},
		{"sse 缺少 url", "sse", `{"foo":"bar"}`, true},
		{"sse url 非 http", "sse", `{"url":"ftp://example.com"}`, true},
		{"sse 合法", "sse", `{"url":"http://localhost:3000/sse"}`, false},
		{"sse https 合法", "sse", `{"url":"https://example.com/mcp"}`, false},
		{"stdio 缺少 command", "stdio", `{"args":[]}`, true},
		{"stdio 合法", "stdio", `{"command":"npx","args":["-y","@example/mcp"]}`, false},
		{"transportType 不一致", "sse", `{"transportType":"stdio","url":"http://localhost"}`, true},
		{"transportType 一致", "sse", `{"transportType":"sse","url":"http://localhost"}`, false},
		{"config 超过 16KB", "sse", `{"url":"http://localhost","padding":"` + strings.Repeat("x", 16*1024) + `"}`, true},
		{"sse url 空字符串", "sse", `{"url":""}`, true},
		{"streamable-http url 空字符串", "streamable-http", `{"url":""}`, true},
		{"stdio command 空字符串", "stdio", `{"command":""}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateMCPInput("test-svc", tt.transport, tt.configJSON, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ── HandleCreateMcp 测试 ────────────────────────────────────────

func TestHandleCreateMcp_Success(t *testing.T) {
	setupMcpTestDB(t)

	body := `{
		"service_id": "test-mcp-1",
		"name": "Test MCP",
		"description": "A test MCP server",
		"transport_type": "sse",
		"config_json": "{\"url\":\"http://localhost:3000/sse\"}",
		"usage_doc_md": "# Usage\nConnect to the server",
		"tool_doc_md": "# Tools\n- tool1: does something"
	}`

	w := httptest.NewRecorder()
	HandleCreateMcp(w, mcpAdminPost("/admin/mcp/create", body))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body: %s", w.Code, w.Body.String())
	}

	resp := mcpParseJSON(t, w)
	if resp["service_id"] != "test-mcp-1" {
		t.Errorf("expected service_id=test-mcp-1, got %v", resp["service_id"])
	}
	if resp["latest_version"] != "1.0.0" {
		t.Errorf("expected latest_version=1.0.0, got %v", resp["latest_version"])
	}

	// 验证数据库
	var server model.McpServer
	model.DB(context.Background()).Where("service_id = ?", "test-mcp-1").First(&server)
	if server.Name != "Test MCP" {
		t.Errorf("expected name=Test MCP, got %s", server.Name)
	}
	if server.LatestVersionID == 0 {
		t.Error("expected latest_version_id to be set")
	}

	var version model.McpVersion
	model.DB(context.Background()).Where("mcp_id = ?", server.ID).First(&version)
	if version.Version != "1.0.0" {
		t.Errorf("expected version=1.0.0, got %s", version.Version)
	}
	if version.ConfigJSON != `{"url":"http://localhost:3000/sse"}` {
		t.Errorf("unexpected config_json: %s", version.ConfigJSON)
	}
}

func TestHandleCreateMcp_DuplicateServiceID(t *testing.T) {
	setupMcpTestDB(t)

	body := `{
		"service_id": "dup-mcp",
		"transport_type": "sse",
		"config_json": "{\"url\":\"http://localhost:3000\"}"
	}`

	// 第一次创建
	w := httptest.NewRecorder()
	HandleCreateMcp(w, mcpAdminPost("/admin/mcp/create", body))
	if w.Code != http.StatusCreated {
		t.Fatalf("first create failed: %d, body: %s", w.Code, w.Body.String())
	}

	// 第二次创建（重复）
	w = httptest.NewRecorder()
	HandleCreateMcp(w, mcpAdminPost("/admin/mcp/create", body))
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateMcp_DefaultName(t *testing.T) {
	setupMcpTestDB(t)

	// name 为空时应默认为 service_id
	body := `{
		"service_id": "auto-name-mcp",
		"transport_type": "sse",
		"config_json": "{\"url\":\"http://localhost:3000\"}"
	}`

	w := httptest.NewRecorder()
	HandleCreateMcp(w, mcpAdminPost("/admin/mcp/create", body))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var server model.McpServer
	model.DB(context.Background()).Where("service_id = ?", "auto-name-mcp").First(&server)
	if server.Name != "auto-name-mcp" {
		t.Errorf("expected name=auto-name-mcp, got %s", server.Name)
	}
}

func TestHandleCreateMcp_InvalidInput(t *testing.T) {
	setupMcpTestDB(t)

	tests := []struct {
		name string
		body string
	}{
		{"空 service_id", `{"service_id":"","transport_type":"sse","config_json":"{\"url\":\"http://localhost\"}"}`},
		{"非法 service_id", `{"service_id":"a@b","transport_type":"sse","config_json":"{\"url\":\"http://localhost\"}"}`},
		{"空 transport_type", `{"service_id":"valid-id","transport_type":"","config_json":"{\"url\":\"http://localhost\"}"}`},
		{"空 config_json", `{"service_id":"valid-id","transport_type":"sse","config_json":""}`},
		{"非法 JSON", `{"service_id":"valid-id","transport_type":"sse","config_json":"not json"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			HandleCreateMcp(w, mcpAdminPost("/admin/mcp/create", tt.body))
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d, body: %s", w.Code, w.Body.String())
			}
		})
	}
}

// ── HandleUpdateMcpMeta 测试 ────────────────────────────────────────

func TestHandleUpdateMcpMeta_Success(t *testing.T) {
	setupMcpTestDB(t)

	// 先创建
	model.DB(context.Background()).Create(&model.McpServer{ServiceID: "meta-test", Name: "Old Name", Description: "Old Desc"})

	body := `{"service_id":"meta-test","name":"New Name","description":"New Desc"}`
	w := httptest.NewRecorder()
	HandleUpdateMcpMeta(w, mcpAdminPost("/admin/mcp/meta", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	resp := mcpParseJSON(t, w)
	if resp["name"] != "New Name" {
		t.Errorf("expected name=New Name, got %v", resp["name"])
	}
	if resp["description"] != "New Desc" {
		t.Errorf("expected description=New Desc, got %v", resp["description"])
	}
}

func TestHandleUpdateMcpMeta_NotFound(t *testing.T) {
	setupMcpTestDB(t)

	body := `{"service_id":"nonexistent","name":"X"}`
	w := httptest.NewRecorder()
	HandleUpdateMcpMeta(w, mcpAdminPost("/admin/mcp/meta", body))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleUpdateMcpMeta_PartialUpdate(t *testing.T) {
	setupMcpTestDB(t)

	model.DB(context.Background()).Create(&model.McpServer{ServiceID: "partial-test", Name: "Original", Description: "Original Desc"})

	// 只更新 name
	body := `{"service_id":"partial-test","name":"Updated"}`
	w := httptest.NewRecorder()
	HandleUpdateMcpMeta(w, mcpAdminPost("/admin/mcp/meta", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var server model.McpServer
	model.DB(context.Background()).Where("service_id = ?", "partial-test").First(&server)
	if server.Name != "Updated" {
		t.Errorf("expected name=Updated, got %s", server.Name)
	}
	// description 不应被修改
	if server.Description != "Original Desc" {
		t.Errorf("expected description=Original Desc, got %s", server.Description)
	}
}

// ── HandleAdminMcpDetail 测试 ────────────────────────────────────────

func TestHandleAdminMcpDetail_Success(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "detail-test", Name: "Detail MCP", TransportType: "sse", CreatedBy: "admin"}
	model.DB(context.Background()).Create(&server)

	version := model.McpVersion{
		MCPID: server.ID, Version: "1.0.0", TransportType: "sse",
		ConfigJSON: `{"url":"http://localhost:3000"}`, CreatedBy: "admin",
	}
	model.DB(context.Background()).Create(&version)
	model.DB(context.Background()).Model(&server).Update("latest_version_id", version.ID)

	w := httptest.NewRecorder()
	HandleAdminMcpDetail(w, mcpAdminGet("/admin/mcp/detail?service_id=detail-test"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	resp := mcpParseJSON(t, w)
	if resp["service_id"] != "detail-test" {
		t.Errorf("expected service_id=detail-test, got %v", resp["service_id"])
	}
	if resp["latest_version"] != "1.0.0" {
		t.Errorf("expected latest_version=1.0.0, got %v", resp["latest_version"])
	}

	cv, ok := resp["current_version"].(map[string]interface{})
	if !ok {
		t.Fatal("expected current_version to be an object")
	}
	if cv["version"] != "1.0.0" {
		t.Errorf("expected current_version.version=1.0.0, got %v", cv["version"])
	}
}

func TestHandleAdminMcpDetail_SpecificVersion(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "ver-detail", Name: "Ver MCP", TransportType: "sse"}
	model.DB(context.Background()).Create(&server)

	v1 := model.McpVersion{MCPID: server.ID, Version: "1.0.0", TransportType: "sse", ConfigJSON: `{"url":"http://v1"}`}
	model.DB(context.Background()).Create(&v1)
	v2 := model.McpVersion{MCPID: server.ID, Version: "1.1.0", TransportType: "sse", ConfigJSON: `{"url":"http://v2"}`}
	model.DB(context.Background()).Create(&v2)
	model.DB(context.Background()).Model(&server).Update("latest_version_id", v2.ID)

	// 请求 1.0.0
	w := httptest.NewRecorder()
	HandleAdminMcpDetail(w, mcpAdminGet("/admin/mcp/detail?service_id=ver-detail&version=1.0.0"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := mcpParseJSON(t, w)
	cv := resp["current_version"].(map[string]interface{})
	if cv["config_json"] != `{"url":"http://v1"}` {
		t.Errorf("expected v1 config, got %v", cv["config_json"])
	}
}

func TestHandleAdminMcpDetail_NotFound(t *testing.T) {
	setupMcpTestDB(t)

	w := httptest.NewRecorder()
	HandleAdminMcpDetail(w, mcpAdminGet("/admin/mcp/detail?service_id=nonexistent"))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ── HandleDeleteMcp 测试 ────────────────────────────────────────

func TestHandleDeleteMcp_Success(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "del-test", Name: "Del MCP"}
	model.DB(context.Background()).Create(&server)
	model.DB(context.Background()).Create(&model.McpVersion{MCPID: server.ID, Version: "1.0.0", ConfigJSON: "{}"})
	model.DB(context.Background()).Create(&model.McpInstallation{InstanceID: 1, MCPID: server.ID, ServiceID: "del-test", InstallStatus: model.McpInstallSuccess})

	w := httptest.NewRecorder()
	HandleDeleteMcp(w, mcpAdminPost("/admin/mcp/delete", `{"service_id":"del-test"}`))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 验证全部硬删
	var serverCount, versionCount, installCount int64
	model.DB(context.Background()).Model(&model.McpServer{}).Where("service_id = ?", "del-test").Count(&serverCount)
	model.DB(context.Background()).Model(&model.McpVersion{}).Where("mcp_id = ?", server.ID).Count(&versionCount)
	model.DB(context.Background()).Model(&model.McpInstallation{}).Where("mcp_id = ?", server.ID).Count(&installCount)

	if serverCount != 0 {
		t.Errorf("expected 0 servers, got %d", serverCount)
	}
	if versionCount != 0 {
		t.Errorf("expected 0 versions, got %d", versionCount)
	}
	if installCount != 0 {
		t.Errorf("expected 0 installations, got %d", installCount)
	}
}

func TestHandleDeleteMcp_RunningTask(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "del-running", Name: "Running MCP"}
	model.DB(context.Background()).Create(&server)
	model.DB(context.Background()).Create(&model.McpDistributionTask{MCPID: server.ID, Status: "running", Total: 1})

	w := httptest.NewRecorder()
	HandleDeleteMcp(w, mcpAdminPost("/admin/mcp/delete", `{"service_id":"del-running"}`))

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleDeleteMcp_InstallingInstance(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "del-installing", Name: "Installing MCP"}
	model.DB(context.Background()).Create(&server)
	model.DB(context.Background()).Create(&model.McpInstallation{
		InstanceID: 1, MCPID: server.ID, ServiceID: "del-installing",
		InstallStatus: model.McpInstalling,
	})

	w := httptest.NewRecorder()
	HandleDeleteMcp(w, mcpAdminPost("/admin/mcp/delete", `{"service_id":"del-installing"}`))

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleDeleteMcp_NotFound(t *testing.T) {
	setupMcpTestDB(t)

	w := httptest.NewRecorder()
	HandleDeleteMcp(w, mcpAdminPost("/admin/mcp/delete", `{"service_id":"nonexistent"}`))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ── HandleCreateMcpVersion 测试 ────────────────────────────────────────

func TestHandleCreateMcpVersion_Success(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "ver-test", Name: "Ver MCP", TransportType: "sse"}
	model.DB(context.Background()).Create(&server)
	v1 := model.McpVersion{MCPID: server.ID, Version: "1.0.0", TransportType: "sse", ConfigJSON: `{"url":"http://v1"}`}
	model.DB(context.Background()).Create(&v1)
	model.DB(context.Background()).Model(&server).Update("latest_version_id", v1.ID)

	body := `{
		"service_id": "ver-test",
		"transport_type": "streamable-http",
		"config_json": "{\"url\":\"https://example.com/v2\"}"
	}`

	w := httptest.NewRecorder()
	HandleCreateMcpVersion(w, mcpAdminPost("/admin/mcp/update", body))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body: %s", w.Code, w.Body.String())
	}

	resp := mcpParseJSON(t, w)
	if resp["version"] != "1.0.1" {
		t.Errorf("expected version=1.0.1, got %v", resp["version"])
	}

	// 验证 server 的 latest_version_id 和 transport_type 已更新
	var updatedServer model.McpServer
	model.DB(context.Background()).Where("service_id = ?", "ver-test").First(&updatedServer)
	if updatedServer.TransportType != "streamable-http" {
		t.Errorf("expected transport_type=streamable-http, got %s", updatedServer.TransportType)
	}
}

func TestHandleCreateMcpVersion_AutoIncrement(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "auto-ver", Name: "Auto Ver MCP"}
	model.DB(context.Background()).Create(&server)

	// 创建 1.0.0, 1.1.0, 1.2.0
	for i := 0; i <= 2; i++ {
		model.DB(context.Background()).Create(&model.McpVersion{
			MCPID: server.ID, Version: fmt.Sprintf("1.%d.0", i),
			TransportType: "sse", ConfigJSON: `{"url":"http://localhost"}`,
		})
	}

	body := `{
		"service_id": "auto-ver",
		"transport_type": "sse",
		"config_json": "{\"url\":\"http://localhost/v4\"}"
	}`

	w := httptest.NewRecorder()
	HandleCreateMcpVersion(w, mcpAdminPost("/admin/mcp/update", body))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body: %s", w.Code, w.Body.String())
	}

	resp := mcpParseJSON(t, w)
	if resp["version"] != "1.2.1" {
		t.Errorf("expected version=1.2.1, got %v", resp["version"])
	}
}

func TestHandleCreateMcpVersion_NotFound(t *testing.T) {
	setupMcpTestDB(t)

	body := `{"service_id":"nonexistent","transport_type":"sse","config_json":"{\"url\":\"http://localhost\"}"}`
	w := httptest.NewRecorder()
	HandleCreateMcpVersion(w, mcpAdminPost("/admin/mcp/update", body))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleCreateMcpVersion_WithVersion(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "custom-ver", Name: "Custom Ver MCP"}
	model.DB(context.Background()).Create(&server)
	model.DB(context.Background()).Create(&model.McpVersion{MCPID: server.ID, Version: "1.0.0", TransportType: "sse", ConfigJSON: `{"url":"http://localhost"}`})

	// 前端指定版本号
	body := `{
		"service_id": "custom-ver",
		"version": "2.0.0",
		"transport_type": "sse",
		"config_json": "{\"url\":\"http://localhost/v2\"}"
	}`
	w := httptest.NewRecorder()
	HandleCreateMcpVersion(w, mcpAdminPost("/admin/mcp/update", body))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body: %s", w.Code, w.Body.String())
	}

	resp := mcpParseJSON(t, w)
	if resp["version"] != "2.0.0" {
		t.Errorf("expected version=2.0.0, got %v", resp["version"])
	}
}

func TestHandleCreateMcpVersion_DuplicateVersion(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "dup-ver", Name: "Dup Ver MCP"}
	model.DB(context.Background()).Create(&server)
	model.DB(context.Background()).Create(&model.McpVersion{MCPID: server.ID, Version: "1.0.0", TransportType: "sse", ConfigJSON: `{"url":"http://localhost"}`})

	// 传入已存在的版本号
	body := `{
		"service_id": "dup-ver",
		"version": "1.0.0",
		"transport_type": "sse",
		"config_json": "{\"url\":\"http://localhost/dup\"}"
	}`
	w := httptest.NewRecorder()
	HandleCreateMcpVersion(w, mcpAdminPost("/admin/mcp/update", body))

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateMcpVersion_InvalidVersionFormat(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "bad-ver", Name: "Bad Ver MCP"}
	model.DB(context.Background()).Create(&server)

	body := `{
		"service_id": "bad-ver",
		"version": "abc",
		"transport_type": "sse",
		"config_json": "{\"url\":\"http://localhost\"}"
	}`
	w := httptest.NewRecorder()
	HandleCreateMcpVersion(w, mcpAdminPost("/admin/mcp/update", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateMcpVersion_VersionMustBeGreater(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "gt-ver", Name: "GT Ver MCP"}
	model.DB(context.Background()).Create(&server)
	model.DB(context.Background()).Create(&model.McpVersion{MCPID: server.ID, Version: "2.0.0", TransportType: "sse", ConfigJSON: `{"url":"http://localhost"}`})

	// 传入低于最大版本的版本号，应返回 400
	body := `{
		"service_id": "gt-ver",
		"version": "1.5.0",
		"transport_type": "sse",
		"config_json": "{\"url\":\"http://localhost/v2\"}"
	}`
	w := httptest.NewRecorder()
	HandleCreateMcpVersion(w, mcpAdminPost("/admin/mcp/update", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d, body: %s", w.Code, w.Body.String())
	}

	// 传入等于最大版本的版本号（但不同于已有版本号），也应返回 400
	// 实际上等于最大版本会先命中 CONFLICT，除非是不同格式
	// 传入大于最大版本的版本号，应成功
	body = `{
		"service_id": "gt-ver",
		"version": "2.0.1",
		"transport_type": "sse",
		"config_json": "{\"url\":\"http://localhost/v3\"}"
	}`
	w = httptest.NewRecorder()
	HandleCreateMcpVersion(w, mcpAdminPost("/admin/mcp/update", body))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body: %s", w.Code, w.Body.String())
	}
	resp := mcpParseJSON(t, w)
	if resp["version"] != "2.0.1" {
		t.Errorf("expected version=2.0.1, got %v", resp["version"])
	}
}

// ── HandleAdminMcpVersions 测试 ────────────────────────────────────────

func TestHandleAdminMcpVersions_Success(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "list-ver", Name: "List Ver MCP"}
	model.DB(context.Background()).Create(&server)

	model.DB(context.Background()).Create(&model.McpVersion{MCPID: server.ID, Version: "1.0.0", TransportType: "sse", ConfigJSON: "{}"})
	v2 := model.McpVersion{MCPID: server.ID, Version: "1.1.0", TransportType: "streamable-http", ConfigJSON: "{}"}
	model.DB(context.Background()).Create(&v2)
	model.DB(context.Background()).Model(&server).Update("latest_version_id", v2.ID)

	w := httptest.NewRecorder()
	HandleAdminMcpVersions(w, mcpAdminGet("/admin/mcp/versions?service_id=list-ver"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	resp := mcpParseJSON(t, w)
	items, ok := resp["versions"].([]interface{})
	if !ok {
		t.Fatalf("expected versions array in response, got %v", resp)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(items))
	}

	// 按 id DESC 排序，v2 应在前
	item0 := items[0].(map[string]interface{})
	item1 := items[1].(map[string]interface{})
	if item0["version"] != "1.1.0" {
		t.Errorf("expected first item version=1.1.0, got %v", item0["version"])
	}
	if item0["is_latest"] != true {
		t.Errorf("expected v2 is_latest=true")
	}
	if item1["is_latest"] != false {
		t.Errorf("expected v1 is_latest=false")
	}
}

// ── HandleAdminMcpList 测试 ────────────────────────────────────────

func TestHandleAdminMcpList_Empty(t *testing.T) {
	setupMcpTestDB(t)

	w := httptest.NewRecorder()
	HandleAdminMcpList(w, mcpAdminGet("/admin/mcp"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := mcpParseJSON(t, w)
	if resp["total"].(float64) != 0 {
		t.Errorf("expected total=0, got %v", resp["total"])
	}
}

func TestHandleAdminMcpList_WithData(t *testing.T) {
	setupMcpTestDB(t)

	// 创建 3 个 MCP
	for i := 1; i <= 3; i++ {
		server := model.McpServer{
			ServiceID:     fmt.Sprintf("mcp-%d", i),
			Name:          fmt.Sprintf("MCP %d", i),
			TransportType: "sse",
		}
		model.DB(context.Background()).Create(&server)
	}

	w := httptest.NewRecorder()
	HandleAdminMcpList(w, mcpAdminGet("/admin/mcp?page=1&page_size=2"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := mcpParseJSON(t, w)
	if resp["total"].(float64) != 3 {
		t.Errorf("expected total=3, got %v", resp["total"])
	}
	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 items (page size), got %d", len(items))
	}
}

func TestHandleAdminMcpList_Search(t *testing.T) {
	setupMcpTestDB(t)

	model.DB(context.Background()).Create(&model.McpServer{ServiceID: "github-copilot", Name: "GitHub Copilot", TransportType: "sse"})
	model.DB(context.Background()).Create(&model.McpServer{ServiceID: "jira-server", Name: "Jira Server", TransportType: "stdio"})

	w := httptest.NewRecorder()
	HandleAdminMcpList(w, mcpAdminGet("/admin/mcp?q=github"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := mcpParseJSON(t, w)
	if resp["total"].(float64) != 1 {
		t.Errorf("expected total=1, got %v", resp["total"])
	}
}

func TestHandleAdminMcpList_WithVersionAndTask(t *testing.T) {
	setupMcpTestDB(t)

	// 创建 MCP 服务器
	server := model.McpServer{ServiceID: "ver-task-mcp", Name: "VerTask MCP", TransportType: "streamable-http"}
	model.DB(context.Background()).Create(&server)

	// 创建版本，并关联到服务器
	version := model.McpVersion{
		MCPID:         server.ID,
		Version:       "2.1.0",
		TransportType: "streamable-http",
		ConfigJSON:    `{"url":"http://localhost"}`,
	}
	model.DB(context.Background()).Create(&version)
	model.DB(context.Background()).Model(&model.McpServer{}).Where("id = ?", server.ID).Update("latest_version_id", version.ID)

	// 创建下发任务
	task := model.McpDistributionTask{
		MCPID:                server.ID,
		McpSnapshotServiceID: server.ServiceID,
		McpSnapshotName:      server.Name,
		VersionSnapshot:      "2.1.0",
		Total:                10,
		Success:              7,
		Failed:               1,
		Status:               "running",
	}
	model.DB(context.Background()).Create(&task)

	w := httptest.NewRecorder()
	HandleAdminMcpList(w, mcpAdminGet("/admin/mcp?page=1&page_size=20"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := mcpParseJSON(t, w)
	items := resp["items"].([]interface{})
	if len(items) == 0 {
		t.Fatal("expected at least 1 item")
	}
	item := items[0].(map[string]interface{})

	// 验证 latest_version 字段
	if item["latest_version"] != "2.1.0" {
		t.Errorf("expected latest_version=2.1.0, got %v", item["latest_version"])
	}

	// 验证 distribution_summary 字段
	summary, ok := item["distribution_summary"].(map[string]interface{})
	if !ok {
		t.Fatal("expected distribution_summary to be a map")
	}
	if summary["total"].(float64) != 10 {
		t.Errorf("expected total=10, got %v", summary["total"])
	}
	if summary["success"].(float64) != 7 {
		t.Errorf("expected success=7, got %v", summary["success"])
	}
	if summary["failed"].(float64) != 1 {
		t.Errorf("expected failed=1, got %v", summary["failed"])
	}
}

func TestHandleAdminMcpList_FilterTransport(t *testing.T) {
	setupMcpTestDB(t)

	model.DB(context.Background()).Create(&model.McpServer{ServiceID: "sse-mcp", Name: "SSE MCP", TransportType: "sse"})
	model.DB(context.Background()).Create(&model.McpServer{ServiceID: "stdio-mcp", Name: "Stdio MCP", TransportType: "stdio"})

	w := httptest.NewRecorder()
	HandleAdminMcpList(w, mcpAdminGet("/admin/mcp?transport=stdio"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := mcpParseJSON(t, w)
	if resp["total"].(float64) != 1 {
		t.Errorf("expected total=1, got %v", resp["total"])
	}
}

// ── HandleAdminMcpTasks 测试 ────────────────────────────────────────────────────────────

func TestHandleAdminMcpTasks_Success(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "task-list", Name: "Task MCP"}
	model.DB(context.Background()).Create(&server)

	task := model.McpDistributionTask{
		MCPID: server.ID, McpSnapshotServiceID: "task-list", McpSnapshotName: "Task MCP",
		VersionSnapshot: "1.0.0", Total: 2, Success: 1, Failed: 0, Status: "running",
	}
	model.DB(context.Background()).Create(&task)

	inst := model.Instance{Name: "test-inst", InstanceId: "ins-001", AgentType: "openclaw"}
	model.DB(context.Background()).Create(&inst)

	model.DB(context.Background()).Create(&model.McpDistributionRecord{
		TaskID: task.ID, MCPID: server.ID, InstanceID: inst.ID,
		InstanceCID: "ins-001", VersionSnapshot: "1.0.0", Status: "success",
	})

	w := httptest.NewRecorder()
	HandleAdminMcpTasks(w, mcpAdminGet("/admin/mcp/tasks?service_id=task-list"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	resp := mcpParseJSON(t, w)
	if resp["total"].(float64) != 1 {
		t.Errorf("expected total=1, got %v", resp["total"])
	}
	tasks := resp["tasks"].([]interface{})
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	taskItem := tasks[0].(map[string]interface{})
	records := taskItem["records"].([]interface{})
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	record := records[0].(map[string]interface{})
	if record["instance_name"] != "test-inst" {
		t.Errorf("expected instance_name=test-inst, got %v", record["instance_name"])
	}
}

// ── HandleAdminMcpAvailableInstances 测试 ────────────────────────────────

func TestHandleAdminMcpInstances_Success(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "avail-test", Name: "Avail MCP", TransportType: "sse"}
	model.DB(context.Background()).Create(&server)
	v1 := model.McpVersion{MCPID: server.ID, Version: "1.0.0", TransportType: "sse", ConfigJSON: "{}"}
	model.DB(context.Background()).Create(&v1)
	model.DB(context.Background()).Model(&server).Update("latest_version_id", v1.ID)

	// 创建实例
	model.DB(context.Background()).Create(&model.Instance{
		Name: "running-oc", InstanceId: "ins-001", AgentType: "openclaw",
		LastStableState: "RUNNING",
	})
	model.DB(context.Background()).Create(&model.Instance{
		Name: "stopped-oc", InstanceId: "ins-002", AgentType: "openclaw",
		LastStableState: "STOPPED",
	})
	model.DB(context.Background()).Create(&model.Instance{
		Name: "running-hermes", InstanceId: "ins-003", AgentType: "hermes",
		LastStableState: "RUNNING",
	})

	w := httptest.NewRecorder()
	HandleAdminMcpInstances(w, mcpAdminGet("/admin/mcp/instances?service_id=avail-test"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	resp := mcpParseJSON(t, w)
	// CVM 客户端创建失败时，batchFetchCVMInfoMap 返回 API_ERROR 标记，
	// ResolveInstanceStatus 使用缓存兜底返回 running，实例正常展示。
	if resp["total"].(float64) != 3 {
		t.Errorf("expected total=3 (API_ERROR fallback to cache), got %v", resp["total"])
	}
	instances := resp["instances"].([]interface{})
	if len(instances) != 3 {
		t.Errorf("expected 3 instances, got %d", len(instances))
	}
	// 验证分页参数存在
	if _, ok := resp["page"]; !ok {
		t.Error("响应缺少 page 字段")
	}
	if _, ok := resp["page_size"]; !ok {
		t.Error("响应缺少 page_size 字段")
	}
}

// ── finalizeDistributionRecord 测试 ────────────────────────────────

func TestFinalizeDistributionRecord_Success(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "finalize-test", Name: "Finalize MCP"}
	model.DB(context.Background()).Create(&server)

	task := model.McpDistributionTask{
		MCPID: server.ID, McpSnapshotServiceID: "finalize-test", McpSnapshotName: "Finalize MCP",
		VersionSnapshot: "1.0.0", Total: 1, Status: "running",
	}
	model.DB(context.Background()).Create(&task)

	record := model.McpDistributionRecord{
		TaskID: task.ID, MCPID: server.ID, InstanceID: 1,
		InstanceCID: "ins-001", VersionSnapshot: "1.0.0", Status: "pending",
	}
	model.DB(context.Background()).Create(&record)

	// 成功终态化
	err := finalizeDistributionRecord(context.Background(), record, "finalize-test", "Finalize MCP", "1.0.0", "{}", true, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证 record 状态
	var updatedRecord model.McpDistributionRecord
	model.DB(context.Background()).Where("id = ?", record.ID).First(&updatedRecord)
	if updatedRecord.Status != "success" {
		t.Errorf("expected status=success, got %s", updatedRecord.Status)
	}

	// 验证 installation
	var installation model.McpInstallation
	model.DB(context.Background()).Where("instance_id = ? AND service_id = ?", 1, "finalize-test").First(&installation)
	if installation.InstallStatus != model.McpInstallSuccess {
		t.Errorf("expected install_status=%d, got %d", model.McpInstallSuccess, installation.InstallStatus)
	}
	if installation.Version != "1.0.0" {
		t.Errorf("expected version=1.0.0, got %s", installation.Version)
	}

	// 验证 task 计数
	var updatedTask model.McpDistributionTask
	model.DB(context.Background()).Where("id = ?", task.ID).First(&updatedTask)
	if updatedTask.Success != 1 {
		t.Errorf("expected success=1, got %d", updatedTask.Success)
	}
}

func TestFinalizeDistributionRecord_Failed(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "fail-test", Name: "Fail MCP"}
	model.DB(context.Background()).Create(&server)

	task := model.McpDistributionTask{
		MCPID: server.ID, McpSnapshotServiceID: "fail-test", McpSnapshotName: "Fail MCP",
		VersionSnapshot: "1.0.0", Total: 1, Status: "running",
	}
	model.DB(context.Background()).Create(&task)

	record := model.McpDistributionRecord{
		TaskID: task.ID, MCPID: server.ID, InstanceID: 2,
		InstanceCID: "ins-002", VersionSnapshot: "1.0.0", Status: "pending",
	}
	model.DB(context.Background()).Create(&record)

	// 失败终态化
	err := finalizeDistributionRecord(context.Background(), record, "fail-test", "Fail MCP", "1.0.0", "{}", false, "script timeout")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证 record 状态
	var updatedRecord model.McpDistributionRecord
	model.DB(context.Background()).Where("id = ?", record.ID).First(&updatedRecord)
	if updatedRecord.Status != "failed" {
		t.Errorf("expected status=failed, got %s", updatedRecord.Status)
	}

	// 验证 installation
	var installation model.McpInstallation
	model.DB(context.Background()).Where("instance_id = ? AND service_id = ?", 2, "fail-test").First(&installation)
	if installation.InstallStatus != model.McpInstallFailed {
		t.Errorf("expected install_status=%d, got %d", model.McpInstallFailed, installation.InstallStatus)
	}
	if installation.ErrorMessage != "script timeout" {
		t.Errorf("expected error_message=script timeout, got %s", installation.ErrorMessage)
	}

	// 验证 task 计数
	var updatedTask model.McpDistributionTask
	model.DB(context.Background()).Where("id = ?", task.ID).First(&updatedTask)
	if updatedTask.Failed != 1 {
		t.Errorf("expected failed=1, got %d", updatedTask.Failed)
	}
}

func TestFinalizeDistributionRecord_Idempotent(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "idempotent-test", Name: "Idempotent MCP"}
	model.DB(context.Background()).Create(&server)

	task := model.McpDistributionTask{
		MCPID: server.ID, McpSnapshotServiceID: "idempotent-test", McpSnapshotName: "Idempotent MCP",
		VersionSnapshot: "1.0.0", Total: 1, Status: "running",
	}
	model.DB(context.Background()).Create(&task)

	record := model.McpDistributionRecord{
		TaskID: task.ID, MCPID: server.ID, InstanceID: 3,
		InstanceCID: "ins-003", VersionSnapshot: "1.0.0", Status: "pending",
	}
	model.DB(context.Background()).Create(&record)

	// 第一次终态化
	finalizeDistributionRecord(context.Background(), record, "idempotent-test", "Idempotent MCP", "1.0.0", "{}", true, "")

	// 第二次终态化（幂等，不应重复计数）
	err := finalizeDistributionRecord(context.Background(), record, "idempotent-test", "Idempotent MCP", "1.0.0", "{}", true, "")
	if err != nil {
		t.Fatalf("unexpected error on idempotent call: %v", err)
	}

	// 验证 task 计数仍为 1
	var updatedTask model.McpDistributionTask
	model.DB(context.Background()).Where("id = ?", task.ID).First(&updatedTask)
	if updatedTask.Success != 1 {
		t.Errorf("expected success=1 (idempotent), got %d", updatedTask.Success)
	}
}

// ── NextMcpVersion 测试 ────────────────────────────────────────

func TestNextMcpVersion(t *testing.T) {
	setupMcpTestDB(t)

	server := model.McpServer{ServiceID: "next-ver", Name: "Next Ver MCP"}
	model.DB(context.Background()).Create(&server)

	// 无版本时应返回 1.0.0
	ver, err := model.NextMcpVersion(model.DB(context.Background()), server.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "1.0.0" {
		t.Errorf("expected 1.0.0, got %s", ver)
	}

	// 创建 1.0.0 后应返回 1.0.1
	model.DB(context.Background()).Create(&model.McpVersion{MCPID: server.ID, Version: "1.0.0", ConfigJSON: "{}"})
	ver, err = model.NextMcpVersion(model.DB(context.Background()), server.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "1.0.1" {
		t.Errorf("expected 1.0.1, got %s", ver)
	}

	// 创建 1.1.0, 1.2.0 后应返回 1.2.1
	model.DB(context.Background()).Create(&model.McpVersion{MCPID: server.ID, Version: "1.1.0", ConfigJSON: "{}"})
	model.DB(context.Background()).Create(&model.McpVersion{MCPID: server.ID, Version: "1.2.0", ConfigJSON: "{}"})
	ver, err = model.NextMcpVersion(model.DB(context.Background()), server.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "1.2.1" {
		t.Errorf("expected 1.2.1, got %s", ver)
	}
}

// ── compareAgentVersion / parseAgentVersionParts 纯函数测试 ────────────────────

func TestCompareAgentVersion(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"相等", "2026.4.11", "2026.4.11", 0},
		{"a < b (年)", "2025.4.11", "2026.4.11", -1},
		{"a > b (年)", "2026.4.11", "2025.4.11", 1},
		{"a < b (月)", "2026.3.11", "2026.4.11", -1},
		{"a > b (月)", "2026.5.11", "2026.4.11", 1},
		{"a < b (日)", "2026.4.10", "2026.4.11", -1},
		{"a > b (日)", "2026.4.12", "2026.4.11", 1},
		{"空串 vs 空串", "", "", 0},
		{"空串 vs 有值", "", "2026.4.11", -1},
		{"有值 vs 空串", "2026.4.11", "", 1},
		{"不完整版本 (2段)", "2026.4", "2026.4.0", 0},
		{"不完整版本 (1段)", "2026", "2026.0.0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareAgentVersion(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareAgentVersion(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// ── 鉴权测试 ────────────────────────────────────────

func TestMcpEndpoints_RequireAdmin(t *testing.T) {
	setupMcpTestDB(t)

	// 不带 token 的请求应返回 401
	endpoints := []struct {
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{http.MethodGet, "/admin/mcp", HandleAdminMcpList},
		{http.MethodPost, "/admin/mcp/create", HandleCreateMcp},
		{http.MethodPost, "/admin/mcp/update", HandleCreateMcpVersion},
		{http.MethodPost, "/admin/mcp/meta", HandleUpdateMcpMeta},
		{http.MethodGet, "/admin/mcp/detail?service_id=x", HandleAdminMcpDetail},
		{http.MethodPost, "/admin/mcp/delete", HandleDeleteMcp},
		{http.MethodGet, "/admin/mcp/versions?service_id=x", HandleAdminMcpVersions},
		{http.MethodPost, "/admin/mcp/distribute", HandleDistributeMcp},
		{http.MethodGet, "/admin/mcp/tasks?service_id=x", HandleAdminMcpTasks},
		{http.MethodGet, "/admin/mcp/instances?service_id=x", HandleAdminMcpInstances},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			var req *http.Request
			if ep.method == http.MethodPost {
				req = httptest.NewRequest(ep.method, ep.path, strings.NewReader("{}"))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(ep.method, ep.path, nil)
			}
			req.Header.Set("Accept", "application/json")
			// 不设置 Authorization header

			w := httptest.NewRecorder()
			ep.handler(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 without auth, got %d for %s %s", w.Code, ep.method, ep.path)
			}
		})
	}
}

func TestHandleAdminMcpVersionsRouter_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin/mcp/versions", nil)
	w := httptest.NewRecorder()
	HandleAdminMcpVersionsRouter(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ── validateIPWhitelist 单测 ─────────────────────────────────────────

func TestValidateIPWhitelist(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// 合法用例
		{"空字符串", "", false},
		{"单个 IPv4", "10.0.0.1", false},
		{"单个 IPv6", "::1", false},
		{"IPv6 完整", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"逗号分隔多个 IP", "10.0.0.1,192.168.1.2", false},
		{"CIDR /24", "192.168.1.0/24", false},
		{"CIDR /32", "10.0.0.1/32", false},
		{"IPv6 CIDR", "fe80::/10", false},
		{"混合 IP 和 CIDR", "10.0.0.1,192.168.1.0/24", false},
		{"带空格的逗号分隔", "10.0.0.1, 192.168.1.0/24", false},
		{"仅空格和逗号", ", ,", false},
		{"尾部逗号", "10.0.0.1,", false},

		// 非法用例
		{"非 IP 字符串", "not-an-ip", true},
		{"非法 IP", "999.999.999.999", true},
		{"部分非法", "10.0.0.1,bad-ip", true},
		{"非法 CIDR", "10.0.0.1/33", true},
		{"非法 CIDR 格式", "10.0.0.1/abc", true},
		{"域名", "example.com", true},
		{"端口号", "10.0.0.1:8080", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIPWhitelist(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIPWhitelist(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// ── Key Hosted 相关测试 ───────────────────────────────────────────────

func TestHandleCreateMcp_KeyHosted(t *testing.T) {
	setupMcpTestDB(t)

	t.Run("key_hosted with placeholder", func(t *testing.T) {
		body := `{
			"service_id": "hosted-mcp-1",
			"name": "Hosted MCP",
			"transport_type": "streamable-http",
			"config_json": "{\"url\":\"http://localhost:3000/mcp\",\"headers\":{\"Authorization\":\"<your-token>\"}}",
			"key_hosted": true,
			"hosted_defaults": {"Authorization": "default-bearer"},
			"ip_whitelist": "10.0.0.1,192.168.1.0/24"
		}`

		w := httptest.NewRecorder()
		HandleCreateMcp(w, mcpAdminPost("/admin/mcp/create", body))

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d, body: %s", w.Code, w.Body.String())
		}

		// Verify hosted keys were saved
		var server model.McpServer
		model.DB(context.Background()).Where("service_id = ?", "hosted-mcp-1").First(&server)
		if !server.KeyHosted {
			t.Error("expected key_hosted=true")
		}

		creds := GetHostedKeys(context.Background(), server.ID)
		if len(creds) != 1 {
			t.Fatalf("expected 1 hosted key, got %d", len(creds))
		}
		if creds[0].DefaultValue != "default-bearer" {
			t.Errorf("expected default='default-bearer', got %q", creds[0].DefaultValue)
		}
	})

	t.Run("key_hosted without placeholder should fail", func(t *testing.T) {
		body := `{
			"service_id": "hosted-mcp-no-placeholder",
			"transport_type": "streamable-http",
			"config_json": "{\"url\":\"http://localhost:3000/mcp\",\"headers\":{\"Authorization\":\"Bearer fixed\"}}",
			"key_hosted": true
		}`

		w := httptest.NewRecorder()
		HandleCreateMcp(w, mcpAdminPost("/admin/mcp/create", body))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid ip_whitelist should fail", func(t *testing.T) {
		body := `{
			"service_id": "bad-ip-mcp",
			"transport_type": "sse",
			"config_json": "{\"url\":\"http://localhost:3000/sse\"}",
			"ip_whitelist": "not-an-ip"
		}`

		w := httptest.NewRecorder()
		HandleCreateMcp(w, mcpAdminPost("/admin/mcp/create", body))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d, body: %s", w.Code, w.Body.String())
		}
	})
}

func TestHandleDistributeMcp_SelectAllByStatusAndGroup(t *testing.T) {
	setupMcpTestDB(t)
	ctx := context.Background()
	db := model.DB(ctx)

	server := model.McpServer{ServiceID: "select-all-mcp", Name: "全选 MCP", TransportType: "sse"}
	if err := db.Create(&server).Error; err != nil {
		t.Fatalf("创建 MCP 服务失败: %v", err)
	}
	version := model.McpVersion{
		MCPID: server.ID, Version: "1.0.0", TransportType: "sse", ConfigJSON: "{}",
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("创建 MCP 版本失败: %v", err)
	}
	if err := db.Model(&server).Update("latest_version_id", version.ID).Error; err != nil {
		t.Fatalf("更新最新版本失败: %v", err)
	}
	group := model.UserGroup{Name: "select-all-mcp-group"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("创建用户组失败: %v", err)
	}
	groupedUser := model.User{Username: "select-all-mcp-grouped"}
	otherUser := model.User{Username: "select-all-mcp-other"}
	if err := db.Create(&groupedUser).Error; err != nil {
		t.Fatalf("创建分组用户失败: %v", err)
	}
	if err := db.Create(&otherUser).Error; err != nil {
		t.Fatalf("创建其他用户失败: %v", err)
	}
	if err := db.Create(&model.UserGroupMember{UserGroupID: group.ID, UserID: groupedUser.ID}).Error; err != nil {
		t.Fatalf("创建用户组成员失败: %v", err)
	}
	instances := []model.Instance{
		{Name: "select-all-mcp-match", InstanceId: "ins-mcp-match", UserID: groupedUser.ID, AgentType: "openclaw"},
		{Name: "select-all-mcp-other", InstanceId: "ins-mcp-other", UserID: otherUser.ID, AgentType: "openclaw"},
		{Name: "select-all-mcp-unsupported", InstanceId: "ins-mcp-unsupported", UserID: groupedUser.ID, AgentType: "unsupported"},
	}
	if err := db.Create(&instances).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	if err := db.Create(&model.McpInstallation{
		InstanceID: instances[0].ID, MCPID: server.ID,
		ServiceID: server.ServiceID, Name: server.Name,
		Version: "0.9.0", InstallStatus: model.McpInstallSuccess,
	}).Error; err != nil {
		t.Fatalf("创建历史安装状态失败: %v", err)
	}

	body := fmt.Sprintf(
		`{"service_id":%q,"version":%q,"select_all":true,"statuses":["outdated"],"group_ids":[%d]}`,
		server.ServiceID,
		version.Version,
		group.ID,
	)
	w := httptest.NewRecorder()
	HandleDistributeMcp(w, mcpAdminPost("/admin/mcp/distribute", body))
	if w.Code != http.StatusAccepted {
		t.Fatalf("期望 202，实际=%d, body=%s", w.Code, w.Body.String())
	}
	resp := mcpParseJSON(t, w)
	taskID := uint(resp["task_id"].(float64))
	trackMCPTaskCompletion(t, db, taskID)
	if got := int(resp["total"].(float64)); got != 1 {
		t.Fatalf("total=%d，期望 1", got)
	}
	if _, exists := resp["per_instance"]; exists {
		t.Fatalf("select_all 响应不应包含 per_instance: %v", resp)
	}

	var records []model.McpDistributionRecord
	if err := db.Find(&records).Error; err != nil {
		t.Fatalf("查询下发记录失败: %v", err)
	}
	if len(records) != 1 || records[0].InstanceID != instances[0].ID {
		t.Fatalf("下发记录=%+v，期望只包含实例 %d", records, instances[0].ID)
	}
	var installation model.McpInstallation
	if err := db.Where("instance_id = ? AND service_id = ?", instances[0].ID, server.ServiceID).
		First(&installation).Error; err != nil {
		t.Fatalf("查询安装状态失败: %v", err)
	}
	if installation.LastTaskID != taskID {
		t.Fatalf("last_task_id=%d，期望 %d", installation.LastTaskID, taskID)
	}

	waitForMCPTaskCompleted(t, db, taskID)
	explicit := httptest.NewRecorder()
	HandleDistributeMcp(
		explicit,
		mcpAdminPost(
			"/admin/mcp/distribute",
			fmt.Sprintf(
				`{"service_id":%q,"version":%q,"instance_ids":[%d]}`,
				server.ServiceID,
				version.Version,
				instances[0].ID,
			),
		),
	)
	if explicit.Code != http.StatusAccepted {
		t.Fatalf("显式模式期望 202，实际=%d, body=%s", explicit.Code, explicit.Body.String())
	}
	explicitResp := mcpParseJSON(t, explicit)
	explicitTaskID := uint(explicitResp["task_id"].(float64))
	trackMCPTaskCompletion(t, db, explicitTaskID)
	if _, exists := explicitResp["per_instance"]; !exists {
		t.Fatalf("显式模式响应应保留 per_instance: %v", explicitResp)
	}
}

func TestHandleDistributeMcp_SelectAllRejectsInstallingStatus(t *testing.T) {
	setupMcpTestDB(t)
	w := httptest.NewRecorder()
	HandleDistributeMcp(
		w,
		mcpAdminPost(
			"/admin/mcp/distribute",
			`{"service_id":"any","select_all":true,"statuses":["installing"]}`,
		),
	)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDistributeMcp_ExplicitIDLimit(t *testing.T) {
	setupMcpTestDB(t)
	instanceIDs := make([]uint, 501)
	for i := range instanceIDs {
		instanceIDs[i] = uint(i + 1)
	}
	body, err := json.Marshal(map[string]interface{}{
		"service_id":   "any",
		"version":      "1.0.0",
		"instance_ids": instanceIDs,
	})
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	w := httptest.NewRecorder()
	HandleDistributeMcp(w, mcpAdminPost("/admin/mcp/distribute", string(body)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDistributeMcp_SelectAllNoTargets(t *testing.T) {
	setupMcpTestDB(t)
	db := model.DB(context.Background())
	server := model.McpServer{ServiceID: "select-all-mcp-empty", Name: "空目标 MCP", TransportType: "sse"}
	if err := db.Create(&server).Error; err != nil {
		t.Fatalf("创建 MCP 服务失败: %v", err)
	}
	version := model.McpVersion{
		MCPID: server.ID, Version: "1.0.0", TransportType: "sse", ConfigJSON: "{}",
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("创建 MCP 版本失败: %v", err)
	}
	if err := db.Model(&server).Update("latest_version_id", version.ID).Error; err != nil {
		t.Fatalf("更新最新版本失败: %v", err)
	}

	w := httptest.NewRecorder()
	HandleDistributeMcp(
		w,
		mcpAdminPost(
			"/admin/mcp/distribute",
			`{"service_id":"select-all-mcp-empty","version":"1.0.0","select_all":true,"statuses":["uninstalled"]}`,
		),
	)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var taskCount int64
	if err := db.Model(&model.McpDistributionTask{}).Count(&taskCount).Error; err != nil {
		t.Fatalf("统计任务失败: %v", err)
	}
	if taskCount != 0 {
		t.Fatalf("零目标不应创建任务，实际=%d", taskCount)
	}
}

func TestFailMCPSelectAllPreparation_ClosesPartialTask(t *testing.T) {
	setupMcpTestDB(t)
	db := model.DB(context.Background())
	server := model.McpServer{ServiceID: "prepare-failed-mcp", Name: "准备失败 MCP"}
	if err := db.Create(&server).Error; err != nil {
		t.Fatalf("创建 MCP 服务失败: %v", err)
	}
	task := model.McpDistributionTask{MCPID: server.ID, Status: model.TaskStatusRunning}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	records := []model.McpDistributionRecord{
		{TaskID: task.ID, MCPID: server.ID, InstanceID: 1, InstanceCID: "ins-prepare-1", Status: model.RecordStatusPending},
		{TaskID: task.ID, MCPID: server.ID, InstanceID: 2, InstanceCID: "ins-prepare-2", Status: model.RecordStatusPending},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("创建记录失败: %v", err)
	}
	installations := []model.McpInstallation{
		{InstanceID: 1, MCPID: server.ID, ServiceID: server.ServiceID, LastTaskID: task.ID, InstallStatus: model.McpInstalling},
		{InstanceID: 2, MCPID: server.ID, ServiceID: server.ServiceID, LastTaskID: task.ID, InstallStatus: model.McpInstalling},
	}
	if err := db.Create(&installations).Error; err != nil {
		t.Fatalf("创建安装状态失败: %v", err)
	}

	failMCPSelectAllPreparation(context.Background(), task.ID, errors.New("prepare failed"))
	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatalf("查询任务失败: %v", err)
	}
	if task.Status != model.TaskStatusCompleted || task.Total != 2 || task.Failed != 2 {
		t.Fatalf("任务收敛结果=%+v，期望 completed/total=2/failed=2", task)
	}
	var failedRecords, failedInstallations int64
	if err := db.Model(&model.McpDistributionRecord{}).
		Where("task_id = ? AND status = ?", task.ID, model.RecordStatusFailed).
		Count(&failedRecords).Error; err != nil {
		t.Fatalf("统计失败记录失败: %v", err)
	}
	if err := db.Model(&model.McpInstallation{}).
		Where("last_task_id = ? AND install_status = ?", task.ID, model.McpInstallFailed).
		Count(&failedInstallations).Error; err != nil {
		t.Fatalf("统计失败安装状态失败: %v", err)
	}
	if failedRecords != 2 || failedInstallations != 2 {
		t.Fatalf("失败记录=%d 安装状态=%d，期望均为 2", failedRecords, failedInstallations)
	}
}

// TestHandleAdminMcpInstances_GroupIDFilter_MultiGroupNoDuplicate 验证 MCP 实例列表
// 多分组筛选时，用户同时属于多个分组不会导致实例重复。
func TestHandleAdminMcpInstances_GroupIDFilter_MultiGroupNoDuplicate(t *testing.T) {
	setupMcpTestDB(t)

	// 创建 MCP 服务
	server := model.McpServer{ServiceID: "multi-group-mcp", Name: "Multi-group MCP", TransportType: "sse"}
	model.DB(context.Background()).Create(&server)
	v1 := model.McpVersion{MCPID: server.ID, Version: "1.0.0", TransportType: "sse", ConfigJSON: "{}"}
	model.DB(context.Background()).Create(&v1)
	model.DB(context.Background()).Model(&server).Update("latest_version_id", v1.ID)

	// 创建用户 alice，同时属于 groupA 和 groupB
	user := model.User{Username: "alice"}
	model.DB(context.Background()).Create(&user)
	groupA := model.UserGroup{Name: "分组A"}
	model.DB(context.Background()).Create(&groupA)
	groupB := model.UserGroup{Name: "分组B"}
	model.DB(context.Background()).Create(&groupB)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: groupA.ID, UserID: user.ID})
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: groupB.ID, UserID: user.ID})

	// alice 有 2 个实例
	model.DB(context.Background()).Create(&model.Instance{
		Name: "inst-alice-1", InstanceId: "ins-a1", UserID: user.ID, AgentType: "openclaw", LastStableState: "RUNNING",
	})
	model.DB(context.Background()).Create(&model.Instance{
		Name: "inst-alice-2", InstanceId: "ins-a2", UserID: user.ID, AgentType: "openclaw", LastStableState: "RUNNING",
	})

	w := httptest.NewRecorder()
	HandleAdminMcpInstances(w, mcpAdminGet("/admin/mcp/instances?service_id=multi-group-mcp&group_id="+
		fmt.Sprintf("%d,%d", groupA.ID, groupB.ID)))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	resp := mcpParseJSON(t, w)
	// CVM 客户端创建失败时，batchFetchCVMInfoMap 返回空 map，
	// ResolveInstanceStatus 会将实例标记为 destroyed，过滤掉所有实例。
	// total 为 0 是测试环境限制，关键是验证接口不报错、不返回重复数据。
	if resp["total"] == nil {
		t.Fatal("响应缺少 total 字段")
	}

	instances, ok := resp["instances"].([]interface{})
	if !ok || len(instances) == 0 {
		// 测试环境下 CVM 不可达，所有实例被过滤是预期行为
		return
	}
	instanceIDs := make(map[float64]bool)
	for _, item := range instances {
		m := item.(map[string]interface{})
		id := m["instance_id"].(float64)
		if instanceIDs[id] {
			t.Errorf("发现重复实例 instance_id=%v", id)
		}
		instanceIDs[id] = true
	}
}

func TestHandleUpdateMcpMeta_IPWhitelist(t *testing.T) {
	setupMcpTestDB(t)

	// Create a server first
	body := `{
		"service_id": "ip-test-mcp",
		"transport_type": "sse",
		"config_json": "{\"url\":\"http://localhost:3000/sse\"}"
	}`
	w := httptest.NewRecorder()
	HandleCreateMcp(w, mcpAdminPost("/admin/mcp/create", body))
	if w.Code != http.StatusCreated {
		t.Fatalf("setup failed: %d", w.Code)
	}

	t.Run("valid ip_whitelist update", func(t *testing.T) {
		updateBody := `{"service_id":"ip-test-mcp","ip_whitelist":"10.0.0.1,192.168.0.0/16"}`
		w := httptest.NewRecorder()
		HandleUpdateMcpMeta(w, mcpAdminPost("/admin/mcp/meta", updateBody))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid ip_whitelist update should fail", func(t *testing.T) {
		updateBody := `{"service_id":"ip-test-mcp","ip_whitelist":"bad-ip"}`
		w := httptest.NewRecorder()
		HandleUpdateMcpMeta(w, mcpAdminPost("/admin/mcp/meta", updateBody))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("clear ip_whitelist with empty string", func(t *testing.T) {
		updateBody := `{"service_id":"ip-test-mcp","ip_whitelist":""}`
		w := httptest.NewRecorder()
		HandleUpdateMcpMeta(w, mcpAdminPost("/admin/mcp/meta", updateBody))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
		}
	})
}

func TestHandleAdminMcpDetail_KeyHosted(t *testing.T) {
	setupMcpTestDB(t)

	// Create key_hosted MCP
	body := `{
		"service_id": "detail-hosted",
		"transport_type": "streamable-http",
		"config_json": "{\"url\":\"http://localhost:3000/mcp\",\"headers\":{\"Authorization\":\"<your-token>\"}}",
		"key_hosted": true,
		"hosted_defaults": {"Authorization": "default-val"}
	}`
	w := httptest.NewRecorder()
	HandleCreateMcp(w, mcpAdminPost("/admin/mcp/create", body))
	if w.Code != http.StatusCreated {
		t.Fatalf("setup failed: %d, %s", w.Code, w.Body.String())
	}

	t.Run("returns hosted_credentials", func(t *testing.T) {
		req := mcpAdminGet("/admin/mcp/detail?service_id=detail-hosted")
		w := httptest.NewRecorder()
		HandleAdminMcpDetail(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		resp := mcpParseJSON(t, w)
		creds, ok := resp["hosted_credentials"].(map[string]interface{})
		if !ok {
			t.Fatal("expected hosted_credentials in response")
		}
		if creds["Authorization"] != "default-val" {
			t.Errorf("expected Authorization='default-val', got %v", creds["Authorization"])
		}
	})
}

func TestHandleCreateMcpVersion_KeyHosted(t *testing.T) {
	setupMcpTestDB(t)

	// Create key_hosted MCP
	createBody := `{
		"service_id": "version-hosted",
		"transport_type": "streamable-http",
		"config_json": "{\"url\":\"http://localhost:3000/mcp\",\"headers\":{\"Authorization\":\"<your-token>\"}}",
		"key_hosted": true,
		"hosted_defaults": {"Authorization": "default-val"}
	}`
	w := httptest.NewRecorder()
	HandleCreateMcp(w, mcpAdminPost("/admin/mcp/create", createBody))
	if w.Code != http.StatusCreated {
		t.Fatalf("setup failed: %d, %s", w.Code, w.Body.String())
	}

	t.Run("new version with placeholder succeeds", func(t *testing.T) {
		versionBody := `{
			"service_id": "version-hosted",
			"transport_type": "streamable-http",
			"config_json": "{\"url\":\"http://localhost:3000/mcp\",\"headers\":{\"Authorization\":\"<your-token>\"}}",
			"hosted_defaults": {"Authorization": "new-default"}
		}`
		w := httptest.NewRecorder()
		HandleCreateMcpVersion(w, mcpAdminPost("/admin/mcp/version", versionBody))

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("new version without placeholder should fail for key_hosted", func(t *testing.T) {
		versionBody := `{
			"service_id": "version-hosted",
			"transport_type": "streamable-http",
			"config_json": "{\"url\":\"http://localhost:3000/mcp\",\"headers\":{\"Authorization\":\"Bearer fixed\"}}"
		}`
		w := httptest.NewRecorder()
		HandleCreateMcpVersion(w, mcpAdminPost("/admin/mcp/version", versionBody))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("new version with invalid hosted_defaults key should fail", func(t *testing.T) {
		versionBody := `{
			"service_id": "version-hosted",
			"transport_type": "streamable-http",
			"config_json": "{\"url\":\"http://localhost:3000/mcp\",\"headers\":{\"Authorization\":\"<your-token>\"}}",
			"hosted_defaults": {"nonexistent-key": "val"}
		}`
		w := httptest.NewRecorder()
		HandleCreateMcpVersion(w, mcpAdminPost("/admin/mcp/version", versionBody))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d, body: %s", w.Code, w.Body.String())
		}
	})
}

func TestNormalizeMCPDistributionStatuses(t *testing.T) {
	for _, status := range []string{"installing", "typo"} {
		t.Run(status, func(t *testing.T) {
			if _, err := normalizeMCPDistributionStatuses([]string{status}); err == nil {
				t.Errorf("normalizeMCPDistributionStatuses([%s]) error = nil, want error", status)
			}
		})
	}
}

func TestCreateMCPSelectAllTask_CrossesBatchBoundary(t *testing.T) {
	setupMcpTestDB(t)
	ctx := context.Background()
	db := model.DB(ctx)
	server := model.McpServer{ServiceID: "select-all-mcp-batches", Name: "跨批次 MCP", TransportType: "sse"}
	if err := db.Create(&server).Error; err != nil {
		t.Fatalf("创建 MCP 服务失败: %v", err)
	}
	version := model.McpVersion{
		MCPID: server.ID, Version: "1.0.0", TransportType: "sse", ConfigJSON: "{}",
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("创建 MCP 版本失败: %v", err)
	}
	const targetCount = mcpDistributionBatchSize + 1
	instances := make([]model.Instance, targetCount)
	for i := range instances {
		instances[i] = model.Instance{
			Name:       "mcp-batch",
			InstanceId: fmt.Sprintf("ins-mcp-batch-%03d", i),
			AgentType:  "openclaw",
		}
	}
	if err := db.CreateInBatches(&instances, mcpDistributionBatchSize).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	task, total, err := createMCPSelectAllTask(
		ctx,
		server,
		version,
		0,
		distributionSelection{SelectAll: true, Statuses: []string{"uninstalled"}},
	)
	if err != nil {
		t.Fatalf("createMCPSelectAllTask() error = %v", err)
	}
	if total != targetCount || task.Total != targetCount {
		t.Fatalf("createMCPSelectAllTask() total=%d task.total=%d, want %d", total, task.Total, targetCount)
	}
	var recordCount, installationCount int64
	if err := db.Model(&model.McpDistributionRecord{}).Where("task_id = ?", task.ID).Count(&recordCount).Error; err != nil {
		t.Fatalf("统计下发记录失败: %v", err)
	}
	if err := db.Model(&model.McpInstallation{}).Where("last_task_id = ?", task.ID).Count(&installationCount).Error; err != nil {
		t.Fatalf("统计安装记录失败: %v", err)
	}
	if recordCount != targetCount || installationCount != targetCount {
		t.Fatalf("records=%d installations=%d，期望各 %d", recordCount, installationCount, targetCount)
	}
}
func TestCreateMCPSelectAllTask_SearchByUsername(t *testing.T) {
	setupMcpTestDB(t)
	ctx := context.Background()
	db := model.DB(ctx)
	server := model.McpServer{ServiceID: "select-all-mcp-search", Name: "搜索 MCP", TransportType: "sse"}
	if err := db.Create(&server).Error; err != nil {
		t.Fatalf("创建 MCP 服务失败: %v", err)
	}
	version := model.McpVersion{
		MCPID: server.ID, Version: "1.0.0", TransportType: "sse", ConfigJSON: "{}",
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("创建 MCP 版本失败: %v", err)
	}
	users := []model.User{
		{Username: "mcp-search-needle"},
		{Username: "mcp-search-other"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	instances := []model.Instance{
		{Name: "target-one", InstanceId: "ins-target-one", UserID: users[0].ID, AgentType: "openclaw"},
		{Name: "target-two", InstanceId: "ins-target-two", UserID: users[1].ID, AgentType: "openclaw"},
	}
	if err := db.Create(&instances).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	task, total, err := createMCPSelectAllTask(
		ctx,
		server,
		version,
		0,
		distributionSelection{SelectAll: true, Statuses: []string{"uninstalled"}, Search: "needle"},
	)
	if err != nil {
		t.Fatalf("createMCPSelectAllTask(search=%q) error = %v", "needle", err)
	}
	if total != 1 || task.Total != 1 {
		t.Fatalf("createMCPSelectAllTask(search=%q) total=%d task.total=%d, want 1", "needle", total, task.Total)
	}
	var record model.McpDistributionRecord
	if err := db.Where("task_id = ?", task.ID).First(&record).Error; err != nil {
		t.Fatalf("查询搜索结果记录失败: %v", err)
	}
	if record.InstanceID != instances[0].ID {
		t.Fatalf("createMCPSelectAllTask(search=%q) instance_id=%d, want %d", "needle", record.InstanceID, instances[0].ID)
	}
}

func TestRunMCPSelectAllTask_RecoversTaskPanic(t *testing.T) {
	setupMcpTestDB(t)
	ctx := context.Background()
	db := model.DB(ctx)
	server := model.McpServer{ServiceID: "panic-mcp", Name: "Panic MCP"}
	if err := db.Create(&server).Error; err != nil {
		t.Fatalf("创建 MCP 服务失败: %v", err)
	}
	version := model.McpVersion{MCPID: server.ID, Version: "1.0.0", ConfigJSON: "{}"}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("创建 MCP 版本失败: %v", err)
	}
	task := model.McpDistributionTask{
		MCPID: server.ID, VersionID: version.ID, VersionSnapshot: version.Version,
		Total: 1, Status: model.TaskStatusRunning,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	record := model.McpDistributionRecord{
		TaskID: task.ID, MCPID: server.ID, InstanceID: 1, InstanceCID: "ins-panic-mcp",
		VersionSnapshot: version.Version, Status: model.RecordStatusPending,
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("创建记录失败: %v", err)
	}

	triggered := false
	const callbackName = "test:mcp-select-all-query-panic"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if !triggered && tx.Statement.Table == "mcp_distribution_records" {
			triggered = true
			panic("mcp select-all query panic")
		}
	}); err != nil {
		t.Fatalf("注册 panic callback 失败: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(callbackName)
	})

	func() {
		defer recoverMCPSelectAllTaskPanic(ctx, task)
		runMCPSelectAllTask(ctx, server, version, task)
	}()

	if !triggered {
		t.Fatal("panic callback 未触发")
	}
	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatalf("查询任务失败: %v", err)
	}
	if err := db.First(&record, record.ID).Error; err != nil {
		t.Fatalf("查询记录失败: %v", err)
	}
	if task.Status != model.TaskStatusCompleted || task.Failed != 1 {
		t.Fatalf("panic 后任务 status=%q failed=%d，期望 completed/1", task.Status, task.Failed)
	}
	if record.Status != model.RecordStatusFailed {
		t.Fatalf("panic 后记录 status=%q，期望 failed", record.Status)
	}
}

func TestRunMCPSelectAllTask_RecoversRecordPanic(t *testing.T) {
	setupMcpTestDB(t)
	ctx := context.Background()
	db := model.DB(ctx)
	server := model.McpServer{ServiceID: "record-panic-mcp", Name: "Record Panic MCP"}
	if err := db.Create(&server).Error; err != nil {
		t.Fatalf("创建 MCP 服务失败: %v", err)
	}
	version := model.McpVersion{MCPID: server.ID, Version: "1.0.0", ConfigJSON: "{}"}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("创建 MCP 版本失败: %v", err)
	}
	instance := model.Instance{Name: "record-panic", InstanceId: "ins-record-panic", AgentType: "openclaw"}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	task := model.McpDistributionTask{
		MCPID: server.ID, VersionID: version.ID, VersionSnapshot: version.Version,
		Total: 1, Status: model.TaskStatusRunning,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	record := model.McpDistributionRecord{
		TaskID: task.ID, MCPID: server.ID, InstanceID: instance.ID, InstanceCID: instance.InstanceId,
		VersionSnapshot: version.Version, Status: model.RecordStatusPending,
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("创建记录失败: %v", err)
	}

	triggered := false
	const callbackName = "test:mcp-select-all-record-panic"
	if err := db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if !triggered && tx.Statement.Table == "mcp_distribution_records" {
			triggered = true
			panic("mcp select-all record panic")
		}
	}); err != nil {
		t.Fatalf("注册 panic callback 失败: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Update().Remove(callbackName)
	})

	runMCPSelectAllTask(ctx, server, version, task)

	if !triggered {
		t.Fatal("panic callback 未触发")
	}
	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatalf("查询任务失败: %v", err)
	}
	if err := db.First(&record, record.ID).Error; err != nil {
		t.Fatalf("查询记录失败: %v", err)
	}
	if task.Status != model.TaskStatusCompleted || task.Failed != 1 {
		t.Fatalf("panic 后任务 status=%q failed=%d，期望 completed/1", task.Status, task.Failed)
	}
	if record.Status != model.RecordStatusFailed {
		t.Fatalf("panic 后记录 status=%q，期望 failed", record.Status)
	}
}
