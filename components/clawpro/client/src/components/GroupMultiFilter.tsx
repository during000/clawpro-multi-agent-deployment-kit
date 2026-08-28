/**
 * GroupMultiFilter - 组织多选筛选组件（Popover）
 *
 * 场景：
 *   - 顶部工具栏的「组织」/「开启范围」筛选器。
 *   - 多分区（部门 + 自定义组织），多选；未选 = 全部。
 *
 * 内部多选树交互复用 ./groupTreeShared。
 */
import { useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Input } from "@/components/ui/input";
import { ChevronDown, Search } from "lucide-react";
import { cn } from "@/lib/utils";
import {
  type FilterSection,
  type TreeNodeData,
  collectDescendantIds,
  computeInitialExpanded,
  expandSelection,
  filterRoots,
  findNode,
  getCheckState,
  normalizeSelection,
  TreeNode,
} from "./groupTreeShared";

// 重新导出，保持外部老引用兼容（OpsObservation / SessionManagement 仍从此处 import）
export type { FilterSection, TreeNodeData } from "./groupTreeShared";

export interface GroupMultiFilterProps {
  sections: FilterSection[];
  value: string[];
  onChange: (value: string[]) => void;
  /** 触发器宽度，默认 120px（对齐 Agent 列表） */
  triggerWidth?: number;
  placeholder?: string;
  searchPlaceholder?: string;
}

export function GroupMultiFilter({
  sections,
  value,
  onChange,
  triggerWidth = 120,
  placeholder = "全部组织",
  searchPlaceholder = "请输入组织名称",
}: GroupMultiFilterProps) {
  const [open, setOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [draftSelected, setDraftSelected] = useState<Set<string>>(() =>
    expandSelection(sections, value),
  );
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  // 打开面板时同步草稿 & 展开
  useEffect(() => {
    if (open) {
      const draft = expandSelection(sections, value);
      setDraftSelected(draft);
      setSearchQuery("");
      setExpanded(computeInitialExpanded(sections, draft));
    }
  }, [open, value, sections]);

  // Trigger 文案：首项名 + 多选时 +N
  const selectedNodes = useMemo(
    () =>
      value
        .map((id) => findNode(sections, id))
        .filter((n): n is TreeNodeData => !!n),
    [sections, value],
  );
  const firstSelectedName = selectedNodes[0]?.name ?? "";
  const extraCount = Math.max(0, selectedNodes.length - 1);

  // 底部"已选 N 项"实时计数
  const draftSelectedCount = useMemo(
    () => normalizeSelection(sections, draftSelected).length,
    [sections, draftSelected],
  );

  const filteredSections = useMemo(
    () =>
      sections.map((s) => ({
        ...s,
        roots: filterRoots(s.roots, searchQuery),
      })),
    [sections, searchQuery],
  );

  const toggleNodeSelect = (node: TreeNodeData) => {
    const next = new Set(draftSelected);
    const state = getCheckState(node, next);
    const descendants = collectDescendantIds(node);
    if (state === "checked") {
      descendants.forEach((id) => next.delete(id));
    } else {
      descendants.forEach((id) => next.add(id));
    }
    setDraftSelected(next);
  };

  const toggleExpand = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleClearAll = () => setDraftSelected(new Set());
  const handleConfirm = () => {
    onChange(normalizeSelection(sections, draftSelected));
    setOpen(false);
  };
  const handleCancel = () => {
    setDraftSelected(expandSelection(sections, value));
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          className={cn(
            "justify-between bg-white text-sm font-normal hover:bg-white",
            "data-[state=open]:border-ring data-[state=open]:ring-[3px] data-[state=open]:ring-ring/50",
            selectedNodes.length === 0 && "text-muted-foreground",
          )}
          style={{ width: triggerWidth, height: 36 }}
        >
          <span className="truncate flex-1 text-left">
            {selectedNodes.length === 0
              ? placeholder
              : extraCount > 0
                ? `${firstSelectedName} +${extraCount}`
                : firstSelectedName}
          </span>
          <ChevronDown
            className={cn(
              "w-3.5 h-3.5 ml-1 shrink-0 opacity-50 transition-transform duration-200",
              open && "rotate-180",
            )}
          />
        </Button>
      </PopoverTrigger>

      <PopoverContent className="w-[320px] p-0" align="start">
        <div className="p-2 border-b border-gray-100">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400" />
            <Input
              placeholder={searchPlaceholder}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-8 h-8 text-sm"
            />
          </div>
        </div>

        <div className="max-h-[320px] overflow-y-auto py-1">
          {filteredSections.every((s) => s.roots.length === 0) ? (
            <div className="text-center text-sm text-gray-400 py-6">未找到匹配的组织</div>
          ) : (
            filteredSections.map((section) =>
              section.roots.length > 0 ? (
                <div key={section.key} className="pt-2 pb-1">
                  <div className="px-3 pb-1 text-xs text-gray-400">{section.label}</div>
                  {section.roots.map((root) => (
                    <TreeNode
                      key={root.id}
                      node={root}
                      level={0}
                      selected={draftSelected}
                      expanded={expanded}
                      onToggleSelect={toggleNodeSelect}
                      onToggleExpand={toggleExpand}
                    />
                  ))}
                </div>
              ) : null,
            )
          )}
        </div>

        <div className="border-t border-gray-100 px-3 py-2 flex items-center justify-between gap-2">
          <span className="text-xs text-gray-500">
            {draftSelectedCount > 0 ? `已选 ${draftSelectedCount} 项` : "未选择"}
          </span>
          <div className="flex items-center gap-1.5">
            <Button
              variant="ghost"
              size="sm"
              className="text-xs text-gray-500 h-7 px-2"
              onClick={handleClearAll}
            >
              清空
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="text-xs text-gray-500 h-7 px-2"
              onClick={handleCancel}
            >
              取消
            </Button>
            <Button
              size="sm"
              className="text-xs h-7 px-3 text-white"
              style={{ background: "linear-gradient(135deg, #007AFF, #5856D6)" }}
              onClick={handleConfirm}
            >
              确认
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

/** 把最小覆盖集展开为"全量 id 集"（含所有后代），供数据侧按归属过滤 */
export function getGroupFilterIds(
  sections: FilterSection[],
  value: string[],
): Set<string> {
  return expandSelection(sections, value);
}
