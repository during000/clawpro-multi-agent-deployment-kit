package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// roleDistributeMaxInstances 单次批量分发的实例数量上限。
// 与 MCP 分发 (admin_mcp_distribute.go) 对齐为 500。
const roleDistributeMaxInstances = 500

// HandleDistributeRole 管理端把角色当前最新版本批量推送到选中的实例。
// POST /admin/roles/distribute?id=<role_id>
//
//	请求 body: { "instance_ids": [<uint>, ...] }
//	响应: { "ok": true, "data": { "accepted": <int>, "skipped": [{instance_id, reason}] } }
//
// 校验：
//   - role_id 必填且 > 0
//   - len(instance_ids) ∈ [1, 500]
//   - 实例必须 role_id == 目标角色 id（否则跳 role_mismatch）
//   - 实例 distributed_role_version 必须 < 角色当前 version（否则跳 already_updated）
func HandleDistributeRole(w http.ResponseWriter, r *http.Request) {
	handleDistributeRole(w, r, defaultStatusResolver)
}

func handleDistributeRole(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	roleID, err := parseDistributeRoleID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	instanceIDs, err := parseDistributeInstanceIDs(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// operator：AdminToken 场景无 session，OperatorID=0；有 session 则取用户 ID
	operatorID := currentOperatorID(r)

	result, err := applyRoleToInstances(r.Context(), instanceIDs, roleID, applyModeDistribute, resolver,
		applyOptions{OperatorID: operatorID, Source: model.RoleRecordSourceDistribute})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	jsonOK(w, result)
}

// parseDistributeRoleID 解析 URL 上的 ?id= 参数（角色 ID，必填且 > 0）。
func parseDistributeRoleID(r *http.Request) (uint, error) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		return 0, hcommon.I18nError(i18n.MsgRoleDistributeRoleIDRequired)
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		return 0, hcommon.I18nError(i18n.MsgRoleDistributeRoleIDRequired)
	}
	return uint(id), nil
}

// parseDistributeInstanceIDs 解析 body 中的 instance_ids 数组并校验上限。
func parseDistributeInstanceIDs(r *http.Request) ([]uint, error) {
	var req struct {
		InstanceIDs []uint `json:"instance_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, hcommon.I18nError(i18n.MsgBadRequest)
	}
	if len(req.InstanceIDs) == 0 {
		return nil, hcommon.I18nError(i18n.MsgRoleDistributeIDsEmpty)
	}
	if len(req.InstanceIDs) > roleDistributeMaxInstances {
		return nil, hcommon.I18nError(i18n.MsgRoleDistributeMaxInstances, roleDistributeMaxInstances)
	}
	return req.InstanceIDs, nil
}

// currentOperatorID 从请求上下文取当前操作者 user.ID。
// 无 session（如 AdminToken 场景）返回 0。
func currentOperatorID(r *http.Request) uint {
	user, err := getLoginUser(r)
	if err != nil || user == nil {
		return 0
	}
	return user.ID
}
