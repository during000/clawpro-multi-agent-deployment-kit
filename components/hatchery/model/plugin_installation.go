package model

import "time"

// 插件安装状态枚举
const (
	PluginInstallNone      = 0 // 未安装
	PluginInstalling       = 1 // 安装中
	PluginInstallSuccess   = 2 // 安装成功
	PluginInstallFailed    = 3 // 安装失败
	PluginInstallCancelled = 4 // 已取消
)

// PluginInstallation 插件安装记录
type PluginInstallation struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	Identifier    string    `gorm:"index;default:''" json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	InstanceID    uint      `gorm:"not null;uniqueIndex:idx_plugin_inst_slug" json:"instance_id"`
	Name          string    `gorm:"not null;default:''" json:"name"`
	Slug          string    `gorm:"not null;uniqueIndex:idx_plugin_inst_slug" json:"slug"`
	PluginID      string    `gorm:"not null;default:''" json:"plugin_id"`       // openclaw 运行时 id
	Version       string    `gorm:"not null;default:''" json:"version"`
	CosZipKey     string    `gorm:"not null;default:''" json:"cos_zip_key"`
	NpmPackage    string    `gorm:"not null;default:''" json:"npm_package"`
	InstallMode   string    `gorm:"not null;default:'smh'" json:"install_mode"` // smh / npm
	Kind          string    `gorm:"not null;default:''" json:"kind"`
	InstallStatus int       `gorm:"not null;default:0" json:"install_status"`
	ErrorMessage  string    `gorm:"type:text;not null" json:"error_message"`
}
