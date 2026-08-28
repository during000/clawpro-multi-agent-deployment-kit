package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// 测试辅助
// ---------------------------------------------------------------------------

// initInstanceListTestDB 初始化 HandleInstanceList 单测所需的内存数据库
func initInstanceListTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Instance{},
		&model.AIModel{},
		&model.AIImage{},
		&model.OpenClawRole{},
		&model.Notification{},
		&model.SiteConfig{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	restore := model.UseDBForTest(db)

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	return func() {
		restore()
		Store = origStore
	}
}

// instanceListJSONReq 构造带 session 的 JSON 请求
func instanceListJSONReq(t *testing.T, method, path, username string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Accept", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username

	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

// instanceListCreateInstances 为指定 user 创建 n 个 openclaw 实例
func instanceListCreateInstances(t *testing.T, userID uint, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		inst := &model.Instance{
			Name:      fmt.Sprintf("inst-%d", i),
			UserID:    userID,
			AgentType: model.AgentTypeOpenClaw,
		}
		if err := model.DB(context.Background()).Create(inst).Error; err != nil {
			t.Fatalf("创建测试实例失败: %v", err)
		}
	}
}

// instanceListCreateInstancesWithRole 创建带 RoleID 的实例
func instanceListCreateInstancesWithRole(t *testing.T, userID, roleID uint, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		inst := &model.Instance{
			Name:      fmt.Sprintf("inst-role-%d-%d", roleID, i),
			UserID:    userID,
			AgentType: model.AgentTypeOpenClaw,
			RoleID:    roleID,
		}
		if err := model.DB(context.Background()).Create(inst).Error; err != nil {
			t.Fatalf("创建测试实例失败: %v", err)
		}
	}
}

// instanceListParsePaginatedResp 解析分页响应
func instanceListParsePaginatedResp(t *testing.T, body []byte) (instances []interface{}, total, page, pageSize, totalPages int) {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("解析 JSON 失败: %v, body=%s", err, string(body))
	}

	if data, ok := resp["data"]; ok {
		resp = data.(map[string]interface{})
	}

	if arr, ok := resp["instances"].([]interface{}); ok {
		instances = arr
	}
	if v, ok := resp["total"].(float64); ok {
		total = int(v)
	}
	if v, ok := resp["page"].(float64); ok {
		page = int(v)
	}
	if v, ok := resp["page_size"].(float64); ok {
		pageSize = int(v)
	}
	if v, ok := resp["total_pages"].(float64); ok {
		totalPages = int(v)
	}
	return
}

// ---------------------------------------------------------------------------
// HandleInstanceList — 完整单测
// ---------------------------------------------------------------------------

// ===========================================================================
// 认证相关测试
// ===========================================================================

// TestHandleInstanceList_Unauthorized_JSON 未登录 JSON 请求返回 401
func TestHandleInstanceList_Unauthorized_JSON(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/list", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("未登录 JSON 请求应返回 401，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleInstanceList_UserNotFound session 中的用户在数据库中不存在
func TestHandleInstanceList_UserNotFound(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	req := instanceListJSONReq(t, "GET", "/openclaw/list", "ghost_user")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("用户不存在应返回 401，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ===========================================================================
// 分页功能测试
// ===========================================================================

// TestHandleInstanceList_Pagination_Defaults 不传分页参数时使用默认值 page=1, page_size=30
func TestHandleInstanceList_Pagination_Defaults(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 50)

	req := instanceListJSONReq(t, "GET", "/openclaw/list", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	instances, total, page, pageSize, totalPages := instanceListParsePaginatedResp(t, rr.Body.Bytes())

	if page != 1 {
		t.Errorf("默认 page 应为 1，实际=%d", page)
	}
	if pageSize != 30 {
		t.Errorf("默认 page_size 应为 30，实际=%d", pageSize)
	}
	if total != 50 {
		t.Errorf("total 应为 50，实际=%d", total)
	}
	if totalPages != 2 {
		t.Errorf("total_pages 应为 2 (ceil(50/30))，实际=%d", totalPages)
	}
	if len(instances) != 30 {
		t.Errorf("第一页应返回 30 条，实际=%d", len(instances))
	}
}

// TestHandleInstanceList_Pagination_SecondPage 请求第二页
func TestHandleInstanceList_Pagination_SecondPage(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 50)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?page=2&page_size=30", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	instances, total, page, pageSize, totalPages := instanceListParsePaginatedResp(t, rr.Body.Bytes())

	if page != 2 {
		t.Errorf("page 应为 2，实际=%d", page)
	}
	if pageSize != 30 {
		t.Errorf("page_size 应为 30，实际=%d", pageSize)
	}
	if total != 50 {
		t.Errorf("total 应为 50，实际=%d", total)
	}
	if totalPages != 2 {
		t.Errorf("total_pages 应为 2，实际=%d", totalPages)
	}
	if len(instances) != 20 {
		t.Errorf("第二页应返回 20 条（50-30），实际=%d", len(instances))
	}
}

// TestHandleInstanceList_Pagination_CustomPageSize 自定义 page_size
func TestHandleInstanceList_Pagination_CustomPageSize(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 15)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?page=1&page_size=10", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	instances, total, page, pageSize, totalPages := instanceListParsePaginatedResp(t, rr.Body.Bytes())

	if page != 1 {
		t.Errorf("page 应为 1，实际=%d", page)
	}
	if pageSize != 10 {
		t.Errorf("page_size 应为 10，实际=%d", pageSize)
	}
	if total != 15 {
		t.Errorf("total 应为 15，实际=%d", total)
	}
	if totalPages != 2 {
		t.Errorf("total_pages 应为 2 (ceil(15/10))，实际=%d", totalPages)
	}
	if len(instances) != 10 {
		t.Errorf("第一页应返回 10 条，实际=%d", len(instances))
	}
}

// TestHandleInstanceList_Pagination_PageSizeMax 超过上限 100 时被截断
func TestHandleInstanceList_Pagination_PageSizeMax(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 5)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?page_size=500", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	_, _, _, pageSize, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())

	if pageSize != 100 {
		t.Errorf("page_size 超上限应被截断为 100，实际=%d", pageSize)
	}
}

// TestHandleInstanceList_Pagination_PageSizeExact100 刚好等于上限 100
func TestHandleInstanceList_Pagination_PageSizeExact100(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 5)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?page_size=100", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	_, _, _, pageSize, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())

	if pageSize != 100 {
		t.Errorf("page_size=100 应保持为 100，实际=%d", pageSize)
	}
}

// TestHandleInstanceList_Pagination_InvalidParams 非法参数回退默认值
func TestHandleInstanceList_Pagination_InvalidParams(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 3)

	tests := []struct {
		name         string
		query        string
		wantPage     int
		wantPageSize int
	}{
		{"page 负数", "page=-1&page_size=10", 1, 10},
		{"page 零", "page=0&page_size=10", 1, 10},
		{"page_size 负数", "page=1&page_size=-5", 1, 30},
		{"page_size 零", "page=1&page_size=0", 1, 30},
		{"page 非数字", "page=abc&page_size=10", 1, 10},
		{"page_size 非数字", "page=1&page_size=xyz", 1, 30},
		{"均非法", "page=abc&page_size=xyz", 1, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := instanceListJSONReq(t, "GET", "/openclaw/list?"+tt.query, "u1")
			rr := httptest.NewRecorder()
			HandleInstanceList(rr, req)

			if rr.Code != 200 {
				t.Fatalf("应 200，实际=%d", rr.Code)
			}

			_, _, page, pageSize, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
			if page != tt.wantPage {
				t.Errorf("page 应为 %d，实际=%d", tt.wantPage, page)
			}
			if pageSize != tt.wantPageSize {
				t.Errorf("page_size 应为 %d，实际=%d", tt.wantPageSize, pageSize)
			}
		})
	}
}

// TestHandleInstanceList_Pagination_EmptyResult 无实例时返回空数组 + 合理分页元数据
func TestHandleInstanceList_Pagination_EmptyResult(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?page=1&page_size=10", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	instances, total, _, _, totalPages := instanceListParsePaginatedResp(t, rr.Body.Bytes())

	if total != 0 {
		t.Errorf("total 应为 0，实际=%d", total)
	}
	if totalPages != 0 {
		t.Errorf("total_pages 应为 0，实际=%d", totalPages)
	}
	if len(instances) != 0 {
		t.Errorf("instances 应为空数组，实际=%d 条", len(instances))
	}
}

// TestHandleInstanceList_Pagination_ExactDivision total 恰好整除 page_size
func TestHandleInstanceList_Pagination_ExactDivision(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 20)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?page=1&page_size=10", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	instances, total, _, _, totalPages := instanceListParsePaginatedResp(t, rr.Body.Bytes())

	if total != 20 {
		t.Errorf("total 应为 20，实际=%d", total)
	}
	if totalPages != 2 {
		t.Errorf("total_pages 应为 2 (20/10 恰好整除)，实际=%d", totalPages)
	}
	if len(instances) != 10 {
		t.Errorf("第一页应返回 10 条，实际=%d", len(instances))
	}
}

// TestHandleInstanceList_Pagination_PageBeyondTotal 请求超出范围的页码时返回空数组
func TestHandleInstanceList_Pagination_PageBeyondTotal(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 5)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?page=99&page_size=10", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	instances, total, page, _, totalPages := instanceListParsePaginatedResp(t, rr.Body.Bytes())

	if total != 5 {
		t.Errorf("total 应为 5，实际=%d", total)
	}
	if page != 99 {
		t.Errorf("page 应为 99（保持请求值），实际=%d", page)
	}
	if totalPages != 1 {
		t.Errorf("total_pages 应为 1，实际=%d", totalPages)
	}
	if len(instances) != 0 {
		t.Errorf("超出范围页应返回空数组，实际=%d 条", len(instances))
	}
}

// TestHandleInstanceList_Pagination_TotalPagesCalculation 验证 total_pages 的 ceil 计算
func TestHandleInstanceList_Pagination_TotalPagesCalculation(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	tests := []struct {
		name           string
		instanceCount  int
		pageSize       int
		wantTotalPages int
	}{
		{"0 条", 0, 10, 0},
		{"1 条", 1, 10, 1},
		{"恰好整除 10/10", 10, 10, 1},
		{"余 1 条 11/10", 11, 10, 2},
		{"大量 99/30", 99, 30, 4},
		{"大量 100/30", 100, 30, 4},
		{"大量 91/30", 91, 30, 4},
		{"大量 90/30", 90, 30, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清理之前的实例
			model.DB(context.Background()).Where("user_id = ?", user.ID).Delete(&model.Instance{})
			instanceListCreateInstances(t, user.ID, tt.instanceCount)

			req := instanceListJSONReq(t, "GET",
				fmt.Sprintf("/openclaw/list?page=1&page_size=%d", tt.pageSize), "u1")
			rr := httptest.NewRecorder()
			HandleInstanceList(rr, req)

			if rr.Code != 200 {
				t.Fatalf("应 200，实际=%d", rr.Code)
			}

			_, _, _, _, totalPages := instanceListParsePaginatedResp(t, rr.Body.Bytes())
			if totalPages != tt.wantTotalPages {
				t.Errorf("total_pages 应为 %d，实际=%d", tt.wantTotalPages, totalPages)
			}
		})
	}
}

// TestHandleInstanceList_PageSize1 最小合法 page_size=1
func TestHandleInstanceList_PageSize1(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 3)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?page=1&page_size=1", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	instances, total, _, pageSize, totalPages := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if pageSize != 1 {
		t.Errorf("page_size 应为 1，实际=%d", pageSize)
	}
	if total != 3 {
		t.Errorf("total 应为 3，实际=%d", total)
	}
	if totalPages != 3 {
		t.Errorf("total_pages 应为 3，实际=%d", totalPages)
	}
	if len(instances) != 1 {
		t.Errorf("page_size=1 应返回 1 条，实际=%d", len(instances))
	}
}

// ===========================================================================
// 响应字段验证测试
// ===========================================================================

// TestHandleInstanceList_ResponseFields 验证响应包含预期的分页元数据字段
func TestHandleInstanceList_ResponseFields(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 3)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?page=1&page_size=10", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析 JSON 失败: %v", err)
	}

	requiredFields := []string{"instances", "total", "page", "page_size", "total_pages"}
	for _, field := range requiredFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("响应缺少必需字段: %s", field)
		}
	}
}

// TestHandleInstanceList_InstanceFields 验证每条实例包含 role_name 和 light_claw_user_id 字段
func TestHandleInstanceList_InstanceFields(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建一个带角色的实例
	role := &model.OpenClawRole{Name: "测试角色"}
	model.DB(context.Background()).Create(role)
	instanceListCreateInstancesWithRole(t, user.ID, role.ID, 1)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?page=1&page_size=10", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	instances, _, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if len(instances) == 0 {
		t.Fatal("应至少返回 1 条实例")
	}

	inst := instances[0].(map[string]interface{})

	// 验证 role_name 字段
	roleName, ok := inst["role_name"]
	if !ok {
		t.Error("实例缺少 role_name 字段")
	} else if roleName != "测试角色" {
		t.Errorf("role_name 应为 '测试角色'，实际=%v", roleName)
	}

	// 验证 light_claw_user_id 字段
	if _, ok := inst["light_claw_user_id"]; !ok {
		t.Error("实例缺少 light_claw_user_id 字段")
	}
}

// TestHandleInstanceList_ModelInfo 验证 AIModelID > 1 时返回模型信息
func TestHandleInstanceList_ModelInfo(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建占位模型（ID=1 为自定义模型，不会触发查询）
	placeholder := &model.AIModel{Provider: "placeholder", ModelName: "placeholder", ModelID: "placeholder"}
	model.DB(context.Background()).Create(placeholder)

	// 创建一个预置模型（ID=2，> 1）
	aiModel := &model.AIModel{
		Provider:  "openai",
		ModelName: "gpt-4",
		ModelID:   "gpt-4-turbo",
	}
	model.DB(context.Background()).Create(aiModel)

	// 创建带模型 ID 的实例
	inst := &model.Instance{
		Name:      "inst-with-model",
		UserID:    user.ID,
		AgentType: model.AgentTypeOpenClaw,
		AIModelID: aiModel.ID,
	}
	model.DB(context.Background()).Create(inst)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?page=1&page_size=10", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	instances, _, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if len(instances) == 0 {
		t.Fatal("应至少返回 1 条实例")
	}

	instResp := instances[0].(map[string]interface{})
	if provider, ok := instResp["model_provider"]; !ok || provider != "openai" {
		t.Errorf("model_provider 应为 'openai'，实际=%v", instResp["model_provider"])
	}
	if modelName, ok := instResp["model_name"]; !ok || modelName != "gpt-4" {
		t.Errorf("model_name 应为 'gpt-4'，实际=%v", instResp["model_name"])
	}
}

// TestHandleInstanceList_ModelInfoOmittedForCustomModel AIModelID <= 1 时不返回模型信息
func TestHandleInstanceList_ModelInfoOmittedForCustomModel(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建自定义模型实例（AIModelID = 0）
	inst := &model.Instance{
		Name:      "inst-custom-model",
		UserID:    user.ID,
		AgentType: model.AgentTypeOpenClaw,
		AIModelID: 0,
	}
	model.DB(context.Background()).Create(inst)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?page=1&page_size=10", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	instances, _, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if len(instances) == 0 {
		t.Fatal("应至少返回 1 条实例")
	}

	instResp := instances[0].(map[string]interface{})
	// omitempty: 自定义模型不应有 model_provider/model_name
	if _, ok := instResp["model_provider"]; ok {
		t.Error("自定义模型实例不应返回 model_provider 字段")
	}
	if _, ok := instResp["model_name"]; ok {
		t.Error("自定义模型实例不应返回 model_name 字段")
	}
}

// ===========================================================================
// 数据隔离测试
// ===========================================================================

// TestHandleInstanceList_UserIsolation 不同用户之间数据隔离
func TestHandleInstanceList_UserIsolation(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user1 := &model.User{Username: "user1", Password: "x", Role: "user"}
	user2 := &model.User{Username: "user2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user1)
	model.DB(context.Background()).Create(user2)

	instanceListCreateInstances(t, user1.ID, 5)
	instanceListCreateInstances(t, user2.ID, 3)

	// user1 应只看到自己的 5 条
	req := instanceListJSONReq(t, "GET", "/openclaw/list?page=1&page_size=50", "user1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	instances, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 5 {
		t.Errorf("user1 total 应为 5，实际=%d", total)
	}
	if len(instances) != 5 {
		t.Errorf("user1 应返回 5 条，实际=%d", len(instances))
	}

	// user2 应只看到自己的 3 条
	req2 := instanceListJSONReq(t, "GET", "/openclaw/list?page=1&page_size=50", "user2")
	rr2 := httptest.NewRecorder()
	HandleInstanceList(rr2, req2)

	if rr2.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr2.Code)
	}

	instances2, total2, _, _, _ := instanceListParsePaginatedResp(t, rr2.Body.Bytes())
	if total2 != 3 {
		t.Errorf("user2 total 应为 3，实际=%d", total2)
	}
	if len(instances2) != 3 {
		t.Errorf("user2 应返回 3 条，实际=%d", len(instances2))
	}
}

// ===========================================================================
// 排序验证测试
// ===========================================================================

// TestHandleInstanceList_OrderByCreatedAtDesc 验证返回顺序为 created_at 降序
func TestHandleInstanceList_OrderByCreatedAtDesc(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建实例（SQLite 中顺序创建，ID 递增即为时间递增）
	instanceListCreateInstances(t, user.ID, 5)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?page=1&page_size=10", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	instances, _, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if len(instances) < 2 {
		t.Skip("不够两条数据验证排序")
	}

	// 验证降序：第一条的 ID 应大于第二条
	first := instances[0].(map[string]interface{})
	second := instances[1].(map[string]interface{})
	firstID := first["ID"].(float64)
	secondID := second["ID"].(float64)
	if firstID <= secondID {
		t.Errorf("应按 created_at desc 排序，第一条 ID(%v) 应大于第二条 ID(%v)", firstID, secondID)
	}
}

// ===========================================================================
// Content-Type 验证
// ===========================================================================

// TestHandleInstanceList_ContentTypeJSON 验证 JSON 响应的 Content-Type
func TestHandleInstanceList_ContentTypeJSON(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := instanceListJSONReq(t, "GET", "/openclaw/list", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type 应为 application/json，实际=%s", ct)
	}
}

// ===========================================================================
// 兼容性测试（老版本前端）
// ===========================================================================

// TestHandleInstanceList_BackwardCompatible 老版本前端不传分页参数仍正常返回
func TestHandleInstanceList_BackwardCompatible(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 10)

	// 模拟老版本前端：不传任何分页参数
	req := instanceListJSONReq(t, "GET", "/openclaw/list", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析 JSON 失败: %v", err)
	}

	// instances 字段仍然存在（兼容老前端解析）
	if _, ok := resp["instances"]; !ok {
		t.Error("响应必须包含 instances 字段（兼容老前端）")
	}

	// 新增字段不影响老前端（额外字段会被忽略）
	instances, total, page, pageSize, totalPages := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 10 {
		t.Errorf("total 应为 10，实际=%d", total)
	}
	if page != 1 {
		t.Errorf("page 应默认为 1，实际=%d", page)
	}
	if pageSize != 30 {
		t.Errorf("page_size 应默认为 30，实际=%d", pageSize)
	}
	if totalPages != 1 {
		t.Errorf("total_pages 应为 1 (ceil(10/30))，实际=%d", totalPages)
	}
	if len(instances) != 10 {
		t.Errorf("应返回全部 10 条（未超过默认 page_size），实际=%d", len(instances))
	}
}

// ===========================================================================
// ID / instance_id 精准搜索测试
// ===========================================================================

// TestHandleInstanceList_SearchByID 通过主键 ID 精准搜索
func TestHandleInstanceList_SearchByID(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建 3 个实例
	for i := 0; i < 3; i++ {
		inst := &model.Instance{
			Name:       fmt.Sprintf("inst-%d", i),
			InstanceId: fmt.Sprintf("ins-search-%d", i),
			UserID:     user.ID,
			AgentType:  model.AgentTypeOpenClaw,
		}
		model.DB(context.Background()).Create(inst)
	}

	// 查询第一个实例的 ID
	var target model.Instance
	model.DB(context.Background()).Where("user_id = ?", user.ID).First(&target)

	req := instanceListJSONReq(t, "GET", fmt.Sprintf("/openclaw/list?id=%d", target.ID), "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	instances, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 1 {
		t.Errorf("按 ID 精准搜索 total 应为 1，实际=%d", total)
	}
	if len(instances) != 1 {
		t.Errorf("按 ID 精准搜索应返回 1 条，实际=%d", len(instances))
	}
}

// TestHandleInstanceList_SearchByInstanceID 通过 instance_id 精准搜索
func TestHandleInstanceList_SearchByInstanceID(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建多个实例，每个有唯一的 instance_id
	for i := 0; i < 5; i++ {
		inst := &model.Instance{
			Name:       fmt.Sprintf("inst-%d", i),
			InstanceId: fmt.Sprintf("ins-abc%d", i),
			UserID:     user.ID,
			AgentType:  model.AgentTypeOpenClaw,
		}
		model.DB(context.Background()).Create(inst)
	}

	req := instanceListJSONReq(t, "GET", "/openclaw/list?instance_id=ins-abc3", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	instances, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 1 {
		t.Errorf("按 instance_id 精准搜索 total 应为 1，实际=%d", total)
	}
	if len(instances) != 1 {
		t.Errorf("按 instance_id 精准搜索应返回 1 条，实际=%d", len(instances))
	}
}

// TestHandleInstanceList_SearchByID_Priority id 优先级高于 instance_id
func TestHandleInstanceList_SearchByID_Priority(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	inst1 := &model.Instance{
		Name:       "target-by-id",
		InstanceId: "ins-aaa",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst1)

	inst2 := &model.Instance{
		Name:       "target-by-instance-id",
		InstanceId: "ins-bbb",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst2)

	// 同时传 id 和 instance_id，id 优先
	req := instanceListJSONReq(t, "GET", fmt.Sprintf("/openclaw/list?id=%d&instance_id=%s", inst1.ID, inst2.InstanceId), "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	instances, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 1 {
		t.Errorf("id 优先时 total 应为 1，实际=%d", total)
	}
	if len(instances) != 1 {
		t.Fatalf("id 优先时应返回 1 条，实际=%d", len(instances))
	}

	// 验证返回的是 inst1（按 id 搜索的结果）而非 inst2
	got := instances[0].(map[string]interface{})
	if gotName, ok := got["Name"].(string); ok && gotName != "target-by-id" {
		t.Errorf("id 优先，应返回 target-by-id，实际=%s", gotName)
	}
}

// TestHandleInstanceList_SearchByID_InvalidID id 为非数字时忽略，返回全部
func TestHandleInstanceList_SearchByID_InvalidID(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 3)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?id=abc", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	_, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	// id 解析失败时不添加过滤条件，返回全部
	if total != 3 {
		t.Errorf("id 无效时应返回全部 3 条，实际=%d", total)
	}
}

// TestHandleInstanceList_SearchByID_Zero id=0 时忽略，返回全部
func TestHandleInstanceList_SearchByID_Zero(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 4)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?id=0", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	_, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 4 {
		t.Errorf("id=0 时应返回全部 4 条，实际=%d", total)
	}
}

// TestHandleInstanceList_SearchByInstanceID_NotFound instance_id 不存在时返回空列表
func TestHandleInstanceList_SearchByInstanceID_NotFound(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 3)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?instance_id=ins-nonexist", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	instances, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 0 {
		t.Errorf("instance_id 不存在时 total 应为 0，实际=%d", total)
	}
	if len(instances) != 0 {
		t.Errorf("instance_id 不存在时应返回空列表，实际=%d 条", len(instances))
	}
}

// TestHandleInstanceList_SearchByID_OtherUserIsolation 搜索 id 时不能跨用户查看
func TestHandleInstanceList_SearchByID_OtherUserIsolation(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user1 := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user1)
	user2 := &model.User{Username: "u2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user2)

	// 实例属于 user2
	inst := &model.Instance{
		Name:       "user2-inst",
		InstanceId: "ins-u2-001",
		UserID:     user2.ID,
		AgentType:  model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// user1 尝试按 id 搜索 user2 的实例
	req := instanceListJSONReq(t, "GET", fmt.Sprintf("/openclaw/list?id=%d", inst.ID), "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	instances, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 0 {
		t.Errorf("不应跨用户查看，total 应为 0，实际=%d", total)
	}
	if len(instances) != 0 {
		t.Errorf("不应跨用户查看，应返回空列表，实际=%d 条", len(instances))
	}
}

// ===========================================================================
// keyword 模糊搜索 / agent_type 多值过滤测试
// ===========================================================================

// instanceListCreateNamedInstance 创建一个指定 name + instance_id + agent_type 的实例
func instanceListCreateNamedInstance(t *testing.T, userID uint, name, instanceID, agentType string) {
	t.Helper()
	if agentType == "" {
		agentType = model.AgentTypeOpenClaw
	}
	inst := &model.Instance{
		Name:       name,
		InstanceId: instanceID,
		UserID:     userID,
		AgentType:  agentType,
	}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
}

// instanceListCollectNames 从响应实例数组里取出 Name 字段集合
func instanceListCollectNames(instances []interface{}) map[string]bool {
	out := map[string]bool{}
	for _, it := range instances {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		if name, ok := m["Name"].(string); ok {
			out[name] = true
		}
	}
	return out
}

// TestHandleInstanceList_KeywordHitName keyword 命中 name 字段
func TestHandleInstanceList_KeywordHitName(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateNamedInstance(t, user.ID, "alpha-prod", "ins-001", "")
	instanceListCreateNamedInstance(t, user.ID, "beta-staging", "ins-002", "")
	instanceListCreateNamedInstance(t, user.ID, "alpha-staging", "ins-003", "")

	req := instanceListJSONReq(t, "GET", "/openclaw/list?keyword=alpha", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	instances, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 2 {
		t.Errorf("keyword=alpha 应命中 2 条 (name 含 alpha)，实际 total=%d", total)
	}
	names := instanceListCollectNames(instances)
	if !names["alpha-prod"] || !names["alpha-staging"] {
		t.Errorf("应命中 alpha-prod 和 alpha-staging，实际 names=%v", names)
	}
	if names["beta-staging"] {
		t.Errorf("不应命中 beta-staging")
	}
}

// TestHandleInstanceList_KeywordHitInstanceID keyword 命中 instance_id 字段
func TestHandleInstanceList_KeywordHitInstanceID(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateNamedInstance(t, user.ID, "first", "ins-abc-001", "")
	instanceListCreateNamedInstance(t, user.ID, "second", "ins-xyz-002", "")
	instanceListCreateNamedInstance(t, user.ID, "third", "ins-abc-003", "")

	req := instanceListJSONReq(t, "GET", "/openclaw/list?keyword=abc", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	_, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 2 {
		t.Errorf("keyword=abc 应命中 2 条 (instance_id 含 abc)，实际 total=%d", total)
	}
}

// TestHandleInstanceList_KeywordIgnoredWhenIDProvided 同时传 id 和 keyword 时,id 优先,keyword 被忽略
func TestHandleInstanceList_KeywordIgnoredWhenIDProvided(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateNamedInstance(t, user.ID, "alpha-1", "ins-001", "")
	instanceListCreateNamedInstance(t, user.ID, "alpha-2", "ins-002", "")

	var first model.Instance
	model.DB(context.Background()).Where("name = ?", "alpha-1").First(&first)

	// 同时传 id 和会命中 2 条的 keyword,应只返回 id 命中的 1 条
	req := instanceListJSONReq(t, "GET", fmt.Sprintf("/openclaw/list?id=%d&keyword=alpha", first.ID), "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	instances, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 1 {
		t.Errorf("id 优先时 total 应为 1，实际=%d", total)
	}
	names := instanceListCollectNames(instances)
	if !names["alpha-1"] {
		t.Errorf("应只命中 alpha-1，实际 names=%v", names)
	}
}

// TestHandleInstanceList_KeywordIgnoredWhenInstanceIDProvided 同时传 instance_id 和 keyword 时,instance_id 优先
func TestHandleInstanceList_KeywordIgnoredWhenInstanceIDProvided(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateNamedInstance(t, user.ID, "alpha-1", "ins-001", "")
	instanceListCreateNamedInstance(t, user.ID, "alpha-2", "ins-002", "")

	req := instanceListJSONReq(t, "GET", "/openclaw/list?instance_id=ins-002&keyword=alpha", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	instances, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 1 {
		t.Errorf("instance_id 优先时 total 应为 1，实际=%d", total)
	}
	names := instanceListCollectNames(instances)
	if !names["alpha-2"] {
		t.Errorf("应只命中 alpha-2，实际 names=%v", names)
	}
}

// TestHandleInstanceList_KeywordEmptyAndWhitespace 空白/纯空格 keyword 等同于不传
func TestHandleInstanceList_KeywordEmptyAndWhitespace(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateInstances(t, user.ID, 3)

	cases := []string{"", "%20%20%20"} // 空 / URL-encoded 三个空格
	for _, kw := range cases {
		path := "/openclaw/list?keyword=" + kw
		req := instanceListJSONReq(t, "GET", path, "u1")
		rr := httptest.NewRecorder()
		HandleInstanceList(rr, req)
		if rr.Code != 200 {
			t.Fatalf("keyword=%q 应 200，实际=%d", kw, rr.Code)
		}
		_, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
		if total != 3 {
			t.Errorf("keyword=%q 应返回全部 3 条，实际 total=%d", kw, total)
		}
	}
}

// TestHandleInstanceList_KeywordCrossUserIsolation keyword 不会越过 user_id 隔离
func TestHandleInstanceList_KeywordCrossUserIsolation(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	u1 := &model.User{Username: "u1", Password: "x", Role: "user"}
	u2 := &model.User{Username: "u2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(u1)
	model.DB(context.Background()).Create(u2)

	instanceListCreateNamedInstance(t, u1.ID, "alpha-mine", "ins-a-001", "")
	instanceListCreateNamedInstance(t, u2.ID, "alpha-other", "ins-a-002", "")

	req := instanceListJSONReq(t, "GET", "/openclaw/list?keyword=alpha", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}
	instances, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 1 {
		t.Errorf("跨用户隔离失败，total 应为 1，实际=%d", total)
	}
	if !instanceListCollectNames(instances)["alpha-mine"] {
		t.Errorf("应只命中 alpha-mine，实际 names=%v", instanceListCollectNames(instances))
	}
}

// TestHandleInstanceList_KeywordRuneTruncation 超长 keyword 按 rune 截断,不会切坏多字节 UTF-8 字符
//
// 构造:50 个中文 rune (150 字节) 作为 instance name,keyword 为这 50 个 rune + 末尾再追加 2 个 rune
// (共 52 rune / 156 字节)。
//   - 按 rune 截断 (正确):keyword -> 50 个 rune -> LIKE %前50中文% 命中 name (相等比较成立)
//   - 按 byte 截断 (错误):keyword[:50] -> 16 完整中文 + 2 残余字节,SQLite LIKE 无法命中
//
// total != 1 即说明截断把汉字切坏了。
func TestHandleInstanceList_KeywordRuneTruncation(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	const ch = "中" // 单个汉字 = 3 字节
	nameRunes := make([]rune, 50)
	for i := range nameRunes {
		nameRunes[i] = []rune(ch)[0]
	}
	name := string(nameRunes)
	instanceListCreateNamedInstance(t, user.ID, name, "ins-cn-trunc", "")

	// keyword: 52 个汉字 (前 50 与 name 相同,末尾多 2 个),触发 rune 截断
	keyword := name + "尾巴"
	if utf8RuneCount := len([]rune(keyword)); utf8RuneCount != 52 {
		t.Fatalf("测试构造错误,keyword rune 数=%d", utf8RuneCount)
	}

	req := instanceListJSONReq(t, "GET", "/openclaw/list?keyword="+url.QueryEscape(keyword), "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	_, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 1 {
		t.Errorf("rune 截断后 keyword 应等于 name,LIKE 命中 1 条,实际 total=%d", total)
	}
}

// TestHandleInstanceList_AgentTypeSingle 按单一 agent_type 过滤
func TestHandleInstanceList_AgentTypeSingle(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateNamedInstance(t, user.ID, "openclaw-1", "ins-001", model.AgentTypeOpenClaw)
	instanceListCreateNamedInstance(t, user.ID, "openclaw-2", "ins-002", model.AgentTypeOpenClaw)
	instanceListCreateNamedInstance(t, user.ID, "hermes-1", "ins-003", model.AgentTypeHermes)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?agent_type=hermes", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}
	instances, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 1 {
		t.Errorf("agent_type=hermes 应命中 1 条，实际 total=%d", total)
	}
	if !instanceListCollectNames(instances)["hermes-1"] {
		t.Errorf("应只返回 hermes-1，实际 names=%v", instanceListCollectNames(instances))
	}
}

// TestHandleInstanceList_AgentTypeMultiOR 多值 OR + 去重 + 空白片段被剔除
func TestHandleInstanceList_AgentTypeMultiOR(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateNamedInstance(t, user.ID, "openclaw-1", "ins-001", model.AgentTypeOpenClaw)
	instanceListCreateNamedInstance(t, user.ID, "hermes-1", "ins-002", model.AgentTypeHermes)
	instanceListCreateNamedInstance(t, user.ID, "ace-1", "ins-003", model.AgentTypeLightclawACE)

	// 同时传两个值,中间夹空白片段和重复值,验证解析的健壮性
	req := instanceListJSONReq(t, "GET", "/openclaw/list?agent_type=openclaw,,hermes,openclaw", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}
	instances, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 2 {
		t.Errorf("agent_type=openclaw,hermes 应命中 2 条，实际 total=%d", total)
	}
	names := instanceListCollectNames(instances)
	if !names["openclaw-1"] || !names["hermes-1"] {
		t.Errorf("应命中 openclaw-1 和 hermes-1，实际 names=%v", names)
	}
	if names["ace-1"] {
		t.Errorf("不应命中 ace-1")
	}
}

// TestHandleInstanceList_AgentTypeEmptyIgnored 空 / 纯空白 agent_type 视为不过滤
func TestHandleInstanceList_AgentTypeEmptyIgnored(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateNamedInstance(t, user.ID, "openclaw-1", "ins-001", model.AgentTypeOpenClaw)
	instanceListCreateNamedInstance(t, user.ID, "hermes-1", "ins-002", model.AgentTypeHermes)

	cases := []string{"", "%20%20%20", ",,,%20,"}
	for _, v := range cases {
		req := instanceListJSONReq(t, "GET", "/openclaw/list?agent_type="+v, "u1")
		rr := httptest.NewRecorder()
		HandleInstanceList(rr, req)
		if rr.Code != 200 {
			t.Fatalf("agent_type=%q 应 200，实际=%d", v, rr.Code)
		}
		_, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
		if total != 2 {
			t.Errorf("agent_type=%q 应不过滤,返回全部 2 条,实际 total=%d", v, total)
		}
	}
}

// TestHandleInstanceList_KeywordAndAgentTypeCombined keyword + agent_type 同时存在,按 AND 组合
func TestHandleInstanceList_KeywordAndAgentTypeCombined(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	instanceListCreateNamedInstance(t, user.ID, "alpha-openclaw", "ins-001", model.AgentTypeOpenClaw)
	instanceListCreateNamedInstance(t, user.ID, "alpha-hermes", "ins-002", model.AgentTypeHermes)
	instanceListCreateNamedInstance(t, user.ID, "beta-openclaw", "ins-003", model.AgentTypeOpenClaw)

	req := instanceListJSONReq(t, "GET", "/openclaw/list?keyword=alpha&agent_type=openclaw", "u1")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != 200 {
		t.Fatalf("应 200，实际=%d", rr.Code)
	}
	instances, total, _, _, _ := instanceListParsePaginatedResp(t, rr.Body.Bytes())
	if total != 1 {
		t.Errorf("AND 组合应命中 1 条，实际 total=%d", total)
	}
	if !instanceListCollectNames(instances)["alpha-openclaw"] {
		t.Errorf("应命中 alpha-openclaw，实际 names=%v", instanceListCollectNames(instances))
	}
}

// ---------------------------------------------------------------------------
// 本地实例字段补全测试 (本次需求)
// ---------------------------------------------------------------------------

// TestHandleInstanceList_LocalInstance_FillsHostInfo 验证：
//   - source=local 的实例在 list 接口里返回 host_name / os / last_report_at
//   - source=cvm 的实例不返回这三个字段（omitempty）
func TestHandleInstanceList_LocalInstance_FillsHostInfo(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := model.DB(ctx).AutoMigrate(&model.LocalInstanceInfo{}); err != nil {
		t.Fatalf("migrate LocalInstanceInfo: %v", err)
	}

	user := &model.User{Username: "list-local-user", Role: "user"}
	if err := model.DB(ctx).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	// 一个 local 实例 + 一个 CVM 实例
	localInst := &model.Instance{
		Name: "local-inst", InstanceId: "local-codebuddy-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: model.AgentTypeOpenClaw,
	}
	cvmInst := &model.Instance{
		Name: "cvm-inst", InstanceId: "ins-cvm-001",
		UserID: user.ID, Source: model.InstanceSourceCVM, AgentType: model.AgentTypeOpenClaw,
	}
	if err := model.DB(ctx).Create(localInst).Error; err != nil {
		t.Fatalf("create local instance: %v", err)
	}
	if err := model.DB(ctx).Create(cvmInst).Error; err != nil {
		t.Fatalf("create cvm instance: %v", err)
	}

	// 给 local 实例补 LocalInstanceInfo
	reportAt := mustParseTime(t, "2026-06-29T12:00:00Z")
	info := &model.LocalInstanceInfo{
		InstanceID: localInst.ID, HostName: "alex-mbp", OS: "darwin",
		LastReportAt: &reportAt,
	}
	if err := model.DB(ctx).Create(info).Error; err != nil {
		t.Fatalf("create local info: %v", err)
	}

	req := instanceListJSONReq(t, "GET", "/openclaw/list", "list-local-user")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Instances []map[string]any `json:"instances"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, rr.Body.String())
	}
	if len(resp.Instances) != 2 {
		t.Fatalf("应有 2 条实例，实际=%d", len(resp.Instances))
	}

	var local, cvm map[string]any
	for _, item := range resp.Instances {
		if item["InstanceId"] == "local-codebuddy-001" {
			local = item
		}
		if item["InstanceId"] == "ins-cvm-001" {
			cvm = item
		}
	}
	if local == nil {
		t.Fatalf("未找到 local 实例")
	}
	if local["host_name"] != "alex-mbp" {
		t.Errorf("local.host_name 应=alex-mbp，实际=%v", local["host_name"])
	}
	if local["os"] != "darwin" {
		t.Errorf("local.os 应=darwin，实际=%v", local["os"])
	}
	if got, _ := local["last_report_at"].(string); got == "" {
		t.Errorf("local.last_report_at 应非空")
	}
	// CVM 实例不应有这些字段（omitempty）
	if cvm == nil {
		t.Fatalf("未找到 cvm 实例")
	}
	if _, ok := cvm["host_name"]; ok {
		t.Errorf("cvm 实例不应返回 host_name 字段，实际有：%v", cvm["host_name"])
	}
	if _, ok := cvm["last_report_at"]; ok {
		t.Errorf("cvm 实例不应返回 last_report_at 字段")
	}
}

// mustParseTime 在测试里解析 RFC3339，挂在文件级避免与其他 helper 冲突。
func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("time.Parse(%q): %v", s, err)
	}
	return v
}

// ---------------------------------------------------------------------------
// os_name 字段验证测试
// ---------------------------------------------------------------------------

// TestHandleInstanceList_OsName_FromCVMInfo 验证 CVM 实例通过 batchFetchCVMInfoMap
// 获取到的 OsName 会反映在 list 响应的 os_name 字段中。
func TestHandleInstanceList_OsName_FromCVMInfo(t *testing.T) {
	cleanup := initInstanceListTestDB(t)
	defer cleanup()

	// Mock CVM server：返回 DescribeInstances 成功响应并携带 OsName
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		const osName = "CentOS 7.9 64位"
		fmt.Fprintf(w, `{"Response":{"InstanceSet":[{"InstanceId":"ins-osname-001","InstanceState":"RUNNING","ImageId":"img-test","OsName":%q,"InstanceChargeType":"POSTPAID_BY_HOUR","Tags":[]}],"TotalCount":1,"RequestId":"mock-req-id"}}`, osName)
	}))
	defer ts.Close()

	origCVM := NewCVMClient
	NewCVMClient = func(_ context.Context) (*cvm.Client, error) {
		return newMockCVMClient(t, ts.URL), nil
	}
	defer func() { NewCVMClient = origCVM }()

	user := &model.User{Username: "u-osname", Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	model.DB(context.Background()).Create(&model.Instance{
		Name:       "os-name-cvm",
		InstanceId: "ins-osname-001",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
	})

	req := instanceListJSONReq(t, http.MethodGet, "/openclaw/list", "u-osname")
	rr := httptest.NewRecorder()
	HandleInstanceList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Instances []map[string]interface{} `json:"instances"`
		Total     int                      `json:"total"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v, body=%s", err, rr.Body.String())
	}
	if resp.Total != 1 {
		t.Fatalf("期望 total=1，实际=%d", resp.Total)
	}
	if len(resp.Instances) != 1 {
		t.Fatalf("期望 1 条实例，实际=%d", len(resp.Instances))
	}

	osName, _ := resp.Instances[0]["os_name"].(string)
	if osName != "CentOS 7.9 64位" {
		t.Errorf("os_name 期望 'CentOS 7.9 64位'，实际=%q", osName)
	}
}
