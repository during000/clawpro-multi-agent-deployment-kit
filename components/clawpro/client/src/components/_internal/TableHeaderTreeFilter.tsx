/**
 * TableHeaderTreeFilter - 表头筛选组件（树结构单选）
 *
 * 封装筛选图标 + 下拉面板，适用于表格列头树形单选筛选场景。
 * 功能：
 *   - 搜索框（无分割线）
 *   - "全部"选项 + 树形展开/折叠单选
 *   - 选中态 bg-[var(--bg-brand-selected)] + 蓝色文字 + Check 图标
 *   - 底部已选标签 + 取消/确认按钮
 *   - 默认展开第一个根节点
 *   - 滚动条仅交互时显示
 *   - 筛选激活时图标变蓝
 *
 * 视觉规范：
 *   - 选项行：h-8 px-3 rounded-[6px]
 *   - Footer：mx-2 border-t border-[#EAEEF4]
 */
import { useState, useMemo } from "react";
import { Filter, Search, ChevronDown, ChevronRight, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { MetaText } from "@/components/ui/Typography";

// ─── 类型定义 ────────────────────────────────────────────────────────────────

export interface TreeFilterNode {
  id: string;
  name: string;
  children?: TreeFilterNode[];
}

export interface TableHeaderTreeFilterProps {
  /** 列标题 */
  title: string;
  /** 树形数据 */
  nodes: TreeFilterNode[];
  /** 当前选中的节点 id，空字符串表示"全部" */
  value: string;
  /** 选中变化回调（confirm 模式：点确认才触发；instant 模式：点选项立即触发） */
  onConfirm: (value: string) => void;
  /** "全部"选项文字（默认 "全部{title}"） */
  allLabel?: string;
  /** 搜索框 placeholder（默认 "搜索{title}"） */
  searchPlaceholder?: string;
  /** 面板宽度（默认 280px） */
  panelWidth?: number;
  /** 面板对齐方式 */
  align?: "start" | "center" | "end";
  /**
   * 提交模式（默认 "confirm"）：
   * - "confirm"：底部带"取消/确认"按钮，点确认才生效
   * - "instant"：点选项立即生效并关闭面板
   */
  commitMode?: "confirm" | "instant";
  /** 是否显示搜索框（默认 true） */
  showSearch?: boolean;
  /** 是否显示底部 footer（默认 true，instant 模式下建议传 false） */
  showFooter?: boolean;
}

// ─── 树节点渲染 ──────────────────────────────────────────────────────────────

function TreeNodeRow({
  node,
  level,
  selected,
  expanded,
  onToggle,
  onSelect,
  searchQuery,
  isFlat,
}: {
  node: TreeFilterNode;
  level: number;
  selected: string;
  expanded: Set<string>;
  onToggle: (id: string) => void;
  onSelect: (id: string) => void;
  searchQuery: string;
  /** 是否扁平模式（整棵树无任何 children）。扁平时不渲染 chevron 占位、不做 level 缩进，与"全部"选项左对齐 */
  isFlat?: boolean;
}) {
  const hasChildren = node.children && node.children.length > 0;
  const isExpanded = expanded.has(node.id);
  const isSelected = selected === node.id;

  // 搜索过滤
  const isVisible = useMemo(() => {
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    const matchNode = (n: TreeFilterNode): boolean =>
      n.name.toLowerCase().includes(q) || (n.children?.some(matchNode) ?? false);
    return matchNode(node);
  }, [node, searchQuery]);

  if (!isVisible) return null;

  return (
    <div className="space-y-0.5">
      <div
        className={`relative flex items-center gap-2 h-8 px-3 rounded-[6px] cursor-pointer transition-colors text-sm ${
          isSelected ? "bg-[var(--bg-brand-selected)] text-blue-500 font-medium" : "text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)]"
        }`}
        // 扁平模式：与"全部"选项一致的 px-3 左内边距；树形模式：每层 16px 缩进 + 12px 基准内边距
        style={isFlat ? undefined : { paddingLeft: `${level * 16 + 12}px` }}
        onClick={() => onSelect(node.id)}
      >
        {/* 扁平模式不渲染 chevron 区域，让文字与"全部"选项左对齐 */}
        {!isFlat && (hasChildren ? (
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
        ))}
        <span className="truncate flex-1">{node.name}</span>
        {isSelected && (
          <span className="absolute right-3 flex size-3.5 items-center justify-center">
            <Check className="size-4 text-blue-500" />
          </span>
        )}
      </div>
      {hasChildren && isExpanded && node.children!.map((child) => (
        <TreeNodeRow
          key={child.id}
          node={child}
          level={level + 1}
          selected={selected}
          isFlat={isFlat}
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

export function TableHeaderTreeFilter({
  title,
  nodes,
  value,
  onConfirm,
  allLabel,
  searchPlaceholder,
  panelWidth = 280,
  align = "start",
  commitMode = "confirm",
  showSearch = true,
  showFooter = true,
}: TableHeaderTreeFilterProps) {
  const [open, setOpen] = useState(false);
  const [tempValue, setTempValue] = useState(value);
  const [searchQuery, setSearchQuery] = useState("");
  const [expanded, setExpanded] = useState<Set<string>>(() => {
    const first = nodes[0];
    return first ? new Set([first.id]) : new Set();
  });

  // 自动检测扁平模式：所有顶层节点均无 children → 视为扁平列表，不缩进、不渲染 chevron 占位
  const isFlat = useMemo(
    () => nodes.every((n) => !n.children || n.children.length === 0),
    [nodes]
  );

  const resolvedAllLabel = allLabel || `全部${title}`;
  const resolvedSearchPlaceholder = searchPlaceholder || `搜索${title}`;

  const isFiltered = value !== "";

  const handleOpenChange = (v: boolean) => {
    if (v) {
      setTempValue(value);
      setSearchQuery("");
    }
    setOpen(v);
  };

  const toggleExpand = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  // 获取选中节点名称
  const selectedLabel = useMemo(() => {
    if (tempValue === "") return resolvedAllLabel;
    const find = (list: TreeFilterNode[]): string | null => {
      for (const n of list) {
        if (n.id === tempValue) return n.name;
        if (n.children) {
          const found = find(n.children);
          if (found) return found;
        }
      }
      return null;
    };
    return find(nodes) || "";
  }, [tempValue, nodes, resolvedAllLabel]);

  const handleSelect = (id: string) => {
    if (commitMode === "instant") {
      onConfirm(id);
      setOpen(false);
    } else {
      setTempValue(id);
    }
  };

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <button className="flex items-center gap-1 group/filter">
          <span>{title}</span>
          <Filter className={`w-3.5 h-3.5 transition-colors ${isFiltered ? "text-[#355EF1]" : "text-[var(--text-weak)] group-hover/filter:text-[var(--text-muted)]"}`} />
        </button>
      </PopoverTrigger>
      <PopoverContent
        className="p-0 rounded-[4px] border-none shadow-[var(--shadow-popover)]"
        style={{ width: panelWidth }}
        align={align}
        side="bottom"
      >
        {/* 搜索框（可选） */}
        {showSearch && (
          <div className="p-2 pb-0">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)]" />
              <Input
                className="h-8 pl-8 text-sm"
                placeholder={resolvedSearchPlaceholder}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
          </div>
        )}
        {/* 列表区 */}
        <div
          className="max-h-[260px] overflow-y-auto p-2 space-y-0.5"
          style={{ scrollbarWidth: "thin", scrollbarColor: "transparent transparent" }}
          onMouseEnter={(e) => { (e.currentTarget as HTMLElement).style.scrollbarColor = "rgba(0,0,0,0.2) transparent"; }}
          onMouseLeave={(e) => { (e.currentTarget as HTMLElement).style.scrollbarColor = "transparent transparent"; }}
          onWheel={(e) => e.stopPropagation()}
        >
          {/* "全部"选项 */}
          {!searchQuery.trim() && (
            <div
              className={`relative flex items-center gap-2 h-8 px-3 rounded-[6px] cursor-pointer transition-colors text-sm ${
                tempValue === "" ? "bg-[var(--bg-brand-selected)] text-blue-500 font-medium" : "text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)]"
              }`}
              onClick={() => handleSelect("")}
            >
              <span className="flex-1">{resolvedAllLabel}</span>
              {tempValue === "" && (
                <span className="absolute right-3 flex size-3.5 items-center justify-center">
                  <Check className="size-4 text-blue-500" />
                </span>
              )}
            </div>
          )}
          {/* 树形节点 */}
          {nodes.map((node) => (
            <TreeNodeRow
              key={node.id}
              node={node}
              level={0}
              selected={tempValue}
              expanded={expanded}
              onToggle={toggleExpand}
              onSelect={handleSelect}
              isFlat={isFlat}
              searchQuery={searchQuery}
            />
          ))}
        </div>
        {/* Footer（可选：confirm 模式默认显示，instant 模式建议传 showFooter=false） */}
        {showFooter && (
          <div className="mx-2 border-t border-[#EAEEF4] py-2 flex items-center justify-between">
            <MetaText className="truncate">{selectedLabel}</MetaText>
            <div className="flex items-center gap-1.5 shrink-0">
              <Button variant="claw-outline" size="sm" className="text-xs h-7 px-2" onClick={() => { setTempValue(value); setOpen(false); }}>
                取消
              </Button>
              <Button variant="dialog-confirm" size="sm" className="text-xs h-7 px-3" onClick={() => { onConfirm(tempValue); setOpen(false); }}>
                确认
              </Button>
            </div>
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}

export default TableHeaderTreeFilter;
