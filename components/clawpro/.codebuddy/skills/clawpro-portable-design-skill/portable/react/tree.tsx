/**
 * Portable Tree — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 Tree 时的可移植兜底实现（分组 / 文件树 / 目录导航）。
 *  - 不依赖 @radix-ui / shadcn / Tailwind；样式由 portable/css/tree.css 提供。
 *  - 视觉规范（component-specs/tree.md §3）：
 *      行高 32px；圆角 4px；缩进 8 + depth×16；行间距 4px；
 *      文字 #09090b；hover/选中底色统一 #f4f4f5（选中加 font-medium）；
 *      图标统一中灰 #71717a 16px；箭头 14px 展开 rotate-90；计数弱灰 #a1a1aa。
 *  - 图标统一弱色（不要多色 / emoji）；选中态靠行底色传达，不靠图标变色；
 *    hover/selected 用中性灰底（不要品牌蓝，避免与 Sidebar active 冲突）。
 *  - 大数据量（上千节点）请在宿主仓接入虚拟滚动，本兜底为非虚拟实现。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/tree.css";
 *
 * 用法：
 *   <PortableTree
 *     items={[{ id: "g1", label: "分组 A", count: 3, children: [{ id: "n1", label: "成员 1" }] }]}
 *     activeId={active}
 *     onSelect={setActive}
 *     defaultExpandAll
 *   />
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export interface PortableTreeItem {
  id: string;
  label: React.ReactNode;
  /** 末尾弱色计数 */
  count?: number;
  /** 自定义 16px 图标（currentColor → #71717a） */
  icon?: React.ReactNode;
  disabled?: boolean;
  children?: PortableTreeItem[];
}

export interface PortableTreeProps {
  items: PortableTreeItem[];
  activeId?: string;
  onSelect?: (id: string, item: PortableTreeItem) => void;
  /** 默认展开全部分支 */
  defaultExpandAll?: boolean;
  className?: string;
}

const Chevron = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <polyline points="9 18 15 12 9 6" />
  </svg>
);

function TreeNode({
  item,
  depth,
  activeId,
  onSelect,
  defaultExpandAll,
}: {
  item: PortableTreeItem;
  depth: number;
  activeId?: string;
  onSelect?: (id: string, item: PortableTreeItem) => void;
  defaultExpandAll?: boolean;
}) {
  const hasChildren = !!item.children?.length;
  const [open, setOpen] = React.useState(!!defaultExpandAll && hasChildren);
  const isActive = activeId === item.id;

  const rowCls = [
    "cp-tree__row",
    isActive && "cp-tree__row--active",
    item.disabled && "cp-tree__row--disabled",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className="cp-tree__node">
      <div
        role="treeitem"
        aria-expanded={hasChildren ? open : undefined}
        aria-selected={isActive || undefined}
        aria-disabled={item.disabled || undefined}
        tabIndex={item.disabled ? -1 : 0}
        className={rowCls}
        style={{ paddingLeft: 8 + depth * 16 }}
        onClick={() => {
          if (item.disabled) return;
          onSelect?.(item.id, item);
          if (hasChildren) setOpen((v) => !v);
        }}
        onKeyDown={(e) => {
          if (item.disabled) return;
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            onSelect?.(item.id, item);
            if (hasChildren) setOpen((v) => !v);
          }
        }}
      >
        <span
          className={[
            "cp-tree__arrow",
            hasChildren ? open && "cp-tree__arrow--open" : "cp-tree__arrow--placeholder",
          ]
            .filter(Boolean)
            .join(" ")}
        >
          {hasChildren ? <Chevron /> : null}
        </span>
        {item.icon != null && <span className="cp-tree__icon">{item.icon}</span>}
        <span className="cp-tree__label">{item.label}</span>
        {item.count != null && <span className="cp-tree__count">({item.count})</span>}
      </div>
      {hasChildren && open && (
        <div className="cp-tree__children" role="group">
          {item.children!.map((child) => (
            <TreeNode
              key={child.id}
              item={child}
              depth={depth + 1}
              activeId={activeId}
              onSelect={onSelect}
              defaultExpandAll={defaultExpandAll}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export function PortableTree({
  items,
  activeId,
  onSelect,
  defaultExpandAll,
  className = "",
}: PortableTreeProps) {
  const merged = ["cp-tree", className].filter(Boolean).join(" ");
  return (
    <div role="tree" className={merged}>
      {items.map((item) => (
        <TreeNode
          key={item.id}
          item={item}
          depth={0}
          activeId={activeId}
          onSelect={onSelect}
          defaultExpandAll={defaultExpandAll}
        />
      ))}
    </div>
  );
}
