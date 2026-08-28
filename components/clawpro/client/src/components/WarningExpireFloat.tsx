/**
 * WarningExpireFloat - 「预警 D2-6」/「停服」阶段左下角悬浮预警组件
 *
 * 展示矩阵：
 *   phase                | 大卡片（初次进入）        | 迷你条（常驻）
 *   ---------------------|--------------------------|--------------------------
 *   warning-d2-6         | 橙版（Figma 440_45663）  | 橙版（440_47632）
 *   suspended-d1         | 无（直接迷你条）          | 红版（466_11004）
 *   suspended-d2-12      | 红版（Figma 896_45679）  | 红版（466_11004）
 *   recycling-d13-15     | 无（直接迷你条）          | 红版（466_11004）
 *
 * 两态：
 *  1. 展开态（首次进入档位时默认，仅 warning-d2-6 / suspended-d2-12 有）：220×209 左下角大卡片
 *     - fixed left:10, bottom:82
 *     - X 关闭后 → 折叠成迷你条（不是消失）
 *  2. 折叠态（常驻）：240×44 迷你条
 *     - fixed left:0, bottom:72（贴紧 sidebar footer 上边缘）
 *     - 整个停服/预警档位期间常驻；切档到别的档位后自动重置回展开态
 *     - 迷你条无独立关闭按钮，「续费 →」跳转腾讯云费用中心
 */
import { useEffect, useRef, useState } from "react";
import { AlertCircle } from "lucide-react";
import { useServiceStatus } from "@/contexts/ServiceStatusContext";
import { useDeferredSidebarCollapsed } from "@/components/useDeferredSidebarCollapsed";
import {
  useArrearsMiniVisible,
  setExpireMiniVisible,
} from "@/components/AdminLeftMiniBarState";

const RENEW_URL = "https://console.cloud.tencent.com/account/renewal";
const EXPIRE_DATE = "2026年8月15日";

/** 两态：expanded（大卡片）↔ collapsed（迷你条），切换无 exit 动画（对齐 GuideModuleFloat 直接卸载） */
type FloatStage = "expanded" | "collapsed";

export default function WarningExpireFloat() {
  const { phase, daysUntilExpire, recyclingDaysLeft } = useServiceStatus();
  // 订阅账户欠费迷你条的可见状态，用于决定本迷你条的 bottom：
  //   欠费迷你条显示  → 叠在其上方（+44）
  //   欠费迷你条未显示 → 下移到贴 sidebar footer 位置
  const arrearsMiniVisible = useArrearsMiniVisible();
  // sidebar 收起态（--admin-sidebar-width-collapsed=64px）下迷你条塞不下文案。
  // 用 useDeferredSidebarCollapsed 而不是原始 useAdminSidebar().collapsed：
  //   - 收起：立即为 true → 迷你条第一时间隐藏，避免溢出到 64px 收起态外
  //   - 展开：延迟 300ms（与 sidebar transition-[width] duration-300 对齐）后才变 false
  //     → 保证 sidebar 宽度动画走完再挂载迷你条，避免「告警条比侧边栏先出现」
  // 所有 sidebar 内告警条（本组件+ AdminArrearsFloatCard）共用同一 hook，节奏完全一致。
  const deferredSidebarCollapsed = useDeferredSidebarCollapsed();

  const isWarning = phase === "warning-d2-6";
  // 停服 3 档全部走「销毁倒计时」红色迷你条 —— 常驻直至充值
  const isSuspended =
    phase === "suspended-d1" ||
    phase === "suspended-d2-12" ||
    phase === "recycling-d13-15";
  // 只有 warning-d2-6 / suspended-d2-12 有大卡片设计；其余停服档位直接进入迷你态
  const hasExpandedCard = isWarning || phase === "suspended-d2-12";
  const [stage, setStage] = useState<FloatStage>(
    hasExpandedCard ? "expanded" : "collapsed",
  );

  // 切档时：有大卡片的档位重置为 expanded，其他档位保持 collapsed 常驻
  const prevPhaseRef = useRef(phase);
  useEffect(() => {
    if (prevPhaseRef.current !== phase) {
      setStage(hasExpandedCard ? "expanded" : "collapsed");
      prevPhaseRef.current = phase;
    }
  }, [phase, hasExpandedCard]);

  // 上报自身「销毁倒计时迷你条」可见状态给 AdminArrearsFloatCard 大卡片订阅：
  // 只有在到期/停服档位 + 迷你态 + 有 phase 命中 + sidebar 未收起（且展开动画已完成） 时，
  // 才算迷你条正在显示。sidebar 收起时迷你条实际隐藏，必须同步上报为false，
  // 否则欠费大卡片会误以为本条占了 44px 而错误抬升位置。
  const isExpireMiniShowing =
    (isWarning || isSuspended) && stage === "collapsed" && !deferredSidebarCollapsed;
  useEffect(() => {
    setExpireMiniVisible(isExpireMiniShowing);
    return () => setExpireMiniVisible(false);
  }, [isExpireMiniShowing]);

  if (!isWarning && !isSuspended) return null;

  const handleRenew = () => window.open(RENEW_URL, "_blank");
  const handleCollapse = () => setStage("collapsed");

  const daysLeft = isSuspended
    ? recyclingDaysLeft ?? (phase === "recycling-d13-15" ? 1 : phase === "suspended-d1" ? 14 : 8)
    : daysUntilExpire ?? 4;

  // ─── 折叠态：sidebar 内通栏迷你条（与 AdminArrearsFloatCard 迷你条同一套骨架） ────────
  // 视觉规范统一：40x44 通栏、flex 布局、AlertCircle + 文字 + 蓝色链接 + ArrowRight。
  // 销毁倒计时（红版）与账户欠费迷你条并排叠放时字号/间距/图标视觉完全一致，
  // 只在配色（背景/图标色）、文案和「数字告警强调」上区分。
  if (stage === "collapsed") {
    // sidebar 收起（64px）时迷你条塞不下文案，与欠费迷你条策略一致直接不渲染；
    // 展开时 deferred 值会延迟 300ms 才变 false，确保 sidebar 宽度动画走完再挂载迷你条，
    // 避免「告警条比侧边栏先出现」。红版（销毁倒计时）与橙版（warning-d2-6）都走这里，一并处理。
    if (deferredSidebarCollapsed) return null;
    const mini = isSuspended
      ? {
          bg: "#FCE8E8",
          iconColor: "#DC2626",
          numberColor: "#DC2626",
          prefix: "服务域名销毁仅剩 ",
          suffix: " 天",
          cta: "续费",
        }
      : {
          bg: "#FFF3ED",
          iconColor: "#F97316",
          numberColor: "#C04100",
          prefix: "服务还剩 ",
          suffix: " 天到期",
          cta: "续费",
        };

    return (
      <div
        className="fixed z-[45] transition-[bottom] duration-200"
        style={{
          left: 0,
          // 迷你条叠放规则（自底向上）：账户欠费迷你条在下（贴 footer），
          // 销毁倒计时迷你条在上（叠在欠费之上 44px）。
          // 独立出现时（账户欠费迷你条未显示）自动下移到贴 sidebar footer 位置，避免中间出现空档。
          bottom: arrearsMiniVisible
            ? "calc(var(--admin-sidebar-footer-height) + 44px)"
            : "var(--admin-sidebar-footer-height)",
          width: "var(--admin-sidebar-width)",
          height: 44,
          background: mini.bg,
          borderTop: "1px solid #E2E8F0",
        }}
        role="alert"
        aria-label="服务到期提醒"
        data-billing-exempt
      >
        <div className="flex h-full items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <AlertCircle
              className="h-3 w-3 shrink-0"
              style={{ color: "#000000" }}
              strokeWidth={2}
              aria-hidden
            />
            <span
              className="text-[12px] font-normal"
              style={{ color: "#000", letterSpacing: "0.12px", lineHeight: "18px" }}
            >
              {mini.prefix}
              <span
                className="font-semibold"
                style={{ color: mini.numberColor }}
              >
                {daysLeft}
              </span>
              {mini.suffix}
            </span>
          </div>
          <button
            type="button"
            onClick={handleRenew}
            className="inline-flex items-center gap-0.5 transition-opacity hover:opacity-80"
            style={{
              color: "#0052D9",
              fontSize: 12,
              fontWeight: 400,
              letterSpacing: "0.12px",
              lineHeight: "18px",
            }}
            data-billing-exempt
          >
            <span>{mini.cta}</span>
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="16"
              height="16"
              viewBox="0 0 16 16"
              fill="none"
              aria-hidden
              className="shrink-0"
            >
              <path
                d="M6.94922 4.09766H4.44922C4.17308 4.09766 3.94922 4.32151 3.94922 4.59766V11.5977C3.94922 11.8738 4.17308 12.0977 4.44922 12.0977H11.4492C11.7254 12.0977 11.9492 11.8738 11.9492 11.5977V9.09766"
                stroke="#1447E6"
                strokeWidth="0.7"
                strokeLinecap="round"
              />
              <path
                d="M11.6898 3.84002C11.8362 3.69357 12.0736 3.69357 12.2201 3.84002C12.3665 3.98647 12.3665 4.2239 12.2201 4.37035L7.27035 9.3201C7.1239 9.46654 6.88647 9.46654 6.74002 9.3201C6.59357 9.17365 6.59357 8.93621 6.74002 8.78977L11.6898 3.84002Z"
                fill="#1447E6"
              />
              <path
                d="M8.63022 4.10518C8.63022 3.89808 8.79808 3.73022 9.00518 3.73022L11.9549 3.73022C12.162 3.73022 12.3299 3.89808 12.3299 4.10518L12.3299 7.05493C12.3299 7.26204 12.162 7.42989 11.9549 7.42989C11.7478 7.42989 11.58 7.26204 11.58 7.05493L11.58 4.48014L9.00518 4.48014C8.79808 4.48014 8.63022 4.31229 8.63022 4.10518Z"
                fill="#1447E6"
              />
            </svg>
          </button>
        </div>
      </div>
    );
  }

  // ─── 展开态：220×209 左下角大卡片（橙色 D2-6 版 / 红色 停服 D2-D7 版） ────────
  // 顶部渐变 & 主标题/描述按档位切换（红色版 = Figma 896_45679）
  // 展开态只在 warning-d2-6 / suspended-d2-12 出现（hasExpandedCard=true 才会进入本分支），
  // 这里 isSuspendedMiddle 用来区分「橙 warning-d2-6」和「红 suspended-d2-12」两套样式/文案。
  const isSuspendedMiddle = phase === "suspended-d2-12";
  const cardGradient = isSuspendedMiddle
    ? "linear-gradient(174deg, rgba(255, 133, 133, 0.48) -85.04%, rgba(252, 252, 254, 0.37) 95.52%), #FCFCFE"
    : "linear-gradient(179deg, rgba(255, 217.59, 190.87, 0.48) 0%, rgba(252, 252, 254, 0.37) 100%), #FCFCFE";

  return (
    <div
      className="fixed z-[45] animate-in slide-in-from-bottom-4 duration-300 transition-[bottom] duration-200"
      style={{
        // 与 AdminArrearsFloatCard 浮层态对齐：sidebar 内水平居中、贴 sidebar footer 上方 12px。
        // 但若账户欠费迷你条正在显示（贴 footer 上方一条 44px），则将大卡片再上移一个迷你条高度，
        // 避免遮挡下面的欠费迷你条。
        left: "calc((var(--admin-sidebar-width) - 220px) / 2)",
        bottom: arrearsMiniVisible
          ? "calc(var(--admin-sidebar-footer-height) + 44px + 12px)"
          : "calc(var(--admin-sidebar-footer-height) + 12px)",
        width: 220,
        height: 209,
      }}
      data-billing-exempt
    >
      <div
        className="relative h-full w-full overflow-hidden"
        style={{
          borderRadius: 8,
          background: cardGradient,
          boxShadow: "0px 4px 12px rgba(0, 0, 0, 0.10)",
          outline: "1px solid #E3E8FA",
          outlineOffset: -1,
        }}
      >
        {/* 顶部行：left:12, top:12, 196×21 */}
        <div
          className="absolute flex items-center justify-between"
          style={{ left: 12, top: 12, width: 196, height: 21 }}
        >
          <span
            className="text-[12px] font-normal leading-none tracking-[0.12px]"
            style={{ color: "rgba(0, 0, 0, 0.50)" }}
          >
            重要通知
          </span>
          <button
            type="button"
            onClick={handleCollapse}
            aria-label="收起为迷你提示条"
            className="flex h-5 w-5 items-center justify-center rounded transition-colors hover:bg-black/5"
            data-billing-exempt
          >
            <img
              src="/assets/warning-float/close.svg"
              alt=""
              aria-hidden
              className="h-5 w-5 select-none"
            />
          </button>
        </div>

        {/* 内容区：left:12, top:41, 196×156 */}
        <div
          className="absolute"
          style={{ left: 12, top: 41, width: 196, height: 156 }}
        >
          {/* 主标题：13px 500，数字告警色 #C04100 */}
          <div
            className="absolute left-0 top-0 text-[13px] font-medium tracking-[0.13px]"
            style={{ color: "#000" }}
          >
            {isSuspendedMiddle ? (
              <>
                仅剩{" "}
                <span className="font-medium" style={{ color: "#C04100" }}>
                  {daysLeft}
                </span>{" "}
                天，服务域名将被销毁
              </>
            ) : (
              <>
                距离服务到期仅剩{" "}
                <span className="font-medium" style={{ color: "#C04100" }}>
                  {daysLeft}
                </span>{" "}
                天
              </>
            )}
          </div>

          {/* 描述文案：top:24, 196 宽自适应换行 */}
          <p
            className="absolute left-0 top-[24px] w-[196px] text-[12px] font-normal leading-[1.6]"
            style={{ color: "rgba(0, 0, 0, 0.70)" }}
          >
            {isSuspendedMiddle ? (
              <>
                服务已于
                <span className="font-medium" style={{ color: "#C04100" }}>
                  {EXPIRE_DATE}
                </span>
                到期，当前已暂停服务。服务将在&nbsp;{EXPIRE_DATE}&nbsp;无法继续访问，相关数据将被
                <span className="font-semibold" style={{ color: "#C04100" }}>
                  永久删除，不可恢复
                </span>
                。如需继续使用，请尽快前往续费。
              </>
            ) : (
              <>
                您的服务将于
                <span className="font-semibold" style={{ color: "#C04100" }}>
                  {EXPIRE_DATE}
                </span>
                到期。到期后1天自动停服。届时管控端可访问但无法配置操作，用户端将无法新建&nbsp;Agent。续费后可立即恢复服务。
              </>
            )}
          </p>

          {/* CTA 按钮：top:128, 196×28，黑底圆角 4px */}
          <button
            type="button"
            onClick={handleRenew}
            data-billing-exempt
            className="absolute left-0 flex w-[196px] items-center justify-center gap-1 rounded-[4px] text-[12px] font-medium leading-none tracking-[0.18px] text-[#F8FAFC] transition-opacity hover:opacity-90"
            style={{ top: 128, height: 28, background: "#000" }}
          >
            立刻续费
            <img
              src="/assets/warning-float/arrow.svg"
              alt=""
              aria-hidden
              className="h-4 w-4 select-none"
            />
          </button>
        </div>
      </div>
    </div>
  );
}
