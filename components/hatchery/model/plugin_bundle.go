package model

import (
	"log/slog"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// PluginBundle 插件包
type PluginBundle struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	Identifier     string    `gorm:"index;default:''" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Name           string    `gorm:"not null;default:''" json:"name"`
	PluginCount    int       `gorm:"not null;default:0" json:"plugin_count"`
	Enabled        bool      `gorm:"default:false" json:"enabled"`
	VisibilityType string    `gorm:"size:16;not null;default:'all'" json:"visibility_type"` // 应用范围：'all'=全部用户, 'group'=按组可见
}

// BundlePlugin 插件包内插件
type BundlePlugin struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	Identifier     string    `gorm:"index;default:''" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	PluginBundleID uint      `gorm:"not null;index" json:"plugin_bundle_id"`
	Name           string    `gorm:"not null;default:''" json:"name"`
	Slug           string    `gorm:"not null;default:''" json:"slug"`
	PluginID       string    `gorm:"not null;default:''" json:"plugin_id"` // openclaw 运行时 id
	Version        string    `gorm:"not null;default:''" json:"version"`
	Source         string    `gorm:"not null;default:'enterprise'" json:"source"` // enterprise / npm
	CosZipKey      string    `gorm:"not null;default:''" json:"cos_zip_key"`
	NpmPackage     string    `gorm:"not null;default:''" json:"npm_package"`
	InstallMode    string    `gorm:"not null;default:'smh'" json:"install_mode"` // smh / npm
	Kind           string    `gorm:"not null;default:''" json:"kind"`            // memory / context-engine / ""
}

// DefaultPluginBundleName 默认插件包名称常量
const DefaultPluginBundleName = "通用插件包"

// SeedDefaultPluginBundle 初始化默认插件包。
// 通过 SiteConfig.DefaultPluginBundleSeeded 标记判断是否已初始化过，
// 一旦标记为 true，即使插件包被管理员删除，重启也不会重新创建。
func SeedDefaultPluginBundle(tx *gorm.DB) error {
	var config SiteConfig
	if err := tx.First(&config).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgPluginSeedReadSiteConfig)
	}
	if config.DefaultPluginBundleSeeded {
		return nil // 已经初始化过
	}

	// 再检查一下是否已存在同名插件包（兼容旧数据升级场景）
	var existing PluginBundle
	if tx.Where("name = ?", DefaultPluginBundleName).First(&existing).Error == nil {
		// 插件包已存在但标记未设置（旧版本升级），补标记即可
		if err := tx.Model(&config).Update("default_plugin_bundle_seeded", true).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgPluginSeedSetSeededIfExists)
		} else {
			slog.Info("默认插件包已存在，补设 DefaultPluginBundleSeeded 标记")
		}
		return nil
	}

	// 创建空的默认插件包（管理员后续通过管理界面添加插件）
	err := tx.Transaction(func(tx *gorm.DB) error {
		bundle := PluginBundle{
			Name:        DefaultPluginBundleName,
			PluginCount: 0,
			Enabled:     false, // 默认不启用，管理员手动启用
		}
		if err := tx.Create(&bundle).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgPluginSeedCreateBundle)
		}

		// 标记已初始化
		if err := tx.Model(&config).Update("default_plugin_bundle_seeded", true).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgPluginSeedUpdateSeededFlag)
		}

		return nil
	})

	if err != nil {
		slog.Error("初始化默认插件包失败", "error", err)
		return err
	}

	slog.Info("默认插件包初始化成功", "name", DefaultPluginBundleName)
	return nil
}
