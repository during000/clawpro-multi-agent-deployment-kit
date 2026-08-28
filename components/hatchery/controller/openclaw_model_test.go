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
	"sync/atomic"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initModelTestDB 初始化内存 SQLite 数据库用于模型配置测试
func initModelTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	// :memory: 数据库必须限制为单连接，否则并发或连接重建会触发 "no such table"。
	// 见 docs/testing.md「初始化测试用 sqlite 数据库步骤」。
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层 *sql.DB 失败: %v", err)
	}
	sqlDB.SetConnMaxIdleTime(0)
	sqlDB.SetConnMaxLifetime(0)
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.AIModel{},
		&model.SiteConfig{},
		&model.InstanceModel{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.ModelVisibilityGroup{},
		&model.GroupClosure{},
		&model.GroupConfigBinding{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	AdminToken = "test-admin-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	return func() {
		origDB()
		Store = origStore
	}
}

func modelReqWithSession(t *testing.T, method, path, username, body string) *http.Request {
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

// Note: TestHandleSetModel_UnsupportedAgentType 已删除（v7）：
// 矩阵放开 Hermes/ACE 的 SupportsModel 后，这些类型不再是"不支持模型配置"的类型。
// 单 agent_type 级别的未支持场景由 ResolveScript 的 fail-closed 覆盖（unknown 类型），
// 由 TestResolveScript 的 "unknown-agent" 用例验证。

func TestHandleSetModel_MethodNotAllowed(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/model", nil)
	rr := httptest.NewRecorder()

	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 请求应返回 405，实际=%d", rr.Code)
	}
}

// TestBuildImageModelRefs 覆盖 imageModel 推导规则的 5 个场景
func TestBuildImageModelRefs(t *testing.T) {
	t.Cleanup(initModelTestDB(t))

	ctx := t.Context()
	db := model.DB(ctx)

	// 准备 ai_models：1 个支持 image，1 个 text-only
	aimVL := model.AIModel{
		ModelID:    "qwen-vl",
		ModelName:  "Qwen VL",
		Provider:   "hatchery",
		ModelType:  "openai-compatible",
		URL:        "http://x",
		APIKey:     "k",
		ContextLen: 8000,
		Enabled:    true,
		Visible:    true,
		InputTypes: `["text","image"]`,
	}
	aimText := model.AIModel{
		ModelID:    "deepseek-chat",
		ModelName:  "Deepseek Chat",
		Provider:   "hatchery",
		ModelType:  "openai-compatible",
		URL:        "http://x",
		APIKey:     "k",
		ContextLen: 8000,
		Enabled:    true,
		Visible:    true,
		InputTypes: `["text"]`,
	}
	aimVL2 := model.AIModel{
		ModelID:    "glm-4v",
		ModelName:  "GLM-4V",
		Provider:   "hatchery",
		ModelType:  "openai-compatible",
		URL:        "http://x",
		APIKey:     "k",
		ContextLen: 8000,
		Enabled:    true,
		Visible:    true,
		InputTypes: `["text","image"]`,
	}
	if err := db.Create(&aimVL).Error; err != nil {
		t.Fatalf("create aimVL: %v", err)
	}
	if err := db.Create(&aimText).Error; err != nil {
		t.Fatalf("create aimText: %v", err)
	}
	if err := db.Create(&aimVL2).Error; err != nil {
		t.Fatalf("create aimVL2: %v", err)
	}

	refVL := "hatchery-qwen-vl/qwen-vl"
	refText := "hatchery-deepseek-chat/deepseek-chat"
	refVL2 := "hatchery-glm-4v/glm-4v"

	// 工具：清空 instance_models
	clearIM := func() {
		if err := db.Exec("DELETE FROM instance_models").Error; err != nil {
			t.Fatalf("clear instance_models: %v", err)
		}
	}

	// 场景 1: 主模型支持 image，候选 ≥ 2 → primary=主模型, fallbacks=其余
	t.Run("primary supports image with multiple candidates", func(t *testing.T) {
		clearIM()
		db.Create(&model.InstanceModel{InstanceID: 1, AIModelID: aimVL.ID, Role: model.ModelRolePrimary, SortOrder: 1})
		db.Create(&model.InstanceModel{InstanceID: 1, AIModelID: aimVL2.ID, Role: model.ModelRoleFallback, SortOrder: 2})
		db.Create(&model.InstanceModel{InstanceID: 1, AIModelID: aimText.ID, Role: model.ModelRoleFallback, SortOrder: 3})

		primary, fallbacks, err := buildImageModelRefs(ctx, 1, refVL)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if primary != refVL {
			t.Errorf("primary 应=%s, 实际=%s", refVL, primary)
		}
		if fallbacks != `["`+refVL2+`"]` {
			t.Errorf("fallbacks 应=[%q], 实际=%s", refVL2, fallbacks)
		}
	})

	// 场景 2: 主模型不支持 image，候选 ≥ 1 → primary=候选[0], fallbacks=其余
	t.Run("primary not supports image, fallback to first candidate", func(t *testing.T) {
		clearIM()
		db.Create(&model.InstanceModel{InstanceID: 1, AIModelID: aimText.ID, Role: model.ModelRolePrimary, SortOrder: 1})
		db.Create(&model.InstanceModel{InstanceID: 1, AIModelID: aimVL.ID, Role: model.ModelRoleFallback, SortOrder: 2})
		db.Create(&model.InstanceModel{InstanceID: 1, AIModelID: aimVL2.ID, Role: model.ModelRoleFallback, SortOrder: 3})

		primary, fallbacks, err := buildImageModelRefs(ctx, 1, refText)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if primary != refVL {
			t.Errorf("primary 应=%s, 实际=%s", refVL, primary)
		}
		if fallbacks != `["`+refVL2+`"]` {
			t.Errorf("fallbacks 应=[%q], 实际=%s", refVL2, fallbacks)
		}
	})

	// 场景 3: 候选为空 → primary="", fallbacks="[]"
	t.Run("no image candidate", func(t *testing.T) {
		clearIM()
		db.Create(&model.InstanceModel{InstanceID: 1, AIModelID: aimText.ID, Role: model.ModelRolePrimary, SortOrder: 1})

		primary, fallbacks, err := buildImageModelRefs(ctx, 1, refText)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if primary != "" {
			t.Errorf("primary 应为空, 实际=%s", primary)
		}
		if fallbacks != "[]" {
			t.Errorf("fallbacks 应=[], 实际=%s", fallbacks)
		}
	})

	// 场景 4: 自定义模型 InputTypes 含 image → 进候选
	t.Run("custom model with image input", func(t *testing.T) {
		clearIM()
		cfgJSON := `{"provider":"custom","model_id":"my-vl","model_name":"My VL","api_key":"k","url":"http://x","model_type":"openai-compatible","input_types":["text","image"],"context_len":8000}`
		db.Create(&model.InstanceModel{
			InstanceID:        1,
			AIModelID:         0,
			CustomModelID:     "my-vl",
			Role:              model.ModelRolePrimary,
			SortOrder:         1,
			CustomModelConfig: cfgJSON,
		})

		primary, _, err := buildImageModelRefs(ctx, 1, "custom-my-vl/my-vl")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if primary != "custom-my-vl/my-vl" {
			t.Errorf("primary 应=custom-my-vl/my-vl, 实际=%s", primary)
		}
	})

	// 场景 5: 自定义模型 CustomModelConfig JSON 异常 → 不进候选
	t.Run("custom model with invalid config", func(t *testing.T) {
		clearIM()
		// 一个支持 image 的内置模型 + 一个 JSON 损坏的自定义模型
		db.Create(&model.InstanceModel{InstanceID: 1, AIModelID: aimVL.ID, Role: model.ModelRolePrimary, SortOrder: 1})
		db.Create(&model.InstanceModel{
			InstanceID:        1,
			AIModelID:         0,
			CustomModelID:     "broken",
			Role:              model.ModelRoleFallback,
			SortOrder:         2,
			CustomModelConfig: `{not valid json`,
		})

		primary, fallbacks, err := buildImageModelRefs(ctx, 1, refVL)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if primary != refVL {
			t.Errorf("primary 应=%s, 实际=%s", refVL, primary)
		}
		// 损坏的自定义模型不应进入 fallbacks
		if fallbacks != "[]" {
			t.Errorf("fallbacks 应=[]（损坏的自定义模型不进候选）, 实际=%s", fallbacks)
		}
	})
}

func openclawModelConnectivityReq(t *testing.T, method, path, username, body string) *http.Request {
	t.Helper()
	req := modelReqWithSession(t, method, path, username, body)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func decodeOpenClawModelConnectivityResp(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, w.Body.String())
	}
	return resp
}

func TestHandleModelConnectivity_Unauthorized(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/models/connectivity", strings.NewReader(`{"url":"http://example.com","api_key":"sk","model_type":"openai-completions"}`))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleModelConnectivity(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("未登录应返回 401，实际=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleModelConnectivity_MethodNotAllowed(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	admin := model.User{Username: "model-connect-admin-get", Password: "x", Role: "admin"}
	model.DB(context.Background()).Create(&admin)

	req := openclawModelConnectivityReq(t, http.MethodGet, "/openclaw/models/connectivity", admin.Username, "")
	w := httptest.NewRecorder()

	HandleModelConnectivity(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET 应返回 405，实际=%d", w.Code)
	}
}

func TestHandleModelConnectivity_BadRequests(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	admin := model.User{Username: "model-connect-admin-bad", Password: "x", Role: "admin"}
	model.DB(context.Background()).Create(&admin)

	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "invalid id", path: "/openclaw/models/connectivity?id=abc"},
		{name: "model not found", path: "/openclaw/models/connectivity?id=99999"},
		{name: "bad json", body: `not json`},
		{name: "missing url", body: `{"api_key":"sk","model_type":"openai-completions"}`},
		{name: "missing api key", body: `{"url":"http://example.com","model_type":"openai-completions"}`},
		{name: "invalid model type", body: `{"url":"http://example.com","api_key":"sk","model_type":"bad-type"}`},
		{name: "invalid url", body: `{"url":"ftp://example.com","api_key":"sk","model_type":"openai-completions"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.path
			if path == "" {
				path = "/openclaw/models/connectivity"
			}
			req := openclawModelConnectivityReq(t, http.MethodPost, path, admin.Username, tt.body)
			w := httptest.NewRecorder()

			HandleModelConnectivity(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("应返回 400，实际=%d body=%s", w.Code, w.Body.String())
			}
			resp := decodeOpenClawModelConnectivityResp(t, w)
			if resp["error"] == "" {
				t.Fatalf("错误响应应包含 error 字段，实际=%v", resp)
			}
		})
	}
}

func TestHandleModelConnectivity_SavedModelVisibilityDenied(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := model.User{Username: "model-connect-denied", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	group := model.UserGroup{Name: "secret-model-group", Description: "secret"}
	model.DB(context.Background()).Create(&group)

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	aiModel := model.AIModel{
		Provider:       "openai",
		ModelID:        "secret-model",
		ModelName:      "Secret Model",
		URL:            server.URL,
		APIKey:         "sk-secret",
		ModelType:      "openai-completions",
		Enabled:        true,
		VisibilityType: model.VisibilityGroup,
	}
	model.DB(context.Background()).Create(&aiModel)
	model.DB(context.Background()).Create(&model.ModelVisibilityGroup{AIModelID: aiModel.ID, GroupID: group.ID})

	req := openclawModelConnectivityReq(t, http.MethodPost, fmt.Sprintf("/openclaw/models/connectivity?id=%d", aiModel.ID), user.Username, "")
	w := httptest.NewRecorder()

	HandleModelConnectivity(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("不可见模型应返回 403，实际=%d body=%s", w.Code, w.Body.String())
	}
	if called {
		t.Fatalf("模型不可见时不应探测上游")
	}
}

func TestHandleModelConnectivity_SavedModelVisibilityAllowed(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := model.User{Username: "model-connect-allowed", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	group := model.UserGroup{Name: "allowed-model-group", Description: "allowed"}
	model.DB(context.Background()).Create(&group)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserID: user.ID, UserGroupID: group.ID})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 已保存模型携带 ModelID，handleModelConnectivity 使用 chat 探活，
		// 命中 /chat/completions。
		if r.URL.Path != "/chat/completions" {
			t.Errorf("请求路径=%s，期望 /chat/completions", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-visible" {
			t.Errorf("Authorization=%q，期望 Bearer sk-visible", got)
			http.Error(w, "unexpected auth", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"id":"x","choices":[{"message":{"role":"assistant","content":""}}]}`))
	}))
	defer server.Close()

	aiModel := model.AIModel{
		Provider:       "openai",
		ModelID:        "visible-model",
		ModelName:      "Visible Model",
		URL:            server.URL,
		APIKey:         "sk-visible",
		ModelType:      "openai-completions",
		Enabled:        true,
		Visible:        true,
		VisibilityType: model.VisibilityGroup,
	}
	model.DB(context.Background()).Create(&aiModel)
	model.DB(context.Background()).Create(&model.ModelVisibilityGroup{AIModelID: aiModel.ID, GroupID: group.ID})

	req := openclawModelConnectivityReq(t, http.MethodPost, fmt.Sprintf("/openclaw/models/connectivity?id=%d", aiModel.ID), user.Username, "")
	w := httptest.NewRecorder()

	HandleModelConnectivity(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("可见模型应返回 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	resp := decodeOpenClawModelConnectivityResp(t, w)
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("ok 应为 true，实际=%v", resp)
	}
}

func TestHandleModelConnectivity_SavedModelDisabledDenied(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := model.User{Username: "model-connect-disabled", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	aiModel := model.AIModel{
		Provider:       "openai",
		ModelID:        "disabled-model",
		ModelName:      "Disabled Model",
		URL:            "http://example.com",
		APIKey:         "sk-disabled",
		ModelType:      "openai-completions",
		Enabled:        false,
		VisibilityType: model.VisibilityAll,
	}
	model.DB(context.Background()).Create(&aiModel)

	req := openclawModelConnectivityReq(t, http.MethodPost, fmt.Sprintf("/openclaw/models/connectivity?id=%d", aiModel.ID), user.Username, "")
	w := httptest.NewRecorder()

	HandleModelConnectivity(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("禁用模型应返回 403，实际=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleModelConnectivity_TemporaryCredentialsSuccess(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := model.User{Username: "model-connect-user-ok", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("请求路径=%s，期望 /chat/completions", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-ok" {
			t.Errorf("Authorization=%q，期望 Bearer sk-ok", got)
			http.Error(w, "unexpected auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","choices":[{"message":{"role":"assistant","content":""}}]}`))
	}))
	defer server.Close()

	body := fmt.Sprintf(`{"url":%q,"api_key":"sk-ok","model_type":"openai-completions","model":"deepseek-v4-flash"}`, server.URL)
	req := openclawModelConnectivityReq(t, http.MethodPost, "/openclaw/models/connectivity", user.Username, body)
	w := httptest.NewRecorder()

	HandleModelConnectivity(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	resp := decodeOpenClawModelConnectivityResp(t, w)
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("ok 应为 true，实际=%v", resp)
	}
	if _, ok := resp["latency_ms"].(float64); !ok {
		t.Fatalf("响应应包含 latency_ms，实际=%v", resp)
	}
}

func TestHandleModelConnectivity_AnthropicTemporaryCredentialsSuccess(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := model.User{Username: "model-connect-anthropic", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("请求路径=%s，期望 /v1/messages", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("x-api-key"); got != "sk-anthropic" {
			t.Errorf("x-api-key=%q，期望 sk-anthropic", got)
			http.Error(w, "unexpected api key", http.StatusUnauthorized)
			return
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Errorf("anthropic-version 不能为空")
			http.Error(w, "missing anthropic version", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"id":"x","content":[]}`))
	}))
	defer server.Close()

	body := fmt.Sprintf(`{"url":%q,"api_key":"sk-anthropic","model_type":"anthropic-messages","model":"deepseek-v4-flash"}`, server.URL)
	req := openclawModelConnectivityReq(t, http.MethodPost, "/openclaw/models/connectivity", user.Username, body)
	w := httptest.NewRecorder()

	HandleModelConnectivity(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	resp := decodeOpenClawModelConnectivityResp(t, w)
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("ok 应为 true，实际=%v", resp)
	}
}

// ─── setModel / deleteModel 补充覆盖（成功路径 + TAT 失败补偿回滚） ────────

// withDomainCtx 在已构造的请求上覆盖租户 snapshot 的 Domain，
// 使 setModel / customModel 通过 "服务地址未配置" 校验并下发 TAT。
func withDomainCtx(req *http.Request, domain string) *http.Request {
	snap := hcommon.TenantSnapshot{Domain: domain}
	if hcommon.FixedSnapshot != nil {
		snap.Identifier = hcommon.FixedSnapshot.Identifier
		snap.Uin = hcommon.FixedSnapshot.Uin
		snap.InternalSecret = hcommon.FixedSnapshot.InternalSecret
	}
	return req.WithContext(hcommon.InjectTenant(req.Context(), snap))
}

// stubSyncScriptRunnerFail 将 syncScriptRunner 替换为始终失败的桩，
// 触发 deleteModel 在 DB 提交后下发 TAT 失败 → 补偿回滚分支。
func stubSyncScriptRunnerFail(t *testing.T) {
	t.Helper()
	orig := syncScriptRunner
	syncScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATSyncFailedFor, scriptName)
	}
	t.Cleanup(func() { syncScriptRunner = orig })
}

// stubAgentScriptRunnerOK 将 agentScriptRunner（RunAgentScript 内部使用）替换为
// 始终成功的桩，使 setModel / customModel 走 TAT 成功分支。
func stubAgentScriptRunnerOK(t *testing.T) {
	t.Helper()
	orig := agentScriptRunner
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return `{"ok":true}`, nil
	}
	t.Cleanup(func() { agentScriptRunner = orig })
}

// stubAgentScriptRunnerFail 将 agentScriptRunner 替换为始终失败的桩，
// 触发 setModel / customModel 在 DB 提交后下发 TAT 失败 → 补偿回滚分支。
func stubAgentScriptRunnerFail(t *testing.T) {
	t.Helper()
	orig := agentScriptRunner
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATFailed)
	}
	t.Cleanup(func() { agentScriptRunner = orig })
}

// ─── deleteModel: TAT 同步失败 → 补偿回滚 ─────────────────────────────────

// TestDeleteModel_SyncFailRollback_PrimaryPromote 验证删除 primary 后 TAT 同步失败时：
// 1) 返回 500 + "TAT 执行失败"
// 2) 被删 primary 记录被重建
// 3) 被自动提升的 fallback 回滚为 fallback
// 4) instances.ai_model_id 恢复为原 primary 的模型 ID
func TestDeleteModel_SyncFailRollback_PrimaryPromote(t *testing.T) {
	setupMultiModelTestDB(t)
	stubSyncScriptRunnerFail(t)

	user, inst := createMultiModelUserAndInstance(t, "del-rb-u", "del-rb-inst")

	m1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-4-plus", Enabled: true}
	m2 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "qwen-max", Enabled: true}
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Create(&m2)

	imPrimary := model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 10}
	imFallback := model.InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: model.ModelRoleFallback, SortOrder: 5}
	model.DB(context.Background()).Create(&imPrimary)
	model.DB(context.Background()).Create(&imFallback)
	model.DB(context.Background()).Model(inst).Update("ai_model_id", m1.ID)

	form := url.Values{}
	form.Set("instance_model_id", strconv.Itoa(int(imPrimary.ID)))

	path := fmt.Sprintf("/openclaw/del-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode())
	rr := httptest.NewRecorder()

	handleDelModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("TAT 同步失败应返回 500, 实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 1) 被删 primary 记录已被重建
	var restored model.InstanceModel
	if err := model.DB(context.Background()).First(&restored, imPrimary.ID).Error; err != nil {
		t.Fatalf("被删 primary 记录应被回滚重建: %v", err)
	}

	// 2) 被提升的 fallback 应回滚为 fallback
	var afterFallback model.InstanceModel
	if err := model.DB(context.Background()).First(&afterFallback, imFallback.ID).Error; err != nil {
		t.Fatalf("查询 fallback 失败: %v", err)
	}
	if afterFallback.Role != model.ModelRoleFallback {
		t.Errorf("被提升的 fallback 应回滚为 fallback, 实际=%s", afterFallback.Role)
	}

	// 3) instances.ai_model_id 应恢复
	var reloadInst model.Instance
	model.DB(context.Background()).First(&reloadInst, inst.ID)
	if reloadInst.AIModelID != m1.ID {
		t.Errorf("ai_model_id 应回滚为 %d, 实际=%d", m1.ID, reloadInst.AIModelID)
	}
}

// TestDeleteModel_SyncFailRollback_LastModel 验证删除最后一个模型后 TAT 同步
// （remove_model_provider.sh）失败时回滚：记录重建、ai_model_id 恢复。
func TestDeleteModel_SyncFailRollback_LastModel(t *testing.T) {
	setupMultiModelTestDB(t)
	stubSyncScriptRunnerFail(t)

	user, inst := createMultiModelUserAndInstance(t, "del-last-rb-u", "del-last-rb-inst")

	m1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-4-plus", Enabled: true}
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

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("TAT 同步失败应返回 500, 实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 记录应被回滚重建
	var restored model.InstanceModel
	if err := model.DB(context.Background()).First(&restored, imOnly.ID).Error; err != nil {
		t.Fatalf("被删记录应被回滚重建: %v", err)
	}

	// ai_model_id 应恢复
	var reloadInst model.Instance
	model.DB(context.Background()).First(&reloadInst, inst.ID)
	if reloadInst.AIModelID != m1.ID {
		t.Errorf("ai_model_id 应回滚为 %d, 实际=%d", m1.ID, reloadInst.AIModelID)
	}
}

// TestDeleteModel_MissingParam 缺少 instance_model_id → 400
func TestDeleteModel_MissingParam(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "del-mp-u", "del-mp-inst")

	path := fmt.Sprintf("/openclaw/del-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, "")
	rr := httptest.NewRecorder()

	handleDelModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少 instance_model_id 应返回 400, 实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestDeleteModel_NotFound instance_model_id 不属于该实例 → 400
func TestDeleteModel_NotFound(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "del-nf-u", "del-nf-inst")

	form := url.Values{}
	form.Set("instance_model_id", "99999")

	path := fmt.Sprintf("/openclaw/del-model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode())
	rr := httptest.NewRecorder()

	handleDelModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("绑定不存在应返回 400, 实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestDeleteModel_FallbackSuccess_CurrentPrimary 删除 fallback（场景 A），TAT 成功，
// 验证响应携带 current_primary。
func TestDeleteModel_FallbackSuccess_CurrentPrimary(t *testing.T) {
	setupMultiModelTestDB(t)
	stubSyncScriptRunnerOK(t)

	user, inst := createMultiModelUserAndInstance(t, "del-fb-cp-u", "del-fb-cp-inst")

	m1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-4-plus", ModelName: "GLM", Enabled: true}
	m2 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "qwen-max", ModelName: "Qwen", Enabled: true}
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

	if rr.Code != http.StatusOK {
		t.Fatalf("删除 fallback 应返回 200, 实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "current_primary") {
		t.Errorf("响应应含 current_primary, 实际=%s", rr.Body.String())
	}
}

// ─── setModel: 内置模型成功 / TAT 失败回滚 ────────────────────────────────

// TestSetModel_BuiltinSuccess_NewRecord 内置模型首次设置（无现有 primary），
// TAT 成功，验证创建 primary 记录 + instances.ai_model_id 更新。
func TestSetModel_BuiltinSuccess_NewRecord(t *testing.T) {
	setupMultiModelTestDB(t)
	stubAgentScriptRunnerOK(t)

	user, inst := createMultiModelUserAndInstance(t, "set-ok-u", "set-ok-inst")

	m1 := model.AIModel{Provider: "p1", ModelID: "m1", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: "all"}
	model.DB(context.Background()).Create(&m1)

	form := url.Values{}
	form.Set("ai_model_id", strconv.Itoa(int(m1.ID)))

	path := fmt.Sprintf("/openclaw/model?id=%d", inst.ID)
	req := withDomainCtx(multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode()), "https://hatchery.example.com")
	rr := httptest.NewRecorder()

	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusOK {
		t.Fatalf("设置内置模型应返回 200, 实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// primary 记录被创建
	var im model.InstanceModel
	if err := model.DB(context.Background()).Where("instance_id = ? AND role = ?", inst.ID, model.ModelRolePrimary).First(&im).Error; err != nil {
		t.Fatalf("应创建 primary 记录: %v", err)
	}
	if im.AIModelID != m1.ID {
		t.Errorf("primary 应绑定 m1=%d, 实际=%d", m1.ID, im.AIModelID)
	}

	// instances.ai_model_id 同步
	var reloadInst model.Instance
	model.DB(context.Background()).First(&reloadInst, inst.ID)
	if reloadInst.AIModelID != m1.ID {
		t.Errorf("ai_model_id 应=%d, 实际=%d", m1.ID, reloadInst.AIModelID)
	}
}

// TestSetModel_BuiltinSuccess_UpdateExisting 已有 primary 时再设置另一个内置模型，
// 走"更新现有 primary"分支（而非新增），TAT 成功。
func TestSetModel_BuiltinSuccess_UpdateExisting(t *testing.T) {
	setupMultiModelTestDB(t)
	var gotPrimary string
	origRunner := agentScriptRunner
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		gotPrimary = params["primary"]
		return `{"ok":true}`, nil
	}
	t.Cleanup(func() { agentScriptRunner = origRunner })

	user, inst := createMultiModelUserAndInstance(t, "set-upd-u", "set-upd-inst")

	m1 := model.AIModel{Provider: "p1", ModelID: "m1", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: "all"}
	m2 := model.AIModel{Provider: "p2", ModelID: "m2", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: "all"}
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Create(&m2)

	// 预置一条 primary 绑定 m1
	imOld := model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 1}
	model.DB(context.Background()).Create(&imOld)
	model.DB(context.Background()).Model(inst).Update("ai_model_id", m1.ID)

	form := url.Values{}
	form.Set("ai_model_id", strconv.Itoa(int(m2.ID)))

	path := fmt.Sprintf("/openclaw/model?id=%d", inst.ID)
	req := withDomainCtx(multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode()), "https://hatchery.example.com")
	rr := httptest.NewRecorder()

	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusOK {
		t.Fatalf("更新 primary 应返回 200, 实际=%d body=%s", rr.Code, rr.Body.String())
	}

	if gotPrimary != "p2-m2/m2" {
		t.Errorf("TAT primary 应使用提交后的新模型 p2-m2/m2，实际=%q", gotPrimary)
	}

	// 同一条记录被更新为 m2（未新增）
	var im model.InstanceModel
	if err := model.DB(context.Background()).First(&im, imOld.ID).Error; err != nil {
		t.Fatalf("原 primary 记录应被复用更新: %v", err)
	}
	if im.AIModelID != m2.ID {
		t.Errorf("primary 应被更新为 m2=%d, 实际=%d", m2.ID, im.AIModelID)
	}

	var count int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 1 {
		t.Errorf("应只有 1 条 primary 记录（更新而非新增）, 实际=%d", count)
	}
}

// TestSetModel_BuiltinTATFail_RollbackUpdate 已有 primary 时设置新模型但 TAT 失败，
// 验证 rollbackSetModelDB 的"更新分支"：还原旧 primary 字段 + instances.ai_model_id。
func TestSetModel_BuiltinTATFail_RollbackUpdate(t *testing.T) {
	setupMultiModelTestDB(t)
	stubAgentScriptRunnerFail(t)

	user, inst := createMultiModelUserAndInstance(t, "set-rb-u", "set-rb-inst")

	m1 := model.AIModel{Provider: "p1", ModelID: "m1", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: "all"}
	m2 := model.AIModel{Provider: "p2", ModelID: "m2", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: "all"}
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Create(&m2)

	imOld := model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 1}
	model.DB(context.Background()).Create(&imOld)
	model.DB(context.Background()).Model(inst).Update("ai_model_id", m1.ID)

	form := url.Values{}
	form.Set("ai_model_id", strconv.Itoa(int(m2.ID)))

	path := fmt.Sprintf("/openclaw/model?id=%d", inst.ID)
	req := withDomainCtx(multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode()), "https://hatchery.example.com")
	rr := httptest.NewRecorder()

	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("TAT 失败应返回 500, 实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 旧 primary 记录字段应被还原为 m1
	var im model.InstanceModel
	if err := model.DB(context.Background()).First(&im, imOld.ID).Error; err != nil {
		t.Fatalf("原 primary 记录应保留: %v", err)
	}
	if im.AIModelID != m1.ID {
		t.Errorf("primary 应回滚为 m1=%d, 实际=%d", m1.ID, im.AIModelID)
	}

	// instances.ai_model_id 应回滚为 m1
	var reloadInst model.Instance
	model.DB(context.Background()).First(&reloadInst, inst.ID)
	if reloadInst.AIModelID != m1.ID {
		t.Errorf("ai_model_id 应回滚为 %d, 实际=%d", m1.ID, reloadInst.AIModelID)
	}
}

func dropAIModelsAfterFirstUpdate(t *testing.T) *error {
	t.Helper()
	cb := model.DB(context.Background()).Callback().Update()
	name := "test:drop_ai_models_after_first_update:" + strings.ReplaceAll(t.Name(), "/", "_")
	var dropErr error
	dropped := false
	if err := cb.After("gorm:after_update").Register(name, func(tx *gorm.DB) {
		if dropped {
			return
		}
		dropped = true
		dropErr = tx.Session(&gorm.Session{NewDB: true}).Migrator().DropTable(&model.AIModel{})
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() {
		if err := cb.Remove(name); err != nil {
			t.Fatalf("remove callback: %v", err)
		}
	})
	return &dropErr
}

func TestSetModel_BuiltinBuildParamsFail_Rollback(t *testing.T) {
	setupMultiModelTestDB(t)

	var tatCalled atomic.Int32
	origRunner := agentScriptRunner
	agentScriptRunner = func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
		tatCalled.Add(1)
		return `{"ok":true}`, nil
	}
	t.Cleanup(func() { agentScriptRunner = origRunner })

	user, inst := createMultiModelUserAndInstance(t, "set-param-rb-u", "set-param-rb-inst")

	m1 := model.AIModel{Provider: "p1", ModelID: "m1", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: model.VisibilityAll}
	m2 := model.AIModel{Provider: "p2", ModelID: "m2", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: model.VisibilityAll}
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Create(&m2)

	imOld := model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 1}
	model.DB(context.Background()).Create(&imOld)
	model.DB(context.Background()).Model(inst).Update("ai_model_id", m1.ID)

	dropErr := dropAIModelsAfterFirstUpdate(t)

	form := url.Values{}
	form.Set("ai_model_id", strconv.Itoa(int(m2.ID)))

	path := fmt.Sprintf("/openclaw/model?id=%d", inst.ID)
	req := withDomainCtx(multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode()), "https://hatchery.example.com")
	rr := httptest.NewRecorder()

	handleSetModel(rr, req, testCVMFetcher)

	if *dropErr != nil {
		t.Fatalf("drop ai_models: %v", *dropErr)
	}
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("参数生成失败应返回 500, 实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if got := tatCalled.Load(); got != 0 {
		t.Errorf("参数生成失败时不应下发 TAT, 实际调用次数=%d", got)
	}
	var im model.InstanceModel
	if err := model.DB(context.Background()).First(&im, imOld.ID).Error; err != nil {
		t.Fatalf("原 primary 记录应保留: %v", err)
	}
	if im.AIModelID != m1.ID {
		t.Errorf("primary 应回滚为 m1=%d, 实际=%d", m1.ID, im.AIModelID)
	}
}

func TestSetModel_CustomBuildParamsFail_Rollback(t *testing.T) {
	setupMultiModelTestDB(t)

	var tatCalled atomic.Int32
	origRunner := agentScriptRunner
	agentScriptRunner = func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
		tatCalled.Add(1)
		return `{"ok":true}`, nil
	}
	t.Cleanup(func() { agentScriptRunner = origRunner })

	user, inst := createMultiModelUserAndInstance(t, "set-custom-param-rb-u", "set-custom-param-rb-inst")

	customFlag := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: model.BuiltinModelID, Enabled: true, Visible: true, VisibilityType: model.VisibilityAll}
	m1 := model.AIModel{Provider: "p1", ModelID: "m1", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: model.VisibilityAll}
	m2 := model.AIModel{Provider: "p2", ModelID: "m2", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", InputTypes: `["image"]`, ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: model.VisibilityAll}
	model.DB(context.Background()).Create(&customFlag)
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Create(&m2)

	imOld := model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 1}
	model.DB(context.Background()).Create(&imOld)
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: model.ModelRoleFallback, SortOrder: 2})
	model.DB(context.Background()).Model(inst).Update("ai_model_id", m1.ID)

	dropErr := dropAIModelsAfterFirstUpdate(t)

	form := url.Values{}
	form.Set("ai_model_id", "0")
	form.Set("provider", "custom")
	form.Set("model_id", "custom-param-rb")
	form.Set("api_key", "sk-test")
	form.Set("url", "https://api.example.com/v1")
	form.Set("model_type", "openai-completions")

	path := fmt.Sprintf("/openclaw/model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode())
	rr := httptest.NewRecorder()

	handleSetModel(rr, req, testCVMFetcher)

	if *dropErr != nil {
		t.Fatalf("drop ai_models: %v", *dropErr)
	}
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("自定义模型参数生成失败应返回 500, 实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if got := tatCalled.Load(); got != 0 {
		t.Errorf("参数生成失败时不应下发 TAT, 实际调用次数=%d", got)
	}
	var im model.InstanceModel
	if err := model.DB(context.Background()).First(&im, imOld.ID).Error; err != nil {
		t.Fatalf("原 primary 记录应保留: %v", err)
	}
	if im.AIModelID != m1.ID || im.CustomModelID != "" || im.CustomModelConfig != "" {
		t.Errorf("primary 应回滚为内置 m1，实际 ai_model_id=%d custom_model_id=%q", im.AIModelID, im.CustomModelID)
	}
}

func TestSetModel_BuiltinTATFail_RollbackFailure(t *testing.T) {
	setupMultiModelTestDB(t)

	origRunner := agentScriptRunner
	agentScriptRunner = func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		if err := model.DB(ctx).Migrator().DropTable(&model.InstanceModel{}); err != nil {
			t.Fatalf("drop instance_models: %v", err)
		}
		return "", hcommon.I18nError(i18n.MsgTATFailed)
	}
	t.Cleanup(func() { agentScriptRunner = origRunner })

	user, inst := createMultiModelUserAndInstance(t, "set-rb-fail-u", "set-rb-fail-inst")

	m1 := model.AIModel{Provider: "p1", ModelID: "m1", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: "all"}
	m2 := model.AIModel{Provider: "p2", ModelID: "m2", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: "all"}
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Create(&m2)

	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 1})
	model.DB(context.Background()).Model(inst).Update("ai_model_id", m1.ID)

	form := url.Values{}
	form.Set("ai_model_id", strconv.Itoa(int(m2.ID)))

	path := fmt.Sprintf("/openclaw/model?id=%d", inst.ID)
	req := withDomainCtx(multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode()), "https://hatchery.example.com")
	rr := httptest.NewRecorder()

	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("TAT 失败且回滚失败应返回 500, 实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSetModel_CustomTATFail_RollbackUpdate(t *testing.T) {
	setupMultiModelTestDB(t)
	stubAgentScriptRunnerFail(t)

	user, inst := createMultiModelUserAndInstance(t, "set-custom-rb-u", "set-custom-rb-inst")

	customFlag := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: model.BuiltinModelID, Enabled: true, Visible: true, VisibilityType: model.VisibilityAll}
	model.DB(context.Background()).Create(&customFlag)

	m1 := model.AIModel{Provider: "p1", ModelID: "m1", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: model.VisibilityAll}
	model.DB(context.Background()).Create(&m1)

	imOld := model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 1}
	model.DB(context.Background()).Create(&imOld)
	model.DB(context.Background()).Model(inst).Update("ai_model_id", m1.ID)

	form := url.Values{}
	form.Set("ai_model_id", "0")
	form.Set("provider", "custom")
	form.Set("model_id", "custom-rb")
	form.Set("api_key", "sk-test")
	form.Set("url", "https://api.example.com/v1")
	form.Set("model_type", "openai-completions")

	path := fmt.Sprintf("/openclaw/model?id=%d", inst.ID)
	req := multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode())
	rr := httptest.NewRecorder()

	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("自定义模型 TAT 失败应返回 500, 实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var im model.InstanceModel
	if err := model.DB(context.Background()).First(&im, imOld.ID).Error; err != nil {
		t.Fatalf("原 primary 记录应保留: %v", err)
	}
	if im.AIModelID != m1.ID || im.CustomModelID != "" || im.CustomModelConfig != "" {
		t.Errorf("primary 应回滚为内置 m1，实际 ai_model_id=%d custom_model_id=%q", im.AIModelID, im.CustomModelID)
	}

	var reloadInst model.Instance
	model.DB(context.Background()).First(&reloadInst, inst.ID)
	if reloadInst.AIModelID != m1.ID || reloadInst.CustomModelConfig != "" {
		t.Errorf("instances 应回滚为 m1，实际 ai_model_id=%d custom_config=%q", reloadInst.AIModelID, reloadInst.CustomModelConfig)
	}
}

// TestSetModel_VisibilityDenied 模型不在实例分组可见范围 → 403。
func TestSetModel_VisibilityDenied(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "set-vis-u", "set-vis-inst")

	// group 可见但实例无分组 → 不可见
	m1 := model.AIModel{Provider: "p1", ModelID: "m1", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", Enabled: true, Visible: true, VisibilityType: model.VisibilityGroup}
	model.DB(context.Background()).Create(&m1)

	form := url.Values{}
	form.Set("ai_model_id", strconv.Itoa(int(m1.ID)))

	path := fmt.Sprintf("/openclaw/model?id=%d", inst.ID)
	req := withDomainCtx(multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode()), "https://hatchery.example.com")
	rr := httptest.NewRecorder()

	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("不可见模型应返回 403, 实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── setModel 前置 fallback 校验（ea21ff80） ──────────────────────────────
//
// 覆盖 setModel 新增的"目标 ai_model_id 已作为 fallback 绑到本实例 → 409"
// 前置校验代码块：
//
//	var boundFallback model.InstanceModel
//	if err := model.DB(r.Context()).Where(
//	    "instance_id = ? AND ai_model_id = ? AND custom_model_id = ? AND role = ?",
//	    instance.ID, aiModel.ID, "", model.ModelRoleFallback,
//	).First(&boundFallback).Error; err == nil {
//	    writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgModelAlreadyBoundAsFallback))
//	    return
//	}
//
// 业务目标：避免 setModel UPDATE primary 行时撞 (instance_id, ai_model_id, '')
// 唯一键，触发 MySQL 1062 错误。

// TestSetModel_TargetAlreadyBoundAsFallback_Returns409 验证核心场景：
// 目标 ai_model_id 已作为 fallback 绑到本实例 → 拦截 → 返回 409 +
// MsgModelAlreadyBoundAsFallback，且 TAT 完全未被调用、DB 状态保持原样。
func TestSetModel_TargetAlreadyBoundAsFallback_Returns409(t *testing.T) {
	setupMultiModelTestDB(t)

	// 拦截 TAT：命中前置校验时不应进入事务、不应调用 set_model 脚本
	var tatCalled atomic.Int32
	origRunner := agentScriptRunner
	agentScriptRunner = func(_ context.Context, _, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		tatCalled.Add(1)
		return `{"ok":true}`, nil
	}
	t.Cleanup(func() { agentScriptRunner = origRunner })

	user, inst := createMultiModelUserAndInstance(t, "set-fb-conflict-u", "set-fb-conflict-inst")

	// m1 当前是 primary；m2 当前是 fallback（要被设为 primary 时触发拦截）
	m1 := model.AIModel{Provider: "hatchery", ModelID: "m-primary", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: model.VisibilityAll}
	m2 := model.AIModel{Provider: "hatchery", ModelID: "m-fallback", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: model.VisibilityAll}
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Create(&m2)

	imPrimary := model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 1}
	imFallback := model.InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: model.ModelRoleFallback, SortOrder: 2}
	model.DB(context.Background()).Create(&imPrimary)
	model.DB(context.Background()).Create(&imFallback)
	model.DB(context.Background()).Model(inst).Update("ai_model_id", m1.ID)

	// 把 m2 设为 primary —— 应被前置校验拦截
	form := url.Values{}
	form.Set("ai_model_id", strconv.Itoa(int(m2.ID)))
	path := fmt.Sprintf("/openclaw/model?id=%d", inst.ID)
	req := withDomainCtx(multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode()), "https://hatchery.example.com")
	rr := httptest.NewRecorder()

	handleSetModel(rr, req, testCVMFetcher)

	// 1) HTTP 409
	if rr.Code != http.StatusConflict {
		t.Fatalf("应返回 409, 实际=%d body=%s", rr.Code, rr.Body.String())
	}
	// 2) 响应包含 MsgModelAlreadyBoundAsFallback 的翻译文案
	wantMsg := i18n.T(req.Context(), i18n.MsgModelAlreadyBoundAsFallback)
	if !strings.Contains(rr.Body.String(), wantMsg) {
		t.Errorf("响应应包含 %q, 实际 body=%s", wantMsg, rr.Body.String())
	}
	// 3) TAT 未被调用
	if got := tatCalled.Load(); got != 0 {
		t.Errorf("前置校验命中时不应下发 TAT, 实际调用次数=%d", got)
	}
	// 4) primary 记录未被改写
	var afterPrimary model.InstanceModel
	if err := model.DB(context.Background()).First(&afterPrimary, imPrimary.ID).Error; err != nil {
		t.Fatalf("primary 记录应保留: %v", err)
	}
	if afterPrimary.AIModelID != m1.ID || afterPrimary.Role != model.ModelRolePrimary {
		t.Errorf("primary 记录被错误改写: ai_model_id=%d role=%s", afterPrimary.AIModelID, afterPrimary.Role)
	}
	// 5) fallback 记录未被改写
	var afterFallback model.InstanceModel
	if err := model.DB(context.Background()).First(&afterFallback, imFallback.ID).Error; err != nil {
		t.Fatalf("fallback 记录应保留: %v", err)
	}
	if afterFallback.AIModelID != m2.ID || afterFallback.Role != model.ModelRoleFallback {
		t.Errorf("fallback 记录被错误改写: ai_model_id=%d role=%s", afterFallback.AIModelID, afterFallback.Role)
	}
	// 6) instances.ai_model_id 保持 m1
	var reloadInst model.Instance
	model.DB(context.Background()).First(&reloadInst, inst.ID)
	if reloadInst.AIModelID != m1.ID {
		t.Errorf("instances.ai_model_id 应保持 %d, 实际=%d", m1.ID, reloadInst.AIModelID)
	}
}

// TestSetModel_TargetFallbackOnOtherInstance_NoBlock 覆盖 WHERE 条件中
// instance_id 维度：同一模型作为 fallback 绑在 *别的* 实例上时，
// 本实例的 setModel 应放行（前置校验 First 返回 ErrRecordNotFound，进入事务）。
func TestSetModel_TargetFallbackOnOtherInstance_NoBlock(t *testing.T) {
	setupMultiModelTestDB(t)
	stubAgentScriptRunnerOK(t) // 放行后会下发 TAT，必须 stub 成功

	user, inst := createMultiModelUserAndInstance(t, "set-fb-other-u", "set-fb-other-inst")

	// 另一个实例 + 另一个用户，作为干扰项
	otherUser := &model.User{Username: "set-fb-other-user", Password: "t", Role: "user"}
	model.DB(context.Background()).Create(otherUser)
	otherInst := &model.Instance{Name: "other", InstanceId: "ins-other", UserID: otherUser.ID, AgentType: model.AgentTypeOpenClaw, RuntimeUser: "ubuntu"}
	model.DB(context.Background()).Create(otherInst)

	m := model.AIModel{Provider: "hatchery", ModelID: "shared", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: model.VisibilityAll}
	model.DB(context.Background()).Create(&m)

	// shared 模型作为 fallback 绑在 *别的* 实例
	model.DB(context.Background()).Create(&model.InstanceModel{
		InstanceID: otherInst.ID, AIModelID: m.ID, Role: model.ModelRoleFallback, SortOrder: 1,
	})

	// 在本实例上 setModel(shared) → 不应被拦截
	form := url.Values{}
	form.Set("ai_model_id", strconv.Itoa(int(m.ID)))
	path := fmt.Sprintf("/openclaw/model?id=%d", inst.ID)
	req := withDomainCtx(multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode()), "https://hatchery.example.com")
	rr := httptest.NewRecorder()

	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusOK {
		t.Fatalf("其他实例的 fallback 不应阻塞本实例, 实际=%d body=%s", rr.Code, rr.Body.String())
	}
	// 本实例应新建一条 primary 绑定 shared
	var im model.InstanceModel
	if err := model.DB(context.Background()).Where("instance_id = ? AND role = ?", inst.ID, model.ModelRolePrimary).First(&im).Error; err != nil {
		t.Fatalf("本实例应有 primary 记录: %v", err)
	}
	if im.AIModelID != m.ID {
		t.Errorf("primary 应=%d, 实际=%d", m.ID, im.AIModelID)
	}
}

// TestSetModel_TargetAlreadyBoundAsPrimary_NoBlock 覆盖 WHERE 条件中
// role 维度：目标模型已是该实例的 primary（而非 fallback），不构成唯一键冲突，
// 应走 setModel 的"更新已有 primary"路径。
func TestSetModel_TargetAlreadyBoundAsPrimary_NoBlock(t *testing.T) {
	setupMultiModelTestDB(t)
	stubAgentScriptRunnerOK(t)

	user, inst := createMultiModelUserAndInstance(t, "set-fb-asprim-u", "set-fb-asprim-inst")

	m1 := model.AIModel{Provider: "p1", ModelID: "p1m", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: model.VisibilityAll}
	m2 := model.AIModel{Provider: "p2", ModelID: "p2m", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: model.VisibilityAll}
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Create(&m2)

	// 当前 primary = m1，无 fallback
	imOld := model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 1}
	model.DB(context.Background()).Create(&imOld)
	model.DB(context.Background()).Model(inst).Update("ai_model_id", m1.ID)

	// 把 m2 设为 primary（m2 未在本实例任何记录中，前置校验应放行）
	form := url.Values{}
	form.Set("ai_model_id", strconv.Itoa(int(m2.ID)))
	path := fmt.Sprintf("/openclaw/model?id=%d", inst.ID)
	req := withDomainCtx(multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode()), "https://hatchery.example.com")
	rr := httptest.NewRecorder()

	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusOK {
		t.Fatalf("无 fallback 冲突应放行, 实际=%d body=%s", rr.Code, rr.Body.String())
	}
	// 原 primary 行应被复用更新为 m2
	var im model.InstanceModel
	if err := model.DB(context.Background()).First(&im, imOld.ID).Error; err != nil {
		t.Fatalf("原 primary 记录应保留: %v", err)
	}
	if im.AIModelID != m2.ID {
		t.Errorf("primary 应被更新为 m2=%d, 实际=%d", m2.ID, im.AIModelID)
	}
}

// TestSetModel_TargetFallbackWithDifferentCustomModelID_NoBlock 覆盖 WHERE
// 条件中 custom_model_id 维度：前置校验只在 custom_model_id=” 时命中，
// 因此带非空 custom_model_id 的历史 fallback 行不应误伤本次 setModel。
//
// 理由：setModel 走的是内置模型路径，DB 行的 custom_model_id 恒为 ”；
// 用唯一键 (instance_id, ai_model_id, ”) 才会真正冲突，
// (instance_id, ai_model_id, 'stale') 不会冲突，不应被拦截。
func TestSetModel_TargetFallbackWithDifferentCustomModelID_NoBlock(t *testing.T) {
	setupMultiModelTestDB(t)
	stubAgentScriptRunnerOK(t)

	user, inst := createMultiModelUserAndInstance(t, "set-fb-cmid-u", "set-fb-cmid-inst")

	m := model.AIModel{Provider: "hatchery", ModelID: "qx", ModelType: "openai-completions", APIKey: "sk", URL: "https://x", ContextLen: 128000, Enabled: true, Visible: true, VisibilityType: model.VisibilityAll}
	model.DB(context.Background()).Create(&m)

	// 制造一条"同 instance_id + 同 ai_model_id + role=fallback 但 custom_model_id 非空"的记录
	model.DB(context.Background()).Create(&model.InstanceModel{
		InstanceID:    inst.ID,
		AIModelID:     m.ID,
		CustomModelID: "stale-cmid",
		Role:          model.ModelRoleFallback,
		SortOrder:     1,
	})

	form := url.Values{}
	form.Set("ai_model_id", strconv.Itoa(int(m.ID)))
	path := fmt.Sprintf("/openclaw/model?id=%d", inst.ID)
	req := withDomainCtx(multiModelReqWithSession(t, http.MethodPost, path, user.Username, form.Encode()), "https://hatchery.example.com")
	rr := httptest.NewRecorder()

	handleSetModel(rr, req, testCVMFetcher)

	// custom_model_id != '' → 前置校验不命中 → 不应返回 409
	if rr.Code == http.StatusConflict {
		t.Fatalf("custom_model_id 非空的 fallback 不应触发前置拦截, body=%s", rr.Body.String())
	}
}

func TestApplyBuiltinModel_MatchesManualAddModelOrdering(t *testing.T) {
	setupMultiModelTestDB(t)
	originalRunner := injectModelScriptRunner
	injectModelScriptRunner = func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
		return `{"ok":true}`, nil
	}
	t.Cleanup(func() { injectModelScriptRunner = originalRunner })

	_, instance := createMultiModelUserAndInstance(t, "preset-model-u", "preset-model-inst")
	primary := model.AIModel{
		Provider:       model.BuiltinModelProvider,
		ModelID:        "preset-primary",
		ModelName:      "Preset Primary",
		ModelType:      "openai-completions",
		Enabled:        true,
		Visible:        true,
		VisibilityType: model.VisibilityAll,
	}
	fallback := model.AIModel{
		Provider:       model.BuiltinModelProvider,
		ModelID:        "preset-fallback",
		ModelName:      "Preset Fallback",
		ModelType:      "openai-completions",
		Enabled:        true,
		Visible:        true,
		VisibilityType: model.VisibilityAll,
	}
	if err := model.DB(context.Background()).Create(&primary).Error; err != nil {
		t.Fatalf("create primary model: %v", err)
	}
	if err := model.DB(context.Background()).Create(&fallback).Error; err != nil {
		t.Fatalf("create fallback model: %v", err)
	}

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Domain: "https://hatchery.example.com",
	})
	first, firstErr := applyBuiltinModel(ctx, instance, primary.ID)
	if firstErr != nil {
		t.Fatalf("apply primary: %v", firstErr.err)
	}
	second, secondErr := applyBuiltinModel(ctx, instance, fallback.ID)
	if secondErr != nil {
		t.Fatalf("apply fallback: %v", secondErr.err)
	}
	if first.role != model.ModelRolePrimary {
		t.Fatalf("first model role = %q, want primary", first.role)
	}
	if second.role != model.ModelRoleFallback {
		t.Fatalf("second model role = %q, want fallback", second.role)
	}

	var persisted model.Instance
	if err := model.DB(ctx).First(&persisted, instance.ID).Error; err != nil {
		t.Fatalf("reload instance: %v", err)
	}
	if persisted.AIModelID != primary.ID {
		t.Fatalf("instance ai_model_id = %d, want %d", persisted.AIModelID, primary.ID)
	}
	var bindings []model.InstanceModel
	if err := model.DB(ctx).Where("instance_id = ?", instance.ID).Order("sort_order ASC").Find(&bindings).Error; err != nil {
		t.Fatalf("list bindings: %v", err)
	}
	if len(bindings) != 2 {
		t.Fatalf("binding count = %d, want 2", len(bindings))
	}
	if bindings[0].AIModelID != primary.ID || bindings[0].Role != model.ModelRolePrimary {
		t.Fatalf("first binding = %+v, want primary model", bindings[0])
	}
	if bindings[1].AIModelID != fallback.ID || bindings[1].Role != model.ModelRoleFallback {
		t.Fatalf("second binding = %+v, want fallback model", bindings[1])
	}
}

// --- parseCustomModelFromForm max_tokens 选填语义 测试 ---

func doParseCustomModel(t *testing.T, form url.Values) (*customModelConfig, error) {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/openclaw/set-model",
		strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return parseCustomModelFromForm(r)
}

func customModelFormBase() url.Values {
	form := url.Values{}
	form.Set("provider", "custom")
	form.Set("model_id", "test-custom")
	form.Set("api_key", "sk-test")
	form.Set("url", "https://api.example.com/v1")
	form.Set("model_type", "openai-completions")
	form.Set("context_len", "128000")
	form.Add("input_types", "text")
	return form
}

func TestParseCustomModelFromForm_MaxTokensOmitted(t *testing.T) {
	cfg, err := doParseCustomModel(t, customModelFormBase())
	if err != nil {
		t.Fatalf("期望成功, 错误=%v", err)
	}
	if cfg.MaxTokens != 0 {
		t.Errorf("不传 max_tokens 应返回 0, 实际=%d", cfg.MaxTokens)
	}
}

func TestParseCustomModelFromForm_MaxTokensExplicitZero(t *testing.T) {
	form := customModelFormBase()
	form.Set("max_tokens", "0")
	cfg, err := doParseCustomModel(t, form)
	if err != nil {
		t.Fatalf("期望成功, 错误=%v", err)
	}
	if cfg.MaxTokens != 0 {
		t.Errorf("传 max_tokens=0 应返回 0, 实际=%d", cfg.MaxTokens)
	}
}

func TestParseCustomModelFromForm_MaxTokensPositive(t *testing.T) {
	form := customModelFormBase()
	form.Set("max_tokens", "4096")
	cfg, err := doParseCustomModel(t, form)
	if err != nil {
		t.Fatalf("期望成功, 错误=%v", err)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("传 max_tokens=4096 应返回 4096, 实际=%d", cfg.MaxTokens)
	}
}

func TestParseCustomModelFromForm_MaxTokensNegative(t *testing.T) {
	form := customModelFormBase()
	form.Set("max_tokens", "-1")
	_, err := doParseCustomModel(t, form)
	if err == nil {
		t.Fatal("传 max_tokens=-1 应返回错误, 实际 nil")
	}
}

func TestParseCustomModelFromForm_MaxTokensNonInteger(t *testing.T) {
	form := customModelFormBase()
	form.Set("max_tokens", "abc")
	_, err := doParseCustomModel(t, form)
	if err == nil {
		t.Fatal("传 max_tokens=abc 应返回错误, 实际 nil")
	}
}

// --- resolveSetModelBindingForInstance max_tokens 选填语义 测试 ---

func TestResolveSetModelBinding_MaxTokensNegative(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.AIModel{
		Provider: model.BuiltinModelProvider, ModelID: model.BuiltinModelID, Visible: true,
	})

	user := model.User{Username: "test-user", Password: "x", Role: "admin"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{Name: "test-inst", UserID: user.ID, InstanceId: "ins-test"}
	model.DB(context.Background()).Create(&inst)

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Domain: "test.com"})
	in := setModelInput{
		AIModelID:  0,
		Provider:   "custom",
		ModelID:    "test-model",
		APIKey:     "sk-test",
		URL:        "https://api.example.com/v1",
		ModelType:  "openai-completions",
		InputTypes: []string{"text"},
		MaxTokens:  -1,
	}

	binding, applyErr := resolveSetModelBindingForInstance(ctx, &inst, in, model.ModelRolePrimary, 1)
	if applyErr != nil {
		t.Fatalf("期望成功, 错误=%v", applyErr)
	}
	if binding.InjectModel.MaxTokens != 0 {
		t.Errorf("max_tokens=-1 应归零为 0, 实际=%d", binding.InjectModel.MaxTokens)
	}
}

func TestResolveSetModelBinding_MaxTokensOmitted(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.AIModel{
		Provider: model.BuiltinModelProvider, ModelID: model.BuiltinModelID, Visible: true,
	})

	user := model.User{Username: "test-user2", Password: "x", Role: "admin"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{Name: "test-inst2", UserID: user.ID, InstanceId: "ins-test2"}
	model.DB(context.Background()).Create(&inst)

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Domain: "test.com"})
	in := setModelInput{
		AIModelID:  0,
		Provider:   "custom",
		ModelID:    "test-model2",
		APIKey:     "sk-test",
		URL:        "https://api.example.com/v1",
		ModelType:  "openai-completions",
		InputTypes: []string{"text"},
	}

	binding, applyErr := resolveSetModelBindingForInstance(ctx, &inst, in, model.ModelRolePrimary, 1)
	if applyErr != nil {
		t.Fatalf("期望成功, 错误=%v", applyErr)
	}
	if binding.InjectModel.MaxTokens != 0 {
		t.Errorf("不传 max_tokens 应为 0, 实际=%d", binding.InjectModel.MaxTokens)
	}
}

func TestResolveSetModelBinding_MaxTokensPositive(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.AIModel{
		Provider: model.BuiltinModelProvider, ModelID: model.BuiltinModelID, Visible: true,
	})

	user := model.User{Username: "test-user3", Password: "x", Role: "admin"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{Name: "test-inst3", UserID: user.ID, InstanceId: "ins-test3"}
	model.DB(context.Background()).Create(&inst)

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Domain: "test.com"})
	in := setModelInput{
		AIModelID:  0,
		Provider:   "custom",
		ModelID:    "test-model3",
		APIKey:     "sk-test",
		URL:        "https://api.example.com/v1",
		ModelType:  "openai-completions",
		InputTypes: []string{"text"},
		MaxTokens:  4096,
	}

	binding, applyErr := resolveSetModelBindingForInstance(ctx, &inst, in, model.ModelRolePrimary, 1)
	if applyErr != nil {
		t.Fatalf("期望成功, 错误=%v", applyErr)
	}
	if binding.InjectModel.MaxTokens != 4096 {
		t.Errorf("传 max_tokens=4096 应保留, 实际=%d", binding.InjectModel.MaxTokens)
	}
}
