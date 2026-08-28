package usergroup

import (
	"context"
	"encoding/json"

	"hatchery/i18n"
	"hatchery/model"
)

// ──────────────────────────────────────────────
// 配置总览聚合
// ──────────────────────────────────────────────
// 为组详情接口提供 config_overview 数据：
// 每种资源返回带来源标记的列表。

// OverviewItem 加法型资源总览条目
type OverviewItem struct {
	ResourceID   uint   `json:"resource_id,omitempty"`
	ResourceName string `json:"resource_name"`
	Source       Source `json:"source"`
}

// PolicyOverviewItem 策略总览条目
type PolicyOverviewItem struct {
	Key    string      `json:"key"`
	Label  string      `json:"label"`
	Value  interface{} `json:"value"`
	Source Source      `json:"source"`
}

// AdditiveOverview 加法型资源配置总览结果
type AdditiveOverview struct {
	Items []OverviewItem `json:"items"`
	Total int            `json:"total"`
}

// ResolveAdditiveOverview 生成某类加法型资源的配置总览。
// groupID: 当前组 ID
// ancestors: 含自己的祖先链（近→远）
// resourceNameFn: 根据 configKey 获取资源名称的回调
func ResolveAdditiveOverview(ctx context.Context, configType string, groupID uint, ancestors []uint, resourceNameFn func(string) string) (*AdditiveOverview, error) {
	if len(ancestors) == 0 {
		return &AdditiveOverview{Items: []OverviewItem{}, Total: 0}, nil
	}

	// 查询所有祖先链上的绑定
	bindings, err := model.GetBindingsByGroups(ctx, ancestors, configType)
	if err != nil {
		return nil, err
	}

	// 获取祖先组名映射
	groupNames := batchGetGroupNames(ctx, ancestors)

	// 按资源去重，记录来源（优先级：本组 > 近祖先 > 远祖先）
	// 用 ancestors 顺序作为优先级，构建 groupID → priority 映射
	priorityMap := make(map[uint]int, len(ancestors))
	for i, gid := range ancestors {
		priorityMap[gid] = i
	}

	type resourceEntry struct {
		configKey string
		source    Source
		priority  int
	}
	bestMap := make(map[string]*resourceEntry)

	for _, b := range bindings {
		priority, ok := priorityMap[b.GroupID]
		if !ok {
			continue
		}
		existing, exists := bestMap[b.ConfigKey]
		if !exists || priority < existing.priority {
			sourceType := SourceInherited
			if b.GroupID == groupID {
				sourceType = SourceLocal
			}
			bestMap[b.ConfigKey] = &resourceEntry{
				configKey: b.ConfigKey,
				source: Source{
					Type:     sourceType,
					GroupID:  b.GroupID,
					FullPath: groupNames[b.GroupID],
				},
				priority: priority,
			}
		}
	}

	items := make([]OverviewItem, 0, len(bestMap))
	for _, entry := range bestMap {
		name := entry.configKey
		if resourceNameFn != nil {
			name = resourceNameFn(entry.configKey)
		}
		var rid uint
		parseUintStr(entry.configKey, &rid)
		items = append(items, OverviewItem{
			ResourceID:   rid,
			ResourceName: name,
			Source:       entry.source,
		})
	}

	return &AdditiveOverview{Items: items, Total: len(items)}, nil
}

// ResolvePolicyOverview 生成所有策略项的配置总览。
func ResolvePolicyOverview(ctx context.Context, groupID uint, ancestors []uint, siteConfig *model.SiteConfig) ([]PolicyOverviewItem, error) {
	// 一次查询拿到祖先链上所有策略绑定
	allBindings, err := model.GetAllPolicyBindingsByGroups(ctx, ancestors)
	if err != nil {
		return nil, err
	}

	// 按 key → {groupID → binding} 分组
	bindingIndex := make(map[string]map[uint]string)
	for _, b := range allBindings {
		if _, ok := bindingIndex[b.ConfigKey]; !ok {
			bindingIndex[b.ConfigKey] = make(map[uint]string)
		}
		bindingIndex[b.ConfigKey][b.GroupID] = b.ValueJSON
	}

	// 获取组名映射
	groupNames := batchGetGroupNames(ctx, ancestors)

	// 优先级映射
	priorityMap := make(map[uint]int, len(ancestors))
	for i, gid := range ancestors {
		priorityMap[gid] = i
	}

	items := make([]PolicyOverviewItem, 0, len(PolicyDefs))
	for _, key := range policyKeyOrder {
		def, ok := PolicyDefs[key]
		if !ok {
			continue
		}
		label := def.Label
		// GlobalTokenQuotaRules 的 label 需根据实际周期动态显示
		if key == PolicyKeyGlobalTokenQuotaRules && siteConfig != nil {
			if siteConfig.NormalizedGlobalTokenQuotaPeriod() == model.GlobalTokenQuotaPeriodMonth {
				label = i18n.T(ctx, i18n.MsgPolicyGlobalTokenQuotaRulesMonthly)
			} else {
				label = i18n.T(ctx, i18n.MsgPolicyGlobalTokenQuotaRulesDaily)
			}
		}
		item := PolicyOverviewItem{
			Key:   key,
			Label: label,
		}

		if isQuotaRulesCompatPolicyKey(key) {
			item.Value, item.Source = resolveQuotaRulesCompatPolicyOverview(ctx, key, ancestors, siteConfig)
			items = append(items, item)
			continue
		}

		groupBindings, hasBindings := bindingIndex[key]
		if !hasBindings {
			// 使用全局默认
			item.Value = getSiteConfigValue(ctx, siteConfig, def)
			item.Source = Source{Type: SourceSiteDefault}
		} else {
			// 按祖先优先级找最近的
			found := false
			for _, gid := range ancestors {
				if val, ok := groupBindings[gid]; ok {
					sourceType := SourceInherited
					if gid == groupID {
						sourceType = SourceLocal
					}
					item.Value = parseValueJSON(val, def.ValueType)
					item.Source = Source{
						Type:     sourceType,
						GroupID:  gid,
						FullPath: groupNames[gid],
					}
					found = true
					break
				}
			}
			if !found {
				item.Value = getSiteConfigValue(ctx, siteConfig, def)
				item.Source = Source{Type: SourceSiteDefault}
			}
		}

		items = append(items, item)
	}

	return items, nil
}

// ──────────────────────────────────────────────
// 内部辅助
// ──────────────────────────────────────────────

func batchGetGroupNames(ctx context.Context, groupIDs []uint) map[uint]string {
	result := make(map[uint]string, len(groupIDs))
	groups, err := model.GetGroupsByIDs(ctx, groupIDs)
	if err != nil {
		return result
	}
	for _, g := range groups {
		result[g.ID] = g.FullPath
	}
	return result
}

func getSiteConfigValue(ctx context.Context, sc *model.SiteConfig, def PolicyDef) interface{} {
	if sc == nil {
		return nil
	}
	switch def.Key {
	case PolicyKeyTokenQuotaDay:
		return model.TokenQuotaDayFromRules(sc.ResolvedDefaultTokenQuotaRules())
	case PolicyKeyTokenQuotaRules:
		return model.MarshalTokenQuotaRules(sc.ResolvedDefaultTokenQuotaRules())
	case PolicyKeyInstanceQuota:
		return sc.DefaultInstanceQuota
	case PolicyKeyGlobalTokenQuotaDay:
		day, _ := model.EffectiveGlobalTokenQuotaLegacyFields(sc.GlobalTokenQuotaDay, sc.GlobalTokenQuotaPeriod, sc.GlobalTokenQuotaRules)
		return day
	case PolicyKeyGlobalTokenQuotaRules:
		return model.MarshalTokenQuotaRules(sc.ResolvedGlobalTokenQuotaRules())
	case PolicyKeyAgentTerminal:
		return sc.TerminalEnabled
	case PolicyKeyChatView:
		return sc.ChatViewEnabled
	case PolicyKeyGatewayUI:
		return sc.GatewayUIEnable
	case PolicyKeyBrowserVNC:
		return sc.BrowserVNCEnable
	case PolicyKeyUserConfigModel:
		return sc.UserConfigModelEnabled
	case PolicyKeyUserConfigChannel:
		return sc.UserConfigChannelEnabled
	case PolicyKeyModelQuota:
		return sc.ModelQuotaEnabled
	case PolicyKeyCustomModel:
		return model.IsCustomModelEnabled(ctx) // 读 ai_models 表 model_id="custom" 占位记录的 enabled+visible
	case PolicyKeyLobsterDoctor:
		return sc.DoctorEnabled
	case PolicyKeySMHAutoProvision:
		return sc.SMHAutoProvisionOnCreate
	default:
		return nil
	}
}

func isQuotaRulesCompatPolicyKey(key string) bool {
	switch key {
	case PolicyKeyTokenQuotaRules, PolicyKeyTokenQuotaDay, PolicyKeyGlobalTokenQuotaRules, PolicyKeyGlobalTokenQuotaDay:
		return true
	default:
		return false
	}
}

func resolveQuotaRulesCompatPolicyOverview(ctx context.Context, key string, ancestors []uint, sc *model.SiteConfig) (interface{}, Source) {
	switch key {
	case PolicyKeyTokenQuotaRules:
		rulesJSON, source := resolveUserTokenQuotaRulesPolicy(ctx, ancestors, sc)
		return rulesJSON, source
	case PolicyKeyTokenQuotaDay:
		rulesJSON, source := resolveUserTokenQuotaRulesPolicy(ctx, ancestors, sc)
		if rules, ok := model.ParseTokenQuotaRules(rulesJSON); ok {
			return model.TokenQuotaDayFromRules(rules), source
		}
		return -1, source
	case PolicyKeyGlobalTokenQuotaRules:
		rulesJSON, source := resolveGlobalTokenQuotaRulesPolicy(ctx, ancestors, sc)
		return rulesJSON, source
	case PolicyKeyGlobalTokenQuotaDay:
		rulesJSON, source := resolveGlobalTokenQuotaRulesPolicy(ctx, ancestors, sc)
		if rules, ok := model.ParseTokenQuotaRules(rulesJSON); ok {
			return model.TokenQuotaLimitFromRules(rules, overviewGlobalTokenQuotaMode(sc)), source
		}
		return -1, source
	default:
		return nil, Source{Type: SourceSiteDefault}
	}
}

func resolveUserTokenQuotaRulesPolicy(ctx context.Context, ancestors []uint, sc *model.SiteConfig) (string, Source) {
	fallbackRules := "[]"
	if sc != nil {
		fallbackRules = model.MarshalTokenQuotaRules(sc.ResolvedDefaultTokenQuotaRules())
	}
	return ResolveTokenQuotaRulesForAncestors(ctx, ancestors, fallbackRules, -1)
}

func resolveGlobalTokenQuotaRulesPolicy(ctx context.Context, ancestors []uint, sc *model.SiteConfig) (string, Source) {
	mode := overviewGlobalTokenQuotaMode(sc)
	rulesRaw, source, _ := ResolvePolicyString(ctx, PolicyKeyGlobalTokenQuotaRules, ancestors, "")
	if rulesRaw != "" {
		// 显式 "[]"（空规则 = 无限制）→ 补 limit=-1 保留周期上下文
		if rules, ok := model.ParseTokenQuotaRules(rulesRaw); ok && len(rules) == 0 {
			return model.MarshalTokenQuotaRules([]model.TokenQuotaRule{{Mode: mode, Limit: -1}}), source
		}
		return rulesRaw, source
	}
	dayVal, daySource, _ := ResolvePolicyInt(ctx, PolicyKeyGlobalTokenQuotaDay, ancestors, -1)
	if daySource.Type != SourceSiteDefault {
		if dayVal < 0 {
			// 无限制：保留周期上下文，formatSingleQuotaRule 会输出 "每日 无限制" / "每月 无限制"
			return model.MarshalTokenQuotaRules([]model.TokenQuotaRule{{Mode: mode, Limit: -1}}), daySource
		}
		return model.MarshalTokenQuotaRules(model.GlobalRulesFromLegacyQuota(dayVal, mode)), daySource
	}
	if sc == nil {
		return model.MarshalTokenQuotaRules([]model.TokenQuotaRule{{Mode: model.QuotaModeDay, Limit: -1}}), Source{Type: SourceSiteDefault}
	}
	rules := sc.ResolvedGlobalTokenQuotaRules()
	if len(rules) == 0 {
		// GlobalTokenQuotaDay=-1（无限制）→ 空规则，补一条 limit=-1 保留周期上下文
		return model.MarshalTokenQuotaRules([]model.TokenQuotaRule{{Mode: mode, Limit: -1}}), Source{Type: SourceSiteDefault}
	}
	return model.MarshalTokenQuotaRules(rules), Source{Type: SourceSiteDefault}
}

func overviewGlobalTokenQuotaMode(sc *model.SiteConfig) string {
	if sc != nil && sc.NormalizedGlobalTokenQuotaPeriod() == model.GlobalTokenQuotaPeriodMonth {
		return model.QuotaModeMonth
	}
	return model.QuotaModeDay
}

func parseValueJSON(raw string, valueType PolicyValueType) interface{} {
	switch valueType {
	case PolicyValueInt:
		var v struct {
			Value int `json:"value"`
		}
		if err := json.Unmarshal([]byte(raw), &v); err == nil {
			return v.Value
		}
	case PolicyValueBool:
		var v struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.Unmarshal([]byte(raw), &v); err == nil {
			return v.Enabled
		}
	case PolicyValueString:
		var v struct {
			Value string `json:"value"`
		}
		if err := json.Unmarshal([]byte(raw), &v); err == nil {
			return v.Value
		}
	}
	return nil
}
