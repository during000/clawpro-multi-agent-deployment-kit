package controller

import (
	"context"
	"encoding/json"
	"net/http"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// HandleSwitchRole 用户端切换单个实例的角色入口（含 role_id=0 即移除角色）。
// POST /openclaw/switch-role
//
//	请求 body: { "instance_id": <uint>, "role_id": <uint> }
//	响应: { "ok": true, "data": { "accepted": <int>, "skipped": [{instance_id, reason}] } }
//
// 与批量接口契约一致：被跳过仍返回 200 + accepted=0 + skipped。
// owner 校验失败 / 实例不存在 → skipped reason="not_found"，不返回 4xx。
func HandleSwitchRole(w http.ResponseWriter, r *http.Request) {
	handleSwitchRole(w, r, defaultStatusResolver)
}

// handleSwitchRole 测试可注入 resolver 的小写实现。
func handleSwitchRole(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	req, err := parseSwitchRoleRequest(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// owner 校验：只有实例的所有者可以切换。失败按 not_found 处理（不暴露存在性）。
	if !instanceOwnedByUser(r.Context(), req.InstanceID, user.ID) {
		jsonOK(w, &ApplyResult{
			Skipped: []SkippedItem{{InstanceID: req.InstanceID, Reason: skipReasonNotFound}},
		})
		return
	}

	result, err := applyRoleToInstances(r.Context(), []uint{req.InstanceID}, req.RoleID, applyModeSwitch, resolver,
		applyOptions{OperatorID: user.ID, Source: model.RoleRecordSourceSwitch})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	jsonOK(w, result)
}

// switchRoleRequest 是 /openclaw/switch-role 的请求体。
type switchRoleRequest struct {
	InstanceID uint `json:"instance_id"`
	RoleID     uint `json:"role_id"`
}

// parseSwitchRoleRequest 解析并校验 switch-role 的请求体。
// instance_id 必填且 > 0；role_id 允许 0（表示移除角色）。
func parseSwitchRoleRequest(r *http.Request) (*switchRoleRequest, error) {
	var req switchRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, hcommon.I18nError(i18n.MsgBadRequest)
	}
	if req.InstanceID == 0 {
		return nil, hcommon.I18nError(i18n.MsgRoleSwitchInstanceIDInvalid)
	}
	return &req, nil
}

// instanceOwnedByUser 校验 instance 是否属于当前用户。
// 通过 ctx 的 identifier 自动隔离；只查存在性，不返回实例对象。
func instanceOwnedByUser(ctx context.Context, instanceID uint, userID uint) bool {
	var count int64
	model.DB(ctx).
		Model(&model.Instance{}).
		Where("id = ? AND user_id = ?", instanceID, userID).
		Count(&count)
	return count > 0
}
