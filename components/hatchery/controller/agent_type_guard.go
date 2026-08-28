package controller

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// checkAgentTypeValid 校验智能体类型有效性
func checkAgentTypeValid(ctx context.Context, agentType string) *hcommon.RichError {
	if agentType == "" {
		return nil // 允许空值（兼容）
	}
	if !model.IsValidAgentType(ctx, agentType) {
		return hcommon.I18nError(i18n.MsgInvalidAgentType, agentType)
	}
	return nil
}

// rejectLocalInstance 为 CVM 专属操作接口过滤本地实例。
// 调用者拿到 *hcommon.RichError != nil 时需返 400。
func rejectLocalInstance(instance *model.Instance) *hcommon.RichError {
	if instance == nil {
		return nil
	}
	if instance.Source == model.InstanceSourceLocal {
		return hcommon.I18nError(i18n.MsgLocalInstanceUnsupportedOp)
	}
	return nil
}

// rejectLocalOrWrite 是 rejectLocalInstance 的 writer 包装，返 true 表示已写入 400 响应、handler 应立即 return。
// 用法：
//
//	if rejectLocalOrWrite(w, r, instance) { return }
func rejectLocalOrWrite(w http.ResponseWriter, r *http.Request, instance *model.Instance) bool {
	if rerr := rejectLocalInstance(instance); rerr != nil {
		writeError(w, r, http.StatusBadRequest, rerr)
		return true
	}
	return false
}

// checkAgentVersionValid 校验版本号格式
func checkAgentVersionValid(version string) *hcommon.RichError {
	if !model.IsValidAgentVersion(version) {
		return hcommon.I18nError(i18n.MsgAgentVersionFormatInvalid, version)
	}
	return nil
}

// checkInstanceSupportsDetailConfig 校验实例是否支持详细配置（任意一项：模型/通道/技能/插件）
func checkInstanceSupportsDetailConfig(ctx context.Context, instance *model.Instance) *hcommon.RichError {
	if instance == nil {
		return nil
	}
	if !model.AgentTypeSupportsDetailConfig(ctx, instance.AgentType) {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		slog.WarnContext(ctx, "[Guard] 实例类型不支持详细配置",
			"instance_id", instance.ID,
			"agent_type", instance.AgentType)
		return hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportDetailConfigWithDetail, typeName)
	}
	return nil
}

// checkInstanceSupportsPlugin 校验实例是否支持插件安装
func checkInstanceSupportsPlugin(ctx context.Context, instance *model.Instance) *hcommon.RichError {
	if instance == nil {
		return nil
	}
	if !model.AgentTypeSupportsPlugin(ctx, instance.AgentType) {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		slog.WarnContext(ctx, "[Guard] 实例类型不支持插件",
			"instance_id", instance.ID,
			"agent_type", instance.AgentType)
		return hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportPluginWithDetail, typeName)
	}
	return nil
}

// checkInstanceSupportsSkill 校验实例是否支持技能安装
func checkInstanceSupportsSkill(ctx context.Context, instance *model.Instance) *hcommon.RichError {
	if instance == nil {
		return nil
	}
	if !model.AgentTypeSupportsSkill(ctx, instance.AgentType) {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		slog.WarnContext(ctx, "[Guard] 实例类型不支持技能",
			"instance_id", instance.ID,
			"agent_type", instance.AgentType)
		return hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportSkillWithDetail, typeName)
	}
	return nil
}

// checkInstanceSupportsModel 校验实例是否支持模型配置
func checkInstanceSupportsModel(ctx context.Context, instance *model.Instance) *hcommon.RichError {
	if instance == nil {
		return nil
	}
	if !model.AgentTypeSupportsModel(ctx, instance.AgentType) {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		slog.WarnContext(ctx, "[Guard] 实例类型不支持模型配置",
			"instance_id", instance.ID,
			"agent_type", instance.AgentType)
		return hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportModelConfigWithDetail, typeName)
	}
	return nil
}

// checkInstanceSupportsChannel 校验实例是否支持通道配置
func checkInstanceSupportsChannel(ctx context.Context, instance *model.Instance) *hcommon.RichError {
	if instance == nil {
		return nil
	}
	if !model.AgentTypeSupportsChannel(ctx, instance.AgentType) {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		slog.WarnContext(ctx, "[Guard] 实例类型不支持通道配置",
			"instance_id", instance.ID,
			"agent_type", instance.AgentType)
		return hcommon.I18nError(i18n.MsgChannelNotSupportedWithDetail, typeName)
	}
	return nil
}

// checkInstanceSupportsChatbot 校验实例是否支持 Chatbot
func checkInstanceSupportsChatbot(ctx context.Context, instance *model.Instance) *hcommon.RichError {
	if instance == nil {
		return nil
	}
	if !model.AgentTypeSupportsChatbot(ctx, instance.AgentType) {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		slog.WarnContext(ctx, "[Guard] 实例类型不支持 Chatbot",
			"instance_id", instance.ID,
			"agent_type", instance.AgentType)
		return hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportChatbotWithDetail, typeName)
	}
	return nil
}

// checkInstanceSupportsReinstall 校验实例是否允许 CVM 重装（final §6 C3/C4）
// OpenClaw / Hermes / ACE 均支持，各自按 detect_install 分派探测脚本。
func checkInstanceSupportsReinstall(ctx context.Context, instance *model.Instance) *hcommon.RichError {
	if instance == nil {
		return nil
	}
	if !model.AgentTypeSupportsReinstall(ctx, instance.AgentType) {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		slog.WarnContext(ctx, "[Guard] 实例类型不支持重装",
			"instance_id", instance.ID,
			"agent_type", instance.AgentType)
		return hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportReinstallWithDetail, typeName)
	}
	return nil
}

// checkInstanceSupportsUpgrade 校验实例是否支持一键升级（备份→SMH上传→重装→恢复）。
// 当前 OpenClaw / Hermes 通过 ResolveScript 分派各自的 backup_pre_reinstall / restore_post_reinstall
// 脚本链路；其它类型在 HandleUpgrade / HandleUpgradeRetry 入口拦截。
func checkInstanceSupportsUpgrade(ctx context.Context, instance *model.Instance) *hcommon.RichError {
	if instance == nil {
		return nil
	}
	if !model.AgentTypeSupportsUpgrade(ctx, instance.AgentType) {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		slog.WarnContext(ctx, "[Guard] 实例类型不支持一键升级",
			"instance_id", instance.ID,
			"agent_type", instance.AgentType)
		return hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportUpgradeWithDetail, typeName)
	}
	return nil
}

// checkInstanceSupportsMemory 校验实例是否支持记忆功能（Memory TDAI）。
// OpenClaw / Hermes 支持，ACE 不支持。
// 用于 HandleOpenClawMemoryTDAIStatus 等入口拦截。
func checkInstanceSupportsMemory(ctx context.Context, instance *model.Instance) *hcommon.RichError {
	if instance == nil {
		return nil
	}
	if !model.AgentTypeSupportsMemory(ctx, instance.AgentType) {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		slog.WarnContext(ctx, "[Guard] 实例类型不支持记忆功能",
			"instance_id", instance.ID,
			"agent_type", instance.AgentType)
		return hcommon.I18nError(i18n.MsgMemoryProAgentTypeNoMemoryFmt, typeName)
	}
	return nil
}

// checkInstanceSupportsBrowserVNC 校验实例是否支持云端浏览器（Browser VNC）。
// 仅 OpenClaw 支持；Hermes/ACE 镜像没有 noVNC/Chrome/openclaw 浏览器自动化栈。
// 用于 HandleBrowserVNCInstall/Check/Access/Takeover 等入口拦截。
// 高频轮询的 HandleBrowserStatus 不走本 guard，而是返回 200 + unsupported 字段
// 避免前端每 3 秒报 403。
func checkInstanceSupportsBrowserVNC(ctx context.Context, instance *model.Instance) *hcommon.RichError {
	if instance == nil {
		return nil
	}
	if !model.AgentTypeSupportsBrowserVNC(ctx, instance.AgentType) {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		slog.WarnContext(ctx, "[Guard] 实例类型不支持云端浏览器",
			"instance_id", instance.ID,
			"agent_type", instance.AgentType)
		return hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportBrowserVNCWithDetail, typeName)
	}
	return nil
}

// checkInstanceSupportsApprove 校验实例是否支持 approve / approve_device 流程。
// 仅 OpenClaw 支持；Hermes/ACE 走自己的 OAuth / Server API，不需要 approve CLI 回调。
// 用于 HandleApprove 入口拦截，防止 ACE/Hermes 实例点击飞书授权时执行不存在的
// approve.sh 导致 500 失败。
func checkInstanceSupportsApprove(ctx context.Context, instance *model.Instance) *hcommon.RichError {
	if instance == nil {
		return nil
	}
	if !model.AgentTypeSupportsApprove(ctx, instance.AgentType) {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		slog.WarnContext(ctx, "[Guard] 实例类型不支持 approve 流程",
			"instance_id", instance.ID,
			"agent_type", instance.AgentType)
		return hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportApproveWithDetail, typeName)
	}
	return nil
}

// verifyReinstallImageMatches 校验即将用于重装的镜像与实例 agent_type 是否匹配。
//
// 背景：`GetEnabledImageByType` 查不到对应类型时会回退到空类型镜像（存量兼容），
// 这对 OpenClaw 实例是安全的（大多数老启用镜像 agent_type 就是空），但 Hermes/ACE
// 实例拿到空类型镜像（往往是旧 openclaw 镜像）去重装会造成类型错乱。
//
// 三期新增的 Hermes/ACE 重装入口必须堵死这条 fallback 路径。本函数由两个重装
// handler（用户端 HandleResetInstance / 管控端 HandleAdminResetInstance）共用。
//
// 规则：
//   - 类型严格相等 → 通过
//   - OpenClaw 实例 + 空类型镜像 → 通过（唯一例外，保留存量数据兼容）
//   - 其他所有组合 → 拒绝
func verifyReinstallImageMatches(ctx context.Context, instance *model.Instance, img *model.AIImage) *hcommon.RichError {
	if instance == nil || img == nil {
		return nil
	}
	// 兼容存量数据：实例 agent_type 为空时视为 openclaw（与 model 层
	// GetEnabledImageByType / GetEnabledImagesMap / site_config 口径一致）。
	instAgentType := model.NormalizeAgentType(instance.AgentType)
	if img.AgentType == instAgentType {
		return nil
	}
	// 唯一兼容例外：OpenClaw 实例 + 空类型（legacy）镜像。
	if instAgentType == model.AgentTypeOpenClaw && img.AgentType == "" {
		return nil
	}
	instTypeName := model.GetAgentTypeDisplayName(ctx, instAgentType)
	var imgTypeName string
	if img.AgentType == "" {
		imgTypeName = "未分类"
	} else {
		imgTypeName = model.GetAgentTypeDisplayName(ctx, img.AgentType)
	}
	slog.WarnContext(ctx, "[Guard] 重装镜像类型与实例类型不匹配",
		"instance_id", instance.ID,
		"instance_agent_type", instance.AgentType,
		"image_id", img.ImageId,
		"image_agent_type", img.AgentType)
	return hcommon.I18nError(i18n.MsgReinstallImageTypeMismatchWithDetail, imgTypeName, instTypeName)
}

// mcpMinAgentVersion 用户端 MCP 功能要求的最低 openclaw 版本（2026.3.28）
const mcpMinAgentVersion = "2026.3.28"

// checkInstanceSupportsMcp 校验实例是否支持用户端 MCP 操作。
// 条件：agent_type=openclaw 且 agent_version >= 2026.3.28。
func checkInstanceSupportsMcp(ctx context.Context, instance *model.Instance) *hcommon.RichError {
	if instance == nil {
		return nil
	}
	// 1. 仅 openclaw 类型支持 MCP
	if strings.ToLower(instance.AgentType) != model.AgentTypeOpenClaw {
		typeName := model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		slog.WarnContext(ctx, "[Guard] 实例类型不支持 MCP",
			"instance_id", instance.ID,
			"agent_type", instance.AgentType)
		return hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportMcpWithDetail, typeName)
	}
	// 2. 版本必须 >= 2026.3.28
	if instance.AgentVersion == "" || compareAgentVersion(instance.AgentVersion, mcpMinAgentVersion) < 0 {
		slog.WarnContext(ctx, "[Guard] 实例版本过低，不支持 MCP",
			"instance_id", instance.ID,
			"agent_version", instance.AgentVersion,
			"min_version", mcpMinAgentVersion)
		return hcommon.I18nError(i18n.MsgMcpVersionTooLow, instance.AgentVersion, mcpMinAgentVersion)
	}
	return nil
}
