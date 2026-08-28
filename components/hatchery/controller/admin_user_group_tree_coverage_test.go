package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hatchery/controller/usergroup"
	"hatchery/model"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// setupTreeTestDB 初始化内存 SQLite 并迁移树相关表
func setupTreeTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.GroupClosure{},
		&model.GroupConfigBinding{},
		&model.Instance{},
		&model.SiteConfig{},
		&model.ResourcePolicy{},
		&model.AIModel{},
		&model.AIChannel{},
		&model.AIImage{},
		&model.VpcConfig{},
		&model.SkillBundle{},
		&model.SkillBundleVisibilityGroup{},
		&model.ModelVisibilityGroup{},
		&model.SkillVisibilityGroup{},
		&model.RoleVisibilityGroup{},
		&model.OpenClawRole{},
		&model.McpServer{},
		&model.PluginBundle{},
		&model.Plugin{},
		&model.Skill{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	unlock := model.UseDBForTest(db)
	AdminToken = "test-admin-token"
	// 初始化 session Store 避免 requireAdmin nil 解引用
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	// 创建默认 SiteConfig
	db.Create(&model.SiteConfig{})
	t.Cleanup(func() {
		unlock()
		model.UseNilDBForTest()()
	})
}

// createGroupWithClosure 在测试数据库中创建分组并维护闭包表
func createGroupWithClosure(t *testing.T, id uint, parentID uint, name string) {
	t.Helper()
	model.DB(context.Background()).Create(&model.UserGroup{ID: id, Name: name, ParentID: parentID, Source: "manual", FullPath: name})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: id, DescendantID: id, Depth: 0})
	if parentID > 0 {
		var parentClosures []model.GroupClosure
		model.DB(context.Background()).Where("descendant_id = ?", parentID).Find(&parentClosures)
		for _, c := range parentClosures {
			model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: c.AncestorID, DescendantID: id, Depth: c.Depth + 1})
		}
	}
}

// adminTreeReq 构造管理员请求
func adminTreeReq(method, path string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// seedTreeData 创建测试用的分组树数据
func seedTreeData(t *testing.T) {
	t.Helper()
	// 创建根组
	root := model.UserGroup{Name: "根组", FullPath: "根组", Source: "manual"}
	model.DB(context.Background()).Create(&root)
	// 创建子组
	child := model.UserGroup{Name: "研发组", FullPath: "根组/研发组", Source: "manual", ParentID: root.ID}
	model.DB(context.Background()).Create(&child)
	// 创建孙组
	grandchild := model.UserGroup{Name: "前端组", FullPath: "根组/研发组/前端组", Source: "manual", ParentID: child.ID}
	model.DB(context.Background()).Create(&grandchild)

	// 创建 closure 关系
	closures := []model.GroupClosure{
		{AncestorID: root.ID, DescendantID: root.ID, Depth: 0},
		{AncestorID: child.ID, DescendantID: child.ID, Depth: 0},
		{AncestorID: grandchild.ID, DescendantID: grandchild.ID, Depth: 0},
		{AncestorID: root.ID, DescendantID: child.ID, Depth: 1},
		{AncestorID: root.ID, DescendantID: grandchild.ID, Depth: 2},
		{AncestorID: child.ID, DescendantID: grandchild.ID, Depth: 1},
	}
	for _, c := range closures {
		model.DB(context.Background()).Create(&c)
	}

	// 创建用户和成员关系
	user1 := model.User{Username: "user1", Password: "hash", Role: "user"}
	model.DB(context.Background()).Create(&user1)
	user2 := model.User{Username: "user2", Password: "hash", Role: "user"}
	model.DB(context.Background()).Create(&user2)
	user3 := model.User{Username: "user3", Password: "hash", Role: "user"}
	model.DB(context.Background()).Create(&user3)

	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: child.ID, UserID: user1.ID, Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: child.ID, UserID: user2.ID, Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: grandchild.ID, UserID: user3.ID, Source: "manual"})
}

// ── HandleAdminGetGroupTree 测试 ───────────────────────────────────────────

func TestCoverageGetGroupTree_Success(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	w := httptest.NewRecorder()
	HandleAdminGetGroupTree(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/tree?with_user_counts=true", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Error("期望 ok=true")
	}
}

func TestCoverageGetGroupTree_WithQuery(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	w := httptest.NewRecorder()
	HandleAdminGetGroupTree(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/tree?q=研发&with_user_counts=true", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageGetGroupTree_WithSources(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	w := httptest.NewRecorder()
	HandleAdminGetGroupTree(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/tree?sources=manual", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageGetGroupTree_WithoutUserCounts(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	w := httptest.NewRecorder()
	HandleAdminGetGroupTree(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/tree?with_user_counts=false", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageGetGroupTree_MethodNotAllowed(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleAdminGetGroupTree(w, adminTreeReq(http.MethodPost,
		"/admin/user-groups/tree", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

func TestCoverageGetGroupTree_Unauthorized(t *testing.T) {
	setupTreeTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/user-groups/tree", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-token")

	w := httptest.NewRecorder()
	HandleAdminGetGroupTree(w, req)

	if w.Code == http.StatusOK {
		t.Error("期望非 200 响应（未授权）")
	}
}

func TestCoverageGetGroupTree_EmptyDB(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleAdminGetGroupTree(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/tree", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageGetGroupTree_WithHealth(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	w := httptest.NewRecorder()
	HandleAdminGetGroupTree(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/tree?with_health=true", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ── HandleAdminGetGroupMembers 测试 ────────────────────────────────────────

func TestCoverageGetGroupMembers_Success(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	// 获取研发组 ID
	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/members?id="+itoa(child.ID), nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Error("期望 ok=true")
	}
}

func TestCoverageGetGroupMembers_WithDescendants(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/members?id="+itoa(child.ID)+"&include_descendants=true", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageGetGroupMembers_WithQuery(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/members?id="+itoa(child.ID)+"&q=user1", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageGetGroupMembers_MissingID(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/members", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageGetGroupMembers_InvalidID(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/members?id=abc", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageGetGroupMembers_ZeroID(t *testing.T) {
	setupTreeTestDB(t)

	// id=0 代表未分组用户，应返回 200
	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/members?id=0", nil))

	if w.Code != http.StatusOK {
		t.Errorf("id=0 应返回未分组用户(200)，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageGetGroupMembers_MethodNotAllowed(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, adminTreeReq(http.MethodPost,
		"/admin/user-groups/members", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

func TestCoverageGetGroupMembers_Pagination(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/members?id="+itoa(child.ID)+"&page=1&page_size=1", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["page_size"].(float64) != 1 {
		t.Error("期望 page_size=1")
	}
}

func TestGetGroupMembers_PageSizeCappedAt200(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)
	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/members?id="+itoa(group.ID)+"&page_size=201", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["page_size"] != float64(200) {
		t.Fatalf("page_size=%v, want 200", resp["page_size"])
	}
}

// ── 未分组用户通过 id=0 查询测试 ─────────────────────────────────────

func TestCoverageGetMembers_Ungrouped_Success(t *testing.T) {
	setupTreeTestDB(t)
	// 创建一些没有分组的用户
	model.DB(context.Background()).Create(&model.User{Username: "lonely1", Password: "hash", Role: "user"})
	model.DB(context.Background()).Create(&model.User{Username: "lonely2", Password: "hash", Role: "user"})

	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/members?id=0", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Error("期望 ok=true")
	}
	if int(resp["total"].(float64)) < 2 {
		t.Errorf("期望至少 2 个未分组用户，实际 total=%v", resp["total"])
	}
}

func TestCoverageGetMembers_Ungrouped_WithQuery(t *testing.T) {
	setupTreeTestDB(t)
	model.DB(context.Background()).Create(&model.User{Username: "lonely1", Password: "hash", Role: "user"})
	model.DB(context.Background()).Create(&model.User{Username: "admin", Password: "hash", Role: "admin"})

	w := httptest.NewRecorder()
	HandleAdminGetGroupMembers(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/members?id=0&q=lonely&page=1&page_size=10", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ── HandleAdminGetGroupConfigOverview 测试 ─────────────────────────────────

func TestCoverageGetGroupConfigOverview_Success(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	w := httptest.NewRecorder()
	HandleAdminGetGroupConfigOverview(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/config-overview?group_ids="+itoa(child.ID), nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Error("期望 ok=true")
	}
	results := resp["results"].([]interface{})
	if len(results) == 0 {
		t.Error("期望至少有一个结果")
	}
}

func TestCoverageGetGroupConfigOverview_MultipleGroups(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var child, grandchild model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)
	model.DB(context.Background()).Where("name = ?", "前端组").First(&grandchild)

	w := httptest.NewRecorder()
	HandleAdminGetGroupConfigOverview(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/config-overview?group_ids="+itoa(child.ID)+","+itoa(grandchild.ID), nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	results := resp["results"].([]interface{})
	if len(results) != 2 {
		t.Errorf("期望 2 个结果，实际=%d", len(results))
	}
}

func TestCoverageGetGroupConfigOverview_WithKeys(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	w := httptest.NewRecorder()
	HandleAdminGetGroupConfigOverview(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/config-overview?group_ids="+itoa(child.ID)+"&keys=model,channel", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	results := resp["results"].([]interface{})
	item := results[0].(map[string]interface{})
	categories := item["categories"].([]interface{})
	if len(categories) != 2 {
		t.Errorf("期望 2 个 categories（model 和 channel），实际=%d", len(categories))
	}
}

func TestCoverageGetGroupConfigOverview_MissingGroupIDs(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleAdminGetGroupConfigOverview(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/config-overview", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageGetGroupConfigOverview_InvalidGroupIDs(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleAdminGetGroupConfigOverview(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/config-overview?group_ids=abc", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageGetGroupConfigOverview_GroupNotExist(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleAdminGetGroupConfigOverview(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/config-overview?group_ids=99999", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageGetGroupConfigOverview_InvalidKey(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	w := httptest.NewRecorder()
	HandleAdminGetGroupConfigOverview(w, adminTreeReq(http.MethodGet,
		"/admin/user-groups/config-overview?group_ids="+itoa(child.ID)+"&keys=invalid_key", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageGetGroupConfigOverview_MethodNotAllowed(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleAdminGetGroupConfigOverview(w, adminTreeReq(http.MethodPost,
		"/admin/user-groups/config-overview", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

// ── parseCSVQuery 测试 ─────────────────────────────────────────────────────

func TestCoverageParseCSVQuery_EmptyParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	result := parseCSVQuery(req, "sources")
	if result != nil {
		t.Errorf("期望 nil，实际=%v", result)
	}
}

func TestCoverageParseCSVQuery_SingleValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?sources=manual", nil)
	result := parseCSVQuery(req, "sources")
	if len(result) != 1 || result[0] != "manual" {
		t.Errorf("期望 [manual]，实际=%v", result)
	}
}

func TestCoverageParseCSVQuery_MultipleValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?sources=manual,oneid_dept", nil)
	result := parseCSVQuery(req, "sources")
	if len(result) != 2 {
		t.Errorf("期望 2 个值，实际=%d", len(result))
	}
}

func TestCoverageParseCSVQuery_WithSpaces(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?sources=+manual+,+oneid_dept+", nil)
	result := parseCSVQuery(req, "sources")
	if len(result) != 2 {
		t.Errorf("期望 2 个值，实际=%d", len(result))
	}
}

func TestCoverageParseCSVQuery_AllEmpty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?sources=,,", nil)
	result := parseCSVQuery(req, "sources")
	if result != nil {
		t.Errorf("期望 nil，实际=%v", result)
	}
}

// ── parseBoolQuery / parseBoolQueryDefault 测试 ────────────────────────────

func TestCoverageParseBoolQuery_True(t *testing.T) {
	cases := []string{"1", "true", "yes", "on"}
	for _, v := range cases {
		req := httptest.NewRequest(http.MethodGet, "/test?flag="+v, nil)
		if !parseBoolQuery(req, "flag") {
			t.Errorf("parseBoolQuery(%q) should return true", v)
		}
	}
}

func TestCoverageParseBoolQuery_False(t *testing.T) {
	cases := []string{"0", "false", "no", "off", "other"}
	for _, v := range cases {
		req := httptest.NewRequest(http.MethodGet, "/test?flag="+v, nil)
		if parseBoolQuery(req, "flag") {
			t.Errorf("parseBoolQuery(%q) should return false", v)
		}
	}
}

func TestCoverageParseBoolQuery_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if parseBoolQuery(req, "flag") {
		t.Error("parseBoolQuery empty should return false")
	}
}

func TestCoverageParseBoolQueryDefault_DefaultTrue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if !parseBoolQueryDefault(req, "flag", true) {
		t.Error("应该返回默认值 true")
	}
}

func TestCoverageParseBoolQueryDefault_DefaultFalse(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if parseBoolQueryDefault(req, "flag", false) {
		t.Error("应该返回默认值 false")
	}
}

func TestCoverageParseBoolQueryDefault_ExplicitFalse(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?flag=false", nil)
	if parseBoolQueryDefault(req, "flag", true) {
		t.Error("显式 false 应覆盖默认 true")
	}
}

func TestCoverageParseBoolQueryDefault_ExplicitTrue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?flag=true", nil)
	if !parseBoolQueryDefault(req, "flag", false) {
		t.Error("显式 true 应覆盖默认 false")
	}
}

func TestCoverageParseBoolQueryDefault_UnknownValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?flag=maybe", nil)
	if !parseBoolQueryDefault(req, "flag", true) {
		t.Error("未知值应返回默认值 true")
	}
}

// ── parseUintCSV 测试 ──────────────────────────────────────────────────────

func TestCoverageParseUintCSV_Success(t *testing.T) {
	result, err := parseUintCSV("1,2,3")
	if err != nil {
		t.Fatalf("parseUintCSV err: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("期望 3 个值，实际=%d", len(result))
	}
}

func TestCoverageParseUintCSV_WithSpaces(t *testing.T) {
	result, err := parseUintCSV(" 1 , 2 , 3 ")
	if err != nil {
		t.Fatalf("parseUintCSV err: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("期望 3 个值，实际=%d", len(result))
	}
}

func TestCoverageParseUintCSV_Invalid(t *testing.T) {
	_, err := parseUintCSV("1,abc,3")
	if err == nil {
		t.Error("期望错误")
	}
}

func TestCoverageParseUintCSV_EmptyParts(t *testing.T) {
	result, err := parseUintCSV("1,,3")
	if err != nil {
		t.Fatalf("parseUintCSV err: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("期望 2 个值（空段被跳过），实际=%d", len(result))
	}
}

// ── isValidCategoryKey 测试 ────────────────────────────────────────────────

func TestCoverageIsValidCategoryKey_Valid(t *testing.T) {
	if !isValidCategoryKey("model") {
		t.Error("model 应是有效 key")
	}
	if !isValidCategoryKey("channel") {
		t.Error("channel 应是有效 key")
	}
	if !isValidCategoryKey("platformPolicy") {
		t.Error("platformPolicy 应是有效 key")
	}
}

func TestCoverageIsValidCategoryKey_Invalid(t *testing.T) {
	if isValidCategoryKey("foobar") {
		t.Error("foobar 不应是有效 key")
	}
	if isValidCategoryKey("") {
		t.Error("空字符串不应是有效 key")
	}
}

// ── buildCategoriesForGroup 测试（覆盖各个分支） ───────────────────────────

func TestCoverageBuildCategories_AllKeys(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	ancestors, _ := model.ClosureAncestors(context.Background(), child.ID, true)
	cfg := model.GetSiteConfig(context.Background())

	// 不做 key 过滤，应返回所有 category
	categories := buildCategoriesForGroup(context.Background(), child.ID, ancestors, &cfg, nil)
	if len(categories) == 0 {
		t.Error("期望有 categories 返回")
	}
}

func TestCoverageBuildCategories_WithModelData(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	// 添加模型数据
	model.DB(context.Background()).Create(&model.AIModel{ModelName: "gpt-4", VisibilityType: "all", Enabled: true, Visible: true})
	model.DB(context.Background()).Create(&model.AIModel{ModelName: "gpt-3.5", VisibilityType: "group", Enabled: true, Visible: true})

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	// 绑定 gpt-3.5 到研发组
	var m model.AIModel
	model.DB(context.Background()).Where("model_name = ?", "gpt-3.5").First(&m)
	model.DB(context.Background()).Create(&model.ModelVisibilityGroup{AIModelID: m.ID, GroupID: child.ID})

	ancestors, _ := model.ClosureAncestors(context.Background(), child.ID, true)
	cfg := model.GetSiteConfig(context.Background())

	categories := buildCategoriesForGroup(context.Background(), child.ID, ancestors, &cfg, map[string]bool{"model": true})
	if len(categories) != 1 {
		t.Fatalf("期望 1 个 category，实际=%d", len(categories))
	}
	if len(categories[0].Entries) < 1 {
		t.Error("期望有模型 entries")
	}
}

func TestCoverageBuildCategories_WithChannelData(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	// 创建 channel
	model.DB(context.Background()).Create(&model.AIChannel{Name: "微信", VisibilityType: "all"})

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	ancestors, _ := model.ClosureAncestors(context.Background(), child.ID, true)
	cfg := model.GetSiteConfig(context.Background())

	categories := buildCategoriesForGroup(context.Background(), child.ID, ancestors, &cfg, map[string]bool{"channel": true})
	if len(categories) != 1 {
		t.Fatalf("期望 1 个 category，实际=%d", len(categories))
	}
}

func TestCoverageBuildCategories_MemoryEnabled(t *testing.T) {
	setupTreeTestDB(t)

	// 修改 SiteConfig
	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]interface{}{
		"memory_tdai_enable":  true,
		"memory_default_plan": "pro",
	})

	cfg := model.GetSiteConfig(context.Background())
	entries := buildMemoryEntries(context.Background(), nil, &cfg)
	if len(entries) != 1 {
		t.Fatalf("期望 1 个 entry，实际=%d", len(entries))
	}
	if entries[0].Label != "Pro 版" {
		t.Errorf("期望 Pro 版，实际=%s", entries[0].Label)
	}
}

func TestCoverageBuildCategories_MemoryStandard(t *testing.T) {
	setupTreeTestDB(t)

	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]interface{}{
		"memory_tdai_enable":  true,
		"memory_default_plan": "free",
	})

	cfg := model.GetSiteConfig(context.Background())
	entries := buildMemoryEntries(context.Background(), nil, &cfg)
	if entries[0].Label != "Free 版" {
		t.Errorf("期望 Free 版，实际=%s", entries[0].Label)
	}
}

func TestCoverageBuildCategories_DriveEnabled(t *testing.T) {
	setupTreeTestDB(t)

	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Update("smh_auto_provision_on_create", true)

	cfg := model.GetSiteConfig(context.Background())
	entries := buildDriveEntries(context.Background(), nil, &cfg)
	if entries[0].Label != "开启" {
		t.Errorf("期望 开启，实际=%s", entries[0].Label)
	}
}

// 测试：分组策略命中覆盖全局默认
func TestCoverageBuildCategories_DriveGroupPolicy(t *testing.T) {
	setupTreeTestDB(t)

	// 全局默认关闭
	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Update("smh_auto_provision_on_create", false)

	// 创建分组并设置策略为开启
	createGroupWithClosure(t, 1, 0, "Root")
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeySMHAutoProvision,
		GroupID:    1,
		ValueJSON:  `{"enabled":true}`,
	})

	cfg := model.GetSiteConfig(context.Background())
	entries := buildDriveEntries(context.Background(), []uint{1}, &cfg)
	if entries[0].Label != "开启" {
		t.Errorf("期望 开启（分组策略覆盖），实际=%s", entries[0].Label)
	}
	if entries[0].Source.Type != usergroup.SourceLocal {
		t.Errorf("期望 source=local，实际=%s", entries[0].Source.Type)
	}
}

func TestCoverageBuildCategories_DriveInherited(t *testing.T) {
	setupTreeTestDB(t)

	// 全局默认关闭
	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Update("smh_auto_provision_on_create", false)

	// 创建父组和子组
	createGroupWithClosure(t, 1, 0, "Root")
	createGroupWithClosure(t, 2, 1, "Child")

	// 仅在父组设置策略为开启
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeySMHAutoProvision,
		GroupID:    1,
		ValueJSON:  `{"enabled":true}`,
	})

	cfg := model.GetSiteConfig(context.Background())
	// 子组未配置，应继承父组
	entries := buildDriveEntries(context.Background(), []uint{2, 1}, &cfg)
	if entries[0].Label != "开启" {
		t.Errorf("期望 开启（继承自父组），实际=%s", entries[0].Label)
	}
	if entries[0].Source.Type != usergroup.SourceInherited {
		t.Errorf("期望 source=inherited，实际=%s", entries[0].Source.Type)
	}
}

func TestCoverageBuildCategories_DriveSiteDefault(t *testing.T) {
	setupTreeTestDB(t)

	// 全局默认关闭
	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Update("smh_auto_provision_on_create", false)

	// 创建分组但不设置任何策略
	createGroupWithClosure(t, 1, 0, "Root")

	cfg := model.GetSiteConfig(context.Background())
	// 祖先链均未配置，应回退到全局默认
	entries := buildDriveEntries(context.Background(), []uint{1}, &cfg)
	if entries[0].Label != "关闭" {
		t.Errorf("期望 关闭（全局默认），实际=%s", entries[0].Label)
	}
	if entries[0].Source.Type != usergroup.SourceSiteDefault {
		t.Errorf("期望 source=site_default，实际=%s", entries[0].Source.Type)
	}
}

func TestCoverageBuildCategories_DriveFalseOverridesGlobalTrue(t *testing.T) {
	setupTreeTestDB(t)

	// 全局默认开启
	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Update("smh_auto_provision_on_create", true)

	// 创建分组并设置策略为关闭（覆盖全局开启）
	createGroupWithClosure(t, 1, 0, "Root")
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeySMHAutoProvision,
		GroupID:    1,
		ValueJSON:  `{"enabled":false}`,
	})

	cfg := model.GetSiteConfig(context.Background())
	entries := buildDriveEntries(context.Background(), []uint{1}, &cfg)
	if entries[0].Label != "关闭" {
		t.Errorf("期望 关闭（分组策略 false 覆盖全局 true），实际=%s", entries[0].Label)
	}
	if entries[0].Source.Type != usergroup.SourceLocal {
		t.Errorf("期望 source=local，实际=%s", entries[0].Source.Type)
	}
}

func TestCoverageBuildCategories_CLSEnabled(t *testing.T) {
	setupTreeTestDB(t)

	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Update("cls_enabled", 1)

	cfg := model.GetSiteConfig(context.Background())
	entries := buildCLSEntries(context.Background(), 0, &cfg)
	if entries[0].Label != "开启" {
		t.Errorf("期望 开启，实际=%s", entries[0].Label)
	}
}

func TestCoverageBuildCLSEntries_GroupMode_InScope(t *testing.T) {
	setupTreeTestDB(t)

	ctx := context.Background()
	model.DB(ctx).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]interface{}{
		"cls_enabled":    1,
		"cls_scope_mode": "group",
	})

	// 配置 scope: group 10
	model.DB(ctx).Create(&model.GroupConfigBinding{
		GroupID:    10,
		ConfigType: model.ConfigTypeCLSCollectScope,
		ConfigKey:  model.CLSCollectScopeKey,
	})
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: 10, DescendantID: 10, Depth: 0})
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: 10, DescendantID: 11, Depth: 1})

	cfg := model.GetSiteConfig(ctx)

	// group 11 在 scope 子树中 → 开启
	entries := buildCLSEntries(ctx, 11, &cfg)
	if entries[0].Label != "开启" {
		t.Errorf("scope 内分组期望 开启，实际=%s", entries[0].Label)
	}
	if meta, ok := entries[0].Meta.(map[string]interface{}); ok {
		if meta["scope_type"] != "group" {
			t.Errorf("期望 scope_type=group，实际=%v", meta["scope_type"])
		}
	}
}

func TestCoverageBuildCLSEntries_GroupMode_NotInScope(t *testing.T) {
	setupTreeTestDB(t)

	ctx := context.Background()
	model.DB(ctx).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]interface{}{
		"cls_enabled":    1,
		"cls_scope_mode": "group",
	})

	// 配置 scope: group 10
	model.DB(ctx).Create(&model.GroupConfigBinding{
		GroupID:    10,
		ConfigType: model.ConfigTypeCLSCollectScope,
		ConfigKey:  model.CLSCollectScopeKey,
	})
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: 10, DescendantID: 10, Depth: 0})

	cfg := model.GetSiteConfig(ctx)

	// group 99 不在 scope 中 → 关闭
	entries := buildCLSEntries(ctx, 99, &cfg)
	if entries[0].Label != "关闭" {
		t.Errorf("scope 外分组期望 关闭，实际=%s", entries[0].Label)
	}
}

func TestCoverageBuildCLSEntries_GroupMode_EmptyScope(t *testing.T) {
	setupTreeTestDB(t)

	ctx := context.Background()
	model.DB(ctx).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]interface{}{
		"cls_enabled":    1,
		"cls_scope_mode": "group",
	})
	// scope 为空 → 全量模式回退

	cfg := model.GetSiteConfig(ctx)
	entries := buildCLSEntries(ctx, 5, &cfg)
	if entries[0].Label != "开启" {
		t.Errorf("scope 为空时期望 开启（全量模式），实际=%s", entries[0].Label)
	}
}

func TestCoverageBuildCategories_Network(t *testing.T) {
	setupTreeTestDB(t)

	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]interface{}{
		"vpc_id":            "vpc-test123",
		"subnet_ids":        `{"ap-guangzhou-1": ["subnet-a"]}`,
		"security_group_id": "sg-test123",
	})

	// 创建分组和 VPC 配置绑定
	group := model.UserGroup{Name: "测试组"}
	model.DB(context.Background()).Create(&group)
	vpcCfg := model.VpcConfig{
		VpcId:          "vpc-group1",
		SubnetIds:      `{"ap-guangzhou-1": ["subnet-g1"]}`,
		VisibilityType: "group",
		StrategyName:   "测试策略",
	}
	model.DB(context.Background()).Create(&vpcCfg)
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypeVPC,
		ConfigKey:  fmt.Sprintf("%d", vpcCfg.ID),
		GroupID:    group.ID,
	})
	model.DB(context.Background()).Create(&model.GroupClosure{
		AncestorID: group.ID, DescendantID: group.ID, Depth: 0,
	})

	cfg := model.GetSiteConfig(context.Background())
	ancestors := []uint{group.ID}
	entries := buildNetworkEntries(context.Background(), group.ID, ancestors, &cfg)
	// 应有: VPC 主条目 + 子网 + 安全组 = 至少 3 个
	if len(entries) < 3 {
		t.Errorf("期望至少 3 个 network entries (VPC + subnet + SG)，实际=%d", len(entries))
	}
}

func TestCoverageBuildCategories_ImageType(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-001", AgentType: "openclaw", Enabled: true})
	model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-002", AgentType: "lobster", Enabled: true})

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	ancestors, _ := model.ClosureAncestors(context.Background(), child.ID, true)
	cfg := model.GetSiteConfig(context.Background())

	categories := buildCategoriesForGroup(context.Background(), child.ID, ancestors, &cfg, map[string]bool{"imageType": true})
	if len(categories) != 1 {
		t.Fatalf("期望 1 个 category，实际=%d", len(categories))
	}
	if len(categories[0].Entries) < 2 {
		t.Errorf("期望至少 2 个 image entries，实际=%d", len(categories[0].Entries))
	}
}

func TestCoverageBuildCategories_SkillEntries(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	// 创建技能包
	model.DB(context.Background()).Create(&model.SkillBundle{Name: "通用技能包", VisibilityType: "all"})

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	ancestors, _ := model.ClosureAncestors(context.Background(), child.ID, true)
	cfg := model.GetSiteConfig(context.Background())

	categories := buildCategoriesForGroup(context.Background(), child.ID, ancestors, &cfg, map[string]bool{"skill": true})
	if len(categories) != 1 {
		t.Fatalf("期望 1 个 category，实际=%d", len(categories))
	}
	// 至少有 SkillHub entry
	if len(categories[0].Entries) < 1 {
		t.Error("期望至少 1 个 skill entry")
	}
}

func TestCoverageBuildCategories_AgentToolEntries(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	// 创建企业插件和 MCP
	model.DB(context.Background()).Create(&model.Plugin{Slug: "test-plugin", Name: "测试插件", Version: "1.0.0", VersionMajor: 1})
	model.DB(context.Background()).Create(&model.McpServer{Name: "天气MCP", VisibilityType: "all"})

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	ancestors, _ := model.ClosureAncestors(context.Background(), child.ID, true)
	cfg := model.GetSiteConfig(context.Background())

	categories := buildCategoriesForGroup(context.Background(), child.ID, ancestors, &cfg, map[string]bool{"agentTool": true})
	if len(categories) != 1 {
		t.Fatalf("期望 1 个 category，实际=%d", len(categories))
	}
	if len(categories[0].Entries) < 2 {
		t.Errorf("期望至少 2 个 agent tool entries，实际=%d", len(categories[0].Entries))
	}
}

func TestCoverageBuildCategories_PolicyEntries(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	ancestors, _ := model.ClosureAncestors(context.Background(), child.ID, true)
	cfg := model.GetSiteConfig(context.Background())

	categories := buildCategoriesForGroup(context.Background(), child.ID, ancestors, &cfg, map[string]bool{"platformPolicy": true})
	if len(categories) != 1 {
		t.Fatalf("期望 1 个 category，实际=%d", len(categories))
	}
	// 默认策略应有 entries
	if len(categories[0].Entries) == 0 {
		t.Error("期望有策略 entries")
	}
	for _, entry := range categories[0].Entries {
		if entry.ID == usergroup.PolicyKeyInstanceQuota && entry.SubLabel == "" {
			t.Fatal("instance quota should retain the user-quota sublabel")
		}
	}
}

// ── appendVisibilityAllResources 测试 ──────────────────────────────────────

func TestCoverageAppendVisibilityAll_Channel(t *testing.T) {
	setupTreeTestDB(t)

	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "feishu", Name: "飞书", VisibilityType: "all"})
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "wecom", Name: "组通道", VisibilityType: "group"})

	overview := appendVisibilityAllResources(context.Background(), nil, "channel")
	if overview == nil {
		t.Fatal("期望非空 overview")
	}
	if len(overview.Items) != 1 {
		t.Errorf("期望 1 个全局可见通道，实际=%d", len(overview.Items))
	}
}

// TestBuildChannelEntries_SiteScopeFilter 验证 buildChannelEntries 过滤掉 overseas-only 通道
func TestBuildChannelEntries_SiteScopeFilter(t *testing.T) {
	setupTreeTestDB(t)

	// domestic 通道
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "feishu", Name: "飞书", VisibilityType: "all"})
	// overseas-only 通道（非海外模式应被过滤）
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "slack", Name: "Slack", VisibilityType: "all"})
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "discord", Name: "Discord", VisibilityType: "all"})

	// context.Background() 视为非海外模式
	entries := buildChannelEntries(context.Background(), 0, nil)
	for _, e := range entries {
		if e.Label == "Slack" || e.Label == "Discord" {
			t.Errorf("非海外模式下不应出现 overseas-only 通道：%s", e.Label)
		}
	}
}

// TestAppendVisibilityAllResources_ChannelSiteScope 验证 appendVisibilityAllResources 过滤掉 overseas-only 通道
func TestAppendVisibilityAllResources_ChannelSiteScope(t *testing.T) {
	setupTreeTestDB(t)

	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "wecom", Name: "企业微信", VisibilityType: "all"})
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "slack", Name: "Slack", VisibilityType: "all"})

	// 非海外模式：slack 应被过滤
	overview := appendVisibilityAllResources(context.Background(), nil, "channel")
	for _, item := range overview.Items {
		if item.ResourceName == "Slack" {
			t.Error("非海外模式下 slack 不应出现在 overview 中")
		}
	}
	found := false
	for _, item := range overview.Items {
		if item.ResourceName == "企业微信" {
			found = true
		}
	}
	if !found {
		t.Error("企业微信应出现在 overview 中")
	}
}

func TestCoverageAppendVisibilityAll_PluginBundle(t *testing.T) {
	setupTreeTestDB(t)

	model.DB(context.Background()).Create(&model.PluginBundle{Name: "全局插件", VisibilityType: "all", Enabled: true})

	overview := appendVisibilityAllResources(context.Background(), nil, "plugin_bundle")
	if len(overview.Items) != 1 {
		t.Errorf("期望 1 个全局可见插件，实际=%d", len(overview.Items))
	}
}

func TestCoverageAppendVisibilityAll_MCP(t *testing.T) {
	setupTreeTestDB(t)

	model.DB(context.Background()).Create(&model.McpServer{Name: "全局MCP", VisibilityType: "all"})

	overview := appendVisibilityAllResources(context.Background(), nil, "mcp")
	if len(overview.Items) != 1 {
		t.Errorf("期望 1 个全局可见 MCP，实际=%d", len(overview.Items))
	}
}

func TestCoverageAppendVisibilityAll_ImageType(t *testing.T) {
	setupTreeTestDB(t)

	model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-x1", AgentType: "openclaw", Enabled: true})
	model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-x2", AgentType: "lobster", Enabled: false}) // 未启用

	overview := appendVisibilityAllResources(context.Background(), nil, "image_type")
	if len(overview.Items) != 1 {
		t.Errorf("期望 1 个启用的 image type，实际=%d", len(overview.Items))
	}
}

// ── resolveVisibilityItems 测试 ─────────────────────────────────────────────

func TestCoverageResolveVisibilityItems_AllVisibility(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	model.DB(context.Background()).Create(&model.AIModel{ModelName: "gpt-4", VisibilityType: "all", Enabled: true})

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)
	ancestors, _ := model.ClosureAncestors(context.Background(), child.ID, true)

	resources := queryModelResources()
	items := resolveVisibilityItems(context.Background(), resources, "model_visibility_groups", "ai_model_id", child.ID, ancestors)
	if len(items) != 1 {
		t.Errorf("期望 1 个全局模型，实际=%d", len(items))
	}
}

func TestCoverageResolveVisibilityItems_GroupVisibility(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	m := model.AIModel{ModelName: "internal-model", VisibilityType: "group", Enabled: true}
	model.DB(context.Background()).Create(&m)

	var child model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&child)

	model.DB(context.Background()).Create(&model.ModelVisibilityGroup{AIModelID: m.ID, GroupID: child.ID})

	ancestors, _ := model.ClosureAncestors(context.Background(), child.ID, true)
	resources := queryModelResources()
	items := resolveVisibilityItems(context.Background(), resources, "model_visibility_groups", "ai_model_id", child.ID, ancestors)

	found := false
	for _, item := range items {
		if item.ResourceName == "internal-model" {
			found = true
		}
	}
	if !found {
		t.Error("期望找到 group visibility 的模型")
	}
}

func TestCoverageResolveVisibilityItems_InheritedFromParent(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	m := model.AIModel{ModelName: "parent-model", VisibilityType: "group", Enabled: true}
	model.DB(context.Background()).Create(&m)

	var root model.UserGroup
	model.DB(context.Background()).Where("name = ?", "根组").First(&root)
	model.DB(context.Background()).Create(&model.ModelVisibilityGroup{AIModelID: m.ID, GroupID: root.ID})

	var grandchild model.UserGroup
	model.DB(context.Background()).Where("name = ?", "前端组").First(&grandchild)

	ancestors, _ := model.ClosureAncestors(context.Background(), grandchild.ID, true)
	resources := queryModelResources()
	items := resolveVisibilityItems(context.Background(), resources, "model_visibility_groups", "ai_model_id", grandchild.ID, ancestors)

	found := false
	for _, item := range items {
		if item.ResourceName == "parent-model" {
			found = true
		}
	}
	if !found {
		t.Error("期望孙组继承根组的模型")
	}
}

// queryModelResources 测试辅助：查询模型并转换为 overviewResource
func queryModelResources() []overviewResource {
	var rows []struct {
		ID             uint   `gorm:"column:id"`
		Name           string `gorm:"column:name"`
		VisibilityType string `gorm:"column:visibility_type"`
	}
	model.DB(context.Background()).Model(&model.AIModel{}).
		Select("id, model_name as name, COALESCE(visibility_type,'all') as visibility_type").
		Where("enabled = ?", true).
		Where("NOT (provider = ? AND model_id = ?)", model.BuiltinModelProvider, model.BuiltinModelID).
		Find(&rows)
	resources := make([]overviewResource, 0, len(rows))
	for _, r := range rows {
		resources = append(resources, overviewResource{ID: r.ID, Name: r.Name, VisibilityType: r.VisibilityType})
	}
	return resources
}

// ── parseSubnetsForOverview 测试 ───────────────────────────────────────────

func TestCoverageParseSubnetsForOverview(t *testing.T) {
	setupTreeTestDB(t)

	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Update("subnet_ids", `{"ap-guangzhou-1": ["subnet-a", "subnet-b"]}`)

	cfg := model.GetSiteConfig(context.Background())
	result := parseSubnetsForOverview(&cfg)
	if len(result) != 2 {
		t.Errorf("期望 2 个子网，实际=%d", len(result))
	}
}

func TestCoverageParseSubnetsForOverview_Empty(t *testing.T) {
	setupTreeTestDB(t)

	cfg := model.GetSiteConfig(context.Background())
	result := parseSubnetsForOverview(&cfg)
	if result != nil {
		t.Errorf("期望 nil，实际=%v", result)
	}
}

// ── buildCLSEntries 测试 ──────────────────────────────────────────────────

func TestBuildCLSEntries_Disabled(t *testing.T) {
	setupTreeTestDB(t)
	ctx := context.Background()
	cfg := model.GetSiteConfig(ctx)
	entries := buildCLSEntries(ctx, 1, &cfg)
	if len(entries) != 1 || entries[0].Label != "关闭" {
		t.Errorf("CLS 未开启应返回关闭，实际=%v", entries)
	}
}

func TestBuildCLSEntries_EnabledAllMode(t *testing.T) {
	setupTreeTestDB(t)
	ctx := context.Background()
	model.DB(ctx).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]interface{}{"cls_enabled": 1})
	cfg := model.GetSiteConfig(ctx)
	entries := buildCLSEntries(ctx, 1, &cfg)
	if len(entries) != 1 || entries[0].Label != "开启" {
		t.Errorf("全量模式应返回开启，实际=%v", entries)
	}
	if meta, ok := entries[0].Meta.(map[string]interface{}); ok {
		if meta["scope_type"] != "all" {
			t.Errorf("全量模式 scope_type 应为 all，实际=%v", meta["scope_type"])
		}
	}
}

func TestBuildCLSEntries_GroupMode_InScope(t *testing.T) {
	setupTreeTestDB(t)
	ctx := context.Background()
	model.DB(ctx).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]interface{}{"cls_enabled": 1, "cls_scope_mode": "group"})

	// 创建一个分组，设置 scope 包含该分组
	model.DB(ctx).Create(&model.UserGroup{Name: "CLS组", FullPath: "CLS组", Source: "manual"})
	var grp model.UserGroup
	model.DB(ctx).Where("name = ?", "CLS组").First(&grp)

	model.SetCLSCollectScope(ctx, []uint{grp.ID})
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: grp.ID, DescendantID: grp.ID, Depth: 0})

	cfg := model.GetSiteConfig(ctx)
	entries := buildCLSEntries(ctx, grp.ID, &cfg)
	if len(entries) != 1 || entries[0].Label != "开启" {
		t.Errorf("scope 内分组应返回开启，实际=%v", entries)
	}
	if meta, ok := entries[0].Meta.(map[string]interface{}); ok {
		if meta["scope_type"] != "group" {
			t.Errorf("分组模式 scope_type 应为 group，实际=%v", meta["scope_type"])
		}
	}
}

func TestBuildCLSEntries_GroupMode_NotInScope(t *testing.T) {
	setupTreeTestDB(t)
	ctx := context.Background()
	model.DB(ctx).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]interface{}{"cls_enabled": 1, "cls_scope_mode": "group"})

	// scope 包含 group 100
	model.DB(ctx).Create(&model.UserGroup{Name: "ScopeA", FullPath: "ScopeA", Source: "manual"})
	var grp model.UserGroup
	model.DB(ctx).Where("name = ?", "ScopeA").First(&grp)
	model.SetCLSCollectScope(ctx, []uint{grp.ID})
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: grp.ID, DescendantID: grp.ID, Depth: 0})

	cfg := model.GetSiteConfig(ctx)
	// 查询不在 scope 内的分组 (ID=99999)
	entries := buildCLSEntries(ctx, 99999, &cfg)
	if len(entries) != 1 || entries[0].Label != "关闭" {
		t.Errorf("scope 外分组应返回关闭，实际=%v", entries)
	}
}

// TestBuildCLSEntries_GroupMode_ScopeWithDescendants 验证子组在父组 scope 子孙中时命中。
func TestBuildCLSEntries_GroupMode_ScopeWithDescendants(t *testing.T) {
	setupTreeTestDB(t)
	ctx := context.Background()
	model.DB(ctx).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]interface{}{"cls_enabled": 1, "cls_scope_mode": "group"})

	// 创建父子分组
	model.DB(ctx).Create(&model.UserGroup{Name: "父组CLS", FullPath: "父组CLS", Source: "manual"})
	model.DB(ctx).Create(&model.UserGroup{Name: "子组CLS", FullPath: "父组CLS/子组CLS", Source: "manual"})
	var parent, child model.UserGroup
	model.DB(ctx).Where("name = ?", "父组CLS").First(&parent)
	model.DB(ctx).Where("name = ?", "子组CLS").First(&child)

	// scope 包含父组
	model.SetCLSCollectScope(ctx, []uint{parent.ID})
	// closure：父组包含子组
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: parent.ID, DescendantID: parent.ID, Depth: 0})
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: parent.ID, DescendantID: child.ID, Depth: 1})

	cfg := model.GetSiteConfig(ctx)
	// 子组在父组的子孙中，应命中 scope
	entries := buildCLSEntries(ctx, child.ID, &cfg)
	if len(entries) != 1 || entries[0].Label != "开启" {
		t.Errorf("子组在 scope 子孙中应返回开启，实际=%v", entries)
	}
	if meta, ok := entries[0].Meta.(map[string]interface{}); ok {
		if meta["scope_type"] != "group" {
			t.Errorf("分组模式 scope_type 应为 group，实际=%v", meta["scope_type"])
		}
		if meta["enabled"] != true {
			t.Errorf("子组在 scope 内 enabled 应为 true，实际=%v", meta["enabled"])
		}
	}
}
