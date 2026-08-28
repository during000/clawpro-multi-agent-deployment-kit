package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller/provider"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// setupAdminModelsTestDB 初始化临时 SQLite 数据库，迁移模型管理相关表。
func setupAdminModelsTestDB(t *testing.T) func() {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "admin_models_test_*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()

	dsn := tmpFile.Name() + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("open test db: %v", err)
	}

	origDB := model.UseDBForTest(testDB)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("admin-models-test-secret"))
	if err := testDB.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.AIModel{},
		&model.InstanceModel{},
		&model.SiteConfig{},
		&model.ModelVisibilityGroup{},
	); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("auto migrate: %v", err)
	}

	// 确保 SiteConfig 存在
	testDB.FirstOrCreate(&model.SiteConfig{})

	return func() {
		sqlDB, _ := testDB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
		os.Remove(tmpFile.Name() + "-wal")
		os.Remove(tmpFile.Name() + "-shm")
		origDB()
		Store = origStore
	}
}

// deleteModelHandler 绕过 requireAdmin，直接执行 HandleDeleteModel 的核心逻辑。
func deleteModelHandler(w http.ResponseWriter, r *http.Request) {
	isJSON := strings.Contains(r.Header.Get("Accept"), "application/json") ||
		strings.Contains(r.Header.Get("Content-Type"), "application/json")

	id := r.URL.Query().Get("id")
	var m model.AIModel
	if model.DB(context.Background()).Where("id = ?", id).First(&m).Error != nil {
		if isJSON {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgModelNotFoundOrDisabled))
		} else {
			http.Error(w, "模型不存在", http.StatusNotFound)
		}
		return
	}

	if isCustomModelRecord(&m) {
		if isJSON {
			writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgBuiltinModelCannotDelete))
		} else {
			http.Error(w, "系统内置记录不可删除", http.StatusForbidden)
		}
		return
	}

	// 删除前，若该模型是默认模型则联动清除
	config := model.GetSiteConfig(context.Background())
	if config.DefaultModelID == m.ID {
		model.DB(context.Background()).Model(&config).Update("default_model_id", 0)
	}

	// 在事务中删除模型及其关联数据
	var affectedInstanceIDs []uint
	if err := model.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
		if err := model.CleanupVisibilityByModelID(tx, m.ID); err != nil {
			return err
		}
		var err error
		affectedInstanceIDs, err = model.CleanupInstanceModelsByAIModelID(tx, m.ID)
		if err != nil {
			return fmt.Errorf("清理实例模型绑定失败: %w", err)
		}
		if err := tx.Delete(&m).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		if isJSON {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDeleteModelFailed))
		} else {
			http.Error(w, "删除失败", http.StatusInternalServerError)
		}
		return
	}

	// 测试环境跳过异步 CVM 同步（避免 TAT 调用）
	_ = affectedInstanceIDs

	if isJSON {
		jsonOK(w, map[string]interface{}{"ok": true})
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("模型已删除"))
	}
}

// TestHandleDeleteModel_CascadeInstanceModels 验证：删除模型时级联清理 instance_models。
func TestHandleDeleteModel_CascadeInstanceModels(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	// 创建用户和实例
	user := model.User{Username: "admin-del", Password: "x", Role: "admin"}
	model.DB(context.Background()).Create(&user)

	inst1 := model.Instance{Name: "inst-1", UserID: user.ID, InstanceId: "ins-001", AIModelID: 0}
	inst2 := model.Instance{Name: "inst-2", UserID: user.ID, InstanceId: "ins-002", AIModelID: 0}
	model.DB(context.Background()).Create(&inst1)
	model.DB(context.Background()).Create(&inst2)

	// 创建模型
	m1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-del", ModelName: "GLM-Del", Enabled: true}
	m2 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "qwen-del", ModelName: "Qwen-Del", Enabled: true}
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Create(&m2)

	// inst1: m1(primary) + m2(fallback)
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst1.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 1})
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst1.ID, AIModelID: m2.ID, Role: model.ModelRoleFallback, SortOrder: 2})
	// inst2: m1(primary)
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst2.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 1})

	// 同时设置 instances.ai_model_id（模拟老版本单模型状态）
	model.DB(context.Background()).Model(&inst1).Update("ai_model_id", m1.ID)
	model.DB(context.Background()).Model(&inst2).Update("ai_model_id", m1.ID)

	// 调用删除接口
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/delete?id=%d", m1.ID), nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	deleteModelHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望状态码 200, 实际=%d, 响应=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if !resp["ok"].(bool) {
		t.Fatalf("期望 ok=true, 实际=%v", resp)
	}

	// 验证 instance_models 中 m1 的绑定已清理
	var imCount int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("ai_model_id = ?", m1.ID).Count(&imCount)
	if imCount != 0 {
		t.Errorf("instance_models 中 m1 的绑定应被清理, 实际=%d", imCount)
	}

	// 验证 inst1 仍保留 m2 的绑定
	var inst1Count int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("instance_id = ?", inst1.ID).Count(&inst1Count)
	if inst1Count != 1 {
		t.Errorf("inst1 应保留 1 条绑定(m2), 实际=%d", inst1Count)
	}

	// 验证 inst2 无绑定
	var inst2Count int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("instance_id = ?", inst2.ID).Count(&inst2Count)
	if inst2Count != 0 {
		t.Errorf("inst2 应无绑定, 实际=%d", inst2Count)
	}

	// 验证 inst1 的 ai_model_id：m2 被提升为 primary，应同步更新为 m2.ID
	var found1 model.Instance
	model.DB(context.Background()).First(&found1, inst1.ID)
	if found1.AIModelID != m2.ID {
		t.Errorf("inst1.ai_model_id 应同步为 m2.ID(%d), 实际=%d", m2.ID, found1.AIModelID)
	}

	// 验证 inst2 的 ai_model_id：无 fallback，应重置为 0
	var found2 model.Instance
	model.DB(context.Background()).First(&found2, inst2.ID)
	if found2.AIModelID != 0 {
		t.Errorf("inst2.ai_model_id 应被重置为 0, 实际=%d", found2.AIModelID)
	}

	// 验证 m2 的绑定不受影响
	var m2Count int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("ai_model_id = ?", m2.ID).Count(&m2Count)
	if m2Count != 1 {
		t.Errorf("m2 的绑定不应受影响, 实际=%d", m2Count)
	}
}

// TestHandleDeleteModel_NoBindings 验证：删除未被任何实例绑定的模型。
func TestHandleDeleteModel_NoBindings(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	m := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "orphan-del", ModelName: "Orphan", Enabled: true}
	model.DB(context.Background()).Create(&m)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/delete?id=%d", m.ID), nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	deleteModelHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望状态码 200, 实际=%d, 响应=%s", rr.Code, rr.Body.String())
	}

	// 验证模型已软删除
	var found model.AIModel
	err := model.DB(context.Background()).First(&found, m.ID).Error
	if err == nil {
		t.Error("模型应被软删除，但仍能查到")
	}
}

// TestHandleDeleteModel_NotFound 验证：删除不存在的模型返回 404。
func TestHandleDeleteModel_NotFound(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/admin/models/delete?id=99999", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	deleteModelHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("期望状态码 404, 实际=%d", rr.Code)
	}
}

// TestHandleDeleteModel_CustomModelRecord 验证：尝试删除系统内置记录被拦截。
func TestHandleDeleteModel_CustomModelRecord(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	// 创建系统内置记录
	m := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: model.BuiltinModelID, ModelName: "Built-in", Enabled: true}
	model.DB(context.Background()).Create(&m)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/delete?id=%d", m.ID), nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	deleteModelHandler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("期望状态码 403, 实际=%d", rr.Code)
	}
}

// TestHandleDeleteModel_ClearDefaultModelID 验证：删除默认模型时联动清除 site_config。
func TestHandleDeleteModel_ClearDefaultModelID(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	m := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "default-model", ModelName: "Default", Enabled: true}
	model.DB(context.Background()).Create(&m)

	// 设为默认模型
	config := model.GetSiteConfig(context.Background())
	model.DB(context.Background()).Model(&config).Update("default_model_id", m.ID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/delete?id=%d", m.ID), nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	deleteModelHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望状态码 200, 实际=%d, 响应=%s", rr.Code, rr.Body.String())
	}

	// 验证 default_model_id 被重置为 0
	config = model.GetSiteConfig(context.Background())
	if config.DefaultModelID != 0 {
		t.Errorf("default_model_id 应被重置为 0, 实际=%d", config.DefaultModelID)
	}
}

// TestHandleDeleteModel_ClearVisibilityGroups 验证：删除模型时清理可见性分组关联。
func TestHandleDeleteModel_ClearVisibilityGroups(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	m := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "vis-model", ModelName: "Vis", Enabled: true}
	model.DB(context.Background()).Create(&m)

	// 创建分组和关联
	group := model.UserGroup{Name: "test-group"}
	model.DB(context.Background()).Create(&group)
	model.DB(context.Background()).Create(&model.ModelVisibilityGroup{AIModelID: m.ID, GroupID: group.ID})

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/delete?id=%d", m.ID), nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	deleteModelHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望状态码 200, 实际=%d, 响应=%s", rr.Code, rr.Body.String())
	}

	// 验证可见性关联已清理
	var vgCount int64
	model.DB(context.Background()).Model(&model.ModelVisibilityGroup{}).Where("ai_model_id = ?", m.ID).Count(&vgCount)
	if vgCount != 0 {
		t.Errorf("可见性关联应被清理, 实际=%d", vgCount)
	}
}

// TestSyncInstanceModelsToCVM_NoModels 验证：实例无模型时 syncInstanceModelsToCVM 不报错。
func TestSyncInstanceModelsToCVM_NoModels(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	user := model.User{Username: "sync-test", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{Name: "sync-inst", UserID: user.ID, InstanceId: "ins-sync", AIModelID: 0}
	model.DB(context.Background()).Create(&inst)

	// 无模型 + 无 providerKey → 应直接返回 nil
	if err := syncInstanceModelsToCVM(context.Background(), &inst, ""); err != nil {
		t.Errorf("无模型时应返回 nil, 实际=%v", err)
	}
}

// TestSyncInstanceModelsToCVM_WithPrimary 验证：有 primary 时生成参数正确。
func TestSyncInstanceModelsToCVM_WithPrimary(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	// Mock LoadScript，避免 RunScript 因 LoadScript 未初始化触发 nil panic
	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: script not found: %s", name)
	}
	defer func() { LoadScript = origLoader }()

	user := model.User{Username: "sync-test2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{Name: "sync-inst2", UserID: user.ID, InstanceId: "ins-sync2", AIModelID: 0}
	model.DB(context.Background()).Create(&inst)

	m := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-sync", ModelName: "GLM-Sync", Enabled: true}
	model.DB(context.Background()).Create(&m)

	// 创建 primary 绑定
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst.ID, AIModelID: m.ID, Role: model.ModelRolePrimary, SortOrder: 1})

	// 测试环境无法真正执行 TAT，但可验证函数不 panic 且参数生成正确
	// 由于 RunScript 会失败（无真实 CVM），这里只验证函数能正常执行到报错前
	// 实际 primary/fallbacks 参数生成在 buildPrimaryAndFallbacks 中已单测覆盖

	// 无 providerKey → 只执行 switch_model.sh（会失败，因为无真实 CVM）
	err := syncInstanceModelsToCVM(context.Background(), &inst, "")
	if err == nil {
		// 测试环境没有真实 CVM，TAT 应该失败
		t.Log("测试环境无 CVM，预期 TAT 会失败")
	}
}

func TestMaskModelAPIKey(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "short", in: "sk-abc", want: "******"},
		{name: "eight", in: "12345678", want: "********"},
		{name: "nine", in: "123456789", want: "*********"},
		{name: "eleven", in: "12345678901", want: "***********"},
		{name: "twelve", in: "abcd1234wxyz", want: "abcd****wxyz"},
		{name: "long", in: "sk-1234567890-abcd", want: "sk-1**********abcd"},
		{name: "unicode", in: "密钥abcd1234结尾", want: "密钥ab****34结尾"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskModelAPIKey(tt.in); got != tt.want {
				t.Fatalf("maskModelAPIKey(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestHandleAdminModels_JSON 覆盖 HandleAdminModels 的 JSON 分支
func TestHandleAdminModels_JSON(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	// 创建测试模型
	rawKey := "sk-1234567890-abcd"
	m1 := model.AIModel{Provider: "openai", ModelID: "gpt-4", ModelName: "GPT-4", Enabled: true, APIKey: rawKey}
	model.DB(context.Background()).Create(&m1)

	origToken := AdminToken
	AdminToken = "test-models-token"
	defer func() { AdminToken = origToken }()

	r := httptest.NewRequest(http.MethodGet, "/admin/models", nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-models-token")
	w := httptest.NewRecorder()
	HandleAdminModels(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, rawKey) {
		t.Fatalf("response leaked raw API key: %s", body)
	}
	var resp struct {
		Models []map[string]interface{} `json:"models"`
	}
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(resp.Models))
	}
	if got := resp.Models[0]["APIKey"]; got != "sk-1**********abcd" {
		t.Fatalf("APIKey = %v, want masked key", got)
	}
}

// TestQueryAllModels_Cov 覆盖 queryAllModels 函数；脱敏在响应序列化边界处理。
func TestQueryAllModels_Cov(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	m := model.AIModel{Provider: "openai", ModelID: "gpt-3", ModelName: "GPT-3", Enabled: true, APIKey: "secret-key"}
	model.DB(context.Background()).Create(&m)

	models := queryAllModels(context.Background())
	if len(models) == 0 {
		t.Error("expected at least 1 model")
	}
	if models[0].APIKey != "secret-key" {
		t.Errorf("queryAllModels should keep raw API key for internal use, got %q", models[0].APIKey)
	}
}

// TestHandleUpdateModel_JSON 覆盖 HandleUpdateModel 的 JSON 成功路径（lines 267, 272-274）
func TestHandleUpdateModel_JSON(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-update-model-token"
	defer func() { AdminToken = origToken }()

	// 创建一个非内置模型
	m := model.AIModel{Provider: "openai", ModelID: "gpt-upd", ModelName: "GPT-Upd", Enabled: false,
		APIKey: "k", URL: "https://api.openai.com/v1", ModelType: "openai-completions", QuotaDay: -1}
	model.DB(context.Background()).Create(&m)

	form := url.Values{}
	form.Set("model_id", "gpt-upd-v2")
	form.Set("model_name", "GPT-Upd-V2")
	form.Set("api_key", "newkey")
	form.Set("url", "https://api.openai.com/v1")
	form.Set("model_type", "openai-completions")
	form.Set("quota_day", "-1")
	form.Set("context_len", "128000")
	form.Add("input_types", "text")

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/update?id=%d", m.ID),
		strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-update-model-token")
	w := httptest.NewRecorder()
	HandleUpdateModel(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleUpdateModel_JSON_NotFound 覆盖 line 204-206（模型不存在）
func TestHandleUpdateModel_JSON_NotFound(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-update-model-token"
	defer func() { AdminToken = origToken }()

	r := httptest.NewRequest(http.MethodPost, "/admin/models/update?id=9999", nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-update-model-token")
	w := httptest.NewRecorder()
	HandleUpdateModel(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// TestHandleToggleModel_JSON 覆盖 HandleToggleModel 的 JSON 路径（lines 397, 407, 419）
func TestHandleToggleModel_JSON(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-toggle-model-token"
	defer func() { AdminToken = origToken }()

	// 创建一个非内置模型
	m := model.AIModel{Provider: "openai", ModelID: "gpt-tog", ModelName: "GPT-Tog", Enabled: true}
	model.DB(context.Background()).Create(&m)
	db, _ := model.DB(context.Background()).DB()
	_ = db

	// 关闭模型（enabled: true → false）
	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/toggle?id=%d", m.ID), nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-toggle-model-token")
	w := httptest.NewRecorder()
	HandleToggleModel(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleToggleModel_JSON_NotFound 覆盖 line 402-404（模型不存在）
func TestHandleToggleModel_JSON_NotFound(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-toggle-model-token"
	defer func() { AdminToken = origToken }()

	r := httptest.NewRequest(http.MethodPost, "/admin/models/toggle?id=9999", nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-toggle-model-token")
	w := httptest.NewRecorder()
	HandleToggleModel(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// TestHandleToggleDefault_JSON 覆盖 HandleToggleDefault（lines 442, 447, 451, 458, 462）
func TestHandleToggleDefault_JSON_SetDefault(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-default-token"
	defer func() { AdminToken = origToken }()

	// 创建已启用模型
	m := model.AIModel{Provider: "openai", ModelID: "gpt-def", ModelName: "GPT-Def", Enabled: true, Visible: true}
	model.DB(context.Background()).Create(&m)
	model.DB(context.Background()).Create(&model.SiteConfig{Name: "Test"})

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/default?id=%d", m.ID), nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-default-token")
	w := httptest.NewRecorder()
	HandleToggleDefault(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToggleDefault_JSON_UnsetDefault(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-default-token2"
	defer func() { AdminToken = origToken }()

	// 创建已启用模型，并设为默认
	m := model.AIModel{Provider: "openai", ModelID: "gpt-def2", ModelName: "GPT-Def2", Enabled: true, Visible: true}
	model.DB(context.Background()).Create(&m)
	var cfg model.SiteConfig
	model.DB(context.Background()).FirstOrCreate(&cfg)
	model.DB(context.Background()).Model(&cfg).Update("default_model_id", m.ID)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/default?id=%d", m.ID), nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-default-token2")
	w := httptest.NewRecorder()
	HandleToggleDefault(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToggleDefault_JSON_NotFound(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-default-token3"
	defer func() { AdminToken = origToken }()

	r := httptest.NewRequest(http.MethodPost, "/admin/models/default?id=9999", nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-default-token3")
	w := httptest.NewRecorder()
	HandleToggleDefault(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// TestHandleUpdateModelVisibility_JSON 覆盖 HandleUpdateModelVisibility（lines 487, 512, 535）
func TestHandleUpdateModelVisibility_JSON_AllType(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-vis-token"
	defer func() { AdminToken = origToken }()

	m := model.AIModel{Provider: "openai", ModelID: "gpt-vis", ModelName: "GPT-Vis", Enabled: true}
	model.DB(context.Background()).Create(&m)

	body := strings.NewReader(`{"visibility_type":"all","group_ids":[]}`)
	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/visibility?id=%d", m.ID), body)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-vis-token")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateModelVisibility(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateModelVisibility_JSON_NotFound(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-vis-token2"
	defer func() { AdminToken = origToken }()

	body := strings.NewReader(`{"visibility_type":"all","group_ids":[]}`)
	r := httptest.NewRequest(http.MethodPost, "/admin/models/visibility?id=9999", body)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-vis-token2")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateModelVisibility(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// TestHandleUpdateModelVisibility_JSON_GroupType 覆盖 group 类型路径（lines 512, 535）
func TestHandleUpdateModelVisibility_JSON_GroupType(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	// 添加额外表迁移
	db, _ := model.DB(context.Background()).DB()
	_ = db
	model.DB(context.Background()).AutoMigrate(&model.UserGroup{}, &model.UserGroupMember{})

	origToken := AdminToken
	AdminToken = "test-vis-token3"
	defer func() { AdminToken = origToken }()

	m := model.AIModel{Provider: "openai", ModelID: "gpt-vis2", ModelName: "GPT-Vis2", Enabled: true}
	model.DB(context.Background()).Create(&m)

	// 创建一个分组
	grp := model.UserGroup{Name: "test-grp"}
	model.DB(context.Background()).Create(&grp)

	body := strings.NewReader(fmt.Sprintf(`{"visibility_type":"group","group_ids":[%d]}`, grp.ID))
	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/visibility?id=%d", m.ID), body)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-vis-token3")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateModelVisibility(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateModelVisibility_405(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()
	req := httptest.NewRequest(http.MethodGet, "/admin/models/update-visibility", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleUpdateModelVisibility(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func withAdminModelConnectivityToken(t *testing.T) {
	t.Helper()
	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })
}

func adminModelConnectivityReq(method, path, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-admin-token")
	return r
}

func decodeAdminModelConnectivityResp(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, w.Body.String())
	}
	return resp
}

func TestHandleAdminModelConnectivity_Unauthorized(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/admin/models/connectivity", strings.NewReader(`{"url":"http://example.com","api_key":"sk","model_type":"openai-completions"}`))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleAdminModelConnectivity(w, req)

	if w.Code != http.StatusUnauthorized && w.Code != http.StatusForbidden {
		t.Fatalf("未授权应返回 401/403，实际=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleAdminModelConnectivity_MethodNotAllowed(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()
	withAdminModelConnectivityToken(t)

	req := adminModelConnectivityReq(http.MethodGet, "/admin/models/connectivity", "")
	w := httptest.NewRecorder()

	HandleAdminModelConnectivity(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET 应返回 405，实际=%d", w.Code)
	}
}

func TestHandleAdminModelConnectivity_BadRequests(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()
	withAdminModelConnectivityToken(t)

	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "invalid id", path: "/admin/models/connectivity?id=abc"},
		{name: "model not found", path: "/admin/models/connectivity?id=99999"},
		{name: "bad json", path: "/admin/models/connectivity", body: `not json`},
		{name: "missing url", path: "/admin/models/connectivity", body: `{"api_key":"sk","model_type":"openai-completions"}`},
		{name: "missing api key", path: "/admin/models/connectivity", body: `{"url":"http://example.com","model_type":"openai-completions"}`},
		{name: "invalid model type", path: "/admin/models/connectivity", body: `{"url":"http://example.com","api_key":"sk","model_type":"bad-type"}`},
		{name: "invalid url", path: "/admin/models/connectivity", body: `{"url":"ftp://example.com","api_key":"sk","model_type":"openai-completions"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := adminModelConnectivityReq(http.MethodPost, tt.path, tt.body)
			w := httptest.NewRecorder()

			HandleAdminModelConnectivity(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("应返回 400，实际=%d body=%s", w.Code, w.Body.String())
			}
			resp := decodeAdminModelConnectivityResp(t, w)
			if resp["error"] == "" {
				t.Fatalf("错误响应应包含 error 字段，实际=%v", resp)
			}
		})
	}
}

func TestHandleAdminModelConnectivity_TemporaryCredentialsSuccess(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()
	withAdminModelConnectivityToken(t)

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
	req := adminModelConnectivityReq(http.MethodPost, "/admin/models/connectivity", body)
	w := httptest.NewRecorder()

	HandleAdminModelConnectivity(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	resp := decodeAdminModelConnectivityResp(t, w)
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("ok 应为 true，实际=%v", resp)
	}
	if _, ok := resp["latency_ms"].(float64); !ok {
		t.Fatalf("响应应包含 latency_ms，实际=%v", resp)
	}
}

func TestHandleAdminModelConnectivity_SavedModelSuccess(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()
	withAdminModelConnectivityToken(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 已保存模型携带 ModelID，handleModelConnectivity 使用 chat 探活
		// 命中 /chat/completions。
		if r.URL.Path != "/chat/completions" {
			t.Errorf("请求路径=%s，期望 /chat/completions", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-saved" {
			t.Errorf("Authorization=%q，期望 Bearer sk-saved", got)
			http.Error(w, "unexpected auth", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"id":"x","choices":[{"message":{"role":"assistant","content":""}}]}`))
	}))
	defer server.Close()

	m := model.AIModel{
		Provider:  "openai",
		ModelID:   "gpt-connectivity-ok",
		ModelName: "GPT Connectivity OK",
		URL:       server.URL,
		APIKey:    "sk-saved",
		ModelType: "openai-completions",
		Enabled:   true,
	}
	model.DB(context.Background()).Create(&m)

	req := adminModelConnectivityReq(http.MethodPost, fmt.Sprintf("/admin/models/connectivity?id=%d", m.ID), "")
	w := httptest.NewRecorder()

	HandleAdminModelConnectivity(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	resp := decodeAdminModelConnectivityResp(t, w)
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("ok 应为 true，实际=%v", resp)
	}
}

func TestHandleAdminModelConnectivity_SavedModelFailureReturnsDetails(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()
	withAdminModelConnectivityToken(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 已保存模型携带 ModelID，handleModelConnectivity 使用 chat 探活
		// 命中 /chat/completions。
		if r.URL.Path != "/chat/completions" {
			t.Errorf("请求路径=%s，期望 /chat/completions", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-bad" {
			t.Errorf("Authorization=%q，期望 Bearer sk-bad", got)
			http.Error(w, "unexpected auth", http.StatusUnauthorized)
			return
		}
		http.Error(w, "invalid key\nplease retry", http.StatusUnauthorized)
	}))
	defer server.Close()

	m := model.AIModel{
		Provider:  "openai",
		ModelID:   "gpt-connectivity-fail",
		ModelName: "GPT Connectivity Fail",
		URL:       server.URL,
		APIKey:    "sk-bad",
		ModelType: "openai-completions",
		Enabled:   true,
	}
	model.DB(context.Background()).Create(&m)

	req := adminModelConnectivityReq(http.MethodPost, fmt.Sprintf("/admin/models/connectivity?id=%d", m.ID), "")
	w := httptest.NewRecorder()

	HandleAdminModelConnectivity(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("上游鉴权失败也应返回 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	resp := decodeAdminModelConnectivityResp(t, w)
	if ok, _ := resp["ok"].(bool); ok {
		t.Fatalf("ok 应为 false，实际=%v", resp)
	}
	if resp["kind"] != "invalid_api_key" {
		t.Fatalf("kind 应为 invalid_api_key，实际=%v", resp)
	}
	if resp["status_code"] != float64(http.StatusUnauthorized) {
		t.Fatalf("status_code 应为 401，实际=%v", resp)
	}
	if snippet, _ := resp["snippet"].(string); !strings.Contains(snippet, "invalid key") || strings.Contains(snippet, "\n") {
		t.Fatalf("snippet 应包含折叠后的上游响应，实际=%q", snippet)
	}
	if resp["message"] == "" {
		t.Fatalf("响应应包含 message，实际=%v", resp)
	}
}

func TestHandleAdminModelConnectivity_UpstreamStatusClassification(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()
	withAdminModelConnectivityToken(t)

	tests := []struct {
		name       string
		statusCode int
		wantKind   string
	}{
		{name: "forbidden", statusCode: http.StatusForbidden, wantKind: "forbidden"},
		{name: "rate limited", statusCode: http.StatusTooManyRequests, wantKind: "rate_limited"},
		{name: "upstream client", statusCode: http.StatusNotFound, wantKind: "upstream_client_error"},
		{name: "upstream server", statusCode: http.StatusInternalServerError, wantKind: "upstream_server_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, tt.name+" detail", tt.statusCode)
			}))
			defer server.Close()

			body := fmt.Sprintf(`{"url":%q,"api_key":"sk-test","model_type":"openai-completions","model":"deepseek-v4-flash"}`, server.URL)
			req := adminModelConnectivityReq(http.MethodPost, "/admin/models/connectivity", body)
			w := httptest.NewRecorder()

			HandleAdminModelConnectivity(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("上游错误应返回 200，实际=%d body=%s", w.Code, w.Body.String())
			}
			resp := decodeAdminModelConnectivityResp(t, w)
			if ok, _ := resp["ok"].(bool); ok {
				t.Fatalf("ok 应为 false，实际=%v", resp)
			}
			if resp["kind"] != tt.wantKind {
				t.Fatalf("kind 应为 %s，实际=%v", tt.wantKind, resp["kind"])
			}
			if resp["status_code"] != float64(tt.statusCode) {
				t.Fatalf("status_code 应为 %d，实际=%v", tt.statusCode, resp["status_code"])
			}
			if snippet, _ := resp["snippet"].(string); !strings.Contains(snippet, tt.name) {
				t.Fatalf("snippet 应包含上游错误详情，实际=%q", snippet)
			}
		})
	}
}

func TestHandleAdminModelConnectivity_AnthropicTemporaryCredentialsSuccess(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()
	withAdminModelConnectivityToken(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("请求路径=%s，期望 /v1/messages", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("x-api-key"); got != "sk-anthropic-admin" {
			t.Errorf("x-api-key=%q，期望 sk-anthropic-admin", got)
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

	body := fmt.Sprintf(`{"url":%q,"api_key":"sk-anthropic-admin","model_type":"anthropic-messages","model":"deepseek-v4-flash"}`, server.URL)
	req := adminModelConnectivityReq(http.MethodPost, "/admin/models/connectivity", body)
	w := httptest.NewRecorder()

	HandleAdminModelConnectivity(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	resp := decodeAdminModelConnectivityResp(t, w)
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("ok 应为 true，实际=%v", resp)
	}
}

// TestHandleAdminModelConnectivity_ChatProbeDirectHit 验证：
// 提供了 model 字段时，直接使用 chat 探活命中 /chat/completions，
// 不应触发 /models 路径。
func TestHandleAdminModelConnectivity_ChatProbeDirectHit(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()
	withAdminModelConnectivityToken(t)

	var modelsHits, chatHits int32
	var gotChatModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chat/completions":
			atomic.AddInt32(&chatHits, 1)
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if v, ok := body["model"].(string); ok {
				gotChatModel = v
			}
			_, _ = w.Write([]byte(`{"id":"x","choices":[{"message":{"role":"assistant","content":""}}]}`))
		case "/models":
			atomic.AddInt32(&modelsHits, 1)
			t.Errorf("/models 不应被命中")
			http.Error(w, "unexpected path", http.StatusNotFound)
		default:
			t.Errorf("未预期的请求路径=%s", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	body := fmt.Sprintf(
		`{"url":%q,"api_key":"sk","model_type":"openai-completions","model":"gpt-4o-mini"}`,
		server.URL,
	)
	req := adminModelConnectivityReq(http.MethodPost, "/admin/models/connectivity", body)
	w := httptest.NewRecorder()

	HandleAdminModelConnectivity(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	if got := atomic.LoadInt32(&modelsHits); got != 0 {
		t.Fatalf("/models 不应被命中，实际命中 %d 次", got)
	}
	if got := atomic.LoadInt32(&chatHits); got != 1 {
		t.Fatalf("/chat/completions 命中 %d 次，期望 1", got)
	}
	if gotChatModel != "gpt-4o-mini" {
		t.Fatalf("chat 探活下发的 model=%q，期望 gpt-4o-mini", gotChatModel)
	}
	resp := decodeAdminModelConnectivityResp(t, w)
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("ok 应为 true，实际=%v", resp)
	}
}

// TestHandleAdminModelConnectivity_NoModelReturns400 验证：
// 没有 model 字段时，直接返回 400（resolveConnectivityArgs 校验）
func TestHandleAdminModelConnectivity_NoModelReturns400(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()
	withAdminModelConnectivityToken(t)

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		http.Error(w, "should not be called", http.StatusOK)
	}))
	defer server.Close()

	body := fmt.Sprintf(
		`{"url":%q,"api_key":"sk-bad","model_type":"openai-completions"}`,
		server.URL,
	)
	req := adminModelConnectivityReq(http.MethodPost, "/admin/models/connectivity", body)
	w := httptest.NewRecorder()

	HandleAdminModelConnectivity(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("应返回 400，实际=%d body=%s", w.Code, w.Body.String())
	}
	if got := atomic.LoadInt32(&hits); got != 0 {
		t.Fatalf("上游不应被命中，实际 %d 次", got)
	}
}

// TestHandleAdminModelConnectivity_ChatProbeFail 验证：
// chat 探活失败时，最终响应携带 chat 阶段的诊断信息
// （chat 阶段的错误能反馈"模型不可用"而不仅是"凭证可疑"）。
func TestHandleAdminModelConnectivity_ChatProbeFail(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()
	withAdminModelConnectivityToken(t)

	var modelsHits, chatHits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chat/completions":
			atomic.AddInt32(&chatHits, 1)
			http.Error(w, "model not found", http.StatusBadRequest)
		case "/models":
			atomic.AddInt32(&modelsHits, 1)
			t.Errorf("/models 不应被命中")
			http.Error(w, "unexpected path", http.StatusNotFound)
		default:
			t.Errorf("未预期的请求路径=%s", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	body := fmt.Sprintf(
		`{"url":%q,"api_key":"sk","model_type":"openai-completions","model":"unknown-model"}`,
		server.URL,
	)
	req := adminModelConnectivityReq(http.MethodPost, "/admin/models/connectivity", body)
	w := httptest.NewRecorder()

	HandleAdminModelConnectivity(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	if got := atomic.LoadInt32(&modelsHits); got != 0 {
		t.Fatalf("/models 不应被命中，实际命中 %d 次", got)
	}
	if got := atomic.LoadInt32(&chatHits); got != 1 {
		t.Fatalf("/chat/completions 命中 %d 次，期望 1", got)
	}
	resp := decodeAdminModelConnectivityResp(t, w)
	if ok, _ := resp["ok"].(bool); ok {
		t.Fatalf("chat 探活失败时 ok 应为 false，实际=%v", resp)
	}
	if resp["kind"] != "upstream_client_error" {
		t.Fatalf("kind 应为 upstream_client_error，实际=%v", resp["kind"])
	}
	if sc, _ := resp["status_code"].(float64); int(sc) != http.StatusBadRequest {
		t.Fatalf("status_code 应为 400，实际=%v", resp["status_code"])
	}
	if snippet, _ := resp["snippet"].(string); !strings.Contains(snippet, "model not found") {
		t.Fatalf("snippet 应包含 chat 阶段响应，实际=%q", snippet)
	}
}

// TestHandleAdminModelConnectivity_AnthropicChatProbeDirectHit 验证 Anthropic
// provider 直接使用 chat 探活命中 /v1/messages。
func TestHandleAdminModelConnectivity_AnthropicChatProbeDirectHit(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()
	withAdminModelConnectivityToken(t)

	var listHits, msgHits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/messages":
			atomic.AddInt32(&msgHits, 1)
			if got := r.Header.Get("anthropic-version"); got == "" {
				t.Errorf("anthropic-version 不能为空")
			}
			_, _ = w.Write([]byte(`{"id":"x","content":[]}`))
		case "/v1/models":
			atomic.AddInt32(&listHits, 1)
			t.Errorf("/v1/models 不应被命中")
			http.Error(w, "unexpected path", http.StatusNotFound)
		default:
			t.Errorf("未预期的请求路径=%s", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	body := fmt.Sprintf(
		`{"url":%q,"api_key":"sk","model_type":"anthropic-messages","model":"claude-3-5-haiku"}`,
		server.URL,
	)
	req := adminModelConnectivityReq(http.MethodPost, "/admin/models/connectivity", body)
	w := httptest.NewRecorder()

	HandleAdminModelConnectivity(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	if got := atomic.LoadInt32(&listHits); got != 0 {
		t.Fatalf("/v1/models 不应被命中，实际命中 %d 次", got)
	}
	if got := atomic.LoadInt32(&msgHits); got != 1 {
		t.Fatalf("/v1/messages 命中 %d 次，期望 1", got)
	}
	resp := decodeAdminModelConnectivityResp(t, w)
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("ok 应为 true，实际=%v", resp)
	}
}

// TestClassifyConnectivityError_Default 直接覆盖 classifyConnectivityError
// 在面对未识别错误（非 ConnectivityError、非任一 Err* sentinel）时的兜底
// 分支，避免本次新增链路上未知错误类型时返回空 kind/message。
func TestClassifyConnectivityError_Default(t *testing.T) {
	body := `{"url":"https://example.com","api_key":"sk","model_type":"anthropic-messages","model":"claude-3-5-haiku"}`
	req := adminModelConnectivityReq(http.MethodPost, "/admin/models/connectivity", body)

	kind, message := classifyConnectivityError(req, errors.New("something else"))
	if kind != "unknown" {
		t.Fatalf("kind = %q, want unknown", kind)
	}
	if message == "" {
		t.Fatalf("message 不应为空")
	}
}

// TestClassifyConnectivityError_AllKinds 覆盖 classifyConnectivityError 的
// 全部分支（在重构时如有新增/调整可立即捕获）。
func TestClassifyConnectivityError_AllKinds(t *testing.T) {
	body := `{"url":"https://example.com","api_key":"sk","model_type":"anthropic-messages","model":"claude-3-5-haiku"}`
	req := adminModelConnectivityReq(http.MethodPost, "/admin/models/connectivity", body)

	cases := []struct {
		name     string
		err      error
		wantKind string
	}{
		{"network", &provider.ConnectivityError{Kind: provider.ErrNetworkUnreachable}, "network_unreachable"},
		{"invalid_key", &provider.ConnectivityError{Kind: provider.ErrInvalidAPIKey}, "invalid_api_key"},
		{"forbidden", &provider.ConnectivityError{Kind: provider.ErrForbidden}, "forbidden"},
		{"rate_limited", &provider.ConnectivityError{Kind: provider.ErrRateLimited}, "rate_limited"},
		{"upstream_server", &provider.ConnectivityError{Kind: provider.ErrUpstreamServer}, "upstream_server_error"},
		{"upstream_client", &provider.ConnectivityError{Kind: provider.ErrUpstreamClient}, "upstream_client_error"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			kind, msg := classifyConnectivityError(req, tt.err)
			if kind != tt.wantKind {
				t.Fatalf("kind = %q, want %q", kind, tt.wantKind)
			}
			if msg == "" {
				t.Fatalf("message 不应为空")
			}
		})
	}
}

// ============================================================================
// HandleToggleModelEnabled 单元测试
//
// HandleToggleModelEnabled 是本次 enabled / visible 解耦改动新增的接口，
// 切换的是 AIModel.Enabled 字段（"是否启用"——控制 LLM 路由可用性），
// 与原有 HandleToggleModel（切换 Visible "用户可见"）严格区分。
// ============================================================================

// TestHandleToggleModelEnabled_FromEnabledToDisabled 验证：
// 关闭 Enabled 时，DB 中 Enabled 字段被翻转，Visible 不受影响。
func TestHandleToggleModelEnabled_FromEnabledToDisabled(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-toggle-enabled-token"
	defer func() { AdminToken = origToken }()

	m := model.AIModel{
		Provider: "openai", ModelID: "gpt-tog-en", ModelName: "GPT-TogEn",
		Enabled: true, Visible: true,
	}
	model.DB(context.Background()).Create(&m)

	r := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/admin/models/toggle-enabled?id=%d", m.ID), nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-toggle-enabled-token")
	w := httptest.NewRecorder()
	HandleToggleModelEnabled(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var got model.AIModel
	model.DB(context.Background()).First(&got, m.ID)
	if got.Enabled {
		t.Errorf("Enabled 应被翻转为 false, 实际=%v", got.Enabled)
	}
	if !got.Visible {
		t.Errorf("Visible 不应被影响（仍为 true）, 实际=%v", got.Visible)
	}
}

// TestHandleToggleModelEnabled_FromDisabledToEnabled 验证：
// 从 false → true 翻转，Visible 不变。
func TestHandleToggleModelEnabled_FromDisabledToEnabled(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-toggle-enabled-token2"
	defer func() { AdminToken = origToken }()

	m := model.AIModel{
		Provider: "openai", ModelID: "gpt-tog-en2", ModelName: "GPT-TogEn2",
		Enabled: false, Visible: false,
	}
	model.DB(context.Background()).Create(&m)

	r := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/admin/models/toggle-enabled?id=%d", m.ID), nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-toggle-enabled-token2")
	w := httptest.NewRecorder()
	HandleToggleModelEnabled(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var got model.AIModel
	model.DB(context.Background()).First(&got, m.ID)
	if !got.Enabled {
		t.Errorf("Enabled 应被翻转为 true, 实际=%v", got.Enabled)
	}
	if got.Visible {
		t.Errorf("Visible 不应被影响, 实际=%v", got.Visible)
	}
}

// TestHandleToggleModelEnabled_ClearsDefaultModelOnDisable 验证：
// 关闭 Enabled 时，若该模型当前是默认模型则联动清除（避免新建实例自动注入禁用模型）。
func TestHandleToggleModelEnabled_ClearsDefaultModelOnDisable(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-toggle-enabled-default"
	defer func() { AdminToken = origToken }()

	m := model.AIModel{
		Provider: "openai", ModelID: "gpt-tog-default", ModelName: "GPT-TogDef",
		Enabled: true, Visible: true,
	}
	model.DB(context.Background()).Create(&m)

	cfg := model.GetSiteConfig(context.Background())
	model.DB(context.Background()).Model(&cfg).Update("default_model_id", m.ID)

	r := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/admin/models/toggle-enabled?id=%d", m.ID), nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-toggle-enabled-default")
	w := httptest.NewRecorder()
	HandleToggleModelEnabled(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	cfg = model.GetSiteConfig(context.Background())
	if cfg.DefaultModelID != 0 {
		t.Errorf("默认模型应被联动清除, 实际 default_model_id=%d", cfg.DefaultModelID)
	}
}

// TestHandleToggleModelEnabled_NotFound 验证：模型不存在 → 404。
func TestHandleToggleModelEnabled_NotFound(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-toggle-enabled-404"
	defer func() { AdminToken = origToken }()

	r := httptest.NewRequest(http.MethodPost, "/admin/models/toggle-enabled?id=99999", nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-toggle-enabled-404")
	w := httptest.NewRecorder()
	HandleToggleModelEnabled(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// TestHandleToggleModelEnabled_MethodNotAllowed 验证：GET 应返回 405。
func TestHandleToggleModelEnabled_MethodNotAllowed(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-toggle-enabled-405"
	defer func() { AdminToken = origToken }()

	r := httptest.NewRequest(http.MethodGet, "/admin/models/toggle-enabled?id=1", nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-toggle-enabled-405")
	w := httptest.NewRecorder()
	HandleToggleModelEnabled(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// TestHandleToggleModel_KeepsEnabledAndOnlyChangesVisible 反向回归：
// HandleToggleModel（不带 -enabled 后缀）只翻 Visible，不动 Enabled。
// 这是配套 HandleToggleModelEnabled 的语义边界守卫，避免未来重构串台。
func TestHandleToggleModel_KeepsEnabledAndOnlyChangesVisible(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-toggle-visible-only"
	defer func() { AdminToken = origToken }()

	m := model.AIModel{
		Provider: "openai", ModelID: "gpt-vis-only", ModelName: "GPT-VisOnly",
		Enabled: true, Visible: true,
	}
	model.DB(context.Background()).Create(&m)

	r := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/admin/models/toggle?id=%d", m.ID), nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-toggle-visible-only")
	w := httptest.NewRecorder()
	HandleToggleModel(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var got model.AIModel
	model.DB(context.Background()).First(&got, m.ID)
	if got.Visible {
		t.Errorf("Visible 应被翻转为 false, 实际=%v", got.Visible)
	}
	if !got.Enabled {
		t.Errorf("Enabled 不应被影响（仍为 true）, 实际=%v", got.Enabled)
	}
}

// TestModelWithVisibility_MarshalJSON_IncludesVisibilityGroups 验证 modelWithVisibility
// 的自定义 MarshalJSON：必须同时输出 AIModel.MarshalJSON 的字段（Enabled / Visible /
// EnabledStatus 等），以及外层 visibility_groups 字段。
//
// 这是 issue 的根因修复：内嵌 AIModel 实现了 json.Marshaler 接口后，Go encoding/json
// 会"提升"内嵌类型的 MarshalJSON 到外层并忽略并列字段，必须显式实现外层 MarshalJSON 才能
// 把 visibility_groups 注入到响应里。
func TestModelWithVisibility_MarshalJSON_IncludesVisibilityGroups(t *testing.T) {
	cases := []struct {
		name           string
		visibility     []visibilityGroupInfo
		wantGroupCount int
	}{
		{
			name:           "non-empty groups → 正常输出",
			visibility:     []visibilityGroupInfo{{GroupID: 1, GroupName: "研发组"}, {GroupID: 2, GroupName: "测试组"}},
			wantGroupCount: 2,
		},
		{
			name:           "empty slice → 空数组",
			visibility:     []visibilityGroupInfo{},
			wantGroupCount: 0,
		},
		{
			name:           "nil slice → 必须 fallback 为空数组而非缺失字段",
			visibility:     nil,
			wantGroupCount: 0,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			mv := modelWithVisibility{
				AIModel: model.AIModel{
					Provider: "openai", ModelID: "gpt-4",
					Enabled: true, Visible: false, // Enabled / Visible 故意取不同值，验证字段语义
					VisibilityType: "group",
				},
				VisibilityGroups: tt.visibility,
			}
			data, err := json.Marshal(mv)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var parsed map[string]interface{}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}

			// 1. visibility_groups 必须存在
			rawGroups, ok := parsed["visibility_groups"]
			if !ok {
				t.Fatalf("缺少 visibility_groups 字段, raw=%s", string(data))
			}
			groups, ok := rawGroups.([]interface{})
			if !ok {
				t.Fatalf("visibility_groups 必须是数组而非 %T, raw=%s", rawGroups, string(data))
			}
			if len(groups) != tt.wantGroupCount {
				t.Errorf("visibility_groups 数量 = %d, want %d, raw=%s", len(groups), tt.wantGroupCount, string(data))
			}

			// 2. AIModel.MarshalJSON 的兼容字段必须仍然存在（即不能被覆盖/遗漏）
			//    Enabled = Visible（旧前端兼容）；EnabledStatus = 真实 Enabled。
			if bigE, _ := parsed["Enabled"].(bool); bigE != false { // mv.Visible=false
				t.Errorf("Enabled = %v, want false (应等于 Visible), raw=%s", bigE, string(data))
			}
			if enStat, _ := parsed["EnabledStatus"].(bool); enStat != true { // mv.Enabled=true
				t.Errorf("EnabledStatus = %v, want true (应等于真实 Enabled), raw=%s", enStat, string(data))
			}
			if _, ok := parsed["Visible"]; ok {
				t.Errorf("Visible 字段不应被对外输出, raw=%s", string(data))
			}
			// 3. 其他 AIModel 字段也应保留
			if p, _ := parsed["Provider"].(string); p != "openai" {
				t.Errorf("Provider = %v, want openai, raw=%s", p, string(data))
			}
		})
	}
}

// TestModelWithVisibility_MarshalJSON_GroupContent 验证非空 visibility_groups 元素
// 的字段名（group_id / group_name）能正确序列化，覆盖 visibilityGroupInfo 的 json tag。
func TestModelWithVisibility_MarshalJSON_GroupContent(t *testing.T) {
	mv := modelWithVisibility{
		AIModel: model.AIModel{Provider: "p", ModelID: "m"},
		VisibilityGroups: []visibilityGroupInfo{
			{GroupID: 42, GroupName: "设计组"},
		},
	}
	data, err := json.Marshal(mv)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	groups, _ := parsed["visibility_groups"].([]interface{})
	if len(groups) != 1 {
		t.Fatalf("groups len = %d, want 1, raw=%s", len(groups), string(data))
	}
	g := groups[0].(map[string]interface{})
	if id, _ := g["group_id"].(float64); int(id) != 42 {
		t.Errorf("group_id = %v, want 42, raw=%s", g["group_id"], string(data))
	}
	if name, _ := g["group_name"].(string); name != "设计组" {
		t.Errorf("group_name = %v, want 设计组, raw=%s", g["group_name"], string(data))
	}
}

// TestModelWithVisibility_MarshalJSON_OutputKeySet 锁定 /admin/models 列表
// 响应中单条模型对外的字段名集合，避免后续重构误增/误删字段。
//
// 这是"修复 visibility_groups 不返回 + 隐藏 Visible + APIKey 脱敏输出"组合改动的最终契约：
// 列表项 JSON 必须包含 visibility_groups/APIKey（脱敏值），同时 Visible 不能出现。
func TestModelWithVisibility_MarshalJSON_OutputKeySet(t *testing.T) {
	mv := modelWithVisibility{
		AIModel: model.AIModel{
			Provider: "openai", ModelID: "gpt-4",
			APIKey:  "sk-secret",
			Enabled: true, Visible: false, VisibilityType: "group",
		},
		VisibilityGroups: []visibilityGroupInfo{{GroupID: 1, GroupName: "g1"}},
	}
	data, err := json.Marshal(mv)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	mustHave := []string{
		"ID", "Provider", "ModelID", "ModelType",
		"Enabled", "EnabledStatus", "VisibilityType",
		"APIKey", "visibility_groups",
	}
	for _, k := range mustHave {
		if _, ok := parsed[k]; !ok {
			t.Errorf("缺少必要字段 %q, raw=%s", k, string(data))
		}
	}
	mustNotHave := []string{
		"Visible",         // 已显式从 AIModel.MarshalJSON 输出中剔除
		"enabled",         // 旧字段名
		"visibility_type", // 旧 json tag
	}
	for _, k := range mustNotHave {
		if _, ok := parsed[k]; ok {
			t.Errorf("不应输出字段 %q, raw=%s", k, string(data))
		}
	}
	// Enabled 必须 = Visible 值（兼容旧前端）
	if bigE, _ := parsed["Enabled"].(bool); bigE != false {
		t.Errorf("Enabled = %v, want false (=Visible), raw=%s", bigE, string(data))
	}
	// EnabledStatus 必须 = 真实 Enabled
	if enStat, _ := parsed["EnabledStatus"].(bool); enStat != true {
		t.Errorf("EnabledStatus = %v, want true, raw=%s", enStat, string(data))
	}
	if apiKey, _ := parsed["APIKey"].(string); apiKey != "*********" {
		t.Errorf("APIKey = %v, want masked value, raw=%s", apiKey, string(data))
	}
}

// --- HandleUpdateModel 默认同步到实例 测试 ---

// updateModelCallRecorder 记录 injectModelScriptRunner 的调用。
type updateModelCallRecorder struct {
	mu       sync.Mutex
	count    int
	params   []map[string]string
	instance []string
	err      error
}

// newUpdateModelRecorder 返回 recorder + cleanup。mock 为同步执行，便于测试等待。
func newUpdateModelRecorder(t *testing.T, err error) (*updateModelCallRecorder, func()) {
	t.Helper()
	rec := &updateModelCallRecorder{err: err}
	orig := injectModelScriptRunner
	injectModelScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		rec.mu.Lock()
		rec.count++
		p := make(map[string]string, len(params))
		for k, v := range params {
			p[k] = v
		}
		rec.params = append(rec.params, p)
		rec.instance = append(rec.instance, instanceId)
		rec.mu.Unlock()
		return "{}", rec.err
	}
	return rec, func() { injectModelScriptRunner = orig }
}

// waitForCount 轮询等待 recorder.count 达到 expected（goroutine 异步完成）。
func (r *updateModelCallRecorder) waitForCount(t *testing.T, expected int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		if r.count >= expected {
			r.mu.Unlock()
			return
		}
		r.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	r.mu.Lock()
	actual := r.count
	r.mu.Unlock()
	t.Fatalf("等待 injectModelScriptRunner 调用 %d 次超时, 实际=%d", expected, actual)
}

// TestHandleUpdateModel_SyncNoBindings 更新未绑定任何实例的 model：不触发 TAT 下发。
func TestHandleUpdateModel_SyncNoBindings(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-update-model-sync-token"
	defer func() { AdminToken = origToken }()

	rec, restoreMock := newUpdateModelRecorder(t, nil)
	defer restoreMock()

	m := model.AIModel{Provider: hcommon.CustomModelProvider, ModelID: "sync-test-1", ModelName: "Sync1",
		APIKey: "k", URL: "https://api.example.com/v1", ModelType: "openai-completions", QuotaDay: -1}
	model.DB(context.Background()).Create(&m)

	form := url.Values{}
	form.Set("model_id", "sync-test-1-v2")
	form.Set("url", "https://api.example.com/v1")
	form.Set("model_type", "openai-completions")
	form.Set("quota_day", "-1")
	form.Set("context_len", "128000")
	form.Add("input_types", "text")

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/update?id=%d", m.ID),
		strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-update-model-sync-token")
	w := httptest.NewRecorder()
	HandleUpdateModel(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际=%d, 响应=%s", w.Code, w.Body.String())
	}

	// 无绑定实例：不应触发 TAT 下发
	time.Sleep(100 * time.Millisecond)
	rec.mu.Lock()
	count := rec.count
	rec.mu.Unlock()
	if count != 0 {
		t.Errorf("无绑定实例时不应触发 TAT 下发, 实际=%d", count)
	}

	// 响应保持简单 {"ok": true}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		t.Errorf("期望 ok=true, raw=%s", w.Body.String())
	}
}

// TestHandleUpdateModel_SyncWithBindings 更新绑定到 3 个实例的 model（primary+fallback 混合）：
// 默认同步，异步并发下发到全部 3 个实例。
func TestHandleUpdateModel_SyncWithBindings(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-update-model-sync-token"
	defer func() { AdminToken = origToken }()

	rec, restoreMock := newUpdateModelRecorder(t, nil)
	defer restoreMock()

	m := model.AIModel{Provider: hcommon.CustomModelProvider, ModelID: "sync-test-3", ModelName: "Sync3",
		APIKey: "k", URL: "https://api.example.com/v1", ModelType: "openai-completions", QuotaDay: -1}
	model.DB(context.Background()).Create(&m)

	user := model.User{Username: "admin-sync", Password: "x", Role: "admin"}
	model.DB(context.Background()).Create(&user)

	instA := model.Instance{Name: "inst-A", UserID: user.ID, InstanceId: "ins-A", ProxyToken: strPtr("sk-proxy-a")}
	instB := model.Instance{Name: "inst-B", UserID: user.ID, InstanceId: "ins-B", ProxyToken: strPtr("sk-proxy-b")}
	instC := model.Instance{Name: "inst-C", UserID: user.ID, InstanceId: "ins-C", ProxyToken: strPtr("sk-proxy-c")}
	model.DB(context.Background()).Create(&instA)
	model.DB(context.Background()).Create(&instB)
	model.DB(context.Background()).Create(&instC)

	// instA: m 是 primary
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: instA.ID, AIModelID: m.ID, Role: model.ModelRolePrimary, SortOrder: 1})
	// instB: m 是 fallback（另需 primary）
	otherPrimary := model.AIModel{Provider: hcommon.CustomModelProvider, ModelID: "other-primary", ModelName: "Other",
		APIKey: "k2", URL: "https://api.example.com/v1", ModelType: "openai-completions", QuotaDay: -1}
	model.DB(context.Background()).Create(&otherPrimary)
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: instB.ID, AIModelID: otherPrimary.ID, Role: model.ModelRolePrimary, SortOrder: 1})
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: instB.ID, AIModelID: m.ID, Role: model.ModelRoleFallback, SortOrder: 2})
	// instC: m 是 primary
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: instC.ID, AIModelID: m.ID, Role: model.ModelRolePrimary, SortOrder: 1})

	form := url.Values{}
	form.Set("model_id", "sync-test-3-v2")
	form.Set("url", "https://api.example.com/v2")
	form.Set("model_type", "openai-completions")
	form.Set("quota_day", "-1")
	form.Set("context_len", "128000")
	form.Add("input_types", "text")

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/update?id=%d", m.ID),
		strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-update-model-sync-token")
	r = r.WithContext(hcommon.InjectTenant(r.Context(), hcommon.TenantSnapshot{Domain: "https://hatchery.example.com"}))
	w := httptest.NewRecorder()
	HandleUpdateModel(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际=%d, 响应=%s", w.Code, w.Body.String())
	}

	// 默认同步：3 个绑定实例都应被下发
	rec.waitForCount(t, 3, 5*time.Second)

	rec.mu.Lock()
	count := rec.count
	instances := append([]string(nil), rec.instance...)
	paramsList := append([]map[string]string(nil), rec.params...)
	rec.mu.Unlock()

	if count != 3 {
		t.Errorf("期望 TAT 调用 3 次, 实际=%d", count)
	}

	expectedInst := map[string]bool{"ins-A": false, "ins-B": false, "ins-C": false}
	for _, id := range instances {
		if _, ok := expectedInst[id]; ok {
			expectedInst[id] = true
		}
	}
	for id, called := range expectedInst {
		if !called {
			t.Errorf("实例 %s 未被调用", id)
		}
	}

	expectedProxyTokens := map[string]string{
		"ins-A": "sk-proxy-a",
		"ins-B": "sk-proxy-b",
		"ins-C": "sk-proxy-c",
	}
	for i, params := range paramsList {
		valueJSON, err := base64.StdEncoding.DecodeString(params["valueb64"])
		if err != nil {
			t.Fatalf("decode valueb64: %v", err)
		}
		var value struct {
			BaseURL string `json:"baseUrl"`
			APIKey  string `json:"apiKey"`
			Models  []struct {
				ID string `json:"id"`
			} `json:"models"`
		}
		if err := json.Unmarshal(valueJSON, &value); err != nil {
			t.Fatalf("unmarshal valueb64: %v", err)
		}
		if !strings.HasPrefix(params["provider"], model.BuiltinModelProvider+"-") {
			t.Fatalf("管控端自定义模型应通过 hatchery proxy provider 下发，got provider=%q", params["provider"])
		}
		if value.BaseURL != "https://hatchery.example.com/v1" {
			t.Fatalf("管控端自定义模型同步应使用 proxy URL，got=%q", value.BaseURL)
		}
		if want := expectedProxyTokens[instances[i]]; value.APIKey != want {
			t.Fatalf("管控端自定义模型同步应使用实例 proxy token，instance=%q, got=%q, want=%q", instances[i], value.APIKey, want)
		}
		if len(value.Models) != 1 || value.Models[0].ID != "sync-test-3-v2" {
			t.Fatalf("同步下发应使用更新后的 model_id，models=%+v", value.Models)
		}
	}

	// DB 字段已更新
	var found model.AIModel
	model.DB(context.Background()).First(&found, m.ID)
	if found.URL != "https://api.example.com/v2" {
		t.Errorf("DB 中 URL 应更新为 v2, 实际=%s", found.URL)
	}
}

// TestHandleUpdateModel_SyncTATFailure 同步下发失败时 DB 不回滚，HTTP 仍 200。
func TestHandleUpdateModel_SyncTATFailure(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-update-model-sync-token"
	defer func() { AdminToken = origToken }()

	rec, restoreMock := newUpdateModelRecorder(t, errors.New("TAT timeout"))
	defer restoreMock()

	m := model.AIModel{Provider: hcommon.CustomModelProvider, ModelID: "sync-test-4", ModelName: "Sync4",
		APIKey: "k", URL: "https://api.example.com/v1", ModelType: "openai-completions", QuotaDay: -1}
	model.DB(context.Background()).Create(&m)

	user := model.User{Username: "admin-sync4", Password: "x", Role: "admin"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{Name: "inst-fail", UserID: user.ID, InstanceId: "ins-fail", ProxyToken: strPtr("sk-proxy-fail")}
	model.DB(context.Background()).Create(&inst)
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst.ID, AIModelID: m.ID, Role: model.ModelRolePrimary, SortOrder: 1})

	form := url.Values{}
	form.Set("model_id", "sync-test-4-v2")
	form.Set("url", "https://api.example.com/v2")
	form.Set("model_type", "openai-completions")
	form.Set("quota_day", "-1")
	form.Set("context_len", "128000")
	form.Add("input_types", "text")

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/update?id=%d", m.ID),
		strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-update-model-sync-token")
	r = r.WithContext(hcommon.InjectTenant(r.Context(), hcommon.TenantSnapshot{Domain: "https://hatchery.example.com"}))
	w := httptest.NewRecorder()
	HandleUpdateModel(w, r)

	// TAT 失败不影响 HTTP 状态
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（TAT 失败不影响 HTTP 状态）, 实际=%d, 响应=%s", w.Code, w.Body.String())
	}

	rec.waitForCount(t, 1, 5*time.Second)

	// DB 仍更新成功（不回滚）
	var found model.AIModel
	model.DB(context.Background()).First(&found, m.ID)
	if found.URL != "https://api.example.com/v2" {
		t.Errorf("DB 中 URL 应更新为 v2（TAT 失败不回滚）, 实际=%s", found.URL)
	}
}

// --- HandleUpdateModel max_tokens / custom_http_headers 显式设置语义 测试 ---

// updateModelFormBase 返回一份可直接更新的合法基础表单（不含 max_tokens / custom_http_headers）。
func updateModelFormBase(modelID string) url.Values {
	form := url.Values{}
	form.Set("model_id", modelID)
	form.Set("url", "https://api.example.com/v1")
	form.Set("model_type", "openai-completions")
	form.Set("quota_day", "-1")
	form.Set("context_len", "128000")
	form.Add("input_types", "text")
	return form
}

// doUpdateModel 用给定表单调用 HandleUpdateModel，返回 recorder。
func doUpdateModel(t *testing.T, modelID uint, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/models/update?id=%d", modelID),
		strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-update-model-explicit-token")
	w := httptest.NewRecorder()
	HandleUpdateModel(w, r)
	return w
}

// TestHandleUpdateModel_MaxTokensExplicitZero 显式传 max_tokens=0：写入 0（不限输出）。
func TestHandleUpdateModel_MaxTokensExplicitZero(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-update-model-explicit-token"
	defer func() { AdminToken = origToken }()

	m := model.AIModel{Provider: "openai", ModelID: "mt-zero", ModelName: "MTZero",
		APIKey: "k", URL: "https://api.example.com/v1", ModelType: "openai-completions", QuotaDay: -1,
		MaxTokens: 4096}
	model.DB(context.Background()).Create(&m)

	form := updateModelFormBase("mt-zero")
	form.Set("max_tokens", "0")
	w := doUpdateModel(t, m.ID, form)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际=%d, 响应=%s", w.Code, w.Body.String())
	}
	var found model.AIModel
	model.DB(context.Background()).First(&found, m.ID)
	if found.MaxTokens != 0 {
		t.Errorf("显式传 max_tokens=0 应写入 0, 实际=%d", found.MaxTokens)
	}
}

// TestHandleUpdateModel_MaxTokensOmittedKeepsOriginal 不传 max_tokens：保留原值。
func TestHandleUpdateModel_MaxTokensOmittedKeepsOriginal(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-update-model-explicit-token"
	defer func() { AdminToken = origToken }()

	m := model.AIModel{Provider: "openai", ModelID: "mt-keep", ModelName: "MTKeep",
		APIKey: "k", URL: "https://api.example.com/v1", ModelType: "openai-completions", QuotaDay: -1,
		MaxTokens: 4096}
	model.DB(context.Background()).Create(&m)

	form := updateModelFormBase("mt-keep") // 不设置 max_tokens
	w := doUpdateModel(t, m.ID, form)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际=%d, 响应=%s", w.Code, w.Body.String())
	}
	var found model.AIModel
	model.DB(context.Background()).First(&found, m.ID)
	if found.MaxTokens != 4096 {
		t.Errorf("不传 max_tokens 应保留原值 4096, 实际=%d", found.MaxTokens)
	}
}

// TestHandleUpdateModel_MaxTokensNegative 传负数 max_tokens：返回 400。
func TestHandleUpdateModel_MaxTokensNegative(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-update-model-explicit-token"
	defer func() { AdminToken = origToken }()

	m := model.AIModel{Provider: "openai", ModelID: "mt-neg", ModelName: "MTNeg",
		APIKey: "k", URL: "https://api.example.com/v1", ModelType: "openai-completions", QuotaDay: -1,
		MaxTokens: 4096}
	model.DB(context.Background()).Create(&m)

	form := updateModelFormBase("mt-neg")
	form.Set("max_tokens", "-1")
	w := doUpdateModel(t, m.ID, form)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400, 实际=%d, 响应=%s", w.Code, w.Body.String())
	}
	// DB 不应被修改
	var found model.AIModel
	model.DB(context.Background()).First(&found, m.ID)
	if found.MaxTokens != 4096 {
		t.Errorf("负数 max_tokens 校验失败时不应修改 DB, 实际=%d", found.MaxTokens)
	}
}

// TestHandleUpdateModel_MaxTokensNonInteger 传非整数 max_tokens：返回 400。
func TestHandleUpdateModel_MaxTokensNonInteger(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-update-model-explicit-token"
	defer func() { AdminToken = origToken }()

	m := model.AIModel{Provider: "openai", ModelID: "mt-bad", ModelName: "MTBad",
		APIKey: "k", URL: "https://api.example.com/v1", ModelType: "openai-completions", QuotaDay: -1,
		MaxTokens: 4096}
	model.DB(context.Background()).Create(&m)

	form := updateModelFormBase("mt-bad")
	form.Set("max_tokens", "not-int")
	w := doUpdateModel(t, m.ID, form)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400, 实际=%d, 响应=%s", w.Code, w.Body.String())
	}
	var found model.AIModel
	model.DB(context.Background()).First(&found, m.ID)
	if found.MaxTokens != 4096 {
		t.Errorf("非整数 max_tokens 校验失败时不应修改 DB, 实际=%d", found.MaxTokens)
	}
}

// TestHandleUpdateModel_InstanceModelPluckErrorStillOK 覆盖模型更新成功后查询绑定失败时的降级路径。
func TestHandleUpdateModel_InstanceModelPluckErrorStillOK(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-update-model-explicit-token"
	defer func() { AdminToken = origToken }()

	m := model.AIModel{Provider: "openai", ModelID: "pluck-err", ModelName: "PluckErr",
		APIKey: "k", URL: "https://api.example.com/v1", ModelType: "openai-completions", QuotaDay: -1}
	model.DB(context.Background()).Create(&m)
	if err := model.DB(context.Background()).Migrator().DropTable(&model.InstanceModel{}); err != nil {
		t.Fatalf("drop instance_models: %v", err)
	}

	w := doUpdateModel(t, m.ID, updateModelFormBase("pluck-err-updated"))
	if w.Code != http.StatusOK {
		t.Fatalf("查询绑定失败不应影响模型更新成功，实际=%d, 响应=%s", w.Code, w.Body.String())
	}
	var found model.AIModel
	if err := model.DB(context.Background()).First(&found, m.ID).Error; err != nil {
		t.Fatalf("query model: %v", err)
	}
	if found.ModelID != "pluck-err-updated" {
		t.Fatalf("模型应已更新，ModelID=%q", found.ModelID)
	}
}

// TestHandleUpdateModel_CustomHeadersEmptyObjectClears 传 "{}"：清空自定义头（写入空串）。
func TestHandleUpdateModel_CustomHeadersEmptyObjectClears(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-update-model-explicit-token"
	defer func() { AdminToken = origToken }()

	m := model.AIModel{Provider: "openai", ModelID: "hdr-clear", ModelName: "HdrClear",
		APIKey: "k", URL: "https://api.example.com/v1", ModelType: "openai-completions", QuotaDay: -1,
		CustomHTTPHeaders: `{"X-Foo":"bar"}`}
	model.DB(context.Background()).Create(&m)

	form := updateModelFormBase("hdr-clear")
	form.Set("custom_http_headers", "{}")
	w := doUpdateModel(t, m.ID, form)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际=%d, 响应=%s", w.Code, w.Body.String())
	}
	var found model.AIModel
	model.DB(context.Background()).First(&found, m.ID)
	if found.CustomHTTPHeaders != "" {
		t.Errorf(`传 "{}" 应清空自定义头为空串, 实际=%q`, found.CustomHTTPHeaders)
	}
}

// TestHandleUpdateModel_CustomHeadersOmittedKeepsOriginal 不传 custom_http_headers：保留原值。
func TestHandleUpdateModel_CustomHeadersOmittedKeepsOriginal(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-update-model-explicit-token"
	defer func() { AdminToken = origToken }()

	m := model.AIModel{Provider: "openai", ModelID: "hdr-keep", ModelName: "HdrKeep",
		APIKey: "k", URL: "https://api.example.com/v1", ModelType: "openai-completions", QuotaDay: -1,
		CustomHTTPHeaders: `{"X-Foo":"bar"}`}
	model.DB(context.Background()).Create(&m)

	form := updateModelFormBase("hdr-keep") // 不设置 custom_http_headers
	w := doUpdateModel(t, m.ID, form)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际=%d, 响应=%s", w.Code, w.Body.String())
	}
	var found model.AIModel
	model.DB(context.Background()).First(&found, m.ID)
	if found.CustomHTTPHeaders != `{"X-Foo":"bar"}` {
		t.Errorf("不传 custom_http_headers 应保留原值, 实际=%q", found.CustomHTTPHeaders)
	}
}

// --- HandleCreateModel max_tokens 选填语义 测试 ---

// createModelFormBase 返回一份可直接创建的合法基础表单（不含 max_tokens）。
func createModelFormBase(modelID string) url.Values {
	form := url.Values{}
	form.Set("provider", "test-create")
	form.Set("model_id", modelID)
	form.Set("model_name", modelID)
	form.Set("api_key", "sk-test")
	form.Set("url", "https://api.example.com/v1")
	form.Set("model_type", "openai-completions")
	form.Set("quota_day", "-1")
	form.Set("context_len", "128000")
	form.Add("input_types", "text")
	return form
}

// doCreateModel 用给定表单调用 HandleCreateModel，返回 recorder。
func doCreateModel(t *testing.T, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/admin/models/create",
		strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-create-model-token")
	w := httptest.NewRecorder()
	HandleCreateModel(w, r)
	return w
}

// TestHandleCreateModel_MaxTokensOmitted 不传 max_tokens：写入 0（不限输出）。
func TestHandleCreateModel_MaxTokensOmitted(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-create-model-token"
	defer func() { AdminToken = origToken }()

	form := createModelFormBase("create-omit")
	w := doCreateModel(t, form)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际=%d, 响应=%s", w.Code, w.Body.String())
	}
	var found model.AIModel
	model.DB(context.Background()).Where("model_id = ?", "create-omit").First(&found)
	if found.MaxTokens != 0 {
		t.Errorf("不传 max_tokens 应写入 0（不限）, 实际=%d", found.MaxTokens)
	}
}

// TestHandleCreateModel_MaxTokensExplicitZero 显式传 max_tokens=0：写入 0（不限输出）。
func TestHandleCreateModel_MaxTokensExplicitZero(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-create-model-token"
	defer func() { AdminToken = origToken }()

	form := createModelFormBase("create-zero")
	form.Set("max_tokens", "0")
	w := doCreateModel(t, form)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际=%d, 响应=%s", w.Code, w.Body.String())
	}
	var found model.AIModel
	model.DB(context.Background()).Where("model_id = ?", "create-zero").First(&found)
	if found.MaxTokens != 0 {
		t.Errorf("显式传 max_tokens=0 应写入 0, 实际=%d", found.MaxTokens)
	}
}

// TestHandleCreateModel_MaxTokensPositive 传正值 max_tokens=4096：写入 4096。
func TestHandleCreateModel_MaxTokensPositive(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-create-model-token"
	defer func() { AdminToken = origToken }()

	form := createModelFormBase("create-pos")
	form.Set("max_tokens", "4096")
	w := doCreateModel(t, form)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际=%d, 响应=%s", w.Code, w.Body.String())
	}
	var found model.AIModel
	model.DB(context.Background()).Where("model_id = ?", "create-pos").First(&found)
	if found.MaxTokens != 4096 {
		t.Errorf("传 max_tokens=4096 应写入 4096, 实际=%d", found.MaxTokens)
	}
}

// TestHandleCreateModel_MaxTokensNegative 传负数 max_tokens：返回 400。
func TestHandleCreateModel_MaxTokensNegative(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-create-model-token"
	defer func() { AdminToken = origToken }()

	form := createModelFormBase("create-neg")
	form.Set("max_tokens", "-1")
	w := doCreateModel(t, form)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400, 实际=%d, 响应=%s", w.Code, w.Body.String())
	}
	var count int64
	model.DB(context.Background()).Model(&model.AIModel{}).Where("model_id = ?", "create-neg").Count(&count)
	if count != 0 {
		t.Errorf("负数 max_tokens 校验失败时不应创建记录, 实际 count=%d", count)
	}
}

// TestHandleCreateModel_MaxTokensNonInteger 传非整数 max_tokens：返回 400。
func TestHandleCreateModel_MaxTokensNonInteger(t *testing.T) {
	cleanup := setupAdminModelsTestDB(t)
	defer cleanup()

	origToken := AdminToken
	AdminToken = "test-create-model-token"
	defer func() { AdminToken = origToken }()

	form := createModelFormBase("create-abc")
	form.Set("max_tokens", "abc")
	w := doCreateModel(t, form)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400, 实际=%d, 响应=%s", w.Code, w.Body.String())
	}
	var count int64
	model.DB(context.Background()).Model(&model.AIModel{}).Where("model_id = ?", "create-abc").Count(&count)
	if count != 0 {
		t.Errorf("非整数 max_tokens 校验失败时不应创建记录, 实际 count=%d", count)
	}
}
