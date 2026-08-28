package model

import (
	"context"
	"errors"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"time"

	"gorm.io/gorm"
)

// ============================================================================
// AgentCommandDispatch
//
// 一次"用户视角"的命令下发（dispatch）= 1 行；包含 1~2 条 invocation：
//
//	dispatch（dispatch_slug=task-xxxxxxxx）  ← 本表
//	  ↓ 1 : 1~2
//	invocation（一次 RunCommand）
//	  ↓ 1 : N
//	task（每台 agent 的执行任务）
//
// 字段从原 invocation 上"上提"：command_id / command_snapshot /
// param_values_json / triggered_by_user_id 现在统一存在 dispatch 行上。
//
// status 是显式持久字段（替代旧的"读时聚合 + awaiting_confirmation 虚拟态"）。
// 状态推进路径：写路径事件式 + 后台 reconcile 兜底。
// ============================================================================

// AgentCommandDispatchStatus dispatch 整体状态枚举。
const (
	AgentDispatchStatusInProgress           = "in_progress"
	AgentDispatchStatusAwaitingConfirmation = "awaiting_confirmation"
	AgentDispatchStatusSuccess              = "success"
	AgentDispatchStatusPartial              = "partial"
	AgentDispatchStatusFailed               = "failed"
	AgentDispatchStatusCancelled            = "cancelled"
)

// AgentCommandDispatchSlugPrefix dispatch_slug 前缀。
const AgentCommandDispatchSlugPrefix = "task-"

// AgentCommandDispatchSlugRandLen dispatch_slug 随机部分长度。
const AgentCommandDispatchSlugRandLen = 8

// AgentDispatchMaxTargets 单次 dispatch 最多目标 Agent 数。
//
// 与 TAT RunCommand 单次调用上限（200，详见
// https://cloud.tencent.com/document/api/1340/52676 ）对齐，即一次 dispatch
// 在生产阶段只触发 1 次 RunCommand。
const AgentDispatchMaxTargets = 200

// AgentCommandDispatch 一次下发的顶层实体。
//
// 不软删：保留为永久审计记录。user 软删后 triggered_by_user_id 仍指向原 ID
// （查不到 user 时前端展示"已删除用户 (#42)"）。
//
// 索引说明：
//   - (identifier, slug) UNIQUE — 同租户内 slug 全局唯一
//   - (identifier, status) — list / reconcile 查未终态
//   - (identifier, command_id) — 命令删除前置校验、模板执行历史
//   - (identifier, triggered_by_user_id) — 按发起人过滤
//   - (identifier, created_at desc) — list 默认排序
type AgentCommandDispatch struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Identifier string `gorm:"uniqueIndex:idx_dispatch_ident_slug,priority:1;index;default:''" json:"-"`
	Slug       string `gorm:"uniqueIndex:idx_dispatch_ident_slug,priority:2;type:varchar(32);not null;default:''" json:"slug"`

	CommandID         uint   `gorm:"not null;default:0;index" json:"command_id"`
	CommandSnapshot   string `gorm:"type:text" json:"-"`
	ParamValuesJSON   string `gorm:"type:varchar(4096);not null;default:'{}'" json:"-"`
	TriggeredByUserID uint   `gorm:"not null;default:0;index" json:"triggered_by_user_id"`

	TestFirst            bool `gorm:"not null;default:false" json:"test_first"`
	TestTargetInstanceID uint `gorm:"not null;default:0" json:"test_target_instance_id"`

	// target_count 含测试机（与 dispatch 全部 task 数对齐）
	TargetCount    uint `gorm:"not null;default:0" json:"target_count"`
	SuccessCount   uint `gorm:"not null;default:0" json:"success_count"`
	FailedCount    uint `gorm:"not null;default:0" json:"failed_count"`
	CancelledCount uint `gorm:"not null;default:0" json:"cancelled_count"`

	Status     string     `gorm:"type:varchar(24);not null;default:'in_progress';index" json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `gorm:"default:null" json:"finished_at,omitempty"`
}

// TableName 显式表名（与 sql/init.sql 对齐）。
func (AgentCommandDispatch) TableName() string { return "agent_command_dispatch" }

// IsTerminal 判断 dispatch 是否到达终态。
//
// awaiting_confirmation 不算终态：用户尚未决定，dispatch 还活跃。
func (d *AgentCommandDispatch) IsTerminal() bool {
	switch d.Status {
	case AgentDispatchStatusSuccess, AgentDispatchStatusPartial,
		AgentDispatchStatusFailed, AgentDispatchStatusCancelled:
		return true
	}
	return false
}

// GenerateAgentDispatchSlug 生成 dispatch_slug，格式 task-{8 位随机}。
func GenerateAgentDispatchSlug() string {
	return AgentCommandDispatchSlugPrefix + randomLowerAlnum(AgentCommandDispatchSlugRandLen)
}

// ============================================================================
// 错误
// ============================================================================

var (
	ErrDispatchNotFound     = hcommon.I18nError(i18n.MsgDispatchRecordNotFound)
	ErrDispatchSlugConflict = hcommon.I18nError(i18n.MsgDispatchSlugConflictRetry)
)

// ============================================================================
// CRUD helper
// ============================================================================

// FindDispatchBySlug 按 slug 查 dispatch。
func FindDispatchBySlug(ctx context.Context, slug string) (*AgentCommandDispatch, error) {
	var d AgentCommandDispatch
	err := DB(ctx).Where("slug = ?", slug).First(&d).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDispatchNotFound
		}
		return nil, hcommon.I18nRichError(err, i18n.MsgDispatchFindBySlugFailed)
	}
	return &d, nil
}

// FindDispatchByID 按主键查 dispatch。
func FindDispatchByID(ctx context.Context, id uint) (*AgentCommandDispatch, error) {
	var d AgentCommandDispatch
	err := DB(ctx).Where("id = ?", id).First(&d).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDispatchNotFound
		}
		return nil, hcommon.I18nRichError(err, i18n.MsgDispatchFindByIDFailed)
	}
	return &d, nil
}

// HasInProgressDispatches 判断某 commandID 是否有未到终态的 dispatch。
//
// 用于命令删除前置校验：进行中 / awaiting_confirmation 都视为活跃。
// 返回 (是否存在, 活跃的 dispatch_slug 列表, error)。
func HasInProgressDispatches(ctx context.Context, commandID uint) (bool, []string, error) {
	var rows []AgentCommandDispatch
	err := DB(ctx).
		Select("id, slug, status").
		Where("command_id = ? AND status IN ?", commandID,
			[]string{AgentDispatchStatusInProgress, AgentDispatchStatusAwaitingConfirmation}).
		Find(&rows).Error
	if err != nil {
		return false, nil, hcommon.I18nRichError(err, i18n.MsgDispatchQueryInProgressFailed)
	}
	if len(rows) == 0 {
		return false, nil, nil
	}
	slugs := make([]string, 0, len(rows))
	for _, r := range rows {
		slugs = append(slugs, r.Slug)
	}
	return true, slugs, nil
}

// CommandExecutionStat 命令模板执行统计：最近一次执行时间 + 累计 dispatch 次数。
type CommandExecutionStat struct {
	LastExecutedAt *time.Time
	ExecutedCount  uint
}

// BatchCommandExecutionStats 批量查询多个 command_id 的执行统计。
//
// 新模型下 dispatch 行 = 一次执行，行数即 ExecutedCount，无需再 dedup dispatch_slug。
func BatchCommandExecutionStats(ctx context.Context, commandIDs []uint) (map[uint]CommandExecutionStat, error) {
	if len(commandIDs) == 0 {
		return map[uint]CommandExecutionStat{}, nil
	}
	type row struct {
		CommandID uint
		StartedAt time.Time
		CreatedAt time.Time
	}
	var rows []row
	err := DB(ctx).Model(&AgentCommandDispatch{}).
		Select("command_id, started_at, created_at").
		Where("command_id IN ?", commandIDs).
		Find(&rows).Error
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgDispatchBatchStatsFailed)
	}
	type acc struct {
		count uint
		last  *time.Time
	}
	tmp := make(map[uint]*acc, len(commandIDs))
	for _, r := range rows {
		a, ok := tmp[r.CommandID]
		if !ok {
			a = &acc{}
			tmp[r.CommandID] = a
		}
		a.count++
		t := r.StartedAt
		if t.IsZero() {
			t = r.CreatedAt
		}
		if a.last == nil || t.After(*a.last) {
			a.last = &t
		}
	}
	out := make(map[uint]CommandExecutionStat, len(tmp))
	for id, a := range tmp {
		out[id] = CommandExecutionStat{
			LastExecutedAt: a.last,
			ExecutedCount:  a.count,
		}
	}
	return out, nil
}

// FindUnfinishedDispatches 列出当前租户内未终态的 dispatch。
//
// 用于后台 reconcile 协程：扫 in_progress / awaiting_confirmation 的 dispatch
// 按 invocation/task 真实状态重新推算 dispatch.status，兜底事件式更新可能漏掉的边界。
func FindUnfinishedDispatches(ctx context.Context, limit int) ([]AgentCommandDispatch, error) {
	q := DB(ctx).
		Where("status IN ?", []string{
			AgentDispatchStatusInProgress,
			AgentDispatchStatusAwaitingConfirmation,
		}).
		Order("id asc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []AgentCommandDispatch
	if err := q.Find(&rows).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgDispatchFindUnfinishedFailed)
	}
	return rows, nil
}
