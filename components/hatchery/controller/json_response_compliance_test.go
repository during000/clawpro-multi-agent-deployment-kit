package controller

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// expectJSONError 校验响应是否符合 writeError 输出契约。
func expectJSONError(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int, wantSubstr string) {
	t.Helper()
	if rr.Code != wantStatus {
		t.Fatalf("status: got=%d want=%d body=%s", rr.Code, wantStatus, rr.Body.String())
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type: got=%q, want application/json (body=%s)", ct, rr.Body.String())
	}
	if rr.Header().Get("X-Audit-Failed") != "1" {
		t.Fatalf("缺少 X-Audit-Failed=1 头（writeError 必须设置），实际=%q",
			rr.Header().Get("X-Audit-Failed"))
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body 不是合法 JSON: %v body=%s", err, rr.Body.String())
	}
	errMsg, ok := resp["error"].(string)
	if !ok {
		t.Fatalf("body 缺少 string 字段 error: body=%s", rr.Body.String())
	}
	if wantSubstr != "" && !strings.Contains(errMsg, wantSubstr) {
		t.Fatalf("error 字段不含 %q: %q", wantSubstr, errMsg)
	}
}

func TestMethodNotAllowed_JSONResponse_NoAuth(t *testing.T) {
	cases := []struct {
		name     string
		method   string
		path     string
		handler  http.HandlerFunc
		statusOK int // 改动前/改动后都应该是 405
	}{
		// auth.go：6 处 405 入口
		{"HandleLogin_GET", http.MethodGet, "/auth/login", HandleLogin, http.StatusMethodNotAllowed},
		{"HandleChangePassword_GET", http.MethodGet, "/auth/change-password", HandleChangePassword, http.StatusMethodNotAllowed},
		{"HandleGetAPIToken_POST", http.MethodPost, "/auth/api-token", HandleGetAPIToken, http.StatusMethodNotAllowed},
		{"HandleCreateAPIToken_GET", http.MethodGet, "/auth/api-token/create", HandleCreateAPIToken, http.StatusMethodNotAllowed},
		{"HandleResetAPIToken_GET", http.MethodGet, "/auth/api-token/reset", HandleResetAPIToken, http.StatusMethodNotAllowed},
		{"HandleRevokeAPIToken_GET", http.MethodGet, "/auth/api-token/revoke", HandleRevokeAPIToken, http.StatusMethodNotAllowed},

		// auth_oneid.go：HandleOneIDLogout / HandleOneIDEvent
		{"HandleOneIDLogout_GET", http.MethodGet, "/spi/logout", HandleOneIDLogout, http.StatusMethodNotAllowed},
		{"HandleOneIDEvent_GET", http.MethodGet, "/spi/event", HandleOneIDEvent, http.StatusMethodNotAllowed},

		// admin_skill_security_scan.go：switch default 分支
		{"HandleSkillScanConfigRouter_PUT", http.MethodPut, "/admin/skill-scan/config", HandleSkillScanConfigRouter, http.StatusMethodNotAllowed},

		// admin_mcp.go：switch default 分支
		{"HandleAdminMcpVersionsRouter_POST", http.MethodPost, "/admin/mcp/versions", HandleAdminMcpVersionsRouter, http.StatusMethodNotAllowed},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			rr := httptest.NewRecorder()
			c.handler.ServeHTTP(rr, req)
			// 注意：HandleAdminMcpVersionsRouter 的 default 分支会先调用 writeError。
			// 上述表里的 method 都被设计为不在 handler 接受范围内，必然进入 405 分支。
			expectJSONError(t, rr, http.StatusMethodNotAllowed, "")
		})
	}
}

func TestLightClawHandleToken_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/lightclaw/token", nil)
	rr := httptest.NewRecorder()
	HandleLightClawToken(rr, req)
	expectJSONError(t, rr, http.StatusMethodNotAllowed, "")
}

func TestHandleOneIDLogin_GatewayUnconfigured(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = ""
	defer func() { GatewayURL = origGW }()

	req := httptest.NewRequest(http.MethodGet, "/auth/oneid", nil)
	rr := httptest.NewRecorder()
	HandleOneIDLogin(rr, req)

	expectJSONError(t, rr, http.StatusBadRequest, "Gateway 未配置")
}

// TestHandleOneIDLogin_TenantUnconfigured 验证 GatewayURL 已配置但 tenant 缺失时返回 JSON 400。
func TestHandleOneIDLogin_TenantUnconfigured(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = "https://gw.example.com"
	defer func() { GatewayURL = origGW }()

	// 不设置 TenantIDFromCtx → context 中 tenant 为空
	req := httptest.NewRequest(http.MethodGet, "/auth/oneid", nil)
	rr := httptest.NewRecorder()
	HandleOneIDLogin(rr, req)

	expectJSONError(t, rr, http.StatusBadRequest, "OneID 租户未配置")
}

func TestForwardCodeToGateway_GatewayUnconfigured(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = ""
	defer func() { GatewayURL = origGW }()

	req := httptest.NewRequest(http.MethodGet, "/auth/oneid-callback?code=abc", nil)
	rr := httptest.NewRecorder()
	forwardCodeToGateway(rr, req, "test")

	expectJSONError(t, rr, http.StatusBadRequest, "Gateway 未配置")
}

// TestForwardCodeToGateway_MissingCode 验证缺少 code 参数时返回 JSON 400。
func TestForwardCodeToGateway_MissingCode(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = "https://gw.example.com"
	defer func() { GatewayURL = origGW }()

	req := httptest.NewRequest(http.MethodGet, "/auth/oneid-callback", nil)
	rr := httptest.NewRecorder()
	forwardCodeToGateway(rr, req, "test")

	expectJSONError(t, rr, http.StatusBadRequest, i18n.T(req.Context(), i18n.MsgOneIDMissingCode))
}

// TestForwardCodeToGateway_TenantUnconfigured 验证 tenant 缺失时返回 JSON 400。
func TestForwardCodeToGateway_TenantUnconfigured(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = "https://gw.example.com"
	defer func() { GatewayURL = origGW }()

	req := httptest.NewRequest(http.MethodGet, "/auth/oneid-callback?code=abc", nil)
	rr := httptest.NewRecorder()
	forwardCodeToGateway(rr, req, "test")

	expectJSONError(t, rr, http.StatusBadRequest, i18n.T(req.Context(), i18n.MsgOneIDTenantNotConfigured))
}

// TestHandleInternalLogin_InternalSecretNotConfigured 验证 InternalSecret 为空时
// 返回 500 JSON 而非纯文本（曾经是 http.Error 的位置）。
func TestHandleInternalLogin_InternalSecretNotConfigured(t *testing.T) {
	// context 不注入 InternalSecret → InternalSecretFromCtx 返回 ""
	req := httptest.NewRequest(http.MethodGet, "/auth/internal-login", nil)
	rr := httptest.NewRecorder()
	HandleInternalLogin(rr, req)

	expectJSONError(t, rr, http.StatusInternalServerError, i18n.T(req.Context(), i18n.MsgInternalLoginNotConfig))
}

// TestHandleInternalLogin_MissingToken 验证 InternalSecret 已配置但缺少 token 参数时返回 JSON 400。
func TestHandleInternalLogin_MissingToken(t *testing.T) {
	// 注入 tenant snapshot 让 InternalSecretFromCtx 返回非空
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		InternalSecret: "test-secret-32-bytes-long-padding-!!!",
	})
	req := httptest.NewRequest(http.MethodGet, "/auth/internal-login", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	HandleInternalLogin(rr, req)

	expectJSONError(t, rr, http.StatusBadRequest, i18n.T(req.Context(), i18n.MsgInternalLoginMissToken))
}

// TestHandleInternalLogin_InvalidToken 验证 token 校验失败时返回 JSON 401。
func TestHandleInternalLogin_InvalidToken(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		InternalSecret: "test-secret-32-bytes-long-padding-!!!",
	})
	req := httptest.NewRequest(http.MethodGet, "/auth/internal-login?token=garbage", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	HandleInternalLogin(rr, req)

	expectJSONError(t, rr, http.StatusUnauthorized, i18n.T(req.Context(), i18n.MsgInternalLoginTokenInval))
}

// TestHandleOneIDLogout_UnauthorizedInternalToken 内部 token 缺失/不匹配 → JSON 401。
func TestHandleOneIDLogout_UnauthorizedInternalToken(t *testing.T) {
	// 没有注入 InternalSecret，verifyInternalRequestToken 必然返回 err
	req := httptest.NewRequest(http.MethodPost, "/spi/logout", strings.NewReader("logout_token=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	HandleOneIDLogout(rr, req)

	expectJSONError(t, rr, http.StatusUnauthorized, i18n.T(req.Context(), i18n.MsgOneIDUnauthorized))
}

// TestHandleOneIDEvent_UnauthorizedInternalToken 内部 token 缺失 → JSON 401。
func TestHandleOneIDEvent_UnauthorizedInternalToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/spi/event", strings.NewReader(`{"event_type":"member.created"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleOneIDEvent(rr, req)

	expectJSONError(t, rr, http.StatusUnauthorized, i18n.T(req.Context(), i18n.MsgOneIDUnauthorized))
}

// TestWriteOneIDEventOK_ReturnsJSONUUID 验证从 http.Error/手动 Encode 改为 jsonOK 后，
// 成功响应仍然是 OneID 协议要求的 {"uuid": "..."} JSON 结构。
func TestWriteOneIDEventOK_ReturnsJSONUUID(t *testing.T) {
	rr := httptest.NewRecorder()
	writeOneIDEventOK(rr, "evt-uuid-12345")

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got=%d want=200", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type: got=%q want application/json", ct)
	}
	// 成功响应一定不带 X-Audit-Failed
	if rr.Header().Get("X-Audit-Failed") != "" {
		t.Fatalf("成功响应不应有 X-Audit-Failed 头，实际=%q", rr.Header().Get("X-Audit-Failed"))
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body 不是 JSON: %v body=%s", err, rr.Body.String())
	}
	if resp["uuid"] != "evt-uuid-12345" {
		t.Fatalf("uuid 字段不匹配：got=%q want=%q", resp["uuid"], "evt-uuid-12345")
	}
}

// signInternalRequestForTest 复刻 Gateway SignInternalRequest 逻辑，
// 用于在测试中构造合法的 X-Internal-Token，从而越过 verifyInternalRequestToken。
func signInternalRequestForTest(secret string) string {
	tsStr := strconv.FormatInt(time.Now().Unix(), 10)
	sig := hmacSHA256([]byte(secret), []byte(tsStr))
	return tsStr + "." + hex.EncodeToString(sig)
}

// TestHandleOneIDLogout_MissingLogoutToken token 通过后缺 logout_token → 400 JSON。
func TestHandleOneIDLogout_MissingLogoutToken(t *testing.T) {
	secret := "test-internal-secret-32-bytes-pad!"
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{InternalSecret: secret})

	form := strings.NewReader("") // 不含 logout_token
	req := httptest.NewRequest(http.MethodPost, "/spi/logout", form).WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Internal-Token", signInternalRequestForTest(secret))

	rr := httptest.NewRecorder()
	HandleOneIDLogout(rr, req)

	expectJSONError(t, rr, http.StatusBadRequest, i18n.T(req.Context(), i18n.MsgOneIDMissingLogoutToken))
}

// TestHandleOneIDLogout_InvalidLogoutToken token 通过但 logout_token 解析失败 → 400 JSON。
func TestHandleOneIDLogout_InvalidLogoutToken(t *testing.T) {
	secret := "test-internal-secret-32-bytes-pad!"
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{InternalSecret: secret})

	form := strings.NewReader("logout_token=not-a-jwt")
	req := httptest.NewRequest(http.MethodPost, "/spi/logout", form).WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Internal-Token", signInternalRequestForTest(secret))

	rr := httptest.NewRecorder()
	HandleOneIDLogout(rr, req)

	expectJSONError(t, rr, http.StatusBadRequest, i18n.T(req.Context(), i18n.MsgOneIDInvalidLogoutToken))
}

// TestHandleOneIDEvent_MalformedJSONBody token 通过但 body 不是合法 JSON → 400 JSON。
func TestHandleOneIDEvent_MalformedJSONBody(t *testing.T) {
	secret := "test-internal-secret-32-bytes-pad!"
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{InternalSecret: secret})

	req := httptest.NewRequest(http.MethodPost, "/spi/event", strings.NewReader("not json"))
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", signInternalRequestForTest(secret))

	rr := httptest.NewRecorder()
	HandleOneIDEvent(rr, req)

	expectJSONError(t, rr, http.StatusBadRequest, i18n.T(req.Context(), i18n.MsgBadRequest))
}

// jumpReqWithSession 构造带 hatchery-session（admin 已登录）的 GET /admin/oneid/jump 请求。
func jumpReqWithSession(t *testing.T, username string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/admin/oneid/jump", nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username
	rr := httptest.NewRecorder()
	_ = session.Save(req, rr)
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

// TestHandleOneIDJump_GatewayUnconfigured_JSONError 验证 GatewayURL 为空 → 503 JSON。
func TestHandleOneIDJump_GatewayUnconfigured_JSONError(t *testing.T) {
	initOneIDJumpTestDB(t)

	origGW := GatewayURL
	GatewayURL = ""
	defer func() { GatewayURL = origGW }()

	req := httptest.NewRequest(http.MethodGet, "/admin/oneid/jump", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleOneIDJump(rr, req)

	expectJSONError(t, rr, http.StatusServiceUnavailable, i18n.T(req.Context(), i18n.MsgGatewayNotConfigured))
}

// TestHandleOneIDJump_UserMissingOneIDSub_JSONError 验证 admin 用户未关联 OneID → 400 JSON。
func TestHandleOneIDJump_UserMissingOneIDSub_JSONError(t *testing.T) {
	initOneIDJumpTestDB(t)

	origGW := GatewayURL
	GatewayURL = "https://gateway.example.com"
	defer func() { GatewayURL = origGW }()

	// 创建 admin 用户但 OneIDSub=nil
	model.DB(context.Background()).Create(&model.User{
		Username: "admin-no-sub",
		Role:     "admin",
	})

	req := jumpReqWithSession(t, "admin-no-sub")
	rr := httptest.NewRecorder()
	HandleOneIDJump(rr, req)

	expectJSONError(t, rr, http.StatusBadRequest, i18n.T(req.Context(), i18n.MsgUserNotLinkedToOneID))
}

// TestHandleOneIDJump_TenantMissing_JSONError 验证 sub 已绑定但 tenant 未注入 → 400 JSON。
func TestHandleOneIDJump_TenantMissing_JSONError(t *testing.T) {
	initOneIDJumpTestDB(t)

	origGW := GatewayURL
	GatewayURL = "https://gateway.example.com"
	defer func() { GatewayURL = origGW }()

	sub := "admin-with-sub"
	model.DB(context.Background()).Create(&model.User{
		Username: "admin-tenant-miss",
		Role:     "admin",
		OneIDSub: &sub,
	})

	// 不注入 tenant；当前 ctx TenantIDFromCtx 应返回 ""
	req := jumpReqWithSession(t, "admin-tenant-miss")
	rr := httptest.NewRecorder()
	HandleOneIDJump(rr, req)

	expectJSONError(t, rr, http.StatusBadRequest, i18n.T(req.Context(), i18n.MsgOneIDTenantNotConfigured))
}

// TestJsonOK_ContentTypeAndBody 验证 jsonOK 设置了 application/json 且写入了正确 JSON。
func TestJsonOK_ContentTypeAndBody(t *testing.T) {
	rr := httptest.NewRecorder()
	jsonOK(rr, map[string]any{"hello": "world", "n": 42})

	if rr.Code != http.StatusOK {
		t.Errorf("默认 status 应为 200，实际 %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type 应为 application/json，实际 %q", got)
	}
	// 成功响应不应有 X-Audit-Failed 头
	if rr.Header().Get("X-Audit-Failed") != "" {
		t.Errorf("成功响应不应有 X-Audit-Failed，实际 %q", rr.Header().Get("X-Audit-Failed"))
	}
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("body 不是合法 JSON: %v body=%s", err, rr.Body.String())
	}
	if got["hello"] != "world" {
		t.Errorf("hello 字段不匹配：got=%v", got["hello"])
	}
	// JSON 中数字反序列化为 float64
	if v, ok := got["n"].(float64); !ok || int(v) != 42 {
		t.Errorf("n 字段不匹配：got=%v(%T)", got["n"], got["n"])
	}
}

// TestJsonOK_NilData 验证传入 nil 时 jsonOK 输出 "null\n"，不应 panic。
func TestJsonOK_NilData(t *testing.T) {
	rr := httptest.NewRecorder()
	jsonOK(rr, nil)
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("即使 data 为 nil 也应设置 Content-Type")
	}
	body := strings.TrimSpace(rr.Body.String())
	if body != "null" {
		t.Errorf("nil 应序列化为 null，实际 body=%q", rr.Body.String())
	}
}

// TestJsonAPI_OnlySetsContentType 验证 jsonAPI 仅设置 Content-Type，不写 body 也不写 status code。
func TestJsonAPI_OnlySetsContentType(t *testing.T) {
	rr := httptest.NewRecorder()
	jsonAPI(rr)

	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("jsonAPI 应设置 Content-Type=application/json，实际 %q", got)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("jsonAPI 不应写 body，实际 body=%q", rr.Body.String())
	}
	// httptest.NewRecorder 默认 Code=200，但只要没有显式 WriteHeader 就视为未写
	// 这里只确保 jsonAPI 自身没有调用 WriteHeader
	if rr.Result().Header.Get("Content-Length") != "" {
		t.Errorf("jsonAPI 不应触发 Content-Length 计算")
	}
}

// TestJsonRedirect_FormatAndContentType 验证 jsonRedirect 输出 {"ok":true,"redirect":"<url>"}。
func TestJsonRedirect_FormatAndContentType(t *testing.T) {
	rr := httptest.NewRecorder()
	jsonRedirect(rr, "/dashboard?foo=bar")

	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type 应为 application/json，实际 %q", got)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body 不是合法 JSON: %v body=%s", err, rr.Body.String())
	}
	if resp["ok"] != true {
		t.Errorf("ok 字段应为 true，实际 %v", resp["ok"])
	}
	if resp["redirect"] != "/dashboard?foo=bar" {
		t.Errorf("redirect 字段不匹配：got=%v", resp["redirect"])
	}
}

// errPlain 仅用于测试：构造一个 *errors.errorString 之外的任意 error。
type errPlain string

func (e errPlain) Error() string { return string(e) }

// TestWriteError_StatusBucketLogLevels 验证不同状态码段日志级别的判定：
// 400 → WARN，500 → ERROR；其它 4xx/5xx 同级别归类。
func TestWriteError_StatusBucketLogLevels(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		wantLevel string // "WARN" or "ERROR"
	}{
		{"400_BadRequest", http.StatusBadRequest, "WARN"},
		{"401_Unauthorized", http.StatusUnauthorized, "WARN"},
		{"403_Forbidden", http.StatusForbidden, "WARN"},
		{"404_NotFound", http.StatusNotFound, "WARN"},
		{"409_Conflict", http.StatusConflict, "WARN"},
		{"500_InternalServerError", http.StatusInternalServerError, "ERROR"},
		{"502_BadGateway", http.StatusBadGateway, "ERROR"},
		{"503_ServiceUnavailable", http.StatusServiceUnavailable, "ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf, restore := captureSlogOutput(t, slog.LevelDebug)
			defer restore()

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/x", nil)
			writeError(rr, r, tc.status, common.I18nError(i18n.MsgOperationFailed))

			out := buf.String()
			wantTag := "level=" + tc.wantLevel
			if !strings.Contains(out, wantTag) {
				t.Errorf("status=%d 应输出 %s 级别日志，实际=%s", tc.status, wantTag, out)
			}
			// 同时校验响应也是 JSON 错误格式
			expectJSONError(t, rr, tc.status, "操作失败")
		})
	}
}

// ─── browser_vnc / openclaw_channel 修改后行为的快速合规性回归 ───────────────

// TestBrowserVNCAccess_UnauthorizedReturnsJSON 验证 HandleBrowserVNCProxy 入口
// 在 requireLogin 失败时返回 JSON 错误而非纯文本（覆盖 browser_vnc.go 之前 http.Error 的位置）。
//
// 由于 HandleBrowserVNCProxy 直接走 requireLogin 失败路径，是 instance fetch 之前的
// 错误，本测试不需要构造 instance/CVM mock。
func TestHandleBrowserVNCProxy_Unauthorized_JSONError(t *testing.T) {
	initOneIDJumpTestDB(t) // 复用一个最小 DB 初始化（含 SiteConfig）

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-proxy?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleBrowserVNCProxy(rr, req)

	// requireLogin 在内部走 writeError，返回 401 JSON
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Fatalf("未登录应返回 401/403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Errorf("Content-Type 应为 application/json，实际 %q body=%s", got, rr.Body.String())
	}
}
