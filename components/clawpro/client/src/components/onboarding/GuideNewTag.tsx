/**
 * GuideNewTag - 侧边栏导航标签（New / 即将开放 / 自定义）
 * 对应场景：元素层 2.5 New Tag —— 最轻量的版本更新感知手段
 *
 * 视觉对齐管控端左侧导航 New 角标：胶囊形 h-18px，色值 color-mix(#1447E6 10%) 底 + 品牌蓝字。
 *
 * 变体 variant：
 * - new          淡蓝底 + 品牌蓝文字「New」
 * - coming-soon  淡橙底 + 橙色文字「即将开放」
 * - custom       灰底灰字 + 自定义文案（children）
 *
 * 零打断：用户浏览导航时自然发现。可配合 TTL 过期机制由业务侧控制显隐。
 */
import type { ReactNode } from "react";

export type NewTagVariant = "new" | "coming-soon" | "custom";

interface GuideNewTagProps {
  /** 标签变体 */
  variant?: NewTagVariant;
  /** custom 变体下的自定义文案 */
  children?: ReactNode;
  /** 仅用于布局微调（如 margin），禁止覆盖颜色 / 字号 */
  className?: string;
}

const VARIANT_STYLE: Record<NewTagVariant, { bg: string; color: string }> = {
  "new": { bg: "color-mix(in srgb, #1447E6 10%, #FFFFFF)", color: "#1447E6" },
  "coming-soon": { bg: "color-mix(in srgb, #F97316 12%, #FFFFFF)", color: "#F97316" },
  "custom": { bg: "#F5F5F5", color: "#737373" },
};

export function GuideNewTag({ variant = "new", children, className = "" }: GuideNewTagProps) {
  const s = VARIANT_STYLE[variant];
  const label =
    variant === "new" ? "New" : variant === "coming-soon" ? "即将开放" : children;

  return (
    <span
      className={`inline-flex h-[18px] shrink-0 items-center justify-center rounded-full px-1.5 text-[11px] font-normal leading-none ${className}`}
      style={{ background: s.bg, color: s.color, letterSpacing: "0.015em" }}
    >
      {label}
    </span>
  );
}
