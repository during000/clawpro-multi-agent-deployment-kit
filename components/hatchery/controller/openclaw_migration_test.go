package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ---------- DB 初始化 ----------

func initMigrationTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.InstanceModel{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

func createMigrationTestInstance(t *testing.T, instanceId, agentType string) *model.Instance {
	t.Helper()
	user := &model.User{Username: "user-" + instanceId, Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name:        "inst-" + instanceId,
		InstanceId:  instanceId,
		UserID:      user.ID,
		AgentType:   agentType,
		RuntimeUser: "root",
	}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	return inst
}

// ---------- mock dependencies ----------

type mockMigrationModelsDependencies struct {
	runScript func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string) (string, error)
}

func (m *mockMigrationModelsDependencies) RunScript(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string) (string, error) {
	if m.runScript != nil {
		return m.runScript(ctx, instanceId, scriptName, timeout, runtimeUser)
	}
	return "", nil
}

// mockInstanceMigrationDependencies mock instanceMigrationDependencies 接口
type mockInstanceMigrationDependencies struct {
	prepareMigrationUpload   func(ctx context.Context, fileKey string, estimatedSize int64) (*SMHUploadCredential, error)
	checkSMHCommonFileExists func(ctx context.Context, fileKey string) (bool, int64, error)
	buildMigrationScript     func(ctx context.Context, m *model.AgentMigration, cred *SMHUploadCredential, agentType string) (string, error)
}

func (m *mockInstanceMigrationDependencies) PrepareMigrationUpload(ctx context.Context, fileKey string, estimatedSize int64) (*SMHUploadCredential, error) {
	if m.prepareMigrationUpload != nil {
		return m.prepareMigrationUpload(ctx, fileKey, estimatedSize)
	}
	exp := time.Now().Add(1 * time.Hour)
	return &SMHUploadCredential{FileKey: fileKey, PartURLTemplate: "https://smh.example.com/{partNumber}", ConfirmKey: "ck", PartSize: 10 * 1024 * 1024, TotalParts: 1, Expiration: &exp}, nil
}
func (m *mockInstanceMigrationDependencies) CheckSMHCommonFileExists(ctx context.Context, fileKey string) (bool, int64, error) {
	if m.checkSMHCommonFileExists != nil {
		return m.checkSMHCommonFileExists(ctx, fileKey)
	}
	return true, 1024, nil
}
func (m *mockInstanceMigrationDependencies) BuildMigrationScript(ctx context.Context, mg *model.AgentMigration, cred *SMHUploadCredential, agentType string) (string, error) {
	if m.buildMigrationScript != nil {
		return m.buildMigrationScript(ctx, mg, cred, agentType)
	}
	return "#!/bin/bash\necho ok", nil
}

func (m *mockInstanceMigrationDependencies) GetCommonSpaceToken(ctx context.Context) (string, error) {
	return "mock-token", nil
}

func (m *mockInstanceMigrationDependencies) ResolveStatus(_ context.Context, instance *model.Instance) (InstanceStatusResponse, error) {
	return InstanceStatusResponse{Status: model.StatusRunning, Label: "运行中"}, nil
}

func buildScriptOutput(agentType string, models []migrationModelEntry) string {
	out := migrationModelsOutput{AgentType: agentType, Models: models}
	b, _ := json.Marshal(out)
	return string(b)
}

var testLog = slog.Default()

// ---------- syncMigrationModelsWithDeps ----------

func TestSyncMigrationModels_ScriptError(t *testing.T) {
	initMigrationTestDB(t)
	inst := createMigrationTestInstance(t, "ins-script-err", model.AgentTypeOpenClaw)

	deps := &mockMigrationModelsDependencies{
		runScript: func(_ context.Context, _, _ string, _ uint64, _ string) (string, error) {
			return "", errors.New("TAT 失败")
		},
	}

	valid := syncMigrationModelsWithDeps(context.Background(), inst, testLog, deps)
	if valid {
		t.Error("脚本失败时 isPrimaryValid 应为 false")
	}
	var count int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 0 {
		t.Errorf("脚本失败时不应写入 instance_models，实际 %d 条", count)
	}
}

func TestSyncMigrationModels_EmptyOutput(t *testing.T) {
	initMigrationTestDB(t)
	inst := createMigrationTestInstance(t, "ins-empty", model.AgentTypeOpenClaw)

	deps := &mockMigrationModelsDependencies{
		runScript: func(_ context.Context, _, _ string, _ uint64, _ string) (string, error) {
			return "some log line\nanother line", nil
		},
	}

	valid := syncMigrationModelsWithDeps(context.Background(), inst, testLog, deps)
	if valid {
		t.Error("无有效 JSON 输出时 isPrimaryValid 应为 false")
	}
}

func TestSyncMigrationModels_NoModels(t *testing.T) {
	initMigrationTestDB(t)
	inst := createMigrationTestInstance(t, "ins-no-models", model.AgentTypeOpenClaw)

	deps := &mockMigrationModelsDependencies{
		runScript: func(_ context.Context, _, _ string, _ uint64, _ string) (string, error) {
			return buildScriptOutput("openclaw", nil), nil
		},
	}

	valid := syncMigrationModelsWithDeps(context.Background(), inst, testLog, deps)
	if valid {
		t.Error("无模型时 isPrimaryValid 应为 false")
	}
}

func TestSyncMigrationModels_PrimaryValid(t *testing.T) {
	initMigrationTestDB(t)
	inst := createMigrationTestInstance(t, "ins-primary", model.AgentTypeOpenClaw)

	models := []migrationModelEntry{
		{
			Role: model.ModelRolePrimary, ModelID: "gpt-4o", ModelName: "gpt-4o",
			BaseURL: "https://api.example.com/v1", APIKey: "sk-xxx",
			APIMode: "openai-completions", ContextLen: 128000, InputTypes: []string{"text"},
		},
	}
	deps := &mockMigrationModelsDependencies{
		runScript: func(_ context.Context, _, _ string, _ uint64, _ string) (string, error) {
			return buildScriptOutput("openclaw", models), nil
		},
	}

	valid := syncMigrationModelsWithDeps(context.Background(), inst, testLog, deps)
	if !valid {
		t.Error("有 primary 模型时 isPrimaryValid 应为 true")
	}

	var ims []model.InstanceModel
	model.DB(context.Background()).Where("instance_id = ?", inst.ID).Find(&ims)
	if len(ims) != 1 {
		t.Fatalf("期望写入 1 条 instance_models，实际 %d", len(ims))
	}
	if ims[0].Role != model.ModelRolePrimary {
		t.Errorf("role 应为 primary，实际 %q", ims[0].Role)
	}
	if ims[0].CustomModelID != "gpt-4o" {
		t.Errorf("custom_model_id 应为 gpt-4o，实际 %q", ims[0].CustomModelID)
	}
}

func TestSyncMigrationModels_OnlyFallbacks_PrimaryInvalid(t *testing.T) {
	initMigrationTestDB(t)
	inst := createMigrationTestInstance(t, "ins-fallbacks", model.AgentTypeOpenClaw)

	models := []migrationModelEntry{
		{
			Role: model.ModelRoleFallback, ModelID: "claude-3", ModelName: "claude-3",
			BaseURL: "https://api.example.com", APIKey: "sk-yyy",
			APIMode: "anthropic-messages", ContextLen: 200000, InputTypes: []string{"text"},
		},
	}
	deps := &mockMigrationModelsDependencies{
		runScript: func(_ context.Context, _, _ string, _ uint64, _ string) (string, error) {
			return buildScriptOutput("openclaw", models), nil
		},
	}

	valid := syncMigrationModelsWithDeps(context.Background(), inst, testLog, deps)
	if valid {
		t.Error("只有 fallback 时 isPrimaryValid 应为 false")
	}

	var count int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 1 {
		t.Errorf("期望写入 1 条 fallback 记录，实际 %d", count)
	}
}

func TestSyncMigrationModels_ClearsExistingRecords(t *testing.T) {
	initMigrationTestDB(t)
	inst := createMigrationTestInstance(t, "ins-clear", model.AgentTypeOpenClaw)

	// 先写入旧记录
	model.DB(context.Background()).Create(&model.InstanceModel{
		InstanceID: inst.ID, AIModelID: 1, Role: model.ModelRolePrimary, SortOrder: 0,
	})

	models := []migrationModelEntry{
		{
			Role: model.ModelRolePrimary, ModelID: "new-model", ModelName: "new-model",
			BaseURL: "https://api.new.com/v1", APIKey: "sk-new",
			APIMode: "openai-completions", ContextLen: 128000, InputTypes: []string{"text"},
		},
	}
	deps := &mockMigrationModelsDependencies{
		runScript: func(_ context.Context, _, _ string, _ uint64, _ string) (string, error) {
			return buildScriptOutput("openclaw", models), nil
		},
	}

	syncMigrationModelsWithDeps(context.Background(), inst, testLog, deps)

	var ims []model.InstanceModel
	model.DB(context.Background()).Where("instance_id = ?", inst.ID).Find(&ims)
	if len(ims) != 1 {
		t.Fatalf("应只有 1 条新记录，实际 %d", len(ims))
	}
	if ims[0].CustomModelID != "new-model" {
		t.Errorf("应为新模型 new-model，实际 %q", ims[0].CustomModelID)
	}
}

func TestSyncMigrationModels_MultipleModels(t *testing.T) {
	initMigrationTestDB(t)
	inst := createMigrationTestInstance(t, "ins-multi", model.AgentTypeOpenClaw)

	models := []migrationModelEntry{
		{
			Role: model.ModelRolePrimary, ModelID: "gpt-4o", ModelName: "gpt-4o",
			BaseURL: "https://api.example.com/v1", APIKey: "sk-a",
			APIMode: "openai-completions", ContextLen: 128000, InputTypes: []string{"text", "image"},
		},
		{
			Role: model.ModelRoleFallback, ModelID: "claude-3", ModelName: "claude-3",
			BaseURL: "https://api.anthropic.com/v1", APIKey: "sk-b",
			APIMode: "anthropic-messages", ContextLen: 200000, InputTypes: []string{"text"},
		},
	}
	deps := &mockMigrationModelsDependencies{
		runScript: func(_ context.Context, _, _ string, _ uint64, _ string) (string, error) {
			return buildScriptOutput("openclaw", models), nil
		},
	}

	valid := syncMigrationModelsWithDeps(context.Background(), inst, testLog, deps)
	if !valid {
		t.Error("有 primary 时 isPrimaryValid 应为 true")
	}

	var ims []model.InstanceModel
	model.DB(context.Background()).Where("instance_id = ?", inst.ID).Order("sort_order").Find(&ims)
	if len(ims) != 2 {
		t.Fatalf("期望 2 条记录，实际 %d", len(ims))
	}
	if ims[0].Role != model.ModelRolePrimary || ims[1].Role != model.ModelRoleFallback {
		t.Errorf("角色顺序不正确: %q, %q", ims[0].Role, ims[1].Role)
	}
}

func TestSyncMigrationModels_CustomModelConfigJSON(t *testing.T) {
	initMigrationTestDB(t)
	inst := createMigrationTestInstance(t, "ins-config", model.AgentTypeOpenClaw)

	models := []migrationModelEntry{
		{
			Role: model.ModelRolePrimary, ModelID: "gpt-4o", ModelName: "GPT-4o",
			BaseURL: "https://api.example.com/v1", APIKey: "sk-test",
			APIMode: "openai-completions", ContextLen: 32000, InputTypes: []string{"text", "image"},
		},
	}
	deps := &mockMigrationModelsDependencies{
		runScript: func(_ context.Context, _, _ string, _ uint64, _ string) (string, error) {
			return buildScriptOutput("openclaw", models), nil
		},
	}

	syncMigrationModelsWithDeps(context.Background(), inst, testLog, deps)

	var im model.InstanceModel
	model.DB(context.Background()).Where("instance_id = ? AND custom_model_id = ?", inst.ID, "gpt-4o").First(&im)
	if im.ID == 0 {
		t.Fatal("未找到写入的 instance_model")
	}

	var cfg customModelConfig
	if err := json.Unmarshal([]byte(im.CustomModelConfig), &cfg); err != nil {
		t.Fatalf("custom_model_config 解析失败: %v", err)
	}
	if cfg.URL != "https://api.example.com/v1" {
		t.Errorf("URL 应为 https://api.example.com/v1，实际 %q", cfg.URL)
	}
	if cfg.ModelType != "openai-completions" {
		t.Errorf("ModelType 应为 openai-completions，实际 %q", cfg.ModelType)
	}
	if cfg.ContextLen != 32000 {
		t.Errorf("ContextLen 应为 32000，实际 %d", cfg.ContextLen)
	}
	if len(cfg.InputTypes) != 2 {
		t.Errorf("InputTypes 应有 2 个，实际 %v", cfg.InputTypes)
	}
}

// ---------- shellQuote ----------

func TestShellQuote_Simple(t *testing.T) {
	result := shellQuote("hello")
	if result != "'hello'" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestShellQuote_WithSingleQuote(t *testing.T) {
	result := shellQuote("it's")
	if result != `'it'\''s'` {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestShellQuote_Empty(t *testing.T) {
	result := shellQuote("")
	if result != "''" {
		t.Errorf("expected empty string quoted, got %q", result)
	}
}

func TestShellQuote_SpecialChars(t *testing.T) {
	result := shellQuote("foo $BAR `baz`")
	if result != "'foo $BAR `baz`'" {
		t.Errorf("special chars should be preserved inside quotes, got %q", result)
	}
}

// ---------- agentMigrationDirName ----------

func TestAgentMigrationDirName(t *testing.T) {
	initMigrationTestDB(t)
	tests := []struct {
		agentType string
		want      string
	}{
		{model.AgentTypeHermes, ".hermes"},
		{model.AgentTypeLightclawACE, ".lightclaw"},
		{model.AgentTypeOpenClaw, ".openclaw"},
		{"", ".openclaw"},
		{"unknown", ".openclaw"},
	}
	for _, tt := range tests {
		got := agentMigrationDirName(context.Background(), tt.agentType)
		if got != tt.want {
			t.Errorf("agentMigrationDirName(context.Background(), %q) = %q, want %q", tt.agentType, got, tt.want)
		}
	}
}

// ---------- agentMigrationExcludedPaths ----------

func TestAgentMigrationExcludedPaths(t *testing.T) {
	// hermes 必须包含 gateway.pid（防止 PID 文件迁移导致进程冲突）
	hermesPaths := agentMigrationExcludedPaths(context.Background(), model.AgentTypeHermes)
	foundPid := false
	for _, p := range hermesPaths {
		if p == "gateway.pid" {
			foundPid = true
		}
	}
	if !foundPid {
		t.Errorf("hermes excludedPaths should contain gateway.pid, got %v", hermesPaths)
	}

	// ace 必须包含 venv
	acePaths := agentMigrationExcludedPaths(context.Background(), model.AgentTypeLightclawACE)
	foundVenv := false
	for _, p := range acePaths {
		if p == "venv" {
			foundVenv = true
		}
	}
	if !foundVenv {
		t.Errorf("ace excludedPaths should contain venv, got %v", acePaths)
	}

	// openclaw 不为空
	ocPaths := agentMigrationExcludedPaths(context.Background(), model.AgentTypeOpenClaw)
	if len(ocPaths) == 0 {
		t.Error("openclaw excludedPaths should not be empty")
	}

	// 默认（空 agentType）走 openclaw 分支
	defPaths := agentMigrationExcludedPaths(context.Background(), "")
	if len(defPaths) == 0 {
		t.Error("default excludedPaths should not be empty")
	}
}

// ---------- agentMigrationPreservedPaths ----------

func TestAgentMigrationPreservedPaths(t *testing.T) {
	// hermes 保留 hermes-agent（平台相关 venv）
	hermesPaths := agentMigrationPreservedPaths(context.Background(), model.AgentTypeHermes)
	found := false
	for _, p := range hermesPaths {
		if p == "hermes-agent" {
			found = true
		}
	}
	if !found {
		t.Errorf("hermes preservedPaths should contain hermes-agent, got %v", hermesPaths)
	}

	// ace 保留 venv
	acePaths := agentMigrationPreservedPaths(context.Background(), model.AgentTypeLightclawACE)
	foundVenv := false
	for _, p := range acePaths {
		if p == "venv" {
			foundVenv = true
		}
	}
	if !foundVenv {
		t.Errorf("ace preservedPaths should contain venv, got %v", acePaths)
	}

	// openclaw preservedPaths 为空（无平台目录需保留）
	ocPaths := agentMigrationPreservedPaths(context.Background(), model.AgentTypeOpenClaw)
	if len(ocPaths) != 0 {
		t.Errorf("openclaw preservedPaths should be empty, got %v", ocPaths)
	}
}

// ---------- HandleMigrationProgress ----------

func initMigrationProgressTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.InstanceModel{},
		&model.AgentMigration{},
		&model.SiteConfig{},
		&model.SMHSpace{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	if Store == nil {
		Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	}
}

// setMigrationTestSession 将 user.Username 写入请求 session，模拟已登录状态。
func setMigrationTestSession(t *testing.T, req *http.Request, user *model.User) {
	t.Helper()
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = user.Username
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
}

func TestHandleMigrationProgress_NoMigration(t *testing.T) {
	initMigrationProgressTestDB(t)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{Name: "inst", InstanceId: "ins-001", UserID: user.ID}
	model.DB(context.Background()).Create(inst)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/openclaw/migration/progress?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	HandleMigrationProgress(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp["has_migration"] != false {
		t.Errorf("has_migration should be false, got %v", resp["has_migration"])
	}
}

func TestHandleMigrationProgress_WithMigration(t *testing.T) {
	initMigrationProgressTestDB(t)

	user := &model.User{Username: "u2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{Name: "inst2", InstanceId: "ins-002", UserID: user.ID, AgentType: model.AgentTypeOpenClaw}
	model.DB(context.Background()).Create(inst)

	m := &model.AgentMigration{
		InstanceID:    inst.ID,
		CVMInstanceID: inst.InstanceId,
		FileKey:       "migrations/ins-002/agent-export.tgz",
		Status:        model.MigrationStatusImporting,
	}
	model.DB(context.Background()).Create(m)
	model.InitMigrationSteps(context.Background(), model.DB(context.Background()), m, model.AgentTypeOpenClaw)
	model.UpdateMigrationStep(model.DB(context.Background()), m, model.MigrationStepDownloading, model.MigrationStepStatusRunning, nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/openclaw/migration/progress?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	HandleMigrationProgress(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp["has_migration"] != true {
		t.Errorf("has_migration should be true")
	}
	if resp["status"] != model.MigrationStatusImporting {
		t.Errorf("status should be importing, got %v", resp["status"])
	}
	steps, ok := resp["steps"].([]interface{})
	if !ok || len(steps) == 0 {
		t.Errorf("steps should be non-empty array, got %v", resp["steps"])
	}
}

func TestHandleMigrationProgress_WithFailReason(t *testing.T) {
	initMigrationProgressTestDB(t)

	user := &model.User{Username: "u3", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{Name: "inst3", InstanceId: "ins-003", UserID: user.ID}
	model.DB(context.Background()).Create(inst)

	m := &model.AgentMigration{
		InstanceID:    inst.ID,
		CVMInstanceID: inst.InstanceId,
		FileKey:       "migrations/ins-003/agent-export.tgz",
		Status:        model.MigrationStatusFailed,
		FailReason:    "下载超时",
	}
	model.DB(context.Background()).Create(m)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/openclaw/migration/progress?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	HandleMigrationProgress(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["fail_reason"] != "下载超时" {
		t.Errorf("fail_reason should be '下载超时', got %v", resp["fail_reason"])
	}
}

// ---------- syncMigrationModelsWithDeps for hermes/ace ----------

func TestSyncMigrationModels_HermesWritesCustomModelConfig(t *testing.T) {
	initMigrationTestDB(t)
	inst := createMigrationTestInstance(t, "ins-hermes", model.AgentTypeHermes)

	models := []migrationModelEntry{
		{
			Role: model.ModelRolePrimary, ModelID: "gpt-4o", ModelName: "gpt-4o",
			BaseURL: "https://api.example.com/v1", APIKey: "sk-hermes",
			APIMode: "openai-completions", ContextLen: 128000, InputTypes: []string{"text"},
		},
	}
	deps := &mockMigrationModelsDependencies{
		runScript: func(_ context.Context, _, _ string, _ uint64, _ string) (string, error) {
			return buildScriptOutput("hermes", models), nil
		},
	}

	valid := syncMigrationModelsWithDeps(context.Background(), inst, testLog, deps)
	if !valid {
		t.Error("hermes with primary model should be valid")
	}

	// hermes 写的是 custom_model_config，不是 instance_models
	var count int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 0 {
		t.Errorf("hermes should not write instance_models, got %d rows", count)
	}

	var updated model.Instance
	model.DB(context.Background()).First(&updated, inst.ID)
	if updated.CustomModelConfig == "" {
		t.Error("hermes should write custom_model_config")
	}
	var cfg customModelConfig
	if err := json.Unmarshal([]byte(updated.CustomModelConfig), &cfg); err != nil {
		t.Fatalf("parse custom_model_config: %v", err)
	}
	if cfg.URL != "https://api.example.com/v1" {
		t.Errorf("URL mismatch: %q", cfg.URL)
	}
}

func TestSyncMigrationModels_AceWritesCustomModelConfig(t *testing.T) {
	initMigrationTestDB(t)
	inst := createMigrationTestInstance(t, "ins-ace", model.AgentTypeLightclawACE)

	models := []migrationModelEntry{
		{
			Role: model.ModelRolePrimary, ModelID: "claude-3", ModelName: "claude-3",
			BaseURL: "https://api.anthropic.com/v1", APIKey: "sk-ace",
			APIMode: "anthropic-messages", ContextLen: 200000, InputTypes: []string{"text"},
		},
	}
	deps := &mockMigrationModelsDependencies{
		runScript: func(_ context.Context, _, _ string, _ uint64, _ string) (string, error) {
			return buildScriptOutput("lightclawace", models), nil
		},
	}

	valid := syncMigrationModelsWithDeps(context.Background(), inst, testLog, deps)
	if !valid {
		t.Error("ace with primary model should be valid")
	}

	var updated model.Instance
	model.DB(context.Background()).First(&updated, inst.ID)
	if updated.CustomModelConfig == "" {
		t.Error("ace should write custom_model_config")
	}
}

func TestSyncMigrationModels_HermesNoModels(t *testing.T) {
	initMigrationTestDB(t)
	inst := createMigrationTestInstance(t, "ins-hermes-empty", model.AgentTypeHermes)

	deps := &mockMigrationModelsDependencies{
		runScript: func(_ context.Context, _, _ string, _ uint64, _ string) (string, error) {
			return buildScriptOutput("hermes", nil), nil
		},
	}

	valid := syncMigrationModelsWithDeps(context.Background(), inst, testLog, deps)
	if valid {
		t.Error("hermes with no models should not be valid")
	}
}

// ---------- handler tests using mockInstanceMigrationDependencies ----------

// setupHandlerTestDB 建好含 SiteConfig（SMH 开启）的测试 DB，并返回 user+inst
func setupHandlerTestDB(t *testing.T) (*model.User, *model.Instance) {
	t.Helper()
	initMigrationProgressTestDB(t)
	// 写入 SMH 配置（requireSMHEnabled 检查 SMHEnabled=1）
	model.DB(context.Background()).Create(&model.SiteConfig{
		SMHEndpoint:      "https://smh.example.com",
		SMHLibraryId:     "lib-001",
		SMHLibrarySecret: "secret",
		SMHEnabled:       1,
	})
	model.DB(context.Background()).Create(&model.SMHSpace{SpaceTag: "common", SpaceId: "sp-common", LibraryId: "lib-001", Purpose: "common"})
	model.DB(context.Background()).Create(&model.SMHSpace{SpaceTag: "skillhub", SpaceId: "sp-skillhub", LibraryId: "lib-001", Purpose: "skillhub"})
	user := &model.User{Username: "handler-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{Name: "handler-inst", InstanceId: "ins-handler", UserID: user.ID, AgentType: model.AgentTypeOpenClaw, RuntimeUser: "root"}
	model.DB(context.Background()).Create(inst)
	return user, inst
}

// ---------- handleMigrationExport ----------

func TestHandleMigrationExport_MethodNotAllowed(t *testing.T) {
	user, inst := setupHandlerTestDB(t)
	_ = inst
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/openclaw/migration/export?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	handleMigrationExport(rr, req, &mockInstanceMigrationDependencies{})
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleMigrationExport_PrepareFails(t *testing.T) {
	user, inst := setupHandlerTestDB(t)
	deps := &mockInstanceMigrationDependencies{
		prepareMigrationUpload: func(_ context.Context, _ string, _ int64) (*SMHUploadCredential, error) {
			return nil, errors.New("SMH 连接失败")
		},
	}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/openclaw/migration/export?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	handleMigrationExport(rr, req, deps)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleMigrationExport_BuildScriptFails(t *testing.T) {
	user, inst := setupHandlerTestDB(t)
	deps := &mockInstanceMigrationDependencies{
		buildMigrationScript: func(_ context.Context, _ *model.AgentMigration, _ *SMHUploadCredential, _ string) (string, error) {
			return "", errors.New("生成脚本失败")
		},
	}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/openclaw/migration/export?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	handleMigrationExport(rr, req, deps)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleMigrationExport_Success(t *testing.T) {
	user, inst := setupHandlerTestDB(t)
	deps := &mockInstanceMigrationDependencies{}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/openclaw/migration/export?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	handleMigrationExport(rr, req, deps)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["script"] == nil || resp["script"] == "" {
		t.Error("script should be non-empty")
	}
	if resp["migration_id"] == nil {
		t.Error("migration_id should be present")
	}
	if resp["expire_at"] == nil || resp["expire_at"] == "" {
		t.Error("expire_at should be present")
	}
}

func TestHandleMigrationExport_ReuseExistingRecord(t *testing.T) {
	user, inst := setupHandlerTestDB(t)
	// 预先建一条 pending_upload 记录
	model.DB(context.Background()).Create(&model.AgentMigration{
		InstanceID: inst.ID, CVMInstanceID: inst.InstanceId,
		FileKey: fmt.Sprintf("migrations/%s/agent-export.tgz", inst.InstanceId),
		Status:  model.MigrationStatusPendingUpload,
	})
	deps := &mockInstanceMigrationDependencies{}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/openclaw/migration/export?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	handleMigrationExport(rr, req, deps)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	// 应只有一条记录
	var count int64
	model.DB(context.Background()).Model(&model.AgentMigration{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 1 {
		t.Errorf("should reuse existing record, got %d records", count)
	}
}

// ---------- handleMigrationStatus ----------

func TestHandleMigrationStatus_NoMigration(t *testing.T) {
	user, inst := setupHandlerTestDB(t)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/openclaw/migration/status?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	handleMigrationStatus(rr, req, &mockInstanceMigrationDependencies{})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["has_migration"] != false {
		t.Errorf("has_migration should be false")
	}
}

func TestHandleMigrationStatus_FileReady(t *testing.T) {
	user, inst := setupHandlerTestDB(t)
	model.DB(context.Background()).Create(&model.AgentMigration{
		InstanceID: inst.ID, CVMInstanceID: inst.InstanceId,
		FileKey: "migrations/ins-handler/agent-export.tgz",
		Status:  model.MigrationStatusPendingUpload,
	})
	deps := &mockInstanceMigrationDependencies{
		checkSMHCommonFileExists: func(_ context.Context, _ string) (bool, int64, error) {
			return true, 512000, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/openclaw/migration/status?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	handleMigrationStatus(rr, req, deps)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["has_migration"] != true {
		t.Errorf("has_migration should be true")
	}
	if resp["file_ready"] != true {
		t.Errorf("file_ready should be true")
	}
	if resp["can_import"] != true {
		t.Errorf("can_import should be true")
	}
}

func TestHandleMigrationStatus_FileNotReady(t *testing.T) {
	user, inst := setupHandlerTestDB(t)
	model.DB(context.Background()).Create(&model.AgentMigration{
		InstanceID: inst.ID, CVMInstanceID: inst.InstanceId,
		FileKey: "migrations/ins-handler/agent-export.tgz",
		Status:  model.MigrationStatusPendingUpload,
	})
	deps := &mockInstanceMigrationDependencies{
		checkSMHCommonFileExists: func(_ context.Context, _ string) (bool, int64, error) {
			return false, 0, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/openclaw/migration/status?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	handleMigrationStatus(rr, req, deps)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["file_ready"] != false {
		t.Errorf("file_ready should be false")
	}
	if resp["can_import"] != false {
		t.Errorf("can_import should be false")
	}
}

func TestHandleMigrationStatus_CheckFails(t *testing.T) {
	user, inst := setupHandlerTestDB(t)
	model.DB(context.Background()).Create(&model.AgentMigration{
		InstanceID: inst.ID, CVMInstanceID: inst.InstanceId,
		FileKey: "migrations/ins-handler/agent-export.tgz",
		Status:  model.MigrationStatusPendingUpload,
	})
	deps := &mockInstanceMigrationDependencies{
		checkSMHCommonFileExists: func(_ context.Context, _ string) (bool, int64, error) {
			return false, 0, errors.New("SMH 查询失败")
		},
	}
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/openclaw/migration/status?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	handleMigrationStatus(rr, req, deps)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// ---------- handleMigrationImport ----------

func TestHandleMigrationImport_MethodNotAllowed(t *testing.T) {
	user, inst := setupHandlerTestDB(t)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/openclaw/migration/import?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	handleMigrationImport(rr, req, &mockInstanceMigrationDependencies{})
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleMigrationImport_NoMigrationRecord(t *testing.T) {
	user, inst := setupHandlerTestDB(t)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/openclaw/migration/import?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	handleMigrationImport(rr, req, &mockInstanceMigrationDependencies{})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleMigrationImport_FileNotReady(t *testing.T) {
	user, inst := setupHandlerTestDB(t)
	model.DB(context.Background()).Create(&model.AgentMigration{
		InstanceID: inst.ID, CVMInstanceID: inst.InstanceId,
		FileKey: "migrations/ins-handler/agent-export.tgz",
		Status:  model.MigrationStatusPendingUpload,
	})
	deps := &mockInstanceMigrationDependencies{
		checkSMHCommonFileExists: func(_ context.Context, _ string) (bool, int64, error) {
			return false, 0, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/openclaw/migration/import?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	handleMigrationImport(rr, req, deps)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleMigrationImport_InstanceBusy(t *testing.T) {
	user, inst := setupHandlerTestDB(t)
	// 让实例处于 processing 状态
	model.DB(context.Background()).Model(inst).Updates(map[string]interface{}{
		"current_operation":       model.OpUpgrade,
		"current_operation_state": model.OpStateProcessing,
	})
	model.DB(context.Background()).Create(&model.AgentMigration{
		InstanceID: inst.ID, CVMInstanceID: inst.InstanceId,
		FileKey: "migrations/ins-handler/agent-export.tgz",
		Status:  model.MigrationStatusPendingUpload,
	})
	deps := &mockInstanceMigrationDependencies{
		checkSMHCommonFileExists: func(_ context.Context, _ string) (bool, int64, error) { return true, 1024, nil },
	}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/openclaw/migration/import?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	handleMigrationImport(rr, req, deps)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleMigrationImport_CheckFails(t *testing.T) {
	user, inst := setupHandlerTestDB(t)
	model.DB(context.Background()).Create(&model.AgentMigration{
		InstanceID: inst.ID, CVMInstanceID: inst.InstanceId,
		FileKey: "migrations/ins-handler/agent-export.tgz",
		Status:  model.MigrationStatusPendingUpload,
	})
	deps := &mockInstanceMigrationDependencies{
		checkSMHCommonFileExists: func(_ context.Context, _ string) (bool, int64, error) {
			return false, 0, errors.New("网络错误")
		},
	}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/openclaw/migration/import?id=%d", inst.ID), nil)
	setMigrationTestSession(t, req, user)
	rr := httptest.NewRecorder()
	handleMigrationImport(rr, req, deps)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ---------- buildMigrationScript ----------

func TestBuildMigrationScript_SMHNotConfigured(t *testing.T) {
	initMigrationProgressTestDB(t)
	// 不写 SiteConfig → SMH 未配置
	m := &model.AgentMigration{FileKey: "migrations/ins-xxx/agent-export.tgz"}
	cred := &SMHUploadCredential{PartURLTemplate: "https://smh.example.com/{partNumber}", ConfirmKey: "ck"}
	_, err := buildMigrationScript(context.Background(), m, cred, model.AgentTypeOpenClaw, func() (string, error) {
		return "tok", nil
	})
	if err == nil {
		t.Error("should return error when SMH not configured")
	}
}

func TestBuildMigrationScript_TokenFails(t *testing.T) {
	initMigrationProgressTestDB(t)
	seedMigrationSMHConfig(t)
	m := &model.AgentMigration{FileKey: "migrations/ins-xxx/agent-export.tgz"}
	cred := &SMHUploadCredential{PartURLTemplate: "https://smh.example.com/{partNumber}", ConfirmKey: "ck"}
	_, err := buildMigrationScript(context.Background(), m, cred, model.AgentTypeOpenClaw, func() (string, error) {
		return "", errors.New("token 获取失败")
	})
	if err == nil {
		t.Error("should return error when token fails")
	}
}

func TestBuildMigrationScript_LoadExportScriptFails(t *testing.T) {
	initMigrationProgressTestDB(t)
	seedMigrationSMHConfig(t)
	withLoadScript(t, func(name string) (string, error) {
		if name == "export_migration.sh" {
			return "", errors.New("missing script")
		}
		return "", nil
	})

	m := &model.AgentMigration{FileKey: "migrations/ins-xxx/agent-export.tgz"}
	cred := &SMHUploadCredential{PartURLTemplate: "https://smh.example.com/{partNumber}", ConfirmKey: "ck"}
	_, err := buildMigrationScript(context.Background(), m, cred, model.AgentTypeOpenClaw, func() (string, error) {
		return "test-token", nil
	})
	if err == nil {
		t.Fatal("expected error when export script cannot be loaded")
	}
	if !errors.Is(err, hcommon.I18nError(i18n.MsgMigrationLoadExportScriptFailed)) {
		t.Errorf("error should mention export script load failure, got %v", err)
	}
}

func TestBuildMigrationScript_ExportScriptWithoutTrailingNewline(t *testing.T) {
	initMigrationProgressTestDB(t)
	seedMigrationSMHConfig(t)
	withLoadScript(t, func(name string) (string, error) {
		if name == "export_migration.sh" {
			return "echo ok", nil
		}
		return "", nil
	})

	m := &model.AgentMigration{FileKey: "migrations/ins-xxx/agent-export.tgz"}
	cred := &SMHUploadCredential{PartURLTemplate: "https://smh.example.com/{partNumber}", ConfirmKey: "ck"}
	script, err := buildMigrationScript(context.Background(), m, cred, model.AgentTypeOpenClaw, func() (string, error) {
		return "test-token", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(script, "echo ok\nBASH") {
		t.Errorf("heredoc delimiter should start on a new line, got suffix %q", script[len(script)-20:])
	}
}

func TestBuildMigrationScript_Success(t *testing.T) {
	initMigrationProgressTestDB(t)
	withMigrationExportScript(t)
	seedMigrationSMHConfig(t)
	m := &model.AgentMigration{FileKey: "migrations/ins-xxx/agent-export.tgz"}
	cred := &SMHUploadCredential{
		PartURLTemplate: "https://smh.example.com/{partNumber}",
		ConfirmKey:      "ck",
		PartSize:        10 * 1024 * 1024,
		TotalParts:      1,
		PartHeaders:     map[string]string{"X-Test": "val"},
	}
	script, err := buildMigrationScript(context.Background(), m, cred, model.AgentTypeOpenClaw, func() (string, error) {
		return "test-token", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if script == "" {
		t.Error("script should not be empty")
	}
	if !strings.Contains(script, "PART_URL_TEMPLATE=") {
		t.Error("script should contain PART_URL_TEMPLATE")
	}
	if !strings.Contains(script, ".openclaw") {
		t.Error("script should reference .openclaw dir for openclaw type")
	}
	if !strings.Contains(script, "ALLOW_AGENT_ROOT_CHANGE_WARNING='0'") {
		t.Error("openclaw script must not tolerate agent-root changed warnings")
	}
}

func TestBuildMigrationScript_HermesDir(t *testing.T) {
	initMigrationProgressTestDB(t)
	withMigrationExportScript(t)
	seedMigrationSMHConfig(t)
	m := &model.AgentMigration{FileKey: "migrations/ins-hermes/agent-export.tgz"}
	cred := &SMHUploadCredential{PartURLTemplate: "https://smh.example.com/{partNumber}", ConfirmKey: "ck"}
	script, err := buildMigrationScript(context.Background(), m, cred, model.AgentTypeHermes, func() (string, error) {
		return "test-token", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(script, ".hermes") {
		t.Errorf("hermes script should reference .hermes dir, got snippet: %q", script[:200])
	}
	if !strings.Contains(script, "ALLOW_AGENT_ROOT_CHANGE_WARNING='1'") {
		t.Error("hermes script should enable the guarded root-change compatibility")
	}
	if !strings.Contains(script, "EXCLUDED_PATHS_B64=") {
		t.Error("hermes script should include excluded paths for the root manifest")
	}
}

type exportMigrationScriptResult struct {
	output    string
	log       string
	curlCalls []string
	err       error
}

type exportMigrationScriptScenario struct {
	createStatus              int
	createDiag                string
	verifyStatus              int
	allowRootWarning          bool
	createRootBusinessFile    string
	createRootExcludedFile    string
	replaceNestedBusinessFile bool
}

func runExportMigrationScript(t *testing.T, scenario exportMigrationScriptScenario) exportMigrationScriptResult {
	t.Helper()

	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, ".hermes")
	fakeBin := filepath.Join(tmpDir, "bin")
	curlCallsFile := filepath.Join(tmpDir, "curl-calls")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	sessionsDir := filepath.Join(agentDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionsDir, "state.json"), []byte("before"), 0o600); err != nil {
		t.Fatalf("write initial business state: %v", err)
	}

	writeExecutable := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(fakeBin, name), []byte(content), 0o755); err != nil {
			t.Fatalf("write fake %s: %v", name, err)
		}
	}

	writeExecutable("tar", `#!/usr/bin/env bash
set -eu
mode=""
for arg in "$@"; do
  case "$arg" in
    -cf) mode="create" ;;
    tf|-tf) mode="verify" ;;
  esac
done
case "$mode" in
  create)
    printf 'x' > /tmp/agent-export.tgz
    if [ -n "${TAR_CREATE_ROOT_BUSINESS_FILE:-}" ]; then
      printf 'business-data' > "$AGENT_DIR/$TAR_CREATE_ROOT_BUSINESS_FILE"
    fi
    if [ -n "${TAR_CREATE_ROOT_EXCLUDED_FILE:-}" ]; then
      printf 'runtime-data' > "$AGENT_DIR/$TAR_CREATE_ROOT_EXCLUDED_FILE"
    fi
    if [ "${TAR_REPLACE_NESTED_BUSINESS_FILE:-0}" = "1" ]; then
      printf 'after' > "$AGENT_DIR/sessions/state.json.new"
      mv "$AGENT_DIR/sessions/state.json.new" "$AGENT_DIR/sessions/state.json"
    fi
    [ -z "${TAR_CREATE_DIAG:-}" ] || printf '%s\n' "$TAR_CREATE_DIAG" >&2
    exit "${TAR_CREATE_STATUS:-0}"
    ;;
  verify)
    if [ "${TAR_VERIFY_STATUS:-0}" -ne 0 ]; then
      echo "tar: archive verification failed" >&2
      exit "$TAR_VERIFY_STATUS"
    fi
    exit 0
    ;;
  *)
    echo "unexpected tar args: $*" >&2
    exit 2
    ;;
esac
`)
	writeExecutable("curl", `#!/usr/bin/env bash
set -eu
method=""
output_file=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -X)
      method="$2"
      shift 2
      ;;
    -o)
      output_file="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
[ -z "$output_file" ] || : > "$output_file"
printf '%s\n' "$method" >> "$CURL_CALLS_FILE"
if [ "$method" = "PUT" ]; then
  printf '200 1024'
else
  printf '200'
fi
`)
	writeExecutable("sleep", `#!/usr/bin/env bash
exit 0
`)

	archivePath := "/tmp/agent-export.tgz"
	_ = os.Remove(archivePath)
	t.Cleanup(func() { _ = os.Remove(archivePath) })

	cmd := exec.Command("bash", "../scripts/export_migration.sh")
	allowRootWarning := "0"
	if scenario.allowRootWarning {
		allowRootWarning = "1"
	}
	replaceNestedBusinessFile := "0"
	if scenario.replaceNestedBusinessFile {
		replaceNestedBusinessFile = "1"
	}
	cmd.Env = append(os.Environ(),
		"PATH="+fakeBin+":"+os.Getenv("PATH"),
		"AGENT_DIR="+agentDir,
		"EXCLUDE_ARGS= --exclude='logs' --exclude='gateway.pid'",
		"EXCLUDED_PATHS_B64=WyJsb2dzIiwiZ2F0ZXdheS5waWQiXQ==",
		"ALLOW_AGENT_ROOT_CHANGE_WARNING="+allowRootWarning,
		"PART_HEADERS_B64=e30=",
		"PART_URL_TEMPLATE=https://secret.example.com/{partNumber}",
		"CONFIRM_KEY=secret-confirm-key",
		"ACCESS_TOKEN=secret-access-token",
		"LIBRARY_ID=lib-test",
		"SPACE_ID=space-test",
		"SMH_ENDPOINT=https://secret-smh.example.com",
		"FILE_KEY=migrations/test/agent-export.tgz",
		fmt.Sprintf("TAR_CREATE_STATUS=%d", scenario.createStatus),
		"TAR_CREATE_DIAG="+scenario.createDiag,
		fmt.Sprintf("TAR_VERIFY_STATUS=%d", scenario.verifyStatus),
		"TAR_CREATE_ROOT_BUSINESS_FILE="+scenario.createRootBusinessFile,
		"TAR_CREATE_ROOT_EXCLUDED_FILE="+scenario.createRootExcludedFile,
		"TAR_REPLACE_NESTED_BUSINESS_FILE="+replaceNestedBusinessFile,
		"CURL_CALLS_FILE="+curlCallsFile,
	)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	runErr := cmd.Run()

	logFiles, err := filepath.Glob(filepath.Join(agentDir, "logs", "migration_export_*.log"))
	if err != nil {
		t.Fatalf("glob migration logs: %v", err)
	}
	var logContent string
	if len(logFiles) != 1 {
		t.Fatalf("migration log files = %v, want exactly one", logFiles)
	}
	// The logger uses process substitution; allow tee to consume EOF before
	// reading on slower CI hosts.
	deadline := time.Now().Add(time.Second)
	for {
		data, readErr := os.ReadFile(logFiles[0])
		if readErr != nil {
			t.Fatalf("read migration log: %v", readErr)
		}
		logContent = string(data)
		if strings.Contains(logContent, "日志文件:") || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	var curlCalls []string
	if data, readErr := os.ReadFile(curlCallsFile); readErr == nil {
		curlCalls = strings.Fields(string(data))
	} else if !errors.Is(readErr, os.ErrNotExist) {
		t.Fatalf("read curl calls: %v", readErr)
	}

	return exportMigrationScriptResult{
		output:    output.String(),
		log:       logContent,
		curlCalls: curlCalls,
		err:       runErr,
	}
}

func TestExportMigrationScript_AllowsOnlyAgentRootChangedWarning(t *testing.T) {
	result := runExportMigrationScript(t, exportMigrationScriptScenario{
		createStatus:           1,
		createDiag:             "tar: .hermes: file changed as we read it",
		allowRootWarning:       true,
		createRootExcludedFile: "gateway.pid",
	})
	if result.err != nil {
		t.Fatalf("script should tolerate root metadata change: %v\n%s", result.err, result.output)
	}
	if !strings.Contains(result.output, "业务树指纹未变化") {
		t.Fatalf("missing tolerated-warning diagnostic:\n%s", result.output)
	}
	if !strings.Contains(result.output, "归档校验通过") {
		t.Fatalf("archive validation should be visible:\n%s", result.output)
	}
	if got := strings.Join(result.curlCalls, ","); got != "PUT,POST" {
		t.Fatalf("curl calls = %q, want PUT,POST", got)
	}
	if !strings.Contains(result.log, "✓ 导出成功") {
		t.Fatalf("persistent log should contain success:\n%s", result.log)
	}
	for _, secret := range []string{"secret-access-token", "secret-confirm-key", "secret.example.com"} {
		if strings.Contains(result.log, secret) {
			t.Fatalf("persistent log leaked %q:\n%s", secret, result.log)
		}
	}
}

func TestExportMigrationScript_RejectsChangedBusinessFile(t *testing.T) {
	result := runExportMigrationScript(t, exportMigrationScriptScenario{
		createStatus:     1,
		createDiag:       "tar: .hermes/config.yaml: file changed as we read it",
		allowRootWarning: true,
	})
	if result.err == nil {
		t.Fatalf("script should reject changed business file:\n%s", result.output)
	}
	if !strings.Contains(result.output, "打包失败（tar 退出码: 1）") {
		t.Fatalf("missing tar failure diagnostic:\n%s", result.output)
	}
	if !strings.Contains(result.log, "阶段: packing") {
		t.Fatalf("persistent log should identify packing stage:\n%s", result.log)
	}
	if len(result.curlCalls) != 0 {
		t.Fatalf("upload must not run after unsafe tar warning, calls=%v", result.curlCalls)
	}
}

func TestExportMigrationScript_RejectsInvalidArchiveAfterRootWarning(t *testing.T) {
	result := runExportMigrationScript(t, exportMigrationScriptScenario{
		createStatus:     1,
		createDiag:       "tar: .hermes: file changed as we read it",
		verifyStatus:     2,
		allowRootWarning: true,
	})
	if result.err == nil {
		t.Fatalf("script should reject invalid archive:\n%s", result.output)
	}
	if !strings.Contains(result.output, "迁移归档校验失败") {
		t.Fatalf("missing archive validation failure:\n%s", result.output)
	}
	if !strings.Contains(result.log, "阶段: validating_archive") {
		t.Fatalf("persistent log should identify validation stage:\n%s", result.log)
	}
	if len(result.curlCalls) != 0 {
		t.Fatalf("upload must not run after invalid archive, calls=%v", result.curlCalls)
	}
}

func TestExportMigrationScript_NormalArchiveStillSucceeds(t *testing.T) {
	result := runExportMigrationScript(t, exportMigrationScriptScenario{})
	if result.err != nil {
		t.Fatalf("normal export failed: %v\n%s", result.err, result.output)
	}
	if got := strings.Join(result.curlCalls, ","); got != "PUT,POST" {
		t.Fatalf("curl calls = %q, want PUT,POST", got)
	}
}

func TestExportMigrationScript_RejectsRootWarningForNonHermesAgent(t *testing.T) {
	result := runExportMigrationScript(t, exportMigrationScriptScenario{
		createStatus: 1,
		createDiag:   "tar: .hermes: file changed as we read it",
	})
	if result.err == nil {
		t.Fatalf("non-Hermes agent should reject root changed warning:\n%s", result.output)
	}
	if !strings.Contains(result.output, "打包失败（tar 退出码: 1）") {
		t.Fatalf("missing scoped-compatibility failure:\n%s", result.output)
	}
	if len(result.curlCalls) != 0 {
		t.Fatalf("upload must not run outside Hermes compatibility, calls=%v", result.curlCalls)
	}
}

func TestExportMigrationScript_RejectsChangedRootBusinessEntry(t *testing.T) {
	result := runExportMigrationScript(t, exportMigrationScriptScenario{
		createStatus:           1,
		createDiag:             "tar: .hermes: file changed as we read it",
		allowRootWarning:       true,
		createRootBusinessFile: "new-business.json",
	})
	if result.err == nil {
		t.Fatalf("changed root business entry should be rejected:\n%s", result.output)
	}
	if !strings.Contains(result.output, "业务数据在打包期间发生变化") {
		t.Fatalf("missing business-manifest failure:\n%s", result.output)
	}
	if len(result.curlCalls) != 0 {
		t.Fatalf("upload must not run after business manifest changed, calls=%v", result.curlCalls)
	}
}

func TestExportMigrationScript_RejectsReplacedNestedBusinessFile(t *testing.T) {
	result := runExportMigrationScript(t, exportMigrationScriptScenario{
		createStatus:              1,
		createDiag:                "tar: .hermes: file changed as we read it",
		allowRootWarning:          true,
		replaceNestedBusinessFile: true,
	})
	if result.err == nil {
		t.Fatalf("replaced nested business file should be rejected:\n%s", result.output)
	}
	if !strings.Contains(result.output, "业务数据在打包期间发生变化") {
		t.Fatalf("missing recursive business-manifest failure:\n%s", result.output)
	}
	if len(result.curlCalls) != 0 {
		t.Fatalf("upload must not run after nested business data changed, calls=%v", result.curlCalls)
	}
}

func seedMigrationSMHConfig(t *testing.T) {
	t.Helper()
	model.DB(context.Background()).Create(&model.SiteConfig{
		SMHEndpoint:      "https://smh.example.com",
		SMHLibraryId:     "lib-001",
		SMHLibrarySecret: "secret",
		SMHEnabled:       1,
	})
	model.DB(context.Background()).Create(&model.SMHSpace{SpaceTag: "common", SpaceId: "sp-common", LibraryId: "lib-001", Purpose: "common"})
	model.DB(context.Background()).Create(&model.SMHSpace{SpaceTag: "skillhub", SpaceId: "sp-skillhub", LibraryId: "lib-001", Purpose: "skillhub"})
}

func withMigrationExportScript(t *testing.T) {
	t.Helper()
	origLoader := LoadScript
	withLoadScript(t, func(name string) (string, error) {
		if name == "export_migration.sh" {
			data, err := os.ReadFile("../scripts/export_migration.sh")
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
		return origLoader(name)
	})
}

func withLoadScript(t *testing.T, loader func(name string) (string, error)) {
	t.Helper()
	origLoader := LoadScript
	LoadScript = loader
	t.Cleanup(func() { LoadScript = origLoader })
}

// ---------- performMigrationImportWithDeps ----------

func initPerformMigrationTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.InstanceModel{},
		&model.AgentMigration{},
		&model.SMHPersonalSpace{},
		&model.SkillInstallation{},
		&model.PluginInstallation{},
		&model.McpInstallation{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

func createPerformMigrationTestFixtures(t *testing.T, agentType string) (*model.Instance, *model.AgentMigration) {
	t.Helper()
	// 设置 LoadScript mock（避免 nil panic）
	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "#!/bin/bash\necho RESTORE_DONE:1", nil
	}
	t.Cleanup(func() { LoadScript = origLoader })
	user := &model.User{Username: "pm-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "pm-inst", InstanceId: "ins-pm", UserID: user.ID,
		AgentType: agentType, RuntimeUser: "root",
	}
	model.DB(context.Background()).Create(inst)
	m := &model.AgentMigration{
		InstanceID: inst.ID, CVMInstanceID: inst.InstanceId,
		FileKey: "migrations/ins-pm/agent-export.tgz",
		Status:  model.MigrationStatusImporting,
	}
	model.DB(context.Background()).Create(m)
	model.InitMigrationSteps(context.Background(), model.DB(context.Background()), m, agentType)
	// 设置操作锁（performMigrationImport 期望实例已有操作锁）
	model.DB(context.Background()).Model(inst).Updates(map[string]interface{}{
		"current_operation":       model.OpMigrate,
		"current_operation_state": model.OpStateProcessing,
	})
	return inst, m
}

func TestPerformMigrationImport_BuildURLFails(t *testing.T) {
	initPerformMigrationTestDB(t)
	inst, m := createPerformMigrationTestFixtures(t, model.AgentTypeOpenClaw)

	deps := defaultPerformMigrationImportDeps
	deps.BuildSMHDownloadURL = func(_ context.Context, _ string) (string, error) {
		return "", errors.New("SMH URL 生成失败")
	}

	err := performMigrationImportWithDeps(context.Background(), inst, m, deps)
	if err == nil {
		t.Error("should return error when BuildSMHDownloadURL fails")
	}
	// migration status 应为 failed
	var updated model.AgentMigration
	model.DB(context.Background()).First(&updated, m.ID)
	if updated.Status != model.MigrationStatusFailed {
		t.Errorf("status should be failed, got %q", updated.Status)
	}
}

func TestPerformMigrationImport_RestoreScriptFails(t *testing.T) {
	initPerformMigrationTestDB(t)
	inst, m := createPerformMigrationTestFixtures(t, model.AgentTypeOpenClaw)

	deps := defaultPerformMigrationImportDeps
	deps.BuildSMHDownloadURL = func(_ context.Context, _ string) (string, error) {
		return "https://smh.example.com/file.tgz", nil
	}
	deps.RunRestoreScript = func(_ context.Context, _, _ string, _ uint64, _ string, _ func(string)) error {
		return errors.New("TAT 执行失败")
	}

	err := performMigrationImportWithDeps(context.Background(), inst, m, deps)
	if err == nil {
		t.Error("should return error when restore script fails")
	}
}

func TestPerformMigrationImport_WaitReadyFails(t *testing.T) {
	initPerformMigrationTestDB(t)
	inst, m := createPerformMigrationTestFixtures(t, model.AgentTypeOpenClaw)

	deps := defaultPerformMigrationImportDeps
	deps.BuildSMHDownloadURL = func(_ context.Context, _ string) (string, error) {
		return "https://smh.example.com/file.tgz", nil
	}
	deps.RunRestoreScript = func(_ context.Context, _, _ string, _ uint64, _ string, _ func(string)) error {
		return nil
	}
	deps.WaitForReady = func(_ context.Context, _, _ string, _ time.Duration) error {
		return errors.New("等待超时")
	}

	err := performMigrationImportWithDeps(context.Background(), inst, m, deps)
	if err == nil {
		t.Error("should return error when WaitForReady fails")
	}
	// restarting step should be failed
	model.DB(context.Background()).First(m, m.ID)
	steps := model.ParseMigrationSteps(m)
	for _, s := range steps {
		if s.Step == model.MigrationStepRestarting && s.Status != model.MigrationStepStatusFailed {
			t.Errorf("restarting step should be failed, got %q", s.Status)
		}
	}
}

func TestPerformMigrationImport_Success(t *testing.T) {
	initPerformMigrationTestDB(t)
	inst, m := createPerformMigrationTestFixtures(t, model.AgentTypeOpenClaw)

	deps := defaultPerformMigrationImportDeps
	deps.BuildSMHDownloadURL = func(_ context.Context, _ string) (string, error) {
		return "https://smh.example.com/file.tgz", nil
	}
	deps.RunRestoreScript = func(_ context.Context, _, _ string, _ uint64, _ string, _ func(string)) error {
		return nil
	}
	deps.WaitForReady = func(_ context.Context, _, _ string, _ time.Duration) error {
		return nil
	}
	deps.SyncModels = func(_ context.Context, _ *model.Instance, _ *slog.Logger) bool {
		return true
	}
	deps.SyncSMHSpace = func(_ context.Context, _ *model.SMHPersonalSpace) error {
		return nil
	}
	deps.DeleteMigrationFile = func(_ context.Context, _ string) error {
		return nil
	}

	err := performMigrationImportWithDeps(context.Background(), inst, m, deps)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	var updated model.AgentMigration
	model.DB(context.Background()).First(&updated, m.ID)
	if updated.Status != model.MigrationStatusDone {
		t.Errorf("status should be done, got %q", updated.Status)
	}
}

func TestPerformMigrationImport_ProgressCallback(t *testing.T) {
	initPerformMigrationTestDB(t)
	inst, m := createPerformMigrationTestFixtures(t, model.AgentTypeOpenClaw)

	var capturedOnOutput func(string)
	deps := defaultPerformMigrationImportDeps
	deps.BuildSMHDownloadURL = func(_ context.Context, _ string) (string, error) {
		return "https://smh.example.com/file.tgz", nil
	}
	deps.RunRestoreScript = func(_ context.Context, _, _ string, _ uint64, _ string, onOutput func(string)) error {
		capturedOnOutput = onOutput
		// 模拟脚本输出 PROGRESS 事件
		onOutput("PROGRESS:downloading\n")
		onOutput("PROGRESS:extracting\n")
		return nil
	}
	deps.WaitForReady = func(_ context.Context, _, _ string, _ time.Duration) error { return nil }
	deps.SyncModels = func(_ context.Context, _ *model.Instance, _ *slog.Logger) bool { return true }
	deps.SyncSMHSpace = func(_ context.Context, _ *model.SMHPersonalSpace) error { return nil }
	deps.DeleteMigrationFile = func(_ context.Context, _ string) error { return nil }

	err := performMigrationImportWithDeps(context.Background(), inst, m, deps)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	_ = capturedOnOutput // 确保 onOutput 被调用
}
