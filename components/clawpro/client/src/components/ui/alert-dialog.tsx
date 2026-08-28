import * as React from "react";
import * as AlertDialogPrimitive from "@radix-ui/react-alert-dialog";
import { XIcon } from "lucide-react";
import type { VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";
import { buttonVariants } from "@/components/ui/button";

function AlertDialog({
  ...props
}: React.ComponentProps<typeof AlertDialogPrimitive.Root>) {
  return <AlertDialogPrimitive.Root data-slot="alert-dialog" {...props} />;
}

function AlertDialogTrigger({
  ...props
}: React.ComponentProps<typeof AlertDialogPrimitive.Trigger>) {
  return (
    <AlertDialogPrimitive.Trigger data-slot="alert-dialog-trigger" {...props} />
  );
}

function AlertDialogPortal({
  ...props
}: React.ComponentProps<typeof AlertDialogPrimitive.Portal>) {
  return (
    <AlertDialogPrimitive.Portal data-slot="alert-dialog-portal" {...props} />
  );
}

function AlertDialogOverlay({
  className,
  ...props
}: React.ComponentProps<typeof AlertDialogPrimitive.Overlay>) {
  return (
    <AlertDialogPrimitive.Overlay
      data-slot="alert-dialog-overlay"
      className={cn(
        "data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 fixed inset-0 z-50 bg-black/45",
        className
      )}
      {...props}
    />
  );
}

function AlertDialogContent({
  className,
  children,
  showCloseButton = true,
  ...props
}: React.ComponentProps<typeof AlertDialogPrimitive.Content> & {
  showCloseButton?: boolean;
}) {
  return (
    <AlertDialogPortal>
      <AlertDialogOverlay />
      <AlertDialogPrimitive.Content
        data-slot="alert-dialog-content"
        className={cn(
          "bg-[var(--popover)] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 fixed top-[50%] left-[50%] z-50 grid w-full max-w-[calc(100%-2rem)] translate-x-[-50%] translate-y-[-50%] rounded-[12px] px-6 py-0 shadow-[var(--shadow-overlay)] duration-200 overflow-clip sm:max-w-lg",
          className
        )}
        {...props}
      >
        {children}
        {showCloseButton && (
          <AlertDialogPrimitive.Cancel
            data-slot="alert-dialog-close"
            className="absolute top-5 right-5 flex items-center justify-center size-5 rounded-sm text-[var(--text-muted)] transition-colors hover:text-[var(--text-title)] focus:outline-none outline-none ring-0 focus:ring-0 disabled:pointer-events-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-5"
          >
            <XIcon />
            <span className="sr-only">关闭</span>
          </AlertDialogPrimitive.Cancel>
        )}
      </AlertDialogPrimitive.Content>
    </AlertDialogPortal>
  );
}

/**
 * AlertDialogHeader - 警示弹窗标题区
 *
 * ⚠️ 规范：AlertDialogHeader 内仅放 AlertDialogTitle。
 * AlertDialogDescription（正文/说明）必须放在 AlertDialogHeader **外部**，
 * 作为 AlertDialogContent 的直接子元素，与 Header/Footer 平级。
 * 禁止将所有内容塞进 Header 导致 Footer 与 Header 间出现大面积空白。
 *
 * 正确结构：
 *   <AlertDialogContent>
 *     <AlertDialogHeader>
 *       <AlertDialogTitle>标题</AlertDialogTitle>
 *     </AlertDialogHeader>
 *     <AlertDialogDescription>描述正文</AlertDialogDescription>
 *     <AlertDialogFooter>...</AlertDialogFooter>
 *   </AlertDialogContent>
 */
function AlertDialogHeader({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="alert-dialog-header"
      className={cn("flex flex-col justify-center gap-0.5 -mx-6 px-6 pt-4 pb-2 shrink-0", className)}
      {...props}
    />
  );
}

function AlertDialogFooter({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="alert-dialog-footer"
      className={cn(
        "flex gap-2 justify-end -mx-6 px-6 pt-4 pb-4 mt-2",
        className
      )}
      {...props}
    />
  );
}

function AlertDialogTitle({
  className,
  ...props
}: React.ComponentProps<typeof AlertDialogPrimitive.Title>) {
  return (
    <AlertDialogPrimitive.Title
      data-slot="alert-dialog-title"
      className={cn("text-lg font-semibold", className)}
      {...props}
    />
  );
}

function AlertDialogDescription({
  className,
  ...props
}: React.ComponentProps<typeof AlertDialogPrimitive.Description>) {
  return (
    <AlertDialogPrimitive.Description
      data-slot="alert-dialog-description"
      className={cn("text-[var(--text-secondary)] text-sm", className)}
      {...props}
    />
  );
}

/**
 * AlertDialogAction - 警示弹窗主按钮
 *
 * 默认 variant 为 "destructive"（红底白字），符合警示弹窗规范：
 * - 删除/解绑/重置/禁用等危险操作 → 默认即红色，无需手动覆盖
 *
 * 注意：禁止通过 className 用 bg-* 覆盖颜色 —— Button default 使用
 * background 简写（含渐变），会覆盖单纯的 background-color。
 * 如需切换样式，请使用 variant prop。
 */
function AlertDialogAction({
  className,
  variant = "destructive",
  ...props
}: React.ComponentProps<typeof AlertDialogPrimitive.Action> &
  Pick<VariantProps<typeof buttonVariants>, "variant">) {
  return (
    <AlertDialogPrimitive.Action
      className={cn(buttonVariants({ variant }), className)}
      {...props}
    />
  );
}

function AlertDialogCancel({
  className,
  ...props
}: React.ComponentProps<typeof AlertDialogPrimitive.Cancel>) {
  return (
    <AlertDialogPrimitive.Cancel
      className={cn(buttonVariants({ variant: "outline" }), className)}
      {...props}
    />
  );
}

export {
  AlertDialog,
  AlertDialogPortal,
  AlertDialogOverlay,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogFooter,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogAction,
  AlertDialogCancel,
};
