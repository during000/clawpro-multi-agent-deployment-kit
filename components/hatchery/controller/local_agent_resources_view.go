package controller

// 本地 Agent 资源反序列化与前端视图构建。

import (
	"context"

	"hatchery/model"
)

func deserializeLocalAgentResources(r *model.LocalAgentResources) *model.LocalAgentResources {
	if r != nil {
		return r
	}
	return &model.LocalAgentResources{
		UserLevel:  model.UserLevelResources{},
		Workspaces: []model.WorkspaceResource{},
	}
}

// ---- /openclaw/list?id=N 二期视图构建 -----------------------------------

// SkillStatusView 前端展示用的 skill 状态视图。
type SkillStatusView struct {
	Slug          string `json:"slug"`
	Version       string `json:"version"`
	DisplayName   string `json:"display_name,omitempty"`
	Source        string `json:"source,omitempty"`
	InstallStatus string `json:"install_status"` // distributing | distributed | failed
}

// RuleStatusView 前端展示用的规范状态视图。
type RuleStatusView struct {
	Slug          string `json:"slug"`
	Version       string `json:"version"`
	DisplayName   string `json:"display_name,omitempty"`
	Type          string `json:"type"`             // prompt / rule
	Source        string `json:"source,omitempty"` // enterprise / local
	InstallStatus string `json:"install_status"`   // distributed / distributing / failed
}

// UserLevelView 用户级资源视图（含已装 skill + rule 列表）。
type UserLevelView struct {
	GroupID     uint              `json:"group_id"`
	GroupName   string            `json:"group_name,omitempty"`
	GroupActive bool              `json:"group_active"`
	Skills      []SkillStatusView `json:"skills"`
	Rules       []RuleStatusView  `json:"rules"`
}

// WorkspaceView 项目级 workspace 视图（含已装 skill + rule 列表）。
type WorkspaceView struct {
	Path          string            `json:"path"`
	Name          string            `json:"name"`
	IDEType       string            `json:"ide_type"`
	ProjectID     uint              `json:"project_id,omitempty"`
	ProjectName   string            `json:"project_name"`
	ProjectExists bool              `json:"project_exists"`
	Skills        []SkillStatusView `json:"skills"`
	Rules         []RuleStatusView  `json:"rules"`
}

// LocalAgentResourcesView 返回给前端的完整资源视图（含已装 skill 状态）。
type LocalAgentResourcesView struct {
	UserLevel  UserLevelView   `json:"user_level"`
	Workspaces []WorkspaceView `json:"workspaces"`
}

// buildLocalAgentResourcesView 构建 local_agent_resources 视图（含已装 skill 状态）。
// 仅在精准查询（id != 0）且 source='local' 时调用。
func buildLocalAgentResourcesView(ctx context.Context, inst *model.Instance, userID uint) *LocalAgentResourcesView {
	resources := deserializeLocalAgentResources(inst.LocalAgentResources)
	if resources == nil {
		return nil
	}

	// 查 local_instance_skills 中用户级和项目级快照。
	var skills []model.LocalInstanceSkill
	if err := model.DB(ctx).Where("instance_id = ? AND (scope = ? OR scope = ?)",
		inst.ID, model.LocalSkillScopeUser, model.LocalSkillScopeWorkspace).
		Find(&skills).Error; err != nil {
		// 查询失败不影响主流程，返回不含 skills 的视图
		Logger(ctx).Warn("[LocalAgent] failed to query installed skills for resource view",
			"error", err, "instance_id", inst.ID)
		skills = nil
	}

	// 按 scope + workspace_path 分组
	userSkills := make([]SkillStatusView, 0)
	wsSkillMap := make(map[string][]SkillStatusView)
	installedSkillSlugs := make(map[string]bool, len(skills))
	for _, s := range skills {
		sv := SkillStatusView{
			Slug:          s.Slug,
			Version:       s.Version,
			DisplayName:   s.DisplayName,
			Source:        s.Source,
			InstallStatus: s.InstallStatus,
		}
		installedSkillSlugs[s.Slug] = true
		if s.Scope == model.LocalSkillScopeUser {
			userSkills = append(userSkills, sv)
		} else if s.Scope == model.LocalSkillScopeWorkspace {
			wsSkillMap[s.WorkspacePath] = append(wsSkillMap[s.WorkspacePath], sv)
		}
	}

	// 补充 pending/failed 的技能（skill_distribution_records 中有但 local_instance_skills 没有的）
	// 这些是已下发但还没 ack 的技能，默认归到 user_level 展示为 distributing/failed。
	// 与上面 rule 的处理对称（rule 走 rule_distribution_records，skill 走 skill_distribution_records）。
	type pendingSkillRow struct {
		TaskSlug string
		SkillSlug string
		SkillName string
		Version  string
		Status   string
	}
	var pendingSkillRows []pendingSkillRow
	if err := model.DB(ctx).
		Model(&model.SkillDistributionRecord{}).
		Select(`skill_distribution_tasks.slug AS task_slug,
		        skills.slug AS skill_slug,
	        skills.name AS skill_name,
	        skill_distribution_records.version AS version,
	        skill_distribution_records.status AS status`).
		Joins("JOIN skill_distribution_tasks ON skill_distribution_tasks.id = skill_distribution_records.task_id").
		Joins("LEFT JOIN skills ON skills.id = skill_distribution_records.skill_id").
		Where("skill_distribution_records.instance_id = ? AND skill_distribution_records.status IN ? AND skill_distribution_records.type = ?",
			inst.ID,
			[]string{model.RecordStatusPending, model.RecordStatusFailed},
			model.TaskTypeDistribute).
		Scan(&pendingSkillRows).Error; err != nil {
		Logger(ctx).Warn("[LocalAgent] failed to query pending skills for resource view",
			"error", err, "instance_id", inst.ID)
		pendingSkillRows = nil
	}
	for _, ps := range pendingSkillRows {
		slug := ps.SkillSlug
		if slug == "" {
			slug = ps.TaskSlug
		}
		// 已有 local_instance_skills 行的跳过（已在上面按 scope 分组了）
		if installedSkillSlugs[slug] {
			continue
		}
		status := model.LocalSkillInstallStatusDistributing
		if ps.Status == model.RecordStatusFailed {
			status = model.LocalSkillInstallStatusFailed
		}
		displayName := ps.SkillName
		if displayName == "" {
			displayName = slug
		}
		userSkills = append(userSkills, SkillStatusView{
			Slug:          slug,
			Version:       ps.Version,
			DisplayName:   displayName,
			Source:        model.LocalSkillSourceEnterprise,
			InstallStatus: status,
		})
	}

	// 查 local_instance_rules 中用户级和项目级快照。
	var installedRules []model.LocalInstanceRule
	if err := model.DB(ctx).Where("instance_id = ? AND (scope = ? OR scope = ?)",
		inst.ID, model.LocalSkillScopeUser, model.LocalSkillScopeWorkspace).
		Find(&installedRules).Error; err != nil {
		Logger(ctx).Warn("[LocalAgent] failed to query installed rules for resource view",
			"error", err, "instance_id", inst.ID)
		installedRules = nil
	}

	// 按 scope + workspace_path 分组（与 skills 同模式）
	userRules := make([]RuleStatusView, 0)
	wsRuleMap := make(map[string][]RuleStatusView)
	installedRuleSlugs := make(map[string]bool, len(installedRules))
	for _, r := range installedRules {
		rv := RuleStatusView{
			Slug:          r.Slug,
			Version:       r.Version,
			DisplayName:   r.DisplayName,
			Type:          r.RuleType,
			Source:        r.Source,
			InstallStatus: r.InstallStatus,
		}
		installedRuleSlugs[r.Slug] = true
		if r.Scope == model.LocalSkillScopeUser {
			userRules = append(userRules, rv)
		} else if r.Scope == model.LocalSkillScopeWorkspace {
			wsRuleMap[r.WorkspacePath] = append(wsRuleMap[r.WorkspacePath], rv)
		}
	}

	// 补充 pending/failed 的规范（rule_distribution_records 中有但 local_instance_rules 没有的）
	// 这些是已下发但还没 ack 的规范，默认归到 user_level 展示为 distributing/failed
	type pendingRuleRow struct {
		TaskSlug string
		RuleSlug string
		RuleName string
		RuleType string
		Version  string
		Status   string
	}
	var pendingRows []pendingRuleRow
	if err := model.DB(ctx).
		Model(&model.RuleDistributionRecord{}).
		Select(`rule_distribution_tasks.slug AS task_slug,
		        enterprise_rules.slug AS rule_slug,
		        enterprise_rules.name AS rule_name,
		        rule_distribution_tasks.rule_type AS rule_type,
		        rule_distribution_records.version AS version,
		        rule_distribution_records.status AS status`).
		Joins("JOIN rule_distribution_tasks ON rule_distribution_tasks.id = rule_distribution_records.task_id").
		Joins("LEFT JOIN enterprise_rules ON enterprise_rules.id = rule_distribution_records.rule_id").
		Where("rule_distribution_records.instance_id = ? AND rule_distribution_records.status IN ? AND rule_distribution_records.type = ?",
			inst.ID,
			[]string{model.RuleRecordStatusPending, model.RuleRecordStatusFailed, model.RuleRecordStatusUpgradeFailed},
			model.RuleTaskTypeDistribute).
		Scan(&pendingRows).Error; err != nil {
		Logger(ctx).Warn("[LocalAgent] failed to query pending rules for resource view",
			"error", err, "instance_id", inst.ID)
		pendingRows = nil
	}
	for _, pr := range pendingRows {
		slug := pr.RuleSlug
		if slug == "" {
			slug = pr.TaskSlug
		}
		// 已有 local_instance_rules 行的跳过（已在上面按 scope 分组了）
		if installedRuleSlugs[slug] {
			continue
		}
		status := model.LocalSkillInstallStatusDistributing
		if pr.Status == model.RuleRecordStatusFailed || pr.Status == model.RuleRecordStatusUpgradeFailed {
			status = model.LocalSkillInstallStatusFailed
		}
		displayName := pr.RuleName
		if displayName == "" {
			displayName = slug
		}
		userRules = append(userRules, RuleStatusView{
			Slug:          slug,
			Version:       pr.Version,
			DisplayName:   displayName,
			Type:          pr.RuleType,
			Source:        model.LocalRuleSourceEnterprise,
			InstallStatus: status,
		})
	}

	// 构建视图
	view := &LocalAgentResourcesView{
		UserLevel: UserLevelView{
			GroupID: resources.UserLevel.GroupID,
			Skills:  userSkills,
			Rules:   userRules,
		},
	}

	// 补充 group_active + group_name（用户可能被移出分组）
	if resources.UserLevel.GroupID != 0 {
		view.UserLevel.GroupActive = computeGroupActive(ctx, userID, resources.UserLevel.GroupID)
		if resources.UserLevel.GroupName == "" {
			view.UserLevel.GroupName = model.GetUserGroupName(ctx, resources.UserLevel.GroupID)
		} else {
			view.UserLevel.GroupName = resources.UserLevel.GroupName
		}
	}

	// 构建 workspaces 视图
	projectIDs := make([]uint, 0, len(resources.Workspaces))
	for _, ws := range resources.Workspaces {
		if ws.ProjectID > 0 {
			projectIDs = append(projectIDs, ws.ProjectID)
		}
	}
	projectByID := make(map[uint]model.Project)
	if len(projectIDs) > 0 {
		var projects []model.Project
		if err := model.DB(ctx).Where("id IN ?", uniqueUintIDs(projectIDs)).Find(&projects).Error; err != nil {
			Logger(ctx).Warn("[LocalAgent] failed to query workspace projects for resource view",
				"error", err, "instance_id", inst.ID, "project_ids", projectIDs)
		} else {
			for _, project := range projects {
				projectByID[project.ID] = project
			}
		}
	}
	view.Workspaces = make([]WorkspaceView, 0, len(resources.Workspaces))
	for _, ws := range resources.Workspaces {
		wv := WorkspaceView{
			Path:      ws.Path,
			Name:      ws.Name,
			IDEType:   ws.IDEType,
			ProjectID: ws.ProjectID,
			Skills:    wsSkillMap[ws.Path],
			Rules:     wsRuleMap[ws.Path],
		}
		if project, ok := projectByID[ws.ProjectID]; ok {
			wv.ProjectName, wv.ProjectExists = project.Name, true
		}
		view.Workspaces = append(view.Workspaces, wv)
	}

	return view
}
