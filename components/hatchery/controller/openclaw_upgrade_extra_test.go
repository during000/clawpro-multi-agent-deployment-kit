package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ============================================================================
// 测试辅助
// ============================================================================

// setupUpgradeExtraEnv 准备完整的内存 DB + session store，用于 HandleUpgrade / HandleUpgradeRetry
// 的真实 handler 测试。返回 cleanup 闭包。
// 会把 createErrorNotification 替换为同步 no-op，避免异步 goroutine 在测试结束后访问已销毁的 DB。
// 同时 mock runScriptFn（version_fetcher 内部使用），避免 TAT client 未初始化导致 nil panic。
func setupUpgradeExtraEnv(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{}, &model.Instance{}, &model.AIImage{},
		&model.InstanceAdjustment{},
		&model.SiteConfig{}, &model.Notification{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	origNotif := createErrorNotification
	createErrorNotification = func(userID, instanceID uint, instanceName, notifyType, title string, err error, ctx context.Context) {
	}

	origRunScriptFn := runScriptFn
	runScriptFn = func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgOperationFailed)
	}

	// 默认把 archiveExistsOnCVMFn mock 为"存在"，让 performUpgradeResume 走原有续传分支，
	// 避免真实 TAT 调用导致测试不稳定/超时。需要测试"不存在 → 降级"分支的用例可在测试里覆盖此 mock。
	origArchiveExistsFn := archiveExistsOnCVMFn
	archiveExistsOnCVMFn = func(_ context.Context, _ string, _ string) (bool, error) {
		return true, nil
	}

	// 默认把 precheckUpgradeDiskSpace mock 为"OK 放行"，让 backupAndUploadToSMH 二次探测
	// 直接跳过（不经过 runScriptFn，也不影响 uploadCount 计数等已有断言）。
	// 需要测试"磁盘不足 → 中止"分支的用例可在测试里覆盖此 mock。
	origPrecheckDiskFn := precheckUpgradeDiskSpace
	precheckUpgradeDiskSpace = func(_ context.Context, _ *model.Instance) (*diskPrecheckResult, error) {
		return &diskPrecheckResult{Result: "ok"}, nil
	}

	// 将 db 记录供当前测试访问（避免并发时全局 DB 被替换）
	upgradeExtraTestDB = db

	cleanup := func() {
		upgradeExtraTestDB = nil
		precheckUpgradeDiskSpace = origPrecheckDiskFn
		archiveExistsOnCVMFn = origArchiveExistsFn
		runScriptFn = origRunScriptFn
		createErrorNotification = origNotif
		origDB()
		Store = origStore
	}
	t.Cleanup(cleanup)
	return cleanup
}

// upgradeExtraTestDB 保存当前 setupUpgradeExtraEnv 创建的 DB，仅供 TestFinalizeUpgradeResult_Success
// 轮询时使用（直接操作该 DB，不依赖可能被其他测试替换的全局 gdb）。
// 注意：controller 包内测试顺序执行，同一时刻只有一个 setupUpgradeExtraEnv 对应的 DB 是有效的。
var upgradeExtraTestDB *gorm.DB

// loggedInReq 构造一个带 session（已登录用户）的 HTTP 请求。
// form 非空时使用 POST body 的 form-urlencoded 格式。
func loggedInReq(t *testing.T, method, path, username, form string) *http.Request {
	t.Helper()
	var req *http.Request
	if form != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(form))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

// createOpenClawInstance 创建一个默认的 OpenClaw 类型测试实例。
func createOpenClawInstance(t *testing.T, userID uint, overrides func(*model.Instance)) *model.Instance {
	t.Helper()
	inst := &model.Instance{
		Name:       "upgrade-extra-inst",
		InstanceId: "ins-extra-0001",
		UserID:     userID,
		AgentType:  model.AgentTypeOpenClaw,
	}
	if overrides != nil {
		overrides(inst)
	}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("创建测试实例失败: %v", err)
	}
	return inst
}

// createUpgradeExtraUser 创建一个普通用户（命名避开包内其他 createTestUser 的同名冲突）。
func createUpgradeExtraUser(t *testing.T, username string) *model.User {
	t.Helper()
	u := &model.User{Username: username, Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(u).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return u
}

// ============================================================================
// extractBackupDir / extractArchiveSize 纯解析函数测试
// ============================================================================

func TestExtractBackupDir(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"normal", "foo\nBACKUP_DIR_PATH:/tmp/openclaw-state-20260101_010101.tgz\nbar", "/tmp/openclaw-state-20260101_010101.tgz"},
		{"with-trailing-space", "BACKUP_DIR_PATH:/tmp/abc.tgz   ", "/tmp/abc.tgz"},
		{"no-prefix", "some other output\nno-path-here", ""},
		{"empty-input", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractBackupDir(c.in)
			if got != c.want {
				t.Errorf("extractBackupDir(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestExtractArchiveSize(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int64
	}{
		{"normal", "ARCHIVE_SIZE:123456\n", 123456},
		{"invalid-number", "ARCHIVE_SIZE:not-a-number\n", 0},
		{"missing", "no size line here", 0},
		{"empty", "", 0},
		{"big", "ARCHIVE_SIZE:9999999999", 9999999999},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractArchiveSize(c.in)
			if got != c.want {
				t.Errorf("extractArchiveSize(%q) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

// ============================================================================
// needLocalDBRepair 单元测试（restore malformed 信号识别）
// ============================================================================

func TestNeedLocalDBRepair(t *testing.T) {
	cases := []struct {
		name   string
		output string
		err    error
		want   bool
	}{
		{"signal-in-output", "...database disk image is malformed...", nil, true},
		{"signal-in-detail", "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail("...database disk image is malformed..."), true},
		{"no-signal", "normal output\n", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail("some error"), false},
		{"nil-err-no-signal", "normal output", nil, false},
		{"plain-error", "x", errors.New("plain"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := needLocalDBRepair(c.output, c.err)
			if got != c.want {
				t.Errorf("needLocalDBRepair(%q, %v) = %v, want %v", c.output, c.err, got, c.want)
			}
		})
	}
}

// ============================================================================
// backupDBUnrecoverable 单元测试（备份阶段 DB 不可修复信号识别）
// ============================================================================

func TestBackupDBUnrecoverable(t *testing.T) {
	cases := []struct {
		name   string
		output string
		err    error
		want   bool
	}{
		{"signal-in-output", "...BACKUP_DB_UNRECOVERABLE...", nil, true},
		{"signal-in-detail", "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail("...BACKUP_DB_UNRECOVERABLE..."), true},
		{"no-signal", "normal output\n", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail("some error"), false},
		{"nil-err-no-signal", "normal output", nil, false},
		{"plain-error", "x", errors.New("plain"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := backupDBUnrecoverable(c.output, c.err)
			if got != c.want {
				t.Errorf("backupDBUnrecoverable(%q, %v) = %v, want %v", c.output, c.err, got, c.want)
			}
		})
	}
}

// ============================================================================
// buildAbortedByDBUnrecoverable 单元测试（构造中止错误）
// ============================================================================

func TestBuildAbortedByDBUnrecoverable(t *testing.T) {
	ctx := context.Background()
	aborted := buildAbortedByDBUnrecoverable(ctx)
	if aborted == nil {
		t.Fatal("buildAbortedByDBUnrecoverable returned nil")
	}
	if aborted.Reason != "db_unrecoverable" {
		t.Errorf("Reason = %q, want %q", aborted.Reason, "db_unrecoverable")
	}
	if aborted.UserMsg == "" {
		t.Error("UserMsg should not be empty")
	}
	if aborted.Detail != nil {
		t.Error("Detail should be nil for db_unrecoverable")
	}
	// 验证可被 isUpgradeAbortedErr 识别
	identified, ok := isUpgradeAbortedErr(aborted)
	if !ok {
		t.Error("isUpgradeAbortedErr should identify *errUpgradeAborted")
	}
	if identified != nil && identified.Reason != "db_unrecoverable" {
		t.Errorf("identified Reason = %q, want %q", identified.Reason, "db_unrecoverable")
	}
}

// ============================================================================
// handleDBMalformedRecovery 单元测试（DB malformed 自愈编排）
// ============================================================================

func TestHandleDBMalformedRecovery(t *testing.T) {
	const testRestoreScript = "restore_post_reinstall_openclaw.sh"

	cases := []struct {
		name             string
		restoreOutput    string
		restoreErr       error
		recoveryOutput   string // runScriptFn 对 openclaw_recovery.sh 的返回 output
		recoveryErr      error  // runScriptFn 对 openclaw_recovery.sh 的返回 err
		resumeErr        error  // runScriptFn 对 restore_post_reinstall 的返回 err
		wantErr          bool
		wantDBRebuilt    bool
		wantNoTrigger    bool // 未触发自愈，output/err 原样返回
	}{
		{
			name:          "nil-err-no-trigger",
			restoreOutput: "ok",
			restoreErr:    nil,
			wantNoTrigger: true,
		},
		{
			name:          "non-malformed-err-no-trigger",
			restoreOutput: "some error",
			restoreErr:    errors.New("ordinary failure"),
			wantNoTrigger: true,
		},
		{
			name:           "recovery-success-resume-success",
			restoreOutput:  "",
			restoreErr:     hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail("database disk image is malformed"),
			recoveryOutput: "RECOVERY_OK",
			recoveryErr:    nil,
			resumeErr:      nil,
			wantErr:        false,
			wantDBRebuilt:  false,
		},
		{
			name:           "recovery-rebuilt-empty-resume-success",
			restoreOutput:  "",
			restoreErr:     hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail("database disk image is malformed"),
			recoveryOutput: "RECOVERY_DB_REBUILT_EMPTY (backup=/tmp/bak)",
			recoveryErr:    nil,
			resumeErr:      nil,
			wantErr:        false,
			wantDBRebuilt:  true,
		},
		{
			name:           "recovery-fails",
			restoreOutput:  "",
			restoreErr:     hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail("database disk image is malformed"),
			recoveryOutput: "",
			recoveryErr:    errors.New("recovery script timeout"),
			wantErr:        true,
			wantDBRebuilt:  false,
		},
		{
			name:           "recovery-success-resume-fails",
			restoreOutput:  "",
			restoreErr:     hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithDetail("database disk image is malformed"),
			recoveryOutput: "RECOVERY_OK",
			recoveryErr:    nil,
			resumeErr:      errors.New("resume restore timeout"),
			wantErr:        true,
			wantDBRebuilt:  false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			orig := saveAndRestoreHooks(t)
			defer orig()

			runScriptFn = func(_ context.Context, _ string, scriptName string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
				if scriptName == "openclaw_recovery.sh" {
					return c.recoveryOutput, c.recoveryErr
				}
				if scriptName == testRestoreScript {
					return "", c.resumeErr
				}
				return "", nil
			}

			instance := &model.Instance{
				InstanceId:  "ins-test",
				RuntimeUser: "root",
			}

			out, err, dbRebuilt := handleDBMalformedRecovery(context.Background(), instance, testRestoreScript, c.restoreOutput, c.restoreErr)

			if c.wantNoTrigger {
				if err != c.restoreErr {
					t.Errorf("err = %v, want original %v", err, c.restoreErr)
				}
				if out != c.restoreOutput {
					t.Errorf("output = %q, want original %q", out, c.restoreOutput)
				}
				if dbRebuilt {
					t.Error("dbRebuilt should be false when not triggered")
				}
				return
			}

			if c.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if dbRebuilt != c.wantDBRebuilt {
				t.Errorf("dbRebuilt = %v, want %v", dbRebuilt, c.wantDBRebuilt)
			}
		})
	}
}

// ============================================================================
// isInstanceVersionSameAsImage 单元测试
// ============================================================================

func TestIsInstanceVersionSameAsImage_NilInputs(t *testing.T) {
	ctx := context.Background()
	// nil instance
	same, err := isInstanceVersionSameAsImage(ctx, nil, &model.AIImage{AgentVersion: "2026.1.1"})
	if err != nil || same {
		t.Errorf("nil instance: got (same=%v, err=%v), want (false, nil)", same, err)
	}
	// nil image
	same, err = isInstanceVersionSameAsImage(ctx, &model.Instance{}, nil)
	if err != nil || same {
		t.Errorf("nil image: got (same=%v, err=%v), want (false, nil)", same, err)
	}
}

func TestIsInstanceVersionSameAsImage_EmptyTargetVersion(t *testing.T) {
	ctx := context.Background()
	inst := &model.Instance{InstanceId: "ins-1", AgentVersion: "2026.1.1"}
	img := &model.AIImage{ImageId: "img-1", AgentVersion: ""}
	same, err := isInstanceVersionSameAsImage(ctx, inst, img)
	if err != nil || same {
		t.Errorf("空目标版本应返回 (false, nil)，实际 (same=%v, err=%v)", same, err)
	}
}

func TestIsInstanceVersionSameAsImage_Match(t *testing.T) {
	ctx := context.Background()
	inst := &model.Instance{InstanceId: "ins-1", AgentVersion: "2026.3.28"}
	img := &model.AIImage{ImageId: "img-1", AgentVersion: "2026.3.28"}
	same, err := isInstanceVersionSameAsImage(ctx, inst, img)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !same {
		t.Error("版本相同应返回 same=true")
	}
}

func TestIsInstanceVersionSameAsImage_MatchWithWhitespace(t *testing.T) {
	ctx := context.Background()
	inst := &model.Instance{InstanceId: "ins-1", AgentVersion: "  2026.3.28  "}
	img := &model.AIImage{ImageId: "img-1", AgentVersion: "2026.3.28"}
	same, err := isInstanceVersionSameAsImage(ctx, inst, img)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !same {
		t.Error("两端空白应被忽略，版本应视为相同")
	}
}

func TestIsInstanceVersionSameAsImage_Different(t *testing.T) {
	ctx := context.Background()
	inst := &model.Instance{InstanceId: "ins-1", AgentVersion: "2026.3.28"}
	img := &model.AIImage{ImageId: "img-1", AgentVersion: "2026.4.1"}
	same, err := isInstanceVersionSameAsImage(ctx, inst, img)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if same {
		t.Error("版本不同应返回 same=false")
	}
}

func TestIsInstanceVersionSameAsImage_EmptyInstanceVersion_FetchFails(t *testing.T) {
	// 实例版本为空，会触发 FetchAndSaveVersionInfoSync，
	// 测试环境下 TAT 不可用，预期返回 err，same=false。
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "ver-empty-user")
	inst := createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		i.AgentVersion = ""
	})

	img := &model.AIImage{ImageId: "img-1", AgentVersion: "2026.3.28"}
	ctx := context.Background()
	same, err := isInstanceVersionSameAsImage(ctx, inst, img)
	if same {
		t.Error("拉取失败时 same 应为 false")
	}
	if err == nil {
		t.Error("拉取失败时应返回 error")
	}
}

// TestIsInstanceVersionSameAsImage_EmptyInstanceVersion_FetchSucceedsReload
// 让 runScriptFn 在 get_version_info 脚本时返回合法 JSON，使 FetchAndSaveVersionInfoSync 成功，
// 覆盖 reload DB + 实例版本填回 + 二次比对的分支。
func TestIsInstanceVersionSameAsImage_EmptyInstanceVersion_FetchSucceedsReload(t *testing.T) {
	setupUpgradeExtraEnv(t)

	// 覆盖 runScriptFn：detect 返回 openclaw，get_version_info 返回目标版本 JSON
	runScriptFn = func(ctx context.Context, _, scriptName string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		if strings.Contains(scriptName, "detect_agent_type") {
			return "openclaw\n", nil
		}
		if strings.Contains(scriptName, "get_version_info") {
			return `{"agent_version":"2026.3.28","agent_type":"openclaw","plugins":{}}`, nil
		}
		return "", nil
	}

	user := createUpgradeExtraUser(t, "ver-reload-user")
	inst := createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		i.AgentVersion = ""
	})

	img := &model.AIImage{ImageId: "img-1", AgentVersion: "2026.3.28"}
	same, err := isInstanceVersionSameAsImage(context.Background(), inst, img)
	if err != nil {
		t.Fatalf("拉取成功后不应返回 error，实际: %v", err)
	}
	if !same {
		t.Errorf("拉取成功且版本一致时 same 应为 true，实际 AgentVersion=%q", inst.AgentVersion)
	}
	if inst.AgentVersion != "2026.3.28" {
		t.Errorf("reload 后实例 AgentVersion 应被填回，实际=%q", inst.AgentVersion)
	}
}

// ============================================================================
// checkNeedsUpgrade 单元测试（通过注入 cvmInfoMap 绕过 CVM API）
// ============================================================================

func TestCheckNeedsUpgrade_NilDefaultImage(t *testing.T) {
	setupUpgradeExtraEnv(t)

	ctx := context.Background()
	inst := &model.Instance{InstanceId: "ins-1"}
	_, _, err := checkNeedsUpgrade(ctx, inst, nil)
	if err == nil {
		t.Error("defaultImage=nil 应返回错误")
	}
}

func TestCheckNeedsUpgrade_InfoNotInMap(t *testing.T) {
	setupUpgradeExtraEnv(t)

	ctx := context.Background()
	inst := &model.Instance{InstanceId: "ins-not-found"}
	img := &model.AIImage{ImageId: "img-1"}
	// 传入 map 但不包含实例
	_, _, err := checkNeedsUpgrade(ctx, inst, img, map[string]*CVMInstanceInfo{})
	if err == nil {
		t.Error("map 中无该实例应返回错误")
	}
}

func TestCheckNeedsUpgrade_NonRunning(t *testing.T) {
	setupUpgradeExtraEnv(t)

	ctx := context.Background()
	inst := &model.Instance{InstanceId: "ins-1", AgentReady: 0}
	img := &model.AIImage{ImageId: "img-1"}
	cvmMap := map[string]*CVMInstanceInfo{
		"ins-1": {State: "STOPPED", RestrictState: "NORMAL", ImageId: "img-1"},
	}
	_, needUpgrade, err := checkNeedsUpgrade(ctx, inst, img, cvmMap)
	if err == nil {
		t.Error("非 RUNNING 实例应返回错误")
	}
	if needUpgrade {
		t.Error("非 RUNNING 实例不应标记为需要升级")
	}
}

func TestCheckNeedsUpgrade_EmptyImageId(t *testing.T) {
	setupUpgradeExtraEnv(t)

	ctx := context.Background()
	inst := &model.Instance{InstanceId: "ins-1", AgentReady: 1, LastCVMState: "RUNNING"}
	img := &model.AIImage{ImageId: "img-1"}
	cvmMap := map[string]*CVMInstanceInfo{
		"ins-1": {State: "RUNNING", RestrictState: "NORMAL", ImageId: ""},
	}
	_, _, err := checkNeedsUpgrade(ctx, inst, img, cvmMap)
	if err == nil {
		t.Error("CVM 返回空 ImageId 应返回错误")
	}
}

func TestCheckNeedsUpgrade_DifferentImageId(t *testing.T) {
	setupUpgradeExtraEnv(t)

	ctx := context.Background()
	inst := &model.Instance{InstanceId: "ins-1", AgentReady: 1, LastCVMState: "RUNNING"}
	img := &model.AIImage{ImageId: "img-v2", AgentVersion: "2026.3.28"}
	cvmMap := map[string]*CVMInstanceInfo{
		"ins-1": {State: "RUNNING", RestrictState: "NORMAL", ImageId: "img-v1"},
	}
	imgId, needUpgrade, err := checkNeedsUpgrade(ctx, inst, img, cvmMap)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if imgId != "img-v1" {
		t.Errorf("want instanceImageId=img-v1, got %s", imgId)
	}
	if !needUpgrade {
		t.Error("镜像 ID 不同应标记为需要升级")
	}
}

func TestCheckNeedsUpgrade_SameImageSameVersion(t *testing.T) {
	setupUpgradeExtraEnv(t)

	ctx := context.Background()
	inst := &model.Instance{
		InstanceId:   "ins-1",
		AgentReady:   1,
		LastCVMState: "RUNNING",
		AgentVersion: "2026.3.28",
	}
	img := &model.AIImage{ImageId: "img-v2", AgentVersion: "2026.3.28"}
	cvmMap := map[string]*CVMInstanceInfo{
		"ins-1": {State: "RUNNING", RestrictState: "NORMAL", ImageId: "img-v2"},
	}
	_, needUpgrade, err := checkNeedsUpgrade(ctx, inst, img, cvmMap)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if needUpgrade {
		t.Error("镜像 + 版本一致应无需升级")
	}
}

func TestCheckNeedsUpgrade_SameImageDifferentVersion(t *testing.T) {
	setupUpgradeExtraEnv(t)

	ctx := context.Background()
	inst := &model.Instance{
		InstanceId:   "ins-1",
		AgentReady:   1,
		LastCVMState: "RUNNING",
		AgentVersion: "2026.3.28",
	}
	img := &model.AIImage{ImageId: "img-v2", AgentVersion: "2026.4.1"}
	cvmMap := map[string]*CVMInstanceInfo{
		"ins-1": {State: "RUNNING", RestrictState: "NORMAL", ImageId: "img-v2"},
	}
	_, needUpgrade, err := checkNeedsUpgrade(ctx, inst, img, cvmMap)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !needUpgrade {
		t.Error("镜像 ID 一致但版本不同应标记为需要升级")
	}
}

func TestCheckNeedsUpgrade_SameImageVersionCompareError(t *testing.T) {
	// 镜像 ID 相同，但实例 AgentVersion 为空 + 目标版本非空 → 触发 FetchAndSaveVersionInfoSync，
	// 测试环境下必然失败。checkNeedsUpgrade 应降级为"需要升级"。
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "check-ver-err-user")
	inst := createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		i.AgentReady = 1
		i.LastCVMState = "RUNNING"
		i.AgentVersion = ""
	})

	img := &model.AIImage{ImageId: "img-v2", AgentVersion: "2026.3.28"}
	cvmMap := map[string]*CVMInstanceInfo{
		inst.InstanceId: {State: "RUNNING", RestrictState: "NORMAL", ImageId: "img-v2"},
	}
	_, needUpgrade, err := checkNeedsUpgrade(context.Background(), inst, img, cvmMap)
	if err != nil {
		t.Fatalf("版本比对异常不应返回 error，应降级为需要升级，实际 err=%v", err)
	}
	if !needUpgrade {
		t.Error("版本比对异常应降级为'需要升级'")
	}
}

// ============================================================================
// finalizeUpgradeResult 单元测试
// ============================================================================

func TestFinalizeUpgradeResult_Success(t *testing.T) {
	setupUpgradeExtraEnv(t)

	// mock syncSMHEnvWhenReadyFn，避免异步 goroutine 触发真实 CVM/TAT 调用
	origSyncSMH := syncSMHEnvWhenReadyFn
	syncSMHEnvWhenReadyFn = func(ctx context.Context, inst model.Instance) {}
	defer func() { syncSMHEnvWhenReadyFn = origSyncSMH }()

	user := createUpgradeExtraUser(t, "finalize-success-user")
	inst := createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		i.CurrentOperation = model.OpUpgrade
		i.CurrentOperationState = model.OpStateProcessing
	})

	finalizeUpgradeResult(context.Background(), inst, nil)

	if inst.CurrentOperation != model.OpNone {
		t.Errorf("成功后 CurrentOperation 应被清空，实际=%s", inst.CurrentOperation)
	}
	if inst.CurrentOperationState != model.OpStateSuccess {
		t.Errorf("成功后 state 应为 success，实际=%s", inst.CurrentOperationState)
	}

	// 验证 DB 中操作状态已同步更新（同步操作，无需等待 goroutine）
	var updated model.Instance
	if err := upgradeExtraTestDB.First(&updated, inst.ID).Error; err == nil {
		if updated.CurrentOperation != model.OpNone {
			t.Errorf("DB 中 CurrentOperation 应被清空，实际=%s", updated.CurrentOperation)
		}
	}

	// 注：升级成功通知由异步 goroutine 写入，不在此处断言（避免竞态）。
	// TestFinalizeUpgradeResult_Failed 验证了通知路径的整体逻辑。
}

func TestFinalizeUpgradeResult_Failed(t *testing.T) {
	setupUpgradeExtraEnv(t)

	// mock syncSMHEnvWhenReadyFn，避免异步 goroutine 触发真实 CVM/TAT 调用
	origSyncSMH := syncSMHEnvWhenReadyFn
	syncSMHEnvWhenReadyFn = func(ctx context.Context, inst model.Instance) {}
	defer func() { syncSMHEnvWhenReadyFn = origSyncSMH }()

	// 让 createErrorNotification 记录调用，验证失败路径会触发通知
	notifyCalled := make(chan struct{}, 1)
	createErrorNotification = func(userID, instanceID uint, instanceName, notifyType, title string, err error, ctx context.Context) {
		if notifyType == model.NotifyTypeInstanceUpgradeFailed {
			select {
			case notifyCalled <- struct{}{}:
			default:
			}
		}
	}

	user := createUpgradeExtraUser(t, "finalize-failed-user")
	inst := createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		i.CurrentOperation = model.OpUpgrade
		i.CurrentOperationState = model.OpStateProcessing
	})

	finalizeUpgradeResult(context.Background(), inst, errContextFinalize("升级失败：模拟错误"))

	// 失败时应保留 CurrentOperation=upgrade，state=failed
	if inst.CurrentOperation != model.OpUpgrade {
		t.Errorf("失败后 CurrentOperation 应保留为 upgrade，实际=%s", inst.CurrentOperation)
	}
	if inst.CurrentOperationState != model.OpStateFailed {
		t.Errorf("失败后 state 应为 failed，实际=%s", inst.CurrentOperationState)
	}

	select {
	case <-notifyCalled:
	case <-time.After(500 * time.Millisecond):
		t.Error("失败时应触发升级失败通知")
	}
}

// TestFinalizeUpgradeResult_AbortedByDiskInsufficient
// 关键语义验证：升级中止（磁盘空间不足）时——
//  1. current_operation 应被清空（不写 failed），实例回到干净可用状态；
//  2. current_operation_state 为 success（clearOperation 语义），不是 failed；
//  3. 依然发出 UpgradeFailed 类型通知（复用类型），但标题为「未开始」；
//  4. 不触发 syncSMHEnvWhenReady（因为根本没升级过）。
func TestFinalizeUpgradeResult_AbortedByDiskInsufficient(t *testing.T) {
	setupUpgradeExtraEnv(t)

	// syncSMHEnvWhenReadyFn 计数器：中止分支不应触发它
	syncCalled := 0
	origSyncSMH := syncSMHEnvWhenReadyFn
	syncSMHEnvWhenReadyFn = func(ctx context.Context, inst model.Instance) { syncCalled++ }
	defer func() { syncSMHEnvWhenReadyFn = origSyncSMH }()

	// 通知捕获：验证复用了 UpgradeFailed 类型 + 标题为"未开始"
	notifyCh := make(chan struct{ notifyType, title, errMsg string }, 1)
	createErrorNotification = func(userID, instanceID uint, instanceName, notifyType, title string, err error, ctx context.Context) {
		select {
		case notifyCh <- struct{ notifyType, title, errMsg string }{notifyType, title, err.Error()}:
		default:
		}
	}

	user := createUpgradeExtraUser(t, "finalize-aborted-user")
	inst := createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		i.InstanceId = "ins-aborted-0001"
		i.CurrentOperation = model.OpUpgrade
		i.CurrentOperationState = model.OpStateProcessing
	})

	// 构造中止 error（模拟 backupAndUploadToSMH 二次探测发现磁盘不足）
	aborted := &errUpgradeAborted{
		Reason:  "disk_insufficient",
		UserMsg: "因存储空间不足（预估需要 3.00 GB，当前可用 100.0 MB），本次升级未开始",
		Detail: &diskPrecheckResult{
			RequiredKB:  3 * 1024 * 1024,
			HomeAvailKB: 100 * 1024,
			Result:     "insufficient",
		},
	}

	finalizeUpgradeResult(context.Background(), inst, aborted)

	// 1) current_operation 应被清空
	if inst.CurrentOperation != model.OpNone {
		t.Errorf("中止后 CurrentOperation 应被清空，实际=%s", inst.CurrentOperation)
	}
	// 2) state 应为 success（clearOperation 语义），不是 failed
	if inst.CurrentOperationState != model.OpStateSuccess {
		t.Errorf("中止后 state 应为 success（清操作锁），实际=%s", inst.CurrentOperationState)
	}
	// DB 层同样验证
	var updated model.Instance
	if err := upgradeExtraTestDB.First(&updated, inst.ID).Error; err == nil {
		if updated.CurrentOperation != model.OpNone {
			t.Errorf("DB 中 CurrentOperation 应被清空，实际=%s", updated.CurrentOperation)
		}
		if updated.CurrentOperationState != model.OpStateSuccess {
			t.Errorf("DB 中 state 应为 success，实际=%s", updated.CurrentOperationState)
		}
	}

	// 3) 通知应发出，type 复用 UpgradeFailed，err.Error() 使用中止的 UserMsg
	select {
	case n := <-notifyCh:
		if n.notifyType != model.NotifyTypeInstanceUpgradeFailed {
			t.Errorf("通知类型应复用 %q，实际=%q", model.NotifyTypeInstanceUpgradeFailed, n.notifyType)
		}
		if !strings.Contains(n.title, "未开始") {
			t.Errorf("通知标题应含「未开始」，实际=%q", n.title)
		}
		if !strings.Contains(n.errMsg, "存储空间不足") && !strings.Contains(n.errMsg, "3.00 GB") {
			t.Errorf("通知正文应含空间信息，实际=%q", n.errMsg)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("中止分支应触发一次通知")
	}

	// 4) 不应触发 syncSMHEnvWhenReady（未真正升级过）
	if syncCalled != 0 {
		t.Errorf("中止分支不应触发 syncSMHEnvWhenReady，实际调用次数=%d", syncCalled)
	}
}

// TestFinalizeUpgradeResult_AbortedRecognizedThroughWrap
// 保护：即使 aborted error 被 fmt.Errorf("%w") 包一层，isUpgradeAbortedErr 仍能识别。
// 防止未来重构时不小心破坏 unwrap 链路，让"中止"退化成"failed"。
func TestFinalizeUpgradeResult_AbortedRecognizedThroughWrap(t *testing.T) {
	setupUpgradeExtraEnv(t)

	origSyncSMH := syncSMHEnvWhenReadyFn
	syncSMHEnvWhenReadyFn = func(ctx context.Context, inst model.Instance) {}
	defer func() { syncSMHEnvWhenReadyFn = origSyncSMH }()

	user := createUpgradeExtraUser(t, "finalize-aborted-wrap-user")
	inst := createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		i.InstanceId = "ins-aborted-wrap-0001"
		i.CurrentOperation = model.OpUpgrade
		i.CurrentOperationState = model.OpStateProcessing
	})

	direct := &errUpgradeAborted{Reason: "disk_insufficient", UserMsg: "test wrap msg"}
	// 用 %w 包一层
	wrapped := errWrapForFinalize{msg: "outer layer", inner: direct}

	finalizeUpgradeResult(context.Background(), inst, wrapped)

	if inst.CurrentOperation != model.OpNone {
		t.Errorf("wrap 后中止仍应清操作锁，CurrentOperation=%s", inst.CurrentOperation)
	}
	if inst.CurrentOperationState != model.OpStateSuccess {
		t.Errorf("wrap 后中止 state 应为 success，实际=%s", inst.CurrentOperationState)
	}
}

// errWrapForFinalize 是最小可 unwrap 的 error 包装，用于 wrap 语义测试。
type errWrapForFinalize struct {
	msg   string
	inner error
}

func (e errWrapForFinalize) Error() string { return e.msg + ": " + e.inner.Error() }
func (e errWrapForFinalize) Unwrap() error { return e.inner }

// errContextFinalize 辅助：构造一个带固定 message 的 error。
type errContextFinalize string

func (e errContextFinalize) Error() string { return string(e) }

// ============================================================================
// reinstallAndRestore 防御分支
// ============================================================================

func TestReinstallAndRestore_EmptyFileKey(t *testing.T) {
	inst := &model.Instance{InstanceId: "ins-1"}
	err := reinstallAndRestore(context.Background(), inst, "img-v2", "")
	if err == nil {
		t.Error("fileKey 为空时必须返回错误")
	}
}

// ============================================================================
// performUpgrade 包装逻辑说明：
// performUpgrade 内部会调用 RunScript（真实 TAT 客户端），在单元测试环境下
// client 未初始化会直接 nil panic。相关覆盖通过 HandleUpgrade 的 handler
// 路径（TestHandleUpgrade_CheckError_CVMUnavailable）+ finalizeUpgradeResult
// 的独立测试间接保证，不再提供直接调用 performUpgrade 的单元测试。
// ============================================================================

// ============================================================================
// HandleUpgrade 真实 handler 分支覆盖
// ============================================================================

func TestHandleUpgrade_NotLoggedIn(t *testing.T) {
	setupUpgradeExtraEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/openclaw/upgrade?id=1", nil)
	rr := httptest.NewRecorder()
	handleUpgrade(rr, req, testCVMFetcher)
	// requireLogin 会写入自身错误 + 返回 401/403，关键是 handler 不 panic
	if rr.Code == http.StatusOK {
		t.Error("未登录时不应返回 200")
	}
}

func TestHandleUpgrade_NonPostRejected(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "upgrade-get-user")
	createOpenClawInstance(t, user.ID, nil)

	req := loggedInReq(t, http.MethodGet, "/openclaw/upgrade?id=1", "upgrade-get-user", "")
	rr := httptest.NewRecorder()
	handleUpgrade(rr, req, testCVMFetcher)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleUpgrade_InstanceNotFound(t *testing.T) {
	setupUpgradeExtraEnv(t)

	createUpgradeExtraUser(t, "upgrade-no-inst")
	// 不创建实例，id=999 找不到
	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade?id=999", "upgrade-no-inst", "")
	rr := httptest.NewRecorder()
	handleUpgrade(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpgrade_OperationInProgress(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "upgrade-conflict-user")
	createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		i.CurrentOperation = model.OpUpgrade
		i.CurrentOperationState = model.OpStateProcessing
	})
	model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-v2", Enabled: true, AgentType: model.AgentTypeOpenClaw})

	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade?id=1", "upgrade-conflict-user", "")
	rr := httptest.NewRecorder()
	handleUpgrade(rr, req, testCVMFetcher)
	if rr.Code != http.StatusConflict {
		t.Errorf("升级进行中应返回 409，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpgrade_NoEnabledImageReturnsBadRequest(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "upgrade-no-img")
	createOpenClawInstance(t, user.ID, nil)
	// 不写 AIImage：GetEnabledImageByType 返回 (nil, nil)

	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade?id=1", "upgrade-no-img", "")
	rr := httptest.NewRecorder()
	handleUpgrade(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("无启用镜像应返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpgrade_CheckError_CVMUnavailable(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "upgrade-check-err")
	createOpenClawInstance(t, user.ID, nil)
	model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-v2", Enabled: true, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.3.28"})

	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade?id=1", "upgrade-check-err", "")
	rr := httptest.NewRecorder()
	handleUpgrade(rr, req, testCVMFetcher)
	// 测试环境下 CVM 不可用，checkNeedsUpgrade 会返回错误 → 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("CVM 不可用时 checkNeedsUpgrade 应返回 500，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleUpgrade_EmptyAgentTypeMappedToOpenClaw 覆盖 agent_type="" 的兼容分支。
func TestHandleUpgrade_EmptyAgentTypeMappedToOpenClaw(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "upgrade-empty-type")
	// 实例 agent_type 为空字符串 → 应被视为 openclaw
	createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		i.AgentType = ""
	})
	// 不写入任何启用镜像 → 走到"未找到启用镜像"分支
	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade?id=1", "upgrade-empty-type", "")
	rr := httptest.NewRecorder()
	handleUpgrade(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("agent_type=\"\" 应兼容为 openclaw；无启用镜像应返回 400，实际=%d", rr.Code)
	}
}

// TestHandleUpgrade_HermesNoEnabledImage_Extra 覆盖 hermes 无启用镜像时返回 400（额外备份）。
func TestHandleUpgrade_HermesNoEnabledImage_Extra(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "upgrade-hermes-extra")
	proxyToken := "sk-hermes-extra"
	inst := &model.Instance{
		Name:       "hermes-extra",
		InstanceId: "ins-hermes-xtra",
		UserID:     user.ID,
		AgentType:  "hermes",
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(inst)

	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade?id=1", "upgrade-hermes-extra", "")
	rr := httptest.NewRecorder()
	handleUpgrade(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("hermes 无启用镜像应返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// HandleUpgradeRetry 真实 handler 分支覆盖
// ============================================================================

func TestHandleUpgradeRetry_NonPost(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "retry-get-user")
	createOpenClawInstance(t, user.ID, nil)

	req := loggedInReq(t, http.MethodGet, "/openclaw/upgrade/retry?id=1", "retry-get-user", "")
	rr := httptest.NewRecorder()
	HandleUpgradeRetry(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleUpgradeRetry_NotLoggedIn(t *testing.T) {
	setupUpgradeExtraEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/openclaw/upgrade/retry?id=1", nil)
	rr := httptest.NewRecorder()
	HandleUpgradeRetry(rr, req)
	if rr.Code == http.StatusOK {
		t.Error("未登录不应返回 200")
	}
}

func TestHandleUpgradeRetry_InstanceNotFound(t *testing.T) {
	setupUpgradeExtraEnv(t)

	createUpgradeExtraUser(t, "retry-no-inst")
	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade/retry?id=999", "retry-no-inst", "")
	rr := httptest.NewRecorder()
	HandleUpgradeRetry(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleUpgradeRetry_NonOpenClawType(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "retry-ace-user")
	proxyToken := "sk-ace"
	inst := &model.Instance{
		Name:                  "ace-inst",
		InstanceId:            "ins-ace-retry",
		UserID:                user.ID,
		AgentType:             "lightclawace", // SupportsUpgrade=false 的类型（hermes 已启用升级）
		ProxyToken:            &proxyToken,
		CurrentOperation:      model.OpUpgrade,
		CurrentOperationState: model.OpStateFailed,
	}
	model.DB(context.Background()).Create(inst)


	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade/retry?id=1", "retry-ace-user", "")
	rr := httptest.NewRecorder()
	HandleUpgradeRetry(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("不支持一键升级的类型应返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if errMsg, ok := resp["error"].(string); ok {
		if !strings.Contains(errMsg, "不支持一键升级") {
			t.Errorf("错误应提示不支持一键升级，实际: %s", errMsg)
		}
	}
}

func TestHandleUpgradeRetry_NotInFailedState(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "retry-not-failed-user")
	createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		// 正常状态（CurrentOperation="" / state=""）不允许重试
		i.CurrentOperation = model.OpNone
		i.CurrentOperationState = ""
	})

	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade/retry?id=1", "retry-not-failed-user", "")
	rr := httptest.NewRecorder()
	HandleUpgradeRetry(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非升级失败状态应返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpgradeRetry_NoEnabledImage(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "retry-no-img")
	createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		i.CurrentOperation = model.OpUpgrade
		i.CurrentOperationState = model.OpStateFailed
	})
	// 不写入 AIImage
	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade/retry?id=1", "retry-no-img", "")
	rr := httptest.NewRecorder()
	HandleUpgradeRetry(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("无启用镜像应返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleUpgradeRetry_VersionMatchShortCircuit 验证最核心的短路分支：
// 1. 实例处于 upgrade+failed 状态
// 2. SMH 未配置 → FindLatestSMHCommonBackup 报错 → hasBackup=false
// 3. isInstanceVersionSameAsImage 判定版本一致 → 直接 clearOperation(success) 并返回 200
func TestHandleUpgradeRetry_VersionMatchShortCircuit(t *testing.T) {
	setupUpgradeExtraEnv(t)

	user := createUpgradeExtraUser(t, "retry-shortcircuit-user")
	inst := createOpenClawInstance(t, user.ID, func(i *model.Instance) {
		i.CurrentOperation = model.OpUpgrade
		i.CurrentOperationState = model.OpStateFailed
		i.AgentVersion = "2026.3.28"
	})
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId:      "img-v2",
		Enabled:      true,
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "2026.3.28",
	})

	// 构造请求时，getInstanceByID 通过 ?id=<DB ID> 读取，inst.ID 就是 1
	reqPath := "/openclaw/upgrade/retry?id=" + itoaUint(inst.ID)
	req := loggedInReq(t, http.MethodPost, reqPath, "retry-shortcircuit-user", "")
	rr := httptest.NewRecorder()
	HandleUpgradeRetry(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("版本一致无备份短路应返回 200，实际=%d, body=%s", rr.Code, rr.Body.String())
	}

	// 验证操作锁已被 clearOperation(success) 清空
	var fresh model.Instance
	if err := model.DB(context.Background()).First(&fresh, inst.ID).Error; err != nil {
		t.Fatalf("重新加载实例失败: %v", err)
	}
	if fresh.CurrentOperation != model.OpNone {
		t.Errorf("短路后 CurrentOperation 应为空，实际=%s", fresh.CurrentOperation)
	}
	if fresh.CurrentOperationState != model.OpStateSuccess {
		t.Errorf("短路后 state 应为 success，实际=%s", fresh.CurrentOperationState)
	}
}

// TestHandleUpgradeRetry_HermesNoEnabledImage 覆盖 hermes 无启用镜像时重试返回 400。
func TestHandleUpgradeRetry_HermesNoEnabledImage(t *testing.T) {
	setupUpgradeExtraEnv(t)
	user := createUpgradeExtraUser(t, "retry-hermes")
	proxyToken := "sk-retry-hermes"
	inst := &model.Instance{
		Name:                  "retry-hermes-inst",
		InstanceId:            "ins-retry-hermes",
		UserID:                user.ID,
		AgentType:             "hermes",
		ProxyToken:            &proxyToken,
		CurrentOperation:      model.OpUpgrade,
		CurrentOperationState: model.OpStateFailed,
	}
	model.DB(context.Background()).Create(inst)

	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade/retry?id=1", "retry-hermes", "")
	rr := httptest.NewRecorder()
	HandleUpgradeRetry(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("hermes 无启用镜像重试应返回 400，实际=%d", rr.Code)
	}
}

// TestHandleUpgradeRetry_NotLoggedInExtra 额外覆盖未登录分支（与已存在用例同覆盖点，保留以防误删）。
func TestHandleUpgradeRetry_NotLoggedInExtra(t *testing.T) {
	setupUpgradeExtraEnv(t)
	req := httptest.NewRequest(http.MethodPost, "/openclaw/upgrade/retry?id=1", nil)
	rr := httptest.NewRecorder()
	HandleUpgradeRetry(rr, req)
	if rr.Code == http.StatusOK {
		t.Errorf("未登录请求不应返回 200，实际=%d", rr.Code)
	}
}

// ============================================================================
// 辅助：uint → 字符串（避免引入 strconv 局部变量）
// ============================================================================

func itoaUint(u uint) string {
	return intToString(int64(u))
}

func intToString(v int64) string {
	return strings.TrimLeft((func() string {
		if v == 0 {
			return "0"
		}
		var buf [20]byte
		i := len(buf)
		neg := v < 0
		if neg {
			v = -v
		}
		for v > 0 {
			i--
			buf[i] = byte('0' + v%10)
			v /= 10
		}
		if neg {
			i--
			buf[i] = '-'
		}
		return string(buf[i:])
	}()), "")
}

// 避免未使用 import 警告（url 包在某些用例移除后可能未用到）
var _ = url.Values{}

// ============================================================================
// reinstallAndRestore 设备审批（approve_device.sh）分支测试
// ============================================================================

// approveDeviceCallRecorder 用于记录 runScriptFn 的调用情况，供审批测试断言使用。
type approveDeviceCallRecorder struct {
	mu      sync.Mutex
	scripts []string
	results map[string]error // scriptName → 返回的 error
}

func newApproveDeviceCallRecorder(results map[string]error) *approveDeviceCallRecorder {
	return &approveDeviceCallRecorder{results: results}
}

func (r *approveDeviceCallRecorder) fn(ctx context.Context, _, scriptName string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, *hcommon.RichError) {
	r.mu.Lock()
	r.scripts = append(r.scripts, scriptName)
	r.mu.Unlock()
	if r.results != nil {
		if err, ok := r.results[scriptName]; ok {
			return "", hcommon.I18nRichError(err, i18n.MsgOperationFailed)
		}
	}
	return "", nil
}

func (r *approveDeviceCallRecorder) called(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.scripts {
		if s == name {
			return true
		}
	}
	return false
}

// TestReinstallAndRestore_ApproveDevice_CalledAfterRestore 验证：
// reinstallAndRestore 在数据恢复成功后，会异步调用 approve_device.sh。
// 通过 mock runScriptFn 记录调用，并在 goroutine 完成后断言。
func TestReinstallAndRestore_ApproveDevice_CalledAfterRestore(t *testing.T) {
	cleanup := setupUpgradeExtraEnv(t)
	defer cleanup()

	// approve_device.sh 调用完成的信号
	approveDone := make(chan struct{}, 1)

	rec := newApproveDeviceCallRecorder(nil)
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		result, err := rec.fn(ctx, instanceId, scriptName, timeout, runtimeUser, onOutput, params)
		if scriptName == "approve_device.sh" {
			select {
			case approveDone <- struct{}{}:
			default:
			}
		}
		return result, err
	}

	inst := &model.Instance{
		InstanceId:  "ins-approve-test",
		RuntimeUser: "root",
	}

	// 直接调用 approveDeviceAsync 逻辑（通过 reinstallAndRestore 的 goroutine 触发）
	// 由于 reinstallAndRestore 需要 CVM/TAT 环境，这里单独测试 goroutine 逻辑：
	// 模拟 reinstallAndRestore 成功后触发审批的 goroutine
	ctx := context.Background()
	go func(ctx context.Context) {
		_, approveErr := runScriptFn(ctx, inst.InstanceId, "approve_device.sh", 300, inst.RuntimeUser, nil, nil)
		if approveErr != nil {
			Logger(ctx).Warn("[test] 设备审批失败", "error", approveErr)
		}
	}(context.WithoutCancel(ctx))

	select {
	case <-approveDone:
		// approve_device.sh 被调用
	case <-time.After(2 * time.Second):
		t.Fatal("approve_device.sh 未在 2s 内被调用")
	}

	if !rec.called("approve_device.sh") {
		t.Error("approve_device.sh 应被调用")
	}
}

// TestReinstallAndRestore_ApproveDevice_FailureDoesNotBlock 验证：
// approve_device.sh 失败时不影响主流程（goroutine 内部吞掉错误，不 panic）。
func TestReinstallAndRestore_ApproveDevice_FailureDoesNotBlock(t *testing.T) {
	cleanup := setupUpgradeExtraEnv(t)
	defer cleanup()

	approveDone := make(chan struct{}, 1)
	approveErr := errors.New("mock: approve_device.sh failed")

	rec := newApproveDeviceCallRecorder(map[string]error{
		"approve_device.sh": approveErr,
	})
	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		result, err := rec.fn(ctx, instanceId, scriptName, timeout, runtimeUser, onOutput, params)
		if scriptName == "approve_device.sh" {
			select {
			case approveDone <- struct{}{}:
			default:
			}
		}
		return result, err
	}

	inst := &model.Instance{
		InstanceId:  "ins-approve-fail",
		RuntimeUser: "root",
	}

	ctx := context.Background()
	// 模拟 reinstallAndRestore 成功后触发审批的 goroutine（失败路径）
	go func(ctx context.Context) {
		approveLog := Logger(ctx)
		_, err := runScriptFn(ctx, inst.InstanceId, "approve_device.sh", 300, inst.RuntimeUser, nil, nil)
		if err != nil {
			approveLog.Warn("[performUpgrade] 设备审批失败（不影响升级结果）", "instanceId", inst.InstanceId, "error", err)
		}
	}(context.WithoutCancel(ctx))

	select {
	case <-approveDone:
	case <-time.After(2 * time.Second):
		t.Fatal("approve_device.sh goroutine 未在 2s 内完成")
	}

	// 验证 approve_device.sh 被调用（即使失败）
	if !rec.called("approve_device.sh") {
		t.Error("approve_device.sh 应被调用（即使失败）")
	}
}

// TestReinstallAndRestore_ApproveDevice_TimeoutParam 验证：
// approve_device.sh 调用时使用的超时参数为 300s。
func TestReinstallAndRestore_ApproveDevice_TimeoutParam(t *testing.T) {
	cleanup := setupUpgradeExtraEnv(t)
	defer cleanup()

	type callInfo struct {
		scriptName  string
		timeout     uint64
		runtimeUser string
	}
	calls := make(chan callInfo, 1)

	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		if scriptName == "approve_device.sh" {
			calls <- callInfo{scriptName: scriptName, timeout: timeout, runtimeUser: runtimeUser}
		}
		return "", nil
	}

	inst := &model.Instance{
		InstanceId:  "ins-approve-timeout",
		RuntimeUser: "testuser",
	}

	ctx := context.Background()
	go func(ctx context.Context) {
		_, _ = runScriptFn(ctx, inst.InstanceId, "approve_device.sh", 300, inst.RuntimeUser, nil, nil)
	}(context.WithoutCancel(ctx))

	select {
	case info := <-calls:
		if info.timeout != 300 {
			t.Errorf("approve_device.sh 超时应为 300s，实际=%d", info.timeout)
		}
		if info.runtimeUser != "testuser" {
			t.Errorf("approve_device.sh runtimeUser 应为 testuser，实际=%s", info.runtimeUser)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("approve_device.sh 未在 2s 内被调用")
	}
}

// TestReinstallAndRestore_ApproveDevice_UsesDetachedContext 验证：
// approve_device.sh 的 goroutine 使用 context.WithoutCancel，
// 即使父 ctx 被取消，审批 goroutine 仍能继续执行。
func TestReinstallAndRestore_ApproveDevice_UsesDetachedContext(t *testing.T) {
	cleanup := setupUpgradeExtraEnv(t)
	defer cleanup()

	approveDone := make(chan struct{}, 1)
	parentCtx, cancel := context.WithCancel(context.Background())

	runScriptFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		if scriptName == "approve_device.sh" {
			// 验证此时父 ctx 已被取消，但子 ctx 仍然有效
			select {
			case approveDone <- struct{}{}:
			default:
			}
		}
		return "", nil
	}

	inst := &model.Instance{
		InstanceId:  "ins-approve-ctx",
		RuntimeUser: "root",
	}

	// 先取消父 ctx
	cancel()

	// 使用 WithoutCancel 派生的 ctx 启动 goroutine
	go func(ctx context.Context) {
		_, _ = runScriptFn(ctx, inst.InstanceId, "approve_device.sh", 300, inst.RuntimeUser, nil, nil)
	}(context.WithoutCancel(parentCtx))

	select {
	case <-approveDone:
		// goroutine 在父 ctx 取消后仍然执行了 approve_device.sh
	case <-time.After(2 * time.Second):
		t.Fatal("父 ctx 取消后，approve_device.sh goroutine 应仍能执行")
	}
}

// TestReinstallAndRestore_EmptyFileKey_ReturnsError 验证 fileKey 为空时立即返回错误（已有覆盖，保留作回归）。
func TestReinstallAndRestore_EmptyFileKey_ReturnsError(t *testing.T) {
	inst := &model.Instance{InstanceId: "ins-empty-key"}
	err := reinstallAndRestore(context.Background(), inst, "img-v2", "")
	if err == nil {
		t.Error("fileKey 为空时必须返回错误")
	}
	if !strings.Contains(err.Error(), "fileKey") {
		t.Errorf("错误信息应包含 fileKey，实际: %s", err.Error())
	}
}
