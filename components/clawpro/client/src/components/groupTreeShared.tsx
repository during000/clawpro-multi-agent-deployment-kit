/**
 * groupTreeShared - GroupMultiFilter / GroupSingleFilter / ScopeSelect 等
 * 共用的「组织树」内核（数据结构、工具函数、单节点 UI）。
 *
 * 拆分动机：多种筛选/选择器外层容器不同，但内部树形交互接近，抽到这里避免重复实现。
 */
import { Checkbox } from "@/components/ui/checkbox";
import { ChevronDown, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

// ──────────────────────────────────────────────
// 数据结构
// ──────────────────────────────────────────────

/** 通用树节点（同时兼容 DepartmentNode 与 UserGroup 展开后的结构） */
export interface TreeNodeData {
  id: string;
  name: string;
  children?: TreeNodeData[];
}

/** 一个分区（部门 / 自定义组织），可包含多棵树 */
export interface FilterSection {
  /** 分区唯一 key（例如 "dept" / "custom"），用于 React key 与内部区分 */
  key: string;
  /** 分区标题 */
  label: string;
  /** 该分区对应的多棵树（多根） */
  roots: TreeNodeData[];
}

// ──────────────────────────────────────────────
// 工具函数
// ──────────────────────────────────────────────

/** 自身 + 所有后代 id */
export function collectDescendantIds(node: TreeNodeData): string[] {
  const ids: string[] = [node.id];
  node.children?.forEach((c) => ids.push(...collectDescendantIds(c)));
  return ids;
}

/** 跨所有分区查找节点 */
export function findNode(sections: FilterSection[], id: string): TreeNodeData | undefined {
  const walk = (nodes: TreeNodeData[]): TreeNodeData | undefined => {
    for (const n of nodes) {
      if (n.id === id) return n;
      if (n.children) {
        const hit = walk(n.children);
        if (hit) return hit;
      }
    }
    return undefined;
  };
  for (const section of sections) {
    const hit = walk(section.roots);
    if (hit) return hit;
  }
  return undefined;
}

export type CheckState = "checked" | "unchecked" | "indeterminate";

/**
 * 计算节点勾选态。
 * 注意：indeterminate 仅看节点的「可勾选」子树，否则会因为 disabled 子节点导致永远 indeterminate。
 * 所以传入 isDisabled 用于过滤；当所有子节点都 disabled 时按 unchecked 处理。
 */
export function getCheckState(
  node: TreeNodeData,
  selected: Set<string>,
  isDisabled?: (id: string) => boolean,
): CheckState {
  if (selected.has(node.id)) return "checked";
  if (!node.children || node.children.length === 0) return "unchecked";
  // 仅考虑可勾选的子节点
  const enabledChildren = isDisabled
    ? node.children.filter((c) => !isDisabled(c.id) || hasEnabledDescendant(c, isDisabled))
    : node.children;
  if (enabledChildren.length === 0) return "unchecked";
  let hasChecked = false;
  let hasUnchecked = false;
  for (const child of enabledChildren) {
    const s = getCheckState(child, selected, isDisabled);
    if (s === "indeterminate") return "indeterminate";
    if (s === "checked") hasChecked = true;
    else hasUnchecked = true;
    if (hasChecked && hasUnchecked) return "indeterminate";
  }
  if (hasChecked && !hasUnchecked) return "checked";
  return "unchecked";
}

function hasEnabledDescendant(
  node: TreeNodeData,
  isDisabled: (id: string) => boolean,
): boolean {
  if (!isDisabled(node.id)) return true;
  return !!node.children?.some((c) => hasEnabledDescendant(c, isDisabled));
}

/** 把"平铺打勾集"规整为"最小覆盖集"（子孙全勾则只留父节点）。跨分区合并。 */
export function normalizeSelection(
  sections: FilterSection[],
  selected: Set<string>,
): string[] {
  const result = new Set<string>();
  const dfs = (node: TreeNodeData): boolean => {
    if (!node.children || node.children.length === 0) {
      if (selected.has(node.id)) {
        result.add(node.id);
        return true;
      }
      return false;
    }
    let allChildrenChecked = true;
    const childIds: string[] = [];
    for (const child of node.children) {
      const c = dfs(child);
      if (!c) allChildrenChecked = false;
      else childIds.push(...collectDescendantIds(child));
    }
    if (allChildrenChecked) {
      childIds.forEach((id) => result.delete(id));
      result.add(node.id);
      return true;
    }
    return false;
  };
  sections.forEach((s) => s.roots.forEach(dfs));
  return Array.from(result);
}

/** 把最小覆盖集展开为全量 id（含所有后代）。用于面板内初始勾选 & 外部数据筛选。 */
export function expandSelection(sections: FilterSection[], value: string[]): Set<string> {
  const set = new Set<string>();
  value.forEach((id) => {
    const node = findNode(sections, id);
    if (node) collectDescendantIds(node).forEach((d) => set.add(d));
  });
  return set;
}

/** 树过滤：保留命中关键字的节点及其祖先 */
export function filterRoots(roots: TreeNodeData[], keyword: string): TreeNodeData[] {
  if (!keyword) return roots;
  const lower = keyword.toLowerCase();
  const walk = (list: TreeNodeData[]): TreeNodeData[] => {
    const out: TreeNodeData[] = [];
    for (const n of list) {
      const childMatched = n.children ? walk(n.children) : [];
      const selfMatched = n.name.toLowerCase().includes(lower);
      if (selfMatched || childMatched.length > 0) {
        out.push({
          ...n,
          children: childMatched.length > 0 ? childMatched : n.children,
        });
      }
    }
    return out;
  };
  return walk(roots);
}

// ──────────────────────────────────────────────
// 树节点 UI（共用）
// ──────────────────────────────────────────────

export interface TreeNodeProps {
  node: TreeNodeData;
  level: number;
  selected: Set<string>;
  expanded: Set<string>;
  onToggleSelect: (node: TreeNodeData) => void;
  onToggleExpand: (id: string) => void;
  /** 节点是否禁用（不能勾选/取消，但可见可展开） */
  isDisabled?: (id: string) => boolean;
  /** 节点右侧附加文本（如 "未开启" 标注） */
  renderSuffix?: (node: TreeNodeData) => React.ReactNode;
}

export function TreeNode({
  node,
  level,
  selected,
  expanded,
  onToggleSelect,
  onToggleExpand,
  isDisabled,
  renderSuffix,
}: TreeNodeProps) {
  const hasChildren = !!node.children && node.children.length > 0;
  const isExpanded = expanded.has(node.id);
  const state = getCheckState(node, selected, isDisabled);
  const disabled = isDisabled?.(node.id) ?? false;

  return (
    <div>
      <div
        className={cn(
          "flex items-center gap-1 py-1 px-2 rounded-md",
          disabled
            ? "cursor-not-allowed opacity-50"
            : "hover:bg-gray-50 cursor-pointer",
        )}
        style={{ paddingLeft: `${level * 16 + 8}px` }}
        onClick={() => {
          if (!disabled) onToggleSelect(node);
        }}
      >
        {hasChildren ? (
          <button
            type="button"
            className="w-4 h-4 flex items-center justify-center flex-shrink-0"
            onClick={(e) => {
              e.stopPropagation();
              onToggleExpand(node.id);
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
        <Checkbox
          checked={state === "indeterminate" ? "indeterminate" : state === "checked"}
          disabled={disabled}
          className="flex-shrink-0"
          onClick={(e) => e.stopPropagation()}
          onCheckedChange={() => {
            if (!disabled) onToggleSelect(node);
          }}
        />
        <span className="text-sm text-gray-700 truncate flex-1">{node.name}</span>
        {renderSuffix?.(node)}
      </div>
      {hasChildren && isExpanded &&
        node.children!.map((child) => (
          <TreeNode
            key={child.id}
            node={child}
            level={level + 1}
            selected={selected}
            expanded={expanded}
            onToggleSelect={onToggleSelect}
            onToggleExpand={onToggleExpand}
            isDisabled={isDisabled}
            renderSuffix={renderSuffix}
          />
        ))}
    </div>
  );
}

// ──────────────────────────────────────────────
// 默认初始展开计算
// ──────────────────────────────────────────────

/**
 * 计算面板打开时默认展开的节点集合：
 * - 所有分区的根节点
 * - 所有已勾选节点的祖先链
 */
export function computeInitialExpanded(
  sections: FilterSection[],
  selected: Set<string>,
): Set<string> {
  const exp = new Set<string>();
  sections.forEach((section) => {
    section.roots.forEach((root) => exp.add(root.id));
  });
  const expandAncestors = (nodes: TreeNodeData[], path: string[]) => {
    for (const n of nodes) {
      const newPath = [...path, n.id];
      if (selected.has(n.id)) path.forEach((p) => exp.add(p));
      if (n.children) expandAncestors(n.children, newPath);
    }
  };
  sections.forEach((s) => expandAncestors(s.roots, []));
  return exp;
}
