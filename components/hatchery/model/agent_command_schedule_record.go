package model

import (
	"context"
	"fmt"
	"time"
)

// ============================================================================
// AgentCommandScheduleRecord —— 定时任务的历史执行记录
//
// 极简 append-only：每次 schedule 触发一次 dispatch 就插一条，只存 dispatch_slug。
// 执行状态/计数不冗余存储，由 controller 实时查 dispatch 表拼装返回（唯一真相源）。
// dispatch 是永久审计记录（不软删），slug 始终可查。
// ============================================================================

// AgentCommandScheduleRecord 一次定时触发的执行记录。
type AgentCommandScheduleRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	Identifier   string `gorm:"index;default:''" json:"-"`
	ScheduleID   uint   `gorm:"not null;default:0;index:idx_sched_record_sched,priority:1" json:"schedule_id"`
	DispatchSlug string `gorm:"type:varchar(32);not null;default:''" json:"dispatch_slug"`
}

func (AgentCommandScheduleRecord) TableName() string { return "agent_command_schedule_records" }

// CreateScheduleRecord 追加一条触发记录。
func CreateScheduleRecord(ctx context.Context, scheduleID uint, dispatchSlug string) error {
	rec := &AgentCommandScheduleRecord{
		ScheduleID:   scheduleID,
		DispatchSlug: dispatchSlug,
	}
	if err := DB(ctx).Create(rec).Error; err != nil {
		return fmt.Errorf("create schedule record: %w", err)
	}
	return nil
}

// ListScheduleRecords 按 schedule_id 倒序分页列出执行记录。
func ListScheduleRecords(ctx context.Context, scheduleID uint, page, pageSize int) ([]AgentCommandScheduleRecord, int64, error) {
	tx := DB(ctx).Model(&AgentCommandScheduleRecord{}).Where("schedule_id = ?", scheduleID)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count schedule records: %w", err)
	}
	if page < 1 {
		page = 1
	}
	var rows []AgentCommandScheduleRecord
	if err := tx.Order("id desc").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list schedule records: %w", err)
	}
	return rows, total, nil
}
