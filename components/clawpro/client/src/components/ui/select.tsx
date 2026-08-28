/**
 * Select - 统一下拉选择组件
 *
 * 支持两种模式：
 * 1. 基础模式（默认）：基于 Radix Select 的标准下拉
 * 2. 搜索模式（searchable=true）：基于 Popover + 搜索框的增强下拉
 *
 * 可选功能：
 * - searchable: 启用搜索框
 * - showCount: 启用底部计数（仅搜索模式下生效）
 */
import * as React from "react";
import { useState, useMemo, useCallback, useRef, useLayoutEffect } from "react";
import * as SelectPrimitive from "@radix-ui/react-select";
import { Check, CheckIcon, ChevronDown, ChevronDownIcon, ChevronUpIcon, Search, X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@/components/ui/button";
import { MetaText, MetaMedium } from "@/components/ui/Typography";
import { cn } from "@/lib/utils";

// ─── 内部工具：仅当文本被 truncate 时才显示 tooltip 的包裹器 ─────────────────
// 用于下拉触发器选中文案、下拉项 label 等可能被截断的字符串。
// 通过比较 scrollWidth / clientWidth 判断是否溢出，避免给未截断的文本误展示 tooltip。
function TruncatedText({
  text,
  className,
}: {
  text: string;
  className?: string;
}) {
  const ref = useRef<HTMLSpanElement>(null);
  const [overflow, setOverflow] = useState(false);

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const check = () => setOverflow(el.scrollWidth > el.clientWidth + 1);
    check();
    // 监听父容器尺寸变化（如弹窗宽度变化、面板内容变化等）
    const ro = new ResizeObserver(check);
    ro.observe(el);
    return () => ro.disconnect();
  }, [text]);

  const node = (
    <span ref={ref} className={cn("block truncate", className)}>
      {text}
    </span>
  );

  if (!overflow) return node;

  return (
    <Tooltip delayDuration={150}>
      <TooltipTrigger asChild>{node}</TooltipTrigger>
      <TooltipContent side="top" className="max-w-[320px] break-all">
        {text}
      </TooltipContent>
    </Tooltip>
  );
}

// ═══════════════════════════════════════════════════════════════════════════════
// 一、搜索增强模式（Popover 实现）
// ═══════════════════════════════════════════════════════════════════════════════

// ─── 类型定义 ────────────────────────────────────────────────────────────────

export interface SearchableSelectOption {
  /** 唯一标识 */
  value: string;
  /** 用于搜索匹配的关键词（不传则用 label） */
  searchText?: string;
  /** 渲染内容（支持自定义 JSX） */
  label: React.ReactNode;
  /** 触发器中显示的文本（不传则使用 label） */
  triggerLabel?: React.ReactNode;
}

export interface SearchableSelectProps {
  /** 选项列表 */
  options: SearchableSelectOption[];
  /** 当前选中值 */
  value?: string;
  /** 选中回调 */
  onChange: (value: string) => void;
  /** placeholder 文字 */
  placeholder?: string;
  /** 是否显示搜索框（默认 true） */
  searchable?: boolean;
  /** 搜索框 placeholder */
  searchPlaceholder?: string;
  /** 是否显示底部计数（默认 true） */
  showCount?: boolean;
  /** 底部计数模板，{count} 会被替换 */
  countTemplate?: string;
  /** 触发器 className */
  triggerClassName?: string;
  /** 面板宽度（默认跟随触发器宽度） */
  panelWidth?: string;
  /** 面板对齐方式 */
  align?: "start" | "center" | "end";
  /** 禁用 */
  disabled?: boolean;
  /** 是否允许清除已选项（hover/有值时在触发器内显示 X 按钮） */
  clearable?: boolean;
}

// ─── 搜索 Select 组件实现 ────────────────────────────────────────────────────

export function SearchableSelect({
  options,
  value,
  onChange,
  placeholder = "请选择",
  searchable = true,
  searchPlaceholder = "搜索...",
  showCount = true,
  countTemplate = "共 {count} 条",
  triggerClassName,
  panelWidth,
  align = "start",
  disabled = false,
  clearable = false,
}: SearchableSelectProps) {
  const [open, setOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");

  const handleOpenChange = useCallback((v: boolean) => {
    setOpen(v);
    if (v) setSearchQuery("");
  }, []);

  const filteredOptions = useMemo(() => {
    if (!searchable || !searchQuery.trim()) return options;
    const q = searchQuery.toLowerCase();
    return options.filter((opt) => {
      const text = opt.searchText || (typeof opt.label === "string" ? opt.label : opt.value);
      return String(text).toLowerCase().includes(q);
    });
  }, [options, searchQuery, searchable]);

  const selectedOption = options.find((o) => o.value === value);

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <button
          type="button"
          disabled={disabled}
          data-state={open ? "open" : "closed"}
          className={cn(
            "flex w-full items-center justify-between gap-2 border border-border bg-white px-3 py-[5px] text-sm font-normal whitespace-nowrap transition-colors outline-none rounded-[4px] h-9",
            "hover:border-blue-500 data-[state=open]:border-blue-500",
            "disabled:cursor-not-allowed disabled:bg-[#FAFAFA] disabled:border-[var(--border)] disabled:text-gray-400",
            triggerClassName
          )}
        >
          {selectedOption ? (
            (() => {
              const display = selectedOption.triggerLabel ?? selectedOption.label;
              // 字符串 label：开启溢出 tooltip；JSX 自定义渲染：保持原行为
              if (typeof display === "string") {
                return (
                  <span className="flex-1 min-w-0 flex items-center gap-2 text-left">
                    <TruncatedText text={display} />
                  </span>
                );
              }
              return (
                <span className="flex-1 min-w-0 flex items-center gap-2 truncate text-left">
                  {display}
                </span>
              );
            })()
          ) : (
            <span className="flex-1 min-w-0 truncate text-left text-[var(--text-weak)]">{placeholder}</span>
          )}
          {clearable && selectedOption && !disabled && (
            <span
              role="button"
              tabIndex={-1}
              aria-label="清除"
              className="shrink-0 flex items-center justify-center text-[var(--text-weak)] hover:text-[var(--text-title)] transition-colors"
              onMouseDown={(e) => e.preventDefault()}
              onClick={(e) => {
                e.stopPropagation();
                onChange("");
              }}
            >
              <X className="w-3.5 h-3.5" />
            </span>
          )}
          <ChevronDown className={cn("w-4 h-4 text-[var(--text-weak)] shrink-0 transition-transform", open && "rotate-180")} />
        </button>
      </PopoverTrigger>
      <PopoverContent
        className="p-0 rounded-[4px] border-none shadow-[var(--shadow-popover)] flex flex-col max-h-[320px]"
        style={
          panelWidth
            ? { minWidth: 220, width: panelWidth }
            : { minWidth: 220, width: "var(--radix-popover-trigger-width)" }
        }
        align={align}
        sideOffset={4}
      >
        {/* 搜索框（可选） */}
        {searchable && (
          <div className="p-2 pb-0 shrink-0">
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
        <div
          className="flex-1 min-h-0 overflow-y-auto overscroll-contain p-2 space-y-0.5"
          style={{ scrollbarWidth: "thin", scrollbarColor: "transparent transparent" }}
          onMouseEnter={(e) => { (e.currentTarget as HTMLElement).style.scrollbarColor = "rgba(0,0,0,0.2) transparent"; }}
          onMouseLeave={(e) => { (e.currentTarget as HTMLElement).style.scrollbarColor = "transparent transparent"; }}
          onWheel={(e) => e.stopPropagation()}
        >
          {filteredOptions.length === 0 ? (
            <div className="py-3 text-center">
              <MetaText tone="weak">未找到匹配项</MetaText>
            </div>
          ) : (
            filteredOptions.map((opt) => {
              const isSelected = opt.value === value;
              return (
                <div
                  key={opt.value}
                  className={cn(
                    "relative flex items-center gap-2 h-8 px-3 rounded-[6px] cursor-pointer transition-colors text-sm",
                    isSelected
                      ? "bg-[var(--bg-brand-selected)] text-blue-500 font-medium"
                      : "text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)]"
                  )}
                  onClick={() => {
                    onChange(opt.value);
                    setOpen(false);
                  }}
                >
                  {typeof opt.label === "string" ? (
                    <span className={cn("flex-1 min-w-0", isSelected && "pr-5")}>
                      <TruncatedText text={opt.label} />
                    </span>
                  ) : (
                    <span className={cn("flex-1 min-w-0 truncate", isSelected && "pr-5")}>{opt.label}</span>
                  )}
                  {isSelected && (
                    <span className="absolute right-3 flex size-3.5 items-center justify-center">
                      <Check className="size-4 text-blue-500" />
                    </span>
                  )}
                </div>
              );
            })
          )}
        </div>
        {/* 底部计数（可选） */}
        {showCount && (
          <div className="shrink-0 mx-2 border-t border-[#EAEEF4] py-2 flex items-center">
            <MetaText>{countTemplate.replace("{count}", String(options.length))}</MetaText>
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}

// ═══════════════════════════════════════════════════════════════════════════════
// 一·扩展、即时多选模式（Popover + Checkbox 实现，多选 / 即时生效）
// ═══════════════════════════════════════════════════════════════════════════════

// ─── 类型定义 ────────────────────────────────────────────────────────────────

export interface InstantMultiSelectOption {
  /** 唯一标识 */
  value: string;
  /** 用于搜索匹配的关键词（不传则用 label） */
  searchText?: string;
  /** 渲染内容（支持自定义 JSX） */
  label: React.ReactNode;
  /** 触发器拼接显示的纯文本（不传则使用 label 字符串） */
  triggerLabel?: string;
  /** 禁用 */
  disabled?: boolean;
  /** 该项下方插入分割线（1px border-[#EAEEF4]） */
  dividerAfter?: boolean;
}

export interface InstantMultiSelectSection {
  /** 组织标题 */
  label: string;
  /** 该组织下的选项 */
  options: InstantMultiSelectOption[];
}

export interface InstantMultiSelectProps {
  /** 扁平选项（与 sections 互斥；若同时传入，优先 sections） */
  options?: InstantMultiSelectOption[];
  /** 组织选项（每段独立标题） */
  sections?: InstantMultiSelectSection[];
  /** 当前选中值集合 */
  value: Set<string>;
  /** 选中变化回调（即时生效） */
  onChange: (value: Set<string>) => void;
  /** placeholder 文字 */
  placeholder?: string;
  /** 是否显示搜索框（默认 true） */
  searchable?: boolean;
  /** 搜索框 placeholder */
  searchPlaceholder?: string;
  /** 是否显示顶部全选（默认 true） */
  showSelectAll?: boolean;
  /** 全选行文案（默认 "全选"） */
  selectAllLabel?: string;
  /** 是否显示底部计数 + 清除 footer（默认 true，仅当有选中项时呈现） */
  showFooter?: boolean;
  /** 底部计数文案模板，{count} 会被替换（默认 "已选 {count} 项"） */
  selectedCountTemplate?: string;
  /** 清除按钮文案（默认 "清除"） */
  clearLabel?: string;
  /** 触发器 className */
  triggerClassName?: string;
  /** 面板宽度（默认跟随触发器宽度，最小 220） */
  panelWidth?: string;
  /** 面板对齐方式（默认 start） */
  align?: "start" | "center" | "end";
  /** 禁用 */
  disabled?: boolean;
}

// ─── 组件实现 ────────────────────────────────────────────────────────────────

/**
 * InstantMultiSelect — 下拉框触发器 + 即时生效多选面板
 *
 * 特性：
 *   - 触发器：与 SearchableSelect 同款（hover/open 蓝色描边、ChevronDown、disabled 灰底）
 *   - 选中态：所有选中项名称用「、」拼接显示，溢出自动 truncate + Tooltip
 *   - 面板：搜索框 + 顶部「全选」+ 列表（扁平 options 或组织 sections）+ 底部计数/清除
 *   - 交互：勾选即生效，无需「确认」按钮
 *
 * 与 SearchableSelect 的差异：单选 → 多选；选中即关闭 → 持续打开；Check 图标 → Checkbox。
 */
export function InstantMultiSelect({
  options,
  sections,
  value,
  onChange,
  placeholder = "请选择",
  searchable = true,
  searchPlaceholder = "搜索...",
  showSelectAll = true,
  selectAllLabel = "全选",
  showFooter = true,
  selectedCountTemplate = "已选 {count} 项",
  clearLabel = "清除",
  triggerClassName,
  panelWidth,
  align = "start",
  disabled = false,
}: InstantMultiSelectProps) {
  const [open, setOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");

  const handleOpenChange = useCallback((v: boolean) => {
    setOpen(v);
    if (v) setSearchQuery("");
  }, []);

  // 统一为 sections（扁平 options 自动包装成单段无标题组织）
  const normalizedSections = useMemo<InstantMultiSelectSection[]>(() => {
    if (sections && sections.length > 0) return sections;
    return [{ label: "", options: options ?? [] }];
  }, [sections, options]);

  // 扁平所有选项（用于全选 / 计数 / 触发器拼接）
  const allOptions = useMemo(
    () => normalizedSections.flatMap((s) => s.options),
    [normalizedSections]
  );

  // 搜索过滤（保留组织结构，只过滤选项）
  const filteredSections = useMemo<InstantMultiSelectSection[]>(() => {
    if (!searchable || !searchQuery.trim()) return normalizedSections;
    const q = searchQuery.toLowerCase();
    return normalizedSections
      .map((sec) => ({
        ...sec,
        options: sec.options.filter((opt) => {
          const text =
            opt.searchText ||
            (typeof opt.label === "string" ? opt.label : opt.value);
          return String(text).toLowerCase().includes(q);
        }),
      }))
      .filter((sec) => sec.options.length > 0);
  }, [normalizedSections, searchable, searchQuery]);

  const hasAnyMatch = filteredSections.length > 0;

  // 全选态计算（基于"可选中（非 disabled）的所有选项"）
  const selectableValues = useMemo(
    () => allOptions.filter((o) => !o.disabled).map((o) => o.value),
    [allOptions]
  );
  const selectedSelectableCount = selectableValues.filter((v) =>
    value.has(v)
  ).length;
  const isAllSelected =
    selectableValues.length > 0 &&
    selectedSelectableCount === selectableValues.length;
  const isIndeterminate =
    selectedSelectableCount > 0 && !isAllSelected;

  // 触发器拼接文本（用「、」拼接所有选中项的纯文本）
  // 全选时仅显示全选文案（selectAllLabel），不拼接所有选项
  const triggerText = useMemo(() => {
    if (value.size === 0) return "";
    if (isAllSelected) return selectAllLabel;
    const map = new Map(allOptions.map((o) => [o.value, o]));
    return Array.from(value)
      .map((k) => {
        const opt = map.get(k);
        if (!opt) return null;
        return (
          opt.triggerLabel ||
          (typeof opt.label === "string" ? opt.label : opt.value)
        );
      })
      .filter((s): s is string => Boolean(s))
      .join("、");
  }, [value, allOptions, isAllSelected, selectAllLabel]);

  // ── 操作 ─────────────────────────────────────────────────────────────────
  const toggleOne = useCallback(
    (key: string) => {
      const next = new Set(value);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      onChange(next);
    },
    [value, onChange]
  );

  const toggleAll = useCallback(() => {
    if (isAllSelected) {
      // 取消所有可选中项（保留 disabled 的现有状态）
      const next = new Set(value);
      selectableValues.forEach((v) => next.delete(v));
      onChange(next);
    } else {
      const next = new Set(value);
      selectableValues.forEach((v) => next.add(v));
      onChange(next);
    }
  }, [isAllSelected, selectableValues, value, onChange]);

  const handleClear = useCallback(() => {
    onChange(new Set());
    setSearchQuery("");
  }, [onChange]);

  // 阻止滚轮事件冒泡（避免页面滚动）
  const handleWheelStop = useCallback((e: React.WheelEvent) => {
    e.stopPropagation();
  }, []);

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <button
          type="button"
          disabled={disabled}
          data-state={open ? "open" : "closed"}
          className={cn(
            "flex w-full items-center justify-between gap-2 border border-border bg-white px-3 py-[5px] text-sm font-normal whitespace-nowrap transition-colors outline-none rounded-[4px] h-9",
            "hover:border-blue-500 data-[state=open]:border-blue-500",
            "disabled:cursor-not-allowed disabled:bg-[#FAFAFA] disabled:border-[var(--border)] disabled:text-gray-400",
            triggerClassName
          )}
        >
          {triggerText ? (
            <span className="flex-1 min-w-0 text-left">
              <TruncatedText text={triggerText} />
            </span>
          ) : (
            <span className="flex-1 min-w-0 truncate text-left text-[var(--text-weak)]">
              {placeholder}
            </span>
          )}
          <ChevronDown
            className={cn(
              "w-4 h-4 text-[var(--text-weak)] shrink-0 transition-transform",
              open && "rotate-180"
            )}
          />
        </button>
      </PopoverTrigger>
      <PopoverContent
        className="p-0 rounded-[4px] border-none shadow-[var(--shadow-popover)] flex flex-col max-h-[360px]"
        style={
          panelWidth
            ? { minWidth: 220, width: panelWidth }
            : { minWidth: 220, width: "var(--radix-popover-trigger-width)" }
        }
        align={align}
        sideOffset={4}
        onWheel={handleWheelStop}
      >
        {/* 搜索框（可选） */}
        {searchable && (
          <div className="p-2 pb-0 shrink-0">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)] pointer-events-none" />
              <Input
                className="h-8 pl-8 pr-7 text-sm"
                placeholder={searchPlaceholder}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onClick={(e) => e.stopPropagation()}
              />
              {searchQuery && (
                <button
                  type="button"
                  onClick={() => setSearchQuery("")}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[var(--text-weak)] hover:text-[var(--text-secondary)]"
                >
                  <X className="w-3.5 h-3.5" />
                </button>
              )}
            </div>
          </div>
        )}

        {/* 顶部全选（搜索时若无匹配则隐藏） */}
        {showSelectAll && hasAnyMatch && (
          <div className="px-2 pt-2 shrink-0">
            <button
              type="button"
              onClick={toggleAll}
              className={cn(
                "flex items-center gap-2 w-full h-8 px-3 rounded-[6px] transition-colors",
                isAllSelected
                  ? "bg-[var(--bg-brand-selected)]"
                  : "hover:bg-[var(--bg-grey-hover)]"
              )}
            >
              <Checkbox
                checked={
                  isIndeterminate ? "indeterminate" : isAllSelected
                }
                className="pointer-events-none"
              />
              <span className="flex-1 min-w-0 text-left text-sm text-[var(--text-title)]">
                {selectAllLabel}
              </span>
            </button>
          </div>
        )}

        {/* 列表区 — 无搜索框+无全选时上边距 8px (pt-2)，否则 4px (pt-1) */}
        <div
          className={cn(
            "flex-1 min-h-0 overflow-y-auto overscroll-contain p-2 space-y-0.5",
            (searchable || showSelectAll) ? "pt-1" : "pt-2"
          )}
          style={{ scrollbarWidth: "thin", scrollbarColor: "transparent transparent" }}
          onMouseEnter={(e) => {
            (e.currentTarget as HTMLElement).style.scrollbarColor =
              "rgba(0,0,0,0.2) transparent";
          }}
          onMouseLeave={(e) => {
            (e.currentTarget as HTMLElement).style.scrollbarColor =
              "transparent transparent";
          }}
          onWheel={handleWheelStop}
        >
          {!hasAnyMatch ? (
            <div className="py-3 text-center">
              <MetaText tone="weak">未找到匹配项</MetaText>
            </div>
          ) : (
            filteredSections.map((sec, idx) => (
              <React.Fragment key={sec.label || `__sec_${idx}`}>
                {sec.label && (
                  <div className="px-3 pt-1.5 pb-1 select-none">
                    <MetaMedium tone="weak">{sec.label}</MetaMedium>
                  </div>
                )}
                {sec.options.map((opt) => {
                  const checked = value.has(opt.value);
                  return (
                    <React.Fragment key={opt.value}>
                      <button
                        type="button"
                        disabled={opt.disabled}
                        onClick={() => !opt.disabled && toggleOne(opt.value)}
                        className={cn(
                          "flex items-center gap-2 w-full h-8 px-3 rounded-[6px] transition-colors text-sm text-left",
                          opt.disabled
                            ? "cursor-not-allowed text-gray-400"
                            : checked
                              ? "bg-[var(--bg-brand-selected)] text-[var(--text-title)]"
                              : "text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)]"
                        )}
                      >
                        <Checkbox
                          checked={checked}
                          disabled={opt.disabled}
                          className="pointer-events-none shrink-0"
                        />
                        {typeof opt.label === "string" ? (
                          <span className="flex-1 min-w-0">
                            <TruncatedText text={opt.label} />
                          </span>
                        ) : (
                          <span className="flex-1 min-w-0 truncate">
                            {opt.label}
                          </span>
                        )}
                      </button>
                      {opt.dividerAfter && (
                        <div className="mx-1 my-1 h-px bg-[#EAEEF4]" />
                      )}
                    </React.Fragment>
                  );
                })}
              </React.Fragment>
            ))
          )}
        </div>

        {/* 底部：已选计数 + 清除（即时生效：计数灰色 + 清除蓝色文字链接） */}
        {showFooter && value.size > 0 && (
          <div className="shrink-0 mx-2 border-t border-[#EAEEF4] py-2 flex items-center justify-between">
            <MetaText>
              {isAllSelected ? "已选全部" : selectedCountTemplate.replace("{count}", String(value.size))}
            </MetaText>
            <button
              type="button"
              className="hover:opacity-80 transition-opacity"
              onClick={handleClear}
            >
              <MetaText tone="brand">{clearLabel}</MetaText>
            </button>
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}

// ═══════════════════════════════════════════════════════════════════════════════
// 二、基础模式（Radix Select 原子组件）
// ═══════════════════════════════════════════════════════════════════════════════

function Select({
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Root>) {
  return <SelectPrimitive.Root data-slot="select" {...props} />;
}

function SelectGroup({
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Group>) {
  return <SelectPrimitive.Group data-slot="select-group" {...props} />;
}

function SelectValue({
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Value>) {
  return <SelectPrimitive.Value data-slot="select-value" {...props} />;
}

function SelectTrigger({
  className,
  size = "default",
  tenant = false,
  children,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Trigger> & {
  size?: "sm" | "default";
  /**
   * 用户端形态：全圆角胶囊（rounded-full），与用户端整体圆角风格统一。
   * 仅 pages/tenant/** 业务页使用；管理端保持 rounded-[4px]。
   * 规范来源：Figma 1116-6220 / SKILL-TENANT.md
   */
  tenant?: boolean;
}) {
  return (
    <SelectPrimitive.Trigger
      data-slot="select-trigger"
      data-size={size}
      data-tenant={tenant ? "true" : undefined}
      className={cn(
        // 描边统一走全局 token --border (#EAEEF4)，与 Input / 容器 / Tag 等控件对齐
        // w-full 保证 trigger 填满父容器，图标始终靠右（与 SearchableSelect / InstantMultiSelect 对齐）
        "flex w-full items-center justify-between gap-2 border border-border bg-white px-3 py-[5px] text-sm font-normal whitespace-nowrap transition-colors outline-none",
        tenant ? "rounded-full" : "rounded-[4px]",
        "hover:border-blue-500",
        "data-[state=open]:border-blue-500",
        "data-[placeholder]:text-[var(--text-weak)]",
        "disabled:cursor-not-allowed disabled:bg-[#FAFAFA] disabled:border-[var(--border)] disabled:text-gray-400",
        "aria-invalid:border-destructive",
        "data-[size=default]:h-9 data-[size=sm]:h-8",
        "*:data-[slot=select-value]:line-clamp-1 *:data-[slot=select-value]:flex *:data-[slot=select-value]:items-center *:data-[slot=select-value]:gap-2",
        "[&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        className
      )}
      {...props}
    >
      {children}
      <SelectPrimitive.Icon asChild>
        <ChevronDownIcon className="size-4 text-gray-500 transition-transform duration-200 [[data-state=open]>&]:rotate-180" />
      </SelectPrimitive.Icon>
    </SelectPrimitive.Trigger>
  );
}

function SelectContent({
  className,
  children,
  position = "popper",
  align = "center",
  tenant = false,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Content> & {
  /**
   * 用户端形态：下拉面板圆角 12px，与 `PopoverContent` / `TenantCard` 等用户端浮层圆角统一
   * （配套 `SelectTrigger tenant` / `Input tenant` 的全圆角控件形态）；管理端保持 rounded-[4px]。
   * 业务层不要再用 className 覆盖圆角，统一通过该 prop 切换形态。
   */
  tenant?: boolean;
}) {
  const contentRef = React.useRef<HTMLDivElement>(null);

  const handleWheel = React.useCallback((e: React.WheelEvent<HTMLDivElement>) => {
    const viewport = contentRef.current?.querySelector('[data-radix-select-viewport]') as HTMLElement | null;
    if (viewport) {
      viewport.scrollTop += e.deltaY;
    }
  }, []);

  return (
    <SelectPrimitive.Portal>
      <SelectPrimitive.Content
        ref={contentRef}
        onWheel={handleWheel}
        data-slot="select-content"
        data-tenant={tenant ? "true" : undefined}
        className={cn(
          "bg-white text-[color:var(--wm-color-text-primary,black)]",
          "data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2",
          "relative z-50 max-h-(--radix-select-content-available-height) min-w-[8rem] origin-(--radix-select-content-transform-origin) overflow-x-hidden overflow-y-auto shadow-[var(--shadow-popover)]",
          tenant ? "rounded-[12px]" : "rounded-[4px]",
          position === "popper" &&
            "data-[side=bottom]:translate-y-1 data-[side=left]:-translate-x-1 data-[side=right]:translate-x-1 data-[side=top]:-translate-y-1",
          className
        )}
        position={position}
        align={align}
        collisionBoundary={[]}
        {...props}
      >
        <SelectScrollUpButton />
        <SelectPrimitive.Viewport
          className={cn(
            // p-2 提供面板内边距；space-y-0.5 让相邻 SelectItem 之间留 2px 视觉间距，
            // 与 SearchableSelect（Popover 版）完全对齐。
            "p-2 space-y-0.5",
            position === "popper" &&
              "h-[var(--radix-select-trigger-height)] w-full min-w-[var(--radix-select-trigger-width)] scroll-my-1"
          )}
        >
          {children}
        </SelectPrimitive.Viewport>
        <SelectScrollDownButton />
      </SelectPrimitive.Content>
    </SelectPrimitive.Portal>
  );
}

function SelectLabel({
  className,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Label>) {
  return (
    <SelectPrimitive.Label
      data-slot="select-label"
      className={cn("text-muted-foreground px-2 py-1.5 text-xs", className)}
      {...props}
    />
  );
}

function SelectItem({
  className,
  children,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Item>) {
  return (
    <SelectPrimitive.Item
      data-slot="select-item"
      className={cn(
        "group/item relative flex w-full cursor-default items-center rounded-[6px] h-8 px-3 py-[9px] text-sm font-normal text-[color:var(--wm-color-text-primary,black)] outline-hidden select-none",
        "hover:bg-[var(--bg-grey-hover)]",
        "focus:bg-[var(--bg-grey-hover)]",
        "data-[state=checked]:bg-[var(--bg-brand-selected)] data-[state=checked]:text-blue-500 data-[state=checked]:font-medium",
        "data-[disabled]:pointer-events-none data-[disabled]:text-gray-400",
        "[&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        className
      )}
      {...props}
    >
      {/* 文字区域：flex-1 撑满剩余宽度，min-w-0 允许 truncate 生效，
          选中时 pr-[18px] 为对勾(14px)+间距(4px)预留空间，避免文字与对勾重叠 */}
      <span className="flex-1 min-w-0 group-data-[state=checked]/item:pr-[18px]">
        <SelectPrimitive.ItemText>
          {typeof children === "string" ? (
            <TruncatedText text={children} />
          ) : (
            <span className="block truncate">{children}</span>
          )}
        </SelectPrimitive.ItemText>
      </span>
      {/* 对勾图标：绝对定位在右侧，与文字保持 4px 间距（通过文字区 pr-[18px] 保证） */}
      <span className="absolute right-3 flex size-3.5 items-center justify-center">
        <SelectPrimitive.ItemIndicator>
          <CheckIcon className="size-4 text-blue-500" />
        </SelectPrimitive.ItemIndicator>
      </span>
    </SelectPrimitive.Item>
  );
}

function SelectSeparator({
  className,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Separator>) {
  return (
    <SelectPrimitive.Separator
      data-slot="select-separator"
      className={cn("bg-border pointer-events-none -mx-1 my-1 h-px", className)}
      {...props}
    />
  );
}

function SelectScrollUpButton({
  className,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.ScrollUpButton>) {
  return (
    <SelectPrimitive.ScrollUpButton
      data-slot="select-scroll-up-button"
      className={cn(
        "flex cursor-default items-center justify-center py-1",
        className
      )}
      {...props}
    >
      <ChevronUpIcon className="size-4" />
    </SelectPrimitive.ScrollUpButton>
  );
}

function SelectScrollDownButton({
  className,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.ScrollDownButton>) {
  return (
    <SelectPrimitive.ScrollDownButton
      data-slot="select-scroll-down-button"
      className={cn(
        "flex cursor-default items-center justify-center py-1",
        className
      )}
      {...props}
    >
      <ChevronDownIcon className="size-4" />
    </SelectPrimitive.ScrollDownButton>
  );
}

// ═══════════════════════════════════════════════════════════════════════════════
// 三、表头多选模式（filter-icon 触发 + confirm）
// ═══════════════════════════════════════════════════════════════════════════════

/**
 * FilterMultiSelect - 表头多选下拉（filter-icon 触发，confirm 模式）
 *
 * 数据结构：FilterOption[]（{ value, label, disabled? }）
 * 承载原 TableHeaderFilter，作为 Select 家族的 "filter-multi" 变体。
 *
 * 设计：内部委托给 TableHeaderFilter 旧实现，
 * 待业务全部迁移完成后，再把实现内联进本文件并删除旧文件。
 */
import { TableHeaderFilter, type TableHeaderFilterProps, type FilterOption } from "@/components/_internal/TableHeaderFilter";

export type { FilterOption };

export interface FilterMultiSelectProps extends Omit<TableHeaderFilterProps, "selectedValues" | "onConfirm"> {
  /** 当前选中值集合（新 API） */
  value?: Set<string>;
  /** 确认回调（新 API） */
  onChange?: (value: Set<string>) => void;
  /** @deprecated 用 value 替代 */
  selectedValues?: Set<string>;
  /** @deprecated 用 onChange 替代 */
  onConfirm?: (value: Set<string>) => void;
}

export function FilterMultiSelect({ value, onChange, selectedValues, onConfirm, ...rest }: FilterMultiSelectProps) {
  return (
    <TableHeaderFilter
      selectedValues={value ?? selectedValues ?? new Set()}
      onConfirm={onChange ?? onConfirm ?? (() => {})}
      {...rest}
    />
  );
}

// ═══════════════════════════════════════════════════════════════════════════════
// 四、统一导出
// ═══════════════════════════════════════════════════════════════════════════════

export {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectScrollDownButton,
  SelectScrollUpButton,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
};
