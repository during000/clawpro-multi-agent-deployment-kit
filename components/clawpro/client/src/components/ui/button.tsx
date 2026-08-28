import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

/**
 * Button 组件
 *
 * 在 shadcn 默认 variant/size 之外，扩展了两组「ClawPro Figma 按钮规范」变体：
 *   - `claw-*`   ：管控端（Admin）按钮，4px 圆角，对齐 Figma ComponentSet 317:1051
 *   - `tenant-*` ：用户端（Tenant）按钮，全圆角胶囊，对齐 Figma 0522 修改点 / node 1141:11617~1141:11909
 *
 * 设计令牌（claw-*，来自 Figma 317:1051；颜色与 spec §3 / portable tokens 对齐）：
 *   ┌─────────────────────────┬──────────────────────────────────────────────────────────────┐
 *   │ Token                   │ Value                                                         │
 *   ├─────────────────────────┼──────────────────────────────────────────────────────────────┤
 *   │ claw-outline / bg       │ #FFFFFF                                                       │
 *   │ claw-outline / hover bg │ var(--bg-grey-hover) / #F5F6FA                                │
 *   │ claw-outline / border   │ 1px solid #EAEEF4                                             │
 *   │ claw-outline / hover    │ 1px solid #EAEEF4                                             │
 *   │ claw-outline / text     │ var(--cp-brand-black) = #020617                               │
 *   │ claw-primary / bg       │ var(--cp-brand-black) = #020617（spec §3）                    │
 *   │ claw-primary / hover bg │ #404040（与 dialog-confirm 对齐）                              │
 *   │ claw-primary / text     │ #FFFFFF                                                       │
 *   │ 圆角                     │ 4px（已由基类提供）                                            │
 *   │ icon size               │ 16×16（已由基类 [&_svg:not([size-])]:size-4 提供）              │
 *   └─────────────────────────┴──────────────────────────────────────────────────────────────┘
 *
 * 设计令牌（tenant-*，来自 Figma 0522 修改点；与 claw 故意隔离，颜色独立维护）：
 *   ┌──────────────────────────┬─────────────────────────────────────────────────────────────┐
 *   │ Token                    │ Value                                                        │
 *   ├──────────────────────────┼─────────────────────────────────────────────────────────────┤
 *   │ tenant-* / 圆角           │ rounded-full（全圆角胶囊，覆盖基类 4px）                     │
 *   │ tenant-primary / bg      │ #0A0A0A                                                      │
 *   │ tenant-primary / hover   │ #333333                                                      │
 *   │ tenant-outline / bg      │ #FFFFFF + 1px #EAEEF4                                        │
 *   │ tenant-outline / hover   │ var(--bg-grey-hover) + 1px #EAEEF4 + shadow 0 1px 3px rgba(0,0,0,0.08) │
 *   │ tenant-destructive / bg  │ #D42A1E → hover #B91C1C                                      │
 *   │ tenant-ghost / bg        │ transparent → hover #F5F5F5                                  │
 *   └──────────────────────────┴─────────────────────────────────────────────────────────────┘
 *
 * 使用示例（管控端）：
 *   <Button variant="claw-outline" size="claw">详细配置</Button>
 *   <Button variant="claw-primary" size="claw-lg"><Plus /> 创建 Agent</Button>
 *
 * 使用示例（用户端）：
 *   <Button variant="tenant-outline" size="claw">详细配置</Button>
 *   <Button variant="tenant-primary" size="claw-lg"><Plus /> 创建 Agent</Button>
 *   <Button variant="tenant-destructive" size="claw">删除</Button>
 */

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-[4px] text-sm font-medium transition-all disabled:cursor-not-allowed disabled:pointer-events-auto [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
  {
    variants: {
      variant: {
        default:
          // 黑底主按钮（与 claw-primary / dialog-confirm 同 hover 口径）
          "bg-[#0A0A0A] text-white font-normal border-0 " +
          "hover:bg-[#404040] " +
          "active:bg-[#000000] " +
          "disabled:bg-[#0A0A0A]/40 disabled:text-white/50 disabled:opacity-100",
        destructive:
          "bg-[#d42a1e] text-white font-normal border-0 " +
          "hover:bg-[#b91c1c] " +
          "active:bg-[#991b1b] " +
          "disabled:bg-[#d42a1e]/40 disabled:text-white/60 disabled:opacity-100",

        /**
         * 次级危险按钮（outline-destructive）
         * - 红色描边 + 红色文字 + 红色图标 + 浅粉色背景
         * - 用途：卡片/详情页中的次级危险操作（如删除）
         */
        "outline-destructive":
          "bg-white border border-[#FEE2E2] text-[#DC2626] font-normal " +
          "[&_svg]:text-[#DC2626] " +
          "hover:bg-[#FEF2F2] hover:border-[#FEE2E2] " +
          "active:bg-[#FEE2E2] active:border-[#F87171] " +
          "disabled:bg-white disabled:border-[#FEE2E2]/60 disabled:text-[#DC2626]/40 disabled:opacity-100 disabled:[&_svg]:opacity-40",

        outline:
          "bg-white border border-gray-200 text-gray-950 font-normal " +
          "hover:bg-[var(--bg-grey-hover)] hover:border-gray-200 " +
          "active:bg-white active:border-gray-200 " +
          "disabled:bg-white disabled:border-gray-200 disabled:text-gray-400 disabled:opacity-100 disabled:[&_svg]:opacity-30",
        secondary:
          "bg-[#f5f5f5] border border-[#e3e3e3] text-gray-950 font-normal " +
          "hover:bg-[#ebebeb] hover:border-[#d4d4d4] " +
          "active:bg-[#e0e0e0] " +
          "disabled:bg-[#f5f5f5] disabled:text-gray-400 disabled:opacity-100",
        ghost:
          "text-gray-950 font-normal " +
          "hover:bg-[#f5f5f5] " +
          "active:bg-[#ebebeb] " +
          "disabled:text-gray-400 disabled:opacity-100",
        /**
         * 分类筛选 Tab（管控端 / Admin）
         * - 形态：4px 方角（沿用基类 rounded-[4px]，对齐管控端规范）
         * - normal: 白底 + #EAEEF4 边 + #020617 字
         * - hover : 边色加深至 #020617
         * - active / data-state=active: 黑底白字（被选中态）
         * - disabled: 白底 + 灰字
         * - 使用场景：admin 端分类筛选条
         *   （SkillListTab / PublicSkillLibraryTab / EditCategoriesDialog）
         * - 用户端胶囊版本请用 `tenant-plain`
         */
        plain:
          "bg-white border border-[#EAEEF4] text-gray-950 font-normal " +
          "hover:border-gray-950 " +
          "active:bg-gray-950 active:border-gray-950 active:text-white " +
          "data-[state=active]:bg-gray-950 data-[state=active]:border-gray-950 data-[state=active]:text-white " +
          "disabled:bg-white disabled:border-[#EAEEF4] disabled:text-gray-400 disabled:opacity-100",
        link:
          // link 형태：무 padding 무 높이，직接作为内联文字渲染（用 ! 提升优先级以胜过 size variant 的 px-6 / h-9）
          "!px-0 !py-0 !h-auto has-[>svg]:!px-0 text-[#355EF1] font-normal underline-offset-4 " +
          "hover:underline " +
          "active:text-[#0a226f] " +
          "disabled:text-[#A3A3A3] disabled:opacity-100 disabled:no-underline",

        /**
         * 黑色文字按钮（用于表格操作列）
         * - normal: #020617 字色，无背景无边框
         * - hover : #525252 字色
         * - click/active: #020617 字色 + 下划线
         * - disabled: rgba(2,6,23,0.3) 字色
         */
        "link-dark":
          // link 形态：无 padding 无高度（用 ! 提升优先级以胜过 size variant 的 px-6 / h-9）
          "!px-0 !py-0 !h-auto has-[>svg]:!px-0 text-gray-950 font-normal underline-offset-4 " +
          "hover:text-gray-600 " +
          "active:text-gray-950 active:underline " +
          "disabled:text-gray-400 disabled:no-underline disabled:opacity-100",

        /* ============================================================== */
        /*  Figma「按钮」ComponentSet 317:1051 对齐变体                       */
        /* ============================================================== */

        /**
         * 线性描边（次级按钮）
         * - normal: 白底 + #EAEEF4 边 + var(--cp-brand-black) 字（spec §3 = #020617）
         * - hover : #f5f5f5 底 + #e3e3e3 边
         * - active: 白底 + #e3e3e3 边
         * - disabled: 白底 + #EAEEF4 边 + rgba(2,6,23,0.3) 字
         */
        "claw-outline":
          "bg-white border border-[#EAEEF4] text-[var(--cp-brand-black)] font-normal " +
          "hover:bg-[var(--bg-grey-hover)] hover:border-[#EAEEF4] " +
          "active:bg-white active:border-[#EAEEF4] " +
          "disabled:bg-white disabled:border-[#EAEEF4] disabled:text-gray-400 disabled:opacity-100 disabled:[&_svg]:opacity-50",

        /**
         * 深色填充（主按钮）
         * - normal: 品牌黑底 var(--cp-brand-black) = #020617（spec §3）+ 白字
         * - hover : #404040（与 dialog-confirm 对齐 · 2026-06-10 调整，原 #1a1a1a 视觉过深易与 active 混淆）
         * - active: #000000
         * - disabled: 品牌黑/40 半透明 + 半透明白字（走 token，宿主仓覆盖时跟随）
         */
        "claw-primary":
          "bg-[var(--cp-brand-black)] text-white font-normal border-0 " +
          "hover:bg-[#404040] " +
          "active:bg-[#000000] " +
          "disabled:bg-[var(--cp-brand-black)]/40 disabled:text-white/50 disabled:opacity-100",

        /**
         * 普通弹窗主按钮
         * - normal: 品牌黑底 var(--cp-brand-black) = #020617 + 白字（与 claw-primary 同 token）
         * - hover : 深灰底
         * - disabled: 中灰底 + 白字
         */
        "dialog-confirm":
          "bg-[var(--cp-brand-black)] text-white font-normal border-0 " +
          "hover:bg-[#404040] " +
          "active:bg-[#262626] " +
          "disabled:bg-[#A3A3A3] disabled:text-white disabled:opacity-100",

        /* ============================================================== */
        /*  用户端（Tenant）按钮变体 — Figma 0522 修改点 / node 1141:11617    */
        /*  全圆角胶囊（rounded-full），与 claw-* 仅圆角差异                 */
        /*  规范来源：SKILL-TENANT.md §3                                    */
        /*                                                                */
        /*  ⚠ 与 claw-* 故意隔离：tenant 系列底色保留 #0A0A0A（纯黑）/      */
        /*  #D42A1E（品牌红）/ #E5E7EB（gray-200 边），不与 spec            */
        /*  --cp-brand-black / --cp-text-danger / --cp-border 对齐。       */
        /*  原因：tenant 视觉以 Figma 0522 设计稿为最终口径，hex 沿用稿件，  */
        /*  避免随宿主仓主题色（admin 侧）漂移；如需统一请走单独 spec PR。   */
        /* ============================================================== */

        /**
         * 用户端主按钮（tenant-primary）
         * - 与 claw-primary 同色（纯黑 #0A0A0A + 白字，hover #333333）
         * - 圆角：rounded-full（覆盖基类 4px）
         * - 用途：用户端业务页 CTA、弹窗确认、表单提交
         */
        "tenant-primary":
          "!rounded-full bg-[#0A0A0A] text-white font-normal border-0 " +
          "hover:bg-[#333333] " +
          "active:bg-[#1a1a1a] " +
          "disabled:bg-[#0A0A0A]/50 disabled:text-white/50 disabled:opacity-100",

        /**
         * 用户端线性描边按钮（tenant-outline）
         * - 与 claw-outline 同色（白底 / #EAEEF4 边 / #020617 字）
         * - 圆角：rounded-full
         * - hover：#F5F5F5 + 描边 + 轻阴影 0 1px 3px rgba(0,0,0,0.08)
         * - 用途：用户端业务页次级按钮、弹窗取消、表单重置
         */
        "tenant-outline":
          "!rounded-full bg-white border border-gray-200 text-gray-950 font-normal " +
          "hover:bg-[var(--bg-grey-hover)] hover:border-gray-200 hover:[box-shadow:0_1px_3px_rgba(0,0,0,0.08)] " +
          "active:bg-white active:border-gray-200 active:[box-shadow:none] " +
          "disabled:bg-white disabled:border-gray-200 disabled:text-gray-400 disabled:opacity-100 disabled:[box-shadow:none] disabled:[&_svg]:opacity-30",

        /**
         * 用户端中等强调描边按钮（tenant-outline-strong）
         * - 介于 tenant-outline（最轻）与 tenant-primary（最重）之间
         * - 白底 + 深灰描边（#A3A3A3）+ 黑字 font-medium
         * - hover：描边再加深 + 阴影
         * - 用途：需要比 outline 更突出但不需要纯黑实心的操作按钮
         */
        "tenant-outline-strong":
          "!rounded-full bg-white border border-[#cbcbcb] text-gray-950 font-medium " +
          "hover:bg-[var(--bg-grey-hover)] hover:border-[#cbcbcb] hover:[box-shadow:0_1px_3px_rgba(0,0,0,0.08)] " +
          "active:bg-white active:border-[#cbcbcb] active:[box-shadow:none] " +
          "disabled:bg-white disabled:border-[#cbcbcb] disabled:text-gray-400 disabled:opacity-100 disabled:[box-shadow:none] disabled:[&_svg]:opacity-30",

        /**
         * 用户端线性描边按钮（tenant-outline-r20） — [Figma 1077-33986] 卡片底部专用
         * - 颜色规格与 tenant-outline 一致（白底 / #EAEEF4 边 / #020617 字）
         * - 唯一差异：圆角 = 20px（rounded-[20px]），覆盖基类 4px
         * - 用途：AgentCard 底部「设置」/「对话」按钮，对齐 Figma 1077:33986 节点
         *   规范的 20px 圆角，介于 4px（claw）和 full（tenant）之间
         */
        "tenant-outline-r20":
          "!rounded-[20px] bg-white border border-[#cbcbcb] text-gray-950 font-normal " +
          "hover:bg-[var(--bg-grey-hover)] hover:border-[#cbcbcb] " +
          "active:bg-white active:border-[#cbcbcb] active:[box-shadow:none] " +
          "disabled:bg-white disabled:border-[#cbcbcb] disabled:text-gray-400 disabled:opacity-100 disabled:[box-shadow:none] disabled:[&_svg]:opacity-30",

        /**
         * 用户端危险按钮（tenant-destructive）
         * - 与 destructive 同色（#D42A1E 底 + 白字）
         * - 圆角：rounded-full
         * - 用途：用户端删除、注销、清空等危险操作
         */
        "tenant-destructive":
          "!rounded-full bg-[#d42a1e] text-white font-normal border-0 " +
          "hover:bg-[#b91c1c] " +
          "active:bg-[#991b1b] " +
          "disabled:bg-[#d42a1e]/40 disabled:text-white/60 disabled:opacity-100",

        /**
         * 用户端幽灵按钮（tenant-ghost）
         * - 透明底，hover 时浅灰底
         * - 圆角：rounded-full
         * - 用途：用户端工具条 / 卡片角操作 / 极弱视觉权重的按钮
         */
        "tenant-ghost":
          "!rounded-full bg-transparent text-gray-950 font-normal border-0 " +
          "hover:bg-[#f5f5f5] " +
          "active:bg-[#ebebeb] " +
          "disabled:text-gray-400 disabled:opacity-100",

        /**
         * 用户端分类筛选 Tab（tenant-plain，Pill / Chip 形态）
         * - 与管控端 `plain` 唯一差异：圆角 = rounded-full（胶囊）
         * - 颜色 / 状态规范完全一致：白底 + #EAEEF4 边 + #020617 字；
         *   hover 边加深；active / data-state=active 黑底白字；
         *   disabled 浅灰底（#f5f5f5）+ 灰字（对齐 Figma 用户端禁用胶囊样式）
         * - 使用场景：tenant 端「分类筛选条」（SkillSquare / OpenClawDetailGuide 技能分类）
         * - 为什么用 !rounded-full：size="sm" 写死了 rounded-[4px]，cva 拼接顺序 size 在 variant
         *   之后，twMerge 后会覆盖 variant 圆角；tenant-* 系列统一用 !important 兜底（与
         *   tenant-primary / tenant-outline 等同手法）。
         */
        "tenant-plain":
          "!rounded-full bg-white border border-[#EAEEF4] text-gray-950 font-normal " +
          "hover:border-gray-950 " +
          "active:bg-gray-950 active:border-gray-950 active:text-white " +
          "data-[state=active]:bg-gray-950 data-[state=active]:border-gray-950 data-[state=active]:text-white " +
          "disabled:bg-[#f5f5f5] disabled:border-[#EAEEF4] disabled:text-gray-400 disabled:opacity-100",

        /**
         * 用户端纯黑实心按钮（tenant-dialog-confirm）
         * - 与管控端 `dialog-confirm` 同色（纯黑底 #0A0A0A + 白字，hover #404040）
         * - 唯一差异：圆角 = rounded-full（胶囊）
         * - 用途：用户端页面需要纯黑（非渐变）实心按钮的场景，例如
         *   ToolsMcpPanel「添加 MCP」、用户端弹窗主确认按钮等
         */
        "tenant-dialog-confirm":
          "!rounded-full bg-[#0A0A0A] text-white font-normal border-0 " +
          "hover:bg-[#404040] " +
          "active:bg-[#262626] " +
          "disabled:bg-[#A3A3A3] disabled:text-white disabled:opacity-100",
      },
      size: {
        default: "h-9 px-6 py-2 has-[>svg]:px-4",
        sm: "h-8 rounded-[4px] gap-1.5 px-4 has-[>svg]:px-3",
        lg: "h-10 rounded-[4px] px-6 has-[>svg]:px-4",
        icon: "size-9",
        "icon-sm": "size-8",
        "icon-lg": "size-10",

        /* --------------------------- Figma 尺寸 --------------------------- */

        /** 36×hug (中)，padding 8/24，icon-text gap 8 */
        claw: "h-9 gap-2 px-6 py-2",

        /** 32×hug (小)，padding 4/16，icon-text gap 6 */
        "claw-sm": "h-8 gap-1.5 px-4 py-1",

        /** 40×hug (大)，padding 4/18，icon-text gap 8 */
        "claw-lg": "h-10 gap-2 px-[18px] py-1",

        /** 48×36 纯图标（线性描边方形按钮，卡片角落「刷新」） */
        "claw-square": "h-9 w-12 p-0",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

const Button = React.forwardRef<
  HTMLButtonElement,
  React.ComponentProps<"button"> &
    VariantProps<typeof buttonVariants> & {
      asChild?: boolean;
    }
>(function Button({
  className,
  variant,
  size,
  asChild = false,
  ...props
}, ref) {
  const Comp = asChild ? Slot : "button";

  return (
    <Comp
      ref={ref}
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  );
});

type SmallIconButtonState = "default" | "disabled";

type SmallIconStateButtonProps = Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, "children"> & {
  state?: SmallIconButtonState;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
};

const smallIconButtonStateConfig: Record<
  SmallIconButtonState,
  {
    disabled?: boolean;
    className: string;
  }
> = {
  default: {
    className:
      "border-[#D4D4D4] bg-white text-gray-950 hover:border-[#C9C9C9] hover:bg-[var(--bg-grey-hover)] active:bg-[#F5F5F5]",
  },
  disabled: {
    disabled: true,
    className:
      "border-[#D4D4D4] bg-white text-gray-400 disabled:border-[#D4D4D4] disabled:bg-white disabled:text-gray-400 disabled:opacity-100 disabled:[&_svg]:opacity-100",
  },
};

function SmallIconStateButton({
  state = "default",
  label,
  icon: Icon,
  className,
  disabled,
  type = "button",
  ...props
}: SmallIconStateButtonProps) {
  const config = smallIconButtonStateConfig[state];
  const resolvedDisabled = disabled ?? config.disabled ?? false;

  return (
    <button
      type={type}
      disabled={resolvedDisabled}
      className={cn(
        "inline-flex h-6 items-center justify-center gap-1.5 whitespace-nowrap rounded-[4px] border px-2 text-xs font-medium leading-none transition-colors disabled:cursor-not-allowed",
        config.className,
        className,
      )}
      {...props}
    >
      <Icon className="w-3 h-3" />
      {label && <span className="whitespace-nowrap">{label}</span>}
    </button>
  );
}

export { Button, buttonVariants, SmallIconStateButton };
export type { SmallIconButtonState };
