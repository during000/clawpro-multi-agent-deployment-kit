/**
 * Portable Tabs / LineTabs — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有 LineTabs（下划线一级 Tab）时的可移植兜底实现。
 *  - 不依赖 @radix-ui / shadcn / Tailwind；样式由 portable/css/tabs.css 提供。
 *  - 视觉规范（spec/component-specs/tabs.md §3）：
 *      容器：flex + gap-1 + border-b border-[#dbe6ff]（浅蓝灰底边）
 *      项 padding：px-4 py-3
 *      字号：14px / Medium（active / default 同字号同字重）
 *      Active：border-b-2 border-[#0A0A0A] -mb-px（黑色 2px 下划线压住底边）
 *      Active 文字：var(--cp-text-emphasis)（#020617）
 *      Default 文字：var(--cp-text-secondary)（#334155 / muted）
 *      Hover：仅文字加深为 emphasis，不出下划线
 *      ⚠ 不要把 active 下划线换成品牌蓝（蓝色是按钮主操作色）
 *
 *  - 受控组件，value 与 onValueChange 自管。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/tabs.css";
 *
 * 用法：
 *   <PortableTabs value={tab} onValueChange={setTab}>
 *     <PortableTabsList>
 *       <PortableTabsTrigger value="basic">基础</PortableTabsTrigger>
 *       <PortableTabsTrigger value="advanced">高级</PortableTabsTrigger>
 *       <PortableTabsTrigger value="permission" comingSoon>权限</PortableTabsTrigger>
 *     </PortableTabsList>
 *     <PortableTabsContent value="basic">…</PortableTabsContent>
 *     <PortableTabsContent value="advanced">…</PortableTabsContent>
 *   </PortableTabs>
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

interface TabsContextValue {
  value: string;
  onValueChange: (value: string) => void;
}

const TabsContext = React.createContext<TabsContextValue | undefined>(undefined);

function useTabs() {
  const ctx = React.useContext(TabsContext);
  if (!ctx) throw new Error("Tabs component used outside PortableTabs");
  return ctx;
}

/* ─────────────── PortableTabs（容器） ─────────────── */

export interface PortableTabsProps {
  value: string;
  onValueChange: (value: string) => void;
  children: React.ReactNode;
  className?: string;
}

export function PortableTabs({
  value,
  onValueChange,
  children,
  className = "",
}: PortableTabsProps) {
  return (
    <TabsContext.Provider value={{ value, onValueChange }}>
      <div className={className}>{children}</div>
    </TabsContext.Provider>
  );
}

/* ─────────────── PortableTabsList（Tab 头） ─────────────── */

export interface PortableTabsListProps {
  children: React.ReactNode;
  className?: string;
}

export function PortableTabsList({
  children,
  className = "",
}: PortableTabsListProps) {
  const merged = ["cp-tabs__list", className].filter(Boolean).join(" ");
  return <div className={merged}>{children}</div>;
}

/* ─────────────── PortableTabsTrigger（单个 Tab 按钮） ─────────────── */

export interface PortableTabsTriggerProps {
  value: string;
  children: React.ReactNode;
  disabled?: boolean;
  /** 是否显示"即将开放"Badge（右侧 outline 胶囊） */
  comingSoon?: boolean;
  className?: string;
}

export const PortableTabsTrigger = React.forwardRef<
  HTMLButtonElement,
  PortableTabsTriggerProps
>(({ value, children, disabled, comingSoon, className = "" }, ref) => {
  const { value: activeValue, onValueChange } = useTabs();
  const isActive = value === activeValue;

  const merged = [
    "cp-tabs__trigger",
    isActive && "cp-tabs__trigger--active",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button
      ref={ref}
      type="button"
      onClick={() => !disabled && onValueChange(value)}
      disabled={disabled}
      className={merged}
    >
      {children}
      {comingSoon && (
        <span className="cp-tabs__badge" aria-label="即将开放">
          即将开放
        </span>
      )}
    </button>
  );
});
PortableTabsTrigger.displayName = "PortableTabsTrigger";

/* ─────────────── PortableTabsContent（Tab 内容区） ─────────────── */

export interface PortableTabsContentProps {
  value: string;
  children: React.ReactNode;
  className?: string;
  forceMount?: boolean;
}

export function PortableTabsContent({
  value,
  children,
  className = "",
  forceMount = false,
}: PortableTabsContentProps) {
  const { value: activeValue } = useTabs();
  const isActive = value === activeValue;

  if (!isActive && !forceMount) return null;

  const merged = [
    "cp-tabs__content",
    !isActive && "cp-tabs__content--hidden",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return <div className={merged}>{children}</div>;
}

/* ─────────────── 别名：PortableLineTabs（语义更准确） ─────────────── */

export const PortableLineTabs = PortableTabs;
export const PortableLineTabsList = PortableTabsList;
export const PortableLineTabsTrigger = PortableTabsTrigger;
export const PortableLineTabsContent = PortableTabsContent;
