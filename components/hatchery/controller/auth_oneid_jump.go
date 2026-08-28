package controller

// auth_oneid_jump.go — 免登跳转到 OneID 管理后台/个人中心/工作台。
//
// GET /auth/oneid/jump?module=admin&route=users
//
// 前端调用此接口，Hatchery 通过 Gateway 的 /api/sso-link 代理端点
// 向 OneID 获取一个带时效的免登链接，返回给前端供跳转。
//
// 响应：
// - 成功：{ "link": "https://...", "expires_in": 300 }
// - 失败：{ "error": "xxx" }

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
)

// HandleOneIDJump 处理 GET /auth/oneid/jump?module=admin&route=users
// 需要已登录（admin 权限），返回 OneID 免登链接。
func HandleOneIDJump(w http.ResponseWriter, r *http.Request) {
	if GatewayURL == "" {
		writeError(w, r, http.StatusServiceUnavailable, hcommon.I18nError(i18n.MsgGatewayNotConfigured))
		return
	}

	// 鉴权：获取当前登录用户，并校验 admin 角色
	user, err := getLoginUser(r)
	if err != nil || user == nil {
		writeError(w, r, http.StatusUnauthorized, ErrUnauthorized)
		return
	}
	if user.Role != "admin" {
		writeError(w, r, http.StatusForbidden, ErrAdminRequired)
		return
	}

	// 获取 OneID sub 和 enterprise ID
	sub := ""
	if user.OneIDSub != nil {
		sub = *user.OneIDSub
	}
	if sub == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserNotLinkedToOneID))
		return
	}

	accountID := hcommon.TenantIDFromCtx(r.Context())
	if accountID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgOneIDTenantNotConfigured))
		return
	}

	// 读取参数
	module := r.URL.Query().Get("module")
	if module == "" {
		module = "admin" // 默认跳转管理后台
	}
	route := r.URL.Query().Get("route")

	// 获取客户端 IP
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}

	ua := r.Header.Get("User-Agent")

	// 从 session 读取登录时的认证方式（amr），传给 Gateway 用于免登链接
	var amr []string
	session := getSession(r)
	if amrJSON, ok := session.Values["oneid_amr"].(string); ok && amrJSON != "" {
		json.Unmarshal([]byte(amrJSON), &amr)
	}

	// 构造请求体
	reqPayload := map[string]interface{}{
		"account_id": accountID,
		"union_id":   sub,
		"module":     module,
		"route":      route,
		"ip":         ip,
		"ua":         ua,
	}
	if len(amr) > 0 {
		reqPayload["amr"] = amr
	}
	reqBody, _ := json.Marshal(reqPayload)

	// 调用 Gateway /api/sso-link（携带内部认证头）
	apiURL := GatewayURL + "/api/sso-link"
	httpReq, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		slog.Error("oneid/jump: build request failed", "err", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInternalError))
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if hcommon.InternalSecretFromCtx(r.Context()) != "" {
		httpReq.Header.Set("X-Internal-Token", signInternalRequest(hcommon.InternalSecretFromCtx(r.Context())))
		if hcommon.TenantIDFromCtx(r.Context()) != "" {
			httpReq.Header.Set("X-Internal-Tenant", hcommon.TenantIDFromCtx(r.Context()))
		}
	}

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(httpReq)
	if err != nil {
		slog.Error("oneid/jump: gateway request failed", "url", apiURL, "err", err)
		writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDGatewayRequestFailed))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		slog.Error("oneid/jump: gateway returned error", "status", resp.StatusCode)
		writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgFetchSSOLinkFailed, resp.StatusCode))
		return
	}

	// 原样转发 Gateway/OneID 响应（包含 link 和 expires_in）
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}
