package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hatchery/model"
)

// seedLocalRuleAckCase 创建 user/instance/enterprise_rule/task/record 一套，
// 并返回 record.ID 与 task.ID。rule 路径走独立表，区别于 skill 路径。
func seedLocalRuleAckCase(t *testing.T, username, slug, ruleType string, taskType string) (uint, uint) {
	t.Helper()
	ctx := context.Background()
	// rule 路径表：setupSkillInstancesDB 默认不含，本文件专补。
	if err := model.DB(ctx).AutoMigrate(
		&model.LocalInstanceRule{},
		&model.RuleDistributionRecord{},
		&model.RuleDistributionTask{},
		&model.EnterpriseRule{},
	); err != nil {
		t.Fatalf("migrate rule tables: %v", err)
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
	rule := model.EnterpriseRule{
		Slug: slug, Name: slug, Version: "1.0.0",
		Type: ruleType, COSKey: "cos/" + slug + ".zip",
	}
	if err := model.DB(ctx).Create(&rule).Error; err != nil {
		t.Fatalf("create rule: %v", err)
	}
	task := model.RuleDistributionTask{
		RuleID: rule.ID, Slug: slug, Version: rule.Version,
		Total: 1, Status: model.TaskStatusRunning, RuleType: ruleType,
		Type: taskType,
	}
	if err := model.DB(ctx).Create(&task).Error; err != nil {
		t.Fatalf("create rule task: %v", err)
	}
	rec := model.RuleDistributionRecord{
		TaskID: task.ID, RuleID: rule.ID,
		InstanceID: inst.ID, InstanceCID: inst.InstanceId,
		Version: rule.Version, Status: model.RuleRecordStatusPending, Type: taskType,
	}
	if err := model.DB(ctx).Create(&rec).Error; err != nil {
		t.Fatalf("create rule rec: %v", err)
	}
	return rec.ID, task.ID
}

// ruleAckReq 构造 rule 路径的 ack 请求：type 必须为 rule 命令名（install_prompt_rule 等）。
func ruleAckReq(t *testing.T, recordID uint, username, ackType, status string) *http.Request {
	t.Helper()
	body := map[string]any{
		"type":   ackType,
		"status": status,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("ruleAckReq encode: %v", err)
	}
	// jsonAckReq 会把 recordID 注入 body.id 并带登录态。
	return jsonAckReq(t, recordID, username, string(encoded))
}

// TestHandleLocalAgentRuleAck_InstallSuccess_UpsertAndCount
// rule 路径 install 成功 → local_instance_rules 落 distributed + task.success+1。
func TestHandleLocalAgentRuleAck_InstallSuccess_UpsertAndCount(t *testing.T) {
	setupSkillInstancesDB(t)
	recID, taskID := seedLocalRuleAckCase(t, "rule-ok", "rule-ok-1", model.EnterpriseRuleTypePrompt, model.RuleTaskTypeDistribute)

	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, ruleAckReq(t, recID, "rule-ok", model.CommandTypeInstallPromptRule, model.RuleRecordStatusSuccess))
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	ctx := context.Background()
	var lir model.LocalInstanceRule
	if err := model.DB(ctx).Where("slug = ?", "rule-ok-1").First(&lir).Error; err != nil {
		t.Fatalf("local_instance_rules 未落库: %v", err)
	}
	if lir.InstallStatus != model.LocalSkillInstallStatusDistributed {
		t.Errorf("期望 install_status=distributed，实际=%s", lir.InstallStatus)
	}
	if lir.Source != model.LocalRuleSourceEnterprise {
		t.Errorf("期望 source=enterprise，实际=%s", lir.Source)
	}
	var task model.RuleDistributionTask
	if err := model.DB(ctx).First(&task, taskID).Error; err != nil {
		t.Fatalf("查 task 失败: %v", err)
	}
	if task.Success != 1 {
		t.Errorf("期望 task.success=1，实际=%d", task.Success)
	}
	if task.Status != model.TaskStatusCompleted {
		t.Errorf("期望 task 已完成，实际=%s", task.Status)
	}
}

// TestHandleLocalAgentRuleAck_UninstallSuccess_DeletesAndCounts
// rule 路径 uninstall 成功 → local_instance_rules 被删 + task 计数。
func TestHandleLocalAgentRuleAck_UninstallSuccess_DeletesAndCounts(t *testing.T) {
	setupSkillInstancesDB(t)
	recID, taskID := seedLocalRuleAckCase(t, "rule-un", "rule-un-1", model.EnterpriseRuleTypeRule, model.RuleTaskTypeUninstall)

	ctx := context.Background()
	// 预置一条 distributing 的 local_instance_rule，uninstall 应删掉它。
	model.DB(ctx).Create(&model.LocalInstanceRule{
		InstanceID: 1, Slug: "rule-un-1", Version: "1.0.0",
		RuleType: model.EnterpriseRuleTypeRule, Source: model.LocalRuleSourceEnterprise,
		Scope: model.LocalSkillScopeUser, InstallStatus: model.LocalSkillInstallStatusDistributing,
	})
	// recID 对应的 instance 是 setupSkillInstancesDB 创建的，需要把预置行绑到正确 instance。
	// 重建：用真实 inst 查 ID。
	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "local-rule-un-001").First(&inst)
	model.DB(ctx).Where("slug = ?", "rule-un-1").Delete(&model.LocalInstanceRule{})
	model.DB(ctx).Create(&model.LocalInstanceRule{
		InstanceID: inst.ID, Slug: "rule-un-1", Version: "1.0.0",
		RuleType: model.EnterpriseRuleTypeRule, Source: model.LocalRuleSourceEnterprise,
		Scope: model.LocalSkillScopeUser, InstallStatus: model.LocalSkillInstallStatusDistributing,
	})

	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, ruleAckReq(t, recID, "rule-un", model.CommandTypeUninstallRuleRule, model.RuleRecordStatusSuccess))
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var count int64
	model.DB(ctx).Model(&model.LocalInstanceRule{}).Where("instance_id = ? AND slug = ?", inst.ID, "rule-un-1").Count(&count)
	if count != 0 {
		t.Errorf("uninstall 成功应删除 local_instance_rule，实际剩余 %d 条", count)
	}
	var task model.RuleDistributionTask
	model.DB(ctx).First(&task, taskID)
	if task.Success != 1 {
		t.Errorf("期望 task.success=1，实际=%d", task.Success)
	}
}

// TestHandleLocalAgentRuleAck_Failed_MarksLocalRuleFailed
// rule 路径 ack 失败 → local_instance_rules.install_status=failed。
func TestHandleLocalAgentRuleAck_Failed_MarksLocalRuleFailed(t *testing.T) {
	setupSkillInstancesDB(t)
	recID, _ := seedLocalRuleAckCase(t, "rule-fail", "rule-fail-1", model.EnterpriseRuleTypePrompt, model.RuleTaskTypeDistribute)

	ctx := context.Background()
	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "local-rule-fail-001").First(&inst)
	model.DB(ctx).Create(&model.LocalInstanceRule{
		InstanceID: inst.ID, Slug: "rule-fail-1", Version: "1.0.0",
		RuleType: model.EnterpriseRuleTypePrompt, Source: model.LocalRuleSourceEnterprise,
		Scope: model.LocalSkillScopeUser, InstallStatus: model.LocalSkillInstallStatusDistributing,
	})

	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, ruleAckReq(t, recID, "rule-fail", model.CommandTypeInstallPromptRule, model.RuleRecordStatusFailed))
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var lir model.LocalInstanceRule
	model.DB(ctx).Where("instance_id = ? AND slug = ?", inst.ID, "rule-fail-1").First(&lir)
	if lir.InstallStatus != model.LocalSkillInstallStatusFailed {
		t.Errorf("期望 install_status=failed，实际=%s", lir.InstallStatus)
	}
}

// TestHandleLocalAgentRuleAck_Idempotent_FailedToFailedNoChange
// 已 failed 的 record 再收到 failed → 幂等，不改 local_instance_rules（仍 failed）。
func TestHandleLocalAgentRuleAck_Idempotent_FailedToFailedNoChange(t *testing.T) {
	setupSkillInstancesDB(t)
	recID, _ := seedLocalRuleAckCase(t, "rule-idem", "rule-idem-1", model.EnterpriseRuleTypePrompt, model.RuleTaskTypeDistribute)

	ctx := context.Background()
	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "local-rule-idem-001").First(&inst)
	model.DB(ctx).Create(&model.LocalInstanceRule{
		InstanceID: inst.ID, Slug: "rule-idem-1", Version: "1.0.0",
		RuleType: model.EnterpriseRuleTypePrompt, Source: model.LocalRuleSourceEnterprise,
		Scope: model.LocalSkillScopeUser, InstallStatus: model.LocalSkillInstallStatusDistributing,
	})

	// 第一次 failed
	HandleLocalAgentAck(httptest.NewRecorder(), ruleAckReq(t, recID, "rule-idem", model.CommandTypeInstallPromptRule, model.RuleRecordStatusFailed))
	// 第二次 failed（幂等）
	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, ruleAckReq(t, recID, "rule-idem", model.CommandTypeInstallPromptRule, model.RuleRecordStatusFailed))
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var lir model.LocalInstanceRule
	model.DB(ctx).Where("instance_id = ? AND slug = ?", inst.ID, "rule-idem-1").First(&lir)
	if lir.InstallStatus != model.LocalSkillInstallStatusFailed {
		t.Errorf("failed→failed 应保持 failed，实际=%s", lir.InstallStatus)
	}
}

// TestHandleLocalAgentRuleAck_NotFound
// record 不存在 → 404（inst 不属于当前用户同样触发 RecordNotFound 分支）。
func TestHandleLocalAgentRuleAck_NotFound(t *testing.T) {
	setupSkillInstancesDB(t)
	seedLocalRuleAckCase(t, "rule-nf", "rule-nf-1", model.EnterpriseRuleTypePrompt, model.RuleTaskTypeDistribute)

	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, ruleAckReq(t, 999999, "rule-nf", model.CommandTypeInstallPromptRule, model.RuleRecordStatusSuccess))
	if rr.Code != http.StatusNotFound {
		t.Errorf("期望 404，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleLocalAgentRuleAck_InvalidType_Rejected
// type 不是 rule 命令名 → 走 Handler 的 type 校验分支返回 400。
func TestHandleLocalAgentRuleAck_InvalidType_Rejected(t *testing.T) {
	setupSkillInstancesDB(t)
	recID, _ := seedLocalRuleAckCase(t, "rule-badtype", "rule-badtype-1", model.EnterpriseRuleTypePrompt, model.RuleTaskTypeDistribute)

	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, ruleAckReq(t, recID, "rule-badtype", "not_a_valid_type", model.RuleRecordStatusSuccess))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleLocalAgentRuleAck_UnknownRecordID_404 确保非 nil recordID 但查不到也走 404。
func TestHandleLocalAgentRuleAck_UnknownRecordID_404(t *testing.T) {
	setupSkillInstancesDB(t)
	seedLocalRuleAckCase(t, "rule-unknown", "rule-unknown-1", model.EnterpriseRuleTypePrompt, model.RuleTaskTypeDistribute)

	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, ruleAckReq(t, 123456, "rule-unknown", model.CommandTypeInstallPromptRule, model.RuleRecordStatusSuccess))
	if rr.Code != http.StatusNotFound {
		t.Errorf("期望 404，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestQueryRuleCommands_PendingReturnsCommand
// queryRuleCommands 应返回 pending 的 rule record，组装出带 scope/downloadURL 的 command。
func TestQueryRuleCommands_PendingReturnsCommand(t *testing.T) {
	setupSkillInstancesDB(t)
	recID, _ := seedLocalRuleAckCase(t, "rule-q", "rule-q-1", model.EnterpriseRuleTypeRule, model.RuleTaskTypeDistribute)

	ctx := context.Background()
	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "local-rule-q-001").First(&inst)
	// 预置一条 distributing 的 local_instance_rule，验证 scope/workspace_path 反查。
	model.DB(ctx).Create(&model.LocalInstanceRule{
		InstanceID: inst.ID, Slug: "rule-q-1", Version: "1.0.0",
		RuleType: model.EnterpriseRuleTypeRule, Source: model.LocalRuleSourceEnterprise,
		Scope: model.LocalSkillScopeUser, WorkspacePath: "", InstallStatus: model.LocalSkillInstallStatusDistributing,
	})

	cmds, err := queryRuleCommands(ctx, &inst)
	if err != nil {
		t.Fatalf("queryRuleCommands 失败: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("期望 1 条 rule command，实际=%d", len(cmds))
	}
	cmd := cmds[0]
	if cmd.ID != recID {
		t.Errorf("command.id 期望 %d，实际=%d", recID, cmd.ID)
	}
	if cmd.RuleSlug != "rule-q-1" {
		t.Errorf("rule_slug 期望 rule-q-1，实际=%s", cmd.RuleSlug)
	}
	if cmd.Type != model.CommandTypeInstallRuleRule {
		t.Errorf("type 期望 install_rule_rule，实际=%s", cmd.Type)
	}
	if cmd.Scope != model.LocalSkillScopeUser {
		t.Errorf("scope 期望 user（反查 local_instance_rules），实际=%s", cmd.Scope)
	}
	// downloadURL 依赖 SMH 配置，测试环境生成可能失败被静默跳过，不强制断言非空。
	_ = cmd.DownloadURL
}

// TestQueryRuleCommands_NotPendingExcluded
// 已 success 的 rule record 不应出现在 sync 命令里。
func TestQueryRuleCommands_NotPendingExcluded(t *testing.T) {
	setupSkillInstancesDB(t)
	recID, _ := seedLocalRuleAckCase(t, "rule-q2", "rule-q2-1", model.EnterpriseRuleTypePrompt, model.RuleTaskTypeDistribute)

	ctx := context.Background()
	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "local-rule-q2-001").First(&inst)
	// 把 record 改成 success，sync 不应再返回。
	model.DB(ctx).Model(&model.RuleDistributionRecord{}).Where("id = ?", recID).Update("status", model.RuleRecordStatusSuccess)

	cmds, err := queryRuleCommands(ctx, &inst)
	if err != nil {
		t.Fatalf("queryRuleCommands 失败: %v", err)
	}
	if len(cmds) != 0 {
		t.Errorf("success record 不应出现在 sync 命令，实际 %d 条", len(cmds))
	}
}

// TestQueryRuleCommands_LegacyRecordUsesUserScope 确保未关联二期 scope 行的历史任务
// 不会把已废弃的 instance scope 下发给 TeamAI。
func TestQueryRuleCommands_LegacyRecordUsesUserScope(t *testing.T) {
	setupSkillInstancesDB(t)
	seedLocalRuleAckCase(t, "rule-legacy-scope", "rule-legacy-scope-1", model.EnterpriseRuleTypeRule, model.RuleTaskTypeDistribute)

	ctx := context.Background()
	var inst model.Instance
	if err := model.DB(ctx).Where("instance_id = ?", "local-rule-legacy-scope-001").First(&inst).Error; err != nil {
		t.Fatalf("查询实例失败: %v", err)
	}
	cmds, err := queryRuleCommands(ctx, &inst)
	if err != nil {
		t.Fatalf("queryRuleCommands 失败: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("期望 1 条历史 rule command，实际=%d", len(cmds))
	}
	if cmds[0].Scope != model.LocalSkillScopeUser || cmds[0].WorkspacePath != "" {
		t.Errorf("历史 rule command 应兼容为 user scope，实际=%+v", cmds[0])
	}
}

func TestShouldApplyLocalAgentRuleAck_AllStatuses(t *testing.T) {
	tests := []struct {
		current string
		ack     string
		want    bool
	}{
		{model.RuleRecordStatusPending, model.RuleRecordStatusSuccess, true},
		{model.RuleRecordStatusPending, model.RuleRecordStatusFailed, true},
		{model.RuleRecordStatusFailed, model.RuleRecordStatusSuccess, true},
		{model.RuleRecordStatusFailed, model.RuleRecordStatusFailed, false},
		{model.RuleRecordStatusSuccess, model.RuleRecordStatusSuccess, false},
	}
	for _, tt := range tests {
		if got := shouldApplyLocalAgentRuleAck(tt.current, tt.ack); got != tt.want {
			t.Fatalf("shouldApplyLocalAgentRuleAck(%q, %q)=%v want=%v", tt.current, tt.ack, got, tt.want)
		}
	}
}

func TestLocalAgentRuleAckHelpers_NoOpAndExistingUpdate(t *testing.T) {
	setupSkillInstancesDB(t)
	seedLocalRuleAckCase(t, "rule-helper", "rule-helper-1", model.EnterpriseRuleTypeRule, model.RuleTaskTypeDistribute)
	db := model.DB(t.Context())

	state := localAgentRuleAckState{now: time.Now()}
	if err := updateLocalAgentRuleAckTask(db, &state, model.RuleRecordStatusSuccess); err != nil {
		t.Fatalf("taskID=0 should be a no-op: %v", err)
	}
	if err := resolveLocalAgentRuleAckTarget(db, &state); err != nil {
		t.Fatalf("empty target should be a no-op: %v", err)
	}
	for _, status := range []string{model.RuleRecordStatusFailed, model.RuleRecordStatusSuccess, "ignored"} {
		state.record.Type = model.RuleTaskTypeUninstall
		if err := applyLocalAgentRuleAck(db, &state, status, ""); err != nil {
			t.Fatalf("empty target status=%q: %v", status, err)
		}
	}

	var inst model.Instance
	if err := db.Where("instance_id = ?", "local-rule-helper-001").First(&inst).Error; err != nil {
		t.Fatalf("load instance: %v", err)
	}
	localRule := model.LocalInstanceRule{
		InstanceID: inst.ID, Slug: "rule-helper-1", Version: "0.9.0",
		RuleType: model.EnterpriseRuleTypeRule, Source: model.LocalRuleSourceEnterprise,
		Scope: model.LocalSkillScopeUser, InstallStatus: model.LocalSkillInstallStatusDistributing,
	}
	if err := db.Create(&localRule).Error; err != nil {
		t.Fatalf("create local rule: %v", err)
	}
	state = localAgentRuleAckState{
		instance: inst, ruleSlug: localRule.Slug, localRuleID: localRule.ID,
		scope: model.LocalSkillScopeUser, now: time.Now(),
		record: model.RuleDistributionRecord{Version: "1.0.0"},
		rule:   model.EnterpriseRule{Type: model.EnterpriseRuleTypeRule},
	}
	if err := upsertLocalInstanceRuleFromAck(db, &state, " 2.0.0 "); err != nil {
		t.Fatalf("update existing local rule: %v", err)
	}
	if err := db.First(&localRule, localRule.ID).Error; err != nil {
		t.Fatalf("reload local rule: %v", err)
	}
	if localRule.Version != "2.0.0" || localRule.DisplayName != "rule-helper-1" || localRule.InstallStatus != model.LocalSkillInstallStatusDistributed {
		t.Fatalf("updated local rule=%#v", localRule)
	}
}

func TestHandleLocalAgentRuleAck_WrongOwnerAndDBError(t *testing.T) {
	setupSkillInstancesDB(t)
	recordID, _ := seedLocalRuleAckCase(t, "rule-owner", "rule-owner-1", model.EnterpriseRuleTypePrompt, model.RuleTaskTypeDistribute)
	other := model.User{Username: "rule-other", Role: "user"}
	if err := model.DB(t.Context()).Create(&other).Error; err != nil {
		t.Fatalf("create other user: %v", err)
	}

	rr := httptest.NewRecorder()
	HandleLocalAgentAck(rr, ruleAckReq(t, recordID, other.Username, model.CommandTypeInstallPromptRule, model.RuleRecordStatusSuccess))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("wrong owner status=%d body=%s", rr.Code, rr.Body.String())
	}

	req := ruleAckReq(t, recordID, "rule-owner", model.CommandTypeInstallPromptRule, model.RuleRecordStatusSuccess)
	if err := model.DB(t.Context()).Migrator().DropTable(&model.RuleDistributionRecord{}); err != nil {
		t.Fatalf("drop record table: %v", err)
	}
	rr = httptest.NewRecorder()
	HandleLocalAgentAck(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("DB error status=%d body=%s", rr.Code, rr.Body.String())
	}
}
