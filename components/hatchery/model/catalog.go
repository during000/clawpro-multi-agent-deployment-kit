package model

import (
	"context"

	"gorm.io/gorm"
)

// SkillCatalogItem 表示可下发的企业技能版本。
type SkillCatalogItem struct {
	SkillID     uint
	Slug        string
	Version     string
	DisplayName string
	COSZipKey   string
	Source      string // enterprise / public（本期只用 enterprise，rule 留接口预留）
}

// ListSkillsByGroupWithDB 返回分组及其祖先直接绑定的技能最新版本。
// 可见范围只决定候选集，不代表应下发；本函数只读取 asset_skill 绑定。
func ListSkillsByGroupWithDB(tx *gorm.DB, ctx context.Context, groupID uint) ([]SkillCatalogItem, error) {
	if groupID == 0 {
		return nil, nil
	}
	boundSlugs, err := groupAssetSlugsWithDB(tx, groupID, AssetBindingTypeSkill)
	if err != nil || len(boundSlugs) == 0 {
		return nil, err
	}

	var items []SkillCatalogItem
	err = tx.Model(&Skill{}).
		Select("skills.id AS skill_id, skills.slug, skills.version, skills.name AS display_name, skills.cos_zip_key, 'enterprise' AS source").
		Where("skills.id IN (?)", LatestVersionSkillIDs(ctx)).
		Where("skills.slug IN ?", boundSlugs).
		Where("skills.status = ?", SkillStatusPublished).
		Scan(&items).Error
	return items, err
}

// ListSkillsByProjectWithDB 返回项目直接绑定的企业技能，不做项目继承。
func ListSkillsByProjectWithDB(tx *gorm.DB, ctx context.Context, projectID uint) ([]SkillCatalogItem, error) {
	if projectID == 0 {
		return nil, nil
	}
	boundSlugs := tx.Model(&ProjectConfigBinding{}).
		Select("config_key").
		Where("project_id = ? AND config_type = ?", projectID, AssetBindingTypeSkill)
	var items []SkillCatalogItem
	err := tx.Model(&Skill{}).
		Select("skills.id AS skill_id, skills.slug, skills.version, skills.name AS display_name, skills.cos_zip_key, 'enterprise' AS source").
		Where("skills.id IN (?)", LatestVersionSkillIDs(ctx)).
		Where("skills.slug IN (?)", boundSlugs).
		Where("skills.status = ?", SkillStatusPublished).
		Scan(&items).Error
	return items, err
}

// ListInstalledSkillsWithDB 返回某 (scope, instanceID, workspacePath) 下已装的 skill 快照。
// diffAndQueue 用它和 catalog 做差集。调用方传入事务句柄 tx。
func ListInstalledSkillsWithDB(tx *gorm.DB, scope string, instanceID uint, workspacePath string) ([]LocalInstanceSkill, error) {
	var rows []LocalInstanceSkill
	err := tx.
		Where("scope = ? AND instance_id = ? AND workspace_path = ?", scope, instanceID, workspacePath).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ---- Rule catalog ----

// RuleCatalogItem 表示可下发的企业规范版本。
type RuleCatalogItem struct {
	RuleID      uint
	Slug        string
	Version     string
	DisplayName string
	COSKey      string
	RuleType    string
	Source      string
}

// ListRulesByGroupWithDB 返回分组及其祖先直接绑定的规范最新版本。
// 可见范围只决定候选集，不代表应下发；本函数只读取 asset_rule 绑定。
func ListRulesByGroupWithDB(tx *gorm.DB, ctx context.Context, groupID uint) ([]RuleCatalogItem, error) {
	if groupID == 0 {
		return nil, nil
	}
	boundSlugs, err := groupAssetSlugsWithDB(tx, groupID, AssetBindingTypeRule)
	if err != nil || len(boundSlugs) == 0 {
		return nil, err
	}

	var items []RuleCatalogItem
	err = tx.Model(&EnterpriseRule{}).
		Select("enterprise_rules.id AS rule_id, enterprise_rules.slug, enterprise_rules.version, enterprise_rules.name AS display_name, enterprise_rules.cos_key, enterprise_rules.type AS rule_type, 'enterprise' AS source").
		Where("enterprise_rules.id IN (?)", LatestVersionRuleIDs(ctx)).
		Where("enterprise_rules.slug IN ?", boundSlugs).
		Scan(&items).Error
	return items, err
}

// ListRulesByProjectWithDB 返回项目直接绑定的企业规范，不做项目继承。
func ListRulesByProjectWithDB(tx *gorm.DB, ctx context.Context, projectID uint) ([]RuleCatalogItem, error) {
	if projectID == 0 {
		return nil, nil
	}
	boundSlugs := tx.Model(&ProjectConfigBinding{}).
		Select("config_key").
		Where("project_id = ? AND config_type = ?", projectID, AssetBindingTypeRule)
	var items []RuleCatalogItem
	err := tx.Model(&EnterpriseRule{}).
		Select("enterprise_rules.id AS rule_id, enterprise_rules.slug, enterprise_rules.version, enterprise_rules.name AS display_name, enterprise_rules.cos_key, enterprise_rules.type AS rule_type, 'enterprise' AS source").
		Where("enterprise_rules.id IN (?)", LatestVersionRuleIDs(ctx)).
		Where("enterprise_rules.slug IN (?)", boundSlugs).
		Scan(&items).Error
	return items, err
}

// ListInstalledRulesWithDB 返回某 (scope, instanceID, workspacePath) 下已装的 rule 快照。
func ListInstalledRulesWithDB(tx *gorm.DB, scope string, instanceID uint, workspacePath string) ([]LocalInstanceRule, error) {
	var rows []LocalInstanceRule
	err := tx.
		Where("scope = ? AND instance_id = ? AND workspace_path = ?", scope, instanceID, workspacePath).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// AssetCatalog 汇总某个目标可下发的资产。
// 每种资产保留独立的强类型结构，后续接入新资产时只需新增对应字段，避免以公共字段压缩资产特有信息。
type AssetCatalog struct {
	Skills []SkillCatalogItem
	Rules  []RuleCatalogItem
}

// ListAssetCatalogByGroup 返回分组（含祖先继承）的绑定资产。
func ListAssetCatalogByGroup(ctx context.Context, groupID uint) (AssetCatalog, error) {
	return ListAssetCatalogByGroupWithDB(DB(ctx), ctx, groupID)
}

// ListAssetCatalogByGroupWithDB 返回分组（含祖先继承）的绑定资产。
func ListAssetCatalogByGroupWithDB(tx *gorm.DB, ctx context.Context, groupID uint) (AssetCatalog, error) {
	skills, err := ListSkillsByGroupWithDB(tx, ctx, groupID)
	if err != nil {
		return AssetCatalog{}, err
	}
	rules, err := ListRulesByGroupWithDB(tx, ctx, groupID)
	if err != nil {
		return AssetCatalog{}, err
	}
	return AssetCatalog{Skills: skills, Rules: rules}, nil
}

// ListAssetCatalogByProject 返回项目直接绑定的资产；项目没有继承关系。
func ListAssetCatalogByProject(ctx context.Context, projectID uint) (AssetCatalog, error) {
	return ListAssetCatalogByProjectWithDB(DB(ctx), ctx, projectID)
}

// ListAssetCatalogByProjectWithDB 返回项目直接绑定的资产；项目没有继承关系。
func ListAssetCatalogByProjectWithDB(tx *gorm.DB, ctx context.Context, projectID uint) (AssetCatalog, error) {
	skills, err := ListSkillsByProjectWithDB(tx, ctx, projectID)
	if err != nil {
		return AssetCatalog{}, err
	}
	rules, err := ListRulesByProjectWithDB(tx, ctx, projectID)
	if err != nil {
		return AssetCatalog{}, err
	}
	return AssetCatalog{Skills: skills, Rules: rules}, nil
}

// AssetBindingTarget 是资产被直接绑定到的分组或项目。
// 分组继承在获取目标资产时计算，反查始终只返回直接绑定目标。
type AssetBindingTarget struct {
	TargetType string `json:"target_type"`
	TargetID   uint   `json:"target_id"`
	Name       string `json:"name"`
	FullPath   string `json:"full_path,omitempty"`
}

// ListAssetBindingTargets 返回一个技能或规范的所有直接绑定目标。
func ListAssetBindingTargets(ctx context.Context, assetType, slug string) ([]AssetBindingTarget, error) {
	return ListAssetBindingTargetsWithDB(DB(ctx), assetType, slug)
}

// ListAssetBindingTargetsWithDB 是 ListAssetBindingTargets 的事务安全版本。
func ListAssetBindingTargetsWithDB(tx *gorm.DB, assetType, slug string) ([]AssetBindingTarget, error) {
	configType, ok := AssetBindingConfigType(assetType)
	if !ok || slug == "" {
		return []AssetBindingTarget{}, nil
	}
	var groupBindings []GroupConfigBinding
	if err := tx.Where("config_type = ? AND config_key = ?", configType, slug).Find(&groupBindings).Error; err != nil {
		return nil, err
	}
	var projectBindings []ProjectConfigBinding
	if err := tx.Where("config_type = ? AND config_key = ?", configType, slug).Find(&projectBindings).Error; err != nil {
		return nil, err
	}
	return collectAssetBindingTargets(tx, groupBindings, projectBindings)
}

// RemoveAssetBindingsForTargets 删除指定目标上某资产的直接绑定，并返回实际被删除的目标。
// 应用范围缩小时，只有原本显式选择过该资产的目标才需要产生资产版本记录。
func RemoveAssetBindingsForTargets(tx *gorm.DB, assetType, slug string, projectIDs, groupIDs []uint) ([]AssetBindingTarget, error) {
	configType, ok := AssetBindingConfigType(assetType)
	if !ok || slug == "" {
		return []AssetBindingTarget{}, nil
	}

	var affected []AssetBindingTarget
	if err := removeProjectAssetBindings(tx, configType, slug, projectIDs, &affected); err != nil {
		return nil, err
	}
	if err := removeGroupAssetBindings(tx, configType, slug, groupIDs, &affected); err != nil {
		return nil, err
	}
	return affected, nil
}

func removeProjectAssetBindings(tx *gorm.DB, configType, slug string, projectIDs []uint, affected *[]AssetBindingTarget) error {
	if len(projectIDs) == 0 {
		return nil
	}
	var ids []uint
	if err := tx.Model(&ProjectConfigBinding{}).
		Where("config_type = ? AND config_key = ? AND project_id IN ?", configType, slug, projectIDs).
		Pluck("project_id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if err := tx.Where("config_type = ? AND config_key = ? AND project_id IN ?", configType, slug, ids).
		Delete(&ProjectConfigBinding{}).Error; err != nil {
		return err
	}
	for _, id := range ids {
		*affected = append(*affected, AssetBindingTarget{TargetType: TargetTypeProject, TargetID: id})
	}
	return nil
}

func removeGroupAssetBindings(tx *gorm.DB, configType, slug string, groupIDs []uint, affected *[]AssetBindingTarget) error {
	if len(groupIDs) == 0 {
		return nil
	}
	var ids []uint
	if err := tx.Model(&GroupConfigBinding{}).
		Where("config_type = ? AND config_key = ? AND group_id IN ?", configType, slug, groupIDs).
		Pluck("group_id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if err := tx.Where("config_type = ? AND config_key = ? AND group_id IN ?", configType, slug, ids).
		Delete(&GroupConfigBinding{}).Error; err != nil {
		return err
	}
	for _, id := range ids {
		*affected = append(*affected, AssetBindingTarget{TargetType: TargetTypeGroup, TargetID: id})
	}
	return nil
}

func groupAssetSlugsWithDB(tx *gorm.DB, groupID uint, configType string) ([]string, error) {
	groupIDs := []uint{groupID}
	var ancestors []GroupClosure
	if err := tx.Where("descendant_id = ? AND ancestor_id <> ?", groupID, groupID).Order("depth ASC").Find(&ancestors).Error; err != nil {
		return nil, err
	}
	for _, ancestor := range ancestors {
		groupIDs = append(groupIDs, ancestor.AncestorID)
	}
	var slugs []string
	err := tx.Model(&GroupConfigBinding{}).Where("config_type = ? AND group_id IN ?", configType, groupIDs).
		Distinct().Pluck("config_key", &slugs).Error
	return slugs, err
}

func collectAssetBindingTargets(tx *gorm.DB, groupBindings []GroupConfigBinding, projectBindings []ProjectConfigBinding) ([]AssetBindingTarget, error) {
	groupIDs := make([]uint, 0, len(groupBindings))
	for _, binding := range groupBindings {
		groupIDs = append(groupIDs, binding.GroupID)
	}
	projectIDs := make([]uint, 0, len(projectBindings))
	for _, binding := range projectBindings {
		projectIDs = append(projectIDs, binding.ProjectID)
	}
	var groups []UserGroup
	if len(groupIDs) > 0 {
		if err := tx.Where("id IN ?", groupIDs).Order("full_path ASC, id ASC").Find(&groups).Error; err != nil {
			return nil, err
		}
	}
	var projects []Project
	if len(projectIDs) > 0 {
		if err := tx.Where("id IN ?", projectIDs).Order("name ASC, id ASC").Find(&projects).Error; err != nil {
			return nil, err
		}
	}
	targets := make([]AssetBindingTarget, 0, len(groups)+len(projects))
	for _, group := range groups {
		targets = append(targets, AssetBindingTarget{TargetType: "group", TargetID: group.ID, Name: group.Name, FullPath: group.FullPath})
	}
	for _, project := range projects {
		targets = append(targets, AssetBindingTarget{TargetType: "project", TargetID: project.ID, Name: project.Name})
	}
	return targets, nil
}
