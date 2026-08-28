package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ─── 测试辅助 ────────────────────────────────────────────────────────────────

func initUpgradeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}, &model.AIImage{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	return db
}

func createUpgradeTestInstance(t *testing.T, db *gorm.DB, userID uint) *model.Instance {
	t.Helper()
	ins := &model.Instance{Name: "upgrade-test", UserID: userID, InstanceId: "ins-a8av074c"}
	if err := db.Create(ins).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	return ins
}

// upgradeHandlerCore 是 HandleUpgrade 的可测试内核：
// 绕过 requireLogin / getInstanceByID，直接注入 instance，
// 并通过函数参数注入 checkNeedsUpgrade 和 performUpgrade，方便 mock。
func upgradeHandlerCore(
	w http.ResponseWriter,
	r *http.Request,
	instance *model.Instance,
	checkFn func(ctx context.Context, instance *model.Instance, defaultImage *model.AIImage, cvmInfoMap ...map[string]*CVMInstanceInfo) (string, bool, error),
	upgradeFn func(ctx context.Context, instance *model.Instance, defaultImageId, currentImageId string) error,
) {
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	defaultImage := model.GetEnabledImage(context.Background())
	if defaultImage == nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryImageFailed))
		return
	}

	instanceImageId, needUpgrade, err := checkFn(r.Context(), instance, defaultImage)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	if !needUpgrade {
		jsonOK(w, "实例已是最新版本，无需升级")
		return
	}

	if err := upgradeFn(r.Context(), instance, defaultImage.ImageId, instanceImageId); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	jsonOK(w, "升级成功")
}

// ─── 测试用例 ─────────────────────────────────────────────────────────────────

// TestHandleUpgrade_NoEnabledImage 验证：未配置启用镜像时返回 500
func TestHandleUpgrade_NoEnabledImage(t *testing.T) {
	db := initUpgradeTestDB(t)
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	db.Create(user)
	instance := createUpgradeTestInstance(t, db, user.ID)
	// 不插入任何 AIImage，GetEnabledImage 返回 nil

	req := httptest.NewRequest(http.MethodPost, "/openclaw/upgrade", nil)
	w := httptest.NewRecorder()

	upgradeHandlerCore(w, req, instance,
		func(_ context.Context, _ *model.Instance, _ *model.AIImage, _ ...map[string]*CVMInstanceInfo) (string, bool, error) {
			return "", false, nil
		},
		func(_ context.Context, _ *model.Instance, _, _ string) error { return nil },
	)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == nil {
		t.Error("期望响应包含 error 字段")
	}
}

// TestHandleUpgrade_MethodNotAllowed 验证：非 POST 请求返回 405
func TestHandleUpgrade_MethodNotAllowed(t *testing.T) {
	db := initUpgradeTestDB(t)
	user := &model.User{Username: "u2", Password: "x", Role: "user"}
	db.Create(user)
	instance := createUpgradeTestInstance(t, db, user.ID)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/upgrade", nil)
	w := httptest.NewRecorder()

	upgradeHandlerCore(w, req, instance,
		func(_ context.Context, _ *model.Instance, _ *model.AIImage, _ ...map[string]*CVMInstanceInfo) (string, bool, error) {
			return "", false, nil
		},
		func(_ context.Context, _ *model.Instance, _, _ string) error { return nil },
	)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

// TestHandleUpgrade_AlreadyLatest 验证：checkNeedsUpgrade 返回不需要升级时，返回 200 + 提示信息
func TestHandleUpgrade_AlreadyLatest(t *testing.T) {
	db := initUpgradeTestDB(t)
	user := &model.User{Username: "u3", Password: "x", Role: "user"}
	db.Create(user)
	instance := createUpgradeTestInstance(t, db, user.ID)
	db.Create(&model.AIImage{ImageId: "img-latest", Enabled: true})

	req := httptest.NewRequest(http.MethodPost, "/openclaw/upgrade", nil)
	w := httptest.NewRecorder()

	upgradeHandlerCore(w, req, instance,
		func(_ context.Context, _ *model.Instance, _ *model.AIImage, _ ...map[string]*CVMInstanceInfo) (string, bool, error) {
			return "img-latest", false, nil
		}, // 不需要升级
		func(_ context.Context, _ *model.Instance, _, _ string) error { return nil },
	)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d", w.Code)
	}
	var msg string
	json.NewDecoder(w.Body).Decode(&msg)
	if msg != "实例已是最新版本，无需升级" {
		t.Errorf("期望提示已是最新版本，实际=%v", msg)
	}
}

// TestHandleUpgrade_CheckError 验证：checkNeedsUpgrade 返回错误时，返回 500
func TestHandleUpgrade_CheckError(t *testing.T) {
	db := initUpgradeTestDB(t)
	user := &model.User{Username: "u4", Password: "x", Role: "user"}
	db.Create(user)
	instance := createUpgradeTestInstance(t, db, user.ID)
	db.Create(&model.AIImage{ImageId: "img-v2", Enabled: true})

	req := httptest.NewRequest(http.MethodPost, "/openclaw/upgrade", nil)
	w := httptest.NewRecorder()

	upgradeHandlerCore(w, req, instance,
		func(_ context.Context, _ *model.Instance, _ *model.AIImage, _ ...map[string]*CVMInstanceInfo) (string, bool, error) {
			return "", false, hcommon.I18nError(i18n.MsgQueryCVMInstanceFailed)
		},
		func(_ context.Context, _ *model.Instance, _, _ string) error { return nil },
	)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == nil {
		t.Error("期望响应包含 error 字段")
	}
}

// TestHandleUpgrade_UpgradeSuccess 验证：需要升级且 performUpgrade 成功时，返回 200 + "升级成功"
func TestHandleUpgrade_UpgradeSuccess(t *testing.T) {
	db := initUpgradeTestDB(t)
	user := &model.User{Username: "u5", Password: "x", Role: "user"}
	db.Create(user)
	instance := createUpgradeTestInstance(t, db, user.ID)
	db.Create(&model.AIImage{ImageId: "img-v2", Enabled: true})

	req := httptest.NewRequest(http.MethodPost, "/openclaw/upgrade", nil)
	w := httptest.NewRecorder()

	upgradeHandlerCore(w, req, instance,
		func(_ context.Context, _ *model.Instance, _ *model.AIImage, _ ...map[string]*CVMInstanceInfo) (string, bool, error) {
			return "img-v1", true, nil
		}, // 需要升级
		func(_ context.Context, _ *model.Instance, _, _ string) error { return nil }, // 升级成功
	)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d", w.Code)
	}
	var msg string
	json.NewDecoder(w.Body).Decode(&msg)
	if msg != "升级成功" {
		t.Errorf("期望 data=升级成功，实际=%v", msg)
	}
}

// TestHandleUpgrade_UpgradeFailed 验证：performUpgrade 返回错误时，返回 500
func TestHandleUpgrade_UpgradeFailed(t *testing.T) {
	db := initUpgradeTestDB(t)
	user := &model.User{Username: "u6", Password: "x", Role: "user"}
	db.Create(user)
	instance := createUpgradeTestInstance(t, db, user.ID)
	db.Create(&model.AIImage{ImageId: "img-v2", Enabled: true})

	req := httptest.NewRequest(http.MethodPost, "/openclaw/upgrade", nil)
	w := httptest.NewRecorder()

	upgradeHandlerCore(w, req, instance,
		func(_ context.Context, _ *model.Instance, _ *model.AIImage, _ ...map[string]*CVMInstanceInfo) (string, bool, error) {
			return "img-v1", true, nil
		},
		func(_ context.Context, _ *model.Instance, _, _ string) error {
			return errors.New("重装实例失败: API 超时")
		},
	)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == nil {
		t.Error("期望响应包含 error 字段")
	}
}

// ─── 真实调用集成测试 ──────────────────────────────────────────────────────────
//
// 运行方式（需要真实腾讯云凭证和实例）：
//
//	UPGRADE_TEST_SECRET_ID=AKIDxxx \
//	UPGRADE_TEST_SECRET_KEY=xxx \
//	UPGRADE_TEST_REGION=ap-guangzhou \
//	UPGRADE_TEST_INSTANCE_ID=ins-xxxxxxxx \
//	UPGRADE_TEST_IMAGE_ID=img-xxxxxxxx \
//	go test ./controller/ -run TestHandleUpgrade_Real -v -timeout 60s

// setupRealEnv 从环境变量读取真实调用所需配置，SecretId/SecretKey 有内置默认值，其余缺少则跳过测试。
func setupRealEnv(t *testing.T) (instanceId, imageId string) {
	t.Helper()
	getOrDefault := func(key, defaultVal string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return defaultVal
	}

	secretId := getOrDefault("UPGRADE_TEST_SECRET_ID", "")
	secretKey := getOrDefault("UPGRADE_TEST_SECRET_KEY", "")
	region := getOrDefault("UPGRADE_TEST_REGION", "ap-guangzhou")
	instanceId = getOrDefault("UPGRADE_TEST_INSTANCE_ID", "ins-a8av074c")
	imageId = getOrDefault("UPGRADE_TEST_IMAGE_ID", "img-idzg74s9")

	// 初始化 LoadScript（测试环境 main 包不参与，需手动赋值，否则调用时 nil panic）
	LoadScript = func(name string) (string, error) {
		data, err := os.ReadFile("../scripts/" + name)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	// 初始化 DB
	db := initUpgradeTestDB(t)

	// 写入 SiteConfig（凭证 + Region）
	db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{})
	db.Save(&model.SiteConfig{
		ID:           1,
		CVMSecretId:  secretId,
		CVMSecretKey: secretKey,
	})
	CVMRegion = region

	// 写入启用镜像
	db.Create(&model.AIImage{ImageId: imageId, Enabled: true})

	return instanceId, imageId
}

// TestHandleUpgrade_RealCheckNeedsUpgrade 真实调用 checkNeedsUpgrade，验证能正常查询 CVM 实例镜像信息。
func TestHandleUpgrade_RealCheckNeedsUpgrade(t *testing.T) {
	instanceId, imageId := setupRealEnv(t)

	// 没有配置真实凭据时跳过
	config := model.GetSiteConfig(context.Background())
	if config.CVMSecretId == "" || config.CVMSecretKey == "" {
		t.Skip("未配置 UPGRADE_TEST_SECRET_ID / UPGRADE_TEST_SECRET_KEY，跳过真实集成测试")
	}

	// 构造临时 Instance 用于真实调用测试
	testInst := &model.Instance{InstanceId: instanceId, AgentReady: 1}
	testImage := &model.AIImage{ImageId: imageId}
	instanceImageId, needUpgrade, err := checkNeedsUpgrade(context.Background(), testInst, testImage)
	if err != nil {
		t.Fatalf("checkNeedsUpgrade 调用失败: %v", err)
	}
	t.Logf("实例当前镜像: %s，目标镜像: %s，是否需要升级: %v", instanceImageId, imageId, needUpgrade)
}

// TestHandleUpgrade_RealHandlerFlow 真实调用完整 upgradeHandlerCore 流程（checkNeedsUpgrade + performUpgrade 均真实调用）。
// 会触发真实的备份 → 重装 → 恢复流程，请确认实例数据已备份后再运行。
func TestHandleUpgrade_RealHandlerFlow(t *testing.T) {
	instanceId, imageId := setupRealEnv(t)

	// 没有配置真实凭据时跳过
	config := model.GetSiteConfig(context.Background())
	if config.CVMSecretId == "" || config.CVMSecretKey == "" {
		t.Skip("未配置 UPGRADE_TEST_SECRET_ID / UPGRADE_TEST_SECRET_KEY，跳过真实集成测试")
	}

	db := model.DB(context.Background())
	user := &model.User{Username: "real-user", Password: "x", Role: "user"}
	db.Create(user)
	instance := &model.Instance{Name: "real-instance", UserID: user.ID, InstanceId: instanceId}
	db.Create(instance)

	req := httptest.NewRequest(http.MethodPost, "/openclaw/upgrade", nil)
	w := httptest.NewRecorder()

	// checkNeedsUpgrade 和 performUpgrade 均真实调用，触发完整重装流程
	upgradeHandlerCore(w, req, instance,
		checkNeedsUpgrade,
		performUpgrade,
	)

	t.Logf("响应状态码: %d，响应体: %s", w.Code, w.Body.String())
	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d，响应: %s", w.Code, w.Body.String())
	}

	_ = imageId
}

// ==================== agent_type 前置校验 Tests ====================

// initUpgradeTestDBWithSession 初始化内存 SQLite 数据库 + session store 用于完整 handler 测试
func initUpgradeTestDBWithSession(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}, &model.AIImage{}, &model.SiteConfig{}); err != nil {
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

func userUpgradeReqWithSession(t *testing.T, method, path string, username string, body string) *http.Request {
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

func TestHandleUpgrade_MethodNotAllowed_Full(t *testing.T) {
	cleanup := initUpgradeTestDBWithSession(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := userUpgradeReqWithSession(t, http.MethodGet, "/openclaw/upgrade?id=1", "testuser", "")
	rr := httptest.NewRecorder()

	handleUpgrade(rr, req, testCVMFetcher)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleUpgrade_HermesNoEnabledImage_Rejected(t *testing.T) {
	cleanup := initUpgradeTestDBWithSession(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	proxyToken := "sk-test-hermes"
	inst := model.Instance{
		Name:       "hermes-inst",
		InstanceId: "ins-hermes-001",
		UserID:     user.ID,
		AgentType:  "hermes",
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	form := url.Values{}
	form.Set("id", "1")
	req := userUpgradeReqWithSession(t, http.MethodPost, "/openclaw/upgrade?id=1", "testuser", form.Encode())
	rr := httptest.NewRecorder()

	handleUpgrade(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for hermes upgrade, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if errMsg, ok := resp["error"].(string); ok {
		// hermes 已支持升级（SupportsUpgrade=true），但本测试未创建 hermes 启用镜像，
		// GetEnabledImageByType 返回 nil → 400。拒绝原因是"无生效镜像"而非"类型不支持"。
		if !strings.Contains(errMsg, "OpenClaw") && !strings.Contains(strings.ToLower(errMsg), "hermes") {
			t.Errorf("expected error mentioning OpenClaw or Hermes, got %q", errMsg)
		}
	}
}

func TestHandleUpgrade_LightclawACETypeRejected_Full(t *testing.T) {
	cleanup := initUpgradeTestDBWithSession(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	proxyToken := "sk-test-lightclaw"
	inst := model.Instance{
		Name:       "lightclaw-inst",
		InstanceId: "ins-lightclaw-001",
		UserID:     user.ID,
		AgentType:  "lightclawace",
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	form := url.Values{}
	form.Set("id", "1")
	req := userUpgradeReqWithSession(t, http.MethodPost, "/openclaw/upgrade?id=1", "testuser", form.Encode())
	rr := httptest.NewRecorder()

	handleUpgrade(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for lightclawace upgrade, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

// TestHandleUpgrade_LightclawACEWithImage_RejectedByPrereq 确认：
// 当后台为 lightclawace 类型确实启用了镜像时，handleUpgrade 仍会被
// prepareInstanceForUpgrade 第 3 步（SupportsUpgrade=false）拒绝。
//
// 这是单实例入口对不支持一键升级类型的限制的真覆盖（不再被前置的"无镜像"分支短路），
// 同时也确保 handleUpgrade 中
//
//	if outcome := prepareInstanceForUpgrade(...); !outcome.OK { writeError(...) ... }
//
// 这一段改动行被覆盖。
//
// 历史：原用 hermes 验证，随 Hermes 升级能力放开，改为用 lightclawace。
func TestHandleUpgrade_LightclawACEWithImage_RejectedByPrereq(t *testing.T) {
	cleanup := initUpgradeTestDBWithSession(t)
	defer cleanup()

	user := &model.User{Username: "testuser-h", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 关键：为 lightclawace 类型启用一条镜像，让 GetEnabledImageByType 不再返回 nil
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId:   "img-ace-fake",
		Enabled:   true,
		AgentType: "lightclawace",
	})

	proxyToken := "sk-test-ace-with-image"
	inst := model.Instance{
		Name:       "ace-with-image",
		InstanceId: "ins-ace-img-001",
		UserID:     user.ID,
		AgentType:  "lightclawace",
		AgentReady: 1,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	form := url.Values{}
	form.Set("id", "1")
	req := userUpgradeReqWithSession(t, http.MethodPost, "/openclaw/upgrade?id=1", "testuser-h", form.Encode())
	rr := httptest.NewRecorder()

	handleUpgrade(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 from prepareInstanceForUpgrade, got %d, body: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	errMsg, _ := resp["error"].(string)
	// 必须命中 prereq 的"暂不支持一键升级"分支（错误信息源自 checkInstanceSupportsUpgrade）
	if !strings.Contains(errMsg, "不支持一键升级") {
		t.Errorf("expected error to mention '不支持一键升级' (from prereq SupportsUpgrade check), got %q", errMsg)
	}
}

// TestHandleUpgrade_NeedUpgradeCheckFails_StartOutcomeErr 覆盖 handleUpgrade 中
// startUpgradeForInstance 返回 Err 分支的改动行：
//
//	switch outcome := startUpgradeForInstance(...); {
//	case outcome.Err != nil:
//	    writeError(w, r, outcome.HTTPCode, outcome.Err)
//	    return
//	}
//
// 由于单测没有真实 CVM 凭证，checkNeedsUpgrade 内部会因
// "CVM 实例镜像 ID 为空 / 无法获取信息" 而返回 error，触发本分支。
func TestHandleUpgrade_NeedUpgradeCheckFails_StartOutcomeErr(t *testing.T) {
	cleanup := initUpgradeTestDBWithSession(t)
	defer cleanup()

	user := &model.User{Username: "testuser-cv", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	// OpenClaw 实例 + 启用镜像 → 通过 prereq → 进入 startUpgradeForInstance
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId:   "img-openclaw-fake",
		Enabled:   true,
		AgentType: "openclaw",
	})

	proxyToken := "sk-test-cv-fail"
	inst := model.Instance{
		Name:       "openclaw-cv-fail",
		InstanceId: "ins-openclaw-cv-001",
		UserID:     user.ID,
		AgentType:  "openclaw",
		AgentReady: 1,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	form := url.Values{}
	form.Set("id", "1")
	req := userUpgradeReqWithSession(t, http.MethodPost, "/openclaw/upgrade?id=1", "testuser-cv", form.Encode())
	rr := httptest.NewRecorder()

	handleUpgrade(rr, req, testCVMFetcher)

	// 期望 500（startUpgradeForInstance.Err 分支映射到 HTTP 500）
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from startUpgradeForInstance Err branch, got %d, body: %s",
			rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	errMsg, _ := resp["error"].(string)
	if !strings.Contains(errMsg, "检查升级状态失败") {
		t.Errorf("expected error about query update status failed, got %q", errMsg)
	}
}

// ─── waitForOpenclawReady 单元测试 ────────────────────────────────────────────

func TestWaitForOpenclawReady_UnsupportedAgentType(t *testing.T) {
	// 不支持 check_ready 的 agent_type → ResolveScript 返回错误 → 立即返回错误
	ctx := context.Background()
	err := waitForOpenclawReady(ctx, "ins-xxx", "unknown_agent_type", 0)
	if err == nil {
		t.Error("unsupported agent type should return error")
	}
}

func TestWaitForOpenclawReady_EmptyAgentTypeDefaultsToOpenClaw(t *testing.T) {
	// agentType="" 应默认使用 openclaw，ResolveScript 应成功（返回 check_openclaw_ready.sh）
	// 但 RunScript 会失败（无真实 TAT），验证走到了 RunScript 阶段
	// timeout=0 → deadline 立即过期，循环不执行，直接超时
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()
	ctx := context.Background()
	// timeout 极小，deadline 立刻过期，不会真正执行 RunScript
	err := waitForOpenclawReady(ctx, "ins-xxx", "", 0)
	// 应返回超时错误（而非 ResolveScript 错误）
	if err == nil {
		t.Error("should return timeout error")
	}
	if hcommon.ErrorMessageWithCtx(ctx, err) == "" {
		t.Error("error message should not be empty")
	}
}
