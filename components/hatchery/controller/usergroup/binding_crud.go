package usergroup

import (
	"context"
	"strconv"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// ──────────────────────────────────────────────
// 绑定 CRUD 封装（供 controller 层调用）
// ──────────────────────────────────────────────

// SetVisibility 设置加法型资源的应用范围。
// visibilityType="all" 时清空绑定；"group" 时全量替换。
// 调用方需传入事务 tx 或 model.DB。
func SetVisibility(tx *gorm.DB, configType string, resourceID uint, visibilityType string, groupIDs []uint) error {
	if !IsValidConfigType(configType) {
		return hcommon.I18nError(i18n.MsgBindingUnsupportedConfigType, configType)
	}
	meta := ConfigTypes[configType]
	if meta.Cardinality != CardinalityAdditive {
		return hcommon.I18nError(i18n.MsgBindingConfigTypeNotAdditive, configType)
	}

	configKey := strconv.FormatUint(uint64(resourceID), 10)

	if visibilityType == VisibilityAll {
		groupIDs = nil
	}
	return model.SetAdditiveBindings(tx, configType, configKey, groupIDs)
}

// SetImageTypeVisibility 设置镜像类型的应用范围。
// visibilityType="all" 时删除该 agent_type 的所有绑定行；"group" 时全量替换。
func SetImageTypeVisibility(tx *gorm.DB, agentType string, visibilityType string, groupIDs []uint) error {
	if visibilityType == VisibilityAll {
		groupIDs = nil
	}
	return model.SetAdditiveBindings(tx, ConfigTypeImageType, agentType, groupIDs)
}

// SetPolicy 为某组设置（或更新）一项策略配置。
func SetPolicy(tx *gorm.DB, groupID uint, configKey, valueJSON string) error {
	if !IsValidPolicyKey(configKey) {
		return hcommon.I18nError(i18n.MsgBindingUnsupportedPolicyKey, configKey)
	}
	return model.UpsertPolicyBinding(tx, groupID, configKey, valueJSON)
}

// DeletePolicy 删除某组的某项策略配置（幂等）。
func DeletePolicy(tx *gorm.DB, groupID uint, configKey string) error {
	if !IsValidPolicyKey(configKey) {
		return hcommon.I18nError(i18n.MsgBindingUnsupportedPolicyKey, configKey)
	}
	if dbErr := model.DeletePolicyBinding(tx, groupID, configKey); dbErr != nil {
		return hcommon.I18nRichError(dbErr, i18n.MsgDeletePolicyBindingFailed)
	}
	return nil
}

// GetResourceBindingGroups 查询某资源绑定了哪些组（含组名）。
// 用于管理端展示"应用范围"和前端置灰逻辑。
func GetResourceBindingGroups(ctx context.Context, configType, configKey string) ([]GroupInfo, error) {
	bindings, err := model.GetBindingsByResource(ctx, configType, configKey)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgBindingQueryByResourceFailed)
	}
	if len(bindings) == 0 {
		return []GroupInfo{}, nil
	}

	// 批量获取组名
	groupIDs := make([]uint, 0, len(bindings))
	for _, b := range bindings {
		groupIDs = append(groupIDs, b.GroupID)
	}
	groups, err := model.GetGroupsByIDs(ctx, groupIDs)
	if err != nil {
		return nil, err
	}
	nameMap := make(map[uint]string, len(groups))
	for _, g := range groups {
		nameMap[g.ID] = g.Name
	}

	result := make([]GroupInfo, 0, len(bindings))
	for _, b := range bindings {
		result = append(result, GroupInfo{
			GroupID:   b.GroupID,
			GroupName: nameMap[b.GroupID],
		})
	}
	return result, nil
}

// GroupInfo 组简要信息
type GroupInfo struct {
	GroupID   uint   `json:"group_id"`
	GroupName string `json:"group_name"`
}

// ValidateGroupIDs 校验所有 group_id 是否存在。
func ValidateGroupIDs(ctx context.Context, groupIDs []uint) error {
	if len(groupIDs) == 0 {
		return nil
	}
	// 去重后再比较
	seen := make(map[uint]struct{}, len(groupIDs))
	for _, id := range groupIDs {
		seen[id] = struct{}{}
	}
	uniqueIDs := make([]uint, 0, len(seen))
	for id := range seen {
		uniqueIDs = append(uniqueIDs, id)
	}
	groups, err := model.GetGroupsByIDs(ctx, uniqueIDs)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	if len(groups) != len(uniqueIDs) {
		return hcommon.I18nError(i18n.MsgBindingInvalidGroupIDs)
	}
	return nil
}
