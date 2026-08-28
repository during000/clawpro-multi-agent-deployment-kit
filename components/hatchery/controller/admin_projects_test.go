package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"
)

// setupProjectsTestDB 初始化项目管理测试 DB（复用 setupPhase2DB 的表迁移 + 注入 AdminToken）。
func setupProjectsTestDB(t *testing.T) {
	t.Helper()
	setupPhase2DB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() {
		AdminToken = origToken
	})
}

func adminGet(path string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

func decodeOK(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	return resp
}

// ── HandleAdminProjects (列表) ──────────────────────────────────────────────

func TestHandleAdminProjects_Empty(t *testing.T) {
	setupProjectsTestDB(t)
	w := httptest.NewRecorder()
	HandleAdminProjects(w, adminGet("/admin/projects"))
	resp := decodeOK(t, w)
	if resp["total"].(float64) != 0 {
		t.Errorf("期望 total=0，实际=%v", resp["total"])
	}
}

func TestHandleAdminProjects_WithQuery(t *testing.T) {
	setupProjectsTestDB(t)
	ctx := context.Background()
	model.DB(ctx).Create(&model.Project{Name: "项目A", Description: ""})
	model.DB(ctx).Create(&model.Project{Name: "项目B", Description: ""})

	w := httptest.NewRecorder()
	HandleAdminProjects(w, adminGet("/admin/projects?q=项目A"))
	resp := decodeOK(t, w)
	if resp["total"].(float64) != 1 {
		t.Errorf("期望 total=1，实际=%v", resp["total"])
	}
}

func TestHandleAdminProjects_WithCounts(t *testing.T) {
	setupProjectsTestDB(t)
	ctx := context.Background()
	model.DB(ctx).Create(&model.Project{Name: "项目C"})

	w := httptest.NewRecorder()
	HandleAdminProjects(w, adminGet("/admin/projects?with_counts=true"))
	resp := decodeOK(t, w)
	projects := resp["projects"].([]any)
	if len(projects) != 1 {
		t.Fatalf("期望 1 个项目，实际=%d", len(projects))
	}
	item := projects[0].(map[string]any)
	if item["member_count"] == nil {
		t.Error("期望 member_count 非空")
	}
}

func TestHandleAdminProjects_WithoutCountsReturnsAll(t *testing.T) {
	setupProjectsTestDB(t)
	for _, name := range []string{"项目 D", "项目 E"} {
		if err := model.DB(t.Context()).Create(&model.Project{Name: name}).Error; err != nil {
			t.Fatalf("create project: %v", err)
		}
	}
	w := httptest.NewRecorder()
	HandleAdminProjects(w, adminGet("/admin/projects?with_counts=false&page=9&page_size=1"))
	resp := decodeOK(t, w)
	if len(resp["projects"].([]any)) != 2 || resp["page"].(float64) != 1 || resp["page_size"].(float64) != 2 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

// ── HandleAdminProjectCreate ────────────────────────────────────────────────

func TestHandleAdminProjectCreate_Success(t *testing.T) {
	setupProjectsTestDB(t)
	w := httptest.NewRecorder()
	HandleAdminProjectCreate(w, adminJSONPost("/admin/projects/create", `{"name":"新项目","description":"测试"}`))
	resp := decodeOK(t, w)
	project := resp["project"].(map[string]any)
	if project["name"] != "新项目" {
		t.Errorf("期望 name=新项目，实际=%v", project["name"])
	}
}

func TestHandleAdminProjectCreate_InvalidName(t *testing.T) {
	setupProjectsTestDB(t)
	w := httptest.NewRecorder()
	HandleAdminProjectCreate(w, adminJSONPost("/admin/projects/create", `{"name":"  ","description":""}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestHandleAdminProjectCreate_DuplicateName(t *testing.T) {
	setupProjectsTestDB(t)
	ctx := context.Background()
	model.DB(ctx).Create(&model.Project{Name: "重复项目"})

	w := httptest.NewRecorder()
	HandleAdminProjectCreate(w, adminJSONPost("/admin/projects/create", `{"name":"重复项目"}`))
	if w.Code != http.StatusConflict {
		t.Errorf("期望 409，实际=%d", w.Code)
	}
}

// ── HandleAdminProjectUpdate ────────────────────────────────────────────────

func TestHandleAdminProjectUpdate_Success(t *testing.T) {
	setupProjectsTestDB(t)
	ctx := context.Background()
	p := model.Project{Name: "旧名"}
	model.DB(ctx).Create(&p)

	w := httptest.NewRecorder()
	HandleAdminProjectUpdate(w, adminJSONPost("/admin/projects/update",
		`{"id":`+uintStr(p.ID)+`,"name":"新名","description":"  新描述  "}`))
	resp := decodeOK(t, w)
	project := resp["project"].(map[string]any)
	if project["name"] != "新名" {
		t.Errorf("期望 name=新名，实际=%v", project["name"])
	}
	if project["description"] != "新描述" {
		t.Errorf("期望 description=新描述，实际=%v", project["description"])
	}
}

func TestHandleAdminProjectUpdate_InvalidName(t *testing.T) {
	setupProjectsTestDB(t)
	p := model.Project{Name: "有效旧名"}
	if err := model.DB(t.Context()).Create(&p).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	w := httptest.NewRecorder()
	HandleAdminProjectUpdate(w, adminJSONPost("/admin/projects/update",
		`{"id":`+uintStr(p.ID)+`,"name":"   "}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleAdminProjectUpdate_NotFound(t *testing.T) {
	setupProjectsTestDB(t)
	w := httptest.NewRecorder()
	HandleAdminProjectUpdate(w, adminJSONPost("/admin/projects/update", `{"id":99999,"name":"x"}`))
	if w.Code != http.StatusNotFound {
		t.Errorf("期望 404，实际=%d", w.Code)
	}
}

func TestHandleAdminProjectUpdate_NoFields(t *testing.T) {
	setupProjectsTestDB(t)
	w := httptest.NewRecorder()
	HandleAdminProjectUpdate(w, adminJSONPost("/admin/projects/update", `{"id":1}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

// ── HandleAdminProjectDeleteImpact ──────────────────────────────────────────

func TestHandleAdminProjectDeleteImpact_CanDelete(t *testing.T) {
	setupProjectsTestDB(t)
	ctx := context.Background()
	p := model.Project{Name: "可删项目"}
	model.DB(ctx).Create(&p)

	w := httptest.NewRecorder()
	HandleAdminProjectDeleteImpact(w, adminGet("/admin/projects/delete-impact?project_ids="+uintStr(p.ID)))
	resp := decodeOK(t, w)
	results := resp["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("期望 1 条结果，实际=%d", len(results))
	}
	item := results[0].(map[string]any)
	if item["can_delete"] != true {
		t.Errorf("期望 can_delete=true，实际=%v", item["can_delete"])
	}
}

func TestHandleAdminProjectDeleteImpact_HasBindings(t *testing.T) {
	setupProjectsTestDB(t)
	ctx := context.Background()
	p := model.Project{Name: "有绑定项目"}
	model.DB(ctx).Create(&p)
	model.DB(ctx).Create(&model.ProjectConfigBinding{ProjectID: p.ID, ConfigType: model.ProjectConfigTypeSkill, ConfigKey: "skill-x"})

	w := httptest.NewRecorder()
	HandleAdminProjectDeleteImpact(w, adminGet("/admin/projects/delete-impact?project_ids="+uintStr(p.ID)))
	resp := decodeOK(t, w)
	results := resp["results"].([]any)
	item := results[0].(map[string]any)
	if item["can_delete"] != false {
		t.Errorf("期望 can_delete=false，实际=%v", item["can_delete"])
	}
}

// ── HandleAdminProjectDelete ────────────────────────────────────────────────

func TestHandleAdminProjectDelete_Success(t *testing.T) {
	setupProjectsTestDB(t)
	ctx := context.Background()
	p := model.Project{Name: "待删项目"}
	model.DB(ctx).Create(&p)

	w := httptest.NewRecorder()
	HandleAdminProjectDelete(w, adminJSONPost("/admin/projects/delete", `{"project_ids":[`+uintStr(p.ID)+`]}`))
	decodeOK(t, w)

	var count int64
	model.DB(ctx).Model(&model.Project{}).Where("id = ?", p.ID).Count(&count)
	if count != 0 {
		t.Errorf("期望项目已删除，实际 count=%d", count)
	}
}

func TestHandleAdminProjectDelete_BlockedByBindings(t *testing.T) {
	setupProjectsTestDB(t)
	ctx := context.Background()
	p := model.Project{Name: "有绑定待删"}
	model.DB(ctx).Create(&p)
	model.DB(ctx).Create(&model.ProjectConfigBinding{ProjectID: p.ID, ConfigType: model.ProjectConfigTypeSkill, ConfigKey: "s1"})

	w := httptest.NewRecorder()
	HandleAdminProjectDelete(w, adminJSONPost("/admin/projects/delete", `{"project_ids":[`+uintStr(p.ID)+`]}`))
	if w.Code != http.StatusConflict {
		t.Errorf("期望 409，实际=%d", w.Code)
	}
}

// ── HandleAdminProjectMembers ───────────────────────────────────────────────

func TestHandleAdminProjectMembers_List(t *testing.T) {
	setupProjectsTestDB(t)
	ctx := context.Background()
	p := model.Project{Name: "成员项目"}
	model.DB(ctx).Create(&p)
	otherProject := model.Project{Name: "其他成员项目"}
	model.DB(ctx).Create(&otherProject)
	u := model.User{Username: "member1", Password: "x"}
	model.DB(ctx).Create(&u)
	model.DB(ctx).Create(&model.ProjectMember{ProjectID: p.ID, UserID: u.ID})
	model.DB(ctx).Create(&model.ProjectMember{ProjectID: otherProject.ID, UserID: u.ID})
	inst := model.Instance{InstanceId: "member-local-1", Name: "member-local", UserID: u.ID, Source: model.InstanceSourceLocal}
	model.DB(ctx).Create(&inst)
	model.DB(ctx).Create(&model.LocalAgentScopeBinding{
		InstanceID: inst.ID, Scope: model.LocalAgentScopeWorkspace, ScopeKey: "/member", ProjectID: p.ID,
	})

	w := httptest.NewRecorder()
	HandleAdminProjectMembers(w, adminGet("/admin/projects/members?id="+uintStr(p.ID)+"&q=member"))
	resp := decodeOK(t, w)
	members := resp["members"].([]any)
	if len(members) != 1 {
		t.Fatalf("期望 1 个成员，实际=%d", len(members))
	}
	if members[0].(map[string]any)["local_workspace_count"] != float64(1) {
		t.Fatalf("member=%#v", members[0])
	}
	projects := members[0].(map[string]any)["projects"].([]any)
	if len(projects) != 2 {
		t.Fatalf("期望成员所属的 2 个项目，实际=%d", len(projects))
	}
}

func TestHandleAdminProjectMembers_NoProjectID(t *testing.T) {
	setupProjectsTestDB(t)
	w := httptest.NewRecorder()
	HandleAdminProjectMembers(w, adminGet("/admin/projects/members"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

// ── HandleAdminProjectMembersAdd / Remove / Set ─────────────────────────────

func TestHandleAdminProjectMembersAdd_Success(t *testing.T) {
	setupProjectsTestDB(t)
	ctx := context.Background()
	p := model.Project{Name: "添加成员项目"}
	model.DB(ctx).Create(&p)
	u := model.User{Username: "add1", Password: "x"}
	model.DB(ctx).Create(&u)

	w := httptest.NewRecorder()
	HandleAdminProjectMembersAdd(w, adminJSONPost("/admin/projects/members/add",
		`{"id":`+uintStr(p.ID)+`,"user_ids":[`+uintStr(u.ID)+`]}`))
	decodeOK(t, w)

	var count int64
	model.DB(ctx).Model(&model.ProjectMember{}).Where("project_id = ?", p.ID).Count(&count)
	if count != 1 {
		t.Errorf("期望 1 个成员，实际=%d", count)
	}
}

func TestHandleAdminProjectMembersRemove_Success(t *testing.T) {
	setupProjectsTestDB(t)
	ctx := context.Background()
	p := model.Project{Name: "移除成员项目"}
	model.DB(ctx).Create(&p)
	u := model.User{Username: "rm1", Password: "x"}
	model.DB(ctx).Create(&u)
	model.DB(ctx).Create(&model.ProjectMember{ProjectID: p.ID, UserID: u.ID})

	w := httptest.NewRecorder()
	HandleAdminProjectMembersRemove(w, adminJSONPost("/admin/projects/members/remove",
		`{"id":`+uintStr(p.ID)+`,"user_ids":[`+uintStr(u.ID)+`]}`))
	decodeOK(t, w)

	var count int64
	model.DB(ctx).Model(&model.ProjectMember{}).Where("project_id = ? AND user_id = ?", p.ID, u.ID).Count(&count)
	if count != 0 {
		t.Errorf("期望成员已移除，实际 count=%d", count)
	}
}

func TestHandleAdminProjectMembersSet_Success(t *testing.T) {
	setupProjectsTestDB(t)
	ctx := context.Background()
	p := model.Project{Name: "设置成员项目"}
	model.DB(ctx).Create(&p)
	u1 := model.User{Username: "set1", Password: "x"}
	u2 := model.User{Username: "set2", Password: "x"}
	model.DB(ctx).Create(&u1)
	model.DB(ctx).Create(&u2)
	// 预置一个旧成员
	model.DB(ctx).Create(&model.ProjectMember{ProjectID: p.ID, UserID: u1.ID})

	w := httptest.NewRecorder()
	HandleAdminProjectMembersSet(w, adminJSONPost("/admin/projects/members/set",
		`{"id":`+uintStr(p.ID)+`,"user_ids":[`+uintStr(u2.ID)+`]}`))
	decodeOK(t, w)

	// set 后应只有 u2
	var count int64
	model.DB(ctx).Model(&model.ProjectMember{}).Where("project_id = ? AND user_id = ?", p.ID, u1.ID).Count(&count)
	if count != 0 {
		t.Errorf("期望旧成员已移除，实际 count=%d", count)
	}
	model.DB(ctx).Model(&model.ProjectMember{}).Where("project_id = ? AND user_id = ?", p.ID, u2.ID).Count(&count)
	if count != 1 {
		t.Errorf("期望新成员已添加，实际 count=%d", count)
	}
}

// ── HandleAdminProjectsByUsers ──────────────────────────────────────────────

func TestHandleAdminProjectsByUsers_Success(t *testing.T) {
	setupProjectsTestDB(t)
	ctx := context.Background()
	p := model.Project{Name: "用户项目"}
	model.DB(ctx).Create(&p)
	u := model.User{Username: "byuser1", Password: "x"}
	model.DB(ctx).Create(&u)
	model.DB(ctx).Create(&model.ProjectMember{ProjectID: p.ID, UserID: u.ID})

	w := httptest.NewRecorder()
	HandleAdminProjectsByUsers(w, adminGet("/admin/projects/projects-by-users?user_ids="+uintStr(u.ID)))
	resp := decodeOK(t, w)
	data := resp["data"].(map[string]any)
	projects := data[uintStr(u.ID)].([]any)
	if len(projects) != 1 {
		t.Fatalf("期望 1 个项目，实际=%d", len(projects))
	}
}

func TestHandleAdminProjectsByUsers_NoUserIDs(t *testing.T) {
	setupProjectsTestDB(t)
	w := httptest.NewRecorder()
	HandleAdminProjectsByUsers(w, adminGet("/admin/projects/projects-by-users"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

// ── uniqueUintIDs / projectIDsFromRequest ───────────────────────────────────

func TestUniqueUintIDs(t *testing.T) {
	result := uniqueUintIDs([]uint{0, 1, 2, 2, 3, 0})
	if len(result) != 3 || result[0] != 1 || result[1] != 2 || result[2] != 3 {
		t.Errorf("期望 [1,2,3]，实际=%v", result)
	}
}

func TestProjectUserIDsFromRequest_PostBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin/projects/by-users", strings.NewReader(`{"user_ids":[1,2,2,0]}`))
	ids, ok := projectUserIDsFromRequest(req)
	if !ok || len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("ids=%v ok=%v", ids, ok)
	}
	req = httptest.NewRequest(http.MethodPost, "/admin/projects/by-users", strings.NewReader("{"))
	if _, ok := projectUserIDsFromRequest(req); ok {
		t.Fatal("invalid JSON should be rejected")
	}
	overLimit := make([]uint, maxProjectUserIDsPerRequest+1)
	for i := range overLimit {
		overLimit[i] = uint(i + 1)
	}
	body, _ := json.Marshal(map[string]any{"user_ids": overLimit})
	req = httptest.NewRequest(http.MethodPost, "/admin/projects/by-users", strings.NewReader(string(body)))
	if _, ok := projectUserIDsFromRequest(req); ok {
		t.Fatalf("more than %d user IDs should be rejected", maxProjectUserIDsPerRequest)
	}
}

func TestAdminProjectHandlers_RequestValidation(t *testing.T) {
	setupProjectsTestDB(t)

	for _, tc := range []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		req     *http.Request
		status  int
	}{
		{"create method", HandleAdminProjectCreate, adminGet("/admin/projects/create"), http.StatusMethodNotAllowed},
		{"create JSON", HandleAdminProjectCreate, adminJSONPost("/admin/projects/create", "{"), http.StatusBadRequest},
		{"update method", HandleAdminProjectUpdate, adminGet("/admin/projects/update"), http.StatusMethodNotAllowed},
		{"update JSON", HandleAdminProjectUpdate, adminJSONPost("/admin/projects/update", "{"), http.StatusBadRequest},
		{"delete method", HandleAdminProjectDelete, adminGet("/admin/projects/delete"), http.StatusMethodNotAllowed},
		{"delete IDs", HandleAdminProjectDelete, adminJSONPost("/admin/projects/delete", `{}`), http.StatusBadRequest},
		{"impact IDs", HandleAdminProjectDeleteImpact, adminGet("/admin/projects/delete-impact"), http.StatusBadRequest},
		{"members method", HandleAdminProjectMembersAdd, adminGet("/admin/projects/members/add"), http.StatusMethodNotAllowed},
		{"members JSON", HandleAdminProjectMembersAdd, adminJSONPost("/admin/projects/members/add", "{"), http.StatusBadRequest},
		{"projects by users method", HandleAdminProjectsByUsers, adminJSONPost("/admin/projects/by-users", `{}`), http.StatusMethodNotAllowed},
		{"projects by users IDs", HandleAdminProjectsByUsers, adminGet("/admin/projects/by-users?user_ids=bad"), http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.handler(w, tc.req)
			if w.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", w.Code, tc.status, w.Body.String())
			}
		})
	}

	project := model.Project{Name: "成员校验项目"}
	if err := model.DB(t.Context()).Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	w := httptest.NewRecorder()
	handleProjectMembersWrite(w, adminJSONPost("/admin/projects/members/add",
		`{"id":`+uintStr(project.ID)+`,"user_ids":[]}`), "unsupported")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("未知成员操作期望 500，实际=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleProjectsMine_WithAssetSummaryAndQuery(t *testing.T) {
	setupProjectsTestDB(t)
	user := model.User{Username: "project-mine-user", Password: "x", Email: "mine@example.com"}
	if err := model.DB(t.Context()).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	projects := []model.Project{{Name: "Alpha Project"}, {Name: "Beta Project"}}
	if err := model.DB(t.Context()).Create(&projects).Error; err != nil {
		t.Fatalf("create projects: %v", err)
	}
	if err := model.DB(t.Context()).Create(&[]model.ProjectMember{
		{ProjectID: projects[0].ID, UserID: user.ID},
		{ProjectID: projects[1].ID, UserID: user.ID},
	}).Error; err != nil {
		t.Fatalf("create memberships: %v", err)
	}
	if err := model.DB(t.Context()).Create(&[]model.ProjectConfigBinding{
		{ProjectID: projects[0].ID, ConfigType: model.AssetBindingTypeSkill, ConfigKey: "skill-a"},
		{ProjectID: projects[0].ID, ConfigType: model.AssetBindingTypeRule, ConfigKey: "rule-a"},
	}).Error; err != nil {
		t.Fatalf("create asset bindings: %v", err)
	}

	w := httptest.NewRecorder()
	HandleProjectsMine(w, authReqWithSession(t, user.Username,
		"/projects/mine?q=alpha&include_asset_summary=true"))
	resp := decodeOK(t, w)
	items := resp["projects"].([]any)
	if len(items) != 1 {
		t.Fatalf("projects=%#v", items)
	}
	item := items[0].(map[string]any)
	if item["skill_count"] != float64(1) || item["rule_count"] != float64(1) {
		t.Fatalf("asset summary=%#v", item)
	}
}

func TestHandleProjectsMine_RequestValidation(t *testing.T) {
	setupProjectsTestDB(t)

	w := httptest.NewRecorder()
	HandleProjectsMine(w, httptest.NewRequest(http.MethodPost, "/projects/mine", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status=%d", w.Code)
	}
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/projects/mine", nil)
	req.Header.Set("Accept", "application/json")
	HandleProjectsMine(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("未登录 status=%d body=%s", w.Code, w.Body.String())
	}
}
