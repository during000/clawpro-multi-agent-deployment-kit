/**
 * TenantSuspendedNoticeBar - 用户端顶部停服通知条
 *
 * 场景：管控台到期停服（consoleStatus === "suspended"）时，在用户端 TopNav
 *   上方展示一条 40px 高的全宽告警条，告知用户账号将在 N 天后停用销毁、
 *   当前无法创建新 agent、已有助手仍可用。
 *
 * 视觉规范（对齐 Figma 440_46567）：
 *   - 容器：宽 100%、高 40px、背景 #FCE8E8（--color-bg-error-lighten-default）
 *   - 内容居中：AlertCircle 图标 + 文字，横向 gap 8px
 *   - 图标：16×16，黑色
 *   - 文字：12px / PingFang SC / regular；数字加粗 semibold + 颜色 #C04100
 *
 * 触发条件：consoleStatus === "suspended"（stopped-d1 / d2-12 / d13-15）
 *   - 剩余天数取 recyclingDaysLeft，未设置时兜底为 15
 *   - 预警 / 临期阶段（active + 快到期）不在此组件覆盖范围内
 */
import { AlertCircle } from "lucide-react";
import { useServiceStatus } from "@/contexts/ServiceStatusContext";

export default function TenantSuspendedNoticeBar() {
  const { isAdminDisabled, recyclingDaysLeft } = useServiceStatus();

  if (!isAdminDisabled) return null;

  // 回收站剩余天数：默认为 15（刚停服）
  const daysLeft = recyclingDaysLeft ?? 15;

  return (
    <div
      className="w-full h-10 flex items-center justify-center gap-2 px-6"
      style={{ background: "#FCE8E8" }}
      role="alert"
      data-billing-exempt
    >
      <AlertCircle
        className="w-4 h-4 shrink-0"
        style={{ color: "#000" }}
        aria-hidden="true"
      />
      <p className="text-xs leading-5 text-black m-0 whitespace-nowrap">
        服务已到期，您的服务将在&nbsp;
        <span className="font-semibold" style={{ color: "#C04100" }}>
          {daysLeft}
        </span>
        &nbsp;天后无法访问。当前您无法再创建新的Agent，但已有Agent仍可继续使用。如有疑问请联系管理员
      </p>
    </div>
  );
}
