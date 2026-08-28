package controller

// 本地 Agent 资源差异计算与待执行任务入队。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hatchery/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func diffAndQueue(ctx context.Context, tx *gorm.DB, scope string, instanceID uint,
	newGroupID uint, workspacePath string) (int, error) {
	return diffTargetAndQueue(ctx, tx, scope, instanceID, newGroupID, 0, workspacePath)
}

// diffProjectAndQueue 按项目直接资产为指定 Workspace 生成按需安装命令。
func diffProjectAndQueue(ctx context.Context, tx *gorm.DB, instanceID, projectID uint,
	workspacePath string) (int, error) {
	return diffTargetAndQueue(ctx, tx, model.LocalSkillScopeWorkspace, instanceID, 0, projectID, workspacePath)
}

func diffTargetAndQueue(ctx context.Context, tx *gorm.DB, scope string, instanceID uint,
	newGroupID, projectID uint, workspacePath string) (int, error) {
	if err := validateLocalScopeTarget(scope, newGroupID, projectID, workspacePath); err != nil {
		return 0, err
	}
	if projectID == 0 {
		if err := cleanupLocalSkillState(tx, instanceID, scope, workspacePath); err != nil {
			return 0, err
		}
	}

	catalog, err := loadLocalTargetCatalog(ctx, tx, scope, newGroupID, projectID)
	if err != nil {
		return 0, err
	}
	installed, err := model.ListInstalledSkillsWithDB(tx, scope, instanceID, workspacePath)
	if err != nil {
		return 0, fmt.Errorf("查询已装 skill: %w", err)
	}
	Logger(ctx).Info("[diffAndQueue] diff detail",
		"scopes", scope, "instance_id", instanceID, "group_id", newGroupID, "project_id", projectID,
		"workspace_path", workspacePath,
		"catalog_count", len(catalog.Skills), "installed_count", len(installed))

	now := time.Now()
	pendingCount, err := queueSkillCatalogDiff(tx, instanceID, scope, workspacePath, catalog.Skills, installed, now)
	if err != nil {
		return pendingCount, err
	}
	ruleCount, err := queueRuleCatalogDiff(tx, instanceID, scope, workspacePath, projectID, catalog.Rules, now)
	return pendingCount + ruleCount, err
}

func cleanupLocalSkillState(tx *gorm.DB, instanceID uint, scope, workspacePath string) error {
	if err := tx.Where("instance_id = ? AND scope = ? AND workspace_path = ? AND install_status IN ?",
		instanceID, scope, workspacePath,
		[]string{model.LocalSkillInstallStatusFailed, model.LocalSkillInstallStatusDistributing}).
		Delete(&model.LocalInstanceSkill{}).Error; err != nil {
		return fmt.Errorf("清理 failed/distributing local_instance_skills: %w", err)
	}
	var scopeSlugs []string
	if err := tx.Model(&model.LocalInstanceSkill{}).
		Where("instance_id = ? AND scope = ? AND workspace_path = ?", instanceID, scope, workspacePath).
		Pluck("slug", &scopeSlugs).Error; err != nil {
		return fmt.Errorf("查询 scope slugs: %w", err)
	}
	recordIDs, err := findSkillRecordIDsToCancel(tx, instanceID, scopeSlugs)
	if err != nil || len(recordIDs) == 0 {
		return err
	}
	if err := tx.Model(&model.SkillDistributionRecord{}).Where("id IN ?", recordIDs).
		Update("status", model.RecordStatusCancelled).Error; err != nil {
		return fmt.Errorf("清理 pending/failed records: %w", err)
	}
	return nil
}

func findSkillRecordIDsToCancel(tx *gorm.DB, instanceID uint, scopeSlugs []string) ([]uint, error) {
	var recordIDs []uint
	statuses := []string{model.RecordStatusPending, model.RecordStatusFailed}
	if len(scopeSlugs) == 0 {
		if err := tx.Model(&model.SkillDistributionRecord{}).
			Where("instance_id = ? AND status IN ?", instanceID, statuses).
			Pluck("id", &recordIDs).Error; err != nil {
			return nil, fmt.Errorf("查询 pending/failed records (all): %w", err)
		}
		return recordIDs, nil
	}
	type recordRow struct {
		ID       uint
		Slug     string
		TaskSlug string
	}
	var rows []recordRow
	if err := tx.Model(&model.SkillDistributionRecord{}).
		Select("skill_distribution_records.id AS id, skills.slug AS slug, skill_distribution_tasks.slug AS task_slug").
		Joins("LEFT JOIN skills ON skills.id = skill_distribution_records.skill_id").
		Joins("LEFT JOIN skill_distribution_tasks ON skill_distribution_tasks.id = skill_distribution_records.task_id").
		Where("skill_distribution_records.instance_id = ? AND skill_distribution_records.status IN ?", instanceID, statuses).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询 pending/failed records: %w", err)
	}
	scopeSlugSet := make(map[string]bool, len(scopeSlugs))
	for _, slug := range scopeSlugs {
		scopeSlugSet[slug] = true
	}
	for _, row := range rows {
		slug := row.Slug
		if slug == "" {
			slug = row.TaskSlug
		}
		if scopeSlugSet[slug] {
			recordIDs = append(recordIDs, row.ID)
		}
	}
	return recordIDs, nil
}

func loadLocalTargetCatalog(ctx context.Context, tx *gorm.DB, scope string, groupID, projectID uint) (model.AssetCatalog, error) {
	// 必须复用事务句柄；使用全局 DB 会导致事务内数据不可见，sqlite :memory 下还可能死锁。
	var catalog model.AssetCatalog
	var err error
	if scope == model.LocalSkillScopeUser {
		catalog, err = model.ListAssetCatalogByGroupWithDB(tx, ctx, groupID)
	} else {
		catalog, err = model.ListAssetCatalogByProjectWithDB(tx, ctx, projectID)
	}
	if err != nil {
		return catalog, fmt.Errorf("查询 scope 绑定资产 catalog: %w", err)
	}
	return catalog, nil
}

func queueSkillCatalogDiff(tx *gorm.DB, instanceID uint, scope, workspacePath string,
	catalog []model.SkillCatalogItem, installed []model.LocalInstanceSkill, now time.Time) (int, error) {
	installedMap := make(map[string]*model.LocalInstanceSkill, len(installed))
	for i := range installed {
		installedMap[installed[i].Slug] = &installed[i]
	}
	pendingCount := 0
	for _, item := range catalog {
		cur, ok := installedMap[item.Slug]
		switch {
		case !ok:
			if err := createPendingRecordAndLocalSkill(tx, instanceID, item, scope, workspacePath, now); err != nil {
				return pendingCount, err
			}
			pendingCount++
		case cur.InstallStatus == model.LocalSkillInstallStatusFailed || model.VersionScore(cur.Version) != model.VersionScore(item.Version):
			if err := updatePendingRecordAndLocalSkill(tx, instanceID, cur.ID, item, now); err != nil {
				return pendingCount, err
			}
			pendingCount++
		}
	}
	return pendingCount, nil
}

func queueRuleCatalogDiff(tx *gorm.DB, instanceID uint, scope, workspacePath string, projectID uint,
	catalog []model.RuleCatalogItem, now time.Time) (int, error) {
	if len(catalog) == 0 {
		return 0, nil
	}
	if projectID == 0 {
		if err := tx.Where("instance_id = ? AND scope = ? AND workspace_path = ? AND install_status IN ?",
			instanceID, scope, workspacePath,
			[]string{model.LocalSkillInstallStatusFailed, model.LocalSkillInstallStatusDistributing}).
			Delete(&model.LocalInstanceRule{}).Error; err != nil {
			return 0, fmt.Errorf("清理 failed/distributing local_instance_rules: %w", err)
		}
	}
	installed, err := model.ListInstalledRulesWithDB(tx, scope, instanceID, workspacePath)
	if err != nil {
		return 0, fmt.Errorf("查询已装 rule: %w", err)
	}
	installedMap := make(map[string]*model.LocalInstanceRule, len(installed))
	for i := range installed {
		installedMap[installed[i].Slug] = &installed[i]
	}
	pendingCount := 0
	for _, item := range catalog {
		cur, ok := installedMap[item.Slug]
		switch {
		case !ok:
			err = createPendingRuleRecordAndLocalRule(tx, instanceID, item, scope, workspacePath, now)
		case cur.InstallStatus == model.LocalSkillInstallStatusFailed || model.VersionScore(cur.Version) != model.VersionScore(item.Version):
			err = updatePendingRuleRecordAndLocalRule(tx, instanceID, cur.ID, item, now)
		default:
			continue
		}
		if err != nil {
			return pendingCount, err
		}
		pendingCount++
	}
	return pendingCount, nil
}

func validateLocalScopeTarget(scope string, groupID, projectID uint, workspacePath string) error {
	switch scope {
	case model.LocalSkillScopeUser:
		if groupID == 0 || projectID != 0 || workspacePath != "" {
			return fmt.Errorf("用户级 scope 必须且只能绑定分组")
		}
	case model.LocalSkillScopeWorkspace:
		if projectID == 0 || groupID != 0 || strings.TrimSpace(workspacePath) == "" {
			return fmt.Errorf("workspace scope 必须且只能绑定项目")
		}
	default:
		return fmt.Errorf("未知本地资源 scope: %s", scope)
	}
	return nil
}

// createPendingRecordAndLocalSkill 写一条 pending record + upsert local_instance_skills(distributing)。
func createPendingRecordAndLocalSkill(tx *gorm.DB, instanceID uint, item model.SkillCatalogItem,
	scope, workspacePath string, now time.Time) error {
	// 先写本地 scope 行，借其 ID 直接构造 task.batch_id，避免 task 创建后的回写 UPDATE。
	row := model.LocalInstanceSkill{
		InstanceID:    instanceID,
		Slug:          item.Slug,
		Version:       item.Version,
		DisplayName:   item.DisplayName,
		Source:        item.Source,
		Scope:         scope,
		WorkspacePath: workspacePath,
		InstallStatus: model.LocalSkillInstallStatusDistributing,
		LastSeenAt:    &now,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "scope"},
			{Name: "instance_id"},
			{Name: "workspace_path"},
			{Name: "slug"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"version", "display_name", "source", "install_status",
			"last_seen_at", "updated_at",
		}),
	}).Create(&row).Error; err != nil {
		return fmt.Errorf("upsert local_instance_skills (slug=%s): %w", item.Slug, err)
	}
	if row.ID == 0 {
		if err := tx.Where("scope = ? AND instance_id = ? AND workspace_path = ? AND slug = ?", scope, instanceID, workspacePath, item.Slug).First(&row).Error; err != nil {
			return err
		}
	}
	task := model.SkillDistributionTask{SkillID: item.SkillID, Version: item.Version, Source: item.Source,
		Slug: item.Slug, BatchID: localScopeBatchID(row.ID), Total: 1, Status: model.TaskStatusCompleted, Type: model.TaskTypeDistribute}
	if err := tx.Create(&task).Error; err != nil {
		return fmt.Errorf("创建 distribution task (slug=%s): %w", item.Slug, err)
	}
	rec := model.SkillDistributionRecord{InstanceID: instanceID, TaskID: task.ID, SkillID: item.SkillID,
		Type: model.TaskTypeDistribute, Version: item.Version, Status: model.RecordStatusPending}
	if err := tx.Create(&rec).Error; err != nil {
		return fmt.Errorf("创建 pending record (slug=%s): %w", item.Slug, err)
	}

	return nil
}

// ---- rule pending 创建/更新（与 skill 同模式）----

// createPendingRuleRecordAndLocalRule 写一条 pending rule record + upsert local_instance_rules(distributing)。
func createPendingRuleRecordAndLocalRule(tx *gorm.DB, instanceID uint, item model.RuleCatalogItem,
	scope, workspacePath string, now time.Time) error {
	row := model.LocalInstanceRule{
		InstanceID:    instanceID,
		Slug:          item.Slug,
		Version:       item.Version,
		DisplayName:   item.DisplayName,
		RuleType:      item.RuleType,
		Source:        item.Source,
		Scope:         scope,
		WorkspacePath: workspacePath,
		InstallStatus: model.LocalSkillInstallStatusDistributing,
		LastSeenAt:    &now,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "scope"},
			{Name: "instance_id"},
			{Name: "workspace_path"},
			{Name: "slug"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"version", "display_name", "rule_type", "source", "install_status",
			"last_seen_at", "updated_at",
		}),
	}).Create(&row).Error; err != nil {
		return fmt.Errorf("upsert local_instance_rules (slug=%s): %w", item.Slug, err)
	}
	if row.ID == 0 {
		if err := tx.Where("scope = ? AND instance_id = ? AND workspace_path = ? AND slug = ?", scope, instanceID, workspacePath, item.Slug).First(&row).Error; err != nil {
			return err
		}
	}
	task := model.RuleDistributionTask{RuleID: item.RuleID, Slug: item.Slug, RuleType: item.RuleType,
		Version: item.Version, BatchID: localScopeBatchID(row.ID), Total: 1, Status: model.TaskStatusCompleted, Type: model.RuleTaskTypeDistribute}
	if err := tx.Create(&task).Error; err != nil {
		return fmt.Errorf("创建 rule distribution task (slug=%s): %w", item.Slug, err)
	}
	rec := model.RuleDistributionRecord{InstanceID: instanceID, TaskID: task.ID, RuleID: item.RuleID,
		Type: model.RuleTaskTypeDistribute, Version: item.Version, Status: model.RuleRecordStatusPending}
	if err := tx.Create(&rec).Error; err != nil {
		return fmt.Errorf("创建 pending rule record (slug=%s): %w", item.Slug, err)
	}

	return nil
}

// updatePendingRuleRecordAndLocalRule 对已装但版本不同的 rule 写 pending + update distributing。
func updatePendingRuleRecordAndLocalRule(tx *gorm.DB, instanceID, localRuleID uint,
	item model.RuleCatalogItem, now time.Time) error {

	task := model.RuleDistributionTask{
		RuleID:   item.RuleID,
		Slug:     item.Slug,
		RuleType: item.RuleType,
		Version:  item.Version,
		Total:    1,
		Status:   model.TaskStatusCompleted,
		Type:     model.RuleTaskTypeDistribute,
		BatchID:  localScopeBatchID(localRuleID),
	}
	if err := tx.Create(&task).Error; err != nil {
		return fmt.Errorf("创建 rule distribution task (slug=%s, upgrade): %w", item.Slug, err)
	}

	rec := model.RuleDistributionRecord{
		InstanceID: instanceID,
		TaskID:     task.ID,
		RuleID:     item.RuleID,
		Type:       model.RuleTaskTypeDistribute,
		Version:    item.Version,
		Status:     model.RuleRecordStatusPending,
	}
	if err := tx.Create(&rec).Error; err != nil {
		return fmt.Errorf("创建 pending rule record (slug=%s, upgrade): %w", item.Slug, err)
	}

	if err := tx.Model(&model.LocalInstanceRule{}).
		Where("id = ?", localRuleID).
		Updates(map[string]any{
			"version":        item.Version,
			"display_name":   item.DisplayName,
			"rule_type":      item.RuleType,
			"source":         item.Source,
			"install_status": model.LocalSkillInstallStatusDistributing,
			"last_seen_at":   &now,
		}).Error; err != nil {
		return fmt.Errorf("update local_instance_rules (id=%d): %w", localRuleID, err)
	}

	return nil
}
func updatePendingRecordAndLocalSkill(tx *gorm.DB, instanceID, localSkillID uint,
	item model.SkillCatalogItem, now time.Time) error {

	// 1. 创建 skill_distribution_task（status=completed，下发动作已完成）
	task := model.SkillDistributionTask{
		SkillID: item.SkillID,
		Version: item.Version,
		Source:  item.Source,
		Slug:    item.Slug,
		Total:   1,
		Status:  model.TaskStatusCompleted,
		Type:    model.TaskTypeDistribute,
		BatchID: localScopeBatchID(localSkillID),
	}
	if err := tx.Create(&task).Error; err != nil {
		return fmt.Errorf("创建 distribution task (slug=%s, upgrade): %w", item.Slug, err)
	}

	// 2. 写 skill_distribution_record (pending)
	rec := model.SkillDistributionRecord{
		InstanceID: instanceID,
		TaskID:     task.ID,
		SkillID:    item.SkillID,
		Type:       model.TaskTypeDistribute,
		Version:    item.Version,
		Status:     model.RecordStatusPending,
	}
	if err := tx.Create(&rec).Error; err != nil {
		return fmt.Errorf("创建 pending record (slug=%s, upgrade): %w", item.Slug, err)
	}

	// 3. update local_instance_skills (install_status=distributing, version=item.Version)
	if err := tx.Model(&model.LocalInstanceSkill{}).
		Where("id = ?", localSkillID).
		Updates(map[string]any{
			"version":        item.Version,
			"display_name":   item.DisplayName,
			"source":         item.Source,
			"install_status": model.LocalSkillInstallStatusDistributing,
			"last_seen_at":   &now,
		}).Error; err != nil {
		return fmt.Errorf("update local_instance_skills (id=%d): %w", localSkillID, err)
	}

	return nil
}

// ---- report 二期处理 -----------------------------------------------------

// processReportUserLevelAndWorkspaces 处理 report 的二期部分：用户级 + 项目级资源。
// 在一期事务内调用，tx 为已开启的事务。
//
// 返回 user_level_synced、project_synced、rules_synced 计数。
