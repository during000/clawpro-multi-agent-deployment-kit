package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/gorilla/sessions"
)

// initPluginHandlerTestDB 扩展 plugin 测试 DB。
func initPluginHandlerTestDB(t *testing.T) func() {
	t.Helper()
	cleanup := initPluginTestDB(t)
	model.DB(context.Background()).AutoMigrate(
		&model.CustomAgentType{},
		&model.PluginBundle{},
		&model.BundlePlugin{},
		&model.OpenClawRolePlugin{},
		&model.Notification{},
	)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	return func() {
		Store = origStore
		cleanup()
	}
}

// ─── HandleAddPlugin 额外分支 ───────────────────────────────────────────

func TestHandleAddPlugin_Unauthorized(t *testing.T) {
	cleanup := initPluginHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/plugin", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handleAddPlugin(rr, req, testCVMFetcher)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleAddPlugin_MethodNotAllowed(t *testing.T) {
	cleanup := initPluginHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := pluginReqWithSession(t, http.MethodGet, "/openclaw/plugin", "u1", "")
	rr := httptest.NewRecorder()
	handleAddPlugin(rr, req, testCVMFetcher)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleAddPlugin_InstanceNotFound(t *testing.T) {
	cleanup := initPluginHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	form := url.Values{}
	form.Set("plugin", "some-plugin")
	req := pluginReqWithSession(t, http.MethodPost, "/openclaw/plugin?id=999", "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddPlugin(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAddPlugin_EmptyPluginName(t *testing.T) {
	cleanup := initPluginHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-plugin-empty",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{} // 无 plugin
	req := pluginReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/plugin?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddPlugin(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("空 plugin 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAddPlugin_InvalidPluginName(t *testing.T) {
	cleanup := initPluginHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-plugin-bad",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("plugin", "bad plugin name!") // 含空格和叹号
	req := pluginReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/plugin?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddPlugin(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 plugin 名应返回 400，实际=%d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "格式不合法") {
		t.Errorf("错误应含 '格式不合法'，实际=%s", rr.Body.String())
	}
}

func TestHandleAddPlugin_UnknownAgentType(t *testing.T) {
	cleanup := initPluginHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-plugin-unk",
		UserID: user.ID, AgentType: "future_unknown",
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("plugin", "valid-plugin")
	req := pluginReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/plugin?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddPlugin(rr, req, testCVMFetcher)
	if rr.Code != http.StatusForbidden {
		t.Errorf("未知 agent_type 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddPlugin_ValidPluginNameFormats(t *testing.T) {
	// 验证合法的插件名格式能通过正则校验，进入 RunScript 触发 LoadScript mock 失败
	cleanup := initPluginHandlerTestDB(t)
	defer cleanup()

	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: not found")
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-plugin-ok",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// @scope/pkg-name 形式
	form := url.Values{}
	form.Set("plugin", "@tencent/openclaw-weixin")
	req := pluginReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/plugin?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddPlugin(rr, req, testCVMFetcher)

	// 格式通过 → 进入 RunScript → LoadScript 失败 → 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("LoadScript 失败应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── createPluginInstallTasks 额外场景 ──────────────────────────────────

func TestCreatePluginInstallTasks_WithEnabledBundle(t *testing.T) {
	cleanup := initPluginHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-with-bundle",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// 创建启用的 bundle + 一个 plugin 条目
	bundle := &model.PluginBundle{Name: "built-in", Enabled: true}
	model.DB(context.Background()).Create(bundle)
	model.DB(context.Background()).Create(&model.BundlePlugin{
		PluginBundleID: bundle.ID,
		Name:           "auto-plugin",
		Slug:           "auto-plugin-slug",
		PluginID:       "p1",
		Version:        "1.0.0",
		InstallMode:    "npm",
	})

	createPluginInstallTasks(context.Background(), inst.ID, 0)

	var count int64
	model.DB(context.Background()).Model(&model.PluginInstallation{}).
		Where("instance_id = ? AND slug = ?", inst.ID, "auto-plugin-slug").
		Count(&count)
	if count != 1 {
		t.Errorf("应创建 1 条 PluginInstallation 记录，实际=%d", count)
	}
}

func TestCreatePluginInstallTasks_UpdatesExistingOnReinstall(t *testing.T) {
	// 已存在的 slug 应该被更新为 None 而非新建（重装场景）
	cleanup := initPluginHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-reinstall",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// 预创建一条 Failed 状态的记录
	preExisting := &model.PluginInstallation{
		InstanceID:    inst.ID,
		Name:          "old-name",
		Slug:          "my-slug",
		Version:       "0.9.0",
		InstallStatus: model.PluginInstallFailed,
		ErrorMessage:  "previous error",
	}
	model.DB(context.Background()).Create(preExisting)

	// 创建启用的 bundle，包含同 slug 的新版本
	bundle := &model.PluginBundle{Name: "built-in", Enabled: true}
	model.DB(context.Background()).Create(bundle)
	model.DB(context.Background()).Create(&model.BundlePlugin{
		PluginBundleID: bundle.ID,
		Name:           "updated-name",
		Slug:           "my-slug",
		PluginID:       "p1",
		Version:        "1.0.0",
		InstallMode:    "npm",
	})

	createPluginInstallTasks(context.Background(), inst.ID, 0)

	// 验证记录被更新
	var installation model.PluginInstallation
	model.DB(context.Background()).Where("instance_id = ? AND slug = ?", inst.ID, "my-slug").First(&installation)
	if installation.Name != "updated-name" {
		t.Errorf("name 应被更新为 updated-name，实际=%q", installation.Name)
	}
	if installation.Version != "1.0.0" {
		t.Errorf("version 应被更新为 1.0.0，实际=%q", installation.Version)
	}
	if installation.InstallStatus != model.PluginInstallNone {
		t.Errorf("status 应被重置为 None，实际=%v", installation.InstallStatus)
	}
	if installation.ErrorMessage != "" {
		t.Errorf("error_message 应被清空，实际=%q", installation.ErrorMessage)
	}

	// 确保未新建重复记录
	var count int64
	model.DB(context.Background()).Model(&model.PluginInstallation{}).
		Where("instance_id = ? AND slug = ?", inst.ID, "my-slug").Count(&count)
	if count != 1 {
		t.Errorf("应只有 1 条记录，实际=%d", count)
	}
}

func TestCreatePluginInstallTasks_RolePluginsTakePriority(t *testing.T) {
	cleanup := initPluginHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-role-prio",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	roleID := uint(42)
	// 创建角色插件
	model.DB(context.Background()).Create(&model.OpenClawRolePlugin{
		OpenClawRoleID: roleID,
		Name:           "role-plugin",
		Slug:           "shared-slug",
		PluginID:       "p-role",
		Version:        "2.0.0",
		InstallMode:    "npm",
	})

	// 创建启用 bundle，包含相同 slug（应被去重，以 role 为准）
	bundle := &model.PluginBundle{Name: "b1", Enabled: true}
	model.DB(context.Background()).Create(bundle)
	model.DB(context.Background()).Create(&model.BundlePlugin{
		PluginBundleID: bundle.ID,
		Name:           "bundle-plugin",
		Slug:           "shared-slug",
		PluginID:       "p-bundle",
		Version:        "1.0.0",
		InstallMode:    "npm",
	})

	createPluginInstallTasks(context.Background(), inst.ID, roleID)

	var installation model.PluginInstallation
	model.DB(context.Background()).Where("instance_id = ? AND slug = ?", inst.ID, "shared-slug").First(&installation)
	// 角色插件优先，应为 role-plugin
	if installation.Name != "role-plugin" {
		t.Errorf("应优先使用角色插件 role-plugin，实际=%q", installation.Name)
	}
}

// ─── installPluginsAsync：guard 分支 ────────────────────────────────────

func TestInstallPluginsAsync_UnsupportedType_Skips(t *testing.T) {
	// 防御式 guard（229-232 行）：agent_type 不支持插件时跳过
	cleanup := initPluginHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-hermes-plugin-async",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	// 即使显式创建 PluginInstallation，hermes 也应该被 guard 拦截不执行
	model.DB(context.Background()).Create(&model.PluginInstallation{
		InstanceID:    inst.ID,
		Name:          "p1",
		Slug:          "p1-slug",
		InstallStatus: model.PluginInstallNone,
	})

	// 直接调用 waitModeRetry 避免等待 CVM 状态
	installPluginsAsync(context.Background(), inst.ID, inst.InstanceId, waitModeRetry)

	// 记录状态不应被更新（因为 guard 拦截了）
	var installation model.PluginInstallation
	model.DB(context.Background()).Where("slug = ?", "p1-slug").First(&installation)
	if installation.InstallStatus != model.PluginInstallNone {
		t.Errorf("guard 拦截时状态不应被改动，实际=%v", installation.InstallStatus)
	}
}

func TestInstallPluginsAsync_InstanceNotFound_Skips(t *testing.T) {
	cleanup := initPluginHandlerTestDB(t)
	defer cleanup()

	// 不存在的 instance ID，不应 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("不应 panic，实际=%v", r)
		}
	}()
	installPluginsAsync(context.Background(), 99999, "ins-nonexistent", waitModeRetry)
}

func TestInstallPluginsAsync_NoPluginsToInstall(t *testing.T) {
	cleanup := initPluginHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-noplugins",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// 没有 PluginInstallation 记录 → 函数应优雅返回
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("不应 panic，实际=%v", r)
		}
	}()
	installPluginsAsync(context.Background(), inst.ID, inst.InstanceId, waitModeRetry)
}
