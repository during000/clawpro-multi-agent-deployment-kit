/**
 * NavIconButton - 顶部导航右侧的图标 / 图标+文本按钮
 *
 * 设计来源：Figma 「图标文本」组件（节点 297:3285、363:5028 默认；1077:34989 胶囊变体）
 * 视觉规范：
 *   默认（图标 / 图标+文字）：
 *     - padding: 6px 8px、圆角 4px、透明底
 *     - 文本：14 / line-height 22 / color #020617
 *     - hover：bg #F5F5F5 + text #020617
 *     - 红点：4x4，绝对定位，#E85C5C
 *   pill 胶囊变体（如「切换管控端」按钮，Figma 1077:34989）：
 *     - padding: 6px 12px、圆角 20px（胶囊）
 *     - 默认底：透明
 *     - hover：rgba(219, 221, 228, 0.32) 浅灰
 *
 * 用法：
 *   <NavIconButton icon={<HelpIcon />} title="使用指南" />
 *   <NavIconButton icon={<BellIcon />} title="消息通知" showDot />
 *   <NavIconButton icon={<SwitchAdminIcon />} label="切换管控端" pill />
 */
import React from "react";

export interface NavIconButtonProps
  extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, "title"> {
  /** 图标节点（推荐使用 currentColor 着色的 SVG，便于 hover 跟随） */
  icon: React.ReactNode;
  /** 文字标签（不传则只显示图标） */
  label?: string;
  /** title 提示（无 label 时建议传） */
  title?: string;
  /** 是否显示右上红点 */
  showDot?: boolean;
  /** 文字后的徽章插槽（如未读数）— 与 label 同处按钮内部，hover 背景一并覆盖 */
  badge?: React.ReactNode;
  /**
   * 胶囊形态（pill）：圆角 20px、padding 6px 12px、默认带浅灰底；
   * 适用于「切换管控端」这类需要强调的图标+文字组合按钮（Figma 1077:34989）。
   */
  pill?: boolean;
  /** 内部 className */
  className?: string;
}

const NavIconButton = React.forwardRef<HTMLButtonElement, NavIconButtonProps>(
  function NavIconButton(
    { icon, label, title, showDot, badge, pill = false, className = "", ...rest },
    ref
  ) {
    // 形态分支：默认 = 4px 圆角无底色；pill = 20px 胶囊带浅灰底
    const shape = pill
      ? "rounded-full px-3 py-[6px] h-8 hover:bg-[rgba(219,221,228,0.32)]"
      : "rounded-full px-2 py-[6px] h-8 hover:bg-[rgba(219,221,228,0.32)]";

    return (
      <button
        ref={ref}
        type="button"
        title={title ?? label}
        {...rest}
        className={[
          "relative inline-flex items-center gap-2",
          shape,
          "text-[14px] leading-[22px]",
          "text-[#020617]/90 hover:text-[#020617]",
          "transition-colors flex-shrink-0 whitespace-nowrap nav-icon-btn",
          className,
        ].join(" ")}
      >
        <span className="inline-flex items-center justify-center flex-shrink-0">
          {icon}
        </span>
        {label && <span className="truncate min-w-0 nav-btn-label">{label}</span>}
        {badge && (
          <span className="inline-flex items-center flex-shrink-0 nav-btn-label">
            {badge}
          </span>
        )}
        {showDot && (
          <span
            aria-hidden
            className="absolute"
            style={{
              top: 6,
              right: 6,
              width: 4,
              height: 4,
              borderRadius: "50%",
              background: "#E85C5C",
            }}
          />
        )}
      </button>
    );
  }
);

export default NavIconButton;
