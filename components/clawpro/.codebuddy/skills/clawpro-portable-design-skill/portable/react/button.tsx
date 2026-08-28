/**
 * Portable Button — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 Button 时的可移植兜底实现。
 *  - 不依赖 shadcn / cva / class-variance-authority / Tailwind。
 *  - 颜色 / 尺寸 / 变体全部由 portable/css/button.css 提供。
 *  - 6 个核心 variant 与 demo 仓 1:1 对齐：
 *      claw-primary    管控端主按钮（品牌黑底 + 白字）
 *      claw-outline    管控端次级按钮（白底 + 蓝灰描边）
 *      destructive     危险主按钮（红底 + 白字）
 *      outline-destructive 次级危险（红描边 + 红字 + 浅粉底）
 *      link            内联文字链接（品牌蓝）
 *      dialog-confirm  弹窗确认按钮（同 claw-primary，提供别名给弹窗调用方）
 *  - 3 档尺寸：sm（h-32） / md（h-36，默认） / lg（h-40）
 *  - 圆角统一 4px（管控端铁律）
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/button.css";
 *
 * 用法：
 *   <PortableButton>添加</PortableButton>
 *   <PortableButton variant="claw-outline" size="sm">取消</PortableButton>
 *   <PortableButton variant="destructive">确认删除</PortableButton>
 *   <PortableButton variant="link">查看详情</PortableButton>
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export type PortableButtonVariant =
  | "claw-primary"
  | "claw-outline"
  | "destructive"
  | "outline-destructive"
  | "link"
  | "dialog-confirm";

export type PortableButtonSize = "sm" | "md" | "lg";

export interface PortableButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: PortableButtonVariant;
  size?: PortableButtonSize;
}

export const PortableButton = React.forwardRef<HTMLButtonElement, PortableButtonProps>(
  (
    {
      variant = "claw-primary",
      size = "md",
      className = "",
      type = "button",
      ...props
    },
    ref
  ) => {
    const cls = [
      "cp-btn",
      `cp-btn--${size}`,
      `cp-btn--${variant}`,
      className,
    ]
      .filter(Boolean)
      .join(" ");
    return <button ref={ref} type={type} className={cls} {...props} />;
  }
);
PortableButton.displayName = "PortableButton";
