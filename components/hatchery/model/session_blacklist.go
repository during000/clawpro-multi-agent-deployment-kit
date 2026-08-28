package model

import (
	"context"
	"log/slog"
	"time"
)

// SessionBlacklist 记录被 OneID 登出的 session，防止持有旧 cookie 的用户继续访问。
// 支持双维度吊销：
//   - sid（OneID 登录会话唯一标识）：精确吊销特定会话
//   - sub（OneID 用户唯一标识）：当 id_token 不含 sid 时，按用户维度吊销
//
// 记录只需保留到 ExpireAt（OneID logout_token 的 exp），之后即可安全清理。
type SessionBlacklist struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	Identifier string    `gorm:"uniqueIndex:idx_sid_identifier;index;default:''"` // 多租户标识，MySQL 模式下自动填充和过滤
	OneIDSid   string    `gorm:"uniqueIndex:idx_sid_identifier;not null"`         // OneID 登录会话唯一标识（logout_token.sid）
	OneIDSub   string    `gorm:"index"`                                           // OneID 用户唯一标识（logout_token.sub），用于 fallback 吊销
	ExpireAt   time.Time `gorm:"not null"`                                        // OneID 侧该会话的过期时间
	CreatedAt  time.Time
}

// RevokeSession 将 sid（及可选的 sub）加入黑名单。
// 若该 sid 已存在则跳过（幂等）。
func RevokeSession(ctx context.Context, sid string, sub string, expireAt time.Time) error {
	entry := SessionBlacklist{
		OneIDSid: sid,
		OneIDSub: sub,
		ExpireAt: expireAt,
	}
	// INSERT OR IGNORE：重复收到同一 logout_token 时不报错
	if err := DB(ctx).Where(SessionBlacklist{OneIDSid: sid}).
		Attrs(SessionBlacklist{ExpireAt: expireAt, OneIDSub: sub}).
		FirstOrCreate(&entry).Error; err != nil {
		slog.Error("RevokeSession failed", "sid", sid, "sub", sub, "err", err)
		return err
	}
	return nil
}

// IsSessionRevoked 检查给定 sid 或 sub 是否已被吊销（在黑名单内且未过期）。
// sid 和 sub 独立检查，任一命中即返回 true：
//   - sid 非空时按 sid 精确匹配，命中则直接吊销；
//   - sub 非空时按 sub 匹配（无论 sid 是否已匹配），用于覆盖
//     "logout_token 无 sid 但有 sub" 的场景，以及同用户多设备统一踢出。
//
// loginAt 参数用于 sub 维度的时间比较：只有在用户本次登录之后写入的黑名单记录才对当前 session 生效。
// 也就是说如果用户在黑名单写入之后重新登录，新 session 不会被误判为已吊销。
// sid 维度不需要 loginAt 比较，因为每次登录会产生新的 sid。
func IsSessionRevoked(ctx context.Context, sid, sub string, loginAt time.Time) bool {
	now := time.Now()
	if sid != "" {
		var count int64
		DB(ctx).Model(&SessionBlacklist{}).
			Where("one_id_sid = ? AND expire_at > ?", sid, now).
			Count(&count)
		if count > 0 {
			return true
		}
	}
	if sub != "" {
		var count int64
		DB(ctx).Model(&SessionBlacklist{}).
			Where("one_id_sub = ? AND expire_at > ? AND created_at > ?", sub, now, loginAt).
			Count(&count)
		return count > 0
	}
	return false
}

// IsSidRevoked 检查给定 sid 是否已被吊销（在黑名单内且未过期）。
// 保留向后兼容，内部调用 IsSessionRevoked。
func IsSidRevoked(ctx context.Context, sid string) bool {
	return IsSessionRevoked(ctx, sid, "", time.Time{})
}

// CleanExpiredSessions 删除 ExpireAt 已过期的黑名单记录，供定期维护调用。
func CleanExpiredSessions(ctx context.Context) {
	result := DB(ctx).Where("expire_at <= ?", time.Now()).Delete(&SessionBlacklist{})
	if result.Error != nil {
		slog.Error("CleanExpiredSessions failed", "err", result.Error)
	} else if result.RowsAffected > 0 {
		slog.Info("Cleaned expired session blacklist entries", "count", result.RowsAffected)
	}
}
