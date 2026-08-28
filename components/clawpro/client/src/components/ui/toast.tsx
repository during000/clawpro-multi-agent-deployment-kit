/**
 * Portable Toast Component
 * ─────────────────────────────────────────────────────────────────
 * 独立的 Toast 通知组件，不依赖 sonner。
 * 用于宿主仓不支持 sonner 或需要完全独立实现的场景。
 *
 * 视觉规范：
 *   - 背景：白色 (#FFFFFF)
 *   - 边框：#EAEEF4 (1px)
 *   - 圆角：12px (rounded-xl)
 *   - 内边距：12px 16px
 *   - 字号：14px / font-medium
 *   - 文字对齐：左对齐
 *   - 阴影：shadow-lg
 *   - 定位：页面顶部居中 (top-center)
 *   - z-index：99999
 *   - 自动消失：4000ms (4秒)
 *   - 关闭按钮：右上角外侧，20×20 圆形白底
 *
 * 类型：
 *   - success: 绿色勾
 *   - error: 黑色感叹号
 *   - info: 蓝色 i
 *   - warning: 橙色感叹号
 *
 * 使用方式：
 *   import { PortableToast } from "@/components/ui/toast";
 *
 *   export function MyComponent() {
 *     const [show, setShow] = useState(false);
 *
 *     return (
 *       <>
 *         {show && (
 *           <PortableToast
 *             type="success"
 *             message="操作成功"
 *             onClose={() => setShow(false)}
 *           />
 *         )}
 *       </>
 *     );
 *   }
 */

import React, { useEffect, useState } from "react";

type ToastType = "success" | "error" | "info" | "warning";

interface PortableToastProps {
  type: ToastType;
  message: string;
  duration?: number; // 毫秒，默认 4000
  onClose: () => void;
}

/**
 * 获取对应类型的图标
 */
function getIcon(type: ToastType) {
  switch (type) {
    case "success":
      return (
        <svg
          className="h-5 w-5 text-emerald-500"
          viewBox="0 0 20 20"
          fill="currentColor"
        >
          <path
            fillRule="evenodd"
            d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
            clipRule="evenodd"
          />
        </svg>
      );
    case "error":
      return (
        <svg
          className="h-5 w-5 text-gray-900"
          viewBox="0 0 20 20"
          fill="currentColor"
        >
          <path
            fillRule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
            clipRule="evenodd"
          />
        </svg>
      );
    case "warning":
      return (
        <svg
          className="h-5 w-5 text-amber-500"
          viewBox="0 0 20 20"
          fill="currentColor"
        >
          <path
            fillRule="evenodd"
            d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
            clipRule="evenodd"
          />
        </svg>
      );
    case "info":
    default:
      return (
        <svg
          className="h-5 w-5 text-blue-500"
          viewBox="0 0 20 20"
          fill="currentColor"
        >
          <path
            fillRule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
            clipRule="evenodd"
          />
        </svg>
      );
  }
}

/**
 * PortableToast 组件
 * 独立的 Toast，不依赖 sonner，自动消失。
 */
export function PortableToast({
  type,
  message,
  duration = 4000,
  onClose,
}: PortableToastProps) {
  const [isExiting, setIsExiting] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => {
      setIsExiting(true);
      const exitTimer = setTimeout(onClose, 200); // 配合淡出动画
      return () => clearTimeout(exitTimer);
    }, duration);

    return () => clearTimeout(timer);
  }, [duration, onClose]);

  return (
    <div
      className={`fixed top-6 left-1/2 -translate-x-1/2 z-[99999]
        flex items-start gap-3
        rounded-xl border border-[#EAEEF4]
        bg-white px-4 py-3
        shadow-lg
        max-w-[420px]
        text-left
        relative overflow-visible
        transition-all duration-200
        ${isExiting ? "opacity-0 scale-95" : "opacity-100 scale-100"}`}
      role="alert"
      aria-live="polite"
      aria-atomic="true"
    >
      {/* 图标 */}
      <span className="shrink-0 mt-0.5 flex items-center justify-center">
        {getIcon(type)}
      </span>

      {/* 消息文本 */}
      <span className="text-sm font-medium text-[#09090b] flex-1 break-words">
        {message}
      </span>

      {/* 关闭按钮 */}
      <button
        className="absolute -right-2 -top-2 flex h-5 w-5 items-center justify-center
          rounded-full bg-white border border-[#EAEEF4] shadow-sm
          text-[#7b818f] hover:bg-[#f4f4f5] hover:text-[#09090b]
          transition-colors flex-shrink-0"
        onClick={() => {
          setIsExiting(true);
          setTimeout(onClose, 200);
        }}
        aria-label="关闭通知"
        type="button"
      >
        <svg
          className="h-3.5 w-3.5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <line x1="18" y1="6" x2="6" y2="18" />
          <line x1="6" y1="6" x2="18" y2="18" />
        </svg>
      </button>
    </div>
  );
}

/**
 * Toast 容器组件 - 用于管理多个 toast
 * 允许同时显示多个 toast，按需添加和移除
 */
interface ToastItem {
  id: string | number;
  type: ToastType;
  message: string;
  duration?: number;
}

interface ToastContainerProps {
  toasts: ToastItem[];
  onRemove: (id: string | number) => void;
}

export function ToastContainer({ toasts, onRemove }: ToastContainerProps) {
  return (
    <div className="fixed top-0 left-0 right-0 z-[99999] pointer-events-none">
      <div className="flex flex-col items-center gap-3 pt-6 px-4">
        {toasts.map((toast) => (
          <div key={toast.id} className="pointer-events-auto">
            <PortableToast
              type={toast.type}
              message={toast.message}
              duration={toast.duration}
              onClose={() => onRemove(toast.id)}
            />
          </div>
        ))}
      </div>
    </div>
  );
}

/**
 * Hook: useToast
 * 用于在组件中管理多个 toast 的显示与隐藏
 *
 * 使用方式：
 *   const { toasts, showToast, removeToast } = useToast();
 *
 *   return (
 *     <>
 *       <ToastContainer toasts={toasts} onRemove={removeToast} />
 *       <button onClick={() => showToast("success", "操作成功")}>
 *         Show Toast
 *       </button>
 *     </>
 *   );
 */
export function useToast() {
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const showToast = (
    type: ToastType,
    message: string,
    duration?: number
  ) => {
    const id = Date.now();
    setToasts((prev) => [...prev, { id, type, message, duration }]);
  };

  const removeToast = (id: string | number) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
  };

  return {
    toasts,
    showToast,
    removeToast,
  };
}
