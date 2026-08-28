package controller

// oneid_unified.go — 统一账号模式：OneID 自建应用用户管理。
//
// 统一账号模式（OneIDAppID 非空）下，用户管理操作需双写 OneID。
// - OpenAPI 接口（用户 CRUD）：Hatchery 直调 OneID，使用自建应用 Token
// - Internal 接口（角色/密码）：通过 Gateway 代理，使用套件 tenant Token
//
// 本文件封装：
//   1. 自建应用 Token 获取 + 内存缓存
//   2. OneID OpenAPI 用户管理接口调用

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// oneIDAPIBaseURL 是 OneID OpenAPI 的基础域名（国内）。
// 生产默认值为 https://api.account.tencent.com，可通过环境变量 ONEID_API_BASE_URL 覆盖。
var oneIDAPIBaseURL = "https://api.account.tencent.com"

// oneIDAPIBaseURLOverseas 是 OneID OpenAPI 的基础域名（海外）。
// 可通过环境变量 ONEID_API_BASE_URL_OVERSEAS 覆盖。
var oneIDAPIBaseURLOverseas = "https://api.account.tencentid.com"

// oneIDWorkspaceBaseURL 是 OneID Workspace API 的基础域名（国内，用于 session 级别接口如 /v1/authn:get_self_v3）。
// 生产默认值为 https://api.workspace.tencent.com，可通过环境变量 ONEID_WORKSPACE_BASE_URL 覆盖。
var oneIDWorkspaceBaseURL = "https://api.workspace.tencent.com"

// oneIDWorkspaceBaseURLOverseas 是 OneID Workspace API 的基础域名（海外）。
// 可通过环境变量 ONEID_WORKSPACE_BASE_URL_OVERSEAS 覆盖。
var oneIDWorkspaceBaseURLOverseas = "https://api.workspace.tencentid.com"

func init() {
	if v := os.Getenv("ONEID_API_BASE_URL"); v != "" {
		oneIDAPIBaseURL = v
	}
	if v := os.Getenv("ONEID_API_BASE_URL_OVERSEAS"); v != "" {
		oneIDAPIBaseURLOverseas = v
	}
	if v := os.Getenv("ONEID_WORKSPACE_BASE_URL"); v != "" {
		oneIDWorkspaceBaseURL = v
	}
	if v := os.Getenv("ONEID_WORKSPACE_BASE_URL_OVERSEAS"); v != "" {
		oneIDWorkspaceBaseURLOverseas = v
	}
}

// getOneIDAPIBaseURL 根据当前租户的 lang 返回对应的 OneID OpenAPI base URL。
// lang=en（海外）使用 oneIDAPIBaseURLOverseas，否则使用 oneIDAPIBaseURL。
func getOneIDAPIBaseURL(ctx context.Context) string {
	if IsOverseasFromCtx(ctx) {
		return oneIDAPIBaseURLOverseas
	}
	return oneIDAPIBaseURL
}

// getOneIDWorkspaceBaseURL 根据当前租户的 lang 返回对应的 OneID Workspace base URL。
func getOneIDWorkspaceBaseURL(ctx context.Context) string {
	if IsOverseasFromCtx(ctx) {
		return oneIDWorkspaceBaseURLOverseas
	}
	return oneIDWorkspaceBaseURL
}

// ── 自建应用 Token 缓存 ──────────────────────────────────────────────────────

type appTokenEntry struct {
	accessToken string
	expiresAt   time.Time
}

var (
	appTokenMu    sync.Mutex
	appTokenCache = map[string]*appTokenEntry{} // key: client_id
)

// getOneIDAppToken 获取自建应用的 access_token（client_credentials 模式）。
// 内存缓存，提前 60s 刷新。
func getOneIDAppToken(ctx context.Context) (string, error) {
	snap, ok := hcommon.GetTenantSnapshot(ctx)
	if !ok || snap.OneIDClientID == "" || snap.OneIDClientSecret == "" || snap.OneIDTokenEndpoint == "" {
		return "", hcommon.I18nError(i18n.MsgOneIDCredsNotConfigured)
	}

	slog.Info("[OneID] getAppToken",
		"token_endpoint", snap.OneIDTokenEndpoint,
		"client_id", snap.OneIDClientID,
		"api_base_url", getOneIDAPIBaseURL(ctx),
	)

	appTokenMu.Lock()
	defer appTokenMu.Unlock()

	entry := appTokenCache[snap.OneIDClientID]
	if entry != nil && time.Now().Before(entry.expiresAt.Add(-60*time.Second)) {
		return entry.accessToken, nil
	}

	// 请求新 Token（form-urlencoded 格式）
	form := url.Values{}
	form.Set("client_id", snap.OneIDClientID)
	form.Set("client_secret", snap.OneIDClientSecret)
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, snap.OneIDTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		slog.Error("[OneID] 构建获取 Token 请求失败",
			"token_endpoint", snap.OneIDTokenEndpoint, "client_id", snap.OneIDClientID, "err", err)
		return "", hcommon.I18nRichError(err, i18n.MsgOneIDBuildTokenRequestFailed)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setAcceptLanguageHeader(req, ctx)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		slog.Error("[OneID] 请求 Token 端点失败",
			"token_endpoint", snap.OneIDTokenEndpoint, "client_id", snap.OneIDClientID, "err", err)
		return "", hcommon.I18nRichError(err, i18n.MsgOneIDTokenRequestFailed)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		slog.Error("[OneID] Token 端点返回非 200",
			"token_endpoint", snap.OneIDTokenEndpoint, "client_id", snap.OneIDClientID,
			"status", resp.StatusCode, "body", safeBodySnippet(body, 512))
		return "", hcommon.I18nError(i18n.MsgOneIDTokenEndpointError, resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.AccessToken == "" {
		slog.Error("[OneID] 解析 Token 响应失败",
			"token_endpoint", snap.OneIDTokenEndpoint, "client_id", snap.OneIDClientID,
			"err", err, "body", safeBodySnippet(body, 512))
		return "", hcommon.I18nError(i18n.MsgOneIDParseTokenFailed, string(body))
	}

	appTokenCache[snap.OneIDClientID] = &appTokenEntry{
		accessToken: result.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}

	slog.Info("[OneID] app token refreshed", "client_id", snap.OneIDClientID, "expires_in", result.ExpiresIn, "token_prefix", result.AccessToken[:min(20, len(result.AccessToken))])
	return result.AccessToken, nil
}

// ── OneID OpenAPI 用户管理 ───────────────────────────────────────────────────

// 根部门 ID 缓存（每个 client_id 一个，启动后首次查询时填充）
var (
	rootDeptMu    sync.Mutex
	rootDeptCache = map[string]string{} // key: client_id → root department_id
)

// getOneIDRootDepartmentID 获取 OneID 根部门 ID（缓存）。
func getOneIDRootDepartmentID(ctx context.Context, token string) (string, error) {
	snap, _ := hcommon.GetTenantSnapshot(ctx)

	rootDeptMu.Lock()
	if cached, ok := rootDeptCache[snap.OneIDClientID]; ok {
		rootDeptMu.Unlock()
		return cached, nil
	}
	rootDeptMu.Unlock()

	apiURL := getOneIDAPIBaseURL(ctx) + "/openapi/v3/contacts/departments/roots"
	respBody, err := oneIDAPICall(ctx, http.MethodGet, apiURL, token, nil)
	if err != nil {
		return "", err
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Roots []struct {
				DepartmentID string `json:"department_id"`
			} `json:"roots"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgOneIDParseRootsFailed)
	}
	if result.Code != 0 {
		return "", hcommon.I18nError(i18n.MsgOneIDAPIError, result.Code, result.Msg)
	}
	if len(result.Data.Roots) == 0 {
		return "", nil
	}

	rootID := result.Data.Roots[0].DepartmentID
	rootDeptMu.Lock()
	rootDeptCache[snap.OneIDClientID] = rootID
	rootDeptMu.Unlock()

	slog.Info("[OneID] root department cached", "department_id", rootID)
	return rootID, nil
}

// OneIDCreateUserReq 创建用户请求参数。
type OneIDCreateUserReq struct {
	Name          string   `json:"name"`
	Username      string   `json:"username"`
	Email         string   `json:"email,omitempty"`
	Password      string   `json:"password,omitempty"`
	DepartmentIDs []string `json:"department_ids,omitempty"`
}

// OneIDCreateUserResp 创建用户响应。
type OneIDCreateUserResp struct {
	UnionID  string `json:"union_id"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// OneIDCreateUser 调用 OneID OpenAPI 创建用户。
// 成功返回 union_id 等信息。
// 如果未指定 department_ids，自动查询根部门并填入。
func OneIDCreateUser(ctx context.Context, req OneIDCreateUserReq) (*OneIDCreateUserResp, error) {
	token, err := getOneIDAppToken(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDGetAppTokenFailed)
	}

	// 如果未传 department_ids，自动获取根部门
	if len(req.DepartmentIDs) == 0 {
		rootID, err := getOneIDRootDepartmentID(ctx, token)
		if err != nil {
			slog.Warn("[OneID] get root department failed, creating user without department", "err", err)
		} else if rootID != "" {
			req.DepartmentIDs = []string{rootID}
		}
	}

	apiURL := getOneIDAPIBaseURL(ctx) + "/openapi/v3/contacts/users"
	respBody, err := oneIDAPICall(ctx, http.MethodPost, apiURL, token, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code int                  `json:"code"`
		Msg  string               `json:"msg"`
		Data *OneIDCreateUserResp `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDParseCreateUserFailed)
	}
	if result.Code != 0 {
		return nil, hcommon.I18nError(i18n.MsgOneIDAPIError, result.Code, result.Msg)
	}
	return result.Data, nil
}

// OneIDDeleteUser 调用 OneID OpenAPI 删除用户（硬删除）。
func OneIDDeleteUser(ctx context.Context, unionID, appID string) error {
	token, err := getOneIDAppToken(ctx)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgOneIDGetAppTokenFailed)
	}

	apiURL := getOneIDAPIBaseURL(ctx) + "/openapi/v3/contacts/users/" + unionID
	payload := map[string]interface{}{
		"legacies": []map[string]interface{}{
			{
				"resolve_method":          "reserve",
				"transfer_to_entity_type": "user",
				"app_id":                  appID,
				"app_type":                "clawpro",
				"resources":               []string{"clawpro/openclaw"},
			},
		},
	}
	_, err = oneIDAPICall(ctx, http.MethodDelete, apiURL, token, payload)
	if err != nil {
		return err
	}
	return nil
}

// OneIDDisableUser 调用 OneID OpenAPI 停用用户（软删除）。
func OneIDDisableUser(ctx context.Context, unionIDs []string) error {
	token, err := getOneIDAppToken(ctx)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgOneIDGetAppTokenFailed)
	}

	apiURL := getOneIDAPIBaseURL(ctx) + "/openapi/v3/contacts/users/batch_disable"
	payload := map[string]interface{}{
		"union_ids": unionIDs,
	}
	_, err = oneIDAPICall(ctx, http.MethodPost, apiURL, token, payload)
	if err != nil {
		return err
	}
	return nil
}

// OneIDEnableUser 调用 OneID OpenAPI 启用用户。
func OneIDEnableUser(ctx context.Context, unionID string) error {
	token, err := getOneIDAppToken(ctx)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgOneIDGetAppTokenFailed)
	}

	apiURL := getOneIDAPIBaseURL(ctx) + "/openapi/v3/contacts/users/" + unionID + "/enable"
	_, err = oneIDAPICall(ctx, http.MethodPost, apiURL, token, nil)
	if err != nil {
		return err
	}
	return nil
}

// OneIDUpdateUser 调用 OneID OpenAPI 更新用户信息。
func OneIDUpdateUser(ctx context.Context, unionID string, fields map[string]interface{}) error {
	token, err := getOneIDAppToken(ctx)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgOneIDGetAppTokenFailed)
	}

	apiURL := getOneIDAPIBaseURL(ctx) + "/openapi/v3/contacts/users/" + unionID
	_, err = oneIDAPICall(ctx, http.MethodPatch, apiURL, token, fields)
	if err != nil {
		return err
	}
	return nil
}

// OneIDAddRoleUsers 通过 Gateway 为用户添加角色（Internal 接口需套件 Token）。
func OneIDAddRoleUsers(ctx context.Context, unionIDs []string) error {
	if GatewayURL == "" {
		return hcommon.I18nError(i18n.MsgOneIDGatewayURLNotConfigured)
	}
	accountID := hcommon.TenantIDFromCtx(ctx)
	if accountID == "" {
		return hcommon.I18nError(i18n.MsgOneIDAccountIDNotAvailable)
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"account_id": accountID,
		"role_id":    "1400000",
		"union_ids":  unionIDs,
	})

	apiURL := GatewayURL + "/api/add-role-users"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgOneIDBuildRequestFailed)
	}
	setGatewayHeaders(req, ctx, accountID)

	slog.Info("[Gateway] API call", "url", apiURL, "has_secret", hcommon.InternalSecretFromCtx(ctx) != "")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		slog.Error("[Gateway] request failed", "url", apiURL, "err", err)
		return hcommon.I18nRichError(err, i18n.MsgOneIDGatewayRequestFailed)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		slog.Error("[Gateway] API error", "url", apiURL, "status", resp.StatusCode, "body", safeBodySnippet(body, 512))
		return hcommon.I18nError(i18n.MsgOneIDGatewayReturnedError, resp.StatusCode, body)
	}
	return nil
}

// OneIDRemoveRoleUsers 通过 Gateway 为用户移除角色（Internal 接口需套件 Token）。
func OneIDRemoveRoleUsers(ctx context.Context, unionIDs []string) error {
	if GatewayURL == "" {
		return hcommon.I18nError(i18n.MsgOneIDGatewayURLNotConfigured)
	}
	accountID := hcommon.TenantIDFromCtx(ctx)
	if accountID == "" {
		return hcommon.I18nError(i18n.MsgOneIDAccountIDNotAvailable)
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"account_id": accountID,
		"role_id":    "1400000",
		"union_ids":  unionIDs,
	})

	apiURL := GatewayURL + "/api/remove-role-users"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgOneIDBuildRequestFailed)
	}
	setGatewayHeaders(req, ctx, accountID)

	slog.Info("[Gateway] API call", "url", apiURL, "has_secret", hcommon.InternalSecretFromCtx(ctx) != "")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		slog.Error("[Gateway] request failed", "url", apiURL, "err", err)
		return hcommon.I18nRichError(err, i18n.MsgOneIDGatewayRequestFailed)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		slog.Error("[Gateway] API error", "url", apiURL, "status", resp.StatusCode, "body", safeBodySnippet(body, 512))
		return hcommon.I18nError(i18n.MsgOneIDGatewayReturnedError, resp.StatusCode, body)
	}
	return nil
}

// OneIDResetPassword 通过 Gateway 重置用户密码（Internal 接口需套件 Token）。
func OneIDResetPassword(ctx context.Context, unionID, password string) error {
	if GatewayURL == "" {
		return hcommon.I18nError(i18n.MsgOneIDGatewayURLNotConfigured)
	}
	accountID := hcommon.TenantIDFromCtx(ctx)
	if accountID == "" {
		return hcommon.I18nError(i18n.MsgOneIDAccountIDNotAvailable)
	}

	payload, _ := json.Marshal(map[string]string{
		"account_id": accountID,
		"union_id":   unionID,
		"password":   password,
	})

	apiURL := GatewayURL + "/api/reset-password"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgOneIDBuildRequestFailed)
	}
	setGatewayHeaders(req, ctx, accountID)

	slog.Info("[Gateway] API call", "url", apiURL, "has_secret", hcommon.InternalSecretFromCtx(ctx) != "")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		slog.Error("[Gateway] request failed", "url", apiURL, "err", err)
		return hcommon.I18nRichError(err, i18n.MsgOneIDGatewayRequestFailed)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		slog.Error("[Gateway] API error", "url", apiURL, "status", resp.StatusCode, "body", safeBodySnippet(body, 512))
		return hcommon.I18nError(i18n.MsgOneIDGatewayReturnedError, resp.StatusCode, body)
	}
	return nil
}

// ── 批量密码重置（迁移工具专用）────────────────────────────────────────────────

// PasswordResetItem 单条密码重置项。
type PasswordResetItem struct {
	UnionID  string `json:"union_id"`
	Password string `json:"password"` // JSON string: {"hash":{"algorithm":"Bcrypt","value":"..."}}
}

// BatchResetFailure 单条失败记录。
type BatchResetFailure struct {
	UnionID   string `json:"union_id"`
	ErrorCode int32  `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

// BatchResetResult 批量密码重置结果。
type BatchResetResult struct {
	SuccessCount int32               `json:"success_count"`
	FailCount    int32               `json:"fail_count"`
	Failures     []BatchResetFailure `json:"failures"`
}

// OneIDBatchResetPassword 通过 Gateway 批量重置用户密码（迁移用）。
// 走 HMAC X-Internal-Token，max 100 条/批。
// OneID 采用部分失败模式：HTTP 200 + Failures 数组，调用方需解析 Failures。
func OneIDBatchResetPassword(ctx context.Context, items []PasswordResetItem) (*BatchResetResult, error) {
	if GatewayURL == "" {
		return nil, fmt.Errorf("GATEWAY_URL not configured")
	}
	accountID := hcommon.TenantIDFromCtx(ctx)
	if accountID == "" {
		return nil, fmt.Errorf("OneID account_id not available in ctx")
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"account_id": accountID,
		"items":      items,
	})

	apiURL := GatewayURL + "/api/batch-reset-passwords"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	setGatewayHeaders(req, ctx, accountID)

	slog.Info("[Gateway] API call", "url", apiURL, "has_secret", hcommon.InternalSecretFromCtx(ctx) != "")
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("gateway request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		slog.Error("[Gateway] API error", "url", apiURL, "status", resp.StatusCode, "body", safeBodySnippet(body, 512))
		return nil, fmt.Errorf("gateway returned %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			SuccessCount int32               `json:"success_count"`
			FailCount    int32               `json:"fail_count"`
			Failures     []BatchResetFailure `json:"failures"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse batch-reset-passwords response: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("batch-reset-passwords error: code=%d msg=%s", result.Code, result.Msg)
	}

	return &BatchResetResult{
		SuccessCount: result.Data.SuccessCount,
		FailCount:    result.Data.FailCount,
		Failures:     result.Data.Failures,
	}, nil
}

// ── 通用 OneID API 调用 ─────────────────────────────────────────────────────

// oneIDAPICall 发送 OneID OpenAPI 请求，返回响应体。
// 统一处理错误码检查（非创建接口）。
func oneIDAPICall(ctx context.Context, method, apiURL, token string, payload interface{}) ([]byte, error) {
	var bodyReader *bytes.Reader
	if payload != nil {
		reqBody, err := json.Marshal(payload)
		if err != nil {
			return nil, hcommon.I18nRichError(err, i18n.MsgProviderMarshalRequest)
		}
		bodyReader = bytes.NewReader(reqBody)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, bodyReader)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDBuildRequestFailed)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	setAcceptLanguageHeader(req, ctx)

	slog.Info("[OneID] API call", "method", method, "url", apiURL, "token_prefix", safeTokenPrefix(token))

	slog.Info("[Gateway] API call", "url", apiURL, "has_secret", hcommon.InternalSecretFromCtx(ctx) != "")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgOneIDAPIRequestFailed)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	slog.Info("[OneID] API response", "method", method, "url", apiURL, "status", resp.StatusCode, "body_snippet", safeBodySnippet(body, 256))
	if resp.StatusCode != http.StatusOK {
		return nil, hcommon.I18nError(i18n.MsgOneIDAPIReturned, resp.StatusCode, string(body))
	}

	// 检查业务错误码（创建接口由调用方自行解析 data）
	var check struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &check); err == nil && check.Code != 0 {
		return nil, hcommon.I18nError(i18n.MsgOneIDAPIError, check.Code, check.Msg)
	}

	return body, nil
}

// ── OneID 登录代理（透明转发，无需 Token） ───────────────────────────────────────

// oneIDPathMap 将 hatchery 路由路径映射为 OneID 实际 API 路径。
// hatchery 统一使用 /oneid/ 前缀，转发时替换为 OneID 真实路径。
var oneIDPathMap = map[string]string{
	"/oneid/encrypt_setting": "/v1/authn/encrypt_setting",
	"/oneid/enterprise":      "/v1/authn/enterprise",
	"/oneid/password-reset":  "/v1/authn/enterprise/password:reset",
	"/oneid/password-verify": "/user/v1/user/enterprise/password:verify",
	"/oneid/password-change": "/user/v1/user/enterprise/password:reset",
}

// resolveOneIDPath 将 hatchery 请求路径转为 OneID API 路径。
func resolveOneIDPath(hatcheryPath string) string {
	if p, ok := oneIDPathMap[hatcheryPath]; ok {
		return p
	}
	return hatcheryPath
}

// oneIDUsernameRegex 用户名格式：仅允许大小写字母、数字和特殊字符，最长 191 字符（对齐 DB varchar(191)）。
var oneIDUsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9!@#$%^&*()_+\-=\[\]{};':"\\|,.<>/?~` + "`" + `]+$`)

// validateOneIDUsername 校验用户名格式和长度。
func validateOneIDUsername(username string) error {
	if username == "" {
		return hcommon.I18nError(i18n.MsgOneIDUsernameNotEmpty)
	}
	if len(username) > 191 {
		return hcommon.I18nError(i18n.MsgOneIDUsernameTooLong)
	}
	if !oneIDUsernameRegex.MatchString(username) {
		return hcommon.I18nError(i18n.MsgOneIDUsernameInvalidChars)
	}
	return nil
}

// generateRandomLoginName 生成随机登录名（当用户名含中文等不符合规范时使用）。
// 格式：user_ + 8位随机hex
func generateRandomLoginName() string {
	b := make([]byte, 4)
	for i := range b {
		b[i] = byte(time.Now().UnixNano()>>uint(i*8)) ^ byte(i*37+13)
	}
	return fmt.Sprintf("user_%x", b)
}

// HandleOneIDLoginName 查询本地用户对应的 OneID 登录名。
// GET /oneid/login-name?username=xxx
// 无需登录态，供前端登录前获取 OneID 登录名。
func HandleOneIDLoginName(w http.ResponseWriter, r *http.Request) {
	if !hcommon.IsUnifiedAccountMode(r.Context()) {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgNotUnifiedAccountMode))
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgOneIDMissingUsernameParam))
		return
	}

	var user model.User
	if model.DB(r.Context()).Where("username = ?", username).First(&user).Error != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":            "用户不存在",
			"username":         username,
			"oneid_login_name": "",
		})
		return
	}

	loginName := ""
	if user.OneIDLoginName != nil {
		loginName = *user.OneIDLoginName
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"username":         user.Username,
		"oneid_login_name": loginName,
	})
}

// HandleOneIDAuthnProxy 透明代理 OneID 登录相关接口到 base API。
// 支持的路径：
//   - GET  /v1/authn/encrypt_setting
//   - POST /v1/authn/enterprise
func HandleOneIDAuthnProxy(w http.ResponseWriter, r *http.Request) {
	// 仅统一账号模式可用
	if !hcommon.IsUnifiedAccountMode(r.Context()) {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgNotUnifiedAccountMode))
		return
	}

	targetURL := strings.TrimRight(getOneIDAPIBaseURL(r.Context()), "/") + resolveOneIDPath(r.URL.Path)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	var proxyReq *http.Request
	var err error

	if r.Method == http.MethodGet {
		proxyReq, err = http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
	} else {
		// POST：原样转发 body
		body, readErr := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		if readErr != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgReadRequestBody))
			return
		}
		proxyReq, err = http.NewRequestWithContext(r.Context(), r.Method, targetURL, bytes.NewReader(body))
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgOneIDBuildProxyRequestFailed))
		return
	}

	// 转发 Content-Type、OneID Cookie 和 Accept-Language
	forwardProxyHeaders(proxyReq, r)

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(proxyReq)
	if err != nil {
		slog.Error("[OneIDAuthnProxy] request failed", "url", targetURL, "err", err)
		writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDRequestFailed))
		return
	}
	defer resp.Body.Close()

	// 原样返回响应
	respBody, _ := io.ReadAll(resp.Body)
	for _, h := range []string{"Content-Type", "X-Request-Id"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	// 只转发 state_token 的 Set-Cookie，将 domain 替换为租户配置的对外域名
	ourDomain := hcommon.DomainFromCtx(r.Context())
	for _, raw := range resp.Header.Values("Set-Cookie") {
		if ourDomain != "" {
			raw = rewriteCookieDomain(raw, ourDomain)
		}
		w.Header().Add("Set-Cookie", raw)
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// HandleOneIDPasswordReset 处理 POST /v1/authn/enterprise/password:reset。
// 透明代理到 OneID，成功后清除本地 hatchery session（强制重新登录）。
func HandleOneIDPasswordReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgMethodNotAllowed))
		return
	}
	if !hcommon.IsUnifiedAccountMode(r.Context()) {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgNotUnifiedAccountMode))
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgReadRequestBody))
		return
	}

	targetURL := strings.TrimRight(getOneIDAPIBaseURL(r.Context()), "/") + resolveOneIDPath(r.URL.Path)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	proxyReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgOneIDBuildProxyRequestFailed))
		return
	}
	forwardProxyHeaders(proxyReq, r)

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(proxyReq)
	if err != nil {
		slog.Error("[OneIDPasswordReset] request failed", "url", targetURL, "err", err)
		writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDRequestFailed))
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// 转发响应头
	for _, h := range []string{"Content-Type", "X-Request-Id"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	ourDomain := hcommon.DomainFromCtx(r.Context())
	for _, raw := range resp.Header.Values("Set-Cookie") {
		if ourDomain != "" {
			raw = rewriteCookieDomain(raw, ourDomain)
		}
		w.Header().Add("Set-Cookie", raw)
	}

	// OneID 重置密码成功后清除本地 session，强制用户重新登录
	if resp.StatusCode == http.StatusOK {
		var errCheck struct {
			ErrCode string `json:"errCode"`
		}
		if json.Unmarshal(respBody, &errCheck) == nil && errCheck.ErrCode == "" {
			session := getSession(r)
			session.Values["username"] = ""
			session.Options.MaxAge = -1
			session.Save(r, w)
			// 清掉旧的 state_token cookie，避免与新 session_token 冲突
			expireCookie := "state_token=; Path=/; Max-Age=0"
			if ourDomain != "" {
				cleanDomain := strings.TrimPrefix(strings.TrimPrefix(ourDomain, "https://"), "http://")
				expireCookie += "; Domain=" + cleanDomain
			}
			w.Header().Add("Set-Cookie", expireCookie)
			slog.Info("[OneIDPasswordReset] password reset success, session cleared")
		}
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// HandleOneIDAuthnLogin 处理 POST /oneid/enterprise 登录代理。
// 转发到 OneID 验证，成功后通过 get_self_v3 获取用户身份，匹配本地用户建立 session。
func HandleOneIDAuthnLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgMethodNotAllowed))
		return
	}
	if !hcommon.IsUnifiedAccountMode(r.Context()) {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgNotUnifiedAccountMode))
		return
	}

	// 读取 request body
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgReadRequestBody))
		return
	}

	// 转发到 OneID
	targetURL := strings.TrimRight(getOneIDAPIBaseURL(r.Context()), "/") + resolveOneIDPath(r.URL.Path)

	proxyReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgOneIDBuildProxyRequestFailed))
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "" {
		proxyReq.Header.Set("Content-Type", ct)
	}
	setAcceptLanguageHeader(proxyReq, r.Context())
	// 登录接口不转发浏览器旧 cookie，强制 OneID 创建新 session

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(proxyReq)
	if err != nil {
		slog.Error("[OneIDAuthnLogin] request failed", "url", targetURL, "err", err)
		writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDRequestFailed))
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// 转发响应头
	for _, h := range []string{"Content-Type", "X-Request-Id"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	ourDomain := hcommon.DomainFromCtx(r.Context())
	for _, raw := range resp.Header.Values("Set-Cookie") {
		if ourDomain != "" {
			raw = rewriteCookieDomain(raw, ourDomain)
		}
		w.Header().Add("Set-Cookie", raw)
	}

	// 判断 OneID 是否真正登录成功：HTTP 200 且 next.type 不是需要额外步骤的值
	oneIDLoginOK := false
	if resp.StatusCode == http.StatusOK {
		var loginResp struct {
			Next *struct {
				Type string `json:"type"`
			} `json:"next"`
		}
		if json.Unmarshal(respBody, &loginResp) == nil {
			// next 不存在 或 type 不是 CAPTCHA_OPTIONS / ACCOUNT_RESET_PASSWORD → 登录完成
			if loginResp.Next == nil ||
				(loginResp.Next.Type != "CAPTCHA_OPTIONS" && loginResp.Next.Type != "ACCOUNT_RESET_PASSWORD") {
				oneIDLoginOK = true
			}
		}
	}

	// OneID 验证成功时，通过 get_self_v3 获取登录用户身份，匹配本地用户建 session
	if oneIDLoginOK {
		sessionToken := extractSessionToken(resp.Header)
		if sessionToken == "" {
			slog.Error("[OneIDAuthnLogin] login success but no session_token in response")
			writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIdRequireSessionToken))
			return
		}

		// 调 get_self_v3 获取当前登录的 OneID 用户信息
		oneIDUsername, oneIDUnionId, verifyErr := verifyOneIDSessionUser(r.Context(), sessionToken)
		if verifyErr != nil {
			slog.Error("[OneIDAuthnLogin] get_self_v3 failed", "err", verifyErr)
			writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIdObtainUserInfoFailed))
			return
		}

		// 用 unionId 匹配本地用户的 one_id_sub
		var user model.User
		if oneIDUnionId == "" {
			slog.Error("[OneIDAuthnLogin] get_self_v3 returned empty unionId", "oneid_username", oneIDUsername)
			writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIdUnionIDEmpty))
			return
		}

		if model.DB(r.Context()).Where("one_id_sub = ?", oneIDUnionId).First(&user).Error != nil {
			slog.Error("[OneIDAuthnLogin] local user not found by one_id_sub",
				"oneid_union_id", oneIDUnionId,
				"oneid_username", oneIDUsername,
			)
			writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgOneIdUserNotFound))
			return
		}

		session := getSession(r)
		session.Values["username"] = user.Username
		session.Values["role"] = user.Role
		session.Values["user_id"] = user.ID
		session.Values["login_at"] = time.Now().Unix()
		setSessionIdentifier(session, r.Context())
		delete(session.Values, "oneid_sid")
		delete(session.Values, "oneid_sub")
		delete(session.Values, "oneid_amr")
		delete(session.Values, "login_failures")
		session.Save(r, w)
		slog.Info("[OneIDAuthnLogin] session established",
			"username", user.Username,
			"role", user.Role,
			"oneid_username", oneIDUsername,
			"oneid_union_id", oneIDUnionId,
		)

		// 在 OneID 原始响应基础上追加登录态数据
		var merged map[string]interface{}
		if json.Unmarshal(respBody, &merged) == nil {
			merged["ok"] = true
			merged["redirect"] = "/"
			merged["role"] = user.Role
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(merged)
			return
		}
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// rewriteCookieDomain 将 Set-Cookie 中的 Domain=xxx 替换为目标域名。
// 若原始 cookie 没有 Domain 属性则不添加。
func rewriteCookieDomain(raw, newDomain string) string {
	// 去掉协议前缀，cookie Domain 只接受纯域名
	newDomain = strings.TrimPrefix(newDomain, "https://")
	newDomain = strings.TrimPrefix(newDomain, "http://")
	// Set-Cookie 格式: name=value; attr1; attr2=val; ...
	parts := strings.Split(raw, ";")
	for i, part := range parts {
		trimmed := strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(trimmed), "domain=") {
			parts[i] = " Domain=" + newDomain
		}
	}
	return strings.Join(parts, ";")
}

// filterOneIDCookies 从 Cookie 头中只保留 OneID 相关的 cookie（state_token / session_token），
// 过滤掉 hatchery-session 等我方 cookie，避免干扰 OneID 接口。
func filterOneIDCookies(raw string) string {
	if raw == "" {
		return ""
	}
	var kept []string
	for _, part := range strings.Split(raw, ";") {
		trimmed := strings.TrimSpace(part)
		name := strings.SplitN(trimmed, "=", 2)[0]
		switch name {
		case "state_token", "session_token":
			kept = append(kept, trimmed)
		}
	}
	return strings.Join(kept, "; ")
}

// verifyOneIDSessionUser 用 session_token 调 OneID Workspace API（/v1/authn:get_self_v3）
// 获取当前登录用户信息，用 username 与本地 oneid_login_name 做对比，防止接口参数被篡改导致越权。
// 返回 (oneID username, oneID unionId, error)。
func verifyOneIDSessionUser(ctx context.Context, sessionToken string) (string, string, error) {
	if sessionToken == "" {
		return "", "", hcommon.I18nError(i18n.MsgOneIDSessionTokenEmpty)
	}

	apiURL := strings.TrimRight(getOneIDWorkspaceBaseURL(ctx), "/") + "/v1/authn:get_self_v3"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", "", hcommon.I18nRichError(err, i18n.MsgOneIDBuildRequestFailed)
	}
	req.Header.Set("Accept", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	setAcceptLanguageHeader(req, ctx)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", "", hcommon.I18nRichError(err, i18n.MsgOneIDGetSelfV3Failed)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", "", hcommon.I18nError(i18n.MsgOneIDGetSelfV3HTTPError, resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			AccountUser struct {
				Name     string `json:"name"`
				Username string `json:"username"`
				UnionId  string `json:"unionId"`
			} `json:"accountUser"`
		} `json:"data"`
		ErrCode    string `json:"errCode"`
		ErrMessage string `json:"errMessage"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", hcommon.I18nRichError(err, i18n.MsgOneIDParseGetSelfV3Failed)
	}
	if result.ErrCode != "" {
		return "", "", hcommon.I18nError(i18n.MsgOneIDGetSelfV3BizError, result.ErrCode, result.ErrMessage)
	}

	slog.Info("[OneIDAuthnLogin] verify session user",
		"oneid_username", result.Data.AccountUser.Username,
		"oneid_name", result.Data.AccountUser.Name,
		"oneid_union_id", result.Data.AccountUser.UnionId,
	)

	return result.Data.AccountUser.Username, result.Data.AccountUser.UnionId, nil
}

// extractSessionToken 从 HTTP 响应的 Set-Cookie 头中提取 session_token 的值。
func extractSessionToken(respHeaders http.Header) string {
	for _, raw := range respHeaders.Values("Set-Cookie") {
		parts := strings.Split(raw, ";")
		if len(parts) == 0 {
			continue
		}
		nv := strings.SplitN(strings.TrimSpace(parts[0]), "=", 2)
		if len(nv) == 2 && nv[0] == "session_token" {
			return nv[1]
		}
	}
	return ""
}

// setAcceptLanguageHeader 设置 Accept-Language 头（如果 ctx 中有语言偏好）。
func setAcceptLanguageHeader(req *http.Request, ctx context.Context) {
	if al := i18n.AcceptLanguageFromCtx(ctx); al != "" {
		req.Header.Set("Accept-Language", al)
	}
}

// setGatewayHeaders 设置 Gateway 内部接口请求所需的公共 header：
//   - Content-Type: application/json
//   - X-Internal-Tenant: accountID
//   - Accept-Language: 来自 ctx
//   - X-Internal-Token: HMAC 签名（如果 ctx 中有 internal secret）
func setGatewayHeaders(req *http.Request, ctx context.Context, accountID string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Tenant", accountID)
	setAcceptLanguageHeader(req, ctx)
	if secret := hcommon.InternalSecretFromCtx(ctx); secret != "" {
		req.Header.Set("X-Internal-Token", signInternalRequest(secret))
	}
}

// forwardProxyHeaders 将原始请求的 Content-Type、过滤后的 OneID Cookie 和 Accept-Language 转发到代理请求。
func forwardProxyHeaders(proxyReq *http.Request, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != "" {
		proxyReq.Header.Set("Content-Type", ct)
	}
	if ck := filterOneIDCookies(r.Header.Get("Cookie")); ck != "" {
		proxyReq.Header.Set("Cookie", ck)
	}
	setAcceptLanguageHeader(proxyReq, r.Context())
}

// safeTokenPrefix 返回 token 前 20 字符用于日志，避免完整 token 泄露。
func safeTokenPrefix(token string) string {
	if len(token) <= 20 {
		return token[:len(token)/2] + "..."
	}
	return token[:20] + "..."
}

// safeBodySnippet 截取响应体前 n 字节用于日志。
func safeBodySnippet(body []byte, n int) string {
	if len(body) <= n {
		return string(body)
	}
	return string(body[:n]) + "...(truncated)"
}
