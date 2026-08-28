/**
 * MyOpenClaw - 我的 Agent 页面
 * Design: 「流动蓝图」Fluid Blueprint
 * - Hero Banner + 快速上手引导（可关闭）
 * - Agent 卡片列表（支持 8 种状态，含组织变更专属状态）
 * - 创建 Agent 弹窗（单步表单：组织 + 名称 + 类型 + 角色）
 * - 操作确认弹窗（重启 / 重装 / 删除 / 移除角色 / 重命名 / 关机）
 * - 自动轮询状态转换（creating/loading/maintaining → running）
 * - 组织变更：主动提醒 / 迁移 / 移交 / 接收方蒙版
 *
 * NOTE：本页通知逻辑（Notification）由顶导航 `topnav/NotificationPanel` 统一承担，
 *       此前页面内的本地通知面板（Bell + Notification 列表）已无 UI 入口，于 0609 整体删除。
 */
import { useState, useEffect, useRef, useCallback, useMemo, Fragment } from "react";
import { Link, useLocation } from "wouter";
import TenantLayout from "@/components/TenantLayout";
import { Button } from "@/components/ui/button";
import { DialogPagination } from "@/components/ui/pagination";
import { Progress } from "@/components/ui/progress";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogBody,
  DialogDescription,
} from "@/components/ui/dialog";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SearchableSelect,
} from "@/components/ui/select";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";
import { SelectPanel, SelectPanelItem } from "@/components/ui/select-panel";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { SurfaceInner } from "@/components/ui/Surface";
import { cn } from "@/lib/utils";
import { Separator } from "@/components/ui/separator";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import { Empty, EmptyMedia, EmptyHeader, EmptyDescription, EmptyContent } from "@/components/ui/empty";
import { BodyText, MetaText, CodeText, HelperText, SectionTitle, BodyMedium, MetaMedium, PanelTitle, CompactText } from "@/components/ui/Typography";
import {
  AlertDialog, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  TableActionCell,
  TableExpandedRow,
} from "@/components/ui/table";
import { toast } from "sonner";
import { Plus, X, AlertCircle, Copy, Search, ChevronDown, Plug, Laptop, CheckCircle2, Apple, Monitor, Share2, Users, User, HelpCircle, Info, RotateCcw, Trash2, Loader2, MoreHorizontal, Pencil, Check, ArrowLeftRight, Star } from "lucide-react";
import { ShareScopeTree } from "@/components/agent/ShareScopeTree";
import { ShareScopeTags } from "@/components/agent/ShareScopeTags";
import { groupStore } from "../admin/MemberManagement/groupStore";
import type { PolicyRule } from "@/components/policy/types";
import AgentChat from "./AgentChat";
// 注：旧版对话视图 `ChatView` 暂不再渲染，相关业务逻辑保留在 `./ChatView`，需要时再接入
import { MOCK_ROLES } from "@/lib/mockData";
import type { Role, AgentRoleSlot } from "@/lib/mockData";
import { loadClawList, saveClawList, notifyClawListChange } from "@/lib/openclawStore";
import { ensureBackupDemoAgents } from "@/lib/backupDemo";
import { pushAdminNotification } from "@/lib/adminNotificationStore";
import { isSharedToMe } from "@/lib/currentUser";
import { HeroBanner } from "@/components/agent/HeroBanner";
import { QuickStartGuide } from "@/components/agent/QuickStartGuide";
import { ViewModeSegmented } from "@/components/agent/ViewModeSegmented";
import { AgentCard } from "@/components/agent/AgentCard";
import { AgentAvatar } from "@/components/agent/AgentAvatar";
import { RoleManageSheet } from "@/components/agent/RoleManageSheet";
import { BatchSwitchRoleDialog } from "@/components/agent/BatchSwitchRoleDialog";
import { AddRoleDialog } from "@/components/agent/AddRoleDialog";
import { RoleConfigProgressDialog, RoleConfigProgressContent, useRoleConfigProgress } from "@/components/agent/RoleConfigProgress";
import { useServiceStatus } from "@/contexts/ServiceStatusContext";
import { getAgentBillingMode, getDefaultLaunchBillingMode } from "@/lib/agentBillingStore";
import {
  GroupChangeBadge,
  MigrateDialog,
  TransferDialog,
  TransferReceiveOverlay,
  MOCK_TRANSFER_REQUESTS,
} from "./GroupChangeComponents";

const DISABLED_TIP = "您的 OpenClaw 已被管理员停用，无法操作";
const LAUNCH_FAILED_TIP = "创建失败，无法操作";

// [006] 列表分页：每页默认 30 条，与后端 GET /openclaw/list 默认 page_size 保持一致
const PAGE_SIZE = 30;
const AGENT_NAME_MAX_BYTES = 128;
/** 角色名称长度上限（与 BatchSwitchRoleDialog / RoleManageSheet 保持一致） */
const ROLE_NAME_MAX_LEN = 25;
const LOCAL_AGENT_REMOVE_OPERATIONS_KEY = "openclaw_local_agent_remove_operations";

const getAgentNameByteLength = (value: string) => new TextEncoder().encode(value).length;

// 技能名 → 技能描述映射（RoleSkill 无 description 字段，此处按常见技能名给出说明，未命中回落到通用文案）
const SKILL_DESCRIPTION_MAP: Record<string, string> = {
  // ===== 研发工具类 =====
  "github": "通过 GitHub CLI 管理 Issue、PR、CI 运行与 API 查询，覆盖代码协作全流程",
  "code-reviewer": "自动审查代码质量与安全漏洞，支持主流编程语言，提供修改建议与最佳实践",
  "api-tester": "接口自动化测试，覆盖参数校验、断言验证与回归测试",
  "code-runner": "代码执行沙箱，安全运行并返回结果",
  "self-improving-agent": "自我优化代理，持续学习并改进任务执行策略",
  // ===== 数据分析类 =====
  "data-analyst": "数据分析与可视化，支持数据清洗、统计分析与图表生成",
  "sql-expert": "SQL 查询编写与优化，支持多数据库方言、索引建议与执行计划分析",
  // ===== 信息检索类 =====
  "web-search": "联网检索，实时获取网页与资讯信息",
  "web-search-pro": "增强版联网检索，支持深度搜索、多源交叉验证与结构化摘要",
  "file-reader": "文件解析，读取并提取多格式文档内容",
  // ===== 运维部署类 =====
  "docker-ops": "容器编排与运维，支持镜像构建、部署与监控",
  "k8s-manager": "Kubernetes 集群管理，负责编排、扩缩容与故障排查",
  // ===== 办公效率类 =====
  "email-writer": "邮件撰写助手，自动生成规范专业的邮件正文",
  "文档总结助手": "快速总结长文档并提取关键要点，支持多种文档格式",
  "智能翻译工具": "多语言智能翻译，支持上下文感知与专业术语准确转换",
  "API 自动化测试": "接口自动化测试套件管理，覆盖参数校验与回归验证",
  "代码质量扫描": "静态代码分析，检测规范违规、安全漏洞与性能问题",
};
const getSkillDescription = (name: string) => SKILL_DESCRIPTION_MAP[name] ?? "该技能将随角色一并预装并激活";

type LocalClientAccessTarget = "CodeBuddy" | "WorkBuddy" | "Claude Code" | "Codex" | "iMate" | "KnotBot";
type LocalClientSystem = "mac" | "win";
type LocalClientAccessStep = "system" | "product";

const LOCAL_CLIENT_ACCESS_OPTIONS: Array<{
  name: LocalClientAccessTarget;
  description: string;
}> = [
  {
    name: "CodeBuddy",
    description: "接入 CodeBuddy 外部 Agent，支持本地开发环境纳入企业管理。",
  },
  {
    name: "WorkBuddy",
    description: "接入 WorkBuddy 外部 Agent，统一展示接入状态和企业 Skill。",
  },
  {
    name: "Claude Code",
    description: "接入 Claude Code 外部 Agent，支持终端环境纳入企业管理。",
  },
  {
    name: "Codex",
    description: "接入 Codex 外部 Agent，支持 Codex 开发环境纳入企业管理。",
  },
  {
    name: "iMate",
    description: "接入 iMate 外部 Agent，统一展示接入状态并同步企业资源。",
  },
  {
    name: "KnotBot",
    description: "接入 KnotBot 外部 Agent，统一展示接入状态并同步企业资源。",
  },
];

const LOCAL_CLIENT_SYSTEM_OPTIONS: Array<{
  value: LocalClientSystem;
  label: string;
  description: string;
  Icon: typeof Apple;
}> = [
  {
    value: "mac",
    label: "macOS",
    description: "适用于 MacBook、iMac 等 Apple 设备。",
    Icon: Apple,
  },
  {
    value: "win",
    label: "Windows",
    description: "适用于 Windows 工作站或笔记本。",
    Icon: Monitor,
  },
];

const LOCAL_CLIENT_SYSTEM_LABEL: Record<LocalClientSystem, string> = {
  mac: "macOS",
  win: "Windows",
};

function buildLocalClientInstallPrompt() {
  return `请帮我配置 ClawPro 外部 Agent 接入，将外部 Agent（如 CodeBuddy / WorkBuddy / Claude Code / Codex / iMate / KnotBot）接入企业管理。

接入要求：
1. 先识别当前可用的接入方式，以及底层 Agent 类型（OpenClaw / Hermes 等）。
2. 检查系统环境、企业授权、接入凭证，以及插件目录、扩展管理入口或云端授权入口是否可用。
3. 按 Agent 提示完成企业登录和授权绑定；如缺少企业内部安装包、下载地址或接入凭证，请明确提示我提供，不要使用不可信来源。
4. 完成授权绑定后，检查外部 Agent 是否能向 ClawPro / Hatchery 同步基本信息、接入状态和已安装企业 Skill。

完成后请返回：
- 接入配置状态
- 插件版本、安装路径或外部 Agent 接入信息
- 当前接入状态
- 仍需我手动处理的事项`;
}

async function copyTextToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    // Fall back to a selected textarea for browsers that block the Clipboard API.
  }

  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.setAttribute("readonly", "");
  textarea.style.position = "fixed";
  textarea.style.left = "-9999px";
  textarea.style.top = "0";
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  textarea.setSelectionRange(0, text.length);
  const copied = document.execCommand("copy");
  document.body.removeChild(textarea);
  return copied;
}

// 8 种状态配置
type OpenClawStatus = "creating" | "createFail" | "running" | "shutdown" | "loading" | "loadFail" | "maintaining" | "pending" | "rollingBack";
const RUNNING_ONLY_ACTION_STATUSES: OpenClawStatus[] = ["running"];
const RENAME_ALLOWED_STATUSES: OpenClawStatus[] = ["running", "shutdown"];
const canRunOnlyAction = (status: OpenClawStatus) => RUNNING_ONLY_ACTION_STATUSES.includes(status);
const canRenameStatus = (status: OpenClawStatus) => RENAME_ALLOWED_STATUSES.includes(status);



// ────────────────────────────────────────────────────────────────────
// 创建弹窗内"类型 / 角色"胶囊单选 —— 统一样式（避免每处复制 13 处 className）
// 视觉：未选中白底深字描边，选中变实心黑底白字
// ────────────────────────────────────────────────────────────────────
function PillRadioOption({
  value,
  id,
  children,
}: {
  value: string;
  id: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center">
      <RadioGroupItem value={value} id={id} className="peer sr-only" />
      <Label
        htmlFor={id}
        className="flex items-center justify-center whitespace-nowrap h-6 rounded-full border border-[var(--border)] bg-[var(--card)] px-3 text-xs font-medium text-[var(--text-emphasis)] hover:bg-[var(--accent)] hover:border-[var(--border-control)] cursor-pointer peer-data-[state=checked]:bg-[var(--text-emphasis)] peer-data-[state=checked]:text-white peer-data-[state=checked]:border-[var(--text-emphasis)] peer-data-[state=checked]:hover:bg-[var(--text-emphasis)] peer-data-[state=checked]:hover:text-white transition-colors"
      >
        {children}
      </Label>
    </div>
  );
}

// ────────────────────────────────────────────────────────────────────
// 批量切换弹窗「目标角色」列下拉：默认回填当前角色（语义=保持不变）。
// 下拉首项即「当前角色类型」（选中=不切换），不再单独提供「保持不变」文案项。
// ────────────────────────────────────────────────────────────────────
// Radix Select 的 Item value 不允许为空串，用哨兵值表示「当前角色（不切换）」
const BATCH_NO_SWITCH = "__no_switch__";

interface OpenClawItem {
  id: string;
  instanceId: string;
  name: string;
  status: OpenClawStatus;
  createdAt: string;
  model: string;
  modelVersion: string;
  channels: any[];
  skills: any[];
  standards?: any[];
  op?: string; // 操作标记：restart, reinstall
  roleName?: string; // 角色名称
  distributedRoleVersion?: string; // 角色版本号（x.y 格式）
  roles?: AgentRoleSlot[]; // 多角色实例的 slot 列表（main + sub Agent），缺省时退回单角色语义
  memoryStatus?: 'none' | 'free' | 'pro'; // 记忆状态
  agentType?: "openclaw" | "hermes" | "lightclawace" | "localagent"; // Agent 类型
  /** 当前运行的 Agent 版本（用于与目标镜像版本对比） */
  version?: string;
  creator?: string; // 创建人邮箱（与 mockData.MOCK_OPENCLAW_LIST 字段对齐）
  /** 归属人邮箱。用户端自建场景下与 creator 相同；管控端代建时为目标用户邮箱 */
  owner?: string;
  groupId?: string;   // 所属组织 ID（多组织模式）
  groupName?: string; // 所属组织名称（多组织模式）
  /** 关联项目 ID 列表（创建时可多选；实例会额外下发这些项目在「项目资产管理」中的 Agent 工具） */
  projectIds?: string[];
  /** 关联项目名称列表（与 projectIds 一一对应，用于展示） */
  projectNames?: string[];
  // ===== 计费相关字段（与 lib/openclawStore.AgentItem 对齐）=====
  billingMode?: "subscription" | "payg"; // 计费模式
  /** 小时单价（元/小时），payg 模式下生效 */
  hourlyRate?: number;
  /** 累计运行分钟数（仅运行态累加） */
  runningMinutes?: number;
  /** 最近一次进入运行态的时间戳（ISO 字符串，用于计时） */
  lastStartedAt?: string;
  /** 累计费用（元）= 运行小时 × 单价 */
  accumulatedCost?: number;
  localProduct?: "Claude" | "CodeBuddy" | "WorkBuddy" | "Codex";
  localResourceSyncStatus?: "syncing" | "synced";
  /** 外部 Agent 最近一次向 Hatchery 同步信息的时间 */
  lastReportedAt?: string;
  shutdownType?: "keep" | "stopCharging"; // 关机方式：keep=标准关机，stopCharging=释放资源关机
  // 组织变更处理状态
  // archived：管理员已保留并关机，无需用户处理，卡片保持常规关机态，仅保留「您已不在该组织」提示
  groupChangeStatus?: "pending" | "pendingConfirm" | "transferring" | "rejected" | "expired" | "migrating" | "archived";
  groupChangeOriginalGroup?: string;
  groupChangeTransferTarget?: string;
  // 仅可移交、不可迁移（如原组织已被删除）：true 时隐藏「迁移到新组织」按钮
  groupChangeTransferOnly?: boolean;
  // 迁移到新组织的执行阶段：migrating=迁移组织中 / success=迁移成功 / fail=迁移失败
  migrationPhase?: "migrating" | "success" | "fail";
  // 移交给其他用户的执行阶段：pendingConfirm=移交待对方确认 / transferring=移交中 / done=移交完成 / rejected=对方拒绝移交 / failed=对方已接收但移交失败
  transferPhase?: "pendingConfirm" | "transferring" | "done" | "rejected" | "failed";
  // 别人移交给我的
  incomingTransfer?: { fromUser: string; instanceName: string };
  // ===== 共享范围 =====
  /** 共享范围类型：private=仅自己，shared=共享到分组或个人 */
  shareScope?: "private" | "shared";
  /** 共享的分组 ID 列表 */
  shareGroupIds?: string[];
  /** 共享的分组名称列表（用于展示，与 shareGroupIds 一一对应） */
  shareGroupNames?: string[];
  /** 共享的用户 ID 列表 */
  shareUserIds?: string[];
  /** 共享的用户名称列表（用于展示，与 shareUserIds 一一对应） */
  shareUserNames?: string[];
}

function getExternalAgentAccessTarget(claw: Pick<OpenClawItem, "localProduct">): LocalClientAccessTarget | undefined {
  if (claw.localProduct === "Claude") return "Claude Code";
  return claw.localProduct;
}

interface LocalAgentRemoveOperation {
  id: string;
  instanceId: string;
  name: string;
  localProduct?: "Claude" | "CodeBuddy" | "WorkBuddy" | "Codex";
  operation: "remove-local-agent";
  preserveInstalledAssets: true;
  createdAt: string;
}

function recordLocalAgentRemoveOperation(claw: OpenClawItem) {
  try {
    const raw = localStorage.getItem(LOCAL_AGENT_REMOVE_OPERATIONS_KEY);
    const current = raw ? JSON.parse(raw) as LocalAgentRemoveOperation[] : [];
    const operation: LocalAgentRemoveOperation = {
      id: claw.id,
      instanceId: claw.instanceId,
      name: claw.name,
      localProduct: claw.localProduct,
      operation: "remove-local-agent",
      preserveInstalledAssets: true,
      createdAt: new Date().toISOString(),
    };
    const next: LocalAgentRemoveOperation[] = [
      operation,
      ...current,
    ].slice(0, 50);
    localStorage.setItem(LOCAL_AGENT_REMOVE_OPERATIONS_KEY, JSON.stringify(next));
    window.dispatchEvent(new StorageEvent("storage", { key: LOCAL_AGENT_REMOVE_OPERATIONS_KEY }));
  } catch {
    // 本地操作记录失败不阻塞 UI 移除。
  }
}

// ==================== 多组织模式类型与Mock数据 ====================
type UserGroupMode = "normal" | "multi-group";

interface UserGroup {
  id: string;
  name: string;         // 组织全称
  type: "department" | "custom"; // 部门 / 自定义组织
  isPrimary: boolean;   // 是否主部门
  depth: number;        // 层级深度
  permissions: {
    allowTerminal: boolean;
    allowChatView: boolean;
    agentTypes: ("openclaw" | "hermes" | "lightclawace" | "localagent")[];
    roles: string[];    // 可用角色列表
    panelAccess: "full" | "partial" | "limited"; // 详细配置面板访问级别
  };
}

// Alice 所属的 3 个组织
// 注：permissions.roles 的角色名必须与 MOCK_ROLES（lib/mockData.ts）中的 role.name 完全一致，
// 否则创建弹窗里的角色身份过滤 (selectedGroup.permissions.roles.includes(role.name)) 会全部 miss。
// MOCK_ROLES 当前 visible:true 的角色：行业分析师 / 开发工程师 / 设计师 / 项目经理 / 内容创作者。
// "通用助手" 是兜底选项，不通过 permissions.roles 控制（弹窗中始终展示）。
const MOCK_USER_GROUPS: UserGroup[] = [
  {
    id: "grp-fe",
    name: "A公司 / 技术部 / 前端组",
    type: "department",
    isPrimary: true,
    depth: 3,
    permissions: {
      allowTerminal: true,
      allowChatView: true,
      agentTypes: ["openclaw", "hermes", "lightclawace", "localagent"],
      roles: ["行业分析师", "开发工程师", "设计师", "项目经理", "内容创作者"],
      panelAccess: "full",
    },
  },
  {
    id: "grp-ai",
    name: "A公司 / 技术部 / AI 组",
    type: "department",
    isPrimary: false,
    depth: 3,
    permissions: {
      allowTerminal: true,
      allowChatView: true,
      agentTypes: ["openclaw", "hermes", "lightclawace", "localagent"],
      roles: ["行业分析师", "开发工程师", "项目经理"],
      panelAccess: "partial",
    },
  },
  {
    id: "grp-custom",
    name: "前端研发同学",
    type: "custom",
    isPrimary: false,
    depth: 1,
    permissions: {
      allowTerminal: false,
      allowChatView: false,
      agentTypes: ["openclaw", "localagent"],
      roles: ["开发工程师", "设计师"],
      panelAccess: "limited",
    },
  },
];

// ===== 共享范围 Mock 数据：二级树形结构（分组 → 成员） =====
interface ShareGroupNode {
  id: string;
  name: string;
  members: { id: string; name: string; email: string }[];
}
const MOCK_SHARE_TREE: ShareGroupNode[] = [
  {
    id: "grp-fe", name: "前端组",
    members: [
      { id: "bob@a.com", name: "Bob", email: "bob@a.com" },
      { id: "carol@a.com", name: "Carol", email: "carol@a.com" },
    ],
  },
  {
    id: "grp-ai", name: "AI 组",
    members: [
      { id: "frank@a.com", name: "Frank", email: "frank@a.com" },
      { id: "grace@a.com", name: "Grace", email: "grace@a.com" },
    ],
  },
  {
    id: "grp-custom", name: "前端研发同学",
    members: [
      { id: "bob@a.com", name: "Bob", email: "bob@a.com" },
      { id: "dave@a.com", name: "Dave", email: "dave@a.com" },
    ],
  },
  {
    id: "grp-ops", name: "运营组",
    members: [
      { id: "dave@a.com", name: "Dave", email: "dave@a.com" },
      { id: "grace@a.com", name: "Grace", email: "grace@a.com" },
    ],
  },
  {
    id: "grp-be", name: "后端研发同学",
    members: [
      { id: "frank@a.com", name: "Frank", email: "frank@a.com" },
      { id: "carol@a.com", name: "Carol", email: "carol@a.com" },
    ],
  },
  {
    id: "grp-qa", name: "测试组",
    members: [
      { id: "henry@a.com", name: "Henry", email: "henry@a.com" },
      { id: "ivy@a.com", name: "Ivy", email: "ivy@a.com" },
    ],
  },
  {
    id: "grp-design", name: "设计组",
    members: [
      { id: "jack@a.com", name: "Jack", email: "jack@a.com" },
      { id: "kate@a.com", name: "Kate", email: "kate@a.com" },
    ],
  },
  {
    id: "grp-pm", name: "产品组",
    members: [
      { id: "leo@a.com", name: "Leo", email: "leo@a.com" },
      { id: "mia@a.com", name: "Mia", email: "mia@a.com" },
    ],
  },
  {
    id: "grp-data", name: "数据组",
    members: [
      { id: "nick@a.com", name: "Nick", email: "nick@a.com" },
      { id: "olivia@a.com", name: "Olivia", email: "olivia@a.com" },
    ],
  },
];

/**
 * 共享展示派生计算——统一「整组 vs 个人」的展示规则。
 *
 * 规则（与 ShareScopeTree 的 isGroupFullyCovered 完全一致）：
 *   - 一个组只要被「整组覆盖」（显式勾选整组，或该组所有成员都被勾选）→ 归并为组名展示；
 *   - 未被整组覆盖的成员 → 作为个人单独展示（并去重，避免一个人出现在多个组里重复显示）。
 *
 * 返回：
 *   - groupNames：要以「组名」展示的分组名称列表
 *   - userNames ：要以「个人」展示的成员名称列表（已剔除归入整组的成员）
 */
const deriveShareDisplay = (
  groupIds: string[],
  userIds: string[],
): { groupNames: string[]; userNames: string[] } => {
  // 1. 找出所有「整组覆盖」的组
  const fullyCoveredGroups = MOCK_SHARE_TREE.filter((g) => {
    if (groupIds.includes(g.id)) return true;
    return g.members.length > 0 && g.members.every((m) => userIds.includes(m.id));
  });
  const groupNames = fullyCoveredGroups.map((g) => g.name);

  // 2. 已被整组覆盖的成员 ID 集合（这些人不再单独作为个人展示）
  const coveredMemberIds = new Set(
    fullyCoveredGroups.flatMap((g) => g.members.map((m) => m.id)),
  );

  // 3. 剩余散选成员（去重）→ 以个人展示
  const userNames: string[] = [];
  const seen = new Set<string>();
  for (const uid of userIds) {
    if (coveredMemberIds.has(uid) || seen.has(uid)) continue;
    seen.add(uid);
    const member = MOCK_SHARE_TREE.flatMap((g) => g.members).find((m) => m.id === uid);
    userNames.push(member?.name ?? uid);
  }

  return { groupNames, userNames };
};

// 获取默认选中的组织：优先选主部门，否则选层级最浅的
const getDefaultGroup = (groups: UserGroup[]): UserGroup => {
  const primary = groups.find(g => g.isPrimary);
  if (primary) return primary;
  return [...groups].sort((a, b) => a.depth - b.depth)[0];
};

// 解析管控端写入的「共享 agent」按组规则 JSON（admin_share_agent_group_rules）。
// 容错：非法/损坏数据一律回退为空数组（即仅按全员开关 admin_allow_share_agent 判定）。
const parseShareAgentGroupRules = (raw: string | null): PolicyRule<boolean>[] => {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed
      .filter((r): r is PolicyRule<boolean> => !!r && Array.isArray(r.groupIds) && r.groupIds.length > 0)
      .map((r) => ({ id: String(r.id), groupIds: r.groupIds.map(String), value: !!r.value }));
  } catch {
    return [];
  }
};

// 状态视觉配置已迁移到 components/agent/StatusBadge.tsx，本页直接使用 AGENT_STATUS_DISABLED 判定禁用

const getRecentLocalAgentReportedAt = () => {
  const date = new Date(Date.now() - 60 * 60 * 1000);
  const pad = (value: number) => String(value).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
};

export default function MyOpenClaw() {
  const [, navigate] = useLocation();
  const { isCreateAgentDisabled, createAgentDisabledTip } = useServiceStatus();
  const [claws, setClawsRaw] = useState<OpenClawItem[]>(() => {
    // 1) 读 store，先剔除历史残留的演示用 gc-* / 外部 Agent 示例卡片，避免旧版本缓存干扰展示
    const base = (loadClawList() as OpenClawItem[]).filter((c) =>
      !c.id.startsWith("gc-") &&
      ![
        "oc-016",
        "oc-local-agent-demo",
        "oc-local-codebuddy-demo",
        "oc-local-workbuddy-demo",
        "oc-local-workbuddy-offline-demo",
        "oc-external-imate-demo",
        "oc-external-knotbot-demo",
        "oc-external-claude-demo",
        "oc-external-codex-demo",
        "oc-external-hermes-demo",
        "oc-external-openclaw-demo",
      ].includes(c.id)
    );
    const localAgentDemos: OpenClawItem[] = [{
      id: "oc-external-claude-demo",
      instanceId: "external-claude-01",
      name: "Claude Code 外部 Agent",
      status: "running",
      agentType: "localagent",
      localProduct: "Claude",
      localResourceSyncStatus: "synced",
      lastReportedAt: getRecentLocalAgentReportedAt(),
      createdAt: "2026-07-31 10:58:22",
      model: "Claude Code",
      modelVersion: "claude-code-1.0.90",
      channels: [],
      skills: ["code-review 2.1.0", "project-summary 1.2.0"],
      standards: ["Agent 安全合规基线", "研发协作规范"],
      groupId: "grp-fe",
      groupName: "大计算 ClawPro / 计算产品中心",
      billingMode: "payg",
      hourlyRate: 0,
      runningMinutes: 0,
    }, {
      id: "oc-external-codex-demo",
      instanceId: "external-codex-01",
      name: "Codex 外部 Agent",
      status: "running",
      agentType: "localagent",
      localProduct: "Codex",
      localResourceSyncStatus: "synced",
      lastReportedAt: getRecentLocalAgentReportedAt(),
      createdAt: "2026-07-31 11:58:51",
      model: "Codex",
      modelVersion: "codex-0.28.0",
      channels: [],
      skills: ["task-planner 1.0.0", "project-sync 1.1.0"],
      standards: ["项目协作规范"],
      groupId: "grp-fe",
      groupName: "大计算 ClawPro / 计算产品中心",
      billingMode: "payg",
      hourlyRate: 0,
      runningMinutes: 0,
    }, {
      id: "oc-external-hermes-demo",
      instanceId: "external-hermes-01",
      name: "Hermes 外部 Agent",
      status: "running",
      agentType: "localagent",
      localResourceSyncStatus: "synced",
      lastReportedAt: getRecentLocalAgentReportedAt(),
      createdAt: "2026-07-31 10:58:22",
      model: "Hermes",
      modelVersion: "hermes-0.13.0",
      channels: [],
      skills: ["code-review 2.1.0", "project-summary 1.2.0"],
      standards: ["Agent 安全合规基线", "研发协作规范"],
      groupId: "grp-fe",
      groupName: "大计算 ClawPro / 计算产品中心",
      billingMode: "payg",
      hourlyRate: 0,
      runningMinutes: 0,
    }, {
      id: "oc-external-openclaw-demo",
      instanceId: "external-openclaw-01",
      name: "OpenClaw 外部 Agent",
      status: "running",
      agentType: "localagent",
      localResourceSyncStatus: "synced",
      lastReportedAt: "2026-07-20 20:50:44",
      createdAt: "2026-07-28 20:50:44",
      model: "OpenClaw",
      modelVersion: "2026.4.23",
      channels: [],
      skills: ["task-planner 1.0.0", "project-sync 1.1.0"],
      standards: ["项目协作规范"],
      groupId: "grp-fe",
      groupName: "默认",
      billingMode: "payg",
      hourlyRate: 0,
      runningMinutes: 0,
    }, {
      id: "oc-local-workbuddy-demo",
      instanceId: "local-workbuddy-01",
      name: "WorkBuddy-运营笔记本",
      status: "running",
      agentType: "localagent",
      localProduct: "WorkBuddy",
      localResourceSyncStatus: "syncing",
      lastReportedAt: getRecentLocalAgentReportedAt(),
      createdAt: "2026-04-06 09:30:00",
      model: "WorkBuddy",
      modelVersion: "workbuddy-2.3.1",
      channels: [],
      skills: ["doc-summarizer 1.3.0", "meeting-summary 1.3.0"],
      standards: ["Agent 安全合规基线", "交付协作规范"],
      groupId: "grp-ops",
      groupName: "运营组",
      billingMode: "payg",
      hourlyRate: 0,
      runningMinutes: 360,
    }, {
      id: "oc-local-workbuddy-offline-demo",
      instanceId: "local-workbuddy-02",
      name: "WorkBuddy-离线笔记本",
      status: "running",
      agentType: "localagent",
      localProduct: "WorkBuddy",
      localResourceSyncStatus: "synced",
      lastReportedAt: "2026-06-22 09:12:40",
      createdAt: "2026-04-07 10:10:00",
      model: "WorkBuddy",
      modelVersion: "workbuddy-2.2.0",
      channels: [],
      skills: ["doc-summarizer 1.3.0"],
      standards: ["Agent 安全合规基线"],
      groupId: "grp-custom",
      groupName: "前端研发同学",
      billingMode: "payg",
      hourlyRate: 0,
      runningMinutes: 0,
    }];
    const baseWithoutLocalDemos = base.filter((c) => c.agentType !== "localagent");
    const baseWithLocalAgent = [...localAgentDemos, ...baseWithoutLocalDemos];
    // 2) 演示用：组织变更状态 mock 卡片（不持久化到 store，仅页面内追加，避免反复 saveClawList 越积越多）
    const groupChangeCards: OpenClawItem[] = [
      {
        id: "gc-pending", instanceId: "ins-gc-pending", name: "API 调试助手", status: "shutdown",
        createdAt: "2026-03-20", model: "混元 TurboS", modelVersion: "latest", channels: [], skills: [],
        agentType: "openclaw", version: "2026.4.20",
        groupName: "后端研发同学", groupChangeStatus: "pending", groupChangeOriginalGroup: "后端研发同学",
      },
      {
        // 演示：用户已不在该组织，但管理员已保留并关机处理，卡片保持常规关机态，不再展示迁移/移交按钮
        id: "gc-archived", instanceId: "ins-gc-archived", name: "日报工具", status: "shutdown",
        createdAt: "2026-03-18", model: "混元 Turbo", modelVersion: "latest", channels: [], skills: [],
        agentType: "openclaw", version: "2026.4.20",
        groupName: "前端组", groupChangeStatus: "archived", groupChangeOriginalGroup: "前端组",
      },
      {
        id: "gc-incoming", instanceId: "ins-gc-in", name: "数据分析助手", status: "shutdown",
        createdAt: "2026-03-22", model: "Claude Sonnet 4", modelVersion: "latest", channels: [], skills: [],
        groupName: "AI 组", incomingTransfer: { fromUser: "alice@acompany.com", instanceName: "数据分析助手" },
      },
      {
        // 演示：用户已不在该组织（原组织已被删除，无可迁移的新组织），仅可「移交给其他用户」；
        // 每次发起移交都固定为「对方接收后，移交最终处理失败」，失败后移交按钮重新出现，可再次发起。
        id: "gc-transfer-fail", instanceId: "ins-gc-transfer-fail", name: "运营报表助手", status: "shutdown",
        createdAt: "2026-03-15", model: "混元 Turbo", modelVersion: "latest", channels: [], skills: [],
        agentType: "openclaw", version: "2026.4.20",
        groupName: "运营组", groupChangeStatus: "pending", groupChangeOriginalGroup: "运营组", groupChangeTransferOnly: true,
      },
    ];
    // 3) 演示用：共享实例 mock 卡片（补充 shareScope + creator）
    const sharedDemoCards: OpenClawItem[] = [
      // 演示「整组全选」：勾选了「前端组」全部成员（Bob、Carol）→ 卡片展示归并为组名「前端组」
      (() => {
        const groupIds = ["grp-fe"];
        const userIds: string[] = [];
        const { groupNames, userNames } = deriveShareDisplay(groupIds, userIds);
        return {
          id: "gc-shared-01", instanceId: "ins-shared-01", name: "团队协作助手", status: "running" as OpenClawStatus,
          createdAt: "2026-03-25", model: "DeepSeek V3", modelVersion: "0324", channels: ["飞书"], skills: [],
          creator: "bob@acompany.com", shareScope: "shared" as const, shareGroupIds: groupIds, shareGroupNames: groupNames,
          shareUserIds: userIds, shareUserNames: userNames, groupName: "前端组", billingMode: "payg" as const, hourlyRate: 1.5, runningMinutes: 1200,
        };
      })(),
      // 演示「组内部分成员」：只选了某些散人（未选满任何组）→ 卡片展示逐个个人名字
      (() => {
        const groupIds: string[] = [];
        const userIds = ["bob@a.com", "dave@a.com", "grace@a.com"];
        const { groupNames, userNames } = deriveShareDisplay(groupIds, userIds);
        return {
          id: "gc-shared-02", instanceId: "ins-shared-02", name: "客服知识库助手", status: "running" as OpenClawStatus,
          createdAt: "2026-03-28", model: "混元 Turbo", modelVersion: "latest", channels: ["企业微信"], skills: [],
          creator: "carol@acompany.com", shareScope: "shared" as const, shareGroupIds: groupIds, shareGroupNames: groupNames,
          shareUserIds: userIds, shareUserNames: userNames, groupName: "运营组", billingMode: "subscription" as const,
        };
      })(),
    ];
    return [...baseWithLocalAgent, ...groupChangeCards, ...sharedDemoCards];
  });
  // 包装 setClaws：写回 store 时剔除演示用 gc-* 卡片，避免污染 localStorage
  const setClaws = (v: OpenClawItem[] | ((prev: OpenClawItem[]) => OpenClawItem[])) => {
    setClawsRaw((prev) => {
      const next = typeof v === "function" ? v(prev) : v;
      saveClawList(next.filter((c) => !c.id.startsWith("gc-")));
      notifyClawListChange();
      return next;
    });
  };
  const [showCreate, setShowCreate] = useState(false);
  const [localClientAccessDialog, setLocalClientAccessDialog] = useState<{
    step: LocalClientAccessStep;
    system?: LocalClientSystem;
    targets: LocalClientAccessTarget[];
  } | null>(null);
  const [newName, setNewName] = useState("");
  const [showQuickStart, setShowQuickStart] = useState(true);

  // ===== 组织变更处理 =====
  const [showMigrateDialog, setShowMigrateDialog] = useState<{ id: string; instanceName: string } | null>(null);
  const [showTransferDialog, setShowTransferDialog] = useState<{ id: string; instanceName: string; groupName: string } | null>(null);

  // ===== 多组织模式 =====
  const [groupMode, setGroupMode] = useState<UserGroupMode>(() => {
    return (localStorage.getItem("openclaw_group_mode") as UserGroupMode) || "normal";
  });
  const handleGroupModeChange = (mode: UserGroupMode) => {
    setGroupMode(mode);
    localStorage.setItem("openclaw_group_mode", mode);
    // 通知同页面其他组件
    window.dispatchEvent(new StorageEvent("storage", { key: "openclaw_group_mode", newValue: mode }));
  };
  // 监听来自 TenantLayout 下拉菜单的组织模式切换
  useEffect(() => {
    const handleStorage = (e: StorageEvent) => {
      if (e.key === "openclaw_group_mode") {
        setGroupMode((e.newValue as UserGroupMode) || "normal");
      }
    };
    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, []);

  // [数据备份需求 · Demo] 幂等注入 3 个示例 Agent，分别对应 none/creating/ready 三种备份点状态。
  // 联调真实接口后整段删除即可。
  useEffect(() => {
    ensureBackupDemoAgents();
  }, []);
  // 创建弹窗当前选中组织（多组织模式下用）
  const [selectedGroup, setSelectedGroup] = useState<UserGroup>(() => getDefaultGroup(MOCK_USER_GROUPS));

  // 视图模式
  const [viewMode, setViewMode] = useState<"card" | "chat">(() => {
    return (localStorage.getItem("openclaw_view_mode") as "card" | "chat") || "chat";
  });
  const handleViewModeChange = (mode: "card" | "chat") => {
    setViewMode(mode);
    localStorage.setItem("openclaw_view_mode", mode);
  };

  // 「新增角色」独立弹窗：记录目标 Agent（id / name / 现有角色位 / 分组白名单），提交后直接落库
  const [standaloneAddRole, setStandaloneAddRole] = useState<{ id: string; name: string; roles: AgentRoleSlot[]; allowedRoleNames?: string[] } | null>(null);
  // 编辑角色交互方案切换（方案1=当前弹窗 / 方案2=独立批量切换弹窗 / 方案3=左目录右切换）
  const editRoleScheme = 1 as const;
  // 「切换角色」独立批量切换弹窗：不打开角色管理抽屉，直接对该 Agent 的全部角色位做批量切换。
  // 弹窗内部交互态（目标选择 / 右栏激活 / 技能勾选 / 配置加载动画）已收敛到共享组件 BatchSwitchRoleDialog，
  // 本页仅保留「打开数据源」与落库 / 后台标记回调。
  const [standaloneBatchSwitch, setStandaloneBatchSwitch] = useState<{ id: string; name: string; slots: AgentRoleSlot[]; allowedRoleNames?: string[] } | null>(null);
  // 「角色切换中」的 Agent：agentId → 本次正在切换的角色数量。
  // 进度弹窗被「我知道了」关闭后（后台仍在切换），卡片据此展示「角色切换中（N）」状态提示，
  // 切换完成落库时清除。
  const [switchingRoleAgents, setSwitchingRoleAgents] = useState<Record<string, number>>({});
  // 存储切换进度的详细数据，用于点击胶囊时重新打开进度弹窗
  const [switchingProgressData, setSwitchingProgressData] = useState<Record<string, { agentName: string; items: import("@/components/agent/RoleConfigProgress").RoleConfigProgressItem[] }>>({});
  const markRoleSwitching = useCallback((agentId: string, count: number) => {
    setSwitchingRoleAgents((prev) => ({ ...prev, [agentId]: count }));
  }, []);
  const storeSwitchingProgress = useCallback((agentId: string, agentName: string, items: import("@/components/agent/RoleConfigProgress").RoleConfigProgressItem[]) => {
    setSwitchingProgressData((prev) => ({ ...prev, [agentId]: { agentName, items } }));
  }, []);
  const clearRoleSwitching = useCallback((agentId: string) => {
    setSwitchingRoleAgents((prev) => {
      if (!(agentId in prev)) return prev;
      const next = { ...prev };
      delete next[agentId];
      return next;
    });
    setSwitchingProgressData((prev) => {
      if (!(agentId in prev)) return prev;
      const next = { ...prev };
      delete next[agentId];
      return next;
    });
  }, []);
  // 「角色新增中」状态：与 switchingRoleAgents 对齐——新增角色进度弹窗被「我知道了」关闭后，
  // 卡片上继续提示「角色新增中（N）」，直至进度走完落库后清除。
  // 点击「角色切换中」胶囊时弹出切换进度弹窗的目标 agentId
  const [switchingProgressDialogAgent, setSwitchingProgressDialogAgent] = useState<string | null>(null);
  const [addingRoleAgents, setAddingRoleAgents] = useState<Record<string, number>>({});
  const markRoleAdding = useCallback((agentId: string, count: number) => {
    setAddingRoleAgents((prev) => ({ ...prev, [agentId]: count }));
  }, []);
  const clearRoleAdding = useCallback((agentId: string) => {
    setAddingRoleAgents((prev) => {
      if (!(agentId in prev)) return prev;
      const next = { ...prev };
      delete next[agentId];
      return next;
    });
  }, []);
  // 已经单独给过「角色切换成功」提示的角色位 slotId（方案1 抽屉内逐行「确认」场景）。
  // 抽屉关闭时统一提交会再走一次汇总提示，这里用于剔除已提示过的行，避免同一次切换重复提示。
  const announcedSwitchSlotsRef = useRef<Set<string>>(new Set());
  // 「新增角色」/「切换角色（方案1 单角色位）」确认后的独立配置进度弹窗：
  // 与批量切换弹窗内联加载态同一套进度 UI 与节奏，走完 100% 后执行落库回调。
  const roleConfigProgress = useRoleConfigProgress();

  // 全屏模式
  const [isFullscreen, setIsFullscreen] = useState(false);
  const handleToggleFullscreen = () => setIsFullscreen(prev => !prev);

  // Agent 类型
  const [agentType, setAgentType] = useState<"openclaw" | "hermes" | "lightclawace" | "localagent">("openclaw");

  // 角色选择
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const visibleRoles = MOCK_ROLES.filter((r) => r.visible);

  // 创建弹窗：高级选项折叠态（默认全部收起，保持表单简洁）
  const [agentTypeOpen, setAgentTypeOpen] = useState(false);
  const [roleOpen, setRoleOpen] = useState(false);

  // ===== 创建弹窗：项目（可多选，默认不选） =====
  // 项目池来自共享 groupStore（管控端「项目资产管理」创建的项目实时可见）；
  // 选中后，创建的实例除使用组织配置外，还会额外下发所选项目在「项目资产管理」里的 Agent 工具。
  const [projectPool, setProjectPool] = useState<{ id: string; name: string }[]>(
    () => groupStore.getAll().filter((g) => g.source === "project" && g.parentId === null).map((g) => ({ id: g.id, name: g.name })),
  );
  useEffect(
    () => groupStore.subscribe(() =>
      setProjectPool(
        groupStore.getAll().filter((g) => g.source === "project" && g.parentId === null).map((g) => ({ id: g.id, name: g.name })),
      ),
    ),
    [],
  );
  const [selectedProjectIds, setSelectedProjectIds] = useState<string[]>([]);
  const [projectOpen, setProjectOpen] = useState(false);
  const toggleProject = (id: string) =>
    setSelectedProjectIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));

  // ===== 创建弹窗：共享范围 =====
  const [shareScope, setShareScope] = useState<"private" | "shared">("private");
  const [shareScopeOpen, setShareScopeOpen] = useState(false);
  const [shareGroupIds, setShareGroupIds] = useState<string[]>([]);
  const [shareUserIds, setShareUserIds] = useState<string[]>([]);
  const [shareScopeError, setShareScopeError] = useState("");
  // 二级下拉（Popover）展开状态——受控，便于点击 +N 标签时展开
  const [shareTreeOpen, setShareTreeOpen] = useState(false);

  // ===== 更改共享范围弹窗 =====
  const [shareScopeDialog, setShareScopeDialog] = useState<{ id: string; name: string } | null>(null);
  const [editShareScope, setEditShareScope] = useState<"private" | "shared">("private");
  const [editShareGroupIds, setEditShareGroupIds] = useState<string[]>([]);
  const [editShareUserIds, setEditShareUserIds] = useState<string[]>([]);
  const [editShareScopeError, setEditShareScopeError] = useState("");
  // 更改弹窗——二级下拉（Popover）展开状态，受控
  const [editShareTreeOpen, setEditShareTreeOpen] = useState(false);


  // 确认弹窗
  const [deleteConfirm, setDeleteConfirm] = useState<{ id: string; name: string; status: OpenClawStatus; memoryStatus?: 'none' | 'free' | 'pro'; agentType?: OpenClawItem["agentType"]; localProduct?: OpenClawItem["localProduct"] } | null>(null);
  const [deleteConfirmInput, setDeleteConfirmInput] = useState("");
  const [restartConfirm, setRestartConfirm] = useState<{ id: string; name: string } | null>(null);
  const [restartFullServer, setRestartFullServer] = useState(false);
  const [reinstallConfirm, setReinstallConfirm] = useState<{ id: string; name: string } | null>(null);
  const [reinstallConfirmInput, setReinstallConfirmInput] = useState("");
  const [renameConfirm, setRenameConfirm] = useState<{ id: string; name: string } | null>(null);
  const [renameInput, setRenameInput] = useState("");
  // 切换角色：统一直达弹窗（角色标签 / 三点菜单点击后都进这一个 Dialog，不再有外挂下拉/子菜单）。
  //   · 单角色实例：沿用「切换为」Pill 单选 + 角色介绍卡（零回归）。
  //   · 多角色实例：映射表（N→N）——每个角色位独立成行，各自挂一个「切换为」下拉（可各选不同目标），
  //     一次性批量提交；重名角色位靠 slotId 精确定位、各自独立，不合并/不编号。
  const [switchRoleDialog, setSwitchRoleDialog] = useState<{ id: string; name: string; roleName: string; roleCount?: number; allowedRoleNames?: string[]; roles?: AgentRoleSlot[] } | null>(null);
  // 单角色分支的目标（Pill 单选）
  const [switchRoleTarget, setSwitchRoleTarget] = useState<Role | "__general__" | null>(null);
  // 多角色分支的逐行目标：slotId → 目标（Role / "__general__" / null）；null 表示「保持不变」（该行不切换）
  const [switchRoleTargets, setSwitchRoleTargets] = useState<Record<string, Role | "__general__" | null>>({});
  // 多角色映射表：当前展开查看目标角色介绍的行（默认全部收起，保持弹窗紧凑）
  const [expandedSlotId, setExpandedSlotId] = useState<string | null>(null);
  // 多角色映射表：某行「确认」后的加载态（loading 期间禁用交互），加载完成后把该行 editSlots.roleName 真正切换过来
  const [confirmingSlotId, setConfirmingSlotId] = useState<string | null>(null);
  // 多角色映射表：已完成「确认」切换的行（角色列即时展示为目标角色的名称/头像）；不影响底部计数与统一提交（仍以 switchRoleTargets 为准）
  const [confirmedSlotIds, setConfirmedSlotIds] = useState<Set<string>>(new Set());
  // 多角色映射表：删除角色位的二次确认弹窗（记录待删除行的 slotId + 角色名）
  const [deleteSlotConfirm, setDeleteSlotConfirm] = useState<{ slotId: string; roleName: string } | null>(null);
  // 多角色映射表：「修改角色名称」弹窗（记录待改名行的 slotId + 原角色名）
  const [renameSlotTarget, setRenameSlotTarget] = useState<{ slotId: string; roleName: string } | null>(null);
  // 修改角色名称弹窗：输入框当前值 + 校验错误提示
  const [renameValue, setRenameValue] = useState("");
  const [renameError, setRenameError] = useState("");
  // 技能选择面板自适应宽度：ref 挂载到各卡片容器，打开时测量并传入 PopoverContent
  const switchSlotSkillPanelRef = useRef<HTMLDivElement>(null);
  const [skillPanelWidth, setSkillPanelWidth] = useState<Record<string, number>>({});
  // 切换角色弹窗（Sheet 内逐行 per-slot）选中的预装技能子集：slotId → 选中技能名集合。
  // 选中某目标角色时默认全选其全部技能；仅「编辑预装技能」态下可增删。
  const [switchSlotSkillNames, setSwitchSlotSkillNames] = useState<Record<string, Set<string>>>({});
  // 切换角色弹窗：当前打开「编辑预装技能」Popover 的角色位 slotId（null=未打开）
  const [switchSlotSkillPopoverSlot, setSwitchSlotSkillPopoverSlot] = useState<string | null>(null);
  // 切换角色弹窗：进入编辑前的技能勾选快照（「取消」回滚用）
  const [switchSlotSkillsBackup, setSwitchSlotSkillsBackup] = useState<Set<string>>(new Set());
  // 多角色映射表：本地可编辑的角色位副本（支持新增 / 删除角色位）。
  //   打开多角色弹窗时用 switchRoleDialog.roles 深拷贝初始化；确认时与原 roles 对比计算增/删/改统一提交。
  //   新增行 roleName === "" 表示「待选择」，需在该行「切换为」下拉里选定具体角色后方可提交。
  const [editSlots, setEditSlots] = useState<AgentRoleSlot[]>([]);
  // 单角色实例正在操作的角色名（用于展示 + 过滤"切换为"候选 + 判断能否切回通用助手）
  const currentSlotRoleName = switchRoleDialog?.roleName ?? "通用助手";
  // 是否以「多设计师协作助手」同款映射表格式（抽屉）渲染角色管理：
  //   只要携带角色位数据（roles.length >= 1）即走映射表抽屉，含单角色 / 多角色 / 受限分组三类。
  //   打开抽屉时已对所有实例（含单角色 / 受限）合成 roles，故这里统一按 roles 是否存在判定。
  const isMappingRoleDialog = (
    dialog: { allowedRoleNames?: string[]; roles?: AgentRoleSlot[] } | null | undefined
  ): boolean => {
    const slots = dialog?.roles;
    return !!slots && slots.length >= 1;
  };
  // 打开弹窗时初始化：多角色实例下每行默认「保持不变」(null)，并收起所有介绍卡
  useEffect(() => {
    if (!switchRoleDialog) { setSwitchRoleTargets({}); setExpandedSlotId(null); setEditSlots([]); setConfirmingSlotId(null); setConfirmedSlotIds(new Set()); setDeleteSlotConfirm(null); setRenameSlotTarget(null); return; }
    const slots = switchRoleDialog.roles;
    if (slots && isMappingRoleDialog(switchRoleDialog)) {
      const init: Record<string, Role | "__general__" | null> = {};
      slots.forEach((s) => { init[s.slotId] = null; });
      setSwitchRoleTargets(init);
      setExpandedSlotId(null);
      setConfirmingSlotId(null);
      setConfirmedSlotIds(new Set());
      // 深拷贝原角色位作为本地可编辑副本（新增 / 删除只作用于副本，确认时统一提交）
      setEditSlots(slots.map((s) => ({ ...s })));
    }
  }, [switchRoleDialog]);
  // 单角色实例：打开弹窗时计算默认可切换角色，避免"确认切换"默认禁用、无默认选中态（多角色/受限映射表走 switchRoleTargets，跳过）
  useEffect(() => {
    if (!switchRoleDialog) return;
    if (isMappingRoleDialog(switchRoleDialog)) return;
    const allowedRoleNames = switchRoleDialog.allowedRoleNames;
    const candidateRoles = allowedRoleNames
      ? visibleRoles.filter((r) => allowedRoleNames.includes(r.name))
      : visibleRoles;
    const switchableRoles = candidateRoles.filter((r) => r.name !== currentSlotRoleName);
    const canSwitchToGeneral = currentSlotRoleName !== "通用助手";
    setSwitchRoleTarget(canSwitchToGeneral ? "__general__" : switchableRoles[0] ?? null);
  }, [switchRoleDialog]);
  // 计算某个角色位可切换到的目标候选（allowedRoleNames 过滤 + 排除自身角色名 + 是否可切通用助手）
  const computeRowOptions = useCallback((slotRoleName: string) => {
    const allowedRoleNames = switchRoleDialog?.allowedRoleNames;
    const candidateRoles = allowedRoleNames
      ? visibleRoles.filter((r) => allowedRoleNames.includes(r.name))
      : visibleRoles;
    const switchableRoles = candidateRoles.filter((r) => r.name !== slotRoleName);
    const canSwitchToGeneral = slotRoleName !== "通用助手";
    return { switchableRoles, canSwitchToGeneral };
  }, [switchRoleDialog, visibleRoles]);
  // 生成角色介绍（技能 + 风格），单角色 / 多角色映射表按行展开共用同一套内容口径
  const getRoleIntro = useCallback((target: Role | "__general__" | null) => {
    const generalIntro = {
      name: "通用助手",
      skills: "web-search、file-reader、code-runner",
      soul: "无固定行业偏好的通用 AI 伙伴，擅长日常问答、信息检索与轻量创作，按需切换专业度",
    };
    if (!target || target === "__general__") return generalIntro;
    return {
      name: target.name,
      skills: target.skills.map((s) => s.name).join("、"),
      soul: target.soul,
    };
  }, []);
  const [shutdownConfirm, setShutdownConfirm] = useState<{ id: string; name: string; billingMode: "subscription" | "payg" } | null>(null);

  // 正在运行的 Agent 名称集合：用于新增/切换角色时校验角色名称不得与运行中 Agent 重名。
  const runningAgentNames = useMemo(
    () => claws.filter((c) => c.status === "running").map((c) => c.name),
    [claws],
  );

  // 开启面板弹窗（暂未接入业务，UI 仍保留以便后续接入）
  const [panelDialog, setPanelDialog] = useState<{ id: string; name: string } | null>(null);

  // [006] 当前分页页码
  const [page, setPage] = useState(1);

  // 搜索关键词
  const [searchKeyword, setSearchKeyword] = useState("");

  const [refreshingIds, setRefreshingIds] = useState<Set<string>>(new Set());
  
  // 从管控端同步的开关
  const [allowTerminal] = useState(() => {
    return localStorage.getItem("admin_allow_terminal") === "true";
  });

  // ── 平台策略：允许用户「共享 agent」 ──
  // 管控端 PlatformPolicy 写入两个 key：
  //   - admin_allow_share_agent：是否对全员开启（兜底规则）
  //   - admin_share_agent_group_rules：针对部分组的规则数组 JSON（[{ id, groupIds, value }]）
  // 用户端据此决定：创建弹窗是否显示「共享范围」、agent 卡片是否显示/可改共享范围。
  const [shareAgentAllowAll, setShareAgentAllowAll] = useState(() => {
    // 默认「允许」：与管控端保持一致，localStorage 无存值时兜底为 true。
    const stored = localStorage.getItem("admin_allow_share_agent");
    return stored === null ? true : stored === "true";
  });
  const [shareAgentGroupRules, setShareAgentGroupRules] = useState<PolicyRule<boolean>[]>(() =>
    parseShareAgentGroupRules(localStorage.getItem("admin_share_agent_group_rules"))
  );
  useEffect(() => {
    const handleStorage = (e: StorageEvent) => {
      if (e.key === "admin_allow_share_agent") {
        setShareAgentAllowAll(e.newValue === "true");
      } else if (e.key === "admin_share_agent_group_rules") {
        setShareAgentGroupRules(parseShareAgentGroupRules(e.newValue));
      }
    };
    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, []);
  // 某个组是否被允许共享：全员开启则恒为 true；否则命中任一 value=true 的组规则
  const canShareForGroup = useCallback(
    (groupId?: string | null): boolean => {
      if (shareAgentAllowAll) return true;
      if (!groupId) return false;
      return shareAgentGroupRules.some((r) => r.value && r.groupIds.includes(groupId));
    },
    [shareAgentAllowAll, shareAgentGroupRules]
  );
  // 创建弹窗：当前操作上下文是否允许共享。
  // 多组织模式按当前选中组判定；普通模式只要所属任一组（或全员）被允许即可。
  const canShareAgent = useMemo(() => {
    if (shareAgentAllowAll) return true;
    if (groupMode === "multi-group") return canShareForGroup(selectedGroup?.id);
    return MOCK_USER_GROUPS.some((g) => canShareForGroup(g.id));
  }, [shareAgentAllowAll, groupMode, selectedGroup, canShareForGroup]);
  // 某张 agent 卡片是否允许共享（决定卡片上是否显示/可改共享范围）。
  // 多组织模式按该 agent 所属组判定；普通模式回退到「所属任一组允许」。
  const canShareForClaw = useCallback(
    (claw: OpenClawItem): boolean => {
      if (shareAgentAllowAll) return true;
      if (groupMode === "multi-group") return canShareForGroup(claw.groupId);
      return MOCK_USER_GROUPS.some((g) => canShareForGroup(g.id));
    },
    [shareAgentAllowAll, groupMode, canShareForGroup]
  );

  // 自动轮询
  const pollingTimerRef = useRef<NodeJS.Timeout | null>(null);

  // 自动轮询逻辑
  useEffect(() => {
    const startPolling = () => {
      pollingTimerRef.current = setInterval(() => {
        setClaws(prevClaws =>
          prevClaws.map(claw => {
            // creating -> running 或 createFail
            if (claw.status === "creating") {
              const rand = Math.random();
              return rand > 0.1 ? { ...claw, status: "running" } : { ...claw, status: "createFail" };
            }
            // loading -> running 或 loadFail
            if (claw.status === "loading") {
              // 迁移/移交流程中的「加载中」由各自状态机自行驱动到运行中，跳过通用轮询，避免误判为加载失败
              if (claw.migrationPhase || claw.transferPhase) return claw;
              const rand = Math.random();
              return rand > 0.1 ? { ...claw, status: "running" } : { ...claw, status: "loadFail" };
            }
            // maintaining -> running
            if (claw.status === "maintaining") {
      return { ...claw, status: "running" };
     }
        // rollingBack -> running(成功) 或 shutdown(失败)，50% 随机，联调时删除
         if (claw.status === "rollingBack") {
 const isSuccess = Math.random() > 0.5;
  const name = claw.name || "Agent";
   if (isSuccess) {
   toast.success(`${name} 数据回滚成功，Agent 已恢复可用`);
   pushAdminNotification({
   message: `${name} 数据回滚成功，Agent 已恢复可用`,
      category: "success",
      dedupeKey: `backup-rollback:success:${claw.id}`,
       });
         return { ...claw, status: "running" };
    } else {
         toast.error(`${name} 数据回滚失败，请确认 Agent 状态`);
            pushAdminNotification({
     message: `${name} 数据回滚失败，请确认 Agent 状态`,
         category: "failure",
         actionHref: `/openclaw/${claw.id}`,
              actionLabel: "查看详情",
         dedupeKey: `backup-rollback:failure:${claw.id}`,
     });
return { ...claw, status: "shutdown" };
       }
            }
            return claw;
          })
        );
      }, 3000); // 每 3 秒轮询一次
    };

    startPolling();
    return () => {
      if (pollingTimerRef.current) clearInterval(pollingTimerRef.current);
    };
  }, []);

  const handleRefreshStatus = (e: React.MouseEvent, id: string, name: string) => {
    e.stopPropagation();
    if (refreshingIds.has(id)) return;
    setRefreshingIds(prev => { const next = new Set(prev); next.add(id); return next; });
    setTimeout(() => {
      setRefreshingIds(prev => { const next = new Set(prev); next.delete(id); return next; });
      toast.success(`「${name}」状态已刷新`);
    }, 1500);
  };

  const handleCreate = () => {
    if (!newName.trim()) {
      toast.error("请输入 OpenClaw 名称");
      return;
    }
    // 平台策略防御：管控端未对当前用户/组开启「共享 agent」时，强制按「仅自己」提交
    // （正常情况下共享范围区块已隐藏，此处兜底应对策略中途变更等边界情况）
    const effectiveShareScope: "private" | "shared" = canShareAgent ? shareScope : "private";
    // 共享范围校验：选了共享但没选任何分组或成员 —— 顶部 toast 提示，而非置灰按钮
    if (effectiveShareScope === "shared" && shareGroupIds.length === 0 && shareUserIds.length === 0) {
      toast.error("请选择 Agent 的共享范围");
      return;
    }
    const ts = Date.now();
    const DEFAULT_VERSION: Record<string, string> = {
      openclaw: "2026.4.23",
      hermes: "0.12.0",
      lightclawace: "0.1.8",
    };
    const newClaw: OpenClawItem = {
      id: `oc-${ts}`,
      instanceId: `ins-${ts.toString(36).slice(-8)}`,
      name: newName.trim(),
      status: "creating",
      agentType: agentType,
      version: DEFAULT_VERSION[agentType] ?? "2026.4.23",
      createdAt: new Date().toLocaleString("zh-CN"),
      model: "",
      modelVersion: "",
      channels: [],
      skills: [],
      roleName: selectedRole?.name ?? "通用助手",
      memoryStatus: 'none',
      groupId: groupMode === "multi-group" ? selectedGroup.id : "default",
      groupName: groupMode === "multi-group" ? selectedGroup.name : "默认",
      projectIds: selectedProjectIds,
      projectNames: projectPool.filter((p) => selectedProjectIds.includes(p.id)).map((p) => p.name),
      // 用户端自建场景：创建人 == 归属人 == 当前登录用户
      creator: "alice@acompany.com",
      owner: "alice@acompany.com",
      shareScope: effectiveShareScope,
      shareGroupIds: effectiveShareScope === "shared" ? shareGroupIds : [],
      shareUserIds: effectiveShareScope === "shared" ? shareUserIds : [],
      // 展示名按「整组 vs 个人」统一派生
      ...(effectiveShareScope === "shared"
        ? (() => {
            const { groupNames, userNames } = deriveShareDisplay(shareGroupIds, shareUserIds);
            return { shareGroupNames: groupNames, shareUserNames: userNames };
          })()
        : { shareGroupNames: [], shareUserNames: [] }),
    };
    setClaws([newClaw, ...claws]);
    setNewName("");
    setSelectedRole(null);
    setSelectedProjectIds([]);
    setProjectOpen(false);
    setShowCreate(false);
    // 重置共享范围
    setShareScope("private");
    setShareScopeOpen(false);
    setShareGroupIds([]);
    setShareUserIds([]);
    setPage(1); // [006] 创建后跳回第 1 页，展示刚创建的实例
    toast.success(`「${newClaw.name}」创建中...`);
  };

  const handleDelete = (id: string, name: string) => {
    const target = claws.find((c) => c.id === id);
    if (target?.agentType === "localagent") {
      recordLocalAgentRemoveOperation(target);
    }
    setClaws(claws.filter((c) => c.id !== id));
    setDeleteConfirm(null);
    setDeleteConfirmInput("");
    toast.success(target?.agentType === "localagent" ? `「${name}」移除操作已提交` : `「${name}」已删除`);
  };

  const handleSwitchRole = (id: string, name: string, targetRole: string, targetSlotId?: string, previousRoleName?: string) => {
    setClaws(claws.map((c) => {
      if (c.id !== id) return c;
      // 多角色实例且带精确 slot：只替换该 slot 的角色名，其余 slot 不受影响；
      // 若替换的恰好是主角色 slot，同步更新顶层 roleName 保持头像等兼容字段一致
      if (targetSlotId && c.roles && c.roles.length > 0) {
        const targetSlot = c.roles.find((slot) => slot.slotId === targetSlotId);
        return {
          ...c,
          roles: c.roles.map((slot) =>
            slot.slotId === targetSlotId ? { ...slot, roleName: targetRole } : slot
          ),
          roleName: targetSlot?.isMain ? targetRole : c.roleName,
        };
      }
      // 无 slot 数据：保持旧的整实例覆盖语义
      return { ...c, roleName: targetRole };
    }));
    setSwitchRoleDialog(null);
    setSwitchRoleTarget(null);
    clearRoleSwitching(id);
    // 角色切换类操作结束：统一「角色切换成功」提示，并说明从什么切换成什么
    toast.success(
      previousRoleName
        ? `「${name}」角色切换成功：${previousRoleName} → ${targetRole}`
        : `「${name}」角色切换成功：已切换为 ${targetRole}`
    );
  };

  // 多角色实例统一提交（支持新增角色位 / 删除角色位 / 切换角色三合一）：
  //   - nextSlots：本地编辑后的最终角色位列表（已应用增删）
  //   - targets：slotId → 目标角色（切换意图；null 表示保持不变）
  // 与原始 roles 对比后一次性回写 claw.roles，并汇总提示新增 / 删除 / 切换的数量。
  const handleApplyRoleChanges = (
    id: string,
    name: string,
    nextSlots: AgentRoleSlot[],
    targets: Record<string, Role | "__general__" | null>,
    originalSlots: AgentRoleSlot[]
  ) => {
    const originalIds = new Set(originalSlots.map((s) => s.slotId));
    const nextIds = new Set(nextSlots.map((s) => s.slotId));
    const addedCount = nextSlots.filter((s) => !originalIds.has(s.slotId)).length;
    const removedCount = originalSlots.filter((s) => !nextIds.has(s.slotId)).length;

    // 应用「切换为」意图到最终角色位（仅对仍存在的角色位生效）
    let switchedCount = 0;
    // 切换明细（原角色 → 目标角色），用于成功提示里说明「哪些角色切换成了什么」
    const switchedPairs: { fromName: string; toName: string }[] = [];
    const resolvedSlots = nextSlots.map((slot) => {
      const target = targets[slot.slotId];
      if (!target) return slot;
      const targetName = target === "__general__" ? "通用助手" : target.name;
      if (targetName === slot.roleName) return slot;
      // 仅统计"原本已存在且改了角色名"的行为切换；新增行的选定不计入切换
      if (originalIds.has(slot.slotId)) {
        switchedCount += 1;
        // 该行已在抽屉内逐行「确认」时单独提示过，汇总提示里不再重复
        if (!announcedSwitchSlotsRef.current.has(slot.slotId)) {
          switchedPairs.push({ fromName: slot.roleName || "通用助手", toName: targetName });
        }
      }
      return { ...slot, roleName: targetName };
    });

    setClaws(claws.map((c) => {
      if (c.id !== id) return c;
      const mainSlot = resolvedSlots.find((slot) => slot.isMain);
      return {
        ...c,
        roles: resolvedSlots,
        roleCount: resolvedSlots.length,
        roleName: mainSlot?.roleName ?? c.roleName,
      };
    }));
    setSwitchRoleDialog(null);
    setSwitchRoleTargets({});
    setExpandedSlotId(null);
    setEditSlots([]);
    // 仅当本次确实提交了改动（新增 / 删除 / 切换）才视为落库完成，清除卡片上的「角色切换中」提示。
    // 若用户只是打开抽屉查看后原样关闭（无任何改动），说明后台切换仍在进行，须保留「角色切换中（N）」状态。
    const hasAnyChange = addedCount > 0 || removedCount > 0 || switchedCount > 0;
    if (hasAnyChange) {
      clearRoleSwitching(id);
    }

    // 汇总提示：优先合并展示，单一类型变更时给更精确的文案
    const parts: string[] = [];
    if (addedCount > 0) parts.push(`新增 ${addedCount} 个角色`);
    if (removedCount > 0) parts.push(`删除 ${removedCount} 个角色`);
    if (switchedCount > 0) parts.push(`切换 ${switchedCount} 个角色`);
    // 纯切换场景（无新增 / 删除）：统一给「角色切换成功」提示，并说明从什么切换成什么
    if (switchedCount > 0 && addedCount === 0 && removedCount === 0) {
      // 全部行都已在抽屉内单独提示过 → 不再重复提示
      if (switchedPairs.length === 0) {
        announcedSwitchSlotsRef.current = new Set();
        return;
      }
      const detail = switchedPairs.map((p) => `${p.fromName} → ${p.toName}`).join("、");
      toast.success(`「${name}」角色切换成功：${detail}`);
      announcedSwitchSlotsRef.current = new Set();
      return;
    }
    announcedSwitchSlotsRef.current = new Set();
    // 无任何改动（用户打开抽屉后原样关闭）：后台仍在切换、尚未落库，此时应保留卡片的
    // 「角色切换中（N）」状态，禁止再弹「配置已更新」的成功提示，否则会与切换中状态自相矛盾。
    if (!hasAnyChange) {
      return;
    }
    toast.success(`「${name}」已${parts.join("、")}`);
  };

  // ===== 共享范围：打开更改弹窗 =====
  const openShareScopeDialog = (claw: { id: string; name: string; shareScope?: "private" | "shared"; shareGroupIds?: string[]; shareUserIds?: string[] }) => {
    setShareScopeDialog({ id: claw.id, name: claw.name });
    setEditShareScope(claw.shareScope ?? "private");
    setEditShareGroupIds(claw.shareGroupIds ?? []);
    setEditShareUserIds(claw.shareUserIds ?? []);
    setEditShareScopeError("");
    setEditShareTreeOpen(false);
  };

  // ===== 共享范围：确认更改 =====
  const handleShareScopeChange = () => {
    if (!shareScopeDialog) return;
    // 共享范围校验 —— 顶部 toast 提示，而非置灰按钮
    if (editShareScope === "shared" && editShareGroupIds.length === 0 && editShareUserIds.length === 0) {
      toast.error("请选择 Agent 的共享范围");
      return;
    }
    setClaws(claws.map((c) => {
      if (c.id !== shareScopeDialog.id) return c;
      const derived =
        editShareScope === "shared"
          ? deriveShareDisplay(editShareGroupIds, editShareUserIds)
          : { groupNames: [], userNames: [] };
      return {
        ...c,
        shareScope: editShareScope,
        shareGroupIds: editShareScope === "shared" ? editShareGroupIds : [],
        shareUserIds: editShareScope === "shared" ? editShareUserIds : [],
        shareGroupNames: derived.groupNames,
        shareUserNames: derived.userNames,
      };
    }));
    setShareScopeDialog(null);
    toast.success(`「${shareScopeDialog.name}」共享范围已更新`);
  };

  const handleRestart = (id: string, name: string, restartServer: boolean = false) => {
    setClaws(claws.map(c => c.id === id ? { ...c, status: "loading" as OpenClawStatus } : c));
    setRestartConfirm(null);
    setRestartFullServer(false);
    if (restartServer) {
      toast.success(`「${name}」正在重启整台服务器，预计需要约 2 分钟...`);
    } else {
      toast.success(`「${name}」正在重启 Agent 服务...`);
    }
  };

  const handleReinstall = (id: string, name: string) => {
    setClaws(claws.map(c => c.id === id ? { ...c, status: "loading" as OpenClawStatus, op: "reinstall" } : c));
    setReinstallConfirm(null);
    setReinstallConfirmInput("");
    toast.success(`「${name}」正在重新安装...`);
  };

  const openRenameDialog = (claw: { id: string; name: string }) => {
    const target = claws.find((item) => item.id === claw.id);
    if (!target || !canRenameStatus(target.status)) return;
    setRenameConfirm(claw);
    setRenameInput(claw.name);
  };


  const handleRenameInputChange = (value: string) => {
    const noLineBreakValue = value.replace(/[\r\n]/g, "");
    setRenameInput(noLineBreakValue);
  };

  const renameTrimmedValue = renameInput.trim();
  const renameInputBytes = getAgentNameByteLength(renameInput);
  const isRenameOverByteLimit = renameInputBytes > AGENT_NAME_MAX_BYTES;
  const isRenameConfirmDisabled = renameTrimmedValue.length === 0 || isRenameOverByteLimit;

  const handleRenameConfirm = () => {
    if (!renameConfirm || isRenameConfirmDisabled) return;

    try {
      const targetExists = claws.some((claw) => claw.id === renameConfirm.id);
      if (!targetExists) {
        throw new Error("target-not-found");
      }

      setClaws(claws.map((claw) => {
        if (claw.id !== renameConfirm.id) return claw;
        return {
          ...claw,
          name: renameTrimmedValue,
        };
      }));

      setRenameConfirm(null);
      setRenameInput("");
    } catch {
      toast.error("重命名失败，请重试");
    }
  };

  const handleRetry = (id: string, name: string) => {

    setClaws(claws.map(c => c.id === id ? { ...c, status: "loading" as OpenClawStatus } : c));
    toast.success(`「${name}」正在重试...`);
  };

  const openLocalClientAccessDialog = (target?: LocalClientAccessTarget) => {
    setLocalClientAccessDialog({
      step: "system",
      targets: target ? [target] : [],
    });
  };

  const handleCopyLocalClientInstallPrompt = async () => {
    const prompt = buildLocalClientInstallPrompt();
    try {
      const copied = await copyTextToClipboard(prompt);
      if (!copied) throw new Error("copy-failed");
      toast.success("已复制接入 Prompt");
    } catch {
      toast.error("复制失败，请手动复制 Prompt");
    }
  };

  return (
    <TooltipProvider delayDuration={200}>
      <TenantLayout>
        {/* SKILL §7.4 响应式：min-w-[1200px] 保最低可用宽度 / max-w-[1920px] 大屏限宽
            页面段落对齐（Figma 1077:33419）：所有主区块统一 padding-left/right = 120px，
            禁止段内再叠加 px-[42px]/w-20 双层缩进 */}
        <div className="min-w-[1200px]">
          <div className="max-w-[1920px] mx-auto page-enter">
            <div
              className="relative min-h-[calc(100vh-64px)] pl-[120px] pr-[120px] pt-5 pb-[75px]"
            >
          {/* Hero Banner - Figma 358:2325 / 363:5079
              QuickStart 关闭后传入 onShowQuickStart 回调，副文右侧会出现「查看步骤指引」按钮 */}
          <HeroBanner
            onShowQuickStart={
              !showQuickStart ? () => setShowQuickStart(true) : undefined
            }
          />

          {/* Quick Start Guide - Figma 358:2341 */}
          {showQuickStart && (
            <QuickStartGuide onClose={() => setShowQuickStart(false)} />
          )}

          {/* Section Header - 标题 + 视图切换（左） + 组织模式 + 创建按钮（右），合并为一行
              QuickStart 展开时，由 QuickStartGuide 自带的 mb-5 提供与 hero 之间的段间距；
              QuickStart 关闭时，QuickStartGuide 不渲染，需在此补 mt-5 让 hero 与 section 之间保持一致段间距 */}
          <div className={`flex items-center justify-between mb-4 ${!showQuickStart ? "mt-5" : ""}`}>
            {/* 左侧：标题 + 搜索框，左对齐（「我的 Agent」在左，搜索框紧随其后） */}
            <div className="flex items-center gap-3">
              <SectionTitle>
    我的 Agent
      <span className="text-[var(--text-muted)] font-normal">（{claws.length}）</span>
              </SectionTitle>
              {/* 搜索框 */}
              <div className="relative">
                <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-muted)] pointer-events-none" />
                <Input
                  tenant
                  type="text"
                  placeholder="搜索名称、ID或类型"
                  value={searchKeyword}
                  onChange={(e) => { setSearchKeyword(e.target.value); setPage(1); }}
                  className="h-8 w-52 pl-8 pr-8 text-sm"
                />
                {searchKeyword && (
                  <button
                    onClick={() => { setSearchKeyword(""); setPage(1); }}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--text-weak)] hover:text-[var(--text-muted)]"
                  >
                    <X className="w-3.5 h-3.5" />
                  </button>
                )}
              </div>
            </div>
            {/* 视图切换与按钮之间间距 20px */}
            <div className="flex items-center gap-5">
              {/* 视图切换：管理视图 / 对话视图（移到右侧原"普通/多组织"位置）
                  data-guide：供步骤指引气泡按真实位置贴合标注 */}
              <span data-guide="tenant-view-switch" className="inline-flex">
                <ViewModeSegmented value={viewMode} onChange={handleViewModeChange} />
              </span>

              <Button
                variant="tenant-outline"
                size="claw-lg"
                className="!h-[34px]"
                onClick={() => openLocalClientAccessDialog()}
              >
                <Plug className="w-4 h-4" />
                接入外部 Agent
              </Button>
              {/* 创建 Agent 按钮：分支样式(tenant-primary/34px) + 主干停服禁用逻辑 */}
              <Tooltip>
                <TooltipTrigger asChild>
                  <span tabIndex={isCreateAgentDisabled ? 0 : undefined} data-guide="tenant-create-agent">
                    <Button
                      onClick={() => {
                        if (isCreateAgentDisabled) return;
                        if (groupMode === "multi-group") {
                          setSelectedGroup(getDefaultGroup(MOCK_USER_GROUPS));
                        }
                        setShareScope("private");
                        setShareScopeOpen(false);
                        setShareGroupIds([]);
                        setShareUserIds([]);
                        setShareScopeError("");
                        setSelectedProjectIds([]);
                        setProjectOpen(false);
                        setShowCreate(true);
                      }}
                      disabled={isCreateAgentDisabled}
                      variant="tenant-primary"
                      size="claw-lg"
                      className="!h-[34px]"
                    >
                      <Plus className="w-4 h-4" />
                      创建 Agent
                    </Button>
                  </span>
                </TooltipTrigger>
                {isCreateAgentDisabled && (
                  <TooltipContent side="bottom" className="text-xs max-w-[240px]">
                    {createAgentDisabledTip}
                  </TooltipContent>
                )}
              </Tooltip>
            </div>
          </div>

          {/* Content Area - 段落左右内边距由父级 120px 统一控制，本层不再额外缩进 */}
          <div className="relative pb-8">
            {/* Main Content */}
              {viewMode === "chat" ? (
                /* 新版对话卡片视图：Figma 1003:22598 还原稿（AgentChat）。
                 * 注：视觉先替换到位，原 ChatView 的业务逻辑（claws / 状态机 / resize / 权限）
                 * 后续按需接入 AgentChat。 */
                <AgentChat embedded />
              ) : (() => {
                const allClaws = claws;
                // 搜索过滤：按名称 / instanceId / 类型匹配
                const keyword = searchKeyword.trim().toLowerCase();
                const filteredClaws = keyword
                  ? allClaws.filter((c) => {
                      const typeName =
                        c.agentType === "hermes"
                          ? "hermes agent"
                            : c.agentType === "lightclawace"
                              ? "lightclaw ace"
                              : c.agentType === "localagent"
                              ? `外部 agent external agent ${getExternalAgentAccessTarget(c) || ""} ${c.model || ""}`
                              : "openclaw";
                      return (
                        c.name.toLowerCase().includes(keyword) ||
                        c.instanceId.toLowerCase().includes(keyword) ||
                        typeName.includes(keyword)
                      );
                    })
                  : allClaws;
                // [006] 分页切片：先按创建时间倒序，再按当前页切出 30 条
                const sortedClaws = [...filteredClaws].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
                const totalPages = Math.max(1, Math.ceil(sortedClaws.length / PAGE_SIZE));
                const safePage = Math.min(page, totalPages);
                const paginatedClaws = sortedClaws.slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE);
                return (
                  <div>
                    {/* 单页展示所有实例（不按 agent 类型分 Tab） */}
                    {allClaws.length === 0 ? (
                      <Empty className="border-0 py-20">
                        <EmptyHeader>
                          <EmptyMedia />
                          <EmptyDescription>暂无实例</EmptyDescription>
                        </EmptyHeader>
                        <EmptyContent>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span tabIndex={isCreateAgentDisabled ? 0 : undefined}>
                                <Button
                                  onClick={() => {
                                    if (isCreateAgentDisabled) return;
                                    setShareScope("private");
                                    setShareScopeOpen(false);
                                    setShareGroupIds([]);
                                    setShareUserIds([]);
                                    setShareScopeError("");
                                    setShowCreate(true);
                                  }}
                                  disabled={isCreateAgentDisabled}
                                  variant="tenant-outline"
                                >
                                  <Plus className="w-4 h-4 mr-1.5" />
                                  创建 Agent
                                </Button>
                              </span>
                            </TooltipTrigger>
                            {isCreateAgentDisabled && (
                              <TooltipContent side="bottom" className="text-xs max-w-[240px]">
                                {createAgentDisabledTip}
                              </TooltipContent>
                            )}
                          </Tooltip>
                        </EmptyContent>
                      </Empty>
                    ) : filteredClaws.length === 0 ? (
                      <Empty className="border-0 bg-transparent py-24">
                        <EmptyHeader>
                          <EmptyMedia />
                          <EmptyDescription>未找到匹配的 Agent，尝试搜索其他关键词</EmptyDescription>
                        </EmptyHeader>
                      </Empty>
                    ) : (
                      <>
                      {/* 卡片响应式网格：2 / 3 列分段（不降级到 1 列）。
                          - 视口 < 1200px：由页面 min-w-[1200px] 兜底，触发横向滚动，
                            网格保持 2 列结构，绝不挤压成单列。
                          - 视口 1200~1420px（含 13 寸 MBP 1280）：2 列（每列 ≈ 470px，舒展）
                          - 视口 ≥ 1420px：3 列（每列 ≥ 380px，时间 + 双按钮不挤压）
                          AgentCard min-w-[360px] 保证单卡内按钮始终不溢出。 */}
                      <div data-guide="tenant-agent-grid" className="grid items-stretch gap-5 grid-cols-2 min-[1420px]:grid-cols-3">
                        {paginatedClaws.map((claw) => (
                          <AgentCard
                            key={claw.id}
                            claw={claw}
                            onClickCard={(c) => navigate(`/openclaw/${c.id}`)}
                            onRefresh={(e, id, name) => handleRefreshStatus(e, id, name)}
                            onRestart={(c) => setRestartConfirm({ id: c.id, name: c.name })}
                            onReinstall={(c) => setReinstallConfirm({ id: c.id, name: c.name })}
                            onDelete={(c) => {
                              setDeleteConfirm({
                                id: c.id,
                                name: c.name,
                                status: c.status,
                                memoryStatus: c.memoryStatus,
                                agentType: c.agentType,
                                localProduct: c.localProduct,
                              });
                              setDeleteConfirmInput("");
                            }}
    onSwitchRole={(c) => {
      const slots: AgentRoleSlot[] =
        c.roles && c.roles.length > 0
          ? c.roles.map((s) => ({ ...s }))
          : [{ slotId: `slot-main-${c.id}`, roleName: c.roleName ?? "通用助手", isMain: true }];
      // 卡片「切换角色」统一走「角色管理抽屉」（映射表格式）：单角色实例也在此，无 roles 时合成单元素角色位。
      // （历史上曾按 editRoleScheme 分流到独立 BatchSwitchRoleDialog，现该分支已废弃，抽屉工具栏仍保留批量切换入口。）
      setSwitchRoleDialog({ id: c.id, name: c.name, roleName: c.roleName ?? "通用助手", roleCount: c.roleCount, allowedRoleNames: c.allowedRoleNames, roles: slots });
    }}
                            onRetry={(id, name) => handleRetry(id, name)}
                            onChat={() => setViewMode("chat")}
                            canOpenTerminal={(c) => {
                              if (c.agentType === "localagent") return false;
                              const clawGroup =
                                MOCK_USER_GROUPS.find((g) => g.id === (c.groupId || "grp-fe")) ||
                                null;
                              return groupMode === "multi-group" && clawGroup
                                ? clawGroup.permissions.allowTerminal
                                : allowTerminal;
                            }}
                            refreshing={refreshingIds.has(claw.id)}
                            groupMode={groupMode}
                            onRename={(c) => openRenameDialog({ id: c.id, name: c.name })}
                            onShutdown={(c) => {
                              const billing = c.billingMode ?? getAgentBillingMode(c.id) ?? getDefaultLaunchBillingMode();
                              setShutdownConfirm({ id: c.id, name: c.name, billingMode: billing });
                            }}
                            onPowerOn={(c) => {
                              setClaws((prev) => prev.map((x) => (x.id === c.id ? { ...x, status: "running" as OpenClawStatus } : x)));
                              toast.success(`已开机 ${c.name}`);
                            }}
                            onMigrate={(c) => setShowMigrateDialog({ id: c.id, instanceName: c.name })}
                            onTransfer={(c) => setShowTransferDialog({ id: c.id, instanceName: c.name, groupName: c.groupChangeOriginalGroup || "—" })}
                            onLocalReconnect={(c) => openLocalClientAccessDialog(getExternalAgentAccessTarget(c))}
                            onCancelTransfer={(c) => {
                              setClaws((prev) => prev.map((x) => (x.id === c.id ? { ...x, transferPhase: undefined } : x)));
                              toast.info("已取消移交");
                            }}
                            onAcceptTransfer={(c) => {
                              // 确认接收：卡片保持原位置，清除移交蒙版，已关机 → 加载中 → 运行中
                              setClaws((prev) => prev.map((x) =>
                                x.id === c.id ? { ...x, incomingTransfer: undefined, status: "loading" as OpenClawStatus } : x
                              ));
                              toast.success("已确认接收，实例即将开机");
                              setTimeout(() => {
                                setClaws((prev) => prev.map((x) =>
                                  x.id === c.id ? { ...x, status: "running" as OpenClawStatus } : x
                                ));
                              }, 1500);
                            }}
                            onRejectTransfer={(c) => {
                              setClaws((prev) => prev.filter((x) => x.id !== c.id));
                              toast.info("已拒绝接收");
                            }}
           onChangeShareScope={canShareForClaw(claw) ? openShareScopeDialog : undefined}
       sharedReadonly={isSharedToMe(claw)}
     roleSwitchingCount={switchingRoleAgents[claw.id] ?? 0}
                            onClickSwitchingBadge={(c) => {
                              setSwitchingProgressDialogAgent(c.id);
                            }}
                            roleAddingCount={addingRoleAgents[claw.id] ?? 0}
                            onAddRole={(c) => {
                              // 方案2「新增」：直接打开独立新增角色弹窗（不打开角色管理抽屉）；
                              // 默认选中第一个角色 tag 由 standaloneAddRole 的 useEffect 统一处理
                              const existingSlots: AgentRoleSlot[] =
                                c.roles && c.roles.length > 0
                                  ? c.roles.map((s) => ({ ...s }))
                                  : [{ slotId: `slot-main-${c.id}`, roleName: c.roleName ?? "通用助手", isMain: true }];
                              setStandaloneAddRole({ id: c.id, name: c.name, roles: existingSlots, allowedRoleNames: c.allowedRoleNames });
                            }}
                          />
                        ))}
                    </div>
                    {/* [006] 分页控件 */}
                    {totalPages > 1 && (
                    <div className="relative mt-6 px-6 py-3">
                      <DialogPagination
                        total={sortedClaws.length}
                        currentPage={safePage}
                        totalPages={totalPages}
                        onPrevPage={() => setPage((p) => Math.max(1, p - 1))}
                        onNextPage={() => setPage((p) => Math.min(totalPages, p + 1))}
                        className="w-full justify-between"
                      />
                    </div>
                    )}
                    </>
                  )}
                  </div>
                );
              })()}
          </div>

            {/* /内容区（120px padding）闭合 */}
          </div>
          </div>
        </div>

        {/* External Agent Access Dialog */}
        <Dialog
          open={!!localClientAccessDialog}
          onOpenChange={(open) => {
            if (!open) setLocalClientAccessDialog(null);
          }}
        >
          <DialogContent size="lg">
            {localClientAccessDialog && (
                <>
                  <DialogHeader>
                    <DialogTitle>接入外部 Agent</DialogTitle>
                    <DialogDescription className="text-sm text-[var(--text-muted)]">
                      支持将 CodeBuddy、WorkBuddy、Claude Code、Codex、iMate、KnotBot 等外部 Agent 接入企业管理，统一同步企业 Skill 和规范。
                    </DialogDescription>
                  </DialogHeader>

                  <div
                    className="mb-2 flex items-start gap-2 rounded-lg border border-[#93C5FD] bg-[#F8FBFF] px-3 py-2.5"
                  >
                    <Info className="mt-0.5 h-3.5 w-3.5 shrink-0 text-[#1447E6]" />
                    <p className="text-xs leading-5 text-[var(--text-secondary)]">
                      复制接入 Prompt，在外部 Agent 的对话或授权入口中执行，即可完成 ClawPro 授权绑定和资源同步。
                    </p>
                  </div>

                  <div className="space-y-3">
                    <div>
                      <div className="mb-2 text-sm font-medium text-[var(--text-body)]">支持的接入环境</div>
                      <div className="flex flex-wrap gap-2">
                        {LOCAL_CLIENT_SYSTEM_OPTIONS.map((option) => (
                          <span
                            key={option.value}
                            className="inline-flex items-center gap-1.5 rounded-md border border-[#E2E8F0] bg-white px-3 py-1.5 text-xs text-[var(--text-body)]"
                          >
                            <option.Icon className="h-3.5 w-3.5 text-[var(--text-weak)]" />
                            {option.label}
                          </span>
                        ))}
                      </div>
                    </div>

                    <div>
                      <div className="mb-2 text-sm font-medium text-[var(--text-body)]">支持的外部 Agent</div>
                      <div className="flex flex-wrap gap-2">
                        {LOCAL_CLIENT_ACCESS_OPTIONS.map((option) => (
                          <span
                            key={option.name}
                            className="inline-flex items-center gap-1.5 rounded-md border border-[#E2E8F0] bg-white px-3 py-1.5 text-xs text-[var(--text-body)]"
                          >
                            <Laptop className="h-3.5 w-3.5 text-[var(--text-weak)]" />
                            {option.name}
                          </span>
                        ))}
                      </div>
                    </div>
                  </div>

                  <DialogFooter>
                    <Button variant="tenant-outline" onClick={() => setLocalClientAccessDialog(null)}>
                      关闭
                    </Button>
                    <Button
                      variant="tenant-primary"
                      onClick={handleCopyLocalClientInstallPrompt}
                    >
                      <Copy className="h-4 w-4" />
                      复制接入 Prompt
                    </Button>
                  </DialogFooter>
                </>
            )}
          </DialogContent>
        </Dialog>

        {/* Rename Dialog */}
        <Dialog
          open={!!renameConfirm}
          onOpenChange={(open) => {
            if (!open) {
              setRenameConfirm(null);
              setRenameInput("");
            }
          }}
        >
          <DialogContent
            size="sm"
            onInteractOutside={(event) => event.preventDefault()}
          >
            <DialogHeader>
              <DialogTitle>重命名 Agent</DialogTitle>
              <DialogDescription className="text-sm text-[var(--text-muted)]">
                支持中英文、数字、空格及常用符号。
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-2">
              <Label htmlFor="rename-agent-input">名称</Label>
              <Input
                id="rename-agent-input"
                tenant
                value={renameInput}
                placeholder="请输入 Agent 名称"
                aria-invalid={isRenameOverByteLimit}
                className={isRenameOverByteLimit ? "border-[var(--text-danger)] focus-visible:ring-[var(--text-danger)]" : undefined}
                onChange={(e) => handleRenameInputChange(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    handleRenameConfirm();
                  }
                }}
              />
              <p
                className={`text-xs min-h-5 ${isRenameOverByteLimit ? "text-[var(--text-danger)]" : "text-transparent"}`}
                aria-live="polite"
              >
                {isRenameOverByteLimit ? "名称不能超过 128 字节" : ""}
              </p>
            </div>
            <DialogFooter>
              <Button
                variant="tenant-outline"
                onClick={() => {
                  setRenameConfirm(null);
                  setRenameInput("");
                }}
              >
                取消
              </Button>
              <Button
                variant="tenant-primary"
                disabled={isRenameConfirmDisabled}
                onClick={handleRenameConfirm}
              >
                确认
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Shutdown Confirm Dialog */}
        <Dialog open={!!shutdownConfirm} onOpenChange={(open) => { if (!open) setShutdownConfirm(null); }}>
          <DialogContent size="sm">
            <DialogHeader>
              <DialogTitle>确认关机</DialogTitle>
            </DialogHeader>
            <BodyText as="p" tone="secondary">
              关机后该 Agent「{shutdownConfirm?.name}」将无法使用，直到重新开机。
            </BodyText>
            <DialogFooter>
              <Button variant="tenant-outline" onClick={() => setShutdownConfirm(null)}>取消</Button>
              <Button
                variant="tenant-destructive"
                onClick={() => {
                  if (!shutdownConfirm) return;
                  setClaws(prev => prev.map(c => c.id === shutdownConfirm.id ? { ...c, status: "shutdown" as OpenClawStatus } : c));
                  toast.success(`已关机 ${shutdownConfirm.name}`);
                  setShutdownConfirm(null);
                }}
              >
                确认关机
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Restart Confirm Dialog */}
        <Dialog open={!!restartConfirm} onOpenChange={(open) => { if (!open) { setRestartConfirm(null); setRestartFullServer(false); } }}>
          <DialogContent size="sm">
            <DialogHeader>
              <DialogTitle>重启 Agent</DialogTitle>
            </DialogHeader>
            <BodyText as="p" tone="secondary">
              将会重启 Agent「{restartConfirm?.name}」服务，重启期间该 Agent 将短暂不可用。
            </BodyText>
            <div className="flex items-center gap-2 pt-1">
              <Checkbox
                id="restart-full-server"
                checked={restartFullServer}
                onCheckedChange={(checked) => setRestartFullServer(checked === true)}
              />
              <Label htmlFor="restart-full-server" className="text-sm font-normal cursor-pointer">
                重启云服务器
              </Label>
              <Tooltip>
                <TooltipTrigger asChild>
                  <HelpCircle className="w-4 h-4 text-[var(--text-muted)] cursor-help shrink-0" />
                </TooltipTrigger>
                <TooltipContent side="bottom" className="text-xs max-w-[240px]">
                  勾选后，将重启整台云服务器，建议仅在重启 Agent 后服务仍异常时使用
                </TooltipContent>
              </Tooltip>
            </div>
            <DialogFooter>
              <Button variant="tenant-outline" onClick={() => { setRestartConfirm(null); setRestartFullServer(false); }}>取消</Button>
              <Button
                variant="tenant-destructive"
                onClick={() => handleRestart(restartConfirm!.id, restartConfirm!.name, restartFullServer)}
              >
                确认重启
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Reinstall Confirm Dialog */}
        <Dialog open={!!reinstallConfirm} onOpenChange={(open) => { if (!open) setReinstallConfirm(null); }}>
          <DialogContent size="sm">
            <DialogHeader>
              <DialogTitle>确认重新安装</DialogTitle>
            </DialogHeader>
            <BodyText as="p" tone="secondary">
              将使用最新镜像重新安装「{reinstallConfirm?.name}」，清除当前所有配置且无法恢复，安装完成后需重新配置模型和通道。
            </BodyText>
            <div className="space-y-2">
              <Label>请输入「重装」以确认</Label>
              <Input
                tenant
                placeholder="输入「重装」"
                value={reinstallConfirmInput}
                onChange={(e) => setReinstallConfirmInput(e.target.value)}
              />
            </div>
            <DialogFooter>
              <Button variant="tenant-outline" onClick={() => setReinstallConfirm(null)}>取消</Button>
              <Button
                variant="tenant-destructive"
                disabled={reinstallConfirmInput !== "重装"}
                onClick={() => handleReinstall(reinstallConfirm!.id, reinstallConfirm!.name)}
              >
                确认重新安装
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Delete Confirm Dialog */}
        <Dialog open={!!deleteConfirm} onOpenChange={(open) => { if (!open) setDeleteConfirm(null); }}>
          <DialogContent size="sm">
            <DialogHeader>
              <DialogTitle>
                {deleteConfirm?.agentType === "localagent"
                  ? "确认移除"
                  : deleteConfirm?.status === "createFail"
                    ? "删除记录"
                    : "确认删除"}
              </DialogTitle>
            </DialogHeader>
            <BodyText as="p" tone="secondary">
                {deleteConfirm?.agentType === "localagent"
                  ? `移除后「${(deleteConfirm?.name || "").replace(/ 外部 Agent$/, "")}」将不再出现在我的 Agent 列表中，仅解除与 ClawPro 的接入关系，不影响该外部 Agent 在外部平台的运行。`
                  : deleteConfirm?.status === "createFail"
                    ? `此操作将移除「${deleteConfirm?.name}」该创建失败的记录，底层资源将由系统自动回收。`
                    : `此操作不可撤销。「${deleteConfirm?.name}」实例及相关数据将被永久删除，已配置的模型、通道和插件将全部清除且无法恢复。`}
            </BodyText>
            {deleteConfirm?.agentType === "localagent" && (
              <Alert variant="info" className="mt-3">
                <AlertInfoIcon />
                <AlertDescription>
                  已同步或已安装的 Skill 和插件会保留，不会被删除。
                </AlertDescription>
              </Alert>
            )}
            {/* 记忆数据清理提示 */}
            {deleteConfirm?.agentType !== "localagent" && deleteConfirm?.status !== "createFail" && deleteConfirm?.memoryStatus && deleteConfirm.memoryStatus !== 'none' && (
              <Alert variant="warning">
                <AlertCircle />
                <AlertDescription>
                  该 Agent 已开启 Memory {deleteConfirm.memoryStatus === 'pro' ? 'Pro' : 'Free'}，相关记忆数据也将被一并清理。
                </AlertDescription>
              </Alert>
            )}
            {deleteConfirm?.agentType !== "localagent" && deleteConfirm?.status === "running" && (
              <div className="space-y-2">
                <Label>请输入「删除」以确认</Label>
                <Input
                  tenant
                  placeholder="输入「删除」"
                  value={deleteConfirmInput}
                  onChange={(e) => setDeleteConfirmInput(e.target.value)}
                />
              </div>
            )}
            <DialogFooter>
              <Button variant="tenant-outline" onClick={() => setDeleteConfirm(null)}>取消</Button>
              <Button
                variant="tenant-destructive"
                disabled={deleteConfirm?.agentType !== "localagent" && deleteConfirm?.status === "running" && deleteConfirmInput !== "删除"}
                onClick={() => handleDelete(deleteConfirm!.id, deleteConfirm!.name)}
              >
                {deleteConfirm?.agentType === "localagent" ? "确认移除" : "确认删除"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* 「新增角色」独立弹窗：抽取为共享组件 AddRoleDialog，与设置详情页（OpenClawDetailGuide）
            共用同一实现，任何 UI 改动两页自动联动。 */}
        <AddRoleDialog
          source={standaloneAddRole}
          onClose={() => setStandaloneAddRole(null)}
          visibleRoles={visibleRoles}
          getRoleIntro={getRoleIntro}
          runningAgentNames={runningAgentNames}
          onConfirm={({ target, role, nextSlots, newSlotId, skillCount, roleName }) => {
            // 不直接落库：先进入配置进度弹窗，进度走完 100% 后由回调统一落库。
            // 进度弹窗头部副标题显示用户在新增弹窗里配置的「角色名称」（而非角色风格描述）。
            roleConfigProgress.start({
              roleName: role.name,
              roleSoul: roleName,
              skillCount,
              mode: "add",
              agentId: target.id,
              agentName: target.name,
              apply: () => {
                handleApplyRoleChanges(
                  target.id,
                  target.name,
                  nextSlots,
                  { [newSlotId]: role },
                  target.roles
                );
                // 新增落库完成：清除卡片「角色新增中」状态（与切换角色的 clearRoleSwitching 对齐）
                clearRoleAdding(target.id);
                // 同步更新编辑角色弹窗数据源（使弹窗列表立即显示新增的角色行）
                setStandaloneBatchSwitch((prev) => prev && prev.id === target.id ? { ...prev, slots: nextSlots } : prev);
              },
            });
          }}
        />

        {/* 方案2/3「切换角色」独立批量切换弹窗：抽取为共享组件 BatchSwitchRoleDialog，
            与设置详情页（OpenClawDetailGuide）共用同一实现，任何 UI 改动两页自动联动。 */}
        <BatchSwitchRoleDialog
          source={standaloneBatchSwitch}
          onClose={() => setStandaloneBatchSwitch(null)}
          visibleRoles={visibleRoles}
          getRoleIntro={getRoleIntro}
          runningAgentNames={runningAgentNames}
          onBackgrounded={(agentId, count, agentName, items) => {
            markRoleSwitching(agentId, count);
            storeSwitchingProgress(agentId, agentName, items);
          }}
          scheme={editRoleScheme}
          roleAddingCount={standaloneBatchSwitch ? (addingRoleAgents[standaloneBatchSwitch.id] ?? 0) : 0}
          onAddRole={() => {
            if (!standaloneBatchSwitch) return;
            const existingSlots: AgentRoleSlot[] = standaloneBatchSwitch.slots;
            setStandaloneAddRole({ id: standaloneBatchSwitch.id, name: standaloneBatchSwitch.name, roles: existingSlots, allowedRoleNames: standaloneBatchSwitch.allowedRoleNames });
          }}
          onCommit={({ id, name, slots, targets }) => {
            handleApplyRoleChanges(id, name, slots, targets, slots);
            clearRoleSwitching(id);
          }}
          onDeleteSlot={(slotId, _roleName) => {
            if (!standaloneBatchSwitch) return;
            // 从弹窗数据源中移除该角色位
            const updatedSlots = standaloneBatchSwitch.slots.filter((s) => s.slotId !== slotId);
            setStandaloneBatchSwitch({ ...standaloneBatchSwitch, slots: updatedSlots });
            // 同步更新本地 claws 数据
            setClaws((prev) => prev.map((c) => {
              if (c.id !== standaloneBatchSwitch.id) return c;
              const newRoles = (c.roles || []).filter((r) => r.slotId !== slotId);
              return { ...c, roles: newRoles, roleCount: newRoles.length };
            }));
          }}
        />

        {/* ===== 角色管理抽屉（共享组件 RoleManageSheet） ===== */}
        <RoleManageSheet
          dialog={switchRoleDialog}
          onOpenChange={(open) => {
            if (!open) {
              const slots = switchRoleDialog?.roles;
              // 单角色 / 多角色统一走映射表抽屉：只要携带角色位（>=1）关闭即落库本次增删改。
              if (switchRoleDialog && slots && slots.length >= 1) {
                handleApplyRoleChanges(
                  switchRoleDialog.id,
                  switchRoleDialog.name,
                  editSlots,
                  switchRoleTargets,
                  slots
                );
                return;
              }
              setSwitchRoleDialog(null);
              setSwitchRoleTarget(null);
              setSwitchRoleTargets({});
              setExpandedSlotId(null);
              setSwitchSlotSkillPopoverSlot(null);
              setSwitchSlotSkillNames({});
            }
          }}
          visibleRoles={visibleRoles}
          roleConfigProgress={roleConfigProgress}
          switchingRoleAgents={switchingRoleAgents}
          addingRoleAgents={addingRoleAgents}
          clearRoleSwitching={clearRoleSwitching}
          getSkillDescription={getSkillDescription}
          onApplyRoleChanges={handleApplyRoleChanges}
          onSwitchRole={handleSwitchRole}
          onOpenAddRole={({ id, name, roles, allowedRoleNames }) => {
            setStandaloneAddRole({ id, name, roles, allowedRoleNames });
          }}
          onOpenBatchSwitch={({ id, name, roles, allowedRoleNames }) => {
            const initSlots: AgentRoleSlot[] =
              roles && roles.length > 0
                ? roles.map((s) => ({ ...s }))
                : [{ slotId: `slot-main-${id}`, roleName: switchRoleDialog?.roleName ?? "通用助手", isMain: true }];
            setStandaloneBatchSwitch({ id, name, slots: initSlots, allowedRoleNames });
          }}
          onNavigateSettings={(id) => navigate(`/openclaw/${id}`)}
        />

        {/* Panel Dialog - 开启面板 */}
        <Dialog open={!!panelDialog} onOpenChange={(open) => { if (!open) setPanelDialog(null); }}>
          <DialogContent size="lg">
            <DialogHeader>
              <DialogTitle>开启面板</DialogTitle>
            </DialogHeader>
            <Alert variant="warning">
              <AlertCircle />
              <AlertDescription>
                访问链接已生成，该链接含有您的 API Key 和加密配置，请勿分享给第三方，以防隐私泄露或资产损失。
              </AlertDescription>
            </Alert>
            <div className="space-y-3 py-1">
              <div className="flex items-center gap-3">
                <MetaText as="span" className="w-24 shrink-0">WebSocket URL</MetaText>
                <CodeText as="span" className="flex-1 truncate">
                  http://43.139.137.45:38341/knmnz8?token=8512b8ef...
                </CodeText>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label="复制 WebSocket URL"
                  onClick={() => { navigator.clipboard.writeText("http://43.139.137.45:38341/knmnz8?token=8512b8ef93cdfd393ad6af5efa42c1e54981f3cb69f381eb"); toast.success("已复制"); }}
                >
                  <Copy className="w-4 h-4" />
                </Button>
              </div>
              <Separator />
              <div className="flex items-center gap-3">
                <MetaText as="span" className="w-24 shrink-0">网关令牌</MetaText>
                <CodeText as="span" className="flex-1 truncate">
                  8512b8ef93cdfd393ad6af5efa42c1e54981f3cb69f381eb
                </CodeText>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label="复制网关令牌"
                  onClick={() => { navigator.clipboard.writeText("8512b8ef93cdfd393ad6af5efa42c1e54981f3cb69f381eb"); toast.success("已复制"); }}
                >
                  <Copy className="w-4 h-4" />
                </Button>
              </div>
            </div>
            <HelperText as="p">
              用浏览器打开 WebSocket URL，如面板需要填入网关令牌，则将网关令牌复制并粘贴过去，即可进入面板。
            </HelperText>
            <DialogFooter>
              <Button
                variant="tenant-primary"
                className="w-full"
                onClick={() => { window.open("http://43.139.137.45:38341/knmnz8?token=8512b8ef93cdfd393ad6af5efa42c1e54981f3cb69f381eb", "_blank"); }}
              >
                立即访问
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Create Dialog —— 单弹窗：组织 / 名称 / 类型 / 角色 + 共享范围（角色介绍内联展示） */}
        <Dialog open={showCreate} onOpenChange={(open) => { setShowCreate(open); if (!open) { setSelectedRole(null); setAgentType("openclaw"); setAgentTypeOpen(false); setRoleOpen(false); setShareScope("private"); setShareScopeOpen(false); setShareGroupIds([]); setShareUserIds([]); setShareTreeOpen(false); } }}>

          <DialogContent size="md" className="flex flex-col max-h-[85vh]">
            <DialogHeader>
              <DialogTitle>
                创建 Agent
              </DialogTitle>
              <DialogDescription className="sr-only">
                创建 Agent：选择所属组织、填写名称、选择类型与角色身份
              </DialogDescription>
            </DialogHeader>

            {/* ===== 单步表单：所属组织 + 名称 + 类型 + 角色（含介绍卡） ===== */}
            {/* DialogBody：内容超过最大高度时滚动，标题/底部按钮冻结 */}
            <DialogBody className="px-6">
            <div className="py-2 space-y-5">
              {/* 所属组织下拉框（仅多组织模式显示） */}
              {groupMode === "multi-group" && (
                <div className="space-y-2">
                  {/* 与 “Agent 类型 / 角色身份” 同款字号字重，保证四个 label 视觉权重一致 */}
                  <span className="block text-sm font-medium text-[var(--text-emphasis)]">所属组织</span>
                  <HelperText as="p">
                    您属于多个组织，不同组织对应不同的 Agent 配置和权限，请确认要使用的组织
                  </HelperText>
                  <Select
                    value={selectedGroup.id}
                    onValueChange={(value) => {
                      const group = MOCK_USER_GROUPS.find(g => g.id === value);
                      if (group) {
                        setSelectedGroup(group);
                        // 重置 agent 类型为该组织允许的第一个（如果当前类型不被允许）
                        if (!group.permissions.agentTypes.includes(agentType)) {
                          setAgentType(group.permissions.agentTypes[0]);
                        }
                        // 重置角色（如果当前角色不被新组织允许）
                        if (selectedRole && !group.permissions.roles.includes(selectedRole.name)) {
                          setSelectedRole(null);
                        }
                      }
                    }}
                  >
                    <SelectTrigger tenant className="w-full">
                      <SelectValue placeholder="选择所属组织" />
                    </SelectTrigger>
                    <SelectContent>
                      {MOCK_USER_GROUPS.map((group) => (
                        <SelectItem key={group.id} value={group.id}>{group.name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}

              {/* Name Input */}
              <div className="space-y-2">
                {/* 与 “Agent 类型 / 角色身份” 同款字号字重 */}
                <label
                  htmlFor="claw-name"
                  className="block text-sm font-medium text-[var(--text-emphasis)]"
                >
                  Agent 名称
                </label>
                <Input
                  tenant
                  id="claw-name"
                  placeholder="请输入文本内容"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && handleCreate()}
                  autoFocus
                />
              </div>

              {/* Agent Type —— 默认收起；无外框，与上方 Label/Input 同样视觉权重 */}
              <Collapsible open={agentTypeOpen} onOpenChange={setAgentTypeOpen}>
                <CollapsibleTrigger
                  className="group flex w-full items-center gap-3 h-9 px-0 text-left focus-visible:outline-none"
                >
                  <span className="text-sm font-medium text-[var(--text-emphasis)]">Agent 类型</span>
                  <span className="ml-auto flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                    {/* 「已选」前置标记当前值，与右侧选项形成 标签 + 值 的语义 */}
                    <span className="text-xs text-[var(--text-weak)]">已选</span>
                      {agentType === "openclaw"
                        ? "OpenClaw"
                        : agentType === "hermes"
                        ? "Hermes"
                        : agentType === "lightclawace"
                          ? "Lightclaw ACE"
                          : "外部 Agent"}
                    <ChevronDown className="w-4 h-4 text-[var(--text-weak)] transition-transform duration-200 group-data-[state=open]:rotate-180" />
                  </span>
                </CollapsibleTrigger>
                <CollapsibleContent className="overflow-hidden">
                  <div className="px-0 pt-2 pb-1">
                    <RadioGroup
                      value={agentType}
                      onValueChange={(value) => {
                        setAgentType(value as "openclaw" | "hermes" | "lightclawace" | "localagent");
                        if (value !== "openclaw") {
                          setSelectedRole(null);
                        }
                      }}
                      className="flex flex-wrap gap-2"
                    >
                      {([["openclaw", "OpenClaw"], ["hermes", "Hermes"], ["lightclawace", "Lightclaw ACE"], ["localagent", "外部 Agent"]] as const)
                        .filter(([value]) => groupMode !== "multi-group" || selectedGroup.permissions.agentTypes.includes(value))
                        .map(([value, label]) => (
                          <PillRadioOption key={value} value={value} id={`agent-type-${value}`}>
                            {label}
                          </PillRadioOption>
                        ))}
                    </RadioGroup>
                  </div>
                </CollapsibleContent>
              </Collapsible>

              {/* Role Selection —— 默认收起；无外框 */}
              <Collapsible open={roleOpen} onOpenChange={setRoleOpen}>
                <CollapsibleTrigger
                  className="group flex w-full items-center gap-3 h-9 px-0 text-left focus-visible:outline-none"
                >
                  <span className="text-sm font-medium text-[var(--text-emphasis)]">角色身份</span>
                  <span className="ml-auto flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                    <span className="text-xs text-[var(--text-weak)]">已选</span>
                    {selectedRole?.name ?? "通用助手"}
                    <ChevronDown className="w-4 h-4 text-[var(--text-weak)] transition-transform duration-200 group-data-[state=open]:rotate-180" />
                  </span>
                </CollapsibleTrigger>
                <CollapsibleContent className="overflow-hidden">
                  <div className="px-0 pt-2 pb-1 space-y-3">
                    <RadioGroup
                      value={selectedRole?.id ?? "__general__"}
                      onValueChange={(value) => {
                        if (value === "__general__") {
                          setSelectedRole(null);
                          return;
                        }
                        const role = visibleRoles.find((r) => r.id === value);
                        setSelectedRole(role ?? null);
                      }}
                      className="flex flex-wrap gap-2"
                    >
                      <PillRadioOption value="__general__" id="role-general">
                        通用助手
                      </PillRadioOption>
                      {visibleRoles
                        .filter((role) => groupMode !== "multi-group" || selectedGroup.permissions.roles.includes(role.name))
                        .map((role) => (
                          <PillRadioOption key={role.id} value={role.id} id={`role-${role.id}`}>
                            {role.name}
                          </PillRadioOption>
                        ))}
                    </RadioGroup>

                    {/* Role Detail —— 选中具体角色后展示介绍卡片；未选（通用助手）展示通用介绍 */}
                    {(() => {
                      // 通用助手兜底文案（与具体角色介绍卡复用同一视觉容器）
                      const generalIntro = {
                        name: "通用助手",
                        skills: "web-search、file-reader、code-runner",
                        soul: "无固定行业偏好的通用 AI 伙伴，擅长日常问答、信息检索与轻量创作，按需切换专业度",
                      };
                      const display = selectedRole
                        ? {
                            name: selectedRole.name,
                            skills: selectedRole.skills.map((s) => s.name).join("、"),
                            soul: selectedRole.soul,
                          }
                        : generalIntro;

                      return (
                        <SurfaceInner className="overflow-hidden bg-[var(--bg-grey-normal)] relative rounded-[var(--radius-card)]">
                          <div className="p-4 space-y-3 relative z-10">
                            <div className="flex items-center gap-2">
                              <AgentAvatar
                                roleName={display.name}
                                size={28}
                              />
                              <p className="text-sm font-semibold text-[var(--text-emphasis)]">
                                {display.name}
                              </p>
                            </div>
                            <Separator />
                            <div className="space-y-1.5">
                              <p className="text-xs font-semibold text-[var(--text-emphasis)]">
                                角色技能
                              </p>
                              <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
                                {display.skills}
                              </p>
                            </div>
                            <div className="space-y-1.5">
                              <p className="text-xs font-semibold text-[var(--text-emphasis)]">
                                角色风格
                              </p>
                              <p
                                className="text-xs text-[var(--text-secondary)] leading-relaxed max-h-[120px] overflow-y-auto"
                                style={{ scrollbarGutter: "stable" }}
                              >
                                {display.soul}
                              </p>
                            </div>
                          </div>
                        </SurfaceInner>
                      );
                    })()}
                  </div>
                </CollapsibleContent>
              </Collapsible>

              {/* ===== 项目（可多选，默认不选） ===== */}
              <Collapsible open={projectOpen} onOpenChange={setProjectOpen}>
                <CollapsibleTrigger
                  className="group flex w-full items-center gap-3 h-9 px-0 text-left focus-visible:outline-none"
                >
                  <span className="text-sm font-medium text-[var(--text-emphasis)]">项目</span>
                  <span className="ml-auto flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                    <span className="text-xs text-[var(--text-weak)]">已选</span>
                    {selectedProjectIds.length === 0 ? "不关联项目" : `${selectedProjectIds.length} 个项目`}
                    <ChevronDown className="w-4 h-4 text-[var(--text-weak)] transition-transform duration-200 group-data-[state=open]:rotate-180" />
                  </span>
                </CollapsibleTrigger>
                <CollapsibleContent className="overflow-hidden">
                  <div className="pl-0 pr-1.5 pt-2.5 pb-1 space-y-2">
                    <HelperText as="p">
                      可选，关联项目后，会额外安装所选项目配置的特定技能、规范等。
                    </HelperText>
                    {projectPool.length === 0 ? (
                      <MetaText as="p" tone="weak">你名下暂无可关联的项目</MetaText>
                    ) : (
                      <div className="flex flex-wrap gap-2">
                        {projectPool.map((p) => {
                          const active = selectedProjectIds.includes(p.id);
                          return (
                            <button
                              key={p.id}
                              type="button"
                              onClick={() => toggleProject(p.id)}
                              className={`relative flex items-center justify-center whitespace-nowrap h-6 px-3 rounded-full border text-xs font-medium transition-colors cursor-pointer ${
                                active
                                  ? "bg-[var(--text-emphasis)] text-white border-[var(--text-emphasis)]"
                                  : "border-[var(--border)] bg-[var(--card)] text-[var(--text-emphasis)] hover:bg-[var(--accent)] hover:border-[var(--border-control)]"
                              }`}
                            >
                              {p.name}
                              {active && (
                                <CheckCircle2 className="absolute -top-1.5 -right-1.5 w-3.5 h-3.5 fill-white text-[var(--text-emphasis)]" />
                              )}
                            </button>
                          );
                        })}
                      </div>
                    )}
                  </div>
                </CollapsibleContent>
              </Collapsible>

              {/* ===== 共享范围 ===== */}
              {/* 仅当管控端「允许用户共享 agent」对当前用户/组开启时才展示；否则用户端不支持共享 */}
              {canShareAgent && (
              <Collapsible open={shareScopeOpen} onOpenChange={setShareScopeOpen}>
                <CollapsibleTrigger
                  className="group flex w-full items-center gap-3 h-9 px-0 text-left focus-visible:outline-none"
                >
                  <span className="text-sm font-medium text-[var(--text-emphasis)]">共享范围</span>
                  <span className="ml-auto flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                    <span className="text-xs text-[var(--text-weak)]">已选</span>
                    {shareScope === "private"
                      ? "仅自己"
                      : `共享（${shareGroupIds.length + shareUserIds.length}）`}
                    <ChevronDown className="w-4 h-4 text-[var(--text-weak)] transition-transform duration-200 group-data-[state=open]:rotate-180" />
                  </span>
                </CollapsibleTrigger>
                <CollapsibleContent className="overflow-hidden">
                  <div className="px-0 pt-2 pb-1 space-y-3">
                    {/* 共享范围类型选择 */}
                    <RadioGroup
                      value={shareScope}
                      onValueChange={(value) => {
                        setShareScope(value as "private" | "shared");
                        setShareScopeError("");
                        if (value === "private") {
                          setShareGroupIds([]);
                          setShareUserIds([]);
                          setShareScopeOpen(false);
                        }
                      }}
                      className="flex flex-wrap gap-2"
                    >
                      <PillRadioOption value="private" id="share-private">
                        仅自己
                      </PillRadioOption>
                      <PillRadioOption value="shared" id="share-shared">
                        共享
                      </PillRadioOption>
                    </RadioGroup>

                    {/* 共享目标选择：二级树形列表 */}
                    {shareScope === "shared" && (
                      <div className="space-y-3">
                        <ShareScopeTree
                          groups={MOCK_SHARE_TREE}
                          selectedGroupIds={shareGroupIds}
                          selectedUserIds={shareUserIds}
                          onSelectedGroupIdsChange={(ids) => { setShareGroupIds(ids); setShareScopeError(""); }}
                          onSelectedUserIdsChange={(ids) => { setShareUserIds(ids); setShareScopeError(""); }}
                          error={shareScopeError}
                          open={shareTreeOpen}
                          onOpenChange={setShareTreeOpen}
                        />
                        {/* 已选项标签：单行展示，超出 +N */}
                        {(shareGroupIds.length > 0 || shareUserIds.length > 0) && (
                          <ShareScopeTags
                            groups={MOCK_SHARE_TREE}
                            groupIds={shareGroupIds}
                            userIds={shareUserIds}
                            onRemoveGroup={(id) => {
                              // 移除整组：清掉 groupId，同时清掉该组所有成员（兼容“成员全选”归并出的组）
                              setShareGroupIds(prev => prev.filter(x => x !== id));
                              const g = MOCK_SHARE_TREE.find(g => g.id === id);
                              if (g) setShareUserIds(prev => prev.filter(uid => !g.members.some(m => m.id === uid)));
                            }}
                            onRemoveUser={(id) => setShareUserIds(prev => prev.filter(x => x !== id))}
                            onExpandClick={() => setShareTreeOpen(true)}
                          />
                        )}
                      </div>
                    )}
                  </div>
                </CollapsibleContent>
              </Collapsible>
              )}
            </div>
            </DialogBody>

            <DialogFooter className="shrink-0">
              <Button
                variant="tenant-outline"
                onClick={() => setShowCreate(false)}
              >
                取消
              </Button>
              <Button
                variant="tenant-dialog-confirm"
                onClick={handleCreate}
              >
                确认创建
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* ===== 更改共享范围弹窗 ===== */}
        <Dialog open={!!shareScopeDialog} onOpenChange={(open) => { if (!open) setShareScopeDialog(null); }}>
          <DialogContent size="md">
            <DialogHeader>
              <DialogTitle>更改共享范围</DialogTitle>
              <DialogDescription className="sr-only">
                更改「{shareScopeDialog?.name}」的共享范围，选择哪些分组或个人可以看到并使用此 Agent
              </DialogDescription>
            </DialogHeader>
            <div className="py-2 space-y-5">
              <HelperText as="p">
                选择可以看到并使用「{shareScopeDialog?.name}」的分组或个人。管理员可直接共享，普通用户需管理员授权。
              </HelperText>

              {/* 共享范围类型选择 */}
              <RadioGroup
                value={editShareScope}
                onValueChange={(value) => {
                  setEditShareScope(value as "private" | "shared");
                  setEditShareScopeError("");
                  if (value === "private") {
                    setEditShareGroupIds([]);
                    setEditShareUserIds([]);
                  }
                }}
                className="flex flex-wrap gap-2"
              >
                <PillRadioOption value="private" id="edit-share-private">
                  仅自己
                </PillRadioOption>
                <PillRadioOption value="shared" id="edit-share-shared">
                  共享
                </PillRadioOption>
              </RadioGroup>

              {/* 共享目标选择：二级树形列表 */}
              {editShareScope === "shared" && (
                <div className="space-y-3">
                  <ShareScopeTree
                    groups={MOCK_SHARE_TREE}
                    selectedGroupIds={editShareGroupIds}
                    selectedUserIds={editShareUserIds}
                    onSelectedGroupIdsChange={(ids) => { setEditShareGroupIds(ids); setEditShareScopeError(""); }}
                    onSelectedUserIdsChange={(ids) => { setEditShareUserIds(ids); setEditShareScopeError(""); }}
                    error={editShareScopeError}
                    open={editShareTreeOpen}
                    onOpenChange={setEditShareTreeOpen}
                  />
                  {/* 已选项标签：单行展示，超出 +N */}
                  {(editShareGroupIds.length > 0 || editShareUserIds.length > 0) && (
                    <ShareScopeTags
                      groups={MOCK_SHARE_TREE}
                      groupIds={editShareGroupIds}
                      userIds={editShareUserIds}
                      onRemoveGroup={(id) => {
                        // 移除整组：清掉 groupId，同时清掉该组所有成员（兼容“成员全选”归并出的组）
                        setEditShareGroupIds(prev => prev.filter(x => x !== id));
                        const g = MOCK_SHARE_TREE.find(g => g.id === id);
                        if (g) setEditShareUserIds(prev => prev.filter(uid => !g.members.some(m => m.id === uid)));
                      }}
                      onRemoveUser={(id) => setEditShareUserIds(prev => prev.filter(x => x !== id))}
                      onExpandClick={() => setEditShareTreeOpen(true)}
                    />
                  )}
                </div>
              )}
            </div>
            <DialogFooter>
              <Button
                variant="tenant-outline"
                onClick={() => setShareScopeDialog(null)}
              >
                取消
              </Button>
              <Button
                variant="tenant-dialog-confirm"
                onClick={handleShareScopeChange}
              >
                确认更改
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* CSS for animations */}
        <style>{`
          @keyframes pulse {
            0%, 100% { opacity: 0.15; }
            50% { opacity: 1; }
          }
        `}</style>
      </TenantLayout>

      {/* 角色配置进度弹窗：「新增角色」与「切换角色（方案1 单角色位）」确认后统一展示，
          进度走完 100% 才真正落库；「我知道了」仅隐藏弹窗，后台进度与落库不中断，
          隐藏后由卡片上的「角色切换中」状态提示继续告知用户 */}
      <RoleConfigProgressDialog
        progress={roleConfigProgress.progress}
        onDismiss={() => {
          const p = roleConfigProgress.progress;
          if (p?.mode === "switch" && p.agentId) {
            markRoleSwitching(p.agentId, p.items?.length ?? 1);
          }
          // 新增角色：与切换角色对齐——「我知道了」隐藏进度弹窗后，卡片继续展示「角色新增中（N）」
          if (p?.mode === "add" && p.agentId) {
            markRoleAdding(p.agentId, p.items?.length ?? 1);
          }
          roleConfigProgress.dismiss();
        }}
      />

      {/* 点击卡片「角色切换中」胶囊时弹出的进度查看弹窗 */}
      {switchingProgressDialogAgent && switchingProgressData[switchingProgressDialogAgent] && (
        <Dialog open onOpenChange={(open) => { if (!open) setSwitchingProgressDialogAgent(null); }}>
          <DialogContent size="lg" showCloseButton>
            <DialogHeader>
              <DialogTitle>
                正在切换角色（共 {switchingProgressData[switchingProgressDialogAgent].items.length} 个）
              </DialogTitle>
            </DialogHeader>
            <DialogBody className="px-6">
              <RoleConfigProgressContent
                roleName="通用助手"
                roleSoul=""
                skillCount={0}
                mode="switch"
                agentName={switchingProgressData[switchingProgressDialogAgent].agentName}
                items={switchingProgressData[switchingProgressDialogAgent].items}
                percent={75}
                stepIndex={2}
              />
            </DialogBody>
            <DialogFooter className="justify-center">
              <Button variant="tenant-dialog-confirm" onClick={() => setSwitchingProgressDialogAgent(null)}>
                我知道了
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}

      {/* ===== 组织变更相关弹窗 ===== */}
      {/* 迁移到新组织弹窗 */}
      <MigrateDialog
        open={!!showMigrateDialog}
        onOpenChange={(open) => { if (!open) setShowMigrateDialog(null); }}
        instanceName={showMigrateDialog?.instanceName ?? ""}
        availableGroups={[{ id: "grp-ai", name: "AI 组" }, { id: "grp-be", name: "后端研发同学" }]}
        onConfirm={() => {
          const targetId = showMigrateDialog?.id;
          setShowMigrateDialog(null);
          if (!targetId) return;
          // 1) 进入「迁移组织中」：实例保持已关机，迁移/移交按钮隐藏 → 置灰「设置」
          setClaws((prev) => prev.map((c) =>
            c.id === targetId ? { ...c, migrationPhase: "migrating" as const, transferPhase: undefined, status: "shutdown" as OpenClawStatus } : c
          ));
          toast.success("已发起迁移至新组织");
          // 2) 模拟迁移结果（约 2.5s）
          setTimeout(() => {
            const success = Math.random() > 0.3; // 70% 成功
            if (success) {
              // 迁移成功：标记成功 + 已关机 → 加载中
              setClaws((prev) => prev.map((c) =>
                c.id === targetId ? { ...c, migrationPhase: "success" as const, status: "loading" as OpenClawStatus } : c
              ));
              // 加载中 → 运行中，并清理组织变更态，回到普通运行卡片
              setTimeout(() => {
                setClaws((prev) => prev.map((c) =>
                  c.id === targetId
                    ? {
                        ...c,
                        migrationPhase: undefined,
                        status: "running" as OpenClawStatus,
                        groupChangeStatus: undefined,
                        groupChangeOriginalGroup: undefined,
                        groupChangeTransferTarget: undefined,
                      }
                    : c
                ));
                toast.success("迁移成功，实例已自动开机");
              }, 2000);
            } else {
              // 迁移失败：保持已关机，迁移/移交按钮重新出现
              setClaws((prev) => prev.map((c) =>
                c.id === targetId ? { ...c, migrationPhase: "fail" as const, status: "shutdown" as OpenClawStatus } : c
              ));
              toast.error("迁移失败，请重试");
            }
          }, 2500);
        }}
      />
      {/* 移交给同组织其他用户弹窗 */}
      <TransferDialog
        open={!!showTransferDialog}
        onOpenChange={(open) => { if (!open) setShowTransferDialog(null); }}
        instanceName={showTransferDialog?.instanceName ?? ""}
        originalGroupName={showTransferDialog?.groupName ?? ""}
        availableUsers={[{ userId: "bob@a.com" }, { userId: "carol@a.com" }, { userId: "frank@a.com" }]}
        onConfirm={(targetUserId) => {
          const targetId = showTransferDialog?.id;
          setShowTransferDialog(null);
          if (!targetId) return;
          // 1) 进入「移交待对方确认」：实例保持已关机，迁移/移交按钮隐藏 → 置灰「设置」，旁带「取消」按钮
          setClaws((prev) => prev.map((c) =>
            c.id === targetId
              ? { ...c, transferPhase: "pendingConfirm" as const, migrationPhase: undefined, status: "shutdown" as OpenClawStatus, groupChangeTransferTarget: targetUserId }
              : c
          ));
          toast.success(`已发起移交给 ${targetUserId}`);
          // 演示卡片 gc-transfer-fail：固定为「对方接收 → 移交中 → 最终移交失败」，用于展示失败态
          const isDemoTransferFailCard = targetId === "gc-transfer-fail";
          // 2) 模拟对方处理（约 3s 后随机接收/拒绝；演示卡片固定接收）
          setTimeout(() => {
            const accepted = isDemoTransferFailCard ? true : Math.random() > 0.4; // 60% 接收
            if (accepted) {
              // 对方接收：移交待对方确认 → 移交中
              setClaws((prev) => prev.map((c) =>
                c.id === targetId ? { ...c, transferPhase: "transferring" as const } : c
              ));
              // 移交中 → 移交完成（或演示卡片固定移交失败）
              setTimeout(() => {
                if (isDemoTransferFailCard) {
                  // 移交失败：保持已关机，移交按钮重新出现（该卡片仅支持移交，不支持迁移）
                  setClaws((prev) => prev.map((c) =>
                    c.id === targetId ? { ...c, transferPhase: "failed" as const, status: "shutdown" as OpenClawStatus } : c
                  ));
                  toast.error("移交失败，请重试");
                  return;
                }
                setClaws((prev) => prev.map((c) =>
                  c.id === targetId ? { ...c, transferPhase: "done" as const, status: "loading" as OpenClawStatus } : c
                ));
                // 加载中 → 运行中
                setTimeout(() => {
                  setClaws((prev) => prev.map((c) =>
                    c.id === targetId ? { ...c, status: "running" as OpenClawStatus } : c
                  ));
                  // 运行中短暂展示后，从我的列表移除（实例已归属对方）
                  setTimeout(() => {
                    setClaws((prev) => prev.filter((c) => c.id !== targetId));
                    toast.success("移交完成，实例已转交给对方");
                  }, 1200);
                }, 1500);
              }, 1500);
            } else {
              // 对方拒绝：保持已关机，迁移/移交按钮重新出现
              setClaws((prev) => prev.map((c) =>
                c.id === targetId ? { ...c, transferPhase: "rejected" as const, status: "shutdown" as OpenClawStatus } : c
              ));
              toast.error("对方已拒绝移交");
            }
          }, 3000);
        }}
      />
    </TooltipProvider>
  );
}
