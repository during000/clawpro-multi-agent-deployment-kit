/**
 * Portable Surface / Card — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 SurfaceCard / SurfaceInner / SurfaceConfig / TenantCard
 * 时的可移植兜底实现。
 *  - 不依赖 shadcn / Radix / cva / Tailwind。
 *  - 颜色 / 圆角 / 阴影 / 内边距全部由 portable/css/card.css 提供。
 *  - 视觉规范（spec/component-specs/card-surface.md §3）：
 *      SurfaceCard      管控端默认卡片：白底 + 蓝灰描边 + 4px 圆角 + 微弱阴影
 *      SurfaceInner     卡片内嵌的二级容器：白底 + 蓝灰描边 + 4px 圆角，无阴影
 *      SurfaceConfig    高亮配置卡：白底 + 0.5px 蓝色高亮描边 + 双层柔和阴影
 *      TenantCard       用户端业务卡片（白底 + 12px 圆角 + 弱阴影） ⚠ 仅 Tenant 场景
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/card.css";
 *
 * 用法：
 *   <PortableSurfaceCard>...</PortableSurfaceCard>
 *   <PortableSurfaceCard padding="lg">...</PortableSurfaceCard>
 *   <PortableSurfaceInner>...</PortableSurfaceInner>
 *   <PortableSurfaceConfig>...</PortableSurfaceConfig>
 *   <PortableTenantCard>...</PortableTenantCard>
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

type Padding = "none" | "sm" | "md" | "lg";

const PADDING_MAP: Record<Padding, string> = {
  none: "cp-surface--pad-none",
  sm: "cp-surface--pad-sm",
  md: "cp-surface--pad-md",
  lg: "cp-surface--pad-lg",
};

export interface PortableSurfaceCardProps
  extends React.HTMLAttributes<HTMLElement> {
  padding?: Padding;
}

export const PortableSurfaceCard = React.forwardRef<
  HTMLElement,
  PortableSurfaceCardProps
>(({ padding = "md", className = "", children, ...props }, ref) => {
  const cls = ["cp-surface-card", PADDING_MAP[padding], className]
    .filter(Boolean)
    .join(" ");
  return (
    <section ref={ref as any} className={cls} {...props}>
      {children}
    </section>
  );
});
PortableSurfaceCard.displayName = "PortableSurfaceCard";

/* ─── SurfaceInner：卡片内嵌容器（无阴影） ─── */

export interface PortableSurfaceInnerProps
  extends React.HTMLAttributes<HTMLElement> {
  padding?: Padding;
}

export const PortableSurfaceInner = React.forwardRef<
  HTMLElement,
  PortableSurfaceInnerProps
>(({ padding = "md", className = "", children, ...props }, ref) => {
  const cls = ["cp-surface-inner", PADDING_MAP[padding], className]
    .filter(Boolean)
    .join(" ");
  return (
    <div ref={ref as any} className={cls} {...props}>
      {children}
    </div>
  );
});
PortableSurfaceInner.displayName = "PortableSurfaceInner";

/* ─── SurfaceConfig：高亮配置卡（双层阴影 + 0.5px 高亮描边） ─── */

export interface PortableSurfaceConfigProps
  extends React.HTMLAttributes<HTMLElement> {
  padding?: Padding;
}

export const PortableSurfaceConfig = React.forwardRef<
  HTMLElement,
  PortableSurfaceConfigProps
>(({ padding = "md", className = "", children, ...props }, ref) => {
  const cls = ["cp-surface-config", PADDING_MAP[padding], className]
    .filter(Boolean)
    .join(" ");
  return (
    <section ref={ref as any} className={cls} {...props}>
      {children}
    </section>
  );
});
PortableSurfaceConfig.displayName = "PortableSurfaceConfig";

/* ─── TenantCard：用户端业务卡片（12px 圆角，仅 Tenant 场景） ─── */

export interface PortableTenantCardProps
  extends React.HTMLAttributes<HTMLElement> {
  padding?: Padding;
}

export const PortableTenantCard = React.forwardRef<
  HTMLElement,
  PortableTenantCardProps
>(({ padding = "md", className = "", children, ...props }, ref) => {
  const cls = ["cp-tenant-card", PADDING_MAP[padding], className]
    .filter(Boolean)
    .join(" ");
  return (
    <section ref={ref as any} data-note="allow-radius" className={cls} {...props}>
      {children}
    </section>
  );
});
PortableTenantCard.displayName = "PortableTenantCard";
