package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ============================================================================
// 测试基础设施
// ============================================================================

// initAgentCommandsTestDB 创建内存 SQLite + 迁移所需的所有表。
//
// 同时启用 agentDispatchAsyncWG，让 dispatch handler 启动的 goroutine 在
// cleanup 时被 wg.Wait() 等到结束。这是 docs/testing.md「异步任务」一节的
// 强要求 —— 异步 goroutine 不能跨测试存活，否则前一个测试的 goroutine 会
// 在下一个测试 init 新 DB 时使用旧句柄、引发不稳定的 race / sleep-poll 失败
// （TestHandleDispatchAgentCommand_HappyPath 历史 flake 的根因）。
func initAgentCommandsTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// :memory: 单连接 + 长生命周期，避免 distlock 释放连接导致 no such table
	// 详见 docs/testing.md「显式指定 dbDriver」节
	sqlDB, _ := db.DB()
	sqlDB.SetConnMaxIdleTime(0)
	sqlDB.SetConnMaxLifetime(0)
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.AgentCommand{},
		&model.AgentCommandDispatch{},
		&model.AgentCommandInvocation{},
		&model.AgentCommandTask{},
		&model.AgentCommandSchedule{},
		&model.AgentCommandScheduleRecord{},
		&model.User{},
		&model.Instance{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long-pad"))

	var wg sync.WaitGroup
	agentDispatchAsyncWG = &wg
	return func() {
		// 必须先 Wait 异步 goroutine 跑完，再 restore DB；
		// 顺序反了的话 goroutine 还在用旧 DB 句柄就会和下一个测试抢 model.DB(ctx) 赋值产生竞态。
		wg.Wait()
		agentDispatchAsyncWG = nil
		origDB()
		Store = origStore
	}
}

// makeAdminUser 在 DB 创建一个 admin 角色用户并返回其 *User。
func makeAdminUser(t *testing.T, ctx context.Context, username string) *model.User {
	t.Helper()
	u := &model.User{
		Username: username,
		Email:    username + "@example.com",
		Role:     "admin",
	}
	if err := model.DB(ctx).Create(u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

// adminSessionReq 构造一个带管理员 session 的请求，handler 内 getLoginUser 即可拿到 *User。
func adminSessionReq(t *testing.T, method, path string, body any, username string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username
	session.Values["role"] = "admin"
	rr := httptest.NewRecorder()
	if err := session.Save(req, rr); err != nil {
		t.Fatalf("save session: %v", err)
	}
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

// seedDispatchRow 在 DB 中创建一个 agent_command_dispatch 行，返回主键 ID。
//
// 测试中必须先 seed dispatch 才能创建对应的 invocation/task —— v2 数据模型下
// command_id / command_snapshot / param_values / triggered_by 字段已上提到 dispatch 表。
//
// 默认 status=in_progress，调用方可通过 opts 覆盖。
type seedDispatchOpts struct {
	Status               string
	TestFirst            bool
	TestTargetInstanceID uint
	TargetCount          uint
	SuccessCount         uint
	FailedCount          uint
	CancelledCount       uint
	StartedAt            time.Time
	FinishedAt           *time.Time
	ParamValuesJSON      string
}

func seedDispatchRow(t *testing.T, ctx context.Context, slug string, cmdID, ownerID uint, snap string, opts ...seedDispatchOpts) uint {
	t.Helper()
	o := seedDispatchOpts{}
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Status == "" {
		o.Status = model.AgentDispatchStatusInProgress
	}
	if o.ParamValuesJSON == "" {
		o.ParamValuesJSON = "{}"
	}
	if o.StartedAt.IsZero() {
		o.StartedAt = time.Now()
	}
	d := &model.AgentCommandDispatch{
		Slug:                 slug,
		CommandID:            cmdID,
		CommandSnapshot:      snap,
		ParamValuesJSON:      o.ParamValuesJSON,
		TriggeredByUserID:    ownerID,
		TestFirst:            o.TestFirst,
		TestTargetInstanceID: o.TestTargetInstanceID,
		TargetCount:          o.TargetCount,
		SuccessCount:         o.SuccessCount,
		FailedCount:          o.FailedCount,
		CancelledCount:       o.CancelledCount,
		Status:               o.Status,
		StartedAt:            o.StartedAt,
		FinishedAt:           o.FinishedAt,
	}
	if err := model.DB(ctx).Create(d).Error; err != nil {
		t.Fatalf("seed dispatch: %v", err)
	}
	return d.ID
}

// ============================================================================
// 命令模板 CRUD
// ============================================================================

// TestHandleCreateAgentCommand_NameDuplicated 同租户下创建已存在名称返回 409 name_already_exists
func TestHandleCreateAgentCommand_NameDuplicated(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "admin1")
	// 已存在一条
	existing := &model.AgentCommand{
		Name: "清理日志", Content: "echo", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: u.ID,
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, existing, 5); err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
		map[string]any{
			"name":    "清理日志",
			"content": "echo dup",
			"type":    "SHELL",
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommand(rr, req)
	if rr.Code != http.StatusConflict ||
		!strings.Contains(rr.Body.String(), "name_already_exists") {
		t.Errorf("got %d %s, want 409 name_already_exists", rr.Code, rr.Body.String())
	}
}

// TestHandleCreateAgentCommand_NameReusableAfterSoftDelete 软删后同名可重新创建
func TestHandleCreateAgentCommand_NameReusableAfterSoftDelete(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "admin1")
	cmd := &model.AgentCommand{
		Name: "清理日志", Content: "echo", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: u.ID,
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := model.DB(ctx).Delete(cmd).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
		map[string]any{
			"name":    "清理日志",
			"content": "echo new",
			"type":    "SHELL",
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommand(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("got %d %s, want 201 (软删后同名可重用)", rr.Code, rr.Body.String())
	}
}

func TestHandleCreateAgentCommand_Success(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "admin1")

	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
		map[string]any{
			"name":        "清理日志",
			"description": "清理 /tmp 下的 *.log",
			"type":        "SHELL",
			"content":     "#!/bin/bash\necho 'hello {{name}}'",
			"timeout_sec": 60,
			"params": []map[string]any{
				{"name": "name", "default": "world", "description": "对象"},
			},
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommand(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var got agentCommandView
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
	if !strings.HasPrefix(got.Slug, model.AgentCommandSlugPrefix) {
		t.Errorf("slug = %q, want cmd- prefix", got.Slug)
	}
	if got.Content != "#!/bin/bash\necho 'hello {{name}}'" {
		t.Errorf("content not raw: %q", got.Content)
	}
	if !got.CanEdit {
		t.Error("creator should have can_edit=true")
	}
}

func TestHandleCreateAgentCommand_NameInvalid(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "admin1")
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
		map[string]any{
			"name":    "bad/name", // 含 / 非法
			"content": "echo ok",
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommand(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCreateAgentCommand_ContentRequired(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "admin1")
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
		map[string]any{
			"name":    "ok",
			"content": "   \n\t  ",
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommand(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "content_required") {
		t.Errorf("expected content_required in body, got %s", rr.Body.String())
	}
}

func TestHandleCreateAgentCommand_DescriptionTooLong(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "admin1")
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
		map[string]any{
			"name":        "ok",
			"description": strings.Repeat("中", model.AgentCommandDescMaxChars+1),
			"content":     "df -h",
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommand(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "description_too_long") {
		t.Errorf("expected description_too_long in body, got %s", rr.Body.String())
	}
	// 越界请求不应落库
	var count int64
	_ = model.DB(context.Background()).Model(&model.AgentCommand{}).Count(&count).Error
	if count != 0 {
		t.Errorf("DB row count = %d, want 0 (request must not persist on validation failure)", count)
	}
}

func TestHandleCreateAgentCommand_DescriptionAtLimitOK(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "admin1")
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
		map[string]any{
			"name":        "ok",
			"description": strings.Repeat("中", model.AgentCommandDescMaxChars), // 恰好上限
			"content":     "df -h",
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommand(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201; body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCreateAgentCommand_ParamDescriptionTooLong(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "admin1")
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
		map[string]any{
			"name":    "ok",
			"content": "df -h {{x}}",
			"params": []map[string]any{
				{"name": "x", "description": strings.Repeat("中", 201)}, // 超 200 字符上限
			},
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommand(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "param_description_too_long") {
		t.Errorf("expected param_description_too_long in body, got %s", rr.Body.String())
	}
}

func TestHandleCreateAgentCommand_ParamDefaultTooLong(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "admin1")
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
		map[string]any{
			"name":    "ok",
			"content": "df -h {{x}}",
			"params": []map[string]any{
				{"name": "x", "default": strings.Repeat("a", 129)}, // 超 128 字符上限
			},
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommand(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "param_default_too_long") {
		t.Errorf("expected param_default_too_long in body, got %s", rr.Body.String())
	}
}

func TestHandleCreateAgentCommand_ParamsTooMany(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "admin1")
	// 11 项参数 → 超过 10 上限
	params := make([]map[string]any, 11)
	for i := range params {
		params[i] = map[string]any{"name": "p" + string(rune('A'+i))}
	}
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
		map[string]any{
			"name":    "ok",
			"content": "echo hi",
			"params":  params,
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommand(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "params_too_many") {
		t.Errorf("expected params_too_many in body, got %s", rr.Body.String())
	}
}

// TestHandleCreateAgentCommand_ParamErrorOffender 验证 ValidateAgentCommandParams 抛出的
// 参数级错误（name 非法 / 重名 / default 超长 / description 超长）会把冲突参数名透传到
// 响应文案里，避免「参数名格式错误：」冒号后空白的歧义。
func TestHandleCreateAgentCommand_ParamErrorOffender(t *testing.T) {
	// 注意：不在外层 init DB；子用例各自 init/restore，避免 Release/2026_05_22
	// 起 UseDBForTest 加的 testDBLock 因外层持锁、内层再 Lock 而死锁。

	cases := []struct {
		name        string
		params      []map[string]any
		wantInBody  string
		wantOffender string
	}{
		{
			name:         "name starts with digit",
			params:       []map[string]any{{"name": "4DaE9BbB"}},
			wantInBody:   "param_name_invalid",
			wantOffender: "4DaE9BbB",
		},
		{
			name:         "duplicated name",
			params:       []map[string]any{{"name": "x"}, {"name": "x"}},
			wantInBody:   "param_name_duplicated",
			wantOffender: "x",
		},
		{
			name: "default too long",
			params: []map[string]any{
				{"name": "myparam", "default": strings.Repeat("a", model.AgentCommandParamDefaultMax+1)},
			},
			wantInBody:   "param_default_too_long",
			wantOffender: "myparam",
		},
		{
			name: "description too long",
			params: []map[string]any{
				{"name": "another", "description": strings.Repeat("中", model.AgentCommandParamDescMax+1)},
			},
			wantInBody:   "param_description_too_long",
			wantOffender: "another",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer initAgentCommandsTestDB(t)()
			makeAdminUser(t, context.Background(), "admin1")
			req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
				map[string]any{
					"name":    "ok",
					"content": "echo hi",
					"params":  c.params,
				}, "admin1")
			rr := httptest.NewRecorder()
			HandleCreateAgentCommand(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
			}
			body := rr.Body.String()
			if !strings.Contains(body, c.wantInBody) {
				t.Errorf("body = %q, want contains %q", body, c.wantInBody)
			}
			if !strings.Contains(body, c.wantOffender) {
				t.Errorf("body = %q, want contains offender %q", body, c.wantOffender)
			}
		})
	}
}

func TestHandleCreateAgentCommand_TimeoutBoundary(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "admin1")

	cases := []struct {
		name       string
		timeoutSec int
		wantStatus int
		wantInBody string
	}{
		{"at_lower_bound_1s", 1, http.StatusCreated, ""},
		{"default_60s", 60, http.StatusCreated, ""},
		{"middle_3600s", 3600, http.StatusCreated, ""},
		{"at_upper_bound_86400s", 86400, http.StatusCreated, ""},
		{"over_upper_bound_86401s", 86401, http.StatusBadRequest, "timeout_out_of_range"},
		{"zero", 0, http.StatusCreated, ""}, // 0 走默认 60，不报错
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body := map[string]any{
				"name":    "ok-" + c.name,
				"content": "echo hi",
			}
			if c.timeoutSec != 0 {
				body["timeout_sec"] = c.timeoutSec
			}
			req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create", body, "admin1")
			rr := httptest.NewRecorder()
			HandleCreateAgentCommand(rr, req)
			if rr.Code != c.wantStatus {
				t.Errorf("status = %d, want %d; body=%s", rr.Code, c.wantStatus, rr.Body.String())
			}
			if c.wantInBody != "" && !strings.Contains(rr.Body.String(), c.wantInBody) {
				t.Errorf("body = %q, want contains %q", rr.Body.String(), c.wantInBody)
			}
		})
	}
}

func TestHandleCreateAgentCommand_RunUserTooLong(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "admin1")
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
		map[string]any{
			"name":     "ok",
			"content":  "echo hi",
			"run_user": strings.Repeat("a", model.AgentCommandRunUserMaxChars+1), // 65 char
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommand(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "run_user_too_long") {
		t.Errorf("expected run_user_too_long in body, got %s", rr.Body.String())
	}
}

func TestHandleCreateAgentCommand_WorkdirTooLong(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "admin1")
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
		map[string]any{
			"name":    "ok",
			"content": "echo hi",
			"workdir": "/" + strings.Repeat("a", model.AgentCommandWorkdirMaxChars), // 256 char
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommand(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "workdir_too_long") {
		t.Errorf("expected workdir_too_long in body, got %s", rr.Body.String())
	}
}

func TestHandleCreateAgentCommand_RunUserAndWorkdirAtLimit(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "admin1")
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
		map[string]any{
			"name":     "ok-at-limit",
			"content":  "echo hi",
			"run_user": strings.Repeat("u", model.AgentCommandRunUserMaxChars), // 64
			"workdir":  "/" + strings.Repeat("d", model.AgentCommandWorkdirMaxChars-1),
		}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommand(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201; body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCreateAgentCommand_QuotaExceeded(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "admin1")
	// 直接灌 500 条
	for i := 0; i < model.MaxAgentCommandsPerTenant; i++ {
		c := &model.AgentCommand{
			Name:            "ok",
			Content:         "echo " + localItoa(i),
			Type:            "SHELL",
			TimeoutSec:      60,
			RunUser:         "root",
			Workdir:         "/root",
			ParamsJSON:      "[]",
			VisibilityType:  "tenant",
			CreatedByUserID: u.ID,
		}
		if err := model.CreateAgentCommandWithSlugRetry(ctx, c, 5); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/create",
		map[string]any{"name": "ok", "content": "echo last"}, "admin1")
	rr := httptest.NewRecorder()
	HandleCreateAgentCommand(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409; body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleListAgentCommands_FiltersSoftDeleted(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "admin1")
	cmd := &model.AgentCommand{
		Name: "alive", Content: "echo alive", Type: "SHELL",
		TimeoutSec: 60, ParamsJSON: "[]", CreatedByUserID: u.ID,
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5); err != nil {
		t.Fatalf("seed: %v", err)
	}
	deleted := &model.AgentCommand{
		Name: "deleted", Content: "echo deleted", Type: "SHELL",
		TimeoutSec: 60, ParamsJSON: "[]", CreatedByUserID: u.ID,
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, deleted, 5); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := model.DB(ctx).Delete(deleted).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	req := adminSessionReq(t, http.MethodGet, "/admin/agent-commands", nil, "admin1")
	rr := httptest.NewRecorder()
	HandleListAgentCommands(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var list agentCommandListResp
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if list.Total != 1 || len(list.Commands) != 1 {
		t.Errorf("Total=%d Commands=%d, want 1/1", list.Total, len(list.Commands))
	}
	if list.Commands[0].Name != "alive" {
		t.Errorf("name = %q, want alive (soft-deleted should be hidden)", list.Commands[0].Name)
	}
}

// TestHandleListAgentCommands_QSearch q 参数应能跨 4 个字段模糊命中：slug / name / content / 创建人用户名。
//
// 注意：description 不在搜索范围内，符合产品需求。
func TestHandleListAgentCommands_QSearch(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	alice := makeAdminUser(t, ctx, "alice")
	bob := makeAdminUser(t, ctx, "bobby_admin")

	// alice 创建 2 条
	c1 := &model.AgentCommand{
		Name: "清理日志", Description: "rotate /var/log", Content: "rm -rf /var/log/*",
		Type: "SHELL", TimeoutSec: 60, ParamsJSON: "[]", CreatedByUserID: alice.ID,
	}
	c2 := &model.AgentCommand{
		Name: "df-h", Description: "DESC_ONLY_TOKEN_xyz", Content: "df -h",
		Type: "SHELL", TimeoutSec: 60, ParamsJSON: "[]", CreatedByUserID: alice.ID,
	}
	// bob 创建 1 条
	c3 := &model.AgentCommand{
		Name: "ping-google", Description: "网络连通性", Content: "ping -c 3 google.com",
		Type: "SHELL", TimeoutSec: 60, ParamsJSON: "[]", CreatedByUserID: bob.ID,
	}
	for _, c := range []*model.AgentCommand{c1, c2, c3} {
		if err := model.CreateAgentCommandWithSlugRetry(ctx, c, 5); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	doSearch := func(q string) agentCommandListResp {
		req := adminSessionReq(t, http.MethodGet, "/admin/agent-commands?q="+q, nil, "alice")
		rr := httptest.NewRecorder()
		HandleListAgentCommands(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("q=%q status = %d body=%s", q, rr.Code, rr.Body.String())
		}
		var list agentCommandListResp
		_ = json.Unmarshal(rr.Body.Bytes(), &list)
		return list
	}

	cases := []struct {
		q       string
		want    int
		comment string
	}{
		{"清理", 1, "name 中文命中 c1"},
		{"df-h", 1, "name 命中 c2"},
		{"google", 1, "content 命中 c3"},
		{"bobby_admin", 1, "创建人 username 完全匹配 c3"},
		{"bobby", 1, "创建人 username 模糊匹配 c3"},
		{"alice", 2, "创建人 username 命中 alice 的 2 条"},
		{"DESC_ONLY_TOKEN_xyz", 0, "description 不在搜索范围内，应不命中"},
		{"nomatch_xxx_zzz", 0, "全 4 字段都不命中"},
	}
	for _, c := range cases {
		t.Run(c.q, func(t *testing.T) {
			got := doSearch(c.q)
			if int(got.Total) != c.want {
				t.Errorf("q=%q (%s): Total=%d want %d", c.q, c.comment, got.Total, c.want)
			}
		})
	}
}

func TestHandleUpdateAgentCommand_PermissionDenied(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u1 := makeAdminUser(t, ctx, "admin1") // initial admin (smallest id)
	u2 := makeAdminUser(t, ctx, "admin2")
	// admin2 创建命令
	cmd := &model.AgentCommand{
		Name: "owned", Content: "echo", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: u2.ID,
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_ = u1

	// 以另一个非创建者非初始管理员身份编辑 → 403
	u3 := makeAdminUser(t, ctx, "admin3")
	_ = u3
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/update",
		map[string]any{
			"id": cmd.ID, "name": "evil-rename", "content": "echo",
		}, "admin3")
	rr := httptest.NewRecorder()
	HandleUpdateAgentCommand(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403; body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdateAgentCommand_OwnerOK(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner")
	cmd := &model.AgentCommand{
		Name: "old", Content: "old", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: u.ID, RunUser: "root", Workdir: "/root",
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5); err != nil {
		t.Fatalf("seed: %v", err)
	}
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/update",
		map[string]any{
			"id": cmd.ID, "name": "renamed", "content": "echo new",
			"timeout_sec": 60,
		}, "owner")
	rr := httptest.NewRecorder()
	HandleUpdateAgentCommand(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rr.Code, rr.Body.String())
	}
	got, err := model.FindAgentCommandByID(ctx, cmd.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Name != "renamed" || got.Content != "echo new" {
		t.Errorf("update not applied: name=%q content=%q", got.Name, got.Content)
	}
}

// TestHandleUpdateAgentCommand_NameDuplicated 改名为已存在的别人的名 → 409
func TestHandleUpdateAgentCommand_NameDuplicated(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner")
	a := &model.AgentCommand{
		Name: "alpha", Content: "echo", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: u.ID, RunUser: "root", Workdir: "/root",
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, a, 5); err != nil {
		t.Fatalf("seed a: %v", err)
	}
	b := &model.AgentCommand{
		Name: "beta", Content: "echo", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: u.ID, RunUser: "root", Workdir: "/root",
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, b, 5); err != nil {
		t.Fatalf("seed b: %v", err)
	}
	// 把 b 改名成 alpha → 撞 a
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/update",
		map[string]any{"id": b.ID, "name": "alpha", "content": "echo", "timeout_sec": 60},
		"owner")
	rr := httptest.NewRecorder()
	HandleUpdateAgentCommand(rr, req)
	if rr.Code != http.StatusConflict ||
		!strings.Contains(rr.Body.String(), "name_already_exists") {
		t.Errorf("got %d %s, want 409 name_already_exists", rr.Code, rr.Body.String())
	}
}

// TestHandleUpdateAgentCommand_SameNameNoConflict 改成自己原来的名（不变更）应通过
func TestHandleUpdateAgentCommand_SameNameNoConflict(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner")
	cmd := &model.AgentCommand{
		Name: "stable", Content: "echo old", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: u.ID, RunUser: "root", Workdir: "/root",
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// 名字不变、只改内容
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/update",
		map[string]any{"id": cmd.ID, "name": "stable", "content": "echo new", "timeout_sec": 60},
		"owner")
	rr := httptest.NewRecorder()
	HandleUpdateAgentCommand(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d %s, want 200 (改自己原名应允许)", rr.Code, rr.Body.String())
	}
}

func TestHandleDeleteAgentCommand_BlockedByInProgress(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner")
	cmd := &model.AgentCommand{
		Name: "ok", Content: "echo", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: u.ID,
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5); err != nil {
		t.Fatalf("seed: %v", err)
	}
	dID := seedDispatchRow(t, ctx, "task-blocker1", cmd.ID, u.ID, `{"name":"echo"}`)
	inv := &model.AgentCommandInvocation{
		DispatchID: dID, DispatchSlug: "task-blocker1",
		Status: model.AgentInvocationStatusInProgress, BatchIndex: 1,
	}
	if err := model.DB(ctx).Create(inv).Error; err != nil {
		t.Fatalf("seed inv: %v", err)
	}
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/delete",
		map[string]any{"id": cmd.ID}, "owner")
	rr := httptest.NewRecorder()
	HandleDeleteAgentCommand(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "command_in_use") {
		t.Errorf("body missing command_in_use: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "task-blocker1") {
		t.Errorf("body missing blocking dispatch_slug: %s", rr.Body.String())
	}
}

func TestHandleDeleteAgentCommand_OK(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "owner")
	cmd := &model.AgentCommand{
		Name: "ok", Content: "echo", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: u.ID,
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5); err != nil {
		t.Fatalf("seed: %v", err)
	}
	req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/delete",
		map[string]any{"id": cmd.ID}, "owner")
	rr := httptest.NewRecorder()
	HandleDeleteAgentCommand(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rr.Code, rr.Body.String())
	}
	if _, err := model.FindAgentCommandByID(ctx, cmd.ID); err == nil {
		t.Error("expected soft-delete to hide row from default scope")
	}
}

// ============================================================================
// Method 校验 & 未登录
// ============================================================================

func TestAgentCommandHandlers_MethodNotAllowed(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "admin1")
	cases := []struct {
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{http.MethodPost, "/admin/agent-commands", HandleListAgentCommands},
		{http.MethodGet, "/admin/agent-commands/create", HandleCreateAgentCommand},
		{http.MethodGet, "/admin/agent-commands/update", HandleUpdateAgentCommand},
		{http.MethodGet, "/admin/agent-commands/delete", HandleDeleteAgentCommand},
		{http.MethodGet, "/admin/agent-commands/dispatch", HandleDispatchAgentCommand},
		{http.MethodPost, "/admin/agent-commands/tasks", HandleListAgentCommandTasks},
		{http.MethodPost, "/admin/agent-commands/tasks/detail", HandleAgentCommandTaskDetail},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := adminSessionReq(t, c.method, c.path, nil, "admin1")
			rr := httptest.NewRecorder()
			c.handler(rr, req)
			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want 405; body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

// ============================================================================
// dispatch 校验链
// ============================================================================

func TestHandleDispatch_ValidationChain(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	u := makeAdminUser(t, ctx, "admin1")
	cmd := &model.AgentCommand{
		Name: "ok", Content: "echo", Type: "SHELL", TimeoutSec: 60,
		ParamsJSON: "[]", CreatedByUserID: u.ID, RunUser: "root", Workdir: "/root",
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, cmd, 5); err != nil {
		t.Fatalf("seed: %v", err)
	}

	t.Run("command_required", func(t *testing.T) {
		req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/dispatch",
			map[string]any{"instance_ids": []uint{1}}, "admin1")
		rr := httptest.NewRecorder()
		HandleDispatchAgentCommand(rr, req)
		if rr.Code != http.StatusBadRequest ||
			!strings.Contains(rr.Body.String(), "command_required") {
			t.Errorf("got %d %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("targets_required", func(t *testing.T) {
		req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/dispatch",
			map[string]any{"command_id": cmd.ID, "instance_ids": []uint{}}, "admin1")
		rr := httptest.NewRecorder()
		HandleDispatchAgentCommand(rr, req)
		if rr.Code != http.StatusBadRequest ||
			!strings.Contains(rr.Body.String(), "targets_required") {
			t.Errorf("got %d %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("too_many_targets", func(t *testing.T) {
		ids := make([]uint, model.AgentDispatchMaxTargets+1)
		for i := range ids {
			ids[i] = uint(i + 1)
		}
		req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/dispatch",
			map[string]any{"command_id": cmd.ID, "instance_ids": ids}, "admin1")
		rr := httptest.NewRecorder()
		HandleDispatchAgentCommand(rr, req)
		if rr.Code != http.StatusBadRequest ||
			!strings.Contains(rr.Body.String(), "too_many_targets") {
			t.Errorf("got %d %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("instance_not_found", func(t *testing.T) {
		req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/dispatch",
			map[string]any{"command_id": cmd.ID, "instance_ids": []uint{999999}},
			"admin1")
		rr := httptest.NewRecorder()
		HandleDispatchAgentCommand(rr, req)
		if rr.Code != http.StatusNotFound ||
			!strings.Contains(rr.Body.String(), "instance_not_found") {
			t.Errorf("got %d %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("test_target_required", func(t *testing.T) {
		ins := &model.Instance{
			Name: "ins-a", InstanceId: "ins-aaa",
			LastCVMState: "RUNNING", UserID: u.ID,
		}
		if err := model.DB(ctx).Create(ins).Error; err != nil {
			t.Fatalf("seed instance: %v", err)
		}
		req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/dispatch",
			map[string]any{
				"command_id":   cmd.ID,
				"instance_ids": []uint{ins.ID},
				"test_first":   true,
			}, "admin1")
		rr := httptest.NewRecorder()
		HandleDispatchAgentCommand(rr, req)
		if rr.Code != http.StatusBadRequest ||
			!strings.Contains(rr.Body.String(), "test_target_required") {
			t.Errorf("got %d %s", rr.Code, rr.Body.String())
		}
		// cleanup
		_ = model.DB(ctx).Unscoped().Delete(ins).Error
	})

	t.Run("local_instance_target_unsupported", func(t *testing.T) {
		// 本地实例（source=local）不能被 dispatch，未走 TAT 之前应返 400。
		local := &model.Instance{
			Name: "local-bad", InstanceId: "local-workbuddy-deadbe",
			LastCVMState: "RUNNING", UserID: u.ID,
			Source: model.InstanceSourceLocal,
		}
		if err := model.DB(ctx).Create(local).Error; err != nil {
			t.Fatalf("seed local instance: %v", err)
		}
		defer func() { _ = model.DB(ctx).Unscoped().Delete(local).Error }()

		req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/dispatch",
			map[string]any{
				"command_id":   cmd.ID,
				"instance_ids": []uint{local.ID},
			}, "admin1")
		rr := httptest.NewRecorder()
		HandleDispatchAgentCommand(rr, req)
		if rr.Code != http.StatusBadRequest ||
			!strings.Contains(rr.Body.String(), "local_instance_target_unsupported") {
			t.Errorf("got %d %s", rr.Code, rr.Body.String())
		}
		// 错误体应包含实例名，让管理员知道哪几个是本地
		if !strings.Contains(rr.Body.String(), "local-bad") {
			t.Errorf("错误响应应包含本地实例名 local-bad：%s", rr.Body.String())
		}
	})

	t.Run("local_instance_target_unsupported_mixed", func(t *testing.T) {
		// 本地+CVM 混批：只要含 1 个本地就拒，错误体仅列本地名。
		cvmIns := &model.Instance{
			Name: "cvm-ok", InstanceId: "ins-cvm-ok",
			LastCVMState: "RUNNING", UserID: u.ID,
			Source: model.InstanceSourceCVM,
		}
		if err := model.DB(ctx).Create(cvmIns).Error; err != nil {
			t.Fatalf("seed cvm instance: %v", err)
		}
		localIns := &model.Instance{
			Name: "local-mixed", InstanceId: "local-workbuddy-cafe01",
			LastCVMState: "RUNNING", UserID: u.ID,
			Source: model.InstanceSourceLocal,
		}
		if err := model.DB(ctx).Create(localIns).Error; err != nil {
			t.Fatalf("seed local instance: %v", err)
		}
		defer func() {
			_ = model.DB(ctx).Unscoped().Delete(cvmIns).Error
			_ = model.DB(ctx).Unscoped().Delete(localIns).Error
		}()

		req := adminSessionReq(t, http.MethodPost, "/admin/agent-commands/dispatch",
			map[string]any{
				"command_id":   cmd.ID,
				"instance_ids": []uint{cvmIns.ID, localIns.ID},
			}, "admin1")
		rr := httptest.NewRecorder()
		HandleDispatchAgentCommand(rr, req)
		if rr.Code != http.StatusBadRequest ||
			!strings.Contains(rr.Body.String(), "local_instance_target_unsupported") {
			t.Errorf("got %d %s", rr.Code, rr.Body.String())
		}
		body := rr.Body.String()
		if !strings.Contains(body, "local-mixed") {
			t.Errorf("错误体应列出本地实例 local-mixed：%s", body)
		}
		if strings.Contains(body, "cvm-ok") {
			t.Errorf("错误体不应列出 CVM 实例 cvm-ok：%s", body)
		}
	})
}

// ============================================================================
// 参数组装
// ============================================================================

func TestAssembleParamValues(t *testing.T) {
	cmd := &model.AgentCommand{}
	_ = cmd.SetParams([]model.AgentCommandParam{
		{Name: "name", Default: "world"},
		{Name: "days", Default: ""}, // 必填
	})

	t.Run("user value overrides default", func(t *testing.T) {
		got, err := assembleParamValues(cmd, map[string]string{"name": "alice", "days": "7"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got["name"] != "alice" || got["days"] != "7" {
			t.Errorf("got=%v", got)
		}
	})

	t.Run("default fallback", func(t *testing.T) {
		got, err := assembleParamValues(cmd, map[string]string{"days": "14"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got["name"] != "world" {
			t.Errorf("name should fallback to default, got=%q", got["name"])
		}
	})

	t.Run("missing required no default", func(t *testing.T) {
		if _, err := assembleParamValues(cmd, map[string]string{"name": "alice"}); err == nil ||
			!strings.Contains(err.Error(), "param_value_required") {
			t.Errorf("expected param_value_required, got %v", err)
		}
	})

	t.Run("explicit empty string passed through (not fallback to default)", func(t *testing.T) {
		// name 有 default="world"；用户显式传 "" 时应保留空字符串，不回退 default。
		// 这是产品上「在表单清空字段」和「没动字段」的语义区分。
		got, err := assembleParamValues(cmd, map[string]string{"name": "", "days": "1"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got["name"] != "" {
			t.Errorf("name=%q, want \"\" (not fallback to default)", got["name"])
		}
		if got["days"] != "1" {
			t.Errorf("days=%q, want \"1\"", got["days"])
		}
	})

	t.Run("explicit empty string for required-no-default also accepted", func(t *testing.T) {
		// days 是必填无 default；用户显式传 "" 仍应被透传（key 在 map 里就算"传了"）。
		got, err := assembleParamValues(cmd, map[string]string{"name": "x", "days": ""})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got["days"] != "" {
			t.Errorf("days=%q, want \"\" (explicit empty should be allowed)", got["days"])
		}
	})

	t.Run("unknown param", func(t *testing.T) {
		if _, err := assembleParamValues(cmd, map[string]string{"name": "x", "days": "1", "extra": "y"}); err == nil ||
			!strings.Contains(err.Error(), "param_unknown") {
			t.Errorf("expected param_unknown, got %v", err)
		}
	})
}

// ============================================================================
// 工具函数
// ============================================================================

func TestChunkUint(t *testing.T) {
	got := chunkUint([]uint{1, 2, 3, 4, 5}, 2)
	if len(got) != 3 || len(got[0]) != 2 || len(got[2]) != 1 {
		t.Errorf("unexpected chunks: %v", got)
	}
	if r := chunkUint(nil, 2); r != nil {
		t.Error("nil input should yield nil")
	}
	if r := chunkUint([]uint{1}, 0); r != nil {
		t.Error("size 0 should yield nil")
	}
}

func TestDedupUintSlice(t *testing.T) {
	got := dedupUintSlice([]uint{1, 2, 2, 3, 0, 1})
	if len(got) != 3 {
		t.Errorf("want 3 unique, got %v", got)
	}
}

func TestPreviewContent(t *testing.T) {
	if previewContent("hello", 10) != "hello" {
		t.Error("short string should pass through")
	}
	long := previewContent(strings.Repeat("a", 100), 10)
	if !strings.HasSuffix(long, "...") || len(long) != 13 {
		t.Errorf("preview = %q", long)
	}

	// 中文场景：n 是 rune 数而不是字节数；不能切到 UTF-8 字节中间。
	zh := "你好世界这是一段中文文本"
	got := previewContent(zh, 4)
	want := "你好世界..."
	if got != want {
		t.Errorf("Chinese preview = %q, want %q", got, want)
	}
	// 长度刚好等于 rune 数：原样返回，不加省略号。
	if got := previewContent(zh, utf8.RuneCountInString(zh)); got != zh {
		t.Errorf("equal-length preview = %q, want %q", got, zh)
	}
	// n=0 直接返回空。
	if got := previewContent(zh, 0); got != "" {
		t.Errorf("n=0 preview = %q, want empty", got)
	}
}

func TestTrimRunes_Chinese(t *testing.T) {
	if trimRunes("hi", 10) != "hi" {
		t.Error("short ascii should pass through")
	}
	if got := trimRunes("hello world", 5); got != "hello" {
		t.Errorf("ascii trim = %q, want hello", got)
	}
	// 中文按 rune 截断，不会出现半字节乱码。
	zh := "你好世界这是一段中文文本"
	got := trimRunes(zh, 4)
	if got != "你好世界" {
		t.Errorf("Chinese trim = %q, want 你好世界", got)
	}
	// 字节切片才会触发的 bug：rune 4 → 字节 12，原 s[:4] 会切到「你」字节中间。
	if !utf8.ValidString(got) {
		t.Errorf("trimRunes produced invalid UTF-8: %q", got)
	}
}

func TestAggregateDispatchStatus(t *testing.T) {
	if got := aggregateDispatchStatus(nil, true, 0, 0); got != model.AgentInvocationStatusInProgress {
		t.Errorf("any in_progress -> %s, want in_progress", got)
	}
	if got := aggregateDispatchStatus(nil, false, 5, 0); got != model.AgentInvocationStatusSuccess {
		t.Errorf("all success -> %s, want success", got)
	}
	if got := aggregateDispatchStatus(nil, false, 3, 2); got != model.AgentInvocationStatusPartial {
		t.Errorf("mixed -> %s, want partial", got)
	}
	if got := aggregateDispatchStatus(nil, false, 0, 5); got != model.AgentInvocationStatusFailed {
		t.Errorf("all fail -> %s, want failed", got)
	}
}

// TestAggregateDispatchStatus_AwaitingConfirmation 测试机已成功 + 生产 invocation 仍 pending → awaiting_confirmation。
//
// 即使有 anyInProgress=true（pending 也被计为非终态），awaiting_confirmation 优先级更高，
// 因为前端要先看到「测试结果 + 决定按钮」而不是「整个还在跑」的占位状态。
func TestAggregateDispatchStatus_AwaitingConfirmation(t *testing.T) {
	invs := []model.AgentCommandInvocation{
		{IsTestRun: true, Status: model.AgentInvocationStatusSuccess, TATInvocationID: "inv-test-ok"},
		{IsTestRun: false, Status: model.AgentInvocationStatusPending, TATInvocationID: ""},
	}
	if got := aggregateDispatchStatus(invs, true, 1, 0); got != model.AgentDispatchStatusAwaitingConfirmation {
		t.Errorf("test_success + prod_pending -> %s, want awaiting_confirmation", got)
	}

	// 反例：测试 invocation 还在 pending → 还不算 awaiting_confirmation
	invs2 := []model.AgentCommandInvocation{
		{IsTestRun: true, Status: model.AgentInvocationStatusPending},
		{IsTestRun: false, Status: model.AgentInvocationStatusPending},
	}
	if got := aggregateDispatchStatus(invs2, true, 0, 0); got != model.AgentInvocationStatusInProgress {
		t.Errorf("test_pending -> %s, want in_progress", got)
	}

	// 反例：prod invocation 已经拿到 tat_invocation_id（即续跑已经触发） → 不再 awaiting
	invs3 := []model.AgentCommandInvocation{
		{IsTestRun: true, Status: model.AgentInvocationStatusSuccess, TATInvocationID: "inv-test"},
		{IsTestRun: false, Status: model.AgentInvocationStatusInProgress, TATInvocationID: "inv-prod"},
	}
	if got := aggregateDispatchStatus(invs3, true, 1, 0); got != model.AgentInvocationStatusInProgress {
		t.Errorf("prod_running after continue -> %s, want in_progress", got)
	}
}

// TestAggregateDispatchStatus_Cancelled cancelled invocation 在全部终态后优先于 success/partial/failed。
func TestAggregateDispatchStatus_Cancelled(t *testing.T) {
	// test_success + prod_cancelled → cancelled（用户主动放弃，覆盖 success）
	invs := []model.AgentCommandInvocation{
		{IsTestRun: true, Status: model.AgentInvocationStatusSuccess},
		{IsTestRun: false, Status: model.AgentInvocationStatusCancelled},
	}
	if got := aggregateDispatchStatus(invs, false, 1, 0); got != model.AgentInvocationStatusCancelled {
		t.Errorf("test_success + prod_cancelled -> %s, want cancelled", got)
	}

	// 全部 cancelled
	invs2 := []model.AgentCommandInvocation{
		{IsTestRun: false, Status: model.AgentInvocationStatusCancelled},
	}
	if got := aggregateDispatchStatus(invs2, false, 0, 0); got != model.AgentInvocationStatusCancelled {
		t.Errorf("all_cancelled -> %s, want cancelled", got)
	}
}

func TestBuildCommandSnapshot_StoresRawContent(t *testing.T) {
	cmd := &model.AgentCommand{
		ID: 1, Slug: "cmd-rawtest", Name: "n", Type: "SHELL",
		Content:    "#!/bin/bash\necho 'hello {{name}}'\n# 中文",
		TimeoutSec: 60, RunUser: "root", Workdir: "/root", ParamsJSON: "[]",
	}
	b, err := buildCommandSnapshot(cmd, "task-x")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	got := decodeSnapshot(string(b))
	content, _ := got["content"].(string)
	if content != cmd.Content {
		t.Errorf("snapshot.content = %q, want raw original (no base64)", content)
	}
}

// ============================================================================
// dispatch B/C 模式：续跑 / 终止 / 入参冲突
// ============================================================================

// seedAwaitingDispatch 在 DB 中铺设一个「测试机已 success + prod 仍 pending」的 dispatch 局面。
// 返回 dispatch_slug + 测试 invocation id + 生产 invocation id（按顺序）。
func seedAwaitingDispatch(t *testing.T, ctx context.Context, owner *model.User) (slug string, testInvID, prodInvID uint) {
	t.Helper()
	slug = "task-await01"
	snap := `{"name":"echo","content":"echo hi","timeout_sec":5,"run_user":"root","workdir":"/root","params":[]}`
	dID := seedDispatchRow(t, ctx, slug, 1, owner.ID, snap, seedDispatchOpts{
		Status:      model.AgentDispatchStatusAwaitingConfirmation,
		TestFirst:   true,
		TargetCount: 3,
	})
	testInv := &model.AgentCommandInvocation{
		DispatchID:      dID,
		DispatchSlug:    slug,
		IsTestRun:       true,
		BatchIndex:      0,
		TargetCount:     1,
		SuccessCount:    1,
		Status:          model.AgentInvocationStatusSuccess,
		TATInvocationID: "inv-test-ok",
	}
	if err := model.DB(ctx).Create(testInv).Error; err != nil {
		t.Fatalf("create testInv: %v", err)
	}
	prodInv := &model.AgentCommandInvocation{
		DispatchID:   dID,
		DispatchSlug: slug,
		IsTestRun:    false,
		BatchIndex:   1,
		TargetCount:  2,
		Status:       model.AgentInvocationStatusPending,
	}
	if err := model.DB(ctx).Create(prodInv).Error; err != nil {
		t.Fatalf("create prodInv: %v", err)
	}
	prodTask1 := &model.AgentCommandTask{
		DispatchID: dID, InvocationID: prodInv.ID, DispatchSlug: slug,
		InstanceID: 11, CVMInstanceID: "ins-prod1", AgentName: "prod1",
		Status: model.AgentTaskStatusPending,
	}
	prodTask2 := &model.AgentCommandTask{
		DispatchID: dID, InvocationID: prodInv.ID, DispatchSlug: slug,
		InstanceID: 12, CVMInstanceID: "ins-prod2", AgentName: "prod2",
		Status: model.AgentTaskStatusPending,
	}
	if err := model.DB(ctx).Create(prodTask1).Error; err != nil {
		t.Fatalf("create prodTask1: %v", err)
	}
	if err := model.DB(ctx).Create(prodTask2).Error; err != nil {
		t.Fatalf("create prodTask2: %v", err)
	}
	return slug, testInv.ID, prodInv.ID
}

func TestHandleDispatch_BMode_RejectsExtraParams(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "owner")
	body := map[string]any{
		"dispatch_slug": "task-x9k2m4n7",
		"instance_ids":  []uint{1, 2}, // ← 多余字段
	}
	req := adminSessionReq(t, "POST", "/admin/agent-commands/dispatch", body, "owner")
	rr := httptest.NewRecorder()
	HandleDispatchAgentCommand(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "dispatch_slug_with_extra_params") {
		t.Errorf("body = %q, want dispatch_slug_with_extra_params", rr.Body.String())
	}
}

func TestHandleDispatch_AbortOnly_RequiresSlug(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "owner")
	req := adminSessionReq(t, "POST", "/admin/agent-commands/dispatch",
		map[string]any{"abort": true}, "owner")
	rr := httptest.NewRecorder()
	HandleDispatchAgentCommand(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "dispatch_slug_required") {
		t.Errorf("body = %q, want dispatch_slug_required", rr.Body.String())
	}
}

func TestHandleDispatchContinue_NotFound(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "owner")
	body := map[string]any{"dispatch_slug": "task-doesnotexist"}
	req := adminSessionReq(t, "POST", "/admin/agent-commands/dispatch", body, "owner")
	rr := httptest.NewRecorder()
	HandleDispatchAgentCommand(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "dispatch_not_found") {
		t.Errorf("body = %q, want dispatch_not_found", rr.Body.String())
	}
}

func TestHandleDispatchContinue_TestPhaseInProgress(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	owner := makeAdminUser(t, ctx, "owner")
	slug := "task-running"
	dID := seedDispatchRow(t, ctx, slug, 1, owner.ID, "{}", seedDispatchOpts{
		Status: model.AgentDispatchStatusInProgress, TestFirst: true, TargetCount: 1,
	})
	// 测试 invocation 还在 pending → 不是 awaiting_confirmation
	testInv := &model.AgentCommandInvocation{
		DispatchID: dID, DispatchSlug: slug,
		IsTestRun: true, BatchIndex: 0, TargetCount: 1,
		Status: model.AgentInvocationStatusPending,
	}
	if err := model.DB(ctx).Create(testInv).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	body := map[string]any{"dispatch_slug": slug}
	req := adminSessionReq(t, "POST", "/admin/agent-commands/dispatch", body, "owner")
	rr := httptest.NewRecorder()
	HandleDispatchAgentCommand(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "test_phase_in_progress") {
		t.Errorf("body = %q, want test_phase_in_progress", rr.Body.String())
	}
}

func TestHandleDispatchContinue_AlreadyCompleted(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	owner := makeAdminUser(t, ctx, "owner")
	slug := "task-done"
	dID := seedDispatchRow(t, ctx, slug, 1, owner.ID, "{}", seedDispatchOpts{
		Status: model.AgentDispatchStatusSuccess, TargetCount: 1, SuccessCount: 1,
	})
	inv := &model.AgentCommandInvocation{
		DispatchID: dID, DispatchSlug: slug,
		IsTestRun: false, BatchIndex: 0, TargetCount: 1, SuccessCount: 1,
		Status: model.AgentInvocationStatusSuccess,
	}
	if err := model.DB(ctx).Create(inv).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	body := map[string]any{"dispatch_slug": slug}
	req := adminSessionReq(t, "POST", "/admin/agent-commands/dispatch", body, "owner")
	rr := httptest.NewRecorder()
	HandleDispatchAgentCommand(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "already_completed") {
		t.Errorf("body = %q, want already_completed", rr.Body.String())
	}
}

func TestHandleDispatchContinue_PermissionDenied(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	owner := makeAdminUser(t, ctx, "owner") // initial admin (smallest id)
	other := makeAdminUser(t, ctx, "other")
	slug, _, _ := seedAwaitingDispatch(t, ctx, other) // dispatch 由 other 发起

	// owner 是 initial admin，应该能继续；切换 other 也能；用第三个用户应被拒
	stranger := makeAdminUser(t, ctx, "stranger")
	_ = owner
	_ = stranger

	body := map[string]any{"dispatch_slug": slug}
	req := adminSessionReq(t, "POST", "/admin/agent-commands/dispatch", body, "stranger")
	rr := httptest.NewRecorder()
	HandleDispatchAgentCommand(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "permission_denied") {
		t.Errorf("body = %q, want permission_denied", rr.Body.String())
	}
}

func TestHandleDispatchAbort_HappyPath(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	owner := makeAdminUser(t, ctx, "owner")
	slug, _, prodInvID := seedAwaitingDispatch(t, ctx, owner)

	body := map[string]any{"dispatch_slug": slug, "abort": true}
	req := adminSessionReq(t, "POST", "/admin/agent-commands/dispatch", body, "owner")
	rr := httptest.NewRecorder()
	HandleDispatchAgentCommand(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["status"] != model.AgentInvocationStatusCancelled {
		t.Errorf("resp.status = %v, want cancelled", resp["status"])
	}
	// 至少 1 条 task 被取消
	if cnt, _ := resp["cancelled_count"].(float64); cnt < 1 {
		t.Errorf("resp.cancelled_count = %v, want >=1", resp["cancelled_count"])
	}

	// DB 校验
	var prodInv model.AgentCommandInvocation
	_ = model.DB(ctx).First(&prodInv, prodInvID).Error
	if prodInv.Status != model.AgentInvocationStatusCancelled {
		t.Errorf("prod invocation status = %s, want cancelled", prodInv.Status)
	}
	if prodInv.FinishedAt == nil {
		t.Error("prod invocation finished_at should be set on cancel")
	}
	var tasks []model.AgentCommandTask
	_ = model.DB(ctx).Where("dispatch_slug = ? AND is_test_target = ?", slug, false).Find(&tasks).Error
	for _, tk := range tasks {
		if tk.Status != model.AgentTaskStatusCancelled {
			t.Errorf("prod task status = %s, want cancelled", tk.Status)
		}
	}
}

func TestHandleDispatchAbort_NothingToAbort(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	ctx := context.Background()
	owner := makeAdminUser(t, ctx, "owner")
	// 一个全部 in_progress（含 prod 已发出）的 dispatch
	slug := "task-running2"
	dID := seedDispatchRow(t, ctx, slug, 1, owner.ID, "{}", seedDispatchOpts{
		Status: model.AgentDispatchStatusInProgress, TestFirst: true, TargetCount: 2, SuccessCount: 1,
	})
	testInv := &model.AgentCommandInvocation{
		DispatchID: dID, DispatchSlug: slug,
		IsTestRun: true, BatchIndex: 0, TargetCount: 1, SuccessCount: 1,
		Status: model.AgentInvocationStatusSuccess, TATInvocationID: "inv-t",
	}
	prodInv := &model.AgentCommandInvocation{
		DispatchID: dID, DispatchSlug: slug,
		IsTestRun: false, BatchIndex: 1, TargetCount: 1,
		Status: model.AgentInvocationStatusInProgress, TATInvocationID: "inv-p",
	}
	_ = model.DB(ctx).Create(testInv).Error
	_ = model.DB(ctx).Create(prodInv).Error

	body := map[string]any{"dispatch_slug": slug, "abort": true}
	req := adminSessionReq(t, "POST", "/admin/agent-commands/dispatch", body, "owner")
	rr := httptest.NewRecorder()
	HandleDispatchAgentCommand(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	// 已经 already_continued（prod 已发出），错误码应是这个；nothing_to_abort 是兜底
	body2 := rr.Body.String()
	if !strings.Contains(body2, "already_continued") && !strings.Contains(body2, "nothing_to_abort") {
		t.Errorf("body = %q, want already_continued or nothing_to_abort", body2)
	}
}
