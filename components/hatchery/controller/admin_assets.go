package controller

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

const (
	assetTargetProject = "project"
	assetTargetGroup   = "group"
)

type assetTarget struct {
	typeName string
	id       uint
	name     string
	fullPath string
	syncMode string // 同步模式：项目固定 continuous，分组取 user_groups.sync_mode
}

type assetItem struct {
	AssetType    string               `json:"asset_type"`
	Slug         string               `json:"slug"`
	Name         string               `json:"name"`
	Version      string               `json:"version"`
	Selected     bool                 `json:"selected,omitempty"`
	Source       usergroup.SourceType `json:"source,omitempty"`
	SourceTarget *assetSourceTarget   `json:"source_target,omitempty"`
}

type assetCandidateItem struct {
	AssetType string           `json:"asset_type"`
	Slug      string           `json:"slug"`
	Name      string           `json:"name"`
	Version   string           `json:"version"`
	Selected  bool             `json:"selected"`
	Source    usergroup.Source `json:"source"`
}

type assetSourceTarget struct {
	Type     string `json:"type"`
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	FullPath string `json:"full_path,omitempty"`
}

func assetKeys(items []assetItem) (skills, rules []string) {
	for _, item := range items {
		switch item.AssetType {
		case model.AssetTypeSkill:
			skills = append(skills, item.Slug)
		case model.AssetTypeRule:
			rules = append(rules, item.Slug)
		}
	}
	return skills, rules
}

func projectCandidateAssets(db *gorm.DB, projectID uint) ([]assetItem, error) {
	var bindings []model.ProjectConfigBinding
	if err := db.Where("project_id = ? AND config_type IN ?", projectID, model.ProjectVisibilityConfigTypes).Find(&bindings).Error; err != nil {
		return nil, err
	}
	skillSlugs, ruleSlugs := make([]string, 0), make([]string, 0)
	for _, binding := range bindings {
		switch binding.ConfigType {
		case model.ProjectConfigTypeSkill:
			skillSlugs = append(skillSlugs, binding.ConfigKey)
		case model.ProjectConfigTypeRule:
			ruleSlugs = append(ruleSlugs, binding.ConfigKey)
		}
	}
	return queryVisibleAssets(db, skillSlugs, ruleSlugs)
}

func queryVisibleAssets(db *gorm.DB, skillSlugs, ruleSlugs []string) ([]assetItem, error) {
	var skills []model.Skill
	skillQuery := db.Where("id IN (?)", model.LatestVersionSkillIDs(db.Statement.Context))
	if len(skillSlugs) > 0 {
		skillQuery = skillQuery.Where("visibility_type = ? OR slug IN ?", model.VisibilityAll, skillSlugs)
	} else {
		skillQuery = skillQuery.Where("visibility_type = ?", model.VisibilityAll)
	}
	if err := skillQuery.Order("name ASC").Find(&skills).Error; err != nil {
		return nil, err
	}
	var rules []model.EnterpriseRule
	ruleQuery := db.Where("id IN (?)", model.LatestVersionRuleIDs(db.Statement.Context))
	if len(ruleSlugs) > 0 {
		ruleQuery = ruleQuery.Where("visibility_type = ? OR slug IN ?", model.VisibilityAll, ruleSlugs)
	} else {
		ruleQuery = ruleQuery.Where("visibility_type = ?", model.VisibilityAll)
	}
	if err := ruleQuery.Order("name ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	items := make([]assetItem, 0, len(skills)+len(rules))
	for _, skill := range skills {
		items = append(items, assetItem{AssetType: model.AssetTypeSkill, Slug: skill.Slug, Name: skill.Name, Version: skill.Version, Source: projectCandidateSource(skill.VisibilityType)})
	}
	for _, rule := range rules {
		items = append(items, assetItem{AssetType: model.AssetTypeRule, Slug: rule.Slug, Name: rule.Name, Version: rule.Version, Source: projectCandidateSource(rule.VisibilityType)})
	}
	return items, nil
}

func projectCandidateSource(visibilityType string) usergroup.SourceType {
	if visibilityType == model.VisibilityAll {
		return usergroup.SourceAllUsers
	}
	return usergroup.SourceLocal
}

func parseAssetTarget(r *http.Request) (assetTarget, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(r.URL.Query().Get("target_id")), 10, 64)
	if err != nil || id == 0 {
		return assetTarget{}, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "target_id")
	}
	return assetTarget{typeName: strings.TrimSpace(r.URL.Query().Get("target_type")), id: uint(id)}, nil
}

func loadAssetTarget(db *gorm.DB, target assetTarget) (assetTarget, error) {
	switch target.typeName {
	case assetTargetProject:
		project, err := requireProject(db, target.id)
		if err != nil {
			return target, err
		}
		target.name = project.Name
		target.syncMode = model.SyncModeContinuous
	case assetTargetGroup:
		var group model.UserGroup
		if err := db.Where("id = ?", target.id).First(&group).Error; err != nil {
			return target, err
		}
		target.name = group.Name
		target.fullPath = group.FullPath
		target.syncMode = group.SyncMode
	default:
		return target, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "target_type")
	}
	return target, nil
}

func groupCandidateAssets(db *gorm.DB, groupID uint) ([]assetItem, error) {
	ancestors, err := model.ClosureAncestors(db.Statement.Context, groupID, true)
	if err != nil {
		return nil, err
	}
	var skills []model.Skill
	if err := db.Where("id IN (?)", model.LatestVersionSkillIDs(db.Statement.Context)).Find(&skills).Error; err != nil {
		return nil, err
	}
	var rules []model.EnterpriseRule
	if err := db.Where("id IN (?)", model.LatestVersionRuleIDs(db.Statement.Context)).Find(&rules).Error; err != nil {
		return nil, err
	}
	var skillVisibility []model.SkillVisibilityGroup
	if err := db.Where("group_id IN ?", ancestors).Find(&skillVisibility).Error; err != nil {
		return nil, err
	}
	var ruleVisibility []model.RuleVisibilityGroup
	if err := db.Where("group_id IN ?", ancestors).Find(&ruleVisibility).Error; err != nil {
		return nil, err
	}
	ancestorRank := make(map[uint]int, len(ancestors))
	for rank, ancestorID := range ancestors {
		ancestorRank[ancestorID] = rank
	}
	skillSourceGroup, ruleSourceGroup := make(map[uint]uint), make(map[uint]uint)
	for _, row := range skillVisibility {
		setNearestAssetGroup(skillSourceGroup, row.SkillID, row.GroupID, ancestorRank)
	}
	for _, row := range ruleVisibility {
		setNearestAssetGroup(ruleSourceGroup, row.RuleID, row.GroupID, ancestorRank)
	}
	var groups []model.UserGroup
	if len(ancestors) > 0 {
		if err := db.Where("id IN ?", ancestors).Find(&groups).Error; err != nil {
			return nil, err
		}
	}
	groupByID := make(map[uint]model.UserGroup, len(groups))
	for _, group := range groups {
		groupByID[group.ID] = group
	}
	items := make([]assetItem, 0, len(skills)+len(rules))
	for _, skill := range skills {
		if source, sourceTarget, visible := groupCandidateSource(skill.VisibilityType, skillSourceGroup[skill.ID], groupID, groupByID); visible {
			items = append(items, assetItem{AssetType: model.AssetTypeSkill, Slug: skill.Slug, Name: skill.Name, Version: skill.Version, Source: source, SourceTarget: sourceTarget})
		}
	}
	for _, rule := range rules {
		if source, sourceTarget, visible := groupCandidateSource(rule.VisibilityType, ruleSourceGroup[rule.ID], groupID, groupByID); visible {
			items = append(items, assetItem{AssetType: model.AssetTypeRule, Slug: rule.Slug, Name: rule.Name, Version: rule.Version, Source: source, SourceTarget: sourceTarget})
		}
	}
	return items, nil
}

func setNearestAssetGroup(byAsset map[uint]uint, assetID, groupID uint, ancestorRank map[uint]int) {
	groupRank, inScope := ancestorRank[groupID]
	if !inScope {
		return
	}
	currentGroupID, exists := byAsset[assetID]
	if !exists || groupRank < ancestorRank[currentGroupID] {
		byAsset[assetID] = groupID
	}
}

func groupCandidateSource(visibilityType string, sourceGroupID, targetGroupID uint, groups map[uint]model.UserGroup) (usergroup.SourceType, *assetSourceTarget, bool) {
	if visibilityType == model.VisibilityAll {
		return usergroup.SourceAllUsers, nil, true
	}
	if sourceGroupID == 0 {
		return "", nil, false
	}
	if sourceGroupID == targetGroupID {
		return usergroup.SourceLocal, nil, true
	}
	group, ok := groups[sourceGroupID]
	if !ok {
		return usergroup.SourceInherited, nil, true
	}
	return usergroup.SourceInherited, &assetSourceTarget{Type: assetTargetGroup, ID: group.ID, Name: group.Name, FullPath: group.FullPath}, true
}

func toAssetCandidateItem(item assetItem) assetCandidateItem {
	source := usergroup.Source{Type: item.Source}
	if item.SourceTarget != nil {
		source.GroupID = item.SourceTarget.ID
		source.FullPath = item.SourceTarget.FullPath
	}
	return assetCandidateItem{
		AssetType: item.AssetType,
		Slug:      item.Slug,
		Name:      item.Name,
		Version:   item.Version,
		Selected:  item.Selected,
		Source:    source,
	}
}

func assetCandidates(db *gorm.DB, target assetTarget) ([]assetItem, error) {
	switch target.typeName {
	case assetTargetProject:
		return projectCandidateAssets(db, target.id)
	case assetTargetGroup:
		return groupCandidateAssets(db, target.id)
	default:
		return nil, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "target_type")
	}
}

func targetAssetBindings(db *gorm.DB, target assetTarget, inherited bool) ([]assetItem, error) {
	switch target.typeName {
	case assetTargetProject:
		if inherited {
			return []assetItem{}, nil
		}
		var projectBindings []model.ProjectConfigBinding
		if err := db.Where("project_id = ? AND config_type IN ?", target.id, model.ProjectAssetConfigTypes).
			Order("config_type ASC, config_key ASC").Find(&projectBindings).Error; err != nil {
			return nil, err
		}
		items := make([]assetItem, 0, len(projectBindings))
		for _, binding := range projectBindings {
			assetType, _ := assetTypeForConfig(binding.ConfigType)
			items = append(items, assetItem{AssetType: assetType, Slug: binding.ConfigKey})
		}
		enriched, err := enrichAssetItems(db, items)
		if err != nil {
			return nil, err
		}
		return markAssetSource(enriched, usergroup.SourceLocal, nil), nil
	case assetTargetGroup:
		return groupAssetBindings(db, target, inherited)
	default:
		return nil, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "target_type")
	}
}

func groupAssetBindings(db *gorm.DB, target assetTarget, inherited bool) ([]assetItem, error) {
	groupIDs := []uint{target.id}
	if inherited {
		ids, err := model.ClosureAncestors(db.Statement.Context, target.id, false)
		if err != nil {
			return nil, err
		}
		groupIDs = ids
	}
	var bindings []model.GroupConfigBinding
	if err := db.Where("group_id IN ? AND config_type IN ?", groupIDs, model.GroupAssetConfigTypes).
		Order("config_type ASC, config_key ASC").Find(&bindings).Error; err != nil {
		return nil, err
	}
	groupRank := make(map[uint]int, len(groupIDs))
	for index, groupID := range groupIDs {
		groupRank[groupID] = index
	}
	groupByAsset := make(map[string]uint, len(bindings))
	for _, binding := range bindings {
		assetType, ok := assetTypeForConfig(binding.ConfigType)
		if !ok {
			continue
		}
		key := assetType + ":" + binding.ConfigKey
		if existingGroupID, exists := groupByAsset[key]; exists && groupRank[existingGroupID] <= groupRank[binding.GroupID] {
			continue
		}
		groupByAsset[key] = binding.GroupID
	}
	items := make([]assetItem, 0, len(groupByAsset))
	for _, assetType := range []string{model.AssetTypeSkill, model.AssetTypeRule} {
		for _, binding := range bindings {
			bindingAssetType, ok := assetTypeForConfig(binding.ConfigType)
			if !ok || bindingAssetType != assetType {
				continue
			}
			key := assetType + ":" + binding.ConfigKey
			if groupByAsset[key] == binding.GroupID {
				items = append(items, assetItem{AssetType: assetType, Slug: binding.ConfigKey})
			}
		}
	}
	items, err := enrichAssetItems(db, items)
	if err != nil {
		return nil, err
	}
	if !inherited {
		return markAssetSource(items, usergroup.SourceLocal, nil), nil
	}
	return markInheritedGroupAssetSource(db, items, groupByAsset)
}

func assetTypeForConfig(configType string) (string, bool) {
	switch configType {
	case model.AssetBindingTypeSkill:
		return model.AssetTypeSkill, true
	case model.AssetBindingTypeRule:
		return model.AssetTypeRule, true
	}
	return "", false
}

func markAssetSource(items []assetItem, source usergroup.SourceType, sourceTarget *assetSourceTarget) []assetItem {
	for i := range items {
		items[i].Source = source
		items[i].SourceTarget = sourceTarget
	}
	return items
}

func markInheritedGroupAssetSource(db *gorm.DB, items []assetItem, groupByAsset map[string]uint) ([]assetItem, error) {
	groupIDs := make([]uint, 0, len(groupByAsset))
	for _, groupID := range groupByAsset {
		groupIDs = append(groupIDs, groupID)
	}
	var groups []model.UserGroup
	if len(groupIDs) > 0 {
		if err := db.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
			return nil, err
		}
	}
	groupByID := make(map[uint]model.UserGroup, len(groups))
	for _, group := range groups {
		groupByID[group.ID] = group
	}
	for i := range items {
		items[i].Source = usergroup.SourceInherited
		if group, ok := groupByID[groupByAsset[items[i].AssetType+":"+items[i].Slug]]; ok {
			items[i].SourceTarget = &assetSourceTarget{Type: assetTargetGroup, ID: group.ID, Name: group.Name, FullPath: group.FullPath}
		}
	}
	return items, nil
}

// enrichAssetItems 仅按已绑定 slug 读取资产元数据，不计算候选资格。
// 因此 detail 永远只包含目标直接或继承拥有的资产，不会混入全员可见资产。
func enrichAssetItems(db *gorm.DB, items []assetItem) ([]assetItem, error) {
	skillSlugs, ruleSlugs := assetKeys(items)
	byKey, err := loadAssetMetadata(db, skillSlugs, ruleSlugs)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if item, ok := byKey[items[i].AssetType+":"+items[i].Slug]; ok {
			items[i] = item
		}
	}
	return items, nil
}

func loadAssetMetadata(db *gorm.DB, skillSlugs, ruleSlugs []string) (map[string]assetItem, error) {
	result := make(map[string]assetItem, len(skillSlugs)+len(ruleSlugs))
	if len(skillSlugs) > 0 {
		var skills []model.Skill
		if err := db.Where("slug IN ? AND id IN (?)", skillSlugs, model.LatestVersionSkillIDs(db.Statement.Context)).Find(&skills).Error; err != nil {
			return nil, err
		}
		for _, skill := range skills {
			result[model.AssetTypeSkill+":"+skill.Slug] = assetItem{AssetType: model.AssetTypeSkill, Slug: skill.Slug, Name: skill.Name, Version: skill.Version}
		}
	}
	if len(ruleSlugs) > 0 {
		var rules []model.EnterpriseRule
		if err := db.Where("slug IN ? AND id IN (?)", ruleSlugs, model.LatestVersionRuleIDs(db.Statement.Context)).Find(&rules).Error; err != nil {
			return nil, err
		}
		for _, rule := range rules {
			result[model.AssetTypeRule+":"+rule.Slug] = assetItem{AssetType: model.AssetTypeRule, Slug: rule.Slug, Name: rule.Name, Version: rule.Version}
		}
	}
	return result, nil
}

func HandleAdminAssetCandidates(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	target, err := parseAssetTarget(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.(*hcommon.RichError))
		return
	}
	target, err = loadAssetTarget(model.DB(r.Context()), target)
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	items, err := assetCandidates(model.DB(r.Context()), target)
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	local, err := targetAssetBindings(model.DB(r.Context()), target, false)
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	selectedSet := make(map[string]bool, len(local))
	for _, item := range local {
		selectedSet[item.AssetType+":"+item.Slug] = true
	}
	assetType, query, selected := strings.TrimSpace(r.URL.Query().Get("asset_type")), strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q"))), strings.TrimSpace(r.URL.Query().Get("selected"))
	filtered := make([]assetCandidateItem, 0, len(items))
	for _, item := range items {
		item.Selected = selectedSet[item.AssetType+":"+item.Slug]
		matchesSelected := selected == "" || selected == "all" || (selected == "selected" && item.Selected) || (selected == "unselected" && !item.Selected)
		if (assetType == "" || item.AssetType == assetType) && matchesSelected && (query == "" || strings.Contains(strings.ToLower(item.Name+" "+item.Slug), query)) {
			filtered = append(filtered, toAssetCandidateItem(item))
		}
	}
	page, pageSize := parsePagination(r)
	start, end := (page-1)*pageSize, page*pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}
	jsonOK(w, map[string]any{"ok": true, "assets": filtered[start:end], "total": len(filtered), "page": page, "page_size": pageSize})
}

func HandleAdminAssetDetail(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	target, err := parseAssetTarget(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.(*hcommon.RichError))
		return
	}
	db := model.DB(r.Context())
	target, err = loadAssetTarget(db, target)
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	local, err := targetAssetBindings(db, target, false)
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	inherited := []assetItem{}
	switch target.typeName {
	case assetTargetProject:
	case assetTargetGroup:
		inherited, err = targetAssetBindings(db, target, true)
		if err != nil {
			writeProjectDBError(w, r, err)
			return
		}
	default:
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "target_type"))
		return
	}
	jsonOK(w, map[string]any{
		"ok":              true,
		"target":          assetTargetResponse(target),
		"current_version": currentVersionOf(db, target),
		"assets":          orderedAssetItems(local, inherited),
	})
}

// currentVersionOf 返回目标当前的资产版本号（asset_versions 表 MAX(version)），
// 无版本记录时为 0（前端据此展示「暂无版本」，而非 v0）。
// 版本记录从 v1 起，v0 不会出现在版本历史里。
func currentVersionOf(db *gorm.DB, target assetTarget) int {
	var maxVer int
	if err := db.Model(&model.AssetVersionRecord{}).
		Where("target_type = ? AND target_id = ?", target.typeName, target.id).
		Select("COALESCE(MAX(version), 0)").Scan(&maxVer).Error; err != nil {
		return 0
	}
	return maxVer
}

func assetTargetResponse(target assetTarget) map[string]any {
	response := map[string]any{"type": target.typeName, "id": target.id, "name": target.name, "sync_mode": target.syncMode}
	if target.typeName == assetTargetGroup {
		response["full_path"] = target.fullPath
	}
	return response
}

// orderedAssetItems 保持本地绑定在前、继承绑定在后；各来源内沿用绑定查询顺序。
func orderedAssetItems(local, inherited []assetItem) []assetItem {
	items := make([]assetItem, 0, len(local)+len(inherited))
	items = append(items, local...)
	items = append(items, inherited...)
	return items
}

func HandleAdminAssetSave(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	var req struct {
		TargetType string      `json:"target_type"`
		TargetID   uint        `json:"target_id"`
		SyncMode   string      `json:"sync_mode"` // 必填：continuous / initial_only
		Assets     []assetItem `json:"assets"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.TargetID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	// sync_mode 校验：必填且取值合法
	if req.SyncMode != model.SyncModeContinuous && req.SyncMode != model.SyncModeInitialOnly {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "sync_mode"))
		return
	}
	if req.TargetType == model.TargetTypeProject && req.SyncMode != model.SyncModeContinuous {
		// 项目只允许 continuous（文档 §10.8）
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "sync_mode"))
		return
	}
	db := model.DB(r.Context())
	target, err := loadAssetTarget(db, assetTarget{typeName: req.TargetType, id: req.TargetID})
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	candidates, err := assetCandidates(db, target)
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	allowed := make(map[string]bool)
	for _, item := range candidates {
		allowed[item.AssetType+":"+item.Slug] = true
	}
	skills, rules := assetKeys(req.Assets)
	for _, item := range req.Assets {
		if !allowed[item.AssetType+":"+item.Slug] {
			invalidParam := "assets[" + item.AssetType + ":" + item.Slug + "]"
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, invalidParam))
			return
		}
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		// 保存前读取旧绑定，用于生成版本记录时计算 diff
		oldAssets, err := loadCurrentAssetBindings(tx, target)
		if err != nil {
			return err
		}
		// 必须在 UPDATE user_groups.sync_mode 之前读取旧同步模式，否则跳变检测失效
		oldSyncMode, smErr := targetSyncMode(tx, target)
		if smErr != nil {
			return smErr
		}
		switch target.typeName {
		case assetTargetProject:
			if err := model.ReplaceProjectConfigBindings(tx, target.id, model.AssetBindingTypeSkill, skills); err != nil {
				return err
			}
			if err := model.ReplaceProjectConfigBindings(tx, target.id, model.AssetBindingTypeRule, rules); err != nil {
				return err
			}
		case assetTargetGroup:
			if err := replaceGroupAssetBindings(tx, target.id, model.AssetBindingTypeSkill, skills); err != nil {
				return err
			}
			if err := replaceGroupAssetBindings(tx, target.id, model.AssetBindingTypeRule, rules); err != nil {
				return err
			}
			// 保存分组同步模式到 user_groups 表
			if err := tx.Model(&model.UserGroup{}).Where("id = ?", target.id).
				Update("sync_mode", req.SyncMode).Error; err != nil {
				return err
			}
		default:
			return hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "target_type")
		}
		// 生成资产版本记录（按同步模式决定是否下发）
		operatorID := uint(0)
		operatorName := ""
		if u, uErr := getLoginUser(r); uErr == nil && u != nil {
			operatorID = u.ID
			operatorName = u.Username
		}
		if err := RecordAssetSave(r.Context(), tx, SaveInput{
			TargetType:   target.typeName,
			TargetID:     target.id,
			SyncMode:     req.SyncMode, // 本次请求的新同步模式
			OldSyncMode:  oldSyncMode,  // UPDATE 前查出的旧值，用于跳变检测
			Assets:       assetsToBindingItems(req.Assets),
			OldAssets:    oldAssets,
			OperatorID:   operatorID,
			OperatorName: operatorName,
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	jsonOK(w, map[string]any{"ok": true})
}

func replaceGroupAssetBindings(tx *gorm.DB, groupID uint, configType string, keys []string) error {
	if err := tx.Where("group_id = ? AND config_type = ?", groupID, configType).Delete(&model.GroupConfigBinding{}).Error; err != nil {
		return err
	}
	seen := make(map[string]bool)
	rows := make([]model.GroupConfigBinding, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" && !seen[key] {
			rows = append(rows, model.GroupConfigBinding{GroupID: groupID, ConfigType: configType, ConfigKey: key})
			seen[key] = true
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

// loadCurrentAssetBindings 读取目标当前直接绑定的资产(skill/rule)列表，用于保存时计算 diff。
func loadCurrentAssetBindings(tx *gorm.DB, target assetTarget) ([]AssetBindingItem, error) {
	items := make([]AssetBindingItem, 0)
	switch target.typeName {
	case assetTargetProject:
		var binds []model.ProjectConfigBinding
		if err := tx.Where("project_id = ? AND config_type IN ?", target.id, model.ProjectAssetConfigTypes).
			Find(&binds).Error; err != nil {
			return nil, err
		}
		for _, b := range binds {
			if at, ok := assetTypeFromConfigType(b.ConfigType); ok {
				items = append(items, AssetBindingItem{AssetType: at, Slug: b.ConfigKey})
			}
		}
	case assetTargetGroup:
		var binds []model.GroupConfigBinding
		if err := tx.Where("group_id = ? AND config_type IN ?", target.id, model.GroupAssetConfigTypes).
			Find(&binds).Error; err != nil {
			return nil, err
		}
		for _, b := range binds {
			if at, ok := assetTypeFromConfigType(b.ConfigType); ok {
				items = append(items, AssetBindingItem{AssetType: at, Slug: b.ConfigKey})
			}
		}
	}
	return items, nil
}

// assetTypeFromConfigType 把绑定表的 config_type 映射回 asset_type。
func assetTypeFromConfigType(configType string) (string, bool) {
	switch configType {
	case model.AssetBindingTypeSkill:
		return model.AssetTypeSkill, true
	case model.AssetBindingTypeRule:
		return model.AssetTypeRule, true
	default:
		return "", false
	}
}

// assetsToBindingItems 把请求里的 assetItem 列表转成 AssetBindingItem 列表。
func assetsToBindingItems(assets []assetItem) []AssetBindingItem {
	items := make([]AssetBindingItem, 0, len(assets))
	for _, a := range assets {
		if a.Slug == "" {
			continue
		}
		items = append(items, AssetBindingItem{AssetType: a.AssetType, Slug: a.Slug})
	}
	return items
}

// targetSyncMode 读取目标的同步模式：项目固定 continuous，分组读库。
func targetSyncMode(db *gorm.DB, target assetTarget) (string, error) {
	if target.typeName == assetTargetProject {
		return model.SyncModeContinuous, nil
	}
	var g model.UserGroup
	if err := db.Where("id = ?", target.id).First(&g).Error; err != nil {
		return "", err
	}
	return g.SyncMode, nil
}
