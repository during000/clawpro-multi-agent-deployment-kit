package controller

// ============================================================
// 本地 Agent 资源分发：前端 HTTP handler
//
// 路由前缀：/openclaw/local/*
//
// 设计文档：openspec/changes/local-agent-resources/design.md §B.5/B.6
// 接口文档：openspec/changes/local-agent-resources/teamai-api.md §3.5/3.6
//
// 本文件只包含前端用 HTTP handler：
//   - HandleSwitchUserLevelGroup: POST /openclaw/local/user-group
//   - list 视图辅助函数在 local_agent_diff.go (buildLocalAgentResourcesView)
// ============================================================

import (
	"encoding/json"
	"net/http"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// switchUserGroupRequest 前端切换用户级分组的请求体。
type switchUserGroupRequest struct {
	GroupID    uint `json:"group_id"`
	InstanceID uint `json:"instance_id"`
}

// HandleSwitchUserLevelGroup 前端切换用户级分组。
// POST /openclaw/local/user-group
//
// 行为：
//  1. 鉴权 requireLogin；解析 {group_id, instance_id}
//  2. 校验 instance 属于当前 user 且 source='local'
//  3. 校验 group_id 属于当前用户分组
//  4. 更新 instances.local_agent_resources.user_level.group_id
//  5. diffAndQueue(scope='user', workspace_path=”)
//  6. 返回 new_pending_count
func HandleSwitchUserLevelGroup(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	var req switchUserGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}
	defer r.Body.Close()

	if req.GroupID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "group_id"))
		return
	}
	if req.InstanceID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "instance_id"))
		return
	}

	ctx := r.Context()

	// 校验 instance 属于当前 user 且 source='local'
	var inst model.Instance
	if err := model.DB(ctx).Where("id = ? AND user_id = ? AND source = ?",
		req.InstanceID, user.ID, model.InstanceSourceLocal).First(&inst).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceNotFoundOrNoPerm))
		return
	}

	// 校验 group_id 属于当前用户分组
	if !computeGroupActive(ctx, user.ID, req.GroupID) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupNotFound))
		return
	}

	// 事务：更新 resources + diffAndQueue（复用 applyUserLevelGroupSwitch）
	var pendingCount int
	txErr := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		resources := deserializeLocalAgentResources(inst.LocalAgentResources)
		count, err := applyUserLevelGroupSwitch(ctx, tx, &inst, resources, req.GroupID)
		if err != nil {
			return err
		}
		pendingCount = count
		return nil
	})

	if txErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgInternalError))
		return
	}

	jsonOK(w, map[string]any{
		"ok":                true,
		"new_pending_count": pendingCount,
	})
}
