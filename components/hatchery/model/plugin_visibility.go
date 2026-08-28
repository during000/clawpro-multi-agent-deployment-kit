package model

import (
	"context"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"time"

	"gorm.io/gorm"
)

// PluginVisibilityGroup 插件-分组可见性关联。
// 仅当 Plugin.VisibilityType = "group" 时，此表中的记录生效。
type PluginVisibilityGroup struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	Identifier string    `gorm:"uniqueIndex:idx_pvg_unique;index;default:''" json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	PluginID   uint      `gorm:"uniqueIndex:idx_pvg_unique;not null;index" json:"plugin_id"`
	GroupID    uint      `gorm:"uniqueIndex:idx_pvg_unique;not null;index" json:"group_id"`
}

// ──────────────────────────────────────────────
// 辅助查询函数
// ──────────────────────────────────────────────

// GetPluginVisibilityGroupIDs 批量查询多个插件的可见分组 ID。
// 返回 map[pluginID][]groupID。
func GetPluginVisibilityGroupIDs(ctx context.Context, pluginIDs []uint) (map[uint][]uint, error) {
	if len(pluginIDs) == 0 {
		return nil, nil
	}
	var rows []PluginVisibilityGroup
	if err := DB(ctx).Where("plugin_id IN ?", pluginIDs).Find(&rows).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgPluginVisQueryGroupIDsFailed)
	}
	result := make(map[uint][]uint)
	for _, r := range rows {
		result[r.PluginID] = append(result[r.PluginID], r.GroupID)
	}
	return result, nil
}

// SetPluginVisibility 设置插件的应用范围。
// 在事务内执行：删除旧关联 → 创建新关联 → 更新 visibility_type。
// visibilityType 为 "all" 时清空关联；为 "group" 时创建新关联。
func SetPluginVisibility(tx *gorm.DB, pluginID uint, visibilityType string, groupIDs []uint) error {
	// 删除该插件的所有旧关联
	if err := tx.Where("plugin_id = ?", pluginID).Delete(&PluginVisibilityGroup{}).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgPluginVisDeleteOldFailed)
	}
	// 如果是 group 类型，批量创建新关联
	if visibilityType == VisibilityGroup && len(groupIDs) > 0 {
		records := make([]PluginVisibilityGroup, 0, len(groupIDs))
		for _, gid := range groupIDs {
			records = append(records, PluginVisibilityGroup{
				PluginID: pluginID,
				GroupID:  gid,
			})
		}
		if err := tx.Create(&records).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgPluginVisCreateFailed)
		}
	}
	// 更新 visibility_type
	if err := tx.Model(&Plugin{}).Where("id = ?", pluginID).Update("visibility_type", visibilityType).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgPluginVisUpdateTypeFailed)
	}
	return nil
}

// CopyPluginVisibility 从同 slug 的旧版本复制应用范围到新版本。
// 查找同 slug 最新的非当前版本，复制其 visibility_type 和分组关联。
// 如果无旧版本，新版本保持默认 "all"，不做任何操作。
func CopyPluginVisibility(tx *gorm.DB, slug string, toPluginID uint) error {
	// 查找同 slug 的最新旧版本（排除当前新建的 plugin）
	var oldPlugin Plugin
	err := tx.Where("slug = ? AND id != ?", slug, toPluginID).
		Order("version_major DESC, version_minor DESC, version_patch DESC").
		First(&oldPlugin).Error
	if err != nil {
		// 无旧版本（首次上传），保持默认 all
		return nil
	}

	// 旧版本是 all，无需复制关联，只确保新版本也是 all（默认值）
	if oldPlugin.VisibilityType != VisibilityGroup {
		return nil
	}

	// 复制 visibility_type
	if err := tx.Model(&Plugin{}).Where("id = ?", toPluginID).
		Update("visibility_type", oldPlugin.VisibilityType).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgPluginVisCopyTypeFailed)
	}

	// 复制分组关联
	var oldGroups []PluginVisibilityGroup
	if err := tx.Where("plugin_id = ?", oldPlugin.ID).Find(&oldGroups).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgPluginVisQueryOldFailed)
	}
	if len(oldGroups) > 0 {
		records := make([]PluginVisibilityGroup, 0, len(oldGroups))
		for _, g := range oldGroups {
			records = append(records, PluginVisibilityGroup{
				PluginID: toPluginID,
				GroupID:  g.GroupID,
			})
		}
		if err := tx.Create(&records).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgPluginVisCopyGroupsFailed)
		}
	}
	return nil
}

// CleanupPluginVisibilityByGroupID 清理某分组被删除后的插件可见性关联。
// 必须传入事务 tx，确保与分组删除在同一事务中执行。
func CleanupPluginVisibilityByGroupID(tx *gorm.DB, groupID uint) error {
	return tx.Where("group_id = ?", groupID).Delete(&PluginVisibilityGroup{}).Error
}

// CleanupPluginVisibilityByPluginID 清理某插件被删除后的可见性关联。
// 由插件删除逻辑调用。
func CleanupPluginVisibilityByPluginID(tx *gorm.DB, pluginID uint) error {
	return tx.Where("plugin_id = ?", pluginID).Delete(&PluginVisibilityGroup{}).Error
}

// IsGroupUsedByPluginVisibility 检查用户组是否被插件可见性配置引用。
// 返回 true 表示该用户组被至少一个插件的可见性配置使用，不应被删除。
func IsGroupUsedByPluginVisibility(ctx context.Context, groupID uint) (bool, error) {
	var count int64
	if err := DB(ctx).Model(&PluginVisibilityGroup{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
