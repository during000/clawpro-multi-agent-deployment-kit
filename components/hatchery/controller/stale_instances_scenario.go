package controller

import (
	"context"

	"hatchery/model"
)

// stale_instances_scenario.go — 场景判定与可见性矩阵
//
// 入参为某 instance 的当前 (user_id, group_id) 与该 user 当前所属 group_ids，
// 结合上游写端点的 trigger_source，判定属于哪种场景（A / B / C / D / noop）。

// scenarioInput apply 内部使用的场景判定入参。
type scenarioInput struct {
	UserID       uint
	GroupID      uint   // instance.group_id 当前值（0 = 未分组）
	UserGroupIDs []uint // user 当前所属分组 ID 列表（来自 user_group_members）
}

// detectScenario 判定 (instance, user, trigger) 三元组属于哪种场景。
//
//   - group_parent_change → 永远 D
//   - 其他 trigger：
//     用户已无任何分组       → B
//     instance.group_id == 0 → C（用户从无到有加入分组）
//     用户不在 instance.group_id → A
//     否则                    → noop
func detectScenario(s scenarioInput, triggerSource string) string {
	if triggerSource == TriggerSourceGroupParentChange {
		return ScenarioD
	}
	if len(s.UserGroupIDs) == 0 {
		return ScenarioB
	}
	if s.GroupID == 0 {
		return ScenarioC
	}
	for _, gid := range s.UserGroupIDs {
		if gid == s.GroupID {
			return ScenarioNoop
		}
	}
	return ScenarioA
}

// actionAllowedInScenario 校验 (scenario, action) 是否合法。
//
// | action          | A | B | C | D |
// |-----------------|---|---|---|---|
// | migrate         | ✅ | ✅(target=0) | ✅ | ✅ (no-op，仅写记录) |
// | handover        | ✅ | ✅ | ✅ | ❌ |
// | pending_user    | ✅ | ✅(仅 allow_migrate) | ✅(仅 allow_migrate) | ❌ |
// | archive_stop    | ✅ | ✅ | ✅ | ❌ |
//
// 关于 (C, handover)：早期产品要求"只能移交给同分组下的其他用户"，未分组实例
// 没法满足"同分组"约束，因此禁。后续产品改为"允许移交给任意用户"（target 多分组
// 时显式指定 group_id），同分组限制不再存在，未分组实例也应能 handover。
func actionAllowedInScenario(scenario, action string) bool {
	switch scenario {
	case ScenarioA:
		return action == StaleActionMigrate || action == StaleActionHandover ||
			action == StaleActionPendingUser || action == StaleActionArchiveStop
	case ScenarioB:
		return action == StaleActionMigrate || action == StaleActionHandover ||
			action == StaleActionPendingUser || action == StaleActionArchiveStop
	case ScenarioC:
		return action == StaleActionMigrate || action == StaleActionHandover ||
			action == StaleActionPendingUser || action == StaleActionArchiveStop
	case ScenarioD:
		return action == StaleActionMigrate
	case ScenarioNoop:
		// handover 是强制换主人，非 stale 实例也应能被 admin 移交
		return action == StaleActionHandover
	}
	return false
}

// loadUserGroupIDs 取某 user 当前所属的所有 group_ids（不含闭包祖先）。
func loadUserGroupIDs(ctx context.Context, userID uint) ([]uint, error) {
	if userID == 0 {
		return nil, nil
	}
	ids, rerr := model.GetUserGroupIDs(ctx, userID)
	if rerr != nil {
		return nil, rerr
	}
	return ids, nil
}

// userInGroup 判断 user 是否属于指定 group_id 的直属成员。
func userInGroup(ctx context.Context, userID, groupID uint) (bool, error) {
	if userID == 0 || groupID == 0 {
		return false, nil
	}
	ids, err := loadUserGroupIDs(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, gid := range ids {
		if gid == groupID {
			return true, nil
		}
	}
	return false, nil
}
