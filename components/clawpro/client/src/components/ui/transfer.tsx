"use client";

import * as React from "react";
import { SearchIcon, X, ChevronRight, ChevronLeft } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Pagination } from "@/components/ui/pagination";
import { BodyMedium, MetaText, HelperText } from "@/components/ui/Typography";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

/* ════════════════════════════════════════════════════════════════════
 * Transfer 穿梭框
 *
 * 结构对齐 Ant Design Transfer（index.tsx / Section.tsx / ListBody.tsx /
 * ListItem.tsx / search.tsx / Actions.tsx）：
 *
 *   <Transfer>
 *     ├─ <TransferSection direction="left">     ← Ant: Section
 *     │     ├─ Header (Checkbox + Title + Count + ClearAll)
 *     │     ├─ <TransferSearch>                  ← Ant: search
 *     │     ├─ <TransferTableBody>               ← Ant: ListBody（这里换成 Table）
 *     │     └─ Footer (Pagination)
 *     ├─ <TransferActions>                       ← Ant: Actions（batch 模式才渲染）
 *     └─ <TransferSection direction="right">
 *
 * 视觉 / 颜色 / 字号 全部走 clawpro 全局 token：
 *
 *   外壳描边              border-border (= var(--border) = #EAEEF4，§28)
 *   面板内分割线           border-[#f0f0f0]（§11.6 表内分割线规范）
 *   面板头部底色           var(--bg-grey-normal)
 *   面板头部标题           <BodyMedium>（强调级 §2）
 *   计数 / 占位 / 空态     <MetaText> / <HelperText>（弱辅助级）
 *   行字号 / 行高          由 <Table density="compact"> 强制（§11.6 / §0）
 *   行 hover / selected    继承 TableRow 全局态，组件不自造背景
 *   分页                   小尺寸 simple Pagination（弹窗内更紧凑）
 *
 * 移动模式：
 *   - mode="instant"（默认 / 截图体验）：
 *       左侧 row checkbox 勾上 → 立刻搬到右侧
 *       右侧 header 显示「清空选择」link，每行末尾显示 X
 *   - mode="batch"（Ant 经典）：
 *       左右各维护内部勾选；中间渲染 ">" / "<" 按钮做批量穿梭
 *
 * 业务化扩展（用于 CvmSelectComponent 后续薄壳化）：
 *   - isItemDisabled        判断行是否禁用
 *   - renderDisabledTrigger 在禁用行外侧包 Tooltip（如"基础版资产请升级到旗舰版"）
 *   - leftColumns / rightColumns / columns 独立或共用列定义
 * ════════════════════════════════════════════════════════════════════ */

/* ═══════════ 类型 ═══════════ */

export type TransferDirection = "left" | "right";

export interface TransferItem {
  key: string;
  [k: string]: any;
}

export interface TransferColumn<T extends TransferItem = TransferItem> {
  /** 列标识（仅用于 React key） */
  key: string;
  header?: React.ReactNode;
  /** 列宽（数字按 px；字符串原样应用，如 "32%" / "max-content"） */
  width?: number | string;
  className?: string;
  cellClassName?: string;
  render?: (item: T) => React.ReactNode;
}

export type TransferPaginationConfig = boolean | { pageSize?: number };

export interface TransferProps<T extends TransferItem = TransferItem> {
  /** 全集 */
  dataSource: T[];
  /** 受控：当前已选 keys */
  targetKeys: string[];
  /** 受控变更回调；direction="right" 表示从左到右 */
  onChange: (
    nextTargetKeys: string[],
    direction: TransferDirection,
    moveKeys: string[]
  ) => void;

  /** 受控（可选）：左右两侧"内部勾选"keys。仅 batch 模式有意义 */
  selectedKeys?: string[];
  onSelectChange?: (
    sourceSelectedKeys: string[],
    targetSelectedKeys: string[]
  ) => void;

  /** 列定义；同时作用于左右两侧；如需分别控制使用 leftColumns / rightColumns */
  columns?: TransferColumn<T>[];
  leftColumns?: TransferColumn<T>[];
  rightColumns?: TransferColumn<T>[];

  /** 取行 key，默认 item.key */
  rowKey?: keyof T | ((item: T) => string);

  /** 标题：[左, 右]，默认 ['全部', '已选'] */
  titles?: [React.ReactNode, React.ReactNode];

  /** 单侧 body 可滚动区域高度，默认 330 */
  height?: number;

  /** 搜索 */
  showSearch?: boolean;
  /** 搜索占位符；可分别指定 [左, 右] */
  searchPlaceholder?: string | [string, string];
  /** 自定义过滤逻辑；默认按所有 string 字段做大小写不敏感包含匹配 */
  filterOption?: (input: string, item: T) => boolean;

  /** 分页：boolean 或 { pageSize }；默认开启 pageSize=10。可分别控制 */
  pagination?: TransferPaginationConfig;
  leftPagination?: TransferPaginationConfig;
  rightPagination?: TransferPaginationConfig;

  /** 行禁用判断 */
  isItemDisabled?: (item: T) => boolean;
  /** 自定义禁用行外壳（用于挂 Tooltip）。仅在该行 disabled 时调用 */
  renderDisabledTrigger?: (
    item: T,
    defaultCheckbox: React.ReactElement
  ) => React.ReactNode;

  /** 移动模式，默认 instant */
  mode?: "instant" | "batch";

  /** 右侧不可移回；instant 模式下隐藏行末 X，batch 模式下隐藏中间 < 按钮 */
  oneWay?: boolean;

  /** 空态文案 */
  notFoundContent?: React.ReactNode;

  className?: string;
}

/* ═══════════ 工具 ═══════════ */

const DEFAULT_COLUMNS: TransferColumn[] = [
  {
    key: "__title",
    header: "名称",
    render: (item) =>
      (item as any).title ?? (item as any).name ?? String(item.key),
  },
];

function makeKeyResolver<T extends TransferItem>(
  rowKey: TransferProps<T>["rowKey"]
) {
  if (!rowKey) return (item: T) => String(item.key);
  if (typeof rowKey === "function") return (item: T) => String(rowKey(item));
  return (item: T) => String((item as any)[rowKey] ?? item.key);
}

function defaultFilterOption<T extends TransferItem>(input: string, item: T) {
  if (!input) return true;
  const needle = input.toLowerCase();
  return Object.values(item).some(
    (v) => typeof v === "string" && v.toLowerCase().includes(needle)
  );
}

function normalizePagination(
  p: TransferPaginationConfig | undefined,
  fallback: TransferPaginationConfig | undefined
): { enabled: boolean; pageSize: number } {
  const v = p ?? fallback;
  if (v === false) return { enabled: false, pageSize: 10 };
  if (v === undefined || v === true) return { enabled: true, pageSize: 10 };
  return { enabled: true, pageSize: v.pageSize ?? 10 };
}

/* ═══════════ 顶层 Transfer ═══════════ */

function Transfer<T extends TransferItem = TransferItem>(
  props: TransferProps<T>
) {
  const {
    dataSource,
    targetKeys,
    onChange,
    selectedKeys: selectedKeysProp,
    onSelectChange,
    columns,
    leftColumns,
    rightColumns,
    rowKey,
    titles,
    height = 330,
    showSearch = false,
    searchPlaceholder = "搜索",
    filterOption,
    pagination,
    leftPagination,
    rightPagination,
    isItemDisabled,
    renderDisabledTrigger,
    mode = "instant",
    oneWay = false,
    notFoundContent = "暂无数据",
    className,
  } = props;

  const getKey = React.useMemo(() => makeKeyResolver(rowKey), [rowKey]);
  const isDisabled = React.useCallback(
    (item: T) => Boolean(isItemDisabled?.(item)),
    [isItemDisabled]
  );

  const [leftTitle, rightTitle] = titles ?? ["全部", "已选"];
  const leftPlaceholder =
    typeof searchPlaceholder === "string"
      ? searchPlaceholder
      : searchPlaceholder[0];
  const rightPlaceholder =
    typeof searchPlaceholder === "string"
      ? searchPlaceholder
      : searchPlaceholder[1];

  const finalLeftColumns = (leftColumns ??
    columns ??
    DEFAULT_COLUMNS) as TransferColumn<T>[];
  const finalRightColumns = (rightColumns ??
    columns ??
    DEFAULT_COLUMNS) as TransferColumn<T>[];

  const leftPag = normalizePagination(leftPagination, pagination);
  const rightPag = normalizePagination(rightPagination, pagination);

  /* ── 数据集（参考 Ant useData）─────────────────── */
  const targetSet = React.useMemo(() => new Set(targetKeys), [targetKeys]);
  const leftDataSource = React.useMemo(
    () => dataSource.filter((item) => !targetSet.has(getKey(item))),
    [dataSource, targetSet, getKey]
  );
  const rightDataSource = React.useMemo(
    () => dataSource.filter((item) => targetSet.has(getKey(item))),
    [dataSource, targetSet, getKey]
  );

  /* ── 内部勾选（参考 Ant useSelection）──────────── */
  const [innerSourceKeys, setInnerSourceKeys] = React.useState<string[]>([]);
  const [innerTargetKeys, setInnerTargetKeys] = React.useState<string[]>([]);

  // 受控 selectedKeys 拆分：在 leftDataSource 中的归 source，在 rightDataSource 中的归 target
  const { sourceSelectedKeys, targetSelectedKeys } = React.useMemo(() => {
    if (selectedKeysProp) {
      const inTarget = new Set(rightDataSource.map(getKey));
      return {
        sourceSelectedKeys: selectedKeysProp.filter((k) => !inTarget.has(k)),
        targetSelectedKeys: selectedKeysProp.filter((k) => inTarget.has(k)),
      };
    }
    return {
      sourceSelectedKeys: innerSourceKeys,
      targetSelectedKeys: innerTargetKeys,
    };
  }, [
    selectedKeysProp,
    innerSourceKeys,
    innerTargetKeys,
    rightDataSource,
    getKey,
  ]);

  // 数据集变化时清理无效 key
  React.useEffect(() => {
    if (selectedKeysProp) return;
    const validLeft = new Set(leftDataSource.map(getKey));
    setInnerSourceKeys((prev) => {
      const next = prev.filter((k) => validLeft.has(k));
      return next.length === prev.length ? prev : next;
    });
  }, [leftDataSource, getKey, selectedKeysProp]);
  React.useEffect(() => {
    if (selectedKeysProp) return;
    const validRight = new Set(rightDataSource.map(getKey));
    setInnerTargetKeys((prev) => {
      const next = prev.filter((k) => validRight.has(k));
      return next.length === prev.length ? prev : next;
    });
  }, [rightDataSource, getKey, selectedKeysProp]);

  const setStateKeys = React.useCallback(
    (direction: TransferDirection, keys: string[]) => {
      if (direction === "left") {
        if (!selectedKeysProp) setInnerSourceKeys(keys);
        onSelectChange?.(keys, targetSelectedKeys);
      } else {
        if (!selectedKeysProp) setInnerTargetKeys(keys);
        onSelectChange?.(sourceSelectedKeys, keys);
      }
    },
    [selectedKeysProp, sourceSelectedKeys, targetSelectedKeys, onSelectChange]
  );

  /* ── 移动（参考 Ant moveTo）─────────────────────── */
  const moveTo = React.useCallback(
    (direction: TransferDirection, keys: string[]) => {
      // 过滤禁用
      const allDisabled = new Set(
        dataSource.filter(isDisabled).map(getKey)
      );
      const moveKeys = keys.filter((k) => !allDisabled.has(k));
      if (!moveKeys.length) return;

      const nextTargetKeys =
        direction === "right"
          ? Array.from(new Set([...targetKeys, ...moveKeys]))
          : targetKeys.filter((k) => !moveKeys.includes(k));

      // 清空对侧 selection
      setStateKeys(direction === "right" ? "left" : "right", []);
      onChange(nextTargetKeys, direction, moveKeys);
    },
    [dataSource, isDisabled, getKey, targetKeys, onChange, setStateKeys]
  );

  /* ── Section 回调 ──────────────────────────────── */
  const onLeftItemSelect = (key: string, checked: boolean) => {
    if (mode === "instant") {
      // instant：勾上即移到右侧
      if (checked) moveTo("right", [key]);
      return;
    }
    const set = new Set(sourceSelectedKeys);
    if (checked) set.add(key);
    else set.delete(key);
    setStateKeys("left", Array.from(set));
  };

  const onRightItemSelect = (key: string, checked: boolean) => {
    if (mode === "instant") return;
    const set = new Set(targetSelectedKeys);
    if (checked) set.add(key);
    else set.delete(key);
    setStateKeys("right", Array.from(set));
  };

  const onLeftItemSelectAll = (keys: string[], checkAll: boolean) => {
    if (mode === "instant") {
      // instant：左侧 header → 一键搬全部到右
      if (checkAll) moveTo("right", keys);
      return;
    }
    const set = new Set(sourceSelectedKeys);
    if (checkAll) keys.forEach((k) => set.add(k));
    else keys.forEach((k) => set.delete(k));
    setStateKeys("left", Array.from(set));
  };

  const onRightItemSelectAll = (keys: string[], checkAll: boolean) => {
    if (mode === "instant") return;
    const set = new Set(targetSelectedKeys);
    if (checkAll) keys.forEach((k) => set.add(k));
    else keys.forEach((k) => set.delete(k));
    setStateKeys("right", Array.from(set));
  };

  const onRightItemRemove = (keys: string[]) => moveTo("left", keys);
  const onClearAll = () => moveTo("left", rightDataSource.map(getKey));

  /* ── batch 中间按钮 ─────────────────────────────── */
  const moveToRight = () => moveTo("right", sourceSelectedKeys);
  const moveToLeft = () => moveTo("left", targetSelectedKeys);
  const leftActive = targetSelectedKeys.length > 0;
  const rightActive = sourceSelectedKeys.length > 0;

  /* ── 渲染 ───────────────────────────────────────── */
  return (
    <div
      data-slot="transfer"
      className={cn("flex items-stretch gap-3", className)}
    >
      <TransferSection<T>
        direction="left"
        titleText={leftTitle}
        dataSource={leftDataSource}
        checkedKeys={sourceSelectedKeys}
        onItemSelect={onLeftItemSelect}
        onItemSelectAll={onLeftItemSelectAll}
        showSearch={showSearch}
        searchPlaceholder={leftPlaceholder}
        filterOption={filterOption}
        columns={finalLeftColumns}
        getKey={getKey}
        isDisabled={isDisabled}
        renderDisabledTrigger={renderDisabledTrigger}
        height={height}
        pagination={leftPag}
        notFoundContent={notFoundContent}
        // instant 模式：左侧 header checkbox 当作"全部搬到右侧"的快捷开关，
        //   表现为永远未选中（搬走后该侧不再有数据），点击即触发 onItemSelectAll(checkAll=true)
        mode={mode}
      />

      {mode === "batch" && (
        <TransferActions
          rightActive={rightActive}
          leftActive={leftActive}
          oneWay={oneWay}
          moveToRight={moveToRight}
          moveToLeft={moveToLeft}
        />
      )}

      <TransferSection<T>
        direction="right"
        titleText={rightTitle}
        dataSource={rightDataSource}
        checkedKeys={targetSelectedKeys}
        onItemSelect={onRightItemSelect}
        onItemSelectAll={onRightItemSelectAll}
        onItemRemove={mode === "instant" && !oneWay ? onRightItemRemove : undefined}
        onClearAll={mode === "instant" && !oneWay ? onClearAll : undefined}
        showSearch={showSearch}
        searchPlaceholder={rightPlaceholder}
        filterOption={filterOption}
        columns={finalRightColumns}
        getKey={getKey}
        isDisabled={isDisabled}
        renderDisabledTrigger={renderDisabledTrigger}
        height={height}
        pagination={rightPag}
        notFoundContent={notFoundContent}
        mode={mode}
        showRemove={mode === "instant" && !oneWay}
      />
    </div>
  );
}

/* ════════════════════════════════════════════════════════════════════
 * TransferSection（单侧面板）
 * 对齐 Ant Section.tsx：内部维护 filterValue，渲染 Header + Search + Body + Footer
 * ══════════════════════════════════════════════════════════════════ */

interface TransferSectionProps<T extends TransferItem> {
  direction: TransferDirection;
  titleText: React.ReactNode;
  dataSource: T[];
  checkedKeys: string[];
  onItemSelect: (key: string, checked: boolean) => void;
  onItemSelectAll: (keys: string[], checkAll: boolean) => void;
  /** instant 模式右侧：每行末尾 X 删除 */
  onItemRemove?: (keys: string[]) => void;
  /** instant 模式右侧：header 「清空选择」 */
  onClearAll?: () => void;

  showSearch: boolean;
  searchPlaceholder: string;
  filterOption?: (input: string, item: T) => boolean;

  columns: TransferColumn<T>[];
  getKey: (item: T) => string;
  isDisabled: (item: T) => boolean;
  renderDisabledTrigger?: TransferProps<T>["renderDisabledTrigger"];

  height: number;
  pagination: { enabled: boolean; pageSize: number };
  notFoundContent: React.ReactNode;

  mode: "instant" | "batch";
  /** 仅右侧：true 时每行末尾出现 X 而非 row checkbox */
  showRemove?: boolean;
}

function TransferSection<T extends TransferItem>(
  props: TransferSectionProps<T>
) {
  const {
    direction,
    titleText,
    dataSource,
    checkedKeys,
    onItemSelect,
    onItemSelectAll,
    onItemRemove,
    onClearAll,
    showSearch,
    searchPlaceholder,
    filterOption,
    columns,
    getKey,
    isDisabled,
    renderDisabledTrigger,
    height,
    pagination,
    notFoundContent,
    mode,
    showRemove,
  } = props;

  const [filterValue, setFilterValue] = React.useState("");
  const filterFn = filterOption ?? defaultFilterOption;

  const filteredItems = React.useMemo(
    () =>
      filterValue
        ? dataSource.filter((i) => filterFn(filterValue, i))
        : dataSource,
    [dataSource, filterValue, filterFn]
  );

  const enabledFiltered = React.useMemo(
    () => filteredItems.filter((i) => !isDisabled(i)),
    [filteredItems, isDisabled]
  );

  /* ── header checkbox 状态 ───────────────────────── */
  const checkedKeySet = React.useMemo(
    () => new Set(checkedKeys),
    [checkedKeys]
  );
  const headerChecked =
    enabledFiltered.length > 0 &&
    enabledFiltered.every((i) => checkedKeySet.has(getKey(i)));
  const headerIndeterminate =
    !headerChecked && enabledFiltered.some((i) => checkedKeySet.has(getKey(i)));

  // instant 模式右侧不显示 header checkbox（用「清空选择」link 取代）
  const hideHeaderCheckbox = mode === "instant" && direction === "right";
  // instant 模式右侧不显示 row checkbox
  const hideRowCheckbox = mode === "instant" && direction === "right";

  const onHeaderToggle = () => {
    if (hideHeaderCheckbox) return;
    const keys = enabledFiltered.map(getKey);
    onItemSelectAll(keys, !headerChecked);
  };

  /* ── 计数文案 ───────────────────────────────────── */
  // 与 Ant Design Transfer 一致：右侧 header 只展示总数，不重复 title 里的"已选"语义
  const totalCount = filteredItems.length;
  const selectedCount = filteredItems.filter((i) =>
    checkedKeySet.has(getKey(i))
  ).length;
  let countLabel: React.ReactNode;
  if (mode === "instant") {
    countLabel = `${totalCount} 项`;
  } else {
    countLabel = selectedCount > 0
      ? `${selectedCount}/${totalCount} 项`
      : `${totalCount} 项`;
  }

  return (
    <div
      data-slot="transfer-section"
      data-direction={direction}
      className="flex w-full flex-1 min-w-0 flex-col overflow-hidden rounded-[4px] border border-border bg-white"
    >
      {/* ── Header ────────────────────────────────── */}
      <div className="flex h-9 items-center justify-between border-b border-[#f0f0f0] bg-[var(--bg-grey-normal)] px-3">
        <div className="flex min-w-0 items-center gap-2">
          {!hideHeaderCheckbox && (
            <Checkbox
              checked={
                headerIndeterminate ? "indeterminate" : headerChecked
              }
              disabled={enabledFiltered.length === 0}
              onCheckedChange={onHeaderToggle}
              aria-label="全选"
            />
          )}
          <BodyMedium className="truncate">{titleText}</BodyMedium>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <MetaText>{countLabel}</MetaText>
          {onClearAll && totalCount > 0 && (
            <Button
              type="button"
              variant="link"
              size="sm"
              className="h-auto p-0"
              onClick={onClearAll}
            >
              清空选择
            </Button>
          )}
        </div>
      </div>

      {/* ── Search ────────────────────────────────── */}
      {showSearch && (
        <div className="border-b border-[#f0f0f0] px-3 py-2">
          <TransferSearch
            value={filterValue}
            onChange={setFilterValue}
            placeholder={searchPlaceholder}
          />
        </div>
      )}

      {/* ── Body ──────────────────────────────────── */}
      <TransferTableBody<T>
        items={filteredItems}
        columns={columns}
        getKey={getKey}
        checkedKeySet={checkedKeySet}
        isDisabled={isDisabled}
        renderDisabledTrigger={renderDisabledTrigger}
        onItemSelect={onItemSelect}
        onItemRemove={onItemRemove}
        showRemove={showRemove}
        hideRowCheckbox={hideRowCheckbox}
        height={height}
        pagination={pagination}
        notFoundContent={notFoundContent}
      />
    </div>
  );
}

/* ════════════════════════════════════════════════════════════════════
 * TransferSearch（对齐 Ant search.tsx）
 * ══════════════════════════════════════════════════════════════════ */

interface TransferSearchProps {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  disabled?: boolean;
}

function TransferSearch({
  value,
  onChange,
  placeholder,
  disabled,
}: TransferSearchProps) {
  return (
    <div className="relative">
      <SearchIcon
        aria-hidden
        className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-[var(--text-weak)]"
      />
      <Input
        value={value}
        disabled={disabled}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
        className="!h-7 pl-7"
      />
    </div>
  );
}

/* ════════════════════════════════════════════════════════════════════
 * TransferTableBody（对齐 Ant ListBody.tsx，但用 Table）
 * 内部分页 + 列渲染 + row checkbox / row remove
 * ══════════════════════════════════════════════════════════════════ */

interface TransferTableBodyProps<T extends TransferItem> {
  items: T[];
  columns: TransferColumn<T>[];
  getKey: (item: T) => string;
  checkedKeySet: Set<string>;
  isDisabled: (item: T) => boolean;
  renderDisabledTrigger?: TransferProps<T>["renderDisabledTrigger"];

  onItemSelect: (key: string, checked: boolean) => void;
  onItemRemove?: (keys: string[]) => void;

  showRemove?: boolean;
  hideRowCheckbox?: boolean;

  height: number;
  pagination: { enabled: boolean; pageSize: number };
  notFoundContent: React.ReactNode;
}

function TransferTableBody<T extends TransferItem>({
  items,
  columns,
  getKey,
  checkedKeySet,
  isDisabled,
  renderDisabledTrigger,
  onItemSelect,
  onItemRemove,
  showRemove,
  hideRowCheckbox,
  height,
  pagination,
  notFoundContent,
}: TransferTableBodyProps<T>) {
  const [page, setPage] = React.useState(1);
  React.useEffect(() => {
    setPage((p) => {
      const max = Math.max(1, Math.ceil(items.length / pagination.pageSize));
      return Math.min(p, max);
    });
  }, [items.length, pagination.pageSize]);

  const pagedItems = React.useMemo(() => {
    if (!pagination.enabled) return items;
    const start = (page - 1) * pagination.pageSize;
    return items.slice(start, start + pagination.pageSize);
  }, [items, page, pagination]);

  const showPaginationFooter =
    pagination.enabled && items.length > pagination.pageSize;

  return (
    <>
      <ScrollArea
        className="min-h-0 flex-1"
        style={{ height, maxHeight: height }}
      >
        {pagedItems.length === 0 ? (
          <div className="flex h-full min-h-[160px] items-center justify-center">
            <HelperText as="span">{notFoundContent}</HelperText>
          </div>
        ) : (
          <Table density="compact" autoFixedColumns={false}>
            <TableHeader>
              <TableRow>
                {!hideRowCheckbox && <TableHead className="w-9 px-3" />}
                {columns.map((col) => (
                  <TableHead
                    key={col.key}
                    className={col.className}
                    style={
                      col.width != null
                        ? {
                            width:
                              typeof col.width === "number"
                                ? `${col.width}px`
                                : col.width,
                          }
                        : undefined
                    }
                  >
                    {col.header}
                  </TableHead>
                ))}
                {showRemove && <TableHead className="w-9 px-3" />}
              </TableRow>
            </TableHeader>
            <TableBody>
              {pagedItems.map((item) => {
                const k = getKey(item);
                const disabled = isDisabled(item);
                const checked = checkedKeySet.has(k);

                const checkboxNode = (
                  <Checkbox
                    checked={checked}
                    disabled={disabled}
                    onCheckedChange={(v) =>
                      onItemSelect(k, v === true)
                    }
                    aria-label="选择此行"
                  />
                );

                return (
                  <TableRow
                    key={k}
                    data-state={checked ? "selected" : undefined}
                    className={cn(disabled && "opacity-60")}
                  >
                    {!hideRowCheckbox && (
                      <TableCell
                        className="w-9 px-3"
                        onClick={(e) => {
                          e.stopPropagation();
                          if (!disabled) onItemSelect(k, !checked);
                        }}
                      >
                        {disabled && renderDisabledTrigger
                          ? renderDisabledTrigger(item, checkboxNode)
                          : checkboxNode}
                      </TableCell>
                    )}
                    {columns.map((col) => (
                      <TableCell
                        key={col.key}
                        className={cn(col.cellClassName)}
                        style={
                          col.width != null
                            ? {
                                width:
                                  typeof col.width === "number"
                                    ? `${col.width}px`
                                    : col.width,
                              }
                            : undefined
                        }
                      >
                        {col.render
                          ? col.render(item)
                          : ((item as any)[col.key] as React.ReactNode) ?? "-"}
                      </TableCell>
                    ))}
                    {showRemove && (
                      <TableCell className="w-9 px-3 text-right">
                        <button
                          type="button"
                          aria-label="移除"
                          disabled={disabled}
                          className="inline-flex size-5 items-center justify-center rounded-[2px] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-grey-hover-subtle)] hover:text-[var(--text-emphasis)] disabled:cursor-not-allowed disabled:opacity-50"
                          onClick={(e) => {
                            e.stopPropagation();
                            onItemRemove?.([k]);
                          }}
                        >
                          <X className="size-3.5" />
                        </button>
                      </TableCell>
                    )}
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </ScrollArea>

      {/* Footer：§11.6「数量统计 + 分页」标准 */}
      {showPaginationFooter && (
        <div className="grid grid-cols-[1fr_auto] items-center gap-4 border-t border-[#f0f0f0] px-3 py-2">
          <MetaText className="justify-self-start">共 {items.length} 条</MetaText>
          <Pagination
            simple
            size="small"
            total={items.length}
            current={page}
            pageSize={pagination.pageSize}
            onChange={setPage}
            className="justify-self-end justify-end flex-nowrap"
          />
        </div>
      )}
    </>
  );
}

/* ════════════════════════════════════════════════════════════════════
 * TransferActions（对齐 Ant Actions.tsx，仅 batch 模式渲染）
 * ══════════════════════════════════════════════════════════════════ */

interface TransferActionsProps {
  leftActive: boolean;
  rightActive: boolean;
  oneWay?: boolean;
  moveToLeft: () => void;
  moveToRight: () => void;
}

function TransferActions({
  leftActive,
  rightActive,
  oneWay,
  moveToLeft,
  moveToRight,
}: TransferActionsProps) {
  return (
    <div
      data-slot="transfer-actions"
      className="flex flex-col items-center justify-center gap-2 self-center"
    >
      <Button
        type="button"
        variant="outline"
        size="icon"
        className="h-7 w-7 rounded-[4px]"
        disabled={!rightActive}
        onClick={moveToRight}
        aria-label="移到右侧"
      >
        <ChevronRight className="h-4 w-4" />
      </Button>
      {!oneWay && (
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="h-7 w-7 rounded-[4px]"
          disabled={!leftActive}
          onClick={moveToLeft}
          aria-label="移到左侧"
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>
      )}
    </div>
  );
}

/* ═══════════ 导出 ═══════════ */

Transfer.Section = TransferSection;
Transfer.Search = TransferSearch;
Transfer.Actions = TransferActions;
Transfer.TableBody = TransferTableBody;

export {
  Transfer,
  TransferSection,
  TransferSearch,
  TransferActions,
  TransferTableBody,
};
