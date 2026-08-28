package controller

import (
	"encoding/json"
	"net/http"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// validCategories 合法的消息类别值
var validCategories = map[string]bool{
	model.NotifCategorySuccess: true,
	model.NotifCategoryError:   true,
	model.NotifCategoryNotice:  true,
}

// HandleGetNotifications GET /openclaw/notifications - 获取通知列表
func HandleGetNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)

	page, pageSize := parsePagination(r)
	log.Info("[GetNotifications] 收到获取通知列表请求", "user_id", user.ID, "page", page, "page_size", pageSize)

	// 解析 is_read 参数：true=仅已读, false=仅未读, 不传=全部, 其他值报错
	var isRead *bool
	if isReadStr := strings.TrimSpace(r.URL.Query().Get("is_read")); isReadStr != "" {
		if isReadStr != "true" && isReadStr != "false" {
			log.Warn("[GetNotifications] is_read 参数值无效", "user_id", user.ID, "is_read", isReadStr)
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNotificationIsReadInvalid))
			return
		}
		v := isReadStr == "true"
		isRead = &v
	}

	// 解析 category 参数：success/error/notice，不传=全部，其他值报错
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	if category != "" && !validCategories[category] {
		log.Warn("[GetNotifications] category 参数值无效", "user_id", user.ID, "category", category)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNotificationCategoryInvalid))
		return
	}

	notifications, total, err := model.GetUserNotifications(ctx, user.ID, page, pageSize, isRead, category)
	if err != nil {
		log.Error("[GetNotifications] 查询通知失败", "user_id", user.ID, "page", page, "page_size", pageSize, "category", category, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgNotificationQueryFailed))
		return
	}
	log.Info("[GetNotifications] 查询通知成功", "user_id", user.ID, "returned", len(notifications), "total", total)

	jsonOK(w, map[string]interface{}{
		"notifications": notifications,
		"page":          page,
		"page_size":     pageSize,
		"total":         total,
	})
}

// HandleReadNotification POST /openclaw/notifications/read - 标记通知已读
// id=0 时标记所有通知为已读（可选 category 过滤）；id>0 时标记指定通知为已读
func HandleReadNotification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)

	var req struct {
		ID       uint   `json:"id"`
		Category string `json:"category"` // 可选，仅 id=0 时生效
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("[ReadNotification] 解析请求体失败", "user_id", user.ID, "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	log.Info("[ReadNotification] 收到标记已读请求", "user_id", user.ID, "id", req.ID, "category", req.Category)

	// 校验 category 合法性
	if req.Category != "" && !validCategories[req.Category] {
		log.Warn("[ReadNotification] category 参数值无效", "user_id", user.ID, "category", req.Category)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNotificationCategoryInvalid))
		return
	}

	if req.ID == 0 {
		// id=0：全部已读（可选按 category 过滤）
		if err := model.MarkAllNotificationsRead(ctx, user.ID, req.Category); err != nil {
			log.Error("[ReadNotification] 标记全部已读失败", "user_id", user.ID, "category", req.Category, "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgNotificationMarkAllFailed))
			return
		}
		log.Info("[ReadNotification] 标记全部已读成功", "user_id", user.ID, "category", req.Category)
	} else {
		// id>0：单条已读（忽略 category）
		if err := model.MarkNotificationRead(ctx, req.ID, user.ID); err != nil {
			log.Error("[ReadNotification] 标记单条已读失败", "user_id", user.ID, "id", req.ID, "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgNotificationMarkReadFailed))
			return
		}
		log.Info("[ReadNotification] 标记单条已读成功", "user_id", user.ID, "id", req.ID)
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleGetUnreadCount GET /openclaw/notifications/count - 获取未读通知数量（含分类计数）
func HandleGetUnreadCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)

	log.Info("[GetUnreadCount] 收到获取未读数量请求", "user_id", user.ID)

	total, byCategory, err := model.GetUnreadNotificationCountByCategory(ctx, user.ID)
	if err != nil {
		log.Error("[GetUnreadCount] 查询未读数量失败", "user_id", user.ID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgNotificationQueryUnread))
		return
	}
	log.Info("[GetUnreadCount] 查询未读数量成功", "user_id", user.ID, "total", total, "by_category", byCategory)

	jsonOK(w, map[string]interface{}{
		"unread_count":       total,
		"unread_by_category": byCategory,
	})
}

// HandleDeleteNotification POST /openclaw/notifications/delete - 删除通知
// 支持三种模式：
//   - id > 0:           删除单条通知
//   - ids 非空:          批量删除指定通知
//   - id = 0 且 ids 为空: 删除全部通知（可选 category 过滤）
func HandleDeleteNotification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		log.Warn("[DeleteNotification] 非法方法", "user_id", user.ID, "method", r.Method)
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		ID       uint   `json:"id"`
		IDs      []uint `json:"ids"`
		Category string `json:"category"` // 可选，仅全部删除时生效
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("[DeleteNotification] 解析请求体失败", "user_id", user.ID, "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	log.Info("[DeleteNotification] 收到删除通知请求", "user_id", user.ID, "id", req.ID, "ids_count", len(req.IDs), "category", req.Category)

	// 校验 category 合法性
	if req.Category != "" && !validCategories[req.Category] {
		log.Warn("[DeleteNotification] category 参数值无效", "user_id", user.ID, "category", req.Category)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNotificationCategoryInvalid))
		return
	}

	var affected int64
	var err error
	var mode string

	switch {
	case req.ID > 0:
		// 单条删除
		mode = "single"
		affected, err = model.DeleteNotification(ctx, req.ID, user.ID)
	case len(req.IDs) > 0:
		// 批量删除：去重 + 过滤无效 ID
		mode = "batch"
		seen := make(map[uint]bool, len(req.IDs))
		var uniqueIDs []uint
		for _, id := range req.IDs {
			if id > 0 && !seen[id] {
				seen[id] = true
				uniqueIDs = append(uniqueIDs, id)
			}
		}
		if len(uniqueIDs) == 0 {
			log.Info("[DeleteNotification] 批量删除无有效ID，跳过", "user_id", user.ID)
			jsonOK(w, map[string]interface{}{"ok": true, "deleted": 0})
			return
		}
		if len(uniqueIDs) > 100 {
			log.Warn("[DeleteNotification] 批量删除数量超限", "user_id", user.ID, "count", len(uniqueIDs))
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNotificationDeleteMax100))
			return
		}
		log.Info("[DeleteNotification] 批量删除", "user_id", user.ID, "unique_count", len(uniqueIDs))
		affected, err = model.DeleteNotifications(ctx, uniqueIDs, user.ID)
	default:
		// 全部删除（可选 category）
		mode = "all"
		affected, err = model.DeleteAllNotifications(ctx, user.ID, req.Category)
	}

	if err != nil {
		log.Error("[DeleteNotification] 删除通知失败", "user_id", user.ID, "mode", mode, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgNotificationDeleteFailed))
		return
	}
	log.Info("[DeleteNotification] 删除通知成功", "user_id", user.ID, "mode", mode, "affected", affected)

	jsonOK(w, map[string]interface{}{"ok": true, "deleted": affected})
}
