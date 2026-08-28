package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// openclaw_stale_instances.go — 用户端 5 个端点
//
//   POST /openclaw/stale-instances/rebind      用户自迁分组
//   POST /openclaw/stale-instances/initiate    发起同组移交（target_username）
//   POST /openclaw/stale-instances/cancel      原 owner 取消移交
//   POST /openclaw/stale-instances/accept      接收方同意
//   POST /openclaw/stale-instances/reject      接收方拒绝

// userRebindRequest 用户自迁请求体。
type userRebindRequest struct {
	ID            uint `json:"id"`
	TargetGroupID uint `json:"target_group_id"`
}

// HandleUserStaleInstancesRebind 用户端自迁到自己的某个分组（仅当 allow_migrate=true）。
// 成功后 instance.group_id := target_group_id，清除标，开机。
func HandleUserStaleInstancesRebind(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	var req userRebindRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	var inst model.Instance
	if err := model.DB(r.Context()).First(&inst, req.ID).Error; err != nil {
		writeError(w, r, http.StatusNotFound, ErrInstanceNotFound)
		return
	}
	if inst.UserID != user.ID {
		writeError(w, r, http.StatusForbidden, ErrForbidden)
		return
	}
	// 必须带 allow_migrate 且 pending_user_action
	if ok, _ := model.HasInstanceFlag(r.Context(), inst.ID, model.InstanceFlagPendingUserAction); !ok {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRebindNotPendingUserAction))
		return
	}
	if ok, _ := model.HasInstanceFlag(r.Context(), inst.ID, model.InstanceFlagAllowMigrate); !ok {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgRebindNotAllowedByAdmin))
		return
	}

	// target_group_id 必须 ∈ 当前用户的 group_ids（含 0 = 未分组的特殊情况：仅当用户已无任何分组时）
	userGroups, _ := loadUserGroupIDs(r.Context(), user.ID)
	if req.TargetGroupID == 0 {
		if len(userGroups) > 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRebindTargetGroupZeroButUserHasGroups, userGroups))
			return
		}
	} else if !uintInSlice(req.TargetGroupID, userGroups) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRebindTargetGroupNotInUserGroups, req.TargetGroupID, userGroups))
		return
	}

	oldGroupID := inst.GroupID
	err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Instance{}).Where("id = ?", inst.ID).
			Update("group_id", req.TargetGroupID).Error; err != nil {
			return err
		}
		if err := writeICGRTx(tx, &inst, oldGroupID, req.TargetGroupID, user.ID, user.ID,
			model.ICGRActionUserRebind, model.ICGRActorUser, user.ID, model.ICGRTriggerUserSelf, ""); err != nil {
			return err
		}
		return clearStaleFlagsTx(tx, inst.ID)
	})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	go startInstanceCloud(detachStaleCtx(r), inst.InstanceId)
	_ = model.CreateSuccessNotification(r.Context(), user.ID, inst.ID, inst.Name,
		model.NotifyTypeInstanceMigrated, i18n.T(r.Context(), i18n.NotifTitleAgentMigrated),
		i18n.T(r.Context(), i18n.NotifMsgMigratedByUser, inst.Name, lookupGroupName(r.Context(), req.TargetGroupID)))
	jsonOK(w, map[string]interface{}{"ok": true})
}

// handoverInitiateRequest 用户发起移交请求体（按用户名而非 user_id）。
type handoverInitiateRequest struct {
	ID             uint   `json:"id"`
	TargetUsername string `json:"target_username"`
}

// HandleUserStaleInstancesHandoverInitiate 用户端发起同组移交。
func HandleUserStaleInstancesHandoverInitiate(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	var req handoverInitiateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == 0 || req.TargetUsername == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	var inst model.Instance
	if err := model.DB(r.Context()).First(&inst, req.ID).Error; err != nil {
		writeError(w, r, http.StatusNotFound, ErrInstanceNotFound)
		return
	}
	if inst.UserID != user.ID {
		writeError(w, r, http.StatusForbidden, ErrForbidden)
		return
	}
	// 实例 group_id != 0
	if inst.GroupID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgHandoverInstanceUngrouped))
		return
	}
	if ok, _ := model.HasInstanceFlag(r.Context(), inst.ID, model.InstanceFlagAllowSameGroupHandover); !ok {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgHandoverNotAllowedByAdmin))
		return
	}
	// 解析 username → user_id
	var target model.User
	if err := model.DB(r.Context()).Where("username = ?", req.TargetUsername).First(&target).Error; err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgHandoverTargetUserNotFound, req.TargetUsername))
		return
	}
	if target.ID == user.ID {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgHandoverTargetIsSelf))
		return
	}
	inGroup, err := userInGroup(r.Context(), target.ID, inst.GroupID)
	if err != nil || !inGroup {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgHandoverTargetNotInInstanceGroup, req.TargetUsername))
		return
	}

	now := time.Now()
	err = model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"handover_target_user_id":      target.ID,
			"handover_rejected_by_user_id": uint(0),
			"handover_initiated_at":        &now,
		}
		if err := tx.Model(&model.Instance{}).Where("id = ?", inst.ID).Updates(updates).Error; err != nil {
			return err
		}
		return writeICGRTx(tx, &inst, inst.GroupID, inst.GroupID, user.ID, target.ID,
			model.ICGRActionUserHandoverInit, model.ICGRActorUser, user.ID, model.ICGRTriggerUserSelf, "")
	})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	_ = model.CreateNotification(r.Context(), user.ID, inst.ID, inst.Name,
		model.NotifyTypeInstanceHandoverInitiated, i18n.T(r.Context(), i18n.NotifTitleHandoverInitiated),
		i18n.T(r.Context(), i18n.NotifMsgHandoverInitiatedToInitiator, target.Username, inst.Name))
	_ = model.CreateNotification(r.Context(), target.ID, inst.ID, inst.Name,
		model.NotifyTypeInstanceHandoverReceived, i18n.T(r.Context(), i18n.NotifTitleHandoverReceived),
		i18n.T(r.Context(), i18n.NotifMsgHandoverInitiatedToTarget, user.Username, inst.Name))

	jsonOK(w, map[string]interface{}{"ok": true})
}

// handoverIDOnlyRequest 仅 id 入参的 handover 请求体（cancel/accept/reject 共用）。
type handoverIDOnlyRequest struct {
	ID uint `json:"id"`
}

// HandleUserStaleInstancesHandoverCancel 原 owner 取消移交。
func HandleUserStaleInstancesHandoverCancel(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	var req handoverIDOnlyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	var inst model.Instance
	if err := model.DB(r.Context()).First(&inst, req.ID).Error; err != nil {
		writeError(w, r, http.StatusNotFound, ErrInstanceNotFound)
		return
	}
	if inst.UserID != user.ID {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgHandoverCancelNotOwner,
			lookupUsername(r.Context(), inst.UserID), user.Username))
		return
	}
	if inst.HandoverTargetUserID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgHandoverNoActiveHandover))
		return
	}

	targetUID := inst.HandoverTargetUserID
	err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Instance{}).Where("id = ?", inst.ID).Updates(map[string]interface{}{
			"handover_target_user_id":      uint(0),
			"handover_rejected_by_user_id": uint(0),
			"handover_initiated_at":        nil,
		}).Error; err != nil {
			return err
		}
		return writeICGRTx(tx, &inst, inst.GroupID, inst.GroupID, user.ID, user.ID,
			model.ICGRActionUserHandoverCancel, model.ICGRActorUser, user.ID, model.ICGRTriggerUserSelf, "")
	})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	_ = model.CreateNotification(r.Context(), user.ID, inst.ID, inst.Name,
		model.NotifyTypeInstanceHandoverCancelled, i18n.T(r.Context(), i18n.NotifTitleHandoverCancelled),
		i18n.T(r.Context(), i18n.NotifMsgHandoverCancelledToInitiator, inst.Name, lookupUsername(r.Context(), targetUID)))
	_ = model.CreateNotification(r.Context(), targetUID, inst.ID, inst.Name,
		model.NotifyTypeInstanceHandoverCancelled, i18n.T(r.Context(), i18n.NotifTitleHandoverCancelledByOther),
		i18n.T(r.Context(), i18n.NotifMsgHandoverCancelledToTarget, user.Username, inst.Name))
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleUserStaleInstancesHandoverAccept 接收方同意：换 user_id + 开机 + 清标。
func HandleUserStaleInstancesHandoverAccept(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	var req handoverIDOnlyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	var inst model.Instance
	if err := model.DB(r.Context()).First(&inst, req.ID).Error; err != nil {
		writeError(w, r, http.StatusNotFound, ErrInstanceNotFound)
		return
	}
	if inst.HandoverTargetUserID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgHandoverNoActiveHandover))
		return
	}
	if inst.HandoverTargetUserID != user.ID {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgHandoverAcceptNotTarget,
			lookupUsername(r.Context(), inst.HandoverTargetUserID), user.Username))
		return
	}

	oldUserID := inst.UserID
	err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Instance{}).Where("id = ?", inst.ID).Updates(map[string]interface{}{
			"user_id":                      user.ID,
			"handover_target_user_id":      uint(0),
			"handover_rejected_by_user_id": uint(0),
			"handover_initiated_at":        nil,
		}).Error; err != nil {
			return err
		}
		if err := writeICGRTx(tx, &inst, inst.GroupID, inst.GroupID, oldUserID, user.ID,
			model.ICGRActionUserHandoverAccept, model.ICGRActorUser, user.ID, model.ICGRTriggerUserSelf, ""); err != nil {
			return err
		}
		return clearStaleFlagsTx(tx, inst.ID)
	})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	go startInstanceCloud(detachStaleCtx(r), inst.InstanceId)
	_ = model.CreateSuccessNotification(r.Context(), oldUserID, inst.ID, inst.Name,
		model.NotifyTypeInstanceHandoverAccepted, i18n.T(r.Context(), i18n.NotifTitleHandoverAccepted),
		i18n.T(r.Context(), i18n.NotifMsgHandoverAcceptedToOwner, user.Username, inst.Name))
	_ = model.CreateSuccessNotification(r.Context(), user.ID, inst.ID, inst.Name,
		model.NotifyTypeInstanceHandoverAccepted, i18n.T(r.Context(), i18n.NotifTitleHandoverAcceptedReceived),
		i18n.T(r.Context(), i18n.NotifMsgHandoverAcceptedToReceiver, lookupUsername(r.Context(), oldUserID), inst.Name))
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleUserStaleInstancesHandoverReject 接收方拒绝：保持关机，写 rejected_by。
func HandleUserStaleInstancesHandoverReject(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	var req handoverIDOnlyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	var inst model.Instance
	if err := model.DB(r.Context()).First(&inst, req.ID).Error; err != nil {
		writeError(w, r, http.StatusNotFound, ErrInstanceNotFound)
		return
	}
	if inst.HandoverTargetUserID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgHandoverNoActiveHandover))
		return
	}
	if inst.HandoverTargetUserID != user.ID {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgHandoverRejectNotTarget,
			lookupUsername(r.Context(), inst.HandoverTargetUserID), user.Username))
		return
	}

	originalOwner := inst.UserID
	err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Instance{}).Where("id = ?", inst.ID).Updates(map[string]interface{}{
			"handover_target_user_id":      uint(0),
			"handover_rejected_by_user_id": user.ID,
			"handover_initiated_at":        nil,
		}).Error; err != nil {
			return err
		}
		return writeICGRTx(tx, &inst, inst.GroupID, inst.GroupID, originalOwner, originalOwner,
			model.ICGRActionUserHandoverReject, model.ICGRActorUser, user.ID, model.ICGRTriggerUserSelf, "")
	})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	_ = model.CreateNotificationWithCategory(r.Context(), originalOwner, inst.ID, inst.Name,
		model.NotifyTypeInstanceHandoverRejected, model.NotifCategoryError,
		i18n.T(r.Context(), i18n.NotifTitleHandoverRejected),
		i18n.T(r.Context(), i18n.NotifMsgHandoverRejectedToOwner, user.Username, inst.Name), nil)
	_ = model.CreateNotification(r.Context(), user.ID, inst.ID, inst.Name,
		model.NotifyTypeInstanceHandoverRejected, i18n.T(r.Context(), i18n.NotifTitleHandoverRejectedConfirmed),
		i18n.T(r.Context(), i18n.NotifMsgHandoverRejectedToRejecter, lookupUsername(r.Context(), originalOwner), inst.Name))
	jsonOK(w, map[string]interface{}{"ok": true})
}

// ──────────────────────────────────────────────
// 辅助
// ──────────────────────────────────────────────

// detachStaleCtx 给后台 goroutine 用：保留 TenantSnapshot，断开 request 取消信号。
func detachStaleCtx(r *http.Request) context.Context {
	return hcommon.DetachContext(r.Context())
}

// lookupUsername 反查 user_id → username；命中不到或 id=0 返回空串（调用方
// 通常用于错误消息展示，空串场景由消息模板 %q 自然渲染为 ""）。
func lookupUsername(ctx context.Context, userID uint) string {
	if userID == 0 {
		return ""
	}
	var u model.User
	if err := model.DB(ctx).Select("username").Where("id = ?", userID).First(&u).Error; err != nil {
		return ""
	}
	return u.Username
}
