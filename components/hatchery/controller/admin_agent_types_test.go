package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initAgentTypesTestDB 初始化内存 SQLite 数据库用于 agent types 测试
func initAgentTypesTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.UserGroup{}, &model.UserGroupMember{}, &model.GroupClosure{}, &model.GroupConfigBinding{}, &model.Instance{}, &model.AIImage{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	// 设置 AdminToken 用于测试
	AdminToken = "test-admin-token"

	// 初始化 session store
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	return func() {
		origDB()
		Store = origStore
	}
}

// adminAgentTypesReq 构造携带管理员 Token 的 HTTP 请求
func adminAgentTypesReq(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// userAgentTypesReqWithSession 构造携带用户 session 的 HTTP 请求
func userAgentTypesReqWithSession(t *testing.T, method, path string, username string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Accept", "application/json")

	// 创建一个带有 session 的请求
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username

	// 由于 httptest.Request 不支持真正的 cookie，我们需要模拟 session
	// 这里使用一种 workaround：直接在 Store 中保存 session
	rr := httptest.NewRecorder()
	session.Save(req, rr)

	// 从响应中获取 cookie 并添加到新请求中
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	return req
}

func TestHandleAdminAgentTypes_MethodNotAllowed(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()

	// 测试 POST 请求应该返回 405
	req := adminAgentTypesReq(http.MethodPost, "/api/admin/agent-types")
	rr := httptest.NewRecorder()

	HandleAdminAgentTypes(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleAdminAgentTypes_Success(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()

	// 创建一些测试镜像
	images := []model.AIImage{
		{ImageId: "img-001", ImageName: "OpenClaw Image", AgentType: "openclaw", AgentVersion: "2026.1.1", Enabled: true},
		{ImageId: "img-002", ImageName: "Hermes Image", AgentType: "hermes", AgentVersion: "1.0.0", Enabled: true},
		{ImageId: "img-003", ImageName: "Disabled Image", AgentType: "lightclawace", AgentVersion: "0.1.0", Enabled: false},
	}
	for _, img := range images {
		if err := model.DB(context.Background()).Create(&img).Error; err != nil {
			t.Fatalf("创建镜像失败: %v", err)
		}
	}

	// 创建测试实例
	instances := []model.Instance{
		{Name: "OpenClaw Instance", AgentType: "openclaw"},
		{Name: "Legacy OpenClaw Instance", AgentType: ""},
		{Name: "Hermes Instance", AgentType: "hermes"},
	}
	for _, inst := range instances {
		if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
			t.Fatalf("创建实例失败: %v", err)
		}
	}

	// 创建 site config
	config := model.SiteConfig{ID: 1, DefaultAgentType: "openclaw"}
	if err := model.DB(context.Background()).Create(&config).Error; err != nil {
		t.Fatalf("创建配置失败: %v", err)
	}

	req := adminAgentTypesReq(http.MethodGet, "/api/admin/agent-types")
	rr := httptest.NewRecorder()

	HandleAdminAgentTypes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证返回内容
	agentTypes, ok := resp["agent_types"].([]interface{})
	if !ok {
		t.Fatalf("expected agent_types array")
	}
	if len(agentTypes) == 0 {
		t.Error("expected non-empty agent_types")
	}

	defaultType, ok := resp["default_agent_type"].(string)
	if !ok || defaultType != "openclaw" {
		t.Errorf("expected default_agent_type=openclaw, got %v", resp["default_agent_type"])
	}

	// 验证 openclaw 有启用镜像
	foundOpenclaw := false
	for _, at := range agentTypes {
		atMap := at.(map[string]interface{})
		if atMap["code"] == "openclaw" {
			foundOpenclaw = true
			if atMap["has_enabled_image"] != true {
				t.Error("openclaw should have enabled image")
			}
			if atMap["is_default"] != true {
				t.Error("openclaw should be default")
			}
			if atMap["image_count"].(float64) != 1 {
				t.Errorf("openclaw image_count should be 1, got %v", atMap["image_count"])
			}
			if atMap["instance_count"].(float64) != 2 {
				t.Errorf("openclaw instance_count should be 2, got %v", atMap["instance_count"])
			}
		}
	}
	if !foundOpenclaw {
		t.Error("openclaw not found in response")
	}
}

func TestAgentTypesDisabledBySiteConfig(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.SiteConfig{ID: 1, DefaultAgentType: "openclaw", DisabledAgentTypes: `["hermes"]`}).Error; err != nil {
		t.Fatalf("创建配置失败: %v", err)
	}
	images := []model.AIImage{
		{ImageId: "img-001", ImageName: "OpenClaw", AgentType: "openclaw", AgentVersion: "2026.1.1", Enabled: true},
		{ImageId: "img-002", ImageName: "Hermes", AgentType: "hermes", AgentVersion: "1.0.0", Enabled: true},
	}
	for _, img := range images {
		if err := model.DB(context.Background()).Create(&img).Error; err != nil {
			t.Fatalf("创建镜像失败: %v", err)
		}
	}

	adminReq := adminAgentTypesReq(http.MethodGet, "/api/admin/agent-types")
	adminRR := httptest.NewRecorder()
	HandleAdminAgentTypes(adminRR, adminReq)
	if adminRR.Code != http.StatusOK {
		t.Fatalf("admin expected status 200, got %d body=%s", adminRR.Code, adminRR.Body.String())
	}
	var adminResp map[string]interface{}
	if err := json.Unmarshal(adminRR.Body.Bytes(), &adminResp); err != nil {
		t.Fatalf("解析管理响应失败: %v", err)
	}
	for _, at := range adminResp["agent_types"].([]interface{}) {
		atMap := at.(map[string]interface{})
		if atMap["code"] == "hermes" {
			if atMap["enabled"] != false {
				t.Fatalf("hermes enabled=%v，期望 false", atMap["enabled"])
			}
			if atMap["has_enabled_image"] != true {
				t.Fatalf("hermes 应仍有当前启用镜像")
			}
			if atMap["user_selectable"] != false {
				t.Fatalf("hermes user_selectable=%v，期望 false", atMap["user_selectable"])
			}
		}
	}

	userReq := userAgentTypesReqWithSession(t, http.MethodGet, "/openclaw/agent-types", "testuser")
	userRR := httptest.NewRecorder()
	HandleUserAgentTypes(userRR, userReq)
	if userRR.Code != http.StatusOK {
		t.Fatalf("user expected status 200, got %d body=%s", userRR.Code, userRR.Body.String())
	}
	var userResp map[string]interface{}
	if err := json.Unmarshal(userRR.Body.Bytes(), &userResp); err != nil {
		t.Fatalf("解析用户响应失败: %v", err)
	}
	typeNames := map[string]bool{}
	for _, at := range userResp["agent_types"].([]interface{}) {
		atMap := at.(map[string]interface{})
		typeNames[atMap["code"].(string)] = true
	}
	if !typeNames["openclaw"] {
		t.Fatal("openclaw should be selectable")
	}
	if typeNames["hermes"] {
		t.Fatal("disabled hermes should not be selectable")
	}
}

func TestHandleUpdateAgentTypeEnabledSupportsEnableDisableToggle(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).Create(&model.SiteConfig{ID: 1, DefaultAgentType: "openclaw"}).Error; err != nil {
		t.Fatalf("创建配置失败: %v", err)
	}

	queryReq := func(query string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/admin/agent-types/enabled?"+query, nil)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer test-admin-token")
		return req
	}

	// disable by explicit enabled boolean in query string
	rr := httptest.NewRecorder()
	HandleUpdateAgentTypeEnabled(rr, queryReq("agent_type=hermes&enabled=false"))
	if rr.Code != http.StatusOK {
		t.Fatalf("disable expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if model.IsAgentTypeEnabled(context.Background(), "hermes") {
		t.Fatal("hermes should be disabled")
	}

	// enable by explicit enabled boolean (form request)
	req := adminAgentTypesPost("/admin/agent-types/enabled", "agent_type=hermes&enabled=true")
	rr = httptest.NewRecorder()
	HandleUpdateAgentTypeEnabled(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("enable expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !model.IsAgentTypeEnabled(context.Background(), "hermes") {
		t.Fatal("hermes should be enabled")
	}

	// toggle flips current state
	rr = httptest.NewRecorder()
	HandleUpdateAgentTypeEnabled(rr, queryReq("agent_type=hermes&toggle=true"))
	if rr.Code != http.StatusOK {
		t.Fatalf("toggle expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if model.IsAgentTypeEnabled(context.Background(), "hermes") {
		t.Fatal("hermes should be disabled after toggle")
	}

	// enabled and toggle are mutually exclusive
	rr = httptest.NewRecorder()
	HandleUpdateAgentTypeEnabled(rr, queryReq("agent_type=hermes&enabled=true&toggle=true"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("enabled+toggle expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}

	// default type cannot be disabled
	rr = httptest.NewRecorder()
	HandleUpdateAgentTypeEnabled(rr, queryReq("agent_type=openclaw&enabled=false"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("disable default expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAdminAgentTypes_NoImages(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()

	req := adminAgentTypesReq(http.MethodGet, "/api/admin/agent-types")
	rr := httptest.NewRecorder()

	HandleAdminAgentTypes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证所有类型都没有启用镜像
	agentTypes := resp["agent_types"].([]interface{})
	for _, at := range agentTypes {
		atMap := at.(map[string]interface{})
		if atMap["has_enabled_image"] == true {
			t.Errorf("type %s should not have enabled image", atMap["code"])
		}
	}
}

func TestHandleUserAgentTypes_MethodNotAllowed(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()

	// 创建测试用户
	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := userAgentTypesReqWithSession(t, http.MethodPost, "/api/user/agent-types", "testuser")
	rr := httptest.NewRecorder()

	HandleUserAgentTypes(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleUserAgentTypes_Success(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()

	// 创建测试用户
	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建启用的镜像
	images := []model.AIImage{
		{ImageId: "img-001", ImageName: "OpenClaw", AgentType: "openclaw", AgentVersion: "2026.1.1", Enabled: true},
		{ImageId: "img-002", ImageName: "Hermes", AgentType: "hermes", AgentVersion: "1.0.0", Enabled: true},
		{ImageId: "img-003", ImageName: "Disabled", AgentType: "lightclawace", AgentVersion: "0.1.0", Enabled: false},
	}
	for _, img := range images {
		model.DB(context.Background()).Create(&img)
	}

	// 创建 site config
	config := model.SiteConfig{ID: 1, DefaultAgentType: "openclaw"}
	model.DB(context.Background()).Create(&config)

	req := userAgentTypesReqWithSession(t, http.MethodGet, "/api/user/agent-types", "testuser")
	rr := httptest.NewRecorder()

	HandleUserAgentTypes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证只返回有启用镜像的类型
	agentTypes := resp["agent_types"].([]interface{})

	// 应该只有 openclaw 和 hermes（lightclawace 未启用）
	if len(agentTypes) != 2 {
		t.Errorf("expected 2 selectable types, got %d", len(agentTypes))
	}

	typeNames := make(map[string]bool)
	for _, at := range agentTypes {
		atMap := at.(map[string]interface{})
		typeNames[atMap["code"].(string)] = true
	}

	if !typeNames["openclaw"] {
		t.Error("openclaw should be selectable")
	}
	if !typeNames["hermes"] {
		t.Error("hermes should be selectable")
	}
	if typeNames["lightclawace"] {
		t.Error("lightclawace should NOT be selectable (disabled)")
	}

	// 验证内置类型返回 is_builtin=true，无 compatible_with
	for _, at := range agentTypes {
		atMap := at.(map[string]interface{})
		if atMap["code"] == "openclaw" {
			if atMap["is_builtin"] != true {
				t.Errorf("openclaw should have is_builtin=true, got %v", atMap["is_builtin"])
			}
			if _, hasCompat := atMap["compatible_with"]; hasCompat {
				t.Errorf("openclaw should not have compatible_with (omitempty), got %v", atMap["compatible_with"])
			}
		}
	}
}

// TestHandleUserAgentTypes_CustomWithCompatibleWith 验证自定义类型在用户侧返回 compatible_with 和 is_builtin=false。
func TestHandleUserAgentTypes_CustomWithCompatibleWith(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建自定义类型（兼容 openclaw）
	if _, err := model.CreateCustomAgentType(context.Background(), "my-oc", model.AgentTypeOpenClaw); err != nil {
		t.Fatalf("create custom type: %v", err)
	}

	// 给自定义类型创建启用镜像
	model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-custom", ImageName: "Custom", AgentType: "my-oc", Enabled: true})
	// 内置类型也需要有镜像才会出现
	model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-oc", ImageName: "OC", AgentType: "openclaw", AgentVersion: "2026.1.1", Enabled: true})

	config := model.SiteConfig{ID: 1, DefaultAgentType: "openclaw"}
	model.DB(context.Background()).Create(&config)

	req := userAgentTypesReqWithSession(t, http.MethodGet, "/api/user/agent-types", "testuser")
	rr := httptest.NewRecorder()
	HandleUserAgentTypes(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	agentTypes := resp["agent_types"].([]interface{})

	var foundCustom bool
	for _, at := range agentTypes {
		atMap := at.(map[string]interface{})
		if atMap["code"] == "my-oc" {
			foundCustom = true
			if atMap["is_builtin"] != false {
				t.Errorf("custom type should have is_builtin=false, got %v", atMap["is_builtin"])
			}
			if atMap["compatible_with"] != "openclaw" {
				t.Errorf("custom type should have compatible_with=openclaw, got %v", atMap["compatible_with"])
			}
		}
	}
	if !foundCustom {
		t.Error("custom type 'my-oc' with enabled image should be in the list")
	}
}

func TestHandleUserAgentTypes_NoImages(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()

	// 创建测试用户
	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := userAgentTypesReqWithSession(t, http.MethodGet, "/api/user/agent-types", "testuser")
	rr := httptest.NewRecorder()

	HandleUserAgentTypes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 没有启用镜像时，selectable types 应该为空
	agentTypes := resp["agent_types"]
	if agentTypes != nil {
		typesArr := agentTypes.([]interface{})
		if len(typesArr) != 0 {
			t.Errorf("expected empty selectable types, got %d", len(typesArr))
		}
	}
}

func adminAgentTypesPost(path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

func TestHandleCreateCustomAgentType(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()

	req := adminAgentTypesPost("/admin/agent-types/create", "name=my-agent&compatible_with=openclaw")
	rr := httptest.NewRecorder()
	HandleCreateCustomAgentType(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !model.IsCustomAgentType(context.Background(), "my-agent") {
		t.Fatal("custom agent type should be created")
	}
	if got := model.GetAgentRuntimeType(context.Background(), "my-agent"); got != model.AgentTypeOpenClaw {
		t.Fatalf("runtime type = %q", got)
	}

	req = adminAgentTypesPost("/admin/agent-types/create", "name=my-agent&compatible_with=openclaw")
	rr = httptest.NewRecorder()
	HandleCreateCustomAgentType(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("duplicate create should fail, got %d", rr.Code)
	}

	req = adminAgentTypesPost("/admin/agent-types/create", "name=openclaw")
	rr = httptest.NewRecorder()
	HandleCreateCustomAgentType(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("builtin name should fail, got %d", rr.Code)
	}

	req = adminAgentTypesPost("/admin/agent-types/create", "name=+spaced")
	rr = httptest.NewRecorder()
	HandleCreateCustomAgentType(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("name with leading space should fail, got %d", rr.Code)
	}
	if model.IsCustomAgentType(context.Background(), "spaced") {
		t.Fatal("handler must not create a different display name")
	}

	req = adminAgentTypesPost("/admin/agent-types/create", "name=bad-compatible&compatible_with=+openclaw")
	rr = httptest.NewRecorder()
	HandleCreateCustomAgentType(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("compatible_with with leading space should fail, got %d", rr.Code)
	}
}

func TestHandleDeleteCustomAgentType(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()

	if _, err := model.CreateCustomAgentType(context.Background(), "delete-agent", ""); err != nil {
		t.Fatalf("create custom type: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.SiteConfig{ID: 1, DefaultAgentType: model.AgentTypeOpenClaw, DisabledAgentTypes: `["delete-agent"]`}).Error; err != nil {
		t.Fatalf("create site config: %v", err)
	}
	req := adminAgentTypesPost("/admin/agent-types/delete", "name=delete-agent")
	rr := httptest.NewRecorder()
	HandleDeleteCustomAgentType(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if model.IsCustomAgentType(context.Background(), "delete-agent") {
		t.Fatal("custom agent type should be deleted")
	}

	req = adminAgentTypesPost("/admin/agent-types/delete", "name=openclaw")
	rr = httptest.NewRecorder()
	HandleDeleteCustomAgentType(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("deleting builtin should fail, got %d", rr.Code)
	}
}

func TestHandleUserAgentTypes_CustomRequiresOwnEnabledImage(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()

	user := model.User{Username: "custom-user", Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := model.CreateCustomAgentType(context.Background(), "custom-no-image", model.AgentTypeOpenClaw); err != nil {
		t.Fatalf("create custom-no-image: %v", err)
	}
	if _, err := model.CreateCustomAgentType(context.Background(), "custom-with-image", model.AgentTypeOpenClaw); err != nil {
		t.Fatalf("create custom-with-image: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-openclaw", AgentType: model.AgentTypeOpenClaw, Enabled: true}).Error; err != nil {
		t.Fatalf("create openclaw image: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-custom", AgentType: "custom-with-image", Enabled: true}).Error; err != nil {
		t.Fatalf("create custom image: %v", err)
	}

	req := userAgentTypesReqWithSession(t, http.MethodGet, "/openclaw/agent-types", user.Username)
	rr := httptest.NewRecorder()
	HandleUserAgentTypes(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	seen := map[string]bool{}
	for _, raw := range resp["agent_types"].([]any) {
		item := raw.(map[string]any)
		seen[item["code"].(string)] = true
	}
	if seen["custom-no-image"] {
		t.Fatal("custom type without its own enabled image should not be selectable")
	}
	if !seen["custom-with-image"] {
		t.Fatal("custom type with its own enabled image should be selectable")
	}
}

func TestHandleCreateCustomAgentType_MethodNotAllowed(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/admin/agent-types/create", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleCreateCustomAgentType(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleDeleteCustomAgentType_MethodNotAllowed(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/admin/agent-types/delete?name=x", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleDeleteCustomAgentType(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleUpdateAgentTypeEnabledValidationErrors(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()
	if err := model.DB(context.Background()).Create(&model.SiteConfig{ID: 1, DefaultAgentType: model.AgentTypeOpenClaw}).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	cases := []struct {
		name string
		req  *http.Request
	}{
		{"method", adminAgentTypesReq(http.MethodGet, "/admin/agent-types/enabled")},
		{"missing-agent-type", adminAgentTypesPost("/admin/agent-types/enabled", "enabled=true")},
		{"bad-enabled", adminAgentTypesPost("/admin/agent-types/enabled", "agent_type=hermes&enabled=maybe")},
		{"bad-toggle", adminAgentTypesPost("/admin/agent-types/enabled", "agent_type=hermes&toggle=maybe")},
		{"both", adminAgentTypesPost("/admin/agent-types/enabled", "agent_type=hermes&enabled=true&toggle=true")},
		{"neither", adminAgentTypesPost("/admin/agent-types/enabled", "agent_type=hermes")},
		{"toggle-false", adminAgentTypesPost("/admin/agent-types/enabled", "agent_type=hermes&toggle=false")},
		{"invalid-type", adminAgentTypesPost("/admin/agent-types/enabled", "agent_type=bad-type&enabled=false")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			HandleUpdateAgentTypeEnabled(rr, tc.req)
			if rr.Code < 400 {
				t.Fatalf("expected error status, got %d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestAgentTypeListValidationHelpers(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()
	if err := model.DB(context.Background()).Create(&model.SiteConfig{ID: 1, DefaultAgentType: model.AgentTypeOpenClaw}).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	normalized := normalizeAgentTypeList([]string{" hermes ", "", "hermes", "lightclawace"})
	if len(normalized) != 2 || normalized[0] != model.AgentTypeHermes || normalized[1] != model.AgentTypeLightclawACE {
		t.Fatalf("normalized = %v", normalized)
	}
	if err := validateDisabledAgentTypes(context.Background(), []string{model.AgentTypeHermes}); err != nil {
		t.Fatalf("valid disabled types: %v", err)
	}
	if err := validateDisabledAgentTypes(context.Background(), []string{"bad-type"}); err == nil {
		t.Fatal("invalid disabled type should fail")
	}
	if err := validateDisabledAgentTypes(context.Background(), []string{model.AgentTypeOpenClaw}); err == nil {
		t.Fatal("default disabled type should fail")
	}
}

func TestHandleAdminAgentTypesInstanceCountError(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()
	if err := model.DB(context.Background()).Migrator().DropTable(&model.Instance{}); err != nil {
		t.Fatalf("drop instances: %v", err)
	}

	req := adminAgentTypesReq(http.MethodGet, "/api/admin/agent-types")
	rr := httptest.NewRecorder()
	HandleAdminAgentTypes(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdateAgentTypeEnabledUnauthorized(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodPost, "/admin/agent-types/enabled?agent_type=hermes&enabled=false", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleUpdateAgentTypeEnabled(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}
