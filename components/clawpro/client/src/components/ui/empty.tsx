import { cva, type VariantProps } from "class-variance-authority";

import { MetaText, CardTitle } from "@/components/ui/Typography";
import { cn } from "@/lib/utils";

function Empty({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="empty"
      className={cn(
        "flex min-w-0 flex-1 flex-col items-center justify-center gap-6 rounded-[4px] border-dashed p-6 text-center text-balance md:p-12",
        className
      )}
      {...props}
    />
  );
}

function EmptyHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="empty-header"
      className={cn(
        "flex max-w-sm flex-col items-center gap-2 text-center",
        className
      )}
      {...props}
    />
  );
}

/**
 * EmptyMedia 视觉容器
 *
 * - default（推荐）：100x80 透明容器，自动渲染统一空态插画。
 *   业务侧使用 `<EmptyMedia />` 即可，禁止传 children/src。
 * - hint：提示类插画，用于功能关闭、未开通、需管理员处理等引导场景。
 * - empty-data：default 的别名，兼容旧调用，行为完全一致。
 *
 * 详见 SKILL-GLOBAL-COMPONENTS.md §24。
 */
const emptyMediaVariants = cva(
  // 插画尺寸硬约束：max-w/max-h 锁死 100x80，防止外层放大；select-none 防误选
  // 基础类已包含尺寸约束 → 即使 variant 传错也不会撑大
  "flex shrink-0 items-center justify-center mb-2 select-none w-[100px] h-20 max-w-[100px] max-h-20 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_img]:w-full [&_img]:h-full [&_img]:max-w-full [&_img]:max-h-full [&_img]:object-contain [&_img]:select-none",
  {
    variants: {
      variant: {
        default: "bg-transparent",
        hint: "!bg-transparent [&_img]:!bg-transparent [&_img]:mix-blend-multiply",
        // 兼容旧调用，与 default 行为一致
        "empty-data": "bg-transparent",
        "empty-logs": "bg-transparent",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

// 统一插画资源（由设计统一维护）
type EmptyMediaVariant = NonNullable<VariantProps<typeof emptyMediaVariants>["variant"]>;

const EMPTY_MEDIA_ASSETS: Record<EmptyMediaVariant, { src: string; alt: string }> = {
  default: { src: "/assets/admin-sidebar/empty-aiagent.png", alt: "暂无数据" },
  hint: { src: "/assets/admin-sidebar/empty-aiagent-hint.png", alt: "提示" },
  "empty-data": { src: "/assets/admin-sidebar/empty-aiagent.png", alt: "暂无数据" },
  "empty-logs": { src: "/assets/admin-sidebar/empty-aiagent.png", alt: "暂无日志" },
};

function EmptyMedia({
  className,
  variant = "default",
  children,
  ...props
}: React.ComponentProps<"div"> & VariantProps<typeof emptyMediaVariants>) {
  const asset = EMPTY_MEDIA_ASSETS[variant ?? "default"] ?? EMPTY_MEDIA_ASSETS.default;
  const content =
    children ?? (
      <img
        src={asset.src}
        alt={asset.alt}
        draggable={false}
      />
    );

  return (
    <div
      data-slot="empty-icon"
      data-variant={variant}
      className={cn(emptyMediaVariants({ variant, className }))}
      {...props}
    >
      {content}
    </div>
  );
}

function EmptyTitle({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <CardTitle
      as="div"
      data-slot="empty-title"
      className={cn("tracking-tight", className)}
      {...props}
    />
  );
}

function EmptyDescription({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <MetaText
      as="div"
      tone="weak"
      data-slot="empty-description"
      className={cn(
        "[&>a]:text-[var(--text-brand)] [&>a:hover]:text-[var(--text-brand)] [&>a]:underline [&>a]:underline-offset-4",
        className
      )}
      {...props}
    />
  );
}

function EmptyContent({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="empty-content"
      className={cn(
        "flex w-full max-w-sm min-w-0 flex-col items-center justify-center gap-4 text-sm text-balance",
        className
      )}
      {...props}
    />
  );
}

export {
  Empty,
  EmptyHeader,
  EmptyTitle,
  EmptyDescription,
  EmptyContent,
  EmptyMedia,
};
