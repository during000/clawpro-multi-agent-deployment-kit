package controller

import (
	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"net/http"
)

// HandleAdminMultiGroupStats §8 /admin/users/multi-group-stats
//
// GET /admin/users/multi-group-stats
//
// 返回多归属用户统计 + top 5 示例，驱动顶部常驻 Banner。
func HandleAdminMultiGroupStats(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	stats, err := usergroup.GetMultiGroupStats(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok":                true,
		"total_users":       stats.TotalUsers,
		"multi_group_users": stats.MultiGroupUsers,
		"ungrouped_users":   stats.UngroupedUsers,
		"top_examples":      stats.TopExamples,
	})
}
