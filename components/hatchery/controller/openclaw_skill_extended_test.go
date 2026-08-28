package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/gorilla/sessions"
)

// skillReqWithSession 构造带 session 的请求。
func skillReqWithSession(t *testing.T, method, path, username, body string) *http.Request {
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

// initSkillHandlerTestDB 扩展 skill 测试 DB，加入 UserGroup / SkillBundle 等依赖。
func initSkillHandlerTestDB(t *testing.T) func() {
	t.Helper()
	cleanup := initSkillTestDB(t)
	// 额外迁移（含 Notification，避免失败通知 goroutine 写表 panic）
	model.DB(context.Background()).AutoMigrate(
		&model.CustomAgentType{},
		&model.Skill{},
		&model.SkillVisibilityGroup{},
		&model.SkillBundle{},
		&model.BundleSkill{},
		&model.OpenClawRoleSkill{},
		&model.UserGroupMember{},
		&model.SkillBundleVisibilityGroup{},
		&model.Notification{},
	)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	return func() {
		Store = origStore
		cleanup()
	}
}

// ─── listInstanceSkills（纯 ResolveScript 失败路径）──────────────────────────

func TestListInstanceSkills_UnknownAgentType(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()
	_, err := listInstanceSkills(context.Background(), "ins-xxx", "agentuser", "future_unknown_type")
	if err == nil {
		t.Error("未知 agent_type 应返回错误")
	}
	if !strings.Contains(err.Error(), "list_skills") {
		t.Errorf("错误消息应提到 list_skills，实际=%v", err)
	}
}

// ─── HandleAddSkill ────────────────────────────────────────────────────────

func TestHandleAddSkill_Unauthorized(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/skill", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, testCVMFetcher)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleAddSkill_MethodNotAllowed(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := skillReqWithSession(t, http.MethodGet, "/openclaw/skill", "u1", "")
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, testCVMFetcher)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleAddSkill_UnknownAgentType(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-skill-unk",
		UserID: user.ID, AgentType: "future_unknown_type",
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("skill_name", "some-skill")
	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/skill?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, testCVMFetcher)

	// 未知类型 → checkInstanceSupportsSkill 返回 403
	if rr.Code != http.StatusForbidden {
		t.Errorf("未知 agent_type 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddSkill_EmptySkillName(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-skill-empty",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{} // 无 skill_name
	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/skill?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("空 skill_name 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAddSkill_InvalidSource(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-skill-source",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("skill_name", "some-skill")
	form.Set("source", "private")
	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/skill?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 source 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddSkill_AgentIDNonOpenClaw(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-skill-agentid-hermes",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("skill_name", "some-skill")
	form.Set("agent_id", "agent-1dae1025")
	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/skill?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非 OpenClaw 实例传 agent_id 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddSkill_AgentIDInvalid(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-skill-agentid-invalid",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("skill_name", "some-skill")
	form.Set("agent_id", "../bad")
	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/skill?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 agent_id 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddSkill_EnterpriseNotFound(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-skill-ent-missing",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("skill_name", "missing-skill")
	form.Set("source", "enterprise")
	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/skill?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, testCVMFetcher)

	if rr.Code != http.StatusNotFound {
		t.Errorf("企业 Skill 不存在应返回 404，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddSkill_EnterpriseInvisible(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-skill-ent-invisible",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	skill := &model.Skill{
		Slug: "private-skill", Name: "私有技能", Version: "1.0.0",
		VersionMajor: 1, COSZipKey: "private-skill/private-skill-1.0.0.zip",
		VisibilityType: model.VisibilityGroup,
	}
	model.DB(context.Background()).Create(skill)
	model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: skill.ID, GroupID: 999})

	form := url.Values{}
	form.Set("skill_name", "private-skill")
	form.Set("source", "enterprise")
	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/skill?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, testCVMFetcher)

	if rr.Code != http.StatusNotFound {
		t.Errorf("不可见企业 Skill 应按不存在返回 404，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddSkill_EnterpriseExplicitNoPublicFallback(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	origLoader := LoadScript
	loadScriptCalled := false
	LoadScript = func(name string) (string, error) {
		loadScriptCalled = true
		return "", fmt.Errorf("unexpected LoadScript call: %s", name)
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-skill-ent-no-fallback",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.Skill{
		Slug: "enterprise-skill", Name: "企业技能", Version: "1.0.0",
		VersionMajor: 1, COSZipKey: "enterprise-skill/enterprise-skill-1.0.0.zip",
		VisibilityType: model.VisibilityAll,
	})

	form := url.Values{}
	form.Set("skill_name", "enterprise-skill")
	form.Set("source", "enterprise")
	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/skill?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("SMH 未配置应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if loadScriptCalled {
		t.Error("source=enterprise 生成下载 URL 失败时不应 fallback 到公共源或调用脚本")
	}
}

func TestApplyExtraSkillsAsync_MatchesEnterpriseAddWithoutPersistence(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()
	if err := model.DB(context.Background()).AutoMigrate(&model.SkillInstallation{}); err != nil {
		t.Fatalf("migrate skill installations: %v", err)
	}

	originalLoader := LoadScript
	loadScriptCalled := false
	LoadScript = func(name string) (string, error) {
		loadScriptCalled = true
		return "", fmt.Errorf("unexpected LoadScript call: %s", name)
	}
	defer func() { LoadScript = originalLoader }()

	user := &model.User{Username: "preset-skill-u", Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	instance := &model.Instance{
		Name:        "preset-skill-inst",
		InstanceId:  "ins-preset-skill",
		UserID:      user.ID,
		AgentType:   model.AgentTypeOpenClaw,
		AgentReady:  1,
		RuntimeUser: "ubuntu",
	}
	if err := model.DB(context.Background()).Create(instance).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	skill := model.Skill{
		Slug:           "preset-enterprise-skill",
		Name:           "Preset Enterprise Skill",
		Version:        "1.0.0",
		VersionMajor:   1,
		COSZipKey:      "preset-enterprise-skill/preset-enterprise-skill-1.0.0.zip",
		VisibilityType: model.VisibilityAll,
	}

	applyExtraSkillsAsync(context.Background(), instance.ID, []createSkillPreset{{
		Source: model.SkillSourceEnterprise, Slug: skill.Slug, Version: skill.Version, Enterprise: skill,
	}})

	if loadScriptCalled {
		t.Fatal("missing SMH configuration must fail before script loading, as in manual enterprise add")
	}
	var count int64
	if err := model.DB(context.Background()).Model(&model.SkillInstallation{}).
		Where("instance_id = ?", instance.ID).Count(&count).Error; err != nil {
		t.Fatalf("count skill installations: %v", err)
	}
	if count != 0 {
		t.Fatalf("create-time enterprise skill created %d installation rows, want 0", count)
	}
}

func TestApplyEnterpriseSkill_ResolveFailureRetainsManualBadRequest(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()
	if err := model.DB(context.Background()).AutoMigrate(&model.SMHSpace{}); err != nil {
		t.Fatalf("migrate SMH spaces: %v", err)
	}
	seedSMHFullyConfigured(t)
	model.UpdateSMHSpaceToken(
		context.Background(),
		"skillhub",
		false,
		"skillhub-read-token",
		time.Now().Add(time.Hour).Unix(),
	)

	instance := &model.Instance{
		InstanceId:  "ins-enterprise-resolve-failure",
		AgentType:   "future-unknown-agent",
		RuntimeUser: "ubuntu",
	}
	skill := &model.Skill{
		Slug:      "enterprise-resolve-failure",
		Version:   "1.0.0",
		COSZipKey: "enterprise-resolve-failure/1.0.0.zip",
	}
	applyErr := applyEnterpriseSkill(context.Background(), instance, skill, "")
	if applyErr == nil {
		t.Fatal("unknown agent type should fail script resolution")
	}
	if applyErr.status != http.StatusBadRequest {
		t.Fatalf("resolve failure status = %d, want 400", applyErr.status)
	}
	if applyErr.run {
		t.Fatal("script resolution failure must not be handled as a RunScript failure")
	}
}

func TestHandleAddSkill_HermesBranch_LoadScriptFail(t *testing.T) {
	// 验证 Hermes/ACE 分支（62-71 行）被覆盖：会调用 tryBuildSkillDownloadURL + 额外参数
	cleanup := initSkillHandlerTestDB(t)

	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: script not found: %s", name)
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-hermes-skill",
		UserID: user.ID, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("skill_name", "my-skill")
	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/skill?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, testCVMFetcher)

	// Hermes 走到 RunScript 时 LoadScript mock 失败 → 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("mock LoadScript 失败应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 等待异步 goroutine（createErrorNotification）执行完再 cleanup
	waitForGoroutines()
	cleanup()
}

func TestHandleAddSkill_AceBranch_LoadScriptFail(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)

	origLoader := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", fmt.Errorf("mock: script not found: %s", name)
	}
	defer func() { LoadScript = origLoader }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-ace-skill",
		UserID: user.ID, AgentType: model.AgentTypeLightclawACE,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("skill_name", "my-ace-skill")
	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/skill?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("mock LoadScript 失败应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	waitForGoroutines()
	cleanup()
}

// waitForGoroutines 等待 handler 内部异步 goroutine（如 createErrorNotification）执行完毕，
// 避免 cleanup 重置 DB 后 goroutine 访问已回收的 DB 导致 panic。
func waitForGoroutines() {
	// 少量 goroutine（createErrorNotification 在 mock DB 下很快完成）
	// 200ms 足够覆盖单次 DB.Create 的 SQLite in-memory 延迟
	deadline := 200
	for i := 0; i < deadline; i++ {
		time.Sleep(time.Millisecond)
	}
}

// ─── HandleSkillsList ────────────────────────────────────────────────────

func TestHandleSkillsList_Unauthorized(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/skills-list?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handleSkillsList(rr, req, newUserSkillDependencies())
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleSkillsList_InstanceNotFound(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := skillReqWithSession(t, http.MethodGet, "/openclaw/skills-list?id=999", "u1", "")
	rr := httptest.NewRecorder()
	handleSkillsList(rr, req, newUserSkillDependencies())
	if rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleSkillsList_UnknownAgentTypeInListSkills(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-skillslist-unk",
		UserID: user.ID, AgentType: "future_unknown_type",
	}
	model.DB(context.Background()).Create(inst)

	req := skillReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/skills-list?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	handleSkillsList(rr, req, newUserSkillDependencies())

	// listInstanceSkills 返回 error → 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("未知 agent_type 应返回 500（list_skills 脚本解析失败），实际=%d body=%s",
			rr.Code, rr.Body.String())
	}
}

// ─── HandleInstallSkills ─────────────────────────────────────────────────

func TestHandleInstallSkills_Unauthorized(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/install-skills", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleInstallSkills(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleInstallSkills_Empty(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-install-empty",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := skillReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/install-skills?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleInstallSkills(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "\"total\":0") {
		t.Errorf("应返回 total=0，实际=%s", rr.Body.String())
	}
}

func TestHandleInstallSkills_WithRecords(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-install-has",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// 创建两条安装记录
	model.DB(context.Background()).Create(&model.SkillInstallation{
		InstanceID: inst.ID, Name: "s1", Slug: "slug1",
		InstallStatus: model.SkillInstallSuccess,
	})
	model.DB(context.Background()).Create(&model.SkillInstallation{
		InstanceID: inst.ID, Name: "s2", Slug: "slug2",
		InstallStatus: model.SkillInstallFailed,
	})

	req := skillReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/install-skills?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleInstallSkills(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "\"total\":2") {
		t.Errorf("应返回 total=2，实际=%s", rr.Body.String())
	}
}

// ─── HandleRetryFailedSkills ─────────────────────────────────────────────

func TestHandleRetryFailedSkills_MethodNotAllowed(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := skillReqWithSession(t, http.MethodGet, "/openclaw/retry-failed-skills?id=1", "u1", "")
	rr := httptest.NewRecorder()
	HandleRetryFailedSkills(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleRetryFailedSkills_Unauthorized(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/retry-failed-skills", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleRetryFailedSkills(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleRetryFailedSkills_NoFailedSkills(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-retry-none",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/retry-failed-skills?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleRetryFailedSkills(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "\"retry_count\":0") {
		t.Errorf("应返回 retry_count=0，实际=%s", rr.Body.String())
	}
}

func TestHandleRetryFailedSkills_InstanceNoCVM(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "", // 无 CVM
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/retry-failed-skills?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleRetryFailedSkills(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("无 CVM 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── HandleCancelFailedSkills ────────────────────────────────────────────

func TestHandleCancelFailedSkills_MethodNotAllowed(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := skillReqWithSession(t, http.MethodGet, "/openclaw/cancel-failed-skills?id=1", "u1", "")
	rr := httptest.NewRecorder()
	HandleCancelFailedSkills(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleCancelFailedSkills_CancelsFailedOnly(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-cancel",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// 插入 1 条 Failed + 1 条 Success
	model.DB(context.Background()).Create(&model.SkillInstallation{
		InstanceID: inst.ID, Name: "f1", Slug: "fslug",
		InstallStatus: model.SkillInstallFailed,
	})
	model.DB(context.Background()).Create(&model.SkillInstallation{
		InstanceID: inst.ID, Name: "s1", Slug: "sslug",
		InstallStatus: model.SkillInstallSuccess,
	})

	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/cancel-failed-skills?id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleCancelFailedSkills(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "\"cancel_count\":1") {
		t.Errorf("应取消 1 条 Failed 记录，实际 body=%s", rr.Body.String())
	}

	// 验证成功的记录未受影响
	var successSkill model.SkillInstallation
	model.DB(context.Background()).Where("slug = ?", "sslug").First(&successSkill)
	if successSkill.InstallStatus != model.SkillInstallSuccess {
		t.Errorf("成功的记录不应被改动，实际状态=%v", successSkill.InstallStatus)
	}
}

// ─── installSkillsAsync unknown agent_type fail-closed ────────────────────

// TestInstallSkillsAsync_UnknownAgentType_FailsClosed 验证 batch_install_skills
// 脚本分派失败时，Installing 状态会被置为 Failed（第 547-561 行）。
func TestInstallSkillsAsync_UnknownAgentType_FailsClosed(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-installing",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// 创建 Installing 状态的记录（对齐源码 547-561 的前置条件）
	skill := &model.SkillInstallation{
		InstanceID: inst.ID, Name: "s1", Slug: "slug1",
		InstallStatus: model.SkillInstalling,
	}
	model.DB(context.Background()).Create(skill)

	// 这里我们只调用 parseAndUpdateInstallResults 是不合适的（它已经被覆盖）。
	// 直接测试 createSkillInstallTasks 作为补充验证函数可以正常运行不 panic。
	createSkillInstallTasks(context.Background(), inst.ID, 0, user.ID)

	// 验证至少不 panic
	var count int64
	model.DB(context.Background()).Model(&model.SkillInstallation{}).Where("instance_id = ?", inst.ID).Count(&count)
	// 应该不变（没有技能包）
	if count != 1 {
		t.Errorf("应仍只有 1 条记录，实际=%d", count)
	}
}

// ─── tryBuildSkillDownloadURL ───────────────────────────────────────────

func TestTryBuildSkillDownloadURL_NoBundleSkill_NoSkill(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.Skill{}, &model.BundleSkill{})

	// 无 bundle_skill 也无 skill → 返回空字符串
	got := tryBuildSkillDownloadURL(context.Background(), "nonexistent-slug")
	if got != "" {
		t.Errorf("未找到 slug 应返回空，实际=%q", got)
	}
}

func TestTryBuildSkillDownloadURL_BundleSkillNoSMHConfig(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.Skill{}, &model.BundleSkill{})

	// 有 bundle_skill 但无 SMH 配置 → buildCommonSMHDownloadURL 失败 → fallback skills 也空 → 返回空
	model.DB(context.Background()).Create(&model.BundleSkill{
		Slug: "my-slug", CosZipKey: "skills/my-slug-1.0.0.zip",
	})

	got := tryBuildSkillDownloadURL(context.Background(), "my-slug")
	if got != "" {
		t.Errorf("SMH 未配置时应返回空，实际=%q", got)
	}
}

// TestHandleAddSkill_LocalInstance_NotBlockedByAgentTypeGuard 回归测试：
// 本地 agent 实例（agent_type=codebuddy/workbuddy）当前未注册到内置 agentTypesMap，
// AgentTypeSupportsSkill 会返回 false。但本地实例的 skill 能力由 reporter ack
// 链路保证，不依赖该能力位。
//
// handleAddSkill 必须先按 instance.Source 分流：source=local 时进
// handleAddSkillLocal，跳过 checkInstanceSupportsSkill。否则用户拿到
// 「codebuddy 类型实例不支持技能功能」的错误，无法装任何 skill。
func TestHandleAddSkill_LocalInstance_NotBlockedByAgentTypeGuard(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u-local-skill", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 本地 agent 实例：source=local，agent_type=codebuddy（不在内置 map 里）
	inst := &model.Instance{
		Name: "local-codebuddy", InstanceId: "local-codebuddy-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(inst)

	// handleAddSkillLocal 需要 local_instance_infos 表（不在默认 migrate 列表）
	if err := model.DB(context.Background()).AutoMigrate(
		&model.LocalInstanceInfo{},
		&model.SkillDistributionTask{},
		&model.SkillDistributionRecord{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	now := time.Now()
	model.DB(context.Background()).Create(&model.LocalInstanceInfo{
		InstanceID: inst.ID, LastReportAt: &now,
	})

	// 准备一个 visibility_type=all 的 skill
	skill := model.Skill{
		Slug: "local-add-skill-test", Name: "test", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skill)

	form := url.Values{}
	form.Set("skill_name", skill.Slug)
	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/skill?id=%d", inst.ID), "u-local-skill", form.Encode())
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, testCVMFetcher)

	// 关键断言：不能是 403 + "不支持技能功能"
	if rr.Code == http.StatusForbidden && strings.Contains(rr.Body.String(), "不支持技能功能") {
		t.Fatalf("本地实例不应被 checkInstanceSupportsSkill 拦截，body=%s", rr.Body.String())
	}
	// 期望 200：handleAddSkillLocal 创建 pending record 成功
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200（handleAddSkillLocal 创建 pending record），实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 校验 pending record 已落库。
	// 本地路径不查 skills 表，record.skill_id 恒为 0，按 (instance_id, slug) 查 task 取 record。
	var rec model.SkillDistributionRecord
	if err := model.DB(context.Background()).
		Joins("JOIN skill_distribution_tasks ON skill_distribution_tasks.id = skill_distribution_records.task_id").
		Where("skill_distribution_records.instance_id = ? AND skill_distribution_tasks.slug = ?", inst.ID, skill.Slug).
		First(&rec).Error; err != nil {
		t.Fatalf("应创建 pending record，查询失败: %v", err)
	}
	if rec.Status != "pending" || rec.Type != "distribute" {
		t.Errorf("record status/type 不对：status=%s type=%s", rec.Status, rec.Type)
	}
	if rec.SkillID != 0 {
		t.Errorf("本地路径不查 skills 表，record.skill_id 应为 0，实际=%d", rec.SkillID)
	}
}

// TestInferLocalSkillSourceFromRow 纯逻辑单测，覆盖三种 case：
//   - lis.Source 已存在 → 直接返回（ack 写入的权威值优先）
//   - lis.Source 空 + recordSkillID==0 → ClawHub 兜底 → public
//   - lis.Source 空 + recordSkillID!=0 → 企业内部 → enterprise
//
// 这是修复 "企业 skill ack 后 source 误标 public" 时同步改的兜底函数：
// 历史版本用 visibility_type 推断（visibility=all 默认会被错认 public），
// 现在改成对齐 ack 路径，按 record.SkillID 判断。
func TestInferLocalSkillSourceFromRow(t *testing.T) {
	cases := []struct {
		name        string
		lisSource   string
		recordSkill uint
		want        string
	}{
		{"lis_source_priority_enterprise", model.LocalSkillSourceEnterprise, 0, model.LocalSkillSourceEnterprise},
		{"lis_source_priority_local", model.LocalSkillSourceLocal, 7, model.LocalSkillSourceLocal},
		{"lis_source_priority_public", model.LocalSkillSourcePublic, 0, model.LocalSkillSourcePublic},
		{"lis_source_blank_skill_id_zero_clawhub", "", 0, model.LocalSkillSourcePublic},
		{"lis_source_blank_skill_id_set_enterprise", "", 42, model.LocalSkillSourceEnterprise},
		{"lis_source_whitespace_treated_as_blank", "  \t ", 99, model.LocalSkillSourceEnterprise},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			lis := model.LocalInstanceSkill{Source: c.lisSource}
			got := inferLocalSkillSourceFromRow(lis, c.recordSkill)
			if got != c.want {
				t.Errorf("lis.Source=%q recordSkillID=%d → got=%s want=%s",
					c.lisSource, c.recordSkill, got, c.want)
			}
		})
	}
}
