/**
 * TenantExpireModal - 用户端停服提醒弹窗（600×456）
 *
 * 触发场景：管控台已停服（consoleStatus === "suspended"），且当前处于「关键日」档
 *   即 phase === "suspended-d1"（覆盖 D1 & D8 两个关键日）。
 *   D2-D7 / D9-D12 中间档、以及 D13-D15 回收站末段不弹此窗，避免用户端过度打扰。
 *
 * 视觉规范：Figma 898_45615
 *   - 外壳 600×456 圆角 12px、白底、阴影，背景图 tenant-suspended-bg.png
 *   - 内容区 552×408，四周内边距 24px
 *   - 标题 #020617 / 20px semibold；数字 #1447E6 / 24px bold（DIN Next LT Pro）
 *   - 副标题 #64748B / 12px；正文 slate-700 / 14px
 *   - 底部左：Checkbox「下次不再提醒」；底部右：黑色胶囊按钮「我知道了」
 *
 * 交互口径：
 *   - 未勾「下次不再提醒」→ 点击关闭：当天不再弹（localStorage 记 YYYY-MM-DD），次日恢复
 *   - 勾选「下次不再提醒」→ 点击关闭：永久不再弹（除非 phase 切换到新档）
 *
 * 与其他自动弹窗通过 PopupQueueContext 排队，共用 SERVICE_SUSPENDED 优先级。
 */
import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { useServiceStatus } from "@/contexts/ServiceStatusContext";
import { usePopupSlot, POPUP_PRIORITY } from "@/contexts/PopupQueueContext";
import * as DialogPrimitive from "@radix-ui/react-dialog";

/** localStorage key：记录最近一次「今天已关闭」的日期，实现「每天弹一次，当天关闭后不再弹」 */
const DAILY_DISMISS_KEY = "tenantExpireModal:dismissedDate";
/** localStorage key：勾了「下次不再提醒」→ 永久不弹（记录 phase 值，切档后失效） */
const NEVER_KEY = "tenantExpireModal:neverForPhase";
const todayKey = () => new Date().toISOString().slice(0, 10); // YYYY-MM-DD

export default function TenantExpireModal() {
  const { phase, isAdminDisabled, recyclingDaysLeft } = useServiceStatus();

  // 「今天已关闭」标记：localStorage 记 YYYY-MM-DD，跨天后失效 → 又能弹
  const [dailyDismissed, setDailyDismissed] = useState(() => {
    if (typeof window === "undefined") return false;
    return localStorage.getItem(DAILY_DISMISS_KEY) === todayKey();
  });

  // 「永久不再提醒」标记：记录当时 phase，phase 未变则不再弹
  const [neverForPhase, setNeverForPhase] = useState(() => {
    if (typeof window === "undefined") return null;
    return localStorage.getItem(NEVER_KEY);
  });

  // 当前弹窗内的「下次不再提醒」勾选态（本次未关闭前的临时勾选）
  const [dontRemind, setDontRemind] = useState(false);

  // 切档时清除标记，让新档能重新弹一次
  const prevPhaseRef = useRef(phase);
  useEffect(() => {
    if (prevPhaseRef.current !== phase) {
      if (typeof window !== "undefined") {
        localStorage.removeItem(DAILY_DISMISS_KEY);
        localStorage.removeItem(NEVER_KEY);
      }
      setDailyDismissed(false);
      setNeverForPhase(null);
      setDontRemind(false);
      prevPhaseRef.current = phase;
    }
  }, [phase]);

  // 触发条件：停服 D1/D8 关键日档；今天已关闭 / 永久不再提醒则不弹
  const wantShow =
    isAdminDisabled &&
    phase === "suspended-d1" &&
    !dailyDismissed &&
    neverForPhase !== phase;
  const shouldShow = usePopupSlot(
    "tenant-expire",
    POPUP_PRIORITY.SERVICE_SUSPENDED,
    wantShow,
  );

  const handleClose = () => {
    if (typeof window !== "undefined") {
      if (dontRemind) {
        // 永久标记：记录当前 phase
        localStorage.setItem(NEVER_KEY, phase);
        setNeverForPhase(phase);
      } else {
        // 当日标记
        localStorage.setItem(DAILY_DISMISS_KEY, todayKey());
        setDailyDismissed(true);
      }
    } else {
      setDailyDismissed(true);
    }
  };

  const daysLeft = recyclingDaysLeft ?? 15;

  return (
    <DialogPrimitive.Root
      open={shouldShow}
      // 弹窗关闭仅由显式动作触发（X / "我知道了" 等），均直接调用 handleClose。
      // 不接管 onOpenChange，避免 Radix 在 modal=false 模式下把外部 pointerDown（如点击右下角 Demo
      // 浮层的折叠按钮）误当成关闭意图。
      onOpenChange={() => {}}
      // modal={false}：不拦截 Portal 外部指针事件，让右下角三个 Demo 浮层（BillingStatusToggle /
      // AdminModeFloatingToggle / OnboardingDemoPanel）在弹窗打开时依然可点击。
      // Overlay 承担视觉遮罩，Demo 浮层 z-index 均 ≥ 9999 高于遮罩不受影响。
      // 「点击遮罩不关闭」由 Content 的 onPointerDownOutside preventDefault 保证。
      modal={false}
    >
      <DialogPrimitive.Portal>
        {/* 视觉遮罩：modal=false 时 Radix 不渲染 Overlay，手写 div 承担遮罩视觉。
            z-50 低于三个 Demo 浮层（z ≥ 9999）不影响浮层可点。 */}
        <div
          aria-hidden
          className="fixed inset-0 z-50 bg-black/45 animate-in fade-in-0"
        />
        <DialogPrimitive.Content
          data-billing-exempt
          onPointerDownOutside={(e) => e.preventDefault()}
          className="fixed top-[50%] left-[50%] z-50 -translate-x-[50%] -translate-y-[50%] w-[600px] h-[456px] rounded-[12px] bg-white overflow-hidden shadow-[0_8px_10px_-6px_rgba(0,0,0,0.10),0_20px_25px_-5px_rgba(0,0,0,0.10)] outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95"
          style={{
            backgroundImage: 'url("/assets/warning-modal/tenant-suspended-bg.png")',
            backgroundSize: "cover",
            backgroundPosition: "center",
            backgroundRepeat: "no-repeat",
          }}
        >
          <DialogPrimitive.Title className="sr-only">
            ClawPro 服务已到期
          </DialogPrimitive.Title>
          <DialogPrimitive.Description className="sr-only">
            ClawPro 服务已到期，账户资源即将销毁，请联系管理员续费。
          </DialogPrimitive.Description>

          {/* 关闭按钮 X（位于 24px 内边距外的绝对定位，右上角贴弹窗边） */}
          <button
            type="button"
            data-billing-exempt
            onClick={handleClose}
            aria-label="关闭"
            className="absolute right-5 top-5 z-10 flex h-6 w-6 items-center justify-center rounded-[4px] text-[var(--text-weak)] hover:text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)] transition-colors"
          >
            <X className="h-4 w-4" />
          </button>

          {/* 内容区：距离弹窗上下左右 24px，内部 552×408 */}
          <div className="absolute inset-x-6 top-6 bottom-6 flex flex-col">
            {/* 标题行：ClawPro服务已到期，距离销毁账户资源还剩 [15] 天 */}
            <div className="flex flex-wrap items-baseline gap-x-1 pr-8">
              <span className="text-[20px] font-semibold text-[#020617] leading-[28px]">
                ClawPro服务已到期，距离销毁账户资源还剩
              </span>
              <span
                className="text-[24px] font-bold"
                style={{
                  color: "#1447E6",
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

            {/* 副标题：来自系统的通知 */}
            <p className="mt-2 text-[12px] font-normal leading-[1.6] tracking-[0.18px] text-[#64748B]">
              来自系统的通知
            </p>

            {/* 正文（Figma 861:50130）：整段紧凑连排，<br/> 分行，无段间距 */}
            <div className="mt-6 flex-1">
              <p className="text-[14px] font-medium leading-[1.6] tracking-[0.07px] text-[#020617]">
                尊敬的用户，您好：
              </p>
              {/* 段 1 */}
              <p className="mt-2 text-[14px] font-normal leading-[22px] tracking-[0.07px] text-[#334155]">
                ClawPro服务已到期，目前已暂停服务。
              </p>
              {/* 段 2：段间距一个空行（22px） */}
              <p className="mt-[22px] text-[14px] font-normal leading-[22px] tracking-[0.07px] text-[#334155]">
                当前用户端已无法新建 Agent（已有 Agent 可继续使用）。
              </p>
              {/* 段 3：三行紧邻换行，无段间距 */}
              <p className="mt-[22px] text-[14px] font-normal leading-[22px] tracking-[0.07px] text-[#334155]">
                管理员续费后可立即恢复服务。
                <br />
                如未续费，用户端将在&nbsp;
                <span className="font-medium text-[#020617]">
                  xxxx年xx月xx日
                </span>
                &nbsp;无法继续访问。
              </p>
              {/* 段 4 */}
              <p className="mt-[22px] text-[14px] font-normal leading-[22px] tracking-[0.07px] text-[#334155]">
                再次感谢您对ClawPro的支持。给各位带来不便，敬请谅解。
              </p>
            </div>

            {/* 分割线（Figma 861:50119）：552×1，#E2E8F0，距上下文字各 16px */}
            <div
              role="separator"
              aria-hidden="true"
              className="my-4 h-px w-full bg-[#E2E8F0]"
            />

            {/* 联系人行（Figma 861:50120：14×14 电话图标 + 文字） */}
            <div className="flex items-center gap-1.5">
              <svg
                width="14"
                height="14"
                viewBox="0 0 14 14"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
                aria-hidden="true"
                className="shrink-0"
              >
                <path
                  d="M12.25 8.46563L9.67477 7.31118L9.66493 7.3068C9.49753 7.23464 9.31472 7.20558 9.1332 7.2223C8.95169 7.23902 8.77726 7.30098 8.62586 7.4025C8.60464 7.41671 8.5842 7.43205 8.56461 7.44844L7.34454 8.4875C6.6336 8.10196 5.89914 7.37352 5.51305 6.67133L6.55539 5.43211C6.57215 5.41211 6.58767 5.39111 6.60188 5.36922C6.70074 5.21837 6.76077 5.04541 6.77661 4.86574C6.79245 4.68608 6.76361 4.50528 6.69266 4.33946C6.69098 4.33628 6.68951 4.33299 6.68829 4.32961L5.53438 1.75C5.43966 1.53427 5.27801 1.35477 5.07334 1.23806C4.86867 1.12134 4.63187 1.07362 4.39797 1.10196C3.60411 1.20624 2.87536 1.59593 2.34781 2.19825C1.82027 2.80057 1.53001 3.57432 1.53125 4.375C1.53125 8.83805 5.16196 12.4688 9.625 12.4688C10.4257 12.47 11.1994 12.1797 11.8018 11.6522C12.4041 11.1247 12.7938 10.3959 12.8981 9.60204C12.9264 9.36814 12.8787 9.13133 12.762 8.92667C12.6452 8.722 12.4657 8.56035 12.25 8.46563ZM9.625 11.1563C7.82717 11.1541 6.10359 10.4389 4.83233 9.16767C3.56107 7.89641 2.84592 6.17284 2.84375 4.375C2.8425 3.92192 2.99632 3.48204 3.27966 3.12847C3.56299 2.7749 3.95878 2.52892 4.40125 2.43141L5.43047 4.72829L4.38266 5.97625C4.36572 5.99644 4.35002 6.01762 4.33563 6.03969C4.23235 6.19749 4.17163 6.37932 4.15937 6.56752C4.1471 6.75571 4.18371 6.94388 4.26563 7.11375C4.78079 8.16813 5.84227 9.2225 6.90758 9.73875C7.07859 9.81983 7.26771 9.85514 7.45645 9.84124C7.64519 9.82734 7.82709 9.7647 7.98438 9.65946C8.00553 9.64519 8.0258 9.62967 8.04508 9.61297L9.27172 8.57008L11.5686 9.59875C11.4711 10.0412 11.2251 10.437 10.8715 10.7204C10.518 11.0037 10.0781 11.1575 9.625 11.1563Z"
                  fill="#334155"
                />
              </svg>
              <p className="text-[12px] font-normal leading-[1.6] tracking-[0.18px] text-[#334155]">
                如有任何疑问，欢迎联系您的专属客户经理。
              </p>
            </div>

            {/* 底部：Checkbox + 按钮 */}
            <div className="mt-4 flex items-center justify-between">
              <label className="flex items-center gap-2 cursor-pointer select-none">
                <Checkbox
                  checked={dontRemind}
                  onCheckedChange={(v) => setDontRemind(v === true)}
                  data-billing-exempt
                />
                <span className="text-[12px] font-normal leading-[1.6] tracking-[0.18px] text-[#334155]">
                  下次不再提醒
                </span>
              </label>
              <Button
                variant="tenant-primary"
                size="claw-sm"
                onClick={handleClose}
                data-billing-exempt
                className="min-w-[105px]"
              >
                我知道了
              </Button>
            </div>
          </div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
