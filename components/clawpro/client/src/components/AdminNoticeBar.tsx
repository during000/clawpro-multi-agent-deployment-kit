/**
 * AdminNoticeBar - 管控端顶部常驻通知条
 * Design: 「流动蓝图」Fluid Blueprint
 * - 三类通知：基础配置告警（橙色）、腾讯云配额告警（橙色）、产品动态（蓝色）
 * - 支持自动轮播（5s）+ 手动左右切换
 * - 只有 1 条通知时隐藏切换按钮
 * - 位于管控端独立滚动内容区上方，告警条不透明，下方内容滚动时不会视觉穿透
 * - 跳转链接紧跟在通知文字末尾
 * - 产品动态图标使用星星符号
 */
import { useCallback, useEffect, useState } from "react";
import { Link } from "wouter";
import { ExternalLink } from "lucide-react";
import { AdminNoticeAlert, type AdminNoticeAlertType } from "@/components/ui/admin-notice-alert";
import { useAdminMode } from "@/contexts/AdminModeContext";
import { useServiceStatus } from "@/contexts/ServiceStatusContext";
import AdminDisabledOverlay from "@/components/AdminDisabledOverlay";
import WarningExpireModal from "@/components/WarningExpireModal";
import WarningExpireFloat from "@/components/WarningExpireFloat";

// ─── 基础配置项完成状态（与 BasicInfo.tsx 保持一致） ──────────────────────
// 说明：
//   - custom 模式：8 项（步骤 1-8）
//   - unified 模式：在原第 3 项「导入企业用户」之后插入「设置用户登录方式」作为第 4 步，
//                   原第 4-8 项顺延为第 5-9 步，共 9 项。
const STEP_STATUS_BASE: Record<number, { label: string; done: boolean }> = {
  1: { label: "设置平台名称与品牌", done: true },
  2: { label: "配置用户默认配额", done: true },
  3: { label: "导入企业用户", done: false },
  4: { label: "配置至少一个模型", done: true },
  5: { label: "配置至少一个通道", done: false },
  6: { label: "配置至少一个镜像", done: true },
  7: { label: "配置私有网络", done: true },
  8: { label: "配置安全组", done: false },
};

// unified 模式下第 4 步：设置用户登录方式
const UNIFIED_LOGIN_STEP: { label: string; done: boolean } = {
  label: "设置用户登录方式",
  done: false,
};

function buildStepStatus(isUnified: boolean): Record<number, { label: string; done: boolean }> {
  if (!isUnified) return STEP_STATUS_BASE;
  // 顺延：1,2,3 保留 → 4 = 新增登录方式 → 5..9 = 原 4..8
  return {
    1: STEP_STATUS_BASE[1],
    2: STEP_STATUS_BASE[2],
    3: STEP_STATUS_BASE[3],
    4: UNIFIED_LOGIN_STEP,
    5: STEP_STATUS_BASE[4],
    6: STEP_STATUS_BASE[5],
    7: STEP_STATUS_BASE[6],
    8: STEP_STATUS_BASE[7],
    9: STEP_STATUS_BASE[8],
  };
}

// ─── 腾讯云配额问题 mock 数据 ─────────────────────────────────────────────────
const QUOTA_ALERTS = [
  {
    id: "vpc",
    message: "私有网络（VPC）配额已耗尽，将影响用户端云设备的正常创建与使用。",
    link: "https://console.cloud.tencent.com/workorder/category",
  },
  {
    id: "ai2",
    message: "云服务器 Ai2 机型购买配额已耗尽，将影响用户端 AI 云设备的正常分配。",
    link: "https://console.cloud.tencent.com/workorder/category",
  },
];

// ─── 产品动态 mock 数据 ───────────────────────────────────────────────────────
const PRODUCT_NEWS = [
  {
    id: "news1",
    message: "OpenClaw v2.4.0 已发布：记忆管理功能上线，支持 Pro / Free 版本切换，Pro 版提供长期记忆存储与跨会话召回能力。",
  },
  {
    id: "news2",
    message: "OpenClaw v2.3.0 已发布：技能配置全面升级，支持公共技能库浏览、收藏与批量分发至指定用户或全体成员。",
  },
];

interface NoticeItem {
  id: string;
  type: AdminNoticeAlertType;
  message: string;
  action?: {
    label: string;
    href: string;
    external?: boolean;
  };
}

// ─── 构建通知列表 ─────────────────────────────────────────────────────────────
function buildNotices(stepStatus: Record<number, { label: string; done: boolean }>): NoticeItem[] {
  const notices: NoticeItem[] = [];

  // 1. 基础配置未完成
  const incompleteSteps = Object.values(stepStatus).filter((s) => !s.done);
  if (incompleteSteps.length > 0) {
    const names = incompleteSteps.map((s) => s.label).join("、");
    notices.push({
      id: "basic-config",
      type: "pending-config",
      message: `有 ${incompleteSteps.length} 项基础配置未完成（${names}），未完成配置将影响用户端的正常使用，`,
      action: {
        label: "前往基础信息配置处理",
        href: "/admin/basic-info",
        external: false,
      },
    });
  }

  // 2. 腾讯云配额问题
  for (const alert of QUOTA_ALERTS) {
    notices.push({
      id: `quota-${alert.id}`,
      type: "resource-alert",
      message: alert.message.replace("。", "，"),
      action: {
        label: "前往腾讯云控制台提交工单",
        href: alert.link,
        external: true,
      },
    });
  }

  // 3. 产品动态
  for (const news of PRODUCT_NEWS) {
    notices.push({
      id: news.id,
      type: "product-news",
      message: news.message,
    });
  }

  return notices;
}

const AUTO_PLAY_INTERVAL = 5000;

export function getAdminNotices(isUnified: boolean): NoticeItem[] {
  return buildNotices(buildStepStatus(isUnified));
}

/** 腾讯云费用中心续费管理页地址 */
const RENEW_URL = "https://console.cloud.tencent.com/account/renewal";

function AdminNoticePrevIcon() {
  return (
    <svg width="6" height="10" viewBox="0 0 6 10" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M4.59766 1.06067L1.06216 4.59619L4.59766 8.13169" stroke="currentColor" strokeWidth="1.5" strokeLinecap="square" />
    </svg>
  );
}

function AdminNoticeNextIcon() {
  return (
    <svg width="6" height="10" viewBox="0 0 6 10" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M1.0625 1.06067L4.598 4.59619L1.0625 8.13169" stroke="currentColor" strokeWidth="1.5" strokeLinecap="square" />
    </svg>
  );
}

export default function AdminNoticeBar() {
  const { isUnified } = useAdminMode();
  const { consoleStatus, isAdminDisabled, isInRecycling, recyclingDaysLeft } = useServiceStatus();

  // [004] 每次渲染都重算通知列表，以便存量企业 ack 状态变化时能即时从通知条消失
  const STEP_STATUS = buildStepStatus(isUnified);
  const NOTICES = buildNotices(STEP_STATUS);
  const [current, setCurrent] = useState(0);
  const [paused, setPaused] = useState(false);

  const total = NOTICES.length;

  const goNext = useCallback(() => {
    setCurrent((prev) => (prev + 1) % total);
  }, [total]);

  const goPrev = useCallback(() => {
    setCurrent((prev) => (prev - 1 + total) % total);
  }, [total]);

  useEffect(() => {
    if (total > 0 && current >= total) {
      setCurrent(total - 1);
    }
  }, [current, total]);

  // 自动轮播
  useEffect(() => {
    if (total <= 1 || paused) return;
    const timer = setInterval(goNext, AUTO_PLAY_INTERVAL);
    return () => clearInterval(timer);
  }, [total, paused, goNext]);

  // ─── 停服告警条（始终展示，不受"下次不再提醒"影响） ─────────────────────────
  if (isAdminDisabled) {
    const suspendedMessage = isInRecycling
      ? `管控台将在 ${recyclingDaysLeft} 天后永久删除，请尽快续费，`
      : "管控台已到期，仅支持查看，";

    return (
      <>
        {/* 停服弹窗（复用 D7 橙色版 WarningExpireModal，替代旧 ServiceSuspendedModal）—— Portal，独立堆叠层 */}
        <WarningExpireModal />
        {/* 停服 D2-D7、D9-D12 左下角红色浮窗 —— 挂在 sticky 容器外，避免被 z-20 stacking context 压制 */}
        <WarningExpireFloat />
        {/* 全局禁用层（无 DOM 副作用，返回 null） */}
        <AdminDisabledOverlay />
        {/* 顶部停服告警条 —— sticky 父容器保留，包裹通知条本身 */}
        <div className="w-full sticky top-0 z-20" data-billing-exempt>
          <div className="w-full min-w-[960px] max-w-[1600px] mx-auto px-10 pt-4 pb-2">
            <AdminNoticeAlert type="service-suspended">
              <span>{suspendedMessage}</span>
              <a
                href={RENEW_URL}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-0.5 text-[var(--text-emphasis)] underline underline-offset-2 whitespace-nowrap hover:opacity-80 transition-opacity"
                data-billing-exempt
              >
                立即续费
                <ExternalLink className="w-3 h-3" />
              </a>
            </AdminNoticeAlert>
          </div>
        </div>
      </>
    );
  }

  // 无通知条时也要挂载临期预警弹窗 / D2-6 悬浮卡片
  if (total === 0)
    return (
      <>
        <WarningExpireModal />
        <WarningExpireFloat />
      </>
    );

  const notice = NOTICES[current];

  const noticeContent = (
    <>
      <span>{notice.message}</span>
      {notice.action && (
        <>
          {notice.action.external ? (
            <a
              href={notice.action.href}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-0.5 text-[var(--text-emphasis)] underline underline-offset-2 whitespace-nowrap hover:opacity-80 transition-opacity"
            >
              {notice.action.label}
              <ExternalLink className="w-3 h-3" />
            </a>
          ) : (
            <Link href={notice.action.href}>
              <span className="inline text-[var(--text-emphasis)] underline underline-offset-2 whitespace-nowrap cursor-pointer hover:opacity-80 transition-opacity">
                {notice.action.label}
              </span>
            </Link>
          )}
        </>
      )}
    </>
  );

  const renderControls = () => {
    if (total <= 1) return null;

    return (
      <div className="flex h-5 items-center gap-1 text-[var(--text-secondary)]">
        <button
          onClick={goPrev}
          className="inline-flex size-5 items-center justify-center rounded-[2px] text-[var(--text-secondary)] transition-colors hover:bg-black/10 hover:text-[var(--text-emphasis)] active:bg-black/15"
          aria-label="上一条"
        >
          <AdminNoticePrevIcon />
        </button>
        <span className="min-w-[28px] text-center text-xs leading-5 tabular-nums text-[var(--text-secondary)]">
          {current + 1}/{total}
        </span>
        <button
          onClick={goNext}
          className="inline-flex size-5 items-center justify-center rounded-[2px] text-[var(--text-secondary)] transition-colors hover:bg-black/10 hover:text-[var(--text-emphasis)] active:bg-black/15"
          aria-label="下一条"
        >
          <AdminNoticeNextIcon />
        </button>
      </div>
    );
  };

  return (
    <>
      {/* 临期预警弹窗（warning-d1 / critical-d7）+ D2-6 左下角悬浮卡片，自行按 phase 判断展示 */}
      <WarningExpireModal />
      <WarningExpireFloat />
      <div
        className="w-full min-w-[960px] max-w-[1600px] mx-auto px-10 pt-4 pb-2"
        onMouseEnter={() => setPaused(true)}
        onMouseLeave={() => setPaused(false)}
      >
        <AdminNoticeAlert type={notice.type} controls={renderControls()}>
          {noticeContent}
        </AdminNoticeAlert>
      </div>
    </>
  );
}
