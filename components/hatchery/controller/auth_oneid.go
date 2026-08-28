package controller

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// validateAndGetSafeRedirectURL 安全验证重定向 URL。
// 仅允许相对路径（Scheme 和 Host 为空），且不允许协议相对 URL（// 前缀）。
// 验证失败时返回 defaultURL（通常是首页 /my-openclaw）。
// 注：路由规范化（如 /admin → /admin/basic-info）已在 Gateway 侧完成。
//
// nolint: gosec
func validateAndGetSafeRedirectURL(raw string, defaultURL string) string {
	if raw == "" {
		return defaultURL
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		slog.Warn("validateAndGetSafeRedirectURL: failed to parse URL", "raw", raw, "error", err)
		return defaultURL
	}

	// 仅允许相对路径：Scheme 和 Host 为空，Path 以 / 开头但不是 //
	if parsed.Scheme == "" && parsed.Host == "" &&
		strings.HasPrefix(parsed.Path, "/") &&
		!strings.HasPrefix(parsed.Path, "//") {
		// 使用解析后重组的 URL，而非原始输入：
		// url.Parse 会规范化路径编码（如 /%2e%2e → /..），避免编码绕过。
		safe := parsed.RequestURI()
		if parsed.Fragment != "" {
			safe += "#" + parsed.Fragment
		}
		return safe
	}

	slog.Warn("validateAndGetSafeRedirectURL: rejected suspicious URL",
		"raw", raw, "scheme", parsed.Scheme, "host", parsed.Host, "path", parsed.Path)
	return defaultURL
}

// HandleOneIDLogin 发起 OneID SSO 跳转。
// GET /auth/oneid — 302 重定向至 Gateway /auth/oneid?tenant=xxx，
// Gateway 完成 OIDC 后通过 internal-login 回传登录态，Pod 不直接与 OneID 通信。
func HandleOneIDLogin(w http.ResponseWriter, r *http.Request) {
	if GatewayURL == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGatewayNotConfigured))
		return
	}

	// 取 tenant ID（启动时通过 ONEID_ACCOUNT_ID 环境变量注入）
	tenant := hcommon.TenantIDFromCtx(r.Context())
	if tenant == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgOneIDTenantNotConfigured))
		return
	}

	params := url.Values{}
	params.Set("tenant", tenant)
	// 透传 acr_values（前端指定登录方式：sms / email / sso:preferred_email 等）
	if acr := r.URL.Query().Get("acr_values"); acr != "" {
		params.Set("acr_values", acr)
	}
	// 透传 id_token_hint（IDP-initiated 登录场景）
	if hint := r.URL.Query().Get("id_token_hint"); hint != "" {
		params.Set("id_token_hint", hint)
	}
	// 透传 idp（跳过 OneID SSO 选择页，直达指定 IdP）
	if idp := r.URL.Query().Get("idp"); idp != "" {
		params.Set("idp", idp)
	}
	gatewayAuthURL := GatewayURL + "/auth/oneid?" + params.Encode()
	slog.Info("SSO redirect to Gateway", "gateway_url", gatewayAuthURL, "tenant", tenant)
	// NOCA:open_redirect(目标为 GatewayURL，由环境变量注入，属于可信内部地址，非用户可控输入)
	http.Redirect(w, r, gatewayAuthURL, http.StatusFound) // nolint: gosec
}

// HandleOneIDCode 接收 OneID 嵌入组件（select_account）返回的授权码，
// 转发给 Gateway 的 IDP-initiated callback 端点完成登录。
//
// GET /auth/oneid-code?code=xxx
//
// OneID 嵌入组件调用 /v1/auth/select_account 后直接把 code 返回给前端 JS，
// 前端无需构造复杂参数，直接跳转此地址即可。
func HandleOneIDCode(w http.ResponseWriter, r *http.Request) {
	forwardCodeToGateway(w, r, "oneid-code")
}

// HandleOneIDRegister 接收 OneID 手机号登录返回的 next_step=register_with_app 场景下的 code，
// 与 HandleOneIDCode 走相同的 IDP-initiated 流程（Gateway /auth/idp-callback）。
// 全新手机号注册后 Gateway 会自动在本 Pod 创建用户并建立登录态。
//
// GET /auth/oneid-register?code=xxx
func HandleOneIDRegister(w http.ResponseWriter, r *http.Request) {
	forwardCodeToGateway(w, r, "oneid-register")
}

// forwardCodeToGateway 将前端传来的 OneID code 302 转发给 Gateway /auth/idp-callback。
func forwardCodeToGateway(w http.ResponseWriter, r *http.Request, logTag string) {
	if GatewayURL == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGatewayNotConfigured))
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgOneIDMissingCode))
		return
	}

	// 取 tenant ID（同 /auth/oneid 逻辑）
	tenant := hcommon.TenantIDFromCtx(r.Context())
	if tenant == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgOneIDTenantNotConfigured))
		return
	}

	params := url.Values{}
	params.Set("code", code)
	params.Set("tenant", tenant)
	gatewayURL := GatewayURL + "/auth/idp-callback?" + params.Encode()
	slog.Info("code forwarded to Gateway", "tag", logTag, "tenant", tenant)
	// NOCA:open_redirect(目标为 GatewayURL，由环境变量注入，属于可信内部地址，非用户可控输入)
	http.Redirect(w, r, gatewayURL, http.StatusFound) // nolint: gosec
}

// POST /spi/logout
// 请求格式（OneID 标准）：Content-Type: application/x-www-form-urlencoded
//
//	logout_token=<JWT>
//
// JWT payload 含 sid（会话唯一标识）和 exp（过期时间），解析后写入黑名单。
//
// 安全：Pod 暴露公网，必须验证请求来自 Gateway（通过 X-Internal-Token）。
func HandleOneIDLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	// ── 验证 Gateway 内部 Token ──
	if err := verifyInternalRequestToken(r); err != nil {
		slog.Warn("OneID logout: internal token verification failed", "err", err, "remote", r.RemoteAddr)
		writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgOneIDUnauthorized))
		return
	}

	if err := r.ParseForm(); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	logoutToken := r.FormValue("logout_token")
	if logoutToken == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgOneIDMissingLogoutToken))
		return
	}

	// 解析 logout_token JWT payload（不验签，OneID RS256 公钥暂不做离线验证）
	claims, err := parseLogoutTokenClaims(logoutToken)
	if err != nil {
		slog.Warn("OneID logout: failed to parse logout_token", "err", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgOneIDInvalidLogoutToken))
		return
	}

	if claims.Sid == "" && claims.Sub == "" {
		slog.Warn("OneID logout: logout_token missing both sid and sub claims")
		// 两者都为空时无法定向注销，仍返回 200 避免 OneID 重试
		w.WriteHeader(http.StatusOK)
		return
	}

	// 当 sid 为空时，用 sub 作为唯一标识写入黑名单（生成合成 sid）
	sid := claims.Sid
	if sid == "" {
		sid = "sub:" + claims.Sub
		slog.Warn("OneID logout: sid is empty, using sub as fallback key", "sub", claims.Sub)
	}

	// 确定黑名单 TTL：优先用 sexp（OneID 登录态过期时间），fallback 到 exp（token 过期时间）
	ttlUnix := claims.Sexp
	if ttlUnix == 0 {
		ttlUnix = claims.Exp
	}
	expireAt := time.Unix(ttlUnix, 0)
	// 至少保留 1 小时，避免因时钟误差导致黑名单立即失效
	minExpire := time.Now().Add(time.Hour)
	if expireAt.Before(minExpire) {
		expireAt = minExpire
	}

	if err := model.RevokeSession(r.Context(), sid, claims.Sub, expireAt); err != nil {
		slog.Error("OneID logout: failed to revoke session", "sid", sid, "sub", claims.Sub, "err", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInternalServerError))
		return
	}

	slog.Info("OneID logout: session revoked", "sid", sid, "sub", claims.Sub, "tid", claims.Tid, "expire_at", expireAt)
	w.WriteHeader(http.StatusOK)
}

// oneIDWebhookEvent 是 OneID Webhook 回调的标准结构。
// 签名算法：HEX(SHA256(SECRET_KEY + UUID + DATA))
// 响应必须回复：{"uuid": "<uuid>"}
type oneIDWebhookEvent struct {
	UUID string          `json:"uuid"`
	Type string          `json:"type"`
	Time uint64          `json:"time,string"` // 毫秒时间戳
	Data json.RawMessage `json:"data"`
	Sign string          `json:"sign"`
}

// oneIDEventData 是 /spi/event 回调 data 字段里的事件数据结构。
// 字段名待 OneID 文档确认后对齐，当前按推断字段名处理。
type oneIDEventData struct {
	Sub           string `json:"sub"`
	EnterpriseID  string `json:"enterprise_id"`
	Name          string `json:"name"`
	AssetAction   string `json:"asset_action"`    // keep / delete / transfer
	TransferToSub string `json:"transfer_to_sub"` // asset_action=transfer 时的目标用户 sub
}

// HandleOneIDEvent 处理 OneID 组织架构 Webhook 事件。
// POST /spi/event — 按 type 字段分发处理。
// 响应格式：{"uuid": "<uuid>"}（必须在 3000ms 内回复，否则 OneID 会重试）
//
// 安全：Pod 暴露公网，必须验证请求来自 Gateway（通过 X-Internal-Token）。
// Gateway 侧已验证 OneID webhook 签名，Pod 侧保留二次验证作为双重保险。
func HandleOneIDEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	// ── 验证 Gateway 内部 Token ──
	if err := verifyInternalRequestToken(r); err != nil {
		slog.Warn("OneID event: internal token verification failed", "err", err, "remote", r.RemoteAddr)
		writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgOneIDUnauthorized))
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Warn("OneID event: failed to read body", "err", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	var event oneIDWebhookEvent
	if err := json.Unmarshal(bodyBytes, &event); err != nil {
		slog.Warn("OneID event: failed to parse body", "err", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	slog.Info("OneID event received", "uuid", event.UUID, "type", event.Type)

	// 解析 data 字段
	var data oneIDEventData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		slog.Warn("OneID event: failed to parse data", "uuid", event.UUID, "type", event.Type, "err", err)
		// 解析失败仍返回 200，避免 OneID 无限重试
		writeOneIDEventOK(w, event.UUID)
		return
	}

	switch event.Type {
	case "member.created":
		handleOneIDMemberCreated(r.Context(), data)
	case "member.updated":
		handleOneIDMemberUpdated(r.Context(), data)
	case "member.deleted":
		handleOneIDMemberDeleted(r.Context(), data)
	case "admin.added":
		handleOneIDAdminAdded(r.Context(), data)
	case "admin.removed":
		handleOneIDAdminRemoved(r.Context(), data)
	default:
		slog.Warn("OneID event: unknown type, ignoring", "uuid", event.UUID, "type", event.Type)
	}

	writeOneIDEventOK(w, event.UUID)
}

// writeOneIDEventOK 回复 OneID Webhook 确认响应。
// OneID 要求必须回复 {"uuid": "<uuid>"}，否则会按退避策略重新投递。
func writeOneIDEventOK(w http.ResponseWriter, uuid string) {
	jsonOK(w, map[string]string{"uuid": uuid})
}

func handleOneIDMemberCreated(ctx context.Context, event oneIDEventData) {
	// 使用 Unscoped() 查找包括已软删除的用户，避免重新启用时唯一索引冲突。
	var existing model.User
	if model.DB(ctx).Unscoped().Where("one_id_sub = ?", event.Sub).First(&existing).Error == nil {
		if existing.DeletedAt.Valid {
			// 用户曾被软删除，现在 OneID 重新添加，恢复该用户。
			if err := model.DB(ctx).Unscoped().Model(&existing).Update("deleted_at", nil).Error; err != nil {
				slog.Error("OneID member.created: failed to restore soft-deleted user", "sub", event.Sub, "err", err)
			} else {
				slog.Info("OneID member.created: restored soft-deleted user", "sub", event.Sub, "username", existing.Username)
				// 恢复后同步用户名（禁用期间 OneID 侧可能已改名）
				if event.Name != "" && event.Name != existing.Username {
					safeUpdateUsername(ctx, &existing, event.Name)
				}
			}
		} else {
			slog.Info("OneID member.created: user already exists, skipping", "sub", event.Sub)
		}
		return
	}
	sub := event.Sub
	username := uniqueUsername(ctx, event.Name, sub, 0)
	cfg := model.GetSiteConfig(ctx)
	user := model.User{
		Username:        username,
		Password:        "",
		Role:            "user",
		InstanceQuota:   cfg.DefaultInstanceQuota,
		TokenQuotaDay:   cfg.DefaultTokenQuotaDay,
		TokenQuotaRules: cfg.DefaultTokenQuotaRules,
		OneIDSub:        &sub,
	}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		slog.Error("OneID member.created: failed to create user", "sub", event.Sub, "username", username, "err", err)
	} else {
		slog.Info("OneID member.created: user created", "username", username, "sub", event.Sub)
	}
}

func handleOneIDMemberUpdated(ctx context.Context, event oneIDEventData) {
	var user model.User
	if model.DB(ctx).Where("one_id_sub = ?", event.Sub).First(&user).Error != nil {
		slog.Warn("OneID member.updated: user not found", "sub", event.Sub)
		return
	}
	// 安全更新 username，避免与其他用户冲突
	if event.Name != "" && event.Name != user.Username {
		safeUpdateUsername(ctx, &user, event.Name)
	}
}

func handleOneIDMemberDeleted(ctx context.Context, event oneIDEventData) {
	var user model.User
	if model.DB(ctx).Where("one_id_sub = ?", event.Sub).First(&user).Error != nil {
		slog.Warn("OneID member.deleted: user not found", "sub", event.Sub)
		return
	}

	assetAction := event.AssetAction
	if assetAction == "" {
		assetAction = "keep"
	}

	switch assetAction {
	case "delete":
		// 删除用户名下所有实例（软删除）
		MarkPersonalSpacesToBeDeletedByUser(ctx, user.ID)
		if err := model.DisableAgentProxyRoutesForUser(ctx, user.ID); err != nil {
			slog.Warn("OneID member.deleted (delete): disable proxy routes failed", "user_id", user.ID, "err", err)
		} else if err := RefreshAllRuleSetsForRequiredRules(ctx); err != nil {
			slog.Warn("OneID member.deleted (delete): refresh proxy security group rules failed", "user_id", user.ID, "err", err)
		}
		model.DB(ctx).Where("user_id = ?", user.ID).Delete(&model.Instance{})
		slog.Info("OneID member.deleted (delete): instances deleted", "user_id", user.ID)

	case "transfer":
		var targetUser model.User
		if model.DB(ctx).Where("one_id_sub = ?", event.TransferToSub).First(&targetUser).Error != nil {
			slog.Warn("OneID member.deleted (transfer): target user not found", "transfer_to_sub", event.TransferToSub)
		} else {
			model.DB(ctx).Model(&model.Instance{}).Where("user_id = ?", user.ID).Update("user_id", targetUser.ID)
			model.DB(ctx).Model(&model.SMHPersonalSpace{}).Where("user_id = ?", user.ID).Updates(map[string]interface{}{
				"user_id":   targetUser.ID,
				"user_name": targetUser.Username,
			})
			slog.Info("OneID member.deleted (transfer): instances transferred", "from_user_id", user.ID, "to_user_id", targetUser.ID)
		}

	default: // "keep" 或其他
		// 保留实例，解除用户绑定（将 user_id 清零）
		MarkPersonalSpacesToBeDeletedByUser(ctx, user.ID)
		model.DB(ctx).Model(&model.Instance{}).Where("user_id = ?", user.ID).Update("user_id", 0)
		slog.Info("OneID member.deleted (keep): instances kept, user binding released", "user_id", user.ID)
	}

	// 尝试硬删除（无资源则物理删除），否则软删除
	if tryHardDeleteUser(ctx, &user) {
		slog.Info("OneID member.deleted: user hard-deleted (no resources)", "sub", event.Sub, "username", user.Username)
	} else if err := model.DB(ctx).Delete(&user).Error; err != nil {
		slog.Error("OneID member.deleted: failed to soft-delete user", "sub", event.Sub, "err", err)
	} else {
		slog.Info("OneID member.deleted: user soft-deleted (has resources)", "sub", event.Sub, "username", user.Username)
	}
}

func handleOneIDAdminAdded(ctx context.Context, event oneIDEventData) {
	if err := model.DB(ctx).Model(&model.User{}).Where("one_id_sub = ?", event.Sub).Update("role", "admin").Error; err != nil {
		slog.Error("OneID admin.added: failed to update role", "sub", event.Sub, "err", err)
	} else {
		slog.Info("OneID admin.added: role updated to admin", "sub", event.Sub)
	}
}

func handleOneIDAdminRemoved(ctx context.Context, event oneIDEventData) {
	if err := model.DB(ctx).Model(&model.User{}).Where("one_id_sub = ?", event.Sub).Update("role", "user").Error; err != nil {
		slog.Error("OneID admin.removed: failed to update role", "sub", event.Sub, "err", err)
	} else {
		slog.Info("OneID admin.removed: role updated to user", "sub", event.Sub)
	}
}

// idTokenClaims 是 OneID id_token payload 中的标准 OIDC claims。
type idTokenClaims struct {
	Sub               string `json:"sub"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
	Tid               string `json:"tid"`  // OneID 租户 ID（同 account_id）
	Sid               string `json:"sid"`  // OneID 登录会话唯一标识，用于单点登出
	Sexp              int64  `json:"sexp"` // OneID 登录态过期时间（Unix 秒），用于黑名单 TTL
}

// parseIDTokenClaims 从 id_token JWT 中解码 payload，不验签（验签由 OneID 负责）。
func parseIDTokenClaims(idToken string) (*idTokenClaims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, hcommon.I18nError(i18n.MsgOneIDAuthInvalidIDTokenFormat)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDAuthBase64DecodePayload)
	}
	var claims idTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDAuthJSONParsePayload)
	}
	if claims.Sub == "" {
		return nil, hcommon.I18nError(i18n.MsgOneIDAuthIDTokenMissingSub)
	}
	return &claims, nil
}

// logoutTokenClaims 是 OneID logout_token JWT payload 中的 claims。
// 参考 OIDC Backchannel Logout 规范及 OneID 文档。
type logoutTokenClaims struct {
	Sub  string `json:"sub"`  // 待登出用户唯一标识
	Tid  string `json:"tid"`  // 租户唯一标识
	Sid  string `json:"sid"`  // OneID 登录会话唯一标识，用于关联应用侧 session
	Exp  int64  `json:"exp"`  // logout_token 过期时间（Unix 秒）
	Sexp int64  `json:"sexp"` // OneID 登录态过期时间（Unix 秒），用作黑名单 TTL 上限
}

// parseLogoutTokenClaims 从 logout_token JWT 中解码 payload，不验签。
func parseLogoutTokenClaims(token string) (*logoutTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, hcommon.I18nError(i18n.MsgOneIDAuthInvalidLogoutTokenFmt)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDAuthBase64DecodePayload)
	}
	var claims logoutTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDAuthJSONParsePayload)
	}
	return &claims, nil
}

// --- Internal Login (Gateway Mode) ---

// verifyInternalRequestToken 验证 Gateway 转发 /spi/* 请求时附加的 X-Internal-Token。
// 格式: timestamp.hex_hmac (HMAC-SHA256(derived_key, timestamp))
// Pod 持有的 InternalSecret 即为 per-tenant 派生密钥（hex 编码的字符串），
// Gateway 签名时直接用 hex 字符串的 []byte 作为 HMAC key，此处保持一致。
//
// 时间窗口: 120 秒（与 Gateway InternalAuth 中间件一致）。
func verifyInternalRequestToken(r *http.Request) error {
	if hcommon.InternalSecretFromCtx(r.Context()) == "" {
		return hcommon.I18nError(i18n.MsgOneIDAuthInternalSecretNotSet)
	}

	token := r.Header.Get("X-Internal-Token")
	if token == "" {
		return hcommon.I18nError(i18n.MsgOneIDAuthMissingInternalToken)
	}

	dotIdx := strings.Index(token, ".")
	if dotIdx < 0 {
		return hcommon.I18nError(i18n.MsgOneIDAuthTokenMissingSeparator)
	}

	tsStr := token[:dotIdx]
	sigHex := token[dotIdx+1:]

	// 解析时间戳并检查时间窗口
	var tsVal int64
	if _, err := fmt.Sscanf(tsStr, "%d", &tsVal); err != nil {
		return hcommon.I18nError(i18n.MsgOneIDAuthInvalidTimestamp)
	}
	now := time.Now().Unix()
	diff := now - tsVal
	if diff < 0 {
		diff = -diff
	}
	if diff > 120 {
		return hcommon.I18nError(i18n.MsgOneIDAuthTokenExpiredDiff, diff)
	}

	// 直接用 InternalSecret 的 []byte 作为 HMAC key（与 Gateway SignInternalRequest 一致）。
	// InternalSecret 是 hex 编码的派生密钥字符串，Gateway 签名时也是直接用此字符串的 bytes。
	expectedSig := hmacSHA256([]byte(hcommon.InternalSecretFromCtx(r.Context())), []byte(tsStr))
	gotSig, err := hex.DecodeString(sigHex)
	if err != nil {
		return hcommon.I18nError(i18n.MsgOneIDAuthInvalidSignatureEnc)
	}

	if !hmac.Equal(gotSig, expectedSig) {
		return hcommon.I18nError(i18n.MsgOneIDAuthSignatureMismatch)
	}

	return nil
}

// internalTokenPayload mirrors the payload signed by the Gateway.
type internalTokenPayload struct {
	Sub   string   `json:"sub"`
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Tid   string   `json:"tid"`
	Sid   string   `json:"sid,omitempty"` // OneID session ID，用于 Backchannel Logout 黑名单关联
	Amr   []string `json:"amr,omitempty"` // 认证方式：sms, pwd, email, sso, wechat, qq 等（来自 id_token）
	Iat   int64    `json:"iat"`
	Exp   int64    `json:"exp"`

	// ── 角色 & 画像（Gateway 侧查询后注入）──
	IsAdmin         bool   `json:"is_admin,omitempty"`
	Mobile          string `json:"mobile,omitempty"`
	Position        string `json:"position,omitempty"`
	EmployeeNumber  string `json:"employee_number,omitempty"`
	Status          string `json:"status,omitempty"`
	DepartmentsJSON string `json:"departments_json,omitempty"`
}

// verifyInternalToken verifies and decodes a Gateway-signed internal token.
// Format: base64url(json_payload).base64url(hmac_sha256)
//
// The secret parameter is a hex-encoded per-tenant derived key
// (produced by Operator as fmt.Sprintf("%x", HMAC(master, tenant_id))).
// We must hex-decode it back to raw bytes before using it as HMAC key.
func verifyInternalToken(secret, token string) (*internalTokenPayload, error) {
	// Find the last dot separator.
	dot := strings.LastIndex(token, ".")
	if dot < 0 {
		return nil, hcommon.I18nError(i18n.MsgOneIDAuthTokenMissingSeparator)
	}

	payloadB64 := token[:dot]
	sigB64 := token[dot+1:]

	// The secret is hex-encoded by Operator's DerivePerTenantSecret();
	// decode it back to raw bytes so the HMAC matches Gateway's signing key.
	secretBytes, err := hex.DecodeString(secret)
	if err != nil {
		// Fallback: treat as raw string (backward compat with non-hex secrets).
		secretBytes = []byte(secret)
	}

	// Verify HMAC-SHA256 signature.
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(payloadB64))
	expectedSig := mac.Sum(nil)

	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDAuthDecodeSignatureFailed)
	}

	if !hmac.Equal(sig, expectedSig) {
		return nil, hcommon.I18nError(i18n.MsgOneIDAuthInvalidSignature)
	}

	// Decode payload.
	payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDAuthDecodePayloadFailed)
	}

	var p internalTokenPayload
	if err := json.Unmarshal(payloadJSON, &p); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDAuthUnmarshalPayloadFailed)
	}

	// Check expiry.
	if time.Now().Unix() > p.Exp {
		return nil, hcommon.I18nError(i18n.MsgOneIDAuthTokenExpired)
	}

	return &p, nil
}

// HandleInternalLogin handles GET /auth/internal-login.
// This endpoint is called by the Gateway after successful OIDC authentication.
// It verifies the HMAC-signed internal token, creates/finds the local user,
// syncs admin role, establishes a session, and redirects to the application.
func HandleInternalLogin(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()

	if hcommon.InternalSecretFromCtx(r.Context()) == "" {
		slog.Error("internal-login: InternalSecret not configured")
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInternalLoginNotConfig))
		return
	}

	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInternalLoginMissToken))
		return
	}

	// Verify the internal token.
	payload, err := verifyInternalToken(hcommon.InternalSecretFromCtx(r.Context()), tokenStr)
	if err != nil {
		slog.Warn("internal-login: token verification failed", "err", err)
		writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgInternalLoginTokenInval))
		return
	}

	sub := payload.Sub
	name := payload.Name
	accountID := payload.Tid

	if sub == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInternalLoginMissSub))
		return
	}

	slog.Info("internal-login: user authenticated",
		"sub", sub, "name", name, "email", payload.Email, "account_id", accountID)

	// Sync user profile from token payload (Gateway already queried OneID API).
	// This replaces the old SyncOneIDUserProfile() which required Hatchery→OneID connectivity.
	if sub != "" && payload.DepartmentsJSON != "" {
		// 多租户阶段一：保留 TenantSnapshot，脱离 r.Context() 的 deadline
		go func(ctx context.Context) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("syncProfileFromToken: recovered from panic", "sub", sub, "panic", r)
				}
			}()
			syncProfileFromToken(ctx, sub, payload)
		}(hcommon.DetachContext(r.Context()))
	}

	// Find or create local user.
	// 使用 Unscoped() 绕过 GORM 软删除过滤，以便找到已被软删除但在 OneID 侧重新启用的用户。
	var user model.User
	isNewOrRestored := false // 标记是否为新创建或恢复的用户，用于后续画像补全
	result := model.DB(r.Context()).Unscoped().Where("one_id_sub = ?", sub).First(&user)
	if result.Error != nil {
		// New user, auto-create.
		subCopy := sub
		newRole := "user"
		if payload.IsAdmin {
			newRole = "admin"
		}
		username := uniqueUsername(r.Context(), name, sub, 0)
		cfg := model.GetSiteConfig(r.Context())
		user = model.User{
			Username:        username,
			Email:           payload.Email,
			Password:        "",
			Role:            newRole,
			InstanceQuota:   cfg.DefaultInstanceQuota,
			TokenQuotaDay:   cfg.DefaultTokenQuotaDay,
			TokenQuotaRules: cfg.DefaultTokenQuotaRules,
			OneIDSub:        &subCopy,
		}
		if err := model.DB(r.Context()).Create(&user).Error; err != nil {
			slog.Error("internal-login: failed to create user", "sub", sub, "username", username, "err", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInternalLoginCreateUser))
			return
		}
		slog.Info("internal-login: created new user", "username", user.Username, "sub", sub, "role", newRole)

		// P0 Fix: Try to assign user to main department group if available
		if payload.DepartmentsJSON != "" {
			var depts []model.OneIDDepartment
			if json.Unmarshal([]byte(payload.DepartmentsJSON), &depts) == nil {
				for _, d := range depts {
					if d.IsMainDepartment && d.DepartmentID != "" {
						// Find group corresponding to this OneID department
						if deptGroup, err := model.GroupBySourceRef(r.Context(), model.GroupSourceOneIDDept, d.DepartmentID); err == nil && deptGroup != nil {
							// Assign user to group
							if err := model.UpdateUserGroupMemberships(model.DB(r.Context()), user.ID, []uint{deptGroup.ID}); err != nil {
								slog.Warn("internal-login: failed to assign user to department group", "sub", sub, "user_id", user.ID, "group_id", deptGroup.ID, "err", err)
							} else {
								slog.Info("internal-login: assigned user to department group", "sub", sub, "user_id", user.ID, "group_id", deptGroup.ID, "group_name", deptGroup.Name)
							}
						}
						break
					}
				}
			}
		}

		isNewOrRestored = true
	} else if user.DeletedAt.Valid {
		// 用户曾被软删除，现在在 OneID 侧重新启用，恢复该用户。
		if err := model.DB(r.Context()).Unscoped().Model(&user).Update("deleted_at", nil).Error; err != nil {
			slog.Error("internal-login: failed to restore soft-deleted user", "sub", sub, "err", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInternalLoginRestoreUser))
			return
		}
		slog.Info("internal-login: restored soft-deleted user", "username", user.Username, "sub", sub, "user_id", user.ID)
		// 恢复后同步用户名（禁用期间 OneID 侧可能已改名）
		if name != "" && user.Username != name {
			safeUpdateUsername(r.Context(), &user, name)
		}
		isNewOrRestored = true
	} else {
		// Existing user — sync display name from OneID on every login,
		// so name changes in OneID are reflected here automatically.
		// 使用安全更新，避免与其他用户的 username 冲突。
		if name != "" && user.Username != name {
			safeUpdateUsername(r.Context(), &user, name)
		}
	}

	// 新用户创建或恢复软删除用户时，异步通过 Gateway 拉取完整用户信息（部门、手机号等）。
	// 作为 syncProfileFromToken 的 fallback：当 token payload 未携带 DepartmentsJSON 时，
	// 通过 batch_query_users 补齐画像数据，确保用户登录后画像完整。
	if isNewOrRestored && GatewayURL != "" && accountID != "" {
		// 多租户阶段一：保留 TenantSnapshot，脱离 r.Context() 的 deadline
		go func(ctx context.Context, unionID, enterpriseID string) {
			batchBody, err := gwCallContacts(ctx, gwContactsRequest{
				Action:    "batch_query_users",
				AccountID: enterpriseID,
				UnionIDs:  []string{unionID},
			})
			if err != nil {
				slog.Warn("internal-login: gateway batch_query_users failed", "sub", unionID, "err", err)
				return
			}
			var batchResult struct {
				Code int    `json:"code"`
				Msg  string `json:"msg"`
				Data struct {
					Users []gwUserInfo `json:"users"`
				} `json:"data"`
			}
			if json.Unmarshal(batchBody, &batchResult) == nil && batchResult.Code == 0 && len(batchResult.Data.Users) > 0 {
				u := batchResult.Data.Users[0]
				deptJSON, _ := json.Marshal(u.Departments)
				profile := &model.OneIDUserProfile{
					OneIDSub:        u.UnionID,
					UnionID:         u.UnionID,
					Name:            u.Name,
					Email:           u.Email,
					Mobile:          u.Mobile,
					Position:        u.Position,
					EmployeeNumber:  u.EmployeeNumber,
					Status:          u.Status,
					DepartmentsJSON: string(deptJSON),
				}
				for _, d := range u.Departments {
					if d.IsMainDepartment {
						profile.MainDeptID = d.DepartmentID
						profile.MainDeptName = d.DepartmentName
						profile.MainDeptParentID = d.DepartmentParentID
						break
					}
				}
				if err := model.UpsertOneIDUserProfile(ctx, profile); err != nil {
					slog.Warn("internal-login: upsert profile failed", "sub", unionID, "err", err)
				} else {
					slog.Info("internal-login: user profile synced via gateway", "sub", unionID)
				}
			}
		}(hcommon.DetachContext(r.Context()), sub, accountID)
	}

	// Sync admin role from token payload (Gateway already checked via OneID API).
	// This replaces the old CheckIsAdminRole() which required Hatchery→OneID connectivity.
	newRole := "user"
	if payload.IsAdmin {
		newRole = "admin"
	}
	if user.Role != newRole {
		model.DB(r.Context()).Model(&user).Update("role", newRole)
		user.Role = newRole
		slog.Info("internal-login: role synced from token", "sub", sub, "role", newRole)
	}

	// Establish session.
	session := getSession(r)
	session.Values["username"] = user.Username
	session.Values["role"] = user.Role
	session.Values["user_id"] = user.ID
	session.Values["login_at"] = time.Now().Unix() // 用于 sub 维度黑名单的时间比较
	if sub != "" {
		session.Values["oneid_sub"] = sub
	}
	if payload.Sid != "" {
		session.Values["oneid_sid"] = payload.Sid
	}
	if len(payload.Amr) > 0 {
		amrJSON, _ := json.Marshal(payload.Amr)
		session.Values["oneid_amr"] = string(amrJSON)
	}
	// 多租户阶段一：OneID 登录成功后也写入 identifier 防跨租户冒用
	setSessionIdentifier(session, r.Context())
	delete(session.Values, "login_failures")
	session.Save(r, w)

	// Audit log.
	go model.LogAudit(hcommon.DetachContext(r.Context()), startedAt, user.ID, user.Username, "internal_login", "session", "", "success")

	// IDP-initiated 登录时 Gateway 会透传 redirect_to（来自 target_link_uri）。
	// 使用 validateAndGetSafeRedirectURL 进行安全验证和路由规范化。
	redirectToParam := r.URL.Query().Get("redirect_to")
	slog.Info("internal-login: redirect_to parameter", "redirect_to", redirectToParam)
	redirectTo := validateAndGetSafeRedirectURL(redirectToParam, "/my-openclaw")
	slog.Info("internal-login: final redirect", "redirectTo", redirectTo)
	// NOCA:open_redirect(已经在validateAndGetSafeRedirectURL中处理)
	http.Redirect(w, r, redirectTo, http.StatusFound) // nolint: gosec
}

// syncProfileFromToken 从 InternalToken payload 中提取用户画像信息，
// 写入 oneid_user_profiles 表。这替代了原来需要 Hatchery→OneID 网络连通的
// SyncOneIDUserProfile()，画像数据现在由 Gateway 在登录时查询后通过 token 传递。
func syncProfileFromToken(ctx context.Context, sub string, p *internalTokenPayload) {
	profile := &model.OneIDUserProfile{
		OneIDSub:       sub,
		UnionID:        sub,
		Name:           p.Name,
		Email:          p.Email,
		Mobile:         p.Mobile,
		Position:       p.Position,
		EmployeeNumber: p.EmployeeNumber,
		Status:         p.Status,
	}

	if p.DepartmentsJSON != "" {
		profile.DepartmentsJSON = p.DepartmentsJSON

		// 解析部门列表，提取主部门信息
		var depts []model.OneIDDepartment
		if json.Unmarshal([]byte(p.DepartmentsJSON), &depts) == nil {
			for _, d := range depts {
				if d.IsMainDepartment {
					profile.MainDeptID = d.DepartmentID
					profile.MainDeptName = d.DepartmentName
					profile.MainDeptParentID = d.DepartmentParentID
					break
				}
			}
		}
	}

	if err := model.UpsertOneIDUserProfile(ctx, profile); err != nil {
		slog.Error("syncProfileFromToken: upsert failed", "sub", sub, "err", err)
	} else {
		slog.Info("syncProfileFromToken: profile synced from token", "sub", sub)
	}
}
