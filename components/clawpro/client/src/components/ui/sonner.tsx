/**
 * Toast 通知组件
 * ─────────────────────────────────────────────────────────────────
 * 基于 sonner 的全局 Toast 通知，统一样式与交互规范。
 *
 * 视觉规范：
 *   - 白色背景 + 管控端 4px 圆角 + 阴影
 *   - 图标在左侧，文字左对齐，关闭按钮在左上角
 *   - 颜色 / 圆角 / 边框走 ClawPro token
 *
 * 使用方式：
 *   import { toast } from 'sonner';
 *   toast.error("请输入用户 ID");
 *   toast.success("操作成功");
 *
 * 在 App 根组件挂载 <Toaster /> 即可。
 */
import type { CSSProperties } from "react";
import { Toaster as Sonner, toast, type ToasterProps } from "sonner";
import { X } from "lucide-react";

/**
 * 封装 toast 方法，自动加上左上角关闭按钮
 */
function withClose(id: string | number) {
  return (
    <button
      className="absolute -left-2 -top-2 flex h-5 w-5 items-center justify-center rounded-full border border-[var(--cp-border)] bg-[var(--cp-surface)] text-[var(--text-weak)] shadow-sm transition-colors hover:bg-[var(--bg-grey-hover)] hover:text-[var(--text-title)]"
      onClick={() => toast.dismiss(id)}
      aria-label="关闭"
    >
      <X className="h-3.5 w-3.5" />
    </button>
  );
}

// 重新导出 toast，让业务侧可以直接用
export { toast, withClose };

const Toaster = ({ ...props }: ToasterProps) => {
  return (
    <Sonner
      theme="light"
      className="toaster group"
      position="top-center"
      duration={4000}
      closeButton
      toastOptions={{
        duration: 4000,
        classNames: {
          toast:
            "group toast group-[.toaster]:relative group-[.toaster]:overflow-visible group-[.toaster]:rounded-[12px] group-[.toaster]:border-[var(--cp-border)] group-[.toaster]:bg-[var(--cp-surface)] group-[.toaster]:px-4 group-[.toaster]:py-3 group-[.toaster]:text-left group-[.toaster]:text-sm group-[.toaster]:font-medium group-[.toaster]:text-[var(--text-title)] group-[.toaster]:shadow-lg",
          title:
            "group-[.toast]:flex-1 group-[.toast]:text-left group-[.toast]:font-medium group-[.toast]:text-[var(--text-title)]",
          description:
            "group-[.toast]:text-left group-[.toast]:text-[var(--text-secondary)]",
          actionButton:
            "group-[.toast]:rounded-[12px] group-[.toast]:bg-[var(--cp-brand-black)] group-[.toast]:text-xs group-[.toast]:font-medium group-[.toast]:text-white",
          cancelButton:
            "group-[.toast]:rounded-[12px] group-[.toast]:border group-[.toast]:border-[var(--cp-border)] group-[.toast]:bg-[var(--cp-surface)] group-[.toast]:text-xs group-[.toast]:text-[var(--text-secondary)]",
          closeButton:
            "!left-0 !right-auto !top-0 !h-5 !w-5 !-translate-x-1/2 !-translate-y-1/2 !rounded-full !border-[var(--cp-border)] !bg-[var(--cp-surface)] !text-[var(--text-weak)] !shadow-sm hover:!bg-[var(--bg-grey-hover)] hover:!text-[var(--text-title)]",
        },
      }}
      style={
        {
          "--normal-bg": "var(--cp-surface)",
          "--normal-text": "var(--text-title)",
          "--normal-border": "var(--cp-border)",
          "--error-bg": "var(--cp-surface)",
          "--error-text": "var(--text-title)",
          "--error-border": "var(--cp-border)",
          "--success-bg": "var(--cp-surface)",
          "--success-text": "var(--text-title)",
          "--success-border": "var(--cp-border)",
          zIndex: 99999,
        } as CSSProperties
      }
      {...props}
    />
  );
};

export { Toaster };
