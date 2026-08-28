package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ─────────────────────────────────────────────────────────────────────────────
// helper
// ─────────────────────────────────────────────────────────────────────────────

func setupGroupPolicyDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试 DB 失败: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&model.User{},
		&model.SiteConfig{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.GroupClosure{},
		&model.MemoryPlanGroupPolicy{},
		&model.MemoryTDAIPlugin{},
		&model.Instance{},
	); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	oldSnap := common.FixedSnapshot
	if common.FixedSnapshot == nil {
		common.FixedSnapshot = &common.TenantSnapshot{}
	}
	t.Cleanup(func() {
		time.Sleep(50 * time.Millisecond)
		origDB()
		common.FixedSnapshot = oldSnap
	})
	// 默认 site_config：预设策略=off
	db.Create(&model.SiteConfig{
		MemoryTDAIEnable: true,
		MemoryDefaultPlan: "off",
	})
	// 初始化 session store
	if Store == nil {
		Store = sessions.NewCookieStore([]byte("test-secret-key-32bytes-padding!"))
	}
	return db
}

func gpAdminReq(method, path, body string) (*http.Request, *httptest.ResponseRecorder) {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req, httptest.NewRecorder()
}

func decodeResp(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v\nbody: %s", err, w.Body.String())
	}
	return result
}

// seedGroups 创建分组树：产品部(1) -> 设计组(2), 运营组(3) -> 运营一组(4)
func seedGroups(t *testing.T, db *gorm.DB) {
	t.Helper()
	db.Create(&model.UserGroup{ID: 1, Name: "产品部", ParentID: 0, Depth: 0, FullPath: "产品部"})
	db.Create(&model.UserGroup{ID: 2, Name: "设计组", ParentID: 1, Depth: 1, FullPath: "产品部/设计组"})
	db.Create(&model.UserGroup{ID: 3, Name: "运营组", ParentID: 1, Depth: 1, FullPath: "产品部/运营组"})
	db.Create(&model.UserGroup{ID: 4, Name: "运营一组", ParentID: 3, Depth: 2, FullPath: "产品部/运营组/运营一组"})

	// 闭包表
	closures := []model.GroupClosure{
		// 自指
		{AncestorID: 1, DescendantID: 1, Depth: 0},
		{AncestorID: 2, DescendantID: 2, Depth: 0},
		{AncestorID: 3, DescendantID: 3, Depth: 0},
		{AncestorID: 4, DescendantID: 4, Depth: 0},
		// 父子
		{AncestorID: 1, DescendantID: 2, Depth: 1},
		{AncestorID: 1, DescendantID: 3, Depth: 1},
		{AncestorID: 3, DescendantID: 4, Depth: 1},
		// 祖孙
		{AncestorID: 1, DescendantID: 4, Depth: 2},
	}
	for _, c := range closures {
		db.Create(&c)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// POST /admin/memory/group-policy 测试
// ─────────────────────────────────────────────────────────────────────────────

func TestCreateGroupPolicy_Success(t *testing.T) {
	db := setupGroupPolicyDB(t)
	seedGroups(t, db)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := gpAdminReq("POST", "/admin/memory/group-policy", `{"priority":1,"plan":"free","group_ids":[2,3]}`)
	HandleAdminMemoryGroupPolicy(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeResp(t, w)
	if resp["ok"] != true {
		t.Fatalf("expected ok=true, got %v", resp)
	}

	// 验证数据库
	var count int64
	db.Model(&model.MemoryPlanGroupPolicy{}).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 rows, got %d", count)
	}
}

func TestCreateGroupPolicy_PlanSameAsDefault(t *testing.T) {
	setupGroupPolicyDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 预设策略=off，尝试创建 plan=off 的策略
	req, w := gpAdminReq("POST", "/admin/memory/group-policy", `{"priority":1,"plan":"off","group_ids":[1]}`)
	HandleAdminMemoryGroupPolicy(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateGroupPolicy_InvalidPriority(t *testing.T) {
	setupGroupPolicyDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := gpAdminReq("POST", "/admin/memory/group-policy", `{"priority":3,"plan":"free","group_ids":[1]}`)
	HandleAdminMemoryGroupPolicy(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateGroupPolicy_EmptyGroupIDs(t *testing.T) {
	setupGroupPolicyDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := gpAdminReq("POST", "/admin/memory/group-policy", `{"priority":1,"plan":"free","group_ids":[]}`)
	HandleAdminMemoryGroupPolicy(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateGroupPolicy_PriorityAlreadyExists(t *testing.T) {
	db := setupGroupPolicyDB(t)
	seedGroups(t, db)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 先创建 priority=1
	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 2, Plan: "free", Priority: 1})

	// 再创建 priority=1 应该 409
	req, w := gpAdminReq("POST", "/admin/memory/group-policy", `{"priority":1,"plan":"pro","group_ids":[3]}`)
	HandleAdminMemoryGroupPolicy(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateGroupPolicy_GroupNotExists(t *testing.T) {
	setupGroupPolicyDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := gpAdminReq("POST", "/admin/memory/group-policy", `{"priority":1,"plan":"free","group_ids":[999]}`)
	HandleAdminMemoryGroupPolicy(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// PUT /admin/memory/group-policy 测试
// ─────────────────────────────────────────────────────────────────────────────

func TestUpdateGroupPolicy_Success(t *testing.T) {
	db := setupGroupPolicyDB(t)
	seedGroups(t, db)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 先创建 priority=1
	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 2, Plan: "free", Priority: 1})

	// PUT 全量替换
	req, w := gpAdminReq("PUT", "/admin/memory/group-policy", `{"priority":1,"plan":"pro","group_ids":[3,4]}`)
	HandleAdminMemoryGroupPolicy(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 验证旧数据被替换
	var policies []model.MemoryPlanGroupPolicy
	db.Where("priority = ?", 1).Find(&policies)
	if len(policies) != 2 {
		t.Fatalf("expected 2 rows after update, got %d", len(policies))
	}
	for _, p := range policies {
		if p.Plan != "pro" {
			t.Errorf("expected plan=pro, got %s", p.Plan)
		}
	}
}

func TestUpdateGroupPolicy_PriorityNotExists(t *testing.T) {
	setupGroupPolicyDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := gpAdminReq("PUT", "/admin/memory/group-policy", `{"priority":1,"plan":"free","group_ids":[1]}`)
	HandleAdminMemoryGroupPolicy(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateGroupPolicy_PlanConflictWithOther(t *testing.T) {
	db := setupGroupPolicyDB(t)
	seedGroups(t, db)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 创建两条策略
	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 2, Plan: "free", Priority: 1})
	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 3, Plan: "pro", Priority: 2})

	// 尝试把 priority=1 的 plan 改成 pro（跟 priority=2 冲突）
	req, w := gpAdminReq("PUT", "/admin/memory/group-policy", `{"priority":1,"plan":"pro","group_ids":[2]}`)
	HandleAdminMemoryGroupPolicy(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// POST /admin/memory/group-policy/delete 测试
// ─────────────────────────────────────────────────────────────────────────────

func TestDeleteGroupPolicy_Success(t *testing.T) {
	db := setupGroupPolicyDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 2, Plan: "free", Priority: 1})
	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 3, Plan: "free", Priority: 1})

	req, w := gpAdminReq("POST", "/admin/memory/group-policy/delete", `{"priority":1}`)
	HandleAdminMemoryGroupPolicyDelete(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var count int64
	db.Model(&model.MemoryPlanGroupPolicy{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 rows after delete, got %d", count)
	}
}

func TestDeleteGroupPolicy_Priority1_DegradesPriority2(t *testing.T) {
	db := setupGroupPolicyDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 2, Plan: "free", Priority: 1})
	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 3, Plan: "pro", Priority: 2})

	// 删除 priority=1
	req, w := gpAdminReq("POST", "/admin/memory/group-policy/delete", `{"priority":1}`)
	HandleAdminMemoryGroupPolicyDelete(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// priority=2 应该降级为 1
	var policy model.MemoryPlanGroupPolicy
	db.Where("group_id = ?", 3).First(&policy)
	if policy.Priority != 1 {
		t.Errorf("expected priority=1 after degrade, got %d", policy.Priority)
	}
}

func TestDeleteGroupPolicy_InvalidPriority(t *testing.T) {
	setupGroupPolicyDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := gpAdminReq("POST", "/admin/memory/group-policy/delete", `{"priority":5}`)
	HandleAdminMemoryGroupPolicyDelete(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// GET /admin/memory/group-policies 测试
// ─────────────────────────────────────────────────────────────────────────────

func TestGetGroupPolicies_Empty(t *testing.T) {
	setupGroupPolicyDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req, w := gpAdminReq("GET", "/admin/memory/group-policies", "")
	HandleAdminMemoryGroupPolicies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeResp(t, w)
	policies := resp["policies"].([]any)
	if len(policies) != 0 {
		t.Errorf("expected empty policies, got %d", len(policies))
	}
}

func TestGetGroupPolicies_WithData(t *testing.T) {
	db := setupGroupPolicyDB(t)
	seedGroups(t, db)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 2, Plan: "free", Priority: 1})
	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 3, Plan: "pro", Priority: 2})

	req, w := gpAdminReq("GET", "/admin/memory/group-policies", "")
	HandleAdminMemoryGroupPolicies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeResp(t, w)
	policies := resp["policies"].([]any)
	if len(policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(policies))
	}

	// 验证第一条策略
	p1 := policies[0].(map[string]any)
	if p1["plan"] != "free" {
		t.Errorf("policy1 plan=%v, want free", p1["plan"])
	}
	groups1 := p1["groups"].([]any)
	if len(groups1) != 1 {
		t.Errorf("policy1 groups count=%d, want 1", len(groups1))
	}
}

func TestGetGroupPolicies_FiltersDeletedGroups(t *testing.T) {
	db := setupGroupPolicyDB(t)
	seedGroups(t, db)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	// 创建策略引用 group_id=2 和 group_id=99（不存在的）
	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 2, Plan: "free", Priority: 1})
	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 99, Plan: "free", Priority: 1})

	req, w := gpAdminReq("GET", "/admin/memory/group-policies", "")
	HandleAdminMemoryGroupPolicies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeResp(t, w)
	policies := resp["policies"].([]any)
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy (filtered), got %d", len(policies))
	}
	p1 := policies[0].(map[string]any)
	groups := p1["groups"].([]any)
	if len(groups) != 1 {
		t.Errorf("expected 1 group (99 filtered out), got %d", len(groups))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// resolveMemoryPlanForGroup 测试
// ─────────────────────────────────────────────────────────────────────────────

func TestResolveMemoryPlanForGroup_NoGroupID(t *testing.T) {
	setupGroupPolicyDB(t)

	config := model.SiteConfig{
		MemoryTDAIEnable:  true,
		MemoryDefaultPlan: "off",
	}
	plan := resolveMemoryPlanForGroup(context.Background(), 0, config)
	if plan != "off" {
		t.Errorf("expected 'off' for groupID=0, got '%s'", plan)
	}
}

func TestResolveMemoryPlanForGroup_DirectMatch(t *testing.T) {
	db := setupGroupPolicyDB(t)
	seedGroups(t, db)

	// 策略选了"设计组"(id=2)，plan=free
	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 2, Plan: "free", Priority: 1})

	config := model.SiteConfig{
		MemoryTDAIEnable:  true,
		MemoryDefaultPlan: "off",
	}
	// 用户选了"设计组"(id=2) → 直接命中
	plan := resolveMemoryPlanForGroup(context.Background(), 2, config)
	if plan != "free" {
		t.Errorf("expected 'free', got '%s'", plan)
	}
}

func TestResolveMemoryPlanForGroup_AncestorMatch(t *testing.T) {
	db := setupGroupPolicyDB(t)
	seedGroups(t, db)

	// 策略选了"运营组"(id=3)，plan=pro
	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 3, Plan: "pro", Priority: 1})

	config := model.SiteConfig{
		MemoryTDAIEnable:  true,
		MemoryDefaultPlan: "off",
	}
	// 用户选了"运营一组"(id=4) → 祖先链 [4,3,1] → 命中 3
	plan := resolveMemoryPlanForGroup(context.Background(), 4, config)
	if plan != "pro" {
		t.Errorf("expected 'pro', got '%s'", plan)
	}
}

func TestResolveMemoryPlanForGroup_NoMatch_FallbackDefault(t *testing.T) {
	db := setupGroupPolicyDB(t)
	seedGroups(t, db)

	// 策略选了"设计组"(id=2)
	db.Create(&model.MemoryPlanGroupPolicy{GroupID: 2, Plan: "free", Priority: 1})

	config := model.SiteConfig{
		MemoryTDAIEnable:  true,
		MemoryDefaultPlan: "off",
	}
	// 用户选了"运营一组"(id=4) → 祖先链 [4,3,1] → 没有命中
	plan := resolveMemoryPlanForGroup(context.Background(), 4, config)
	if plan != "off" {
		t.Errorf("expected 'off' (default), got '%s'", plan)
	}
}

func TestResolveMemoryPlanForGroup_DefaultPlanFallback_WhenNoExplicitPlan(t *testing.T) {
	setupGroupPolicyDB(t)

	// MemoryDefaultPlan 为空但 MemoryTDAIEnable=true → fallback 到 free
	config := model.SiteConfig{
		MemoryTDAIEnable:  true,
		MemoryDefaultPlan: "",
	}
	plan := resolveMemoryPlanForGroup(context.Background(), 0, config)
	if plan != "free" {
		t.Errorf("expected 'free' (fallback), got '%s'", plan)
	}
}
