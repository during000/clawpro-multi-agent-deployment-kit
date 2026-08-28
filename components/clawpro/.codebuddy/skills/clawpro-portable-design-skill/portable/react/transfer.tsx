/**
 * Portable Transfer — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 Transfer 时的可移植兜底实现（弹窗内双面板穿梭）。
 *  - 不依赖 antd / shadcn / Tailwind；样式由 portable/css/transfer.css 提供。
 *  - 视觉规范（component-specs/transfer.md §3）：
 *      等宽双面板（flex-1 / min-w-0）；外壳 4px 圆角 + var(--cp-border) 描边；
 *      面板内分割线 #f0f0f0；头部底色 var(--cp-bg-subtle)；
 *      标题 14px medium emphasis；计数 12px muted；空态 12px weak；
 *      行高 40px / 字号 12px（Table compact）；simple small 分页。
 *  - 三种移动模式：
 *      instant（默认）：左侧勾选立即搬右；右侧行末 X 移除；右侧 header「清空选择」；
 *      batch：左右各自勾选，中间渲染 › / ‹ 按钮做批量穿梭；
 *      oneWay：右侧不可移回（instant 隐藏行末 X；batch 隐藏 ‹）。
 *  - 空态文案区分 4 种：左侧无数据 / 左侧搜不到 / 右侧空 / 调用方覆盖。
 *  - 禁用项 60% 透明 + cursor-not-allowed；如需原因请在宿主仓外侧包 Tooltip。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/transfer.css";
 *
 * 用法：
 *   <PortableTransfer
 *     dataSource={items}
 *     targetKeys={selected}
 *     onChange={setSelected}
 *     showSearch
 *     titles={["全部资产", "已选资产"]}
 *     renderItem={(it) => it.name}
 *   />
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export interface PortableTransferItem {
  key: string;
  disabled?: boolean;
  [k: string]: unknown;
}

export interface PortableTransferProps<T extends PortableTransferItem = PortableTransferItem> {
  /** 全集（含已选项） */
  dataSource: T[];
  /** 受控：当前已选 keys */
  targetKeys: string[];
  /** keys 变更回调 */
  onChange: (nextKeys: string[], direction: "right" | "left", moveKeys: string[]) => void;
  /** 单行渲染（默认取 item.name / item.label / item.key） */
  renderItem?: (item: T) => React.ReactNode;
  /** 面板标题，默认 ['全部', '已选'] */
  titles?: [React.ReactNode, React.ReactNode];
  /** 移动模式，默认 instant */
  mode?: "instant" | "batch";
  /** 右侧不可移回 */
  oneWay?: boolean;
  /** 显示搜索框 */
  showSearch?: boolean;
  /** 搜索占位（单值或 [左, 右]） */
  searchPlaceholder?: string | [string, string];
  /** 自定义搜索过滤（默认对 renderItem 文本不区分大小写包含匹配） */
  filterOption?: (input: string, item: T) => boolean;
  /** 分页页大小；传 0 / false 关闭分页 */
  pageSize?: number;
  /** 单侧 body 高度，默认 330 */
  height?: number;
  /** 空态文案覆盖 */
  notFoundContent?: React.ReactNode;
  className?: string;
}

const SearchIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <circle cx="11" cy="11" r="8" />
    <line x1="21" y1="21" x2="16.65" y2="16.65" />
  </svg>
);

const CloseIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <line x1="18" y1="6" x2="6" y2="18" />
    <line x1="6" y1="6" x2="18" y2="18" />
  </svg>
);

const ChevronRight = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <polyline points="9 18 15 12 9 6" />
  </svg>
);

const ChevronLeft = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <polyline points="15 18 9 12 15 6" />
  </svg>
);

function defaultLabel<T extends PortableTransferItem>(item: T): React.ReactNode {
  return (item.name as React.ReactNode) ?? (item.label as React.ReactNode) ?? item.key;
}

function labelText<T extends PortableTransferItem>(item: T, renderItem?: (i: T) => React.ReactNode): string {
  const node = renderItem ? renderItem(item) : defaultLabel(item);
  if (typeof node === "string" || typeof node === "number") return String(node);
  return String((item.name as string) ?? (item.label as string) ?? item.key);
}

interface PanelProps<T extends PortableTransferItem> {
  title: React.ReactNode;
  side: "left" | "right";
  items: T[];
  total: number;
  height: number;
  showSearch?: boolean;
  searchPlaceholder?: string;
  search: string;
  onSearch: (v: string) => void;
  renderItem?: (item: T) => React.ReactNode;
  /** instant：行点击搬运；右侧行末 X */
  onRowAction?: (key: string) => void;
  /** batch：内部勾选受控 */
  checkedKeys?: Set<string>;
  onToggleCheck?: (key: string) => void;
  /** 是否显示行末 X（右侧 instant 且非 oneWay） */
  showRemove?: boolean;
  /** 右侧 header「清空选择」 */
  onClear?: () => void;
  emptyText: React.ReactNode;
  /** 分页 */
  page: number;
  pageCount: number;
  onPage: (p: number) => void;
  pageSize: number;
}

function TransferPanel<T extends PortableTransferItem>({
  title,
  side,
  items,
  total,
  height,
  showSearch,
  searchPlaceholder,
  search,
  onSearch,
  renderItem,
  onRowAction,
  checkedKeys,
  onToggleCheck,
  showRemove,
  onClear,
  emptyText,
  page,
  pageCount,
  onPage,
  pageSize,
}: PanelProps<T>) {
  const start = pageSize > 0 ? (page - 1) * pageSize : 0;
  const visible = pageSize > 0 ? items.slice(start, start + pageSize) : items;

  return (
    <div className="cp-transfer__panel">
      <header className="cp-transfer__header">
        <span className="cp-transfer__title">{title}</span>
        {side === "right" && onClear ? (
          <button type="button" className="cp-transfer__clear" onClick={onClear} disabled={total === 0}>
            清空选择
          </button>
        ) : (
          <span className="cp-transfer__count">共 {total} 条</span>
        )}
      </header>

      {showSearch && (
        <div className="cp-transfer__search">
          <span className="cp-transfer__search-icon">
            <SearchIcon />
          </span>
          <input
            className="cp-transfer__search-input"
            placeholder={searchPlaceholder ?? "搜索"}
            value={search}
            onChange={(e) => onSearch(e.target.value)}
          />
        </div>
      )}

      <div className="cp-transfer__body" style={{ minHeight: height }}>
        {items.length === 0 ? (
          <div className="cp-transfer__empty">{emptyText}</div>
        ) : (
          visible.map((item) => {
            const checked = checkedKeys?.has(item.key) ?? false;
            const rowCls = ["cp-transfer__row", item.disabled && "cp-transfer__row--disabled"]
              .filter(Boolean)
              .join(" ");
            return (
              <div
                key={item.key}
                className={rowCls}
                role={onToggleCheck || onRowAction ? "button" : undefined}
                tabIndex={item.disabled ? -1 : 0}
                onClick={() => {
                  if (item.disabled) return;
                  if (onToggleCheck) onToggleCheck(item.key);
                  else onRowAction?.(item.key);
                }}
                onKeyDown={(e) => {
                  if (item.disabled) return;
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    if (onToggleCheck) onToggleCheck(item.key);
                    else onRowAction?.(item.key);
                  }
                }}
              >
                {onToggleCheck && (
                  <input
                    type="checkbox"
                    className="cp-transfer__checkbox"
                    checked={checked}
                    disabled={item.disabled}
                    onChange={() => onToggleCheck(item.key)}
                    onClick={(e) => e.stopPropagation()}
                  />
                )}
                <span className="cp-transfer__cell">{renderItem ? renderItem(item) : defaultLabel(item)}</span>
                {showRemove && !item.disabled && (
                  <button
                    type="button"
                    className="cp-transfer__remove"
                    aria-label="移除"
                    onClick={(e) => {
                      e.stopPropagation();
                      onRowAction?.(item.key);
                    }}
                  >
                    <CloseIcon />
                  </button>
                )}
              </div>
            );
          })
        )}
      </div>

      {pageSize > 0 && pageCount > 1 && (
        <footer className="cp-transfer__footer">
          <span>共 {items.length} 条</span>
          <span className="cp-transfer__pager">
            <button
              type="button"
              className="cp-transfer__pager-btn"
              disabled={page <= 1}
              aria-label="上一页"
              onClick={() => onPage(page - 1)}
            >
              <ChevronLeft />
            </button>
            <span>
              {page} / {pageCount}
            </span>
            <button
              type="button"
              className="cp-transfer__pager-btn"
              disabled={page >= pageCount}
              aria-label="下一页"
              onClick={() => onPage(page + 1)}
            >
              <ChevronRight />
            </button>
          </span>
        </footer>
      )}
    </div>
  );
}

export function PortableTransfer<T extends PortableTransferItem = PortableTransferItem>({
  dataSource,
  targetKeys,
  onChange,
  renderItem,
  titles = ["全部", "已选"],
  mode = "instant",
  oneWay = false,
  showSearch = false,
  searchPlaceholder,
  filterOption,
  pageSize = 10,
  height = 330,
  notFoundContent,
  className = "",
}: PortableTransferProps<T>) {
  const targetSet = React.useMemo(() => new Set(targetKeys), [targetKeys]);

  const leftItems = React.useMemo(() => dataSource.filter((i) => !targetSet.has(i.key)), [dataSource, targetSet]);
  const rightItems = React.useMemo(() => dataSource.filter((i) => targetSet.has(i.key)), [dataSource, targetSet]);

  const [leftSearch, setLeftSearch] = React.useState("");
  const [rightSearch, setRightSearch] = React.useState("");
  const [leftPage, setLeftPage] = React.useState(1);
  const [rightPage, setRightPage] = React.useState(1);
  const [leftChecked, setLeftChecked] = React.useState<Set<string>>(new Set());
  const [rightChecked, setRightChecked] = React.useState<Set<string>>(new Set());

  const doFilter = React.useCallback(
    (items: T[], input: string): T[] => {
      if (!input.trim()) return items;
      if (filterOption) return items.filter((it) => filterOption(input, it));
      const needle = input.toLowerCase();
      return items.filter((it) => labelText(it, renderItem).toLowerCase().includes(needle));
    },
    [filterOption, renderItem]
  );

  const filteredLeft = React.useMemo(() => doFilter(leftItems, leftSearch), [doFilter, leftItems, leftSearch]);
  const filteredRight = React.useMemo(() => doFilter(rightItems, rightSearch), [doFilter, rightItems, rightSearch]);

  const leftPageCount = pageSize > 0 ? Math.max(1, Math.ceil(filteredLeft.length / pageSize)) : 1;
  const rightPageCount = pageSize > 0 ? Math.max(1, Math.ceil(filteredRight.length / pageSize)) : 1;

  const [leftPh, rightPh] = Array.isArray(searchPlaceholder)
    ? searchPlaceholder
    : [searchPlaceholder, searchPlaceholder];

  const moveRight = (keys: string[]) => {
    const movable = keys.filter((k) => {
      const it = dataSource.find((d) => d.key === k);
      return it && !it.disabled && !targetSet.has(k);
    });
    if (!movable.length) return;
    onChange([...targetKeys, ...movable], "right", movable);
    setLeftChecked(new Set());
  };

  const moveLeft = (keys: string[]) => {
    if (oneWay) return;
    const movable = keys.filter((k) => targetSet.has(k));
    if (!movable.length) return;
    onChange(
      targetKeys.filter((k) => !movable.includes(k)),
      "left",
      movable
    );
    setRightChecked(new Set());
  };

  const toggle = (set: Set<string>, setFn: (s: Set<string>) => void, key: string) => {
    const next = new Set(set);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    setFn(next);
  };

  const leftEmpty =
    notFoundContent ?? (leftSearch.trim() ? "没有匹配项，换个关键词试试" : "暂无可选数据");
  const rightEmpty =
    notFoundContent ?? (rightSearch.trim() ? "没有匹配项，换个关键词试试" : "从左侧选择条目添加");

  const merged = ["cp-transfer", className].filter(Boolean).join(" ");
  const isBatch = mode === "batch";

  return (
    <div className={merged}>
      <TransferPanel
        title={titles[0]}
        side="left"
        items={filteredLeft}
        total={leftItems.length}
        height={height}
        showSearch={showSearch}
        searchPlaceholder={leftPh}
        search={leftSearch}
        onSearch={(v) => {
          setLeftSearch(v);
          setLeftPage(1);
        }}
        renderItem={renderItem}
        checkedKeys={isBatch ? leftChecked : undefined}
        onToggleCheck={isBatch ? (k) => toggle(leftChecked, setLeftChecked, k) : undefined}
        onRowAction={isBatch ? undefined : (k) => moveRight([k])}
        emptyText={leftEmpty}
        page={leftPage}
        pageCount={leftPageCount}
        onPage={setLeftPage}
        pageSize={pageSize}
      />

      {isBatch && (
        <div className="cp-transfer__actions">
          <button
            type="button"
            className="cp-transfer__move-btn"
            aria-label="移到右侧"
            disabled={leftChecked.size === 0}
            onClick={() => moveRight([...leftChecked])}
          >
            <ChevronRight />
          </button>
          {!oneWay && (
            <button
              type="button"
              className="cp-transfer__move-btn"
              aria-label="移到左侧"
              disabled={rightChecked.size === 0}
              onClick={() => moveLeft([...rightChecked])}
            >
              <ChevronLeft />
            </button>
          )}
        </div>
      )}

      <TransferPanel
        title={titles[1]}
        side="right"
        items={filteredRight}
        total={rightItems.length}
        height={height}
        showSearch={showSearch}
        searchPlaceholder={rightPh}
        search={rightSearch}
        onSearch={(v) => {
          setRightSearch(v);
          setRightPage(1);
        }}
        renderItem={renderItem}
        checkedKeys={isBatch ? rightChecked : undefined}
        onToggleCheck={isBatch ? (k) => toggle(rightChecked, setRightChecked, k) : undefined}
        onRowAction={isBatch ? undefined : (k) => moveLeft([k])}
        showRemove={!isBatch && !oneWay}
        onClear={oneWay ? undefined : () => moveLeft(rightItems.map((i) => i.key))}
        emptyText={rightEmpty}
        page={rightPage}
        pageCount={rightPageCount}
        onPage={setRightPage}
        pageSize={pageSize}
      />
    </div>
  );
}
