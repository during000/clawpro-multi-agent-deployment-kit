package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"
)

// taskRunScriptFn 是 controller.RunScript 的可替换包装，方便单元测试 mock TAT 调用。
var taskRunScriptFn = controller.RunScript

// handleSwitchToFree 处理 OFF -> FREE 切换。
func handleSwitchToFree(job *model.TdaiJob) error {
	ctx := jobCtx(job)
	log := jobLogger(job)
	// Step 1: validate
	updateStep(job, "validate_request", 10)
	if err := checkInstanceSupportsMemoryTask(ctx, job.InstanceID); err != nil {
		return err
	}
	plugin, err := getPluginByInstanceID(ctx, job.InstanceID)
	if err != nil {
		return err
	}
	log.Info("[SwitchToFree] validate 完成",
		"current_plan", plugin.CurrentPlan,
		"desired_plan", plugin.DesiredPlan,
		"switch_status", plugin.SwitchStatus,
	)
	if plugin.CurrentPlan == model.MemoryPlanFree {
		log.Info("[SwitchToFree] 实例已在 FREE，跳过")
		if err := model.DB(ctx).Model(plugin).Updates(map[string]any{
			"switch_status": model.MemorySwitchStatusNone,
			"last_task_id":  job.ID,
		}).Error; err != nil {
			log.Warn("[SwitchToFree] 重置 switch_status 失败", "error", err)
		}
		return nil
	}
	if plugin.CurrentPlan == model.MemoryPlanPro {
		return NewNonRetryableError(i18n.T(ctx, i18n.MsgProToFreeNotSupported))
	}

	// Step 2: 等待实例就绪（CVM RUNNING）
	updateStep(job, "wait_instance_ready", 20)
	if err := checkInstanceRunning(ctx, job.InstanceID); err != nil {
		return err // 可重试错误，退避后自动重试
	}
	log.Info("[SwitchToFree] 实例已就绪")

	// Step 3: Hermes 实例先执行 install 脚本（安装插件），再 ensure 校验
	// 兼容 hermes 的自定义类型走相同路径。
	agentType := controller.LookupAgentType(ctx, job.InstanceID)
	runtimeUser := controller.LookupRuntimeUser(ctx, job.InstanceID)
	if model.GetAgentRuntimeType(ctx, agentType) == model.AgentTypeHermes {
		updateStep(job, "install_plugin", 25)
		if _, err := taskRunScriptFn(ctx, job.InstanceID, "install_hermes_tdai_gateway.sh", 600, runtimeUser, nil, nil); err != nil {
			log.Warn("[SwitchToFree] Hermes install 脚本执行失败",
				"error", err)
			return hcommon.I18nRichError(err, i18n.MsgTDAIHermesInstallFailed)
		}
	}

	updateStep(job, "ensure_plugin", 30)
	ensureStart := time.Now()
	log.Info("[SwitchToFree] 调用 EnsureMemoryPlugin 开始")
	if err := controller.EnsureMemoryPlugin(ctx, job.InstanceID); err != nil {
		log.Warn("[SwitchToFree] 插件就绪检查失败",
			"cost", time.Since(ensureStart).String(),
			"error", err)
		logTATDetail(ctx, log, "[SwitchToFree]", err)
		return hcommon.I18nRichError(err, i18n.MsgTDAIPluginReadyCheckFailed)
	}
	log.Info("[SwitchToFree] EnsureMemoryPlugin 完成", "cost", time.Since(ensureStart).String())

	// Step 4: enable free plugin
	updateStep(job, "enable_free_plugin", 40)
	cfg := model.GetSiteConfig(ctx)
	pluginName := model.DefaultMemoryTDAIPluginName
	supportedVersions, _, _ := model.NormalizeMemoryTDAISupportedVersions(cfg.MemoryTDAISupportedVersions)
	runtimeUser = controller.LookupRuntimeUser(ctx, job.InstanceID)

	enableStart := time.Now()
	log.Info("[SwitchToFree] 调用 doEnablePlugin 开始（执行 memory_tdai_switch_free.sh）",
		"plugin", pluginName,
		"runtime_user", runtimeUser,
		"agent_type", agentType,
	)
	if err := doEnablePlugin(ctx, job.InstanceID, runtimeUser, agentType, pluginName, supportedVersions, plugin); err != nil {
		log.Warn("[SwitchToFree] 启用 Free 插件失败",
			"cost", time.Since(enableStart).String(),
			"error", err)
		logTATDetail(ctx, log, "[SwitchToFree]", err)
		return hcommon.I18nRichError(err, i18n.MsgTDAIEnableFreePluginFailed)
	}
	log.Info("[SwitchToFree] doEnablePlugin 完成", "cost", time.Since(enableStart).String())

	// Step 4: update plan status
	updateStep(job, "update_plan_status", 80)
	now := time.Now()
	if err := model.DB(ctx).Model(plugin).Updates(map[string]any{
		"current_plan":     model.MemoryPlanFree,
		"desired_plan":     model.MemoryPlanFree,
		"switch_status":    model.MemorySwitchStatusNone,
		"last_task_id":     job.ID,
		"last_switched_at": &now,
	}).Error; err != nil {
		log.Error("[SwitchToFree] 更新 plan 状态失败", "error", err)
		return hcommon.I18nRichError(err, i18n.MsgTDAIUpdatePlanFailed)
	}
	log.Info("[SwitchToFree] 计划已更新为 FREE")

	updateStep(job, "mark_succeeded", 100)
	return nil
}

// handleSwitchToOff 处理 FREE/PRO -> OFF 切换。
func handleSwitchToOff(job *model.TdaiJob) error {
	ctx := jobCtx(job)
	log := jobLogger(job)
	// Step 1: validate
	updateStep(job, "validate_request", 10)
	if err := checkInstanceSupportsMemoryTask(ctx, job.InstanceID); err != nil {
		return err
	}
	plugin, err := getPluginByInstanceID(ctx, job.InstanceID)
	if err != nil {
		return err
	}
	log.Info("[SwitchToOff] validate 完成",
		"current_plan", plugin.CurrentPlan,
		"desired_plan", plugin.DesiredPlan,
		"switch_status", plugin.SwitchStatus,
		"pool_id", plugin.PoolID,
	)
	if plugin.CurrentPlan == model.MemoryPlanOff {
		log.Info("[SwitchToOff] 实例已在 OFF，跳过")
		if err := model.DB(ctx).Model(plugin).Updates(map[string]any{
			"switch_status": model.MemorySwitchStatusNone,
			"last_task_id":  job.ID,
		}).Error; err != nil {
			log.Warn("[SwitchToOff] 重置 switch_status 失败", "error", err)
		}
		return nil
	}

	// Step 2: 等待实例就绪（CVM RUNNING）
	updateStep(job, "wait_instance_ready", 20)
	if err := checkInstanceRunning(ctx, job.InstanceID); err != nil {
		return err
	}
	log.Info("[SwitchToOff] 实例已就绪")

	// Step 3: disable plugin（含 Pro→OFF 数据备份 + 清理 VDB 配置）
	// 注意：必须先 backup 再释放远端 VDB 库，否则数据丢失
	// Pro→OFF 时由 hatchery 侧先做 VDB 连通性预检，决定是否跳过 export，
	// 透传 skip_export 给 disable 脚本。disable 脚本只负责执行，不做网络判断。
	updateStep(job, "disable_plugin", 35)
	agentType := controller.LookupAgentType(ctx, job.InstanceID)

	// 探测插件根目录（仅 OpenClaw 需要）
	pluginRoot := ""
	if agentType != model.AgentTypeHermes {
		var resolveErr error
		pluginRoot, resolveErr = controller.ResolveMemoryPluginRoot(ctx, job.InstanceID)
		if resolveErr != nil {
			log.Warn("[SwitchToOff] 插件路径探测失败（非阻断，使用空值）", "error", resolveErr)
		}
	}

	disableParams := map[string]string{
		"plugin":           model.DefaultMemoryTDAIPluginName,
		"clear_pro_config": "false",
		"vdb_endpoint":     "",
		"vdb_database":     "",
		"vdb_api_key":      "",
		"vdb_username":     "",
		"job_id":           fmt.Sprintf("%d", job.ID),
		"agent_type":       agentType,
		"skip_export":      "false",
		"plugin_root":      pluginRoot,
	}
	if plugin.CurrentPlan == model.MemoryPlanPro {
		disableParams["clear_pro_config"] = "true"
		disableParams["vdb_endpoint"] = plugin.Endpoint
		disableParams["vdb_database"] = plugin.DatabaseName
		disableParams["vdb_api_key"] = plugin.ApiKeySecretRef
		disableParams["vdb_username"] = plugin.VdbUsername
		// VDB 连通性预检：网络不通时跳过 export（避免必然超时阻断 OFF 流程）
		// shouldSkipVDBExportOnDisable 永不抛错，预检本身的异常按"通"保守处理，
		// 不会阻断 OFF 这种"止损"动作。
		// 必须传 username/apiKey，否则 VDB 网关会对无鉴权请求直接 RST，无法与"网络真不通"区分。
		precheckStart := time.Now()
		if shouldSkipVDBExportOnDisable(ctx, job.InstanceID, plugin.Endpoint, plugin.VdbUsername, plugin.ApiKeySecretRef) {
			disableParams["skip_export"] = "true"
			log.Warn("[SwitchToOff] VDB 连通性预检判定不通，本次 disable 将跳过 VDB 数据 export",
				"endpoint", plugin.Endpoint,
				"cost", time.Since(precheckStart).String(),
			)
		} else {
			log.Info("[SwitchToOff] VDB 连通性预检通过（或不可判定，保守按通处理）",
				"endpoint", plugin.Endpoint,
				"cost", time.Since(precheckStart).String(),
			)
		}
	}
	disableStart := time.Now()
	log.Info("[SwitchToOff] 调用 doDisablePluginWithParams 开始（执行 memory_tdai_disable.sh）",
		"current_plan", plugin.CurrentPlan,
		"clear_pro_config", disableParams["clear_pro_config"],
		"skip_export", disableParams["skip_export"],
	)
	if err := doDisablePluginWithParams(ctx, job.InstanceID, disableParams, plugin); err != nil {
		log.Warn("[SwitchToOff] 禁用插件失败",
			"cost", time.Since(disableStart).String(),
			"current_plan", plugin.CurrentPlan, "error", err)
		logTATDetail(ctx, log, "[SwitchToOff]", err)
		return hcommon.I18nRichError(err, i18n.MsgTDAIDisablePluginFailed)
	}
	log.Info("[SwitchToOff] 插件禁用成功", "cost", time.Since(disableStart).String())

	// Step 4: 若当前为 Pro，备份完成后再释放远端 VDB 库 + 清理 DB 绑定信息
	if plugin.CurrentPlan == model.MemoryPlanPro {
		updateStep(job, "release_database", 65)
		releaseStart := time.Now()
		log.Info("[SwitchToOff] 调用 handleDeleteMemSpace 开始", "pool_id", plugin.PoolID)
		if err := handleDeleteMemSpace(ctx, plugin); err != nil {
			log.Warn("[SwitchToOff] 释放 Pro 记忆库失败",
				"pool_id", plugin.PoolID,
				"cost", time.Since(releaseStart).String(),
				"error", err)
			// 释放失败：回滚流程，保留本地绑定信息，便于后续重试或手动清理
			return hcommon.I18nRichError(err, i18n.MsgTDAIReleaseProMemFailed)
		}
		log.Info("[SwitchToOff] handleDeleteMemSpace 完成",
			"pool_id", plugin.PoolID,
			"cost", time.Since(releaseStart).String())
		// 释放成功后才清理 DB 绑定信息
		log.Info("[SwitchToOff] 远端记忆库已释放，清理 DB 绑定信息",
			"pool_id", plugin.PoolID)
		if err := model.DB(ctx).Model(plugin).Updates(map[string]any{
			"pool_id":            "",
			"database_name":      "",
			"endpoint":           "",
			"api_key_secret_ref": "",
			"vdb_username":       "",
			"embedding_model":    "",
		}).Error; err != nil {
			log.Warn("[SwitchToOff] 清理 DB 绑定信息失败", "error", err)
		}
	}

	// Step 5: update plan status
	updateStep(job, "update_plan_status", 85)
	now := time.Now()
	if err := model.DB(ctx).Model(plugin).Updates(map[string]any{
		"current_plan":     model.MemoryPlanOff,
		"desired_plan":     model.MemoryPlanOff,
		"switch_status":    model.MemorySwitchStatusNone,
		"last_task_id":     job.ID,
		"last_switched_at": &now,
	}).Error; err != nil {
		log.Error("[SwitchToOff] 更新 plan 状态失败", "error", err)
		return hcommon.I18nRichError(err, i18n.MsgTDAIUpdatePlanFailed)
	}
	log.Info("[SwitchToOff] 计划已更新为 OFF")

	updateStep(job, "mark_succeeded", 100)
	return nil
}

// --- helper ---

// jobLogger 为后台任务返回带完整 trace 字段的 logger。
//
// 优先从 jobCtxRegistry 取出 dispatcher 在 executeJob 入口注册的 task ctx，
// 用 controller.Logger(ctx) 构造，使 handler 内的日志自动携带 request_id / trace_id 等链路字段；
// 再 With job 维度字段（job_id / job_type / instance_id / attempt），便于按 job_id 追溯一次任务的所有日志。
//
// 若 jobCtxRegistry 中无对应记录（例如单元测试直接调用 handler），降级为只携带 job 字段的 logger，
// 保证向下兼容。
func jobLogger(job *model.TdaiJob) *slog.Logger {
	ctx := lookupJobCtx(job.ID)
	return controller.Logger(ctx).With(
		"job_id", job.ID,
		"job_type", job.JobType,
		"instance_id", job.InstanceID,
		"attempt", job.Attempt,
	)
}

// jobCtx 返回当前 job 关联的 task ctx，handler 调用 DB / SDK 时应传入，
// 让 GORM / SDK 自动日志也能携带统一的 request_id / trace_id。
// 测试场景下未注册时返回 context.Background()。
func jobCtx(job *model.TdaiJob) context.Context {
	return lookupJobCtx(job.ID)
}

func updateStep(job *model.TdaiJob, step string, progress int) {
	log := jobLogger(job)
	log.Info("[TaskEngine] 步骤推进", "step", step, "progress", progress)
	if err := model.DB(jobCtx(job)).Model(job).Updates(map[string]any{
		"current_step": step,
		"progress":     progress,
	}).Error; err != nil {
		log.Warn("[TaskEngine] 更新步骤进度失败", "step", step, "progress", progress, "error", err)
	}
}

func getPluginByInstanceID(ctx context.Context, instanceID string) (*model.MemoryTDAIPlugin, error) {
	var plugin model.MemoryTDAIPlugin
	if err := model.DB(ctx).Where("instance_id = ?", instanceID).First(&plugin).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgTDAIQueryPluginRowFailed, instanceID)
	}
	return &plugin, nil
}

// logTATDetail 在 TAT 脚本调用失败时，从 RichError.Detail 提取脚本的部分输出并打印为独立的 WARN 日志。
// TAT 超时/失败时，Detail 里包含脚本在超时前已经打印到 stdout 的内容（最多 24KB），
// 是定位"脚本卡在哪一步"的关键信息。
// 调用方传入 jobLogger 和 tag（如 "[SwitchToFree]"），确保日志带完整 trace 字段。
func logTATDetail(ctx context.Context, log *slog.Logger, tag string, err error) {
	var re *hcommon.RichError
	if errors.As(err, &re) && hcommon.ErrorDetailWithCtx(ctx, re) != "" {
		log.Warn(tag+" TAT 脚本输出（部分）",
			"tat_message", hcommon.ErrorMessageWithCtx(ctx, re),
			"tat_output", hcommon.ErrorDetailWithCtx(ctx, re),
		)
	}
}

// checkInstanceSupportsMemoryTask 任务执行器保险：若实例 agent_type 不支持记忆功能，
// 返回 NonRetryableError，任务直接终态失败，避免无意义重试。
// 正常流程下 HTTP 入口已拦截，这里作为兜底防护。
func checkInstanceSupportsMemoryTask(ctx context.Context, instanceID string) error {
	var inst model.Instance
	if err := model.DB(ctx).Where("instance_id = ?", instanceID).First(&inst).Error; err != nil {
		// 实例不存在按可重试处理，等上游创建完成
		return hcommon.I18nRichError(err, i18n.MsgTDAIQueryInstanceFailed, instanceID)
	}
	if !model.AgentTypeSupportsMemory(ctx, inst.AgentType) {
		return NewNonRetryableError(
			i18n.T(ctx, i18n.MsgTDAIAgentTypeNotSupportMemory, instanceID, inst.AgentType))
	}
	return nil
}

// checkInstanceRunning 检查实例 CVM 是否处于 RUNNING 状态。
// 若实例不存在或非 RUNNING，返回可重试错误（退避后自动重试，等待实例创建完成）。
// 注：本函数无 job 上下文，故内部不打日志，状态信息通过返回值传出。
// 由 caller（handler）通过 jobLogger 携带 trace 字段输出，便于按 job_id / request_id 追溯。
func checkInstanceRunning(ctx context.Context, instanceID string) error {
	var inst model.Instance
	if err := model.DB(ctx).Where("instance_id = ?", instanceID).First(&inst).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgTDAIInstanceNotFound, instanceID)
	}
	if inst.LastCVMState != "RUNNING" {
		return hcommon.I18nError(i18n.MsgTDAIInstanceNotReady, instanceID, inst.LastCVMState)
	}
	return nil
}

// doSwitchFree 调用 switch_free 脚本，安装并启用插件（Free 模式）。
// 注：当前已由 handleSwitchToFree → doEnablePlugin 路径替代，保留以备回退/调试用。
func doSwitchFree(ctx context.Context, instanceID, pluginName, supportedVersions string, plugin *model.MemoryTDAIPlugin) error {
	model.DB(ctx).Model(plugin).Updates(map[string]any{
		"status":     model.MemoryTDAIPluginStatusEnabling,
		"last_error": "",
	})

	runtimeUser := controller.LookupRuntimeUser(ctx, instanceID)
	agentType := controller.LookupAgentType(ctx, instanceID)

	// Hermes：先下发 install 脚本（幂等）；兼容 hermes 的自定义类型走相同路径
	if model.GetAgentRuntimeType(ctx, agentType) == model.AgentTypeHermes {
		if _, err := taskRunScriptFn(ctx, instanceID, "install_hermes_tdai_gateway.sh", 600, runtimeUser, nil, nil); err != nil {
			slog.Warn("[SwitchToFree] Hermes install 脚本执行失败（非阻断）",
				"instance_id", instanceID, "error", err)
		}
	}

	_, err := taskRunScriptFn(ctx, instanceID, "memory_tdai_switch_free.sh", 300, runtimeUser, nil,
		map[string]string{
			"plugin":             pluginName,
			"supported_versions": supportedVersions,
			"agent_type":         agentType,
		})
	if err != nil {
		return handlePluginScriptError(ctx, plugin, err)
	}

	model.DB(ctx).Model(plugin).Updates(map[string]any{
		"status":      model.MemoryTDAIPluginStatusEnabled,
		"last_error":  "",
		"retry_count": 0,
	})
	return nil
}

// doDisablePluginWithParams 调用 disable 脚本，支持 Pro→OFF 场景的数据备份和 VDB 配置清理。
func doDisablePluginWithParams(ctx context.Context, instanceID string, params map[string]string, plugin *model.MemoryTDAIPlugin) error {
	model.DB(ctx).Model(plugin).Updates(map[string]any{
		"status":     model.MemoryTDAIPluginStatusDisabling,
		"last_error": "",
	})

	runtimeUser := controller.LookupRuntimeUser(ctx, instanceID)
	_, err := taskRunScriptFn(ctx, instanceID, "memory_tdai_disable.sh", 600, runtimeUser, nil, params)
	if err != nil {
		return handlePluginScriptError(ctx, plugin, err)
	}

	model.DB(ctx).Model(plugin).Updates(map[string]any{
		"status":      model.MemoryTDAIPluginStatusDisabled,
		"last_error":  "",
		"retry_count": 0,
	})
	return nil
}

// doEnablePlugin 调用 switch_free 脚本启用 Free 模式插件。
// 由 handleSwitchToFree 调用。
func doEnablePlugin(ctx context.Context, instanceID, runtimeUser, agentType, pluginName, supportedVersions string, plugin *model.MemoryTDAIPlugin) error {
	model.DB(ctx).Model(plugin).Updates(map[string]any{
		"status":     model.MemoryTDAIPluginStatusEnabling,
		"last_error": "",
	})

	// 注：install 脚本和 EnsureMemoryPlugin 已由外层 handleSwitchToFree / handleSwitchToPro 执行，
	// 此处不再重复调用，避免同一次切换中多次 TAT 下发。
	_, err := taskRunScriptFn(ctx, instanceID, "memory_tdai_switch_free.sh", 300, runtimeUser, nil,
		map[string]string{
			"plugin":             pluginName,
			"supported_versions": supportedVersions,
			"agent_type":         agentType,
		})
	if err != nil {
		return handlePluginScriptError(ctx, plugin, err)
	}

	model.DB(ctx).Model(plugin).Updates(map[string]any{
		"status":      model.MemoryTDAIPluginStatusEnabled,
		"last_error":  "",
		"retry_count": 0,
	})
	return nil
}

// handlePluginScriptError 脚本失败：记录错误、retry_count+1。
// 特殊处理：脚本输出含 UNSUPPORTED_VERSION 时，标记为 UNSUPPORTED_VERSION 状态，不消耗重试次数。
func handlePluginScriptError(ctx context.Context, plugin *model.MemoryTDAIPlugin, scriptErr error) error {
	errMsg := scriptErr.Error()
	// 提取 RichError.Detail（含脚本输出），写入 last_error 方便排查
	if re, ok := scriptErr.(*hcommon.RichError); ok && hcommon.ErrorDetailWithCtx(ctx, re) != "" {
		errMsg = hcommon.ErrorMessageWithCtx(ctx, re) + ": " + hcommon.ErrorDetailWithCtx(ctx, re)
	}

	// 检测版本不支持
	if strings.Contains(strings.ToUpper(errMsg), "UNSUPPORTED_VERSION") {
		model.DB(ctx).Model(plugin).Updates(map[string]any{
			"status":     model.MemoryTDAIPluginStatusUnsupportedVersion,
			"last_error": errMsg,
		})
		return errors.New(errMsg)
	}

	model.DB(ctx).Model(plugin).Updates(map[string]any{
		"status":      model.MemoryTDAIPluginStatusFailed,
		"last_error":  errMsg,
		"retry_count": plugin.RetryCount + 1,
	})
	return errors.New(errMsg)
}
