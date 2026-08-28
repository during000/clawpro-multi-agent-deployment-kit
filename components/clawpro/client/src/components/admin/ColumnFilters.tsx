/**
 * 共享列头筛选组件
 * - DepartmentColumnFilter：表头部门筛选 popover 内容
 * - GroupColumnFilter：表头组织筛选 popover 内容（按 source 分桶）
 *
 * 抽自 OpenClawMonitor.tsx，便于在 MemberManagement / 其它管控页复用。
 * 行为/样式与原实现保持一致（搜索 + 树形 + 取消/确认）。
 */
import { useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Search, ChevronDown, ChevronRight, Check } from "lucide-react";
import { type DepartmentNode } from "@/lib/mockData";
import type { UserGroup } from "@/pages/admin/MemberManagement/types";
import { buildGroupTree, buildUnifiedGroupTree, type GroupTreeNode } from "@/pages/admin/MemberManagement/health";

// ─── 部门列头筛选面板 ─────────────────────────────────────────────────────
export function DepartmentColumnFilter({
  departments,
  value,
  onConfirm,
  onCancel,
}: {
  departments: DepartmentNode[];
  value: string;
  onConfirm: (v: string) => void;
  onCancel: () => void;
}) {
  const [tempValue, setTempValue] = useState(value);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [searchQuery, setSearchQuery] = useState("");

  useEffect(() => {
    setTempValue(value);
  }, [value]);

  const toggleExpand = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const isNodeVisible = (node: DepartmentNode): boolean => {
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    if (node.name.toLowerCase().includes(q)) return true;
    return (node.children || []).some(isNodeVisible);
  };

  const renderNode = (node: DepartmentNode, level: number) => {
    if (!isNodeVisible(node)) return null;
    const hasChildren = node.children && node.children.length > 0;
    const isExpanded = expanded.has(node.id);
    const isSelected = tempValue === node.id;
    return (
      <div key={node.id}>
        <div
          className={`flex items-center gap-1 py-1.5 px-2 mb-0.5 rounded-[4px] cursor-pointer transition-colors ${
            isSelected ? "bg-[var(--cp-brand-tint)] text-[var(--text-brand)]" : "text-[var(--text-secondary)] hover:bg-[var(--bg-grey-hover)]"
          }`}
          style={{ paddingLeft: `${level * 16 + 8}px` }}
          onClick={() => setTempValue(node.id)}
        >
          {hasChildren ? (
            <button
              className="w-4 h-4 flex items-center justify-center flex-shrink-0"
              onClick={(e) => {
                e.stopPropagation();
                toggleExpand(node.id);
              }}
            >
              {isExpanded ? (
                <ChevronDown className="w-3.5 h-3.5 text-[var(--text-weak)]" />
              ) : (
                <ChevronRight className="w-3.5 h-3.5 text-[var(--text-weak)]" />
              )}
            </button>
          ) : (
            <span className="w-4 h-4 flex items-center justify-center flex-shrink-0">
              <span className="w-1.5 h-1.5 rounded-full bg-[var(--text-weak)]" />
            </span>
          )}
          <span className={`text-sm truncate flex-1 ${isSelected ? "text-[var(--text-brand)] font-medium" : ""}`}>{node.name}</span>
          {isSelected && <Check className="w-4 h-4 ml-auto text-[var(--text-brand)] flex-shrink-0" />}
        </div>
        {hasChildren && isExpanded && node.children!.map((child) => renderNode(child, level + 1))}
      </div>
    );
  };

  return (
    <>
      <div className="px-3 pt-3 pb-2">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)]" />
          <Input
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="搜索部门"
            className="h-8 pl-8 pr-3 text-sm"
          />
        </div>
      </div>
      <div className="max-h-[280px] overflow-y-auto px-2 pb-2">
        <div
          className={`flex items-center gap-2 py-1.5 px-2 mb-0.5 rounded-[4px] cursor-pointer transition-colors ${
            tempValue === "" ? "bg-[var(--cp-brand-tint)]" : "hover:bg-[var(--bg-grey-hover)]"
          }`}
          onClick={() => setTempValue("")}
        >
          <span className={`text-sm flex-1 ${tempValue === "" ? "text-[var(--text-brand)] font-medium" : "text-[var(--text-secondary)]"}`}>全部部门</span>
          {tempValue === "" && <Check className="w-4 h-4 text-[var(--text-brand)] flex-shrink-0" />}
        </div>
        {departments.map((d) => renderNode(d, 0))}
      </div>
      <div className="border-t border-[var(--border)] px-3 py-2 flex items-center justify-end gap-1.5">
        <Button variant="claw-outline" size="claw-sm" className="h-7 px-2 text-xs" onClick={onCancel}>
          取消
        </Button>
        <Button
          variant="claw-primary"
          size="claw-sm"
          className="h-7 px-3 text-xs"
          onClick={() => onConfirm(tempValue)}
        >
          确认
        </Button>
      </div>
    </>
  );
}

// ─── 组织列头筛选树节点（递归） ──────────────────────────────────────────
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
          className={`flex items-center gap-1 py-1.5 px-2 mb-0.5 rounded-[4px] cursor-pointer transition-colors ${
            isSelected ? "bg-[var(--cp-brand-tint)] text-[var(--text-brand)]" : "text-[var(--text-secondary)] hover:bg-[var(--bg-grey-hover)]"
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
              <ChevronDown className="w-3.5 h-3.5 text-[var(--text-weak)]" />
            ) : (
              <ChevronRight className="w-3.5 h-3.5 text-[var(--text-weak)]" />
            )}
          </button>
        ) : (
          <span className="w-4 h-4 flex items-center justify-center flex-shrink-0">
            <span className="w-1.5 h-1.5 rounded-full bg-[var(--text-weak)]" />
          </span>
        )}
        <span className={`text-sm truncate flex-1 ${isSelected ? "text-[var(--text-brand)] font-medium" : ""}`}>{node.name}</span>
        {isSelected && <Check className="w-4 h-4 ml-auto text-[var(--text-brand)] flex-shrink-0" />}
      </div>
      {hasChildren && isExpanded &&
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

// ─── 组织列头筛选面板 ─────────────────────────────────────────────────────
export function GroupColumnFilter({
  groups,
  value,
  hasOneid,
  onConfirm,
  onCancel,
  specialOptions = [],
}: {
  groups: UserGroup[];
  value: string;
  hasOneid: boolean;
  onConfirm: (v: string) => void;
  onCancel: () => void;
  specialOptions?: Array<{ value: string; label: string; description?: string }>;
}) {
  const [tempValue, setTempValue] = useState(value);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [searchQuery, setSearchQuery] = useState("");

  useEffect(() => {
    setTempValue(value);
  }, [value]);

  const toggleExpand = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  // OneID 模式：部门 + 自定义组织合并为一棵树，自定义组织统一挂到 A 公司（根部门）下，不再分桶
  const trees = useMemo(
    () => (hasOneid ? buildUnifiedGroupTree(groups) : buildGroupTree(groups)),
    [groups, hasOneid],
  );

  const isNodeVisible = (node: GroupTreeNode): boolean => {
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    if (node.name.toLowerCase().includes(q)) return true;
    return node.children.some(isNodeVisible);
  };

  const normalizedSearch = searchQuery.trim().toLowerCase();
  const visibleSpecialOptions = specialOptions.filter((option) => {
    if (!normalizedSearch) return true;
    return option.label.toLowerCase().includes(normalizedSearch)
      || option.value.toLowerCase().includes(normalizedSearch)
      || (option.description ?? "").toLowerCase().includes(normalizedSearch);
  });

  return (
    <>
      <div className="px-3 pt-3 pb-2">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)]" />
          <Input
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="搜索组织"
            className="h-8 pl-8 pr-3 text-sm"
          />
        </div>
      </div>
      <div className="max-h-[280px] overflow-y-auto px-2 pb-2">
        <div
          className={`flex items-center gap-2 py-1.5 px-2 mb-0.5 rounded-[4px] cursor-pointer transition-colors ${
            tempValue === "" ? "bg-[var(--cp-brand-tint)]" : "hover:bg-[var(--bg-grey-hover)]"
          }`}
          onClick={() => setTempValue("")}
        >
          <span className={`text-sm flex-1 ${tempValue === "" ? "text-[var(--text-brand)] font-medium" : "text-[var(--text-secondary)]"}`}>全部组织</span>
          {tempValue === "" && <Check className="w-4 h-4 text-[var(--text-brand)] flex-shrink-0" />}
        </div>
        {visibleSpecialOptions.length > 0 && (
          <div className="border-b border-[var(--border)] pb-2 mb-2">
            {visibleSpecialOptions.map((option) => {
              const isSelected = tempValue === option.value;
              return (
                <div
                  key={option.value}
                  className={`flex items-start gap-2 py-1.5 px-2 mb-0.5 rounded-[4px] cursor-pointer transition-colors ${
                    isSelected ? "bg-[var(--cp-brand-tint)] text-[var(--text-brand)]" : "text-[var(--text-secondary)] hover:bg-[var(--bg-grey-hover)]"
                  }`}
                  onClick={() => setTempValue(option.value)}
                >
                  <span className="w-4 h-4 flex items-center justify-center flex-shrink-0 mt-0.5">
                    {isSelected && <Check className="w-4 h-4 text-[var(--text-brand)]" />}
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className={`block text-sm truncate ${isSelected ? "font-medium" : ""}`}>{option.label}</span>
                    {option.description && (
                      <span className="block text-xs text-[var(--text-muted)] truncate mt-0.5">{option.description}</span>
                    )}
                  </span>
                </div>
              );
            })}
          </div>
        )}
        {trees.map((root) =>
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
      <div className="border-t border-[var(--border)] px-3 py-2 flex items-center justify-end gap-1.5">
        <Button variant="claw-outline" size="claw-sm" className="h-7 px-2 text-xs" onClick={onCancel}>
          取消
        </Button>
        <Button
          variant="claw-primary"
          size="claw-sm"
          className="h-7 px-3 text-xs"
          onClick={() => onConfirm(tempValue)}
        >
          确认
        </Button>
      </div>
    </>
  );
}

/** 根据部门 id 在部门树中查找该节点及其所有子孙 id */
export function findDeptAndChildren(nodes: DepartmentNode[], targetId: string): string[] {
  if (!targetId) return [];
  const collect = (n: DepartmentNode): string[] => {
    const ids = [n.id];
    (n.children || []).forEach((c) => ids.push(...collect(c)));
    return ids;
  };
  for (const n of nodes) {
    if (n.id === targetId) return collect(n);
    const found = findDeptAndChildren(n.children || [], targetId);
    if (found.length > 0) return found;
  }
  return [];
}
