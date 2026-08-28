/**
 * PluginUpgradeFloating - CLS 采集插件升级 - 右下角浮动组件
 *
 * 设计参考：VSCode / Cursor 的扩展安装进度条
 *
 * 形态：
 * - running：右下角 320px 卡片，显示标题 + 真实进度条 + N/M 分数
 * - succeeded：卡片变成绿色「升级成功」，3.5s 后自动消失
 * - failed：卡片变红，显示已升级数
 *
 * 跨页面常驻 + localStorage 持久化（在 PluginUpgradeContext 实现）
 */
import { useEffect } from "react";
import { Loader2, CheckCircle2, X } from "lucide-react";
import { usePluginUpgrade } from "@/contexts/PluginUpgradeContext";

const SUCCESS_AUTO_DISMISS_MS = 3500;

export function PluginUpgradeFloating() {
  const { status, total, progress, dismiss } = usePluginUpgrade();

  // succeeded 后自动 dismiss
  useEffect(() => {
    if (status !== "succeeded") return;
    const timer = window.setTimeout(() => dismiss(), SUCCESS_AUTO_DISMISS_MS);
    return () => window.clearTimeout(timer);
  }, [status, dismiss]);

  // idle 不渲染
  if (status === "idle") return null;

  const pct = total > 0 ? Math.min(100, (progress / total) * 100) : 0;

  // ─── 成功形态 ─────────────────────────────────────────────────────
  if (status === "succeeded") {
    return (
      <div
        className="fixed bottom-5 right-5 z-[9998] w-[320px] bg-white rounded-lg border border-gray-200 p-4 animate-in fade-in slide-in-from-bottom-2"
        style={{ boxShadow: "0 4px 12px rgba(0,0,0,0.08), 0 1px 3px rgba(0,0,0,0.04)" }}
      >
        <div className="flex items-center gap-3">
          <CheckCircle2 className="w-4 h-4 text-green-600 flex-shrink-0" />
          <div className="flex-1 text-sm font-medium text-gray-900">
            CLS 采集插件升级成功
          </div>
          <button
            onClick={dismiss}
            className="text-gray-400 hover:text-gray-600 transition-colors"
            title="关闭"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    );
  }

  // ─── 失败形态 ─────────────────────────────────────────────────────
  if (status === "failed") {
    return (
      <div
        className="fixed bottom-5 right-5 z-[9998] w-[320px] bg-white rounded-lg border border-red-200 p-4 animate-in fade-in slide-in-from-bottom-2"
        style={{ boxShadow: "0 4px 12px rgba(0,0,0,0.08), 0 1px 3px rgba(0,0,0,0.04)" }}
      >
        <div className="flex items-start gap-3">
          <div className="w-4 h-4 rounded-full bg-red-100 flex items-center justify-center flex-shrink-0 mt-0.5">
            <X className="w-3 h-3 text-red-600" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium text-gray-900 mb-0.5">
              CLS 采集插件升级失败
            </div>
            <div className="text-xs text-gray-500">
              已升级 {progress} / {total}
            </div>
          </div>
          <button
            onClick={dismiss}
            className="text-gray-400 hover:text-gray-600 transition-colors flex-shrink-0"
            title="关闭"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    );
  }

  // ─── 默认形态：卡片（running） ────────────────────────────────────
  return (
    <div
      className="fixed bottom-5 right-5 z-[9998] w-[320px] bg-white rounded-lg border border-gray-200 p-4 animate-in fade-in slide-in-from-bottom-2"
      style={{ boxShadow: "0 4px 12px rgba(0,0,0,0.08), 0 1px 3px rgba(0,0,0,0.04)" }}
    >
      {/* 头部：标题 */}
      <div className="flex items-center gap-3 mb-3">
        <Loader2 className="w-4 h-4 text-blue-600 animate-spin flex-shrink-0" />
        <div className="flex-1 text-sm font-medium text-gray-900">
          CLS 采集插件升级中
        </div>
      </div>

      {/* 进度条 + 分数（同一行） */}
      <div className="flex items-center gap-3">
        <div className="flex-1 relative h-1.5 overflow-hidden rounded-full bg-blue-100">
          <div
            className="absolute left-0 top-0 h-full rounded-full bg-blue-500 transition-all duration-300 ease-out"
            style={{ width: `${pct}%` }}
          />
        </div>
        <span className="text-xs font-medium text-gray-600 tabular-nums whitespace-nowrap">
          {progress} / {total}
        </span>
      </div>
    </div>
  );
}
