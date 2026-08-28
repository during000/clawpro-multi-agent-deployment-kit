package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// InstanceFlag 实例轻量布尔标记表（stale-instances v1.0）。
// 一个实例可有多条标记；按 flag 反查需要索引；MySQL TEXT 不能加 DEFAULT，故 extra 用 varchar。
type InstanceFlag struct {
	ID         uint   `gorm:"primaryKey"`
	Identifier string `gorm:"type:varchar(191);not null;default:'';uniqueIndex:uk_instance_flags_inst_flag,priority:1;index:idx_instance_flags_flag_lookup,priority:1"`
	InstanceID uint   `gorm:"not null;uniqueIndex:uk_instance_flags_inst_flag,priority:2;index:idx_instance_flags_flag_lookup,priority:3"`
	// 标记名（stale_group / pending_user_action / allow_migrate / allow_same_group_handover）
	Flag string `gorm:"type:varchar(64);not null;uniqueIndex:uk_instance_flags_inst_flag,priority:3;index:idx_instance_flags_flag_lookup,priority:2"`
	// JSON 附加（如 trigger_source）；MySQL TEXT 不能 DEFAULT，故 varchar
	Extra     string    `gorm:"type:varchar(1024);not null;default:'{}'"`
	CreatedAt time.Time `gorm:"not null"`
}

// 标记常量
const (
	InstanceFlagStaleGroup             = "stale_group"
	InstanceFlagPendingUserAction      = "pending_user_action"
	InstanceFlagAllowMigrate           = "allow_migrate"
	InstanceFlagAllowSameGroupHandover = "allow_same_group_handover"
)

// AddInstanceFlag 写入或更新实例标记（按 (identifier, instance_id, flag) 唯一）。
func AddInstanceFlag(ctx context.Context, instanceID uint, flag, extra string) error {
	if extra == "" {
		extra = "{}"
	}
	f := InstanceFlag{InstanceID: instanceID, Flag: flag, Extra: extra}
	// ON DUPLICATE KEY UPDATE 更新 extra + created_at（重新打标视作新事件）
	return DB(ctx).Where("instance_id = ? AND flag = ?", instanceID, flag).
		Assign(map[string]any{"extra": extra, "created_at": time.Now()}).
		FirstOrCreate(&f).Error
}

// RemoveInstanceFlag 删除实例的某个标记（不存在则 noop）。
func RemoveInstanceFlag(ctx context.Context, instanceID uint, flag string) error {
	return DB(ctx).Where("instance_id = ? AND flag = ?", instanceID, flag).
		Delete(&InstanceFlag{}).Error
}

// GetInstanceFlags 列出某实例的所有标记。
func GetInstanceFlags(ctx context.Context, instanceID uint) ([]InstanceFlag, error) {
	var flags []InstanceFlag
	err := DB(ctx).Where("instance_id = ?", instanceID).Find(&flags).Error
	return flags, err
}

// GetInstanceFlagsBatch 批量查询多个实例的标记，返回 instance_id → []flag 名映射。
func GetInstanceFlagsBatch(ctx context.Context, instanceIDs []uint) (map[uint][]string, error) {
	result := make(map[uint][]string)
	if len(instanceIDs) == 0 {
		return result, nil
	}
	var flags []InstanceFlag
	if err := DB(ctx).Where("instance_id IN ?", instanceIDs).Find(&flags).Error; err != nil {
		return nil, err
	}
	for _, f := range flags {
		result[f.InstanceID] = append(result[f.InstanceID], f.Flag)
	}
	return result, nil
}

// HasInstanceFlag 判断实例是否带某标记。
func HasInstanceFlag(ctx context.Context, instanceID uint, flag string) (bool, error) {
	var count int64
	err := DB(ctx).Model(&InstanceFlag{}).
		Where("instance_id = ? AND flag = ?", instanceID, flag).
		Count(&count).Error
	return count > 0, err
}

// ClearAllInstanceFlags 清除某实例的所有标记（apply 完成后调用，保留处理记录但清掉标）。
func ClearAllInstanceFlags(tx *gorm.DB, instanceID uint) error {
	return tx.Where("instance_id = ?", instanceID).Delete(&InstanceFlag{}).Error
}
