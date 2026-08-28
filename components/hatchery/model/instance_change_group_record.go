package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// InstanceChangeGroupRecord 存量实例分组归属处理的处理记录（stale-instances v1.0）。
// 表名 instance_change_group_records 与 DB 层主表 instances 对齐。
type InstanceChangeGroupRecord struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Identifier string `gorm:"type:varchar(191);not null;default:'';index:idx_icgr_instance,priority:1;index:idx_icgr_user,priority:1;index:idx_icgr_group,priority:1;index:idx_icgr_actor,priority:1" json:"-"`
	// instances.id（自增主键）
	InstancePK uint `gorm:"not null;index:idx_icgr_instance,priority:2" json:"instance_pk"`
	// CVM ins-xxx 字符串（便于运维），与 InstancePK 同时保留
	// 类型对齐 instances.instance_id：varchar(191) utf8mb4_unicode_ci
	InstanceID    string `gorm:"type:varchar(191);not null;default:''" json:"instance_id"`
	UserIDBefore  uint   `gorm:"not null;default:0;index:idx_icgr_user,priority:2" json:"user_id_before"`
	UserIDAfter   uint   `gorm:"not null;default:0" json:"user_id_after"`
	GroupIDBefore uint   `gorm:"not null;default:0;index:idx_icgr_group,priority:2" json:"group_id_before"`
	GroupIDAfter  uint   `gorm:"not null;default:0" json:"group_id_after"`
	// 操作类型: migrate / handover / pending_user / archive_stop / parent_change_pending /
	//           user_self_rebind / user_self_handover_initiate / user_self_handover_accept /
	//           user_self_handover_reject / user_self_handover_cancel
	Action string `gorm:"type:varchar(48);not null" json:"action"`
	// 操作发起者类型: admin / user / oneid_sync
	ActorType string `gorm:"type:varchar(16);not null;index:idx_icgr_actor,priority:2" json:"actor_type"`
	ActorID   uint   `gorm:"not null;default:0;index:idx_icgr_actor,priority:3" json:"actor_id"`
	// 触发来源: user_edit / user_added_to_group / group_parent_change / oneid_sync /
	//          list_page_followup / user_self
	TriggerSource string `gorm:"type:varchar(32);not null" json:"trigger_source"`
	ExtraJSON     string `gorm:"type:varchar(2048);not null;default:'{}'" json:"extra_json"`
	// 关联的通知 ID；0 表示无
	NotificationID uint      `gorm:"not null;default:0" json:"notification_id"`
	CreatedAt      time.Time `gorm:"not null;index:idx_icgr_instance,priority:3;index:idx_icgr_user,priority:3;index:idx_icgr_group,priority:3;index:idx_icgr_actor,priority:4" json:"created_at"`
}

// Action / ActorType / TriggerSource 常量
const (
	ICGRActionMigrate              = "migrate"
	ICGRActionHandover             = "handover"
	ICGRActionPendingUser          = "pending_user"
	ICGRActionArchiveStop          = "archive_stop"
	ICGRActionParentChangePending  = "parent_change_pending"
	ICGRActionUserRebind           = "user_self_rebind"
	ICGRActionUserHandoverInit     = "user_self_handover_initiate"
	ICGRActionUserHandoverAccept   = "user_self_handover_accept"
	ICGRActionUserHandoverReject   = "user_self_handover_reject"
	ICGRActionUserHandoverCancel   = "user_self_handover_cancel"

	ICGRActorAdmin     = "admin"
	ICGRActorUser      = "user"
	ICGRActorOneIDSync = "oneid_sync"

	ICGRTriggerUserEdit          = "user_edit"
	ICGRTriggerUserAddedToGroup  = "user_added_to_group"
	ICGRTriggerGroupParentChange = "group_parent_change"
	ICGRTriggerOneIDSync         = "oneid_sync"
	ICGRTriggerListPageFollowup  = "list_page_followup"
	ICGRTriggerUserSelf          = "user_self"
)

// CreateICGR 写入一条处理记录（事务内调用时传 *gorm.DB；普通调用走 ctx）。
func CreateICGR(ctx context.Context, r *InstanceChangeGroupRecord) error {
	return DB(ctx).Create(r).Error
}

// CreateICGRTx 在事务里写一条处理记录。
func CreateICGRTx(tx *gorm.DB, r *InstanceChangeGroupRecord) error {
	return tx.Create(r).Error
}

// ListICGRsParams 处理记录列表查询参数。
//
// GroupID 用 *uint 区分"未传"与"显式 0（未分组）"两种语义：
//   - nil      → 不过滤
//   - 非 nil 0 → 过滤 group_id_before=0 OR group_id_after=0（"未分组"语义）
//   - 非 nil >0 → 过滤指定 group_id
//
// UserID / InstancePK 仍用 uint，因为 0 在业务上不会出现（user.id 与 instance.id
// 均为自增主键 ≥ 1），无需消歧。
type ListICGRsParams struct {
	InstancePK    uint
	InstanceID    string
	UserID        uint
	GroupID       *uint
	Action        string
	ActorType     string
	TriggerSource string
	From          *time.Time
	To            *time.Time
	Page          int
	PageSize      int
}

// ListICGRs 按条件分页查询处理记录，按时间倒序返回。
func ListICGRs(ctx context.Context, p ListICGRsParams) ([]InstanceChangeGroupRecord, int64, error) {
	q := DB(ctx).Model(&InstanceChangeGroupRecord{})
	if p.InstancePK > 0 {
		q = q.Where("instance_pk = ?", p.InstancePK)
	}
	if p.InstanceID != "" {
		q = q.Where("instance_id = ?", p.InstanceID)
	}
	if p.UserID > 0 {
		q = q.Where("user_id_before = ? OR user_id_after = ?", p.UserID, p.UserID)
	}
	if p.GroupID != nil {
		q = q.Where("group_id_before = ? OR group_id_after = ?", *p.GroupID, *p.GroupID)
	}
	if p.Action != "" {
		q = q.Where("action = ?", p.Action)
	}
	if p.ActorType != "" {
		q = q.Where("actor_type = ?", p.ActorType)
	}
	if p.TriggerSource != "" {
		q = q.Where("trigger_source = ?", p.TriggerSource)
	}
	if p.From != nil {
		q = q.Where("created_at >= ?", *p.From)
	}
	if p.To != nil {
		q = q.Where("created_at < ?", *p.To)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := normalizePagination(p.Page, p.PageSize)
	offset := (page - 1) * pageSize
	var rows []InstanceChangeGroupRecord
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
