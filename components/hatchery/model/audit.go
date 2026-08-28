package model

import (
	"context"
	"log/slog"
	"time"
)

type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Identifier string    `gorm:"index;index:idx_audit_logs_identifier_user_id,priority:1;index:idx_audit_logs_identifier_username,priority:1;index:idx_audit_logs_identifier_resource_id,priority:1;default:''" json:"-"` // 多租户标识，MySQL 模式下自动填充和过滤
	StartedAt  time.Time `gorm:"not null;default:'1970-01-01 00:00:00'" json:"started_at"`
	CreatedAt  time.Time `gorm:"index;not null" json:"created_at"`
	UserID     uint      `gorm:"index;index:idx_audit_logs_identifier_user_id,priority:2;not null;default:0" json:"user_id"`
	Username   string    `gorm:"index:idx_audit_logs_identifier_username,priority:2;not null;default:''" json:"username"`
	Action     string    `gorm:"index;not null;default:''" json:"action"`
	Resource   string    `gorm:"not null;default:''" json:"resource"`
	ResourceID string    `gorm:"index:idx_audit_logs_identifier_resource_id,priority:2" json:"resource_id"`
	Status     string    `gorm:"not null;default:''" json:"status"`
}

func LogAudit(ctx context.Context, startedAt time.Time, userID uint, username, action, resource, resourceID, status string) {
	if err := DB(ctx).Create(&AuditLog{
		StartedAt:  startedAt,
		UserID:     userID,
		Username:   username,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Status:     status,
	}).Error; err != nil {
		slog.Error("[Audit] 写入审计日志失败", "error", err, "action", action, "resource_id", resourceID, "user_id", userID)
	}
}
