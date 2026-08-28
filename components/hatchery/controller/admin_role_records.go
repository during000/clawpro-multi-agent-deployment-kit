package controller

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// HandleAdminRoleRecords 分页查询角色下发记录（仅管理员 distribute 操作，排除用户 switch/create）。
//
// GET /admin/roles/records?instance_ids=1,2,3&role_ids=7,8&page=1&page_size=20
//
// - instance_ids 可选：逗号分隔的实例 ID 列表，不传则查全部实例
// - role_ids 可选：逗号分隔的角色 ID 列表，不传则查全部角色
// - instance_ids/role_ids 传了但全部值无效时返回 400
// - source 固定过滤 == distribute（管理员操作）
// - 按 id DESC 排序，支持分页
// - page_size=1 时取最新一条
// - 响应 items 中额外回填 operator_username（联表 users 表，非持久化字段）
func HandleAdminRoleRecords(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	page, pageSize := parsePagination(r)

	q := model.DB(r.Context()).Model(&model.RoleDistributionRecord{}).
		Where("source = ?", model.RoleRecordSourceDistribute)

	// instance_ids 可选：逗号分隔多实例过滤
	instanceIDs, err := parseUintIDsQuery(r, "instance_ids")
	if err != nil {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "instance_ids"))
		return
	}
	if len(instanceIDs) > 0 {
		q = q.Where("instance_id IN ?", instanceIDs)
	}

	// role_ids 可选：逗号分隔多角色过滤
	roleIDs, err := parseUintIDsQuery(r, "role_ids")
	if err != nil {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "role_ids"))
		return
	}
	if len(roleIDs) > 0 {
		q = q.Where("role_id IN ?", roleIDs)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		slog.Error("[RoleRecords] COUNT 失败", "error", err)
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	var rows []model.RoleDistributionRecord
	if err := q.Order("id DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&rows).Error; err != nil {
		slog.Error("[RoleRecords] 查询失败", "error", err)
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	// 批量回填 operator_username（联表 users 表，避免 N+1 查询）
	fillOperatorUsernames(r.Context(), rows)

	jsonOK(w, map[string]interface{}{
		"page":      page,
		"page_size": pageSize,
		"total":     total,
		"items":     rows,
	})
}

// fillOperatorUsernames 批量查询 users 表并回填 records 的 OperatorUsername 字段。
// operator_id=0 的记录保持空字符串；未找到对应用户（含软删除）的也保持空字符串。
func fillOperatorUsernames(ctx context.Context, rows []model.RoleDistributionRecord) {
	if len(rows) == 0 {
		return
	}
	seen := make(map[uint]bool)
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		if row.OperatorID == 0 || seen[row.OperatorID] {
			continue
		}
		seen[row.OperatorID] = true
		ids = append(ids, row.OperatorID)
	}
	if len(ids) == 0 {
		return
	}
	var users []model.User
	if err := model.DB(ctx).Select("id, username").Where("id IN ?", ids).Find(&users).Error; err != nil {
		slog.Warn("[RoleRecords] 回填 operator_username 失败", "error", err)
		return
	}
	nameByID := make(map[uint]string, len(users))
	for _, u := range users {
		nameByID[u.ID] = u.Username
	}
	for i := range rows {
		if name, ok := nameByID[rows[i].OperatorID]; ok {
			rows[i].OperatorUsername = name
		}
	}
}

// parseUintIDsQuery 解析逗号分隔的 uint ID 列表参数。
// 参数未传时返回 (nil, nil)；传了但全部值无效时返回 (nil, error)。
func parseUintIDsQuery(r *http.Request, paramName string) ([]uint, error) {
	idsStr := strings.TrimSpace(r.URL.Query().Get(paramName))
	if idsStr == "" {
		return nil, nil
	}

	seen := make(map[uint]bool)
	var result []uint
	for _, part := range strings.Split(idsStr, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseUint(part, 10, 64)
		if err != nil || id == 0 {
			continue // 跳过无效 ID
		}
		uid := uint(id)
		if !seen[uid] {
			seen[uid] = true
			result = append(result, uid)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("参数 %s 全部无效", paramName)
	}
	return result, nil
}
