package usergroup

import (
	"context"
	"encoding/json"
	"fmt"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ──────────────────────────────────────────────────────────────────────────────
// resolve.go — 统一的资源/策略解析器
// ──────────────────────────────────────────────────────────────────────────────
// 包含两类 Resolver：
//  1. 策略型（policy）：最近祖先覆盖语义
//  2. 加法型（additive）：Union 语义（channel/plugin_bundle/mcp/image_type/model/skill/role）

// ══════════════════════════════════════════════════
// 一、策略型 Resolver
// ══════════════════════════════════════════════════
// 解析规则：按祖先链从近到远查找，最先命中的胜出。
// ancestors[0] = 本组, ancestors[1] = 父, ..., ancestors[N] = 根
// 均未命中 → 使用 fallback（来自 site_configs）。

// PolicyResult 策略解析结果
type PolicyResult struct {
	Value  interface{} `json:"value"`
	Source Source      `json:"source"`
}

// ── 按 group_id 解析（通用） ─────────────────────────
// groupID=0 表示未绑定任何组，直接返回 fallback。

// ResolvePolicyBoolForGroup 按 group_id 解析布尔策略。
func ResolvePolicyBoolForGroup(ctx context.Context, key string, groupID uint, fallback bool) bool {
	if groupID == 0 {
		return fallback
	}
	ancestors, err := GetAncestorIDs(ctx, groupID)
	if err != nil || len(ancestors) == 0 {
		return fallback
	}
	val, _, _ := ResolvePolicyBool(ctx, key, ancestors, fallback)
	return val
}

// ResolvePolicyIntForGroup 按 group_id 解析整型策略。
func ResolvePolicyIntForGroup(ctx context.Context, key string, groupID uint, fallback int) int {
	if groupID == 0 {
		return fallback
	}
	ancestors, err := GetAncestorIDs(ctx, groupID)
	if err != nil || len(ancestors) == 0 {
		return fallback
	}
	val, _, _ := ResolvePolicyInt(ctx, key, ancestors, fallback)
	return val
}

// ── 底层函数（接收已获取的祖先链） ───────────────────────

// ResolvePolicyInt 按祖先链解析整型策略值。
func ResolvePolicyInt(ctx context.Context, key string, ancestors []uint, fallback int) (int, Source, error) {
	raw, source, err := resolvePolicyRaw(ctx, key, ancestors)
	if err != nil {
		return fallback, Source{Type: SourceSiteDefault}, err
	}
	if raw == "" {
		return fallback, Source{Type: SourceSiteDefault}, nil
	}
	var val struct {
		Value int `json:"value"`
	}
	if err := json.Unmarshal([]byte(raw), &val); err != nil {
		return fallback, Source{Type: SourceSiteDefault}, hcommon.I18nRichError(err, i18n.MsgResolvePolicyValueFailed, key)
	}
	return val.Value, source, nil
}

// ResolvePolicyBool 按祖先链解析布尔策略值。
func ResolvePolicyBool(ctx context.Context, key string, ancestors []uint, fallback bool) (bool, Source, error) {
	raw, source, err := resolvePolicyRaw(ctx, key, ancestors)
	if err != nil {
		return fallback, Source{Type: SourceSiteDefault}, err
	}
	if raw == "" {
		return fallback, Source{Type: SourceSiteDefault}, nil
	}
	var val struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal([]byte(raw), &val); err != nil {
		return fallback, Source{Type: SourceSiteDefault}, hcommon.I18nRichError(err, i18n.MsgResolvePolicyValueFailed, key)
	}
	return val.Enabled, source, nil
}

// ResolvePolicyString 按祖先链解析字符串策略值。
// value_json 格式: {"value": "..."}
func ResolvePolicyString(ctx context.Context, key string, ancestors []uint, fallback string) (string, Source, error) {
	raw, source, err := resolvePolicyRaw(ctx, key, ancestors)
	if err != nil {
		return fallback, Source{Type: SourceSiteDefault}, err
	}
	if raw == "" {
		return fallback, Source{Type: SourceSiteDefault}, nil
	}
	var val struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(raw), &val); err != nil {
		return fallback, Source{Type: SourceSiteDefault}, hcommon.I18nRichError(err, i18n.MsgResolvePolicyValueFailed, key)
	}
	if val.Value == "" {
		return fallback, Source{Type: SourceSiteDefault}, nil
	}
	return val.Value, source, nil
}

// ResolvePolicyStringForGroup 按 group_id 解析字符串策略。
func ResolvePolicyStringForGroup(ctx context.Context, key string, groupID uint, fallback string) string {
	if groupID == 0 {
		return fallback
	}
	ancestors, err := GetAncestorIDs(ctx, groupID)
	if err != nil || len(ancestors) == 0 {
		return fallback
	}
	val, _, _ := ResolvePolicyString(ctx, key, ancestors, fallback)
	return val
}

// ResolveTokenQuotaRulesForGroup 按分组 ID 解析用户 Token 配额规则。
// 优先级与 LLM proxy 一致：组/祖先 token_quota_rules > 组/祖先 token_quota_day(转换) > fallback。
func ResolveTokenQuotaRulesForGroup(ctx context.Context, groupID uint, fallbackRules string, fallbackDay int) (string, Source) {
	if groupID == 0 {
		return fallbackTokenQuotaRules(fallbackRules, fallbackDay), Source{Type: SourceSiteDefault}
	}
	ancestors, err := GetAncestorIDs(ctx, groupID)
	if err != nil || len(ancestors) == 0 {
		return fallbackTokenQuotaRules(fallbackRules, fallbackDay), Source{Type: SourceSiteDefault}
	}
	return ResolveTokenQuotaRulesForAncestors(ctx, ancestors, fallbackRules, fallbackDay)
}

// ResolveTokenQuotaRulesForAncestors 按已知祖先链解析用户 Token 配额规则。
// ancestors[0] 应为当前组，后续为父级直到根组。
func ResolveTokenQuotaRulesForAncestors(ctx context.Context, ancestors []uint, fallbackRules string, fallbackDay int) (string, Source) {
	if len(ancestors) == 0 {
		return fallbackTokenQuotaRules(fallbackRules, fallbackDay), Source{Type: SourceSiteDefault}
	}
	rulesRaw, source, _ := ResolvePolicyString(ctx, PolicyKeyTokenQuotaRules, ancestors, "")
	if rulesRaw != "" {
		// 显式 "[]"（空规则 = 无限制）→ 补 limit=-1 保留周期上下文
		if rules, ok := model.ParseTokenQuotaRules(rulesRaw); ok && len(rules) == 0 {
			return `[{"mode":"day","limit":-1}]`, source
		}
		return rulesRaw, source
	}
	const noTokenQuotaDayPolicy = -2
	dayVal, daySource, _ := ResolvePolicyInt(ctx, PolicyKeyTokenQuotaDay, ancestors, noTokenQuotaDayPolicy)
	if daySource.Type != SourceSiteDefault {
		return tokenQuotaRulesFromLegacyDay(dayVal), daySource
	}
	return fallbackTokenQuotaRules(fallbackRules, fallbackDay), Source{Type: SourceSiteDefault}
}

func fallbackTokenQuotaRules(fallbackRules string, fallbackDay int) string {
	if fallbackRules != "" {
		// 显式 "[]"（空规则 = 无限制）→ 补 limit=-1 保留周期上下文
		if rules, ok := model.ParseTokenQuotaRules(fallbackRules); ok && len(rules) == 0 {
			return `[{"mode":"day","limit":-1}]`
		}
		return fallbackRules
	}
	return tokenQuotaRulesFromLegacyDay(fallbackDay)
}

func tokenQuotaRulesFromLegacyDay(day int) string {
	if day >= 0 {
		return fmt.Sprintf(`[{"mode":"day","limit":%d}]`, day)
	}
	// 无限制：保留周期上下文，formatSingleQuotaRule 会输出 "每日 无限制"
	return `[{"mode":"day","limit":-1}]`
}

// ResolveExplicitGlobalTokenQuotaRulesForGroup 按 group_id 解析显式配置的分组全局配额规则。
// 优先级：组的 global_token_quota_rules > 组的 global_token_quota_day (按 period 转换)。
// 返回第二个 bool 表示是否命中了显式组策略；未命中时调用方可只执行站点全局限制。
func ResolveExplicitGlobalTokenQuotaRulesForGroup(ctx context.Context, groupID uint, period string) (string, bool) {
	if groupID == 0 {
		return "", false
	}
	ancestors, err := GetAncestorIDs(ctx, groupID)
	if err != nil || len(ancestors) == 0 {
		return "", false
	}
	rulesRaw, _, _ := ResolvePolicyString(ctx, PolicyKeyGlobalTokenQuotaRules, ancestors, "")
	if rulesRaw != "" {
		return rulesRaw, true
	}
	dayVal, source, _ := ResolvePolicyInt(ctx, PolicyKeyGlobalTokenQuotaDay, ancestors, -1)
	if source.Type != SourceSiteDefault {
		if dayVal < 0 {
			return "[]", true
		}
		return model.MarshalTokenQuotaRules(model.GlobalRulesFromLegacyQuota(dayVal, period)), true
	}
	return "", false
}

// resolvePolicyRaw 从祖先链中找到最近的配置行。
func resolvePolicyRaw(ctx context.Context, key string, ancestors []uint) (string, Source, error) {
	if len(ancestors) == 0 {
		return "", Source{Type: SourceSiteDefault}, nil
	}
	bindings, err := model.GetPolicyBindingsByGroups(ctx, ancestors, key)
	if err != nil {
		return "", Source{}, hcommon.I18nRichError(err, i18n.MsgResolvePolicyBindingFailed, key)
	}
	if len(bindings) == 0 {
		return "", Source{Type: SourceSiteDefault}, nil
	}
	bindingMap := make(map[uint]string, len(bindings))
	for _, b := range bindings {
		bindingMap[b.GroupID] = b.ValueJSON
	}
	for i, gid := range ancestors {
		if val, ok := bindingMap[gid]; ok {
			sourceType := SourceLocal
			if i > 0 {
				sourceType = SourceInherited
			}
			return val, Source{
				Type:     sourceType,
				GroupID:  gid,
				FullPath: getGroupFullPath(ctx, gid),
			}, nil
		}
	}
	return "", Source{Type: SourceSiteDefault}, nil
}

// getGroupFullPath 获取组全路径，失败返回空串。
func getGroupFullPath(ctx context.Context, groupID uint) string {
	groups, err := model.GetGroupsByIDs(ctx, []uint{groupID})
	if err != nil || len(groups) == 0 {
		return ""
	}
	return groups[0].FullPath
}

// ══════════════════════════════════════════════════
// 二、加法型 Resolver
// ══════════════════════════════════════════════════
// 可见性规则：
//   - visibility_type='all' → 所有人可见
//   - visibility_type='group' → 仅绑定的组（含祖先链）可见
//   - 多组用户取并集

// ResolveAdditiveResources 获取可见的某类加法型资源 ID 列表。
func ResolveAdditiveResources(ctx context.Context, configType string, allGroupIDs []uint) ([]uint, error) {
	if configType == ConfigTypeImageType {
		return nil, nil // 镜像类型走 ResolveImageTypes
	}
	bindings, err := model.GetBindingsByGroups(ctx, allGroupIDs, configType)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(bindings))
	result := make([]uint, 0, len(bindings))
	for _, b := range bindings {
		if _, ok := seen[b.ConfigKey]; !ok {
			seen[b.ConfigKey] = struct{}{}
			var id uint
			if _, err := parseUintStr(b.ConfigKey, &id); err == nil {
				result = append(result, id)
			}
		}
	}
	return result, nil
}

// IsResourceVisible 判断某个加法型资源对指定组列表是否可见。
func IsResourceVisible(ctx context.Context, configType string, resourceID uint, visibilityType string, allGroupIDs []uint) (bool, error) {
	if visibilityType != VisibilityGroup {
		return true, nil
	}
	if len(allGroupIDs) == 0 {
		return false, nil
	}
	configKey := fmt.Sprintf("%d", resourceID)
	var count int64
	err := model.DB(ctx).Model(&model.GroupConfigBinding{}).
		Where("config_type = ? AND config_key = ? AND group_id IN ?",
			configType, configKey, allGroupIDs).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ResolveImageTypes 获取可见的 agent_type 列表。
// 无绑定行的 agent_type 视为全部可见；有绑定行的仅对绑定组可见。
func ResolveImageTypes(ctx context.Context, allGroupIDs []uint, allAgentTypes []string) ([]string, error) {
	restricted, err := model.GetRestrictedImageTypes(ctx)
	if err != nil {
		return nil, err
	}
	if len(restricted) == 0 {
		return model.FilterEnabledAgentTypes(ctx, allAgentTypes), nil
	}
	restrictedSet := make(map[string]struct{}, len(restricted))
	for _, r := range restricted {
		restrictedSet[r] = struct{}{}
	}
	visibleRestricted := make(map[string]struct{})
	if len(allGroupIDs) > 0 {
		visible, err := model.GetVisibleImageTypesByGroups(ctx, allGroupIDs)
		if err != nil {
			return nil, err
		}
		for _, v := range visible {
			visibleRestricted[v] = struct{}{}
		}
	}
	result := make([]string, 0, len(allAgentTypes))
	for _, t := range allAgentTypes {
		if _, isRestricted := restrictedSet[t]; !isRestricted {
			result = append(result, t)
		} else if _, isVisible := visibleRestricted[t]; isVisible {
			result = append(result, t)
		}
	}
	return model.FilterEnabledAgentTypes(ctx, result), nil
}

// ══════════════════════════════════════════════════
// 三、角色可见性
// ══════════════════════════════════════════════════

// IsRoleVisibleToGroups 检查角色对指定分组列表是否可见。
// visibility_type 不是 "group" 则对所有组可见；groupIDs 为空则不限制。
func IsRoleVisibleToGroups(ctx context.Context, roleID uint, groupIDs []uint) bool {
	if len(groupIDs) == 0 {
		return true
	}
	var role model.OpenClawRole
	if err := model.DB(ctx).First(&role, roleID).Error; err != nil {
		return false
	}
	if role.VisibilityType == VisibilityAll || role.VisibilityType == "" {
		return true
	}
	if role.VisibilityType != VisibilityGroup {
		return false // 未知类型不可见
	}
	var count int64
	model.DB(ctx).Model(&model.RoleVisibilityGroup{}).
		Where("open_claw_role_id = ? AND group_id IN ?", roleID, groupIDs).
		Count(&count)
	return count > 0
}

// IsRoleGloballyVisible 判断角色是否为全局可见（visibility_type='all' 或空）。
// 用于未分组用户（group_id=0）的角色校验。
func IsRoleGloballyVisible(ctx context.Context, roleID uint) bool {
	var role model.OpenClawRole
	if err := model.DB(ctx).First(&role, roleID).Error; err != nil {
		return false
	}
	return role.VisibilityType == VisibilityAll || role.VisibilityType == ""
}

// ══════════════════════════════════════════════════
// 辅助函数
// ══════════════════════════════════════════════════

func parseUintStr(s string, out *uint) (int, error) {
	var v uint64
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		v = v*10 + uint64(c-'0')
		n++
	}
	if n == 0 {
		return 0, hcommon.I18nError(i18n.MsgResolveInvalidUint, s)
	}
	*out = uint(v)
	return n, nil
}

// ══════════════════════════════════════════════════
// 四、用户端可见性过滤
// ══════════════════════════════════════════════════
// 按 agent 绑定的 group_id 过滤资源列表：
//   - agentGroupID > 0: 返回 visibility_type='all' + 绑定到该组祖先链的 'group' 类资源
//   - agentGroupID == 0: 仅返回 visibility_type!='group' 的资源

// FilterModelsByVisibility 按 agent 分组过滤模型列表。
// 固定 2 次 DB 查询（祖先链 + 批量绑定），其余内存过滤。
func FilterModelsByVisibility(ctx context.Context, models []model.AIModel, agentGroupID uint) []model.AIModel {
	// 获取祖先链
	var userGroupSet map[uint]bool
	if agentGroupID > 0 {
		ancestors, err := GetAncestorIDs(ctx, agentGroupID)
		if err == nil && len(ancestors) > 0 {
			userGroupSet = make(map[uint]bool, len(ancestors))
			for _, gid := range ancestors {
				userGroupSet[gid] = true
			}
		}
	}

	// 收集需要检查分组的模型 ID
	var groupModelIDs []uint
	for _, m := range models {
		if m.VisibilityType == VisibilityGroup {
			groupModelIDs = append(groupModelIDs, m.ID)
		}
	}

	// 批量查绑定关系
	var modelGroupMap map[uint][]uint
	if len(groupModelIDs) > 0 {
		modelGroupMap, _ = model.GetModelVisibilityGroupIDs(ctx, groupModelIDs)
	}

	// 内存过滤
	result := make([]model.AIModel, 0, len(models))
	for _, m := range models {
		if m.VisibilityType == VisibilityAll {
			result = append(result, m)
			continue
		}
		if m.VisibilityType == VisibilityGroup {
			if userGroupSet == nil {
				continue // agentGroupID=0，group 类型不可见
			}
			for _, gid := range modelGroupMap[m.ID] {
				if userGroupSet[gid] {
					result = append(result, m)
					break
				}
			}
		}
	}
	return result
}

// FilterChannelsByVisibility 按 agent 分组过滤通道列表。
// 固定 2 次 DB 查询（祖先链 + 批量绑定），其余内存过滤。
func FilterChannelsByVisibility(ctx context.Context, channels []model.AIChannel, agentGroupID uint) []model.AIChannel {
	var userGroupSet map[uint]bool
	if agentGroupID > 0 {
		ancestors, err := GetAncestorIDs(ctx, agentGroupID)
		if err == nil && len(ancestors) > 0 {
			userGroupSet = make(map[uint]bool, len(ancestors))
			for _, gid := range ancestors {
				userGroupSet[gid] = true
			}
		}
	}

	var groupChannelIDs []uint
	for _, ch := range channels {
		if ch.VisibilityType == VisibilityGroup {
			groupChannelIDs = append(groupChannelIDs, ch.ID)
		}
	}

	var channelGroupMap map[uint][]uint
	if len(groupChannelIDs) > 0 {
		channelGroupMap, _ = model.GetResourceVisibilityGroupIDsByUint(ctx, model.ConfigTypeChannel, groupChannelIDs)
	}

	result := make([]model.AIChannel, 0, len(channels))
	for _, ch := range channels {
		if ch.VisibilityType == VisibilityAll {
			result = append(result, ch)
			continue
		}
		if ch.VisibilityType == VisibilityGroup {
			if userGroupSet == nil {
				continue
			}
			for _, gid := range channelGroupMap[ch.ID] {
				if userGroupSet[gid] {
					result = append(result, ch)
					break
				}
			}
		}
	}
	return result
}

// FilterPluginBundlesByVisibility 按 agent 分组过滤插件包列表。
// 固定 2 次 DB 查询（祖先链 + 批量绑定），其余内存过滤。
func FilterPluginBundlesByVisibility(ctx context.Context, bundles []model.PluginBundle, agentGroupID uint) []model.PluginBundle {
	var userGroupSet map[uint]bool
	if agentGroupID > 0 {
		ancestors, err := GetAncestorIDs(ctx, agentGroupID)
		if err == nil && len(ancestors) > 0 {
			userGroupSet = make(map[uint]bool, len(ancestors))
			for _, gid := range ancestors {
				userGroupSet[gid] = true
			}
		}
	}

	var groupBundleIDs []uint
	for _, b := range bundles {
		if b.VisibilityType == VisibilityGroup {
			groupBundleIDs = append(groupBundleIDs, b.ID)
		}
	}

	var bundleGroupMap map[uint][]uint
	if len(groupBundleIDs) > 0 {
		bundleGroupMap, _ = model.GetResourceVisibilityGroupIDsByUint(ctx, model.ConfigTypePluginBundle, groupBundleIDs)
	}

	result := make([]model.PluginBundle, 0, len(bundles))
	for _, b := range bundles {
		if b.VisibilityType == VisibilityAll {
			result = append(result, b)
			continue
		}
		if b.VisibilityType == VisibilityGroup {
			if userGroupSet == nil {
				continue
			}
			for _, gid := range bundleGroupMap[b.ID] {
				if userGroupSet[gid] {
					result = append(result, b)
					break
				}
			}
		}
	}
	return result
}

// ══════════════════════════════════════════════════
// 五、管理端可见性分组引用（Enriching）
// ══════════════════════════════════════════════════

// VisibilityGroupRef 管理端响应中资源绑定的组引用
type VisibilityGroupRef struct {
	GroupID   uint   `json:"group_id"`
	GroupName string `json:"group_name"`
}

// GetVisibilityGroupRefs 批量获取多个资源的可见性分组引用（管理端列表用）。
// configType: model.ConfigTypeChannel / ConfigTypePluginBundle / ConfigTypeMCP 等
// resourceIDs: 需要查询的资源 ID 列表（仅 visibility_type='group' 的传入即可）
// 返回: map[resourceID] → []VisibilityGroupRef
func GetVisibilityGroupRefs(ctx context.Context, configType string, resourceIDs []uint) map[uint][]VisibilityGroupRef {
	if len(resourceIDs) == 0 {
		return nil
	}

	result := make(map[uint][]VisibilityGroupRef, len(resourceIDs))

	// 收集所有绑定关系 + 所有涉及的 groupID
	allGroupIDs := make(map[uint]struct{})
	bindingsPerResource := make(map[uint][]uint, len(resourceIDs))

	for _, rid := range resourceIDs {
		bindings, err := model.GetBindingsByResource(ctx, configType, fmt.Sprintf("%d", rid))
		if err != nil || len(bindings) == 0 {
			continue
		}
		gids := make([]uint, 0, len(bindings))
		for _, b := range bindings {
			gids = append(gids, b.GroupID)
			allGroupIDs[b.GroupID] = struct{}{}
		}
		bindingsPerResource[rid] = gids
	}

	if len(allGroupIDs) == 0 {
		return result
	}

	// 一次性查询所有组名
	gidSlice := make([]uint, 0, len(allGroupIDs))
	for gid := range allGroupIDs {
		gidSlice = append(gidSlice, gid)
	}
	groups, _ := model.GetGroupsByIDs(ctx, gidSlice)
	nameMap := make(map[uint]string, len(groups))
	for _, g := range groups {
		nameMap[g.ID] = g.Name
	}

	// 构建结果
	for rid, gids := range bindingsPerResource {
		refs := make([]VisibilityGroupRef, 0, len(gids))
		for _, gid := range gids {
			refs = append(refs, VisibilityGroupRef{GroupID: gid, GroupName: nameMap[gid]})
		}
		result[rid] = refs
	}
	return result
}

// GetVisibilityGroupRefsStr 与 GetVisibilityGroupRefs 类似，但 configKey 为字符串。
// 用于镜像类型等 configKey 不是纯数字 ID 的场景。
// 返回: map[configKey] → []VisibilityGroupRef
func GetVisibilityGroupRefsStr(ctx context.Context, configType string, configKeys []string) map[string][]VisibilityGroupRef {
	if len(configKeys) == 0 {
		return nil
	}

	result := make(map[string][]VisibilityGroupRef, len(configKeys))
	allGroupIDs := make(map[uint]struct{})
	bindingsPerKey := make(map[string][]uint, len(configKeys))

	for _, key := range configKeys {
		bindings, err := model.GetBindingsByResource(ctx, configType, key)
		if err != nil || len(bindings) == 0 {
			continue
		}
		gids := make([]uint, 0, len(bindings))
		for _, b := range bindings {
			gids = append(gids, b.GroupID)
			allGroupIDs[b.GroupID] = struct{}{}
		}
		bindingsPerKey[key] = gids
	}

	if len(allGroupIDs) == 0 {
		return result
	}

	gidSlice := make([]uint, 0, len(allGroupIDs))
	for gid := range allGroupIDs {
		gidSlice = append(gidSlice, gid)
	}
	groups, _ := model.GetGroupsByIDs(ctx, gidSlice)
	nameMap := make(map[uint]string, len(groups))
	for _, g := range groups {
		nameMap[g.ID] = g.Name
	}

	for key, gids := range bindingsPerKey {
		refs := make([]VisibilityGroupRef, 0, len(gids))
		for _, gid := range gids {
			refs = append(refs, VisibilityGroupRef{GroupID: gid, GroupName: nameMap[gid]})
		}
		result[key] = refs
	}
	return result
}

// ══════════════════════════════════════════════════
// 六、VPC 配置解析
// ══════════════════════════════════════════════════

// ResolveVpcConfig 按分组祖先链解析 VPC 配置（最近祖先覆盖语义）。
// groupID=0 时直接返回全局兜底配置。
func ResolveVpcConfig(ctx context.Context, groupID uint, globalVpcId string, globalSubnetMap map[string][]string) (string, map[string][]string) {
	if groupID > 0 {
		// 收集当前组 + 所有祖先（按从近到远排列）
		candidates := []uint{groupID}
		ancestors, err := model.ClosureAncestors(ctx, groupID, false)
		if err == nil {
			candidates = append(candidates, ancestors...)
		}

		// 查询所有候选分组的 vpc 绑定
		bindings, err := model.GetBindingsByGroups(ctx, candidates, model.ConfigTypeVPC)
		if err == nil && len(bindings) > 0 {
			// 按祖先链顺序（从近到远）匹配第一条
			bindingMap := make(map[uint]model.GroupConfigBinding)
			for _, b := range bindings {
				bindingMap[b.GroupID] = b
			}
			for _, gid := range candidates {
				if b, ok := bindingMap[gid]; ok {
					var vpcConfigID uint
					fmt.Sscanf(b.ConfigKey, "%d", &vpcConfigID)
					var vpcConfig model.VpcConfig
					if model.DB(ctx).First(&vpcConfig, vpcConfigID).Error == nil {
						subnetMap, _ := vpcConfig.GetSubnetMap()
						return vpcConfig.VpcId, subnetMap
					}
				}
			}
		}
	}

	// 兜底：全局配置
	return globalVpcId, globalSubnetMap
}
