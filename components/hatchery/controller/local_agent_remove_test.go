package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// setupLocalAgentRemoveTestDB 内存 sqlite + 全量 model migrate + session store。
func setupLocalAgentRemoveTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(model.AllModelsForTest()...); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	// 开启本地 Agent 功能（SiteConfig.LocalAgentEnabled）+ 跨租户白名单，确保 ensureLocalAgentAllowed 通过。
	now := time.Now()
	db.Create(&model.SiteConfig{LocalAgentEnabled: true, SMHEnabled: 1, LastFullSyncFinishedAt: &now})
	if err := db.Create(&model.FeatureAllowlist{
		Type:       model.FeatureAllowlistTypeLocalAgent,
		Identifier: "test-tenant",
	}).Error; err != nil {
		t.Fatalf("create allowlist: %v", err)
	}
	// admin 用户：requireAdmin 走 session 时查 DB role=admin
	db.Create(&model.User{Username: "admin", Role: "admin", Identifier: "test-tenant"})
	AdminToken = "test-admin-token"
	if Store == nil {
		Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	}
	t.Cleanup(model.UseDBForTest(db))
}

// seedLocalUserAndInstance 创建普通用户 + 一个本地 agent 实例（source=local）。
func seedLocalUserAndInstance(t *testing.T, username, agentType string) (uint, uint) {
	t.Helper()
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "test-tenant"})
	user := model.User{Username: username, Role: "user", Identifier: "test-tenant"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	inst := model.Instance{
		Name:       "local-" + username,
		InstanceId: "local-" + username + "-001",
		UserID:     user.ID,
		Source:     model.InstanceSourceLocal,
		AgentType:  agentType,
	}
	if err := model.DB(ctx).Create(&inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}
	return user.ID, inst.ID
}

// loginCookie 用 username 登录拿到 session cookie。
func loginCookie(t *testing.T, username string) string {
	t.Helper()
	ensureSessionStore()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username
	rr := httptest.NewRecorder()
	if err := session.Save(req, rr); err != nil {
		t.Fatalf("save session: %v", err)
	}
	for _, c := range rr.Result().Cookies() {
		if c.Name == "hatchery-session" {
			return c.String()
		}
	}
	t.Fatal("no session cookie")
	return ""
}

// adminCookie 用 admin-token 登录拿到 cookie。
func adminCookie(t *testing.T) string {
	t.Helper()
	ensureSessionStore()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+AdminToken)
	session, _ := Store.Get(req, "hatchery-session")
	// requireAdmin 读 username；admin-token user 在 ctx 无具体 user，这里放一个 admin 用户名占位。
	session.Values["username"] = "admin"
	rr := httptest.NewRecorder()
	if err := session.Save(req, rr); err != nil {
		t.Fatalf("save session: %v", err)
	}
	for _, c := range rr.Result().Cookies() {
		if c.Name == "hatchery-session" {
			return c.String()
		}
	}
	t.Fatal("no session cookie")
	return ""
}

// seedLocalUser 仅创建用户（不创建实例），用于登录态校验场景。
func seedLocalUser(t *testing.T, username string) uint {
	t.Helper()
	ctx := context.Background()
	user := model.User{Username: username, Role: "user", Identifier: "test-tenant"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user.ID
}

// seedLocalUserAndInstanceWithCID 创建用户 + 本地 agent 实例（指定 InstanceId）。
func seedLocalUserAndInstanceWithCID(t *testing.T, username, agentType, instanceCID string) (*model.User, *model.Instance) {
	t.Helper()
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "test-tenant"})
	user := model.User{Username: username, Role: "user", Identifier: "test-tenant"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	inst := model.Instance{
		Identifier: "test-tenant",
		Name:       "local-" + username,
		InstanceId: instanceCID,
		UserID:     user.ID,
		Source:     model.InstanceSourceLocal,
		AgentType:  agentType,
	}
	if err := model.DB(ctx).Create(&inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}
	return &user, &inst
}

func doRemoveReq(t *testing.T, path, cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	rr := httptest.NewRecorder()
	if path == "/admin/local-agent/remove" {
		req.Header.Set("Authorization", "Bearer "+AdminToken)
	}
	// 直接调 handler（绕过 main 路由，避免 WithOpenAPI 依赖注入遗漏）
	if path == "/local-agent/remove" {
		HandleLocalAgentRemove(rr, req)
	} else {
		HandleAdminLocalAgentRemove(rr, req)
	}
	return rr
}

func TestHandleLocalAgentRemove_CreatesTask(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	_, instID := seedLocalUserAndInstance(t, "alice", "codebuddy")
	cookie := loginCookie(t, "alice")

	rr := doRemoveReq(t, "/local-agent/remove", cookie, `{"instance_id": `+strconv.Itoa(int(instID))+`}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		OK     bool   `json:"ok"`
		TaskID uint   `json:"task_id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.OK || resp.TaskID == 0 || resp.Status != model.LocalAgentTaskStatusPending {
		t.Fatalf("unexpected resp: %+v", resp)
	}

	// 校验落库：type=uninstall_teamai, cmd 拼装正确
	var task model.LocalAgentTask
	if err := model.DB(context.Background()).First(&task, resp.TaskID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if task.Type != model.LocalAgentTaskTypeUninstallTeamai {
		t.Fatalf("type=%s", task.Type)
	}
	if task.Cmd != "teamai uninstall --force --agent codebuddy" {
		t.Fatalf("cmd=%q", task.Cmd)
	}
	if task.InstanceID != instID {
		t.Fatalf("instance_id=%d want %d", task.InstanceID, instID)
	}
}

func TestHandleLocalAgentRemove_Idempotent(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	_, instID := seedLocalUserAndInstance(t, "alice", "codebuddy")
	cookie := loginCookie(t, "alice")

	rr1 := doRemoveReq(t, "/local-agent/remove", cookie, `{"instance_id": `+strconv.Itoa(int(instID))+`}`)
	var r1 struct {
		TaskID uint `json:"task_id"`
	}
	json.Unmarshal(rr1.Body.Bytes(), &r1)

	rr2 := doRemoveReq(t, "/local-agent/remove", cookie, `{"instance_id": `+strconv.Itoa(int(instID))+`}`)
	var r2 struct {
		TaskID uint `json:"task_id"`
	}
	json.Unmarshal(rr2.Body.Bytes(), &r2)

	if r1.TaskID != r2.TaskID {
		t.Fatalf("idempotent broken: %d vs %d", r1.TaskID, r2.TaskID)
	}
	// 只应有一条 pending 任务
	var cnt int64
	model.DB(context.Background()).Model(&model.LocalAgentTask{}).
		Where("instance_id = ? AND type = ? AND status = ?", instID, model.LocalAgentTaskTypeUninstallTeamai, model.LocalAgentTaskStatusPending).
		Count(&cnt)
	if cnt != 1 {
		t.Fatalf("pending task count=%d want 1", cnt)
	}
}

func TestHandleLocalAgentRemove_NotFoundForOtherUser(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	// bob 的实例，alice 尝试删
	seedLocalUser(t, "alice") // 确保 alice 用户存在（登录态可解析）
	_, bobInst := seedLocalUserAndInstance(t, "bob", "codebuddy")
	cookie := loginCookie(t, "alice")

	rr := doRemoveReq(t, "/local-agent/remove", cookie, `{"instance_id": `+itoa(bobInst)+`}`)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleLocalAgentRemove_MissingInstanceID(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	seedLocalUser(t, "alice")
	cookie := loginCookie(t, "alice")
	rr := doRemoveReq(t, "/local-agent/remove", cookie, `{}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAdminLocalAgentRemove_CreatesTask(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	_, instID := seedLocalUserAndInstance(t, "alice", "codex")
	cookie := adminCookie(t)

	rr := doRemoveReq(t, "/admin/local-agent/remove", cookie, `{"instance_id": `+strconv.Itoa(int(instID))+`}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		TaskID uint   `json:"task_id"`
		Status string `json:"status"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.TaskID == 0 || resp.Status != model.LocalAgentTaskStatusPending {
		t.Fatalf("unexpected resp: %+v", resp)
	}
	var task model.LocalAgentTask
	model.DB(context.Background()).First(&task, resp.TaskID)
	if task.Cmd != "teamai uninstall --force --agent codex" {
		t.Fatalf("cmd=%q", task.Cmd)
	}
}

func TestHandleAdminLocalAgentRemove_RejectsNonLocal(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	// 创建一个 CVM 实例（source=cvm）
	ctx := context.Background()
	user := model.User{Username: "carol", Role: "user"}
	model.DB(ctx).Create(&user)
	cvm := model.Instance{Name: "cvm-1", InstanceId: "cvm-001", UserID: user.ID, Source: model.InstanceSourceCVM, AgentType: "codebuddy"}
	model.DB(ctx).Create(&cvm)
	cookie := adminCookie(t)

	rr := doRemoveReq(t, "/admin/local-agent/remove", cookie, `{"instance_id": `+itoa(cvm.ID)+`}`)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404 body=%s", rr.Code, rr.Body.String())
	}
}
