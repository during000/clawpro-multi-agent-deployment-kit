/**
 * Toast 通知组件
 * ─────────────────────────────────────────────────────────────────
 * 基于 sonner 的全局 Toast 通知，统一样式与交互规范。
 *
 * 视觉规范：
 *   - 白色背景 + 圆角 12px + 阴影
 *   - 图标在左侧，文字左对齐，关闭按钮在右上角
 *   - 边框颜色统一 #EAEEF4
 *
 * 使用方式：
 *   import { toast } from 'sonner';
 *   toast.error("请输入用户 ID");
 *   toast.success("操作成功");
 *
 * 在 App 根组件挂载 <Toaster /> 即可。
 */
import { Toaster as Sonner, toast, type ToasterProps } from "sonner";
import { X } from "lucide-react";

/**
 * 封装 toast 方法，自动加上右上角关闭按钮
 */
function withClose(id: string | number) {
  return (
    <button
      className="absolute -right-2 -top-2 flex h-5 w-5 items-center justify-center rounded-full bg-white border border-[#EAEEF4] shadow-sm text-[#7b818f] hover:bg-[#f4f4f5] hover:text-[#09090b] transition-colors"
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
      toastOptions={{
        classNames: {
          toast:
            "group toast group-[.toaster]:relative group-[.toaster]:overflow-visible group-[.toaster]:bg-white group-[.toaster]:text-gray-950 group-[.toaster]:border-[#EAEEF4] group-[.toaster]:shadow-lg group-[.toaster]:rounded-xl group-[.toaster]:px-4 group-[.toaster]:py-3 group-[.toaster]:text-sm group-[.toaster]:font-medium group-[.toaster]:text-left",
          title: "group-[.toast]:text-gray-950 group-[.toast]:font-medium group-[.toast]:text-left group-[.toast]:flex-1",
          description: "group-[.toast]:text-gray-500 group-[.toast]:text-left",
          actionButton:
            "group-[.toast]:bg-gray-950 group-[.toast]:text-white group-[.toast]:rounded-md group-[.toast]:text-xs group-[.toast]:font-medium",
          cancelButton:
            "group-[.toast]:bg-white group-[.toast]:text-gray-600 group-[.toast]:border group-[.toast]:border-[#EAEEF4] group-[.toast]:rounded-md group-[.toast]:text-xs",
        },
      }}
      style={
        {
          "--normal-bg": "#ffffff",
          "--normal-text": "#0a0a0a",
          "--normal-border": "#EAEEF4",
          "--error-bg": "#ffffff",
          "--error-text": "#0a0a0a",
          "--error-border": "#EAEEF4",
          "--success-bg": "#ffffff",
          "--success-text": "#0a0a0a",
          "--success-border": "#EAEEF4",
          zIndex: 99999,
        } as React.CSSProperties
      }
      {...props}
    />
  );
};

export { Toaster };
