package model

import (
	"context"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// SkillBundleVisibilityGroup 技能包-分组可见性关联。
// 仅当 SkillBundle.VisibilityType = "group" 时，此表中的记录生效。
type SkillBundleVisibilityGroup struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	Identifier    string    `gorm:"uniqueIndex:idx_sbvg_unique;index;default:''" json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	SkillBundleID uint      `gorm:"uniqueIndex:idx_sbvg_unique;not null;index" json:"skill_bundle_id"`
	GroupID       uint      `gorm:"uniqueIndex:idx_sbvg_unique;not null;index" json:"group_id"`
}

// ──────────────────────────────────────────────
// 辅助查询函数
// ──────────────────────────────────────────────

// IsSkillBundleVisibleToUser 检查某技能包对指定用户是否可见。
// visibility_type=all 时返回 (true, nil)。
// visibility_type=group 时检查用户所属分组与技能包关联分组是否有交集。
// 查询失败时返回 (false, err)，调用方应按不可见处理并记录日志。
func IsSkillBundleVisibleToUser(ctx context.Context, bundle *SkillBundle, userID uint) (bool, error) {
	if bundle.VisibilityType != "group" {
		return true, nil // all 或空值均视为全部可见
	}
	userGroupIDs, err := GetUserGroupIDs(ctx, userID)
	if err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgQueryUserGroupFailed)
	}
	if len(userGroupIDs) == 0 {
		return false, nil // 用户不属于任何分组
	}
	var count int64
	if err := DB(ctx).Model(&SkillBundleVisibilityGroup{}).
		Where("skill_bundle_id = ? AND group_id IN ?", bundle.ID, userGroupIDs).
		Count(&count).Error; err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgSBVisQueryAssocFailed)
	}
	return count > 0, nil
}

// GetSkillBundleVisibilityGroupIDs 批量查询多个技能包的可见分组 ID。
// 返回 map[bundleID][]groupID。
func GetSkillBundleVisibilityGroupIDs(ctx context.Context, bundleIDs []uint) (map[uint][]uint, error) {
	if len(bundleIDs) == 0 {
		return nil, nil
	}
	var rows []SkillBundleVisibilityGroup
	if err := DB(ctx).Where("skill_bundle_id IN ?", bundleIDs).Find(&rows).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSBVisBatchQueryAssocFailed)
	}
	result := make(map[uint][]uint)
	for _, r := range rows {
		result[r.SkillBundleID] = append(result[r.SkillBundleID], r.GroupID)
	}
	return result, nil
}

// SetSkillBundleVisibility 设置技能包的应用范围。
// 在事务内执行：删除旧关联 → 创建新关联 → 更新 visibility_type。
// visibilityType 为 "all" 时清空关联；为 "group" 时创建新关联。
func SetSkillBundleVisibility(tx *gorm.DB, bundleID uint, visibilityType string, groupIDs []uint) error {
	// 删除该技能包的所有旧关联
	if err := tx.Where("skill_bundle_id = ?", bundleID).Delete(&SkillBundleVisibilityGroup{}).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSBVisDeleteOldAssocFailed)
	}
	// 如果是 group 类型，批量创建新关联
	if visibilityType == "group" && len(groupIDs) > 0 {
		records := make([]SkillBundleVisibilityGroup, 0, len(groupIDs))
		for _, gid := range groupIDs {
			records = append(records, SkillBundleVisibilityGroup{
				SkillBundleID: bundleID,
				GroupID:       gid,
			})
		}
		if err := tx.Create(&records).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSBVisCreateAssocFailed)
		}
	}
	// 更新 visibility_type
	if err := tx.Model(&SkillBundle{}).Where("id = ?", bundleID).Update("visibility_type", visibilityType).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSBVisUpdateVisibilityFailed)
	}
	return nil
}

// CleanupSkillBundleVisibilityByGroupID 清理某分组被删除后的技能包可见性关联。
// 必须传入事务 tx，确保与分组删除在同一事务中执行。
func CleanupSkillBundleVisibilityByGroupID(tx *gorm.DB, groupID uint) error {
	return tx.Where("group_id = ?", groupID).Delete(&SkillBundleVisibilityGroup{}).Error
}

// CleanupSkillBundleVisibilityByBundleID 清理某技能包被删除后的可见性关联。
// 由技能包删除逻辑调用。
func CleanupSkillBundleVisibilityByBundleID(tx *gorm.DB, bundleID uint) error {
	return tx.Where("skill_bundle_id = ?", bundleID).Delete(&SkillBundleVisibilityGroup{}).Error
}

// IsGroupUsedBySkillBundleVisibility 检查用户组是否被技能包可见性配置引用。
// 返回 true 表示该用户组被至少一个技能包的可见性配置使用，不应被删除。
func IsGroupUsedBySkillBundleVisibility(ctx context.Context, groupID uint) (bool, error) {
	var count int64
	if err := DB(ctx).Model(&SkillBundleVisibilityGroup{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
