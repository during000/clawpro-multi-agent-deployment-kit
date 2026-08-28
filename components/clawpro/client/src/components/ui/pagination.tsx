import * as React from "react";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  MoreHorizontalIcon,
} from "lucide-react";

import { cn } from "@/lib/utils";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// ─── shadcn 原子组件（保留供高级场景组合使用）────────────────────────────────

function PaginationNav({ className, ...props }: React.ComponentProps<"nav">) {
  return (
    <nav
      role="navigation"
      aria-label="pagination"
      data-slot="pagination"
      className={cn("mx-auto flex w-full justify-center", className)}
      {...props}
    />
  );
}

function PaginationContent({
  className,
  ...props
}: React.ComponentProps<"ul">) {
  return (
    <ul
      data-slot="pagination-content"
      className={cn("flex flex-row items-center gap-1", className)}
      {...props}
    />
  );
}

function PaginationItem({ ...props }: React.ComponentProps<"li">) {
  return <li data-slot="pagination-item" {...props} />;
}

type PaginationLinkProps = {
  isActive?: boolean;
} & Pick<React.ComponentProps<typeof Button>, "size"> &
  React.ComponentProps<"a">;

function PaginationLink({
  className,
  isActive,
  size = "icon",
  ...props
}: PaginationLinkProps) {
  return (
    <a
      aria-current={isActive ? "page" : undefined}
      data-slot="pagination-link"
      data-active={isActive}
      className={cn(
        buttonVariants({
          variant: isActive ? "outline" : "ghost",
          size,
        }),
        className
      )}
      {...props}
    />
  );
}

function PaginationPrevious({
  className,
  ...props
}: React.ComponentProps<typeof PaginationLink>) {
  return (
    <PaginationLink
      aria-label="Go to previous page"
      size="default"
      className={cn("gap-1 px-2.5 sm:pl-2.5", className)}
      {...props}
    >
      <ChevronLeftIcon />
      <span className="hidden sm:block">Previous</span>
    </PaginationLink>
  );
}

function PaginationNext({
  className,
  ...props
}: React.ComponentProps<typeof PaginationLink>) {
  return (
    <PaginationLink
      aria-label="Go to next page"
      size="default"
      className={cn("gap-1 px-2.5 sm:pr-2.5", className)}
      {...props}
    >
      <span className="hidden sm:block">Next</span>
      <ChevronRightIcon />
    </PaginationLink>
  );
}

function PaginationEllipsis({
  className,
  ...props
}: React.ComponentProps<"span">) {
  return (
    <span
      aria-hidden
      data-slot="pagination-ellipsis"
      className={cn("flex size-9 items-center justify-center", className)}
      {...props}
    >
      <MoreHorizontalIcon className="size-4" />
      <span className="sr-only">More pages</span>
    </span>
  );
}

// ─── 高级 Pagination 组件（ClawPro 规范 §3 / §12）──────────────────────────

/**
 * 计算页码序列（含省略号占位）。
 * - totalPages <= 7：全部展示
 * - > 7：始终展示首尾页 + 当前页 ± 1，中间用 "..." 折叠
 */
function buildPages(current: number, totalPages: number): (number | "...")[] {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }
  const pages: (number | "...")[] = [1];
  const start = Math.max(2, current - 1);
  const end = Math.min(totalPages - 1, current + 1);
  if (start > 2) pages.push("...");
  for (let i = start; i <= end; i++) pages.push(i);
  if (end < totalPages - 1) pages.push("...");
  pages.push(totalPages);
  return pages;
}

type ChangeEventHandler<T extends HTMLElement, E extends Element = Element> = (
  page: number,
  pageSize: number,
) => void;

export interface PaginationProps {
  /** 总条数 */
  total: number;
  /** 当前页（从 1 开始） */
  current: number;
  /** 每页条数 */
  pageSize: number;
  /** 翻页 / 改页大小时的回调 */
  onChange?: ChangeEventHandler<HTMLElement>;
  /** 总数文案；传 false 不显示；默认 `共 N 条` */
  showTotal?: ((total: number) => React.ReactNode) | false;
  /** 是否显示页大小切换器 */
  showSizeChanger?: boolean;
  /** 页大小选项 */
  pageSizeOptions?: readonly number[];
  /** 尺寸：default 28px / small 24px */
  size?: "default" | "small";
  /** 模式：default 完整页码 / simple 仅前后页 + 当前页指示（弹窗/浮层用） */
  mode?: "default" | "simple";
  /** @deprecated 使用 `mode="simple"` 替代；兼容旧版 API */
  simple?: boolean;
  /** 仅单页时隐藏分页器 */
  hideOnSinglePage?: boolean;
  /** 容器 className */
  className?: string;
}

/**
 * Pagination — ClawPro 标准分页器
 *
 * 视觉规范（component-specs/pagination.md §3）：
 *   - 字号 12px，按钮 28×28（small 24×24），圆角 8px
 *   - Active：白底 + 蓝描边 + 蓝字（不做实心色块）
 *   - Hover：弱灰；Disabled：禁用色 + cursor-not-allowed
 *   - 总数文案 token `var(--text-muted)` / `#A3A3A3`
 *
 * 弹窗/浮层场景：
 *   <Pagination mode="simple" size="small" ... />
 */
function Pagination({
  total,
  current,
  pageSize,
  onChange,
  showTotal = (t) => `共 ${t} 条`,
  showSizeChanger = false,
  pageSizeOptions = [10, 20, 50, 100],
  size = "default",
  mode = "default",
  simple,
  hideOnSinglePage = false,
  className,
}: PaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const safeCurrent = Math.min(Math.max(1, current), totalPages);

  if (hideOnSinglePage && totalPages <= 1 && !showSizeChanger) {
    if (showTotal !== false) {
      return (
        <nav aria-label="分页" className={cn("flex items-center text-xs text-[var(--text-muted)]", className)}>
          <span>{showTotal(total)}</span>
        </nav>
      );
    }
    return null;
  }

  const isSmall = size === "small";
  const isSimple = mode === "simple" || simple === true;
  const btnH = isSmall ? "h-6 min-w-[24px]" : "h-7 min-w-[28px]";
  const pages = buildPages(safeCurrent, totalPages);

  const go = (p: number) => {
    if (p < 1 || p > totalPages || p === safeCurrent) return;
    onChange?.(p, pageSize);
  };

  const handleSizeChange = (newSize: number) => {
    onChange?.(1, newSize);
  };

  return (
    <nav aria-label="分页" className={cn("flex items-center gap-2 text-xs", className)}>
      {/* 左侧：总数文案 */}
      {showTotal !== false && (
        <span className="text-[var(--text-muted)]">{showTotal(total)}</span>
      )}

      {/* 右侧：分页按钮组 */}
      <div className="flex items-center gap-1">
        {/* 上一页 */}
        <button
          type="button"
          aria-label="上一页"
          disabled={safeCurrent <= 1}
          onClick={() => go(safeCurrent - 1)}
          className={cn(
            "inline-flex items-center justify-center rounded-lg border border-[var(--border)] bg-white text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-grey-hover)] disabled:opacity-40 disabled:cursor-not-allowed",
            btnH,
          )}
        >
          <ChevronLeftIcon className="size-3.5" />
        </button>

        {/* 页码 / simple 模式 */}
        {isSimple ? (
          <span className={cn("inline-flex items-center justify-center px-2 text-[var(--text-muted)] select-none", btnH)}>
            {safeCurrent} / {totalPages}
          </span>
        ) : (
          pages.map((p, idx) =>
            p === "..." ? (
              <span
                key={`ellipsis-${idx}`}
                className={cn("inline-flex items-center justify-center text-[var(--text-muted)] select-none", btnH)}
              >
                ···
              </span>
            ) : (
              <button
                key={p}
                type="button"
                aria-label={`第 ${p} 页`}
                aria-current={p === safeCurrent ? "page" : undefined}
                onClick={() => go(p as number)}
                className={cn(
                  "inline-flex items-center justify-center rounded-lg border px-1.5 transition-colors",
                  btnH,
                  p === safeCurrent
                    ? "border-[var(--text-brand)] bg-white text-[var(--text-brand)] font-medium"
                    : "border-[var(--border)] bg-white text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)]",
                )}
              >
                {p}
              </button>
            )
          )
        )}

        {/* 下一页 */}
        <button
          type="button"
          aria-label="下一页"
          disabled={safeCurrent >= totalPages}
          onClick={() => go(safeCurrent + 1)}
          className={cn(
            "inline-flex items-center justify-center rounded-lg border border-[var(--border)] bg-white text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-grey-hover)] disabled:opacity-40 disabled:cursor-not-allowed",
            btnH,
          )}
        >
          <ChevronRightIcon className="size-3.5" />
        </button>
      </div>

      {/* 页大小切换器 */}
      {showSizeChanger && (
        <div className="flex items-center gap-1">
          <Select
            value={String(pageSize)}
            onValueChange={(val) => handleSizeChange(Number(val))}
          >
            <SelectTrigger
              className={cn(
                "rounded-lg border border-[var(--border)] bg-white px-2 !py-0 text-xs text-[var(--text-title)] outline-none transition-colors hover:border-[var(--text-brand)] focus:border-[var(--text-brand)] gap-1 min-w-[80px]",
                isSmall ? "!h-6" : "!h-7",
              )}
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent align="end" className="min-w-[80px]">
              {pageSizeOptions.map((opt) => (
                <SelectItem key={opt} value={String(opt)}>
                  {opt} 条/页
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}
    </nav>
  );
}

// ─── DialogPagination（弹窗/浮层内简化分页器）────────────────────────────────

export interface DialogPaginationProps {
  /** 总条数 */
  total: number;
  /** 当前页（从 1 开始） */
  currentPage: number;
  /** 总页数 */
  totalPages: number;
  /** 上一页回调 */
  onPrevPage: () => void;
  /** 下一页回调 */
  onNextPage: () => void;
  /** 容器 className */
  className?: string;
}

/**
 * DialogPagination — 弹窗/浮层内使用的简化分页器
 *
 * 仅显示"共 N 条"文案 + 上一页/下一页按钮 + 当前页/总页数指示。
 * 适用于弹窗内列表分页场景。
 */
function DialogPagination({
  total,
  currentPage,
  totalPages,
  onPrevPage,
  onNextPage,
  className,
}: DialogPaginationProps) {
  return (
    <nav
      aria-label="分页"
      className={cn("flex items-center gap-2 text-xs", className)}
    >
      <span className="text-[var(--text-muted)]">共 {total} 条</span>
      <div className="flex items-center gap-1 ml-auto">
        <button
          type="button"
          aria-label="上一页"
          disabled={currentPage <= 1}
          onClick={onPrevPage}
          className="inline-flex items-center justify-center rounded-lg border border-[var(--border)] bg-white text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-grey-hover)] disabled:opacity-40 disabled:cursor-not-allowed h-6 min-w-[24px]"
        >
          <ChevronLeftIcon className="size-3.5" />
        </button>
        <span className="inline-flex items-center justify-center px-2 text-[var(--text-muted)] select-none h-6">
          {currentPage} / {totalPages}
        </span>
        <button
          type="button"
          aria-label="下一页"
          disabled={currentPage >= totalPages}
          onClick={onNextPage}
          className="inline-flex items-center justify-center rounded-lg border border-[var(--border)] bg-white text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-grey-hover)] disabled:opacity-40 disabled:cursor-not-allowed h-6 min-w-[24px]"
        >
          <ChevronRightIcon className="size-3.5" />
        </button>
      </div>
    </nav>
  );
}

export {
  Pagination,
  DialogPagination,
  PaginationNav,
  PaginationContent,
  PaginationLink,
  PaginationItem,
  PaginationPrevious,
  PaginationNext,
  PaginationEllipsis,
};
