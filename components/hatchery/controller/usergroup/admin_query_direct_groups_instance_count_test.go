package usergroup

import (
	"context"
	"testing"

	"hatchery/model"
)

// TestGetGroupMembersPaged_DirectGroupsInstanceCount 验证 /admin/user-groups/members
// 返回的 members[].direct_groups[].instance_count 字段正确统计每个 (user_id, group_id)
// 下的 instances 数量。
//
// 场景：
//
//	alice 属于 manual组A + oneid组B
//	bob   属于 oneid组B
//	instances:
//	  - alice × A: 2 条
//	  - alice × B: 1 条
//	  - bob   × B: 3 条
//	  - 无主用户 × A: 1 条（不应计入 alice/bob 任何直属组）
func TestGetGroupMembersPaged_DirectGroupsInstanceCount(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// 建两个分组
	gA := model.UserGroup{Name: "A", FullPath: "A", Depth: 0, Source: model.GroupSourceManual}
	gB := model.UserGroup{Name: "B", FullPath: "B", Depth: 0, Source: model.GroupSourceOneIDDept, SourceRef: "D100"}
	if err := model.DB(context.Background()).Create(&gA).Error; err != nil {
		t.Fatalf("create gA: %v", err)
	}
	if err := model.DB(context.Background()).Create(&gB).Error; err != nil {
		t.Fatalf("create gB: %v", err)
	}

	// 建两个用户
	alice := model.User{Username: "alice", Password: "x", Role: "user"}
	bob := model.User{Username: "bob", Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(&alice).Error; err != nil {
		t.Fatalf("create alice: %v", err)
	}
	if err := model.DB(context.Background()).Create(&bob).Error; err != nil {
		t.Fatalf("create bob: %v", err)
	}

	// 成员关系
	members := []model.UserGroupMember{
		{UserGroupID: gA.ID, UserID: alice.ID, Source: model.MemberSourceManual},
		{UserGroupID: gB.ID, UserID: alice.ID, Source: model.MemberSourceOneIDDept, IsMain: true},
		{UserGroupID: gB.ID, UserID: bob.ID, Source: model.MemberSourceOneIDDept, IsMain: true},
	}
	for i := range members {
		if err := model.DB(context.Background()).Create(&members[i]).Error; err != nil {
			t.Fatalf("create member: %v", err)
		}
	}

	// 实例：按 (user_id, group_id) 计数需与断言匹配
	seedInst := func(name string, userID, groupID uint) {
		tk := "sk-" + name
		inst := model.Instance{Name: name, UserID: userID, GroupID: groupID, ProxyToken: &tk}
		if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
			t.Fatalf("create instance %s: %v", name, err)
		}
	}
	seedInst("a-on-A-1", alice.ID, gA.ID)
	seedInst("a-on-A-2", alice.ID, gA.ID)
	seedInst("a-on-B-1", alice.ID, gB.ID)
	seedInst("b-on-B-1", bob.ID, gB.ID)
	seedInst("b-on-B-2", bob.ID, gB.ID)
	seedInst("b-on-B-3", bob.ID, gB.ID)
	seedInst("other-A", 999, gA.ID) // 无主用户，用于校验不串行

	// 查询 B 组成员（两人）
	resp, err := GetGroupMembersPaged(context.Background(), MembersOptions{GroupID: gB.ID, Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("GetGroupMembersPaged: %v", err)
	}
	if len(resp.Members) != 2 {
		t.Fatalf("期望 2 个成员，实际 %d: %+v", len(resp.Members), resp.Members)
	}

	// 查找某成员的某分组 direct_groups 项
	getGroupRef := func(uid, gid uint) *MemberDirectGroupRef {
		for _, m := range resp.Members {
			if m.UserID != uid {
				continue
			}
			for i := range m.DirectGroups {
				if m.DirectGroups[i].ID == gid {
					return &m.DirectGroups[i]
				}
			}
		}
		return nil
	}

	// alice 在 A 组 instance_count=2
	if r := getGroupRef(alice.ID, gA.ID); r == nil {
		t.Errorf("alice × A 找不到 direct_groups 项")
	} else if r.InstanceCount != 2 {
		t.Errorf("alice × A instance_count 期望 2，实际 %d", r.InstanceCount)
	}

	// alice 在 B 组 instance_count=1
	if r := getGroupRef(alice.ID, gB.ID); r == nil {
		t.Errorf("alice × B 找不到 direct_groups 项")
	} else if r.InstanceCount != 1 {
		t.Errorf("alice × B instance_count 期望 1，实际 %d", r.InstanceCount)
	}

	// bob 在 B 组 instance_count=3
	if r := getGroupRef(bob.ID, gB.ID); r == nil {
		t.Errorf("bob × B 找不到 direct_groups 项")
	} else if r.InstanceCount != 3 {
		t.Errorf("bob × B instance_count 期望 3，实际 %d", r.InstanceCount)
	}

	// bob 不在 A 组（无 direct_groups 项）
	if r := getGroupRef(bob.ID, gA.ID); r != nil {
		t.Errorf("bob 不应出现在 A 组的 direct_groups：%+v", r)
	}
}

// TestGetGroupMembersPaged_DirectGroups_NoInstances 用户在某组但该组下 0 实例 → instance_count=0。
func TestGetGroupMembersPaged_DirectGroups_NoInstances(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	g := model.UserGroup{Name: "零实例组", FullPath: "零实例组", Source: model.GroupSourceManual}
	if err := model.DB(context.Background()).Create(&g).Error; err != nil {
		t.Fatal(err)
	}
	u := model.User{Username: "alice", Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.DB(context.Background()).Create(&model.UserGroupMember{
		UserGroupID: g.ID, UserID: u.ID, Source: model.MemberSourceManual,
	}).Error; err != nil {
		t.Fatal(err)
	}

	resp, err := GetGroupMembersPaged(context.Background(), MembersOptions{GroupID: g.ID, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Members) != 1 {
		t.Fatalf("期望 1 个成员，实际 %d", len(resp.Members))
	}
	dg := resp.Members[0].DirectGroups
	if len(dg) != 1 {
		t.Fatalf("期望 1 条 direct_groups，实际 %d", len(dg))
	}
	if dg[0].InstanceCount != 0 {
		t.Errorf("零实例组 instance_count 期望 0，实际 %d", dg[0].InstanceCount)
	}
}

// TestGetGroupMembersPaged_DirectGroups_SoftDeletedInstancesExcluded 已软删
// 的 instance 不应计入 instance_count。
//
// 复现的线上问题：销毁 agent 后 instance_count 仍非零。根因是早期实现使用
// Table("instances") 而非 Model(&model.Instance{}) 聚合，绕开了 GORM
// 的 deleted_at IS NULL 自动过滤。
func TestGetGroupMembersPaged_DirectGroups_SoftDeletedInstancesExcluded(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	g := model.UserGroup{Name: "G", FullPath: "G", Source: model.GroupSourceManual}
	if err := model.DB(context.Background()).Create(&g).Error; err != nil {
		t.Fatal(err)
	}
	u := model.User{Username: "alice", Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.DB(context.Background()).Create(&model.UserGroupMember{
		UserGroupID: g.ID, UserID: u.ID, Source: model.MemberSourceManual,
	}).Error; err != nil {
		t.Fatal(err)
	}

	// 2 条活实例 + 2 条软删实例 → 期望 instance_count=2
	mkInst := func(name string) model.Instance {
		tk := "sk-" + name
		return model.Instance{Name: name, UserID: u.ID, GroupID: g.ID, ProxyToken: &tk}
	}
	live1 := mkInst("live-1")
	live2 := mkInst("live-2")
	dead1 := mkInst("dead-1")
	dead2 := mkInst("dead-2")
	for _, inst := range []*model.Instance{&live1, &live2, &dead1, &dead2} {
		if err := model.DB(context.Background()).Create(inst).Error; err != nil {
			t.Fatalf("create %s: %v", inst.Name, err)
		}
	}
	// 软删 dead-1 / dead-2
	if err := model.DB(context.Background()).Delete(&dead1).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.DB(context.Background()).Delete(&dead2).Error; err != nil {
		t.Fatal(err)
	}

	resp, err := GetGroupMembersPaged(context.Background(), MembersOptions{GroupID: g.ID, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Members) != 1 || len(resp.Members[0].DirectGroups) != 1 {
		t.Fatalf("响应结构异常：%+v", resp.Members)
	}
	got := resp.Members[0].DirectGroups[0].InstanceCount
	if got != 2 {
		t.Errorf("软删 instance 不应计入：期望 2(live)，实际 %d", got)
	}
}
