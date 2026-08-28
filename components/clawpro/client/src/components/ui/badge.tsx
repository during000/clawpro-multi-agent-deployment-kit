import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

/**
 * Badge 组件（对齐 shadcn Radix UI 规范）
 * https://ui.shadcn.com/docs/components/radix/badge
 *
 * Variants:
 * - default：黑底白字（Radix neutral 主色）
 * - secondary：浅灰底深字
 * - destructive：浅红底红字
 * - outline：白底描边
 *
 * Custom Colors（沿用 shadcn 官方 Custom Colors 示例的轻量胶囊样式）：
 * - blue / green / purple / red
 * 业务侧通过 `color` 属性使用，例如：
 *   <Badge color="blue">Blue</Badge>
 */

const badgeVariants = cva(
  "inline-flex items-center justify-center rounded-full border px-2.5 py-0.5 text-xs font-normal w-fit whitespace-nowrap shrink-0 gap-1 [&>svg]:size-3 [&>svg]:pointer-events-none focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none aria-invalid:border-destructive transition-[color,box-shadow] overflow-hidden",
  {
    variants: {
      variant: {
        default:
          "border-transparent bg-[#0A0A0A] text-white [a&]:hover:bg-[#0A0A0A]/90",
        secondary:
          "border-transparent bg-[#F5F5F5] text-gray-950 [a&]:hover:bg-[#EDEDED]",
        destructive:
          "border-transparent bg-red-100/60 text-red-600 [a&]:hover:bg-red-100 dark:bg-red-950/40 dark:text-red-300",
        outline:
          "border-gray-200 bg-white text-gray-950 [a&]:hover:bg-[#F5F5F5]",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

const badgeColorVariants = {
  blue: "border-transparent bg-[#E8ECFE] text-[#1447E6] dark:bg-blue-950/40 dark:text-blue-300",
  green:
    "border-transparent bg-green-50 text-green-700 dark:bg-green-950/40 dark:text-green-300",
  purple:
    "border-transparent bg-purple-50 text-purple-700 dark:bg-purple-950/40 dark:text-purple-300",
  red: "border-transparent bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300",
} as const;

export type BadgeColor = keyof typeof badgeColorVariants;

interface BadgeProps
  extends React.ComponentProps<"span">,
    VariantProps<typeof badgeVariants> {
  asChild?: boolean;
  /**
   * Custom Colors（对应 shadcn 官方 Custom Colors 示例）。
   * 设置后会覆盖 `variant` 的视觉样式，只保留尺寸/字号。
   */
  color?: BadgeColor;
}

function Badge({
  className,
  variant,
  color,
  asChild = false,
  ...props
}: BadgeProps) {
  const Comp = asChild ? Slot : "span";

  return (
    <Comp
      data-slot="badge"
      data-color={color}
      className={cn(
        badgeVariants({ variant: color ? undefined : variant }),
        color && badgeColorVariants[color],
        className,
      )}
      {...props}
    />
  );
}

export { Badge, badgeVariants, badgeColorVariants };
