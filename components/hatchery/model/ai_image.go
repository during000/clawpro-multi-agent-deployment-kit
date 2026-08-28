package model

import (
	"context"
	"errors"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"time"

	"gorm.io/gorm"
)

type AIImage struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	Identifier          string    `gorm:"uniqueIndex:idx_image_identifier;index;default:''" json:"-"` // 多租户标识，MySQL 模式下自动填充和过滤
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	ImageId             string    `gorm:"uniqueIndex:idx_image_identifier;not null;default:''" json:"image_id"`
	ImageName           string    `json:"image_name"`
	ImageType           string    `json:"image_type"`
	OsName              string    `json:"os_name"`
	ImageSize           int64     `json:"image_size"`
	ImageState          string    `json:"image_state"`
	Enabled             bool      `gorm:"default:false" json:"enabled"`
	AgentType           string    `gorm:"type:varchar(32);default:''" json:"agent_type"`
	AgentVersion        string    `gorm:"type:varchar(64);default:''" json:"agent_version"`
	UpdateNoticeEnabled bool      `gorm:"not null;default:false;index" json:"update_notice_enabled"`
}

// GetEnabledImage returns the currently enabled image, or nil if none is enabled.
func GetEnabledImage(ctx context.Context) *AIImage {
	var img AIImage
	if DB(ctx).Where("enabled = ?", true).First(&img).Error != nil {
		return nil
	}
	return &img
}

// GetEnabledImageByType 获取指定类型的启用镜像
// 返回值：(*AIImage, error)
//   - 找到镜像：返回 (&img, nil)
//   - 镜像不存在：返回 (nil, nil)
//   - 数据库错误：返回 (nil, err)
func GetEnabledImageByType(ctx context.Context, agentType string) (*AIImage, error) {
	// 兼容存量数据：空字符串视为 openclaw 类型
	agentType = NormalizeAgentType(agentType)

	var img AIImage
	// 优先精确匹配类型
	err := DB(ctx).Where("agent_type = ? AND enabled = ?", agentType, true).First(&img).Error
	if err == nil {
		return &img, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, hcommon.I18nRichError(err, i18n.MsgQueryImageFailed)
	}

	// 自定义 Agent 类型必须有自己的启用镜像，不回退到空类型 legacy 镜像。
	if !IsBuiltinAgentType(agentType) {
		return nil, nil
	}

	// 兼容：内置类型回退到空类型的镜像（旧数据）
	err = DB(ctx).Where("(agent_type = '' OR agent_type IS NULL) AND enabled = ?", true).First(&img).Error
	if err == nil {
		return &img, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil // 没有可用镜像
	}
	return nil, hcommon.I18nRichError(err, i18n.MsgQueryImageFailed)
}

// GetEnabledImagesMap 批量获取所有类型的启用镜像
func GetEnabledImagesMap(ctx context.Context) (map[string]*AIImage, error) {
	var images []AIImage
	if err := DB(ctx).Where("enabled = ?", true).Find(&images).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgQueryEnabledImageFailed)
	}

	result := make(map[string]*AIImage)
	for i := range images {
		img := &images[i]
		// 兼容存量数据：空字符串视为 openclaw 类型
		agentType := NormalizeAgentType(img.AgentType)
		// 同类型多个启用镜像时，优先保留第一个（按 DB 查询顺序）
		if _, exists := result[agentType]; !exists {
			result[agentType] = img
		}
	}
	return result, nil
}

// CanEnableImage 检查镜像是否可以被启用。
// 返回 nil 表示可以启用；非 nil *RichError 表示不可启用的原因。
func (img *AIImage) CanEnableImage(ctx context.Context) error {
	// 1. 检查类型有效性
	if img.AgentType != "" && !IsValidAgentType(ctx, img.AgentType) {
		return hcommon.I18nError(i18n.MsgInvalidAgentType, img.AgentType)
	}

	// 2. 检查版本（存量镜像除外）
	// 存量镜像特征：agent_type 为空或 agent_version 为空
	// 自定义类型没有版本概念，允许 agent_version 为空
	if img.AgentType != "" && img.AgentVersion == "" && !IsCustomAgentType(ctx, img.AgentType) {
		return hcommon.I18nError(i18n.MsgSetAgentVersionBeforeEnable)
	}

	return nil
}

// IsLegacyImage 判断是否为存量镜像（无类型或无版本）。
// 自定义类型镜像允许无版本号，不算 legacy。
func (img *AIImage) IsLegacyImage(ctx context.Context) bool {
	if img.AgentType == "" {
		return true
	}
	if img.AgentVersion == "" && !IsCustomAgentType(ctx, img.AgentType) {
		return true
	}
	return false
}

// GetEnabledImageCountByType 获取指定类型的已启用镜像数量
func GetEnabledImageCountByType(ctx context.Context, agentType string) (int64, error) {
	var count int64
	err := DB(ctx).Model(&AIImage{}).
		Where("agent_type = ? AND enabled = ?", agentType, true).
		Count(&count).Error
	if err != nil {
		return 0, hcommon.I18nRichError(err, i18n.MsgQueryImageFailed)
	}
	return count, nil
}

// GetImageStatsByType 一次查询获取各类型镜像统计
func GetImageStatsByType(ctx context.Context) (map[string]int64, error) {
	type stat struct {
		AgentType string
		Count     int64
	}
	var stats []stat

	err := DB(ctx).Model(&AIImage{}).
		Select("agent_type, COUNT(*) as count").
		Group("agent_type").
		Scan(&stats).Error
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgQueryImageStatsFailed)
	}

	result := make(map[string]int64, len(stats))
	for _, s := range stats {
		result[s.AgentType] = s.Count
	}
	return result, nil
}
