package controller

import (
	"context"
	"net/http"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

// ─────────────────────────────────────────────────────────────────────
// commonHandleResetInstance —— 用户端 / 管控端重装入口的统一编排
//
// 设计意图：
//   - 入口 handler（HandleResetInstance / HandleAdminResetInstance、以及测试
//     专用的小写 handleResetInstance / handleAdminResetInstance）只负责注入
//     不同的鉴权 / 实例查询 / 状态准入策略，主流程一律走本函数。
//   - 主流程顺序与用户端历史版本完全一致：method check → 身份 → 实例 →
//     龙虾医生 guard → 类型 guard → 状态 guard → CVM 关联 guard → 乐观锁 →
//     镜像 + 版本信息重置 → CVM ResetInstance → 业务状态事务清理 → 后置异步任务。
//   - 管控端补齐了用户端原本独有的能力：
//       1. 结构化日志（统一 [Admin-]ResetInstance 前缀，带 instance_id / cvm_id /
//          agent_type 等关键上下文，便于排查）。
//       2. CVM ResetInstance 调用失败时给实例所有者发"实例重装失败"错误通知，
//          与用户自助重装失败的体验一致。
//       3. 通过 prepareReinstallImage / buildReinstallRequest /
//          resetReinstallBusinessState / kickOffReinstallAsyncTasks 等已有公共
//          步骤直接复用，保证 UserData 渲染、模型绑定清空、后置任务编排两端
//          完全对齐。
//
// 响应模式：本地 HTML 渲染已移除，用户端 / 管控端两条入口统一走纯 JSON API
// （入口处 jsonAPI(w)，成功 jsonOK，失败 writeError / writeAgentGuardError），
// 与重构后的 handleResetInstance / handleAdminResetInstance 保持一致。
//
// 不动现有方法：所有原有 helper（getInstanceByID、getAdminInstanceByIDOrInstanceID、
// requireActionAllowedFor{User,Admin}、setOperationWithAgentReset、
// resetInstanceVersionInfo 等）一律按现状复用，不修改任何签名 / 内部实现。
// ─────────────────────────────────────────────────────────────────────

// reinstallOpts 描述用户端与管控端两条入口的差异点。
type reinstallOpts struct {
	// isAdmin true 走管控端鉴权 + 实例查询 + 状态白名单，否则走用户端。
	isAdmin bool
	// logPrefix 用于日志前缀（"ResetInstance" / "Admin-ResetInstance"）。
	logPrefix string
	// actor 用于日志中的 actor 字段（"user" / "admin"），便于审计聚合。
	actor string
}

var (
	reinstallUserOpts  = reinstallOpts{isAdmin: false, logPrefix: "User-ResetInstance", actor: "user"}
	reinstallAdminOpts = reinstallOpts{isAdmin: true, logPrefix: "Admin-ResetInstance", actor: "admin"}
)

// commonHandleResetInstance 是 HandleResetInstance / HandleAdminResetInstance
// 的统一实现。两端只在 reinstallOpts 与 resolver 上有差异。
func commonHandleResetInstance(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver, opts reinstallOpts) {
	ctx := r.Context()
	log := Logger(ctx)
	tag := "[" + opts.logPrefix + "]"

	if r.Method != http.MethodPost {
		log.Warn(tag+" 非法方法", "method", r.Method, "actor", opts.actor)
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var (
		user     *model.User
		instance *model.Instance
		err      error
	)

	// 本地 HTML 渲染已移除，用户端 / 管控端统一走纯 JSON API。
	jsonAPI(w)

	if opts.isAdmin {
		if !requireAdmin(w, r) {
			return
		}
		instance, err = getAdminInstanceByIDOrInstanceID(r)
	} else {
		user = requireLogin(w, r)
		if user == nil {
			return
		}
		instance, err = getInstanceByID(&w, r, user)
	}
	if err != nil {
		log.Warn(tag+" 获取实例失败", "actor", opts.actor, "user_id", userIDOf(user), "error", err)
		writeError(w, r, instanceErrStatus(err), hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	if instance.IsDoctorNode {
		log.Warn(tag+" 拒绝龙虾医生节点", "actor", opts.actor, "user_id", userIDOf(user), "instance_id", instance.ID, "cvm_id", instance.InstanceId)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDoctorNodeNotAllowed))
		return
	}
	log.Info(tag+" 收到重装请求",
		"actor", opts.actor, "user_id", userIDOf(user),
		"instance_id", instance.ID, "cvm_id", instance.InstanceId,
		"name", instance.Name, "agent_type", instance.AgentType,
	)

	// final §6 C3/C4：仅支持重装的类型（OpenClaw）允许该操作，否则 403。
	// Hermes/ACE 重装会走到 openclaw 脚本链路失败，必须在入口拦截。
	if err := checkInstanceSupportsReinstall(ctx, instance); err != nil {
		log.Warn(tag+" 该类型不支持重装",
			"actor", opts.actor, "user_id", userIDOf(user),
			"instance_id", instance.ID, "cvm_id", instance.InstanceId, "agent_type", instance.AgentType, "error", err)
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 本地实例：不支持重装。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	// 状态准入：用户端仅 running 允许，管控端 running/stopped 允许。
	var statusErr error
	if opts.isAdmin {
		_, statusErr = requireActionAllowedForAdmin(ctx, instance, "reinstall", resolver)
	} else {
		_, statusErr = requireActionAllowedForUser(ctx, instance, "reinstall", resolver)
	}
	if statusErr != nil {
		log.Warn(tag+" 当前状态不允许重装",
			"actor", opts.actor, "user_id", userIDOf(user),
			"instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", statusErr)
		writeAgentGuardError(w, r, statusErr)
		return
	}

	if instance.InstanceId == "" {
		log.Warn(tag+" 实例无关联的 CVM",
			"actor", opts.actor, "user_id", userIDOf(user), "instance_id", instance.ID)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}

	// 乐观锁：写操作标记（含并发检查）+ 原子重置 Agent 状态。
	if err := setOperationWithAgentReset(model.DB(ctx), instance, model.OpReinstall); err != nil {
		log.Warn(tag+" 写入操作标记失败（并发冲突）",
			"actor", opts.actor, "user_id", userIDOf(user),
			"instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
		writeError(w, r, http.StatusConflict, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 查启用镜像 + 跨 agent_type 防御校验。
	enabledImage, err := prepareReinstallImage(ctx, instance)
	if err != nil {
		log.Error(tag+" 查询/校验启用镜像失败",
			"actor", opts.actor, "user_id", userIDOf(user),
			"instance_id", instance.ID, "cvm_id", instance.InstanceId, "agent_type", instance.AgentType, "error", err)
		clearOperation(model.DB(ctx), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 重置版本信息（重装后需重新拉取，不清空 agent_type）。
	// 与用户端历史顺序对齐：在 CVM 调用之前完成，让前端立即看到"版本未知"状态。
	if err := resetInstanceVersionInfo(ctx, instance); err != nil {
		log.Error(tag+" 重置版本信息失败",
			"actor", opts.actor, "user_id", userIDOf(user),
			"instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
		clearOperation(model.DB(ctx), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgResetVersionFailed))
		return
	}

	client, err := NewCVMClient(ctx)
	if err != nil {
		log.Error(tag+" 创建 CVM 客户端失败",
			"actor", opts.actor, "user_id", userIDOf(user),
			"instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
		clearOperation(model.DB(ctx), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateCVMClientFailed))
		return
	}

	req, err := buildReinstallRequest(ctx, instance, enabledImage)
	if err != nil {
		log.Error(tag+" 渲染 UserData 失败",
			"actor", opts.actor, "user_id", userIDOf(user),
			"instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
		clearOperation(model.DB(ctx), instance, model.OpStateFailed)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	log.Info(tag+" 调用 CVM ResetInstance",
		"actor", opts.actor, "user_id", userIDOf(user),
		"instance_id", instance.ID, "cvm_id", instance.InstanceId, "image_id", enabledImage.ImageId)
	if _, err := client.ResetInstance(req); err != nil {
		log.Error(tag+" CVM 重装失败",
			"actor", opts.actor, "user_id", userIDOf(user),
			"instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
		clearOperation(model.DB(ctx), instance, model.OpStateFailed)
		richErr := hcommon.I18nRichError(err, i18n.MsgReinstallInstanceFailed)
		writeError(w, r, http.StatusInternalServerError, richErr)
		// 与用户端对齐：CVM 失败时给实例所有者发"实例重装失败"错误通知。
		// 管控端操作也通知实例所有者（owner = instance.UserID），避免管理员
		// 重装失败用户毫不知情。
		// 异步 ctx 在 detached 后复制 i18n.Printer，保留请求者语言偏好。
		notifyCtx := hcommon.DetachContext(ctx)
		go createErrorNotification(instance.UserID, instance.ID, instance.Name,
			model.NotifyTypeInstanceReinstallFailed, i18n.T(notifyCtx, i18n.MsgReinstallInstanceFailedNotify), richErr,
			notifyCtx)
		return
	}
	clearAdjustmentFailure(ctx, instance.ID)
	log.Info(tag+" 重装请求已下发成功",
		"actor", opts.actor, "user_id", userIDOf(user),
		"instance_id", instance.ID, "cvm_id", instance.InstanceId, "image_id", enabledImage.ImageId)

	// 重装成功后直写镜像缓存（失败仅记录日志，不影响主流程）
	if err := model.DB(ctx).Model(instance).Update("img_id", enabledImage.ImageId).Error; err != nil {
		log.Warn(tag+" 直写 img_id 缓存失败", "instanceId", instance.InstanceId, "error", err)
	}

	// 重置业务状态（清空模型绑定 + ai_model_id=0），版本字段已在前置 resetInstanceVersionInfo
	// 单独清过，这里只需要 resetVersionInfo=false 即可。
	if err := resetReinstallBusinessState(ctx, instance, false); err != nil {
		log.Warn(tag+" 清空模型绑定失败（非阻塞）",
			"instance_pk", instance.ID, "cvm_id", instance.InstanceId, "error", err)
	}

	// 后置异步任务（memory 重置、技能/插件重新下发、approve device、SMH 恢复）。
	kickOffReinstallAsyncTasks(ctx, instance, tag)

	if opts.isAdmin {
		log.Info("[Admin] 重装", "admin", getAdminUser(r), "instanceId", instance.ID, "cvm_id", instance.InstanceId)
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// userIDOf 安全提取 user.ID，避免管控端 user=nil 时空指针。
func userIDOf(user *model.User) uint {
	if user == nil {
		return 0
	}
	return user.ID
}

// prepareReinstallImage 查询实例当前 agent_type 对应的启用镜像，并做跨类型防御校验。
// 失败时返回的 error 已经是面向用户的中文文案，调用方应直接 writeError 500。
func prepareReinstallImage(ctx context.Context, instance *model.Instance) (*model.AIImage, error) {
	enabledImage, err := model.GetEnabledImageByType(ctx, instance.AgentType)
	if err != nil {
		return nil, hcommon.I18nError(i18n.MsgQueryImageFailed)
	}
	if enabledImage == nil {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		return nil, hcommon.I18nError(i18n.MsgNoImageForType, typeName)
	}
	// 跨 agent_type 防御：堵住 GetEnabledImageByType 回退到空类型镜像导致
	// Hermes/ACE 实例拿到老 OpenClaw 镜像去重装的错乱路径。
	if err := verifyReinstallImageMatches(ctx, instance, enabledImage); err != nil {
		return nil, err
	}
	return enabledImage, nil
}

// buildReinstallRequest 构造 CVM ResetInstance 请求。
// 当站点配置了 SkillHub（即开启了 init.sh 注入），将渲染并合并 UserData 并开启
// AutomationService；否则保持空 UserData。两端复用这段渲染逻辑，确保管控端重装
// 与用户端重装在镜像之外不会再产生差异。
func buildReinstallRequest(ctx context.Context, instance *model.Instance, enabledImage *model.AIImage) (*cvm.ResetInstanceRequest, error) {
	req := cvm.NewResetInstanceRequest()
	req.InstanceId = common.StringPtr(instance.InstanceId)
	req.ImageId = common.StringPtr(enabledImage.ImageId)

	siteConfig := model.GetSiteConfig(ctx)
	var systemUserDataConfig *initUserDataConfig
	if siteConfig.SkillHub != "" {
		systemUserDataConfig = &initUserDataConfig{
			SkillHub:    siteConfig.SkillHub,
			RuntimeUser: getEffectiveRuntimeUser(instance.RuntimeUser),
			AgentType:   instance.AgentType,
		}
	}
	mergedUserData, err := buildUserData(ctx, systemUserDataConfig, instance.UserData)
	if err != nil {
		return nil, err
	}
	if mergedUserData != "" {
		req.EnhancedService = &cvm.EnhancedService{
			AutomationService: &cvm.RunAutomationServiceEnabled{
				Enabled: common.BoolPtr(true),
			},
		}
		req.UserData = common.StringPtr(mergedUserData)
	}
	return req, nil
}

// resetReinstallBusinessState 在 CVM 重装请求下发成功后，于单事务内重置实例业务状态。
//
// 总会执行：
//   - instance.ai_model_id = 0
//   - HardDelete instance_models（物理删除避免软删除残留占用唯一索引导致重装
//     后绑定模型报 Duplicate entry）
//
// resetVersionInfo=true 时额外清空 cls_agent_status / agent_version /
// plugin_versions_json / version_fetched_at 四个字段（管控端使用：用户端会在
// CVM 调用之前先调 resetInstanceVersionInfo 单独处理，因此用户端传 false）。
//
// 注意：不清空 agent_type（agent_type 是实例固有属性，重装本身不会改变它；清空
// 会导致后续 installSkillsAsync / batch_install_skills 脚本分派失去 agent_type 形参）。
func resetReinstallBusinessState(ctx context.Context, instance *model.Instance, resetVersionInfo bool) error {
	return model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{"ai_model_id": 0}
		if resetVersionInfo {
			updates["cls_agent_status"] = 0
			updates["agent_version"] = ""
			updates["plugin_versions_json"] = ""
			updates["version_fetched_at"] = nil
		}
		if err := tx.Model(instance).Updates(updates).Error; err != nil {
			return err
		}
		return model.HardDeleteInstanceModels(tx, instance.ID)
	})
}

// clearEnterpriseDistributionRecords 清空管控端企业技能 / 企业插件 / 企业 MCP 的 下发（安装）记录：
//   - SkillDistributionRecord —— 企业技能下发记录
//   - PluginDistributionRecord —— 企业插件下发记录
//   - McpInstallation —— 企业 MCP 安装记录
//
// 若失败，只记录错误日志不阻塞后置流程，避免影响重装后实例的管控端下发页恢复。
func clearEnterpriseDistributionRecords(ctx context.Context, instance *model.Instance, tag string) {
	log := Logger(ctx)

	// 清空管控端企业技能下发记录，支持再次下发
	if err := model.DB(ctx).Where("instance_id = ?", instance.ID).
		Delete(&model.SkillDistributionRecord{}).Error; err != nil {
		log.Error(tag+" 清空企业技能下发记录失败",
			"instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
	} else {
		log.Info(tag+" 已清空企业技能下发记录", "instance_id", instance.ID, "cvm_id", instance.InstanceId)
	}

	// 清空管控端企业插件下发记录，支持再次下发
	if err := model.DB(ctx).Where("instance_id = ?", instance.ID).
		Delete(&model.PluginDistributionRecord{}).Error; err != nil {
		log.Error(tag+" 清空企业插件下发记录失败",
			"instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
	} else {
		log.Info(tag+" 已清空企业插件下发记录", "instance_id", instance.ID, "cvm_id", instance.InstanceId)
	}

	// 清空管控端企业 MCP 安装记录，支持再次下发
	if err := model.DB(ctx).Where("instance_id = ?", instance.ID).
		Delete(&model.McpInstallation{}).Error; err != nil {
		log.Error(tag+" 清空企业 MCP 安装记录失败",
			"instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
	} else {
		log.Info(tag+" 已清空企业 MCP 安装记录", "instance_id", instance.ID, "cvm_id", instance.InstanceId)
	}
}

// kickOffReinstallAsyncTasks 在 CVM ResetInstance 下发成功后启动重装后置动作：
//   - 同步：重置 memory 插件状态（保留 Pro 绑定）、清空旧 SkillInstallation /
//     PluginInstallation / McpInstallation、清空管控端企业技能 / 企业插件下发记录
//     （SkillDistributionRecord / PluginDistributionRecord），使重装后实例在管控端
//     技能 / 插件 / MCP 下发页恢复为 uninstalled，支持再次下发
//   - 异步：重新下发技能 / 插件安装任务、approve device、SMH 个人空间环境恢复
//
// 入参 ctx 传入 r.Context() 即可，函数对所有 goroutine 自行 DetachContext，
// 避免 handler return 后 ctx 被 cancel 导致后台任务异常退出。
func kickOffReinstallAsyncTasks(ctx context.Context, instance *model.Instance, tag string) {
	log := Logger(ctx)
	log.Info(tag+" 开始重置 memory 插件状态", "instance_id", instance.ID, "cvm_id", instance.InstanceId)
	resetMemoryPluginForReinstall(ctx, instance.InstanceId)

	// 清空管控端企业技能 / 企业插件 / 企业 MCP 下发记录，使重装后实例在管控端
	// 下发页恢复为 uninstalled，支持再次下发。
	clearEnterpriseDistributionRecords(ctx, instance, tag)

	model.DB(ctx).Where("instance_id = ?", instance.ID).Delete(&model.SkillInstallation{})
	createSkillInstallTasks(ctx, instance.ID, instance.RoleID, instance.GroupID)
	log.Info(tag+" 开始异步安装技能", "instance_id", instance.ID, "cvm_id", instance.InstanceId)
	go installSkillsAsync(hcommon.DetachContext(ctx), instance.ID, instance.InstanceId, instance.AgentType, waitModeReinstall)

	model.DB(ctx).Where("instance_id = ?", instance.ID).Delete(&model.PluginInstallation{})
	createPluginInstallTasks(ctx, instance.ID, instance.RoleID)
	log.Info(tag+" 开始异步安装插件", "instance_id", instance.ID, "cvm_id", instance.InstanceId)
	go installPluginsAsync(hcommon.DetachContext(ctx), instance.ID, instance.InstanceId, waitModeReinstall)

	log.Info(tag+" 开始异步 approve device", "instance_id", instance.ID, "cvm_id", instance.InstanceId)
	go approveDeviceAsync(hcommon.DetachContext(ctx), instance.ID, instance.InstanceId, instance.RuntimeUser)

	log.Info(tag+" 开始异步恢复 SMH 个人空间环境", "instance_id", instance.ID, "cvm_id", instance.InstanceId)
	go syncSMHEnvWhenReadyFn(hcommon.DetachContext(ctx), *instance)
}
