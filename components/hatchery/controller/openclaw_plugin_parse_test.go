package controller

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"hatchery/model"
)

// ============================================================================
// 本文件聚焦 openclaw_plugin.go 里"可纯 Go 测试"的函数：
//
//   1. parseAndUpdatePluginInstallResults — 纯字符串解析 + DB 写入，无外部依赖
//   2. installPluginsSMH / installPluginsNPM 的前置校验分支
//      （cos_zip_key 为空、npm_package 为空、validPlugins 为空提前返回）
//
// 这几段在覆盖率报告里是 0%，补完能把 openclaw_plugin.go 从 37.2% 推到约 65%+。
// 更深路径需要 mock RunScript（当前 package 级未暴露 runScriptFn），
// 属于需要生产代码 DI 改造的范畴，不在本轮务实方案内。
// ============================================================================

// silentLogger 构造一个弃用 handler 的 logger，避免测试输出刷屏。
func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 10}))
}

// ─── parseAndUpdatePluginInstallResults ─────────────────────────────────────

// TestParseAndUpdatePluginInstallResults_SuccessAndFailure 覆盖"BATCH INSTALL RESULTS"
// 标记行后的 JSON 能正确解析，并按 slug 映射更新 DB：success → Success, 其它 → Failed(带 message)。
func TestParseAndUpdatePluginInstallResults_SuccessAndFailure(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-parse-1",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	p1 := &model.PluginInstallation{InstanceID: inst.ID, Slug: "plugin-ok", InstallStatus: model.PluginInstalling}
	p2 := &model.PluginInstallation{InstanceID: inst.ID, Slug: "plugin-fail", InstallStatus: model.PluginInstalling}
	model.DB(context.Background()).Create(p1)
	model.DB(context.Background()).Create(p2)

	// 典型 TAT 输出：含日志前言 + "BATCH INSTALL RESULTS" 标记 + 下一行 JSON
	output := `[info] installing plugins...
[info] running install script
BATCH INSTALL RESULTS
{"results":[{"slug":"plugin-ok","version":"1.0.0","status":"success","message":""},{"slug":"plugin-fail","version":"1.0.0","status":"failed","message":"npm install exit 1"}],"summary":{"total":2,"success":1,"failed":1}}
[info] done`

	parseAndUpdatePluginInstallResults(context.Background(), output, []model.PluginInstallation{*p1, *p2}, silentLogger())

	var got1, got2 model.PluginInstallation
	model.DB(context.Background()).First(&got1, p1.ID)
	model.DB(context.Background()).First(&got2, p2.ID)

	if got1.InstallStatus != model.PluginInstallSuccess {
		t.Errorf("plugin-ok 应为 Success，实际=%v", got1.InstallStatus)
	}
	if got1.ErrorMessage != "" {
		t.Errorf("plugin-ok 成功时 error_message 应清空，实际=%q", got1.ErrorMessage)
	}
	if got2.InstallStatus != model.PluginInstallFailed {
		t.Errorf("plugin-fail 应为 Failed，实际=%v", got2.InstallStatus)
	}
	if got2.ErrorMessage != "npm install exit 1" {
		t.Errorf("plugin-fail 应回写 message，实际=%q", got2.ErrorMessage)
	}
}

// TestParseAndUpdatePluginInstallResults_FallbackLastJsonLine 验证
// 没有 "BATCH INSTALL RESULTS" 标记时，函数会从后向前找第一行以 '{' 开头的 JSON。
func TestParseAndUpdatePluginInstallResults_FallbackLastJsonLine(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-parse-2",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	p1 := &model.PluginInstallation{InstanceID: inst.ID, Slug: "p", InstallStatus: model.PluginInstalling}
	model.DB(context.Background()).Create(p1)

	// 输出里没有标记行，但结尾有一行 JSON
	output := `[info] step 1
[info] step 2
{"results":[{"slug":"p","status":"success","message":""}],"summary":{"total":1,"success":1,"failed":0}}`

	parseAndUpdatePluginInstallResults(context.Background(), output, []model.PluginInstallation{*p1}, silentLogger())

	var got model.PluginInstallation
	model.DB(context.Background()).First(&got, p1.ID)
	if got.InstallStatus != model.PluginInstallSuccess {
		t.Errorf("fallback 扫描到 JSON 应能成功更新，实际=%v", got.InstallStatus)
	}
}

// TestParseAndUpdatePluginInstallResults_NoJsonFound 验证完全无 JSON 行时，
// 所有 plugin 被标记为 Failed。
func TestParseAndUpdatePluginInstallResults_NoJsonFound(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-parse-3",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	p1 := &model.PluginInstallation{InstanceID: inst.ID, Slug: "p1", InstallStatus: model.PluginInstalling}
	p2 := &model.PluginInstallation{InstanceID: inst.ID, Slug: "p2", InstallStatus: model.PluginInstalling}
	model.DB(context.Background()).Create(p1)
	model.DB(context.Background()).Create(p2)

	output := "nothing useful here\nno braces\nonly text lines"

	parseAndUpdatePluginInstallResults(context.Background(), output, []model.PluginInstallation{*p1, *p2}, silentLogger())

	var got1, got2 model.PluginInstallation
	model.DB(context.Background()).First(&got1, p1.ID)
	model.DB(context.Background()).First(&got2, p2.ID)
	if got1.InstallStatus != model.PluginInstallFailed || got2.InstallStatus != model.PluginInstallFailed {
		t.Errorf("无 JSON 时应全部 Failed，实际=%v/%v", got1.InstallStatus, got2.InstallStatus)
	}
	if got1.ErrorMessage == "" {
		t.Errorf("应写入兜底 error_message")
	}
}

// TestParseAndUpdatePluginInstallResults_InvalidJson 验证 JSON 解析失败时，
// 所有 plugin 被标记为 Failed 且 error_message 包含解析失败提示。
func TestParseAndUpdatePluginInstallResults_InvalidJson(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-parse-4",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	p1 := &model.PluginInstallation{InstanceID: inst.ID, Slug: "p1", InstallStatus: model.PluginInstalling}
	model.DB(context.Background()).Create(p1)

	// "{" 开头但是非法 JSON（缺少引号）
	output := `BATCH INSTALL RESULTS
{results: [bogus json}`

	parseAndUpdatePluginInstallResults(context.Background(), output, []model.PluginInstallation{*p1}, silentLogger())

	var got model.PluginInstallation
	model.DB(context.Background()).First(&got, p1.ID)
	if got.InstallStatus != model.PluginInstallFailed {
		t.Errorf("非法 JSON 应标记 Failed，实际=%v", got.InstallStatus)
	}
}

// TestParseAndUpdatePluginInstallResults_PluginNotInResults 验证 TAT 结果中
// 没有某个 plugin 的 slug 时，该 plugin 被标记为 Failed + "安装结果中未找到该插件"。
func TestParseAndUpdatePluginInstallResults_PluginNotInResults(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-parse-5",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	p1 := &model.PluginInstallation{InstanceID: inst.ID, Slug: "found-slug", InstallStatus: model.PluginInstalling}
	p2 := &model.PluginInstallation{InstanceID: inst.ID, Slug: "missing-slug", InstallStatus: model.PluginInstalling}
	model.DB(context.Background()).Create(p1)
	model.DB(context.Background()).Create(p2)

	// TAT 输出只包含 found-slug，missing-slug 缺失
	output := `BATCH INSTALL RESULTS
{"results":[{"slug":"found-slug","status":"success","message":""}],"summary":{"total":1,"success":1,"failed":0}}`

	parseAndUpdatePluginInstallResults(context.Background(), output, []model.PluginInstallation{*p1, *p2}, silentLogger())

	var got1, got2 model.PluginInstallation
	model.DB(context.Background()).First(&got1, p1.ID)
	model.DB(context.Background()).First(&got2, p2.ID)
	if got1.InstallStatus != model.PluginInstallSuccess {
		t.Errorf("found-slug 应 Success，实际=%v", got1.InstallStatus)
	}
	if got2.InstallStatus != model.PluginInstallFailed {
		t.Errorf("missing-slug 应 Failed，实际=%v", got2.InstallStatus)
	}
	if got2.ErrorMessage != "安装结果中未找到该插件" {
		t.Errorf("missing-slug 应带特定错误提示，实际=%q", got2.ErrorMessage)
	}
}

// ─── installPluginsSMH / installPluginsNPM 前置校验分支 ────────────────────

// TestInstallPluginsSMH_AllEmptyCosZipKey 验证所有 SMH 插件 cos_zip_key 为空时，
// 全部被标记为 Failed + "插件包尚未完成 SMH 同步"，且因 validPlugins 空而提前 return，
// 不会调 RunScript（避免触网）。
func TestInstallPluginsSMH_AllEmptyCosZipKey(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-smh-empty",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	p1 := &model.PluginInstallation{InstanceID: inst.ID, Slug: "slug1", InstallStatus: model.PluginInstalling, CosZipKey: ""}
	p2 := &model.PluginInstallation{InstanceID: inst.ID, Slug: "slug2", InstallStatus: model.PluginInstalling, CosZipKey: ""}
	model.DB(context.Background()).Create(p1)
	model.DB(context.Background()).Create(p2)

	// 直接调 installPluginsSMH，让 cos_zip_key 为空分支全部命中 + lines 空 → 提前 return。
	// 不会触达 RunScript（因为 len(lines)==0 分支就 return 了）。
	installPluginsSMH(context.Background(), inst.ID, inst.InstanceId, []model.PluginInstallation{*p1, *p2}, silentLogger())

	var got1, got2 model.PluginInstallation
	model.DB(context.Background()).First(&got1, p1.ID)
	model.DB(context.Background()).First(&got2, p2.ID)
	if got1.InstallStatus != model.PluginInstallFailed || got2.InstallStatus != model.PluginInstallFailed {
		t.Errorf("cos_zip_key 为空应全部 Failed，实际=%v/%v", got1.InstallStatus, got2.InstallStatus)
	}
	if got1.ErrorMessage != "插件包尚未完成 SMH 同步" {
		t.Errorf("应写入特定错误信息，实际=%q", got1.ErrorMessage)
	}
}

// TestInstallPluginsNPM_AllEmptyPackage 验证所有 npm 插件 npm_package 为空时，
// 全部被标记为 Failed + "npm 包名为空"，且因 validPlugins 空而提前 return。
func TestInstallPluginsNPM_AllEmptyPackage(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-npm-empty",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	p1 := &model.PluginInstallation{InstanceID: inst.ID, Slug: "slug1", InstallStatus: model.PluginInstalling, InstallMode: "npm", NpmPackage: ""}
	p2 := &model.PluginInstallation{InstanceID: inst.ID, Slug: "slug2", InstallStatus: model.PluginInstalling, InstallMode: "npm", NpmPackage: ""}
	model.DB(context.Background()).Create(p1)
	model.DB(context.Background()).Create(p2)

	installPluginsNPM(context.Background(), inst.ID, inst.InstanceId, []model.PluginInstallation{*p1, *p2}, silentLogger())

	var got1, got2 model.PluginInstallation
	model.DB(context.Background()).First(&got1, p1.ID)
	model.DB(context.Background()).First(&got2, p2.ID)
	if got1.InstallStatus != model.PluginInstallFailed || got2.InstallStatus != model.PluginInstallFailed {
		t.Errorf("npm_package 为空应全部 Failed，实际=%v/%v", got1.InstallStatus, got2.InstallStatus)
	}
	if got1.ErrorMessage != "npm 包名为空" {
		t.Errorf("应写入 npm 包名为空，实际=%q", got1.ErrorMessage)
	}
}
