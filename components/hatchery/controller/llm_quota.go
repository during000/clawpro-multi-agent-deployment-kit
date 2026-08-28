package controller

import (
	"net/http"
	"strconv"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// HandleQuotaData returns the current user's usage data as JSON.
// Supports the same query parameters as /admin/usage/data (except user_id is forced to current user).
// 可选 group_id 参数：按 agent 绑定的分组过滤用量，并返回该组的配额策略值。
func HandleQuotaData(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	start, end := parseDateRange(r)
	groupBy := parseGroupBySet(r.URL.Query().Get("group_by"))
	delete(groupBy, "user") // user dimension not applicable

	orderBy := r.URL.Query().Get("order_by")
	if orderBy == "" {
		orderBy = "total_tokens"
	}
	if orderBy != "total_tokens" && orderBy != "request_count" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidOrderBy))
		return
	}
	groupID := parseUint(r.URL.Query().Get("group_id"))

	result, err := queryUsageData(r.Context(), usageDataParams{
		Start:         start,
		End:           end,
		GroupBy:       groupBy,
		FilterUserID:  uint64(user.ID),
		FilterModelID: parseUint(r.URL.Query().Get("ai_model_id")),
		// instance_id 允许传 DB 主键数字或 CVM ID 字符串（如 ins-xxx）。
		// 双参数兼容：优先 id（DB 主键），否则退化到 instance_id。
		FilterInstanceID: resolveInstancePKFromIDOrParam(r.Context(), r.URL.Query().Get("id"), r.URL.Query().Get("instance_id"), user.ID),
		OrderBy:          orderBy,
		FilterGroupID:    groupID,
		// 兼容场景：用户最初无分组创建 agent，之后被加入分组 X，
		// 在 X 视图下需要看到这些旧 agent（group_id=0）的用量。
		// 仅用户端启用，管理端按分组维度统计不启用以免污染聚合口径。
		IncludeUserUngrouped: groupID > 0,
		OrderDesc:            r.URL.Query().Get("order") == "desc",
	})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryUsageDataFailed))
		return
	}

	siteConfig := model.GetSiteConfig(r.Context())
	_, quotaPeriod := model.EffectiveGlobalTokenQuotaLegacyFields(siteConfig.GlobalTokenQuotaDay, siteConfig.GlobalTokenQuotaPeriod, siteConfig.GlobalTokenQuotaRules)
	globalQuotaRules := siteConfig.ResolvedGlobalTokenQuotaRules()
	quotaRules := resolveEffectiveUserTokenQuotaRules(r.Context(), *user, uint(groupID))
	quotaDay := model.TokenQuotaDayFromRules(quotaRules)

	jsonOK(w, map[string]interface{}{
		"quota_day":                 quotaDay,
		"quota_period":              quotaPeriod,
		"token_quota_rules":         quotaRules,
		"token_quota_usages":        userTokenQuotaUsagesCompat(r.Context(), user.ID, uint(groupID), quotaRules, groupID > 0),
		"global_token_quota_rules":  globalQuotaRules,
		"global_token_quota_usages": globalTokenQuotaUsages(r.Context(), 0, globalQuotaRules),
		"start_date":                result.StartDate,
		"end_date":                  result.EndDate,
		"group_by":                  result.GroupBy,
		"rows":                      result.Rows,
	})
}

// HandleQuotaLogs returns paginated LLMUsageLog records for the current user.
// Supports the same query parameters as /admin/usage/logs (except user_id is forced to current user).
// 可选 group_id 参数：按 agent 绑定的分组过滤明细记录。
func HandleQuotaLogs(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	start, end := parseDateRange(r)
	endNext := end.Add(24 * time.Hour)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	q := model.DB(r.Context()).Model(&model.LLMUsageLog{}).
		Where("user_id = ? AND created_at >= ? AND created_at < ?", user.ID, start, endNext)
	if filterModelID := parseUint(r.URL.Query().Get("ai_model_id")); filterModelID > 0 {
		q = q.Where("ai_model_id = ?", filterModelID)
	}
	// instance_id 允许传 DB 主键数字或 CVM ID 字符串（如 ins-xxx）。
	// 双参数兼容：优先 id（DB 主键），否则退化到 instance_id。
	if filterInstanceID := resolveInstancePKFromIDOrParam(r.Context(), r.URL.Query().Get("id"), r.URL.Query().Get("instance_id"), user.ID); filterInstanceID > 0 {
		q = q.Where("instance_id = ?", filterInstanceID)
	}
	if groupID := parseUint(r.URL.Query().Get("group_id")); groupID > 0 {
		// 用户端兼容：把该用户名下 group_id=0 的"无分组创建的旧 agent"日志一并展示，
		// 避免老 agent 用量在分组视图下"消失"。详见 HandleQuotaData 的注释。
		q = q.Where("group_id IN ?", []uint64{groupID, 0})
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryRecordCountFailed))
		return
	}

	var logs []model.LLMUsageLog
	qb := q.Order("created_at DESC")
	if pageSize > 0 {
		if page < 1 {
			page = 1
		}
		qb = qb.Offset((page - 1) * pageSize).Limit(pageSize)
	}
	if err := qb.Find(&logs).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryUsageLogsFailed))
		return
	}

	type usageLogRow struct {
		ID                     uint      `json:"id"`
		Provider               string    `json:"provider"`
		Model                  string    `json:"model"`
		PromptTokens           int       `json:"prompt_tokens"`
		CompletionTokens       int       `json:"completion_tokens"`
		TotalTokens            int       `json:"total_tokens"`
		PromptCacheReadTokens  int       `json:"prompt_cache_read_tokens"`
		PromptCacheWriteTokens int       `json:"prompt_cache_write_tokens"`
		StatusCode             int       `json:"status_code"`
		Latency                int       `json:"latency"`
		CreatedAt              time.Time `json:"created_at"`
	}
	rows := make([]usageLogRow, len(logs))
	for i, log := range logs {
		rows[i] = usageLogRow{
			ID:                     log.ID,
			Provider:               log.Provider,
			Model:                  log.Model,
			PromptTokens:           log.PromptTokens,
			CompletionTokens:       log.CompletionTokens,
			TotalTokens:            log.TotalTokens,
			PromptCacheReadTokens:  log.PromptCacheReadTokens,
			PromptCacheWriteTokens: log.PromptCacheWriteTokens,
			StatusCode:             log.StatusCode,
			Latency:                log.Latency,
			CreatedAt:              log.CreatedAt,
		}
	}

	jsonOK(w, map[string]interface{}{
		"start_date": start.Format("2006-01-02"),
		"end_date":   end.Format("2006-01-02"),
		"page":       page,
		"page_size":  pageSize,
		"total":      total,
		"logs":       rows,
	})
}
