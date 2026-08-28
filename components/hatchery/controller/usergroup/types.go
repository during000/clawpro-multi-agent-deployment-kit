package usergroup

import "hatchery/model"

// ──────────────────────────────────────────────
// 可见性类型常量（引用 model 包，避免循环依赖）
// ──────────────────────────────────────────────

// VisibilityType 枚举：资源（模型/通道/技能包/插件包/角色等）的可见范围类型
const (
	VisibilityAll   = model.VisibilityAll   // 全部用户可见
	VisibilityGroup = model.VisibilityGroup // 仅指定分组可见
)

// ──────────────────────────────────────────────
// 配置类型常量与元信息
// ──────────────────────────────────────────────

// ConfigType — group_config_bindings 表中的 config_type 枚举
const (
	ConfigTypeChannel         = "channel"
	ConfigTypePluginBundle    = "plugin_bundle"
	ConfigTypeMCP             = "mcp"
	ConfigTypeImageType       = "image_type"
	ConfigTypePolicy          = "policy"
	ConfigTypeCLSCollectScope = "cls_collect_scope"
)

// CategoryKey — 配置总览中的分类标识（对齐 ConfigCategoryList）
const (
	CategoryKeyModel           = "model"
	CategoryKeyChannel         = "channel"
	CategoryKeySkill           = "skill"
	CategoryKeyAgentTool       = "agentTool"
	CategoryKeyMemory          = "memory"
	CategoryKeyDrive           = "drive"
	CategoryKeyImageType       = "imageType"
	CategoryKeyNetwork         = "network"
	CategoryKeyCLS             = "cls"
	CategoryKeyAIAgentSecurity = "aiAgentSecurity"
	CategoryKeyPlatformPolicy  = "platformPolicy"
	CategoryKeyChargeType      = "chargeType" // T25：实例计费模式
	CategoryKeyResourcePolicy  = "resourcePolicy"
)

// Cardinality 资源基数类型
type Cardinality string

const (
	CardinalityAdditive  Cardinality = "additive"  // 加法型：取并集
	CardinalityExclusive Cardinality = "exclusive" // 单资源型：最近祖先胜出
)

// ConfigTypeMeta 配置类型元信息
type ConfigTypeMeta struct {
	Cardinality Cardinality
	Label       string
}

// ConfigTypes 所有支持的配置类型及其元信息
var ConfigTypes = map[string]ConfigTypeMeta{
	ConfigTypeChannel:         {CardinalityAdditive, "通道"},
	ConfigTypePluginBundle:    {CardinalityAdditive, "企业插件包"},
	ConfigTypeMCP:             {CardinalityAdditive, "MCP"},
	ConfigTypeImageType:       {CardinalityAdditive, "镜像类型"},
	ConfigTypePolicy:          {CardinalityExclusive, "平台策略"},
	ConfigTypeCLSCollectScope: {CardinalityAdditive, "CLS 采集范围"},
}

// IsValidConfigType 校验 config_type 是否合法
func IsValidConfigType(configType string) bool {
	_, ok := ConfigTypes[configType]
	return ok
}

// ──────────────────────────────────────────────
// 策略配置项定义
// ──────────────────────────────────────────────

// PolicyValueType 策略值类型
type PolicyValueType string

const (
	PolicyValueInt    PolicyValueType = "int"
	PolicyValueBool   PolicyValueType = "bool"
	PolicyValueString PolicyValueType = "string"
)

// ── 策略 Key 常量（全局唯一标识） ──

const (
	PolicyKeyTokenQuotaDay = "token_quota_day"
	PolicyKeyInstanceQuota = "instance_quota"
	// 历史策略 key 保留 day；实际统计周期由 site_configs.global_token_quota_period 决定，可表示每日或每月全局 Token 上限。
	PolicyKeyGlobalTokenQuotaDay   = "global_token_quota_day"
	PolicyKeyTokenQuotaRules       = "token_quota_rules"        // JSON 配额规则，优先于 token_quota_day
	PolicyKeyGlobalTokenQuotaRules = "global_token_quota_rules" // JSON 全局/分组配额规则，优先于 global_token_quota_day
	PolicyKeyUserConfigModel       = "user_config_model"
	PolicyKeyUserConfigChannel     = "user_config_channel"
	PolicyKeyCustomModel           = "custom_model"
	PolicyKeyAgentTerminal         = "agent_terminal"
	PolicyKeyGatewayUI             = "gateway_ui"
	PolicyKeyChatView              = "chat_view"
	PolicyKeyBrowserVNC            = "browser_vnc"
	PolicyKeyLobsterDoctor         = "lobster_doctor"
	PolicyKeyModelQuota            = "model_quota"
	PolicyKeySMHAutoProvision      = "smh_auto_provision"
)

// PolicyCategory 策略分类：用户配额 / 模型配额 / 功能权限开关
const (
	PolicyCategoryUserQuota     = "user_quota"
	PolicyCategoryModelQuota    = "model_quota"
	PolicyCategoryFeatureToggle = "feature_toggle"
)

// PolicyDef 策略配置项定义
type PolicyDef struct {
	Key             string          // config_key
	Label           string          // 显示名称
	ValueType       PolicyValueType // 值类型
	SiteConfigField string          // 对应的 site_configs 字段名（用于兜底）
	Category        string          // T29：分类（user_quota / model_quota / feature_toggle）
}

// PolicyDefs 所有支持的策略配置项
var PolicyDefs = map[string]PolicyDef{
	PolicyKeyTokenQuotaDay:         {Key: PolicyKeyTokenQuotaDay, Label: "单用户 Tokens 上限", ValueType: PolicyValueInt, SiteConfigField: "DefaultTokenQuotaDay", Category: PolicyCategoryUserQuota},
	PolicyKeyTokenQuotaRules:       {Key: PolicyKeyTokenQuotaRules, Label: "用户 Token 配额规则", ValueType: PolicyValueString, SiteConfigField: "DefaultTokenQuotaRules", Category: PolicyCategoryUserQuota},
	PolicyKeyInstanceQuota:         {Key: PolicyKeyInstanceQuota, Label: "单用户 Agent 数量上限", ValueType: PolicyValueInt, SiteConfigField: "DefaultInstanceQuota", Category: PolicyCategoryUserQuota},
	PolicyKeyGlobalTokenQuotaDay:   {Key: PolicyKeyGlobalTokenQuotaDay, Label: "全局 Tokens 上限", ValueType: PolicyValueInt, SiteConfigField: "GlobalTokenQuotaDay", Category: PolicyCategoryModelQuota},
	PolicyKeyGlobalTokenQuotaRules: {Key: PolicyKeyGlobalTokenQuotaRules, Label: "全局 Token 配额规则", ValueType: PolicyValueString, SiteConfigField: "GlobalTokenQuotaRules", Category: PolicyCategoryModelQuota},
	PolicyKeyUserConfigModel:       {Key: PolicyKeyUserConfigModel, Label: "允许用户配置模型", ValueType: PolicyValueBool, SiteConfigField: "UserConfigModelEnabled", Category: PolicyCategoryFeatureToggle},
	PolicyKeyUserConfigChannel:     {Key: PolicyKeyUserConfigChannel, Label: "允许用户配置通道", ValueType: PolicyValueBool, SiteConfigField: "UserConfigChannelEnabled", Category: PolicyCategoryFeatureToggle},
	PolicyKeyCustomModel:           {Key: PolicyKeyCustomModel, Label: "允许用户添加自定义模型", ValueType: PolicyValueBool, SiteConfigField: "", Category: PolicyCategoryFeatureToggle},
	PolicyKeyAgentTerminal:         {Key: PolicyKeyAgentTerminal, Label: "允许用户进入 Agent 终端", ValueType: PolicyValueBool, SiteConfigField: "TerminalEnabled", Category: PolicyCategoryFeatureToggle},
	PolicyKeyGatewayUI:             {Key: PolicyKeyGatewayUI, Label: "允许用户访问 Agent 面板", ValueType: PolicyValueBool, SiteConfigField: "GatewayUIEnable", Category: PolicyCategoryFeatureToggle},
	PolicyKeyChatView:              {Key: PolicyKeyChatView, Label: "允许用户使用对话视图", ValueType: PolicyValueBool, SiteConfigField: "ChatViewEnabled", Category: PolicyCategoryFeatureToggle},
	PolicyKeyBrowserVNC:            {Key: PolicyKeyBrowserVNC, Label: "允许用户访问 Agent 云桌面", ValueType: PolicyValueBool, SiteConfigField: "BrowserVNCEnable", Category: PolicyCategoryFeatureToggle},
	PolicyKeyLobsterDoctor:         {Key: PolicyKeyLobsterDoctor, Label: "允许用户使用龙虾医生", ValueType: PolicyValueBool, SiteConfigField: "DoctorEnabled", Category: PolicyCategoryFeatureToggle},
	PolicyKeyModelQuota:            {Key: PolicyKeyModelQuota, Label: "允许用户查看模型额度", ValueType: PolicyValueBool, SiteConfigField: "ModelQuotaEnabled", Category: PolicyCategoryFeatureToggle},
	PolicyKeySMHAutoProvision:      {Key: PolicyKeySMHAutoProvision, Label: "创建实例时自动开启网盘", ValueType: PolicyValueBool, SiteConfigField: "SMHAutoProvisionOnCreate", Category: PolicyCategoryFeatureToggle},
}

// policyKeyOrder 策略配置项的固定展示顺序（用户配额 → 模型配额 → 功能权限开关）
var policyKeyOrder = []string{
	PolicyKeyInstanceQuota,
	PolicyKeyTokenQuotaDay,
	PolicyKeyTokenQuotaRules,
	PolicyKeyGlobalTokenQuotaDay,
	PolicyKeyGlobalTokenQuotaRules,
	PolicyKeyUserConfigModel,
	PolicyKeyUserConfigChannel,
	PolicyKeyCustomModel,
	PolicyKeyAgentTerminal,
	PolicyKeyGatewayUI,
	PolicyKeyChatView,
	PolicyKeyBrowserVNC,
	PolicyKeyLobsterDoctor,
	PolicyKeyModelQuota,
}

// IsValidPolicyKey 校验策略名是否合法
func IsValidPolicyKey(key string) bool {
	_, ok := PolicyDefs[key]
	return ok
}

// QuotaPolicyPairedKey 返回配额策略 key 的配对 key（legacy day ↔ new rules）。
// token_quota_day ↔ token_quota_rules，global_token_quota_day ↔ global_token_quota_rules。
// 非配额 key 返回空串。
func QuotaPolicyPairedKey(key string) string {
	switch key {
	case PolicyKeyTokenQuotaDay:
		return PolicyKeyTokenQuotaRules
	case PolicyKeyTokenQuotaRules:
		return PolicyKeyTokenQuotaDay
	case PolicyKeyGlobalTokenQuotaDay:
		return PolicyKeyGlobalTokenQuotaRules
	case PolicyKeyGlobalTokenQuotaRules:
		return PolicyKeyGlobalTokenQuotaDay
	}
	return ""
}

// GetPolicyDef 获取策略定义
func GetPolicyDef(key string) (PolicyDef, bool) {
	def, ok := PolicyDefs[key]
	return def, ok
}

// ──────────────────────────────────────────────
// 配置总览来源类型
// ──────────────────────────────────────────────

// SourceType 配置来源类型
type SourceType string

const (
	SourceLocal       SourceType = "local"        // 本组直接配置
	SourceInherited   SourceType = "inherited"    // 继承自祖先组
	SourceAllUsers    SourceType = "all_users"    // 全部用户可见（加法型 visibility_type='all'）
	SourceSiteDefault SourceType = "site_default" // 平台默认值（site_configs）
	SourceGlobal      SourceType = "global"       // 全局配置项（网络/记忆/CLS 等，不按组配置）
	SourceUnset       SourceType = "unset"        // 未配置
)

// Source 配置来源信息
type Source struct {
	Type     SourceType `json:"type"`
	GroupID  uint       `json:"group_id,omitempty"`
	FullPath string     `json:"full_path,omitempty"`
}
