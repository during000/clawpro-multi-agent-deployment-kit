package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
)

// admin_stale_instances_action_options_test.go — 纯函数单测 + handler 基础校验。
//
// 覆盖：
//   - buildActionOptNoGroup：nil safe、username 映射、按 user_id 聚合
//   - buildActionOptUserRemoved：nil safe、handover OR 逻辑、按 user_id 聚合
//   - buildActionOptSubtree：按 group_id 聚合
//   - collectUintGroupIDs：去重与过滤 0
//   - loadGroupMemberCounts：0 个 groupIDs 时立即返回
//   - Handler：Method Not Allowed / 未认证返回 401 / BadJSON / 两列表均空

// ── no_group ──────────────────────────────────────────────────────────────────

func TestBuildActionOptNoGroup_NilSafe(t *testing.T) {
	ng := buildActionOptNoGroup(nil, map[uint]string{}, map[uint][]actionOptUserGroupBrief{})
	if ng.Options == nil {
		t.Error("options should not be nil (must be [])")
	}
	if len(ng.Options) != 0 {
		t.Errorf("options should be empty, got %d", len(ng.Options))
	}
}

func TestBuildActionOptNoGroup_MapsUsername(t *testing.T) {
	rows := []staleActionInstRow{
		{ID: 1, Name: "inst-1", UserID: 10, GroupID: 0},
		{ID: 2, Name: "inst-2", UserID: 20, GroupID: 0},
	}
	usernameMap := map[uint]string{10: "alice", 20: "bob"}
	ng := buildActionOptNoGroup(rows, usernameMap, map[uint][]actionOptUserGroupBrief{})
	if len(ng.Options) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(ng.Options))
	}
	if ng.Options[0].Username != "alice" {
		t.Errorf("expected alice, got %s", ng.Options[0].Username)
	}
	if ng.Options[1].Username != "bob" {
		t.Errorf("expected bob, got %s", ng.Options[1].Username)
	}
	// inline options must be fixed
	if ng.Options[0].Migrate != true || ng.Options[0].Handover != false ||
		ng.Options[0].PendingUser != true || ng.Options[0].PendingUserAllowMigrate != true ||
		ng.Options[0].PendingUserAllowHandover != false || ng.Options[0].ArchiveStop != true {
		t.Error("no_group entry options should have fixed values: migrate=T handover=F pending_user=T allow_migrate=T allow_handover=F archive_stop=T")
	}
}

func TestBuildActionOptNoGroup_GroupsByUserID(t *testing.T) {
	rows := []staleActionInstRow{
		{ID: 1, Name: "inst-1", UserID: 10, GroupID: 0},
		{ID: 2, Name: "inst-2", UserID: 10, GroupID: 0}, // same user
		{ID: 3, Name: "inst-3", UserID: 20, GroupID: 0},
	}
	ng := buildActionOptNoGroup(rows, map[uint]string{10: "alice", 20: "bob"}, map[uint][]actionOptUserGroupBrief{})
	if len(ng.Options) != 2 {
		t.Fatalf("expected 2 entries (grouped by user), got %d", len(ng.Options))
	}
	if ng.Options[0].UserID != 10 {
		t.Errorf("first entry should be user 10, got %d", ng.Options[0].UserID)
	}
	if len(ng.Options[0].Instances) != 2 {
		t.Errorf("user 10 should have 2 instances, got %d", len(ng.Options[0].Instances))
	}
	if ng.Options[1].UserID != 20 {
		t.Errorf("second entry should be user 20, got %d", ng.Options[1].UserID)
	}
	if len(ng.Options[1].Instances) != 1 {
		t.Errorf("user 20 should have 1 instance, got %d", len(ng.Options[1].Instances))
	}
}

// ── user_removed ──────────────────────────────────────────────────────────────

func TestBuildActionOptUserRemoved_HandoverAvailable(t *testing.T) {
	rows := []staleActionInstRow{
		{ID: 10, Name: "a", UserID: 1, GroupID: 100},
		{ID: 11, Name: "b", UserID: 2, GroupID: 200},
	}
	usernameMap := map[uint]string{1: "u1", 2: "u2"}
	memberCounts := map[uint]int{100: 3, 200: 0}
	groupPaths := map[uint]string{100: "root/g100", 200: "root/g200"}
	userGroupsMap := map[uint][]actionOptUserGroupBrief{}
	ur := buildActionOptUserRemoved(rows, usernameMap, memberCounts, groupPaths, userGroupsMap)

	if len(ur.Options) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(ur.Options))
	}
	// user 1: group 100 has members → handover=true
	if !ur.Options[0].Handover {
		t.Error("user 1 in group 100 (has members) should have handover=true")
	}
	if !ur.Options[0].PendingUserAllowHandover {
		t.Error("pending_user_allow_handover should equal handover (true)")
	}
	// user 2: group 200 has no members → handover=false
	if ur.Options[1].Handover {
		t.Error("user 2 in group 200 (no members) should have handover=false")
	}
	if ur.Options[1].PendingUserAllowHandover {
		t.Error("pending_user_allow_handover should equal handover (false)")
	}
}

func TestBuildActionOptUserRemoved_NilSafe(t *testing.T) {
	ur := buildActionOptUserRemoved(nil, map[uint]string{}, map[uint]int{}, map[uint]string{}, map[uint][]actionOptUserGroupBrief{})
	if ur.Options == nil {
		t.Error("options should not be nil")
	}
	if len(ur.Options) != 0 {
		t.Errorf("options should be empty, got %d", len(ur.Options))
	}
}

func TestBuildActionOptUserRemoved_HandoverOrLogic(t *testing.T) {
	// same user, two instances: one in group with members, one without
	rows := []staleActionInstRow{
		{ID: 10, Name: "a", UserID: 5, GroupID: 100},
		{ID: 11, Name: "b", UserID: 5, GroupID: 200},
	}
	memberCounts := map[uint]int{100: 3, 200: 0}
	groupPaths := map[uint]string{100: "g100", 200: "g200"}
	userGroupsMap := map[uint][]actionOptUserGroupBrief{}
	ur := buildActionOptUserRemoved(rows, map[uint]string{5: "user5"}, memberCounts, groupPaths, userGroupsMap)

	if len(ur.Options) != 1 {
		t.Fatalf("expected 1 entry (same user), got %d", len(ur.Options))
	}
	// OR logic: at least one group has handover_available → entry.Handover=true
	if !ur.Options[0].Handover {
		t.Error("entry handover should be true (OR of group handover_available)")
	}
	// two groups (grouped by agent's group_id)
	if len(ur.Options[0].Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(ur.Options[0].Groups))
	}
	// group 100 has members → handover_available=true
	g0 := ur.Options[0].Groups[0]
	if g0.GroupID != 100 || !g0.HandoverAvailable {
		t.Errorf("group 0 should be group_id=100 handover=true, got id=%d avail=%v", g0.GroupID, g0.HandoverAvailable)
	}
	if len(g0.Instances) != 1 {
		t.Errorf("group 100 should have 1 instance, got %d", len(g0.Instances))
	}
	// group 200 no members → handover_available=false
	g1 := ur.Options[0].Groups[1]
	if g1.GroupID != 200 || g1.HandoverAvailable {
		t.Errorf("group 1 should be group_id=200 handover=false, got id=%d avail=%v", g1.GroupID, g1.HandoverAvailable)
	}
}

// ── subtree ───────────────────────────────────────────────────────────────────

func TestBuildActionOptSubtree_GroupedByGroupID(t *testing.T) {
	rows := []staleActionInstRow{
		{ID: 1, Name: "a", UserID: 10, GroupID: 5},
		{ID: 2, Name: "b", UserID: 11, GroupID: 5},
		{ID: 3, Name: "c", UserID: 12, GroupID: 6},
	}
	usernameMap := map[uint]string{10: "u10", 11: "u11", 12: "u12"}
	groupPaths := map[uint]string{5: "g5", 6: "g6"}
	st := buildActionOptSubtree(rows, usernameMap, groupPaths)

	if len(st.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(st.Groups))
	}
	if st.Groups[0].GroupID != 5 {
		t.Errorf("expected first group=5, got %d", st.Groups[0].GroupID)
	}
	if len(st.Groups[0].Instances) != 2 {
		t.Errorf("group 5 should have 2 instances, got %d", len(st.Groups[0].Instances))
	}
	if st.Groups[1].GroupID != 6 {
		t.Errorf("expected second group=6, got %d", st.Groups[1].GroupID)
	}
	if len(st.Groups[1].Instances) != 1 {
		t.Errorf("group 6 should have 1 instance, got %d", len(st.Groups[1].Instances))
	}
}

func TestBuildActionOptSubtree_NilSafe(t *testing.T) {
	st := buildActionOptSubtree(nil, map[uint]string{}, map[uint]string{})
	if st.Groups == nil {
		t.Error("groups should not be nil")
	}
}

// ── collectUintGroupIDs ───────────────────────────────────────────────────────

func TestCollectUintGroupIDs_DeduplicatesAndSkipsZero(t *testing.T) {
	rows := []staleActionInstRow{
		{GroupID: 0},
		{GroupID: 1},
		{GroupID: 2},
		{GroupID: 1}, // duplicate
		{GroupID: 0}, // skip
	}
	got := collectUintGroupIDs(rows)
	if len(got) != 2 {
		t.Errorf("expected 2 unique non-zero group IDs, got %d: %v", len(got), got)
	}
}

func TestCollectUintGroupIDs_Empty(t *testing.T) {
	got := collectUintGroupIDs(nil)
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

// ── loadGroupMemberCounts 零输入短路 ──────────────────────────────────────────

func TestLoadGroupMemberCounts_EmptyGroupIDs(t *testing.T) {
	result := loadGroupMemberCounts(nil, []uint{})
	if result == nil {
		t.Error("should return non-nil map")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

// ── filterRowsExcludingIDs ────────────────────────────────────────────────────

func TestFilterRowsExcludingIDs(t *testing.T) {
	rows := []staleActionInstRow{
		{ID: 1, Name: "a"},
		{ID: 2, Name: "b"},
		{ID: 3, Name: "c"},
	}
	exclude := map[uint]struct{}{2: {}}
	got := filterRowsExcludingIDs(rows, exclude)
	if len(got) != 2 {
		t.Fatalf("want 2, got %d", len(got))
	}
	if got[0].ID != 1 || got[1].ID != 3 {
		t.Errorf("want IDs [1,3], got [%d,%d]", got[0].ID, got[1].ID)
	}
}

func TestFilterRowsExcludingIDs_EmptyExclude(t *testing.T) {
	rows := []staleActionInstRow{{ID: 1}, {ID: 2}}
	got := filterRowsExcludingIDs(rows, nil)
	if len(got) != 2 {
		t.Errorf("empty exclude → want 2, got %d", len(got))
	}
}

func TestFilterRowsExcludingIDs_ExcludeAll(t *testing.T) {
	rows := []staleActionInstRow{{ID: 1}, {ID: 2}}
	exclude := map[uint]struct{}{1: {}, 2: {}}
	got := filterRowsExcludingIDs(rows, exclude)
	if len(got) != 0 {
		t.Errorf("exclude all → want 0, got %d", len(got))
	}
}



// ── Handler 基础校验 ───────────────────────────────────────────────────────────

func TestHandleAdminStaleInstancesActionOptions_MethodNotAllowed(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/admin/stale-instances/action-options", nil)
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesActionOptions(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleAdminStaleInstancesActionOptions_Unauthenticated(t *testing.T) {
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret"))
	defer func() { Store = origStore }()

	body, _ := json.Marshal(instancesByUserGroupRequest{
		UserGroupIDs: []userGroupPair{{UserID: 1, GroupID: 2}},
	})
	r := httptest.NewRequest(http.MethodPost, "/admin/stale-instances/action-options", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesActionOptions(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleAdminStaleInstancesActionOptions_BadJSON(t *testing.T) {
	origToken := AdminToken
	AdminToken = "test-tok"
	defer func() { AdminToken = origToken }()

	r := httptest.NewRequest(http.MethodPost, "/admin/stale-instances/action-options",
		bytes.NewBufferString("not-json"))
	r.Header.Set("Authorization", "Bearer test-tok")
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesActionOptions(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleAdminStaleInstancesActionOptions_BothEmpty(t *testing.T) {
	origToken := AdminToken
	AdminToken = "test-tok"
	defer func() { AdminToken = origToken }()
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret"))
	defer func() { Store = origStore }()

	body, _ := json.Marshal(instancesByUserGroupRequest{
		UserGroupIDs: []userGroupPair{},
		GroupIDs:     []uint{},
	})
	r := httptest.NewRequest(http.MethodPost, "/admin/stale-instances/action-options", bytes.NewReader(body))
	r.Header.Set("Authorization", "Bearer test-tok")
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleAdminStaleInstancesActionOptions(w, r)
	// 两参数均空时不再返回 400；no_group 场景始终全量扫描，应返回 200。
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for both empty (no_group always scans), got %d, body=%s", w.Code, w.Body.String())
	}
}
