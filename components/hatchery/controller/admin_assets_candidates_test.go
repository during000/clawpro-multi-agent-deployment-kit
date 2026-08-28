package controller

import (
	"context"
	"net/http/httptest"
	"testing"

	"hatchery/controller/usergroup"
	"hatchery/model"
)

func candidateAssetsBySlug(t *testing.T, response map[string]any) map[string]map[string]any {
	t.Helper()
	result := make(map[string]map[string]any)
	for _, raw := range response["assets"].([]any) {
		item := raw.(map[string]any)
		result[item["slug"].(string)] = item
	}
	return result
}

func TestHandleAdminAssetCandidates_ProjectSources(t *testing.T) {
	setupProjectsTestDB(t)
	db := model.DB(context.Background())
	project := model.Project{Name: "候选资产项目"}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	skills := []model.Skill{
		{Slug: "all-skill", Name: "All Skill", Version: "1.0.0", VersionMajor: 1, VisibilityType: model.VisibilityAll},
		{Slug: "local-skill", Name: "Local Skill", Version: "1.0.0", VersionMajor: 1, VisibilityType: model.VisibilityGroup},
	}
	if err := db.Create(&skills).Error; err != nil {
		t.Fatalf("create skills: %v", err)
	}
	allSkill, localSkill := skills[0], skills[1]
	if err := db.Create(&[]model.ProjectConfigBinding{
		{ProjectID: project.ID, ConfigType: model.ProjectConfigTypeSkill, ConfigKey: localSkill.Slug},
		{ProjectID: project.ID, ConfigType: model.AssetBindingTypeSkill, ConfigKey: localSkill.Slug},
	}).Error; err != nil {
		t.Fatalf("create project bindings: %v", err)
	}

	w := httptest.NewRecorder()
	HandleAdminAssetCandidates(w, adminGet("/admin/assets/candidates?target_type=project&target_id="+uintStr(project.ID)))
	assets := candidateAssetsBySlug(t, decodeOK(t, w))

	if assets[allSkill.Slug]["selected"] != false {
		t.Fatalf("all_users candidate must explicitly return selected=false: %#v", assets[allSkill.Slug])
	}
	if assets[allSkill.Slug]["source"].(map[string]any)["type"] != string(usergroup.SourceAllUsers) {
		t.Fatalf("unexpected all_users source: %#v", assets[allSkill.Slug]["source"])
	}
	if assets[localSkill.Slug]["selected"] != true {
		t.Fatalf("local candidate must return selected=true: %#v", assets[localSkill.Slug])
	}
	if assets[localSkill.Slug]["source"].(map[string]any)["type"] != string(usergroup.SourceLocal) {
		t.Fatalf("unexpected local source: %#v", assets[localSkill.Slug]["source"])
	}
}

func TestHandleAdminAssetCandidates_GroupSources(t *testing.T) {
	setupProjectsTestDB(t)
	db := model.DB(context.Background())
	parent := model.UserGroup{Name: "研发中心", FullPath: "研发中心", Source: model.GroupSourceManual}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("create parent group: %v", err)
	}
	child := model.UserGroup{Name: "后端组", FullPath: "研发中心/后端组", ParentID: parent.ID, Source: model.GroupSourceManual}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create child group: %v", err)
	}
	if err := db.Create(&[]model.GroupClosure{
		{AncestorID: parent.ID, DescendantID: parent.ID, Depth: 0},
		{AncestorID: parent.ID, DescendantID: child.ID, Depth: 1},
		{AncestorID: child.ID, DescendantID: child.ID, Depth: 0},
	}).Error; err != nil {
		t.Fatalf("create closures: %v", err)
	}
	skills := []model.Skill{
		{Slug: "group-all-skill", Name: "Group All Skill", Version: "1.0.0", VersionMajor: 1, VisibilityType: model.VisibilityAll},
		{Slug: "group-local-skill", Name: "Group Local Skill", Version: "1.0.0", VersionMajor: 1, VisibilityType: model.VisibilityGroup},
	}
	inheritedRule := model.EnterpriseRule{Slug: "inherited-rule", Name: "Inherited Rule", Type: "prompt", Version: "1.0.0", VersionMajor: 1, VisibilityType: model.VisibilityGroup}
	if err := db.Create(&skills).Error; err != nil {
		t.Fatalf("create skills: %v", err)
	}
	allSkill, localSkill := skills[0], skills[1]
	if err := db.Create(&inheritedRule).Error; err != nil {
		t.Fatalf("create rule: %v", err)
	}
	if err := db.Create(&model.SkillVisibilityGroup{SkillID: localSkill.ID, GroupID: child.ID}).Error; err != nil {
		t.Fatalf("create local visibility: %v", err)
	}
	if err := db.Create(&model.RuleVisibilityGroup{RuleID: inheritedRule.ID, GroupID: parent.ID}).Error; err != nil {
		t.Fatalf("create inherited visibility: %v", err)
	}

	w := httptest.NewRecorder()
	HandleAdminAssetCandidates(w, adminGet("/admin/assets/candidates?target_type=group&target_id="+uintStr(child.ID)))
	assets := candidateAssetsBySlug(t, decodeOK(t, w))

	if assets[allSkill.Slug]["source"].(map[string]any)["type"] != string(usergroup.SourceAllUsers) {
		t.Fatalf("unexpected all_users source: %#v", assets[allSkill.Slug]["source"])
	}
	if assets[localSkill.Slug]["source"].(map[string]any)["type"] != string(usergroup.SourceLocal) {
		t.Fatalf("unexpected local source: %#v", assets[localSkill.Slug]["source"])
	}
	inheritedSource := assets[inheritedRule.Slug]["source"].(map[string]any)
	if inheritedSource["type"] != string(usergroup.SourceInherited) || inheritedSource["group_id"] != float64(parent.ID) || inheritedSource["full_path"] != parent.FullPath {
		t.Fatalf("unexpected inherited source: %#v", inheritedSource)
	}
}
