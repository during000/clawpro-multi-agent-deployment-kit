package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

func TestPreCheckTargetExistence(t *testing.T) {
	groups := map[uint]bool{10: true, 20: true}
	users := map[uint]bool{100: true}

	uintPtr := func(v uint) *uint { return &v }

	cases := []struct {
		name        string
		item        applyActionItem
		wantErrMsg  string
		wantBlocked bool
	}{
		{
			name:        "migrate_target_group_exists",
			item:        applyActionItem{Action: StaleActionMigrate, TargetGroupID: uintPtr(10)},
			wantErrMsg:  "",
			wantBlocked: false,
		},
		{
			name:        "migrate_target_group_missing",
			item:        applyActionItem{Action: StaleActionMigrate, TargetGroupID: uintPtr(999)},
			wantErrMsg:  "target_group_not_found",
			wantBlocked: true,
		},
		{
			name:        "migrate_target_zero_skipped",
			item:        applyActionItem{Action: StaleActionMigrate, TargetGroupID: uintPtr(0)},
			wantErrMsg:  "",
			wantBlocked: false,
		},
		{
			name:        "handover_user_exists",
			item:        applyActionItem{Action: StaleActionHandover, TargetUserID: 100},
			wantErrMsg:  "",
			wantBlocked: false,
		},
		{
			name:        "handover_user_missing",
			item:        applyActionItem{Action: StaleActionHandover, TargetUserID: 9999},
			wantErrMsg:  "target_user_not_found",
			wantBlocked: true,
		},
		{
			name:        "handover_user_zero_skipped",
			item:        applyActionItem{Action: StaleActionHandover, TargetUserID: 0},
			wantErrMsg:  "",
			wantBlocked: false,
		},
		{
			// pending_user 不读这两个字段，即使写错也应通过预校验
			name:        "pending_user_target_group_set_ignored",
			item:        applyActionItem{Action: StaleActionPendingUser, TargetGroupID: uintPtr(999)},
			wantErrMsg:  "",
			wantBlocked: false,
		},
		{
			name:        "archive_stop_passthrough",
			item:        applyActionItem{Action: StaleActionArchiveStop, TargetGroupID: uintPtr(999), TargetUserID: 9999},
			wantErrMsg:  "",
			wantBlocked: false,
		},
		{
			// migrate 走 group 路径，TargetUserID 是错的也不查
			name:        "migrate_user_id_ignored",
			item:        applyActionItem{Action: StaleActionMigrate, TargetGroupID: uintPtr(10), TargetUserID: 9999},
			wantErrMsg:  "",
			wantBlocked: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			msg, blocked := preCheckTargetExistence(c.item, groups, users)
			if blocked != c.wantBlocked {
				t.Fatalf("blocked got %v want %v", blocked, c.wantBlocked)
			}
			if msg != c.wantErrMsg {
				t.Fatalf("msg got %q want %q", msg, c.wantErrMsg)
			}
		})
	}
}

func TestResolveHandoverTargetGroupID(t *testing.T) {
	cases := []struct {
		name             string
		targetGroups     []uint
		requestedGroupID uint
		groupIDSpecified bool
		currentGroupID   uint
		wantNewGroupID   uint
		wantErrCode      string
	}{
		// ── 未指定 target_group_id（同分组移交） ──
		{
			name:             "same_group_no_target_groups_ok",
			targetGroups:     []uint{},
			groupIDSpecified: false,
			currentGroupID:   0,
			wantNewGroupID:   0,
			wantErrCode:      "",
		},
		{
			name:             "same_group_no_target_groups_but_instance_has_group_rejected",
			targetGroups:     []uint{},
			groupIDSpecified: false,
			currentGroupID:   5,
			wantErrCode:      "target_user_not_in_same_group",
		},
		{
			name:             "same_group_target_in_current_group_ok",
			targetGroups:     []uint{3, 5, 7},
			groupIDSpecified: false,
			currentGroupID:   5,
			wantNewGroupID:   5,
			wantErrCode:      "",
		},
		{
			name:             "same_group_target_not_in_current_group_rejected",
			targetGroups:     []uint{3, 7},
			groupIDSpecified: false,
			currentGroupID:   5,
			wantErrCode:      "target_user_not_in_same_group",
		},
		{
			name:             "same_group_single_group_matches_current_ok",
			targetGroups:     []uint{7},
			groupIDSpecified: false,
			currentGroupID:   7,
			wantNewGroupID:   7,
			wantErrCode:      "",
		},
		{
			name:             "same_group_single_group_not_current_rejected",
			targetGroups:     []uint{7},
			groupIDSpecified: false,
			currentGroupID:   3,
			wantErrCode:      "target_user_not_in_same_group",
		},
		// ── 显式指定 target_group_id ──
		// 0 groups
		{
			name:             "no_groups_no_request",
			targetGroups:     []uint{},
			requestedGroupID: 0,
			groupIDSpecified: true,
			wantNewGroupID:   0,
			wantErrCode:      "",
		},
		{
			name:             "no_groups_request_nonzero_rejected",
			targetGroups:     []uint{},
			requestedGroupID: 5,
			groupIDSpecified: true,
			wantErrCode:      "target_user_has_no_groups_target_group_id_must_be_zero",
		},
		// 1 group
		{
			name:             "single_group_auto_assign",
			targetGroups:     []uint{7},
			requestedGroupID: 0,
			groupIDSpecified: true,
			wantNewGroupID:   7,
		},
		{
			name:             "single_group_explicit_match",
			targetGroups:     []uint{7},
			requestedGroupID: 7,
			groupIDSpecified: true,
			wantNewGroupID:   7,
		},
		{
			name:             "single_group_explicit_mismatch_rejected",
			targetGroups:     []uint{7},
			requestedGroupID: 99,
			groupIDSpecified: true,
			wantErrCode:      "target_group_id_not_in_target_user_groups",
		},
		// 2+ groups
		{
			name:             "multi_group_no_request_rejected",
			targetGroups:     []uint{3, 5, 7},
			requestedGroupID: 0,
			groupIDSpecified: true,
			wantErrCode:      "target_group_id_required_for_multi_group_target",
		},
		{
			name:             "multi_group_valid_request",
			targetGroups:     []uint{3, 5, 7},
			requestedGroupID: 5,
			groupIDSpecified: true,
			wantNewGroupID:   5,
		},
		{
			name:             "multi_group_invalid_request_rejected",
			targetGroups:     []uint{3, 5, 7},
			requestedGroupID: 99,
			groupIDSpecified: true,
			wantErrCode:      "target_group_id_not_in_target_user_groups",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, code := resolveHandoverTargetGroupID(c.targetGroups, c.requestedGroupID, c.groupIDSpecified, c.currentGroupID)
			if code != c.wantErrCode {
				t.Fatalf("err code mismatch: got %q want %q", code, c.wantErrCode)
			}
			if c.wantErrCode == "" && got != c.wantNewGroupID {
				t.Fatalf("group id mismatch: got %d want %d", got, c.wantNewGroupID)
			}
		})
	}
}

// ─── handler-level 校验 ─────────────────────────────────────────────────────

func setupApplyHandlerTest(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserGroup{}, &model.Instance{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	origStore := Store
	origToken := AdminToken
	AdminToken = "test-admin-token"
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	return func() {
		origDB()
		Store = origStore
		AdminToken = origToken
	}
}

func applyReq(t *testing.T, body interface{}) *http.Request {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/stale-instances/apply", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// 超过 staleApplyMaxBatch 应返回 400 + 含 "数量超过上限" 文案。
func TestHandleAdminStaleInstancesApply_ExceedsMaxBatch(t *testing.T) {
	defer setupApplyHandlerTest(t)()

	ids := make([]uint, staleApplyMaxBatch+1)
	for i := range ids {
		ids[i] = uint(i + 1)
	}
	// 用 archive_stop 避开 migrate 的 target_group_id 必填校验。
	req := applyReq(t, map[string]interface{}{
		"trigger_source": "user_edit",
		"actions":        []map[string]interface{}{{"action": "archive_stop", "instance_ids": ids}},
	})
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesApply(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status got %d want 400, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "数量超过上限") {
		t.Errorf("expect '数量超过上限' in body, got %s", w.Body.String())
	}
}

// 多个 actions 的 instance_ids 相加超 500 也应触发 400（按总和校验）。
func TestHandleAdminStaleInstancesApply_SumAcrossActionsExceedsMax(t *testing.T) {
	defer setupApplyHandlerTest(t)()

	half := staleApplyMaxBatch/2 + 1
	mkIDs := func(start, n int) []uint {
		out := make([]uint, n)
		for i := range out {
			out[i] = uint(start + i)
		}
		return out
	}
	req := applyReq(t, map[string]interface{}{
		"trigger_source": "user_edit",
		"actions": []map[string]interface{}{
			{"action": "archive_stop", "instance_ids": mkIDs(1, half)},
			{"action": "archive_stop", "instance_ids": mkIDs(10000, half)},
		},
	})
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesApply(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status got %d want 400, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "数量超过上限") {
		t.Errorf("expect '数量超过上限' in body, got %s", w.Body.String())
	}
}

// 任一 action 枚举值非法应返回 400，不进入 engine.run。
func TestHandleAdminStaleInstancesApply_InvalidActionEnum(t *testing.T) {
	defer setupApplyHandlerTest(t)()

	req := applyReq(t, map[string]interface{}{
		"trigger_source": "user_edit",
		"actions": []map[string]interface{}{
			{"action": "delete", "instance_ids": []uint{1}}, // delete 不在白名单
		},
	})
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesApply(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status got %d want 400, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "actions[0].action") {
		t.Errorf("expect 'actions[0].action' in body for fail-fast hint, got %s", w.Body.String())
	}
}

// migrate 必须显式给 target_group_id（即便是 0）；省略 → 400。
// 这是对"用户漏传 target_group_id 被 noop 吞掉"问题的核心修复。
func TestHandleAdminStaleInstancesApply_MigrateMissingTargetGroupID(t *testing.T) {
	defer setupApplyHandlerTest(t)()

	req := applyReq(t, map[string]interface{}{
		"trigger_source": "user_edit",
		"actions": []map[string]interface{}{
			{"action": "migrate", "instance_ids": []uint{1}},
		},
	})
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesApply(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status got %d want 400, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "actions[0].target_group_id") {
		t.Errorf("expect 'actions[0].target_group_id' in body, got %s", w.Body.String())
	}
}

// migrate 显式 target_group_id=0（场景 B 的合法形态）应当通过 handler 校验，
// 不被"必填"误拦。
func TestHandleAdminStaleInstancesApply_MigrateExplicitZeroTargetGroupAllowed(t *testing.T) {
	defer setupApplyHandlerTest(t)()

	req := applyReq(t, map[string]interface{}{
		"trigger_source": "user_edit",
		"actions": []map[string]interface{}{
			{"action": "migrate", "instance_ids": []uint{99999}, "target_group_id": 0},
		},
	})
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesApply(w, req)

	// 通过 handler 必填校验 → 进入 engine.run；实例 99999 不存在 → 单条 failed
	// instance_not_found，但 HTTP 200。
	if w.Code != http.StatusOK {
		t.Fatalf("status got %d want 200 (handler 必填校验应通过), body=%s", w.Code, w.Body.String())
	}
}

// handover 不给 target_user_id（=0）→ 400。
func TestHandleAdminStaleInstancesApply_HandoverMissingTargetUserID(t *testing.T) {
	defer setupApplyHandlerTest(t)()

	req := applyReq(t, map[string]interface{}{
		"trigger_source": "user_edit",
		"actions": []map[string]interface{}{
			{"action": "handover", "instance_ids": []uint{1}},
		},
	})
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesApply(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status got %d want 400, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "actions[0].target_user_id") {
		t.Errorf("expect 'actions[0].target_user_id' in body, got %s", w.Body.String())
	}
}

// 任一 action 的 instance_ids 为空 → 400。
func TestHandleAdminStaleInstancesApply_EmptyInstanceIDs(t *testing.T) {
	defer setupApplyHandlerTest(t)()

	req := applyReq(t, map[string]interface{}{
		"trigger_source": "user_edit",
		"actions": []map[string]interface{}{
			{"action": "archive_stop", "instance_ids": []uint{}},
		},
	})
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesApply(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status got %d want 400, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "actions[0].instance_ids") {
		t.Errorf("expect 'actions[0].instance_ids' in body, got %s", w.Body.String())
	}
}

// ─── records 端点分页默认值兜底 ─────────────────────────────────────────

func setupRecordsHandlerTest(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.InstanceChangeGroupRecord{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	origStore := Store
	origToken := AdminToken
	AdminToken = "test-admin-token"
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	return func() {
		origDB()
		Store = origStore
		AdminToken = origToken
	}
}

func recordsReq(path string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

func decodeRecordsResp(t *testing.T, w *httptest.ResponseRecorder) (page, pageSize int) {
	t.Helper()
	var resp struct {
		OK       bool `json:"ok"`
		Page     int  `json:"page"`
		PageSize int  `json:"page_size"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v, body=%s", err, w.Body.String())
	}
	if !resp.OK {
		t.Fatalf("ok=false, body=%s", w.Body.String())
	}
	return resp.Page, resp.PageSize
}

// 不传 page / page_size → 响应回显 page=1, page_size=20。
func TestHandleAdminStaleInstancesRecords_DefaultsWhenOmitted(t *testing.T) {
	defer setupRecordsHandlerTest(t)()
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesRecords(w, recordsReq("/admin/stale-instances/records"))

	if w.Code != http.StatusOK {
		t.Fatalf("status got %d, body=%s", w.Code, w.Body.String())
	}
	page, size := decodeRecordsResp(t, w)
	if page != 1 || size != 20 {
		t.Errorf("got page=%d page_size=%d, want 1/20", page, size)
	}
}

// 显式传 0 也走兜底。
func TestHandleAdminStaleInstancesRecords_DefaultsWhenZero(t *testing.T) {
	defer setupRecordsHandlerTest(t)()
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesRecords(w, recordsReq("/admin/stale-instances/records?page=0&page_size=0"))

	if w.Code != http.StatusOK {
		t.Fatalf("status got %d, body=%s", w.Code, w.Body.String())
	}
	page, size := decodeRecordsResp(t, w)
	if page != 1 || size != 20 {
		t.Errorf("got page=%d page_size=%d, want 1/20", page, size)
	}
}

// page_size > 100 → 截到 100。
func TestHandleAdminStaleInstancesRecords_PageSizeClampedTo100(t *testing.T) {
	defer setupRecordsHandlerTest(t)()
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesRecords(w, recordsReq("/admin/stale-instances/records?page=2&page_size=500"))

	if w.Code != http.StatusOK {
		t.Fatalf("status got %d, body=%s", w.Code, w.Body.String())
	}
	page, size := decodeRecordsResp(t, w)
	if page != 2 || size != 100 {
		t.Errorf("got page=%d page_size=%d, want 2/100", page, size)
	}
}

// 合法值原样保留。
func TestHandleAdminStaleInstancesRecords_ValidValuesPreserved(t *testing.T) {
	defer setupRecordsHandlerTest(t)()
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesRecords(w, recordsReq("/admin/stale-instances/records?page=3&page_size=50"))

	if w.Code != http.StatusOK {
		t.Fatalf("status got %d, body=%s", w.Code, w.Body.String())
	}
	page, size := decodeRecordsResp(t, w)
	if page != 3 || size != 50 {
		t.Errorf("got page=%d page_size=%d, want 3/50", page, size)
	}
}

// group_id=0 应当过滤出 group_id_before=0 OR group_id_after=0 的记录（"未分组"语义），
// 而不是被当成"未传过滤"全量返回。
func TestHandleAdminStaleInstancesRecords_GroupIDZeroFiltersUngrouped(t *testing.T) {
	defer setupRecordsHandlerTest(t)()

	ctx := context.Background()
	// 4 条记录：only id=1 / id=2 涉及 group 0；id=3 / id=4 不涉及。
	seed := []model.InstanceChangeGroupRecord{
		{InstancePK: 100, GroupIDBefore: 0, GroupIDAfter: 5, Action: "migrate", ActorType: "admin", TriggerSource: "user_edit"},   // id=1
		{InstancePK: 100, GroupIDBefore: 5, GroupIDAfter: 0, Action: "migrate", ActorType: "admin", TriggerSource: "user_edit"},   // id=2
		{InstancePK: 100, GroupIDBefore: 5, GroupIDAfter: 7, Action: "migrate", ActorType: "admin", TriggerSource: "user_edit"},   // id=3
		{InstancePK: 100, GroupIDBefore: 7, GroupIDAfter: 9, Action: "migrate", ActorType: "admin", TriggerSource: "user_edit"},   // id=4
	}
	for i := range seed {
		if err := model.CreateICGR(ctx, &seed[i]); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	// 不传 group_id → 4 条全返回
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesRecords(w, recordsReq("/admin/stale-instances/records"))
	if w.Code != http.StatusOK {
		t.Fatalf("baseline: %d, %s", w.Code, w.Body.String())
	}
	var baseline struct {
		Total int `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &baseline)
	if baseline.Total != 4 {
		t.Fatalf("baseline total got %d want 4", baseline.Total)
	}

	// group_id=0 → 应只返回 id=1 / id=2 两条（涉及未分组）
	w = httptest.NewRecorder()
	HandleAdminStaleInstancesRecords(w, recordsReq("/admin/stale-instances/records?group_id=0"))
	if w.Code != http.StatusOK {
		t.Fatalf("filter: %d, %s", w.Code, w.Body.String())
	}
	var filtered struct {
		Total int `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &filtered)
	if filtered.Total != 2 {
		t.Errorf("group_id=0 total got %d want 2 (实际过滤未分组), body=%s", filtered.Total, w.Body.String())
	}

	// group_id=5 → 应返回 id=1 / id=2 / id=3 三条（涉及 group 5）
	w = httptest.NewRecorder()
	HandleAdminStaleInstancesRecords(w, recordsReq("/admin/stale-instances/records?group_id=5"))
	if w.Code != http.StatusOK {
		t.Fatalf("filter: %d", w.Code)
	}
	var nonZero struct {
		Total int `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &nonZero)
	if nonZero.Total != 3 {
		t.Errorf("group_id=5 total got %d want 3", nonZero.Total)
	}
}

// TestActionAllowedInScenario 锁定 (scenario, action) 的可见性矩阵。
//
// 注意 (ScenarioC, handover) = true：早期产品要求"只能移交给同分组下的其他用户"，
// 因此未分组实例（C）不允许 handover；后续产品改为"允许移交给任意用户"，限制
// 取消，未分组实例也应能 handover。这条用例做回归保护，避免再被误改回 false。
func TestActionAllowedInScenario(t *testing.T) {
	cases := []struct {
		scenario string
		action   string
		want     bool
	}{
		// A：用户已不在 instance.group_id 下 — 4 种全允许
		{ScenarioA, StaleActionMigrate, true},
		{ScenarioA, StaleActionHandover, true},
		{ScenarioA, StaleActionPendingUser, true},
		{ScenarioA, StaleActionArchiveStop, true},

		// B：用户已无任何分组 — 4 种全允许
		{ScenarioB, StaleActionMigrate, true},
		{ScenarioB, StaleActionHandover, true},
		{ScenarioB, StaleActionPendingUser, true},
		{ScenarioB, StaleActionArchiveStop, true},

		// C：实例未分组（GroupID==0）— 4 种全允许（产品需求变更后放开 handover）
		{ScenarioC, StaleActionMigrate, true},
		{ScenarioC, StaleActionHandover, true}, // 回归保护：不能被改回 false
		{ScenarioC, StaleActionPendingUser, true},
		{ScenarioC, StaleActionArchiveStop, true},

		// D：group_parent_change 触发 — 仅 migrate（实际 no-op 写记录）
		{ScenarioD, StaleActionMigrate, true},
		{ScenarioD, StaleActionHandover, false},
		{ScenarioD, StaleActionPendingUser, false},
		{ScenarioD, StaleActionArchiveStop, false},

		// Noop：状态已对齐 — handover 例外（强制换主人不受 stale 限制）
		{ScenarioNoop, StaleActionMigrate, false},
		{ScenarioNoop, StaleActionHandover, true},
		{ScenarioNoop, StaleActionPendingUser, false},
		{ScenarioNoop, StaleActionArchiveStop, false},
	}
	for _, c := range cases {
		got := actionAllowedInScenario(c.scenario, c.action)
		if got != c.want {
			t.Errorf("(%s, %s) got %v want %v", c.scenario, c.action, got, c.want)
		}
	}
}

func TestDetectScenario(t *testing.T) {
	cases := []struct {
		name          string
		input         scenarioInput
		triggerSource string
		want          string
	}{
		{"parent_change_always_D", scenarioInput{UserID: 1, GroupID: 5, UserGroupIDs: []uint{5}}, TriggerSourceGroupParentChange, ScenarioD},
		{"parent_change_even_if_noop", scenarioInput{UserID: 1, GroupID: 5, UserGroupIDs: []uint{5}}, TriggerSourceGroupParentChange, ScenarioD},
		{"user_has_no_groups_B", scenarioInput{UserID: 1, GroupID: 5, UserGroupIDs: nil}, TriggerSourceUserEdit, ScenarioB},
		{"group_id_zero_has_groups_C", scenarioInput{UserID: 1, GroupID: 0, UserGroupIDs: []uint{3}}, TriggerSourceUserEdit, ScenarioC},
		{"user_not_in_instance_group_A", scenarioInput{UserID: 1, GroupID: 5, UserGroupIDs: []uint{3, 7}}, TriggerSourceUserEdit, ScenarioA},
		{"user_in_instance_group_noop", scenarioInput{UserID: 1, GroupID: 5, UserGroupIDs: []uint{3, 5, 7}}, TriggerSourceUserEdit, ScenarioNoop},
		{"oneid_sync_uses_normal_logic", scenarioInput{UserID: 1, GroupID: 5, UserGroupIDs: []uint{3}}, TriggerSourceOneIDSync, ScenarioA},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := detectScenario(c.input, c.triggerSource)
			if got != c.want {
				t.Errorf("got %q want %q", got, c.want)
			}
		})
	}
}
