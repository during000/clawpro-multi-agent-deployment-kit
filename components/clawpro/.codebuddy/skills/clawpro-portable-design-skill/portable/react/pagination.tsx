/**
 * Portable Pagination — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 Pagination 时的可移植兜底实现。
 *  - 不依赖 shadcn / Radix / Tailwind；样式由 portable/css/pagination.css 提供。
 *  - 视觉规范（spec/component-specs/pagination.md §3）：
 *      字号 12px / 与表格口径对齐；按钮 28×28（small 24×24）
 *      圆角 8px（pagination 唯一例外，已在 component-spec 标 allow-radius）
 *      active：白底 + 蓝描边 + 蓝字（不做实心色块）
 *      hover：弱灰 hover；disabled：禁用色 + cursor-not-allowed
 *      总数文案 `共 N 条记录` 用 var(--cp-text-muted)
 *  - 受控组件：current / pageSize 由调用方维护；onChange 拿到下一页 page 号。
 *  - "page items" 渲染策略（与 demo 仓一致）：
 *      <= 7 页：全部展示
 *      > 7 页：1 ... [current-1, current, current+1] ... last
 *  - mode="simple"：紧凑模式（弹窗 / 浮层 / 窄区），仅 前后页 + 当前页指示 + 总数，
 *      不渲染完整页码（spec/component-specs/pagination.md §5 / §12.5）。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/pagination.css";
 *
 * 用法：
 *   <PortablePagination
 *     total={245}
 *     current={1}
 *     pageSize={10}
 *     onChange={setCurrent}
 *     showTotal={(t) => `共 ${t} 条记录`}
 *   />
 *
 *   // 弹窗 / 浮层等紧凑场景：仅前后页 + 当前页指示 + 总数
 *   <PortablePagination
 *     mode="simple"
 *     size="small"
 *     total={245}
 *     current={1}
 *     pageSize={10}
 *     onChange={setCurrent}
 *     showTotal={(t) => `共 ${t} 条`}
 *   />
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export interface PortablePaginationProps {
  /** 总条数 */
  total: number;
  /** 当前页（从 1 开始） */
  current: number;
  /** 每页条数 */
  pageSize: number;
  /** 翻页时的回调（page 从 1 开始） */
  onChange?: (page: number) => void;
  /** 总数文案；传 false 表示不显示 */
  showTotal?: ((total: number) => React.ReactNode) | false;
  /** 紧凑尺寸：按钮 24×24（默认 28×28） */
  size?: "default" | "small";
  /** 模式：default 完整页码；simple 仅前后页 + 当前页指示 + 总数（紧凑场景 / 弹窗） */
  mode?: "default" | "simple";
  className?: string;
}

/* 计算页码序列：返回 number 或 "..." */
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

export function PortablePagination({
  total,
  current,
  pageSize,
  onChange,
  showTotal = (t) => `共 ${t} 条记录`,
  size = "default",
  mode = "default",
  className = "",
}: PortablePaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const safeCurrent = Math.min(Math.max(1, current), totalPages);
  const pages = buildPages(safeCurrent, totalPages);
  const isSmall = size === "small";
  const isSimple = mode === "simple";
  const btnSizeCls = isSmall ? "cp-pagination__btn--sm" : "";
  const ellipsisCls = [
    "cp-pagination__ellipsis",
    isSmall && "cp-pagination__ellipsis--sm",
  ]
    .filter(Boolean)
    .join(" ");
  const simpleCls = [
    "cp-pagination__simple",
    isSmall && "cp-pagination__simple--sm",
  ]
    .filter(Boolean)
    .join(" ");

  const go = (p: number) => {
    if (p < 1 || p > totalPages || p === safeCurrent) return;
    onChange?.(p);
  };

  const prevDisabled = safeCurrent <= 1;
  const nextDisabled = safeCurrent >= totalPages;

  const merged = ["cp-pagination", className].filter(Boolean).join(" ");

  const btnCls = (extra?: string) =>
    ["cp-pagination__btn", btnSizeCls, extra].filter(Boolean).join(" ");

  return (
    <nav aria-label="分页" className={merged}>
      {showTotal !== false && (
        <span className="cp-pagination__total">{showTotal(total)}</span>
      )}
      <div className="cp-pagination__list">
        <button
          type="button"
          aria-label="上一页"
          disabled={prevDisabled}
          onClick={() => go(safeCurrent - 1)}
          className={btnCls()}
        >
          ‹
        </button>
        {isSimple ? (
          <span className={simpleCls} aria-current="page">
            {safeCurrent} / {totalPages}
          </span>
        ) : (
          pages.map((p, idx) =>
            p === "..." ? (
              <span key={`ellipsis-${idx}`} className={ellipsisCls}>
                …
              </span>
            ) : (
              <button
                key={p}
                type="button"
                aria-label={`第 ${p} 页`}
                aria-current={p === safeCurrent ? "page" : undefined}
                onClick={() => go(p)}
                className={btnCls(p === safeCurrent ? "cp-pagination__btn--active" : "")}
              >
                {p}
              </button>
            )
          )
        )}
        <button
          type="button"
          aria-label="下一页"
          disabled={nextDisabled}
          onClick={() => go(safeCurrent + 1)}
          className={btnCls()}
        >
          ›
        </button>
      </div>
    </nav>
  );
}
