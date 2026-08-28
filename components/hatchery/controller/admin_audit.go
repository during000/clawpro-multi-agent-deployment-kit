package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

type adminAuditFilters struct {
	UserID       uint
	HasUserID    bool
	Username     string
	Fuzzy        bool
	Action       string
	ResourceID   string
	StartUnix    int64
	HasStartTime bool
	EndUnix      int64
	HasEndTime   bool
}

func parseAdminAuditFilters(r *http.Request) (adminAuditFilters, error) {
	query := r.URL.Query()
	filters := adminAuditFilters{
		Username:   query.Get("username"),
		Fuzzy:      query.Get("fuzzy") == "1",
		Action:     query.Get("action"),
		ResourceID: query.Get("resource_id"),
	}

	if query.Has("user_id") {
		userID, err := strconv.ParseUint(query.Get("user_id"), 10, strconv.IntSize)
		if err != nil {
			return adminAuditFilters{}, fmt.Errorf("parse user_id: %w", err)
		}
		filters.UserID = uint(userID)
		filters.HasUserID = true
	}

	if start, err := strconv.ParseInt(query.Get("start_time"), 10, 64); err == nil {
		filters.StartUnix = start
		filters.HasStartTime = true
	}
	if end, err := strconv.ParseInt(query.Get("end_time"), 10, 64); err == nil {
		filters.EndUnix = end
		filters.HasEndTime = true
	}

	return filters, nil
}

func applyAdminAuditFilters(query *gorm.DB, filters adminAuditFilters) *gorm.DB {
	if filters.HasUserID {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.Username != "" {
		if filters.Fuzzy {
			query = query.Where("username LIKE ?", "%"+filters.Username+"%")
		} else {
			query = query.Where("username = ?", filters.Username)
		}
	}
	// 支持按 action 前缀筛选（如 "agent_bridge_" 可筛选所有 Agent-Bridge 审计记录）。
	if filters.Action != "" {
		query = query.Where("action LIKE ?", filters.Action+"%")
	}
	// 支持按 resource_id 精确筛选（如按实例 ID 查询某台机器的所有审计记录）。
	if filters.ResourceID != "" {
		query = query.Where("resource_id = ?", filters.ResourceID)
	}
	if filters.HasStartTime {
		query = query.Where("created_at >= ?", time.Unix(filters.StartUnix, 0))
	}
	if filters.HasEndTime {
		query = query.Where("created_at < ?", time.Unix(filters.EndUnix, 0))
	}
	return query
}

func HandleAdminAudit(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	filters, err := parseAdminAuditFilters(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "user_id"))
		return
	}

	page, pageSize := parsePagination(r, 1000)
	offset := (page - 1) * pageSize

	handleAdminAuditListWithTotal(w, r, filters, page, pageSize, offset)
}

func handleAdminAuditListWithTotal(
	w http.ResponseWriter,
	r *http.Request,
	filters adminAuditFilters,
	page int,
	pageSize int,
	offset int,
) {
	countQuery := applyAdminAuditFilters(model.DB(r.Context()).Model(&model.AuditLog{}), filters)
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgDatabaseOperationFailed))
		return
	}

	var logs []model.AuditLog
	listQuery := applyAdminAuditFilters(model.DB(r.Context()).Model(&model.AuditLog{}), filters)
	if err := listQuery.Order("id desc").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgDatabaseOperationFailed))
		return
	}

	totalPages := int(total / int64(pageSize))
	if total%int64(pageSize) != 0 {
		totalPages++
	}
	jsonOK(w, map[string]interface{}{
		"logs":        logs,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}
