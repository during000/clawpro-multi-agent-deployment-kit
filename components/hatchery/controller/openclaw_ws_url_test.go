package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

// ─── 测试辅助函数 ──────────────────────────────────────────────────────

func initWSUrlTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Instance{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	origDB := model.UseDBForTest(db)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	origSnap := hcommon.FixedSnapshot
	hcommon.FixedSnapshot = &hcommon.TenantSnapshot{
		Domain: "https://example.com",
	}

	t.Cleanup(func() {
		origDB()
		Store = origStore
		hcommon.FixedSnapshot = origSnap
	})
}

func wsUrlReqWithSession(t *testing.T, method, path, body, username string) *http.Request {
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
	if hcommon.FixedSnapshot != nil {
		ctx := hcommon.InjectTenant(req.Context(), *hcommon.FixedSnapshot)
		req = req.WithContext(ctx)
	}
	return req
}

func saveMockFns(t *testing.T) {
	t.Helper()
	origFetchIP := fetchPrivateIPFn
	origOC := getOpenClawWSInfoFn
	origHermes := getHermesAPIInfoFn
	origSG := checkWSPortAccessFn
	origRunScript := wsUrlRunScriptFn
	origCheckSGIngress := wsUrlCheckSGIngressFn
	t.Cleanup(func() {
		fetchPrivateIPFn = origFetchIP
		getOpenClawWSInfoFn = origOC
		getHermesAPIInfoFn = origHermes
		checkWSPortAccessFn = origSG
		wsUrlRunScriptFn = origRunScript
		wsUrlCheckSGIngressFn = origCheckSGIngress
	})
}

// ─── 测试用例 ──────────────────────────────────────────────────────────

func TestHandleGetWSUrl_MethodNotAllowed(t *testing.T) {
	initWSUrlTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/ws-url", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleGetWSUrl(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleGetWSUrl_Unauthorized(t *testing.T) {
	initWSUrlTestDB(t)

	body := `{"instance_id":"ins-abc12345"}`
	req := httptest.NewRequest(http.MethodPost, "/openclaw/ws-url", strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleGetWSUrl(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleGetWSUrl_InvalidBody(t *testing.T) {
	initWSUrlTestDB(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := wsUrlReqWithSession(t, http.MethodPost, "/openclaw/ws-url", "not json", "u1")
	rr := httptest.NewRecorder()
	HandleGetWSUrl(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 JSON 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleGetWSUrl_MissingInstanceId(t *testing.T) {
	initWSUrlTestDB(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := wsUrlReqWithSession(t, http.MethodPost, "/openclaw/ws-url", `{"instance_id":""}`, "u1")
	rr := httptest.NewRecorder()
	HandleGetWSUrl(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("空 instance_id 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleGetWSUrl_InvalidInstanceIdFormat(t *testing.T) {
	initWSUrlTestDB(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := wsUrlReqWithSession(t, http.MethodPost, "/openclaw/ws-url", `{"instance_id":"bad-format"}`, "u1")
	rr := httptest.NewRecorder()
	HandleGetWSUrl(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("格式错误应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleGetWSUrl_InstanceNotFound(t *testing.T) {
	initWSUrlTestDB(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := wsUrlReqWithSession(t, http.MethodPost, "/openclaw/ws-url", `{"instance_id":"ins-notexist"}`, "u1")
	rr := httptest.NewRecorder()
	HandleGetWSUrl(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("实例不存在应返回 403，实际=%d", rr.Code)
	}
}

func TestHandleGetWSUrl_InstanceNotRunning(t *testing.T) {
	initWSUrlTestDB(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		UserID:          user.ID,
		InstanceId:      "ins-running01",
		LastStableState: "STOPPED",
		AgentType:       model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := wsUrlReqWithSession(t, http.MethodPost, "/openclaw/ws-url", `{"instance_id":"ins-running01"}`, "u1")
	rr := httptest.NewRecorder()
	HandleGetWSUrl(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非 RUNNING 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleGetWSUrl_OpenClaw_PortNotAllocated(t *testing.T) {
	initWSUrlTestDB(t)
	saveMockFns(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	// SiteConfig 不设置 GatewayUIPort（默认 0）
	sc := &model.SiteConfig{}
	model.DB(context.Background()).Create(sc)
	inst := &model.Instance{
		UserID:          user.ID,
		InstanceId:      "ins-noport01",
		LastStableState: "RUNNING",
		AgentType:       model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	fetchPrivateIPFn = func(ctx context.Context, id string) (string, *cvm.Instance, error) {
		return "10.0.0.1", &cvm.Instance{}, nil
	}

	req := wsUrlReqWithSession(t, http.MethodPost, "/openclaw/ws-url", `{"instance_id":"ins-noport01"}`, "u1")
	rr := httptest.NewRecorder()
	HandleGetWSUrl(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("端口未分配应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleGetWSUrl_OpenClaw_Success(t *testing.T) {
	initWSUrlTestDB(t)
	saveMockFns(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	sc := &model.SiteConfig{GatewayUIPort: 18080}
	model.DB(context.Background()).Create(sc)
	inst := &model.Instance{
		UserID:          user.ID,
		InstanceId:      "ins-oc000001",
		LastStableState: "RUNNING",
		AgentType:       model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// Mock fetchPrivateIP
	fetchPrivateIPFn = func(ctx context.Context, id string) (string, *cvm.Instance, error) {
		return "10.0.0.1", &cvm.Instance{}, nil
	}
	// Mock getOpenClawWSInfo
	getOpenClawWSInfoFn = func(ctx context.Context, inst *model.Instance, gatewayUIPort int) (int, string, string, error) {
		return 3000, "test-token-abc", "/abc", nil
	}
	// Mock checkWSPortAccessible (no error)
	checkWSPortAccessFn = func(ctx context.Context, inst *cvm.Instance, port int) error {
		return nil
	}

	req := wsUrlReqWithSession(t, http.MethodPost, "/openclaw/ws-url", `{"instance_id":"ins-oc000001"}`, "u1")
	rr := httptest.NewRecorder()
	HandleGetWSUrl(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	url, _ := resp["url"].(string)
	protocol, _ := resp["protocol"].(string)
	token, _ := resp["token"].(string)
	basePath, _ := resp["basePath"].(string)

	if protocol != "websocket" {
		t.Errorf("protocol 应为 websocket，实际=%s", protocol)
	}
	if token != "test-token-abc" {
		t.Errorf("token 应为 test-token-abc，实际=%s", token)
	}
	if basePath != "/abc" {
		t.Errorf("basePath 应为 /abc，实际=%s", basePath)
	}
	if !strings.Contains(url, "ws://10.0.0.1:3000/ws?token=test-token-abc") {
		t.Errorf("url 格式异常: %s", url)
	}
}

func TestHandleGetWSUrl_Hermes_Success(t *testing.T) {
	initWSUrlTestDB(t)
	saveMockFns(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		UserID:          user.ID,
		InstanceId:      "ins-hermes01",
		LastStableState: "RUNNING",
		AgentType:       model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	fetchPrivateIPFn = func(ctx context.Context, id string) (string, *cvm.Instance, error) {
		return "10.0.0.2", &cvm.Instance{}, nil
	}
	getHermesAPIInfoFn = func(ctx context.Context, inst *model.Instance) (int, string, error) {
		return 8080, "hermes-key-xyz", nil
	}
	checkWSPortAccessFn = func(ctx context.Context, inst *cvm.Instance, port int) error {
		return nil
	}

	req := wsUrlReqWithSession(t, http.MethodPost, "/openclaw/ws-url", `{"instance_id":"ins-hermes01"}`, "u1")
	rr := httptest.NewRecorder()
	HandleGetWSUrl(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	protocol, _ := resp["protocol"].(string)
	token, _ := resp["token"].(string)
	url, _ := resp["url"].(string)

	if protocol != "sse" {
		t.Errorf("protocol 应为 sse，实际=%s", protocol)
	}
	if token != "hermes-key-xyz" {
		t.Errorf("token 应为 hermes-key-xyz，实际=%s", token)
	}
	if !strings.Contains(url, "http://10.0.0.2:8080") {
		t.Errorf("url 格式异常: %s", url)
	}
}

func TestHandleGetWSUrl_UnsupportedAgentType(t *testing.T) {
	initWSUrlTestDB(t)
	saveMockFns(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		UserID:          user.ID,
		InstanceId:      "ins-unknown01",
		LastStableState: "RUNNING",
		AgentType:       "some_unknown_type",
	}
	model.DB(context.Background()).Create(inst)

	fetchPrivateIPFn = func(ctx context.Context, id string) (string, *cvm.Instance, error) {
		return "10.0.0.3", &cvm.Instance{}, nil
	}

	req := wsUrlReqWithSession(t, http.MethodPost, "/openclaw/ws-url", `{"instance_id":"ins-unknown01"}`, "u1")
	rr := httptest.NewRecorder()
	HandleGetWSUrl(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("不支持的类型应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleGetWSUrl_FetchPrivateIPError(t *testing.T) {
	initWSUrlTestDB(t)
	saveMockFns(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		UserID:          user.ID,
		InstanceId:      "ins-ipfail01",
		LastStableState: "RUNNING",
		AgentType:       model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	fetchPrivateIPFn = func(ctx context.Context, id string) (string, *cvm.Instance, error) {
		return "", nil, hcommon.I18nError(i18n.MsgQueryCVMInstanceFailed)
	}

	req := wsUrlReqWithSession(t, http.MethodPost, "/openclaw/ws-url", `{"instance_id":"ins-ipfail01"}`, "u1")
	rr := httptest.NewRecorder()
	HandleGetWSUrl(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("fetchPrivateIP 失败应返回 500，实际=%d", rr.Code)
	}
}

func TestHandleGetWSUrl_OpenClaw_WSInfoError(t *testing.T) {
	initWSUrlTestDB(t)
	saveMockFns(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	sc := &model.SiteConfig{GatewayUIPort: 18080}
	model.DB(context.Background()).Create(sc)
	inst := &model.Instance{
		UserID:          user.ID,
		InstanceId:      "ins-wsfail01",
		LastStableState: "RUNNING",
		AgentType:       model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	fetchPrivateIPFn = func(ctx context.Context, id string) (string, *cvm.Instance, error) {
		return "10.0.0.4", &cvm.Instance{}, nil
	}
	getOpenClawWSInfoFn = func(ctx context.Context, inst *model.Instance, gatewayUIPort int) (int, string, string, error) {
		return 0, "", "", hcommon.I18nError(i18n.MsgTATTimeout)
	}

	req := wsUrlReqWithSession(t, http.MethodPost, "/openclaw/ws-url", `{"instance_id":"ins-wsfail01"}`, "u1")
	rr := httptest.NewRecorder()
	HandleGetWSUrl(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("getOpenClawWSInfo 失败应返回 500，实际=%d", rr.Code)
	}
}

func TestHandleGetWSUrl_Hermes_InfoError(t *testing.T) {
	initWSUrlTestDB(t)
	saveMockFns(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		UserID:          user.ID,
		InstanceId:      "ins-hfail001",
		LastStableState: "RUNNING",
		AgentType:       model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	fetchPrivateIPFn = func(ctx context.Context, id string) (string, *cvm.Instance, error) {
		return "10.0.0.5", &cvm.Instance{}, nil
	}
	getHermesAPIInfoFn = func(ctx context.Context, inst *model.Instance) (int, string, error) {
		return 0, "", hcommon.I18nError(i18n.MsgOperationFailed)
	}

	req := wsUrlReqWithSession(t, http.MethodPost, "/openclaw/ws-url", `{"instance_id":"ins-hfail001"}`, "u1")
	rr := httptest.NewRecorder()
	HandleGetWSUrl(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("getHermesAPIInfo 失败应返回 500，实际=%d", rr.Code)
	}
}

// ─── 底层函数直接测试 ──────────────────────────────────────────────────

func TestCheckWSPortAccessible_NoSecurityGroups(t *testing.T) {
	ctx := context.Background()
	inst := &cvm.Instance{SecurityGroupIds: nil}
	err := checkWSPortAccessible(ctx, inst, 3000)
	if err == nil {
		t.Error("无安全组应返回错误")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(ctx, err), "未绑定任何安全组") {
		t.Errorf("错误信息不符: %v", err)
	}
}

func TestCheckWSPortAccessible_PortAccessible(t *testing.T) {
	saveMockFns(t)

	sgId := "sg-test001"
	inst := &cvm.Instance{SecurityGroupIds: []*string{&sgId}}

	wsUrlCheckSGIngressFn = func(ctx context.Context, sg string, port int) (bool, error) {
		return true, nil
	}

	err := checkWSPortAccessible(context.Background(), inst, 3000)
	if err != nil {
		t.Errorf("端口放通应返回 nil，实际=%v", err)
	}
}

func TestCheckWSPortAccessible_PortNotAccessible(t *testing.T) {
	saveMockFns(t)

	sgId := "sg-test002"
	inst := &cvm.Instance{SecurityGroupIds: []*string{&sgId}}

	wsUrlCheckSGIngressFn = func(ctx context.Context, sg string, port int) (bool, error) {
		return false, nil
	}

	err := checkWSPortAccessible(context.Background(), inst, 3000)
	if err == nil {
		t.Error("端口未放通应返回错误")
	}
}

func TestGetOpenClawWSInfo_Success(t *testing.T) {
	saveMockFns(t)

	wsUrlRunScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return `{"port":3000,"authToken":"abc123","basePath":"/abc"}`, nil
	}

	inst := &model.Instance{InstanceId: "ins-test01"}
	port, token, basePath, err := getOpenClawWSInfo(context.Background(), inst, 12345)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if port != 3000 || token != "abc123" || basePath != "/abc" {
		t.Errorf("返回值异常 port=%d token=%s basePath=%s", port, token, basePath)
	}
}

func TestGetOpenClawWSInfo_ScriptError(t *testing.T) {
	saveMockFns(t)

	wsUrlRunScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATTimeout)
	}

	inst := &model.Instance{InstanceId: "ins-test02"}
	_, _, _, err := getOpenClawWSInfo(context.Background(), inst, 12345)
	if err == nil {
		t.Error("脚本失败应返回错误")
	}
}

func TestGetOpenClawWSInfo_InvalidJSON(t *testing.T) {
	saveMockFns(t)

	wsUrlRunScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "not json at all", nil
	}

	inst := &model.Instance{InstanceId: "ins-test03"}
	_, _, _, err := getOpenClawWSInfo(context.Background(), inst, 12345)
	if err == nil {
		t.Error("非法 JSON 应返回错误")
	}
}

func TestGetOpenClawWSInfo_ScriptReturnError(t *testing.T) {
	saveMockFns(t)

	wsUrlRunScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return `{"error":"gateway not running"}`, nil
	}

	inst := &model.Instance{InstanceId: "ins-test04"}
	_, _, _, err := getOpenClawWSInfo(context.Background(), inst, 12345)
	if err == nil {
		t.Error("脚本返回 error 字段应报错")
	}
}

func TestGetOpenClawWSInfo_EmptyPortOrToken(t *testing.T) {
	saveMockFns(t)

	wsUrlRunScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return `{"port":0,"authToken":""}`, nil
	}

	inst := &model.Instance{InstanceId: "ins-test05"}
	_, _, _, err := getOpenClawWSInfo(context.Background(), inst, 12345)
	if err == nil {
		t.Error("port=0 且 token 空应报错")
	}
}

func TestGetHermesAPIInfo_Success(t *testing.T) {
	saveMockFns(t)

	wsUrlRunScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return `{"port":8080,"key":"hermes-key"}`, nil
	}

	inst := &model.Instance{InstanceId: "ins-hermes01"}
	port, key, err := getHermesAPIInfo(context.Background(), inst)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if port != 8080 || key != "hermes-key" {
		t.Errorf("返回值异常 port=%d key=%s", port, key)
	}
}

func TestGetHermesAPIInfo_ScriptError(t *testing.T) {
	saveMockFns(t)

	wsUrlRunScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATFailed)
	}

	inst := &model.Instance{InstanceId: "ins-hermes02"}
	_, _, err := getHermesAPIInfo(context.Background(), inst)
	if err == nil {
		t.Error("脚本失败应返回错误")
	}
}

func TestGetHermesAPIInfo_InvalidJSON(t *testing.T) {
	saveMockFns(t)

	wsUrlRunScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "garbage", nil
	}

	inst := &model.Instance{InstanceId: "ins-hermes03"}
	_, _, err := getHermesAPIInfo(context.Background(), inst)
	if err == nil {
		t.Error("非法 JSON 应返回错误")
	}
}

func TestGetHermesAPIInfo_EmptyPortOrKey(t *testing.T) {
	saveMockFns(t)

	wsUrlRunScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return `{"port":0,"key":""}`, nil
	}

	inst := &model.Instance{InstanceId: "ins-hermes04"}
	_, _, err := getHermesAPIInfo(context.Background(), inst)
	if err == nil {
		t.Error("port=0 且 key 空应报错")
	}
}
