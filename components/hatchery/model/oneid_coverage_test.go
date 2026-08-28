package model

import (
	"context"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupOneIDTestDB(t *testing.T) (cleanup func()) {
	t.Helper()
	f, err := os.CreateTemp("", "test-oneid-*.db")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	f.Close()
	db, err := gorm.Open(sqlite.Open(f.Name()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}
	db.AutoMigrate(&OneIDUserProfile{}, &OneIDDepartmentRecord{})
	orig := UseDBForTest(db)
	dbDriver = "sqlite"
	return func() {
		orig()
		os.Remove(f.Name())
	}
}

func TestUpsertOneIDUserProfile(t *testing.T) {
	cleanup := setupOneIDTestDB(t)
	defer cleanup()

	p := &OneIDUserProfile{OneIDSub: "sub-1", Name: "Alice"}
	if err := UpsertOneIDUserProfile(context.Background(), p); err != nil {
		t.Fatalf("UpsertOneIDUserProfile failed: %v", err)
	}

	// Update
	p.Name = "Alice Updated"
	if err := UpsertOneIDUserProfile(context.Background(), p); err != nil {
		t.Fatalf("UpsertOneIDUserProfile(update) failed: %v", err)
	}

	got, err := GetOneIDUserProfile(context.Background(), "sub-1")
	if err != nil {
		t.Fatalf("GetOneIDUserProfile failed: %v", err)
	}
	if got == nil || got.Name != "Alice Updated" {
		t.Errorf("unexpected profile: %+v", got)
	}
}

func TestGetOneIDUserProfile_Empty(t *testing.T) {
	cleanup := setupOneIDTestDB(t)
	defer cleanup()

	got, err := GetOneIDUserProfile(context.Background(), "")
	if err != nil || got != nil {
		t.Errorf("empty sub should return nil, nil; got %v, %v", got, err)
	}
}

func TestGetOneIDUserProfile_NotFound(t *testing.T) {
	cleanup := setupOneIDTestDB(t)
	defer cleanup()

	got, err := GetOneIDUserProfile(context.Background(), "nonexistent")
	if err != nil || got != nil {
		t.Errorf("not found should return nil, nil; got %v, %v", got, err)
	}
}

func TestGetOneIDUserProfiles_Empty(t *testing.T) {
	cleanup := setupOneIDTestDB(t)
	defer cleanup()

	got := GetOneIDUserProfiles(context.Background(), nil)
	if got != nil {
		t.Errorf("nil subs should return nil")
	}
	got = GetOneIDUserProfiles(context.Background(), []string{})
	if got != nil {
		t.Errorf("empty subs should return nil")
	}
}

func TestGetOneIDUserProfiles_Found(t *testing.T) {
	cleanup := setupOneIDTestDB(t)
	defer cleanup()

	UpsertOneIDUserProfile(context.Background(), &OneIDUserProfile{OneIDSub: "a", Name: "A"})
	UpsertOneIDUserProfile(context.Background(), &OneIDUserProfile{OneIDSub: "b", Name: "B"})

	got := GetOneIDUserProfiles(context.Background(), []string{"a", "b", "c"})
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
}

func TestBuildGlobalDeptMap(t *testing.T) {
	cleanup := setupOneIDTestDB(t)
	defer cleanup()

	UpsertOneIDUserProfile(context.Background(), &OneIDUserProfile{
		OneIDSub:        "u1",
		DepartmentsJSON: `[{"department_id":"d1","department_name":"研发","department_parent_id":""}]`,
	})

	m := BuildGlobalDeptMap(context.Background())
	if len(m) != 1 {
		t.Fatalf("expected 1 dept, got %d", len(m))
	}
	if m["d1"].DepartmentName != "研发" {
		t.Errorf("unexpected dept name: %q", m["d1"].DepartmentName)
	}
}

func TestBuildGlobalDeptMap_InvalidJSON(t *testing.T) {
	cleanup := setupOneIDTestDB(t)
	defer cleanup()

	UpsertOneIDUserProfile(context.Background(), &OneIDUserProfile{
		OneIDSub:        "u-bad",
		DepartmentsJSON: "not-json",
	})

	m := BuildGlobalDeptMap(context.Background())
	if len(m) != 0 {
		t.Errorf("invalid JSON should be skipped, got %d entries", len(m))
	}
}

func TestUpsertDepartment(t *testing.T) {
	cleanup := setupOneIDTestDB(t)
	defer cleanup()

	dept := &OneIDDepartmentRecord{DepartmentID: "dept-1", DepartmentName: "Engineering"}
	if err := UpsertDepartment(context.Background(), dept); err != nil {
		t.Fatalf("UpsertDepartment failed: %v", err)
	}

	got, err := GetDepartment(context.Background(), "dept-1")
	if err != nil {
		t.Fatalf("GetDepartment failed: %v", err)
	}
	if got == nil || got.DepartmentName != "Engineering" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestGetDepartment_Empty(t *testing.T) {
	cleanup := setupOneIDTestDB(t)
	defer cleanup()

	got, err := GetDepartment(context.Background(), "")
	if err != nil || got != nil {
		t.Errorf("empty deptID should return nil")
	}
}

func TestGetDepartment_NotFound(t *testing.T) {
	cleanup := setupOneIDTestDB(t)
	defer cleanup()

	got, err := GetDepartment(context.Background(), "nonexistent")
	if err != nil || got != nil {
		t.Errorf("not found should return nil, nil")
	}
}

func TestDeleteDepartmentsNotIn(t *testing.T) {
	cleanup := setupOneIDTestDB(t)
	defer cleanup()

	UpsertDepartment(context.Background(), &OneIDDepartmentRecord{DepartmentID: "keep"})
	UpsertDepartment(context.Background(), &OneIDDepartmentRecord{DepartmentID: "remove"})

	affected, err := DeleteDepartmentsNotIn(context.Background(), []string{"keep"})
	if err != nil {
		t.Fatalf("DeleteDepartmentsNotIn failed: %v", err)
	}
	if affected != 1 {
		t.Errorf("expected 1 deleted, got %d", affected)
	}
}

func TestDeleteDepartmentsNotIn_Empty(t *testing.T) {
	cleanup := setupOneIDTestDB(t)
	defer cleanup()

	affected, err := DeleteDepartmentsNotIn(context.Background(), nil)
	if err != nil || affected != 0 {
		t.Errorf("empty list should skip deletion")
	}
}

func TestBuildFullDeptMap(t *testing.T) {
	cleanup := setupOneIDTestDB(t)
	defer cleanup()

	UpsertDepartment(context.Background(), &OneIDDepartmentRecord{
		DepartmentID:       "d1",
		DepartmentName:     "Tech",
		DepartmentParentID: "",
	})
	UpsertDepartment(context.Background(), &OneIDDepartmentRecord{
		DepartmentID:       "d2",
		DepartmentName:     "Product",
		DepartmentParentID: "d1",
	})

	m := BuildFullDeptMap(context.Background())
	if len(m) != 2 {
		t.Fatalf("expected 2 depts, got %d", len(m))
	}
	if m["d2"].DepartmentParentID != "d1" {
		t.Errorf("unexpected parent: %q", m["d2"].DepartmentParentID)
	}
}
