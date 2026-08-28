package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/gorilla/sessions"
)

// ── 测试辅助 ──────────────────────────────────────────────────────────

// setupSkillStoreTestDB 初始化内存 SQLite，包含技能广场所需的全部表。
func setupSkillStoreTestDB(t *testing.T) {
	t.Helper()
	setupSkillInstancesDB(t) // 复用已有的完整 setup（含 Instance/User/Skill/Task/Record 等）

	// 补充 setupSkillInstancesDB 中未包含的表
	model.DB(context.Background()).AutoMigrate(
		&model.SkillCategory{},
		&model.SkillCategoryMapping{},
		&model.SkillVisibilityGroup{},
	)

	// 确保 Store 已初始化（session 认证需要）
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	t.Cleanup(func() {
		Store = origStore
	})
}

// skillStoreReq 创建带 session 认证的 JSON GET 请求
func skillStoreReq(t *testing.T, path, username string) *http.Request {
	t.Helper()
	return openclawReqWithSession(t, http.MethodGet, path, username, "")
}

// skillStorePostReq 创建带 session 认证的 JSON POST 请求
func skillStorePostReq(t *testing.T, path, username, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

// ── HandleSkillStore 列表接口测试 ────────────────────────────────────

func TestHandleSkillStore_Unauthorized(t *testing.T) {
	setupSkillStoreTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/skillstore", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleSkillStore(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("未登录应返回 401，实际=%d", rr.Code)
	}
}

func TestHandleSkillStore_EmptyList(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "store-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := skillStoreReq(t, "/openclaw/skillstore", "store-user")
	rr := httptest.NewRecorder()
	HandleSkillStore(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["total"].(float64) != 0 {
		t.Errorf("空列表 total 应为 0，实际=%v", resp["total"])
	}
}

func TestHandleSkillStore_VisibilityFilter_AllVisible(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "vis-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	// 创建 visibility_type=all 的技能
	skill := model.Skill{
		Slug: "public-skill", Name: "公开技能", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skill)

	req := skillStoreReq(t, "/openclaw/skillstore", "vis-user")
	rr := httptest.NewRecorder()
	HandleSkillStore(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["total"].(float64) != 1 {
		t.Errorf("应返回 1 个公开技能，实际=%v", resp["total"])
	}
}

func TestHandleSkillStore_VisibilityFilter_GroupNotVisible(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "nogroup-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	// 创建 visibility_type=group 的技能（用户不在任何分组中）
	skill := model.Skill{
		Slug: "group-skill", Name: "分组技能", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "group",
	}
	model.DB(context.Background()).Create(&skill)

	req := skillStoreReq(t, "/openclaw/skillstore", "nogroup-user")
	rr := httptest.NewRecorder()
	HandleSkillStore(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["total"].(float64) != 0 {
		t.Errorf("不在分组中的用户不应看到 group 技能，total=%v", resp["total"])
	}
}

func TestHandleSkillStore_KeywordSearch(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "search-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	model.DB(context.Background()).Create(&model.Skill{
		Slug: "aippt", Name: "PPT助手", Description: "生成PPT", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "all",
	})
	model.DB(context.Background()).Create(&model.Skill{
		Slug: "code-review", Name: "代码审查", Description: "review code", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "all",
	})

	req := skillStoreReq(t, "/openclaw/skillstore?keyword=PPT", "search-user")
	rr := httptest.NewRecorder()
	HandleSkillStore(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["total"].(float64) != 1 {
		t.Errorf("搜索 PPT 应返回 1 条，实际=%v", resp["total"])
	}
}

func TestHandleSkillStore_SortByDownloads(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "sort-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	model.DB(context.Background()).Create(&model.Skill{
		Slug: "low-dl", Name: "低下载", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "all", DistributeCount: 5,
	})
	model.DB(context.Background()).Create(&model.Skill{
		Slug: "high-dl", Name: "高下载", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "all", DistributeCount: 100,
	})

	req := skillStoreReq(t, "/openclaw/skillstore?sort=downloads", "sort-user")
	rr := httptest.NewRecorder()
	HandleSkillStore(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	skills := resp["skills"].([]interface{})
	if len(skills) != 2 {
		t.Fatalf("应返回 2 条，实际=%d", len(skills))
	}
	first := skills[0].(map[string]interface{})
	if first["slug"].(string) != "high-dl" {
		t.Errorf("按下载量排序第一条应是 high-dl，实际=%s", first["slug"])
	}
}

func TestHandleSkillStore_CategoryFilter(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "cat-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	cat := model.SkillCategory{Name: "测试分类"}
	model.DB(context.Background()).Create(&cat)

	s1 := model.Skill{Slug: "cat-skill", Name: "分类技能", Version: "1.0.0", VersionMajor: 1, VisibilityType: "all"}
	model.DB(context.Background()).Create(&s1)
	model.DB(context.Background()).Create(&model.SkillCategoryMapping{SkillID: s1.ID, CategoryID: cat.ID})

	s2 := model.Skill{Slug: "nocat-skill", Name: "无分类技能", Version: "1.0.0", VersionMajor: 1, VisibilityType: "all"}
	model.DB(context.Background()).Create(&s2)

	req := skillStoreReq(t, "/openclaw/skillstore?category_ids="+uintStr(cat.ID), "cat-user")
	rr := httptest.NewRecorder()
	HandleSkillStore(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["total"].(float64) != 1 {
		t.Errorf("分类筛选应返回 1 条，实际=%v", resp["total"])
	}
}

func TestHandleSkillStore_Pagination(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "page-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	for i := 0; i < 5; i++ {
		model.DB(context.Background()).Create(&model.Skill{
			Slug: "page-skill-" + uintStr(uint(i)), Name: "分页测试", Version: "1.0.0",
			VersionMajor: 1, VisibilityType: "all",
		})
	}

	req := skillStoreReq(t, "/openclaw/skillstore?page=1&page_size=2", "page-user")
	rr := httptest.NewRecorder()
	HandleSkillStore(rr, req)

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	skills := resp["skills"].([]interface{})
	if len(skills) != 2 {
		t.Errorf("page_size=2 应返回 2 条，实际=%d", len(skills))
	}
	if resp["total"].(float64) != 5 {
		t.Errorf("total 应为 5，实际=%v", resp["total"])
	}
}

// ── HandleSkillStoreDetail 详情接口测试 ──────────────────────────────

func TestHandleSkillStoreDetail_NotFound(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "detail-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := skillStoreReq(t, "/openclaw/skillstore/detail?slug=nonexist", "detail-user")
	rr := httptest.NewRecorder()
	HandleSkillStoreDetail(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("不存在的 slug 应返回 404，实际=%d", rr.Code)
	}
}

func TestHandleSkillStoreDetail_MissingSlug(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "detail-user2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := skillStoreReq(t, "/openclaw/skillstore/detail", "detail-user2")
	rr := httptest.NewRecorder()
	HandleSkillStoreDetail(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少 slug 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleSkillStoreDetail_Success(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "detail-ok", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	skill := model.Skill{
		Slug: "detail-test", Name: "详情测试", Version: "2.0.0",
		VersionMajor: 2, VisibilityType: "all", Description: "desc",
		FileList: `["detail-test/SKILL.md"]`, DistributeCount: 42,
	}
	model.DB(context.Background()).Create(&skill)

	req := skillStoreReq(t, "/openclaw/skillstore/detail?slug=detail-test", "detail-ok")
	rr := httptest.NewRecorder()
	HandleSkillStoreDetail(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	skillData := resp["skill"].(map[string]interface{})
	if skillData["slug"].(string) != "detail-test" {
		t.Errorf("slug 不匹配: %v", skillData["slug"])
	}
	if skillData["distribute_count"].(float64) != 42 {
		t.Errorf("distribute_count 应为 42，实际=%v", skillData["distribute_count"])
	}
	if resp["versions"] == nil {
		t.Error("应返回 versions 字段")
	}
}

func TestHandleSkillStoreDetail_VisibilityBlocked(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "blocked-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	skill := model.Skill{
		Slug: "secret-skill", Name: "保密技能", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "group",
	}
	model.DB(context.Background()).Create(&skill)

	req := skillStoreReq(t, "/openclaw/skillstore/detail?slug=secret-skill", "blocked-user")
	rr := httptest.NewRecorder()
	HandleSkillStoreDetail(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("不可见技能应返回 404，实际=%d", rr.Code)
	}
}

func TestHandleSkillStoreDetail_SpecificVersion(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "ver-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	model.DB(context.Background()).Create(&model.Skill{
		Slug: "ver-skill", Name: "版本测试", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "all",
	})
	model.DB(context.Background()).Create(&model.Skill{
		Slug: "ver-skill", Name: "版本测试", Version: "2.0.0",
		VersionMajor: 2, VisibilityType: "all",
	})

	req := skillStoreReq(t, "/openclaw/skillstore/detail?slug=ver-skill&version=1.0.0", "ver-user")
	rr := httptest.NewRecorder()
	HandleSkillStoreDetail(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	skillData := resp["skill"].(map[string]interface{})
	if skillData["version"].(string) != "1.0.0" {
		t.Errorf("指定 version=1.0.0 但返回了 %s", skillData["version"])
	}
}

// ── HandleSkillStoreCategories 分类接口测试 ──────────────────────────

func TestHandleSkillStoreCategories_Success(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "cat-list-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	model.DB(context.Background()).Create(&model.SkillCategory{Name: "分类A"})
	model.DB(context.Background()).Create(&model.SkillCategory{Name: "分类B"})

	req := skillStoreReq(t, "/openclaw/skillstore/categories", "cat-list-user")
	rr := httptest.NewRecorder()
	HandleSkillStoreCategories(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	cats := resp["categories"].([]interface{})
	if len(cats) < 2 {
		t.Errorf("应至少返回 2 个分类，实际=%d", len(cats))
	}
}

// ── HandleSkillStoreInstances 实例状态接口测试 ───────────────────────

func TestHandleSkillStoreInstances_MissingSlug(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "inst-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := skillStoreReq(t, "/openclaw/skillstore/instances", "inst-user")
	rr := httptest.NewRecorder()
	HandleSkillStoreInstances(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少 slug 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleSkillStoreInstances_SkillNotFound(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "inst-user2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := skillStoreReq(t, "/openclaw/skillstore/instances?slug=nonexist", "inst-user2")
	rr := httptest.NewRecorder()
	HandleSkillStoreInstances(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("不存在技能应返回 404，实际=%d", rr.Code)
	}
}

func TestHandleSkillStoreInstances_OnlyUserInstances(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "my-inst-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	otherUser := model.User{Username: "other-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&otherUser)

	skill := model.Skill{
		Slug: "inst-skill", Name: "实例测试", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skill)

	// 当前用户的实例
	model.DB(context.Background()).Create(&model.Instance{
		Name: "我的实例", InstanceId: "ins-mine-001", UserID: user.ID, AgentType: "openclaw",
	})
	// 其他用户的实例
	model.DB(context.Background()).Create(&model.Instance{
		Name: "别人的实例", InstanceId: "ins-other-001", UserID: otherUser.ID, AgentType: "openclaw",
	})

	req := skillStoreReq(t, "/openclaw/skillstore/instances?slug=inst-skill&status=uninstalled", "my-inst-user")
	rr := httptest.NewRecorder()
	HandleSkillStoreInstances(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	// 注意：返回的实例可能为空（因为只返回 running 状态的），但不应包含其他用户的
	// 这里主要验证接口不报错
}

// ── HandleSkillStoreTasks 下发记录测试 ───────────────────────────────

func TestHandleSkillStoreTasks_MissingSlug(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "task-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := skillStoreReq(t, "/openclaw/skillstore/tasks", "task-user")
	rr := httptest.NewRecorder()
	HandleSkillStoreTasks(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少 slug 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleSkillStoreTasks_EmptyRecords(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "task-user2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	skill := model.Skill{
		Slug: "task-skill", Name: "记录测试", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skill)

	req := skillStoreReq(t, "/openclaw/skillstore/tasks?slug=task-skill", "task-user2")
	rr := httptest.NewRecorder()
	HandleSkillStoreTasks(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["total"].(float64) != 0 {
		t.Errorf("无记录时 total 应为 0，实际=%v", resp["total"])
	}
}

func TestHandleSkillStoreTasks_WithRecords(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "task-record-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	skill := model.Skill{
		Slug: "task-rec-skill", Name: "记录查询", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skill)

	inst := model.Instance{
		Name: "我的实例", InstanceId: "ins-task-001", UserID: user.ID, AgentType: "openclaw",
	}
	model.DB(context.Background()).Create(&inst)

	task := model.SkillDistributionTask{
		SkillID: skill.ID, Version: "1.0.0", OperatorID: user.ID,
		Total: 1, Status: "completed",
	}
	model.DB(context.Background()).Create(&task)

	record := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: skill.ID, InstanceID: inst.ID,
		InstanceCID: "ins-task-001", Version: "1.0.0", Status: "success",
	}
	model.DB(context.Background()).Create(&record)

	req := skillStoreReq(t, "/openclaw/skillstore/tasks?slug=task-rec-skill", "task-record-user")
	rr := httptest.NewRecorder()
	HandleSkillStoreTasks(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["total"].(float64) != 1 {
		t.Errorf("应有 1 条任务记录，实际=%v", resp["total"])
	}
	tasks := resp["tasks"].([]interface{})
	taskData := tasks[0].(map[string]interface{})
	if taskData["success"].(float64) != 1 {
		t.Errorf("success 应为 1，实际=%v", taskData["success"])
	}
	records := taskData["records"].([]interface{})
	if len(records) != 1 {
		t.Errorf("应有 1 条 record，实际=%d", len(records))
	}
}

func TestHandleSkillStoreTasks_VisibilityBlocked(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "task-blocked-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	skill := model.Skill{
		Slug: "task-secret", Name: "保密任务", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "group",
	}
	model.DB(context.Background()).Create(&skill)

	req := skillStoreReq(t, "/openclaw/skillstore/tasks?slug=task-secret", "task-blocked-user")
	rr := httptest.NewRecorder()
	HandleSkillStoreTasks(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("不可见技能的 tasks 应返回 404，实际=%d", rr.Code)
	}
}

// ── HandleSkillStoreDistribute 下发接口测试 ──────────────────────────

func TestHandleSkillStoreDistribute_MissingSlug(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "dist-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := skillStorePostReq(t, "/openclaw/skillstore/distribute", "dist-user", `{"instance_ids":[1]}`)
	rr := httptest.NewRecorder()
	HandleSkillStoreDistribute(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少 slug 应返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSkillStoreDistribute_MissingInstances(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "dist-user2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := skillStorePostReq(t, "/openclaw/skillstore/distribute", "dist-user2", `{"slug":"test"}`)
	rr := httptest.NewRecorder()
	HandleSkillStoreDistribute(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少 instance_ids 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleSkillStoreDistribute_SkillNotFound(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "dist-user3", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := skillStorePostReq(t, "/openclaw/skillstore/distribute", "dist-user3",
		`{"slug":"nonexist","instance_ids":[1]}`)
	rr := httptest.NewRecorder()
	HandleSkillStoreDistribute(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("不存在技能应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleSkillStoreDistribute_ForbiddenOtherInstance(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "dist-user4", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	otherUser := model.User{Username: "dist-other", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&otherUser)

	skill := model.Skill{
		Slug: "dist-skill", Name: "下发测试", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skill)

	otherInst := model.Instance{
		Name: "别人实例", InstanceId: "ins-other-002", UserID: otherUser.ID, AgentType: "openclaw",
	}
	model.DB(context.Background()).Create(&otherInst)

	req := skillStorePostReq(t, "/openclaw/skillstore/distribute", "dist-user4",
		`{"slug":"dist-skill","instance_ids":[`+uintStr(otherInst.ID)+`]}`)
	rr := httptest.NewRecorder()
	HandleSkillStoreDistribute(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("下发他人实例应返回 403，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSkillStoreDistribute_VisibilityBlocked(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "dist-vis-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	skill := model.Skill{
		Slug: "dist-secret", Name: "保密下发", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "group",
	}
	model.DB(context.Background()).Create(&skill)

	inst := model.Instance{
		Name: "我的实例", InstanceId: "ins-dist-vis-001", UserID: user.ID, AgentType: "openclaw",
	}
	model.DB(context.Background()).Create(&inst)

	req := skillStorePostReq(t, "/openclaw/skillstore/distribute", "dist-vis-user",
		`{"slug":"dist-secret","instance_ids":[`+uintStr(inst.ID)+`]}`)
	rr := httptest.NewRecorder()
	HandleSkillStoreDistribute(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("不可见技能下发应返回 404，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSkillStoreDistribute_VersionNotFound(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "dist-ver-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	skill := model.Skill{
		Slug: "dist-ver-skill", Name: "版本测试", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skill)

	req := skillStorePostReq(t, "/openclaw/skillstore/distribute", "dist-ver-user",
		`{"slug":"dist-ver-skill","version":"9.9.9","instance_ids":[1]}`)
	rr := httptest.NewRecorder()
	HandleSkillStoreDistribute(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("不存在版本应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleSkillStoreDistribute_InvalidJSON(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "dist-json-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := skillStorePostReq(t, "/openclaw/skillstore/distribute", "dist-json-user", `{invalid}`)
	rr := httptest.NewRecorder()
	HandleSkillStoreDistribute(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("无效 JSON 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleSkillStoreDistribute_UnsupportedAgentType(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "dist-agent-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	skill := model.Skill{
		Slug: "dist-agent-skill", Name: "类型测试", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skill)

	inst := model.Instance{
		Name: "不支持实例", InstanceId: "ins-unsupport-001", UserID: user.ID, AgentType: "unknown_type",
	}
	model.DB(context.Background()).Create(&inst)

	req := skillStorePostReq(t, "/openclaw/skillstore/distribute", "dist-agent-user",
		`{"slug":"dist-agent-skill","instance_ids":[`+uintStr(inst.ID)+`]}`)
	rr := httptest.NewRecorder()
	HandleSkillStoreDistribute(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("不支持的 agent_type 应返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

// ── HandleSkillStoreDownload 下载接口测试 ────────────────────────────

func TestHandleSkillStoreDownload_MissingSlug(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "dl-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := skillStoreReq(t, "/openclaw/skillstore/download", "dl-user")
	rr := httptest.NewRecorder()
	HandleSkillStoreDownload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少 slug 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleSkillStoreDownload_SkillNotFound(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "dl-user2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := skillStoreReq(t, "/openclaw/skillstore/download?slug=nonexist", "dl-user2")
	rr := httptest.NewRecorder()
	HandleSkillStoreDownload(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("不存在技能应返回 404，实际=%d", rr.Code)
	}
}

func TestHandleSkillStoreDownload_VisibilityBlocked(t *testing.T) {
	setupSkillStoreTestDB(t)

	user := model.User{Username: "dl-user3", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	model.DB(context.Background()).Create(&model.Skill{
		Slug: "dl-secret", Name: "保密下载", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "group",
	})

	req := skillStoreReq(t, "/openclaw/skillstore/download?slug=dl-secret", "dl-user3")
	rr := httptest.NewRecorder()
	HandleSkillStoreDownload(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("不可见技能下载应返回 404，实际=%d", rr.Code)
	}
}

// ── HandleAdminSkillDownload 管控端下载测试 ──────────────────────────

func TestHandleAdminSkillDownload_MissingSlug(t *testing.T) {
	setupSkillStoreTestDB(t)

	req := adminJSONGet("/admin/skills/download")
	rr := httptest.NewRecorder()
	HandleAdminSkillDownload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少 slug 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAdminSkillDownload_SkillNotFound(t *testing.T) {
	setupSkillStoreTestDB(t)

	req := adminJSONGet("/admin/skills/download?slug=nonexist")
	rr := httptest.NewRecorder()
	HandleAdminSkillDownload(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("不存在技能应返回 404，实际=%d", rr.Code)
	}
}

// TestHandleSkillStoreDistribute_LocalAndCVM_BothLanded 覆盖本次需求里的拆分逻辑：
//   - CVM 实例和 local 实例同时下发到一个 task
//   - records 表两行都创建（status=pending）
//   - local 那条 record 不交给 executor，executor 不报错（也不挂断）
//   - 接口返回 200
//
// HandleSkillStoreDistribute 之前覆盖率 52.3%，主要差在 happy path 没测。
// 这个 test 走完 select instances → split srcMap → create records → executor 派发
// 整条链路。executor 内部对 CVM 实例会调 TAT，会失败，但 finalize 会被异步
// goroutine 跑，不影响接口返回。
func TestHandleSkillStoreDistribute_LocalAndCVM_BothLanded(t *testing.T) {
	setupSkillStoreTestDB(t)
	ctx := context.Background()

	user := model.User{Username: "dist-mix-user", Password: "x", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	skill := model.Skill{
		Slug: "dist-mix-skill", Name: "混合下发", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: "all",
	}
	if err := model.DB(ctx).Create(&skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}

	cvmInst := model.Instance{
		Name: "cvm-inst", InstanceId: "ins-cvm-001",
		UserID: user.ID, Source: model.InstanceSourceCVM, AgentType: "openclaw",
	}
	if err := model.DB(ctx).Create(&cvmInst).Error; err != nil {
		t.Fatalf("create cvm: %v", err)
	}
	localInst := model.Instance{
		Name: "local-inst", InstanceId: "local-codebuddy-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "openclaw",
	}
	if err := model.DB(ctx).Create(&localInst).Error; err != nil {
		t.Fatalf("create local: %v", err)
	}

	body := `{"slug":"dist-mix-skill","instance_ids":[` + uintStr(cvmInst.ID) + `,` + uintStr(localInst.ID) + `]}`
	req := skillStorePostReq(t, "/openclaw/skillstore/distribute", "dist-mix-user", body)
	rr := httptest.NewRecorder()
	HandleSkillStoreDistribute(rr, req)

	// 接口同步返回 200（即使 CVM executor 异步出错，主路径仍返回成功）
	if rr.Code != http.StatusOK {
		t.Fatalf("混合下发应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 等异步 goroutine 退出
	time.Sleep(200 * time.Millisecond)

	// records 表两行都应创建（local + cvm），都是 distribute / pending or failed
	var records []model.SkillDistributionRecord
	if err := model.DB(ctx).Where("instance_id IN ?", []uint{cvmInst.ID, localInst.ID}).
		Find(&records).Error; err != nil {
		t.Fatalf("query records: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("应创建 2 条 record（CVM + local），实际=%d", len(records))
	}
	for _, rec := range records {
		if rec.Type != "distribute" {
			t.Errorf("record type 应=distribute，实际=%q (instance_id=%d)", rec.Type, rec.InstanceID)
		}
	}

	// task 应为 distribute 类型
	var task model.SkillDistributionTask
	if err := model.DB(ctx).First(&task, records[0].TaskID).Error; err != nil {
		t.Fatalf("query task: %v", err)
	}
	if task.Type != "distribute" {
		t.Errorf("task type 应=distribute，实际=%q", task.Type)
	}
	if task.Total != 2 {
		t.Errorf("task total 应=2，实际=%d", task.Total)
	}
}
