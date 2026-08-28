/**
 * teamAssetsStore —— 员工端「团队门户」Store
 *
 * 定位：团队日常干活的统一入口。两类内容：
 *  1. Agent 广场——对标内部智能体控制台（agenthub）：按团队职能分类（研发/产品/运营/架构），
 *     卡片含 owner / 浏览 / 评论 / 调用量 / 复用 / 评分，点开卡片可在页面内直接对话预览（mock，不接真模型），
 *     并提供 A2A / API / MCP 接入信息供被别的 agent 调用；
 *  2. 嵌入工具——把内部已有的自研系统（成就墙/ 信息检索 / Token 效能等）以 iframe 方式嵌入，
 *     配置驱动，需要新增只加一条 EMBED_TABS 配置即可（见 TeamAssets.tsx）。
 *
 * Agent 广场为 mock，自包含（localStorage + useSyncExternalStore），不依赖后端。
 */
import { useSyncExternalStore } from 'react';

// ════════════════════════════════════════════════════════════
//  Agent 广场
// ════════════════════════════════════════════════════════════

/** Agent 广场职能分类（对齐内部 agenthub：研发/产品/运营/架构） */
export type AgentCategory = 'dev' | 'product' | 'ops' | 'arch';

export const AGENT_CATEGORY_META: Record<AgentCategory, { label: string; tint: string }> = {
  dev: { label: '研发', tint: '#1447E6' },
  product: { label: '产品', tint: '#EA580C' },
  ops: { label: '运营', tint: '#0891B2' },
  arch: { label: '架构', tint: '#9333EA' },
};

/** 接入信息（被别的 agent 调用） */
export interface AgentEndpoint {
  a2a: string; // A2A endpoint
  apiCurl: string; // API curl 片段
  mcp: string; // MCP 配置片段
}

/** 广场里的一个可复用 agent */
export interface MarketAgent {
  id: string;
  name: string;
  avatar: string; // emoji 头像
  category: AgentCategory;
  tagline: string; // 一句话简介
  description: string; // 详细说明
  owner: string; // 负责人
  team: string; // 归属团队/产品线
  tags: string[];
  model: string; // 运行模型（展示）
  // ── 使用数据 ──
  views: number; // 浏览量
  comments: number; // 评论数
  calls: number; // 累计调用量
  reuseCount: number; // 被其他团队/项目复用数
  rating: number; // 评分（0-5）
  trendPct: number; // 近 7 天调用环比 %
  featured: boolean; // 是否精选
  collected: boolean; // 我是否收藏
  // ── 页面内直接用（对标agenthub 点开即对话）──
  presetQuestions: string[]; // 左侧常用问题
  endpoint: AgentEndpoint; // 被集成接入信息
  url?: string; // 访问地址（knot/imate 等机器人的直链，用户手动上架时填）
}

/** 用户自定义的嵌入工具项（内置的三个在 TeamAssets.tsx 里代码配置，这里只存用户新增的） */
export interface CustomEmbed {
  id: string;
  label: string; // Tab 名称
  url: string; // 嵌入地址（https，需通过白名单校验）
  desc: string; // 一句话说明
  createdAt: number;
}

/** 门户全量可变状态 */
export interface AssetsState {
  agents: MarketAgent[];
  customEmbeds: CustomEmbed[];
}

// ════════════════════════════════════════════════════════════
//  种子数据（mock）
// ════════════════════════════════════════════════════════════

/** 「我」的归属人（用于身份高亮） */
export const ME_OWNER = 'alice';

function ep(id: string): AgentEndpoint {
  return {
    a2a: `https://agenthub.claw.woa.com/a2a/${id}`,
    apiCurl: `curl -X POST https://agenthub.claw.woa.com/api/v1/agents/${id}/invoke \\
  -H "Authorization: Bearer $CLAW_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"input": "你的问题"}'`,
    mcp: `{
  "mcpServers": {
    "${id}": {
      "url": "https://agenthub.claw.woa.com/mcp/${id}",
      "headers": { "Authorization": "Bearer \${CLAW_TOKEN}" }
    }
  }
}`,
  };
}

// ── Agent 广场 ───────────────────────────────────────────
const SEED_AGENTS: MarketAgent[] = [
  {
    id: 'ag-codebuddy', name: 'CodeReview 助手', avatar: '🔍', category: 'dev',
    tagline: '增量MR 自动评审，产出可执行改进建议',
    description: '对接 git.woa.com，MR 提交即触发评审：命名/边界/安全/复杂度多维打分，输出行级建议，支持团队自定义 rules。',
    owner: 'alice', team: 'ClawPro', tags: ['代码评审', 'MR', 'CI'], model: 'Claude-5-Sonnet',
    views: 18240, comments: 132, calls: 18240, reuseCount: 12, rating: 4.8, trendPct: 32, featured: true, collected: false,
    presetQuestions: ['帮我评审这个 MR 的安全性', '这段函数复杂度过高怎么拆？', '生成本次变更的评审摘要', '按团队 rules 检查命名规范'],
    endpoint: ep('ag-codebuddy'),
  },
  {
    id: 'ag-cvm-doctor', name: 'CVM 排障医生', avatar: '🩺', category: 'ops',
    tagline: '一句话描述现象，自动拉取 Events/监控定位根因',
    description: '聚合云监控、CVM Events、日志，按现象反查根因并给出处置步骤，沉淀过的排障 case 越用越准。',
    owner: 'carol', team: 'CVM控制台', tags: ['排障', '监控', '根因分析'], model: 'Claude-5-Sonnet',
    views: 15630, comments: 98, calls: 15630, reuseCount: 9, rating: 4.7, trendPct: 21, featured: true, collected: false,
    presetQuestions: ['实例 CPU 突然打满怎么排查？', '拉一下这台机器近 1 小时的 Events', '磁盘 IO 高的常见根因有哪些？', '给出重启前的检查清单'],
    endpoint: ep('ag-cvm-doctor'),
  },
  {
    id: 'ag-token-guard', name: 'Token 治理官', avatar: '🪙', category: 'ops',
    tagline: '团队 Token 用量透视 + 异常告警 + 优化建议',
    description: '按项目/人/模型拆解 Token 成本，识别高耗低效调用，给出压缩上下文、换模型、加缓存的具体建议。',
    owner: 'bob', team: 'ClawPro', tags: ['Token', '成本', '治理'], model: 'Claude-5-Sonnet',
    views: 12100, comments: 156, calls: 12100, reuseCount: 22, rating: 4.9, trendPct: 58, featured: true, collected: false,
    presetQuestions: ['本月哪个项目 Token 涨最多？', '有哪些高耗低效调用可优化？', '给我一份成本压缩建议清单', '按人拆解一下上周用量'],
    endpoint: ep('ag-token-guard'),
  },
  {
    id: 'ag-deploy-hand', name: '一键部署手', avatar: '🚀', category: 'ops',
    tagline: 'Lighthouse 应用编排 + 灰度发布助手',
    description: '把重复的部署编排标准化：环境检查、镜像构建、灰度切流、回滚预案一条龙，能自动化绝不手动。',
    owner: 'iris', team: 'Lighthouse', tags: ['部署', '灰度', '自动化'], model: 'Claude-5-Sonnet',
    views: 9820, comments: 64, calls: 9820, reuseCount: 15, rating: 4.6, trendPct: 44, featured: false, collected: false,
    presetQuestions: ['帮我编排一次灰度发布', '发布前环境检查项有哪些？', '回滚预案怎么配？', '生成本次部署的变更说明'],
    endpoint: ep('ag-deploy-hand'),
  },
  {
    id: 'ag-doc-writer', name: '文档匠', avatar: '✍️', category: 'product',
    tagline: '代码/需求一键生成规范文档与发布说明',
    description: '从 diff、需求单自动生成 API 文档、变更说明、发布 note，风格对齐团队文档规范，支持中英双语。',
    owner: 'alice', team: 'ClawPro', tags: ['文档', '发布说明', '规范'], model: 'Claude-5-Sonnet',
    views: 7290, comments: 71, calls: 7290, reuseCount: 11, rating: 4.6, trendPct: 24, featured: true, collected: false,
    presetQuestions: ['根据这个需求单写产品动态文案', '生成本次发布的 release note', '把这段代码写成 API 文档', '帮我润色这段功能说明'],
    endpoint: ep('ag-doc-writer'),
  },
  {
    id: 'ag-market-radar', name: '市场雷达', avatar: '📡', category: 'product',
    tagline: '行业与竞品动态每日聚合成简报',
    description: '抓取公开渠道的行业/竞品动态，按产品线归类去噪，每日产出可读简报，作为市场调研统一入口的供给源。',
    owner: 'jack', team: '商机', tags: ['市场', '竞品', '简报'], model: 'Claude-5-Sonnet',
    views: 6140, comments: 43, calls: 6140, reuseCount: 7, rating: 4.5, trendPct: 33, featured: false, collected: false,
    presetQuestions: ['今天的行业动态帮我总结', '竞品 A 最近有什么新动作？', '按产品线归类本周简报', '这条动态对我们有什么影响？'],
    endpoint: ep('ag-market-radar'),
  },
  {
    id: 'ag-slide-maker', name: '汇报速成', avatar: '📊', category: 'product',
    tagline: '数据 + 结论自动排版成汇报页',
    description: '输入指标与结论，自动生成结构化汇报页与讲稿要点，套用团队品牌样式，周报月报省一半时间。',
    owner: 'carol', team: 'ClawPro', tags: ['汇报', '周报', 'PPT'], model: 'Claude-5-Sonnet',
    views: 5510, comments: 38, calls: 5510, reuseCount: 6, rating: 4.2, trendPct: 11, featured: false, collected: false,
    presetQuestions: ['把这组数据做成周报页', '帮我写汇报的结论要点', '生成月度复盘大纲', '套用团队品牌样式'],
    endpoint: ep('ag-slide-maker'),
  },
  {
    id: 'ag-log-miner', name: '日志淘金者', avatar: '⛏️', category: 'ops',
    tagline: '海量日志聚类，异常模式自动归纳',
    description: '面向 CLS/自建日志，按 traceId 收敛上下文后聚类，产出异常 Top 模式与影响面，减少来回翻日志。',
    owner: 'frank', team: 'CVM控制台', tags: ['日志', '聚类', '可观测'], model: 'Claude-5-Sonnet',
    views: 7430, comments: 52, calls: 7430, reuseCount: 6, rating: 4.4, trendPct: 12, featured: false, collected: false,
    presetQuestions: ['这批日志的异常 Top 模式是什么？', '按 traceId 收敛一下上下文', '影响面有多大？', '归纳一下报错原因'],
    endpoint: ep('ag-log-miner'),
  },
  {
    id: 'ag-secret-scanner', name: '密钥哨兵', avatar: '🔐', category: 'arch',
    tagline: '提交前扫描明文密钥，命中即阻断',
    description: '在提交/评审阶段扫描 token/secret/password 明文，命中即阻断并提示改用环境变量。已拦下多次误提交。',
    owner: 'bob', team: 'ClawPro', tags: ['安全', '密钥', '合规'], model: 'Claude-5-Sonnet',
    views: 8760, comments: 87, calls: 8760, reuseCount: 19, rating: 4.8, trendPct: 18, featured: true, collected: false,
    presetQuestions: ['扫描这段代码有没有明文密钥', '密钥该怎么改成环境变量？', '接入提交前扫描的步骤', '误报了怎么加白名单？'],
    endpoint: ep('ag-secret-scanner'),
  },
  {
    id: 'ag-arch-review', name: '架构评审官', avatar: '🏛️', category: 'arch',
    tagline: '方案设计稿自动做架构 review 与风险提示',
    description: '按内部架构基线审阅设计稿：分层合理性、依赖方向、容量与容灾、扩展性，产出风险清单与改进建议。',
    owner: 'carol', team: 'ClawPro', tags: ['架构', '评审', '基线'], model: 'Claude-5-Sonnet',
    views: 6480, comments: 61, calls: 6480, reuseCount: 8, rating: 4.7, trendPct: 27, featured: true, collected: false,
    presetQuestions: ['帮我 review 这份架构设计', '这个依赖方向合理吗？', '容灾方案有什么风险？', '扩展性上有哪些隐患？'],
    endpoint: ep('ag-arch-review'),
  },
  {
    id: 'ag-gpu-scheduler', name: 'GPU 算力管家', avatar: '🧮', category: 'arch',
    tagline: 'HAI 算力碎片整理与调度建议',
    description: '透视 GPU 利用率与碎片，给出合并/迁移/降配建议，闲置算力自动提醒回收。',
    owner: 'iris', team: 'HAI', tags: ['GPU', '调度', '算力'], model: 'Claude-5-Sonnet',
    views: 4680, comments: 29, calls: 4680, reuseCount: 5, rating: 4.3, trendPct: 15, featured: false, collected: false,
    presetQuestions: ['当前 GPU 碎片有多少？', '哪些算力可以合并？', '有闲置资源要回收吗？', '给出调度优化建议'],
    endpoint: ep('ag-gpu-scheduler'),
  },
  {
    id: 'ag-image-optimizer', name: '镜像瘦身师', avatar: '📦', category: 'dev',
    tagline: 'Dockerfile 分层优化，构建缓存命中翻倍',
    description: '分析构建耗时与层缓存，重排依赖安装顺序、合并层、剔除冗余，实践中缓存命中率从 ~40% 提到 90%。',
    owner: 'frank', team: 'image', tags: ['镜像', 'CI', '优化'], model: 'Claude-5-Sonnet',
    views: 5210, comments: 34, calls: 5210, reuseCount: 8, rating: 4.5, trendPct: 9, featured: false, collected: false,
    presetQuestions: ['帮我优化这个 Dockerfile', '为什么缓存总不命中？', '哪些层可以合并？', '构建为什么这么慢？'],
    endpoint: ep('ag-image-optimizer'),
  },
];

// ════════════════════════════════════════════════════════════
//  持久化 + 订阅
// ════════════════════════════════════════════════════════════

const CACHE_KEY = 'team_assets_cache';
const CACHE_VERSION_KEY = 'team_assets_cache_version';
const CACHE_VERSION = '3';

export const TEAM_ASSETS_EVENT = 'team-assets-store-updated';

function seedState(): AssetsState {
  return {
    agents: SEED_AGENTS.map((a) => ({ ...a })),
    customEmbeds: [],
  };
}

let state: AssetsState | null = null;
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

function load(): AssetsState {
  ensureVersion();
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<AssetsState>;
      if (parsed && Array.isArray(parsed.agents)) {
        return { ...seedState(), ...parsed } as AssetsState;
      }
    }
  } catch {
    /* ignore */
  }
  const seed = seedState();
  persist(seed);
  return seed;
}

function ensure(): AssetsState {
  if (state === null) state = load();
  return state;
}

function persist(next: AssetsState) {
  try {
    localStorage.setItem(CACHE_KEY, JSON.stringify(next));
    localStorage.setItem(CACHE_VERSION_KEY, CACHE_VERSION);
  } catch {
    /* ignore */
  }
}

function commit(next: AssetsState) {
  state = next;
  persist(next);
  listeners.forEach((l) => l());
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent(TEAM_ASSETS_EVENT));
  }
}

// ════════════════════════════════════════════════════════════
//  公共 API
// ════════════════════════════════════════════════════════════

export const teamAssetsStore = {
  getState(): AssetsState {
    return ensure();
  },

  /** Agent 广场：收藏 / 取消 */
  toggleAgentCollect(id: string) {
    const s = ensure();
    commit({
      ...s,
      agents: s.agents.map((a) => (a.id === id ? { ...a, collected: !a.collected } : a)),
    });
  },

  /** Agent 广场：用户手动上架一个 agent（填基本信息，其余给默认值），返回新 agent id */
  addCustomAgent(input: {
    name: string;
    url?: string;
    category: AgentCategory;
    tagline: string;
    owner?: string;
    team?: string;
  }): string {
    const s = ensure();
    const id = `ag-custom-${Date.now()}`;
    const owner = input.owner?.trim() || ME_OWNER;
    const agent: MarketAgent = {
      id,
      name: input.name.trim(),
      avatar: '🤖',
      category: input.category,
      tagline: input.tagline.trim(),
      description: input.tagline.trim(),
      owner,
      team: input.team?.trim() || '自定义',
      tags: [],
      model: '—',
      views: 0,
      comments: 0,
      calls: 0,
      reuseCount: 0,
      rating: 0,
      trendPct: 0,
      featured: false,
      collected: false,
      presetQuestions: [],
      endpoint: {
        a2a: input.url || '',
        apiCurl: input.url ? `curl -X POST ${input.url}` : '',
        mcp: input.url
          ? `{\n  "mcpServers": {\n    "${id}": {\n      "url": "${input.url}"\n    }\n  }\n}`
          : '',
      },
      url: input.url?.trim() || undefined,
    };
    commit({ ...s, agents: [agent, ...s.agents] });
    return id;
  },

  /** 嵌入工具：新增一个用户自定义嵌入项，返回新项 id */
  addCustomEmbed(input: { label: string; url: string; desc?: string }): string {
    const s = ensure();
    const id = `embed-custom-${Date.now()}`;
    const item: CustomEmbed = {
      id,
      label: input.label.trim(),
      url: input.url.trim(),
      desc: (input.desc ?? '').trim(),
      createdAt: Date.now(),
    };
    commit({ ...s, customEmbeds: [...s.customEmbeds, item] });
    return id;
  },

  /** 嵌入工具：删除一个用户自定义嵌入项 */
  removeCustomEmbed(id: string) {
    const s = ensure();
    commit({ ...s, customEmbeds: s.customEmbeds.filter((e) => e.id !== id) });
  },

  subscribe(cb: () => void): () => void {
    listeners.add(cb);
    return () => listeners.delete(cb);
  },
};

// ════════════════════════════════════════════════════════════
//  选择器 / 排行
// ════════════════════════════════════════════════════════════

export interface AgentRankItem {
  agent: MarketAgent;
  value: number;
  unit: string;
}

/** 热门 Agent（按调用量） */
export function getTopByCalls(agents: MarketAgent[]): AgentRankItem[] {
  return [...agents]
    .sort((a, b) => b.calls - a.calls)
    .slice(0, 6)
    .map((a) => ({ agent: a, value: a.calls, unit: '次调用' }));
}

/** 复用之星（按被复用数） */
export function getTopByReuse(agents: MarketAgent[]): AgentRankItem[] {
  return [...agents]
    .sort((a, b) => b.reuseCount - a.reuseCount)
    .slice(0, 6)
    .map((a) => ({ agent: a, value: a.reuseCount, unit: '次复用' }));
}

/** 好评榜（按评分） */
export function getTopByRating(agents: MarketAgent[]): AgentRankItem[] {
  return [...agents]
    .sort((a, b) => b.rating - a.rating)
    .slice(0, 6)
    .map((a) => ({ agent: a, value: a.rating, unit: '分' }));
}

// ════════════════════════════════════════════════════════════
//  Hook
// ════════════════════════════════════════════════════════════

function subscribe(cb: () => void): () => void {
  return teamAssetsStore.subscribe(cb);
}

export function useAssetsState(): AssetsState {
  return useSyncExternalStore(subscribe, () => ensure(), () => ensure());
}
