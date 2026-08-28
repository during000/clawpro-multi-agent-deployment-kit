package model

import "time"

// 安装状态枚举
const (
	SkillInstallNone      = 0 // 未安装
	SkillInstalling       = 1 // 安装中
	SkillInstallSuccess   = 2 // 安装成功
	SkillInstallFailed    = 3 // 安装失败
	SkillInstallCancelled = 4 // 已取消
)

// SkillInstallation 技能安装记录
type SkillInstallation struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	Identifier    string    `gorm:"index;default:''"` // 多租户标识，MySQL 模式下自动填充和过滤
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	InstanceID    uint      `gorm:"not null;index" json:"instance_id"`
	Name          string    `gorm:"not null;default:''" json:"name"`
	Slug          string    `gorm:"not null;default:''" json:"slug"`
	Version       string    `gorm:"not null;default:''" json:"version"`
	CosZipKey     string    `gorm:"not null;default:''" json:"cos_zip_key"`
	InstallStatus int       `gorm:"not null;default:0" json:"install_status"`
	ErrorMessage  string    `gorm:"type:text;not null" json:"error_message"`
}
