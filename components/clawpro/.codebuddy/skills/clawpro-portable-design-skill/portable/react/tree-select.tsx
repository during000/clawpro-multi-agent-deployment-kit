/**
 * Portable TreeSelect — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 TreeSelect 时的可移植兜底实现（树形层级数据的「单选」下拉）。
 *  - 不依赖 @radix-ui / shadcn / lucide / Tailwind；样式由 portable/css/tree-select.css 提供。
 *  - 两种触发器变体（component-specs/tree-select.md §3）：
 *      button      → toolbar 筛选按钮（默认）
 *      filter-icon → 表格列头标题 + 漏斗图标（激活变品牌蓝）
 *  - 视觉规范（tree-select.md §4）：
 *      触发器 h-36 / 4px 圆角；面板 4px 圆角 + overlay 阴影，默认宽 280；
 *      节点行 h-32 / 4px 圆角 / 缩进 level*16+12；
 *      选中态浅蓝底 + 控件交互蓝字 + Check（不靠整行重色块）；
 *      底部 footer 已选回显 + 取消/确认（filter-icon instant 模式建议关 footer）。
 *  - 颜色/圆角已对齐 token：触发器 hover/open 描边、选中文字/Check、filter 激活图标统一
 *    用「控件交互蓝」--cp-control-accent（#355EF1，select hover·focus / 选中态语义，
 *    非品牌蓝 #1447E6）；圆角统一 4px。不要在调用处覆盖。
 *  - 不可信数据（节点 name / path）一律按纯文本渲染，不拼 HTML。
 *  - 大数据量（上千节点）请在宿主仓接入虚拟滚动，本兜底为非虚拟实现。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/tree-select.css";
 *
 * 用法：
 *   // button 变体（toolbar 筛选）
 *   <PortableTreeSelect nodes={deptTree} value={dept} onChange={setDept} allLabel="全部部门" />
 *
 *   // filter-icon 变体（表头列筛选，点即生效）
 *   <PortableTreeSelect
 *     triggerVariant="filter-icon"
 *     title="部门"
 *     nodes={deptTree}
 *     value={dept}
 *     onChange={setDept}
 *     commitMode="instant"
 *     showFooter={false}
 *   />
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export interface PortableTreeSelectNode {
  id: string;
  name: string;
  children?: PortableTreeSelectNode[];
  /** 节点路径（仅 button 变体用于底部回显，格式如 "A公司/技术部/前端组"） */
  path?: string;
}

/** 分区（部门 / 自定义分组等多分区单选场景，仅 button 变体支持） */
export interface PortableTreeSelectSection {
  key: string;
  label: string;
  roots: PortableTreeSelectNode[];
}

export type PortableTreeSelectAlign = "start" | "center" | "end";

interface CommonProps {
  /** 树形数据（与 sections 二选一） */
  nodes: PortableTreeSelectNode[];
  /** 当前选中节点 id（""=全部） */
  value: string;
  onChange?: (value: string) => void;
  /** "全部"选项文案（默认 "全部"） */
  allLabel?: string;
  searchPlaceholder?: string;
  /** 面板宽度（默认 280） */
  panelWidth?: number;
  align?: PortableTreeSelectAlign;
  className?: string;
}

interface ButtonVariantProps extends CommonProps {
  triggerVariant?: "button";
  /** 触发器宽度（默认 160） */
  triggerWidth?: number;
  /** 多分区单选（优先级高于 nodes） */
  sections?: PortableTreeSelectSection[];
  /** 是否显示搜索框（默认 true） */
  showSearch?: boolean;
}

interface FilterIconVariantProps extends CommonProps {
  triggerVariant: "filter-icon";
  /** 列标题（必填） */
  title: string;
  /**
   * 提交模式（默认 "confirm"）：
   * - "confirm"：点选项只更新临时态，点"确认"才生效
   * - "instant"：点选项立即生效并关闭面板（建议同时 showFooter={false}）
   */
  commitMode?: "confirm" | "instant";
  showSearch?: boolean;
  showFooter?: boolean;
}

export type PortableTreeSelectProps = ButtonVariantProps | FilterIconVariantProps;

// ─── 图标 ────────────────────────────────────────────────────────────────────

const ChevronDownIcon = () => (
  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M4 6l4 4 4-4" />
  </svg>
);

const ChevronRightIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <polyline points="9 18 15 12 9 6" />
  </svg>
);

const CheckIcon = () => (
  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M13.5 4L6 11.5L2.5 8" />
  </svg>
);

const SearchIcon = () => (
  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" aria-hidden="true">
    <circle cx="6.5" cy="6.5" r="5" />
    <path d="M10 10L14 14" />
  </svg>
);

const FilterIcon = () => (
  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M2 3h12l-4.5 5.5V13l-3 1.5V8.5L2 3z" />
  </svg>
);

// ─── 工具 ────────────────────────────────────────────────────────────────────

function findNode(
  list: PortableTreeSelectNode[],
  id: string
): PortableTreeSelectNode | undefined {
  for (const n of list) {
    if (n.id === id) return n;
    if (n.children) {
      const found = findNode(n.children, id);
      if (found) return found;
    }
  }
  return undefined;
}

function matchNode(node: PortableTreeSelectNode, q: string): boolean {
  return (
    node.name.toLowerCase().includes(q) ||
    (node.children?.some((c) => matchNode(c, q)) ?? false)
  );
}

// ─── 节点行 ──────────────────────────────────────────────────────────────────

function TreeRow({
  node,
  level,
  selected,
  expanded,
  onToggle,
  onSelect,
  searchQuery,
  isFlat,
}: {
  node: PortableTreeSelectNode;
  level: number;
  selected: string;
  expanded: Set<string>;
  onToggle: (id: string) => void;
  onSelect: (id: string) => void;
  searchQuery: string;
  isFlat?: boolean;
}) {
  const hasChildren = !!node.children?.length;
  const isExpanded = expanded.has(node.id);
  const isSelected = selected === node.id;

  const q = searchQuery.trim().toLowerCase();
  if (q && !matchNode(node, q)) return null;

  const rowCls = [
    "cp-tree-select__row",
    isSelected && "cp-tree-select__row--selected",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className="cp-tree-select__node">
      <div
        role="option"
        aria-selected={isSelected}
        className={rowCls}
        style={isFlat ? undefined : { paddingLeft: level * 16 + 12 }}
        onClick={() => onSelect(node.id)}
      >
        {!isFlat &&
          (hasChildren ? (
            <button
              type="button"
              className={[
                "cp-tree-select__arrow",
                isExpanded && "cp-tree-select__arrow--open",
              ]
                .filter(Boolean)
                .join(" ")}
              onClick={(e) => {
                e.stopPropagation();
                onToggle(node.id);
              }}
              aria-label={isExpanded ? "折叠" : "展开"}
            >
              <ChevronRightIcon />
            </button>
          ) : (
            <span className="cp-tree-select__arrow cp-tree-select__arrow--placeholder" />
          ))}
        <span className="cp-tree-select__label">{node.name}</span>
        {isSelected && (
          <span className="cp-tree-select__check">
            <CheckIcon />
          </span>
        )}
      </div>
      {hasChildren && isExpanded && (
        <div className="cp-tree-select__children">
          {node.children!.map((child) => (
            <TreeRow
              key={child.id}
              node={child}
              level={level + 1}
              selected={selected}
              expanded={expanded}
              onToggle={onToggle}
              onSelect={onSelect}
              searchQuery={searchQuery}
              isFlat={isFlat}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// ─── 主组件 ──────────────────────────────────────────────────────────────────

export function PortableTreeSelect(props: PortableTreeSelectProps) {
  const {
    nodes,
    value,
    onChange,
    allLabel = "全部",
    searchPlaceholder,
    panelWidth = 280,
    align = "start",
    className,
  } = props;

  const isFilterIcon = props.triggerVariant === "filter-icon";
  const commitMode = isFilterIcon ? props.commitMode ?? "confirm" : "confirm";
  const showSearch = props.showSearch ?? true;
  const showFooter = isFilterIcon ? props.showFooter ?? true : true;

  // button 变体的分区数据
  const sections: PortableTreeSelectSection[] = React.useMemo(() => {
    if (!isFilterIcon && (props as ButtonVariantProps).sections?.length) {
      return (props as ButtonVariantProps).sections!;
    }
    return [{ key: "__default", label: "", roots: nodes ?? [] }];
  }, [isFilterIcon, props, nodes]);

  const isMultiSection =
    !isFilterIcon && !!(props as ButtonVariantProps).sections?.length;

  const allRoots = React.useMemo(
    () => sections.flatMap((s) => s.roots),
    [sections]
  );

  // filter-icon 扁平模式：所有顶层节点均无 children
  const isFlat = React.useMemo(
    () =>
      isFilterIcon &&
      nodes.every((n) => !n.children || n.children.length === 0),
    [isFilterIcon, nodes]
  );

  const [open, setOpen] = React.useState(false);
  const [tempValue, setTempValue] = React.useState(value);
  const [searchQuery, setSearchQuery] = React.useState("");
  const [expanded, setExpanded] = React.useState<Set<string>>(() => {
    const first = (isFilterIcon ? nodes[0] : allRoots[0]);
    return first ? new Set([first.id]) : new Set();
  });

  const wrapRef = React.useRef<HTMLDivElement>(null);

  // 打开时同步外部值
  React.useEffect(() => {
    if (open) {
      setTempValue(value);
      setSearchQuery("");
    }
  }, [open, value]);

  // 点击外部关闭
  React.useEffect(() => {
    if (!open) return;
    const onDocClick = (e: MouseEvent) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", onDocClick);
    return () => document.removeEventListener("mousedown", onDocClick);
  }, [open]);

  const toggleExpand = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleSelect = (id: string) => {
    if (isFilterIcon && commitMode === "instant") {
      onChange?.(id);
      setOpen(false);
    } else {
      setTempValue(id);
    }
  };

  const handleConfirm = () => {
    onChange?.(tempValue);
    setOpen(false);
  };

  const handleCancel = () => {
    setTempValue(value);
    setOpen(false);
  };

  const selectedNode = tempValue ? findNode(allRoots, tempValue) : undefined;
  const triggerNode = value ? findNode(allRoots, value) : undefined;

  const footerLabel =
    tempValue === "" ? allLabel : selectedNode ? selectedNode.name : "未选择";

  const panelCls = [
    "cp-tree-select__panel",
    `cp-tree-select__panel--${align}`,
  ].join(" ");

  const panel = open ? (
    <div className={panelCls} style={{ width: panelWidth }} role="listbox">
      {showSearch && (
        <div className="cp-tree-select__search">
          <div className="cp-tree-select__search-box">
            <span className="cp-tree-select__search-icon">
              <SearchIcon />
            </span>
            <input
              type="text"
              className="cp-tree-select__search-input"
              placeholder={
                searchPlaceholder ??
                (isFilterIcon ? `搜索${props.title}` : "搜索")
              }
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
        </div>
      )}

      <div
        className={[
          "cp-tree-select__list",
          isFilterIcon
            ? "cp-tree-select__list--filter"
            : "cp-tree-select__list--button",
        ].join(" ")}
        onWheel={(e) => e.stopPropagation()}
      >
        {/* "全部"选项（搜索态隐藏） */}
        {!searchQuery.trim() && (
          <div
            role="option"
            aria-selected={tempValue === ""}
            className={[
              "cp-tree-select__row",
              tempValue === "" && "cp-tree-select__row--selected",
            ]
              .filter(Boolean)
              .join(" ")}
            onClick={() => handleSelect("")}
          >
            <span className="cp-tree-select__label">{allLabel}</span>
            {tempValue === "" && (
              <span className="cp-tree-select__check">
                <CheckIcon />
              </span>
            )}
          </div>
        )}

        {/* 树形节点 */}
        {sections.map((section) =>
          section.roots.length === 0 ? null : (
            <div key={section.key} className="cp-tree-select__node">
              {isMultiSection && section.label && (
                <div className="cp-tree-select__section-label">
                  {section.label}
                </div>
              )}
              {section.roots.map((node) => (
                <TreeRow
                  key={node.id}
                  node={node}
                  level={0}
                  selected={tempValue}
                  expanded={expanded}
                  onToggle={toggleExpand}
                  onSelect={handleSelect}
                  searchQuery={searchQuery}
                  isFlat={isFlat}
                />
              ))}
            </div>
          )
        )}
      </div>

      {showFooter && (
        <div className="cp-tree-select__footer">
          <span className="cp-tree-select__footer-label">{footerLabel}</span>
          <div className="cp-tree-select__footer-actions">
            <button
              type="button"
              className="cp-tree-select__btn cp-tree-select__btn--cancel"
              onClick={handleCancel}
            >
              取消
            </button>
            <button
              type="button"
              className="cp-tree-select__btn cp-tree-select__btn--confirm"
              onClick={handleConfirm}
            >
              确认
            </button>
          </div>
        </div>
      )}
    </div>
  ) : null;

  // ─── 渲染：触发器 + 浮层 ───
  if (isFilterIcon) {
    const isFiltered = value !== "";
    return (
      <div
        ref={wrapRef}
        className={["cp-tree-select", className].filter(Boolean).join(" ")}
      >
        <button
          type="button"
          className="cp-tree-select__filter-trigger"
          onClick={() => setOpen((v) => !v)}
        >
          <span>{props.title}</span>
          <span
            className={[
              "cp-tree-select__filter-icon",
              isFiltered && "cp-tree-select__filter-icon--active",
            ]
              .filter(Boolean)
              .join(" ")}
          >
            <FilterIcon />
          </span>
        </button>
        {panel}
      </div>
    );
  }

  // button 变体
  const triggerWidth = (props as ButtonVariantProps).triggerWidth ?? 160;
  const triggerLabel = triggerNode?.name || allLabel;
  return (
    <div
      ref={wrapRef}
      className={["cp-tree-select", className].filter(Boolean).join(" ")}
    >
      <button
        type="button"
        role="combobox"
        aria-expanded={open}
        className={[
          "cp-tree-select__trigger-btn",
          open && "cp-tree-select__trigger-btn--open",
        ]
          .filter(Boolean)
          .join(" ")}
        style={{ width: triggerWidth }}
        onClick={() => setOpen((v) => !v)}
      >
        <span className="cp-tree-select__trigger-label">{triggerLabel}</span>
        <span className="cp-tree-select__chevron">
          <ChevronDownIcon />
        </span>
      </button>
      {panel}
    </div>
  );
}

export default PortableTreeSelect;
