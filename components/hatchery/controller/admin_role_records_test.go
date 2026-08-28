package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// setupRoleRecordsTestDB 初始化包含角色记录相关表的内存 SQLite 数据库，
// 并设置 AdminToken 使 requireAdmin 可通过 Bearer Token 验证。
func setupRoleRecordsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.RoleDistributionRecord{},
		&model.Instance{},
		&model.OpenClawRole{},
		&model.OpenClawRoleSkill{},
		&model.User{},
		&model.SiteConfig{},
		&model.SkillInstallation{},
		&model.UserGroup{},
		&model.RoleVisibilityGroup{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	t.Cleanup(func() {
		origDB()
		if testSafeDB != nil {
			model.SetDBForTest(testSafeDB)
		}
	})

	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	t.Cleanup(func() { Store = origStore })

	return db
}

// parseTestTime 解析 ISO 时间字符串用于测试。
func parseTestTime(s string) (t time.Time) {
	t, _ = time.Parse(time.RFC3339, s)
	return
}

// ─── HandleAdminRoleRecords ──────────────────────────────────────────────────

func TestHandleAdminRoleRecords_Success(t *testing.T) {
	db := setupRoleRecordsTestDB(t)
	for i := 1; i <= 3; i++ {
		db.Create(&model.RoleDistributionRecord{
			InstanceID: 200, InstanceCID: "ins-200", RoleID: 1, RoleName: "designer",
			Version: "1.0", Status: model.RoleRecordStatusUpdated,
			Source: model.RoleRecordSourceDistribute,
		})
	}
	req := adminRolesReq(http.MethodGet, "/admin/roles/records?instance_ids=200&page=1&page_size=2", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleRecords(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["total"].(float64) != 3 {
		t.Errorf("期望 total=3，实际=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("期望 2 items，实际=%d", len(items))
	}
}

func TestHandleAdminRoleRecords_NoRecords(t *testing.T) {
	setupRoleRecordsTestDB(t)
	req := adminRolesReq(http.MethodGet, "/admin/roles/records?instance_ids=999", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleRecords(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["total"].(float64) != 0 {
		t.Errorf("期望 total=0，实际=%v", resp["total"])
	}
}

func TestHandleAdminRoleRecords_Page2(t *testing.T) {
	db := setupRoleRecordsTestDB(t)
	for i := 1; i <= 3; i++ {
		db.Create(&model.RoleDistributionRecord{
			InstanceID: 300, InstanceCID: "ins-300", RoleID: 1, RoleName: "r",
			Version: "1.0", Status: model.RoleRecordStatusUpdated,
			Source: model.RoleRecordSourceDistribute,
		})
	}
	req := adminRolesReq(http.MethodGet, "/admin/roles/records?instance_ids=300&page=2&page_size=2", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleRecords(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["total"].(float64) != 3 {
		t.Errorf("期望 total=3，实际=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("期望 1 item（第2页），实际=%d", len(items))
	}
}

// ─── 无 instance_ids 查全部 ───────────────────────────────────────────────────

func TestHandleAdminRoleRecords_AllInstances(t *testing.T) {
	db := setupRoleRecordsTestDB(t)
	// 3 个不同实例各 1 条 distribute 记录
	for _, instID := range []uint{100, 200, 300} {
		db.Create(&model.RoleDistributionRecord{
			InstanceID: instID, InstanceCID: "ins-x", RoleID: 1, RoleName: "r",
			Version: "1.0", Status: model.RoleRecordStatusUpdated,
			Source: model.RoleRecordSourceDistribute,
		})
	}
	req := adminRolesReq(http.MethodGet, "/admin/roles/records?page=1&page_size=10", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleRecords(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["total"].(float64) != 3 {
		t.Errorf("期望 total=3（全部实例），实际=%v", resp["total"])
	}
}

// ─── instance_ids 逗号分隔多实例查询 ─────────────────────────────────────────

func TestHandleAdminRoleRecords_MultiInstanceIDs(t *testing.T) {
	db := setupRoleRecordsTestDB(t)
	for _, instID := range []uint{100, 200, 300, 400} {
		db.Create(&model.RoleDistributionRecord{
			InstanceID: instID, InstanceCID: "ins-x", RoleID: 1, RoleName: "r",
			Version: "1.0", Status: model.RoleRecordStatusUpdated,
			Source: model.RoleRecordSourceDistribute,
		})
	}
	// 查 instance_ids=100,300
	req := adminRolesReq(http.MethodGet, "/admin/roles/records?instance_ids=100,300&page=1&page_size=10", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleRecords(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["total"].(float64) != 2 {
		t.Errorf("期望 total=2（instance_ids=100,300），实际=%v", resp["total"])
	}
}

// ─── instance_ids 传了但全无效返回 400 ────────────────────────────────────────

func TestHandleAdminRoleRecords_InvalidInstanceIDs(t *testing.T) {
	setupRoleRecordsTestDB(t)
	// instance_ids=0 全部无效 → 期望 400
	req := adminRolesReq(http.MethodGet, "/admin/roles/records?instance_ids=0&page=1&page_size=10", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleRecords(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望 400（instance_ids 全无效），实际 %d", rr.Code)
	}
}

// ─── source 过滤测试（只返回 distribute，排除 switch/create）─────────────────

func TestHandleAdminRoleRecords_OnlyDistributeRecords(t *testing.T) {
	db := setupRoleRecordsTestDB(t)
	// 2 条 distribute + 1 条 switch + 1 条 create
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 200, Source: model.RoleRecordSourceDistribute,
		Version: "1.0", Status: model.RoleRecordStatusUpdated,
	})
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 200, Source: model.RoleRecordSourceSwitch,
		Version: "2.0", Status: model.RoleRecordStatusUpdated,
	})
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 200, Source: model.RoleRecordSourceCreate,
		Version: "3.0", Status: model.RoleRecordStatusUpdated,
	})
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 200, Source: model.RoleRecordSourceDistribute,
		Version: "4.0", Status: model.RoleRecordStatusFailed,
	})

	req := adminRolesReq(http.MethodGet, "/admin/roles/records?instance_ids=200&page=1&page_size=10", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleRecords(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	// 只返回 2 条 distribute 记录
	if resp["total"].(float64) != 2 {
		t.Errorf("期望 total=2（仅 distribute），实际=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	for _, item := range items {
		m := item.(map[string]interface{})
		if m["source"] != model.RoleRecordSourceDistribute {
			t.Errorf("不应包含非 distribute 记录，实际 source=%v", m["source"])
		}
	}
}

// ─── 鉴权测试 ────────────────────────────────────────────────────────────────

func TestHandleAdminRoleRecords_Unauthorized(t *testing.T) {
	setupRoleRecordsTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/admin/roles/records?instance_ids=1", nil)
	rr := httptest.NewRecorder()
	HandleAdminRoleRecords(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("期望 401/403，实际 %d", rr.Code)
	}
}

// ─── page_size=1 取最新记录 ────────────────────────────────────────────────────

func TestHandleAdminRoleRecords_LatestRecord(t *testing.T) {
	db := setupRoleRecordsTestDB(t)
	now := parseTestTime("2026-07-03T10:00:00Z")
	// 旧记录
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 100, Source: model.RoleRecordSourceDistribute,
		Version: "1.0", Status: model.RoleRecordStatusFailed,
		SoulStatus: model.RoleSubStatusFailed, SoulError: "TAT fail",
	})
	// 最新记录
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 100, Source: model.RoleRecordSourceDistribute,
		Version: "2.0", Status: model.RoleRecordStatusUpdated,
		SoulStatus: model.RoleSubStatusSuccess, SoulSetAt: &now,
	})

	req := adminRolesReq(http.MethodGet, "/admin/roles/records?instance_ids=100&page=1&page_size=1", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleRecords(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("期望 1 item，实际=%d", len(items))
	}
	item := items[0].(map[string]interface{})
	if item["version"] != "2.0" {
		t.Errorf("期望 version=2.0（最新），实际=%v", item["version"])
	}
	if item["status"] != model.RoleRecordStatusUpdated {
		t.Errorf("期望 status=updated，实际=%v", item["status"])
	}
}

// ─── role_ids 逗号分隔多角色过滤 ──────────────────────────────────────────────

func TestHandleAdminRoleRecords_MultiRoleIDs(t *testing.T) {
	db := setupRoleRecordsTestDB(t)
	// 4 条记录，role_id 分别为 7、8、7、9
	for _, rid := range []uint{7, 8, 7, 9} {
		db.Create(&model.RoleDistributionRecord{
			InstanceID: 100, InstanceCID: "ins-x", RoleID: rid, RoleName: "r",
			Version: "1.0", Status: model.RoleRecordStatusUpdated,
			Source: model.RoleRecordSourceDistribute,
		})
	}
	// 只查 role_ids=7,9 → 期望 3 条（7×2 + 9×1）
	req := adminRolesReq(http.MethodGet, "/admin/roles/records?role_ids=7,9&page=1&page_size=10", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleRecords(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["total"].(float64) != 3 {
		t.Errorf("期望 total=3（role_ids=7,9），实际=%v", resp["total"])
	}
	items := resp["items"].([]interface{})
	for _, item := range items {
		m := item.(map[string]interface{})
		rid := uint(m["role_id"].(float64))
		if rid != 7 && rid != 9 {
			t.Errorf("不应包含 role_id=%d 的记录", rid)
		}
	}
}

// ─── role_ids 传了但全无效返回 400 ────────────────────────────────────────────

func TestHandleAdminRoleRecords_InvalidRoleIDs(t *testing.T) {
	setupRoleRecordsTestDB(t)
	// role_ids=0 全部无效 → 期望 400
	req := adminRolesReq(http.MethodGet, "/admin/roles/records?role_ids=0&page=1&page_size=10", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleRecords(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望 400（role_ids 全无效），实际 %d", rr.Code)
	}
}

// ─── role_ids 与 instance_ids 组合过滤 ────────────────────────────────────────

func TestHandleAdminRoleRecords_InstanceAndRoleCombo(t *testing.T) {
	db := setupRoleRecordsTestDB(t)
	// inst=100 role=7, inst=100 role=8, inst=200 role=7, inst=200 role=8
	db.Create(&model.RoleDistributionRecord{InstanceID: 100, RoleID: 7, Source: model.RoleRecordSourceDistribute, Version: "1.0", Status: model.RoleRecordStatusUpdated})
	db.Create(&model.RoleDistributionRecord{InstanceID: 100, RoleID: 8, Source: model.RoleRecordSourceDistribute, Version: "1.0", Status: model.RoleRecordStatusUpdated})
	db.Create(&model.RoleDistributionRecord{InstanceID: 200, RoleID: 7, Source: model.RoleRecordSourceDistribute, Version: "1.0", Status: model.RoleRecordStatusUpdated})
	db.Create(&model.RoleDistributionRecord{InstanceID: 200, RoleID: 8, Source: model.RoleRecordSourceDistribute, Version: "1.0", Status: model.RoleRecordStatusUpdated})
	// instance_ids=100,200 & role_ids=7 → 期望 2 条
	req := adminRolesReq(http.MethodGet, "/admin/roles/records?instance_ids=100,200&role_ids=7&page=1&page_size=10", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleRecords(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["total"].(float64) != 2 {
		t.Errorf("期望 total=2（instance=100,200 & role=7），实际=%v", resp["total"])
	}
}

// ─── operator_username 联表回填 ───────────────────────────────────────────────

func TestHandleAdminRoleRecords_OperatorUsername(t *testing.T) {
	db := setupRoleRecordsTestDB(t)
	// 创建 2 个用户
	user1 := model.User{Username: "alice", Role: "admin"}
	user2 := model.User{Username: "bob", Role: "user"}
	db.Create(&user1)
	db.Create(&user2)

	// 创建 3 条记录：user1 操作 2 条，user2 操作 1 条，operator_id=0 一条
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 100, RoleID: 7, OperatorID: user1.ID,
		Source: model.RoleRecordSourceDistribute, Version: "1.0", Status: model.RoleRecordStatusUpdated,
	})
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 100, RoleID: 8, OperatorID: user2.ID,
		Source: model.RoleRecordSourceDistribute, Version: "1.0", Status: model.RoleRecordStatusUpdated,
	})
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 100, RoleID: 9, OperatorID: user1.ID,
		Source: model.RoleRecordSourceDistribute, Version: "1.0", Status: model.RoleRecordStatusUpdated,
	})
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 100, RoleID: 10, OperatorID: 0,
		Source: model.RoleRecordSourceDistribute, Version: "1.0", Status: model.RoleRecordStatusUpdated,
	})

	req := adminRolesReq(http.MethodGet, "/admin/roles/records?page=1&page_size=10", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleRecords(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 4 {
		t.Fatalf("期望 4 items，实际=%d", len(items))
	}
	// 验证 operator_username 回填正确
	for _, item := range items {
		m := item.(map[string]interface{})
		opID := uint(m["operator_id"].(float64))
		opName, _ := m["operator_username"].(string)
		switch opID {
		case user1.ID:
			if opName != "alice" {
				t.Errorf("期望 operator_username=alice，实际=%s", opName)
			}
		case user2.ID:
			if opName != "bob" {
				t.Errorf("期望 operator_username=bob，实际=%s", opName)
			}
		case 0:
			if opName != "" {
				t.Errorf("operator_id=0 期望空 username，实际=%s", opName)
			}
		}
	}
}

// ─── operator 对应用户被软删除时 username 为空 ─────────────────────────────────

func TestHandleAdminRoleRecords_OperatorSoftDeleted(t *testing.T) {
	db := setupRoleRecordsTestDB(t)
	user := model.User{Username: "deleted-user", Role: "admin"}
	db.Create(&user)
	// 软删除用户
	db.Delete(&user)

	db.Create(&model.RoleDistributionRecord{
		InstanceID: 100, RoleID: 7, OperatorID: user.ID,
		Source: model.RoleRecordSourceDistribute, Version: "1.0", Status: model.RoleRecordStatusUpdated,
	})

	req := adminRolesReq(http.MethodGet, "/admin/roles/records?page=1&page_size=10", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleRecords(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("期望 1 item，实际=%d", len(items))
	}
	m := items[0].(map[string]interface{})
	opName, _ := m["operator_username"].(string)
	if opName != "" {
		t.Errorf("软删除用户的 operator_username 期望为空，实际=%s", opName)
	}
}
