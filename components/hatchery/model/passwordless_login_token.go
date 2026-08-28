package model

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// ErrPasswordlessLoginTokenInvalid intentionally combines unknown, expired,
// replayed, and cross-tenant tokens so callers cannot disclose token state.
var ErrPasswordlessLoginTokenInvalid = errors.New("invalid passwordless login token")

// PasswordlessLoginToken stores only the digest of a short-lived login token.
// Identifier is populated and filtered by the tenant GORM callbacks.
type PasswordlessLoginToken struct {
	ID         uint      `gorm:"primaryKey"`
	Identifier string    `gorm:"size:191;not null;default:'';index:idx_passwordless_login_tokens_identifier"`
	TokenHash  string    `gorm:"type:char(64);not null;uniqueIndex:idx_passwordless_login_tokens_hash"`
	UserID     uint      `gorm:"not null;index:idx_passwordless_login_tokens_user_id"`
	ExpiresAt  time.Time `gorm:"not null;index:idx_passwordless_login_tokens_expires_at"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

// CreatePasswordlessLoginToken persists a token digest in the current tenant.
func CreatePasswordlessLoginToken(ctx context.Context, tokenHash string, userID uint, expiresAt time.Time) (*PasswordlessLoginToken, error) {
	record := &PasswordlessLoginToken{
		TokenHash: tokenHash,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
	if err := DB(ctx).Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

// ConsumePasswordlessLoginToken atomically grants one caller ownership of a
// valid token by deleting it conditionally. Exactly one concurrent caller can
// observe RowsAffected == 1.
func ConsumePasswordlessLoginToken(ctx context.Context, tokenHash string, now time.Time) (*PasswordlessLoginToken, error) {
	var record PasswordlessLoginToken
	if err := DB(ctx).
		Where("token_hash = ? AND expires_at > ?", tokenHash, now).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPasswordlessLoginTokenInvalid
		}
		return nil, err
	}

	result := DB(ctx).
		Where("id = ? AND token_hash = ? AND expires_at > ?", record.ID, tokenHash, now).
		Delete(&PasswordlessLoginToken{})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, ErrPasswordlessLoginTokenInvalid
	}
	return &record, nil
}

// DeleteExpiredPasswordlessLoginTokens removes expired records only from the
// current tenant, as enforced by the tenant GORM callbacks.
func DeleteExpiredPasswordlessLoginTokens(ctx context.Context, now time.Time) error {
	return DB(ctx).
		Where("expires_at <= ?", now).
		Delete(&PasswordlessLoginToken{}).Error
}
