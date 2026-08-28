package controller

import (
	"context"
	"encoding/json"
	"net/http"

	"gorm.io/gorm"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

// ──────────────────────────────────────────────
// 加法型资源 — 设置应用范围（通用 handler）
// ──────────────────────────────────────────────

// handleSetVisibility 通用加法型资源设置应用范围逻辑。
// configType: 绑定表的 config_type
// getResourceID: 从请求体 map 中提取资源 ID 的函数
// updateMainTable: 更新主表 visibility_type 的回调（在事务内执行，接收 tx）
func handleSetVisibility(w http.ResponseWriter, r *http.Request, configType string,
	getResourceID func(map[string]json.RawMessage) (uint, error),
	updateMainTable func(tx *gorm.DB, id uint, visType string) error) {

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var reqBody map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}

	// 解析 visibility_type
	var visibilityType string
	if vt, ok := reqBody["visibility_type"]; ok {
		json.Unmarshal(vt, &visibilityType)
	}
	if visibilityType != usergroup.VisibilityAll && visibilityType != usergroup.VisibilityGroup {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "visibility_type"))
		return
	}

	// 解析 group_ids
	var groupIDs []uint
	if gids, ok := reqBody["group_ids"]; ok {
		json.Unmarshal(gids, &groupIDs)
	}
	if visibilityType == usergroup.VisibilityGroup && len(groupIDs) == 0 {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "group_ids"))
		return
	}

	// 解析资源 ID
	resourceID, err := getResourceID(reqBody)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, hcommon.I18nError(i18n.MsgNotFound))
		return
	}

	// 校验 group_ids 存在性
	if visibilityType == usergroup.VisibilityGroup {
		if err := usergroup.ValidateGroupIDs(r.Context(), groupIDs); err != nil {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "group_ids"))
			return
		}
	}

	// 事务：更新绑定 + 更新主表
	tx := model.DB(r.Context()).Begin()
	if err := usergroup.SetVisibility(tx, configType, resourceID, visibilityType, groupIDs); err != nil {
		tx.Rollback()
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDatabaseOperationFailed))
		return
	}
	if updateMainTable != nil {
		if err := updateMainTable(tx, resourceID, visibilityType); err != nil {
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDatabaseOperationFailed))
			return
		}
	}
	tx.Commit()

	jsonOK(w, map[string]interface{}{"ok": true})
}

// ──────────────────────────────────────────────
// POST /admin/channels/visibility
// ──────────────────────────────────────────────

// HandleChannelVisibility 设置通道应用范围
func HandleChannelVisibility(w http.ResponseWriter, r *http.Request) {
	handleSetVisibility(w, r, usergroup.ConfigTypeChannel,
		func(body map[string]json.RawMessage) (uint, error) {
			raw, ok := body["channel_id"]
			if !ok {
				return 0, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "channel_id")
			}
			// 支持字符串（channel_id 标识）和 uint（DB ID）
			var channelID string
			if err := json.Unmarshal(raw, &channelID); err != nil {
				var id uint
				if err2 := json.Unmarshal(raw, &id); err2 == nil {
					return id, nil
				}
				return 0, err
			}
			var ch model.AIChannel
			if err := model.DB(r.Context()).Where("channel_id = ?", channelID).First(&ch).Error; err != nil {
				return 0, hcommon.I18nError(i18n.MsgChannelNotFound, channelID)
			}
			return ch.ID, nil
		},
		func(tx *gorm.DB, id uint, visType string) error {
			return tx.Model(&model.AIChannel{}).Where("id = ?", id).
				Update("visibility_type", visType).Error
		},
	)
}

// ──────────────────────────────────────────────
// POST /admin/mcp/visibility
// ──────────────────────────────────────────────

// ──────────────────────────────────────────────
// POST /admin/mcp/visibility
// ──────────────────────────────────────────────

// HandleMCPVisibility 设置 MCP 应用范围
func HandleMCPVisibility(w http.ResponseWriter, r *http.Request) {
	handleSetVisibility(w, r, usergroup.ConfigTypeMCP,
		func(body map[string]json.RawMessage) (uint, error) {
			raw, ok := body["mcp_id"]
			if !ok {
				return 0, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "mcp_id")
			}
			var id uint
			if err := json.Unmarshal(raw, &id); err != nil {
				return 0, err
			}
			var mcp model.McpServer
			if err := model.DB(r.Context()).First(&mcp, id).Error; err != nil {
				return 0, hcommon.I18nError(i18n.MsgMcpNotFound)
			}
			return id, nil
		},
		func(tx *gorm.DB, id uint, visType string) error {
			return tx.Model(&model.McpServer{}).Where("id = ?", id).
				Update("visibility_type", visType).Error
		},
	)
}

// ──────────────────────────────────────────────
// POST /admin/images/type-visibility
// ──────────────────────────────────────────────

// HandleImageTypeVisibility 设置镜像类型（agent_type 级别）应用范围
func HandleImageTypeVisibility(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req struct {
		AgentType      string `json:"agent_type"`
		VisibilityType string `json:"visibility_type"`
		GroupIDs       []uint `json:"group_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}

	if req.AgentType == "" || !model.IsValidAgentType(r.Context(), req.AgentType) {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamInvalidWithDetail, "agent_type", req.AgentType))
		return
	}
	if req.VisibilityType != usergroup.VisibilityAll && req.VisibilityType != usergroup.VisibilityGroup {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "visibility_type"))
		return
	}
	if req.VisibilityType == usergroup.VisibilityGroup && len(req.GroupIDs) == 0 {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "group_ids"))
		return
	}
	if req.VisibilityType == usergroup.VisibilityGroup {
		if err := usergroup.ValidateGroupIDs(r.Context(), req.GroupIDs); err != nil {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "group_ids"))
			return
		}
	}

	tx := model.DB(r.Context()).Begin()
	if err := usergroup.SetImageTypeVisibility(tx, req.AgentType, req.VisibilityType, req.GroupIDs); err != nil {
		tx.Rollback()
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDatabaseOperationFailed))
		return
	}
	tx.Commit()

	jsonOK(w, map[string]interface{}{"ok": true})
}

// ──────────────────────────────────────────────
// GET /admin/group-config/groups
// 批量查询资源绑定了哪些组（策略类型同时返回 value）
// ──────────────────────────────────────────────

// configQuery 单个查询项
type configQuery struct {
	ConfigType string `json:"config_type"`
	ConfigKey  string `json:"config_key"`
}

// configGroupResult 单个查询结果
type configGroupResult struct {
	ConfigType string            `json:"config_type"`
	ConfigKey  string            `json:"config_key"`
	Groups     []configGroupItem `json:"groups"`
}

// configGroupItem 绑定的组信息
type configGroupItem struct {
	GroupID   uint        `json:"group_id"`
	GroupName string      `json:"group_name"`
	Value     interface{} `json:"value,omitempty"` // 仅 policy 类型返回
}

// HandleGroupConfigGroups 批量查询多个配置项各自绑定了哪些组
func HandleGroupConfigGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	// 解析 queries 参数
	queriesStr := r.URL.Query().Get("queries")
	if queriesStr == "" {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "queries"))
		return
	}

	var queries []configQuery
	if err := json.Unmarshal([]byte(queriesStr), &queries); err != nil {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamFormatError, "queries"))
		return
	}
	if len(queries) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgQueriesNotEmpty))
		return
	}

	// 校验所有 config_type
	for _, q := range queries {
		if !usergroup.IsValidConfigType(q.ConfigType) {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidConfigType, q.ConfigType))
			return
		}
	}

	// 逐项查询
	results := make([]configGroupResult, 0, len(queries))
	for _, q := range queries {
		bindings, err := model.GetBindingsByResource(r.Context(), q.ConfigType, q.ConfigKey)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDatabaseOperationFailed))
			return
		}

		// 兼容：查询 token_quota_day 但 binding 已合并到 token_quota_rules 时，从 rules 反推
		if q.ConfigType == usergroup.ConfigTypePolicy && q.ConfigKey == usergroup.PolicyKeyTokenQuotaDay {
			existingGroupIDs := make(map[uint]bool, len(bindings))
			for _, b := range bindings {
				existingGroupIDs[b.GroupID] = true
			}
			rulesBindings, err := model.GetBindingsByResource(r.Context(), usergroup.ConfigTypePolicy, usergroup.PolicyKeyTokenQuotaRules)
			if err == nil {
				for _, rb := range rulesBindings {
					// 该组仍有旧 token_quota_day binding，无需从 rules 反推
					if existingGroupIDs[rb.GroupID] {
						continue
					}
					var wrapper struct {
						Value string `json:"value"`
					}
					if json.Unmarshal([]byte(rb.ValueJSON), &wrapper) == nil && wrapper.Value != "" {
						if rules, ok := model.ParseTokenQuotaRules(wrapper.Value); ok {
							d := model.TokenQuotaDayFromRules(rules)
							// 构造兼容旧格式的 value_json
							dayJSON, _ := json.Marshal(map[string]int{"value": d})
							rb.ValueJSON = string(dayJSON)
							bindings = append(bindings, rb)
						}
					}
				}
			}
		}

		// 兼容：查询 token_quota_rules 但仍存在旧 token_quota_day binding 时，从 day 反推 rules。
		// 这与运行时 ResolveTokenQuotaRulesForAncestors 的优先级保持一致：rules > day > fallback。
		if q.ConfigType == usergroup.ConfigTypePolicy && q.ConfigKey == usergroup.PolicyKeyTokenQuotaRules {
			existingGroupIDs := make(map[uint]bool, len(bindings))
			for _, b := range bindings {
				existingGroupIDs[b.GroupID] = true
			}
			dayBindings, err := model.GetBindingsByResource(r.Context(), usergroup.ConfigTypePolicy, usergroup.PolicyKeyTokenQuotaDay)
			if err == nil {
				for _, db := range dayBindings {
					// 该组已有 token_quota_rules binding，保持 rules 优先，不被旧 day 覆盖。
					if existingGroupIDs[db.GroupID] {
						continue
					}
					var wrapper struct {
						Value *int `json:"value"`
					}
					if json.Unmarshal([]byte(db.ValueJSON), &wrapper) == nil && wrapper.Value != nil {
						rules := []model.TokenQuotaRule{}
						if *wrapper.Value >= 0 {
							rules = []model.TokenQuotaRule{{Mode: model.QuotaModeDay, Limit: *wrapper.Value}}
						}
						rulesJSON := model.MarshalTokenQuotaRules(rules)
						rulesValueJSON, _ := json.Marshal(map[string]string{"value": rulesJSON})
						db.ValueJSON = string(rulesValueJSON)
						bindings = append(bindings, db)
					}
				}
			}
		}

		// 兼容：查询 global_token_quota_day 但 binding 已合并到 global_token_quota_rules 时，从 rules 反推
		if q.ConfigType == usergroup.ConfigTypePolicy && q.ConfigKey == usergroup.PolicyKeyGlobalTokenQuotaDay {
			existingGroupIDs := make(map[uint]bool, len(bindings))
			for _, b := range bindings {
				existingGroupIDs[b.GroupID] = true
			}
			siteConfig := model.GetSiteConfig(r.Context())
			mode := model.QuotaModeDay
			if siteConfig.NormalizedGlobalTokenQuotaPeriod() == model.GlobalTokenQuotaPeriodMonth {
				mode = model.QuotaModeMonth
			}
			rulesBindings, err := model.GetBindingsByResource(r.Context(), usergroup.ConfigTypePolicy, usergroup.PolicyKeyGlobalTokenQuotaRules)
			if err == nil {
				for _, rb := range rulesBindings {
					if existingGroupIDs[rb.GroupID] {
						continue
					}
					var wrapper struct {
						Value string `json:"value"`
					}
					if json.Unmarshal([]byte(rb.ValueJSON), &wrapper) == nil && wrapper.Value != "" {
						if rules, ok := model.ParseTokenQuotaRules(wrapper.Value); ok {
							limit := model.TokenQuotaLimitFromRules(rules, mode)
							quotaJSON, _ := json.Marshal(map[string]int{"value": limit})
							rb.ValueJSON = string(quotaJSON)
							bindings = append(bindings, rb)
						}
					}
				}
			}
		}

		// 兼容：查询 global_token_quota_rules 但仍存在旧 global_token_quota_day binding 时，从 day 反推 rules。
		if q.ConfigType == usergroup.ConfigTypePolicy && q.ConfigKey == usergroup.PolicyKeyGlobalTokenQuotaRules {
			existingGroupIDs := make(map[uint]bool, len(bindings))
			for _, b := range bindings {
				existingGroupIDs[b.GroupID] = true
			}
			siteConfig := model.GetSiteConfig(r.Context())
			mode := model.QuotaModeDay
			if siteConfig.NormalizedGlobalTokenQuotaPeriod() == model.GlobalTokenQuotaPeriodMonth {
				mode = model.QuotaModeMonth
			}
			dayBindings, err := model.GetBindingsByResource(r.Context(), usergroup.ConfigTypePolicy, usergroup.PolicyKeyGlobalTokenQuotaDay)
			if err == nil {
				for _, db := range dayBindings {
					if existingGroupIDs[db.GroupID] {
						continue
					}
					var wrapper struct {
						Value *int `json:"value"`
					}
					if json.Unmarshal([]byte(db.ValueJSON), &wrapper) == nil && wrapper.Value != nil {
						rulesJSON := model.MarshalTokenQuotaRules(model.GlobalRulesFromLegacyQuota(*wrapper.Value, mode))
						rulesValueJSON, _ := json.Marshal(map[string]string{"value": rulesJSON})
						db.ValueJSON = string(rulesValueJSON)
						bindings = append(bindings, db)
					}
				}
			}
		}

		// 批量获取组名
		groupIDs := make([]uint, 0, len(bindings))
		for _, b := range bindings {
			groupIDs = append(groupIDs, b.GroupID)
		}
		nameMap := batchGetGroupNames(r.Context(), groupIDs)

		// 组装 groups 列表
		groups := make([]configGroupItem, 0, len(bindings))
		for _, b := range bindings {
			item := configGroupItem{
				GroupID:   b.GroupID,
				GroupName: nameMap[b.GroupID],
			}
			// 策略类型返回 value
			if q.ConfigType == usergroup.ConfigTypePolicy && b.ValueJSON != "" && b.ValueJSON != "{}" {
				var val interface{}
				if err := json.Unmarshal([]byte(b.ValueJSON), &val); err == nil {
					item.Value = val
				}
			}
			groups = append(groups, item)
		}

		results = append(results, configGroupResult{
			ConfigType: q.ConfigType,
			ConfigKey:  q.ConfigKey,
			Groups:     groups,
		})
	}

	jsonOK(w, map[string]interface{}{"ok": true, "results": results})
}

// batchGetGroupNames 批量获取组名映射
func batchGetGroupNames(ctx context.Context, groupIDs []uint) map[uint]string {
	result := make(map[uint]string, len(groupIDs))
	if len(groupIDs) == 0 {
		return result
	}
	groups, err := model.GetGroupsByIDs(ctx, groupIDs)
	if err != nil {
		return result
	}
	for _, g := range groups {
		result[g.ID] = g.Name
	}
	return result
}

// ──────────────────────────────────────────────
// POST /admin/group-config/policy
// 为某组设置（或更新）一项策略配置
// ──────────────────────────────────────────────

// HandleSetGroupPolicy 设置组策略
func HandleSetGroupPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req struct {
		GroupID   uint   `json:"group_id"`
		ConfigKey string `json:"config_key"`
		ValueJSON string `json:"value_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}

	if req.GroupID == 0 {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "group_id"))
		return
	}
	if !usergroup.IsValidPolicyKey(req.ConfigKey) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidConfigKey))
		return
	}
	if req.ValueJSON == "" {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "value_json"))
		return
	}
	if !json.Valid([]byte(req.ValueJSON)) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}

	// 校验组存在
	if err := usergroup.ValidateGroupIDs(r.Context(), []uint{req.GroupID}); err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, hcommon.I18nError(i18n.MsgGroupNotFound))
		return
	}

	// token_quota_rules/global_token_quota_rules 策略特殊处理：校验 rules + 自动填充 custom start + normalize
	valueJSON := req.ValueJSON
	if req.ConfigKey == usergroup.PolicyKeyTokenQuotaRules || req.ConfigKey == usergroup.PolicyKeyGlobalTokenQuotaRules {
		var wrapper struct {
			Value string `json:"value"`
		}
		if err := json.Unmarshal([]byte(valueJSON), &wrapper); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
			return
		}
		normalized, err := model.NormalizeTokenQuotaRules(wrapper.Value)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidTokenQuotaRules))
			return
		}
		b, _ := json.Marshal(map[string]string{"value": normalized})
		valueJSON = string(b)

		// 事务：写 rules + 清理配对 day binding（与写 day 路径对称，防止 legacy day binding 残留导致删除后复活）
		tx := model.DB(r.Context()).Begin()
		if err := usergroup.SetPolicy(tx, req.GroupID, req.ConfigKey, valueJSON); err != nil {
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		if pairedKey := usergroup.QuotaPolicyPairedKey(req.ConfigKey); pairedKey != "" {
			// 幂等删除配对 day binding（不存在不算错）
			if err := usergroup.DeletePolicy(tx, req.GroupID, pairedKey); err != nil {
				tx.Rollback()
				writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
				return
			}
		}
		tx.Commit()
		jsonOK(w, map[string]interface{}{"ok": true})
		return
	}

	// token_quota_day 策略：upsert day 规则到同组的 token_quota_rules 策略中
	if req.ConfigKey == usergroup.PolicyKeyTokenQuotaDay {
		var wrapper struct {
			Value int `json:"value"`
		}
		if err := json.Unmarshal([]byte(valueJSON), &wrapper); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
			return
		}
		// 读取该组现有的 token_quota_rules 策略
		existingRules := usergroup.ResolvePolicyStringForGroup(r.Context(), usergroup.PolicyKeyTokenQuotaRules, req.GroupID, "")
		upserted := model.UpsertDayRule(existingRules, wrapper.Value)
		normalized, err := model.NormalizeTokenQuotaRules(upserted)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidTokenQuotaRules))
			return
		}
		// 事务：写入 token_quota_rules + 删除 token_quota_day
		rulesValueJSON, _ := json.Marshal(map[string]string{"value": normalized})
		tx := model.DB(r.Context()).Begin()
		if err := usergroup.SetPolicy(tx, req.GroupID, usergroup.PolicyKeyTokenQuotaRules, string(rulesValueJSON)); err != nil {
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDatabaseOperationFailed))
			return
		}
		if err := usergroup.DeletePolicy(tx, req.GroupID, usergroup.PolicyKeyTokenQuotaDay); err != nil {
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDatabaseOperationFailed))
			return
		}
		tx.Commit()
		jsonOK(w, map[string]interface{}{"ok": true})
		return
	}

	// global_token_quota_day 策略：upsert 当前站点全局周期对应规则到同组的 global_token_quota_rules 策略中
	if req.ConfigKey == usergroup.PolicyKeyGlobalTokenQuotaDay {
		var wrapper struct {
			Value int `json:"value"`
		}
		if err := json.Unmarshal([]byte(valueJSON), &wrapper); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
			return
		}
		siteConfig := model.GetSiteConfig(r.Context())
		existingRules := usergroup.ResolvePolicyStringForGroup(r.Context(), usergroup.PolicyKeyGlobalTokenQuotaRules, req.GroupID, "")
		upserted := model.UpsertGlobalPeriodRule(existingRules, siteConfig.NormalizedGlobalTokenQuotaPeriod(), wrapper.Value)
		normalized, err := model.NormalizeTokenQuotaRules(upserted)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		rulesValueJSON, _ := json.Marshal(map[string]string{"value": normalized})
		tx := model.DB(r.Context()).Begin()
		if err := usergroup.SetPolicy(tx, req.GroupID, usergroup.PolicyKeyGlobalTokenQuotaRules, string(rulesValueJSON)); err != nil {
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		if err := usergroup.DeletePolicy(tx, req.GroupID, usergroup.PolicyKeyGlobalTokenQuotaDay); err != nil {
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		tx.Commit()
		jsonOK(w, map[string]interface{}{"ok": true})
		return
	}

	if err := usergroup.SetPolicy(model.DB(r.Context()), req.GroupID, req.ConfigKey, valueJSON); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDatabaseOperationFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// ──────────────────────────────────────────────
// POST /admin/group-config/policy/delete
// 删除某组的某项策略配置
// ──────────────────────────────────────────────

// HandleDeleteGroupPolicy 删除组策略
func HandleDeleteGroupPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req struct {
		GroupID   uint   `json:"group_id"`
		ConfigKey string `json:"config_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}

	if req.GroupID == 0 {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "group_id"))
		return
	}
	if !usergroup.IsValidPolicyKey(req.ConfigKey) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidConfigKey))
		return
	}

	// 校验组存在
	if err := usergroup.ValidateGroupIDs(r.Context(), []uint{req.GroupID}); err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, hcommon.I18nError(i18n.MsgGroupNotFound))
		return
	}

	// 校验绑定是否存在
	configKeyToDelete := req.ConfigKey
	bindings, err := model.GetPolicyBindingsByGroups(r.Context(), []uint{req.GroupID}, req.ConfigKey)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	if len(bindings) == 0 {
		// 兼容：目标 key 不存在时，尝试删除配对 key（仅对配额类 key 生效）。
		// 例如组仅有 legacy token_quota_day binding、无 token_quota_rules binding 时，
		// 前端删 token_quota_rules 请求会 fallback 删掉 token_quota_day，避免"删不掉/刷新复活"循环。
		if pairedKey := usergroup.QuotaPolicyPairedKey(req.ConfigKey); pairedKey != "" {
			pairedBindings, perr := model.GetPolicyBindingsByGroups(r.Context(), []uint{req.GroupID}, pairedKey)
			if perr != nil {
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(perr, i18n.MsgOperationFailed))
				return
			}
			if len(pairedBindings) > 0 {
				configKeyToDelete = pairedKey
			} else {
				writeError(w, r, http.StatusUnprocessableEntity, hcommon.I18nError(i18n.MsgPolicyNotConfigured))
				return
			}
		} else {
			writeError(w, r, http.StatusUnprocessableEntity, hcommon.I18nError(i18n.MsgPolicyNotConfigured))
			return
		}
	}

	if err := usergroup.DeletePolicy(model.DB(r.Context()), req.GroupID, configKeyToDelete); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}
