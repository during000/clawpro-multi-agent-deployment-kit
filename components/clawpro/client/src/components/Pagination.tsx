/**
 * Pagination — 通用智能分页器组件
 * 页码过多时自动折叠为省略号，始终显示首尾页和当前页附近的页码。
 */
import { ChevronLeft, ChevronRight } from "lucide-react";

interface PaginationProps {
  /** 当前页（从 1 开始） */
  page: number;
  /** 总记录数 */
  total: number;
  /** 每页条数 */
  pageSize: number;
  /** 页码变更回调 */
  onChange: (page: number) => void;
  /** 总记录数左侧标签，默认 "共 N 条记录" */
  totalLabel?: string;
  /** 简洁模式：仅显示当前页码（不可点击），隐藏其余页码 */
  simpleMode?: boolean;
  /** 自定义容器 className，覆盖默认 px-6 等样式 */
  className?: string;
}

/** 计算需要显示的页码列表（包含 -1 表示省略号） */
function getPageNumbers(current: number, totalPages: number): number[] {
  // 总页数 <= 7 时全部显示
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }

  const pages: number[] = [];

  // 始终显示第 1 页
  pages.push(1);

  if (current <= 3) {
    // 靠近左边：1 2 3 4 5 ... last
    pages.push(2, 3, 4, 5);
    pages.push(-1); // 省略号
    pages.push(totalPages);
  } else if (current >= totalPages - 2) {
    // 靠近右边：1 ... last-4 last-3 last-2 last-1 last
    pages.push(-1);
    pages.push(totalPages - 4, totalPages - 3, totalPages - 2, totalPages - 1, totalPages);
  } else {
    // 中间：1 ... current-1 current current+1 ... last
    pages.push(-1);
    pages.push(current - 1, current, current + 1);
    pages.push(-2); // 第二个省略号（用 -2 区分 key）
    pages.push(totalPages);
  }

  return pages;
}

export default function Pagination({ page, total, pageSize, onChange, totalLabel, simpleMode = false, className }: PaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const safePage = Math.min(page, totalPages);

  if (totalPages <= 1) {
    return (
      <div className={className ?? "px-6 py-3 border-t border-gray-50 text-xs text-gray-400"}>
        {totalLabel ?? `共 ${total} 条记录`}
      </div>
    );
  }

  const pageNumbers = getPageNumbers(safePage, totalPages);

  return (
    <div className={className ?? "px-6 py-3 border-t border-gray-50 flex items-center justify-between"}>
      <span className="text-xs text-gray-400">
        {totalLabel ?? `共 ${total} 条记录`}
        {total > 0 && `，第 ${safePage} / ${totalPages} 页`}
      </span>
      <div className="flex items-center gap-1">
        {/* 上一页 */}
        <button
          onClick={() => onChange(Math.max(1, safePage - 1))}
          disabled={safePage <= 1}
          className="w-7 h-7 flex items-center justify-center rounded-md border border-gray-200 text-gray-400 hover:text-blue-500 hover:border-blue-300 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        >
          <ChevronLeft className="w-3.5 h-3.5" />
        </button>

        {/* 页码 */}
        {simpleMode ? (
          /* 简洁模式：仅显示当前页码，不可点击 */
          <span className="w-7 h-7 flex items-center justify-center rounded-md text-xs font-medium select-none bg-blue-500 text-white border border-blue-500">
            {safePage}
          </span>
        ) : (
          /* 普通模式：显示完整页码，支持点击 */
          pageNumbers.map((p) =>
            p < 0 ? (
              <span
                key={`ellipsis-${p}`}
                className="w-7 h-7 flex items-center justify-center text-xs text-gray-400 select-none"
              >
                ···
              </span>
            ) : (
              <button
                key={p}
                onClick={() => onChange(p)}
                className={`w-7 h-7 flex items-center justify-center rounded-md text-xs font-medium transition-colors ${
                  p === safePage
                    ? "bg-blue-500 text-white border border-blue-500"
                    : "border border-gray-200 text-gray-500 hover:border-blue-300 hover:text-blue-500"
                }`}
              >
                {p}
              </button>
            )
          )
        )}

        {/* 下一页 */}
        <button
          onClick={() => onChange(Math.min(totalPages, safePage + 1))}
          disabled={safePage >= totalPages}
          className="w-7 h-7 flex items-center justify-center rounded-md border border-gray-200 text-gray-400 hover:text-blue-500 hover:border-blue-300 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        >
          <ChevronRight className="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  );
}
