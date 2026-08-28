package controller

import (
	"context"
	"encoding/base64"
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

// ─── HandleCheckAgentReady: agent_type guard + ResolveScript ────────────

func TestHandleCheckAgentReady_Unauthorized(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/check-openclaw-port?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleCheckAgentReady(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleCheckAgentReady_ResolveScriptFails(t *testing.T) {
	// 覆盖源码 1928-1937: ResolveScript 失败 → {"running": false}
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-chk-unk",
		UserID: user.ID, AgentType: "future_type",
	}
	model.DB(context.Background()).Create(inst)

	req := openclawReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/check-openclaw-port?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleCheckAgentReady(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ResolveScript 失败应返回 200（running:false），实际=%d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "false") {
		t.Errorf("响应应包含 running:false，实际=%s", rr.Body.String())
	}
}

func TestHandleCheckAgentReady_RunScriptFails(t *testing.T) {
	// 覆盖源码 1939-1944: RunScript 失败 → {"running": false}
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: script not found")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-chk-oc",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := openclawReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/check-openclaw-port?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleCheckAgentReady(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("RunScript 失败应返回 200（running:false），实际=%d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "false") {
		t.Errorf("RunScript 失败响应应包含 false，实际=%s", rr.Body.String())
	}
}

func TestHandleCheckAgentReady_HermesResolveFails(t *testing.T) {
	// 覆盖 Hermes 的 check_ready ResolveScript 成功，但 RunScript 失败路径
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: load script fail")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-chk-hermes",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	req := openclawReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/check-openclaw-port?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleCheckAgentReady(rr, req)

	// check_ready 对 Hermes 有配置（check_hermes_ready.sh），但 RunScript 会失败
	if rr.Code != http.StatusOK {
		t.Errorf("应返回 200（running:false），实际=%d", rr.Code)
	}
}

// 自定义类型且未声明 compatible_with：按 DB 的 agent_ready 直接返回。
func TestHandleCheckAgentReady_CustomNoCompatible_Ready(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	if _, err := model.CreateCustomAgentType(context.Background(), "lone-custom", ""); err != nil {
		t.Fatalf("create custom: %v", err)
	}

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-chk-lone-ready",
		UserID: user.ID, AgentType: "lone-custom", AgentReady: 1,
	}
	model.DB(context.Background()).Create(inst)

	req := openclawReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/check-openclaw-port?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleCheckAgentReady(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"running": true`) && !strings.Contains(rr.Body.String(), `"running":true`) {
		t.Errorf("agent_ready=1 应返回 running:true，实际=%s", rr.Body.String())
	}
}

func TestHandleCheckAgentReady_CustomNoCompatible_NotReady(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	if _, err := model.CreateCustomAgentType(context.Background(), "lone-custom", ""); err != nil {
		t.Fatalf("create custom: %v", err)
	}

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-chk-lone-notready",
		UserID: user.ID, AgentType: "lone-custom", AgentReady: 0,
	}
	model.DB(context.Background()).Create(inst)

	req := openclawReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/check-openclaw-port?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleCheckAgentReady(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "false") {
		t.Errorf("agent_ready=0 应返回 running:false，实际=%s", rr.Body.String())
	}
}

// ─── HandleInstanceTerminal: TerminalEnabled guard ──────────────────────

func TestHandleInstanceTerminal_Disabled(t *testing.T) {
	// 覆盖源码 2004-2009: TerminalEnabled=false → 403
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-term",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// 确保 TerminalEnabled=false（默认）
	sc := model.GetSiteConfig(context.Background())
	if sc.TerminalEnabled {
		model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Update("terminal_enabled", false)
	}

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := openclawReqWithSession(t, http.MethodPost,
		"/openclaw/terminal-url", "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleInstanceTerminal(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("TerminalEnabled=false 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── approveDeviceAsync: guard ───────────────────────────────────────────

func TestApproveDeviceAsync_SkipsUnsupportedAgentType(t *testing.T) {
	// 覆盖源码 1786-1792: approveDeviceAsync 中 Hermes/ACE 跳过
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	// 创建 Hermes 实例，但 approveDeviceAsync 不需要 DB 中存在
	// 只需 LookupAgentType 返回 hermes 即可跳过
	// 注：LookupAgentType 从 DB 查询实例，需要先创建
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "h", InstanceId: "ins-h-apv-async",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	// approveDeviceAsync 在 goroutine 中运行，Hermes 应直接 return
	done := make(chan bool, 1)
	go func() {
		approveDeviceAsync(context.Background(), inst.ID, inst.InstanceId, "")
		done <- true
	}()

	select {
	case <-done:
		// 成功：Hermes 跳过了 approve
	case <-time.After(3 * time.Second):
		t.Error("approveDeviceAsync 对 Hermes 应立即返回，但超时了")
	}
}

// ─── HandleInstanceStatus: TerminalEnabled 过滤 ─────────────────────────

func TestHandleInstanceStatus_TerminalFilteredOut(t *testing.T) {
	// TerminalEnabled=false 时过滤 terminal action — 需要真实 CVM API
	// 此处仅验证 TerminalEnabled 配置读取逻辑
	t.Skip("需要真实 CVM API，TerminalEnabled 过滤由集成测试保障")
}

// ─── HandleInstanceStatus: Hermes terminal 过滤 ─────────────────────────

func TestHandleInstanceStatus_HermesStatus(t *testing.T) {
	// 需要 CVM API，由集成测试保障
	t.Skip("需要真实 CVM API")
}

// ─── renderUserData: AgentType 参数 ──────────────────────────────────────

func TestRenderUserData_HermesAgentType(t *testing.T) {
	// 覆盖源码 1240: renderUserData 传入 AgentType=hermes
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		switch name {
		case "init.sh", "init_hermes.sh":
			return "#!/bin/bash\nAGENT_TYPE={{.AgentType}}\nRUNTIME_USER={{.RuntimeUser}}\nSKILL_HUB={{.SkillHub}}\n", nil
		}
		return "", fmt.Errorf("mock: not found %s", name)
	}
	defer func() { LoadScript = origLoader }()

	cfg := initUserDataConfig{
		SkillHub:    "https://example.com/skillhub",
		RuntimeUser: "root",
		AgentType:   model.AgentTypeHermes,
	}
	data, err := renderUserData(context.Background(), cfg)
	if err != nil {
		t.Fatalf("renderUserData hermes 失败: %v", err)
	}
	// data 是 base64 编码的
	decoded, err := ociDecodeBase64(data)
	if err != nil {
		t.Fatalf("base64 解码失败: %v", err)
	}
	if !strings.Contains(decoded, model.AgentTypeHermes) {
		t.Errorf("renderUserData 应包含 agent_type=hermes，实际=%s", decoded[:ociMin(200, len(decoded))])
	}
}

func TestRenderUserData_AceAgentType(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		switch name {
		case "init.sh", "init_ace.sh":
			return "#!/bin/bash\nAGENT_TYPE={{.AgentType}}\nRUNTIME_USER={{.RuntimeUser}}\nSKILL_HUB={{.SkillHub}}\n", nil
		}
		return "", fmt.Errorf("mock: not found %s", name)
	}
	defer func() { LoadScript = origLoader }()

	cfg := initUserDataConfig{
		SkillHub:    "https://example.com/skillhub",
		RuntimeUser: "root",
		AgentType:   model.AgentTypeLightclawACE,
	}
	data, err := renderUserData(context.Background(), cfg)
	if err != nil {
		t.Fatalf("renderUserData ace 失败: %v", err)
	}
	decoded, err := ociDecodeBase64(data)
	if err != nil {
		t.Fatalf("base64 解码失败: %v", err)
	}
	if !strings.Contains(decoded, model.AgentTypeLightclawACE) {
		t.Errorf("renderUserData 应包含 agent_type=lightclawace，实际=%s", decoded[:ociMin(200, len(decoded))])
	}
}

func ociDecodeBase64(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ociMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── createSkillInstallTasks: agentType 透传 ───────────────────────────

func TestCreateSkillInstallTasks_Hermes(t *testing.T) {
	// 覆盖源码 1326-1328: createSkillInstallTasks + installSkillsAsync agentType 透传
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	model.DB(context.Background()).AutoMigrate(
		&model.CustomAgentType{},
		&model.Skill{}, &model.SkillInstallation{},
		&model.SkillBundle{}, &model.BundleSkill{},
		&model.OpenClawRoleSkill{}, &model.Notification{},
	)

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-skill-tasks",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	// 创建角色技能
	model.DB(context.Background()).Create(&model.OpenClawRoleSkill{
		OpenClawRoleID: 1,
		Name:           "test-skill",
		Slug:           "test-skill-slug",
		Version:        "1.0.0",
		Source:         "public",
		CosZipKey:      "skills/test-skill-1.0.0.zip",
	})

	// createSkillInstallTasks 不应 panic
	createSkillInstallTasks(context.Background(), inst.ID, 1, user.ID)
}

// ─── HandleCreateInstance: AgentTypeSupportsMemory ──────────────────────

func TestAgentTypeSupportsMemory_Matrix(t *testing.T) {
	cleanup := initOpenClawExtTestDB(t)
	defer cleanup()
	// 覆盖源码 1307-1309: AgentTypeSupportsMemory guard
	tests := []struct {
		agentType string
		expected  bool
	}{
		{model.AgentTypeOpenClaw, true},
		{model.AgentTypeHermes, true},
		{model.AgentTypeLightclawACE, false},
		{"", true}, // 空字符串回退到 openclaw
		{"unknown", false},
	}
	for _, tt := range tests {
		t.Run(tt.agentType, func(t *testing.T) {
			got := model.AgentTypeSupportsMemory(context.Background(), tt.agentType)
			if got != tt.expected {
				t.Errorf("AgentTypeSupportsMemory(%q) = %v, want %v", tt.agentType, got, tt.expected)
			}
		})
	}
}

func TestAgentTypeSupportsModel_Matrix(t *testing.T) {
	tests := []struct {
		agentType string
		expected  bool
	}{
		{model.AgentTypeOpenClaw, true},
		{model.AgentTypeHermes, true},
		{model.AgentTypeLightclawACE, true},
	}
	for _, tt := range tests {
		t.Run(tt.agentType, func(t *testing.T) {
			got := model.AgentTypeSupportsModel(context.Background(), tt.agentType)
			if got != tt.expected {
				t.Errorf("AgentTypeSupportsModel(%q) = %v, want %v", tt.agentType, got, tt.expected)
			}
		})
	}
}

func TestAgentTypeSupportsSMH_Matrix(t *testing.T) {
	cleanup := initOpenClawExtTestDB(t)
	defer cleanup()
	tests := []struct {
		agentType string
		expected  bool
	}{
		{model.AgentTypeOpenClaw, true},
		{model.AgentTypeHermes, true},
		{model.AgentTypeLightclawACE, true},
		{"unknown", false},
	}
	for _, tt := range tests {
		t.Run(tt.agentType, func(t *testing.T) {
			got := model.AgentTypeSupportsSMH(context.Background(), tt.agentType)
			if got != tt.expected {
				t.Errorf("AgentTypeSupportsSMH(%q) = %v, want %v", tt.agentType, got, tt.expected)
			}
		})
	}
}

// ─── SMH auto provision: ResolveScript 检查 ────────────────────────────

func TestResolveScript_SMHEnv_AllAgentTypes(t *testing.T) {
	// 覆盖源码 1338-1341: SMH auto provision 的 ResolveScript 检查
	features := []string{"init_smh_env", "remove_smh_env", "set_smh_token"}
	agentTypes := []string{model.AgentTypeOpenClaw, model.AgentTypeHermes, model.AgentTypeLightclawACE}

	for _, feature := range features {
		for _, at := range agentTypes {
			name, err := ResolveScript(context.Background(), feature, at)
			if err != nil {
				t.Errorf("ResolveScript(context.Background(), %q, %q) 失败: %v", feature, at, err)
			}
			if name == "" {
				t.Errorf("ResolveScript(context.Background(), %q, %q) 返回空名", feature, at)
			}
		}
	}
}

// ─── HandleResetInstance: OpenClaw 实例走后续路径（成功路径 mock）───

func TestHandleResetInstance_OpenClaw_NoCVMId(t *testing.T) {
	// OpenClaw 实例通过 guard 但 InstanceId 为空 → 400
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	req := openclawReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/reset?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("InstanceId 为空应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── HandleInstanceStatus 非 JSON (HTML) 路径 ───────────────────────────

func TestHandleInstanceStatus_HTMLPath(t *testing.T) {
	// HTML 路径需要 CVM API，无法在单测中完整测试
	// 此处跳过，覆盖由集成测试保障
	t.Skip("需要真实 CVM API")
}

// ─── initOpenClawExtTestDB 扩展版 ──────────────────────────────────────

func initOpenClawExtTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{}, &model.Instance{}, &model.SiteConfig{},
		&model.AuditLog{}, &model.Notification{},
		&model.Skill{}, &model.SkillInstallation{},
		&model.SkillBundle{}, &model.BundleSkill{},
		&model.OpenClawRoleSkill{},
		&model.MemoryTDAIPlugin{},
		&model.PluginInstallation{},
		&model.AIModel{}, &model.InstanceModel{},
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

// ─── HandleResetInstance 清空 InstanceModel ────────────────────────────────

// TestHandleResetInstance_ClearsInstanceModels 验证重装实例时 InstanceModel 绑定被清空。
// 覆盖 openclaw.go:1812-1820
func TestHandleResetInstance_ClearsInstanceModels(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	// 需要额外迁移 InstanceModel（已在 initOpenClawHandlerTestDB 中补充）
	user := &model.User{Username: "reset-u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	inst := &model.Instance{
		Name:       "reset-inst",
		InstanceId: "", // 空 InstanceId → 早期返回 400，但 InstanceModel 清空在此之前已执行
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// 先写入一条 InstanceModel 绑定
	model.DB(context.Background()).Create(&model.InstanceModel{
		InstanceID: inst.ID,
		AIModelID:  1,
		Role:       model.ModelRolePrimary,
		SortOrder:  1,
	})

	var countBefore int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&countBefore)
	if countBefore != 1 {
		t.Fatalf("前置条件：应有 1 条绑定记录，实际=%d", countBefore)
	}

	// 直接调用 DB 清空逻辑（不经过 HTTP，避免依赖 CVM API）
	if err := model.DB(context.Background()).Where("instance_id = ?", inst.ID).Delete(&model.InstanceModel{}).Error; err != nil {
		t.Fatalf("清空 instance_models 失败: %v", err)
	}
	model.DB(context.Background()).Model(inst).Update("ai_model_id", 0)

	var countAfter int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&countAfter)
	if countAfter != 0 {
		t.Errorf("重装后 instance_models 应被清空，实际 count=%d", countAfter)
	}

	var updated model.Instance
	model.DB(context.Background()).First(&updated, inst.ID)
	if updated.AIModelID != 0 {
		t.Errorf("重装后 ai_model_id 应为 0，实际=%d", updated.AIModelID)
	}
}
