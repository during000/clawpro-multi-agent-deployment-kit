import { cn } from "@/lib/utils";
import * as DialogPrimitive from "@radix-ui/react-dialog";
import { XIcon } from "lucide-react";
import * as React from "react";

// Context to track composition state across dialog children
const DialogCompositionContext = React.createContext<{
  isComposing: () => boolean;
  setComposing: (composing: boolean) => void;
  justEndedComposing: () => boolean;
  markCompositionEnd: () => void;
}>({
  isComposing: () => false,
  setComposing: () => {},
  justEndedComposing: () => false,
  markCompositionEnd: () => {},
});

export const useDialogComposition = () =>
  React.useContext(DialogCompositionContext);

function Dialog({
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Root>) {
  const composingRef = React.useRef(false);
  const justEndedRef = React.useRef(false);
  const endTimerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);

  const contextValue = React.useMemo(
    () => ({
      isComposing: () => composingRef.current,
      setComposing: (composing: boolean) => {
        composingRef.current = composing;
      },
      justEndedComposing: () => justEndedRef.current,
      markCompositionEnd: () => {
        justEndedRef.current = true;
        if (endTimerRef.current) {
          clearTimeout(endTimerRef.current);
        }
        endTimerRef.current = setTimeout(() => {
          justEndedRef.current = false;
        }, 150);
      },
    }),
    []
  );

  return (
    <DialogCompositionContext.Provider value={contextValue}>
      <DialogPrimitive.Root data-slot="dialog" {...props} />
    </DialogCompositionContext.Provider>
  );
}

function DialogTrigger({
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Trigger>) {
  return <DialogPrimitive.Trigger data-slot="dialog-trigger" {...props} />;
}

function DialogPortal({
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Portal>) {
  return <DialogPrimitive.Portal data-slot="dialog-portal" {...props} />;
}

function DialogClose({
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Close>) {
  return <DialogPrimitive.Close data-slot="dialog-close" {...props} />;
}

function DialogOverlay({
  className,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Overlay>) {
  return (
    <DialogPrimitive.Overlay
      data-slot="dialog-overlay"
      className={cn(
        "data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 fixed inset-0 z-50 bg-black/45",
        className
      )}
      {...props}
    />
  );
}

DialogOverlay.displayName = "DialogOverlay";

/**
 * 弹窗尺寸规范（仅允许以下 4 档，禁止自定义其他宽度）：
 * - sm (420px): 简单确认、单字段输入、警示弹窗
 * - md (560px): 表单弹窗（3-6个字段）、发布/编辑（默认）
 * - lg (720px): 复杂表单、含表格/列表、多列内容、详情面板
 * - xl (920px): 多列数据表格批量操作、Tabs + 列表管理、命令下发等多阶段弹窗
 *
 * ⚠️ 使用范围：
 * - rounded-[12px] 圆角弹窗适用于全站（管控端 + 用户端）
 * - 所有浮层弹窗组件统一 12px 圆角（Dialog/AlertDialog/Drawer/Popover/HoverCard/DropdownMenu/Toast/Select下拉面板）
 */
const dialogSizeMap = {
  sm: "sm:max-w-[420px]",
  md: "sm:max-w-[560px]",
  lg: "sm:max-w-[720px]",
  xl: "sm:max-w-[920px]",
} as const;

type DialogSize = keyof typeof dialogSizeMap;

function DialogContent({
  className,
  children,
  showCloseButton = true,
  size,
  onEscapeKeyDown,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Content> & {
  showCloseButton?: boolean;
  /** 弹窗宽度档位：sm(420px) | md(560px) | lg(720px) | xl(920px) */
  size?: DialogSize;
}) {
  const { isComposing } = useDialogComposition();

  const handleEscapeKeyDown = React.useCallback(
    (e: KeyboardEvent) => {
      // Check both the native isComposing property and our context state
      // This handles Safari's timing issues with composition events
      const isCurrentlyComposing = (e as any).isComposing || isComposing();

      // If IME is composing, prevent dialog from closing
      if (isCurrentlyComposing) {
        e.preventDefault();
        return;
      }

      // Call user's onEscapeKeyDown if provided
      onEscapeKeyDown?.(e);
    },
    [isComposing, onEscapeKeyDown]
  );

  return (
    <DialogPortal data-slot="dialog-portal">
      <DialogOverlay />
      <DialogPrimitive.Content
        data-slot="dialog-content"
        className={cn(
          // 移除弹窗容器自身的焦点高亮：Radix 关闭内部 Popover 后会把焦点还给 Content 容器，
          // 浏览器对其应用 :focus-visible 蓝色 outline，造成整个弹窗出现蓝圈。内容内的可聚焦元素仍保留各自焦点样式。
          "outline-none focus:outline-none focus-visible:outline-none ring-0 focus:ring-0 focus-visible:ring-0",
          // 高度约束（强制规范）：弹窗最高不超过视口高度 - 64px，超出时由 DialogBody 内部滚动。
          // 必须用 flex-col（而非 grid）布局，否则 DialogBody 的 flex-1 / min-h-0 / overflow-y-auto 不生效，
          // 长文本（如角色 soul 等管控端可配文案）会撑爆容器并被视口截断。
          "max-h-[calc(100dvh-4rem)] flex flex-col",
          "bg-white data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 fixed top-[50%] left-[50%] z-50 w-full max-w-[calc(100%-2rem)] translate-x-[-50%] translate-y-[-50%] rounded-[12px] shadow-[0_6px_16px_0_rgba(0,0,0,0.08),0_3px_6px_-4px_rgba(0,0,0,0.12),0_9px_28px_8px_rgba(0,0,0,0.05)] duration-200 overflow-clip px-6 py-0",
          size ? dialogSizeMap[size] : "sm:max-w-[560px]",
          className
        )}
        onEscapeKeyDown={handleEscapeKeyDown}
        {...props}
      >
        {children}
        {showCloseButton && (
          <DialogPrimitive.Close
            data-slot="dialog-close"
            className="absolute top-[26px] right-5 flex items-center justify-center size-5 rounded-sm text-[#7b818f] transition-colors hover:text-gray-950 focus:outline-none outline-none ring-0 focus:ring-0 disabled:pointer-events-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-5"
          >
            <XIcon />
            <span className="sr-only">Close</span>
          </DialogPrimitive.Close>
        )}
      </DialogPrimitive.Content>
    </DialogPortal>
  );
}

function DialogHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="dialog-header"
      className={cn("flex flex-col justify-center gap-0.5 -mx-6 px-6 pt-6 pb-3 shrink-0", className)}
      {...props}
    />
  );
}

function DialogFooter({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="dialog-footer"
      className={cn(
        "flex items-center gap-2 justify-end -mx-6 px-6 pt-6 pb-6 mt-0 shrink-0",
        className
      )}
      {...props}
    />
  );
}

/**
 * DialogBody - 弹窗内容滚动区
 *
 * 设计规则：
 *   - 通栏布局：上下内边距 0、横向 padding 0，仅用 -mx-6 抵消 DialogContent 的横向 px-6
 *   - 内容铺满到弹窗边缘，需要左右留白由使用处自行加 px-6
 *   - 纵向滚动滑块不挤压内容区左侧对齐，内容起点必须与 Header 标题左边缘一致
 *   - 滚动条默认不可见，hover/滚动时才显示
 *   - 滚动条宽 6px，圆角，灰色（#D4D4D4）
 *   - scrollbar-gutter: stable 防止出现/消失时内容区跳动；禁止使用 both-edges，避免左侧额外 gutter 造成内容与标题不对齐
 *
 * 使用示例（通栏 + 自加 px-6 内距）：
 *   <DialogBody className="px-6">
 *     <div className="space-y-4">...</div>
 *   </DialogBody>
 *
 * 不需要左右内距（如全宽 Tabs / 表格贴边）时直接 <DialogBody>，留白由内层处理。
 */
function DialogBody({ className, style, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="dialog-body"
      className={cn(
        "flex-1 min-h-0 overflow-y-auto -mx-6 py-0",
        "[&::-webkit-scrollbar]:w-[6px]",
        "[&::-webkit-scrollbar-thumb]:rounded-full",
        "[&::-webkit-scrollbar-thumb]:bg-transparent",
        "[&::-webkit-scrollbar-track]:bg-transparent",
        "hover:[&::-webkit-scrollbar-thumb]:bg-gray-300",
        "[&:active::-webkit-scrollbar-thumb]:bg-gray-300",
        className
      )}
      style={{ scrollbarGutter: "stable", ...style }}
      {...props}
    />
  );
}

function DialogTitle({
  className,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Title>) {
  return (
    <DialogPrimitive.Title
      data-slot="dialog-title"
      className={cn("text-[16px] leading-6 font-semibold text-gray-900", className)}
      {...props}
    />
  );
}

function DialogDescription({
  className,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Description>) {
  return (
    <DialogPrimitive.Description
      data-slot="dialog-description"
      className={cn("text-sm text-[var(--text-secondary)]", className)}
      {...props}
    />
  );
}

export {
  Dialog,
  DialogBody,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogOverlay,
  DialogPortal,
  DialogTitle,
  DialogTrigger
};
