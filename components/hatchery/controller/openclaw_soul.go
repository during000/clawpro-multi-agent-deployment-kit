package controller

import (
	"context"
	"encoding/base64"
	"log/slog"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// SetInstanceSoul 将实例绑定的角色 Soul 下发到 CVM 实例。
// 仅当实例满足以下条件时执行：
//   - RuntimeUser 已检测（否则无法确定写入路径）
//   - RoleID > 0（有关联角色）
//   - role 存在且 Soul 非空
//
// 成功后写入 soul_set_at 时间戳；失败返回 error 供调用方决定是否重试。
//
// recordID > 0 时同时更新 role_distribution_records 的 soul_status/soul_error/soul_set_at，
// 并触发 refreshRoleRecord 聚合状态。recordID = 0（老 remove-role 兜底 / task/soul.go）时
// 跳过 record 写入，仅更新 instance.soul_set_at。
func SetInstanceSoul(ctx context.Context, instancePK uint, recordID uint) error {
	logger := slog.With("task", "SetInstanceSoul", "instance_pk", instancePK, "record_id", recordID)

	instance, role, skip, err := loadSetSoulPreconditions(ctx, instancePK)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}

	// 标记 record 进入 running（若绑定了 record）
	updateRecordSoulStatus(ctx, recordID, model.RoleSubStatusRunning, "")

	if err := runSetSoulScript(ctx, instance, role); err != nil {
		logger.Error("下发 Soul 失败", "error", err)
		updateRecordSoulStatus(ctx, recordID, model.RoleSubStatusFailed, hcommon.ErrorMessageWithCtx(ctx, err))
		refreshRoleRecordIfBound(ctx, instancePK, recordID)
		return err
	}

	if err := markSoulSetAt(ctx, instance); err != nil {
		logger.Error("更新 soul_set_at 失败", "error", err)
		updateRecordSoulStatus(ctx, recordID, model.RoleSubStatusFailed, hcommon.ErrorMessageWithCtx(ctx, err))
		refreshRoleRecordIfBound(ctx, instancePK, recordID)
		return err
	}

	updateRecordSoulStatus(ctx, recordID, model.RoleSubStatusSuccess, "")
	refreshRoleRecordIfBound(ctx, instancePK, recordID)

	logger.Info("Soul 下发成功", "role_name", role.Name)
	return nil
}

// loadSetSoulPreconditions 加载实例 + 角色，检查是否可以下发。
// skip=true 表示前置条件不满足（无角色 / RuntimeUser 未探测 / agent_type 不支持 / soul 为空），跳过下发。
func loadSetSoulPreconditions(ctx context.Context, instancePK uint) (*model.Instance, *model.OpenClawRole, bool, error) {
	var instance model.Instance
	if err := model.DB(ctx).First(&instance, instancePK).Error; err != nil {
		return nil, nil, false, err
	}
	if instance.RoleID == 0 || instance.RuntimeUser == "" {
		return &instance, nil, true, nil
	}
	if !model.AgentTypeSupportsRole(ctx, instance.AgentType) {
		return &instance, nil, true, nil
	}
	var role model.OpenClawRole
	if err := model.DB(ctx).First(&role, instance.RoleID).Error; err != nil {
		return &instance, nil, false, err
	}
	if role.Soul == "" {
		return &instance, &role, true, nil
	}
	return &instance, &role, false, nil
}

// runSetSoulScript 通过 TAT 下发 SOUL.md。
func runSetSoulScript(ctx context.Context, instance *model.Instance, role *model.OpenClawRole) error {
	soulB64 := base64.StdEncoding.EncodeToString([]byte(role.Soul))
	_, err := RunAgentScript(ctx, instance, "set_soul", 30, nil, map[string]string{
		"soul_b64": soulB64,
	})
	return err
}

// markSoulSetAt 写入 instance.soul_set_at。
func markSoulSetAt(ctx context.Context, instance *model.Instance) error {
	now := time.Now()
	if err := model.DB(ctx).Model(instance).Update("soul_set_at", now).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSoulSetAtUpdateFailed)
	}
	return nil
}

// refreshRoleRecordIfBound 若绑定了 record，触发聚合评估。
func refreshRoleRecordIfBound(ctx context.Context, instancePK uint, recordID uint) {
	if recordID == 0 {
		return
	}
	refreshRoleRecord(ctx, instancePK)
}

// RemoveInstanceSoul 移除 CVM 实例上的 SOUL.md 文件并重启 Gateway。
// 当用户移除实例角色时调用。成功返回 nil，失败返回 error 供调用方返回给用户。
//
// recordID > 0 时同时更新 record（尽管 role_id=0 的场景 apply 时已 finalize 老 record 为 cancelled，
// 一般不会传 recordID > 0；此参数保留是为未来扩展）。
func RemoveInstanceSoul(ctx context.Context, instancePK uint, recordID uint) error {
	logger := slog.With("task", "RemoveInstanceSoul", "instance_pk", instancePK, "record_id", recordID)

	var instance model.Instance
	if err := model.DB(ctx).First(&instance, instancePK).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed)
	}

	if instance.RuntimeUser == "" {
		// 实例尚未就绪，没有 SOUL.md 需要删除，仅清除标记
		if err := model.DB(ctx).Model(&instance).Update("soul_set_at", nil).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgClearSoulSetAtFailed)
		}
		return nil
	}

	if !model.AgentTypeSupportsRole(ctx, instance.AgentType) {
		return nil
	}

	_, err := RunAgentScript(ctx, &instance, "remove_soul", 30, nil, nil)
	if err != nil {
		logger.Warn("移除 Soul 失败", "error", err)
		updateRecordSoulStatus(ctx, recordID, model.RoleSubStatusFailed, hcommon.ErrorMessageWithCtx(ctx, err))
		refreshRoleRecordIfBound(ctx, instancePK, recordID)
		return hcommon.I18nError(i18n.MsgSoulRemoveFailed)
	}

	// 清除已下发标记
	if err := model.DB(ctx).Model(&instance).Update("soul_set_at", nil).Error; err != nil {
		logger.Error("清除 soul_set_at 失败", "error", err)
	}

	updateRecordSoulStatus(ctx, recordID, model.RoleSubStatusSuccess, "")
	refreshRoleRecordIfBound(ctx, instancePK, recordID)

	logger.Info("Soul 移除成功")
	return nil
}

// setInstanceSoulWhenReady 等待 RuntimeUser 就绪后立即下发 Soul。
// 用于实例创建流程中的显式下发（保证即时性）。
//
// RuntimeUser 由 detectAndSaveRuntimeUser 在 Agent 就绪后异步检测，Agent 就绪
// 意味着 CVM 已 RUNNING，因此这里无需再等待 CVM 状态。
//
// 最多等 10 分钟，超时或失败由 task 周期任务兜底重试。
func setInstanceSoulWhenReady(ctx context.Context, instancePK uint, cvmInstanceId string) {
	logger := slog.With("task", "setInstanceSoulWhenReady", "instance_pk", instancePK)

	// 等待 RuntimeUser 检测完成（最多 10 分钟，120 × 5s）
	var runtimeUser string
	for i := 0; i < 120; i++ {
		time.Sleep(5 * time.Second)
		var inst model.Instance
		if err := model.DB(ctx).Select("role_id, runtime_user").First(&inst, instancePK).Error; err != nil {
			logger.Warn("轮询实例失败", "error", err)
			return
		}
		if inst.RoleID == 0 {
			logger.Info("角色已被移除，取消下发")
			return
		}
		if inst.RuntimeUser != "" {
			runtimeUser = inst.RuntimeUser
			break
		}
	}

	if runtimeUser == "" {
		logger.Warn("等待 RuntimeUser 超时，交周期任务兜底")
		return
	}

	// 立即尝试下发，失败由周期任务兜底。反查 updating record 挂钩到该次 apply。
	recordID := findLatestUpdatingRecordID(ctx, instancePK)
	if err := SetInstanceSoul(ctx, instancePK, recordID); err != nil {
		logger.Warn("即时 Soul 下发失败，交周期任务兜底", "error", err)
	}
}
