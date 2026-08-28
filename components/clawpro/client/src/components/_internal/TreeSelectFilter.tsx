/**
 * TreeSelectFilter - 筛选下拉面板（树结构单选）
 *
 * 通用的树形单选筛选组件，视觉对齐标准 Select 组件。
 * 支持：搜索、树形展开/折叠、单选+确认、底部路径面包屑。
 *
 * 使用场景：部门筛选、组织筛选等需要树形单选的 toolbar 筛选器。
 */
import { useState, useEffect, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { MetaText } from "@/components/ui/Typography";
import {
  Popover, PopoverContent, PopoverTrigger,
} from "@/components/ui/popover";
import { ChevronDown, ChevronRight, Check, Search } from "lucide-react";

// ─── 类型定义 ────────────────────────────────────────────────────────────────

/** 树节点 */
export interface TreeNode {
  id: string;
  name: string;
  /** 节点路径（可选），用于底部面包屑展示，格式如 "A公司/技术部/前端组" */
  path?: string;
  children?: TreeNode[];
}

/** 一个分区（部门 / 自定义分组），可包含多棵树 */
export interface TreeSelectSection {
  /** 分区唯一 key（用于 React key 与内部区分） */
  key: string;
  /** 分区标题 */
  label: string;
  /** 该分区对应的多棵树（多根） */
  roots: TreeNode[];
}

export interface TreeSelectFilterProps {
  /** 树形数据（与 sections 二选一；同时传时优先用 sections） */
  nodes?: TreeNode[];
  /** 分区数据（部门 + 自定义分组等多分区场景；优先级高于 nodes） */
  sections?: TreeSelectSection[];
  /** 当前选中的节点 id，空字符串表示选中"全部"选项 */
  value: string;
  /** 确认选择回调 */
  onChange: (value: string) => void;
  /** "全部"选项的展示文字（默认 "全部"） */
  allLabel?: string;
  /** 是否显示搜索框（默认 true） */
  showSearch?: boolean;
  /** 搜索框 placeholder（默认 "搜索"） */
  searchPlaceholder?: string;
  /** 触发器宽度（默认 160px） */
  triggerWidth?: number;
  /** 面板宽度（默认 280px） */
  panelWidth?: number;
  /** 面板对齐方式 */
  align?: "start" | "center" | "end";
}

// ─── 树节点组件 ──────────────────────────────────────────────────────────────

function TreeNodeItem({
  node,
  level,
  selected,
  expanded,
  onToggle,
  onSelect,
  searchQuery,
}: {
  node: TreeNode;
  level: number;
  selected: string;
  expanded: Set<string>;
  onToggle: (id: string) => void;
  onSelect: (id: string) => void;
  searchQuery: string;
}) {
  const hasChildren = node.children && node.children.length > 0;
  const isExpanded = expanded.has(node.id);
  const isSelected = selected === node.id;

  // 搜索过滤
  const isVisible = useMemo(() => {
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    const matchNode = (n: TreeNode): boolean =>
      n.name.toLowerCase().includes(q) || (n.children?.some(matchNode) ?? false);
    return matchNode(node);
  }, [node, searchQuery]);

  if (!isVisible) return null;

  return (
    <div className="space-y-0.5">
      <div
        className={`relative flex items-center gap-2 h-8 px-3 rounded-[6px] cursor-pointer transition-colors text-sm ${
          isSelected ? "bg-[var(--bg-brand-selected)] text-blue-500 font-medium" : "font-normal text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)]"
        }`}
        style={{ paddingLeft: `${level * 16 + 12}px` }}
        onClick={() => onSelect(node.id)}
      >
        {hasChildren ? (
          <button
            className="w-4 h-4 flex items-center justify-center flex-shrink-0"
            onClick={(e) => { e.stopPropagation(); onToggle(node.id); }}
          >
            {isExpanded ? (
              <ChevronDown className="w-3.5 h-3.5 text-gray-400" />
            ) : (
              <ChevronRight className="w-3.5 h-3.5 text-gray-400" />
            )}
          </button>
        ) : (
          <span className="w-4 h-4 flex-shrink-0" />
        )}
        <span className="truncate flex-1">{node.name}</span>
        {isSelected && (
          <span className="absolute right-3 flex size-3.5 items-center justify-center">
            <Check className="size-4 text-blue-500" />
          </span>
        )}
      </div>
      {hasChildren && isExpanded && node.children!.map((child) => (
        <TreeNodeItem
          key={child.id}
          node={child}
          level={level + 1}
          selected={selected}
          expanded={expanded}
          onToggle={onToggle}
          onSelect={onSelect}
          searchQuery={searchQuery}
        />
      ))}
    </div>
  );
}

// ─── 主组件 ──────────────────────────────────────────────────────────────────

export function TreeSelectFilter({
  nodes,
  sections,
  value,
  onChange,
  allLabel = "全部",
  showSearch = true,
  searchPlaceholder = "搜索",
  triggerWidth = 160,
  panelWidth = 280,
  align = "start",
}: TreeSelectFilterProps) {
  // 统一规整为分区结构：未传 sections 时把 nodes 包装为单个无标题分区
  const resolvedSections = useMemo<TreeSelectSection[]>(() => {
    if (sections && sections.length > 0) return sections;
    return [{ key: "__default", label: "", roots: nodes ?? [] }];
  }, [sections, nodes]);

  // 是否多分区（需要展示分区标题）
  const isMultiSection = !!sections && sections.length > 0;

  // 所有根节点（跨分区），用于查找 / 默认展开
  const allRoots = useMemo(
    () => resolvedSections.flatMap((s) => s.roots),
    [resolvedSections],
  );

  const [open, setOpen] = useState(false);
  const [tempValue, setTempValue] = useState(value);
  const [searchQuery, setSearchQuery] = useState("");
  const [expanded, setExpanded] = useState<Set<string>>(() => {
    const first = allRoots[0];
    return first ? new Set([first.id]) : new Set();
  });

  // 打开时同步外部值
  useEffect(() => {
    if (open) {
      setTempValue(value);
      setSearchQuery("");
    }
  }, [open, value]);

  const toggleExpand = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const handleConfirm = () => {
    onChange(tempValue);
    setOpen(false);
  };

  const handleCancel = () => {
    setTempValue(value);
    setOpen(false);
  };

  // 查找节点（递归）
  const findNode = (list: TreeNode[], id: string): TreeNode | undefined => {
    for (const n of list) {
      if (n.id === id) return n;
      if (n.children) {
        const found = findNode(n.children, id);
        if (found) return found;
      }
    }
    return undefined;
  };

  const selectedNode = tempValue ? findNode(allRoots, tempValue) : undefined;
  const triggerNode = value ? findNode(allRoots, value) : undefined;
  const pathParts = selectedNode?.path?.split("/").filter(Boolean) || [];

  // 触发器展示文字
  const triggerLabel = triggerNode?.name || allLabel;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          role="combobox"
          aria-expanded={open}
          data-state={open ? "open" : "closed"}
          className="flex items-center justify-between gap-2 border border-border bg-white px-3 py-[5px] text-sm font-normal whitespace-nowrap transition-colors outline-none rounded-[4px] h-9 hover:border-blue-500 data-[state=open]:border-blue-500 text-[var(--text-title)]"
          style={{ width: triggerWidth }}
        >
          <span className="truncate">{triggerLabel}</span>
          <ChevronDown className="size-4 text-gray-500 shrink-0 transition-transform duration-200 [[data-state=open]>&]:rotate-180" />
        </button>
      </PopoverTrigger>
      <PopoverContent
        className="p-0 rounded-[4px] shadow-[var(--shadow-popover)] border-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95"
        style={{ width: panelWidth }}
        align={align}
        sideOffset={4}
      >
        {/* 搜索框 */}
        {showSearch && (
          <div className="p-2 pb-0">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)]" />
              <Input
                className="h-8 pl-8 text-sm"
                placeholder={searchPlaceholder}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
          </div>
        )}

        {/* 选项列表 */}
        <div className="max-h-[280px] overflow-y-auto p-2 space-y-0.5">
          {/* "全部"选项 */}
          {!searchQuery.trim() && (
            <div
              className={`relative flex w-full cursor-pointer items-center gap-2 rounded-[6px] h-8 px-3 py-[9px] text-sm transition-colors ${
                tempValue === "" ? "bg-[var(--bg-brand-selected)] text-blue-500 font-medium" : "font-normal text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)]"
              }`}
              onClick={() => setTempValue("")}
            >
              <span className="flex-1">{allLabel}</span>
              {tempValue === "" && (
                <span className="absolute right-3 flex size-3.5 items-center justify-center">
                  <Check className="size-4 text-blue-500" />
                </span>
              )}
            </div>
          )}
          {/* 分区 + 树形节点 */}
          {resolvedSections.map((section) =>
            section.roots.length === 0 ? null : (
              <div key={section.key} className={isMultiSection ? "mt-1" : ""}>
                {isMultiSection && section.label && (
                  <div className="px-3 py-1 text-[10px] font-medium text-[var(--text-weak)] uppercase tracking-wider">
                    {section.label}
                  </div>
                )}
                {section.roots.map((node) => (
                  <TreeNodeItem
                    key={node.id}
                    node={node}
                    level={0}
                    selected={tempValue}
                    expanded={expanded}
                    onToggle={toggleExpand}
                    onSelect={setTempValue}
                    searchQuery={searchQuery}
                  />
                ))}
              </div>
            ),
          )}
        </div>

        {/* 底部操作栏 */}
        <div className="mx-2 border-t border-[#EAEEF4] py-2 flex items-center justify-between gap-2">
          <MetaText className="flex-1 min-w-0 truncate">
            {tempValue === "" ? allLabel : selectedNode ? selectedNode.name : "未选择"}
          </MetaText>
          <div className="flex items-center gap-1.5 shrink-0">
            <Button variant="claw-outline" size="sm" className="text-xs h-7 px-2" onClick={handleCancel}>取消</Button>
            <Button variant="dialog-confirm" size="sm" className="text-xs h-7 px-3" onClick={handleConfirm}>确认</Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
