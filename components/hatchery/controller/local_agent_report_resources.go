package controller

// 本地 Agent report 的用户级、Workspace 资源对齐。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"hatchery/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func processReportUserLevelAndWorkspaces(ctx context.Context, tx *gorm.DB, inst *model.Instance,
	user *model.User, req *reportRequest) (userSynced, projectSynced, rulesSynced int, err error) {

	now := time.Now()

	// 反序列化现有 local_agent_resources（可能为 nil）
	resources := deserializeLocalAgentResources(inst.LocalAgentResources)

	// ─── 用户级处理 ───
	if req.UserLevel != nil {
		us, rs, err := processReportUserLevel(tx, inst, req.UserLevel, now)
		if err != nil {
			return 0, 0, 0, err
		}
		userSynced = us
		rulesSynced += rs
		if err := upsertUserScopeBinding(tx, inst.ID, resources.UserLevel.GroupID, now); err != nil {
			return 0, 0, 0, err
		}
	}

	// ─── 项目级处理 ───
	if len(req.Workspaces) > 0 {
		ps, rs, err := processReportWorkspaces(ctx, tx, inst, user, resources, req.Workspaces, now)
		if err != nil {
			return userSynced, 0, 0, err
		}
		projectSynced = ps
		rulesSynced += rs
	}

	// 序列化回 instances.local_agent_resources
	jsonBytes, mErr := json.Marshal(resources)
	if mErr != nil {
		return userSynced, projectSynced, rulesSynced, fmt.Errorf("序列化 local_agent_resources: %w", mErr)
	}
	if err := tx.Model(&model.Instance{}).Where("id = ?", inst.ID).
		Update("local_agent_resources", string(jsonBytes)).Error; err != nil {
		return userSynced, projectSynced, rulesSynced, fmt.Errorf("更新 local_agent_resources: %w", err)
	}

	return userSynced, projectSynced, rulesSynced, nil
}

// processReportUserLevel 处理用户级资源（skills + rules 全量对齐）。
// 分组管理由 ensureUserLevelGroup 按服务端用户关系统一负责；本函数不采纳
// report.user_level.group_id，也不自行处理 group_id 变化。
func processReportUserLevel(tx *gorm.DB, inst *model.Instance,
	ul *reportUserLevel, now time.Time) (skillSynced, ruleSynced int, err error) {
	// 全量对齐 local_instance_skills(scope='user')
	skillSynced, err = alignLocalSkills(tx, inst.ID, model.LocalSkillScopeUser, "", ul.Skills, now)
	if err != nil {
		return 0, 0, fmt.Errorf("用户级 skill 对齐: %w", err)
	}

	// 全量对齐 local_instance_rules(scope='user')，仅当 rules 字段存在时触发
	if ul.Rules != nil {
		ruleSynced, err = alignLocalRules(tx, inst.ID, model.LocalSkillScopeUser, "", ul.Rules, now)
		if err != nil {
			return skillSynced, 0, fmt.Errorf("用户级 rule 对齐: %w", err)
		}
	}
	return skillSynced, ruleSynced, nil
}

func resolveWorkspaceProjectIDFromSet(oldProjectID uint, requested *uint, memberProjectIDs map[uint]struct{}) (uint, error) {
	if requested == nil {
		return oldProjectID, nil
	}
	projectID := *requested
	if projectID == 0 || projectID == oldProjectID {
		return projectID, nil
	}
	if _, ok := memberProjectIDs[projectID]; !ok {
		return 0, gorm.ErrRecordNotFound
	}
	return projectID, nil
}

func loadWorkspaceProjectSets(tx *gorm.DB, userID uint, oldProjectByPath map[string]uint,
	workspaces []reportWorkspace) (memberProjectIDs, existingProjectIDs map[uint]struct{}, err error) {
	memberCandidates := make(map[uint]struct{})
	projectCandidates := make(map[uint]struct{})
	seenPaths := make(map[string]struct{}, len(workspaces))
	for _, workspace := range workspaces {
		path := strings.TrimSpace(workspace.Path)
		if path == "" {
			continue
		}
		if _, seen := seenPaths[path]; seen {
			continue
		}
		seenPaths[path] = struct{}{}
		oldProjectID := oldProjectByPath[path]
		projectID := oldProjectID
		if workspace.ProjectID != nil {
			projectID = *workspace.ProjectID
			if projectID != 0 && projectID != oldProjectID {
				memberCandidates[projectID] = struct{}{}
			}
		}
		if projectID != 0 {
			projectCandidates[projectID] = struct{}{}
		}
	}

	memberProjectIDs = make(map[uint]struct{}, len(memberCandidates))
	if len(memberCandidates) > 0 {
		var ids []uint
		if err := tx.Model(&model.ProjectMember{}).
			Where("user_id = ? AND project_id IN ?", userID, uintSetKeys(memberCandidates)).
			Pluck("project_id", &ids).Error; err != nil {
			return nil, nil, err
		}
		for _, id := range ids {
			memberProjectIDs[id] = struct{}{}
		}
	}

	existingProjectIDs = make(map[uint]struct{}, len(projectCandidates))
	if len(projectCandidates) > 0 {
		var ids []uint
		if err := tx.Model(&model.Project{}).Where("id IN ?", uintSetKeys(projectCandidates)).
			Pluck("id", &ids).Error; err != nil {
			return nil, nil, err
		}
		for _, id := range ids {
			existingProjectIDs[id] = struct{}{}
		}
	}
	return memberProjectIDs, existingProjectIDs, nil
}

func uintSetKeys(values map[uint]struct{}) []uint {
	keys := make([]uint, 0, len(values))
	for value := range values {
		keys = append(keys, value)
	}
	return keys
}

func upsertWorkspaceScopeBindings(tx *gorm.DB, instID uint, workspaces []model.WorkspaceResource, now time.Time) error {
	if len(workspaces) == 0 {
		return nil
	}
	bindings := make([]model.LocalAgentScopeBinding, 0, len(workspaces))
	for _, ws := range workspaces {
		bindings = append(bindings, model.LocalAgentScopeBinding{InstanceID: instID, Scope: model.LocalAgentScopeWorkspace, ScopeKey: ws.Path, ScopeName: ws.Name, IDEType: ws.IDEType, ProjectID: ws.ProjectID, LastSeenAt: &now})
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "identifier"}, {Name: "instance_id"}, {Name: "scope"}, {Name: "scope_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"scope_name", "ide_type", "project_id", "last_seen_at", "updated_at"}),
	}).Create(&bindings).Error
}

func upsertWorkspaceScopeBinding(tx *gorm.DB, instID uint, workspace model.WorkspaceResource, now time.Time) error {
	return upsertWorkspaceScopeBindings(tx, instID, []model.WorkspaceResource{workspace}, now)
}

func upsertUserScopeBinding(tx *gorm.DB, instID, groupID uint, now time.Time) error {
	binding := model.LocalAgentScopeBinding{InstanceID: instID, Scope: model.LocalAgentScopeUser, ScopeKey: "", GroupID: groupID, LastSeenAt: &now}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "identifier"}, {Name: "instance_id"}, {Name: "scope"}, {Name: "scope_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"group_id", "last_seen_at", "updated_at"}),
	}).Create(&binding).Error
}

// processReportWorkspaces 处理项目级资源（每个 workspace）。
// 返回 skill_synced 和 rule_synced 计数。
func processReportWorkspaces(_ context.Context, tx *gorm.DB, inst *model.Instance,
	user *model.User, resources *model.LocalAgentResources,
	workspaces []reportWorkspace, now time.Time) (skillSynced, ruleSynced int, err error) {

	reportedPaths := make(map[string]bool)
	for _, ws := range workspaces {
		if path := strings.TrimSpace(ws.Path); path != "" {
			reportedPaths[path] = true
		}
	}
	existingSkillsByPath, err := loadWorkspaceSkillsByPath(tx, inst.ID, workspacePathKeys(reportedPaths))
	if err != nil {
		return 0, 0, err
	}
	existingRulesByPath := make(map[string][]model.LocalInstanceRule)
	if hasWorkspaceRules(workspaces) {
		existingRulesByPath, err = loadWorkspaceRulesByPath(tx, inst.ID, workspacePathKeys(reportedPaths))
		if err != nil {
			return 0, 0, err
		}
	}
	reportedPaths = make(map[string]bool, len(reportedPaths))
	oldProjectByPath := make(map[string]uint, len(resources.Workspaces))
	for _, workspace := range resources.Workspaces {
		if _, exists := oldProjectByPath[workspace.Path]; !exists {
			oldProjectByPath[workspace.Path] = workspace.ProjectID
		}
	}
	memberProjectIDs, existingProjectIDs, err := loadWorkspaceProjectSets(tx, user.ID, oldProjectByPath, workspaces)
	if err != nil {
		return 0, 0, fmt.Errorf("批量查询 workspace 项目关系: %w", err)
	}

	// 构建新 workspace 列表（保留 report 里的顺序 + 信息）
	newWorkspaces := make([]model.WorkspaceResource, 0, len(workspaces))

	for _, ws := range workspaces {
		path := strings.TrimSpace(ws.Path)
		if path == "" {
			continue
		}
		if reportedPaths[path] {
			continue // 同一批重复 path 只取首次
		}
		reportedPaths[path] = true

		// project_id 缺失时保持旧值；变更到新项目时校验批量预取的成员关系。
		projectID, err := resolveWorkspaceProjectIDFromSet(oldProjectByPath[path], ws.ProjectID, memberProjectIDs)
		if err != nil {
			return skillSynced, ruleSynced, fmt.Errorf("校验 workspace project_id (path=%s): %w", path, err)
		}

		// 更新 resources.Workspaces
		wsRes := model.WorkspaceResource{
			Path:      path,
			Name:      ws.Name,
			IDEType:   ws.IDEType,
			ProjectID: projectID,
		}
		newWorkspaces = append(newWorkspaces, wsRes)
		if projectID > 0 {
			if _, exists := existingProjectIDs[projectID]; !exists {
				// 删除项目后保留历史绑定和已装资源快照，不做目录对账。
				continue
			}
		}
		// 全量对齐 local_instance_skills(scope='workspace', workspace_path=path)
		count, err := alignLocalSkillsWithExisting(tx, inst.ID, model.LocalSkillScopeWorkspace, path, ws.Skills, existingSkillsByPath[path], now)
		if err != nil {
			return skillSynced, ruleSynced, fmt.Errorf("项目级 skill 对齐 (path=%s): %w", path, err)
		}
		skillSynced += count

		// 全量对齐 local_instance_rules(scope='workspace', workspace_path=path)
		if ws.Rules != nil {
			rcount, err := alignLocalRulesWithExisting(tx, inst.ID, model.LocalSkillScopeWorkspace, path, ws.Rules, existingRulesByPath[path], now)
			if err != nil {
				return skillSynced, ruleSynced, fmt.Errorf("项目级 rule 对齐 (path=%s): %w", path, err)
			}
			ruleSynced += rcount
		}
	}
	if err := upsertWorkspaceScopeBindings(tx, inst.ID, newWorkspaces, now); err != nil {
		return skillSynced, ruleSynced, fmt.Errorf("批量写入 workspace project binding: %w", err)
	}

	// 消失即删：旧 workspace 不在新 report 里 → 从 resources 删除
	// 注意：local_instance_skills 的消失即删已在 alignLocalSkills 内处理
	resources.Workspaces = newWorkspaces
	if err := tx.Where("instance_id = ? AND scope = ? AND scope_key NOT IN ?", inst.ID, model.LocalAgentScopeWorkspace, workspacePathKeys(reportedPaths)).Delete(&model.LocalAgentScopeBinding{}).Error; err != nil {
		return skillSynced, ruleSynced, fmt.Errorf("清理 workspace project binding: %w", err)
	}

	return skillSynced, ruleSynced, nil
}

func hasWorkspaceRules(workspaces []reportWorkspace) bool {
	for _, workspace := range workspaces {
		if workspace.Rules != nil {
			return true
		}
	}
	return false
}

func loadWorkspaceSkillsByPath(tx *gorm.DB, instanceID uint, paths []string) (map[string][]model.LocalInstanceSkill, error) {
	var rows []model.LocalInstanceSkill
	if err := tx.Where("instance_id = ? AND scope = ? AND workspace_path IN ?", instanceID, model.LocalSkillScopeWorkspace, paths).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("批量查询 workspace skills: %w", err)
	}
	result := make(map[string][]model.LocalInstanceSkill, len(paths))
	for _, row := range rows {
		result[row.WorkspacePath] = append(result[row.WorkspacePath], row)
	}
	return result, nil
}

func loadWorkspaceRulesByPath(tx *gorm.DB, instanceID uint, paths []string) (map[string][]model.LocalInstanceRule, error) {
	var rows []model.LocalInstanceRule
	if err := tx.Where("instance_id = ? AND scope = ? AND workspace_path IN ?", instanceID, model.LocalSkillScopeWorkspace, paths).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("批量查询 workspace rules: %w", err)
	}
	result := make(map[string][]model.LocalInstanceRule, len(paths))
	for _, row := range rows {
		result[row.WorkspacePath] = append(result[row.WorkspacePath], row)
	}
	return result, nil
}

func workspacePathKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return []string{""}
	}
	return keys
}

// alignLocalSkills 全量对齐某 scope+workspace_path 下的 local_instance_skills。
// 规则（对齐一期 "消失即删"）：
//   - report 出现、DB 没有 → 插入 install_status='distributed' + 同 slug pending record 标 cancelled
//   - report 出现、DB 有 → upsert（不改 install_status），刷新 LastSeenAt
//   - DB 有、report 没出现 → 硬删（消失即删）
//
// 返回 report 中出现的 skill 数量。
func alignLocalSkills(tx *gorm.DB, instanceID uint, scope string, workspacePath string,
	reported []reportSkillEntry, now time.Time) (int, error) {
	var existing []model.LocalInstanceSkill
	if err := tx.Where("instance_id = ? AND scope = ? AND workspace_path = ?",
		instanceID, scope, workspacePath).Find(&existing).Error; err != nil {
		return 0, fmt.Errorf("查询 existing local_instance_skills: %w", err)
	}
	return alignLocalSkillsWithExisting(tx, instanceID, scope, workspacePath, reported, existing, now)
}

func alignLocalSkillsWithExisting(tx *gorm.DB, instanceID uint, scope, workspacePath string,
	reported []reportSkillEntry, existing []model.LocalInstanceSkill, now time.Time) (int, error) {
	existingBySlug := make(map[string]model.LocalInstanceSkill, len(existing))
	for _, row := range existing {
		existingBySlug[row.Slug] = row
	}
	reportedSlugs := make(map[string]bool, len(reported))
	newSlugs := make([]string, 0, len(reported))
	newRows := make([]model.LocalInstanceSkill, 0, len(reported))
	sameIDs := make([]uint, 0, len(reported))
	changedRows := make([]model.LocalInstanceSkill, 0, len(reported))
	for _, s := range reported {
		slug := strings.TrimSpace(s.Slug)
		if slug == "" || reportedSlugs[slug] {
			continue
		}
		reportedSlugs[slug] = true
		source := strings.ToLower(strings.TrimSpace(s.Source))
		if source == "" {
			source = model.LocalSkillSourceLocal
		}
		row := model.LocalInstanceSkill{InstanceID: instanceID, Slug: slug, Version: s.Version,
			DisplayName: s.DisplayName, Source: source, Scope: scope, WorkspacePath: workspacePath,
			InstallStatus: model.LocalSkillInstallStatusDistributed, LastSeenAt: &now}
		current, ok := existingBySlug[slug]
		if !ok {
			row.InstalledAt = &now
			newSlugs = append(newSlugs, slug)
			newRows = append(newRows, row)
		} else if current.Version == row.Version && current.DisplayName == row.DisplayName && current.Source == row.Source {
			sameIDs = append(sameIDs, current.ID)
		} else {
			changedRows = append(changedRows, row)
		}
	}
	if len(newRows) > 0 {
		if err := tx.Create(&newRows).Error; err != nil {
			return len(reportedSlugs), fmt.Errorf("批量插入 local_instance_skills: %w", err)
		}
	}
	if len(sameIDs) > 0 {
		if err := tx.Model(&model.LocalInstanceSkill{}).Where("id IN ?", sameIDs).UpdateColumn("last_seen_at", &now).Error; err != nil {
			return len(reportedSlugs), fmt.Errorf("批量刷新 local_instance_skills: %w", err)
		}
	}
	if len(changedRows) > 0 {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "scope"}, {Name: "instance_id"}, {Name: "workspace_path"}, {Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{"version", "display_name", "source", "last_seen_at", "updated_at"}),
		}).Create(&changedRows).Error; err != nil {
			return len(reportedSlugs), fmt.Errorf("批量更新 local_instance_skills: %w", err)
		}
	}
	if err := cancelPendingSkills(tx, instanceID, newSlugs); err != nil {
		return len(reportedSlugs), err
	}
	toDelete := make([]uint, 0, len(existing))
	for slug, cur := range existingBySlug {
		if !reportedSlugs[slug] && cur.InstallStatus == model.LocalSkillInstallStatusDistributed {
			toDelete = append(toDelete, cur.ID)
		}
	}
	if len(toDelete) > 0 {
		if err := tx.Where("id IN ?", toDelete).Delete(&model.LocalInstanceSkill{}).Error; err != nil {
			return len(reportedSlugs), fmt.Errorf("删除消失的 local_instance_skills: %w", err)
		}
	}
	return len(reportedSlugs), nil
}

// alignLocalRules 全量对齐某 scope+workspace_path 下的 local_instance_rules（消失即删）。
// 与 alignLocalSkills 同模式，支持 user/project scope。
//
// 规则：
//   - report 出现、DB 没有 → INSERT（installed_at + last_seen_at = now）
//   - report 出现、DB 有且信息一致 → 只刷 last_seen_at
//   - report 出现、DB 有但信息变了 → UPDATE 业务字段
//   - DB 有、report 没出现 → 硬删（消失即删）
//
// 返回 report 中出现的 rule 数量。
func alignLocalRules(tx *gorm.DB, instanceID uint, scope string, workspacePath string,
	reported []reportRuleEntry, now time.Time) (int, error) {
	var existing []model.LocalInstanceRule
	if err := tx.Where("instance_id = ? AND scope = ? AND workspace_path = ?",
		instanceID, scope, workspacePath).Find(&existing).Error; err != nil {
		return 0, fmt.Errorf("查询 existing local_instance_rules: %w", err)
	}
	return alignLocalRulesWithExisting(tx, instanceID, scope, workspacePath, reported, existing, now)
}

func alignLocalRulesWithExisting(tx *gorm.DB, instanceID uint, scope, workspacePath string,
	reported []reportRuleEntry, existing []model.LocalInstanceRule, now time.Time) (int, error) {
	existingBySlug := make(map[string]model.LocalInstanceRule, len(existing))
	for _, row := range existing {
		existingBySlug[row.Slug] = row
	}
	reportedSlugs := make(map[string]bool, len(reported))
	newSlugs := make([]string, 0, len(reported))
	newRows := make([]model.LocalInstanceRule, 0, len(reported))
	sameIDs := make([]uint, 0, len(reported))
	changedRows := make([]model.LocalInstanceRule, 0, len(reported))
	for _, r := range reported {
		slug := strings.TrimSpace(r.Slug)
		if slug == "" || reportedSlugs[slug] {
			continue
		}
		reportedSlugs[slug] = true
		ruleType := strings.ToLower(strings.TrimSpace(r.RuleType))
		source := strings.ToLower(strings.TrimSpace(r.Source))
		if source == "" {
			source = model.LocalRuleSourceLocal
		}
		row := model.LocalInstanceRule{InstanceID: instanceID, Slug: slug, Version: r.Version,
			DisplayName: r.DisplayName, RuleType: ruleType, Source: source, Scope: scope,
			WorkspacePath: workspacePath, InstallStatus: model.LocalSkillInstallStatusDistributed, LastSeenAt: &now}
		current, ok := existingBySlug[slug]
		if !ok {
			row.InstalledAt = &now
			newSlugs = append(newSlugs, slug)
			newRows = append(newRows, row)
		} else if current.Version == row.Version && current.DisplayName == row.DisplayName && current.RuleType == row.RuleType && current.Source == row.Source {
			sameIDs = append(sameIDs, current.ID)
		} else {
			changedRows = append(changedRows, row)
		}
	}
	if len(newRows) > 0 {
		if err := tx.Create(&newRows).Error; err != nil {
			return len(reportedSlugs), fmt.Errorf("批量插入 local_instance_rules: %w", err)
		}
	}
	if len(sameIDs) > 0 {
		if err := tx.Model(&model.LocalInstanceRule{}).Where("id IN ?", sameIDs).UpdateColumn("last_seen_at", &now).Error; err != nil {
			return len(reportedSlugs), fmt.Errorf("批量刷新 local_instance_rules: %w", err)
		}
	}
	if len(changedRows) > 0 {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "scope"}, {Name: "instance_id"}, {Name: "workspace_path"}, {Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{"version", "display_name", "rule_type", "source", "last_seen_at", "updated_at"}),
		}).Create(&changedRows).Error; err != nil {
			return len(reportedSlugs), fmt.Errorf("批量更新 local_instance_rules: %w", err)
		}
	}
	if err := cancelPendingRules(tx, instanceID, newSlugs); err != nil {
		return len(reportedSlugs), err
	}
	toDelete := make([]uint, 0, len(existing))
	for slug, cur := range existingBySlug {
		if !reportedSlugs[slug] && cur.InstallStatus == model.LocalSkillInstallStatusDistributed {
			toDelete = append(toDelete, cur.ID)
		}
	}
	if len(toDelete) > 0 {
		if err := tx.Where("id IN ?", toDelete).Delete(&model.LocalInstanceRule{}).Error; err != nil {
			return len(reportedSlugs), fmt.Errorf("删除消失的 local_instance_rules: %w", err)
		}
	}

	return len(reportedSlugs), nil
}

func cancelPendingSkills(tx *gorm.DB, instanceID uint, slugs []string) error {
	if len(slugs) == 0 {
		return nil
	}
	var ids []uint
	err := tx.Model(&model.SkillDistributionRecord{}).Select("skill_distribution_records.id").
		Joins("JOIN skill_distribution_tasks ON skill_distribution_tasks.id = skill_distribution_records.task_id").
		Joins("LEFT JOIN skills ON skills.id = skill_distribution_records.skill_id").
		Where("skill_distribution_records.instance_id = ? AND skill_distribution_records.status = ? AND COALESCE(skills.slug, skill_distribution_tasks.slug) IN ?", instanceID, model.RecordStatusPending, slugs).
		Scan(&ids).Error
	if err != nil || len(ids) == 0 {
		return err
	}
	return tx.Model(&model.SkillDistributionRecord{}).Where("id IN ?", ids).Update("status", model.RecordStatusCancelled).Error
}

func cancelPendingRules(tx *gorm.DB, instanceID uint, slugs []string) error {
	if len(slugs) == 0 {
		return nil
	}
	var ids []uint
	err := tx.Model(&model.RuleDistributionRecord{}).Select("rule_distribution_records.id").
		Joins("JOIN rule_distribution_tasks ON rule_distribution_tasks.id = rule_distribution_records.task_id").
		Joins("LEFT JOIN enterprise_rules ON enterprise_rules.id = rule_distribution_records.rule_id").
		Where("rule_distribution_records.instance_id = ? AND rule_distribution_records.status = ? AND COALESCE(enterprise_rules.slug, rule_distribution_tasks.slug) IN ?", instanceID, model.RuleRecordStatusPending, slugs).
		Scan(&ids).Error
	if err != nil || len(ids) == 0 {
		return err
	}
	return tx.Model(&model.RuleDistributionRecord{}).Where("id IN ?", ids).Update("status", model.RuleRecordStatusCancelled).Error
}

// ---- sync 二期处理 -------------------------------------------------------

// processSyncWorkspaces 处理 Workspace 关系，并按 project_id 对账项目直接资产。
