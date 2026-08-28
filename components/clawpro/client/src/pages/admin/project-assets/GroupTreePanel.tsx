/**
 * GroupTreePanel - 「项目资产管理」左侧组织/项目树
 * 数据来自跨页共享的 groupStore（与「用户管理-组织视图」双向同步）。
 * 顶部以 Segment 切换「组织」/「项目」两棵树，避免管理员滚动到页面下方才能看到项目：
 *   - 「组织」：source 为 manual / oneid-dept / oneid-group 的组织树
 *   - 「项目」：source 为 project 的项目树（特殊组织，可增删改、加子项目、加用户）
 * 行样式与操作按钮对齐「用户管理-组织视图」：纯文字树 + 人数(N) + 添加子级 / 更多(编辑/删除)。
 */
import { useMemo, useState } from 'react';
import {
  ChevronDown,
  ChevronRight,
  MoreHorizontal,
  Pencil,
  Plus,
  RotateCw,
  Search,
  Trash2,
} from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { SegmentGroup, SegmentOption } from '@/components/ui/segment';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { MetaText } from '@/components/ui/Typography';
import type { UserGroup, UserOrg } from '../MemberManagement/types';
import { buildGroupTree, type GroupTreeNode } from '../MemberManagement/health';
import { getOrgMembersDeep } from './projectRelations';

type TreeTab = 'org' | 'project';

interface GroupTreePanelProps {
  groups: UserGroup[];
  users: UserOrg[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  /** 刷新当前树（重新拉取组织 / 项目数据） */
  onRefresh: () => void;
  /** 新建顶层组织 */
  onCreateOrg: () => void;
  /** 新建顶层项目 */
  onCreateProject: () => void;
  onCreateChild: (parentId: string) => void;
  onRename: (groupId: string) => void;
  onDelete: (groupId: string) => void;
}

export default function GroupTreePanel({
  groups,
  users,
  selectedId,
  onSelect,
  onRefresh,
  onCreateOrg,
  onCreateProject,
  onCreateChild,
  onRename,
  onDelete,
}: GroupTreePanelProps) {
  const [activeTab, setActiveTab] = useState<TreeTab>('org');
  const [query, setQuery] = useState('');
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set(groups.map((g) => g.id)));

  const orgGroups = useMemo(() => groups.filter((g) => g.source !== 'project'), [groups]);
  const projectGroups = useMemo(() => groups.filter((g) => g.source === 'project'), [groups]);

  const orgRoots = useMemo(() => buildGroupTree(orgGroups), [orgGroups]);
  const projectRoots = useMemo(() => buildGroupTree(projectGroups), [projectGroups]);

  const isProjectTab = activeTab === 'project';
  const term = isProjectTab ? '项目' : '组织';
  const currentGroups = isProjectTab ? projectGroups : orgGroups;
  const currentRoots = isProjectTab ? projectRoots : orgRoots;

  // 人数口径：
  //   · 组织：聚合本组织 + 所有下级组织成员（去重），故上级组织会含全部下属成员
  //   · 项目：单层无下级，取直接成员
  const memberCount = (groupId: string) =>
    isProjectTab
      ? users.filter((u) => u.groupIds.includes(groupId)).length
      : getOrgMembersDeep(groupId, groups, users).length;

  // 搜索仅在当前 Tab 内匹配
  const flatMatches = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return null;
    return currentGroups.filter((g) => g.name.toLowerCase().includes(q));
  }, [currentGroups, query]);

  const toggle = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  // 切换「组织 / 项目」Tab：联动右侧，自动选中该 Tab 下第一个节点并清空搜索，
  // 避免切到项目后右侧仍停留在组织（导致文案 / 弹窗 / 存量实例逻辑走错分支）。
  const handleTabChange = (tab: TreeTab) => {
    if (tab === activeTab) return;
    setActiveTab(tab);
    setQuery('');
    const nextGroups = tab === 'project' ? projectGroups : orgGroups;
    const stillValid = selectedId != null && nextGroups.some((g) => g.id === selectedId);
    if (!stillValid) {
      const roots = tab === 'project' ? projectRoots : orgRoots;
      if (roots[0]) onSelect(roots[0].id);
    }
  };

  const renderRow = (node: GroupTreeNode, depth: number, showToggle: boolean) => {
    const isActive = selectedId === node.id;
    const hasChildren = node.children.length > 0;
    const isExpanded = expanded.has(node.id);
    const editable = !node.readonly;
    const count = memberCount(node.id);
    const btnClass = `w-5 h-5 flex items-center justify-center rounded transition-colors ${
      isActive
        ? 'text-[#737373] hover:text-[#020617] hover:bg-white'
        : 'text-[#d4d4d4] hover:text-[#525252] hover:bg-white'
    }`;
    return (
      <div
        key={node.id}
        role="button"
        tabIndex={0}
        onClick={() => onSelect(node.id)}
        className={`group flex items-center gap-1.5 h-8 pr-2 text-sm rounded-[4px] cursor-pointer transition-colors ${
          isActive
            ? 'bg-[var(--bg-grey-hover)] text-[#09090b] font-medium'
            : 'text-[#09090b] hover:bg-[var(--bg-grey-hover)]'
        }`}
        style={{ paddingLeft: `${8 + depth * 16}px` }}
      >
        {showToggle && hasChildren ? (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              toggle(node.id);
            }}
            className="shrink-0 w-4 h-4 flex items-center justify-center text-[#71717a] hover:text-[#09090b] transition-colors"
          >
            {isExpanded ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
          </button>
        ) : (
          <span className="shrink-0 w-4" />
        )}

        <span className="truncate" title={node.name}>
          {node.name}
        </span>
        <span className={`text-[11px] tabular-nums shrink-0 ${isActive ? 'text-[#71717a]' : 'text-[#a1a1aa]'}`}>
          ({count})
        </span>

        <span className="flex-1" />

        {/* 操作按钮：添加子级（仅组织，项目为单层级不支持子项目） + 更多（编辑 / 删除） */}
        {editable && (
          <span className="flex items-center gap-0.5 shrink-0" onClick={(e) => e.stopPropagation()}>
            {!isProjectTab && (
            <Tooltip>
              <TooltipTrigger asChild>
                <button
                  type="button"
                  className={btnClass}
                  onClick={(e) => {
                    e.stopPropagation();
                    onCreateChild(node.id);
                  }}
                >
                  <Plus className="w-3 h-3" />
                </button>
              </TooltipTrigger>
              <TooltipContent side="top" className="text-xs">
                添加子{term}
              </TooltipContent>
            </Tooltip>
            )}

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button type="button" className={btnClass} onClick={(e) => e.stopPropagation()}>
                  <MoreHorizontal className="w-3 h-3" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => onRename(node.id)}>
                  <Pencil className="w-4 h-4" />
                  编辑{term}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => onDelete(node.id)}
                  disabled={hasChildren}
                  className="text-[var(--text-danger)]"
                >
                  <Trash2 className="w-4 h-4" />
                  删除{term}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </span>
        )}
      </div>
    );
  };

  const renderTree = (nodes: GroupTreeNode[], depth: number): React.ReactNode =>
    nodes.map((node) => (
      <div key={node.id}>
        {renderRow(node, depth, true)}
        {node.children.length > 0 && expanded.has(node.id) && renderTree(node.children, depth + 1)}
      </div>
    ));

  const findFlatNode = (id: string): GroupTreeNode | undefined =>
    currentRoots
      .flatMap(function collect(n): GroupTreeNode[] {
        return [n, ...n.children.flatMap(collect)];
      })
      .find((n) => n.id === id);

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* 顶部：组织/ 项目切换 + 搜索
        * 停服态豁免：切换「组织 / 项目」属于查看类导航（不产生变更），
        * 与其他视图切换同档，需保持 100% 不透明与正常交互。
        * SegmentOption 自身未设置 disabled，"停服前已禁用则延续禁用"
        * 约束通过组件 disabled 属性依然生效（此处无）。 */}
      <div className="px-3 py-3 border-b border-[var(--cp-border)] space-y-3">
        <SegmentGroup className="w-full" data-billing-exempt>
          <SegmentOption
            className="flex-1"
            active={activeTab === 'org'}
            onClick={() => handleTabChange('org')}
          >
            组织
          </SegmentOption>
          <SegmentOption
            className="flex-1"
            active={activeTab === 'project'}
            onClick={() => handleTabChange('project')}
          >
            项目
          </SegmentOption>
        </SegmentGroup>
        <div className="flex items-center gap-2">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="claw-outline"
                size="icon"
                onClick={isProjectTab ? onCreateProject : onCreateOrg}
                aria-label={`新建${term}`}
              >
                <Plus className="w-4 h-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="top" className="text-xs">
              新建{term}
            </TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="claw-outline"
                size="icon"
                onClick={onRefresh}
                aria-label={`刷新${term}`}
              >
                <RotateCw className="w-4 h-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="top" className="text-xs">
              刷新{term}列表
            </TooltipContent>
          </Tooltip>
          <div className="relative flex-1 min-w-0">
            <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[var(--text-weak)]" />
            <Input
              placeholder={`搜索${term}...`}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="pl-10"
            />
          </div>
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto pt-2 pb-3">
        {flatMatches ? (
          flatMatches.length === 0 ? (
            <div className="py-8 text-center">
              <MetaText tone="secondary">没有匹配的{term}</MetaText>
            </div>
          ) : (
            <div className="space-y-0.5 px-3 pt-1">
              {flatMatches.map((g) => {
                const node = findFlatNode(g.id);
                return node ? renderRow(node, 0, false) : null;
              })}
            </div>
          )
        ) : (
          <div className="space-y-0.5 px-3">
            {currentRoots.length > 0 ? (
              renderTree(currentRoots, 0)
            ) : (
              <div className="py-4 text-center">
                <MetaText tone="secondary">暂无{term}</MetaText>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
