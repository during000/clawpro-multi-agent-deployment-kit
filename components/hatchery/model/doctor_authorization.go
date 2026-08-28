package model

import "gorm.io/gorm"

// DoctorAuthorization 记录用户对实例的龙虾医生授权。
// 每个用户+实例组合最多一条记录，存在即表示已授权。
type DoctorAuthorization struct {
	gorm.Model
	Identifier string `gorm:"uniqueIndex:idx_auth_user_instance;index;default:''" json:"-"` // 多租户标识
	UserID     uint   `gorm:"not null;uniqueIndex:idx_auth_user_instance"`
	InstanceID uint   `gorm:"not null;uniqueIndex:idx_auth_user_instance"`
}
