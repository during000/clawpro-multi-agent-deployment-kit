/**
 * TenantQuotaNotice - 用户端席位/配额不足右下角悬浮通知卡（360×175）
 *
 * 触发场景：席位处于任一欠费档（seatArrears === true，即 D1 缓冲期 或 正式欠费档），
 *           且当前在用户端页面时弹出；每天一次（当天关闭后 localStorage 记 YYYY-MM-DD，次日恢复）；
 *           管控端有独立的常态提醒（ArrearsModal / AdminArrearsFloatCard），此卡仅面向用户端。
 *
 * 视觉规范（Figma 466_12404）：
 *   - 外壳 360×175、圆角 8px、白底
 *   - 阴影 0 8 24 -4 rgba(0,0,0,0.10) + 0 8 12 -8 rgba(0,0,0,0.05)
 *   - 内边距 12px
 *   - 小标签「配额不足」#C04100 / 12px / 400
 *   - 主标题「当前团队配额已用满」 14px / 500 / rgba(0,0,0,0.90)
 *   - 正文 12px / 400 / rgba(0,0,0,0.70)
 *   - 底部右侧胶囊按钮 81×28 / 圆角 24px / #000 背景 / 白字「我知道了」
 *   - 右上角 X 关闭
 *
 * 交互口径：
 *   - 点击关闭 / 我知道了：当天不再弹（localStorage 记 YYYY-MM-DD），次日恢复
 *   - 位置：fixed right:24 bottom:24（右下角悬浮，避开 TopNav）
 */
import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import { useServiceStatus } from "@/contexts/ServiceStatusContext";

/** localStorage key：记录最近一次「今天已关闭」的日期，实现「每天弹一次」 */
const DAILY_DISMISS_KEY = "tenantQuotaNotice:dismissedDate";
const todayKey = () => new Date().toISOString().slice(0, 10); // YYYY-MM-DD

export default function TenantQuotaNotice() {
  const { seatArrears } = useServiceStatus();

  const [dailyDismissed, setDailyDismissed] = useState(() => {
    if (typeof window === "undefined") return false;
    return localStorage.getItem(DAILY_DISMISS_KEY) === todayKey();
  });

  // Demo/开发：每次从「非欠费」切入「任一欠费档」时，清掉当天关闭标记，便于反复验收。
  const prevArrearsRef = useRef(seatArrears);
  useEffect(() => {
    if (seatArrears && !prevArrearsRef.current) {
      if (typeof window !== "undefined") {
        localStorage.removeItem(DAILY_DISMISS_KEY);
      }
      setDailyDismissed(false);
    }
    prevArrearsRef.current = seatArrears;
  }, [seatArrears]);

  const shouldShow = seatArrears && !dailyDismissed;
  if (!shouldShow) return null;

  const handleClose = () => {
    if (typeof window !== "undefined") {
      localStorage.setItem(DAILY_DISMISS_KEY, todayKey());
    }
    setDailyDismissed(true);
  };

  return (
    <div
      className="fixed z-[10000]"
      style={{ right: 24, bottom: 24, width: 360 }}
      role="alertdialog"
      aria-label="团队配额已用满"
      data-billing-exempt
    >
      <div
        className="relative bg-white overflow-hidden"
        style={{
          borderRadius: 8,
          boxShadow:
            "0px 8px 24px -4px rgba(0, 0, 0, 0.10), 0px 8px 12px -8px rgba(0, 0, 0, 0.05)",
        }}
      >
        {/* 内容区：内边距 12px */}
        <div className="p-3">
          {/* 标题行 */}
          <div className="relative pr-6">
            {/* 小标签 */}
            <p
              className="text-[12px] font-normal leading-[18px] m-0"
              style={{ color: "#C04100", letterSpacing: "0.12px" }}
            >
              配额不足
            </p>
            {/* 主标题 */}
            <p
              className="mt-1 text-[14px] font-medium leading-[22px] m-0"
              style={{ color: "rgba(0, 0, 0, 0.90)", letterSpacing: "0.14px" }}
            >
              当前团队配额已用满
            </p>

            {/* 关闭按钮：右上角 */}
            <button
              type="button"
              onClick={handleClose}
              aria-label="关闭"
              className="absolute right-0 top-0 inline-flex h-5 w-5 items-center justify-center rounded transition-colors hover:bg-black/5 active:bg-black/10"
              style={{ color: "rgba(0, 0, 0, 0.55)" }}
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          {/* 正文 */}
          <p
            className="mt-3 text-[12px] font-normal leading-[20px] m-0"
            style={{ color: "rgba(0, 0, 0, 0.70)", letterSpacing: "0.12px" }}
          >
            团队可用配额已达上限，暂时无法继续创建新的 agent，已创建的 agent 正常使用。
            <br />
            请联系管理员提升配额，即可恢复正常使用。
          </p>

          {/* 底部：右下角胶囊按钮 81×28 */}
          <div className="mt-2 flex justify-end">
            <button
              type="button"
              onClick={handleClose}
              className="inline-flex items-center justify-center transition-opacity hover:opacity-90 active:opacity-80"
              style={{
                width: 81,
                height: 28,
                borderRadius: 24,
                background: "#000000",
                color: "#FFFFFF",
                fontSize: 12,
                fontWeight: 400,
                letterSpacing: "0.30px",
              }}
            >
              我知道了
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
