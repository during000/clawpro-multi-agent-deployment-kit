package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// SG 池成员状态常量。
const (
	SGStatusActive   = "ACTIVE"   // 参与规则投影 + 接受新实例分配
	SGStatusFrozen   = "FROZEN"   // 不投影 + 不分配；等 Drain 把实例搬走
	SGStatusDraining = "DRAINING" // 显式废弃中；守护任务在 5 条件满足时删云端 SG
	SGStatusRetired  = "RETIRED"  // 云端 SG 已不存在，DB 保留审计；不参与任何选择/投影/drain
)

// DefaultSGPoolAutoScaleThreshold base 达到此值时触发扩容。
// 腾讯云单 SG 硬上限 2000，默认 1800 留 200 buffer。
const DefaultSGPoolAutoScaleThreshold = 1800

// MaxSGPerRuleSet 每个 RuleSet 子池的 ACTIVE SG 数硬上限。
// 20 × 2000 = 4w 实例容量上限。按 rule_set_id 独立计数（不是整个 identifier）。
const MaxSGPerRuleSet = 20

// SGPoolHardLimit 单 SG 腾讯云硬上限；SelectSGForNewInstance 的 buffer fallback 用它作上界。
const SGPoolHardLimit = 2000

// ManagedSGPool 记录 clawpro 管理的所有云端 SG。
//
// 与旧 sg-managed-sharding 方案的本质区别：
//   - 无 parent_sg_id 字段（没有 base/shard 区分）
//   - 所有 SG 通过 rule_set_id 挂到某个 RuleSet；规则 fan-out 按 rule_set_id 圈
//   - 状态机三态：ACTIVE / FROZEN / DRAINING（不再有 shard 子概念）
type ManagedSGPool struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Identifier string `gorm:"index;size:191;default:''" json:"identifier"`

	SGID        string     `gorm:"column:sg_id;uniqueIndex;size:64;not null" json:"sg_id"`
	SGName      string     `gorm:"column:sg_name;size:191;default:''" json:"sg_name"` // 云端 SG 名（由 Guardian 每 5 分钟从云 API 同步）
	RuleSetID   uint       `gorm:"index;not null;default:0" json:"rule_set_id"`       // 所属 RuleSet。default:0 让 SQLite AutoMigrate 能用 ADD COLUMN NOT NULL DEFAULT 0 给存量行补值（存量表由旧 sg-managed-sharding 留下，没有 rule_set_id 列）
	RuleVersion int        `gorm:"default:0" json:"rule_version"`                     // 最近成功同步到的 RuleSet.Version
	Status      string     `gorm:"type:varchar(16);index;default:'ACTIVE'" json:"status"`
	CVMCount    int        `gorm:"column:cvm_count;default:0" json:"cvm_count"`
	CVMCountAt  *time.Time `gorm:"column:cvm_count_at" json:"cvm_count_at,omitempty"`
	DrainedAt   *time.Time `json:"drained_at,omitempty"`
	DriftAt     *time.Time `json:"drift_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 固定表名。
func (ManagedSGPool) TableName() string {
	return "managed_sg_pool"
}

// IncrementSGCVMCount 原子 +1。
func IncrementSGCVMCount(ctx context.Context, sgID string) error {
	return DB(ctx).Model(&ManagedSGPool{}).
		Where("sg_id = ?", sgID).
		UpdateColumn("cvm_count", gorm.Expr("cvm_count + 1")).Error
}

// DecrementSGCVMCount 原子 -1，WHERE cvm_count > 0 保护不跌破 0。
func DecrementSGCVMCount(ctx context.Context, sgID string) error {
	return DB(ctx).Model(&ManagedSGPool{}).
		Where("sg_id = ? AND cvm_count > 0", sgID).
		UpdateColumn("cvm_count", gorm.Expr("cvm_count - 1")).Error
}

// IsManagedSG 判断某 sg_id 是否在 clawpro 池中（任意状态）。
// 用于校验 "从其他安全组导入规则" 时禁止选自己池里的 SG（避免循环）。
func IsManagedSG(ctx context.Context, sgID string) (bool, error) {
	var count int64
	err := DB(ctx).Model(&ManagedSGPool{}).Where("sg_id = ?", sgID).Count(&count).Error

	return count > 0, err
}

// ListActiveSGsByRuleSet 按 rule_set_id 查 ACTIVE SG，按 cvm_count 升序
// （SelectSGForNewInstance 的常规路径使用）。排除 drift_at 非空的成员。
func ListActiveSGsByRuleSet(ctx context.Context, ruleSetID uint) ([]ManagedSGPool, error) {
	var sgs []ManagedSGPool
	err := DB(ctx).Where("rule_set_id = ? AND status = ? AND drift_at IS NULL",
		ruleSetID, SGStatusActive).
		Order("cvm_count ASC, created_at ASC").
		Find(&sgs).Error
	return sgs, err
}

// ListActiveSGsForFanout 按 rule_set_id 查所有 ACTIVE SG，用于规则投影 fan-out。
// 和 ListActiveSGsByRuleSet 的区别：这里不排除 drift_at 非空（drift SG 也要被重写整包覆盖，
// 成功后 drift_at 被清除）。
func ListActiveSGsForFanout(ctx context.Context, ruleSetID uint) ([]ManagedSGPool, error) {
	var sgs []ManagedSGPool
	err := DB(ctx).Where("rule_set_id = ? AND status = ?", ruleSetID, SGStatusActive).
		Order("created_at ASC").
		Find(&sgs).Error
	return sgs, err
}

// ListFrozenSGs 返回所有 FROZEN SG（DrainWorker 扫描起点）。
func ListFrozenSGs(ctx context.Context) ([]ManagedSGPool, error) {

	var sgs []ManagedSGPool
	err := DB(ctx).Where("status = ?", SGStatusFrozen).Find(&sgs).Error
	return sgs, err
}

// CountActiveSGsInRuleSet 统计某 RuleSet 下的 ACTIVE SG 数（AutoScaleSG 撞上限检查用）。
// 按 rule_set_id 分别计数，不是整个 identifier——为未来多 RuleSet 做预留。
func CountActiveSGsInRuleSet(ctx context.Context, ruleSetID uint) (int, error) {
	var count int64
	err := DB(ctx).Model(&ManagedSGPool{}).
		Where("rule_set_id = ? AND status = ?", ruleSetID, SGStatusActive).
		Count(&count).Error
	return int(count), err
}

// UpdateSGRuleVersion 规则 fan-out 成功后，更新本 SG 的 rule_version 为最新版。
func UpdateSGRuleVersion(ctx context.Context, sgID string, version int) error {
	return DB(ctx).Model(&ManagedSGPool{}).
		Where("sg_id = ?", sgID).
		Updates(map[string]interface{}{
			"rule_version": version,
			"drift_at":     nil, // 成功同步即清除 drift 标记
		}).Error
}

// MarkSGDrift fan-out 失败时标记 drift_at=NOW()。Guardian 后续自愈。
func MarkSGDrift(ctx context.Context, sgID string) error {
	now := time.Now()
	return DB(ctx).Model(&ManagedSGPool{}).
		Where("sg_id = ?", sgID).
		UpdateColumn("drift_at", now).Error
}

// UpdateSGCVMCount 设置绝对值（Guardian cvm_count 纠偏使用）。
func UpdateSGCVMCount(ctx context.Context, sgID string, count int) error {
	now := time.Now()
	return DB(ctx).Model(&ManagedSGPool{}).
		Where("sg_id = ?", sgID).
		Updates(map[string]interface{}{
			"cvm_count":    count,
			"cvm_count_at": &now,
		}).Error
}

// UpdateSGName 更新某 SG 的云端名称（Guardian 5 分钟一次从云 API 同步）。
func UpdateSGName(ctx context.Context, sgID, name string) error {
	return DB(ctx).Model(&ManagedSGPool{}).
		Where("sg_id = ?", sgID).
		UpdateColumn("sg_name", name).Error
}

// NextSGOrdinalForRuleSet 返回给某 RuleSet 建下一个分片时要用的序号。
// 算法：该 RuleSet 下所有 clawpro 自建行（rule_version > 0，含 FROZEN / DRAINING）的总数 + 1。
//
// 排除 rule_version = 0 的行：这些是初始化导入的遗留用户 SG（非 clawpro 创建，
// 名字不是 clawpro-sg-* 格式，不占编号位）。
// FROZEN 和 DRAINING 成员的 SG 名字还占着编号，必须计入，取总数 + 1 保证新号不撞已有 SG。
func NextSGOrdinalForRuleSet(ctx context.Context, ruleSetID uint) (int, error) {
	var count int64
	err := DB(ctx).Model(&ManagedSGPool{}).
		Where("rule_set_id = ? AND rule_version > 0", ruleSetID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count) + 1, nil
}
