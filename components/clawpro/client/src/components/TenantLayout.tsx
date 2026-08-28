/**
 * TenantLayout - 租户端布局
 *
 * Design: 「流动蓝图」Fluid Blueprint
 * - 用户端背景：v5 使用设计稿提供的新背景图片 /tenant_bg.png（客户端背景.3d165716d3）
 *   · 淡蓝渐变 + cover 铺满 + fixed 固定
 *   · 顶部柔和蓝紫色云雾，过渡到底部白色
 * - 顶部固定导航栏 (64px) — 基于可复用的 TopNav 组合（对照 Figma 358:2322 还原）
 * - 主色 #1447E6
 *
 * 顶部导航相关的视觉/交互全部下沉到 `@/components/topnav`，
 * 本文件只关心：
 *   1) 路由 / 角色相关的状态接入（active tab、isAdmin、modelQuotaEnabled、groupMode）
 *   2) 通知数据（mock）的来源
 *   3) UserMenu 内的下拉菜单项业务文案
 */
import { useState, useEffect } from "react";
import { Link, useLocation } from "wouter";
import {
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { KeyRound, LogOut, UserCog, Eye, EyeOff, Shield, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
  DialogFooter,
} from "@/components/ui/dialog";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { BodyText, MetaText } from "@/components/ui/Typography";

import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { toast } from "sonner";
import TenantSuspendedNoticeBar from "@/components/TenantSuspendedNoticeBar";
import TenantExpireModal from "@/components/TenantExpireModal";
import TenantExpireNotice from "@/components/TenantExpireNotice";
import TenantQuotaNotice from "@/components/TenantQuotaNotice";
import { useUserRole } from "@/contexts/UserRoleContext";
import { useAdminNotifications } from "@/lib/adminNotificationStore";
import { useProjectCollaborationAccessAllowed } from "@/lib/tenantExternalAgentAccess";
import {
  TopNav,
  NavDivider,
  CenterTabs,
  NavIconButton,
  SwitchAdminIcon,
  NotificationPanel,
  HelpPanel,
  UserMenu,
  type Notification,
} from "@/components/topnav";
import {
  GuidePointBubble,
  isDismissed,
  markDismissed,
  resolveBehavior,
  trackOnboarding,
  useBubbleQueue,
  useExposure,
} from "@/components/onboarding";
import {
  TENANT_SKILL_SQUARE_PUBLIC_ANALYTICS,
  TENANT_SKILL_SQUARE_PUBLIC_VISITED_EVENT,
  TENANT_SKILL_SQUARE_PUBLIC_POINT_BUBBLE_KEY,
  TENANT_SKILL_SQUARE_PUBLIC_UPDATE,
  clearTenantSkillSquarePublicVisited,
  hasTenantSkillSquarePublicBeenVisited,
  isTenantSkillSquarePublicGuideActive,
  shouldCompleteTenantSkillSquarePublicGuide,
} from "@/lib/tenantSkillSquarePublicOnboarding";

// 中央 Tab 导航（Figma 358:2322 中央 segmented：我的 Agent / 项目协作 / 技能广场 / 模型额度）
const CENTER_NAV_ITEMS = [
  { label: "我的 Agent", value: "/my-openclaw" },
  { label: "项目协作", value: "/project-collaboration" },
  {
    label: "技能广场",
    value: "/skill-square",
    dataGuide: "tenant-skill-square-nav-tab",
  },
  { label: "模型额度", value: "/model-quota" },
];

const TENANT_SKILL_SQUARE_PUBLIC_POINT_BUBBLE_QUEUE_ID =
  "point-bubble:tenant-skill-square-nav-tab";
const TENANT_SKILL_SQUARE_PUBLIC_POINT_BUBBLE_BEHAVIOR = resolveBehavior("point-bubble", {
  showOnce: true,
  maxExposures: 0,
  startsAt: TENANT_SKILL_SQUARE_PUBLIC_UPDATE.releaseDate,
  expiresAt: TENANT_SKILL_SQUARE_PUBLIC_UPDATE.expiresAt,
});

const CURRENT_USER = "alice@acompany.com";

// ── API Token 状态机 ──────────────────────────────────────

/** Token 弹窗 6 态面板 */
type TokenPanel = "create" | "generated" | "info" | "disabled" | "reset" | "destroy";

interface ApiTokenState {
  exists: boolean;         // 是否已创建 Token
  full: string;            // 完整 token 值
  createdAt: string | null;
  lastCalledAt: string | null;
  disabled: boolean;       // 是否被管理员禁用
}

function generateMockToken(): string {
  const hex = (len: number) => {
    let out = "";
    const chars = "0123456789abcdef";
    for (let i = 0; i < len; i++) out += chars[Math.floor(Math.random() * 16)];
    return out;
  };
  return `hk-${hex(32)}`;
}

function maskToken(t: string): string {
  if (!t) return "";
  return t.slice(0, 7) + "****" + t.slice(-4);
}

function formatDate(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

/** Demo 初始态：已有 Token（信息态） */
const INITIAL_TOKEN_STATE: ApiTokenState = {
  exists: true,
  full: "hk-fdf73e4a91b2c05f8d7e6a3b9c1d4e5f",
  createdAt: "2026/5/7 19:20:03",
  lastCalledAt: null,
  disabled: false,
};

// ==================== Mock 通知 ====================

const MOCK_NOTIFICATIONS: Notification[] = [
  // ==================== 组织变更 / 迁移 / 移交 通知（A/B/C 三组） ====================
  // A 组：组织变更（被动接收，全部 notice，按 Agent 维度逐条推送，A5 为同组织聚合通知）
  // A1 管理员将用户的某个 Agent 迁移到新组织/回退为未分配组织
  { id: "og-a1", message: "由于您已不在原组织「后端研发组」，管理员已将您在原组织下创建的 Agent「API 调试助手」迁移至「平台与基础架构部」，平台策略会立即应用新组织配置（包括您可创建的 Agent 数量上限、您的单用户 Tokens 上限、功能权限等），其他已配置项保留不变。", timestamp: "2026-06-30 16:40", category: "notice", read: false },
  // A2 管理员将用户的某个 Agent 移交给另一用户
  { id: "og-a2", message: "由于您已不在原组织「后端研发组」，管理员已将您在原组织下创建的 Agent「客户反馈整理」移交给 张同学(zhangmou)，该 Agent 不再归属于您。", timestamp: "2026-06-30 15:22", category: "notice", read: false },
  // A3 管理员允许用户自行处理
  { id: "og-a3", message: "由于您已不在原组织「数据分析小组」，您在原组织下创建的 Agent「数据清洗助手」需要您选择迁移至新组织或移交给原组织其他用户，请前往「我的 Agent」处理。", timestamp: "2026-06-29 18:05", category: "notice", read: false },
  // A4 管理员将用户的 Agent 保留关机
  { id: "og-a4", message: "由于您已不在原组织「产品中心」，管理员已将您在原组织下创建的 Agent「测试报告助手」保留和关机，如需恢复开机使用请联系管理员。", timestamp: "2026-06-29 11:18", category: "notice", read: true },
  // A5 管理员改变上级组织（同一组织下的 Agent 聚合为一条通知）
  { id: "og-a5", message: "由于您原组织的上级组织发生变更，变为新组织「产品二部/计算产品中心」，您在原组织下创建的 Agent「需求分析助手」、「日报生成器」已自动迁移至新组织。", timestamp: "2026-06-28 10:12", category: "notice", read: true },

  // B 组：我发起的「迁移到新组织」/「回退为未分配组织」
  { id: "og-b1", message: "您的 Agent「API 调试助手」已发起迁移至「平台与基础架构部」，迁移期间 Agent 保持关机状态。", timestamp: "2026-06-30 14:05", category: "notice", read: false },
  { id: "og-b2", message: "您的 Agent「API 调试助手」已迁移至「平台与基础架构部」，平台策略已应用新组织配置（包括您可创建的 Agent 数量上限、您的单用户 Tokens 上限、功能权限等），其他已配置项保留不变，已为您自动开机。", timestamp: "2026-06-30 14:07", category: "success", read: false },
  { id: "og-b3", message: "您的 Agent「数据清洗助手」迁移至「未分配组织」失败，请稍后在「我的 Agent」重试。", timestamp: "2026-06-26 16:20", category: "failure", read: true },

  // C1 组：我作为「移交发起方」
  { id: "og-c1-1", message: "您已向 李同学(lixue) 发起 Agent「日报生成器」的移交，待对方确认接收，移交期间 Agent 保持关机状态。", timestamp: "2026-06-30 11:25", category: "notice", read: false },
  { id: "og-c1-2", message: "李同学(lixue) 已确认接收 Agent「周报生成器」，移交成功，该 Agent 已转移到对方的 Agent 列表。", timestamp: "2026-06-26 09:42", category: "success", read: true },
  { id: "og-c1-3", message: "您的 Agent「监控告警助手」移交至 陈同学(chenli) 失败，移交已自动取消，请在「我的 Agent」中继续处理。", timestamp: "2026-06-25 10:00", category: "failure", read: true },
  { id: "og-c1-4", message: "王同学(wangjie) 已拒绝接收 Agent「线上巡检助手」，移交失败，请前往「我的 Agent」继续处理。", timestamp: "2026-06-25 15:10", category: "failure", read: true },
  { id: "og-c1-5", message: "您已取消 Agent「客户答疑助手」对 赵同学(zhaoming) 的移交，请前往「我的 Agent」继续处理。", timestamp: "2026-06-24 17:55", category: "notice", read: true },

  // C2 组：我作为「移交接收方」
  { id: "og-c2-1", message: "周同学(zhoufan) 向您移交了 Agent「营销数据分析」，请前往「我的 Agent」选择确认接收或拒绝。", timestamp: "2026-06-30 09:18", category: "notice", read: false },
  { id: "og-c2-2", message: "您已确认接收来自 孙同学(sunqi) 的 Agent「合同审阅助手」，移交成功，已配置项保留不变，已为您自动开机。", timestamp: "2026-06-22 16:02", category: "success", read: true },
  { id: "og-c2-3", message: "吴同学(wuyan) 向您移交的 Agent「费用报销助手」，移交失败，如仍有移交需要，待发起方重新操作。", timestamp: "2026-06-21 18:20", category: "failure", read: true },
  { id: "og-c2-4", message: "您已拒绝接收来自 钱同学(qianwei) 的 Agent「投诉处理助手」。", timestamp: "2026-06-21 11:40", category: "notice", read: true },
  { id: "og-c2-5", message: "孙同学(sunqi) 已取消向您移交 Agent「采购对账助手」。", timestamp: "2026-06-20 14:25", category: "notice", read: true },

  // ==================== 原有平台运行类通知 ====================
  { id: "n1", message: "『Alice的工作助手』TAT 执行命令错误：脚本返回非零退出码 (exit code 1)", timestamp: "2026-03-26 11:05", category: "failure", read: false },
  { id: "n2", message: "『Noah的分析助手』命令执行超时，已自动终止（超时阈值 60s）", timestamp: "2026-03-26 10:42", category: "failure", read: false },
  { id: "n3", message: "『Bob的数据分析』重启失败，实例状态异常，请联系管理员", timestamp: "2026-03-26 09:30", category: "failure", read: false },
  { id: "n4", message: "『Eve的编程助手』TAT Agent 离线，命令下发失败", timestamp: "2026-03-25 17:15", category: "failure", read: false },
  { id: "n5", message: "『Alice的工作助手』API 密钥存在泄露风险，请立即轮换", timestamp: "2026-03-25 14:00", category: "failure", read: false },
  { id: "n6", message: "检测到异常登录行为：账号 bob@bcompany.com 于境外 IP 登录，请确认", timestamp: "2026-03-24 08:55", category: "failure", read: false },
  { id: "n7", message: "『Alice的工作助手』已成功删除", timestamp: "2026-03-23 15:30", category: "success", read: false },
  { id: "n8", message: "『Noah的分析助手』创建成功，已进入运行状态", timestamp: "2026-03-22 10:10", category: "success", read: false },
  { id: "n9", message: "『Bob的数据分析』配置更新成功", timestamp: "2026-03-21 09:00", category: "success", read: false },
  { id: "n10", message: "平台版本已更新至 v2.4.0，新增多模型切换与指令库功能，点击查看更新日志", timestamp: "2026-03-20 09:00", category: "notice", read: false },
];

// [004] 独立化升级完成消息（仅对"兼具管理员身份"的用户端账号推送）
const MIGRATION_NOTIFICATION: Notification = {
  id: "sg-migration-done",
  message: "ClawPro 安全组独立化升级已完成，原规则与绑定 Agent 已迁移至 ClawPro-Default",
  timestamp: "2026-05-05 15:00",
  category: "notice",
  read: false,
  actionHref: "/admin/security-group",
  actionLabel: "前往查看",
};

function TenantSkillSquarePublicPointBubble({ pathname }: { pathname: string }) {
  const [dismissed, setDismissed] = useState(() =>
    isDismissed(
      TENANT_SKILL_SQUARE_PUBLIC_POINT_BUBBLE_KEY,
      TENANT_SKILL_SQUARE_PUBLIC_POINT_BUBBLE_BEHAVIOR,
    ),
  );
  const [hasVisitedPublicSkills, setHasVisitedPublicSkills] = useState(
    hasTenantSkillSquarePublicBeenVisited,
  );
  const [position, setPosition] = useState<{ left: number; top: number } | null>(null);
  const shouldComplete = shouldCompleteTenantSkillSquarePublicGuide(
    hasVisitedPublicSkills,
    pathname,
  );
  const open = isTenantSkillSquarePublicGuideActive() && !dismissed && !shouldComplete;

  useEffect(() => {
    const recordPublicSkillsVisit = () => {
      if (!open) return;
      setHasVisitedPublicSkills(true);
      trackOnboarding("onboarding_click", {
        ...TENANT_SKILL_SQUARE_PUBLIC_ANALYTICS,
        component: "point-bubble",
        target: "public-skill-tab",
      });
    };
    window.addEventListener(TENANT_SKILL_SQUARE_PUBLIC_VISITED_EVENT, recordPublicSkillsVisit);
    return () => {
      window.removeEventListener(TENANT_SKILL_SQUARE_PUBLIC_VISITED_EVENT, recordPublicSkillsVisit);
    };
  }, [open]);

  useEffect(() => {
    if (!shouldComplete || dismissed) return;

    markDismissed(TENANT_SKILL_SQUARE_PUBLIC_POINT_BUBBLE_KEY);
    clearTenantSkillSquarePublicVisited();
    setDismissed(true);
  }, [dismissed, shouldComplete]);

  useEffect(() => {
    if (!open) {
      setPosition(null);
      return;
    }

    let animationFrame: number | null = null;
    const measure = () => {
      animationFrame = null;
      const target = document.querySelector<HTMLElement>(
        '[data-guide="tenant-skill-square-nav-tab"]',
      );
      if (!target) {
        setPosition(null);
        return;
      }

      const rect = target.getBoundingClientRect();
      setPosition({
        left: rect.left + rect.width / 2,
        top: rect.bottom,
      });
    };
    const scheduleMeasure = () => {
      if (animationFrame !== null) cancelAnimationFrame(animationFrame);
      animationFrame = requestAnimationFrame(measure);
    };

    scheduleMeasure();
    window.addEventListener("resize", scheduleMeasure);
    window.addEventListener("scroll", scheduleMeasure, true);

    return () => {
      window.removeEventListener("resize", scheduleMeasure);
      window.removeEventListener("scroll", scheduleMeasure, true);
      if (animationFrame !== null) cancelAnimationFrame(animationFrame);
    };
  }, [open]);

  const canShow = useBubbleQueue(
    TENANT_SKILL_SQUARE_PUBLIC_POINT_BUBBLE_QUEUE_ID,
    open && position !== null,
  );
  const visible = open && position !== null && canShow;

  useExposure(visible, {
    ...TENANT_SKILL_SQUARE_PUBLIC_ANALYTICS,
    component: "point-bubble",
  });

  const handleClose = () => {
    markDismissed(TENANT_SKILL_SQUARE_PUBLIC_POINT_BUBBLE_KEY);
    clearTenantSkillSquarePublicVisited();
    setDismissed(true);
    trackOnboarding("onboarding_dismiss", {
      ...TENANT_SKILL_SQUARE_PUBLIC_ANALYTICS,
      component: "point-bubble",
    });
  };

  if (!visible || !position) return null;

  return (
    <GuidePointBubble
      open
      onClose={handleClose}
      title="公共技能上线"
      description="可在公共技能页面，浏览精选技能并按需安装到你的 Agent。"
      variant="light"
      contentVariant="text-only"
      placement="bottom"
      endpoint="tenant"
      style={{
        position: "fixed",
        left: position.left,
        top: position.top,
        marginTop: "calc(var(--spacing) * -1)",
        transform: "translateX(-50%)",
      }}
    />
  );
}

// ==================== TenantLayout ====================

export default function TenantLayout({ children }: { children: React.ReactNode }) {
  const [location, navigate] = useLocation();
  const { isAdmin, toggleRole } = useUserRole();
  const projectCollaborationEnabled = useProjectCollaborationAccessAllowed();

  // 多组织模式
  const [groupMode, setGroupMode] = useState<"normal" | "multi-group">(() => {
    return (localStorage.getItem("openclaw_group_mode") as "normal" | "multi-group") || "normal";
  });
  const handleGroupModeChange = (mode: "normal" | "multi-group") => {
    setGroupMode(mode);
    localStorage.setItem("openclaw_group_mode", mode);
    window.dispatchEvent(new StorageEvent("storage", { key: "openclaw_group_mode", newValue: mode }));
  };

  // 重置密码弹窗
  const [resetPwdOpen, setResetPwdOpen] = useState(false);

  // API Token 弹窗 — 5 态状态机
  const [showTokenDialog, setShowTokenDialog] = useState(false);
  const [tokenPanel, setTokenPanel] = useState<TokenPanel>("info");
  const [tokenState, setTokenState] = useState<ApiTokenState>({ ...INITIAL_TOKEN_STATE });
  const [copyLabel, setCopyLabel] = useState("复制");

  /** 从用户菜单点击"API Token"入口 */
  const openTokenDialog = () => {
    setTokenPanel(tokenState.disabled ? "disabled" : tokenState.exists ? "info" : "create");
    setShowTokenDialog(true);
  };

  const closeTokenDialog = () => {
    setShowTokenDialog(false);
    setCopyLabel("复制");
  };

  /** 创建 Token → 已生成态 */
  const handleCreateToken = () => {
    const newToken = generateMockToken();
    setTokenState({ exists: true, full: newToken, createdAt: formatDate(new Date()), lastCalledAt: null, disabled: false });
    setTokenPanel("generated");
  };

  /** 确认重置 → 已生成态（新 Token） */
  const handleResetToken = () => {
    const newToken = generateMockToken();
    setTokenState({ ...tokenState, full: newToken, createdAt: formatDate(new Date()), lastCalledAt: null });
    setTokenPanel("generated");
  };

  /** 确认销毁 → 关闭弹窗，回到创建态 */
  const handleDestroyToken = () => {
    setTokenState({ exists: false, full: "", createdAt: null, lastCalledAt: null, disabled: false });
    closeTokenDialog();
    toast.success("API Token 已销毁");
  };

  /** 复制 Token */
  const handleCopyToken = () => {
    if (!tokenState.full) return;
    navigator.clipboard.writeText(tokenState.full);
    setCopyLabel("已复制");
    setTimeout(() => setCopyLabel("复制"), 1500);
  };
  const [oldPwd, setOldPwd] = useState("");
  const [newPwd, setNewPwd] = useState("");
  const [confirmPwd, setConfirmPwd] = useState("");
  const [showOld, setShowOld] = useState(false);
  const [showNew, setShowNew] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const handleResetPwd = () => {
    if (!oldPwd || !newPwd || !confirmPwd) { toast.error("请填写所有字段"); return; }
    if (newPwd !== confirmPwd) { toast.error("两次输入的新密码不一致"); return; }
    if (newPwd.length < 8) { toast.error("新密码长度不能少于 8 位"); return; }
    toast.success("密码重置成功");
    setResetPwdOpen(false);
    setOldPwd(""); setNewPwd(""); setConfirmPwd("");
  };
  useEffect(() => {
    const handleStorage = (e: StorageEvent) => {
      if (e.key === "openclaw_group_mode") {
        setGroupMode((e.newValue as "normal" | "multi-group") || "normal");
      }
    };
    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, []);

  // 模型额度开关
  const [modelQuotaEnabled, setModelQuotaEnabled] = useState(() => {
    const v = localStorage.getItem("admin_allow_model_quota");
    return v !== null ? v === "true" : true;
  });
  useEffect(() => {
    const handleStorage = (e: StorageEvent) => {
      if (e.key === "admin_allow_model_quota") {
        setModelQuotaEnabled(e.newValue !== null ? e.newValue === "true" : true);
      }
    };
    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, []);

  // 过滤后的中央 Tab
  const visibleCenterNavItems = CENTER_NAV_ITEMS.filter(
    (item) =>
      !(item.value === "/model-quota" && !modelQuotaEnabled) &&
      !(
        item.value === "/project-collaboration" &&
        !projectCollaborationEnabled
      )
  );

  // 给中央 Tab 设置基于"路由前缀"的匹配
  const centerItemsWithMatcher = visibleCenterNavItems.map((item) => ({
    ...item,
    matches: (current: string) => {
      if (item.value === "/my-openclaw") {
        // 详情页路由 /openclaw/:id 和 /openclaw-guide 也属于"我的 Agent"
        return current.startsWith("/my-openclaw") || current.startsWith("/openclaw");
      }
      return current.startsWith(item.value);
    },
  }));

  // 运行时管控端动态推送的通知（如安全组校验失败、自动补规则失败等）
  const extraAdminNotifications = useAdminNotifications();

  // 通知数据（管理员多推一条独立化消息 + 运行时推送的通知都放在最前）
  const notificationData: Notification[] = isAdmin
    ? [...extraAdminNotifications, MIGRATION_NOTIFICATION, ...MOCK_NOTIFICATIONS]
    : [...extraAdminNotifications, ...MOCK_NOTIFICATIONS];

  return (
    <div
      className="min-h-screen"
      style={{
        // v5：新背景图片 — cover 铺满整个视口，固定不随滚动
        backgroundColor: "#FFFFFF",
        backgroundImage: "url(/tenant_bg.png)",
        backgroundSize: "cover",
        backgroundPosition: "center top",
        backgroundRepeat: "no-repeat",
        backgroundAttachment: "fixed",
      }}
    >
      {/* 管控台停服态：顶部 40px 通知条（Figma 440:46567），仅 suspended 时展示 */}
      <TenantSuspendedNoticeBar />

      {/* 管控台停服 D1 / D8 关键日档：用户端 600×456 停服提醒弹窗（每日一次） */}
      <TenantExpireModal />

      {/* 管控台停服中间档（D2-D7 / D9-D12）+ 回收站末段（D13-D15）：右下角 360×173 悬浮通知卡（每日一次） */}
      <TenantExpireNotice />

      {/* 席位欠费（D1 缓冲期 / 正式欠费档均触发）：用户端右下角 360×175「配额不足」悬浮通知卡（每日一次，Figma 466_12404） */}
      <TenantQuotaNotice />

      {/* [Figma 358:2322] Top Navigation 64px：左 Logo + 中央 Tab + 右图标 */}
      <TopNav
        center={
          <CenterTabs
            items={centerItemsWithMatcher}
            activeValue={location}
            onChange={(value) => navigate(value)}
          />
        }
        right={
          <>
            {/* 使用指南 */}
            <HelpPanel />

            <NavDivider />

            {/* 消息中心 */}
            <NotificationPanel notifications={notificationData} isAdmin={isAdmin} />

            <NavDivider />

            {/* 切换管控端：管理员可见 */}
            {isAdmin && (
              <>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Link href="/admin/basic-info" className="flex min-w-0 shrink overflow-hidden">
                      <NavIconButton
                        icon={<SwitchAdminIcon />}
                        label="管控端"
                        pill
                      />
                    </Link>
                  </TooltipTrigger>
                  <TooltipContent side="bottom" sideOffset={6}>
                    前往管控端
                  </TooltipContent>
                </Tooltip>
                <NavDivider />
              </>
            )}

            {/* 用户菜单 */}
            <UserMenu username={CURRENT_USER}>
              <div className="px-3 py-2 border-b border-gray-100">
                <p className="text-xs text-gray-500">当前账号</p>
                <p className="text-sm font-medium text-gray-900 truncate">{CURRENT_USER}</p>
                <span
                  className={`inline-block mt-1 text-xs px-1.5 py-0.5 rounded font-medium ${
                    isAdmin ? "bg-blue-100 text-blue-700" : "bg-gray-100 text-gray-600"
                  }`}
                >
                  {isAdmin ? "管理员" : "普通成员"}
                </span>
              </div>
              {/* 所在组织 */}
              <div className="px-3 py-2 border-b border-gray-100">
                <p className="text-xs text-gray-500 mb-1.5">所在组织</p>
                <div className="flex flex-wrap gap-1">
                  {groupMode === "multi-group" ? (
                    <>
                      <span
                        className="inline-block text-xs px-1.5 py-0.5 rounded-[4px] font-medium bg-white text-[#737373] border border-gray-200"
                      >A公司 / 技术部 / 前端组</span>
                      <span
                        className="inline-block text-xs px-1.5 py-0.5 rounded-[4px] font-medium bg-white text-[#737373] border border-gray-200"
                      >A公司 / 技术部 / AI 组</span>
                      <span
                        className="inline-block text-xs px-1.5 py-0.5 rounded-[4px] font-medium bg-white text-[#737373] border border-gray-200"
                      >前端研发同学</span>
                    </>
                  ) : (
                    <span
                      className="inline-block text-xs px-1.5 py-0.5 rounded-[4px] font-medium bg-white text-[#737373] border border-gray-200"
                    >默认</span>
                  )}
                </div>
              </div>
              {/* 组织模式切换 */}
              <div className="px-3 py-2 border-b border-gray-100">
                <p className="text-xs text-gray-500 mb-1.5">组织模式</p>
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => handleGroupModeChange("normal")}
                    className={`text-xs px-2 py-1 rounded-[4px] font-medium transition-colors ${
                      groupMode === "normal"
                        ? "bg-blue-50 text-blue-700 border border-blue-200"
                        : "bg-white text-[#737373] border border-gray-200 hover:bg-gray-50"
                    }`}
                  >
                    普通
                  </button>
                  <button
                    type="button"
                    onClick={() => handleGroupModeChange("multi-group")}
                    className={`text-xs px-2 py-1 rounded-[4px] font-medium transition-colors ${
                      groupMode === "multi-group"
                        ? "bg-blue-50 text-blue-700 border border-blue-200"
                        : "bg-white text-[#737373] border border-gray-200 hover:bg-gray-50"
                    }`}
                  >
                    多组织
                  </button>
                </div>
              </div>
              <DropdownMenuItem onClick={openTokenDialog}>
                <Shield className="w-4 h-4 mr-2 text-[#0A0A0A]" />
                API Token
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setResetPwdOpen(true)}>
                <KeyRound className="w-4 h-4 mr-2 text-black" />
                重置密码
              </DropdownMenuItem>
              {/* 演示用：切换角色 */}
              <DropdownMenuItem
                onClick={() => {
                  toggleRole();
                  toast.info(`已切换为${isAdmin ? "普通成员" : "管理员"}视角`);
                }}
              >
                <UserCog className="w-4 h-4 mr-2 text-[#0A0A0A]" />
                切换为{isAdmin ? "普通成员" : "管理员"}视角
              </DropdownMenuItem>
              {/* 仅管理员：保留旧版"管理后台"快捷入口 */}
              {isAdmin && (
                <DropdownMenuItem onClick={() => (window.location.href = "/admin/basic-info")}>
                  <SwitchAdminIcon size={16} className="mr-2 text-[#0A0A0A]" />
                  前往管控端
                </DropdownMenuItem>
              )}
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => toast.info("已退出登录")}
                className="text-[#0A0A0A]"
              >
                <LogOut className="w-4 h-4 mr-2 text-[#0A0A0A]" />
                退出登录
              </DropdownMenuItem>
            </UserMenu>
          </>
        }
      />

      <TenantSkillSquarePublicPointBubble pathname={location} />

      {/* Main Content */}
      <main className="min-h-[calc(100vh-64px)]">{children}</main>

      {/* 重置密码弹窗 */}
      <Dialog open={resetPwdOpen} onOpenChange={(open) => { setResetPwdOpen(open); if (!open) { setOldPwd(""); setNewPwd(""); setConfirmPwd(""); setShowOld(false); setShowNew(false); setShowConfirm(false); } }}>
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>重置密码</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6">
            <div className="space-y-4">
              <div className="space-y-1.5">
                <Label className="text-sm text-[var(--text-secondary)]">当前密码</Label>
                <div className="relative">
                  <Input
                    tenant
                    type={showOld ? "text" : "password"}
                    value={oldPwd}
                    onChange={(e) => setOldPwd(e.target.value)}
                    placeholder="请输入当前密码"
                    className="pr-10"
                  />
                  <button type="button" onClick={() => setShowOld(!showOld)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--text-emphasis)]">
                    {showOld ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                  </button>
                </div>
              </div>
              <div className="space-y-1.5">
                <Label className="text-sm text-[var(--text-secondary)]">新密码</Label>
                <div className="relative">
                  <Input
                    tenant
                    type={showNew ? "text" : "password"}
                    value={newPwd}
                    onChange={(e) => setNewPwd(e.target.value)}
                    placeholder="请输入新密码（至少 8 位）"
                    className="pr-10"
                  />
                  <button type="button" onClick={() => setShowNew(!showNew)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--text-emphasis)]">
                    {showNew ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                  </button>
                </div>
              </div>
              <div className="space-y-1.5">
                <Label className="text-sm text-[var(--text-secondary)]">确认新密码</Label>
                <div className="relative">
                  <Input
                    tenant
                    type={showConfirm ? "text" : "password"}
                    value={confirmPwd}
                    onChange={(e) => setConfirmPwd(e.target.value)}
                    placeholder="请再次输入新密码"
                    className="pr-10"
                  />
                  <button type="button" onClick={() => setShowConfirm(!showConfirm)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--text-emphasis)]">
                    {showConfirm ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                  </button>
                </div>
              </div>
            </div>
          </DialogBody>
          <DialogFooter className="gap-3">
            <Button variant="tenant-outline" onClick={() => setResetPwdOpen(false)}>
              取消
            </Button>
            <Button variant="tenant-primary" onClick={handleResetPwd}>
              确认重置
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ============ API Token 弹窗（5 态共用一个遮罩层） ============ */}
      <Dialog open={showTokenDialog} onOpenChange={(open) => { if (!open) closeTokenDialog(); }}>
        <DialogContent>
          {/* ── 状态 1：创建态（无 Token 时） ── */}
          {tokenPanel === "create" && (
            <>
              <DialogHeader>
                <DialogTitle>API Token</DialogTitle>
              </DialogHeader>
              <DialogBody className="px-6 py-0">
                <BodyText tone="secondary" className="leading-[1.7]">
                  创建后可通过 Token 调用 OpenClaw API，实现与第三方系统的集成。
                </BodyText>
              </DialogBody>
              <DialogFooter className="gap-3 pt-6">
                <Button variant="tenant-outline" onClick={closeTokenDialog}>取消</Button>
                <Button variant="tenant-primary" onClick={handleCreateToken}>创建 Token</Button>
              </DialogFooter>
            </>
          )}

          {/* ── 状态 2：已生成态（创建/重置成功后） ── */}
          {tokenPanel === "generated" && (
            <>
              <DialogHeader>
                <DialogTitle>Token 已生成</DialogTitle>
              </DialogHeader>
              <DialogBody className="px-6 py-0">
                <BodyText tone="secondary" className="leading-[1.7] mb-5">
                  请立即复制并妥善保存，此 Token 仅显示一次，关闭后将无法再次查看完整内容
                </BodyText>
                <div className="flex items-center justify-between gap-4 rounded-[var(--radius-lg)] bg-[var(--muted)] px-[18px] py-4 border border-[var(--border)]">
                  <span className="font-mono text-[13.5px] text-[var(--text-emphasis)] break-all">
                    {tokenState.full}
                  </span>
                  <button
                    type="button"
                    onClick={handleCopyToken}
                    className={`inline-flex items-center gap-1.5 flex-none whitespace-nowrap rounded-[var(--radius-lg)] border px-3 py-1.5 text-xs cursor-pointer transition-colors ${
                      copyLabel === "已复制"
                        ? "border-[var(--alert-success-border)] text-[var(--alert-success-foreground)] bg-[var(--alert-success-bg)]"
                        : "border-[var(--border)] text-[var(--text-emphasis)] bg-[var(--card)] hover:bg-[var(--muted)]"
                    }`}
                  >
                    <Copy className="w-3.5 h-3.5" />
                    <span>{copyLabel}</span>
                  </button>
                </div>
              </DialogBody>
              <DialogFooter className="gap-3 pt-6">
                <Button variant="tenant-primary" onClick={closeTokenDialog}>我已保存，关闭</Button>
              </DialogFooter>
            </>
          )}

          {/* ── 状态 3：信息态（已有 Token 时的默认视图） ── */}
          {tokenPanel === "info" && (
            <>
              <DialogHeader>
                <DialogTitle>API Token</DialogTitle>
              </DialogHeader>
              <DialogBody className="px-6 py-0">
                <div className="mb-1">
                  <div className="flex items-center justify-between py-[15px] border-b border-[var(--border)]">
                    <MetaText tone="muted">Token</MetaText>
                    <span className="text-[13px] font-medium font-mono text-[var(--text-emphasis)]">{maskToken(tokenState.full)}</span>
                  </div>
                  <div className="flex items-center justify-between py-[15px] border-b border-[var(--border)]">
                    <MetaText tone="muted">状态</MetaText>
                    <span className="inline-flex items-center gap-1.5 rounded-full bg-[var(--alert-success-bg)] text-[var(--alert-success-foreground)] px-3 py-1 text-xs font-medium">
                      <span className="w-1.5 h-1.5 rounded-full bg-[var(--alert-success-icon)]" />
                      启用中
                    </span>
                  </div>
                  <div className="flex items-center justify-between py-[15px] border-b border-[var(--border)]">
                    <MetaText tone="muted">创建时间</MetaText>
                    <span className="text-[13px] font-medium text-[var(--text-emphasis)]">{tokenState.createdAt}</span>
                  </div>
                  <div className="flex items-center justify-between py-[15px]">
                    <MetaText tone="muted">最近调用</MetaText>
                    <MetaText tone="weak">{tokenState.lastCalledAt ?? "—"}</MetaText>
                  </div>
                </div>
              </DialogBody>
              <DialogFooter className="gap-3 pt-4">
                <Button variant="tenant-outline" onClick={() => setTokenPanel("reset")}>
                  重置 Token
                </Button>
                <Button variant="tenant-destructive" onClick={() => setTokenPanel("destroy")}>
                  销毁 Token
                </Button>
              </DialogFooter>
            </>
          )}

          {/* ── 状态 3b：已禁用态（管理员禁用后的用户端视图） ── */}
          {tokenPanel === "disabled" && (
            <>
              <DialogHeader>
                <DialogTitle>API Token</DialogTitle>
              </DialogHeader>
              <DialogBody className="px-6 py-0">
                <div className="mb-1">
                  <div className="flex items-center justify-between py-[15px] border-b border-[var(--border)]">
                    <MetaText tone="muted">Token</MetaText>
                    <span className="text-[13px] font-medium font-mono text-[var(--text-emphasis)]">{maskToken(tokenState.full)}</span>
                  </div>
                  <div className="flex items-center justify-between py-[15px] border-b border-[var(--border)]">
                    <MetaText tone="muted">状态</MetaText>
                    <span className="inline-flex items-center gap-1.5 rounded-full bg-[var(--alert-error-bg)] text-[var(--alert-error-foreground)] px-3 py-1 text-xs font-medium">
                      <span className="w-1.5 h-1.5 rounded-full bg-[var(--alert-error-icon)]" />
                      已禁用
                    </span>
                  </div>
                  <div className="flex items-center justify-between py-[15px] border-b border-[var(--border)]">
                    <MetaText tone="muted">创建时间</MetaText>
                    <span className="text-[13px] font-medium text-[var(--text-emphasis)]">{tokenState.createdAt}</span>
                  </div>
                  <div className="flex items-center justify-between py-[15px]">
                    <MetaText tone="muted">最近调用</MetaText>
                    <MetaText tone="weak">{tokenState.lastCalledAt ?? "—"}</MetaText>
                  </div>
                </div>
                {/* 管理员禁用提示（warning Alert） */}
                <Alert variant="warning" className="mt-4">
                  <AlertDescription>
                    您的 Token 已被管理员禁用，如需恢复请联系企业管理员
                  </AlertDescription>
                </Alert>
              </DialogBody>
              <DialogFooter className="gap-3 pt-6 justify-end">
                <Button variant="tenant-outline" onClick={closeTokenDialog}>关闭</Button>
              </DialogFooter>
            </>
          )}

          {/* ── 状态 4：重置确认态 ── */}
          {tokenPanel === "reset" && (
            <>
              <DialogHeader>
                <DialogTitle>重置 Token</DialogTitle>
              </DialogHeader>
              <DialogBody className="px-6 py-0">
                <BodyText tone="secondary" className="leading-[1.7]">
                  重置后旧 Token 立即失效，将生成新 Token。请确保已更新所有使用该 Token 的调用方。
                </BodyText>
              </DialogBody>
              <DialogFooter className="gap-3 pt-6">
                <Button variant="tenant-outline" onClick={() => setTokenPanel("info")}>取消</Button>
                <Button variant="tenant-primary" onClick={handleResetToken}>确认重置</Button>
              </DialogFooter>
            </>
          )}

          {/* ── 状态 5：销毁确认态 ── */}
          {tokenPanel === "destroy" && (
            <>
              <DialogHeader>
                <DialogTitle>销毁 Token</DialogTitle>
              </DialogHeader>
              <DialogBody className="px-6 py-0">
                <BodyText tone="secondary" className="leading-[1.7]">
                  销毁后，使用该 Token 的 API 调用将
                  <strong className="text-[var(--text-emphasis)] font-semibold">立即失效</strong>
                  ，且操作不可恢复。
                </BodyText>
              </DialogBody>
              <DialogFooter className="gap-3 pt-6">
                <Button variant="tenant-outline" onClick={() => setTokenPanel("info")}>取消</Button>
                <Button variant="tenant-destructive" onClick={handleDestroyToken}>确认销毁</Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
