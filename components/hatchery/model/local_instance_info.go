package model

import (
	"time"

	"gorm.io/gorm"
)

// LocalInstanceInfo 本地 agent 实例的扩展信息（与 Instance 1:1）。
//
// 当 instances.source='local' 时，对应一行 LocalInstanceInfo，存放 reporter
// 上报的本机基本信息（host_name / os）以及最近一次 report 的时间戳。
//
// 设计：把这些「仅 local 实例需要」的字段独立成表，避免污染主 instances 表的 schema。
//
// 迁移脚本: sql/0624-local-agent.sql | 初始化脚本: sql/init.sql
type LocalInstanceInfo struct {
	gorm.Model
	Identifier string `gorm:"index;default:''" json:"-"` // 多租户标识，MySQL 模式下自动填充和过滤
	InstanceID uint   `gorm:"uniqueIndex;not null" json:"instance_id"`
	HostName   string `gorm:"type:varchar(128);default:''" json:"host_name"`
	OS         string `gorm:"type:varchar(64);default:''" json:"os"`
	// reporter 上报时附带的进程启动时间（首次上报后基本不变）
	StartedAt *time.Time `gorm:"default:null" json:"started_at"`
	// 最近一次 report / sync 上报时间，用于派生 status=running/stopped
	LastReportAt *time.Time `gorm:"default:null;index" json:"last_report_at"`
	// 最近一次 report 的运行状态文案（reporter 端 Hook 派生，如 "running"/"error"）
	LastStatus string `gorm:"type:varchar(32);default:''" json:"last_status"`
}

// LocalInstanceInfoTable 表名常量，便于 JOIN 引用。
const LocalInstanceInfoTable = "local_instance_infos"

// LocalInstanceOfflineThreshold 本地实例 running → stopped 的阈值。
// 超过该时长仍未收到 report，视为 stopped（三期从 24h 调整为 7 天，对应需求「7 天不活跃」）。
// 状态枚举不变（无第三态），前端按 stopped 自行展示「不活跃」差异。
const LocalInstanceOfflineThreshold = 7 * 24 * time.Hour
