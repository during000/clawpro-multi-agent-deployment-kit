/**
 * AdminSidebar Tooltip / HoverCard 集成层
 * ───────────────────────────────────────────────────────────────────────────
 *
 * 这个文件展示如何为 AdminSidebar 添加 Tooltip 和 HoverCard 支持。
 *
 * 用法：将这个文件中的组件替换默认导出组件，或按需集成到现有项目中。
 *
 * 依赖：
 *   - @radix-ui/react-tooltip（Tooltip）
 *   - @radix-ui/react-hover-card（HoverCard）
 *   或其他兼容的 Tooltip / HoverCard 库
 *
 * ─── 关键集成点 ───────────────────────────────────────────────────────────────
 *
 * 1️⃣ 展开态
 *    - 折叠按钮需要 Tooltip「收起导航」
 *
 * 2️⃣ 收起态
 *    - 菜单项需要 Tooltip <label>
 *    - Header "前往用户端" 按钮需要 Tooltip
 *    - 折叠按钮需要 Tooltip「展开导航」
 *    - Footer 用户需要 HoverCard（用户管理 + 退出登录）
 *
 * ───────────────────────────────────────────────────────────────────────────
 */

import * as React from "react";
import {
  AdminSidebar,
  AdminSidebarHeader,
  AdminSidebarBrand,
  AdminSidebarLogo,
  AdminSidebarHeaderAction,
  AdminSidebarContent,
  AdminSidebarGroup,
  AdminSidebarGroupLabel,
  AdminSidebarGroupTrigger,
  AdminSidebarGroupContent,
  AdminSidebarMenu,
  AdminSidebarMenuItem,
  AdminSidebarMenuButton,
  AdminSidebarBadge,
  AdminSidebarFooter,
  AdminSidebarUser,
  AdminSidebarFooterAction,
  AdminSidebarInset,
  AdminSidebarProvider,
  AdminSidebarTrigger,
  useAdminSidebar,
  SidebarCollapseIcon,
} from "./admin-sidebar";

/* ─── Tooltip Wrapper（假设使用 @radix-ui/react-tooltip） ─────────────── */

/**
 * 简单的 Tooltip 包装组件
 * 如果你用的是 @radix-ui/react-tooltip，可以这样集成：
 *
 * import * as Tooltip from "@radix-ui/react-tooltip";
 */
interface TooltipProps {
  content: React.ReactNode;
  children: React.ReactNode;
  side?: "top" | "right" | "bottom" | "left";
  sideOffset?: number;
  delayDuration?: number;
}

/**
 * 使用 @radix-ui/react-tooltip 的实现
 * 取消注释即可使用真实的 Tooltip
 */
/*
import * as RadixTooltip from "@radix-ui/react-tooltip";

const Tooltip: React.FC<TooltipProps> = ({
  content,
  children,
  side = "top",
  sideOffset = 4,
  delayDuration = 200,
}) => (
  <RadixTooltip.Provider>
    <RadixTooltip.Root delayDuration={delayDuration}>
      <RadixTooltip.Trigger asChild>{children}</RadixTooltip.Trigger>
      <RadixTooltip.Content
        side={side}
        sideOffset={sideOffset}
        className="bg-gray-900 text-white text-xs px-2 py-1 rounded z-50"
      >
        {content}
      </RadixTooltip.Content>
    </RadixTooltip.Root>
  </RadixTooltip.Provider>
);
*/

/**
 * 简单的降级实现（title attribute）
 * 如果没有 Tooltip 库，至少可以用 title 提供基础提示
 */
const Tooltip: React.FC<TooltipProps> = ({
  content,
  children,
  side,
  sideOffset,
  delayDuration,
}) => {
  const contentText = typeof content === "string" ? content : "查看更多";
  return (
    <div title={contentText} className="cursor-help">
      {children}
    </div>
  );
};

/* ─── HoverCard Wrapper（假设使用 @radix-ui/react-hover-card） ─────────── */

/**
 * 使用 @radix-ui/react-hover-card 的实现
 * 取消注释即可使用真实的 HoverCard
 */
/*
import * as RadixHoverCard from "@radix-ui/react-hover-card";

interface HoverCardProps {
  trigger: React.ReactNode;
  content: React.ReactNode;
  side?: "top" | "right" | "bottom" | "left";
  sideOffset?: number;
}

const HoverCard: React.FC<HoverCardProps> = ({
  trigger,
  content,
  side = "right",
  sideOffset = 8,
}) => (
  <RadixHoverCard.Root>
    <RadixHoverCard.Trigger asChild>{trigger}</RadixHoverCard.Trigger>
    <RadixHoverCard.Content side={side} sideOffset={sideOffset} className="w-80 p-4 bg-white border rounded-lg shadow-lg z-50">
      {content}
    </RadixHoverCard.Content>
  </RadixHoverCard.Root>
);
*/

/**
 * 简单的降级实现（无 HoverCard）
 */
interface HoverCardProps {
  trigger: React.ReactNode;
  content: React.ReactNode;
  side?: "top" | "right" | "bottom" | "left";
  sideOffset?: number;
}

const HoverCard: React.FC<HoverCardProps> = ({
  trigger,
  content,
}) => <>{trigger}</>;

/* ─── 增强的 Header 折叠按钮（加 Tooltip） ────────────────────────────── */

/**
 * 展开态：折叠按钮带 Tooltip「收起导航」
 * 收起态：显示 Tooltip「展开导航」
 */
export const AdminSidebarHeaderWithTooltip = React.forwardRef<
  HTMLDivElement,
  React.ComponentProps<"div">
>(({ ...props }, ref) => {
  const { collapsed, toggleSidebar } = useAdminSidebar();

  return (
    <AdminSidebarHeader ref={ref} {...props}>
      <AdminSidebarBrand asChild>
        <a href="/admin" className="flex items-center gap-2.5">
          <span className="cp-admin-sidebar-logo-box">
            <AdminSidebarLogo />
          </span>
          {!collapsed && (
            <span className="flex flex-col">
              <strong className="text-sm leading-[22px] text-[var(--cp-admin-sidebar-fg)]">
                管控端
              </strong>
              <small className="text-xs leading-5 text-[var(--cp-admin-sidebar-muted)]">
                ClawPro Admin
              </small>
            </span>
          )}
        </a>
      </AdminSidebarBrand>

      {!collapsed && (
        <>
          {/* 展开态：前往用户端按钮 */}
          <AdminSidebarHeaderAction asChild className="mx-4 mt-2 h-8 w-[208px] justify-between px-3">
            <a href="/">
              前往用户端
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <path
                  d="M1 7h12M8 3l4 4-4 4"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </a>
          </AdminSidebarHeaderAction>

          {/* 展开态：折叠按钮 + Tooltip「收起导航」 */}
          <Tooltip
            content="收起导航"
            side="right"
            sideOffset={8}
            delayDuration={0}
          >
            <button
              onClick={toggleSidebar}
              className="mt-3 size-5 text-[var(--cp-admin-sidebar-muted)] hover:text-[var(--cp-text-brand)] transition-colors outline-none focus-visible:ring-2 focus-visible:ring-[var(--cp-brand-blue)]"
              aria-label="收起导航"
            >
              <SidebarCollapseIcon />
            </button>
          </Tooltip>
        </>
      )}

      {collapsed && (
        <>
          {/* 收起态：Logo（hover 换展开图标） */}
          <Tooltip content="展开导航" side="right" sideOffset={8}>
            <button
              onClick={toggleSidebar}
              className="size-8 rounded-[4px] flex items-center justify-center text-[var(--cp-admin-sidebar-muted)] hover:text-[var(--cp-text-brand)] transition-colors outline-none focus-visible:ring-2 focus-visible:ring-[var(--cp-brand-blue)]"
              aria-label="展开导航"
            >
              <AdminSidebarLogo />
            </button>
          </Tooltip>

          {/* 收起态：前往用户端图标按钮 */}
          <Tooltip content="前往用户端" side="right" sideOffset={8}>
            <a
              href="/"
              className="size-8 rounded-[4px] flex items-center justify-center text-[var(--cp-admin-sidebar-muted)] hover:text-[var(--cp-text-brand)] transition-colors outline-none focus-visible:ring-2 focus-visible:ring-[var(--cp-brand-blue)]"
              aria-label="前往用户端"
            >
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <path
                  d="M1 8h14M9 2l6 6-6 6"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </a>
          </Tooltip>
        </>
      )}
    </AdminSidebarHeader>
  );
});
AdminSidebarHeaderWithTooltip.displayName = "AdminSidebarHeaderWithTooltip";

/* ─── 增强的菜单项（收起态加 Tooltip） ──────────────────────────────── */

/**
 * 收起态时显示 Tooltip
 */
interface AdminSidebarMenuButtonWithTooltipProps
  extends React.ComponentProps<typeof AdminSidebarMenuButton> {
  label?: string;
}

export const AdminSidebarMenuButtonWithTooltip = React.forwardRef<
  HTMLButtonElement,
  AdminSidebarMenuButtonWithTooltipProps
>(({ label, children, ...props }, ref) => {
  const { collapsed } = useAdminSidebar();

  const button = (
    <AdminSidebarMenuButton ref={ref} {...props}>
      {children}
    </AdminSidebarMenuButton>
  );

  if (collapsed && label) {
    return <Tooltip content={label} side="right" sideOffset={8}>{button}</Tooltip>;
  }

  return button;
});
AdminSidebarMenuButtonWithTooltip.displayName = "AdminSidebarMenuButtonWithTooltip";

/* ─── 增强的 Footer（用户 HoverCard） ──────────────────────────────────── */

/**
 * 收起态显示 HoverCard（用户管理 + 退出登录）
 */
interface AdminSidebarUserWithHoverCardProps
  extends React.ComponentProps<typeof AdminSidebarUser> {
  onLogout?: () => void;
  onManageUsers?: () => void;
}

export const AdminSidebarUserWithHoverCard = React.forwardRef<
  HTMLDivElement,
  AdminSidebarUserWithHoverCardProps
>(({ name, role, fallback, onLogout, onManageUsers, ...props }, ref) => {
  const { collapsed } = useAdminSidebar();

  const userContent = (
    <AdminSidebarUser
      ref={ref}
      name={name}
      role={role}
      fallback={fallback}
      {...props}
    />
  );

  if (collapsed) {
    return (
      <HoverCard
        trigger={
          <button className="size-8 rounded-full bg-[var(--cp-admin-sidebar-avatar-bg)] font-mono text-[14.22px] font-normal leading-none text-[var(--cp-admin-sidebar-avatar-fg)] outline-none focus-visible:ring-2 focus-visible:ring-[var(--cp-brand-blue)] hover:opacity-80">
            {fallback ?? name.charAt(0).toUpperCase()}
          </button>
        }
        content={
          <div className="space-y-2">
            <div className="px-2 py-2 border-b">
              <p className="text-sm font-medium">{name}</p>
              <p className="text-xs text-gray-500">{role}</p>
            </div>
            <button
              onClick={onManageUsers}
              className="w-full text-left px-2 py-1 text-sm hover:bg-gray-100 rounded"
            >
              用户管理
            </button>
            <button
              onClick={onLogout}
              className="w-full text-left px-2 py-1 text-sm hover:bg-gray-100 rounded text-red-600"
            >
              退出登录
            </button>
          </div>
        }
        side="right"
        sideOffset={8}
      />
    );
  }

  return userContent;
});
AdminSidebarUserWithHoverCard.displayName = "AdminSidebarUserWithHoverCard";

/* ─── 完整使用示例 ─────────────────────────────────────────────────────── */

/**
 * 示例：如何在你的项目中使用增强版 AdminSidebar
 *
 * import {
 *   AdminSidebarProvider,
 *   AdminSidebar,
 *   AdminSidebarHeaderWithTooltip,
 *   AdminSidebarContent,
 *   AdminSidebarGroup,
 *   AdminSidebarGroupLabel,
 *   AdminSidebarMenu,
 *   AdminSidebarMenuItem,
 *   AdminSidebarMenuButtonWithTooltip,
 *   AdminSidebarBadge,
 *   AdminSidebarFooter,
 *   AdminSidebarUserWithHoverCard,
 *   AdminSidebarFooterAction,
 *   AdminSidebarInset,
 * } from "./admin-sidebar-with-tooltip";
 *
 * export function App() {
 *   const handleLogout = () => {
 *     console.log("用户退出登录");
 *   };
 *
 *   return (
 *     <AdminSidebarProvider>
 *       <AdminSidebar>
 *         <AdminSidebarHeaderWithTooltip />
 *         <AdminSidebarContent>
 *           <AdminSidebarGroup>
 *             <AdminSidebarGroupLabel>基础信息</AdminSidebarGroupLabel>
 *             <AdminSidebarMenu>
 *               <AdminSidebarMenuItem>
 *                 <AdminSidebarMenuButtonWithTooltip
 *                   label="基础信息配置"
 *                   isActive
 *                   asChild
 *                 >
 *                   <a href="/admin/basic">
 *                     <img src={icon} alt="" aria-hidden />
 *                     <span>基础信息配置</span>
 *                   </a>
 *                 </AdminSidebarMenuButtonWithTooltip>
 *               </AdminSidebarMenuItem>
 *             </AdminSidebarMenu>
 *           </AdminSidebarGroup>
 *         </AdminSidebarContent>
 *         <AdminSidebarFooter>
 *           <AdminSidebarUserWithHoverCard
 *             name="jingsujiang"
 *             role="管理员"
 *             onLogout={handleLogout}
 *           />
 *           <AdminSidebarFooterAction aria-label="更多">
 *             <MoreIcon />
 *           </AdminSidebarFooterAction>
 *         </AdminSidebarFooter>
 *       </AdminSidebar>
 *       <AdminSidebarInset>
 *         {/* 你的页面内容 */}
 *       </AdminSidebarInset>
 *     </AdminSidebarProvider>
 *   );
 * }
 */

export {
  AdminSidebar,
  AdminSidebarBadge,
  AdminSidebarBrand,
  AdminSidebarContent,
  AdminSidebarFooter,
  AdminSidebarFooterAction,
  AdminSidebarGroup,
  AdminSidebarGroupContent,
  AdminSidebarGroupLabel,
  AdminSidebarGroupTrigger,
  AdminSidebarHeader,
  AdminSidebarHeaderAction,
  AdminSidebarInset,
  AdminSidebarLogo,
  AdminSidebarMenu,
  AdminSidebarMenuButton,
  AdminSidebarMenuItem,
  AdminSidebarProvider,
  AdminSidebarTrigger,
  AdminSidebarUser,
  SidebarCollapseIcon,
  useAdminSidebar,
  Tooltip,
  HoverCard,
};
