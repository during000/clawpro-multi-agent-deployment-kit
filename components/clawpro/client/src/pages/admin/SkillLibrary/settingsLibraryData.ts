/**
 * 企业设定库 — 类型、Mock 数据与规范校验
 *
 * 「企业设定库」管理的是 Agent 每次会话都会自动加载的「团队全局设定」文档
 * （业界常见的 CLAUDE.md 这类全局设定文件）。它承载团队身份、协作准则与全局
 * 质量底线，并指向规范库 / 知识库等细节资产。
 *
 * 定位：每份设定绑定到一个具体「项目」。同一项目通常只维护一份团队设定，
 * 项目在拉取时会读取这份设定作为 Agent 的基础上下文。下发时落成各工具各自的
 * 设定文件（CLAUDE.md / CODEBUDDY.md），仅为落盘细节，不影响页面形态。
 */

/** 设定文档下发/草稿状态 */
export type TeamSettingStatus = 'distributed' | 'draft';

/** 状态显示映射 */
export const TEAM_SETTING_STATUS_MAP: Record<
  TeamSettingStatus,
  { label: string; variant: 'green' | 'gray' }
> = {
  distributed: { label: '已下发', variant: 'green' },
  draft: { label: '草稿', variant: 'gray' },
};

// ────────────────────────────────────────────────────────────
// 项目：设定的归属主体（关键信息）
// ────────────────────────────────────────────────────────────

/** 一个可绑定设定的项目 */
export interface SettingProject {
  id: string;
  /** 项目名称 */
  name: string;
  /** 代码仓库标识 */
  repo: string;
}

/** Mock 项目列表 */
export const MOCK_PROJECTS: SettingProject[] = [
  { id: 'proj-clawpro', name: 'ClawPro', repo: 'clawpro' },
  { id: 'proj-cvm', name: 'CVM', repo: 'cvm' },
  { id: 'proj-image', name: '镜像', repo: 'image' },
  { id: 'proj-as', name: 'AS', repo: 'as' },
];

export function getProjectName(projectId: string): string {
  return MOCK_PROJECTS.find((p) => p.id === projectId)?.name || projectId;
}

// ────────────────────────────────────────────────────────────
// 下发目标：目前仅支持 Claude Code、CodeBuddy
// ────────────────────────────────────────────────────────────

/** 一个可下发的目标工具 */
export interface DistributeTarget {
  id: string;
  /** 工具名称 */
  name: string;
  /** 落盘的设定文件名 */
  file: string;
  /** 是否已支持下发 */
  supported: boolean;
}

/**
 * 下发目标工具清单。
 * 当前仅 Claude Code、CodeBuddy 支持下发；其余工具暂不支持（置灰展示）。
 */
export const DISTRIBUTE_TARGETS: DistributeTarget[] = [
  { id: 'claude-code', name: 'Claude Code', file: 'CLAUDE.md', supported: true },
  { id: 'codebuddy', name: 'CodeBuddy', file: 'CODEBUDDY.md', supported: true },
  { id: 'cursor', name: 'Cursor', file: '.cursorrules', supported: false },
  { id: 'cline', name: 'Cline', file: '.clinerules', supported: false },
  { id: 'windsurf', name: 'Windsurf', file: '.windsurfrules', supported: false },
];

/** 一份团队设定文档 */
export interface TeamSetting {
  id: string;
  /** 设定名称 */
  name: string;
  /** 一句话描述（用于列表与卡片） */
  description: string;
  /** 绑定的项目 ID（关键信息，一个项目通常只维护一份设定） */
  projectId: string;
  /** 适用范围：public=全部用户，private=按组织 */
  scope: 'public' | 'private';
  /** 当 scope=private 时关联的组织 ID 列表 */
  groupIds: string[];
  /** 版本号 */
  version: string;
  /** 状态 */
  status: TeamSettingStatus;
  /** 更新时间 */
  updatedAt: Date;
  /** 设定文档正文（含文档头部元信息） */
  content: string;
}

/**
 * 新建时预填的标准模板
 * - 文档头部元信息：project / version / updated_at
 * - 正文分区：团队身份 / 核心价值观与工作准则 / 协作与沟通风格 / 全局质量底线 / 引用入口
 */
export const TEAM_SETTING_TEMPLATE = `---
project: 项目名称
version: 1.0
updated_at: 2026-06-29
---

# 团队设定（Team Context）

## 我们是谁
请用一两句话说明本项目团队的定位与职责。

## 核心价值观与工作准则
- 可读性优先于聪明写法
- 小步提交，宁可多个小 PR
- 不确定先问，不默默猜

## 协作与沟通风格
- 结论先行，必要时再补充背景
- 对事不对人，评审聚焦代码本身

## 全局质量底线
- 关键路径必须有测试与监控
- 安全与数据合规高于交付速度

## 引用入口
- 具体编码规范见「企业规范库」
- 开工前先检索「企业知识库」
`;

// ────────────────────────────────────────────────────────────
// 规范校验：在编辑时给出「精炼 / 误写细则 / 误写步骤」等提示
// ────────────────────────────────────────────────────────────

export type SettingLintLevel = 'warning' | 'info';

export interface SettingLintIssue {
  level: SettingLintLevel;
  /** 提示标题 */
  title: string;
  /** 详细建议 */
  detail: string;
}

/** 建议的正文篇幅上限（去除头部元信息后的字符数，纯提示不阻断） */
export const RECOMMENDED_MAX_BODY_LENGTH = 1500;

/** 移除文档头部元信息（YAML frontmatter），返回正文 */
export function stripFrontmatter(content: string): string {
  return content.replace(/^---\s*\n[\s\S]*?\n---\s*\n?/, '');
}

/** 解析头部元信息为键值对（仅支持简单的 key: value 形式） */
export function parseFrontmatter(content: string): Record<string, string> {
  const match = content.match(/^---\s*\n([\s\S]*?)\n---/);
  if (!match) return {};
  const result: Record<string, string> = {};
  for (const line of match[1].split('\n')) {
    const idx = line.indexOf(':');
    if (idx === -1) continue;
    const key = line.slice(0, idx).trim();
    const value = line.slice(idx + 1).trim();
    if (key) result[key] = value;
  }
  return result;
}

/**
 * 将头部元信息字段与正文重新组合为完整文档
 * - 仅写出有值的字段，保持文档精炼
 */
export function composeContent(
  frontmatter: Record<string, string>,
  body: string,
): string {
  const entries = Object.entries(frontmatter).filter(([, v]) => v && v.trim());
  if (entries.length === 0) return body.replace(/^\n+/, '');
  const fmLines = entries.map(([k, v]) => `${k}: ${v}`).join('\n');
  return `---\n${fmLines}\n---\n\n${body.replace(/^\n+/, '')}`;
}

/**
 * 校验设定文档是否符合「全局、精炼、不写细则/步骤」原则
 * 返回提示列表（不阻断保存，仅引导）
 */
export function lintTeamSetting(content: string): SettingLintIssue[] {
  const issues: SettingLintIssue[] = [];
  const body = stripFrontmatter(content);
  const trimmed = body.trim();

  // 1. 篇幅过长 → 提醒精炼
  if (trimmed.length > RECOMMENDED_MAX_BODY_LENGTH) {
    issues.push({
      level: 'warning',
      title: '篇幅偏长，建议精炼',
      detail: `当前正文约 ${trimmed.length} 字，超过建议的 ${RECOMMENDED_MAX_BODY_LENGTH} 字。团队设定每次会话都会被加载并占用上下文，应只确立全局基准，不堆砌细则。`,
    });
  }

  // 2. 疑似「编码规则细则」→ 应归入规范库
  const ruleSignals = [
    /缩进/,
    /\b\d+\s*(空格|spaces?|tab)/i,
    /命名(规范|规则|约定)/,
    /(必须|禁止|不得|不允许).{0,12}(使用|采用|调用)/,
    /eslint|prettier|lint/i,
    /分号|引号|大括号|驼峰|下划线/,
    /行宽|最大行长|max-len/i,
  ];
  if (ruleSignals.some((re) => re.test(body))) {
    issues.push({
      level: 'info',
      title: '检测到疑似「编码规则细则」',
      detail: '具体的编码规则（如缩进、命名、禁用语法等硬约束）属于「企业规范库（Rules）」。团队设定只需在「引用入口」指向规范库即可。',
    });
  }

  // 3. 疑似「任务操作步骤」→ 应归入技能库
  const hasStepList = /^\s*(?:第?[一二三四五六七八九十]+步|步骤\s*\d|[1-9]\.\s|[1-9]、)/m.test(body);
  const stepSignals = [/操作步骤/, /执行(以下|如下)(命令|步骤)/, /先.{0,8}然后.{0,8}最后/, /怎么(做|操作|执行)/];
  if (hasStepList || stepSignals.some((re) => re.test(body))) {
    issues.push({
      level: 'info',
      title: '检测到疑似「任务操作步骤」',
      detail: '"怎么做某件事"的具体操作步骤属于「企业技能库（Skills）」，按需调用即可。团队设定应保持全局、抽象。',
    });
  }

  return issues;
}

// ────────────────────────────────────────────────────────────
// Mock 列表数据
// ────────────────────────────────────────────────────────────

export const MOCK_TEAM_SETTINGS: TeamSetting[] = [
  {
    id: 'setting-001',
    name: 'ClawPro · 团队设定',
    description: 'ClawPro 项目团队设定，强调 Agent 管控平台的稳定性与性能优先，覆盖核心链路质量准则。',
    projectId: 'proj-clawpro',
    scope: 'private',
    groupIds: ['grp-2'],
    version: '2.1',
    status: 'distributed',
    updatedAt: new Date('2026-06-20'),
    content: `---
project: ClawPro
version: 2.1
updated_at: 2026-06-20
---

# 团队设定（Team Context）

## 我们是谁
ClawPro 项目团队，负责企业级 Agent 管控平台的稳定与性能。

## 核心价值观与工作准则
- 可读性优先于聪明写法
- 小步提交，宁可多个小 PR
- 不确定先问，不默默猜

## 全局质量底线
- 关键路径必须有测试与监控
- 安全与数据合规高于交付速度

## 引用入口
- 具体编码规范见「企业规范库」
- 开工前先检索「企业知识库」
`,
  },
  {
    id: 'setting-002',
    name: 'CVM · 团队设定',
    description: 'CVM 项目团队设定，聚焦云服务器管理的一致性与可维护性，明确设计走查与协作风格。',
    projectId: 'proj-cvm',
    scope: 'private',
    groupIds: ['grp-1'],
    version: '1.3',
    status: 'distributed',
    updatedAt: new Date('2026-06-18'),
    content: `---
project: CVM
version: 1.3
updated_at: 2026-06-18
---

# 团队设定（Team Context）

## 我们是谁
CVM 项目团队，对云服务器实例管理系统体验负责。

## 核心价值观与工作准则
- 体验一致性优先，遵循统一设计语言
- 可访问性是基本要求，而非加分项

## 协作与沟通风格
- 改动附带截图或录屏，便于评审
- 与设计同学保持高频对齐

## 引用入口
- 组件与视觉规范见「企业规范库」
- 设计资产与历史结论见「企业知识库」
`,
  },
  {
    id: 'setting-003',
    name: '镜像 · 团队设定',
    description: '镜像服务团队设定，强调变更可回滚、操作可审计，覆盖故障响应与值班协作的基本约定。',
    projectId: 'proj-image',
    scope: 'private',
    groupIds: ['grp-4'],
    version: '1.0',
    status: 'distributed',
    updatedAt: new Date('2026-06-12'),
    content: `---
project: 镜像
version: 1.0
updated_at: 2026-06-12
---

# 团队设定（Team Context）

## 我们是谁
镜像服务项目团队，对镜像存储与分发链路的稳定性与可用性负责。

## 核心价值观与工作准则
- 一切变更可灰度、可回滚、可观测
- 操作留痕，关键动作必须可审计

## 协作与沟通风格
- 故障期间结论先行，先止血再复盘
- 值班交接信息完整、不留模糊地带

## 引用入口
- 变更与发布规范见「企业规范库」
- 历史故障与预案见「企业知识库」
`,
  },
  {
    id: 'setting-004',
    name: 'AS · 团队设定（草稿）',
    description: 'AS 项目团队设定初稿，定义弹性伸缩策略的可复现与数据脱敏等底线，待评审后下发。',
    projectId: 'proj-as',
    scope: 'private',
    groupIds: ['grp-3'],
    version: '0.2',
    status: 'draft',
    updatedAt: new Date('2026-06-27'),
    content: `---
project: AS
version: 0.2
updated_at: 2026-06-27
---

# 团队设定（Team Context）

## 我们是谁
AS 项目团队，用弹性伸缩策略支撑业务高可用。

## 核心价值观与工作准则
- 策略可复现：记录伸缩规则版本与参数
- 结论需有监控数据支撑

## 全局质量底线
- 训练数据必须脱敏，遵守数据合规
- 上线策略须有评估报告与回滚预案

## 引用入口
- 策略规范与口径见「企业规范库」
- 历史配置与口径定义见「企业知识库」
`,
  },
];

