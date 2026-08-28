/**
 * FilterTrigger - 筛选触发器统一组件
 *
 * 三种变体：
 *   1. "button" — 仿 Select 的下拉按钮（用于表单场景）
 *   2. "icon" — 表头 Filter 图标（用于表格列头筛选）
 *   3. "badge-pencil" — 徽章 + 铅笔编辑图标（用于内联范围展示+编辑）
 *
 * 所有变体遵循统一规范：
 *   - hover/active 描边色：blue-500
 *   - 已筛选（active）时图标/边框变蓝
 */
import React from "react";
import { ChevronDown, Filter, Pencil } from "lucide-react";
import { cn } from "@/lib/utils";

// ─── 类型定义 ────────────────────────────────────────────────────────────────

export type FilterTriggerVariant = "button" | "icon" | "badge-pencil";

export interface FilterTriggerProps {
  /** 触发器变体 */
  variant: FilterTriggerVariant;
  /** 是否处于激活/已筛选状态 */
  active?: boolean;
  /** 是否打开状态（影响旋转动画） */
  open?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 显示文字（button variant 的显示内容） */
  label?: React.ReactNode;
  /** placeholder（button variant，未选中时显示） */
  placeholder?: string;
  /** 列标题（icon variant 的标题文字） */
  title?: string;
  /** 额外 className */
  className?: string;
  /** 点击回调 */
  onClick?: (e: React.MouseEvent) => void;
}

// ─── 组件实现 ────────────────────────────────────────────────────────────────

export const FilterTrigger = React.forwardRef<HTMLButtonElement, FilterTriggerProps>(
  (
    {
      variant,
      active = false,
      open = false,
      disabled = false,
      label,
      placeholder = "请选择",
      title,
      className,
      onClick,
      ...props
    },
    ref
  ) => {
    if (variant === "button") {
      return (
        <button
          ref={ref}
          type="button"
          disabled={disabled}
          data-state={open ? "open" : "closed"}
          className={cn(
            "flex w-full items-center justify-between gap-2 border border-border bg-white px-3 py-[5px] text-sm font-normal whitespace-nowrap transition-colors outline-none rounded-[4px] h-9",
            "hover:border-blue-500 data-[state=open]:border-blue-500",
            "disabled:cursor-not-allowed disabled:bg-[#FAFAFA] disabled:border-[var(--border)] disabled:text-gray-400",
            className
          )}
          onClick={onClick}
          {...props}
        >
          {label ? (
            <span className="flex-1 min-w-0 flex items-center gap-2 truncate text-left">
              {label}
            </span>
          ) : (
            <span className="text-[var(--text-weak)]">{placeholder}</span>
          )}
          <ChevronDown
            className={cn(
              "w-4 h-4 text-[var(--text-weak)] shrink-0 transition-transform duration-200",
              open && "rotate-180"
            )}
          />
        </button>
      );
    }

    if (variant === "icon") {
      return (
        <button
          ref={ref}
          type="button"
          disabled={disabled}
          className={cn("flex items-center gap-1 group/filter", className)}
          onClick={onClick}
          {...props}
        >
          <span>{title}</span>
          <Filter
            className={cn(
              "w-3.5 h-3.5 transition-colors",
              active
                ? "text-[#355EF1]"
                : "text-[var(--text-weak)] group-hover/filter:text-[var(--text-muted)]"
            )}
          />
        </button>
      );
    }

    // variant === "badge-pencil"
    return (
      <button
        ref={ref}
        type="button"
        disabled={disabled}
        className={cn(
          "self-center text-[var(--text-weak)] hover:text-[var(--text-brand)] transition-colors",
          className
        )}
        title="编辑"
        onClick={onClick}
        {...props}
      >
        <Pencil className="w-3 h-3" />
      </button>
    );
  }
);

FilterTrigger.displayName = "FilterTrigger";

export default FilterTrigger;
