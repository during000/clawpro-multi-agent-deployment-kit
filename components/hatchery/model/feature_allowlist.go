package model

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// FeatureAllowlistType 是 FeatureAllowlist.Type 的可选取值集合。
const (
	// FeatureAllowlistTypeLocalAgent 本地 agent 功能白名单。
	FeatureAllowlistTypeLocalAgent = "local-agent"
	// FeatureAllowlistTypePasswordlessLogin 免登录跳转链接功能白名单。
	FeatureAllowlistTypePasswordlessLogin = "passwordless-login"
)

type featureAllowlistConfig struct {
	allowWhenEmpty bool
}

var featureAllowlistConfigs = map[string]featureAllowlistConfig{
	FeatureAllowlistTypeLocalAgent:        {allowWhenEmpty: true},
	FeatureAllowlistTypePasswordlessLogin: {allowWhenEmpty: false},
}

func featureAllowsWhenAllowlistEmpty(featureType string) bool {
	config, ok := featureAllowlistConfigs[featureType]
	if !ok {
		// 保持历史兼容：尚未注册配置的旧类型沿用空表放行语义。
		return true
	}
	return config.allowWhenEmpty
}

// FeatureAllowlist 全局功能白名单表（无 Identifier 隔离字段，跨租户）。
//
// 用于平台维度按 type 控制某项功能对哪些租户（identifier）开放。
// 一行 = 「某 type 下放行某个租户」。
//
// 空表行为由 featureAllowlistConfigs 集中定义；存在记录时，仅精确命中的
// identifier 放行。调用方只传 feature type，避免 type 与空表策略错配。
type FeatureAllowlist struct {
	ID         uint      `gorm:"primaryKey"`
	Type       string    `gorm:"uniqueIndex:idx_feature_allowlist_type_identifier;size:64;not null"` // 功能类别
	Identifier string    `gorm:"uniqueIndex:idx_feature_allowlist_type_identifier;size:191;not null"`
	Note       string    `gorm:"size:255;default:''"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

// IsFeatureAllowed 判断给定 (type, identifier) 是否按白名单放行。
//
// 空表行为由 featureAllowlistConfigs 按 feature type 决定。使用 DBGlobal
// 绕过 GORM 自动注入的 identifier 过滤回调（FeatureAllowlist 是全局表）。
func IsFeatureAllowed(ctx context.Context, featureType, identifier string) (bool, error) {
	if featureType == "" {
		// 调用方传空 type 视为不做白名单控制（防御性默认）
		return true, nil
	}
	db := DBGlobal(ctx)

	var total int64
	if err := db.Model(&FeatureAllowlist{}).
		Where("type = ?", featureType).
		Count(&total).Error; err != nil {
		return false, err
	}
	if total == 0 {
		return featureAllowsWhenAllowlistEmpty(featureType), nil
	}

	var hit FeatureAllowlist
	err := db.Where("type = ? AND identifier = ?", featureType, identifier).
		First(&hit).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
