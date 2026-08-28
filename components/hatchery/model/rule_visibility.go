package model

import (
	"context"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// RuleVisibilityGroup 规范-分组可见性关联。
// 语义/结构完全对齐 SkillVisibilityGroup（本地 agent 二期沿用 skill 侧成熟设计）。
// 仅当 EnterpriseRule.VisibilityType = "group" 时，此表中的记录生效。
//
// 设计决策：镜像而非泛型化 SkillVisibilityGroup。原因：
//  1. skill 侧代码路径（migrate、group 删除清理、group_closure、can_delete_group 等）均已强绑定
//     SkillVisibilityGroup，泛型化改动面过大；
//  2. rule 与 skill 表结构可能未来独立演化；
//  3. 复用面主要在 UserGroup / GetUserGroupIDs / GetGroupsByIDs 等**共用 helper**，那些无需镜像。
type RuleVisibilityGroup struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	Identifier string    `gorm:"uniqueIndex:idx_ervg_unique;index;default:''" json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	RuleID     uint      `gorm:"uniqueIndex:idx_ervg_unique;not null;index" json:"rule_id"`
	GroupID    uint      `gorm:"uniqueIndex:idx_ervg_unique;not null;index" json:"group_id"`
}

// TableName 表名固定。
func (RuleVisibilityGroup) TableName() string { return "rule_visibility_groups" }

// ──────────────────────────────────────────────
// 辅助查询函数（对齐 skill_visibility.go）
// ──────────────────────────────────────────────

// IsRuleVisibleToUser 检查某规范对指定用户是否可见。
// visibility_type=all 时返回 (true, nil)。
// visibility_type=group 时检查用户所属分组与规范关联分组是否有交集。
// 查询失败时返回 (false, err)，调用方应按不可见处理并记录日志。
func IsRuleVisibleToUser(ctx context.Context, rule *EnterpriseRule, userID uint) (bool, error) {
	if rule.VisibilityType != VisibilityGroup {
		return true, nil // all 或空值均视为全部可见
	}
	userGroupIDs, err := GetUserGroupIDs(ctx, userID)
	if err != nil {
		return false, err
	}
	if len(userGroupIDs) == 0 {
		return false, nil // 用户不属于任何分组
	}
	var count int64
	if err := DB(ctx).Model(&RuleVisibilityGroup{}).
		Where("rule_id = ? AND group_id IN ?", rule.ID, userGroupIDs).
		Count(&count).Error; err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgSkillVisQueryAssocFailed)
	}
	return count > 0, nil
}

// GetRuleVisibilityGroupIDs 批量查询多个规范的可见分组 ID。
// 返回 map[ruleID][]groupID。
func GetRuleVisibilityGroupIDs(ctx context.Context, ruleIDs []uint) (map[uint][]uint, error) {
	if len(ruleIDs) == 0 {
		return nil, nil
	}
	var rows []RuleVisibilityGroup
	if err := DB(ctx).Where("rule_id IN ?", ruleIDs).Find(&rows).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSkillVisBatchQueryAssocFailed)
	}
	result := make(map[uint][]uint)
	for _, r := range rows {
		result[r.RuleID] = append(result[r.RuleID], r.GroupID)
	}
	return result, nil
}

// SetRuleVisibility 设置规范的应用范围。
// 在事务内执行：删除旧关联 → 创建新关联 → 更新 visibility_type。
// visibilityType 为 "all" 时清空关联；为 "group" 时创建新关联。
func SetRuleVisibility(tx *gorm.DB, ruleID uint, visibilityType string, groupIDs []uint) error {
	// 删除该规范的所有旧关联
	if err := tx.Where("rule_id = ?", ruleID).Delete(&RuleVisibilityGroup{}).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSkillVisDeleteOldAssocFailed)
	}
	// 如果是 group 类型，批量创建新关联
	if visibilityType == VisibilityGroup && len(groupIDs) > 0 {
		records := make([]RuleVisibilityGroup, 0, len(groupIDs))
		for _, gid := range groupIDs {
			records = append(records, RuleVisibilityGroup{
				RuleID:  ruleID,
				GroupID: gid,
			})
		}
		if err := tx.Create(&records).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSkillVisCreateAssocFailed)
		}
	}
	// 更新 visibility_type
	if err := tx.Model(&EnterpriseRule{}).Where("id = ?", ruleID).Update("visibility_type", visibilityType).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSkillVisUpdateVisibilityFailed)
	}
	return nil
}

// CopyRuleVisibility 从同 slug 的旧版本复制应用范围到新版本。
// 查找同 slug 最新的非当前版本，复制其 visibility_type 和分组关联。
// 如果无旧版本，新版本保持默认 "all"，不做任何操作。
func CopyRuleVisibility(tx *gorm.DB, slug string, toRuleID uint) error {
	// 查找同 slug 的最新旧版本（排除当前新建的 rule）
	var oldRule EnterpriseRule
	err := tx.Where("slug = ? AND id != ?", slug, toRuleID).
		Order("version_major DESC, version_minor DESC, version_patch DESC").
		First(&oldRule).Error
	if err != nil {
		// 无旧版本（首次上传），保持默认 all
		return nil
	}

	// 旧版本是 all，无需复制关联，只确保新版本也是 all（默认值）
	if oldRule.VisibilityType != VisibilityGroup {
		return nil
	}

	// 复制 visibility_type
	if err := tx.Model(&EnterpriseRule{}).Where("id = ?", toRuleID).
		Update("visibility_type", oldRule.VisibilityType).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgPluginVisCopyTypeFailed)
	}

	// 复制分组关联
	var oldGroups []RuleVisibilityGroup
	if err := tx.Where("rule_id = ?", oldRule.ID).Find(&oldGroups).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgPluginVisQueryOldFailed)
	}
	if len(oldGroups) > 0 {
		records := make([]RuleVisibilityGroup, 0, len(oldGroups))
		for _, g := range oldGroups {
			records = append(records, RuleVisibilityGroup{
				RuleID:  toRuleID,
				GroupID: g.GroupID,
			})
		}
		if err := tx.Create(&records).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSkillVisCopyAssocFailed)
		}
	}
	return nil
}

// CleanupRuleVisibilityByGroupID 清理某分组被删除后的规范可见性关联。
// 必须传入事务 tx，确保与分组删除在同一事务中执行。
func CleanupRuleVisibilityByGroupID(tx *gorm.DB, groupID uint) error {
	return tx.Where("group_id = ?", groupID).Delete(&RuleVisibilityGroup{}).Error
}

// CleanupRuleVisibilityByRuleID 清理某规范被删除后的可见性关联。
// 由规范删除逻辑调用。
func CleanupRuleVisibilityByRuleID(tx *gorm.DB, ruleID uint) error {
	return tx.Where("rule_id = ?", ruleID).Delete(&RuleVisibilityGroup{}).Error
}

// IsGroupUsedByRuleVisibility 检查用户组是否被规范可见性配置引用。
// 返回 true 表示该用户组被至少一个规范的可见性配置使用，不应被删除。
func IsGroupUsedByRuleVisibility(ctx context.Context, groupID uint) (bool, error) {
	var count int64
	if err := DB(ctx).Model(&RuleVisibilityGroup{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
