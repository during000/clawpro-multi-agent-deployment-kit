package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// mockLightClawDepsForToken 构造一个 lightClawHandlerDependencies mock 实现，
// 让 deps.RunScript 把 scriptName 转发给传入的 fn，便于在 handleToken 单测中精确控制
// approve_device.sh 的返回值，避免依赖真实 LoadScript / TAT。
func mockLightClawDepsForToken(fn func(scriptName string) (string, error)) lightClawHandlerDependencies {
	return lightClawDepsFunc(fn)
}

type lightClawDepsFunc func(scriptName string) (string, error)

func (f lightClawDepsFunc) RunScript(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
	return f(scriptName)
}

// waitLCGoroutines 等待 HandleLightClawToken 内的 go LogAudit 写完 DB 再 cleanup。
func waitLCGoroutines() {
	time.Sleep(150 * time.Millisecond)
}

// initLightClawHandlerTestDB 初始化 LightClaw handler 测试 DB。
func initLightClawHandlerTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{}, &model.Instance{}, &model.AuditLog{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	origSnap := common.FixedSnapshot
	newSnap := &common.TenantSnapshot{
		Domain: "https://example.com",
	}
	if origSnap != nil {
		newSnap.Identifier = origSnap.Identifier
		newSnap.Uin = origSnap.Uin
		newSnap.InternalSecret = origSnap.InternalSecret
	}
	common.FixedSnapshot = newSnap

	t.Cleanup(func() {
		origDB()
		Store = origStore
		common.FixedSnapshot = origSnap
	})
	return func() {
		origDB()
		Store = origStore
		common.FixedSnapshot = origSnap
	}
}

// lightClawReqWithSession 构造带 session 的请求。
func lightClawReqWithSession(t *testing.T, method, path, username string) *http.Request {
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

	// 注入租户 snapshot 到上下文（模拟 IdentifierMiddleware 的行为）
	if common.FixedSnapshot != nil {
		ctx := common.InjectTenant(req.Context(), *common.FixedSnapshot)
		req = req.WithContext(ctx)
	}

	return req
}

// ─── HandleLightClawToken Handler 级别测试 ────────────────────────────

func TestHandleLightClawToken_Unauthorized(t *testing.T) {
	initLightClawHandlerTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/lightclaw/token?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleLightClawToken(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleLightClawToken_MethodNotAllowed(t *testing.T) {
	initLightClawHandlerTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/openclaw/lightclaw/token?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleLightClawToken(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleLightClawToken_DomainEmpty(t *testing.T) {
	initLightClawHandlerTestDB(t)

	origDomain := common.FixedSnapshot.Domain
	common.FixedSnapshot.Domain = "" // 清空
	t.Cleanup(func() { common.FixedSnapshot.Domain = origDomain })

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := lightClawReqWithSession(t, http.MethodGet, "/openclaw/lightclaw/token?id=1", "u1")
	rr := httptest.NewRecorder()
	HandleLightClawToken(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Domain 空应返回 500，实际=%d", rr.Code)
	}
}

func TestHandleLightClawToken_InstanceNotFound(t *testing.T) {
	initLightClawHandlerTestDB(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := lightClawReqWithSession(t, http.MethodGet, "/openclaw/lightclaw/token?id=999", "u1")
	rr := httptest.NewRecorder()
	HandleLightClawToken(rr, req)
	if rr.Code != http.StatusNotFound && rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应返回 404/400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleLightClawToken_ProxyTokenNil(t *testing.T) {
	initLightClawHandlerTestDB(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-nt",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		ProxyToken: nil,
	}
	model.DB(context.Background()).Create(inst)

	req := lightClawReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/lightclaw/token?id=%d", inst.ID), "u1")
	rr := httptest.NewRecorder()
	HandleLightClawToken(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ProxyToken 为 nil 应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleLightClawToken_HermesSkipsApproveScript 覆盖源码 148-151/157：
// Hermes/ACE 不支持 approve，跳过 approve_device.sh 的执行。
// 关键：即使 LoadScript mock 返回 err，hermes 也能通过（因为根本不调用 approve_device.sh）。
// 最终会因为 Domain 配好 + ProxyToken 有值 → 返回 200。
func TestHandleLightClawToken_HermesSkipsApproveScript(t *testing.T) {
	initLightClawHandlerTestDB(t)

	// 故意让 LoadScript 失败；若 hermes 分支正确跳过 approve，就不会触发此失败
	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("should-not-be-called-for-hermes")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u-hermes", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	proxyToken := "pt-hermes"
	inst := &model.Instance{
		Name: "hermes-inst", InstanceId: "ins-hermes-token",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(inst)

	req := lightClawReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/lightclaw/token?id=%d", inst.ID), "u-hermes")
	rr := httptest.NewRecorder()
	HandleLightClawToken(rr, req)

	// Hermes 走 skip 分支 → 返回 200
	if rr.Code != http.StatusOK {
		t.Errorf("Hermes 应跳过 approve 返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	waitLCGoroutines()
}

// TestHandleLightClawToken_AceSkipsApproveScript 类似 Hermes 场景。
func TestHandleLightClawToken_AceSkipsApproveScript(t *testing.T) {
	initLightClawHandlerTestDB(t)

	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("should-not-be-called-for-ace")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u-ace", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	proxyToken := "pt-ace"
	inst := &model.Instance{
		Name: "ace-inst", InstanceId: "ins-ace-token",
		UserID: user.ID, AgentType: model.AgentTypeLightclawACE,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(inst)

	req := lightClawReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/lightclaw/token?id=%d", inst.ID), "u-ace")
	rr := httptest.NewRecorder()
	HandleLightClawToken(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ACE 应跳过 approve 返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	waitLCGoroutines()
}

// TestHandleLightClawToken_OpenClawRunsApproveScript 覆盖 openclaw 分支：
// 会调用 approve_device.sh，此处用 mock 返回 err → 500。
//
// 注意（hotfix/approve_device_bug_fix）：v0.x.x 起 handleToken 在执行 approve_device.sh
// 之前会先调 waitForOpenclawReady（5 分钟），后者通过 runScriptFn(check_ready) 探测就绪。
// 测试里需 mock 包级 runScriptFn 让 wait 立即就绪，避免单测真的等 5 分钟。
// 本用例覆盖：wait 成功 → approve_device.sh 失败（LoadScript mock）→ 500。
func TestHandleLightClawToken_OpenClawRunsApproveScript(t *testing.T) {
	initLightClawHandlerTestDB(t)

	// 让 waitForOpenclawReady 内的 check_ready 立即返回 ready=true
	// 注意：ResolveScript("check_ready", AgentTypeOpenClaw) 解析后的真实脚本名是
	// "check_openclaw_ready.sh"，必须显式匹配该名字，否则 mock 不命中会让 wait 循环跑满
	// 5 分钟超时（导致 go test 包级 timeout 被触发）。
	origRunScriptFn := runScriptFn
	runScriptFn = func(ctx context.Context, instanceId, scriptName string,
		timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		if scriptName == "check_openclaw_ready.sh" {
			return `{"ready": true}`, nil
		}
		return "", common.I18nError(i18n.MsgUnexpectedScript, scriptName)
	}
	defer func() { runScriptFn = origRunScriptFn }()

	// approve_device.sh 走 deps.RunScript → 真实 RunScript → LoadScript 失败 → 返回 err → 500
	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: approve_device.sh failed")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u-oc", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	proxyToken := "pt-oc"
	inst := &model.Instance{
		Name: "oc-inst", InstanceId: "ins-oc-token",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(inst)

	req := lightClawReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/lightclaw/token?id=%d", inst.ID), "u-oc")
	rr := httptest.NewRecorder()
	HandleLightClawToken(rr, req)

	// OpenClaw wait 通过 → approve LoadScript 失败 → 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("OpenClaw approve 失败应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleLightClawToken_WaitOpenclawReadyTimeout 覆盖
// hotfix/approve_device_bug_fix 新增的 wait 失败分支：
// 当 waitForOpenclawReady 在 5 分钟内未达就绪条件时，handleToken 直接返回 500 +
// "实例服务尚未就绪" 文案，且不会再调用 approve_device.sh（保护 gateway 不被无谓打扰）。
//
// 实现要点：在 runScriptFn mock 中，check_openclaw_ready.sh 第一次执行时 cancel ctx，
// 让 attempt=2 进入 select 后立即 ctx.Done 退出。注意不能预先 cancel ctx，否则
// model.DB(ctx) 查询用户会失败 → 401，必须等 DB 查询完成后再 cancel。
func TestHandleLightClawToken_WaitOpenclawReadyTimeout(t *testing.T) {
	initLightClawHandlerTestDB(t)

	// 关键：不预先 cancel ctx（否则 model.DB(ctx) 查询用户会因 ctx 已取消而失败 → 401）。
	// 正确做法：先构造请求（DB 查询所需的 ctx 正常），然后从请求 ctx 派生一个可 cancel 的子 ctx，
	// 在 runScriptFn mock 中第一次调用 check_openclaw_ready.sh 时 cancel 该子 ctx，
	// 这样 attempt=2 的 select 立即触发 ctx.Done() → waitForOpenclawReady 返回 error → 500。
	user := &model.User{Username: "u-wait", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	proxyToken := "pt-wait"
	inst := &model.Instance{
		Name: "wait-inst", InstanceId: "ins-wait-token",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(inst)

	// 先构造带 session 的请求（此时 ctx 正常，保证后续 DB 查询成功）
	req := lightClawReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/lightclaw/token?id=%d", inst.ID), "u-wait")

	// 从请求 ctx 派生可 cancel 的子 ctx，保留 TenantSnapshot 和 session cookie
	cancelCtx, cancelFn := context.WithCancel(req.Context())
	defer cancelFn()
	req = req.WithContext(cancelCtx)

	origRunScriptFn := runScriptFn
	var approveCalled bool
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64,
		runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		if scriptName == "check_openclaw_ready.sh" {
			// 在脚本执行时 cancel ctx：此时 DB 查询已完成，cancel 只影响 waitForOpenclawReady 的 select
			cancelFn()
			return `{"ready": false, "reason": "gateway_not_active"}`, nil
		}
		if scriptName == "approve_device.sh" {
			approveCalled = true
		}
		return "", common.I18nError(i18n.MsgUnexpectedScript, scriptName)
	}
	defer func() { runScriptFn = origRunScriptFn }()

	// 即便 wait 失败前 deps.RunScript 不会被调，也保险地 mock LoadScript 防御
	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("should-not-be-called-when-wait-fails")
	}
	defer func() { LoadScript = origLoader }()

	rr := httptest.NewRecorder()
	HandleLightClawToken(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("wait 超时应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "实例服务尚未就绪") {
		t.Errorf("响应体应包含 '实例服务尚未就绪'，实际=%s", rr.Body.String())
	}
	if approveCalled {
		t.Error("wait 失败时不应调用 approve_device.sh")
	}
}

// TestHandleLightClawToken_OpenClawWaitReadyThenApproveOK 覆盖完整 happy path：
// wait 通过 + approve 通过 → 200 + 返回 token / sign / callbackUrl。
// 通过自定义 lightClawHandler + mock RunScript 来精确控制 approve 行为，
// 避免依赖真实 LoadScript 与 TAT。
func TestHandleLightClawToken_OpenClawWaitReadyThenApproveOK(t *testing.T) {
	initLightClawHandlerTestDB(t)

	// wait 阶段：让 check_ready 立即 ready=true
	// 注意：必须显式匹配 "check_openclaw_ready.sh"（ResolveScript 对 OpenClaw 的解析结果）。
	origRunScriptFn := runScriptFn
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string,
		onOutput func(string), params map[string]string) (string, error) {
		if scriptName == "check_openclaw_ready.sh" {
			return `{"ready": true}`, nil
		}
		return "", common.I18nError(i18n.MsgUnexpectedScript, scriptName)
	}
	defer func() { runScriptFn = origRunScriptFn }()

	// approve 阶段：注入 mockLightClawDeps，让 deps.RunScript 直接 mock 成功
	var approveCalled bool
	mockDeps := mockLightClawDepsForToken(func(scriptName string) (string, error) {
		if scriptName == "approve_device.sh" {
			approveCalled = true
			return "approved", nil
		}
		return "", fmt.Errorf("unexpected script in deps.RunScript: %s", scriptName)
	})
	h := newLightClawHandler(mockDeps)

	user := &model.User{Username: "u-ok", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	proxyToken := "pt-ok"
	inst := &model.Instance{
		Name: "ok-inst", InstanceId: "ins-ok-token",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(inst)

	req := lightClawReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/lightclaw/token?id=%d", inst.ID), "u-ok")
	rr := httptest.NewRecorder()
	h.handleToken(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("happy path 应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !approveCalled {
		t.Error("wait 通过后应调用 approve_device.sh")
	}
	if !strings.Contains(rr.Body.String(), "proxyToken") {
		t.Errorf("响应体应包含 proxyToken，实际=%s", rr.Body.String())
	}

	waitLCGoroutines()
}
