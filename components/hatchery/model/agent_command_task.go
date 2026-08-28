package model

import (
	"context"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"time"
)

// ============================================================================
// 状态机常量（参见 design.md §6 / spec.md "状态机三层"）
// ============================================================================

// AgentCommandTaskStatus 一台 instance 在一次 RunCommand 内的执行状态。
const (
	AgentTaskStatusPending     = "pending"     // 未触发（所属 invocation 还没调 RunCommand，或测试机失败导致后续不发出）
	AgentTaskStatusInProgress  = "in_progress" // TAT 侧正在执行
	AgentTaskStatusSuccess     = "success"     // TAT 终态 + exit_code == 0
	AgentTaskStatusFailed      = "failed"      // TAT 终态 + exit_code != 0
	AgentTaskStatusTimeout     = "timeout"     // TAT 报告超时
	AgentTaskStatusUnreachable = "unreachable" // TAT 报告 Agent 离线 / 不可达
	AgentTaskStatusCancelled   = "cancelled"   // 用户在测试机阶段后选择「终止下发」，本 task 在未触发 RunCommand 状态下被取消
)

// IsTerminalAgentTaskStatus 判断 task.status 是否到达终态。
func IsTerminalAgentTaskStatus(s string) bool {
	switch s {
	case AgentTaskStatusSuccess, AgentTaskStatusFailed,
		AgentTaskStatusTimeout, AgentTaskStatusUnreachable,
		AgentTaskStatusCancelled:
		return true
	}
	return false
}

// IsFailureAgentTaskStatus 是否计入 failed_count 的状态。
//
// 注意：cancelled 不在此列。cancelled 是用户主动放弃下发的语义，与 TAT 报告的失败（failed/timeout/unreachable）
// 在 dispatch 侧应当区分展示，避免把"我自己取消的"误读成"执行失败"。
func IsFailureAgentTaskStatus(s string) bool {
	switch s {
	case AgentTaskStatusFailed, AgentTaskStatusTimeout, AgentTaskStatusUnreachable:
		return true
	}
	return false
}

// ============================================================================
// 模型
// ============================================================================

// AgentCommandTask 一次 TAT RunCommand 内、每台 instance 的执行任务。对应 TAT InvocationTask。
//
// ⚠️ 决策 Q10：本表**不含 stdout / stderr 字段**；详情接口实时调用 TAT
// DescribeInvocationTasks 获取，避免库膨胀与跨地域复制成本。
//
// 不软删：保留为永久审计记录，跟随 TAT invocation 自然过期。
//
// v2 数据模型：新增 DispatchID FK；DispatchSlug 保留为冗余便于按 slug 反查。
type AgentCommandTask struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Identifier          string     `gorm:"index;default:''" json:"-"`
	TATInvocationTaskID string     `gorm:"column:tat_invocation_task_id;index;type:varchar(64);not null;default:''" json:"tat_invocation_task_id"`
	DispatchID          uint       `gorm:"not null;default:0;index" json:"dispatch_id"`
	InvocationID        uint       `gorm:"not null;default:0;index" json:"invocation_id"`                   // fk → agent_command_invocations.id
	DispatchSlug        string     `gorm:"index;type:varchar(32);not null;default:''" json:"dispatch_slug"` // 冗余便于聚合查询
	InstanceID          uint       `gorm:"not null;default:0;index" json:"instance_id"`
	CVMInstanceID       string     `gorm:"column:cvm_instance_id;type:varchar(64);not null;default:''" json:"cvm_instance_id"`
	AgentName           string     `gorm:"type:varchar(255);not null;default:''" json:"agent_name"`
	OwnerUsername       string     `gorm:"type:varchar(191);not null;default:''" json:"owner_username"`
	IsTestTarget        bool       `gorm:"not null;default:false" json:"is_test_target"`
	Status              string     `gorm:"type:varchar(16);not null;default:'pending';index" json:"status"`
	ExitCode            *int       `gorm:"default:null" json:"exit_code,omitempty"`
	ElapsedMs           *uint      `gorm:"default:null" json:"elapsed_ms,omitempty"`
	StartedAt           *time.Time `gorm:"default:null" json:"started_at,omitempty"`
	FinishedAt          *time.Time `gorm:"default:null" json:"finished_at,omitempty"`
}

func (AgentCommandTask) TableName() string { return "agent_command_tasks" }

// ============================================================================
// CRUD helper
// ============================================================================

// FindTasksByInvocationID 列出某 invocation 下所有 task。
func FindTasksByInvocationID(ctx context.Context, invocationID uint) ([]AgentCommandTask, error) {
	var rows []AgentCommandTask
	err := DB(ctx).
		Where("invocation_id = ?", invocationID).
		Order("is_test_target desc, id asc").
		Find(&rows).Error
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgTaskFindByInvocationIDFailed)
	}
	return rows, nil
}

// FindTasksByDispatchSlug 列出某 dispatch 下所有 task。
func FindTasksByDispatchSlug(ctx context.Context, slug string) ([]AgentCommandTask, error) {
	var rows []AgentCommandTask
	err := DB(ctx).
		Where("dispatch_slug = ?", slug).
		Order("is_test_target desc, invocation_id asc, id asc").
		Find(&rows).Error
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgTaskFindBySlugFailed)
	}
	return rows, nil
}

// FindUnfinishedTaskTATIDs 列出当前租户内所有未终态 task 的 tat_invocation_task_id（去除空值）。
// 用于全局轮询协程批量查询 TAT 刷新状态。
//
// 仅按当前 ctx 注入的 identifier 过滤；如需跨租户使用 DBGlobal 调用。
func FindUnfinishedTaskTATIDs(ctx context.Context, limit int) ([]AgentCommandTask, error) {
	var rows []AgentCommandTask
	q := DB(ctx).Model(&AgentCommandTask{}).
		Where("status IN ? AND tat_invocation_task_id <> ?",
			[]string{AgentTaskStatusInProgress}, "").
		Order("id asc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgTaskFindUnfinishedFailed)
	}
	return rows, nil
}
