package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hatchery/model"

	"gorm.io/gorm"
)

// ── HTTP handler test helpers ──────────────────────────────────────────────────

// covAdminReq 构造一个带 admin-token 的 HTTP 请求。
func covAdminReq(method, url string, body interface{}) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// covUserReq 构造一个带 session 登录（指定 username）的 HTTP 请求。
func covUserReq(t *testing.T, method, url string, body interface{}, username string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, url, &buf)
	// 通过 CookieStore 注入 session
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username
	rec := httptest.NewRecorder()
	if err := session.Save(req, rec); err != nil {
		t.Fatalf("save session: %v", err)
	}
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

func decodeJSONBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v, body=%s", err, rec.Body.String())
	}
	return out
}

// ── stale_instances_apply.go: scenario B/C/D + handover error paths + helpers ──

func TestApplyEngine_Migrate_ScenarioB(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	// user 10 has NO groups → scenario B
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, nil, map[uint]bool{10: true})
	// target_group_id=0 → success (回退到未分组)
	engine.run([]applyActionItem{
		{Action: StaleActionMigrate, InstanceIDs: []uint{inst.ID}, TargetGroupID: uintPtr(0)},
	})
	r := engine.results[0]
	if r.Status != "success" {
		t.Errorf("status: got %q want success, err=%s", r.Status, r.Error)
	}
	var updated model.Instance
	model.DB(ctx).First(&updated, inst.ID)
	if updated.GroupID != 0 {
		t.Errorf("group_id: got %d want 0", updated.GroupID)
	}
}

func TestApplyEngine_Migrate_ScenarioB_TargetNotZero(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{7: true}, map[uint]bool{10: true})
	// target_group_id != 0 but user has no groups → fail
	engine.run([]applyActionItem{
		{Action: StaleActionMigrate, InstanceIDs: []uint{inst.ID}, TargetGroupID: uintPtr(7)},
	})
	r := engine.results[0]
	if r.Status != "failed" {
		t.Errorf("status: got %q want failed", r.Status)
	}
}

func TestApplyEngine_Migrate_ScenarioC(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 7, 10) // user in group 7
	inst := covSeedInstance(t, 1, 10, 0, "agent-1", "ins-1") // instance ungrouped

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{7: true}, map[uint]bool{10: true})
	engine.run([]applyActionItem{
		{Action: StaleActionMigrate, InstanceIDs: []uint{inst.ID}, TargetGroupID: uintPtr(7)},
	})
	r := engine.results[0]
	if r.Status != "success" {
		t.Errorf("status: got %q want success, err=%s", r.Status, r.Error)
	}
	var updated model.Instance
	model.DB(ctx).First(&updated, inst.ID)
	if updated.GroupID != 7 {
		t.Errorf("group_id: got %d want 7", updated.GroupID)
	}
}

func TestApplyEngine_Migrate_ScenarioC_TargetZeroFails(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 7, 10)
	inst := covSeedInstance(t, 1, 10, 0, "agent-1", "ins-1")

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, nil, map[uint]bool{10: true})
	// scenario C: target_group_id=0 → fail (must specify a real group)
	engine.run([]applyActionItem{
		{Action: StaleActionMigrate, InstanceIDs: []uint{inst.ID}, TargetGroupID: uintPtr(0)},
	})
	r := engine.results[0]
	if r.Status != "failed" {
		t.Errorf("status: got %q want failed", r.Status)
	}
}

func TestApplyEngine_Migrate_ScenarioA_TargetNotInUserGroups(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 7, 10) // user in group 7
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true, 7: true}, map[uint]bool{10: true})
	// target=5 but user is in group 7 → fail
	engine.run([]applyActionItem{
		{Action: StaleActionMigrate, InstanceIDs: []uint{inst.ID}, TargetGroupID: uintPtr(5)},
	})
	r := engine.results[0]
	if r.Status != "failed" {
		t.Errorf("status: got %q want failed", r.Status)
	}
}

func TestApplyEngine_Migrate_ScenarioD(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	engine := newApplyEngine(ctx, 99, TriggerSourceGroupParentChange, map[uint]bool{5: true}, map[uint]bool{10: true})
	engine.run([]applyActionItem{
		{Action: StaleActionMigrate, InstanceIDs: []uint{inst.ID}, TargetGroupID: uintPtr(5)},
	})
	r := engine.results[0]
	if r.Status != "success" {
		t.Errorf("status: got %q want success, err=%s", r.Status, r.Error)
	}
	// scenario D: group_id stays unchanged
	var updated model.Instance
	model.DB(ctx).First(&updated, inst.ID)
	if updated.GroupID != 5 {
		t.Errorf("group_id: got %d want 5 (D should not change)", updated.GroupID)
	}
}

func TestApplyEngine_Migrate_ListPageFollowup_Noop(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10) // user still in group 5 → normally noop
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	// list_page_followup bypasses noop check for migrate
	engine := newApplyEngine(ctx, 99, TriggerSourceListPageFollowup, map[uint]bool{5: true}, map[uint]bool{10: true})
	engine.run([]applyActionItem{
		{Action: StaleActionMigrate, InstanceIDs: []uint{inst.ID}, TargetGroupID: uintPtr(5)},
	})
	r := engine.results[0]
	if r.Status != "success" {
		t.Errorf("status: got %q want success (list_page_followup bypasses noop), err=%s", r.Status, r.Error)
	}
}

func TestApplyEngine_Handover_TargetNotInSameGroup(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 5, 10) // alice in group 5
	covSeedMember(t, 7, 20) // bob in group 7, NOT in group 5
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	// same-group handover (no target_group_id specified), bob not in group 5 → fail
	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true, 7: true}, map[uint]bool{10: true, 20: true})
	engine.run([]applyActionItem{
		{Action: StaleActionHandover, InstanceIDs: []uint{inst.ID}, TargetUserID: 20},
	})
	r := engine.results[0]
	if r.Status != "failed" {
		t.Errorf("status: got %q want failed", r.Status)
	}
}

func TestApplyEngine_Handover_TargetUserNotFound(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	// target user 999 doesn't exist → preCheck fails
	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true}, map[uint]bool{10: true})
	engine.run([]applyActionItem{
		{Action: StaleActionHandover, InstanceIDs: []uint{inst.ID}, TargetUserID: 999},
	})
	r := engine.results[0]
	if r.Status != "failed" {
		t.Errorf("status: got %q want failed", r.Status)
	}
}

func TestApplyEngine_Handover_ExplicitGroupID_MultiGroup(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 5, 10)
	covSeedMember(t, 7, 20)
	covSeedMember(t, 5, 20) // bob in groups 5 and 7
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	// explicit target_group_id=5, bob is in 5 → success
	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true, 7: true}, map[uint]bool{10: true, 20: true})
	engine.run([]applyActionItem{
		{Action: StaleActionHandover, InstanceIDs: []uint{inst.ID}, TargetUserID: 20, TargetGroupID: uintPtr(5)},
	})
	r := engine.results[0]
	if r.Status != "success" {
		t.Errorf("status: got %q want success, err=%s", r.Status, r.Error)
	}
	var updated model.Instance
	model.DB(ctx).First(&updated, inst.ID)
	if updated.UserID != 20 {
		t.Errorf("user_id: got %d want 20", updated.UserID)
	}
}

func TestApplyEngine_Handover_ExplicitGroupID_NotInTargetGroups(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 5, 10)
	covSeedMember(t, 7, 20) // bob only in group 7
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	// explicit target_group_id=5, but bob not in group 5 → fail
	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true, 7: true}, map[uint]bool{10: true, 20: true})
	engine.run([]applyActionItem{
		{Action: StaleActionHandover, InstanceIDs: []uint{inst.ID}, TargetUserID: 20, TargetGroupID: uintPtr(5)},
	})
	r := engine.results[0]
	if r.Status != "failed" {
		t.Errorf("status: got %q want failed", r.Status)
	}
}

func TestApplyEngine_Handover_TargetNoGroups_ExplicitZero(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob") // bob has no groups
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	// target_group_id=0, bob has no groups → success, instance.group_id=0
	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true}, map[uint]bool{10: true, 20: true})
	engine.run([]applyActionItem{
		{Action: StaleActionHandover, InstanceIDs: []uint{inst.ID}, TargetUserID: 20, TargetGroupID: uintPtr(0)},
	})
	r := engine.results[0]
	if r.Status != "success" {
		t.Errorf("status: got %q want success, err=%s", r.Status, r.Error)
	}
	var updated model.Instance
	model.DB(ctx).First(&updated, inst.ID)
	if updated.GroupID != 0 {
		t.Errorf("group_id: got %d want 0", updated.GroupID)
	}
}

func TestApplyEngine_Handover_TargetSingleGroup_AutoSelect(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 5, 10)
	covSeedMember(t, 7, 20) // bob has single group 7
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	// target_group_id=0, bob has single group 7 → auto-select 7
	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true, 7: true}, map[uint]bool{10: true, 20: true})
	engine.run([]applyActionItem{
		{Action: StaleActionHandover, InstanceIDs: []uint{inst.ID}, TargetUserID: 20, TargetGroupID: uintPtr(0)},
	})
	r := engine.results[0]
	if r.Status != "success" {
		t.Errorf("status: got %q want success, err=%s", r.Status, r.Error)
	}
}

func TestApplyEngine_Handover_TargetMultiGroup_NoGroupID(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 5, 10)
	covSeedMember(t, 5, 20)
	covSeedMember(t, 7, 20) // bob in groups 5 and 7
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	// target_group_id=0, bob has 2 groups → fail
	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true, 7: true}, map[uint]bool{10: true, 20: true})
	engine.run([]applyActionItem{
		{Action: StaleActionHandover, InstanceIDs: []uint{inst.ID}, TargetUserID: 20, TargetGroupID: uintPtr(0)},
	})
	r := engine.results[0]
	if r.Status != "failed" {
		t.Errorf("status: got %q want failed", r.Status)
	}
}

func TestApplyEngine_PendingUser_BothOptions(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 7, 10)
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true, 7: true}, map[uint]bool{10: true})
	engine.run([]applyActionItem{
		{Action: StaleActionPendingUser, InstanceIDs: []uint{inst.ID}, AllowMigrate: true, AllowSameGroupHandover: true},
	})
	r := engine.results[0]
	if r.Status != "success" {
		t.Errorf("status: got %q want success, err=%s", r.Status, r.Error)
	}
}

func TestApplyEngine_PendingUser_HandoverOnly(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 7, 10)
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true, 7: true}, map[uint]bool{10: true})
	engine.run([]applyActionItem{
		{Action: StaleActionPendingUser, InstanceIDs: []uint{inst.ID}, AllowSameGroupHandover: true},
	})
	r := engine.results[0]
	if r.Status != "success" {
		t.Errorf("status: got %q want success, err=%s", r.Status, r.Error)
	}
}

func TestApplyEngine_Handover_NoopScenario(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	covSeedMember(t, 5, 20)
	inst := covSeedInstance(t, 1, 10, 5, "agent-1", "ins-1")

	// user is in group 5 = noop, but handover bypasses noop → should succeed
	engine := newApplyEngine(ctx, 99, TriggerSourceUserEdit, map[uint]bool{5: true}, map[uint]bool{10: true, 20: true})
	engine.run([]applyActionItem{
		{Action: StaleActionHandover, InstanceIDs: []uint{inst.ID}, TargetUserID: 20},
	})
	r := engine.results[0]
	if r.Status != "success" {
		t.Errorf("status: got %q want success (handover bypasses noop), err=%s", r.Status, r.Error)
	}
}

// ── stale_instances_apply.go: pure helper functions ────────────────────────────

func TestTranslateApplyError(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	// known code
	s := translateApplyError(ctx, "target_group_not_found")
	if s == "target_group_not_found" {
		t.Error("known code should be translated")
	}
	// unknown code → returned as-is
	s = translateApplyError(ctx, "some_unknown_code")
	if s != "some_unknown_code" {
		t.Errorf("unknown code: got %q want some_unknown_code", s)
	}
}

func TestLookupUsername(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	if s := lookupUsername(ctx, 0); s != "" {
		t.Errorf("userID=0: got %q want empty", s)
	}
	if s := lookupUsername(ctx, 10); s != "alice" {
		t.Errorf("userID=10: got %q want alice", s)
	}
	if s := lookupUsername(ctx, 999); s != "" {
		t.Errorf("unknown user: got %q want empty", s)
	}
}

func TestLookupGroupName(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	if s := lookupGroupName(ctx, 0); s != "默认" {
		t.Errorf("groupID=0: got %q want 默认", s)
	}
	covSeedGroup(t, 5, "g5")
	if s := lookupGroupName(ctx, 5); s != "g5" {
		t.Errorf("groupID=5: got %q want g5", s)
	}
	// unknown group → "#id"
	if s := lookupGroupName(ctx, 999); s != "#999" {
		t.Errorf("unknown group: got %q want #999", s)
	}
}

func TestFormatNullableTime(t *testing.T) {
	nilRes := formatNullableTime(nil)
	if nilRes != nil {
		t.Error("nil input should return nil")
	}
	now := time.Now()
	s := formatNullableTime(&now)
	if s == nil || *s == "" {
		t.Error("non-nil input should return non-empty string")
	}
}

func TestPendingUserExtra(t *testing.T) {
	s := pendingUserExtra(true, false)
	if s == "" || s == "{}" {
		t.Errorf("pendingUserExtra(true,false): got %q", s)
	}
}

func TestScenarioMigrateAction(t *testing.T) {
	if s := scenarioMigrateAction(ScenarioD); s != model.ICGRActionParentChangePending {
		t.Errorf("D: got %q want %q", s, model.ICGRActionParentChangePending)
	}
	if s := scenarioMigrateAction(ScenarioA); s != model.ICGRActionMigrate {
		t.Errorf("A: got %q want %q", s, model.ICGRActionMigrate)
	}
}

func TestUintInSlice(t *testing.T) {
	if !uintInSlice(5, []uint{1, 5, 9}) {
		t.Error("5 in [1,5,9] should be true")
	}
	if uintInSlice(5, []uint{1, 9}) {
		t.Error("5 in [1,9] should be false")
	}
	if uintInSlice(5, nil) {
		t.Error("5 in nil should be false")
	}
}

func TestEnrichAdminInstancesWithStaleFields(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedInstance(t, 1, 10, 5, "a", "ins-a")
	model.AddInstanceFlag(ctx, 1, model.InstanceFlagStaleGroup, "")
	model.AddInstanceFlag(ctx, 1, model.InstanceFlagPendingUserAction, "")

	items := []adminInstanceItemWithStatus{{ID: 1}}
	enrichAdminInstancesWithStaleFields(ctx, items)
	if len(items[0].Flags) != 2 {
		t.Errorf("want 2 flags, got %v", items[0].Flags)
	}
	// empty items → no-op
	enrichAdminInstancesWithStaleFields(ctx, nil)
}

func TestMarkStaleForSubtree(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 1, "root")
	covSeedGroup(t, 2, "child")
	covSeedClosureSelf(t, 1)
	covSeedClosureSelf(t, 2)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})
	covSeedInstance(t, 1, 10, 1, "agent-1", "ins-1")
	covSeedInstance(t, 2, 10, 2, "agent-2", "ins-2")

	// rootGroupID=0 → early return
	markStaleForSubtree(ctx, 0)

	// rootGroupID=1 → writes records for groups 1 and 2
	markStaleForSubtree(ctx, 1)

	rows, total, _ := model.ListICGRs(ctx, model.ListICGRsParams{Page: 1, PageSize: 10})
	if total != 2 {
		t.Errorf("want 2 ICGR records, got %d", total)
	}
	_ = rows
}

func TestStopStartInstanceCloud_EmptyID(t *testing.T) {
	defer setupCoverageTest(t)()
	// empty cvm ID → early return, no panic
	stopInstanceCloud(context.Background(), "")
	startInstanceCloud(context.Background(), "")
}

// ── stale_instances_scenario.go ────────────────────────────────────────────────

func TestLoadUserGroupIDs_Zero(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	ids, err := loadUserGroupIDs(ctx, 0)
	if err != nil || len(ids) != 0 {
		t.Errorf("userID=0: got ids=%v err=%v", ids, err)
	}
}

func TestUserInGroup(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedMember(t, 5, 10)

	// userID=0 → false
	ok, _ := userInGroup(ctx, 0, 5)
	if ok {
		t.Error("userID=0 should be false")
	}
	// groupID=0 → false
	ok, _ = userInGroup(ctx, 10, 0)
	if ok {
		t.Error("groupID=0 should be false")
	}
	// user in group → true
	ok, _ = userInGroup(ctx, 10, 5)
	if !ok {
		t.Error("user 10 in group 5 should be true")
	}
	// user not in group → false
	ok, _ = userInGroup(ctx, 10, 7)
	if ok {
		t.Error("user 10 in group 7 should be false")
	}
}

// ── admin_stale_instances.go: loadExistingGroupIDs / loadExistingUserIDs ────────

func TestLoadExistingGroupIDs(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")

	m := loadExistingGroupIDs(ctx, map[uint]struct{}{5: {}, 7: {}, 999: {}})
	if !m[5] || !m[7] {
		t.Errorf("5 and 7 should exist, got %v", m)
	}
	if m[999] {
		t.Error("999 should not exist")
	}
	// empty → empty map
	m = loadExistingGroupIDs(ctx, nil)
	if len(m) != 0 {
		t.Errorf("nil → want empty, got %v", m)
	}
}

func TestLoadExistingUserIDs(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")

	m := loadExistingUserIDs(ctx, map[uint]struct{}{10: {}, 20: {}, 999: {}})
	if !m[10] || !m[20] {
		t.Errorf("10 and 20 should exist, got %v", m)
	}
	if m[999] {
		t.Error("999 should not exist")
	}
	// empty → empty map
	m = loadExistingUserIDs(ctx, nil)
	if len(m) != 0 {
		t.Errorf("nil → want empty, got %v", m)
	}
}

// ── admin_stale_instances_action_options.go: query + builder functions ─────────

func TestLoadGroupMemberCounts(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedMember(t, 5, 10)
	covSeedMember(t, 5, 20)
	covSeedMember(t, 7, 30)

	m := loadGroupMemberCounts(ctx, []uint{5, 7, 999})
	if m[5] != 2 {
		t.Errorf("group 5: want 2, got %d", m[5])
	}
	if m[7] != 1 {
		t.Errorf("group 7: want 1, got %d", m[7])
	}
	if _, ok := m[999]; ok {
		t.Error("group 999 should not be in map")
	}
	// empty
	m = loadGroupMemberCounts(ctx, nil)
	if len(m) != 0 {
		t.Errorf("nil → want empty")
	}
}

func TestLoadActionOptionsUsernameMap(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")

	m := loadActionOptionsUsernameMap(ctx, []uint{10, 20, 999})
	if m[10] != "alice" || m[20] != "bob" {
		t.Errorf("got %v", m)
	}
	if _, ok := m[999]; ok {
		t.Error("999 should not be in map")
	}
	// empty
	m = loadActionOptionsUsernameMap(ctx, nil)
	if len(m) != 0 {
		t.Errorf("nil → want empty")
	}
}

func TestLoadUserGroupsBriefMap(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 5, 10)
	covSeedMember(t, 7, 10)

	m := loadUserGroupsBriefMap(ctx, []uint{10})
	if len(m[10]) != 2 {
		t.Fatalf("want 2 groups for user 10, got %d", len(m[10]))
	}
	// empty
	m = loadUserGroupsBriefMap(ctx, nil)
	if len(m) != 0 {
		t.Errorf("nil → want empty")
	}
}

func TestCollectUintGroupIDs(t *testing.T) {
	rows := []staleActionInstRow{
		{ID: 1, GroupID: 5},
		{ID: 2, GroupID: 0}, // filtered
		{ID: 3, GroupID: 5}, // dup
		{ID: 4, GroupID: 7},
	}
	got := collectUintGroupIDs(rows)
	if len(got) != 2 || got[0] != 5 || got[1] != 7 {
		t.Errorf("got %v", got)
	}
}

func TestBuildActionOptNoGroup(t *testing.T) {
	rows := []staleActionInstRow{
		{ID: 1, InstanceID: "ins-1", Name: "a", UserID: 10},
		{ID: 2, InstanceID: "ins-2", Name: "b", UserID: 10},
		{ID: 3, InstanceID: "ins-3", Name: "c", UserID: 20},
	}
	usernameMap := map[uint]string{10: "alice", 20: "bob"}
	userGroupsMap := map[uint][]actionOptUserGroupBrief{
		10: {{GroupID: 5, GroupFullPath: "g5"}},
	}
	out := buildActionOptNoGroup(rows, usernameMap, userGroupsMap)
	if len(out.Options) != 2 {
		t.Fatalf("want 2 entries, got %d", len(out.Options))
	}
	if out.Options[0].UserID != 10 || len(out.Options[0].Instances) != 2 {
		t.Errorf("entry 0: %+v", out.Options[0])
	}
	if !out.Options[0].Migrate || !out.Options[0].PendingUser || !out.Options[0].ArchiveStop {
		t.Errorf("entry 0 should have migrate/pending_user/archive_stop=true")
	}
}

func TestBuildActionOptUserRemoved(t *testing.T) {
	rows := []staleActionInstRow{
		{ID: 1, InstanceID: "ins-1", Name: "a", UserID: 10, GroupID: 5},
		{ID: 2, InstanceID: "ins-2", Name: "b", UserID: 10, GroupID: 7},
		{ID: 3, InstanceID: "ins-3", Name: "c", UserID: 20, GroupID: 5},
	}
	usernameMap := map[uint]string{10: "alice", 20: "bob"}
	groupMemberCounts := map[uint]int{5: 2, 7: 0}
	groupPaths := map[uint]string{5: "g5", 7: "g7"}
	userGroupsMap := map[uint][]actionOptUserGroupBrief{}

	out := buildActionOptUserRemoved(rows, usernameMap, groupMemberCounts, groupPaths, userGroupsMap)
	if len(out.Options) != 2 {
		t.Fatalf("want 2 entries, got %d", len(out.Options))
	}
	// user 10 has groups 5 and 7, group 5 has members → handover=true
	if !out.Options[0].Handover {
		t.Error("user 10 should have handover=true (group 5 has members)")
	}
	// user 20 has only group 5 which has members → handover=true
	if !out.Options[1].Handover {
		t.Error("user 20 should have handover=true")
	}
}

func TestBuildActionOptSubtree(t *testing.T) {
	rows := []staleActionInstRow{
		{ID: 1, InstanceID: "ins-1", Name: "a", UserID: 10, GroupID: 5},
		{ID: 2, InstanceID: "ins-2", Name: "b", UserID: 20, GroupID: 5},
		{ID: 3, InstanceID: "ins-3", Name: "c", UserID: 30, GroupID: 7},
	}
	usernameMap := map[uint]string{10: "alice", 20: "bob", 30: "carol"}
	groupPaths := map[uint]string{5: "g5", 7: "g7"}

	out := buildActionOptSubtree(rows, usernameMap, groupPaths)
	if len(out.Groups) != 2 {
		t.Fatalf("want 2 groups, got %d", len(out.Groups))
	}
	if len(out.Groups[0].Instances) != 2 {
		t.Errorf("group 5: want 2 instances, got %d", len(out.Groups[0].Instances))
	}
}

// ── admin_instances_group_check.go: HTTP handler ───────────────────────────────

func TestHandleAdminInstancesGroupCheck(t *testing.T) {
	defer setupCoverageTest(t)()
	covSeedUser(t, 10, "alice")
	covSeedMember(t, 5, 10)
	covSeedInstance(t, 1, 10, 5, "a", "ins-1")
	covSeedInstance(t, 2, 10, 7, "b", "ins-2") // mismatch
	covSeedInstance(t, 3, 10, 0, "c", "ins-3") // ungrouped but has group → mismatch

	// method check
	rec := httptest.NewRecorder()
	HandleAdminInstancesGroupCheck(rec, covAdminReq("GET", "/admin/instances/group-check", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET: got %d want 405", rec.Code)
	}

	// success with check_user_group
	rec = httptest.NewRecorder()
	HandleAdminInstancesGroupCheck(rec, covAdminReq("POST", "/admin/instances/group-check", groupCheckRequest{
		IDs: []uint{1, 2, 3}, CheckUserGroup: true,
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST: got %d want 200, body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONBody(t, rec)
	results := body["results"].([]interface{})
	if len(results) != 3 {
		t.Errorf("want 3 results, got %d", len(results))
	}

	// empty ids → empty results
	rec = httptest.NewRecorder()
	HandleAdminInstancesGroupCheck(rec, covAdminReq("POST", "/admin/instances/group-check", groupCheckRequest{
		IDs: []uint{0, 0},
	}))
	if rec.Code != http.StatusOK {
		t.Errorf("empty ids: got %d want 200", rec.Code)
	}

	// too many IDs
	ids := make([]uint, groupCheckMaxIDs+1)
	for i := range ids {
		ids[i] = uint(i + 1)
	}
	rec = httptest.NewRecorder()
	HandleAdminInstancesGroupCheck(rec, covAdminReq("POST", "/admin/instances/group-check", groupCheckRequest{
		IDs: ids,
	}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("too many: got %d want 400", rec.Code)
	}

	// bad JSON
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/instances/group-check", bytes.NewBufferString("bad json"))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminInstancesGroupCheck(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad json: got %d want 400", rec.Code)
	}
}

// ── admin_stale_instances.go: HTTP handlers ────────────────────────────────────

func TestHandleAdminStaleInstancesRecords(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	r := &model.InstanceChangeGroupRecord{
		InstancePK: 1, InstanceID: "ins-1", UserIDBefore: 1, UserIDAfter: 2,
		GroupIDBefore: 5, GroupIDAfter: 7, Action: model.ICGRActionMigrate,
		ActorType: model.ICGRActorAdmin, ActorID: 99, TriggerSource: "user_edit",
	}
	model.CreateICGR(ctx, r)

	// method check
	rec := httptest.NewRecorder()
	HandleAdminStaleInstancesRecords(rec, covAdminReq("POST", "/admin/stale-instances/records", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST: got %d want 405", rec.Code)
	}

	// success GET
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesRecords(rec, covAdminReq("GET", "/admin/stale-instances/records?action=migrate&page=1&page_size=10", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET: got %d want 200, body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONBody(t, rec)
	if body["total"].(float64) != 1 {
		t.Errorf("total: got %v want 1", body["total"])
	}

	// GET with string instance_id (ins-xxx)
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesRecords(rec, covAdminReq("GET", "/admin/stale-instances/records?instance_id=ins-1", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("GET with ins-id: got %d want 200", rec.Code)
	}

	// GET with numeric instance_id
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesRecords(rec, covAdminReq("GET", "/admin/stale-instances/records?instance_id=1&user_id=1&group_id=5&from=2025-01-01T00:00:00Z&to=2026-01-01T00:00:00Z", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("GET with filters: got %d want 200", rec.Code)
	}

	// GET with page_size > 100 → capped
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesRecords(rec, covAdminReq("GET", "/admin/stale-instances/records?page_size=200", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("GET page_size=200: got %d want 200", rec.Code)
	}
}

func TestHandleAdminStaleInstancesApply(t *testing.T) {
	defer setupCoverageTest(t)()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 7, 10)
	covSeedInstance(t, 1, 10, 5, "a", "ins-1")

	// method check
	rec := httptest.NewRecorder()
	HandleAdminStaleInstancesApply(rec, covAdminReq("GET", "/admin/stale-instances/apply", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET: got %d want 405", rec.Code)
	}

	// bad JSON
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/stale-instances/apply", bytes.NewBufferString("bad"))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminStaleInstancesApply(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad json: got %d want 400", rec.Code)
	}

	// invalid trigger_source
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesApply(rec, covAdminReq("POST", "/admin/stale-instances/apply", applyRequest{
		TriggerSource: "invalid_source",
	}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid trigger: got %d want 400", rec.Code)
	}

	// empty actions
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesApply(rec, covAdminReq("POST", "/admin/stale-instances/apply", applyRequest{
		TriggerSource: TriggerSourceUserEdit,
	}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty actions: got %d want 400", rec.Code)
	}

	// invalid action
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesApply(rec, covAdminReq("POST", "/admin/stale-instances/apply", applyRequest{
		TriggerSource: TriggerSourceUserEdit,
		Actions: []applyActionItem{{Action: "delete", InstanceIDs: []uint{1}}},
	}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid action: got %d want 400", rec.Code)
	}

	// migrate without target_group_id
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesApply(rec, covAdminReq("POST", "/admin/stale-instances/apply", applyRequest{
		TriggerSource: TriggerSourceUserEdit,
		Actions: []applyActionItem{{Action: StaleActionMigrate, InstanceIDs: []uint{1}}},
	}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("migrate no target: got %d want 400", rec.Code)
	}

	// handover without target_user_id
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesApply(rec, covAdminReq("POST", "/admin/stale-instances/apply", applyRequest{
		TriggerSource: TriggerSourceUserEdit,
		Actions: []applyActionItem{{Action: StaleActionHandover, InstanceIDs: []uint{1}}},
	}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("handover no target: got %d want 400", rec.Code)
	}

	// empty instance_ids
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesApply(rec, covAdminReq("POST", "/admin/stale-instances/apply", applyRequest{
		TriggerSource: TriggerSourceUserEdit,
		Actions: []applyActionItem{{Action: StaleActionArchiveStop, InstanceIDs: []uint{}}},
	}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty instance_ids: got %d want 400", rec.Code)
	}

	// success
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesApply(rec, covAdminReq("POST", "/admin/stale-instances/apply", applyRequest{
		TriggerSource: TriggerSourceUserEdit,
		Actions: []applyActionItem{
			{Action: StaleActionMigrate, InstanceIDs: []uint{1}, TargetGroupID: uintPtr(7)},
		},
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("success: got %d want 200, body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONBody(t, rec)
	results := body["results"].([]interface{})
	if len(results) != 1 {
		t.Errorf("want 1 result, got %d", len(results))
	}
}

func TestHandleAdminStaleInstancesConfigDiff(t *testing.T) {
	defer setupCoverageTest(t)()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedInstance(t, 1, 10, 5, "a", "ins-1")

	// method check
	rec := httptest.NewRecorder()
	HandleAdminStaleInstancesConfigDiff(rec, covAdminReq("GET", "/admin/stale-instances/config-diff", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET: got %d want 405", rec.Code)
	}

	// bad JSON
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/stale-instances/config-diff", bytes.NewBufferString("bad"))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminStaleInstancesConfigDiff(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad json: got %d want 400", rec.Code)
	}

	// empty instance_ids
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesConfigDiff(rec, covAdminReq("POST", "/admin/stale-instances/config-diff", configDiffRequest{
		TargetGroupID: uintPtr(7),
	}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty ids: got %d want 400", rec.Code)
	}

	// nil target_group_id
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesConfigDiff(rec, covAdminReq("POST", "/admin/stale-instances/config-diff", configDiffRequest{
		InstanceIDs: []uint{1},
	}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("nil target: got %d want 400", rec.Code)
	}

	// too many instance_ids
	ids := make([]uint, staleConfigDiffMaxBatch+1)
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesConfigDiff(rec, covAdminReq("POST", "/admin/stale-instances/config-diff", configDiffRequest{
		InstanceIDs: ids, TargetGroupID: uintPtr(7),
	}))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("too many: got %d want 400", rec.Code)
	}

	// success
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesConfigDiff(rec, covAdminReq("POST", "/admin/stale-instances/config-diff", configDiffRequest{
		InstanceIDs: []uint{1}, TargetGroupID: uintPtr(7),
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("success: got %d want 200, body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONBody(t, rec)
	if body["ok"] != true {
		t.Errorf("ok: got %v want true", body["ok"])
	}
}

// ── admin_stale_instances_action_options.go: HTTP handler ──────────────────────

func TestHandleAdminStaleInstancesActionOptions(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedClosureSelf(t, 5)
	covSeedClosureSelf(t, 7)
	covSeedMember(t, 5, 10)
	covSeedMember(t, 7, 20)
	covSeedInstance(t, 1, 10, 5, "a", "ins-1")
	covSeedInstance(t, 2, 20, 0, "b", "ins-2") // no_group scenario

	// method check
	rec := httptest.NewRecorder()
	HandleAdminStaleInstancesActionOptions(rec, covAdminReq("GET", "/admin/stale-instances/action-options", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET: got %d want 405", rec.Code)
	}

	// bad JSON
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/stale-instances/action-options", bytes.NewBufferString("bad"))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleAdminStaleInstancesActionOptions(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad json: got %d want 400", rec.Code)
	}

	// success with user_group_ids
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesActionOptions(rec, covAdminReq("POST", "/admin/stale-instances/action-options", instancesByUserGroupRequest{
		UserGroupIDs: []userGroupPair{{UserID: 10, GroupID: 5}},
		GroupIDs:     []uint{5},
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("success: got %d want 200, body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONBody(t, rec)
	if body["ok"] != true {
		t.Errorf("ok: got %v want true", body["ok"])
	}

	// success with empty request (no_group scan only)
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesActionOptions(rec, covAdminReq("POST", "/admin/stale-instances/action-options", instancesByUserGroupRequest{}))
	if rec.Code != http.StatusOK {
		t.Errorf("empty req: got %d want 200", rec.Code)
	}

	// flag some instances as stale_group and verify filtering
	model.AddInstanceFlag(ctx, 1, model.InstanceFlagStaleGroup, "")
	rec = httptest.NewRecorder()
	HandleAdminStaleInstancesActionOptions(rec, covAdminReq("POST", "/admin/stale-instances/action-options", instancesByUserGroupRequest{
		UserGroupIDs: []userGroupPair{{UserID: 10, GroupID: 5}},
	}))
	if rec.Code != http.StatusOK {
		t.Errorf("with stale flag: got %d want 200", rec.Code)
	}
}

// ── openclaw_stale_instances.go: HTTP handlers ─────────────────────────────────

func TestHandleUserStaleInstancesRebind(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	covSeedInstance(t, 1, 10, 0, "a", "ins-1") // group_id=0, pending
	model.AddInstanceFlag(ctx, 1, model.InstanceFlagPendingUserAction, "")
	model.AddInstanceFlag(ctx, 1, model.InstanceFlagAllowMigrate, "")

	// method check
	rec := httptest.NewRecorder()
	HandleUserStaleInstancesRebind(rec, covUserReq(t, "GET", "/openclaw/stale-instances/rebind", nil, "alice"))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET: got %d want 405", rec.Code)
	}

	// bad request body
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesRebind(rec, covUserReq(t, "POST", "/openclaw/stale-instances/rebind", nil, "alice"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("nil body: got %d want 400", rec.Code)
	}

	// instance not found
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesRebind(rec, covUserReq(t, "POST", "/openclaw/stale-instances/rebind", userRebindRequest{
		ID: 999, TargetGroupID: 5,
	}, "alice"))
	if rec.Code != http.StatusNotFound {
		t.Errorf("not found: got %d want 404", rec.Code)
	}

	// not owner
	covSeedUser(t, 20, "bob")
	covSeedInstance(t, 2, 20, 0, "b", "ins-2")
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesRebind(rec, covUserReq(t, "POST", "/openclaw/stale-instances/rebind", userRebindRequest{
		ID: 2, TargetGroupID: 5,
	}, "alice"))
	if rec.Code != http.StatusForbidden {
		t.Errorf("not owner: got %d want 403", rec.Code)
	}

	// success: rebind to group 5
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesRebind(rec, covUserReq(t, "POST", "/openclaw/stale-instances/rebind", userRebindRequest{
		ID: 1, TargetGroupID: 5,
	}, "alice"))
	if rec.Code != http.StatusOK {
		t.Fatalf("success: got %d want 200, body=%s", rec.Code, rec.Body.String())
	}
	var updated model.Instance
	model.DB(ctx).First(&updated, 1)
	if updated.GroupID != 5 {
		t.Errorf("group_id: got %d want 5", updated.GroupID)
	}
}

func TestHandleUserStaleInstancesRebind_NoFlags(t *testing.T) {
	defer setupCoverageTest(t)()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	covSeedInstance(t, 1, 10, 0, "a", "ins-1")
	// no pending_user_action flag → 400
	rec := httptest.NewRecorder()
	HandleUserStaleInstancesRebind(rec, covUserReq(t, "POST", "/openclaw/stale-instances/rebind", userRebindRequest{
		ID: 1, TargetGroupID: 5,
	}, "alice"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("no pending flag: got %d want 400", rec.Code)
	}
}

func TestHandleUserStaleInstancesRebind_NoAllowMigrate(t *testing.T) {
	defer setupCoverageTest(t)()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	covSeedInstance(t, 1, 10, 0, "a", "ins-1")
	model.AddInstanceFlag(context.Background(), 1, model.InstanceFlagPendingUserAction, "")
	// no allow_migrate flag → 403
	rec := httptest.NewRecorder()
	HandleUserStaleInstancesRebind(rec, covUserReq(t, "POST", "/openclaw/stale-instances/rebind", userRebindRequest{
		ID: 1, TargetGroupID: 5,
	}, "alice"))
	if rec.Code != http.StatusForbidden {
		t.Errorf("no allow_migrate: got %d want 403", rec.Code)
	}
}

func TestHandleUserStaleInstancesRebind_TargetZeroButUserHasGroups(t *testing.T) {
	defer setupCoverageTest(t)()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	covSeedInstance(t, 1, 10, 0, "a", "ins-1")
	model.AddInstanceFlag(context.Background(), 1, model.InstanceFlagPendingUserAction, "")
	model.AddInstanceFlag(context.Background(), 1, model.InstanceFlagAllowMigrate, "")
	// target_group_id=0 but user has groups → 400
	rec := httptest.NewRecorder()
	HandleUserStaleInstancesRebind(rec, covUserReq(t, "POST", "/openclaw/stale-instances/rebind", userRebindRequest{
		ID: 1, TargetGroupID: 0,
	}, "alice"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("target=0 with groups: got %d want 400", rec.Code)
	}
}

func TestHandleUserStaleInstancesRebind_TargetNotInUserGroups(t *testing.T) {
	defer setupCoverageTest(t)()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 5, 10) // alice in group 5
	covSeedInstance(t, 1, 10, 0, "a", "ins-1")
	model.AddInstanceFlag(context.Background(), 1, model.InstanceFlagPendingUserAction, "")
	model.AddInstanceFlag(context.Background(), 1, model.InstanceFlagAllowMigrate, "")
	// target_group_id=7 not in user's groups → 400
	rec := httptest.NewRecorder()
	HandleUserStaleInstancesRebind(rec, covUserReq(t, "POST", "/openclaw/stale-instances/rebind", userRebindRequest{
		ID: 1, TargetGroupID: 7,
	}, "alice"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("target not in user groups: got %d want 400", rec.Code)
	}
}

func TestHandleUserStaleInstancesHandoverInitiate(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	covSeedMember(t, 5, 20)
	covSeedInstance(t, 1, 10, 5, "a", "ins-1")
	model.AddInstanceFlag(ctx, 1, model.InstanceFlagAllowSameGroupHandover, "")

	// method check
	rec := httptest.NewRecorder()
	HandleUserStaleInstancesHandoverInitiate(rec, covUserReq(t, "GET", "/openclaw/stale-instances/initiate", nil, "alice"))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET: got %d want 405", rec.Code)
	}

	// bad request
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverInitiate(rec, covUserReq(t, "POST", "/openclaw/stale-instances/initiate", nil, "alice"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("nil body: got %d want 400", rec.Code)
	}

	// instance not found
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverInitiate(rec, covUserReq(t, "POST", "/openclaw/stale-instances/initiate", handoverInitiateRequest{
		ID: 999, TargetUsername: "bob",
	}, "alice"))
	if rec.Code != http.StatusNotFound {
		t.Errorf("not found: got %d want 404", rec.Code)
	}

	// not owner
	covSeedInstance(t, 2, 20, 5, "b", "ins-2")
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverInitiate(rec, covUserReq(t, "POST", "/openclaw/stale-instances/initiate", handoverInitiateRequest{
		ID: 2, TargetUsername: "bob",
	}, "alice"))
	if rec.Code != http.StatusForbidden {
		t.Errorf("not owner: got %d want 403", rec.Code)
	}

	// success
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverInitiate(rec, covUserReq(t, "POST", "/openclaw/stale-instances/initiate", handoverInitiateRequest{
		ID: 1, TargetUsername: "bob",
	}, "alice"))
	if rec.Code != http.StatusOK {
		t.Fatalf("success: got %d want 200, body=%s", rec.Code, rec.Body.String())
	}
	var updated model.Instance
	model.DB(ctx).First(&updated, 1)
	if updated.HandoverTargetUserID != 20 {
		t.Errorf("handover_target_user_id: got %d want 20", updated.HandoverTargetUserID)
	}
}

func TestHandleUserStaleInstancesHandoverInitiate_Errors(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	covSeedMember(t, 5, 20)
	covSeedInstance(t, 1, 10, 5, "a", "ins-1")
	model.AddInstanceFlag(ctx, 1, model.InstanceFlagAllowSameGroupHandover, "")

	// ungrouped instance → 400
	covSeedInstance(t, 2, 10, 0, "b", "ins-2")
	rec := httptest.NewRecorder()
	HandleUserStaleInstancesHandoverInitiate(rec, covUserReq(t, "POST", "/openclaw/stale-instances/initiate", handoverInitiateRequest{
		ID: 2, TargetUsername: "bob",
	}, "alice"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("ungrouped: got %d want 400", rec.Code)
	}

	// no allow_same_group_handover flag
	covSeedInstance(t, 3, 10, 5, "c", "ins-3")
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverInitiate(rec, covUserReq(t, "POST", "/openclaw/stale-instances/initiate", handoverInitiateRequest{
		ID: 3, TargetUsername: "bob",
	}, "alice"))
	if rec.Code != http.StatusForbidden {
		t.Errorf("no flag: got %d want 403", rec.Code)
	}

	// target user not found
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverInitiate(rec, covUserReq(t, "POST", "/openclaw/stale-instances/initiate", handoverInitiateRequest{
		ID: 1, TargetUsername: "nobody",
	}, "alice"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("user not found: got %d want 400", rec.Code)
	}

	// target is self
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverInitiate(rec, covUserReq(t, "POST", "/openclaw/stale-instances/initiate", handoverInitiateRequest{
		ID: 1, TargetUsername: "alice",
	}, "alice"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("self: got %d want 400", rec.Code)
	}

	// target not in instance group
	covSeedUser(t, 30, "carol")
	covSeedGroup(t, 7, "g7")
	covSeedMember(t, 7, 30) // carol in group 7, not 5
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverInitiate(rec, covUserReq(t, "POST", "/openclaw/stale-instances/initiate", handoverInitiateRequest{
		ID: 1, TargetUsername: "carol",
	}, "alice"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("not in group: got %d want 400", rec.Code)
	}
}

func TestHandleUserStaleInstancesHandoverCancel(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	covSeedMember(t, 5, 20)
	inst := covSeedInstance(t, 1, 10, 5, "a", "ins-1")
	// set handover_target_user_id
	model.DB(ctx).Model(&model.Instance{}).Where("id = ?", inst.ID).
		Update("handover_target_user_id", 20)

	// method check
	rec := httptest.NewRecorder()
	HandleUserStaleInstancesHandoverCancel(rec, covUserReq(t, "GET", "/openclaw/stale-instances/cancel", nil, "alice"))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET: got %d want 405", rec.Code)
	}

	// bad request
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverCancel(rec, covUserReq(t, "POST", "/openclaw/stale-instances/cancel", nil, "alice"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("nil body: got %d want 400", rec.Code)
	}

	// not owner
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverCancel(rec, covUserReq(t, "POST", "/openclaw/stale-instances/cancel", handoverIDOnlyRequest{
		ID: 1,
	}, "bob"))
	if rec.Code != http.StatusForbidden {
		t.Errorf("not owner: got %d want 403", rec.Code)
	}

	// success: cancel handover
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverCancel(rec, covUserReq(t, "POST", "/openclaw/stale-instances/cancel", handoverIDOnlyRequest{
		ID: 1,
	}, "alice"))
	if rec.Code != http.StatusOK {
		t.Fatalf("success: got %d want 200, body=%s", rec.Code, rec.Body.String())
	}
	var updated model.Instance
	model.DB(ctx).First(&updated, 1)
	if updated.HandoverTargetUserID != 0 {
		t.Errorf("handover_target_user_id: got %d want 0", updated.HandoverTargetUserID)
	}

	// no active handover
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverCancel(rec, covUserReq(t, "POST", "/openclaw/stale-instances/cancel", handoverIDOnlyRequest{
		ID: 1,
	}, "alice"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("no active: got %d want 400", rec.Code)
	}
}

func TestHandleUserStaleInstancesHandoverAccept(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	covSeedMember(t, 5, 20)
	inst := covSeedInstance(t, 1, 10, 5, "a", "ins-1")
	model.DB(ctx).Model(&model.Instance{}).Where("id = ?", inst.ID).
		Update("handover_target_user_id", 20)

	// method check
	rec := httptest.NewRecorder()
	HandleUserStaleInstancesHandoverAccept(rec, covUserReq(t, "GET", "/openclaw/stale-instances/accept", nil, "bob"))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET: got %d want 405", rec.Code)
	}

	// bad request
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverAccept(rec, covUserReq(t, "POST", "/openclaw/stale-instances/accept", nil, "bob"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("nil body: got %d want 400", rec.Code)
	}

	// not target (alice tries to accept, but target is bob)
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverAccept(rec, covUserReq(t, "POST", "/openclaw/stale-instances/accept", handoverIDOnlyRequest{
		ID: 1,
	}, "alice"))
	if rec.Code != http.StatusForbidden {
		t.Errorf("not target: got %d want 403", rec.Code)
	}

	// success: bob accepts
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverAccept(rec, covUserReq(t, "POST", "/openclaw/stale-instances/accept", handoverIDOnlyRequest{
		ID: 1,
	}, "bob"))
	if rec.Code != http.StatusOK {
		t.Fatalf("success: got %d want 200, body=%s", rec.Code, rec.Body.String())
	}
	var updated model.Instance
	model.DB(ctx).First(&updated, 1)
	if updated.UserID != 20 {
		t.Errorf("user_id: got %d want 20", updated.UserID)
	}
	// 注意：不在此处测试 "no active handover"——accept 成功后会 spawn
	// go startInstanceCloud goroutine，其 DB 访问与 SQLite 内存 DB 并发冲突
	// 导致后续 requireLogin 查询偶发失败。no-active 场景由独立测试覆盖。
}

// TestHandleUserStaleInstancesHandoverAccept_NoActive 独立测试 "no active handover" 场景。
// 不经过成功 accept（不 spawn goroutine），避免 SQLite 并发问题。
func TestHandleUserStaleInstancesHandoverAccept_NoActive(t *testing.T) {
	defer setupCoverageTest(t)()
	covSeedUser(t, 10, "alice")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	covSeedInstance(t, 1, 10, 5, "a", "ins-1")
	// handover_target_user_id 默认为 0 → no active handover

	rec := httptest.NewRecorder()
	HandleUserStaleInstancesHandoverAccept(rec, covUserReq(t, "POST", "/openclaw/stale-instances/accept", handoverIDOnlyRequest{
		ID: 1,
	}, "alice"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("no active: got %d want 400", rec.Code)
	}
}

func TestHandleUserStaleInstancesHandoverReject(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	covSeedUser(t, 10, "alice")
	covSeedUser(t, 20, "bob")
	covSeedGroup(t, 5, "g5")
	covSeedMember(t, 5, 10)
	covSeedMember(t, 5, 20)
	inst := covSeedInstance(t, 1, 10, 5, "a", "ins-1")
	model.DB(ctx).Model(&model.Instance{}).Where("id = ?", inst.ID).
		Update("handover_target_user_id", 20)

	// method check
	rec := httptest.NewRecorder()
	HandleUserStaleInstancesHandoverReject(rec, covUserReq(t, "GET", "/openclaw/stale-instances/reject", nil, "bob"))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET: got %d want 405", rec.Code)
	}

	// bad request
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverReject(rec, covUserReq(t, "POST", "/openclaw/stale-instances/reject", nil, "bob"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("nil body: got %d want 400", rec.Code)
	}

	// not target (alice tries to reject)
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverReject(rec, covUserReq(t, "POST", "/openclaw/stale-instances/reject", handoverIDOnlyRequest{
		ID: 1,
	}, "alice"))
	if rec.Code != http.StatusForbidden {
		t.Errorf("not target: got %d want 403", rec.Code)
	}

	// success: bob rejects
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverReject(rec, covUserReq(t, "POST", "/openclaw/stale-instances/reject", handoverIDOnlyRequest{
		ID: 1,
	}, "bob"))
	if rec.Code != http.StatusOK {
		t.Fatalf("success: got %d want 200, body=%s", rec.Code, rec.Body.String())
	}
	var updated model.Instance
	model.DB(ctx).First(&updated, 1)
	if updated.HandoverRejectedByUserID != 20 {
		t.Errorf("rejected_by: got %d want 20", updated.HandoverRejectedByUserID)
	}

	// no active handover
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverReject(rec, covUserReq(t, "POST", "/openclaw/stale-instances/reject", handoverIDOnlyRequest{
		ID: 1,
	}, "bob"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("no active: got %d want 400", rec.Code)
	}
}

func TestHandleUserStaleInstances_NotLogin(t *testing.T) {
	defer setupCoverageTest(t)()
	// not logged in → 401
	rec := httptest.NewRecorder()
	HandleUserStaleInstancesRebind(rec, covUserReq(t, "POST", "/openclaw/stale-instances/rebind", userRebindRequest{ID: 1}, ""))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("not logged in: got %d want 401", rec.Code)
	}
}

func TestHandleUserStaleInstances_InstanceNotFound(t *testing.T) {
	defer setupCoverageTest(t)()
	covSeedUser(t, 10, "alice")
	// handover cancel with non-existent instance
	rec := httptest.NewRecorder()
	HandleUserStaleInstancesHandoverCancel(rec, covUserReq(t, "POST", "/openclaw/stale-instances/cancel", handoverIDOnlyRequest{
		ID: 999,
	}, "alice"))
	if rec.Code != http.StatusNotFound {
		t.Errorf("not found: got %d want 404", rec.Code)
	}
	// handover accept with non-existent instance
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverAccept(rec, covUserReq(t, "POST", "/openclaw/stale-instances/accept", handoverIDOnlyRequest{
		ID: 999,
	}, "alice"))
	if rec.Code != http.StatusNotFound {
		t.Errorf("not found: got %d want 404", rec.Code)
	}
	// handover reject with non-existent instance
	rec = httptest.NewRecorder()
	HandleUserStaleInstancesHandoverReject(rec, covUserReq(t, "POST", "/openclaw/stale-instances/reject", handoverIDOnlyRequest{
		ID: 999,
	}, "alice"))
	if rec.Code != http.StatusNotFound {
		t.Errorf("not found: got %d want 404", rec.Code)
	}
}

// ── admin_stale_instances.go: not admin checks ─────────────────────────────────

func TestHandleAdminStale_NotAdmin(t *testing.T) {
	defer setupCoverageTest(t)()
	// no admin token → 401
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/admin/stale-instances/config-diff", nil)
	HandleAdminStaleInstancesConfigDiff(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("config-diff not admin: got %d want 401", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/admin/stale-instances/apply", nil)
	HandleAdminStaleInstancesApply(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("apply not admin: got %d want 401", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/admin/stale-instances/records", nil)
	HandleAdminStaleInstancesRecords(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("records not admin: got %d want 401", rec.Code)
	}
}

// ── stale_instances_apply.go: writeICGRTx with empty extraJSON ─────────────────

func TestWriteICGRTx_EmptyExtra(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	tx := model.DB(ctx).Begin()
	inst := &model.Instance{Model: gorm.Model{ID: 1}, InstanceId: "ins-1"}
	// empty extraJSON → defaults to "{}"
	if err := writeICGRTx(tx, inst, 0, 5, 1, 2, model.ICGRActionMigrate, model.ICGRActorAdmin, 99, "test", ""); err != nil {
		tx.Rollback()
		t.Fatalf("writeICGRTx: %v", err)
	}
	tx.Commit()
	rows, _, _ := model.ListICGRs(ctx, model.ListICGRsParams{Page: 1, PageSize: 10})
	if len(rows) != 1 {
		t.Errorf("want 1 row, got %d", len(rows))
	}
}

func TestSetFlagTx_EmptyExtra(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	tx := model.DB(ctx).Begin()
	// empty extra → defaults to "{}"
	if err := setFlagTx(tx, 1, model.InstanceFlagStaleGroup, ""); err != nil {
		tx.Rollback()
		t.Fatalf("setFlagTx: %v", err)
	}
	tx.Commit()
	has, _ := model.HasInstanceFlag(ctx, 1, model.InstanceFlagStaleGroup)
	if !has {
		t.Error("flag should be set")
	}
}

func TestDelFlagTx(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	model.AddInstanceFlag(ctx, 1, model.InstanceFlagStaleGroup, "")
	tx := model.DB(ctx).Begin()
	if err := delFlagTx(tx, 1, model.InstanceFlagStaleGroup); err != nil {
		tx.Rollback()
		t.Fatalf("delFlagTx: %v", err)
	}
	tx.Commit()
	has, _ := model.HasInstanceFlag(ctx, 1, model.InstanceFlagStaleGroup)
	if has {
		t.Error("flag should be deleted")
	}
}

func TestClearStaleFlagsTx(t *testing.T) {
	defer setupCoverageTest(t)()
	ctx := context.Background()
	model.AddInstanceFlag(ctx, 1, model.InstanceFlagStaleGroup, "")
	model.AddInstanceFlag(ctx, 1, model.InstanceFlagPendingUserAction, "")
	model.AddInstanceFlag(ctx, 1, model.InstanceFlagAllowMigrate, "")
	tx := model.DB(ctx).Begin()
	if err := clearStaleFlagsTx(tx, 1); err != nil {
		tx.Rollback()
		t.Fatalf("clearStaleFlagsTx: %v", err)
	}
	tx.Commit()
	flags, _ := model.GetInstanceFlags(ctx, 1)
	if len(flags) != 0 {
		t.Errorf("want 0 flags, got %v", flags)
	}
}
