package model

import "time"

// PublicSkill 收藏的公共技能
type PublicSkill struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	Identifier     string    `gorm:"index;default:''"` // 多租户标识，MySQL 模式下自动填充和过滤
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Name           string    `gorm:"not null;default:''" json:"name"`
	Slug           string    `gorm:"not null;default:''" json:"slug"`
	Version        string    `gorm:"not null;default:''" json:"version"`
	Description    string    `gorm:"type:text;not null" json:"description"`
	TotalDownloads int64     `gorm:"not null;default:0" json:"total_downloads"`
	TotalFavorites int64     `gorm:"not null;default:0" json:"total_favorites"`
}
