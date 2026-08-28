package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

func setupRoleInstancesTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Instance{},
		&model.OpenClawRole{},
		&model.User{},
		&model.SiteConfig{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.RoleVisibilityGroup{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
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

func TestParseRoleSyncStatuses(t *testing.T) {
	tests := []struct {
		input   string
		wantLen int
		wantErr bool
	}{
		{"", 0, false},
		{"all", 0, false},
		{"pending", 1, false},
		{"pending,failed", 2, false},
		{"pending, failed", 2, false},
		{"invalid", 0, true},
		{"pending,invalid", 0, true},
	}
	for _, tt := range tests {
		got, err := parseRoleSyncStatuses(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseRoleSyncStatuses(%q) err=%v, wantErr=%v", tt.input, err, tt.wantErr)
		}
		if !tt.wantErr && len(got) != tt.wantLen {
			t.Errorf("parseRoleSyncStatuses(%q) len=%d, want=%d", tt.input, len(got), tt.wantLen)
		}
	}
}

func TestHandleAdminRoleInstances_Empty(t *testing.T) {
	db := setupRoleInstancesTestDB(t)
	db.Create(&model.OpenClawRole{Name: "test", Version: "1.0", VisibilityType: "all", Visible: true})

	req := adminRolesReq(http.MethodGet, "/admin/roles/instances?role_id=1", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleInstances(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAdminRoleInstances_MultiStatusFilter(t *testing.T) {
	db := setupRoleInstancesTestDB(t)
	db.Create(&model.OpenClawRole{Name: "test", Version: "1.0", VisibilityType: "all", Visible: true})
	db.Create(&model.Instance{Name: "a1", RoleID: 1, DistributedRoleVersion: "1.0", RoleSyncStatus: model.RoleSyncStatusUpdated})
	db.Create(&model.Instance{Name: "a2", RoleID: 1, DistributedRoleVersion: "1.0", RoleSyncStatus: model.RoleSyncStatusPending})
	db.Create(&model.Instance{Name: "a3", RoleID: 1, DistributedRoleVersion: "1.0", RoleSyncStatus: model.RoleSyncStatusFailed})
	db.Create(&model.Instance{Name: "a4", RoleID: 1, DistributedRoleVersion: "1.0", RoleSyncStatus: model.RoleSyncStatusUpdating})

	req := adminRolesReq(http.MethodGet, "/admin/roles/instances?role_id=1&role_sync_status=pending,failed", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleInstances(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
}

func TestHandleAdminRoleInstances_InvalidStatus(t *testing.T) {
	db := setupRoleInstancesTestDB(t)
	db.Create(&model.OpenClawRole{Name: "test", Version: "1.0", VisibilityType: "all", Visible: true})
	req := adminRolesReq(http.MethodGet, "/admin/roles/instances?role_id=1&role_sync_status=invalid", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleInstances(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d", rr.Code)
	}
}

func TestHandleAdminRoleInstances_MissingRoleID(t *testing.T) {
	setupRoleInstancesTestDB(t)
	req := adminRolesReq(http.MethodGet, "/admin/roles/instances", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleInstances(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d", rr.Code)
	}
}

func TestHandleAdminRoleInstances_RoleNotFound(t *testing.T) {
	setupRoleInstancesTestDB(t)
	req := adminRolesReq(http.MethodGet, "/admin/roles/instances?role_id=999", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleInstances(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("期望 404，实际 %d", rr.Code)
	}
}

func TestHandleAdminRoleInstances_Search(t *testing.T) {
	db := setupRoleInstancesTestDB(t)
	db.Create(&model.OpenClawRole{Name: "test", Version: "1.0", VisibilityType: "all", Visible: true})
	db.Create(&model.Instance{Name: "agent-foo", RoleID: 1, DistributedRoleVersion: "1.0", RoleSyncStatus: model.RoleSyncStatusUpdated})
	db.Create(&model.Instance{Name: "agent-bar", RoleID: 1, DistributedRoleVersion: "1.0", RoleSyncStatus: model.RoleSyncStatusUpdated})

	req := adminRolesReq(http.MethodGet, "/admin/roles/instances?role_id=1&search=foo", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleInstances(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
}

func TestHandleAdminRoleInstances_Pagination(t *testing.T) {
	db := setupRoleInstancesTestDB(t)
	db.Create(&model.OpenClawRole{Name: "test", Version: "1.0", VisibilityType: "all", Visible: true})
	for i := 0; i < 5; i++ {
		db.Create(&model.Instance{Name: "agent", RoleID: 1, DistributedRoleVersion: "1.0", RoleSyncStatus: model.RoleSyncStatusUpdated})
	}

	req := adminRolesReq(http.MethodGet, "/admin/roles/instances?role_id=1&page=1&page_size=2", "")
	rr := httptest.NewRecorder()
	HandleAdminRoleInstances(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
}

func TestHandleAdminRoleInstances_Unauthorized(t *testing.T) {
	setupRoleInstancesTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/admin/roles/instances?role_id=1", nil)
	rr := httptest.NewRecorder()
	HandleAdminRoleInstances(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("期望 401/403，实际 %d", rr.Code)
	}
}
