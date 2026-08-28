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

// AgentCommandInvocationStatus 一次 TAT RunCommand 调用的状态。
const (
	AgentInvocationStatusPending    = "pending"     // 已预创建，但还没调 RunCommand（如测试机失败导致后续不发出）
	AgentInvocationStatusInProgress = "in_progress" // 已拿到 tat_invocation_id，部分或全部 task 仍在执行
	AgentInvocationStatusSuccess    = "success"     // 本 invocation 内所有 task exit_code == 0
	AgentInvocationStatusPartial    = "partial"     // 本 invocation 内有成功也有失败
	AgentInvocationStatusFailed     = "failed"      // 本 invocation 内所有 task 都失败
	AgentInvocationStatusCancelled  = "cancelled"   // 用户在测试机阶段后选择「终止下发」，本 invocation 被取消，未触发 RunCommand
)

// ============================================================================
// 模型
// ============================================================================

// AgentCommandInvocation 表示一次 TAT RunCommand 调用。
//
// 三层语义：
//
//	dispatch（dispatch_slug=task-...）              ← AgentCommandDispatch
//	  ↓ 1 : 1~2
//	invocation（一次 RunCommand，inv-...）           ← 本表
//	  ↓ 1 : N
//	task（每台 instance，invt-...）                   ← AgentCommandTask
//
// 字段重组（v2 数据模型）：
//   - 去掉 CommandID / CommandSnapshot / ParamValuesJSON / TriggeredByUserID
//     —— 这些字段语义上属于 dispatch，已上提到 AgentCommandDispatch
//   - 新增 DispatchID FK；DispatchSlug 保留为冗余便于按 slug 反查
//
// 不软删：保留为永久审计记录。
type AgentCommandInvocation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Identifier      string     `gorm:"index;default:''" json:"-"`
	TATInvocationID string     `gorm:"column:tat_invocation_id;index;type:varchar(64);not null;default:''" json:"tat_invocation_id"`
	DispatchID      uint       `gorm:"not null;default:0;index" json:"dispatch_id"`
	DispatchSlug    string     `gorm:"index;type:varchar(32);not null;default:''" json:"dispatch_slug"`
	IsTestRun       bool       `gorm:"not null;default:false" json:"is_test_run"`
	BatchIndex      uint       `gorm:"not null;default:0" json:"batch_index"`
	TargetCount     uint       `gorm:"not null;default:0" json:"target_count"`
	SuccessCount    uint       `gorm:"not null;default:0" json:"success_count"`
	FailedCount     uint       `gorm:"not null;default:0" json:"failed_count"`
	Status          string     `gorm:"type:varchar(16);not null;default:'pending';index" json:"status"`
	StartedAt       *time.Time `gorm:"default:null" json:"started_at,omitempty"`
	FinishedAt      *time.Time `gorm:"default:null" json:"finished_at,omitempty"`
}

func (AgentCommandInvocation) TableName() string { return "agent_command_invocations" }

// IsTerminal 判断 invocation 状态是否到达终态（success / partial / failed / cancelled）。
func (inv *AgentCommandInvocation) IsTerminal() bool {
	switch inv.Status {
	case AgentInvocationStatusSuccess, AgentInvocationStatusPartial,
		AgentInvocationStatusFailed, AgentInvocationStatusCancelled:
		return true
	}
	return false
}

// ============================================================================
// CRUD helper
// ============================================================================

// FindInvocationsByDispatchSlug 列出某 dispatch 下的所有 invocation，按 (is_test_run desc, batch_index asc) 排序。
func FindInvocationsByDispatchSlug(ctx context.Context, slug string) ([]AgentCommandInvocation, error) {
	var rows []AgentCommandInvocation
	err := DB(ctx).
		Where("dispatch_slug = ?", slug).
		Order("is_test_run desc, batch_index asc").
		Find(&rows).Error
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgInvocationFindBySlugFailed)
	}
	return rows, nil
}

// FindInvocationsByDispatchID 列出某 dispatch 下的所有 invocation。
//
// 与 FindInvocationsByDispatchSlug 等价，按 dispatch_id（FK）查询，
// 适用于已经拿到 dispatch 行的场景，省掉 slug 字符串 hash 查找。
func FindInvocationsByDispatchID(ctx context.Context, dispatchID uint) ([]AgentCommandInvocation, error) {
	var rows []AgentCommandInvocation
	err := DB(ctx).
		Where("dispatch_id = ?", dispatchID).
		Order("is_test_run desc, batch_index asc").
		Find(&rows).Error
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgInvocationFindByIDFailed)
	}
	return rows, nil
}
