package model

import "time"

// PublicPlugin 收藏的公共插件
type PublicPlugin struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	Identifier     string    `gorm:"index;default:''" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Name           string    `gorm:"not null;default:''" json:"name"`
	Slug           string    `gorm:"uniqueIndex:idx_public_plugin_slug;not null;default:''" json:"slug"`
	PluginID       string    `gorm:"not null;default:''" json:"plugin_id"`
	Version        string    `gorm:"not null;default:''" json:"version"`
	Description    string    `gorm:"type:text" json:"description"`
	NpmPackage     string    `gorm:"not null;default:''" json:"npm_package"`
	TotalDownloads int64     `gorm:"not null;default:0" json:"total_downloads"`
	TotalFavorites int64     `gorm:"not null;default:0" json:"total_favorites"`
}
