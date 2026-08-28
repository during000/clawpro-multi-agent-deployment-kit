package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initGatewayTestDB 初始化内存 SQLite 数据库用于 gateway 测试
func initGatewayTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	AdminToken = "test-admin-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	return func() {
		origDB()
		Store = origStore
	}
}

func userGatewayReqWithSession(t *testing.T, method, path string, username string) *http.Request {
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

// ==================== HandleSetGatewayUi Tests ====================

func TestHandleSetGatewayUi_MethodNotAllowed(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := userGatewayReqWithSession(t, http.MethodGet, "/openclaw/gateway-ui?id=1", "testuser")
	rr := httptest.NewRecorder()

	handleSetGatewayUi(rr, req, testCVMFetcher)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", rr.Code)
	}
}

func TestHandleSetGatewayUi_GatewayNotEnabled(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建实例，使 getInstanceByID 可以找到（id=1 属于该用户）
	inst := &model.Instance{Name: "test-inst", InstanceId: "ins-gw-test", UserID: user.ID, AgentType: model.AgentTypeOpenClaw}
	model.DB(context.Background()).Create(inst)

	// SiteConfig: GatewayUIEnable=false
	config := model.SiteConfig{ID: 1, GatewayUIEnable: false}
	model.DB(context.Background()).Create(&config)

	req := userGatewayReqWithSession(t, http.MethodPost, "/openclaw/gateway-ui?id=1", "testuser")
	rr := httptest.NewRecorder()

	handleSetGatewayUi(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 when gateway not enabled, got %d", rr.Code)
	}
}

func TestHandleSetGatewayUi_GatewayNoPort(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	config := model.SiteConfig{ID: 1, GatewayUIEnable: true, GatewayUIPort: 0}
	model.DB(context.Background()).Create(&config)

	req := userGatewayReqWithSession(t, http.MethodPost, "/openclaw/gateway-ui?id=1", "testuser")
	rr := httptest.NewRecorder()

	handleSetGatewayUi(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when gateway port not set, got %d", rr.Code)
	}
}

func TestHandleSetGatewayUi_InstanceNotFound(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	config := model.SiteConfig{ID: 1, GatewayUIEnable: true, GatewayUIPort: 8080}
	model.DB(context.Background()).Create(&config)

	req := userGatewayReqWithSession(t, http.MethodPost, "/openclaw/gateway-ui?id=99999", "testuser")
	rr := httptest.NewRecorder()

	handleSetGatewayUi(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-existent instance, got %d", rr.Code)
	}
}

// ==================== HandleCheckGatewayAccess Tests ====================

func TestHandleCheckGatewayAccess_MethodNotAllowed(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := userGatewayReqWithSession(t, http.MethodPost, "/openclaw/check-gateway-access?id=1", "testuser")
	rr := httptest.NewRecorder()

	HandleCheckGatewayAccess(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for POST, got %d", rr.Code)
	}
}

func TestHandleCheckGatewayAccess_GatewayNotEnabled(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建实例，使 getInstanceByID 可以找到
	inst := &model.Instance{Name: "test-inst", InstanceId: "ins-gw-test", UserID: user.ID, AgentType: model.AgentTypeOpenClaw}
	model.DB(context.Background()).Create(inst)

	config := model.SiteConfig{ID: 1, GatewayUIEnable: false}
	model.DB(context.Background()).Create(&config)

	req := userGatewayReqWithSession(t, http.MethodGet, "/openclaw/check-gateway-access?id=1", "testuser")
	rr := httptest.NewRecorder()

	HandleCheckGatewayAccess(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 when gateway not enabled, got %d", rr.Code)
	}
}

func TestHandleCheckGatewayAccess_InstanceNotFound(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	config := model.SiteConfig{ID: 1, GatewayUIEnable: true, GatewayUIPort: 8080}
	model.DB(context.Background()).Create(&config)

	req := userGatewayReqWithSession(t, http.MethodGet, "/openclaw/check-gateway-access?id=99999", "testuser")
	rr := httptest.NewRecorder()

	HandleCheckGatewayAccess(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-existent instance, got %d", rr.Code)
	}
}

func TestHandleCheckGatewayAccess_InstanceNoCVM(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	config := model.SiteConfig{ID: 1, GatewayUIEnable: true, GatewayUIPort: 8080}
	model.DB(context.Background()).Create(&config)

	proxyToken := "sk-test"
	inst := model.Instance{
		Name:       "no-cvm",
		InstanceId: "", // 无 CVM
		UserID:     user.ID,
		AgentType:  "openclaw",
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	req := userGatewayReqWithSession(t, http.MethodGet, "/openclaw/check-gateway-access?id=1", "testuser")
	rr := httptest.NewRecorder()

	HandleCheckGatewayAccess(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for instance without CVM, got %d", rr.Code)
	}
}

// ==================== HandleAdminAgentTypes 补充测试 ====================

func TestHandleAdminAgentTypes_ResponseHasAgentTypes(t *testing.T) {
	cleanup := initAgentTypesTestDB(t)
	defer cleanup()

	// 创建 SiteConfig 避免 GetSiteConfig 报错
	config := model.SiteConfig{ID: 1}
	model.DB(context.Background()).Create(&config)

	images := []model.AIImage{
		{ImageId: "img-001", ImageName: "OC1", AgentType: "openclaw", AgentVersion: "2026.1.1", Enabled: true},
		{ImageId: "img-002", ImageName: "Hermes1", AgentType: "hermes", AgentVersion: "1.0.0", Enabled: true},
	}
	for _, img := range images {
		model.DB(context.Background()).Create(&img)
	}

	req := adminAgentTypesReq(http.MethodGet, "/api/admin/agent-types")
	rr := httptest.NewRecorder()

	HandleAdminAgentTypes(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// 验证包含 agent_types 字段
	agentTypes, ok := resp["agent_types"].([]interface{})
	if !ok {
		t.Fatalf("expected agent_types array, got %T", resp["agent_types"])
	}

	if len(agentTypes) != 5 {
		t.Errorf("expected 5 agent types, got %d", len(agentTypes))
	}

	// 验证每个 agent type 有必要字段
	for i, at := range agentTypes {
		atMap, ok := at.(map[string]interface{})
		if !ok {
			t.Errorf("agent_types[%d] is not a map", i)
			continue
		}
		if _, ok := atMap["code"]; !ok {
			t.Errorf("agent_types[%d] missing 'code' field", i)
		}
		if _, ok := atMap["name"]; !ok {
			t.Errorf("agent_types[%d] missing 'name' field", i)
		}
	}

	// 验证包含 default_agent_type 字段
	if _, ok := resp["default_agent_type"]; !ok {
		t.Error("expected default_agent_type field in response")
	}
}

func TestHandleRestartGatewayInstance_Success(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	ctx := context.Background()
	user := &model.User{Username: "restart-gateway-user", Password: "test", Role: "user"}
	model.DB(ctx).Create(user)
	inst := &model.Instance{
		Name:             "restart-gateway-user-inst",
		InstanceId:       "ins-user-restart-gateway",
		UserID:           user.ID,
		AgentType:        model.AgentTypeOpenClaw,
		RuntimeUser:      "agentuser",
		AgentReady:       1,
		LastCVMState:     "RUNNING",
		CurrentOperation: "",
	}
	model.DB(ctx).Create(inst)

	origRunner := agentScriptRunner
	var gotScriptName string
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		if instanceId != inst.InstanceId || runtimeUser != "agentuser" || timeout != 60 {
			t.Fatalf("脚本下发参数不对: instance=%s runtime=%s timeout=%d", instanceId, runtimeUser, timeout)
		}
		gotScriptName = scriptName
		return "ok", nil
	}
	t.Cleanup(func() { agentScriptRunner = origRunner })

	req := userGatewayReqWithSession(t, http.MethodPost, "/openclaw/restart-gateway?id=1", "restart-gateway-user")
	rr := httptest.NewRecorder()
	handleRestartGatewayInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusOK {
		t.Fatalf("重启 Agent 应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if gotScriptName != "restart_gateway.sh" {
		t.Fatalf("脚本名=%q, want restart_gateway.sh", gotScriptName)
	}

	var fresh model.Instance
	if err := model.DB(ctx).First(&fresh, inst.ID).Error; err != nil {
		t.Fatalf("查询实例失败: %v", err)
	}
	if fresh.CurrentOperation != "" || fresh.AgentReady != 1 {
		t.Fatalf("重启 Agent 不应写实例操作状态或重置 agent_ready: op=%q agent_ready=%d", fresh.CurrentOperation, fresh.AgentReady)
	}
}

func TestHandleRestartGatewayInstance_MethodNotAllowed(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/restart-gateway?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleRestartGatewayInstance(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET 重启 Agent 应返回 405，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRestartGatewayInstance_RequiresLogin(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/restart-gateway?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	handleRestartGatewayInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("未登录重启 Agent 应返回 401，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRestartGatewayInstance_InstanceNotFound(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	ctx := context.Background()
	user := &model.User{Username: "restart-gateway-missing-user", Password: "test", Role: "user"}
	model.DB(ctx).Create(user)

	req := userGatewayReqWithSession(t, http.MethodPost, "/openclaw/restart-gateway?id=99999", "restart-gateway-missing-user")
	rr := httptest.NewRecorder()

	handleRestartGatewayInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("不存在实例重启 Agent 应返回 404，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRestartGatewayInstance_InstanceWithoutCVM(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	ctx := context.Background()
	user := &model.User{Username: "restart-gateway-nocvm-user", Password: "test", Role: "user"}
	model.DB(ctx).Create(user)
	inst := &model.Instance{
		Name:       "restart-gateway-nocvm-inst",
		InstanceId: "",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
	}
	model.DB(ctx).Create(inst)

	req := userGatewayReqWithSession(t, http.MethodPost, "/openclaw/restart-gateway?id=1", "restart-gateway-nocvm-user")
	rr := httptest.NewRecorder()

	handleRestartGatewayInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("无 CVM 实例重启 Agent 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ==================== setHermesUI Tests ====================

func TestSetHermesUI_Success(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	ctx := context.Background()
	user := &model.User{Username: "hermes-ui-user", Password: "test", Role: "user"}
	model.DB(ctx).Create(user)
	inst := &model.Instance{
		Name:        "hermes-inst",
		InstanceId:  "ins-hermes-001",
		UserID:      user.ID,
		AgentType:   model.AgentTypeHermes,
		RuntimeUser: "agentuser",
	}
	model.DB(ctx).Create(inst)

	// 完整 mock RunScript 链路：LoadScript + NewTATClient → mock server
	m := newRunScriptMockServer(t)
	m.invocationStates = append(m.invocationStates, runningTask(), finishedTask("SUCCESS", "ok", ""))
	setupRunScriptEnv(t, m.server.URL)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	params := map[string]string{"gateway_ui_port": "10815"}

	setHermesUI(rr, req, inst, "1.2.3.4", params)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["gatewayUI"] != "http://1.2.3.4:10815/" {
		t.Fatalf("gatewayUI=%q, want http://1.2.3.4:10815/", resp["gatewayUI"])
	}
	if resp["token"] != "" {
		t.Fatalf("token=%q, want empty", resp["token"])
	}
}

func TestSetHermesUI_RunScriptError(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	ctx := context.Background()
	user := &model.User{Username: "hermes-ui-err-user", Password: "test", Role: "user"}
	model.DB(ctx).Create(user)
	inst := &model.Instance{
		Name:        "hermes-err-inst",
		InstanceId:  "ins-hermes-002",
		UserID:      user.ID,
		AgentType:   model.AgentTypeHermes,
		RuntimeUser: "agentuser",
	}
	model.DB(ctx).Create(inst)

	// LoadScript 失败 → RunScript 返回错误 → setHermesUI 返回 500
	origLoadScript := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", errors.New("mock: script not found")
	}
	t.Cleanup(func() { LoadScript = origLoadScript })

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	params := map[string]string{"gateway_ui_port": "10815"}

	setHermesUI(rr, req, inst, "1.2.3.4", params)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}
