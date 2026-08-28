package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
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

// ─── 辅助函数 ────────────────────────────────────────────────────────────────

// initInternalLoginTestDB 初始化内存 SQLite，迁移 HandleInternalLogin 所需的表。
func initInternalLoginTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.SiteConfig{},
		&model.AuditLog{},
		&model.SessionBlacklist{},
		&model.OneIDUserProfile{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(useDBForTestWithSafeRestore(db))
	db.Create(&model.SiteConfig{})
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
}

// ctxWithSecret 返回注入了 InternalSecret 的 context（通过 common.FixedSnapshot）。
func ctxWithSecret(t *testing.T, secret string) func() {
	t.Helper()
	old := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{
		InternalSecret: secret,
		Domain:         "https://example.com",
	}
	return func() { common.FixedSnapshot = old }
}

// buildTestInternalToken 用 hex-encoded secret 构造合法的 verifyInternalToken token。
// 格式：base64url(jsonPayload).base64url(hmac-sha256)
func buildTestInternalToken(hexSecret string, payload internalTokenPayload) string {
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	secretBytes, err := hex.DecodeString(hexSecret)
	if err != nil {
		secretBytes = []byte(hexSecret)
	}
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(payloadB64))
	sig := mac.Sum(nil)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return payloadB64 + "." + sigB64
}

// buildTestRequestToken 构造合法的 verifyInternalRequestToken header 值。
// 格式：timestamp.hex_hmac
func buildTestRequestToken(secret, tsStr string) string {
	sig := hmacSHA256([]byte(secret), []byte(tsStr))
	return tsStr + "." + hex.EncodeToString(sig)
}

// makeTestJWT 构造三段式 JWT（仅 base64url payload，header/sig 为占位符）。
func makeTestJWT(payload any) string {
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return "eyJhbGciOiJub25lIn0." + payloadB64 + ".sig"
}

// ─── Task 2: verifyInternalToken 单元测试 ────────────────────────────────────

func TestVerifyInternalToken_Valid(t *testing.T) {
	secret := hex.EncodeToString([]byte("test-raw-secret-32bytes-padding!!"))
	payload := internalTokenPayload{
		Sub:     "user-sub-001",
		Name:    "测试用户",
		IsAdmin: true,
		Exp:     time.Now().Add(time.Hour).Unix(),
	}
	token := buildTestInternalToken(secret, payload)

	got, err := verifyInternalToken(secret, token)
	if err != nil {
		t.Fatalf("合法 token 应通过验证，got err: %v", err)
	}
	if got.Sub != payload.Sub {
		t.Errorf("Sub 期望 %q，实际 %q", payload.Sub, got.Sub)
	}
	if got.Name != payload.Name {
		t.Errorf("Name 期望 %q，实际 %q", payload.Name, got.Name)
	}
	if got.IsAdmin != payload.IsAdmin {
		t.Errorf("IsAdmin 期望 %v，实际 %v", payload.IsAdmin, got.IsAdmin)
	}
}

func TestVerifyInternalToken_Expired(t *testing.T) {
	secret := hex.EncodeToString([]byte("test-raw-secret-32bytes-padding!!"))
	payload := internalTokenPayload{
		Sub: "user-expired",
		Exp: time.Now().Add(-time.Hour).Unix(), // 已过期
	}
	token := buildTestInternalToken(secret, payload)

	_, err := verifyInternalToken(secret, token)
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDAuthTokenExpired)) {
		t.Fatalf("过期 token 应返回 expired error，got: %v", err)
	}
}

func TestVerifyInternalToken_WrongSecret(t *testing.T) {
	secret := hex.EncodeToString([]byte("correct-secret-32bytes-padding!!!"))
	wrongSecret := hex.EncodeToString([]byte("wrong-secret-32bytes-padding!!!!"))
	payload := internalTokenPayload{
		Sub: "user-sub",
		Exp: time.Now().Add(time.Hour).Unix(),
	}
	token := buildTestInternalToken(secret, payload)

	_, err := verifyInternalToken(wrongSecret, token)
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDAuthInvalidSignature)) {
		t.Fatalf("错误 secret 应返回 invalid signature error，got: %v", err)
	}
}

func TestVerifyInternalToken_MissingSeparator(t *testing.T) {
	_, err := verifyInternalToken("anysecret", "nodotintoken")
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDAuthTokenMissingSeparator)) {
		t.Fatalf("缺少分隔符应返回 missing separator error，got: %v", err)
	}
}

func TestVerifyInternalToken_NonHexSecretFallback(t *testing.T) {
	// 非 hex secret（raw string 直接作为 key）
	rawSecret := "not-a-hex-string!!!"
	payload := internalTokenPayload{
		Sub: "user-fallback",
		Exp: time.Now().Add(time.Hour).Unix(),
	}
	// buildTestInternalToken 内部也会走 fallback 分支（hex.DecodeString 失败 → []byte(rawSecret)）
	token := buildTestInternalToken(rawSecret, payload)

	got, err := verifyInternalToken(rawSecret, token)
	if err != nil {
		t.Fatalf("non-hex secret fallback 应通过验证，got err: %v", err)
	}
	if got.Sub != payload.Sub {
		t.Errorf("Sub 期望 %q，实际 %q", payload.Sub, got.Sub)
	}
}

// ─── Task 3: verifyInternalRequestToken 单元测试 ─────────────────────────────

func TestVerifyInternalRequestToken_Valid(t *testing.T) {
	secret := "test-internal-secret"
	defer ctxWithSecret(t, secret)()

	tsStr := fmt.Sprintf("%d", time.Now().Unix())
	tokenVal := buildTestRequestToken(secret, tsStr)

	req := httptest.NewRequest(http.MethodGet, "/spi/test", nil)
	req.Header.Set("X-Internal-Token", tokenVal)
	// 注入 secret 到 context
	snap := common.TenantSnapshot{InternalSecret: secret}
	req = req.WithContext(common.InjectTenant(req.Context(), snap))

	if err := verifyInternalRequestToken(req); err != nil {
		t.Fatalf("合法 token 应通过验证，got err: %v", err)
	}
}

func TestVerifyInternalRequestToken_MissingHeader(t *testing.T) {
	secret := "test-internal-secret"
	req := httptest.NewRequest(http.MethodGet, "/spi/test", nil)
	snap := common.TenantSnapshot{InternalSecret: secret}
	req = req.WithContext(common.InjectTenant(req.Context(), snap))

	err := verifyInternalRequestToken(req)
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDAuthMissingInternalToken)) {
		t.Fatalf("缺少 header 应返回 missing error，got: %v", err)
	}
}

func TestVerifyInternalRequestToken_Expired(t *testing.T) {
	secret := "test-internal-secret"
	tsStr := fmt.Sprintf("%d", time.Now().Unix()-200) // 超出 120s 窗口
	tokenVal := buildTestRequestToken(secret, tsStr)

	req := httptest.NewRequest(http.MethodGet, "/spi/test", nil)
	req.Header.Set("X-Internal-Token", tokenVal)
	snap := common.TenantSnapshot{InternalSecret: secret}
	req = req.WithContext(common.InjectTenant(req.Context(), snap))

	err := verifyInternalRequestToken(req)
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDAuthTokenExpiredDiff)) {
		t.Fatalf("超时 token 应返回 expired error，got: %v", err)
	}
}

func TestVerifyInternalRequestToken_WrongSig(t *testing.T) {
	secret := "test-internal-secret"
	tsStr := fmt.Sprintf("%d", time.Now().Unix())
	tokenVal := tsStr + ".deadbeefdeadbeef" // 错误签名

	req := httptest.NewRequest(http.MethodGet, "/spi/test", nil)
	req.Header.Set("X-Internal-Token", tokenVal)
	snap := common.TenantSnapshot{InternalSecret: secret}
	req = req.WithContext(common.InjectTenant(req.Context(), snap))

	err := verifyInternalRequestToken(req)
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDAuthSignatureMismatch)) {
		t.Fatalf("错误签名应返回 signature mismatch error，got: %v", err)
	}
}

func TestVerifyInternalRequestToken_NoSecret(t *testing.T) {
	// context 中 InternalSecret 为空
	req := httptest.NewRequest(http.MethodGet, "/spi/test", nil)
	req.Header.Set("X-Internal-Token", "1234567890.abc")
	// 不注入 TenantSnapshot → InternalSecretFromCtx 返回 ""

	err := verifyInternalRequestToken(req)
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDAuthInternalSecretNotSet)) {
		t.Fatalf("未配置 secret 应返回 not configured error，got: %v", err)
	}
}

// ─── Task 4: parseIDTokenClaims 单元测试 ─────────────────────────────────────

func TestParseIDTokenClaims_Valid(t *testing.T) {
	claims := map[string]any{
		"sub":   "sub-001",
		"name":  "张三",
		"email": "zhang@example.com",
		"sid":   "session-id-001",
	}
	jwt := makeTestJWT(claims)

	got, err := parseIDTokenClaims(jwt)
	if err != nil {
		t.Fatalf("合法 id_token 应解析成功，got err: %v", err)
	}
	if got.Sub != "sub-001" {
		t.Errorf("Sub 期望 sub-001，实际 %q", got.Sub)
	}
	if got.Name != "张三" {
		t.Errorf("Name 期望 张三，实际 %q", got.Name)
	}
	if got.Email != "zhang@example.com" {
		t.Errorf("Email 期望 zhang@example.com，实际 %q", got.Email)
	}
	if got.Sid != "session-id-001" {
		t.Errorf("Sid 期望 session-id-001，实际 %q", got.Sid)
	}
}

func TestParseIDTokenClaims_MissingSub(t *testing.T) {
	claims := map[string]any{
		"name": "无 sub 用户",
	}
	jwt := makeTestJWT(claims)

	_, err := parseIDTokenClaims(jwt)
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDAuthIDTokenMissingSub)) {
		t.Fatalf("缺少 sub 应返回 missing sub error，got: %v", err)
	}
}

func TestParseIDTokenClaims_InvalidFormat(t *testing.T) {
	_, err := parseIDTokenClaims("only.twoparts")
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDAuthInvalidIDTokenFormat)) {
		t.Fatalf("两段格式应返回 invalid error，got: %v", err)
	}
}

func TestParseIDTokenClaims_BadBase64(t *testing.T) {
	// 第二段不是合法 base64url
	_, err := parseIDTokenClaims("header.!!!invalid_base64!!!.sig")
	if err == nil {
		t.Fatal("非法 base64 第二段应返回 error")
	}
}

// ─── Task 5: parseLogoutTokenClaims 单元测试 ─────────────────────────────────

func TestParseLogoutTokenClaims_Valid(t *testing.T) {
	now := time.Now().Unix()
	claims := map[string]any{
		"sub":  "logout-sub",
		"sid":  "logout-sid",
		"exp":  now + 3600,
		"sexp": now + 86400,
	}
	jwt := makeTestJWT(claims)

	got, err := parseLogoutTokenClaims(jwt)
	if err != nil {
		t.Fatalf("合法 logout_token 应解析成功，got err: %v", err)
	}
	if got.Sub != "logout-sub" {
		t.Errorf("Sub 期望 logout-sub，实际 %q", got.Sub)
	}
	if got.Sid != "logout-sid" {
		t.Errorf("Sid 期望 logout-sid，实际 %q", got.Sid)
	}
	if got.Exp != now+3600 {
		t.Errorf("Exp 期望 %d，实际 %d", now+3600, got.Exp)
	}
}

func TestParseLogoutTokenClaims_InvalidFormat(t *testing.T) {
	_, err := parseLogoutTokenClaims("singlepart")
	if err == nil || !errors.Is(err, common.I18nError(i18n.MsgOneIDAuthInvalidLogoutTokenFmt)) {
		t.Fatalf("单段格式应返回 invalid error，got: %v", err)
	}
}

// ─── Task 6: HandleInternalLogin HTTP 集成测试 ────────────────────────────────

// internalLoginRequest 构造注入了 InternalSecret 的 GET 请求。
func internalLoginRequest(secret, token, redirectTo string) *http.Request {
	url := "/auth/internal-login?token=" + token
	if redirectTo != "" {
		url += "&redirect_to=" + redirectTo
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	snap := common.TenantSnapshot{InternalSecret: secret}
	return req.WithContext(common.InjectTenant(req.Context(), snap))
}

// validLoginToken 构造一个合法的、未过期的 HandleInternalLogin token。
func validLoginToken(secret string, sub, name string, isAdmin bool) string {
	payload := internalTokenPayload{
		Sub:     sub,
		Name:    name,
		IsAdmin: isAdmin,
		Exp:     time.Now().Add(time.Hour).Unix(),
		Iat:     time.Now().Unix(),
	}
	return buildTestInternalToken(secret, payload)
}

func TestHandleInternalLogin_NoSecretReturns500(t *testing.T) {
	initInternalLoginTestDB(t)
	// 不注入 InternalSecret（使用空 context）
	req := httptest.NewRequest(http.MethodGet, "/auth/internal-login", nil)
	w := httptest.NewRecorder()
	HandleInternalLogin(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，got %d", w.Code)
	}
}

func TestHandleInternalLogin_MissingTokenReturns400(t *testing.T) {
	initInternalLoginTestDB(t)
	secret := "test-secret"
	req := httptest.NewRequest(http.MethodGet, "/auth/internal-login", nil)
	snap := common.TenantSnapshot{InternalSecret: secret}
	req = req.WithContext(common.InjectTenant(req.Context(), snap))
	w := httptest.NewRecorder()
	HandleInternalLogin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，got %d", w.Code)
	}
}

func TestHandleInternalLogin_InvalidTokenReturns401(t *testing.T) {
	initInternalLoginTestDB(t)
	secret := "test-secret"
	req := internalLoginRequest(secret, "invalid.token.value", "")
	w := httptest.NewRecorder()
	HandleInternalLogin(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，got %d", w.Code)
	}
}

func TestHandleInternalLogin_NewUser(t *testing.T) {
	initInternalLoginTestDB(t)
	secret := hex.EncodeToString([]byte("new-user-test-secret-32bytes!!!!"))
	sub := "new-user-sub-001"
	token := validLoginToken(secret, sub, "新用户", false)

	req := internalLoginRequest(secret, token, "")
	w := httptest.NewRecorder()
	HandleInternalLogin(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("期望 302，got %d body=%s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if loc != "/my-openclaw" {
		t.Errorf("Location 期望 /my-openclaw，got %q", loc)
	}

	var user model.User
	if model.DB(req.Context()).Where("one_id_sub = ?", sub).First(&user).Error != nil {
		t.Fatal("应在数据库中创建新用户")
	}
}

// ─── HandleOneIDLogin / forwardCodeToGateway 覆盖率补充 ──────────────────────

func TestHandleOneIDLogin_RedirectsToGateway(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = "https://gateway.example.com"
	defer func() { GatewayURL = origGW }()

	snap := common.TenantSnapshot{
		OneIDAccountID: "tenant-001",
		Domain:         "https://example.com",
	}
	req := httptest.NewRequest(http.MethodGet, "/auth/oneid?acr_values=sms&idp=wechat", nil)
	req = req.WithContext(common.InjectTenant(req.Context(), snap))
	w := httptest.NewRecorder()
	HandleOneIDLogin(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("期望 302，got %d body=%s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "https://gateway.example.com/auth/oneid") {
		t.Errorf("Location 应指向 gateway /auth/oneid，got %q", loc)
	}
	if !strings.Contains(loc, "tenant=tenant-001") {
		t.Errorf("Location 应包含 tenant 参数，got %q", loc)
	}
	if !strings.Contains(loc, "acr_values=sms") {
		t.Errorf("Location 应透传 acr_values，got %q", loc)
	}
	if !strings.Contains(loc, "idp=wechat") {
		t.Errorf("Location 应透传 idp，got %q", loc)
	}
}

func TestHandleOneIDCode_RedirectsToGateway(t *testing.T) {
	origGW := GatewayURL
	GatewayURL = "https://gateway.example.com"
	defer func() { GatewayURL = origGW }()

	snap := common.TenantSnapshot{
		OneIDAccountID: "tenant-001",
		Domain:         "https://example.com",
	}
	req := httptest.NewRequest(http.MethodGet, "/auth/oneid-code?code=abc123", nil)
	req = req.WithContext(common.InjectTenant(req.Context(), snap))
	w := httptest.NewRecorder()
	HandleOneIDCode(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("期望 302，got %d body=%s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "https://gateway.example.com/auth/idp-callback") {
		t.Errorf("Location 应指向 gateway /auth/idp-callback，got %q", loc)
	}
	if !strings.Contains(loc, "code=abc123") {
		t.Errorf("Location 应包含 code 参数，got %q", loc)
	}
}

func TestHandleInternalLogin_RestoreSoftDeleted(t *testing.T) {
	initInternalLoginTestDB(t)
	secret := hex.EncodeToString([]byte("restore-user-secret-32bytes!!!!!!"))
	sub := "restore-sub-002"

	// 先创建再软删除
	user := model.User{Username: "soft-del-login", OneIDSub: &sub}
	db := model.DB(nil)
	db.Create(&user)
	db.Delete(&user)

	token := validLoginToken(secret, sub, "已恢复用户", false)
	req := internalLoginRequest(secret, token, "")
	w := httptest.NewRecorder()
	HandleInternalLogin(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("期望 302，got %d body=%s", w.Code, w.Body.String())
	}

	var restored model.User
	db.Unscoped().Where("one_id_sub = ?", sub).First(&restored)
	if restored.DeletedAt.Valid {
		t.Fatal("软删除用户应被恢复（deleted_at 清除）")
	}
}

func TestHandleInternalLogin_RedirectTo_Valid(t *testing.T) {
	initInternalLoginTestDB(t)
	secret := hex.EncodeToString([]byte("redirect-valid-secret-32bytes!!!!"))
	sub := "redirect-sub-valid"
	token := validLoginToken(secret, sub, "redirect用户", false)

	req := internalLoginRequest(secret, token, "/admin/basic-info")
	w := httptest.NewRecorder()
	HandleInternalLogin(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("期望 302，got %d body=%s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if loc != "/admin/basic-info" {
		t.Errorf("Location 期望 /admin/basic-info，got %q", loc)
	}
}

func TestHandleInternalLogin_RedirectTo_Rejected(t *testing.T) {
	initInternalLoginTestDB(t)
	secret := hex.EncodeToString([]byte("redirect-evil-secret-32bytes!!!!!"))
	sub := "redirect-sub-evil"
	token := validLoginToken(secret, sub, "evil用户", false)

	req := internalLoginRequest(secret, token, "https://evil.com/steal")
	w := httptest.NewRecorder()
	HandleInternalLogin(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("期望 302，got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/my-openclaw" {
		t.Errorf("非法 redirect_to 应 fallback 到 /my-openclaw，got %q", loc)
	}
}

func TestHandleInternalLogin_AdminRoleSync(t *testing.T) {
	initInternalLoginTestDB(t)
	secret := hex.EncodeToString([]byte("admin-role-secret-32bytes!!!!!!!!!"))
	sub := "admin-role-sub-003"

	// 先创建普通用户
	user := model.User{Username: "to-be-admin", Role: "user", OneIDSub: &sub}
	model.DB(nil).Create(&user)

	token := validLoginToken(secret, sub, "to-be-admin", true) // IsAdmin=true
	req := internalLoginRequest(secret, token, "")
	w := httptest.NewRecorder()
	HandleInternalLogin(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("期望 302，got %d body=%s", w.Code, w.Body.String())
	}

	var updated model.User
	model.DB(nil).Where("one_id_sub = ?", sub).First(&updated)
	if updated.Role != "admin" {
		t.Errorf("IsAdmin=true 时角色应更新为 admin，实际 %q", updated.Role)
	}
}
