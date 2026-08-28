package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

const (
	productNewsCacheTTL = 3 * time.Minute
)

// ProductNewsItem 产品动态条目（对外返回给前端的精简格式）
type ProductNewsItem struct {
	ID          string `json:"id"` // 产品动态 ID
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	Type        string `json:"type"`         // "feature" | "improvement"
	PublishDate string `json:"publish_date"` // "YYYY-MM-DD"
	Link        string `json:"link"`         // 产品动态详情链接
	ShowBanner  bool   `json:"show_banner"`
	BannerText  string `json:"banner_text"`
}

// cloudProductNewsItem 腾讯云 API 返回的原始结构
type cloudProductNewsItem struct {
	ClawProNewsId string `json:"ClawProNewsId"`
	Title         string `json:"Title"`
	Summary       string `json:"Summary"`
	Type          string `json:"Type"`
	NewsVersion   string `json:"NewsVersion"`
	PublishDate   string `json:"PublishDate"`
	Link          string `json:"Link"`
	ShowBanner    bool   `json:"ShowBanner"`
	BannerText    string `json:"BannerText"`
	SortOrder     int    `json:"SortOrder"`
	Status        int    `json:"Status"`
}

// cloudProductNewsResponse 腾讯云 API 响应
type cloudProductNewsResponse struct {
	Response struct {
		ClawProProductNewsSet []cloudProductNewsItem `json:"ClawProProductNewsSet"`
		TotalCount            int                    `json:"TotalCount"`
		Error                 *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
	} `json:"Response"`
}

// productNewsCache 产品动态内存缓存，避免频繁调用云 API
type productNewsCache struct {
	mu      sync.RWMutex
	items   []ProductNewsItem
	expires time.Time
}

// get 获取缓存，过期返回 nil
func (c *productNewsCache) get() []ProductNewsItem {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.expires.IsZero() || time.Now().After(c.expires) {
		return nil
	}
	result := make([]ProductNewsItem, len(c.items))
	copy(result, c.items)
	return result
}

// set 写入缓存并设置过期时间
func (c *productNewsCache) set(items []ProductNewsItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make([]ProductNewsItem, len(items))
	copy(c.items, items)
	c.expires = time.Now().Add(productNewsCacheTTL)
}

var globalProductNewsCache = &productNewsCache{}

// convertCloudNewsToItems 将云 API 原始响应转换为前端精简格式
func convertCloudNewsToItems(cloudItems []cloudProductNewsItem) []ProductNewsItem {
	items := make([]ProductNewsItem, 0, len(cloudItems))
	for _, ci := range cloudItems {
		items = append(items, ProductNewsItem{
			ID:          ci.ClawProNewsId,
			Title:       ci.Title,
			Summary:     ci.Summary,
			Type:        ci.Type,
			PublishDate: ci.PublishDate,
			Link:        ci.Link,
			ShowBanner:  ci.ShowBanner,
			BannerText:  ci.BannerText,
		})
	}
	return items
}

// fetchProductNewsFromCloud 调用腾讯云 DescribeClawProProductNews
// limit/offset 控制分页，limit=0 时默认查 20 条
func fetchProductNewsFromCloud(ctx context.Context, limit, offset int) ([]cloudProductNewsItem, error) {
	credential, err := getCredential(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgGetCloudCredentialFailed)
	}

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cvm.tencentcloudapi.com"
	cpf.HttpProfile.ReqMethod = "POST"

	client := common.NewCommonClient(credential, CVMRegion, cpf)

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	params := fmt.Sprintf(`{"Limit": %d, "Offset": %d}`, limit, offset)

	request := tchttp.NewCommonRequest("cvm", "2017-03-12", "DescribeClawProProductNews")
	if err := request.SetActionParameters(params); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSetRequestParamsFailed)
	}

	response := tchttp.NewCommonResponse()
	if err := client.Send(request, response); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgProductNewsAPICallFailed)
	}

	var parsed cloudProductNewsResponse
	if err := json.Unmarshal(response.GetBody(), &parsed); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgParseResponseFailed)
	}

	if parsed.Response.Error != nil {
		return nil, hcommon.I18nError(i18n.MsgProductNewsAPIError,
			parsed.Response.Error.Code, parsed.Response.Error.Message)
	}

	return parsed.Response.ClawProProductNewsSet, nil
}

// buildProductNews 构建产品动态列表，带缓存
// limit/offset 控制返回条数，limit=0 表示返回全部（默认查 20 条）
func buildProductNews(ctx context.Context, limit, offset int) []ProductNewsItem {
	// 命中缓存直接返回
	if cached := globalProductNewsCache.get(); cached != nil {
		return applyPagination(cached, limit, offset)
	}

	// 未配置云 API 密钥时直接返回空列表，避免无意义的 error 日志
	config := model.GetSiteConfig(ctx)
	if config.CVMSecretId == "" {
		return []ProductNewsItem{}
	}

	// 缓存未命中时始终拉取全量（Limit=20），缓存后再截取
	cloudItems, err := fetchProductNewsFromCloud(ctx, 20, 0)
	if err != nil {
		slog.Error("[产品动态] 获取产品动态失败", "error", err)
		return []ProductNewsItem{}
	}

	items := convertCloudNewsToItems(cloudItems)
	globalProductNewsCache.set(items)
	return applyPagination(items, limit, offset)
}

// applyPagination 对结果切片做分页截取
func applyPagination(items []ProductNewsItem, limit, offset int) []ProductNewsItem {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []ProductNewsItem{}
	}
	items = items[offset:]
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items
}
