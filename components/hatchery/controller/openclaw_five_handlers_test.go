package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// 测试辅助
// ---------------------------------------------------------------------------

// initFiveHandlersTestDB 初始化 5 个 handler 所需的内存数据库。
func initFiveHandlersTestDB(t *testing.T) func() {
	t.Helper()
	// createInstance 在事务中还会通过 model.DB 查询，需要多个连接共享同一内存库。
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{}, &model.Instance{}, &model.AIImage{}, &model.AIModel{},
		&model.InstanceAdjustment{},
		&model.SiteConfig{}, &model.AuditLog{}, &model.Notification{},
		&model.SkillInstallation{}, &model.SMHPersonalSpace{},
		&model.MemoryTDAIPlugin{}, &model.RuleSet{}, &model.Tag{},
		&model.GroupConfigBinding{}, &model.ResourcePolicy{}, &model.OpenClawRole{},
		&model.RoleVisibilityGroup{}, &model.UserGroup{}, &model.GroupClosure{},
	); err != nil {
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

// jsonReqWithSession 构造 application/json 请求。
func jsonReqWithSession(t *testing.T, method, path, username, body string) *http.Request {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

// ---------------------------------------------------------------------------
// HandleInstanceList
// ---------------------------------------------------------------------------

func TestHandleInstanceList_Unauthorized(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/list", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleInstanceList(rr, req)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleInstanceList_JSON_Empty(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := jsonReqWithSession(t, http.MethodGet, "/openclaw/list", "u1", "")
	rr := httptest.NewRecorder()

	HandleInstanceList(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "instances") {
		t.Errorf("响应应包含 instances 字段，实际=%s", rr.Body.String())
	}
}

func TestHandleInstanceList_JSON_WithInstances(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 创建一个普通实例 + 一个没有 agent_type 的旧实例（测试兼容处理）
	inst1 := &model.Instance{
		Name: "inst1", InstanceId: "", // 无 CVM id → 跳过 batchFetch
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw, AIModelID: 5,
	}
	model.DB(context.Background()).Create(inst1)
	inst2 := &model.Instance{
		Name: "inst2-legacy", InstanceId: "",
		UserID: user.ID, AgentType: "", // 空值应被兼容处理
	}
	model.DB(context.Background()).Create(inst2)

	// 创建一个"已销毁"实例以覆盖 CleanupDestroyedInstances 分支
	// InstanceId == "" + CurrentOperation == OpDelete → Step1 → StatusDestroyed
	inst3 := &model.Instance{
		Name: "inst3-destroyed", InstanceId: "",
		UserID:           user.ID,
		AgentType:        model.AgentTypeOpenClaw,
		CurrentOperation: model.OpDelete,
	}
	model.DB(context.Background()).Create(inst3)

	// 创建一个 AIModel，测试批量模型查询
	aim := &model.AIModel{
		Model:     gorm.Model{ID: 5},
		Provider:  "openai",
		ModelName: "gpt-4",
		ModelID:   "gpt-4",
	}
	model.DB(context.Background()).Create(aim)

	req := jsonReqWithSession(t, http.MethodGet, "/openclaw/list", "u1", "")
	rr := httptest.NewRecorder()

	HandleInstanceList(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "inst1") || !strings.Contains(body, "inst2-legacy") {
		t.Errorf("响应应包含两条实例，实际=%s", body)
	}
}

// ---------------------------------------------------------------------------
// HandleCurrentImage —— 已有 openclaw_test.go 覆盖，补一个非 GET 方法兼容测试
// ---------------------------------------------------------------------------

// 由于 HandleCurrentImage 已被 openclaw_test.go 覆盖完整分支，
// 这里仅补充一个 Logger(ctx) 路径的幂等性测试。
func TestHandleCurrentImage_RepeatedCall(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	img := model.AIImage{
		ImageId:      "img-repeat",
		ImageName:    "repeat",
		ImageType:    "PRIVATE_IMAGE",
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "1.0.0",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(&img)

	for i := 0; i < 2; i++ {
		req := jsonReqWithSession(t, http.MethodGet,
			"/openclaw/current-image?agent_type=openclaw", "u1", "")
		rr := httptest.NewRecorder()
		HandleCurrentImage(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("第 %d 次调用应 200，实际=%d", i, rr.Code)
		}
	}
}

// ---------------------------------------------------------------------------
// HandleDeleteInstance
// ---------------------------------------------------------------------------

func TestHandleDeleteInstance_MethodNotAllowed(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/delete", nil)
	rr := httptest.NewRecorder()

	HandleDeleteInstance(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应 405，实际=%d", rr.Code)
	}
}

func TestHandleDeleteInstance_Unauthorized(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/delete", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleDeleteInstance(rr, req)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleDeleteInstance_InstanceNotFound(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	form := url.Values{}
	form.Set("id", "9999")
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/delete", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleDeleteInstance(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDeleteInstance_CreatingForbidsDelete(t *testing.T) {
	// InstanceId 为空 + 无 CurrentOperation → ResolveInstanceStatus=creating
	// canOperate(OpDelete, creating) → 返回错误 → 409 Conflict
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "creating", InstanceId: "",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/delete", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleDeleteInstance(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("创建中应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "创建中") {
		t.Errorf("错误信息应含'创建中'，实际=%s", rr.Body.String())
	}
}

func TestHandleDeleteInstance_LocalCleanup_NoCVMId_JSON(t *testing.T) {
	// 前提：InstanceId 为空 + CurrentOperation=OpDelete
	// Step1(OpDelete + cvmInfo nil) → StatusDestroyed
	// canOperate(OpDelete) 允许（OpDelete 可覆盖）→ 走本地清理分支
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	now := time.Now()
	inst := &model.Instance{
		Name: "local-cleanup", InstanceId: "",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		CurrentOperation:          model.OpDelete,
		CurrentOperationUpdatedAt: &now,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/delete", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleDeleteInstance(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("本地清理应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "ok") {
		t.Errorf("响应应含 ok，实际=%s", rr.Body.String())
	}

	// 验证实例已删除
	var count int64
	model.DB(context.Background()).Model(&model.Instance{}).Where("id = ?", inst.ID).Count(&count)
	if count != 0 {
		t.Errorf("实例应已从 DB 删除，但仍存在")
	}
}

func TestHandleDeleteInstance_CreateFailed_LocalCleanup(t *testing.T) {
	// CurrentOperationState=OpStateFailed + CurrentOperation=OpCreate
	// Step0 → StatusLoadFailed（由于走 Step0，load_failed 不属于 creating/loading/pending）
	// canOperate(OpDelete, load_failed) → nil，允许删除；
	// 但 instance.InstanceId != "" → 不会走 "InstanceId == ''" 的本地清理分支。
	// 测试：InstanceId != "" + LastCVMState=PENDING（无 cvmInfo）→ StatusCreateFailed（Step2.2a2）
	// 但 fetchCVMInstanceInfo(非空) 会调 CVM → 在测试环境会失败。
	// 为避免外部依赖，采用另一种组合：InstanceId == "" → Step2.1 creating ×
	// 改用 InstanceId=="" + OpDelete 已经覆盖。这里改测试 InstanceId=="" 无 OpDelete 的情况
	// 不走 if 分支，而是常规 canOperate 拦截。该路径已由 Creating 测试覆盖。
	// 这里直接跳过，保留占位。
	t.Skip("create_failed 分支需要真实 CVM API，已由 NoCVMId 分支覆盖核心逻辑")
}

func TestHandleDeleteInstance_BannedUser(t *testing.T) {
	// 被封禁用户走 requireLogin 拦截
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u-banned", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	model.DB(context.Background()).Delete(user) // 软删除

	form := url.Values{}
	form.Set("id", "1")
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/delete", "u-banned", form.Encode())
	rr := httptest.NewRecorder()

	HandleDeleteInstance(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("封禁用户应 403，实际=%d", rr.Code)
	}
}

// 构造 StatusCreateFailed 路径：InstanceId 非空 + LastStable/LastCVMState 为空 → Step2.2b2
// 此时不要求 CurrentOperation==OpDelete，也不会走 "InstanceId == ”" 分支，
// 进入按 StatusCreateFailed 走本地清理的分支。
func TestHandleDeleteInstance_CreateFailedStatus_LocalCleanup_JSON(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "cf", InstanceId: "ins-cf",
		UserID:    user.ID,
		AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/delete", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleDeleteInstance(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("create_failed 本地清理应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var count int64
	model.DB(context.Background()).Model(&model.Instance{}).Where("id = ?", inst.ID).Count(&count)
	if count != 0 {
		t.Errorf("实例应已被删除，实际 count=%d", count)
	}
}

// 覆盖 "Pro 记忆库释放失败，中止删除" 分支（JSON）：
// 对非空 instance_id 存在 PRO plugin 带 pool_id → ReleaseProMemSpaceForMissingInstance 返回 false。
func TestHandleDeleteInstance_ProMemReleaseFail_JSON(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "pro-fail", InstanceId: "ins-pro-fail",
		UserID:    user.ID,
		AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// 带 PoolID 的 PRO plugin → 远端 SDK 初始化失败 → 释放失败
	plugin := &model.MemoryTDAIPlugin{
		InstanceID:   "ins-pro-fail",
		CurrentPlan:  model.MemoryPlanPro,
		PoolID:       "pool-xxx",
		DatabaseName: "db-xxx",
		Endpoint:     "http://example.invalid",
	}
	model.DB(context.Background()).Create(plugin)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/delete", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleDeleteInstance(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Pro 记忆库释放失败应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Pro 记忆库") {
		t.Errorf("错误信息应包含 'Pro 记忆库'，实际=%s", rr.Body.String())
	}

	// 实例不应被删除
	var count int64
	model.DB(context.Background()).Model(&model.Instance{}).Where("id = ?", inst.ID).Count(&count)
	if count != 1 {
		t.Errorf("释放失败时不应删除实例，实际 count=%d", count)
	}
}
