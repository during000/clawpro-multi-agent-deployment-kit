package controller

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

// doctorFnSnapshot 保存所有可替换函数变量的快照，用于测试后恢复。
type doctorFnSnapshot struct {
	runScript              func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error)
	runScriptAsync         func(context.Context, string, string, uint64, string, map[string]string) (string, error)
	describeInvocationTask func(context.Context, string) (*InvocationTaskResult, error)
	newCVMClient           func(context.Context) (*cvm.Client, error)
	requestSTS             func(context.Context, string) (*STSCredentials, error)
	fetchCVMInfo           func(context.Context, string) (*CVMInstanceInfo, error)
	lookupRuntimeUser      func(context.Context, string) string
	checkGatewayReady      func(context.Context, string, string) bool
	injectDefaultModel     func(context.Context, uint, uint)
	deleteSMHFile          func(context.Context, string) error
	uploadArchive          func(context.Context, string, string, string, int64) (string, error)
	uploadArchiveKey       func(context.Context, string, string, string, int64, string) (string, error)
	buildSMHDownload       func(context.Context, string, bool) (string, error)
	installComponents      func(context.Context, string, string) error
	checkAgentOnline       func(context.Context, string) error
	resolveZone            func(context.Context, string, *model.SiteConfig) string
	approveDeviceAsync     func(context.Context, uint, string, string)
	restartTargetGateway   func(context.Context, *model.Instance)
}

func saveDoctorFns() doctorFnSnapshot {
	return doctorFnSnapshot{
		runScript:              doctorRunScriptFn,
		runScriptAsync:         doctorRunScriptAsyncFn,
		describeInvocationTask: doctorDescribeInvocationTaskFn,
		newCVMClient:           doctorNewCVMClientFn,
		requestSTS:             doctorRequestSTSFn,
		fetchCVMInfo:           doctorFetchCVMInfoFn,
		lookupRuntimeUser:      doctorLookupRuntimeUserFn,
		checkGatewayReady:      doctorCheckGatewayReadyFn,
		injectDefaultModel:     doctorInjectDefaultModelFn,
		deleteSMHFile:          doctorDeleteSMHFileFn,
		uploadArchive:          doctorUploadArchiveFn,
		uploadArchiveKey:       doctorUploadArchiveKeyFn,
		buildSMHDownload:       doctorBuildSMHDownloadFn,
		installComponents:      doctorInstallComponentsFn,
		checkAgentOnline:       doctorCheckAgentOnlineFn,
		resolveZone:            doctorResolveZoneFn,
		approveDeviceAsync:     doctorApproveDeviceAsyncFn,
		restartTargetGateway:   doctorRestartTargetGatewayFn,
	}
}

func (s doctorFnSnapshot) restore() {
	doctorRunScriptFn = s.runScript
	doctorRunScriptAsyncFn = s.runScriptAsync
	doctorDescribeInvocationTaskFn = s.describeInvocationTask
	doctorNewCVMClientFn = s.newCVMClient
	doctorRequestSTSFn = s.requestSTS
	doctorFetchCVMInfoFn = s.fetchCVMInfo
	doctorLookupRuntimeUserFn = s.lookupRuntimeUser
	doctorCheckGatewayReadyFn = s.checkGatewayReady
	doctorInjectDefaultModelFn = s.injectDefaultModel
	doctorDeleteSMHFileFn = s.deleteSMHFile
	doctorUploadArchiveFn = s.uploadArchive
	doctorUploadArchiveKeyFn = s.uploadArchiveKey
	doctorBuildSMHDownloadFn = s.buildSMHDownload
	doctorInstallComponentsFn = s.installComponents
	doctorCheckAgentOnlineFn = s.checkAgentOnline
	doctorResolveZoneFn = s.resolveZone
	doctorApproveDeviceAsyncFn = s.approveDeviceAsync
	doctorRestartTargetGatewayFn = s.restartTargetGateway
}

// ─── 测试基础设施 ───────────────────────────────────────────────────────────

// initDoctorTestDBOnly 初始化内存 SQLite 和基础数据，不做任何 mock。
func initDoctorTestDBOnly(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(
		sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Instance{},
		&model.DoctorSession{},
		&model.DoctorAuthorization{},
		&model.SiteConfig{},
		&model.AIImage{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	origDB := model.UseDBForTest(db)

	// 默认将 doctorApproveDeviceAsyncFn mock 为 noop，
	// 避免异步 goroutine 在 cleanup 后跳到下一个测试访问 DB。
	origApproveAsync := doctorApproveDeviceAsyncFn
	doctorApproveDeviceAsyncFn = func(context.Context, uint, string, string) {}

	origStore := Store
	Store = sessions.NewCookieStore(
		[]byte("test-secret-key-32-bytes-long!!!"))

	db.Create(&model.SiteConfig{
		DoctorEnabled:   true,
		SecurityGroupId: "sg-test",
		VpcId:           "vpc-test",
	})
	db.Create(&model.AIImage{
		ImageId: "img-test", Enabled: true,
		AgentType: model.AgentTypeOpenClaw,
	})
	db.Create(&model.User{
		Username: "doctoruser", Password: "test", Role: "user",
	})
	db.Create(&model.Instance{
		Name: "test-instance", InstanceId: "ins-testdoctor",
		UserID: 1,
	})

	return func() {
		origDB()
		Store = origStore
		doctorApproveDeviceAsyncFn = origApproveAsync
	}
}

// ─── buildDoctorCVMRequest 测试 ─────────────────────────────────────────────

func TestBuildDoctorCVMRequest_基本字段(t *testing.T) {
	snap := saveDoctorFns()
	defer snap.restore()
	doctorResolveZoneFn =
		func(context.Context, string, *model.SiteConfig) string {
			return "ap-guangzhou-3"
		}

	config := &model.SiteConfig{
		SecurityGroupId: "sg-12345",
		VpcId:           "vpc-abc",
		SubnetIds:       `{"ap-guangzhou-3":"subnet-001"}`,
		CVMTemplate:     `{"InstanceType":"S5.MEDIUM2"}`,
	}
	target := &model.Instance{
		InstanceId: "ins-target", AgentType: model.AgentTypeOpenClaw,
	}
	cred := &STSCredentials{
		SecretId: "AKIDxxx", SecretKey: "secret", Token: "tok123",
	}

	req, _, err := buildDoctorCVMRequest(
		context.Background(), config, target,
		"proxy-tok", cred, "img-001")
	if err != nil {
		t.Fatalf("期望成功，实际 err: %v", err)
	}

	if *req.ImageId != "img-001" {
		t.Errorf("ImageId = %s, want img-001", *req.ImageId)
	}
	if *req.InstanceCount != 1 {
		t.Errorf("InstanceCount = %d, want 1", *req.InstanceCount)
	}
	if *req.InstanceChargeType != "POSTPAID_BY_HOUR" {
		t.Errorf("InstanceChargeType = %s", *req.InstanceChargeType)
	}
	if *req.InstanceType != "S5.MEDIUM2" {
		t.Errorf("InstanceType = %s", *req.InstanceType)
	}
	if len(req.SecurityGroupIds) != 1 ||
		*req.SecurityGroupIds[0] != "sg-12345" {
		t.Errorf("SecurityGroupIds = %v", req.SecurityGroupIds)
	}
	if req.EnhancedService == nil ||
		req.EnhancedService.AutomationService == nil ||
		!*req.EnhancedService.AutomationService.Enabled {
		t.Error("TAT 未启用")
	}
	if req.UserData == nil {
		t.Fatal("UserData 为空")
	}
	if req.VirtualPrivateCloud == nil {
		t.Fatal("VPC 未设置")
	}
	if *req.VirtualPrivateCloud.VpcId != "vpc-abc" {
		t.Errorf("VpcId = %s", *req.VirtualPrivateCloud.VpcId)
	}
}

func TestBuildDoctorCVMRequest_UserData包含环境变量(t *testing.T) {
	snap := saveDoctorFns()
	defer snap.restore()
	doctorResolveZoneFn =
		func(context.Context, string, *model.SiteConfig) string {
			return "ap-guangzhou-3"
		}

	config := &model.SiteConfig{
		SecurityGroupId: "sg-1",
		CVMTemplate:     `{"InstanceType":"S5.MEDIUM2"}`,
	}
	target := &model.Instance{
		InstanceId: "ins-target-123", AgentType: "openclaw",
	}
	cred := &STSCredentials{
		SecretId: "AKID_TEST", SecretKey: "SK_TEST", Token: "TOKEN_TEST",
	}

	req, _, err := buildDoctorCVMRequest(
		context.Background(), config, target,
		"pt-123", cred, "img-x")
	if err != nil {
		t.Fatalf("期望成功，实际 err: %v", err)
	}

	decoded, _ := base64.StdEncoding.DecodeString(*req.UserData)
	ud := string(decoded)
	for _, want := range []string{
		`DOCTOR_TARGET_INSTANCE_ID="ins-target-123"`,
		`TEMP_SECRET_ID="AKID_TEST"`,
		`TEMP_SECRET_KEY="SK_TEST"`,
		`TEMP_TOKEN="TOKEN_TEST"`,
		`DOCTOR_TARGET_AGENT_TYPE="openclaw"`,
	} {
		if !strings.Contains(ud, want) {
			t.Errorf("UserData 缺少 %q", want)
		}
	}
}

// ─── ActivateDoctorSession 测试 ──────────────────────────────────────────────

func TestActivateDoctorSession_节点未就绪返回false(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	doctorInstID := uint(99)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-node", InstanceId: "ins-doctor-1",
		UserID: 1, IsDoctorNode: true,
		AgentReady: 0,
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusCreating,
	}
	model.DB(context.Background()).Create(&session)

	result := ActivateDoctorSession(context.Background(), &session)

	if result {
		t.Error("期望返回 false（节点未就绪），实际返回 true")
	}
	var s model.DoctorSession
	model.DB(context.Background()).First(&s, session.ID)
	if s.Status != model.DoctorStatusCreating {
		t.Errorf("status = %s, want creating", s.Status)
	}
}

func TestActivateDoctorSession_安装失败标记failed(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	doctorInstID := uint(99)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-node", InstanceId: "ins-doctor-1",
		UserID: 1, IsDoctorNode: true,
		AgentReady: 1, LastCVMState: "RUNNING",
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusCreating,
	}
	model.DB(context.Background()).Create(&session)

	snap := saveDoctorFns()
	defer snap.restore()

	doctorLookupRuntimeUserFn =
		func(context.Context, string) string { return "root" }
	doctorCheckAgentOnlineFn =
		func(context.Context, string) error { return nil }
	doctorCheckGatewayReadyFn =
		func(context.Context, string, string) bool { return true }
	doctorInstallComponentsFn =
		func(context.Context, string, string) error {
			return fmt.Errorf("install failed")
		}

	result := ActivateDoctorSession(context.Background(), &session)

	if !result {
		t.Error("期望返回 true（已处理），实际返回 false")
	}
	var s model.DoctorSession
	model.DB(context.Background()).First(&s, session.ID)
	if s.Status != model.DoctorStatusFailed {
		t.Errorf("status = %s, want failed", s.Status)
	}
}

func TestActivateDoctorSession_TATAgent未就绪返回false(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	doctorInstID := uint(99)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-node", InstanceId: "ins-doctor-1",
		UserID: 1, IsDoctorNode: true,
		AgentReady: 1, LastCVMState: "RUNNING",
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusCreating,
	}
	model.DB(context.Background()).Create(&session)

	snap := saveDoctorFns()
	defer snap.restore()

	doctorLookupRuntimeUserFn =
		func(context.Context, string) string { return "root" }
	// TAT Agent 未上线
	doctorCheckAgentOnlineFn =
		func(context.Context, string) error {
			return fmt.Errorf("TAT Agent 未就绪")
		}
	// installComponents 不应被调用
	installCalled := false
	doctorInstallComponentsFn =
		func(context.Context, string, string) error {
			installCalled = true
			return nil
		}

	result := ActivateDoctorSession(context.Background(), &session)

	// 期望：返回 false（未完成，下次重试），且 session 仍为 creating
	if result {
		t.Error("期望返回 false（TAT 未就绪应重试），实际返回 true")
	}
	if installCalled {
		t.Error("installComponents 不应被调用，TAT 未就绪应提前跳出")
	}
	var s model.DoctorSession
	model.DB(context.Background()).First(&s, session.ID)
	if s.Status != model.DoctorStatusCreating {
		t.Errorf("status = %s, want creating（不应被误标为 failed）", s.Status)
	}
}

func TestActivateDoctorSession_成功激活(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	doctorInstID := uint(99)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-node", InstanceId: "ins-doctor-1",
		UserID: 1, IsDoctorNode: true,
		AgentReady: 1, LastCVMState: "RUNNING",
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusCreating,
	}
	model.DB(context.Background()).Create(&session)

	snap := saveDoctorFns()
	defer snap.restore()

	doctorLookupRuntimeUserFn =
		func(context.Context, string) string { return "root" }
	doctorCheckAgentOnlineFn =
		func(context.Context, string) error { return nil }
	doctorInstallComponentsFn =
		func(context.Context, string, string) error {
			return nil
		}
	doctorCheckGatewayReadyFn =
		func(context.Context, string, string) bool {
			return true
		}
	// inject 现在是同步
	injectCalled := false
	doctorInjectDefaultModelFn =
		func(context.Context, uint, uint) {
			injectCalled = true
		}
	// 处理 SiteConfig，让 DefaultModelID > 0 才会触发 inject
	model.DB(context.Background()).Model(&model.SiteConfig{}).
		Where("id = 1").Update("default_model_id", 7)

	activationStartedAt := time.Now()
	result := ActivateDoctorSession(context.Background(), &session)

	if !result {
		t.Error("期望返回 true（已激活），实际返回 false")
	}
	if !injectCalled {
		t.Error("inject 同步未被调用")
	}
	var s model.DoctorSession
	model.DB(context.Background()).First(&s, session.ID)
	if s.Status != model.DoctorStatusActive {
		t.Errorf("status = %s, want active", s.Status)
	}
	if s.ActivatedAt == nil {
		t.Fatal("activated_at 为空，成功激活时应记录激活时间")
	}
	if s.ActivatedAt.Before(activationStartedAt.Add(-time.Second)) ||
		s.ActivatedAt.After(time.Now().Add(time.Second)) {
		t.Errorf("activated_at = %v，不在本次激活时间范围内",
			s.ActivatedAt)
	}
}

func TestActivateDoctorSession_激活状态写入失败(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	doctorInstID := uint(199)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-node-update-failed", InstanceId: "ins-doctor-update-failed",
		UserID: 1, IsDoctorNode: true,
		AgentReady: 1, LastCVMState: "RUNNING",
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusCreating,
	}
	model.DB(context.Background()).Create(&session)

	snap := saveDoctorFns()
	defer snap.restore()

	doctorLookupRuntimeUserFn =
		func(context.Context, string) string { return "root" }
	doctorCheckAgentOnlineFn =
		func(context.Context, string) error { return nil }
	doctorInstallComponentsFn =
		func(context.Context, string, string) error { return nil }
	doctorCheckGatewayReadyFn =
		func(context.Context, string, string) bool { return true }

	db := model.DB(context.Background())
	const callbackName = "test:doctor_activate_update_error"
	if err := db.Callback().Update().Before("gorm:update").
		Register(callbackName, func(tx *gorm.DB) {
			if tx.Statement.Table == "doctor_sessions" {
				tx.AddError(errors.New("mock activate update failure"))
			}
		}); err != nil {
		t.Fatalf("register update callback: %v", err)
	}
	defer db.Callback().Update().Remove(callbackName)

	if ActivateDoctorSession(context.Background(), &session) {
		t.Error("激活状态写入失败时应返回 false")
	}

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusCreating {
		t.Errorf("status = %s, want creating", updated.Status)
	}
	if updated.ActivatedAt != nil {
		t.Errorf("activated_at = %v, want nil", updated.ActivatedAt)
	}
}

func TestActivateDoctorSession_inject同步阻塞主流程(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	doctorInstID := uint(99)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-node", InstanceId: "ins-doctor-1",
		UserID: 1, IsDoctorNode: true,
		AgentReady: 1, LastCVMState: "RUNNING",
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusCreating,
	}
	model.DB(context.Background()).Create(&session)

	snap := saveDoctorFns()
	defer snap.restore()

	doctorLookupRuntimeUserFn =
		func(context.Context, string) string { return "root" }
	doctorCheckAgentOnlineFn =
		func(context.Context, string) error { return nil }
	doctorInstallComponentsFn =
		func(context.Context, string, string) error {
			return nil
		}
	doctorCheckGatewayReadyFn =
		func(context.Context, string, string) bool {
			return true
		}

	// 同步调用：inject 耗时 200ms，主流程必须等它跑完才返回。
	doctorInjectDefaultModelFn =
		func(context.Context, uint, uint) {
			time.Sleep(200 * time.Millisecond)
		}
	// 让 DefaultModelID > 0 才会触发 inject
	model.DB(context.Background()).Model(&model.SiteConfig{}).
		Where("id = 1").Update("default_model_id", 7)

	start := time.Now()
	result := ActivateDoctorSession(context.Background(), &session)
	elapsed := time.Since(start)

	if !result {
		t.Error("期望返回 true")
	}
	// 主流程必须等 inject 同步完成
	if elapsed < 200*time.Millisecond {
		t.Errorf("主流程未等 inject 同步完成，耗时 %v", elapsed)
	}
	var s model.DoctorSession
	model.DB(context.Background()).First(&s, session.ID)
	if s.Status != model.DoctorStatusActive {
		t.Errorf("status = %s, want active", s.Status)
	}
}

func TestActivateDoctorSession_快照失败标failed(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	doctorInstID := uint(99)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-node", InstanceId: "ins-doctor-1",
		UserID: 1, IsDoctorNode: true,
		AgentReady: 1, LastCVMState: "RUNNING",
	})
	// 目标实例（ID 不能为 1，initDoctorTestDBOnly 已创建 ID=1 的实例）
	targetInstID := uint(200)
	model.DB(context.Background()).Create(&model.Instance{
		Model:      gorm.Model{ID: targetInstID},
		InstanceId: "ins-target-1",
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: targetInstID,
		DoctorInstanceID:  &doctorInstID,
		Status:            model.DoctorStatusCreating,
		SnapshotRequested: true,
		HasSnapshot:       false,
	}
	model.DB(context.Background()).Create(&session)

	snap := saveDoctorFns()
	defer snap.restore()

	doctorLookupRuntimeUserFn =
		func(context.Context, string) string { return "root" }
	doctorCheckAgentOnlineFn =
		func(context.Context, string) error { return nil }
	doctorInstallComponentsFn =
		func(context.Context, string, string) error { return nil }
	doctorCheckGatewayReadyFn =
		func(context.Context, string, string) bool { return true }
	// 快照脚本失败
	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			return "", hcommon.I18nError(i18n.MsgOperationFailed)
		}

	result := ActivateDoctorSession(context.Background(), &session)

	if !result {
		t.Error("期望返回 true（终态）")
	}
	var s model.DoctorSession
	model.DB(context.Background()).First(&s, session.ID)
	if s.Status != model.DoctorStatusFailed {
		t.Errorf("status = %s, want failed", s.Status)
	}
	if s.HasSnapshot {
		t.Error("HasSnapshot 不应被设为 true")
	}
}

func TestActivateDoctorSession_无DoctorInstanceID(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: nil,
		Status:           model.DoctorStatusCreating,
	}
	model.DB(context.Background()).Create(&session)

	result := ActivateDoctorSession(context.Background(), &session)

	if result {
		t.Error("期望返回 false，实际返回 true")
	}
}

// ─── CleanupDoctorSession 测试 ──────────────────────────────────────────────

func TestCleanupDoctorSession_完整清理流程(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	doctorInstID := uint(99)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-node", InstanceId: "ins-doctor-1",
		UserID: 1, IsDoctorNode: true,
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusEnding, HasSnapshot: true,
		SnapshotFileKey: "snapshot/key.tar.gz",
	}
	model.DB(context.Background()).Create(&session)

	deletedFiles := []string{}

	snap := saveDoctorFns()
	defer snap.restore()

	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			return "NO_SESSIONS", nil
		}
	doctorDeleteSMHFileFn =
		func(_ context.Context, key string) error {
			deletedFiles = append(deletedFiles, key)
			return nil
		}
	doctorNewCVMClientFn =
		func(context.Context) (*cvm.Client, error) {
			return nil, hcommon.I18nError(i18n.MsgCreateCVMClientFailed)
		}
	doctorUploadArchiveFn =
		func(context.Context, string, string, string, int64) (string, error) {
			return "", nil
		}

	CleanupDoctorSession(context.Background(), &session)

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusEnded {
		t.Errorf("status = %s, want ended", updated.Status)
	}
	found := false
	for _, f := range deletedFiles {
		if f == "snapshot/key.tar.gz" {
			found = true
		}
	}
	if !found {
		t.Errorf("快照 key 不匹配, deletedFiles=%v", deletedFiles)
	}
}

func TestCleanupDoctorSession_无快照无龙虾实例(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		Status: model.DoctorStatusEnding, HasSnapshot: false,
	}
	model.DB(context.Background()).Create(&session)

	deleteCalled := false
	snap := saveDoctorFns()
	defer snap.restore()

	doctorDeleteSMHFileFn =
		func(context.Context, string) error { deleteCalled = true; return nil }
	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			return "NO_SESSIONS", nil
		}
	doctorUploadArchiveFn =
		func(context.Context, string, string, string, int64) (string, error) {
			return "", nil
		}

	CleanupDoctorSession(context.Background(), &session)

	if deleteCalled {
		t.Error("不应调用 SMH 删除")
	}
	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusEnded {
		t.Errorf("status = %s, want ended", updated.Status)
	}
}

// ─── RefreshDoctorSTS 测试 ──────────────────────────────────────────────────

func TestRefreshDoctorSTS_刷新即将过期的密钥(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	doctorInstID := uint(50)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-sts", InstanceId: "ins-doctor-sts",
		UserID: 1, IsDoctorNode: true,
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		STSExpiredAt:     time.Now().Unix() + 120, // 2 分钟后过期
	}
	model.DB(context.Background()).Create(&session)
	fixedUpdatedAt := time.Now().Add(-6 * time.Hour).Truncate(time.Millisecond)
	model.DB(context.Background()).Model(&session).
		UpdateColumn("updated_at", fixedUpdatedAt)

	var beforeRefresh model.DoctorSession
	model.DB(context.Background()).First(&beforeRefresh, session.ID)

	stsRefreshed := false
	scriptRan := false
	snap := saveDoctorFns()
	defer snap.restore()

	doctorRequestSTSFn =
		func(context.Context, string) (*STSCredentials, error) {
			stsRefreshed = true
			return &STSCredentials{
				SecretId: "new-id", SecretKey: "new-key",
				Token: "new-token",
			}, nil
		}
	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			scriptRan = true
			return "STS_REFRESHED", nil
		}

	RefreshDoctorSTS(context.Background())

	if !stsRefreshed {
		t.Error("STS 未刷新")
	}
	if !scriptRan {
		t.Error("TAT 脚本未执行")
	}
	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.STSExpiredAt <= session.STSExpiredAt {
		t.Errorf("STSExpiredAt 未更新: old=%d, new=%d",
			session.STSExpiredAt, updated.STSExpiredAt)
	}
	if !updated.UpdatedAt.Equal(beforeRefresh.UpdatedAt) {
		t.Errorf("STS 刷新不应改变 updated_at: before=%v, after=%v",
			beforeRefresh.UpdatedAt, updated.UpdatedAt)
	}
}

func TestRefreshDoctorSTS_未到期则跳过(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	doctorInstID := uint(51)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-sts2", InstanceId: "ins-doctor-sts2",
		UserID: 1, IsDoctorNode: true,
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		STSExpiredAt:     time.Now().Unix() + 3600,
	}
	model.DB(context.Background()).Create(&session)

	stsRefreshed := false
	snap := saveDoctorFns()
	defer snap.restore()

	doctorRequestSTSFn =
		func(context.Context, string) (*STSCredentials, error) {
			stsRefreshed = true
			return &STSCredentials{}, nil
		}

	RefreshDoctorSTS(context.Background())

	if stsRefreshed {
		t.Error("STS 不应被刷新（还有 1 小时）")
	}
}

func TestRefreshDoctorSTS_STS申请失败不崩溃(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	doctorInstID := uint(52)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-sts3", InstanceId: "ins-doctor-sts3",
		UserID: 1, IsDoctorNode: true,
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		STSExpiredAt:     time.Now().Unix() + 120,
	}
	model.DB(context.Background()).Create(&session)

	snap := saveDoctorFns()
	defer snap.restore()

	doctorRequestSTSFn =
		func(context.Context, string) (*STSCredentials, error) {
			return nil, hcommon.I18nError(i18n.MsgSTSServiceDown)
		}

	// 不应 panic
	RefreshDoctorSTS(context.Background())
}

func TestRefreshDoctorSTS_过期时间写入失败(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	doctorInstID := uint(52)
	model.DB(context.Background()).Create(&model.Instance{
		Model: gorm.Model{ID: doctorInstID},
		Name:  "doctor-sts-update-failed", InstanceId: "ins-doctor-sts-update-failed",
		UserID: 1, IsDoctorNode: true,
	})

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &doctorInstID,
		Status:           model.DoctorStatusActive,
		STSExpiredAt:     time.Now().Unix() + 120,
	}
	model.DB(context.Background()).Create(&session)

	snap := saveDoctorFns()
	defer snap.restore()

	doctorRequestSTSFn =
		func(context.Context, string) (*STSCredentials, error) {
			return &STSCredentials{
				SecretId: "new-id", SecretKey: "new-key",
				Token: "new-token",
			}, nil
		}
	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			return "STS_REFRESHED", nil
		}

	db := model.DB(context.Background())
	const callbackName = "test:doctor_sts_update_error"
	if err := db.Callback().Update().Before("gorm:update").
		Register(callbackName, func(tx *gorm.DB) {
			if tx.Statement.Table == "doctor_sessions" {
				tx.AddError(errors.New("mock STS update failure"))
			}
		}); err != nil {
		t.Fatalf("register update callback: %v", err)
	}
	defer db.Callback().Update().Remove(callbackName)

	RefreshDoctorSTS(context.Background())

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.STSExpiredAt != session.STSExpiredAt {
		t.Errorf("STSExpiredAt = %d, want unchanged %d",
			updated.STSExpiredAt, session.STSExpiredAt)
	}
}

// ─── UploadDoctorSessions 测试 ──────────────────────────────────────────────

func TestUploadDoctorSessions_无session文件则跳过(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	uploadCalled := false
	snap := saveDoctorFns()
	defer snap.restore()

	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			return "NO_SESSIONS", nil
		}
	doctorUploadArchiveFn =
		func(context.Context, string, string, string, int64) (string, error) {
			uploadCalled = true
			return "", nil
		}

	UploadDoctorSessions(context.Background(), "ins-test", 1, 1)

	if uploadCalled {
		t.Error("不应调用 SMH 上传（无 session 文件）")
	}
}

func TestUploadDoctorSessions_打包失败则跳过(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	uploadCalled := false
	snap := saveDoctorFns()
	defer snap.restore()

	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			return "", hcommon.I18nError(i18n.MsgTATFailed)
		}
	doctorUploadArchiveFn =
		func(context.Context, string, string, string, int64) (string, error) {
			uploadCalled = true
			return "", nil
		}

	UploadDoctorSessions(context.Background(), "ins-test", 1, 1)

	if uploadCalled {
		t.Error("不应调用 SMH 上传（打包失败）")
	}
}

func TestUploadDoctorSessions_正常上传(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	uploadCalled := false
	snap := saveDoctorFns()
	defer snap.restore()

	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			return "12345", nil
		}
	doctorUploadArchiveKeyFn =
		func(_ context.Context, instId, user, path string, size int64, key string) (string, error) {
			uploadCalled = true
			if size != 12345 {
				t.Errorf("archive size = %d, want 12345", size)
			}
			if key != "doctor-sessions/1/1/sessions.tar.gz" {
				t.Errorf("fileKey = %s, want doctor-sessions/1/1/sessions.tar.gz", key)
			}
			return key, nil
		}

	UploadDoctorSessions(context.Background(), "ins-test", 1, 1)

	if !uploadCalled {
		t.Error("应调用 SMH 上传")
	}
}

// ─── GetDoctorSessionMtime 测试 ─────────────────────────────────────────────

func TestGetDoctorSessionMtime_正常返回(t *testing.T) {
	now := time.Now().Unix()
	snap := saveDoctorFns()
	defer snap.restore()

	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			return fmt.Sprintf("%d", now), nil
		}

	mtime := GetDoctorSessionMtime(context.Background(), "ins-test")
	if mtime.Err != nil {
		t.Errorf("unexpected error: %v", mtime.Err)
	}
	if mtime.Mtime.IsZero() {
		t.Error("mtime 不应为零值")
	}
	if mtime.Mtime.Unix() != now {
		t.Errorf("mtime = %d, want %d", mtime.Mtime.Unix(), now)
	}
}

func TestGetDoctorSessionMtime_无文件返回NoFiles(t *testing.T) {
	snap := saveDoctorFns()
	defer snap.restore()

	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			return "NO_FILES", nil
		}

	mtime := GetDoctorSessionMtime(context.Background(), "ins-test")
	if !mtime.NoFiles {
		t.Error("NoFiles 应为 true")
	}
	if !mtime.Mtime.IsZero() {
		t.Errorf("mtime 应为零值, got %v", mtime.Mtime)
	}
}

func TestGetDoctorSessionMtime_TAT失败返回Err(t *testing.T) {
	snap := saveDoctorFns()
	defer snap.restore()

	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			return "", hcommon.I18nError(i18n.MsgTATTimeout)
		}

	mtime := GetDoctorSessionMtime(context.Background(), "ins-test")
	if mtime.Err == nil {
		t.Error("应返回 error")
	}
}

// ─── installDoctorComponents 测试 ───────────────────────────────────────────

func TestInstallDoctorComponents_全部成功(t *testing.T) {
	snap := saveDoctorFns()
	defer snap.restore()

	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			return "OK", nil
		}

	err := installDoctorComponents(context.Background(), "ins-test", "root")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInstallDoctorComponents_CLI安装失败(t *testing.T) {
	callCount := 0
	snap := saveDoctorFns()
	defer snap.restore()

	doctorRunScriptFn =
		func(_ context.Context, _ string, script string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
			callCount++
			if callCount == 1 { // CLI 安装是第一个脚本
				return "", hcommon.I18nError(i18n.MsgDoctorInstallCLIFailed)
			}
			return "OK", nil
		}

	err := installDoctorComponents(context.Background(), "ins-test", "root")
	if err == nil {
		t.Error("expected error")
	}
}

// ─── HandleDoctorStart AgentType 校验 ───────────────────────────────────────

func TestDoctorStart_不支持的AgentType(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.Instance{
		Name: "hermes-instance", InstanceId: "ins-hermes",
		UserID: 1, AgentType: "hermes",
	})

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/start?id=2", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStart(rr, req)

	resp := parseDoctorJSON(t, rr)
	if resp["error"] != "unsupported_agent_type" {
		t.Errorf("expected error=unsupported_agent_type, got %v",
			resp["error"])
	}
}

// ─── HandleDoctorEnd 异步集成测试 ───────────────────────────────────────────

func TestDoctorEnd_正常结束无回滚(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		Status: model.DoctorStatusActive,
	})

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/end?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorEnd(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}
	if resp["status"] != "ending" {
		t.Errorf("expected status=ending, got %v", resp["status"])
	}

	// 检查 DB 状态
	var session model.DoctorSession
	model.DB(context.Background()).First(&session)
	if session.Status != model.DoctorStatusEnding {
		t.Errorf("DB status = %s, want ending", session.Status)
	}
	if session.RollbackRequested {
		t.Errorf("expected RollbackRequested=false")
	}
}

func TestDoctorEnd_回滚请求记录(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		Status: model.DoctorStatusActive, HasSnapshot: true,
		SnapshotFileKey: "snapshot/rb.tar.gz",
	})

	// 构造带 rollback body 的请求
	req := httptest.NewRequest(http.MethodPost,
		"/openclaw/doctor/end?id=1",
		strings.NewReader(`{"rollback":true}`))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	sess, _ := Store.Get(req, "hatchery-session")
	sess.Values["username"] = "doctoruser"
	sess.Values["role"] = "user"
	w := httptest.NewRecorder()
	sess.Save(req, w)
	for _, c := range w.Result().Cookies() {
		req.AddCookie(c)
	}

	rr := httptest.NewRecorder()
	HandleDoctorEnd(rr, req)

	resp := parseDoctorJSON(t, rr)
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}
	if resp["status"] != "ending" {
		t.Errorf("expected status=ending, got %v", resp["status"])
	}

	// 检查 DB
	var session model.DoctorSession
	model.DB(context.Background()).First(&session)
	if session.Status != model.DoctorStatusEnding {
		t.Errorf("DB status = %s, want ending", session.Status)
	}
	if !session.RollbackRequested {
		t.Errorf("expected RollbackRequested=true")
	}
}

func TestDoctorEnd_Creating状态可以结束(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		Status: model.DoctorStatusCreating,
	})

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/end?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorEnd(rr, req)

	resp := parseDoctorJSON(t, rr)
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}
	var session model.DoctorSession
	model.DB(context.Background()).First(&session)
	if session.Status != model.DoctorStatusEnding {
		t.Errorf("status = %s, want ending", session.Status)
	}
}

// ─── HandleDoctorQuickFix （异步下发） ───────────────────────────────────────────────────

func TestDoctorQuickFix_下发成功(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	snap := saveDoctorFns()
	defer snap.restore()

	doctorRunScriptAsyncFn =
		func(context.Context, string, string, uint64, string, map[string]string) (string, error) {
			return "inv-test-123", nil
		}

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/quick-fix?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorQuickFix(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}
	if resp["invocation_id"] != "inv-test-123" {
		t.Errorf("invocation_id = %v", resp["invocation_id"])
	}
}

func TestDoctorQuickFix_下发失败(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	snap := saveDoctorFns()
	defer snap.restore()

	doctorRunScriptAsyncFn =
		func(context.Context, string, string, uint64, string, map[string]string) (string, error) {
			return "", hcommon.I18nError(i18n.MsgTATCommandDispatchFailed)
		}

	req := doctorReqWithUser(t, http.MethodPost,
		"/openclaw/doctor/quick-fix?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorQuickFix(rr, req)

	resp := parseDoctorJSON(t, rr)
	if resp["ok"] != false {
		t.Errorf("expected ok=false, got %v", resp["ok"])
	}
	if resp["error"] != "fix_failed" {
		t.Errorf("error = %v", resp["error"])
	}
}

// ─── HandleDoctorQuickFixStatus ─────────────────────────────────────────────────

func TestDoctorQuickFixStatus_执行中(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	snap := saveDoctorFns()
	defer snap.restore()

	doctorDescribeInvocationTaskFn =
		func(context.Context, string) (*InvocationTaskResult, error) {
			return &InvocationTaskResult{
				Status:   "RUNNING",
				Output:   "正在检查...",
				Finished: false,
			}, nil
		}

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/quick-fix/status?id=1&invocation_id=inv-123", 1)
	rr := httptest.NewRecorder()
	HandleDoctorQuickFixStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	resp := parseDoctorJSON(t, rr)
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}
	if resp["status"] != "RUNNING" {
		t.Errorf("status = %v", resp["status"])
	}
	if resp["finished"] != false {
		t.Errorf("expected finished=false, got %v", resp["finished"])
	}
}

func TestDoctorQuickFixStatus_执行完成(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	snap := saveDoctorFns()
	defer snap.restore()

	doctorDescribeInvocationTaskFn =
		func(context.Context, string) (*InvocationTaskResult, error) {
			return &InvocationTaskResult{
				Status:   "SUCCESS",
				Output:   "All checks passed",
				ExitCode: 0,
				Finished: true,
			}, nil
		}

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/quick-fix/status?id=1&invocation_id=inv-123", 1)
	rr := httptest.NewRecorder()
	HandleDoctorQuickFixStatus(rr, req)

	resp := parseDoctorJSON(t, rr)
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}
	if resp["status"] != "SUCCESS" {
		t.Errorf("status = %v", resp["status"])
	}
	if resp["finished"] != true {
		t.Errorf("expected finished=true, got %v", resp["finished"])
	}
	if resp["output"] != "All checks passed" {
		t.Errorf("output = %v", resp["output"])
	}
}

func TestDoctorQuickFixStatus_缺少invocation_id(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/quick-fix/status?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorQuickFixStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ─── createDoctorSnapshot ───────────────────────────────────────────────────

func TestCreateDoctorSnapshot_正常创建快照(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.Instance{
		Model:      gorm.Model{ID: 99},
		InstanceId: "ins-target",
	})
	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID: 1, TargetInstanceID: 99,
		Status: model.DoctorStatusActive, HasSnapshot: false,
		SnapshotRequested: true,
	})

	snap := saveDoctorFns()
	defer snap.restore()

	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			return "BACKUP_DIR_PATH:/tmp/backup.tar.gz\nARCHIVE_SIZE:1024", nil
		}
	doctorUploadArchiveFn =
		func(context.Context, string, string, string, int64) (string, error) {
			return "doctor-backup/test.tgz", nil
		}
	restartCalled := false
	doctorRestartTargetGatewayFn =
		func(_ context.Context, inst *model.Instance) {
			restartCalled = true
			if inst == nil || inst.InstanceId != "ins-target" {
				t.Errorf("期望重启目标实例 ins-target, 实际 %+v", inst)
			}
		}

	var session model.DoctorSession
	model.DB(context.Background()).First(&session)

	if err := createDoctorSnapshot(
		context.Background(), &session); err != nil {
		t.Fatalf("期望 err=nil, 实际: %v", err)
	}

	// 检查 DB 更新
	model.DB(context.Background()).First(&session)
	if !session.HasSnapshot {
		t.Errorf("期望 HasSnapshot=true")
	}
	if session.SnapshotFileKey != "doctor-backup/test.tgz" {
		t.Errorf("期望 SnapshotFileKey=doctor-backup/test.tgz, 实际 %s",
			session.SnapshotFileKey)
	}
	if !restartCalled {
		t.Error("期望备份成功后 defer 触发目标实例 Gateway 重启，实际未调用")
	}
}

func TestCreateDoctorSnapshot_备份脚本失败返回err(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.Instance{
		Model:      gorm.Model{ID: 99},
		InstanceId: "ins-target",
	})
	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID: 1, TargetInstanceID: 99,
		Status: model.DoctorStatusCreating, HasSnapshot: false,
		SnapshotRequested: true,
	})

	snap := saveDoctorFns()
	defer snap.restore()

	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			return "", hcommon.I18nError(i18n.MsgOperationFailed)
		}
	restartCalled := false
	doctorRestartTargetGatewayFn =
		func(context.Context, *model.Instance) { restartCalled = true }

	var session model.DoctorSession
	model.DB(context.Background()).First(&session)
	err := createDoctorSnapshot(context.Background(), &session)
	if err == nil {
		t.Fatal("期望 err 非nil")
	}
	if !strings.Contains(err.Error(), "快照备份失败") {
		t.Errorf("错误信息应包含 “快照备份失败”: %v", err)
	}

	// HasSnapshot 应仍为 false
	model.DB(context.Background()).First(&session)
	if session.HasSnapshot {
		t.Error("HasSnapshot 不应被设为 true")
	}
	// 备份脚本本身没跑成功，说明 Gateway 未被停止，不应触发兜底重启
	if restartCalled {
		t.Error("备份脚本失败（Gateway 未被停止）时不应触发兜底重启")
	}
}

func TestCreateDoctorSnapshot_上传失败返回err(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.Instance{
		Model:      gorm.Model{ID: 99},
		InstanceId: "ins-target",
	})
	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID: 1, TargetInstanceID: 99,
		Status: model.DoctorStatusCreating, HasSnapshot: false,
		SnapshotRequested: true,
	})

	snap := saveDoctorFns()
	defer snap.restore()

	doctorRunScriptFn =
		func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
			return "BACKUP_DIR_PATH:/tmp/backup.tar.gz\nARCHIVE_SIZE:1024", nil
		}
	doctorUploadArchiveFn =
		func(context.Context, string, string, string, int64) (string, error) {
			return "", hcommon.I18nError(i18n.MsgSMHUploadFailed)
		}
	restartCalled := false
	doctorRestartTargetGatewayFn =
		func(context.Context, *model.Instance) { restartCalled = true }

	var session model.DoctorSession
	model.DB(context.Background()).First(&session)
	err := createDoctorSnapshot(context.Background(), &session)
	if err == nil {
		t.Fatal("期望 err 非nil")
	}
	if !strings.Contains(err.Error(), "上传失败") {
		t.Errorf("错误信息应包含 “上传失败”: %v", err)
	}
	// 备份脚本已经跑成功（Gateway 已被停止），即使后续上传失败也必须兜底重启
	if !restartCalled {
		t.Error("上传失败但备份已停止 Gateway 时，仍应触发兜底重启")
	}
}

// ─── restartDoctorTargetGateway ─────────────────────────────────────────────
// 直接测试真实函数体（不 mock doctorRestartTargetGatewayFn），
// 通过 mock 更底层的 agentScriptRunner 覆盖成功/失败两条分支。

func TestRestartDoctorTargetGateway_成功(t *testing.T) {
	origRunner := agentScriptRunner
	defer func() { agentScriptRunner = origRunner }()

	var gotInstanceID, gotScriptName string
	agentScriptRunner = func(_ context.Context, instanceId, scriptName string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		gotInstanceID = instanceId
		gotScriptName = scriptName
		return "gateway restarted", nil
	}

	target := &model.Instance{
		Model:       gorm.Model{ID: 1},
		InstanceId:  "ins-target-real",
		AgentType:   model.AgentTypeOpenClaw,
		RuntimeUser: "agentuser",
	}

	// 不应 panic，且应正确下发 restart_gateway.sh 到目标实例
	restartDoctorTargetGateway(context.Background(), target)

	if gotInstanceID != "ins-target-real" {
		t.Errorf("instanceID = %s, want ins-target-real", gotInstanceID)
	}
	if gotScriptName != "restart_gateway.sh" {
		t.Errorf("scriptName = %s, want restart_gateway.sh", gotScriptName)
	}
}

func TestRestartDoctorTargetGateway_失败仅记录日志不panic(t *testing.T) {
	origRunner := agentScriptRunner
	defer func() { agentScriptRunner = origRunner }()

	agentScriptRunner = func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgOperationFailed)
	}

	target := &model.Instance{
		Model:       gorm.Model{ID: 2},
		InstanceId:  "ins-target-fail",
		AgentType:   model.AgentTypeOpenClaw,
		RuntimeUser: "agentuser",
	}

	// 重启失败时函数应仅记录日志、不 panic、不向上抛错（最佳努力语义）
	restartDoctorTargetGateway(context.Background(), target)
}

// ─── HandleDoctorStatus ─────────────────────────────────────────────────────

func TestDoctorStatus_Failed状态含错误信息(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		Status: model.DoctorStatusFailed,
	})

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/status?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStatus(rr, req)

	resp := parseDoctorJSON(t, rr)
	if resp["error_message"] != "节点创建失败" {
		t.Errorf("error_message = %v", resp["error_message"])
	}
}

func TestDoctorStatus_有DoctorInstanceID(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	did := uint(77)
	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		DoctorInstanceID: &did,
		Status:           model.DoctorStatusActive,
	})

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/status?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStatus(rr, req)

	resp := parseDoctorJSON(t, rr)
	if resp["doctor_instance_db_id"] == nil {
		t.Error("expected doctor_instance_db_id")
	}
}

// ─── DoctorSessionsSMHKey ───────────────────────────────────────────────────

func TestDoctorSessionsSMHKey(t *testing.T) {
	key := DoctorSessionsSMHKey(42, 7)
	expected := "doctor-sessions/42/7/sessions.tar.gz"
	if key != expected {
		t.Errorf("key = %s, want %s", key, expected)
	}
}

// ─── cleanupDoctorInstance ──────────────────────────────────────────────────

func TestCleanupDoctorInstance(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.Instance{
		Name: "to-delete", InstanceId: "ins-del",
		UserID: 1, IsDoctorNode: true,
	})

	var inst model.Instance
	model.DB(context.Background()).
		Where("name = ?", "to-delete").First(&inst)

	cleanupDoctorInstance(context.Background(), inst.ID)

	var count int64
	model.DB(context.Background()).Model(&model.Instance{}).
		Where("name = ?", "to-delete").Count(&count)
	if count != 0 {
		t.Errorf("instance should be deleted, count=%d", count)
	}
}

// ─── HandleDoctorStatus 不传 session_id 测试 ────────────────────────────────

func TestDoctorStatus_不传sessionID_有活跃会话(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		Status: model.DoctorStatusActive,
	})

	// 只传 id，不传 session_id
	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/status?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStatus(rr, req)

	resp := parseDoctorJSON(t, rr)
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}
	// 应返回找到的活跃会话
	if resp["status"] != "active" {
		t.Errorf("status = %v, want active", resp["status"])
	}
}

func TestDoctorStatus_只有已结束会话也能返回(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	// 只有 ended 的会话，现在也能返回（不再过滤 status）
	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		Status: model.DoctorStatusEnded,
	})

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/status?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStatus(rr, req)

	resp := parseDoctorJSON(t, rr)
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}
	if resp["status"] != "ended" {
		t.Errorf("status = %v, want ended", resp["status"])
	}
}

func TestDoctorStatus_不传sessionID_返回最新会话(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	// 创建两个活跃会话，应返回最新的（ID 更大）
	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		Status: model.DoctorStatusActive,
	})
	model.DB(context.Background()).Create(&model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		Status: model.DoctorStatusCreating,
	})

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/status?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStatus(rr, req)

	resp := parseDoctorJSON(t, rr)
	// 最新的 session（ID=2）状态是 creating
	if resp["status"] != "creating" {
		t.Errorf("status = %v, want creating (latest)", resp["status"])
	}
}

func TestDoctorStatus_不传id返回400(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/status", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestDoctorStatus_无记录返回NoSession(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	// 实例存在但无任何诊断记录
	req := doctorReqWithUser(t, http.MethodGet,
		"/openclaw/doctor/status?id=1", 1)
	rr := httptest.NewRecorder()
	HandleDoctorStatus(rr, req)

	resp := parseDoctorJSON(t, rr)
	if resp["has_active_session"] != false {
		t.Errorf("expected has_active_session=false, got %v",
			resp["has_active_session"])
	}
}

// ─── buildDoctorCVMRequest / resolveDoctorZone 测试 ────────────────────────

func TestBuildDoctorCVMRequest_zone查不到返回错误(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	snap := saveDoctorFns()
	defer snap.restore()

	// 模拟任何手段都查不到 zone
	doctorResolveZoneFn =
		func(context.Context, string, *model.SiteConfig) string {
			return ""
		}

	cfg := &model.SiteConfig{
		CVMTemplate: `{"InstanceType":"S5.MEDIUM2"}`,
	}
	target := &model.Instance{
		VpcId: "vpc-x", SubnetId: "subnet-x",
		SecurityGroupId: "sg-x",
	}
	request, info, err := buildDoctorCVMRequest(
		context.Background(), cfg, target,
		"proxy-token", &STSCredentials{}, "img-1")

	if err == nil {
		t.Fatal("期望返回 error，实际为 nil")
	}
	if !strings.Contains(err.Error(), "subnet-x") {
		t.Errorf("错误信息应包含 subnetId，实际: %v", err)
	}
	if request != nil {
		t.Error("期望 request 为 nil")
	}
	if info != (doctorNetworkInfo{}) {
		t.Errorf("期望 info 为零值，实际: %+v", info)
	}
}

func TestBuildDoctorCVMRequest_zone查到成功返回(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	snap := saveDoctorFns()
	defer snap.restore()

	doctorResolveZoneFn =
		func(_ context.Context, subnetId string, _ *model.SiteConfig) string {
			if subnetId == "subnet-ok" {
				return "ap-guangzhou-7"
			}
			return ""
		}

	cfg := &model.SiteConfig{
		CVMTemplate: `{"InstanceType":"S5.MEDIUM2"}`,
	}
	target := &model.Instance{
		VpcId: "vpc-x", SubnetId: "subnet-ok",
		SecurityGroupId: "sg-x",
	}
	request, info, err := buildDoctorCVMRequest(
		context.Background(), cfg, target,
		"proxy-token", &STSCredentials{}, "img-1")

	if err != nil {
		t.Fatalf("期望成功，实际 err: %v", err)
	}
	if request == nil || request.Placement == nil ||
		request.Placement.Zone == nil {
		t.Fatal("期望 request.Placement.Zone 被设置")
	}
	if *request.Placement.Zone != "ap-guangzhou-7" {
		t.Errorf("zone = %s, want ap-guangzhou-7",
			*request.Placement.Zone)
	}
	if info.SubnetId != "subnet-ok" {
		t.Errorf("info.SubnetId = %s, want subnet-ok", info.SubnetId)
	}
}

func TestResolveDoctorZone_config命中立即返回(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	// 设个 subnetMap。SubnetIds 是 "zone -> [subnetIds]" 的 JSON
	subnetMapJSON := `{"ap-guangzhou-3":["subnet-known"]}`
	cfg := &model.SiteConfig{
		SubnetIds: subnetMapJSON,
	}
	zone := resolveDoctorZone(
		context.Background(), "subnet-known", cfg)
	if zone != "ap-guangzhou-3" {
		t.Errorf("zone = %s, want ap-guangzhou-3", zone)
	}
}

func TestResolveDoctorZone_subnetId为空返回空(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()

	cfg := &model.SiteConfig{}
	zone := resolveDoctorZone(context.Background(), "", cfg)
	if zone != "" {
		t.Errorf("zone = %s, want \"\"", zone)
	}
}

func TestResolveDoctorZone_config未命中走兑底(t *testing.T) {
	cleanup := initDoctorTestDBOnly(t)
	defer cleanup()
	snap := saveDoctorFns()
	defer snap.restore()

	doctorDescribeSubnetZoneFn = func(_ context.Context, sid string) string {
		if sid == "subnet-unknown" {
			return "ap-shanghai-2"
		}
		return ""
	}

	cfg := &model.SiteConfig{}
	zone := resolveDoctorZone(
		context.Background(), "subnet-unknown", cfg)
	if zone != "ap-shanghai-2" {
		t.Errorf("zone = %s, want ap-shanghai-2", zone)
	}
}

// describeSubnetZoneViaVPC 函数本体。测试环境凭据未配置，
// newVpcClient 会报错 → 走第一个 fail path。
func TestDescribeSubnetZoneViaVPC_凭据未配置(t *testing.T) {
	got := describeSubnetZoneViaVPC(
		context.Background(), "subnet-x")
	if got != "" {
		t.Errorf("got = %s, want empty", got)
	}
}

func TestBuildDoctorCVMRequest_复用模板机型(t *testing.T) {
	snap := saveDoctorFns()
	defer snap.restore()
	doctorResolveZoneFn =
		func(context.Context, string, *model.SiteConfig) string {
			return "ap-guangzhou-3"
		}

	cfg := &model.SiteConfig{
		SecurityGroupId: "sg-1",
		CVMTemplate:     `{"InstanceType":"S5.LARGE16"}`,
	}
	target := &model.Instance{
		InstanceId: "ins-target", VpcId: "vpc-x",
		SubnetId: "subnet-x", SecurityGroupId: "sg-x",
	}
	req, _, err := buildDoctorCVMRequest(
		context.Background(), cfg, target,
		"pt", &STSCredentials{}, "img-1")
	if err != nil {
		t.Fatalf("期望成功，实际 err: %v", err)
	}
	if req.InstanceType == nil ||
		*req.InstanceType != "S5.LARGE16" {
		t.Errorf("InstanceType = %v, want S5.LARGE16",
			req.InstanceType)
	}
}

func TestBuildDoctorCVMRequest_模板为空报错(t *testing.T) {
	snap := saveDoctorFns()
	defer snap.restore()
	doctorResolveZoneFn =
		func(context.Context, string, *model.SiteConfig) string {
			return "ap-guangzhou-3"
		}

	cfg := &model.SiteConfig{SecurityGroupId: "sg-1"}
	target := &model.Instance{
		InstanceId: "ins-target", VpcId: "vpc-x",
		SubnetId: "subnet-x", SecurityGroupId: "sg-x",
	}
	req, _, err := buildDoctorCVMRequest(
		context.Background(), cfg, target,
		"pt", &STSCredentials{}, "img-1")
	if err == nil {
		t.Fatal("期望 err 非 nil")
	}
	if !strings.Contains(err.Error(), "InstanceType") {
		t.Errorf("错误信息应提及 InstanceType: %v", err)
	}
	if req != nil {
		t.Error("期望 req 为 nil")
	}
}

func TestBuildDoctorCVMRequest_模板解析失败报错(t *testing.T) {
	snap := saveDoctorFns()
	defer snap.restore()
	doctorResolveZoneFn =
		func(context.Context, string, *model.SiteConfig) string {
			return "ap-guangzhou-3"
		}

	cfg := &model.SiteConfig{
		SecurityGroupId: "sg-1",
		CVMTemplate:     "not-json",
	}
	target := &model.Instance{
		InstanceId: "ins-target", VpcId: "vpc-x",
		SubnetId: "subnet-x", SecurityGroupId: "sg-x",
	}
	_, _, err := buildDoctorCVMRequest(
		context.Background(), cfg, target,
		"pt", &STSCredentials{}, "img-1")
	if err == nil {
		t.Fatal("期望 err 非 nil")
	}
	if !strings.Contains(err.Error(), "模板配置错误") {
		t.Errorf("错误信息应提及 “模板配置错误”: %v", err)
	}
}

func TestParseCVMTemplateInstanceType_有效JSON(t *testing.T) {
	got, err := parseCVMTemplateInstanceType(
		`{"InstanceType":"S5.MEDIUM4","SystemDisk":{"DiskSize":50}}`)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "S5.MEDIUM4" {
		t.Errorf("got = %s, want S5.MEDIUM4", got)
	}
}

func TestParseCVMTemplateInstanceType_空模板(t *testing.T) {
	got, err := parseCVMTemplateInstanceType("")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "" {
		t.Errorf("got = %s, want empty", got)
	}
}

func TestParseCVMTemplateInstanceType_无效JSON返回err(t *testing.T) {
	got, err := parseCVMTemplateInstanceType("not-json")
	if err == nil {
		t.Fatal("期望 err 非 nil")
	}
	if got != "" {
		t.Errorf("got = %s, want empty", got)
	}
}

func TestParseCVMTemplateInstanceType_缺InstanceType字段(t *testing.T) {
	got, err := parseCVMTemplateInstanceType(
		`{"SystemDisk":{"DiskSize":50}}`)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "" {
		t.Errorf("got = %s, want empty", got)
	}
}
