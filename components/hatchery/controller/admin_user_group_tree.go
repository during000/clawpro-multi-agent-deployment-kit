package controller

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

const defaultPageSize = 200

// HandleAdminGetGroupTree §1 /admin/user-groups/tree
//
// GET /admin/user-groups/tree?q=&with_user_counts=&with_resource_policy=&sources=
//
// 🆕 v6.12 P1：列表树（oneid_dept + manual）一次性拉取，含 summary / ungrouped。
func HandleAdminGetGroupTree(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	opts := usergroup.TreeOptions{
		Query:              strings.TrimSpace(r.URL.Query().Get("q")),
		WithUserCounts:     parseBoolQueryDefault(r, "with_user_counts", true),
		WithHealth:         parseBoolQuery(r, "with_health"),
		WithResourcePolicy: parseBoolQuery(r, "with_resource_policy"),
		Sources:            parseCSVQuery(r, "sources"),
	}
	resp, err := usergroup.GetGroupTree(r.Context(), opts)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok":          true,
		"summary":     resp.Summary,
		"org_tree":    resp.OrgTree,
		"user_groups": resp.UserGroups,
		"ungrouped":   resp.Ungrouped,
	})
}

// parseCSVQuery 解析英文逗号分隔的 query 参数为 []string。
// 自动去除每个段的前后空白，并丢弃空段。未传参数或全为空白时返回 nil。
func parseCSVQuery(r *http.Request, key string) []string {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseBoolQuery(r *http.Request, key string) bool {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return false
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// parseBoolQueryDefault 语义同 parseBoolQuery，但允许调用方指定"未传时的默认值"。
// 仅当参数被显式提供且值为真/假关键字时才覆盖默认。
// 用于需要"未传=true"语义的查询参数（如 with_user_counts）。
func parseBoolQueryDefault(r *http.Request, key string, defaultVal bool) bool {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return defaultVal
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return defaultVal
}

// HandleAdminGetGroupMembers §2 /admin/user-groups/members
//
// GET /admin/user-groups/members?id=&page=&page_size=&include_descendants=&q=
//
// 🆕 v6.5：路径调整。🆕 v6.14：members[] 项含 direct_groups[{id, full_path, source, is_main}]。
func HandleAdminGetGroupMembers(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	idStr := strings.TrimSpace(r.URL.Query().Get("id"))
	if idStr == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserGroupIDRequired))
		return
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUserGroupIDFormatError))
		return
	}

	page, pageSize := parsePagination(r, defaultPageSize)
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	// id=0 代表查询未分组用户
	if id == 0 {
		resp, err := usergroup.GetUngroupedMembersPaged(r.Context(), page, pageSize, q)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
			return
		}
		jsonOK(w, map[string]interface{}{
			"ok":        true,
			"total":     resp.Total,
			"page":      resp.Page,
			"page_size": resp.PageSize,
			"members":   resp.Members,
		})
		return
	}

	opts := usergroup.MembersOptions{
		GroupID:            uint(id),
		Page:               page,
		PageSize:           pageSize,
		Query:              q,
		IncludeDescendants: parseBoolQuery(r, "include_descendants"),
	}

	resp, err := usergroup.GetGroupMembersPaged(r.Context(), opts)
	if err != nil {
		writeError(w, r, mapGroupErrToHTTP(err), hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok":        true,
		"total":     resp.Total,
		"page":      resp.Page,
		"page_size": resp.PageSize,
		"members":   resp.Members,
	})
}

// HandleAdminGetGroupConfigOverview 配置总览
//
// GET /admin/user-groups/config-overview?group_ids=3,7&keys=model,channel
//
// 按 category 返回配置项列表，支持多组查询和 keys 过滤。
// group_ids: 必传，逗号分隔的用户组 ID 数组；group_ids=0 代表查询全局默认配置（未分组用户视角）
// keys: 可选，逗号分隔的 category key 过滤；不传返回全部 category
func HandleAdminGetGroupConfigOverview(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	// 解析 group_ids（必传）
	groupIDsStr := strings.TrimSpace(r.URL.Query().Get("group_ids"))
	if groupIDsStr == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "group_ids"))
		return
	}

	groupIDs, err := parseUintCSV(groupIDsStr)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamFormatError, "group_ids"))
		return
	}
	if len(groupIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "group_ids"))
		return
	}

	// group_ids=[0] 代表全局默认配置（未分组用户视角），跳过存在性校验
	isUngrouped := len(groupIDs) == 1 && groupIDs[0] == 0
	if !isUngrouped {
		// 校验用户组存在性
		if err := usergroup.ValidateGroupIDs(r.Context(), groupIDs); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPartialUserGroupsNotFound))
			return
		}
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

	siteConfig := model.GetSiteConfig(r.Context())

	// 为每个组构建配置总览
	type groupOverview struct {
		GroupID    uint                             `json:"group_id"`
		Categories []usergroup.ConfigCategoryResult `json:"categories"`
	}

	results := make([]groupOverview, 0, len(groupIDs))
	for _, gid := range groupIDs {
		// 获取祖先链
		ancestors, err := model.ClosureAncestors(r.Context(), gid, true)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
			return
		}

		categories := buildCategoriesForGroup(r.Context(), gid, ancestors, &siteConfig, keyFilter)
		results = append(results, groupOverview{
			GroupID:    gid,
			Categories: categories,
		})
	}

	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"results": results,
	})
}

// isValidCategoryKey 校验 category key 是否合法
func isValidCategoryKey(key string) bool {
	for _, meta := range usergroup.ConfigCategoryList {
		if meta.Key == key {
			return true
		}
	}
	return false
}

// parseUintCSV 解析逗号分隔的 uint 数组
func parseUintCSV(s string) ([]uint, error) {
	parts := strings.Split(s, ",")
	result := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return nil, err
		}
		result = append(result, uint(v))
	}
	return result, nil
}

// buildCategoriesForGroup 为单个组构建配置总览（支持 keyFilter 过滤）
func buildCategoriesForGroup(ctx context.Context, groupID uint, ancestors []uint, siteConfig *model.SiteConfig, keyFilter map[string]bool) []usergroup.ConfigCategoryResult {
	categories := make([]usergroup.ConfigCategoryResult, 0, len(usergroup.ConfigCategoryList))
	for _, meta := range usergroup.ConfigCategoryList {
		// 如果指定了 keys 过滤，跳过不在列表中的
		if keyFilter != nil && !keyFilter[meta.Key] {
			continue
		}

		cat := usergroup.ConfigCategoryResult{
			Key:         meta.Key,
			Label:       i18n.T(ctx, i18n.NewKey(meta.Label)),
			Description: meta.Description,
			Icon:        meta.Icon,
			Entries:     []usergroup.ConfigEntry{},
		}

		switch meta.Key {
		case usergroup.CategoryKeyChargeType:
			cat.Entries = buildChargeTypeEntries(ctx, groupID)
		case usergroup.CategoryKeyResourcePolicy:
			cat.Entries = buildResourcePolicyEntries(ctx, groupID)
		case usergroup.CategoryKeyModel:
			cat.Entries = buildModelEntries(ctx, groupID, ancestors)
		case usergroup.CategoryKeyChannel:
			cat.Entries = buildChannelEntries(ctx, groupID, ancestors)
		case usergroup.CategoryKeySkill:
			cat.Entries = buildSkillEntries(ctx, groupID, ancestors, siteConfig)
		case usergroup.CategoryKeyAgentTool:
			cat.Entries = buildAgentToolEntries(ctx, groupID, ancestors)
		case usergroup.CategoryKeyMemory:
			cat.Entries = buildMemoryEntries(ctx, ancestors, siteConfig)
		case usergroup.CategoryKeyDrive:
			cat.Entries = buildDriveEntries(ctx, ancestors, siteConfig)
		case usergroup.CategoryKeyImageType:
			cat.Entries = buildImageTypeEntries(ctx, groupID, ancestors)
		case usergroup.CategoryKeyNetwork:
			cat.Entries = buildNetworkEntries(ctx, groupID, ancestors, siteConfig)
		case usergroup.CategoryKeyCLS:
			cat.Entries = buildCLSEntries(ctx, groupID, siteConfig)
		case usergroup.CategoryKeyAIAgentSecurity:
			cat.Entries = buildAIAgentSecurityEntries(ctx)
		case usergroup.CategoryKeyPlatformPolicy:
			cat.Entries = buildPolicyEntries(ctx, groupID, ancestors, siteConfig)
		}

		categories = append(categories, cat)
	}
	return categories
}

// ──────────────────────────────────────────────
// 各 category 的 entries 构建函数
// ──────────────────────────────────────────────

// buildResourcePolicyEntries renders the effective resource policy.
func buildResourcePolicyEntries(ctx context.Context, groupID uint) []usergroup.ConfigEntry {
	resolved, err := model.ResolveEffectiveResourcePolicy(ctx, groupID)
	if err != nil {
		return nil
	}
	cfg, err := ParseResourceConfig(resolved.Policy.ConfigJSON)
	if err != nil {
		return nil
	}
	source := resourcePolicySource(resolved)
	if resolved.SourceGroupID != 0 {
		if groups, err := model.GetGroupsByIDs(ctx, []uint{resolved.SourceGroupID}); err == nil && len(groups) > 0 {
			source.FullPath = groups[0].FullPath
		}
	}
	return []usergroup.ConfigEntry{{
		ID:     strconv.FormatUint(uint64(resolved.Policy.ID), 10),
		Label:  resolved.Policy.DisplayName(ctx),
		Source: source,
		// value preserves the pre-redesign config-overview contract used by the
		// generic frontend renderer; resource_config is the explicit alias.
		Meta: map[string]interface{}{
			"policy_id":       resolved.Policy.ID,
			"is_default":      resolved.Policy.IsDefault,
			"resource_config": cfg,
			"value":           cfg,
		},
	}}
}

// buildChargeTypeEntries renders the charge type from the effective resource policy.
func buildChargeTypeEntries(ctx context.Context, groupID uint) []usergroup.ConfigEntry {
	resolved, err := model.ResolveEffectiveResourcePolicy(ctx, groupID)
	if err != nil {
		return nil
	}
	cfg, err := ParseResourceConfig(resolved.Policy.ConfigJSON)
	if err != nil || cfg.InstanceChargeType == "" {
		return nil
	}
	source := resourcePolicySource(resolved)
	return []usergroup.ConfigEntry{{
		ID:     cfg.InstanceChargeType,
		Label:  chargeTypeDisplayName(ctx, cfg.InstanceChargeType),
		Source: source,
	}}
}

// buildModelEntries 模型（旧表 model_visibility_groups）
// 排除自定义模型占位记录，用 Model() 自动过滤软删除，仅返回已启用模型
func buildModelEntries(ctx context.Context, groupID uint, ancestors []uint) []usergroup.ConfigEntry {
	var rows []struct {
		ID             uint   `gorm:"column:id"`
		Name           string `gorm:"column:name"`
		VisibilityType string `gorm:"column:visibility_type"`
	}
	model.DB(ctx).Model(&model.AIModel{}).
		Select("id, CASE WHEN model_name != '' THEN model_name ELSE model_id END as name, COALESCE(visibility_type,'all') as visibility_type").
		Where("enabled = ? AND visible = ?", true, true).
		Where("NOT (provider = ? AND model_id = ?)", model.BuiltinModelProvider, model.BuiltinModelID).
		Find(&rows)

	resources := make([]overviewResource, 0, len(rows))
	for _, r := range rows {
		resources = append(resources, overviewResource{ID: r.ID, Name: r.Name, VisibilityType: r.VisibilityType})
	}

	items := resolveVisibilityItems(ctx, resources, "model_visibility_groups", "ai_model_id", groupID, ancestors)
	entries := make([]usergroup.ConfigEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, usergroup.ConfigEntry{
			ID:     strconv.FormatUint(uint64(item.ResourceID), 10),
			Label:  item.ResourceName,
			Source: item.Source,
		})
	}
	return entries
}

// buildChannelEntries 通道（新表 + visibility=all）
func buildChannelEntries(ctx context.Context, groupID uint, ancestors []uint) []usergroup.ConfigEntry {
	// 预加载已启用通道的 ID→Name 映射（通过 Model 自动过滤软删除）
	type channelRow struct {
		ID        uint   `gorm:"column:id"`
		ChannelID string `gorm:"column:channel_id"`
		Name      string `gorm:"column:name"`
	}
	var channels []channelRow
	model.DB(ctx).Model(&model.AIChannel{}).Select("id, channel_id, name").Where("enabled = ?", true).Find(&channels)
	channelNameMap := make(map[string]string, len(channels))
	enabledChannelIDs := make(map[uint]bool, len(channels))
	channelIDByDBID := make(map[uint]string, len(channels))
	for _, ch := range channels {
		key := strconv.FormatUint(uint64(ch.ID), 10)
		channelNameMap[key] = ch.Name
		enabledChannelIDs[ch.ID] = true
		channelIDByDBID[ch.ID] = ch.ChannelID
	}
	channelNameFn := func(configKey string) string {
		if name, ok := channelNameMap[configKey]; ok && name != "" {
			return name
		}
		return configKey
	}

	overview, _ := usergroup.ResolveAdditiveOverview(ctx, model.ConfigTypeChannel, groupID, ancestors, channelNameFn)
	overview = appendVisibilityAllResources(ctx, overview, usergroup.ConfigTypeChannel)

	entries := make([]usergroup.ConfigEntry, 0, len(overview.Items))
	for _, item := range overview.Items {
		if item.ResourceID > 0 && !enabledChannelIDs[item.ResourceID] {
			continue
		}
		if !channelInCurrentSiteScope(ctx, channelIDByDBID[item.ResourceID]) {
			continue
		}
		entries = append(entries, usergroup.ConfigEntry{
			ID:     strconv.FormatUint(uint64(item.ResourceID), 10),
			Label:  item.ResourceName,
			Source: item.Source,
		})
	}
	return entries
}

// buildSkillEntries 技能（技能包 + 角色 + 技能安装来源）
func buildSkillEntries(ctx context.Context, groupID uint, ancestors []uint, cfg *model.SiteConfig) []usergroup.ConfigEntry {
	entries := make([]usergroup.ConfigEntry, 0)

	// 技能包（无软删除，过滤 enabled）
	var bundleRows []struct {
		ID             uint   `gorm:"column:id"`
		Name           string `gorm:"column:name"`
		VisibilityType string `gorm:"column:visibility_type"`
	}
	model.DB(ctx).Model(&model.SkillBundle{}).
		Select("id, name, COALESCE(visibility_type,'all') as visibility_type").
		Where("enabled = ?", true).
		Find(&bundleRows)
	bundleResources := make([]overviewResource, 0, len(bundleRows))
	for _, r := range bundleRows {
		bundleResources = append(bundleResources, overviewResource{ID: r.ID, Name: r.Name, VisibilityType: r.VisibilityType})
	}
	bundleItems := resolveVisibilityItems(ctx, bundleResources, "skill_bundle_visibility_groups", "skill_bundle_id", groupID, ancestors)
	for _, item := range bundleItems {
		entries = append(entries, usergroup.ConfigEntry{
			ID:       strconv.FormatUint(uint64(item.ResourceID), 10),
			Label:    item.ResourceName,
			SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelInitialSkillBundle),
			Source:   item.Source,
		})
	}

	// 角色（无软删除，过滤 visible）
	var roleRows []struct {
		ID             uint   `gorm:"column:id"`
		Name           string `gorm:"column:name"`
		VisibilityType string `gorm:"column:visibility_type"`
	}
	model.DB(ctx).Model(&model.OpenClawRole{}).
		Select("id, name, COALESCE(visibility_type,'all') as visibility_type").
		Where("visible = ?", true).
		Find(&roleRows)
	roleResources := make([]overviewResource, 0, len(roleRows))
	for _, r := range roleRows {
		roleResources = append(roleResources, overviewResource{ID: r.ID, Name: r.Name, VisibilityType: r.VisibilityType})
	}
	roleItems := resolveVisibilityItems(ctx, roleResources, "role_visibility_groups", "open_claw_role_id", groupID, ancestors)
	for _, item := range roleItems {
		entries = append(entries, usergroup.ConfigEntry{
			ID:       strconv.FormatUint(uint64(item.ResourceID), 10),
			Label:    item.ResourceName,
			SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelRole),
			Source:   item.Source,
		})
	}

	// 技能安装来源（SkillHub）
	skillHubLabel := i18n.T(ctx, i18n.MsgGroupTreeLabelDefault)
	if cfg.SkillHub != "" {
		skillHubLabel = cfg.SkillHub
	}
	entries = append(entries, usergroup.ConfigEntry{
		ID:       "skillhub",
		Label:    skillHubLabel,
		SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelSkillSource),
		Source:   usergroup.Source{Type: usergroup.SourceGlobal},
	})
	return entries
}

// buildAgentToolEntries Agent 工具（企业技能 + 企业插件 + 企业 MCP）
func buildAgentToolEntries(ctx context.Context, groupID uint, ancestors []uint) []usergroup.ConfigEntry {
	entries := make([]usergroup.ConfigEntry, 0)

	// 企业技能库（旧表 skill_visibility_groups，有软删除）
	var skillRows []struct {
		ID             uint   `gorm:"column:id"`
		Name           string `gorm:"column:name"`
		VisibilityType string `gorm:"column:visibility_type"`
	}
	model.DB(ctx).Model(&model.Skill{}).
		Select("id, name, COALESCE(visibility_type,'all') as visibility_type").
		Find(&skillRows)
	skillResources := make([]overviewResource, 0, len(skillRows))
	for _, r := range skillRows {
		skillResources = append(skillResources, overviewResource{ID: r.ID, Name: r.Name, VisibilityType: r.VisibilityType})
	}
	skillItems := resolveVisibilityItems(ctx, skillResources, "skill_visibility_groups", "skill_id", groupID, ancestors)
	for _, item := range skillItems {
		entries = append(entries, usergroup.ConfigEntry{
			ID:       strconv.FormatUint(uint64(item.ResourceID), 10),
			Label:    item.ResourceName,
			SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelEnterpriseSkill),
			Source:   item.Source,
		})
	}

	// 企业插件（Plugin 表，有软删除，不支持分组，全部用户可见）
	var pluginRows []struct {
		ID   uint   `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	model.DB(ctx).Model(&model.Plugin{}).Select("id, name").Find(&pluginRows)
	for _, p := range pluginRows {
		entries = append(entries, usergroup.ConfigEntry{
			ID:       strconv.FormatUint(uint64(p.ID), 10),
			Label:    p.Name,
			SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelEnterprisePlugin),
			Source:   usergroup.Source{Type: usergroup.SourceAllUsers},
		})
	}

	// 企业 MCP（无软删除）
	var mcpRows []struct {
		ID   uint   `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	model.DB(ctx).Model(&model.McpServer{}).Select("id, name").Find(&mcpRows)
	mcpNameMap := make(map[string]string, len(mcpRows))
	for _, m := range mcpRows {
		mcpNameMap[strconv.FormatUint(uint64(m.ID), 10)] = m.Name
	}
	mcpNameFn := func(configKey string) string {
		if name, ok := mcpNameMap[configKey]; ok && name != "" {
			return name
		}
		return configKey
	}
	mcpOverview, _ := usergroup.ResolveAdditiveOverview(ctx, model.ConfigTypeMCP, groupID, ancestors, mcpNameFn)
	mcpOverview = appendVisibilityAllResources(ctx, mcpOverview, usergroup.ConfigTypeMCP)
	for _, item := range mcpOverview.Items {
		entries = append(entries, usergroup.ConfigEntry{
			ID:       strconv.FormatUint(uint64(item.ResourceID), 10),
			Label:    item.ResourceName,
			SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelEnterpriseMCP),
			Source:   item.Source,
		})
	}
	return entries
}

// buildMemoryEntries 记忆（支持分组策略覆盖）
func buildMemoryEntries(ctx context.Context, ancestors []uint, cfg *model.SiteConfig) []usergroup.ConfigEntry {
	plan := cfg.MemoryDefaultPlan
	if len(ancestors) > 0 {
		var policy model.MemoryPlanGroupPolicy
		if err := model.DB(ctx).Where("group_id IN ?", ancestors).First(&policy).Error; err == nil {
			plan = policy.Plan
		}
	}
	label := i18n.T(ctx, i18n.MsgGroupTreeLabelOff)
	if cfg.MemoryTDAIEnable {
		if plan == model.MemoryDefaultPlanPro {
			label = i18n.T(ctx, i18n.MsgGroupTreeMemoryProEdition)
		} else {
			label = i18n.T(ctx, i18n.MsgGroupTreeMemoryFreeEdition)
		}
	}
	return []usergroup.ConfigEntry{
		{
			ID:     "tdai",
			Label:  label,
			Source: usergroup.Source{Type: usergroup.SourceGlobal},
			Meta:   map[string]interface{}{"enabled": cfg.MemoryTDAIEnable, "plan": plan},
		},
	}
}

// buildDriveEntries 网盘（按分组策略解析）
// 解析 smh_auto_provision 策略，返回带来源标记的结果。
// 未命中任何分组策略时，回退到 site_configs.SMHAutoProvisionOnCreate 全局默认值。
func buildDriveEntries(ctx context.Context, ancestors []uint, cfg *model.SiteConfig) []usergroup.ConfigEntry {
	fallback := cfg.SMHAutoProvisionOnCreate
	enabled, source, _ := usergroup.ResolvePolicyBool(ctx, usergroup.PolicyKeySMHAutoProvision, ancestors, fallback)

	label := i18n.T(ctx, i18n.MsgGroupTreeLabelOff)
	if enabled {
		label = i18n.T(ctx, i18n.MsgGroupTreeLabelOn)
	}
	return []usergroup.ConfigEntry{
		{
			ID:     "smh",
			Label:  label,
			Source: source,
			Meta:   map[string]interface{}{"enabled": enabled},
		},
	}
}

// buildImageTypeEntries 镜像类型（新表 + 已启用镜像）
func buildImageTypeEntries(ctx context.Context, groupID uint, ancestors []uint) []usergroup.ConfigEntry {
	imageOverview, _ := usergroup.ResolveAdditiveOverview(ctx, model.ConfigTypeImageType, groupID, ancestors, nil)
	imageOverview = appendVisibilityAllResources(ctx, imageOverview, usergroup.ConfigTypeImageType)

	// 查询所有有已启用镜像的 agent_type 集合，过滤掉全部未启用的
	enabledImageTypes := getEnabledImageTypes(ctx)

	entries := make([]usergroup.ConfigEntry, 0, len(imageOverview.Items))
	for _, item := range imageOverview.Items {
		if !enabledImageTypes[item.ResourceName] {
			continue
		}
		displayName := item.ResourceName
		if name, ok := model.AgentTypeDisplayNames[item.ResourceName]; ok {
			displayName = name
		}
		entries = append(entries, usergroup.ConfigEntry{
			ID:     item.ResourceName,
			Label:  displayName,
			Source: item.Source,
		})
	}
	return entries
}

// buildNetworkEntries 网络配置
// 按产品 demo 分三个子板块：私有网络与子网 / 安全组 / 公网
// VPC 部分支持分组解析（仿照 channel），安全组/公网仍为全局。
func buildNetworkEntries(ctx context.Context,
	groupID uint, ancestors []uint, cfg *model.SiteConfig,
) []usergroup.ConfigEntry {
	entries := make([]usergroup.ConfigEntry, 0)

	// 私有网络与子网：分组感知
	entries = append(entries, buildVpcSubnetEntries(ctx, groupID, ancestors, cfg)...)

	// 安全组（全局）— 优先取默认 RuleSet 下的 ACTIVE SG ID，为空回退 site_config
	globalSource := usergroup.Source{Type: usergroup.SourceGlobal}
	var sgIDs []string
	if rs, err := model.GetDefaultRuleSet(ctx); err == nil {
		if sgs, err := model.ListActiveSGsByRuleSet(ctx, rs.ID); err == nil {
			for _, sg := range sgs {
				sgIDs = append(sgIDs, sg.SGID)
			}
		}
	}
	if len(sgIDs) == 0 && cfg.SecurityGroupId != "" {
		sgIDs = []string{cfg.SecurityGroupId}
	}
	for _, sgID := range sgIDs {
		entries = append(entries, usergroup.ConfigEntry{
			ID: sgID, Label: sgID, SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelSecurityGroup),
			Source: globalSource,
		})
	}

	// 公网（全局）— config-overview 返回单条 public-ip + meta（原始格式）
	// config-diff 的 3 子行拆分由 targetInternetEntries 独立处理，不影响此处
	if overview, err := model.ParseCVMTemplateOverview(
		cfg.CVMTemplate); err == nil && overview != nil && overview.InternetAccessible != nil {
		ia := overview.InternetAccessible
		entries = append(entries, usergroup.ConfigEntry{
			ID: "public-ip", Label: i18n.T(ctx, i18n.MsgGroupTreeLabelInternetConfig), SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelInternet),
			Source: globalSource,
			Meta: map[string]interface{}{
				"public_ip_assigned":         ia.PublicIpAssigned,
				"internet_charge_type":       ia.InternetChargeType,
				"internet_max_bandwidth_out": ia.InternetMaxBandwidthOut,
			},
		})
	}
	return entries
}

// targetInternetEntries 组侧公网三子项（T28）—— 全部归属同一 SubLabel="公网"，
// 让 buildSubLabelRows 合并成一行，instance_values / target_values 各含 3 项。
// ID 是子项标签（"公网 IP"/"计费模式"/"带宽上限"），Label 是值（"是"/"按流量计费"/"5 Mbps"），全部走 i18n。
// computeRowStatus 用 ID+Name 做集合比较：3 项都同 → same，任一不同 → different + highlight_keys 含该子项 ID。
func targetInternetEntries(ctx context.Context, ia *model.InternetAccessible, src usergroup.Source) []usergroup.ConfigEntry {
	subInternet := i18n.T(ctx, i18n.MsgGroupTreeSubLabelInternet)
	idIP := i18n.T(ctx, i18n.MsgGroupTreeSubLabelInternetPublicIP)
	idCT := i18n.T(ctx, i18n.MsgGroupTreeSubLabelInternetChargeType)
	idBW := i18n.T(ctx, i18n.MsgGroupTreeSubLabelInternetBandwidth)
	ipLabel := publicIPAssignedLabel(ctx, ia.PublicIpAssigned)
	chargeLabel := internetChargeTypeDisplayName(ctx, ia.InternetChargeType)
	bwLabel := bandwidthDisplayName(int64(ia.InternetMaxBandwidthOut))
	return []usergroup.ConfigEntry{
		{ID: idIP, Label: ipLabel, SubLabel: subInternet, Source: src, NameHint: idIP},
		{ID: idCT, Label: chargeLabel, SubLabel: subInternet, Source: src, NameHint: idCT},
		{ID: idBW, Label: bwLabel, SubLabel: subInternet, Source: src, NameHint: idBW},
	}
}

// buildVpcSubnetEntries 解析分组级别的 VPC 配置，仿照 channel 的加法型逻辑。
// 优先级：分组绑定(local/inherited) > visibility_type=all > site_configs(site_default)。
func buildVpcSubnetEntries(ctx context.Context,
	groupID uint, ancestors []uint, cfg *model.SiteConfig,
) []usergroup.ConfigEntry {
	entries := make([]usergroup.ConfigEntry, 0)

	// 覆盖语义：按祖先链从近到远查找，取第一条命中的 VPC 配置
	if groupID > 0 && len(ancestors) > 0 {
		bindings, err := model.GetBindingsByGroups(ctx, ancestors, model.ConfigTypeVPC)
		if err == nil && len(bindings) > 0 {
			// 按祖先链顺序匹配第一条
			bindingMap := make(map[uint]model.GroupConfigBinding)
			for _, b := range bindings {
				bindingMap[b.GroupID] = b
			}
			for _, gid := range ancestors {
				b, ok := bindingMap[gid]
				if !ok {
					continue
				}
				var vpcCfg model.VpcConfig
				if model.DB(ctx).First(&vpcCfg, b.ConfigKey).Error != nil {
					continue
				}
				// 确定 source
				sourceType := usergroup.SourceInherited
				if gid == groupID {
					sourceType = usergroup.SourceLocal
				}
				var grp model.UserGroup
				model.DB(ctx).Select("full_path").First(&grp, gid)
				source := usergroup.Source{
					Type:     sourceType,
					GroupID:  gid,
					FullPath: grp.FullPath,
				}
				return expandVpcConfig(ctx, vpcCfg, source)
			}
		}
	}

	// 未命中：fallback 到全局 site_configs
	if cfg.VpcId != "" {
		// 管理员预设策略：显示真实 VPC/子网 ID
		siteSource := usergroup.Source{Type: usergroup.SourceSiteDefault}
		entries = append(entries, usergroup.ConfigEntry{
			ID: cfg.VpcId, Label: cfg.VpcId, SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelVpcSubnet),
			Source: siteSource,
			Meta:   map[string]interface{}{"type": "vpc"},
		})
		for zone, subnets := range cfg.GetSubnetMap() {
			for _, subnetId := range subnets {
				entries = append(entries, usergroup.ConfigEntry{
					ID: subnetId, Label: subnetId, SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelVpcSubnet),
					Source: siteSource,
					Meta:   map[string]interface{}{"type": "subnet", "zone": zone},
				})
			}
		}
	} else {
		// 自动分配模式（VpcId 为空）：VPC 和子网 ID/Label 统一显示"自动分配"
		siteSource := usergroup.Source{Type: usergroup.SourceSiteDefault}
		autoAssignLabel := i18n.T(ctx, i18n.MsgGroupTreeAutoAssign)
		entries = append(entries, usergroup.ConfigEntry{
			ID: autoAssignLabel, Label: autoAssignLabel, SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelVpcSubnet),
			Source: siteSource,
			Meta:   map[string]interface{}{"type": "vpc"},
		})
		for zone, subnets := range cfg.GetDefaultSubnetMap() {
			for range subnets {
				entries = append(entries, usergroup.ConfigEntry{
					ID: autoAssignLabel, Label: autoAssignLabel, SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelVpcSubnet),
					Source: siteSource,
					Meta:   map[string]interface{}{"type": "subnet", "zone": zone},
				})
			}
		}
	}
	return entries
}

// expandVpcConfig 将一条 vpc_configs 展开为 VPC 主条目 + 子网子条目。
func expandVpcConfig(ctx context.Context,
	vpcCfg model.VpcConfig, source usergroup.Source,
) []usergroup.ConfigEntry {
	entries := make([]usergroup.ConfigEntry, 0)
	subnetMap, _ := vpcCfg.GetSubnetMap()

	meta := map[string]interface{}{"type": "vpc"}
	if vpcCfg.StrategyName != "" {
		meta["strategy_name"] = vpcCfg.StrategyName
	}
	entries = append(entries, usergroup.ConfigEntry{
		ID: vpcCfg.VpcId, Label: vpcCfg.VpcId, SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelVpcSubnet),
		Source: source,
		Meta:   meta,
	})

	for zone, subnets := range subnetMap {
		for _, subnetId := range subnets {
			entries = append(entries, usergroup.ConfigEntry{
				ID: subnetId, Label: subnetId, SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelVpcSubnet),
				Source: source,
				Meta:   map[string]interface{}{"type": "subnet", "zone": zone},
			})
		}
	}
	return entries
}

// buildCLSEntries CLS 日志（全局配置 + 分组 scope 感知）
func buildCLSEntries(ctx context.Context, groupID uint, cfg *model.SiteConfig) []usergroup.ConfigEntry {
	if cfg.CLSEnabled != 1 {
		return []usergroup.ConfigEntry{
			{
				ID:     usergroup.CategoryKeyCLS,
				Label:  i18n.T(ctx, i18n.MsgGroupTreeLabelOff),
				Source: usergroup.Source{Type: usergroup.SourceGlobal},
				Meta:   map[string]interface{}{"enabled": false},
			},
		}
	}

	scopeGroupIDs, err := model.GetCLSCollectScopeGroupIDs(ctx)
	if err != nil {
		slog.Error("[CLS] 查询采集范围分组失败", "error", err)
		// 降级为全量模式展示
		return []usergroup.ConfigEntry{
			{
				ID:     usergroup.CategoryKeyCLS,
				Label:  i18n.T(ctx, i18n.MsgGroupTreeLabelOn),
				Source: usergroup.Source{Type: usergroup.SourceGlobal},
				Meta:   map[string]interface{}{"enabled": true, "scope_type": "all"},
			},
		}
	}
	if len(scopeGroupIDs) == 0 {
		// 全量模式
		return []usergroup.ConfigEntry{
			{
				ID:     usergroup.CategoryKeyCLS,
				Label:  i18n.T(ctx, i18n.MsgGroupTreeLabelOn),
				Source: usergroup.Source{Type: usergroup.SourceGlobal},
				Meta:   map[string]interface{}{"enabled": true, "scope_type": "all"},
			},
		}
	}

	// 按分组模式：展开 scope 所有子孙，判断当前组是否命中
	allScopeGroups, err := model.ExpandGroupIDsWithDescendants(ctx, scopeGroupIDs)
	if err != nil {
		slog.Error("[CLS] 展开分组子孙失败", "scope_group_ids", scopeGroupIDs, "error", err)
		// 降级为未命中
		return []usergroup.ConfigEntry{
			{
				ID:     usergroup.CategoryKeyCLS,
				Label:  i18n.T(ctx, i18n.MsgGroupTreeLabelOff),
				Source: usergroup.Source{Type: usergroup.SourceGlobal},
				Meta:   map[string]interface{}{"enabled": false, "scope_type": "group"},
			},
		}
	}
	inScope := false
	for _, gid := range allScopeGroups {
		if gid == groupID {
			inScope = true
			break
		}
	}

	label := i18n.T(ctx, i18n.MsgGroupTreeLabelOff)
	if inScope {
		label = i18n.T(ctx, i18n.MsgGroupTreeLabelOn)
	}
	return []usergroup.ConfigEntry{
		{
			ID:     usergroup.CategoryKeyCLS,
			Label:  label,
			Source: usergroup.Source{Type: usergroup.SourceGlobal},
			Meta:   map[string]interface{}{"enabled": inScope, "scope_type": "group"},
		},
	}
}

// buildAIAgentSecurityEntries AI Agent 安全
// 每个新建实例默认为基础版，版本升级在 AI Agent 安全管控页面按资产操作
func buildAIAgentSecurityEntries(ctx context.Context) []usergroup.ConfigEntry {
	return []usergroup.ConfigEntry{
		{
			ID:     usergroup.CategoryKeyAIAgentSecurity,
			Label:  i18n.T(ctx, i18n.MsgGroupTreeAIAgentSecurityBasic),
			Source: usergroup.Source{Type: usergroup.SourceGlobal},
			Meta:   map[string]interface{}{"protect_level": 0},
		},
	}
}

// buildPolicyEntries 平台策略
func buildPolicyEntries(ctx context.Context, groupID uint, ancestors []uint, cfg *model.SiteConfig) []usergroup.ConfigEntry {
	policyItems, _ := usergroup.ResolvePolicyOverview(ctx, groupID, ancestors, cfg)
	entries := make([]usergroup.ConfigEntry, 0, len(policyItems))

	for _, item := range policyItems {
		entry := usergroup.ConfigEntry{
			ID:     item.Key,
			Label:  i18n.T(ctx, i18n.NewKey(item.Label)),
			Source: item.Source,
		}
		// 分配 sub_label
		switch item.Key {
		case usergroup.PolicyKeyTokenQuotaDay, usergroup.PolicyKeyInstanceQuota, usergroup.PolicyKeyTokenQuotaRules:
			entry.SubLabel = i18n.T(ctx, i18n.MsgGroupTreeSubLabelUserQuota)
		case usergroup.PolicyKeyGlobalTokenQuotaDay, usergroup.PolicyKeyGlobalTokenQuotaRules:
			entry.SubLabel = i18n.T(ctx, i18n.MsgGroupTreeSubLabelModelQuota)
		default:
			entry.SubLabel = i18n.T(ctx, i18n.MsgGroupTreeSubLabelFeatureToggle)
		}
		// meta
		def, _ := usergroup.GetPolicyDef(item.Key)
		switch def.ValueType {
		case usergroup.PolicyValueBool:
			entry.Meta = map[string]interface{}{"enabled": item.Value}
		case usergroup.PolicyValueInt:
			// -1 表示无限制
			if v, ok := item.Value.(int); ok && v == -1 {
				entry.Meta = map[string]interface{}{"value": i18n.T(ctx, i18n.MsgGroupTreeMetaUnlimited)}
			} else {
				entry.Meta = map[string]interface{}{"value": item.Value}
			}
		case usergroup.PolicyValueString:
			entry.Meta = map[string]interface{}{"value": item.Value}
		}
		entries = append(entries, entry)
	}
	return entries
}

// overviewResource 用于可见性解析的资源统一结构
type overviewResource struct {
	ID             uint
	Name           string
	VisibilityType string
}

// resolveVisibilityItems 根据资源列表和绑定表，解析每个资源对指定组的可见性来源。
// bindingTable: 绑定关系表名, fkColumn: 绑定表中资源外键列名。
func resolveVisibilityItems(ctx context.Context, resources []overviewResource, bindingTable, fkColumn string, groupID uint, ancestors []uint) []usergroup.OverviewItem {
	items := make([]usergroup.OverviewItem, 0, len(resources))
	for _, r := range resources {
		if r.Name == "" {
			continue
		}
		if r.VisibilityType == usergroup.VisibilityAll || r.VisibilityType == "" {
			items = append(items, usergroup.OverviewItem{
				ResourceID:   r.ID,
				ResourceName: r.Name,
				Source:       usergroup.Source{Type: usergroup.SourceAllUsers},
			})
		} else if r.VisibilityType == usergroup.VisibilityGroup && len(ancestors) > 0 {
			var bindings []struct {
				GroupID uint `gorm:"column:group_id"`
			}
			model.DB(ctx).Table(bindingTable).Select("group_id").
				Where(fkColumn+" = ? AND group_id IN ?", r.ID, ancestors).
				Find(&bindings)
			if len(bindings) > 0 {
				priorityMap := make(map[uint]int)
				for i, gid := range ancestors {
					priorityMap[gid] = i
				}
				bestGID := bindings[0].GroupID
				bestPri := priorityMap[bestGID]
				for _, b := range bindings[1:] {
					if p, ok := priorityMap[b.GroupID]; ok && p < bestPri {
						bestGID = b.GroupID
						bestPri = p
					}
				}
				srcType := usergroup.SourceInherited
				if bestGID == groupID {
					srcType = usergroup.SourceLocal
				}
				var grp model.UserGroup
				model.DB(ctx).Select("name, full_path").First(&grp, bestGID)
				items = append(items, usergroup.OverviewItem{
					ResourceID:   r.ID,
					ResourceName: r.Name,
					Source: usergroup.Source{
						Type:     srcType,
						GroupID:  bestGID,
						FullPath: grp.FullPath,
					},
				})
			}
		}
	}
	return items
}

// parseSubnetsForOverview 解析子网配置为展示用结构
func parseSubnetsForOverview(cfg *model.SiteConfig) []map[string]interface{} {
	subnetMap := cfg.GetSubnetMap()
	if len(subnetMap) == 0 {
		return nil
	}
	result := make([]map[string]interface{}, 0)
	for zone, subnets := range subnetMap {
		for _, subnetId := range subnets {
			result = append(result, map[string]interface{}{
				"zone":      zone,
				"subnet_id": subnetId,
			})
		}
	}
	return result
}

// getEnabledImageTypes 返回所有有已启用镜像的 agent_type 集合
func getEnabledImageTypes(ctx context.Context) map[string]bool {
	var types []string
	model.DB(ctx).Model(&model.AIImage{}).Where("enabled = ?", true).Distinct().Pluck("agent_type", &types)
	m := make(map[string]bool, len(types))
	for _, t := range types {
		m[t] = true
	}
	return m
}

// appendVisibilityAllResources 补充 visibility=all 的全局可用资源到 overview。
// 已在 overview 中出现的资源（按 ID 或名称去重）不重复添加。
func appendVisibilityAllResources(ctx context.Context, overview *usergroup.AdditiveOverview, resourceType string) *usergroup.AdditiveOverview {
	if overview == nil {
		overview = &usergroup.AdditiveOverview{Items: []usergroup.OverviewItem{}}
	}
	// 构建已有资源集合用于去重
	existingIDs := make(map[uint]bool)
	existingNames := make(map[string]bool)
	for _, item := range overview.Items {
		if item.ResourceID > 0 {
			existingIDs[item.ResourceID] = true
		}
		if item.ResourceName != "" {
			existingNames[item.ResourceName] = true
		}
	}
	type visRow struct {
		ID   uint   `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	switch resourceType {
	case usergroup.ConfigTypeChannel:
		var rows []struct {
			ID        uint   `gorm:"column:id"`
			ChannelID string `gorm:"column:channel_id"`
			Name      string `gorm:"column:name"`
		}
		model.DB(ctx).Model(&model.AIChannel{}).Select("id, channel_id, name").
			Where("(visibility_type = 'all' OR visibility_type = '') AND enabled = ?", true).Find(&rows)
		for _, r := range rows {
			if existingIDs[r.ID] {
				continue
			}
			if !channelInCurrentSiteScope(ctx, r.ChannelID) {
				continue
			}
			overview.Items = append(overview.Items, usergroup.OverviewItem{
				ResourceID: r.ID, ResourceName: r.Name,
				Source: usergroup.Source{Type: usergroup.SourceAllUsers},
			})
		}
	case usergroup.ConfigTypePluginBundle:
		var rows []visRow
		model.DB(ctx).Model(&model.PluginBundle{}).Select("id, name").
			Where("(visibility_type = 'all' OR visibility_type = '') AND enabled = ?", true).Find(&rows)
		for _, r := range rows {
			if existingIDs[r.ID] {
				continue
			}
			overview.Items = append(overview.Items, usergroup.OverviewItem{
				ResourceID: r.ID, ResourceName: r.Name,
				Source: usergroup.Source{Type: usergroup.SourceAllUsers},
			})
		}
	case usergroup.ConfigTypeMCP:
		var rows []visRow
		model.DB(ctx).Model(&model.McpServer{}).Select("id, name").
			Where("visibility_type = 'all' OR visibility_type = ''").Find(&rows)
		for _, r := range rows {
			if existingIDs[r.ID] {
				continue
			}
			overview.Items = append(overview.Items, usergroup.OverviewItem{
				ResourceID: r.ID, ResourceName: r.Name,
				Source: usergroup.Source{Type: usergroup.SourceAllUsers},
			})
		}
	case usergroup.ConfigTypeImageType:
		// 查出所有已启用的 agent_type
		var rows []struct {
			AgentType string `gorm:"column:agent_type"`
		}
		model.DB(ctx).Model(&model.AIImage{}).Select("DISTINCT agent_type").
			Where("enabled = ?", true).Find(&rows)
		// 查出所有被限制的 agent_type（在 group_config_bindings 中有绑定行的）
		restricted, _ := model.GetRestrictedImageTypes(ctx)
		restrictedSet := make(map[string]bool, len(restricted))
		for _, r := range restricted {
			restrictedSet[r] = true
		}
		for _, r := range rows {
			if existingNames[r.AgentType] {
				continue
			}
			// 已被限制的 agent_type 不标记为全局可见（它们只对绑定的组可见）
			if restrictedSet[r.AgentType] {
				continue
			}
			overview.Items = append(overview.Items, usergroup.OverviewItem{
				ResourceName: r.AgentType,
				Source:       usergroup.Source{Type: usergroup.SourceAllUsers},
			})
		}
	}
	overview.Total = len(overview.Items)
	return overview
}
