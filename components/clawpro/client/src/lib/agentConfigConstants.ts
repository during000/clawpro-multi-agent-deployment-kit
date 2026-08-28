/**
 * agentConfigConstants.ts
 * Agent 模型 / 通道 配置共享常量
 * 同时被用户端 (tenant/OpenClawDetail) 与管控端 (admin/OpenClawMonitor) 引用，
 * 避免重复定义导致后续维护漂移。
 */

// ─── 通道配置类型与选项 ────────────────────────────────────────────────────────

export type ChannelField = {
  key: string;
  label: string;
  /** true = 加密显示（保留前3字符） */
  secret: boolean;
};

export type ChannelConfig = {
  value: string;
  label: string;
  descText: string;
  detailUrl: string;
  hasInfoIcon?: boolean;
  fields?: ChannelField[];
  /** 飞书特殊处理（用户端：快捷/手动 Tab + 二维码） */
  feishuMode?: true;
  /** 企业微信特殊处理（用户端：快捷/手动 Tab） */
  weworkMode?: true;
  /** 微信特殊处理（用户端：扫码授权） */
  wechatMode?: true;
  /** WhatsApp 特殊处理（用户端：手机号 + Pairing Code 配对流程） */
  whatsappMode?: true;
  /** 管控端配置的自定义通道（用户端动态注入） */
  adminCustomMode?: true;
  adminCustomId?: string;
  /** 管控端自定义通道原始 channelId（用于识别特殊通道类型，如 whatsapp） */
  adminChannelId?: string;
  /** 对应管控端内置通道 ID，用于可见性过滤 */
  builtinId?: string;
};

export const CHANNEL_OPTIONS: ChannelConfig[] = [
  {
    value: "wework",
    label: "企业微信",
    descText: "企业微信是一款高效协同办公的企业通讯与办公工具。",
    detailUrl: "#",
    hasInfoIcon: true,
    weworkMode: true,
    fields: [
      { key: "botId", label: "企业微信机器人的botId", secret: false },
      { key: "secret", label: "企业微信机器人的secret", secret: true },
    ],
  },
  {
    value: "qq",
    label: "QQ",
    descText: "一键解锁智能玩法，开启你的个性化QQ机器人之旅。",
    detailUrl: "#",
    fields: [
      { key: "appId", label: "QQ机器人的App ID", secret: false },
      { key: "appSecret", label: "QQ机器人的App Secret", secret: true },
    ],
  },
  {
    value: "wework-app",
    label: "企业微信应用",
    descText: "通过企业微信应用接口，将 Agent 接入企业微信应用，支持消息互动与业务集成。",
    detailUrl: "#",
    fields: [
      { key: "corpId",         label: "企业微信应用的Corp ID",           secret: false },
      { key: "corpSecret",     label: "企业微信应用的Corp Secret",       secret: true  },
      { key: "agentId",        label: "企业微信应用的Agent ID",          secret: false },
      { key: "token",          label: "企业微信应用的Token",             secret: false },
      { key: "encodingAesKey", label: "企业微信应用的Encoding AES Key", secret: true  },
    ],
    builtinId: "wework-app",
  },
  {
    value: "feishu",
    label: "飞书",
    descText: "飞书是字节跳动推出的一站式先进协作平台，AI 赋能助力高效办公。",
    detailUrl: "#",
    feishuMode: true,
    // 快捷配置和手动配置都存 appId + appSecret
    fields: [
      { key: "appId", label: "飞书应用的App ID", secret: false },
      { key: "appSecret", label: "飞书应用的App Secret", secret: true },
    ],
  },
  {
    value: "dingtalk",
    label: "钉钉",
    descText: "钉钉是阿里打造的智能办公平台，驱动组织数字化管理升级。",
    detailUrl: "#",
    fields: [
      { key: "clientId", label: "钉钉应用的Client ID", secret: false },
      { key: "clientSecret", label: "钉钉应用的Client Secret", secret: true },
    ],
  },
  {
    value: "wechat",
    label: "微信",
    descText: "通过微信扫码授权，将 Agent 接入微信，支持微信消息交互。",
    detailUrl: "#",
    wechatMode: true,
  },
];

// ─── 模型配置类型与选项 ────────────────────────────────────────────────────────

export type ModelVersion = {
  value: string;
  label: string;
  badge?: string;
  badgeColor?: string;
};

/**
 * 模型来源分组：
 * - "admin"：管理员配置的模型（管理员预置、对用户可见，用户直接选用）
 * - "self" ：自行配置（用户自己添加，目前为「自定义模型」）
 */
export type ModelProviderGroup = "admin" | "self";

/**
 * 「自行配置」下的二级分类（与管控端添加模型一致）：
 * - "codingPlan"：Coding Plan 套餐（按套餐计费，选厂商/套餐 + Key + 限额）
 * - "modelApi" ：模型 API（公开厂商，选厂商/版本 + Key + 限额）
 * - "custom"   ：自定义模型（JSON / 表单自由填写）
 * 仅当 group === "self" 时该字段有意义。
 */
export type SelfConfigCategory = "codingPlan" | "modelApi" | "custom";

export type ModelProvider = {
  value: string;
  label: string;
  versions: ModelVersion[];
  /** 模型来源分组，用于下拉分区展示，缺省视为 "admin" */
  group?: ModelProviderGroup;
  /**
   * 「自行配置」下的二级分类，仅当 group === "self" 时有意义。
   * codingPlan / modelApi 为可选具体厂商；custom 为自定义模型入口。
   */
  selfCategory?: SelfConfigCategory;
  /**
   * true = 该厂商为「自定义模型」入口（走 JSON / 表单自由填写）。
   * 其余 group="self" 的厂商均为「公开模型」——用户自行选择厂商/版本、
   * 填写 API Key、限额、高级配置并做连通性检测。
   */
  isCustom?: boolean;
  /**
   * true = 管理员预置的「由用户端自行配置」类模型（group="admin" 时有意义）。
   * 管理员在管控端添加该自定义模型时未填写密钥，改由用户端使用时自行填写 API Key。
   * 用户端选用此类模型时需手动填写 API Key，并支持连通性检测。
   */
  userProvidedKey?: boolean;
};

/** 模型来源分组的展示文案（下拉 SelectLabel 使用） */
export const MODEL_PROVIDER_GROUP_LABELS: Record<ModelProviderGroup, string> = {
  admin: "管理员预置",
  self: "自行配置",
};

/** 分组在下拉中的展示顺序 */
export const MODEL_PROVIDER_GROUP_ORDER: ModelProviderGroup[] = ["admin", "self"];

/** 「自行配置」二级分类的展示文案与顺序 */
export const SELF_CONFIG_CATEGORY_LABELS: Record<SelfConfigCategory, string> = {
  codingPlan: "Coding Plan",
  modelApi: "模型 API",
  custom: "自定义模型",
};

export const SELF_CONFIG_CATEGORY_ORDER: SelfConfigCategory[] = ["codingPlan", "modelApi", "custom"];

export const MODEL_PROVIDERS: ModelProvider[] = [
  {
    value: "tencent-deepseek",
    label: "腾讯云 DeepSeek",
    group: "admin",
    versions: [
      { value: "deepseek-v3", label: "DeepSeek V3 0324" },
      { value: "deepseek-r1", label: "DeepSeek R1" },
    ],
  },
  {
    value: "tencent-hunyuan",
    label: "腾讯云混元",
    group: "admin",
    versions: [
      { value: "hunyuan-turbos", label: "混元 TurboS Latest" },
      { value: "hunyuan-pro", label: "混元 Pro" },
    ],
  },
  {
    // 管理员预置的「由用户端自行配置」自定义模型：管理员未填写密钥，
    // 用户端选用时需自行填写 API Key 并做连通性检测。
    value: "admin-custom-userkey",
    label: "自定义模型/deepseek-v3-custom（需自行填写 Key）",
    group: "admin",
    userProvidedKey: true,
    versions: [
      { value: "deepseek-v3-custom", label: "deepseek-v3-custom" },
    ],
  },
  // ── 自行配置 · Coding Plan（套餐式，选厂商/套餐 + Key + 限额）──
  {
    value: "tencent-coding-plan",
    label: "腾讯云 Coding Plan",
    group: "self",
    selfCategory: "codingPlan",
    versions: [
      { value: "auto", label: "自动" },
    ],
  },
  {
    value: "claude-coding-plan",
    label: "Claude Code Plan",
    group: "self",
    selfCategory: "codingPlan",
    versions: [
      { value: "claude-sonnet", label: "Claude Sonnet" },
      { value: "claude-opus", label: "Claude Opus" },
    ],
  },
  {
    value: "kimi-coding-plan",
    label: "Kimi Coding Plan",
    group: "self",
    selfCategory: "codingPlan",
    versions: [
      { value: "kimi-coding", label: "自动" },
    ],
  },
  // ── 自行配置 · 模型 API（公开厂商，选厂商/版本 + Key + 限额）──
  {
    value: "deepseek",
    label: "深度求索（DeepSeek）",
    group: "self",
    selfCategory: "modelApi",
    versions: [
      { value: "deepseek-chat", label: "deepseek-chat" },
      { value: "deepseek-reasoner", label: "deepseek-reasoner" },
    ],
  },
  {
    value: "bailian",
    label: "百炼（千问）",
    group: "self",
    selfCategory: "modelApi",
    versions: [
      { value: "qwen-max", label: "qwen-max" },
      { value: "qwen-plus", label: "qwen-plus" },
      { value: "qwen-turbo", label: "qwen-turbo" },
    ],
  },
  {
    value: "minimax",
    label: "MiniMax（国内）",
    group: "self",
    selfCategory: "modelApi",
    versions: [
      { value: "abab6.5s-chat", label: "abab6.5s-chat" },
      { value: "abab6.5g-chat", label: "abab6.5g-chat" },
    ],
  },
  {
    value: "moonshot",
    label: "Moonshot AI（Kimi国内）",
    group: "self",
    selfCategory: "modelApi",
    versions: [
      { value: "moonshot-v1-8k", label: "moonshot-v1-8k" },
      { value: "moonshot-v1-32k", label: "moonshot-v1-32k" },
      { value: "moonshot-v1-128k", label: "moonshot-v1-128k" },
    ],
  },
  {
    value: "zhipu",
    label: "智谱 AI（GLM国内）",
    group: "self",
    selfCategory: "modelApi",
    versions: [
      { value: "glm-4-plus", label: "glm-4-plus" },
      { value: "glm-4", label: "glm-4" },
      { value: "glm-4-air", label: "glm-4-air" },
    ],
  },
  {
    value: "volcengine",
    label: "火山引擎（豆包）",
    group: "self",
    selfCategory: "modelApi",
    versions: [
      { value: "doubao-pro-32k", label: "doubao-pro-32k" },
      { value: "doubao-pro-128k", label: "doubao-pro-128k" },
      { value: "doubao-lite-32k", label: "doubao-lite-32k" },
    ],
  },
  // ── 自行配置 · 自定义模型 ──
  {
    value: "custom",
    label: "自定义模型",
    group: "self",
    selfCategory: "custom",
    isCustom: true,
    versions: [
      { value: "custom", label: "自定义模型", badge: "需自费", badgeColor: "bg-amber-50 text-amber-600 border-amber-100" },
    ],
  },
];

/**
 * 按来源分组返回厂商列表，保留每组内原始顺序。
 * 返回结构供下拉用 SelectGroup + SelectLabel 分区渲染。
 */
export function getGroupedModelProviders(): { group: ModelProviderGroup; label: string; providers: ModelProvider[] }[] {
  return MODEL_PROVIDER_GROUP_ORDER
    .map((group) => ({
      group,
      label: MODEL_PROVIDER_GROUP_LABELS[group],
      providers: MODEL_PROVIDERS.filter((p) => (p.group ?? "admin") === group),
    }))
    .filter((g) => g.providers.length > 0);
}

/** 管理员配置的模型（group === "admin"）。 */
export function getAdminModelProviders(): ModelProvider[] {
  return MODEL_PROVIDERS.filter((p) => (p.group ?? "admin") === "admin");
}

/** 按「自行配置」二级分类返回厂商列表（custom 分类返回自定义模型入口）。 */
export function getSelfProvidersByCategory(category: SelfConfigCategory): ModelProvider[] {
  return MODEL_PROVIDERS.filter((p) => p.group === "self" && p.selfCategory === category);
}

export const DEFAULT_CUSTOM_JSON = `{
  "provider": "provider_name",
  "base_url": "baseurl",
  "api": "API协议",
  "api_key": "your-api-key-here",
  "model": {
    "id": "model_id",
    "name": "model_name"
  }
}`;
