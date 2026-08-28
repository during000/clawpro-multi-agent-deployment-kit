package model

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// 通知类型常量
const (
	NotifyTypeAdminDelete     = "admin_delete"     // 管理员删除
	NotifyTypeExternalDestroy = "external_destroy" // 外部销毁

	// 🆕 新增通知类型
	NotifyTypeInstanceCreateSuccess    = "instance_create_success"
	NotifyTypeInstanceDeleteSuccess    = "instance_delete_success"
	NotifyTypeInstanceUpgradeSuccess   = "instance_upgrade_success"
	NotifyTypeInstanceReinstallSuccess = "instance_reinstall_success"
	NotifyTypeInstanceCreateFailed     = "instance_create_failed"
	NotifyTypeInstanceReinstallFailed  = "instance_reinstall_failed"
	NotifyTypeInstanceUpgradeFailed    = "instance_upgrade_failed"
	NotifyTypeQuotaExceeded            = "quota_exceeded"
	NotifyTypeModelConfigFailed        = "model_config_failed"
	NotifyTypeChannelConfigFailed      = "channel_config_failed"
	NotifyTypeSkillInstallFailed       = "skill_install_failed"

	// 存量实例分组归属处理（stale-instances v1.0）
	NotifyTypeInstanceMigrated          = "instance_migrated"
	NotifyTypeInstanceHandoverInitiated = "instance_handover_initiated"
	NotifyTypeInstanceHandoverReceived  = "instance_handover_received"
	NotifyTypeInstanceHandoverAccepted  = "instance_handover_accepted"
	NotifyTypeInstanceHandoverRejected  = "instance_handover_rejected"
	NotifyTypeInstanceHandoverCancelled = "instance_handover_cancelled"
	NotifyTypeInstanceArchivedByAdmin   = "instance_archived_by_admin"
	NotifyTypePendingUserAction         = "instance_pending_user_action"
	NotifyTypeStaleGroupOneIDSync       = "instance_stale_group_oneid_sync"
)

// 消息类别常量
const (
	NotifCategorySuccess = "success" // 成功：创建成功、升级完成等
	NotifCategoryError   = "error"   // 错误：操作失败、配额超限等
	NotifCategoryNotice  = "notice"  // 通知：管理员删除、外部销毁等
)

// NotifErrorDetail 存储错误类通知的结构化信息，供前端"复制详情"使用
type NotifErrorDetail struct {
	Error      string `json:"error"`
	Detail     string `json:"detail,omitempty"`
	RequestId  string `json:"request_id,omitempty"`
	InstanceId string `json:"instance_id,omitempty"`
}

// Notification 消息通知模型
type Notification struct {
	gorm.Model
	Identifier   string     `gorm:"index:idx_notifications_identifier_category;index;default:''"`                           // 多租户标识，MySQL 模式下自动填充和过滤
	UserID       uint       `gorm:"not null;index"`                                                                         // 接收用户 ID
	InstanceID   uint       `gorm:"not null;index"`                                                                         // 关联实例 ID
	InstanceName string     `gorm:"type:varchar(255)"`                                                                      // 实例名称（冗余，避免 JOIN）
	Type         string     `gorm:"type:varchar(32);not null;index"`                                                        // 通知类型：admin_delete/external_destroy/...
	Category     string     `gorm:"type:varchar(16);not null;index:idx_notifications_identifier_category;default:'notice'"` // 消息类别：success/error/notice
	Title        string     `gorm:"type:varchar(255);not null"`                                                             // 通知标题
	Message      string     `gorm:"type:text"`                                                                              // 通知详情
	ErrorDetail  string     `gorm:"type:text"`                                                                              // 错误详情 JSON（仅 error 类有值）
	IsRead       bool       `gorm:"default:false;index"`                                                                    // 是否已读
	ReadAt       *time.Time `gorm:"default:null"`                                                                           // 阅读时间
}

// CreateNotification 创建一条新通知（兼容已有调用，Category 默认为 notice）
func CreateNotification(ctx context.Context, userID, instanceID uint, instanceName, notifyType, title, message string) error {
	notif := Notification{
		UserID:       userID,
		InstanceID:   instanceID,
		InstanceName: instanceName,
		Type:         notifyType,
		Category:     NotifCategoryNotice,
		Title:        title,
		Message:      message,
		IsRead:       false,
	}
	return DB(ctx).Create(&notif).Error
}

// CreateNotificationWithCategory 创建带类别的通知
func CreateNotificationWithCategory(
	ctx context.Context,
	userID, instanceID uint,
	instanceName, notifyType, category, title, message string,
	errorDetail *NotifErrorDetail,
) error {
	detailJSON := ""
	if errorDetail != nil {
		if b, err := json.Marshal(errorDetail); err == nil {
			detailJSON = string(b)
		}
	}
	notif := Notification{
		UserID:       userID,
		InstanceID:   instanceID,
		InstanceName: instanceName,
		Type:         notifyType,
		Category:     category,
		Title:        title,
		Message:      message,
		ErrorDetail:  detailJSON,
		IsRead:       false,
	}
	if err := DB(ctx).Create(&notif).Error; err != nil {
		slog.Error("[Notification] 创建通知失败", "user_id", userID, "instance_id", instanceID, "type", notifyType, "category", category, "error", err)
		return err
	}
	return nil
}

// CreateSuccessNotification 快捷方法：创建 success 类通知
func CreateSuccessNotification(ctx context.Context, userID, instanceID uint, instanceName, notifyType, title, message string) error {
	return CreateNotificationWithCategory(
		ctx, userID, instanceID, instanceName,
		notifyType, NotifCategorySuccess,
		title, message, nil,
	)
}

// MarkNotificationRead 标记通知为已读
func MarkNotificationRead(ctx context.Context, notificationID, userID uint) error {
	now := time.Now()
	return DB(ctx).Model(&Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		}).Error
}

// MarkAllNotificationsRead 标记用户通知为已读
// category 为空时标记所有类别，非空时仅标记指定类别
func MarkAllNotificationsRead(ctx context.Context, userID uint, category string) error {
	now := time.Now()
	query := DB(ctx).Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false)
	if category != "" {
		query = query.Where("category = ?", category)
	}
	return query.Updates(map[string]interface{}{
		"is_read": true,
		"read_at": &now,
	}).Error
}

// GetUnreadNotificationCount 获取用户未读通知数量
func GetUnreadNotificationCount(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := DB(ctx).Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// UnreadCountByCategory 按 category 分类的未读计数
type UnreadCountByCategory struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

// GetUnreadNotificationCountByCategory 获取用户未读通知的分类计数
func GetUnreadNotificationCountByCategory(ctx context.Context, userID uint) (int64, map[string]int64, error) {
	var total int64
	if err := DB(ctx).Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var rows []UnreadCountByCategory
	if err := DB(ctx).Model(&Notification{}).
		Select("category, count(*) as count").
		Where("user_id = ? AND is_read = ?", userID, false).
		Group("category").
		Find(&rows).Error; err != nil {
		return total, nil, err
	}

	byCategory := make(map[string]int64)
	for _, r := range rows {
		byCategory[r.Category] = r.Count
	}
	return total, byCategory, nil
}

// normalizePagination 校验并修正分页参数，确保 page >= 1、1 <= pageSize <= 100
func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	} else if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

// GetUserNotifications 获取用户通知列表（分页）
// isRead: nil=全部, *true=已读, *false=未读
// category: ""=全部, "success"/"error"/"notice"=按类别过滤
func GetUserNotifications(ctx context.Context, userID uint, page, pageSize int, isRead *bool, category string) ([]Notification, int64, error) {
	var notifications []Notification
	var total int64

	page, pageSize = normalizePagination(page, pageSize)

	query := DB(ctx).Model(&Notification{}).Where("user_id = ?", userID)

	if isRead != nil {
		query = query.Where("is_read = ?", *isRead)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&notifications).Error; err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

// CleanupExpiredNotifications 删除超过 retentionDays 天的通知（物理删除）
// 以 created_at（消息创建时间）为准，无论是否已读，超期统一清理
func CleanupExpiredNotifications(ctx context.Context, retentionDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result := DB(ctx).Unscoped().Where("created_at < ?", cutoff).Delete(&Notification{})
	return result.RowsAffected, result.Error
}

// DeleteNotification 删除指定通知（用户只能删除自己的通知）
func DeleteNotification(ctx context.Context, notificationID, userID uint) (int64, error) {
	result := DB(ctx).Unscoped().Where("id = ? AND user_id = ?", notificationID, userID).Delete(&Notification{})
	return result.RowsAffected, result.Error
}

// DeleteNotifications 批量删除指定通知（用户只能删除自己的通知）
func DeleteNotifications(ctx context.Context, notificationIDs []uint, userID uint) (int64, error) {
	if len(notificationIDs) == 0 {
		return 0, nil
	}
	result := DB(ctx).Unscoped().Where("id IN ? AND user_id = ?", notificationIDs, userID).Delete(&Notification{})
	return result.RowsAffected, result.Error
}

// DeleteAllNotifications 删除用户的所有通知
// category 为空时删除全部，非空时仅删除指定类别
func DeleteAllNotifications(ctx context.Context, userID uint, category string) (int64, error) {
	query := DB(ctx).Unscoped().Where("user_id = ?", userID)
	if category != "" {
		query = query.Where("category = ?", category)
	}
	result := query.Delete(&Notification{})
	return result.RowsAffected, result.Error
}
