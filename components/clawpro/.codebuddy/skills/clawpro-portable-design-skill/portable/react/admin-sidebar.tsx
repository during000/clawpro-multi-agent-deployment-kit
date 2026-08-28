/**
 * Portable AdminSidebar — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有 client/src/components/ui/admin-sidebar.tsx 时的可移植兜底实现。
 *  - 与 demo 仓 admin-sidebar.tsx 1:1 对齐（21 个子组件全量补齐）。
 *  - 不依赖 shadcn / cva / class-variance-authority / @radix-ui/react-slot / @/lib/utils。
 *    asChild 用内置轻量 Slot（cloneElement）实现，cn 为本地实现。
 *  - 颜色 / 尺寸 / 圆角全部走 --cp-admin-sidebar-* token（见 portable/css/admin-sidebar.css）。
 *
 * ⚠️ 必须同时复制 portable/css/admin-sidebar.css 并在入口引入，否则只有结构没有：
 *    hover / active 背景、SubGroup 引导线、Badge 配色、选中态图标变黑、scrollbar-on-hover。
 *
 * 关键规格（QA 口径）：
 *  - 展开 240 / 收起 64；Header 104 / Footer 72。
 *  - 视口 ≤1200px 自动收起；首次进入优先读 localStorage["admin_sidebar_collapsed"]。
 *  - Menu item 高 34 / 圆角 4 / 字号 14 / 图标 16。
 *  - Active 弱蓝渐变 + 图标转纯黑；Hover 弱化背景；Focus ring 蓝。
 *  - 收起态不塞文字 / Badge，文本走 Tooltip（Tooltip 请用宿主仓自有组件包裹）。
 *
 * ─── 最小用法 ───────────────────────────────────────────────────────────────
 * import "../css/admin-sidebar.css";
 * import {
 *   AdminSidebarProvider, AdminSidebar, AdminSidebarHeader, AdminSidebarLogo,
 *   AdminSidebarBrand, AdminSidebarHeaderAction, AdminSidebarContent,
 *   AdminSidebarGroup, AdminSidebarGroupLabel, AdminSidebarMenu, AdminSidebarMenuItem,
 *   AdminSidebarMenuButton, AdminSidebarBadge, AdminSidebarFooter, AdminSidebarUser,
 *   AdminSidebarFooterAction, AdminSidebarInset,
 * } from "./admin-sidebar";
 *
 * <AdminSidebarProvider>
 *   <AdminSidebar>
 *     <AdminSidebarHeader>
 *       <AdminSidebarBrand asChild>
 *         <a href="/admin">
 *           <span className="cp-admin-sidebar-logo-box"><AdminSidebarLogo /></span>
 *           <span className="flex flex-col">
 *             <strong className="text-base font-medium leading-[22px] tracking-[0.08px] text-[var(--cp-admin-sidebar-fg)]">管控端</strong>
 *             <small className="text-xs font-normal leading-5 tracking-[0.18px] text-[var(--cp-admin-sidebar-muted)]">ClawPro Admin</small>
 *           </span>
 *         </a>
 *       </AdminSidebarBrand>
 *       <AdminSidebarHeaderAction asChild className="mx-4 mt-2 h-8 w-[208px] justify-between px-3">
 *         <a href="/">前往用户端<ArrowRightIcon /></a>
 *       </AdminSidebarHeaderAction>
 *     </AdminSidebarHeader>
 *     <AdminSidebarContent>
 *       <AdminSidebarGroup>
 *         <AdminSidebarGroupLabel>基础信息</AdminSidebarGroupLabel>
 *         <AdminSidebarMenu>
 *           <AdminSidebarMenuItem>
 *             <AdminSidebarMenuButton asChild isActive>
 *               <a href="/admin/basic"><img src={icon} alt="" aria-hidden /><span>基础信息配置</span></a>
 *             </AdminSidebarMenuButton>
 *           </AdminSidebarMenuItem>
 *           <AdminSidebarMenuItem>
 *             <AdminSidebarMenuButton asChild>
 *               <a href="/admin/skill"><img src={icon} alt="" aria-hidden /><span>技能配置</span>
 *                 <AdminSidebarBadge variant="new" /></a>
 *             </AdminSidebarMenuButton>
 *           </AdminSidebarMenuItem>
 *         </AdminSidebarMenu>
 *       </AdminSidebarGroup>
 *     </AdminSidebarContent>
 *     <AdminSidebarFooter>
 *       <AdminSidebarUser name="jingsujiang" role="管理员" />
 *       <AdminSidebarFooterAction aria-label="更多"><MoreIcon /></AdminSidebarFooterAction>
 *     </AdminSidebarFooter>
 *   </AdminSidebar>
 *   <AdminSidebarInset>{children}</AdminSidebarInset>
 * </AdminSidebarProvider>
 *
 * SubGroup（二级缩进 + 引导线）：在子项 AdminSidebarMenuButton 上加 className="cp-admin-sidebar-subitem-button"。
 * **子项不显示 icon**（仅保留文本 + badge + 引导线），图标由 CSS 规则 `.cp-admin-sidebar-subitem-button > .cp-admin-sidebar-icon { display: none }` 强制隐藏。
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

/* ─── 本地工具：cn / 轻量 Slot（替代 @/lib/utils 与 @radix-ui/react-slot） ─── */

function cn(...classes: Array<string | false | null | undefined>): string {
  return classes.filter(Boolean).join(" ");
}

type SlotProps = React.HTMLAttributes<HTMLElement> & {
  children?: React.ReactNode;
};

const Slot = React.forwardRef<HTMLElement, SlotProps>(({ children, ...props }, ref) => {
  if (React.isValidElement(children)) {
    const child = children as React.ReactElement<Record<string, unknown>>;
    const childProps = child.props ?? {};
    return React.cloneElement(child, {
      ...props,
      ...childProps,
      ref,
      className: cn(props.className, childProps.className as string | undefined),
      style: { ...(props.style as object), ...(childProps.style as object) },
    } as Record<string, unknown>);
  }
  return null;
});
Slot.displayName = "PortableSlot";

/* ─── Provider / context ─────────────────────────────────────────────────── */

type AdminSidebarContextValue = {
  collapsed: boolean;
  setCollapsed: (collapsed: boolean) => void;
  toggleSidebar: () => void;
};

const AdminSidebarContext = React.createContext<AdminSidebarContextValue | null>(null);

function useAdminSidebar() {
  const context = React.useContext(AdminSidebarContext);
  if (!context) {
    throw new Error("useAdminSidebar must be used within AdminSidebarProvider.");
  }
  return context;
}

/** 视口宽度 ≤ 此值时侧栏自动收起 */
const ADMIN_SIDEBAR_AUTO_COLLAPSE_BP = 1200;
/** 持久化用户主动设置的折叠态 */
const ADMIN_SIDEBAR_STORAGE_KEY = "admin_sidebar_collapsed";

function readStoredCollapsed(): boolean | null {
  if (typeof window === "undefined") return null;
  const v = window.localStorage.getItem(ADMIN_SIDEBAR_STORAGE_KEY);
  if (v === "true") return true;
  if (v === "false") return false;
  return null;
}

function AdminSidebarProvider({
  defaultCollapsed = false,
  collapsed: collapsedProp,
  onCollapsedChange,
  children,
}: {
  defaultCollapsed?: boolean;
  collapsed?: boolean;
  onCollapsedChange?: (collapsed: boolean) => void;
  children: React.ReactNode;
}) {
  const [internalCollapsed, setInternalCollapsed] = React.useState(() => {
    const stored = readStoredCollapsed();
    if (stored !== null) return stored;
    if (typeof window !== "undefined") {
      return window.matchMedia(`(max-width: ${ADMIN_SIDEBAR_AUTO_COLLAPSE_BP}px)`).matches;
    }
    return defaultCollapsed;
  });
  const collapsed = collapsedProp ?? internalCollapsed;

  const setCollapsed = React.useCallback(
    (nextCollapsed: boolean) => {
      onCollapsedChange?.(nextCollapsed);
      if (collapsedProp === undefined) {
        setInternalCollapsed(nextCollapsed);
        if (typeof window !== "undefined") {
          window.localStorage.setItem(ADMIN_SIDEBAR_STORAGE_KEY, String(nextCollapsed));
        }
      }
    },
    [collapsedProp, onCollapsedChange]
  );

  const toggleSidebar = React.useCallback(() => {
    setCollapsed(!collapsed);
  }, [collapsed, setCollapsed]);

  React.useEffect(() => {
    if (collapsedProp !== undefined) return;
    const mql = window.matchMedia(`(max-width: ${ADMIN_SIDEBAR_AUTO_COLLAPSE_BP}px)`);
    const handler = (e: MediaQueryListEvent) => {
      setInternalCollapsed(e.matches);
      window.localStorage.setItem(ADMIN_SIDEBAR_STORAGE_KEY, String(e.matches));
    };
    mql.addEventListener("change", handler);
    return () => mql.removeEventListener("change", handler);
  }, [collapsedProp]);

  const value = React.useMemo(
    () => ({ collapsed, setCollapsed, toggleSidebar }),
    [collapsed, setCollapsed, toggleSidebar]
  );

  return (
    <AdminSidebarContext.Provider value={value}>{children}</AdminSidebarContext.Provider>
  );
}

/** 收起/展开图标 */
function SidebarCollapseIcon({ className }: { className?: string }) {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" aria-hidden="true" className={className}>
      <path d="M2.5 4.16663H17.5M7.5 9.99996H17.5M2.5 15.8333H17.5" stroke="currentColor" strokeWidth="1.25" strokeLinecap="square" />
      <path d="M3.97314 8.52673L2.5 9.9999L3.97314 11.4731" stroke="currentColor" strokeWidth="1.25" strokeLinecap="square" />
    </svg>
  );
}

/* ─── Sidebar 壳 ─────────────────────────────────────────────────────────── */

const AdminSidebar = React.forwardRef<HTMLElement, React.ComponentProps<"aside">>(
  ({ className, style, ...props }, ref) => {
    const { collapsed } = useAdminSidebar();
    return (
      <aside
        ref={ref}
        data-slot="admin-sidebar"
        data-state={collapsed ? "collapsed" : "expanded"}
        className={cn(
          "fixed inset-y-0 left-0 z-40 flex flex-col overflow-hidden border-r bg-[var(--cp-admin-sidebar-bg)] transition-[width] duration-300",
          "border-[var(--cp-admin-sidebar-border)]",
          collapsed ? "w-[var(--cp-admin-sidebar-w-collapsed)]" : "w-[var(--cp-admin-sidebar-w)]",
          className
        )}
        style={style}
        {...props}
      />
    );
  }
);
AdminSidebar.displayName = "AdminSidebar";

const AdminSidebarHeader = React.forwardRef<HTMLDivElement, React.ComponentProps<"div">>(
  ({ className, ...props }, ref) => {
    const { collapsed } = useAdminSidebar();
    return (
      <div
        ref={ref}
        data-slot="admin-sidebar-header"
        className={cn(
          "flex shrink-0 flex-col",
          collapsed ? "items-center px-2 py-3" : "h-[var(--cp-admin-sidebar-header-h)] px-0 py-0",
          className
        )}
        {...props}
      />
    );
  }
);
AdminSidebarHeader.displayName = "AdminSidebarHeader";

const AdminSidebarLogo = React.forwardRef<SVGSVGElement, React.ComponentProps<"svg">>(
  ({ className, ...props }, ref) => (
    <svg
      ref={ref}
      width="36"
      height="28"
      viewBox="0 0 36 28"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      className={className}
      {...props}
    >
      <path d="M23.6238 20.7044L23.6252 20.7055L20.4357 24.3263C18.6332 26.3722 16.038 27.5442 13.3114 27.5442H9.73657V23.6673H13.3114C14.9247 23.6673 16.4602 22.9739 17.5267 21.7634L20.8349 18.0085C21.8951 16.8052 23.4215 16.1157 25.0252 16.1157H26.7467C27.1043 16.1157 27.2948 16.5374 27.0584 16.8057L23.6238 20.7044Z" fill="url(#cp-admin-sidebar-logo-a)" />
      <path d="M26.2641 12.2392H22.6892C21.076 12.2393 19.5405 12.9326 18.474 14.1431L15.1658 17.898C14.1056 19.1013 12.5792 19.7908 10.9754 19.7908H9.24722C8.88966 19.7908 8.69918 19.3691 8.93554 19.1008L12.1913 15.4055C12.2138 15.3827 12.2357 15.3593 12.257 15.3351L15.565 11.5802C17.3674 9.53434 19.9626 8.36233 22.6892 8.3623H26.2641V12.2392Z" fill="url(#cp-admin-sidebar-logo-b)" />
      <path d="M26.2644 8.36292C31.5608 8.36308 35.8542 12.6573 35.8543 17.9537C35.854 23.1556 31.713 27.3896 26.5467 27.5397C26.5374 27.5399 26.5276 27.5394 26.5183 27.5397V27.5406H26.5008V27.5436H23.699C23.3415 27.5435 23.1512 27.1214 23.3875 26.8531L26.0301 23.8541C26.1336 23.7366 26.2818 23.6686 26.4383 23.6666L26.4392 23.6637C29.5129 23.5717 31.9771 21.0504 31.9773 17.9537C31.9773 14.7985 29.4197 12.24 26.2644 12.2399V8.36292Z" fill="url(#cp-admin-sidebar-logo-c)" />
      <path d="M17.9992 0.0976562C21.9114 0.0976562 25.3907 1.94864 27.611 4.82213C27.8145 5.08551 27.62 5.45547 27.2872 5.45547H26.2944C26.2843 5.45545 26.2742 5.4552 26.2641 5.4552H22.8936C22.7803 5.4552 22.67 5.42001 22.5758 5.35719C21.2655 4.48395 19.6919 3.97458 17.9992 3.97458C13.4637 3.97466 9.78227 7.62843 9.73635 12.153V12.2398C6.58097 12.2398 4.02296 14.7979 4.02292 17.9532C4.02296 21.1086 6.58097 23.6664 9.73635 23.6664V27.5433C4.43981 27.5433 0.146034 23.2498 0.145996 17.9532C0.146024 13.8692 2.69891 10.3818 6.29591 8.99895C7.71347 3.86665 12.416 0.0977252 17.9992 0.0976562Z" fill="#1447E6" />
      <defs>
        <linearGradient id="cp-admin-sidebar-logo-a" x1="11.6207" y1="25.6058" x2="28.3904" y2="25.6058" gradientUnits="userSpaceOnUse">
          <stop stopColor="#1447E6" />
          <stop offset="1" stopColor="black" />
        </linearGradient>
        <linearGradient id="cp-admin-sidebar-logo-b" x1="24.3799" y1="10.3007" x2="7.60998" y2="10.3007" gradientUnits="userSpaceOnUse">
          <stop stopColor="#1447E6" />
          <stop offset="1" stopColor="black" />
        </linearGradient>
        <linearGradient id="cp-admin-sidebar-logo-c" x1="28.3202" y1="12.5381" x2="28.3483" y2="27.0011" gradientUnits="userSpaceOnUse">
          <stop stopColor="#1447E6" />
          <stop offset="1" stopColor="black" />
        </linearGradient>
      </defs>
    </svg>
  )
);
AdminSidebarLogo.displayName = "AdminSidebarLogo";

const AdminSidebarBrand = React.forwardRef<
  HTMLDivElement,
  React.ComponentProps<"div"> & { asChild?: boolean }
>(({ asChild = false, className, ...props }, ref) => {
  const Comp = (asChild ? Slot : "div") as React.ElementType;
  return (
    <Comp
      ref={ref}
      data-slot="admin-sidebar-brand"
      className={cn(
        "group flex min-w-0 items-center gap-2.5 rounded-[4px] text-left outline-none transition-colors focus-visible:ring-2 focus-visible:ring-[var(--cp-brand-blue,#1447e6)]",
        className
      )}
      {...props}
    />
  );
});
AdminSidebarBrand.displayName = "AdminSidebarBrand";

const AdminSidebarHeaderAction = React.forwardRef<
  HTMLButtonElement,
  React.ComponentProps<"button"> & { asChild?: boolean }
>(({ asChild = false, className, ...props }, ref) => {
  const Comp = (asChild ? Slot : "button") as React.ElementType;
  return (
    <Comp
      ref={ref}
      data-slot="admin-sidebar-header-action"
      className={cn(
        "flex size-8 shrink-0 items-center justify-center rounded-[4px] border border-[var(--cp-admin-sidebar-action-border)] bg-[var(--cp-admin-sidebar-action-bg)] text-[var(--cp-admin-sidebar-fg)] outline-none transition-[color,box-shadow] duration-150 hover:text-[var(--cp-admin-sidebar-fg)] focus-visible:ring-2 focus-visible:ring-[var(--cp-brand-blue,#1447e6)] [&>svg]:size-4",
        className
      )}
      {...props}
    />
  );
});
AdminSidebarHeaderAction.displayName = "AdminSidebarHeaderAction";

const AdminSidebarContent = React.forwardRef<HTMLElement, React.ComponentProps<"nav">>(
  ({ className, onScroll, onWheel, ...props }, ref) => {
    const [isScrolling, setIsScrolling] = React.useState(false);
    const hideTimerRef = React.useRef<number | null>(null);

    const showScrollbarTemporarily = React.useCallback(() => {
      setIsScrolling(true);
      if (hideTimerRef.current) window.clearTimeout(hideTimerRef.current);
      hideTimerRef.current = window.setTimeout(() => {
        setIsScrolling(false);
        hideTimerRef.current = null;
      }, 700);
    }, []);

    React.useEffect(() => {
      return () => {
        if (hideTimerRef.current) window.clearTimeout(hideTimerRef.current);
      };
    }, []);

    return (
      <nav
        ref={ref}
        data-slot="admin-sidebar-content"
        data-scrolling={isScrolling ? "true" : "false"}
        className={cn("cp-scrollbar-on-hover min-h-0 flex-1 overflow-y-auto px-4 mt-4 mb-3", className)}
        onScroll={(event) => {
          showScrollbarTemporarily();
          onScroll?.(event);
        }}
        onWheel={(event) => {
          showScrollbarTemporarily();
          onWheel?.(event);
        }}
        {...props}
      />
    );
  }
);
AdminSidebarContent.displayName = "AdminSidebarContent";

/* ─── Group（可折叠） ────────────────────────────────────────────────────── */

type AdminSidebarGroupContextValue = {
  open: boolean;
  setOpen: (open: boolean) => void;
};

const AdminSidebarGroupContext = React.createContext<AdminSidebarGroupContextValue | null>(null);

function useAdminSidebarGroup() {
  const context = React.useContext(AdminSidebarGroupContext);
  if (!context) {
    throw new Error("useAdminSidebarGroup must be used within AdminSidebarGroup.");
  }
  return context;
}

const AdminSidebarGroup = React.forwardRef<
  HTMLDivElement,
  React.ComponentProps<"div"> & { defaultOpen?: boolean }
>(({ defaultOpen = true, className, ...props }, ref) => {
  const [open, setOpen] = React.useState(defaultOpen);
  const value = React.useMemo(() => ({ open, setOpen }), [open]);
  return (
    <AdminSidebarGroupContext.Provider value={value}>
      <div
        ref={ref}
        data-slot="admin-sidebar-group"
        data-state={open ? "open" : "closed"}
        className={cn("mb-5 last:mb-0", className)}
        {...props}
      />
    </AdminSidebarGroupContext.Provider>
  );
});
AdminSidebarGroup.displayName = "AdminSidebarGroup";

const AdminSidebarGroupTrigger = React.forwardRef<HTMLButtonElement, React.ComponentProps<"button">>(
  ({ className, children, ...props }, ref) => {
    const { open, setOpen } = useAdminSidebarGroup();
    return (
      <button
        ref={ref}
        type="button"
        data-slot="admin-sidebar-group-trigger"
        data-state={open ? "open" : "closed"}
        className={cn(
          "mb-1 flex h-5 w-full items-center justify-between px-2 text-left text-xs font-normal leading-[1.5] tracking-[0.015em] text-[var(--cp-admin-sidebar-muted)] outline-none transition-colors hover:text-[var(--cp-admin-sidebar-fg)] focus-visible:ring-2 focus-visible:ring-[var(--cp-brand-blue,#1447e6)]",
          className
        )}
        onClick={() => setOpen(!open)}
        {...props}
      >
        <span>{children}</span>
        <svg
          width="12"
          height="12"
          viewBox="0 0 12 12"
          fill="none"
          aria-hidden="true"
          className={cn("size-3 shrink-0 transition-transform text-[var(--cp-admin-sidebar-muted)]", open ? "" : "rotate-180")}
        >
          <path d="M3 7.5L6 4.5L9 7.5" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </button>
    );
  }
);
AdminSidebarGroupTrigger.displayName = "AdminSidebarGroupTrigger";

const AdminSidebarGroupLabel = React.forwardRef<HTMLDivElement, React.ComponentProps<"div">>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      data-slot="admin-sidebar-group-label"
      className={cn("mb-2 h-5 px-2 text-xs font-normal leading-[1.5] tracking-[0.015em] text-[var(--cp-admin-sidebar-muted)]", className)}
      {...props}
    />
  )
);
AdminSidebarGroupLabel.displayName = "AdminSidebarGroupLabel";

const AdminSidebarGroupContent = React.forwardRef<HTMLDivElement, React.ComponentProps<"div">>(
  ({ className, ...props }, ref) => {
    const { open } = useAdminSidebarGroup();
    if (!open) return null;
    return (
      <div ref={ref} data-slot="admin-sidebar-group-content" className={cn("w-full", className)} {...props} />
    );
  }
);
AdminSidebarGroupContent.displayName = "AdminSidebarGroupContent";

/* ─── Menu ───────────────────────────────────────────────────────────────── */

const AdminSidebarMenu = React.forwardRef<HTMLUListElement, React.ComponentProps<"ul">>(
  ({ className, ...props }, ref) => (
    <ul ref={ref} data-slot="admin-sidebar-menu" className={cn("flex w-full flex-col gap-0.5", className)} {...props} />
  )
);
AdminSidebarMenu.displayName = "AdminSidebarMenu";

const AdminSidebarMenuItem = React.forwardRef<HTMLLIElement, React.ComponentProps<"li">>(
  ({ className, ...props }, ref) => (
    <li ref={ref} data-slot="admin-sidebar-menu-item" className={cn("relative", className)} {...props} />
  )
);
AdminSidebarMenuItem.displayName = "AdminSidebarMenuItem";

const AdminSidebarMenuButton = React.forwardRef<
  HTMLButtonElement,
  React.ComponentProps<"button"> & {
    asChild?: boolean;
    isActive?: boolean;
    tone?: "default" | "muted";
  }
>(({ asChild = false, isActive = false, tone = "default", className, ...props }, ref) => {
  const Comp = (asChild ? Slot : "button") as React.ElementType;
  return (
    <Comp
      ref={ref}
      data-slot="admin-sidebar-menu-button"
      data-active={isActive}
      aria-current={isActive ? "page" : undefined}
      className={cn(
        "flex h-[var(--cp-admin-sidebar-item-h)] w-full items-center gap-2 rounded-[var(--cp-admin-sidebar-item-r)] px-2 text-left text-sm leading-[22px] tracking-[0.005em] text-[var(--cp-admin-sidebar-fg)] outline-none transition-all duration-150 focus-visible:ring-2 focus-visible:ring-[var(--cp-brand-blue,#1447e6)] [&>img]:size-4 [&>img]:shrink-0 [&>svg]:size-4 [&>svg]:shrink-0",
        tone === "muted" && "text-[var(--cp-admin-sidebar-muted)]",
        className
      )}
      {...props}
    />
  );
});
AdminSidebarMenuButton.displayName = "AdminSidebarMenuButton";

const AdminSidebarBadge = React.forwardRef<
  HTMLSpanElement,
  React.ComponentProps<"span"> & { variant?: "new" | "coming-soon" | "custom" }
>(({ variant = "new", className, children, ...props }, ref) => (
  <span
    ref={ref}
    data-slot="admin-sidebar-badge"
    data-variant={variant}
    className={cn(
      "ml-auto inline-flex h-[18px] w-auto shrink-0 items-center justify-center rounded-full px-1.5 text-[11px] font-normal leading-none tracking-[0.015em] transition-colors duration-150",
      className
    )}
    {...props}
  >
    {children ?? (variant === "new" ? "New" : "即将开放")}
  </span>
));
AdminSidebarBadge.displayName = "AdminSidebarBadge";

/* ─── Footer ─────────────────────────────────────────────────────────────── */

const AdminSidebarFooter = React.forwardRef<HTMLDivElement, React.ComponentProps<"div">>(
  ({ className, children, ...props }, ref) => (
    <div
      ref={ref}
      data-slot="admin-sidebar-footer"
      className={cn(
        "relative flex h-[var(--cp-admin-sidebar-footer-h)] shrink-0 items-center gap-2 px-4 before:absolute before:left-4 before:right-4 before:top-0 before:h-px before:bg-[var(--cp-admin-sidebar-border)] before:content-['']",
        className
      )}
      {...props}
    >
      {children}
    </div>
  )
);
AdminSidebarFooter.displayName = "AdminSidebarFooter";

const AdminSidebarUser = React.forwardRef<
  HTMLDivElement,
  React.ComponentProps<"div"> & { name: string; role: string; fallback?: string }
>(({ name, role, fallback, className, ...props }, ref) => (
  <div
    ref={ref}
    data-slot="admin-sidebar-user"
    className={cn("flex min-w-0 flex-1 items-center gap-2.5", className)}
    {...props}
  >
    <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-[var(--cp-admin-sidebar-avatar-bg)] font-mono text-[14.22px] font-normal leading-none text-[var(--cp-admin-sidebar-avatar-fg)]">
      {fallback ?? name.charAt(0).toUpperCase()}
    </div>
    <div className="min-w-0 flex-1">
      <p className="truncate text-sm font-medium leading-5 tracking-[0.005em] text-[var(--cp-admin-sidebar-fg)]">{name}</p>
      <p className="truncate text-xs font-normal leading-5 tracking-[0.015em] text-[var(--cp-admin-sidebar-fg)]">{role}</p>
    </div>
  </div>
));
AdminSidebarUser.displayName = "AdminSidebarUser";

const AdminSidebarFooterAction = React.forwardRef<
  HTMLButtonElement,
  React.ComponentProps<"button"> & { asChild?: boolean }
>(({ asChild = false, className, ...props }, ref) => {
  const Comp = (asChild ? Slot : "button") as React.ElementType;
  return (
    <Comp
      ref={ref}
      data-slot="admin-sidebar-footer-action"
      className={cn(
        "flex size-8 shrink-0 items-center justify-center rounded-[4px] text-[var(--cp-admin-sidebar-muted)] outline-none transition-colors hover:bg-[var(--bg-grey-hover,#f5f6fa)] hover:text-gray-900 focus-visible:ring-2 focus-visible:ring-[var(--cp-brand-blue,#1447e6)] [&>svg]:size-4",
        className
      )}
      {...props}
    />
  );
});
AdminSidebarFooterAction.displayName = "AdminSidebarFooterAction";

/* ─── Inset（主内容区，与 sidebar 宽度联动） ─────────────────────────────── */

const AdminSidebarInset = React.forwardRef<HTMLElement, React.ComponentProps<"main">>(
  ({ className, style, ...props }, ref) => {
    const { collapsed } = useAdminSidebar();
    return (
      <main
        ref={ref}
        data-slot="admin-sidebar-inset"
        data-state={collapsed ? "collapsed" : "expanded"}
        className={cn("h-screen flex flex-col flex-1 min-w-0 overflow-x-hidden transition-[margin-left] duration-300", className)}
        style={{
          marginLeft: collapsed ? "var(--cp-admin-sidebar-w-collapsed)" : "var(--cp-admin-sidebar-w)",
          ...style,
        }}
        {...props}
      />
    );
  }
);
AdminSidebarInset.displayName = "AdminSidebarInset";

/* ─── Trigger（折叠/展开按钮） ───────────────────────────────────────────── */

const AdminSidebarTrigger = React.forwardRef<
  HTMLButtonElement,
  React.ComponentProps<"button"> & { asChild?: boolean }
>(({ asChild = false, className, children, onClick, ...props }, ref) => {
  const { toggleSidebar } = useAdminSidebar();
  const Comp = (asChild ? Slot : "button") as React.ElementType;
  return (
    <Comp
      ref={ref}
      data-slot="admin-sidebar-trigger"
      className={cn(
        "flex size-8 items-center justify-center rounded-[4px] text-[var(--cp-admin-sidebar-muted)] transition-colors hover:bg-gray-50 hover:text-gray-900 [&>svg]:size-4",
        className
      )}
      onClick={(event: React.MouseEvent<HTMLButtonElement>) => {
        onClick?.(event);
        toggleSidebar();
      }}
      {...props}
    >
      {children ?? <SidebarCollapseIcon className="size-4" />}
    </Comp>
  );
});
AdminSidebarTrigger.displayName = "AdminSidebarTrigger";

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
};
