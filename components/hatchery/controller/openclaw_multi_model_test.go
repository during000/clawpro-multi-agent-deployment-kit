package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// setupMultiModelTestDB 初始化多模型接口测试所需的内存 SQLite 数据库与 Session Store。
// 返回 cleanup 用于恢复全局 DB / Store。
func setupMultiModelTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.AIModel{},
		&model.InstanceModel{},
		&model.SiteConfig{},
		&model.Notification{},
		&model.ModelVisibilityGroup{},
		&model.UserGroup{},
		&model.GroupClosure{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	origDB := model.UseDBForTest(db)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	// 清空 Domain 确保 injectModelConfigToCVM 返回错误而不是真的调用 TAT
	origSnap := common.FixedSnapshot
	newSnap := &common.TenantSnapshot{Domain: ""}
	if origSnap != nil {
		newSnap.Identifier = origSnap.Identifier
		newSnap.Uin = origSnap.Uin
		newSnap.InternalSecret = origSnap.InternalSecret
	}
	common.FixedSnapshot = newSnap

	// Mock LoadScript，避免 RunScript 因 LoadScript 未初始化触发 nil panic
	origLoadScript := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("test: LoadScript stub — script %s not available", name)
	}

	// Mock createErrorNotification 为 no-op，避免 handler 内 `go createErrorNotification(...)`
	// 启动的 goroutine 在测试 cleanup 替换全局 DB 之后才被调度执行，触发
	// "no such table: instances/notifications" 之类的竞态错误。
	// 多模型测试只关心事务回滚等 DB 状态是否正确，通知写入不是断言对象。
	origNotif := createErrorNotification
	createErrorNotification = func(
		userID, instanceID uint,
		instanceName, notifyType, title string,
		err error,
		ctx context.Context,
	) {
		// no-op
	}

	// Mock syncHermesLLMToTDAI 为 no-op，避免 customModel/setModel 走 Hermes 分支时
	// `go syncHermesLLMToTDAI(...)` 启动的 goroutine 在测试 cleanup 替换全局 DB 之后
	// 才被调度，内部 LookupRuntimeUser 触发 "no such table: instance_models" 竞态。
	origSyncHermes := syncHermesLLMToTDAI
	syncHermesLLMToTDAI = func(ctx context.Context, instanceID string) {
		// no-op
	}

	cleanup := func() {
		syncHermesLLMToTDAI = origSyncHermes
		createErrorNotification = origNotif
		origDB()
		Store = origStore
		common.FixedSnapshot = origSnap
		LoadScript = origLoadScript
	}
	t.Cleanup(cleanup)
	return cleanup
}

// stubSyncScriptRunnerOK 将 syncScriptRunner（syncInstanceModelsToCVM 内部使用）替换为
// 始终成功的桩，使删除流程在 DB 提交后下发 TAT 不会失败、不触发补偿回滚，便于单测
// 专注验证删除 / 提升等 DB 状态。t.Cleanup 自动恢复。
func stubSyncScriptRunnerOK(t *testing.T) {
	t.Helper()
	orig := syncScriptRunner
	syncScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "", nil
	}
	t.Cleanup(func() { syncScriptRunner = orig })
}

// multiModelReqWithSession 构造带 Session cookie（以 username 登录）的请求。
func multiModelReqWithSession(t *testing.T, method, path, username, body string) *http.Request {
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
	if err := session.Save(req, rr); err != nil {
		t.Fatalf("保存 session 失败: %v", err)
	}
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	// 注入租户 snapshot 到上下文（模拟 IdentifierMiddleware 的行为）
	if common.FixedSnapshot != nil {
		ctx := common.InjectTenant(req.Context(), *common.FixedSnapshot)
		req = req.WithContext(ctx)
	}

	return req
}

// createMultiModelUserAndInstance 创建测试所需的 user + OpenClaw 实例。
func createMultiModelUserAndInstance(t *testing.T, username, instName string) (*model.User, *model.Instance) {
	t.Helper()
	user := &model.User{Username: username, Password: "t", Role: "user"}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	proxyToken := "sk-test-" + username
	inst := &model.Instance{
		Name:        instName,
		InstanceId:  "ins-" + instName,
		UserID:      user.ID,
		AgentType:   model.AgentTypeOpenClaw,
		RuntimeUser: "ubuntu",
		ProxyToken:  &proxyToken,
	}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	// 默认开放自定义模型：创建 hatchery/custom 占位记录（enabled+visible），
	// 使 IsCustomModelEnabled 为 true。自定义模型 add-model / set-model 校验需要此前提
	// （GroupID=0 时按站点级开关回退）。占位记录已存在则忽略，避免唯一约束冲突。
	var customCnt int64
	model.DB(context.Background()).Model(&model.AIModel{}).
		Where("provider = ? AND model_id = ?", model.BuiltinModelProvider, model.BuiltinModelID).
		Count(&customCnt)
	if customCnt == 0 {
		model.DB(context.Background()).Create(&model.AIModel{
			Provider: model.BuiltinModelProvider, ModelID: model.BuiltinModelID,
			Enabled: true, Visible: true, VisibilityType: "all",
		})
	}
	return user, inst
}

// ─── HandleInstanceModels ────────────────────────────────────────────────

func TestHandleInstanceModels(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "models-u", "models-inst")

	m1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-4-plus", ModelName: "GLM-4-Plus", ModelType: "openai-completions", Enabled: true, Visible: true}
	m2 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "qwen-max", ModelName: "Qwen-Max", ModelType: "openai-completions", Enabled: true, Visible: true}
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Create(&m2)

	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 10})
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: model.ModelRoleFallback, SortOrder: 5})

	path := fmt.Sprintf("/openclaw/instance-models?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodGet, path, user.Username, "")
	rr := httptest.NewRecorder()

	HandleInstanceModels(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望状态码 200, 实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("期望 ok=true, 实际=%v", resp["ok"])
	}

	models, ok := resp["models"].([]interface{})
	if !ok {
		t.Fatalf("响应缺少 models 字段, body=%s", rr.Body.String())
	}
	if len(models) != 2 {
		t.Fatalf("期望 2 条记录, 实际=%d", len(models))
	}

	// sort_order DESC → primary(10) 在前
	first, _ := models[0].(map[string]interface{})
	if first["role"] != "primary" {
		t.Errorf("第一个应为 primary, 实际=%v", first["role"])
	}
}

func TestHandleInstanceModels_Unauthorized(t *testing.T) {
	setupMultiModelTestDB(t)

	// 不设置 session → 未登录
	req := httptest.NewRequest(http.MethodGet, "/openclaw/instance-models?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleInstanceModels(rr, req)

	if rr.Code == http.StatusOK {
		t.Errorf("未登录请求不应返回 200, 实际=%d", rr.Code)
	}
}

// ─── HandleAddModel ──────────────────────────────────────────────────────

// TestHandleAddModel_Builtin_CreatesRecord 验证：
// Domain 未配置 → TAT 失败 → 返回 500，但 InstanceModel 记录在 Create 后因回滚被删除。
// 通过 errorRich 中的错误消息确认走到了 TAT 分支（说明 DB 校验、role 判断都已通过）。
func TestHandleAddModel_Builtin_TATFailureRollback(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "add-u", "add-inst")

	m1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-4-plus", ModelName: "GLM-4-Plus", ModelType: "openai-completions", Enabled: true, Visible: true}
	model.DB(context.Background()).Create(&m1)

	form := url.Values{}
	form.Set("ai_model_id", strconv.Itoa(int(m1.ID)))

	path := fmt.Sprintf("/openclaw/add-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode())
	rr := httptest.NewRecorder()

	handleAddModel(rr, req, testCVMFetcher)

	// TAT 未配置 → Handler 应返回 500 且带 "TAT 执行失败" 信息
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500, 实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "TAT") {
		t.Errorf("期望响应包含 'TAT', 实际=%s", rr.Body.String())
	}

	// 验证回滚：记录应被删除
	var count int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 0 {
		t.Errorf("TAT 失败后记录应被回滚, 剩余=%d", count)
	}
}

// TestHandleAddModel_Duplicate 验证重复绑定返回 409（在 TAT 调用之前的校验路径）。
func TestHandleAddModel_Duplicate(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "dup-u", "dup-inst")

	m1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-4-plus", Enabled: true, Visible: true}
	model.DB(context.Background()).Create(&m1)

	// 预先绑定 → 再次调用应冲突
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary})

	form := url.Values{}
	form.Set("ai_model_id", strconv.Itoa(int(m1.ID)))

	path := fmt.Sprintf("/openclaw/add-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode())
	rr := httptest.NewRecorder()

	handleAddModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusConflict {
		t.Errorf("重复绑定应返回 409, 实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddModel_MissingAIModelID(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "noid-u", "noid-inst")

	// 空 form
	path := fmt.Sprintf("/openclaw/add-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, "")
	rr := httptest.NewRecorder()

	handleAddModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少 ai_model_id 应返回 400, 实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddModel_MethodNotAllowed(t *testing.T) {
	setupMultiModelTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/add-model", nil)
	rr := httptest.NewRecorder()

	handleAddModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 请求应返回 405, 实际=%d", rr.Code)
	}
}

// TestHandleAddModel_Agent328_Blocked 验证 Agent 3.28.x 版本被拒绝。
func TestHandleAddModel_Agent328_Blocked(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "block-u", "block-inst")
	inst.AgentVersion = "3.28" // 被拦截的版本
	model.DB(context.Background()).Save(inst)

	m1 := model.AIImage{ImageId: "img-blocked", ImageName: "Blocked-M", AgentType: "openclaw", AgentVersion: "3.28", Enabled: true}
	model.DB(context.Background()).Create(&m1)
	// 同时创建 AIModel 记录（add-model 内置模式查询 ai_models 表）
	am1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "blocked-m", ModelName: "Blocked-M", ModelType: "openai-completions", Enabled: true, Visible: true}
	model.DB(context.Background()).Create(&am1)

	form := url.Values{}
	form.Set("ai_model_id", strconv.Itoa(int(am1.ID)))

	path := fmt.Sprintf("/openclaw/add-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode())
	rr := httptest.NewRecorder()

	handleAddModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusConflict {
		t.Errorf("Agent 3.28 应返回 409, 实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "3.28") || !strings.Contains(body, "不支持") {
		t.Errorf("错误信息应包含版本号和'不支持', 实际=%s", body)
	}
}

// TestHandleAddModel_Agent329_Allowed 验证 Agent 3.29+ 版本正常通过。
func TestHandleAddModel_Agent329_Allowed(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "ok-u", "ok-inst")
	inst.AgentVersion = "3.29" // 允许的版本
	model.DB(context.Background()).Save(inst)

	am1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "ok-m", ModelName: "OK-M", ModelType: "openai-completions", Enabled: true, Visible: true}
	model.DB(context.Background()).Create(&am1)

	form := url.Values{}
	form.Set("ai_model_id", strconv.Itoa(int(am1.ID)))

	path := fmt.Sprintf("/openclaw/add-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode())
	rr := httptest.NewRecorder()

	handleAddModel(rr, req, testCVMFetcher)

	// 不应该返回 409（版本拦截）；TAT 可能失败返回 500，但不应是版本问题
	if rr.Code == http.StatusConflict {
		t.Errorf("Agent 3.29 不应被版本拦截, 实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddModel_HermesAllowsFirstSingleModel(t *testing.T) {
	setupMultiModelTestDB(t)

	user, instance := createMultiModelUserAndInstance(t, "hermes-single-u", "hermes-single-inst")
	instance.AgentType = model.AgentTypeHermes
	if err := model.DB(context.Background()).Save(instance).Error; err != nil {
		t.Fatalf("save instance: %v", err)
	}
	aiModel := model.AIModel{
		Provider: model.BuiltinModelProvider,
		ModelID:  "hermes-single", ModelName: "Hermes Single",
		ModelType: "openai-completions", Enabled: true, Visible: true,
	}
	if err := model.DB(context.Background()).Create(&aiModel).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	form := url.Values{"ai_model_id": {strconv.Itoa(int(aiModel.ID))}}
	req := multiModelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/add-model?id=%d", instance.ID), user.Username, form.Encode())
	rr := httptest.NewRecorder()
	handleAddModel(rr, req, testCVMFetcher)

	if rr.Code == http.StatusConflict {
		t.Fatalf("first Hermes model must remain allowed, got 409: %s", rr.Body.String())
	}
}

func TestHandleAddModel_HermesRejectsFallback(t *testing.T) {
	setupMultiModelTestDB(t)

	user, instance := createMultiModelUserAndInstance(t, "hermes-fallback-u", "hermes-fallback-inst")
	instance.AgentType = model.AgentTypeHermes
	if err := model.DB(context.Background()).Save(instance).Error; err != nil {
		t.Fatalf("save instance: %v", err)
	}
	primary := model.AIModel{
		Provider: model.BuiltinModelProvider,
		ModelID:  "hermes-primary", ModelName: "Hermes Primary",
		ModelType: "openai-completions", Enabled: true, Visible: true,
	}
	fallback := model.AIModel{
		Provider: model.BuiltinModelProvider,
		ModelID:  "hermes-fallback", ModelName: "Hermes Fallback",
		ModelType: "openai-completions", Enabled: true, Visible: true,
	}
	if err := model.DB(context.Background()).Create(&primary).Error; err != nil {
		t.Fatalf("create primary: %v", err)
	}
	if err := model.DB(context.Background()).Create(&fallback).Error; err != nil {
		t.Fatalf("create fallback: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.InstanceModel{
		InstanceID: instance.ID,
		AIModelID:  primary.ID,
		Role:       model.ModelRolePrimary,
		SortOrder:  1,
	}).Error; err != nil {
		t.Fatalf("create primary binding: %v", err)
	}

	form := url.Values{"ai_model_id": {strconv.Itoa(int(fallback.ID))}}
	req := multiModelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/add-model?id=%d", instance.ID), user.Username, form.Encode())
	rr := httptest.NewRecorder()
	handleAddModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusConflict {
		t.Fatalf("Hermes fallback status = %d, want 409; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "单模型") {
		t.Fatalf("fallback rejection should explain single-model capability: %s", rr.Body.String())
	}
}

// ─── HandleDelModel ──────────────────────────────────────────────────────

// TestHandleDelModel_PrimaryAutoPromote 验证删除 primary 时剩余 fallback 被自动提升为 primary。
// 删除流程在 DB 提交后下发 TAT；TAT 失败会触发补偿回滚，因此本用例 stub TAT 成功，
// 以验证删除 + 自动提升的 DB 逻辑。
func TestHandleDelModel_PrimaryAutoPromote(t *testing.T) {
	setupMultiModelTestDB(t)
	stubSyncScriptRunnerOK(t)

	user, inst := createMultiModelUserAndInstance(t, "del-u", "del-inst")

	m1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-4-plus", Enabled: true, Visible: true}
	m2 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "qwen-max", Enabled: true, Visible: true}
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Create(&m2)

	imPrimary := model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 10}
	imFallback := model.InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: model.ModelRoleFallback, SortOrder: 5}
	model.DB(context.Background()).Create(&imPrimary)
	model.DB(context.Background()).Create(&imFallback)

	form := url.Values{}
	form.Set("instance_model_id", strconv.Itoa(int(imPrimary.ID)))

	path := fmt.Sprintf("/openclaw/del-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode())
	rr := httptest.NewRecorder()

	handleDelModel(rr, req, testCVMFetcher)

	// 验证：primary 记录已被删除（忽略返回状态码，因 TAT 未配置可能返回 500）
	var leftCount int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("id = ?", imPrimary.ID).Count(&leftCount)
	if leftCount != 0 {
		t.Errorf("原 primary 记录应被删除, 剩余=%d, status=%d, body=%s", leftCount, rr.Code, rr.Body.String())
	}

	// 验证：剩余 fallback 被提升为 primary
	var promoted model.InstanceModel
	if err := model.DB(context.Background()).First(&promoted, imFallback.ID).Error; err != nil {
		t.Fatalf("查询剩余记录失败: %v", err)
	}
	if promoted.Role != model.ModelRolePrimary {
		t.Errorf("剩余 fallback 应被提升为 primary, 实际=%s, status=%d, body=%s", promoted.Role, rr.Code, rr.Body.String())
	}
}

func TestHandleDelModel_MethodNotAllowed(t *testing.T) {
	setupMultiModelTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/del-model", nil)
	rr := httptest.NewRecorder()

	handleDelModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 请求应返回 405, 实际=%d", rr.Code)
	}
}

// ─── HandleSwitchPrimaryModel ────────────────────────────────────────────

// TestHandleSwitchPrimaryModel_TATFailedRollback 验证 TAT 失败时事务完整回滚（方案文档 §13.1 一致性要求）：
// 1) DB 内 role 恢复初始状态（primary/fallback 未交换）
// 2) instances.ai_model_id 保留原主模型 ID
// 3) 响应 500 + TAT 执行失败描述
// 测试环境 LoadScript stub 强制返回错误，触发 TAT 失败分支。
func TestHandleSwitchPrimaryModel_TATFailedRollback(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "switch-u", "switch-inst")

	m1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-4-plus", ModelName: "GLM-4-Plus", Enabled: true, Visible: true}
	m2 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "qwen-max", ModelName: "Qwen-Max", Enabled: true, Visible: true}
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Create(&m2)

	// 初始：m1=primary, m2=fallback，instance.ai_model_id=m1.ID
	imPrimary := model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 10}
	imFallback := model.InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: model.ModelRoleFallback, SortOrder: 5}
	model.DB(context.Background()).Create(&imPrimary)
	model.DB(context.Background()).Create(&imFallback)
	model.DB(context.Background()).Model(inst).Update("ai_model_id", m1.ID)

	form := url.Values{}
	form.Set("instance_model_id", strconv.Itoa(int(imFallback.ID)))

	path := fmt.Sprintf("/openclaw/switch-primary-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode())
	rr := httptest.NewRecorder()

	handleSwitchPrimaryModel(rr, req, testCVMFetcher)

	// 1. 响应 500 + TAT 执行失败
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500, 实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "TAT 执行失败") {
		t.Errorf("响应应包含 'TAT 执行失败', 实际=%s", rr.Body.String())
	}

	// 2. 验证 DB 角色已回滚到初始状态
	var afterPrimary model.InstanceModel
	if err := model.DB(context.Background()).First(&afterPrimary, imPrimary.ID).Error; err != nil {
		t.Fatalf("查询原 primary 失败: %v", err)
	}
	if afterPrimary.Role != model.ModelRolePrimary {
		t.Errorf("原 primary 应保持 primary（事务已回滚）, 实际=%s", afterPrimary.Role)
	}

	var afterFallback model.InstanceModel
	if err := model.DB(context.Background()).First(&afterFallback, imFallback.ID).Error; err != nil {
		t.Fatalf("查询原 fallback 失败: %v", err)
	}
	if afterFallback.Role != model.ModelRoleFallback {
		t.Errorf("原 fallback 应保持 fallback（事务已回滚）, 实际=%s", afterFallback.Role)
	}

	// 3. 验证 instances.ai_model_id 未被修改
	var reloadInst model.Instance
	model.DB(context.Background()).First(&reloadInst, inst.ID)
	if reloadInst.AIModelID != m1.ID {
		t.Errorf("instance.ai_model_id 应保持 %d（事务已回滚）, 实际=%d", m1.ID, reloadInst.AIModelID)
	}
}

// TestHandleSwitchPrimaryModel_AlreadyPrimary 目标已是 primary → 400
func TestHandleSwitchPrimaryModel_AlreadyPrimary(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "switch-already-u", "switch-already-inst")

	m1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-4-plus", Enabled: true, Visible: true}
	model.DB(context.Background()).Create(&m1)

	im := model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 10}
	model.DB(context.Background()).Create(&im)

	form := url.Values{}
	form.Set("instance_model_id", strconv.Itoa(int(im.ID)))

	path := fmt.Sprintf("/openclaw/switch-primary-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode())
	rr := httptest.NewRecorder()

	handleSwitchPrimaryModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("目标已是 primary 应返回 400, 实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "无法切换到自身") {
		t.Errorf("响应应包含 '无法切换到自身', 实际=%s", rr.Body.String())
	}
}

// TestHandleSwitchPrimaryModel_NotFound instance_model_id 不属于该实例 → 400
func TestHandleSwitchPrimaryModel_NotFound(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "switch-nf-u", "switch-nf-inst")

	form := url.Values{}
	form.Set("instance_model_id", "99999")

	path := fmt.Sprintf("/openclaw/switch-primary-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode())
	rr := httptest.NewRecorder()

	handleSwitchPrimaryModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("绑定不存在应返回 400, 实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleSwitchPrimaryModel_MissingParam 缺少 instance_model_id → 400
func TestHandleSwitchPrimaryModel_MissingParam(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "switch-mp-u", "switch-mp-inst")

	path := fmt.Sprintf("/openclaw/switch-primary-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, "")
	rr := httptest.NewRecorder()

	handleSwitchPrimaryModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少参数应返回 400, 实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleSwitchPrimaryModel_MethodNotAllowed GET → 405
func TestHandleSwitchPrimaryModel_MethodNotAllowed(t *testing.T) {
	setupMultiModelTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/switch-primary-model", nil)
	rr := httptest.NewRecorder()

	handleSwitchPrimaryModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 请求应返回 405, 实际=%d", rr.Code)
	}
}

// ─── HandleVersionInfo ───────────────────────────────────────────────────

// TestHandleVersionInfo_TATFailure 由于 setupMultiModelTestDB 将 LoadScript mock 为总是返回错误，
// RunScript 会失败 → Handler 返回 500 且响应含 "TAT 执行失败"。
func TestHandleVersionInfo_TATFailure(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "ver-u", "ver-inst")

	path := fmt.Sprintf("/openclaw/version?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodGet, path, user.Username, "")
	rr := httptest.NewRecorder()

	HandleVersionInfo(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("TAT 失败应返回 500, 实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "TAT") {
		t.Errorf("响应应包含 'TAT', 实际=%s", rr.Body.String())
	}
}

// TestHandleVersionInfo_Unauthorized 未登录 → 非 200
func TestHandleVersionInfo_Unauthorized(t *testing.T) {
	setupMultiModelTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/version?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleVersionInfo(rr, req)

	if rr.Code == http.StatusOK {
		t.Errorf("未登录请求不应返回 200, 实际=%d", rr.Code)
	}
}

// TestHandleVersionInfo_InvalidInstance 非本人实例 / 不存在的 id → 400
func TestHandleVersionInfo_InvalidInstance(t *testing.T) {
	setupMultiModelTestDB(t)

	user, _ := createMultiModelUserAndInstance(t, "ver-inv-u", "ver-inv-inst")

	path := "/openclaw/version?id=99999"
	req := multiModelReqWithSession(t, http.MethodGet, path, user.Username, "")
	rr := httptest.NewRecorder()

	HandleVersionInfo(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 id 应返回 400, 实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

// ─── HandleDelModel 场景 A：删除 fallback 模型 ───────────────────────────────

// TestHandleDelModel_DeleteFallback 验证删除 fallback 时 primary 不变。
func TestHandleDelModel_DeleteFallback(t *testing.T) {
	setupMultiModelTestDB(t)
	stubSyncScriptRunnerOK(t)

	user, inst := createMultiModelUserAndInstance(t, "del-fb-u", "del-fb-inst")

	m1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-4-plus", Enabled: true, Visible: true}
	m2 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "qwen-max", Enabled: true, Visible: true}
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Create(&m2)

	imPrimary := model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 10}
	imFallback := model.InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: model.ModelRoleFallback, SortOrder: 5}
	model.DB(context.Background()).Create(&imPrimary)
	model.DB(context.Background()).Create(&imFallback)

	form := url.Values{}
	form.Set("instance_model_id", strconv.Itoa(int(imFallback.ID)))

	path := fmt.Sprintf("/openclaw/del-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode())
	rr := httptest.NewRecorder()

	handleDelModel(rr, req, testCVMFetcher)

	// fallback 应被删除
	var deletedCount int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("id = ?", imFallback.ID).Count(&deletedCount)
	if deletedCount != 0 {
		t.Errorf("fallback 记录应被删除，实际 count=%d, status=%d, body=%s", deletedCount, rr.Code, rr.Body.String())
	}

	// primary 应保持不变
	var primaryModel model.InstanceModel
	if err := model.DB(context.Background()).First(&primaryModel, imPrimary.ID).Error; err != nil {
		t.Fatalf("primary 记录不应被删除: %v", err)
	}
	if primaryModel.Role != model.ModelRolePrimary {
		t.Errorf("primary 角色不应变化，实际=%s", primaryModel.Role)
	}
}

// ─── HandleDelModel 场景 C：删除最后一个模型 ─────────────────────────────────

// TestHandleDelModel_DeleteLastModel 验证删除唯一模型后 ai_model_id 被置 0。
func TestHandleDelModel_DeleteLastModel(t *testing.T) {
	setupMultiModelTestDB(t)
	stubSyncScriptRunnerOK(t)

	user, inst := createMultiModelUserAndInstance(t, "del-last-u", "del-last-inst")

	m1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-4-plus", Enabled: true, Visible: true}
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Model(inst).Update("ai_model_id", m1.ID)

	imOnly := model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 1}
	model.DB(context.Background()).Create(&imOnly)

	form := url.Values{}
	form.Set("instance_model_id", strconv.Itoa(int(imOnly.ID)))

	path := fmt.Sprintf("/openclaw/del-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode())
	rr := httptest.NewRecorder()

	handleDelModel(rr, req, testCVMFetcher)

	// 记录应被删除
	var count int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 0 {
		t.Errorf("最后一条记录应被删除，实际 count=%d, status=%d, body=%s", count, rr.Code, rr.Body.String())
	}

	// ai_model_id 应被置 0
	var updatedInst model.Instance
	model.DB(context.Background()).First(&updatedInst, inst.ID)
	if updatedInst.AIModelID != 0 {
		t.Errorf("删除最后一个模型后 ai_model_id 应为 0，实际=%d", updatedInst.AIModelID)
	}
}

// ─── HandleInstanceModels 含自定义模型 ───────────────────────────────────────

// TestHandleInstanceModels_WithCustomModel 验证自定义模型绑定的列表查询。
func TestHandleInstanceModels_WithCustomModel(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "custom-list-u", "custom-list-inst")

	// 绑定一条自定义模型
	model.DB(context.Background()).Create(&model.InstanceModel{
		InstanceID:        inst.ID,
		AIModelID:         0,
		CustomModelID:     "my-custom",
		CustomModelConfig: `{"provider":"custom","model_id":"my-custom","model_name":"My Model","api_key":"sk","url":"https://x.com","model_type":"openai-completions","context_len":32000}`,
		Role:              model.ModelRolePrimary,
		SortOrder:         1,
	})

	path := fmt.Sprintf("/openclaw/instance-models?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodGet, path, user.Username, "")
	rr := httptest.NewRecorder()

	HandleInstanceModels(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	models, _ := resp["models"].([]interface{})
	if len(models) != 1 {
		t.Fatalf("期望 1 条记录，实际=%d", len(models))
	}
	first := models[0].(map[string]interface{})
	if first["is_custom"] != true {
		t.Errorf("自定义模型 is_custom 应为 true，实际=%v", first["is_custom"])
	}
	if first["model_id"] != "my-custom" {
		t.Errorf("model_id=%v, want my-custom", first["model_id"])
	}
}
