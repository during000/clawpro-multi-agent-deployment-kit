package model

import (
	"time"
)

// LocalSkillSource 是 LocalInstanceSkill.Source 的取值集合。
const (
	LocalSkillSourcePublic     = "public"     // 公共技能库下发
	LocalSkillSourceEnterprise = "enterprise" // 企业技能库下发
	LocalSkillSourceLocal      = "local"      // 用户在本地手动安装（reporter 上报、hatchery 从未下发过）
)

// LocalSkillScope 是 LocalInstanceSkill.Scope 的取值集合。
// 二期字段仅区分 user（用户级）/ workspace（项目级）。
const (
	LocalSkillScopeUser      = "user"      // 用户级：跟随用户账号
	LocalSkillScopeWorkspace = "workspace" // 项目级：绑定 workspace_path
)

// LocalSkillInstallStatus 是 LocalInstanceSkill.InstallStatus 的取值集合。
// 二期新增字段，前端直接读，不派生（PRD 要求）。
const (
	LocalSkillInstallStatusDistributing = "distributing" // 下发中（pending record 已写，agent 未 ack）
	LocalSkillInstallStatusDistributed  = "distributed"  // 已下发（agent ack 成功 / report 上报已装）
	LocalSkillInstallStatusFailed       = "failed"       // 下发失败（agent ack 失败）
)

// LocalInstanceSkill 本地 agent 实例当前已安装 skill 的事实快照。
//
// 重要约定（与 iwiki 4022150701 对齐）：本表保存「成功装着」的事实；
// 二期加 install_status 字段后承担状态机职责（前端直接读，不派生）。
// 任务流水（pending/failed）与装/卸全过程由 skill_distribution_records 表承担，CVM 与 local 共用同一张状态机表。
//
//   - ack=success（type=distribute）→ 本表 INSERT...ON DUPLICATE KEY UPDATE 一行
//   - ack=success（type=uninstall） → 本表对应行 DELETE（硬删）
//   - report 上报已装 skill        → 本表 per-skill upsert + source='local' 范围消失即删（硬删）
//   - ack=failed                   → 本表不变（二期：install_status='failed'）
//
// 唯一索引 (scope, instance_id, workspace_path, slug) 保证一行 = 一个 (作用域, 实例, workspace, slug) 装着的事实。
// **不使用 soft delete**（不嵌 gorm.Model、无 deleted_at 列）：避免 （1） ON DUPLICATE
// 命中软删行变"装了但查不到"的认知错位；（2）与表「只记当前事实」的语义冲突。
//
// 迁移脚本: sql/0624-local-agent.sql + sql/0706-local-agent-resources.sql | 初始化脚本: sql/init.sql
type LocalInstanceSkill struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Identifier  string    `gorm:"index;default:''" json:"-"` // 多租户标识，MySQL 模式下自动填充和过滤
	InstanceID  uint      `gorm:"index:idx_lis_scope_inst_ws_slug,unique,priority:1;not null" json:"instance_id"`
	Slug        string    `gorm:"index:idx_lis_scope_inst_ws_slug,unique,priority:4;type:varchar(64);not null" json:"slug"`
	Version     string    `gorm:"type:varchar(32);default:''" json:"version"`
	DisplayName string    `gorm:"type:varchar(128);default:''" json:"display_name"`
	// 来源：public / enterprise / local
	Source string `gorm:"type:varchar(16);default:'local';index" json:"source"`
	// 最近一次 ack success / report 上报已装的时刻
	InstalledAt *time.Time `gorm:"default:null" json:"installed_at"`
	// report 中最后一次出现的时刻；用于 source='local' 范围内的"消失即删"
	LastSeenAt *time.Time `gorm:"default:null;index" json:"last_seen_at"`

	// ─── 二期新增 ───
	// 作用域：user（用户级）/ workspace（项目级）
	Scope string `gorm:"type:varchar(16);not null;default:'user';index:idx_lis_scope_inst_ws_slug,unique,priority:2" json:"scope"`
	// 项目级独有：workspace 路径（scope='workspace' 时存；其他为空）
	WorkspacePath string `gorm:"type:varchar(512);not null;default:'';index:idx_lis_scope_inst_ws_slug,unique,priority:3" json:"workspace_path"`
	// 安装状态：distributing / distributed / failed（前端直接读，不派生）
	InstallStatus string `gorm:"type:varchar(16);not null;default:'distributed'" json:"install_status"`
}

// LocalInstanceSkillTable 表名常量。
const LocalInstanceSkillTable = "local_instance_skills"
