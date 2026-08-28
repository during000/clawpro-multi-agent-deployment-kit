package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ── wrapBcryptPassword ───────────────────────────────────────────────────────

func TestMigration_WrapBcryptPassword(t *testing.T) {
	hash := "$2a$10$abcdefghijklmnopqrstuuVGXzWqB3Kq5E0aO5OqNLAp5AGZRiuCa"
	got := wrapBcryptPassword(hash)

	var parsed struct {
		Hash struct {
			Algorithm string `json:"algorithm"`
			Value     string `json:"value"`
		} `json:"hash"`
	}
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("wrapBcryptPassword output is not valid JSON: %v, got: %s", err, got)
	}
	if parsed.Hash.Algorithm != "bcrypt" {
		t.Errorf("algorithm = %q, want bcrypt", parsed.Hash.Algorithm)
	}
	if parsed.Hash.Value != hash {
		t.Errorf("value = %q, want %q", parsed.Hash.Value, hash)
	}
}

// ── topoSortGroups ───────────────────────────────────────────────────────────

func TestMigration_TopoSortGroups_RootFirst(t *testing.T) {
	groups := []model.UserGroup{
		{ID: 3, ParentID: 1, Name: "child1"},
		{ID: 1, ParentID: 0, Name: "root"},
		{ID: 4, ParentID: 2, Name: "child2"},
		{ID: 2, ParentID: 0, Name: "root2"},
		{ID: 5, ParentID: 3, Name: "grandchild"},
	}
	sorted := topoSortGroups(groups)

	posOf := make(map[uint]int, len(sorted))
	for i, g := range sorted {
		posOf[g.ID] = i
	}
	checks := [][2]uint{{1, 3}, {3, 5}, {2, 4}}
	for _, c := range checks {
		parent, child := c[0], c[1]
		if posOf[parent] >= posOf[child] {
			t.Errorf("group %d (pos %d) should appear before group %d (pos %d)",
				parent, posOf[parent], child, posOf[child])
		}
	}
	if len(sorted) != len(groups) {
		t.Errorf("sorted length = %d, want %d", len(sorted), len(groups))
	}
}

func TestMigration_TopoSortGroups_AllRoot(t *testing.T) {
	groups := []model.UserGroup{
		{ID: 1, ParentID: 0, Name: "a"},
		{ID: 2, ParentID: 0, Name: "b"},
		{ID: 3, ParentID: 0, Name: "c"},
	}
	sorted := topoSortGroups(groups)
	if len(sorted) != 3 {
		t.Errorf("want 3 items, got %d", len(sorted))
	}
}

// ── groupIDsEqual ────────────────────────────────────────────────────────────

func TestMigration_GroupIDsEqual(t *testing.T) {
	cases := []struct {
		a, b []uint
		want bool
	}{
		{[]uint{1, 2, 3}, []uint{3, 2, 1}, true},
		{[]uint{1, 2}, []uint{1, 2, 3}, false},
		{nil, nil, true},
		{[]uint{}, []uint{}, true},
		{[]uint{1}, nil, false},
	}
	for _, c := range cases {
		got := groupIDsEqual(c.a, c.b)
		if got != c.want {
			t.Errorf("groupIDsEqual(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// ── JobState ring buffer ─────────────────────────────────────────────────────

func TestMigration_JobState_FailureRingBuffer(t *testing.T) {
	job := &JobState{UserMirror: make(map[uint]UserSnapshot)}

	for i := 0; i < maxFailures+10; i++ {
		job.addFailure(MigrateFailureRecord{Phase: 1, TargetID: uint(i), Error: "e"})
	}

	all := job.Failures()
	if len(all) != maxFailures {
		t.Errorf("failures len = %d, want %d", len(all), maxFailures)
	}
	if all[0].TargetID != 10 {
		t.Errorf("oldest failure TargetID = %d, want 10", all[0].TargetID)
	}
}

// ── IsMigrating ──────────────────────────────────────────────────────────────

func TestMigration_IsMigrating(t *testing.T) {
	const id = "test-tenant-is-migrating"
	jobs.Delete(id)
	defer jobs.Delete(id)

	if IsMigrating(id) {
		t.Fatal("should not be migrating before storing job")
	}
	jobs.Store(id, &JobState{UserMirror: make(map[uint]UserSnapshot)})
	if !IsMigrating(id) {
		t.Fatal("should be migrating after storing job")
	}
	jobs.Delete(id)
	if IsMigrating(id) {
		t.Fatal("should not be migrating after deleting job")
	}
}

// ── test helpers ─────────────────────────────────────────────────────────────

func newMigrateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	db.AutoMigrate(&model.User{}, &model.SiteConfig{}, &model.UserGroup{}, &model.UserGroupMember{})
	return db
}

func migrateTestCtx(identifier string) context.Context {
	return common.InjectTenant(context.Background(), common.TenantSnapshot{
		Identifier: identifier,
		Domain:     "https://test.example.com",
	})
}

func setupMigrateAdminToken(t *testing.T, req *http.Request) func() {
	t.Helper()
	orig := AdminToken
	AdminToken = "test-migration-token"
	req.Header.Set("Authorization", "Bearer test-migration-token")
	return func() { AdminToken = orig }
}

// ── HandleMigrateInit ────────────────────────────────────────────────────────

func TestMigration_HandleMigrateInit_MissingParams(t *testing.T) {
	const id = "test-init-missing"
	defer jobs.Delete(id)

	origGateway := GatewayURL
	GatewayURL = "http://gw"
	defer func() { GatewayURL = origGateway }()

	// 缺少 oneid_app_id
	body := `{"oneid_account_id":"acct","oneid_client_id":"cid","oneid_client_secret":"sec","oneid_token_endpoint":"https://ep"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/migrate", bytes.NewBufferString(body))
	req = req.WithContext(common.InjectTenant(req.Context(), common.TenantSnapshot{Identifier: id}))
	t.Cleanup(setupMigrateAdminToken(t, req))
	w := httptest.NewRecorder()
	HandleMigrateInit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", w.Code)
	}
}

func TestMigration_HandleMigrateInit_AlreadyUnified(t *testing.T) {
	const id = "test-init-already-unified"
	defer jobs.Delete(id)

	origGateway := GatewayURL
	GatewayURL = "http://gw"
	defer func() { GatewayURL = origGateway }()

	body := `{"oneid_account_id":"a","oneid_app_id":"app","oneid_client_id":"c","oneid_client_secret":"s","oneid_token_endpoint":"https://ep"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/migrate", bytes.NewBufferString(body))
	req = req.WithContext(common.InjectTenant(req.Context(), common.TenantSnapshot{
		Identifier: id, OneIDAppID: "existing-app",
	}))
	t.Cleanup(setupMigrateAdminToken(t, req))
	w := httptest.NewRecorder()
	HandleMigrateInit(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", w.Code)
	}
}

func TestMigration_HandleMigrateInit_Success(t *testing.T) {
	const id = "test-init-ok"
	jobs.Delete(id)
	defer jobs.Delete(id)

	origGateway := GatewayURL
	GatewayURL = "http://gw"
	defer func() { GatewayURL = origGateway }()

	body := `{"oneid_account_id":"acct","oneid_app_id":"app","oneid_client_id":"cid","oneid_client_secret":"sec","oneid_token_endpoint":"https://ep","gateway_internal_secret":"gw-secret","admin_union_id":"uid-123","admin_login_name":"admin_user"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/migrate", bytes.NewBufferString(body))
	req = req.WithContext(common.InjectTenant(req.Context(), common.TenantSnapshot{Identifier: id}))
	t.Cleanup(setupMigrateAdminToken(t, req))
	w := httptest.NewRecorder()
	HandleMigrateInit(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("code = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if !IsMigrating(id) {
		t.Error("job should exist after init")
	}
	v, _ := jobs.Load(id)
	job := v.(*JobState)
	if job.Config.AppID != "app" {
		t.Errorf("AppID = %q, want app", job.Config.AppID)
	}
	if job.Config.GatewaySecret != "gw-secret" {
		t.Errorf("GatewaySecret = %q, want gw-secret", job.Config.GatewaySecret)
	}
	if job.Config.AdminUnionID != "uid-123" {
		t.Errorf("AdminUnionID = %q, want uid-123", job.Config.AdminUnionID)
	}
	if job.Config.AdminLoginName != "admin_user" {
		t.Errorf("AdminLoginName = %q, want admin_user", job.Config.AdminLoginName)
	}
}

// ── HandleMigrateStatus ──────────────────────────────────────────────────────

func TestMigration_HandleMigrateStatus_NoJob(t *testing.T) {
	const id = "test-status-nojob"
	jobs.Delete(id)

	req := httptest.NewRequest(http.MethodGet, "/admin/migrate", nil)
	req = req.WithContext(migrateTestCtx(id))
	t.Cleanup(setupMigrateAdminToken(t, req))
	w := httptest.NewRecorder()
	HandleMigrateStatus(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", w.Code)
	}
}

func TestMigration_HandleMigrateStatus_HasJob(t *testing.T) {
	const id = "test-status-ok"
	defer jobs.Delete(id)

	job := &JobState{
		Config:     OneIDMigrateConfig{AccountID: "acct", AppID: "app", ClientID: "cid"},
		UserMirror: map[uint]UserSnapshot{1: {}, 2: {}},
		Phase3At:   time.Now().Add(-time.Hour),
	}
	job.addFailure(MigrateFailureRecord{Phase: 1, TargetID: 99, Error: "some error"})
	jobs.Store(id, job)

	req := httptest.NewRequest(http.MethodGet, "/admin/migrate", nil)
	req = req.WithContext(migrateTestCtx(id))
	t.Cleanup(setupMigrateAdminToken(t, req))
	w := httptest.NewRecorder()
	HandleMigrateStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	jobData, _ := resp["job"].(map[string]interface{})
	if jobData == nil {
		t.Fatal("missing job field in response")
	}
	if jobData["oneid_app_id"] != "app" {
		t.Errorf("oneid_app_id = %v, want app", jobData["oneid_app_id"])
	}
	if _, ok := jobData["oneid_client_secret"]; ok {
		t.Error("response should not contain oneid_client_secret")
	}
	if cnt, ok := jobData["mirror_user_count"].(float64); !ok || cnt != 2 {
		t.Errorf("mirror_user_count = %v, want 2", jobData["mirror_user_count"])
	}
	// failures 现在直接在响应里
	failuresList, _ := resp["failures"].([]interface{})
	if len(failuresList) != 1 {
		t.Errorf("failures len = %d, want 1", len(failuresList))
	}
}

// ── HandleMigrateRun ─────────────────────────────────────────────────────────

func TestMigration_HandleMigrateRun_NoJob(t *testing.T) {
	const id = "test-run-nojob"
	jobs.Delete(id)

	req := httptest.NewRequest(http.MethodPost, "/admin/migrate/run", nil)
	req = req.WithContext(migrateTestCtx(id))
	t.Cleanup(setupMigrateAdminToken(t, req))
	w := httptest.NewRecorder()
	HandleMigrateRun(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", w.Code)
	}
}

func TestMigration_HandleMigrateRun_AlreadyUnified(t *testing.T) {
	const id = "test-run-unified"
	jobs.Store(id, &JobState{UserMirror: make(map[uint]UserSnapshot)})
	defer jobs.Delete(id)

	req := httptest.NewRequest(http.MethodPost, "/admin/migrate/run", nil)
	req = req.WithContext(common.InjectTenant(req.Context(), common.TenantSnapshot{
		Identifier: id, OneIDAppID: "existing",
	}))
	t.Cleanup(setupMigrateAdminToken(t, req))
	w := httptest.NewRecorder()
	HandleMigrateRun(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", w.Code)
	}
}

// ── Phase 1 logic ─────────────────────────────────────────────────────────────

func TestMigration_Phase1_SkipsNonManual(t *testing.T) {
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))

	db.Create(&model.UserGroup{Name: "g1", Source: model.GroupSourceManual, SourceRef: ""})
	db.Create(&model.UserGroup{Name: "g2", Source: model.GroupSourceManual, SourceRef: "dept-already"})
	db.Create(&model.UserGroup{Name: "g3", Source: model.GroupSourceOneIDDept, SourceRef: ""})

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"access_token":"tok","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openapi/v3/contacts/departments/roots" {
			w.Write([]byte(`{"code":0,"data":{"roots":[{"department_id":"root-id"}]}}`))
		} else {
			w.Write([]byte(`{"code":0,"data":{"department_id":"new-dept-id"}}`))
		}
	}))
	defer mockServer.Close()

	orig := oneIDAPIBaseURL
	oneIDAPIBaseURL = mockServer.URL
	defer func() { oneIDAPIBaseURL = orig }()

	job := &JobState{
		Config:     OneIDMigrateConfig{ClientID: "c", ClientSecret: "s", TokenEndpoint: tokenServer.URL, AppID: "app"},
		UserMirror: make(map[uint]UserSnapshot),
	}
	snap := common.TenantSnapshot{
		OneIDClientID:      "c",
		OneIDClientSecret:  "s",
		OneIDTokenEndpoint: tokenServer.URL,
		OneIDAppID:         "app",
	}
	ctx := common.InjectTenant(context.Background(), snap)

	result := runMigratePhase1(ctx, job)

	if result.Total != 1 {
		t.Errorf("Total = %d, want 1 (only g1 needs sync)", result.Total)
	}
}

// ── Phase 3 logic ─────────────────────────────────────────────────────────────

func TestMigration_Phase3_WatermarkFiltering(t *testing.T) {
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))

	old := time.Now().Add(-24 * time.Hour)
	recent := time.Now().Add(-1 * time.Minute)

	unionOld := "u-old"
	unionRecent := "u-recent"
	db.Create(&model.User{Username: "alice", Password: "$2a$10$hash1", OneIDSub: &unionOld,
		Model: gorm.Model{CreatedAt: old, UpdatedAt: old}})
	db.Create(&model.User{Username: "bob", Password: "$2a$10$hash2", OneIDSub: &unionRecent,
		Model: gorm.Model{CreatedAt: recent, UpdatedAt: recent}})

	gwServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		items := body["items"].([]interface{})
		w.Write([]byte(fmt.Sprintf(`{"code":0,"data":{"success_count":%d,"fail_count":0,"failures":[]}}`, len(items))))
	}))
	defer gwServer.Close()

	origGW := GatewayURL
	GatewayURL = gwServer.URL
	defer func() { GatewayURL = origGW }()

	job := &JobState{
		Config:     OneIDMigrateConfig{AccountID: "acct"},
		UserMirror: make(map[uint]UserSnapshot),
		Phase3At:   old.Add(time.Hour),
	}
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{OneIDAccountID: "acct"})
	result := runMigratePhase3(ctx, job)

	if result.Total != 1 {
		t.Errorf("Total = %d, want 1 (only bob)", result.Total)
	}
	job.mu.Lock()
	updated := job.Phase3At
	job.mu.Unlock()
	if updated.Equal(old.Add(time.Hour)) {
		t.Error("Phase3At should have been updated")
	}
}

func TestMigration_Phase3_EmptyWatermark_AllUsers(t *testing.T) {
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))

	u1, u2 := "uid-1", "uid-2"
	db.Create(&model.User{Username: "u1", Password: "$2a$10$h1", OneIDSub: &u1})
	db.Create(&model.User{Username: "u2", Password: "$2a$10$h2", OneIDSub: &u2})

	gwServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		items := body["items"].([]interface{})
		w.Write([]byte(fmt.Sprintf(`{"code":0,"data":{"success_count":%d,"fail_count":0,"failures":[]}}`, len(items))))
	}))
	defer gwServer.Close()

	origGW := GatewayURL
	GatewayURL = gwServer.URL
	defer func() { GatewayURL = origGW }()

	job := &JobState{Config: OneIDMigrateConfig{AccountID: "acct"}, UserMirror: make(map[uint]UserSnapshot)}
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{OneIDAccountID: "acct"})
	result := runMigratePhase3(ctx, job)

	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
}

// ── HandleMigrateFinalize ────────────────────────────────────────────────────

func TestMigration_Finalize_NoJob(t *testing.T) {
	const id = "test-finalize-nojob"
	jobs.Delete(id)

	req := httptest.NewRequest(http.MethodPost, "/admin/migrate/finalize", nil)
	req = req.WithContext(migrateTestCtx(id))
	t.Cleanup(setupMigrateAdminToken(t, req))
	w := httptest.NewRecorder()
	HandleMigrateFinalize(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", w.Code)
	}
}

func TestMigration_Finalize_AlreadyUnified(t *testing.T) {
	const id = "test-finalize-already"
	jobs.Store(id, &JobState{Config: OneIDMigrateConfig{AccountID: "a", AppID: "app"}, UserMirror: make(map[uint]UserSnapshot)})
	defer jobs.Delete(id)

	req := httptest.NewRequest(http.MethodPost, "/admin/migrate/finalize", nil)
	req = req.WithContext(common.InjectTenant(context.Background(), common.TenantSnapshot{
		Identifier: id, OneIDAppID: "existing",
	}))
	t.Cleanup(setupMigrateAdminToken(t, req))
	w := httptest.NewRecorder()
	HandleMigrateFinalize(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", w.Code)
	}
}

func TestMigration_Finalize_WritesSiteConfig(t *testing.T) {
	const id = "test-finalize-writes"
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))
	db.Create(&model.SiteConfig{Identifier: id})

	jobs.Store(id, &JobState{
		Config: OneIDMigrateConfig{
			AccountID: "acct-123", AppID: "app-456",
			ClientID: "cid", ClientSecret: "sec", TokenEndpoint: "https://ep/token",
			// GatewaySecret 为 tke-tools 派生后下发的 per-tenant 密钥，
			// finalize 必须持久化到 internal_secret，否则切换统一模式后内部接口签名缺失 secret。
			GatewaySecret: "derived-per-tenant-secret",
		},
		UserMirror: make(map[uint]UserSnapshot),
	})
	defer jobs.Delete(id)

	req := httptest.NewRequest(http.MethodPost, "/admin/migrate/finalize?force=true", nil)
	req = req.WithContext(common.InjectTenant(context.Background(), common.TenantSnapshot{
		Identifier: id, Domain: "https://test.example.com",
	}))
	t.Cleanup(setupMigrateAdminToken(t, req))
	w := httptest.NewRecorder()
	HandleMigrateFinalize(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	var cfg model.SiteConfig
	db.Where("identifier = ?", id).First(&cfg)
	if cfg.OneIDAppID != "app-456" {
		t.Errorf("one_id_app_id = %q, want app-456", cfg.OneIDAppID)
	}
	if cfg.OneIDAccountID != "acct-123" {
		t.Errorf("one_id_account_id = %q, want acct-123", cfg.OneIDAccountID)
	}
	if cfg.InternalSecret != "derived-per-tenant-secret" {
		t.Errorf("internal_secret = %q, want derived-per-tenant-secret", cfg.InternalSecret)
	}

	// finalize 后 job 应被清除
	if IsMigrating(id) {
		t.Error("job should be deleted after finalize")
	}
}

// TestMigration_Finalize_EmptyGatewaySecretPreservesExisting 验证 GatewaySecret 为空时
// 不覆盖已有的 internal_secret（避免误清空导致后续内部接口 401）。
func TestMigration_Finalize_EmptyGatewaySecretPreservesExisting(t *testing.T) {
	const id = "test-finalize-empty-secret"
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))
	// 已有 internal_secret（如之前已迁移过）。
	db.Create(&model.SiteConfig{Identifier: id, InternalSecret: "existing-secret"})

	jobs.Store(id, &JobState{
		Config:     OneIDMigrateConfig{AccountID: "a", AppID: "app", ClientID: "c", ClientSecret: "s", TokenEndpoint: "https://ep"},
		UserMirror: make(map[uint]UserSnapshot),
	})
	defer jobs.Delete(id)

	req := httptest.NewRequest(http.MethodPost, "/admin/migrate/finalize?force=true", nil)
	req = req.WithContext(common.InjectTenant(context.Background(), common.TenantSnapshot{
		Identifier: id, Domain: "https://test.example.com",
	}))
	t.Cleanup(setupMigrateAdminToken(t, req))
	w := httptest.NewRecorder()
	HandleMigrateFinalize(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	var cfg model.SiteConfig
	db.Where("identifier = ?", id).First(&cfg)
	if cfg.InternalSecret != "existing-secret" {
		t.Errorf("internal_secret = %q, want existing-secret（空 GatewaySecret 不应覆盖）", cfg.InternalSecret)
	}
}

func TestMigration_Finalize_BlocksUnsyncedUsersWithoutForce(t *testing.T) {
	const id = "test-finalize-block"
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))
	db.Create(&model.SiteConfig{Identifier: id})
	// 有未同步用户
	db.Create(&model.User{Username: "unsynced", Password: "x"})

	jobs.Store(id, &JobState{
		Config:     OneIDMigrateConfig{AccountID: "a", AppID: "app", ClientID: "c", ClientSecret: "s", TokenEndpoint: "https://ep"},
		UserMirror: make(map[uint]UserSnapshot),
	})
	defer jobs.Delete(id)

	req := httptest.NewRequest(http.MethodPost, "/admin/migrate/finalize", nil)
	req = req.WithContext(common.InjectTenant(context.Background(), common.TenantSnapshot{
		Identifier: id, Domain: "https://test.example.com",
	}))
	t.Cleanup(setupMigrateAdminToken(t, req))
	w := httptest.NewRecorder()
	HandleMigrateFinalize(w, req)

	// 无 force 且有未同步用户 → 409
	if w.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409 (blocked by unsynced users)", w.Code)
	}
}

// ── migrateCtx ───────────────────────────────────────────────────────────────

func TestMigration_MigrateCtx_InjectsJobConfig(t *testing.T) {
	const id = "test-migrate-ctx"
	job := &JobState{
		Config: OneIDMigrateConfig{
			AccountID: "acct", AppID: "app-injected",
			ClientID: "cid", ClientSecret: "sec", TokenEndpoint: "https://ep",
		},
		UserMirror: make(map[uint]UserSnapshot),
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = req.WithContext(common.InjectTenant(req.Context(), common.TenantSnapshot{Identifier: id}))

	ctx := migrateCtx(req, job)
	snap, ok := common.GetTenantSnapshot(ctx)
	if !ok {
		t.Fatal("no tenant snapshot in migrateCtx")
	}
	if snap.OneIDAppID != "app-injected" {
		t.Errorf("OneIDAppID = %q, want app-injected", snap.OneIDAppID)
	}
	if snap.Identifier != id {
		t.Errorf("Identifier = %q, want %q", snap.Identifier, id)
	}
}

// ── admin_union_id / admin_login_name 必填 ───────────────────────────────────

func TestMigration_HandleMigrateInit_MissingAdminUnionID(t *testing.T) {
	const id = "test-init-no-admin-uid"
	defer jobs.Delete(id)
	origGateway := GatewayURL
	GatewayURL = "http://gw"
	defer func() { GatewayURL = origGateway }()

	body := `{"oneid_account_id":"acct","oneid_app_id":"app","oneid_client_id":"cid","oneid_client_secret":"sec","oneid_token_endpoint":"https://ep","gateway_internal_secret":"gw-secret","admin_login_name":"admin"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/migrate", bytes.NewBufferString(body))
	req = req.WithContext(common.InjectTenant(req.Context(), common.TenantSnapshot{Identifier: id}))
	t.Cleanup(setupMigrateAdminToken(t, req))
	w := httptest.NewRecorder()
	HandleMigrateInit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400 (missing admin_union_id)", w.Code)
	}
}

func TestMigration_HandleMigrateInit_MissingAdminLoginName(t *testing.T) {
	const id = "test-init-no-admin-login"
	defer jobs.Delete(id)
	origGateway := GatewayURL
	GatewayURL = "http://gw"
	defer func() { GatewayURL = origGateway }()

	body := `{"oneid_account_id":"acct","oneid_app_id":"app","oneid_client_id":"cid","oneid_client_secret":"sec","oneid_token_endpoint":"https://ep","gateway_internal_secret":"gw-secret","admin_union_id":"uid-123"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/migrate", bytes.NewBufferString(body))
	req = req.WithContext(common.InjectTenant(req.Context(), common.TenantSnapshot{Identifier: id}))
	t.Cleanup(setupMigrateAdminToken(t, req))
	w := httptest.NewRecorder()
	HandleMigrateInit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400 (missing admin_login_name)", w.Code)
	}
}

// ── migrateCtx 注入 InternalSecret ──────────────────────────────────────────

func TestMigration_MigrateCtx_InjectsGatewaySecret(t *testing.T) {
	const id = "test-migrate-ctx-secret"
	job := &JobState{
		Config: OneIDMigrateConfig{
			AccountID:      "acct",
			AppID:          "app-injected",
			GatewaySecret:  "my-master-secret",
			AdminUnionID:   "uid-123",
			AdminLoginName: "admin",
		},
		UserMirror: make(map[uint]UserSnapshot),
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = req.WithContext(common.InjectTenant(req.Context(), common.TenantSnapshot{Identifier: id}))

	ctx := migrateCtx(req, job)
	snap, ok := common.GetTenantSnapshot(ctx)
	if !ok {
		t.Fatal("no tenant snapshot in migrateCtx")
	}
	if snap.InternalSecret != "my-master-secret" {
		t.Errorf("InternalSecret = %q, want my-master-secret", snap.InternalSecret)
	}
}

// ── generateMigrateInitPassword 密码复杂度 ───────────────────────────────────

func TestMigration_GenerateInitPassword_Complexity(t *testing.T) {
	for i := 0; i < 20; i++ {
		pwd := generateMigrateInitPassword()
		if len(pwd) < 8 || len(pwd) > 16 {
			t.Errorf("password length %d out of range [8,16]: %q", len(pwd), pwd)
		}
		hasLower, hasUpper, hasDigit, hasSpecial := false, false, false, false
		for _, c := range pwd {
			switch {
			case c >= 'a' && c <= 'z':
				hasLower = true
			case c >= 'A' && c <= 'Z':
				hasUpper = true
			case c >= '0' && c <= '9':
				hasDigit = true
			case c == '!' || c == '@' || c == '#' || c == '$':
				hasSpecial = true
			}
		}
		if !hasLower || !hasUpper || !hasDigit || !hasSpecial {
			t.Errorf("password %q missing required character class (lower=%v upper=%v digit=%v special=%v)",
				pwd, hasLower, hasUpper, hasDigit, hasSpecial)
		}
	}
}

// ── wrapBcryptPassword 小写 algorithm ───────────────────────────────────────

func TestMigration_WrapBcryptPassword_LowercaseAlgorithm(t *testing.T) {
	got := wrapBcryptPassword("$2a$10$test")
	var parsed struct {
		Hash struct {
			Algorithm string `json:"algorithm"`
		} `json:"hash"`
	}
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if parsed.Hash.Algorithm != "bcrypt" {
		t.Errorf("algorithm = %q, want bcrypt (lowercase)", parsed.Hash.Algorithm)
	}
}
