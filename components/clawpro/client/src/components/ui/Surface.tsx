/**
 * Surface（语义化卡片容器）
 * ─────────────────────────────────────────────────────────────────
 * v3 / 0529 设计系统 §5「卡片三态」唯一卡片 API。
 * 业务层一律用本组件，禁止再写 inline boxShadow / Tailwind shadow-md/lg/xl。
 *
 * 卡片三态规范（对齐 Figma EN6XTROqVtXZEDfZ2rLkjr node 1:118）：
 *   - normal：白底 + 1.5px 白色描边 + backdrop-blur(5px) + 4px 圆角
 *   - hover：白底 + 1px #C9D5FC 蓝灰描边 + shadow(0 1px 3px rgba(0,0,0,0.05)) + 微抬
 *   - selected：白底 + 1px #1447E6 品牌蓝描边 + shadow(0 1px 3px rgba(0,0,0,0.05))
 *
 * 6 档语义：
 *   L1 SurfaceCard    管理端 + 全局表层卡片（4px 圆角，对应 --radius-xl）
 *   L2 SurfaceInner   内嵌卡片（卡片内的子卡 / 表格容器，4px 圆角，无阴影）
 *   L3 SurfaceOverlay 浮层（Dialog/Sheet/Drawer/Popover/DropdownMenu，由 shadcn 内部使用）
 *   L4 SurfaceConfig  管理端高亮配置卡（"操作要点""引导卡"等需强调的卡，4px 圆角）
 *   L5 segment 滑块   见 index.css --shadow-segment（直接写在 Tab/Segmented 内部）
 *   L6 TenantCard     【用户端专属】业务列表卡（12px 圆角，三状态 normal/hover/static）
 *                     对齐 SKILL-TENANT.md §5 / Figma 1141:11921 / 1077:33986
 *
 * 圆角分流（0523 修订）：
 *   - 管理端 / 全局：rounded-xl → --radius-xl = 4px（控件层级几何感）
 *   - 用户端业务卡：TenantCard → --radius-card = 12px（柔和感，对齐 Figma）
 *   ⚠️ 历史误注释："SurfaceCard 系列内部使用 rounded-xl 已对齐 12px"——
 *      实际上 --radius-xl 在本项目被压到 4px（见 index.css 第 43 行注释"控件层上限"），
 *      所以 SurfaceCard 实测就是 4px。用户端要 12px 必须用 TenantCard。
 *
 * 修改阴影/描边/圆角：改 index.css 的 --shadow-card / --shadow-card-hover /
 * --shadow-config / --radius-card 即可批量影响全站，无需跨文件搜索替换。
 * ─────────────────────────────────────────────────────────────────
 */
import { forwardRef, type HTMLAttributes } from "react";
import { cn } from "@/lib/utils";

/* ───────────── 公共 props ───────────── */

interface SurfaceBaseProps extends HTMLAttributes<HTMLDivElement> {
  /** 是否启用卡片三态（hover 阴影升档 + 微抬 0.5px）。仅 L1/L4 推荐，且仅当卡片可点击/可交互时开启。 */
  hover?: boolean;
  /** 是否禁用底色（让卡片透明，仅保留描边/阴影框） */
  bare?: boolean;
}

/* ───────────── L1 SurfaceCard ─────────────
 * 用于：管理端 + 全局表层卡片（页面主区块、列表卡、统计卡）。
 *
 * 视觉规范（对齐 Figma EN6XTROqVtXZEDfZ2rLkjr node 1:118）：
 *   - normal：白底 + 1.5px 白色描边 + 4px 圆角（rounded-xl → --radius-xl=4px）
 *   - hover（需启用 hover prop）：1px #C9D5FC 蓝灰描边 + 微阴影
 *   - active/selected：1px #1447E6 品牌蓝描边 + 微阴影（业务侧通过 className 或 data-state 控制）
 *
 * ⚠️ 用户端业务列表卡（Agent 卡片、技能卡片等）请改用 <TenantCard>（12px 圆角，对齐 Figma）。
 */
export const SurfaceCard = forwardRef<HTMLDivElement, SurfaceBaseProps>(
  ({ className, hover, bare, style, ...props }, ref) => {
    return (
      <div
        ref={ref}
        data-surface="card"
        className={cn(
          "rounded-xl border border-[#EAEEF4]",
          !bare && "bg-white",
          hover && "transition-all duration-200 hover:border-[#C9D5FC] hover:shadow-[0px_1px_3px_0px_rgba(0,0,0,0.05)] hover:-translate-y-0.5",
          "data-[state=selected]:border-[#1447E6] data-[state=selected]:shadow-[0px_1px_3px_0px_rgba(0,0,0,0.05)]",
          className,
        )}
        style={style}
        {...props}
      />
    );
  },
);
SurfaceCard.displayName = "SurfaceCard";

/* ───────────── L2 SurfaceInner ─────────────
 * 用于：卡片内的"子卡片/表格容器/组织面板"，无阴影、靠 #F5F5F5 浅描边。
 * 典型场景：模型额度页面里"模型使用汇总""详细使用记录"两个表格容器。
 */
export const SurfaceInner = forwardRef<HTMLDivElement, SurfaceBaseProps>(
  ({ className, bare, style, ...props }, ref) => {
    return (
      <div
        ref={ref}
        data-surface="inner"
        className={cn(
          "rounded-xl border border-[#EAEEF4]",
          !bare && "bg-white",
          className,
        )}
        style={style}
        {...props}
      />
    );
  },
);
SurfaceInner.displayName = "SurfaceInner";

/* ───────────── L3 SurfaceOverlay ─────────────
 * 用于：自定义浮层（自实现 Dropdown / 浮动菜单）。
 * 注意：shadcn 自带 Dialog/Sheet/Drawer/Popover/DropdownMenu 已在
 *       components/ui 内部用 var(--shadow-overlay)，无需手动包一层。
 */
export const SurfaceOverlay = forwardRef<HTMLDivElement, SurfaceBaseProps>(
  ({ className, bare, style, ...props }, ref) => {
    return (
      <div
        ref={ref}
        data-surface="overlay"
        className={cn(
          "rounded-xl border border-[#EAEEF4]",
          !bare && "bg-white",
          className,
        )}
        style={{ boxShadow: "var(--shadow-overlay)", ...style }}
        {...props}
      />
    );
  },
);
SurfaceOverlay.displayName = "SurfaceOverlay";

/* ───────────── L4 SurfaceConfig ─────────────
 * 用于：管理端「操作要点」「引导卡」「Pro 套餐推荐卡」等需要"略强存在感"的卡。
 * 比 L1 略重，但远低于 L3 浮层。
 *
 * ⚠️ 之前一版误把 `shadow-[var(--shadow-config)]` 默认加上，污染了管理端 SurfaceConfig 视觉，
 *    已于 0523 修正回原貌：默认不带 shadow，仅靠浅描边。如需阴影请显式传 className 加。
 */
export const SurfaceConfig = forwardRef<HTMLDivElement, SurfaceBaseProps>(
  ({ className, hover, bare, style, ...props }, ref) => {
    return (
      <div
        ref={ref}
        data-surface="config"
        className={cn(
          "rounded-xl border border-[#EAEEF4]",
          !bare && "bg-white",
          hover && "transition-all duration-200 hover:-translate-y-0.5",
          className,
        )}
        style={style}
        {...props}
      />
    );
  },
);
SurfaceConfig.displayName = "SurfaceConfig";

/* ───────────── L6 TenantCard ─────────────
 * 用户端业务列表卡专属（0523 实现 SKILL-TENANT.md §5）。
 *
 * 与 SurfaceCard 的差异：
 *   - 圆角：12px（var(--radius-card)）vs SurfaceCard 的 4px（rounded-xl/--radius-xl）
 *   - 三状态：normal（默认 + 描边 + 阴影）/ hover（无描边 + 加强阴影 + 微抬）/ static（无阴影）
 *   - 描边：normal=#E2E8F0，static=#EAEEF4，hover=transparent（让阴影替代描边）
 *   - 阴影（normal/active）：var(--shadow-tenant-card) = 0px 1px 4px rgba(0,0,0,0.05)
 *     【0523 修订】严格对齐 Figma 1077:33987 effect_KNJ2UO 单层值；
 *     不再复用 --shadow-card（管理端用，含一层 2px 环形描边阴影，会让 12px 大圆角呈现"灰圈感"）。
 *   - 内边距 / 行间距【0523-2 修订】：默认 padding 20px + column gap 24px，
 *     与 AgentCard（Figma 1077:33986）对齐，业务侧不必再每次 inline 写。
 *     如需自定义，传 `padding="none"` 关闭默认内边距，或通过 className 覆盖。
 *
 * 对齐 Figma 节点：
 *   - 1141:11921 / 1141:12016 / 1141:11970（用户端卡片三态规范）
 *   - 1077:33986（AgentCard 修订版，borderRadius=12px）
 *   - 1077:33987（卡片实例 effect_KNJ2UO 阴影源 of truth，padding=20、gap=24）
 *
 * 仅用于 client/src/pages/tenant/** 与用户端共享组件（AgentCard/MemoryCard 等）。
 * 管理端禁用——会被 SKILL-TENANT.md §16 强制规则拦截。
 */
interface TenantCardProps extends HTMLAttributes<HTMLDivElement> {
  /** 卡片状态。normal=可交互默认 / hover=已悬停（一般由 interactive 自动驱动）/ static=纯展示无阴影 */
  state?: "normal" | "hover" | "static";
  /** 是否启用 hover 动效（无描边 + 加强阴影 + 微抬 + cursor-pointer）。仅 state=normal 时生效 */
  interactive?: boolean;
  /** 透明背景（让卡片融入页面背景，仅保留圆角 + 描边） */
  bare?: boolean;
  /**
   * 内边距 + 子元素纵向间距预设（0523-2 新增，对齐 Figma 1077:33987）：
   *   - "default"（默认）：padding 20px + flex column + gap 24px。AgentCard / 技能卡 / 入口卡等业务列表卡。
   *   - "compact"：padding 16px + gap 12px。次级密集型卡（暂未使用，预留）。
   *   - "none"：不设 padding / 不设 flex / 不设 gap，业务侧完全自定义（适用于带表头的容器卡）。
   * 默认 "default" 即可与 AgentCard 完全一致，业务侧无需再 inline 写 padding/gap。
   */
  padding?: "default" | "compact" | "none";
}

export const TenantCard = forwardRef<HTMLDivElement, TenantCardProps>(
  ({ className, state = "normal", interactive = false, bare, padding = "default", style, ...props }, ref) => {
    const variants = {
      normal: "border border-[#E2E8F0]",
      hover: "border border-transparent",
      static: "border border-gray-200",
    };
    const hoverEffect =
      interactive && state === "normal"
        ? "cursor-pointer hover:border-transparent hover:-translate-y-0.5 " +
          "hover:[box-shadow:0_4px_24px_rgba(0,0,0,0.08),0_0_2px_rgba(0,0,0,0.1)] " +
          "active:translate-y-0 active:[box-shadow:var(--shadow-tenant-card)]"
        : "";
    const shadow =
      state === "static"
        ? ""
        : "[box-shadow:var(--shadow-tenant-card)]";
    const paddingClass =
      padding === "default"
        ? "flex flex-col p-5 gap-6" // 20px / 24px ＝ Figma 1077:33987
        : padding === "compact"
          ? "flex flex-col p-4 gap-3" // 16px / 12px
          : "";

    return (
      <div
        ref={ref}
        data-surface="tenant-card"
        data-state={state}
        className={cn(
          "rounded-[var(--radius-card)] transition-all duration-200",
          variants[state],
          shadow,
          hoverEffect,
          paddingClass,
          !bare && "bg-white",
          className,
        )}
        style={style}
        {...props}
      />
    );
  },
);
TenantCard.displayName = "TenantCard";
