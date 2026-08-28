/**
 * 组织列公共组件 - 复用于 Agent 列表（OpenClawMonitor）与记忆空间列表（InstanceTable）
 *
 * 提供：
 *   - 工具函数：根据 creator (userId) 解析所属组织
 *   - GroupColumnFilter：列头筛选 Popover 的内容（不含外壳，调用方用 <PopoverContent> 包裹）
 *   - GroupCell：单元格渲染（部门/自定义组织徽标 + 路径 + Tooltip）
 *   - filterByCreatorGroup：组织筛选谓词
 *
 * 数据源：MemberManagement/mock 的 MOCK_GROUPS / MOCK_MANUAL_GROUPS / MOCK_USERS / MOCK_USERS_MANUAL
 * 组织层级：通过 buildGroupTree 构造，选中某组织时其所有子孙组织下的用户都会被命中
 */
import React, { useMemo, useState } from 'react';
import { Search, Check, ChevronDown, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import {
  MOCK_GROUPS,
  MOCK_MANUAL_GROUPS,
  MOCK_USERS,
  MOCK_USERS_MANUAL,
} from '../../MemberManagement/mock';
import type { UserGroup, GroupSource } from '../../MemberManagement/types';
import { buildGroupTree, type GroupTreeNode } from '../../MemberManagement/health';

// ─── 工具函数 ────────────────────────────────────────────────────────────

/** 获取组织的完整路径（如 "产品组" 或 "研发组 / 前端"） */
export function getGroupPath(groupId: string, groups: UserGroup[]): string {
  const map = new Map(groups.map((g) => [g.id, g]));
  const chain: string[] = [];
  let cur = map.get(groupId);
  while (cur) {
    chain.unshift(cur.name);
    cur = cur.parentId ? map.get(cur.parentId) : undefined;
  }
  return chain.join(' / ');
}

/** 按 source 分桶标题 */
const GROUP_SOURCE_LABELS: Record<GroupSource, string> = {
  'oneid-dept': '部门',
  'oneid-group': '自定义组织',
  manual: '自定义组织',
  project: '项目',
};

/** 获取某 creator 对应的组织信息（OneID 模式，只返回一个） */
export function getCreatorGroupItemOneid(
  creator: string,
): { id: string; path: string; kind: 'oneid-dept' | 'oneid-group' } | null {
  const user = MOCK_USERS.find((u) => u.userId === creator);
  if (!user) return null;
  // 优先取 oneid-group（自定义组织），其次取 oneid-dept（部门）
  let deptItem: { id: string; path: string; kind: 'oneid-dept' | 'oneid-group' } | null = null;
  for (const gid of user.groupIds) {
    const g = MOCK_GROUPS.find((g) => g.id === gid);
    if (!g) continue;
    if (g.source === 'oneid-group') {
      return { id: gid, path: getGroupPath(gid, MOCK_GROUPS), kind: 'oneid-group' };
    }
    if (g.source === 'oneid-dept' && !deptItem) {
      deptItem = { id: gid, path: getGroupPath(gid, MOCK_GROUPS), kind: 'oneid-dept' };
    }
  }
  return deptItem;
}

/** 获取某 creator 对应的组织信息（普通模式，只返回一个） */
export function getCreatorGroupItemManual(
  creator: string,
): { id: string; path: string } | null {
  const user = MOCK_USERS_MANUAL.find((u) => u.userId === creator);
  if (!user || user.groupIds.length === 0) return null;
  const gid = user.groupIds[0];
  return { id: gid, path: getGroupPath(gid, MOCK_MANUAL_GROUPS) };
}

/** 获取某 creator 所属的所有组织 id（含子孙逻辑：选中某组织时，其用户应该被命中） */
export function getCreatorAllGroupIds(creator: string, hasOneid: boolean): string[] {
  if (hasOneid) {
    const user = MOCK_USERS.find((u) => u.userId === creator);
    return user ? user.groupIds : [];
  }
  const user = MOCK_USERS_MANUAL.find((u) => u.userId === creator);
  return user ? user.groupIds : [];
}

/** 获取节点及其所有子孙 ID */
export function getGroupDescendantIds(node: GroupTreeNode): string[] {
  const ids: string[] = [node.id];
  node.children.forEach((c) => ids.push(...getGroupDescendantIds(c)));
  return ids;
}

/** 组织筛选谓词：creator 是否属于 groupFilter 选中组织（含子孙）。groupFilter 为空表示不筛选 */
export function matchesGroupFilter(
  creator: string | undefined,
  groupFilter: string,
  hasOneid: boolean,
): boolean {
  if (!groupFilter) return true;
  if (!creator) return false;
  const groups = hasOneid ? MOCK_GROUPS : MOCK_MANUAL_GROUPS;
  const trees = buildGroupTree(groups);
  const findNode = (nodes: GroupTreeNode[], id: string): GroupTreeNode | null => {
    for (const n of nodes) {
      if (n.id === id) return n;
      const f = findNode(n.children, id);
      if (f) return f;
    }
    return null;
  };
  const targetNode = findNode(trees, groupFilter);
  if (!targetNode) return true;
  const allowedGroupIds = new Set(getGroupDescendantIds(targetNode));
  const creatorGroupIds = getCreatorAllGroupIds(creator, hasOneid);
  return creatorGroupIds.some((gid) => allowedGroupIds.has(gid));
}

/** 取当前模式下的组织数据源（供调用方传入 GroupColumnFilter） */
export function getGroupsForMode(hasOneid: boolean): UserGroup[] {
  return hasOneid ? MOCK_GROUPS : MOCK_MANUAL_GROUPS;
}

// ─── 组织筛选树节点（递归） ────────────────────────────────────────────
function GroupTreeNodeItem({
  node,
  level,
  selected,
  expanded,
  onToggle,
  onSelect,
}: {
  node: GroupTreeNode;
  level: number;
  selected: string;
  expanded: Set<string>;
  onToggle: (id: string) => void;
  onSelect: (id: string) => void;
}) {
  const hasChildren = node.children && node.children.length > 0;
  const isExpanded = expanded.has(node.id);
  const isSelected = selected === node.id;

  return (
    <div>
      <div
        className={`flex items-center gap-1 py-1.5 px-2 rounded-md cursor-pointer transition-colors ${
          isSelected ? 'bg-blue-50 text-blue-600' : 'text-gray-700 hover:bg-gray-100'
        }`}
        style={{ paddingLeft: `${level * 16 + 8}px` }}
        onClick={() => onSelect(node.id)}
      >
        {hasChildren ? (
          <button
            className="w-4 h-4 flex items-center justify-center flex-shrink-0"
            onClick={(e) => {
              e.stopPropagation();
              onToggle(node.id);
            }}
          >
            {isExpanded ? (
              <ChevronDown className="w-3.5 h-3.5 text-gray-400" />
            ) : (
              <ChevronRight className="w-3.5 h-3.5 text-gray-400" />
            )}
          </button>
        ) : (
          <span className="w-4 h-4 flex items-center justify-center flex-shrink-0">
            <span className="w-1.5 h-1.5 rounded-full bg-gray-300" />
          </span>
        )}
        <span className={`text-sm truncate flex-1 ${isSelected ? 'text-blue-600 font-medium' : ''}`}>
          {node.name}
        </span>
        {isSelected && <Check className="w-4 h-4 ml-auto text-blue-600 flex-shrink-0" />}
      </div>
      {hasChildren &&
        isExpanded &&
        node.children.map((child) => (
          <GroupTreeNodeItem
            key={child.id}
            node={child}
            level={level + 1}
            selected={selected}
            expanded={expanded}
            onToggle={onToggle}
            onSelect={onSelect}
          />
        ))}
    </div>
  );
}

// ─── 组织列头筛选面板 ────────────────────────────────────────────────
/**
 * 列头筛选 Popover 的内容（不含外壳）。调用方需用 <PopoverContent> 或等价容器包裹。
 *
 * 行为：内部维护 tempValue / 搜索 / 展开态，仅在用户点击「确认」时才回调 onConfirm；点击「取消」回调 onCancel。
 */
export function GroupColumnFilter({
  groups,
  value,
  hasOneid,
  onConfirm,
  onCancel,
}: {
  groups: UserGroup[];
  value: string;
  hasOneid: boolean;
  onConfirm: (v: string) => void;
  onCancel: () => void;
}) {
  const [tempValue, setTempValue] = useState(value);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [searchQuery, setSearchQuery] = useState('');

  const toggleExpand = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const { activeSources, treesMap } = useMemo(() => {
    if (hasOneid) {
      const buckets: Record<string, UserGroup[]> = { 'oneid-dept': [], 'oneid-group': [] };
      groups.forEach((g) => {
        if (buckets[g.source]) buckets[g.source].push(g);
      });
      const order: GroupSource[] = ['oneid-dept', 'oneid-group'];
      const active = order.filter((s) => (buckets[s] || []).length > 0);
      const tMap: Record<string, GroupTreeNode[]> = {};
      active.forEach((s) => {
        tMap[s] = buildGroupTree(buckets[s]);
      });
      return { activeSources: active, treesMap: tMap };
    }
    const trees = buildGroupTree(groups);
    return { activeSources: ['manual' as GroupSource], treesMap: { manual: trees } };
  }, [groups, hasOneid]);

  const isNodeVisible = (node: GroupTreeNode): boolean => {
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    if (node.name.toLowerCase().includes(q)) return true;
    return node.children.some(isNodeVisible);
  };

  return (
    <>
      <div className="px-3 pt-3 pb-2">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="搜索组织"
            className="w-full h-8 pl-8 pr-3 text-sm border border-gray-200 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-300"
          />
        </div>
      </div>
      <div className="max-h-[280px] overflow-y-auto px-2 pb-2">
        <div
          className={`flex items-center gap-2 py-1.5 px-2 rounded-md cursor-pointer transition-colors ${
            tempValue === '' ? 'bg-blue-50' : 'hover:bg-gray-100'
          }`}
          onClick={() => setTempValue('')}
        >
          <span className={`text-sm flex-1 ${tempValue === '' ? 'text-blue-600 font-medium' : 'text-gray-700'}`}>
            全部组织
          </span>
          {tempValue === '' && <Check className="w-4 h-4 text-blue-600 flex-shrink-0" />}
        </div>
        {activeSources.map((source) => (
          <div key={source}>
            {hasOneid && (
              <div className="px-2 pt-3 pb-1">
                <span className="text-xs font-semibold text-gray-400 uppercase tracking-wider">
                  {GROUP_SOURCE_LABELS[source]}
                </span>
              </div>
            )}
            {(treesMap[source] || []).map((root) =>
              isNodeVisible(root) ? (
                <GroupTreeNodeItem
                  key={root.id}
                  node={root}
                  level={0}
                  selected={tempValue}
                  expanded={expanded}
                  onToggle={toggleExpand}
                  onSelect={setTempValue}
                />
              ) : null,
            )}
          </div>
        ))}
      </div>
      <div className="border-t border-gray-100 px-3 py-2 flex items-center justify-end gap-1.5">
        <Button variant="ghost" size="sm" className="text-xs text-gray-500 h-7 px-2" onClick={onCancel}>
          取消
        </Button>
        <Button
          size="sm"
          className="text-xs h-7 px-3"
          style={{ background: 'linear-gradient(135deg, #007AFF, #5856D6)', color: 'white' }}
          onClick={() => onConfirm(tempValue)}
        >
          确认
        </Button>
      </div>
    </>
  );
}

// ─── 单元格 ────────────────────────────────────────────────────────────
/**
 * 组织单元格：根据 creator 渲染部门/自定义组织徽标 + 路径，hover 显示完整路径 Tooltip。
 * 找不到组织时显示 "—"。
 */
export function GroupCell({ creator, hasOneid }: { creator: string | undefined; hasOneid: boolean }) {
  if (!creator) return <span className="text-sm text-gray-300">—</span>;
  if (hasOneid) {
    const item = getCreatorGroupItemOneid(creator);
    if (!item) return <span className="text-sm text-gray-300">—</span>;
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="flex items-center gap-1.5 max-w-[200px] cursor-default">
            <span
              className={`inline-flex items-center text-[10px] font-medium rounded px-1.5 py-0.5 shrink-0 ${
                item.kind === 'oneid-dept' ? 'text-blue-600 bg-blue-50' : 'text-purple-600 bg-purple-50'
              }`}
            >
              {item.kind === 'oneid-dept' ? '部门' : '自定义组织'}
            </span>
            <span className="text-sm text-gray-700 truncate max-w-[120px]">{item.path}</span>
          </div>
        </TooltipTrigger>
        <TooltipContent side="bottom" align="start">
          <span className="text-xs">{item.path}</span>
        </TooltipContent>
      </Tooltip>
    );
  }
  const item = getCreatorGroupItemManual(creator);
  if (!item) return <span className="text-sm text-gray-300">—</span>;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="text-sm text-gray-700 truncate max-w-[160px] block cursor-default">{item.path}</span>
      </TooltipTrigger>
      <TooltipContent side="bottom" align="start">
        <span className="text-xs">{item.path}</span>
      </TooltipContent>
    </Tooltip>
  );
}
