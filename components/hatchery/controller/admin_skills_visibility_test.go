package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ── 测试辅助 ────────────────────────────────────────────────────────

// setupSkillVisibilityScopeDB 初始化包含可见性+分组信息相关表的内存 SQLite 数据库，
// 并设置 AdminToken 使 requireAdmin 可通过 Bearer Token 验证，
// 启用 SMH 使 requireSMHEnabled 通过。
func setupSkillVisibilityScopeDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.Skill{},
		&model.SkillCategoryMapping{},
		&model.SkillCategory{},
		&model.SkillBundle{},
		&model.BundleSkill{},
		&model.SkillDistributionTask{},
		&model.SkillDistributionRecord{},
		&model.SiteConfig{},
		&model.SMHSpace{},
		&model.User{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.SkillVisibilityGroup{},
		&model.SkillBundleVisibilityGroup{},
		&model.RoleVisibilityGroup{},
		&model.Project{},
		&model.ProjectConfigBinding{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	// 创建 SiteConfig 并启用 SMH（requireSMHEnabled 需要 SMHEnabled=1）
	db.Create(&model.SiteConfig{SMHEnabled: 1})

	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })
}

// makeSkill 创建带可见性类型的技能并返回
func makeSkill(t *testing.T, name, slug, visType string) model.Skill {
	t.Helper()
	s := model.Skill{
		Slug:           slug,
		Name:           name,
		Version:        "1.0.0",
		VersionMajor:   1,
		VersionMinor:   0,
		VersionPatch:   0,
		VisibilityType: visType,
	}
	if err := model.DB(context.Background()).Create(&s).Error; err != nil {
		t.Fatalf("创建技能失败: %v", err)
	}
	return s
}

// adminSkillsGet 创建带 admin Bearer Token 的 GET 请求
func adminSkillsGet(url string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// ── HandleAdminSkills 可见性筛选测试（调用真实 handler） ─────────────

func TestSkillVisibilityScope_AllFilter(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	makeSkill(t, "全局技能A", "global-a", "all")
	makeSkill(t, "全局技能B", "global-b", "all")
	makeSkill(t, "分组技能C", "group-c", "group")

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet("/admin/skills?visibility_type=all"))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 2 {
		t.Errorf("期望 2 个 all 技能，实际=%d", total)
	}

	skills := resp["skills"].([]interface{})
	for _, s := range skills {
		vt := s.(map[string]interface{})["visibility_type"].(string)
		if vt != "all" {
			t.Errorf("期望所有结果 visibility_type=all，实际=%s", vt)
		}
	}
}

func TestSkillVisibilityScope_GroupIDFilter(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	group1 := model.UserGroup{Name: "研发组"}
	model.DB(context.Background()).Create(&group1)
	group2 := model.UserGroup{Name: "产品组"}
	model.DB(context.Background()).Create(&group2)

	skillA := makeSkill(t, "技能A", "skill-a", "group")
	skillB := makeSkill(t, "技能B", "skill-b", "group")
	makeSkill(t, "技能C", "skill-c", "all")

	model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: skillA.ID, GroupID: group1.ID})
	model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: skillB.ID, GroupID: group2.ID})

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet(fmt.Sprintf("/admin/skills?group_id=%d", group1.ID)))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 1 {
		t.Errorf("期望 1 个技能（只匹配 group1），实际=%d", total)
	}
	skills := resp["skills"].([]interface{})
	skillName := skills[0].(map[string]interface{})["name"].(string)
	if skillName != "技能A" {
		t.Errorf("期望技能A，实际=%s", skillName)
	}
}

func TestSkillVisibilityScope_BothFilters(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	group := model.UserGroup{Name: "测试组"}
	model.DB(context.Background()).Create(&group)

	makeSkill(t, "全局技能", "global-skill", "all")
	skillGroup := makeSkill(t, "分组技能", "group-skill", "group")
	makeSkill(t, "其他分组技能", "other-group", "group")

	model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: skillGroup.ID, GroupID: group.ID})

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet(fmt.Sprintf("/admin/skills?visibility_type=all&group_id=%d", group.ID)))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 2 {
		t.Errorf("期望 2 个技能（全局+匹配分组），实际=%d", total)
	}

	skills := resp["skills"].([]interface{})
	nameSet := make(map[string]bool)
	for _, s := range skills {
		nameSet[s.(map[string]interface{})["name"].(string)] = true
	}
	if !nameSet["全局技能"] || !nameSet["分组技能"] {
		t.Errorf("期望包含全局技能和分组技能，实际=%v", nameSet)
	}
	if nameSet["其他分组技能"] {
		t.Error("不应包含其他分组技能")
	}
}

func TestSkillVisibilityScope_GroupAndProjectFiltersUseUnion(t *testing.T) {
	setupSkillVisibilityScopeDB(t)
	group := model.UserGroup{Name: "项目与分组联合筛选组"}
	project := model.Project{Name: "项目与分组联合筛选项目"}
	model.DB(context.Background()).Create(&group)
	model.DB(context.Background()).Create(&project)

	groupSkill := makeSkill(t, "分组技能", "union-group-skill", model.VisibilityGroup)
	projectSkill := makeSkill(t, "项目技能", "union-project-skill", model.VisibilityAll)
	model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: groupSkill.ID, GroupID: group.ID})
	model.DB(context.Background()).Create(&model.ProjectConfigBinding{ProjectID: project.ID, ConfigType: model.ProjectConfigTypeSkill, ConfigKey: projectSkill.Slug})

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet(fmt.Sprintf("/admin/skills?group_id=%d&project_id=%d", group.ID, project.ID)))
	var resp struct {
		Total  int64 `json:"total"`
		Skills []struct {
			Slug string `json:"slug"`
		} `json:"skills"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Fatalf("分组和项目筛选应返回并集 2 条，实际=%d body=%s", resp.Total, w.Body.String())
	}
	slugs := map[string]bool{}
	for _, skill := range resp.Skills {
		slugs[skill.Slug] = true
	}
	if !slugs[groupSkill.Slug] || !slugs[projectSkill.Slug] {
		t.Fatalf("筛选结果应同时包含分组和项目命中项，实际=%v", slugs)
	}
}

// ── buildSkillVisibilityData 测试（含 group_name 验证） ──────────────

func TestSkillVisibilityScope_BuildData_AllType(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	skill := makeSkill(t, "全局技能", "global", "all")

	result := buildSkillVisibilityData(context.Background(), []model.Skill{skill})
	if len(result) != 0 {
		t.Errorf("期望 all 类型技能的 visibilityData 为空 map，实际=%v", result)
	}
}

func TestSkillVisibilityScope_BuildData_GroupType(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	group1 := model.UserGroup{Name: "前端组"}
	model.DB(context.Background()).Create(&group1)
	group2 := model.UserGroup{Name: "后端组"}
	model.DB(context.Background()).Create(&group2)

	skill := makeSkill(t, "分组技能", "group-skill", "group")
	model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: skill.ID, GroupID: group1.ID})
	model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: skill.ID, GroupID: group2.ID})

	result := buildSkillVisibilityData(context.Background(), []model.Skill{skill})

	groups := result[skill.ID]
	if len(groups) != 2 {
		t.Fatalf("期望 2 个分组信息，实际=%d", len(groups))
	}

	nameSet := make(map[string]bool)
	for _, g := range groups {
		nameSet[g.GroupName] = true
	}
	if !nameSet["前端组"] || !nameSet["后端组"] {
		t.Errorf("期望包含前端组和后端组，实际=%v", groups)
	}
}

func TestSkillVisibilityScope_BuildData_MixedTypes(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	group := model.UserGroup{Name: "混合测试组"}
	model.DB(context.Background()).Create(&group)

	skillAll := makeSkill(t, "全局", "global", "all")
	skillGroup := makeSkill(t, "分组", "grouped", "group")
	model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: skillGroup.ID, GroupID: group.ID})

	result := buildSkillVisibilityData(context.Background(), []model.Skill{skillAll, skillGroup})

	if _, ok := result[skillAll.ID]; ok {
		t.Error("all 类型技能不应有分组数据")
	}

	groups := result[skillGroup.ID]
	if len(groups) != 1 {
		t.Errorf("期望 1 个分组，实际=%d", len(groups))
	}
	if groups[0].GroupName != "混合测试组" {
		t.Errorf("期望 group_name=混合测试组，实际=%s", groups[0].GroupName)
	}
}

func TestSkillVisibilityScope_BuildData_EmptyList(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	result := buildSkillVisibilityData(context.Background(), []model.Skill{})
	if len(result) != 0 {
		t.Errorf("期望空列表返回空 map，实际=%v", result)
	}
}

// ── 列表响应包含 visibility_groups 字段测试（调用真实 handler） ──────

func TestSkillVisibilityScope_ResponseIncludesGroups(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	group := model.UserGroup{Name: "响应测试组"}
	model.DB(context.Background()).Create(&group)

	skill := makeSkill(t, "带分组技能", "vis-skill", "group")
	model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: skill.ID, GroupID: group.ID})

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet("/admin/skills"))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	skills := resp["skills"].([]interface{})
	if len(skills) != 1 {
		t.Fatalf("期望 1 个技能，实际=%d", len(skills))
	}

	s := skills[0].(map[string]interface{})
	visGroups := s["visibility_groups"].([]interface{})
	if len(visGroups) != 1 {
		t.Fatalf("期望 1 个 visibility_groups，实际=%d", len(visGroups))
	}

	vg := visGroups[0].(map[string]interface{})
	if vg["group_name"] != "响应测试组" {
		t.Errorf("期望 group_name=响应测试组，实际=%v", vg["group_name"])
	}
	if uint(vg["group_id"].(float64)) != group.ID {
		t.Errorf("期望 group_id=%d，实际=%v", group.ID, vg["group_id"])
	}
}

func TestSkillVisibilityScope_AllTypeEmptyGroups(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	makeSkill(t, "全局技能", "all-skill", "all")

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet("/admin/skills"))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	skills := resp["skills"].([]interface{})
	s := skills[0].(map[string]interface{})
	visGroups := s["visibility_groups"].([]interface{})
	if len(visGroups) != 0 {
		t.Errorf("all 类型应返回空 visibility_groups，实际=%d", len(visGroups))
	}
}

// ── 分页与筛选组合测试（调用真实 handler） ──────────────────────────

func TestSkillVisibilityScope_PaginationWithFilter(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	for i := 0; i < 5; i++ {
		makeSkill(t, fmt.Sprintf("全局技能%d", i), fmt.Sprintf("global-%d", i), "all")
	}
	makeSkill(t, "分组技能", "group-0", "group")

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet("/admin/skills?visibility_type=all&page=1&page_size=2"))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 5 {
		t.Errorf("期望 total=5，实际=%d", total)
	}
	skills := resp["skills"].([]interface{})
	if len(skills) != 2 {
		t.Errorf("期望本页 2 条，实际=%d", len(skills))
	}
}

func TestSkillVisibilityScope_GroupIDNoMatch(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	group := model.UserGroup{Name: "无关组"}
	model.DB(context.Background()).Create(&group)

	makeSkill(t, "无关联技能", "no-assoc", "group")
	makeSkill(t, "全局", "global", "all")

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet(fmt.Sprintf("/admin/skills?group_id=%d", group.ID)))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 0 {
		t.Errorf("期望 0 个技能（无分组关联匹配），实际=%d", total)
	}
}

func TestSkillVisibilityScope_MultipleGroupIDs(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	group1 := model.UserGroup{Name: "组一"}
	model.DB(context.Background()).Create(&group1)
	group2 := model.UserGroup{Name: "组二"}
	model.DB(context.Background()).Create(&group2)

	skillA := makeSkill(t, "技能X", "skill-x", "group")
	skillB := makeSkill(t, "技能Y", "skill-y", "group")

	model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: skillA.ID, GroupID: group1.ID})
	model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: skillB.ID, GroupID: group2.ID})

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet(fmt.Sprintf("/admin/skills?group_id=%d,%d", group1.ID, group2.ID)))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 2 {
		t.Errorf("期望 2 个技能，实际=%d", total)
	}
}

// ── 一个技能关联多个分组测试 ────────────────────────────────────────

func TestSkillVisibilityScope_SkillInMultipleGroups(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	group1 := model.UserGroup{Name: "组甲"}
	model.DB(context.Background()).Create(&group1)
	group2 := model.UserGroup{Name: "组乙"}
	model.DB(context.Background()).Create(&group2)

	skill := makeSkill(t, "多组技能", "multi-group", "group")
	model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: skill.ID, GroupID: group1.ID})
	model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: skill.ID, GroupID: group2.ID})

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet(fmt.Sprintf("/admin/skills?group_id=%d", group1.ID)))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 1 {
		t.Errorf("期望 1 个技能，实际=%d", total)
	}

	// buildSkillVisibilityData 应返回 2 个分组
	result := buildSkillVisibilityData(context.Background(), []model.Skill{skill})
	groups := result[skill.ID]
	if len(groups) != 2 {
		t.Errorf("期望 2 个分组信息，实际=%d", len(groups))
	}
}

// ── slug 精确匹配筛选测试 ──────────────────────────────────────────

func TestHandleAdminSkills_SlugFilter_ExactMatch(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	makeSkill(t, "技能A", "skill-alpha", "all")
	makeSkill(t, "技能B", "skill-beta", "all")
	makeSkill(t, "技能C", "skill-gamma", "all")

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet("/admin/skills?slug=skill-beta"))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 1 {
		t.Errorf("期望精确匹配返回 1 个技能，实际=%d", total)
	}

	skills := resp["skills"].([]interface{})
	if len(skills) != 1 {
		t.Fatalf("期望 1 个技能，实际=%d", len(skills))
	}
	slug := skills[0].(map[string]interface{})["slug"].(string)
	if slug != "skill-beta" {
		t.Errorf("期望 slug=skill-beta，实际=%s", slug)
	}
}

func TestHandleAdminSkills_SlugFilter_NoMatch(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	makeSkill(t, "技能A", "skill-alpha", "all")
	makeSkill(t, "技能B", "skill-beta", "all")

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet("/admin/skills?slug=nonexistent"))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 0 {
		t.Errorf("不存在的 slug 应返回 0，实际=%d", total)
	}
}

func TestHandleAdminSkills_SlugFilter_PartialNotMatch(t *testing.T) {
	// slug 是精确匹配（=），部分字符串不应匹配
	setupSkillVisibilityScopeDB(t)

	makeSkill(t, "技能A", "skill-alpha-beta", "all")
	makeSkill(t, "技能B", "skill-gamma", "all")

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet("/admin/skills?slug=skill-alpha"))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 0 {
		t.Errorf("部分 slug 不应匹配（精确匹配），期望 0，实际=%d", total)
	}
}

func TestHandleAdminSkills_SlugFilter_CombinedWithName(t *testing.T) {
	// slug + name 同时传，两个条件 AND
	setupSkillVisibilityScopeDB(t)

	makeSkill(t, "搜索技能", "search-skill", "all")
	makeSkill(t, "其他技能", "other-skill", "all")
	makeSkill(t, "搜索技能V2", "search-skill-v2", "all")

	// slug=search-skill 且 name 包含 "搜索"
	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet("/admin/skills?slug=search-skill&name=%E6%90%9C%E7%B4%A2"))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 1 {
		t.Errorf("期望 slug+name 组合返回 1 个，实际=%d", total)
	}

	skills := resp["skills"].([]interface{})
	s := skills[0].(map[string]interface{})
	if s["slug"] != "search-skill" {
		t.Errorf("期望 slug=search-skill，实际=%s", s["slug"])
	}
}

// ── keyword 含 slug 匹配测试 ──────────────────────────────────────

func TestHandleAdminSkills_KeywordFilter_SlugMatch(t *testing.T) {
	// keyword 应同时匹配 name、description、slug
	setupSkillVisibilityScopeDB(t)

	makeSkill(t, "全局技能Alpha", "global-skill-001", "all")
	makeSkill(t, "全局技能Beta", "beta-toolkit", "all")

	// keyword=global-skill-001 — 名称不含此字符串，但 slug 包含
	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet("/admin/skills?keyword=global-skill-001"))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 1 {
		t.Errorf("keyword 应通过 slug 匹配到 1 个技能，实际=%d", total)
	}

	skills := resp["skills"].([]interface{})
	slug := skills[0].(map[string]interface{})["slug"].(string)
	if slug != "global-skill-001" {
		t.Errorf("期望 slug=global-skill-001，实际=%s", slug)
	}
}

func TestHandleAdminSkills_KeywordFilter_NameMatch(t *testing.T) {
	// keyword 应同时匹配 name（已有能力回归测试）
	setupSkillVisibilityScopeDB(t)

	makeSkill(t, "Python代码助手", "py-helper", "all")
	makeSkill(t, "Go开发工具", "go-tools", "all")
	makeSkill(t, "Java框架", "java-fwk", "all")

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet("/admin/skills?keyword=Python"))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 1 {
		t.Errorf("keyword 应通过 name 匹配到 1 个技能，实际=%d", total)
	}
}

func TestHandleAdminSkills_KeywordFilter_DescriptionMatch(t *testing.T) {
	// keyword 应同时匹配 description
	setupSkillVisibilityScopeDB(t)

	model.DB(context.Background()).Create(&model.Skill{
		Slug: "img-tool", Name: "图像工具", Version: "1.0.0", VisibilityType: "all",
		Description: "用于图像处理与批量转换",
	})
	model.DB(context.Background()).Create(&model.Skill{
		Slug: "txt-tool", Name: "文本工具", Version: "1.0.0", VisibilityType: "all",
		Description: "文本格式化与清洗",
	})

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet("/admin/skills?keyword=批量转换"))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 1 {
		t.Errorf("keyword 应通过 description 匹配到 1 个技能，实际=%d", total)
	}

	skills := resp["skills"].([]interface{})
	slug := skills[0].(map[string]interface{})["slug"].(string)
	if slug != "img-tool" {
		t.Errorf("期望 slug=img-tool，实际=%s", slug)
	}
}

func TestHandleAdminSkills_KeywordFilter_CombinedNameSlugDesc(t *testing.T) {
	// keyword 综合匹配 name、slug、description
	setupSkillVisibilityScopeDB(t)

	model.DB(context.Background()).Create(&model.Skill{
		Slug: "data-cleaner-pro", Name: "数据处理专家", Version: "1.0.0", VisibilityType: "all",
		Description: "数据清洗与预处理工具",
	})
	model.DB(context.Background()).Create(&model.Skill{
		Slug: "clean-desk", Name: "桌面清理工具", Version: "1.0.0", VisibilityType: "all",
		Description: "清理桌面文件",
	})

	// keyword=clean 匹配: slug "data-cleaner-pro" + "clean-desk" + desc "数据清洗"
	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet("/admin/skills?keyword=clean"))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total < 1 {
		t.Errorf("keyword 'clean' 应至少匹配 1 个技能，实际=%d", total)
	}
}

func TestHandleAdminSkills_KeywordFilter_NoMatch(t *testing.T) {
	setupSkillVisibilityScopeDB(t)

	makeSkill(t, "技能A", "skill-a", "all")
	makeSkill(t, "技能B", "skill-b", "all")

	w := httptest.NewRecorder()
	HandleAdminSkills(w, adminSkillsGet("/admin/skills?keyword=zzz_unmatchable"))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 0 {
		t.Errorf("不匹配的 keyword 应返回 0，实际=%d", total)
	}
}
