package usergroup

// ──────────────────────────────────────────────
// 配置总览 — Category 元信息与 Entry 结构
// ──────────────────────────────────────────────

// ConfigCategoryMeta 配置分类元信息（对齐产品 demo 的 CONFIG_CATEGORY_META）
type ConfigCategoryMeta struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// ConfigCategoryList 所有配置分类的固定顺序。
// 注：chargeType 置于首项，便于 stale-instances config-diff 弹窗对比列表首项渲染。
var ConfigCategoryList = []ConfigCategoryMeta{
	{Key: CategoryKeyChargeType, Label: "计费模式", Description: "实例计费类型（按量计费 / 包年包月）", Icon: "CreditCard"},
	{Key: CategoryKeyResourcePolicy, Label: "资源策略", Description: "创建 Agent 使用的云资源配置", Icon: "ServerCog"},
	{Key: CategoryKeyModel, Label: "模型", Description: "用户能使用哪些模型", Icon: "Brain"},
	{Key: CategoryKeyChannel, Label: "通道", Description: "用户通过哪些通道访问模型", Icon: "MessageSquare"},
	{Key: CategoryKeySkill, Label: "技能", Description: "初始技能包、角色与技能安装来源", Icon: "Puzzle"},
	{Key: CategoryKeyAgentTool, Label: "Agent 工具", Description: "企业技能、企业插件与企业 MCP", Icon: "Wrench"},
	{Key: CategoryKeyMemory, Label: "记忆", Description: "记忆功能状态", Icon: "MemoryStick"},
	{Key: CategoryKeyDrive, Label: "网盘", Description: "网盘功能开关", Icon: "FolderOpen"},
	{Key: CategoryKeyImageType, Label: "镜像", Description: "Agent 运行镜像", Icon: "HardDrive"},
	{Key: CategoryKeyNetwork, Label: "网络", Description: "私有网络、安全组与公网配置", Icon: "ShieldCheck"},
	{Key: CategoryKeyCLS, Label: "CLS 日志服务", Description: "用于运维观测与会话管理", Icon: "Gauge"},
	{Key: CategoryKeyAIAgentSecurity, Label: "AI Agent 安全", Description: "AI Agent 安全防护开关", Icon: "Shield"},
	{Key: CategoryKeyPlatformPolicy, Label: "平台策略", Description: "用户配额、模型配额与功能权限开关", Icon: "Shield"},
}

// ConfigEntry 配置总览中的单条配置项
type ConfigEntry struct {
	ID       string      `json:"id"`
	Label    string      `json:"label"`
	SubLabel string      `json:"sub_label,omitempty"`
	NameHint string      `json:"-"` // diff 层用：公网子项 ID 即人读名，需与 Label（实际值）分开
	Source   Source      `json:"source"`
	Meta     interface{} `json:"meta,omitempty"`
}

// ConfigCategoryResult 单个分类的总览结果
type ConfigCategoryResult struct {
	Key         string        `json:"key"`
	Label       string        `json:"label"`
	Description string        `json:"description"`
	Icon        string        `json:"icon"`
	Entries     []ConfigEntry `json:"entries"`
}
