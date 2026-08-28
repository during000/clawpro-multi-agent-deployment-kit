/**
 * Portable Loading / Progress — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构加载/进度组件时的可移植兜底实现。
 *  - 不依赖 @radix-ui / shadcn / Tailwind；样式由 portable/css/loading-progress.css 提供。
 *  - 视觉规范（component-specs/loading-progress.md §3）：
 *      Spinner：16 / 20px，品牌蓝（默认）或弱灰（muted），按钮内用小尺寸。
 *      Skeleton：贴合最终布局尺寸，宽高由 className / style 指定。
 *      Progress：6px 高，品牌蓝进度 + 弱灰轨道（单色，不渐变彩色）。
 *      TableSkeleton：骨架行高 54px，3 段灰条接近真实列宽。
 *  - 局部加载优先，不要整页遮罩锁死；按钮 loading 只锁按钮、保留文案；
 *    Skeleton 不要用一块大灰条替代复杂布局；进度条无量化时改用 Spinner。
 *  - 轻量动效，reduced-motion 自动降级（见 css）。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/loading-progress.css";
 *
 * 用法：
 *   <PortableSpinner size="md" />
 *   <PortableSkeleton style={{ height: 16, width: 128 }} />
 *   <PortableProgress value={48} />
 *   <PortableTableSkeleton rows={5} />
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

/* ───────────── PortableSpinner ───────────── */

export interface PortableSpinnerProps
  extends Omit<React.SVGProps<SVGSVGElement>, "ref"> {
  /** sm=16px（按钮内）/ md=20px，默认 sm */
  size?: "sm" | "md";
  /** 弱灰配色（非品牌蓝） */
  muted?: boolean;
}

export function PortableSpinner({
  size = "sm",
  muted,
  className = "",
  ...props
}: PortableSpinnerProps) {
  const merged = [
    "cp-spinner",
    `cp-spinner--${size}`,
    muted && "cp-spinner--muted",
    className,
  ]
    .filter(Boolean)
    .join(" ");
  return (
    <svg
      role="status"
      aria-label="Loading"
      viewBox="0 0 24 24"
      fill="none"
      className={merged}
      {...props}
    >
      <circle
        cx="12"
        cy="12"
        r="9"
        stroke="currentColor"
        strokeWidth="2.5"
        opacity="0.2"
      />
      <path
        d="M21 12a9 9 0 0 0-9-9"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
      />
    </svg>
  );
}

/* ───────────── PortableSkeleton ───────────── */

export type PortableSkeletonProps = React.HTMLAttributes<HTMLDivElement>;

export function PortableSkeleton({ className = "", ...props }: PortableSkeletonProps) {
  const merged = ["cp-skeleton", className].filter(Boolean).join(" ");
  return <div data-slot="skeleton" className={merged} {...props} />;
}

/* ───────────── PortableProgress ───────────── */

export interface PortableProgressProps
  extends Omit<React.HTMLAttributes<HTMLDivElement>, "children"> {
  /** 0~100，缺省/越界自动夹取 */
  value?: number;
}

export function PortableProgress({
  value = 0,
  className = "",
  ...props
}: PortableProgressProps) {
  const pct = Math.max(0, Math.min(100, value));
  const merged = ["cp-progress", className].filter(Boolean).join(" ");
  return (
    <div
      role="progressbar"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={pct}
      className={merged}
      {...props}
    >
      <div className="cp-progress__bar" style={{ width: `${pct}%` }} />
    </div>
  );
}

/* ───────────── PortableTableSkeleton ───────────── */

export interface PortableTableSkeletonProps
  extends React.HTMLAttributes<HTMLDivElement> {
  /** 骨架行数，默认 5 */
  rows?: number;
}

export function PortableTableSkeleton({
  rows = 5,
  className = "",
  ...props
}: PortableTableSkeletonProps) {
  const merged = ["cp-table-skeleton", className].filter(Boolean).join(" ");
  return (
    <div className={merged} {...props}>
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="cp-table-skeleton__row">
          <span className="cp-table-skeleton__cell cp-table-skeleton__cell--a" />
          <span className="cp-table-skeleton__cell cp-table-skeleton__cell--b" />
          <span className="cp-table-skeleton__cell cp-table-skeleton__cell--c" />
        </div>
      ))}
    </div>
  );
}
