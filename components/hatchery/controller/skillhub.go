package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/model"
	"hatchery/skillhubclient"
)

// ── access_token 缓存（与 oneid_unified.go getOneIDAppToken 同模式）──────────────

type skillHubTokenEntry struct {
	token     string
	expiresAt time.Time
}

var (
	skillHubTokenMu    sync.Mutex
	skillHubTokenCache = map[string]*skillHubTokenEntry{} // key: "identifier:sub:tid"
)

// ── OrgInfo 缓存 ──────────────────────────────────────────────────────────────

var (
	skillHubOrgMu    sync.Mutex
	skillHubOrgCache = map[string]*skillhubclient.OrgInfo{} // key: identifier
)

// isSkillHubEnabled 检查当前租户是否启用 SkillHub 迁移。
func isSkillHubEnabled(r *http.Request) bool {
	return model.GetSiteConfig(r.Context()).SkillHubEnabled
}

// getSkillHubClient 创建带 access_token 的 SkillHub 客户端。
// 返回 (client, error)：error!=nil 表示配置问题，不应静默降级。
func getSkillHubClient(r *http.Request) (*skillhubclient.Client, error) {
	ctx := r.Context()
	config := model.GetSiteConfig(ctx)
	if config.SkillHubAPIURL == "" {
		return nil, fmt.Errorf("skillhub: skill_hub_api_url not configured")
	}

	// 获取当前登录用户的 OneID sub
	loginUser, err := getLoginUser(r)
	if err != nil || loginUser == nil {
		return nil, fmt.Errorf("skillhub: 无法获取登录用户: %w", err)
	}
	if loginUser.OneIDSub == nil || *loginUser.OneIDSub == "" {
		return nil, fmt.Errorf("skillhub: 当前用户没有 OneID 身份，无法获取 access_token")
	}
	sub := *loginUser.OneIDSub

	// tid 从租户快照获取
	snap, ok := hcommon.GetTenantSnapshot(ctx)
	if !ok || snap.OneIDAccountID == "" {
		return nil, fmt.Errorf("skillhub: 租户缺少 OneID account_id")
	}
	tid := snap.OneIDAccountID

	// 获取 access_token（带缓存，通过 Gateway 代理获取）
	token, err := getSkillHubAccessToken(ctx, snap, sub, tid)
	if err != nil {
		return nil, fmt.Errorf("skillhub: 获取 access_token 失败: %w", err)
	}

	// 获取 OrgID（带缓存）
	orgID, err := getSkillHubOrgID(ctx, snap, config.SkillHubAPIURL, token)
	if err != nil {
		return nil, fmt.Errorf("skillhub: 获取 OrgID 失败: %w", err)
	}

	return skillhubclient.NewClient(config.SkillHubAPIURL, token, orgID), nil
}

// getSkillHubAccessToken 通过 Gateway POST /api/access-token 获取用户级 access_token。
// 与 getOneIDAppToken / OneIDAddRoleUsers 同模式：调 Gateway + X-Internal-Token 鉴权。
func getSkillHubAccessToken(ctx context.Context, snap hcommon.TenantSnapshot, sub, tid string) (string, error) {
	if GatewayURL == "" {
		return "", fmt.Errorf("skillhub: GATEWAY_URL not configured")
	}
	if snap.InternalSecret == "" {
		return "", fmt.Errorf("skillhub: internal_secret not configured")
	}

	cacheKey := snap.Identifier + ":" + sub + ":" + tid

	skillHubTokenMu.Lock()
	defer skillHubTokenMu.Unlock()

	// 检查缓存
	if entry, ok := skillHubTokenCache[cacheKey]; ok && time.Now().Before(entry.expiresAt.Add(-60*time.Second)) {
		return entry.token, nil
	}

	// 调 Gateway POST /api/access-token
	payload, _ := json.Marshal(map[string]string{
		"sub": sub,
		"tid": tid,
	})

	apiURL := strings.TrimRight(GatewayURL, "/") + "/api/access-token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("skillhub: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Tenant", tid)
	req.Header.Set("X-Internal-Token", signInternalRequest(snap.InternalSecret))

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("skillhub: gateway request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("skillhub: gateway /api/access-token returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("skillhub: parse gateway response: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("skillhub: empty access_token from gateway")
	}

	// 缓存 token（提前 60s 刷新）
	skillHubTokenCache[cacheKey] = &skillHubTokenEntry{
		token:     result.AccessToken,
		expiresAt: time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}

	slog.InfoContext(ctx, "[SkillHub] access_token cached via gateway",
		"sub", sub, "tid", tid, "expires_in", result.ExpiresIn)

	return result.AccessToken, nil
}

// getSkillHubOrgID 获取 SkillHub 中的 OrgID（按租户缓存）。
// 首次调用时通过 /api/v1/auth/me 获取，之后从缓存读取。
func getSkillHubOrgID(ctx context.Context, snap hcommon.TenantSnapshot, baseURL, token string) (uint64, error) {
	skillHubOrgMu.Lock()
	defer skillHubOrgMu.Unlock()

	if cached, ok := skillHubOrgCache[snap.Identifier]; ok {
		return cached.OrgID, nil
	}

	// 未命中：用已有 token 调 /api/v1/auth/me
	tempClient := skillhubclient.NewClient(baseURL, token, 0)
	orgInfo, err := tempClient.FetchOrgInfo(ctx)
	if err != nil {
		return 0, fmt.Errorf("fetch org info: %w", err)
	}

	skillHubOrgCache[snap.Identifier] = orgInfo
	slog.InfoContext(ctx, "[SkillHub] OrgID cached",
		"identifier", snap.Identifier, "orgId", orgInfo.OrgID, "orgPublicId", orgInfo.OrgPublicID)

	return orgInfo.OrgID, nil
}

// HandleSkillHubStatus 返回当前租户的 SkillHub 灰度状态。
// GET /admin/skillhub-status
// 响应中 skillhub_url 是前端跳转地址，由 skill_hub_api_url 去掉 "api." 前缀推导。
// 例如：https://api.skillhub.cn → https://skillhub.cn
func HandleSkillHubStatus(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	config := model.GetSiteConfig(r.Context())

	// 从 API URL 推导前端 URL：去掉 host 中的 "api." 前缀
	frontendURL := config.SkillHubAPIURL
	if idx := strings.Index(frontendURL, "://api."); idx >= 0 {
		frontendURL = frontendURL[:idx+3] + frontendURL[idx+7:]
	}

	slog.InfoContext(r.Context(), "[SkillHub] 灰度状态查询",
		"skill_hub_enabled", config.SkillHubEnabled,
		"skillhub_api_url", config.SkillHubAPIURL,
		"skillhub_frontend_url", frontendURL)

	jsonOK(w, map[string]interface{}{
		"enabled":      config.SkillHubEnabled,
		"skillhub_url": frontendURL,
	})
}
