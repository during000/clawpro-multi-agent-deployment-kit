/**
 * Toast Component - Demo & Examples
 * ─────────────────────────────────────────────────────────────────
 * 完整的 Toast 组件演示与测试组件。
 * 用于开发和测试所有 Toast 变体与交互。
 */

import { useState } from "react";
import { toast, withClose } from "@/components/ui/sonner";
import { useToast, ToastContainer, PortableToast } from "@/components/ui/toast";

/**
 * Toast 演示组件 - 展示所有类型与用法
 */
export function ToastDemo() {
  const { toasts, showToast, removeToast } = useToast();
  const [showSingleToast, setShowSingleToast] = useState(false);
  const [toastType, setToastType] = useState<
    "success" | "error" | "info" | "warning"
  >("success");

  return (
    <div className="min-h-screen bg-gradient-to-b from-gray-50 to-white p-8">
      <ToastContainer toasts={toasts} onRemove={removeToast} />

      {showSingleToast && (
        <PortableToast
          type={toastType}
          message={`这是一条 ${toastType} 类型的 Toast 提示`}
          onClose={() => setShowSingleToast(false)}
        />
      )}

      <div className="mx-auto max-w-4xl">
        {/* 标题 */}
        <div className="mb-12">
          <h1 className="mb-2 text-4xl font-bold text-gray-900">
            Toast 组件演示
          </h1>
          <p className="text-lg text-gray-600">
            完整的通知组件展示，包括所有类型、交互与使用方式。
          </p>
        </div>

        {/* ─────────────────────────────────────────────────────────────
            部分 1：使用 sonner 库
            ───────────────────────────────────────────────────────────── */}
        <section className="mb-12 rounded-lg border border-gray-200 bg-white p-8">
          <h2 className="mb-6 text-2xl font-bold text-gray-900">
            方式 1：使用 Sonner（推荐）
          </h2>

          <div className="grid grid-cols-2 gap-4 sm:grid-cols-1">
            {/* 简单提示 */}
            <button
              onClick={() => toast.success("操作成功！")}
              className="rounded-lg bg-green-600 px-6 py-3 font-medium text-white transition-all hover:bg-green-700 active:scale-95"
            >
              ✓ 成功提示
            </button>

            <button
              onClick={() => toast.error("操作失败，请重试")}
              className="rounded-lg bg-red-600 px-6 py-3 font-medium text-white transition-all hover:bg-red-700 active:scale-95"
            >
              ✗ 错误提示
            </button>

            <button
              onClick={() => toast.warning("配额即将用尽")}
              className="rounded-lg bg-amber-600 px-6 py-3 font-medium text-white transition-all hover:bg-amber-700 active:scale-95"
            >
              ⚠ 警告提示
            </button>

            <button
              onClick={() => toast.info("数据已更新")}
              className="rounded-lg bg-blue-600 px-6 py-3 font-medium text-white transition-all hover:bg-blue-700 active:scale-95"
            >
              ⓘ 信息提示
            </button>
          </div>

          {/* 带关闭按钮的提示 */}
          <div className="mt-6 border-t border-gray-200 pt-6">
            <p className="mb-4 text-sm font-medium text-gray-700">
              带关闭按钮的提示（长文本）
            </p>

            <button
              onClick={() => {
                const id = Date.now();
                toast.error(
                  () => (
                    <>
                      {`保存失败：权限不足，请联系管理员处理此问题。`}
                      {withClose(id)}
                    </>
                  ),
                  { id, duration: 0 } // duration: 0 表示不自动消失
                );
              }}
              className="rounded-lg bg-red-600 px-6 py-3 font-medium text-white transition-all hover:bg-red-700 active:scale-95"
            >
              显示错误（需手动关闭）
            </button>
          </div>

          {/* 自定义持续时间 */}
          <div className="mt-6 border-t border-gray-200 pt-6">
            <p className="mb-4 text-sm font-medium text-gray-700">
              自定义消失时间
            </p>

            <div className="flex gap-3">
              <button
                onClick={() => toast.success("2 秒后消失", { duration: 2000 })}
                className="rounded-lg bg-gray-600 px-4 py-2 text-sm font-medium text-white transition-all hover:bg-gray-700 active:scale-95"
              >
                2 秒
              </button>

              <button
                onClick={() => toast.success("6 秒后消失", { duration: 6000 })}
                className="rounded-lg bg-gray-600 px-4 py-2 text-sm font-medium text-white transition-all hover:bg-gray-700 active:scale-95"
              >
                6 秒
              </button>
            </div>
          </div>

          {/* 异步操作 */}
          <div className="mt-6 border-t border-gray-200 pt-6">
            <p className="mb-4 text-sm font-medium text-gray-700">
              异步操作反馈
            </p>

            <button
              onClick={() => {
                const promise = new Promise((resolve) => {
                  setTimeout(() => resolve(null), 2000);
                });

                toast.promise(promise, {
                  loading: "上传中...",
                  success: "上传成功！",
                  error: "上传失败",
                });
              }}
              className="rounded-lg bg-purple-600 px-6 py-3 font-medium text-white transition-all hover:bg-purple-700 active:scale-95"
            >
              模拟上传（2秒）
            </button>
          </div>
        </section>

        {/* ─────────────────────────────────────────────────────────────
            部分 2：使用独立 Toast（无依赖）
            ───────────────────────────────────────────────────────────── */}
        <section className="mb-12 rounded-lg border border-gray-200 bg-white p-8">
          <h2 className="mb-6 text-2xl font-bold text-gray-900">
            方式 2：使用独立 Toast（无 Sonner）
          </h2>

          <div className="grid grid-cols-2 gap-4 sm:grid-cols-1">
            {/* 四种类型按钮 */}
            {[
              { type: "success" as const, label: "✓ 成功", color: "bg-green-600" },
              { type: "error" as const, label: "✗ 错误", color: "bg-red-600" },
              { type: "warning" as const, label: "⚠ 警告", color: "bg-amber-600" },
              { type: "info" as const, label: "ⓘ 信息", color: "bg-blue-600" },
            ].map(({ type, label, color }) => (
              <button
                key={type}
                onClick={() => {
                  setToastType(type);
                  setShowSingleToast(true);
                }}
                className={`rounded-lg px-6 py-3 font-medium text-white transition-all active:scale-95 ${color} hover:opacity-90`}
              >
                {label}
              </button>
            ))}
          </div>

          {/* useToast Hook */}
          <div className="mt-6 border-t border-gray-200 pt-6">
            <p className="mb-4 text-sm font-medium text-gray-700">
              使用 useToast Hook
            </p>

            <div className="flex flex-wrap gap-3">
              <button
                onClick={() => showToast("success", "操作成功！")}
                className="rounded-lg bg-green-600 px-6 py-2 text-sm font-medium text-white transition-all hover:bg-green-700 active:scale-95"
              >
                成功
              </button>

              <button
                onClick={() =>
                  showToast(
                    "error",
                    "这是一条很长的错误提示，演示多行文本的显示效果。"
                  )
                }
                className="rounded-lg bg-red-600 px-6 py-2 text-sm font-medium text-white transition-all hover:bg-red-700 active:scale-95"
              >
                长文本
              </button>

              <button
                onClick={() => {
                  showToast("warning", "配额即将用尽", 3000);
                  showToast("info", "请尽快续费", 3000);
                }}
                className="rounded-lg bg-purple-600 px-6 py-2 text-sm font-medium text-white transition-all hover:bg-purple-700 active:scale-95"
              >
                多个 Toast
              </button>
            </div>
          </div>
        </section>

        {/* ─────────────────────────────────────────────────────────────
            部分 3：设计规范说明
            ───────────────────────────────────────────────────────────── */}
        <section className="rounded-lg border border-gray-200 bg-white p-8">
          <h2 className="mb-6 text-2xl font-bold text-gray-900">
            设计规范
          </h2>

          <div className="grid grid-cols-2 gap-8 sm:grid-cols-1">
            {/* 视觉规范 */}
            <div>
              <h3 className="mb-4 text-lg font-semibold text-gray-900">
                视觉规范
              </h3>
              <ul className="space-y-2 text-sm text-gray-700">
                <li className="flex items-center gap-2">
                  <span className="inline-block h-3 w-3 rounded-full bg-white ring-1 ring-gray-300"></span>
                  背景：白色 (#FFFFFF)
                </li>
                <li className="flex items-center gap-2">
                  <span className="inline-block h-3 w-3 rounded-full border border-gray-300"></span>
                  边框：#EAEEF4（蓝灰）
                </li>
                <li className="flex items-center gap-2">
                  <span className="text-gray-900">●</span>
                  文字色：#09090b（深灰/黑）
                </li>
                <li>圆角：12px (rounded-xl)</li>
                <li>内边距：12px 16px (py-3 px-4)</li>
                <li>字号：14px (text-sm)</li>
                <li>字重：500 (font-medium)</li>
                <li>阴影：shadow-lg</li>
                <li>定位：顶部居中</li>
                <li>Z-Index：99999</li>
                <li>自动消失：4000ms (4秒)</li>
              </ul>
            </div>

            {/* 强制规则 */}
            <div>
              <h3 className="mb-4 text-lg font-semibold text-gray-900">
                强制规则
              </h3>
              <div className="space-y-3">
                <div>
                  <p className="font-medium text-green-700">✓ DO</p>
                  <ul className="mt-1 space-y-1 text-sm text-gray-700">
                    <li>统一使用 toast API</li>
                    <li>所有类型白底统一风格</li>
                    <li>关闭按钮在右上角外侧</li>
                    <li>文字左对齐</li>
                  </ul>
                </div>

                <div>
                  <p className="font-medium text-red-700">✗ DON'T</p>
                  <ul className="mt-1 space-y-1 text-sm text-gray-700">
                    <li>不要按类型换底色</li>
                    <li>不要自行拼装 UI</li>
                    <li>不要用 sonner 内置 closeButton</li>
                    <li>不要放在其他位置</li>
                  </ul>
                </div>
              </div>
            </div>
          </div>

          {/* 代码示例 */}
          <div className="mt-8 border-t border-gray-200 pt-8">
            <h3 className="mb-4 text-lg font-semibold text-gray-900">
              代码示例
            </h3>

            <div className="grid grid-cols-2 gap-4 sm:grid-cols-1">
              <div className="rounded-lg bg-gray-50 p-4">
                <p className="mb-2 text-xs font-medium text-gray-600">
                  Sonner 方式
                </p>
                <pre className="overflow-x-auto text-xs text-gray-700">
{`import { toast } from 'sonner';

// 简单提示
toast.success("成功");

// 带关闭按钮
const id = Date.now();
toast.error(
  () => <>{msg}{withClose(id)}</>,
  { id }
);`}
                </pre>
              </div>

              <div className="rounded-lg bg-gray-50 p-4">
                <p className="mb-2 text-xs font-medium text-gray-600">
                  独立 Toast 方式
                </p>
                <pre className="overflow-x-auto text-xs text-gray-700">
{`import { useToast } from '@/components/ui/toast';

const { showToast } = useToast();

// 显示 toast
showToast(
  "success",
  "操作成功"
);`}
                </pre>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}

/**
 * 简化版演示 - 仅展示 4 种类型
 */
export function ToastSimpleDemo() {
  const { toasts, showToast, removeToast } = useToast();

  return (
    <>
      <ToastContainer toasts={toasts} onRemove={removeToast} />

      <div className="flex flex-col gap-3">
        <button
          onClick={() => showToast("success", "操作成功")}
          className="rounded bg-green-600 px-4 py-2 text-white hover:bg-green-700"
        >
          成功
        </button>
        <button
          onClick={() => showToast("error", "操作失败")}
          className="rounded bg-red-600 px-4 py-2 text-white hover:bg-red-700"
        >
          错误
        </button>
        <button
          onClick={() => showToast("warning", "警告提示")}
          className="rounded bg-amber-600 px-4 py-2 text-white hover:bg-amber-700"
        >
          警告
        </button>
        <button
          onClick={() => showToast("info", "信息提示")}
          className="rounded bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
        >
          信息
        </button>
      </div>
    </>
  );
}
