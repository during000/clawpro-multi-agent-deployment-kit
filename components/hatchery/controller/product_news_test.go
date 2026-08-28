package controller

import (
	"testing"
	"time"
)

func TestProductNewsCacheExpiry(t *testing.T) {
	cache := &productNewsCache{}
	if got := cache.get(); got != nil {
		t.Error("空缓存 get() 应返回 nil")
	}

	items := []ProductNewsItem{
		{Title: "测试标题", Type: "feature", PublishDate: "2026-04-10"},
	}
	cache.set(items)
	if got := cache.get(); got == nil {
		t.Error("刚写入的缓存 get() 不应返回 nil")
	}

	cache.mu.Lock()
	cache.expires = time.Now().Add(-1 * time.Second)
	cache.mu.Unlock()
	if got := cache.get(); got != nil {
		t.Error("过期缓存 get() 应返回 nil")
	}
}

func TestProductNewsCacheCopy(t *testing.T) {
	cache := &productNewsCache{}
	items := []ProductNewsItem{
		{Title: "标题A", Type: "feature", PublishDate: "2026-04-01"},
		{Title: "标题B", Type: "improvement", PublishDate: "2026-04-02"},
	}
	cache.set(items)

	got := cache.get()
	if len(got) != 2 || got[0].Title != "标题A" {
		t.Errorf("缓存内容不匹配: %+v", got)
	}

	got[0].Title = "被修改"
	got2 := cache.get()
	if got2[0].Title != "标题A" {
		t.Error("修改 get() 返回值不应影响缓存内容")
	}
}

func TestConvertCloudNewsToItems(t *testing.T) {
	cloudItems := []cloudProductNewsItem{
		{
			ClawProNewsId: "cpn-abc123",
			Title:         "记忆管理功能上线",
			Summary:       "支持 Pro / Free 版本切换",
			Type:          "feature",
			NewsVersion:   "v2.4.0",
			PublishDate:   "2026-04-10",
			Link:          "https://example.com",
			ShowBanner:    true,
			BannerText:    "【产品动态】记忆管理功能上线",
			SortOrder:     100,
			Status:        1,
		},
		{
			ClawProNewsId: "cpn-def456",
			Title:         "模型支持设为默认",
			Summary:       "管理员可在模型配置页将模型设为默认",
			Type:          "improvement",
			PublishDate:   "2026-04-02",
			ShowBanner:    false,
			SortOrder:     50,
			Status:        1,
		},
	}

	items := convertCloudNewsToItems(cloudItems)

	if len(items) != 2 {
		t.Fatalf("期望 2 条，实际 %d 条", len(items))
	}

	if items[0].Title != "记忆管理功能上线" {
		t.Errorf("Title 不匹配: %s", items[0].Title)
	}
	if items[0].Type != "feature" {
		t.Errorf("Type 不匹配: %s", items[0].Type)
	}
	if items[0].PublishDate != "2026-04-10" {
		t.Errorf("PublishDate 不匹配: %s", items[0].PublishDate)
	}
	if !items[0].ShowBanner {
		t.Error("ShowBanner 应为 true")
	}
	if items[0].BannerText != "【产品动态】记忆管理功能上线" {
		t.Errorf("BannerText 不匹配: %s", items[0].BannerText)
	}
	if items[0].ID != "cpn-abc123" {
		t.Errorf("ID 不匹配: %s", items[0].ID)
	}
	if items[0].Link != "https://example.com" {
		t.Errorf("Link 不匹配: %s", items[0].Link)
	}

	if items[1].ShowBanner {
		t.Error("第二条 ShowBanner 应为 false")
	}
	if items[1].BannerText != "" {
		t.Errorf("第二条 BannerText 应为空: %s", items[1].BannerText)
	}
	if items[1].ID != "cpn-def456" {
		t.Errorf("第二条 ID 不匹配: %s", items[1].ID)
	}
}

func TestConvertCloudNewsToItemsEmpty(t *testing.T) {
	items := convertCloudNewsToItems(nil)
	if len(items) != 0 {
		t.Errorf("空输入应返回空列表，实际: %d", len(items))
	}

	items = convertCloudNewsToItems([]cloudProductNewsItem{})
	if len(items) != 0 {
		t.Errorf("空切片应返回空列表，实际: %d", len(items))
	}
}
