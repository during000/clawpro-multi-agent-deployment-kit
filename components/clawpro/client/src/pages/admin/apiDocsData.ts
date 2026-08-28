// API 文档结构化数据
// 导航树 + API 概览表 + 静态页面内容

// ────────────────────────────────────────
// 类型定义
// ────────────────────────────────────────

export interface NavItem {
  id: string;
  label: string;
  children?: NavItem[];
  /** 额外搜索关键词（接口描述等），不在导航中显示，仅用于搜索匹配 */
  keywords?: string[];
}

export interface OverviewEntry {
  name: string;        // "METHOD /path"
  description: string; // 接口功能
}

export interface OverviewSection {
  title: string;
  sectionId: string;
  entries: OverviewEntry[];
}

export interface EndpointParam {
  name: string;
  type: string;
  required: string;
  description: string;
}

export interface OutputParam {
  name: string;
  type: string;
  description: string;
}

export interface ErrorCode {
  code: string;
  error: string;
}

export interface EndpointDetail {
  method: string;
  path: string;
  description: string;
  auth: string;
  contentType?: string;
  inputParams: EndpointParam[];
  outputParams: OutputParam[];
  requestExample: string;
  responseExample: string;
  errorCodes: ErrorCode[];
}

// ────────────────────────────────────────
// 导航树（3 个顶级组织）
// 接口文档下的每个分类展开后显示具体接口
// ────────────────────────────────────────

// 生成单个接口的导航节点 id，格式：endpoint:{sectionId}:{METHOD /path}
export function endpointNavId(sectionId: string, endpointName: string): string {
  return `endpoint:${sectionId}:${endpointName}`;
}

// 解析导航节点 id，返回 { sectionId, endpointName } 或 null
export function parseEndpointNavId(id: string): { sectionId: string; endpointName: string } | null {
  if (!id.startsWith("endpoint:")) return null;
  const firstColon = id.indexOf(":");
  const secondColon = id.indexOf(":", firstColon + 1);
  if (secondColon === -1) return null;
  return {
    sectionId: id.substring(firstColon + 1, secondColon),
    endpointName: id.substring(secondColon + 1),
  };
}

export const navTree: NavItem[] = [
  { id: "intro", label: "简介", keywords: ["基础信息", "Base URL", "协议", "数据格式", "REST API", "ClawPro", "Agent"] },
  { id: "changelog", label: "更新历史", keywords: ["版本", "发布", "变更", "更新日志"] },
  { id: "overview", label: "API 概览", keywords: ["接口列表", "汇总", "所有接口"] },
  {
    id: "calling",
    label: "调用方式",
    keywords: ["请求", "认证", "返回", "参数"],
    children: [
      { id: "request-structure", label: "请求结构", keywords: ["HTTP", "请求方法", "GET", "POST", "PUT", "DELETE", "请求头", "Header", "Authorization", "Content-Type", "Base URL"] },
      { id: "auth", label: "认证方式", keywords: ["Token", "Bearer", "API Token", "管理员", "权限", "认证", "鉴权"] },
      { id: "response", label: "返回结果", keywords: ["JSON", "成功", "失败", "错误", "响应格式", "error", "ok"] },
      { id: "param-types", label: "参数类型", keywords: ["string", "int", "bool", "float", "array", "object", "类型说明"] },
    ],
  },
  {
    id: "api",
    label: "接口文档",
    children: [], // 将在 overviewSections 定义之后动态填充
  },
];

// ────────────────────────────────────────
// API 概览表格（2 列：接口名称 + 接口功能）
// ────────────────────────────────────────

export const overviewSections: OverviewSection[] = [
  {
    title: "公共接口",
    sectionId: "public",
    entries: [
      { name: "GET /site", description: "获取站点基础信息" },
    ],
  },
  {
    title: "实例管理",
    sectionId: "instance",
    entries: [
      { name: "GET /openclaw/list", description: "获取当前用户的实例列表" },
      { name: "POST /openclaw/create", description: "创建一个新实例" },
      { name: "POST /openclaw/delete", description: "删除指定实例" },
      { name: "GET /openclaw/status", description: "查询指定实例的云服务器运行状态" },
      { name: "POST /openclaw/reboot", description: "重启指定实例的云服务器" },
      { name: "POST /openclaw/reset", description: "重装指定实例的云服务器系统" },
      { name: "POST /openclaw/approve", description: "在实例上执行通道审批操作" },
      { name: "GET /openclaw/service-status", description: "查询指定实例上的服务运行状态" },
      { name: "POST /openclaw/terminal-url", description: "获取指定实例的 Web 终端登录 URL" },
      { name: "POST /openclaw/denied-actions", description: "批量查询实例的禁用操作列表" },
      { name: "GET /openclaw/zones", description: "查询当前地域下的可用区列表" },
      { name: "POST /openclaw/set-gateway-ui", description: "为指定实例开启 Gateway UI 面板" },
      { name: "POST /openclaw/retry", description: "重试加载失败的实例" },
      { name: "GET /openclaw/check-agent-port", description: "检查实例上的服务端口是否正在运行" },
      { name: "GET /openclaw/check-gateway-access", description: "检查实例的 Gateway UI 端口安全组是否可访问" },
      { name: "POST /openclaw/set-env", description: "为指定实例批量设置或删除环境变量" },
      { name: "GET /openclaw/env", description: "查询指定实例当前的环境变量列表" },
    ],
  },
  {
    title: "通知管理",
    sectionId: "notification",
    entries: [
      { name: "GET /openclaw/notifications", description: "获取当前用户的通知列表" },
      { name: "POST /openclaw/notifications/read", description: "标记通知为已读" },
      { name: "GET /openclaw/notifications/count", description: "获取当前用户的未读通知数量" },
    ],
  },
  {
    title: "通道管理",
    sectionId: "channel",
    entries: [
      { name: "GET /openclaw/channels", description: "获取通道列表" },
      { name: "POST /openclaw/set-channel", description: "为指定实例配置通道凭证" },
      { name: "POST /openclaw/del-channel", description: "删除指定实例上的通道配置" },
      { name: "GET /openclaw/auto-channel", description: "自动配置通道（SSE 流式）" },
    ],
  },
  {
    title: "模型管理",
    sectionId: "model",
    entries: [
      { name: "GET /openclaw/models", description: "获取所有已启用的 AI 模型列表" },
      { name: "POST /openclaw/set-model", description: "为指定实例绑定 AI 模型" },
    ],
  },
  {
    title: "技能管理",
    sectionId: "skill",
    entries: [
      { name: "GET /openclaw/skills", description: "获取指定实例上已安装的技能列表" },
      { name: "POST /openclaw/add-skill", description: "为指定实例安装技能" },
      { name: "GET /openclaw/install-skills", description: "查询指定实例的技能安装状态列表" },
      { name: "POST /openclaw/retry-failed-skills", description: "重试指定实例上安装失败的技能" },
      { name: "POST /openclaw/cancel-failed-skills", description: "取消指定实例上安装失败的技能" },
    ],
  },
  {
    title: "插件管理",
    sectionId: "plugin",
    entries: [
      { name: "POST /openclaw/add-plugin", description: "为指定实例安装插件" },
    ],
  },
  {
    title: "用户配额",
    sectionId: "quota",
    entries: [
      { name: "GET /quota/data", description: "查询当前用户的模型用量统计数据" },
      { name: "GET /quota/logs", description: "查询当前用户的模型调用明细日志" },
    ],
  },
  {
    title: "管理接口 — 站点配置",
    sectionId: "admin-site",
    entries: [
      { name: "GET /admin/config", description: "获取站点配置信息" },
      { name: "POST /admin/config", description: "更新站点基础配置" },
      { name: "POST /admin/config/cvm", description: "更新云服务器相关配置" },
      { name: "POST /admin/config/template", description: "更新云服务器模板的指定子模块" },
      { name: "GET /admin/vpc/cloud", description: "查询当前区域下的云端 VPC 列表" },
      { name: "GET /admin/subnet/cloud", description: "查询指定 VPC 和可用区下的云端子网列表" },
    ],
  },
  {
    title: "管理接口 — 安全组",
    sectionId: "admin-security",
    entries: [
      { name: "GET /admin/config/security-group", description: "查询当前已配置的安全组信息" },
      { name: "POST /admin/config/security-group", description: "创建新的安全组" },
      { name: "PUT /admin/config/security-group", description: "修改当前已绑定的安全组属性" },
      { name: "GET /admin/config/security-group/policies", description: "查询当前安全组的入站和出站规则列表" },
      { name: "POST /admin/config/security-group/policies", description: "为当前安全组添加入站或出站规则" },
      { name: "PUT /admin/config/security-group/policies", description: "替换当前安全组的指定规则" },
      { name: "DELETE /admin/config/security-group/policies", description: "删除当前安全组的指定规则" },
    ],
  },
  {
    title: "管理接口 — 用户管理",
    sectionId: "admin-user",
    entries: [
      { name: "GET /admin/users", description: "获取用户列表" },
      { name: "POST /admin/create", description: "创建新用户" },
      { name: "POST /admin/batch-create", description: "批量创建用户" },
      { name: "POST /admin/delete", description: "软删除指定用户（封禁）" },
      { name: "POST /admin/hard-delete", description: "永久删除指定用户" },
      { name: "POST /admin/restore", description: "恢复已封禁的用户" },
      { name: "POST /admin/reset-password", description: "重置指定用户的密码" },
      { name: "POST /admin/update-user", description: "更新指定用户的信息" },
      { name: "POST /admin/export-tokens", description: "批量导出所有用户的 API Token" },
      { name: "GET /admin/user-limit", description: "查询当前用户数量和用户数上限" },
      { name: "GET /admin/user-vpc", description: "查询指定用户自动创建的 VPC 信息" },
      { name: "GET /admin/departments", description: "查询部门列表和部门树" },
      { name: "POST /admin/token/disable", description: "禁用指定用户的 API Token" },
      { name: "POST /admin/token/enable", description: "启用指定用户的 API Token" },
    ],
  },
  {
    title: "管理接口 — 通道管理",
    sectionId: "admin-channel",
    entries: [
      { name: "GET /admin/channels", description: "查询所有 AI 通道列表" },
      { name: "POST /admin/channels/toggle", description: "切换指定通道的启用/禁用状态" },
      { name: "POST /admin/channels/add", description: "添加自定义通道" },
      { name: "POST /admin/channels/delete", description: "删除指定的自定义通道" },
    ],
  },
  {
    title: "管理接口 — 模型管理",
    sectionId: "admin-model",
    entries: [
      { name: "GET /admin/models", description: "查询所有 AI 模型列表" },
      { name: "POST /admin/models/create", description: "创建新的自定义 AI 模型" },
      { name: "POST /admin/models/update", description: "更新自定义 AI 模型配置" },
      { name: "POST /admin/models/delete", description: "删除指定的自定义 AI 模型" },
      { name: "POST /admin/models/toggle", description: "切换指定 AI 模型的启用/禁用状态" },
      { name: "POST /admin/models/toggle-default", description: "切换指定 AI 模型的默认状态" },
    ],
  },
  {
    title: "管理接口 — 镜像管理",
    sectionId: "admin-image",
    entries: [
      { name: "GET /admin/images", description: "查询所有已导入的镜像列表" },
      { name: "GET /admin/images/cloud", description: "查询云端可导入的私有镜像列表" },
      { name: "POST /admin/images/import", description: "从云端导入指定镜像" },
      { name: "POST /admin/images/delete", description: "删除指定的已导入镜像" },
      { name: "POST /admin/images/enable", description: "切换指定镜像的启用/禁用状态" },
    ],
  },
  {
    title: "管理接口 — 实例监控",
    sectionId: "admin-instance",
    entries: [
      { name: "GET /admin/instances", description: "分页查询所有用户的实例列表" },
      { name: "POST /admin/instances/delete", description: "管理员强制删除指定实例" },
      { name: "POST /admin/instances/terminal-url", description: "获取指定实例的终端授权访问 URL" },
      { name: "POST /admin/instances/denied-actions", description: "批量查询实例的禁用操作" },
      { name: "GET /admin/instances/status", description: "查询指定实例的云实例运行状态" },
      { name: "GET /admin/instances/channels", description: "查询指定实例的已配置通道列表" },
      { name: "GET /admin/instances/skills", description: "查询指定实例的已安装技能列表" },
      { name: "POST /admin/instances/start", description: "管理员开机指定实例" },
      { name: "POST /admin/instances/stop", description: "管理员关机指定实例" },
      { name: "POST /admin/instances/reboot", description: "管理员重启指定实例" },
      { name: "POST /admin/instances/reset", description: "管理员重装指定实例" },
    ],
  },
  {
    title: "管理接口 — 技能包管理",
    sectionId: "admin-skill-bundle",
    entries: [
      { name: "POST /admin/skill-bundles/create", description: "创建新的技能包" },
      { name: "GET /admin/skill-bundles", description: "查询技能包列表" },
      { name: "POST /admin/skill-bundles/delete", description: "删除指定技能包" },
      { name: "POST /admin/skill-bundles/toggle", description: "启用或禁用指定技能包" },
      { name: "GET /admin/skill-bundles/detail", description: "查询技能包详情及包内技能列表" },
      { name: "POST /admin/skill-bundles/update-skills", description: "批量更新技能包内的技能" },
    ],
  },
  {
    title: "管理接口 — 技能收藏",
    sectionId: "admin-skill-fav",
    entries: [
      { name: "POST /admin/skills/favorite", description: "收藏公共技能" },
      { name: "POST /admin/skills/unfavorite", description: "取消收藏公共技能" },
      { name: "GET /admin/skills/favorited", description: "查询已收藏的公共技能列表" },
    ],
  },
  {
    title: "管理接口 — 使用统计",
    sectionId: "admin-usage",
    entries: [
      { name: "GET /admin/usage/data", description: "查询模型使用量统计数据" },
      { name: "GET /admin/usage/logs", description: "分页查询 LLM 使用明细日志" },
    ],
  },
  {
    title: "管理接口 — 审计日志",
    sectionId: "admin-audit",
    entries: [
      { name: "GET /admin/audit", description: "分页查询系统操作审计日志" },
    ],
  },
];

// ────────────────────────────────────────
// 静态页面内容
// ────────────────────────────────────────

export const baseInfo = {
  baseUrl: "https://example.com/api",
  protocol: "HTTPS",
  dataFormat: "JSON",
};

export const authInfo = {
  header: "Authorization: Bearer hk-xxxxxxxxxxxxxxxxxxxx",
  tokenTypes: [
    {
      type: "用户 API Token",
      prefix: "hk-",
      scope: "当前用户的业务接口和管理接口（取决于用户角色）",
    },
  ],
  note: "管理接口（/admin/*）需要管理员角色的 Token。",
};

export const responseFormat = {
  success: `{"ok": true}`,
  successNote: "或返回具体数据对象/数组。",
  error: `{
  "error": "错误描述",
  "detail": "详细错误信息（可选）",
  "request_id": "请求ID（可选）"
}`,
};

export const changelogEntries = [
  {
    version: "第 3 次发布",
    date: "2026-04-07",
    summary: "本次发布包含了以下内容：",
    sections: [
      {
        title: "新增接口",
        groups: [
          {
            name: "实例管理",
            items: [
              "POST /openclaw/set-env（为指定实例批量设置或删除环境变量）",
              "GET /openclaw/env（查询指定实例当前的环境变量列表）",
            ],
          },
        ],
      },
      {
        title: "改善已有的文档",
        notes: [
          "补充管理接口缺失的输出参数字段",
          "修正批量创建用户和导出 Token 接口的字段前缀",
        ],
      },
    ],
  },
  {
    version: "第 2 次发布",
    date: "2026-04-01",
    summary: "本次发布包含了以下内容：",
    sections: [
      {
        title: "新增接口",
        groups: [
          {
            name: "实例管理",
            items: [
              "POST /openclaw/retry（重试加载失败的实例）",
              "GET /openclaw/check-agent-port（检查实例服务端口）",
              "GET /openclaw/check-gateway-access（检查 Gateway UI 端口可访问性）",
            ],
          },
          {
            name: "通知管理（新增模块）",
            items: [
              "GET /openclaw/notifications（获取通知列表）",
              "POST /openclaw/notifications/read（标记通知已读）",
              "GET /openclaw/notifications/count（获取未读通知数）",
            ],
          },
          {
            name: "技能管理",
            items: [
              "GET /openclaw/install-skills（查询技能安装状态列表）",
              "POST /openclaw/retry-failed-skills（重试安装失败的技能）",
              "POST /openclaw/cancel-failed-skills（取消安装失败的技能）",
            ],
          },
          {
            name: "管理接口 — 用户管理",
            items: [
              "GET /admin/departments（查询部门列表和部门树）",
              "POST /admin/token/disable（禁用用户 API Token）",
              "POST /admin/token/enable（启用用户 API Token）",
            ],
          },
          {
            name: "管理接口 — 模型管理",
            items: [
              "POST /admin/models/toggle-default（切换默认模型）",
            ],
          },
          {
            name: "管理接口 — 实例监控",
            items: [
              "POST /admin/instances/start（管理员开机实例）",
              "POST /admin/instances/stop（管理员关机实例）",
              "POST /admin/instances/reboot（管理员重启实例）",
              "POST /admin/instances/reset（管理员重装实例）",
            ],
          },
          {
            name: "管理接口 — 技能包管理（新增模块）",
            items: [
              "POST /admin/skill-bundles/create（创建技能包）",
              "GET /admin/skill-bundles（查询技能包列表）",
              "POST /admin/skill-bundles/delete（删除技能包）",
              "POST /admin/skill-bundles/toggle（切换技能包启用状态）",
              "GET /admin/skill-bundles/detail（查询技能包详情）",
              "POST /admin/skill-bundles/update-skills（更新技能包内技能）",
            ],
          },
          {
            name: "管理接口 — 技能收藏（新增模块）",
            items: [
              "POST /admin/skills/favorite（收藏公共技能）",
              "POST /admin/skills/unfavorite（取消收藏技能）",
              "GET /admin/skills/favorited（查询已收藏技能列表）",
            ],
          },
        ],
      },
      {
        title: "改善已有的文档",
        notes: [
          "完善所有接口的输入参数、输出参数和错误码描述",
          "补充更多详细的请求和响应示例",
        ],
      },
    ],
  },
  {
    version: "第 1 次发布",
    date: "2026-03-28",
    summary: "本次发布包含了以下内容：",
    sections: [
      {
        title: "说明",
        notes: [
          "发布 ClawPro API 初始版本，包含 72 个接口，涵盖公共接口、实例管理、通道管理、模型管理、技能管理、插件管理、用户配额、站点配置、安全组、用户管理、通道管理（管理）、模型管理（管理）、镜像管理、实例监控、使用统计和审计日志等模块。",
        ],
      },
    ],
  },
];

export const parameterTypes = [
  { type: "string", desc: "字符串", example: '"hello"' },
  { type: "int", desc: "整数", example: "42" },
  { type: "uint", desc: "无符号整数", example: "1" },
  { type: "int64", desc: "64 位整数", example: "1000000" },
  { type: "bool", desc: "布尔值", example: "true / false" },
  { type: "float", desc: "浮点数", example: "3.14" },
  { type: "array", desc: "数组", example: "[1, 2, 3]" },
  { type: "object", desc: "对象 / 键值对", example: '{"key": "value"}' },
  { type: "array[uint]", desc: "无符号整数数组", example: "[1, 2, 3]" },
  { type: "array[string]", desc: "字符串数组（方括号风格）", example: '["a", "b"]' },
  { type: "array<int>", desc: "整数数组（尖括号风格）", example: "[1, 2, 3]" },
  { type: "array<string>", desc: "字符串数组（尖括号风格）", example: '["a", "b"]' },
  { type: "string[]", desc: "字符串数组（Query 多值参数）", example: '["a", "b"]' },
  { type: "file", desc: "文件上传", example: "multipart/form-data 文件字段" },
  { type: "SSE", desc: "Server-Sent Events 事件流", example: "event: message\\ndata: {...}" },
];

// ────────────────────────────────────────
// 动态填充导航树中"接口文档"的子节点
// 每个分类节点下展开显示具体接口
// ────────────────────────────────────────

const apiGroup = navTree.find((g) => g.id === "api");
if (apiGroup) {
  apiGroup.children = overviewSections.map((section) => ({
    id: section.sectionId,
    label: section.title,
    keywords: section.entries.map((entry) => entry.description),
    children: section.entries.map((entry) => ({
      id: endpointNavId(section.sectionId, entry.name),
      label: entry.name,
      keywords: [entry.description],
    })),
  }));
}
