package controller

import (
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

var (
	ErrOperationInProgress = hcommon.I18nError(i18n.MsgOperationInProgress)
	ErrOperationConflict   = hcommon.I18nError(i18n.MsgOperationConflict)
)

// setOperationWithAgentReset 乐观锁写操作标记，同时原子重置 agent_ready=0
// 用于 reboot/reinstall 等操作，避免两次独立 DB 操作的不一致窗口
func setOperationWithAgentReset(db *gorm.DB, instance *model.Instance, operation string) error {
	now := time.Now()

	updates := map[string]interface{}{
		"current_operation":            operation,
		"current_operation_state":      model.OpStateProcessing,
		"current_operation_updated_at": &now,
		"agent_ready":                  0,
	}
	// 操作即时写 last_known_status（P3 实时性补偿）
	if transitStatus, ok := model.OperationTransitStatus[operation]; ok {
		updates["last_known_status"] = transitStatus
		updates["status_synced_at"] = now
	}

	result := db.Model(&model.Instance{}).
		Where("id = ? AND (current_operation = '' OR current_operation = ?)", instance.ID, operation).
		Updates(updates)

	if result.Error != nil {
		return hcommon.I18nRichError(result.Error, i18n.MsgOperationFailed)
	}
	if result.RowsAffected == 0 {
		return ErrOperationInProgress
	}

	instance.CurrentOperation = operation
	instance.CurrentOperationState = model.OpStateProcessing
	instance.CurrentOperationUpdatedAt = &now
	instance.AgentReady = 0

	if transitStatus, ok := model.OperationTransitStatus[operation]; ok {
		instance.LastKnownStatus = transitStatus
		instance.StatusSyncedAt = &now
	}

	return nil
}

// setOperation 乐观锁写操作标记
// 如果实例已有操作进行中，返回错误（删除操作例外，可覆盖其他操作）
// 成功返回 nil，失败返回错误（调用方应返回 409）
func setOperation(db *gorm.DB, instance *model.Instance, operation string) error {
	now := time.Now()

	// 构造操作标记更新（含即时写 last_known_status）
	updates := map[string]interface{}{
		"current_operation":            operation,
		"current_operation_state":      model.OpStateProcessing,
		"current_operation_updated_at": &now,
	}
	if transitStatus, ok := model.OperationTransitStatus[operation]; ok {
		updates["last_known_status"] = transitStatus
		updates["status_synced_at"] = now
	}

	// 删除可以覆盖普通操作，但不能抢占资源调整 worker 持有的锁。
	if operation == model.OpDelete {
		result := db.Model(&model.Instance{}).
			Where("id = ? AND (current_operation IS NULL OR current_operation NOT IN ?)",
				instance.ID, []string{model.OpAdjustInstanceType, model.OpAdjustSystemDisk}).
			Updates(updates)

		if result.Error != nil {
			return hcommon.I18nRichError(result.Error, i18n.MsgOperationFailed)
		}

		if result.RowsAffected == 0 {
			return ErrOperationInProgress
		}

		// 更新内存中的实例对象
		instance.CurrentOperation = operation
		instance.CurrentOperationState = model.OpStateProcessing
		instance.CurrentOperationUpdatedAt = &now
		if transitStatus, ok := model.OperationTransitStatus[operation]; ok {
			instance.LastKnownStatus = transitStatus
			instance.StatusSyncedAt = &now
		}

		return nil
	}

	// 非删除操作：只在 current_operation 为空或与目标操作相同时更新
	result := db.Model(&model.Instance{}).
		Where("id = ? AND (current_operation = '' OR current_operation = ?)", instance.ID, operation).
		Updates(updates)

	if result.Error != nil {
		return hcommon.I18nRichError(result.Error, i18n.MsgOperationFailed)
	}
	if result.RowsAffected == 0 {
		// 另一个操作正在进行中
		return ErrOperationInProgress
	}

	// 更新内存中的实例对象
	instance.CurrentOperation = operation
	instance.CurrentOperationState = model.OpStateProcessing
	instance.CurrentOperationUpdatedAt = &now
	if transitStatus, ok := model.OperationTransitStatus[operation]; ok {
		instance.LastKnownStatus = transitStatus
		instance.StatusSyncedAt = &now
	}

	return nil
}

// setOperationForRetry 重试场景的操作标记（允许覆盖）
// 重试时允许覆盖 loading 状态的操作标记
func setOperationForRetry(db *gorm.DB, instance *model.Instance, operation string) error {
	now := time.Now()

	// 重试允许覆盖：比普通 setOperation 更宽松，但仍需乐观锁保护
	// 允许条件：current_operation 为空、与目标操作相同、或上次操作已失败
	updates := map[string]interface{}{
		"current_operation":            operation,
		"current_operation_state":      model.OpStateProcessing,
		"current_operation_updated_at": &now,
		"agent_ready":                  0, // 重试时重置 Agent 状态
	}
	if transitStatus, ok := model.OperationTransitStatus[operation]; ok {
		updates["last_known_status"] = transitStatus
		updates["status_synced_at"] = now
	}

	result := db.Model(&model.Instance{}).
		Where("id = ? AND (current_operation = '' OR current_operation = ? OR current_operation_state = ?)",
			instance.ID, operation, model.OpStateFailed).
		Updates(updates)

	if result.Error != nil {
		return hcommon.I18nRichError(result.Error, i18n.MsgOperationFailed)
	}
	if result.RowsAffected == 0 {
		// 另一个不同的操作正在进行中，拒绝重试覆盖
		return ErrOperationConflict
	}

	// 更新内存中的实例对象
	instance.CurrentOperation = operation
	instance.CurrentOperationState = model.OpStateProcessing
	instance.CurrentOperationUpdatedAt = &now
	instance.AgentReady = 0
	if transitStatus, ok := model.OperationTransitStatus[operation]; ok {
		instance.LastKnownStatus = transitStatus
		instance.StatusSyncedAt = &now
	}

	return nil
}

// clearOperation 清除操作标记（操作完成或失败后调用）
func clearOperation(db *gorm.DB, instance *model.Instance, state string) error {
	now := time.Now()

	updates := map[string]interface{}{
		"current_operation":            model.OpNone,
		"current_operation_state":      state,
		"current_operation_updated_at": &now,
	}

	// 如果是成功状态，同时更新 last_stable_state
	if state == model.OpStateSuccess {
		if instance.LastCVMState != "" {
			updates["last_stable_state"] = instance.LastCVMState
		}
	}

	result := db.Model(&model.Instance{}).
		Where("id = ? AND current_operation != ''", instance.ID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	// 更新内存中的实例对象
	instance.CurrentOperation = model.OpNone
	instance.CurrentOperationState = state
	instance.CurrentOperationUpdatedAt = &now
	if state == model.OpStateSuccess && instance.LastCVMState != "" {
		instance.LastStableState = instance.LastCVMState
	}

	return nil
}

// canOperate 检查实例是否可执行指定操作
// 返回 nil 表示可以操作，否则返回错误原因
// openaClawStatus 为当前 OpenClaw 语义状态（由 ResolveInstanceStatus 计算），用于删除可用性校验
func canOperate(instance *model.Instance, operation string, openClawStatus ...string) error {
	// 删除操作需要额外校验 OpenClaw 状态
	// loading/destroying/pending 状态下底层 CVM 可能处于过渡态，禁止删除
	if operation == model.OpDelete && len(openClawStatus) > 0 {
		status := openClawStatus[0]
		switch status {
		case model.StatusLoading, model.StatusDestroying:
			return hcommon.I18nError(i18n.MsgInstanceCannotDeleteLoading)
		case model.StatusPending:
			return hcommon.I18nError(i18n.MsgInstanceCannotDeleteDisabled)
		case model.StatusCreating:
			return hcommon.I18nError(i18n.MsgInstanceCannotDeleteCreating)
		}
	}

	// 如果有操作进行中，不允许新操作。
	if instance.CurrentOperation != "" {
		// 删除可以覆盖普通操作，但不能覆盖资源调整。
		if operation == model.OpDelete && !model.IsResourceAdjustmentOperation(instance.CurrentOperation) {
			return nil
		}
		return hcommon.I18nError(i18n.MsgUpgradeOperationInProgress, instance.CurrentOperation)
	}

	return nil
}

// isOperationTimedOut 检查操作是否超时
func isOperationTimedOut(instance *model.Instance) bool {
	if instance.CurrentOperation == "" || instance.CurrentOperationUpdatedAt == nil {
		return false
	}

	timeout, ok := model.OperationTimeouts[instance.CurrentOperation]
	if !ok {
		timeout = 300 // 默认 5 分钟
	}

	return time.Since(*instance.CurrentOperationUpdatedAt).Seconds() > float64(timeout)
}

// markOperationFailed 标记操作失败
func markOperationFailed(db *gorm.DB, instance *model.Instance, reason string) error {
	now := time.Now()

	result := db.Model(&model.Instance{}).
		Where("id = ? AND current_operation = ?", instance.ID, instance.CurrentOperation).
		Updates(map[string]interface{}{
			"current_operation_state":      model.OpStateFailed,
			"current_operation_updated_at": &now,
		})

	if result.Error != nil {
		return result.Error
	}

	instance.CurrentOperationState = model.OpStateFailed
	instance.CurrentOperationUpdatedAt = &now

	return nil
}
