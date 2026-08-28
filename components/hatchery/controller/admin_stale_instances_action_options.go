package controller

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ── 响应子类型 ────────────────────────────────────────────────────────────────

// actionOptBaseInstance 实例基础信息（no_group、subtree 共用）。
type actionOptBaseInstance struct {
	ID         uint   `json:"id"`
	InstanceID string `json:"instance_id"`
	Name       string `json:"name"`
	UserID     uint   `json:"user_id"`
	Username   string `json:"username"`
}

// actionOptNoGroupEntry no_group 按 user_id 聚合的操作项。
type actionOptNoGroupEntry struct {
	UserID                   uint                    `json:"user_id"`
	Username                 string                  `json:"username"`
	UserGroups               []actionOptUserGroupBrief `json:"user_groups"`
	Instances                []actionOptBaseInstance `json:"instances"`
	Migrate                  bool                    `json:"migrate"`
	Handover                 bool                    `json:"handover"`
	PendingUser              bool                    `json:"pending_user"`
	PendingUserAllowMigrate  bool                    `json:"pending_user_allow_migrate"`
	PendingUserAllowHandover bool                    `json:"pending_user_allow_handover"`
	ArchiveStop              bool                    `json:"archive_stop"`
}

// actionOptNoGroup no_group：用户已加入某分组但实例仍未分组（group_id=0），按 user_id 聚合。
type actionOptNoGroup struct {
	Options []actionOptNoGroupEntry `json:"options"`
}

// actionOptUserGroupBrief user 当前所属的分组信息。
type actionOptUserGroupBrief struct {
	GroupID       uint   `json:"group_id"`
	GroupFullPath string `json:"group_full_path"`
}

// actionOptUserRemovedGroupInstance user_removed 中按 group_id 聚合后的 per-instance 信息。
type actionOptUserRemovedGroupInstance struct {
	ID         uint   `json:"id"`
	InstanceID string `json:"instance_id"`
	Name       string `json:"name"`
	UserID     uint   `json:"user_id"`
	Username   string `json:"username"`
}

// actionOptUserRemovedGroup user_removed 中按 agent 的 group_id 二级聚合。
type actionOptUserRemovedGroup struct {
	GroupID           uint                               `json:"group_id"`
	GroupFullPath     string                             `json:"group_full_path"`
	Instances         []actionOptUserRemovedGroupInstance `json:"instances"`
	HandoverAvailable bool                               `json:"handover_available"`
}

// actionOptUserRemovedEntry user_removed 按 user_id 聚合，再按 agent 的 group_id 二级聚合。
// Handover = OR(all groups.HandoverAvailable)；PendingUserAllowHandover = Handover。
type actionOptUserRemovedEntry struct {
	UserID                   uint                        `json:"user_id"`
	Username                 string                      `json:"username"`
	UserGroups               []actionOptUserGroupBrief   `json:"user_groups"`
	Groups                   []actionOptUserRemovedGroup `json:"groups"`
	Migrate                  bool                        `json:"migrate"`
	Handover                 bool                        `json:"handover"`
	PendingUser              bool                        `json:"pending_user"`
	PendingUserAllowMigrate  bool                        `json:"pending_user_allow_migrate"`
	PendingUserAllowHandover bool                        `json:"pending_user_allow_handover"`
	ArchiveStop              bool                        `json:"archive_stop"`
}

// actionOptUserRemoved user_removed：用户被移出某/所有分组，实例 group_id != 0，按 user_id 聚合。
type actionOptUserRemoved struct {
	Options []actionOptUserRemovedEntry `json:"options"`
}

// actionOptSubtreeGroup subtree 按 group_id 聚合。
type actionOptSubtreeGroup struct {
	GroupID       uint                    `json:"group_id"`
	GroupFullPath string                  `json:"group_full_path"`
	Instances     []actionOptBaseInstance `json:"instances"`
}

// actionOptSubtree subtree：分组父级变更，仅支持 migrate。
type actionOptSubtree struct {
	Groups []actionOptSubtreeGroup `json:"groups"`
}

// ── 内部行结构 ───────────────────────────────────────────────────────────────

type staleActionInstRow struct {
	ID         uint
	InstanceID string
	Name       string
	UserID     uint
	GroupID    uint
}

// ── Handler ──────────────────────────────────────────────────────────────────

// HandleAdminStaleInstancesActionOptions POST /admin/stale-instances/action-options
// 给定 user_group_ids / group_ids，返回 3 个场景下的实例列表及可用操作选项。
//
// 场景来源：
//   - no_group：全量扫描，当前有分组成员关系的 user_id 中 group_id=0 的实例（按 user_id 聚合）
//   - user_removed：user_group_ids 精确 (user_id, group_id) 对命中的实例（按 user_id 聚合）
//   - subtree：group_ids 子树展开后成员的实例，按 group_id 聚合
func HandleAdminStaleInstancesActionOptions(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req instancesByUserGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	// user_group_ids 和 group_ids 均为可选；no_group 场景始终全量扫描，不依赖请求参数。

	ctx := r.Context()

	// ── no_group：全量扫描 — 当前有分组成员关系的 user_id 中 group_id=0 的实例 ──

	var noGroupRows []staleActionInstRow
	{
		var usersWithGroups []uint
		if err := model.DB(ctx).Model(&model.UserGroupMember{}).
			Distinct("user_id").
			Pluck("user_id", &usersWithGroups).Error; err != nil {
			slog.Warn("[ActionOptions] users with groups query failed", "err", err)
		}
		if len(usersWithGroups) > 0 {
			noGroupRows = queryGroupZeroInstances(ctx, usersWithGroups)
		}
	}

	// ── user_removed：user_group_ids 精确 (user_id, group_id) 对 ───────────────

	var userRemovedRows []staleActionInstRow
	if len(req.UserGroupIDs) > 0 {
		pairSet := make(map[userGroupPair]struct{}, len(req.UserGroupIDs))
		for _, p := range req.UserGroupIDs {
			if p.UserID != 0 {
				pairSet[p] = struct{}{}
			}
		}
		if len(pairSet) > 0 {
			userRemovedRows = queryInstancesByPairs(ctx, pairSet)
		}
	}

	// ── subtree：group_ids 子树展开 → 直接按 group_id IN subtree 查实例 ──────────
	//    不走 user_group_members pair 匹配 —— 子树场景需要返回子树下所有实例，
	//    包括用户不是该子分组直属成员但实例 group_id 仍在子树内的情况。

	var subtreeRows []staleActionInstRow
	if len(req.GroupIDs) > 0 {
		subtreeRows = queryInstancesBySubtree(ctx, req.GroupIDs)
	}

	// ── 过滤已打 stale_group 标记的实例，避免重复处理 ──────────────────────────

	staleGroupInstanceIDs := loadStaleGroupFlaggedInstanceIDs(ctx, noGroupRows, userRemovedRows, subtreeRows)
	if len(staleGroupInstanceIDs) > 0 {
		noGroupRows = filterRowsExcludingIDs(noGroupRows, staleGroupInstanceIDs)
		userRemovedRows = filterRowsExcludingIDs(userRemovedRows, staleGroupInstanceIDs)
		subtreeRows = filterRowsExcludingIDs(subtreeRows, staleGroupInstanceIDs)
	}

	// ── 汇总所有 user_id，批量加载 username ──────────────────────────────────────

	allUserIDSet := make(map[uint]struct{})
	for _, row := range noGroupRows {
		if row.UserID != 0 {
			allUserIDSet[row.UserID] = struct{}{}
		}
	}
	for _, row := range userRemovedRows {
		if row.UserID != 0 {
			allUserIDSet[row.UserID] = struct{}{}
		}
	}
	for _, row := range subtreeRows {
		if row.UserID != 0 {
			allUserIDSet[row.UserID] = struct{}{}
		}
	}
	allUserIDs := make([]uint, 0, len(allUserIDSet))
	for uid := range allUserIDSet {
		allUserIDs = append(allUserIDs, uid)
	}
	usernameMap := loadActionOptionsUsernameMap(ctx, allUserIDs)
	// 所有 user 的当前分组列表（no_group 和 user_removed 共用）
	allUserGroupsMap := loadUserGroupsBriefMap(ctx, allUserIDs)

	// ── user_removed 辅助数据 ────────────────────────────────────────────────────

	userRemovedGroupIDs := collectUintGroupIDs(userRemovedRows)
	userRemovedMemberCounts := loadGroupMemberCounts(ctx, userRemovedGroupIDs)
	userRemovedGroupPaths := fetchGroupFullPathMap(ctx, userRemovedGroupIDs)

	// ── subtree 辅助数据 ─────────────────────────────────────────────────────────

	subtreeGroupIDs := collectUintGroupIDs(subtreeRows)
	subtreeGroupPaths := fetchGroupFullPathMap(ctx, subtreeGroupIDs)

	jsonOK(w, map[string]interface{}{
		"ok":           true,
		"no_group":     buildActionOptNoGroup(noGroupRows, usernameMap, allUserGroupsMap),
		"user_removed": buildActionOptUserRemoved(userRemovedRows, usernameMap, userRemovedMemberCounts, userRemovedGroupPaths, allUserGroupsMap),
		"subtree":      buildActionOptSubtree(subtreeRows, usernameMap, subtreeGroupPaths),
	})
}

// ── 场景构建辅助 ──────────────────────────────────────────────────────────────

func buildActionOptNoGroup(rows []staleActionInstRow, usernameMap map[uint]string, userGroupsMap map[uint][]actionOptUserGroupBrief) actionOptNoGroup {
	userOrder := make([]uint, 0)
	userMap := make(map[uint][]actionOptBaseInstance)
	for _, row := range rows {
		if _, exists := userMap[row.UserID]; !exists {
			userOrder = append(userOrder, row.UserID)
			userMap[row.UserID] = []actionOptBaseInstance{}
		}
		userMap[row.UserID] = append(userMap[row.UserID], actionOptBaseInstance{
			ID:         row.ID,
			InstanceID: row.InstanceID,
			Name:       row.Name,
			UserID:     row.UserID,
			Username:   usernameMap[row.UserID],
		})
	}
	entries := make([]actionOptNoGroupEntry, 0, len(userOrder))
	for _, uid := range userOrder {
		entries = append(entries, actionOptNoGroupEntry{
			UserID:                   uid,
			Username:                 usernameMap[uid],
			UserGroups:               userGroupsMap[uid],
			Instances:                userMap[uid],
			Migrate:                  true,
			Handover:                 false,
			PendingUser:              true,
			PendingUserAllowMigrate:  true,
			PendingUserAllowHandover: false,
			ArchiveStop:              true,
		})
	}
	return actionOptNoGroup{Options: entries}
}

func buildActionOptUserRemoved(
	rows []staleActionInstRow,
	usernameMap map[uint]string,
	groupMemberCounts map[uint]int,
	groupPaths map[uint]string,
	userGroupsMap map[uint][]actionOptUserGroupBrief,
) actionOptUserRemoved {
	// 第一层：按 user_id 聚合
	userOrder := make([]uint, 0)
	userRows := make(map[uint][]staleActionInstRow)
	for _, row := range rows {
		if _, exists := userRows[row.UserID]; !exists {
			userOrder = append(userOrder, row.UserID)
			userRows[row.UserID] = nil
		}
		userRows[row.UserID] = append(userRows[row.UserID], row)
	}

	entries := make([]actionOptUserRemovedEntry, 0, len(userOrder))
	for _, uid := range userOrder {
		insts := userRows[uid]

		// 第二层：按 agent 的 group_id 聚合
		groupOrder := make([]uint, 0)
		groupInsts := make(map[uint][]actionOptUserRemovedGroupInstance)
		for _, row := range insts {
			if _, exists := groupInsts[row.GroupID]; !exists {
				groupOrder = append(groupOrder, row.GroupID)
				groupInsts[row.GroupID] = nil
			}
			groupInsts[row.GroupID] = append(groupInsts[row.GroupID], actionOptUserRemovedGroupInstance{
				ID:         row.ID,
				InstanceID: row.InstanceID,
				Name:       row.Name,
				UserID:     row.UserID,
				Username:   usernameMap[row.UserID],
			})
		}

		groups := make([]actionOptUserRemovedGroup, 0, len(groupOrder))
		handover := false
		for _, gid := range groupOrder {
			avail := groupMemberCounts[gid] > 0
			if avail {
				handover = true
			}
			groups = append(groups, actionOptUserRemovedGroup{
				GroupID:           gid,
				GroupFullPath:     groupPaths[gid],
				Instances:         groupInsts[gid],
				HandoverAvailable: avail,
			})
		}

		entries = append(entries, actionOptUserRemovedEntry{
			UserID:                   uid,
			Username:                 usernameMap[uid],
			UserGroups:               userGroupsMap[uid],
			Groups:                   groups,
			Migrate:                  true,
			Handover:                 handover,
			PendingUser:              true,
			PendingUserAllowMigrate:  true,
			PendingUserAllowHandover: handover,
			ArchiveStop:              true,
		})
	}
	return actionOptUserRemoved{Options: entries}
}

func buildActionOptSubtree(
	rows []staleActionInstRow,
	usernameMap map[uint]string,
	groupPaths map[uint]string,
) actionOptSubtree {
	groupOrder := make([]uint, 0)
	groupMap := make(map[uint][]actionOptBaseInstance)
	for _, row := range rows {
		if _, exists := groupMap[row.GroupID]; !exists {
			groupOrder = append(groupOrder, row.GroupID)
			groupMap[row.GroupID] = []actionOptBaseInstance{}
		}
		groupMap[row.GroupID] = append(groupMap[row.GroupID], actionOptBaseInstance{
			ID:         row.ID,
			InstanceID: row.InstanceID,
			Name:       row.Name,
			UserID:     row.UserID,
			Username:   usernameMap[row.UserID],
		})
	}
	groups := make([]actionOptSubtreeGroup, 0, len(groupOrder))
	for _, gid := range groupOrder {
		groups = append(groups, actionOptSubtreeGroup{
			GroupID:       gid,
			GroupFullPath: groupPaths[gid],
			Instances:     groupMap[gid],
		})
	}
	return actionOptSubtree{Groups: groups}
}

// ── 批量查询辅助 ──────────────────────────────────────────────────────────────

// queryGroupZeroInstances 查询指定 user_id 列表中 group_id=0 的实例。
func queryGroupZeroInstances(ctx context.Context, userIDs []uint) []staleActionInstRow {
	if len(userIDs) == 0 {
		return nil
	}
	var rows []struct {
		ID         uint   `gorm:"column:id"`
		InstanceID string `gorm:"column:instance_id"`
		Name       string `gorm:"column:name"`
		UserID     uint   `gorm:"column:user_id"`
		GroupID    uint   `gorm:"column:group_id"`
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Select("id, instance_id, name, user_id, group_id").
		Where("user_id IN ? AND group_id = 0", userIDs).
		Find(&rows).Error; err != nil {
		slog.Warn("[ActionOptions] group_zero instances query failed", "err", err)
		return nil
	}
	out := make([]staleActionInstRow, len(rows))
	for i, r := range rows {
		out[i] = staleActionInstRow{ID: r.ID, InstanceID: r.InstanceID, Name: r.Name, UserID: r.UserID, GroupID: r.GroupID}
	}
	return out
}

// queryInstancesByPairs 给定 (user_id, group_id) pair set，查询命中的实例（内存精确过滤）。
func queryInstancesByPairs(ctx context.Context, pairSet map[userGroupPair]struct{}) []staleActionInstRow {
	if len(pairSet) == 0 {
		return nil
	}
	userIDSet := make(map[uint]struct{}, len(pairSet))
	groupIDSet := make(map[uint]struct{}, len(pairSet))
	for p := range pairSet {
		userIDSet[p.UserID] = struct{}{}
		groupIDSet[p.GroupID] = struct{}{}
	}
	userIDs := make([]uint, 0, len(userIDSet))
	for uid := range userIDSet {
		userIDs = append(userIDs, uid)
	}
	groupIDs := make([]uint, 0, len(groupIDSet))
	for gid := range groupIDSet {
		groupIDs = append(groupIDs, gid)
	}
	var candidates []struct {
		ID         uint   `gorm:"column:id"`
		InstanceID string `gorm:"column:instance_id"`
		Name       string `gorm:"column:name"`
		UserID     uint   `gorm:"column:user_id"`
		GroupID    uint   `gorm:"column:group_id"`
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Select("id, instance_id, name, user_id, group_id").
		Where("user_id IN ? AND group_id IN ?", userIDs, groupIDs).
		Find(&candidates).Error; err != nil {
		slog.Warn("[ActionOptions] instances pair query failed", "err", err)
		return nil
	}
	out := make([]staleActionInstRow, 0, len(candidates))
	for _, c := range candidates {
		if _, ok := pairSet[userGroupPair{UserID: c.UserID, GroupID: c.GroupID}]; ok {
			out = append(out, staleActionInstRow{ID: c.ID, InstanceID: c.InstanceID, Name: c.Name, UserID: c.UserID, GroupID: c.GroupID})
		}
	}
	return out
}

// queryInstancesBySubtree 给定根 group_id 列表，通过 group_closure 展开子树，
// 直接按 group_id IN subtree 查询实例（不依赖 user_group_members）。
// 子树场景（父级变更）需要返回子树下所有实例，包括用户不是子分组直属成员的情况。
func queryInstancesBySubtree(ctx context.Context, rootGroupIDs []uint) []staleActionInstRow {
	if len(rootGroupIDs) == 0 {
		return nil
	}
	rootSet := make(map[uint]struct{}, len(rootGroupIDs))
	rootIDs := make([]uint, 0, len(rootGroupIDs))
	for _, gid := range rootGroupIDs {
		if gid == 0 {
			continue
		}
		if _, ok := rootSet[gid]; ok {
			continue
		}
		rootSet[gid] = struct{}{}
		rootIDs = append(rootIDs, gid)
	}
	if len(rootIDs) == 0 {
		return nil
	}
	var descIDs []uint
	if err := model.DB(ctx).Model(&model.GroupClosure{}).
		Distinct("descendant_id").
		Where("ancestor_id IN ?", rootIDs).
		Pluck("descendant_id", &descIDs).Error; err != nil {
		slog.Warn("[ActionOptions] group_closure subtree query failed", "err", err)
		return nil
	}
	if len(descIDs) == 0 {
		return nil
	}
	var rows []struct {
		ID         uint   `gorm:"column:id"`
		InstanceID string `gorm:"column:instance_id"`
		Name       string `gorm:"column:name"`
		UserID     uint   `gorm:"column:user_id"`
		GroupID    uint   `gorm:"column:group_id"`
	}
	if err := model.DB(ctx).Model(&model.Instance{}).
		Select("id, instance_id, name, user_id, group_id").
		Where("group_id IN ?", descIDs).
		Find(&rows).Error; err != nil {
		slog.Warn("[ActionOptions] subtree instances query failed", "err", err)
		return nil
	}
	out := make([]staleActionInstRow, len(rows))
	for i, r := range rows {
		out[i] = staleActionInstRow{ID: r.ID, InstanceID: r.InstanceID, Name: r.Name, UserID: r.UserID, GroupID: r.GroupID}
	}
	return out
}

// loadGroupMemberCounts 批量查询 user_group_members 中每个 group 的直属成员数。
// 返回 group_id → count（未出现的 group_id 表示 count=0）。
func loadGroupMemberCounts(ctx context.Context, groupIDs []uint) map[uint]int {
	out := make(map[uint]int, len(groupIDs))
	if len(groupIDs) == 0 {
		return out
	}
	var rows []struct {
		UserGroupID uint `gorm:"column:user_group_id"`
		Count       int  `gorm:"column:count"`
	}
	if err := model.DB(ctx).Model(&model.UserGroupMember{}).
		Select("user_group_id, COUNT(*) as count").
		Where("user_group_id IN ?", groupIDs).
		Group("user_group_id").
		Find(&rows).Error; err != nil {
		slog.Warn("[ActionOptions] group member count query failed", "err", err)
		return out
	}
	for _, r := range rows {
		out[r.UserGroupID] = r.Count
	}
	return out
}

// loadActionOptionsUsernameMap 批量查 user_id → username。
func loadActionOptionsUsernameMap(ctx context.Context, userIDs []uint) map[uint]string {
	out := make(map[uint]string, len(userIDs))
	if len(userIDs) == 0 {
		return out
	}
	var users []struct {
		ID       uint   `gorm:"column:id"`
		Username string `gorm:"column:username"`
	}
	if err := model.DB(ctx).Model(&model.User{}).
		Select("id, username").
		Where("id IN ?", userIDs).
		Find(&users).Error; err != nil {
		slog.Warn("[ActionOptions] username batch query failed", "err", err)
		return out
	}
	for _, u := range users {
		out[u.ID] = u.Username
	}
	return out
}

// loadUserGroupsBriefMap 批量查询多个 user 当前所属的分组列表，返回 user_id → []actionOptUserGroupBrief。
func loadUserGroupsBriefMap(ctx context.Context, userIDs []uint) map[uint][]actionOptUserGroupBrief {
	out := make(map[uint][]actionOptUserGroupBrief, len(userIDs))
	if len(userIDs) == 0 {
		return out
	}
	type memberRow struct {
		UserID      uint `gorm:"column:user_id"`
		UserGroupID uint `gorm:"column:user_group_id"`
	}
	var members []memberRow
	if err := model.DB(ctx).Model(&model.UserGroupMember{}).
		Select("user_id, user_group_id").
		Where("user_id IN ?", userIDs).
		Scan(&members).Error; err != nil {
		slog.Warn("[ActionOptions] user_group_members batch query failed", "err", err)
		return out
	}
	// 收集所有 group_id 用于批量查 full_path
	groupIDSet := make(map[uint]struct{})
	userToGroups := make(map[uint][]uint)
	for _, m := range members {
		userToGroups[m.UserID] = append(userToGroups[m.UserID], m.UserGroupID)
		groupIDSet[m.UserGroupID] = struct{}{}
	}
	groupIDs := make([]uint, 0, len(groupIDSet))
	for gid := range groupIDSet {
		groupIDs = append(groupIDs, gid)
	}
	groupPathMap := fetchGroupFullPathMap(ctx, groupIDs)

	for _, uid := range userIDs {
		gids := userToGroups[uid]
		briefs := make([]actionOptUserGroupBrief, 0, len(gids))
		for _, gid := range gids {
			briefs = append(briefs, actionOptUserGroupBrief{
				GroupID:       gid,
				GroupFullPath: groupPathMap[gid],
			})
		}
		out[uid] = briefs
	}
	return out
}

// collectUintGroupIDs 从 staleActionInstRow 列表中收集唯一非零 group_id。
func collectUintGroupIDs(rows []staleActionInstRow) []uint {
	seen := make(map[uint]struct{})
	out := make([]uint, 0)
	for _, r := range rows {
		if r.GroupID == 0 {
			continue
		}
		if _, ok := seen[r.GroupID]; ok {
			continue
		}
		seen[r.GroupID] = struct{}{}
		out = append(out, r.GroupID)
	}
	return out
}

// loadStaleGroupFlaggedInstanceIDs 收集所有行中的实例 ID，批量查询 instance_flags，
// 返回已打 stale_group 标记的实例 ID 集合。
func loadStaleGroupFlaggedInstanceIDs(ctx context.Context, rowSets ...[]staleActionInstRow) map[uint]struct{} {
	idSet := make(map[uint]struct{})
	for _, rows := range rowSets {
		for _, r := range rows {
			idSet[r.ID] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	flagsMap, err := model.GetInstanceFlagsBatch(ctx, ids)
	if err != nil {
		slog.Warn("[ActionOptions] batch query instance flags failed", "err", err)
		return nil
	}
	out := make(map[uint]struct{})
	for instID, flags := range flagsMap {
		for _, f := range flags {
			if f == model.InstanceFlagStaleGroup {
				out[instID] = struct{}{}
				break
			}
		}
	}
	return out
}

// filterRowsExcludingIDs 过滤掉 ID 在 excludeSet 中的行。
func filterRowsExcludingIDs(rows []staleActionInstRow, excludeSet map[uint]struct{}) []staleActionInstRow {
	if len(excludeSet) == 0 {
		return rows
	}
	out := make([]staleActionInstRow, 0, len(rows))
	for _, r := range rows {
		if _, excluded := excludeSet[r.ID]; excluded {
			continue
		}
		out = append(out, r)
	}
	return out
}
