/**
 * AdminArrearsFloatCard - 管控端席位「正式欠费」左下角常态提醒（两态合一）
 *
 * 触发场景：席位欠费正式档（isSeatArrearsBlocked === true，即开关面板「欠费」档，
 *           = 2h D1 缓冲结束后进入的正式欠费态）；管控端页面持续展示，作为常态提醒。
 *           与 D1 缓冲期的强提醒（ArrearsModal 中央弹窗 + TenantQuotaNotice 用户端右下角通知卡）
 *           形成互补：D1 强提醒一次 → 正式欠费改用管控端左下角常态提醒（浮层 / mini 条二态）持续但不打扰。
 *
 * 两态设计（互斥切换，均在正式欠费档持续常态展示）：
 *   ┌──────────────────────────┬────────────────────────────────────────────┐
 *   │ 态                       │ 视觉规范                                    │
 *   ├──────────────────────────┼────────────────────────────────────────────┤
 *   │ ① 浮层态（默认）         │ Figma 906_45782：220×169 浅橙渐变卡         │
 *   │   dismissed = false      │ 位置与 GuideAdminNotify 对齐（sidebar 内   │
 *   │                          │ 水平居中、footer 上方 12px）                │
 *   ├──────────────────────────┼────────────────────────────────────────────┤
 *   │ ② mini 条态（关闭后）    │ Figma 466_11416：240×44 sidebar 内通栏     │
 *   │   dismissed = true       │ 底色 #FFF3ED、顶部 1px #E2E8F0 分隔线      │
 *   │                          │ 左：⚠️ + "账户欠费，尽快充值" 12/400 black │
 *   │                          │ 右：蓝色"充值 →" #0052D9                    │
 *   │                          │ 位置：sidebar 内、footer 顶部之上（贴齐）   │
 *   │                          │ mini 条本身无 X，持续常态展示直至充值       │
 *   └──────────────────────────┴────────────────────────────────────────────┘
 *
 * 浮层态视觉规范（Figma 906_45782）：
 *   - 外壳 220×169、圆角 8px
 *   - 背景 linear-gradient(179deg, rgba(255,217.59,190.87,0.48) 0%, rgba(252,252,254,0.37) 100%), #FCFCFE
 *   - 阴影 0 4 12 rgba(0,0,0,0.10)
 *   - 描边 1px #E3E8FA（inside）
 *   - 内边距 12px
 *   - 顶行：小标签「重要通知」12/400 rgba(0,0,0,0.50) + 右上 X 关闭 → 切 mini 条态
 *   - 主标题「账户欠费，影响用户端使用」13/500 black
 *   - 正文 12/400 rgba(0,0,0,0.70)
 *   - CTA「立即充值 →」黑色胶囊 196×28、圆角 4px、白字 12/500，居中带右箭头
 *
 * mini 条态视觉规范（Figma 466_11416）：
 *   - 尺寸 240×44（宽正好等于 --admin-sidebar-width）
 *   - 底色 #FFF3ED；顶部 1px #E2E8F0 分隔线
 *   - 左侧 padding-left 16、图标 ⚠️（AlertCircle 12×12）+ 文案「账户欠费，尽快充值」12/400 black
 *   - 右侧 padding-right 16、蓝色链接「充值 →」#0052D9 12/400
 *
 * 交互口径：
 *   - 常态展示（不设"当天一次"限制，因正式欠费需要持续常态提醒）
 *   - 浮层态点 X：切换为 mini 条态（不再消失）
 *   - mini 条不可关闭，持续常态展示
 *   - sidebar 收起态（--admin-sidebar-width-collapsed=64px）下 mini 条隐藏（塞不下文案），
 *     等 sidebar 重新展开后自动恢复
 *   - 点击「立即充值 / 充值」：跳费用中心（当前 demo 不做真实跳转，可留 hook）
 *
 * 定位口径：
 *   - 浮层态：与「用户引导模拟 - 管控端非阻断弹窗（GuideAdminNotify）」保持一致 ——
 *     挂在 sidebar 内、水平居中于 sidebar、贴 sidebar footer 上方 12px。两者互斥占同一坑位。
 *   - mini 条态：sidebar 内通栏贴条，left=0、bottom=--admin-sidebar-footer-height，宽度铺满 sidebar
 */
import { useEffect, useRef, useState } from "react";
import { X, AlertCircle } from "lucide-react";
import { useServiceStatus } from "@/contexts/ServiceStatusContext";
import { useDeferredSidebarCollapsed } from "@/components/useDeferredSidebarCollapsed";
import {
  setArrearsMiniVisible,
  useExpireMiniVisible,
} from "@/components/AdminLeftMiniBarState";

export default function AdminArrearsFloatCard() {
  const { seatArrears, seatArrearsPhase, isSeatArrearsBuffer } = useServiceStatus();
  // 用 deferred 版本代替 useAdminSidebar().collapsed：
  //   - 收起→展开：延迟 300ms（对齐 sidebar transition-[width] duration-300）再解锁迷你条挂载
  //   - 展开→收起：立即返回 true 让迷你条第一时间隐藏
  // 这样与 WarningExpireFloat 完全同步，避免出现「一个告警条先展开、另一个后展开」的节奏错位。
  const collapsed = useDeferredSidebarCollapsed();
  // 订阅「到期/停服迷你条」可见状态，用于决定大卡片的 bottom：
  //   到期迷你条显示  → 抬高一个迷你条高度（+44），避免遮挡下方迷你条
  //   到期迷你条未显示 → 常规位置（贴 sidebar footer 上方 12px）
  const expireMiniVisible = useExpireMiniVisible();
  // 缓冲期（arrears-buffer）跳过大卡片，直接以迷你条形态常驻——避免与 ArrearsModal 中央弹窗视觉打架
  // 正式欠费（arrears）保留原行为：默认大卡片 → 用户 X 关闭后折叠为迷你条常驻
  const [dismissed, setDismissed] = useState(false);

  // 切档时重置 dismissed，让新档位从"默认态"开始
  //   active         → 组件不渲染
  //   arrears-buffer → 强制走迷你条（effectiveDismissed=true 由 isSeatArrearsBuffer 提供）
  //   arrears        → 重置为 false → 展示大卡片，用户 X 关闭后折叠成迷你条常驻
  const prevPhaseRef = useRef(seatArrearsPhase);
  useEffect(() => {
    if (prevPhaseRef.current !== seatArrearsPhase) {
      setDismissed(false);
      prevPhaseRef.current = seatArrearsPhase;
    }
  }, [seatArrearsPhase]);

  const effectiveDismissed = dismissed || isSeatArrearsBuffer;

  // 迷你条真实展示条件：欠费态（缓冲期 or 正式）+ dismissed（用户已关闭 or 缓冲期强制）+ sidebar 未收起
  const isMiniShowing = seatArrears && effectiveDismissed && !collapsed;

  // 上报给 WarningExpireFloat：让销毁倒计时迷你条决定要不要叠在自己上方
  useEffect(() => {
    setArrearsMiniVisible(isMiniShowing);
    return () => setArrearsMiniVisible(false);
  }, [isMiniShowing]);

  if (!seatArrears) return null;

  const handleRecharge = () => {
    // Demo：真实场景应跳转腾讯云费用中心；此处保留 hook
    // window.open("https://console.cloud.tencent.com/expense/recharge", "_blank");
  };

  // ── 态 ②：mini 条态（浮层被关闭后的收起态；缓冲期 arrears-buffer 强制走此态）
  if (effectiveDismissed) {
    // sidebar 收起时（64px）小条塞不下文案，先隐藏；展开后自动恢复
    if (collapsed) return null;
    return (
      <div
        className="fixed z-[45] animate-in fade-in duration-200"
        style={{
          left: 0,
          // 迷你条叠放规则（自底向上）：账户欠费迷你条在下、销毁倒计时迷你条在上
          bottom: "var(--admin-sidebar-footer-height)",
          width: "var(--admin-sidebar-width)",
          height: 44,
          background: "#FFF3ED",
          borderTop: "1px solid #E2E8F0",
        }}
        role="alert"
        aria-label="账户欠费提醒"
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
              账户欠费，尽快充值
            </span>
          </div>
          <button
            type="button"
            onClick={handleRecharge}
            className="inline-flex items-center gap-0.5 transition-opacity hover:opacity-80"
            style={{
              color: "#0052D9",
              fontSize: 12,
              fontWeight: 400,
              letterSpacing: "0.12px",
              lineHeight: "18px",
            }}
          >
            <span>充值</span>
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

  // ── 态 ①：浮层态（默认，Figma 906_45782）────────────────────────────
  return (
    <div
      className="fixed z-[45] animate-in slide-in-from-bottom-4 duration-300 transition-[bottom] duration-200"
      style={{
        // 与 GuideAdminNotify（管控端非阻断产品动态卡）对齐，占同一"左下坑位"
        left: "calc((var(--admin-sidebar-width) - 220px) / 2)",
        // 若「到期/停服迷你条」正在显示（贴 footer 上方一条 44px），
        // 则大卡片再上移一个迷你条高度，避免压住下方迷你条。
        bottom: expireMiniVisible
          ? "calc(var(--admin-sidebar-footer-height) + 44px + 12px)"
          : "calc(var(--admin-sidebar-footer-height) + 12px)",
        width: 220,
      }}
      role="alert"
      aria-label="账户欠费提醒"
      data-billing-exempt
    >
      <div
        className="relative overflow-hidden"
        style={{
          borderRadius: 8,
          background:
            "linear-gradient(179deg, rgba(255, 217.59, 190.87, 0.48) 0%, rgba(252, 252, 254, 0.37) 100%), #FCFCFE",
          boxShadow: "0px 4px 12px rgba(0, 0, 0, 0.10)",
          outline: "1px solid #E3E8FA",
          outlineOffset: -1,
        }}
      >
        <div className="p-3">
          {/* 顶行：小标签 + 关闭（关闭 → 切 mini 条态） */}
          <div className="relative pr-6" style={{ height: 21 }}>
            <span
              className="text-[12px] font-normal m-0"
              style={{
                color: "rgba(0, 0, 0, 0.50)",
                letterSpacing: "0.12px",
                lineHeight: "18px",
              }}
            >
              重要通知
            </span>
            <button
              type="button"
              onClick={() => setDismissed(true)}
              aria-label="收起为提示条"
              className="absolute right-0 top-0 inline-flex h-5 w-5 items-center justify-center rounded transition-colors hover:bg-black/5 active:bg-black/10"
              style={{ color: "rgba(0, 0, 0, 0.55)" }}
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          {/* 主标题 */}
          <p
            className="mt-2 text-[13px] font-medium m-0"
            style={{
              color: "#000",
              letterSpacing: "0.13px",
              lineHeight: "20px",
            }}
          >
            账户欠费，影响用户端使用
          </p>

          {/* 正文 */}
          <p
            className="mt-1 text-[12px] font-normal m-0"
            style={{
              color: "rgba(0, 0, 0, 0.70)",
              lineHeight: "18px",
            }}
          >
            您的腾讯云账号已欠费，请尽快前往费用中心充值避免影响正常使用。充值完成后服务将立即恢复。
          </p>

          {/* CTA：黑色胶囊按钮 196×28 */}
          <button
            type="button"
            onClick={handleRecharge}
            className="mt-2 inline-flex w-full items-center justify-center gap-1 transition-opacity hover:opacity-90 active:opacity-80"
            style={{
              height: 28,
              borderRadius: 4,
              background: "#000000",
              color: "#F8FAFC",
              fontSize: 12,
              fontWeight: 500,
              letterSpacing: "0.18px",
            }}
          >
            <span>立即充值</span>
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
                stroke="white"
                strokeWidth="0.7"
                strokeLinecap="round"
              />
              <path
                d="M11.6898 3.84002C11.8362 3.69357 12.0736 3.69357 12.2201 3.84002C12.3665 3.98647 12.3665 4.2239 12.2201 4.37035L7.27035 9.3201C7.1239 9.46654 6.88647 9.46654 6.74002 9.3201C6.59357 9.17365 6.59357 8.93621 6.74002 8.78977L11.6898 3.84002Z"
                fill="white"
              />
              <path
                d="M8.63022 4.10518C8.63022 3.89808 8.79808 3.73022 9.00518 3.73022L11.9549 3.73022C12.162 3.73022 12.3299 3.89808 12.3299 4.10518L12.3299 7.05493C12.3299 7.26204 12.162 7.42989 11.9549 7.42989C11.7478 7.42989 11.58 7.26204 11.58 7.05493L11.58 4.48014L9.00518 4.48014C8.79808 4.48014 8.63022 4.31229 8.63022 4.10518Z"
                fill="white"
              />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}
