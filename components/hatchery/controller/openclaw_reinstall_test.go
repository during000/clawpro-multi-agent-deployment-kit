package controller

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	tccommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tcprofile "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// 测试基础设施：DB / mock / request 工具
// ---------------------------------------------------------------------------

// initReinstallCommonTestDB 初始化重装公共逻辑测试所需的内存 SQLite。
//
// 测试覆盖 commonHandleResetInstance 整条链路，会触发：
//   - getInstanceByID / getAdminInstanceByIDOrInstanceID 的查实例
//   - setOperationWithAgentReset 写实例状态
//   - prepareReinstallImage 查启用镜像
//   - resetInstanceVersionInfo 更新实例
//   - buildReinstallRequest 渲染 UserData（依赖 SiteConfig）
//   - resetReinstallBusinessState 事务（依赖 InstanceModel）
//   - kickOffReinstallAsyncTasks 后置 goroutine（依赖 SkillInstallation /
//     PluginInstallation / MemoryTDAIPlugin / OpenClawRole 等表）
//
// 所有可能被异步 goroutine 访问的表都要建好，避免 cleanup 后 goroutine
// 触发 "no such table" panic。
func initReinstallCommonTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// SQLite :memory: 每个连接都是独立 DB；限定 1 个连接，避免 AutoMigrate 建表
	// 后另一连接看不到表的问题。约定见 docs/testing.md。
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{}, &model.Instance{},
		&model.InstanceAdjustment{},
		&model.AIImage{}, &model.AIModel{},
		&model.SiteConfig{}, &model.AuditLog{}, &model.Notification{},
		&model.SkillInstallation{}, &model.PluginInstallation{},
		&model.SkillDistributionRecord{}, &model.PluginDistributionRecord{},
		&model.McpInstallation{},
		&model.InstanceModel{},
		&model.SMHPersonalSpace{},
		&model.MemoryTDAIPlugin{}, &model.RuleSet{},
		&model.GroupConfigBinding{}, &model.OpenClawRole{},
		&model.RoleVisibilityGroup{}, &model.UserGroup{}, &model.GroupClosure{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	origAdminToken := AdminToken
	AdminToken = "test-admin-token"

	// 替换异步通知为同步空实现，避免 cleanup 后 goroutine 触发"no such table"。
	origNotif := createErrorNotification
	createErrorNotification = func(uint, uint, string, string, string, error, context.Context) {}

	// 替换 SMH 同步入口为同步 no-op，避免依赖 TAT/STS 环境。
	origSyncSMH := syncSMHEnvWhenReadyFn
	syncSMHEnvWhenReadyFn = func(context.Context, model.Instance) {}

	return func() {
		// kickOffReinstallAsyncTasks 会 spawn 4 个 goroutine（installSkills /
		// installPlugins / approveDevice / syncSMHEnv），其中 approveDevice 内
		// 会调 NewTATClient → 在测试环境无凭据通常很快返回；installSkills /
		// installPlugins 在 SkillInstallation / PluginInstallation 表空时也会
		// 立即返回。给一点缓冲时间让它们退出，再恢复 DB。
		time.Sleep(100 * time.Millisecond)
		createErrorNotification = origNotif
		syncSMHEnvWhenReadyFn = origSyncSMH
		AdminToken = origAdminToken
		Store = origStore
		origDB()
	}
}

// reinstallCommonStubCVMClient 把 NewCVMClient 替换为返回未连接 client（用于
// 让流程穿过 NewCVMClient 一直走到 ResetInstance 失败的分支）。
func reinstallCommonStubCVMClient(t *testing.T) func() {
	t.Helper()
	orig := NewCVMClient
	NewCVMClient = func(context.Context) (*cvm.Client, error) {
		return &cvm.Client{}, nil
	}
	return func() { NewCVMClient = orig }
}

// reinstallCommonFailCVMClient 把 NewCVMClient 替换为始终返回错误的 mock，
// 用于覆盖 "创建 CVM 客户端失败" 分支。
func reinstallCommonFailCVMClient(t *testing.T) func() {
	t.Helper()
	orig := NewCVMClient
	NewCVMClient = func(context.Context) (*cvm.Client, error) {
		return nil, hcommon.I18nError(i18n.MsgCreateCVMClientFailed)
	}
	return func() { NewCVMClient = orig }
}

// reinstallCommonStubLoadScript 把 LoadScript 替换为始终失败的 mock，用于
// 触发 buildReinstallRequest → renderUserData 的"加载 init 脚本失败"分支。
func reinstallCommonStubLoadScript(t *testing.T) func() {
	t.Helper()
	orig := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: script %q not found", name)
	}
	return func() { LoadScript = orig }
}

// reinstallCommonOKLoadScript 把 LoadScript 替换为返回固定脚本内容，
// 让 SkillHub 分支下 buildUserData 渲染成功。模板内不放 {{.X}}，避免触发
// template Execute 阶段对未知字段访问错误。
func reinstallCommonOKLoadScript(t *testing.T) func() {
	t.Helper()
	orig := LoadScript
	LoadScript = func(string) (string, error) {
		return "#!/bin/bash\necho hello\n", nil
	}
	return func() { LoadScript = orig }
}

// userJSONPost 构造已登录用户的 JSON POST 请求。
func userJSONPost(t *testing.T, username, path, body string) *http.Request {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(http.MethodPost, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	sess, _ := Store.Get(req, "hatchery-session")
	sess.Values["username"] = username
	rr := httptest.NewRecorder()
	sess.Save(req, rr)
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

// adminTokenPost 构造带 admin token 的 POST 请求。
func adminTokenPost(_ *testing.T, path, body string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(http.MethodPost, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// runningResolver 返回 running 状态（允许 reboot/reinstall/delete/terminal）。
var runningResolver = &mockStatusResolverWithStatus{status: model.StatusRunning, label: "运行中"}

// errStatusResolver 让状态解析返回基础设施错误，用于覆盖
// writeAgentGuardError 中"非 ErrAgentNotAllowed → 500"的分支。
type errStatusResolver struct{}

func (errStatusResolver) ResolveStatus(context.Context, *model.Instance) (InstanceStatusResponse, error) {
	return InstanceStatusResponse{}, hcommon.I18nError(i18n.MsgQueryCVMInstanceFailed)
}

// ---------------------------------------------------------------------------
// userIDOf
// ---------------------------------------------------------------------------

func TestUserIDOf_Nil(t *testing.T) {
	if got := userIDOf(nil); got != 0 {
		t.Errorf("userIDOf(nil) = %d, want 0", got)
	}
}

func TestUserIDOf_NonNil(t *testing.T) {
	u := &model.User{}
	u.ID = 42
	if got := userIDOf(u); got != 42 {
		t.Errorf("userIDOf(user{ID:42}) = %d, want 42", got)
	}
}

// ---------------------------------------------------------------------------
// prepareReinstallImage
// ---------------------------------------------------------------------------

func TestPrepareReinstallImage_NoImage(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	inst := &model.Instance{AgentType: model.AgentTypeOpenClaw}
	ctx := context.Background()
	img, err := prepareReinstallImage(ctx, inst)
	if err == nil {
		t.Fatalf("无启用镜像应返回错误，但 err=nil img=%v", img)
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(ctx, err), "未为") {
		t.Errorf("错误信息应提示镜像未配置，实际=%v", err)
	}
}

func TestPrepareReinstallImage_Mismatch(t *testing.T) {
	// 实例类型 hermes，仅有 legacy（空类型）启用镜像 →
	// GetEnabledImageByType 命中回退，但 verifyReinstallImageMatches 拒绝。
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId: "img-legacy", ImageName: "legacy", ImageType: "PRIVATE_IMAGE",
		AgentType: "", AgentVersion: "1.0.0", Enabled: true,
	})

	inst := &model.Instance{AgentType: model.AgentTypeHermes}
	if _, err := prepareReinstallImage(context.Background(), inst); err == nil {
		t.Fatal("镜像类型不匹配应返回错误")
	}
}

func TestPrepareReinstallImage_Success(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId: "img-ok", ImageName: "ok", ImageType: "PRIVATE_IMAGE",
		AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
	})

	inst := &model.Instance{AgentType: model.AgentTypeOpenClaw}
	img, err := prepareReinstallImage(context.Background(), inst)
	if err != nil {
		t.Fatalf("应成功返回镜像，err=%v", err)
	}
	if img == nil || img.ImageId != "img-ok" {
		t.Errorf("应返回 img-ok，实际=%+v", img)
	}
}

// ---------------------------------------------------------------------------
// buildReinstallRequest
// ---------------------------------------------------------------------------

func TestBuildReinstallRequest_NoSkillHub_NoUserData(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	inst := &model.Instance{InstanceId: "ins-1", AgentType: model.AgentTypeOpenClaw}
	img := &model.AIImage{ImageId: "img-1"}
	req, err := buildReinstallRequest(context.Background(), inst, img)
	if err != nil {
		t.Fatalf("无 SkillHub + 无 UserData 应成功，err=%v", err)
	}
	if req.InstanceId == nil || *req.InstanceId != "ins-1" {
		t.Errorf("InstanceId 应为 ins-1，实际=%v", req.InstanceId)
	}
	if req.ImageId == nil || *req.ImageId != "img-1" {
		t.Errorf("ImageId 应为 img-1，实际=%v", req.ImageId)
	}
	if req.EnhancedService != nil {
		t.Errorf("无 UserData 时不应设 EnhancedService")
	}
	if req.UserData != nil {
		t.Errorf("无 UserData 时不应设 UserData")
	}
}

func TestBuildReinstallRequest_NoSkillHub_WithUserData(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	userPart := base64.StdEncoding.EncodeToString([]byte("#!/bin/bash\necho user\n"))
	inst := &model.Instance{
		InstanceId: "ins-2", AgentType: model.AgentTypeOpenClaw, UserData: userPart,
	}
	img := &model.AIImage{ImageId: "img-2"}
	req, err := buildReinstallRequest(context.Background(), inst, img)
	if err != nil {
		t.Fatalf("仅用户 UserData 应成功，err=%v", err)
	}
	if req.EnhancedService == nil || req.EnhancedService.AutomationService == nil ||
		req.EnhancedService.AutomationService.Enabled == nil ||
		*req.EnhancedService.AutomationService.Enabled != true {
		t.Errorf("应开启 AutomationService，实际=%+v", req.EnhancedService)
	}
	if req.UserData == nil || *req.UserData == "" {
		t.Errorf("应携带渲染后的 UserData")
	}
}

func TestBuildReinstallRequest_WithSkillHub_Success(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()
	restore := reinstallCommonOKLoadScript(t)
	defer restore()
	model.DB(context.Background()).Create(&model.SiteConfig{SkillHub: "https://hub.example"})

	inst := &model.Instance{
		InstanceId: "ins-3", AgentType: model.AgentTypeOpenClaw,
	}
	img := &model.AIImage{ImageId: "img-3"}
	req, err := buildReinstallRequest(context.Background(), inst, img)
	if err != nil {
		t.Fatalf("SkillHub 渲染应成功，err=%v", err)
	}
	if req.UserData == nil || *req.UserData == "" {
		t.Errorf("SkillHub 模式下应携带渲染后的 UserData")
	}
	if req.EnhancedService == nil {
		t.Errorf("应启用 AutomationService")
	}
}

func TestBuildReinstallRequest_WithSkillHub_ScriptFailed(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()
	restore := reinstallCommonStubLoadScript(t)
	defer restore()
	model.DB(context.Background()).Create(&model.SiteConfig{SkillHub: "https://hub.example"})

	inst := &model.Instance{InstanceId: "ins-4", AgentType: model.AgentTypeOpenClaw}
	img := &model.AIImage{ImageId: "img-4"}
	if _, err := buildReinstallRequest(context.Background(), inst, img); err == nil {
		t.Fatal("LoadScript 失败应返回错误")
	}
}

func TestBuildReinstallRequest_InvalidUserData(t *testing.T) {
	// 用户 UserData 非合法 base64 → buildUserData 返回错误。
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		InstanceId: "ins-5", AgentType: model.AgentTypeOpenClaw,
		UserData: "!!!not-base64!!!",
	}
	img := &model.AIImage{ImageId: "img-5"}
	if _, err := buildReinstallRequest(context.Background(), inst, img); err == nil {
		t.Fatal("非法 UserData 应返回错误")
	}
}

// ---------------------------------------------------------------------------
// resetReinstallBusinessState
// ---------------------------------------------------------------------------

func TestResetReinstallBusinessState_OnlyAIModel(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	now := time.Now()
	inst := &model.Instance{
		Name: "rrbs", InstanceId: "ins-rrbs",
		AgentType:          model.AgentTypeOpenClaw,
		AIModelID:          7,
		CLSAgentStatus:     1,
		AgentVersion:       "1.2.3",
		PluginVersionsJSON: `{"a":"1"}`,
		VersionFetchedAt:   &now,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.InstanceModel{
		InstanceID: inst.ID, AIModelID: 7, Role: model.ModelRolePrimary, SortOrder: 1,
	})

	if err := resetReinstallBusinessState(context.Background(), inst, false); err != nil {
		t.Fatalf("应成功，err=%v", err)
	}

	var got model.Instance
	model.DB(context.Background()).First(&got, inst.ID)
	if got.AIModelID != 0 {
		t.Errorf("AIModelID 应清空，实际=%d", got.AIModelID)
	}
	// resetVersionInfo=false 时版本字段保持原值
	if got.AgentVersion != "1.2.3" || got.PluginVersionsJSON != `{"a":"1"}` {
		t.Errorf("resetVersionInfo=false 时版本字段不应被清空，实际=%+v", got)
	}

	var imCount int64
	model.DB(context.Background()).Unscoped().Model(&model.InstanceModel{}).
		Where("instance_id = ?", inst.ID).Count(&imCount)
	if imCount != 0 {
		t.Errorf("InstanceModel 应物理删除，剩余=%d", imCount)
	}
}

func TestResetReinstallBusinessState_WithVersion(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	now := time.Now()
	inst := &model.Instance{
		Name: "rrbs-v", InstanceId: "ins-rrbs-v",
		AgentType:          model.AgentTypeOpenClaw,
		AIModelID:          9,
		CLSAgentStatus:     1,
		AgentVersion:       "9.9.9",
		PluginVersionsJSON: `{"b":"2"}`,
		VersionFetchedAt:   &now,
	}
	model.DB(context.Background()).Create(inst)

	if err := resetReinstallBusinessState(context.Background(), inst, true); err != nil {
		t.Fatalf("应成功，err=%v", err)
	}

	var got model.Instance
	model.DB(context.Background()).First(&got, inst.ID)
	if got.AIModelID != 0 || got.CLSAgentStatus != 0 ||
		got.AgentVersion != "" || got.PluginVersionsJSON != "" ||
		got.VersionFetchedAt != nil {
		t.Errorf("resetVersionInfo=true 时版本字段应全部清空，实际=%+v", got)
	}
}

// TestResetReinstallBusinessState_UpdateError 通过关闭底层 *sql.DB
// 让 Updates 调用返回错误，覆盖 "tx.Model(...).Updates(...)" 失败的早期返回分支。
func TestResetReinstallBusinessState_UpdateError(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "rrbs-err", InstanceId: "ins-rrbs-err",
		AgentType: model.AgentTypeOpenClaw}
	model.DB(context.Background()).Create(inst)

	// 拿到底层 *sql.DB 关闭 → 后续 Transaction/Updates 失败。
	sqlDB, err := model.DB(context.Background()).DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql.DB: %v", err)
	}

	if err := resetReinstallBusinessState(context.Background(), inst, true); err == nil {
		t.Fatal("DB 关闭后应返回错误")
	}
}

// ---------------------------------------------------------------------------
// kickOffReinstallAsyncTasks
// ---------------------------------------------------------------------------

// TestKickOffReinstallAsyncTasks_Smoke 覆盖 kickOffReinstallAsyncTasks 全部
// 同步路径：memory 状态重置、清空旧 SkillInstallation/PluginInstallation 并
// 创建新任务，并验证它能 spawn 4 个 goroutine 而不 panic。
//
// 异步 goroutine 内部因测试环境无 TAT/CVM 凭据会快速失败/早期返回，
// initReinstallCommonTestDB cleanup 里的 100ms sleep 已足够它们退出。
func TestKickOffReinstallAsyncTasks_Smoke(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "kick", InstanceId: "ins-kick",
		AgentType: model.AgentTypeOpenClaw, UserID: 1,
	}
	model.DB(context.Background()).Create(inst)

	// 预置一条 memory plugin，验证 status 被重置。
	model.DB(context.Background()).Create(&model.MemoryTDAIPlugin{
		InstanceID: inst.InstanceId, Status: "INSTALLED",
		RetryCount: 5, LastError: "old-error",
	})

	// 预置旧的 SkillInstallation / PluginInstallation，验证被清空。
	model.DB(context.Background()).Create(&model.SkillInstallation{
		InstanceID: inst.ID, Name: "old-skill", Slug: "old-skill",
	})
	model.DB(context.Background()).Create(&model.PluginInstallation{
		InstanceID: inst.ID, Slug: "old-plugin", Name: "old",
	})

	// 预置一条管控端企业技能下发成功记录，验证重装后被软删除，
	// 否则 /admin/skills/instances 会把该实例算成 installed/outdated 而无法重新下发。
	model.DB(context.Background()).Create(&model.SkillDistributionRecord{
		InstanceID: inst.ID, InstanceCID: inst.InstanceId,
		SkillID: 1, Version: "1.0.0",
		Status: model.RecordStatusSuccess, Type: model.TaskTypeDistribute,
	})

	// 预置一条管控端企业插件下发成功记录，验证重装后被软删除，
	// 否则 /admin/plugins/instances 会把该实例算成 installed 而无法重新下发。
	model.DB(context.Background()).Create(&model.PluginDistributionRecord{
		InstanceID: inst.ID, InstanceCID: inst.InstanceId,
		PluginDBID: 1, Version: "1.0.0", Status: "success",
	})

	// 预置一条 MCP 安装记录，验证重装后被清空，
	// 否则 /admin/mcp/instances 会把该实例算成 installed/outdated 而无法重新下发。
	model.DB(context.Background()).Create(&model.McpInstallation{
		InstanceID: inst.ID, MCPID: 1, ServiceID: "svc-1", Name: "old-mcp",
		Version: "1.0.0", InstallStatus: model.McpInstallSuccess,
	})

	kickOffReinstallAsyncTasks(context.Background(), inst, "[test]")

	var skillCnt int64
	model.DB(context.Background()).Model(&model.SkillInstallation{}).
		Where("instance_id = ?", inst.ID).Count(&skillCnt)
	if skillCnt != 0 {
		t.Errorf("SkillInstallation 应被清空（再由 createSkillInstallTasks 按需重建），剩余=%d", skillCnt)
	}

	// 企业技能下发记录应被软删除（默认查询不到），从而实例状态回到 uninstalled。
	var distCnt int64
	model.DB(context.Background()).Model(&model.SkillDistributionRecord{}).
		Where("instance_id = ?", inst.ID).Count(&distCnt)
	if distCnt != 0 {
		t.Errorf("SkillDistributionRecord 应被软删除（实例状态回到 uninstalled），剩余=%d", distCnt)
	}

	// 企业插件下发记录应被软删除，实例状态回到 uninstalled。
	var pluginDistCnt int64
	model.DB(context.Background()).Model(&model.PluginDistributionRecord{}).
		Where("instance_id = ?", inst.ID).Count(&pluginDistCnt)
	if pluginDistCnt != 0 {
		t.Errorf("PluginDistributionRecord 应被软删除（实例状态回到 uninstalled），剩余=%d", pluginDistCnt)
	}

	// MCP 安装记录应被清空，实例状态回到 uninstalled。
	var mcpCnt int64
	model.DB(context.Background()).Model(&model.McpInstallation{}).
		Where("instance_id = ?", inst.ID).Count(&mcpCnt)
	if mcpCnt != 0 {
		t.Errorf("McpInstallation 应被清空（实例状态回到 uninstalled），剩余=%d", mcpCnt)
	}

	var pluginCnt int64
	model.DB(context.Background()).Model(&model.PluginInstallation{}).
		Where("instance_id = ?", inst.ID).Count(&pluginCnt)
	if pluginCnt != 0 {
		t.Errorf("PluginInstallation 应被清空，剩余=%d", pluginCnt)
	}

	var plug model.MemoryTDAIPlugin
	if err := model.DB(context.Background()).Where("instance_id = ?", inst.InstanceId).
		First(&plug).Error; err != nil {
		t.Fatalf("memory plugin 应仍存在，err=%v", err)
	}
	if plug.Status != model.MemoryTDAIPluginStatusNotInstalled || plug.RetryCount != 0 || plug.LastError != "" {
		t.Errorf("memory plugin 应被重置为初始状态，实际=%+v", plug)
	}
}

// ---------------------------------------------------------------------------
// clearEnterpriseDistributionRecords
// ---------------------------------------------------------------------------

// TestClearEnterpriseDistributionRecords 单独验证抽取出的清空函数：
// 三类管控端企业下发记录（SkillDistributionRecord / PluginDistributionRecord /
// McpInstallation）都应被清空，使重装后实例在管控端下发页恢复 uninstalled，支持再次下发。
func TestClearEnterpriseDistributionRecords(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "clear-dist", InstanceId: "ins-clear-dist",
		AgentType: model.AgentTypeOpenClaw, UserID: 1,
	}
	model.DB(context.Background()).Create(inst)

	// 另一个实例的记录，验证按 instance_id 精确清空，不误删他人记录。
	other := &model.Instance{
		Name: "clear-dist-other", InstanceId: "ins-clear-dist-other",
		AgentType: model.AgentTypeOpenClaw, UserID: 1,
	}
	model.DB(context.Background()).Create(other)

	model.DB(context.Background()).Create(&model.SkillDistributionRecord{
		InstanceID: inst.ID, InstanceCID: inst.InstanceId,
		SkillID: 1, Version: "1.0.0",
		Status: model.RecordStatusSuccess, Type: model.TaskTypeDistribute,
	})
	model.DB(context.Background()).Create(&model.PluginDistributionRecord{
		InstanceID: inst.ID, InstanceCID: inst.InstanceId,
		PluginDBID: 1, Version: "1.0.0", Status: "success",
	})
	model.DB(context.Background()).Create(&model.McpInstallation{
		InstanceID: inst.ID, MCPID: 1, ServiceID: "svc-1", Name: "old-mcp",
		Version: "1.0.0", InstallStatus: model.McpInstallSuccess,
	})

	// 他人记录，函数执行后应原样保留。
	model.DB(context.Background()).Create(&model.SkillDistributionRecord{
		InstanceID: other.ID, InstanceCID: other.InstanceId,
		SkillID: 2, Version: "1.0.0",
		Status: model.RecordStatusSuccess, Type: model.TaskTypeDistribute,
	})

	clearEnterpriseDistributionRecords(context.Background(), inst, "[test]")

	var skillDistCnt, pluginDistCnt, mcpCnt int64
	model.DB(context.Background()).Model(&model.SkillDistributionRecord{}).
		Where("instance_id = ?", inst.ID).Count(&skillDistCnt)
	if skillDistCnt != 0 {
		t.Errorf("SkillDistributionRecord 应被清空，剩余=%d", skillDistCnt)
	}
	model.DB(context.Background()).Model(&model.PluginDistributionRecord{}).
		Where("instance_id = ?", inst.ID).Count(&pluginDistCnt)
	if pluginDistCnt != 0 {
		t.Errorf("PluginDistributionRecord 应被清空，剩余=%d", pluginDistCnt)
	}
	model.DB(context.Background()).Model(&model.McpInstallation{}).
		Where("instance_id = ?", inst.ID).Count(&mcpCnt)
	if mcpCnt != 0 {
		t.Errorf("McpInstallation 应被清空，剩余=%d", mcpCnt)
	}

	// 他人记录不应被波及。
	var otherCnt int64
	model.DB(context.Background()).Model(&model.SkillDistributionRecord{}).
		Where("instance_id = ?", other.ID).Count(&otherCnt)
	if otherCnt != 1 {
		t.Errorf("不应误删其它实例的下发记录，other 剩余=%d", otherCnt)
	}
}

// TestClearEnterpriseDistributionRecords_NonBlockingOnDBError 验证底层 DB 关闭后，
// 三次删除均失败但函数不 panic、按"非阻塞"语义安全返回。
func TestClearEnterpriseDistributionRecords_NonBlockingOnDBError(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "clear-dist-err", InstanceId: "ins-clear-dist-err",
		AgentType: model.AgentTypeOpenClaw, UserID: 1,
	}
	model.DB(context.Background()).Create(inst)

	sqlDB, err := model.DB(context.Background()).DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql.DB: %v", err)
	}

	// DB 已关闭，三次删除都会失败；函数应只告警、安全返回，不 panic。
	clearEnterpriseDistributionRecords(context.Background(), inst, "[test]")
}

// ---------------------------------------------------------------------------
// commonHandleResetInstance —— 主流程分支覆盖
// ---------------------------------------------------------------------------

// ── 1. method check ─────────────────────────────────────────────────

func TestCommon_MethodNotAllowed_User(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/reset", nil)
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallUserOpts)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应 405，实际=%d", rr.Code)
	}
}

func TestCommon_MethodNotAllowed_Admin(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/admin/instances/reset", nil)
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallAdminOpts)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应 405，实际=%d", rr.Code)
	}
}

// ── 2. 身份 / 实例查询 ───────────────────────────────────────────────

func TestCommon_Admin_Unauthorized(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	form := url.Values{"id": {"1"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/reset", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallAdminOpts)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("无 admin token 应 401/403，实际=%d", rr.Code)
	}
}

func TestCommon_User_Unauthorized(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	form := url.Values{"id": {"1"}}
	req := httptest.NewRequest(http.MethodPost, "/openclaw/reset", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallUserOpts)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestCommon_User_InstanceNotFound(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	form := url.Values{"id": {"9999"}}
	req := userJSONPost(t, "u1", "/openclaw/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallUserOpts)
	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在应 404，实际=%d", rr.Code)
	}
}

func TestCommon_Admin_InstanceNotFound(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	form := url.Values{"id": {"9999"}}
	req := adminTokenPost(t, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallAdminOpts)
	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在应 404，实际=%d", rr.Code)
	}
}

// ── 3. 龙虾医生节点 ─────────────────────────────────────────────────

func TestCommon_User_DoctorNode(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "doc", InstanceId: "ins-doc",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw, IsDoctorNode: true,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := userJSONPost(t, "u1", "/openclaw/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallUserOpts)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("龙虾医生节点应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCommon_Admin_DoctorNode(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "doc-adm", InstanceId: "ins-doc-adm",
		UserID: 1, AgentType: model.AgentTypeOpenClaw, IsDoctorNode: true,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := adminTokenPost(t, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallAdminOpts)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("龙虾医生节点应 400，实际=%d", rr.Code)
	}
}

// ── 4. 类型 guard ───────────────────────────────────────────────────

func TestCommon_User_UnsupportedAgentType(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "x", InstanceId: "ins-x",
		UserID: user.ID, AgentType: "no-such-type",
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := userJSONPost(t, "u1", "/openclaw/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallUserOpts)
	if rr.Code != http.StatusForbidden {
		t.Errorf("不支持的类型应 403，实际=%d", rr.Code)
	}
}

// ── 5. 状态 guard ───────────────────────────────────────────────────

func TestCommon_User_StatusDenied_JSON_409(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "stp", InstanceId: "ins-stp",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := userJSONPost(t, "u1", "/openclaw/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, stoppedResolver, reinstallUserOpts)
	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态用户重装应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCommon_User_StatusInfraError_500(t *testing.T) {
	// 状态解析返回基础设施错误（非 ErrAgentNotAllowed）→ writeAgentGuardError → 500
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "stp-i", InstanceId: "ins-stp-i",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := userJSONPost(t, "u1", "/openclaw/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, errStatusResolver{}, reinstallUserOpts)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("基础设施错误应 500，实际=%d", rr.Code)
	}
}

func TestCommon_Admin_StatusDenied(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "adm-creating", InstanceId: "ins-adm-creating",
		UserID: 1, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	creatingResolver := &mockStatusResolverWithStatus{status: model.StatusCreating, label: "创建中"}

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := adminTokenPost(t, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, creatingResolver, reinstallAdminOpts)
	if rr.Code != http.StatusConflict {
		t.Errorf("creating 状态 admin 重装应 409，实际=%d", rr.Code)
	}
}

// ── 6. CVM 关联 guard ───────────────────────────────────────────────

func TestCommon_User_NoCVMId(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "no-cvm", InstanceId: "",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := userJSONPost(t, "u1", "/openclaw/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallUserOpts)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("无 CVM 关联应 400，实际=%d", rr.Code)
	}
}

func TestCommon_Admin_NoCVMId(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "no-cvm-a", InstanceId: "",
		UserID: 1, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := adminTokenPost(t, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallAdminOpts)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("admin 无 CVM 应 400，实际=%d", rr.Code)
	}
}

// ── 7. 乐观锁并发冲突 ──────────────────────────────────────────────

func TestCommon_User_ConcurrentConflict_JSON(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "busy", InstanceId: "ins-busy",
		UserID:                user.ID,
		AgentType:             model.AgentTypeOpenClaw,
		CurrentOperation:      model.OpDelete,
		CurrentOperationState: model.OpStateProcessing,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := userJSONPost(t, "u1", "/openclaw/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallUserOpts)
	if rr.Code != http.StatusConflict {
		t.Errorf("并发冲突 JSON 应 409，实际=%d", rr.Code)
	}
}

// ── 8. 启用镜像缺失 ─────────────────────────────────────────────────

func TestCommon_User_NoEnabledImage(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "no-img", InstanceId: "ins-no-img",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := userJSONPost(t, "u1", "/openclaw/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallUserOpts)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("无启用镜像应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 操作标记应被回写为 failed（clearOperation 已执行）
	var got model.Instance
	model.DB(context.Background()).First(&got, inst.ID)
	if got.CurrentOperation != model.OpNone {
		t.Errorf("失败后 current_operation 应被清空，实际=%q", got.CurrentOperation)
	}
}

// ── 9. NewCVMClient 失败 ───────────────────────────────────────────

func TestCommon_User_CVMClientFailed(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()
	restoreCVM := reinstallCommonFailCVMClient(t)
	defer restoreCVM()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "no-cli", InstanceId: "ins-no-cli",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId: "img-cli", ImageName: "x", ImageType: "PRIVATE_IMAGE",
		AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
	})

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := userJSONPost(t, "u1", "/openclaw/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallUserOpts)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("NewCVMClient 失败应 500，实际=%d", rr.Code)
	}
}

func TestCommon_Admin_CVMClientFailed(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()
	restoreCVM := reinstallCommonFailCVMClient(t)
	defer restoreCVM()

	inst := &model.Instance{
		Name: "no-cli-a", InstanceId: "ins-no-cli-a",
		UserID: 1, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId: "img-cli-a", ImageName: "x", ImageType: "PRIVATE_IMAGE",
		AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
	})

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := adminTokenPost(t, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallAdminOpts)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Admin NewCVMClient 失败应 500，实际=%d", rr.Code)
	}
}

// ── 10. buildReinstallRequest 失败 ─────────────────────────────────

func TestCommon_User_BuildRequestFailed(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()
	restoreCVM := reinstallCommonStubCVMClient(t)
	defer restoreCVM()
	restoreScript := reinstallCommonStubLoadScript(t)
	defer restoreScript()
	model.DB(context.Background()).Create(&model.SiteConfig{SkillHub: "https://hub.example"})

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "br", InstanceId: "ins-br",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId: "img-br", ImageName: "br", ImageType: "PRIVATE_IMAGE",
		AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
	})

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := userJSONPost(t, "u1", "/openclaw/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallUserOpts)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("LoadScript 失败应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ── 11. CVM ResetInstance 失败 + 触发 createErrorNotification ──────

func TestCommon_User_ResetInstanceCVMFailed(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()
	restoreCVM := reinstallCommonStubCVMClient(t)
	defer restoreCVM()

	// 替换 createErrorNotification 为可观测同步版本，覆盖 "实例重装失败" 通知分支。
	var notified atomic.Int32
	origNotif := createErrorNotification
	createErrorNotification = func(_, _ uint, _, notifyType, _ string, _ error, _ context.Context) {
		if notifyType == model.NotifyTypeInstanceReinstallFailed {
			notified.Add(1)
		}
	}
	defer func() { createErrorNotification = origNotif }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "ri-fail", InstanceId: "ins-ri-fail",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId: "img-ri-fail", ImageName: "x", ImageType: "PRIVATE_IMAGE",
		AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
	})

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := userJSONPost(t, "u1", "/openclaw/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallUserOpts)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ResetInstance 失败应 500，实际=%d", rr.Code)
	}
	// goroutine 异步通知：等一下再断言
	time.Sleep(50 * time.Millisecond)
	if notified.Load() != 1 {
		t.Errorf("应触发 1 次重装失败通知，实际=%d", notified.Load())
	}
}

func TestCommon_Admin_ResetInstanceCVMFailed(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()
	restoreCVM := reinstallCommonStubCVMClient(t)
	defer restoreCVM()

	var notified atomic.Int32
	origNotif := createErrorNotification
	createErrorNotification = func(_, _ uint, _, notifyType, _ string, _ error, _ context.Context) {
		if notifyType == model.NotifyTypeInstanceReinstallFailed {
			notified.Add(1)
		}
	}
	defer func() { createErrorNotification = origNotif }()

	inst := &model.Instance{
		Name: "ri-fail-a", InstanceId: "ins-ri-fail-a",
		UserID: 99, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId: "img-ri-fail-a", ImageName: "x", ImageType: "PRIVATE_IMAGE",
		AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
	})

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := adminTokenPost(t, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallAdminOpts)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Admin ResetInstance 失败应 500，实际=%d", rr.Code)
	}
	time.Sleep(50 * time.Millisecond)
	if notified.Load() != 1 {
		t.Errorf("Admin 重装失败也应通知实例所有者一次，实际=%d", notified.Load())
	}
}

// ---------------------------------------------------------------------------
// 成功路径：通过把 CVM client 指向 httptest 服务器，让 ResetInstance 调用真返回 200
// ---------------------------------------------------------------------------

// successCVMServer 启动一个 httptest 服务器，模拟腾讯云 CVM API 始终返回成功响应。
// 返回 (cleanup, NewCVMClient 替换 fn 的恢复函数)。
func successCVMServer(t *testing.T) func() {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":{"RequestId":"test-req-id"}}`))
	}))

	orig := NewCVMClient
	NewCVMClient = func(context.Context) (*cvm.Client, error) {
		credential := tccommon.NewCredential("AKIDtest-id", "test-secret-key")
		cpf := tcprofile.NewClientProfile()
		// httptest URL 形如 http://127.0.0.1:PORT；剥掉 scheme 给 SDK Endpoint。
		cpf.HttpProfile.Endpoint = strings.TrimPrefix(ts.URL, "http://")
		cpf.HttpProfile.Scheme = "HTTP"
		client, err := cvm.NewClient(credential, "ap-guangzhou", cpf)
		if err != nil {
			return nil, hcommon.I18nRichError(err, i18n.MsgCreateCVMClientFailed)
		}
		return client, nil
	}
	return func() {
		NewCVMClient = orig
		ts.Close()
	}
}

func TestCommon_User_Success_JSON(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()
	restoreCVM := successCVMServer(t)
	defer restoreCVM()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "succ", InstanceId: "ins-succ",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		AIModelID: 3,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId: "img-succ", ImageName: "succ", ImageType: "PRIVATE_IMAGE",
		AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
	})
	model.DB(context.Background()).Create(&model.InstanceModel{
		InstanceID: inst.ID, AIModelID: 3, Role: model.ModelRolePrimary, SortOrder: 1,
	})

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := userJSONPost(t, "u1", "/openclaw/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallUserOpts)
	if rr.Code != http.StatusOK {
		t.Fatalf("用户重装成功路径应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"ok":true`) {
		t.Errorf("响应应包含 ok:true，实际=%s", rr.Body.String())
	}

	// 验证 resetReinstallBusinessState 清空了 ai_model_id 并物理删除了模型绑定
	var got model.Instance
	model.DB(context.Background()).First(&got, inst.ID)
	if got.AIModelID != 0 {
		t.Errorf("AIModelID 应被清空，实际=%d", got.AIModelID)
	}
	var imCnt int64
	model.DB(context.Background()).Unscoped().Model(&model.InstanceModel{}).
		Where("instance_id = ?", inst.ID).Count(&imCnt)
	if imCnt != 0 {
		t.Errorf("InstanceModel 应被物理删除，剩余=%d", imCnt)
	}
}

func TestCommon_Admin_Success_JSON(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()
	restoreCVM := successCVMServer(t)
	defer restoreCVM()

	inst := &model.Instance{
		Name: "succ-a", InstanceId: "ins-succ-a",
		UserID: 11, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId: "img-succ-a", ImageName: "succ-a", ImageType: "PRIVATE_IMAGE",
		AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
	})

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := adminTokenPost(t, "/admin/instances/reset", form.Encode())
	// admin 路径会调 getAdminUser → session（无 session 时返回 "unknown"）
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallAdminOpts)
	if rr.Code != http.StatusOK {
		t.Fatalf("Admin 重装成功应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// 剩余分支兜底覆盖
// ---------------------------------------------------------------------------

// TestPrepareReinstallImage_QueryError 覆盖 GetEnabledImageByType 返回 err 分支
// （即 prepareReinstallImage 内 `return nil, fmt.Errorf("查询镜像失败")`）。
//
// 做法：把 ai_images 表删掉，让 model.GetEnabledImageByType 的 SELECT 失败。
func TestPrepareReinstallImage_QueryError(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).Migrator().DropTable("ai_images"); err != nil {
		t.Fatalf("drop ai_images: %v", err)
	}

	ctx := context.Background()
	inst := &model.Instance{AgentType: model.AgentTypeOpenClaw}
	_, err := prepareReinstallImage(ctx, inst)
	if err == nil {
		t.Fatal("ai_images 表不存在时应返回 '查询镜像失败' 错误")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(ctx, err), "查询镜像失败") {
		t.Errorf("应返回 '查询镜像失败' 文案，实际=%v", err)
	}
}

// registerInstanceUpdateFailCallback 注册一个 GORM Update Before 回调，
// 当 Update 的 map 中**包含**指定字段时，注入错误并阻止真正执行。
// 返回一个清理函数，按需 deregister。
func registerInstanceUpdateFailCallback(t *testing.T, requireField string, errMsg string) func() {
	t.Helper()
	const cbName = "test:fail-on-update"
	db := model.DB(context.Background())
	err := db.Callback().Update().Before("gorm:update").Register(cbName, func(tx *gorm.DB) {
		dst, ok := tx.Statement.Dest.(map[string]interface{})
		if !ok {
			return
		}
		if _, has := dst[requireField]; has {
			_ = tx.AddError(fmt.Errorf("%s", errMsg))
		}
	})
	if err != nil {
		t.Fatalf("注册 update 回调失败: %v", err)
	}
	return func() {
		_ = db.Callback().Update().Remove(cbName)
	}
}

// registerInstanceUpdateFailWhenAIModelOnlyCallback 仅在 Update 的字段集为
// 单纯 {ai_model_id} —— 即 resetReinstallBusinessState(resetVersionInfo=false)
// 路径 —— 时报错。其他 ai_model_id 出现的场景（resetInstanceVersionInfo 同时
// 携带 cls_agent_status）不影响。
func registerInstanceUpdateFailWhenAIModelOnlyCallback(t *testing.T) func() {
	t.Helper()
	const cbName = "test:fail-on-ai-model-only-update"
	db := model.DB(context.Background())
	err := db.Callback().Update().Before("gorm:update").Register(cbName, func(tx *gorm.DB) {
		dst, ok := tx.Statement.Dest.(map[string]interface{})
		if !ok {
			return
		}
		_, hasAIModel := dst["ai_model_id"]
		_, hasCls := dst["cls_agent_status"]
		if hasAIModel && !hasCls {
			_ = tx.AddError(fmt.Errorf("mock: ai_model_id-only update fail"))
		}
	})
	if err != nil {
		t.Fatalf("注册 update 回调失败: %v", err)
	}
	return func() {
		_ = db.Callback().Update().Remove(cbName)
	}
}

// TestCommon_User_ResetVersionInfoFailed 覆盖 commonHandleResetInstance 中
// `if err := resetInstanceVersionInfo(...); err != nil` 分支（181-188 行）。
func TestCommon_User_ResetVersionInfoFailed(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()
	// 不需要替换 CVM client：流程不会走到 NewCVMClient。

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "rvi-fail", InstanceId: "ins-rvi-fail",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId: "img-rvi", ImageName: "rvi", ImageType: "PRIVATE_IMAGE",
		AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
	})

	// 注册回调让带 cls_agent_status 的 Update 失败：resetInstanceVersionInfo
	// 必带该字段；setOperationWithAgentReset 与 clearOperation 都不带，因此
	// 状态机仍能正常翻转到 failed。
	restoreCB := registerInstanceUpdateFailCallback(t,
		"cls_agent_status", "mock: reset version info update fail")
	defer restoreCB()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := userJSONPost(t, "u1", "/openclaw/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallUserOpts)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("resetInstanceVersionInfo 失败应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "重置版本信息失败") {
		t.Errorf("响应应包含 '重置版本信息失败'，实际=%s", rr.Body.String())
	}

	// clearOperation 应已生效（current_operation 已清空）
	var got model.Instance
	model.DB(context.Background()).First(&got, inst.ID)
	if got.CurrentOperation != model.OpNone {
		t.Errorf("失败后 current_operation 应被 clearOperation 清空，实际=%q", got.CurrentOperation)
	}
}

// TestCommon_User_ResetBusinessStateFailed_NonBlocking 覆盖：
//   - commonHandleResetInstance 234-237 行：resetReinstallBusinessState 失败时的
//     非阻塞 warn 分支（请求仍返回 200 成功）
//   - resetReinstallBusinessState 336-338 行：tx.Model(...).Updates(...).Error != nil 分支
func TestCommon_User_ResetBusinessStateFailed_NonBlocking(t *testing.T) {
	cleanup := initReinstallCommonTestDB(t)
	defer cleanup()
	restoreCVM := successCVMServer(t)
	defer restoreCVM()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "rbs-fail", InstanceId: "ins-rbs-fail",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw, AIModelID: 5,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId: "img-rbs", ImageName: "rbs", ImageType: "PRIVATE_IMAGE",
		AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
	})

	// 仅当 update 字段集 == {ai_model_id} 时报错（resetReinstallBusinessState(false) 唯一的 update）。
	// 不影响 resetInstanceVersionInfo（带 cls_agent_status）/ setOperation / clearOperation。
	restoreCB := registerInstanceUpdateFailWhenAIModelOnlyCallback(t)
	defer restoreCB()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := userJSONPost(t, "u1", "/openclaw/reset", form.Encode())
	rr := httptest.NewRecorder()
	commonHandleResetInstance(rr, req, runningResolver, reinstallUserOpts)
	if rr.Code != http.StatusOK {
		t.Fatalf("resetReinstallBusinessState 失败应非阻塞返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"ok":true`) {
		t.Errorf("响应应仍标识成功，实际=%s", rr.Body.String())
	}
}

// 兜底：明确依赖 errors 包以避免误导
var _ = errors.New
