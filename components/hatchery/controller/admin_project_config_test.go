package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hatchery/controller/usergroup"
	"hatchery/model"
)

func TestHandleAdminProjectConfigOverview_CorePaths(t *testing.T) {
	setupProjectsTestDB(t)
	project := model.Project{Name: "配置概览项目"}
	if err := model.DB(t.Context()).Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	w := httptest.NewRecorder()
	HandleAdminProjectConfigOverview(w, adminGet("/admin/project-config?project_ids="+uintStr(project.ID)))
	resp := decodeOK(t, w)
	results, ok := resp["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("results=%#v", resp["results"])
	}

	for _, tc := range []struct {
		name   string
		req    *http.Request
		status int
	}{
		{"method", adminJSONPost("/admin/project-config", `{}`), http.StatusMethodNotAllowed},
		{"missing IDs", adminGet("/admin/project-config"), http.StatusBadRequest},
		{"invalid IDs", adminGet("/admin/project-config?project_ids=bad"), http.StatusBadRequest},
		{"unknown project", adminGet("/admin/project-config?project_ids=99999"), http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			HandleAdminProjectConfigOverview(w, tc.req)
			if w.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", w.Code, tc.status, w.Body.String())
			}
		})
	}
}

func TestHandleAdminTargetInstances_CorePaths(t *testing.T) {
	setupProjectsTestDB(t)
	user, group := createTestUserAndGroup(t, model.DB(t.Context()))
	project := model.Project{Name: "目标实例项目"}
	if err := model.DB(t.Context()).Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	instance := createTestLocalInstance(t, model.DB(t.Context()), user.ID)
	bindings := []model.LocalAgentScopeBinding{
		{InstanceID: instance.ID, Scope: model.LocalAgentScopeUser, ScopeKey: "", GroupID: group.ID},
		{InstanceID: instance.ID, Scope: model.LocalAgentScopeWorkspace, ScopeKey: "/repo", ProjectID: project.ID},
	}
	if err := model.DB(t.Context()).Create(&bindings).Error; err != nil {
		t.Fatalf("create bindings: %v", err)
	}

	for _, tc := range []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		path    string
	}{
		{"project", HandleAdminProjectInstances, "/admin/project-instances?project_id=" + uintStr(project.ID) + "&page=3&page_size=1"},
		{"group", HandleAdminUserGroupInstances, "/admin/group-instances?id=" + uintStr(group.ID)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.handler(w, adminGet(tc.path))
			resp := decodeOK(t, w)
			if resp["total"].(float64) != 1 {
				t.Fatalf("total=%v", resp["total"])
			}
		})
	}

	for _, tc := range []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		req     *http.Request
		status  int
	}{
		{"project method", HandleAdminProjectInstances, adminJSONPost("/admin/project-instances", `{}`), http.StatusMethodNotAllowed},
		{"project ID", HandleAdminProjectInstances, adminGet("/admin/project-instances?project_id=bad"), http.StatusBadRequest},
		{"group method", HandleAdminUserGroupInstances, adminJSONPost("/admin/group-instances", `{}`), http.StatusMethodNotAllowed},
		{"group ID", HandleAdminUserGroupInstances, adminGet("/admin/group-instances?id=0"), http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.handler(w, tc.req)
			if w.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", w.Code, tc.status, w.Body.String())
			}
		})
	}
}

func TestProjectConfigCategoriesIncludesGlobalAndProjectVisibleAssets(t *testing.T) {
	db := setupPhase2DB(t)
	project := model.Project{Name: "项目总览"}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	globalSkill := model.Skill{Slug: "global-skill", Name: "全员技能", Version: "1.0.0", VisibilityType: model.VisibilityAll}
	projectSkill := model.Skill{Slug: "project-skill", Name: "项目技能", Version: "1.0.0", VisibilityType: model.VisibilityGroup}
	globalRule := model.EnterpriseRule{Slug: "global-rule", Name: "全员规范", Version: "1.0.0", VisibilityType: model.VisibilityAll}
	if err := db.Create(&globalSkill).Error; err != nil {
		t.Fatalf("create global skill: %v", err)
	}
	if err := db.Create(&projectSkill).Error; err != nil {
		t.Fatalf("create project skill: %v", err)
	}
	if err := db.Create(&globalRule).Error; err != nil {
		t.Fatalf("create global rule: %v", err)
	}
	if err := db.Create(&model.ProjectConfigBinding{ProjectID: project.ID, ConfigType: model.ProjectConfigTypeSkill, ConfigKey: projectSkill.Slug}).Error; err != nil {
		t.Fatalf("bind project skill: %v", err)
	}

	categories, err := projectConfigCategories(db, project.ID)
	if err != nil {
		t.Fatalf("projectConfigCategories: %v", err)
	}
	if len(categories) != 1 || categories[0].Key != usergroup.CategoryKeyAgentTool || categories[0].Description != "企业技能与企业规范" {
		t.Fatalf("categories=%#v", categories)
	}
	entries := categories[0].Entries
	if len(entries) != 3 {
		t.Fatalf("技能总览应包含全员可见和项目可见资源，实际=%#v", entries)
	}
	if entries[0].Label != globalSkill.Name || entries[0].Source.Type != usergroup.SourceAllUsers {
		t.Fatalf("全员可见技能条目=%#v", entries[0])
	}
	if entries[1].Label != projectSkill.Name || entries[1].Source.Type != usergroup.SourceLocal {
		t.Fatalf("项目范围技能条目=%#v", entries[1])
	}
	if entries[2].Label != globalRule.Name || entries[2].SubLabel != "企业规范" || entries[2].Source.Type != usergroup.SourceAllUsers {
		t.Fatalf("全员可见规范条目=%#v", entries[2])
	}
}

func TestWriteLocalAgentTargetInstancesEnrichesDisplayFields(t *testing.T) {
	db := setupPhase2DB(t)
	user, group := createTestUserAndGroup(t, db)
	group.FullPath = "研发中心/平台组/G1"
	if err := db.Save(group).Error; err != nil {
		t.Fatalf("update group full_path: %v", err)
	}
	project := model.Project{Name: "实例项目"}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	instance := createTestLocalInstance(t, db, user.ID)
	now := time.Now()
	if err := db.Create(&model.LocalInstanceInfo{InstanceID: instance.ID, LastReportAt: &now}).Error; err != nil {
		t.Fatalf("create local info: %v", err)
	}
	bindings := []model.LocalAgentScopeBinding{
		{InstanceID: instance.ID, Scope: model.LocalAgentScopeUser, ScopeKey: "user", GroupID: group.ID},
		{InstanceID: instance.ID, Scope: model.LocalAgentScopeWorkspace, ScopeKey: "/repo", ProjectID: project.ID},
	}
	if err := db.Create(&bindings).Error; err != nil {
		t.Fatalf("create scope bindings: %v", err)
	}

	assertTargetInstanceDisplay(t, assetTargetGroup, group.ID, "bound_user_levels", "group_name", group.Name, group.FullPath)
	assertTargetInstanceDisplay(t, assetTargetProject, project.ID, "bound_workspaces", "project_name", project.Name, "")
}

func TestHandleAdminUserGroupInstancesIncludesDescendants(t *testing.T) {
	db := setupPhase2DB(t)
	user, parent := createTestUserAndGroup(t, db)
	parent.FullPath = "研发中心"
	if err := db.Save(parent).Error; err != nil {
		t.Fatalf("update parent: %v", err)
	}
	child := model.UserGroup{Name: "平台组", ParentID: parent.ID, Depth: 1, FullPath: "研发中心/平台组"}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}
	closures := []model.GroupClosure{
		{AncestorID: parent.ID, DescendantID: parent.ID, Depth: 0},
		{AncestorID: parent.ID, DescendantID: child.ID, Depth: 1},
		{AncestorID: child.ID, DescendantID: child.ID, Depth: 0},
	}
	if err := db.Create(&closures).Error; err != nil {
		t.Fatalf("create closures: %v", err)
	}
	instance := createTestLocalInstance(t, db, user.ID)
	binding := model.LocalAgentScopeBinding{InstanceID: instance.ID, Scope: model.LocalAgentScopeUser, ScopeKey: "user", GroupID: child.ID}
	if err := db.Create(&binding).Error; err != nil {
		t.Fatalf("create child binding: %v", err)
	}

	w := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/user-groups/instances?id="+uintStr(parent.ID), nil)
	writeLocalAgentTargetInstances(w, request, assetTargetGroup, parent.ID)
	resp := decodeOK(t, w)
	if resp["total"].(float64) != 1 {
		t.Fatalf("total=%v", resp["total"])
	}
	instances := resp["instances"].([]any)
	bindings := instances[0].(map[string]any)["bound_user_levels"].([]any)
	got := bindings[0].(map[string]any)
	if got["group_id"].(float64) != float64(child.ID) || got["group_name"] != child.Name || got["group_full_path"] != child.FullPath {
		t.Fatalf("descendant binding=%#v", got)
	}
}

func TestWriteLocalAgentTargetInstancesDatabaseErrors(t *testing.T) {
	for _, tc := range []struct {
		name       string
		targetType string
		drop       any
	}{
		{name: "group closure", targetType: assetTargetGroup, drop: &model.GroupClosure{}},
		{name: "scope bindings", targetType: assetTargetProject, drop: &model.LocalAgentScopeBinding{}},
		{name: "group display", targetType: assetTargetGroup, drop: &model.UserGroup{}},
		{name: "project display", targetType: assetTargetProject, drop: &model.Project{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := setupPhase2DB(t)
			if err := db.Migrator().DropTable(tc.drop); err != nil {
				t.Fatalf("drop table: %v", err)
			}
			w := httptest.NewRecorder()
			writeLocalAgentTargetInstances(w, httptest.NewRequest(http.MethodGet, "/admin/target-instances", nil), tc.targetType, 1)
			if w.Code != http.StatusInternalServerError {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
	t.Run("usernames", func(t *testing.T) {
		db := setupPhase2DB(t)
		user, _ := createTestUserAndGroup(t, db)
		project := model.Project{Name: "用户名查询异常项目"}
		if err := db.Create(&project).Error; err != nil {
			t.Fatalf("create project: %v", err)
		}
		instance := createTestLocalInstance(t, db, user.ID)
		binding := model.LocalAgentScopeBinding{InstanceID: instance.ID, Scope: model.LocalAgentScopeWorkspace, ScopeKey: "/repo", ProjectID: project.ID}
		if err := db.Create(&binding).Error; err != nil {
			t.Fatalf("create binding: %v", err)
		}
		if err := db.Migrator().DropTable(&model.User{}); err != nil {
			t.Fatalf("drop users: %v", err)
		}
		w := httptest.NewRecorder()
		writeLocalAgentTargetInstances(w, httptest.NewRequest(http.MethodGet, "/admin/target-instances", nil), assetTargetProject, project.ID)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestLocalAgentTargetDisplaysAndUsernamesDatabaseErrors(t *testing.T) {
	if usernames, err := localAgentUsernames(t.Context(), nil); err != nil || len(usernames) != 0 {
		t.Fatalf("empty usernames=%#v err=%v", usernames, err)
	}
	t.Run("group display", func(t *testing.T) {
		db := setupPhase2DB(t)
		if err := db.Migrator().DropTable(&model.UserGroup{}); err != nil {
			t.Fatalf("drop groups: %v", err)
		}
		if _, err := localAgentTargetDisplays(t.Context(), assetTargetGroup, []uint{1}); err == nil {
			t.Fatal("expected group query error")
		}
	})
	t.Run("usernames", func(t *testing.T) {
		db := setupPhase2DB(t)
		if err := db.Migrator().DropTable(&model.User{}); err != nil {
			t.Fatalf("drop users: %v", err)
		}
		items := []model.LocalAgentTargetInstance{{Instance: model.Instance{UserID: 1}}}
		if _, err := localAgentUsernames(t.Context(), items); err == nil {
			t.Fatal("expected user query error")
		}
	})
}

func TestLocalAgentTargetInstanceResponseMissingDisplay(t *testing.T) {
	for _, tc := range []struct {
		name        string
		targetType  string
		binding     model.LocalAgentScopeBinding
		bindingName string
	}{
		{name: "group", targetType: assetTargetGroup, binding: model.LocalAgentScopeBinding{GroupID: 99}, bindingName: "bound_user_levels"},
		{name: "project", targetType: assetTargetProject, binding: model.LocalAgentScopeBinding{ProjectID: 88}, bindingName: "bound_workspaces"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := model.LocalAgentTargetInstance{
				Instance:      model.Instance{},
				ScopeBindings: []model.LocalAgentScopeBinding{tc.binding},
			}
			response := localAgentTargetInstanceResponse(
				httptest.NewRequest(http.MethodGet, "/admin/target-instances", nil),
				item, tc.bindingName, tc.targetType, map[uint]localAgentTargetDisplay{}, "",
			)
			bindings := response[tc.bindingName].([]localAgentScopeBindingView)
			if len(bindings) != 1 || bindings[0].GroupName != "" || bindings[0].GroupFullPath != "" || bindings[0].ProjectName != "" {
				t.Fatalf("bindings=%#v", bindings)
			}
		})
	}
}

func assertTargetInstanceDisplay(t *testing.T, targetType string, targetID uint, bindingField, nameField, wantName, wantFullPath string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/admin/target-instances", nil)
	writeLocalAgentTargetInstances(recorder, request, targetType, targetID)
	var response struct {
		Instances []map[string]any `json:"instances"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Instances) != 1 {
		t.Fatalf("instances=%#v", response.Instances)
	}
	instance, _ := response.Instances[0]["instance"].(map[string]any)
	if instance["status"] != model.StatusRunning || instance["username"] != "testuser" {
		t.Fatalf("instance display fields=%#v", instance)
	}
	bindings, _ := response.Instances[0][bindingField].([]any)
	if len(bindings) != 1 || bindings[0].(map[string]any)[nameField] != wantName {
		t.Fatalf("%s=%#v", bindingField, bindings)
	}
	if wantFullPath != "" && bindings[0].(map[string]any)["group_full_path"] != wantFullPath {
		t.Fatalf("%s group_full_path=%#v", bindingField, bindings)
	}
}
