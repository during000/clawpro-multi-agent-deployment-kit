package model

import (
	"time"

	"gorm.io/gorm"
)

// LocalAgentTask 通用本地实例任务表。
//
// 承载「非 skill/rule 下发」的本地任务（如移除本地 Agent 卸载 clawpro-teamai 插件）。
// 与 rule_distribution_tasks 不同：本表是「单实例 + 单任务」粒度，
// 没有 task→records 一对多的展开，status 直接沿用 rule records 的语义子集。
//
// 当前类型：
//   - uninstall_teamai：卸载 clawpro-teamai 插件 + 解绑
//   - execute_agent_task：在本地工作区调用指定 Agent 执行任务
type LocalAgentTask struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	Identifier    string         `gorm:"index:idx_lat_identifier;default:''" json:"-"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index:idx_lat_deleted_at" json:"-"`
	InstanceID    uint           `gorm:"index:idx_lat_instance;not null" json:"instance_id"` // 目标本地实例（instances.id）
	InstanceCID   string         `gorm:"column:instance_c_id;not null;default:''" json:"instance_c_id"`
	Type          string         `gorm:"index:idx_lat_type;not null;default:''" json:"type"`
	Cmd           string         `gorm:"type:text" json:"cmd"`                     // 创建任务时生成落表
	Status        string         `gorm:"not null;default:'pending'" json:"status"` // pending / running / success / failed / cancelled
	Error         string         `gorm:"type:text" json:"error"`
	OperatorID    uint           `gorm:"not null;default:0" json:"operator_id"` // 发起人（用户或管理员）
	ProjectID     uint           `gorm:"index:idx_lat_project;not null;default:0" json:"project_id,omitempty"`
	WorkspacePath string         `gorm:"size:512;not null;default:''" json:"workspace_path,omitempty"`
	AgentType     string         `gorm:"size:32;not null;default:''" json:"agent_type,omitempty"`
	Prompt        string         `gorm:"type:text" json:"prompt,omitempty"`
	Result        string         `gorm:"type:longtext" json:"result,omitempty"`
	SessionID     string         `gorm:"size:191;not null;default:''" json:"session_id,omitempty"`
	StartedAt     *time.Time     `json:"started_at,omitempty"`
	FinishedAt    *time.Time     `json:"finished_at,omitempty"`
}

// TableName 固定表名。
func (LocalAgentTask) TableName() string { return "local_agent_tasks" }

// 本地任务类型常量（Type 字段取值）。
const (
	LocalAgentTaskTypeUninstallTeamai = "uninstall_teamai"
	LocalAgentTaskTypeExecuteAgent    = "execute_agent_task"
)

// 本地任务状态常量（Status 字段取值，对齐 rule records 语义子集）。
const (
	LocalAgentTaskStatusPending   = "pending"   // 待执行（reporter 下次 sync 拉取）
	LocalAgentTaskStatusRunning   = "running"   // 本地 Agent 已领取并开始执行
	LocalAgentTaskStatusSuccess   = "success"   // 执行成功
	LocalAgentTaskStatusFailed    = "failed"    // 执行失败（保留可重试）
	LocalAgentTaskStatusCancelled = "cancelled" // 已取消（重复提交旧任务等场景）
)
