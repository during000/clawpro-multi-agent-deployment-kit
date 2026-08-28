/**
 * Portable Badge — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 Badge 时的可移植兜底实现。
 *  - 不依赖 shadcn / Radix / cva / Tailwind。纯 React + CSS。
 *  - 颜色 / 变体全部由 portable/css/badge.css 提供。
 *  - 与 demo 仓 client/src/components/ui/badge.tsx 1:1 对齐（spec §3）：
 *      4 个标准 variant：
 *        default      黑底白字（#0A0A0A / #FFFFFF），强调分类，少用
 *        secondary    浅灰底深字（#F5F5F5 / #0A0A0A），中性默认，最常用
 *        outline      白底描边（#FFFFFF / gray-200 / #0A0A0A），版本号最常用
 *        destructive  浅红底红字（red-100/60 / red-600），风险类分类
 *      4 个 Custom Color（设置后覆盖 variant 视觉，只保留尺寸 / 字号）：
 *        blue   #E8ECFE / #1447E6
 *        green  green-50 / green-700
 *        purple purple-50 / purple-700
 *        red    red-50 / red-700
 *  - 几何固定：rounded-full + px-2.5 py-0.5 + 12px Regular。
 *  - 与 StatusTag 严格分工：Badge 表"分类"，StatusTag 表"状态"。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/badge.css";
 *
 * 用法：
 *   <PortableBadge>默认</PortableBadge>                       // default 黑底白字
 *   <PortableBadge variant="secondary">企业版</PortableBadge>  // 中性默认
 *   <PortableBadge variant="outline">v2.1.0</PortableBadge>    // 版本号
 *   <PortableBadge variant="destructive">已废弃</PortableBadge>
 *   <PortableBadge color="blue">AI</PortableBadge>             // Custom Color
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export type PortableBadgeVariant =
  | "default"
  | "secondary"
  | "outline"
  | "destructive";

export type PortableBadgeColor = "blue" | "green" | "purple" | "red";

export interface PortableBadgeProps
  extends React.HTMLAttributes<HTMLSpanElement> {
  /** 标准变体（4 个）。设置 color 后将被覆盖。 */
  variant?: PortableBadgeVariant;
  /** Custom Color（4 色）。设置后覆盖 variant 视觉，仅保留尺寸 / 字号。 */
  color?: PortableBadgeColor;
}

export const PortableBadge = React.forwardRef<HTMLSpanElement, PortableBadgeProps>(
  ({ variant = "default", color, className = "", children, ...props }, ref) => {
    const cls = [
      "cp-badge",
      color ? `cp-badge--color-${color}` : `cp-badge--${variant}`,
      className,
    ]
      .filter(Boolean)
      .join(" ");

    return (
      <span ref={ref} data-slot="badge" data-color={color} className={cls} {...props}>
        {children}
      </span>
    );
  }
);
PortableBadge.displayName = "PortableBadge";
