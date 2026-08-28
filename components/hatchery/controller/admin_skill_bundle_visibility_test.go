package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ── 测试辅助 ────────────────────────────────────────────────────────

// setupBundleVisibilityTestDB 初始化包含可见性相关表的内存 SQLite 数据库，
// 并设置 AdminToken 使 requireAdmin 可通过 Bearer Token 验证。
func setupBundleVisibilityTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.SkillBundle{},
		&model.BundleSkill{},
		&model.SiteConfig{},
		&model.SMHSpace{},
		&model.User{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.SkillBundleVisibilityGroup{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	db.Create(&model.SiteConfig{SMHEnabled: 1})

	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })
}

// adminBundleGet 创建带 admin Bearer Token 的 GET 请求
func adminBundleGet(url string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// adminBundlePost 创建带 admin Bearer Token 的 POST 表单请求
func adminBundlePost(urlStr string, form url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, urlStr, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// adminBundlePostNoBody 创建带 admin Bearer Token 的 POST 请求（无 body）
func adminBundlePostNoBody(urlStr string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, urlStr, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// ── HandleAdminSkillBundles 可见性筛选测试（调用真实 handler） ────────

func TestListBundles_VisibilityTypeFilter_All(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	model.DB(context.Background()).Create(&model.SkillBundle{Name: "全局包A", VisibilityType: "all"})
	model.DB(context.Background()).Create(&model.SkillBundle{Name: "全局包B", VisibilityType: "all"})
	model.DB(context.Background()).Create(&model.SkillBundle{Name: "分组包C", VisibilityType: "group"})

	w := httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet("/admin/skill-bundles?visibility_type=all"))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 2 {
		t.Errorf("期望 2 个 all 技能包，实际=%d", total)
	}
}

func TestListBundles_GroupIDFilter(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	group1 := model.UserGroup{Name: "组一"}
	model.DB(context.Background()).Create(&group1)

	bundleA := model.SkillBundle{Name: "分组包A", VisibilityType: "group"}
	model.DB(context.Background()).Create(&bundleA)
	model.DB(context.Background()).Create(&model.SkillBundleVisibilityGroup{SkillBundleID: bundleA.ID, GroupID: group1.ID})

	model.DB(context.Background()).Create(&model.SkillBundle{Name: "全局包", VisibilityType: "all"})

	w := httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet(fmt.Sprintf("/admin/skill-bundles?group_id=%d", group1.ID)))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 1 {
		t.Errorf("期望 1 个匹配分组的技能包，实际=%d", total)
	}
}

func TestListBundles_VisibilityTypePlusGroupID(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	group := model.UserGroup{Name: "联合测试组"}
	model.DB(context.Background()).Create(&group)

	model.DB(context.Background()).Create(&model.SkillBundle{Name: "全局包", VisibilityType: "all"})
	bundleGroup := model.SkillBundle{Name: "分组包", VisibilityType: "group"}
	model.DB(context.Background()).Create(&bundleGroup)
	model.DB(context.Background()).Create(&model.SkillBundleVisibilityGroup{SkillBundleID: bundleGroup.ID, GroupID: group.ID})
	model.DB(context.Background()).Create(&model.SkillBundle{Name: "其他分组包", VisibilityType: "group"})

	w := httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet(fmt.Sprintf("/admin/skill-bundles?visibility_type=all&group_id=%d", group.ID)))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 2 {
		t.Errorf("期望 2 个技能包（全局+匹配分组），实际=%d", total)
	}
}

func TestListBundles_NoFilter_ReturnsAll(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	model.DB(context.Background()).Create(&model.SkillBundle{Name: "包1", VisibilityType: "all"})
	model.DB(context.Background()).Create(&model.SkillBundle{Name: "包2", VisibilityType: "group"})
	model.DB(context.Background()).Create(&model.SkillBundle{Name: "包3", VisibilityType: "all"})

	w := httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet("/admin/skill-bundles"))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	total := int(resp["total"].(float64))
	if total != 3 {
		t.Errorf("无筛选条件应返回全部，期望 3，实际=%d", total)
	}
}

func TestListBundles_SearchByKeywordAndID(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	alpha := model.SkillBundle{Name: "Alpha 初始包", VisibilityType: "all"}
	beta := model.SkillBundle{Name: "Beta 研发包", VisibilityType: "all"}
	if err := model.DB(context.Background()).Create(&alpha).Error; err != nil {
		t.Fatalf("创建 Alpha 失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&beta).Error; err != nil {
		t.Fatalf("创建 Beta 失败: %v", err)
	}

	w := httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet("/admin/skill-bundles?keyword=Alpha"))
	var byName map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&byName); err != nil {
		t.Fatalf("解析名称搜索响应失败: %v", err)
	}
	if total := int(byName["total"].(float64)); total != 1 {
		t.Fatalf("名称搜索期望 1，实际=%d", total)
	}

	w = httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet(fmt.Sprintf("/admin/skill-bundles?id=%d", beta.ID)))
	var byID map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&byID); err != nil {
		t.Fatalf("解析 ID 搜索响应失败: %v", err)
	}
	if total := int(byID["total"].(float64)); total != 1 {
		t.Fatalf("ID 搜索期望 1，实际=%d", total)
	}
	bundles := byID["skill_bundles"].([]interface{})
	got := bundles[0].(map[string]interface{})
	if uint(got["id"].(float64)) != beta.ID {
		t.Fatalf("ID 搜索返回 id=%v，期望 %d", got["id"], beta.ID)
	}
}

func TestListBundles_ReverseLookupBySkillAndSkillset(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	publicBundle := model.SkillBundle{Name: "公共技能包", VisibilityType: "all"}
	enterpriseBundle := model.SkillBundle{Name: "企业技能包", VisibilityType: "all"}
	skillsetBundle := model.SkillBundle{Name: "Skillset 技能包", VisibilityType: "all"}
	if err := model.DB(context.Background()).Create(&publicBundle).Error; err != nil {
		t.Fatalf("创建公共技能包失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&enterpriseBundle).Error; err != nil {
		t.Fatalf("创建企业技能包失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&skillsetBundle).Error; err != nil {
		t.Fatalf("创建 Skillset 技能包失败: %v", err)
	}

	skills := []model.BundleSkill{
		{
			SkillBundleID:      publicBundle.ID,
			Name:               "共享技能",
			Slug:               "shared-skill",
			Version:            "1.0.0",
			Source:             "public",
			SourceSkillsetSlug: "finance-risk-assessment",
			SourceSkillsetName: "金融风控技能包",
		},
		{
			SkillBundleID: enterpriseBundle.ID,
			Name:          "企业共享技能",
			Slug:          "shared-skill",
			Version:       "2.0.0",
			Source:        "enterprise",
		},
		{
			SkillBundleID:      skillsetBundle.ID,
			Name:               "Skillset 其他技能",
			Slug:               "another-skill",
			Version:            "1.0.0",
			Source:             "public",
			SourceSkillsetSlug: "finance-risk-assessment",
			SourceSkillsetName: "金融风控技能包",
		},
	}
	if err := model.DB(context.Background()).Create(&skills).Error; err != nil {
		t.Fatalf("创建技能包技能失败: %v", err)
	}

	decode := func(w *httptest.ResponseRecorder) map[string]interface{} {
		t.Helper()
		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}
		return resp
	}

	w := httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet("/admin/skill-bundles?skill_slug=shared-skill"))
	if w.Code != http.StatusOK {
		t.Fatalf("按技能 slug 反查期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	resp := decode(w)
	if total := int(resp["total"].(float64)); total != 2 {
		t.Fatalf("按技能 slug 反查期望 2，实际=%d", total)
	}

	w = httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet("/admin/skill-bundles?skill_slug=shared-skill&skill_source=public"))
	if w.Code != http.StatusOK {
		t.Fatalf("按公共技能反查期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	resp = decode(w)
	if total := int(resp["total"].(float64)); total != 1 {
		t.Fatalf("按公共技能反查期望 1，实际=%d", total)
	}
	bundles := resp["skill_bundles"].([]interface{})
	got := bundles[0].(map[string]interface{})
	if uint(got["id"].(float64)) != publicBundle.ID {
		t.Fatalf("按公共技能反查返回 id=%v，期望 %d", got["id"], publicBundle.ID)
	}
	if count := int(got["matched_skill_count"].(float64)); count != 1 {
		t.Fatalf("matched_skill_count=%d，期望 1", count)
	}
	matchedSkills := got["matched_skills"].([]interface{})
	matchedSkill := matchedSkills[0].(map[string]interface{})
	if matchedSkill["source"].(string) != "public" {
		t.Fatalf("matched source=%v，期望 public", matchedSkill["source"])
	}

	w = httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet("/admin/skill-bundles?source_skillset_slug=finance-risk-assessment"))
	if w.Code != http.StatusOK {
		t.Fatalf("按 Skillset 反查期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	resp = decode(w)
	if total := int(resp["total"].(float64)); total != 2 {
		t.Fatalf("按 Skillset 反查期望 2，实际=%d", total)
	}

	w = httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet("/admin/skill-bundles?skill_slug=shared-skill&skill_source=all"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("skill_source=all 期望 400，实际=%d", w.Code)
	}

	w = httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet("/admin/skill-bundles?skill_slug=shared-skill&source_skillset_slug=finance-risk-assessment"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("同时传 skill_slug 和 source_skillset_slug 期望 400，实际=%d", w.Code)
	}
}

// ── HandleUpdateSkillBundleVisibility 测试（调用真实 handler） ───────

func TestUpdateBundleVisibility_BasicFlow(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	group := model.UserGroup{Name: "目标组"}
	model.DB(context.Background()).Create(&group)

	bundle := model.SkillBundle{Name: "测试包", VisibilityType: "all"}
	model.DB(context.Background()).Create(&bundle)

	form := url.Values{}
	form.Set("visibility_type", "group")
	form.Set("group_ids", fmt.Sprintf("%d", group.ID))
	w := httptest.NewRecorder()
	HandleUpdateSkillBundleVisibility(w, adminBundlePost(fmt.Sprintf("/admin/skill-bundles/update-visibility?id=%d", bundle.ID), form))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var updated model.SkillBundle
	model.DB(context.Background()).First(&updated, bundle.ID)
	if updated.VisibilityType != "group" {
		t.Errorf("期望 visibility_type=group，实际=%s", updated.VisibilityType)
	}

	var vgs []model.SkillBundleVisibilityGroup
	model.DB(context.Background()).Where("skill_bundle_id = ?", bundle.ID).Find(&vgs)
	if len(vgs) != 1 {
		t.Fatalf("期望 1 条关联，实际=%d", len(vgs))
	}
	if vgs[0].GroupID != group.ID {
		t.Errorf("期望 group_id=%d，实际=%d", group.ID, vgs[0].GroupID)
	}
}

func TestUpdateBundleVisibility_AutoDisableOnSwitchToAll(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	group := model.UserGroup{Name: "禁用测试组"}
	model.DB(context.Background()).Create(&group)

	bundle := model.SkillBundle{Name: "已启用包", VisibilityType: "group", Enabled: true}
	model.DB(context.Background()).Create(&bundle)
	model.DB(context.Background()).Create(&model.SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: group.ID})

	form := url.Values{}
	form.Set("visibility_type", "all")
	w := httptest.NewRecorder()
	HandleUpdateSkillBundleVisibility(w, adminBundlePost(fmt.Sprintf("/admin/skill-bundles/update-visibility?id=%d", bundle.ID), form))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	var updated model.SkillBundle
	model.DB(context.Background()).First(&updated, bundle.ID)
	if updated.Enabled {
		t.Error("切换为 all 类型时应自动禁用技能包")
	}
	if updated.VisibilityType != "all" {
		t.Errorf("期望 visibility_type=all，实际=%s", updated.VisibilityType)
	}

	var count int64
	model.DB(context.Background()).Model(&model.SkillBundleVisibilityGroup{}).Where("skill_bundle_id = ?", bundle.ID).Count(&count)
	if count != 0 {
		t.Errorf("切换为 all 类型后分组关联应清理，实际=%d", count)
	}
}

func TestUpdateBundleVisibility_NoAutoDisableIfAlreadyAll(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	bundle := model.SkillBundle{Name: "已是全局", VisibilityType: "all", Enabled: true}
	model.DB(context.Background()).Create(&bundle)

	form := url.Values{}
	form.Set("visibility_type", "all")
	w := httptest.NewRecorder()
	HandleUpdateSkillBundleVisibility(w, adminBundlePost(fmt.Sprintf("/admin/skill-bundles/update-visibility?id=%d", bundle.ID), form))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	var updated model.SkillBundle
	model.DB(context.Background()).First(&updated, bundle.ID)
	if !updated.Enabled {
		t.Error("已经是 all 类型时不应自动禁用")
	}
}

func TestUpdateBundleVisibility_NotFound(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	form := url.Values{}
	form.Set("visibility_type", "all")
	w := httptest.NewRecorder()
	HandleUpdateSkillBundleVisibility(w, adminBundlePost("/admin/skill-bundles/update-visibility?id=99999", form))

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

func TestUpdateBundleVisibility_MissingVisibilityType(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	bundle := model.SkillBundle{Name: "缺参数包", VisibilityType: "all"}
	model.DB(context.Background()).Create(&bundle)

	form := url.Values{}
	w := httptest.NewRecorder()
	HandleUpdateSkillBundleVisibility(w, adminBundlePost(fmt.Sprintf("/admin/skill-bundles/update-visibility?id=%d", bundle.ID), form))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

func TestUpdateBundleVisibility_InvalidVisibilityType(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	bundle := model.SkillBundle{Name: "无效类型包", VisibilityType: "all"}
	model.DB(context.Background()).Create(&bundle)

	form := url.Values{}
	form.Set("visibility_type", "invalid")
	w := httptest.NewRecorder()
	HandleUpdateSkillBundleVisibility(w, adminBundlePost(fmt.Sprintf("/admin/skill-bundles/update-visibility?id=%d", bundle.ID), form))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// ── HandleToggleSkillBundle 互斥测试（调用真实 handler） ─────────────

func TestToggleBundle_AllTypeMutualExclusion(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	bundleA := model.SkillBundle{Name: "全局A", VisibilityType: "all", Enabled: true}
	model.DB(context.Background()).Create(&bundleA)
	bundleB := model.SkillBundle{Name: "全局B", VisibilityType: "all", Enabled: false}
	model.DB(context.Background()).Create(&bundleB)

	w := httptest.NewRecorder()
	HandleToggleSkillBundle(w, adminBundlePostNoBody(fmt.Sprintf("/admin/skill-bundles/toggle?id=%d", bundleB.ID)))

	if w.Code != http.StatusConflict {
		t.Fatalf("期望 409（互斥），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestToggleBundle_GroupTypeNoMutualExclusion(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	bundleA := model.SkillBundle{Name: "分组A", VisibilityType: "group", Enabled: true}
	model.DB(context.Background()).Create(&bundleA)
	bundleB := model.SkillBundle{Name: "分组B", VisibilityType: "group", Enabled: false}
	model.DB(context.Background()).Create(&bundleB)

	w := httptest.NewRecorder()
	HandleToggleSkillBundle(w, adminBundlePostNoBody(fmt.Sprintf("/admin/skill-bundles/toggle?id=%d", bundleB.ID)))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（group 类型不互斥），实际=%d", w.Code)
	}

	var updated model.SkillBundle
	model.DB(context.Background()).First(&updated, bundleB.ID)
	if !updated.Enabled {
		t.Error("期望 group 类型技能包启用成功")
	}
}

func TestToggleBundle_DisableExistingAll(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	bundle := model.SkillBundle{Name: "全局包", VisibilityType: "all", Enabled: true}
	model.DB(context.Background()).Create(&bundle)

	w := httptest.NewRecorder()
	HandleToggleSkillBundle(w, adminBundlePostNoBody(fmt.Sprintf("/admin/skill-bundles/toggle?id=%d", bundle.ID)))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	var updated model.SkillBundle
	model.DB(context.Background()).First(&updated, bundle.ID)
	if updated.Enabled {
		t.Error("期望技能包已禁用")
	}
}

// ── HandleSkillBundleDetail 测试（调用真实 handler） ─────────────────

func TestSkillBundleDetail_WithVisibilityGroups(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	group := model.UserGroup{Name: "详情测试组"}
	model.DB(context.Background()).Create(&group)

	bundle := model.SkillBundle{Name: "详情包", VisibilityType: "group"}
	model.DB(context.Background()).Create(&bundle)
	model.DB(context.Background()).Create(&model.SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: group.ID})

	w := httptest.NewRecorder()
	HandleSkillBundleDetail(w, adminBundleGet(fmt.Sprintf("/admin/skill-bundles/detail?id=%d", bundle.ID)))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	visGroups := resp["visible_groups"].([]interface{})
	if len(visGroups) != 1 {
		t.Fatalf("期望 1 个 visible_groups，实际=%d", len(visGroups))
	}
	vg := visGroups[0].(map[string]interface{})
	if vg["group_name"] != "详情测试组" {
		t.Errorf("期望 group_name=详情测试组，实际=%v", vg["group_name"])
	}
}

func TestSkillBundleDetail_NotFound(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	w := httptest.NewRecorder()
	HandleSkillBundleDetail(w, adminBundleGet("/admin/skill-bundles/detail?id=99999"))

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

// ── buildBundleVisibilityData 测试 ──────────────────────────────────

func TestBuildBundleVisibilityData_AllType(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	bundle := model.SkillBundle{Name: "全局包", VisibilityType: "all"}
	model.DB(context.Background()).Create(&bundle)

	result := buildBundleVisibilityData(context.Background(), []model.SkillBundle{bundle})
	if len(result) != 0 {
		t.Errorf("期望 all 类型返回空 map，实际=%v", result)
	}
}

func TestBuildBundleVisibilityData_GroupType(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	group1 := model.UserGroup{Name: "开发组"}
	model.DB(context.Background()).Create(&group1)
	group2 := model.UserGroup{Name: "测试组"}
	model.DB(context.Background()).Create(&group2)

	bundle := model.SkillBundle{Name: "分组包", VisibilityType: "group"}
	model.DB(context.Background()).Create(&bundle)
	model.DB(context.Background()).Create(&model.SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: group1.ID})
	model.DB(context.Background()).Create(&model.SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: group2.ID})

	result := buildBundleVisibilityData(context.Background(), []model.SkillBundle{bundle})

	groups := result[bundle.ID]
	if len(groups) != 2 {
		t.Fatalf("期望 2 个分组信息，实际=%d", len(groups))
	}

	nameSet := make(map[string]bool)
	for _, g := range groups {
		nameSet[g.GroupName] = true
	}
	if !nameSet["开发组"] || !nameSet["测试组"] {
		t.Errorf("期望包含开发组和测试组，实际=%v", groups)
	}
}

func TestBuildBundleVisibilityData_EmptyList(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	result := buildBundleVisibilityData(context.Background(), []model.SkillBundle{})
	if len(result) != 0 {
		t.Errorf("期望空列表返回空 map，实际=%v", result)
	}
}

func TestBuildBundleVisibilityData_MixedTypes(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	group := model.UserGroup{Name: "混合组"}
	model.DB(context.Background()).Create(&group)

	bundleAll := model.SkillBundle{Name: "全局", VisibilityType: "all"}
	model.DB(context.Background()).Create(&bundleAll)
	bundleGroup := model.SkillBundle{Name: "分组", VisibilityType: "group"}
	model.DB(context.Background()).Create(&bundleGroup)
	model.DB(context.Background()).Create(&model.SkillBundleVisibilityGroup{SkillBundleID: bundleGroup.ID, GroupID: group.ID})

	result := buildBundleVisibilityData(context.Background(), []model.SkillBundle{bundleAll, bundleGroup})

	if _, ok := result[bundleAll.ID]; ok {
		t.Error("all 类型不应有分组数据")
	}
	if groups := result[bundleGroup.ID]; len(groups) != 1 || groups[0].GroupName != "混合组" {
		t.Errorf("期望分组包含混合组，实际=%v", result[bundleGroup.ID])
	}
}

// ── HandleCreateSkillBundle 可见性参数测试 ──────────────────────────
// Note: HandleCreateSkillBundle 依赖 SMH (getCommonStorageClient)，
// 无法在测试中轻易模拟，因此直接测试 DB 层创建+可见性设置的组合逻辑。

func TestCreateBundle_WithVisibilityGroup(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	group := model.UserGroup{Name: "创建测试组"}
	model.DB(context.Background()).Create(&group)

	var bundle model.SkillBundle
	model.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
		bundle = model.SkillBundle{
			Name:           "分组技能包",
			SkillCount:     0,
			Enabled:        false,
			VisibilityType: "group",
		}
		tx.Create(&bundle)
		return model.SetSkillBundleVisibility(tx, bundle.ID, "group", []uint{group.ID})
	})

	var created model.SkillBundle
	model.DB(context.Background()).First(&created, bundle.ID)
	if created.VisibilityType != "group" {
		t.Errorf("期望 visibility_type=group，实际=%s", created.VisibilityType)
	}

	var vgs []model.SkillBundleVisibilityGroup
	model.DB(context.Background()).Where("skill_bundle_id = ?", bundle.ID).Find(&vgs)
	if len(vgs) != 1 {
		t.Fatalf("期望 1 条关联，实际=%d", len(vgs))
	}
	if vgs[0].GroupID != group.ID {
		t.Errorf("期望 group_id=%d，实际=%d", group.ID, vgs[0].GroupID)
	}
}

func TestCreateBundle_WithVisibilityAll(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	var bundle model.SkillBundle
	model.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
		bundle = model.SkillBundle{
			Name:           "全局技能包",
			SkillCount:     0,
			Enabled:        false,
			VisibilityType: "all",
		}
		return tx.Create(&bundle).Error
	})

	var created model.SkillBundle
	model.DB(context.Background()).First(&created, bundle.ID)
	if created.VisibilityType != "all" {
		t.Errorf("期望 visibility_type=all，实际=%s", created.VisibilityType)
	}

	var count int64
	model.DB(context.Background()).Model(&model.SkillBundleVisibilityGroup{}).Where("skill_bundle_id = ?", bundle.ID).Count(&count)
	if count != 0 {
		t.Errorf("all 类型不应有分组关联，实际=%d", count)
	}
}

// ── 列表响应包含可见性分组信息测试（调用真实 handler） ──────────────

func TestListBundles_ResponseIncludesVisibleGroups(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	group := model.UserGroup{Name: "响应测试组"}
	model.DB(context.Background()).Create(&group)

	bundle := model.SkillBundle{Name: "带分组包", VisibilityType: "group"}
	model.DB(context.Background()).Create(&bundle)
	model.DB(context.Background()).Create(&model.SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: group.ID})

	w := httptest.NewRecorder()
	HandleAdminSkillBundles(w, adminBundleGet("/admin/skill-bundles"))

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	bundles := resp["skill_bundles"].([]interface{})
	if len(bundles) != 1 {
		t.Fatalf("期望 1 个技能包，实际=%d", len(bundles))
	}

	b := bundles[0].(map[string]interface{})
	visibleGroups := b["visible_groups"].([]interface{})
	if len(visibleGroups) != 1 {
		t.Fatalf("期望 1 个 visible_groups，实际=%d", len(visibleGroups))
	}

	vg := visibleGroups[0].(map[string]interface{})
	if vg["group_name"] != "响应测试组" {
		t.Errorf("期望 group_name=响应测试组，实际=%v", vg["group_name"])
	}
}

// ── HandleDeleteSkillBundle 级联清理可见性测试 ──────────────────────────

func TestDeleteBundle_CascadeVisibility(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	group := model.UserGroup{Name: "待删除组"}
	model.DB(context.Background()).Create(&group)
	bundle := model.SkillBundle{Name: "待删除包", Enabled: false, VisibilityType: "group"}
	model.DB(context.Background()).Create(&bundle)
	model.DB(context.Background()).Create(&model.SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: group.ID})

	// 确认关联存在
	var count int64
	model.DB(context.Background()).Model(&model.SkillBundleVisibilityGroup{}).Where("skill_bundle_id = ?", bundle.ID).Count(&count)
	if count != 1 {
		t.Fatalf("删除前应有 1 条关联，实际=%d", count)
	}

	req := adminBundlePostNoBody(fmt.Sprintf("/admin/skill-bundles/delete?id=%d", bundle.ID))
	w := httptest.NewRecorder()
	HandleDeleteSkillBundle(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 确认关联已级联删除
	model.DB(context.Background()).Model(&model.SkillBundleVisibilityGroup{}).Where("skill_bundle_id = ?", bundle.ID).Count(&count)
	if count != 0 {
		t.Errorf("删除后关联应为 0，实际=%d", count)
	}
}

func TestDeleteBundle_EnabledConflict(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	bundle := model.SkillBundle{Name: "生效中", Enabled: true, VisibilityType: "all"}
	model.DB(context.Background()).Create(&bundle)

	req := adminBundlePostNoBody(fmt.Sprintf("/admin/skill-bundles/delete?id=%d", bundle.ID))
	w := httptest.NewRecorder()
	HandleDeleteSkillBundle(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("期望 409（生效中不能删除），实际=%d", w.Code)
	}
}

func TestDeleteBundle_NotFound(t *testing.T) {
	setupBundleVisibilityTestDB(t)

	req := adminBundlePostNoBody("/admin/skill-bundles/delete?id=9999")
	w := httptest.NewRecorder()
	HandleDeleteSkillBundle(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}
