package model

import (
	"context"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// RoleVisibilityGroup 角色-分组可见性关联。
// 仅当 OpenClawRole.VisibilityType = "group" 时，此表中的记录生效。
type RoleVisibilityGroup struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	Identifier     string    `gorm:"uniqueIndex:idx_rvg_unique;index;default:''" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	OpenClawRoleID uint      `gorm:"uniqueIndex:idx_rvg_unique;not null;index" json:"open_claw_role_id"`
	GroupID        uint      `gorm:"uniqueIndex:idx_rvg_unique;not null;index" json:"group_id"`
}

// ──────────────────────────────────────────────
// 辅助查询函数
// ──────────────────────────────────────────────

// IsRoleVisibleToUser 检查某角色对指定用户是否可见。
// visibility_type=all 时返回 (true, nil)。
// visibility_type=group 时检查用户所属分组与角色关联分组是否有交集。
// 查询失败时返回 (false, err)，调用方应按不可见处理并记录日志。
func IsRoleVisibleToUser(ctx context.Context, role *OpenClawRole, userID uint) (bool, error) {
	if role.VisibilityType != "group" {
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
	if err := DB(ctx).Model(&RoleVisibilityGroup{}).
		Where("open_claw_role_id = ? AND group_id IN ?", role.ID, userGroupIDs).
		Count(&count).Error; err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgRoleVisQueryAssocFailed)
	}
	return count > 0, nil
}

// GetRoleVisibilityGroupIDs 批量查询多个角色的可见分组 ID。
// 返回 map[roleID][]groupID。
func GetRoleVisibilityGroupIDs(ctx context.Context, roleIDs []uint) (map[uint][]uint, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}
	var rows []RoleVisibilityGroup
	if err := DB(ctx).Where("open_claw_role_id IN ?", roleIDs).Find(&rows).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgRoleVisBatchQueryAssocFailed)
	}
	result := make(map[uint][]uint)
	for _, r := range rows {
		result[r.OpenClawRoleID] = append(result[r.OpenClawRoleID], r.GroupID)
	}
	return result, nil
}

// SetRoleVisibility 设置角色的应用范围。
// 在事务内执行：删除旧关联 → 创建新关联 → 更新 visibility_type。
// visibilityType 为 "all" 时清空关联；为 "group" 时创建新关联。
func SetRoleVisibility(tx *gorm.DB, roleID uint, visibilityType string, groupIDs []uint) error {
	// 删除该角色的所有旧关联
	if err := tx.Where("open_claw_role_id = ?", roleID).Delete(&RoleVisibilityGroup{}).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgRoleVisDeleteOldAssocFailed)
	}
	// 如果是 group 类型，批量创建新关联
	if visibilityType == "group" && len(groupIDs) > 0 {
		records := make([]RoleVisibilityGroup, 0, len(groupIDs))
		for _, gid := range groupIDs {
			records = append(records, RoleVisibilityGroup{
				OpenClawRoleID: roleID,
				GroupID:        gid,
			})
		}
		if err := tx.Create(&records).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgRoleVisCreateAssocFailed)
		}
	}
	// 更新 visibility_type
	if err := tx.Model(&OpenClawRole{}).Where("id = ?", roleID).Update("visibility_type", visibilityType).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgRoleVisUpdateVisibilityFailed)
	}
	return nil
}

// CleanupRoleVisibilityByGroupID 清理某分组被删除后的角色可见性关联。
// 必须传入事务 tx，确保与分组删除在同一事务中执行。
func CleanupRoleVisibilityByGroupID(tx *gorm.DB, groupID uint) error {
	return tx.Where("group_id = ?", groupID).Delete(&RoleVisibilityGroup{}).Error
}

// CleanupRoleVisibilityByRoleID 清理某角色被删除后的可见性关联。
// 由角色删除逻辑调用。
func CleanupRoleVisibilityByRoleID(tx *gorm.DB, roleID uint) error {
	return tx.Where("open_claw_role_id = ?", roleID).Delete(&RoleVisibilityGroup{}).Error
}

// IsGroupUsedByRoleVisibility 检查用户组是否被角色可见性配置引用。
// 返回 true 表示该用户组被至少一个角色的可见性配置使用，不应被删除。
func IsGroupUsedByRoleVisibility(ctx context.Context, groupID uint) (bool, error) {
	var count int64
	if err := DB(ctx).Model(&RoleVisibilityGroup{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
