package controller

import (
	"context"
	"net/http"
	"testing"

	"hatchery/model"
)

// TestInstancesByUserGroup_NoDuplicateInstance 当 user_group_ids 与 group_ids
// 展开后产生交叉(同一实例既被精确对命中又被子树展开命中)，响应中的同一 instance
// 应只出现一次，不能被重复返回。
//
// 拓扑：
//
//	parent(1) ─ closure depth 0 自指
//	          └── child(2)        depth 1
//	成员：alice(7) 直属于 child(2)
//	实例：alice 在 child(2) 下创建 1 个 agent
//
// 入参：
//
//	user_group_ids = [(7, 2)]   ← 精确对，命中 alice × child
//	group_ids      = [1]        ← 子树展开为 {1, 2}，对 child 直属成员 alice 又得 (7, 2)
//
// 两个条件最终都展开出同一个 (user_id=7, group_id=2)，且只有一条匹配的 instance。
// 期望 instances 列表长度为 1，且不重复出现该实例。
func TestInstancesByUserGroup_NoDuplicateInstance_FromOverlapBetweenConditions(t *testing.T) {
	initInstancesByUserGroupTestDB(t)

	// closure：parent + child + edge parent→child
	seedClosureSelf(t, 1)
	seedClosureSelf(t, 2)
	seedClosureEdge(t, 1, 2, 1)

	// 成员关系：alice (uid=7) 在 child(2) 直属
	mustSeedMember(t, 2, 7)

	// 实例：alice × child
	mustSeedInstance(t, "alice-on-child", "ins-x", 7, 2)

	status, items := callByUserGroup(t, map[string]interface{}{
		"user_group_ids": []map[string]uint{{"user_id": 7, "group_id": 2}},
		"group_ids":      []uint{1},
	})
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	if len(items) != 1 {
		t.Fatalf("交叉命中同一实例时应去重，期望 1 条，实际 %d: %+v", len(items), items)
	}
	if got := items[0]["instance_id"]; got != "ins-x" {
		t.Errorf("instance_id 错: %v", got)
	}
}

// TestInstancesByUserGroup_NoDuplicateInstance_GroupIDsRedundant group_ids 中
// 同时传入 parent 和 child，子树展开有重复的 (user, group) 对时也不应重复返回。
func TestInstancesByUserGroup_NoDuplicateInstance_GroupIDsRedundant(t *testing.T) {
	initInstancesByUserGroupTestDB(t)

	seedClosureSelf(t, 1)
	seedClosureSelf(t, 2)
	seedClosureEdge(t, 1, 2, 1) // parent(1) → child(2)

	mustSeedMember(t, 2, 7) // alice(7) 在 child(2)
	mustSeedInstance(t, "alice-c", "ins-c", 7, 2)

	// group_ids 同时传 parent 和 child：parent 子树包含 child，故 (7,2) 会被展开两次
	status, items := callByUserGroup(t, map[string]interface{}{
		"group_ids": []uint{1, 2},
	})
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	if len(items) != 1 {
		t.Fatalf("group_ids 冗余传入时应去重，期望 1 条，实际 %d: %+v", len(items), items)
	}
}

// TestInstancesByUserGroup_NoDuplicateInstance_UserGroupIDsRedundant
// user_group_ids 中重复传入同一对 (user_id, group_id) 时不应重复返回。
func TestInstancesByUserGroup_NoDuplicateInstance_UserGroupIDsRedundant(t *testing.T) {
	initInstancesByUserGroupTestDB(t)

	seedClosureSelf(t, 1)
	mustSeedInstance(t, "alice", "ins-a", 7, 1)

	status, items := callByUserGroup(t, map[string]interface{}{
		"user_group_ids": []map[string]uint{
			{"user_id": 7, "group_id": 1},
			{"user_id": 7, "group_id": 1}, // 重复传
		},
	})
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	if len(items) != 1 {
		t.Fatalf("重复 pair 应去重，期望 1 条，实际 %d", len(items))
	}
}

// TestInstancesByUserGroup_SoftDeletedInstanceExcluded 已软删(销毁)的 instance
// 不应出现在响应里。线上回归: 销毁 agent 后仍被该接口返回会误导前端。
//
// 场景: alice(7) 在 group(1) 下 3 条 instance, 软删其中 2 条 → 期望返回 1 条。
func TestInstancesByUserGroup_SoftDeletedInstanceExcluded(t *testing.T) {
	initInstancesByUserGroupTestDB(t)

	seedClosureSelf(t, 1)

	mkInst := func(name, cvmID string) model.Instance {
		tk := "sk-" + name
		return model.Instance{Name: name, InstanceId: cvmID, ProxyToken: &tk, UserID: 7, GroupID: 1}
	}
	live := mkInst("live", "ins-live")
	dead1 := mkInst("dead-1", "ins-dead-1")
	dead2 := mkInst("dead-2", "ins-dead-2")
	for _, inst := range []*model.Instance{&live, &dead1, &dead2} {
		if err := model.DB(context.Background()).Create(inst).Error; err != nil {
			t.Fatalf("create %s: %v", inst.Name, err)
		}
	}
	if err := model.DB(context.Background()).Delete(&dead1).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.DB(context.Background()).Delete(&dead2).Error; err != nil {
		t.Fatal(err)
	}

	status, items := callByUserGroup(t, map[string]interface{}{
		"user_group_ids": []map[string]uint{{"user_id": 7, "group_id": 1}},
	})
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	if len(items) != 1 {
		t.Fatalf("软删 instance 不应返回, 期望 1 条(live), 实际 %d: %+v", len(items), items)
	}
	if got := items[0]["instance_id"]; got != "ins-live" {
		t.Errorf("应只返 live, 实际 instance_id=%v", got)
	}
}
