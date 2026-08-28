/**
 * projectAssetStore - 「项目资产管理」组织级配置 Store
 *
 * 职责：
 *  1. 持久化每个组织的资产配置（组织级同步模式 + 6 大类已选资产引用）与版本更新记录
 *  2. 提供草稿保存（saveConfig）：每次保存版本 +1，并生成一条 manual_edit 更新记录
 *  3. 版本漂移检测（checkVersionDrift）：对比已绑定版本快照与工具库最新版本，供 UI 提醒
 *     （工具库更新版本不会自动纳入，始终由管理员在编辑态手动更新并保存）
 *  4. 工具库删除联动（onLibraryItemDeleted）：工具库删除某资产时，从所有引用它的组织配置中
 *     级联移除并自动记一条 lib_deleted_cascade 版本记录（对应「配置项在工具库已不存在」）。
 *
 * 说明：同步模式（仅初始配置 / 始终同步）是组织级整体开关，仅决定「项目资产如何应用到 Agent」，
 * 不影响工具库联动；工具库「新增/更新」资产都不会自动进入项目资产合集（需管理员手动添加/更新）。
 */
import { compareSemver, isValidSemver } from '../SkillLibrary/downloadUtils';
import { skillStore } from '../SkillLibrary/skillStore';
import { pluginStore } from '../SkillLibrary/pluginStore';
import { mcpStore } from '../SkillLibrary/mcpStore';
import { standardsStore } from '../SkillLibrary/standardsStore';
import { publicSkillStore } from '../SkillLibrary/publicSkillStore';
import { loadAdminModels } from '@/lib/modelConfigStore';
import {
  ASSET_CATEGORY_MAP,
  ASSET_CATEGORY_ORDER,
  ASSET_SYNC_MODE_MAP,
  createEmptyProjectAssetConfig,
  type AssetCategory,
  type AssetSyncMode,
  type AssetVersionDrift,
  type ProjectAssetCategoryConfig,
  type ProjectAssetChangeSection,
  type ProjectAssetConfig,
  type ProjectAssetTagKind,
  type ProjectAssetUpdateRecord,
} from './types';

const CONFIG_CACHE_KEY = 'project_assets_config_cache';
const RECORDS_CACHE_KEY = 'project_assets_records_cache';
const CACHE_VERSION_KEY = 'project_assets_cache_version';
const CACHE_VERSION = '4';

/** 预置演示数据挂载的组织：A公司（组织树根节点） */
const SEED_GROUP_ID = 'dept-root';

export const PROJECT_ASSET_STORE_EVENT = 'project-asset-store-updated';

type ConfigMap = Record<string, ProjectAssetConfig>;
type RecordsMap = Record<string, ProjectAssetUpdateRecord[]>;

let configCache: ConfigMap | null = null;
let recordsCache: RecordsMap | null = null;

// ── 持久化 ────────────────────────────────────────────
function ensureVersion() {
  try {
    if (localStorage.getItem(CACHE_VERSION_KEY) !== CACHE_VERSION) {
      localStorage.removeItem(CONFIG_CACHE_KEY);
      localStorage.removeItem(RECORDS_CACHE_KEY);
      localStorage.setItem(CACHE_VERSION_KEY, CACHE_VERSION);
    }
  } catch (e) {
    console.warn('[projectAssetStore] 版本校验失败:', e);
  }
}

function loadConfigs(): ConfigMap {
  ensureVersion();
  let map: ConfigMap = {};
  try {
    const raw = localStorage.getItem(CONFIG_CACHE_KEY);
    if (raw) map = JSON.parse(raw) as ConfigMap;
  } catch (e) {
    console.warn('[projectAssetStore] 加载配置失败:', e);
  }
  if (!map[SEED_GROUP_ID]) map[SEED_GROUP_ID] = buildSeedConfig();
  return map;
}

function loadRecords(): RecordsMap {
  ensureVersion();
  let map: RecordsMap = {};
  try {
    const raw = localStorage.getItem(RECORDS_CACHE_KEY);
    if (raw) map = JSON.parse(raw) as RecordsMap;
  } catch (e) {
    console.warn('[projectAssetStore] 加载记录失败:', e);
  }
  if (!map[SEED_GROUP_ID]) map[SEED_GROUP_ID] = buildSeedRecords();
  return map;
}

function ensureConfigs(): ConfigMap {
  if (configCache === null) configCache = loadConfigs();
  return configCache;
}

function ensureRecords(): RecordsMap {
  if (recordsCache === null) recordsCache = loadRecords();
  return recordsCache;
}

function emit() {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent(PROJECT_ASSET_STORE_EVENT));
  }
}

function persist() {
  try {
    localStorage.setItem(CONFIG_CACHE_KEY, JSON.stringify(ensureConfigs()));
    localStorage.setItem(RECORDS_CACHE_KEY, JSON.stringify(ensureRecords()));
    localStorage.setItem(CACHE_VERSION_KEY, CACHE_VERSION);
  } catch (e) {
    console.warn('[projectAssetStore] 写入失败:', e);
  }
}

function commit() {
  persist();
  emit();
}

// ── 工具库查询 ────────────────────────────────────────
/** 从对应工具库 store 查询资产最新版本；返回 undefined 表示工具库中已不存在该资产 */
export function getLibraryLatestVersion(category: AssetCategory, refId: string): string | undefined {
  switch (category) {
    case 'modelConfig':
      return loadAdminModels().find((model) => model.id === refId)?.version;
    case 'publicSkill':
      return publicSkillStore.getById(refId)?.version;
    case 'enterpriseSkill':
      return skillStore.getById(refId)?.version;
    case 'enterprisePlugin':
      return pluginStore.getById(refId)?.version;
    case 'enterpriseMcp':
      return mcpStore.getById(refId)?.version;
    case 'enterpriseStandard':
      return standardsStore.getById(refId)?.version;
    default:
      return undefined;
  }
}

/** 从对应工具库 store 查询资产名称（用于摘要/展示） */
export function getLibraryItemName(category: AssetCategory, refId: string): string {
  switch (category) {
    case 'modelConfig':
      return loadAdminModels().find((model) => model.id === refId)?.name || refId;
    case 'publicSkill':
      return publicSkillStore.getById(refId)?.nameZh || publicSkillStore.getById(refId)?.name || refId;
    case 'enterpriseSkill':
      return skillStore.getById(refId)?.name || refId;
    case 'enterprisePlugin':
      return pluginStore.getById(refId)?.name || refId;
    case 'enterpriseMcp':
      return mcpStore.getById(refId)?.displayName || mcpStore.getById(refId)?.name || refId;
    case 'enterpriseStandard':
      return standardsStore.getById(refId)?.name || refId;
    default:
      return refId;
  }
}

// ── 版本比较 ──────────────────────────────────────────
function isNewer(latest: string, bound: string): boolean {
  if (isValidSemver(latest) && isValidSemver(bound)) {
    return compareSemver(latest, bound) > 0;
  }
  return latest !== bound;
}

// ── 记录 ──────────────────────────────────────────────
function appendRecord(
  groupId: string,
  version: number,
  tagKind: ProjectAssetTagKind,
  sections: ProjectAssetChangeSection[],
  operator?: string,
) {
  const records = ensureRecords();
  const list = records[groupId] ? [...records[groupId]] : [];
  list.unshift({
    id: `par-${groupId}-${version}-${Date.now()}`,
    groupId,
    version,
    tagKind,
    sections,
    operator,
    createdAt: new Date().toISOString(),
  });
  records[groupId] = list;
}

// ── 文案（统一为：新增 / 删除 / 同步模式修改 三类操作段落）──────────
/**
 * 由前后配置差异生成手动编辑的操作段落。
 * 一次保存可能同时含多个操作（新增 + 删除 + 改同步模式），每个操作各占一段：
 *  - 新增 x 项资产（下方明细列出「企业技能：xxx」等）
 *  - 删除 x 项资产（下方明细列出「企业技能：xxx」等）
 *  - 同步模式修改为「xxxx」
 */
function buildManualEditSections(
  prev: ProjectAssetConfig,
  next: ProjectAssetConfig,
): ProjectAssetChangeSection[] {
  const sections: ProjectAssetChangeSection[] = [];
  const addedItems: string[] = [];
  const removedItems: string[] = [];

  for (const category of ASSET_CATEGORY_ORDER) {
    const label = ASSET_CATEGORY_MAP[category].label;
    const prevIds = new Set(prev.categories[category].items.map((i) => i.refId));
    const nextIds = new Set(next.categories[category].items.map((i) => i.refId));
    for (const id of nextIds) {
      if (!prevIds.has(id)) addedItems.push(`${label}：${getLibraryItemName(category, id)}`);
    }
    for (const id of prevIds) {
      if (!nextIds.has(id)) removedItems.push(`${label}：${getLibraryItemName(category, id)}`);
    }
  }

  if (addedItems.length > 0) {
    sections.push({ title: `新增 ${addedItems.length} 项资产`, items: addedItems });
  }
  if (removedItems.length > 0) {
    sections.push({ title: `删除 ${removedItems.length} 项资产`, items: removedItems });
  }
  if (prev.mode !== next.mode) {
    sections.push({ title: `同步模式修改为「${ASSET_SYNC_MODE_MAP[next.mode].label}」` });
  }
  if (sections.length === 0) {
    sections.push({ title: '保存配置（无实质变更）' });
  }
  return sections;
}

// ── 深拷贝 ────────────────────────────────────────────
function cloneConfig(config: ProjectAssetConfig): ProjectAssetConfig {
  return {
    ...config,
    categories: ASSET_CATEGORY_ORDER.reduce((acc, category) => {
      const cat = config.categories[category];
      acc[category] = { items: cat.items.map((i) => ({ ...i })) };
      return acc;
    }, {} as Record<AssetCategory, ProjectAssetCategoryConfig>),
  };
}

// ── A公司（dept-root）预置演示数据 ────────────────────
/**
 * 构造 A公司 的预置资产配置：反映历史 v1~v8 演进后的最终态。
 * 最终企业技能：知识库问答 / 代码审查工具 / SQL 查询优化助手 / K8s 故障排查助手 / API 文档生成器
 * 最终企业规范：前端 React 规范 / Agent 安全合规基线
 * 同步模式：v8 起改为「所有实例始终同步更新」。
 */
function buildSeedConfig(): ProjectAssetConfig {
  const base = createEmptyProjectAssetConfig(SEED_GROUP_ID);
  const skillRefIds = ['skill-0', 'skill-2', 'skill-5', 'skill-6', 'skill-8'];
  base.categories.enterpriseSkill = {
    items: skillRefIds.map((refId) => ({
      refId,
      versionAtBind: getLibraryLatestVersion('enterpriseSkill', refId) || '1.0.0',
      addedAt: '2026-04-20T10:00:00.000Z',
    })),
  };
  const standardRefIds = ['asset-rule-001', 'asset-rule-002'];
  base.categories.enterpriseStandard = {
    items: standardRefIds.map((refId) => ({
      refId,
      versionAtBind: getLibraryLatestVersion('enterpriseStandard', refId) || '1.0.0',
      addedAt: '2026-07-15T16:00:00.000Z',
    })),
  };
  // 项目资产的组织级同步模式：v8 起为「所有实例始终同步更新」
  base.mode = 'autoSync';
  base.version = 8;
  base.updatedAt = '2026-07-15T16:20:00.000Z';
  base.updatedBy = '平台管理员';
  return base;
}

/**
 * 构造 A公司 的预置更新记录（版本历史，newest → oldest）。
 * 文案统一为四类：
 *  (1) 新增 x 项资产（明细列出「企业技能：xxx」等）
 *  (2) 删除 x 项资产（明细列出「企业技能：xxx」等）
 *  (3) 同步模式修改为「xxxx」
 *  (4) 已添加的资产在 Agent 工具库有调整，自动同步到资产管理（明细含「版本更新 / 删除 / 应用范围调整」括号说明）
 * tag 只有两种：manual（手动编辑）/ auto（自动同步）。
 * v8 为「一次手动保存同时做多个操作（新增 + 删除 + 改同步模式）」的示例。
 */
function buildSeedRecords(): ProjectAssetUpdateRecord[] {
  const rec = (
    version: number,
    tagKind: ProjectAssetTagKind,
    sections: ProjectAssetChangeSection[],
    createdAt: string,
    operator?: string,
  ): ProjectAssetUpdateRecord => ({
    id: `seed-${SEED_GROUP_ID}-v${version}`,
    groupId: SEED_GROUP_ID,
    version,
    tagKind,
    sections,
    operator,
    createdAt,
  });

  // newest first
  return [
    rec(
      8,
      'manual',
      [
        {
          title: '新增 2 项资产',
          items: ['企业技能：API 文档生成器', '企业规范：Agent 安全合规基线'],
        },
        {
          title: '删除 1 项资产',
          items: ['企业技能：会议纪要生成器'],
        },
        {
          title: '同步模式修改为「所有实例始终同步更新」',
        },
      ],
      '2026-07-15T16:20:00.000Z',
      '平台管理员',
    ),
    rec(
      7,
      'manual',
      [
        {
          title: '新增 1 项资产',
          items: ['企业规范：前端 React 规范'],
        },
      ],
      '2026-07-08T09:30:00.000Z',
      '平台管理员',
    ),
    rec(
      6,
      'auto',
      [
        {
          title: '已添加的资产在 Agent 工具库有调整，自动同步到资产管理',
          items: ['企业技能：文档总结助手（应用范围调整，A公司 不再命中，已自动同步移除）'],
        },
      ],
      '2026-06-12T15:10:00.000Z',
      '系统自动同步',
    ),
    rec(
      5,
      'auto',
      [
        {
          title: '已添加的资产在 Agent 工具库有调整，自动同步到资产管理',
          items: ['企业技能：日志分析器（工具库已删除，已自动同步移除）'],
        },
      ],
      '2026-05-18T11:20:00.000Z',
      '系统自动同步',
    ),
    rec(
      4,
      'auto',
      [
        {
          title: '已添加的资产在 Agent 工具库有调整，自动同步到资产管理',
          items: ['企业技能：代码审查工具（版本更新 v2.0.0 → v2.1.0）'],
        },
      ],
      '2026-05-06T14:40:00.000Z',
      '系统自动同步',
    ),
    rec(
      3,
      'manual',
      [
        {
          title: '新增 3 项资产',
          items: ['企业技能：SQL 查询优化助手', '企业技能：K8s 故障排查助手', '企业技能：会议纪要生成器'],
        },
      ],
      '2026-04-20T10:00:00.000Z',
      '平台管理员',
    ),
    rec(
      2,
      'manual',
      [
        {
          title: '新增 2 项资产',
          items: ['企业技能：代码审查工具', '企业技能：日志分析器'],
        },
      ],
      '2026-02-10T09:40:00.000Z',
      '平台管理员',
    ),
    rec(
      1,
      'manual',
      [
        {
          title: '新增 2 项资产',
          items: ['企业技能：知识库问答', '企业技能：文档总结助手'],
        },
      ],
      '2026-01-05T10:20:00.000Z',
      '平台管理员',
    ),
  ];
}

// ── 公共 API ──────────────────────────────────────────
export const projectAssetStore = {
  eventName: PROJECT_ASSET_STORE_EVENT,

  /** 获取某组织配置；不存在时返回空初始配置（不写入） */
  getConfig(groupId: string): ProjectAssetConfig {
    const configs = ensureConfigs();
    const existing = configs[groupId];
    if (existing) return cloneConfig(existing);
    return createEmptyProjectAssetConfig(groupId);
  },

  /** 该组织是否已保存过配置 */
  hasConfig(groupId: string): boolean {
    return !!ensureConfigs()[groupId];
  },

  /**
   * 保存草稿配置：版本 +1，生成 manual_edit 更新记录。
   * @param mode 组织级整体同步模式
   * @param categories 编辑态草稿中的完整 6 大类已选资产
   */
  saveConfig(
    groupId: string,
    mode: AssetSyncMode,
    categories: Record<AssetCategory, ProjectAssetCategoryConfig>,
    operator?: string,
  ): ProjectAssetConfig {
    const configs = ensureConfigs();
    const prev = configs[groupId] ?? createEmptyProjectAssetConfig(groupId);
    const nextVersion = prev.version + 1;
    const next: ProjectAssetConfig = {
      groupId,
      mode,
      categories: ASSET_CATEGORY_ORDER.reduce((acc, category) => {
        const cat = categories[category];
        acc[category] = { items: cat.items.map((i) => ({ ...i })) };
        return acc;
      }, {} as Record<AssetCategory, ProjectAssetCategoryConfig>),
      version: nextVersion,
      updatedAt: new Date().toISOString(),
      updatedBy: operator,
    };
    const sections = buildManualEditSections(prev, next);
    configs[groupId] = next;
    appendRecord(groupId, nextVersion, 'manual', sections, operator);
    commit();
    return cloneConfig(next);
  },

  /** 获取某组织的更新记录（版本历史） */
  getUpdateRecords(groupId: string): ProjectAssetUpdateRecord[] {
    return [...(ensureRecords()[groupId] ?? [])];
  },

  /**
   * 版本漂移检测：对比每个已选资产的绑定版本与工具库最新版本。
   * 工具库更新版本不会自动纳入，漂移用于在编辑态提醒管理员手动更新并保存。
   */
  checkVersionDrift(groupId: string): AssetVersionDrift[] {
    const config = ensureConfigs()[groupId];
    if (!config) return [];
    const drifts: AssetVersionDrift[] = [];
    for (const category of ASSET_CATEGORY_ORDER) {
      const cat = config.categories[category];
      for (const item of cat.items) {
        const latest = getLibraryLatestVersion(category, item.refId);
        if (latest && isNewer(latest, item.versionAtBind)) {
          drifts.push({
            category,
            refId: item.refId,
            boundVersion: item.versionAtBind,
            latestVersion: latest,
          });
        }
      }
    }
    return drifts;
  },

  // ── 工具库联动入口 ──────────────────────────────────

  /**
   * 工具库删除了某资产。
   * 从所有引用它的组织配置中级联移除，并记一条 lib_deleted_cascade 版本记录
   * （对应「配置项在工具库已不存在」，与「始终同步」下发到 Agent 时不卸载删除项是两个层面）。
   */
  onLibraryItemDeleted(category: AssetCategory, refId: string) {
    const configs = ensureConfigs();
    let changed = false;
    for (const groupId of Object.keys(configs)) {
      const config = configs[groupId];
      const cat = config.categories[category];
      const idx = cat.items.findIndex((i) => i.refId === refId);
      if (idx < 0) continue;
      const name = getLibraryItemName(category, refId);
      cat.items = cat.items.filter((i) => i.refId !== refId);
      config.version += 1;
      config.updatedAt = new Date().toISOString();
      appendRecord(
        groupId,
        config.version,
        'auto',
        [
          {
            title: '已添加的资产在 Agent 工具库有调整，自动同步到资产管理',
            items: [`${ASSET_CATEGORY_MAP[category].label}：${name}（工具库已删除，已自动同步移除）`],
          },
        ],
      );
      changed = true;
    }
    if (changed) commit();
  },

  subscribe(listener: () => void): () => void {
    if (typeof window === 'undefined') return () => {};
    window.addEventListener(PROJECT_ASSET_STORE_EVENT, listener);
    return () => window.removeEventListener(PROJECT_ASSET_STORE_EVENT, listener);
  },
};
