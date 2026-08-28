package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initEnvTestDB 初始化内存 SQLite 数据库用于 env handler 测试。
func initEnvTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	return func() {
		origDB()
		Store = origStore
	}
}

// envReqWithSession 构造带 session 的请求。
func envReqWithSession(t *testing.T, method, path, username, body string) *http.Request {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
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

// ─── HandleSetEnv ──────────────────────────────────────────────────────────

func TestHandleSetEnv_MethodNotAllowed(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/set-env", nil)
	rr := httptest.NewRecorder()
	handleSetEnv(rr, req, testCVMFetcher)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleSetEnv_Unauthorized(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	// 未登录
	req := httptest.NewRequest(http.MethodPost, "/openclaw/set-env", strings.NewReader(`{"id":1,"env":{"A":"B"}}`))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleSetEnv(rr, req, testCVMFetcher)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleSetEnv_InvalidJSON(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := envReqWithSession(t, http.MethodPost, "/openclaw/set-env", "u1", "not-a-json")
	rr := httptest.NewRecorder()
	handleSetEnv(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非 JSON 请求体应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetEnv_EmptyEnv(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := envReqWithSession(t, http.MethodPost, "/openclaw/set-env", "u1", `{"id":1,"env":{}}`)
	rr := httptest.NewRecorder()
	handleSetEnv(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("空 env 应返回 400，实际=%d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "env") {
		t.Errorf("错误信息应包含 env，实际=%s", rr.Body.String())
	}
}

func TestHandleSetEnv_TooManyKeys(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 构造 51 个 key
	var b strings.Builder
	b.WriteString(`{"id":1,"env":{`)
	for i := 0; i <= maxEnvKeys; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(`"KEY_%d":"v"`, i))
	}
	b.WriteString(`}}`)

	req := envReqWithSession(t, http.MethodPost, "/openclaw/set-env", "u1", b.String())
	rr := httptest.NewRecorder()
	handleSetEnv(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("超过上限应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleSetEnv_InvalidKeyName(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// key 含非法字符（以数字开头）
	req := envReqWithSession(t, http.MethodPost, "/openclaw/set-env", "u1", `{"id":1,"env":{"1BAD":"v"}}`)
	rr := httptest.NewRecorder()
	handleSetEnv(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 key 应返回 400，实际=%d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "环境变量名") {
		t.Errorf("错误消息应包含 '环境变量名'，实际=%s", rr.Body.String())
	}
}

func TestHandleSetEnv_InvalidValueType(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// value 是数字（非 string/null）
	req := envReqWithSession(t, http.MethodPost, "/openclaw/set-env", "u1", `{"id":1,"env":{"A":123}}`)
	rr := httptest.NewRecorder()
	handleSetEnv(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 value 类型应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleSetEnv_InstanceNotFound(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// id=999 不存在
	req := envReqWithSession(t, http.MethodPost, "/openclaw/set-env", "u1", `{"id":999,"env":{"A":"B"}}`)
	rr := httptest.NewRecorder()
	handleSetEnv(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetEnv_UnsupportedAgentType(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	// 未知 agent_type → ResolveScript 会失败
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-xxx", UserID: user.ID,
		AgentType: "totally_unknown_type",
	}
	model.DB(context.Background()).Create(inst)

	body := fmt.Sprintf(`{"id":%d,"env":{"A":"B"}}`, inst.ID)
	req := envReqWithSession(t, http.MethodPost, "/openclaw/set-env", "u1", body)
	rr := httptest.NewRecorder()
	handleSetEnv(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("未知 agent_type 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "set_env") {
		t.Errorf("错误应提到 set_env，实际=%s", rr.Body.String())
	}
}

func TestHandleSetEnv_ValueCanBeNull(t *testing.T) {
	// 验证 value 为 null（即删除环境变量）也能通过参数校验，
	// 随后因为没有配置 LoadScript（在此测试中）可能引发 RunScript 失败，
	// 但关键的参数校验分支已被覆盖。
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-envnull", UserID: user.ID,
		AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// 注入 LoadScript 使 RunScript 返回 "脚本不存在" 错误（而非真正跑 TAT）
	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: script not found: %s", name)
	}
	defer func() { LoadScript = origLoader }()

	body := fmt.Sprintf(`{"id":%d,"env":{"FOO":null,"BAR":"baz"}}`, inst.ID)
	req := envReqWithSession(t, http.MethodPost, "/openclaw/set-env", "u1", body)
	rr := httptest.NewRecorder()
	handleSetEnv(rr, req, testCVMFetcher)

	// 参数校验通过后，进入 RunScript 走到 LoadScript mock 错误 → 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("mock LoadScript 失败应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── HandleGetEnv ──────────────────────────────────────────────────────────

func TestHandleGetEnv_MethodNotAllowed(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/env", nil)
	rr := httptest.NewRecorder()
	HandleGetEnv(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleGetEnv_Unauthorized(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/env?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleGetEnv(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleGetEnv_InvalidID(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// id=abc 非法
	req := envReqWithSession(t, http.MethodGet, "/openclaw/env?id=abc", "u1", "")
	rr := httptest.NewRecorder()
	HandleGetEnv(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 id 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleGetEnv_ZeroID(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// id=0 会被识别为 invalid
	req := envReqWithSession(t, http.MethodGet, "/openclaw/env?id=0", "u1", "")
	rr := httptest.NewRecorder()
	HandleGetEnv(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("id=0 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleGetEnv_MissingParams(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 既无 id 也无 instance_id
	req := envReqWithSession(t, http.MethodGet, "/openclaw/env", "u1", "")
	rr := httptest.NewRecorder()
	HandleGetEnv(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少参数应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleGetEnv_InstanceNotFound(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := envReqWithSession(t, http.MethodGet, "/openclaw/env?id=999", "u1", "")
	rr := httptest.NewRecorder()
	HandleGetEnv(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleGetEnv_UnsupportedAgentType(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-getenv-unk", UserID: user.ID,
		AgentType: "totally_unknown_type",
	}
	model.DB(context.Background()).Create(inst)

	req := envReqWithSession(t, http.MethodGet, fmt.Sprintf("/openclaw/env?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleGetEnv(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("未知 agent_type 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "get_env") {
		t.Errorf("错误应提到 get_env，实际=%s", rr.Body.String())
	}
}

func TestHandleGetEnv_EmptyInstanceID(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	// 实例存在但 InstanceId 为空
	inst := &model.Instance{
		Name: "inst", InstanceId: "", UserID: user.ID,
		AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := envReqWithSession(t, http.MethodGet, fmt.Sprintf("/openclaw/env?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleGetEnv(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("空 InstanceId 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "CVM") {
		t.Errorf("错误应提到 CVM，实际=%s", rr.Body.String())
	}
}

// ─── getInstanceForEnv 单独分支测试 ────────────────────────────────────────

func TestGetInstanceForEnv_ByInstanceID(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-by-str-id", UserID: user.ID,
		AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	rr := httptest.NewRecorder()
	var w http.ResponseWriter = rr
	got, err := getInstanceForEnv(context.Background(), &w, user, 0, "ins-by-str-id")
	if err != nil {
		t.Fatalf("按 instance_id 查询应成功，实际 err=%v", err)
	}
	if got == nil || got.InstanceId != "ins-by-str-id" {
		t.Errorf("应返回对应实例，实际=%+v", got)
	}
}

func TestGetInstanceForEnv_MissingBoth(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	rr := httptest.NewRecorder()
	var w http.ResponseWriter = rr
	_, err := getInstanceForEnv(context.Background(), &w, user, 0, "")
	if err == nil {
		t.Fatal("id 和 instance_id 都为空应返回错误")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "缺少参数") {
		t.Errorf("错误应含 '缺少参数'，实际=%v", err)
	}
}

func TestGetInstanceForEnv_AdminUserIDZero(t *testing.T) {
	cleanup := initEnvTestDB(t)
	defer cleanup()

	// user.ID==0 表示 admin 场景：不限制 user_id
	adminUser := &model.User{Username: "admin", Password: "x", Role: "admin"}
	// 不创建到 DB（模拟 token 登录场景），强制 ID=0
	adminUser.ID = 0

	// 创建 owner 为 1 的实例
	owner := &model.User{Username: "owner", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(owner)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-admin", UserID: owner.ID,
		AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	rr := httptest.NewRecorder()
	var w http.ResponseWriter = rr
	got, err := getInstanceForEnv(context.Background(), &w, adminUser, inst.ID, "")
	if err != nil {
		t.Fatalf("admin(user.ID=0) 按 id 查询应成功，实际 err=%v", err)
	}
	if got == nil || got.ID != inst.ID {
		t.Errorf("应返回 owner 的实例，实际=%+v", got)
	}
}
