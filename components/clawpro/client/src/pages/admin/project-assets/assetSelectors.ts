/**
 * assetSelectors - 将 6 大类资产 Store 数据标准化，并提供「应用范围匹配」筛选。
 * 供「项目资产管理」页面：列出可加入某组织的资产、展示已选资产信息。
 */
import type { UserGroup } from '../MemberManagement/types';
import { getGroupPath } from './projectRelations';
import { skillStore } from '../SkillLibrary/skillStore';
import { pluginStore } from '../SkillLibrary/pluginStore';
import { mcpStore } from '../SkillLibrary/mcpStore';
import { standardsStore } from '../SkillLibrary/standardsStore';
import { publicSkillStore } from '../SkillLibrary/publicSkillStore';
import { loadAdminModels } from '@/lib/modelConfigStore';
import type { SkillScope } from '../SkillLibrary/types';
import type { AssetCategory } from './types';

/** 目标 Agent 客户端展示名（用于企业规范的来源标识） */
const TARGET_CLIENT_LABELS: Record<string, string> = {
  claude_code: 'Claude Code',
  codebuddy: 'CodeBuddy',
  codex: 'Codex',
  workbuddy: 'WorkBuddy',
};

/** 标准化后的资产库条目 */
export interface AssetLibItem {
  refId: string;
  name: string;
  version: string;
  /** 描述（可选，用于列表副标题） */
  description?: string;
  /** 应用范围；公共技能无此字段（市场全部可选） */
  scope?: SkillScope;
  groupIds?: string[];
  /** 来源标识（企业规范：目标 Agent 名，如 CodeBuddy/WorkBuddy） */
  sourceLabel?: string;
}

/** 获取某大类在工具库中的全部条目（标准化） */
export function getCategoryLibraryItems(category: AssetCategory): AssetLibItem[] {
  switch (category) {
    case 'modelConfig':
      return loadAdminModels()
        .filter((model) => model.visible)
        .map((model) => ({
          refId: model.id,
          name: model.name,
          version: model.version,
          description: model.isDefault ? '平台默认模型' : undefined,
          scope: model.visibilityScope === 'all' ? 'public' : 'groups',
          groupIds: model.visibilityGroupIds,
        }));
    case 'publicSkill':
      return publicSkillStore.getAll().map((s) => ({
        refId: s.id,
        name: s.nameZh || s.name,
        version: s.version,
        description: s.descriptionZh || s.description,
      }));
    case 'enterpriseSkill':
      return skillStore.getAll().map((s) => ({
        refId: s.id,
        name: s.name,
        version: s.version,
        description: s.description,
        scope: s.scope,
        groupIds: s.groupIds,
      }));
    case 'enterprisePlugin':
      return pluginStore.getAll().map((p) => ({
        refId: p.id,
        name: p.name,
        version: p.version,
        description: p.description,
        scope: p.scope,
        groupIds: p.groupIds,
      }));
    case 'enterpriseMcp':
      return mcpStore.getAll().map((m) => ({
        refId: m.name,
        name: m.displayName || m.name,
        version: m.version,
        description: m.description,
        scope: m.scope,
        groupIds: m.groupIds,
      }));
    case 'enterpriseStandard':
      return standardsStore.getAll().map((a) => ({
        refId: a.id,
        name: a.name,
        version: a.version,
        description: a.description,
        scope: a.scope,
        groupIds: a.groupIds,
        sourceLabel:
          a.targetClients.map((c) => TARGET_CLIENT_LABELS[c] || c).join(' / ') || undefined,
      }));
    default:
      return [];
  }
}

/** 计算某组织的所有祖先组织 id（用于「上级组织」范围匹配） */
export function getAncestorGroupIds(groupId: string, groups: UserGroup[]): string[] {
  const byId = new Map(groups.map((g) => [g.id, g]));
  const ancestors: string[] = [];
  let current = byId.get(groupId);
  const guard = new Set<string>();
  while (current && current.parentId && byId.has(current.parentId) && !guard.has(current.parentId)) {
    guard.add(current.parentId);
    ancestors.push(current.parentId);
    current = byId.get(current.parentId);
  }
  return ancestors;
}

/**
 * 计算当前资产分类允许继承的组织范围。
 * - 组织：可以使用本组织、任意上级组织以及全部用户范围的资产；
 * - 项目模型：项目是独立应用范围，只能使用当前项目以及全部用户范围的模型；
 * - 项目其他资产：沿用既有规则，可使用直接所属组织范围的资产。
 *
 * 项目不沿用 parentId 做范围继承，避免未来出现项目层级数据时误把父项目
 * 或组织节点当成当前项目的可用模型范围。
 */
export function getScopeAncestorIds(
  groupId: string,
  groups: UserGroup[],
  isProject: boolean,
  category: AssetCategory,
): string[] {
  const ancestorIds = getAncestorGroupIds(groupId, groups);
  if (!isProject) return ancestorIds;
  return category === 'modelConfig' ? [] : ancestorIds.slice(0, 1);
}

/** 判断某资产的应用范围是否覆盖指定组织（本组织 / 上级分组 / 全部用户） */
export function isItemInScope(item: AssetLibItem, groupId: string, ancestorIds: string[]): boolean {
  // 公共技能无 scope 字段，市场全部可选
  if (!item.scope) return true;
  // 全部用户
  if (item.scope === 'public') return true;
  const targetIds = new Set([groupId, ...ancestorIds]);
  return (item.groupIds || []).some((gid) => targetIds.has(gid));
}

/** 应用范围优先级层级：self=本节点 > ancestor=上级组织 > public=全部用户 */
export type ScopeLevel = 'self' | 'ancestor' | 'public';

/**
 * 计算某资产相对当前节点的应用范围层级 + 展示标签（单个，取最高优先级）。
 * - 公共技能 / scope=public → 全部用户
 * - groupIds 命中当前节点 → 本节点（显示当前节点全路径）
 * - groupIds 命中某上级组织 → 取由近到远第一个命中的祖先（显示其全路径）
 * - 其余兜底 → 全部用户
 */
export function getItemScopeTag(
  item: AssetLibItem,
  groupId: string,
  groups: UserGroup[],
  ancestorIds?: string[],
): { level: ScopeLevel; label: string } {
  if (!item.scope || item.scope === 'public') {
    return { level: 'public', label: '全部用户' };
  }
  const gids = item.groupIds || [];
  if (gids.includes(groupId)) {
    return { level: 'self', label: getGroupPath(groupId, groups) };
  }
  const ancestors = ancestorIds ?? getAncestorGroupIds(groupId, groups);
  const hit = ancestors.find((aid) => gids.includes(aid));
  if (hit) {
    return { level: 'ancestor', label: getGroupPath(hit, groups) };
  }
  return { level: 'public', label: '全部用户' };
}

const SCOPE_LEVEL_RANK: Record<ScopeLevel, number> = { self: 0, ancestor: 1, public: 2 };

/**
 * 获取可加入某节点的资产（按应用范围过滤 + 排序）。
 * - 组织：本组织 + 全部上级组织 + 全部用户
 * - 项目模型：当前项目 + 全部用户（项目范围与组织范围彼此独立）
 * - 项目其他资产：沿用当前项目 + 直接所属组织 + 全部用户
 * 排序：本节点 → 上级组织 → 全部用户
 */
export function getSelectableItems(
  category: AssetCategory,
  groupId: string,
  groups: UserGroup[],
  isProject = false,
): AssetLibItem[] {
  const ancestorIds = getScopeAncestorIds(groupId, groups, isProject, category);
  const items = getCategoryLibraryItems(category).filter((item) =>
    isItemInScope(item, groupId, ancestorIds),
  );
  return items
    .slice()
    .sort(
      (a, b) =>
        SCOPE_LEVEL_RANK[getItemScopeTag(a, groupId, groups, ancestorIds).level] -
        SCOPE_LEVEL_RANK[getItemScopeTag(b, groupId, groups, ancestorIds).level],
    );
}

/** 获取单条资产的展示信息（含是否仍存在于工具库） */
export function getAssetItemDisplay(
  category: AssetCategory,
  refId: string,
): { name: string; version: string; exists: boolean; sourceLabel?: string } {
  const item = getCategoryLibraryItems(category).find((i) => i.refId === refId);
  if (!item) return { name: refId, version: '-', exists: false };
  return { name: item.name, version: item.version, exists: true, sourceLabel: item.sourceLabel };
}

/**
 * 平铺标签的灰色副文本：统一显示版本号 vX。
 * （企业规范此前显示来源 Agent 名 CodeBuddy/WorkBuddy，现改为版本号。）
 * 工具库已删除的条目返回 undefined（由调用方另作已删除提示）。
 */
export function getAssetTagMeta(category: AssetCategory, refId: string): string | undefined {
  const display = getAssetItemDisplay(category, refId);
  if (!display.exists) return undefined;
  return getAssetVersionLabel(category, display.version);
}

/** 模型配置使用模型标识原文，其余资产统一展示语义版本号。 */
export function getAssetVersionLabel(category: AssetCategory, version: string): string {
  return category === 'modelConfig' ? version : `v${version}`;
}
