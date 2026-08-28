package model

import (
	"context"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"time"

	"gorm.io/gorm"
)

// ModelVisibilityGroup 模型-分组可见性关联。
// 仅当 AIModel.VisibilityType = "group" 时，此表中的记录生效。
type ModelVisibilityGroup struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	Identifier string    `gorm:"uniqueIndex:idx_mvg_unique;index;default:''" json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	AIModelID  uint      `gorm:"uniqueIndex:idx_mvg_unique;not null;index" json:"ai_model_id"`
	GroupID    uint      `gorm:"uniqueIndex:idx_mvg_unique;not null;index" json:"group_id"`
}

// ──────────────────────────────────────────────
// 辅助查询函数
// ──────────────────────────────────────────────

// IsModelVisibleToUser 检查某模型对指定用户是否可见。
// visibility_type=all 时返回 (true, nil)。
// visibility_type=group 时检查用户所属分组与模型关联分组是否有交集。
// 查询失败时返回 (false, err)，调用方应按不可见处理并记录日志。
//
// 适用场景：单个模型的可见性判断（绑定模型 HandleSetModel / handleCustomModel）。
// 模型列表批量过滤请使用 controller 中的 filterModelsByVisibility。
func IsModelVisibleToUser(ctx context.Context, aiModel *AIModel, userID uint) (bool, error) {
	if aiModel.VisibilityType != "group" {
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
	if err := DB(ctx).Model(&ModelVisibilityGroup{}).
		Where("ai_model_id = ? AND group_id IN ?", aiModel.ID, userGroupIDs).
		Count(&count).Error; err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgVisibilityQueryModelAssocFailed)
	}
	return count > 0, nil
}

// GetModelVisibilityGroupIDs 批量查询多个模型的可见分组 ID。
// 返回 map[modelID][]groupID。
func GetModelVisibilityGroupIDs(ctx context.Context, modelIDs []uint) (map[uint][]uint, error) {
	if len(modelIDs) == 0 {
		return nil, nil
	}
	var rows []ModelVisibilityGroup
	if err := DB(ctx).Where("ai_model_id IN ?", modelIDs).Find(&rows).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgVisibilityBatchQueryModelAssocFailed)
	}
	result := make(map[uint][]uint)
	for _, r := range rows {
		result[r.AIModelID] = append(result[r.AIModelID], r.GroupID)
	}
	return result, nil
}

// CleanupVisibilityByGroupID 清理某分组被删除后的模型可见性关联。
// 必须传入事务 tx，确保与分组删除在同一事务中执行。
// 内部使用硬删除（Unscoped），因为关联数据不需要保留软删除历史。
func CleanupVisibilityByGroupID(tx *gorm.DB, groupID uint) error {
	if err := tx.Where("group_id = ?", groupID).Delete(&ModelVisibilityGroup{}).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgVisibilityCleanupByGroupFailed)
	}
	return nil
}

// CleanupVisibilityByModelID 清理某模型被删除后的可见性关联。
// 由模型删除逻辑调用。同样建议在事务中调用。
func CleanupVisibilityByModelID(tx *gorm.DB, modelID uint) error {
	if err := tx.Where("ai_model_id = ?", modelID).Delete(&ModelVisibilityGroup{}).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgVisibilityCleanupByModelFailed)
	}
	return nil
}

// IsGroupUsedByModelVisibility 检查用户组是否被模型可见性配置引用。
// 返回 true 表示该用户组被至少一个模型的可见性配置使用，不应被删除。
func IsGroupUsedByModelVisibility(ctx context.Context, groupID uint) (bool, error) {
	var count int64
	if err := DB(ctx).Model(&ModelVisibilityGroup{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgVisibilityCheckGroupUsedFailed)
	}
	return count > 0, nil
}

// GetModelsAssociatedWithGroup 查询与指定用户组关联的模型列表。
// 用于删除用户组前提示用户该组关联了哪些模型。
// 返回关联的模型 ID 列表。
func GetModelsAssociatedWithGroup(ctx context.Context, groupID uint) ([]uint, error) {
	var modelIDs []uint
	if err := DB(ctx).Model(&ModelVisibilityGroup{}).
		Where("group_id = ?", groupID).
		Pluck("ai_model_id", &modelIDs).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgVisibilityQueryModelsByGroupFailed)
	}
	return modelIDs, nil
}
