package controller

import (
	"testing"
)

// TestAdminListUserGroups_InstanceCount 验证 /admin/user-groups 响应里每项带
// instance_count 字段 —— 本组 + 所有子孙组下创建的 agent 总数（通过 group_closure 聚合）。
//
// 层级：
//
//	研发中心(1)
//	  ├ 后端组(2)
//	  │    └ Java组(3)
//	  └ 前端组(4)
//	设计部(5)
//
// 实例分布（user_id 不重要，只看 group_id）：
//
//	group_id=2 × 3     ← 后端组
//	group_id=3 × 2     ← Java 组
//	group_id=4 × 1     ← 前端组
//	group_id=5 × 4     ← 设计部
//	group_id=0 × 2     ← 未指定分组，不计入任何组
//
// 期望的 instance_count：
//
//	研发中心(1) = 3 + 2 + 1 = 6   （后端 + Java + 前端）
//	后端组(2)   = 3 + 2     = 5   （本组 + Java）
//	Java组(3)   = 2
//	前端组(4)   = 1
//	设计部(5)   = 4
func TestAdminListUserGroups_InstanceCount(t *testing.T) {
	t.Skip("instance_count removed from user group response in Release")
}

// TestAdminListUserGroups_InstanceCount_Zero 没有任何 instance 时 instance_count=0（而非缺字段）。
func TestAdminListUserGroups_InstanceCount_Zero(t *testing.T) {
	t.Skip("instance_count removed from user group response in Release")
}

// TestAdminListUserGroups_InstanceCount_SoftDeletedExcluded 已软删的 instance
// 不应计入 instance_count（线上回归：销毁实例后 count 仍非零）。
//
// 层级：单个根组 G + 在 G 下创建 3 条实例，软删其中 2 条 → 期望 instance_count=1。
func TestAdminListUserGroups_InstanceCount_SoftDeletedExcluded(t *testing.T) {
	t.Skip("instance_count removed from user group response in Release")
}
