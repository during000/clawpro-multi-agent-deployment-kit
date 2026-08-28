package model

import (
	"time"

	"gorm.io/gorm"
)

// DoctorSession 龙虾医生诊断会话
type DoctorSession struct {
	gorm.Model
	Identifier        string     `gorm:"index;default:''"` // 多租户标识
	UserID            uint       `gorm:"not null;index"`
	TargetInstanceID  uint       `gorm:"not null;index"`
	DoctorInstanceID  *uint      `gorm:"default:null"`
	Status            string     `gorm:"type:varchar(16);not null;default:'creating';index"`
	ActivatedAt       *time.Time `gorm:"default:null"`           // 会话进入 active 状态的时间
	SnapshotRequested bool       `gorm:"not null;default:false"` // 是否请求在激活后自动创建快照
	HasSnapshot       bool       `gorm:"not null;default:false"`
	SnapshotFileKey   string     `gorm:"type:varchar(512);default:''"`
	SnapshotDeleted   bool       `gorm:"not null;default:false"` // 快照是否已从 SMH 删除
	SessionsDeleted   bool       `gorm:"not null;default:false"` // session 对话备份是否已从 SMH 删除
	RollbackRequested bool       `gorm:"not null;default:false"` // 结束时是否请求回滚
	STSExpiredAt      int64      `gorm:"not null;default:0"`     // STS 临时密钥过期时间(Unix 秒)
}

// DoctorSession status constants
const (
	DoctorStatusCreating = "creating"
	DoctorStatusActive   = "active"
	DoctorStatusEnding   = "ending"
	DoctorStatusEnded    = "ended"
	DoctorStatusFailed   = "failed"
)
