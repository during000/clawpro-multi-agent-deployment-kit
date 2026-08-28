/**
 * GroupSelect - 普通组织选择下拉面板
 *
 * 从 PlatformPolicy 页的 GroupTagSelector 抽取的公共组件。
 * 功能：树形多选组织 + 搜索 + 标签展示 + 清除 + 父子级联（自动聚合/展开）
 *
 * 使用场景：
 *   - 平台策略页：策略规则的组织范围选择
 *   - 添加用户弹窗：用户组织选择
 *   - 其他需要多选组织的场景
 */
import React, { useState, useMemo } from "react";
import { ChevronDown, ChevronRight, Search, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { BodyText, HelperText, MetaText, MetaMedium } from "@/components/ui/Typography";
import { buildGroupTree, buildUnifiedGroupTree, type GroupTreeNode } from "@/pages/admin/MemberManagement/health";
import type { UserGroup, GroupSource } from "@/pages/admin/MemberManagement/types";

// ─── 工具函数 ─────────────────────────────────────────────────────────────────

/** 获取组织的完整路径名（含祖先） */
function getGroupPath(groupId: string, groups: UserGroup[]): string {
  const g = groups.find((x) => x.id === groupId);
  if (!g) return groupId;
  const parts: string[] = [g.name];
  let current = g;
  while (current.parentId) {
    const parent = groups.find((x) => x.id === current.parentId);
    if (!parent) break;
    parts.unshift(parent.name);
    current = parent;
  }
  return parts.join(" / ");
}

type CheckState = "checked" | "unchecked" | "indeterminate";

/** 任一子孙被选中（不含自身） */
function hasSelectedDescendant(node: GroupTreeNode, selectedIds: Set<string>): boolean {
  for (const c of node.children) {
    if (selectedIds.has(c.id)) return true;
    if (hasSelectedDescendant(c, selectedIds)) return true;
  }
  return false;
}

/** 三态：本身被选=checked；祖先被选=checked；有子孙被选=indeterminate；其他=unchecked */
function getCheckState(
  node: GroupTreeNode,
  selectedIds: Set<string>,
  groupMap: Map<string, UserGroup>
): CheckState {
  if (selectedIds.has(node.id)) return "checked";
  let cur: UserGroup | undefined = groupMap.get(node.id);
  while (cur && cur.parentId) {
    if (selectedIds.has(cur.parentId)) return "checked";
    cur = groupMap.get(cur.parentId);
  }
  if (hasSelectedDescendant(node, selectedIds)) return "indeterminate";
  return "unchecked";
}

function getDescendantIds(node: GroupTreeNode): string[] {
  const ids: string[] = [node.id];
  node.children.forEach((c) => ids.push(...getDescendantIds(c)));
  return ids;
}

/**
 * 递归向上聚合：若某父节点的所有直接可用（非 disabled）子节点都已被选中，
 * 则将这些子节点 id 全部移除，换成该父节点 id。继续向上直到无法再聚合。
 */
function aggregateSelection(
  selected: Set<string>,
  roots: GroupTreeNode[],
  disabledIds: Set<string>
): Set<string> {
  const result = new Set(selected);
  let changed = true;
  while (changed) {
    changed = false;
    const walk = (node: GroupTreeNode) => {
      if (node.children.length === 0) return;
      node.children.forEach(walk);
      if (result.has(node.id)) return;
      // 父节点自身已被占用（disabled）时不可作为聚合目标
      if (disabledIds.has(node.id)) return;
      const hasDisabled = node.children.some((c) => disabledIds.has(c.id));
      if (hasDisabled) return;
      const allSelected = node.children.every((c) => result.has(c.id));
      if (!allSelected) return;
      node.children.forEach((c) => result.delete(c.id));
      result.add(node.id);
      changed = true;
    };
    roots.forEach(walk);
  }
  return result;
}

// ─── 默认 source 标签 ─────────────────────────────────────────────────────────

const DEFAULT_SOURCE_LABELS: Record<GroupSource, string> = {
  "oneid-dept": "部门",
  "oneid-group": "用户组",
  "manual": "自定义组织",
  "project": "项目",
};

// ─── 组件 Props ───────────────────────────────────────────────────────────────

/**
 * 面板变体：
 * - "default"：纯选择面板，footer 仅在有选中项时显示"已选 X 项"+"清除"
 * - "confirm"：带确认操作的面板，footer 显示"已选 X 项"+"清除"+"保存"按钮
 */
export type GroupSelectVariant = "default" | "confirm";

export interface GroupSelectProps {
  /** 所有可选的组织列表 */
  groups: UserGroup[];
  /** 当前已选中的组织 ID 列表 */
  selectedIds: string[];
  /** 选中状态变化回调 */
  onChange: (ids: string[]) => void;
  /** 不可选的组织 ID 列表（灰显） */
  disabledIds?: string[];
  /** 禁用项的 tooltip 文案 */
  disabledTooltip?: string;
  /** 仅展示指定 source 类型的组织（默认：部门 + 自定义组织） */
  sourceFilter?: GroupSource[];
  /** 各 source 的小标题（默认：部门/用户组/自定义组织） */
  sourceLabels?: Partial<Record<GroupSource, string>>;
  /** placeholder 文案 */
  placeholder?: string;
  /** 是否启用父子级联聚合逻辑（默认 true） */
  enableAggregation?: boolean;
  /**
   * 触发器紧凑模式（默认 false）。
   * - false：多选标签换行展示，触发器高度随已选项增多而变高
   * - true：触发器高度固定（单行），多选时仅展示「首个标签 + +N」，
   *         hover 触发器时通过 tooltip 展示全部已选项
   */
  compactTrigger?: boolean;
  /**
   * 面板变体（默认 "default"）
   * - "default"：纯选择面板，footer 仅在有选中项时显示"已选 X 项"+"清除"
   * - "confirm"：带确认操作的面板，footer 显示"已选 X 项"+"清除"+"保存"按钮
   */
  variant?: GroupSelectVariant;
  /** 保存回调：点击面板内"保存"按钮时触发（仅 variant="confirm" 时生效） */
  onSave?: () => void;
  /**
   * 自定义触发器节点（可选）。
   * - 设置后将完全覆盖默认 trigger 外观（也忽略 compactTrigger）。
   * - 适合「图标按钮 / Badge / 文字 + tag」等非标准下拉触发场景。
   * - 仍可点击展开面板，面板内容与默认一致。
   */
  customTrigger?: React.ReactNode;
  /** 面板宽度（仅 customTrigger 模式生效，默认 280px） */
  panelWidth?: number;
}

// ─── 组件实现 ─────────────────────────────────────────────────────────────────

export function GroupSelect({
  groups,
  selectedIds,
  onChange,
  disabledIds = [],
  disabledTooltip = "该组织不可选择",
  sourceFilter = ["oneid-dept", "manual"],
  sourceLabels,
  placeholder = "请选择组织",
  enableAggregation = true,
  variant = "default",
  onSave,
  compactTrigger = false,
  customTrigger,
  panelWidth,
}: GroupSelectProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const labels = { ...DEFAULT_SOURCE_LABELS, ...sourceLabels };

  // 仅保留指定 source 类型的组织
  const visibleGroups = useMemo(
    () => groups.filter((g) => sourceFilter.includes(g.source)),
    [groups, sourceFilter]
  );
  const groupMap = useMemo(
    () => new Map(visibleGroups.map((g) => [g.id, g])),
    [visibleGroups]
  );

  // 合并树模式：当 sourceFilter 含 oneid-dept 且数据中真的有 oneid-dept 时启用，
  // 此时把可见组织（visibleGroups）合并为一棵树（A公司 / dept-root 为唯一顶层），
  // 原本顶层的 oneid-group / manual 节点（parentId=null）挂到 dept-root 下。
  // 否则保持旧的"按 source 分桶 + 各自 buildGroupTree + 分区标题"行为，兼容纯本地等场景。
  const unifiedMode = useMemo(
    () => sourceFilter.includes("oneid-dept") && visibleGroups.some((g) => g.source === "oneid-dept"),
    [sourceFilter, visibleGroups]
  );

  // 按 source 分桶 + 建树（仅普通模式使用）
  const groupsBySource = useMemo(() => {
    const buckets: Record<string, UserGroup[]> = {};
    sourceFilter.forEach((s) => { buckets[s] = []; });
    visibleGroups.forEach((g) => { if (buckets[g.source]) buckets[g.source].push(g); });
    return buckets;
  }, [visibleGroups, sourceFilter]);

  const activeSources = useMemo(
    () => sourceFilter.filter((s) => (groupsBySource[s] || []).length > 0),
    [groupsBySource, sourceFilter]
  );

  const treesMap = useMemo(() => {
    const map: Record<string, GroupTreeNode[]> = {};
    activeSources.forEach((s) => { map[s] = buildGroupTree(groupsBySource[s] || []); });
    return map;
  }, [activeSources, groupsBySource]);

  // 合并树（仅 unifiedMode 使用）
  const unifiedTrees = useMemo(
    () => (unifiedMode ? buildUnifiedGroupTree(visibleGroups) : []),
    [unifiedMode, visibleGroups]
  );

  const allRoots = useMemo(
    () => (unifiedMode ? unifiedTrees : activeSources.flatMap((s) => treesMap[s] || [])),
    [unifiedMode, unifiedTrees, activeSources, treesMap]
  );

  const disabledSet = useMemo(() => new Set(disabledIds), [disabledIds]);

  // 打开时：默认展开已选祖先 + 根节点
  const handleOpenChange = (v: boolean) => {
    if (v) {
      setSearch("");
      const expandSet = new Set<string>();
      selectedIds.forEach((gid) => {
        let cur = groupMap.get(gid);
        while (cur && cur.parentId) {
          expandSet.add(cur.parentId);
          cur = groupMap.get(cur.parentId);
        }
      });
      activeSources.forEach((s) => {
        treesMap[s]?.forEach((root) => expandSet.add(root.id));
      });
      // 合并模式下还需展开合并树的根节点（dept-root）
      if (unifiedMode) {
        unifiedTrees.forEach((root) => expandSet.add(root.id));
      }
      setExpanded(expandSet);
    }
    setOpen(v);
  };

  const toggleExpand = (id: string) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  const findTreeNode = (id: string): GroupTreeNode | null => {
    const walk = (nodes: GroupTreeNode[]): GroupTreeNode | null => {
      for (const n of nodes) {
        if (n.id === id) return n;
        const found = walk(n.children);
        if (found) return found;
      }
      return null;
    };
    return walk(allRoots);
  };

  const toggleNode = (node: GroupTreeNode) => {
    if (disabledSet.has(node.id)) return;
    const ids = new Set(selectedIds);

    if (!enableAggregation) {
      // 简单多选模式：toggle 单个节点
      if (ids.has(node.id)) {
        ids.delete(node.id);
      } else {
        ids.add(node.id);
      }
      onChange(Array.from(ids));
      return;
    }

    // 聚合模式：支持祖先展开/子孙合并
    let ancestorSelectedId: string | null = null;
    let cur: UserGroup | undefined = groupMap.get(node.id);
    while (cur && cur.parentId) {
      if (ids.has(cur.parentId)) { ancestorSelectedId = cur.parentId; break; }
      cur = groupMap.get(cur.parentId);
    }

    if (ids.has(node.id)) {
      ids.delete(node.id);
    } else if (ancestorSelectedId) {
      ids.delete(ancestorSelectedId);
      const pathNodes: UserGroup[] = [];
      let p: UserGroup | undefined = groupMap.get(node.id);
      while (p && p.id !== ancestorSelectedId) {
        pathNodes.push(p);
        p = p.parentId ? groupMap.get(p.parentId) : undefined;
      }
      pathNodes.reverse();
      let cursor = findTreeNode(ancestorSelectedId);
      for (let i = 0; i < pathNodes.length; i++) {
        const nextHopId = pathNodes[i].id;
        if (!cursor) break;
        cursor.children.forEach((c) => {
          if (c.id !== nextHopId && !disabledSet.has(c.id)) ids.add(c.id);
        });
        cursor = cursor.children.find((c) => c.id === nextHopId) || null;
      }
    } else {
      const state = hasSelectedDescendant(node, ids) ? "indeterminate" : "unchecked";
      if (state === "indeterminate") {
        getDescendantIds(node).forEach((d) => ids.delete(d));
      }
      ids.add(node.id);
    }

    const aggregated = aggregateSelection(ids, allRoots, disabledSet);
    onChange(Array.from(aggregated));
  };

  // 搜索过滤
  const matchedIds = useMemo(() => {
    if (!search.trim()) return null;
    const q = search.toLowerCase();
    return new Set(
      visibleGroups
        .filter((g) => g.name.toLowerCase().includes(q) || getGroupPath(g.id, visibleGroups).toLowerCase().includes(q))
        .map((g) => g.id)
    );
  }, [search, visibleGroups]);

  const isVisible = (node: GroupTreeNode): boolean => {
    if (!matchedIds) return true;
    if (matchedIds.has(node.id)) return true;
    return node.children.some(isVisible);
  };

  const renderNode = (node: GroupTreeNode, depth: number) => {
    if (!isVisible(node)) return null;
    const isDisabled = disabledSet.has(node.id);
    const checkState = isDisabled
      // 禁用节点始终 unchecked：不受父级选中影响，不展示蓝色选中态
      ? "unchecked" as CheckState
      : enableAggregation
        ? getCheckState(node, new Set(selectedIds), groupMap)
        : selectedIds.includes(node.id) ? "checked" as CheckState : "unchecked" as CheckState;
    const isExpanded = expanded.has(node.id);
    const hasChildren = node.children.length > 0;
    // 禁用节点灰色勾选态：当父级（或祖先）被选中时，表示此节点在覆盖范围内，展示灰勾
    const disabledChecked = isDisabled && (() => {
      let cur: UserGroup | undefined = groupMap.get(node.id);
      while (cur && cur.parentId) {
        if (new Set(selectedIds).has(cur.parentId)) return true;
        cur = groupMap.get(cur.parentId);
      }
      return false;
    })();

    const nameSpan = <BodyText as="span" tone="secondary" className="truncate text-left">{node.name}</BodyText>;

    return (
      <div key={node.id} className="mt-0.5 first:mt-0">
        <button
          type="button"
          onClick={() => !isDisabled && toggleNode(node)}
          disabled={isDisabled}
          className={`w-full flex items-center gap-2 h-8 px-3 rounded-[6px] transition-colors text-left ${isDisabled ? "opacity-40 cursor-not-allowed" : checkState === "checked" ? "bg-[var(--bg-brand-selected)]" : "hover:bg-[var(--bg-grey-hover)]"}`}
          style={{ paddingLeft: 8 + depth * 16 }}
        >
          {hasChildren ? (
            <button
              type="button"
              onClick={(e) => { e.stopPropagation(); toggleExpand(node.id); }}
              className="w-4 h-4 flex items-center justify-center text-[var(--text-muted)] hover:text-[var(--text-secondary)] shrink-0"
            >
              {isExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
            </button>
          ) : (
            <span className="w-4 h-4 shrink-0" />
          )}
          <Checkbox
            checked={disabledChecked || (checkState === "checked" ? true : checkState === "indeterminate" ? "indeterminate" : false)}
            disabled={isDisabled}
            className="pointer-events-none"
          />
          {isDisabled ? (
            <Tooltip>
              <TooltipTrigger asChild>{nameSpan}</TooltipTrigger>
              <TooltipContent side="right" sideOffset={8} className="text-xs max-w-[240px] leading-relaxed">
                {disabledTooltip}
              </TooltipContent>
            </Tooltip>
          ) : (
            nameSpan
          )}
        </button>
        {hasChildren && isExpanded && node.children.map((c) => renderNode(c, depth + 1))}
      </div>
    );
  };

  // 选中项详细数据（用于 tag 渲染）
  const selectedItems = useMemo(() => {
    return selectedIds.map((id) => {
      const g = groupMap.get(id);
      return { id, name: g ? g.name : id };
    });
  }, [selectedIds, groupMap]);

  // 移除单个 tag
  const removeTag = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    onChange(selectedIds.filter((x) => x !== id));
  };

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        {customTrigger ? (
          <span className="inline-flex">{customTrigger}</span>
        ) : (
          <button
            type="button"
            data-state={open ? "open" : "closed"}
            className={cn(
              "flex w-full items-center justify-between gap-2 border border-border bg-white px-2 py-0.5 text-xs font-normal transition-colors outline-none rounded-[4px]",
              compactTrigger ? "h-7 whitespace-nowrap" : "min-h-9",
              "hover:border-[#355EF1] data-[state=open]:border-[#355EF1]",
              "disabled:cursor-not-allowed disabled:bg-[#FAFAFA] disabled:border-[var(--border)] disabled:text-gray-400",
            )}
          >
            {selectedItems.length === 0 ? (
              <span className="flex-1 min-w-0 truncate text-left text-[var(--text-weak)] px-1">{placeholder}</span>
            ) : compactTrigger ? (
              // 紧凑模式：单行横向滚动，展示全部已选 tag（支持鼠标滚轮 / 触控板左右滑动查看）
              // - 外层 relative + 右侧渐隐遮罩，提示存在被截断的内容
              // - 内层超薄滚动条仅在 hover 时显示，避免占用高度
              // - 不在容器上 stopPropagation onClick，否则会阻止 PopoverTrigger 展开面板
              <span className="relative flex-1 min-w-0 flex items-center">
                <span
                  className={cn(
                    "flex-1 min-w-0 flex items-center gap-1 overflow-x-auto overflow-y-hidden",
                    // 隐藏滚动条 UI 但保留滚动能力（Firefox / IE / WebKit）
                    "[scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden",
                  )}
                  onWheel={(e) => {
                    // 将垂直滚轮转换为水平滚动，避免阻塞页面/弹层滚动
                    if (e.deltaY !== 0) {
                      e.currentTarget.scrollLeft += e.deltaY;
                      e.stopPropagation();
                    }
                  }}
                >
                  {selectedItems.map((item) => (
                    <Badge
                      key={item.id}
                      variant="secondary"
                      className="px-2 py-0.5 text-xs inline-flex items-center gap-1 max-w-[140px] shrink-0"
                    >
                      <span className="block truncate">{item.name}</span>
                      <span
                        role="button"
                        tabIndex={-1}
                        aria-label={`移除 ${item.name}`}
                        onClick={(e) => removeTag(item.id, e)}
                        className="inline-flex items-center justify-center hover:text-[var(--text-secondary)] cursor-pointer"
                      >
                        <X className="w-3 h-3" />
                      </span>
                    </Badge>
                  ))}
                </span>
                {/* 右侧渐隐遮罩：暗示存在更多内容可滑动查看；不拦截点击 */}
                <span
                  aria-hidden
                  className="pointer-events-none absolute inset-y-0 right-0 w-6 bg-gradient-to-l from-white to-transparent"
                />
              </span>
            ) : (
              // 默认模式：所有 tag 换行展示
              <span className="flex-1 min-w-0 flex flex-wrap items-center gap-1">
                {selectedItems.map((item) => (
                  <Badge
                    key={item.id}
                    variant="secondary"
                    className="px-2 py-0.5 text-xs inline-flex items-center gap-1 max-w-[160px]"
                  >
                    <span className="block truncate">{item.name}</span>
                    <span
                      role="button"
                      tabIndex={-1}
                      aria-label={`移除 ${item.name}`}
                      onClick={(e) => removeTag(item.id, e)}
                      className="inline-flex items-center justify-center hover:text-[var(--text-secondary)] cursor-pointer"
                    >
                      <X className="w-3 h-3" />
                    </span>
                  </Badge>
                ))}
              </span>
            )}
            <ChevronDown className={cn("w-4 h-4 text-[var(--text-weak)] shrink-0 transition-transform self-center", open && "rotate-180")} />
          </button>
        )}
      </PopoverTrigger>
      <PopoverContent
        className="p-0 rounded-[4px] border-none shadow-[var(--shadow-popover)] flex flex-col max-h-[360px]"
        style={{ minWidth: 220, width: customTrigger ? (panelWidth ?? 280) : "var(--radix-popover-trigger-width)" }}
        align="start"
        sideOffset={4}
        onWheel={(e) => e.stopPropagation()}
      >
        {/* 搜索框 */}
        <div className="px-2 pt-2 pb-0 shrink-0">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)] pointer-events-none" />
            <Input
              type="text"
              placeholder="搜索组织"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="h-8 pl-8 pr-2 text-sm"
              onClick={(e) => e.stopPropagation()}
            />
          </div>
        </div>
        {/* 列表区 - 可滚动 */}
        <div
          className="flex-1 min-h-0 overflow-y-auto overscroll-contain px-2 pt-1 pb-2 space-y-0.5"
          onWheel={(e) => e.stopPropagation()}
        >
          {unifiedMode ? (
            unifiedTrees.length === 0 || !unifiedTrees.some(isVisible) ? (
              <div className="text-center py-4"><HelperText>暂无组织</HelperText></div>
            ) : (
              unifiedTrees.map((root) => renderNode(root, 0))
            )
          ) : activeSources.length === 0 ? (
            <div className="text-center py-4"><HelperText>暂无组织</HelperText></div>
          ) : (
            activeSources.map((source, sourceIdx) => {
              const trees = treesMap[source] || [];
              const hasVisibleTrees = trees.some(isVisible);
              if (!hasVisibleTrees) return null;
              return (
                <div key={source} className="mb-1 last:mb-0">
                  {activeSources.length > 1 && (
                    <div className={cn("px-3 pb-1", sourceIdx === 0 ? "pt-1" : "pt-2")}>
                      <MetaMedium tone="weak">{labels[source]}</MetaMedium>
                    </div>
                  )}
                  {trees.map((root) => renderNode(root, 0))}
                </div>
              );
            })
          )}
        </div>
        {/* Footer：已选计数 + 清除（即时生效模式无需确认按钮） */}
        {variant === "confirm" ? (
          <div className="shrink-0 mx-2 border-t border-border py-2 flex items-center justify-between">
            <MetaText>{selectedIds.length > 0 ? `已选 ${selectedIds.length} 项` : ""}</MetaText>
            <div className="flex items-center gap-2">
              <Button variant="claw-outline" size="sm" className="text-xs h-7 px-2" onClick={() => onChange([])}>
                清除
              </Button>
              <Button variant="dialog-confirm" size="sm" className="text-xs h-7 px-3" onClick={() => { setOpen(false); onSave?.(); }}>
                保存
              </Button>
            </div>
          </div>
        ) : (
          selectedIds.length > 0 && (
            <div className="shrink-0 mx-2 border-t border-border py-2 flex items-center justify-between">
              <MetaText>已选 {selectedIds.length} 项</MetaText>
              <Button variant="claw-outline" size="sm" className="text-xs h-7 px-2" onClick={() => onChange([])}>
                清除
              </Button>
            </div>
          )
        )}
      </PopoverContent>
    </Popover>
  );
}

export default GroupSelect;
