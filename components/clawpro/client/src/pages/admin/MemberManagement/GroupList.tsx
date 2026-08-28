/**
 * 左侧组织列表（多层级树 + 搜索）
 *
 * 视觉规范（流动蓝图）：
 *   - 行高 32~36，每一层左缩进 16px
 *   - 右侧：人数（text-xs gray-400）
 *   - 活跃行：borderLeft: 2px solid #355EF1 + bg-blue-50 text-[#355EF1]
 *   - 按来源分桶：组织架构 / 用户组 / 自建组织，段头用一个极简小标题
 *   - 底部固定「未分配组织」项
 */
import React, { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Building2,
  ChevronRight,
  ChevronDown,
  ChevronLeft,
  Search,
  UserX,
  RefreshCw,
  Loader2,
  Plus,
  MoreHorizontal,
  Pencil,
  Trash2,
  Filter,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreActionsDropdown } from "@/components/ui/more-actions-dropdown";
import { Button } from "@/components/ui/button";
import { Empty, EmptyHeader, EmptyDescription, EmptyMedia } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import type { UserGroup, UserOrg, AnomalousGroup } from "./types";
import {
  buildGroupTree,
  getUsersOfGroupDeep,
  type GroupTreeNode,
} from "./health";

/** 特殊 id：未分配组织 */
export const UNASSIGNED_GROUP_ID = "__unassigned__";

interface GroupListProps {
  groups: UserGroup[];
  users: UserOrg[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  /** OneID 模式下，组织架构是否已同步 */
  deptSynced?: boolean;
  /** 点击「同步组织架构作为组织」 */
  onSyncDepts?: () => void;
  /** 同步中 */
  isSyncingDepts?: boolean;
  /** 是否 OneID 模式（显示组织架构+自定义组织两个桶） */
  hasOneid?: boolean;
  /** 是否普通模式（显示新建组织按钮 + 行操作） */
  isManualMode?: boolean;
  /** 新建组织 */
  onCreateGroup?: () => void;
  /** 添加子组织 */
  onAddChildGroup?: (parentId: string) => void;
  /** 编辑组织 */
  onEditGroup?: (groupId: string) => void;
  /** 删除组织 */
  onDeleteGroup?: (groupId: string) => void;
  /** 收起左侧面板 */
  onCollapse?: () => void;
  /** 异常组织 id 集合（红点标记：包含自身 + 子组织 + 父组织冒泡） */
  anomalousGroupIds?: Set<string>;
  /** 直接异常组织 id 集合（自身 + 子组织，不含父组织冒泡；用于 Tooltip 区分文案） */
  directAnomalousGroupIds?: Set<string>;
  /** 刷新同步回调（触发重新同步以检测异常） */
  onRefreshSync?: () => void;
  /** 初始化未完成组织 id 集合（明黄色点标记：包含自身 + 父组织冒泡） */
  uninitializedGroupIds?: Set<string>;
  /** 直接初始化未完成组织 id 集合（自身，不含父组织冒泡；用于 Tooltip 区分文案） */
  directUninitializedGroupIds?: Set<string>;
  /** 网络配置待更新组织 id 集合（明黄色点标记：仅命中组织自身，不冒泡父子） */
  networkOutdatedGroupIds?: Set<string>;
  /** 异常组织详情 Map（groupId -> AnomalousGroup），用于动态 Tooltip 文案 */
  anomalousGroupDetails?: Map<string, AnomalousGroup>;
}

// ─── 单行节点 ────────────────────────────────────────────
interface RowProps {
  node: GroupTreeNode;
  users: UserOrg[];
  groups: UserGroup[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  expanded: Set<string>;
  onToggle: (id: string) => void;
  keyword: string;
  /** 是否显示人数（默认 true） */
  showCount?: boolean;
  /** 是否普通模式（显示行操作） */
  isManualMode?: boolean;
  /** OneID 合并树模式：单棵树同时包含 oneid-dept / oneid-group / manual。
   *  此时行级显示策略按 node.source 动态分流（dept 显示人数+只读，og/manual 不显示人数+可 CRUD）。 */
  mergedTreeMode?: boolean;
  onAddChildGroup?: (parentId: string) => void;
  onEditGroup?: (groupId: string) => void;
  onDeleteGroup?: (groupId: string) => void;
  /** 异常组织 id 集合 */
  anomalousGroupIds?: Set<string>;
  /** 直接异常组织 id 集合（自身+子组织，不含父组织冒泡） */
  directAnomalousGroupIds?: Set<string>;
  /** 初始化未完成组织 id 集合（明黄色点：包含自身+父组织冒泡） */
  uninitializedGroupIds?: Set<string>;
  /** 直接初始化未完成组织 id 集合（自身，不含父组织冒泡） */
  directUninitializedGroupIds?: Set<string>;
  /** 网络配置待更新组织 id 集合（明黄色点：仅命中组织自身，不冒泡父子） */
  networkOutdatedGroupIds?: Set<string>;
  /** 筛选函数：判断节点是否匹配当前筛选条件 */
  filterFn?: (node: GroupTreeNode) => boolean;
  /** 异常组织详情 Map（groupId -> AnomalousGroup），用于动态 Tooltip 文案 */
  anomalousGroupDetails?: Map<string, AnomalousGroup>;
}

function GroupRow(props: RowProps) {
  const {
    node,
    users,
    groups,
    selectedId,
    onSelect,
    expanded,
    onToggle,
    keyword,
    showCount = true,
    isManualMode,
    mergedTreeMode,
    onAddChildGroup,
    onEditGroup,
    onDeleteGroup,
    anomalousGroupIds,
    directAnomalousGroupIds,
    uninitializedGroupIds,
    directUninitializedGroupIds,
    networkOutdatedGroupIds,
    filterFn,
    anomalousGroupDetails,
  } = props;

  // 递归渲染子节点
  const childRows = node.children.map((c) => (
    <GroupRow key={c.id} {...props} node={c} />
  ));

  // 是否被关键字命中（自身或任意后代）
  const matchKeyword = (n: GroupTreeNode): boolean => {
    if (!keyword.trim()) return true;
    const kw = keyword.trim().toLowerCase();
    if (n.name.toLowerCase().includes(kw)) return true;
    return n.children.some(matchKeyword);
  };

  if (!matchKeyword(node)) return null;
  // 筛选条件过滤
  if (filterFn && !filterFn(node)) return null;

  const isActive = selectedId === node.id;
  const isExpanded = expanded.has(node.id);
  const hasChildren = node.children.length > 0;
  // 人数统计口径与右侧 NodeContentPanel 一致：聚合自身及所有子孙
  const count = getUsersOfGroupDeep(node.id, groups, users).length;

  // 行级显示策略：
  //   - 合并树模式（OneID）：按 node.source 动态分流
  //       * oneid-dept（部门）：显示人数 + 只读，不显示操作
  //       * oneid-group / manual：不显示人数 + 可 CRUD
  //       * 特例 A公司 根节点（dept-root）：仅显示"添加子组织"
  //   - 非合并树模式（普通自建）：保持原行为，按外部传入的 showCount / isManualMode + node.readonly
  const isDeptNode = node.source === "oneid-dept";
  const isCompanyRoot = node.id === "dept-root";
  // 项目节点：操作文案使用"项目"，其余使用"组织"
  const nodeTerm = node.source === "project" ? "项目" : "组织";
  const effectiveShowCount = mergedTreeMode
    ? isDeptNode
    : showCount;
  const showAddChild = mergedTreeMode
    ? !isDeptNode || isCompanyRoot
    : isManualMode && !node.readonly;
  const showMoreActions = mergedTreeMode
    ? !isDeptNode && !isCompanyRoot
    : isManualMode && !node.readonly;

  return (
    <>
      <div
        className={`group flex items-center gap-1.5 h-8 pr-3 text-sm cursor-pointer rounded-[4px] mx-3 mb-0.5 transition-colors ${
          isActive
            ? "bg-[var(--bg-grey-hover)] text-[#09090b] font-medium"
            : "text-[#09090b] hover:bg-[var(--bg-grey-hover)]"
        }`}
        style={{
          paddingLeft: 8 + node.depth * 16,
        }}
        onClick={() => onSelect(node.id)}
      >
        {/* 展开箭头 */}
        {hasChildren ? (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onToggle(node.id);
            }}
            className="w-4 h-4 flex items-center justify-center text-[#71717a] hover:text-[#09090b] shrink-0 transition-colors"
          >
            {isExpanded ? (
              <ChevronDown className="w-3.5 h-3.5" />
            ) : (
              <ChevronRight className="w-3.5 h-3.5" />
            )}
          </button>
        ) : (
          <span className="w-4 h-4 shrink-0" />
        )}

        <span className="truncate" title={node.name}>
          {node.name}
        </span>
        {effectiveShowCount && (
          <span className={`text-[11px] tabular-nums shrink-0 ${isActive ? "text-[#71717a]" : "text-[#a1a1aa]"}`}>
            ({count})
          </span>
        )}

        {/* 异常红色点标记
            方案D：父组织的冒泡点仅在收起状态下显示，展开后隐藏（子组织自己标记了） */}
        {anomalousGroupIds?.has(node.id) &&
          // 如果是冒泡节点（非直接异常），仅在收起时显示
          (directAnomalousGroupIds?.has(node.id) || !isExpanded) && (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="relative shrink-0 ml-1">
                <span className="block w-2 h-2 rounded-full bg-red-500" />
              </span>
            </TooltipTrigger>
            <TooltipContent side="right" className="text-xs max-w-[260px]">
              {directAnomalousGroupIds?.has(node.id)
                ? (() => {
                    const detail = anomalousGroupDetails?.get(node.id);
                    const hasConfig = detail ? detail.boundConfigs.length > 0 : true;
                    const hasAgent = detail ? detail.agentInstanceCount > 0 : false;
                    const reasons = [
                      hasConfig ? "配置未解绑" : null,
                      hasAgent ? "Agent 实例未删除" : null,
                    ].filter(Boolean).join("、");
                    return `该组织对应的部门已在腾讯统一身份管理平台被删除，但仍有${reasons}`;
                  })()
                : (() => {
                    const hasConfig = Array.from(anomalousGroupDetails?.values() ?? []).some((d) => d.boundConfigs.length > 0);
                    const hasAgent = Array.from(anomalousGroupDetails?.values() ?? []).some((d) => d.agentInstanceCount > 0);
                    const reasons = [
                      hasConfig ? "配置未解绑" : null,
                      hasAgent ? "Agent 实例未删除" : null,
                    ].filter(Boolean).join("、");
                    return `该部门下有组织已在腾讯统一身份管理平台被删除，但仍有${reasons}，展开查看`;
                  })()}
            </TooltipContent>
          </Tooltip>
        )}

        {/* 初始化未完成明黄色提醒点（不与异常点同时显示）
            方案D：父组织的冒泡标记仅在收起状态下显示，展开后隐藏（子组织自己标记了）
            互斥优化：若本节点同时命中"网络配置待更新"，让位给后者，避免同一组织出现两个提醒点。 */}
        {!anomalousGroupIds?.has(node.id) &&
          !networkOutdatedGroupIds?.has(node.id) &&
          uninitializedGroupIds?.has(node.id) &&
          // 如果是冒泡节点（非直接未初始化），仅在收起时显示
          (directUninitializedGroupIds?.has(node.id) || !isExpanded) && (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="relative shrink-0 ml-1">
                <span className="block w-2 h-2 rounded-full bg-[var(--alert-warning-icon)]" />
              </span>
            </TooltipTrigger>
            <TooltipContent side="right" className="text-xs max-w-[260px]">
              {directUninitializedGroupIds?.has(node.id)
                ? "该组织未完成初始化配置"
                : "该组织下有子组织未完成初始化配置，展开查看"}
            </TooltipContent>
          </Tooltip>
        )}

        {/* 网络配置待更新明黄色提醒点（VPC / 子网被云端删除）
            - 仅命中组织自身展示，不冒泡父组织、不下发子组织、不影响兄弟组织
            - 与异常红点互斥（异常红点优先级最高）
            - 与「初始化未完成提醒点」共同命中时，本提醒点优先（更具体可定位），
              初始化提醒点会通过 networkOutdatedGroupIds 让位条件主动隐藏。 */}
        {!anomalousGroupIds?.has(node.id) &&
          networkOutdatedGroupIds?.has(node.id) && (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="relative shrink-0 ml-1">
                <span className="block w-2 h-2 rounded-full bg-[var(--alert-warning-icon)]" />
              </span>
            </TooltipTrigger>
            <TooltipContent side="right" className="text-xs max-w-[260px]">
              该组织的网络配置待更新
            </TooltipContent>
          </Tooltip>
        )}

        <span className="flex-1" />

        {/* 操作按钮：
            - 普通可编辑节点：显示「添加子组织」+「更多操作（编辑/删除）」
            - A公司 根节点（dept-root）：仅显示「添加子组织」，不显示更多操作 */}
        {(showAddChild || showMoreActions) && (
          <span className="flex items-center gap-0.5 shrink-0">
            {/* 添加子组织 */}
            {showAddChild && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    className={`w-5 h-5 flex items-center justify-center rounded transition-colors ${isActive ? "text-[#737373] hover:text-[#020617] hover:bg-[var(--bg-grey-hover)]" : "text-[#d4d4d4] hover:text-[#525252] hover:bg-[var(--bg-grey-hover)]"}`}
                    onClick={(e) => {
                      e.stopPropagation();
                      onAddChildGroup?.(node.id);
                    }}
                  >
                    <Plus className="w-3 h-3" />
                  </button>
                </TooltipTrigger>
                <TooltipContent side="top" className="text-xs">
                  添加子{nodeTerm}
                </TooltipContent>
              </Tooltip>
            )}
            {/* 更多操作 */}
            {showMoreActions && (
              <MoreActionsDropdown
                trigger={
                  <button
                    type="button"
                    className={`w-5 h-5 flex items-center justify-center rounded transition-colors ${isActive ? "text-[#737373] hover:text-[#020617] hover:bg-[var(--bg-grey-hover)]" : "text-[#d4d4d4] hover:text-[#525252] hover:bg-[var(--bg-grey-hover)]"}`}
                    onClick={(e) => e.stopPropagation()}
                  >
                    <MoreHorizontal className="w-3 h-3" />
                  </button>
                }
                align="end"
                stopPropagation
                items={[
                  {
                    label: `编辑${nodeTerm}`,
                    icon: Pencil,
                    onClick: () => onEditGroup?.(node.id),
                  },
                  {
                    label: `删除${nodeTerm}`,
                    icon: Trash2,
                    onClick: () => onDeleteGroup?.(node.id),
                    variant: "destructive",
                    disabled: hasChildren,
                  },
                ]}
              />
            )}
          </span>
        )}
      </div>

      {hasChildren && isExpanded && childRows}
    </>
  );
}

// ─── 分桶标题 ────────────────────────────────────────────
function BucketHeader({
  icon,
  title,
  count,
}: {
  icon: React.ReactNode;
  title: string;
  count: number;
}) {
  return (
    <div className="flex items-center gap-1.5 px-4 pt-4 pb-1.5">
      <span className="text-[#A3A3A3]">{icon}</span>
      <span className="text-xs font-semibold text-[#A3A3A3] uppercase tracking-wider">
        {title}
      </span>
      <span className="text-xs text-[#A3A3A3] tabular-nums">· {count}</span>
    </div>
  );
}

/** 筛选类型 */
type FilterType = "all" | "uninitialized" | "anomalous";

/**
 * 计算默认展开的节点集合。
 *   - OneID 合并模式（deptSynced !== undefined）：按"同步后默认平铺"规则——
 *     展开 A公司(dept-root) + 所有用户组(oneid-group) / 自定义组织(manual) 节点，
 *     部门(oneid-dept) 子级保持收起。
 *   - 普通模式（deptSynced === undefined）：展开所有顶层节点（沿用旧行为）。
 */
function computeDefaultExpanded(
  groups: UserGroup[],
  deptSynced: boolean | undefined
): Set<string> {
  const s = new Set<string>();
  if (deptSynced !== undefined) {
    groups.forEach((g) => {
      if (
        g.id === "dept-root" ||
        g.source === "oneid-group" ||
        g.source === "manual"
      ) {
        s.add(g.id);
      }
    });
  } else {
    groups.forEach((g) => {
      if (g.parentId === null) s.add(g.id);
    });
  }
  return s;
}

// ─── 主组件 ─────────────────────────────────────────────
export default function GroupList({
  groups,
  users,
  selectedId,
  onSelect,
  deptSynced,
  onSyncDepts,
  isSyncingDepts,
  hasOneid,
  isManualMode,
  onCreateGroup,
  onAddChildGroup,
  onEditGroup,
  onDeleteGroup,
  anomalousGroupIds,
  directAnomalousGroupIds,
  onRefreshSync,
  uninitializedGroupIds,
  directUninitializedGroupIds,
  networkOutdatedGroupIds,
  anomalousGroupDetails,
}: GroupListProps) {
  const [keyword, setKeyword] = useState("");
  const [filter, setFilter] = useState<FilterType>("all");
  const [filterOpen, setFilterOpen] = useState(false);
  const [expanded, setExpanded] = useState<Set<string>>(() =>
    computeDefaultExpanded(groups, deptSynced)
  );
  // 「同步数据源信息」引导按钮：
  //   - syncRequested：用户点击后置 true，标记"本会话已发起过同步"
  //   - 按钮仅在 !syncRequested 或 isSyncingDepts 为 true 时显示（点击→原地旋转→同步结束后再消失）
  //   - toast 完成提示复用 GroupView 中 handleSyncDepts 里已有的 toast.success("已同步数据源")
  const [syncRequested, setSyncRequested] = useState(false);

  // 同步数据源完成（deptSynced 由 false → true）时，按"默认平铺"规则重算展开状态：
  // A公司 + 用户组/自定义组织展开，部门(同步来的)子级保持收起。
  const prevDeptSyncedRef = useRef(deptSynced);
  useEffect(() => {
    if (prevDeptSyncedRef.current === false && deptSynced === true) {
      setExpanded(computeDefaultExpanded(groups, deptSynced));
    }
    prevDeptSyncedRef.current = deptSynced;
  }, [deptSynced, groups]);

  const toggle = (id: string) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  // 按来源分桶
  // OneID 模式（deptSynced !== undefined）下：把三种来源（oneid-dept / oneid-group / manual）
  // 合并为一棵树构建，使原本顶层的 oneid-group / manual 节点若 parentId 指向 "dept-root"（A公司），
  // 也能正确挂到 A公司 下，而不是被误判为孤儿提升到顶层。
  // 普通模式（deptSynced === undefined）下：保持原分桶逻辑，避免影响纯本地自建场景。
  const buckets = useMemo(() => {
    const deptGroups = groups.filter((g) => g.source === "oneid-dept");
    const ogGroups = groups.filter((g) => g.source === "oneid-group");
    const manualGroups = groups.filter((g) => g.source === "manual");

    if (deptSynced !== undefined) {
      // OneID 模式：合并构树（关键修复 —— 让 og/manual 顶层节点能挂到 dept-root 下）
      const merged = buildGroupTree([...deptGroups, ...ogGroups, ...manualGroups]);

      // 对 A公司（dept-root）下的直接子节点排序：部门 → oneid-group → manual
      const sourceRank = (s: GroupTreeNode["source"]) =>
        s === "oneid-dept" ? 0 : s === "oneid-group" ? 1 : 2;
      const sortRecursive = (nodes: GroupTreeNode[]): GroupTreeNode[] => {
        nodes.sort((a, b) => sourceRank(a.source) - sourceRank(b.source));
        nodes.forEach((n) => sortRecursive(n.children));
        return nodes;
      };
      sortRecursive(merged);

      return {
        merged,
        // 兼容旧字段（仅供"全部为空"判断与筛选无结果占位用）
        dept: deptGroups,
        og: ogGroups,
        manual: manualGroups,
      } as const;
    }

    // 普通模式：保留旧分桶（仅 manual 需要构树渲染）
    return {
      merged: null as GroupTreeNode[] | null,
      dept: [] as UserGroup[],
      og: [] as UserGroup[],
      manual: buildGroupTree(manualGroups),
    } as const;
  }, [groups, deptSynced]);

  // 筛选匹配：节点自身或其任意后代匹配当前 filter
  const matchFilter = (n: GroupTreeNode): boolean => {
    if (filter === "all") return true;
    if (filter === "uninitialized") {
      // 自身或后代有初始化未完成
      if (directUninitializedGroupIds?.has(n.id)) return true;
      return n.children.some(matchFilter);
    }
    if (filter === "anomalous") {
      // 自身或后代异常
      if (directAnomalousGroupIds?.has(n.id)) return true;
      return n.children.some(matchFilter);
    }
    return true;
  };

  // 筛选切换时，自动展开所有包含匹配节点的祖先路径
  React.useEffect(() => {
    if (filter === "all") return;
    // 收集所有需要展开的节点：如果一个节点的后代中有匹配项，则该节点需要展开
    const needExpand = new Set<string>();
    const collectExpandIds = (nodes: GroupTreeNode[]): boolean => {
      let hasMatch = false;
      for (const n of nodes) {
        const childHasMatch = collectExpandIds(n.children);
        const selfMatch =
          filter === "uninitialized"
            ? directUninitializedGroupIds?.has(n.id)
            : directAnomalousGroupIds?.has(n.id);
        if (selfMatch || childHasMatch) {
          hasMatch = true;
          // 如果子节点有匹配，则当前节点需要展开
          if (childHasMatch) {
            needExpand.add(n.id);
          }
        }
      }
      return hasMatch;
    };
    const allTrees: GroupTreeNode[] = buckets.merged
      ? buckets.merged
      : (buckets.manual as GroupTreeNode[]);
    collectExpandIds(allTrees);
    if (needExpand.size > 0) {
      setExpanded((prev) => {
        const next = new Set(prev);
        needExpand.forEach((id) => next.add(id));
        return next;
      });
    }
  }, [filter]);

  // 计算未分配组织用户数：不属于当前已加载组织的用户
  const unassignedCount = useMemo(() => {
    const loadedGroupIds = new Set(groups.map((g) => g.id));
    if (loadedGroupIds.size === 0) return users.length; // 没有任何组织 → 全部都是未分配组织
    return users.filter(
      (u) => !u.groupIds.some((gid) => loadedGroupIds.has(gid))
    ).length;
  }, [users, groups]);

  const isUnassignedActive = selectedId === UNASSIGNED_GROUP_ID;

  return (
    <div className="flex flex-col h-full">
      {/* 第一行：标题"组织" + 新建按钮 */}
      <div className="flex items-center gap-2 px-4 pt-4 pb-2">
        <h3 className="text-lg font-semibold text-[var(--text-title)] m-0">组织</h3>
        <Button
          variant="ghost"
          size="sm"
          className="gap-1 px-2.5 h-7 font-medium"
          onClick={onCreateGroup}
        >
          <Plus className="w-3.5 h-3.5" />
          新建
        </Button>
      </div>

      {/* 第二行：搜索框 + 刷新按钮 */}
      <div className="px-3 pb-2">
        <div className="flex items-center gap-2">
          <div className="relative flex-1 min-w-0">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)]" />
            <Input
              type="text"
              placeholder="搜索组织..."
              className="h-8 pl-8 pr-3 text-xs"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          {/* 筛选按钮 */}
          <DropdownMenu open={filterOpen} onOpenChange={setFilterOpen}>
            <DropdownMenuTrigger asChild>
              <Button
                variant="claw-outline"
                size="icon-sm"
                className={filter !== "all" ? "bg-[var(--bg-grey-hover)] border-[#e5e5e5]" : ""}
                title="筛选"
              >
                <Filter className="w-3.5 h-3.5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="min-w-[140px]">
              <DropdownMenuItem
                className={`text-xs gap-2 ${filter === "all" ? "font-medium text-[#020617]" : ""}`}
                onClick={() => setFilter("all")}
              >
                全部
              </DropdownMenuItem>
              <DropdownMenuItem
                className={`text-xs gap-2 ${filter === "uninitialized" ? "font-medium text-[#020617]" : ""}`}
                onClick={() => setFilter("uninitialized")}
              >
                <span className="w-2 h-2 rounded-full bg-[var(--alert-warning-icon)] shrink-0" />
                初始化未完成
              </DropdownMenuItem>
              <DropdownMenuItem
                className={`text-xs gap-2 ${filter === "anomalous" ? "font-medium text-[#020617]" : ""}`}
                onClick={() => setFilter("anomalous")}
              >
                <span className="w-2 h-2 rounded-full bg-red-500 shrink-0" />
                异常
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          {/* 刷新按钮 */}
          <Button
            variant="claw-outline"
            size="icon-sm"
            onClick={() => {
              if (onRefreshSync) {
                onRefreshSync();
              } else {
                toast.success("组织列表已刷新");
              }
            }}
            title="刷新"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </Button>
        </div>
      </div>

      {/* 「同步数据源信息」引导按钮：仅 OneID 模式下显示。
          点击 → 原地旋转（disabled + Loader2 + "同步中..."）→ 同步完成后按钮从会话中隐藏；
          完成 toast 由父级 handleSyncDepts 内置 toast.success("已同步数据源") 提供。 */}
      {deptSynced !== undefined && (!syncRequested || isSyncingDepts) && (
        <div className="flex justify-center px-3 pb-2">
          <Button
            variant="claw-outline"
            size="sm"
            className="gap-1.5 h-8 px-3 font-medium"
            onClick={() => {
              if (isSyncingDepts || syncRequested) return;
              setSyncRequested(true);
              onSyncDepts?.();
            }}
            disabled={isSyncingDepts}
          >
            {isSyncingDepts ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
            ) : (
              <RefreshCw className="w-3.5 h-3.5" />
            )}
            {isSyncingDepts ? "同步中..." : "同步数据源信息"}
          </Button>
        </div>
      )}

      {/* 列表 */}
      <div className="flex-1 overflow-y-auto pb-3">
        {/* OneID 模式：单一合并树递归渲染，A公司 为唯一顶层
            行级 props（是否显示人数 / 是否显示操作按钮）由 GroupRow 内部按
            node.source / node.readonly 动态判断，无需在此处分桶传不同 prop */}
        {deptSynced !== undefined && buckets.merged && (
          <>
            {buckets.merged.map((n) => (
              <GroupRow
                key={n.id}
                node={n}
                users={users}
                groups={groups}
                selectedId={selectedId}
                onSelect={onSelect}
                expanded={expanded}
                onToggle={toggle}
                keyword={keyword}
                isManualMode={true}
                mergedTreeMode={true}
                onAddChildGroup={onAddChildGroup}
                onEditGroup={onEditGroup}
                onDeleteGroup={onDeleteGroup}
                anomalousGroupIds={anomalousGroupIds}
                directAnomalousGroupIds={directAnomalousGroupIds}
                uninitializedGroupIds={uninitializedGroupIds}
                directUninitializedGroupIds={directUninitializedGroupIds}
                networkOutdatedGroupIds={networkOutdatedGroupIds}
                filterFn={matchFilter}
                anomalousGroupDetails={anomalousGroupDetails}
              />
            ))}
            {/* 全部为空时的占位（顶部已有同步按钮，不再重复引导） */}
            {buckets.merged.length === 0 && (
              <div className="px-4 py-10 text-center text-xs text-[#A3A3A3]">
                暂无组织
              </div>
            )}
          </>
        )}

        {/* 自建组织桶（仅普通模式下直接渲染，OneID 模式下已在上方合并渲染） */}
        {deptSynced === undefined && (buckets.manual as GroupTreeNode[]).length > 0 && (
          <>
            {(buckets.manual as GroupTreeNode[]).map((n) => (
              <GroupRow
                key={n.id}
                node={n}
                users={users}
                groups={groups}
                selectedId={selectedId}
                onSelect={onSelect}
                expanded={expanded}
                onToggle={toggle}
                keyword={keyword}
                isManualMode={isManualMode}
                onAddChildGroup={onAddChildGroup}
                onEditGroup={onEditGroup}
                onDeleteGroup={onDeleteGroup}
                anomalousGroupIds={anomalousGroupIds}
                directAnomalousGroupIds={directAnomalousGroupIds}
                uninitializedGroupIds={uninitializedGroupIds}
                directUninitializedGroupIds={directUninitializedGroupIds}
                networkOutdatedGroupIds={networkOutdatedGroupIds}
                filterFn={matchFilter}
                anomalousGroupDetails={anomalousGroupDetails}
              />
            ))}
          </>
        )}

        {groups.length === 0 && deptSynced !== false && (
          <div className="px-4 py-10 text-center text-xs text-[#A3A3A3]">
            暂无组织，可新建自建组织
          </div>
        )}

        {/* 筛选无结果占位符 */}
        {filter !== "all" && groups.length > 0 && (() => {
          const allTrees: GroupTreeNode[] = buckets.merged
            ? buckets.merged
            : (buckets.manual as GroupTreeNode[]);
          const hasAnyMatch = allTrees.some(matchFilter);
          if (hasAnyMatch) return null;
          return (
            <Empty className="border-0 px-4 py-10">
              <EmptyMedia />
              <EmptyHeader>
                <EmptyDescription>暂无符合筛选条件的组织</EmptyDescription>
              </EmptyHeader>
            </Empty>
          );
        })()}
      </div>

      {/* 底部固定：未分配组织 */}
      <div className="border-t border-[var(--cp-border,#EAEEF4)] shrink-0">
        <div
          className={`group flex items-center gap-1.5 h-8 pr-3 text-sm cursor-pointer rounded-[4px] mx-3 mt-1 transition-colors ${
            isUnassignedActive
              ? "bg-[var(--bg-grey-hover)] text-[#09090b] font-medium"
              : "text-[#09090b] hover:bg-[var(--bg-grey-hover)]"
          }`}
          style={{ paddingLeft: 8 }}
          onClick={() => onSelect(UNASSIGNED_GROUP_ID)}
        >
          <span className="w-4 h-4 shrink-0 flex items-center justify-center">
            <UserX className={`w-3.5 h-3.5 ${isUnassignedActive ? "text-[#71717a]" : "text-[#a1a1aa]"}`} />
          </span>
          <span className="truncate">未分配组织</span>
          <span className={`text-[11px] tabular-nums shrink-0 ml-0.5 ${isUnassignedActive ? "text-[#71717a]" : "text-[#a1a1aa]"}`}>
            ({unassignedCount})
          </span>
        </div>
      </div>
    </div>
  );
}

function countDescendants(n: GroupTreeNode): number {
  return n.children.reduce(
    (acc, c) => acc + 1 + countDescendants(c),
    0
  );
}
