package usergroup

import (
	"testing"

	"hatchery/model"
)

// TestSortDirectGroups 覆盖 /admin/user-groups/members 响应中 direct_groups 的排序规则：
//  1. source='oneid_dept' 在前，其它（manual）在后
//  2. 主部门(is_main=true) 优先
//  3. 层级浅到深（full_path 中 "/" 数量少的在前）
//  4. 同层级按创建时间升序
func TestSortDirectGroups(t *testing.T) {
	t.Run("MainOneIDDeptFirstEvenIfFullPathLater", func(t *testing.T) {
		// 典型场景：张三同时属于"后端中心"(副) 和"基础架构组"(主)。
		// 仅按 full_path 排序时，"后端中心" < "基础架构组/..."，主部门会被压在后面。
		// 期望：is_main=true 的"基础架构组"排第一。
		in := []MemberDirectGroupRef{
			{ID: 65, Name: "后端中心", FullPath: "OpenClaw企业/技术部/后端中心", Source: model.GroupSourceOneIDDept, IsMain: false},
			{ID: 70, Name: "基础架构组", FullPath: "OpenClaw企业/技术部/后端中心/基础架构组", Source: model.GroupSourceOneIDDept, IsMain: true},
		}
		got := sortDirectGroups(in)
		if got[0].ID != 70 || !got[0].IsMain {
			t.Fatalf("期望主部门(id=70)排首位，实际 order=%v", ids(got))
		}
		if got[1].ID != 65 {
			t.Fatalf("期望副部门(id=65)排第二，实际 order=%v", ids(got))
		}
	})

	t.Run("OneIDDeptBeforeManual", func(t *testing.T) {
		// manual 即使 full_path 字典序靠前，也必须排在 oneid_dept 之后。
		in := []MemberDirectGroupRef{
			{ID: 3, Name: "前端组", FullPath: "研发中心/前端组", Source: model.GroupSourceManual},
			{ID: 62, Name: "技术部", FullPath: "OpenClaw企业/技术部", Source: model.GroupSourceOneIDDept, IsMain: true},
		}
		got := sortDirectGroups(in)
		if got[0].Source != model.GroupSourceOneIDDept {
			t.Fatalf("期望 oneid_dept 优先，实际 %+v", got)
		}
		if got[1].Source != model.GroupSourceManual {
			t.Fatalf("期望 manual 后置，实际 %+v", got)
		}
	})

	t.Run("OneIDDept_MainFirst_ThenByDepthAndCreatedAt", func(t *testing.T) {
		// 3 个 oneid_dept：1 个主 + 2 个副；副部门按层级浅到深排序。
		in := []MemberDirectGroupRef{
			{ID: 73, Name: "移动应用组", FullPath: "OpenClaw企业/技术部/前端中心/移动应用组", Source: model.GroupSourceOneIDDept, IsMain: false},
			{ID: 72, Name: "Web 应用组", FullPath: "OpenClaw企业/技术部/前端中心/Web 应用组", Source: model.GroupSourceOneIDDept, IsMain: true},
			{ID: 66, Name: "前端中心", FullPath: "OpenClaw企业/技术部/前端中心", Source: model.GroupSourceOneIDDept, IsMain: false},
		}
		got := sortDirectGroups(in)
		want := []uint{72, 66, 73} // 主(72) → 前端中心(66, depth=2) → 移动应用组(73, depth=3)
		if !equalIDs(got, want) {
			t.Fatalf("want=%v got=%v", want, ids(got))
		}
	})

	t.Run("Manual_IsMainIgnored_PureFullPath", func(t *testing.T) {
		// manual 分组无"主分组"概念；即使 is_main 标记了也应当被忽略，纯粹按 full_path 排。
		in := []MemberDirectGroupRef{
			{ID: 3, Name: "前端组", FullPath: "研发中心/前端组", Source: model.GroupSourceManual, IsMain: false},
			{ID: 1, Name: "研发中心", FullPath: "研发中心", Source: model.GroupSourceManual, IsMain: true},
		}
		got := sortDirectGroups(in)
		// full_path："研发中心" < "研发中心/前端组"
		if got[0].ID != 1 || got[1].ID != 3 {
			t.Fatalf("manual 应按 full_path 排，want=[1,3] got=%v", ids(got))
		}
	})

	t.Run("MixedAllThreeTiers", func(t *testing.T) {
		// oneid_dept 主(A) · oneid_dept 副(B,C) · manual(D,E) 混合。
		// 同层级按 CreatedAt 排序。
		in := []MemberDirectGroupRef{
			{ID: 5, Name: "manual-B", FullPath: "B", Source: model.GroupSourceManual, CreatedAt: "2026-01-02T00:00:00Z"},
			{ID: 4, Name: "manual-A", FullPath: "A", Source: model.GroupSourceManual, CreatedAt: "2026-01-01T00:00:00Z"},
			{ID: 3, Name: "dept-C", FullPath: "Z/dept-C", Source: model.GroupSourceOneIDDept, IsMain: false, CreatedAt: "2026-01-03T00:00:00Z"},
			{ID: 2, Name: "dept-B", FullPath: "M/dept-B", Source: model.GroupSourceOneIDDept, IsMain: false, CreatedAt: "2026-01-02T00:00:00Z"},
			{ID: 1, Name: "dept-A", FullPath: "Z/dept-A", Source: model.GroupSourceOneIDDept, IsMain: true, CreatedAt: "2026-01-01T00:00:00Z"},
		}
		got := sortDirectGroups(in)
		want := []uint{1, 2, 3, 4, 5}
		if !equalIDs(got, want) {
			t.Fatalf("want=%v got=%v", want, ids(got))
		}
	})

	t.Run("Empty", func(t *testing.T) {
		got := sortDirectGroups([]MemberDirectGroupRef{})
		if len(got) != 0 {
			t.Fatalf("want empty, got %v", got)
		}
	})

	t.Run("SingleItem", func(t *testing.T) {
		in := []MemberDirectGroupRef{
			{ID: 1, Name: "x", FullPath: "x", Source: model.GroupSourceManual},
		}
		got := sortDirectGroups(in)
		if len(got) != 1 || got[0].ID != 1 {
			t.Fatalf("want single [1], got %v", ids(got))
		}
	})

	t.Run("TwoMainDepts_StableByCreatedAt", func(t *testing.T) {
		// 数据不一致 / 边界：两个都标了 is_main=true，同层级按 CreatedAt 排列。
		in := []MemberDirectGroupRef{
			{ID: 2, Name: "B", FullPath: "B", Source: model.GroupSourceOneIDDept, IsMain: true, CreatedAt: "2026-01-02T00:00:00Z"},
			{ID: 1, Name: "A", FullPath: "A", Source: model.GroupSourceOneIDDept, IsMain: true, CreatedAt: "2026-01-01T00:00:00Z"},
		}
		got := sortDirectGroups(in)
		want := []uint{1, 2}
		if !equalIDs(got, want) {
			t.Fatalf("want=%v got=%v", want, ids(got))
		}
	})
}

func ids(list []MemberDirectGroupRef) []uint {
	out := make([]uint, len(list))
	for i, g := range list {
		out[i] = g.ID
	}
	return out
}

func equalIDs(list []MemberDirectGroupRef, want []uint) bool {
	if len(list) != len(want) {
		return false
	}
	for i := range list {
		if list[i].ID != want[i] {
			return false
		}
	}
	return true
}
