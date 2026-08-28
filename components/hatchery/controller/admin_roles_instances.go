package controller

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// validRoleSyncStatuses 所有合法的 role_sync_status 值。
var validRoleSyncStatuses = map[string]bool{
	"":                           true,
	model.RoleSyncStatusPending:  true,
	model.RoleSyncStatusUpdating: true,
	model.RoleSyncStatusUpdated:   true,
	model.RoleSyncStatusFailed:    true,
}

// HandleAdminRoleInstances 列出绑定指定角色的实例 + 同步状态。
// GET /admin/roles/instances?role_id=N&role_sync_status=...&search=...&page=1&page_size=20
//
// 返回结构（详见 docs/role-switch-frontend.md）：
//
//	{
//	  ok: true,
//	  data: {
//	    role: { id, name, version },
//	    page, page_size, total,
//	    items: [{instance_id, ..., role_version, role_sync_status}]
//	  }
//	}
func HandleAdminRoleInstances(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	role, statusCode, err := loadAdminRoleByQuery(r)
	if err != nil {
		writeError(w, r, statusCode, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	statuses, err := parseRoleSyncStatuses(r.URL.Query().Get("role_sync_status"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	page, pageSize := parsePagination(r)

	rows, total := queryInstancesByRolePaginated(r.Context(), role.ID, statuses, search, page, pageSize)

	jsonOK(w, map[string]interface{}{
		"role": map[string]interface{}{
			"id":      role.ID,
			"name":    role.Name,
			"version": role.Version,
		},
		"page":      page,
		"page_size": pageSize,
		"total":     total,
		"items":     enrichRoleInstances(r.Context(), rows),
	})
}

// loadAdminRoleByQuery 从 URL ?role_id= 加载并校验角色存在。
// 第二个返回值表示 HTTP 状态码（400 缺参 / 404 角色不存在）。
func loadAdminRoleByQuery(r *http.Request) (*model.OpenClawRole, int, error) {
	idStr := r.URL.Query().Get("role_id")
	if idStr == "" {
		return nil, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRoleDistributeRoleIDRequired)
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		return nil, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRoleDistributeRoleIDRequired)
	}
	var role model.OpenClawRole
	if err := model.DB(r.Context()).First(&role, id).Error; err != nil {
		return nil, http.StatusNotFound, hcommon.I18nRichError(err, i18n.MsgRoleNotFound)
	}
	return &role, 0, nil
}

// parseRoleSyncStatuses 支持逗号分隔多值。
// 空 / "all" → nil（不加 WHERE）；单值/多值 → 校验后返回切片。
func parseRoleSyncStatuses(v string) ([]string, error) {
	v = strings.TrimSpace(v)
	if v == "" || v == "all" {
		return nil, nil
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if !validRoleSyncStatuses[p] {
			return nil, hcommon.I18nError(i18n.MsgRoleSyncStatusInvalid)
		}
		result = append(result, p)
	}
	return result, nil
}

// queryInstancesByRolePaginated 用 SQL LIMIT/OFFSET 分页查询实例。
// 返回 (rows, total)。
func queryInstancesByRolePaginated(
	ctx context.Context,
	roleID uint,
	statuses []string,
	search string,
	page, pageSize int,
) ([]model.Instance, int64) {
	q := model.DB(ctx).Model(&model.Instance{}).Where("role_id = ?", roleID)
	if len(statuses) > 0 {
		q = q.Where("role_sync_status IN ?", statuses)
	}
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("name LIKE ? OR instance_id LIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		slog.Error("[RoleInstances] COUNT 失败", "role_id", roleID, "error", err)
		return nil, 0
	}

	var rows []model.Instance
	if err := q.Order("id DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&rows).Error; err != nil {
		slog.Error("[RoleInstances] 查询失败", "role_id", roleID, "error", err)
		return nil, total
	}
	return rows, total
}

// roleInstanceItem 是返回给前端的实例条目结构。
type roleInstanceItem struct {
	InstanceID     uint            `json:"instance_id"`
	CVMInstanceID  string          `json:"cvm_instance_id"`
	InstanceName   string          `json:"instance_name"`
	UserID         uint            `json:"user_id"`
	Username       string          `json:"username"`
	UserGroups     []roleInstGroup `json:"user_groups"`
	GroupID        uint            `json:"group_id"`
	GroupName      string          `json:"group_name"`
	RoleVersion    string          `json:"role_version"`
	RoleSyncStatus string          `json:"role_sync_status"`
}

type roleInstGroup struct {
	GroupID   uint   `json:"group_id"`
	GroupName string `json:"group_name"`
}

// enrichRoleInstances 批量补充用户名 / 用户分组 / 实例分组，直接读取 role_sync_status 字段。
// 用户视角的"分组"对应 user_groups 数组（一个用户可能在多个分组）。
func enrichRoleInstances(ctx context.Context, rows []model.Instance) []roleInstanceItem {
	if len(rows) == 0 {
		return []roleInstanceItem{}
	}

	userIDs, instanceGroupIDs := collectEnrichIDs(rows)
	userNameMap := loadUsernameMap(ctx, userIDs)
	userGroupMap := loadUserGroupMap(ctx, userIDs)
	groupNameMap := loadGroupNameMap(ctx, instanceGroupIDs)

	items := make([]roleInstanceItem, 0, len(rows))
	for _, inst := range rows {
		items = append(items, buildRoleInstanceItem(inst, userNameMap, userGroupMap, groupNameMap))
	}
	return items
}

// collectEnrichIDs 从实例切片中收集需要批量加载的 user_id 和 group_id 集合。
func collectEnrichIDs(rows []model.Instance) ([]uint, []uint) {
	userSet := make(map[uint]bool, len(rows))
	groupSet := make(map[uint]bool, len(rows))
	for _, inst := range rows {
		if inst.UserID > 0 {
			userSet[inst.UserID] = true
		}
		if inst.GroupID > 0 {
			groupSet[inst.GroupID] = true
		}
	}
	return setKeys(userSet), setKeys(groupSet)
}

func setKeys(s map[uint]bool) []uint {
	ks := make([]uint, 0, len(s))
	for k := range s {
		ks = append(ks, k)
	}
	return ks
}

func loadUsernameMap(ctx context.Context, userIDs []uint) map[uint]string {
	out := make(map[uint]string)
	if len(userIDs) == 0 {
		return out
	}
	var users []model.User
	model.DB(ctx).Where("id IN ?", userIDs).Find(&users)
	for _, u := range users {
		out[u.ID] = u.Username
	}
	return out
}

func loadUserGroupMap(ctx context.Context, userIDs []uint) map[uint][]model.UserGroup {
	out := make(map[uint][]model.UserGroup)
	if len(userIDs) == 0 {
		return out
	}
	if m, err := model.GetUserGroupsByUserIDs(ctx, userIDs); err == nil {
		return m
	} else {
		slog.Error("[RoleInstances] 批量查询用户分组失败", "error", err)
	}
	return out
}

func loadGroupNameMap(ctx context.Context, groupIDs []uint) map[uint]string {
	out := make(map[uint]string)
	if len(groupIDs) == 0 {
		return out
	}
	var groups []model.UserGroup
	model.DB(ctx).Where("id IN ?", groupIDs).Find(&groups)
	for _, g := range groups {
		out[g.ID] = g.Name
	}
	return out
}

// buildRoleInstanceItem 把单个 Instance 行装配为返回给前端的对象。
// role_sync_status 直接读取 Instance.RoleSyncStatus 字段（4 态状态机落库）。
func buildRoleInstanceItem(
	inst model.Instance,
	userNameMap map[uint]string,
	userGroupMap map[uint][]model.UserGroup,
	groupNameMap map[uint]string,
) roleInstanceItem {
	item := roleInstanceItem{
		InstanceID:     inst.ID,
		CVMInstanceID:  inst.InstanceId,
		InstanceName:   inst.Name,
		UserID:         inst.UserID,
		Username:       userNameMap[inst.UserID],
		GroupID:        inst.GroupID,
		GroupName:      groupNameMap[inst.GroupID],
		RoleVersion:    inst.DistributedRoleVersion,
		RoleSyncStatus: inst.RoleSyncStatus,
		UserGroups:     []roleInstGroup{},
	}
	for _, g := range userGroupMap[inst.UserID] {
		item.UserGroups = append(item.UserGroups, roleInstGroup{GroupID: g.ID, GroupName: g.Name})
	}
	return item
}
