package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"
)

// TestHandleLocalAgentSync_CmdsListPresent 验证三期 sync 响应同时包含 commands（老）
// 与 cmds（新统一字段）两个列表，且 uninstall_teamai 任务同时出现在两个列表。
func TestHandleLocalAgentSync_CmdsListPresent(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	// 实例 InstanceId 必须用 formatLocalInstanceID 派生，sync 才能查到
	localAgentID := "0123456789abcdef"
	instCID := formatLocalInstanceID("codebuddy", localAgentID)
	user, inst := seedLocalUserAndInstanceWithCID(t, "alice", "codebuddy", instCID)
	_ = user
	instID := inst.ID
	cookie := loginCookie(t, "alice")

	// 创建一条 pending 的 uninstall_teamai 本地任务
	ctx := context.Background()
	task := model.LocalAgentTask{
		Identifier:  "test-tenant",
		InstanceID:  instID,
		InstanceCID: instCID,
		Type:        model.LocalAgentTaskTypeUninstallTeamai,
		Cmd:         "teamai uninstall --force --agent codebuddy",
		Status:      model.LocalAgentTaskStatusPending,
		OperatorID:  0,
	}
	if err := model.DB(ctx).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	// 需要 LocalInstanceInfo 心跳行（sync 会更新 last_report_at，但查不到不致命）
	now := time.Now()
	model.DB(ctx).Create(&model.LocalInstanceInfo{InstanceID: instID, LastReportAt: &now})

	req := httptest.NewRequest(http.MethodPost, "/local-agent/sync",
		strings.NewReader(`{"agent_type":"codebuddy","local_agent_id":"0123456789abcdef","status":"running"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rr := httptest.NewRecorder()
	HandleLocalAgentSync(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Commands []map[string]any `json:"commands"`
		Cmds     []map[string]any `json:"cmds"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if len(resp.Commands) == 0 || len(resp.Cmds) == 0 {
		t.Fatalf("commands/cmds 不应为空: commands=%d cmds=%d", len(resp.Commands), len(resp.Cmds))
	}

	// uninstall_teamai 必须同时出现在两个列表
	findTask := func(list []map[string]any) map[string]any {
		for _, c := range list {
			if c["type"] == model.LocalAgentTaskTypeUninstallTeamai {
				return c
			}
		}
		return nil
	}
	legacy := findTask(resp.Commands)
	unified := findTask(resp.Cmds)
	if legacy == nil {
		t.Fatalf("commands 列表缺少 uninstall_teamai: %+v", resp.Commands)
	}
	if unified == nil {
		t.Fatalf("cmds 列表缺少 uninstall_teamai: %+v", resp.Cmds)
	}

	// 两个列表里该任务的 cmd 字段必须一致
	if legacy["cmd"] != unified["cmd"] {
		t.Fatalf("cmd 不一致: legacy=%v unified=%v", legacy["cmd"], unified["cmd"])
	}
	// 统一列表用 slug 字段（omitempty，uninstall_teamai 无 slug → 不存在）
	if _, ok := unified["slug"]; ok {
		t.Fatalf("uninstall_teamai 不应有 slug 字段: %+v", unified)
	}
	// 统一列表字段名是 slug 而非 rule_slug/skill_slug
	for _, c := range resp.Cmds {
		if _, ok := c["rule_slug"]; ok {
			t.Fatalf("cmds 列表不应出现 rule_slug: %+v", c)
		}
		if _, ok := c["skill_slug"]; ok {
			t.Fatalf("cmds 列表不应出现 skill_slug: %+v", c)
		}
	}
}

// TestHandleLocalAgentSync_CmdsEmptyWhenNoTasks 无任务时 cmds 为空数组（不报错）。
func TestHandleLocalAgentSync_CmdsEmptyWhenNoTasks(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	localAgentID := "0123456789abcdef"
	instCID := formatLocalInstanceID("codebuddy", localAgentID)
	seedLocalUserAndInstanceWithCID(t, "alice", "codebuddy", instCID)
	cookie := loginCookie(t, "alice")

	req := httptest.NewRequest(http.MethodPost, "/local-agent/sync",
		strings.NewReader(`{"agent_type":"codebuddy","local_agent_id":"0123456789abcdef","status":"running"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rr := httptest.NewRecorder()
	HandleLocalAgentSync(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Cmds []map[string]any `json:"cmds"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Cmds == nil {
		t.Fatalf("cmds 应为空数组而非 nil")
	}
}

// TestHandleLocalAgentSync_HookCmd_CmdsHasEventAndCmd 验证 hook 类型 rule 在 cmds（统一字段）
// 列表里也返回 event + cmd，与 commands（老字段）列表保持一致。回归守护：之前映射漏传了
// Event/Cmd，导致 hook 项 cmds 比 commands 少字段。
func TestHandleLocalAgentSync_HookCmd_CmdsHasEventAndCmd(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	localAgentID := "0123456789abcdef"
	instCID := formatLocalInstanceID("codebuddy", localAgentID)
	seedLocalUserAndInstanceWithCID(t, "alice", "codebuddy", instCID)
	cookie := loginCookie(t, "alice")
	ctx := context.Background()

	rule := model.EnterpriseRule{
		Slug: "clawpro-knowledge-sync", Name: "clawpro-knowledge-sync", Version: "1.0.1",
		Type:   model.EnterpriseRuleTypeHook,
		COSKey: "cos/clawpro-knowledge-sync.zip",
		Event:  "UserPromptSubmit",
		Cmd:    "python3 \"$CODEBUDDY_PROJECT_DIR/.codebuddy/hooks/clawpro_knowledge_sync.py",
	}
	if err := model.DB(ctx).Create(&rule).Error; err != nil {
		t.Fatalf("create rule: %v", err)
	}
	task := model.RuleDistributionTask{
		RuleID: rule.ID, Slug: rule.Slug, Version: rule.Version,
		Total: 1, Status: model.TaskStatusRunning, RuleType: model.EnterpriseRuleTypeHook,
		Type: model.RuleTaskTypeDistribute,
	}
	if err := model.DB(ctx).Create(&task).Error; err != nil {
		t.Fatalf("create rule task: %v", err)
	}
	rec := model.RuleDistributionRecord{
		TaskID: task.ID, RuleID: rule.ID,
		InstanceID: mustInstIDByCID(t, instCID), InstanceCID: instCID,
		Version: rule.Version, Status: model.RuleRecordStatusPending, Type: model.RuleTaskTypeDistribute,
	}
	if err := model.DB(ctx).Create(&rec).Error; err != nil {
		t.Fatalf("create rule rec: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/local-agent/sync",
		strings.NewReader(`{"agent_type":"codebuddy","local_agent_id":"0123456789abcdef","status":"running"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	rr := httptest.NewRecorder()
	HandleLocalAgentSync(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Commands []map[string]any `json:"commands"`
		Cmds     []map[string]any `json:"cmds"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}

	findHook := func(list []map[string]any) map[string]any {
		for _, c := range list {
			if c["type"] == model.CommandTypeInstallHookRule {
				return c
			}
		}
		return nil
	}
	legacy := findHook(resp.Commands)
	unified := findHook(resp.Cmds)
	if legacy == nil {
		t.Fatalf("commands 缺少 hook 项: %+v", resp.Commands)
	}
	if unified == nil {
		t.Fatalf("cmds 缺少 hook 项: %+v", resp.Cmds)
	}
	// cmds 必须含 event + cmd，且与 commands 一致
	if unified["event"] != legacy["event"] {
		t.Fatalf("cmds.event 缺失或不一致: cmds=%v commands=%v", unified["event"], legacy["event"])
	}
	if unified["cmd"] != legacy["cmd"] {
		t.Fatalf("cmds.cmd 缺失或不一致: cmds=%v commands=%v", unified["cmd"], legacy["cmd"])
	}
}

func mustInstIDByCID(t *testing.T, instCID string) uint {
	t.Helper()
	ctx := context.Background()
	var inst model.Instance
	if err := model.DB(ctx).Where("instance_id = ?", instCID).First(&inst).Error; err != nil {
		t.Fatalf("查实例 by cid: %v", err)
	}
	return inst.ID
}
