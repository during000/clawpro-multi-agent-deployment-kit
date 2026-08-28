/**
 * WarningExpireModal - 管控台到期临期预警弹窗（620x442）
 *
 * 触发场景：warning-d1 / warning-d2-6 / critical-d7 三档任一进入管控端首次弹出。
 * 视觉：620x442 圆角卡片，背景图 warning-bg.png，12px 阴影。
 *
 * 交互口径：
 *   - 「我知道了」：临时关闭（当前会话不再弹，刷新后再弹；切换 phase 档位会自动重置）
 *   - 「立即续费」：跳转腾讯云费用中心续费管理页
 *
 * 与 ServiceSuspendedModal 通过 PopupQueueContext 排队，phase 天然互斥。
 */
import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useServiceStatus } from "@/contexts/ServiceStatusContext";
import { usePopupSlot, POPUP_PRIORITY } from "@/contexts/PopupQueueContext";
import * as DialogPrimitive from "@radix-ui/react-dialog";

const RENEW_URL = "https://console.cloud.tencent.com/account/renewal";

/** 到期停服后将失去的能力清单（Figma 861_49465 蓝 / 861_49583 橙，172×128 内部 Frame） */
const LOSE_ITEMS = [
  "用户/分组等平台策略",
  "Agent配置（模型、通道等）",
  "Agent服务（网盘、记忆等）",
  "安全审计（凭据、策略等）",
];

/** 续费即享 6+ 企业特权（Figma 861_49514，424×230，2 列 × 3 行 grid） */
const PRIVILEGE_ITEMS = [
  { col: 60, row: 62, title: "全面管控", desc: "统一管理Agent全生命周期" },
  { col: 60, row: 114, title: "开箱即用", desc: "10分钟部署，零技术门槛" },
  { col: 60, row: 166, title: "提效增速", desc: "运维效率综合提升80%+" },
  { col: 231, row: 62, title: "安全合规", desc: "权限审计+内容安全全覆盖" },
  { col: 231, row: 114, title: "全模型适配", desc: "一个平台接入主流大模型" },
  { col: 231, row: 166, title: "生态开放", desc: "MCP/技能/插件统一分发管控" },
];

/** localStorage key：记录最近一次「今天已关闭」的日期，实现「每天弹一次，当天关闭后不再弹」 */
const DAILY_DISMISS_KEY = "warningExpireModal:dismissedDate";
const todayKey = () => new Date().toISOString().slice(0, 10); // YYYY-MM-DD

export default function WarningExpireModal() {
  const {
    isNearingExpire,
    phase,
    daysUntilExpire,
    isAdminDisabled,
    recyclingDaysLeft,
  } = useServiceStatus();

  // 「今天已关闭」标记：localStorage 记 YYYY-MM-DD，重新到明天该值失效 → 又能弹
  const [dailyDismissed, setDailyDismissed] = useState(() => {
    if (typeof window === "undefined") return false;
    return localStorage.getItem(DAILY_DISMISS_KEY) === todayKey();
  });

  // 停服态视为"最紧迫档"：走橙色主题（复用 D7 视觉）
  const isFinalDay = phase === "critical-d7" || isAdminDisabled;

  // 顶部主数字：临期档取 daysUntilExpire；停服档取回收站剩余天数
  const daysLeft = isAdminDisabled
    ? recyclingDaysLeft ?? 15
    : daysUntilExpire ?? 1;

  // 数字颜色：停服最后一天（回收站剩 1 天）→ 红色最强告警；D7/其他停服 → 橙色；临期 D1 → 品牌蓝
  const isLastRecyclingDay = isAdminDisabled && daysLeft === 1;
  const daysColor = isLastRecyclingDay
    ? "#DC2626"
    : isFinalDay
    ? "#B8640A"
    : "#1447E6";

  // 左侧「失去」Frame 三级主题：
  //   - 停服最后一天（Figma 861_49897）：渐变蓝→红 + icon 红版 + Header 文案改为已失权限
  //   - 临期 D7 / 其他停服（Figma 861_49583）：渐变蓝→橙 + icon 橙版
  //   - 临期 D1（Figma 861_49465）：纯蓝渐变 + icon 蓝版
  const loseGradient = isLastRecyclingDay
    ? "linear-gradient(270deg, #EDF3FF 0%, #FEF2F2 100%)"
    : isFinalDay
    ? "linear-gradient(270deg, #EDF3FF 0%, #FFF7ED 100%)"
    : "linear-gradient(270deg, #EDF3FF 100%)";
  const loseIconSuffix = isLastRecyclingDay
    ? "-red"
    : isFinalDay
    ? "-orange"
    : "";
  // 「失去」Frame 顶部措辞：
  //   - 停服态（isAdminDisabled，含 D1 / D8 / …/ 最后一天）：权限已实际收回，
  //     用完成时「您已经失去以下管理及操作权限」；
  //   - 临期未停服档（warning-d1 / critical-d7 等）：尚未停服，用将来时「到期停服后，您将失去」。
  const loseHeader = isAdminDisabled
    ? "您已经失去以下管理及操作权限"
    : "您将失去以下管理及操作权限";

  // 切档（如 D1 → D7 → suspended）时清除「今天已关闭」标记，让新档能重新弹一次
  const prevPhaseRef = useRef(phase);
  useEffect(() => {
    if (prevPhaseRef.current !== phase) {
      if (typeof window !== "undefined") {
        localStorage.removeItem(DAILY_DISMISS_KEY);
      }
      setDailyDismissed(false);
      prevPhaseRef.current = phase;
    }
  }, [phase]);

  // 触发条件：临期档（排除 warning-d2-6）或 停服档（排除 suspended-d2-12 中间档）；今天已关闭则不再弹
  const wantShow =
    ((isAdminDisabled && phase !== "suspended-d2-12") ||
      (isNearingExpire && phase !== "warning-d2-6")) &&
    !dailyDismissed;
  const shouldShow = usePopupSlot(
    "warning-expire",
    POPUP_PRIORITY.SERVICE_SUSPENDED,
    wantShow,
  );

  const handleClose = () => {
    if (typeof window !== "undefined") {
      localStorage.setItem(DAILY_DISMISS_KEY, todayKey());
    }
    setDailyDismissed(true);
  };
  const handleRenew = () => window.open(RENEW_URL, "_blank");

  return (
    <DialogPrimitive.Root
      open={shouldShow}
      // 弹窗关闭仅由三个显式动作触发：右上角 X / "我知道了" / "立即续费"，均直接调用 handleClose。
      // 不接管 onOpenChange，避免 Radix 在 modal=false 模式下把「外部 pointerDown / Escape / focus loss」
      // 当作 close 请求，误把点击右下角 Demo 浮层的"折叠"按钮当成关闭本弹窗的意图。
      onOpenChange={() => {}}
      // modal={false}：不拦截 Portal 外部指针事件，让右下角三个 Demo 浮层（BillingStatusToggle /
      // AdminModeFloatingToggle / OnboardingDemoPanel，z-index 均 ≥ 9999）在弹窗打开时依然可点击。
      // 视觉遮罩由 Overlay 的 bg-black/45（z-50）承担，Demo 浮层 z-index 高于遮罩不被视觉压制。
      // 「点击遮罩不关闭」仍由 Content 的 onPointerDownOutside preventDefault 保证。
      modal={false}
    >
      <DialogPrimitive.Portal>
        {/* 视觉遮罩：因 modal=false 时 Radix Dialog.Overlay 不会渲染（规范上非模态弹窗没有遮罩），
            这里手写一个普通 div 承担半透明黑遮罩视觉。z-50 低于三个 Demo 浮层（z ≥ 9999），
            不影响浮层可点；遮罩自身默认 pointer-events: auto，仍会吃掉遮罩区域的点击，
            不会穿透到底层页面元素。 */}
        <div
          aria-hidden
          className="fixed inset-0 z-50 bg-black/45 animate-in fade-in-0"
        />
        <DialogPrimitive.Content
          data-billing-exempt
          onPointerDownOutside={(e) => e.preventDefault()}
          className="fixed top-[50%] left-[50%] z-50 -translate-x-[50%] -translate-y-[50%] w-[620px] h-[442px] rounded-[12px] bg-white overflow-hidden shadow-[0_8px_10px_-6px_rgba(0,0,0,0.10),0_20px_25px_-5px_rgba(0,0,0,0.10)] outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95"
          style={{
            backgroundImage: `url("/assets/warning-modal/${
              isAdminDisabled ? "warning-bg-suspended.png" : "warning-bg.png"
            }")`,
            backgroundSize: "cover",
            backgroundPosition: "center",
            backgroundRepeat: "no-repeat",
          }}
        >
          <DialogPrimitive.Title className="sr-only">
            管控台免费试用即将到期
          </DialogPrimitive.Title>
          <DialogPrimitive.Description className="sr-only">
            续费即享企业特权，续费后管控台服务将继续可用。
          </DialogPrimitive.Description>

          {/* 关闭按钮 X */}
          <button
            type="button"
            data-billing-exempt
            onClick={handleClose}
            aria-label="关闭"
            className="absolute right-5 top-5 z-10 flex h-6 w-6 items-center justify-center rounded-[4px] text-[var(--text-weak)] hover:text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)] transition-colors"
          >
            <X className="h-4 w-4" />
          </button>

          {/* 顶部标题区：临期档「您的管控台免费试用仅剩 N 天」；停服档「管控台服务已到期，距离销毁账户资源还剩 N 天」 */}
          <div className="absolute left-6 top-8 right-16 z-10">
            <div className="flex flex-wrap items-baseline gap-x-1 gap-y-1">
              <span className="text-[20px] font-semibold text-[#020617] leading-[28px]">
                {isAdminDisabled
                  ? "管控台服务已到期，距离销毁账户资源还剩"
                  : "您的管控台免费试用仅剩"}
              </span>
              <span
                className="text-[24px] font-bold"
                style={{
                  color: daysColor,
                  fontFamily: '"DIN Next LT Pro", "PingFang SC", sans-serif',
                  lineHeight: "28px",
                }}
              >
                {daysLeft}
              </span>
              <span className="text-[20px] font-semibold text-[#020617] leading-[28px]">
                天
              </span>
            </div>
            {/* 描述文案：临期档沿用原文案；停服档换成 Figma 861_49728 版（末尾日期 + 提示走品牌蓝加粗） */}
            {isAdminDisabled ? (
              <p className="mt-2 text-[12px] leading-[1.6] tracking-[0.18px] text-[#0F172A]">
                当前用户端已无法新建&nbsp;Agent（已有&nbsp;Agent&nbsp;可继续使用），管控端可访问但无法配置操作，续费后可立即恢复服务。用户端和管控端将在&nbsp;
                <span className="font-medium text-[#1447E6]">
                  2026年8月15日&nbsp;无法继续访问
                </span>
              </p>
            ) : (
              <p className="mt-2 text-[12px] leading-[1.6] tracking-[0.18px] text-[#0F172A]">
                ClawPro 将于8月15日起正式收费。到期未续费管控台将在到期后1天自动停服，停服期间用户将无法新建Agent；停服后为您提供15天保留期，15天后管控台将被销毁释放，相关数据不可恢复
              </p>
            )}
          </div>

          {/* 渐变 Frame —— 334×172（Figma 861_49465 蓝 / 861_49583 D7 橙），距弹窗左边 0px、底部 109px */}
          <div
            className="absolute z-10"
            style={{
              left: 0,
              bottom: 109,
              width: 334,
              height: 172,
              background: loseGradient,
            }}
          >
            {/* 内部内容 Frame —— 172×128（canvas: left:24, top:22） */}
            <div
              className="absolute"
              style={{ left: 24, top: 22, width: 172, height: 128 }}
            >
              {/* Header 文案（临期档 vs 停服最后一天不同措辞） */}
              <div className="absolute left-0 top-0 w-[220px] text-[12px] font-medium leading-none tracking-[0.18px] text-[#020617]">
                {loseHeader}
              </div>
              {/* 4 项列表 Frame —— 172×98（canvas: left:0, top:30），icon 与文字 flex 垂直居中 */}
              <div
                className="absolute flex flex-col"
                style={{ left: 0, top: 30, width: 172, height: 98, rowGap: 14 }}
              >
                {LOSE_ITEMS.map((text, i) => (
                  <div key={text} className="flex items-center gap-1">
                    <img
                      src={`/assets/warning-modal/lose-${i + 1}${loseIconSuffix}.svg`}
                      alt=""
                      aria-hidden
                      className="h-3 w-3 shrink-0 select-none"
                    />
                    <span className="text-[12px] leading-none tracking-[0.18px] text-[#64748B]">
                      {text}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* 右侧「续费即享 6+ 企业特权」Frame —— 424×230（Figma 861_49514），距弹窗右侧 0px、底部 80px */}
          <div
            className="absolute z-10"
            style={{ right: 0, bottom: 80, width: 424, height: 230 }}
          >
            {/* 斜切渐变背板（Figma 889_45275 1.svg，left:10, top:0，尺寸 414×231） */}
            <img
              src="/assets/warning-modal/privilege-bg.svg"
              alt=""
              aria-hidden
              className="pointer-events-none absolute select-none"
              style={{ left: 10, top: 0, width: 414, height: 231 }}
            />
            {/* 标题 Frame —— left:60, top:22（闪光 icon + 文案，flex 水平居中对齐） */}
            <div
              className="absolute flex items-center gap-1"
              style={{ left: 60, top: 22, height: 24 }}
            >
              <img
                src="/assets/warning-modal/privilege-sparkle.svg"
                alt=""
                aria-hidden
                className="h-5 w-5 shrink-0 select-none"
              />
              <span className="text-[16px] font-semibold leading-none text-[#0080FF]">
                续费即享&nbsp;6+&nbsp;企业特权
              </span>
            </div>
            {/* 6 项特权：2 列 × 3 行，标题 + 描述（canvas 精确坐标） */}
            {PRIVILEGE_ITEMS.map((it) => (
              <div key={`${it.col}-${it.row}`}>
                <div
                  className="absolute text-[12px] font-medium leading-none tracking-[0.18px] text-[#0F172A]"
                  style={{ left: it.col, top: it.row }}
                >
                  {it.title}
                </div>
                <div
                  className="absolute text-[12px] leading-none tracking-[0.18px] text-[#64748B]"
                  style={{ left: it.col, top: it.row + 20 }}
                >
                  {it.desc}
                </div>
              </div>
            ))}
          </div>

          {/* 底部按钮区 —— 两个按钮固定 89px 宽 */}
          <div className="absolute bottom-6 right-6 z-10 flex items-center gap-2">
            <Button
              variant="claw-outline"
              size="claw-sm"
              onClick={handleClose}
              data-billing-exempt
              className="w-[89px]"
            >
              我知道了
            </Button>
            <Button
              size="claw-sm"
              onClick={handleRenew}
              data-billing-exempt
              className="w-[89px] bg-[#020617] text-white hover:bg-[#0F172A]"
            >
              立即续费
            </Button>
          </div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
