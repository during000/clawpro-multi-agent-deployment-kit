package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

func initAuditTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.AuditLog{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)

	origAdminToken := AdminToken
	origStore := Store
	AdminToken = "test-admin-token"
	Store = sessions.NewCookieStore([]byte("admin-audit-test-secret-key"))

	return func() {
		origDB()
		AdminToken = origAdminToken
		Store = origStore
	}
}

func auditReq(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

func TestHandleAdminAudit_PageSizeLimit(t *testing.T) {
	cleanup := initAuditTestDB(t)
	defer cleanup()

	// 请求 page_size=5000，应被限制为 1000
	r := auditReq("GET", "/admin/audit?page_size=5000")
	w := httptest.NewRecorder()
	HandleAdminAudit(w, r)

	var resp struct {
		PageSize int `json:"page_size"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.PageSize != 1000 {
		t.Errorf("page_size 应为 1000, got %d", resp.PageSize)
	}
}

func TestHandleAdminAudit_DefaultPageSize(t *testing.T) {
	cleanup := initAuditTestDB(t)
	defer cleanup()

	r := auditReq("GET", "/admin/audit")
	w := httptest.NewRecorder()
	HandleAdminAudit(w, r)

	var resp struct {
		PageSize int `json:"page_size"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.PageSize != 20 {
		t.Errorf("默认 page_size 应为 20, got %d", resp.PageSize)
	}
}

func TestHandleAdminAudit_TimeRange(t *testing.T) {
	cleanup := initAuditTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()
	t1 := now.Add(-2 * time.Hour)
	t2 := now.Add(-1 * time.Hour)
	t3 := now.Add(1 * time.Hour)

	// 插入三条日志
	model.DB(ctx).Create(&model.AuditLog{
		StartedAt: t1, Action: "test", Resource: "test", Status: "success",
		CreatedAt: t1,
	})
	model.DB(ctx).Create(&model.AuditLog{
		StartedAt: t2, Action: "test", Resource: "test", Status: "success",
		CreatedAt: t2,
	})
	model.DB(ctx).Create(&model.AuditLog{
		StartedAt: t3, Action: "test", Resource: "test", Status: "success",
		CreatedAt: t3,
	})

	// 不传时间参数，应返回全部
	r := auditReq("GET", "/admin/audit")
	w := httptest.NewRecorder()
	HandleAdminAudit(w, r)
	var resp struct {
		Total int64 `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 3 {
		t.Errorf("无时间筛选应返回 3 条, got %d", resp.Total)
	}

	// 传 start_time，筛选 >= t2
	r = auditReq("GET", "/admin/audit?start_time="+strconv.FormatInt(t2.Unix(), 10))
	w = httptest.NewRecorder()
	HandleAdminAudit(w, r)
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("start_time=%d 应返回 2 条, got %d", t2.Unix(), resp.Total)
	}

	// 传 end_time，筛选 < t3
	r = auditReq("GET", "/admin/audit?end_time="+strconv.FormatInt(t3.Unix(), 10))
	w = httptest.NewRecorder()
	HandleAdminAudit(w, r)
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("end_time=%d 应返回 2 条, got %d", t3.Unix(), resp.Total)
	}

	// 同时传 start_time + end_time，筛选 >=t2 且 < t3，应返回 1 条
	r = auditReq("GET", "/admin/audit?start_time="+strconv.FormatInt(t2.Unix(), 10)+"&end_time="+strconv.FormatInt(t3.Unix(), 10))
	w = httptest.NewRecorder()
	HandleAdminAudit(w, r)
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("start+end 应返回 1 条, got %d", resp.Total)
	}
}

// TestHandleAdminAudit_ActionFilter：按 action 前缀筛选审计记录。
func TestHandleAdminAudit_ActionFilter(t *testing.T) {
	cleanup := initAuditTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// 插入不同 action 的审计记录
	model.DB(ctx).Create(&model.AuditLog{
		StartedAt: now, Action: "agent_bridge_desktop_install", Resource: "instance",
		ResourceID: "ins-001", Status: "success", CreatedAt: now,
	})
	model.DB(ctx).Create(&model.AuditLog{
		StartedAt: now, Action: "agent_bridge_desktop_check", Resource: "instance",
		ResourceID: "ins-001", Status: "success", CreatedAt: now,
	})
	model.DB(ctx).Create(&model.AuditLog{
		StartedAt: now, Action: "desktop_install", Resource: "instance",
		ResourceID: "ins-002", Status: "success", CreatedAt: now,
	})
	model.DB(ctx).Create(&model.AuditLog{
		StartedAt: now, Action: "user_login", Resource: "user",
		Status: "success", CreatedAt: now,
	})

	// 筛选 agent_bridge_ 前缀，应返回 2 条
	r := auditReq("GET", "/admin/audit?action=agent_bridge_")
	w := httptest.NewRecorder()
	HandleAdminAudit(w, r)

	var resp struct {
		Total int64            `json:"total"`
		Logs  []model.AuditLog `json:"logs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("action=agent_bridge_ 应返回 2 条, got %d", resp.Total)
	}

	// 筛选 agent_bridge_desktop_install，应返回 1 条
	r = auditReq("GET", "/admin/audit?action=agent_bridge_desktop_install")
	w = httptest.NewRecorder()
	HandleAdminAudit(w, r)
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("action=agent_bridge_desktop_install 应返回 1 条, got %d", resp.Total)
	}

	// 筛选 user_login，应返回 1 条
	r = auditReq("GET", "/admin/audit?action=user_login")
	w = httptest.NewRecorder()
	HandleAdminAudit(w, r)
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("action=user_login 应返回 1 条, got %d", resp.Total)
	}

	// 筛选不存在的 action，应返回 0 条
	r = auditReq("GET", "/admin/audit?action=nonexistent_")
	w = httptest.NewRecorder()
	HandleAdminAudit(w, r)
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 0 {
		t.Errorf("action=nonexistent_ 应返回 0 条, got %d", resp.Total)
	}
}

// TestHandleAdminAudit_ResourceIDFilter：按 resource_id 精确筛选审计记录。
func TestHandleAdminAudit_ResourceIDFilter(t *testing.T) {
	cleanup := initAuditTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// 插入不同 resource_id 的审计记录
	model.DB(ctx).Create(&model.AuditLog{
		StartedAt: now, Action: "agent_bridge_desktop_install", Resource: "instance",
		ResourceID: "ins-aaa", Status: "success", CreatedAt: now,
	})
	model.DB(ctx).Create(&model.AuditLog{
		StartedAt: now, Action: "agent_bridge_desktop_install", Resource: "instance",
		ResourceID: "ins-aaa", Status: "dispatched", CreatedAt: now,
	})
	model.DB(ctx).Create(&model.AuditLog{
		StartedAt: now, Action: "agent_bridge_desktop_check", Resource: "instance",
		ResourceID: "ins-bbb", Status: "success", CreatedAt: now,
	})

	// 筛选 ins-aaa，应返回 2 条
	r := auditReq("GET", "/admin/audit?resource_id=ins-aaa")
	w := httptest.NewRecorder()
	HandleAdminAudit(w, r)

	var resp struct {
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("resource_id=ins-aaa 应返回 2 条, got %d", resp.Total)
	}

	// 筛选 ins-bbb，应返回 1 条
	r = auditReq("GET", "/admin/audit?resource_id=ins-bbb")
	w = httptest.NewRecorder()
	HandleAdminAudit(w, r)
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("resource_id=ins-bbb 应返回 1 条, got %d", resp.Total)
	}

	// 筛选不存在的 resource_id，应返回 0 条
	r = auditReq("GET", "/admin/audit?resource_id=ins-nonexist")
	w = httptest.NewRecorder()
	HandleAdminAudit(w, r)
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 0 {
		t.Errorf("resource_id=ins-nonexist 应返回 0 条, got %d", resp.Total)
	}
}

// TestHandleAdminAudit_CombinedFilters：组合 action + resource_id + username 筛选。
func TestHandleAdminAudit_CombinedFilters(t *testing.T) {
	cleanup := initAuditTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	model.DB(ctx).Create(&model.AuditLog{
		StartedAt: now, Action: "agent_bridge_desktop_install", Resource: "instance",
		ResourceID: "ins-combo", Username: "alice", Status: "success", CreatedAt: now,
	})
	model.DB(ctx).Create(&model.AuditLog{
		StartedAt: now, Action: "agent_bridge_desktop_install", Resource: "instance",
		ResourceID: "ins-combo", Username: "bob", Status: "failed", CreatedAt: now,
	})
	model.DB(ctx).Create(&model.AuditLog{
		StartedAt: now, Action: "agent_bridge_desktop_check", Resource: "instance",
		ResourceID: "ins-other", Username: "alice", Status: "success", CreatedAt: now,
	})

	// 组合筛选：action=agent_bridge_desktop_install + resource_id=ins-combo + username=alice
	r := auditReq("GET", "/admin/audit?action=agent_bridge_desktop_install&resource_id=ins-combo&username=alice")
	w := httptest.NewRecorder()
	HandleAdminAudit(w, r)

	var resp struct {
		Total int64 `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("组合筛选应返回 1 条, got %d", resp.Total)
	}
}

func TestHandleAdminAudit_UserIDAndUsernameModes(t *testing.T) {
	cleanup := initAuditTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()
	for _, record := range []struct {
		userID   uint
		username string
	}{
		{userID: 1, username: "alice"},
		{userID: 2, username: "malice"},
		{userID: 1, username: "alice-old"},
		{userID: 0, username: "admin-token"},
	} {
		if err := model.DB(ctx).Create(&model.AuditLog{
			StartedAt: now,
			CreatedAt: now,
			UserID:    record.userID,
			Username:  record.username,
			Action:    "user_login",
			Resource:  "user",
			Status:    "success",
		}).Error; err != nil {
			t.Fatalf("创建审计记录失败: %v", err)
		}
	}

	tests := []struct {
		name      string
		query     string
		wantTotal int64
	}{
		{name: "selected user includes historical names", query: "user_id=1", wantTotal: 2},
		{name: "explicit zero is a real filter", query: "user_id=0", wantTotal: 1},
		{name: "username defaults to exact", query: "username=alice", wantTotal: 1},
		{name: "explicit fuzzy username", query: "username=alice&fuzzy=1", wantTotal: 3},
		{name: "non-one fuzzy stays exact", query: "username=ali&fuzzy=true", wantTotal: 0},
		{name: "fuzzy without username adds no filter", query: "fuzzy=1", wantTotal: 4},
		{name: "user id combines with exact username", query: "user_id=1&username=alice-old&action=user_", wantTotal: 1},
		{name: "user id combines with fuzzy username", query: "user_id=1&username=old&fuzzy=1&action=user_", wantTotal: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := auditReq(http.MethodGet, "/admin/audit?"+tt.query)
			w := httptest.NewRecorder()
			HandleAdminAudit(w, r)

			var resp struct {
				Total int64            `json:"total"`
				Logs  []model.AuditLog `json:"logs"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}
			if resp.Total != tt.wantTotal {
				t.Fatalf("total: want %d, got %d", tt.wantTotal, resp.Total)
			}
			if tt.query == "user_id=1" {
				for _, log := range resp.Logs {
					if log.UserID != 1 {
						t.Fatalf("user_id 查询泄漏了其他用户：%+v", resp.Logs)
					}
				}
			}
		})
	}
}

func TestHandleAdminAudit_InvalidUserID(t *testing.T) {
	cleanup := initAuditTestDB(t)
	defer cleanup()

	for _, path := range []string{
		"/admin/audit?user_id=-1",
		"/admin/audit?user_id=not-a-number",
		"/admin/audit?user_id=18446744073709551616",
	} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			HandleAdminAudit(w, auditReq(http.MethodGet, path))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status: want %d, got %d, body=%s", http.StatusBadRequest, w.Code, w.Body.String())
			}
		})
	}
}

func TestAdminAuditHandlers_ListDatabaseError(t *testing.T) {
	cleanup := initAuditTestDB(t)
	defer cleanup()

	db := model.DB(context.Background())
	const callbackName = "audit_test:fail_list_query"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*[]model.AuditLog); ok {
			tx.AddError(errors.New("forced audit list query failure"))
		}
	}); err != nil {
		t.Fatalf("注册列表失败回调失败: %v", err)
	}
	defer func() {
		if err := db.Callback().Query().Remove(callbackName); err != nil {
			t.Errorf("移除列表失败回调失败: %v", err)
		}
	}()

	w := httptest.NewRecorder()
	HandleAdminAudit(w, auditReq(http.MethodGet, "/admin/audit"))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("列表查询错误 status: want %d, got %d, body=%s", http.StatusInternalServerError, w.Code, w.Body.String())
	}
}

func TestAdminAuditHandlers_DatabaseError(t *testing.T) {
	cleanup := initAuditTestDB(t)
	defer cleanup()

	sqlDB, err := model.DB(context.Background()).DB()
	if err != nil {
		t.Fatalf("获取底层数据库失败: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("关闭测试数据库失败: %v", err)
	}

	w := httptest.NewRecorder()
	HandleAdminAudit(w, auditReq(http.MethodGet, "/admin/audit"))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("列表数据库错误 status: want %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
