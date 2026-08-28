package skillhubclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// OrgInfo SkillHub 中租户的企业信息（从 /api/v1/auth/me 获取）。
type OrgInfo struct {
	OrgID       uint64 // SkillHub 内部数字 orgId
	OrgPublicID string // SkillHub 公开 orgPublicId，如 "org-bv6b8qcb"
}

// Client SkillHub API 客户端（无状态，每次请求由 controller 创建）。
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string // 已获取的 access_token
	orgID      uint64 // SkillHub 内部 orgId
}

// NewClient 创建 SkillHub API 客户端。
// token 由 controller 层缓存并传入，Client 不负责获取 token。
func NewClient(baseURL, token string, orgID uint64) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      token,
		orgID:      orgID,
	}
}

// doRequest 带 Bearer token 发起 JSON API 请求。
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("skillhubclient: new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("skillhubclient: request failed: %w", err)
	}
	return resp, nil
}

// FetchOrgInfo 调用 GET /api/v1/auth/me 获取租户的 orgId + orgPublicId。
// 不需要 orgId 路径参数，只需要 Bearer token。
func (c *Client) FetchOrgInfo(ctx context.Context) (*OrgInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/auth/me", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("skillhubclient: auth/me returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		User struct {
			Enterprise struct {
				OrgID       uint64 `json:"orgId"`
				OrgPublicID string `json:"orgPublicId"`
			} `json:"enterprise"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("skillhubclient: decode auth/me response: %w", err)
	}

	if result.User.Enterprise.OrgID == 0 {
		return nil, fmt.Errorf("skillhubclient: auth/me returned empty orgId")
	}

	return &OrgInfo{
		OrgID:       result.User.Enterprise.OrgID,
		OrgPublicID: result.User.Enterprise.OrgPublicID,
	}, nil
}

// ── 技能列表 ──

// SkillListResponse SkillHub 分页响应。
type SkillListResponse struct {
	Total int         `json:"total"`
	Items []SkillItem `json:"items"`
}

// SkillItem SkillHub 技能条目。
type SkillItem struct {
	ID          uint64 `json:"id"`
	DisplayName string `json:"display_name"`
	Slug        string `json:"slug"`
	Summary     string `json:"summary"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	OrgID       uint64 `json:"org_id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ListSkills 调用 SkillHub 获取技能列表。
func (c *Client) ListSkills(ctx context.Context, page, pageSize int, keyword string) (*SkillListResponse, error) {
	path := fmt.Sprintf("/api/v1/orgs/%d/skills?page=%d&page_size=%d", c.orgID, page, pageSize)
	if keyword != "" {
		path += "&keyword=" + url.QueryEscape(keyword)
	}

	slog.InfoContext(ctx, "skillhubclient: ListSkills", "orgId", c.orgID, "page", page)

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("skillhubclient: ListSkills returned %d: %s", resp.StatusCode, string(body))
	}

	var result SkillListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("skillhubclient: decode response: %w", err)
	}
	return &result, nil
}
