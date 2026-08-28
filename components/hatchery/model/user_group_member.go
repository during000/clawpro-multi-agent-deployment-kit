package model

import (
	"context"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// MemberSourceManual 管理员手动添加到分组的关系。
	MemberSourceManual = "manual"
	// MemberSourceOneIDDept OneID 同步写入的部门成员关系（🔄 v6.16：原 oneid_sync，改为 oneid_dept 对齐 user_groups.Source）。
	MemberSourceOneIDDept = "oneid_dept"
)

// UserGroupMember 成员关联表（不使用软删除）
//
// 🆕 v6: 扩展 IsMain + Source 两个字段承接 OneID 主部门语义。
type UserGroupMember struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Identifier  string    `gorm:"size:191;not null;default:'';uniqueIndex:idx_ugm_identifier_group_user" json:"-"`
	UserGroupID uint      `gorm:"not null;index;uniqueIndex:idx_ugm_identifier_group_user" json:"user_group_id"`
	UserID      uint      `gorm:"not null;index;uniqueIndex:idx_ugm_identifier_group_user" json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`

	// 🆕 v6 扩展
	IsMain bool   `gorm:"not null;default:false;index" json:"is_main"`           // OneID 主部门标记（同一用户至多一条 true）
	Source string `gorm:"size:32;not null;default:'manual';index" json:"source"` // manual / oneid_dept（🔄 v6.16：对齐 user_groups.Source）
}

// UpdateUserGroupMemberships 全量替换指定用户的用户组归属（在已有事务 tx 中执行）
// groupIDs 为空时，清除该用户所有用户组归属。仅管理员在管控端手动替换时调用，source 统一写 manual。
func UpdateUserGroupMemberships(tx *gorm.DB, userID uint, groupIDs []uint) error {
	// 先删除该用户所有现有的用户组成员记录（含 oneid_dept 与 manual，保持旧行为）
	if err := tx.Where("user_id = ?", userID).Delete(&UserGroupMember{}).Error; err != nil {
		return err
	}
	if len(groupIDs) == 0 {
		return nil
	}
	// 校验所有 group_id 是否存在
	var count int64
	if err := tx.Model(&UserGroup{}).Where("id IN ?", groupIDs).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(groupIDs)) {
		return ErrInvalidUserGroupID
	}
	// 批量查询所有目标用户组的成员数量（1 次 SQL，避免 N+1）
	memberCounts, err := CountGroupMembersBatch(tx, groupIDs)
	if err != nil {
		return err
	}
	for _, gid := range groupIDs {
		if memberCounts[gid] >= MaxMembersPerUserGroup {
			return ErrGroupMemberLimitReached
		}
	}
	// 批量插入新的关联记录
	members := make([]UserGroupMember, len(groupIDs))
	for i, gid := range groupIDs {
		members[i] = UserGroupMember{
			UserGroupID: gid,
			UserID:      userID,
			Source:      MemberSourceManual,
		}
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&members).Error
}

// UpdateUserGroupMembershipsManualOnly 仅替换指定用户的「manual 类型」组归属，
// 不动 oneid_dept 类型的成员关系（OneID 同步独占维护）。
//
// 适用场景：管理员通过 /admin/update-user 修改 group_ids。无论是否 OneID 模式，
// 用户的 oneid_dept 归属都由 OneID 同步流程独占维护，管控端不能借此清空或新增。
//
// 行为约束：
//   - groupIDs 中存在的 source != manual（含 oneid_dept）或 to_be_deleted 的项
//     **自动忽略**，不报错；只对剩下的 manual 子集生效
//   - groupIDs 中任一 group_id 在 user_groups 中不存在 → ErrInvalidUserGroupID
//   - 任一 manual 目标组成员数量已达上限 → ErrGroupMemberLimitReached
//   - 过滤后的 manual 子集为空（含 groupIDs 本身为空）→ 仅清空该用户所有 manual
//     行，oneid_dept 行保留
//   - 写入的成员记录 Source 统一为 manual
//
// 该函数不写 oneid_dept 行；oneid_dept 行的增删只能由 OneID landing 流程触发。
func UpdateUserGroupMembershipsManualOnly(tx *gorm.DB, userID uint, groupIDs []uint) error {
	// 1) 校验 groupIDs：必须全部存在；source != manual 或 to_be_deleted 的项静默过滤
	manualGroupIDs := make([]uint, 0, len(groupIDs))
	if len(groupIDs) > 0 {
		var groups []UserGroup
		if err := tx.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
			return err
		}
		if len(groups) != len(groupIDs) {
			return ErrInvalidUserGroupID
		}
		for i := range groups {
			g := &groups[i]
			// AssertEditableByAdmin 失败（非 manual 或 to_be_deleted）→ 静默忽略
			if err := AssertEditableByAdmin(g); err != nil {
				continue
			}
			manualGroupIDs = append(manualGroupIDs, g.ID)
		}
	}

	// 2) 仅清空该用户的 manual 行，oneid_dept 行保留
	if err := tx.Where("user_id = ? AND source = ?", userID, MemberSourceManual).
		Delete(&UserGroupMember{}).Error; err != nil {
		return err
	}
	if len(manualGroupIDs) == 0 {
		return nil
	}

	// 3) 容量校验（oneid_dept 行也算同组成员，避免越限）
	memberCounts, err := CountGroupMembersBatch(tx, manualGroupIDs)
	if err != nil {
		return err
	}
	for _, gid := range manualGroupIDs {
		if memberCounts[gid] >= MaxMembersPerUserGroup {
			return ErrGroupMemberLimitReached
		}
	}

	// 4) 批量插入 manual 行
	members := make([]UserGroupMember, len(manualGroupIDs))
	for i, gid := range manualGroupIDs {
		members[i] = UserGroupMember{
			UserGroupID: gid,
			UserID:      userID,
			Source:      MemberSourceManual,
		}
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&members).Error
}

// SetGroupMembers 全量替换组内成员（事务内先清空再批量插入）
// 仅允许操作 manual 组；若目标组非 manual 返回 ErrOneIDDeptReadonly。
func SetGroupMembers(ctx context.Context, groupID uint, userIDs []uint) error {
	if len(userIDs) > MaxMembersPerUserGroup {
		return ErrMemberCountExceeded
	}
	return DB(ctx).Transaction(func(tx *gorm.DB) error {
		g, err := groupByIDTx(tx, groupID)
		if err != nil {
			return err
		}
		if err := AssertEditableByAdmin(g); err != nil {
			return err
		}

		if len(userIDs) > 0 {
			if err := validateUserIDs(tx, userIDs); err != nil {
				return err
			}
		}
		// 先清空（source 无关，manual set 意味着"以当前输入为准"）
		if err := tx.Where("user_group_id = ?", groupID).Delete(&UserGroupMember{}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgDeleteUserGroupMemberFailed)
		}
		if len(userIDs) > 0 {
			members := make([]UserGroupMember, len(userIDs))
			for i, uid := range userIDs {
				members[i] = UserGroupMember{
					UserGroupID: groupID,
					UserID:      uid,
					Source:      MemberSourceManual,
				}
			}
			if err := tx.Create(&members).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgCreateUserGroupMemberFailed)
			}
		}
		return nil
	})
}

// AddGroupMembers 批量添加成员（幂等，已存在的忽略）。仅 manual 组允许。
func AddGroupMembers(ctx context.Context, groupID uint, userIDs []uint) error {
	if len(userIDs) == 0 {
		return nil
	}
	userIDs = deduplicateUintSlice(userIDs)
	return DB(ctx).Transaction(func(tx *gorm.DB) error {
		g, err := groupByIDTx(tx, groupID)
		if err != nil {
			return err
		}
		if err := AssertEditableByAdmin(g); err != nil {
			return err
		}
		if err := validateUserIDs(tx, userIDs); err != nil {
			return err
		}
		var currentCount int64
		if err := tx.Model(&UserGroupMember{}).Where("user_group_id = ?", groupID).Count(&currentCount).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSelectUserGroupMemberFailed)
		}
		if currentCount+int64(len(userIDs)) > MaxMembersPerUserGroup {
			return ErrAddMemberWouldExceed
		}
		members := make([]UserGroupMember, len(userIDs))
		for i, uid := range userIDs {
			members[i] = UserGroupMember{
				UserGroupID: groupID,
				UserID:      uid,
				Source:      MemberSourceManual,
			}
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&members).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgCreateUserGroupMemberFailed)
		}
		return nil
	})
}

// RemoveGroupMembers 批量移除成员（不在组内的静默忽略）。仅 manual 组允许。
func RemoveGroupMembers(ctx context.Context, groupID uint, userIDs []uint) error {
	if len(userIDs) == 0 {
		return nil
	}
	return DB(ctx).Transaction(func(tx *gorm.DB) error {
		g, err := groupByIDTx(tx, groupID)
		if err != nil {
			return err
		}
		if err := AssertEditableByAdmin(g); err != nil {
			return err
		}
		if err := tx.Where("user_group_id = ? AND user_id IN ?", groupID, userIDs).Delete(&UserGroupMember{}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgDeleteUserGroupMemberFailed)
		}
		return nil
	})
}

// GroupMemberInfo 单个成员的聚合视图，供 API 响应使用。
type GroupMemberInfo struct {
	UserID   uint      `json:"user_id"`
	Username string    `json:"username"`
	JoinedAt time.Time `json:"joined_at"`
}

// GetGroupMembers 查询组内成员列表（含用户名，含禁用用户）
func GetGroupMembers(ctx context.Context, groupID uint) ([]GroupMemberInfo, error) {
	var members []UserGroupMember
	if err := DB(ctx).Where("user_group_id = ?", groupID).Find(&members).Error; err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return []GroupMemberInfo{}, nil
	}

	userIDs := make([]uint, len(members))
	for i, m := range members {
		userIDs[i] = m.UserID
	}
	var users []User
	if err := DB(ctx).Unscoped().Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	userMap := make(map[uint]string, len(users))
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

	result := make([]GroupMemberInfo, len(members))
	for i, m := range members {
		result[i] = GroupMemberInfo{
			UserID:   m.UserID,
			Username: userMap[m.UserID],
			JoinedAt: m.CreatedAt,
		}
	}
	return result, nil
}

// GetGroupMembersByGroupIDs 批量查询多个组的成员列表
func GetGroupMembersByGroupIDs(ctx context.Context, groupIDs []uint) (map[uint][]GroupMemberInfo, error) {
	result := make(map[uint][]GroupMemberInfo, len(groupIDs))
	for _, gid := range groupIDs {
		result[gid] = []GroupMemberInfo{}
	}
	if len(groupIDs) == 0 {
		return result, nil
	}

	var members []UserGroupMember
	if err := DB(ctx).Where("user_group_id IN ?", groupIDs).Find(&members).Error; err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return result, nil
	}

	userIDSet := make(map[uint]struct{}, len(members))
	for _, m := range members {
		userIDSet[m.UserID] = struct{}{}
	}
	userIDs := make([]uint, 0, len(userIDSet))
	for uid := range userIDSet {
		userIDs = append(userIDs, uid)
	}

	var users []User
	if err := DB(ctx).Unscoped().Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	userMap := make(map[uint]string, len(users))
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

	for _, m := range members {
		result[m.UserGroupID] = append(result[m.UserGroupID], GroupMemberInfo{
			UserID:   m.UserID,
			Username: userMap[m.UserID],
			JoinedAt: m.CreatedAt,
		})
	}
	return result, nil
}

// GetGroupMembersPaged 分页查询组内成员列表
func GetGroupMembersPaged(ctx context.Context, groupID uint, page, pageSize int) ([]GroupMemberInfo, int64, error) {
	var total int64
	if err := DB(ctx).Model(&UserGroupMember{}).Where("user_group_id = ?", groupID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var members []UserGroupMember
	if err := DB(ctx).Where("user_group_id = ?", groupID).
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&members).Error; err != nil {
		return nil, 0, err
	}

	if len(members) == 0 {
		return []GroupMemberInfo{}, total, nil
	}

	userIDs := make([]uint, len(members))
	for i, m := range members {
		userIDs[i] = m.UserID
	}
	var users []User
	if err := DB(ctx).Unscoped().Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	userMap := make(map[uint]string, len(users))
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

	result := make([]GroupMemberInfo, len(members))
	for i, m := range members {
		result[i] = GroupMemberInfo{
			UserID:   m.UserID,
			Username: userMap[m.UserID],
			JoinedAt: m.CreatedAt,
		}
	}
	return result, total, nil
}

// GroupMemberRow 内部查询聚合行（供 usergroup 包用，返回原始 member 记录以便拼装 direct_groups）。
// 与 GroupMemberInfo 相比，额外包含 IsMain 与 Source（按当前查询分组）。
type GroupMemberRow struct {
	UserID   uint
	Username string
	JoinedAt time.Time
	IsMain   bool
	Source   string
}
