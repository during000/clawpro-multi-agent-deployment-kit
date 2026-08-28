/**
 * VersionManagement · Mock Data
 *
 * 支持三视角的完整数据：
 *   1. 实例视角：每台实例装了什么资产（版本）
 *   2. 分布视角：每个资产装在了哪些实例上（各是什么版本）
 *   3. 审计视角：谁在什么时候做了什么（变更历史）
 */

export type AgentTypeKey = "OpenClaw" | "Hermes" | "LightclawACE" | "MyAgent";
export type AgentKernel = "openclaw" | "hermes" | "lightclawace" | "native";

export const AGENT_TYPE_LABEL: Record<AgentTypeKey, string> = {
  OpenClaw: "OpenClaw",
  Hermes: "Hermes Agent",
  LightclawACE: "LightClaw ACE",
  MyAgent: "MyAgent (自研)",
};

// ────────────────────────────────────────────────────────────────
// 库资产定义：企业技能库 / 插件库 / MCP 库 的最新版
// ────────────────────────────────────────────────────────────────
export interface LibraryAsset {
  id: string;              // 唯一标识
  name: string;            // 展示名
  kind: "skill" | "plugin" | "mcp";
  latestVersion: string;   // 库中最新版
  description: string;
  category?: string;       // 分类
  updatedAt: string;
}

export const LIBRARY_SKILLS: LibraryAsset[] = [
  { id: "data-analysis", name: "数据分析", kind: "skill", latestVersion: "2.2.0", description: "Excel / CSV 数据分析与可视化", category: "数据处理", updatedAt: "2026-05-02" },
  { id: "web-search", name: "网络搜索", kind: "skill", latestVersion: "2.0.0", description: "web_tools + 网页正文提取", category: "通用办公", updatedAt: "2026-04-28" },
  { id: "pdf-ops", name: "PDF 处理", kind: "skill", latestVersion: "1.3.1", description: "PDF 读取、表单填写、转换", category: "通用办公", updatedAt: "2026-04-15" },
  { id: "pptx-ops", name: "PPT 生成", kind: "skill", latestVersion: "1.1.0", description: "PPT 创建、编辑、模板", category: "通用办公", updatedAt: "2026-03-20" },
  { id: "brainstorming", name: "头脑风暴", kind: "skill", latestVersion: "1.0.2", description: "产品创意工作流", category: "通用办公", updatedAt: "2026-02-10" },
  { id: "internal-docs", name: "内部文档助手", kind: "skill", latestVersion: "3.0.0", description: "公司内部文档搜索与问答", category: "内部工具", updatedAt: "2026-05-01" },
];

export const LIBRARY_PLUGINS: LibraryAsset[] = [
  { id: "plugin-feishu-bot", name: "飞书 Bot 增强", kind: "plugin", latestVersion: "1.4.0", description: "飞书机器人自定义能力扩展", category: "通讯", updatedAt: "2026-04-30" },
  { id: "plugin-company-crm", name: "内部 CRM", kind: "plugin", latestVersion: "2.1.0", description: "接入内部 CRM 的客户查询插件", category: "业务系统", updatedAt: "2026-04-22" },
  { id: "plugin-metrics", name: "指标采集", kind: "plugin", latestVersion: "0.5.0", description: "Agent 使用指标采集上报", category: "运维", updatedAt: "2026-03-15" },
];

export const LIBRARY_MCPS: LibraryAsset[] = [
  { id: "mcp-internal-crm", name: "内部 CRM MCP", kind: "mcp", latestVersion: "2026-05-01", description: "通过 MCP 协议访问内部 CRM", category: "业务系统", updatedAt: "2026-05-01" },
  { id: "mcp-ci-cd", name: "CI/CD 工具", kind: "mcp", latestVersion: "2026-04-20", description: "接入公司 CI/CD 系统", category: "研发工具", updatedAt: "2026-04-20" },
  { id: "mcp-github", name: "GitHub", kind: "mcp", latestVersion: "2026-04-01", description: "GitHub 操作（官方）", category: "研发工具", updatedAt: "2026-04-01" },
  { id: "mcp-clickhouse", name: "ClickHouse", kind: "mcp", latestVersion: "2026-03-20", description: "ClickHouse 数据查询", category: "数据处理", updatedAt: "2026-03-20" },
];

// ────────────────────────────────────────────────────────────────
// Agent 版本库
// ────────────────────────────────────────────────────────────────
export interface AgentVersionInfo {
  agentType: AgentTypeKey;
  version: string;
  releaseTime: string;
  isLatest: boolean;
  isActive: boolean;       // 当前管控端生效版本
  description: string;
}

export const AGENT_VERSIONS: AgentVersionInfo[] = [
  { agentType: "OpenClaw", version: "2026.4.26", releaseTime: "2026-04-26", isLatest: true, isActive: false, description: "升级 memory-tencentdb 插件到 0.2.3" },
  { agentType: "OpenClaw", version: "2026.4.21", releaseTime: "2026-04-21", isLatest: false, isActive: true, description: "Agent 版本 2026.4.21（当前生效）" },
  { agentType: "OpenClaw", version: "2026.4.15", releaseTime: "2026-04-15", isLatest: false, isActive: false, description: "Node 运行时升级到 22.22" },
  { agentType: "OpenClaw", version: "2026.4.2", releaseTime: "2026-04-02", isLatest: false, isActive: false, description: "接入 Mimo 模型" },
  { agentType: "OpenClaw", version: "2026.3.28", releaseTime: "2026-03-28", isLatest: false, isActive: false, description: "鉴权链路重构" },
  { agentType: "OpenClaw", version: "2026.3.13", releaseTime: "2026-03-13", isLatest: false, isActive: false, description: "插件子系统初版上线" },
  { agentType: "OpenClaw", version: "2026.3.8", releaseTime: "2026-03-08", isLatest: false, isActive: false, description: "首次上线，提供基础 Agent 能力" },
  { agentType: "Hermes", version: "0.10.0", releaseTime: "2026-04-16", isLatest: true, isActive: true, description: "Hermes 0.10.0 增强插件子系统" },
  { agentType: "Hermes", version: "0.9.1", releaseTime: "2026-03-25", isLatest: false, isActive: false, description: "Gateway 启动修复" },
  { agentType: "LightclawACE", version: "1.0.2", releaseTime: "2026-04-08", isLatest: true, isActive: true, description: "多租户隔离修复" },
  { agentType: "LightclawACE", version: "0.9.0", releaseTime: "2026-03-20", isLatest: false, isActive: false, description: "LightClaw ACE 初始版本" },
  { agentType: "LightclawACE", version: "0.8.5", releaseTime: "2026-02-15", isLatest: false, isActive: false, description: "早期内部测试版" },
  { agentType: "MyAgent", version: "1.0.0", releaseTime: "2026-05-01", isLatest: true, isActive: true, description: "自研 Agent 初版" },
];

// ────────────────────────────────────────────────────────────────
// 实例数据：每台实例装了什么 & 采集状态
// ────────────────────────────────────────────────────────────────
export type InstanceStatus = "running" | "shutdown" | "upgrading" | "createFail";
export type CollectStatus = "ok" | "stale" | "never" | "unsupported"; // 采集状态

export interface InstalledAsset {
  id: string;           // 对应 LibraryAsset.id
  name: string;
  installedVersion: string;
  libraryVersion: string;  // 库里最新版，便于对比
}

export interface AgentInstance {
  instanceId: string;
  name: string;
  agentType: AgentTypeKey;
  agentKernel: AgentKernel;
  agentVersion: string;
  owner: string;
  tags: { key: string; value: string }[];
  status: InstanceStatus;
  createdAt: string;
  lastCollectedAt: string;      // 上次采集时间
  collectStatus: CollectStatus; // 采集状态
  skills: InstalledAsset[];
  plugins: InstalledAsset[];
  mcps: InstalledAsset[];
}

// 帮助函数：根据 id 从库里查 latestVersion
const libMap = new Map<string, string>();
[...LIBRARY_SKILLS, ...LIBRARY_PLUGINS, ...LIBRARY_MCPS].forEach((a) => libMap.set(a.id, a.latestVersion));

function mkAsset(id: string, name: string, installedVersion: string): InstalledAsset {
  return { id, name, installedVersion, libraryVersion: libMap.get(id) ?? installedVersion };
}

export const MOCK_INSTANCES: AgentInstance[] = [
  {
    instanceId: "ins-g71c6vud",
    name: "Alice的技术助手",
    agentType: "OpenClaw",
    agentKernel: "openclaw",
    agentVersion: "2026.4.21",
    owner: "alice@acompany.com",
    tags: [{ key: "env", value: "production" }, { key: "所属产品", value: "gpulab" }],
    status: "running",
    createdAt: "2025-12-01 09:12:34",
    lastCollectedAt: "2026-05-06 13:45:12",
    collectStatus: "ok",
    skills: [
      mkAsset("data-analysis", "数据分析", "2.1.0"),     // 落后
      mkAsset("web-search", "网络搜索", "2.0.0"),          // 最新
      mkAsset("pdf-ops", "PDF 处理", "1.3.1"),             // 最新
      mkAsset("brainstorming", "头脑风暴", "1.0.2"),       // 最新
    ],
    plugins: [mkAsset("plugin-feishu-bot", "飞书 Bot 增强", "1.3.0")],
    mcps: [mkAsset("mcp-internal-crm", "内部 CRM MCP", "2026-05-01")],
  },
  {
    instanceId: "ins-h92d7xwe",
    name: "Bob工作助手",
    agentType: "OpenClaw",
    agentKernel: "openclaw",
    agentVersion: "2026.4.21",
    owner: "bob@acompany.com",
    tags: [],
    status: "running",
    createdAt: "2025-12-15 14:05:22",
    lastCollectedAt: "2026-05-06 13:30:00",
    collectStatus: "ok",
    skills: [
      mkAsset("data-analysis", "数据分析", "2.2.0"),
      mkAsset("web-search", "网络搜索", "1.8.0"),          // 落后严重
    ],
    plugins: [],
    mcps: [
      mkAsset("mcp-github", "GitHub", "2026-04-01"),
      mkAsset("mcp-clickhouse", "ClickHouse", "2026-03-20"),
    ],
  },
  {
    instanceId: "ins-k25f9zwg",
    name: "Dave的代码助手",
    agentType: "OpenClaw",
    agentKernel: "openclaw",
    agentVersion: "2026.4.15",        // 旧版 Agent
    owner: "dave@acompany.com",
    tags: [{ key: "test", value: "test2" }],
    status: "running",
    createdAt: "2026-01-20 16:48:09",
    lastCollectedAt: "2026-05-06 11:00:00",
    collectStatus: "ok",
    skills: [mkAsset("web-search", "网络搜索", "2.0.0")],
    plugins: [mkAsset("plugin-company-crm", "内部 CRM", "2.0.0")],
    mcps: [mkAsset("mcp-internal-crm", "内部 CRM MCP", "2026-04-15")],
  },
  {
    instanceId: "ins-m47h1byi",
    name: "Frank的数据助手",
    agentType: "OpenClaw",
    agentKernel: "openclaw",
    agentVersion: "2026.4.2",         // 更旧
    owner: "frank@acompany.com",
    tags: [],
    status: "running",
    createdAt: "2026-02-18 11:07:30",
    lastCollectedAt: "2026-05-06 13:12:00",
    collectStatus: "ok",
    skills: [
      mkAsset("data-analysis", "数据分析", "2.0.0"),
      mkAsset("pptx-ops", "PPT 生成", "1.0.0"),
    ],
    plugins: [mkAsset("plugin-metrics", "指标采集", "0.5.0")],
    mcps: [mkAsset("mcp-clickhouse", "ClickHouse", "2026-02-01")],
  },
  {
    instanceId: "ins-o69j3dak",
    name: "Henry的销售助手",
    agentType: "OpenClaw",
    agentKernel: "openclaw",
    agentVersion: "2026.4.21",
    owner: "henry@acompany.com",
    tags: [{ key: "team", value: "sales" }, { key: "env", value: "staging" }],
    status: "running",
    createdAt: "2026-03-01 09:58:03",
    lastCollectedAt: "2026-05-06 10:20:00",
    collectStatus: "stale",           // 采集数据有点旧
    skills: [
      mkAsset("data-analysis", "数据分析", "2.2.0"),
      mkAsset("internal-docs", "内部文档助手", "3.0.0"),
    ],
    plugins: [mkAsset("plugin-company-crm", "内部 CRM", "2.1.0")],
    mcps: [mkAsset("mcp-internal-crm", "内部 CRM MCP", "2026-05-01")],
  },
  {
    instanceId: "ins-q81l5fcm",
    name: "Jack的会议助手",
    agentType: "OpenClaw",
    agentKernel: "openclaw",
    agentVersion: "2026.3.28",        // 非常旧
    owner: "jack@acompany.com",
    tags: [],
    status: "running",
    createdAt: "2026-03-08 17:02:15",
    lastCollectedAt: "2026-05-05 22:00:00",
    collectStatus: "stale",
    skills: [mkAsset("pptx-ops", "PPT 生成", "0.9.0")],
    plugins: [],
    mcps: [],
  },
  {
    instanceId: "ins-r92m6gdn",
    name: "Karen的报告助手",
    agentType: "LightclawACE",
    agentKernel: "lightclawace",
    agentVersion: "1.0.2",
    owner: "karen@acompany.com",
    tags: [],
    status: "running",
    createdAt: "2026-03-09 10:15:50",
    lastCollectedAt: "2026-05-06 13:00:00",
    collectStatus: "ok",
    skills: [mkAsset("data-analysis", "数据分析", "2.2.0")],
    plugins: [],
    mcps: [mkAsset("mcp-ci-cd", "CI/CD 工具", "2026-04-20")],
  },
  {
    instanceId: "ins-p70k4ebl",
    name: "Ivy的客服助手",
    agentType: "Hermes",
    agentKernel: "hermes",
    agentVersion: "0.10.0",
    owner: "ivy@acompany.com",
    tags: [],
    status: "running",
    createdAt: "2026-03-05 13:26:41",
    lastCollectedAt: "2026-05-06 13:50:00",
    collectStatus: "ok",
    skills: [
      mkAsset("data-analysis", "数据分析", "2.2.0"),
      mkAsset("web-search", "网络搜索", "2.0.0"),
      mkAsset("pdf-ops", "PDF 处理", "1.3.1"),
    ],
    plugins: [],
    mcps: [mkAsset("mcp-github", "GitHub", "2026-04-01")],
  },
  {
    instanceId: "ins-t14o8ipf",
    name: "Mia的新助手",
    agentType: "Hermes",
    agentKernel: "hermes",
    agentVersion: "0.9.1",            // 旧 Hermes
    owner: "mia@acompany.com",
    tags: [],
    status: "running",
    createdAt: "2026-03-12 11:00:00",
    lastCollectedAt: "2026-05-06 12:00:00",
    collectStatus: "ok",
    skills: [],
    plugins: [],
    mcps: [],
  },
  {
    instanceId: "ins-u25p9jqg",
    name: "Noah的自研助手",
    agentType: "MyAgent",
    agentKernel: "native",            // 自研内核
    agentVersion: "1.0.0",
    owner: "noah@acompany.com",
    tags: [{ key: "project", value: "internal-ai" }],
    status: "running",
    createdAt: "2026-04-10 09:00:00",
    lastCollectedAt: "",
    collectStatus: "unsupported",    // 客户未提供 collect_status 脚本
    skills: [],
    plugins: [],
    mcps: [],
  },
  {
    instanceId: "ins-v36q0krh",
    name: "Olivia的运营助手",
    agentType: "OpenClaw",
    agentKernel: "openclaw",
    agentVersion: "2026.4.21",
    owner: "olivia@acompany.com",
    tags: [],
    status: "running",
    createdAt: "2026-03-14 09:00:00",
    lastCollectedAt: "2026-05-06 09:30:00",
    collectStatus: "ok",
    skills: [mkAsset("internal-docs", "内部文档助手", "2.8.0")],
    plugins: [mkAsset("plugin-feishu-bot", "飞书 Bot 增强", "1.4.0")],
    mcps: [],
  },
  {
    instanceId: "ins-w47r1lsi",
    name: "Peter的财务助手",
    agentType: "OpenClaw",
    agentKernel: "openclaw",
    agentVersion: "2026.4.2",
    owner: "peter@acompany.com",
    tags: [],
    status: "upgrading",              // 正在升级中
    createdAt: "2026-03-15 10:20:00",
    lastCollectedAt: "2026-05-06 08:00:00",
    collectStatus: "ok",
    skills: [mkAsset("data-analysis", "数据分析", "2.0.0")],
    plugins: [],
    mcps: [],
  },
];

// ────────────────────────────────────────────────────────────────
// 升级策略（自动升级）
// ────────────────────────────────────────────────────────────────
export type UpgradeFrequency = "daily" | "weekly" | "monthly";

export interface UpgradePolicy {
  agentType: AgentTypeKey;
  enabled: boolean;
  frequency: UpgradeFrequency;
  dayOfWeek?: number;             // 0-6（周日~周六），仅 weekly 用
  dayOfMonth?: number;            // 1-31，仅 monthly 用
  hour: string;                   // HH:00
  targetVersion: "latest-stable" | "latest-beta" | string;
  autoRollbackOnFailure: boolean;
  nextRun?: string;
  lastRun?: string;
}

export const MOCK_POLICIES: UpgradePolicy[] = [
  { agentType: "OpenClaw", enabled: true, frequency: "weekly", dayOfWeek: 0, hour: "02:00", targetVersion: "latest-stable", autoRollbackOnFailure: true, nextRun: "2026-05-10 02:00", lastRun: "2026-05-03 02:00" },
  { agentType: "Hermes", enabled: false, frequency: "weekly", dayOfWeek: 0, hour: "03:00", targetVersion: "latest-stable", autoRollbackOnFailure: true },
  { agentType: "LightclawACE", enabled: false, frequency: "weekly", dayOfWeek: 0, hour: "03:00", targetVersion: "latest-stable", autoRollbackOnFailure: true },
  { agentType: "MyAgent", enabled: false, frequency: "weekly", dayOfWeek: 0, hour: "03:00", targetVersion: "latest-stable", autoRollbackOnFailure: true },
];

// ────────────────────────────────────────────────────────────────
// 历史记录（运维任务审计 + 每台实例的变更历史）
//
// 「运维任务」记录所有需要在 Agent 实例上批量执行的运维动作：
//   - agent-upgrade：Agent 自身的版本升级（含降级 / 回退到更早版本）
//   - command-execute：管理员通过「命令下发」执行的 Shell 命令任务
//
// 插件 / MCP / 技能的下发、更新、卸载已迁移到「Agent 工具库」对应资产详情页，
// 不在这里记录，避免运维任务列表被大量轻动作淹没。
// ────────────────────────────────────────────────────────────────
export type HistoryAction =
  | "agent-upgrade"
  | "command-execute";

export const HISTORY_ACTION_LABEL: Record<HistoryAction, string> = {
  "agent-upgrade": "Agent 更新",
  "command-execute": "命令执行",
};

// 命令执行的额外字段（仅 action === "command-execute" 时存在）
export interface CommandExecuteExtra {
  commandId: string;          // 关联的命令模板 ID
  commandName: string;        // 命令名称
  commandType: "SHELL";       // 暂只支持 Linux Shell
  commandContent: string;     // 命令内容（截断展示）
  workingDir: string;         // 执行路径
  runAsUser: string;          // 执行用户
  timeoutSec: number;         // 超时时间（秒）
  // 参数化命令
  commandContentTemplate?: string;  // 命令模板（含 {{param}} 占位符）
  paramValues?: Record<string, string>; // 参数值
  // 灰度信息
  testInstanceId?: string;    // 灰度执行：先在 1 台实例（灰度机）上执行
  testStatus?: "success" | "failed";
  testMessage?: string;
}

export type ScheduledTaskStatus =
  | "pending"
  | "waiting"
  | "running"
  | "completed"
  | "paused"
  | "canceled"
  | "failed";

export type ScheduledFrequency = "once" | "minutes" | "hourly" | "daily" | "weekly" | "monthly";

export interface ScheduledTaskExtra {
  taskName: string;
  remark?: string;
  executeAt: string;
  firstExecuteAt: string;
  frequency: ScheduledFrequency;
  intervalMinutes?: number;
  intervalHours?: number;
  status: ScheduledTaskStatus;
  nextExecuteAt?: string;
  createdBy: string;
  createdAt: string;
}

export interface HistoryRecord {
  id: string;
  taskId: string;              // 任务 ID，用于跟踪
  action: HistoryAction;
  assetName: string;         // 动作作用的资产名（Agent 版本号 / 命令名 / ...）
  fromVersion?: string;
  toVersion?: string;
  operator: string;          // 谁干的（人名 / "自动升级"）
  isAuto: boolean;           // 是否自动升级触发
  scheduledAt?: string;      // 定时任务的计划执行时间（仅定时任务有值）
  operatedAt: string;        // 实际执行时间
  scheduledTask?: ScheduledTaskExtra;
  totalInstances: number;
  successCount: number;
  failedCount: number;
  // 命令执行专用扩展字段
  commandExtra?: CommandExecuteExtra;
  // 每台实例的结果（详情用）
  perInstanceResult?: {
    instanceId: string;
    instanceName: string;
    status: "success" | "failed" | "running";
    message?: string;
    // 命令执行场景下，单台机器的 stdout/stderr/exitCode
    stdout?: string;
    stderr?: string;
    exitCode?: number;
    durationMs?: number;
  }[];
}

function createDiskUsageResults(total: number, failedIndexes: number[] = []): HistoryRecord["perInstanceResult"] {
  return Array.from({ length: total }, (_, index) => {
    const no = index + 1;
    const failed = failedIndexes.includes(no);
    const instanceId = `ins-disk-${String(no).padStart(2, "0")}`;
    const instanceName = `磁盘巡检节点-${String(no).padStart(2, "0")}`;

    if (failed) {
      return {
        instanceId,
        instanceName,
        status: "failed",
        stderr: no % 2 === 0 ? "df: /data: input/output error" : "command timeout after 30s",
        exitCode: 1,
        durationMs: 30000,
      };
    }

    return {
      instanceId,
      instanceName,
      status: "success",
      stdout: `/dev/vda1 ${40 + (no % 18)}%\n/data ${58 + (no % 24)}%`,
      exitCode: 0,
      durationMs: 1200 + no * 35,
    };
  });
}

export const MOCK_HISTORY: HistoryRecord[] = [
  // ─── 命令定时任务示例 ──────────────────────
  {
    id: "h-scheduled-command-001",
    taskId: "TASK-20260618-3001",
    action: "command-execute",
    assetName: "定时清理临时日志",
    operator: "admin@acompany.com",
    isAuto: false,
    scheduledAt: "2026-06-18 09:30",
    operatedAt: "—（未开始）",
    scheduledTask: {
      taskName: "定时清理临时日志",
      remark: "每日上班前清理 OpenClaw Agent 临时日志",
      executeAt: "2026-06-18 09:30",
      firstExecuteAt: "2026-06-18 09:30",
      frequency: "daily",
      status: "pending",
      nextExecuteAt: "2026-06-18 09:30",
      createdBy: "admin@acompany.com",
      createdAt: "2026-06-17 15:20:00",
    },
    totalInstances: 12,
    successCount: 0,
    failedCount: 0,
    commandExtra: {
      commandId: "cmd-cleanup-logs",
      commandName: "清理临时日志",
      commandType: "SHELL",
      commandContent: "find /tmp -name '*.log' -mtime +7 -delete && echo done",
      workingDir: "/root",
      runAsUser: "root",
      timeoutSec: 60,
    },
  },
  {
    id: "h-scheduled-command-002",
    taskId: "TASK-20260610-3002",
    action: "command-execute",
    assetName: "周例行磁盘巡检",
    operator: "ops@acompany.com",
    isAuto: false,
    scheduledAt: "2026-06-10 22:00",
    operatedAt: "—（待执行）",
    scheduledTask: {
      taskName: "周例行磁盘巡检",
      executeAt: "2026-06-10 22:00",
      firstExecuteAt: "2026-06-10 22:00",
      frequency: "weekly",
      status: "waiting",
      nextExecuteAt: "2026-06-24 22:00",
      createdBy: "ops@acompany.com",
      createdAt: "2026-06-09 18:02:00",
    },
    totalInstances: 45,
    successCount: 0,
    failedCount: 0,
    commandExtra: {
      commandId: "cmd-disk-usage",
      commandName: "检查磁盘使用率",
      commandType: "SHELL",
      commandContent: "df -h | grep -v tmpfs | awk 'NR>1 {print $1\" \"$5}'",
      workingDir: "/root",
      runAsUser: "root",
      timeoutSec: 30,
    },
  },
  {
    id: "h-scheduled-command-002-run-002",
    taskId: "TASK-20260610-3002-RUN-002",
    action: "command-execute",
    assetName: "周例行磁盘巡检",
    operator: "ops@acompany.com",
    isAuto: false,
    scheduledAt: "2026-06-17 22:00",
    operatedAt: "2026-06-17 22:00:03",
    scheduledTask: {
      taskName: "周例行磁盘巡检",
      executeAt: "2026-06-17 22:00",
      firstExecuteAt: "2026-06-10 22:00",
      frequency: "weekly",
      status: "failed",
      createdBy: "ops@acompany.com",
      createdAt: "2026-06-09 18:02:00",
    },
    totalInstances: 45,
    successCount: 43,
    failedCount: 2,
    commandExtra: {
      commandId: "cmd-disk-usage",
      commandName: "检查磁盘使用率",
      commandType: "SHELL",
      commandContent: "df -h | grep -v tmpfs | awk 'NR>1 {print $1\" \"$5}'",
      workingDir: "/root",
      runAsUser: "root",
      timeoutSec: 30,
    },
    perInstanceResult: createDiskUsageResults(45, [17, 32]),
  },
  {
    id: "h-scheduled-command-002-run-000",
    taskId: "TASK-20260610-3002-RUN-001",
    action: "command-execute",
    assetName: "周例行磁盘巡检",
    operator: "ops@acompany.com",
    isAuto: false,
    scheduledAt: "2026-06-10 22:00",
    operatedAt: "2026-06-10 22:00:02",
    scheduledTask: {
      taskName: "周例行磁盘巡检",
      executeAt: "2026-06-10 22:00",
      firstExecuteAt: "2026-06-10 22:00",
      frequency: "weekly",
      status: "completed",
      createdBy: "ops@acompany.com",
      createdAt: "2026-06-09 18:02:00",
    },
    totalInstances: 45,
    successCount: 45,
    failedCount: 0,
    commandExtra: {
      commandId: "cmd-disk-usage",
      commandName: "检查磁盘使用率",
      commandType: "SHELL",
      commandContent: "df -h | grep -v tmpfs | awk 'NR>1 {print $1\" \"$5}'",
      workingDir: "/root",
      runAsUser: "root",
      timeoutSec: 30,
    },
    perInstanceResult: createDiskUsageResults(45),
  },
  {
    id: "h-scheduled-command-003",
    taskId: "TASK-20260615-3003",
    action: "command-execute",
    assetName: "hosts 刷新",
    operator: "admin@acompany.com",
    isAuto: false,
    scheduledAt: "2026-06-20 02:00",
    operatedAt: "2026-06-20 02:00:04",
    scheduledTask: {
      taskName: "hosts 刷新",
      remark: "单次定时任务已触发完成，任务生命周期结束",
      executeAt: "2026-06-20 02:00",
      firstExecuteAt: "2026-06-20 02:00",
      frequency: "once",
      status: "completed",
      createdBy: "admin@acompany.com",
      createdAt: "2026-06-15 10:12:00",
    },
    totalInstances: 8,
    successCount: 8,
    failedCount: 0,
    commandExtra: {
      commandId: "cmd-refresh-hosts",
      commandName: "刷新 hosts 配置",
      commandType: "SHELL",
      commandContent: "cat >> /etc/hosts <<EOF\n10.0.0.10 internal-llm.local\nEOF\nsystemctl restart nscd || true",
      workingDir: "/root",
      runAsUser: "root",
      timeoutSec: 60,
    },
  },
  {
    id: "h-scheduled-command-004",
    taskId: "TASK-20260618-3004",
    action: "command-execute",
    assetName: "批量刷新软件源",
    operator: "ops@acompany.com",
    isAuto: false,
    scheduledAt: "2026-06-18 10:00",
    operatedAt: "—（执行中）",
    scheduledTask: {
      taskName: "批量刷新软件源",
      remark: "灰度后发现部分节点耗时过长，可在管理页取消本次任务",
      executeAt: "2026-06-18 10:00",
      firstExecuteAt: "2026-06-18 10:00",
      frequency: "once",
      status: "running",
      createdBy: "ops@acompany.com",
      createdAt: "2026-06-18 09:50:00",
    },
    totalInstances: 18,
    successCount: 6,
    failedCount: 0,
    commandExtra: {
      commandId: "cmd-update-pkg-index",
      commandName: "更新软件包索引",
      commandType: "SHELL",
      commandContent: "if command -v apt-get >/dev/null; then apt-get update -y; else yum makecache -y; fi",
      workingDir: "/root",
      runAsUser: "root",
      timeoutSec: 120,
    },
  },
  {
    id: "h-scheduled-command-005",
    taskId: "TASK-20260612-3005",
    action: "command-execute",
    assetName: "每月安全基线扫描",
    operator: "secops@acompany.com",
    isAuto: false,
    scheduledAt: "2026-06-12 01:00",
    operatedAt: "—（已暂停）",
    scheduledTask: {
      taskName: "每月安全基线扫描",
      remark: "等待安全策略模板更新，暂时暂停后续调度",
      executeAt: "2026-06-12 01:00",
      firstExecuteAt: "2026-06-12 01:00",
      frequency: "monthly",
      status: "paused",
      nextExecuteAt: "2026-07-12 01:00",
      createdBy: "secops@acompany.com",
      createdAt: "2026-06-11 16:18:00",
    },
    totalInstances: 32,
    successCount: 0,
    failedCount: 0,
    commandExtra: {
      commandId: "cmd-security-baseline",
      commandName: "安全基线扫描",
      commandType: "SHELL",
      commandContent: "/opt/openclaw/bin/security-scan --baseline cis --format json",
      workingDir: "/opt/openclaw",
      runAsUser: "root",
      timeoutSec: 300,
    },
  },
  {
    id: "h-scheduled-command-006",
    taskId: "TASK-20260614-3006",
    action: "command-execute",
    assetName: "每日 Agent 心跳巡检",
    operator: "sre@acompany.com",
    isAuto: false,
    scheduledAt: "2026-06-14 08:00",
    operatedAt: "—（待执行）",
    scheduledTask: {
      taskName: "每日 Agent 心跳巡检",
      remark: "每天早上巡检核心 Agent 进程和端口状态",
      executeAt: "2026-06-14 08:00",
      firstExecuteAt: "2026-06-14 08:00",
      frequency: "daily",
      status: "waiting",
      nextExecuteAt: "2026-06-24 08:00",
      createdBy: "sre@acompany.com",
      createdAt: "2026-06-13 20:36:00",
    },
    totalInstances: 64,
    successCount: 0,
    failedCount: 0,
    commandExtra: {
      commandId: "cmd-agent-healthcheck",
      commandName: "Agent 心跳巡检",
      commandType: "SHELL",
      commandContent: "systemctl is-active openclaw-agent && ss -lntp | grep 8080",
      workingDir: "/root",
      runAsUser: "root",
      timeoutSec: 45,
    },
  },
  {
    id: "h-scheduled-command-007",
    taskId: "TASK-20260624-3007",
    action: "command-execute",
    assetName: "每 15 分钟心跳探测",
    operator: "sre@acompany.com",
    isAuto: false,
    scheduledAt: "2026-06-24 10:15",
    operatedAt: "—（待执行）",
    scheduledTask: {
      taskName: "每 15 分钟心跳探测",
      remark: "高频巡检本地 Agent 心跳，用于观察分钟级调度能力",
      executeAt: "2026-06-24 10:15",
      firstExecuteAt: "2026-06-24 10:00",
      frequency: "minutes",
      intervalMinutes: 15,
      status: "waiting",
      nextExecuteAt: "2026-06-24 10:15",
      createdBy: "sre@acompany.com",
      createdAt: "2026-06-24 09:45:00",
    },
    totalInstances: 18,
    successCount: 0,
    failedCount: 0,
    commandExtra: {
      commandId: "cmd-agent-heartbeat",
      commandName: "Agent 心跳探测",
      commandType: "SHELL",
      commandContent: "curl -sf http://127.0.0.1:8080/health || systemctl status openclaw-agent --no-pager",
      workingDir: "/root",
      runAsUser: "root",
      timeoutSec: 30,
    },
  },
  {
    id: "h-scheduled-command-008",
    taskId: "TASK-20260624-3008",
    action: "command-execute",
    assetName: "每 2 小时状态采集",
    operator: "ops@acompany.com",
    isAuto: false,
    scheduledAt: "2026-06-24 12:00",
    operatedAt: "—（待执行）",
    scheduledTask: {
      taskName: "每 2 小时状态采集",
      remark: "每 2 小时采集 Agent 服务状态与系统负载",
      executeAt: "2026-06-24 12:00",
      firstExecuteAt: "2026-06-24 10:00",
      frequency: "hourly",
      intervalHours: 2,
      status: "waiting",
      nextExecuteAt: "2026-06-24 12:00",
      createdBy: "ops@acompany.com",
      createdAt: "2026-06-24 09:50:00",
    },
    totalInstances: 24,
    successCount: 0,
    failedCount: 0,
    commandExtra: {
      commandId: "cmd-show-load",
      commandName: "查看系统负载",
      commandType: "SHELL",
      commandContent: "uptime && systemctl is-active openclaw-agent",
      workingDir: "/root",
      runAsUser: "root",
      timeoutSec: 45,
    },
  },
  {
    id: "h-scheduled-command-009",
    taskId: "TASK-20260624-3009",
    action: "command-execute",
    assetName: "低峰期批量重启 Agent",
    operator: "ops@acompany.com",
    isAuto: false,
    scheduledAt: "2026-06-24 03:30",
    operatedAt: "—（未开始）",
    scheduledTask: {
      taskName: "低峰期批量重启 Agent",
      remark: "配合 Agent 配置更新，在低峰期执行一次性重启",
      executeAt: "2026-06-24 03:30",
      firstExecuteAt: "2026-06-24 03:30",
      frequency: "once",
      status: "pending",
      nextExecuteAt: "2026-06-24 03:30",
      createdBy: "ops@acompany.com",
      createdAt: "2026-06-23 11:24:00",
    },
    totalInstances: 27,
    successCount: 0,
    failedCount: 0,
    commandExtra: {
      commandId: "cmd-restart-agent",
      commandName: "重启 Agent 服务",
      commandType: "SHELL",
      commandContent: "systemctl restart openclaw-agent && systemctl status openclaw-agent --no-pager",
      workingDir: "/root",
      runAsUser: "root",
      timeoutSec: 90,
    },
  },
  {
    id: "h-scheduled-command-010",
    taskId: "TASK-20260601-3010",
    action: "command-execute",
    assetName: "月度容量采集",
    operator: "capacity@acompany.com",
    isAuto: false,
    scheduledAt: "2026-06-01 00:30",
    operatedAt: "—（待执行）",
    scheduledTask: {
      taskName: "月度容量采集",
      remark: "每月采集磁盘、CPU、内存容量数据，用于容量规划",
      executeAt: "2026-06-01 00:30",
      firstExecuteAt: "2026-06-01 00:30",
      frequency: "monthly",
      status: "waiting",
      nextExecuteAt: "2026-07-01 00:30",
      createdBy: "capacity@acompany.com",
      createdAt: "2026-05-31 17:45:00",
    },
    totalInstances: 96,
    successCount: 0,
    failedCount: 0,
    commandExtra: {
      commandId: "cmd-capacity-collect",
      commandName: "容量数据采集",
      commandType: "SHELL",
      commandContent: "printf 'cpu=' && nproc && free -m | awk '/Mem:/ {print \"mem=\"$2\"MB\"}' && df -h",
      workingDir: "/root",
      runAsUser: "root",
      timeoutSec: 120,
    },
  },
  // ─── 待执行的定时任务（用于「更新计划」Drawer 待执行区） ─────
  {
    id: "h-scheduled-001",
    taskId: "TASK-20260513-0001",
    action: "agent-upgrade",
    assetName: "OpenClaw",
    fromVersion: "2026.4.21",
    toVersion: "2026.4.26",
    operator: "admin@acompany.com",
    isAuto: false,
    scheduledAt: "2026-05-13 03:00:00",
    operatedAt: "—（待执行）",
    totalInstances: 32,
    successCount: 0,
    failedCount: 0,
  },
  {
    id: "h-scheduled-002",
    taskId: "TASK-20260515-0002",
    action: "agent-upgrade",
    assetName: "Hermes Agent",
    fromVersion: "0.9.1",
    toVersion: "0.10.0",
    operator: "自动升级",
    isAuto: true,
    scheduledAt: "2026-05-15 02:00:00",
    operatedAt: "—（待执行）",
    totalInstances: 8,
    successCount: 0,
    failedCount: 0,
  },
  // ─── 进行中的任务 ─────
  {
    id: "h-active-001",
    taskId: "TASK-20260511-0001",
    action: "agent-upgrade",
    assetName: "OpenClaw",
    fromVersion: "2026.4.15",
    toVersion: "2026.4.26",
    operator: "admin@acompany.com",
    isAuto: false,
    operatedAt: "2026-05-11 20:30:00",
    totalInstances: 6,
    successCount: 3,
    failedCount: 0,
    perInstanceResult: [
      { instanceId: "ins-g71c6vud", instanceName: "Alice的技术助手", status: "success" },
      { instanceId: "ins-h92d7xwe", instanceName: "Bob工作助手", status: "success" },
      { instanceId: "ins-k25f9zwg", instanceName: "Dave的代码助手", status: "success" },
      { instanceId: "ins-m47h1byi", instanceName: "Frank的数据助手", status: "running" },
      { instanceId: "ins-o69j3dak", instanceName: "Henry的销售助手", status: "running" },
      { instanceId: "ins-q81l5fcm", instanceName: "Jack的会议助手", status: "running" },
    ],
  },
  {
    id: "h-20260506-001",
    taskId: "TASK-20260506-0001",
    action: "agent-upgrade",
    assetName: "OpenClaw",
    fromVersion: "2026.4.21",
    toVersion: "2026.4.26",
    operator: "admin@acompany.com",
    isAuto: false,
    operatedAt: "2026-05-06 10:15:22",
    totalInstances: 45,
    successCount: 32,
    failedCount: 1,
    perInstanceResult: [
      { instanceId: "ins-g71c6vud", instanceName: "Alice的技术助手", status: "success" },
      { instanceId: "ins-h92d7xwe", instanceName: "Bob工作助手", status: "running" },
      { instanceId: "ins-k25f9zwg", instanceName: "Dave的代码助手", status: "failed", message: "pnpm 升级超时" },
    ],
  },
  {
    id: "h-20260505-002",
    taskId: "TASK-20260505-0002",
    action: "agent-upgrade",
    assetName: "Hermes Agent",
    fromVersion: "0.7.9",
    toVersion: "0.8.1",
    operator: "admin@acompany.com",
    isAuto: false,
    operatedAt: "2026-05-05 16:00:00",
    totalInstances: 24,
    successCount: 24,
    failedCount: 0,
  },
  {
    id: "h-20260503-003",
    taskId: "TASK-20260503-0003",
    action: "agent-upgrade",
    assetName: "OpenClaw",
    fromVersion: "2026.4.15",
    toVersion: "2026.4.21",
    operator: "自动升级",
    isAuto: true,
    scheduledAt: "2026-05-03 02:00:00",
    operatedAt: "2026-05-03 02:00:12",
    totalInstances: 120,
    successCount: 118,
    failedCount: 2,
  },
  {
    id: "h-20260502-004",
    taskId: "TASK-20260502-0004",
    action: "agent-upgrade",
    assetName: "Hermes Agent",
    fromVersion: "0.8.1",
    toVersion: "0.8.0",
    operator: "admin@acompany.com",
    isAuto: false,
    operatedAt: "2026-05-02 18:30:00",
    totalInstances: 8,
    successCount: 8,
    failedCount: 0,
  },
  {
    id: "h-20260430-005",
    taskId: "TASK-20260430-0005",
    action: "agent-upgrade",
    assetName: "LightclawACE",
    fromVersion: "1.2.0",
    toVersion: "1.3.0",
    operator: "自动升级",
    isAuto: true,
    scheduledAt: "2026-04-30 03:00:00",
    operatedAt: "2026-04-30 03:00:05",
    totalInstances: 36,
    successCount: 35,
    failedCount: 1,
  },
  {
    id: "h-20260428-006",
    taskId: "TASK-20260428-0006",
    action: "agent-upgrade",
    assetName: "MyAgent（自研）",
    fromVersion: "0.9.2",
    toVersion: "1.0.0",
    operator: "admin@acompany.com",
    isAuto: false,
    operatedAt: "2026-04-28 14:20:30",
    totalInstances: 15,
    successCount: 15,
    failedCount: 0,
  },
  {
    id: "h-20260425-007",
    taskId: "TASK-20260425-0007",
    action: "agent-upgrade",
    assetName: "OpenClaw",
    fromVersion: "2026.4.15",
    toVersion: "2026.4.10",
    operator: "admin@acompany.com",
    isAuto: false,
    operatedAt: "2026-04-25 11:05:00",
    totalInstances: 12,
    successCount: 11,
    failedCount: 1,
  },
  {
    id: "h-20260420-008",
    taskId: "TASK-20260420-0008",
    action: "agent-upgrade",
    assetName: "OpenClaw",
    fromVersion: "2026.4.08",
    toVersion: "2026.4.15",
    operator: "admin@acompany.com",
    isAuto: false,
    operatedAt: "2026-04-20 09:30:15",
    totalInstances: 62,
    successCount: 60,
    failedCount: 2,
  },
  // ─── 命令执行类型示例 ──────────────────────
  {
    id: "h-20260508-009",
    taskId: "TASK-20260508-0009",
    action: "command-execute",
    assetName: "清理临时日志",
    operator: "admin@acompany.com",
    isAuto: false,
    operatedAt: "2026-05-08 15:22:00",
    totalInstances: 12,
    successCount: 12,
    failedCount: 0,
    commandExtra: {
      commandId: "cmd-cleanup-logs",
      commandName: "清理临时日志",
      commandType: "SHELL",
      commandContent: "find /tmp -name '*.log' -mtime +7 -delete && echo done",
      workingDir: "/root",
      runAsUser: "root",
      timeoutSec: 60,
      testInstanceId: "ins-g71c6vud",
      testStatus: "success",
      testMessage: "灰度机执行成功，清理 8 个日志文件",
    },
    perInstanceResult: [
      {
        instanceId: "ins-g71c6vud",
        instanceName: "Alice的技术助手",
        status: "success",
        stdout: "done",
        exitCode: 0,
        durationMs: 1234,
      },
      {
        instanceId: "ins-h92d7xwe",
        instanceName: "Bob工作助手",
        status: "success",
        stdout: "done",
        exitCode: 0,
        durationMs: 980,
      },
    ],
  },
  {
    id: "h-20260507-010",
    taskId: "TASK-20260507-0010",
    action: "command-execute",
    assetName: "检查磁盘使用率",
    operator: "ops@acompany.com",
    isAuto: false,
    operatedAt: "2026-05-07 10:08:33",
    totalInstances: 45,
    successCount: 43,
    failedCount: 2,
    commandExtra: {
      commandId: "cmd-disk-usage",
      commandName: "检查磁盘使用率",
      commandType: "SHELL",
      commandContent: "df -h | grep -v tmpfs | awk 'NR>1 {print $1\" \"$5}'",
      workingDir: "/root",
      runAsUser: "root",
      timeoutSec: 30,
    },
  },
  {
    id: "h-20260506-011",
    taskId: "TASK-20260506-0011",
    action: "command-execute",
    assetName: "刷新 hosts 配置",
    operator: "admin@acompany.com",
    isAuto: false,
    scheduledAt: "2026-05-06 22:00:00",
    operatedAt: "2026-05-06 22:00:08",
    totalInstances: 8,
    successCount: 8,
    failedCount: 0,
    commandExtra: {
      commandId: "cmd-refresh-hosts",
      commandName: "刷新 hosts 配置",
      commandType: "SHELL",
      commandContent: "cat >> /etc/hosts <<EOF\n10.0.0.10 internal-llm.local\nEOF\nsystemctl restart nscd || true",
      workingDir: "/root",
      runAsUser: "root",
      timeoutSec: 60,
    },
  },
];

// 某一台实例的变更记录（实例详情页用）
export function getInstanceHistory(instanceId: string): HistoryRecord[] {
  return MOCK_HISTORY.filter((h) => h.perInstanceResult?.some((r) => r.instanceId === instanceId));
}

// ────────────────────────────────────────────────────────────────
// 工具函数：计算状态
// ────────────────────────────────────────────────────────────────

/** 判断实例的 Agent 版本是否需要升级（跟当前生效版本对比）*/
export function getAgentNeedUpgrade(inst: AgentInstance): boolean {
  const active = AGENT_VERSIONS.find((v) => v.agentType === inst.agentType && v.isActive);
  return !!active && active.version !== inst.agentVersion;
}

/** 判断单个资产是否过期 */
export function isAssetStale(asset: InstalledAsset): boolean {
  return asset.installedVersion !== asset.libraryVersion;
}

// ────────────────────────────────────────────────────────────────
// Agent 运行时配置（模型 / 通道）—— 供 Agent 详情页展示
// 这部分是业务运行态数据，不属于版本管理范畴
// ────────────────────────────────────────────────────────────────
export interface InstanceRuntimeProfile {
  model: { name: string; version: string } | null;
  channels: { name: string; botCount: number }[];
}

// 为避免对每个实例都写一份配置，使用确定性的"伪随机"生成
const MODEL_POOL = [
  { name: "tencentcodingplan", version: "minimax-m2.5" },
  { name: "腾讯云混元",          version: "混元 Turbo" },
  { name: "腾讯云混元",          version: "混元 Pro" },
  { name: "腾讯云 DeepSeek",    version: "DeepSeek V3 0324" },
  { name: "tencentcodingplan", version: "deepseek-v3.2" },
];

const CHANNEL_PRESETS: { name: string; botCount: number }[][] = [
  [{ name: "飞书", botCount: 3 }, { name: "企业微信机器人", botCount: 1 }],
  [{ name: "企业微信", botCount: 2 }],
  [{ name: "飞书", botCount: 1 }],
  [{ name: "钉钉", botCount: 2 }, { name: "企业微信", botCount: 1 }],
  [], // 未接入
];

function hashStr(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0;
  return h;
}

export function getInstanceRuntimeProfile(instanceId: string): InstanceRuntimeProfile {
  const h = hashStr(instanceId);
  return {
    model: MODEL_POOL[h % MODEL_POOL.length],
    channels: CHANNEL_PRESETS[h % CHANNEL_PRESETS.length],
  };
}

/** 计算实例某一类资产的同步健康度：0-100 */
export function getAssetHealthScore(assets: InstalledAsset[]): number {
  if (assets.length === 0) return 100;
  const staleCount = assets.filter(isAssetStale).length;
  return Math.round(((assets.length - staleCount) / assets.length) * 100);
}

/** 计算某一个资产的实例分布（分布视角核心） */
export interface AssetDistribution {
  assetId: string;
  assetName: string;
  kind: "skill" | "plugin" | "mcp" | "agent";
  latestVersion: string;
  totalInstances: number;          // 装了该资产的实例总数
  versionSplit: {                   // 按版本统计
    version: string;
    count: number;
    isLatest: boolean;
    instanceIds: string[];
  }[];
}

export function getSkillDistribution(skillId: string): AssetDistribution {
  const lib = LIBRARY_SKILLS.find((s) => s.id === skillId)!;
  const installations = MOCK_INSTANCES
    .map((i) => ({ inst: i, asset: i.skills.find((s) => s.id === skillId) }))
    .filter((x) => x.asset);
  const versionMap = new Map<string, string[]>();
  installations.forEach((x) => {
    const v = x.asset!.installedVersion;
    const arr = versionMap.get(v) || [];
    arr.push(x.inst.instanceId);
    versionMap.set(v, arr);
  });
  return {
    assetId: skillId,
    assetName: lib.name,
    kind: "skill",
    latestVersion: lib.latestVersion,
    totalInstances: installations.length,
    versionSplit: Array.from(versionMap.entries())
      .map(([version, instanceIds]) => ({
        version,
        count: instanceIds.length,
        isLatest: version === lib.latestVersion,
        instanceIds,
      }))
      .sort((a, b) => (b.isLatest ? 1 : 0) - (a.isLatest ? 1 : 0) || b.count - a.count),
  };
}

export function getPluginDistribution(pluginId: string): AssetDistribution {
  const lib = LIBRARY_PLUGINS.find((s) => s.id === pluginId)!;
  const installations = MOCK_INSTANCES
    .map((i) => ({ inst: i, asset: i.plugins.find((s) => s.id === pluginId) }))
    .filter((x) => x.asset);
  const versionMap = new Map<string, string[]>();
  installations.forEach((x) => {
    const v = x.asset!.installedVersion;
    const arr = versionMap.get(v) || [];
    arr.push(x.inst.instanceId);
    versionMap.set(v, arr);
  });
  return {
    assetId: pluginId,
    assetName: lib.name,
    kind: "plugin",
    latestVersion: lib.latestVersion,
    totalInstances: installations.length,
    versionSplit: Array.from(versionMap.entries())
      .map(([version, instanceIds]) => ({
        version,
        count: instanceIds.length,
        isLatest: version === lib.latestVersion,
        instanceIds,
      }))
      .sort((a, b) => (b.isLatest ? 1 : 0) - (a.isLatest ? 1 : 0) || b.count - a.count),
  };
}

export function getMcpDistribution(mcpId: string): AssetDistribution {
  const lib = LIBRARY_MCPS.find((s) => s.id === mcpId)!;
  const installations = MOCK_INSTANCES
    .map((i) => ({ inst: i, asset: i.mcps.find((s) => s.id === mcpId) }))
    .filter((x) => x.asset);
  const versionMap = new Map<string, string[]>();
  installations.forEach((x) => {
    const v = x.asset!.installedVersion;
    const arr = versionMap.get(v) || [];
    arr.push(x.inst.instanceId);
    versionMap.set(v, arr);
  });
  return {
    assetId: mcpId,
    assetName: lib.name,
    kind: "mcp",
    latestVersion: lib.latestVersion,
    totalInstances: installations.length,
    versionSplit: Array.from(versionMap.entries())
      .map(([version, instanceIds]) => ({
        version,
        count: instanceIds.length,
        isLatest: version === lib.latestVersion,
        instanceIds,
      }))
      .sort((a, b) => (b.isLatest ? 1 : 0) - (a.isLatest ? 1 : 0) || b.count - a.count),
  };
}

/** Agent 版本分布（跟 Skill 类似但用 agentType + agentVersion） */
export function getAgentVersionDistribution(agentType: AgentTypeKey): AssetDistribution {
  const instances = MOCK_INSTANCES.filter((i) => i.agentType === agentType);
  const latestActive = AGENT_VERSIONS.find((v) => v.agentType === agentType && v.isActive);
  const versionMap = new Map<string, string[]>();
  instances.forEach((inst) => {
    const arr = versionMap.get(inst.agentVersion) || [];
    arr.push(inst.instanceId);
    versionMap.set(inst.agentVersion, arr);
  });
  return {
    assetId: agentType,
    assetName: AGENT_TYPE_LABEL[agentType],
    kind: "agent",
    latestVersion: latestActive?.version ?? "",
    totalInstances: instances.length,
    versionSplit: Array.from(versionMap.entries())
      .map(([version, instanceIds]) => ({
        version,
        count: instanceIds.length,
        isLatest: version === latestActive?.version,
        instanceIds,
      }))
      .sort((a, b) => (b.isLatest ? 1 : 0) - (a.isLatest ? 1 : 0) || b.count - a.count),
  };
}

// ────────────────────────────────────────────────────────────────
// 预选实例的 sessionStorage key
// ────────────────────────────────────────────────────────────────
export const PRESELECT_INSTANCES_KEY = "version-management:preselect-instances";

// ────────────────────────────────────────────────────────────────
// 公共镜像更新提醒：统计 ClawPro 已发布但企业还未采纳的版本
// ────────────────────────────────────────────────────────────────
export interface PendingPublicImage {
  agentType: AgentTypeKey;
  latestVersion: string;
  latestReleaseTime: string;
  activeVersion: string | null; // 企业当前采纳的版本
}

export function getPendingPublicImages(): PendingPublicImage[] {
  const out: PendingPublicImage[] = [];
  const types: AgentTypeKey[] = ["OpenClaw", "Hermes", "LightclawACE", "MyAgent"];
  for (const t of types) {
    const active = AGENT_VERSIONS.find((v) => v.agentType === t && v.isActive);
    const latest = AGENT_VERSIONS.find((v) => v.agentType === t && v.isLatest);
    if (latest && (!active || active.version !== latest.version)) {
      out.push({
        agentType: t,
        latestVersion: latest.version,
        latestReleaseTime: latest.releaseTime,
        activeVersion: active?.version ?? null,
      });
    }
  }
  return out;
}

// ────────────────────────────────────────────────────────────────
// 镜像元信息：每个 Agent 类型对应一个官方镜像
// 一个镜像下可以有多个版本（发版流水线）
// ────────────────────────────────────────────────────────────────
export interface ImageMeta {
  imageId: string;
  imageName: string;
}

const IMAGE_META_BY_TYPE: Record<AgentTypeKey, ImageMeta> = {
  OpenClaw:     { imageId: "img-agent-official",     imageName: "云服务器 OpenClaw 镜像" },
  Hermes:       { imageId: "img-hermes-official",    imageName: "Hermes Agent 官方镜像" },
  LightclawACE: { imageId: "img-lightclaw-official", imageName: "LightClaw ACE 官方镜像" },
  MyAgent:      { imageId: "img-myagent-custom",     imageName: "MyAgent 自研镜像" },
};

export function getImageMeta(agentType: AgentTypeKey): ImageMeta {
  return IMAGE_META_BY_TYPE[agentType];
}

// ────────────────────────────────────────────────────────────────
// 命令模板库（参考腾讯云 TAT 命令管理）
//
// 当前版本仅支持 Linux Shell（暂不考虑 Windows / PowerShell / Python）
// ────────────────────────────────────────────────────────────────
export type CommandType = "SHELL";

export interface CommandParam {
  key: string;                   // 变量名（命令内容中通过 {{key}} 引用）
  defaultValue: string;          // 默认值
  description?: string;          // 说明（用于下发时给操作者看）
}

export interface CommandTemplate {
  id: string;
  name: string;                  // 命令名称
  description?: string;          // 备注
  type: CommandType;             // 命令类型，目前固定 SHELL
  workingDir: string;            // 执行路径，留空则下发时按默认 /root
  runAsUser: string;             // 执行用户，留空则下发时按默认 root
  timeoutSec: number;            // 超时时间（秒）
  content: string;               // 命令内容
  // 参数化（参考 TAT「使用参数」）
  useParams?: boolean;           // 是否启用参数
  params?: CommandParam[];       // 参数定义列表
  createdBy: string;             // 创建人
  createdAt: string;
  updatedAt: string;
  // 使用统计
  lastRunAt?: string;
  totalRuns: number;
}

// 简单内存存储（前端 mock；真实项目应放在后端）
export const MOCK_COMMAND_TEMPLATES: CommandTemplate[] = [
  // ⭐ 置顶 1 条带参数的命令，方便前端自测参数化下发流程
  {
    id: "cmd-restart-service",
    name: "重启指定服务",
    description: "按 systemd 服务名重启服务并查看状态",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 60,
    content: "systemctl restart {{service_name}} && systemctl status {{service_name}} --no-pager",
    useParams: true,
    params: [
      {
        key: "service_name",
        defaultValue: "nginx",
        description: "要重启的 systemd 服务名（如 nginx、openclaw、redis）",
      },
    ],
    createdBy: "admin@acompany.com",
    createdAt: "2026-05-10 14:00:00",
    updatedAt: "2026-05-15 09:30:00",
    lastRunAt: "2026-05-18 11:20:00",
    totalRuns: 14,
  },
  {
    id: "cmd-cleanup-logs",
    name: "清理临时日志",
    description: "清理 /tmp 下 7 天前的日志文件，避免磁盘占满",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 60,
    content: "find /tmp -name '*.log' -mtime +7 -delete && echo done",
    createdBy: "admin@acompany.com",
    createdAt: "2026-04-15 10:00:00",
    updatedAt: "2026-04-15 10:00:00",
    lastRunAt: "2026-05-08 15:22:00",
    totalRuns: 6,
  },
  {
    id: "cmd-disk-usage",
    name: "检查磁盘使用率",
    description: "查看每个挂载点的使用率，便于排查磁盘告警",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 30,
    content: "df -h | grep -v tmpfs | awk 'NR>1 {print $1\" \"$5}'",
    createdBy: "ops@acompany.com",
    createdAt: "2026-04-10 09:30:00",
    updatedAt: "2026-05-01 14:00:00",
    lastRunAt: "2026-05-07 10:08:33",
    totalRuns: 12,
  },
  {
    id: "cmd-refresh-hosts",
    name: "刷新 hosts 配置",
    description: "追加内部 LLM 服务的 hosts 解析",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 60,
    content: "cat >> /etc/hosts <<EOF\n10.0.0.10 internal-llm.local\nEOF\nsystemctl restart nscd || true",
    createdBy: "admin@acompany.com",
    createdAt: "2026-05-01 11:00:00",
    updatedAt: "2026-05-06 22:00:00",
    lastRunAt: "2026-05-06 22:00:08",
    totalRuns: 2,
  },
  {
    id: "cmd-restart-agent",
    name: "重启 Agent 进程",
    description: "重启 OpenClaw / Hermes Agent 服务",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 90,
    content: "systemctl restart openclaw && systemctl status openclaw --no-pager",
    createdBy: "admin@acompany.com",
    createdAt: "2026-04-20 16:00:00",
    updatedAt: "2026-04-20 16:00:00",
    totalRuns: 0,
  },
  {
    id: "cmd-mem-usage",
    name: "查看内存占用 Top 10",
    description: "按 RSS 排序展示占用最高的 10 个进程",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 30,
    content: "ps aux --sort=-%mem | head -n 11",
    createdBy: "ops@acompany.com",
    createdAt: "2026-04-12 10:20:00",
    updatedAt: "2026-04-12 10:20:00",
    lastRunAt: "2026-05-12 09:15:00",
    totalRuns: 8,
  },
  {
    id: "cmd-cpu-usage",
    name: "查看 CPU 占用 Top 10",
    description: "按 CPU 占用排序展示前 10 个进程",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 30,
    content: "ps aux --sort=-%cpu | head -n 11",
    createdBy: "ops@acompany.com",
    createdAt: "2026-04-12 10:25:00",
    updatedAt: "2026-04-12 10:25:00",
    lastRunAt: "2026-05-13 16:40:00",
    totalRuns: 5,
  },
  {
    id: "cmd-clear-pagecache",
    name: "清理 Page Cache",
    description: "释放系统页缓存（仅清缓存，不影响进程）",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 30,
    content: "sync && echo 1 > /proc/sys/vm/drop_caches",
    createdBy: "admin@acompany.com",
    createdAt: "2026-04-18 14:00:00",
    updatedAt: "2026-04-18 14:00:00",
    totalRuns: 1,
  },
  {
    id: "cmd-tail-syslog",
    name: "查看系统日志末尾",
    description: "返回 /var/log/messages 最后 200 行，便于排查",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 30,
    content: "tail -n 200 /var/log/messages",
    createdBy: "ops@acompany.com",
    createdAt: "2026-04-22 11:00:00",
    updatedAt: "2026-04-22 11:00:00",
    lastRunAt: "2026-05-09 14:22:00",
    totalRuns: 4,
  },
  {
    id: "cmd-check-port",
    name: "检查指定端口监听",
    description: "查看指定端口是否被监听及监听者",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 15,
    content: "ss -tlnp | grep :{{port}} || echo 'port {{port}} not listening'",
    useParams: true,
    params: [
      {
        key: "port",
        defaultValue: "8080",
        description: "要检查的端口号（如 80、8080、443）",
      },
    ],
    createdBy: "ops@acompany.com",
    createdAt: "2026-04-25 09:00:00",
    updatedAt: "2026-05-02 10:00:00",
    lastRunAt: "2026-05-15 11:30:00",
    totalRuns: 9,
  },
  {
    id: "cmd-update-pkg-index",
    name: "更新软件包索引",
    description: "刷新 yum/apt 软件源元数据",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 120,
    content: "if command -v apt-get >/dev/null; then apt-get update -y; else yum makecache -y; fi",
    createdBy: "admin@acompany.com",
    createdAt: "2026-04-28 15:30:00",
    updatedAt: "2026-04-28 15:30:00",
    totalRuns: 0,
  },
  {
    id: "cmd-show-load",
    name: "查看系统负载",
    description: "uptime + 当前 1/5/15 分钟负载",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 10,
    content: "uptime",
    createdBy: "ops@acompany.com",
    createdAt: "2026-05-02 09:30:00",
    updatedAt: "2026-05-02 09:30:00",
    lastRunAt: "2026-05-18 09:00:00",
    totalRuns: 22,
  },
  {
    id: "cmd-list-users",
    name: "列出最近登录用户",
    description: "查看最近 20 条登录记录",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 15,
    content: "last -n 20",
    createdBy: "security@acompany.com",
    createdAt: "2026-05-04 16:00:00",
    updatedAt: "2026-05-04 16:00:00",
    lastRunAt: "2026-05-16 18:00:00",
    totalRuns: 3,
  },
  {
    id: "cmd-network-check",
    name: "网络连通性检查",
    description: "对指定主机做 ping + DNS 解析检查",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 30,
    content: "ping -c 3 {{host}} && dig +short {{host}}",
    useParams: true,
    params: [
      {
        key: "host",
        defaultValue: "internal-llm.local",
        description: "要检查的目标主机名或 IP",
      },
    ],
    createdBy: "ops@acompany.com",
    createdAt: "2026-05-06 10:00:00",
    updatedAt: "2026-05-06 10:00:00",
    lastRunAt: "2026-05-17 14:00:00",
    totalRuns: 7,
  },
  {
    id: "cmd-reload-config",
    name: "重载配置（不重启）",
    description: "向指定服务发送 SIGHUP 信号触发配置重载",
    type: "SHELL",
    workingDir: "/root",
    runAsUser: "root",
    timeoutSec: 15,
    content: "systemctl reload {{service_name}} && echo reloaded",
    useParams: true,
    params: [
      {
        key: "service_name",
        defaultValue: "nginx",
        description: "要重载配置的 systemd 服务名",
      },
    ],
    createdBy: "admin@acompany.com",
    createdAt: "2026-05-08 13:00:00",
    updatedAt: "2026-05-08 13:00:00",
    totalRuns: 0,
  },
];

// 危险命令前缀（创建/下发命令时做前置告警）
export const DANGEROUS_COMMAND_PATTERNS: { pattern: RegExp; reason: string }[] = [
  { pattern: /\brm\s+-rf\s+\/(\s|$)/, reason: "rm -rf / 会删除根目录所有文件，请谨慎" },
  { pattern: /\bmkfs(\.\w+)?\b/, reason: "mkfs 会格式化磁盘，请谨慎" },
  { pattern: /\bdd\s+if=/, reason: "dd 写入命令可能损坏磁盘数据，请谨慎" },
  { pattern: /\bshutdown\b/, reason: "shutdown 会关机，请谨慎" },
  { pattern: /\breboot\b/, reason: "reboot 会重启服务器" },
  { pattern: /:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:/, reason: "Fork 炸弹会耗尽系统资源" },
];

/** 检查命令内容是否触发危险命令规则 */
export function detectDangerousCommand(content: string): { dangerous: boolean; reasons: string[] } {
  const reasons: string[] = [];
  for (const { pattern, reason } of DANGEROUS_COMMAND_PATTERNS) {
    if (pattern.test(content)) reasons.push(reason);
  }
  return { dangerous: reasons.length > 0, reasons };
}
