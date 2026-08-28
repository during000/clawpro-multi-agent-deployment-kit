package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ── 测试辅助 ─────────────────────────────────────────────────────────────────

// initLightClawTestDBWithSession 初始化内存 SQLite 数据库和 session store。
func initLightClawTestDBWithSession(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}, &model.AuditLog{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	origSnap := common.FixedSnapshot
	// 确保 FixedSnapshot 有 Domain
	newSnap := &common.TenantSnapshot{}
	if origSnap != nil {
		newSnap.Identifier = origSnap.Identifier
		newSnap.Uin = origSnap.Uin
		newSnap.InternalSecret = origSnap.InternalSecret
		newSnap.Domain = origSnap.Domain
		if newSnap.Domain == "" {
			newSnap.Domain = "https://example.com" // 默认值
		}
	} else {
		newSnap.Domain = "https://example.com"
	}
	common.FixedSnapshot = newSnap
	return db, func() {
		// HandleLightClawToken 会异步写审计日志，清理前等待 goroutine 使用完测试 DB。
		time.Sleep(10 * time.Millisecond)
		common.FixedSnapshot = origSnap
		origDB()
		Store = origStore
	}
}

type mockLightClawHandlerDependencies struct {
	runScript func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error)
}

func (m *mockLightClawHandlerDependencies) RunScript(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
	if m.runScript != nil {
		return m.runScript(ctx, instanceId, scriptName, timeout, runtimeUser, onOutput, params)
	}
	return "", nil
}

func userLightClawReqWithSession(t *testing.T, method, path, username string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Accept", "application/json")
	session, err := Store.Get(req, "hatchery-session")
	if err != nil {
		t.Fatalf("获取 session 失败: %v", err)
	}
	session.Values["username"] = username
	rr := httptest.NewRecorder()
	if err := session.Save(req, rr); err != nil {
		t.Fatalf("保存 session 失败: %v", err)
	}
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	// 注入租户 snapshot 到上下文（模拟 IdentifierMiddleware 的行为）
	if common.FixedSnapshot != nil {
		ctx := common.InjectTenant(req.Context(), *common.FixedSnapshot)
		req = req.WithContext(ctx)
	}

	return req
}

// ── 测试用例 ─────────────────────────────────────────────────────────────────

// mockRunScriptOK 模拟 RunScript 成功。
func mockRunScriptOK(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
	return "", nil
}

// mockRunScriptFail 模拟 RunScript 失败。
func mockRunScriptFail(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
	return "", errors.New("approve device 超时")
}

func TestLightClawToken_MethodNotAllowed(t *testing.T) {
	_, cleanup := initLightClawTestDBWithSession(t)
	defer cleanup()
	origSnap := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{
		Identifier:     origSnap.Identifier,
		Domain:         "https://example.com",
		Uin:            origSnap.Uin,
		InternalSecret: origSnap.InternalSecret,
	}
	defer func() { common.FixedSnapshot = origSnap }()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/lightclaw/token?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h := newLightClawHandler(&mockLightClawHandlerDependencies{})
	h.handleToken(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

func TestLightClawToken_DomainNotConfigured(t *testing.T) {
	db, cleanup := initLightClawTestDBWithSession(t)
	defer cleanup()
	origSnap := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{
		Identifier:     origSnap.Identifier,
		Domain:         "",
		Uin:            origSnap.Uin,
		InternalSecret: origSnap.InternalSecret,
	}
	defer func() { common.FixedSnapshot = origSnap }()

	user := &model.User{Username: "testuser", Password: "x", Role: "user"}
	db.Create(user)
	req := userLightClawReqWithSession(t, http.MethodGet, "/openclaw/lightclaw/token?id=1", "testuser")
	w := httptest.NewRecorder()
	h := newLightClawHandler(&mockLightClawHandlerDependencies{})
	h.handleToken(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == nil {
		t.Error("期望响应包含 error 字段")
	}
}

func TestLightClawToken_UserNil(t *testing.T) {
	_, cleanup := initLightClawTestDBWithSession(t)
	defer cleanup()
	origSnap := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{
		Identifier:     origSnap.Identifier,
		Domain:         "https://example.com",
		Uin:            origSnap.Uin,
		InternalSecret: origSnap.InternalSecret,
	}
	defer func() { common.FixedSnapshot = origSnap }()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/lightclaw/token?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h := newLightClawHandler(&mockLightClawHandlerDependencies{})
	h.handleToken(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，实际=%d", w.Code)
	}
}

func TestLightClawToken_InstanceNotFound(t *testing.T) {
	db, cleanup := initLightClawTestDBWithSession(t)
	defer cleanup()
	origSnap := common.FixedSnapshot
	newSnap := &common.TenantSnapshot{Domain: "https://example.com"}
	if origSnap != nil {
		newSnap.Identifier = origSnap.Identifier
		newSnap.Uin = origSnap.Uin
		newSnap.InternalSecret = origSnap.InternalSecret
	}
	common.FixedSnapshot = newSnap
	defer func() { common.FixedSnapshot = origSnap }()

	user := &model.User{Username: "testuser", Password: "x", Role: "user"}
	db.Create(user)
	req := userLightClawReqWithSession(t, http.MethodGet, "/openclaw/lightclaw/token?id=999", "testuser")
	w := httptest.NewRecorder()
	h := newLightClawHandler(&mockLightClawHandlerDependencies{})
	h.handleToken(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望 404，实际=%d", w.Code)
	}
}

func TestLightClawToken_ProxyTokenNil(t *testing.T) {
	db, cleanup := initLightClawTestDBWithSession(t)
	defer cleanup()
	origSnap := common.FixedSnapshot
	newSnap := &common.TenantSnapshot{Domain: "https://example.com"}
	if origSnap != nil {
		newSnap.Identifier = origSnap.Identifier
		newSnap.Uin = origSnap.Uin
		newSnap.InternalSecret = origSnap.InternalSecret
	}
	common.FixedSnapshot = newSnap
	defer func() { common.FixedSnapshot = origSnap }()

	user := &model.User{Username: "testuser", Password: "x", Role: "user"}
	db.Create(user)
	instance := &model.Instance{Name: "test-instance", InstanceId: "ins-abc123", UserID: user.ID, ProxyToken: nil}
	db.Create(instance)

	req := userLightClawReqWithSession(t, http.MethodGet, fmt.Sprintf("/openclaw/lightclaw/token?id=%d", instance.ID), "testuser")
	w := httptest.NewRecorder()
	h := newLightClawHandler(&mockLightClawHandlerDependencies{})
	h.handleToken(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	wanted := common.I18nError(i18n.MsgInstanceProxyTokenNotConfigured).ErrorMessage(req.Context())
	errMsg, _ := resp["error"].(string)
	if errMsg != wanted {
		t.Errorf("期望错误信息为 '实例 ProxyToken 未配置'，实际=%s", errMsg)
	}
}

func TestLightClawToken_ProxyTokenEmpty(t *testing.T) {
	db, cleanup := initLightClawTestDBWithSession(t)
	defer cleanup()
	origSnap := common.FixedSnapshot
	newSnap := &common.TenantSnapshot{Domain: "https://example.com"}
	if origSnap != nil {
		newSnap.Identifier = origSnap.Identifier
		newSnap.Uin = origSnap.Uin
		newSnap.InternalSecret = origSnap.InternalSecret
	}
	common.FixedSnapshot = newSnap
	defer func() { common.FixedSnapshot = origSnap }()

	user := &model.User{Username: "testuser", Password: "x", Role: "user"}
	db.Create(user)
	emptyToken := ""
	instance := &model.Instance{Name: "test-instance", InstanceId: "ins-abc123", UserID: user.ID, ProxyToken: &emptyToken}
	db.Create(instance)

	req := userLightClawReqWithSession(t, http.MethodGet, fmt.Sprintf("/openclaw/lightclaw/token?id=%d", instance.ID), "testuser")
	w := httptest.NewRecorder()
	h := newLightClawHandler(&mockLightClawHandlerDependencies{})
	h.handleToken(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", w.Code)
	}
}

func TestLightClawToken_ApproveDeviceFailed(t *testing.T) {
	db, cleanup := initLightClawTestDBWithSession(t)
	defer cleanup()
	origSnap := common.FixedSnapshot
	newSnap := &common.TenantSnapshot{Domain: "https://example.com"}
	if origSnap != nil {
		newSnap.Identifier = origSnap.Identifier
		newSnap.Uin = origSnap.Uin
		newSnap.InternalSecret = origSnap.InternalSecret
	}
	common.FixedSnapshot = newSnap
	defer func() { common.FixedSnapshot = origSnap }()

	// mock runScriptFn：让 waitForOpenclawReady 内的 check_openclaw_ready.sh 立即返回 ready=true，
	// 避免真实等待 5 分钟超时。approve_device.sh 由 deps.RunScript（mockRunScriptFail）处理。
	origRunScriptFn := runScriptFn
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64,
		runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		if scriptName == "check_openclaw_ready.sh" {
			return `{"ready": true}`, nil
		}
		return "", common.I18nError(i18n.MsgUnexpectedScript, scriptName)
	}
	defer func() { runScriptFn = origRunScriptFn }()

	user := &model.User{Username: "testuser", Password: "x", Role: "user"}
	db.Create(user)
	token := "sk-test-token-123"
	instance := &model.Instance{
		Name:        "test-instance",
		InstanceId:  "ins-abc123",
		UserID:      user.ID,
		ProxyToken:  &token,
		RuntimeUser: "ubuntu",
	}
	db.Create(instance)

	req := userLightClawReqWithSession(t, http.MethodGet, fmt.Sprintf("/openclaw/lightclaw/token?id=%d", instance.ID), "testuser")
	w := httptest.NewRecorder()
	h := newLightClawHandler(&mockLightClawHandlerDependencies{runScript: mockRunScriptFail})
	h.handleToken(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	errMsg, _ := resp["error"].(string)
	if errMsg != "获取鉴权失败，请刷新页面重试" {
		t.Errorf("期望错误信息为 '获取鉴权失败，请刷新页面重试'，实际=%s", errMsg)
	}
}

func TestLightClawToken_Success(t *testing.T) {
	db, cleanup := initLightClawTestDBWithSession(t)
	defer cleanup()
	origSnap := common.FixedSnapshot
	newSnap := &common.TenantSnapshot{Domain: "https://x8swfkbg.tcaisite.com"}
	if origSnap != nil {
		newSnap.Identifier = origSnap.Identifier
		newSnap.Uin = origSnap.Uin
		newSnap.InternalSecret = origSnap.InternalSecret
	}
	common.FixedSnapshot = newSnap
	defer func() { common.FixedSnapshot = origSnap }()

	user := &model.User{Username: "testuser", Password: "x", Role: "user"}
	db.Create(user)
	token := "sk-test-token-456"
	instance := &model.Instance{
		Name:        "test-instance",
		InstanceId:  "ins-xyz789",
		UserID:      user.ID,
		ProxyToken:  &token,
		RuntimeUser: "ubuntu",
	}
	db.Create(instance)

	req := userLightClawReqWithSession(t, http.MethodGet, fmt.Sprintf("/openclaw/lightclaw/token?id=%d", instance.ID), "testuser")
	w := httptest.NewRecorder()

	// mock runScriptFn：让 waitForOpenclawReady 内的 check_openclaw_ready.sh 立即返回 ready=true，
	// 避免调用真实 TAT 脚本导致测试卡住。
	origRunScriptFn := runScriptFn
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64,
		runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		if scriptName == "check_openclaw_ready.sh" {
			return `{"ready": true}`, nil
		}
		return "", common.I18nError(i18n.MsgUnexpectedScript, scriptName)
	}
	defer func() { runScriptFn = origRunScriptFn }()

	// 记录 RunScript 被调用时的参数
	var calledInstanceId, calledScript, calledRuntimeUser string
	var calledTimeout uint64
	mockRunScript := func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		calledInstanceId = instanceId
		calledScript = scriptName
		calledTimeout = timeout
		calledRuntimeUser = runtimeUser
		return "", nil
	}

	h := newLightClawHandler(&mockLightClawHandlerDependencies{runScript: mockRunScript})
	h.handleToken(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d，响应: %s", w.Code, w.Body.String())
	}

	// 校验 RunScript 调用参数
	if calledInstanceId != "ins-xyz789" {
		t.Errorf("RunScript instanceId 期望 ins-xyz789，实际=%s", calledInstanceId)
	}
	if calledScript != "approve_device.sh" {
		t.Errorf("RunScript scriptName 期望 approve_device.sh，实际=%s", calledScript)
	}
	if calledTimeout != 300 {
		t.Errorf("RunScript timeout 期望 300，实际=%d", calledTimeout)
	}
	if calledRuntimeUser != "ubuntu" {
		t.Errorf("RunScript runtimeUser 期望 ubuntu，实际=%s", calledRuntimeUser)
	}

	// 解析响应
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 校验 proxyToken
	if resp["proxyToken"] != "sk-test-token-456" {
		t.Errorf("期望 proxyToken=sk-test-token-456，实际=%v", resp["proxyToken"])
	}

	// 校验 callbackUrl
	expectedCallback := "https://x8swfkbg.tcaisite.com/api/openclaw/lightclaw/auth"
	if resp["callbackUrl"] != expectedCallback {
		t.Errorf("期望 callbackUrl=%s，实际=%v", expectedCallback, resp["callbackUrl"])
	}

	// 校验 businessCode
	if resp["businessCode"] != lightClawProductCode {
		t.Errorf("期望 businessCode=%s，实际=%v", lightClawProductCode, resp["businessCode"])
	}

	// 校验 timestamp 存在且为数字
	ts, ok := resp["timestamp"].(float64)
	if !ok || ts <= 0 {
		t.Errorf("期望 timestamp 为正数，实际=%v", resp["timestamp"])
	}

	// 校验 sign 存在且非空
	sign, ok := resp["sign"].(string)
	if !ok || sign == "" {
		t.Errorf("期望 sign 非空，实际=%v", resp["sign"])
	}
}

func TestLightClawToken_SignatureConsistency(t *testing.T) {
	// 验证 generateSign 对相同输入产生相同签名
	params := map[string]string{
		"businessCode": lightClawProductCode,
		"timestamp":    "1700000000",
		"callbackUrl":  "https://example.com/api/openclaw/lightclaw/auth",
		"proxyToken":   "sk-test-token",
	}
	sign1 := generateSign(params)
	sign2 := generateSign(params)
	if sign1 != sign2 {
		t.Errorf("相同参数签名不一致: %s vs %s", sign1, sign2)
	}
	if sign1 == "" {
		t.Error("签名不应为空")
	}
	// 签名应为 64 位 hex（SHA256）
	if len(sign1) != 64 {
		t.Errorf("签名长度期望 64，实际=%d", len(sign1))
	}
}

func TestLightClawToken_SignatureChangesWithParams(t *testing.T) {
	// 验证不同参数产生不同签名
	params1 := map[string]string{
		"businessCode": lightClawProductCode,
		"timestamp":    "1700000000",
		"callbackUrl":  "https://example.com/api/openclaw/lightclaw/auth",
		"proxyToken":   "sk-token-a",
	}
	params2 := map[string]string{
		"businessCode": lightClawProductCode,
		"timestamp":    "1700000000",
		"callbackUrl":  "https://example.com/api/openclaw/lightclaw/auth",
		"proxyToken":   "sk-token-b",
	}
	sign1 := generateSign(params1)
	sign2 := generateSign(params2)
	if sign1 == sign2 {
		t.Error("不同 proxyToken 应产生不同签名")
	}
}

func TestLightClawUserID(t *testing.T) {
	tests := []struct {
		domain   string
		userID   uint
		expected string
	}{
		{"https://x8swfkbg.tcaisite.com", 3, "x8swfkbg-3"},
		{"http://myhost.example.com", 10, "myhost-10"},
		{"https://single", 1, "single-1"},
	}

	for _, tt := range tests {
		ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Domain: tt.domain})
		got := lightClawUserID(ctx, tt.userID)
		if got != tt.expected {
			t.Errorf("lightClawUserID(%d) with Domain=%s: 期望=%s，实际=%s", tt.userID, tt.domain, tt.expected, got)
		}
	}
}

func TestNewTid(t *testing.T) {
	tid1 := newTid()
	tid2 := newTid()
	if tid1 == tid2 {
		t.Error("两次 newTid 不应相同")
	}
	if len(tid1) != 32 {
		t.Errorf("tid 长度期望 32，实际=%d", len(tid1))
	}
}

func TestLightClawOK(t *testing.T) {
	w := httptest.NewRecorder()
	lightClawOK(w, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("期望 Content-Type=application/json，实际=%s", ct)
	}

	var resp lightClawResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != float64(0) { // JSON 数字解码为 float64
		t.Errorf("期望 code=0，实际=%v", resp.Code)
	}
	if resp.Message != "OK" {
		t.Errorf("期望 message=OK，实际=%s", resp.Message)
	}
	if resp.Error != nil {
		t.Errorf("期望 error=nil，实际=%v", resp.Error)
	}
	if resp.Tid == "" {
		t.Error("期望 tid 非空")
	}
	if resp.Timestamp == "" {
		t.Error("期望 timestamp 非空")
	}
}

func TestLightClawError(t *testing.T) {
	w := httptest.NewRecorder()
	lightClawError(w, "InvalidParameterValue", "参数错误", nil)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200（lightClawError 不设置 HTTP 状态码），实际=%d", w.Code)
	}

	var resp lightClawResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != "InvalidParameterValue" {
		t.Errorf("期望 code=InvalidParameterValue，实际=%v", resp.Code)
	}
	if resp.Message != "参数错误" {
		t.Errorf("期望 message=参数错误，实际=%s", resp.Message)
	}
	if resp.Error == nil || *resp.Error != "InvalidParameterValue" {
		t.Errorf("期望 error=InvalidParameterValue，实际=%v", resp.Error)
	}
}
