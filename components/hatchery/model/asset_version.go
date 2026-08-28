package model

import (
	"time"
)

// AssetVersionRecord 记录项目/分组的资产版本变更历史。
// 由本模块（资产管理版本记录子任务）写入，不承载下发状态（下发状态由现有的 task 表负责）。
//
// 触发来源两类：
//   - 手动保存：trigger_type=manual, trigger_reason=manual_save（RecordAssetSave 写入）
//   - 工具库自动变更：trigger_type=system，trigger_reason 为 asset_version_published / asset_deleted / asset_scope_changed（PublishAssetVersion 写入）
type AssetVersionRecord struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"record_id"`
	TargetType    string    `gorm:"size:32;not null;index:idx_av_target" json:"target_type"` // group | project
	TargetID      uint      `gorm:"not null;index:idx_av_target" json:"target_id"`
	Version       int       `gorm:"not null" json:"version"` // 目标内自增序列，首建为 1
	TriggerType   string    `gorm:"size:32; not null" json:"trigger_type"` // manual | system
	TriggerReason string    `gorm:"size:32; not null" json:"trigger_reason"` // manual_save|asset_version_published|asset_deleted|asset_scope_changed
	OperatorType  string    `gorm:"size:16; not null;default:'user'" json:"operator_type"` // admin | system
	OperatorID    uint      `gorm:"not null;default:0" json:"operator_id"`      // 操作人 ID（system 为 0）
	OperatorName  string    `gorm:"size:191; not null;default:''" json:"operator_name"` // 操作人姓名（system 为空）
	ChangesJSON   string    `gorm:"type:text; not null" json:"-"` // AssetChanges 序列化，落库不进响应（source of truth + 审计）
	CreatedAt     time.Time `json:"created_at"`
}

// TableName 显式指定表名（避免 GORM 复数推断差异）。
func (AssetVersionRecord) TableName() string { return "asset_versions" }

// AssetChanges 版本变更的差异化明细，序列化为 AssetVersionRecord.ChangesJSON。
// 所有字段均为展示层服务，后端只返回结构化信息，文案由前端拼接。
type AssetChanges struct {
	Added   []AssetChangeItem `json:"added"`   // 新增资产
	Removed []AssetChangeItem `json:"removed"` // 移除资产（不会卸载）
	Updated []AssetChangeItem `json:"updated"` // 版本变化资产
	SyncMode string           `json:"sync_mode"` // 本次同步模式新值；未变更为空字符串
}

// AssetChangeItem 单项变更（added / removed / updated 共用）。
type AssetChangeItem struct {
	AssetType  string `json:"asset_type"`            // skill | rule
	Slug       string `json:"slug"`                 // 资产稳定 slug
	Name       string `json:"name"`                 // 资产名称
	FromVersion string `json:"from_version,omitempty"` // updated 专用：原版本；不适用为空串
	ToVersion  string `json:"to_version,omitempty"`   // updated 专用：新版本；不适用为空串
}

// AssetVersion 触发类型常量。
const (
	TriggerTypeManual = "manual"
	TriggerTypeSystem = "system"
)

// AssetVersion 触发原因常量。
const (
	TriggerReasonManualSave        = "manual_save"
	TriggerReasonAssetPublished    = "asset_version_published"
	TriggerReasonAssetDeleted      = "asset_deleted"
	TriggerReasonScopeChanged      = "asset_scope_changed"
)

// AssetVersion 同步模式常量（与 save 接口 sync_mode 取值一致）。
const (
	SyncModeInitialOnly = "initial_only"
	SyncModeContinuous  = "continuous"
)

// AssetVersion 目标类型常量。
const (
	TargetTypeGroup   = "group"
	TargetTypeProject = "project"
)
