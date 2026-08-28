/**
 * ScopeEditPopover - 通用应用范围编辑下拉面板
 *
 * 规范：
 *   - SegmentGroup 切换「全部用户 / 按组织」（传入 projects 时该项显示为「按组织/项目」，白底黑字选中态）
 *   - Checkbox 组件选择组织（树结构）
 *   - 确认按钮使用 dialog-confirm（纯黑底白字）
 *   - 取消按钮使用 outline
 *   - 已选计数在 footer 左端
 */
import { useState, useMemo, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";

import { AllUsersTag } from "@/components/ui/all-users-tag";
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  Popover, PopoverContent, PopoverTrigger,
} from "@/components/ui/popover";
import { CompactText, MetaText, MetaMedium } from "@/components/ui/Typography";
import { Pencil, X, ChevronRight, ChevronDown } from "lucide-react";

// ─── 类型定义 ────────────────────────────────────────────────
export interface ScopeGroup {
  id: string;
  name: string;
  parentId?: string | null;
}

export type ScopeType = "all" | "groups";

export interface ScopeEditPopoverProps {
  /** 当前应用范围 */
  scope: ScopeType;
  /** 当前选中的组织 id 列表 */
  selectedGroupIds: string[];
  /** 所有可选组织 */
  groups: ScopeGroup[];
  /**
   * 可选：项目列表。传入且非空时，「按组织」面板会额外渲染「项目」分组小标题，
   * 允许同时选择组织与项目；不传则完全保持原有行为（仅选组织）。
   */
  projects?: ScopeGroup[];
  /** 保存回调 */
  onConfirm: (scope: ScopeType, groupIds: string[]) => void;
  /** 触发器自定义渲染（默认为铅笔图标按钮） */
  trigger?: React.ReactNode;
  /** Popover 对齐方式 */
  align?: "start" | "center" | "end";
  /** 是否显示范围徽章 */
  showBadges?: boolean;
  /** 已选范围的展示标签（默认使用 group name） */
  scopeLabels?: string[];
  /** 最多展示几个组织 tag，超出折叠为 +N（默认 1） */
  maxVisibleBadges?: number;
}

// ─── 树结构工具 ────────────────────────────────────────────
interface TreeNode {
  id: string;
  name: string;
  children: TreeNode[];
}

function buildTree(groups: ScopeGroup[]): TreeNode[] {
  const map = new Map<string, TreeNode>();
  groups.forEach((g) => map.set(g.id, { id: g.id, name: g.name, children: [] }));
  const roots: TreeNode[] = [];
  groups.forEach((g) => {
    const node = map.get(g.id)!;
    if (g.parentId && map.has(g.parentId)) {
      map.get(g.parentId)!.children.push(node);
    } else {
      roots.push(node);
    }
  });
  return roots;
}

export function ScopeEditPopover({
  scope,
  selectedGroupIds,
  groups,
  projects,
  onConfirm,
  trigger,
  align = "start",
  showBadges = true,
  scopeLabels,
  maxVisibleBadges = 1,
}: ScopeEditPopoverProps) {
  const [open, setOpen] = useState(false);
  const [draftScope, setDraftScope] = useState<ScopeType>("all");
  const [draftGroupIds, setDraftGroupIds] = useState<string[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const inputRef = useRef<HTMLInputElement>(null);

  const hasProjects = !!projects && projects.length > 0;

  // 树结构
  const tree = useMemo(() => buildTree(groups), [groups]);
  const projectTree = useMemo(() => buildTree(projects ?? []), [projects]);

  // 组织 + 项目 合并的名称查询（id 全局唯一）
  const nameOf = (id: string): string =>
    groups.find((g) => g.id === id)?.name ??
    (projects ?? []).find((p) => p.id === id)?.name ??
    id;

  // 打开时同步状态
  const handleOpenChange = (v: boolean) => {
    if (v) {
      setDraftScope(scope);
      setDraftGroupIds([...selectedGroupIds]);
      setSearchQuery("");
      // 默认展开已选组织/项目的祖先 + 根节点
      const expandSet = new Set<string>();
      const groupMap = new Map([...groups, ...(projects ?? [])].map((g) => [g.id, g]));
      selectedGroupIds.forEach((gid) => {
        let cur = groupMap.get(gid);
        while (cur && cur.parentId) {
          expandSet.add(cur.parentId);
          cur = groupMap.get(cur.parentId);
        }
      });
      tree.forEach((root) => expandSet.add(root.id));
      projectTree.forEach((root) => expandSet.add(root.id));
      setExpanded(expandSet);
    }
    setOpen(v);
  };

  const toggleExpand = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  // 搜索过滤（组织 + 项目）
  const matchedIds = useMemo(() => {
    if (!searchQuery.trim()) return null;
    const q = searchQuery.toLowerCase();
    const pool = [...groups, ...(projects ?? [])];
    return new Set(pool.filter((g) => g.name.toLowerCase().includes(q)).map((g) => g.id));
  }, [groups, projects, searchQuery]);

  const isNodeVisible = (node: TreeNode): boolean => {
    if (!matchedIds) return true;
    if (matchedIds.has(node.id)) return true;
    return node.children.some(isNodeVisible);
  };

  // 切换组织选择
  const toggleGroup = (gid: string) => {
    setDraftGroupIds((prev) =>
      prev.includes(gid) ? prev.filter((id) => id !== gid) : [...prev, gid]
    );
  };

  // 清除选择
  const handleClearSelection = () => {
    setDraftGroupIds([]);
    setSearchQuery("");
  };

  // 确认按钮禁用条件
  const isConfirmDisabled = draftScope === "groups" && draftGroupIds.length === 0;

  const handleConfirm = () => {
    if (isConfirmDisabled) return;
    onConfirm(draftScope, draftScope === "all" ? [] : draftGroupIds);
    setOpen(false);
  };

  // 判断是否为纯平铺列表（无任何节点有子节点）
  const isFlat = useMemo(() => tree.every((node) => node.children.length === 0), [tree]);
  const projectIsFlat = useMemo(
    () => projectTree.every((node) => node.children.length === 0),
    [projectTree]
  );

  // 渲染树节点
  const renderTreeNode = (node: TreeNode, depth: number, flat: boolean): React.ReactNode => {
    if (!isNodeVisible(node)) return null;
    const checked = draftGroupIds.includes(node.id);
    const hasChildren = node.children.length > 0;
    const isExpanded = expanded.has(node.id);
    return (
      <div key={node.id} className="mt-0.5 first:mt-0">
        <div
          className={`w-full flex items-center gap-1.5 h-8 rounded-[6px] transition-colors cursor-pointer ${
            checked ? "bg-[var(--bg-brand-selected)] text-blue-500 font-medium" : "hover:bg-[var(--bg-grey-hover)]"
          } ${flat ? "px-3" : ""}`}
          style={flat ? undefined : { paddingLeft: 12 + depth * 16 }}
          onClick={() => toggleGroup(node.id)}
        >
          {/* 展开/收起箭头（仅树形模式显示） */}
          {!flat && (
            hasChildren ? (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  toggleExpand(node.id);
                }}
                className="w-4 h-4 flex items-center justify-center text-[var(--text-weak)] hover:text-[var(--text-secondary)] shrink-0"
              >
                {isExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
              </button>
            ) : (
              <span className="w-4 h-4 shrink-0" />
            )
          )}
          <Checkbox
            checked={checked}
            onCheckedChange={() => toggleGroup(node.id)}
            onClick={(e) => e.stopPropagation()}
          />
          <CompactText tone="emphasis" className="truncate">{node.name}</CompactText>
        </div>
        {hasChildren && isExpanded && node.children.map((c) => renderTreeNode(c, depth + 1, flat))}
      </div>
    );
  };

  // 渲染范围徽章
  const renderBadges = () => {
    if (!showBadges) return null;

    if (scope === "all") {
      return (
        <AllUsersTag />
      );
    }

    const labels = scopeLabels || selectedGroupIds.map((gid) => nameOf(gid));

    if (labels.length === 0) {
      return (
        <Badge variant="outline">
          未选组织
        </Badge>
      );
    }

    const visibleCount = Math.max(1, maxVisibleBadges);
    const visibleLabels = labels.slice(0, visibleCount);
    const rest = labels.length - visibleLabels.length;

    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex items-center gap-1 cursor-default flex-wrap">
            {visibleLabels.map((label, idx) => (
              <Badge
                key={`${label}-${idx}`}
                variant="secondary"
                className="max-w-[140px]"
              >
                <span className="block truncate max-w-[124px]">{label}</span>
              </Badge>
            ))}
            {rest > 0 && (
              <Badge variant="secondary">
                +{rest}
              </Badge>
            )}
          </span>
        </TooltipTrigger>
        <TooltipContent side="top" className="max-w-[280px] leading-relaxed">
          <MetaText tone="inherit">{labels.join("，")}</MetaText>
        </TooltipContent>
      </Tooltip>
    );
  };

  return (
    <div className="inline-flex items-center gap-1.5">
      {renderBadges()}
      <Popover open={open} onOpenChange={handleOpenChange}>
        <PopoverTrigger asChild>
          {trigger || (
            <button
              onClick={(e) => e.stopPropagation()}
              className="self-center text-[var(--text-weak)] hover:text-[var(--text-brand)] transition-colors"
              title="编辑应用范围"
            >
              <Pencil className="w-3 h-3" />
            </button>
          )}
        </PopoverTrigger>
        <PopoverContent
          className="w-72 p-0 rounded-[4px] border-none shadow-[var(--shadow-popover)] flex flex-col"
          align={align}
          sideOffset={6}
          onClick={(e) => e.stopPropagation()}
        >
          {/* 内容区 */}
          <div className="p-2 space-y-2">
            {/* Segment 切换：全部用户 / 按组织 */}
            <SegmentGroup className="w-full">
              <SegmentOption
                active={draftScope === "all"}
                onClick={() => setDraftScope("all")}
                className="flex-1"
              >
                全部用户
              </SegmentOption>
              <SegmentOption
                active={draftScope === "groups"}
                onClick={() => setDraftScope("groups")}
                className="flex-1"
              >
                {hasProjects ? "按组织/项目" : "按组织"}
              </SegmentOption>
            </SegmentGroup>

            {/* 按组织内容 */}
            {draftScope === "groups" && (
              <div className="space-y-2.5">
                {/* 输入框：已选标签 + 搜索输入 */}
                <div
                  className="flex flex-wrap items-center gap-1 px-2.5 py-1.5 min-h-[36px] max-h-[80px] overflow-y-auto border border-border rounded-[4px] bg-white focus-within:border-ring transition-colors cursor-text"
                  onClick={() => inputRef.current?.focus()}
                >
                  {draftGroupIds.map((gid) => {
                    const name = nameOf(gid);
                    return (
                      <span key={gid} className="inline-flex items-center gap-1 h-5 rounded-full bg-[#F5F5F5] px-2 text-xs text-[var(--text-title)] leading-none">
                        <span className="truncate max-w-[100px] leading-none">{name}</span>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            toggleGroup(gid);
                          }}
                          className="inline-flex items-center justify-center text-[var(--text-muted)] hover:text-[var(--text-emphasis)] shrink-0 h-3 w-3"
                        >
                          <X className="w-3 h-3" />
                        </button>
                      </span>
                    );
                  })}
                  <input
                    ref={inputRef}
                    type="text"
                    placeholder={draftGroupIds.length === 0 ? (hasProjects ? "搜索组织 / 项目…" : "搜索组织…") : ""}
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="flex-1 min-w-[60px] h-5 text-sm bg-transparent outline-none placeholder:text-[var(--text-weak)]"
                  />
                  {(draftGroupIds.length > 0 || searchQuery) && (
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        handleClearSelection();
                      }}
                      className="text-[var(--text-weak)] hover:text-[var(--text-secondary)] shrink-0 ml-auto"
                    >
                      <X className="w-3.5 h-3.5" />
                    </button>
                  )}
                </div>

                {/* 组织 / 项目 列表 */}
                <div className="max-h-[220px] overflow-y-auto pb-2" onWheel={(e) => e.stopPropagation()}>
                  {tree.length === 0 && projectTree.length === 0 ? (
                    <MetaText as="p" tone="weak" className="text-center py-4">
                      暂无组织
                    </MetaText>
                  ) : (
                    (() => {
                      const orgVisible = tree.filter(isNodeVisible);
                      const projVisible = projectTree.filter(isNodeVisible);
                      if (matchedIds && orgVisible.length === 0 && projVisible.length === 0) {
                        return (
                          <MetaText as="p" tone="weak" className="text-center py-4">
                            {hasProjects ? "无匹配结果" : "无匹配组织"}
                          </MetaText>
                        );
                      }
                      return (
                        <>
                          {/* 组织分区 */}
                          {orgVisible.length > 0 && (
                            <>
                              {hasProjects && (
                                <div className="px-3 pt-0.5 pb-1 select-none">
                                  <MetaMedium tone="weak">组织</MetaMedium>
                                </div>
                              )}
                              {orgVisible.map((node) => renderTreeNode(node, 0, isFlat))}
                            </>
                          )}
                          {/* 项目分区 */}
                          {hasProjects && projVisible.length > 0 && (
                            <>
                              <div className="px-3 pt-2 pb-1 select-none">
                                <MetaMedium tone="weak">项目</MetaMedium>
                              </div>
                              {projVisible.map((node) => renderTreeNode(node, 0, projectIsFlat))}
                            </>
                          )}
                        </>
                      );
                    })()
                  )}
                </div>
              </div>
            )}

            {draftScope === "all" && null}
          </div>

          {/* 底部：已选计数 + 按钮 */}
          <div className="shrink-0 mx-2 border-t border-[#EAEEF4] py-2 flex items-center">
            <MetaText className="flex-1">
              {draftScope === "groups" && draftGroupIds.length > 0
                ? `已选 ${draftGroupIds.length} 项`
                : ""}
            </MetaText>
            <div className="flex items-center gap-2">
              <Button
                variant="claw-outline"
                size="sm"
                className="text-xs h-7 px-2"
                onClick={() => setOpen(false)}
              >
                取消
              </Button>
              <Button
                variant="dialog-confirm"
                size="sm"
                className="text-xs h-7 px-3"
                disabled={isConfirmDisabled}
                onClick={handleConfirm}
              >
                确认
              </Button>
            </div>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
