package controller

import (
	"testing"
)

// admin_instances_group_check_test.go —— user_group_mismatch / has_config_drift 计算的纯函数单测。
//
// 走 DB 的批量查询（loadUserGroupMemberships / loadInstancesByIDs）由集成路径覆盖；
// 这里只验证不依赖 DB 的判定逻辑：
//   - computeUserGroupMismatch：4 种场景（正常/A/B/C）
//   - anyDifferentRow：不同 status 的短路检查
//   - collectUniqueUserIDs / collectUniqueGroupIDs / collectInstanceIDs：去重与 group_id=0 保留
//   - enrichAdminInstancesWithGroupCheck 空短路（items=0 或两开关都 false）

func TestComputeUserGroupMismatch(t *testing.T) {
	// user 加入了 group 1,2,3
	userGroups := map[uint]map[uint]struct{}{
		42: {1: {}, 2: {}, 3: {}},
		99: {}, // 未分组用户
	}
	cases := []struct {
		name    string
		userID  uint
		groupID uint
		want    bool
	}{
		{"in_group_normal", 42, 1, false},
		{"scenario_A_moved_out", 42, 999, true},                 // user 42 不在 group 999
		{"scenario_C_ungrouped_but_now_joined", 42, 0, true},    // user 已加入组但实例仍 group_id=0
		{"user_no_groups_instance_ungrouped", 99, 0, false},     // 用户和实例都未分组 → 匹配
		{"user_no_groups_instance_grouped", 99, 5, true},        // 用户未加入任何组 → 不匹配
		{"unknown_user_grouped", 12345, 1, true},                // user 不在 map → mismatch
		{"unknown_user_ungrouped", 12345, 0, false},             // 未加入任何组 & 实例未分组
	}
	for _, c := range cases {
		got := computeUserGroupMismatch(c.userID, c.groupID, userGroups)
		if got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestAnyDifferentRow(t *testing.T) {
	cases := []struct {
		name string
		rows []instanceConfigRow
		want bool
	}{
		{"empty", nil, false},
		{"all_same", []instanceConfigRow{{Status: "same"}, {Status: "same"}}, false},
		{"contained_in_target_only", []instanceConfigRow{{Status: "contained_in_target"}, {Status: "same"}}, false},
		{"not_check_only", []instanceConfigRow{{Status: "not_check"}, {Status: "same"}}, false},
		{"has_different", []instanceConfigRow{{Status: "same"}, {Status: "different"}, {Status: "same"}}, true},
		{"different_first", []instanceConfigRow{{Status: "different"}}, true},
	}
	for _, c := range cases {
		if got := anyDifferentRow(c.rows); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestCollectUniqueUserIDs(t *testing.T) {
	items := []groupCheckItem{
		{ID: 1, UserID: 100},
		{ID: 2, UserID: 100}, // 去重
		{ID: 3, UserID: 200},
		{ID: 4, UserID: 0}, // UserID=0 忽略
		{ID: 5, UserID: 300},
	}
	got := collectUniqueUserIDs(items)
	if len(got) != 3 {
		t.Fatalf("want 3 unique users, got %d (%v)", len(got), got)
	}
	// 保序：先出现的先加入
	want := []uint{100, 200, 300}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d] got %d want %d", i, got[i], w)
		}
	}
}

func TestCollectUniqueGroupIDs(t *testing.T) {
	items := []groupCheckItem{
		{ID: 1, GroupID: 5},
		{ID: 2, GroupID: 0}, // 未分组也保留（drift 计算需要走全局兜底视图）
		{ID: 3, GroupID: 5},
		{ID: 4, GroupID: 7},
		{ID: 5, GroupID: 0},
	}
	got := collectUniqueGroupIDs(items)
	if len(got) != 3 {
		t.Fatalf("want 3 unique groups, got %d (%v)", len(got), got)
	}
	want := []uint{5, 0, 7}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d] got %d want %d", i, got[i], w)
		}
	}
}

func TestCollectInstanceIDs(t *testing.T) {
	items := []groupCheckItem{{ID: 10}, {ID: 20}, {ID: 30}}
	got := collectInstanceIDs(items)
	if len(got) != 3 || got[0] != 10 || got[2] != 30 {
		t.Errorf("got %v", got)
	}
}

// 两开关全 false 时不做任何改动（items 里的字段保持初始 false）。
func TestEnrichWithGroupCheck_BothOff_NoOp(t *testing.T) {
	items := []groupCheckItem{{ID: 1, UserID: 100, GroupID: 5, UserGroupMismatch: false, HasConfigDrift: false}}
	enrichAdminInstancesWithGroupCheck(nil, items, false, false)
	if items[0].UserGroupMismatch || items[0].HasConfigDrift {
		t.Errorf("both off should keep fields false, got mismatch=%v drift=%v", items[0].UserGroupMismatch, items[0].HasConfigDrift)
	}
}

// 空 items 直接返回，不 panic。
func TestEnrichWithGroupCheck_EmptyItems(t *testing.T) {
	enrichAdminInstancesWithGroupCheck(nil, nil, true, true)
	enrichAdminInstancesWithGroupCheck(nil, []groupCheckItem{}, true, true)
}
