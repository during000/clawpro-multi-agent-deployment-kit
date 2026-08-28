package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

// ─── 辅助函数 ────────────────────────────────────────────────────────────────

// initDoctorTestDB 初始化内存 SQLite，迁移龙虾医生相关表，
// 创建默认 SiteConfig（DoctorEnabled=true）、测试用户和实例。
// 返回 cleanup 函数用于恢复全局状态。
func initDoctorTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(
		sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	err = db.AutoMigrate(
		&model.User{},
		&model.Instance{},
		&model.DoctorSession{},
		&model.DoctorAuthorization{},
		&model.SiteConfig{},
		&model.AIImage{},
	)
	if err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	origDB := model.UseDBForTest(db)

	origStore := Store
	Store = sessions.NewCookieStore(
		[]byte("test-secret-key-32-bytes-long!!!"))

	// 创建默认 SiteConfig
	db.Create(&model.SiteConfig{
		DoctorEnabled:   true,
		SecurityGroupId: "sg-test",
	})

	// 创建启用镜像（龙虾医生通过 GetEnabledImageByType 获取）
	db.Create(&model.AIImage{
		ImageId:   "img-test",
		Enabled:   true,
		AgentType: model.AgentTypeOpenClaw,
	})

	// 创建测试用户
	db.Create(&model.User{
		Username: "doctoruser",
		Password: "test",
		Role:     "user",
	})

	// 创建测试实例（归属 user_id=1）
	db.Create(&model.Instance{
		Name:       "test-instance",
		InstanceId: "ins-testdoctor",
		UserID:     1,
	})

	// 默认创建授权记录，使现有诸多 Start 测试不被授权校验拦下；
	// 需要验证未授权场景的测试可以在用例内删除该记录。
	db.Create(&model.DoctorAuthorization{
		UserID:     1,
		InstanceID: 1,
	})

	return func() {
		origDB()
		Store = origStore
	}
}

// doctorReqWithUser 构造带用户 session 的 HTTP 请求。
// userID 对应数据库中 User.ID，函数会查出 username 写入 session。
func doctorReqWithUser(
	t *testing.T, method, path string, userID uint,
) *http.Request {
	t.Helper()
	var user model.User
	if model.DB(context.Background()).First(&user, userID).Error != nil {
		t.Fatalf("用户 ID=%d 不存在", userID)
	}

	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Accept", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = user.Username
	session.Values["role"] = user.Role

	rr := httptest.NewRecorder()
	if err := session.Save(req, rr); err != nil {
		t.Fatalf("保存 session 失败: %v", err)
	}
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

func doctorReqWithUserBody(
	t *testing.T, method, path string, userID uint,
	body io.Reader,
) *http.Request {
	t.Helper()
	var user model.User
	if model.DB(context.Background()).First(&user, userID).Error != nil {
		t.Fatalf("用户 ID=%d 不存在", userID)
	}

	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = user.Username
	session.Values["role"] = user.Role

	rrTmp := httptest.NewRecorder()
	if err := session.Save(req, rrTmp); err != nil {
		t.Fatalf("保存 session 失败: %v", err)
	}
	for _, c := range rrTmp.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

// parseDoctorJSON 解析 JSON 响应体到 map。
func parseDoctorJSON(
	t *testing.T, rr *httptest.ResponseRecorder,
) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(
		rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应 JSON 失败: %v, body=%s",
			err, rr.Body.String())
	}
	return resp
}

// ─── HandleDoctorFeature ─────────────────────────────────────────────────────

func TestDoctorFeature_正常返回已授权(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	// 创建授权记录
	model.DB(context.Background()).Create(&model.DoctorAuthorization{
		UserID: 1, InstanceID: 1,
	})

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/feature?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorFeature(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["ok"] != true {
		t.Errorf("期望 ok=true, 实际 %v", resp["ok"])
	}
	if resp["doctor_enabled"] != true {
		t.Errorf("期望 doctor_enabled=true, 实际 %v",
			resp["doctor_enabled"])
	}
	if resp["authorized"] != true {
		t.Errorf("期望 authorized=true, 实际 %v",
			resp["authorized"])
	}
}

func TestDoctorFeature_正常返回未授权(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	// initDoctorTestDB 默认会创建授权记录，本用例需验证未授权场景
	model.DB(context.Background()).Unscoped().
		Where("user_id = ? AND instance_id = ?", 1, 1).
		Delete(&model.DoctorAuthorization{})

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/feature?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorFeature(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["authorized"] != false {
		t.Errorf("期望 authorized=false, 实际 %v",
			resp["authorized"])
	}
}

func TestDoctorFeature_未登录返回401(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet,
		"/openclaw/doctor/feature?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleDoctorFeature(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("期望 401, 实际 %d", rr.Code)
	}
}

func TestDoctorFeature_方法不对返回405(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/feature?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorFeature(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405, 实际 %d", rr.Code)
	}
}

// ─── HandleDoctorAuthorize ───────────────────────────────────────────────────

func TestDoctorAuthorize_正常授权成功(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	// initDoctorTestDB 默认会创建授权记录，本用例需验证首次授权场景
	model.DB(context.Background()).Unscoped().
		Where("user_id = ? AND instance_id = ?", 1, 1).
		Delete(&model.DoctorAuthorization{})

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/authorize?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorAuthorize(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["ok"] != true {
		t.Errorf("期望 ok=true, 实际 %v", resp["ok"])
	}
	if resp["message"] != "授权成功" {
		t.Errorf("期望 message=授权成功, 实际 %v",
			resp["message"])
	}

	// 验证数据库记录
	var count int64
	model.DB(context.Background()).Model(&model.DoctorAuthorization{}).
		Where("user_id = 1 AND instance_id = 1").
		Count(&count)
	if count != 1 {
		t.Errorf("期望授权记录数=1, 实际 %d", count)
	}
}

func TestDoctorAuthorize_幂等重复授权(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.DoctorAuthorization{
		UserID: 1, InstanceID: 1,
	})

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/authorize?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorAuthorize(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["message"] != "已授权" {
		t.Errorf("期望 message=已授权, 实际 %v",
			resp["message"])
	}
}

func TestDoctorAuthorize_方法不对返回405(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/authorize?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorAuthorize(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405, 实际 %d", rr.Code)
	}
}

// ─── HandleDoctorStart ───────────────────────────────────────────────────────

func TestDoctorStart_功能未开启时拒绝(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Model(&model.SiteConfig{}).
		Where("1 = 1").
		Update("doctor_enabled", false)

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/start?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStart(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["error"] != "doctor_disabled" {
		t.Errorf("期望 error=doctor_disabled, 实际 %v",
			resp["error"])
	}
}

func TestDoctorStart_未授权时拒绝(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	// initDoctorTestDB 默认会创建授权记录，本用例先删掉
	model.DB(context.Background()).Unscoped().
		Where("user_id = ? AND instance_id = ?", 1, 1).
		Delete(&model.DoctorAuthorization{})

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/start?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStart(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["error"] != "not_authorized" {
		t.Errorf("期望 error=not_authorized, 实际 %v",
			resp["error"])
	}

	// 未授权不应创建会话
	var count int64
	model.DB(context.Background()).
		Model(&model.DoctorSession{}).
		Where("user_id = ?", 1).
		Count(&count)
	if count != 0 {
		t.Errorf("期望 未创建会话, 实际 count=%d", count)
	}
}

func TestDoctorStart_安全组未配置时拒绝(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Model(&model.SiteConfig{}).
		Where("1 = 1").
		Update("security_group_id", "")

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/start?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStart(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["error"] != "security_group_not_set" {
		t.Errorf("期望 error=security_group_not_set, 实际 %v",
			resp["error"])
	}
}

func TestDoctorStart_已有活跃会话时拒绝(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 1,
		Status:           model.DoctorStatusActive,
	})

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/start?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStart(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["error"] != "active_session_exists" {
		t.Errorf(
			"期望 error=active_session_exists, 实际 %v",
			resp["error"])
	}
}

func TestDoctorStart_正常创建会话CVM失败返回错误(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	snap := saveDoctorFns()
	defer snap.restore()

	// mock STS 成功，CVM 客户端失败
	doctorRequestSTSFn =
		func(context.Context, string) (*STSCredentials, error) {
			return &STSCredentials{
				SecretId: "a", SecretKey: "b", Token: "c",
			}, nil
		}
	doctorNewCVMClientFn =
		func(context.Context) (*cvm.Client, error) {
			return nil, hcommon.I18nError(i18n.MsgCreateCVMClientFailed)
		}

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/start?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStart(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["ok"] != false {
		t.Errorf("期望 ok=false, 实际 %v", resp["ok"])
	}
	if resp["error"] != "create_failed" {
		t.Errorf("期望 error=create_failed, 实际 %v",
			resp["error"])
	}

	// 验证 session 被标记为 failed
	var session model.DoctorSession
	model.DB(context.Background()).First(&session)
	if session.Status != model.DoctorStatusFailed {
		t.Errorf("期望 status=failed, 实际 %s",
			session.Status)
	}
}

func TestDoctorStart_方法不对返回405(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/start?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStart(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405, 实际 %d", rr.Code)
	}
}

// ─── HandleDoctorStatus ──────────────────────────────────────────────────────

func TestDoctorStatus_正常返回会话状态(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 1,
		Status:           model.DoctorStatusActive,
		HasSnapshot:      true,
	})

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/status?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["ok"] != true {
		t.Errorf("期望 ok=true, 实际 %v", resp["ok"])
	}
	if resp["status"] != model.DoctorStatusActive {
		t.Errorf("期望 status=active, 实际 %v", resp["status"])
	}
	if resp["has_snapshot"] != true {
		t.Errorf("期望 has_snapshot=true, 实际 %v", resp["has_snapshot"])
	}
}

func TestDoctorStatus_无诊断记录(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/status?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["has_active_session"] != false {
		t.Errorf("无记录时应返回 has_active_session=false, got %v",
			resp["has_active_session"])
	}
}

func TestDoctorStatus_方法不对返回405(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/status?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStatus(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405, 实际 %d", rr.Code)
	}
}

// ─── HandleDoctorQuickFix ────────────────────────────────────────────────────

func TestDoctorQuickFix_方法不对返回405(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/quick-fix?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorQuickFix(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405, 实际 %d", rr.Code)
	}
}

func TestDoctorQuickFix_未登录返回401(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost,
		"/openclaw/doctor/quick-fix?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleDoctorQuickFix(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("期望 401, 实际 %d", rr.Code)
	}
}

func TestDoctorQuickFix_实例参数缺失返回400(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/quick-fix", 1)
	rr := httptest.NewRecorder()
	HandleDoctorQuickFix(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望 400, 实际 %d, body=%s",
			rr.Code, rr.Body.String())
	}
}

// ─── HandleDoctorStart snapshot 参数 ────────────────────────────────────────────

func TestDoctorStart_快照参数记录(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	// 需要开启功能 + 配置安全组
	model.DB(context.Background()).Save(&model.SiteConfig{
		DoctorEnabled:   true,
		SecurityGroupId: "sg-test",
	})

	body := strings.NewReader(`{"snapshot":true}`)
	req := doctorReqWithUserBody(t, http.MethodPost,
		"/openclaw/doctor/start?id=1", 1, body)
	rr := httptest.NewRecorder()
	HandleDoctorStart(rr, req)

	// 不管 CVM 创建是否成功，检查 session 是否记录了 snapshot_requested
	var session model.DoctorSession
	if model.DB(context.Background()).First(&session).Error == nil {
		if !session.SnapshotRequested {
			t.Errorf("期望 SnapshotRequested=true")
		}
	}
}

// ─── HandleDoctorEnd 异步 ─────────────────────────────────────────────────

func TestDoctorEnd_异步标记ending(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 1,
		Status:           model.DoctorStatusActive,
	})

	body := strings.NewReader(`{"rollback":true}`)
	req := doctorReqWithUserBody(t, http.MethodPost,
		"/openclaw/doctor/end?id=1", 1, body)
	rr := httptest.NewRecorder()
	HandleDoctorEnd(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["status"] != "ending" {
		t.Errorf("期望 status=ending, 实际 %v", resp["status"])
	}

	// 检查 DB
	var session model.DoctorSession
	model.DB(context.Background()).First(&session)
	if session.Status != model.DoctorStatusEnding {
		t.Errorf("期望 DB status=ending, 实际 %s",
			session.Status)
	}
	if !session.RollbackRequested {
		t.Errorf("期望 RollbackRequested=true")
	}
}

// ─── HandleDoctorEnd ─────────────────────────────────────────────────────────

func TestDoctorEnd_会话不存在(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/end?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorEnd(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["error"] != "session_not_found" {
		t.Errorf("期望 error=session_not_found, 实际 %v",
			resp["error"])
	}
}

func TestDoctorEnd_会话已结束时拒绝(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 1,
		Status:           model.DoctorStatusEnded,
	})

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/end?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorEnd(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["error"] != "session_not_found" {
		t.Errorf(
			"期望 error=session_not_found, 实际 %v",
			resp["error"])
	}
}

func TestDoctorEnd_已失败的会话也拒绝(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID:           1,
		TargetInstanceID: 1,
		Status:           model.DoctorStatusFailed,
	})

	req := doctorReqWithUser(t, http.MethodPost,
		fmt.Sprintf(
			"/openclaw/doctor/end?id=1"),
		1)
	rr := httptest.NewRecorder()
	HandleDoctorEnd(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["error"] != "session_not_found" {
		t.Errorf(
			"期望 error=session_not_found, 实际 %v",
			resp["error"])
	}
}

func TestDoctorEnd_方法不对返回405(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/end?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorEnd(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405, 实际 %d", rr.Code)
	}
}

func TestDoctorEnd_缺少id返回400(t *testing.T) {
	cleanup := initDoctorTestDB(t)
	defer cleanup()

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/end", 1)
	rr := httptest.NewRecorder()
	HandleDoctorEnd(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望 400, 实际 %d", rr.Code)
	}
}
