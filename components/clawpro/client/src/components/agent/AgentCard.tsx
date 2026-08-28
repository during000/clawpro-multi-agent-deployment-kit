/**
 * AgentCard - 单个 Agent 卡片
 *
 * 严格对齐 Figma node [1077:33986]「卡片」（用户端 0523 修订版）：
 *   - 容器：使用 <TenantCard interactive>（用户端业务卡专用，12px 圆角 + 三状态）
 *           padding 20px、column gap 24px、border-radius 12px、shadow var(--shadow-tenant-card)
 *           ⚠️ padding/gap 0523-2 起由 TenantCard 默认值（padding="default"）承担，
 *              业务侧不再 inline 覆写，便于全用户端业务卡保持一致间距
 *           hover：无描边 + 加强阴影 + 微抬 0.5px（normal→hover 过渡，由 TenantCard 自动处理）
 *   - 头部行：左 (48 头像 + gap 16 + 文字 column gap 4)；
 *            右上角【只保留三点菜单】（[Figma 1077-33986] 删除外露刷新按钮，刷新仍在三点菜单内）
 *   - 元信息组：column gap 4
 *     · 第 1 行：[角色徽章 #FFF→#F9FBFC 边 #DAE0E9 R2 padding 2x6] | 类型：xxx | ID：xxx [复制]
 *     · 第 2 行：组织：xxx
 *   - 底部行：左 创建时间 (#737373)；
 *            右【设置】+【对话】两个带文字按钮（[Figma 1077-33986]）
 *     · 两个按钮统一使用 `tenant-outline-r20` 变体（radius 20px，对齐 Figma 1077:33986）
 *   - 无底部分隔线（依靠 column gap-24 留白即可）
 *
 * 历史：
 *   - v1：严格对齐 Figma 358:2387，使用 SurfaceCard（实际 4px 圆角）
 *   - v2 ([Figma 1077-33986])：简化操作区 + 改文案 + 按钮圆角 20px
 *   - v3 (0523)：改用 <TenantCard interactive>，把卡片本体圆角从 4px 修正为 12px，
 *                对齐 SKILL-TENANT.md §5 用户端卡片规范
 *
 * 所有业务逻辑（删除/重启/重装/移除角色/重试/打开终端/打开面板/对话）通过 props 暴露。
 */
import { useState, useRef, useLayoutEffect } from "react";
import { Link, useLocation } from "wouter";
import {
  MoreVertical,
  Settings,
  RefreshCw,
  Trash2,
  RotateCcw,
  HardDriveDownload,
  Terminal,
UserMinus,
Copy,
  Check,
  MessageSquare,
  Clock3,
  Pencil,
  UserCog,
  PowerOff,
  Power,
  ArrowUpCircle,
  Share2,
  ArrowLeftRight,
  AlertCircle,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { TenantCard } from "@/components/ui/Surface";
import { StatusTag } from "@/components/ui/status-tag";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { toast } from "sonner";
import { AgentAvatar } from "./AgentAvatar";
import {
  StatusBadge,
  STATUS_DISABLED,
  STATUS_GRAY_AVATAR,
  STATUS_DIMMED_TEXT,
  type OpenClawStatus,
} from "./StatusBadge";
import {
  GroupChangeBadge,
  TransferReceiveOverlay,
} from "@/pages/tenant/GroupChangeComponents";
import { compareVersion } from "@/lib/upgradePushStore";
import { isReminderEnabled } from "@/lib/updateReminderStore";
import type { AgentRoleSlot } from "@/lib/mockData";

export interface AgentCardItem {
  id: string;
  instanceId: string;
  name: string;
  status: OpenClawStatus;
  createdAt: string;
  agentType?: "openclaw" | "hermes" | "lightclawace" | "localagent";
  model?: string;
  localProduct?: "Claude" | "CodeBuddy" | "WorkBuddy" | "Codex";
  localResourceSyncStatus?: "syncing" | "synced";
  /** 外部 Agent 最近一次向 Hatchery 同步信息的时间 */
  lastReportedAt?: string;
  /** 当前运行的 Agent 版本（如 "2026.4.23"），用于与目标镜像版本对比判断更新状态 */
  version?: string;
  roleName?: string;
  /** 实例下角色数量，>1 时显示「多角色」标签；若无 roles[] 精确数据则切换仅影响首个角色（兼容旧数据兜底） */
  roleCount?: number;
  /**
   * 实例下的角色 slot 列表（main Agent + sub Agent）。当长度 > 1 时，角色标签 / 三点菜单的
   * 「切换角色」会先展示 slot 选择层，定位到具体某个角色后再进入身份替换弹窗；
   * 缺省或仅 1 项时保持旧的整实例覆盖语义，零回归。
   */
  roles?: AgentRoleSlot[];
  /** 分组允许的角色名称列表，用于限制切换角色时的可选范围（通用助手始终可用） */
  allowedRoleNames?: string[];
  groupId?: string;
  groupName?: string;
  memoryStatus?: "none" | "free" | "pro";
  billingMode?: "subscription" | "payg";
  // 组织变更处理状态（多组织模式）
  // archived：管理员已保留并关机，无需用户处理，卡片保持常规关机态，仅保留「您已不在该组织」提示
  groupChangeStatus?: "pending" | "pendingConfirm" | "transferring" | "rejected" | "expired" | "migrating" | "archived";
  groupChangeOriginalGroup?: string;
  groupChangeTransferTarget?: string;
  /**
   * 仅可移交，不可迁移（如原组织已被删除等场景）：为 true 时隐藏「迁移到新组织」按钮，
   * 只保留「移交给其他用户」按钮。
   */
  groupChangeTransferOnly?: boolean;
  /**
   * 迁移到新组织的执行阶段（点击「迁移到新组织」确认后驱动）：
   *   - migrating：迁移组织中（实例保持已关机，迁移/移交按钮隐藏，回退为置灰「设置」）
   *   - success ：迁移成功（随后已关机→加载中→运行中）
   *   - fail    ：迁移失败（保持已关机，迁移/移交按钮重新出现）
   */
  migrationPhase?: "migrating" | "success" | "fail";
  /**
   * 移交给其他用户的执行阶段（点击「移交给其他用户」确认后驱动）：
   *   - pendingConfirm：移交待对方确认（实例保持已关机，迁移/移交按钮隐藏→置灰「设置」，旁带「取消」按钮）
   *   - transferring  ：移交中（对方已接收，处理中）
   *   - done          ：移交完成（随后已关机→加载中→运行中，并从列表移除）
   *   - rejected      ：对方拒绝移交（保持已关机，迁移/移交按钮重新出现）
   *   - failed        ：对方已接收，但移交处理最终失败（保持已关机，迁移/移交按钮重新出现）
   */
  transferPhase?: "pendingConfirm" | "transferring" | "done" | "rejected" | "failed";
  // 别人移交给我的实例
  incomingTransfer?: { fromUser: string; instanceName: string };
  // ===== 共享范围 =====
  /** 共享范围类型：private=仅自己，shared=共享到分组或个人 */
  shareScope?: "private" | "shared";
  /** 创建者（共享实例展示用） */
  creator?: string;
  /** 共享的分组 ID 列表 */
  shareGroupIds?: string[];
  /**
   * 共享展示用——**整组全选**的分组名称列表。
   * 含义：一个组只要被「整组覆盖」（显式勾选了整组，或该组所有成员都被勾选），
   * 就归并为组名展示，而不是把成员逐个列出来。
   * 由数据源按共享树同款规则（isGroupFullyCovered）派生计算。
   */
  shareGroupNames?: string[];
  /** 共享的用户 ID 列表 */
  shareUserIds?: string[];
  /**
   * 共享展示用——**未被整组覆盖**的散选成员名称列表。
   * 含义：成员所在的组没有被整组全选，才作为个人单独展示。
   * 由数据源派生计算（已剔除归入整组的成员）。
   */
  shareUserNames?: string[];
}

export interface AgentCardCallbacks {
  onClickCard: (claw: AgentCardItem) => void;
  onRefresh: (e: React.MouseEvent, id: string, name: string) => void;
  onRestart: (claw: AgentCardItem) => void;
  onReinstall: (claw: AgentCardItem) => void;
  onDelete: (claw: AgentCardItem) => void;
  /**
   * 切换角色；始终只传整个 Agent，不预先定位具体 slot——多角色实例「选择要切换的角色位」
   * 这一步统一放进「切换角色」Dialog 内部完成（与「选新角色」同屏两组 Pill 选择器），
   * 卡片侧（角色标签 / 三点菜单）只负责打开 Dialog，不再承载任何自定义下拉/子菜单。
   */
  onSwitchRole: (claw: AgentCardItem) => void;
  onRetry: (id: string, name: string) => void;
  /** 点击对话按钮进入对话视图 */
  onChat: (claw: AgentCardItem) => void;
  /** 打开终端权限（综合双模式逻辑） */
  canOpenTerminal: (claw: AgentCardItem) => boolean;
  /** 当前是否在刷新中 */
  refreshing: boolean;
  /** 多组织模式开关，影响"组织"行展示 */
  groupMode: "normal" | "multi-group";
  /** 重命名（running/shutdown 可用），不传则不展示该菜单项 */
  onRename?: (claw: AgentCardItem) => void;
  /** 关机（running 时），不传则不展示 */
  onShutdown?: (claw: AgentCardItem) => void;
  /** 开机（shutdown 时），不传则不展示 */
  onPowerOn?: (claw: AgentCardItem) => void;
  /** 组织变更：迁移到新组织 */
  onMigrate?: (claw: AgentCardItem) => void;
  /** 组织变更：移交 */
  onTransfer?: (claw: AgentCardItem) => void;
  /** 外部 Agent 重新接入指引 */
  onLocalReconnect?: (claw: AgentCardItem) => void;
  /** 接收他人移交：确认接收 */
  onAcceptTransfer?: (claw: AgentCardItem) => void;
  /** 接收他人移交：拒绝接收 */
  onRejectTransfer?: (claw: AgentCardItem) => void;
  /** 更改共享范围 */
  onChangeShareScope?: (claw: AgentCardItem) => void;
  /** 移交：取消「移交待对方确认」，恢复迁移/移交按钮 */
  onCancelTransfer?: (claw: AgentCardItem) => void;
  /**
   * 「他人共享给我」的只读卡片：仅可查看 + 改共享范围，
   * 其余操作（重启/重装/重命名/关机开机/切换角色/删除/迁移/移交）一并禁用。
   */
  sharedReadonly?: boolean;
  /** 「新增角色」入口：直接打开独立的新增角色弹窗 */
  onAddRole?: (claw: AgentCardItem) => void;
  /**
   * 后台正在切换的角色数量（>0 时卡片展示「角色切换中（N）」状态提示）。
   * 用于用户在切换进度弹窗点「我知道了」关闭弹窗后，仍能在卡片上感知切换仍在进行。
   */
  roleSwitchingCount?: number;
  /** 点击「角色切换中」胶囊时触发：打开切换过程弹窗 */
  onClickSwitchingBadge?: (claw: AgentCardItem) => void;
  /**
   * 后台正在新增的角色数量（>0 时卡片展示「角色新增中（N）」状态提示）。
   * 与 roleSwitchingCount 对齐：用户在新增进度弹窗点「我知道了」关闭弹窗后，仍能在卡片上感知新增仍在进行。
   */
  roleAddingCount?: number;
}

interface AgentCardProps extends AgentCardCallbacks {
  claw: AgentCardItem;
}

const TYPE_LABEL: Record<NonNullable<AgentCardItem["agentType"]>, string> = {
  openclaw: "OpenClaw",
  hermes: "Hermes",
  lightclawace: "Lightclaw ACE",
  localagent: "外部 Agent",
};

const LOCAL_AGENT_INACTIVE_DAYS = 7;

const normalizeExternalAgentType = (value?: string) => {
  if (!value) return "OpenClaw";
  if (value === "Hermes Agent") return "Hermes";
  if (value === "Claude") return "Claude Code";
  return value;
};
const LOCAL_AGENT_INACTIVE_MS = LOCAL_AGENT_INACTIVE_DAYS * 24 * 60 * 60 * 1000;

function parseLocalReportedAt(value?: string) {
  if (!value) return null;
  const normalized = value.includes("T") ? value : value.replace(" ", "T");
  const timestamp = new Date(normalized).getTime();
  return Number.isNaN(timestamp) ? null : timestamp;
}

function isLocalAgentInactive(claw: AgentCardItem) {
  if (claw.agentType !== "localagent") return false;
  const lastReportedAt = parseLocalReportedAt(claw.lastReportedAt);
  if (!lastReportedAt) return true;
  return Date.now() - lastReportedAt > LOCAL_AGENT_INACTIVE_MS;
}

// ── 版本更新状态 ───────────────────────────────────────────────

/** Agent 卡片 agentType → 管控端 admin_images_v3 中 agentType key 的映射 */
const AGENT_TYPE_TO_ADMIN_KEY: Record<string, string> = {
  openclaw: "OpenClaw",
  hermes: "HermesAgent",
  lightclawace: "LightClawACE",
};

type UpgradeStatus =
  | { type: "updatable"; current: string; target: string }
  | { type: "latest"; version: string }
  | { type: "downgrade-blocked"; current: string; target: string }
  | { type: "no-effective-image" }
  | { type: "hidden" };

function computeUpgradeStatus(
  agentType: string | undefined,
  currentVersion: string | undefined,
): UpgradeStatus {
  if (!agentType || !currentVersion) return { type: "hidden" };

  const adminKey = AGENT_TYPE_TO_ADMIN_KEY[agentType];
  if (!adminKey) return { type: "hidden" };

  // 提醒更新开关未开启 → 不显示
  if (!isReminderEnabled(adminKey)) return { type: "hidden" };

  // 读取管控端设置的目标版本
  let targetVersion: string | null = null;
  try {
    const raw = localStorage.getItem("admin_images_v3");
    if (raw) {
      const images = JSON.parse(raw) as Array<{
        agentType: string;
        agentVersion: string;
        active: boolean;
      }>;
      const active = images.find(
        (i) => i.agentType === adminKey && i.active,
      );
      if (active) targetVersion = active.agentVersion;
    }
  } catch {
    /* ignore */
  }

  // 兜底：localStorage 为空时使用与 BatchUpdateNotice 一致的默认启用版本
  if (!targetVersion) {
    const DEFAULTS: Record<string, string> = {
      OpenClaw: "2026.4.23",
      HermesAgent: "0.12.0",
      LightClawACE: "0.1.8",
    };
    targetVersion = DEFAULTS[adminKey] ?? null;
  }

  if (!targetVersion) return { type: "no-effective-image" };

  const cmp = compareVersion(currentVersion, targetVersion);
  if (cmp < 0)
    return { type: "updatable", current: currentVersion, target: targetVersion };
  if (cmp > 0)
    return { type: "downgrade-blocked", current: currentVersion, target: targetVersion };
  return { type: "latest", version: currentVersion };
}

/**
 * 从 createdAt 文本里提取「年月日」前缀（YYYY-MM-DD）。
 * 兼容三种入参：
 *   1) "2026-04-04 14:00:00" / "2026-04-04T14:00:00"  —— 直接取前 10 位
 *   2) "2026/4/4 14:00:00"  —— Date 解析后格式化为 YYYY-MM-DD
 *   3) 任意无法识别的字符串  —— 兜底返回原字符串（避免空白 UI）
 */
const extractDateOnly = (raw: string): string => {
  if (!raw) return raw;
  const isoMatch = raw.match(/^(\d{4})[-/](\d{1,2})[-/](\d{1,2})/);
  if (isoMatch) {
    const [, y, m, d] = isoMatch;
    return `${y}-${m.padStart(2, "0")}-${d.padStart(2, "0")}`;
  }
  const parsed = new Date(raw);
  if (!isNaN(parsed.getTime())) {
    const y = parsed.getFullYear();
    const m = String(parsed.getMonth() + 1).padStart(2, "0");
    const d = String(parsed.getDate()).padStart(2, "0");
    return `${y}-${m}-${d}`;
  }
  return raw;
};

/**
 * 创建时间展示。Tooltip + 测量驱动三档降级。
 *
 * 三档展示（按容器可用宽度自动选择最长能放下的那档）：
 *   档 0  full  ——  "2026-04-04 14:00:00 创建"  （宽度足够）
 *   档 1  date  ——  "2026-04-04 创建"          （档 0 放不下）
 *   档 2  bare  ——  "2026-04-04"               （档 1 也放不下，去掉"创建"两字，
 *                                                  绝不出现省略号）
 *   Tooltip 永远展示档 0 完整文本。
 *
 * 关键设计：
 *   1. 每档准备一个 measure 节点（容器内 absolute），offsetWidth 即真实宽度。
 *   2. 默认 tier=2（最短），useLayoutEffect 测完后挑能放下的最长档。这样组件挂载瞬间
 *      不会出现"完整文本被 CSS 半截裁切"。
 *   3. callback ref：Radix `TooltipTrigger asChild` 通过 cloneElement 转发 ref，
 *      直接 useRef 可能被吞，必须用 callback ref 拿到真实 DOM。
 */
const CreatedAtText = ({ createdAt, dimmed }: { createdAt: string; dimmed: boolean }) => {
  const [container, setContainer] = useState<HTMLDivElement | null>(null);
  const fullRef = useRef<HTMLSpanElement>(null);
  const dateRef = useRef<HTMLSpanElement>(null);
  // 0=full, 1=date, 2=bare（默认从最短开始，测完再尽可能切回更长）
  const [tier, setTier] = useState<0 | 1 | 2>(2);

  const dateOnly = extractDateOnly(createdAt);
  const fullText = `${createdAt} 创建`;
  const dateText = `${dateOnly} 创建`;
  const bareText = dateOnly;

  useLayoutEffect(() => {
    if (!container) return;
    const fullEl = fullRef.current;
    const dateEl = dateRef.current;
    if (!fullEl || !dateEl) return;

    const check = () => {
      const available = container.clientWidth;
      // 留 1px 容差防亚像素抖动
      if (fullEl.offsetWidth <= available + 1) {
        setTier(0);
      } else if (dateEl.offsetWidth <= available + 1) {
        setTier(1);
      } else {
        setTier(2);
      }
    };

    check();
    const ro = new ResizeObserver(check);
    ro.observe(container);
    return () => ro.disconnect();
  }, [container, fullText, dateText]);

  const visibleText = tier === 0 ? fullText : tier === 1 ? dateText : bareText;

  return (
    <Tooltip delayDuration={200}>
      <TooltipTrigger asChild>
        <div
          ref={setContainer}
          // 注意：保留 truncate 仅作"理论上不可能再触发"的最终兜底；
          // 由于 tier 2 = 裸日期一定能放下（卡片 min-w-[360px] 保证），实际永远不会截断。
          className={`relative truncate min-w-0 flex-1${dimmed ? " opacity-40" : ""}`}
          style={{
            fontFamily: "PingFang SC, -apple-system, BlinkMacSystemFont, sans-serif",
            fontWeight: 400,
            fontSize: "12px",
            lineHeight: "20px",
            color: "var(--muted-foreground)",
          }}
        >
          {/* 隐藏 measure 节点：absolute 脱离布局，继承字体上下文，offsetWidth 即真实宽度 */}
          <span
            ref={fullRef}
            aria-hidden
            style={{
              position: "absolute",
              left: 0,
              top: 0,
              visibility: "hidden",
              pointerEvents: "none",
              whiteSpace: "nowrap",
            }}
          >
            {fullText}
          </span>
          <span
            ref={dateRef}
            aria-hidden
            style={{
              position: "absolute",
              left: 0,
              top: 0,
              visibility: "hidden",
              pointerEvents: "none",
              whiteSpace: "nowrap",
            }}
          >
            {dateText}
          </span>
          {visibleText}
        </div>
      </TooltipTrigger>
      <TooltipContent side="top" className="text-xs">
        {fullText}
      </TooltipContent>
    </Tooltip>
  );
};

export const AgentCard = ({
  claw,
  onClickCard,
  onRefresh,
  onRestart,
  onReinstall,
  onDelete,
  onSwitchRole,
  onRetry,
  onChat,
  canOpenTerminal,
  refreshing,
  groupMode,
  onRename,
  onShutdown,
  onPowerOn,
  onMigrate,
  onTransfer,
  onLocalReconnect,
  onAcceptTransfer,
  onRejectTransfer,
  onChangeShareScope,
  onCancelTransfer,
  sharedReadonly = false,
  onAddRole,
  roleSwitchingCount = 0,
  onClickSwitchingBadge,
  roleAddingCount = 0,
}: AgentCardProps) => {
  const isDisabled = STATUS_DISABLED[claw.status];
  const isGrayAvatar = STATUS_GRAY_AVATAR[claw.status];
  /**
   * 异常态文字降级（标题 / 元信息 / 组织 / 创建时间整体 40% 透明）
   * 与 main 一致：createFail / shutdown / loadFail / pending = true，无例外排除。
   */
  const isLoadFail = claw.status === "loadFail";
  const isLocalAgent = claw.agentType === "localagent";
  const isLocalInactive = isLocalAgentInactive(claw);
  const isDimmedText = STATUS_DIMMED_TEXT[claw.status];
  const isNonOpenclaw =
    claw.agentType === "hermes" || claw.agentType === "lightclawace" || isLocalAgent;
  const typeLabel = TYPE_LABEL[claw.agentType ?? "openclaw"];
  const externalAgentClientLabel =
    claw.localProduct === "Claude" ? "Claude Code" : claw.localProduct;
  const externalAgentTypeLabel = normalizeExternalAgentType(externalAgentClientLabel || claw.model || typeLabel);
  const avatarRoleName = isLocalAgent ? "办公能手" : claw.roleName;
  const upgradeStatus = computeUpgradeStatus(claw.agentType, claw.version);

  // 「可更新」胶囊点击：不再跳转设置页，仅弹出更新确认弹窗；
  // 用户在弹窗内点「确认更新」后，才跳转详情页并启动更新流程（?action=update）。
  const [, navigate] = useLocation();
  const [showUpdateConfirm, setShowUpdateConfirm] = useState(false);

  // 角色文字展示：
  //   - 仅一个角色位（或无 slot 数据）时：展示该角色名称（此时 agent 名称与该角色名相同）。
  //   - 多个角色位时：仅展示角色个数「N 个」（agent 名称与第一个角色名称相同）。
  // 单角色/无 slot 数据时回退到 roleName，缺省为「通用助手」。
  const roleDisplayText = (() => {
    const count = claw.roleCount ?? claw.roles?.length ?? 1;
    if (count > 1) {
      return `${count} 个`;
    }
    if (claw.roles && claw.roles.length === 1) {
      return claw.roles[0].roleName || "通用助手";
    }
    return claw.roleName || "通用助手";
  })();

  const [copied, setCopied] = useState(false);
  const handleCopyId = (e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard.writeText(claw.instanceId).then(() => {
      setCopied(true);
      toast.success(`已复制 ${claw.instanceId}`);
      setTimeout(() => setCopied(false), 1500);
    });
  };

  /**
   * 版本号渲染节点（抽取自普通卡第 2 行版本逻辑，供普通卡与「组织变更」卡复用，
   * 保证「您已不在该组织」卡片的版本信息与其他卡片完全一致，含可更新蓝胶囊）。
   * 末尾分隔符由调用处按需补充。
   */
  const versionNode =
    upgradeStatus.type === "hidden" ? null : upgradeStatus.type === "updatable" ? (
      <Tooltip delayDuration={100}>
        <TooltipTrigger asChild>
     <button
            type="button"
        onClick={(e) => {
 e.stopPropagation();
              setShowUpdateConfirm(true);
      }}
          >
   <span
        className="inline-flex items-center font-medium rounded-full transition-colors hover:brightness-95 cursor-pointer"
      style={{
         gap: "4px",
     padding: "1px 8px",
    background: "var(--alert-info-bg)",
      color: "var(--text-brand)",
       lineHeight: "18px",
     }}
        aria-label="点击查看可更新版本详情"
role="button"
 >
              <span>版本：v{upgradeStatus.current}</span>
          <ArrowUpCircle className="w-3 h-3 animate-pulse" />
         </span>
          </button>
        </TooltipTrigger>
  <TooltipContent side="top" className="text-xs">
          点击立即更新（当前 v{upgradeStatus.current} → 最新 v{upgradeStatus.target}）
   </TooltipContent>
 </Tooltip>
    ) : upgradeStatus.type === "no-effective-image" ? (
      <span style={{ color: "#737373" }}>无生效镜像</span>
    ) : (
      <span>版本：v{upgradeStatus.type === "latest" ? upgradeStatus.version : upgradeStatus.type === "downgrade-blocked" ? upgradeStatus.current : ""}</span>
    );

  const card = (
    <TenantCard
      interactive={!isDisabled}
      className={`group relative h-full min-h-[266px] min-w-[360px] ${
        !isDisabled ? "cursor-pointer" : "cursor-default"
      }`}
      onClick={() => {
        if (!isDisabled) onClickCard(claw);
      }}
    >
      {/* 接收方移交确认覆盖层 */}
      {claw.incomingTransfer && onAcceptTransfer && onRejectTransfer && (
        <TransferReceiveOverlay
          fromUser={claw.incomingTransfer.fromUser}
          instanceName={claw.incomingTransfer.instanceName}
          onAccept={() => onAcceptTransfer(claw)}
          onReject={() => onRejectTransfer(claw)}
        />
      )}
      {isLocalAgent && (
        <div
          aria-hidden="true"
          className="pointer-events-none absolute left-0 top-0 z-10 h-12 w-12 overflow-hidden rounded-tl-[var(--radius-card)]"
        >
          <div
            className="absolute left-0 top-0 h-0 w-0 border-r-[46px] border-t-[46px] border-r-transparent border-t-[#EAF1FF]"
          />
          <span className="absolute left-[3px] top-[8px] -rotate-45 text-[10px] font-medium leading-none text-[#1447E6]">
            外部
          </span>
        </div>
      )}
      {/* ===== 头部行：头像 + 名称/状态 + 右上角三点菜单 ===== */}
      <div className="flex items-start justify-between gap-3">
        <div className={`flex items-center gap-4 min-w-0 flex-1 ${isLocalAgent ? "pl-3" : ""}`}>
          <AgentAvatar
            roleName={avatarRoleName}
            agentName={claw.name}
            size={48}
            grayed={isDimmedText}
          />
          <div className="flex flex-col gap-1 min-w-0 flex-1">
            <div className="flex items-center gap-2 min-w-0">
              <Tooltip delayDuration={0}>
                <TooltipTrigger asChild>
                  <h3
                    className={`truncate transition-colors ${
                      isDimmedText
                        ? "text-[var(--foreground)] opacity-40"
                        : isGrayAvatar
                        ? "text-muted-foreground"
                        : "text-[var(--foreground)] group-hover:text-[var(--text-brand)]"
                    }`}
                    style={{
                      fontFamily: "PingFang SC, -apple-system, BlinkMacSystemFont, sans-serif",
                      fontWeight: 500,
                      fontSize: "16px",
                      lineHeight: "24px",
                    }}
                  >
                    {claw.name}
                  </h3>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="text-xs max-w-[280px]">
                  {claw.name}
                </TooltipContent>
              </Tooltip>
            </div>
            {/* 状态徽标 + 刷新 icon 按钮（点击触发 onRefresh，复用已有逻辑） */}
            <div className="inline-flex items-center gap-1.5">
              {isLocalAgent ? (
                isLocalInactive ? (
                  <span className="inline-flex items-center gap-1.5">
                    <StatusTag
                      mode="soft"
                      variant="amber"
                      icon={<Clock3 />}
                      className="h-5 border-amber-200 bg-amber-50 text-amber-700"
                    >
                      不活跃
                    </StatusTag>
                    <span className="text-xs leading-5 text-[var(--text-muted)]">超过7天未同步</span>
                  </span>
                ) : (
                  <span
                    className="inline-flex items-center whitespace-nowrap flex-shrink-0"
                    style={{
                      gap: "4px",
                      padding: "2px 0",
                      height: "20px",
                      color: "#020617",
                      fontFamily: "PingFang SC, -apple-system, BlinkMacSystemFont, sans-serif",
                      fontWeight: 400,
                      fontSize: "12px",
                      lineHeight: "20px",
                    }}
                  >
                    <span
                      className="inline-block rounded-full"
                      style={{
                        width: 8,
                        height: 8,
                        background: "#16A34A",
                      }}
                    />
                    运行中
                  </span>
                )
              ) : (
                <StatusBadge status={claw.status} />
              )}
              {/* 迁移到新组织执行阶段：不再展示状态徽章，改为仅用右上角 toast 通知 */}
              {/* 角色切换中：进度弹窗被「我知道了」关闭后，卡片继续提示后台仍在切换 */}
              {roleSwitchingCount > 0 && (
                <span
                  className="cursor-pointer"
                  onClick={(e) => { e.stopPropagation(); onClickSwitchingBadge?.(claw); }}
                >
                  <StatusTag mode="fill" variant="blue" icon={<RefreshCw className="animate-spin" />}>
                    角色切换中（{roleSwitchingCount}）
                  </StatusTag>
                </span>
              )}
              {/* 角色新增中：进度弹窗被「我知道了」关闭后，卡片继续提示后台仍在新增 */}
              {roleAddingCount > 0 && (
                <StatusTag mode="fill" variant="blue" icon={<RefreshCw className="animate-spin" />}>
                  角色新增中（{roleAddingCount}）
                </StatusTag>
              )}
              {/* 移交给其他用户执行阶段：紧跟「已关机」状态展示 */}
              {claw.transferPhase === "pendingConfirm" && (
                <>
                  <StatusTag mode="soft" variant="blue">移交待对方确认</StatusTag>
                  {onCancelTransfer && (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        onCancelTransfer(claw);
                      }}
                      className="text-xs text-[var(--text-secondary)] hover:text-[var(--text-brand)] transition-colors"
                    >
                      取消
                    </button>
                  )}
                </>
              )}
              {claw.transferPhase === "transferring" && (
                <StatusTag mode="soft" variant="blue">移交中</StatusTag>
              )}
              {claw.transferPhase === "done" && (
                <StatusTag mode="soft" variant="green">移交完成</StatusTag>
              )}
              {claw.transferPhase === "rejected" && (
                <StatusTag mode="soft" variant="red">对方拒绝移交</StatusTag>
              )}
              {claw.transferPhase === "failed" && (
                <StatusTag mode="soft" variant="red">移交失败</StatusTag>
              )}
              <Tooltip delayDuration={200}>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      if (!refreshing) onRefresh(e, claw.id, claw.name);
                    }}
                    disabled={refreshing}
                    className="inline-flex items-center justify-center size-5 rounded-full text-[var(--text-muted)] hover:text-[var(--text-brand)] hover:bg-[var(--accent)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    aria-label="刷新状态"
                  >
                    <RefreshCw className={`w-3 h-3 ${refreshing ? "animate-spin" : ""}`} />
                  </button>
                </TooltipTrigger>
                <TooltipContent side="top" className="text-xs">
                  刷新状态
                </TooltipContent>
              </Tooltip>
            </div>
          </div>
        </div>

        {/* 右上角：仅保留三点菜单（[Figma 1077-33986] 删除外露刷新按钮，刷新仍在菜单内） */}
        <div className={`flex items-center ${isDisabled ? "opacity-40 grayscale" : ""}`}>
          <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              className="w-7 h-7 rounded-[var(--radius-lg)] flex items-center justify-center text-muted-foreground hover:text-foreground hover:bg-[var(--accent)] transition-colors flex-shrink-0"
              onClick={(e) => e.stopPropagation()}
              aria-label="更多操作"
            >
              <MoreVertical className="w-4 h-4" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-52">
        {/* ══ 组 1：刷新状态 / 重启 Agent / 关机（顶部）══ */}
   {/* 刷新状态 */}
          <DropdownMenuItem
       onClick={(e) => {
      e.stopPropagation();
         onRefresh(e, claw.id, claw.name);
           }}
disabled={refreshing}
            >
  <RefreshCw
       className={`w-4 h-4 mr-2 text-muted-foreground ${
      refreshing ? "animate-spin" : ""
                }`}
        />
       刷新状态
       </DropdownMenuItem>

    {!isLocalAgent && !sharedReadonly && (
              /* 重启 Agent */
           claw.status === "running" ? (
    <DropdownMenuItem
           onClick={(e) => {
         e.stopPropagation();
        onRestart(claw);
           }}
         >
             <RotateCcw className="w-4 h-4 mr-2 text-muted-foreground" />
                  重启 Agent
   </DropdownMenuItem>
  ) : (
          <DropdownMenuItem
           disabled
             className="opacity-40 cursor-not-allowed"
  >
       <RotateCcw className="w-4 h-4 mr-2" />
       重启 Agent
    </DropdownMenuItem>
           )
        )}

            {/* 关机 / 开机 */}
            {sharedReadonly ? null : !isLocalAgent && claw.status === "running" && onShutdown ? (
      <DropdownMenuItem
      onClick={(e) => {
       e.stopPropagation();
    onShutdown(claw);
            }}
       >
    <PowerOff className="w-4 h-4 mr-2 text-muted-foreground" />
      关机
            </DropdownMenuItem>
      ) : !isLocalAgent && claw.status === "shutdown" && claw.groupChangeStatus ? (
              // 组织变更中（您已不在该组织）：开机不可用，需先迁移到新组织或移交给其他用户
  <DropdownMenuItem disabled className="opacity-40 cursor-not-allowed">
  <Power className="w-4 h-4 mr-2" />
         开机
           </DropdownMenuItem>
       ) : !isLocalAgent && claw.status === "shutdown" && onPowerOn ? (
          <DropdownMenuItem
   onClick={(e) => {
     e.stopPropagation();
    onPowerOn(claw);
                }}
   >
      <Power className="w-4 h-4 mr-2 text-green-500" />
                开机
   </DropdownMenuItem>
            ) : null}

    {/* ══ 组 2：重新安装 / 重命名 / 进入终端 / 切换角色 / 共享范围（中间）══ */}
            {!isLocalAgent && (
              <>
   <DropdownMenuSeparator />

  {/* 重新安装 */}
       {!sharedReadonly && (
    claw.status === "running" ? (
         <DropdownMenuItem
      onClick={(e) => {
     e.stopPropagation();
  onReinstall(claw);
      }}
       >
          <HardDriveDownload className="w-4 h-4 mr-2 text-muted-foreground" />
           {isNonOpenclaw ? "重新安装 Agent" : "重新安装 OpenClaw"}
      </DropdownMenuItem>
       ) : (
         <DropdownMenuItem
        disabled
                 className="opacity-40 cursor-not-allowed"
          >
 <HardDriveDownload className="w-4 h-4 mr-2" />
   {isNonOpenclaw ? "重新安装 Agent" : "重新安装 OpenClaw"}
         </DropdownMenuItem>
    )
                )}

          {/* 重命名 */}
    {onRename && !sharedReadonly && (claw.status === "running" || claw.status === "shutdown") && (
          <DropdownMenuItem
                    onClick={(e) => {
          e.stopPropagation();
            onRename(claw);
        }}
         >
 <Pencil className="w-4 h-4 mr-2 text-muted-foreground" />
   重命名
      </DropdownMenuItem>
    )}

      {/* 进入终端 */}
       {canOpenTerminal(claw) &&
  (claw.status === "running" ? (
  <DropdownMenuItem
     onClick={(e) => {
      e.stopPropagation();
          window.open(`/terminal/${claw.id}`, "_blank");
                      }}
   >
     <Terminal className="w-4 h-4 mr-2 text-muted-foreground" />
          进入终端
    </DropdownMenuItem>
      ) : (
             <DropdownMenuItem
  disabled
    className="opacity-40 cursor-not-allowed"
          >
            <Terminal className="w-4 h-4 mr-2" />
   进入终端
                    </DropdownMenuItem>
          ))}

{/* 切换角色：仅保留「切换角色」（新增角色功能已下线） */}
          {!sharedReadonly && claw.status === "running" && (() => {
   const allowed = claw.allowedRoleNames;
         const isRestricted = allowed !== undefined;
               const currentRoleName = claw.roleName ?? "通用助手";
         const menuCanSwitch = isRestricted
        ? allowed!.filter((n) => n !== currentRoleName).length + (currentRoleName !== "通用助手" ? 1 : 0) > 0
     : true;
return menuCanSwitch ? (
         <DropdownMenuItem
                    onClick={(e) => {
     e.stopPropagation();
          onSwitchRole(claw);
         }}
           >
            <UserCog className="w-4 h-4 mr-2 text-muted-foreground" />
  角色管理
         </DropdownMenuItem>
          ) : (
  <Tooltip delayDuration={200}>
    <TooltipTrigger asChild>
   <DropdownMenuItem
   aria-disabled
   onSelect={(e) => e.preventDefault()}
  onClick={(e) => e.stopPropagation()}
   className="opacity-40 cursor-not-allowed focus:bg-transparent"
 >
     <UserCog className="w-4 h-4 mr-2 text-muted-foreground" />
    角色管理
        </DropdownMenuItem>
        </TooltipTrigger>
       <TooltipContent side="left" className="text-xs max-w-[240px]">
           该 Agent 属受限分组，管理员未开放其它角色，暂不可切换。如需调整请联系管理员。
           </TooltipContent>
      </Tooltip>
      );
      })()}

                {/* 共享范围 —— 本地客户端不支持共享 */}
   {onChangeShareScope && (
         <DropdownMenuItem
        onClick={(e) => {
    e.stopPropagation();
             onChangeShareScope(claw);
         }}
                  >
    <Share2 className="w-4 h-4 mr-2 text-muted-foreground" />
          共享范围
   </DropdownMenuItem>
)}

  {/* ══ 组 3：删除（底部，单独一组）══ */}
                {/* 删除 —— 被共享者无删除权限 */}
   {!sharedReadonly && (
       <>
      <DropdownMenuSeparator />
         {["creating", "loading", "pending"].includes(claw.status) ? (
          <DropdownMenuItem
         disabled
     className="opacity-40 cursor-not-allowed text-destructive"
         >
     <Trash2 className="w-4 h-4 mr-2 text-destructive" />
        删除
    </DropdownMenuItem>
    ) : (
<DropdownMenuItem
             onClick={(e) => {
        e.stopPropagation();
    onDelete(claw);
           }}
       className="text-destructive focus:text-destructive"
            >
         <Trash2 className="w-4 h-4 mr-2 text-destructive" />
             删除
           </DropdownMenuItem>
               )}
     </>
    )}
</>
         )}

            {isLocalAgent && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={(e) => {
                    e.stopPropagation();
                    onDelete(claw);
                  }}
                  className="text-destructive focus:text-destructive"
                >
                  <Trash2 className="w-4 h-4 mr-2 text-destructive" />
                  移除外部 Agent
                </DropdownMenuItem>
              </>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
        </div>
      </div>

      {/* ===== 元信息组：column gap 4 ===== */}
      <div className="flex flex-col" style={{ gap: "4px" }}>
        {/* 元信息第 1 行：角色（非本地）+ 切换按钮 | 类型 | ID + 复制 | 版本，同一行展示 */}
        <div
      className={`flex items-center flex-wrap gap-1 text-xs font-normal leading-5 text-[var(--text-secondary)]${isDimmedText ? " opacity-40" : ""}`}
        >
    {!isLocalAgent && (
       <>
    <span className="min-w-0 truncate">角色：{roleDisplayText}</span>
           {/* 角色切换 / 新增中：隐藏切换 / 新增入口，避免并发操作（状态提示由头部 StatusTag 承载） */}
      {claw.status === "running" && !sharedReadonly && roleSwitchingCount === 0 && roleAddingCount === 0 && (
      /* 「切换角色」图标按钮（hover tooltip）。
  受限分组（allowedRoleNames 存在）时，无可切换角色则禁用并给出说明。 */
        (() => {
   const allowed = claw.allowedRoleNames;
   const isRestricted = allowed !== undefined;
         const currentRoleName = claw.roleName ?? "通用助手";
         // 可切换：白名单内排除当前角色的目标数 +（当前非通用助手可切回通用助手）
                  const switchTargetCount = isRestricted
         ? allowed!.filter((n) => n !== currentRoleName).length + (currentRoleName !== "通用助手" ? 1 : 0)
  : 1;
  const canSwitch = switchTargetCount > 0;
                  return (
           <span className="inline-flex items-center gap-0.5 shrink-0">
    <Tooltip delayDuration={200}>
          <TooltipTrigger asChild>
      <button
type="button"
      aria-disabled={!canSwitch}
   onClick={(e) => {
           e.stopPropagation();
   if (!canSwitch) return;
   onSwitchRole(claw);
     }}
  className={`inline-flex items-center justify-center size-5 rounded-[var(--radius-sm)] text-[var(--muted-foreground)] transition-colors ${canSwitch ? "hover:bg-[var(--accent)] hover:text-[var(--text-emphasis)] cursor-pointer" : "opacity-40 cursor-not-allowed"}`}
             aria-label="角色管理"
    >
   <UserCog className="w-3 h-3" />
          </button>
          </TooltipTrigger>
 <TooltipContent side="top" className="text-xs">
      {canSwitch ? "角色管理" : "该 Agent 属受限分组，管理员未开放其它角色，暂不可切换。如需调整请联系管理员。"}
             </TooltipContent>
    </Tooltip>
             </span>
         );
       })()
        )}
      <span className="text-[var(--border)]">｜</span>
      </>
          )}
    {isLocalAgent && (
 <>
              <span>外部 Agent 类型：{externalAgentTypeLabel}</span>
 <span className="text-[var(--border)]">｜</span>
       </>
          )}
       {!isLocalAgent && (
            <>
 <span>类型：{typeLabel}</span>
   <span className="text-[var(--border)]">｜</span>
     </>
          )}
     <span className="inline-flex items-center" style={{ gap: "6px" }}>
     ID：{claw.instanceId}
     <button
        type="button"
    onClick={handleCopyId}
              className="w-3 h-3 inline-flex items-center justify-center rounded-[var(--radius-sm)] text-[var(--muted-foreground)] hover:text-[var(--foreground)] transition-colors"
              aria-label="复制 ID"
      >
   {copied ? (
                <Check className="w-3 h-3" />
    ) : (
    <Copy className="w-3 h-3" />
        )}
   </button>
</span>
        </div>

        {/* 元信息第 3 行：组织 / 组织变更态 */}
        {claw.groupChangeStatus ? (
          <div className="mt-0.5">
            <GroupChangeBadge
              status={claw.groupChangeStatus}
              originalGroup={claw.groupChangeOriginalGroup || claw.groupName || "—"}
              transferTarget={claw.groupChangeTransferTarget}
              versionSlot={versionNode}
            />
          </div>
        ) : (
          <div
            className={`flex flex-col ${isDimmedText ? "opacity-40" : ""}`}
            style={{
              gap: "2px",
              fontFamily: "PingFang SC, -apple-system, BlinkMacSystemFont, sans-serif",
              fontWeight: 400,
              fontSize: "12px",
              lineHeight: "18px",
              color: "var(--text-secondary)",
            }}
          >
            {/* 组织行（外部 Agent 卡含最近同步，共享卡含创建者） */}
            <div className="flex items-center gap-1">
              {isLocalAgent && (
                <>
                  <span className="truncate">
                    最近同步：{claw.lastReportedAt || "暂无"}
                  </span>
                  <span style={{ color: "#E2E8F0", margin: "0 3px" }}>｜</span>
                </>
              )}
  <span className="truncate">
    组织：
{groupMode === "multi-group"
       ? claw.groupName || "A公司 / 技术部 / 前端组"
 : "默认"}
      </span>
      {/* 版本：固定跟在组织信息右边（无组织变更态时）；有组织变更态时版本仍由下方 GroupChangeBadge 承载 */}
         {!claw.groupChangeStatus && versionNode && (
    <>
    <span className="shrink-0" style={{ color: "#E2E8F0", margin: "0 3px" }}>｜</span>
    <span className="inline-flex items-center shrink-0">{versionNode}</span>
      </>
)}
              {/* 创建者——仅共享实例展示，跟在分组后面 */}
              {onChangeShareScope && !isLocalAgent && claw.shareScope === "shared" && claw.creator && (
                <>
                  <span style={{ color: "#E2E8F0", margin: "0 3px" }}>｜</span>
                  <span>创建者：{claw.creator.split("@")[0]}</span>
                </>
              )}
            </div>
          </div>
        )}

        {/* 元信息第 3 行：共享范围（可点击修改）——外部 Agent 不支持共享 */}
        {onChangeShareScope && !isLocalAgent && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onChangeShareScope(claw);
            }}
            className={`flex items-center gap-1 rounded-[var(--radius-sm)] px-0.5 -mx-0.5 transition-colors ${
              isDimmedText ? "opacity-40" : ""
            } hover:bg-[var(--accent)] hover:text-[var(--text-emphasis)] cursor-pointer text-left`}
            style={{
              fontFamily: "PingFang SC, -apple-system, BlinkMacSystemFont, sans-serif",
              fontWeight: 400,
              fontSize: "12px",
              lineHeight: "20px",
              color: "var(--text-secondary)",
            }}
          >
            <Share2 className="w-3 h-3 flex-shrink-0" />
            {claw.shareScope === "shared"
              ? (() => {
                  // 统一展示规则：
                  //   - 整组全选 → 显示组名（shareGroupNames）
                  //   - 组内只选了部分人 → 显示个人名字（shareUserNames）
                  // 组名优先排在前面，再接散选个人。
                  const groupNames = claw.shareGroupNames ?? [];
                  const userNames = claw.shareUserNames ?? [];
                  const allParts = [...groupNames, ...userNames];
                  if (allParts.length === 0) return "共享：未选择";

                  // 最多显示 3 个，超出 +N
                  const shown = allParts.slice(0, 3);
                  const rest = allParts.length - 3;
                  return (
                    <span className="truncate">
                      共享：{shown.join("、")}{rest > 0 ? `、+${rest}` : ""}
                    </span>
                  );
                })()
              : "仅自己可见"}
          </button>
        )}
      </div>

      {/* ===== 底部行：时间 + 操作按钮（无分隔线）；mt-auto 固定贴卡片底部 =====
          窄屏（如 xl 三列、视口 1280~1366）下卡片内宽仅约 290px，时间 + 双按钮的
          最小宽度会超出卡片导致按钮溢出右边缘被裁切。解决：
          - 时间使用 <CreatedAtText/>：宽度足够时显示完整「YYYY-MM-DD HH:mm:ss 创建」，
            不够时自动降级为「YYYY-MM-DD 创建」，hover Tooltip 始终展示完整时间。
          - 按钮组保持 flex-shrink-0 始终完整可见。 */}
      <div className="flex items-center justify-between gap-3 mt-auto">
        <CreatedAtText createdAt={claw.createdAt} dimmed={isDimmedText} />
        <div className="flex items-center flex-shrink-0" style={{ gap: "12px" }}>
          {(claw.groupChangeStatus === "pending" ||
            claw.groupChangeStatus === "rejected" ||
            claw.groupChangeStatus === "expired") &&
          claw.migrationPhase !== "migrating" &&
          claw.migrationPhase !== "success" &&
          (claw.transferPhase === undefined || claw.transferPhase === "rejected" || claw.transferPhase === "failed") &&
          onTransfer && !sharedReadonly ? (
            <>
              {!claw.groupChangeTransferOnly && onMigrate && (
                <Button
                  variant="tenant-outline-r20"
                  size="claw"
                  onClick={(e) => {
                    e.stopPropagation();
                    onMigrate(claw);
                  }}
                >
                  迁移到新组织
                </Button>
              )}
              <Button
                variant="tenant-outline-r20"
                size="claw"
                onClick={(e) => {
                  e.stopPropagation();
                  onTransfer(claw);
                }}
              >
                移交给其他用户
              </Button>
            </>
          ) : isLoadFail ? (
            <Button
              onClick={(e) => {
                e.stopPropagation();
                onRetry(claw.id, claw.name);
              }}
              variant="tenant-outline-r20"
              size="claw"
            >
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-4 h-4">
                <path d="M8 6V4C5.79086 4 4 5.79086 4 8C4 8.27966 4.0287 8.55263 4.08332 8.8161L2.4993 10.4001C2.17816 9.66513 2 8.85337 2 8C2 4.68629 4.68629 2 8 2H13L8 6Z" fill="url(#paint0_retry)"/>
                <path d="M8 12C10.2091 12 12 10.2091 12 8C12 7.56499 11.9306 7.1462 11.8022 6.75411L13.3246 5.23168C13.7561 6.05993 14 7.00148 14 8C14 11.3137 11.3137 14 8 14H3L8 10V12Z" fill="url(#paint1_retry)"/>
                <defs>
                  <radialGradient id="paint0_retry" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(14 8) rotate(159.444) scale(12.816 21.3007)">
                    <stop offset="0.748539" stopColor="#202020"/>
                    <stop offset="1" stopColor="#1447E6"/>
                  </radialGradient>
                  <radialGradient id="paint1_retry" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(14 8) rotate(159.444) scale(12.816 21.3007)">
                    <stop offset="0.748539" stopColor="#202020"/>
                    <stop offset="1" stopColor="#1447E6"/>
                  </radialGradient>
                </defs>
              </svg>
              重试
            </Button>
          ) : isLocalAgent && isLocalInactive && onLocalReconnect ? (
            <Button
              variant="tenant-outline-r20"
              size="claw"
              onClick={(e) => {
                e.stopPropagation();
                onLocalReconnect(claw);
              }}
            >
              <RotateCcw className="w-4 h-4" />
              重新接入
            </Button>
          ) : isDisabled ? (
            <Button
              variant="tenant-outline-r20"
              size="claw"
              disabled
            >
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-4 h-4 opacity-30">
                <path d="M9.53796 2H6.46178L6.10863 3.76579C5.81461 3.89729 5.53694 4.05843 5.27938 4.24533L3.57275 3.66799L2.03467 6.33202L3.38795 7.52131C3.37177 7.67884 3.3635 7.83855 3.3635 8C3.3635 8.16151 3.37177 8.32116 3.38795 8.47869L2.03467 9.668L3.57275 12.332L5.27939 11.7547C5.53694 11.9416 5.81462 12.1027 6.10863 12.2342L6.46178 14H9.53796L9.89109 12.2342C10.1851 12.1027 10.4628 11.9416 10.7203 11.7547L12.427 12.332L13.965 9.668L12.6118 8.47869C12.628 8.32116 12.6362 8.16151 12.6362 8C12.6362 7.83855 12.628 7.67884 12.6118 7.52131L13.965 6.33202L12.427 3.66799L10.7203 4.24533C10.4628 4.05843 10.1851 3.89729 9.89109 3.76579L9.53796 2ZM7.99978 10.1818C6.79479 10.1818 5.81796 9.20496 5.81796 8C5.81796 6.79501 6.79479 5.81818 7.99978 5.81818C9.20474 5.81818 10.1816 6.79501 10.1816 8C10.1816 9.20496 9.20474 10.1818 7.99978 10.1818Z" fill="url(#paint0_radial_824_3059)"/>
                <defs>
                  <radialGradient id="paint0_radial_824_3059" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(13.965 8) rotate(-180) scale(11.9304 19.9444)">
                    <stop offset="0.748539" stopColor="#202020"/>
                    <stop offset="1" stopColor="#1447E6"/>
                  </radialGradient>
                </defs>
              </svg>
              设置
            </Button>
          ) : (
            <Link href={`/openclaw/${claw.id}`} onClick={(e) => e.stopPropagation()}>
              <Button variant="tenant-outline-r20" size="claw">
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-4 h-4">
                  <path d="M9.53796 2H6.46178L6.10863 3.76579C5.81461 3.89729 5.53694 4.05843 5.27938 4.24533L3.57275 3.66799L2.03467 6.33202L3.38795 7.52131C3.37177 7.67884 3.3635 7.83855 3.3635 8C3.3635 8.16151 3.37177 8.32116 3.38795 8.47869L2.03467 9.668L3.57275 12.332L5.27939 11.7547C5.53694 11.9416 5.81462 12.1027 6.10863 12.2342L6.46178 14H9.53796L9.89109 12.2342C10.1851 12.1027 10.4628 11.9416 10.7203 11.7547L12.427 12.332L13.965 9.668L12.6118 8.47869C12.628 8.32116 12.6362 8.16151 12.6362 8C12.6362 7.83855 12.628 7.67884 12.6118 7.52131L13.965 6.33202L12.427 3.66799L10.7203 4.24533C10.4628 4.05843 10.1851 3.89729 9.89109 3.76579L9.53796 2ZM7.99978 10.1818C6.79479 10.1818 5.81796 9.20496 5.81796 8C5.81796 6.79501 6.79479 5.81818 7.99978 5.81818C9.20474 5.81818 10.1816 6.79501 10.1816 8C10.1816 9.20496 9.20474 10.1818 7.99978 10.1818Z" fill="url(#paint0_radial_824_3059b)"/>
                  <defs>
                    <radialGradient id="paint0_radial_824_3059b" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(13.965 8) rotate(-180) scale(11.9304 19.9444)">
                      <stop offset="0.748539" stopColor="#202020"/>
                      <stop offset="1" stopColor="#1447E6"/>
                    </radialGradient>
                  </defs>
                </svg>
                设置
              </Button>
            </Link>
          )}
          {/* 第二个按钮：对话 — 仅运行中时显示，其他状态（关机/失败/待处理/组织变更等）隐藏 */}
          {!isLocalAgent && claw.status === "running" && (
          <Button
            variant="tenant-outline-r20"
            size="claw"
            onClick={(e) => {
              e.stopPropagation();
              onChat(claw);
            }}
            disabled={isDisabled}
            aria-label="开始对话"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={`w-4 h-4 ${isDisabled ? "opacity-30" : ""}`}>
              <path d="M7.99988 14.5C11.5897 14.5 14.4999 11.5898 14.4999 8C14.4999 4.41015 11.5897 1.5 7.99988 1.5C4.41003 1.5 1.49988 4.41015 1.49988 8C1.49988 9.73056 2.17615 11.3031 3.27884 12.4679L2.14988 14.5H7.99988Z" fill="url(#paint0_radial_824_3063)"/>
              <rect x="7.66602" y="6.16699" width="1.5" height="2" rx="0.75" fill="#D9D9D9"/>
              <rect x="10.666" y="6.16699" width="1.5" height="2" rx="0.75" fill="#D9D9D9"/>
              <defs>
                <radialGradient id="paint0_radial_824_3063" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(14.4999 8) rotate(-180) scale(13 21.6065)">
                  <stop offset="0.748539" stopColor="#202020"/>
                  <stop offset="1" stopColor="#1447E6"/>
                </radialGradient>
              </defs>
            </svg>
            对话
          </Button>
          )}
        </div>
      </div>
  </TenantCard>
  );

  // ── 「可更新」胶囊点击后的更新确认弹窗（不跳转设置页，仅在当前页弹出） ──
  const upgradeCurrent =
  upgradeStatus.type === "updatable" ? upgradeStatus.current : claw.version;
  const upgradeTarget =
    upgradeStatus.type === "updatable" ? upgradeStatus.target : null;
  const updateConfirmDialog = (
    <Dialog open={showUpdateConfirm} onOpenChange={setShowUpdateConfirm}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold text-gray-900">
         更新确认
       </DialogTitle>
          <DialogDescription className="sr-only">更新确认</DialogDescription>
        </DialogHeader>
        <div className="text-sm text-gray-700 leading-relaxed space-y-3 py-1">
          {/* 当前版本 vs 目标版本 */}
          <div className="rounded-lg border border-[var(--border)] p-3 space-y-2">
      <div className="flex items-center justify-between">
      <span className="text-[var(--text-muted)]">当前 Agent 版本</span>
     <span className="font-mono text-[var(--text-emphasis)] font-medium">
           v{upgradeCurrent}
    </span>
 </div>
   <div className="flex items-center justify-between">
       <span className="text-[var(--text-muted)]">目标 Agent 版本</span>
      <span className="font-mono text-[var(--text-brand)] font-medium">
          {upgradeTarget ? `v${upgradeTarget}` : "暂无可更新版本"}
   </span>
       </div>
 </div>
      {!upgradeTarget && (
 <Alert variant="warning">
        <AlertCircle className="w-4 h-4" />
   <AlertDescription>
  暂未找到该 Agent 类型的生效镜像版本，请联系管理员在镜像管理中配置。
       </AlertDescription>
            </Alert>
          )}
          <p>Agent版本将会更新至管理员指定生效镜像所对应的版本，且不支持跨Agent类型更新。</p>
          <p>更新版本预计需要 5～10 分钟不等，请您耐心等待。更新期间 Agent 网关服务暂停，面板不可操作。</p>
   <p>更新版本后模型（Models）、通道（Channels）、技能（Skills）和记忆均不会丢失。</p>
    </div>
        <div className="flex justify-end gap-3 pt-2">
          <Button
         variant="tenant-outline"
      size="claw-sm"
         onClick={() => setShowUpdateConfirm(false)}
 className="px-5"
          >
     取消
          </Button>
          <Button
     size="sm"
     className="px-5"
    disabled={!upgradeTarget}
            onClick={() => {
          setShowUpdateConfirm(false);
     // 确认更新后跳转详情页并启动更新流程
           navigate(`/openclaw/${claw.id}?action=update`);
            }}
          >
            确认更新
          </Button>
    </div>
      </DialogContent>
    </Dialog>
  );

  if (!isLocalInactive) {
    return (
      <>
        {card}
  {updateConfirmDialog}
    </>
    );
  }

  return (
    <>
<Tooltip delayDuration={150}>
     <TooltipTrigger asChild>{card}</TooltipTrigger>
        <TooltipContent side="top" className="text-xs">
          发起对话后恢复活跃
  </TooltipContent>
      </Tooltip>
   {updateConfirmDialog}
    </>
  );
};
