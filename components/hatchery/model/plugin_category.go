package model

import (
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// PluginCategory 插件分类
type PluginCategory struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	Identifier  string    `gorm:"uniqueIndex:idx_plugin_cat_name_identifier;index;default:''" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"uniqueIndex:idx_plugin_cat_name_identifier;not null" json:"name"`
	Description string    `gorm:"type:text;not null" json:"description"`
}

// PluginCategoryMapping 插件-分类关联
type PluginCategoryMapping struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Identifier string `gorm:"uniqueIndex:idx_plugin_cat_map;index;default:''" json:"-"`
	PluginID   uint   `gorm:"uniqueIndex:idx_plugin_cat_map;not null" json:"plugin_id"` // 关联 Plugin 表的 ID
	CategoryID uint   `gorm:"uniqueIndex:idx_plugin_cat_map;not null" json:"category_id"`
}

var predefinedPluginCategories = []struct {
	Name        string
	Description string
}{
	{"AI 模型提供商", "OpenAI、Anthropic、Gemini 等模型接入"},
	{"消息渠道", "企业微信、飞书、钉钉、Slack 等消息渠道"},
	{"智能体工具", "代码搜索、API 调试、文件操作等智能体工具"},
	{"语音与媒体", "TTS、STT、图像生成、视频生成等"},
	{"知识检索", "Web 搜索、网页抓取、知识库查询"},
	{"记忆与上下文", "长期记忆、上下文引擎"},
	{"其他", "其他分类"},
}

// SeedPluginCategories 初始化预置插件分类。
// tx 为调用方传入的事务句柄。
func SeedPluginCategories(tx *gorm.DB) error {
	names := make([]string, len(predefinedPluginCategories))
	for i, cat := range predefinedPluginCategories {
		names[i] = cat.Name
	}
	var existing []PluginCategory
	if err := tx.Where("name IN ?", names).Find(&existing).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgPluginCatSeedQuery)
	}
	existSet := make(map[string]struct{}, len(existing))
	for _, e := range existing {
		existSet[e.Name] = struct{}{}
	}

	var toCreate []PluginCategory
	for _, cat := range predefinedPluginCategories {
		if _, ok := existSet[cat.Name]; !ok {
			toCreate = append(toCreate, PluginCategory{Name: cat.Name, Description: cat.Description})
		}
	}
	if len(toCreate) > 0 {
		if err := tx.Create(&toCreate).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgPluginCatSeedInsert)
		}
	}
	return nil
}
