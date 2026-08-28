package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/gorilla/sessions"
)

// ensureSessionStore 初始化 Store（setupSkillInstancesDB 不 init Store）。
func ensureSessionStore() {
	if Store == nil {
		Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	}
}

// jsonAckReq 构造 reporter ack 接口请求（带登录态 + JSON body）。
//
// recordID 会被注入到 body.id（ack 接口从 path 改成从 body 读 id）。
// body 允许传 "" 表示仅 id；其他情况走 JSON merge。
func jsonAckReq(t *testing.T, recordID uint, username, body string) *http.Request {
	t.Helper()
	ensureSessionStore()
	var payload map[string]any
	if strings.TrimSpace(body) == "" {
		payload = map[string]any{}
	} else {
		if err := json.Unmarshal([]byte(body), &payload); err != nil {
			t.Fatalf("jsonAckReq decode body %q: %v", body, err)
		}
	}
	payload["id"] = recordID
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("jsonAckReq encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/local-agent/commands/ack", strings.NewReader(string(encoded)))
	req.Header.Set("Content-Type", "application/json")
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

// seedLocalAckCase 创建 user/instance/skill/task/record 一套，方便 ack 测试聚焦在
// counter 行为上。返回 record.ID 与 task.ID。
func seedLocalAckCase(t *testing.T, username, slug string, taskType string) (uint, uint) {
	t.Helper()
	ctx := context.Background()
	// distribute+success 路径会 upsert local_instance_skills，预先 migrate。
	// setupSkillInstancesDB 默认不包含该表，为本文件专补。
	if err := model.DB(ctx).AutoMigrate(&model.LocalInstanceSkill{}); err != nil {
		t.Fatalf("migrate LocalInstanceSkill: %v", err)
	}
	user := model.User{Username: username, Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	inst := model.Instance{
		Name: "local-" + username, InstanceId: "local-" + username + "-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	if err := model.DB(ctx).Create(&inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}
	skill := model.Skill{
		Slug: slug, Name: slug, Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: model.VisibilityAll,
	}
	if err := model.DB(ctx).Create(&skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}
	task := model.SkillDistributionTask{
		SkillID: skill.ID, Version: skill.Version,
		Total: 1, Status: "running", Type: taskType,
	}
	if err := model.DB(ctx).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	rec := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: skill.ID,
		InstanceID: inst.ID, InstanceCID: inst.InstanceId,
		Version: skill.Version, Status: "pending", Type: taskType,
	}
	if err := model.DB(ctx).Create(&rec).Error; err != nil {
		t.Fatalf("create rec: %v", err)
	}
	return rec.ID, task.ID
}

// TestHandleLocalAgentAck_Success_IncrementsTaskSuccess reporter ack=success 时
// 应原子递增 task.success；本地实例的 record 不会被 async finalize 触碰，所以
// task.success 必须由 ack 这条路径回写，否则看板「已完成 X/N」会一直停在 0。
func TestHandleLocalAgentAck_Success_IncrementsTaskSuccess(t *testing.T) {
	setupSkillInstancesDB(t)
	recID, taskID := seedLocalAckCase(t, "ack-ok", "ack-ok-skill", model.TaskTypeDistribute)

	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, jsonAckReq(t, recID, "ack-ok", `{"status":"success"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("ack 应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var task model.SkillDistributionTask
	if err := model.DB(context.Background()).First(&task, taskID).Error; err != nil {
		t.Fatalf("查 task: %v", err)
	}
	if task.Success != 1 {
		t.Errorf("task.Success 应=1，实际=%d", task.Success)
	}
	if task.Failed != 0 {
		t.Errorf("task.Failed 应=0，实际=%d", task.Failed)
	}
	// total=1 且已 success=1 → finalize 应标 completed
	if task.Status != "completed" {
		t.Errorf("task.Status 应=completed（total=1, success=1），实际=%s", task.Status)
	}
}

// TestHandleLocalAgentAck_Failed_IncrementsTaskFailed reporter ack=failed 时
// 应原子递增 task.failed。
func TestHandleLocalAgentAck_Failed_IncrementsTaskFailed(t *testing.T) {
	setupSkillInstancesDB(t)
	recID, taskID := seedLocalAckCase(t, "ack-fail", "ack-fail-skill", model.TaskTypeDistribute)

	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, jsonAckReq(t, recID, "ack-fail", `{"status":"failed","error":"安装脚本退出 1"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("ack 应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var task model.SkillDistributionTask
	if err := model.DB(context.Background()).First(&task, taskID).Error; err != nil {
		t.Fatalf("查 task: %v", err)
	}
	if task.Failed != 1 {
		t.Errorf("task.Failed 应=1，实际=%d", task.Failed)
	}
	if task.Success != 0 {
		t.Errorf("task.Success 应=0，实际=%d", task.Success)
	}
	// total=1 且已 failed=1 → finalize 应标 completed
	if task.Status != "completed" {
		t.Errorf("task.Status 应=completed（total=1, failed=1），实际=%s", task.Status)
	}
}

// TestHandleLocalAgentAck_Idempotent 重复 ack 不应重复递增 task 计数。
// HandleLocalAgentAck 在 record.status != pending 时直接 return nil，
// 不会再走计数递增逻辑。
func TestHandleLocalAgentAck_Idempotent(t *testing.T) {
	setupSkillInstancesDB(t)
	recID, taskID := seedLocalAckCase(t, "ack-dedup", "ack-dedup-skill", model.TaskTypeDistribute)

	rr1 := httptest.NewRecorder()
	HandleLocalAgentAck(rr1, jsonAckReq(t, recID, "ack-dedup", `{"status":"success"}`))
	if rr1.Code != http.StatusOK {
		t.Fatalf("第一次 ack 应 200，实际=%d", rr1.Code)
	}

	rr2 := httptest.NewRecorder()
	HandleLocalAgentAck(rr2, jsonAckReq(t, recID, "ack-dedup", `{"status":"success"}`))
	if rr2.Code != http.StatusOK {
		t.Fatalf("第二次 ack 应幂等返回 200，实际=%d", rr2.Code)
	}

	var task model.SkillDistributionTask
	if err := model.DB(context.Background()).First(&task, taskID).Error; err != nil {
		t.Fatalf("查 task: %v", err)
	}
	if task.Success != 1 {
		t.Errorf("重复 ack 后 task.Success 应仍=1，实际=%d", task.Success)
	}
	// 重复 ack 不应造成 status 反复变动，仍为 completed
	if task.Status != "completed" {
		t.Errorf("重复 ack 后 task.Status 应仍=completed，实际=%s", task.Status)
	}
}

// seedLocalAckMultiCase 同 seedLocalAckCase，但创建 N 个实例 + N 条 record，
// 配合 Task.Total=N 测试 finalize 边界。返回 record IDs（按 instance 顺序）和 task ID。
func seedLocalAckMultiCase(t *testing.T, username, slug string, n int) ([]uint, uint) {
	t.Helper()
	ctx := context.Background()
	if err := model.DB(ctx).AutoMigrate(&model.LocalInstanceSkill{}); err != nil {
		t.Fatalf("migrate LocalInstanceSkill: %v", err)
	}
	user := model.User{Username: username, Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	skill := model.Skill{
		Slug: slug, Name: slug, Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: model.VisibilityAll,
	}
	if err := model.DB(ctx).Create(&skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}
	task := model.SkillDistributionTask{
		SkillID: skill.ID, Version: skill.Version,
		Total: n, Status: "running", Type: model.TaskTypeDistribute,
	}
	if err := model.DB(ctx).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	recIDs := make([]uint, 0, n)
	for i := 0; i < n; i++ {
		inst := model.Instance{
			Name:       fmt.Sprintf("local-%s-%d", username, i),
			InstanceId: fmt.Sprintf("local-%s-%03d", username, i),
			UserID:     user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
		}
		if err := model.DB(ctx).Create(&inst).Error; err != nil {
			t.Fatalf("create inst[%d]: %v", i, err)
		}
		rec := model.SkillDistributionRecord{
			TaskID: task.ID, SkillID: skill.ID,
			InstanceID: inst.ID, InstanceCID: inst.InstanceId,
			Version: skill.Version, Status: "pending", Type: model.TaskTypeDistribute,
		}
		if err := model.DB(ctx).Create(&rec).Error; err != nil {
			t.Fatalf("create rec[%d]: %v", i, err)
		}
		recIDs = append(recIDs, rec.ID)
	}
	return recIDs, task.ID
}

// TestHandleLocalAgentAck_Finalize_PartialAck_StaysRunning
// Total=2 时第一次 ack 完只跑了 1 条，task.status 仍应为 running。
func TestHandleLocalAgentAck_Finalize_PartialAck_StaysRunning(t *testing.T) {
	setupSkillInstancesDB(t)
	recIDs, taskID := seedLocalAckMultiCase(t, "ack-partial", "ack-partial-skill", 2)

	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, jsonAckReq(t, recIDs[0], "ack-partial", `{"status":"success"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("ack 应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var task model.SkillDistributionTask
	if err := model.DB(context.Background()).First(&task, taskID).Error; err != nil {
		t.Fatalf("查 task: %v", err)
	}
	if task.Success != 1 || task.Failed != 0 {
		t.Errorf("第一次 ack：task.Success=1 task.Failed=0，实际 success=%d failed=%d",
			task.Success, task.Failed)
	}
	// success+failed=1 < total=2 → 仍为 running
	if task.Status != "running" {
		t.Errorf("还有未 ack record 时 task.Status 应=running，实际=%s", task.Status)
	}
}

// TestHandleLocalAgentAck_Finalize_AllAcked_BecomesCompleted
// Total=2 时两条 record 全部 ack 完后 task.status 应=completed，
// 且 success/failed 计数与每条 record 的终态一致。
func TestHandleLocalAgentAck_Finalize_AllAcked_BecomesCompleted(t *testing.T) {
	setupSkillInstancesDB(t)
	recIDs, taskID := seedLocalAckMultiCase(t, "ack-finalize", "ack-finalize-skill", 2)

	// 第一条 success，第二条 failed
	rr1 := httptest.NewRecorder()
	HandleLocalAgentAck(rr1, jsonAckReq(t, recIDs[0], "ack-finalize", `{"status":"success"}`))
	if rr1.Code != http.StatusOK {
		t.Fatalf("ack#1 应 200，实际=%d body=%s", rr1.Code, rr1.Body.String())
	}
	rr2 := httptest.NewRecorder()
	HandleLocalAgentAck(rr2, jsonAckReq(t, recIDs[1], "ack-finalize", `{"status":"failed","error":"boom"}`))
	if rr2.Code != http.StatusOK {
		t.Fatalf("ack#2 应 200，实际=%d body=%s", rr2.Code, rr2.Body.String())
	}

	var task model.SkillDistributionTask
	if err := model.DB(context.Background()).First(&task, taskID).Error; err != nil {
		t.Fatalf("查 task: %v", err)
	}
	if task.Success != 1 || task.Failed != 1 {
		t.Errorf("最终：task.Success=1 task.Failed=1，实际 success=%d failed=%d",
			task.Success, task.Failed)
	}
	// success+failed=2 == total=2 → finalize 标 completed
	if task.Status != "completed" {
		t.Errorf("全部 ack 后 task.Status 应=completed，实际=%s", task.Status)
	}
}

// TestHandleLocalAgentAck_MissingID body 里没 id 时应返 400。
// ack 从 path 参数改为 body 字段后，这是请求体必填校验。
func TestHandleLocalAgentAck_MissingID(t *testing.T) {
	setupSkillInstancesDB(t)
	ensureSessionStore()

	// 不走 jsonAckReq（它会自动注入 id），手工造个不带 id 的请求。
	req := httptest.NewRequest(http.MethodPost, "/local-agent/commands/ack",
		strings.NewReader(`{"status":"success"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// seed 一个登录 user
	user := model.User{Username: "ack-no-id", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = user.Username
	rrSeed := httptest.NewRecorder()
	session.Save(req, rrSeed)
	for _, c := range rrSeed.Result().Cookies() {
		req.AddCookie(c)
	}

	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("缺 id 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleLocalAgentAck_Source_Enterprise reporter ack=success 时，本地实例
// 安装来源应该按「rec.SkillID 是否为 0」二分写入：
//   - SkillID != 0 → 企业内部 skill（即使 visibility_type=all） → enterprise
//
// 历史 bug：原代码用 inferLocalSkillSource(skill.VisibilityType) 推断，
// VisibilityType 默认是 'all'，企业 skill 会被误判为 public。
func TestHandleLocalAgentAck_Source_Enterprise(t *testing.T) {
	setupSkillInstancesDB(t)
	recID, _ := seedLocalAckCase(t, "src-ent", "src-ent-skill", model.TaskTypeDistribute)
	// seedLocalAckCase 默认 VisibilityType=all（企业 skill 默认）

	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, jsonAckReq(t, recID, "src-ent", `{"status":"success"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("ack 应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var lis model.LocalInstanceSkill
	if err := model.DB(context.Background()).
		Where("slug = ?", "src-ent-skill").First(&lis).Error; err != nil {
		t.Fatalf("查 LocalInstanceSkill: %v", err)
	}
	if lis.Source != model.LocalSkillSourceEnterprise {
		t.Errorf("企业 skill (skill.ID!=0) 应记 enterprise，实际=%s", lis.Source)
	}
}

// TestHandleLocalAgentAck_Source_Public 公共 ClawHub 兜底（add-skill 本地路径）
// 写入的 record 恒 SkillID=0，ack 后应记 public。
func TestHandleLocalAgentAck_Source_Public(t *testing.T) {
	setupSkillInstancesDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.LocalInstanceSkill{}); err != nil {
		t.Fatalf("migrate LocalInstanceSkill: %v", err)
	}
	ctx := context.Background()
	user := model.User{Username: "src-pub", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	inst := model.Instance{
		Name: "local-src-pub", InstanceId: "local-src-pub-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	if err := model.DB(ctx).Create(&inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}
	// task 模拟 add-skill 路径：SkillID=0，task.Slug 兜底
	task := model.SkillDistributionTask{
		SkillID: 0, Slug: "src-pub-clawhub",
		Total: 1, Status: "running", Type: model.TaskTypeDistribute,
	}
	if err := model.DB(ctx).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	rec := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: 0,
		InstanceID: inst.ID, InstanceCID: inst.InstanceId,
		Status: "pending", Type: model.TaskTypeDistribute,
	}
	if err := model.DB(ctx).Create(&rec).Error; err != nil {
		t.Fatalf("create rec: %v", err)
	}

	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, jsonAckReq(t, rec.ID, "src-pub", `{"status":"success","version":"1.2.3"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("ack 应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var lis model.LocalInstanceSkill
	if err := model.DB(ctx).
		Where("slug = ?", "src-pub-clawhub").First(&lis).Error; err != nil {
		t.Fatalf("查 LocalInstanceSkill: %v", err)
	}
	if lis.Source != model.LocalSkillSourcePublic {
		t.Errorf("ClawHub 兜底 (skill.ID==0) 应记 public，实际=%s", lis.Source)
	}
}
