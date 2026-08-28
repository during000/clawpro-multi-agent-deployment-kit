package model

import (
	"context"
	"fmt"
	"regexp"

	hcommon "hatchery/common"
	"hatchery/i18n"
)

// ========== 智能体类型常量 ==========

const (
	AgentTypeOpenClaw     = "openclaw"
	AgentTypeHermes       = "hermes"
	AgentTypeLightclawACE = "lightclawace"
	// DeepSeekTUI / OpenCode 是仅提供 Web 终端的轻量预设 Agent 类型：
	//   - 没有 hatchery 侧脚本支持，所有 Supports* 均为 false；
	//   - 默认对用户端禁用，需管理员通过 /admin/agent-types/enabled 显式启用；
	//   - 镜像版本通过发布平台经 /admin/images/history/publish 写入。
	AgentTypeDeepSeekTUI = "deepseektui"
	AgentTypeOpenCode    = "opencode"
)

// AgentTypeDisplayNames 镜像类型的展示名称映射（用于配置总览等场景）
var AgentTypeDisplayNames = map[string]string{
	AgentTypeOpenClaw:     "OpenClaw",
	AgentTypeHermes:       "Hermes Agent",
	AgentTypeLightclawACE: "LightClaw ACE",
	AgentTypeDeepSeekTUI:  "DeepSeek TUI",
	AgentTypeOpenCode:     "OpenCode",
}

// AgentType 智能体类型配置（硬编码）
type AgentType struct {
	Code               string `json:"code"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	IsBuiltin          bool   `json:"is_builtin"`
	CompatibleWith     string `json:"compatible_with,omitempty"`
	SupportsRole       bool   `json:"supports_role"`
	SupportsModel      bool   `json:"supports_model"`
	SupportsChannel    bool   `json:"supports_channel"`
	SupportsSkill      bool   `json:"supports_skill"`
	SupportsPlugin     bool   `json:"supports_plugin"`
	SupportsChatbot    bool   `json:"supports_chatbot"`
	SupportsSMH        bool   `json:"supports_smh"`
	SupportsMemory     bool   `json:"supports_memory"`
	SupportsReinstall  bool   `json:"supports_reinstall"`   // CVM 重装：OpenClaw / Hermes / ACE 均支持，按 detect_install 分派探测脚本
	SupportsUpgrade    bool   `json:"supports_upgrade"`     // 一键升级（备份→SMH上传→重装→恢复）：OpenClaw / Hermes 支持，按 ResolveScript 分派备份/恢复脚本
	SupportsBrowserVNC bool   `json:"supports_browser_vnc"` // 云端浏览器：依赖 openclaw 内建浏览器自动化，仅 OpenClaw 支持
	SupportsApprove    bool   `json:"supports_approve"`     // 设备/CLI 授权回调（approve.sh / approve_device.sh）：仅 OpenClaw 需要
	SupportsMultiAgent bool   `json:"supports_multi_agent"` // 查询实例内部是否配置多个 agent。当前仅 OpenClaw 支持，后续 runtime 可按能力矩阵放开
	// SupportsDefaultModelInjection：实例创建后是否允许系统自动注入站点默认模型。
	// 与 SupportsModel 独立：SupportsModel=true 表示允许用户手动走 /openclaw/set-model；
	// SupportsDefaultModelInjection=true 表示额外允许 CreateInstance 后起 goroutine
	// 主动把 config.DefaultModelID 绑定到该实例。
	// 该开关用于区分"用户可手动配置模型"与"系统可在创建后自动下发默认模型"。
	// 某些 agent 运行时对 provider/baseUrl 的契约与站点默认模型注入链路不完全一致，
	// 若直接在创建后无条件推送，可能出现首次 set_model 未生效但 ai_model_id 已写入
	// DB 的不一致状态，因此单独保留此能力位做精细控制。
	SupportsDefaultModelInjection bool `json:"supports_default_model_injection"`
	// SupportsAPIGateway：实例是否允许走云 API 网关进行 WebUI 域名化接入。
	// 仅 OpenClaw 打开：Lightclaw/Hermes 的 WebUI 形态（直接公网端口 + token / Basic Auth）
	// 与网关的 AgentID 回源契约不匹配，即便开启 site_config.api_gateway_config 也不生效。
	SupportsAPIGateway bool `json:"supports_api_gateway"`
	// NeedsRuntimeUserCorrection：官方镜像出厂 runtime_user 是否需 Go 侧强制校正（仅 OpenClaw 出厂 root）。
	NeedsRuntimeUserCorrection bool `json:"needs_runtime_user_correction"`
	SortOrder                  int  `json:"sort_order"`
}

// agentTypesMap 硬编码的智能体类型配置
var agentTypesMap = map[string]*AgentType{
	AgentTypeOpenClaw: {
		Code:                          AgentTypeOpenClaw,
		Name:                          "OpenClaw",
		Description:                   "功能最完整的智能体类型，支持模型/通道/技能/插件/Chatbot/Memory 等全量能力",
		IsBuiltin:                     true,
		SupportsRole:                  true,
		SupportsModel:                 true,
		SupportsChannel:               true,
		SupportsSkill:                 true,
		SupportsPlugin:                true,
		SupportsChatbot:               true,
		SupportsSMH:                   true,
		SupportsMemory:                true,
		SupportsReinstall:             true,
		SupportsUpgrade:               true,
		SupportsBrowserVNC:            true,
		SupportsApprove:               true,
		SupportsMultiAgent:            true,
		SupportsDefaultModelInjection: true,
		SupportsAPIGateway:            true,
		NeedsRuntimeUserCorrection:    true, // OpenClaw 官方镜像出厂 root，需校正
		SortOrder:                     1,
	},
	AgentTypeHermes: {
		Code:               AgentTypeHermes,
		Name:               "Hermes",
		Description:        "轻量智能体：支持 Role/Model/Channel/Skill/SMH/Memory/Reinstall；不支持 Plugin/Chatbot/BrowserVNC/Approve",
		IsBuiltin:          true,
		SupportsRole:       true,
		SupportsModel:      true,
		SupportsChannel:    true,
		SupportsSkill:      true,
		SupportsPlugin:     false,
		SupportsChatbot:    false,
		SupportsSMH:        true,  // final §2 / §3.2：三端 SMH 放开
		SupportsMemory:     true,  // Hermes 记忆功能已适配（Free 模式）
		SupportsReinstall:  true,  // 三期放开：set_model_hermes.sh + Hermes 镜像类型闭环后，允许重装
		SupportsUpgrade:    true,  // Hermes 升级链路：backup_pre_reinstall_hermes.sh + restore_post_reinstall_hermes.sh，与 OpenClaw 共用 Go 层升级骨架（按 ResolveScript 分派备份/恢复脚本）
		SupportsBrowserVNC: false, // Hermes 镜像无 noVNC/Chrome/openclaw 浏览器自动化栈
		SupportsApprove:    false, // Hermes 无 approve CLI 子命令
		SupportsMultiAgent: false, // 当前 Hermes 未适配实例内多 agent 配置探测
		// 三期放开：默认模型自动注入失败时由 rollbackDefaultModelIfIntact 精确回滚 DB，
		// 避免"虚绑"留下 ai_model_id != 实例实际配置 的不一致状态。
		SupportsDefaultModelInjection: true,
		SupportsAPIGateway:            false, // Hermes WebUI 无 AgentToken 契约，不走云 API 网关
		SortOrder:                     2,
	},
	AgentTypeLightclawACE: {
		Code:               AgentTypeLightclawACE,
		Name:               "LightclawACE",
		Description:        "轻量智能体：支持 Role/Model/Channel/Skill/Chatbot/SMH/Reinstall/DefaultModel；不支持 Plugin/Memory/BrowserVNC/Approve",
		IsBuiltin:          true,
		SupportsRole:       true,
		SupportsModel:      true,
		SupportsChannel:    true,
		SupportsSkill:      true,
		SupportsPlugin:     false,
		SupportsChatbot:    true, // final §3.2：ACE Chatbot 放开（注：handler 未实现前用户无入口触发）
		SupportsSMH:        true,
		SupportsMemory:     false,
		SupportsReinstall:  true,  // 三期放开：set_model_ace.sh + LightclawACE 镜像类型闭环后，允许重装
		SupportsUpgrade:    false, // ACE 无 backup_pre_reinstall / restore_post_reinstall 脚本链路
		SupportsBrowserVNC: false, // ACE 镜像无 noVNC/Chrome 栈
		SupportsApprove:    false, // ACE 无 approve CLI 子命令
		SupportsMultiAgent: false, // 当前 ACE 未适配实例内多 agent 配置探测
		// 三期放开：默认模型自动注入失败时由 rollbackDefaultModelIfIntact 精确回滚 DB。
		SupportsDefaultModelInjection: true,
		SupportsAPIGateway:            false, // ACE WebUI 依赖 password Basic Auth，不走云 API 网关
		SortOrder:                     3,
	},
	// DeepSeek TUI：仅提供 Web 终端访问，不适配任何面板 / 流程能力。
	// 与「自定义内核」内核的预设型对外契约一致——所有面板按钮/对话视图/角色/技能/插件/通道/记忆等均关闭。
	AgentTypeDeepSeekTUI: {
		Code:                          AgentTypeDeepSeekTUI,
		Name:                          "DeepSeek TUI",
		Description:                   "预设 DeepSeek TUI：仅支持 Web 终端访问，不适配模型/通道/技能/插件/Chatbot/Memory/重装/升级等流程",
		IsBuiltin:                     true,
		SupportsRole:                  false,
		SupportsModel:                 false,
		SupportsChannel:               false,
		SupportsSkill:                 false,
		SupportsPlugin:                false,
		SupportsChatbot:               false,
		SupportsSMH:                   false,
		SupportsMemory:                false,
		SupportsReinstall:             false,
		SupportsUpgrade:               false,
		SupportsBrowserVNC:            false,
		SupportsApprove:               false,
		SupportsMultiAgent:            false,
		SupportsDefaultModelInjection: false,
		SupportsAPIGateway:            false,
		SortOrder:                     4,
	},
	// OpenCode：与 DeepSeekTUI 同类——仅 Web 终端，能力位全关闭。
	AgentTypeOpenCode: {
		Code:                          AgentTypeOpenCode,
		Name:                          "OpenCode",
		Description:                   "预设 OpenCode：仅支持 Web 终端访问，不适配模型/通道/技能/插件/Chatbot/Memory/重装/升级等流程",
		IsBuiltin:                     true,
		SupportsRole:                  false,
		SupportsModel:                 false,
		SupportsChannel:               false,
		SupportsSkill:                 false,
		SupportsPlugin:                false,
		SupportsChatbot:               false,
		SupportsSMH:                   false,
		SupportsMemory:                false,
		SupportsReinstall:             false,
		SupportsUpgrade:               false,
		SupportsBrowserVNC:            false,
		SupportsApprove:               false,
		SupportsMultiAgent:            false,
		SupportsDefaultModelInjection: false,
		SupportsAPIGateway:            false,
		SortOrder:                     5,
	},
}

// agentTypesList 按排序顺序排列的智能体类型列表
var agentTypesList = []*AgentType{
	agentTypesMap[AgentTypeOpenClaw],
	agentTypesMap[AgentTypeHermes],
	agentTypesMap[AgentTypeLightclawACE],
	agentTypesMap[AgentTypeDeepSeekTUI],
	agentTypesMap[AgentTypeOpenCode],
}

var minimalAgentType = AgentType{
	Description:                   "自定义智能体类型，不兼容内置类型，仅支持最小操作集",
	IsBuiltin:                     false,
	SupportsRole:                  false,
	SupportsModel:                 false,
	SupportsChannel:               false,
	SupportsSkill:                 false,
	SupportsPlugin:                false,
	SupportsChatbot:               false,
	SupportsSMH:                   false,
	SupportsMemory:                false,
	SupportsReinstall:             false,
	SupportsUpgrade:               false,
	SupportsBrowserVNC:            false,
	SupportsApprove:               false,
	SupportsMultiAgent:            false,
	SupportsDefaultModelInjection: false,
	SupportsAPIGateway:            false,
	SortOrder:                     1000,
}

// 版本号校验正则
var agentVersionRegex = regexp.MustCompile(`^[a-zA-Z0-9][\w.\-]{0,62}[a-zA-Z0-9]$|^[a-zA-Z0-9]$|^$`)

// OpenClaw 版本号格式：YYYY.M.D 或 YYYY.MM.DD
var openclawVersionRegex = regexp.MustCompile(`^\d{4}\.\d{1,2}\.\d{1,2}$`)

// Hermes/LightclawACE 版本号格式：semver（X.Y.Z）
var semverRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// ========== 查询函数 ==========

// IsBuiltinAgentType 判断是否为内置智能体类型。
func IsBuiltinAgentType(agentType string) bool {
	_, ok := agentTypesMap[agentType]
	return ok
}

// NormalizeAgentType 将存量空 agent_type 归一化为 openclaw。
func NormalizeAgentType(agentType string) string {
	if agentType == "" {
		return AgentTypeOpenClaw
	}
	return agentType
}

// GetAgentTypeByCode 根据代码获取智能体类型。
// 空字符串视为 openclaw（兼容存量数据）。
func GetAgentTypeByCode(ctx context.Context, code string) *AgentType {
	code = NormalizeAgentType(code)
	if t, ok := agentTypesMap[code]; ok {
		return t
	}
	custom, err := GetCustomAgentTypeByName(ctx, code)
	if err != nil || custom == nil {
		return nil
	}
	return buildAgentTypeFromCustom(custom)
}

// GetAllAgentTypes 获取所有智能体类型（内置类型在前，自定义类型按创建顺序）。
func GetAllAgentTypes(ctx context.Context) []*AgentType {
	out := make([]*AgentType, 0, len(agentTypesList))
	out = append(out, agentTypesList...)
	customTypes, err := ListCustomAgentTypes(ctx)
	if err != nil {
		return out
	}
	for _, custom := range customTypes {
		out = append(out, buildAgentTypeFromCustom(&custom))
	}
	return out
}

// buildAgentTypeFromCustom 从 CustomAgentType 记录构造 AgentType，不再查 DB。
func buildAgentTypeFromCustom(custom *CustomAgentType) *AgentType {
	if custom.CompatibleWith != "" {
		base := agentTypesMap[custom.CompatibleWith]
		if base == nil {
			copied := minimalAgentType
			copied.Code = custom.Name
			copied.Name = custom.Name
			copied.CompatibleWith = custom.CompatibleWith
			copied.IsBuiltin = false
			copied.SortOrder = 1000 + int(custom.ID)
			return &copied
		}
		copied := *base
		copied.Code = custom.Name
		copied.Name = custom.Name
		copied.Description = fmt.Sprintf("自定义智能体类型，兼容 %s", base.Name)
		copied.IsBuiltin = false
		copied.CompatibleWith = custom.CompatibleWith
		copied.SortOrder = 1000 + int(custom.ID)
		return &copied
	}
	copied := minimalAgentType
	copied.Code = custom.Name
	copied.Name = custom.Name
	copied.CompatibleWith = ""
	copied.IsBuiltin = false
	copied.SortOrder = 1000 + int(custom.ID)
	return &copied
}

// GetAllAgentTypesMap 获取所有智能体类型（Map 形式）
func GetAllAgentTypesMap(ctx context.Context) map[string]*AgentType {
	types := GetAllAgentTypes(ctx)
	out := make(map[string]*AgentType, len(types))
	for _, t := range types {
		out[t.Code] = t
	}
	return out
}

// IsValidAgentType 校验智能体类型是否有效
func IsValidAgentType(ctx context.Context, agentType string) bool {
	if agentType == "" {
		return false
	}
	return GetAgentTypeByCode(ctx, agentType) != nil
}

// GetAgentRuntimeType 返回脚本分派所使用的内置运行时类型。
// 自定义类型无兼容目标时返回空字符串，表示不支持 agent-specific 脚本。
func GetAgentRuntimeType(ctx context.Context, agentType string) string {
	agentType = NormalizeAgentType(agentType)
	if IsBuiltinAgentType(agentType) {
		return agentType
	}
	custom, err := GetCustomAgentTypeByName(ctx, agentType)
	if err != nil || custom == nil || custom.CompatibleWith == "" {
		return ""
	}
	if !IsBuiltinAgentType(custom.CompatibleWith) {
		return ""
	}
	return custom.CompatibleWith
}

// IsValidAgentVersion 校验版本号格式
func IsValidAgentVersion(v string) bool {
	return agentVersionRegex.MatchString(v)
}

// ========== 功能支持检查函数 ==========

// AgentTypeSupportsRole 检查类型是否支持角色配置
func AgentTypeSupportsRole(ctx context.Context, code string) bool {
	t := GetAgentTypeByCode(ctx, code)
	if t == nil {
		return false
	}
	return t.SupportsRole
}

// AgentTypeDetailConfigFlags 详细配置支持情况
type AgentTypeDetailConfigFlags struct {
	SupportsModel   bool `json:"supports_model"`
	SupportsChannel bool `json:"supports_channel"`
	SupportsSkill   bool `json:"supports_skill"`
	SupportsPlugin  bool `json:"supports_plugin"`
}

// GetAgentTypeDetailConfigFlags 获取类型的详细配置支持情况
func GetAgentTypeDetailConfigFlags(ctx context.Context, code string) *AgentTypeDetailConfigFlags {
	t := GetAgentTypeByCode(ctx, code)
	if t == nil {
		return nil
	}
	return &AgentTypeDetailConfigFlags{
		SupportsModel:   t.SupportsModel,
		SupportsChannel: t.SupportsChannel,
		SupportsSkill:   t.SupportsSkill,
		SupportsPlugin:  t.SupportsPlugin,
	}
}

// AgentTypeSupportsDetailConfig 检查类型是否支持详细配置（模型/通道/技能/插件中任意一项）
func AgentTypeSupportsDetailConfig(ctx context.Context, code string) bool {
	flags := GetAgentTypeDetailConfigFlags(ctx, code)
	if flags == nil {
		return false
	}
	return flags.SupportsModel || flags.SupportsChannel || flags.SupportsSkill || flags.SupportsPlugin
}

// AgentTypeSupportsChatbot 检查类型是否支持 Chatbot
func AgentTypeSupportsChatbot(ctx context.Context, code string) bool {
	t := GetAgentTypeByCode(ctx, code)
	if t == nil {
		return false
	}
	return t.SupportsChatbot
}

// AgentTypeSupportsChannel 检查类型是否支持通道配置
func AgentTypeSupportsChannel(ctx context.Context, code string) bool {
	t := GetAgentTypeByCode(ctx, code)
	if t == nil {
		return false
	}
	return t.SupportsChannel
}

// AgentTypeSupportsModel 检查类型是否支持模型配置
func AgentTypeSupportsModel(ctx context.Context, code string) bool {
	t := GetAgentTypeByCode(ctx, code)
	if t == nil {
		return false
	}
	return t.SupportsModel
}

// AgentTypeSupportsSkill 检查类型是否支持技能安装
func AgentTypeSupportsSkill(ctx context.Context, code string) bool {
	t := GetAgentTypeByCode(ctx, code)
	if t == nil {
		return false
	}
	return t.SupportsSkill
}

// AgentTypeSupportsSkillByMap 基于预加载的 local map 判断类型是否支持技能。
// allTypes 由调用方在循环前通过 GetAllAgentTypesMap(ctx) 一次性获取，避免循环内 N+1。
func AgentTypeSupportsSkillByMap(code string, allTypes map[string]*AgentType) bool {
	t := allTypes[NormalizeAgentType(code)]
	if t == nil {
		return false
	}
	return t.SupportsSkill
}

// GetSkillSupportedAgentTypes 获取所有支持技能的 agent_type code 列表
// 返回值包含空字符串（兼容存量数据）
func GetSkillSupportedAgentTypes(ctx context.Context) []string {
	types := []string{""} // 空字符串兼容存量数据
	for _, t := range GetAllAgentTypes(ctx) {
		if t.SupportsSkill {
			types = append(types, t.Code)
		}
	}
	return types
}

// GetPluginSupportedAgentTypes 获取所有支持插件的 agent_type code 列表
// 返回值包含空字符串（兼容存量数据，存量实例 agent_type 为空默认视为 openclaw）
func GetPluginSupportedAgentTypes(ctx context.Context) []string {
	types := []string{""} // 空字符串兼容存量数据
	for _, t := range GetAllAgentTypes(ctx) {
		if t.SupportsPlugin {
			types = append(types, t.Code)
		}
	}
	return types
}

// AgentTypeSupportsPlugin 检查类型是否支持插件安装
func AgentTypeSupportsPlugin(ctx context.Context, code string) bool {
	t := GetAgentTypeByCode(ctx, code)
	if t == nil {
		return false
	}
	return t.SupportsPlugin
}

// AgentTypeSupportsSMH 检查类型是否支持网盘（SMH 个人空间）
func AgentTypeSupportsSMH(ctx context.Context, code string) bool {
	t := GetAgentTypeByCode(ctx, code)
	if t == nil {
		return false
	}
	return t.SupportsSMH
}

// AgentTypeSupportsMemory 检查类型是否支持记忆功能（Memory TDAI）
func AgentTypeSupportsMemory(ctx context.Context, code string) bool {
	t := GetAgentTypeByCode(ctx, code)
	if t == nil {
		return false
	}
	return t.SupportsMemory
}

// GetMemorySupportedAgentTypes 获取所有支持记忆功能的 agent_type code 列表
// 返回值包含空字符串（兼容存量数据，存量实例 agent_type 为空默认视为 openclaw）
func GetMemorySupportedAgentTypes(ctx context.Context) []string {
	types := []string{""} // 空字符串兼容存量数据
	for _, t := range GetAllAgentTypes(ctx) {
		if t.SupportsMemory {
			types = append(types, t.Code)
		}
	}
	return types
}

// GetSMHSupportedAgentTypes 获取所有支持网盘（SMH 个人空间）的 agent_type code 列表
// 返回值包含空字符串（兼容存量数据，存量实例 agent_type 为空默认视为 openclaw）
func GetSMHSupportedAgentTypes(ctx context.Context) []string {
	types := []string{""} // 空字符串兼容存量数据
	for _, t := range GetAllAgentTypes(ctx) {
		if t.SupportsSMH {
			types = append(types, t.Code)
		}
	}
	return types
}

// GetAgentTypeDisplayName 获取类型显示名称
func GetAgentTypeDisplayName(ctx context.Context, code string) string {
	t := GetAgentTypeByCode(ctx, code)
	if t == nil {
		return code
	}
	return t.Name
}

// ValidateAgentVersion 根据类型校验版本号格式
func ValidateAgentVersion(ctx context.Context, agentType, version string) error {
	if IsCustomAgentType(ctx, agentType) {
		return nil // 自定义类型没有版本概念
	}
	if version == "" {
		return nil // 空版本允许（存量兼容）
	}

	switch agentType {
	case AgentTypeOpenClaw:
		if !openclawVersionRegex.MatchString(version) {
			return hcommon.I18nError(i18n.MsgAgentVersionFormatOpenClaw)
		}
	case AgentTypeHermes, AgentTypeLightclawACE,
		AgentTypeDeepSeekTUI, AgentTypeOpenCode:
		if !semverRegex.MatchString(version) {
			return hcommon.I18nError(i18n.MsgAgentVersionFormatSemver, GetAgentTypeDisplayName(ctx, agentType))
		}
	default:
		if !IsValidAgentVersion(version) {
			return hcommon.I18nError(i18n.MsgAgentVersionFormatInvalid, version)
		}
	}

	return nil
}

// ========== final：通道白名单 + Reinstall 能力查询 ==========

// agentTypeChannelWhitelist 显式列出每种 agentType 允许的 channel_id 集合。
//
// 设计决策（final §3.3）：
//   - 不使用 nil 作"全放行"哨兵，所有 agentType 一律走显式白名单；
//   - channel_id 取值严格对齐 ai_channel.go::predefinedChannels（真实值，非方案文档
//     里的"理论值"）：
//     openclaw-weixin / wecom / wecom_app / feishu / ddingtalk / qqbot / slack
//   - 自定义 Agent 类型会先解析到兼容的内置运行时类型，再使用对应白名单；
//   - 未知 agentType → fail-closed 返回 false。
var agentTypeChannelWhitelist = map[string]map[string]bool{
	AgentTypeOpenClaw: {
		"openclaw-weixin":    true,
		"wecom":              true,
		"wecom_app":          true,
		"feishu":             true,
		"ddingtalk":          true,
		"msteams":            true,
		"qqbot":              true,
		"dingtalk-connector": true,
		"lark":               true,
		"slack":              true,
		"discord":            true,
		"whatsapp":           true,
	},
	AgentTypeHermes: {
		// 注：wecom（企业微信）通过手动配置写入 ~/.hermes/.env（WECOM_BOT_ID/WECOM_SECRET）
		"wecom":           true,
		"openclaw-weixin": true,
		"feishu":          true,
		"lark":            true,
		"ddingtalk":       true,
		"msteams":         true,
		"line":            true,
		"qqbot":           true,
		"slack":           true,
		"discord":         true,
	},
	AgentTypeLightclawACE: {
		"openclaw-weixin": true, // ACE weixin 通道：前端用 openclaw-weixin，脚本层映射为 weixin
		"wecom":           true,
		"feishu":          true,
		"qqbot":           true,
	},
	// DeepSeekTUI / OpenCode 不接入任何通道：显式 fail-closed 空白名单，
	// 让 SupportedChannelsByAgentType 返回 []（前端便于置灰所有通道卡片）。
	AgentTypeDeepSeekTUI: {},
	AgentTypeOpenCode:    {},
}

// AgentTypeChannelAllowed 检查某 agentType 是否允许某 channel_id。
//
// 语义：
//   - 空 agentType 视为 openclaw（兼容存量数据）；
//   - 自定义 Agent 类型按 compatible_with 解析到内置运行时类型；
//   - 无兼容目标或未知 agentType → fail-closed 返回 false；
//   - 该 channel_id 不在运行时类型白名单中 → false。
func AgentTypeChannelAllowed(ctx context.Context, agentType, channelID string) bool {
	runtimeType := GetAgentRuntimeType(ctx, agentType)
	if runtimeType == "" {
		return false
	}
	wl, ok := agentTypeChannelWhitelist[runtimeType]
	if !ok {
		return false
	}
	return wl[channelID]
}

// SupportedChannelsByAgentType 返回某 agentType 的白名单 channel_id 列表（无序）。
// 前端用此置灰不支持的 channel 卡片。未知 agentType 返回 nil。
func SupportedChannelsByAgentType(ctx context.Context, agentType string) []string {
	runtimeType := GetAgentRuntimeType(ctx, agentType)
	if runtimeType == "" {
		return nil
	}
	wl, ok := agentTypeChannelWhitelist[runtimeType]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(wl))
	for k := range wl {
		out = append(out, k)
	}
	return out
}

// SupportedAgentTypesByChannel 返回支持指定 channel_id 的所有 agentType 列表。
// 内置类型和自定义类型统一通过 AgentTypeChannelAllowed 判断；未知 channel_id 返回空 slice。
// 用于 /admin/channels 响应下发，辅助管理员快速识别"哪些 Agent 可以接入该通道"。
func SupportedAgentTypesByChannel(ctx context.Context, channelID string) []string {
	allTypes := GetAllAgentTypes(ctx)
	out := make([]string, 0, len(allTypes))
	for _, t := range allTypes {
		if AgentTypeChannelAllowed(ctx, t.Code, channelID) {
			out = append(out, t.Code)
		}
	}
	return out
}

// GetChannelSupportedAgentTypes 返回所有支持通道配置的 agent_type code 列表。
func GetChannelSupportedAgentTypes(ctx context.Context) []string {
	allTypes := GetAllAgentTypes(ctx)
	out := make([]string, 0, len(allTypes))
	for _, t := range allTypes {
		if t.SupportsChannel {
			out = append(out, t.Code)
		}
	}
	return out
}

func AgentTypeSupportsMCP(ctx context.Context, agentType string) bool {
	return GetAgentRuntimeType(ctx, agentType) == AgentTypeOpenClaw
}

func AgentTypeSupportsMultiAgent(ctx context.Context, agentType string) bool {
	runtimeType := GetAgentRuntimeType(ctx, agentType)
	if runtimeType == "" {
		return false
	}
	t := GetAgentTypeByCode(ctx, runtimeType)
	return t != nil && t.SupportsMultiAgent
}

func GetMCPSupportedAgentTypes(ctx context.Context) []string {
	types := []string{""} // 空字符串兼容存量 openclaw 数据
	for _, t := range GetAllAgentTypes(ctx) {
		if AgentTypeSupportsMCP(ctx, t.Code) {
			types = append(types, t.Code)
		}
	}
	return types
}

// AgentTypeSupportsReinstall 检查类型是否允许 CVM 重装（ResetInstance）。
// OpenClaw / Hermes / ACE 均为 true，各自按 detect_install 分派探测脚本。
func AgentTypeSupportsReinstall(ctx context.Context, code string) bool {
	t := GetAgentTypeByCode(ctx, code)
	return t != nil && t.SupportsReinstall
}

// AgentTypeSupportsUpgrade 检查类型是否支持一键升级（备份→SMH上传→重装→恢复）。
// OpenClaw / Hermes 支持（各自通过 ResolveScript 分派 backup/restore 脚本），ACE 暂未实现。
// 其他类型在 HandleUpgrade / HandleUpgradeRetry 入口拦截。
func AgentTypeSupportsUpgrade(ctx context.Context, code string) bool {
	t := GetAgentTypeByCode(ctx, code)
	return t != nil && t.SupportsUpgrade
}

// AgentTypeSupportsBrowserVNC 检查类型是否支持云端浏览器（Browser VNC）。
// 仅 OpenClaw 为 true。Hermes/ACE 镜像没有 noVNC/Chrome/openclaw 浏览器自动化栈，
// 相关 handler 应返回 403（写操作）或 200 + unsupported（高频只读轮询）。
func AgentTypeSupportsBrowserVNC(ctx context.Context, code string) bool {
	t := GetAgentTypeByCode(ctx, code)
	return t != nil && t.SupportsBrowserVNC
}

// AgentTypeSupportsApprove 检查类型是否支持 approve/approve_device 流程。
// 仅 OpenClaw 为 true。该流程是 openclaw CLI 特有的设备配对 + 飞书/微信授权回调模式，
// Hermes (harness) / ACE (lightclaw) 走自己的 OAuth / Server API，无需此脚本。
func AgentTypeSupportsApprove(ctx context.Context, code string) bool {
	t := GetAgentTypeByCode(ctx, code)
	return t != nil && t.SupportsApprove
}

// AgentTypeSupportsDefaultModelInjection 检查类型是否允许创建实例后自动注入站点默认模型。
//
// 与 AgentTypeSupportsModel 的关系：
//   - AgentTypeSupportsModel：是否允许用户手动 /openclaw/set-model
//   - AgentTypeSupportsDefaultModelInjection：是否允许 CreateInstance 后台 goroutine
//     自动把站点默认模型绑定到新实例
//
// 设计意图：默认模型自动注入与手动 set-model 不是同一语义，前者需要额外确认
// 目标 agent 是否适配当前默认模型注入链路，避免出现首次 set_model 未生效但
// ai_model_id 已写入 DB 的不一致状态。
func AgentTypeSupportsDefaultModelInjection(ctx context.Context, code string) bool {
	t := GetAgentTypeByCode(ctx, code)
	return t != nil && t.SupportsDefaultModelInjection
}

// AgentTypeSupportsAPIGateway 检查类型是否支持 WebUI 云 API 网关域名化接入。
//
// 设计意图：仅 OpenClaw WebUI 契约与网关 CreateSignOnAgentService 的 AgentToken 透传语义对齐；
// Lightclaw/Hermes 的 WebUI 走自身端口 + password/空 token，强接网关反而会导致 token 注入错乱。
func AgentTypeSupportsAPIGateway(ctx context.Context, code string) bool {
	t := GetAgentTypeByCode(ctx, code)
	return t != nil && t.SupportsAPIGateway
}
