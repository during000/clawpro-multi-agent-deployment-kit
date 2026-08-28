package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ── helpers ────────────────────────────────────────────────────────────────────

// setupOneIDDeptTestDB creates an in-memory SQLite DB with relevant tables for dept tests.
func setupOneIDDeptTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	db.AutoMigrate(
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.GroupClosure{},
		&model.User{},
		&model.Instance{},
		&model.SiteConfig{},
	)
	t.Cleanup(useDBForTestWithSafeRestore(db))
	return db
}

// setupOneIDMockServer creates a mock OneID server that handles token + dept APIs.
// It returns the server and a cleanup func.
func setupOneIDMockServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// makeUnifiedCtx creates a context that passes IsUnifiedAccountMode.
func makeUnifiedCtx() context.Context {
	return common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID:         "app-test",
		OneIDClientID:      "client-test",
		OneIDClientSecret:  "secret-test",
		OneIDTokenEndpoint: "will-be-overridden", // set to mock server later
	})
}

// makeNonUnifiedCtx creates a context that does NOT pass IsUnifiedAccountMode.
func makeNonUnifiedCtx() context.Context {
	return common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID: "", // not unified
	})
}

// clearOneIDCaches resets the package-level caches.
func clearOneIDCaches(t *testing.T, clientID string) {
	t.Helper()
	appTokenMu.Lock()
	delete(appTokenCache, clientID)
	appTokenMu.Unlock()

	rootDeptMu.Lock()
	delete(rootDeptCache, clientID)
	rootDeptMu.Unlock()
}

// oneIDMockMux creates a mock mux that handles token endpoint + department APIs.
// The returned mux supports:
//   - POST /token → returns access_token
//   - POST /openapi/v3/contacts/departments → create dept (returns dept_id from counter)
//   - PATCH /openapi/v3/contacts/departments/{id} → update dept
//   - DELETE /openapi/v3/contacts/departments/{id} → delete dept
//   - GET /openapi/v3/contacts/departments/roots → returns root dept
//   - PATCH /openapi/v3/contacts/users/{id} → update user
func oneIDMockMux(t *testing.T) *http.ServeMux {
	t.Helper()
	var mu sync.Mutex
	deptCounter := 0

	mux := http.NewServeMux()

	// Token endpoint
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock-token-12345",
			"expires_in":   7200,
		})
	})

	// Create department
	mux.HandleFunc("/openapi/v3/contacts/departments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			mu.Lock()
			deptCounter++
			id := deptCounter
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"msg":  "success",
				"data": map[string]interface{}{
					"department_id": "dept-" + string(rune('0'+id)),
				},
			})
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	// Update/Delete department (match prefix)
	mux.HandleFunc("/openapi/v3/contacts/departments/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPatch:
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "success"})
		case http.MethodDelete:
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "success"})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Root department
	mux.HandleFunc("/openapi/v3/contacts/departments/roots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"msg":  "success",
			"data": map[string]interface{}{
				"roots": []map[string]interface{}{
					{"department_id": "root-dept-001"},
				},
			},
		})
	})

	// Update user
	mux.HandleFunc("/openapi/v3/contacts/users/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "success"})
	})

	return mux
}

// withMockOneID sets up the mock server and overrides oneIDAPIBaseURL + token endpoint in ctx.
// Returns ctx with correct token endpoint and cleanup in t.Cleanup.
func withMockOneID(t *testing.T) (context.Context, *httptest.Server) {
	t.Helper()
	mux := oneIDMockMux(t)
	srv := setupOneIDMockServer(t, mux)

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = srv.URL
	t.Cleanup(func() { oneIDAPIBaseURL = origBaseURL })

	// Clear caches so tests get fresh tokens
	clearOneIDCaches(t, "client-test")

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID:         "app-test",
		OneIDClientID:      "client-test",
		OneIDClientSecret:  "secret-test",
		OneIDTokenEndpoint: srv.URL + "/token",
	})
	return ctx, srv
}

// withMockOneIDOverseas sets up the mock server and overrides oneIDAPIBaseURLOverseas.
// Returns an overseas context (DefaultLang=en) and cleanup in t.Cleanup.
func withMockOneIDOverseas(t *testing.T) (context.Context, *httptest.Server) {
	t.Helper()
	mux := oneIDMockMux(t)
	srv := setupOneIDMockServer(t, mux)

	origBaseURL := oneIDAPIBaseURLOverseas
	oneIDAPIBaseURLOverseas = srv.URL
	t.Cleanup(func() { oneIDAPIBaseURLOverseas = origBaseURL })

	clearOneIDCaches(t, "client-overseas")

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID:         "app-overseas",
		OneIDClientID:      "client-overseas",
		OneIDClientSecret:  "secret-overseas",
		OneIDTokenEndpoint: srv.URL + "/token",
		DefaultLang:        "en",
	})
	return ctx, srv
}

// ── OneIDCreateDepartment ──────────────────────────────────────────────────────

func TestOneidUnifiedDept_CreateDepartment_Success(t *testing.T) {
	ctx, _ := withMockOneID(t)

	deptID, err := OneIDCreateDepartment(ctx, "Test Dept", "parent-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deptID == "" {
		t.Fatal("expected non-empty dept_id")
	}
}

func TestOneidUnifiedDept_CreateDepartment_TokenError(t *testing.T) {
	// Context without credentials → token error
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID:    "app-test",
		OneIDClientID: "", // missing
	})

	_, err := OneIDCreateDepartment(ctx, "Test", "parent")
	if err == nil {
		t.Fatal("expected error when credentials not configured")
	}
}

func TestOneidUnifiedDept_CreateDepartment_APIError(t *testing.T) {
	// Mock server returns error code
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "tok",
				"expires_in":   3600,
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 10001,
			"msg":  "invalid parent",
		})
	}))
	defer srv.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = srv.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()
	clearOneIDCaches(t, "client-err")

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID:         "app-err",
		OneIDClientID:      "client-err",
		OneIDClientSecret:  "secret-err",
		OneIDTokenEndpoint: srv.URL + "/token",
	})

	_, err := OneIDCreateDepartment(ctx, "Test", "bad-parent")
	if err == nil {
		t.Fatal("expected error on API error code")
	}
}

// ── OneIDUpdateDepartment ──────────────────────────────────────────────────────

func TestOneidUnifiedDept_UpdateDepartment_Success(t *testing.T) {
	ctx, _ := withMockOneID(t)

	err := OneIDUpdateDepartment(ctx, "dept-123", map[string]interface{}{
		"department_name": "New Name",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOneidUnifiedDept_UpdateDepartment_TokenError(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID:    "app-test",
		OneIDClientID: "",
	})

	err := OneIDUpdateDepartment(ctx, "dept-1", map[string]interface{}{"name": "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOneidUnifiedDept_UpdateDepartment_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "tok", "expires_in": 3600})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 40001, "msg": "not found"})
	}))
	defer srv.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = srv.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()
	clearOneIDCaches(t, "client-upd-err")

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID:         "app-upd-err",
		OneIDClientID:      "client-upd-err",
		OneIDClientSecret:  "secret",
		OneIDTokenEndpoint: srv.URL + "/token",
	})

	err := OneIDUpdateDepartment(ctx, "dept-bad", map[string]interface{}{"name": "x"})
	if err == nil {
		t.Fatal("expected error on API error code")
	}
}

// ── OneIDDeleteDepartment ──────────────────────────────────────────────────────

func TestOneidUnifiedDept_DeleteDepartment_Success(t *testing.T) {
	ctx, _ := withMockOneID(t)

	err := OneIDDeleteDepartment(ctx, "dept-to-delete")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOneidUnifiedDept_DeleteDepartment_TokenError(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID:    "app-test",
		OneIDClientID: "",
	})

	err := OneIDDeleteDepartment(ctx, "dept-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOneidUnifiedDept_DeleteDepartment_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "tok", "expires_in": 3600})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 50001, "msg": "internal error"})
	}))
	defer srv.Close()

	origBaseURL := oneIDAPIBaseURL
	oneIDAPIBaseURL = srv.URL
	defer func() { oneIDAPIBaseURL = origBaseURL }()
	clearOneIDCaches(t, "client-del-err")

	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		OneIDAppID:         "app-del-err",
		OneIDClientID:      "client-del-err",
		OneIDClientSecret:  "secret",
		OneIDTokenEndpoint: srv.URL + "/token",
	})

	err := OneIDDeleteDepartment(ctx, "dept-bad")
	if err == nil {
		t.Fatal("expected error on API error code")
	}
}

// ── oneIDEnsureGroupHasDept ────────────────────────────────────────────────────

func TestOneidUnifiedDept_EnsureGroupHasDept_AlreadyHasSourceRef(t *testing.T) {
	ctx, _ := withMockOneID(t)
	_ = setupOneIDDeptTestDB(t)

	group := &model.UserGroup{
		Name:      "Existing",
		Source:    model.GroupSourceManual,
		SourceRef: "existing-dept-id",
	}

	deptID, err := oneIDEnsureGroupHasDept(ctx, group)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deptID != "existing-dept-id" {
		t.Errorf("expected existing-dept-id, got=%s", deptID)
	}
}

func TestOneidUnifiedDept_EnsureGroupHasDept_NonManualNoRef(t *testing.T) {
	ctx, _ := withMockOneID(t)
	_ = setupOneIDDeptTestDB(t)

	group := &model.UserGroup{
		ID:        99,
		Name:      "OneID Dept",
		Source:    model.GroupSourceOneIDDept,
		SourceRef: "",
	}

	_, err := oneIDEnsureGroupHasDept(ctx, group)
	if err == nil {
		t.Fatal("expected error for non-manual group without source_ref")
	}
}

func TestOneidUnifiedDept_EnsureGroupHasDept_CreatesForTopLevel(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	// Create a manual top-level group (ParentID=0, no source_ref)
	group := &model.UserGroup{
		Name:     "Top Group",
		Source:   model.GroupSourceManual,
		ParentID: 0,
	}
	db.Create(group)

	deptID, err := oneIDEnsureGroupHasDept(ctx, group)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deptID == "" {
		t.Fatal("expected non-empty dept_id")
	}

	// Verify source_ref was saved to DB
	var updated model.UserGroup
	db.First(&updated, group.ID)
	if updated.SourceRef == "" {
		t.Error("expected source_ref to be saved in DB")
	}
	if updated.SourceRef != deptID {
		t.Errorf("source_ref mismatch: DB=%s, returned=%s", updated.SourceRef, deptID)
	}
}

func TestOneidUnifiedDept_EnsureGroupHasDept_CreatesForChild(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	// Parent with existing source_ref
	parent := &model.UserGroup{
		Name:      "Parent",
		Source:    model.GroupSourceManual,
		ParentID:  0,
		SourceRef: "parent-dept-id",
	}
	db.Create(parent)

	// Child without source_ref
	child := &model.UserGroup{
		Name:     "Child",
		Source:   model.GroupSourceManual,
		ParentID: parent.ID,
	}
	db.Create(child)

	deptID, err := oneIDEnsureGroupHasDept(ctx, child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deptID == "" {
		t.Fatal("expected non-empty dept_id for child")
	}
}

// ── oneIDResolveParentDeptID ───────────────────────────────────────────────────

func TestOneidUnifiedDept_ResolveParentDeptID_RootParent(t *testing.T) {
	ctx, _ := withMockOneID(t)
	_ = setupOneIDDeptTestDB(t)

	// ParentID=0 → should return root dept ID
	rootDeptID, err := oneIDResolveParentDeptID(ctx, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rootDeptID != "root-dept-001" {
		t.Errorf("expected root-dept-001, got=%s", rootDeptID)
	}
}

func TestOneidUnifiedDept_ResolveParentDeptID_ExistingParent(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	// Parent with source_ref
	parent := &model.UserGroup{
		Name:      "Parent Group",
		Source:    model.GroupSourceManual,
		ParentID:  0,
		SourceRef: "parent-resolved-dept",
	}
	db.Create(parent)

	deptID, err := oneIDResolveParentDeptID(ctx, parent.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deptID != "parent-resolved-dept" {
		t.Errorf("expected parent-resolved-dept, got=%s", deptID)
	}
}

func TestOneidUnifiedDept_ResolveParentDeptID_ParentNotFound(t *testing.T) {
	ctx, _ := withMockOneID(t)
	_ = setupOneIDDeptTestDB(t)

	_, err := oneIDResolveParentDeptID(ctx, 99999)
	if err == nil {
		t.Fatal("expected error when parent not found")
	}
}

// ── oneIDSyncCreateGroup ───────────────────────────────────────────────────────

func TestOneidUnifiedDept_SyncCreateGroup_NotUnifiedMode(t *testing.T) {
	ctx := makeNonUnifiedCtx()
	_ = setupOneIDDeptTestDB(t)

	group := &model.UserGroup{Name: "G1", Source: model.GroupSourceManual}
	err := oneIDSyncCreateGroup(ctx, group)
	if err != nil {
		t.Fatalf("expected nil error in non-unified mode, got=%v", err)
	}
}

func TestOneidUnifiedDept_SyncCreateGroup_UnifiedMode(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	group := &model.UserGroup{
		Name:     "New Group",
		Source:   model.GroupSourceManual,
		ParentID: 0,
	}
	db.Create(group)

	err := oneIDSyncCreateGroup(ctx, group)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group.SourceRef == "" {
		t.Error("expected SourceRef to be set after sync create")
	}
}

// ── oneIDSyncUpdateGroup ───────────────────────────────────────────────────────

func TestOneidUnifiedDept_SyncUpdateGroup_NotUnifiedMode(t *testing.T) {
	ctx := makeNonUnifiedCtx()

	group := &model.UserGroup{Name: "G", SourceRef: "dept-1"}
	err := oneIDSyncUpdateGroup(ctx, group, true, false)
	if err != nil {
		t.Fatalf("expected nil error in non-unified mode, got=%v", err)
	}
}

func TestOneidUnifiedDept_SyncUpdateGroup_NoSourceRef(t *testing.T) {
	ctx, _ := withMockOneID(t)

	group := &model.UserGroup{Name: "G", SourceRef: ""}
	err := oneIDSyncUpdateGroup(ctx, group, true, false)
	if err != nil {
		t.Fatalf("expected nil when no source_ref, got=%v", err)
	}
}

func TestOneidUnifiedDept_SyncUpdateGroup_NoChanges(t *testing.T) {
	ctx, _ := withMockOneID(t)

	group := &model.UserGroup{Name: "G", SourceRef: "dept-1"}
	err := oneIDSyncUpdateGroup(ctx, group, false, false)
	if err != nil {
		t.Fatalf("expected nil when no changes, got=%v", err)
	}
}

func TestOneidUnifiedDept_SyncUpdateGroup_NameChanged(t *testing.T) {
	ctx, _ := withMockOneID(t)
	_ = setupOneIDDeptTestDB(t)

	group := &model.UserGroup{Name: "New Name", SourceRef: "dept-update-1"}
	err := oneIDSyncUpdateGroup(ctx, group, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOneidUnifiedDept_SyncUpdateGroup_ParentChanged(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	// Create parent with source_ref
	parent := &model.UserGroup{
		Name:      "New Parent",
		Source:    model.GroupSourceManual,
		ParentID:  0,
		SourceRef: "new-parent-dept",
	}
	db.Create(parent)

	group := &model.UserGroup{
		Name:      "Child",
		SourceRef: "child-dept-1",
		ParentID:  parent.ID,
	}

	err := oneIDSyncUpdateGroup(ctx, group, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOneidUnifiedDept_SyncUpdateGroup_BothChanged(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	parent := &model.UserGroup{
		Name:      "Parent",
		Source:    model.GroupSourceManual,
		ParentID:  0,
		SourceRef: "parent-dept-both",
	}
	db.Create(parent)

	group := &model.UserGroup{
		Name:      "Updated Child",
		SourceRef: "child-dept-both",
		ParentID:  parent.ID,
	}

	err := oneIDSyncUpdateGroup(ctx, group, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── oneIDSyncDeleteGroup ───────────────────────────────────────────────────────

func TestOneidUnifiedDept_SyncDeleteGroup_NotUnifiedMode(t *testing.T) {
	ctx := makeNonUnifiedCtx()

	group := &model.UserGroup{Name: "G", SourceRef: "dept-1"}
	err := oneIDSyncDeleteGroup(ctx, group)
	if err != nil {
		t.Fatalf("expected nil in non-unified mode, got=%v", err)
	}
}

func TestOneidUnifiedDept_SyncDeleteGroup_NoSourceRef(t *testing.T) {
	ctx, _ := withMockOneID(t)

	group := &model.UserGroup{Name: "G", SourceRef: ""}
	err := oneIDSyncDeleteGroup(ctx, group)
	if err != nil {
		t.Fatalf("expected nil when no source_ref, got=%v", err)
	}
}

func TestOneidUnifiedDept_SyncDeleteGroup_Success(t *testing.T) {
	ctx, _ := withMockOneID(t)
	_ = setupOneIDDeptTestDB(t)

	group := &model.UserGroup{Name: "To Delete", SourceRef: "dept-del-1"}
	err := oneIDSyncDeleteGroup(ctx, group)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── oneIDResolveDepartmentIDsForGroups ─────────────────────────────────────────

func TestOneidUnifiedDept_ResolveDepartmentIDsForGroups_Empty(t *testing.T) {
	ctx, _ := withMockOneID(t)
	_ = setupOneIDDeptTestDB(t)

	deptIDs, err := oneIDResolveDepartmentIDsForGroups(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deptIDs != nil {
		t.Errorf("expected nil for empty input, got=%v", deptIDs)
	}
}

func TestOneidUnifiedDept_ResolveDepartmentIDsForGroups_WithSourceRef(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	g1 := &model.UserGroup{Name: "G1", Source: model.GroupSourceManual, SourceRef: "dept-a"}
	g2 := &model.UserGroup{Name: "G2", Source: model.GroupSourceManual, SourceRef: "dept-b"}
	db.Create(g1)
	db.Create(g2)

	deptIDs, err := oneIDResolveDepartmentIDsForGroups(ctx, []uint{g1.ID, g2.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deptIDs) != 2 {
		t.Errorf("expected 2 dept IDs, got=%d (%v)", len(deptIDs), deptIDs)
	}
}

func TestOneidUnifiedDept_ResolveDepartmentIDsForGroups_Deduplicates(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	// Two groups with same source_ref → should deduplicate
	g1 := &model.UserGroup{Name: "G1", Source: model.GroupSourceManual, SourceRef: "same-dept"}
	g2 := &model.UserGroup{Name: "G2", Source: model.GroupSourceManual, SourceRef: "same-dept"}
	db.Create(g1)
	db.Create(g2)

	deptIDs, err := oneIDResolveDepartmentIDsForGroups(ctx, []uint{g1.ID, g2.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deptIDs) != 1 {
		t.Errorf("expected 1 deduplicated dept ID, got=%d (%v)", len(deptIDs), deptIDs)
	}
}

func TestOneidUnifiedDept_ResolveDepartmentIDsForGroups_CreatesWhenMissing(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	// Group without source_ref → should create via OneID API
	g := &model.UserGroup{Name: "No Ref", Source: model.GroupSourceManual, ParentID: 0}
	db.Create(g)

	deptIDs, err := oneIDResolveDepartmentIDsForGroups(ctx, []uint{g.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deptIDs) != 1 {
		t.Errorf("expected 1 dept ID, got=%d (%v)", len(deptIDs), deptIDs)
	}
}

func TestOneidUnifiedDept_ResolveDepartmentIDsForGroups_SkipsNonManual(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	// oneid_dept group without source_ref → will fail oneIDEnsureGroupHasDept (non-manual) → skipped
	g := &model.UserGroup{Name: "OneID Group", Source: model.GroupSourceOneIDDept, ParentID: 0}
	db.Create(g)

	deptIDs, err := oneIDResolveDepartmentIDsForGroups(ctx, []uint{g.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deptIDs) != 0 {
		t.Errorf("expected 0 dept IDs (non-manual skipped), got=%d (%v)", len(deptIDs), deptIDs)
	}
}

// ── oneIDSyncAddUsersToDept ────────────────────────────────────────────────────

func TestOneidUnifiedDept_SyncAddUsersToDept_NoSourceRef(t *testing.T) {
	ctx, _ := withMockOneID(t)
	_ = setupOneIDDeptTestDB(t)

	group := &model.UserGroup{Name: "G", SourceRef: ""}
	err := oneIDSyncAddUsersToDept(ctx, []uint{1, 2}, group)
	if err != nil {
		t.Fatalf("expected nil when no source_ref, got=%v", err)
	}
}

func TestOneidUnifiedDept_SyncAddUsersToDept_NoUsersWithOneIDSub(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	// Users without one_id_sub
	db.Create(&model.User{Username: "user1"})
	db.Create(&model.User{Username: "user2"})

	group := &model.UserGroup{Name: "G", SourceRef: "dept-add-1"}
	db.Create(group)

	err := oneIDSyncAddUsersToDept(ctx, []uint{1, 2}, group)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOneidUnifiedDept_SyncAddUsersToDept_WithOneIDSub(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	sub := "union-id-user1"
	u := model.User{Username: "user1", OneIDSub: &sub}
	db.Create(&u)

	group := &model.UserGroup{Name: "Target Group", Source: model.GroupSourceManual, SourceRef: "dept-add-target"}
	db.Create(group)

	// Add user to group membership
	db.Create(&model.UserGroupMember{
		UserGroupID: group.ID,
		UserID:      u.ID,
		Source:      model.MemberSourceManual,
	})

	err := oneIDSyncAddUsersToDept(ctx, []uint{u.ID}, group)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOneidUnifiedDept_SyncAddUsersToDept_MultipleGroups(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	sub := "union-multi-user"
	u := model.User{Username: "multi-user", OneIDSub: &sub}
	db.Create(&u)

	g1 := &model.UserGroup{Name: "Group1", Source: model.GroupSourceManual, SourceRef: "dept-g1"}
	g2 := &model.UserGroup{Name: "Group2", Source: model.GroupSourceManual, SourceRef: "dept-g2"}
	db.Create(g1)
	db.Create(g2)

	// User is member of g1
	db.Create(&model.UserGroupMember{UserGroupID: g1.ID, UserID: u.ID, Source: model.MemberSourceManual})

	// Add user to g2
	err := oneIDSyncAddUsersToDept(ctx, []uint{u.ID}, g2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── oneIDSyncRemoveUsersFromDept ───────────────────────────────────────────────

func TestOneidUnifiedDept_SyncRemoveUsersFromDept_NoSourceRef(t *testing.T) {
	ctx, _ := withMockOneID(t)
	_ = setupOneIDDeptTestDB(t)

	group := &model.UserGroup{Name: "G", SourceRef: ""}
	err := oneIDSyncRemoveUsersFromDept(ctx, []uint{1, 2}, group)
	if err != nil {
		t.Fatalf("expected nil when no source_ref, got=%v", err)
	}
}

func TestOneidUnifiedDept_SyncRemoveUsersFromDept_NoUsersWithOneIDSub(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	db.Create(&model.User{Username: "user-no-sub"})

	group := &model.UserGroup{Name: "G", SourceRef: "dept-rm-1"}
	db.Create(group)

	err := oneIDSyncRemoveUsersFromDept(ctx, []uint{1}, group)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOneidUnifiedDept_SyncRemoveUsersFromDept_UserHasRemainingGroups(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	sub := "union-rm-user"
	u := model.User{Username: "rm-user", OneIDSub: &sub}
	db.Create(&u)

	gRemaining := &model.UserGroup{Name: "Remaining", Source: model.GroupSourceManual, SourceRef: "dept-remaining"}
	gRemoved := &model.UserGroup{Name: "Removed", Source: model.GroupSourceManual, SourceRef: "dept-removed"}
	db.Create(gRemaining)
	db.Create(gRemoved)

	// User still in gRemaining (membership exists)
	db.Create(&model.UserGroupMember{UserGroupID: gRemaining.ID, UserID: u.ID, Source: model.MemberSourceManual})
	// Membership in gRemoved already deleted (simulating post-remove state)

	err := oneIDSyncRemoveUsersFromDept(ctx, []uint{u.ID}, gRemoved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOneidUnifiedDept_SyncRemoveUsersFromDept_UserHasNoGroups_FallbackToRoot(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	sub := "union-no-groups"
	u := model.User{Username: "orphan-user", OneIDSub: &sub}
	db.Create(&u)

	gRemoved := &model.UserGroup{Name: "Last Group", Source: model.GroupSourceManual, SourceRef: "dept-last"}
	db.Create(gRemoved)

	// User has no remaining memberships → should fallback to root dept
	err := oneIDSyncRemoveUsersFromDept(ctx, []uint{u.ID}, gRemoved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOneidUnifiedDept_SyncRemoveUsersFromDept_MultipleUsers(t *testing.T) {
	ctx, _ := withMockOneID(t)
	db := setupOneIDDeptTestDB(t)

	sub1 := "union-rm-multi-1"
	sub2 := "union-rm-multi-2"
	u1 := model.User{Username: "rm-multi-1", OneIDSub: &sub1}
	u2 := model.User{Username: "rm-multi-2", OneIDSub: &sub2}
	db.Create(&u1)
	db.Create(&u2)

	gRemoved := &model.UserGroup{Name: "Removed Multi", Source: model.GroupSourceManual, SourceRef: "dept-rm-multi"}
	db.Create(gRemoved)

	// Both users have no remaining groups
	err := oneIDSyncRemoveUsersFromDept(ctx, []uint{u1.ID, u2.ID}, gRemoved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── overseas URL routing for dept API functions ────────────────────────────

func TestOneidUnifiedDept_CreateDepartment_OverseasURL(t *testing.T) {
	ctx, _ := withMockOneIDOverseas(t)

	deptID, err := OneIDCreateDepartment(ctx, "Overseas Dept", "parent-overseas")
	if err != nil {
		t.Fatalf("unexpected error when using overseas URL: %v", err)
	}
	if deptID == "" {
		t.Fatal("expected non-empty dept_id from overseas endpoint")
	}
}

func TestOneidUnifiedDept_UpdateDepartment_OverseasURL(t *testing.T) {
	ctx, _ := withMockOneIDOverseas(t)

	err := OneIDUpdateDepartment(ctx, "dept-overseas-1", map[string]interface{}{
		"department_name": "Overseas Updated",
	})
	if err != nil {
		t.Fatalf("unexpected error when using overseas URL: %v", err)
	}
}

func TestOneidUnifiedDept_DeleteDepartment_OverseasURL(t *testing.T) {
	ctx, _ := withMockOneIDOverseas(t)

	err := OneIDDeleteDepartment(ctx, "dept-overseas-del")
	if err != nil {
		t.Fatalf("unexpected error when using overseas URL: %v", err)
	}
}
