/**
 * ArrearsModal - 管控端账户欠费提醒弹窗（Figma 901_45616，532×314）
 *
 * 触发场景：席位欠费 D1（欠费首日 2h 缓冲期，isSeatArrearsBuffer === true）且当前在管控端页面时弹出；
 *          D1 缓冲期结束进入正式欠费后不再打扰，仅保留常态禁用 & tooltip 提示。
 * 视觉：532×314 圆角 12px 卡片，背景 linear-gradient(180deg, #FFF7ED 0%, #FFFFFF 100%)。
 *
 * 交互口径：
 *   - 「我知道了」/ 关闭 X：临时关闭（当天不再弹，次日复现）
 *   - 「立即充值」：跳转腾讯云费用中心
 *
 * 与其他停服弹窗通过 PopupQueueContext 排队。
 */
import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import { useServiceStatus } from "@/contexts/ServiceStatusContext";
import { usePopupSlot, POPUP_PRIORITY } from "@/contexts/PopupQueueContext";
import * as DialogPrimitive from "@radix-ui/react-dialog";

/** 腾讯云费用中心地址 */
const RECHARGE_URL = "https://console.cloud.tencent.com/account/recharge";

/** localStorage key：记录最近一次「今天已关闭」的日期，实现「每天弹一次」 */
const DAILY_DISMISS_KEY = "arrearsModal:dismissedDate";
const todayKey = () => new Date().toISOString().slice(0, 10);

export default function ArrearsModal() {
  const { isSeatArrearsBuffer } = useServiceStatus();

  const [dailyDismissed, setDailyDismissed] = useState(() => {
    if (typeof window === "undefined") return false;
    return localStorage.getItem(DAILY_DISMISS_KEY) === todayKey();
  });

  // Demo/开发：每次从"非缓冲期"切入"欠费（D1）"时，清掉当天关闭标记，便于反复验收。
  // 生产语义：单日关闭一次后当天不再弹；跨天次日恢复（保持不变）。
  const prevBufferRef = useRef(isSeatArrearsBuffer);
  useEffect(() => {
    if (isSeatArrearsBuffer && !prevBufferRef.current) {
      if (typeof window !== "undefined") {
        localStorage.removeItem(DAILY_DISMISS_KEY);
      }
      setDailyDismissed(false);
    }
    prevBufferRef.current = isSeatArrearsBuffer;
  }, [isSeatArrearsBuffer]);

  const wantShow = isSeatArrearsBuffer && !dailyDismissed;
  const canShow = usePopupSlot(
    "arrears-modal",
    POPUP_PRIORITY.SERVICE_SUSPENDED,
    wantShow,
  );

  const handleClose = () => {
    localStorage.setItem(DAILY_DISMISS_KEY, todayKey());
    setDailyDismissed(true);
  };

  const handleRecharge = () => {
    window.open(RECHARGE_URL, "_blank");
    handleClose();
  };

  return (
    <DialogPrimitive.Root
      open={canShow}
      // 弹窗关闭仅由显式动作触发（X / "知道了" / "立即续费"），均直接调用 handleClose。
      // 不接管 onOpenChange，避免 Radix 在 modal=false 模式下把外部 pointerDown（如点击右下角 Demo
      // 浮层的折叠按钮）误当成关闭意图。
      onOpenChange={() => {}}
      // modal={false}：不拦截 Portal 外部指针事件，让右下角三个 Demo 浮层（BillingStatusToggle /
      // AdminModeFloatingToggle / OnboardingDemoPanel）在弹窗打开时依然可点击。
      // Overlay（z-[80] 半透明黑）承担视觉遮罩，Demo 浮层 z-index 均 ≥ 9999 高于遮罩不受影响。
      // 「点击遮罩不关闭」仍由 Content 的 onPointerDownOutside preventDefault 保证。
      modal={false}
    >
      <DialogPrimitive.Portal>
        {/* 视觉遮罩：modal=false 时 Radix 不渲染 Overlay，手写 div 承担遮罩视觉。
            z-[80] 低于三个 Demo 浮层（z ≥ 9999）不影响浮层可点。 */}
        <div
          aria-hidden
          className="fixed inset-0 z-[80] bg-black/50 animate-in fade-in-0"
        />
        <DialogPrimitive.Content
          data-billing-exempt
          // 与 WarningExpireModal / TenantExpireModal 一致：点击弹窗外部不关闭，只走关闭按钮或跳转
          onPointerDownOutside={(e) => e.preventDefault()}
          className="fixed left-[50%] top-[50%] z-[81] translate-x-[-50%] translate-y-[-50%] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95"
        >
          {/* 532×314 卡片：浅橙渐变 → 白 */}
          <div
            className="relative w-[532px] h-[314px] rounded-[12px] overflow-hidden shadow-[0px_8px_10px_-6px_rgba(0,0,0,0.10),0px_20px_25px_-5px_rgba(0,0,0,0.10)]"
            style={{
              background: "linear-gradient(180deg, #FFF7ED 0%, #FFFFFF 100%)",
              fontFamily: "PingFang SC",
            }}
          >
            {/* 关闭 X：top:24 right:24 */}
            <button
              onClick={handleClose}
              aria-label="关闭"
              className="absolute right-[24px] top-[24px] w-[22px] h-[22px] flex items-center justify-center rounded-[4px] hover:bg-black/5 transition-colors"
            >
              <X className="w-[16px] h-[16px] text-slate-950" />
            </button>

            {/* 主标题：top:36 */}
            <div className="absolute left-[24px] top-[36px] text-[20px] font-semibold leading-[28px] text-slate-950">
              您的腾讯云账号已欠费，请尽快充值避免影响使用
            </div>

            {/* 子标题：top:84 */}
            <div className="absolute left-[24px] top-[84px] text-[14px] font-normal leading-[20px] tracking-[0.07px] text-slate-950">
              现在充值，避免影响正常使用
            </div>

            {/* 欠费影响 Label：top:118 */}
            <div className="absolute left-[24px] top-[118px] text-[12px] font-medium leading-[18px] tracking-[0.18px] text-slate-950">
              欠费影响
            </div>
            {/* 欠费影响 正文：top:142 */}
            <div className="absolute left-[24px] top-[142px] text-[12px] font-normal leading-[18px] tracking-[0.18px] text-slate-500">
              <div>运维观测及会话管理将无法采集新的数据；</div>
              <div>按量计费的Agent实例将被隔离无法正常使用，同时用户端也无法新建Agent实例</div>
            </div>

            {/* 恢复方式 Label：top:194 */}
            <div className="absolute left-[24px] top-[194px] text-[12px] font-medium leading-[18px] tracking-[0.18px] text-slate-950">
              恢复方式
            </div>
            {/* 恢复方式 正文：top:214 */}
            <div className="absolute left-[24px] top-[214px] text-[12px] font-normal leading-[18px] tracking-[0.18px] text-slate-500">
              充值恢复余额后服务自动恢复，无需额外操作
            </div>

            {/* 按钮区：top:258 left:322, 186×32，右对齐 */}
            <div className="absolute left-[322px] top-[258px] w-[186px] h-[32px] flex gap-[8px]">
              {/* 我知道了：89×32 白底 outline */}
              <button
                onClick={handleClose}
                className="w-[89px] h-[32px] rounded-[3.56px] bg-white text-[12px] font-normal tracking-[0.18px] text-slate-950 transition-colors hover:bg-slate-50"
                style={{ outline: "0.89px solid #E5E5E5", outlineOffset: "-0.89px" }}
              >
                我知道了
              </button>
              {/* 立即充值：89×32 黑底白字 */}
              <button
                onClick={handleRecharge}
                className="w-[89px] h-[32px] rounded-[3.56px] bg-slate-950 text-[12px] font-normal tracking-[0.18px] text-white transition-colors hover:bg-slate-800"
              >
                立即充值
              </button>
            </div>
          </div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
