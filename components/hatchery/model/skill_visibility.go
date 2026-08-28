package model

import (
	"context"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// SkillVisibilityGroup 技能-分组可见性关联。
// 仅当 Skill.VisibilityType = "group" 时，此表中的记录生效。
type SkillVisibilityGroup struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	Identifier string    `gorm:"uniqueIndex:idx_svg_unique;index;default:''" json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	SkillID    uint      `gorm:"uniqueIndex:idx_svg_unique;not null;index" json:"skill_id"`
	GroupID    uint      `gorm:"uniqueIndex:idx_svg_unique;not null;index" json:"group_id"`
}

// ──────────────────────────────────────────────
// 辅助查询函数
// ──────────────────────────────────────────────

// IsSkillVisibleToUser 检查某技能对指定用户是否可见。
// visibility_type=all 时返回 (true, nil)。
// visibility_type=group 时检查用户所属分组与技能关联分组是否有交集。
// 查询失败时返回 (false, err)，调用方应按不可见处理并记录日志。
func IsSkillVisibleToUser(ctx context.Context, skill *Skill, userID uint) (bool, error) {
	if skill.VisibilityType != VisibilityGroup {
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
	if err := DB(ctx).Model(&SkillVisibilityGroup{}).
		Where("skill_id = ? AND group_id IN ?", skill.ID, userGroupIDs).
		Count(&count).Error; err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgSkillVisQueryAssocFailed)
	}
	return count > 0, nil
}

// GetSkillVisibilityGroupIDs 批量查询多个技能的可见分组 ID。
// 返回 map[skillID][]groupID。
func GetSkillVisibilityGroupIDs(ctx context.Context, skillIDs []uint) (map[uint][]uint, error) {
	if len(skillIDs) == 0 {
		return nil, nil
	}
	var rows []SkillVisibilityGroup
	if err := DB(ctx).Where("skill_id IN ?", skillIDs).Find(&rows).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSkillVisBatchQueryAssocFailed)
	}
	result := make(map[uint][]uint)
	for _, r := range rows {
		result[r.SkillID] = append(result[r.SkillID], r.GroupID)
	}
	return result, nil
}

// SetSkillVisibility 设置技能的应用范围。
// 在事务内执行：删除旧关联 → 创建新关联 → 更新 visibility_type。
// visibilityType 为 "all" 时清空关联；为 "group" 时创建新关联。
func SetSkillVisibility(tx *gorm.DB, skillID uint, visibilityType string, groupIDs []uint) error {
	// 删除该技能的所有旧关联
	if err := tx.Where("skill_id = ?", skillID).Delete(&SkillVisibilityGroup{}).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSkillVisDeleteOldAssocFailed)
	}
	// 如果是 group 类型，批量创建新关联
	if visibilityType == VisibilityGroup && len(groupIDs) > 0 {
		records := make([]SkillVisibilityGroup, 0, len(groupIDs))
		for _, gid := range groupIDs {
			records = append(records, SkillVisibilityGroup{
				SkillID: skillID,
				GroupID: gid,
			})
		}
		if err := tx.Create(&records).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSkillVisCreateAssocFailed)
		}
	}
	// 更新 visibility_type
	if err := tx.Model(&Skill{}).Where("id = ?", skillID).Update("visibility_type", visibilityType).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSkillVisUpdateVisibilityFailed)
	}
	return nil
}

// CopySkillVisibility 从同 slug 的旧版本复制应用范围到新版本。
// 查找同 slug 最新的非当前版本，复制其 visibility_type 和分组关联。
// 如果无旧版本，新版本保持默认 "all"，不做任何操作。
func CopySkillVisibility(tx *gorm.DB, slug string, toSkillID uint) error {
	// 查找同 slug 的最新旧版本（排除当前新建的 skill）
	var oldSkill Skill
	err := tx.Where("slug = ? AND id != ?", slug, toSkillID).
		Order("version_major DESC, version_minor DESC, version_patch DESC").
		First(&oldSkill).Error
	if err != nil {
		// 无旧版本（首次上传），保持默认 all
		return nil
	}

	// 旧版本是 all，无需复制关联，只确保新版本也是 all（默认值）
	if oldSkill.VisibilityType != VisibilityGroup {
		return nil
	}

	// 复制 visibility_type
	if err := tx.Model(&Skill{}).Where("id = ?", toSkillID).
		Update("visibility_type", oldSkill.VisibilityType).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgPluginVisCopyTypeFailed)
	}

	// 复制分组关联
	var oldGroups []SkillVisibilityGroup
	if err := tx.Where("skill_id = ?", oldSkill.ID).Find(&oldGroups).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgPluginVisQueryOldFailed)
	}
	if len(oldGroups) > 0 {
		records := make([]SkillVisibilityGroup, 0, len(oldGroups))
		for _, g := range oldGroups {
			records = append(records, SkillVisibilityGroup{
				SkillID: toSkillID,
				GroupID: g.GroupID,
			})
		}
		if err := tx.Create(&records).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSkillVisCopyAssocFailed)
		}
	}
	return nil
}

// CleanupSkillVisibilityByGroupID 清理某分组被删除后的技能可见性关联。
// 必须传入事务 tx，确保与分组删除在同一事务中执行。
func CleanupSkillVisibilityByGroupID(tx *gorm.DB, groupID uint) error {
	return tx.Where("group_id = ?", groupID).Delete(&SkillVisibilityGroup{}).Error
}

// CleanupSkillVisibilityBySkillID 清理某技能被删除后的可见性关联。
// 由技能删除逻辑调用。
func CleanupSkillVisibilityBySkillID(tx *gorm.DB, skillID uint) error {
	return tx.Where("skill_id = ?", skillID).Delete(&SkillVisibilityGroup{}).Error
}

// IsGroupUsedBySkillVisibility 检查用户组是否被技能可见性配置引用。
// 返回 true 表示该用户组被至少一个技能的可见性配置使用，不应被删除。
func IsGroupUsedBySkillVisibility(ctx context.Context, groupID uint) (bool, error) {
	var count int64
	if err := DB(ctx).Model(&SkillVisibilityGroup{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
