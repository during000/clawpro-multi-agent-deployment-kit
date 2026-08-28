/**
 * GuideUpdateBar - 强提醒公告条（告警样式）
 * 对应场景：元素层 2.5 / 2.6 / 跨端层 5.x / 时间维度 6.1 预告期
 * 
 * 特点：
 * - 不可关闭（强提醒）
 * - 参考 Alert 组件 warning 样式（琥珀色背景+图标）
 * - 位于导航栏下方，sticky 固定，页面滚动后不消失
 * - 模拟欠费/重要公告场景
 */
import { AlertTriangle, ChevronRight } from "lucide-react";
import { createPortal } from "react-dom";
import { useEffect, useState } from "react";

interface GuideUpdateBarProps {
  open: boolean;
  /** 公告文案 */
  message: string;
  /** 版本号标签 */
  version?: string;
  /** 查看详情回调 */
  onDetail?: () => void;
  /** 查看详情文案 */
  detailText?: string;
}

export function GuideUpdateBar({
  open,
  message,
  version,
  onDetail,
  detailText = "查看详情",
}: GuideUpdateBarProps) {
  const [portalTarget, setPortalTarget] = useState<HTMLElement | null>(null);

  useEffect(() => {
    if (!open) {
      const existing = document.getElementById("onboarding-update-bar-portal");
      if (existing) existing.remove();
      setPortalTarget(null);
      return;
    }

    let target = document.getElementById("onboarding-update-bar-portal");
    if (!target) {
      target = document.createElement("div");
      target.id = "onboarding-update-bar-portal";
      target.style.position = "sticky";
      target.style.top = "64px";
      target.style.zIndex = "49";
      target.style.width = "100%";

      // 插入到 header 后面
      const nav = document.querySelector("header.sticky, header[class*='sticky']");
      if (nav && nav.parentElement) {
        nav.parentElement.insertBefore(target, nav.nextSibling);
      } else {
        // fallback: 找 main 前面插入
        const main = document.querySelector("main");
        if (main && main.parentElement) {
          main.parentElement.insertBefore(target, main);
        }
      }
    }
    setPortalTarget(target);

    return () => {
      const el = document.getElementById("onboarding-update-bar-portal");
      if (el) el.remove();
    };
  }, [open]);

  if (!open || !portalTarget) return null;

  return createPortal(
    <div className="w-full bg-[#FEF3C7] border-b border-[#F59E0B]/30 px-4 py-2 animate-in slide-in-from-top-1 duration-200">
      <div className="max-w-7xl mx-auto flex items-center justify-between gap-4">
        {/* 左侧内容 */}
        <div className="flex items-center gap-2.5 min-w-0">
          <AlertTriangle className="w-4 h-4 shrink-0 text-[#D97706]" />
          {version && (
            <span className="shrink-0 px-1.5 py-0.5 text-[10px] font-medium text-[#92400E] bg-[#FDE68A] rounded">
              {version}
            </span>
          )}
          <span className="text-sm text-[#92400E]">{message}</span>
        </div>

        {/* 右侧操作 */}
        {onDetail && (
          <button
            onClick={onDetail}
            className="inline-flex items-center gap-0.5 text-xs font-medium text-[#D97706] hover:text-[#92400E] transition-colors shrink-0"
          >
            {detailText}
            <ChevronRight className="w-3.5 h-3.5" />
          </button>
        )}
      </div>
    </div>,
    portalTarget,
  );
}
