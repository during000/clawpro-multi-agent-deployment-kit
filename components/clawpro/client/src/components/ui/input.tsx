import { useDialogComposition } from "@/components/ui/dialog";
import { useComposition } from "@/hooks/useComposition";
import { cn } from "@/lib/utils";
import * as React from "react";

function Input({
  className,
  type,
  onKeyDown,
  onCompositionStart,
  onCompositionEnd,
  tenant = false,
  ...props
}: React.ComponentProps<"input"> & {
  /**
   * 用户端形态：圆角变为 rounded-xl（12px），与用户端卡片圆角统一。
   * 仅 pages/tenant/** 业务页使用；管理端保持 rounded-[4px]。
   * 规范来源：SKILL-TENANT.md（2026-05-25 控件圆角对齐）
   */
  tenant?: boolean;
}) {
  // Get dialog composition context if available (will be no-op if not inside Dialog)
  const dialogComposition = useDialogComposition();

  // Add composition event handlers to support input method editor (IME) for CJK languages.
  const {
    onCompositionStart: handleCompositionStart,
    onCompositionEnd: handleCompositionEnd,
    onKeyDown: handleKeyDown,
  } = useComposition<HTMLInputElement>({
    onKeyDown: (e) => {
      // Check if this is an Enter key that should be blocked
      const isComposing = (e.nativeEvent as any).isComposing || dialogComposition.justEndedComposing();

      // If Enter key is pressed while composing or just after composition ended,
      // don't call the user's onKeyDown (this blocks the business logic)
      if (e.key === "Enter" && isComposing) {
        return;
      }

      // Otherwise, call the user's onKeyDown
      onKeyDown?.(e);
    },
    onCompositionStart: e => {
      dialogComposition.setComposing(true);
      onCompositionStart?.(e);
    },
    onCompositionEnd: e => {
      // Mark that composition just ended - this helps handle the Enter key that confirms input
      dialogComposition.markCompositionEnd();
      // Delay setting composing to false to handle Safari's event order
      // In Safari, compositionEnd fires before the ESC keydown event
      setTimeout(() => {
        dialogComposition.setComposing(false);
      }, 100);
      onCompositionEnd?.(e);
    },
  });

  return (
    <input
      type={type}
      data-slot="input"
      data-tenant={tenant ? "true" : undefined}
      className={cn(
        // 描边统一走全局 token --border (#EAEEF4)，与容器/Tag 等所有控件描边语言对齐
        // （0605 修订：原硬编码 #D3D6DB 比 --border 深一档，导致 Input 与同屏 Tag/Card 描边出现"深浅撕裂"）
        "h-9 w-full min-w-0 border border-border bg-white px-3 py-[5px] text-sm text-[var(--text-title)] font-normal transition-colors outline-none",
        tenant ? "rounded-full" : "rounded-[4px]",
        "placeholder:text-[var(--text-weak)]",
        "hover:border-[#355EF1]",
        "focus:border-[#355EF1]",
        "disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-[#f3f3f4] disabled:border-border disabled:text-gray-400 disabled:hover:border-border",
        "aria-invalid:border-destructive",
        "selection:bg-blue-500/10 selection:text-gray-950",
        "file:text-gray-950 file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium",
        "[&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-calendar-picker-indicator]:hidden [&::-webkit-clear-button]:hidden [&::-ms-reveal]:hidden",
        className
      )}
      onCompositionStart={handleCompositionStart}
      onCompositionEnd={handleCompositionEnd}
      onKeyDown={handleKeyDown}
      {...props}
    />
  );
}

export { Input };
