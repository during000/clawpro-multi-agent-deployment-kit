package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// stale_instances_apply.go — apply 接口业务核心
//
// 4 种 action：
//   - migrate:      只换 group_id（场景 A/C），或保持不变（B target_group_id=0；D 仅写记录）
//   - handover:     只换 user_id（不关机），目标 user 必须在实例当前 group 内
//   - pending_user: 打 pending_user_action 标 + 关机；可选子标记 allow_migrate / allow_same_group_handover
//   - archive_stop: 打 stale_group 标 + 关机
//
// apply 同一请求可混合多种 action（OneID 同步必需）。

// applyActionItem 单条 action 在 apply 请求里的结构。
//
// TargetGroupID 用 *uint 区分"未传"和"传 0"两种语义：
//   - nil      → 未传字段（migrate action 必须显式给值，否则 400）
//   - 非 nil 0 → 显式指定未分组（场景 B 必填）
//   - 非 nil >0 → 目标分组 ID
type applyActionItem struct {
	Action                 string `json:"action"`
	InstanceIDs            []uint `json:"instance_ids"`
	TargetGroupID          *uint  `json:"target_group_id,omitempty"`           // migrate
	TargetUserID           uint   `json:"target_user_id,omitempty"`            // handover
	AllowMigrate           bool   `json:"allow_migrate,omitempty"`             // pending_user
	AllowSameGroupHandover bool   `json:"allow_same_group_handover,omitempty"` // pending_user
}

// targetGroupIDValue 内部读取：未传或显式 0 都返回 0；调用方需先在 handler
// 入口确保 migrate 时 TargetGroupID != nil。
func (i applyActionItem) targetGroupIDValue() uint {
	if i.TargetGroupID == nil {
		return 0
	}
	return *i.TargetGroupID
}

// applyRequest apply 接口请求体。
type applyRequest struct {
	TriggerSource string            `json:"trigger_source"`
	Actions       []applyActionItem `json:"actions"`
}

// applyResultItem apply 接口返回的单条结果。
type applyResultItem struct {
	InstanceID uint   `json:"instance_id"`
	Action     string `json:"action"`
	Status     string `json:"status"` // success / failed / noop
	Error      string `json:"error,omitempty"`
}

// applyEngine 持有 apply 过程中的共享上下文（避免在每个 action 里重复传参）。
type applyEngine struct {
	ctx           context.Context
	r             interface{} // 仅用于 audit 追踪，不暴露
	actorID       uint
	triggerSource string
	results       []applyResultItem

	// 缓存：避免重复查询
	userGroupCache map[uint][]uint // user_id → groups（user 当前所属分组）

	// 由 handler 在循环外预查得到，runOne 在 noop/scenario 判定前
	// 用这两张 map 校验 TargetGroupID / TargetUserID 是否真实存在，
	// 不存在的请求在记录层就 failed，避免被 noop 早返回吞掉。
	existingGroupIDs map[uint]bool
	existingUserIDs  map[uint]bool
}

func newApplyEngine(ctx context.Context, actorID uint, triggerSource string, existingGroupIDs, existingUserIDs map[uint]bool) *applyEngine {
	if existingGroupIDs == nil {
		existingGroupIDs = map[uint]bool{}
	}
	if existingUserIDs == nil {
		existingUserIDs = map[uint]bool{}
	}
	return &applyEngine{
		ctx:              ctx,
		actorID:          actorID,
		triggerSource:    triggerSource,
		results:          make([]applyResultItem, 0, 32),
		userGroupCache:   make(map[uint][]uint),
		existingGroupIDs: existingGroupIDs,
		existingUserIDs:  existingUserIDs,
	}
}

func (e *applyEngine) groupsOfUser(userID uint) ([]uint, error) {
	if v, ok := e.userGroupCache[userID]; ok {
		return v, nil
	}
	ids, err := loadUserGroupIDs(e.ctx, userID)
	if err != nil {
		return nil, err
	}
	e.userGroupCache[userID] = ids
	return ids, nil
}

func (e *applyEngine) recordResult(instanceID uint, action, status, errMsg string) {
	e.results = append(e.results, applyResultItem{
		InstanceID: instanceID,
		Action:     action,
		Status:     status,
		Error:      errMsg,
	})
}

// run 顺序执行所有 action（单 instance 失败不影响其他）。
func (e *applyEngine) run(actions []applyActionItem) {
	for _, item := range actions {
		if !validStaleAction(item.Action) {
			for _, id := range item.InstanceIDs {
				e.recordResult(id, item.Action, "failed", i18n.T(e.ctx, i18n.MsgStaleErrUnsupportedAction))
			}
			continue
		}
		for _, instID := range item.InstanceIDs {
			e.runOne(instID, item)
		}
	}
}

// runOne 处理单条 instance 的单个 action。
func (e *applyEngine) runOne(instanceID uint, item applyActionItem) {
	var instance model.Instance
	if err := model.DB(e.ctx).First(&instance, instanceID).Error; err != nil {
		e.recordResult(instanceID, item.Action, "failed", i18n.T(e.ctx, i18n.MsgInstanceNotFound))
		return
	}

	// 预校验 target 字段存在性。这一步故意放在 detectScenario 之前——若实例
	// 已对齐被判为 noop，但请求里的 target_group_id / target_user_id 实际不
	// 存在，应当 failed 提示调用方而不是默默 noop（吞错误）。
	if errMsg, ok := preCheckTargetExistence(item, e.existingGroupIDs, e.existingUserIDs); ok {
		e.recordResult(instanceID, item.Action, "failed", translateApplyError(e.ctx, errMsg))
		return
	}

	// 场景判定
	groups, err := e.groupsOfUser(instance.UserID)
	if err != nil {
		e.recordResult(instanceID, item.Action, "failed", i18n.T(e.ctx, i18n.MsgStaleErrLoadUserGroupsFailed))
		return
	}
	scenario := detectScenario(scenarioInput{
		UserID:       instance.UserID,
		GroupID:      instance.GroupID,
		UserGroupIDs: groups,
	}, e.triggerSource)

	// list_page_followup 场景下 migrate 不受 noop / 场景校验限制：
	// 管理员在 Agent 列表页显式发起的迁移，即使用户-分组关系恰好一致也应允许执行。
	isListPageFollowupMigrate := e.triggerSource == TriggerSourceListPageFollowup && item.Action == StaleActionMigrate

	if scenario == ScenarioNoop && item.Action != StaleActionHandover && !isListPageFollowupMigrate {
		// 状态已对齐，认为是幂等 noop；handover 是强制换主人，不受 stale 状态限制
		e.recordResult(instanceID, item.Action, "noop", "")
		return
	}

	if !isListPageFollowupMigrate && !actionAllowedInScenario(scenario, item.Action) {
		e.recordResult(instanceID, item.Action, "failed", i18n.T(e.ctx, i18n.MsgStaleErrActionNotAllowedInScenario))
		return
	}

	switch item.Action {
	case StaleActionMigrate:
		e.applyMigrate(&instance, item, scenario, groups)
	case StaleActionHandover:
		e.applyHandover(&instance, item, scenario)
	case StaleActionPendingUser:
		e.applyPendingUser(&instance, item, scenario)
	case StaleActionArchiveStop:
		e.applyArchiveStop(&instance, item, scenario)
	}
}

// validStaleAction 校验 action 是否合法（被 stale_instances_types.go 定义并复用此处）。

// preCheckTargetExistence 在 detectScenario 之前判定请求引用的 target 资源是否存在。
// 返回 (错误码, true) 表示应当 failed 早返回；(空, false) 表示通过。
//
//   - migrate:  TargetGroupID > 0 时必须命中 existingGroupIDs
//   - handover: TargetUserID  > 0 时必须命中 existingUserIDs
//   - 其他 action 不读这两个字段，一律视为通过
func preCheckTargetExistence(item applyActionItem, existingGroupIDs, existingUserIDs map[uint]bool) (string, bool) {
	if item.Action == StaleActionMigrate {
		gid := item.targetGroupIDValue()
		if gid > 0 && !existingGroupIDs[gid] {
			return "target_group_not_found", true
		}
	}
	if item.Action == StaleActionHandover && item.TargetUserID > 0 && !existingUserIDs[item.TargetUserID] {
		return "target_user_not_found", true
	}
	return "", false
}

// applyMigrate 实施 migrate：只换 group_id（场景 A/C），或保持不变（B target=0；D 仅写记录）。
func (e *applyEngine) applyMigrate(inst *model.Instance, item applyActionItem, scenario string, userGroups []uint) {
	oldGroupID := inst.GroupID
	newGroupID := item.targetGroupIDValue()

	switch scenario {
	case ScenarioA, ScenarioC:
		// target_group_id 必须 ∈ user 当前 group_ids 中
		if newGroupID == 0 {
			e.recordResult(inst.ID, item.Action, "failed", i18n.T(e.ctx, i18n.MsgStaleErrTargetGroupIDRequired))
			return
		}
		if !uintInSlice(newGroupID, userGroups) {
			e.recordResult(inst.ID, item.Action, "failed", i18n.T(e.ctx, i18n.MsgStaleErrTargetGroupNotInUserGroups))
			return
		}
	case ScenarioB:
		// 用户已无任何分组，target_group_id 必须为 0（回退到全局默认）
		if newGroupID != 0 {
			e.recordResult(inst.ID, item.Action, "failed", i18n.T(e.ctx, i18n.MsgStaleErrTargetGroupIDMustZeroForUngrouped))
			return
		}
	case ScenarioD:
		// 配置继承链由 Resolver 自动按新链解析；instance.group_id 不动
		newGroupID = inst.GroupID
	}

	err := model.DB(e.ctx).Transaction(func(tx *gorm.DB) error {
		// 场景 A/B/C 需要写 group_id；D 不写
		if scenario != ScenarioD {
			if err := tx.Model(&model.Instance{}).Where("id = ?", inst.ID).
				Update("group_id", newGroupID).Error; err != nil {
				return err
			}
		}
		// 写处理记录
		if err := writeICGRTx(tx, inst, oldGroupID, newGroupID, inst.UserID, inst.UserID,
			scenarioMigrateAction(scenario), model.ICGRActorAdmin, e.actorID, e.triggerSource, ""); err != nil {
			return err
		}
		// 处理完毕：清掉所有 stale-instances 相关标记
		return clearStaleFlagsTx(tx, inst.ID)
	})
	if err != nil {
		e.recordResult(inst.ID, item.Action, "failed", err.Error())
		return
	}
	// 通知用户
	var migrateMsg string
	if scenario == ScenarioD {
		migrateMsg = i18n.T(e.ctx, i18n.NotifMsgMigratedScenarioD, inst.Name)
	} else {
		migrateMsg = i18n.T(e.ctx, i18n.NotifMsgMigratedByAdmin,
			lookupGroupName(e.ctx, oldGroupID), inst.Name, lookupGroupName(e.ctx, newGroupID))
	}
	_ = model.CreateNotification(e.ctx, inst.UserID, inst.ID, inst.Name,
		model.NotifyTypeInstanceMigrated, i18n.T(e.ctx, i18n.NotifTitleAgentMigrated), migrateMsg)
	e.recordResult(inst.ID, item.Action, "success", "")
}

// scenarioMigrateAction 场景 D 的 migrate 用专属 action 名记录。
func scenarioMigrateAction(scenario string) string {
	if scenario == ScenarioD {
		return model.ICGRActionParentChangePending
	}
	return model.ICGRActionMigrate
}

// applyHandover 实施 handover：换 user_id 并把 instance.group_id 跟随目标用户。
//
// 管理端语义（与用户端 /openclaw/stale-instances/{initiate,cancel,accept,reject} 不同）：
//   - target_user 可以是任意用户，**不**限制为同分组直属成员
//   - 未指定 target_group_id（nil）→ 同分组移交：instance.group_id 保持不变，
//     目标用户必须在当前分组中
//   - 显式指定 target_group_id：
//       * target 无分组 → instance.group_id := 0（target_group_id 必须为 0）
//       * target 单分组 → target_group_id=0 自动选那一个；非 0 必须匹配
//       * target 多分组 → 必须显式给 target_group_id > 0，且 ∈ target 的分组
//   - 不关机；目标用户立刻就能在自己的列表里看到该实例
func (e *applyEngine) applyHandover(inst *model.Instance, item applyActionItem, scenario string) {
	if item.TargetUserID == 0 {
		e.recordResult(inst.ID, item.Action, "failed", i18n.T(e.ctx, i18n.MsgStaleErrTargetUserIDRequired))
		return
	}
	if item.TargetUserID == inst.UserID {
		e.recordResult(inst.ID, item.Action, "failed", i18n.T(e.ctx, i18n.MsgStaleErrTargetUserSameAsOwner))
		return
	}

	// 解析目标用户的分组列表，决定 instance.group_id 的新值
	targetGroups, err := loadUserGroupIDs(e.ctx, item.TargetUserID)
	if err != nil {
		e.recordResult(inst.ID, item.Action, "failed", i18n.T(e.ctx, i18n.MsgStaleErrLoadTargetUserGroupsFailed))
		return
	}
	newGroupID, errCode := resolveHandoverTargetGroupID(targetGroups, item.targetGroupIDValue(), item.TargetGroupID != nil, inst.GroupID)
	if errCode != "" {
		e.recordResult(inst.ID, item.Action, "failed", translateApplyError(e.ctx, errCode))
		return
	}

	oldUserID := inst.UserID
	oldGroupID := inst.GroupID
	err = model.DB(e.ctx).Transaction(func(tx *gorm.DB) error {
		// 换 user_id + group_id，不关机；清掉 pending 移交相关字段
		updates := map[string]interface{}{
			"user_id":                      item.TargetUserID,
			"group_id":                     newGroupID,
			"handover_target_user_id":      uint(0),
			"handover_rejected_by_user_id": uint(0),
			"handover_initiated_at":        nil,
		}
		if err := tx.Model(&model.Instance{}).Where("id = ?", inst.ID).
			Updates(updates).Error; err != nil {
			return err
		}
		if err := writeICGRTx(tx, inst, oldGroupID, newGroupID, oldUserID, item.TargetUserID,
			model.ICGRActionHandover, model.ICGRActorAdmin, e.actorID, e.triggerSource, ""); err != nil {
			return err
		}
		return clearStaleFlagsTx(tx, inst.ID)
	})
	if err != nil {
		e.recordResult(inst.ID, item.Action, "failed", err.Error())
		return
	}
	_ = model.CreateNotification(e.ctx, oldUserID, inst.ID, inst.Name,
		model.NotifyTypeInstanceHandoverAccepted, i18n.T(e.ctx, i18n.NotifTitleAgentHandoverByAdmin),
		i18n.T(e.ctx, i18n.NotifMsgHandoverByAdminToOwner,
			lookupGroupName(e.ctx, oldGroupID), inst.Name, lookupUsername(e.ctx, item.TargetUserID)))
	_ = model.CreateNotification(e.ctx, item.TargetUserID, inst.ID, inst.Name,
		model.NotifyTypeInstanceHandoverReceived, i18n.T(e.ctx, i18n.NotifTitleAgentReceivedFromAdmin),
		i18n.T(e.ctx, i18n.NotifMsgHandoverByAdminToTarget, inst.Name))
	e.recordResult(inst.ID, item.Action, "success", "")
}

// resolveHandoverTargetGroupID 根据 target user 的分组列表 + 调用方传入的 target_group_id，
// 计算 handover 后实例的 group_id 新值。
//
//   - 未指定 target_group_id（groupIDSpecified=false）→ 同分组移交：
//     instance.group_id 保持 currentGroupID 不变，目标用户必须在 currentGroupID 中
//     （currentGroupID==0 时目标用户也必须无分组）
//   - 显式指定 target_group_id（groupIDSpecified=true）：
//     * target 无分组：返回 0；若调用方传了非 0 target_group_id 则报错
//     * target 1 个分组：传 0 → 自动选那一个；传非 0 必须 == 该分组
//     * target 2+ 个分组：必须传非 0 target_group_id 且 ∈ target 的分组
//
// 第二个返回值是错误码字符串，空表示成功。
func resolveHandoverTargetGroupID(targetGroups []uint, requestedGroupID uint, groupIDSpecified bool, currentGroupID uint) (uint, string) {
	if !groupIDSpecified {
		// 同分组移交：实例保持在当前分组，目标用户必须在当前分组中
		if currentGroupID == 0 {
			if len(targetGroups) > 0 {
				return 0, "target_user_not_in_same_group"
			}
			return 0, ""
		}
		if !uintInSlice(currentGroupID, targetGroups) {
			return 0, "target_user_not_in_same_group"
		}
		return currentGroupID, ""
	}
	// 显式指定了 target_group_id
	switch len(targetGroups) {
	case 0:
		if requestedGroupID != 0 {
			return 0, "target_user_has_no_groups_target_group_id_must_be_zero"
		}
		return 0, ""
	case 1:
		only := targetGroups[0]
		if requestedGroupID == 0 {
			return only, ""
		}
		if requestedGroupID != only {
			return 0, "target_group_id_not_in_target_user_groups"
		}
		return requestedGroupID, ""
	default:
		if requestedGroupID == 0 {
			return 0, "target_group_id_required_for_multi_group_target"
		}
		if !uintInSlice(requestedGroupID, targetGroups) {
			return 0, "target_group_id_not_in_target_user_groups"
		}
		return requestedGroupID, ""
	}
}

// applyPendingUser 实施 pending_user：打标记 + 关机；至少一个子选项。
func (e *applyEngine) applyPendingUser(inst *model.Instance, item applyActionItem, scenario string) {
	allowMigrate := item.AllowMigrate
	allowHandover := item.AllowSameGroupHandover

	// 场景 B/C：「同组移交」恒为 false（无目标组）
	if scenario == ScenarioB || scenario == ScenarioC {
		allowHandover = false
	}
	if !allowMigrate && !allowHandover {
		e.recordResult(inst.ID, item.Action, "failed", i18n.T(e.ctx, i18n.MsgStaleErrAtLeastOneSubOptionRequired))
		return
	}

	err := model.DB(e.ctx).Transaction(func(tx *gorm.DB) error {
		// 写处理记录（关机操作异步，但 flag 同步落地）
		if err := writeICGRTx(tx, inst, inst.GroupID, inst.GroupID, inst.UserID, inst.UserID,
			model.ICGRActionPendingUser, model.ICGRActorAdmin, e.actorID, e.triggerSource,
			pendingUserExtra(allowMigrate, allowHandover)); err != nil {
			return err
		}
		if err := setFlagTx(tx, inst.ID, model.InstanceFlagPendingUserAction, ""); err != nil {
			return err
		}
		if allowMigrate {
			if err := setFlagTx(tx, inst.ID, model.InstanceFlagAllowMigrate, ""); err != nil {
				return err
			}
		} else {
			if err := delFlagTx(tx, inst.ID, model.InstanceFlagAllowMigrate); err != nil {
				return err
			}
		}
		if allowHandover {
			if err := setFlagTx(tx, inst.ID, model.InstanceFlagAllowSameGroupHandover, ""); err != nil {
				return err
			}
		} else {
			if err := delFlagTx(tx, inst.ID, model.InstanceFlagAllowSameGroupHandover); err != nil {
				return err
			}
		}
		// 同时打 stale_group 标，便于列表筛选
		return setFlagTx(tx, inst.ID, model.InstanceFlagStaleGroup, "")
	})
	if err != nil {
		e.recordResult(inst.ID, item.Action, "failed", err.Error())
		return
	}
	// 异步关机
	go stopInstanceCloud(hcommon.DetachContext(e.ctx), inst.InstanceId)
	var pendingMsg string
	if allowMigrate && allowHandover {
		pendingMsg = i18n.T(e.ctx, i18n.NotifMsgPendingUserBoth,
			lookupGroupName(e.ctx, inst.GroupID), inst.Name)
	} else if allowMigrate {
		pendingMsg = i18n.T(e.ctx, i18n.NotifMsgPendingUserMigrateOnly,
			lookupGroupName(e.ctx, inst.GroupID), inst.Name)
	} else {
		pendingMsg = i18n.T(e.ctx, i18n.NotifMsgPendingUserHandoverOnly,
			lookupGroupName(e.ctx, inst.GroupID), inst.Name)
	}
	_ = model.CreateNotification(e.ctx, inst.UserID, inst.ID, inst.Name,
		model.NotifyTypePendingUserAction, i18n.T(e.ctx, i18n.NotifTitleAgentPendingUserAction), pendingMsg)
	e.recordResult(inst.ID, item.Action, "success", "")
}

// applyArchiveStop 实施 archive_stop：打标记 + 关机。
func (e *applyEngine) applyArchiveStop(inst *model.Instance, item applyActionItem, scenario string) {
	err := model.DB(e.ctx).Transaction(func(tx *gorm.DB) error {
		if err := writeICGRTx(tx, inst, inst.GroupID, inst.GroupID, inst.UserID, inst.UserID,
			model.ICGRActionArchiveStop, model.ICGRActorAdmin, e.actorID, e.triggerSource, ""); err != nil {
			return err
		}
		// 仅打 stale_group 标，pending_user_action 必须确保不在
		if err := delFlagTx(tx, inst.ID, model.InstanceFlagPendingUserAction); err != nil {
			return err
		}
		if err := delFlagTx(tx, inst.ID, model.InstanceFlagAllowMigrate); err != nil {
			return err
		}
		if err := delFlagTx(tx, inst.ID, model.InstanceFlagAllowSameGroupHandover); err != nil {
			return err
		}
		return setFlagTx(tx, inst.ID, model.InstanceFlagStaleGroup, "")
	})
	if err != nil {
		e.recordResult(inst.ID, item.Action, "failed", err.Error())
		return
	}
	go stopInstanceCloud(hcommon.DetachContext(e.ctx), inst.InstanceId)
	_ = model.CreateNotification(e.ctx, inst.UserID, inst.ID, inst.Name,
		model.NotifyTypeInstanceArchivedByAdmin, i18n.T(e.ctx, i18n.NotifTitleAgentArchivedByAdmin),
		i18n.T(e.ctx, i18n.NotifMsgArchivedByAdmin,
			lookupGroupName(e.ctx, inst.GroupID), inst.Name))
	e.recordResult(inst.ID, item.Action, "success", "")
}

// ──────────────────────────────────────────────
// 辅助函数（工具方法）
// ──────────────────────────────────────────────

// applyErrorCodeMap 将内部 snake_case 错误码映射到 i18n key，
// 供 translateApplyError 在 recordResult 前翻译为完整句子。
// 未命中映射的字符串（如 DB 原始 err.Error()）原样返回。
var applyErrorCodeMap = map[string]i18n.Key{
	"unsupported_action":                                  i18n.MsgStaleErrUnsupportedAction,
	"instance_not_found":                                  i18n.MsgInstanceNotFound,
	"load_user_groups_failed":                             i18n.MsgStaleErrLoadUserGroupsFailed,
	"action_not_allowed_in_scenario":                      i18n.MsgStaleErrActionNotAllowedInScenario,
	"target_group_not_found":                              i18n.MsgStaleErrTargetGroupNotFound,
	"target_user_not_found":                               i18n.MsgStaleErrTargetUserNotFound,
	"target_group_id_required":                            i18n.MsgStaleErrTargetGroupIDRequired,
	"target_group_not_in_user_groups":                     i18n.MsgStaleErrTargetGroupNotInUserGroups,
	"target_group_id_must_be_zero_for_ungrouped_user":     i18n.MsgStaleErrTargetGroupIDMustZeroForUngrouped,
	"target_user_id_required":                             i18n.MsgStaleErrTargetUserIDRequired,
	"target_user_same_as_owner":                           i18n.MsgStaleErrTargetUserSameAsOwner,
	"load_target_user_groups_failed":                      i18n.MsgStaleErrLoadTargetUserGroupsFailed,
	"target_user_has_no_groups_target_group_id_must_be_zero": i18n.MsgStaleErrTargetUserNoGroupsMustZero,
	"target_group_id_not_in_target_user_groups":           i18n.MsgStaleErrTargetGroupNotInTargetUserGroups,
	"target_group_id_required_for_multi_group_target":     i18n.MsgStaleErrTargetGroupRequiredForMultiGroup,
	"target_user_not_in_same_group":                       i18n.MsgStaleErrTargetUserNotInSameGroup,
	"at_least_one_sub_option_required":                    i18n.MsgStaleErrAtLeastOneSubOptionRequired,
}

// translateApplyError 将内部错误码翻译为 i18n 完整句子。
// 未命中映射的字符串（如 DB 原始 err.Error()）原样返回。
func translateApplyError(ctx context.Context, code string) string {
	if key, ok := applyErrorCodeMap[code]; ok {
		return i18n.T(ctx, key)
	}
	return code
}

func uintInSlice(v uint, s []uint) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func lookupGroupName(ctx context.Context, groupID uint) string {
	if groupID == 0 {
		return "默认"
	}
	g, err := model.GroupByID(ctx, groupID)
	if err != nil || g == nil {
		return fmt.Sprintf("#%d", groupID)
	}
	return g.FullPath
}

func pendingUserExtra(allowMigrate, allowHandover bool) string {
	b, _ := json.Marshal(map[string]bool{
		"allow_migrate":             allowMigrate,
		"allow_same_group_handover": allowHandover,
	})
	return string(b)
}

// writeICGRTx 在事务里写一条处理记录。
func writeICGRTx(tx *gorm.DB, inst *model.Instance, oldGID, newGID, oldUID, newUID uint,
	action, actorType string, actorID uint, triggerSource, extraJSON string) error {
	if extraJSON == "" {
		extraJSON = "{}"
	}
	r := model.InstanceChangeGroupRecord{
		InstancePK:    inst.ID,
		InstanceID:    inst.InstanceId,
		UserIDBefore:  oldUID,
		UserIDAfter:   newUID,
		GroupIDBefore: oldGID,
		GroupIDAfter:  newGID,
		Action:        action,
		ActorType:     actorType,
		ActorID:       actorID,
		TriggerSource: triggerSource,
		ExtraJSON:     extraJSON,
	}
	return tx.Create(&r).Error
}

// setFlagTx 在事务里写一条 instance_flag。
func setFlagTx(tx *gorm.DB, instanceID uint, flag, extra string) error {
	if extra == "" {
		extra = "{}"
	}
	f := model.InstanceFlag{
		InstanceID: instanceID,
		Flag:       flag,
		Extra:      extra,
		CreatedAt:  time.Now(),
	}
	return tx.Where("instance_id = ? AND flag = ?", instanceID, flag).
		Assign(map[string]any{"extra": extra, "created_at": time.Now()}).
		FirstOrCreate(&f).Error
}

// delFlagTx 在事务里删一条 instance_flag（不存在时 noop）。
func delFlagTx(tx *gorm.DB, instanceID uint, flag string) error {
	return tx.Where("instance_id = ? AND flag = ?", instanceID, flag).
		Delete(&model.InstanceFlag{}).Error
}

// clearStaleFlagsTx 处理完成后清除所有 stale-instances 相关标记。
func clearStaleFlagsTx(tx *gorm.DB, instanceID uint) error {
	flags := []string{
		model.InstanceFlagStaleGroup,
		model.InstanceFlagPendingUserAction,
		model.InstanceFlagAllowMigrate,
		model.InstanceFlagAllowSameGroupHandover,
	}
	return tx.Where("instance_id = ? AND flag IN ?", instanceID, flags).
		Delete(&model.InstanceFlag{}).Error
}

// stopInstanceCloud 关机指定 CVM 实例（异步，失败仅记录日志）。
func stopInstanceCloud(ctx context.Context, cvmInstanceID string) {
	if cvmInstanceID == "" {
		return
	}
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("[StaleInstances] stop instance panic", "cvm_id", cvmInstanceID, "panic", rec)
		}
	}()
	cli, err := GetCVMClient(ctx)
	if err != nil {
		slog.Warn("[StaleInstances] CVM client failed", "err", err)
		return
	}
	if err := callStopInstances(cli, []string{cvmInstanceID}); err != nil {
		slog.Warn("[StaleInstances] stop CVM failed", "cvm_id", cvmInstanceID, "err", err)
	}
}

// startInstanceCloud 开机指定 CVM 实例（异步，失败仅记录日志）。
func startInstanceCloud(ctx context.Context, cvmInstanceID string) {
	if cvmInstanceID == "" {
		return
	}
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("[StaleInstances] start instance panic", "cvm_id", cvmInstanceID, "panic", rec)
		}
	}()
	cli, err := GetCVMClient(ctx)
	if err != nil {
		slog.Warn("[StaleInstances] CVM client failed", "err", err)
		return
	}
	if err := callStartInstances(cli, []string{cvmInstanceID}); err != nil {
		slog.Warn("[StaleInstances] start CVM failed", "cvm_id", cvmInstanceID, "err", err)
	}
}

// 让编译器知道 hcommon 被引用（部分错误返回需要）。
var _ = hcommon.I18nError

// enrichAdminInstancesWithStaleFields 批量回填 stale-instances 相关字段。
//
// handover_target_user_id / handover_rejected_by_user_id 直接走 instances 表，
// 已经在 buildAdminInstanceFromCache 里映射；这里只负责 flags 字段
// （instance_flags 是独立表，必须单独查）。
func enrichAdminInstancesWithStaleFields(ctx context.Context, items []adminInstanceItemWithStatus) {
	if len(items) == 0 {
		return
	}
	ids := make([]uint, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	flagsMap, err := model.GetInstanceFlagsBatch(ctx, ids)
	if err != nil {
		slog.Warn("[StaleInstances] enrich flags failed", "err", err)
		return
	}
	for i := range items {
		if v, ok := flagsMap[items[i].ID]; ok && v != nil {
			items[i].Flags = v
		}
		// 否则保持 buildAdminInstanceFromCache 设置的 []string{} 默认值
	}
}

// formatNullableTime 把 *time.Time 格式化为 RFC3339 字符串指针，nil 保持 nil。
func formatNullableTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format("2006-01-02T15:04:05Z")
	return &s
}

// markStaleForSubtree 给某分组及其子树内所有 group_id != 0 的实例写一条
// `parent_change_pending` 记录 + 发通知。用于 group 父级变更场景。
// 注意：不打 stale_group 标记 —— 场景 D 的 instance.group_id 仍然正确（用户还在该分组），
// 配置继承链由 Resolver 自动按新链解析，无需管理员手动处理。
func markStaleForSubtree(ctx context.Context, rootGroupID uint) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("[StaleInstances] markStaleForSubtree panic", "root", rootGroupID, "panic", rec)
		}
	}()
	if rootGroupID == 0 {
		return
	}
	// 子树：closure 表 ancestor_id = rootGroupID 的所有 descendant_id（含自身）
	var descIDs []uint
	if err := model.DB(ctx).Model(&model.GroupClosure{}).
		Distinct("descendant_id").
		Where("ancestor_id = ?", rootGroupID).
		Pluck("descendant_id", &descIDs).Error; err != nil {
		slog.Warn("[StaleInstances] markStaleForSubtree: subtree query failed", "root", rootGroupID, "err", err)
		return
	}
	if len(descIDs) == 0 {
		return
	}
	var instances []model.Instance
	if err := model.DB(ctx).Where("group_id IN ?", descIDs).Find(&instances).Error; err != nil {
		slog.Warn("[StaleInstances] markStaleForSubtree: instances query failed", "root", rootGroupID, "err", err)
		return
	}
	// 批量写处理记录：单事务 + CreateInBatches，避免 N 条实例 N 次事务
	records := make([]model.InstanceChangeGroupRecord, 0, len(instances))
	for i := range instances {
		inst := &instances[i]
		records = append(records, model.InstanceChangeGroupRecord{
			InstancePK:    inst.ID,
			InstanceID:    inst.InstanceId,
			UserIDBefore:  inst.UserID,
			UserIDAfter:   inst.UserID,
			GroupIDBefore: inst.GroupID,
			GroupIDAfter:  inst.GroupID,
			Action:        model.ICGRActionParentChangePending,
			ActorType:     model.ICGRActorAdmin,
			TriggerSource: TriggerSourceGroupParentChange,
			ExtraJSON:     "{}",
		})
	}
	if len(records) > 0 {
		if err := model.DB(ctx).CreateInBatches(&records, 100).Error; err != nil {
			slog.Warn("[StaleInstances] markStaleForSubtree: batch write records failed",
				"root", rootGroupID, "count", len(records), "err", err)
		}
	}
	// 按 (user_id, group_id) 聚合：同一用户在同一分组下的 agent 合并为一条通知
	type userGroupEntry struct {
		userID        uint
		groupID       uint
		firstInstID   uint
		firstInstName string
		names         []string
	}
	entryOrder := make([]userGroupEntry, 0)
	entryIndex := make(map[uint]map[uint]int) // user_id → group_id → index in entryOrder
	for _, inst := range instances {
		if inst.UserID == 0 {
			continue
		}
		if _, ok := entryIndex[inst.UserID]; !ok {
			entryIndex[inst.UserID] = make(map[uint]int)
		}
		idx, exists := entryIndex[inst.UserID][inst.GroupID]
		if !exists {
			idx = len(entryOrder)
			entryOrder = append(entryOrder, userGroupEntry{
				userID:        inst.UserID,
				groupID:       inst.GroupID,
				firstInstID:   inst.ID,
				firstInstName: inst.Name,
			})
			entryIndex[inst.UserID][inst.GroupID] = idx
		}
		entryOrder[idx].names = append(entryOrder[idx].names, inst.Name)
	}
	for _, entry := range entryOrder {
		namesStr := strings.Join(entry.names, "、")
		_ = model.CreateNotification(ctx, entry.userID, entry.firstInstID, entry.firstInstName,
			model.NotifyTypeStaleGroupOneIDSync, i18n.T(ctx, i18n.NotifTitleAgentOrgConfigUpdated),
			i18n.T(ctx, i18n.NotifMsgOrgConfigUpdated, lookupGroupName(ctx, entry.groupID), namesStr))
	}
	slog.Info("[StaleInstances] marked subtree as stale", "root", rootGroupID, "count", len(instances))
}
