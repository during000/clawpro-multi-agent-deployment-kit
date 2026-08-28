package model

import "time"

// MemoryPlanGroupPolicy 记忆分组策略绑定表。
// 每一行表示"某个分组被某条策略选中，对应某个 plan"。
// group_id 唯一索引保证同一分组不可同时出现在两条策略中。
type MemoryPlanGroupPolicy struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID   uint      `gorm:"not null;uniqueIndex:idx_mpgp_group" json:"group_id"`
	Plan      string    `gorm:"size:16;not null" json:"plan"`       // off / free / pro
	Priority  int       `gorm:"not null;default:1" json:"priority"` // 1=第一条策略, 2=第二条策略
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (MemoryPlanGroupPolicy) TableName() string {
	return "memory_plan_group_policies"
}
