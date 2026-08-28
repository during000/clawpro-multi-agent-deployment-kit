package model

import (
	"context"
	"errors"
	"log/slog"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// SkillBundle 技能包
type SkillBundle struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	Identifier     string    `gorm:"index;default:''"` // 多租户标识，MySQL 模式下自动填充和过滤
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Name           string    `gorm:"not null;default:''" json:"name"`
	SkillCount     int       `gorm:"not null;default:0" json:"skill_count"`
	Enabled        bool      `gorm:"default:false" json:"enabled"`
	VisibilityType string    `gorm:"not null;default:'all'" json:"visibility_type"` // 可见范围：all=全部用户, group=按分组
}

// BundleSkill 技能包内技能
type BundleSkill struct {
	ID                 uint      `gorm:"primarykey" json:"id"`
	Identifier         string    `gorm:"index;default:''"` // 多租户标识，MySQL 模式下自动填充和过滤
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	SkillBundleID      uint      `gorm:"not null;index;index:idx_bundle_skills_source_slug_version_bundle,priority:4;index:idx_bundle_skills_source_skillset_bundle,priority:2" json:"skill_bundle_id"`
	Name               string    `gorm:"not null;default:''" json:"name"`
	Slug               string    `gorm:"not null;default:'';index:idx_bundle_skills_source_slug_version_bundle,priority:2" json:"slug"`
	Version            string    `gorm:"not null;default:'';index:idx_bundle_skills_source_slug_version_bundle,priority:3" json:"version"`
	Source             string    `gorm:"not null;default:'public';index:idx_bundle_skills_source_slug_version_bundle,priority:1" json:"source"` // public / enterprise
	SourceSkillsetSlug string    `gorm:"not null;default:'';index;index:idx_bundle_skills_source_skillset_bundle,priority:1" json:"source_skillset_slug"`
	SourceSkillsetName string    `gorm:"not null;default:''" json:"source_skillset_name"`
	CosZipKey          string    `gorm:"not null;default:''" json:"cos_zip_key"`
}

// DefaultBundleName 默认技能包名称常量
const DefaultBundleName = "通用技能包"
const DefaultBundleNameEn = "General Skill Bundle"

func getDefaultBundleName(ctx context.Context) string {
	defaultLang := hcommon.DefaultLangFromCtx(ctx)
	defaultBundleName := DefaultBundleNameEn
	if defaultLang != "en" {
		defaultBundleName = DefaultBundleName
	}
	return defaultBundleName
}

// defaultBundleSkills 定义默认技能包中的公共技能列表。
var defaultBundleSkills = []struct {
	Slug    string
	Version string
}{
	{"openclaw-tavily-search", "0.1.0"},
	{"summarize", "1.0.0"},
	{"agent-browser-clawdbot", "0.1.0"},
	{"github", "1.0.0"},
	{"obsidian", "1.0.0"},
	{"tencent-docs", "1.0.7"},
	{"tencent-cos-skill", "1.0.6"},
}

// SeedDefaultSkillBundle 初始化默认技能包。
// 通过 SiteConfig.DefaultBundleSeeded 标记判断是否已初始化过，
// 一旦标记为 true，即使内置技能包被管理员删除，重启也不会重新创建。
// tx 为调用方传入的事务句柄(可能是 InitTenant 的外层事务)，
// 因此内部不再开启嵌套 Transaction，直接使用 tx 执行所有 DB 操作。
func SeedDefaultSkillBundle(ctx context.Context, tx *gorm.DB) error {
	defaultBundleName := getDefaultBundleName(ctx)

	var config SiteConfig
	if err := tx.First(&config).Error; err != nil {
		slog.Error("读取站点配置失败", slog.Any("error", err))
		return hcommon.I18nRichError(err, i18n.MsgSBReadSiteConfigFailed)
	}
	if config.DefaultBundleSeeded {
		return nil // 已经初始化过，跳过（即使技能包被删除也不重建）
	}

	// 再检查一下是否已存在同名技能包（兼容旧数据升级场景）
	var existing SkillBundle
	err := tx.Where("name IN ?", []string{DefaultBundleName, DefaultBundleNameEn}).First(&existing).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("检查默认技能包失败", slog.Any("error", err))
			return hcommon.I18nRichError(err, i18n.MsgSBReadDefaultBundleFailed)
		}
	} else {
		// 技能包已存在但标记未设置（旧版本升级），补标记即可
		if err := tx.Model(&config).Update("default_bundle_seeded", true).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSBSetDefaultSeededFailed)
		} else {
			slog.Info("默认技能包已存在，补设 DefaultBundleSeeded 标记")
		}
		return nil
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		bundle := SkillBundle{
			Name:       defaultBundleName,
			SkillCount: len(defaultBundleSkills),
			Enabled:    true,
		}
		if err := tx.Create(&bundle).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSBCreateDefaultBundleFailed)
		}

		for _, s := range defaultBundleSkills {
			skill := BundleSkill{
				SkillBundleID: bundle.ID,
				Name:          s.Slug, // name 暂用 slug，阶段2可补充
				Slug:          s.Slug,
				Version:       s.Version,
				Source:        "public",
				CosZipKey:     "", // 阶段2填充
			}
			if err := tx.Create(&skill).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgSBCreateDefaultSkillFailed, s.Slug)
			}
		}

		// 标记已初始化
		if err := tx.Model(&config).Update("default_bundle_seeded", true).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSBUpdateDefaultSeededFailed)
		}

		return nil
	})

	if err != nil {
		slog.Error("初始化默认技能包失败", "error", err)
		return err
	}

	slog.Info("默认技能包初始化成功", "name", defaultBundleName, "skill_count", len(defaultBundleSkills))
	return nil
}
