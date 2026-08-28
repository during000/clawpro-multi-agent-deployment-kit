/**
 * tenantProjectStore —— 员工端「我的项目 / 项目工作台」Mock Store
 *
 * 范式（重要）：真实项目成员负责，成员授权自己的 agent 参与执行。
 *  - ClawPro 以项目为单位连接成员、个人 agent 与项目资产；
 *  - agent 干活时遇到经验「自动上报」，其他 agent 干活前「自动检索(recall)」项目经验 → 正循环；
 *  - 项目公共数据（资产使用次数、被学习次数等）由 agent 行为产生，工作台只做呈现；
 *  - 人负责目标、授权与确认，agent 负责执行、同步上下文与交付产物。
 *
 * 本 store 为员工端「项目工作台」提供自包含 mock（localStorage + useSyncExternalStore）：
 *  - 概览：项目目标 + 健康度指标（含进行中任务）+ 最近动态；
 *  - 任务：负责人 + 优先级 + 截止日期 + 多 agent 工作流 + 节点结果确认；
 *  - 经验：agent 自动上报的经验流 + 被学习(recall)次数（飞轮）+ 经验转 Skill（V2）；
 *  - 资产：6 大类资产（可开关编辑）+ skill 使用统计。
 *
 * 说明：不改 groupStore / projectAssetStore / 用户管理；资产大类与展示复用管控端 project-assets 的类型与选择器。
 * 经验/使用等口径对齐 teamai（share-learnings / recall / usage-tracker），将来接真实数据不返工。
 */
import { useSyncExternalStore } from 'react';
import {
  ASSET_CATEGORY_MAP,
  ASSET_CATEGORY_ORDER,
  type AssetCategory,
  type ProjectAssetChangeSection,
  type ProjectAssetUpdateRecord,
} from '../admin/project-assets/types';
import {
  getCategoryLibraryItems,
  getAssetItemDisplay,
} from '../admin/project-assets/assetSelectors';
import {
  cloneMultiAgentNodeAssets,
  MULTI_AGENT_DEVELOPMENT_ASSET_VERSION,
  type WorkflowNodeConfigAsset,
} from './workflow/multiAgentDevelopmentAssets';

export interface TenantProjectMember {
  userId: string;
  displayName: string;
  /** 职能角色（产品/前端/后端/测试…）——用于流水线按角色匹配授权 agent */
  role: string;
  /** 管理权限：true=项目管理员（可配汇报规格/派汇报/建模板），false=普通成员。可多个管理员 */
  admin: boolean;
}

/** 每个大类保存已选资产的 refId 列表 */
export type TenantProjectAssets = Record<AssetCategory, string[]>;

export type TenantAgentLocation = 'cloud' | 'local';

/** 项目成员明确授权给本项目使用的个人 agent */
export interface TenantAgent {
  id: string;
  name: string;
  platform:
    | 'clawpro'
    | 'imate'
    | 'codex'
    | 'codebuddy'
    | 'workbuddy'
    | 'cloudagent';
  location: TenantAgentLocation;
  ownerId: string;
  owner: string;
  /** 角色标签（可选；新版不再强制，避免"产品/前端"这种固化分类） */
  role?: string;
  status: 'online' | 'offline';
  authorization: string;
  /** 归属：personal=成员授权接入的个人agent；project=项目公共agent(不绑人) */
  kind?: 'personal' | 'project';
  /** 真实工作流 Runtime 路由（POC 联调时由后端实时返回） */
  runtimeId?: string;
  deviceId?: string;
  targetAgentId?: string;
  runtimeDetail?: string;
  /** TeamAI 对该 Runtime 实时校验通过的受控能力。 */
  runtimeCapabilities?: string[];
  runtimeMissingCapabilities?: string[];
}

/** 一条 agent 自动上报的经验（对齐 teamai learnings） */
export interface TenantLearning {
  id: string;
  title: string;
  /** 来源 agent（如 "bob 的 agent"）——经验由 agent 上报，非人工录入 */
  sourceAgent: string;
  /** 触发场景（踩坑 / 被纠正 / 新解法 等） */
  scene: string;
  tags: string[];
  summary: string;
  /** 被其他 agent 学习(recall)的次数——体现飞轮转起来 */
  recalledCount: number;
  time: string;
}

/** 任务状态：待启动 / 进行中(含待确认) / 已完成 / 挂起 */
export type TenantTaskStatus = 'todo' | 'in_progress' | 'review' | 'done' | 'hold';
export type TenantTaskPriority = 'high' | 'medium' | 'low';
export type TenantWorkflowTemplateId =
  | 'none'
  | 'auto'
  | 'clawpro-collaboration'
  | 'rapid-development'
  | 'requirement-development'
  | 'lightweight-development'
  | `custom-${string}`;
export type TenantWorkflowNodeStatus =
  | 'pending'
  | 'running'
  | 'stopped'
  | 'review'
  | 'confirmed';
export type TenantWorkflowNodeKind = 'execution' | 'dispatch' | 'summary';

export interface TenantWorkflowDraftNode {
  id: string;
  title: string;
  kind: TenantWorkflowNodeKind;
  x: number;
  y: number;
  dependsOn: string[];
}

export interface TenantWorkflowDraftStageNode {
  id: string;
  title: string;
  kind: TenantWorkflowNodeKind;
  /** 同一任务需要复制给多少个 Agent 并行执行 */
  repeatCount?: number;
}

export type TenantWorkflowDraftStage = TenantWorkflowDraftStageNode[];

export interface TenantTaskArtifact {
  id: string;
  name: string;
  type: 'Markdown' | 'JSON';
  content: string;
}

/** ── 工作流节点 IO 契约 ──
 * 节点显式声明输入/输出，形成"上游产出=下游输入"的数据流转。
 * source 空表示"来自任务输入"或"上游全部产出的自然语言拼接"。
 */
export type IOFieldType = 'text' | 'markdown' | 'json' | 'file' | 'url';

export interface NodeIOField {
  key: string;
  label: string;
  type: IOFieldType;
  description?: string;
  required?: boolean;
  /** 输入字段：从上游哪个节点的哪个 output 字段拉数据 */
  source?: { nodeId: string; outputKey: string };
}

/** 节点执行时产生的实际 IO 值（运行时变量池的一格） */
export interface IOValue {
  type: IOFieldType;
  value: string;
  producedBy?: string;
  producedAt?: string;
}

export interface TenantWorkflowNode {
  id: string;
  title: string;
  /** 节点类型（main 上新增，本版可空） */
  kind?: TenantWorkflowNodeKind;
  /** 上游节点全部确认后，本节点才能启动 */
  dependsOn: string[];
  /** Agent 节点自动执行；human 节点由对应负责人本人操作（main 上新增，本版可空） */
  executionMode?: 'agent' | 'human';
  /** 真实成员职责，用于匹配该成员授权的个人 Agent（main 上新增，本版可空） */
  role?: string | null;
  /** 当前节点完成后向下一节点同步的协作信息（main 上新增，本版可空） */
  handoff?: string | null;
  agentId: string | null;
  status: TenantWorkflowNodeStatus;
  result: string | null;
  artifacts: TenantTaskArtifact[];
  /** 真实编排 Runtime 使用的稳定节点 ID。 */
  runtimePhaseId?: string;
  /** 下发给真实 Agent 的节点提示词。 */
  runtimePrompt?: string;
  /** 节点运行时加载的 Rules / Skill / Contract 配置资产。 */
  configAssets?: WorkflowNodeConfigAsset[];
  /** 节点输入/输出契约，用于生成 Handoff v2。 */
  runtimeInputs?: NodeIOField[];
  runtimeOutputs?: NodeIOField[];
  /** 节点完成后是否必须由用户确认才能继续。 */
  runtimeApprovalRequired?: boolean;
  /** 真实执行结果中必须出现的来源/调用证据。 */
  runtimeRequiredEvidence?: string[];
  /** 命中任一标记时视为没有完成真实读取。 */
  runtimeRejectOutputMarkers?: string[];
  /** 节点启动前必须由所选 Runtime 具备的能力。 */
  runtimeRequiredCapabilities?: string[];
  /** 节点成功/失败后的显式路由，供真实 Runtime 状态机执行。 */
  runtimeOnPass?: string | null;
  runtimeOnFail?: string | null;
  /** 需要从结果中读取机器可判定结论的节点类型。 */
  runtimeDecisionMode?: 'review_verdict' | 'size_class';
  /** 失败路由最多允许触发的返工次数。 */
  runtimeMaxRetries?: number;
  /** 运行时输入变量池（key = 节点 IO 契约的 inputs[].key） */
  inputValues?: Record<string, IOValue>;
  /** 运行时输出变量池（key = 节点 IO 契约的 outputs[].key） */
  outputValues?: Record<string, IOValue>;
  /** 节点类型：task=普通任务节点，loop=循环节点 */
  type?: 'task' | 'loop';
  /** 循环节点配置（从模板继承） */
  loopConfig?: {
    endNodeId: string;
    maxCount?: number;
    exitCondition?: string;
  };
  /** 已循环次数（运行时累加） */
  loopCount?: number;
}

export interface TenantWorkflowTemplate {
  id: TenantWorkflowTemplateId;
  name: string;
  stages: TenantWorkflowDraftStage[];
}

export const TENANT_WORKFLOW_TEMPLATES: TenantWorkflowTemplate[] = [
  { id: 'none', name: '无', stages: [] },
  { id: 'auto', name: 'PM 智能生成', stages: [] },
  {
    id: 'clawpro-collaboration',
    name: 'ClawPro 产研协作',
    stages: [
      [{ id: 'draft-node-1', title: '产品开发前端 Demo', kind: 'execution' }],
      [{ id: 'draft-node-2', title: '基于 Demo 编写需求单', kind: 'execution' }],
      [{ id: 'draft-node-3', title: '后端开发与自测', kind: 'execution' }],
      [{ id: 'draft-node-4', title: '前端正式开发与联调', kind: 'execution' }],
      [{ id: 'draft-node-5', title: '产品验收', kind: 'execution' }],
    ],
  },
  {
    id: 'rapid-development',
    name: '需求快速开发',
    stages: [
      [{ id: 'draft-node-1', title: '产品需求分析', kind: 'execution' }],
      [{ id: 'draft-node-2', title: '技术开发', kind: 'execution' }],
      [{ id: 'draft-node-3', title: '测试', kind: 'execution' }],
    ],
  },
  {
    id: 'requirement-development',
    name: '需求开发流程',
    stages: [
      [{ id: 'draft-node-1', title: '产品需求分析', kind: 'execution' }],
      [{ id: 'draft-node-2', title: '架构设计与开发', kind: 'execution' }],
      [{ id: 'draft-node-3', title: '代码评审与测试', kind: 'execution' }],
    ],
  },
  {
    id: 'lightweight-development',
    name: '轻量任务开发',
    stages: [
      [{ id: 'draft-node-1', title: '编码实现', kind: 'execution' }],
      [{ id: 'draft-node-2', title: '验证确认', kind: 'execution' }],
    ],
  },
];

/** ── 流水线模板（项目级 SOP，场景2：标准化任务流水线） ──
 * 与全局 TENANT_WORKFLOW_TEMPLATES 的区别：
 *  - 归属到单个项目，可增删改，固化「专门处理某类工作」的既定工作流；
 *  - 节点预设执行角色 agentRole，issue 实例化时按成员 role 自动匹配授权 agent 并指派；
 *  - 一条 issue = 一个 TenantTask，绑定模板后按模板节点实例化 workflow(DAG)。
 */
export interface TenantPipelineTemplateNode {
  id: string;
  title: string;
  /** 可视化画布中的手动布局位置；未设置时使用 DAG 自动布局。 */
  position?: { x: number; y: number };
  /** 依赖的上游节点 id，全部确认后本节点才启动（并行 DAG） */
  dependsOn: string[];
  /** 预设执行角色（旧字段，仅 seed 数据自动匹配用，UI 不再展示） */
  agentRole: string;
  executorKind?: 'memberAgent' | 'projectAgent';
  executorRef?: string;
  /** IO 契约：输入字段声明（可选。为空时 fallback 到"上游全部产出自然语言拼接"给 agent） */
  inputs?: NodeIOField[];
  /** IO 契约：输出字段声明（可选） */
  outputs?: NodeIOField[];
  /** agent 执行时的 prompt 模板，可用 {{inputKey}} 引用 inputs 字段 */
  promptTemplate?: string;
  /** 节点运行时加载的 Rules / Skill / Contract 配置资产。 */
  configAssets?: WorkflowNodeConfigAsset[];
  /** 真实执行时节点完成后暂停，等待用户确认。 */
  approvalRequired?: boolean;
  /** 真实执行结果中必须出现的来源/调用证据。 */
  requiredEvidence?: string[];
  /** 命中任一标记时使节点失败，禁止模拟结果继续流转。 */
  rejectOutputMarkers?: string[];
  /** 可被节点调度的 Runtime 必须实时声明这些能力。 */
  requiredCapabilities?: string[];
  /** 节点成功/失败后的显式路由；为空表示工作流结束。 */
  onPass?: string | null;
  onFail?: string | null;
  /** 结果判定方式：代码评审或需求规模分流。 */
  decisionMode?: 'review_verdict' | 'size_class';
  /** 失败后最多允许的返工次数。 */
  maxRetries?: number;
  /** 节点类型：task=普通任务节点，loop=循环节点（标记循环起点） */
  type?: 'task' | 'loop';
  /** 循环节点配置（仅 type='loop' 时有效） */
  loopConfig?: {
    /** 循环结束节点 id（该节点完成后回到循环节点重新执行循环体） */
    endNodeId: string;
    /** 最大循环次数（默认3） */
    maxCount?: number;
    /** 退出条件说明（提示文字，如"评审通过后退出循环"） */
    exitCondition?: string;
  };
}

export interface TenantPipelineTemplate {
  id: string;
  name: string;
  description: string;
  nodes: TenantPipelineTemplateNode[];
}

export interface TenantRuntimeExecutionPhase {
  id: string;
  title: string;
  runtimeId: string;
  agentInstanceId: string;
  status: string;
  outputs: string[];
  approvalRequired: boolean;
}

export interface TenantRuntimeExecutionArtifact {
  id: string;
  path: string;
  mediaType?: string;
  size?: number;
  sha256?: string;
}

/** 真实 ClawPro 编排后端在当前项目任务上的执行快照。 */
export interface TenantRuntimeExecution {
  backendTaskId: string;
  runtimeId: string;
  assignmentMode: string;
  agentRuntimeId: string;
  status: string;
  canStop?: boolean;
  cancelRequested?: boolean;
  currentPhase: string | null;
  handoffContract: string;
  phases: TenantRuntimeExecutionPhase[];
  artifacts: TenantRuntimeExecutionArtifact[];
  updatedAt: string;
}

/** 一条项目任务：人负责、agent 执行、节点结果由人确认 */
export interface TenantTask {
  id: string;
  title: string;
  description: string;
  status: TenantTaskStatus;
  ownerId: string;
  priority: TenantTaskPriority;
  dueDate: string;
  /** 旧字段，兼容早期"普通/汇报"二分法 */
  taskType?: 'normal' | 'report';
  /** 汇报任务归属的汇报周期 key（如 2026-W31）；普通任务为空 */
  reportPeriod?: string;
  workflowTemplateId: TenantWorkflowTemplateId;
  workflow: TenantWorkflowNode[];
  /** issue 来源：手动 / 导入 / MCP 上报 */
  source?: 'manual' | 'import' | 'mcp';
  pipelineTemplateId?: string;
  /**
   * 新版任务本质属性（触发来源，非业务分类）：
   *  - manual：人手动建（默认）
   *  - periodic：周期触发（如每周汇报，含 periodicSpec）
   *  - external：外部 agent 通过 MCP/API 触发（含 externalContext）
   *  - system：系统自动建
   */
  triggerType?: 'manual' | 'periodic' | 'external' | 'system';
  /** 业务标签（人贴，多选自由文本）；预置：需求/缺陷/汇报/评审 */
  tags?: string[];
  /** 仅 triggerType='external' 时有：外部调用上下文 */
  externalContext?: {
    source: string;
    requester: string;
    inputPrompt: string;
  };
  /** 仅 triggerType='periodic' 时有：周期规格 */
  periodicSpec?: {
    cycle: 'daily' | 'weekly' | 'biweekly';
    nextAt: string;
  };
  /** 任务级输入池（首节点从这里读；例如 workbuddy 传入的 pr_url） */
  taskInputs?: Record<string, IOValue>;
  /** 与真实工作流后端绑定后的执行快照。 */
  runtimeExecution?: TenantRuntimeExecution;
  /** 可通过 ClawPro → TeamAI → 真实 Agent Runtime 启动。 */
  runtimeExecutable?: boolean;
  createdAt: string;
  updatedAt: string;
}

/** 预置业务标签 */
export const TENANT_TASK_TAG_PRESETS = ['需求', '缺陷', '汇报', '评审'] as const;

/** skill 使用统计（对齐 teamai usage-tracker） */
export interface TenantSkillUsage {
  name: string;
  calls: number;
  lastUsed: string;
}

/** 项目动态流条目 */
export interface TenantActivity {
  id: string;
  kind: 'learning_report' | 'learning_recall' | 'asset_add' | 'task_dispatch' | 'task_done' | 'skill_convert' | 'report_dispatch' | 'report_submit' | 'automation_run';
  text: string;
  time: string;
}

/** ── 项目自动化（为项目服务：周期性/事件触发地自动创建并执行项目任务）──
 * 心智：自动化 = 任务的自动生成器。到点触发 → 按输出模式决定是否在项目里建任务 →
 * 可选走某条项目工作流 → 结果沉淀回任务看板，天然与任务打通。
 */
export type TenantAutomationScheduleType = 'periodic' | 'interval' | 'once';
export type TenantAutomationTriggerKind = 'schedule' | 'webhook';
/** 输出模式：createTask=每次运行建一条可追踪任务；runOnly=静默运行不建任务 */
export type TenantAutomationOutputMode = 'createTask' | 'runOnly';

export interface TenantAutomationSchedule {
  type: TenantAutomationScheduleType;
  /** periodic：周期粒度 */
  cycle?: 'daily' | 'weekly';
  /** periodic：每周几（cycle=weekly 时），0=周日 */
  weekday?: number;
  /** periodic：HH:mm */
  time?: string;
  /** interval：每隔多少小时 */
  intervalHours?: number;
  /** once：一次性执行时间 ISO */
  onceAt?: string;
}

export interface TenantAutomation {
  id: string;
  name: string;
  /** 提示词（可选，会作为额外指令/任务描述补充给工作流） */
  prompt: string;
  /** 触发方式：时间表 / webhook（webhook 本期占位） */
  triggerKind: TenantAutomationTriggerKind;
  schedule: TenantAutomationSchedule;
  outputMode: TenantAutomationOutputMode;
  /**
 * 关联工作流（必填）：自动化本身不管执行者，
   * 谁来跑完全由工作流节点自己的 executor 决定。
   */
  pipelineTemplateId: string;
  /** 生效区间（可选） */
  validFrom?: string;
  validUntil?: string;
  /** 启停*/
  enabled: boolean;
  /** 展示用：上次/下次运行时间 */
  lastRunAt?: string;
  nextRunAt?: string;
  createdAt: string;
}

/** 当前用户在 ClawPro 已注册、可直接授权接入项目的个人 agent（mock 池） */
export interface MyRegisteredAgent {
  id: string;
  name: string;
  platform: TenantAgent['platform'];
}

/** 接入外部 agent 的等待态：生成接入码后轮询捕获 */
export interface PendingAgentConnection {
  code: string;
  createdAt: number;
  /** mock：模拟外部安装完成的时间点（毫秒时间戳），到点后 check 即可捕获 */
  readyAt: number;
  name: string;
  platform: TenantAgent['platform'];
}


/** 项目健康度指标 */
export interface TenantProjectMetrics {
  agentCount: number;
  learningCount: number;
  weekRecalled: number;
  weekActiveSessions: number;
  inProgressTasks: number;
}

/** ── 汇报规格（项目配置层·管理员的整体规划，场景1） ──
 * 管理员定义「大家怎么汇报」：填哪些解读字段、多久汇报一次、谁要汇报。
 * 这套规格 = 下发给成员 agent 的 skill：agent 领到汇报任务时按此产出快照。
 * 起步字段固定为四项，接口保留 fields 以便后续扩展成自定义表单。
 */
export interface TenantReportFieldDef {
  key: string;
  label: string;
  placeholder: string;
}

export type TenantReportCycle = 'weekly' | 'biweekly';

export interface TenantReportingSpec {
  /** 是否启用汇报（管理员在项目配置里开启） */
  enabled: boolean;
  cycle: TenantReportCycle;
  /** 解读字段集（起步固定四项：综述/风险/下一步/需要支持） */
  fields: TenantReportFieldDef[];
  /** 汇报人范围：成员 userId 列表；空数组表示全体成员 */
  reporterScope: string[];
}

/** 起步固定的四项解读字段 */
export const DEFAULT_REPORT_FIELDS: TenantReportFieldDef[] = [
  { key: 'summary', label: '本期进展综述', placeholder: '本周期主要进展与亮点' },
  { key: 'risk', label: '风险与阻塞', placeholder: '遇到的风险、卡点、需要预警的问题' },
  { key: 'next', label: '下一步计划', placeholder: '下个周期要推进的重点' },
  { key: 'support', label: '需要的支持', placeholder: '需要管理员/其他成员协调的资源或决策' },
];

function defaultReportingSpec(): TenantReportingSpec {
  return {
    enabled: false,
    cycle: 'weekly',
    fields: DEFAULT_REPORT_FIELDS.map((f) => ({ ...f })),
    reporterScope: [],
  };
}

/** ── 进展快照（上行资产，场景1） ──
 * 快照 = 进度底座（系统从任务/workflow 机械算，不让人重打） + 解读层（agent/人按规格填）。
 * 这样彻底避免「进展」和「任务进度」重复录入：客观进度自动来，人只补机器给不出的判断。
 */
export interface TenantProgressBase {
  /** 该成员名下任务总数 */
  taskTotal: number;
  /** 已完成任务数 */
  taskDone: number;
  /** 进行中/待确认任务数 */
  taskActive: number;
  /** 综合完成度百分比（按节点确认比例 roll-up） */
  completion: number;
  /** 卡点摘要（停止的节点 / 待确认过久等，机械提取） */
  blockers: string[];
  /** 引用的任务 id，便于下钻 */
  taskIds: string[];
}

export interface TenantProgressSnapshot {
  id: string;
  /** 汇报周期 key，如 2026-W31 */
  period: string;
  periodLabel: string;
  reporterId: string;
  reporterName: string;
  /** 关联的汇报任务 id */
  taskId: string;
  /** 系统机械算的进度底座 */
  progressBase: TenantProgressBase;
  /** 按规格填的解读层：字段 key → 内容 */
  interpretation: Record<string, string>;
  /** 快照状态：待填写（agent 未产出）/ 已提交 */
  status: 'pending' | 'submitted';
  submittedAt: string | null;
  createdAt: string;
}

export interface TenantProject {
  id: string;
  name: string;
  description: string;
  /** 项目目标（一句话） */
  goal: string;
  /** 头像取色键 */
  colorKey: string;
  members: TenantProjectMember[];
  /** 项目内被托管的 agent */
  agents: TenantAgent[];
  /** 管控端开关：是否允许项目成员编辑本项目资产 */
  allowMemberEdit: boolean;
  /** 汇报规格（项目配置层·管理员整体规划） */
  reportingSpec: TenantReportingSpec;
  /** 进展快照（按周期留痕的上行资产） */
  progressSnapshots: TenantProgressSnapshot[];
  assets: TenantProjectAssets;
  assetVersion: number;
  /** 资产版本更新记录；存量项目缺省时由页面按当前版本生成兜底记录 */
  assetUpdateRecords?: ProjectAssetUpdateRecord[];
  /** 项目经验库 Skill 的版本（agent 上报的新经验持续并入即 +1，整体下发给项目 agent） */
  experienceSkillVersion: number;
  /** 旧版经验飞轮总开关；存量数据用于两个独立开关的默认值。 */
  experienceFlywheelEnabled?: boolean;
  /** 是否允许项目 Agent 在任务前自动检索经验以节省 Token。 */
  experienceRecallEnabled?: boolean;
  /** 是否允许项目 Agent 自动上报并沉淀任务经验。 */
  experienceDepositionEnabled?: boolean;
  metrics: TenantProjectMetrics;
  learnings: TenantLearning[];
  tasks: TenantTask[];
  /** 项目级流水线模板（场景2：标准化任务流水线，可增删改） */
  pipelineTemplates: TenantPipelineTemplate[];
  /** 项目自动化（为项目服务：定时/事件触发地自动建任务并执行） */
  automations: TenantAutomation[];
  skillUsage: TenantSkillUsage[];
  activities: TenantActivity[];
  /** 本项目经验库被其他项目引用（链接复用）的次数 —— Agent 社区声誉 */
  referencedCount: number;
  /** 本项目已引用的其他项目经验库 id 列表 */
  referencedLibraries: string[];
  updatedAt: string;
}

const CACHE_KEY = 'tenant_projects_cache';
const CACHE_VERSION_KEY = 'tenant_projects_cache_version';
// v26：加入“拜访前简报”真实执行 SOP 与任务样例。
const CACHE_VERSION = '26';

export const TENANT_PROJECT_EVENT = 'tenant-project-store-updated';

/** 空资产集合 */
function emptyAssets(): TenantProjectAssets {
  return ASSET_CATEGORY_ORDER.reduce((acc, c) => {
    acc[c] = [];
    return acc;
  }, {} as TenantProjectAssets);
}

/** 从工具库某大类取前若干真实资产 refId */
function pickRefs(category: AssetCategory, count: number, offset = 0): string[] {
  const items = getCategoryLibraryItems(category);
  return items.slice(offset, offset + count).map((i) => i.refId);
}

function buildAssets(config: Partial<Record<AssetCategory, [number, number]>>): TenantProjectAssets {
  const assets = emptyAssets();
  (Object.keys(config) as AssetCategory[]).forEach((category) => {
    const [count, offset] = config[category]!;
    assets[category] = pickRefs(category, count, offset);
  });
  return assets;
}

const M = (
  userId: string,
  displayName: string,
  role: string,
  admin = false,
): TenantProjectMember => ({
  userId,
  displayName,
  role,
  admin,
});
const CURRENT_USER = 'alice@acompany.com';

/** 当前用户在 ClawPro 已注册的个人 agent 池（mock；方式1"从已有 agent 选"的数据源） */
const MY_REGISTERED_AGENTS: MyRegisteredAgent[] = [
  { id: 'my-ag-1', name: '我的代码审查助手', platform: 'codebuddy' },
  { id: 'my-ag-2', name: '我的文档整理助手', platform: 'clawpro' },
  { id: 'my-ag-3', name: '我的日报助手', platform: 'workbuddy' },
];

// ── 经验池（agent 上报的通用研发经验模板） ────────────────
const LEARNING_POOL: Array<{ title: string; scene: string; tags: string[]; summary: string }> = [
  { title: 'Vite dev server 端口冲突排查', scene: '工具重试', tags: ['troubleshooting', 'networking'], summary: 'dev server 端口被占用时应先用 lsof 定位占用进程，或改用 --port 指定端口，避免反复启动失败。' },
  { title: 'K8s Pod 长时间 Pending 的常见原因', scene: '用户纠正', tags: ['k8s', 'ops'], summary: 'Pending 多为资源不足、节点亲和/污点、或 PVC 未就绪；先 describe pod 看 Events 再逐项排查。' },
  { title: '大文件读取应分块，避免 OOM', scene: '工具报错', tags: ['performance'], summary: '一次性读入超大文件会触发 OOM，改用流式/分块读取并设置上限，超限时降级提示。' },
  { title: '提交前必须扫描明文密钥', scene: '被打断', tags: ['security'], summary: '检测到疑似密钥（token/secret/password）应阻断提交并提示改用密钥管理，不写入仓库。' },
  { title: 'SQL 慢查询先看执行计划', scene: '新解法', tags: ['database'], summary: '用 EXPLAIN 定位缺失索引或全表扫描，优先补索引而非改写语句。' },
  { title: '镜像构建缓存命中优化', scene: '新解法', tags: ['ci', 'image'], summary: '把变动少的依赖安装放在 Dockerfile 前面，合理分层可显著提升缓存命中、缩短构建。' },
  { title: '前端接口联调优先约定 mock', scene: '用户纠正', tags: ['frontend'], summary: '后端未就绪时先按接口约定造 mock 数据并行开发，联调阶段再切真实接口。' },
  { title: '日志排查先串 traceId', scene: '工具重试', tags: ['observability'], summary: '通过 traceId 串联全链路日志，比逐服务翻日志高效得多，先拿到 traceId 再展开。' },
];

// ── 任务池（人派发、agent 执行的通用研发任务模板） ─────────
const TASK_POOL: Array<{ title: string; description: string }> = [
  { title: '梳理项目 README 与快速上手文档', description: '整理项目结构说明，补齐本地启动与常见问题，降低新人上手成本。' },
  { title: '为核心接口补充单元测试', description: '覆盖主要分支与边界，提升回归可靠性。' },
  { title: '排查线上偶发 5xx 报错', description: '结合 traceId 定位链路，给出根因与修复方案。' },
  { title: '优化构建缓存，缩短 CI 时长', description: '调整依赖分层与缓存策略，提升命中率。' },
  { title: '接入统一日志与告警', description: '打通 traceId，补充关键路径的告警规则。' },
  { title: '数据库慢查询专项优化', description: '基于执行计划补索引、改写热点查询。' },
];

export const TENANT_WORKFLOW_DEMO_TASK_ID = 'task-skillhub-issuefix-demo';
export const TENANT_KB_INSPECTION_TASK_ID = 'task-knowledge-base-inspection-demo';
export const TENANT_STANDARD_DEV_SOP_TASK_ID = 'task-standard-development-sop-demo';
export const TENANT_PROJECT_MANAGEMENT_SOP_TASK_ID = 'task-project-management-sop-demo';
export const TENANT_PRODUCT_FRONTEND_DEMO_SOP_TASK_ID = 'task-product-frontend-demo-sop';
export const TENANT_IMAGE_RELEASE_TASK_ID = 'task-public-image-release-follow-demo';
export const TENANT_MULTI_AGENT_DEV_TASK_ID = 'task-multi-agent-development-demo';
export const TENANT_PRE_VISIT_BRIEF_TASK_ID = 'task-pre-visit-brief-demo';
export const TENANT_WORKFLOW_IMATE_AGENT_ID = 'agent-runtime-imate-openclaw';
export const TENANT_WORKFLOW_DEMO_TEMPLATE_ID = 'pl-skillhub-issuefix-demo';
export const TENANT_KB_INSPECTION_TEMPLATE_ID = 'pl-knowledge-base-inspection';
export const TENANT_STANDARD_DEV_SOP_TEMPLATE_ID = 'pl-standard-development-sop';
export const TENANT_PROJECT_MANAGEMENT_SOP_TEMPLATE_ID = 'pl-project-management-sop';
export const TENANT_PRODUCT_FRONTEND_DEMO_SOP_TEMPLATE_ID = 'pl-product-frontend-demo-sop';
export const TENANT_IMAGE_RELEASE_TEMPLATE_ID = 'pl-public-image-release-follow';
export const TENANT_MULTI_AGENT_DEV_TEMPLATE_ID = 'pl-multi-agent-development-sop';
export const TENANT_PRE_VISIT_BRIEF_TEMPLATE_ID = 'pl-pre-visit-brief';

const MULTI_AGENT_DEV_REFERENCE_MANIFEST = JSON.stringify(
  {
    sourceArchive: 'contest_school_registry_20260810_1018.zip',
    comparisonMode: 'structure-and-outcome',
    coreArtifactCount: 10,
    artifacts: [
      { path: 'workflow-state.json', size: 110467, sha256: '8cc1d8d8523ad82bafe323a70d2fe2e441098c76a1ab8fb77d5608855c550bce' },
      { path: '01-requirement/requirement-report.md', size: 16756, sha256: '8fa7177d461425929d9afbc92909e7242e094a2a60eca969ba6077b22d7f0ad5' },
      { path: '02-design/tech-design.md', size: 44405, sha256: '22aba1611a0a2cf21e67601a28a62be223d6aa61ba8f4d9384d6a7fa875bba70' },
      { path: '02-design/execution-plan.md', size: 32078, sha256: 'c4aa8e95d25dba68a61736b051bcb97dcef9cdea4198353772944ab24c9bc435' },
      { path: '03-code/change-report.md', size: 56053, sha256: 'eb71418962eaf9f20636266591ab0b8e17747ffa9ffa181d8ac45af8a72df80f' },
      { path: '03-code/api-docs.md', size: 19289, sha256: 'b6c087aeef6b3e1d4323332cb103bd22fea456257e75bfe0c083f20c72cb6c19' },
      { path: '03-code/review-report.md', size: 21071, sha256: 'ccac2b154e706d57a43a81ef9baa7a35cdba7e6949c2aa30f8a177b52ffd3c41' },
      { path: '04-e2e/test-report.md', size: 44343, sha256: '4caab7b9bb3063a8cfaf58b4fbeaea8a9815931d6eba660b62fa18c5c48c79b9' },
      { path: '04-e2e/added-cases.md', size: 18619, sha256: '58d228948e3a1d8c5f423ca5fa19a9c6370275e8b4ffde882e57a9f91e788421' },
      { path: 'workflow-summary.md', size: 34749, sha256: 'df567174b3d7b51acdbcd6e3cb280b27adb13d5b67a44ad948c8257a5074280d' },
    ],
    optionalReferenceArtifact: {
      path: '04-test/README.md',
      size: 928,
      sha256: '06873553063e2c7d9b9e0292ce332acedb06b5a509c7130769dfd8f82dd829f2',
    },
    referenceOutcome: {
      verdict: 'COMPLETED',
      reviewRounds: 3,
      unresolvedP0: 0,
      unresolvedP1: 0,
      goTest: 'pass',
      incrementalCoverage: '96.4%',
      e2e: '脚本已生成并通过 bash -n，未连接真实测试环境执行',
    },
  },
  null,
  2,
);

const KB_READER_AGENT_ID = 'kb-lint-reader-agent';
const KB_AUDITOR_AGENT_ID = 'kb-lint-auditor-agent';

const ISSUEFIX_IMATE_PHASE_INDEXES = new Set([2, 3, 5, 6]);

const MUST_WIN_PROJECTS = [
  '轻量云域名解析相关产品',
  '轻量云解析数据面项目',
  '轻量云OrcaTerm项目',
  '轻量云AgentChat项目',
  '轻量云Lighthouse AI实验项目',
  '轻量云Lighthouse Agent项目',
  '轻量云Lighthouse非Agent项目',
  '轻量云SkillHub项目',
  '轻量云LightVela项目',
] as const;

const PROJECT_MANAGEMENT_AGENT_IDS = [
  'codebuddy:project-progress-a',
  'codebuddy:project-progress-b',
  'codebuddy:project-progress-c',
] as const;

export function isTenantExecutableWorkflowTask(task: TenantTask): boolean {
  return Boolean(
    task.runtimeExecutable ||
      [
        TENANT_WORKFLOW_DEMO_TASK_ID,
        TENANT_KB_INSPECTION_TASK_ID,
        TENANT_STANDARD_DEV_SOP_TASK_ID,
        TENANT_PROJECT_MANAGEMENT_SOP_TASK_ID,
        TENANT_PRODUCT_FRONTEND_DEMO_SOP_TASK_ID,
        TENANT_IMAGE_RELEASE_TASK_ID,
        TENANT_MULTI_AGENT_DEV_TASK_ID,
        TENANT_PRE_VISIT_BRIEF_TASK_ID,
      ].includes(task.id),
  );
}

const ISSUEFIX_DEMO_PHASES = [
  'Issue 分析',
  '修复实现',
  '代码评审',
  '测试验证',
  '创建 MR',
  'Checkers',
  'E2E 验证',
  '关闭 Issue',
];

const RECALL_SEED = [12, 8, 5, 7, 3];
const CALL_SEED = [156, 89, 42, 30, 18];
const TIME_SEED = ['2 天前', '3 天前', '5 天前', '上周'];
const TASK_STATUS_SEED: TenantTaskStatus[] = ['done', 'in_progress', 'review', 'todo', 'todo'];
const PRIORITY_SEED: TenantTaskPriority[] = ['medium', 'high', 'high', 'low'];
const AGENT_PROFILES: Array<
  Pick<TenantAgent, 'platform' | 'location'> & { label: string; authorization: string }
> = [
  {
    platform: 'clawpro',
    location: 'cloud',
    label: 'ClawPro 云端 Agent',
    authorization: '项目上下文与授权云端工具',
  },
  {
    platform: 'codex',
    location: 'local',
    label: 'Codex',
    authorization: '成员授权的本地 Workspace',
  },
  {
    platform: 'codebuddy',
    location: 'local',
    label: 'CodeBuddy',
    authorization: '成员授权的内网代码与工具',
  },
  {
    platform: 'imate',
    location: 'cloud',
    label: 'iMate 云端 Agent',
    authorization: '项目上下文与 Connector 授权内容',
  },
  {
    platform: 'workbuddy',
    location: 'local',
    label: 'WorkBuddy',
    authorization: '成员授权的本地文件与应用',
  },
];

function buildAgents(members: TenantProjectMember[]): TenantAgent[] {
  return members.map((m, i) => {
    const profile = AGENT_PROFILES[i % AGENT_PROFILES.length];
    return {
      id: `agent-${m.userId}`,
      name:
        profile.location === 'local'
          ? `${profile.label} · ${m.displayName} 的设备`
          : profile.label,
      platform: profile.platform,
      location: profile.location,
      ownerId: m.userId,
      owner: m.displayName,
      role: m.role,
      status: i === members.length - 1 && members.length > 3 ? 'offline' : 'online',
      authorization: profile.authorization,
    };
  });
}

interface WorkflowBlueprintNode {
  title: string;
  dependsOn: number[];
  kind?: TenantWorkflowNodeKind;
  executionMode?: 'agent' | 'human';
  role?: string;
  handoff?: string;
}

const CLAWPRO_COLLABORATION_BLUEPRINT: WorkflowBlueprintNode[] = [
  {
    title: '产品开发前端 Demo',
    dependsOn: [],
    role: '产品',
    handoff: '云端 Agent 输出可运行 Demo、交互说明和接口假设。',
  },
  {
    title: '基于 Demo 编写需求单',
    dependsOn: [0],
    role: '产品',
    handoff: '产品确认 Demo 与需求单版本后，向后端 Agent 发布研发基线。',
  },
  {
    title: '后端开发与自测',
    dependsOn: [1],
    role: '后端',
    handoff: '本地 Agent 回传实现、自测和可联调环境，触发前端进入。',
  },
  {
    title: '前端正式开发与联调',
    dependsOn: [2],
    role: '前端',
    handoff: '本地 Agent 完成 Demo 生产化与联调；失败可退回后端修复。',
  },
  {
    title: '产品验收',
    dependsOn: [3],
    executionMode: 'human',
    role: '产品',
    handoff: '产品本人对照 Demo、需求单和正式实现进行验收并确认结论。',
  },
];

function resolveAutoWorkflow(title: string): WorkflowBlueprintNode[] {
  if (/排查|故障|报错|告警/.test(title)) {
    return [
      { title: '问题定位', dependsOn: [] },
      { title: '日志分析', dependsOn: [0] },
      { title: '环境复现', dependsOn: [0] },
      { title: '修复与验证', dependsOn: [1, 2] },
    ];
  }
  if (/文档|报告|调研/.test(title)) {
    return [
      { title: '资料收集', dependsOn: [] },
      { title: '数据分析', dependsOn: [] },
      { title: '内容生成', dependsOn: [0, 1] },
      { title: '审核定稿', dependsOn: [2] },
    ];
  }
  return [
    { title: '目标与约束确认', dependsOn: [] },
    { title: '方案实现', dependsOn: [0] },
    { title: '风险校验', dependsOn: [0] },
    { title: '验收确认', dependsOn: [1, 2] },
  ];
}

function workflowBlueprint(
  templateId: TenantWorkflowTemplateId,
  title: string,
): WorkflowBlueprintNode[] {
  if (templateId === 'none') return [];
  if (templateId === 'auto') return resolveAutoWorkflow(title);
  if (templateId === 'clawpro-collaboration') {
    return CLAWPRO_COLLABORATION_BLUEPRINT;
  }
  const stages =
    TENANT_WORKFLOW_TEMPLATES.find((template) => template.id === templateId)?.stages ?? [];
  let previousStageIndexes: number[] = [];
  let nodeIndex = 0;
  return stages.flatMap((stage) => {
    const nodes = stage.flatMap((node) =>
      Array.from({ length: Math.max(1, node.repeatCount ?? 1) }, (_, index) => ({
        title:
          (node.repeatCount ?? 1) > 1
            ? `${node.title} ${index + 1}`
            : node.title,
        kind: node.kind,
        dependsOn: previousStageIndexes,
      })),
    );
    const currentStageIndexes = nodes.map((_, index) => nodeIndex + index);
    previousStageIndexes = currentStageIndexes;
    nodeIndex += nodes.length;
    return nodes;
  });
}

function workflowStageIndexes(blueprint: WorkflowBlueprintNode[]): number[] {
  const stages: number[] = [];
  blueprint.forEach((node, index) => {
    stages[index] =
      node.dependsOn.length === 0
        ? 0
        : Math.max(...node.dependsOn.map((dependency) => stages[dependency] ?? 0)) + 1;
  });
  return stages;
}

function buildArtifacts(
  taskTitle: string,
  nodeTitle: string,
  nodeId: string,
  handoff?: string | null,
): TenantTaskArtifact[] {
  const artifactByNode: Record<string, { markdownName: string; jsonName: string; summary: string; next: string }> = {
    '产品开发前端 Demo': {
      markdownName: '前端Demo交付说明.md',
      jsonName: 'demo-version.json',
      summary: '已生成可运行前端 Demo，并整理关键交互、Mock 数据与接口假设。',
      next: '基于 Demo 固化需求单',
    },
    '基于 Demo 编写需求单': {
      markdownName: '需求单.md',
      jsonName: '需求基线.json',
      summary: '已基于 Demo 固化业务规则、范围和验收标准，形成产品确认基线。',
      next: '交给后端本地 Agent 开发',
    },
    后端开发与自测: {
      markdownName: '后端实现与自测说明.md',
      jsonName: '可联调交接.json',
      summary: '后端实现和自测已完成，可联调环境与已知限制已同步给前端。',
      next: '触发前端本地 Agent 正式开发与联调',
    },
    前端正式开发与联调: {
      markdownName: '前后端联调结果.md',
      jsonName: '联合交付.json',
      summary: 'Demo 已完成生产化，核心链路联调通过，并记录与原 Demo 的差异。',
      next: '交给产品验收',
    },
    产品验收: {
      markdownName: '产品验收记录.md',
      jsonName: '验收结论.json',
      summary: '已对照 Demo、需求单和正式实现完成验收，结论等待产品本人确认。',
      next: '关闭协作任务',
    },
  };
  const artifact = artifactByNode[nodeTitle];
  const summary = artifact?.summary ?? `已完成${nodeTitle}，并同步关键结论。`;
  return [
    {
      id: `${nodeId}-summary`,
      name: artifact?.markdownName ?? `${nodeTitle}-执行结果.md`,
      type: 'Markdown',
      content: `# ${nodeTitle}\n\n任务：${taskTitle}\n\n${summary}\n\n协作交接：${handoff ?? '已将授权上下文与产物引用同步到下一节点。'}`,
    },
    {
      id: `${nodeId}-handoff`,
      name: artifact?.jsonName ?? `${nodeTitle}-交接信息.json`,
      type: 'JSON',
      content: JSON.stringify(
        {
          task: taskTitle,
          node: nodeTitle,
          status: 'completed',
          handoff: handoff ?? '已向下一节点同步授权上下文与产物引用',
          next: artifact?.next ?? '等待下一节点接收',
          source: nodeTitle === '产品验收'
            ? 'human'
            : nodeTitle.includes('产品')
              ? 'cloud-agent'
              : 'local-agent',
        },
        null,
        2,
      ),
    },
  ];
}

function buildWorkflow(
  taskId: string,
  title: string,
  templateId: TenantWorkflowTemplateId,
  agents: TenantAgent[],
  taskStatus: TenantTaskStatus,
  keepAssignments: boolean,
  customStages?: TenantWorkflowDraftStage[],
): TenantWorkflowNode[] {
  const templateBlueprint = workflowBlueprint(templateId, title);
  let customNodeIndex = 0;
  let previousStageIndexes: number[] = [];
  const blueprint = customStages
    ? customStages.flatMap(stage => {
        const nodes = stage.flatMap(draftNode => {
          const templateIndexMatch = draftNode.id.match(/^draft-node-(\d+)$/);
          const templateIndex = templateIndexMatch
            ? Number(templateIndexMatch[1]) - 1
            : -1;
          const templateStep =
            templateBlueprint[templateIndex] ??
            templateBlueprint.find(node => node.title === draftNode.title);
          return Array.from(
            { length: Math.max(1, draftNode.repeatCount ?? 1) },
            (_, index) => ({
              ...templateStep,
              title:
                (draftNode.repeatCount ?? 1) > 1
                  ? `${draftNode.title} ${index + 1}`
                  : draftNode.title,
              kind: draftNode.kind,
              dependsOn: previousStageIndexes,
            }),
          );
        });
        const currentStageIndexes = nodes.map(
          (_, index) => customNodeIndex + index,
        );
        previousStageIndexes = currentStageIndexes;
        customNodeIndex += nodes.length;
        return nodes;
      })
    : templateBlueprint;
  const nodeIds = blueprint.map((_, index) => `${taskId}-node-${index + 1}`);
  const stageIndexes = workflowStageIndexes(blueprint);
  return blueprint.map((step, index) => {
    const nodeId = `${taskId}-node-${index + 1}`;
    const stage = stageIndexes[index];
    const confirmed = taskStatus === 'done' || (taskStatus !== 'todo' && stage === 0);
    const reviewing = taskStatus === 'review' && stage === 1 && index === stageIndexes.indexOf(1);
    const running =
      taskStatus !== 'todo' &&
      taskStatus !== 'done' &&
      stage === 1 &&
      !(taskStatus === 'review' && reviewing);
    const hasResult = confirmed || reviewing;
    return {
      id: nodeId,
      title: step.title,
      kind: step.kind ?? 'execution',
      dependsOn: step.dependsOn.map((dependency) => nodeIds[dependency]),
      executionMode: step.executionMode ?? 'agent',
      role: step.role ?? null,
      handoff: step.handoff ?? null,
      agentId: step.executionMode === 'human'
        ? null
        : keepAssignments
        ? (agents.find((agent) => step.role && (agent.role ?? '').includes(step.role))?.id ??
          agents[index % Math.max(agents.length, 1)]?.id ??
          null)
        : null,
      status: confirmed ? 'confirmed' : reviewing ? 'review' : running ? 'running' : 'pending',
      result: hasResult ? `${step.title}已执行完成，结果等待或已经项目负责人确认。` : null,
      artifacts: hasResult ? buildArtifacts(title, step.title, nodeId, step.handoff) : [],
    };
  });
}

// ── 流水线模板：默认 SOP + issue 实例化（场景2） ──────────
/** 每个项目初始化两条通用流水线模板（研发工单 / 单元测试） */
function defaultPipelineTemplates(): TenantPipelineTemplate[] {
  return [
    {
      id: 'pl-ticket',
      name: '工单处理流水线',
      description: '面向线上工单的标准 SOP：受理 → 定位 →（日志分析 ∥ 环境复现 并行）→ 修复 → 回归验证。',
      nodes: [
        {
          id: 'n1',
          title: '问题受理',
          dependsOn: [],
          agentRole: '产品',
          promptTemplate: '请阅读工单，给出摘要与严重级别。',
        },
        {
          id: 'n2',
          title: '根因定位',
          dependsOn: ['n1'],
          agentRole: '后端',
          promptTemplate: '基于上游产出的工单摘要定位根因。',
        },
        { id: 'n3', title: '日志分析', dependsOn: ['n2'], agentRole: '后端' },
        { id: 'n4', title: '环境复现', dependsOn: ['n2'], agentRole: '测试' },
        { id: 'n5', title: '修复实现', dependsOn: ['n3', 'n4'], agentRole: '后端' },
        { id: 'n6', title: '回归验证', dependsOn: ['n5'], agentRole: '测试' },
      ],
    },
    {
      id: 'pl-unittest',
      name: '单元测试流水线',
      description: '为核心模块补齐单测的标准流程：用例设计 → 用例实现 → 执行验证。',
      nodes: [
        { id: 'n1', title: '用例设计', dependsOn: [], agentRole: '测试' },
        { id: 'n2', title: '用例实现', dependsOn: ['n1'], agentRole: '研发' },
        { id: 'n3', title: '执行与验证', dependsOn: ['n2'], agentRole: '测试' },
      ],
    },
    {
      id: 'pl-weekly-report',
      name: '每周进度汇报',
      description:
        'PM 每周汇总项目进度：各成员 agent 并行汇报 → 汇总节点合并输出。',
      nodes: [
        {
          id: 'n1',
          title: '成员进度汇报',
          dependsOn: [],
          agentRole: '成员',
          promptTemplate: '汇报本周你在项目里的进展与阻塞。',
        },
        {
          id: 'n2',
          title: '汇总输出汇报',
          dependsOn: ['n1'],
          agentRole: '产品',
          promptTemplate:
            '综合上游各成员产出的进度与阻塞，产出本周项目汇报。',
        },
      ],
    },
  ];
}

/** 按节点预设角色匹配项目内已授权 agent；匹配不到兜底派第一个 agent */
function matchAgentIdByRole(
  agents: TenantAgent[],
  members: TenantProjectMember[],
  role: string,
): string | null {
  const hit = agents.find((a) => {
    const m = members.find((mm) => mm.userId === a.ownerId);
    if (!m) return false;
    return m.role === role || m.role.includes(role) || role.includes(m.role);
  });
  return hit?.id ?? agents[0]?.id ?? null;
}

/**
 * 按流水线模板实例化一条 issue 的 workflow(DAG)。
 * confirmedCount：前若干节点视为已确认（seed 用于分散卡位）；createIssue 传 0。
 * autoStart：把依赖已满足的就绪节点置为 running（模拟 agent 领活执行）。
 */
function instantiatePipelineWorkflow(
  taskId: string,
  template: TenantPipelineTemplate,
  agents: TenantAgent[],
  members: TenantProjectMember[],
  confirmedCount: number,
  autoStart: boolean,
): TenantWorkflowNode[] {
  const idMap: Record<string, string> = {};
  template.nodes.forEach((n, i) => {
    idMap[n.id] = `${taskId}-node-${i + 1}`;
  });
  const confirmedSet = new Set<string>(
    template.nodes.slice(0, confirmedCount).map((n) => idMap[n.id]),
  );
  const titleById: Record<string, string> = {};
  template.nodes.forEach((n) => {
    titleById[idMap[n.id]] = n.title;
  });
  return template.nodes.map((n) => {
    const nodeId = idMap[n.id];
    const dependsOn = n.dependsOn.map((d) => idMap[d]);
    const confirmed = confirmedSet.has(nodeId);
    const depsConfirmed = dependsOn.every((d) => confirmedSet.has(d));
    const running = !confirmed && depsConfirmed && autoStart;
    // 全部派发给 agent：memberAgent/projectAgent 直接落 agentId；
    // 无executor 的旧数据回退按 agentRole 自动匹配一个 agent。
    let agentId: string | null = null;
    if (n.executorKind === 'memberAgent' || n.executorKind === 'projectAgent') {
      agentId = n.executorRef ?? null;
    } else {
      // 回退旧策略：按 agentRole 自动匹配
      agentId = matchAgentIdByRole(agents, members, n.agentRole);
    }
    const result = confirmed
      ? `${n.title}已执行完成并经项目负责人确认。`
      : null;
    return {
      id: nodeId,
      title: n.title,
      dependsOn,
      agentId,
      status: confirmed ? 'confirmed' : running ? 'running' : 'pending',
      result,
      artifacts: confirmed ? buildArtifacts(template.name, n.title, nodeId) : [],
      runtimePhaseId: n.id,
      runtimePrompt: n.promptTemplate,
      configAssets: n.configAssets?.map(asset => ({ ...asset })),
      runtimeInputs: n.inputs,
      runtimeOutputs: n.outputs,
      runtimeApprovalRequired: Boolean(n.approvalRequired),
      runtimeRequiredEvidence: n.requiredEvidence,
      runtimeRejectOutputMarkers: n.rejectOutputMarkers,
      runtimeRequiredCapabilities: n.requiredCapabilities,
      runtimeOnPass: n.onPass,
      runtimeOnFail: n.onFail,
      runtimeDecisionMode: n.decisionMode,
      runtimeMaxRetries: n.maxRetries,
      // 自动数据流：节点产出 = 本节点跑完的 result（零配置，不再依赖手动声明的 outputs 契约）
      outputValues: confirmed
        ? { result: autoOutputValue(nodeId, result ?? '') }
        : undefined,
      // 输入 = 全部上游节点的产出自动汇集
      inputValues: autoInputValues(dependsOn, titleById, confirmedSet),
      type: n.type,
      loopConfig: n.loopConfig
        ? {
            endNodeId: idMap[n.loopConfig.endNodeId] ?? n.loopConfig.endNodeId,
            maxCount: n.loopConfig.maxCount,
            exitCondition: n.loopConfig.exitCondition,
          }
        : undefined,
    };
  });
}

/** 节点自身产出的自动 IO 值 */
function autoOutputValue(nodeId: string, result: string): IOValue {
  return {
    type: 'markdown',
    value: result,
    producedBy: nodeId,
    producedAt: new Date().toISOString(),
  };
}

/** 输入自动 = 所有上游节点的产出汇集（key = 上游 nodeId） */
function autoInputValues(
  dependsOn: string[],
  titleById: Record<string, string>,
  confirmedSet: Set<string>,
): Record<string, IOValue> | undefined {
  if (dependsOn.length === 0) return undefined;
  const entries = dependsOn
    .filter((d) => confirmedSet.has(d))
    .map(
      (d) =>
        [
          d,
          {
            type: 'markdown' as const,
            value: `来自上游「${titleById[d] ?? d}」的产出`,
            producedBy: d,
          },
        ] as const,
    );
  return entries.length > 0 ? Object.fromEntries(entries) : undefined;
}

const PIPELINE_ISSUE_TITLES = [
  '登录接口偶发 502',
  'CVM 列表加载缓慢',
  '导出报表中文乱码',
  '权限校验存在绕过风险',
];

/** seed：为项目生成分散在各阶段的流水线 issue（演示看板列分布） */
function buildPipelineIssues(
  template: TenantPipelineTemplate,
  agents: TenantAgent[],
  members: TenantProjectMember[],
  now: string,
): TenantTask[] {
  const total = template.nodes.length;
  const confirmedCounts = [0, 1, Math.max(0, total - 2), total];
  return confirmedCounts.map((cc, i) => {
    const taskId = `issue-${template.id}-${i}`;
    const workflow = instantiatePipelineWorkflow(
      taskId,
      template,
      agents,
      members,
      cc,
      true,
    );
    const done = cc >= total;
    const title = PIPELINE_ISSUE_TITLES[i % PIPELINE_ISSUE_TITLES.length];
    return {
      id: taskId,
      title,
      description: `【${template.name}】${title}`,
      status: done ? ('done' as const) : ('in_progress' as const),
      ownerId: members[0]?.userId ?? CURRENT_USER,
      priority: PRIORITY_SEED[i % PRIORITY_SEED.length],
      dueDate: `2026-08-${String(6 + i * 2).padStart(2, '0')}`,
      workflowTemplateId: 'auto' as const,
      workflow,
      source: 'import' as const,
      pipelineTemplateId: template.id,
      createdAt: now,
      updatedAt: now,
    };
  });
}

function buildLearnings(idxs: number[], members: TenantProjectMember[]): TenantLearning[] {
  return idxs.map((idx, i) => {
    const tmpl = LEARNING_POOL[idx];
    const m = members[(i + 1) % members.length];
    return {
      id: `lr-${idx}-${i}`,
      title: tmpl.title,
      sourceAgent: `${m.displayName} 的 agent`,
      scene: tmpl.scene,
      tags: tmpl.tags,
      summary: tmpl.summary,
      recalledCount: RECALL_SEED[i % RECALL_SEED.length],
      time: TIME_SEED[i % TIME_SEED.length],
    };
  });
}

function buildTasks(
  offset: number,
  count: number,
  agents: TenantAgent[],
  members: TenantProjectMember[],
  pipelineTemplates: TenantPipelineTemplate[],
  now: string,
): TenantTask[] {
  return Array.from({ length: count }).map((_, i) => {
    const isClawProCollaborationTask = offset === 0 && i === 0;
    const tmpl = isClawProCollaborationTask
      ? {
          title: '项目协作中心产研闭环',
          description: '产品云端 Agent 交付 Demo 与需求单，后端和前端本地 Agent 接力开发、联调，最终由产品本人验收。',
        }
      : TASK_POOL[(offset + i) % TASK_POOL.length];
    const status = isClawProCollaborationTask ? 'in_progress' : TASK_STATUS_SEED[i % TASK_STATUS_SEED.length];
    const taskId = `task-${offset}-${i}`;
    // 全部 seed 任务都走项目工作流模板（挑一条），保证任务详情 DAG 与工作流 Tab 一致
    const template =
      pipelineTemplates[(offset + i) % Math.max(pipelineTemplates.length, 1)];
    const owner = agents[i % Math.max(agents.length, 1)];
    const confirmedCount =
      status === 'done'
        ? template.nodes.length
        : status === 'review'
          ? Math.max(0, template.nodes.length - 1)
          : status === 'in_progress'
            ? Math.max(1, Math.floor(template.nodes.length / 2))
            : 0;
    return {
      id: taskId,
      title: tmpl.title,
      description: tmpl.description,
      status,
      ownerId: owner?.ownerId ?? CURRENT_USER,
      priority: PRIORITY_SEED[i % PRIORITY_SEED.length],
      dueDate: `2026-08-${String(4 + i * 3).padStart(2, '0')}`,
      workflowTemplateId: 'auto' as const,
      workflow: instantiatePipelineWorkflow(
        taskId,
        template,
        agents,
        members,
        confirmedCount,
        status === 'in_progress',
      ),
      source: 'manual' as const,
      pipelineTemplateId: template.id,
      triggerType: 'manual' as const,
      // 演示 tag：轮换分配 需求/缺陷/汇报/评审
      tags: [TENANT_TASK_TAG_PRESETS[i % TENANT_TASK_TAG_PRESETS.length]],
      createdAt: now,
      updatedAt: now,
    };
  });
}

function buildIssueFixDemoTask(
  agents: TenantAgent[],
  members: TenantProjectMember[],
  now: string,
): TenantTask {
  const codeBuddyAgent = agents.find((agent) => agent.platform === 'codebuddy');
  const imateAgent = agents.find((agent) => agent.platform === 'imate');
  return {
    id: TENANT_WORKFLOW_DEMO_TASK_ID,
    title: 'SkillHub IssueFix 多 Agent 协作演示',
    description:
      '固定演示任务：真实运行已跑通的 SkillHub IssueFix 工作流，可选择 CodeBuddy 单 Agent，或 CodeBuddy 与 iMate OpenClaw 多 Agent 交接。',
    status: 'todo',
    ownerId: members[0]?.userId ?? CURRENT_USER,
    priority: 'high',
    dueDate: '2026-12-31',
    workflowTemplateId: 'auto',
    pipelineTemplateId: TENANT_WORKFLOW_DEMO_TEMPLATE_ID,
    runtimeExecutable: true,
    workflow: ISSUEFIX_DEMO_PHASES.map((title, index) => ({
      id: `${TENANT_WORKFLOW_DEMO_TASK_ID}-node-${index + 1}`,
      title,
      kind: 'execution',
      dependsOn:
        index === 0
          ? []
          : [`${TENANT_WORKFLOW_DEMO_TASK_ID}-node-${index}`],
      executionMode: 'agent',
      role: null,
      handoff: '使用 ClawPro Handoff v2 交接节点结论、结构化数据与产物引用。',
      agentId: ISSUEFIX_IMATE_PHASE_INDEXES.has(index)
        ? (imateAgent?.id ?? codeBuddyAgent?.id ?? null)
        : (codeBuddyAgent?.id ?? imateAgent?.id ?? null),
      status: 'pending',
      result: null,
      artifacts: [],
      runtimePhaseId: [
        'analyze',
        'fix',
        'review',
        'test',
        'mr',
        'checkers',
        'verify',
        'close',
      ][index],
      runtimePrompt: `完成 ${title}，承接上游 Handoff v2 的结论与产物。`,
      runtimeOutputs: [
        {
          key: 'summary',
          label: '节点结论',
          type: 'markdown',
          required: true,
        },
        {
          key: 'artifacts',
          label: `${[
            'fix-plan.md',
            'fix-report.md',
            'review-report.md',
            'test-report.md',
            'mr-body.md',
            'checkers-report.md',
            'e2e-report.md',
            'close-report.md',
          ][index]}`,
          type: 'file',
          required: true,
        },
      ],
      runtimeApprovalRequired: index === 0 || index === 3,
    })),
    source: 'manual',
    triggerType: 'manual',
    tags: ['演示', '多 Agent'],
    createdAt: now,
    updatedAt: now,
  };
}

function ensureIssueFixRuntimeAgents(
  agents: TenantAgent[],
  members: TenantProjectMember[],
): TenantAgent[] {
  if (agents.some((agent) => agent.id === TENANT_WORKFLOW_IMATE_AGENT_ID)) {
    return agents;
  }
  const owner = members[0];
  return [
    ...agents,
    {
      id: TENANT_WORKFLOW_IMATE_AGENT_ID,
      name: 'iMate OpenClaw · 真实 Runtime',
      platform: 'imate',
      location: 'cloud',
      ownerId: owner?.userId ?? CURRENT_USER,
      owner: owner?.displayName ?? '项目 Runtime',
      role: '评审 / 测试',
      status: 'online',
      authorization: '通过 iMate 项目与 OpenClaw Agent 授权执行',
      kind: 'project',
    },
  ];
}

function ensureProjectManagementRuntimeAgents(
  agents: TenantAgent[],
  members: TenantProjectMember[],
): TenantAgent[] {
  const owner = members[0];
  const existingIds = new Set(agents.map(agent => agent.id));
  const additions = PROJECT_MANAGEMENT_AGENT_IDS.map((id, index): TenantAgent => ({
    id,
    name: `CodeBuddy · 项目进展 Agent ${String.fromCharCode(65 + index)}`,
    platform: 'codebuddy',
    location: 'local',
    ownerId: owner?.userId ?? CURRENT_USER,
    owner: owner?.displayName ?? '项目 Runtime',
    role: '项目进展分析',
    status: 'online',
    authorization: '通过 TeamAI 创建独立 CodeBuddy ACP 会话',
    kind: 'project',
  }));
  return [...agents, ...additions.filter(agent => !existingIds.has(agent.id))];
}

function buildIssueFixPipelineTemplate(
  agents: TenantAgent[],
): TenantPipelineTemplate {
  const codeBuddyAgent = agents.find((agent) => agent.platform === 'codebuddy');
  const imateAgent = agents.find((agent) => agent.platform === 'imate');
  return {
    id: TENANT_WORKFLOW_DEMO_TEMPLATE_ID,
    name: 'SkillHub IssueFix · CodeBuddy + iMate',
    description:
      '昨日已验证的真实多 Agent 工作流：CodeBuddy 负责分析、修复、MR 和关闭；iMate OpenClaw 负责评审、测试、Checkers 和 E2E 验证。',
    nodes: ISSUEFIX_DEMO_PHASES.map((title, index) => {
      const useIMate = ISSUEFIX_IMATE_PHASE_INDEXES.has(index);
      const executor = useIMate ? imateAgent : codeBuddyAgent;
      const previousId = index === 0 ? null : `issuefix-node-${index}`;
      return {
        id: `issuefix-node-${index + 1}`,
        title,
        dependsOn: previousId ? [previousId] : [],
        agentRole: useIMate ? '评审 / 测试' : '研发',
        executorKind: executor?.kind === 'project' ? 'projectAgent' : 'memberAgent',
        executorRef: executor?.id,
        inputs:
          index === 0
            ? [
                {
                  key: 'task_goal',
                  label: '任务目标',
                  type: 'markdown',
                  required: true,
                },
              ]
            : [
                {
                  key: 'upstream_result',
                  label: '上游节点结论',
                  type: 'markdown',
                  required: true,
                  source: {
                    nodeId: previousId!,
                    outputKey: 'summary',
                  },
                },
              ],
        outputs: [
          {
            key: 'summary',
            label: '节点结论',
            type: 'markdown',
            required: true,
          },
          {
            key: 'artifacts',
            label: '节点产物',
            type: 'file',
          },
        ],
        promptTemplate:
          index === 0
            ? '根据 {{task_goal}} 完成本节点，并按 ClawPro Handoff v2 输出结论与产物。'
            : '承接 {{upstream_result}}，完成当前节点并回传 Handoff v2。',
        approvalRequired: index === 0 || index === 3,
      };
    }),
  };
}

function ensureKnowledgeInspectionAgents(
  agents: TenantAgent[],
  members: TenantProjectMember[],
): TenantAgent[] {
  const owner = members[0];
  const additions: TenantAgent[] = [
    {
      id: KB_READER_AGENT_ID,
      name: '知识库巡检读取 Agent',
      platform: 'clawpro',
      location: 'cloud',
      ownerId: owner?.userId ?? CURRENT_USER,
      owner: owner?.displayName ?? '项目 Runtime',
      role: 'iWiki 读取 / 平台校验',
      status: 'online',
      authorization: 'iWiki MCP 与平台只读 API',
      kind: 'project',
    },
    {
      id: KB_AUDITOR_AGENT_ID,
      name: '知识库巡检审计 Agent',
      platform: 'clawpro',
      location: 'cloud',
      ownerId: owner?.userId ?? CURRENT_USER,
      owner: owner?.displayName ?? '项目 Runtime',
      role: '问题审计 / 报告生成',
      status: 'online',
      authorization: '只读巡检结果与审计上下文',
      kind: 'project',
    },
  ];
  const existingIds = new Set(agents.map((agent) => agent.id));
  return [...agents, ...additions.filter((agent) => !existingIds.has(agent.id))];
}

function buildKnowledgeInspectionPipelineTemplate(): TenantPipelineTemplate {
  return {
    id: TENANT_KB_INSPECTION_TEMPLATE_ID,
    name: '架构师知识库巡检SOP',
    description:
      '架构师视角的 iWiki 知识库巡检：递归扫描 7 个一级域，交叉校验平台事实，识别矛盾、过期、孤岛和架构边界问题，输出只读审计报告；修正必须经 HITL 确认。',
    nodes: [
      {
        id: 'kb-scan',
        title: '全量读取与链接图',
        dependsOn: [],
        agentRole: 'iWiki 读取',
        executorKind: 'projectAgent',
        executorRef: KB_READER_AGENT_ID,
        inputs: [
          {
            key: 'root_docid',
            label: 'iWiki 目录根 docid',
            type: 'text',
            description: '默认 4025707654',
            required: true,
          },
          {
            key: 'domains',
            label: '巡检域',
            type: 'text',
            description: '默认巡检产品知识、运营流程、客户管理、案例库、外部事实源映射、话术库、CHC归档',
            required: true,
          },
        ],
        outputs: [
          { key: 'pages', label: '页面全集', type: 'json', required: true },
          { key: 'link_graph', label: '页面链接图', type: 'json', required: true },
        ],
        promptTemplate:
          '必须通过真实 iWiki MCP 执行只读调用：先调用 iwiki.metadata 读取根文档 4025707654，再调用 iwiki.getSpacePageTree 获取完整目录，并逐页读取正文。输出中必须明确列出实际调用的工具名、根 docid、spaceid、页面总数和至少 3 个页面 docid 作为证据；生成 pages 与 link_graph。没有 MCP、权限或真实返回数据时必须报错，不得生成结构骨架或模拟页面。',
        requiredEvidence: ['iwiki.metadata', 'iwiki.getSpacePageTree', '4025707654'],
        rejectOutputMarkers: ['unavailable', '无法访问', '不可用', '模拟', '结构骨架'],
        requiredCapabilities: ['iwiki.read'],
      },
      {
        id: 'kb-cross-check',
        title: '平台数据交叉校验',
        dependsOn: ['kb-scan'],
        agentRole: '平台校验',
        executorKind: 'projectAgent',
        executorRef: KB_READER_AGENT_ID,
        inputs: [
          {
            key: 'pages',
            label: '页面全集',
            type: 'json',
            source: { nodeId: 'kb-scan', outputKey: 'pages' },
            required: true,
          },
        ],
        outputs: [
          { key: 'facts', label: '事实校验结果', type: 'json', required: true },
          { key: 'mismatches', label: '不一致项', type: 'json', required: true },
        ],
        promptTemplate:
          '先通过真实 iWiki MCP 读取“外部事实源映射”目录及映射页正文，再按映射调用至少一个真实平台只读 API，校验计费、进度、到期或状态事实。输出中必须包含映射页 docid 与标题、实际调用的工具/API 名称、查询对象、返回记录数和一条脱敏事实。没有对应工具或认证时必须使节点失败，不得把 iWiki 文案当成平台实时数据，也不得模拟。',
        requiredEvidence: ['docid', 'GET ', '返回记录数'],
        rejectOutputMarkers: ['模拟'],
        requiredCapabilities: ['iwiki.read', 'platform.read'],
      },
      {
        id: 'kb-identify',
        title: '问题识别与分级',
        dependsOn: ['kb-scan', 'kb-cross-check'],
        agentRole: '知识审计',
        executorKind: 'projectAgent',
        executorRef: KB_AUDITOR_AGENT_ID,
        inputs: [
          {
            key: 'link_graph',
            label: '页面链接图',
            type: 'json',
            source: { nodeId: 'kb-scan', outputKey: 'link_graph' },
            required: true,
          },
          {
            key: 'mismatches',
            label: '平台不一致项',
            type: 'json',
            source: { nodeId: 'kb-cross-check', outputKey: 'mismatches' },
            required: true,
          },
        ],
        outputs: [
          { key: 'issues', label: '分级问题清单', type: 'json', required: true },
        ],
        promptTemplate:
          '综合页面、链接图和平台校验结果，识别矛盾、过期、孤岛问题并按高、中、低、待人工复核分级。',
      },
      {
        id: 'kb-report',
        title: '审计报告合成',
        dependsOn: ['kb-scan', 'kb-cross-check', 'kb-identify'],
        agentRole: '报告生成',
        executorKind: 'projectAgent',
        executorRef: KB_AUDITOR_AGENT_ID,
        inputs: [
          {
            key: 'issues',
            label: '分级问题清单',
            type: 'json',
            source: { nodeId: 'kb-identify', outputKey: 'issues' },
            required: true,
          },
        ],
        outputs: [
          { key: 'report', label: '知识库审计报告', type: 'markdown', required: true },
          { key: 'action_items', label: '建议动作 Top 20', type: 'json', required: true },
        ],
        promptTemplate:
          '按域和严重度生成 Markdown 审计报告与 Top 20 建议动作；只输出建议，不自动修改页面，修正需 HITL 确认。',
        approvalRequired: true,
      },
    ],
  };
}

function buildPreVisitBriefPipelineTemplate(
  _agents: TenantAgent[],
): TenantPipelineTemplate {
  const executorKind = 'projectAgent' as const;
  const rejectMockOutput = ['模拟数据', '仅供演示', '虚构结果'];

  return {
    id: TENANT_PRE_VISIT_BRIEF_TEMPLATE_ID,
    name: '拜访前简报 SOP',
    description:
      '来源《拜访前简报.yml》：按日程校验商机关联，采集客户与历史拜访数据，组装并暂存会前简报；用户确认后再触发企微推送。',
    nodes: [
      {
        id: 'calendar-match',
        title: '刷新日程与商机匹配',
        dependsOn: [],
        agentRole: 'calendar-scan-agent',
        executorKind,
        executorRef: 'calendar-scan-agent',
        inputs: [
          {
            key: 'input',
            label: '用户指令',
            type: 'text',
            required: true,
          },
        ],
        outputs: [
          {
            key: 'match_result',
            label: '日程关联校验结果',
            type: 'text',
            required: true,
          },
          {
            key: 'calendar_match',
            label: 'calendar-match.json',
            type: 'json',
            required: true,
          },
        ],
        promptTemplate:
          '你承担 calendar-scan-agent 职责。读取任务输入中的日程指令，调用已授权的日历与商机能力，校验日程是否存在并刷新日程与商机的关联。输出一句可读结论（例如“检查结果44813，已正确关联商机。”），并生成 calendar-match.json，至少包含 event_id、matched、opportunity_id、opportunity_name 和 checked_at。必须使用真实数据；无法读取时明确失败原因，不得编造关联结果。',
        rejectOutputMarkers: rejectMockOutput,
      },
      {
        id: 'brief-data-collect',
        title: '采集会前简报数据',
        dependsOn: ['calendar-match'],
        agentRole: 'pre-visit-reminder-agent',
        executorKind,
        executorRef: 'pre-visit-reminder-agent',
        inputs: [
          {
            key: 'input',
            label: '用户指令',
            type: 'text',
            required: true,
          },
          {
            key: 'calendar_match',
            label: '日程与商机匹配结果',
            type: 'json',
            required: true,
            source: { nodeId: 'calendar-match', outputKey: 'calendar_match' },
          },
        ],
        outputs: [
          {
            key: 'briefing_candidates',
            label: 'briefing-candidates.json',
            type: 'json',
            required: true,
          },
          {
            key: 'collection_summary',
            label: 'data-collection.md',
            type: 'markdown',
            required: true,
          },
        ],
        promptTemplate:
          '你承担 pre-visit-reminder-agent 职责。结合用户指令和上游 calendar-match.json，从已授权的数据源采集客户、商机、拜访记录、跟进日志、行动项和风险信息。生成 briefing-candidates.json，结构需包含 status、stage、scanned、collected、source、candidates；每个 candidate 至少包含 event_id、recipient、opp_id、opp_name、subject、raw 和 confirm_context_base。另生成 data-collection.md 汇总采集数量、跳过原因和数据来源。产物正文不得携带 HTTP URL，必须保留真实数据来源和异常，不得生成模拟客户信息。',
        rejectOutputMarkers: rejectMockOutput,
      },
      {
        id: 'brief-message-assemble',
        title: '组装并暂存会前简报',
        dependsOn: ['brief-data-collect'],
        agentRole: 'message-assemble-agent',
        executorKind,
        executorRef: 'message-assemble-agent',
        inputs: [
          {
            key: 'input',
            label: '用户指令',
            type: 'text',
            required: true,
          },
          {
            key: 'briefing_candidates',
            label: '会前简报候选数据',
            type: 'json',
            required: true,
            source: {
              nodeId: 'brief-data-collect',
              outputKey: 'briefing_candidates',
            },
          },
        ],
        outputs: [
          {
            key: 'message_id',
            label: '待发送消息 ID',
            type: 'text',
            required: true,
          },
          {
            key: 'confirm_hint_url',
            label: '确认链接',
            type: 'url',
            required: true,
          },
          {
            key: 'briefing',
            label: 'pre-visit-brief.md',
            type: 'markdown',
            required: true,
          },
          {
            key: 'message_draft',
            label: 'message-draft.json',
            type: 'json',
            required: true,
          },
        ],
        promptTemplate:
          '你承担 message-assemble-agent 职责。使用 briefing-candidates.json 生成可直接阅读的会前简报，必须包含商机名称与编号、金额和状态、拜访时间与人员、本次议题、历史拜访回顾、AI 建议、风险提醒和本次拜访建议。调用已授权的消息能力把消息写入后台暂存为 draft/held，禁止在本节点发送。输出真实 message_id、完整简报内容和 push-message 产生的 confirm_hint URL；生成 pre-visit-brief.md 与 message-draft.json。不要解释处理过程，不得伪造消息 ID 或确认链接。',
        approvalRequired: true,
        rejectOutputMarkers: rejectMockOutput,
      },
      {
        id: 'brief-message-notify',
        title: '确认后触发企微推送',
        dependsOn: ['brief-message-assemble'],
        agentRole: 'message-notify-agent',
        executorKind,
        executorRef: 'message-notify-agent',
        inputs: [
          {
            key: 'message_id',
            label: '待发送消息 ID',
            type: 'text',
            required: true,
            source: {
              nodeId: 'brief-message-assemble',
              outputKey: 'message_id',
            },
          },
          {
            key: 'message_draft',
            label: '消息草稿',
            type: 'json',
            required: true,
            source: {
              nodeId: 'brief-message-assemble',
              outputKey: 'message_draft',
            },
          },
          {
            key: 'briefing',
            label: '会前简报正文',
            type: 'markdown',
            required: true,
            source: {
              nodeId: 'brief-message-assemble',
              outputKey: 'briefing',
            },
          },
        ],
        outputs: [
          {
            key: 'dispatch_result',
            label: 'dispatch-result.json',
            type: 'json',
            required: true,
          },
          {
            key: 'final_briefing',
            label: 'pre-visit-brief-final.md',
            type: 'markdown',
            required: true,
          },
        ],
        promptTemplate:
          '你承担 message-notify-agent 职责。本节点只会在用户确认上游简报后执行。读取真实 message_id，调用已授权的消息推送能力触发发送，并回查消息状态。生成 dispatch-result.json，至少包含 message_id、recipient、previous_status、current_status、triggered、checked_at；生成 pre-visit-brief-final.md，完整保留最终简报正文并附推送结论。只有后端真实返回 draft→pending/sent 才能报告触发成功，失败时保留原状态并明确错误。',
        rejectOutputMarkers: rejectMockOutput,
      },
    ],
  };
}

function buildPublicImageReleasePipelineTemplate(): TenantPipelineTemplate {
  const stageOutput = (key: string, label: string, type: IOFieldType = 'markdown') => [
    { key, label, type, required: true },
  ];
  return {
    id: TENANT_IMAGE_RELEASE_TEMPLATE_ID,
    name: 'ClawPro 公共镜像发布跟进',
    description:
      '来源《小镜同学 - ClawPro 公共镜像发布上线全流程跟进》：TAPD 初始化、镜像制作、四方并行 QA、发布确认、发布、版本记录与完成归档。',
    nodes: [
      {
        id: 'TAPD_INIT',
        title: '需求初始化',
        dependsOn: [],
        agentRole: '产品 / 需求初始化',
        inputs: [
          { key: 'image', label: '镜像类型', type: 'text', required: true },
          { key: 'version', label: '镜像版本', type: 'text', required: true },
          { key: 'tapd_url', label: 'TAPD 需求', type: 'url', required: true },
        ],
        outputs: stageOutput('tapd_init', '需求初始化记录'),
        promptTemplate:
          '核对镜像、版本和 TAPD 三要素，整理需求背景、变更点、影响面、接口人与期望上线时间；生成可追溯的需求初始化记录。不得实际发送企微消息。',
        approvalRequired: true,
      },
      {
        id: 'WAIT_BUILD',
        title: '镜像制作',
        dependsOn: ['TAPD_INIT'],
        agentRole: '镜像制作',
        inputs: [{ key: 'tapd_init', label: '需求初始化记录', type: 'markdown', required: true }],
        outputs: stageOutput('image_build', '镜像制品记录', 'json'),
        promptTemplate:
          '承接需求初始化记录，整理镜像 ID、下载地址、checksum 与变更清单；仅生成镜像制品记录，不执行真实构建或发布。',
        approvalRequired: true,
      },
      {
        id: 'WAIT_CLAWPRO_CHECK',
        title: 'ClawPro 主流程测试',
        dependsOn: ['WAIT_BUILD'],
        agentRole: 'ClawPro 测试',
        inputs: [{ key: 'image_build', label: '镜像制品记录', type: 'json', required: true }],
        outputs: stageOutput('qa_clawpro', 'ClawPro 测试报告'),
        promptTemplate:
          '基于镜像制品记录制定并执行隔离环境中的 ClawPro 主流程回归，输出范围、证据、结论与阻塞项；不得修改真实发布环境。',
        approvalRequired: true,
      },
      {
        id: 'WAIT_CLS_CHECK',
        title: 'CLS 插件测试',
        dependsOn: ['WAIT_BUILD'],
        agentRole: 'CLS 测试',
        inputs: [{ key: 'image_build', label: '镜像制品记录', type: 'json', required: true }],
        outputs: stageOutput('qa_cls', 'CLS 测试报告'),
        promptTemplate:
          '基于同一镜像制品并行验证 CLS 插件核心功能，输出范围、证据、结论与阻塞项；不得修改真实发布环境。',
        approvalRequired: true,
      },
      {
        id: 'WAIT_AGENTCHAT_CHECK',
        title: 'agentchat 插件测试',
        dependsOn: ['WAIT_BUILD'],
        agentRole: 'agentchat 测试',
        inputs: [{ key: 'image_build', label: '镜像制品记录', type: 'json', required: true }],
        outputs: stageOutput('qa_agentchat', 'agentchat 测试报告'),
        promptTemplate:
          '基于同一镜像制品并行验证 agentchat 插件核心功能，输出范围、证据、结论与阻塞项；不得修改真实发布环境。',
        approvalRequired: true,
      },
      {
        id: 'WAIT_MEMORY_CHECK',
        title: 'memory 插件测试',
        dependsOn: ['WAIT_BUILD'],
        agentRole: 'memory 测试',
        inputs: [{ key: 'image_build', label: '镜像制品记录', type: 'json', required: true }],
        outputs: stageOutput('qa_memory', 'memory 测试报告'),
        promptTemplate:
          '基于同一镜像制品并行验证 memory 插件核心功能，输出范围、证据、结论与阻塞项；不得修改真实发布环境。',
        approvalRequired: true,
      },
      {
        id: 'WAIT_RELEASE_CONFIRM',
        title: '发布确认',
        dependsOn: [
          'WAIT_CLAWPRO_CHECK',
          'WAIT_CLS_CHECK',
          'WAIT_AGENTCHAT_CHECK',
          'WAIT_MEMORY_CHECK',
        ],
        agentRole: '发布确认',
        inputs: [
          { key: 'qa_clawpro', label: 'ClawPro 测试报告', type: 'markdown', required: true },
          { key: 'qa_cls', label: 'CLS 测试报告', type: 'markdown', required: true },
          { key: 'qa_agentchat', label: 'agentchat 测试报告', type: 'markdown', required: true },
          { key: 'qa_memory', label: 'memory 测试报告', type: 'markdown', required: true },
        ],
        outputs: stageOutput('release_confirm', '4/4 QA 汇总结论'),
        promptTemplate:
          '必须同时承接四份 QA 报告，逐项列出通过或阻塞状态；只有 4/4 全部通过才给出发布放行建议，否则明确保持阻塞。',
        approvalRequired: true,
      },
      {
        id: 'WAIT_RELEASE',
        title: '公共镜像发布',
        dependsOn: ['WAIT_RELEASE_CONFIRM'],
        agentRole: '公共镜像发布',
        inputs: [{ key: 'release_confirm', label: '4/4 QA 汇总结论', type: 'markdown', required: true }],
        outputs: stageOutput('release_result', '发布结果', 'json'),
        promptTemplate:
          '根据已确认的放行结论生成发布执行计划与结果记录，覆盖 Test、Grey、Online、跨地域同步和旧版本 Offline；POC 不操作真实发布系统。',
        approvalRequired: true,
      },
      {
        id: 'WAIT_RECORD',
        title: '更新 ClawPro 使用版本',
        dependsOn: ['WAIT_RELEASE'],
        agentRole: '版本记录',
        inputs: [{ key: 'release_result', label: '发布结果', type: 'json', required: true }],
        outputs: stageOutput('version_record', '版本切换与文档记录'),
        promptTemplate:
          '承接发布结果，生成 ClawPro 默认版本切换、版本变更文档和使用方通知记录；POC 不修改真实配置或文档。',
        approvalRequired: true,
      },
      {
        id: 'DONE',
        title: '发布完成归档',
        dependsOn: ['WAIT_RECORD'],
        agentRole: '流程归档',
        inputs: [{ key: 'version_record', label: '版本切换与文档记录', type: 'markdown', required: true }],
        outputs: stageOutput('release_summary', '发布全流程总结'),
        promptTemplate:
          '汇总全部阶段、并行 QA 结果、阶段耗时、阻塞与人工确认记录，生成最终发布跟进总结和可追溯状态摘要。',
      },
    ],
  };
}

function buildStandardDevelopmentSopPipelineTemplate(
  agents: TenantAgent[],
): TenantPipelineTemplate {
  const codeBuddyAgent = agents.find((agent) => agent.platform === 'codebuddy');
  const steps = [
    {
      id: 'sop-clarify',
      title: '1. Clarify',
      artifact: '01-clarify.md',
      role: '需求澄清',
      instruction:
        '先恢复当前任务上下文并执行分支与任务目录前置判断。若 01-clarify.md 已有有效内容，取得用户确认后在 00-overview.md 标记跳过并直接进入 Plan；否则澄清背景、目标、范围和待确认问题。',
    },
    {
      id: 'sop-plan',
      title: '2. Plan',
      artifact: '02-plan.md',
      role: '研发规划',
      instruction:
        '输出改动文件、调用链、风险、集成测试用例，并在 §6 完整定义 TDD 单元测试用例。协同开发模式需从 design.md 完整派生，禁止只写“详见 design.md”。',
    },
    {
      id: 'sop-implement',
      title: '3. Implement',
      artifact: '03-implement.md',
      role: '研发实现',
      instruction:
        '严格按 Plan §6 执行红绿循环：先写单元测试并确认跑红，再做最小实现转绿，最后重构；记录关键改动、偏差与原因。',
    },
    {
      id: 'sop-ut',
      title: '4. UT',
      artifact: '04-ut.md',
      role: '单元测试',
      instruction:
        '逐条对齐 Plan §6 的单元测试用例，记录实际结果、覆盖率与未覆盖行；失败项不得静默跳过。',
    },
    {
      id: 'sop-deploy',
      title: '5. Deploy',
      artifact: '05-deploy.md',
      role: '测试部署',
      instruction:
        '通过 cvm-dev-workflow 将代码热更到测试环境，记录目标环境、版本、部署结果和回滚方式。',
    },
    {
      id: 'sop-it',
      title: '6. IT',
      artifact: '06-it.md',
      role: '集成测试',
      instruction:
        '执行集成测试，每条用例附关键日志和 reqid；协同开发模式也不得跳过，失败需记录复现条件。',
    },
    {
      id: 'sop-docs',
      title: '7. Docs',
      artifact: '07-docs.md',
      role: '文档维护',
      instruction:
        '按本次真实改动增量更新 .specs/docs/，保持实现、测试结果与文档一致。',
    },
    {
      id: 'sop-review',
      title: '8. Review',
      artifact: '08-review.md',
      role: '代码审查',
      instruction:
        '执行代码审查，记录问题级别、位置、处理结论、遗留风险和是否允许进入提交阶段。',
    },
    {
      id: 'sop-commit',
      title: '9. Commit',
      artifact: '09-commit.md',
      role: '提交交付',
      instruction:
        '严格按顺序执行：释放测试环境、写 09-commit.md、更新 00-overview.md、git add 代码与 plans、git commit、git push。提交格式为 <type>(<scope>): <subject>，并带 --(bug|story|task|test|other)=数字 脚注。',
    },
  ];

  return {
    id: TENANT_STANDARD_DEV_SOP_TEMPLATE_ID,
    name: 'vstation开发标准SOP',
    description:
      '基于 vstation/api 真实仓库的 Clarify → Plan → Implement → UT → Deploy → IT → Docs → Review → Commit 标准研发流程。每步完成后需人工确认才继续。',
    nodes: steps.map((step, index) => {
      const previous = index === 0 ? null : steps[index - 1];
      return {
        id: step.id,
        title: step.title,
        dependsOn: previous ? [previous.id] : [],
        agentRole: step.role,
        executorKind:
          codeBuddyAgent?.kind === 'project' ? 'projectAgent' : 'memberAgent',
        executorRef: codeBuddyAgent?.id,
        inputs:
          index === 0
            ? [
                {
                  key: 'repository_url',
                  label: '真实仓库',
                  type: 'url',
                  required: true,
                },
                {
                  key: 'requirement',
                  label: '用户原始需求',
                  type: 'markdown',
                  required: true,
                },
                {
                  key: 'current_branch',
                  label: '当前 Git 分支',
                  type: 'text',
                  required: true,
                },
              ]
            : [
                {
                  key: 'upstream_summary',
                  label: '上一步核心结论',
                  type: 'markdown',
                  required: true,
                  source: {
                    nodeId: previous!.id,
                    outputKey: 'summary',
                  },
                },
                {
                  key: 'upstream_artifact',
                  label: '上一步阶段文档',
                  type: 'file',
                  required: true,
                  source: {
                    nodeId: previous!.id,
                    outputKey: 'stage_artifact',
                  },
                },
              ],
        outputs: [
          {
            key: 'summary',
            label: '步骤核心结论',
            type: 'markdown',
            required: true,
          },
          {
            key: 'stage_artifact',
            label: step.artifact,
            type: 'file',
            description: `本步骤必须更新 ${step.artifact}`,
            required: true,
          },
          {
            key: 'overview',
            label: '00-overview.md',
            type: 'file',
            description: '记录步骤开始/结束时间、状态、耗时和下一步',
            required: true,
          },
        ],
        promptTemplate:
          `执行 ${step.title}。${step.instruction} ` +
          '严格遵循五段式：先取得用户开始确认，立即用 date "+%Y-%m-%d %H:%M:%S" 记录开始时间，执行本步骤，立即记录结束时间，展示完成结论与耗时并等待用户结束确认；未经确认不得进入下一步。',
        approvalRequired: true,
      };
    }),
  };
}

/**
 * 对齐 contest_school_registry_20260810_1018 实际执行包的完整研发流水线。
 * 节点职责、输入和文件产物保持通用，不绑定原 SkillHub 业务代码。
 */
function buildMultiAgentDevelopmentPipelineTemplate(
  agents: TenantAgent[],
): TenantPipelineTemplate {
  const codeBuddyAgent = agents.find(agent => agent.platform === 'codebuddy');
  const imateAgent = agents.find(agent => agent.platform === 'imate');
  const codeBuddyExecutorKind =
    codeBuddyAgent?.kind === 'project' ? 'projectAgent' : 'memberAgent';
  const imateExecutorKind =
    imateAgent?.kind === 'project' ? 'projectAgent' : 'memberAgent';

  return {
    id: TENANT_MULTI_AGENT_DEV_TEMPLATE_ID,
    name: '多 Agent 研发交付 SOP',
    description:
      '对齐 contest_school_registry 执行包：PHASE-0 初始化并分流，small 进入 SOLO 轻量交付，medium/large 进入 TASK-01～05 完整研发链路，CODE-REVIEW 失败会打回 TASK-03，最终统一汇总。',
    nodes: [
      {
        id: 'PHASE-0',
        title: 'PHASE-0 · 任务初始化与分级',
        configAssets: cloneMultiAgentNodeAssets('PHASE-0'),
        dependsOn: [],
        agentRole: '工作流调度',
        executorKind: codeBuddyExecutorKind,
        executorRef: codeBuddyAgent?.id,
        inputs: [
          { key: 'requirement', label: '原始需求', type: 'markdown', required: true },
          { key: 'repository_url', label: '指定源码仓库', type: 'url', required: false },
        ],
        outputs: [
          { key: 'workflow_state', label: 'workflow-state.json', type: 'json', required: true },
          { key: 'base_requirement_report', label: '01-requirement/requirement-report.md', type: 'file', required: true },
        ],
        promptTemplate:
          '读取用户需求；用户指定了源码仓库时以该仓库为准，否则优先使用当前项目已绑定的工作区。由 TeamAI 自动定位仓库、当前分支和工作区根目录，并根据任务标题与时间生成 task_slug；按配置资产中的三个维度判断 size_class，生成 workflow-state.json 和 01-requirement/requirement-report.md 两份必需产物。状态文件必须包含 task_id、task_slug、workspace、artifacts_dir、current_stage、next_target、stages、decisions、summary，基础需求报告必须包含原始需求、已知验收标准、仓库定位和初步影响范围。没有仓库时允许先完成需求澄清和分级，但进入代码分析或开发前仍无法定位真实工作区必须返回 blocked。只做初始化，不提前完成后续节点。最终回复必须单独输出机器可判定标记 SIZE_CLASS: SMALL、SIZE_CLASS: MEDIUM 或 SIZE_CLASS: LARGE。',
        onPass: 'TASK-01',
        onFail: 'SOLO',
        decisionMode: 'size_class',
      },
      {
        id: 'SOLO',
        title: 'SOLO · 小需求独立开发',
        configAssets: cloneMultiAgentNodeAssets('SOLO'),
        dependsOn: ['PHASE-0'],
        agentRole: '轻量开发',
        executorKind: codeBuddyExecutorKind,
        executorRef: codeBuddyAgent?.id,
        inputs: [
          { key: 'workflow_state', label: '工作流状态', type: 'json', source: { nodeId: 'PHASE-0', outputKey: 'workflow_state' }, required: true },
          { key: 'base_requirement_report', label: '基础需求报告', type: 'file', source: { nodeId: 'PHASE-0', outputKey: 'base_requirement_report' }, required: true },
          { key: 'requirement', label: '原始需求', type: 'markdown', required: true },
          { key: 'repository_url', label: '指定源码仓库', type: 'url', required: false },
        ],
        outputs: [
          { key: 'solo_report', label: '01-solo/solo-report.md', type: 'file', required: true },
          { key: 'solo_api_docs', label: '01-solo/api-docs.md', type: 'file', required: false },
          { key: 'solo_knowledge', label: 'knowledge/{task_slug}.md', type: 'file', required: true },
        ],
        promptTemplate:
          '仅处理 workflow-state.json 中 size_class=small 的低风险单模块需求。在真实源码工作区中串行完成需求理解、简化设计、最小代码修改和真实检查；严格执行节点配置资产的文件白名单与范围规则。生成 01-solo/solo-report.md 和 knowledge/{task_slug}.md；涉及 HTTP API 变化时再生成 01-solo/api-docs.md。记录真实改动、命令、结果、风险及可复用经验。',
        approvalRequired: true,
        onPass: 'SUMMARY',
      },
      {
        id: 'TASK-01',
        title: 'TASK-01 · 需求分析',
        configAssets: cloneMultiAgentNodeAssets('TASK-01'),
        dependsOn: ['PHASE-0'],
        agentRole: '需求分析',
        executorKind: codeBuddyExecutorKind,
        executorRef: codeBuddyAgent?.id,
        inputs: [
          { key: 'workflow_state', label: '工作流状态', type: 'json', source: { nodeId: 'PHASE-0', outputKey: 'workflow_state' }, required: true },
          { key: 'base_requirement_report', label: '基础需求报告', type: 'file', source: { nodeId: 'PHASE-0', outputKey: 'base_requirement_report' }, required: true },
          { key: 'requirement', label: '原始需求', type: 'markdown', required: true },
          { key: 'repository_url', label: '指定源码仓库', type: 'url', required: false },
        ],
        outputs: [
          { key: 'requirement_report', label: '01-requirement/requirement-report.md', type: 'file', required: true },
        ],
        promptTemplate:
          '读取真实仓库和原始需求，输出 01-requirement/requirement-report.md。必须包含背景、目标、范围、非目标、现状代码证据、验收标准、风险、待确认项和 YAGNI 排除项；不得虚构代码或接口。完成后由编排器更新 workflow-state.json。',
        approvalRequired: true,
        onPass: 'TASK-02',
      },
      {
        id: 'TASK-02',
        title: 'TASK-02 · 技术方案与执行计划',
        configAssets: cloneMultiAgentNodeAssets('TASK-02'),
        dependsOn: ['TASK-01'],
        agentRole: '架构设计',
        executorKind: imateExecutorKind,
        executorRef: imateAgent?.id ?? codeBuddyAgent?.id,
        inputs: [
          { key: 'requirement_report', label: '需求分析报告', type: 'file', source: { nodeId: 'TASK-01', outputKey: 'requirement_report' }, required: true },
          { key: 'workflow_state', label: '工作流状态', type: 'json', source: { nodeId: 'PHASE-0', outputKey: 'workflow_state' }, required: true },
        ],
        outputs: [
          { key: 'tech_design', label: '02-design/tech-design.md', type: 'file', required: true },
          { key: 'execution_plan', label: '02-design/execution-plan.md', type: 'file', required: true },
        ],
        promptTemplate:
          '基于需求报告和真实代码证据输出技术方案与可执行计划。tech-design.md 必须包含调用链、数据结构、兼容性、安全、性能、失败处理和取舍；execution-plan.md 必须包含按文件拆分的任务、依赖、文件白名单、接口/签名契约、测试计划和并行边界。禁止只写原则性建议。',
        approvalRequired: true,
        onPass: 'TASK-03',
      },
      {
        id: 'TASK-03',
        title: 'TASK-03 · 代码实现与接口文档',
        configAssets: cloneMultiAgentNodeAssets('TASK-03'),
        dependsOn: ['TASK-02'],
        agentRole: '研发实现',
        executorKind: codeBuddyExecutorKind,
        executorRef: codeBuddyAgent?.id,
        inputs: [
          { key: 'requirement_report', label: '需求分析报告', type: 'file', source: { nodeId: 'TASK-01', outputKey: 'requirement_report' }, required: true },
          { key: 'tech_design', label: '技术方案', type: 'file', source: { nodeId: 'TASK-02', outputKey: 'tech_design' }, required: true },
          { key: 'execution_plan', label: '执行计划', type: 'file', source: { nodeId: 'TASK-02', outputKey: 'execution_plan' }, required: true },
          { key: 'review_report', label: '上一轮评审意见（首次为空）', type: 'file', source: { nodeId: 'CODE-REVIEW', outputKey: 'review_report' } },
        ],
        outputs: [
          { key: 'change_report', label: '03-code/change-report.md', type: 'file', required: true },
          { key: 'api_docs', label: '03-code/api-docs.md（接口变更时）', type: 'file', required: false },
        ],
        promptTemplate:
          '严格按 execution-plan 的文件白名单在真实源码仓库完成最小实现和自测，不得越界修改；若输入中存在上一轮 review-report.md，必须逐项返工并记录关闭证据。change-report.md 记录实际改动、文件清单、方案映射、自检命令与结果、偏差、遗留项；仅在涉及 HTTP API 新增、修改或删除时生成 api-docs.md，并记录完整接口契约。',
        onPass: 'CODE-REVIEW',
      },
      {
        id: 'CODE-REVIEW',
        title: 'CODE-REVIEW · 独立评审与返工裁决',
        configAssets: cloneMultiAgentNodeAssets('CODE-REVIEW'),
        dependsOn: ['TASK-03'],
        agentRole: '代码审查',
        executorKind: imateExecutorKind,
        executorRef: imateAgent?.id ?? codeBuddyAgent?.id,
        inputs: [
          { key: 'requirement_report', label: '需求分析报告', type: 'file', source: { nodeId: 'TASK-01', outputKey: 'requirement_report' }, required: true },
          { key: 'tech_design', label: '技术方案', type: 'file', source: { nodeId: 'TASK-02', outputKey: 'tech_design' }, required: true },
          { key: 'execution_plan', label: '执行计划', type: 'file', source: { nodeId: 'TASK-02', outputKey: 'execution_plan' }, required: true },
          { key: 'change_report', label: '变更报告', type: 'file', source: { nodeId: 'TASK-03', outputKey: 'change_report' }, required: true },
          { key: 'api_docs', label: '接口文档（接口变更时）', type: 'file', source: { nodeId: 'TASK-03', outputKey: 'api_docs' }, required: false },
        ],
        outputs: [
          { key: 'review_report', label: '03-code/review-report.md', type: 'file', required: true },
        ],
        promptTemplate:
          '独立审查真实 diff 与上游产物，覆盖正确性、安全、性能、兼容性、测试充分性和文件白名单。P0/P1 必须形成可复现证据和修复建议。03-code/review-report.md 必须保留本轮问题、修复状态和未关闭风险；最终回复第一行必须严格输出 REVIEW_VERDICT: PASSED 或 REVIEW_VERDICT: FAILED。FAILED 时编排器打回 TASK-03，最多返工 2 次（共 3 轮评审）。',
        approvalRequired: true,
        onPass: 'TASK-04',
        onFail: 'TASK-03',
        decisionMode: 'review_verdict',
        maxRetries: 2,
      },
      {
        id: 'TASK-04',
        title: 'TASK-04 · 单元测试与 E2E 验证',
        configAssets: cloneMultiAgentNodeAssets('TASK-04'),
        dependsOn: ['CODE-REVIEW'],
        agentRole: '测试验证',
        executorKind: imateExecutorKind,
        executorRef: imateAgent?.id ?? codeBuddyAgent?.id,
        inputs: [
          { key: 'tech_design', label: '技术方案', type: 'file', source: { nodeId: 'TASK-02', outputKey: 'tech_design' }, required: true },
          { key: 'execution_plan', label: '执行计划', type: 'file', source: { nodeId: 'TASK-02', outputKey: 'execution_plan' }, required: true },
          { key: 'change_report', label: '变更报告', type: 'file', source: { nodeId: 'TASK-03', outputKey: 'change_report' }, required: true },
          { key: 'review_report', label: '审查报告', type: 'file', source: { nodeId: 'CODE-REVIEW', outputKey: 'review_report' }, required: true },
          { key: 'workflow_state', label: '工作流状态', type: 'json', source: { nodeId: 'PHASE-0', outputKey: 'workflow_state' }, required: true },
        ],
        outputs: [
          { key: 'test_report', label: '04-e2e/test-report.md', type: 'file', required: true },
          { key: 'added_cases', label: '04-e2e/added-cases.md', type: 'file', required: true },
        ],
        promptTemplate:
          '按验收标准、技术方案和审查问题补齐测试并执行真实验证。test-report.md 必须记录实际命令、通过/失败/跳过数、关键断言、覆盖率或无法统计原因、E2E 是否真实执行和未覆盖项；added-cases.md 必须按功能、边界、异常、回归列出新增用例与结果。不得把静态检查写成已跑通 E2E。',
        approvalRequired: true,
        onPass: 'TASK-05',
      },
      {
        id: 'TASK-05',
        title: 'TASK-05 · 知识沉淀',
        configAssets: cloneMultiAgentNodeAssets('TASK-05'),
        dependsOn: ['TASK-04'],
        agentRole: '知识沉淀',
        executorKind: codeBuddyExecutorKind,
        executorRef: codeBuddyAgent?.id,
        inputs: [
          { key: 'requirement_report', label: '需求分析报告', type: 'file', source: { nodeId: 'TASK-01', outputKey: 'requirement_report' }, required: true },
          { key: 'tech_design', label: '技术方案', type: 'file', source: { nodeId: 'TASK-02', outputKey: 'tech_design' }, required: true },
          { key: 'execution_plan', label: '执行计划', type: 'file', source: { nodeId: 'TASK-02', outputKey: 'execution_plan' }, required: true },
          { key: 'change_report', label: '变更报告', type: 'file', source: { nodeId: 'TASK-03', outputKey: 'change_report' }, required: true },
          { key: 'review_report', label: '审查报告', type: 'file', source: { nodeId: 'CODE-REVIEW', outputKey: 'review_report' }, required: true },
          { key: 'test_report', label: '测试报告', type: 'file', source: { nodeId: 'TASK-04', outputKey: 'test_report' }, required: true },
          { key: 'added_cases', label: '新增用例', type: 'file', source: { nodeId: 'TASK-04', outputKey: 'added_cases' }, required: true },
          { key: 'workflow_state', label: '工作流状态', type: 'json', source: { nodeId: 'PHASE-0', outputKey: 'workflow_state' }, required: true },
        ],
        outputs: [
          { key: 'knowledge', label: 'knowledge/{task_slug}.md', type: 'file', required: true },
        ],
        promptTemplate:
          '从需求、方案、实现、审查和测试产物中提炼可复用且可验证的业务知识、技术决策、改造模式、踩坑、流程改进和遗留项，写入 knowledge/{task_slug}.md。对既有知识做重复检测，只追加或合并，不得覆盖历史内容，不得编造根因。',
        onPass: 'SUMMARY',
      },
      {
        id: 'SUMMARY',
        title: 'SUMMARY · 最终汇总与上线裁决',
        configAssets: cloneMultiAgentNodeAssets('SUMMARY'),
        dependsOn: ['SOLO', 'TASK-05'],
        agentRole: '交付汇总',
        executorKind: codeBuddyExecutorKind,
        executorRef: codeBuddyAgent?.id,
        inputs: [
          { key: 'solo_report', label: '小需求交付报告', type: 'file', source: { nodeId: 'SOLO', outputKey: 'solo_report' }, required: false },
          { key: 'solo_knowledge', label: '小需求知识沉淀', type: 'file', source: { nodeId: 'SOLO', outputKey: 'solo_knowledge' }, required: false },
          { key: 'requirement_report', label: '需求报告', type: 'file', source: { nodeId: 'TASK-01', outputKey: 'requirement_report' }, required: false },
          { key: 'tech_design', label: '技术方案', type: 'file', source: { nodeId: 'TASK-02', outputKey: 'tech_design' }, required: false },
          { key: 'execution_plan', label: '执行计划', type: 'file', source: { nodeId: 'TASK-02', outputKey: 'execution_plan' }, required: false },
          { key: 'change_report', label: '变更报告', type: 'file', source: { nodeId: 'TASK-03', outputKey: 'change_report' }, required: false },
          { key: 'api_docs', label: '接口文档', type: 'file', source: { nodeId: 'TASK-03', outputKey: 'api_docs' }, required: false },
          { key: 'review_report', label: '审查报告', type: 'file', source: { nodeId: 'CODE-REVIEW', outputKey: 'review_report' }, required: false },
          { key: 'test_report', label: '测试报告', type: 'file', source: { nodeId: 'TASK-04', outputKey: 'test_report' }, required: false },
          { key: 'added_cases', label: '新增用例', type: 'file', source: { nodeId: 'TASK-04', outputKey: 'added_cases' }, required: false },
          { key: 'knowledge', label: '知识沉淀', type: 'file', source: { nodeId: 'TASK-05', outputKey: 'knowledge' }, required: false },
          { key: 'workflow_state', label: '工作流状态', type: 'json', source: { nodeId: 'PHASE-0', outputKey: 'workflow_state' }, required: true },
          { key: 'reference_artifacts', label: '压缩包产物基准', type: 'json', required: true },
        ],
        outputs: [
          { key: 'workflow_summary', label: 'workflow-summary.md', type: 'file', required: true },
          { key: 'workflow_state_final', label: 'workflow-state.json', type: 'json', required: true },
        ],
        promptTemplate:
          '先读取 workflow-state.json 的 size_class 和实际完成节点：small 只汇总 SOLO 产物，medium/large 汇总 TASK-01～05 与 CODE-REVIEW 产物，忽略已跳过分支。workflow-summary.md 必须包含一句话结论、任务信息、完整产物索引、变更摘要、实际阶段结论和指标、遗留项、上线硬性检查、回滚约束与后续提醒。medium/large 分支还需读取 reference_artifacts，对照 contest_school_registry_20260810_1018.zip 的 10 份核心产物输出结构覆盖率和结果可比性结论。workflow-state.json 由 ClawPro 编排器持续维护并在本节点结束后封板；只有实际路径的必需检查通过时才能裁决为可交付。',
        approvalRequired: true,
        onPass: null,
      },
    ],
  };
}

function syncMultiAgentTemplateAssets(
  template: TenantPipelineTemplate,
  agents: TenantAgent[],
): TenantPipelineTemplate {
  const canonical = buildMultiAgentDevelopmentPipelineTemplate(agents);
  return {
    ...template,
    description: canonical.description,
    nodes: canonical.nodes.map(canonicalNode => ({
      ...template.nodes.find(node => node.id === canonicalNode.id),
      ...canonicalNode,
      configAssets: cloneMultiAgentNodeAssets(canonicalNode.id),
    })),
  };
}

function syncMultiAgentTaskAssets(
  task: TenantTask,
  agents: TenantAgent[],
  members: TenantProjectMember[],
): TenantTask {
  const canonical = buildMultiAgentDevelopmentTask(
    agents,
    members,
    task.updatedAt || new Date().toISOString(),
  );
  return {
    ...task,
    taskInputs: {
      requirement:
        task.taskInputs?.requirement ?? canonical.taskInputs!.requirement,
      repository_url:
        task.taskInputs?.repository_url ?? canonical.taskInputs!.repository_url,
      reference_artifacts:
        task.taskInputs?.reference_artifacts ??
        canonical.taskInputs!.reference_artifacts,
    },
    workflow: canonical.workflow.map(canonicalNode => {
      const node = task.workflow.find(
        item => item.runtimePhaseId === canonicalNode.runtimePhaseId,
      );
      return {
        ...canonicalNode,
        ...node,
        // 新增或重排节点后必须使用当前模板的节点 ID。
        // 复用旧 ID 会产生重复 ID，导致下游依赖全部失效并挤到同一列。
        id: canonicalNode.id,
        dependsOn: canonicalNode.dependsOn,
        runtimePrompt: canonicalNode.runtimePrompt,
        runtimeInputs: canonicalNode.runtimeInputs,
        runtimeOutputs: canonicalNode.runtimeOutputs,
        runtimeApprovalRequired: canonicalNode.runtimeApprovalRequired,
        runtimeOnPass: canonicalNode.runtimeOnPass,
        runtimeOnFail: canonicalNode.runtimeOnFail,
        runtimeDecisionMode: canonicalNode.runtimeDecisionMode,
        runtimeMaxRetries: canonicalNode.runtimeMaxRetries,
        configAssets: cloneMultiAgentNodeAssets(canonicalNode.runtimePhaseId ?? ''),
      };
    }),
  };
}

function buildProjectManagementSopPipelineTemplate(
  agents: TenantAgent[],
): TenantPipelineTemplate {
  const codeBuddyAgent = agents.find(agent => agent.platform === 'codebuddy');
  const progressAgents = PROJECT_MANAGEMENT_AGENT_IDS.map(id =>
    agents.find(agent => agent.id === id),
  );
  const projectGroups = [
    MUST_WIN_PROJECTS.slice(0, 3),
    MUST_WIN_PROJECTS.slice(3, 6),
    MUST_WIN_PROJECTS.slice(6, 9),
  ];
  const progressNodes: TenantPipelineTemplateNode[] = projectGroups.map(
    (projects, index) => ({
      id: `must-win-${index + 1}`,
      title: `进展 Agent ${String.fromCharCode(65 + index)} · ${projects.length} 个项目`,
      dependsOn: [],
      agentRole: '项目进展分析',
      executorKind: 'projectAgent',
      executorRef: progressAgents[index]?.id ?? codeBuddyAgent?.id,
      inputs: [
        {
          key: 'reporting_period',
          label: '统计周期',
          type: 'text',
          required: true,
        },
        {
          key: 'data_mode',
          label: '数据口径',
          type: 'text',
          required: true,
        },
      ],
      outputs: [
        {
          key: 'progress_report',
          label: `progress-agent-${String.fromCharCode(97 + index)}.md`,
          type: 'markdown',
          required: true,
        },
        {
          key: 'progress_metrics',
          label: `progress-agent-${String.fromCharCode(97 + index)}.json`,
          type: 'json',
          required: true,
        },
      ],
      promptTemplate:
        `你是进展 Agent ${String.fromCharCode(65 + index)}。只处理这 3 个项目：${projects.join('、')}。` +
        '为每个项目输出一行 AI Mock 进展：进度、灯号和一个阻塞；同时生成 markdown 与 JSON 两份极简产物。不得处理其他分组，也不得声称读取了真实平台数据。',
    }),
  );
  const summaryNode: TenantPipelineTemplateNode = {
    id: 'must-win-summary',
    title: '汇总各 Agent 回传结果',
    dependsOn: progressNodes.map(node => node.id),
    agentRole: '项目经营汇总',
    executorKind:
      codeBuddyAgent?.kind === 'project' ? 'projectAgent' : 'memberAgent',
    executorRef: codeBuddyAgent?.id,
    inputs: progressNodes.map(node => ({
      key: `${node.id}-progress`,
      label: node.title,
      type: 'markdown' as const,
      required: true,
      source: { nodeId: node.id, outputKey: 'progress_report' },
    })),
    outputs: [
      {
        key: 'portfolio_summary',
        label: '9 个必赢之战进展总览',
        type: 'markdown',
        required: true,
      },
      {
        key: 'portfolio_metrics',
        label: '汇总指标',
        type: 'json',
        required: true,
      },
    ],
    promptTemplate:
      '等待三个进展 Agent 全部回传后再汇总。读取三份上游进展卡，输出 9 个项目的统一表格、整体完成度、红黄灯分布和 Top 3 风险；报告顶部必须保留“AI Mock 数据”标识。',
    approvalRequired: true,
  };
  return {
    id: TENANT_PROJECT_MANAGEMENT_SOP_TEMPLATE_ID,
    name: '项目管理SOP',
    description:
      '先把 9 个必赢之战拆成 3 组并行下发给三个独立 Agent 会话；等待三组结果全部回传后，再由汇总 Agent 生成组合进展。当前演示使用明确标记的 AI Mock 数据。',
    nodes: [...progressNodes, summaryNode],
  };
}

function buildProjectManagementSopTask(
  agents: TenantAgent[],
  members: TenantProjectMember[],
  now: string,
): TenantTask {
  const template = buildProjectManagementSopPipelineTemplate(agents);
  const workflow = instantiatePipelineWorkflow(
    TENANT_PROJECT_MANAGEMENT_SOP_TASK_ID,
    template,
    agents,
    members,
    0,
    false,
  ).map(node => ({
    ...node,
    handoff:
      node.runtimePhaseId === 'must-win-summary'
        ? '汇聚 9 个并行项目进展卡和指标，产出组合进展总览。'
        : '交接当前必赢之战的 AI Mock 进展卡与结构化指标。',
  }));
  return {
    id: TENANT_PROJECT_MANAGEMENT_SOP_TASK_ID,
    title: '项目管理SOP · 9 个必赢之战周进展统计',
    description:
      '先把 9 个必赢之战分成 3 组下发给三个项目进展 Agent；三个 Agent 分别回传进展卡后，汇总 Agent 再生成整体完成度、红黄灯和风险清单。本演示数据全部为 AI Mock。',
    status: 'todo',
    ownerId: members[0]?.userId ?? CURRENT_USER,
    priority: 'high',
    dueDate: '2026-12-31',
    workflowTemplateId: 'auto',
    pipelineTemplateId: TENANT_PROJECT_MANAGEMENT_SOP_TEMPLATE_ID,
    workflow,
    taskInputs: {
      reporting_period: {
        type: 'text',
        value: '2026-08-17 至 2026-08-21',
      },
      data_mode: { type: 'text', value: 'AI Mock（演示数据）' },
      projects: {
        type: 'text',
        value: MUST_WIN_PROJECTS.join('、'),
      },
    },
    runtimeExecutable: true,
    source: 'manual',
    triggerType: 'manual',
    tags: ['项目管理 SOP', '9 个必赢之战', 'AI Mock'],
    createdAt: now,
    updatedAt: now,
  };
}

function buildProductFrontendDemoSopPipelineTemplate(
  agents: TenantAgent[],
): TenantPipelineTemplate {
  const codeBuddyAgent = agents.find(agent => agent.platform === 'codebuddy');
  const executorKind =
    codeBuddyAgent?.kind === 'project' ? 'projectAgent' : 'memberAgent';
  const executorRef = codeBuddyAgent?.id;
  const workflowCard = (
    stage: string,
    rules: string,
    skills: string,
  ) =>
    `开始执行前必须先输出：\n【ClawPro Workflow 命中】\n阶段：${stage}\nRules：${rules}\nSkills：${skills}\n来源：CODEBUDDY.md + .clawpro/workflow.yaml\n`;

  const nodes: TenantPipelineTemplateNode[] = [
    {
      id: 'frontend-demo-develop',
      title: '拉分支并开发前端 Demo',
      dependsOn: [],
      agentRole: '产品前端 Demo 开发',
      executorKind,
      executorRef,
      inputs: [
        {
          key: 'requirement',
          label: '产品需求',
          type: 'markdown',
          required: true,
        },
        {
          key: 'repository_url',
          label: '源码仓库',
          type: 'url',
          required: true,
        },
        {
          key: 'target_page',
          label: '目标页面',
          type: 'text',
          required: true,
        },
      ],
      outputs: [
        {
          key: 'demo_summary',
          label: 'Demo 开发结论',
          type: 'markdown',
          required: true,
        },
        {
          key: 'demo_artifact',
          label: 'frontend-demo.md',
          type: 'file',
          required: true,
        },
        {
          key: 'preview_url',
          label: '目标页面预览链接',
          type: 'url',
          required: true,
        },
      ],
      promptTemplate:
        workflowCard(
          '拉分支并开发前端 demo',
          'clawpro-portable-design-skill、impeccable',
          'clawpro-portable-design-skill、impeccable',
        ) +
        '按产品需求定位源码仓库和目标分支，遵循项目现有设计规范完成前端 Demo；执行必要检查并启动预览。输出改动范围、核心交互、验证结果和目标页面直达链接。不得提交、推送或创建 MR。',
      approvalRequired: true,
    },
    {
      id: 'frontend-demo-acceptance',
      title: '确认最终前端 Demo',
      dependsOn: ['frontend-demo-develop'],
      agentRole: '产品与交互验收',
      executorKind,
      executorRef,
      inputs: [
        {
          key: 'demo_summary',
          label: 'Demo 开发结论',
          type: 'markdown',
          required: true,
          source: { nodeId: 'frontend-demo-develop', outputKey: 'demo_summary' },
        },
        {
          key: 'demo_artifact',
          label: '前端 Demo 产物',
          type: 'file',
          required: true,
          source: { nodeId: 'frontend-demo-develop', outputKey: 'demo_artifact' },
        },
        {
          key: 'preview_url',
          label: '预览链接',
          type: 'url',
          required: true,
          source: { nodeId: 'frontend-demo-develop', outputKey: 'preview_url' },
        },
      ],
      outputs: [
        {
          key: 'acceptance_summary',
          label: '最终 Demo 验收结论',
          type: 'markdown',
          required: true,
        },
        {
          key: 'acceptance_artifact',
          label: 'demo-acceptance.md',
          type: 'file',
          required: true,
        },
        {
          key: 'final_preview_url',
          label: '最终 Demo 链接',
          type: 'url',
          required: true,
        },
      ],
      promptTemplate:
        workflowCard(
          '合并后确认最终前端 demo',
          'impeccable',
          'impeccable',
        ) +
        '基于上游 Demo 和预览链接执行产品、视觉、交互与状态验收，检查主路径、空态、加载态和错误态。记录问题及修正建议，形成最终 Demo 验收结论；未通过时明确退回原因。',
      approvalRequired: true,
    },
    {
      id: 'frontend-demo-requirement',
      title: '基于最终 Demo 写需求单',
      dependsOn: ['frontend-demo-acceptance'],
      agentRole: '产品需求整理',
      executorKind,
      executorRef,
      inputs: [
        {
          key: 'acceptance_summary',
          label: '最终 Demo 验收结论',
          type: 'markdown',
          required: true,
          source: {
            nodeId: 'frontend-demo-acceptance',
            outputKey: 'acceptance_summary',
          },
        },
        {
          key: 'acceptance_artifact',
          label: '最终 Demo 验收文档',
          type: 'file',
          required: true,
          source: {
            nodeId: 'frontend-demo-acceptance',
            outputKey: 'acceptance_artifact',
          },
        },
        {
          key: 'final_preview_url',
          label: '最终 Demo 链接',
          type: 'url',
          required: true,
          source: {
            nodeId: 'frontend-demo-acceptance',
            outputKey: 'final_preview_url',
          },
        },
      ],
      outputs: [
        {
          key: 'requirement_summary',
          label: '需求摘要',
          type: 'markdown',
          required: true,
        },
        {
          key: 'requirement_artifact',
          label: 'requirement.md',
          type: 'file',
          required: true,
        },
      ],
      promptTemplate:
        workflowCard(
          '基于最终 demo 写需求单',
          'requirement-writer',
          'requirement-writer',
        ) +
        '以最终 Demo 和验收结论为唯一产品口径，输出可交付研发的需求单，包含背景、目标、范围、页面与交互、处理规则、异常场景和可逐条验证的验收标准。不得加入 Demo 中不存在且用户未确认的能力。',
      approvalRequired: true,
    },
    {
      id: 'frontend-demo-test-suite',
      title: '基于需求单和 Demo 写测试用例',
      dependsOn: ['frontend-demo-requirement'],
      agentRole: '测试用例设计',
      executorKind,
      executorRef,
      inputs: [
        {
          key: 'requirement_summary',
          label: '需求摘要',
          type: 'markdown',
          required: true,
          source: {
            nodeId: 'frontend-demo-requirement',
            outputKey: 'requirement_summary',
          },
        },
        {
          key: 'requirement_artifact',
          label: '需求单',
          type: 'file',
          required: true,
          source: {
            nodeId: 'frontend-demo-requirement',
            outputKey: 'requirement_artifact',
          },
        },
        {
          key: 'final_preview_url',
          label: '最终 Demo 链接',
          type: 'url',
          required: true,
          source: {
            nodeId: 'frontend-demo-acceptance',
            outputKey: 'final_preview_url',
          },
        },
      ],
      outputs: [
        {
          key: 'test_summary',
          label: '测试范围摘要',
          type: 'markdown',
          required: true,
        },
        {
          key: 'test_artifact',
          label: 'test-cases.md',
          type: 'file',
          required: true,
        },
      ],
      promptTemplate:
        workflowCard(
          '基于需求单和最终 demo 写测试用例',
          'clawpro-test-suite、requirement-writer',
          'clawpro-test-suite',
        ) +
        '逐条对齐需求单验收标准和最终 Demo，输出测试用例编号、前置条件、步骤、预期结果和优先级；覆盖主路径、边界、权限、空态、加载态、错误态与回归范围，并建立需求到用例的可追踪映射。',
      approvalRequired: true,
    },
  ];

  return {
    id: TENANT_PRODUCT_FRONTEND_DEMO_SOP_TEMPLATE_ID,
    name: '产品前端 Demo 开发 SOP',
    description:
      '来源 CODEBUDDY.md：从拉分支开发 Demo、最终 Demo 验收，到需求单和测试用例输出的 4 阶段产品研发闭环。每个阶段按指定 Rules/Skills 执行并经人工确认后流转。',
    nodes,
  };
}

function buildProductFrontendDemoSopTask(
  agents: TenantAgent[],
  members: TenantProjectMember[],
  now: string,
): TenantTask {
  const template = buildProductFrontendDemoSopPipelineTemplate(agents);
  const codeBuddyAgent = agents.find(agent => agent.platform === 'codebuddy');
  const workflow = instantiatePipelineWorkflow(
    TENANT_PRODUCT_FRONTEND_DEMO_SOP_TASK_ID,
    template,
    agents,
    members,
    0,
    false,
  ).map(node => ({
    ...node,
    agentId: codeBuddyAgent?.id ?? null,
    handoff:
      '交接本阶段结论、文件产物和预览链接；ClawPro 人工确认后才进入下一阶段。',
  }));
  return {
    id: TENANT_PRODUCT_FRONTEND_DEMO_SOP_TASK_ID,
    title: '产品前端 Demo 开发 SOP · 完整执行演示',
    description:
      '从真实源码仓库开发前端 Demo，经产品与交互验收后，继续生成需求单和测试用例；四个阶段逐步确认并自动交接产物。',
    status: 'todo',
    ownerId: members[0]?.userId ?? CURRENT_USER,
    priority: 'high',
    dueDate: '2026-12-31',
    workflowTemplateId: 'auto',
    pipelineTemplateId: TENANT_PRODUCT_FRONTEND_DEMO_SOP_TEMPLATE_ID,
    workflow,
    taskInputs: {
      requirement: {
        type: 'markdown',
        value:
          '优化项目协作工作流执行页：节点完成后可人工确认，可查看输入、输出和产物，支持停止执行，并可为节点选择真实 Agent。',
      },
      repository_url: {
        type: 'url',
        value: 'https://git.woa.com/cvm-openclaw/openclaw-enterprise',
      },
      target_page: {
        type: 'text',
        value: '/project-collaboration',
      },
    },
    runtimeExecutable: true,
    source: 'manual',
    triggerType: 'manual',
    tags: ['完整 SOP', '产品前端 Demo'],
    createdAt: now,
    updatedAt: now,
  };
}

function buildPreVisitBriefTask(
  agents: TenantAgent[],
  members: TenantProjectMember[],
  now: string,
): TenantTask {
  const template = buildPreVisitBriefPipelineTemplate(agents);
  const cloudAgentByPhase: Record<string, string> = {
    'calendar-match': 'calendar-scan-agent',
    'brief-data-collect': 'pre-visit-reminder-agent',
    'brief-message-assemble': 'message-assemble-agent',
    'brief-message-notify': 'message-notify-agent',
  };
  const workflow = instantiatePipelineWorkflow(
    TENANT_PRE_VISIT_BRIEF_TASK_ID,
    template,
    agents,
    members,
    0,
    false,
  ).map(node => ({
    ...node,
    agentId: cloudAgentByPhase[node.runtimePhaseId ?? ''] ?? null,
    handoff:
      node.runtimePhaseId === 'brief-message-assemble'
        ? '交接真实消息 ID、确认链接、消息草稿和完整简报；用户确认后才能触发推送。'
        : '按 Handoff v2 交接结构化节点输出、可读结论和文件产物。',
  }));

  return {
    id: TENANT_PRE_VISIT_BRIEF_TASK_ID,
    title: '拜访前简报 SOP · 44813 号日程',
    description:
      '以“生成 44813 号日程的会前简报”为输入，真实校验日程与商机、采集拜访数据、生成并暂存简报，经人工确认后触发企微推送。',
    status: 'todo',
    ownerId: members[0]?.userId ?? CURRENT_USER,
    priority: 'high',
    dueDate: '2026-12-31',
    workflowTemplateId: 'auto',
    pipelineTemplateId: TENANT_PRE_VISIT_BRIEF_TEMPLATE_ID,
    workflow,
    taskInputs: {
      input: {
        type: 'text',
        value: '生成 44813号日程的会前简报',
      },
      event_id: {
        type: 'text',
        value: '44813',
      },
    },
    runtimeExecutable: true,
    source: 'manual',
    triggerType: 'manual',
    tags: ['完整 SOP', '拜访前简报', '企微推送'],
    createdAt: now,
    updatedAt: now,
  };
}

function buildKnowledgeInspectionTask(
  agents: TenantAgent[],
  members: TenantProjectMember[],
  now: string,
): TenantTask {
  const template = buildKnowledgeInspectionPipelineTemplate();
  const codeBuddyAgent = agents.find(agent => agent.platform === 'codebuddy');
  const imateAgent = agents.find(agent => agent.platform === 'imate');
  const workflow = instantiatePipelineWorkflow(
    TENANT_KB_INSPECTION_TASK_ID,
    template,
    agents,
    members,
    0,
    false,
  ).map((node, index) => ({
    ...node,
    agentId:
      index >= 2
        ? (imateAgent?.id ?? codeBuddyAgent?.id ?? null)
        : (codeBuddyAgent?.id ?? imateAgent?.id ?? null),
    handoff: '按 Handoff v2 交接结构化扫描结果、问题清单与产物引用。',
  }));
  return {
    id: TENANT_KB_INSPECTION_TASK_ID,
    title: '架构师知识库巡检SOP · 完整执行演示',
    description:
      '递归扫描 iWiki 根目录 4025707654 下的 7 个知识域，执行平台事实交叉校验、问题识别与分级，最终输出只读审计报告。',
    status: 'todo',
    ownerId: members[0]?.userId ?? CURRENT_USER,
    priority: 'high',
    dueDate: '2026-12-31',
    workflowTemplateId: 'auto',
    pipelineTemplateId: TENANT_KB_INSPECTION_TEMPLATE_ID,
    workflow,
    taskInputs: {
      root_docid: { type: 'text', value: '4025707654' },
      domains: {
        type: 'text',
        value: '产品知识,运营流程,客户管理,案例库,外部事实源映射,话术库,CHC归档',
      },
      cross_check: { type: 'text', value: 'true' },
    },
    runtimeExecutable: true,
    source: 'manual',
    triggerType: 'manual',
    tags: ['完整 SOP', '知识库巡检'],
    createdAt: now,
    updatedAt: now,
  };
}

function buildPublicImageReleaseTask(
  agents: TenantAgent[],
  members: TenantProjectMember[],
  now: string,
): TenantTask {
  const template = buildPublicImageReleasePipelineTemplate();
  const codeBuddyAgent = agents.find(agent => agent.platform === 'codebuddy');
  const imateAgent = agents.find(agent => agent.platform === 'imate');
  const imatePhases = new Set(['WAIT_CLS_CHECK', 'WAIT_MEMORY_CHECK', 'WAIT_RELEASE']);
  const workflow = instantiatePipelineWorkflow(
    TENANT_IMAGE_RELEASE_TASK_ID,
    template,
    agents,
    members,
    0,
    false,
  ).map(node => ({
    ...node,
    agentId: imatePhases.has(node.runtimePhaseId ?? '')
      ? (imateAgent?.id ?? codeBuddyAgent?.id ?? null)
      : (codeBuddyAgent?.id ?? imateAgent?.id ?? null),
    handoff:
      node.runtimePhaseId === 'WAIT_RELEASE_CONFIRM'
        ? '汇聚四个并行 QA 节点的全部结论和产物；缺一项不得放行。'
        : '按 Handoff v2 交接当前阶段结论、状态、证据与产物引用。',
  }));
  return {
    id: TENANT_IMAGE_RELEASE_TASK_ID,
    title: 'ClawPro 公共镜像发布跟进 · 并发 QA 演示',
    description:
      '输入镜像类型、版本和 TAPD 需求后，按需求初始化、镜像制作、四方并行 QA、发布确认、发布、版本记录和完成归档推进；4 个 QA 节点全部完成后才允许汇聚放行。',
    status: 'todo',
    ownerId: members[0]?.userId ?? CURRENT_USER,
    priority: 'high',
    dueDate: '2026-12-31',
    workflowTemplateId: 'auto',
    pipelineTemplateId: TENANT_IMAGE_RELEASE_TEMPLATE_ID,
    workflow,
    taskInputs: {
      image: { type: 'text', value: 'openclaw' },
      version: { type: 'text', value: 'v1.2' },
      tapd_url: { type: 'url', value: 'https://tapd.woa.com/' },
    },
    runtimeExecutable: true,
    source: 'manual',
    triggerType: 'manual',
    tags: ['完整 SOP', '公共镜像', '并行 QA'],
    createdAt: now,
    updatedAt: now,
  };
}

function buildStandardDevelopmentSopTask(
  agents: TenantAgent[],
  members: TenantProjectMember[],
  now: string,
): TenantTask {
  const template = buildStandardDevelopmentSopPipelineTemplate(agents);
  const codeBuddyAgent = agents.find(agent => agent.platform === 'codebuddy');
  const workflow = instantiatePipelineWorkflow(
    TENANT_STANDARD_DEV_SOP_TASK_ID,
    template,
    agents,
    members,
    0,
    false,
  ).map(node => ({
    ...node,
    agentId: codeBuddyAgent?.id ?? null,
    handoff: '交接当前步骤结论、阶段文档和 00-overview.md，确认后进入下一步。',
  }));
  return {
    id: TENANT_STANDARD_DEV_SOP_TASK_ID,
    title: 'vstation开发标准SOP · 9 步完整执行',
    description:
      '基于 vstation/api 真实仓库执行 Clarify、Plan、Implement、UT、Deploy、IT、Docs、Review、Commit 九步交付；每一步完成后都必须由用户确认才能继续。',
    status: 'todo',
    ownerId: members[0]?.userId ?? CURRENT_USER,
    priority: 'high',
    dueDate: '2026-12-31',
    workflowTemplateId: 'auto',
    pipelineTemplateId: TENANT_STANDARD_DEV_SOP_TEMPLATE_ID,
    workflow,
    taskInputs: {
      repository_url: {
        type: 'url',
        value: 'https://git.woa.com/cvm-openclaw/openclaw-enterprise',
      },
      requirement: {
        type: 'markdown',
        value:
          '在 client/src/pages/tenant/tenantProjectStore.ts 中，将模板名称“标准开发 SOP · 9 步交付”改为“vStation 开发标准 SOP”，并将任务标题“标准开发 SOP · 9 步真实执行演示”改为“vStation 开发标准 SOP · 9 步完整执行”。这是低风险文案改动；只做本地修改和验证，不提交、不推送。',
      },
      current_branch: {
        type: 'text',
        value: 'feature/tenant-project-collaboration',
      },
    },
    runtimeExecutable: true,
    source: 'manual',
    triggerType: 'manual',
    tags: ['完整 SOP', 'vstation', '开发 SOP'],
    createdAt: now,
    updatedAt: now,
  };
}

function buildMultiAgentDevelopmentTask(
  agents: TenantAgent[],
  members: TenantProjectMember[],
  now: string,
): TenantTask {
  const template = buildMultiAgentDevelopmentPipelineTemplate(agents);
  const workflow = instantiatePipelineWorkflow(
    TENANT_MULTI_AGENT_DEV_TASK_ID,
    template,
    agents,
    members,
    0,
    false,
  ).map(node => ({
    ...node,
    handoff:
      '按文件级输入输出契约交接本节点完整产物、核心结论、执行身份、大小与 SHA-256；下游不得只依赖聊天摘要。',
  }));
  return {
    id: TENANT_MULTI_AGENT_DEV_TASK_ID,
    title: '多 Agent 研发交付 SOP · 校园大赛对照测试',
    description:
      '使用 contest_school_registry_20260810_1018 的原始需求和真实产物作为基准，由 CodeBuddy 与 iMate OpenClaw 完成同构研发交付并在最终节点输出逐项对照。',
    status: 'todo',
    ownerId: members[0]?.userId ?? CURRENT_USER,
    priority: 'high',
    dueDate: '2026-12-31',
    workflowTemplateId: 'auto',
    pipelineTemplateId: TENANT_MULTI_AGENT_DEV_TEMPLATE_ID,
    workflow,
    taskInputs: {
      repository_url: {
        type: 'url',
        value: 'https://git.woa.com/ai-skillhub-codesign/skillhub',
      },
      requirement: {
        type: 'markdown',
        value:
          '第三期大赛的主题是校园大赛，参赛时需要允许用户填写学校。用一个新接口来记录，用一个新表来记录，同时关联参赛的 skillId。另外还有一个接口来记录学校的热度，热度用数量表示即可。\n\n追加约束：平台已开放同名 slug，参赛表需要改为以 skillId 为维度。',
      },
      reference_artifacts: {
        type: 'json',
        value: MULTI_AGENT_DEV_REFERENCE_MANIFEST,
      },
    },
    runtimeExecutable: true,
    source: 'manual',
    triggerType: 'manual',
    tags: ['完整 SOP', '多 Agent', '文件级交接'],
    createdAt: now,
    updatedAt: now,
  };
}

/** 计算下次运行时间（展示用；mock，不做真实调度） */
function computeNextRunAt(schedule: TenantAutomationSchedule): string | undefined {
  const now = new Date();
  if (schedule.type === 'once') return schedule.onceAt;
  if (schedule.type === 'interval') {
    const h = schedule.intervalHours ?? 1;
    return new Date(now.getTime() + h * 3600_000).toISOString();
  }
  // periodic：取今天/明天的 HH:mm
  const [hh, mm] = (schedule.time ?? '09:00').split(':').map(Number);
  const next = new Date(now);
  next.setHours(hh || 9, mm || 0, 0, 0);
  if (schedule.cycle === 'weekly' && typeof schedule.weekday === 'number') {
    const diff = (schedule.weekday - next.getDay() + 7) % 7;
    next.setDate(next.getDate() + (diff === 0 && next <= now ? 7 : diff));
  } else if (next <= now) {
    next.setDate(next.getDate() + 1);
  }
  return next.toISOString();
}

/** seed：给项目造一条示例自动化（每日汇总项目动态并建任务） */
function buildAutomations(
  pipelineTemplates: TenantPipelineTemplate[],
): TenantAutomation[] {
  const weekly = pipelineTemplates.find((t) => t.id === 'pl-weekly-report');
  if (!weekly) return [];
  const schedule: TenantAutomationSchedule = {
    type: 'periodic',
    cycle: 'daily',
    time: '10:00',
  };
  return [
    {
      id: 'auto-daily-digest',
      name: '每日项目动态汇总',
      prompt: '汇总过去 24 小时项目内的任务进展、风险与待跟进事项，整理成日报。',
      triggerKind: 'schedule',
      schedule,
      outputMode: 'createTask',
      pipelineTemplateId: weekly.id,
      enabled: true,
      createdAt: new Date().toISOString(),
  nextRunAt: computeNextRunAt(schedule),
    },
  ];
}

function buildSkillUsage(assets: TenantProjectAssets): TenantSkillUsage[] {
  const refs = assets.enterpriseSkill ?? [];  return refs.map((refId, i) => {
    const d = getAssetItemDisplay('enterpriseSkill', refId);
    return { name: d.name, calls: CALL_SEED[i % CALL_SEED.length], lastUsed: TIME_SEED[i % TIME_SEED.length] };
  });
}

function buildActivities(learnings: TenantLearning[], tasks: TenantTask[]): TenantActivity[] {
  const acts: TenantActivity[] = [];
  learnings.slice(0, 2).forEach((l, i) => {
    acts.push({ id: `act-r-${i}`, kind: 'learning_report', text: `${l.sourceAgent}上报了经验《${l.title}》`, time: l.time });
  });
  const inProgress = tasks.find((t) => t.status === 'in_progress');
  if (inProgress) {
    acts.push({ id: 'act-task', kind: 'task_dispatch', text: `任务《${inProgress.title}》已进入协作执行`, time: '1 天前' });
  }
  if (learnings[0]) {
    acts.push({ id: 'act-recall', kind: 'learning_recall', text: `有 agent 在任务前检索并学习了《${learnings[0].title}》`, time: '3 小时前' });
  }
  acts.push({ id: 'act-asset', kind: 'asset_add', text: '项目新增了一项企业技能资产', time: '2 天前' });
  return acts;
}

/** ── 汇报周期 key（ISO 周） ── */
export function isoWeekKey(d = new Date()): { key: string; label: string } {
  const date = new Date(Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()));
  const dayNum = date.getUTCDay() || 7;
  date.setUTCDate(date.getUTCDate() + 4 - dayNum);
  const yearStart = new Date(Date.UTC(date.getUTCFullYear(), 0, 1));
  const week = Math.ceil(((date.getTime() - yearStart.getTime()) / 86400000 + 1) / 7);
  const key = `${date.getUTCFullYear()}-W${String(week).padStart(2, '0')}`;
  return { key, label: `${date.getUTCFullYear()} 年第 ${week} 周` };
}

/** ── 进度底座计算（场景1：进展快照的客观进度层，机械 roll-up） ──
 * 从某成员名下任务的 workflow 节点确认比例算完成度，并提取卡点，供汇报快照直接引用。
 */
export function computeProgressBase(
  tasks: TenantTask[],
  reporterId: string,
): TenantProgressBase {
  const mine = tasks.filter(
    (t) => t.ownerId === reporterId && t.taskType !== 'report',
  );
  const taskTotal = mine.length;
  const taskDone = mine.filter((t) => t.status === 'done').length;
  const taskActive = mine.filter(
    (t) => t.status === 'in_progress' || t.status === 'review',
  ).length;
  let nodeTotal = 0;
  let nodeConfirmed = 0;
  const blockers: string[] = [];
  mine.forEach((t) => {
    t.workflow.forEach((n) => {
      nodeTotal += 1;
      if (n.status === 'confirmed') nodeConfirmed += 1;
      if (n.status === 'stopped') {
        blockers.push(`《${t.title}》的「${n.title}」节点已停止，待重试`);
      }
      if (n.status === 'review') {
        blockers.push(`《${t.title}》的「${n.title}」结果待确认`);
      }
    });
  });
  const completion =
    nodeTotal === 0 ? (taskTotal === 0 ? 0 : Math.round((taskDone / taskTotal) * 100)) : Math.round((nodeConfirmed / nodeTotal) * 100);
  return {
    taskTotal,
    taskDone,
    taskActive,
    completion,
    blockers: blockers.slice(0, 4),
    taskIds: mine.map((t) => t.id),
  };
}

interface SeedInput {
  id: string;
  name: string;
  description: string;
  goal: string;
  colorKey: string;
  members: TenantProjectMember[];
  allowMemberEdit: boolean;
  assets: TenantProjectAssets;
  assetVersion: number;
  weekActiveSessions: number;
  learningIdx: number[];
  taskCount: number;
  /** 经验库被引用次数（Agent 社区声誉 mock 值） */
  referencedCount: number;
  /** 汇报规格（可选，seed 里给首个项目开启演示） */
  reportingSpec?: TenantReportingSpec;
}

function enrich(input: SeedInput, now: string): TenantProject {
  const baseAgents = buildAgents(input.members);
  const agents = input.id === 'tp-clawpro'
    ? ensureProjectManagementRuntimeAgents(
        ensureKnowledgeInspectionAgents(
          ensureIssueFixRuntimeAgents(baseAgents, input.members),
          input.members,
        ),
        input.members,
      )
    : baseAgents;
  const learnings = buildLearnings(input.learningIdx, input.members);
  const basePipelineTemplates = defaultPipelineTemplates();
  const pipelineTemplates = input.id === 'tp-clawpro'
    ? [
        buildStandardDevelopmentSopPipelineTemplate(agents),
        buildMultiAgentDevelopmentPipelineTemplate(agents),
        buildProjectManagementSopPipelineTemplate(agents),
        buildProductFrontendDemoSopPipelineTemplate(agents),
        buildPreVisitBriefPipelineTemplate(agents),
        buildKnowledgeInspectionPipelineTemplate(),
        buildIssueFixPipelineTemplate(agents),
        buildPublicImageReleasePipelineTemplate(),
      ]
    : basePipelineTemplates;
  const collabTasks = buildTasks(
    input.learningIdx[0] ?? 0,
    input.taskCount,
    agents,
    input.members,
    basePipelineTemplates,
    now,
  );
  const pipelineIssues = buildPipelineIssues(
    basePipelineTemplates[0],
    agents,
    input.members,
    now,
  );
  const tasks = input.id === 'tp-clawpro'
    ? [
        buildStandardDevelopmentSopTask(agents, input.members, now),
        buildMultiAgentDevelopmentTask(agents, input.members, now),
        buildProjectManagementSopTask(agents, input.members, now),
        buildProductFrontendDemoSopTask(agents, input.members, now),
        buildPreVisitBriefTask(agents, input.members, now),
        buildKnowledgeInspectionTask(agents, input.members, now),
        buildIssueFixDemoTask(agents, input.members, now),
        buildPublicImageReleaseTask(agents, input.members, now),
      ]
    : [...collabTasks, ...pipelineIssues];
  const skillUsage = buildSkillUsage(input.assets);
  const activities = buildActivities(learnings, tasks);
  const weekRecalled = learnings.reduce((s, l) => s + l.recalledCount, 0);
  const automations = input.id === 'tp-clawpro' ? [] : buildAutomations(basePipelineTemplates);
  return {
    id: input.id,
    name: input.name,
    description: input.description,
    goal: input.goal,
    colorKey: input.colorKey,
    members: input.members,
    agents,
    allowMemberEdit: input.allowMemberEdit,
    reportingSpec: input.reportingSpec ?? defaultReportingSpec(),
    progressSnapshots: [],
    assets: input.assets,
    assetVersion: input.assetVersion,
    // 经验库 Skill 版本：初始按已沉淀经验条数体现（每并入一条经验即一次更新）
    experienceSkillVersion: learnings.length + 2,
    experienceFlywheelEnabled: true,
    experienceRecallEnabled: true,
    experienceDepositionEnabled: true,
    metrics: {
      agentCount: agents.length,
      learningCount: learnings.length,
      weekRecalled,
      weekActiveSessions: input.weekActiveSessions,
      inProgressTasks: tasks.filter(
        (t) => t.status === 'in_progress' || t.status === 'review',
      ).length,
    },
    learnings,
    tasks,
    pipelineTemplates,
    automations,
    skillUsage,
    activities,
    referencedCount: input.referencedCount,
    referencedLibraries: [],
    updatedAt: now,
  };
}

const EXPANDED_PROJECT_CATALOG = [
  {
    id: 'tp-heterogeneous-operations',
    name: '异构经营系统',
    description: '面向异构资源经营分析与协作交付的项目空间。',
    goal: '统一沉淀异构资源经营分析能力与项目经验',
    colorKey: '异',
  },
  {
    id: 'tp-thpc',
    name: 'THPC',
    description: 'THPC 产品研发、联调与交付协作空间。',
    goal: '提升 THPC 产品研发与交付协同效率',
    colorKey: 'T',
  },
  {
    id: 'tp-light-dns-products',
    name: '轻量云域名解析相关产品',
    description: '轻量云域名解析相关产品的研发协作空间。',
    goal: '统一域名解析产品的需求、研发与经验沉淀',
    colorKey: '域',
  },
  {
    id: 'tp-light-dns-dataplane',
    name: '轻量云解析数据面项目',
    description: '轻量云解析数据面的研发、联调与质量协作空间。',
    goal: '提升解析数据面的交付质量和协作效率',
    colorKey: '解',
  },
  {
    id: 'tp-light-orcaterm',
    name: '轻量云OrcaTerm项目',
    description: '轻量云 OrcaTerm 产品研发与 Agent 协作空间。',
    goal: '沉淀 OrcaTerm 的标准研发工作流和项目资产',
    colorKey: 'O',
  },
  {
    id: 'tp-light-agentchat',
    name: '轻量云AgentChat项目',
    description: '轻量云 AgentChat 产品研发与多 Agent 协作空间。',
    goal: '加速 AgentChat 功能研发、验证和经验复用',
    colorKey: 'A',
  },
  {
    id: 'tp-light-lighthouse-ai-lab',
    name: '轻量云Lighthouse AI实验项目',
    description: 'Lighthouse AI 能力探索、实验与验证空间。',
    goal: '快速验证 Lighthouse AI 场景并沉淀可复用方案',
    colorKey: 'A',
  },
  {
    id: 'tp-light-lighthouse-agent',
    name: '轻量云Lighthouse Agent项目',
    description: 'Lighthouse Agent 能力的产品研发与交付空间。',
    goal: '构建 Lighthouse Agent 的标准能力与交付流程',
    colorKey: 'L',
  },
  {
    id: 'tp-light-lighthouse-non-agent',
    name: '轻量云Lighthouse非Agent项目',
    description: 'Lighthouse 非 Agent 功能的产品研发与协作空间。',
    goal: '统一 Lighthouse 基础功能的研发协作与资产沉淀',
    colorKey: 'L',
  },
  {
    id: 'tp-light-skillhub',
    name: '轻量云SkillHub项目',
    description: '轻量云 SkillHub 能力建设与技能资产协作空间。',
    goal: '沉淀可复用 Skill 并提升团队交付效率',
    colorKey: 'S',
  },
  {
    id: 'tp-light-lightvela',
    name: '轻量云LightVela项目',
    description: '轻量云 LightVela 产品研发与协作空间。',
    goal: '推进 LightVela 产品研发和项目知识复用',
    colorKey: 'V',
  },
] as const;

function buildExpandedProjectCatalog(now: string): TenantProject[] {
  return EXPANDED_PROJECT_CATALOG.map(item => {
    const members = [M(CURRENT_USER, 'alice', '产品', true)];
    const agents = buildAgents(members);
    const assets = emptyAssets();
    return {
      ...item,
      members,
      agents,
      allowMemberEdit: true,
      reportingSpec: defaultReportingSpec(),
      progressSnapshots: [],
      assets,
      assetVersion: 1,
      experienceSkillVersion: 1,
      experienceFlywheelEnabled: true,
      experienceRecallEnabled: true,
      experienceDepositionEnabled: true,
      metrics: {
        agentCount: agents.length,
        learningCount: 0,
        weekRecalled: 0,
        weekActiveSessions: 0,
        inProgressTasks: 0,
      },
      learnings: [],
      tasks: [],
      pipelineTemplates: defaultPipelineTemplates(),
      automations: [],
      skillUsage: buildSkillUsage(assets),
      activities: [],
      referencedCount: 0,
      referencedLibraries: [],
      updatedAt: now,
    };
  });
}

function buildSeed(): TenantProject[] {
  const now = '2026-07-27T10:00:00+08:00';
  const inputs: SeedInput[] = [
    {
      id: 'tp-clawpro', name: 'clawpro vibecoding', colorKey: 'C',
      description: 'Agent 托管与管控平台，企业内 AI 编码 Agent 的统一入口。',
      goal: '成为企业内 AI 编码 Agent 的统一入口与 harness 基座',
      members: [
        M(CURRENT_USER, 'alice', '产品', true),
        M('bob@acompany.com', 'bob', '前端', true),
        M('carol@acompany.com', 'carol', '后端'),
      ],
      allowMemberEdit: true,
      assets: buildAssets({ enterpriseSkill: [3, 0], enterpriseStandard: [2, 0], modelConfig: [2, 0] }),
      assetVersion: 5, weekActiveSessions: 12, learningIdx: [0, 1, 3], taskCount: 5, referencedCount: 6,
      reportingSpec: {
        enabled: true,
        cycle: 'weekly',
        fields: DEFAULT_REPORT_FIELDS.map((f) => ({ ...f })),
        reporterScope: [],
      },
    },
    {
      id: 'tp-image', name: 'image', colorKey: 'I',
      description: '镜像服务：Agent 运行镜像的构建、分发与版本管理。',
      goal: '让 Agent 运行镜像的构建与分发全自动、可回滚',
      members: [
        M('frank@acompany.com', 'frank', '研发'),
        M(CURRENT_USER, 'alice', '产品', true),
        M('grace@acompany.com', 'grace', '运维'),
      ],
      allowMemberEdit: true,
      assets: buildAssets({ enterpriseSkill: [2, 1], enterprisePlugin: [1, 0] }),
      assetVersion: 3, weekActiveSessions: 6, learningIdx: [5, 2], taskCount: 4, referencedCount: 2,
    },
    {
      id: 'tp-cvm', name: 'CVM控制台', colorKey: 'C',
      description: '云服务器 CVM 管理控制台前端与 Agent 辅助运维能力。',
      goal: '用 Agent 把 CVM 日常运维与排障标准化',
      members: [
        M('carol@acompany.com', 'carol', '后端'),
        M(CURRENT_USER, 'alice', '产品', true),
        M('iris@acompany.com', 'iris', '运维'),
      ],
      allowMemberEdit: false,
      assets: buildAssets({ enterpriseSkill: [2, 0], enterpriseStandard: [1, 0] }),
      assetVersion: 4, weekActiveSessions: 8, learningIdx: [1, 7], taskCount: 4, referencedCount: 4,
    },
    {
      id: 'tp-cdc', name: 'CDC', colorKey: 'C',
      description: '专属宿主机 CDC 资源纳管与巡检 Agent。',
      goal: '实现 CDC 宿主机资源的自动纳管与巡检',
      members: [M('bob@acompany.com', 'bob', '研发'), M(CURRENT_USER, 'alice', '产品', true)],
      allowMemberEdit: false,
      assets: buildAssets({ enterpriseSkill: [1, 2] }),
      assetVersion: 2, weekActiveSessions: 3, learningIdx: [7], taskCount: 3, referencedCount: 1,
    },
    {
      id: 'tp-lighthouse', name: 'Lighthouse', colorKey: 'L',
      description: '轻量应用服务器 Lighthouse 的一键部署与运维 Agent。',
      goal: '让轻量服务器一键部署与运维零人工',
      members: [
        M(CURRENT_USER, 'alice', '产品', true),
        M('grace@acompany.com', 'grace', '前端'),
        M('iris@acompany.com', 'iris', '后端'),
        M('kate@acompany.com', 'kate', '测试'),
      ],
      allowMemberEdit: true,
      assets: buildAssets({ enterpriseSkill: [2, 3], enterpriseMcp: [1, 0], modelConfig: [1, 0] }),
      assetVersion: 6, weekActiveSessions: 10, learningIdx: [2, 3, 4], taskCount: 5, referencedCount: 5,
    },
    {
      id: 'tp-hai', name: 'HAI', colorKey: 'H',
      description: '高性能应用服务 HAI（GPU 算力）的调度与用量分析 Agent。',
      goal: '让 GPU 算力调度与用量分析由 Agent 托管',
      members: [M('iris@acompany.com', 'iris', '算法'), M(CURRENT_USER, 'alice', '产品', true)],
      allowMemberEdit: false,
      assets: buildAssets({ enterpriseSkill: [1, 0], enterpriseStandard: [1, 1] }),
      assetVersion: 1, weekActiveSessions: 4, learningIdx: [4], taskCount: 3, referencedCount: 0,
    },
    {
      id: 'tp-biz', name: '商机', colorKey: 'S',
      description: '商机线索管理与智能跟进 Agent，面向售前与运营。',
      goal: '商机线索智能跟进，提升转化',
      members: [
        M('david@acompany.com', 'david', '销售'),
        M(CURRENT_USER, 'alice', '产品', true),
        M('jack@acompany.com', 'jack', '运营'),
      ],
      allowMemberEdit: true,
      assets: buildAssets({ enterpriseSkill: [2, 2], publicSkill: [2, 0] }),
      assetVersion: 3, weekActiveSessions: 5, learningIdx: [6, 4], taskCount: 4, referencedCount: 3,
    },
  ];
  return [
    ...inputs.map((i) => enrich(i, now)),
    ...buildExpandedProjectCatalog(now),
  ];
}

// ── module-scoped state ────────────────────────────────────
let projects: TenantProject[] | null = null;
const listeners = new Set<() => void>();

function ensureVersion() {
  try {
    if (localStorage.getItem(CACHE_VERSION_KEY) !== CACHE_VERSION) {
      localStorage.removeItem(CACHE_KEY);
      localStorage.setItem(CACHE_VERSION_KEY, CACHE_VERSION);
    }
  } catch {
    /* ignore */
  }
}

function load(): TenantProject[] {
  ensureVersion();
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as TenantProject[];
      if (Array.isArray(parsed) && parsed.length > 0) {
        let changed = false;
        const migratedExisting = parsed.map((project) => {
          if (project.id !== 'tp-clawpro') return project;
          const agents = ensureProjectManagementRuntimeAgents(
            ensureKnowledgeInspectionAgents(
              ensureIssueFixRuntimeAgents(project.agents, project.members),
              project.members,
            ),
            project.members,
          );
          const canonicalTaskIds = [
            TENANT_STANDARD_DEV_SOP_TASK_ID,
            TENANT_MULTI_AGENT_DEV_TASK_ID,
            TENANT_PROJECT_MANAGEMENT_SOP_TASK_ID,
            TENANT_PRODUCT_FRONTEND_DEMO_SOP_TASK_ID,
            TENANT_PRE_VISIT_BRIEF_TASK_ID,
            TENANT_KB_INSPECTION_TASK_ID,
            TENANT_WORKFLOW_DEMO_TASK_ID,
            TENANT_IMAGE_RELEASE_TASK_ID,
          ];
          const canonicalTemplateIds = [
            TENANT_STANDARD_DEV_SOP_TEMPLATE_ID,
            TENANT_MULTI_AGENT_DEV_TEMPLATE_ID,
            TENANT_PROJECT_MANAGEMENT_SOP_TEMPLATE_ID,
            TENANT_PRODUCT_FRONTEND_DEMO_SOP_TEMPLATE_ID,
            TENANT_PRE_VISIT_BRIEF_TEMPLATE_ID,
            TENANT_KB_INSPECTION_TEMPLATE_ID,
            TENANT_WORKFLOW_DEMO_TEMPLATE_ID,
            TENANT_IMAGE_RELEASE_TEMPLATE_ID,
          ];
          const hasCanonicalTasks =
            project.tasks.length === canonicalTaskIds.length &&
            project.tasks.every((task, index) => task.id === canonicalTaskIds[index]);
          const preVisitAgentByPhase: Record<string, string> = {
            'calendar-match': 'calendar-scan-agent',
            'brief-data-collect': 'pre-visit-reminder-agent',
            'brief-message-assemble': 'message-assemble-agent',
            'brief-message-notify': 'message-notify-agent',
          };
          const existingPreVisitTask = project.tasks.find(
            task => task.id === TENANT_PRE_VISIT_BRIEF_TASK_ID,
          );
          const existingMultiAgentTask = project.tasks.find(
            task => task.id === TENANT_MULTI_AGENT_DEV_TASK_ID,
          );
          const existingMultiAgentTemplate = project.pipelineTemplates.find(
            template => template.id === TENANT_MULTI_AGENT_DEV_TEMPLATE_ID,
          );
          const hasCurrentMultiAgentBenchmark = Boolean(
            existingMultiAgentTask?.runtimeExecution ||
              existingMultiAgentTask?.taskInputs?.reference_artifacts?.value ===
                MULTI_AGENT_DEV_REFERENCE_MANIFEST,
          );
          const hasCurrentMultiAgentBenchmarkTemplate = Boolean(
            existingMultiAgentTemplate?.nodes
              .find(node => node.id === 'SUMMARY')
              ?.inputs?.some(input => input.key === 'reference_artifacts'),
          );
          const hasCurrentMultiAgentAssets = Boolean(
            existingMultiAgentTask?.workflow.length &&
              existingMultiAgentTask.workflow.every(
                node =>
                  (node.configAssets?.length ?? 0) === 3 &&
                  node.configAssets!.every(
                    asset => asset.version === MULTI_AGENT_DEVELOPMENT_ASSET_VERSION,
                  ) &&
                  node.configAssets!.some(asset => asset.type === 'rules') &&
                  node.configAssets!.filter(asset => asset.type === 'contract').length === 2,
              ),
          );
          const hasCurrentMultiAgentTemplateAssets = Boolean(
            existingMultiAgentTemplate?.nodes.length &&
              existingMultiAgentTemplate.nodes.every(
                node =>
                  (node.configAssets?.length ?? 0) === 3 &&
                  node.configAssets!.every(
                    asset => asset.version === MULTI_AGENT_DEVELOPMENT_ASSET_VERSION,
                  ) &&
                  node.configAssets!.some(asset => asset.type === 'rules') &&
                  node.configAssets!.filter(asset => asset.type === 'contract').length === 2,
              ),
          );
          const hasCanonicalPreVisitAgents = Boolean(
            existingPreVisitTask &&
              existingPreVisitTask.workflow.length === 4 &&
              existingPreVisitTask.workflow.every(
                node =>
                  node.agentId ===
                  preVisitAgentByPhase[node.runtimePhaseId ?? ''],
              ),
          );
          const hasCanonicalTemplates =
            project.pipelineTemplates.length === canonicalTemplateIds.length &&
            project.pipelineTemplates.every(
              (template, index) => template.id === canonicalTemplateIds[index],
            );
          const hasCanonicalAgents = agents.length === project.agents.length;
          if (
            project.name === 'clawpro vibecoding' &&
            hasCanonicalTasks &&
            hasCanonicalPreVisitAgents &&
            hasCanonicalTemplates &&
            hasCurrentMultiAgentBenchmark &&
            hasCurrentMultiAgentBenchmarkTemplate &&
            hasCurrentMultiAgentAssets &&
            hasCurrentMultiAgentTemplateAssets &&
            hasCanonicalAgents
          ) {
            return project;
          }
          changed = true;
          const now = new Date().toISOString();
          const taskById = new Map(project.tasks.map(task => [task.id, task]));
          const templateById = new Map(
            project.pipelineTemplates.map(template => [template.id, template]),
          );
          const next = {
            ...project,
            name: 'clawpro vibecoding',
            agents,
            tasks:
              hasCanonicalTasks &&
              hasCanonicalPreVisitAgents &&
              hasCurrentMultiAgentBenchmark &&
              hasCurrentMultiAgentAssets
              ? project.tasks
                : [
                  taskById.get(TENANT_STANDARD_DEV_SOP_TASK_ID) ??
                    buildStandardDevelopmentSopTask(agents, project.members, now),
                  hasCurrentMultiAgentBenchmark
                    ? syncMultiAgentTaskAssets(
                        taskById.get(TENANT_MULTI_AGENT_DEV_TASK_ID)!,
                        agents,
                        project.members,
                      )
                    : buildMultiAgentDevelopmentTask(agents, project.members, now),
                  taskById.get(TENANT_PROJECT_MANAGEMENT_SOP_TASK_ID) ??
                    buildProjectManagementSopTask(agents, project.members, now),
                  taskById.get(TENANT_PRODUCT_FRONTEND_DEMO_SOP_TASK_ID) ??
                    buildProductFrontendDemoSopTask(agents, project.members, now),
                  hasCanonicalPreVisitAgents
                    ? taskById.get(TENANT_PRE_VISIT_BRIEF_TASK_ID)!
                    : buildPreVisitBriefTask(agents, project.members, now),
                  taskById.get(TENANT_KB_INSPECTION_TASK_ID) ??
                    buildKnowledgeInspectionTask(agents, project.members, now),
                  taskById.get(TENANT_WORKFLOW_DEMO_TASK_ID) ??
                    buildIssueFixDemoTask(agents, project.members, now),
                  taskById.get(TENANT_IMAGE_RELEASE_TASK_ID) ??
                    buildPublicImageReleaseTask(agents, project.members, now),
                ],
            pipelineTemplates:
              hasCanonicalTemplates &&
              hasCurrentMultiAgentBenchmarkTemplate &&
              hasCurrentMultiAgentTemplateAssets
              ? project.pipelineTemplates
                : [
                  templateById.get(TENANT_STANDARD_DEV_SOP_TEMPLATE_ID) ??
                    buildStandardDevelopmentSopPipelineTemplate(agents),
                  hasCurrentMultiAgentBenchmarkTemplate
                    ? syncMultiAgentTemplateAssets(
                        templateById.get(TENANT_MULTI_AGENT_DEV_TEMPLATE_ID)!,
                        agents,
                      )
                    : buildMultiAgentDevelopmentPipelineTemplate(agents),
                  templateById.get(TENANT_PROJECT_MANAGEMENT_SOP_TEMPLATE_ID) ??
                    buildProjectManagementSopPipelineTemplate(agents),
                  templateById.get(TENANT_PRODUCT_FRONTEND_DEMO_SOP_TEMPLATE_ID) ??
                    buildProductFrontendDemoSopPipelineTemplate(agents),
                  templateById.get(TENANT_PRE_VISIT_BRIEF_TEMPLATE_ID) ??
                    buildPreVisitBriefPipelineTemplate(agents),
                  templateById.get(TENANT_KB_INSPECTION_TEMPLATE_ID) ??
                    buildKnowledgeInspectionPipelineTemplate(),
                  templateById.get(TENANT_WORKFLOW_DEMO_TEMPLATE_ID) ??
                    buildIssueFixPipelineTemplate(agents),
                  templateById.get(TENANT_IMAGE_RELEASE_TEMPLATE_ID) ??
                    buildPublicImageReleasePipelineTemplate(),
                ],
          };
          return { ...next, metrics: recomputeMetrics(next) };
        });
        const existingIds = new Set(migratedExisting.map(project => project.id));
        const missingCatalogProjects = buildExpandedProjectCatalog(
          new Date().toISOString(),
        ).filter(project => !existingIds.has(project.id));
        if (missingCatalogProjects.length > 0) changed = true;
        const migrated = [...migratedExisting, ...missingCatalogProjects];
        if (changed) persist(migrated);
        return migrated;
      }
    }
  } catch {
    /* ignore */
  }
  const seed = buildSeed();
  persist(seed);
  return seed;
}

function ensure(): TenantProject[] {
  if (projects === null) projects = load();
  return projects;
}

function persist(next: TenantProject[]) {
  try {
    localStorage.setItem(CACHE_KEY, JSON.stringify(next));
    localStorage.setItem(CACHE_VERSION_KEY, CACHE_VERSION);
  } catch {
    /* ignore */
  }
}

function commit(next: TenantProject[]) {
  projects = next;
  persist(next);
  listeners.forEach((l) => l());
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent(TENANT_PROJECT_EVENT));
  }
}

/** 不可变更新单个项目 */
function updateProject(id: string, updater: (p: TenantProject) => TenantProject) {
  const next = ensure().map((p) => (p.id === id ? updater(p) : p));
  commit(next);
}

/** 重算派生指标（进行中任务、经验数、被学习数） */
function recomputeMetrics(p: TenantProject): TenantProjectMetrics {
  return {
    ...p.metrics,
    learningCount: p.learnings.length,
    weekRecalled: p.learnings.reduce((s, l) => s + l.recalledCount, 0),
    inProgressTasks: p.tasks.filter((t) => t.status === 'in_progress' || t.status === 'review').length,
  };
}

// ── 项目介绍 skill 派生（供 UI 显示 + MCP 接口 getProjectIntroSkill 返回） ──
/**
 * 由项目实时数据派生"项目介绍"skill 的内容 —— 完整版：
 *  - 项目名/描述/目标
 *  - 成员 + agent 列表
 *  - 工作流清单
 *  - 进行中任务概览 + 最近完成任务
 * 派生不存储，任何字段变化后调用自动返回最新内容。
 * 版本号 = project.assetVersion（保持与其他资产版本一致，随下发一起变更）。
 */
export interface ProjectIntroSkill {
  id: string;
  name: string;
  version: number;
  updatedAt: string;
  content: string;
}

export function getProjectIntroSkill(project: TenantProject): ProjectIntroSkill {
  const activeTasks = project.tasks
    .filter((t) => t.status === 'in_progress' || t.status === 'review')
    .slice(0, 8);
  const doneRecent = project.tasks
    .filter((t) => t.status === 'done')
    .slice(0, 5);
  const memberLines = project.members
    .map((m) => `  - ${m.displayName} (${m.userId})${m.admin ? ' [管理员]' : ''}`)
    .join('\n');
  const agentLines = project.agents
    .map((a) => {
      const owner =
        project.members.find((m) => m.userId === a.ownerId)?.displayName ??
        a.owner;
      return `  - ${a.name}（${a.kind === 'project' ? '项目公共' : `${owner} 的 agent`}，${a.location === 'local' ? '本地' : '云端'}）`;
    })
    .join('\n');
  const workflowLines = project.pipelineTemplates
    .map((w) => `  - ${w.name}（${w.nodes.length} 节点）：${w.description || '无描述'}`)
    .join('\n');
  const activeLines = activeTasks
    .map((t) => {
      const owner =
        project.members.find((m) => m.userId === t.ownerId)?.displayName ??
        t.ownerId;
      const done = t.workflow.filter((n) => n.status === 'confirmed').length;
      const total = t.workflow.length;
      return `  - [${t.status === 'review' ? '待确认' : '进行中'}] ${t.title} · ${owner} · 节点 ${done}/${total}`;
    })
    .join('\n');
  const doneLines = doneRecent
    .map((t) => `  - ${t.title}（${t.updatedAt.slice(0, 10)}）`)
    .join('\n');

  const content = `# 项目介绍：${project.name}

## 项目描述
${project.description || '（暂无描述）'}

## 项目目标
${project.goal || '（暂无目标）'}

## 成员（${project.members.length}）
${memberLines || '  （暂无）'}

## 项目 Agent（${project.agents.length}）
${agentLines || '  （暂无）'}

## 项目工作流（${project.pipelineTemplates.length}）
${workflowLines || '  （暂无）'}

## 进行中任务（${activeTasks.length}）
${activeLines || '  （暂无）'}

## 最近完成（${doneRecent.length}）
${doneLines || '  （暂无）'}

## 如何使用本项目
- 你（agent）已关联本项目，可直接读取以上上下文开展工作，无需打开 ClawPro 页面。
- 认领/推进任务：从"进行中任务"里挑选与你角色匹配的任务节点，产出后回传结果，节点确认后自动流转下游。
- 复用工作流：本项目工作流为标准 SOP，处理同类工作时按对应工作流的节点顺序执行。
- 资产即能力：项目下发的 skill/规范是你的常驻能力，动手前先据此对齐做法；踩坑/新解法会自动并入项目经验库。
- 产出规范：遵循项目汇报规格提交进展与结果，便于负责人确认与汇总。

## 工作流规则

### 核心概念
- 工作流 = 某类工作的标准 SOP（如代码评审、需求交付）
- 节点 = 一步工作，由一个 agent 执行
- 连线 = 流转依赖（上游全部确认后下游才启动）
- 支持并联：多上游 = 汇合，多下游 = 分支

### 节点配置
- **节点名称**：标识这一步做什么
- **Prompt（必填）**：给 agent 的指令，作为输入下发给工作流的第一个节点
- **派发给**：选择执行该节点的 agent
- **依赖上游**：勾选哪些节点完成后本节点才启动
- **数据流（自动）**：上游产出自动作为本节点输入，本节点产出自动作为下游输入

### 循环节点
- 节点类型设为 \`loop\` 时为循环节点，标记循环起点
- 循环节点不执行 agent，仅标记循环范围
- 配置循环结束节点（只能选循环节点的下游节点，形成闭环）
- 配置最大循环次数（默认3，防止死循环的兜底）
- 配置退出条件说明（提示文字，如"评审通过后退出循环"）
- **循环判断机制**：结束节点的 agent 执行完成后，在输出末尾标注 \`【循环：退出】\` 或 \`【循环：继续】\`。系统解析标记决定是否回退到循环节点重新执行。达到最大次数强制退出。

### JSON 导入格式
在新建工作流弹窗中点击「导入 JSON」可粘贴 JSON 快速创建工作流：

\`\`\`json
{
  "name": "工作流名称",
  "description": "一句话描述",
  "nodes": [
    {
      "title": "节点名称",
      "promptTemplate": "给 agent 的指令",
      "dependsOn": ["上游节点名称"],
      "agentRole": "角色描述（可选）"
    }
  ]
}
\`\`\`

字段说明：
- **name**（必填）：工作流名称
- **description**（可选）：一句话描述
- **nodes**（必填）：节点数组，每个节点包含：
  - **title**（必填）：节点名称
  - **promptTemplate**（必填）：给 agent 的指令
  - **dependsOn**（可选）：上游节点名称数组，用标题引用（不是 id）
  - **agentName**（可选）：指定执行的 agent 名称（需与项目中已接入的 agent 名称匹配）
  - **agentRole**（可选）：角色描述
  - **type**（可选）：节点类型，\`task\`（默认）或 \`loop\`（循环节点）
  - **loopConfig**（仅 type='loop' 时需要）：循环配置
    - **endNodeId**（必填）：循环结束节点名称（用标题引用）
    - **maxCount**（可选）：最大循环次数，默认3
    - **exitCondition**（可选）：退出条件说明

注意：dependsOn 和 loopConfig.endNodeId 用节点标题引用（不是 id），导入时自动解析。支持并联 DAG 和循环。

---
本skill 由 ClawPro 项目自动生成并随资产下发，任何项目字段变化后自动更新。
`;

  return {
    id: `intro-${project.id}`,
    name: `项目上手 · ${project.name}`,
    version: project.assetVersion,
    updatedAt: project.updatedAt,
    content,
  };
}

// ── 公共 API ────────────────────────────────────────────
function runtimePhaseStatusToWorkflowNodeStatus(
  status: string,
  fallback: TenantWorkflowNodeStatus,
): TenantWorkflowNodeStatus {
  switch (status) {
    case 'ready':
    case 'queued':
    case 'pending':
      return 'pending';
    case 'running':
      return 'running';
    case 'awaiting_approval':
    case 'waiting_approval':
      return 'review';
    case 'completed':
    case 'succeeded':
    case 'skipped':
      return 'confirmed';
    case 'failed':
    case 'error':
    case 'canceled':
      return 'stopped';
    default:
      return fallback;
  }
}

/**
 * 将后端真实执行阶段回写到任务工作流节点。
 *
 * 常规工作流优先用 runtimePhaseId 匹配；历史任务和精选工作流的阶段 id
 * 可能经过后端重写，因此继续用节点 id、标题和稳定顺序兜底。
 */
function syncWorkflowNodesFromRuntime(
  workflow: TenantWorkflowNode[],
  phases: TenantRuntimeExecutionPhase[],
): TenantWorkflowNode[] {
  const phaseById = new Map(phases.map((phase) => [phase.id, phase]));
  const phaseByTitle = new Map(phases.map((phase) => [phase.title, phase]));

  return workflow.map((node, index) => {
    const phase =
      (node.runtimePhaseId ? phaseById.get(node.runtimePhaseId) : undefined) ??
      phaseById.get(node.id) ??
      phaseByTitle.get(node.title) ??
      phases[index];
    if (!phase) return node;

    const status = runtimePhaseStatusToWorkflowNodeStatus(phase.status, node.status);
    return status === node.status ? node : { ...node, status };
  });
}

export const tenantProjectStore = {
  getAll(): TenantProject[] {
    return ensure();
  },
  getById(id: string): TenantProject | undefined {
    return ensure().find((p) => p.id === id);
  },
  /** 新建员工协作项目；创建人默认成为产品负责人并授权自己的 ClawPro Agent */
  createProject(input: {
    name: string;
    description: string;
    goal: string;
    allowMemberEdit: boolean;
  }): string {
    const now = new Date().toISOString();
    const id = `tp-${Date.now()}`;
    const members = [M(CURRENT_USER, 'alice', '产品负责人', true)];
    const agents = buildAgents(members);
    const project: TenantProject = {
      id,
      name: input.name,
      description: input.description,
      goal: input.goal,
      colorKey: input.name.slice(0, 1).toUpperCase() || 'P',
      members,
      agents,
      allowMemberEdit: input.allowMemberEdit,
      reportingSpec: defaultReportingSpec(),
      progressSnapshots: [],
      assets: emptyAssets(),
      assetVersion: 1,
      assetUpdateRecords: [
        {
          id: `tenant-asset-record-${id}-1`,
          groupId: id,
          version: 1,
          tagKind: 'manual',
          sections: [{ title: '创建项目资产配置' }],
          operator: CURRENT_USER,
          createdAt: now,
        },
      ],
      experienceSkillVersion: 1,
      experienceFlywheelEnabled: true,
      experienceRecallEnabled: true,
      experienceDepositionEnabled: true,
      metrics: {
        agentCount: agents.length,
        learningCount: 0,
        weekRecalled: 0,
        weekActiveSessions: 0,
        inProgressTasks: 0,
      },
      learnings: [],
      tasks: [],
      pipelineTemplates: defaultPipelineTemplates(),
      automations: [],
      skillUsage: [],
      activities: [],
      referencedCount: 0,
      referencedLibraries: [],
      updatedAt: now,
    };
    commit([project, ...ensure()]);
    return id;
  },
  /** 重命名项目；用户端与管控端共用项目 ID，由页面层同步共享项目 Store */
  renameProject(id: string, name: string) {
    updateProject(id, (p) => ({
      ...p,
      name,
      colorKey: name.slice(0, 1).toUpperCase() || p.colorKey,
      updatedAt: new Date().toISOString(),
    }));
  },
  /** 删除项目工作台数据；共享项目关系由页面层同步清理 */
  deleteProject(id: string) {
    commit(ensure().filter((project) => project.id !== id));
  },
  /** 邀请成员加入项目，同时把该成员授权的个人 Agent 加入项目可用池 */
  addProjectMember(
    id: string,
    member: Omit<TenantProjectMember, 'role'> & { role?: string },
  ) {
    updateProject(id, (p) => {
      if (p.members.some((item) => item.userId === member.userId)) return p;
      const full: TenantProjectMember = { role: '', ...member } as TenantProjectMember;
      const members = [...p.members, full];
      const agents = buildAgents(members);
      return {
        ...p,
        members,
        agents,
        metrics: { ...p.metrics, agentCount: agents.length },
        updatedAt: new Date().toISOString(),
      };
    });
  },
  /** 移除成员；创建人不可移除，相关任务负责人回退到创建人，失效 Agent 授权被清空 */
  removeProjectMember(id: string, userId: string) {
    if (userId === CURRENT_USER) return;
    updateProject(id, (p) => {
      const members = p.members.filter((member) => member.userId !== userId);
      const agents = buildAgents(members);
      const agentIds = new Set(agents.map((agent) => agent.id));
      const tasks = p.tasks.map((task) => ({
        ...task,
        ownerId: task.ownerId === userId ? CURRENT_USER : task.ownerId,
        workflow: task.workflow.map((node) =>
          node.agentId && !agentIds.has(node.agentId)
            ? { ...node, agentId: null, status: 'pending' as const }
            : node,
        ),
      }));
      const next = {
        ...p,
        members,
        agents,
        tasks,
        metrics: { ...p.metrics, agentCount: agents.length },
        updatedAt: new Date().toISOString(),
      };
      return { ...next, metrics: recomputeMetrics(next) };
    });
  },
  /** 切换某项目的「允许项目成员编辑」开关（管控端配置，管理员操作） */
  toggleAllowMemberEdit(id: string, value?: boolean) {
    updateProject(id, (p) => ({ ...p, allowMemberEdit: value ?? !p.allowMemberEdit }));
  },
  /** 保存某项目资产（version +1，其余字段保留） */
  saveAssets(id: string, assets: TenantProjectAssets, version?: number) {
    updateProject(id, (p) => {
      const nextAssets = ASSET_CATEGORY_ORDER.reduce((acc, c) => {
        acc[c] = [...(assets[c] ?? [])];
        return acc;
      }, {} as TenantProjectAssets);
      const nextVersion = version ?? p.assetVersion + 1;
      const createdAt = new Date().toISOString();
      const sections: ProjectAssetChangeSection[] = [];

      ASSET_CATEGORY_ORDER.forEach((category) => {
        const previous = new Set(p.assets[category] ?? []);
        const next = new Set(nextAssets[category] ?? []);
        const added = Array.from(next).filter((refId) => !previous.has(refId));
        const removed = Array.from(previous).filter((refId) => !next.has(refId));
        if (added.length > 0) {
          sections.push({
            title: `新增 ${added.length} 项${ASSET_CATEGORY_MAP[category].label}`,
            items: added.map((refId) => getAssetItemDisplay(category, refId).name),
          });
        }
        if (removed.length > 0) {
          sections.push({
            title: `删除 ${removed.length} 项${ASSET_CATEGORY_MAP[category].label}`,
            items: removed.map((refId) => getAssetItemDisplay(category, refId).name),
          });
        }
      });

      const record: ProjectAssetUpdateRecord = {
        id: `tenant-asset-record-${id}-${nextVersion}-${Date.now()}`,
        groupId: id,
        version: nextVersion,
        tagKind: 'manual',
        sections: sections.length > 0 ? sections : [{ title: '保存项目资产配置' }],
        operator: CURRENT_USER,
        createdAt,
      };

      return {
        ...p,
        assets: nextAssets,
        assetVersion: nextVersion,
        assetUpdateRecords: [record, ...(p.assetUpdateRecords ?? [])],
        updatedAt: createdAt,
      };
    });
  },
  /**
   * 新建任务；负责人必选。
   * - 若传 pipelineTemplateId：按项目自定义工作流实例化 DAG（新链路）
   * - 否则按内置 workflowTemplateId 生成 workflow（兼容旧链路）
   */
  createTask(
    id: string,
    input: {
      title: string;
      description: string;
      ownerId?: string;
      priority: TenantTaskPriority;
      dueDate: string;
      workflowTemplateId: TenantWorkflowTemplateId;
      workflowStages?: TenantWorkflowDraftStage[];
      /** 可选：项目自定义工作流 id；传了则按它实例化，覆盖 workflowTemplateId */
      pipelineTemplateId?: string;
      /** 业务标签 */
      tags?: string[];
    },
  ) {
    const now = new Date().toISOString();
    const taskId = `task-${Date.now()}`;
    updateProject(id, (p) => {
      const pipeline = input.pipelineTemplateId
        ? p.pipelineTemplates.find((t) => t.id === input.pipelineTemplateId)
        : undefined;
      const task: TenantTask = {
        id: taskId,
        title: input.title,
        description: input.description,
        status: 'todo',
        ownerId: input.ownerId || '',
        priority: input.priority,
        dueDate: input.dueDate,
        workflowTemplateId: input.workflowTemplateId,
        pipelineTemplateId: input.pipelineTemplateId,
        triggerType: 'manual',
        tags: input.tags,
        workflow: pipeline
          ? instantiatePipelineWorkflow(
              taskId,
              pipeline,
              p.agents,
              p.members,
              0,
              false,
            )
          : buildWorkflow(
              taskId,
              input.title,
              input.workflowTemplateId,
              p.agents,
              'todo',
              false,
            ),
        createdAt: now,
        updatedAt: now,
      };
      const activities = [
        {
          id: `act-${Date.now()}`,
          kind: 'task_dispatch' as const,
          text: pipeline
            ? `任务《${task.title}》已创建（走工作流「${pipeline.name}」）`
            : `任务《${task.title}》已创建，等待成员为工作流节点授权 Agent`,
          time: '刚刚',
        },
        ...p.activities,
      ];
      const withTask = { ...p, tasks: [task, ...p.tasks], activities };
      return { ...withTask, metrics: recomputeMetrics(withTask) };
    });
    return taskId;
  },
  /**
   * 外部触发（MCP/API）：本地 agent 调 clawpro 触发一条任务。
   * 场景：workbuddy 里 "帮我按代码评审流程 review 这个 PR" → agent 调此方法。
   * 与 createTask 的区别：triggerType='external' + 附外部上下文；结果结构化返回。
   */
  createTaskFromExternal(
    id: string,
    input: {
      title: string;
      pipelineTemplateId: string;
      source: string;
      requester: string;
      inputPrompt: string;
      taskInputs?: Record<string, IOValue>;
    },
  ): string | null {
    const now = new Date().toISOString();
    let taskId: string | null = null;
    updateProject(id, (p) => {
      const pipeline = p.pipelineTemplates.find(
        (t) => t.id === input.pipelineTemplateId,
      );
      if (!pipeline) return p;
      taskId = `task-ext-${Date.now()}`;
      const task: TenantTask = {
        id: taskId,
        title: input.title,
        description: `外部触发 · 来源 ${input.source} · 请求人 ${input.requester}`,
        status: 'in_progress',
        ownerId: input.requester,
        priority: 'medium',
        dueDate: now.slice(0, 10),
        workflowTemplateId: 'auto',
        pipelineTemplateId: input.pipelineTemplateId,
        triggerType: 'external',
        externalContext: {
          source: input.source,
          requester: input.requester,
          inputPrompt: input.inputPrompt,
        },
        taskInputs: input.taskInputs,
        workflow: instantiatePipelineWorkflow(
          taskId,
          pipeline,
          p.agents,
          p.members,
          0,
          true, // 自动启动首节点
        ),
        createdAt: now,
        updatedAt: now,
      };
      const activities = [
        {
          id: `act-${Date.now()}`,
          kind: 'task_dispatch' as const,
          text: `${input.source} 触发任务《${task.title}》，走工作流「${pipeline.name}」`,
          time: '刚刚',
        },
        ...p.activities,
      ];
      const next = {
        ...p,
        tasks: [task, ...p.tasks],
        activities,
        updatedAt: now,
      };
      return { ...next, metrics: recomputeMetrics(next) };
    });
    return taskId;
  },
  /** 更新任务标签 */
  updateTaskTags(id: string, taskId: string, tags: string[]) {
    updateProject(id, (p) => ({
      ...p,
      tasks: p.tasks.map((t) =>
        t.id === taskId ? { ...t, tags: [...tags] } : t,
      ),
      updatedAt: new Date().toISOString(),
    }));
  },
  /** 场景2：按流水线模板新建一条 issue，实例化 DAG 并按角色自动指派 agent */
  createIssue(
    id: string,
    input: {
      title: string;
      description?: string;
      ownerId?: string;
      priority?: TenantTaskPriority;
      dueDate?: string;
      pipelineTemplateId: string;
      source?: 'manual' | 'import' | 'mcp';
    },
  ): string | null {
    let newId: string | null = null;
    const now = new Date().toISOString();
    updateProject(id, (p) => {
      const template = p.pipelineTemplates.find(
        (t) => t.id === input.pipelineTemplateId,
      );
      if (!template) return p;
      const taskId = `issue-${Date.now()}`;
      newId = taskId;
      const task: TenantTask = {
        id: taskId,
        title: input.title,
        description: input.description ?? `【${template.name}】${input.title}`,
        status: 'in_progress',
        ownerId: input.ownerId ?? p.members[0]?.userId ?? CURRENT_USER,
        priority: input.priority ?? 'medium',
        dueDate: input.dueDate ?? '2026-08-31',
        workflowTemplateId: 'auto',
        workflow: instantiatePipelineWorkflow(
          taskId,
          template,
          p.agents,
          p.members,
          0,
          true,
        ),
        source: input.source ?? 'manual',
        pipelineTemplateId: template.id,
        createdAt: now,
        updatedAt: now,
      };
      const activities = [
        {
          id: `act-${Date.now()}`,
          kind: 'task_dispatch' as const,
          text: `流水线「${template.name}」新建 issue《${task.title}》，已按角色自动指派 agent`,
          time: '刚刚',
        },
        ...p.activities,
      ];
      const withTask = { ...p, tasks: [task, ...p.tasks], activities };
      return { ...withTask, metrics: recomputeMetrics(withTask) };
    });
    return newId;
  },
  /** 场景2：批量导入 issue（每行一条），统一走同一流水线模板 */
  createIssuesBatch(
    id: string,
    titles: string[],
    pipelineTemplateId: string,
  ): number {
    const clean = titles.map((t) => t.trim()).filter(Boolean);
    if (clean.length === 0) return 0;
    const baseNow = Date.now();
    updateProject(id, (p) => {
      const template = p.pipelineTemplates.find(
        (t) => t.id === pipelineTemplateId,
      );
      if (!template) return p;
      const iso = new Date().toISOString();
      const newTasks: TenantTask[] = clean.map((title, i) => {
        const taskId = `issue-${baseNow}-${i}`;
        return {
          id: taskId,
          title,
          description: `【${template.name}】${title}`,
          status: 'in_progress' as const,
          ownerId: p.members[0]?.userId ?? CURRENT_USER,
          priority: 'medium' as const,
          dueDate: '2026-08-31',
          workflowTemplateId: 'auto' as const,
          workflow: instantiatePipelineWorkflow(
            taskId,
            template,
            p.agents,
            p.members,
            0,
            true,
          ),
          source: 'import' as const,
          pipelineTemplateId: template.id,
          createdAt: iso,
          updatedAt: iso,
        };
      });
      const activities = [
        {
          id: `act-${baseNow}`,
          kind: 'task_dispatch' as const,
          text: `流水线「${template.name}」批量导入 ${newTasks.length} 条 issue，已自动指派 agent`,
          time: '刚刚',
        },
        ...p.activities,
      ];
      const withTasks = { ...p, tasks: [...newTasks, ...p.tasks], activities };
      return { ...withTasks, metrics: recomputeMetrics(withTasks) };
    });
    return clean.length;
  },
  /** 修改任务状态（供任务详情页状态下拉使用） */
  updateTaskStatus(id: string, taskId: string, status: TenantTaskStatus) {
    updateProject(id, (p) => {
      const task = p.tasks.find((t) => t.id === taskId);
      if (!task) return p;
      const next = {
        ...p,
        tasks: p.tasks.map((t) =>
          t.id === taskId
            ? { ...t, status, updatedAt: new Date().toISOString() }
            : t,
        ),
        updatedAt: new Date().toISOString(),
      };
      return { ...next, metrics: recomputeMetrics(next) };
    });
  },
  /** 修改成员：名称 / 管理员权限 */
  updateProjectMember(
    id: string,
    userId: string,
    patch: { displayName?: string; admin?: boolean },
  ) {
    updateProject(id, (p) => ({
      ...p,
      members: p.members.map((m) =>
        m.userId === userId ? { ...m, ...patch } : m,
      ),
      updatedAt: new Date().toISOString(),
    }));
  },
  /** 更新项目基本信息（名称/描述/目标） */
  updateProjectInfo(
    id: string,
    patch: { name?: string; description?: string; goal?: string },
  ) {
    updateProject(id, (p) => ({
      ...p,
      ...patch,
      updatedAt: new Date().toISOString(),
    }));
  },
  /**
   * 成员授权接入一个已在 ClawPro 注册的个人 agent 到本项目。
   * 现实链路：成员通过插件把本地 agent 注册进 ClawPro → 这里把已注册 agent 挂载进项目。
   * mock：新建一条 TenantAgent（kind='personal'）加入 project.agents。
   */
  attachPersonalAgent(
    id: string,
    input: {
      name: string;
      ownerId: string;
      owner: string;
      platform?: TenantAgent['platform'];
      location?: TenantAgentLocation;
      authorization?: string;
    },
  ): string {
    const agentId = `ag-${Date.now()}`;
    updateProject(id, (p) => ({
      ...p,
      agents: [
        ...p.agents,
        {
          id: agentId,
          name: input.name,
          platform: input.platform ?? 'clawpro',
          location: input.location ?? 'cloud',
          ownerId: input.ownerId,
          owner: input.owner,
          status: 'online',
          authorization: input.authorization ?? '项目上下文与授权云端工具',
          kind: 'personal',
        },
      ],
      updatedAt: new Date().toISOString(),
    }));
    return agentId;
  },
  /**
   * 方式1：列出当前用户在 ClawPro 已注册、可直接授权接入项目的个人 agent（mock 池）。
   * 已接入本项目的会被过滤掉。
   */
  getMyAgents(id: string, userId: string): MyRegisteredAgent[] {
    const p = ensure().find((x) => x.id === id);
    const attached = new Set((p?.agents ?? []).map((a) => a.name));
    return MY_REGISTERED_AGENTS.filter((a) => !attached.has(a.name)).filter(
      () => userId, // userId 占位：真实场景按登录用户过滤
    );
  },
  /** 方式1：把"我的已有 agent"直接授权接入项目 */
  attachMyAgent(id: string, userId: string, agentId: string): string | null {
    const src = MY_REGISTERED_AGENTS.find((a) => a.id === agentId);
    if (!src) return null;
    const owner =
      ensure()
        .find((x) => x.id === id)
        ?.members.find((m) => m.userId === userId)?.displayName ?? userId;
    return this.attachPersonalAgent(id, {
      name: src.name,
      ownerId: userId,
      owner,
      platform: src.platform,
    });
  },
  /**
   * 方式2：接入外部 agent —— 生成接入码进入等待态。
   * mock：readyAt 设为若干秒后，前端轮询 checkPendingAgent 到点即可捕获。
   */
  startPendingAgentConnection(input: {
    name: string;
    platform: TenantAgent['platform'];
  }): PendingAgentConnection {
    const now = Date.now();
    return {
      code: `CP-${Math.random().toString(36).slice(2, 8).toUpperCase()}`,
      createdAt: now,
      readyAt: now + 8000, // 模拟外部安装 + 注册耗时约 8s
      name: input.name,
      platform: input.platform,
    };
  },
  /**
   * 方式2：轮询检查外部 agent 是否已注册进来。
   * 到 readyAt 之后返回 true 并把agent 落进项目（授权给当前用户）。
   */
  checkPendingAgent(
    id: string,
    userId: string,
    pending: PendingAgentConnection,
  ): { ready: boolean; agentId?: string } {
    if (Date.now() < pending.readyAt) return { ready: false };
    const owner =
      ensure()
        .find((x) => x.id === id)
        ?.members.find((m) => m.userId === userId)?.displayName ?? userId;
    const agentId = this.attachPersonalAgent(id, {
      name: pending.name,
      ownerId: userId,
      owner,
      platform: pending.platform,
    });
    return { ready: true, agentId };
  },
  /** 新建项目自动化 */
  createAutomation(
    id: string,
    input: Omit<TenantAutomation, 'id' | 'createdAt' | 'nextRunAt'>,
  ): string {
    const autoId = `auto-${Date.now()}`;
    updateProject(id, (p) => ({
      ...p,
      automations: [
        ...p.automations,
        {
          ...input,
          id: autoId,
          createdAt: new Date().toISOString(),
          nextRunAt: input.enabled
            ? computeNextRunAt(input.schedule)
            : undefined,
        },
      ],
      updatedAt: new Date().toISOString(),
    }));
    return autoId;
  },
  /** 更新自动化 */
  updateAutomation(
    id: string,
    automationId: string,
    patch: Partial<Omit<TenantAutomation, 'id' | 'createdAt'>>,
  ) {
    updateProject(id, (p) => ({
      ...p,
      automations: p.automations.map((a) => {
        if (a.id !== automationId) return a;
        const merged = { ...a, ...patch };
        merged.nextRunAt = merged.enabled
          ? computeNextRunAt(merged.schedule)
          : undefined;
        return merged;
      }),
      updatedAt: new Date().toISOString(),
    }));
  },
  /** 启停自动化 */
  toggleAutomation(id: string, automationId: string, enabled: boolean) {
    this.updateAutomation(id, automationId, { enabled });
  },
  /** 删除自动化 */
  deleteAutomation(id: string, automationId: string) {
    updateProject(id, (p) => ({
      ...p,
      automations: p.automations.filter((a) => a.id !== automationId),
      updatedAt: new Date().toISOString(),
    }));
  },
  /**
   * 立即运行一次自动化（测试运行 / 演示闭环）。
   * outputMode=createTask：在项目里建一条任务（可走关联工作流），返回 taskId。
   * outputMode=runOnly：静默运行，仅记录动态，返回 null。
   */
  runAutomationNow(id: string, automationId: string): string | null {
    const p = ensure().find((x) => x.id === id);
    const auto = p?.automations.find((a) => a.id === automationId);
    if (!p || !auto) return null;
    const now = new Date().toISOString();
    // 先更新 lastRunAt / nextRunAt
    this.updateAutomation(id, automationId, { lastRunAt: now });
    if (auto.outputMode === 'runOnly') {
      updateProject(id, (proj) => ({
        ...proj,
        activities: [
          {
            id: `act-${Date.now()}`,
            kind: 'automation_run' as const,
            text: `自动化「${auto.name}」已运行（仅运行，未建任务）`,
            time: '刚刚',
          },
          ...proj.activities,
        ],
        updatedAt: now,
      }));
      return null;
    }
    // createTask：按关联工作流实例化 DAG，执行者由工作流节点自己的 executor 决定
    const taskId = this.createTask(id, {
      title: `[自动化] ${auto.name}`,
      description: auto.prompt || `自动化「${auto.name}」触发的任务`,
      priority: 'medium',
      dueDate: now.slice(0, 10),
workflowTemplateId: 'auto',
      pipelineTemplateId: auto.pipelineTemplateId,
      tags: ['自动化'],
    });
    // 标记该任务为 system 触发
    if (taskId) {
      updateProject(id, (proj) => ({
        ...proj,
        tasks: proj.tasks.map((t) =>
          t.id === taskId ? { ...t, triggerType: 'system' as const } : t,
        ),
        activities: [
          {
            id: `act-${Date.now()}`,
            kind: 'automation_run' as const,
            text: `自动化「${auto.name}」运行并创建任务《${auto.name}》`,
            time: '刚刚',
          },
          ...proj.activities,
        ],
        updatedAt: now,
      }));
    }
    return taskId;
  },
  /**
   * 新建一个项目公共 agent（不绑具体个人，供工作流节点直接派发）。
   * 场景：日报机器人、自动巡检等，全项目共享。
   */
  createProjectAgent(
    id: string,
    input: {
      name: string;
      platform?: TenantAgent['platform'];
      location?: TenantAgentLocation;
      authorization?: string;
    },
  ): string {
    const agentId = `ag-proj-${Date.now()}`;
    updateProject(id, (p) => ({
      ...p,
      agents: [
        ...p.agents,
        {
          id: agentId,
          name: input.name,
          platform: input.platform ?? 'clawpro',
          location: input.location ?? 'cloud',
          ownerId: '__project__',
          owner: '项目公共',
          status: 'online',
          authorization: input.authorization ?? '项目公共 agent',
          kind: 'project',
        },
      ],
      updatedAt: new Date().toISOString(),
    }));
    return agentId;
  },
  /** 新建项目级流水线模板 */
  createPipelineTemplate(
    id: string,
    input: {
      name: string;
      description: string;
      nodes: TenantPipelineTemplateNode[];
    },
  ): string {
    const templateId = `pl-${Date.now()}`;
    updateProject(id, (p) => ({
      ...p,
      pipelineTemplates: [
        ...p.pipelineTemplates,
        {
          id: templateId,
          name: input.name,
          description: input.description,
          nodes: input.nodes,
        },
      ],
      updatedAt: new Date().toISOString(),
    }));
    return templateId;
  },
  /** 更新流水线模板（名称/描述/节点） */
  updatePipelineTemplate(
    id: string,
    templateId: string,
    patch: Partial<Omit<TenantPipelineTemplate, 'id'>>,
  ) {
    updateProject(id, (p) => ({
      ...p,
      pipelineTemplates: p.pipelineTemplates.map((t) =>
        t.id === templateId ? { ...t, ...patch } : t,
      ),
      updatedAt: new Date().toISOString(),
    }));
  },
  /**
   * 场景（P3）：接收 agent 经 MCP 回传的工作流，校验通过后落库为流水线模板。
   * 体现 ClawPro 自身使用也是 AI-Native——agent「画」工作流并回传，无需人工在画布拖拽。
   * @returns 成功返回新模板 id；失败返回 null。
   */
  receiveWorkflowFromMCP(
    id: string,
    nodes: TenantPipelineTemplateNode[],
    meta: { name: string; description: string },
  ): string | null {
    if (!nodes || nodes.length === 0) return null;
    const templateId = `pl-mcp-${Date.now()}`;
    updateProject(id, (p) => {
      const activities = [
        {
          id: `act-${Date.now()}`,
          kind: 'skill_convert' as const,
          text: `agent 经 MCP 回传工作流「${meta.name}」，已落库为流水线模板`,
          time: '刚刚',
        },
        ...p.activities,
      ];
      return {
        ...p,
        pipelineTemplates: [
          ...p.pipelineTemplates,
          {
            id: templateId,
            name: meta.name,
            description: meta.description || '由 agent 经 MCP 回传',
            nodes: nodes.map((n) => ({
              id: n.id,
              title: n.title,
              dependsOn: [...n.dependsOn],
              agentRole: n.agentRole,
            })),
          },
        ],
        activities,
        updatedAt: new Date().toISOString(),
      };
    });
    return templateId;
  },
  /**
   * 场景（P3）：把跑顺的一条 issue 的 workflow 结构「固化」为流水线模板。
   * 反推：节点标题保留，dependsOn 映射为顺序内新 id，agentRole 从节点当前 agent 归属成员的 role 回填。
   * @returns 成功返回新模板 id；失败返回 null。
   */
  solidifyIssueAsTemplate(
    id: string,
    taskId: string,
    name: string,
  ): string | null {
    let templateId: string | null = null;
    updateProject(id, (p) => {
      const task = p.tasks.find((t) => t.id === taskId);
      if (!task || task.workflow.length === 0) return p;
      const idMap: Record<string, string> = {};
      task.workflow.forEach((n, i) => {
        idMap[n.id] = `n${i + 1}`;
      });
      const nodes: TenantPipelineTemplateNode[] = task.workflow.map((n) => {
        const agent = p.agents.find((a) => a.id === n.agentId);
        const member = agent
          ? p.members.find((m) => m.userId === agent.ownerId)
          : undefined;
        return {
          id: idMap[n.id],
          title: n.title,
          dependsOn: n.dependsOn.map((d) => idMap[d]).filter(Boolean),
          agentRole: member?.role ?? '成员',
        };
      });
      templateId = `pl-solidify-${Date.now()}`;
      const activities = [
        {
          id: `act-${Date.now()}`,
          kind: 'skill_convert' as const,
          text: `已把 issue《${task.title}》的执行流程固化为流水线模板「${name}」`,
          time: '刚刚',
        },
        ...p.activities,
      ];
      return {
        ...p,
        pipelineTemplates: [
          ...p.pipelineTemplates,
          {
            id: templateId,
            name,
            description: `由 issue《${task.title}》历史执行固化生成`,
            nodes,
          },
        ],
        activities,
        updatedAt: new Date().toISOString(),
      };
    });
    return templateId;
  },
  /** 统计某流水线模板被多少条 issue 实例化引用（模板复用次数，北极星铺垫，派生不存储） */
  countTemplateUsage(id: string, templateId: string): number {
    const p = ensure().find((x) => x.id === id);
    if (!p) return 0;
    return p.tasks.filter((t) => t.pipelineTemplateId === templateId).length;
  },
  /** 删除流水线模板（保留已生成的 issue） */
  deletePipelineTemplate(id: string, templateId: string) {
    updateProject(id, (p) => ({
      ...p,
      pipelineTemplates: p.pipelineTemplates.filter((t) => t.id !== templateId),
      updatedAt: new Date().toISOString(),
    }));
  },
  /** 删除项目任务及其工作流数据 */
  deleteTask(id: string, taskId: string) {
    updateProject(id, (p) => {
      const removedTask = p.tasks.find((task) => task.id === taskId);
      const tasks = p.tasks.filter((task) => task.id !== taskId);
      const activities = removedTask
        ? [
            {
              id: `act-${Date.now()}`,
              kind: 'task_dispatch' as const,
              text: `任务《${removedTask.title}》已删除`,
              time: '刚刚',
            },
            ...p.activities,
          ]
        : p.activities;
      const next = {
        ...p,
        tasks,
        activities,
        updatedAt: new Date().toISOString(),
      };
      return { ...next, metrics: recomputeMetrics(next) };
    });
  },
  /** 为工作流节点选择项目成员已授权的个人 Agent */
  assignWorkflowAgent(id: string, taskId: string, nodeId: string, agentId: string) {
    const now = new Date().toISOString();
    updateProject(id, (p) => {
      const tasks = p.tasks.map((t) => {
        if (t.id !== taskId) return t;
        return {
          ...t,
          workflow: t.workflow.map((node) =>
            node.id === nodeId ? { ...node, agentId } : node,
          ),
          updatedAt: now,
        };
      });
      return { ...p, tasks };
    });
  },
  /** 所有节点完成 Agent 授权后，同时启动所有无上游依赖的起始节点 */
  startTask(id: string, taskId: string) {
    const now = new Date().toISOString();
    updateProject(id, (p) => {
      let title = '';
      const tasks = p.tasks.map((t) => {
        if (t.id !== taskId) return t;
        title = t.title;
        return {
          ...t,
          status: 'in_progress' as const,
          workflow: t.workflow.map((node) =>
            node.status === 'pending' && node.dependsOn.length === 0
              ? { ...node, status: 'running' as const }
              : node,
          ),
          updatedAt: now,
        };
      });
      const activities = [
        {
          id: `act-${Date.now()}`,
          kind: 'task_dispatch' as const,
          text: `任务《${title}》已开始执行`,
          time: '刚刚',
        },
        ...p.activities,
      ];
      const withTask = { ...p, tasks, activities };
      return { ...withTask, metrics: recomputeMetrics(withTask) };
    });
  },
  /** 将项目任务绑定到真实 ClawPro 编排任务。 */
  bindRuntimeExecution(
    id: string,
    taskId: string,
    runtimeExecution: TenantRuntimeExecution,
  ) {
    updateProject(id, (p) => {
      const task = p.tasks.find((item) => item.id === taskId);
      const tasks = p.tasks.map((item) =>
        item.id === taskId
          ? {
              ...item,
              status: 'in_progress' as const,
              workflow: syncWorkflowNodesFromRuntime(
                item.workflow,
                runtimeExecution.phases,
              ),
              runtimeExecution,
              updatedAt: runtimeExecution.updatedAt,
            }
          : item,
      );
      const activities = task
        ? [
            {
              id: `act-${Date.now()}`,
              kind: 'task_dispatch' as const,
              text: `任务《${task.title}》已提交到真实 Agent 工作流`,
              time: '刚刚',
            },
            ...p.activities,
          ]
        : p.activities;
      const next = { ...p, tasks, activities };
      return { ...next, metrics: recomputeMetrics(next) };
    });
  },
  /** 保留任务定义，清空上一次 Runtime 快照，返回真实 Agent 节点配置态。 */
  prepareRuntimeRerun(id: string, taskId: string) {
    const now = new Date().toISOString();
    updateProject(id, (p) => {
      const tasks = p.tasks.map((item) => {
        if (item.id !== taskId) return item;
        return {
          ...item,
          runtimeExecution: undefined,
          status: 'todo' as const,
          workflow: item.workflow.map((node) => ({
            ...node,
            status: 'pending' as const,
            result: null,
            artifacts: [],
          })),
          updatedAt: now,
        };
      });
      const next = { ...p, tasks, updatedAt: now };
      return { ...next, metrics: recomputeMetrics(next) };
    });
  },
  /** 接收后端轮询快照，并将状态、节点和产物持久化到项目任务。 */
  syncRuntimeExecution(
    id: string,
    taskId: string,
    runtimeExecution: TenantRuntimeExecution,
  ) {
    updateProject(id, (p) => {
      const tasks = p.tasks.map((item) => {
        if (item.id !== taskId) return item;
        const status: TenantTaskStatus =
          runtimeExecution.status === 'completed'
            ? 'done'
            : runtimeExecution.status === 'failed' ||
                runtimeExecution.status === 'canceled'
              ? 'hold'
              : 'in_progress';
        return {
          ...item,
          status,
          workflow: syncWorkflowNodesFromRuntime(
            item.workflow,
            runtimeExecution.phases,
          ),
          runtimeExecution,
          updatedAt: runtimeExecution.updatedAt,
        };
      });
      const next = { ...p, tasks };
      return { ...next, metrics: recomputeMetrics(next) };
    });
  },
  /** 停止正在运行的节点；不流转下游，保留当前执行上下文供重试 */
  stopWorkflowNode(id: string, taskId: string, nodeId: string) {
    const now = new Date().toISOString();
    updateProject(id, (p) => {
      const tasks = p.tasks.map((t) =>
        t.id === taskId
          ? {
              ...t,
              status: 'in_progress' as const,
              workflow: t.workflow.map((node) =>
                node.id === nodeId && node.status === 'running'
                  ? { ...node, status: 'stopped' as const }
                  : node,
              ),
              updatedAt: now,
            }
          : t,
      );
      const next = { ...p, tasks };
      return { ...next, metrics: recomputeMetrics(next) };
    });
  },
  /** 从停止状态重试节点；结果与附件在本轮执行完成前保持为空 */
  retryWorkflowNode(id: string, taskId: string, nodeId: string) {
    const now = new Date().toISOString();
    updateProject(id, (p) => {
      const tasks = p.tasks.map((t) =>
        t.id === taskId
          ? {
              ...t,
              status: 'in_progress' as const,
              workflow: t.workflow.map((node) =>
                node.id === nodeId && node.status === 'stopped'
                  ? {
                      ...node,
                      status: 'running' as const,
                      result: null,
                      artifacts: [],
                    }
                  : node,
              ),
              updatedAt: now,
            }
          : t,
      );
      const next = { ...p, tasks };
      return { ...next, metrics: recomputeMetrics(next) };
    });
  },
  /**
   * Agent 完成节点：直接 confirmed（全自动流转，不再等人确认），
   * 并把依赖已全部满足的下游节点自动启动为 running。
   * 如需人工卡口，在工作流里画一个 executor=成员 的审核节点即可。
   */
  completeWorkflowNode(id: string, taskId: string, nodeId: string) {
    const now = new Date().toISOString();
    updateProject(id, (p) => {
      let workflowFinished = false;
      let completedTitle = '';
      const tasks = p.tasks.map((t) => {
        if (t.id !== taskId) return t;
        // 1. 本节点直接置 confirmed + 写产出
        const done = t.workflow.map((node) => {
          if (node.id !== nodeId) return node;
          // 如果是循环节点的结束节点，在输出中附上循环判断标记（模拟 agent 判断）
          const loopNodeForThis = t.workflow.find(
            (n) => n.type === 'loop' && n.loopConfig?.endNodeId === nodeId,
          );
          let resultText = `${node.title}已执行完成。`;
          if (loopNodeForThis?.loopConfig) {
            const lc = loopNodeForThis.loopCount ?? 0;
            const mc = loopNodeForThis.loopConfig.maxCount ?? 3;
            resultText += lc >= mc - 1
              ? '\n\n【循环：退出】'
              : '\n\n【循环：继续】';
          }
          return {
            ...node,
            status: 'confirmed' as const,
            result: resultText,
            artifacts: buildArtifacts(t.title, node.title, node.id),
            outputValues: {
              result: autoOutputValue(node.id, resultText),
            },
          };
        });
        // 1.5 循环检查：当前confirmed的节点是否是某个循环节点的结束节点
        let workflow = done;
        const loopNode = done.find(
          (n) => n.type === 'loop' && n.loopConfig?.endNodeId === nodeId,
        );
        // 解析结束节点 agent 输出中的循环判断标记
        const completedNode = done.find((n) => n.id === nodeId);
        const shouldExitLoop = (completedNode?.result ?? '').includes(
          '【循环：退出】',
        );
        if (
          loopNode?.loopConfig &&
          !shouldExitLoop &&
          (loopNode.loopCount ?? 0) <
            (loopNode.loopConfig.maxCount ?? 3)
        ) {
          const loopIds = new Set<string>([loopNode.id, nodeId]);
          const queue = [nodeId];
          const visited = new Set<string>();
          while (queue.length > 0) {
            const cur = queue.shift()!;
            if (visited.has(cur)) continue;
            visited.add(cur);
            const n = done.find((d) => d.id === cur);
            if (!n) continue;
            for (const dep of n.dependsOn) {
              loopIds.add(dep);
              if (dep !== loopNode.id) queue.push(dep);
            }
          }
          workflow = done.map((n) =>
            loopIds.has(n.id)
              ? {
                  ...n,
                  status: 'pending' as const,
                  result: null,
                  artifacts: [],
                  loopCount: (n.loopCount ?? 0) + 1,
                }
              : n,
          );
        }
        // 2. 依赖全部 confirmed 的 pending 节点自动启动
        const confirmedIds = new Set(
          workflow.filter((n) => n.status === 'confirmed').map((n) => n.id),
        );
        workflow = workflow.map((node) =>
          node.status === 'pending' &&
          node.dependsOn.every((d) => confirmedIds.has(d))
            ? { ...node, status: 'running' as const }
            : node,
        );
        workflowFinished = workflow.every((n) => n.status === 'confirmed');
        if (workflowFinished) completedTitle = t.title;
        return {
    ...t,
     status: workflowFinished ? ('done' as const) : ('in_progress' as const),
        workflow,
  updatedAt: now,
     };
      });
   const activities = workflowFinished && completedTitle
      ? [
          {
  id: `act-${Date.now()}`,
kind: 'task_done' as const,
       text: `任务《${completedTitle}》已全部完成`,
    time: '刚刚',
          },
          ...p.activities,
    ]
      : p.activities;
      const withTask = { ...p, tasks, activities };
    return { ...withTask, metrics: recomputeMetrics(withTask) };
    });
  },
  /** 联调失败时退回直接上游 Agent 修复，修复确认后重新触发当前节点 */
  returnWorkflowNode(id: string, taskId: string, nodeId: string) {
    const now = new Date().toISOString();
    updateProject(id, (p) => {
      let taskTitle = '';
      let upstreamTitle = '';
      const tasks = p.tasks.map((t) => {
        if (t.id !== taskId) return t;
        taskTitle = t.title;
        const target = t.workflow.find((node) => node.id === nodeId);
        const upstreamId = target?.dependsOn[target.dependsOn.length - 1];
        const upstream = t.workflow.find((node) => node.id === upstreamId);
        upstreamTitle = upstream?.title ?? '上游节点';
        return {
          ...t,
          status: 'in_progress' as const,
          workflow: t.workflow.map((node) => {
            if (node.id === upstreamId) {
              return {
                ...node,
                status: 'running' as const,
                result: null,
                artifacts: [],
              };
            }
            if (node.id === nodeId) {
              return {
                ...node,
                status: 'pending' as const,
                result: null,
                artifacts: [],
              };
            }
            return node;
          }),
          updatedAt: now,
        };
      });
      const next = {
        ...p,
        tasks,
        activities: [
          {
            id: `act-${Date.now()}`,
            kind: 'task_dispatch' as const,
            text: `任务《${taskTitle}》已由联调节点退回“${upstreamTitle}”修复`,
            time: '刚刚',
          },
          ...p.activities,
        ],
      };
      return { ...next, metrics: recomputeMetrics(next) };
    });
  },
  /** 确认节点结果后，启动全部依赖已确认的就绪节点；并行分支全部确认后汇合 */
  confirmWorkflowNode(id: string, taskId: string, nodeId: string) {
    const now = new Date().toISOString();
    updateProject(id, (p) => {
      let completedTitle = '';
      let workflowFinished = false;
      const tasks = p.tasks.map((t) => {
        if (t.id !== taskId) return t;
        const confirmed = t.workflow.map((node) =>
          node.id === nodeId ? { ...node, status: 'confirmed' as const } : node,
        );
        const confirmedIds = new Set(
          confirmed
            .filter((node) => node.status === 'confirmed')
            .map((node) => node.id),
        );
        const workflow = confirmed.map((node) =>
          node.status === 'pending' &&
          node.dependsOn.every((dependency) => confirmedIds.has(dependency))
            ? { ...node, status: 'running' as const }
            : node,
        );
        workflowFinished = workflow.every((node) => node.status === 'confirmed');
        if (workflowFinished) completedTitle = t.title;
        const awaitingReview = workflow.some((node) => node.status === 'review');
        return {
          ...t,
          status: workflowFinished
            ? ('done' as const)
            : awaitingReview
              ? ('review' as const)
              : ('in_progress' as const),
          workflow,
          updatedAt: now,
        };
      });
      const activities = workflowFinished
        ? [
            {
              id: `act-${Date.now()}`,
              kind: 'task_done' as const,
              text: `任务《${completedTitle}》全部节点已确认`,
              time: '刚刚',
            },
            ...p.activities,
          ]
        : p.activities;
      const withTask = { ...p, tasks, activities };
      return { ...withTask, metrics: recomputeMetrics(withTask) };
    });
  },
  /** 同步「项目经验库 Skill」并下发：版本 +1，整体下发给项目 agent（V2） */
  syncExperienceSkill(id: string) {
    updateProject(id, (p) => {
      const nextVer = p.experienceSkillVersion + 1;
      const activities = [
        { id: `act-${Date.now()}`, kind: 'skill_convert' as const, text: `项目经验库 Skill 已更新到 v${nextVer}，整体下发给 ${p.agents.length} 个 agent`, time: '刚刚' },
        ...p.activities,
      ];
      return { ...p, experienceSkillVersion: nextVer, activities };
    });
  },
  /** 分别控制任务前经验检索和任务中经验沉淀。 */
  setExperienceAutomationEnabled(
    id: string,
    capability: 'recall' | 'deposition',
    enabled: boolean,
  ) {
    updateProject(id, (p) => ({
      ...p,
      ...(capability === 'recall'
        ? { experienceRecallEnabled: enabled }
        : { experienceDepositionEnabled: enabled }),
      activities: [
        {
          id: `act-${Date.now()}`,
          kind: 'skill_convert' as const,
          text: `${capability === 'recall' ? 'Token 节省' : '经验沉淀'}已${enabled ? '开启' : '关闭'}`,
          time: '刚刚',
        },
        ...p.activities,
      ],
      updatedAt: new Date().toISOString(),
    }));
  },
  /** 在 Agent 社区把某个源项目的经验库「引用/链接」到目标项目（复用下发语义，演示态） */
  referenceLibrary(targetId: string, sourceId: string) {
    if (targetId === sourceId) return;
    const source = ensure().find((p) => p.id === sourceId);
    const sourceName = source?.name ?? '';
    const next = ensure().map((p) => {
      if (p.id === targetId) {
        if (p.referencedLibraries.includes(sourceId)) return p;
        const activities = [
          { id: `act-${Date.now()}`, kind: 'skill_convert' as const, text: `引用了「${sourceName} · 项目经验库」，已下发给本项目 agent`, time: '刚刚' },
          ...p.activities,
        ];
        return { ...p, referencedLibraries: [...p.referencedLibraries, sourceId], activities };
      }
      if (p.id === sourceId) {
        return { ...p, referencedCount: p.referencedCount + 1 };
      }
      return p;
    });
    commit(next);
  },
  /** 切换成员的管理员身份（管理员操作）；至少保留一个管理员 */
  toggleMemberAdmin(id: string, userId: string, value?: boolean) {
    updateProject(id, (p) => {
      const target = p.members.find((m) => m.userId === userId);
      if (!target) return p;
      const nextVal = value ?? !target.admin;
      // 取消最后一个管理员则拒绝
      if (!nextVal && p.members.filter((m) => m.admin).length <= 1) return p;
      return {
        ...p,
        members: p.members.map((m) =>
          m.userId === userId ? { ...m, admin: nextVal } : m,
        ),
        updatedAt: new Date().toISOString(),
      };
    });
  },
  /** 保存汇报规格（项目配置层·管理员整体规划） */
  saveReportingSpec(id: string, patch: Partial<TenantReportingSpec>) {
    updateProject(id, (p) => ({
      ...p,
      reportingSpec: { ...p.reportingSpec, ...patch },
      updatedAt: new Date().toISOString(),
    }));
  },
  /**
   * 派发本周期汇报任务：按汇报规格给范围内成员各建一条汇报任务 + 一条待填快照。
   * 快照的进度底座此刻即从各成员名下任务机械算出；解读层留空待 agent/人产出。
   */
  dispatchReports(id: string): number {
    let count = 0;
    const now = new Date().toISOString();
    const { key, label } = isoWeekKey();
    updateProject(id, (p) => {
      const spec = p.reportingSpec;
      if (!spec.enabled) return p;
      const scope =
        spec.reporterScope.length > 0
          ? p.members.filter((m) => spec.reporterScope.includes(m.userId))
          : p.members;
      // 本周期已派发过的成员跳过
      const existing = new Set(
        p.progressSnapshots
          .filter((s) => s.period === key)
          .map((s) => s.reporterId),
      );
      const targets = scope.filter((m) => !existing.has(m.userId));
      if (targets.length === 0) return p;
      count = targets.length;
      const newTasks: TenantTask[] = [];
      const newSnaps: TenantProgressSnapshot[] = [];
      targets.forEach((m, i) => {
        const taskId = `report-${key}-${m.userId}-${Date.now()}-${i}`;
        newTasks.push({
          id: taskId,
          title: `${label} · ${m.displayName} 进展汇报`,
          description: `按项目汇报规格提交 ${label} 进展快照`,
          status: 'in_progress',
          ownerId: m.userId,
          priority: 'medium',
          dueDate: '2026-08-31',
          taskType: 'report',
          reportPeriod: key,
          workflowTemplateId: 'auto',
          workflow: [],
          createdAt: now,
          updatedAt: now,
        });
        newSnaps.push({
          id: `snap-${key}-${m.userId}-${Date.now()}-${i}`,
          period: key,
          periodLabel: label,
          reporterId: m.userId,
          reporterName: m.displayName,
          taskId,
          progressBase: computeProgressBase(p.tasks, m.userId),
          interpretation: {},
          status: 'pending',
          submittedAt: null,
          createdAt: now,
        });
      });
      const activities = [
        {
          id: `act-${Date.now()}`,
          kind: 'report_dispatch' as const,
          text: `管理员派发了「${label}」汇报任务给 ${targets.length} 名成员，规格已随任务下发给其 agent`,
          time: '刚刚',
        },
        ...p.activities,
      ];
      return {
        ...p,
        tasks: [...newTasks, ...p.tasks],
        progressSnapshots: [...newSnaps, ...p.progressSnapshots],
        activities,
      };
    });
    return count;
  },
  /**
   * 提交进展快照（成员/其 agent 按规格填解读层）。
   * 进度底座重新机械算一次（保证提交时是最新客观进度），解读层用入参覆盖。
   */
  submitSnapshot(
    id: string,
    snapshotId: string,
    interpretation: Record<string, string>,
  ) {
    const now = new Date().toISOString();
    updateProject(id, (p) => {
      const snap = p.progressSnapshots.find((s) => s.id === snapshotId);
      if (!snap) return p;
      const progressSnapshots = p.progressSnapshots.map((s) =>
        s.id === snapshotId
          ? {
              ...s,
              progressBase: computeProgressBase(p.tasks, s.reporterId),
              interpretation,
              status: 'submitted' as const,
              submittedAt: now,
            }
          : s,
      );
      // 对应汇报任务置为已完成
      const tasks = p.tasks.map((t) =>
        t.id === snap.taskId ? { ...t, status: 'done' as const, updatedAt: now } : t,
      );
      const activities = [
        {
          id: `act-${Date.now()}`,
          kind: 'report_submit' as const,
          text: `${snap.reporterName} 的 agent 按规格提交了「${snap.periodLabel}」进展快照`,
          time: '刚刚',
        },
        ...p.activities,
      ];
      const next = { ...p, progressSnapshots, tasks, activities };
      return { ...next, metrics: recomputeMetrics(next) };
    });
  },
  subscribe(cb: () => void): () => void {
    listeners.add(cb);
    return () => listeners.delete(cb);
  },
};

// ── hooks ─────────────────────────────────────────────────
function subscribe(cb: () => void): () => void {
  return tenantProjectStore.subscribe(cb);
}

export function useTenantProjects(): TenantProject[] {
  return useSyncExternalStore(subscribe, () => ensure(), () => ensure());
}

export function useTenantProject(id: string | null): TenantProject | undefined {
  const list = useTenantProjects();
  return id ? list.find((p) => p.id === id) : undefined;
}
