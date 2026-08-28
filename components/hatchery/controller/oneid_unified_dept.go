package controller

// oneid_unified_dept.go — 统一账号模式：Hatchery 分组 → OneID 部门单向同步。
//
// 统一账号模式下，管理员在 Hatchery 创建/修改/删除 manual 用户组时，
// 同步到 OneID 对应的部门结构。
//
// 调用方式：Hatchery 用自建应用 Token 直调 OneID OpenAPI。
// 根部门不需要创建，ParentID=0 的顶级分组映射为 OneID 根部门的子部门。

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ── OneID 部门 API ──────────────────────────────────────────────────────────────

// OneIDCreateDepartment 在 OneID 创建部门，返回 department_id。
// POST /openapi/v3/contacts/departments
func OneIDCreateDepartment(ctx context.Context, name, parentDeptID string) (string, error) {
	token, err := getOneIDAppToken(ctx)
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgOneIDGetAppTokenFailed)
	}

	apiURL := getOneIDAPIBaseURL(ctx) + "/openapi/v3/contacts/departments"
	respBody, err := oneIDAPICall(ctx, http.MethodPost, apiURL, token, map[string]interface{}{
		"department_name": name,
		"parent_id":       parentDeptID,
	})
	if err != nil {
		return "", err
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			DepartmentID string `json:"department_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgOneIDParseCreateDeptResponse)
	}
	if result.Code != 0 {
		return "", hcommon.I18nError(i18n.MsgOneIDCreateDeptError, result.Code, result.Msg)
	}
	return result.Data.DepartmentID, nil
}

// OneIDUpdateDepartment 更新 OneID 部门信息（名称/父部门）。
// PATCH /openapi/v3/contacts/departments/{department_id}
func OneIDUpdateDepartment(ctx context.Context, deptID string, fields map[string]interface{}) error {
	token, err := getOneIDAppToken(ctx)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgOneIDGetAppTokenFailed)
	}

	apiURL := getOneIDAPIBaseURL(ctx) + "/openapi/v3/contacts/departments/" + deptID
	respBody, err := oneIDAPICall(ctx, http.MethodPatch, apiURL, token, fields)
	if err != nil {
		return err
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgOneIDParseUpdateDeptResponse)
	}
	if result.Code != 0 {
		return hcommon.I18nError(i18n.MsgOneIDUpdateDeptError, result.Code, result.Msg)
	}
	return nil
}

// OneIDDeleteDepartment 删除 OneID 部门。
// DELETE /openapi/v3/contacts/departments/{department_id}
func OneIDDeleteDepartment(ctx context.Context, deptID string) error {
	token, err := getOneIDAppToken(ctx)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgOneIDGetAppTokenFailed)
	}

	apiURL := getOneIDAPIBaseURL(ctx) + "/openapi/v3/contacts/departments/" + deptID
	respBody, err := oneIDAPICall(ctx, http.MethodDelete, apiURL, token, nil)
	if err != nil {
		return err
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgOneIDParseDeleteDeptResponse)
	}
	if result.Code != 0 {
		return hcommon.I18nError(i18n.MsgOneIDDeleteDeptError, result.Code, result.Msg)
	}
	return nil
}

// ── 递归同步辅助函数 ────────────────────────────────────────────────────────────

// oneIDEnsureGroupHasDept 确保一个 manual 分组有对应的 OneID 部门。
// 如果该分组已有 source_ref（OneID dept ID），直接返回。
// 否则递归确保父组也有对应部门，然后在 OneID 创建本组的部门，
// 并将返回的 department_id 存入 source_ref。
func oneIDEnsureGroupHasDept(ctx context.Context, group *model.UserGroup) (string, error) {
	// 已有映射直接返回
	if group.SourceRef != "" {
		return group.SourceRef, nil
	}

	// 只处理 manual 分组（oneid_dept 分组理论上已有 source_ref）
	if group.Source != model.GroupSourceManual {
		return "", hcommon.I18nError(i18n.MsgOneIDNonManualGroupNoRef, group.ID, group.Source)
	}

	// 解析父部门 ID
	parentDeptID, err := oneIDResolveParentDeptID(ctx, group.ParentID)
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgOneIDResolveParentDeptFailed, group.ID)
	}

	// 创建 OneID 部门
	deptID, err := OneIDCreateDepartment(ctx, group.Name, parentDeptID)
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgOneIDCreateDeptForGroupFailed, group.ID, group.Name)
	}

	// 写回 source_ref
	if err := model.DB(ctx).Model(&model.UserGroup{}).
		Where("id = ?", group.ID).
		Update("source_ref", deptID).Error; err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgOneIDStoreSourceRefFailed, group.ID)
	}
	group.SourceRef = deptID

	slog.Info("[OneID] department created and linked to group",
		"group_id", group.ID, "group_name", group.Name, "dept_id", deptID, "parent_dept_id", parentDeptID)

	return deptID, nil
}

// oneIDResolveParentDeptID 解析一个分组在 OneID 中的父部门 ID。
// ParentID=0（顶级分组）→ 返回 OneID 根部门 ID。
// ParentID>0 → 递归确保父组有 OneID 部门，返回其 department_id。
func oneIDResolveParentDeptID(ctx context.Context, parentID uint) (string, error) {
	if parentID == 0 {
		// 顶级分组 → 父部门是 OneID 根部门
		token, err := getOneIDAppToken(ctx)
		if err != nil {
			return "", hcommon.I18nRichError(err, i18n.MsgOneIDGetAppTokenFailed)
		}
		rootID, err := getOneIDRootDepartmentID(ctx, token)
		if err != nil {
			return "", hcommon.I18nRichError(err, i18n.MsgOneIDGetRootDeptFailed)
		}
		return rootID, nil
	}

	// 查父组
	var parent model.UserGroup
	if err := model.DB(ctx).Where("id = ?", parentID).First(&parent).Error; err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgOneIDFindParentGroupFailed, parentID)
	}

	// 递归确保父组有 dept
	return oneIDEnsureGroupHasDept(ctx, &parent)
}

// ── 分组 CRUD 同步 Hook ─────────────────────────────────────────────────────────

// oneIDSyncCreateGroup 创建分组后调用：在 OneID 创建对应部门。
// 仅统一账号模式生效。失败时调用方应回滚本地创建。
func oneIDSyncCreateGroup(ctx context.Context, group *model.UserGroup) error {
	if !hcommon.IsUnifiedAccountMode(ctx) {
		return nil
	}
	_, err := oneIDEnsureGroupHasDept(ctx, group)
	return err
}

// oneIDSyncUpdateGroup 更新分组后调用：同步名称/父组变更到 OneID。
// 仅统一账号模式 + 分组有 source_ref 时生效。
func oneIDSyncUpdateGroup(ctx context.Context, group *model.UserGroup, nameChanged, parentChanged bool) error {
	if !hcommon.IsUnifiedAccountMode(ctx) {
		return nil
	}
	if group.SourceRef == "" {
		return nil // 没有对应的 OneID 部门，跳过
	}

	fields := map[string]interface{}{}
	if nameChanged {
		fields["department_name"] = group.Name
	}
	if parentChanged {
		parentDeptID, err := oneIDResolveParentDeptID(ctx, group.ParentID)
		if err != nil {
			return hcommon.I18nRichError(err, i18n.MsgOneIDResolveNewParentFailed)
		}
		fields["parent_id"] = parentDeptID
	}

	if len(fields) == 0 {
		return nil
	}

	if err := OneIDUpdateDepartment(ctx, group.SourceRef, fields); err != nil {
		return err
	}
	slog.Info("[OneID] department updated", "group_id", group.ID, "dept_id", group.SourceRef, "fields", fields)
	return nil
}

// oneIDSyncDeleteGroup 删除分组前调用：删除 OneID 部门。
// 仅统一账号模式 + 分组有 source_ref 时生效。
func oneIDSyncDeleteGroup(ctx context.Context, group *model.UserGroup) error {
	if !hcommon.IsUnifiedAccountMode(ctx) {
		return nil
	}
	if group.SourceRef == "" {
		return nil // 没有对应的 OneID 部门，跳过
	}

	if err := OneIDDeleteDepartment(ctx, group.SourceRef); err != nil {
		return err
	}
	slog.Info("[OneID] department deleted", "group_id", group.ID, "dept_id", group.SourceRef)
	return nil
}

// ── 创建用户时获取 department_ids ────────────────────────────────────────────────

// oneIDResolveDepartmentIDsForGroups 将 group_ids 列表转为 OneID department_ids。
// 对于有 source_ref 的分组直接使用；没有的先通过 oneIDEnsureGroupHasDept 创建。
// 返回去重后的 department_ids 列表。
func oneIDResolveDepartmentIDsForGroups(ctx context.Context, groupIDs []uint) ([]string, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}

	var groups []model.UserGroup
	if err := model.DB(ctx).Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgQueryGroupFailed)
	}

	seen := make(map[string]bool)
	var deptIDs []string

	for i := range groups {
		g := &groups[i]
		deptID, err := oneIDEnsureGroupHasDept(ctx, g)
		if err != nil {
			slog.Warn("[OneID] ensure dept for group failed, skip",
				"group_id", g.ID, "group_name", g.Name, "err", err)
			continue
		}
		if deptID != "" && !seen[deptID] {
			seen[deptID] = true
			deptIDs = append(deptIDs, deptID)
		}
	}

	return deptIDs, nil
}

// ── 成员变更同步 ─────────────────────────────────────────────────────────────────

// oneIDSyncAddUsersToDept 将用户添加到分组后，同步更新这些用户在 OneID 的部门归属。
// 对每个用户：获取其当前所有分组对应的 department_ids，全量更新到 OneID。
func oneIDSyncAddUsersToDept(ctx context.Context, userIDs []uint, group *model.UserGroup) error {
	if group.SourceRef == "" {
		return nil
	}

	// 查出这些用户的 one_id_sub
	var users []model.User
	if err := model.DB(ctx).Where("id IN ? AND one_id_sub IS NOT NULL AND one_id_sub != ''", userIDs).Find(&users).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgOneIDQueryUsersFailed)
	}

	for _, u := range users {
		if u.OneIDSub == nil || *u.OneIDSub == "" {
			continue
		}
		// 查该用户当前所属的所有分组（有 source_ref 的），收集 department_ids
		var memberGroups []model.UserGroup
		if err := model.DB(ctx).
			Joins("JOIN user_group_members ON user_group_members.user_group_id = user_groups.id").
			Where("user_group_members.user_id = ? AND user_groups.source_ref != ''", u.ID).
			Find(&memberGroups).Error; err != nil {
			slog.Warn("[OneID] query user groups for dept sync failed", "user_id", u.ID, "err", err)
			continue
		}

		deptIDs := make([]string, 0, len(memberGroups))
		for _, g := range memberGroups {
			if g.SourceRef != "" {
				deptIDs = append(deptIDs, g.SourceRef)
			}
		}
		// 确保新加入的分组的 dept 也在列表里
		found := false
		for _, d := range deptIDs {
			if d == group.SourceRef {
				found = true
				break
			}
		}
		if !found {
			deptIDs = append(deptIDs, group.SourceRef)
		}

		if err := OneIDUpdateUser(ctx, *u.OneIDSub, map[string]interface{}{
			"department_ids": deptIDs,
		}); err != nil {
			return hcommon.I18nRichError(err, i18n.MsgOneIDUpdateUserDeptsFailed, *u.OneIDSub)
		}
		slog.Info("[OneID] user department synced after add to group",
			"union_id", *u.OneIDSub, "dept_ids", deptIDs)
	}
	return nil
}

// oneIDSyncRemoveUsersFromDept 将用户从分组移除后，同步更新这些用户在 OneID 的部门归属。
// 对每个用户：获取其当前剩余分组对应的 department_ids（不含被移除的分组），全量更新到 OneID。
func oneIDSyncRemoveUsersFromDept(ctx context.Context, userIDs []uint, group *model.UserGroup) error {
	if group.SourceRef == "" {
		return nil
	}

	var users []model.User
	if err := model.DB(ctx).Where("id IN ? AND one_id_sub IS NOT NULL AND one_id_sub != ''", userIDs).Find(&users).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgOneIDQueryUsersFailed)
	}

	for _, u := range users {
		if u.OneIDSub == nil || *u.OneIDSub == "" {
			continue
		}
		// 查该用户当前所属的所有分组（有 source_ref 的），收集 department_ids
		// 注意：此时本地成员关系已经移除了，所以查出来的就是剩余的
		var memberGroups []model.UserGroup
		if err := model.DB(ctx).
			Joins("JOIN user_group_members ON user_group_members.user_group_id = user_groups.id").
			Where("user_group_members.user_id = ? AND user_groups.source_ref != ''", u.ID).
			Find(&memberGroups).Error; err != nil {
			slog.Warn("[OneID] query user groups for dept sync failed", "user_id", u.ID, "err", err)
			continue
		}

		deptIDs := make([]string, 0, len(memberGroups))
		for _, g := range memberGroups {
			if g.SourceRef != "" {
				deptIDs = append(deptIDs, g.SourceRef)
			}
		}

		// 如果用户没有任何部门了，需要放回根部门（OneID 不允许用户无部门）
		if len(deptIDs) == 0 {
			token, err := getOneIDAppToken(ctx)
			if err == nil {
				if rootID, rootErr := getOneIDRootDepartmentID(ctx, token); rootErr == nil && rootID != "" {
					deptIDs = []string{rootID}
				}
			}
		}

		if err := OneIDUpdateUser(ctx, *u.OneIDSub, map[string]interface{}{
			"department_ids": deptIDs,
		}); err != nil {
			return hcommon.I18nRichError(err, i18n.MsgOneIDUpdateUserDeptsFailed, *u.OneIDSub)
		}
		slog.Info("[OneID] user department synced after remove from group",
			"union_id", *u.OneIDSub, "dept_ids", deptIDs)
	}
	return nil
}
