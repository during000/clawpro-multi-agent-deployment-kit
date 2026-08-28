package model

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
)

// BannedError 表示用户已被封禁（软删除）。
type BannedError struct{}

func (BannedError) Error() string { return "账户已被封禁" }

// RevokedError 表示 OneID session 已被吊销（backchannel logout），需要清 cookie 并重新登录。
type RevokedError struct{}

func (RevokedError) Error() string { return "登录态已失效，请重新登录" }

type User struct {
	gorm.Model
	Identifier      string  `gorm:"uniqueIndex:idx_username_identifier;uniqueIndex:idx_oneid_sub_identifier;default:''"` // 多租户标识，MySQL 模式下自动填充和过滤
	Username        string  `gorm:"uniqueIndex:idx_username_identifier;not null;default:''"`
	Email           string  `gorm:"default:''"`
	Password        string  `gorm:"not null;default:''" json:"-"`
	Role            string  `gorm:"not null;default:user"`
	InstanceQuota   int     `gorm:"not null;default:1"`   // 实例数量上限
	TokenQuotaDay   int     `gorm:"not null;default:-1"`  // daily token quota, -1 means unlimited (legacy, prefer TokenQuotaRules)
	TokenQuotaRules string  `gorm:"type:text"`            // JSON: [{"mode":"day","limit":100000}], nil/empty = fallback to TokenQuotaDay
	VpcId           string  `gorm:"default:''"`           // 用户级VPC已不再使用，当前改为企业级VPC
	SubnetIds       string  `gorm:"size:1024;default:''"` // JSON map: zone -> subnetId 用户级VPC已不再使用，当前改为企业级VPC
	APIToken        *string `gorm:"uniqueIndex" json:"-"`
	OneIDSub        *string `gorm:"uniqueIndex:idx_oneid_sub_identifier"`          // OneID 用户唯一标识（允许为空，null 值不参与唯一约束）
	OneIDLoginName  *string `gorm:"column:oneid_login_name;size:191;default:null"` // OneID 登录名（username），与本地 username（姓名）分开存储

	// API Token 管理字段
	APITokenDisabled   bool       `gorm:"not null;default:false" json:"-"` // 管理员禁用标记
	APITokenCreatedAt  *time.Time `gorm:"default:null" json:"-"`           // Token 创建/重置时间
	APITokenLastUsedAt *time.Time `gorm:"default:null" json:"-"`           // 最近一次 API 调用时间
}

// IsInitialAdmin reports whether this user is the initial admin —
// i.e. the admin with the smallest ID (within the current tenant when using MySQL).
// This replaces the old hard-coded ID==1 check.
func (u *User) IsInitialAdmin(ctx context.Context) bool {
	if u.Role != "admin" {
		return false
	}
	firstAdmin := GetInitialAdmin(ctx)
	if firstAdmin == nil {
		return false
	}
	return u.ID == firstAdmin.ID
}

// GetInitialAdmin retrieves the admin user with the smallest ID (the initial admin).
// Returns nil if no admin exists.
func GetInitialAdmin(ctx context.Context) *User {
	var firstAdmin User
	if DB(ctx).Where("role = ?", "admin").Order("id ASC").First(&firstAdmin).Error != nil {
		return nil
	}
	return &firstAdmin
}

// GetSubnetMap parses SubnetIds JSON into a map.
func (u *User) GetSubnetMap() map[string]string {
	m := make(map[string]string)
	if u.SubnetIds != "" && u.SubnetIds != "{}" {
		if err := json.Unmarshal([]byte(u.SubnetIds), &m); err != nil {
			slog.Warn("failed to parse SubnetIds", "user", u.Username, "err", err)
		}
	}
	return m
}

// SetSubnetMap serializes the map to JSON and stores it in SubnetIds.
func (u *User) SetSubnetMap(m map[string]string) {
	data, _ := json.Marshal(m)
	u.SubnetIds = string(data)
}

// DisplayName returns the username.
func (u User) DisplayName() string {
	return u.Username
}

// HasAPIToken reports whether the user has a non-empty API Token.
func (u *User) HasAPIToken() bool {
	return u.APIToken != nil && *u.APIToken != ""
}

// TokenDisabledError 表示用户的 API Token 已被管理员禁用。
type TokenDisabledError struct{}

func (TokenDisabledError) Error() string { return "API Token 已被禁用" }

// GenerateRandomToken generates a cryptographically random token with the given prefix.
// The token format is: prefix + 48 hex chars (24 random bytes).
func GenerateRandomToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "hk-" + hex.EncodeToString(b), nil
}

// GenerateAPIToken generates a new "hk-" + 48-hex-char token, persists it, and returns the token string.
// Also updates APITokenCreatedAt and clears APITokenDisabled.
func GenerateAPIToken(ctx context.Context, userID uint) (string, error) {
	token, err := GenerateRandomToken()
	if err != nil {
		return "", err
	}
	now := time.Now()
	if err := DB(ctx).Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"api_token":            token,
		"api_token_disabled":   false,
		"api_token_created_at": now,
	}).Error; err != nil {
		return "", err
	}
	return token, nil
}

// RevokeAPIToken clears the user's API token and all related metadata.
func RevokeAPIToken(ctx context.Context, userID uint) error {
	return DB(ctx).Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"api_token":              nil,
		"api_token_disabled":     false,
		"api_token_created_at":   nil,
		"api_token_last_used_at": nil,
	}).Error
}

// GetUserByAPIToken looks up a user by their API token.
// 被封禁用户返回 (nil, BannedError)；Token 被禁用返回 (nil, TokenDisabledError)；未找到返回 (nil, nil)。
func GetUserByAPIToken(ctx context.Context, token string) (*User, error) {
	if token == "" {
		return nil, nil
	}
	var user User
	if DB(ctx).Unscoped().Where("api_token = ?", token).First(&user).Error != nil {
		return nil, nil
	}
	if user.DeletedAt.Valid {
		return nil, BannedError{}
	}
	if user.APITokenDisabled {
		return nil, TokenDisabledError{}
	}
	return &user, nil
}

// MaskAPIToken returns a masked version of the token: hk- + first 4 chars + **** + last 4 chars.
// If the token (after "hk-" prefix) is too short, returns the full token.
func MaskAPIToken(token string) string {
	const prefix = "hk-"
	if !strings.HasPrefix(token, prefix) || len(token) <= len(prefix)+8 {
		return token
	}
	body := token[len(prefix):]
	return prefix + body[:4] + "****" + body[len(body)-4:]
}

// SetAPITokenDisabled sets the disabled flag for a user's API Token.
func SetAPITokenDisabled(ctx context.Context, userID uint, disabled bool) error {
	return DB(ctx).Model(&User{}).Where("id = ?", userID).Update("api_token_disabled", disabled).Error
}

// UpdateAPITokenLastUsed updates the last-used timestamp for a user's API Token.
func UpdateAPITokenLastUsed(ctx context.Context, userID uint) error {
	return DB(ctx).Model(&User{}).Where("id = ?", userID).Update("api_token_last_used_at", time.Now()).Error
}

// UserTokenExport holds the exported token info for a single user.
type UserTokenExport struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

// BatchEnsureAPITokens iterates all (non-deleted) users, generates an API token
// for any user that doesn't already have one, and returns the full list.
func BatchEnsureAPITokens(ctx context.Context) ([]UserTokenExport, error) {
	var users []User
	if err := DB(ctx).Find(&users).Error; err != nil {
		return nil, err
	}

	results := make([]UserTokenExport, 0, len(users))
	for _, u := range users {
		token := ""
		if u.HasAPIToken() {
			token = *u.APIToken
		} else {
			var err error
			token, err = GenerateAPIToken(ctx, u.ID)
			if err != nil {
				return nil, err
			}
		}
		results = append(results, UserTokenExport{
			ID:       u.ID,
			Username: u.Username,
			Token:    token,
		})
	}
	return results, nil
}

// UserDailyTotalTokenUsage returns today's total tokens for a user across all models.
func UserDailyTotalTokenUsage(ctx context.Context, userID uint) int64 {
	var total int64
	today := LocalToday()
	DB(ctx).Model(&DailyUsageSummary{}).
		Where("user_id = ? AND date = ?", userID, today).
		Select("COALESCE(SUM(total_tokens), 0)").
		Scan(&total)
	return total
}

// ResolvedTokenQuotaRules returns the token quota rules stored on this user.
// If TokenQuotaRules (JSON) is set, parse and return it directly — even if the result
// is empty (e.g. "[]" means explicitly unlimited).
// Only when TokenQuotaRules is empty does it fallback to legacy TokenQuotaDay field.
// TokenQuotaDay=-1 is an explicit unlimited setting, represented as [].
func (u *User) ResolvedTokenQuotaRules() []TokenQuotaRule {
	if rules, ok := ParseTokenQuotaRules(u.TokenQuotaRules); ok {
		return rules // 包括 "[]"（显式为空 = 无限制）
	}
	// Fallback: legacy field → single-rule array (read-only, no DB write)
	if u.TokenQuotaDay >= 0 {
		return []TokenQuotaRule{{Mode: QuotaModeDay, Limit: u.TokenQuotaDay}}
	}
	return []TokenQuotaRule{} // token_quota_day=-1 means unlimited
}
