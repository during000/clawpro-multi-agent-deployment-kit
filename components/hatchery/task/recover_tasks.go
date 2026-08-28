package task

import (
	"context"
	"log/slog"
	"time"

	"hatchery/i18n"
	"hatchery/model"
)

// recoverInterruptedSkillInitTasks 恢复被服务重启中断的初始技能包安装任务。
// 将卡在 Installing 或 None 状态的 SkillInstallation 记录标记为 Failed，使用户可重试。
// None 状态说明 installSkillsAsync 尚未开始执行就被中断，同样需要恢复。
func recoverInterruptedSkillInitTasks(ctx context.Context) {
	result := model.DB(ctx).Model(&model.SkillInstallation{}).
		Where("install_status IN ?", []int{model.SkillInstallNone, model.SkillInstalling}).
		Updates(map[string]interface{}{
			"install_status": model.SkillInstallFailed,
			"error_message":  i18n.T(ctx, i18n.MsgTaskInterruptedByRestart),
		})
	if result.Error != nil {
		slog.Error("恢复中断的技能包安装失败", "error", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		slog.Info("恢复中断的技能包安装", "affected", result.RowsAffected)
	}
}

// recoverInterruptedPluginInitTasks 恢复被服务重启中断的角色插件包安装任务。
// 将卡在 Installing 或 None 状态的 PluginInstallation 记录标记为 Failed，使用户可重试。
// None 状态说明 installPluginsAsync 尚未开始执行就被中断，同样需要恢复。
func recoverInterruptedPluginInitTasks(ctx context.Context) {
	result := model.DB(ctx).Model(&model.PluginInstallation{}).
		Where("install_status IN ?", []int{model.PluginInstallNone, model.PluginInstalling}).
		Updates(map[string]interface{}{
			"install_status": model.PluginInstallFailed,
			"error_message":  i18n.T(ctx, i18n.MsgTaskInterruptedByRestart),
		})
	if result.Error != nil {
		slog.Error("恢复中断的插件包安装失败", "error", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		slog.Info("恢复中断的插件包安装", "affected", result.RowsAffected)
	}
}

// recoverInterruptedPluginTasks 恢复被服务重启中断的插件下发任务。
// 将 running 状态的任务中 pending 记录标记为 failed，任务整体标记为 completed。
func recoverInterruptedPluginTasks(ctx context.Context) {
	var tasks []model.PluginDistributionTask
	if err := model.DB(ctx).Where("status = ?", "running").Find(&tasks).Error; err != nil {
		slog.Error("查询中断的插件下发任务失败", "error", err)
		return
	}
	for _, task := range tasks {
		if err := model.DB(ctx).Model(&model.PluginDistributionRecord{}).
			Where("task_id = ? AND status = ?", task.ID, "pending").
			Updates(map[string]interface{}{
				"status": "failed",
				"error":  i18n.T(ctx, i18n.MsgTaskInterruptedByRestart),
			}).Error; err != nil {
			slog.Error("标记中断插件记录失败", "task_id", task.ID, "error", err)
			continue
		}
		var successCount, failedCount int64
		model.DB(ctx).Model(&model.PluginDistributionRecord{}).Where("task_id = ? AND status = ?", task.ID, "success").Count(&successCount)
		model.DB(ctx).Model(&model.PluginDistributionRecord{}).Where("task_id = ? AND status = ?", task.ID, "failed").Count(&failedCount)
		if err := model.DB(ctx).Model(&task).Updates(map[string]interface{}{
			"status":  "completed",
			"success": int(successCount),
			"failed":  int(failedCount),
		}).Error; err != nil {
			slog.Error("更新插件任务状态失败", "task_id", task.ID, "error", err)
			continue
		}
		slog.Info("恢复中断的插件下发任务", "task_id", task.ID, "success", successCount, "failed", failedCount)
	}
}

// recoverInterruptedRoleSyncTasks 恢复被服务重启中断的角色下发任务。
// 将 status='updating' 的 role_distribution_records 全部标 failed，
// 未完成的子任务补上 "服务重启中断" 错误消息。同步刷新 instance.role_sync_status。
func recoverInterruptedRoleSyncTasks(ctx context.Context) {
	logger := slog.With("task", "RecoverRoleSync")

	// 先查出待恢复的记录，逐条更新（SQLite 不支持复杂 CASE，且要精确控制哪些字段填错误消息）
	var records []model.RoleDistributionRecord
	if err := model.DB(ctx).
		Where("status = ?", model.RoleRecordStatusUpdating).
		Find(&records).Error; err != nil {
		logger.Error("查询待恢复 record 失败", "error", err)
		return
	}

	interruptedMsg := i18n.T(ctx, i18n.MsgTaskInterruptedByRestart)
	for _, r := range records {
		updates := map[string]interface{}{
			"status": model.RoleRecordStatusFailed,
		}
		// pending / running 状态视为未完成，补错误消息
		if r.SoulStatus == model.RoleSubStatusPending || r.SoulStatus == model.RoleSubStatusRunning {
			updates["soul_status"] = model.RoleSubStatusFailed
			if r.SoulError == "" {
				updates["soul_error"] = interruptedMsg
			}
		}
		if r.SkillStatus == model.RoleSubStatusPending || r.SkillStatus == model.RoleSubStatusRunning {
			updates["skill_status"] = model.RoleSubStatusFailed
			if r.SkillError == "" {
				updates["skill_error"] = interruptedMsg
			}
		}
		if err := model.DB(ctx).Model(&model.RoleDistributionRecord{}).
			Where("id = ? AND status = ?", r.ID, model.RoleRecordStatusUpdating).
			Updates(updates).Error; err != nil {
			logger.Warn("恢复 record 失败", "record_id", r.ID, "error", err)
			continue
		}
	}

	// 兜底：instance.role_sync_status='updating' 但没有 updating record 的孤儿实例
	// （比如 record 被物理删除的异常情况），也一并翻 failed
	result := model.DB(ctx).Model(&model.Instance{}).
		Where("role_sync_status = ?", model.RoleSyncStatusUpdating).
		Update("role_sync_status", model.RoleSyncStatusFailed)
	if result.Error != nil {
		logger.Error("恢复 instance.role_sync_status 失败", "error", result.Error)
		return
	}

	if len(records) > 0 || result.RowsAffected > 0 {
		logger.Info("角色下发任务恢复完成",
			"records_recovered", len(records),
			"instances_touched", result.RowsAffected)
	}
}

// recoverInterruptedSkillTasks 恢复被服务重启中断的技能下发任务。
// 将 running 状态的任务中 pending 记录标记为 failed，任务整体标记为 completed。
func recoverInterruptedSkillTasks(ctx context.Context) {
	var tasks []model.SkillDistributionTask
	if err := model.DB(ctx).Where("status = ?", "running").Find(&tasks).Error; err != nil {
		slog.Error("查询中断的技能下发任务失败", "error", err)
		return
	}
	for _, task := range tasks {
		if err := model.DB(ctx).Model(&model.SkillDistributionRecord{}).
			Where("task_id = ? AND status = ?", task.ID, "pending").
			Updates(map[string]interface{}{
				"status": "failed",
				"error":  i18n.T(ctx, i18n.MsgTaskInterruptedByRestart),
			}).Error; err != nil {
			slog.Error("标记中断记录失败", "task_id", task.ID, "error", err)
			continue
		}
		var successCount, failedCount int64
		model.DB(ctx).Model(&model.SkillDistributionRecord{}).Where("task_id = ? AND status = ?", task.ID, "success").Count(&successCount)
		model.DB(ctx).Model(&model.SkillDistributionRecord{}).Where("task_id = ? AND status = ?", task.ID, "failed").Count(&failedCount)
		if err := model.DB(ctx).Model(&task).Updates(map[string]interface{}{
			"status":  "completed",
			"success": int(successCount),
			"failed":  int(failedCount),
		}).Error; err != nil {
			slog.Error("更新任务状态失败", "task_id", task.ID, "error", err)
			continue
		}
		slog.Info("恢复中断的技能下发任务", "task_id", task.ID, "success", successCount, "failed", failedCount)
	}
}

// recoverInterruptedUpgradeAndMigrate 恢复重启中断的升级/迁移：
// upgrade/migrate 异步 goroutine 丢失时，将 processing → failed，供用户重试。
func recoverInterruptedUpgradeAndMigrate(ctx context.Context) {
	result := model.DB(ctx).Model(&model.Instance{}).
		Where("current_operation IN ?", []string{model.OpUpgrade, model.OpMigrate}).
		Where("current_operation_state = ?", model.OpStateProcessing).
		Updates(map[string]interface{}{
			"current_operation_state":      model.OpStateFailed,
			"current_operation_updated_at": time.Now(),
		})
	if result.Error != nil {
		slog.Error("恢复中断的升级/迁移操作失败", "error", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		slog.Info("恢复中断的升级/迁移操作", "affected", result.RowsAffected)
	}
}
