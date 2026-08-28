package model

import (
	"context"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// DefaultRuleSetName 默认规则组名称。本期每租户只有一个 RuleSet，名字固定为 "clawpro-default"。
// (identifier, name) 组合唯一索引为未来 per-user_group RuleSet 预留（未来可加 name='dev' 等）。
const DefaultRuleSetName = "ClawPro-Default"

// DefaultRuleSetRemark 默认规则组备注。SG RuleSet 初始化存量迁移 / 前端空态建组默认值都用它。
const DefaultRuleSetRemark = "Agent 默认安全组"

// RuleSet 规则组：规则真相源。
// 云端 SG 规则仅是 RuleSet 的"投影"，不再作为读写主路径：
//   - 管理员读规则走本表（毫秒级），不调云 API
//   - 规则改动 → UPDATE rule_sets.rules + version++ → fan-out 到所有 ACTIVE SG
//   - Guardian 漂移检测：rule_version < RuleSet.Version → 自愈
//
// 本期每个租户仅有一行 RuleSet（name="default"），UserGroupIDs="[]" / IsDefault=true。
// UserGroupIDs 和 IsDefault 是为"多 RuleSet 按用户组路由"预留的字段，本期不消费。
type RuleSet struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Identifier string `gorm:"uniqueIndex:idx_rs_ident_name;size:191;default:''" json:"identifier"`
	// Name 在 (identifier, name) 维度唯一：同一租户下不允许重名。
	// 由 idx_rs_ident_name 复合唯一索引保证（本字段 + Identifier）。
	Name string `gorm:"uniqueIndex:idx_rs_ident_name;size:64;default:'ClawPro-Default'" json:"name"`
	// Description 管理员自定义备注，UI 展示。对业务逻辑无影响。
	Description string `gorm:"size:256;default:''" json:"description"`
	Rules       string `gorm:"type:text" json:"rules"` // JSON 数组字符串：[]Rule
	Version     int    `gorm:"default:1" json:"version"`

	// ── 预留字段（为未来"多 RuleSet / 按用户组分流"做准备，本期不消费业务逻辑）────
	//
	// UserGroupIDs：当前 RuleSet 作用到的用户组 ID 列表（JSON 字符串 []string）。
	// 未来 SelectSGForNewInstance 会根据实例拥有者的用户组，挑到匹配的 RuleSet 分配 SG。
	// 本期恒为 "" 或 "[]"。
	UserGroupIDs string `gorm:"type:text" json:"user_group_ids"`

	// IsDefault：默认规则组标记。未来某个用户组没有显式匹配到 RuleSet 时，走 IsDefault=true 的兜底。
	// 约束：同一租户下最多一行 IsDefault=true（未来加复合索引强制）。
	// 本期每租户仅有一行，恒为 true。
	IsDefault bool `gorm:"default:true" json:"is_default"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 固定表名。
func (RuleSet) TableName() string {
	return "rule_sets"
}

// GetRuleSetByName 按 (identifier, name) 组合查询 RuleSet。
// 不存在返回 gorm.ErrRecordNotFound，调用方自行判断是否用 sentinel 分支。
func GetRuleSetByName(ctx context.Context, name string) (*RuleSet, error) {
	var rs RuleSet
	err := DB(ctx).Where("name = ?", name).First(&rs).Error
	if err != nil {
		return nil, err
	}
	return &rs, nil
}

// GetDefaultRuleSet 查询当前租户的默认规则组（is_default=true）。
// 本期每租户仅一行；未来多 RuleSet 时作为兜底。
func GetDefaultRuleSet(ctx context.Context) (*RuleSet, error) {
	var rs RuleSet
	err := DB(ctx).Where("is_default = ?", true).First(&rs).Error
	if err != nil {
		return nil, err
	}
	return &rs, nil
}

// IncrementRuleSetVersion 在事务中给指定 RuleSet 的 rules 更新 + version++。
// 调用方 MUST 在外层 tx 里先用 SELECT ... FOR UPDATE 锁行，避免并发覆盖。
func IncrementRuleSetVersion(tx *gorm.DB, rsID uint, newRulesJSON string) error {
	return tx.Model(&RuleSet{}).
		Where("id = ?", rsID).
		Updates(map[string]interface{}{
			"rules":   newRulesJSON,
			"version": gorm.Expr("version + 1"),
		}).Error
}

// LockRuleSetForUpdate 悲观锁读取 RuleSet 行（SELECT ... FOR UPDATE）。
// 返回带最新 version 的行，调用方基于此 version 做 version++ 写入。
func LockRuleSetForUpdate(tx *gorm.DB, name string) (*RuleSet, error) {
	var rs RuleSet
	err := tx.Set("gorm:query_option", "FOR UPDATE").
		Where("name = ?", name).
		First(&rs).Error
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgLockRuleSetFailed, name)
	}
	return &rs, nil
}
