package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/gorilla/sessions"
)

// initLocalCommandsTestDB 初始化 commands 测试用 DB（与 ack 测试共用 helper）
func initLocalCommandsTestDB(t *testing.T) {
	t.Helper()
	setupSkillInstancesDB(t)
	if Store == nil {
		Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	}
	if err := model.DB(context.Background()).AutoMigrate(
		&model.LocalInstanceInfo{},
		&model.LocalInstanceSkill{},
		&model.LocalInstanceRule{},
		&model.RuleDistributionRecord{},
		&model.RuleDistributionTask{},
		&model.EnterpriseRule{},
		&model.LocalAgentTask{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

// seedLocalCommandsFixture 建用户 + 本地 agent 实例 + 心跳行
// agent_type/local_agent_id 用 codebuddy/0123456789abcdef → instance_id=local-codebuddy-89abcd
func seedLocalCommandsFixture(t *testing.T) (*model.User, *model.Instance) {
	t.Helper()
	ctx := context.Background()
	user := &model.User{Username: "u-local-cmd", Password: "x", Role: "user"}
	if err := model.DB(ctx).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	inst := &model.Instance{
		Name: "local-codebuddy", InstanceId: "local-codebuddy-abcdef",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	if err := model.DB(ctx).Create(inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}
	now := time.Now()
	if err := model.DB(ctx).Create(&model.LocalInstanceInfo{
		InstanceID: inst.ID, LastReportAt: &now,
	}).Error; err != nil {
		t.Fatalf("create info: %v", err)
	}
	return user, inst
}

// commandsReq 构造带 session 登录的 POST /sync 请求
func commandsReq(t *testing.T, username string) *http.Request {
	t.Helper()
	url := "/api/v1/local-agent/sync"
	body := strings.NewReader(`{"agent_type":"codebuddy","local_agent_id":"0123456789abcdef","status":"running"}`)
	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// 用 session 登录（与 skillReqWithSession 等价的最小实现）
	sess, _ := Store.Get(req, "hatchery-session")
	sess.Values["username"] = username
	rr := httptest.NewRecorder()
	if err := sess.Save(req, rr); err != nil {
		t.Fatalf("session save: %v", err)
	}
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

// TestLocalAgentCommands_RegisteredSkill_ReturnsSMHURL
// 已注册 skill（cos_zip_key 非空）→ download_url 走 SMH
// 注：SMH 配置在测试环境多半缺，buildSMHDownloadURL 会失败 → 当前条目被跳过。
// 主要为反向断言：commands 至少不报错、返回 commands 数组。
func TestLocalAgentCommands_RegisteredSkill_NotCrashing(t *testing.T) {
	initLocalCommandsTestDB(t)
	user, inst := seedLocalCommandsFixture(t)

	skill := model.Skill{
		Slug: "registered-skill", Name: "registered", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
		COSZipKey:      "skill/cos/path.zip", // 非空 → 走 SMH 分支
	}
	if err := model.DB(context.Background()).Create(&skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}
	task := model.SkillDistributionTask{
		SkillID: skill.ID, Version: skill.Version, OperatorID: user.ID,
		Total: 1, Status: "running", Type: "distribute",
		Slug: skill.Slug,
	}
	model.DB(context.Background()).Create(&task)
	rec := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: skill.ID, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Version: skill.Version,
		Status: "pending", Type: "distribute",
	}
	model.DB(context.Background()).Create(&rec)

	rr := httptest.NewRecorder()
	HandleLocalAgentSync(rr, commandsReq(t, user.Username))
	if rr.Code != http.StatusOK {
		t.Fatalf("commands 期望 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	// 反向断言：响应至少能解码为 {commands, total}
	var resp struct {
		Commands []map[string]any `json:"commands"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
}

// TestLocalAgentCommands_UnregisteredSlug_ReturnsClawHubURL
// 关键测试：record.SkillID=0 + task.Slug="unregistered-slug" → LEFT JOIN skills 拿不到匹配
// → download_url 拼 ClawHub fallback URL（含 ?slug=）
func TestLocalAgentCommands_UnregisteredSlug_ReturnsClawHubURL(t *testing.T) {
	initLocalCommandsTestDB(t)
	user, inst := seedLocalCommandsFixture(t)

	// 不创建 skills 表行；直接构造 record（模拟 handleAddSkillLocal 兜底分支创建出来的）
	task := model.SkillDistributionTask{
		SkillID: 0, Version: "", OperatorID: user.ID,
		Total: 1, Status: "running", Type: "distribute",
		Slug: "unregistered-slug",
	}
	model.DB(context.Background()).Create(&task)
	rec := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: 0, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Version: "",
		Status: "pending", Type: "distribute",
	}
	model.DB(context.Background()).Create(&rec)

	rr := httptest.NewRecorder()
	HandleLocalAgentSync(rr, commandsReq(t, user.Username))
	if rr.Code != http.StatusOK {
		t.Fatalf("commands 期望 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Commands []map[string]any `json:"commands"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if len(resp.Commands) != 1 {
		t.Fatalf("应返回 1 条命令，实际 len=%d body=%s",
			len(resp.Commands), rr.Body.String())
	}
	cmd := resp.Commands[0]
	if cmd["skill_slug"] != "unregistered-slug" {
		t.Errorf("slug 应=unregistered-slug，实际=%v", cmd["skill_slug"])
	}
	if cmd["scope"] != model.LocalSkillScopeUser {
		t.Errorf("未关联二期 scope 的旧任务应兼容为 user，实际=%v", cmd["scope"])
	}
	dl, _ := cmd["download_url"].(string)
	if dl == "" {
		t.Fatalf("download_url 应非空（ClawHub 兜底）")
	}
	if !strings.Contains(dl, "slug=unregistered-slug") {
		t.Errorf("download_url 应含 slug 参数，实际=%s", dl)
	}
	if !strings.HasPrefix(dl, "https://") {
		t.Errorf("ClawHub URL 应是 https，实际=%s", dl)
	}
}

// TestLocalAgentCommands_EmptySlug_Skipped
// 极端边界：task.Slug=” 且 SkillID 找不到 skill → 既无 SMH 也无 ClawHub URL
// 必须跳过这一条，避免 reporter 拿到空 download_url 误装
func TestLocalAgentCommands_EmptySlug_Skipped(t *testing.T) {
	initLocalCommandsTestDB(t)
	user, inst := seedLocalCommandsFixture(t)

	task := model.SkillDistributionTask{
		SkillID: 0, OperatorID: user.ID, Total: 1, Status: "running", Type: "distribute",
		Slug: "",
	}
	model.DB(context.Background()).Create(&task)
	rec := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: 0, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId,
		Status:      "pending", Type: "distribute",
	}
	model.DB(context.Background()).Create(&rec)

	rr := httptest.NewRecorder()
	HandleLocalAgentSync(rr, commandsReq(t, user.Username))
	if rr.Code != http.StatusOK {
		t.Fatalf("commands 期望 200，实际=%d", rr.Code)
	}
	var resp struct {
		Commands []map[string]any `json:"commands"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Commands) != 0 {
		t.Errorf("空 slug 的 distribute record 应被跳过，total 应=0，实际=%d body=%s",
			len(resp.Commands), rr.Body.String())
	}
}

// TestHandleAddSkillLocal_UnregisteredSlug_CreatesRecord
// public source + skills 表无匹配 → 不再返回 SkillNotExist，
// 而是 fallthrough 创建带 Slug 的 record（SkillID=0）
func TestHandleAddSkillLocal_UnregisteredSlug_CreatesRecord(t *testing.T) {
	cleanup := initSkillHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u-local-fb", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "local-codebuddy-fb", InstanceId: "local-codebuddy-fb01",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(inst)

	if err := model.DB(context.Background()).AutoMigrate(
		&model.LocalInstanceInfo{},
		&model.SkillDistributionTask{},
		&model.SkillDistributionRecord{},
		&model.LocalAgentTask{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	now := time.Now()
	model.DB(context.Background()).Create(&model.LocalInstanceInfo{
		InstanceID: inst.ID, LastReportAt: &now,
	})

	// 不创建任何 skills 行 → 走 fallthrough 兜底分支
	form := fmt.Sprintf("skill_name=%s&source=public", "fb-some-public-skill")
	req := skillReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/skill?id=%d", inst.ID), user.Username, form)
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, testCVMFetcher)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200（兜底创建 pending record），实际=%d body=%s", rr.Code, rr.Body.String())
	}
	// 校验：找到 instance 上的 pending record，并通过 record.TaskID 关联到 task；
	// 关键断言：record.SkillID=0、task.Slug=skillName。
	var rec model.SkillDistributionRecord
	if err := model.DB(context.Background()).
		Where("instance_id = ? AND skill_id = 0 AND status = ? AND type = ?",
			inst.ID, "pending", "distribute").
		First(&rec).Error; err != nil {
		t.Fatalf("应创建 fallback record（SkillID=0），未找到: %v", err)
	}
	if rec.Status != "pending" || rec.Type != "distribute" {
		t.Errorf("status/type 不对：%s/%s", rec.Status, rec.Type)
	}
	var task model.SkillDistributionTask
	if err := model.DB(context.Background()).First(&task, rec.TaskID).Error; err != nil {
		t.Fatalf("rec.TaskID 关联的 task 应存在: %v", err)
	}
	if task.SkillID != 0 {
		t.Errorf("task.SkillID 应=0，实际=%d", task.SkillID)
	}
	if task.Slug != "fb-some-public-skill" {
		t.Errorf("task.Slug 应=fb-some-public-skill，实际=%s", task.Slug)
	}
}

func TestLocalAgentCommands_UnregisteredSlug_LastFailed_ReturnsSkillHubURL(t *testing.T) {
	initLocalCommandsTestDB(t)
	user, inst := seedLocalCommandsFixture(t)

	// 先造一条同 slug 的历史 failed record（虚拟上次下发失败）
	oldTask := model.SkillDistributionTask{
		SkillID: 0, Version: "", OperatorID: user.ID,
		Total: 1, Status: "completed", Type: "distribute",
		Slug: "retry-fallback-slug",
	}
	model.DB(context.Background()).Create(&oldTask)
	oldRec := model.SkillDistributionRecord{
		TaskID: oldTask.ID, SkillID: 0, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Version: "",
		Status: "failed", Type: "distribute",
	}
	model.DB(context.Background()).Create(&oldRec)

	// 再造本次重试 pending record
	task := model.SkillDistributionTask{
		SkillID: 0, Version: "", OperatorID: user.ID,
		Total: 1, Status: "running", Type: "distribute",
		Slug: "retry-fallback-slug",
	}
	model.DB(context.Background()).Create(&task)
	rec := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: 0, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Version: "",
		Status: "pending", Type: "distribute",
	}
	model.DB(context.Background()).Create(&rec)

	rr := httptest.NewRecorder()
	HandleLocalAgentSync(rr, commandsReq(t, user.Username))
	if rr.Code != http.StatusOK {
		t.Fatalf("sync 期望 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Commands []map[string]any `json:"commands"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if len(resp.Commands) != 1 {
		t.Fatalf("应返回 1 条 pending，实际=%d body=%s", len(resp.Commands), rr.Body.String())
	}
	dl, _ := resp.Commands[0]["download_url"].(string)
	if !strings.Contains(dl, "api.skillhub.cn") {
		t.Errorf("上次 failed 后重试应走 SkillHub（api.skillhub.cn），实际=%s", dl)
	}
	if !strings.Contains(dl, "slug=retry-fallback-slug") {
		t.Errorf("download_url 应含 slug 参数，实际=%s", dl)
	}
}

// TestLocalAgentCommands_UnregisteredSlug_LastSuccess_ReturnsClawHubURL
// 案例：同 slug 上次是 success 终态（不是失败）→ 仍走 ClawHub。
func TestLocalAgentCommands_UnregisteredSlug_LastSuccess_ReturnsClawHubURL(t *testing.T) {
	initLocalCommandsTestDB(t)
	user, inst := seedLocalCommandsFixture(t)

	oldTask := model.SkillDistributionTask{
		SkillID: 0, Version: "", OperatorID: user.ID,
		Total: 1, Status: "completed", Type: "distribute",
		Slug: "prev-success-slug",
	}
	model.DB(context.Background()).Create(&oldTask)
	oldRec := model.SkillDistributionRecord{
		TaskID: oldTask.ID, SkillID: 0, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Version: "",
		Status: "success", Type: "distribute",
	}
	model.DB(context.Background()).Create(&oldRec)

	task := model.SkillDistributionTask{
		SkillID: 0, Version: "", OperatorID: user.ID,
		Total: 1, Status: "running", Type: "distribute",
		Slug: "prev-success-slug",
	}
	model.DB(context.Background()).Create(&task)
	rec := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: 0, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Version: "",
		Status: "pending", Type: "distribute",
	}
	model.DB(context.Background()).Create(&rec)

	rr := httptest.NewRecorder()
	HandleLocalAgentSync(rr, commandsReq(t, user.Username))
	if rr.Code != http.StatusOK {
		t.Fatalf("sync 期望 200，实际=%d", rr.Code)
	}
	var resp struct {
		Commands []map[string]any `json:"commands"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Commands) != 1 {
		t.Fatalf("应返回 1 条 pending，实际=%d", len(resp.Commands))
	}
	dl, _ := resp.Commands[0]["download_url"].(string)
	if strings.Contains(dl, "api.skillhub.cn") {
		t.Errorf("上次 success 的 slug 重装不应切 SkillHub，实际=%s", dl)
	}
	if !strings.Contains(dl, "convex.site") {
		t.Errorf("应仍是 ClawHub URL，实际=%s", dl)
	}
}

// TestLocalAgentCommands_UnregisteredSlug_LastFailedThenSuccess_ReturnsClawHubURL
// 边界：同 slug 上次是 failed 但再上次是 success → “最新”是 success → 走 ClawHub。
func TestLocalAgentCommands_UnregisteredSlug_LastFailedThenSuccess_ReturnsClawHubURL(t *testing.T) {
	initLocalCommandsTestDB(t)
	user, inst := seedLocalCommandsFixture(t)

	// 最早：failed
	t1 := model.SkillDistributionTask{
		SkillID: 0, OperatorID: user.ID, Total: 1, Status: "completed",
		Type: "distribute", Slug: "flap-slug",
	}
	model.DB(context.Background()).Create(&t1)
	model.DB(context.Background()).Create(&model.SkillDistributionRecord{
		TaskID: t1.ID, SkillID: 0, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Status: "failed", Type: "distribute",
	})

	// 再后：success
	t2 := model.SkillDistributionTask{
		SkillID: 0, OperatorID: user.ID, Total: 1, Status: "completed",
		Type: "distribute", Slug: "flap-slug",
	}
	model.DB(context.Background()).Create(&t2)
	model.DB(context.Background()).Create(&model.SkillDistributionRecord{
		TaskID: t2.ID, SkillID: 0, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Status: "success", Type: "distribute",
	})

	// 本次重试：pending
	t3 := model.SkillDistributionTask{
		SkillID: 0, OperatorID: user.ID, Total: 1, Status: "running",
		Type: "distribute", Slug: "flap-slug",
	}
	model.DB(context.Background()).Create(&t3)
	model.DB(context.Background()).Create(&model.SkillDistributionRecord{
		TaskID: t3.ID, SkillID: 0, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Status: "pending", Type: "distribute",
	})

	rr := httptest.NewRecorder()
	HandleLocalAgentSync(rr, commandsReq(t, user.Username))
	if rr.Code != http.StatusOK {
		t.Fatalf("sync 期望 200，实际=%d", rr.Code)
	}
	var resp struct {
		Commands []map[string]any `json:"commands"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Commands) != 1 {
		t.Fatalf("应返回 1 条 pending，实际=%d", len(resp.Commands))
	}
	dl, _ := resp.Commands[0]["download_url"].(string)
	if strings.Contains(dl, "api.skillhub.cn") {
		t.Errorf("最新终态是 success，不应切 SkillHub，实际=%s", dl)
	}
}
