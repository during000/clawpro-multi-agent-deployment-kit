/**
 * Portable Segment — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有 Segment（局部分段切换）时的可移植兜底实现。
 *  - 不依赖 @radix-ui / shadcn。纯 React + CSS（segment.css）。
 *  - 视觉规范（spec/component-specs/segment.md §3）：
 *      Admin：容器 6px + Item 4px + 半透明灰底 + 活跃项白底 + 无 outline
 *      Tenant：容器 80px 胶囊 + Item 40px 胶囊 + 半透明灰底 + 活跃项白底 + 1px outline
 *  - **重要**：Admin 和 Tenant 绝不共用一套皮肤，端别必须在路由/context层先确定。
 *  - 受控组件，value 与 onValueChange 自管。
 *
 * 用法：
 *   // Admin 方角
 *   <PortableAdminSegment value={tab} onValueChange={setTab}>
 *     <PortableAdminSegmentItem value="overview">概览</PortableAdminSegmentItem>
 *     <PortableAdminSegmentItem value="detail">详情</PortableAdminSegmentItem>
 *   </PortableAdminSegment>
 *
 *   // Tenant 胶囊
 *   <PortableTenantSegment value={filter} onValueChange={setFilter}>
 *     <PortableTenantSegmentItem value="all">全部</PortableTenantSegmentItem>
 *     <PortableTenantSegmentItem value="group">分组</PortableTenantSegmentItem>
 *   </PortableTenantSegment>
 *
 * ─────────────────────────────────────────────────────────────────────────── */

import * as React from "react";

/* ─────────────── Segment Context ─────────────── */

interface SegmentContextValue {
  value: string;
  onValueChange: (value: string) => void;
}

const SegmentContext = React.createContext<SegmentContextValue | undefined>(
  undefined
);

function useSegment() {
  const ctx = React.useContext(SegmentContext);
  if (!ctx) throw new Error("Segment component used outside PortableSegment");
  return ctx;
}

/* ═════════════════════════════════════════════════════════════════════════
 * Admin Segment（方角）
 * ═════════════════════════════════════════════════════════════════════════ */

export interface PortableAdminSegmentProps {
  value: string;
  onValueChange: (value: string) => void;
  children: React.ReactNode;
  className?: string;
}

/**
 * Admin Segment 容器 - 方角 / 6px 圆角 / 半透明灰底
 * 角色（role）：tablist
 */
export function PortableAdminSegment({
  value,
  onValueChange,
  children,
  className = "",
}: PortableAdminSegmentProps) {
  const merged = [
    "cp-segment cp-segment--admin",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <SegmentContext.Provider value={{ value, onValueChange }}>
      <div role="tablist" className={merged}>
        {children}
      </div>
    </SegmentContext.Provider>
  );
}

export interface PortableAdminSegmentItemProps {
  value: string;
  children: React.ReactNode;
  disabled?: boolean;
  className?: string;
  onClick?: () => void;
}

/**
 * Admin Segment Item - 单个分段项
 * 视觉规范：
 *   - Active：白底 + #020617 深灰字 + Semibold + 0px 1px 2px 阴影
 *   - Default：透明底 + #7B818F 灰字 + Normal
 *   - Hover：字色加深到 #4B5563
 *   - Focus-visible：ring-[3px] ring-[#355EF1]/20
 */
export const PortableAdminSegmentItem = React.forwardRef<
  HTMLButtonElement,
  PortableAdminSegmentItemProps
>(({ value, children, disabled, className = "", onClick }, ref) => {
  const { value: activeValue, onValueChange } = useSegment();
  const isActive = value === activeValue;

  const merged = [
    "cp-segment__item",
    isActive && "cp-segment__item--active",
    disabled && "opacity-50",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button
      ref={ref}
      type="button"
      role="tab"
      aria-selected={isActive}
      disabled={disabled}
      onClick={() => {
        if (!disabled) {
          onValueChange(value);
          onClick?.();
        }
      }}
      className={merged}
    >
      {children}
    </button>
  );
});
PortableAdminSegmentItem.displayName = "PortableAdminSegmentItem";

/* ═════════════════════════════════════════════════════════════════════════
 * Tenant Segment（胶囊）
 * ═════════════════════════════════════════════════════════════════════════ */

export interface PortableTenantSegmentProps {
  value: string;
  onValueChange: (value: string) => void;
  children: React.ReactNode;
  className?: string;
}

/**
 * Tenant Segment 容器 - 胶囊 / 80px 圆角 / 半透明灰底
 * 角色（role）：tablist
 */
export function PortableTenantSegment({
  value,
  onValueChange,
  children,
  className = "",
}: PortableTenantSegmentProps) {
  const merged = [
    "cp-segment cp-segment--tenant",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <SegmentContext.Provider value={{ value, onValueChange }}>
      <div role="tablist" className={merged}>
        {children}
      </div>
    </SegmentContext.Provider>
  );
}

export interface PortableTenantSegmentItemProps {
  value: string;
  children: React.ReactNode;
  disabled?: boolean;
  className?: string;
  onClick?: () => void;
}

/**
 * Tenant Segment Item - 单个分段项（胶囊）
 * 视觉规范：
 *   - Active：白底 + #020617 深灰字 + Medium + 1px #CDD4DC outline + 0px 1px 4px 阴影
 *   - Default：透明底 + #334155 灰字 + Normal
 *   - Hover：字色加深到 #020617
 *   - Focus-visible：ring-[3px] ring-[#355EF1]/20
 */
export const PortableTenantSegmentItem = React.forwardRef<
  HTMLButtonElement,
  PortableTenantSegmentItemProps
>(({ value, children, disabled, className = "", onClick }, ref) => {
  const { value: activeValue, onValueChange } = useSegment();
  const isActive = value === activeValue;

  const merged = [
    "cp-segment__item",
    isActive && "cp-segment__item--active",
    disabled && "opacity-50",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button
      ref={ref}
      type="button"
      role="tab"
      aria-selected={isActive}
      disabled={disabled}
      onClick={() => {
        if (!disabled) {
          onValueChange(value);
          onClick?.();
        }
      }}
      className={merged}
    >
      {children}
    </button>
  );
});
PortableTenantSegmentItem.displayName = "PortableTenantSegmentItem";

/* ═════════════════════════════════════════════════════════════════════════
 * 便利导出：SegmentGroup / SegmentOption（非受控）
 * 用于独立状态管理的场景，不与 SegmentContent 联动
 * ═════════════════════════════════════════════════════════════════════════ */

export interface PortableAdminSegmentGroupProps {
  children: React.ReactNode;
  className?: string;
}

/**
 * Admin SegmentGroup - 非受控组件包装
 * 自行管理 active 状态
 */
export function PortableAdminSegmentGroup({
  children,
  className = "",
}: PortableAdminSegmentGroupProps) {
  const merged = [
    "cp-segment cp-segment--admin",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div role="tablist" className={merged}>
      {children}
    </div>
  );
}

export interface PortableAdminSegmentOptionProps {
  active: boolean;
  children: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  className?: string;
}

/**
 * Admin SegmentOption - 非受控的单个项
 */
export const PortableAdminSegmentOption = React.forwardRef<
  HTMLButtonElement,
  PortableAdminSegmentOptionProps
>(({ active, children, onClick, disabled, className = "" }, ref) => {
  const merged = [
    "cp-segment__item",
    active && "cp-segment__item--active",
    disabled && "opacity-50",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button
      ref={ref}
      type="button"
      role="tab"
      aria-selected={active}
      disabled={disabled}
      onClick={onClick}
      className={merged}
    >
      {children}
    </button>
  );
});
PortableAdminSegmentOption.displayName = "PortableAdminSegmentOption";

export interface PortableTenantSegmentGroupProps {
  children: React.ReactNode;
  className?: string;
}

/**
 * Tenant SegmentGroup - 非受控组件包装
 */
export function PortableTenantSegmentGroup({
  children,
  className = "",
}: PortableTenantSegmentGroupProps) {
  const merged = [
    "cp-segment cp-segment--tenant",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div role="tablist" className={merged}>
      {children}
    </div>
  );
}

export interface PortableTenantSegmentOptionProps {
  active: boolean;
  children: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  className?: string;
}

/**
 * Tenant SegmentOption - 非受控的单个项
 */
export const PortableTenantSegmentOption = React.forwardRef<
  HTMLButtonElement,
  PortableTenantSegmentOptionProps
>(({ active, children, onClick, disabled, className = "" }, ref) => {
  const merged = [
    "cp-segment__item",
    active && "cp-segment__item--active",
    disabled && "opacity-50",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button
      ref={ref}
      type="button"
      role="tab"
      aria-selected={active}
      disabled={disabled}
      onClick={onClick}
      className={merged}
    >
      {children}
    </button>
  );
});
PortableTenantSegmentOption.displayName = "PortableTenantSegmentOption";
