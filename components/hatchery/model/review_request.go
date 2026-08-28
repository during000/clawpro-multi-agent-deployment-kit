package model

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// 资源类型常量
const (
	ResourceTypeSkill = "skill"
	ResourceTypeMcp   = "mcp"  // 未来扩展
	ResourceTypeRule  = "rule" // 未来扩展
)

// 操作类型常量
const (
	ActionTypePublish  = "publish"
	ActionTypeTakedown = "takedown"
)

// 审批状态常量
const (
	ReviewStatusPending   = "pending"
	ReviewStatusApproved  = "approved"
	ReviewStatusRejected  = "rejected"
	ReviewStatusWithdrawn = "withdrawn" // 员工主动撤回（终态）
)

// ReviewRequest 通用审核申请单。
// resource_type + resource_id 关联被审核的资源（如 Skill）；
// action_type 表示操作类型（publish / takedown）；
// status 跟踪审批流程（pending → approved / rejected）。
type ReviewRequest struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	Identifier    string         `gorm:"index;default:''" json:"-"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	RequesterID   uint           `gorm:"not null;default:0" json:"requester_id"`                         // 申请人 user_id
	ResourceType  string         `gorm:"type:varchar(32);not null;default:'skill'" json:"resource_type"` // skill / mcp / rule
	ResourceID    uint           `gorm:"not null;default:0" json:"resource_id"`                          // 关联资源 ID（如 Skill.ID）
	ActionType    string         `gorm:"type:varchar(16);not null;default:'publish'" json:"action_type"` // publish / takedown
	Slug          string         `gorm:"type:varchar(191);not null;default:''" json:"slug"`              // 冗余存储，便于互斥查询
	Status        string         `gorm:"type:varchar(16);not null;default:'pending'" json:"status"`      // pending / approved / rejected
	Reason        string         `gorm:"type:text" json:"reason"`                                        // 申请理由（takedown 必填）
	ReviewerID    uint           `gorm:"not null;default:0" json:"reviewer_id"`                          // 审核人 user_id
	ReviewedAt    *time.Time     `gorm:"default:null" json:"reviewed_at"`                                // 审核时间
	ReviewComment string         `gorm:"type:text" json:"review_comment"`                                // 审核意见
}

// HasPendingRequest 检查指定 slug 是否有进行中的申请（互斥校验）。
// 同一 resource_type + slug 只允许存在一个 status=pending 的申请。
func HasPendingRequest(ctx context.Context, resourceType, slug string) bool {
	var count int64
	if err := DB(ctx).Model(&ReviewRequest{}).
		Where("resource_type = ? AND slug = ? AND status = ?", resourceType, slug, ReviewStatusPending).
		Count(&count).Error; err != nil {
		slog.Error("HasPendingRequest count failed", "resource_type", resourceType, "slug", slug, "error", err)
		return false
	}
	return count > 0
}
