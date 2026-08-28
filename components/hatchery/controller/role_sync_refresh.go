package controller

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"hatchery/model"

	"gorm.io/gorm"
)

// refreshRoleRecord 聚合评估某个实例最新 updating record 的状态。
//
// 触发点：
//  1. SetInstanceSoul / RemoveInstanceSoul 完成后（成功/失败均调用）
//  2. 周期任务 role-sync-refresh 每 2 分钟扫 role_sync_status='updating' 的实例
//  3. 启动恢复不调本函数（直接批量标 failed）
//
// 幂等性：多次调用同一 recordID 不会撞车，`WHERE status='updating'` 保证只在中间态更新。
// 已 finalize（updated/failed/cancelled）的 record 不会被再次改写。
func refreshRoleRecord(ctx context.Context, instanceID uint) {
	logger := slog.With("task", "refreshRoleRecord", "instance_id", instanceID)

	record, ok := loadActiveUpdatingRecord(ctx, instanceID)
	if !ok {
		return
	}

	skillStatus, skillError := aggregateSkillDiffStatus(ctx, record)
	updates := buildRefreshUpdates(record, skillStatus, skillError)
	if len(updates) == 0 {
		return
	}

	// WHERE status='updating' 保证不覆盖已 finalize 的 record
	if err := model.DB(ctx).Model(&model.RoleDistributionRecord{}).
		Where("id = ? AND status = ?", record.ID, model.RoleRecordStatusUpdating).
		Updates(updates).Error; err != nil {
		logger.Warn("更新 record 失败", "record_id", record.ID, "error", err)
		return
	}

	// 同步 instance.role_sync_status（只在有终态跃迁时改）
	if newStatus, ok := updates["status"].(string); ok && newStatus != model.RoleRecordStatusUpdating {
		if err := model.DB(ctx).Model(&model.Instance{}).
			Where("id = ?", instanceID).
			Update("role_sync_status", newStatus).Error; err != nil {
			logger.Warn("更新 instance.role_sync_status 失败", "error", err)
		}
	}
}

// loadActiveUpdatingRecord 加载该实例最新 updating record。
// 返回 false 表示不存在（可能已被别的路径 finalize，或从未创建）。
func loadActiveUpdatingRecord(ctx context.Context, instanceID uint) (*model.RoleDistributionRecord, bool) {
	var record model.RoleDistributionRecord
	err := model.DB(ctx).
		Where("instance_id = ? AND status = ?", instanceID, model.RoleRecordStatusUpdating).
		Order("id DESC").First(&record).Error
	if err != nil {
		return nil, false
	}
	return &record, true
}

// aggregateSkillDiffStatus 读 SkillInstallationIDs 对应记录，聚合本次差集的技能状态。
// 返回 (skill_status, skill_error)：
//   - 差集为空 → (success, "")
//   - 有 Failed → (failed, first_error)
//   - 全部 Success → (success, "")
//   - 其他（存在 None/Installing 或部分成功）→ (running, "")
func aggregateSkillDiffStatus(ctx context.Context, record *model.RoleDistributionRecord) (string, string) {
	ids := parseSkillInstallationIDs(record.SkillInstallationIDs)
	if len(ids) == 0 {
		// 差集为空：新角色技能都已装 or 无角色技能
		return model.RoleSubStatusSuccess, ""
	}

	var installs []model.SkillInstallation
	if err := model.DB(ctx).Where("id IN ?", ids).Find(&installs).Error; err != nil {
		slog.Warn("[refreshRoleRecord] 查询 SkillInstallation 差集失败", "error", err)
		return model.RoleSubStatusRunning, "" // 失败时保守视为进行中，等下次周期任务再评
	}
	return classifySkillInstalls(installs, len(ids))
}

// parseSkillInstallationIDs 解析 record.SkillInstallationIDs JSON。
// 空串或格式错误返回空切片。
func parseSkillInstallationIDs(raw string) []uint {
	if raw == "" {
		return nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		slog.Warn("[refreshRoleRecord] 解析 SkillInstallationIDs 失败", "raw", raw, "error", err)
		return nil
	}
	return ids
}

// classifySkillInstalls 根据每条 SkillInstallation 的 install_status 分类。
// expectedCount：本次差集应有的记录数；若实际 < expected 说明有记录被删除，视为进行中。
// 返回 (skill_status, skill_error)：
//   - 差集为空 → (success, "")
//   - 有 Failed → (failed, "err1; err2")，汇总全部失败技能的 ErrorMessage
//   - 全部 Success → (success, "")
//   - 其他（存在 None/Installing 或部分成功）→ (running, "")
//
// 注意：ErrorMessage 由 installSkillsAsync 写入时已包含 slug 上下文（如 i18n 的 %s 占位），
// 此处不再重复拼接 slug 前缀。
func classifySkillInstalls(installs []model.SkillInstallation, expectedCount int) (string, string) {
	if len(installs) < expectedCount {
		// 记录被物理删除（异常场景），保守视为进行中
		return model.RoleSubStatusRunning, ""
	}

	successCount := 0
	failedCount := 0
	var failedMsgs []string
	for _, s := range installs {
		switch s.InstallStatus {
		case model.SkillInstallFailed:
			failedCount++
			if s.ErrorMessage != "" {
				failedMsgs = append(failedMsgs, s.ErrorMessage)
			}
		case model.SkillInstallSuccess:
			successCount++
		}
	}
	if failedCount > 0 {
		// 汇总全部失败技能的错误，截断防止 TEXT 字段过长
		msg := strings.Join(failedMsgs, "; ")
		if len(msg) > 1000 {
			msg = msg[:1000] + "..."
		}
		return model.RoleSubStatusFailed, msg
	}
	if successCount == len(installs) {
		return model.RoleSubStatusSuccess, ""
	}
	// 有 None/Installing 状态的技能（SMH 未同步完成或安装进行中），视为进行中
	return model.RoleSubStatusRunning, ""
}

// buildRefreshUpdates 根据当前 SOUL/skill 子状态计算 record 应有的字段更新。
// 返回空 map 表示无需更新（当前状态已经准确）。
func buildRefreshUpdates(record *model.RoleDistributionRecord, skillStatus, skillError string) map[string]interface{} {
	updates := map[string]interface{}{}

	if skillStatus != record.SkillStatus {
		updates["skill_status"] = skillStatus
	}
	if skillError != record.SkillError {
		updates["skill_error"] = skillError
	}
	if skillStatus == model.RoleSubStatusSuccess && record.SkillSetAt == nil {
		now := time.Now()
		updates["skill_set_at"] = &now
	}

	newRecordStatus := deriveRecordStatus(record.SoulStatus, skillStatus)
	if newRecordStatus != record.Status {
		updates["status"] = newRecordStatus
	}

	return updates
}

// deriveRecordStatus 从 soul_status + skill_status 派生 record.status。
func deriveRecordStatus(soulStatus, skillStatus string) string {
	if soulStatus == model.RoleSubStatusSuccess && skillStatus == model.RoleSubStatusSuccess {
		return model.RoleRecordStatusUpdated
	}
	if soulStatus == model.RoleSubStatusFailed || skillStatus == model.RoleSubStatusFailed {
		return model.RoleRecordStatusFailed
	}
	return model.RoleRecordStatusUpdating
}

// finalizeActiveRecordAsCancelled 把该实例当前 updating record 标 cancelled。
// 用于：apply 前把老 record 结束；remove-role/switch-role role_id=0 时结束当前 record。
// 返回被 finalize 的 record ID（无 record 则返回 0）。
func finalizeActiveRecordAsCancelled(tx *gorm.DB, instanceID uint) uint {
	var record model.RoleDistributionRecord
	err := tx.Where("instance_id = ? AND status = ?", instanceID, model.RoleRecordStatusUpdating).
		Order("id DESC").First(&record).Error
	if err != nil {
		return 0
	}
	if err := tx.Model(&record).Where("status = ?", model.RoleRecordStatusUpdating).
		Update("status", model.RoleRecordStatusCancelled).Error; err != nil {
		slog.Error("[finalizeActiveRecordAsCancelled] 更新 record 为 cancelled 失败", "record_id", record.ID, "error", err)
	}
	return record.ID
}

// updateRecordSoulStatus SOUL 端调用：更新 record 的 soul_status + soul_error + soul_set_at。
// recordID = 0 时无操作（用于不带 record 的兜底路径）。
// WHERE status='updating' 保证不覆盖已 finalize 的 record。
func updateRecordSoulStatus(ctx context.Context, recordID uint, soulStatus, soulError string) {
	if recordID == 0 {
		return
	}
	updates := map[string]interface{}{
		"soul_status": soulStatus,
		"soul_error":  soulError,
	}
	if soulStatus == model.RoleSubStatusSuccess {
		now := time.Now()
		updates["soul_set_at"] = &now
	}
	if err := model.DB(ctx).Model(&model.RoleDistributionRecord{}).
		Where("id = ? AND status = ?", recordID, model.RoleRecordStatusUpdating).
		Updates(updates).Error; err != nil {
		slog.Warn("[updateRecordSoulStatus] 更新 record soul 状态失败",
			"record_id", recordID, "soul_status", soulStatus, "error", err)
	}
}

// findLatestUpdatingRecordID 反查该实例当前 updating record ID。
// 用于 task/soul.go 兜底触发 SetInstanceSoul 时挂钩到正确的 record。
// 无则返回 0（此时 SetInstanceSoul 内部会跳过 record 写入）。
func findLatestUpdatingRecordID(ctx context.Context, instanceID uint) uint {
	var record model.RoleDistributionRecord
	err := model.DB(ctx).Select("id").
		Where("instance_id = ? AND status = ?", instanceID, model.RoleRecordStatusUpdating).
		Order("id DESC").First(&record).Error
	if err != nil {
		return 0
	}
	return record.ID
}

// FindLatestUpdatingRoleRecordID 是 findLatestUpdatingRecordID 的导出版，
// 供 task 包（如 task/soul.go）跨包调用。
func FindLatestUpdatingRoleRecordID(ctx context.Context, instanceID uint) uint {
	return findLatestUpdatingRecordID(ctx, instanceID)
}

// RefreshRoleRecord 是 refreshRoleRecord 的导出版，供 task 包跨包调用。
func RefreshRoleRecord(ctx context.Context, instanceID uint) {
	refreshRoleRecord(ctx, instanceID)
}
