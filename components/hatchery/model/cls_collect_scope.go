package model

import (
	"context"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ──────────────────────────────────────────────
// CLS 采集范围：基于 group_config_bindings 表
// config_type = "cls_collect_scope", config_key = "enabled"
//
// 删除操作采用硬删除，与其他 config_type（如 channel）保持一致。
// ──────────────────────────────────────────────

// GetCLSCollectScopeGroupIDs 查询当前 CLS 开启范围的分组 ID 列表。
// 返回空切片表示未配置分组范围（全量模式）。
func GetCLSCollectScopeGroupIDs(ctx context.Context) ([]uint, error) {
	var ids []uint
	err := DB(ctx).Model(&GroupConfigBinding{}).
		Where("config_type = ? AND config_key = ?", ConfigTypeCLSCollectScope, CLSCollectScopeKey).
		Pluck("group_id", &ids).Error
	return ids, err
}

// SetCLSCollectScope 全量替换 CLS 采集范围（先删后插）。
// groupIDs 为空表示清空范围（全量模式）。
// 复用 SetAdditiveBindings 的硬删除+批量插入逻辑。
//
// 注意：生产代码应使用 DiffAndSetCLSCollectScope，它在同一事务中计算 diff 并更新，
// 能正确触发实例标记变更。此函数仅供测试使用。
func SetCLSCollectScope(ctx context.Context, groupIDs []uint) error {
	return DB(ctx).Transaction(func(tx *gorm.DB) error {
		if rerr := SetAdditiveBindings(tx, ConfigTypeCLSCollectScope, CLSCollectScopeKey, groupIDs); rerr != nil {
			return rerr
		}
		return nil
	})
}

// AddCLSCollectScopeGroups 增量添加分组到 CLS 采集范围（幂等）。
// 已存在的记录不受影响，利用 OnConflict DoNothing 实现幂等。
// 输入会自动去重，避免同一批次中重复记录导致的潜在问题。
func AddCLSCollectScopeGroups(ctx context.Context, groupIDs []uint) error {
	if len(groupIDs) == 0 {
		return nil
	}
	// 去重
	seen := make(map[uint]struct{}, len(groupIDs))
	deduped := make([]uint, 0, len(groupIDs))
	for _, gid := range groupIDs {
		if _, ok := seen[gid]; !ok {
			seen[gid] = struct{}{}
			deduped = append(deduped, gid)
		}
	}
	return DB(ctx).Transaction(func(tx *gorm.DB) error {
		records := make([]GroupConfigBinding, 0, len(deduped))
		for _, gid := range deduped {
			records = append(records, GroupConfigBinding{
				ConfigType: ConfigTypeCLSCollectScope,
				ConfigKey:  CLSCollectScopeKey,
				GroupID:    gid,
				ValueJSON:  "{}",
			})
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&records).Error
	})
}

// RemoveCLSCollectScopeGroups 从 CLS 采集范围中移除指定分组（硬删除）。
func RemoveCLSCollectScopeGroups(ctx context.Context, groupIDs []uint) error {
	if len(groupIDs) == 0 {
		return nil
	}
	return DB(ctx).Where("config_type = ? AND config_key = ? AND group_id IN ?",
		ConfigTypeCLSCollectScope, CLSCollectScopeKey, groupIDs).
		Delete(&GroupConfigBinding{}).Error
}

// ClearCLSCollectScope 清空 CLS 采集范围（硬删除所有记录）。
func ClearCLSCollectScope(ctx context.Context) error {
	return DB(ctx).Where("config_type = ? AND config_key = ?",
		ConfigTypeCLSCollectScope, CLSCollectScopeKey).
		Delete(&GroupConfigBinding{}).Error
}

// DiffCLSCollectScope 计算新旧采集范围的差异（只读，不修改数据）。
// 返回新增分组 ID 列表和移除分组 ID 列表。输入会自动去重。
//
// Deprecated: 此函数在事务外读取，存在 TOCTOU 竞态风险。
// 生产代码应使用 DiffAndSetCLSCollectScope，它在同一事务中原子地计算 diff 并更新。
// 此函数仅供测试中的只读 diff 验证使用。
func DiffCLSCollectScope(ctx context.Context, newGroupIDs []uint) (added []uint, removed []uint, err error) {
	oldGroupIDs, err := GetCLSCollectScopeGroupIDs(ctx)
	if err != nil {
		return nil, nil, hcommon.I18nRichError(err, i18n.MsgCLSQueryExistingScopeFailed)
	}

	oldSet := make(map[uint]struct{}, len(oldGroupIDs))
	for _, id := range oldGroupIDs {
		oldSet[id] = struct{}{}
	}
	newSet := make(map[uint]struct{}, len(newGroupIDs))
	for _, id := range newGroupIDs {
		newSet[id] = struct{}{}
	}

	// 遍历 newSet 而非 newGroupIDs，避免输入重复导致 added 中出现重复 ID
	for id := range newSet {
		if _, ok := oldSet[id]; !ok {
			added = append(added, id)
		}
	}
	for _, id := range oldGroupIDs {
		if _, ok := newSet[id]; !ok {
			removed = append(removed, id)
		}
	}
	return added, removed, nil
}

// DiffAndSetCLSCollectScope 在同一事务中计算 diff 并更新采集范围，避免并发竞态。
// 返回新增分组 ID 列表和移除分组 ID 列表。输入会自动去重。
// 事务内使用 SELECT ... FOR UPDATE 锁定读取行，防止并发覆盖导致 diff 结果与实际不一致。
func DiffAndSetCLSCollectScope(ctx context.Context, newGroupIDs []uint) (added []uint, removed []uint, err error) {
	err = DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 在事务内读取旧数据，加行锁防止并发竞态
		var oldIDs []uint
		if e := tx.Model(&GroupConfigBinding{}).
			Where("config_type = ? AND config_key = ?", ConfigTypeCLSCollectScope, CLSCollectScopeKey).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Pluck("group_id", &oldIDs).Error; e != nil {
			return hcommon.I18nRichError(e, i18n.MsgCLSQueryExistingScopeFailed)
		}

		oldSet := make(map[uint]struct{}, len(oldIDs))
		for _, id := range oldIDs {
			oldSet[id] = struct{}{}
		}
		newSet := make(map[uint]struct{}, len(newGroupIDs))
		for _, id := range newGroupIDs {
			newSet[id] = struct{}{}
		}

		for id := range newSet {
			if _, ok := oldSet[id]; !ok {
				added = append(added, id)
			}
		}
		for _, id := range oldIDs {
			if _, ok := newSet[id]; !ok {
				removed = append(removed, id)
			}
		}

		// 去重后传给 SetAdditiveBindings，避免构建冗余 records
		dedupedIDs := make([]uint, 0, len(newSet))
		for id := range newSet {
			dedupedIDs = append(dedupedIDs, id)
		}

		// 在同一事务内执行全量替换
		if rerr := SetAdditiveBindings(tx, ConfigTypeCLSCollectScope, CLSCollectScopeKey, dedupedIDs); rerr != nil {
			return rerr
		}
		return nil
	})
	// 事务失败时清零闭包变量，防止调用方误用脏数据
	if err != nil {
		added, removed = nil, nil
	}
	return
}

// GetCLSCollectScopeCVMInstanceIDs 查询 CLS 采集范围内所有分组对应的 CVM 实例 ID。
// 会通过 group_closure 展开子孙分组。
// 返回值 scopeSet 标识 scope 是否已配置（true=分组模式，false=全量模式），
// 避免 nil 与空切片的脆弱区分。
//
// 实例归属通过 instances.group_id 直接关联（实例所属分组在 scope 子树中即命中）。
func GetCLSCollectScopeCVMInstanceIDs(ctx context.Context) (ids []string, scopeSet bool, err error) {
	groupIDs, err := GetCLSCollectScopeGroupIDs(ctx)
	if err != nil {
		return nil, false, err
	}
	if len(groupIDs) == 0 {
		return nil, false, nil // scope 未配置 = 全量模式
	}

	// 展开所有子孙分组
	allGroupIDs, err := ExpandGroupIDsWithDescendants(ctx, groupIDs)
	if err != nil {
		return nil, true, hcommon.I18nRichError(err, i18n.MsgCLSDescendantExpandFailed)
	}
	if len(allGroupIDs) == 0 {
		return []string{}, true, nil
	}

	instanceIDs, err := GetCVMInstanceIDsInGroups(ctx, allGroupIDs)
	return instanceIDs, true, err
}

// GetCVMInstanceIDsInGroups 查询指定分组关联的 CVM 实例 ID 列表（通过 instances.group_id 直接关联，已去重）。
func GetCVMInstanceIDsInGroups(ctx context.Context, groupIDs []uint) ([]string, error) {
	if len(groupIDs) == 0 {
		return []string{}, nil
	}

	var instanceIDs []string
	if err := DB(ctx).Model(&Instance{}).
		Select("instance_id").
		Where("group_id IN ? AND instance_id != ''", groupIDs).
		Pluck("instance_id", &instanceIDs).Error; err != nil {
		return nil, err
	}
	if len(instanceIDs) == 0 {
		return []string{}, nil
	}

	return instanceIDs, nil
}

// ExpandGroupIDsWithDescendants 对一组分组 ID，通过 closure 表展开所有子孙（含自身），去重后返回。
func ExpandGroupIDsWithDescendants(ctx context.Context, groupIDs []uint) ([]uint, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	// 只查 descendant_id，避免加载完整的 GroupClosure 结构体
	var descendantIDs []uint
	if err := DB(ctx).Model(&GroupClosure{}).
		Where("ancestor_id IN ?", groupIDs).
		Pluck("descendant_id", &descendantIDs).Error; err != nil {
		return nil, err
	}
	seen := make(map[uint]struct{}, len(descendantIDs)+len(groupIDs))
	for _, id := range descendantIDs {
		seen[id] = struct{}{}
	}
	// 确保原始 groupIDs 也包含（即使 closure 表暂无数据）
	for _, id := range groupIDs {
		seen[id] = struct{}{}
	}
	result := make([]uint, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	return result, nil
}

// IsInstanceGroupInCLSCollectScope 判断指定实例的 group_id 是否命中 CLS 采集范围。
// 根据 CLSScopeMode 判断：
//   - CLSEnabled != 1 时，所有实例不命中
//   - CLSScopeMode="all" 时，所有实例命中
//   - CLSScopeMode="group" 时，仅 scope 分组（含子孙）内的实例命中
//
// groupID=0 表示实例未指定分组，按分组模式时不命中。
func IsInstanceGroupInCLSCollectScope(ctx context.Context, groupID uint) (bool, error) {
	config := GetSiteConfig(ctx)
	if config.CLSEnabled != 1 {
		return false, nil
	}
	isGroupMode := config.CLSScopeMode == "group"

	scopeGroupIDs, err := GetCLSCollectScopeGroupIDs(ctx)
	if err != nil {
		return false, err
	}

	if len(scopeGroupIDs) == 0 {
		if isGroupMode {
			// 分组模式下 scope 为空 = 不命中任何实例
			return false, nil
		}
		// 全量模式，所有实例命中
		return true, nil
	}

	// 实例未指定分组，在分组模式下不命中
	if groupID == 0 {
		return false, nil
	}

	// 展开子孙
	allGroupIDs, err := ExpandGroupIDsWithDescendants(ctx, scopeGroupIDs)
	if err != nil {
		return false, err
	}
	if len(allGroupIDs) == 0 {
		return false, nil
	}

	// 判断实例的 group_id 是否在 scope 子树中
	for _, id := range allGroupIDs {
		if id == groupID {
			return true, nil
		}
	}
	return false, nil
}
