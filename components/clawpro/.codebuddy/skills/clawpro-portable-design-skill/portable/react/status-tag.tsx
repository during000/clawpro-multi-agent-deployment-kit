/**
 * Portable StatusTag — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 StatusTag 时的可移植兜底实现。
 *  - 不依赖 shadcn / Radix / Tailwind；样式由 portable/css/status-tag.css 提供。
 *  - 形态唯一（spec/component-specs/status-tag.md §3.1）：
 *      12px / Medium / 彩色纯文本（无底色 / 无边框 / 无圆点）。
 *  - 5 主语义色（不可扩展）：blue / green / red / orange / gray
 *      blue   #1447E6  信息 / 进行中
 *      green  #008236  成功 / 在线 / 已完成
 *      red    #DC2626  失败 / 离线 / 错误
 *      orange #B45309  警告 / 即将到期（amber 系，不要用 orange）
 *      gray   #0A0A0A  中性 / 未启用 / 草稿
 *  - 非交互元素：不响应 hover / focus / active，不要套 onClick；如需可点击，
 *    在外层包 <button>/<a>。
 *  - 颜色不是唯一信息载体：必须配合文字（运行中 / 异常）。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/status-tag.css";
 *
 * 用法：
 *   <PortableStatusTag variant="green">运行中</PortableStatusTag>
 *   <PortableStatusTag variant="red">异常</PortableStatusTag>
 *   <PortableStatusTag variant="orange" icon={<AlertIcon />}>即将到期</PortableStatusTag>
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export type PortableStatusVariant = "blue" | "green" | "red" | "orange" | "gray";

export interface PortableStatusTagProps
  extends Omit<React.HTMLAttributes<HTMLSpanElement>, "color"> {
  variant?: PortableStatusVariant;
  /** 可选 12×12 图标（currentColor，与文字 gap-1） */
  icon?: React.ReactNode;
}

export const PortableStatusTag = React.forwardRef<
  HTMLSpanElement,
  PortableStatusTagProps
>(({ variant = "gray", icon, className = "", children, ...props }, ref) => {
  const merged = ["cp-status-tag", `cp-status-tag--${variant}`, className]
    .filter(Boolean)
    .join(" ");
  return (
    <span
      ref={ref}
      data-slot="status-tag"
      data-variant={variant}
      role="presentation"
      className={merged}
      {...props}
    >
      {icon ? (
        <span aria-hidden="true" className="cp-status-tag__icon">
          {icon}
        </span>
      ) : null}
      <span>{children}</span>
    </span>
  );
});
PortableStatusTag.displayName = "PortableStatusTag";
