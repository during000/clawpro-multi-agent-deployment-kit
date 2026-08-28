/**
 * TenantExpireNotice - 用户端停服中间档 & 回收站末段右下角悬浮通知卡（360×173）
 *
 * 触发场景：管控台已停服，且当前处于以下两档之一：
 *   1. phase === "suspended-d2-12"    → D2-D7 & D9-D12 中间档（Figma 466_12380）
 *   2. phase === "recycling-d13-15"   → D13-D15 回收站末段（Figma 898_44837）
 *
 * D1/D8 关键日走 TenantExpireModal 全屏弹窗。
 *
 * 视觉规范（两档共享外壳、按钮、关闭图标，仅文案不同）：
 *   - 外壳 360×173、圆角 8px、白底
 *   - 阴影 0 8 24 -4 rgba(0,0,0,0.10) + 0 8 12 -8 rgba(0,0,0,0.05)
 *   - 内边距 12px
 *   - 小标签「服务已到期」#C04100 / 12px / 400
 *   - 主标题 14px / 500，数字/关键字用 #C04100，两档统一：「距离域名无法访问仅剩 X 天」
 *   - 正文 12px / 400 / rgba(0,0,0,0.7)，日期/警告词高亮 #C04100
 *   - 底部右侧胶囊按钮 73×26 / 圆角 65px / #202020 背景 / 白字「我知道了」
 *   - 右上角 X 关闭
 *
 * 交互口径：
 *   - 点击关闭：当天不再弹（localStorage 记 YYYY-MM-DD），次日恢复
 *   - 切档时清标记，让新档能重新弹一次
 *   - 位置：fixed right:24 bottom:24（右下角悬浮，避开 TopNav）
 */
import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import { useServiceStatus } from "@/contexts/ServiceStatusContext";

/** localStorage key：记录最近一次「今天已关闭」的日期，实现「每天弹一次，当天关闭后不再弹」 */
const DAILY_DISMISS_KEY = "tenantExpireNotice:dismissedDate";
const todayKey = () => new Date().toISOString().slice(0, 10); // YYYY-MM-DD

/** 服务永久关闭日期文案（与 WarningExpireFloat / TenantSuspendedNoticeBar 保持一致口径） */
const EXPIRE_DATE = "2026年8月15日";
/** 数据永久删除日期（D15 时刻，Figma 898_44837 示例值） */
const PERMANENT_DELETE_DATE = "2026年8月31日";

export default function TenantExpireNotice() {
  const { phase, isAdminDisabled, recyclingDaysLeft } = useServiceStatus();

  const [dailyDismissed, setDailyDismissed] = useState(() => {
    if (typeof window === "undefined") return false;
    return localStorage.getItem(DAILY_DISMISS_KEY) === todayKey();
  });

  // 切档时清除标记，让新档能重新弹一次
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

  const isMidPhase = phase === "suspended-d2-12";
  const isFinalPhase = phase === "recycling-d13-15";

  const shouldShow =
    isAdminDisabled && (isMidPhase || isFinalPhase) && !dailyDismissed;

  if (!shouldShow) return null;

  const handleClose = () => {
    if (typeof window !== "undefined") {
      localStorage.setItem(DAILY_DISMISS_KEY, todayKey());
    }
    setDailyDismissed(true);
  };

  // 中间档：剩余「无法访问服务」天数（默认 8）
  // 末段档：剩余「数据永久删除」天数（默认 3）
  const daysLeft = recyclingDaysLeft ?? (isFinalPhase ? 3 : 8);

  return (
    <div
      className="fixed z-[10000]"
      style={{ right: 24, bottom: 24, width: 360 }}
      role="alertdialog"
      aria-label="服务已到期通知"
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
              服务已到期
            </p>
            {/* 主标题：两档统一采用「距离域名无法访问仅剩 X 天」口径，仅数字来源不同（中间档默认 8、末段档默认 3） */}
            <p
              className="mt-1 text-[14px] font-medium leading-[22px] m-0"
              style={{ color: "rgba(0, 0, 0, 0.90)", letterSpacing: "0.14px" }}
            >
              距离域名无法访问仅剩
              <span style={{ color: "#C04100" }}>&nbsp;{daysLeft}</span>
              &nbsp;天
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

          {/* 正文：两档文案不同 */}
          <p
            className="mt-3 text-[12px] font-normal leading-[20px] m-0"
            style={{ color: "rgba(0, 0, 0, 0.70)" }}
          >
            {isFinalPhase ? (
              <>
                服务到期已进入最后阶段，所有数据将在
                <span style={{ color: "#C04100", fontWeight: 500 }}>
                  {PERMANENT_DELETE_DATE}
                </span>
                被
                <span style={{ color: "#C04100", fontWeight: 600 }}>
                  永久删除，无法恢复
                </span>
                。 续费需管理员操作，请联系管理员，或先自行备份重要数据。
              </>
            ) : (
              <>
                服务已到期，您将在&nbsp;
                <span style={{ color: "#C04100", fontWeight: 500 }}>
                  {EXPIRE_DATE} 无法继续访问本服务
                </span>
                。
                <br />
                当前您无法再创建新的Agent，但已有Agent仍可继续使用。
                <br />
                管理员续费后可立即恢复服务，如有任何疑问请联系管理员。
              </>
            )}
          </p>

          {/* 底部：右下角胶囊按钮 */}
          <div className="mt-2 flex justify-end">
            <button
              type="button"
              onClick={handleClose}
              className="inline-flex items-center justify-center transition-opacity hover:opacity-90 active:opacity-80"
              style={{
                width: 73,
                height: 26,
                borderRadius: 65,
                background: "#202020",
                color: "rgba(255, 255, 255, 0.90)",
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
