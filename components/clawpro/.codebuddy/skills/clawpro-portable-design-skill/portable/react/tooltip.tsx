/**
 * Portable Tooltip — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 Tooltip 时的可移植兜底实现。
 *  - 不依赖 @radix-ui / shadcn / Tailwind；样式由 portable/css/tooltip.css 提供。
 *  - 受控/非受控的 hover + focus 触发（无定位库，纯 CSS 绝对定位，top 优先）。
 *  - 视觉规范（component-specs/tooltip.md §3）：
 *      深黑底 #020617 + 白字；4px 圆角；px-3 py-1.5；
 *      12px / leading-relaxed；最大宽度 240px（超出换行）。
 *  - 只放短文本（1~2 行）；不要放可交互内容（用 Popover / HoverCard）；
 *    不要 p-0 重置 padding；不要浅色底。
 *  - disabled 元素需在外层包一层可 hover 的 span 作为触发。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/tooltip.css";
 *
 * 用法：
 *   <PortableTooltip content="每日配额上限 1000 次">
 *     <InfoIcon />
 *   </PortableTooltip>
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export type PortableTooltipSide = "top" | "bottom" | "left" | "right";

export interface PortableTooltipProps {
  /** 触发元素 */
  children: React.ReactNode;
  /** 提示短文本（仅文本，勿放可交互内容） */
  content: React.ReactNode;
  /** 出现方位，默认 top */
  side?: PortableTooltipSide;
  className?: string;
  /** 浮层自定义 className */
  contentClassName?: string;
}

let tooltipSeq = 0;

export function PortableTooltip({
  children,
  content,
  side = "top",
  className = "",
  contentClassName = "",
}: PortableTooltipProps) {
  const [open, setOpen] = React.useState(false);
  const idRef = React.useRef<string>();
  if (!idRef.current) idRef.current = `cp-tooltip-${++tooltipSeq}`;

  const show = () => setOpen(true);
  const hide = () => setOpen(false);

  const merged = ["cp-tooltip", className].filter(Boolean).join(" ");
  const contentCls = [
    "cp-tooltip__content",
    `cp-tooltip__content--${side}`,
    contentClassName,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <span
      className={merged}
      onMouseEnter={show}
      onMouseLeave={hide}
      onFocus={show}
      onBlur={hide}
      aria-describedby={open ? idRef.current : undefined}
    >
      {children}
      {open && (
        <span id={idRef.current} role="tooltip" className={contentCls}>
          {content}
        </span>
      )}
    </span>
  );
}
