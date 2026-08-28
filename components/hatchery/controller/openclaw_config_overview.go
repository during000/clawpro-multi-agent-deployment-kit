package controller

import (
	"net/http"
	"strings"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

// HandleOpenClawConfigOverview 用户端接口：查询 agent 的分组配置总览
//
// GET /openclaw/config-overview?ids=1,2,3&keys=model,channel
// GET /openclaw/config-overview?group_ids=3,4&keys=model,channel
//
// 支持两种查询模式（二选一，group_ids 优先）：
//   - group_ids: 直接按分组 ID 查询配置（创建实例时尚无实例 ID 场景）
//   - ids: 按实例 ID 查询其绑定分组的配置
//
// 按 group_id 解析配置。相同 group_id 共享结果（去重查询）。
// group_id=0 返回全局默认配置。
func HandleOpenClawConfigOverview(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	// 解析 keys（可选）
	var keyFilter map[string]bool
	if keysStr := strings.TrimSpace(r.URL.Query().Get("keys")); keysStr != "" {
		keys := strings.Split(keysStr, ",")
		keyFilter = make(map[string]bool, len(keys))
		for _, k := range keys {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			if !isValidCategoryKey(k) {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidKey, k))
				return
			}
			keyFilter[k] = true
		}
	}

	// 判断查询模式：group_ids 优先，其次 ids
	groupIDsStr := strings.TrimSpace(r.URL.Query().Get("group_ids"))
	idsStr := strings.TrimSpace(r.URL.Query().Get("ids"))

	if groupIDsStr == "" && idsStr == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgIDsOrGroupIDsRequired))
		return
	}

	// 模式一：group_ids 直接按分组查询
	if groupIDsStr != "" {
		groupIDs, err := parseUintCSV(groupIDsStr)
		if err != nil || len(groupIDs) == 0 {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgBadRequestParamFormatError, "group_ids"))
			return
		}

		siteConfig := model.GetSiteConfig(r.Context())

		type groupOverview struct {
			GroupID    uint                             `json:"group_id"`
			Categories []usergroup.ConfigCategoryResult `json:"categories"`
		}

		results := make([]groupOverview, 0, len(groupIDs))
		seen := make(map[uint]struct{}, len(groupIDs))

		for _, gid := range groupIDs {
			// 去重：相同 group_id 只计算一次
			if _, ok := seen[gid]; ok {
				continue
			}
			seen[gid] = struct{}{}

			var categories []usergroup.ConfigCategoryResult
			if gid == 0 {
				categories = buildCategoriesForGroup(r.Context(), 0, nil, &siteConfig, keyFilter)
			} else {
				ancestors, err := model.ClosureAncestors(r.Context(), gid, true)
				if err != nil {
					writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgParseGroupAncestorChainFailed))
					return
				}
				categories = buildCategoriesForGroup(r.Context(), gid, ancestors, &siteConfig, keyFilter)
			}
			results = append(results, groupOverview{
				GroupID:    gid,
				Categories: categories,
			})
		}

		jsonOK(w, map[string]interface{}{
			"ok":      true,
			"results": results,
		})
		return
	}

	// 模式二：ids 按实例 ID 查询（原有逻辑）
	instanceIDs, err := parseUintCSV(idsStr)
	if err != nil || len(instanceIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidIDsFormat))
		return
	}

	// 查询实例，校验归属当前用户
	var instances []model.Instance
	if err := model.DB(r.Context()).Where("id IN ? AND user_id = ?", instanceIDs, user.ID).
		Select("id, group_id").Find(&instances).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryInstanceFailed))
		return
	}
	if len(instances) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNoInstancesForUser))
		return
	}

	// 按 group_id 去重，批量获取祖先链
	groupIDSet := make(map[uint]struct{})
	for _, inst := range instances {
		groupIDSet[inst.GroupID] = struct{}{}
	}

	siteConfig := model.GetSiteConfig(r.Context())

	// 预计算每个 group_id 的配置结果（含 group_id=0 的全局配置）
	type groupResult struct {
		Categories []usergroup.ConfigCategoryResult
	}
	groupConfigCache := make(map[uint]*groupResult, len(groupIDSet))

	for gid := range groupIDSet {
		var categories []usergroup.ConfigCategoryResult
		if gid == 0 {
			categories = buildCategoriesForGroup(r.Context(), 0, nil, &siteConfig, keyFilter)
		} else {
			ancestors, err := model.ClosureAncestors(r.Context(), gid, true)
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgParseGroupAncestorChainFailed))
				return
			}
			categories = buildCategoriesForGroup(r.Context(), gid, ancestors, &siteConfig, keyFilter)
		}
		groupConfigCache[gid] = &groupResult{Categories: categories}
	}

	// 构建响应：每个实例对应其 group 的配置
	type agentOverview struct {
		ID         uint                             `json:"id"`
		GroupID    uint                             `json:"group_id"`
		Categories []usergroup.ConfigCategoryResult `json:"categories"`
	}

	results := make([]agentOverview, 0, len(instances))
	for _, inst := range instances {
		cached := groupConfigCache[inst.GroupID]
		results = append(results, agentOverview{
			ID:         inst.ID,
			GroupID:    inst.GroupID,
			Categories: cached.Categories,
		})
	}

	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"results": results,
	})
}
