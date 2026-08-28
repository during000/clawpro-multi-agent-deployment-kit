package model

import (
	"time"
)

// LocalRuleSource 是 LocalInstanceRule.Source 的取值集合，与 skill 对齐。
const (
	LocalRuleSourceEnterprise = "enterprise" // 企业规范库下发
	LocalRuleSourceLocal      = "local"      // 用户在本地手动安装（reporter 上报、hatchery 从未下发过），占位保留
)

// LocalInstanceRule 本地 agent 实例当前已安装规范的事实快照。
//
// 语义完全对齐 LocalInstanceSkill（一期 iwiki 4022150701 §4.3）：
//   - ack=success（type=distribute）→ 本表 upsert 一行
//   - ack=success（type=uninstall） → 本表对应行 DELETE（硬删）
//   - report 上报已装规范           → 本表 per-slug upsert + 消失即删（硬删）
//   - ack=failed                    → 本表 install_status='failed'
//
// 二期新增 scope + workspace_path + install_status 字段（与 local_instance_skills 对齐），
// 支持用户级（scope='user'）和项目级（scope='workspace'）规范。
//
// 唯一索引 (scope, instance_id, workspace_path, slug) 保证一行 = 一个 (作用域, 实例, workspace, slug) 装着的事实。
// **不使用 soft delete**（不嵌 gorm.Model、无 deleted_at 列）。
//
// 迁移脚本：sql/0721-local-agent-phase2.sql；初始化脚本：sql/init.sql
type LocalInstanceRule struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Identifier  string    `gorm:"index;default:''" json:"-"`
	InstanceID  uint      `gorm:"index:idx_lir_scope_inst_ws_slug,unique,priority:1;not null" json:"instance_id"`
	Slug        string    `gorm:"index:idx_lir_scope_inst_ws_slug,unique,priority:4;type:varchar(64);not null" json:"slug"`
	Version     string    `gorm:"type:varchar(32);default:''" json:"version"`
	DisplayName string    `gorm:"type:varchar(128);default:''" json:"display_name"`
	// 类型：prompt / rule；从下发 task 携带过来的 rule_type，reporter ack 或 report 均写入
	RuleType string `gorm:"type:varchar(16);not null;default:'';index:idx_lir_type" json:"rule_type"`
	// 来源：enterprise / local；对齐 skill
	Source string `gorm:"type:varchar(16);default:'enterprise';index" json:"source"`
	// 最近一次 ack success / report 上报已装的时刻
	InstalledAt *time.Time `gorm:"default:null" json:"installed_at"`
	// report 中最后一次出现的时刻；用于 source='enterprise' 范围内的"消失即删"
	LastSeenAt *time.Time `gorm:"default:null;index" json:"last_seen_at"`

	// ─── 二期新增（与 LocalInstanceSkill 对齐）───
	// 作用域：user（用户级）/ workspace（项目级）
	Scope string `gorm:"type:varchar(16);not null;default:'user';index:idx_lir_scope_inst_ws_slug,unique,priority:2" json:"scope"`
	// 项目级独有：workspace 路径（scope='workspace' 时存；其他为空）
	WorkspacePath string `gorm:"type:varchar(512);not null;default:'';index:idx_lir_scope_inst_ws_slug,unique,priority:3" json:"workspace_path"`
	// 安装状态：distributing / distributed / failed（前端直接读，不派生）
	InstallStatus string `gorm:"type:varchar(16);not null;default:'distributed'" json:"install_status"`
}

// LocalInstanceRuleTable 表名常量。
const LocalInstanceRuleTable = "local_instance_rules"
