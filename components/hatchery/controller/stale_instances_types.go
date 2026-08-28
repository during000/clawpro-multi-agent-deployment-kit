package controller

// stale_instances_types.go — 共享类型 / 常量
//
// 存量实例分组归属处理（stale-instances v1.0）：
// 当用户分组发生变更（编辑分组 / 添加未分组用户到分组 / 修改分组父级 / OneID 同步）后，
// 处理 group_id 与所有人当前分组归属不一致的实例。

// Trigger source 常量（apply 接口入参）
const (
	TriggerSourceUserEdit          = "user_edit"
	TriggerSourceUserAddedToGroup  = "user_added_to_group"
	TriggerSourceGroupParentChange = "group_parent_change"
	TriggerSourceOneIDSync         = "oneid_sync"
	TriggerSourceListPageFollowup  = "list_page_followup"
)

// Action 常量（apply 接口入参）
const (
	StaleActionMigrate     = "migrate"
	StaleActionHandover    = "handover"
	StaleActionPendingUser = "pending_user"
	StaleActionArchiveStop = "archive_stop"
)

// 场景标识（apply 内部用）
const (
	ScenarioA     = "A"     // 用户被移出某分组（仍属于其他分组）
	ScenarioB     = "B"     // 用户被移出所有分组（变未分组）
	ScenarioC     = "C"     // 用户从未分组加入新分组
	ScenarioD     = "D"     // 修改分组父级
	ScenarioNoop  = "noop"  // 状态已对齐，无需处理
)

// validTriggerSource 校验 trigger_source 是否合法。
func validTriggerSource(s string) bool {
	switch s {
	case TriggerSourceUserEdit, TriggerSourceUserAddedToGroup,
		TriggerSourceGroupParentChange, TriggerSourceOneIDSync,
		TriggerSourceListPageFollowup:
		return true
	}
	return false
}

// validStaleAction 校验 action 是否合法。
func validStaleAction(a string) bool {
	switch a {
	case StaleActionMigrate, StaleActionHandover, StaleActionPendingUser, StaleActionArchiveStop:
		return true
	}
	return false
}
