package controller

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

// 🆕 v6.13：允许用户同时属于多个分组。
// 旧的 errUserSingleGroupOnly / validateUsersCanJoinOnlyTargetGroup 已移除。

// mapGroupErrToHTTP 把 model.UserGroup 业务错误映射到 HTTP 状态码。
func mapGroupErrToHTTP(err error) int {
	switch {
	case errors.Is(err, model.ErrUserGroupNotFound),
		errors.Is(err, model.ErrParentGroupNotFound):
		return http.StatusNotFound
	case errors.Is(err, model.ErrUserGroupLimitExceeded),
		errors.Is(err, model.ErrInvalidGroupName),
		errors.Is(err, model.ErrGroupNameConflict),
		errors.Is(err, model.ErrMaxGroupDepthExceeded),
		errors.Is(err, model.ErrFullPathTooLong),
		errors.Is(err, model.ErrParentCycleDetected),
		errors.Is(err, model.ErrManualCannotUnderOneIDDept),
		errors.Is(err, model.ErrOneIDDeptReadonly),
		errors.Is(err, model.ErrGroupToBeDeletedReadonly):
		return http.StatusBadRequest
	case errors.Is(err, model.ErrGroupHasDependencies):
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

// parseOptionalUintParent 解析 JSON 中 parent_id 的三态语义（🆕 v6.7）：
//   - 字段缺失  → present=false（不换父 / 不指定）
//   - 字段为 null → present=true, value=0（根组）
//   - 字段为 数字  → present=true, value=number
func parseOptionalUintParent(raw json.RawMessage) (present bool, value uint, err error) {
	if len(raw) == 0 {
		return false, 0, nil
	}
	if string(raw) == "null" {
		return true, 0, nil
	}
	var n uint
	if decodeErr := json.Unmarshal(raw, &n); decodeErr != nil {
		return false, 0, hcommon.I18nError(i18n.MsgParentIDFormatError)
	}
	return true, n, nil
}

// HandleAdminListUserGroups 分页列出用户组（含成员数）。
// GET /admin/user-groups?parent_id=&source=&q=&page=&page_size=
//
// 🆕 v6：支持按 parent_id / source / q 过滤。
func HandleAdminListUserGroups(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	page, pageSize := parsePagination(r)
	opts := model.ListUserGroupsOpts{
		Source:   strings.TrimSpace(r.URL.Query().Get("source")),
		Query:    strings.TrimSpace(r.URL.Query().Get("q")),
		Page:     page,
		PageSize: pageSize,
	}
	if pidStr := strings.TrimSpace(r.URL.Query().Get("parent_id")); pidStr != "" {
		pid, err := strconv.ParseUint(pidStr, 10, 64)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamFormatError, "parent_id"))
			return
		}
		pidUint := uint(pid)
		opts.ParentID = &pidUint
	}

	groups, total, err := model.ListUserGroupsExt(r.Context(), opts)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	type groupItem struct {
		ID            uint   `json:"id"`
		Name          string `json:"name"`
		Description   string `json:"description"`
		ParentID      uint   `json:"parent_id"`
		Depth         int    `json:"depth"`
		FullPath      string `json:"full_path"`
		Source        string `json:"source"`
		SourceRef     string `json:"source_ref,omitempty"`
		ToBeDeleted   bool   `json:"to_be_deleted"`
		Readonly      bool   `json:"readonly"`
		MemberCount   int64  `json:"member_count"`
		InstanceCount int64  `json:"instance_count"` // 🆕 v6.13：本组 + 所有子孙组创建的 agent 总数（通过 group_closure 聚合）
		CreatedAt     string `json:"created_at"`
	}

	// 批量查询成员数，避免 N+1
	groupIDs := make([]uint, len(groups))
	for i, g := range groups {
		groupIDs[i] = g.ID
	}
	memberCounts, err := model.CountGroupMembersBatch(model.DB(r.Context()), groupIDs)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 🆕 批量查询每个分组（含所有子孙）的 agent 总数：
	//   SELECT c.ancestor_id, COUNT(*) FROM group_closure c
	//     INNER JOIN instances i ON i.group_id = c.descendant_id
	//     WHERE c.ancestor_id IN (...) AND i.group_id <> 0 AND i.deleted_at IS NULL
	//     GROUP BY c.ancestor_id
	// closure 自指行（ancestor=descendant, depth=0）已包含"本组"的 case，
	// 所以单条聚合 SQL 即可覆盖"本组 + 子孙"全部命中。
	// ⚠️ 必须显式加 `i.deleted_at IS NULL`：原始 Table() 查询会绕开 GORM 的
	//    自动软删过滤，把已销毁的实例也计入。
	instanceCounts := map[uint]int64{}
	if len(groupIDs) > 0 {
		type instCountRow struct {
			AncestorID uint  `gorm:"column:ancestor_id"`
			Count      int64 `gorm:"column:count"`
		}
		var rows []instCountRow
		if err := model.DB(r.Context()).Table("group_closure AS c").
			Select("c.ancestor_id AS ancestor_id, COUNT(*) AS count").
			Joins("INNER JOIN instances i ON i.group_id = c.descendant_id").
			Where("c.ancestor_id IN ? AND i.group_id <> 0 AND i.deleted_at IS NULL", groupIDs).
			Group("c.ancestor_id").
			Scan(&rows).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryGroupInstanceTotalFailed))
			return
		}
		for _, r := range rows {
			instanceCounts[r.AncestorID] = r.Count
		}
	}

	items := make([]groupItem, len(groups))
	for i, g := range groups {
		items[i] = groupItem{
			ID:            g.ID,
			Name:          g.Name,
			Description:   g.Description,
			ParentID:      g.ParentID,
			Depth:         g.Depth,
			FullPath:      g.FullPath,
			Source:        g.Source,
			SourceRef:     g.SourceRef,
			ToBeDeleted:   g.ToBeDeleted,
			Readonly:      g.Readonly(),
			MemberCount:   memberCounts[g.ID],
			InstanceCount: instanceCounts[g.ID],
			CreatedAt:     g.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
	}

	jsonOK(w, map[string]interface{}{
		"ok":     true,
		"total":  total,
		"groups": items,
	})
}

// HandleAdminCreateUserGroup 创建 manual 分组（支持 parent_id）。
// POST /admin/user-groups/create
func HandleAdminCreateUserGroup(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		ParentID    json.RawMessage `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	req.Name = strings.TrimSpace(req.Name)

	_, parentID, err := parseOptionalUintParent(req.ParentID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	group, err := model.CreateUserGroupWithOpts(r.Context(), req.Name, req.Description, parentID, model.GroupSourceManual, "")
	if err != nil {
		var re *hcommon.RichError
		if errors.As(err, &re) {
			if len(re.Unwrap()) > 0 && isDuplicateKeyError(re.Unwrap()[0]) {
				writeError(w, r, http.StatusBadRequest, model.ErrGroupNameConflict)
				return
			}
			writeError(w, r, mapGroupErrToHTTP(err), re)
			return
		}
		writeError(w, r, mapGroupErrToHTTP(err), hcommon.I18nRichError(err, i18n.MsgCreateUserGroupFailed))
		return
	}

	// 统一账号模式：同步创建 OneID 部门
	if syncErr := oneIDSyncCreateGroup(r.Context(), group); syncErr != nil {
		// 回滚：删除刚创建的本地组
		_ = model.DB(r.Context()).Delete(&model.UserGroup{}, group.ID)
		slog.Error("[OneID] sync create dept failed, rolled back local group",
			"group_id", group.ID, "err", syncErr)
		writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDCreateDeptFailed, syncErr))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok": true,
		"group": map[string]interface{}{
			"id":          group.ID,
			"name":        group.Name,
			"description": group.Description,
			"parent_id":   group.ParentID,
			"depth":       group.Depth,
			"full_path":   group.FullPath,
			"source":      group.Source,
		},
	})
}

// HandleAdminUpdateUserGroup 修改分组（name / description / parent_id 三者可选）。
// POST /admin/user-groups/update
//
// 🆕 v6.7：parent_id 三态：
//   - 字段缺失 → 不换父
//   - null     → 移到根
//   - number   → 挂到该父组下
//
// 换父时事务内删旧祖先链 / 插新祖先链 / 递归更新子树 depth + full_path。
// 用分布式锁 TryLock("group_move") 串行化。
func HandleAdminUpdateUserGroup(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req struct {
		ID          uint            `json:"id"`
		Name        *string         `json:"name"`
		Description *string         `json:"description"`
		ParentID    json.RawMessage `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserGroupIDRequired))
		return
	}
	if req.Name != nil {
		cleaned := strings.TrimSpace(*req.Name)
		req.Name = &cleaned
	}

	parentPresent, parentValue, err := parseOptionalUintParent(req.ParentID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	opts := model.UpdateGroupOpts{
		Name:        req.Name,
		Description: req.Description,
	}
	if parentPresent {
		pid := parentValue
		opts.NewParentIDPtr = &pid
	}

	if parentPresent {
		lock, lockErr := model.TryLock(r.Context(), "group_move")
		if lockErr != nil {
			writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgGroupMoveInProgress))
			return
		}
		defer lock.Release()
	}

	group, err := model.UpdateUserGroupExt(r.Context(), req.ID, opts)
	if err != nil {
		if isDuplicateKeyError(err) {
			writeError(w, r, http.StatusBadRequest, model.ErrGroupNameConflict)
			return
		}
		writeError(w, r, mapGroupErrToHTTP(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 统一账号模式：同步名称/父组变更到 OneID 部门
	nameChanged := req.Name != nil
	if syncErr := oneIDSyncUpdateGroup(r.Context(), group, nameChanged, parentPresent); syncErr != nil {
		slog.Error("[OneID] sync update dept failed", "group_id", group.ID, "err", syncErr)
		writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDUpdateDeptFailed, syncErr))
		return
	}

	// stale-instances v1.0：换父成功后，对子树内所有非空 group_id 的实例写占位记录 + 发通知。
	// 不打 stale_group 标记（场景 D 的 group_id 仍正确，配置由 Resolver 自动按新链解析）。
	// 写库失败仅打日志，不阻断响应（处理是事后补救型，不应影响主链路）。
	if parentPresent {
		go markStaleForSubtree(hcommon.DetachContext(r.Context()), group.ID)
	}

	jsonOK(w, map[string]interface{}{
		"ok": true,
		"group": map[string]interface{}{
			"id":          group.ID,
			"name":        group.Name,
			"description": group.Description,
			"parent_id":   group.ParentID,
			"depth":       group.Depth,
			"full_path":   group.FullPath,
			"source":      group.Source,
		},
	})
}

// HandleAdminDeleteUserGroup 删除分组（软删 + 清 closure + 清成员）。
// POST /admin/user-groups/delete
func HandleAdminDeleteUserGroup(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req struct {
		ID uint `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserGroupIDRequired))
		return
	}

	if ok, err := model.CanDeleteUserGroup(r.Context(), req.ID); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	} else if !ok {
		writeError(w, r, http.StatusConflict, model.ErrGroupHasDependencies)
		return
	}

	lock, lockErr := model.TryLock(r.Context(), "group_move")
	if lockErr != nil {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgGroupOperationInProgress))
		return
	}
	defer lock.Release()

	// 统一账号模式：先删 OneID 部门
	if hcommon.IsUnifiedAccountMode(r.Context()) {
		var group model.UserGroup
		if model.DB(r.Context()).Where("id = ?", req.ID).First(&group).Error == nil && group.SourceRef != "" {
			if syncErr := oneIDSyncDeleteGroup(r.Context(), &group); syncErr != nil {
				slog.Error("[OneID] sync delete dept failed", "group_id", req.ID, "err", syncErr)
				writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDDeleteDeptFailed, syncErr))
				return
			}
		}
	}

	if err := model.DeleteUserGroup(r.Context(), req.ID); err != nil {
		writeError(w, r, mapGroupErrToHTTP(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 分组删除后自动清理 CLS 采集范围中的对应记录
	if err := model.RemoveCLSCollectScopeGroups(r.Context(), []uint{req.ID}); err != nil {
		slog.Error("[UserGroup] 删除分组后清理 CLS scope 失败", "group_id", req.ID, "error", err)
		jsonOK(w, map[string]interface{}{
			"ok":      true,
			"warning": i18n.T(r.Context(), i18n.MsgGroupDeletedButCLSCleanupFail),
		})
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleAdminGetGroupDeleteImpact 查询删除影响报告。
// GET /admin/user-groups/delete-impact?id=
//
// 🆕 v6.8：路径从 {id}/delete-impact 改为 delete-impact?id=。
// 🆕 v6.12 P1：blockers.policy_configs / security_group_configs / scoped_configs 固定为空数组。
func HandleAdminGetGroupDeleteImpact(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	// 解析 group_ids（必传，逗号分隔）
	idsStr := strings.TrimSpace(r.URL.Query().Get("group_ids"))
	if idsStr == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserGroupIDRequired))
		return
	}

	parts := strings.Split(idsStr, ",")
	var groupIDs []uint
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseUint(p, 10, 64)
		if err != nil || id == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserGroupIDFormatError))
			return
		}
		groupIDs = append(groupIDs, uint(id))
	}
	if len(groupIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserGroupIDRequired))
		return
	}

	// 校验所有组存在性
	if err := usergroup.ValidateGroupIDs(r.Context(), groupIDs); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPartialUserGroupsNotFound))
		return
	}

	// 批量查询
	impacts := make([]interface{}, 0, len(groupIDs))
	for _, gid := range groupIDs {
		impact, err := usergroup.GetDeleteImpact(r.Context(), gid)
		if err != nil {
			var re *hcommon.RichError
			if errors.As(err, &re) {
				writeError(w, r, http.StatusBadRequest, re)
			} else {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgFailedToGetDeleteImpact))
			}
			return
		}
		impacts = append(impacts, impact)
	}

	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"results": impacts,
	})
}

// HandleAdminSetGroupMembers 全量替换组内成员。
// POST /admin/user-groups/members/set
func HandleAdminSetGroupMembers(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req struct {
		ID      uint   `json:"id"`
		UserIDs []uint `json:"user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserGroupIDRequired))
		return
	}
	if len(req.UserIDs) > model.MaxMembersPerUserGroup {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupMemberLimitExceeded))
		return
	}

	group, err := model.GroupByID(r.Context(), req.ID)
	if err != nil {
		writeError(w, r, mapGroupErrToHTTP(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if group.Readonly() {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgGroupReadonlyMembers))
		return
	}

	if err := model.SetGroupMembers(r.Context(), req.ID, req.UserIDs); err != nil {
		if errors.Is(err, model.ErrMemberCountExceeded) || errors.Is(err, model.ErrInvalidUserID) {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		writeError(w, r, mapGroupErrToHTTP(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 统一账号模式：全量设置成员后，同步所有成员的 OneID 部门归属
	if hcommon.IsUnifiedAccountMode(r.Context()) && group.SourceRef != "" {
		if err := oneIDSyncAddUsersToDept(r.Context(), req.UserIDs, group); err != nil {
			slog.Error("[OneID] sync set members dept failed", "group_id", req.ID, "err", err)
			writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDSyncUserDeptFailed, err))
			return
		}
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleAdminAddGroupMembers 批量添加成员（幂等）。
// POST /admin/user-groups/members/add
func HandleAdminAddGroupMembers(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req struct {
		ID      uint   `json:"id"`
		UserIDs []uint `json:"user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserGroupIDRequired))
		return
	}

	group, err := model.GroupByID(r.Context(), req.ID)
	if err != nil {
		writeError(w, r, mapGroupErrToHTTP(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if group.Readonly() {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgGroupReadonlyMembers))
		return
	}

	if err := model.AddGroupMembers(r.Context(), req.ID, req.UserIDs); err != nil {
		if errors.Is(err, model.ErrAddMemberWouldExceed) || errors.Is(err, model.ErrInvalidUserID) {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		writeError(w, r, mapGroupErrToHTTP(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 统一账号模式：同步用户的 OneID 部门归属
	if hcommon.IsUnifiedAccountMode(r.Context()) && group.SourceRef != "" {
		if err := oneIDSyncAddUsersToDept(r.Context(), req.UserIDs, group); err != nil {
			slog.Error("[OneID] sync add users to dept failed", "group_id", req.ID, "err", err)
			writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDSyncUserDeptFailed, err))
			return
		}
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleAdminRemoveGroupMembers 批量移除成员。
// POST /admin/user-groups/members/remove
func HandleAdminRemoveGroupMembers(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req struct {
		ID      uint   `json:"id"`
		UserIDs []uint `json:"user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserGroupIDRequired))
		return
	}

	group, err := model.GroupByID(r.Context(), req.ID)
	if err != nil {
		writeError(w, r, mapGroupErrToHTTP(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if group.Readonly() {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgGroupReadonlyMembers))
		return
	}

	if err := model.RemoveGroupMembers(r.Context(), req.ID, req.UserIDs); err != nil {
		writeError(w, r, mapGroupErrToHTTP(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 统一账号模式：同步移除用户的 OneID 部门归属
	if hcommon.IsUnifiedAccountMode(r.Context()) && group.SourceRef != "" {
		if err := oneIDSyncRemoveUsersFromDept(r.Context(), req.UserIDs, group); err != nil {
			slog.Error("[OneID] sync remove users from dept failed", "group_id", req.ID, "err", err)
			writeError(w, r, http.StatusBadGateway, hcommon.I18nError(i18n.MsgOneIDSyncRemoveUserDeptFailed, err))
			return
		}
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleAdminGetGroupsByUsers 批量查询多个用户所在的所有用户组。
// GET /admin/user-groups/groups-by-users?user_ids=1,2,3
func HandleAdminGetGroupsByUsers(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	type groupItem struct {
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		FullPath    string `json:"full_path"`
		Source      string `json:"source"`
	}

	userIDsStr := r.URL.Query().Get("user_ids")
	if userIDsStr == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "user_ids"))
		return
	}

	parts := strings.Split(userIDsStr, ",")
	userIDs := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		uid, err := strconv.ParseUint(p, 10, 64)
		if err != nil || uid == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserIDInvalidFormat, p))
			return
		}
		userIDs = append(userIDs, uint(uid))
	}
	if len(userIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "user_ids"))
		return
	}
	if len(userIDs) > 100 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserIDsTooMany))
		return
	}

	groupsByUser, err := model.GetUserGroupsByUserIDs(r.Context(), userIDs)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	resultMap := make(map[string][]groupItem, len(userIDs))
	for _, uid := range userIDs {
		groups := groupsByUser[uid]
		items := make([]groupItem, len(groups))
		for i, g := range groups {
			items[i] = groupItem{
				ID:          g.ID,
				Name:        g.Name,
				Description: g.Description,
				FullPath:    g.FullPath,
				Source:      g.Source,
			}
		}
		resultMap[strconv.FormatUint(uint64(uid), 10)] = items
	}

	jsonOK(w, map[string]interface{}{
		"ok":   true,
		"data": resultMap,
	})
}

// HandleGetGroupAssociatedModels 查询用户组关联的模型列表
// GET /admin/user-groups/associated-models?group_id=N
func HandleGetGroupAssociatedModels(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	groupIDStr := r.URL.Query().Get("group_id")
	if groupIDStr == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "group_id"))
		return
	}
	groupID, err := strconv.ParseUint(groupIDStr, 10, 64)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamFormatError, "group_id"))
		return
	}

	modelIDs, err := model.GetModelsAssociatedWithGroup(r.Context(), uint(groupID))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	if len(modelIDs) == 0 {
		jsonOK(w, map[string]interface{}{
			"ok":     true,
			"count":  0,
			"models": []interface{}{},
		})
		return
	}

	var models []model.AIModel
	if err := model.DB(r.Context()).Where("id IN ?", modelIDs).Find(&models).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryModelDetailFailed))
		return
	}

	type modelItem struct {
		ID       uint   `json:"id"`
		Provider string `json:"provider"`
		ModelID  string `json:"model_id"`
	}
	items2 := make([]modelItem, len(models))
	for i, m := range models {
		items2[i] = modelItem{
			ID:       m.ID,
			Provider: m.Provider,
			ModelID:  m.ModelID,
		}
	}

	jsonOK(w, map[string]interface{}{
		"ok":     true,
		"count":  len(items2),
		"models": items2,
	})
}
