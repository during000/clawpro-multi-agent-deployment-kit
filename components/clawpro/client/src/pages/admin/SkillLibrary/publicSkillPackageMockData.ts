/**
 * 公共技能包 Mock 数据
 *
 * 数据来源：SkillHub 公共 API (https://api.skillhub.cn/api/v1/skillsets)
 * 由 `scripts/fetch-skillhub-snapshot.mjs` 一次性拉取并落盘到
 * `publicSkillPackageDataSnapshot.ts`，本文件负责把 raw 数据加工成前端需要的形态。
 *
 * 设计说明：
 * 1. 公共技能包是「前端展示层概念」—— 不是后端实体，本质是多个公共技能（PublicSkill）的组合模板
 * 2. 不直接安装、不直接执行；只用于浏览、收藏，以及在角色设定中被展开为多个公共技能
 * 3. 分类与 SkillHub 官方对齐（金融/科技/设计/营销/法律/学术/教育/人力/电商），
 *    在「全部」之后补充「我的收藏」
 * 4. 详情页：
 *    - 顶部信息卡：标题 + 长描述（summary）
 *    - 技能模块（identify）：chip 列表（来自 skillSlugs）
 *    - 工作流：用 MDXRenderer 渲染 raw.content（已剥离 YAML frontmatter）
 */

import { SKILLHUB_PACKAGE_SNAPSHOT } from './publicSkillPackageDataSnapshot';

// ─── 类型定义 ─────────────────────────────────────────────────────────────────

/** 从 SkillHub 接口拉取的原始数据结构（只保留前端用得上的字段） */
export interface PublicSkillPackageRaw {
  id: number;
  slug: string;
  displayName: string;
  /** 长描述（来自 SkillHub 详情页 summary 字段） */
  summary: string;
  /** SkillHub 一级场景（如 'tech'） */
  scene: string;
  /** SkillHub 二级场景（如 'test-automation'） */
  subScene: string;
  /** 映射到我们前端的分类 id（与 PUBLIC_SKILL_PACKAGE_CATEGORIES 对齐） */
  category: string;
  /** 完整 markdown 文档，详情页用 MDXRenderer 渲染（含 frontmatter） */
  content: string;
  skillSlugs: string[];
  skillCount: number;
}

/** 公共技能包内的 Skill 引用：仅记录 slug 与展示名 */
export interface PackageSkillRef {
  /** 与 PublicSkill.slug 对齐（如 'tdd-guide'） */
  slug: string;
  /** 展示名（如 'Tdd Guide'，用于 chip 显示）—— 由 slug 自动转 Title Case */
  name: string;
}

/** 前端使用的公共技能包数据形态（在 raw 基础上加工，并包含 SkillRef[]） */
export interface PublicSkillPackage {
  id: string;
  /** 展示名（如「自动化测试」） */
  name: string;
  /** 卡片用短描述（取 summary 截断，保证两行高度统一） */
  description: string;
  /** 详情页用长描述（完整 summary） */
  descriptionLong: string;
  /** 分类 slug */
  category: string;
  /** SkillHub 二级场景（用于详情页副信息展示，可选） */
  subScene?: string;
  /** 该技能包包含的 Skill 列表（identify 内容） */
  skills: PackageSkillRef[];
  /** 详情页工作流的 markdown 源码（已剥离 YAML frontmatter） */
  workflowMarkdown: string;
}

// ─── 分类 ─────────────────────────────────────────────────────────────────────

/**
 * 分类列表
 * 顺序：全部 → 我的收藏 → SkillHub 官方分类
 */
export const PUBLIC_SKILL_PACKAGE_CATEGORIES = [
  { id: 'all',       name: '全部' },
  { id: 'favorites', name: '我的收藏' },
  { id: 'finance',   name: '金融' },
  { id: 'tech',      name: '科技' },
  { id: 'design',    name: '设计' },
  { id: 'marketing', name: '营销' },
  { id: 'legal',     name: '法律' },
  { id: 'academic',  name: '学术' },
  { id: 'education', name: '教育' },
  { id: 'hr',        name: '人力' },
  { id: 'ecommerce', name: '电商' },
] as const;

// ─── 工具函数 ─────────────────────────────────────────────────────────────────

/**
 * 把 SkillHub 的 skill slug 转成展示名
 * 如 'tdd-guide' → 'Tdd Guide'，'qa-test-plan-generator' → 'Qa Test Plan Generator'
 * 对常见缩写做特殊处理，提升观感
 */
const ACRONYM_MAP: Record<string, string> = {
  api: 'API',
  qa: 'QA',
  ui: 'UI',
  ux: 'UX',
  e2e: 'E2E',
  rest: 'REST',
  graphql: 'GraphQL',
  tdd: 'TDD',
  ddd: 'DDD',
  cicd: 'CI/CD',
  k8s: 'K8s',
  okr: 'OKR',
  seo: 'SEO',
  sql: 'SQL',
  jd: 'JD',
  pr: 'PR',
  mcp: 'MCP',
};

function titleCase(slug: string): string {
  return slug
    .split('-')
    .map((part) => {
      const lower = part.toLowerCase();
      if (ACRONYM_MAP[lower]) return ACRONYM_MAP[lower];
      // 数字开头或纯数字保持原样
      if (/^\d/.test(part)) return part;
      return lower.charAt(0).toUpperCase() + lower.slice(1);
    })
    .join(' ');
}

/**
 * 从 SkillHub 的 markdown 内容中剥离 YAML frontmatter
 *
 * frontmatter 格式：
 * ---
 * scene: "tech"
 * sub_scene: "test-automation"
 * skills:
 *   - "tdd-guide"
 * ---
 *
 * 剥离后保留正文（# 工作流标题 + 步骤）
 */
function stripFrontmatter(md: string): string {
  if (!md) return '';
  // 匹配开头的 --- ... --- 块
  const match = md.match(/^---\n[\s\S]*?\n---\n+/);
  if (match) {
    return md.slice(match[0].length);
  }
  return md;
}

/**
 * 卡片用的短描述：直接用 summary 全文
 * 由 CSS `line-clamp-2` + `minHeight: 2.5rem` 保证：
 * - 内容超过 2 行 → 自动 ellipsis 截断
 * - 内容不足 2 行 → 仍保留 2 行高度，与其他卡片对齐
 * 不要在数据层做"按句号截断"等加工，否则不同句长会导致卡片视觉高度不一致
 */
function truncateForCard(summary: string): string {
  return summary || '';
}

// ─── 数据加工 ─────────────────────────────────────────────────────────────────

/**
 * 把 raw 数据加工成前端使用的 PublicSkillPackage[]
 * - 自动从 skillSlugs 生成 PackageSkillRef[]（slug + 展示名）
 * - 自动从 content 剥离 frontmatter，得到纯工作流 markdown
 * - 短描述/长描述拆分
 */
export const PUBLIC_SKILL_PACKAGES: PublicSkillPackage[] = SKILLHUB_PACKAGE_SNAPSHOT.map(
  (raw) => ({
    id: `pkg-${raw.id}`,
    name: raw.displayName,
    description: truncateForCard(raw.summary),
    descriptionLong: raw.summary,
    category: raw.category,
    subScene: raw.subScene,
    skills: raw.skillSlugs.map((slug) => ({
      slug,
      name: titleCase(slug),
    })),
    workflowMarkdown: stripFrontmatter(raw.content),
  }),
);
