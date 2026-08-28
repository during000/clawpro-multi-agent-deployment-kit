package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"hatchery/common"
	"hatchery/model"
)

// ── bindAdminUser：只绑超管（id 最小的 admin）────────────────────────────────

// strPtr 在 admin_instances_test.go 中定义，这里复用。
// newMigrateTestDB / migrateTestCtx / useDBForTestWithSafeRestore 在 migration_test.go 中定义。

func TestBindAdminUser_OnlyBindsInitialAdmin(t *testing.T) {
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))

	// id=1（超管）、id=2（普通 admin）、id=3（普通 user），全部未绑定。
	db.Create(&model.User{Username: "super", Role: "admin"})  // id=1
	db.Create(&model.User{Username: "normal", Role: "admin"}) // id=2
	db.Create(&model.User{Username: "alice", Role: "user"})   // id=3

	job := &JobState{UserMirror: make(map[uint]UserSnapshot)}
	ctx := migrateTestCtx("t")

	bindAdminUser(ctx, job, "ADMIN-UNION", "admin_login")

	// 仅超管被绑定为 AdminUnionID。
	var super, normal model.User
	db.Where("username = ?", "super").First(&super)
	db.Where("username = ?", "normal").First(&normal)

	if super.OneIDSub == nil || *super.OneIDSub != "ADMIN-UNION" {
		t.Errorf("super.OneIDSub = %v, want ADMIN-UNION", super.OneIDSub)
	}
	// 普通 admin 不应被绑成共享超管账号（否则撞唯一约束 + 身份错乱）。
	if normal.OneIDSub != nil && *normal.OneIDSub == "ADMIN-UNION" {
		t.Error("normal admin 被错误绑定为共享 AdminUnionID")
	}

	// mirror 仅记录超管。
	if _, ok := job.UserMirror[super.ID]; !ok {
		t.Error("mirror 缺少超管记录")
	}
	if _, ok := job.UserMirror[normal.ID]; ok {
		t.Error("mirror 不应包含普通 admin")
	}
}

func TestBindAdminUser_SkipsAlreadyBoundSuper(t *testing.T) {
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))

	// 超管已绑定（例如先前迁移已绑过），不应被覆盖。
	db.Create(&model.User{Username: "super", Role: "admin", OneIDSub: strPtr("EXISTING")})
	db.Create(&model.User{Username: "other", Role: "admin"}) // id 更大，非超管

	job := &JobState{UserMirror: make(map[uint]UserSnapshot)}
	ctx := migrateTestCtx("t")

	bindAdminUser(ctx, job, "ADMIN-UNION", "admin_login")

	var super model.User
	db.Where("username = ?", "super").First(&super)
	if super.OneIDSub == nil || *super.OneIDSub != "EXISTING" {
		t.Errorf("已绑定的超管被覆盖: %v", super.OneIDSub)
	}
}

func TestBindAdminUser_NoAdmin(t *testing.T) {
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))
	db.Create(&model.User{Username: "alice", Role: "user"})

	job := &JobState{UserMirror: make(map[uint]UserSnapshot)}
	ctx := migrateTestCtx("t")

	// 无 admin 时不 panic、不报 failure。
	bindAdminUser(ctx, job, "ADMIN-UNION", "admin_login")
	if len(job.failures) != 0 {
		t.Errorf("无 admin 时不应产生 failure，got %d", len(job.failures))
	}
}

// ── migrateOneUser：admin 角色失败后幂等重试 ──────────────────────────────────

// mockAddRoleGateway 启动一个 mock Gateway，记录 add-role 调用次数与期望的成功/失败模式。
type addRoleTracker struct {
	calls int32
	fail  bool // true → 返回 500 触发 OneIDAddRoleUsers 失败
}

func newAddRoleGateway(t *testing.T, tr *addRoleTracker) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/add-role-users" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&tr.calls, 1)
		if tr.fail {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"code":1,"msg":"mock fail"}`))
			return
		}
		w.Write([]byte(`{"code":0,"msg":"ok"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// phase2Ctx 构造含 OneIDAccountID（OneIDAddRoleUsers 必需）+ token endpoint 的 ctx。
func phase2Ctx(tokenURL string) context.Context {
	return common.InjectTenant(context.Background(), common.TenantSnapshot{
		Identifier:         "t",
		OneIDAccountID:     "acct",
		OneIDClientID:      "c",
		OneIDClientSecret:  "s",
		OneIDTokenEndpoint: tokenURL,
		OneIDAppID:         "app",
	})
}

// tokenServer 返回固定 access_token 的 mock OneID token 端点。
func tokenServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"access_token":"tok","expires_in":3600}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// createOneIDSrv 返回 mock OneID OpenAPI：创建用户返回 union_id，根部门返回 root-id。
func createOneIDSrv(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/openapi/v3/contacts/departments/roots":
			w.Write([]byte(`{"code":0,"data":{"roots":[{"department_id":"root-id"}]}}`))
		case "/openapi/v3/contacts/users":
			w.Write([]byte(`{"code":0,"data":{"union_id":"UNION-NEW"}}`))
		default:
			w.Write([]byte(`{"code":0,"data":{}}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestMigrateOneUser_AdminRoleFailureRetries 验证幂等重试：
// 第一次 OneIDAddRoleUsers 失败 → one_id_sub 已写入、mirror 不更新、记 failure。
// 第二次（已绑定 union_id）→ 跳过创建、重试 admin 角色成功。
func TestMigrateOneUser_AdminRoleFailureRetries(t *testing.T) {
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))
	db.Create(&model.User{Username: "admin1", Role: "admin"}) // 未绑定

	tr := &addRoleTracker{fail: true}
	gw := newAddRoleGateway(t, tr)
	origGW := GatewayURL
	GatewayURL = gw.URL
	defer func() { GatewayURL = origGW }()

	tokSrv := tokenServer(t)
	createSrv := createOneIDSrv(t)
	origAPI := oneIDAPIBaseURL
	oneIDAPIBaseURL = createSrv.URL
	defer func() { oneIDAPIBaseURL = origAPI }()

	job := &JobState{
		Config: OneIDMigrateConfig{ClientID: "c", ClientSecret: "s",
			TokenEndpoint: tokSrv.URL, AppID: "app", AccountID: "acct"},
		UserMirror: make(map[uint]UserSnapshot),
	}
	ctx := phase2Ctx(tokSrv.URL)

	// 第一次：创建用户成功，但 admin 角色绑定失败。
	var admin1 model.User
	db.Where("username = ?", "admin1").First(&admin1)
	err1 := migrateOneUser(ctx, job, &admin1)
	if err1 == nil {
		t.Fatal("第一次 admin 角色失败应返回 error")
	}
	if tr.calls != 1 {
		t.Errorf("add-role 调用次数 = %d, want 1", tr.calls)
	}
	// one_id_sub 已写入（创建成功）。
	db.Where("username = ?", "admin1").First(&admin1)
	if admin1.OneIDSub == nil || *admin1.OneIDSub != "UNION-NEW" {
		t.Errorf("one_id_sub = %v, want UNION-NEW", admin1.OneIDSub)
	}
	// mirror 不应更新（角色未同步成功）。
	if _, ok := job.UserMirror[admin1.ID]; ok {
		t.Error("admin 角色失败时 mirror 不应记录已同步")
	}

	// 第二次：gateway 恢复，角色绑定应重试成功。
	tr.fail = false
	db.Where("username = ?", "admin1").First(&admin1)
	if err2 := migrateOneUser(ctx, job, &admin1); err2 != nil {
		t.Fatalf("重试应成功，got %v", err2)
	}
	if tr.calls != 2 {
		t.Errorf("add-role 重试调用次数 = %d, want 2", tr.calls)
	}
	// 成功后 mirror 标记 role=admin 已同步。
	m, ok := job.UserMirror[admin1.ID]
	if !ok {
		t.Fatal("成功后 mirror 应记录")
	}
	if m.Role != "admin" {
		t.Errorf("mirror.Role = %q, want admin", m.Role)
	}
}

// TestMigrateOneUser_AlreadyBoundSkipsCreate 验证已绑定用户复用 union_id、不重复创建。
func TestMigrateOneUser_AlreadyBoundSkipsCreate(t *testing.T) {
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))
	db.Create(&model.User{Username: "admin1", Role: "admin", OneIDSub: strPtr("UNION-EXISTING")})

	tr := &addRoleTracker{}
	gw := newAddRoleGateway(t, tr)
	origGW := GatewayURL
	GatewayURL = gw.URL
	defer func() { GatewayURL = origGW }()

	// 即便 OneID OpenAPI 不可达，已绑定用户也不应调用创建接口，因此不应报错。
	job := &JobState{UserMirror: make(map[uint]UserSnapshot)}
	ctx := phase2Ctx("http://unreachable-token")

	var u model.User
	db.Where("username = ?", "admin1").First(&u)
	if err := migrateOneUser(ctx, job, &u); err != nil {
		t.Fatalf("已绑定用户应跳过创建，got %v", err)
	}
	if tr.calls != 1 {
		t.Errorf("add-role 调用次数 = %d, want 1（admin 角色仍需同步）", tr.calls)
	}
}

// ── runMirrorDiff：role/dept 同步失败时 mirror 不被污染 ────────────────────────

func TestRunMirrorDiff_RoleFailureKeepsStaleMirror(t *testing.T) {
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))
	// id=1 的超管（最小 id admin）：迁移时只绑定、不参与 role/dept 同步，应被排除。
	// 它与被测 alice 区分开，避免 alice 被误判为超管。
	db.Create(&model.User{Username: "super", Role: "admin", OneIDSub: strPtr("SUPER-UNION")})
	// 已绑定 user→admin 升级：DB role=admin，mirror 旧 role=user。
	db.Create(&model.User{Username: "alice", Role: "admin", OneIDSub: strPtr("U1")})

	tr := &addRoleTracker{fail: true}
	gw := newAddRoleGateway(t, tr)
	origGW := GatewayURL
	GatewayURL = gw.URL
	defer func() { GatewayURL = origGW }()

	// 让 dept 恢复成功（无分组 → 走根部门 mock，使 deptSynced=true），
	// 从而隔离出 role 同步失败的单一变量。
	tokSrv := tokenServer(t)
	createSrv := createOneIDSrv(t) // 提供 roots 端点
	origAPI := oneIDAPIBaseURL
	oneIDAPIBaseURL = createSrv.URL
	defer func() { oneIDAPIBaseURL = origAPI }()

	job := &JobState{
		Config: OneIDMigrateConfig{ClientID: "c", ClientSecret: "s",
			TokenEndpoint: tokSrv.URL, AppID: "app", AccountID: "acct"},
		UserMirror: map[uint]UserSnapshot{1: {Role: "user"}}, // 旧 role=user
	}
	ctx := phase2Ctx(tokSrv.URL)

	var alice model.User
	db.Where("username = ?", "alice").First(&alice)
	job.UserMirror[alice.ID] = UserSnapshot{Role: "user"} // 用真实 ID

	roleChanged, _ := runMirrorDiff(ctx, job)

	// role 同步失败 → 不计入 changed。
	if roleChanged != 0 {
		t.Errorf("roleChanged = %d, want 0（add-role 失败）", roleChanged)
	}
	// mirror.Role 仍为旧值 user，下次 reconcile 会重试。
	if m := job.UserMirror[alice.ID]; m.Role != "user" {
		t.Errorf("role 同步失败后 mirror.Role = %q, want user（保留旧值以待重试）", m.Role)
	}
}

func TestRunMirrorDiff_RoleSuccessUpdatesMirror(t *testing.T) {
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))
	// id=1 的超管（最小 id admin），应被 mirror diff 排除。
	db.Create(&model.User{Username: "super", Role: "admin", OneIDSub: strPtr("SUPER-UNION")})
	db.Create(&model.User{Username: "alice", Role: "admin", OneIDSub: strPtr("U1")})

	tr := &addRoleTracker{} // 成功
	gw := newAddRoleGateway(t, tr)
	origGW := GatewayURL
	GatewayURL = gw.URL
	defer func() { GatewayURL = origGW }()

	tokSrv := tokenServer(t)
	createSrv := createOneIDSrv(t)
	origAPI := oneIDAPIBaseURL
	oneIDAPIBaseURL = createSrv.URL
	defer func() { oneIDAPIBaseURL = origAPI }()

	job := &JobState{
		Config: OneIDMigrateConfig{ClientID: "c", ClientSecret: "s",
			TokenEndpoint: tokSrv.URL, AppID: "app", AccountID: "acct"},
		UserMirror: make(map[uint]UserSnapshot),
	}
	ctx := phase2Ctx(tokSrv.URL)

	var alice model.User
	db.Where("username = ?", "alice").First(&alice)
	job.UserMirror[alice.ID] = UserSnapshot{Role: "user"} // 旧 role=user

	roleChanged, _ := runMirrorDiff(ctx, job)

	if roleChanged != 1 {
		t.Errorf("roleChanged = %d, want 1", roleChanged)
	}
	if m := job.UserMirror[alice.ID]; m.Role != "admin" {
		t.Errorf("role 同步成功后 mirror.Role = %q, want admin", m.Role)
	}
}

// TestRunMirrorDiff_NoMirrorAdminFullSync 验证无 mirror（重启首次）时 admin 全量同步。
func TestRunMirrorDiff_NoMirrorAdminFullSync(t *testing.T) {
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))
	// id=1 的超管（最小 id admin），应被 mirror diff 排除。
	db.Create(&model.User{Username: "super", Role: "admin", OneIDSub: strPtr("SUPER-UNION")})
	db.Create(&model.User{Username: "admin1", Role: "admin", OneIDSub: strPtr("U1")})

	tr := &addRoleTracker{}
	gw := newAddRoleGateway(t, tr)
	origGW := GatewayURL
	GatewayURL = gw.URL
	defer func() { GatewayURL = origGW }()

	tokSrv := tokenServer(t)
	createSrv := createOneIDSrv(t)
	origAPI := oneIDAPIBaseURL
	oneIDAPIBaseURL = createSrv.URL
	defer func() { oneIDAPIBaseURL = origAPI }()

	job := &JobState{
		Config: OneIDMigrateConfig{ClientID: "c", ClientSecret: "s",
			TokenEndpoint: tokSrv.URL, AppID: "app", AccountID: "acct"},
		UserMirror: make(map[uint]UserSnapshot), // 空 mirror
	}
	ctx := phase2Ctx(tokSrv.URL)

	roleChanged, _ := runMirrorDiff(ctx, job)

	if roleChanged != 1 {
		t.Errorf("roleChanged = %d, want 1（全量同步 admin）", roleChanged)
	}
	if tr.calls != 1 {
		t.Errorf("add-role 调用 = %d, want 1", tr.calls)
	}
}

// TestRunMirrorDiff_SuperAdminExcluded 验证超管（id 最小的 admin）被排除，
// 即使 role 发生变化也不触发 add-role，避免给预置超管账号重复加角色/改密。
func TestRunMirrorDiff_SuperAdminExcluded(t *testing.T) {
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))
	// id=1 的超管：mirror 旧 role=user（伪装成"待同步"），但应被排除、不被处理。
	db.Create(&model.User{Username: "super", Role: "admin", OneIDSub: strPtr("SUPER-UNION")})

	tr := &addRoleTracker{}
	gw := newAddRoleGateway(t, tr)
	origGW := GatewayURL
	GatewayURL = gw.URL
	defer func() { GatewayURL = origGW }()

	tokSrv := tokenServer(t)
	createSrv := createOneIDSrv(t)
	origAPI := oneIDAPIBaseURL
	oneIDAPIBaseURL = createSrv.URL
	defer func() { oneIDAPIBaseURL = origAPI }()

	var super model.User
	db.Where("username = ?", "super").First(&super)
	job := &JobState{
		Config: OneIDMigrateConfig{ClientID: "c", ClientSecret: "s",
			TokenEndpoint: tokSrv.URL, AppID: "app", AccountID: "acct"},
		// mirror 标记超管旧 role=user，理论上“需同步”，但超管应被排除。
		UserMirror: map[uint]UserSnapshot{super.ID: {Role: "user"}},
	}
	ctx := phase2Ctx(tokSrv.URL)

	roleChanged, _ := runMirrorDiff(ctx, job)

	if roleChanged != 0 {
		t.Errorf("roleChanged = %d, want 0（超管应被排除，不参与 role 同步）", roleChanged)
	}
	if tr.calls != 0 {
		t.Errorf("add-role 调用 = %d, want 0（超管不应被加角色）", tr.calls)
	}
}

// TestRunMigratePhase3_SuperAdminExcludedFromPasswordReset 验证超管不被批量改密。
// 超管绑定的是 OneID 预置超管账号，其密码不该被本地覆盖。
func TestRunMigratePhase3_SuperAdminExcludedFromPasswordReset(t *testing.T) {
	db := newMigrateTestDB(t)
	t.Cleanup(useDBForTestWithSafeRestore(db))

	// 超管：已绑定、有密码 → 若不被排除会被 phase3 改密。
	super := model.User{Username: "super", Role: "admin", OneIDSub: strPtr("SUPER-UNION"),
		Password: "$2a$10$superhash"}
	// 普通用户：已绑定、有密码 → 应被改密。
	normal := model.User{Username: "alice", Role: "user", OneIDSub: strPtr("U1"),
		Password: "$2a$10$normalhash"}
	db.Create(&super)
	db.Create(&normal)

	// gateway mock 记录被改密的 union_ids。
	var resetUnionIDs []string
	mu := sync.Mutex{}
	gwServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if items, ok := body["items"].([]interface{}); ok {
			mu.Lock()
			for _, it := range items {
				if m, ok := it.(map[string]interface{}); ok {
					resetUnionIDs = append(resetUnionIDs, m["union_id"].(string))
				}
			}
			mu.Unlock()
		}
		w.Write([]byte(`{"code":0,"data":{"success_count":0,"fail_count":0,"failures":[]}}`))
	}))
	defer gwServer.Close()
	origGW := GatewayURL
	GatewayURL = gwServer.URL
	defer func() { GatewayURL = origGW }()

	job := &JobState{
		Config:     OneIDMigrateConfig{AccountID: "acct"},
		UserMirror: make(map[uint]UserSnapshot),
	}
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{OneIDAccountID: "acct"})
	result := runMigratePhase3(ctx, job)

	if result.Total == 0 {
		t.Fatal("应有用户被处理（普通用户需改密）")
	}
	mu.Lock()
	defer mu.Unlock()
	for _, uid := range resetUnionIDs {
		if uid == "SUPER-UNION" {
			t.Error("超管 union_id 不应出现在改密请求中")
		}
	}
	// 至少应包含普通用户的 union_id。
	found := false
	for _, uid := range resetUnionIDs {
		if uid == "U1" {
			found = true
		}
	}
	if !found {
		t.Error("普通用户 union_id 应被改密，但未在请求中找到")
	}
}
