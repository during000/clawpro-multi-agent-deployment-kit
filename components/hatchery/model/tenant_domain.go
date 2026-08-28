package model

import (
	"context"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"hatchery/common"
	"hatchery/i18n"
)

// TenantDomain 域名→租户映射表（全局表，无 Identifier 隔离字段）。
// 一个租户可绑定多个域名，域名全局唯一。
type TenantDomain struct {
	ID         uint      `gorm:"primaryKey"`
	Domain     string    `gorm:"uniqueIndex;size:255;not null"` // 完整域名，如 a.tcaisite.com
	Identifier string    `gorm:"index;size:191;not null"`       // 租户标识
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

// tenantCache 缓存 domain → TenantSnapshot 映射，避免每次请求查库。
// 首次访问 cache miss 时查库回填；通过 WarmTenantCache / InvalidateTenantCache 显式管理。
var tenantCache sync.Map // domain(string) -> common.TenantSnapshot

// IsDuplicateError 判断 GORM 错误是否为唯一约束冲突（兼容 MySQL + SQLite）。
func IsDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate") || strings.Contains(msg, "UNIQUE")
}

// ResolveTenant 从域名解析租户快照。
// 优先查内存缓存，未命中则查库回填。
// 返回 error 表示域名未注册或租户配置不存在。
func ResolveTenant(ctx context.Context, domain string) (common.TenantSnapshot, error) {
	if v, ok := tenantCache.Load(domain); ok {
		return v.(common.TenantSnapshot), nil
	}

	var td TenantDomain
	if err := DBGlobal(ctx).Where("domain = ?", domain).First(&td).Error; err != nil {
		return common.TenantSnapshot{}, common.I18nError(i18n.MsgUnknownDomain, domain)
	}

	var config SiteConfig
	if err := DBGlobal(ctx).Where("identifier = ?", td.Identifier).First(&config).Error; err != nil {
		return common.TenantSnapshot{}, common.I18nError(i18n.MsgTenantConfigNotFound, td.Identifier)
	}

	snap := SnapFromConfig(config)
	tenantCache.Store(domain, snap)
	return snap, nil
}

// WarmTenantCache 预热缓存：在新增域名映射后调用，避免首次请求穿透 DB。
func WarmTenantCache(domain string, snap common.TenantSnapshot) {
	tenantCache.Store(domain, snap)
}

// InvalidateTenantCache 移除指定域名的缓存（删除域名映射时调用）。
func InvalidateTenantCache(domain string) {
	tenantCache.Delete(domain)
}

// ExtractHostFromURL 从完整 URL（如 https://a.example.com）或 host:port 中提取纯主机名。
func ExtractHostFromURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	// 带 scheme 的完整 URL：url.Parse().Hostname() 自动处理端口和 IPv6
	if strings.Contains(rawURL, "://") {
		if u, err := url.Parse(rawURL); err == nil && u.Hostname() != "" {
			return u.Hostname()
		}
	}
	// 无 scheme 的 host:port 格式
	if h, _, err := net.SplitHostPort(rawURL); err == nil {
		return h
	}
	return rawURL
}

// CreateInitAdmin 在给定的事务内创建初始管理员用户。
// 用于 /tenants/init 接口在新租户内创建管理员。
func CreateInitAdmin(tx *gorm.DB, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return common.I18nRichError(err, i18n.MsgPasswordEncryptFailed)
	}
	admin := User{Username: username, Password: string(hash), Role: "admin"}
	if err := tx.Create(&admin).Error; err != nil {
		return common.I18nRichError(err, i18n.MsgTenantCreateAdminFailed)
	}
	slog.Info("[Tenant] 创建初始管理员", "username", username)
	return nil
}
