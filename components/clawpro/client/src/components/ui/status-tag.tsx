import * as React from "react";
import { cn } from "@/lib/utils";
import { SmallBodyText } from "@/components/ui/Typography";

/**
 * StatusTag 状态标签组件
 *
 * 分类：
 * - 文本类：mode="text"，14px 彩色纯文本，无底色无圆点（表格内状态列默认）
 * - 信息类：mode="fill"，有浅色底，不展示圆点
 * - 轻量标签类：mode="soft"，浅色底 + 浅色边框 + 深色字，支持 icon（用于卡片顶部标签）
 * - 角色类：preset，例如 <StatusTag preset="role-admin" />
 * - 图标类：通过 icon 传入，颜色跟随当前 variant
 *
 * @deprecated mode="dot" / dot 属性 — 规范 §16.4 已废弃。
 *   组件内部静默 fallback 到 mode="text"，DEV 环境会输出 warning。
 *   新代码请使用 mode="text"（无底色彩色字）或 mode="fill"（浅底）。
 *
 * 颜色语义（statusTagColorTokens）：
 *   - blue   → 信息/进行中
 *   - green  → 成功/已完成
 *   - red    → 失败/错误/危险
 *   - orange → 警告（如即将到期、需要关注但非阻断）。
 *              色相对齐品牌警告色 #F59E0B (amber 系)：
 *              text amber-700、bg amber-50、border amber-200、dot #F59E0B。
 *   - gray   → 中性/默认/未启用
 */

const statusTagColorTokens = {
  /** Existing global semantic tokens */
  blue: {
    text: "text-[#1447E6]",
    bg: "bg-[#E8ECFE]",
    border: "border-[#C7D7FE]",
    dot: "bg-[#1447E6]",
  },
  green: {
    text: "text-[#008236]",
    bg: "bg-[#E9F8EB]",
    border: "border-[#BFE8C8]",
    dot: "bg-[#008236]",
  },
  red: {
    text: "text-[#DC2626]",
    bg: "bg-[#FEF2F2]",
    border: "border-[#FECACA]",
    dot: "bg-[#DC2626]",
  },
  orange: {
    // 0608：切换到品牌警告色 (#F59E0B amber 系)，与 dot 同色相统一
    // text amber-700 #B45309 / bg amber-50 #FFFBEB / border amber-200 #FDE68A / dot #F59E0B
    text: "text-amber-700",
    bg: "bg-amber-50",
    border: "border-amber-200",
    dot: "bg-[#F59E0B]",
  },
  gray: {
    text: "text-[#0A0A0A]",
    bg: "bg-[#F5F5F5]",
    border: "border-gray-200",
    dot: "bg-[#0A0A0A]",
  },

  /** shadcn / Tailwind palette extensions for soft classification tags */
  slate: {
    text: "text-slate-700",
    bg: "bg-slate-50",
    border: "border-slate-200",
    dot: "bg-slate-500",
  },
  zinc: {
    text: "text-zinc-700",
    bg: "bg-zinc-50",
    border: "border-zinc-200",
    dot: "bg-zinc-500",
  },
  stone: {
    text: "text-stone-700",
    bg: "bg-stone-50",
    border: "border-stone-200",
    dot: "bg-stone-500",
  },
  yellow: {
    text: "text-yellow-700",
    bg: "bg-yellow-50",
    border: "border-yellow-200",
    dot: "bg-yellow-500",
  },
  amber: {
    text: "text-amber-700",
    bg: "bg-amber-50",
    border: "border-amber-200",
    dot: "bg-amber-500",
  },
  lime: {
    text: "text-lime-700",
    bg: "bg-lime-50",
    border: "border-lime-200",
    dot: "bg-lime-500",
  },
  emerald: {
    text: "text-emerald-700",
    bg: "bg-emerald-50",
    border: "border-emerald-200",
    dot: "bg-emerald-500",
  },
  teal: {
    text: "text-teal-700",
    bg: "bg-teal-50",
    border: "border-teal-200",
    dot: "bg-teal-500",
  },
  cyan: {
    text: "text-cyan-700",
    bg: "bg-cyan-50",
    border: "border-cyan-200",
    dot: "bg-cyan-500",
  },
  sky: {
    text: "text-sky-700",
    bg: "bg-sky-50",
    border: "border-sky-200",
    dot: "bg-sky-500",
  },
  indigo: {
    text: "text-indigo-700",
    bg: "bg-indigo-50",
    border: "border-indigo-200",
    dot: "bg-indigo-500",
  },
  violet: {
    text: "text-violet-700",
    bg: "bg-violet-50",
    border: "border-violet-200",
    dot: "bg-violet-500",
  },
  purple: {
    text: "text-purple-700",
    bg: "bg-purple-50",
    border: "border-purple-200",
    dot: "bg-purple-500",
  },
  fuchsia: {
    text: "text-fuchsia-700",
    bg: "bg-fuchsia-50",
    border: "border-fuchsia-200",
    dot: "bg-fuchsia-500",
  },
  pink: {
    text: "text-pink-700",
    bg: "bg-pink-50",
    border: "border-pink-200",
    dot: "bg-pink-500",
  },
  rose: {
    text: "text-rose-700",
    bg: "bg-rose-50",
    border: "border-rose-200",
    dot: "bg-rose-500",
  },
} as const;

const roleTagClassName = "h-[22px] rounded-full border border-gray-200 bg-white px-2 text-gray-950";

export type StatusTagColor = keyof typeof statusTagColorTokens;
type StatusTagVariant = StatusTagColor | "role";
type StatusTagMode = "text" | "dot" | "fill" | "soft";
type StatusTagPreset = "role-admin" | "role-user";
type StatusTagIconComponent = (props: React.SVGProps<SVGSVGElement>) => React.ReactElement;

function AdminRoleIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" {...props}>
      <path d="M8.68164 5.41602C8.84741 5.4447 9.00505 5.51295 9.14062 5.61523C9.29667 5.73308 9.41381 5.89252 9.48535 6.07324L9.83496 6.7627L10.6211 6.88574H10.6201C10.758 6.89729 10.8929 6.93239 11.0156 6.99512L11.1572 7.08301L11.2842 7.19141C11.3624 7.26989 11.4282 7.36026 11.4785 7.45898L11.542 7.6123L11.5811 7.77441C11.6072 7.93843 11.5943 8.10703 11.543 8.2666C11.4831 8.45262 11.3707 8.61512 11.2236 8.74219L11.2246 8.74316L10.6611 9.30371L10.7822 10.0645C10.8326 10.2555 10.8328 10.4565 10.7744 10.6465C10.7074 10.864 10.5719 11.0545 10.3887 11.1895C10.2053 11.3243 9.98255 11.3972 9.75488 11.3965C9.55588 11.3957 9.36291 11.3366 9.19531 11.2314L8.49902 10.874L7.79883 11.2324C7.63161 11.336 7.44004 11.3953 7.24219 11.3955C7.01535 11.3956 6.794 11.323 6.61133 11.1885C6.42869 11.0539 6.29373 10.8642 6.22656 10.6475C6.16852 10.46 6.16683 10.2613 6.21484 10.0723L6.33691 9.30371L5.78906 8.75684C5.6389 8.63249 5.52263 8.47165 5.45898 8.28613C5.38553 8.07181 5.38234 7.83921 5.4502 7.62305C5.51806 7.40696 5.65325 7.21785 5.83594 7.08398L5.97949 6.99512C6.1034 6.9317 6.23965 6.89601 6.37891 6.88477L7.16211 6.7627L7.52344 6.05273C7.59931 5.87199 7.72159 5.71347 7.88184 5.59863L8.02637 5.51172C8.17675 5.43676 8.34406 5.39835 8.51367 5.40039L8.68164 5.41602ZM4 6.9375C4.31066 6.9375 4.5625 7.18934 4.5625 7.5C4.5625 7.81066 4.31066 8.0625 4 8.0625H3.5C3.11875 8.0625 2.75298 8.21381 2.4834 8.4834C2.21381 8.75298 2.0625 9.11875 2.0625 9.5V10.5C2.0625 10.8107 1.81066 11.0625 1.5 11.0625C1.18934 11.0625 0.9375 10.8107 0.9375 10.5V9.5C0.9375 8.82038 1.20791 8.16904 1.68848 7.68848C2.16904 7.20791 2.82038 6.9375 3.5 6.9375H4ZM8.15137 7.30078C8.07517 7.45066 7.96419 7.58091 7.82812 7.67969C7.69214 7.77823 7.53409 7.84221 7.36816 7.86816L6.60938 7.9873L7.1543 8.53027L7.2373 8.62402C7.28923 8.68963 7.33399 8.76099 7.36914 8.83691L7.41504 8.9541L7.44629 9.07617C7.47072 9.1991 7.47389 9.32577 7.4541 9.4502L7.45312 9.45117L7.33105 10.208L8.01465 9.85938L8.12988 9.80859C8.24759 9.76497 8.37283 9.74225 8.49902 9.74219C8.62519 9.74219 8.75046 9.76505 8.86816 9.80859L8.9834 9.85938L9.66699 10.209L9.54492 9.4502C9.5185 9.28423 9.53201 9.11391 9.58398 8.9541L9.62988 8.83691C9.6825 8.72309 9.75544 8.61926 9.84473 8.53027L10.3887 7.9873L9.62988 7.86914C9.46396 7.84313 9.30587 7.7783 9.16992 7.67969C9.03394 7.5809 8.92283 7.45062 8.84668 7.30078L8.49902 6.61523L8.15137 7.30078ZM5 0.9375C6.41523 0.9375 7.5625 2.08477 7.5625 3.5C7.5625 4.91523 6.41523 6.0625 5 6.0625C3.58477 6.0625 2.4375 4.91523 2.4375 3.5C2.4375 2.08477 3.58477 0.9375 5 0.9375ZM5 2.0625C4.20609 2.0625 3.5625 2.70609 3.5625 3.5C3.5625 4.29391 4.20609 4.9375 5 4.9375C5.79391 4.9375 6.4375 4.29391 6.4375 3.5C6.4375 2.70609 5.79391 2.0625 5 2.0625Z" fill="currentColor" />
    </svg>
  );
}

function UserRoleIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" {...props}>
      <path d="M7.5 6.9375C8.17962 6.9375 8.83096 7.20791 9.31152 7.68848C9.79209 8.16904 10.0625 8.82038 10.0625 9.5V10.5C10.0625 10.8107 9.81066 11.0625 9.5 11.0625C9.18934 11.0625 8.9375 10.8107 8.9375 10.5V9.5C8.9375 9.11875 8.78619 8.75298 8.5166 8.4834C8.24702 8.21381 7.88125 8.0625 7.5 8.0625H4.5C4.11875 8.0625 3.75298 8.21381 3.4834 8.4834C3.21381 8.75298 3.0625 9.11875 3.0625 9.5V10.5C3.0625 10.8107 2.81066 11.0625 2.5 11.0625C2.18934 11.0625 1.9375 10.8107 1.9375 10.5V9.5C1.9375 8.82038 2.20791 8.16904 2.68848 7.68848C3.16904 7.20791 3.82038 6.9375 4.5 6.9375H7.5ZM6 0.9375C7.41523 0.9375 8.5625 2.08477 8.5625 3.5C8.5625 4.91523 7.41523 6.0625 6 6.0625C4.58477 6.0625 3.4375 4.9375 3.4375 3.5C3.4375 2.08477 4.58477 0.9375 6 0.9375ZM6 2.0625C5.20609 2.0625 4.5625 2.70609 4.5625 3.5C4.5625 4.29391 5.20609 4.9375 6 4.9375C6.79391 4.9375 7.4375 4.29391 7.4375 3.5C7.4375 2.70609 6.79391 2.0625 6 2.0625Z" fill="currentColor" />
    </svg>
  );
}

const rolePresets: Record<StatusTagPreset, { label: string; icon: StatusTagIconComponent }> = {
  "role-admin": { label: "管理员", icon: AdminRoleIcon },
  "role-user": { label: "用户", icon: UserRoleIcon },
};

interface StatusTagProps extends React.ComponentProps<"span"> {
  variant?: StatusTagVariant;
  /**
   * 显示形态。注意 mode="dot" 已废弃（规范 §16.4），传入会 fallback 到 mode="text"。
   * 新代码请使用 "text" / "fill" / "soft"。
   */
  mode?: StatusTagMode;
  /** @deprecated 已废弃，等价于 mode="dot"，会 fallback 到 mode="text"。仅用于兼容旧调用，不要再使用。 */
  dot?: boolean;
  icon?: React.ReactNode;
  preset?: StatusTagPreset;
}

function StatusTag({
  variant,
  mode,
  preset,
  dot = false,
  icon,
  className,
  children,
  ...props
}: StatusTagProps) {
  const presetConfig = preset ? rolePresets[preset] : undefined;
  const resolvedVariant: StatusTagVariant = presetConfig ? "role" : (variant ?? "gray");
  const isRole = resolvedVariant === "role" || Boolean(presetConfig);
  // 规范 §16.4：mode="dot" 已废弃。组件内部静默 fallback 到 mode="text"，
  // 业务侧旧调用无需立即修改；DEV 环境会输出 console.warn 提示迁移。
  const requestedMode: StatusTagMode = mode ?? (dot ? "dot" : "fill");
  if (import.meta.env?.DEV && requestedMode === "dot") {
    // eslint-disable-next-line no-console
    console.warn(
      "[StatusTag] mode=\"dot\" / dot 属性已废弃，已 fallback 到 mode=\"text\"。请改用 mode=\"text\" 或 mode=\"fill\"。"
    );
  }
  const normalizedMode: StatusTagMode = requestedMode === "dot" ? "text" : requestedMode;
  const resolvedMode: StatusTagMode | "role" = isRole ? "role" : normalizedMode;
  const color = resolvedVariant === "role" ? statusTagColorTokens.gray : statusTagColorTokens[resolvedVariant];
  const PresetIcon = presetConfig?.icon;
  const resolvedChildren = children ?? presetConfig?.label;

  return (
    <span
      data-slot="status-tag"
      data-variant={resolvedVariant}
      data-mode={resolvedMode}
      data-preset={preset}
      className={cn(
        "inline-flex items-center justify-center gap-1 whitespace-nowrap",
        resolvedMode === "text" && "px-0 py-0 bg-transparent text-sm font-medium leading-[1.5]",
        resolvedMode === "fill" && cn("h-5 rounded-full px-2 py-[2px]", color.bg),
        resolvedMode === "soft" && cn("h-5 rounded-[4px] border px-2 py-0", color.bg, color.border),
        resolvedMode === "role" && roleTagClassName,
        color.text,
        className
      )}
      {...props}
    >
      {PresetIcon && (
        <span data-slot="status-tag-icon" className="inline-flex size-3 shrink-0 items-center justify-center text-current">
          <PresetIcon className="size-3" />
        </span>
      )}
      {!PresetIcon && icon && (
        <span data-slot="status-tag-icon" className="inline-flex size-3 shrink-0 items-center justify-center text-current [&_svg]:size-3 [&_svg]:shrink-0">
          {icon}
        </span>
      )}
      {resolvedChildren && (
        resolvedMode === "text" ? (
          <span>{resolvedChildren}</span>
        ) : (
          <SmallBodyText as="span" tone="inherit">
            {resolvedChildren}
          </SmallBodyText>
        )
      )}
    </span>
  );
}

export { StatusTag, statusTagColorTokens, rolePresets as statusTagRolePresets };
