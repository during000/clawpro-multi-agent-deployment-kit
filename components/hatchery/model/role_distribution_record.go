package model

import (
	"time"

	"gorm.io/gorm"
)

// 角色下发状态枚举（对齐 skill/mcp 的 distribution 系列）
const (
	RoleRecordStatusUpdating  = "updating"  // 下发中：SOUL 或技能任一未完成
	RoleRecordStatusUpdated   = "updated"   // 已更新：SOUL + 全部技能都成功
	RoleRecordStatusFailed    = "failed"    // 更新失败：SOUL 或任一技能失败
	RoleRecordStatusCancelled = "cancelled" // 已取消：老记录被新的 apply 覆盖 / role_id 被清空
)

// SOUL 或技能子任务状态
const (
	RoleSubStatusPending = "pending" // 尚未开始（apply 刚建 record）
	RoleSubStatusRunning = "running" // 进行中（TAT 已发起）
	RoleSubStatusSuccess = "success" // 已成功
	RoleSubStatusFailed  = "failed"  // 已失败
)

// Instance.RoleSyncStatus 枚举
const (
	RoleSyncStatusEmpty    = ""         // 未初始化（role_id=0 或存量实例未经过状态机）
	RoleSyncStatusPending  = "pending"  // 待更新（distributed < role.version）
	RoleSyncStatusUpdating = "updating" // 更新中
	RoleSyncStatusUpdated  = "updated"  // 已更新
	RoleSyncStatusFailed   = "failed"   // 更新失败
)

// 触发下发的来源
const (
	RoleRecordSourceSwitch     = "switch"     // 用户端 /openclaw/switch-role
	RoleRecordSourceDistribute = "distribute" // 管理端 /admin/roles/distribute
	RoleRecordSourceCreate     = "create"     // 创建实例时初次绑定角色
)

// RoleDistributionRecord 每次角色 apply 一条记录，追踪 SOUL 与技能两个子任务。
// 对齐现有 SkillDistributionRecord / McpDistributionRecord 的字段与命名风格，
// 但语义是"per-instance apply"，无独立 Task 表。
type RoleDistributionRecord struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	Identifier string         `gorm:"index;default:''" json:"-"` // 多租户隔离
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	InstanceID  uint   `gorm:"column:instance_id;index;not null" json:"instance_id"`
	InstanceCID string `gorm:"column:instance_cid;type:varchar(64);not null;default:''" json:"instance_cid"` // CVM InstanceId 冗余快照
	RoleID      uint   `gorm:"index;not null" json:"role_id"`
	RoleName    string `gorm:"type:varchar(191);not null;default:''" json:"role_name"` // 角色名冗余快照
	Version     string `gorm:"type:varchar(16);not null;default:''" json:"version"`    // 目标版本号（apply 时角色的版本）
	OperatorID  uint   `gorm:"not null;default:0" json:"operator_id"`                  // 触发操作的用户 ID
	Source      string `gorm:"type:varchar(16);not null;default:''" json:"source"`     // switch / distribute / create

	// OperatorUsername 操作者用户名（联表 users.username 回填）。
	// ⚠️ 非持久化字段：gorm:"-" 表示不建表列、不读写数据库，SQL schema 中不存在此列。
	// 仅在 HandleAdminRoleRecords 查询响应中通过批量查 users 表填充，写入 record 时忽略。
	OperatorUsername string `gorm:"-" json:"operator_username"`

	Status string `gorm:"type:varchar(16);not null;default:'updating';index" json:"status"` // updating / updated / failed / cancelled

	SoulStatus string     `gorm:"type:varchar(16);not null;default:'pending'" json:"soul_status"` // pending / running / success / failed
	SoulError  string     `gorm:"type:text" json:"soul_error"`                                    // SOUL 下发失败原因
	SoulSetAt  *time.Time `gorm:"default:null" json:"soul_set_at"`                                // SOUL 成功下发时间

	SkillStatus string     `gorm:"type:varchar(16);not null;default:'pending'" json:"skill_status"` // pending / running / success / failed
	SkillError  string     `gorm:"type:text" json:"skill_error"`                                    // 技能安装失败原因（全部失败技能汇总）
	SkillSetAt  *time.Time `gorm:"default:null" json:"skill_set_at"`                                // 技能全部成功时间

	// 本次差集在 skill_installations 表创建的记录 ID 集合，JSON 数组格式如 "[1,2,3]"。
	// 用 varchar(2048) 而非 TEXT，因为 MySQL TEXT 不允许 DEFAULT 值。
	// 一个实例的差集通常几个到十几个技能，1KB 内足够。
	SkillInstallationIDs string `gorm:"type:varchar(2048);not null;default:''" json:"skill_installation_ids"`
}

// TableName 显式声明表名，避免 GORM 自动推导变化。
func (RoleDistributionRecord) TableName() string {
	return "role_distribution_records"
}
