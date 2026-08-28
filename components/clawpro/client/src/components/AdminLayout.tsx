import type { ReactNode } from "react";
import { useState } from "react";
import { Link, useLocation } from "wouter";
import {
  Bell,
  ChevronDown,
  ChevronRight,
  LogOut,
  MoreVertical,
} from "lucide-react";
import { toast } from "sonner";

import { ProductUpdatesDrawer, hasRecentProductUpdates } from "@/components/onboarding/ProductUpdatesDrawer";

import AdminModeToggle from "@/components/AdminModeToggle";
import AdminModeFloatingToggle from "@/components/AdminModeFloatingToggle";
import ArrearsModal from "@/components/ArrearsModal";
import AdminArrearsFloatCard from "@/components/AdminArrearsFloatCard";
import AdminNoticeBar, { getAdminNotices } from "@/components/AdminNoticeBar";
import {
  AdminSidebar,
  AdminSidebarBadge,
  AdminSidebarBrand,
  AdminSidebarContent,
  AdminSidebarFooter,
  AdminSidebarFooterAction,
  AdminSidebarGroupLabel,
  AdminSidebarHeader,
  AdminSidebarHeaderAction,
  AdminSidebarInset,
  AdminSidebarLogo,
  AdminSidebarMenu,
  AdminSidebarMenuButton,
  AdminSidebarMenuItem,
  AdminSidebarProvider,
  AdminSidebarUser,
  SidebarCollapseIcon,
  useAdminSidebar,
} from "@/components/ui/admin-sidebar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import {
  MetaText,
  MiniBodyText,
  PanelTitle,
} from "@/components/ui/Typography";
import goTenantArrowIcon from "@/assets/icons/go-tenant-arrow.svg";

import { AdminModeProvider, useAdminMode } from "@/contexts/AdminModeContext";
import {
  ADMIN_NAV_GROUPS,
  type AdminNavItem,
  type AdminNavSubGroup,
} from "@/config/adminNav";
import { cn } from "@/lib/utils";

const CURRENT_ADMIN = {
  name: "jingsujiang",
  role: "管理员",
};

const PRESET_BADGE_VARIANTS = new Set(["new", "coming-soon"]);

function GoTenantIcon({ className }: { className?: string }) {
  return (
    <svg
      width="15"
      height="14"
      viewBox="0 0 15 14"
      fill="none"
      aria-hidden="true"
      className={className}
    >
      <path
        d="M5.83333 8.5H3.83333C1.99239 8.5 0.5 9.9924 0.5 11.8333V12.5H5.8672M11.1946 12.8347L13.5014 10.5013L11.1946 8.16801M12.8334 10.5013H7.83203M9.16667 3.5C9.16667 5.15685 7.82353 6.5 6.16667 6.5C4.50981 6.5 3.16667 5.15685 3.16667 3.5C3.16667 1.84315 4.50981 0.5 6.16667 0.5C7.82353 0.5 9.16667 1.84315 9.16667 3.5Z"
        stroke="currentColor"
        strokeLinecap="square"
      />
    </svg>
  );
}

function isActiveRoute(location: string, href: string) {
  return location === href || location.startsWith(`${href}/`);
}

const ADMIN_ROUTE_ALIASES: Record<string, string[]> = {
  "/admin/agent-types": ["/admin/image-management"],
};

function isNavItemActive(location: string, item: Pick<AdminNavItem, "href">) {
  return (
    isActiveRoute(location, item.href) ||
    (ADMIN_ROUTE_ALIASES[item.href] ?? []).some(href =>
      isActiveRoute(location, href)
    )
  );
}

function renderNavItem(
  item: AdminNavItem,
  location: string,
  isSubItem = false,
  collapsed = false
) {
  const isActive = isNavItemActive(location, item);

  let badgeNode: ReactNode = null;
  if (item.badge && !collapsed) {
    if (PRESET_BADGE_VARIANTS.has(item.badge)) {
      badgeNode = (
        <AdminSidebarBadge variant={item.badge as "new" | "coming-soon"} />
      );
    } else {
      badgeNode = (
        <AdminSidebarBadge variant="custom">{item.badge}</AdminSidebarBadge>
      );
    }
  }

  if (collapsed) {
    const iconEl = item.iconSrc ? (
      <img
        src={item.iconSrc}
        alt=""
        className="size-4 shrink-0"
        aria-hidden="true"
      />
    ) : (
      <span className="size-4 shrink-0" aria-hidden="true" />
    );

    return (
      <AdminSidebarMenuItem key={item.href}>
        <Tooltip>
          <TooltipTrigger asChild>
            <Link
              href={item.href}
              data-slot="admin-sidebar-menu-button"
              data-active={isActive}
              aria-current={isActive ? "page" : undefined}
              className="flex h-[var(--admin-sidebar-item-height)] w-full items-center justify-center rounded-[var(--admin-sidebar-item-radius)] text-[var(--admin-sidebar-foreground)] outline-none transition-all duration-150 focus-visible:ring-2 focus-visible:ring-[var(--brand-blue)]"
            >
              {iconEl}
            </Link>
          </TooltipTrigger>
          <TooltipContent side="right" sideOffset={8}>
            {item.label}
          </TooltipContent>
        </Tooltip>
      </AdminSidebarMenuItem>
    );
  }

  return (
    <AdminSidebarMenuItem key={item.href}>
      <AdminSidebarMenuButton
        asChild
        isActive={isActive}
        className={isSubItem ? "admin-sidebar-subitem-button" : undefined}
      >
        <Link href={item.href}>
          {!isSubItem &&
            (item.iconSrc ? (
              <img
                src={item.iconSrc}
                alt=""
                className="size-4 shrink-0"
                aria-hidden="true"
              />
            ) : (
              <span className="size-4 shrink-0" aria-hidden="true" />
            ))}
          <span className="min-w-0 flex-1 truncate">{item.label}</span>
          {badgeNode}
        </Link>
      </AdminSidebarMenuButton>
    </AdminSidebarMenuItem>
  );
}

function SubGroupBlock({
  subGroup,
  location,
  collapsed,
}: {
  subGroup: AdminNavSubGroup;
  location: string;
  collapsed: boolean;
}) {
  const [open, setOpen] = useState(subGroup.defaultExpanded ?? true);

  if (collapsed) {
    const hasActiveChild = subGroup.items.some(item =>
      isNavItemActive(location, item)
    );

    return (
      <AdminSidebarMenuItem>
        <HoverCard openDelay={120} closeDelay={200}>
          <HoverCardTrigger asChild>
            <AdminSidebarMenuButton
              isActive={hasActiveChild}
              className={`justify-center px-0 cursor-pointer ${
                hasActiveChild ? "admin-sidebar-collapsed-subgroup-active" : ""
              }`}
            >
              {subGroup.iconSrc ? (
                <img
                  src={subGroup.iconSrc}
                  alt=""
                  className="size-4 shrink-0"
                  aria-hidden="true"
                />
              ) : (
                <span className="size-4 shrink-0" aria-hidden="true" />
              )}
            </AdminSidebarMenuButton>
          </HoverCardTrigger>
          <HoverCardContent
            side="right"
            sideOffset={12}
            align="start"
            className="w-auto min-w-[140px] p-1.5 rounded-[8px] border border-[#E5E5E5] shadow-lg"
          >
            <p className="px-2 py-1 text-[11px] font-medium text-[#A3A3A3] tracking-wide">
              {subGroup.label}
            </p>
            <ul className="flex flex-col gap-0.5">
              {subGroup.items.map(item => {
                const active = isNavItemActive(location, item);
                return (
                  <li key={item.href}>
                    <Link
                      href={item.href}
                      className={`flex items-center gap-2 px-2 py-1.5 rounded-[4px] text-[13px] transition-colors ${
                        active
                          ? "bg-[#EBF4FF] text-[#1447E6] font-medium"
                          : "text-[#334155] hover:bg-[#F5F5F5]"
                      }`}
                    >
                      <span>{item.label}</span>
                    </Link>
                  </li>
                );
              })}
            </ul>
          </HoverCardContent>
        </HoverCard>
      </AdminSidebarMenuItem>
    );
  }

  return (
    <>
      <AdminSidebarMenuItem>
        <AdminSidebarMenuButton
          isActive={false}
          onClick={() => setOpen(v => !v)}
          aria-expanded={open}
          aria-label={`${open ? "收起" : "展开"}${subGroup.label}`}
        >
          {subGroup.iconSrc ? (
            <img
              src={subGroup.iconSrc}
              alt=""
              className="size-4 shrink-0"
              aria-hidden="true"
            />
          ) : (
            <span className="size-4 shrink-0" aria-hidden="true" />
          )}
          <span className="min-w-0 flex-1 truncate">{subGroup.label}</span>
          <svg
            width="12"
            height="12"
            viewBox="0 0 12 12"
            fill="none"
            aria-hidden="true"
            className={`!size-3 shrink-0 text-[var(--text-muted)] transition-transform ${open ? "" : "rotate-180"}`}
          >
            <path d="M3 7.5L6 4.5L9 7.5" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </AdminSidebarMenuButton>
      </AdminSidebarMenuItem>
      {open &&
        subGroup.items.map(item => renderNavItem(item, location, true, false))}
    </>
  );
}

/** 仅 preview/demo 页使用的额外侧栏菜单项；真实管理页不传，故零影响。
 *  iconSrc 由调用方传入真实侧栏图标路径（/assets/admin-sidebar/*.svg），
 *  与 ADMIN_NAV_GROUPS 同源 <img> 渲染，AdminLayout 不内联任何业务图标。 */
type PreviewNavItem = { label: string; href: string; iconSrc?: string };

function AdminLayoutInner({
  children,
  previewNavItem,
  hideNoticeBar,
}: {
  children: ReactNode;
  previewNavItem?: PreviewNavItem;
  hideNoticeBar?: boolean;
}) {
  const [location] = useLocation();
  const { isUnified } = useAdminMode();
  const { collapsed, toggleSidebar } = useAdminSidebar();
  const hasAdminNoticeBar =
    !hideNoticeBar && getAdminNotices(isUnified).length > 0;
  const isSidebarMenuPage = ADMIN_NAV_GROUPS.some(
    group =>
      (group.items ?? []).some(item => isNavItemActive(location, item)) ||
      (group.trailingItems ?? []).some(item => isNavItemActive(location, item)) ||
      (group.subGroups ?? []).some(subGroup =>
        subGroup.items.some(item => isNavItemActive(location, item))
      )
  );
  const contentTopSpacingClass = isSidebarMenuPage
    ? hasAdminNoticeBar
      ? "pt-2"
      : "pt-[60px]"
    : "pt-8";

  const showFull = !collapsed;
  const [sidebarHovered, setSidebarHovered] = useState(false);
  const [productUpdatesOpen, setProductUpdatesOpen] = useState(false);
  // 铃铛红点：只要存在「近期更新」卡片就常驻展示（与产品动态抽屉数据同源）
  const hasRecentUpdates = hasRecentProductUpdates();

  return (
    <>
    <div className="flex h-screen overflow-hidden admin-theme">
      <AdminSidebar
        className="group/sidebar"
        onMouseEnter={() => setSidebarHovered(true)}
        onMouseLeave={() => setSidebarHovered(false)}
      >
        <AdminSidebarHeader>
          {!showFull ? (
            <div className="flex flex-col items-center gap-2">
              <Tooltip delayDuration={0}>
                <TooltipTrigger asChild>
                  <button
                    onClick={toggleSidebar}
                    className="relative flex items-center justify-center h-10 w-7"
                    aria-label="展开导航"
                  >
                    {sidebarHovered ? (
                      <SidebarCollapseIcon className="size-4 text-[var(--admin-sidebar-muted)]" />
                    ) : (
                      <AdminSidebarLogo className="shrink-0 w-7 h-auto" />
                    )}
                  </button>
                </TooltipTrigger>
                <TooltipContent side="right" sideOffset={8}>
                  展开导航
                </TooltipContent>
              </Tooltip>
              <Tooltip delayDuration={0}>
                <TooltipTrigger asChild>
                  <a
                    href="/my-openclaw"
                    className="flex items-center justify-center size-8 rounded-[4px] border border-[var(--admin-sidebar-action-border)] bg-[var(--admin-sidebar-action-bg)] text-[var(--admin-sidebar-foreground)]"
                  >
                    <GoTenantIcon className="size-4" />
                  </a>
                </TooltipTrigger>
                <TooltipContent side="right" sideOffset={8}>
                  前往用户端
                </TooltipContent>
              </Tooltip>
            </div>
          ) : (
            <>
              <div className="flex h-[72px] items-start justify-between px-4 pt-4">
                <AdminSidebarBrand asChild className="!gap-2">
                  <Link href="/" aria-label="返回首页">
                    <span
                      className="flex size-11 shrink-0 items-center justify-center overflow-hidden rounded-[4px] bg-white [box-shadow:var(--admin-sidebar-logo-shadow)]"
                      aria-hidden="true"
                    >
                      <AdminSidebarLogo className="size-full shrink-0" preserveAspectRatio="xMidYMid meet" />
                    </span>
                    <div className="flex h-[42px] min-w-0 flex-col justify-center">
                      <PanelTitle as="p" className="truncate font-medium leading-[22px] tracking-[0.08px]">
                        管控端
                      </PanelTitle>
                      <MetaText as="p" className="block truncate leading-5 tracking-[0.18px]">
                        ClawPro Admin
                      </MetaText>
                    </div>
                  </Link>
                </AdminSidebarBrand>
                <Tooltip delayDuration={0}>
                  <TooltipTrigger asChild>
                    <button
                      onClick={toggleSidebar}
                      className="mt-3 flex size-5 shrink-0 items-center justify-center rounded-[4px] text-[var(--admin-sidebar-foreground)] transition-colors hover:text-[var(--text-brand)] focus-visible:ring-2 focus-visible:ring-[var(--brand-blue)]"
                      aria-label="收起导航"
                    >
                      <SidebarCollapseIcon className="size-4" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent side="right" sideOffset={8}>
                    收起导航
                  </TooltipContent>
                </Tooltip>
              </div>

              <AdminSidebarHeaderAction
                asChild
                className="group mx-4 !h-8 !w-[208px] justify-center rounded-[4px] px-0 py-0 text-center !text-[var(--text-emphasis)]"
              >
                <Link href="/my-openclaw">
                  <span className="inline-flex items-center justify-center">
                    <MiniBodyText
                      as="span"
                      tone="emphasis"
                      className="leading-5 transition-[transform] duration-300 ease-[cubic-bezier(0.22,1,0.36,1)]"
                    >
                      前往用户端
                    </MiniBodyText>
                    <span className="ml-0 inline-flex w-0 overflow-hidden transition-[width,margin] duration-300 ease-[cubic-bezier(0.22,1,0.36,1)] group-hover:ml-1 group-hover:w-3.5">
                      <img
                        src={goTenantArrowIcon}
                        alt=""
                        aria-hidden="true"
                        className="size-3.5 shrink-0 translate-x-[-6px] opacity-0 transition-[transform,opacity] duration-300 ease-[cubic-bezier(0.22,1,0.36,1)] group-hover:translate-x-0 group-hover:opacity-100"
                      />
                    </span>
                  </span>
                </Link>
              </AdminSidebarHeaderAction>
            </>
          )}
        </AdminSidebarHeader>

        <AdminSidebarContent aria-label="管理后台导航">
          {!showFull ? (
            <div className="flex flex-col">
              {previewNavItem && (
                <AdminSidebarMenu className="mb-2">
                  <AdminSidebarMenuItem>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Link
                          href={previewNavItem.href}
                          data-slot="admin-sidebar-menu-button"
                          data-active={isActiveRoute(location, previewNavItem.href)}
                          aria-current={
                            isActiveRoute(location, previewNavItem.href)
                              ? "page"
                              : undefined
                          }
                          className="flex h-[var(--admin-sidebar-item-height)] w-full items-center justify-center rounded-[var(--admin-sidebar-item-radius)] text-[var(--admin-sidebar-foreground)] outline-none transition-all duration-150 focus-visible:ring-2 focus-visible:ring-[var(--brand-blue)]"
                        >
                          {previewNavItem.iconSrc ? (
                            <img
                              src={previewNavItem.iconSrc}
                              alt=""
                              className="size-4 shrink-0"
                              aria-hidden="true"
                            />
                          ) : (
                            <span className="size-4 shrink-0" aria-hidden="true" />
                          )}
                        </Link>
                      </TooltipTrigger>
                      <TooltipContent side="right" sideOffset={8}>
                        {previewNavItem.label}
                      </TooltipContent>
                    </Tooltip>
                  </AdminSidebarMenuItem>
                </AdminSidebarMenu>
              )}
              {ADMIN_NAV_GROUPS.map((group, idx) => (
                <div key={group.label}>
                  {idx > 0 && (
                    <div className="mx-2 my-2 border-t border-[#E5E5E5]" />
                  )}
                  <AdminSidebarMenu>
                    {group.items?.map(item =>
                      renderNavItem(item, location, false, true)
                    )}
                    {group.subGroups?.map(sub => (
                      <SubGroupBlock
                        key={sub.label}
                        subGroup={sub}
                        location={location}
                        collapsed={true}
                      />
                    ))}
                    {group.trailingItems?.map(item =>
                      renderNavItem(item, location, false, true)
                    )}
                  </AdminSidebarMenu>
                </div>
              ))}
            </div>
          ) : (
            <>
              {previewNavItem && (
                <div className="mb-5">
                  <AdminSidebarGroupLabel>测试</AdminSidebarGroupLabel>
                  <AdminSidebarMenu>
                    <AdminSidebarMenuItem>
                      <AdminSidebarMenuButton
                        asChild
                        isActive={isActiveRoute(location, previewNavItem.href)}
                      >
                        <Link href={previewNavItem.href}>
                          {previewNavItem.iconSrc ? (
                            <img
                              src={previewNavItem.iconSrc}
                              alt=""
                              className="size-4 shrink-0"
                              aria-hidden="true"
                            />
                          ) : (
                            <span className="size-4 shrink-0" aria-hidden="true" />
                          )}
                          <span className="min-w-0 flex-1 truncate">
                            {previewNavItem.label}
                          </span>
                        </Link>
                      </AdminSidebarMenuButton>
                    </AdminSidebarMenuItem>
                  </AdminSidebarMenu>
                </div>
              )}
              {ADMIN_NAV_GROUPS.map((group, idx) => (
                <div key={group.label}>
                  <AdminSidebarGroupLabel
                    className={idx > 0 || previewNavItem ? "mt-5" : undefined}
                  >
                    {group.label}
                  </AdminSidebarGroupLabel>
                  <AdminSidebarMenu>
                    {group.items?.map(item =>
                      renderNavItem(item, location, false, false)
                    )}
                    {group.subGroups?.map(sub => (
                      <SubGroupBlock
                        key={sub.label}
                        subGroup={sub}
                        location={location}
                        collapsed={false}
                      />
                    ))}
                    {group.trailingItems?.map(item =>
                      renderNavItem(item, location, false, false)
                    )}
                  </AdminSidebarMenu>
                </div>
              ))}
            </>
          )}
        </AdminSidebarContent>

        <AdminSidebarFooter>
          {!showFull ? (
            <HoverCard openDelay={120} closeDelay={200}>
              <HoverCardTrigger asChild>
                <div className="mx-auto flex size-8 shrink-0 cursor-pointer items-center justify-center rounded-full bg-[var(--admin-sidebar-avatar-bg)] font-mono text-[14.22px] font-normal leading-none text-[var(--admin-sidebar-avatar-foreground)]">
                  {CURRENT_ADMIN.name.charAt(0).toUpperCase()}
                </div>
              </HoverCardTrigger>
              <HoverCardContent
                side="right"
                sideOffset={12}
                align="end"
                className="w-[240px] p-2 rounded-[8px] border border-[#E5E5E5] shadow-lg"
              >
                <div className="px-2 py-1.5">
                  <p className="text-sm font-medium text-[#0A0A0A]">
                    {CURRENT_ADMIN.name}
                  </p>
                  <p className="text-xs text-[#737373]">{CURRENT_ADMIN.role}</p>
                </div>
                <div className="my-1.5 border-t border-[#E5E5E5]" />
                <div className="px-2 py-1.5">
                  <p className="text-xs text-[#737373] mb-1.5">成员管理模式</p>
                  <AdminModeToggle collapsed={false} />
                </div>
                <div className="my-1.5 border-t border-[#E5E5E5]" />
                <button
                  onClick={() => toast.info("已退出登录")}
                  className="flex items-center gap-2 w-full px-2 py-1.5 rounded-[4px] text-[13px] text-red-600 hover:bg-red-50 transition-colors"
                >
                  <LogOut className="size-3.5" />
                  退出登录
                </button>
              </HoverCardContent>
            </HoverCard>
          ) : (
            <>
              <AdminSidebarUser
                name={CURRENT_ADMIN.name}
                role={CURRENT_ADMIN.role}
              />
              <Tooltip delayDuration={0}>
                <TooltipTrigger asChild>
                  <AdminSidebarFooterAction
                    aria-label="产品动态"
                    className="relative"
                    onClick={() => {
                      setProductUpdatesOpen(true);
                    }}
                  >
                    <Bell />
                    {hasRecentUpdates && (
                      <span className="absolute right-1.5 top-1.5 size-1.5 rounded-full bg-[var(--text-danger)] ring-2 ring-[var(--cp-surface)]" />
                    )}
                  </AdminSidebarFooterAction>
                </TooltipTrigger>
                <TooltipContent side="top" sideOffset={8}>
                  产品动态
                </TooltipContent>
              </Tooltip>
              <DropdownMenu modal={false}>
                <DropdownMenuTrigger asChild>
                  <AdminSidebarFooterAction aria-label="更多管理操作">
                    <MoreVertical />
                  </AdminSidebarFooterAction>
                </DropdownMenuTrigger>
                {/* Figma 420_70231：131×120 窄款菜单，3 项纯文字（服务续费高亮 / 账户充值 / 退出登录警示色） */}
                <DropdownMenuContent
                  side="top"
                  align="end"
                  sideOffset={8}
                  className="w-[131px] min-w-0 p-2"
                >
                  <DropdownMenuItem
                    onClick={() => toast.info("服务续费")}
                    className="h-8 justify-start px-3 text-sm font-medium text-[#020617] bg-[#F3F3F4] focus:bg-[#F3F3F4]"
                  >
                    服务续费
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() => toast.info("账户充值")}
                    className="h-8 justify-start px-3 text-sm font-normal text-[#020617]"
                  >
                    账户充值
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() => toast.info("已退出登录")}
                    className="h-8 justify-start px-3 text-sm font-normal !text-[#C04100] focus:!text-[#C04100]"
                  >
                    退出登录
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          )}
        </AdminSidebarFooter>
      </AdminSidebar>

      <AdminSidebarInset>
        {!hideNoticeBar && <AdminNoticeBar />}
        {/* 告警条与滚动区之间的固定留白：滚动区内容消失的裁切线由此向下偏移，避免内容紧贴告警条 */}
        {hasAdminNoticeBar && <div aria-hidden="true" className={cn("shrink-0", contentTopSpacingClass)} />}
        <div data-slot="admin-content-scroll" className="flex-1 min-h-0 overflow-y-auto">
          <div
            className={cn(
              "min-w-[960px] max-w-[1600px] mx-auto px-10 pb-8 overflow-x-clip",
              hasAdminNoticeBar ? undefined : contentTopSpacingClass
            )}
          >
            <div
              data-admin-menu-page-shell={isSidebarMenuPage ? "true" : undefined}
            >
              {children}
            </div>
          </div>
        </div>
      </AdminSidebarInset>
    </div>
    <ProductUpdatesDrawer
      open={productUpdatesOpen}
      onClose={() => setProductUpdatesOpen(false)}
    />
    </>
  );
}

export default function AdminLayout({
  children,
  previewNavItem,
  hideNoticeBar,
}: {
  children: ReactNode;
  previewNavItem?: PreviewNavItem;
  hideNoticeBar?: boolean;
}) {
  return (
    <AdminModeProvider>
      <AdminSidebarProvider>
        <AdminLayoutInner
          previewNavItem={previewNavItem}
          hideNoticeBar={hideNoticeBar}
        >
          {children}
        </AdminLayoutInner>
        {/* 「成员管理模式」独立悬浮切换面板（Demo/开发用，仅管控端显示） */}
        <AdminModeFloatingToggle />
        {/* 账户欠费提醒弹窗（Figma 751_45149，仅管控端 seatArrears=true 时弹出） */}
        <ArrearsModal />
        {/* 席位正式欠费档：管控端左下角 220×169 常态浮层（Figma 906_45782），持续提醒直至充值 */}
        <AdminArrearsFloatCard />
      </AdminSidebarProvider>
    </AdminModeProvider>
  );
}
