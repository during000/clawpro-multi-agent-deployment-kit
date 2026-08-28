/**
 * ServiceSuspendedModal - 管控台停服弹窗
 * 
 * 管控台到期停服时，首次进入管控端弹出模态弹窗：
 * - 文案：管控台服务已到期，当前仅支持查看，所有操作已禁用。请尽快前往腾讯云费用中心续费。
 * - "立即续费"按钮跳转续费管理页
 * - 支持"下次不再提醒"勾选项
 */
import { useState } from "react";
import { AlertTriangle, ExternalLink, X } from "lucide-react";
import * as DialogPrimitive from "@radix-ui/react-dialog";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { useServiceStatus } from "@/contexts/ServiceStatusContext";
import { usePopupSlot, POPUP_PRIORITY } from "@/contexts/PopupQueueContext";

/** 腾讯云费用中心续费管理页地址 */
const RENEW_URL = "https://console.cloud.tencent.com/account/renewal";

export default function ServiceSuspendedModal() {
  const { consoleStatus, suspendedModalDismissed, dismissSuspendedModal, recyclingDaysLeft, isInRecycling } =
    useServiceStatus();
  const [dontRemind, setDontRemind] = useState(false);
  // 当前会话内的临时关闭标记（不持久化，刷新后重置）
  const [sessionDismissed, setSessionDismissed] = useState(false);

  // 只在停服状态且未被永久关闭且未被当前会话临时关闭时展示
  const wantShow =
    consoleStatus === "suspended" &&
    !suspendedModalDismissed &&
    !sessionDismissed;

  // 接入全局弹窗优先级队列：排在视觉焕新之后、组织变更之前
  const shouldShow = usePopupSlot(
    "service-suspended",
    POPUP_PRIORITY.SERVICE_SUSPENDED,
    wantShow,
  );

  const handleClose = () => {
    if (dontRemind) {
      // 永久关闭（写入localStorage，后续刷新也不再弹出）
      dismissSuspendedModal();
    } else {
      // 临时关闭（仅当前会话内不再弹出，刷新页面后会再次弹出）
      setSessionDismissed(true);
    }
  };

  const handleRenew = () => {
    window.open(RENEW_URL, "_blank");
  };

  return (
    <DialogPrimitive.Root
      open={shouldShow}
      // 弹窗关闭仅由显式动作触发（X / "我知道了" / "立即续费"），均直接调用 handleClose。
      // 不接管 onOpenChange，避免 Radix 在 modal=false 模式下把外部 pointerDown（如点击右下角 Demo
      // 浮层的折叠按钮）误当成关闭意图。
      onOpenChange={() => {}}
      // modal={false}：不拦截 Portal 外部指针事件，让右下角三个 Demo 浮层（BillingStatusToggle /
      // AdminModeFloatingToggle / OnboardingDemoPanel）在弹窗打开时依然可点击。
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
          // 与其他催费/到期弹窗一致：点击弹窗外部不关闭
          onPointerDownOutside={(e) => e.preventDefault()}
          className="fixed left-[50%] top-[50%] z-[51] grid w-full max-w-[calc(100%-2rem)] sm:max-w-sm translate-x-[-50%] translate-y-[-50%] rounded-[12px] bg-white shadow-[0_6px_16px_0_rgba(0,0,0,0.08),0_3px_6px_-4px_rgba(0,0,0,0.12),0_9px_28px_8px_rgba(0,0,0,0.05)] px-6 py-0 outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 duration-200"
        >
          {/* 续费弹窗本身即为管控台停服的告知/引导入口，
              其内的所有交互（我知道了 / 立即续费 / 下次不再提醒 / 关闭 X）都需在停服态下保持可用 */}
          <button
            type="button"
            onClick={handleClose}
            aria-label="关闭"
            className="absolute top-[26px] right-5 flex items-center justify-center size-5 rounded-sm text-[#7b818f] transition-colors hover:text-gray-950"
          >
            <X className="size-5" />
          </button>

          <div className="flex flex-col justify-center gap-0.5 -mx-6 px-6 pt-6 pb-3 shrink-0">
            <div className="flex items-center gap-3 mb-2">
              <div
                className="w-10 h-10 rounded-[12px] flex items-center justify-center shrink-0 border"
                style={{
                  backgroundColor: "var(--alert-warning-bg)",
                  borderColor: "var(--alert-warning-border)",
                }}
              >
                <AlertTriangle
                  className="w-5 h-5"
                  style={{ color: "var(--alert-warning-icon)" }}
                />
              </div>
              <DialogPrimitive.Title className="text-lg font-medium text-[var(--text-title)]">
                管控台服务已到期
              </DialogPrimitive.Title>
            </div>
            <DialogPrimitive.Description className="text-sm text-[var(--text-secondary)] leading-relaxed">
              管控台服务已到期，当前仅支持查看，所有操作已禁用。请尽快前往腾讯云费用中心续费。
              {isInRecycling && recyclingDaysLeft !== null && (
                <span className="block mt-2 font-medium text-[var(--text-danger)]">
                  管控台将在 {recyclingDaysLeft} 天后永久删除，届时数据将无法恢复。
                </span>
              )}
            </DialogPrimitive.Description>
          </div>

          <div className="flex items-center gap-2 mt-2">
            <Checkbox
              id="dont-remind"
              checked={dontRemind}
              onCheckedChange={(checked) => setDontRemind(checked === true)}
            />
            <label
              htmlFor="dont-remind"
              className="text-xs text-[var(--text-muted)] cursor-pointer select-none"
            >
              下次不再提醒
            </label>
          </div>

          <div className="flex items-center gap-3 justify-end -mx-6 px-6 pt-6 pb-6 mt-4">
            <Button variant="claw-outline" size="claw-sm" onClick={handleClose}>
              我知道了
            </Button>
            <Button variant="claw-primary" size="claw-sm" onClick={handleRenew}>
              <ExternalLink className="w-3.5 h-3.5" />
              立即续费
            </Button>
          </div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
