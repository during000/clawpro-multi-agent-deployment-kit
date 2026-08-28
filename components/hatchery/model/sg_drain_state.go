package model

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DrainStuckThreshold 单实例连续换绑失败多少次后标记 drain_stuck。
// 达到后 DrainWorker 跳过该实例，由运维介入处理（可能是实例已删 / 跨账号权限 / 永久 InvalidParameter）。
const DrainStuckThreshold = 10

// SGDrainState DrainWorker 的失败计数和 drain_stuck 状态。
// 稀疏表：仅在换绑异常时才有行；成功换绑后删除对应行（保持表小）。
type SGDrainState struct {
	ID         uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	InstanceID string     `gorm:"uniqueIndex;column:instance_id;size:64" json:"instance_id"`
	Identifier string     `gorm:"index;size:191;default:''" json:"identifier"`
	FailCount  int        `gorm:"default:0" json:"fail_count"`
	StuckAt    *time.Time `json:"stuck_at,omitempty"`
	LastError  string     `gorm:"type:text" json:"last_error"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// TableName 固定表名。
func (SGDrainState) TableName() string {
	return "sg_drain_state"
}

// GetDrainState 查询指定实例的 drain 状态；不存在返回 nil, nil。
func GetDrainState(ctx context.Context, instanceID string) (*SGDrainState, error) {
	var s SGDrainState
	err := DB(ctx).Where("instance_id = ?", instanceID).First(&s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// IncrDrainFail 递增 fail_count。不存在则插入 fail_count=1 行。
// 返回最新的 fail_count 以便调用方判断是否需要标 stuck_at。
func IncrDrainFail(ctx context.Context, instanceID, identifier, errMsg string) (int, error) {
	now := time.Now()

	s := SGDrainState{
		InstanceID: instanceID,
		FailCount:  1,
		LastError:  errMsg,
		UpdatedAt:  now,
	}
	// upsert：冲突时递增 fail_count 并更新 last_error
	err := DB(ctx).Clauses(clause.OnConflict{

		Columns: []clause.Column{{Name: "instance_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"fail_count": gorm.Expr("fail_count + 1"),
			"last_error": errMsg,
			"updated_at": now,
		}),
	}).Create(&s).Error
	if err != nil {
		return 0, err
	}
	// 查一次拿最新 fail_count
	var latest SGDrainState
	if err := DB(ctx).Where("instance_id = ?", instanceID).First(&latest).Error; err != nil {
		return 0, err
	}
	return latest.FailCount, nil
}

// MarkDrainStuck 标记 stuck_at=NOW()。
func MarkDrainStuck(ctx context.Context, instanceID string) error {

	now := time.Now()
	return DB(ctx).Model(&SGDrainState{}).
		Where("instance_id = ?", instanceID).
		Updates(map[string]interface{}{
			"stuck_at":   now,
			"updated_at": now,
		}).Error
}

// ClearDrainState 删除指定实例的 drain 状态行（成功换绑后调用）。
func ClearDrainState(ctx context.Context, instanceID string) error {
	return DB(ctx).Where("instance_id = ?", instanceID).Delete(&SGDrainState{}).Error
}

// ListStuckDrainStates 返回所有 stuck_at 非空的记录（Guardian 告警用）。
func ListStuckDrainStates(ctx context.Context) ([]SGDrainState, error) {
	var list []SGDrainState
	err := DB(ctx).Where("stuck_at IS NOT NULL").Find(&list).Error
	return list, err
}
