/**
 * Portable Avatar — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 Avatar 时的可移植兜底实现。
 *  - 不依赖 @radix-ui / shadcn / Tailwind；样式由 portable/css/avatar.css 提供。
 *  - 视觉规范（component-specs/avatar.md §3）：
 *      圆形裁切（rounded-full）；4 档尺寸 24 / 32 / 40 / 48（默认 32）；
 *      Fallback 背景 var(--cp-bg-subtle)、文字 14px medium var(--cp-text-muted)；
 *      无图片时显示首字母缩写（大写，取前 2 位）。
 *  - 不要方形头像 / 自由尺寸 / fallback 渐变彩底 / emoji。
 *  - Avatar 不承担交互语义；需点击在外层包 <button> 并加 focus ring。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/avatar.css";
 *
 * 用法：
 *   <PortableAvatar name="张三" src="/u/1.png" size="md" />
 *   <PortableAvatarGroup max={3} size="md">
 *     <PortableAvatar name="A" /> <PortableAvatar name="B" /> ...
 *   </PortableAvatarGroup>
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export type PortableAvatarSize = "sm" | "md" | "lg" | "xl";

export interface PortableAvatarProps
  extends Omit<React.HTMLAttributes<HTMLSpanElement>, "children"> {
  /** 显示名（用于 alt + 首字母 fallback） */
  name: string;
  /** 头像图片地址，缺省走首字母 fallback */
  src?: string;
  /** 4 档标准尺寸，默认 md(32px) */
  size?: PortableAvatarSize;
}

function initialsOf(name: string): string {
  return name.trim().slice(0, 2).toUpperCase();
}

export const PortableAvatar = React.forwardRef<HTMLSpanElement, PortableAvatarProps>(
  ({ name, src, size = "md", className = "", ...props }, ref) => {
    const merged = ["cp-avatar", `cp-avatar--${size}`, className]
      .filter(Boolean)
      .join(" ");
    return (
      <span ref={ref} data-slot="avatar" className={merged} {...props}>
        {src ? (
          <img className="cp-avatar__img" src={src} alt={name} />
        ) : (
          <span className="cp-avatar__fallback" aria-label={name}>
            {initialsOf(name)}
          </span>
        )}
      </span>
    );
  }
);
PortableAvatar.displayName = "PortableAvatar";

/* ───────────── PortableAvatarGroup ─────────────
 * 叠放头像组，超过 max 折叠为 +N。
 */

export interface PortableAvatarGroupProps
  extends React.HTMLAttributes<HTMLSpanElement> {
  /** 最多展示几个，其余折叠为 +N */
  max?: number;
  /** 统一应用到溢出 +N 徽标的尺寸（仅影响 +N 容器视觉） */
  size?: PortableAvatarSize;
  children: React.ReactNode;
}

export function PortableAvatarGroup({
  max,
  size = "md",
  className = "",
  children,
  ...props
}: PortableAvatarGroupProps) {
  const items = React.Children.toArray(children);
  const visible = max != null ? items.slice(0, max) : items;
  const overflow = max != null ? items.length - visible.length : 0;
  const merged = ["cp-avatar-group", className].filter(Boolean).join(" ");
  return (
    <span className={merged} {...props}>
      {visible}
      {overflow > 0 && (
        <span
          className={`cp-avatar-group__more cp-avatar--${size}`}
          aria-label={`还有 ${overflow} 个`}
        >
          +{overflow}
        </span>
      )}
    </span>
  );
}
