package controller

import (
	"net/http"
	"sort"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// HandleGetMyUserGroups 查询当前用户所在的所有用户组（含完整成员列表）
// GET /user-groups/mine
func HandleGetMyUserGroups(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user, err := RequestUser(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgUnauthorized))
		return
	}

	groups, err := model.GetUserGroups(r.Context(), user.ID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	type memberItem struct {
		UserID   uint   `json:"user_id"`
		Username string `json:"username"`
	}
	type groupItem struct {
		ID          uint         `json:"id"`
		Name        string       `json:"name"`
		FullPath    string       `json:"full_path"`  // 🆕 v6.13：分组全路径（如"研发中心/后端组"）
		Source      string       `json:"source"`     // 🆕 v6.13：分组来源 manual / oneid_dept
		IsMain      bool         `json:"is_main"`    // 🆕 v6.13：是否是当前用户的 oneid_dept 主部门（manual 分组恒 false）
		CreatedAt   string       `json:"created_at"` // 🆕 v6.13：分组创建时间（UTC RFC3339）
		Description string       `json:"description"`
		MemberCount int64        `json:"member_count"`
		Members     []memberItem `json:"members"`
	}

	// 收集所有组 ID，批量查询成员，避免 N+1 查询
	groupIDs := make([]uint, len(groups))
	for i, g := range groups {
		groupIDs[i] = g.ID
	}
	membersMap, err := model.GetGroupMembersByGroupIDs(r.Context(), groupIDs)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	// 🆕 批量查当前用户的 is_main 标记(仅 oneid_dept 源有真实值,manual 恒 false)。
	// 一条 SQL 取出当前用户在所有 groupIDs 中命中 is_main=true 的组。
	isMainSet := map[uint]struct{}{}
	if len(groupIDs) > 0 {
		var mainRows []model.UserGroupMember
		if err := model.DB(r.Context()).
			Select("user_group_id").
			Where("user_id = ? AND user_group_id IN ? AND is_main = ?", user.ID, groupIDs, true).
			Find(&mainRows).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
			return
		}
		for _, m := range mainRows {
			isMainSet[m.UserGroupID] = struct{}{}
		}
	}

	items := make([]groupItem, len(groups))
	for i, g := range groups {
		members := membersMap[g.ID]
		mItems := make([]memberItem, len(members))
		for j, m := range members {
			mItems[j] = memberItem{
				UserID:   m.UserID,
				Username: m.Username,
			}
		}
		// manual 分组不暴露 is_main 概念，oneid_dept 才读成员行标记
		isMain := false
		if g.Source == model.GroupSourceOneIDDept {
			_, isMain = isMainSet[g.ID]
		}
		items[i] = groupItem{
			ID:          g.ID,
			Name:        g.Name,
			FullPath:    g.FullPath,
			Source:      g.Source,
			IsMain:      isMain,
			CreatedAt:   g.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			Description: g.Description,
			MemberCount: int64(len(members)),
			Members:     mItems,
		}
	}

	// 排序：主部门最优先 → full_path 层级浅到深 → 同层级按创建时间
	sort.Slice(items, func(i, j int) bool {
		// 主部门优先
		if items[i].IsMain != items[j].IsMain {
			return items[i].IsMain
		}
		// full_path 按 "/" 分割数（层级浅的在前）
		depthI := strings.Count(items[i].FullPath, "/")
		depthJ := strings.Count(items[j].FullPath, "/")
		if depthI != depthJ {
			return depthI < depthJ
		}
		// 同层级按创建时间
		return items[i].CreatedAt < items[j].CreatedAt
	})

	jsonOK(w, map[string]interface{}{
		"ok":     true,
		"groups": items,
	})
}
