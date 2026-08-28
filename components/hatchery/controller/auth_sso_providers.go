package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"hatchery/common"
)

// SsoIdpProvider 表示 OneID 返回的一个 SSO 认证源（aliasId 非空的可直达 IdP）。
type SsoIdpProvider struct {
	ID         string `json:"id"`         // OneID 认证源唯一 ID（同一 uniqueName 可能配置多个，用此区分）
	Name       string `json:"name"`
	LogoURL    string `json:"logo_url"`
	AliasID    string `json:"alias_id"`
	UniqueName string `json:"unique_name"`
	FlowType   string `json:"flow_type"` // Redirect（跳转外部 IdP，如飞书）/ Delegation（OneID 内完成，如 OpenLDAP）
}

// ssoProvidersClient 是调用 Gateway /api/sso-providers 的全局 HTTP 客户端。
// 复用底层 TCP 连接（默认 Transport 自带连接池），减少握手开销。
var ssoProvidersClient = &http.Client{Timeout: 10 * time.Second}

// ssoProvidersResponse 是 /auth/sso-providers 的响应结构。
type ssoProvidersResponse struct {
	Providers       []SsoIdpProvider `json:"providers"`        // 仅 flowType=Redirect 的 IdP（如飞书、企微）
	PasswordEnabled bool             `json:"password_enabled"` // 是否配置了密码登录（uniqueName=PASSWORD_V0）
}

// emptySsoProvidersResponse 返回容错响应（空 providers + 密码登录关闭）。
// 当 OneID 接口失败/Gateway 不可用/租户未配 OneID 时，保守起见关闭密码登录入口，
// 避免在 OneID 未真正配置密码登录的情况下误导用户去尝试。
func emptySsoProvidersResponse() ssoProvidersResponse {
	return ssoProvidersResponse{
		Providers:       []SsoIdpProvider{},
		PasswordEnabled: false,
	}
}

// oneIDAuthnSsoResponse 是 OneID GET /v1/authn/sso 接口的响应结构。
type oneIDAuthnSsoResponse struct {
	Next struct {
		Type       string `json:"type"`
		SsoOptions struct {
			IdProviders []struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				LogoURL    string `json:"logoUrl"`
				UniqueName string `json:"uniqueName"`
				FlowType   string `json:"flowType"`
				AliasID    string `json:"aliasId"`
			} `json:"idProviders"`
		} `json:"ssoOptions"`
	} `json:"next"`
}

// HandleSsoProviders 返回当前租户在 OneID 上配置的 SSO 认证源列表。
// GET /auth/sso-providers — 前端打开登录弹窗时调用，用于渲染 IdP 卡片。
// 通过 Gateway /api/sso-providers 代理调用 OneID /v1/authn/sso 接口。
func HandleSsoProviders(w http.ResponseWriter, r *http.Request) {
	accountID := common.TenantIDFromCtx(r.Context())
	if accountID == "" {
		slog.Warn("sso-providers: tenant account_id is empty, returning empty list")
		jsonOK(w, emptySsoProvidersResponse())
		return
	}

	if GatewayURL == "" {
		slog.Warn("sso-providers: GatewayURL is empty, returning empty list")
		jsonOK(w, emptySsoProvidersResponse())
		return
	}

	slog.Info("sso-providers: calling Gateway", "gateway_url", GatewayURL, "account_id", accountID)

	// 通过 Gateway 代理调用 OneID（Hatchery Pod 不直连 OneID）
	apiURL := fmt.Sprintf("%s/api/sso-providers?tenant=%s",
		strings.TrimRight(GatewayURL, "/"), url.QueryEscape(accountID))
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, apiURL, nil)
	if err != nil {
		slog.Warn("sso-providers: failed to build request", "err", err)
		jsonOK(w, emptySsoProvidersResponse())
		return
	}

	// 携带内部认证头（Hatchery 持有 per-tenant 派生密钥）
	// accountID 在前面已校验非空，可直接复用
	if internalSecret := common.InternalSecretFromCtx(r.Context()); internalSecret != "" {
		req.Header.Set("X-Internal-Token", signInternalRequest(internalSecret))
		req.Header.Set("X-Internal-Tenant", accountID)
	}

	resp, err := ssoProvidersClient.Do(req)
	if err != nil {
		slog.Warn("sso-providers: Gateway request failed", "err", err)
		jsonOK(w, emptySsoProvidersResponse())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 仅读取前 512 字节用于排查，避免大响应体写入日志
		bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		slog.Warn("sso-providers: Gateway returned non-200",
			"status", resp.StatusCode, "body_snippet", string(bodySnippet))
		jsonOK(w, emptySsoProvidersResponse())
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("sso-providers: failed to read response", "err", err)
		jsonOK(w, emptySsoProvidersResponse())
		return
	}

	var ssoResp oneIDAuthnSsoResponse
	if err := json.Unmarshal(body, &ssoResp); err != nil {
		slog.Warn("sso-providers: failed to parse response", "err", err)
		jsonOK(w, emptySsoProvidersResponse())
		return
	}

	// 解析 OneID 返回的认证源列表：
	//   - aliasId 非空 → 可直达的 IdP（飞书 Redirect、OpenLDAP Delegation 等都带 aliasId），
	//     加入 providers，前端点击后带 idp=<aliasId> 跳转直达
	//   - uniqueName=PASSWORD_V0 → 密码登录（aliasId 为空，不能直达），
	//     标记 password_enabled=true 供前端展示密码登录入口
	var providers []SsoIdpProvider
	passwordEnabled := false
	for _, idp := range ssoResp.Next.SsoOptions.IdProviders {
		if idp.AliasID != "" {
			providers = append(providers, SsoIdpProvider{
				ID:         idp.ID,
				Name:       idp.Name,
				LogoURL:    idp.LogoURL,
				AliasID:    idp.AliasID,
				UniqueName: idp.UniqueName,
				FlowType:   idp.FlowType,
			})
		}
		if idp.UniqueName == "PASSWORD_V0" {
			passwordEnabled = true
		}
	}

	if providers == nil {
		providers = []SsoIdpProvider{}
	}

	jsonOK(w, ssoProvidersResponse{
		Providers:       providers,
		PasswordEnabled: passwordEnabled,
	})
}
