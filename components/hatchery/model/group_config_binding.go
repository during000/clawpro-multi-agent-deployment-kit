package model

import (
	"context"
	"fmt"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ──────────────────────────────────────────────
// 配置类型常量
// ──────────────────────────────────────────────

const (
	ConfigTypeChannel         = "channel"
	ConfigTypePluginBundle    = "plugin_bundle"
	ConfigTypeMCP             = "mcp"
	ConfigTypeImageType       = "image_type"
	ConfigTypePolicy          = "policy"
	ConfigTypeVPC             = "vpc"
	ConfigTypeCLSCollectScope = "cls_collect_scope"
	ConfigTypeResourcePolicy  = "resource_policy"
)

// GroupAssetConfigTypes 是分组直接选择的资产类型。可见范围继续复用
// skill_visibility_groups / rule_visibility_groups，资产选择统一写入本表。
var GroupAssetConfigTypes = []string{AssetBindingTypeSkill, AssetBindingTypeRule}

// CLS 采集范围 config_key 常量
const (
	// CLSCollectScopeKey 表示该分组的 CLS 采集开关标识。
	// 存储在 group_config_bindings 中：config_type="cls_collect_scope", config_key="enabled"。
	CLSCollectScopeKey = "enabled"
)

// GroupConfigBinding 统一绑定表：分组与资源/配置的绑定关系。
// 兼容两种语义：
//   - 资源型（channel/plugin_bundle/mcp/image_type/vpc/resource_policy）：config_key = 资源 ID 或枚举值，value_json = '{}'
//   - 配置型（policy）：config_key = 策略名，value_json = 配置值 JSON
//
// 分组全局 Token 上限存为 policy / global_token_quota_day；key 历史保留 day，实际周期由 site_configs.global_token_quota_period 决定。
type GroupConfigBinding struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Identifier string    `gorm:"size:191;not null;default:'';uniqueIndex:uk_gcb,priority:1;index:idx_gcb_group,priority:1;index:idx_gcb_resource,priority:1" json:"-"`
	ConfigType string    `gorm:"size:32;not null;uniqueIndex:uk_gcb,priority:2;index:idx_gcb_group,priority:3;index:idx_gcb_resource,priority:2" json:"config_type"`
	ConfigKey  string    `gorm:"size:128;not null;uniqueIndex:uk_gcb,priority:3;index:idx_gcb_resource,priority:3" json:"config_key"`
	GroupID    uint      `gorm:"not null;uniqueIndex:uk_gcb,priority:4;index:idx_gcb_group,priority:2" json:"group_id"`
	ValueJSON  string    `gorm:"type:varchar(4096);not null;default:'{}'" json:"value_json"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ──────────────────────────────────────────────
// 基础 CRUD
// ──────────────────────────────────────────────

// SetAdditiveBindings 全量替换某资源的加法型绑定（先删后插）。
// 当 groupIDs 为空时清空该资源的所有绑定。
func SetAdditiveBindings(tx *gorm.DB, configType string, configKey string, groupIDs []uint) error {
	// 删除旧绑定
	if err := tx.Where("config_type = ? AND config_key = ?", configType, configKey).
		Delete(&GroupConfigBinding{}).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgGroupConfigDeleteOldBindingFailed)
	}
	if len(groupIDs) == 0 {
		return nil
	}
	// 批量插入新绑定
	records := make([]GroupConfigBinding, 0, len(groupIDs))
	for _, gid := range groupIDs {
		records = append(records, GroupConfigBinding{
			ConfigType: configType,
			ConfigKey:  configKey,
			GroupID:    gid,
			ValueJSON:  "{}",
		})
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&records).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgGroupConfigCreateBindingFailed)
	}
	return nil
}

// UpsertPolicyBinding 创建或更新某组的某项策略配置。
func UpsertPolicyBinding(tx *gorm.DB, groupID uint, configKey, valueJSON string) error {
	var existing GroupConfigBinding
	err := tx.Where("config_type = ? AND config_key = ? AND group_id = ?",
		ConfigTypePolicy, configKey, groupID).First(&existing).Error
	if err == nil {
		// 更新
		if err := tx.Model(&existing).Update("value_json", valueJSON).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgGroupConfigUpdatePolicyBindingFailed)
		}
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return hcommon.I18nRichError(err, i18n.MsgGroupConfigQueryPolicyBindingFailed)
	}
	// 创建
	if err := tx.Create(&GroupConfigBinding{
		ConfigType: ConfigTypePolicy,
		ConfigKey:  configKey,
		GroupID:    groupID,
		ValueJSON:  valueJSON,
	}).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgGroupConfigCreatePolicyBindingFailed)
	}
	return nil
}

// DeletePolicyBinding 删除某组的某项策略配置（幂等）。
func DeletePolicyBinding(tx *gorm.DB, groupID uint, configKey string) error {
	return tx.Where("config_type = ? AND config_key = ? AND group_id = ?",
		ConfigTypePolicy, configKey, groupID).Delete(&GroupConfigBinding{}).Error
}

// GetBindingsByGroup 查询某组的所有绑定（可按 configType 过滤）。
func GetBindingsByGroup(ctx context.Context, groupID uint, configType string) ([]GroupConfigBinding, error) {
	q := DB(ctx).Where("group_id = ?", groupID)
	if configType != "" {
		q = q.Where("config_type = ?", configType)
	}
	var bindings []GroupConfigBinding
	err := q.Find(&bindings).Error
	return bindings, err
}

// GetBindingsByResource 查询某资源绑定了哪些组。
func GetBindingsByResource(ctx context.Context, configType, configKey string) ([]GroupConfigBinding, error) {
	var bindings []GroupConfigBinding
	err := DB(ctx).Where("config_type = ? AND config_key = ?", configType, configKey).
		Find(&bindings).Error
	return bindings, err
}

// GetBindingGroupIDs 查询某资源绑定的组 ID 列表。
func GetBindingGroupIDs(ctx context.Context, configType, configKey string) ([]uint, error) {
	var ids []uint
	err := DB(ctx).Model(&GroupConfigBinding{}).
		Where("config_type = ? AND config_key = ?", configType, configKey).
		Pluck("group_id", &ids).Error
	return ids, err
}

// GetBindingsByGroups 批量查询多个组的指定类型绑定。
func GetBindingsByGroups(ctx context.Context, groupIDs []uint, configType string) ([]GroupConfigBinding, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	var bindings []GroupConfigBinding
	err := DB(ctx).Where("config_type = ? AND group_id IN ?", configType, groupIDs).
		Find(&bindings).Error
	return bindings, err
}

// GetPolicyBindingsByGroups 批量查询多个组的策略配置（按 config_key 过滤）。
func GetPolicyBindingsByGroups(ctx context.Context, groupIDs []uint, configKey string) ([]GroupConfigBinding, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	var bindings []GroupConfigBinding
	err := DB(ctx).Where("config_type = ? AND config_key = ? AND group_id IN ?",
		ConfigTypePolicy, configKey, groupIDs).Find(&bindings).Error
	return bindings, err
}

// GetAllPolicyBindingsByGroups 批量查询多个组的所有策略配置。
func GetAllPolicyBindingsByGroups(ctx context.Context, groupIDs []uint) ([]GroupConfigBinding, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	var bindings []GroupConfigBinding
	err := DB(ctx).Where("config_type = ? AND group_id IN ?", ConfigTypePolicy, groupIDs).
		Find(&bindings).Error
	return bindings, err
}

// GetResourceVisibilityGroupIDsByUint 批量查询多个资源的可见分组 ID（uint 版本）。
// 用于按资源 ID 批量查询可见分组（如 channel ID、model ID等）。
// 返回 map[resourceID][]groupID。
// 与 GetModelVisibilityGroupIDs 类似，但从 GroupConfigBinding 表查询。
func GetResourceVisibilityGroupIDsByUint(ctx context.Context, configType string, resourceIDs []uint) (map[uint][]uint, error) {
	if len(resourceIDs) == 0 {
		return make(map[uint][]uint), nil
	}

	// 转换 uint 为 string 进行查询（config_key 存储为字符串）
	resourceKeys := make([]string, len(resourceIDs))
	for i, id := range resourceIDs {
		resourceKeys[i] = fmt.Sprintf("%d", id)
	}

	var bindings []GroupConfigBinding
	if err := DB(ctx).Where("config_type = ? AND config_key IN ?", configType, resourceKeys).
		Find(&bindings).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgGroupConfigBatchQueryVisibilityFailed, configType)
	}

	result := make(map[uint][]uint)
	for _, b := range bindings {
		var id uint
		fmt.Sscanf(b.ConfigKey, "%d", &id)
		result[id] = append(result[id], b.GroupID)
	}
	return result, nil
}

// IsGroupUsedByConfigBindings 检查用户组是否被非资产配置绑定引用（删除前校验）。
// 资产绑定不阻塞分组删除，删除事务会通过 CleanupConfigBindingsByGroupID 清理它们。
func IsGroupUsedByConfigBindings(ctx context.Context, groupID uint) (bool, error) {
	var count int64
	err := DB(ctx).Model(&GroupConfigBinding{}).Where("group_id = ? AND config_type NOT IN ?", groupID, GroupAssetConfigTypes).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CleanupConfigBindingsByGroupID 清理某分组被删除后的绑定记录。
func CleanupConfigBindingsByGroupID(tx *gorm.DB, groupID uint) error {
	return tx.Where("group_id = ?", groupID).Delete(&GroupConfigBinding{}).Error
}

// GetRestrictedImageTypes 查询所有被限制（有绑定行）的 agent_type 列表。
func GetRestrictedImageTypes(ctx context.Context) ([]string, error) {
	var keys []string
	err := DB(ctx).Model(&GroupConfigBinding{}).
		Where("config_type = ?", ConfigTypeImageType).
		Distinct().Pluck("config_key", &keys).Error
	return keys, err
}

// GetVisibleImageTypesByGroups 查询指定组可见的 agent_type 列表。
func GetVisibleImageTypesByGroups(ctx context.Context, groupIDs []uint) ([]string, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	var keys []string
	err := DB(ctx).Model(&GroupConfigBinding{}).
		Where("config_type = ? AND group_id IN ?", ConfigTypeImageType, groupIDs).
		Distinct().Pluck("config_key", &keys).Error
	return keys, err
}
