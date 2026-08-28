package controller

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

func parseProjectID(r *http.Request) (uint, bool) {
	value, err := strconv.ParseUint(strings.TrimSpace(r.URL.Query().Get("project_id")), 10, 64)
	return uint(value), err == nil && value > 0
}

// HandleAdminProjectConfigOverview 返回项目可见的 Agent 工具概览。
//
// 可见集合与项目资产候选一致：项目直接应用范围内的资源，加上全员可见资源。
// 当前直接资产是独立概念，由 /admin/assets/detail 返回，不能替代本接口。
func HandleAdminProjectConfigOverview(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	projectIDs, err := parseUintCSV(r.URL.Query().Get("project_ids"))
	if err != nil || len(projectIDs) == 0 || len(projectIDs) > 100 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "project_ids"))
		return
	}
	if err := model.ValidateProjectIDs(model.DB(r.Context()), projectIDs); err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	results := make([]map[string]any, 0, len(projectIDs))
	for _, projectID := range projectIDs {
		categories, err := projectConfigCategories(model.DB(r.Context()), projectID)
		if err != nil {
			writeProjectDBError(w, r, err)
			return
		}
		results = append(results, map[string]any{"project_id": projectID, "categories": categories})
	}
	jsonOK(w, map[string]any{"ok": true, "results": results})
}

func projectConfigCategories(db *gorm.DB, projectID uint) ([]usergroup.ConfigCategoryResult, error) {
	ctx := db.Statement.Context
	bindings, err := projectVisibilityBindings(db, projectID)
	if err != nil {
		return nil, err
	}
	skills, err := projectVisibleSkills(db, bindings[model.ProjectConfigTypeSkill])
	if err != nil {
		return nil, err
	}
	rules, err := projectVisibleRules(db, bindings[model.ProjectConfigTypeRule])
	if err != nil {
		return nil, err
	}
	entries := make([]usergroup.ConfigEntry, 0, len(skills)+len(rules))
	entries = append(entries, projectSkillOverviewEntries(ctx, skills)...)
	entries = append(entries, projectRuleOverviewEntries(ctx, rules)...)
	return []usergroup.ConfigCategoryResult{{
		Key: usergroup.CategoryKeyAgentTool, Label: i18n.T(ctx, i18n.MsgCategoryAgentTool),
		Description: i18n.T(ctx, i18n.MsgProjectConfigAgentToolDescription), Icon: "Wrench", Entries: entries,
	}}, nil
}

func projectVisibilityBindings(db *gorm.DB, projectID uint) (map[string][]string, error) {
	result := map[string][]string{model.ProjectConfigTypeSkill: {}, model.ProjectConfigTypeRule: {}}
	var rows []model.ProjectConfigBinding
	if err := db.Where("project_id = ? AND config_type IN ?", projectID, model.ProjectVisibilityConfigTypes).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ConfigType] = append(result[row.ConfigType], row.ConfigKey)
	}
	return result, nil
}

func projectVisibleSkills(db *gorm.DB, slugs []string) ([]model.Skill, error) {
	query := db.Where("id IN (?)", model.LatestVersionSkillIDs(db.Statement.Context))
	if len(slugs) > 0 {
		query = query.Where("visibility_type = ? OR slug IN ?", model.VisibilityAll, slugs)
	} else {
		query = query.Where("visibility_type = ?", model.VisibilityAll)
	}
	var skills []model.Skill
	return skills, query.Order("name ASC").Find(&skills).Error
}

func projectVisibleRules(db *gorm.DB, slugs []string) ([]model.EnterpriseRule, error) {
	query := db.Where("id IN (?)", model.LatestVersionRuleIDs(db.Statement.Context))
	if len(slugs) > 0 {
		query = query.Where("visibility_type = ? OR slug IN ?", model.VisibilityAll, slugs)
	} else {
		query = query.Where("visibility_type = ?", model.VisibilityAll)
	}
	var rules []model.EnterpriseRule
	return rules, query.Order("name ASC").Find(&rules).Error
}

func projectSkillOverviewEntries(ctx context.Context, skills []model.Skill) []usergroup.ConfigEntry {
	entries := make([]usergroup.ConfigEntry, 0, len(skills))
	for _, skill := range skills {
		entries = append(entries, usergroup.ConfigEntry{
			ID: strconv.FormatUint(uint64(skill.ID), 10), Label: skill.Name,
			SubLabel: i18n.T(ctx, i18n.MsgGroupTreeSubLabelEnterpriseSkill), Source: projectOverviewSource(skill.VisibilityType),
		})
	}
	return entries
}

func projectRuleOverviewEntries(ctx context.Context, rules []model.EnterpriseRule) []usergroup.ConfigEntry {
	entries := make([]usergroup.ConfigEntry, 0, len(rules))
	for _, rule := range rules {
		entries = append(entries, usergroup.ConfigEntry{
			ID: strconv.FormatUint(uint64(rule.ID), 10), Label: rule.Name,
			SubLabel: i18n.T(ctx, i18n.MsgProjectConfigEnterpriseRule), Source: projectOverviewSource(rule.VisibilityType),
		})
	}
	return entries
}

func projectOverviewSource(visibilityType string) usergroup.Source {
	if visibilityType == model.VisibilityAll {
		return usergroup.Source{Type: usergroup.SourceAllUsers}
	}
	return usergroup.Source{Type: usergroup.SourceLocal}
}

func HandleAdminProjectInstances(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	projectID, ok := parseProjectID(r)
	if !ok {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "project_id"))
		return
	}
	writeLocalAgentTargetInstances(w, r, assetTargetProject, projectID)
}

// HandleAdminUserGroupInstances 返回用户级绑定到指定分组的本地 Agent。
// 分组只匹配 scope=user，项目 Workspace 的绑定不会混入该接口。
func HandleAdminUserGroupInstances(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	groupID, err := strconv.ParseUint(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
	if err != nil || groupID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "id"))
		return
	}
	writeLocalAgentTargetInstances(w, r, assetTargetGroup, uint(groupID))
}

func writeLocalAgentTargetInstances(w http.ResponseWriter, r *http.Request, targetType string, targetID uint) {
	scope, bindingField := localAgentTargetScope(targetType)
	targetIDs, err := localAgentTargetIDs(r.Context(), targetType, targetID)
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	items, err := model.ListLocalAgentInstancesByScopeTargets(r.Context(), scope, targetIDs)
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	targets, err := localAgentTargetDisplays(r.Context(), targetType, targetIDs)
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	usernames, err := localAgentUsernames(r.Context(), items)
	if err != nil {
		writeProjectDBError(w, r, err)
		return
	}
	page, pageSize := parsePagination(r)
	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	result := make([]map[string]any, 0, end-start)
	for _, item := range items[start:end] {
		result = append(result, localAgentTargetInstanceResponse(r, item, bindingField, targetType, targets, usernames[item.Instance.UserID]))
	}
	jsonOK(w, map[string]any{"instances": result, "total": total, "page": page, "page_size": pageSize})
}

// localAgentInstanceView 在保持原始 Instance JSON 字段兼容的前提下，补充资产页展示字段。
type localAgentInstanceView struct {
	model.Instance
	Status   string `json:"status"`
	Username string `json:"username"`
}

// localAgentScopeBindingView 保留原始 scope 绑定字段，并附加目标名称供资产页展示。
type localAgentScopeBindingView struct {
	model.LocalAgentScopeBinding
	GroupName     string `json:"group_name,omitempty"`
	GroupFullPath string `json:"group_full_path,omitempty"`
	ProjectName   string `json:"project_name,omitempty"`
}

type localAgentTargetDisplay struct {
	name     string
	fullPath string
}

func localAgentTargetInstanceResponse(r *http.Request, item model.LocalAgentTargetInstance, bindingField, targetType string, targets map[uint]localAgentTargetDisplay, username string) map[string]any {
	bindings := make([]localAgentScopeBindingView, 0, len(item.ScopeBindings))
	for _, binding := range item.ScopeBindings {
		view := localAgentScopeBindingView{LocalAgentScopeBinding: binding}
		targetID := binding.ProjectID
		if targetType == assetTargetGroup {
			targetID = binding.GroupID
		}
		target, exists := targets[targetID]
		if !exists {
			Logger(r.Context()).Warn("[AdminProjectConfig] target display metadata missing",
				"target_type", targetType, "target_id", targetID, "instance_id", item.Instance.ID)
			bindings = append(bindings, view)
			continue
		}
		if targetType == assetTargetGroup {
			view.GroupName = target.name
			view.GroupFullPath = target.fullPath
		} else {
			view.ProjectName = target.name
		}
		bindings = append(bindings, view)
	}
	status := ResolveInstanceStatus(r.Context(), &item.Instance, nil, nil).Status
	return map[string]any{bindingField: bindings, "instance": localAgentInstanceView{Instance: item.Instance, Status: status, Username: username}}
}

func localAgentTargetIDs(ctx context.Context, targetType string, targetID uint) ([]uint, error) {
	if targetType != assetTargetGroup {
		return []uint{targetID}, nil
	}
	ids, err := model.ClosureDescendants(ctx, targetID, true)
	if err != nil {
		return nil, err
	}
	return uniqueUintIDs(append(ids, targetID)), nil
}

func localAgentTargetDisplays(ctx context.Context, targetType string, targetIDs []uint) (map[uint]localAgentTargetDisplay, error) {
	result := make(map[uint]localAgentTargetDisplay, len(targetIDs))
	switch targetType {
	case assetTargetGroup:
		var groups []model.UserGroup
		if err := model.DB(ctx).Where("id IN ?", targetIDs).Find(&groups).Error; err != nil {
			return nil, err
		}
		for _, group := range groups {
			result[group.ID] = localAgentTargetDisplay{name: group.Name, fullPath: group.FullPath}
		}
	case assetTargetProject:
		var projects []model.Project
		if err := model.DB(ctx).Where("id IN ?", targetIDs).Find(&projects).Error; err != nil {
			return nil, err
		}
		for _, project := range projects {
			result[project.ID] = localAgentTargetDisplay{name: project.Name}
		}
	}
	return result, nil
}

func localAgentUsernames(ctx context.Context, items []model.LocalAgentTargetInstance) (map[uint]string, error) {
	userIDs := make([]uint, 0, len(items))
	for _, item := range items {
		userIDs = append(userIDs, item.Instance.UserID)
	}
	result := make(map[uint]string, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	var users []model.User
	if err := model.DB(ctx).Unscoped().Where("id IN ?", uniqueUintIDs(userIDs)).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, user := range users {
		result[user.ID] = user.Username
	}
	return result, nil
}

func localAgentTargetScope(targetType string) (string, string) {
	if targetType == assetTargetGroup {
		return model.LocalAgentScopeUser, "bound_user_levels"
	}
	return model.LocalAgentScopeWorkspace, "bound_workspaces"
}
