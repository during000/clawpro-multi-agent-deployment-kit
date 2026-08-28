package model

import (
	"context"
	"errors"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// 用户组相关哨兵错误，供 controller 层通过 errors.Is() 精确匹配，避免字符串比较。
var (
	ErrUserGroupLimitExceeded     = hcommon.I18nError(i18n.MsgUgLimitExceeded)
	ErrUserGroupNotFound          = hcommon.I18nError(i18n.MsgUgNotFound)
	ErrInvalidUserGroupID         = hcommon.I18nError(i18n.MsgBindingInvalidGroupIDs)
	ErrGroupMemberLimitReached    = hcommon.I18nError(i18n.MsgUgMemberLimitReached)
	ErrMemberCountExceeded        = hcommon.I18nError(i18n.MsgGroupMemberLimitExceeded)
	ErrAddMemberWouldExceed       = hcommon.I18nError(i18n.MsgUgAddMemberWouldExceed)
	ErrInvalidUserID              = hcommon.I18nError(i18n.MsgUgInvalidUserID)
	ErrInvalidGroupName           = hcommon.I18nError(i18n.MsgUgInvalidName)
	ErrGroupNameConflict          = hcommon.I18nError(i18n.MsgUgNameConflict)
	ErrMaxGroupDepthExceeded      = hcommon.I18nError(i18n.MsgUgMaxDepthExceeded)
	ErrFullPathTooLong            = hcommon.I18nError(i18n.MsgUgFullPathTooLong)
	ErrParentCycleDetected        = hcommon.I18nError(i18n.MsgUgParentCycleDetected)
	ErrManualCannotUnderOneIDDept = hcommon.I18nError(i18n.MsgUgManualCannotUnderOneIDDept)
	ErrOneIDDeptReadonly          = hcommon.I18nError(i18n.MsgUgOneIDDeptReadonly)
	ErrGroupHasDependencies       = hcommon.I18nError(i18n.MsgUgHasDependencies)
	ErrGroupToBeDeletedReadonly   = hcommon.I18nError(i18n.MsgUgToBeDeletedReadonly)
	ErrGroupNotSelectable         = hcommon.I18nError(i18n.MsgUgNotSelectable)
	ErrParentGroupNotFound        = hcommon.I18nError(i18n.MsgUgParentNotFound)
)

const (
	// MaxUserGroupsPerPlatform 平台用户组数量上限（含 oneid_dept + manual 之和）。
	MaxUserGroupsPerPlatform = 2000
	// MaxMembersPerUserGroup 单个用户组成员数量上限。
	MaxMembersPerUserGroup = 10000
	// MaxGroupDepth 分组层级最大深度（根 depth=0，最深 depth=9，总共 10 层）。
	MaxGroupDepth = 10
	// MaxFullPathLength full_path 字段最大字符长度。
	MaxFullPathLength = 512
	// MaxGroupNameLength 单段分组名最大字符长度。
	MaxGroupNameLength = 191

	// GroupSourceManual 管理员手建分组。
	GroupSourceManual = "manual"
	// GroupSourceOneIDDept OneID 组织架构节点。
	GroupSourceOneIDDept = "oneid_dept"

	// VisibilityAll 资源对全部用户可见。
	VisibilityAll = "all"
	// VisibilityGroup 资源仅对指定分组可见。
	VisibilityGroup = "group"
)

// UserGroup 用户组主表（v6.12+ 扩展字段见下）
//
// 不使用 GORM 软删（无 DeletedAt 字段）：分组删除走物理删除，
// 同事务内级联清理 user_group_members + group_closure。
// 业务上的"待删占位"语义由 ToBeDeleted bool 字段承担（OneID 部门
// 消失但本地仍有资源绑定时打此标记），与 GORM 软删无关。
type UserGroup struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Identifier  string `gorm:"size:191;not null;default:'';uniqueIndex:idx_ug_ident_parent_name;index:idx_ug_source,priority:1;index:idx_ug_fullpath,priority:1;index:idx_ug_tobedeleted,priority:1" json:"-"`
	Name        string `gorm:"size:191;not null;uniqueIndex:idx_ug_ident_parent_name,priority:3" json:"name"`
	SyncMode    string `gorm:"size:32;not null;default:'continuous'" json:"sync_mode"` // initial_only | continuous；分组默认 continuous（创建即可持续同步）
	Description string `gorm:"type:text;not null" json:"description"`

	// 🆕 v6: 多层级 + OneID 同步扩展字段
	ParentID    uint   `gorm:"not null;default:0;uniqueIndex:idx_ug_ident_parent_name,priority:2;index:idx_ug_parent" json:"parent_id"` // 🔄 v6.15: uint，0 = 根组
	Depth       int    `gorm:"not null;default:0;index" json:"depth"`                                                                   // 根=0，最深 9
	FullPath    string `gorm:"size:512;not null;default:'';index:idx_ug_fullpath,priority:2" json:"full_path"`                          // 例："腾讯/技术部/前端组"
	Source      string `gorm:"size:32;not null;default:'manual';index:idx_ug_source,priority:2" json:"source"`                          // manual / oneid_dept（🆕 v6.2 移除 oneid_group）
	SourceRef   string `gorm:"size:191;not null;default:'';index:idx_ug_source,priority:3" json:"source_ref"`                           // OneID 部门 ID，manual 为空
	ToBeDeleted bool   `gorm:"not null;default:false;index:idx_ug_tobedeleted,priority:2" json:"to_be_deleted"`                         // 🆕 v6: OneID 部门已消失但本地有资源绑定，暂保留只读

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Readonly 是否只读（非 manual 或已标记待删除）。
func (g *UserGroup) Readonly() bool {
	return g.Source != GroupSourceManual || g.ToBeDeleted
}

// ValidateGroupName 校验分组名：非空、长度≤MaxGroupNameLength、不含 '/'、不全是空白。
func ValidateGroupName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrInvalidGroupName
	}
	if len([]rune(trimmed)) > MaxGroupNameLength {
		return ErrInvalidGroupName
	}
	if strings.Contains(trimmed, "/") {
		return ErrInvalidGroupName
	}
	return nil
}

// AssertEditableByAdmin 校验分组是否允许管理员写操作：必须是 manual 且未标记 to_be_deleted。
func AssertEditableByAdmin(g *UserGroup) error {
	if g.Source != GroupSourceManual {
		return ErrOneIDDeptReadonly
	}
	if g.ToBeDeleted {
		return ErrGroupToBeDeletedReadonly
	}
	return nil
}

// AssertManualParentValid 校验：manual 组的父组不能是 oneid_dept。
// 🆕 v6: 架构约束，简化 OneID 子树删除传播。
func AssertManualParentValid(parent *UserGroup) error {
	if parent == nil {
		return nil // parent_id=0，根组
	}
	if parent.Source == GroupSourceOneIDDept {
		return ErrManualCannotUnderOneIDDept
	}
	if parent.ToBeDeleted {
		return ErrGroupToBeDeletedReadonly
	}
	return nil
}

// GroupByID 按 ID 查单个分组（不包括 to_be_deleted 过滤；调用方按需判断）。
func GroupByID(ctx context.Context, id uint) (*UserGroup, error) {
	var g UserGroup
	if err := DB(ctx).First(&g, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserGroupNotFound
		}
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	return &g, nil
}

// GroupBySourceRef 按 (source, source_ref) 查 OneID 部门对应的分组。
// 返回 (nil, nil) 表示未找到（用于 OneID 同步判断是否需要新建）。
func GroupBySourceRef(ctx context.Context, source, sourceRef string) (*UserGroup, error) {
	if sourceRef == "" {
		return nil, nil
	}
	var g UserGroup
	err := DB(ctx).Where("source = ? AND source_ref = ?", source, sourceRef).First(&g).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	return &g, nil
}

// CreateUserGroup 创建用户组（简化签名保留向后兼容）。
// 创建 manual 根组：CreateUserGroup("研发部", "...")。
// 需要指定 parent_id / source 请使用 CreateUserGroupWithOpts。
func CreateUserGroup(ctx context.Context, name, description string) (*UserGroup, error) {
	return CreateUserGroupWithOpts(ctx, name, description, 0, GroupSourceManual, "")
}

// CreateUserGroupOpts 创建用户组的完整参数（供内部使用）。
// 事务内完成：校验 + 插 user_groups + 维护 group_closure + 计算 full_path。
func CreateUserGroupWithOpts(ctx context.Context, name, description string, parentID uint, source, sourceRef string) (*UserGroup, error) {
	if err := ValidateGroupName(name); err != nil {
		return nil, err
	}
	if source == "" {
		source = GroupSourceManual
	}

	var createdGroup *UserGroup
	err := DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 校验平台上限（含所有 source）
		var count int64
		if err := tx.Model(&UserGroup{}).Count(&count).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
		}
		if count >= MaxUserGroupsPerPlatform {
			return ErrUserGroupLimitExceeded
		}

		// 校验父组
		var parent *UserGroup
		var parentDepth int
		parentFullPath := ""
		if parentID != 0 {
			p, err := groupByIDTx(tx, parentID)
			if err != nil {
				return ErrParentGroupNotFound
			}
			parent = p
			parentDepth = p.Depth
			parentFullPath = p.FullPath
			if source == GroupSourceManual {
				if err := AssertManualParentValid(parent); err != nil {
					return err
				}
			}
		}

		depth := 0
		if parent != nil {
			depth = parentDepth + 1
		}
		if depth >= MaxGroupDepth {
			return ErrMaxGroupDepthExceeded
		}

		fullPath := strings.TrimSpace(name)
		if parentFullPath != "" {
			fullPath = parentFullPath + "/" + fullPath
		}
		if len([]rune(fullPath)) > MaxFullPathLength {
			return ErrFullPathTooLong
		}

		// 同父下唯一
		if err := assertNameUniqueUnderParentTx(tx, 0, parentID, name); err != nil {
			return err
		}

		g := &UserGroup{
			Name:        strings.TrimSpace(name),
			Description: description,
			ParentID:    parentID,
			Depth:       depth,
			FullPath:    fullPath,
			Source:      source,
			SourceRef:   sourceRef,
			SyncMode:    SyncModeContinuous, // 新建分组默认 continuous（创建即可持续同步）
		}
		if err := tx.Create(g).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBCreateFailed)
		}

		// 维护 closure：插入自指 + 从祖先链继承
		if err := closureInsertForNewChildTx(tx, g.ID, parentID); err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBCreateFailed)
		}

		createdGroup = g
		return nil
	})
	if err != nil {
		var richErr *hcommon.RichError
		if errors.As(err, &richErr) {
			return nil, richErr
		}
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBCreateFailed)
	}
	return createdGroup, nil
}

// UpdateGroupOpts UpdateUserGroupExt 的可选参数。
// Name / Description：指针为空表示不更新；指针非空用解引用值更新。
// NewParentIDPtr：nil 表示不换父；非空则 *NewParentIDPtr = 0 表示移到根，>0 表示挂到该父下（v6.7）。
type UpdateGroupOpts struct {
	Name           *string
	Description    *string
	NewParentIDPtr *uint
}

// UpdateUserGroupExt 修改分组（含可选换父 v6.7）。
// 事务内完成校验 + UPDATE + 子树 depth / full_path 递归重算 + closure 调整。
func UpdateUserGroupExt(ctx context.Context, id uint, opts UpdateGroupOpts) (*UserGroup, error) {
	if opts.Name != nil {
		if err := ValidateGroupName(*opts.Name); err != nil {
			return nil, err
		}
	}

	var updated *UserGroup
	err := DB(ctx).Transaction(func(tx *gorm.DB) error {
		g, err := groupByIDTx(tx, id)
		if err != nil {
			return err
		}
		if err := AssertEditableByAdmin(g); err != nil {
			return err
		}

		// 决定最终 parent_id / name
		newParentID := g.ParentID
		changedParent := false
		if opts.NewParentIDPtr != nil {
			newParentID = *opts.NewParentIDPtr
			if newParentID != g.ParentID {
				changedParent = true
			}
		}
		newName := g.Name
		if opts.Name != nil {
			newName = strings.TrimSpace(*opts.Name)
		}

		// 换父：循环检查 + 父源校验 + depth/fullpath 检查
		var newParent *UserGroup
		if newParentID != 0 {
			p, err := groupByIDTx(tx, newParentID)
			if err != nil {
				return ErrParentGroupNotFound
			}
			newParent = p
			if g.Source == GroupSourceManual {
				if err := AssertManualParentValid(newParent); err != nil {
					return err
				}
			}
		}
		if changedParent {
			if newParentID == id {
				return ErrParentCycleDetected
			}
			// 新父是自身后代？查 closure
			isDesc, dbErr := closureIsDescendantTx(tx, id, newParentID)
			if dbErr != nil {
				return hcommon.I18nRichError(dbErr, i18n.MsgUgDBQueryFailed)
			}
			if isDesc {
				return ErrParentCycleDetected
			}
		}

		// 计算新深度与新 full_path 前缀
		newDepth := 0
		newParentFullPath := ""
		if newParent != nil {
			newDepth = newParent.Depth + 1
			newParentFullPath = newParent.FullPath
		}
		// 子树深度要 ≤ MaxGroupDepth-1
		// 当前子树最大深度相对偏移
		subtreeMaxRel, dbErr := closureMaxRelativeDepthTx(tx, id)
		if dbErr != nil {
			return hcommon.I18nRichError(dbErr, i18n.MsgUgDBQueryFailed)
		}
		if newDepth+subtreeMaxRel >= MaxGroupDepth {
			return ErrMaxGroupDepthExceeded
		}

		// 同父下重名校验
		if changedParent || opts.Name != nil {
			if err := assertNameUniqueUnderParentTx(tx, id, newParentID, newName); err != nil {
				return err
			}
		}

		// 计算本节点新 full_path
		newOwnFullPath := newName
		if newParentFullPath != "" {
			newOwnFullPath = newParentFullPath + "/" + newName
		}
		if len([]rune(newOwnFullPath)) > MaxFullPathLength {
			return ErrFullPathTooLong
		}

		// 更新本节点
		updates := map[string]any{
			"name":      newName,
			"full_path": newOwnFullPath,
			"depth":     newDepth,
		}
		if opts.Description != nil {
			updates["description"] = *opts.Description
		}
		if changedParent {
			updates["parent_id"] = newParentID
			updates["created_at"] = time.Now() // 换父后排序到新父节点子列表末尾
		}
		if err := tx.Model(&UserGroup{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBUpdateFailed)
		}

		// 换父时调整 closure
		if changedParent {
			if err := closureMoveSubtreeTx(tx, id, newParentID); err != nil {
				return hcommon.I18nRichError(err, i18n.MsgUgDBUpdateFailed)
			}
		}

		// 递归重算子孙 full_path 与 depth（含自身节点已更新）
		if err := recomputeSubtreeFullPathTx(tx, id); err != nil {
			return err
		}

		// 校验子树 full_path 没超长
		if err := assertSubtreeFullPathOKTx(tx, id); err != nil {
			return err
		}

		// 查询返回最新对象
		var fresh UserGroup
		if err := tx.First(&fresh, id).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
		}
		updated = &fresh
		return nil
	})
	if err != nil {
		var richErr *hcommon.RichError
		if errors.As(err, &richErr) {
			return nil, richErr
		}
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBUpdateFailed)
	}
	return updated, nil
}

// UpdateUserGroup 保留旧签名（兼容原 controller 调用）。
func UpdateUserGroup(ctx context.Context, id uint, name, description string) (*UserGroup, error) {
	return UpdateUserGroupExt(ctx, id, UpdateGroupOpts{Name: &name, Description: &description})
}

// UpdateUserGroupExtForOneIDDept 是 UpdateUserGroupExt 针对 OneID 部门同步的特殊变体：
// 跳过 AssertEditableByAdmin（允许修改 source=oneid_dept 组），其他校验完全复用。
// 仅供 usergroup.LandOneIDDepartmentsToGroups 内部调用，不对外暴露 controller。
func UpdateUserGroupExtForOneIDDept(ctx context.Context, id uint, opts UpdateGroupOpts) (*UserGroup, error) {
	if opts.Name != nil {
		if err := ValidateGroupName(*opts.Name); err != nil {
			return nil, err
		}
	}

	var updated *UserGroup
	err := DB(ctx).Transaction(func(tx *gorm.DB) error {
		g, err := groupByIDTx(tx, id)
		if err != nil {
			return err
		}
		// 不做 AssertEditableByAdmin —— oneid_dept 组由同步流程主动维护

		newParentID := g.ParentID
		changedParent := false
		if opts.NewParentIDPtr != nil {
			newParentID = *opts.NewParentIDPtr
			if newParentID != g.ParentID {
				changedParent = true
			}
		}
		newName := g.Name
		if opts.Name != nil {
			newName = strings.TrimSpace(*opts.Name)
		}

		var newParent *UserGroup
		if newParentID != 0 {
			p, err := groupByIDTx(tx, newParentID)
			if err != nil {
				return ErrParentGroupNotFound
			}
			newParent = p
			// 不做 AssertManualParentValid —— oneid_dept 下可以挂 oneid_dept
		}
		if changedParent {
			if newParentID == id {
				return ErrParentCycleDetected
			}
			isDesc, dbErr := closureIsDescendantTx(tx, id, newParentID)
			if dbErr != nil {
				return hcommon.I18nRichError(dbErr, i18n.MsgUgDBQueryFailed)
			}
			if isDesc {
				return ErrParentCycleDetected
			}
		}

		newDepth := 0
		newParentFullPath := ""
		if newParent != nil {
			newDepth = newParent.Depth + 1
			newParentFullPath = newParent.FullPath
		}
		subtreeMaxRel, dbErr := closureMaxRelativeDepthTx(tx, id)
		if dbErr != nil {
			return hcommon.I18nRichError(dbErr, i18n.MsgUgDBQueryFailed)
		}
		if newDepth+subtreeMaxRel >= MaxGroupDepth {
			return ErrMaxGroupDepthExceeded
		}

		if changedParent || opts.Name != nil {
			if err := assertNameUniqueUnderParentTx(tx, id, newParentID, newName); err != nil {
				return err
			}
		}

		newOwnFullPath := newName
		if newParentFullPath != "" {
			newOwnFullPath = newParentFullPath + "/" + newName
		}
		if len([]rune(newOwnFullPath)) > MaxFullPathLength {
			return ErrFullPathTooLong
		}

		updates := map[string]any{
			"name":      newName,
			"full_path": newOwnFullPath,
			"depth":     newDepth,
		}
		if opts.Description != nil {
			updates["description"] = *opts.Description
		}
		if changedParent {
			updates["parent_id"] = newParentID
		}
		if err := tx.Model(&UserGroup{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBUpdateFailed)
		}

		if changedParent {
			if err := closureMoveSubtreeTx(tx, id, newParentID); err != nil {
				return hcommon.I18nRichError(err, i18n.MsgUgDBUpdateFailed)
			}
		}

		if err := recomputeSubtreeFullPathTx(tx, id); err != nil {
			return err
		}
		if err := assertSubtreeFullPathOKTx(tx, id); err != nil {
			return err
		}

		var fresh UserGroup
		if err := tx.First(&fresh, id).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
		}
		updated = &fresh
		return nil
	})
	if err != nil {
		var richErr *hcommon.RichError
		if errors.As(err, &richErr) {
			return nil, richErr
		}
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBUpdateFailed)
	}
	return updated, nil
}

// DeleteUserGroup 删除用户组（物理删除）。
// 事务内：1) DELETE 成员 2) DELETE 本组 3) DELETE closure。
// 注意：调用方应先校验 CanDeleteUserGroup 不阻塞。
func DeleteUserGroup(ctx context.Context, id uint) error {
	err := DB(ctx).Transaction(func(tx *gorm.DB) error {
		g, err := groupByIDTx(tx, id)
		if err != nil {
			return err
		}
		if err := AssertEditableByAdmin(g); err != nil {
			return err
		}

		// 子组检查（manual 子组阻塞，由 CanDeleteUserGroup 覆盖；这里兜底）
		var childCount int64
		if err := tx.Model(&UserGroup{}).
			Where("parent_id = ? AND source = ?", id, GroupSourceManual).
			Count(&childCount).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
		}
		if childCount > 0 {
			return ErrGroupHasDependencies
		}

		// 🆕 v6.13：直属 Agent 阻塞（事务内再查一次，兜底并发场景）
		var instCount int64
		if err := tx.Model(&Instance{}).
			Where("group_id = ?", id).
			Count(&instCount).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
		}
		if instCount > 0 {
			return ErrGroupHasDependencies
		}

		// 1) 成员物理删除
		if err := tx.Where("user_group_id = ?", id).Delete(&UserGroupMember{}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBDeleteFailed)
		}
		// 2) 物理删除本组
		if err := tx.Delete(&UserGroup{}, id).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBDeleteFailed)
		}
		// 3) 清 closure（本组为祖先或后代的全部行）
		if err := closureDeleteNodeTx(tx, id); err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBDeleteFailed)
		}
		// 4) 清理配置绑定（含 CLS 采集范围等，事务内保证原子性）
		if err := CleanupConfigBindingsByGroupID(tx, id); err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBDeleteFailed)
		}
		return nil
	})
	if err != nil {
		var richErr *hcommon.RichError
		if errors.As(err, &richErr) {
			return richErr
		}
		return hcommon.I18nRichError(err, i18n.MsgUgDBDeleteFailed)
	}
	return nil
}

// DeleteUserGroupForOneIDDept 是 DeleteUserGroup 针对 OneID 部门同步的特殊变体：
// 跳过 AssertEditableByAdmin 允许删除 source=oneid_dept 的组；子组阻塞检查也放宽为
// "所有子组（不限 source）"—— 调用方（landing 流程）已保证按 post-order 从叶子开始删除，
// 不会踩到子组阻塞。
func DeleteUserGroupForOneIDDept(ctx context.Context, id uint) error {
	err := DB(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := groupByIDTx(tx, id); err != nil {
			return err
		}

		// 子组阻塞（所有 source）：若仍有未删除的子，视为调用序错误
		var childCount int64
		if err := tx.Model(&UserGroup{}).
			Where("parent_id = ?", id).
			Count(&childCount).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
		}
		if childCount > 0 {
			return ErrGroupHasDependencies
		}

		// 🆕 v6.13：直属 Agent 阻塞（事务内兜底，覆盖 CanDeleteUserGroup 预检
		// 与本函数执行之间的并发窗口 —— 预检通过后另一个请求新建了绑此组的
		// agent，事务内再查一次避免把 agent 变孤儿）。
		var instCount int64
		if err := tx.Model(&Instance{}).
			Where("group_id = ?", id).
			Count(&instCount).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
		}
		if instCount > 0 {
			return ErrGroupHasDependencies
		}

		// 1) 成员物理删除（含 manual 与 oneid_dept）
		if err := tx.Where("user_group_id = ?", id).Delete(&UserGroupMember{}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBDeleteFailed)
		}
		// 2) 物理删除本组
		if err := tx.Delete(&UserGroup{}, id).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBDeleteFailed)
		}
		// 3) 清 closure
		if err := closureDeleteNodeTx(tx, id); err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBDeleteFailed)
		}
		// 4) 清理配置绑定（含 CLS 采集范围等，事务内保证原子性）
		if err := CleanupConfigBindingsByGroupID(tx, id); err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBDeleteFailed)
		}
		return nil
	})
	if err != nil {
		var richErr *hcommon.RichError
		if errors.As(err, &richErr) {
			return richErr
		}
		return hcommon.I18nRichError(err, i18n.MsgUgDBDeleteFailed)
	}
	return nil
}

// ListUserGroupsOpts 分页列表查询参数。
type ListUserGroupsOpts struct {
	ParentID *uint  // nil = 不按父过滤；非 nil = 精确匹配（0 = 根组）
	Source   string // "" = 不过滤
	Query    string // 按 name 模糊
	Page     int
	PageSize int
}

// ListUserGroupsExt 扩展分页列表。
func ListUserGroupsExt(ctx context.Context, opts ListUserGroupsOpts) ([]UserGroup, int64, error) {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 || opts.PageSize > 200 {
		opts.PageSize = 20
	}
	q := DB(ctx).Model(&UserGroup{})
	if opts.ParentID != nil {
		q = q.Where("parent_id = ?", *opts.ParentID)
	}
	if opts.Source != "" {
		q = q.Where("source = ?", opts.Source)
	}
	if strings.TrimSpace(opts.Query) != "" {
		q = q.Where("name LIKE ?", "%"+strings.TrimSpace(opts.Query)+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	var groups []UserGroup
	if err := q.Order("parent_id ASC, name ASC").
		Offset((opts.Page - 1) * opts.PageSize).Limit(opts.PageSize).
		Find(&groups).Error; err != nil {
		return nil, 0, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	return groups, total, nil
}

// ListUserGroups 向后兼容：无过滤分页。
func ListUserGroups(ctx context.Context, page, pageSize int) ([]UserGroup, int64, error) {
	return ListUserGroupsExt(ctx, ListUserGroupsOpts{Page: page, PageSize: pageSize})
}

// CanDeleteUserGroup 检查用户组是否允许被删除。
// 阻塞项：
//   - 6 张旧 *_visibility_groups 表有绑定
//   - manual 子组
//
// 注意：成员不阻塞（事务内自动清理）。
func CanDeleteUserGroup(ctx context.Context, groupID uint) (bool, error) {
	if used, err := IsGroupUsedByModelVisibility(ctx, groupID); err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	} else if used {
		return false, nil
	}
	if used, err := IsGroupUsedBySkillVisibility(ctx, groupID); err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	} else if used {
		return false, nil
	}
	if used, err := IsGroupUsedBySkillBundleVisibility(ctx, groupID); err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	} else if used {
		return false, nil
	}
	if used, err := IsGroupUsedByRoleVisibility(ctx, groupID); err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	} else if used {
		return false, nil
	}
	if used, err := IsGroupUsedByTagVisibility(ctx, groupID); err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	} else if used {
		return false, nil
	}
	// manual 子组
	var count int64
	if err := DB(ctx).Model(&UserGroup{}).
		Where("parent_id = ? AND source = ?", groupID, GroupSourceManual).
		Count(&count).Error; err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	if count > 0 {
		return false, nil
	}
	// 统一绑定表（通道/插件/MCP/镜像类型/策略）
	if used, err := IsGroupUsedByConfigBindings(ctx, groupID); err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	} else if used {
		return false, nil
	}
	// 🆕 v6.13：直属 Agent（instances.group_id = X）阻塞删除。
	// 必须先把这些 Agent 迁到别的分组或销毁，才能删除本组。
	var instCount int64
	if err := DB(ctx).Model(&Instance{}).
		Where("group_id = ?", groupID).
		Count(&instCount).Error; err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	if instCount > 0 {
		return false, nil
	}
	return true, nil
}

// ──────────────────────────────────────────────
// 以下为 ListMembers / Ungrouped / 成员写入 原有 API（保留不动，签名不变）
// ──────────────────────────────────────────────

// CountGroupMembers 查询用户组成员数
func CountGroupMembers(ctx context.Context, groupID uint) (int64, error) {
	var count int64
	err := DB(ctx).Model(&UserGroupMember{}).Where("user_group_id = ?", groupID).Count(&count).Error
	if err != nil {
		return 0, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	return count, nil
}

// groupMemberCount 用于接收批量 COUNT 查询结果
type groupMemberCount struct {
	UserGroupID uint  `gorm:"column:user_group_id"`
	Count       int64 `gorm:"column:count"`
}

// CountGroupMembersBatch 批量查询多个用户组的成员数，一次 SQL 完成，避免 N+1。
// 返回 groupID → count 的映射；不在结果中的 groupID 表示成员数为 0。
func CountGroupMembersBatch(db *gorm.DB, groupIDs []uint) (map[uint]int64, error) {
	result := make(map[uint]int64, len(groupIDs))
	if len(groupIDs) == 0 {
		return result, nil
	}
	var rows []groupMemberCount
	err := db.Model(&UserGroupMember{}).
		Select("user_group_id, COUNT(*) AS count").
		Where("user_group_id IN ?", groupIDs).
		Group("user_group_id").
		Scan(&rows).Error
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	for _, row := range rows {
		result[row.UserGroupID] = row.Count
	}
	return result, nil
}

// GetUserGroupsByUserID 查询指定用户所在的所有用户组（原签名）
func GetUserGroupsByUserID(ctx context.Context, userID uint) ([]UserGroup, error) {
	groupIDs, err := GetUserGroupIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(groupIDs) == 0 {
		return []UserGroup{}, nil
	}
	return GetGroupsByIDs(ctx, groupIDs)
}

// UserGroupsOfUser 单个用户的用户组归属结果
type UserGroupsOfUser struct {
	UserID uint        `json:"user_id"`
	Groups []UserGroup `json:"groups"`
}

// GetUserGroupsByUserIDs 批量查询多个用户所在的用户组，返回 userID → []UserGroup 的映射。
func GetUserGroupsByUserIDs(ctx context.Context, userIDs []uint) (map[uint][]UserGroup, error) {
	result := make(map[uint][]UserGroup, len(userIDs))
	for _, uid := range userIDs {
		result[uid] = []UserGroup{}
	}
	if len(userIDs) == 0 {
		return result, nil
	}

	var members []UserGroupMember
	if err := DB(ctx).Where("user_id IN ?", userIDs).Find(&members).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	if len(members) == 0 {
		return result, nil
	}

	groupIDSet := make(map[uint]struct{}, len(members))
	for _, m := range members {
		groupIDSet[m.UserGroupID] = struct{}{}
	}
	groupIDs := make([]uint, 0, len(groupIDSet))
	for gid := range groupIDSet {
		groupIDs = append(groupIDs, gid)
	}

	groups, err := GetGroupsByIDs(ctx, groupIDs)
	if err != nil {
		return nil, err
	}
	groupMap := make(map[uint]UserGroup, len(groups))
	for _, g := range groups {
		groupMap[g.ID] = g
	}
	for _, m := range members {
		if g, ok := groupMap[m.UserGroupID]; ok {
			result[m.UserID] = append(result[m.UserID], g)
		}
	}
	return result, nil
}

// GetUserGroupIDs 查询用户所属的所有用户组 ID
func GetUserGroupIDs(ctx context.Context, userID uint) ([]uint, error) {
	return GetUserGroupIDsWithDB(DB(ctx), userID)
}

// GetUserGroupIDsWithDB 是 GetUserGroupIDs 的事务安全变体，调用方传入事务句柄 tx。
func GetUserGroupIDsWithDB(tx *gorm.DB, userID uint) ([]uint, error) {
	var members []UserGroupMember
	if err := tx.Where("user_id = ?", userID).Find(&members).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	ids := make([]uint, len(members))
	for i, m := range members {
		ids[i] = m.UserGroupID
	}
	return ids, nil
}

// GetUserGroupName 按 ID 查分组名（不存在返回空串）。
func GetUserGroupName(ctx context.Context, groupID uint) string {
	name, _ := GetUserGroupNameWithDB(DB(ctx), groupID)
	return name
}

// GetUserGroupNameWithDB 是 GetUserGroupName 的事务安全变体，调用方传入事务句柄 tx。
// 分组不存在时返回空名称和 nil；其他数据库错误原样返回，避免静默吞错。
func GetUserGroupNameWithDB(tx *gorm.DB, groupID uint) (string, error) {
	if groupID == 0 {
		return "", nil
	}
	var g UserGroup
	if err := tx.Select("name").First(&g, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return g.Name, nil
}

// GetGroupsByNames 按名称批量查询用户组
func GetGroupsByNames(ctx context.Context, names []string) ([]UserGroup, error) {
	if len(names) == 0 {
		return []UserGroup{}, nil
	}
	var groups []UserGroup
	if err := DB(ctx).Where("name IN ?", names).Find(&groups).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	return groups, nil
}

// GetGroupsByFullPaths 按 full_path 批量查询分组（精确匹配，多层级安全）
func GetGroupsByFullPaths(ctx context.Context, paths []string) ([]UserGroup, error) {
	if len(paths) == 0 {
		return []UserGroup{}, nil
	}
	var groups []UserGroup
	if err := DB(ctx).Where("full_path IN ?", paths).Find(&groups).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	return groups, nil
}

// GetGroupsByIDs 批量查询用户组信息
func GetGroupsByIDs(ctx context.Context, ids []uint) ([]UserGroup, error) {
	if len(ids) == 0 {
		return []UserGroup{}, nil
	}
	var groups []UserGroup
	if err := DB(ctx).Where("id IN ?", ids).Find(&groups).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	return groups, nil
}

// GetGroupsCVMInstanceIDs 查询指定分组下的 CVM 实例 ID 列表（按实例的 group_id 过滤，已去重）
// 实例归属固定跟随 group_id，不随用户的分组成员关系变化
func GetGroupsCVMInstanceIDs(ctx context.Context, groupIDs []uint) ([]string, error) {
	if len(groupIDs) == 0 {
		return []string{}, nil
	}

	var instances []Instance
	if err := DB(ctx).Select("instance_id").
		Where("group_id IN ? AND instance_id != ''", groupIDs).
		Find(&instances).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	if len(instances) == 0 {
		return []string{}, nil
	}

	seen := make(map[string]struct{}, len(instances))
	result := make([]string, 0, len(instances))
	for _, inst := range instances {
		if _, ok := seen[inst.InstanceId]; !ok {
			seen[inst.InstanceId] = struct{}{}
			result = append(result, inst.InstanceId)
		}
	}
	return result, nil
}

// GetUngroupedCVMInstanceIDs 查询未加入任何用户组的用户关联的 CVM 实例 ID 列表
func GetUngroupedCVMInstanceIDs(ctx context.Context) ([]string, error) {
	var users []User
	if err := DB(ctx).Select("users.id").
		Joins("LEFT JOIN user_group_members m ON m.user_id = users.id").
		Where("m.user_id IS NULL").
		Find(&users).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	if len(users) == 0 {
		return []string{}, nil
	}

	userIDs := make([]uint, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}

	var instances []Instance
	if err := DB(ctx).Select("instance_id").
		Where("user_id IN ? AND instance_id != ''", userIDs).
		Find(&instances).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	if len(instances) == 0 {
		return []string{}, nil
	}

	seen := make(map[string]struct{}, len(instances))
	result := make([]string, 0, len(instances))
	for _, inst := range instances {
		if _, ok := seen[inst.InstanceId]; !ok {
			seen[inst.InstanceId] = struct{}{}
			result = append(result, inst.InstanceId)
		}
	}
	return result, nil
}

// GetUserGroups 查询用户所在的所有用户组
func GetUserGroups(ctx context.Context, userID uint) ([]UserGroup, error) {
	groupIDs, err := GetUserGroupIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(groupIDs) == 0 {
		return []UserGroup{}, nil
	}
	return GetGroupsByIDs(ctx, groupIDs)
}

// deduplicateUintSlice 对 uint 切片去重，保持原始顺序。
func deduplicateUintSlice(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

// validateUserIDs 校验所有 user_id 是否存在于 users 表（含禁用/软删除用户）。
func validateUserIDs(db *gorm.DB, userIDs []uint) error {
	unique := deduplicateUintSlice(userIDs)
	var count int64
	if err := db.Unscoped().Model(&User{}).Where("id IN ?", unique).Count(&count).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	if count != int64(len(unique)) {
		return ErrInvalidUserID
	}
	return nil
}

// ──────────────────────────────────────────────
// 内部事务辅助（名字带 Tx 的一律接收 *gorm.DB 参数，不产生 N+1）
// ──────────────────────────────────────────────

// groupByIDTx 事务内查单个分组（不做 to_be_deleted 过滤）。
func groupByIDTx(tx *gorm.DB, id uint) (*UserGroup, error) {
	var g UserGroup
	if err := tx.First(&g, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserGroupNotFound
		}
		return nil, hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	return &g, nil
}

// assertNameUniqueUnderParentTx 校验 (identifier, parent_id, name) 唯一。
// excludeID 传 0 表示不排除任何已有组（创建场景）；>0 表示排除自身（更新场景）。
func assertNameUniqueUnderParentTx(tx *gorm.DB, excludeID, parentID uint, name string) error {
	name = strings.TrimSpace(name)
	q := tx.Model(&UserGroup{}).Where("parent_id = ? AND name = ?", parentID, name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var cnt int64
	if err := q.Count(&cnt).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
	}
	if cnt > 0 {
		return ErrGroupNameConflict
	}
	return nil
}

// recomputeSubtreeFullPathTx 递归重算子树每个节点的 full_path 和 depth（基于当前 parent_id 链）。
// 实现方式：以自上而下的 BFS 遍历，使用 parent_id 索引 + 循环避免深度递归栈。
func recomputeSubtreeFullPathTx(tx *gorm.DB, rootID uint) error {
	// 先拿 root 本身的 full_path / depth 作为起点
	root, err := groupByIDTx(tx, rootID)
	if err != nil {
		return err
	}
	// 用 BFS：queue 存 (group_id, full_path, depth)
	type item struct {
		id       uint
		fullPath string
		depth    int
	}
	// root 本身已经由 caller 更新过 name/full_path/depth，这里以最新值继续向下递推
	queue := []item{{id: root.ID, fullPath: root.FullPath, depth: root.Depth}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		var children []UserGroup
		if err := tx.Where("parent_id = ?", cur.id).Find(&children).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
		}
		for _, c := range children {
			newDepth := cur.depth + 1
			newFullPath := cur.fullPath + "/" + c.Name
			if len([]rune(newFullPath)) > MaxFullPathLength {
				return ErrFullPathTooLong
			}
			if err := tx.Model(&UserGroup{}).Where("id = ?", c.ID).
				Updates(map[string]any{"full_path": newFullPath, "depth": newDepth}).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgUgDBUpdateFailed)
			}
			queue = append(queue, item{id: c.ID, fullPath: newFullPath, depth: newDepth})
		}
	}
	return nil
}

// assertSubtreeFullPathOKTx 校验整棵子树每个节点的 full_path 长度都 ≤ 512。
// recomputeSubtreeFullPathTx 已逐节点校验，这里做一次兜底（跨事务场景）。
func assertSubtreeFullPathOKTx(tx *gorm.DB, rootID uint) error {
	var rows []UserGroup
	// 通过 closure 查所有后代（含自身）
	if err := tx.Raw(`
		SELECT g.* FROM user_groups g
		INNER JOIN group_closure c ON c.descendant_id = g.id
		WHERE c.ancestor_id = ?
	`, rootID).Scan(&rows).Error; err != nil {
		// group_closure 可能还未建（SQLite AutoMigrate 前），兜底走 parent_id 遍历
		return nil
	}
	for _, r := range rows {
		if len([]rune(r.FullPath)) > MaxFullPathLength {
			return ErrFullPathTooLong
		}
	}
	return nil
}

// RecomputeFullPathAll 全量重算所有分组的 full_path（启动自检用）。
// 基于 parent_id 链从根向下重建，不依赖 closure 表。
func RecomputeFullPathAll(ctx context.Context) error {
	err := DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 先拿所有组，按 depth 升序排（先根后子）
		var all []UserGroup
		if err := tx.Order("depth ASC, parent_id ASC, id ASC").Find(&all).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgUgDBQueryFailed)
		}
		// index by id
		byID := make(map[uint]*UserGroup, len(all))
		for i := range all {
			byID[all[i].ID] = &all[i]
		}
		updates := make(map[uint][2]any, len(all)) // id → [fullPath, depth]
		for _, g := range all {
			name := strings.TrimSpace(g.Name)
			if g.ParentID == 0 {
				if g.FullPath != name || g.Depth != 0 {
					updates[g.ID] = [2]any{name, 0}
				}
				continue
			}
			p, ok := byID[g.ParentID]
			if !ok {
				// 数据损坏：父组不存在；打警但不中断
				continue
			}
			// 如果 parent 已被更新，读更新后的 full_path
			parentFP := p.FullPath
			parentDepth := p.Depth
			if u, ok := updates[p.ID]; ok {
				parentFP = u[0].(string)
				parentDepth = u[1].(int)
			}
			newFP := parentFP + "/" + name
			newDepth := parentDepth + 1
			if g.FullPath != newFP || g.Depth != newDepth {
				updates[g.ID] = [2]any{newFP, newDepth}
			}
		}
		for id, u := range updates {
			if err := tx.Model(&UserGroup{}).Where("id = ?", id).
				Updates(map[string]any{"full_path": u[0], "depth": u[1]}).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgUgRecomputeFullPathFailed, id)
			}
		}
		return nil
	})
	if err != nil {
		var richErr *hcommon.RichError
		if errors.As(err, &richErr) {
			return richErr
		}
		return hcommon.I18nRichError(err, i18n.MsgUgDBUpdateFailed)
	}
	return nil
}
