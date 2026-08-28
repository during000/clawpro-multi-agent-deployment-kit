package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"gorm.io/gorm"
)

// adminJSONPost 创建带 admin Bearer Token 的 POST 请求
func adminJSONPost(url, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

func TestHandleDistributeSkill_AllUnsupportedTypes(t *testing.T) {
	setupSkillInstancesDB(t)

	// 创建技能
	skill := model.Skill{
		Slug: "dist-test", Name: "下发测试技能", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skill)

	// 创建不支持技能的实例（hermes、lightclawace）
	user := model.User{Username: "dist-user"}
	model.DB(context.Background()).Create(&user)

	instances := []model.Instance{
		{Name: "unknown-inst-1", InstanceId: "ins-unk-001", UserID: user.ID, AgentType: "unknown_type_a"},
		{Name: "unknown-inst-2", InstanceId: "ins-unk-002", UserID: user.ID, AgentType: "unknown_type_b"},
	}
	for i := range instances {
		model.DB(context.Background()).Create(&instances[i])
	}

	body := `{"slug":"dist-test","instance_ids":[` +
		uintStr(instances[0].ID) + `,` + uintStr(instances[1].ID) + `]}`

	w := httptest.NewRecorder()
	HandleDistributeSkill(w, adminJSONPost("/admin/skills/distribute", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for all unsupported types, got %d, body: %s", w.Code, w.Body.String())
	}

	// 确认没有创建任何任务记录（先过滤再创建任务）
	var taskCount int64
	model.DB(context.Background()).Model(&model.SkillDistributionTask{}).Where("skill_id = ?", skill.ID).Count(&taskCount)
	if taskCount != 0 {
		t.Errorf("expected 0 tasks (filtered before creation), got %d", taskCount)
	}
}

func TestHandleDistributeSkill_MixedTypes_OnlyValidCreatesTask(t *testing.T) {
	setupSkillInstancesDB(t)
	// 获取当前测试的 DB 实例，用于轮询（避免全局 gdb 被并发测试替换后 WithContext 引用失效）
	testDB := model.DB(context.Background())

	// 创建技能
	skill := model.Skill{
		Slug: "dist-mixed", Name: "混合类型测试", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skill)

	user := model.User{Username: "mixed-user"}
	model.DB(context.Background()).Create(&user)

	// v7：openclaw 支持，unknown 不支持（Hermes/ACE 现已支持，不再适用"混合"场景）
	instances := []model.Instance{
		{Name: "oc-inst", InstanceId: "ins-oc-001", UserID: user.ID, AgentType: "openclaw"},
		{Name: "unk-inst", InstanceId: "ins-unk-001", UserID: user.ID, AgentType: "unknown_type"},
	}
	for i := range instances {
		model.DB(context.Background()).Create(&instances[i])
	}

	body := `{"slug":"dist-mixed","instance_ids":[` +
		uintStr(instances[0].ID) + `,` + uintStr(instances[1].ID) + `]}`

	w := httptest.NewRecorder()
	HandleDistributeSkill(w, adminJSONPost("/admin/skills/distribute", body))

	// 应该成功（至少有一个支持技能的实例）
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for mixed types, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	// 等待异步 goroutine 写入任务记录（最多 5s，用 testDB 避免全局 gdb 被替换）
	var task model.SkillDistributionTask
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if testDB.Where("skill_id = ?", skill.ID).First(&task).Error == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 验证任务记录创建了，且 Total 只包含支持技能的实例
	if testDB.Where("skill_id = ?", skill.ID).First(&task).Error != nil {
		t.Fatal("expected task to be created")
	}
	if task.Total != 1 {
		t.Errorf("expected task.Total=1 (only openclaw), got %d", task.Total)
	}

	// 等待下发记录写入（最多 5s）
	var recordCount int64
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		testDB.Model(&model.SkillDistributionRecord{}).Where("task_id = ?", task.ID).Count(&recordCount)
		if recordCount > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 验证下发记录只为 openclaw 实例创建
	if recordCount != 1 {
		t.Errorf("expected 1 distribution record, got %d", recordCount)
	}
}

// TestHandleDistributeSkill_LocalInstance_KeepsPending /admin/skills/distribute 对
// 本地 agent 实例（agent_type=workbuddy/codebuddy，不在内置 agentTypesMap）必须放行：
//   - 不被 AgentTypeSupportsSkill 过滤掉
//   - 创建对应的 SkillDistributionRecord（status=pending）
//   - 不交给 executeSkillTaskAsync 跑 RunScript（本地实例没法走 TAT/CVM 路径）
//   - record 保留 pending，等 reporter 来 ack
func TestHandleDistributeSkill_LocalInstance_KeepsPending(t *testing.T) {
	setupSkillInstancesDB(t)
	testDB := model.DB(context.Background())

	skill := model.Skill{
		Slug: "dist-local", Name: "本地下发测试", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	testDB.Create(&skill)

	user := model.User{Username: "local-dist-user"}
	testDB.Create(&user)

	localInst := model.Instance{
		Name: "local-codebuddy", InstanceId: "local-codebuddy-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	testDB.Create(&localInst)

	body := `{"slug":"dist-local","instance_ids":[` + uintStr(localInst.ID) + `]}`
	w := httptest.NewRecorder()
	HandleDistributeSkill(w, adminJSONPost("/admin/skills/distribute", body))

	if w.Code != http.StatusOK {
		t.Fatalf("本地实例 distribute 应 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	// 等任务+record 写入
	var task model.SkillDistributionTask
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if testDB.Where("skill_id = ?", skill.ID).First(&task).Error == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if testDB.Where("skill_id = ?", skill.ID).First(&task).Error != nil {
		t.Fatal("expected task to be created")
	}

	var rec model.SkillDistributionRecord
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if testDB.Where("task_id = ? AND instance_id = ?", task.ID, localInst.ID).First(&rec).Error == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if testDB.Where("task_id = ? AND instance_id = ?", task.ID, localInst.ID).First(&rec).Error != nil {
		t.Fatal("expected SkillDistributionRecord to be created for local instance")
	}

	// 关键：本地实例的 record 必须保留 pending（async executor 没碰它），等 reporter ack
	// 等 1s 让 async finalize 跑完
	time.Sleep(1 * time.Second)
	testDB.Where("task_id = ? AND instance_id = ?", task.ID, localInst.ID).First(&rec)
	if rec.Status != "pending" {
		t.Errorf("本地实例 record 应保持 pending（等 reporter ack），实际=%q", rec.Status)
	}
}

func TestHandleDistributeSkill_BatchRejectsTopLevelFields(t *testing.T) {
	setupSkillInstancesDB(t)
	body := `{"slug":"top-level","instance_ids":[1],"skills":[{"source":"public","slug":"public-one","version":"1.0.0"}]}`
	w := httptest.NewRecorder()
	HandleDistributeSkill(w, adminJSONPost("/admin/skills/distribute", body))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when skills[] is combined with top-level skill fields, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDistributeSkill_BatchLimit(t *testing.T) {
	setupSkillInstancesDB(t)
	var b strings.Builder
	b.WriteString(`{"instance_ids":[1],"skills":[`)
	for i := 0; i < skillTaskMaxItems+1; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"source":"public","slug":"public-limit","version":"1.0.0"}`)
	}
	b.WriteString(`]}`)
	w := httptest.NewRecorder()
	HandleDistributeSkill(w, adminJSONPost("/admin/skills/distribute", b.String()))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when skills[] exceeds limit, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDistributeSkill_BatchEnterpriseCreatesTasksWithBatchMetadata(t *testing.T) {
	setupSkillInstancesDB(t)
	inst := seedAdminUninstallInstance(t, "dist-batch-user", "dist-batch-inst", "ins-dist-batch", "openclaw")
	skillA := model.Skill{Slug: "dist-batch-a", Name: "Batch A", Version: "1.0.0", VersionMajor: 1, VersionMinor: 0, VersionPatch: 0, VisibilityType: model.VisibilityAll}
	skillB := model.Skill{Slug: "dist-batch-b", Name: "Batch B", Version: "2.0.0", VersionMajor: 2, VersionMinor: 0, VersionPatch: 0, VisibilityType: model.VisibilityAll}
	if err := model.DB(context.Background()).Create(&skillA).Error; err != nil {
		t.Fatalf("创建技能 A 失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&skillB).Error; err != nil {
		t.Fatalf("创建技能 B 失败: %v", err)
	}

	body := `{"instance_ids":[` + uintStr(inst.ID) + `],"skills":[` +
		`{"source":"enterprise","slug":"` + skillA.Slug + `","version":"1.0.0"},` +
		`{"slug":"` + skillB.Slug + `","version":"2.0.0"}]}`
	w := httptest.NewRecorder()
	HandleDistributeSkill(w, adminJSONPost("/admin/skills/distribute", body))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	batchID, ok := resp["batch_id"].(string)
	if !ok || batchID == "" {
		t.Fatalf("响应缺少 batch_id: %v", resp)
	}
	if int(resp["submitted"].(float64)) != 2 || int(resp["failed"].(float64)) != 0 {
		t.Fatalf("批量汇总错误: %v", resp)
	}

	var tasks []model.SkillDistributionTask
	if err := model.DB(context.Background()).Where("batch_id = ?", batchID).Order("id asc").Find(&tasks).Error; err != nil {
		t.Fatalf("查询批量任务失败: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	for _, task := range tasks {
		if task.Source != model.SkillSourceEnterprise {
			t.Fatalf("task source=%q, want enterprise", task.Source)
		}
		if task.BatchID != batchID {
			t.Fatalf("task batch_id=%q, want %q", task.BatchID, batchID)
		}
		if task.Total != 1 {
			t.Fatalf("task total=%d, want 1", task.Total)
		}
	}
	waitSkillTaskAsync(t)
}

func TestPreparePublicSkillForDistributeStagesZip(t *testing.T) {
	origHTTPClient := SkillHTTPClient
	defer func() {
		SkillHTTPClient = origHTTPClient
	}()

	SkillHTTPClient = &http.Client{Transport: skillBundleRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("zip-data")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}
	fakeStorage := &skillBundleFakeStorage{}
	deps := publicSkillDistributeDeps{
		commonStorageClient: func(ctx context.Context) (StorageClient, error) {
			return fakeStorage, nil
		},
		commonDownloadURL: func(ctx context.Context, fileKey string, internalDomain bool) (string, error) {
			if !internalDomain {
				t.Fatal("expected internal common download URL")
			}
			return "https://smh.example.com/" + fileKey, nil
		},
	}

	item := skillTaskItem{Index: 0, Source: model.SkillSourcePublic, Slug: "public-stage", Version: "1.2.3", SourceSkillsetSlug: "pkg-stage"}
	prepared, reason, richErr := preparePublicSkillForDistribute(context.Background(), item, deps)
	if richErr != nil {
		t.Fatalf("prepare public skill failed reason=%s err=%v", reason, richErr)
	}
	if prepared.Version != "1.2.3" || prepared.DownloadURL == "" || prepared.SourceSkillsetSlug != "pkg-stage" {
		t.Fatalf("unexpected public skill item: %+v", prepared)
	}
	wantKey := "public-skills/public-stage/public-stage-1.2.3.zip"
	if got := string(fakeStorage.uploads[wantKey]); got != "zip-data" {
		t.Fatalf("uploaded zip for %s = %q, want zip-data", wantKey, got)
	}
}

func TestPreparePublicSkillForDistributeLatestResolvesFinalVersion(t *testing.T) {
	origHTTPClient := SkillHTTPClient
	defer func() {
		SkillHTTPClient = origHTTPClient
	}()

	SkillHTTPClient = &http.Client{Transport: skillBundleRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "lightmake.site":
			if got := req.URL.Query().Get("version"); got != "" {
				t.Fatalf("version query = %q, want empty for latest", got)
			}
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": []string{"https://skillhub.example.com/skills/public-latest/4.5.6.zip"}},
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		case "skillhub.example.com":
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("zip-data")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		default:
			t.Fatalf("unexpected host %q", req.URL.Host)
			return nil, nil
		}
	})}
	fakeStorage := &skillBundleFakeStorage{}
	deps := publicSkillDistributeDeps{
		commonStorageClient: func(ctx context.Context) (StorageClient, error) {
			return fakeStorage, nil
		},
		commonDownloadURL: func(ctx context.Context, fileKey string, internalDomain bool) (string, error) {
			return "https://smh.example.com/" + fileKey, nil
		},
	}

	item := skillTaskItem{Index: 0, Source: model.SkillSourcePublic, Slug: "public-latest", Version: "latest"}
	prepared, reason, richErr := preparePublicSkillForDistribute(context.Background(), item, deps)
	if richErr != nil {
		t.Fatalf("prepare public latest failed reason=%s err=%v", reason, richErr)
	}
	if prepared.Version != "4.5.6" {
		t.Fatalf("prepared version = %q, want 4.5.6", prepared.Version)
	}
	if got := string(fakeStorage.uploads["public-skills/public-latest/public-latest-4.5.6.zip"]); got != "zip-data" {
		t.Fatalf("uploaded latest zip = %q, want zip-data", got)
	}
}

func TestPreparePublicSkillForDistributeErrorBranches(t *testing.T) {
	origHTTPClient := SkillHTTPClient
	defer func() {
		SkillHTTPClient = origHTTPClient
	}()

	SkillHTTPClient = &http.Client{Transport: skillBundleRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("zip-data")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}
	defaultDeps := publicSkillDistributeDeps{
		commonStorageClient: func(ctx context.Context) (StorageClient, error) {
			return &skillBundleFakeStorage{}, nil
		},
		commonDownloadURL: func(ctx context.Context, fileKey string, internalDomain bool) (string, error) {
			return "https://smh.example.com/" + fileKey, nil
		},
	}
	if _, reason, richErr := preparePublicSkillForDistribute(context.Background(), skillTaskItem{Source: model.SkillSourcePublic, Slug: "public-no-version"}, defaultDeps); richErr == nil || reason != "version_required" {
		t.Fatalf("expected version_required, got reason=%s err=%v", reason, richErr)
	}

	uploadFailDeps := publicSkillDistributeDeps{
		commonStorageClient: func(ctx context.Context) (StorageClient, error) {
			return &skillBundleFakeStorage{uploadErr: errors.New("upload failed")}, nil
		},
		commonDownloadURL: defaultDeps.commonDownloadURL,
	}
	if _, reason, richErr := preparePublicSkillForDistribute(context.Background(), skillTaskItem{Source: model.SkillSourcePublic, Slug: "public-upload-fail", Version: "1.0.0"}, uploadFailDeps); richErr == nil || reason != "upload_public_zip_failed" {
		t.Fatalf("expected upload_public_zip_failed, got reason=%s err=%v", reason, richErr)
	}

	downloadURLFailDeps := publicSkillDistributeDeps{
		commonStorageClient: func(ctx context.Context) (StorageClient, error) {
			return &skillBundleFakeStorage{}, nil
		},
		commonDownloadURL: func(ctx context.Context, fileKey string, internalDomain bool) (string, error) {
			return "", errors.New("download url failed")
		},
	}
	if _, reason, richErr := preparePublicSkillForDistribute(context.Background(), skillTaskItem{Source: model.SkillSourcePublic, Slug: "public-url-fail", Version: "1.0.0"}, downloadURLFailDeps); richErr == nil || reason != "download_url_failed" {
		t.Fatalf("expected download_url_failed, got reason=%s err=%v", reason, richErr)
	}
}

func TestHandleDistributeSkill_BatchPartialFailureKeepsResults(t *testing.T) {
	setupSkillInstancesDB(t)
	inst := seedAdminUninstallInstance(t, "dist-partial-user", "dist-partial-inst", "ins-dist-partial", "openclaw")
	skill := model.Skill{Slug: "dist-partial", Name: "Partial", Version: "1.0.0", VersionMajor: 1, VersionMinor: 0, VersionPatch: 0, VisibilityType: model.VisibilityAll}
	if err := model.DB(context.Background()).Create(&skill).Error; err != nil {
		t.Fatalf("创建技能失败: %v", err)
	}

	body := `{"instance_ids":[` + uintStr(inst.ID) + `],"skills":[` +
		`{"source":"enterprise","slug":"` + skill.Slug + `","version":"1.0.0"},` +
		`{"source":"unknown","slug":"bad-source","version":"1.0.0"}]}`
	w := httptest.NewRecorder()
	HandleDistributeSkill(w, adminJSONPost("/admin/skills/distribute", body))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for partial failure batch, got %d, body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	if int(resp["submitted"].(float64)) != 1 || int(resp["failed"].(float64)) != 1 {
		t.Fatalf("partial batch summary mismatch: %v", resp)
	}
	results := resp["results"].([]interface{})
	if results[1].(map[string]interface{})["status"] != "failed" {
		t.Fatalf("second result should be failed: %v", results)
	}
	waitSkillTaskAsync(t)
}

// TestHandleDistributeSkill_SupersedePendingPrevious
// 若上一次同 slug+source 的 distribute 仍有 pending 记录（reporter 还没拉走），
// 新一次下发应把上一次任务判失败、其 pending 记录置 failed，再创建本次新任务的 pending。
func TestHandleDistributeSkill_SupersedePendingPrevious(t *testing.T) {
	setupSkillInstancesDB(t)
	testDB := model.DB(context.Background())

	skill := model.Skill{
		Slug: "sup-skill", Name: "上一次pending测试", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	testDB.Create(&skill)

	user := model.User{Username: "sup-skill-user"}
	testDB.Create(&user)

	localInst := model.Instance{
		Name: "sup-skill-box", InstanceId: "local-sup-skill-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	testDB.Create(&localInst)

	// 模拟上一次下发：reporter 还没拉走，record 仍 pending（task 已 completed）
	prevTask := model.SkillDistributionTask{
		SkillID: skill.ID, Version: skill.Version, Source: model.SkillSourceEnterprise,
		Slug: skill.Slug, OperatorID: user.ID, Total: 1, Status: "completed",
		Type: model.TaskTypeDistribute,
	}
	testDB.Create(&prevTask)
	prevRec := model.SkillDistributionRecord{
		TaskID: prevTask.ID, SkillID: skill.ID, InstanceID: localInst.ID,
		InstanceCID: localInst.InstanceId, Version: skill.Version,
		Status: model.RecordStatusPending, Type: model.TaskTypeDistribute,
	}
	testDB.Create(&prevRec)

	body := `{"slug":"sup-skill","instance_ids":[` + uintStr(localInst.ID) + `]}`
	w := httptest.NewRecorder()
	HandleDistributeSkill(w, adminJSONPost("/admin/skills/distribute", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	waitSkillTaskAsync(t)

	// 上一次任务本身不应被改（task 可能给多个本地 agent 下发，其它可能已成功）
	var gotPrev model.SkillDistributionTask
	if testDB.First(&gotPrev, prevTask.ID).Error != nil {
		t.Fatal("上一次任务不存在")
	}
	if gotPrev.Status != "completed" {
		t.Errorf("上一次任务不应被改，应 completed，实际=%s", gotPrev.Status)
	}
	// 上一次任务的 failed 计数应累加本次判失败的 pending 数量（此处为 1）
	if gotPrev.Failed != 1 {
		t.Errorf("上一次任务 failed 计数应为 1（累加 pending 数），实际=%d", gotPrev.Failed)
	}

	// 上一次 record 应 failed（原因：已下发新的版本）
	var gotPrevRec model.SkillDistributionRecord
	if testDB.First(&gotPrevRec, prevRec.ID).Error != nil {
		t.Fatal("上一次 record 不存在")
	}
	if gotPrevRec.Status != model.RecordStatusFailed {
		t.Errorf("上一次 record 应 failed，实际=%s", gotPrevRec.Status)
	}
	if gotPrevRec.Error != "已下发新的版本" {
		t.Errorf("上一次 record error 应为「已下发新的版本」，实际=%s", gotPrevRec.Error)
	}

	// 新任务应存在且有 1 条 pending record
	var newTasks []model.SkillDistributionTask
	testDB.Where("skill_id = ? AND id <> ?", skill.ID, prevTask.ID).Find(&newTasks)
	if len(newTasks) != 1 {
		t.Fatalf("应恰好 1 个新任务，实际=%d", len(newTasks))
	}
	var newPending int64
	testDB.Model(&model.SkillDistributionRecord{}).
		Where("task_id = ? AND status = ?", newTasks[0].ID, model.RecordStatusPending).Count(&newPending)
	if newPending != 1 {
		t.Errorf("新任务应剩 1 条 pending record，实际=%d", newPending)
	}
}

// TestHandleDistributeSkill_NoSupersedeWhenPreviousDone
// 上一次下发已完成（无 pending 记录），新一次下发不应误判上一次失败。
func TestHandleDistributeSkill_NoSupersedeWhenPreviousDone(t *testing.T) {
	setupSkillInstancesDB(t)
	testDB := model.DB(context.Background())

	skill := model.Skill{
		Slug: "nodup-skill", Name: "无pending测试", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	testDB.Create(&skill)

	user := model.User{Username: "nodup-skill-user"}
	testDB.Create(&user)

	localInst := model.Instance{
		Name: "nodup-skill-box", InstanceId: "local-nodup-skill-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	testDB.Create(&localInst)

	// 上一次下发：record 已 success，无 pending
	prevTask := model.SkillDistributionTask{
		SkillID: skill.ID, Version: skill.Version, Source: model.SkillSourceEnterprise,
		Slug: skill.Slug, OperatorID: user.ID, Total: 1, Success: 1, Status: "completed",
		Type: model.TaskTypeDistribute,
	}
	testDB.Create(&prevTask)
	prevRec := model.SkillDistributionRecord{
		TaskID: prevTask.ID, SkillID: skill.ID, InstanceID: localInst.ID,
		InstanceCID: localInst.InstanceId, Version: skill.Version,
		Status: model.RecordStatusSuccess, Type: model.TaskTypeDistribute,
	}
	testDB.Create(&prevRec)

	body := `{"slug":"nodup-skill","instance_ids":[` + uintStr(localInst.ID) + `]}`
	w := httptest.NewRecorder()
	HandleDistributeSkill(w, adminJSONPost("/admin/skills/distribute", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	waitSkillTaskAsync(t)

	var gotPrev model.SkillDistributionTask
	if testDB.First(&gotPrev, prevTask.ID).Error != nil {
		t.Fatal("上一次任务不存在")
	}
	if gotPrev.Status != "completed" {
		t.Errorf("上一次任务不应被改，应 completed，实际=%s", gotPrev.Status)
	}
}

// TestHandleDistributeSkill_SupersedeOnlySameInstance
// 核心规则：只有本次请求涉及的「相同 instance_id」的 pending 记录才被置失败。
// 上一次任务给实例 A、B 都下发了且都 pending；本次只给实例 A 下发——
// 则 A 被置失败、上一次任务 failed+1，而 B 的 pending 必须保留（不被本次误判）。
func TestHandleDistributeSkill_SupersedeOnlySameInstance(t *testing.T) {
	setupSkillInstancesDB(t)
	testDB := model.DB(context.Background())

	skill := model.Skill{
		Slug: "sup-multi", Name: "多实例相同instance测试", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	testDB.Create(&skill)

	user := model.User{Username: "sup-multi-user"}
	testDB.Create(&user)

	instA := model.Instance{
		Name: "sup-multi-a", InstanceId: "local-sup-multi-a",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	instB := model.Instance{
		Name: "sup-multi-b", InstanceId: "local-sup-multi-b",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	testDB.Create(&instA)
	testDB.Create(&instB)

	// 上一次任务给 A、B 都下发，且都 pending（reporter 还没拉走）
	prevTask := model.SkillDistributionTask{
		SkillID: skill.ID, Version: skill.Version, Source: model.SkillSourceEnterprise,
		Slug: skill.Slug, OperatorID: user.ID, Total: 2, Status: "completed",
		Type: model.TaskTypeDistribute,
	}
	testDB.Create(&prevTask)
	recA := model.SkillDistributionRecord{
		TaskID: prevTask.ID, SkillID: skill.ID, InstanceID: instA.ID,
		InstanceCID: instA.InstanceId, Version: skill.Version,
		Status: model.RecordStatusPending, Type: model.TaskTypeDistribute,
	}
	recB := model.SkillDistributionRecord{
		TaskID: prevTask.ID, SkillID: skill.ID, InstanceID: instB.ID,
		InstanceCID: instB.InstanceId, Version: skill.Version,
		Status: model.RecordStatusPending, Type: model.TaskTypeDistribute,
	}
	testDB.Create(&recA)
	testDB.Create(&recB)

	// 本次只给 A 下发
	body := `{"slug":"sup-multi","instance_ids":[` + uintStr(instA.ID) + `]}`
	w := httptest.NewRecorder()
	HandleDistributeSkill(w, adminJSONPost("/admin/skills/distribute", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	waitSkillTaskAsync(t)

	// 上一次任务 failed 计数应只累加本次涉及的实例数（仅 A = 1）
	var gotPrev model.SkillDistributionTask
	if testDB.First(&gotPrev, prevTask.ID).Error != nil {
		t.Fatal("上一次任务不存在")
	}
	if gotPrev.Failed != 1 {
		t.Errorf("上一次任务 failed 计数应为 1，实际=%d", gotPrev.Failed)
	}

	// A 的 pending 应被置 failed
	var gotA model.SkillDistributionRecord
	if testDB.First(&gotA, recA.ID).Error != nil {
		t.Fatal("A record 不存在")
	}
	if gotA.Status != model.RecordStatusFailed {
		t.Errorf("A 的 pending 应被置 failed，实际=%s", gotA.Status)
	}
	if gotA.Error != "已下发新的版本" {
		t.Errorf("A record error 应为「已下发新的版本」，实际=%s", gotA.Error)
	}

	// B 的 pending 必须保留（本次没涉及 B，不能被误判）
	var gotB model.SkillDistributionRecord
	if testDB.First(&gotB, recB.ID).Error != nil {
		t.Fatal("B record 不存在")
	}
	if gotB.Status != model.RecordStatusPending {
		t.Errorf("B 的 pending 不应被本次误判，应保持 pending，实际=%s", gotB.Status)
	}

	// 新任务只给 A 创建 1 条 pending record
	var newTasks []model.SkillDistributionTask
	testDB.Where("skill_id = ? AND id <> ?", skill.ID, prevTask.ID).Find(&newTasks)
	if len(newTasks) != 1 {
		t.Fatalf("应恰好 1 个新任务，实际=%d", len(newTasks))
	}
	var newPending int64
	testDB.Model(&model.SkillDistributionRecord{}).
		Where("task_id = ? AND status = ?", newTasks[0].ID, model.RecordStatusPending).Count(&newPending)
	if newPending != 1 {
		t.Errorf("新任务应只有 1 条 pending record（仅 A），实际=%d", newPending)
	}
}

func TestHandleDistributeSkill_SelectAllByStatusAndGroup(t *testing.T) {
	setupSkillInstancesDB(t)
	ctx := context.Background()
	db := model.DB(ctx)

	skill := model.Skill{
		Slug: "select-all-skill", Name: "全选测试技能", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: model.VisibilityAll,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("创建技能失败: %v", err)
	}
	group := model.UserGroup{Name: "select-all-skill-group"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("创建用户组失败: %v", err)
	}
	groupedUser := model.User{Username: "select-all-skill-grouped"}
	otherUser := model.User{Username: "select-all-skill-other"}
	if err := db.Create(&groupedUser).Error; err != nil {
		t.Fatalf("创建分组用户失败: %v", err)
	}
	if err := db.Create(&otherUser).Error; err != nil {
		t.Fatalf("创建其他用户失败: %v", err)
	}
	if err := db.Create(&model.UserGroupMember{UserGroupID: group.ID, UserID: groupedUser.ID}).Error; err != nil {
		t.Fatalf("创建用户组成员失败: %v", err)
	}
	instances := []model.Instance{
		{Name: "select-all-skill-match", InstanceId: "ins-skill-match", UserID: groupedUser.ID, AgentType: "openclaw"},
		{Name: "select-all-skill-other", InstanceId: "ins-skill-other", UserID: otherUser.ID, AgentType: "openclaw"},
		{Name: "select-all-skill-unsupported", InstanceId: "ins-skill-unsupported", UserID: groupedUser.ID, AgentType: "unsupported"},
	}
	if err := db.Create(&instances).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	body := `{"source":"enterprise","slug":"` + skill.Slug + `","version":"1.0.0",` +
		`"select_all":true,"statuses":["uninstalled"],"group_ids":[` + uintStr(group.ID) + `]}`
	w := httptest.NewRecorder()
	HandleDistributeSkill(w, adminJSONPost("/admin/skills/distribute", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	if got := int(resp["total"].(float64)); got != 1 {
		t.Fatalf("total=%d，期望 1", got)
	}
	var records []model.SkillDistributionRecord
	if err := db.Find(&records).Error; err != nil {
		t.Fatalf("查询下发记录失败: %v", err)
	}
	if len(records) != 1 || records[0].InstanceID != instances[0].ID {
		t.Fatalf("下发记录=%+v，期望只包含实例 %d", records, instances[0].ID)
	}
	waitSkillTaskAsync(t)
}

func TestHandleDistributeSkill_SelectAllBatchResolvesEachSkillIndependently(t *testing.T) {
	setupSkillInstancesDB(t)
	ctx := context.Background()
	db := model.DB(ctx)

	skills := []model.Skill{
		{Slug: "select-all-batch-a", Name: "Batch A", Version: "1.0.0", VersionMajor: 1, VisibilityType: model.VisibilityAll},
		{Slug: "select-all-batch-b", Name: "Batch B", Version: "1.0.0", VersionMajor: 1, VisibilityType: model.VisibilityAll},
	}
	if err := db.Create(&skills).Error; err != nil {
		t.Fatalf("创建技能失败: %v", err)
	}
	user := model.User{Username: "select-all-skill-batch-user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	instances := []model.Instance{
		{Name: "select-all-batch-installed", InstanceId: "ins-skill-batch-installed", UserID: user.ID, AgentType: "openclaw"},
		{Name: "select-all-batch-new", InstanceId: "ins-skill-batch-new", UserID: user.ID, AgentType: "openclaw"},
	}
	if err := db.Create(&instances).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	previousTask := model.SkillDistributionTask{
		SkillID: skills[0].ID, Slug: skills[0].Slug, Version: skills[0].Version,
		Type: model.TaskTypeDistribute, Status: model.TaskStatusCompleted, Total: 1, Success: 1,
	}
	if err := db.Create(&previousTask).Error; err != nil {
		t.Fatalf("创建历史任务失败: %v", err)
	}
	if err := db.Create(&model.SkillDistributionRecord{
		TaskID: previousTask.ID, SkillID: skills[0].ID, InstanceID: instances[0].ID,
		InstanceCID: instances[0].InstanceId, Version: skills[0].Version,
		Type: model.TaskTypeDistribute, Status: model.RecordStatusSuccess,
	}).Error; err != nil {
		t.Fatalf("创建历史记录失败: %v", err)
	}

	body := `{"select_all":true,"statuses":["uninstalled"],"skills":[` +
		`{"source":"enterprise","slug":"` + skills[0].Slug + `","version":"1.0.0"},` +
		`{"source":"enterprise","slug":"` + skills[1].Slug + `","version":"1.0.0"}]}`
	w := httptest.NewRecorder()
	HandleDistributeSkill(w, adminJSONPost("/admin/skills/distribute", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	if got := int(resp["total"].(float64)); got != 2 {
		t.Fatalf("批量 total=%d，期望技能条目数 2", got)
	}
	results := resp["results"].([]interface{})
	if got := int(results[0].(map[string]interface{})["instance_count"].(float64)); got != 1 {
		t.Fatalf("技能 A instance_count=%d，期望 1", got)
	}
	if got := int(results[1].(map[string]interface{})["instance_count"].(float64)); got != 2 {
		t.Fatalf("技能 B instance_count=%d，期望 2", got)
	}
	waitSkillTaskAsync(t)
}

func TestHandleDistributeSkill_SelectAllNoTargets(t *testing.T) {
	setupSkillInstancesDB(t)
	db := model.DB(context.Background())
	skill := model.Skill{
		Slug: "select-all-skill-empty", Name: "空目标技能", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: model.VisibilityAll,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("创建技能失败: %v", err)
	}
	w := httptest.NewRecorder()
	HandleDistributeSkill(w, adminJSONPost(
		"/admin/skills/distribute",
		`{"source":"enterprise","slug":"select-all-skill-empty","version":"1.0.0","select_all":true,"statuses":["uninstalled"]}`,
	))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var taskCount int64
	if err := db.Model(&model.SkillDistributionTask{}).Count(&taskCount).Error; err != nil {
		t.Fatalf("统计任务失败: %v", err)
	}
	if taskCount != 0 {
		t.Fatalf("零目标不应创建任务，实际=%d", taskCount)
	}
}

func TestHandleDistributeSkill_SelectAllKeepsLocalAgentPending(t *testing.T) {
	setupSkillInstancesDB(t)
	db := model.DB(context.Background())
	skill := model.Skill{
		Slug: "select-all-skill-local", Name: "本地技能", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: model.VisibilityAll,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("创建技能失败: %v", err)
	}
	user := model.User{Username: "select-all-skill-local"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	instance := model.Instance{
		Name:       "select-all-skill-local",
		InstanceId: "local-select-all-skill",
		UserID:     user.ID,
		Source:     model.InstanceSourceLocal,
		AgentType:  "unsupported",
	}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatalf("创建本地实例失败: %v", err)
	}

	w := httptest.NewRecorder()
	HandleDistributeSkill(w, adminJSONPost(
		"/admin/skills/distribute",
		`{"source":"enterprise","slug":"select-all-skill-local","version":"1.0.0","select_all":true,"statuses":["uninstalled"]}`,
	))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	waitSkillTaskAsync(t)
	var record model.SkillDistributionRecord
	if err := db.Where("instance_id = ?", instance.ID).First(&record).Error; err != nil {
		t.Fatalf("查询本地实例记录失败: %v", err)
	}
	if record.Status != model.RecordStatusPending {
		t.Fatalf("本地实例 record.status=%q，期望 pending", record.Status)
	}
}

func TestCleanupSkillSelectAllTask_RemovesPartialData(t *testing.T) {
	setupSkillInstancesDB(t)
	db := model.DB(context.Background())
	task := model.SkillDistributionTask{
		Slug: "cleanup-select-all-skill", Version: "1.0.0",
		Status: model.TaskStatusRunning, Type: model.TaskTypeDistribute,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	if err := db.Create(&model.SkillDistributionRecord{
		TaskID: task.ID, InstanceID: 1, InstanceCID: "ins-cleanup-skill",
		Version: "1.0.0", Status: model.RecordStatusPending, Type: model.TaskTypeDistribute,
	}).Error; err != nil {
		t.Fatalf("创建记录失败: %v", err)
	}
	cleanupSkillSelectAllTask(context.Background(), task.ID)
	var taskCount, recordCount int64
	if err := db.Model(&model.SkillDistributionTask{}).Where("id = ?", task.ID).Count(&taskCount).Error; err != nil {
		t.Fatalf("统计任务失败: %v", err)
	}
	if err := db.Model(&model.SkillDistributionRecord{}).Where("task_id = ?", task.ID).Count(&recordCount).Error; err != nil {
		t.Fatalf("统计记录失败: %v", err)
	}
	if taskCount != 0 || recordCount != 0 {
		t.Fatalf("清理后 task=%d record=%d，期望均为 0", taskCount, recordCount)
	}
}

func TestCreateSkillSelectAllTask_PublicUsesTargetVersion(t *testing.T) {
	setupSkillInstancesDB(t)
	db := model.DB(context.Background())
	user := model.User{Username: "select-all-public-version"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	instances := []model.Instance{
		{Name: "select-all-public-outdated", InstanceId: "ins-public-outdated", UserID: user.ID, AgentType: "openclaw"},
		{Name: "select-all-public-uninstalled", InstanceId: "ins-public-uninstalled", UserID: user.ID, AgentType: "openclaw"},
	}
	if err := db.Create(&instances).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	previousTask := model.SkillDistributionTask{
		Source: model.SkillSourcePublic, Slug: "public-version-skill", Version: "0.9.0",
		Status: model.TaskStatusCompleted, Type: model.TaskTypeDistribute, Total: 1, Success: 1,
	}
	if err := db.Create(&previousTask).Error; err != nil {
		t.Fatalf("创建历史任务失败: %v", err)
	}
	if err := db.Create(&model.SkillDistributionRecord{
		TaskID: previousTask.ID, InstanceID: instances[0].ID,
		InstanceCID: instances[0].InstanceId, Version: previousTask.Version,
		Status: model.RecordStatusSuccess, Type: model.TaskTypeDistribute,
	}).Error; err != nil {
		t.Fatalf("创建历史记录失败: %v", err)
	}

	task, total, err := createSkillSelectAllTask(
		context.Background(),
		skillTaskItem{
			Source: model.SkillSourcePublic, Slug: previousTask.Slug,
			Version: "1.0.0", DownloadURL: "https://example.invalid/skill.zip",
		},
		model.TaskTypeDistribute,
		0,
		"",
		distributionSelection{SelectAll: true, Statuses: []string{"outdated"}},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("createSkillSelectAllTask() error=%v", err)
	}
	if total != 1 || task.Total != 1 {
		t.Fatalf("total=%d task.total=%d，期望 1", total, task.Total)
	}
	var record model.SkillDistributionRecord
	if err := db.Where("task_id = ?", task.ID).First(&record).Error; err != nil {
		t.Fatalf("查询新记录失败: %v", err)
	}
	if record.InstanceID != instances[0].ID {
		t.Fatalf("新记录 instance_id=%d，期望 outdated 实例 %d", record.InstanceID, instances[0].ID)
	}
}
func TestCreateSkillSelectAllTask_SearchMatchesAdminInstancesFields(t *testing.T) {
	setupSkillInstancesDB(t)
	ctx := context.Background()
	db := model.DB(ctx)
	skill := model.Skill{
		Slug: "select-all-skill-search", Name: "搜索技能", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: model.VisibilityAll,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("创建技能失败: %v", err)
	}
	users := []model.User{
		{Username: "skill-search-needle"},
		{Username: "skill-search-other"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	instances := []model.Instance{
		{Name: "target-by-username", InstanceId: "ins-target-one", UserID: users[0].ID, Source: model.InstanceSourceLocal},
		{Name: "needle-target-by-name", InstanceId: "ins-target-two", UserID: users[1].ID, Source: model.InstanceSourceLocal},
		{Name: "target-by-instance-id", InstanceId: "ins-needle-three", UserID: users[1].ID, Source: model.InstanceSourceLocal},
		{Name: "unmatched-target", InstanceId: "ins-unmatched", UserID: users[1].ID, Source: model.InstanceSourceLocal},
	}
	if err := db.Create(&instances).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	task, total, err := createSkillSelectAllTask(
		ctx,
		skillTaskItem{
			Source: model.SkillSourceEnterprise, Slug: skill.Slug,
			Version: skill.Version, SkillID: skill.ID,
		},
		model.TaskTypeDistribute,
		0,
		"",
		distributionSelection{SelectAll: true, Statuses: []string{"uninstalled"}, Search: "needle"},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("createSkillSelectAllTask(search=%q) error = %v", "needle", err)
	}
	if total != 3 || task.Total != 3 {
		t.Fatalf("createSkillSelectAllTask(search=%q) total=%d task.total=%d, want 3", "needle", total, task.Total)
	}
	var records []model.SkillDistributionRecord
	if err := db.Where("task_id = ?", task.ID).Find(&records).Error; err != nil {
		t.Fatalf("查询搜索结果记录失败: %v", err)
	}
	wantIDs := map[uint]struct{}{
		instances[0].ID: {},
		instances[1].ID: {},
		instances[2].ID: {},
	}
	for _, record := range records {
		if _, ok := wantIDs[record.InstanceID]; !ok {
			t.Errorf("createSkillSelectAllTask(search=%q) returned unexpected instance_id=%d", "needle", record.InstanceID)
		}
		delete(wantIDs, record.InstanceID)
	}
	if len(wantIDs) != 0 {
		t.Errorf("createSkillSelectAllTask(search=%q) missing instance_ids=%v", "needle", wantIDs)
	}
}

func TestRunSkillSelectAllTask_RecordReadFailureConverges(t *testing.T) {
	db := setupSkillInstancesDB(t)
	task := model.SkillDistributionTask{
		Slug: "read-failed-select-all-skill", Version: "1.0.0",
		Status: model.TaskStatusRunning, Type: model.TaskTypeDistribute, Total: 1,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	record := model.SkillDistributionRecord{
		TaskID: task.ID, InstanceID: 1, InstanceCID: "ins-read-failed-skill",
		Version: task.Version, Status: model.RecordStatusPending, Type: model.TaskTypeDistribute,
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("创建记录失败: %v", err)
	}
	lock, err := model.AcquireLock(context.Background(), "test:read-failed-select-all-skill", time.Minute)
	if err != nil {
		t.Fatalf("获取测试锁失败: %v", err)
	}
	defer lock.Release()
	const callbackName = "test:fail_select_all_record_read"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "skill_distribution_records" {
			tx.AddError(errors.New("injected record read failure"))
		}
	}); err != nil {
		t.Fatalf("注册查询失败回调失败: %v", err)
	}

	runSkillSelectAllTask(
		context.Background(),
		skillTaskItem{Source: model.SkillSourceEnterprise, Slug: task.Slug, Version: task.Version},
		task,
	)
	if err := db.Callback().Query().Remove(callbackName); err != nil {
		t.Fatalf("移除查询失败回调失败: %v", err)
	}
	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatalf("查询任务失败: %v", err)
	}
	if err := db.First(&record, record.ID).Error; err != nil {
		t.Fatalf("查询记录失败: %v", err)
	}
	if task.Status != model.TaskStatusCompleted || task.Failed != 1 {
		t.Fatalf("任务状态=%q failed=%d，期望 completed/1", task.Status, task.Failed)
	}
	if record.Status != model.RecordStatusFailed {
		t.Fatalf("记录状态=%q，期望 failed", record.Status)
	}
}

func TestNormalizeSkillDistributionStatuses(t *testing.T) {
	got, err := normalizeSkillDistributionStatuses(nil)
	if err != nil {
		t.Fatalf("normalizeSkillDistributionStatuses(nil) error = %v", err)
	}
	set := make(map[string]struct{}, len(got))
	for _, status := range got {
		set[status] = struct{}{}
	}
	if _, ok := set["installed"]; !ok {
		t.Error("normalizeSkillDistributionStatuses(nil) must include installed")
	}
	for _, transitional := range []string{"installing", "uninstalling"} {
		if _, ok := set[transitional]; ok {
			t.Errorf("normalizeSkillDistributionStatuses(nil) contains transitional status %q", transitional)
		}
	}
	if _, err := normalizeSkillDistributionStatuses([]string{"installing"}); err == nil {
		t.Error("normalizeSkillDistributionStatuses([installing]) error = nil, want error")
	}
}

func TestCreateSkillSelectAllTask_HasNoExplicitIDLimit(t *testing.T) {
	setupSkillInstancesDB(t)
	ctx := context.Background()
	db := model.DB(ctx)
	skill := model.Skill{
		Slug: "select-all-unlimited", Name: "不限量技能", Version: "1.0.0",
		VersionMajor: 1, VisibilityType: model.VisibilityAll,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}
	user := model.User{Username: "select-all-unlimited"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	const targetCount = 501
	instances := make([]model.Instance, targetCount)
	for i := range instances {
		instances[i] = model.Instance{
			Name:       "unlimited",
			InstanceId: "ins-unlimited-" + uintStr(uint(i+1)),
			UserID:     user.ID,
			Source:     model.InstanceSourceLocal,
			AgentType:  "unsupported",
		}
	}
	if err := db.CreateInBatches(&instances, skillDistributionBatchSize).Error; err != nil {
		t.Fatalf("create instances: %v", err)
	}

	task, total, err := createSkillSelectAllTask(
		ctx,
		skillTaskItem{
			Source:  model.SkillSourceEnterprise,
			Slug:    skill.Slug,
			Version: skill.Version,
			SkillID: skill.ID,
		},
		model.TaskTypeDistribute,
		0,
		"",
		distributionSelection{SelectAll: true, Statuses: []string{"uninstalled"}},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("createSkillSelectAllTask() error = %v", err)
	}
	if total != targetCount || task.Total != targetCount {
		t.Fatalf("total=%d task.total=%d, want %d", total, task.Total, targetCount)
	}
	var recordCount int64
	if err := db.Model(&model.SkillDistributionRecord{}).Where("task_id = ?", task.ID).Count(&recordCount).Error; err != nil {
		t.Fatalf("count records: %v", err)
	}
	if recordCount != targetCount {
		t.Fatalf("record count=%d, want %d", recordCount, targetCount)
	}
}

func TestRunSkillSelectAllTask_RecoversTaskPanic(t *testing.T) {
	db := setupSkillInstancesDB(t)
	ctx := context.Background()
	task := model.SkillDistributionTask{
		Slug: "panic-skill", Version: "1.0.0", Total: 1,
		Status: model.TaskStatusRunning, Type: model.TaskTypeDistribute,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	record := model.SkillDistributionRecord{
		TaskID: task.ID, InstanceID: 1, InstanceCID: "ins-panic-skill",
		Version: task.Version, Status: model.RecordStatusPending, Type: model.TaskTypeDistribute,
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("创建记录失败: %v", err)
	}

	triggered := false
	const callbackName = "test:skill-select-all-query-panic"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if !triggered && tx.Statement.Table == "skill_distribution_records" {
			triggered = true
			panic("skill select-all query panic")
		}
	}); err != nil {
		t.Fatalf("注册 panic callback 失败: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(callbackName)
	})

	func() {
		defer recoverSkillSelectAllTaskPanic(ctx, task)
		runSkillSelectAllTask(
			ctx,
			skillTaskItem{Source: model.SkillSourceEnterprise, Slug: task.Slug, Version: task.Version},
			task,
		)
	}()

	if !triggered {
		t.Fatal("panic callback 未触发")
	}
	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatalf("查询任务失败: %v", err)
	}
	if err := db.First(&record, record.ID).Error; err != nil {
		t.Fatalf("查询记录失败: %v", err)
	}
	if task.Status != model.TaskStatusCompleted || task.Failed != 1 {
		t.Fatalf("panic 后任务 status=%q failed=%d，期望 completed/1", task.Status, task.Failed)
	}
	if record.Status != model.RecordStatusFailed {
		t.Fatalf("panic 后记录 status=%q，期望 failed", record.Status)
	}
}
