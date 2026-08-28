/**
 * MemberManagement - 管控端用户管理页
 * Design: 「流动蓝图」Fluid Blueprint - Admin Side
 */
import React, { useState, useEffect, useCallback, useMemo } from "react";
import { Button, buttonVariants } from "@/components/ui/button";
import { Pagination } from "@/components/ui/pagination";
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";
import { HelperText } from "@/components/ui/Typography";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { StatusTag } from "@/components/ui/status-tag";
import { SurfaceInner } from "@/components/ui/Surface";
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableActionCell,
} from "@/components/ui/table";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogBody,
} from "@/components/ui/dialog";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { DateTimePicker } from "@/components/ui/datetime-picker";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
  Popover, PopoverContent, PopoverTrigger,
} from "@/components/ui/popover";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Alert, AlertDescription, AlertTitle, AlertOperationInfoIcon, AlertInfoIcon } from "@/components/ui/alert";
import { Checkbox } from "@/components/ui/checkbox";
import { InfoPopover } from "@/components/ui/info-popover";
import { MetaText, BodyMedium, BodyText } from "@/components/ui/Typography";
import { toast } from "sonner";
import {
  Search,
  Plus,
  ChevronDown,
  Info,
  Upload,
  Download,
  Trash2,
  UserX,
  UserCheck,
  MoreHorizontal,
  Pencil,
  Key,
  ChevronLeft,
  ChevronRight,
  Copy,
  CheckCircle,
  AlertCircle,
  CircleAlert,
  Loader2,
  X,
  FileText,
  ExternalLink,
  RefreshCw,
  Users,
  Check,
  FolderOpen,
  UserMinus,
  FolderPlus,
  ChevronUp,
  Link2,
  Filter,
  Eye,
  EyeOff,
  Shield,
  ShieldOff,
} from "lucide-react";
import { Switch } from "@/components/ui/switch";
import { useAdminMode } from "@/contexts/AdminModeContext";
import { notifyUsersChanged } from "@/lib/useUsersAndGroups";
import {
  PASSWORD_RULES_HINT,
  PASSWORD_MAX_LENGTH,
  validatePasswordStrength,
} from "@/lib/password-rules";
import AuthSourceImportDialog, { ConfiguredAuthSource } from "./AuthSourceImportDialog";
import { GroupSelect } from "@/components/GroupSelect";
import { TreeSelectFilter } from "@/components/_internal/TreeSelectFilter";
// ─── 新：组织视图（多层级 + 节点健康度 + 覆盖状态 + 就地决策，v2.0） ────
import NewGroupView from "./MemberManagement/GroupView";
import ProjectView from "./MemberManagement/ProjectView";
import { MOCK_USERS as MM_MOCK_USERS, MOCK_USER_OVERRIDES as MM_MOCK_OVERRIDES, MOCK_SYNC_RESULT as MM_MOCK_SYNC_RESULT, MOCK_GROUPS as MM_MOCK_GROUPS, MOCK_MANUAL_GROUPS as MM_MOCK_MANUAL_GROUPS, MOCK_USERS_MANUAL as MM_MOCK_USERS_MANUAL, getPrimaryDeptPath as mmGetPrimaryDeptPath } from "./MemberManagement/mock";
import { groupStore } from "./MemberManagement/groupStore";
import { userStore } from "./MemberManagement/userStore";
import type { UserOverrideInfo as MMUserOverrideInfo, UserOrg as MMUserOrg, UserGroup as MMUserGroup } from "./MemberManagement/types";
import AgentInstanceHandlingDialog, {
  type AffectedUser,
  type AffectedGroup,
  type MigrateTarget,
  type ParentMigration,
  type HandlingMode,
} from "./MemberManagement/AgentInstanceHandlingDialog";
import MmConfigDiffDialog, {
  buildMockInstanceCompare as mmBuildInstanceCompare,
  type InstanceConfigCompare as MmInstanceConfigCompare,
} from "./MemberManagement/ConfigDiffDialog";

import { buildGroupTree as mmBuildGroupTree } from "./MemberManagement/health";
import { AdminPageHeader } from "@/components/ui/admin-page-header";

const PAGE_SIZE = 10;

// ─── 组织选择框触发器（自适应截断） ──────────────────────────────────────────
function OverflowTooltipText({ text, className }: { text: string; className?: string }) {
  const textRef = React.useRef<HTMLSpanElement | null>(null);
  const [isOverflow, setIsOverflow] = React.useState(false);

  const checkOverflow = React.useCallback(() => {
    const el = textRef.current;
    if (!el) return;
    setIsOverflow(el.scrollWidth > el.clientWidth);
  }, []);

  React.useEffect(() => {
    checkOverflow();
    window.addEventListener("resize", checkOverflow);
    return () => window.removeEventListener("resize", checkOverflow);
  }, [checkOverflow, text]);

  const node = (
    <span ref={textRef} className={className}>
      {text}
    </span>
  );

  if (!isOverflow) return node;

  return (
    <Tooltip delayDuration={300}>
      <TooltipTrigger asChild>{node}</TooltipTrigger>
      <TooltipContent side="top" className="max-w-[320px] text-xs">
        {text}
      </TooltipContent>
    </Tooltip>
  );
}

function GroupSelectTrigger({ names, onRemove, onClear, lockedNames = [] }: { names: string[]; onRemove?: (name: string) => void; onClear?: () => void; lockedNames?: string[] }) {
  const [hover, setHover] = React.useState(false);
  const lockedSet = React.useMemo(() => new Set(lockedNames), [lockedNames]);
  if (names.length === 0) {
    return (
      <div className="w-full overflow-hidden">
        <button type="button" className="w-full flex items-center justify-between min-h-9 px-3 rounded-[4px] border border-border bg-white text-sm font-normal hover:border-[#355EF1] transition-colors">
          <span className="text-gray-400 truncate">请选择组织</span>
          <ChevronDown className="w-4 h-4 text-gray-500 shrink-0 ml-1" />
        </button>
      </div>
    );
  }

  return (
    <div className="w-full overflow-hidden" onMouseEnter={() => setHover(true)} onMouseLeave={() => setHover(false)}>
      <button type="button" className="w-full flex items-center flex-wrap gap-1 min-h-9 px-2 py-1.5 rounded-[4px] border border-border bg-white text-sm font-normal hover:border-[#355EF1] transition-colors relative pr-7">
        {names.map((name) => (
          <span
            key={name}
            className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded bg-[#f5f5f5] text-[#525252] text-[11px] shrink-0"
          >
            {name}
            {onRemove && !lockedSet.has(name) && (
              <span onClick={(e) => { e.stopPropagation(); onRemove(name); }} className="text-[#A3A3A3] hover:text-[#737373] cursor-pointer">
                <X className="w-3 h-3" />
              </span>
            )}
          </span>
        ))}
        {hover && onClear ? (
          <span
            onClick={(e) => { e.stopPropagation(); onClear(); }}
            className="absolute right-2 top-1/2 -translate-y-1/2 w-4 h-4 rounded-full bg-[#A3A3A3] hover:bg-[#737373] flex items-center justify-center cursor-pointer"
          >
            <X className="w-2.5 h-2.5 text-white" />
          </span>
        ) : (
          <ChevronDown className="absolute right-2 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500 shrink-0" />
        )}
      </button>
    </div>
  );
}

// 生成更多 mock 数据以演示翻页
// vpcType: "auto" = 我们帮用户创建的 VPC（自动分配）；"custom" = 用户指定 VPC
// vpcName: 自动分配时形如 "openclaw/{username}"，自定义时为 null
// hasVpcResources: 自动分配 VPC 下是否有关联云资源（null 表示自定义 VPC 不适用）
const MOCK_MEMBERS_BASE = [
  // 规则：有 Agent 必有关联资源；无 Agent 可能有也可能没有关联资源
  { id: "alice@acompany.com", role: "admin", status: "active", clawLimit: 5, tokenLimit: 100000, clawCount: 3, joinTime: "2025-01-10", vpcType: "auto" as const, vpcName: "openclaw/alice", hasVpcResources: true },   // 有 claw → 必有资源
  { id: "bob@acompany.com", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 1, joinTime: "2025-02-15", vpcType: "custom" as const, vpcName: null, hasVpcResources: null },              // 自定义 VPC
  { id: "carol@acompany.com", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 2, joinTime: "2025-03-01", vpcType: "auto" as const, vpcName: "openclaw/carol", hasVpcResources: true },  // 有 claw → 必有资源
  { id: "david@acompany.com", role: "member", status: "disabled", clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2025-03-20", vpcType: "auto" as const, vpcName: "openclaw/david", hasVpcResources: true }, // 无 claw，但还有残留资源
  { id: "eve@acompany.com", role: "member", status: "active", clawLimit: 5, tokenLimit: 80000, clawCount: 4, joinTime: "2025-04-05", vpcType: "custom" as const, vpcName: null, hasVpcResources: null },              // 自定义 VPC
  { id: "frank@acompany.com", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 1, joinTime: "2025-04-12", vpcType: "auto" as const, vpcName: "openclaw/frank", hasVpcResources: true },  // 有 claw → 必有资源
  { id: "grace@acompany.com", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 2, joinTime: "2025-05-01", vpcType: "custom" as const, vpcName: null, hasVpcResources: null },             // 自定义 VPC
  { id: "henry@acompany.com", role: "member", status: "disabled", clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2025-05-18", vpcType: "auto" as const, vpcName: "openclaw/henry", hasVpcResources: false }, // 无 claw，且资源已清空
  { id: "iris@acompany.com", role: "member", status: "active", clawLimit: 5, tokenLimit: 80000, clawCount: 3, joinTime: "2025-06-02", vpcType: "auto" as const, vpcName: "openclaw/iris", hasVpcResources: true },   // 有 claw → 必有资源
  { id: "jack@acompany.com", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 1, joinTime: "2025-06-20", vpcType: "custom" as const, vpcName: null, hasVpcResources: null },              // 自定义 VPC
  { id: "kate@acompany.com", role: "admin", status: "active", clawLimit: 5, tokenLimit: 100000, clawCount: 2, joinTime: "2025-07-05", vpcType: "auto" as const, vpcName: "openclaw/kate", hasVpcResources: true },   // 有 claw → 必有资源
  { id: "leo@acompany.com", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2025-07-22", vpcType: "auto" as const, vpcName: "openclaw/leo", hasVpcResources: false },    // 无 claw，资源已清空
  { id: "mike@acompany.com", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2026-03-20", vpcType: "custom" as const, vpcName: null, hasVpcResources: null },             // 自定义 VPC
  { id: "nina@acompany.com", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2026-03-20", vpcType: "auto" as const, vpcName: "openclaw/nina", hasVpcResources: true },   // 无 claw，但还有残留资源
  { id: "oscar@acompany.com", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2026-03-20", vpcType: "auto" as const, vpcName: "openclaw/oscar", hasVpcResources: false }, // 无 claw，资源已清空
  { id: "longname-user@very-long-domain-example.com", role: "member", status: "active", clawLimit: 3, tokenLimit: 50000, clawCount: 1, joinTime: "2026-05-01", vpcType: "auto" as const, vpcName: "openclaw/longname", hasVpcResources: true },  // 超长 ID 测试截断
  { id: "product-ops-admin@enterprise-acompany.com", role: "member", status: "active", clawLimit: 5, tokenLimit: 100000, clawCount: 2, joinTime: "2026-05-02", vpcType: "custom" as const, vpcName: null, hasVpcResources: null },             // 超长 ID 测试截断
];

// ─── OneID 模式用户 → 用户信息快速查表（按 userId） ───────────────
const MM_USERS_BY_ID = new Map<string, MMUserOrg>(MM_MOCK_USERS.map((u) => [u.userId, u]));

/** 获取用户所有 oneid-dept 类型部门的完整路径（主部门排首位） */
function getMmUserDeptPaths(userId: string): Array<{ id: string; path: string; isPrimary: boolean }> {
  const user = MM_USERS_BY_ID.get(userId);
  if (!user) return [];
  const deptGroupIds = user.groupIds.filter((gid) => {
    const g = MM_MOCK_GROUPS.find((g) => g.id === gid);
    return g?.source === "oneid-dept";
  });
  if (deptGroupIds.length === 0) return [];
  return deptGroupIds
    .map((gid) => ({
      id: gid,
      path: mmGetPrimaryDeptPath(gid, MM_MOCK_GROUPS),
      isPrimary: gid === user.primaryGroupId,
    }))
    .sort((a, b) => (a.isPrimary ? -1 : b.isPrimary ? 1 : 0));
}

/** 获取用户的「组织」展示项（组织架构 + 自定义组织），用于全部视图的组织列 */
function getMmUserGroupItems(userId: string): Array<{
  id: string;
  path: string;
  kind: "oneid-dept" | "oneid-group";
}> {
  const user = MM_USERS_BY_ID.get(userId);
  if (!user) return [];
  const result: Array<{ id: string; path: string; kind: "oneid-dept" | "oneid-group" }> = [];
  user.groupIds.forEach((gid) => {
    const g = MM_MOCK_GROUPS.find((g) => g.id === gid);
    if (!g) return;
    if (g.source === "oneid-dept") {
      result.push({ id: gid, path: mmGetPrimaryDeptPath(gid, MM_MOCK_GROUPS), kind: "oneid-dept" });
    } else if (g.source === "oneid-group") {
      result.push({ id: gid, path: mmGetPrimaryDeptPath(gid, MM_MOCK_GROUPS), kind: "oneid-group" });
    }
  });
  return result;
}

// ─── 普通模式：MOCK_USERS_MANUAL 扩展为 member 兼容的数据结构 ─────────────
// 补齐 Agent/VPC 相关字段，便于全部视图渲染
const MM_MANUAL_MEMBER_EXTRAS: Record<string, { clawLimit: number; tokenLimit: number; clawCount: number; joinTime: string; vpcType: "auto" | "custom"; vpcName: string | null; hasVpcResources: boolean | null }> = {
  // ── 产品组 ──
  "anna@acompany.com":   { clawLimit: 3, tokenLimit: 50000,  clawCount: 2, joinTime: "2025-06-05", vpcType: "auto",   vpcName: "openclaw/anna",   hasVpcResources: true },
  "bill@acompany.com":   { clawLimit: 5, tokenLimit: 100000, clawCount: 3, joinTime: "2025-06-05", vpcType: "auto",   vpcName: "openclaw/bill",   hasVpcResources: true },
  "cara@acompany.com":   { clawLimit: 3, tokenLimit: 50000,  clawCount: 0, joinTime: "2025-06-10", vpcType: "custom", vpcName: null,              hasVpcResources: null },
  // ── 研发组 ──
  "daniel@acompany.com": { clawLimit: 10, tokenLimit: 200000, clawCount: 4, joinTime: "2025-06-01", vpcType: "auto", vpcName: "openclaw/daniel", hasVpcResources: true },
  "eric@acompany.com":   { clawLimit: 5, tokenLimit: 100000, clawCount: 2, joinTime: "2025-06-01", vpcType: "auto",  vpcName: "openclaw/eric",   hasVpcResources: true },
  // ── 研发-前端 ──
  "fiona@acompany.com":  { clawLimit: 3, tokenLimit: 50000, clawCount: 1, joinTime: "2025-06-15", vpcType: "auto",   vpcName: "openclaw/fiona",   hasVpcResources: true },
  "george@acompany.com": { clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2025-06-20", vpcType: "auto",   vpcName: "openclaw/george",  hasVpcResources: false },
  "helen@acompany.com":  { clawLimit: 3, tokenLimit: 50000, clawCount: 2, joinTime: "2025-07-01", vpcType: "custom", vpcName: null,               hasVpcResources: null },
  "ivan@acompany.com":   { clawLimit: 3, tokenLimit: 50000, clawCount: 1, joinTime: "2025-07-05", vpcType: "auto",   vpcName: "openclaw/ivan",    hasVpcResources: true },
  // ── 研发-后端 ──
  "jason@acompany.com":  { clawLimit: 3, tokenLimit: 50000, clawCount: 3, joinTime: "2025-07-10", vpcType: "auto",   vpcName: "openclaw/jason",   hasVpcResources: true },
  "kelly@acompany.com":  { clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2025-07-15", vpcType: "custom", vpcName: null,               hasVpcResources: null },
  "lucas@acompany.com":  { clawLimit: 5, tokenLimit: 80000, clawCount: 2, joinTime: "2025-07-20", vpcType: "auto",   vpcName: "openclaw/lucas",   hasVpcResources: true },
  // ── 设计组 ──
  "mia@acompany.com":    { clawLimit: 3, tokenLimit: 50000, clawCount: 1, joinTime: "2025-08-01", vpcType: "auto",   vpcName: "openclaw/mia",     hasVpcResources: true },
  "nick@acompany.com":   { clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2025-08-05", vpcType: "auto",   vpcName: "openclaw/nick",    hasVpcResources: false },
  // ── 产品运营与市场推广团队 ──
  "olivia@acompany.com": { clawLimit: 3, tokenLimit: 50000,  clawCount: 1, joinTime: "2025-09-01", vpcType: "auto",   vpcName: "openclaw/olivia", hasVpcResources: true },
  "paul@acompany.com":   { clawLimit: 5, tokenLimit: 100000, clawCount: 2, joinTime: "2025-09-05", vpcType: "auto",   vpcName: "openclaw/paul",   hasVpcResources: true },
  "quinn@acompany.com":  { clawLimit: 3, tokenLimit: 50000,  clawCount: 0, joinTime: "2025-09-10", vpcType: "custom", vpcName: null,               hasVpcResources: null },
  // ── 未分配组织 ──
  "ryan@acompany.com":   { clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2025-10-01", vpcType: "auto",   vpcName: "openclaw/ryan",    hasVpcResources: false },
  "susan@acompany.com":  { clawLimit: 3, tokenLimit: 50000, clawCount: 1, joinTime: "2025-10-05", vpcType: "auto",   vpcName: "openclaw/susan",   hasVpcResources: true },
};

/** 普通模式下：由 MOCK_USERS_MANUAL + 扩展字段组合得到的 members 基础数据（19 人） */
const MOCK_MEMBERS_MANUAL_BASE = MM_MOCK_USERS_MANUAL.map((u) => {
  const extras = MM_MANUAL_MEMBER_EXTRAS[u.userId] ?? {
    clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2025-06-01", vpcType: "auto" as const, vpcName: `openclaw/${u.userId.split("@")[0]}`, hasVpcResources: false,
  };
  return {
    id: u.userId,
    role: u.role ?? "member",
    status: u.status ?? "active",
    ...extras,
  };
});

// ─── OneID 模式：MOCK_USERS 扩展为 member 兼容的数据结构 ─────────────
// 其中 alice~oscar 15 人复用 MOCK_MEMBERS_BASE 里已 mock 的 Agent/VPC 字段；
// ceo / tim / peter 3 人是组织视图新增的高管，需要单独 mock Agent/VPC 字段
const MM_ONEID_EXTRA_MEMBERS: Record<string, { clawLimit: number; tokenLimit: number; clawCount: number; joinTime: string; vpcType: "auto" | "custom"; vpcName: string | null; hasVpcResources: boolean | null }> = {
  "ceo@acompany.com":   { clawLimit: 10, tokenLimit: 200000, clawCount: 0, joinTime: "2024-12-01", vpcType: "auto", vpcName: "openclaw/ceo",   hasVpcResources: false },
  "tim@acompany.com":   { clawLimit: 5,  tokenLimit: 100000, clawCount: 2, joinTime: "2024-12-15", vpcType: "auto", vpcName: "openclaw/tim",   hasVpcResources: true },
  "peter@acompany.com": { clawLimit: 5,  tokenLimit: 100000, clawCount: 1, joinTime: "2024-12-15", vpcType: "auto", vpcName: "openclaw/peter", hasVpcResources: true },
};

/** OneID 模式下：由 MOCK_USERS + 扩展字段组合得到的 members 基础数据（18 人） */
const MOCK_MEMBERS_ONEID_BASE = MM_MOCK_USERS.map((u) => {
  // 1) 优先用 MOCK_MEMBERS_BASE 里已 mock 好的字段（alice~oscar 15 人）
  const baseMember = MOCK_MEMBERS_BASE.find((m) => m.id === u.userId);
  if (baseMember) {
    return baseMember;
  }
  // 2) 否则用 MM_ONEID_EXTRA_MEMBERS 里 ceo / tim / peter 的 mock
  const extras = MM_ONEID_EXTRA_MEMBERS[u.userId] ?? {
    clawLimit: 3, tokenLimit: 50000, clawCount: 0, joinTime: "2025-01-01", vpcType: "auto" as const, vpcName: `openclaw/${u.userId.split("@")[0]}`, hasVpcResources: false,
  };
  return {
    id: u.userId,
    role: (u.role ?? "member") as "admin" | "member",
    status: (u.status ?? "active") as "active" | "disabled",
    ...extras,
  };
}) as typeof MOCK_MEMBERS_BASE;

/** 普通模式下：构造组织完整路径（如 "研发组 / 研发-前端"） */
function getManualGroupPath(groupId: string): string {
  const map = new Map(MM_MOCK_MANUAL_GROUPS.map((g) => [g.id, g]));
  const chain: string[] = [];
  let cur = map.get(groupId);
  while (cur) {
    chain.unshift(cur.name);
    cur = cur.parentId ? map.get(cur.parentId) : undefined;
  }
  return chain.length > 0 ? chain.join(" / ") : "—";
}

/** 普通模式下：获取某用户的组织完整路径列表 */
function getManualUserGroupPaths(userId: string): Array<{ id: string; path: string }> {
  const user = MM_MOCK_USERS_MANUAL.find((u) => u.userId === userId);
  if (!user) return [];
  return user.groupIds.map((gid) => ({ id: gid, path: getManualGroupPath(gid) }));
}

/**
 * 获取某用户加入的「项目」完整路径列表。
 * 数据来自跨页共享的 userStore（成员归属）+ groupStore（source==='project' 的项目树），
 * 与「组织视图」「项目资产管理」保持同源，项目的新建/加人/移除都会实时反映。
 * 普通模式的用户集（MOCK_USERS_MANUAL）不在 userStore 中，回退按其静态 groupIds 解析。
 */
function getUserProjectPaths(userId: string): Array<{ id: string; path: string }> {
  const storeUser = userStore.getById(userId);
  const groupIds =
    storeUser?.groupIds ??
    MM_MOCK_USERS_MANUAL.find((u) => u.userId === userId)?.groupIds ??
    [];
  if (groupIds.length === 0) return [];
  const projectMap = new Map(
    groupStore.getAll().filter((g) => g.source === "project").map((g) => [g.id, g]),
  );
  const buildPath = (id: string): string => {
    const names: string[] = [];
    let cur = projectMap.get(id);
    let guard = 0;
    while (cur && guard < 20) {
      names.unshift(cur.name);
      cur = cur.parentId ? projectMap.get(cur.parentId) : undefined;
      guard++;
    }
    return names.join(" / ");
  };
  return groupIds
    .filter((gid) => projectMap.has(gid))
    .map((gid) => ({ id: gid, path: buildPath(gid) }));
}

// ─── Mock 部门数据（仅 OneID 模式使用） ─────────────────────────────────────────
interface DepartmentNode {
  id: string;
  name: string;
  path?: string;
  children?: DepartmentNode[];
}

const MOCK_DEPARTMENTS: DepartmentNode[] = [
  {
    id: "dept-root",
    name: "A公司",
    path: "A公司",
    children: [
      {
        id: "dept-tech",
        name: "技术部",
        path: "A公司/技术部",
        children: [
          { id: "dept-fe", name: "前端组", path: "A公司/技术部/前端组" },
          { id: "dept-be", name: "后端组", path: "A公司/技术部/后端组" },
          { id: "dept-ai", name: "AI 组", path: "A公司/技术部/AI 组" },
        ],
      },
      {
        id: "dept-product",
        name: "产品部",
        path: "A公司/产品部",
        children: [
          { id: "dept-pm", name: "产品策划", path: "A公司/产品部/产品策划" },
          { id: "dept-design", name: "设计组", path: "A公司/产品部/设计组" },
        ],
      },
      { id: "dept-hr", name: "人力资源", path: "A公司/人力资源" },
      { id: "dept-finance", name: "财务部", path: "A公司/财务部" },
    ],
  },
];

/** 用户归属 mock 映射 */
const MOCK_MEMBER_DEPARTMENTS: Record<string, string> = {
  "alice@acompany.com": "A公司/技术部/前端组",
  "bob@acompany.com": "A公司/技术部/后端组",
  "carol@acompany.com": "A公司/技术部/AI 组",
  "david@acompany.com": "A公司/产品部/产品策划",
  "eve@acompany.com": "A公司/产品部/设计组",
  "frank@acompany.com": "A公司/技术部/前端组",
  "grace@acompany.com": "A公司/技术部/后端组",
  "henry@acompany.com": "A公司/人力资源",
  "iris@acompany.com": "A公司/技术部/AI 组",
  "jack@acompany.com": "A公司/财务部",
  "kate@acompany.com": "A公司/技术部/前端组",
  "leo@acompany.com": "A公司/产品部/产品策划",
  "mike@acompany.com": "A公司/技术部/后端组",
  "nina@acompany.com": "A公司/产品部/设计组",
  "oscar@acompany.com": "A公司/财务部",
};

const LAST_CLAW_LIMIT = 3;
const LAST_TOKEN_LIMIT = 50000;
const DEFAULT_NEW_MEMBER_CLAW_LIMIT = 5;
const DEFAULT_NEW_MEMBER_TOKEN_LIMIT = 500000;

// ─── 平台策略：预设策略默认值（可被管理员修改） ─────────────────────────────────
const PRESET_POLICY_CLAW_LIMIT = 3;
const PRESET_POLICY_TOKEN_LIMIT = 50000;

// ─── 平台策略：按组织配额（模拟平台策略页配置的结果） ────────────────────────────
/** 普通模式组织配额 */
const GROUP_POLICY_QUOTAS: Record<string, { clawLimit: number; tokenLimit: number }> = {
  "mgrp-product": { clawLimit: 3, tokenLimit: 50000 },
  "mgrp-rd": { clawLimit: 5, tokenLimit: 100000 },
  "mgrp-rd-fe": { clawLimit: 5, tokenLimit: 100000 },
  "mgrp-rd-be": { clawLimit: 5, tokenLimit: 100000 },
  "mgrp-design": { clawLimit: 3, tokenLimit: 50000 },
  "mgrp-ops": { clawLimit: 3, tokenLimit: 50000 },
};
/** OneID 模式组织配额（按部门/用户组） */
const ONEID_GROUP_POLICY_QUOTAS: Record<string, { clawLimit: number; tokenLimit: number }> = {
  "dept-tech": { clawLimit: 5, tokenLimit: 100000 },
  "dept-fe": { clawLimit: 5, tokenLimit: 100000 },
  "dept-be": { clawLimit: 5, tokenLimit: 100000 },
  "dept-ai": { clawLimit: 10, tokenLimit: 200000 },
  "dept-product": { clawLimit: 3, tokenLimit: 80000 },
  "dept-pm": { clawLimit: 3, tokenLimit: 80000 },
  "dept-design": { clawLimit: 3, tokenLimit: 50000 },
  "dept-operation": { clawLimit: 3, tokenLimit: 50000 },
  "og-frontend": { clawLimit: 5, tokenLimit: 100000 },
  "og-backend": { clawLimit: 5, tokenLimit: 100000 },
  "og-ai-core": { clawLimit: 10, tokenLimit: 200000 },
};

// ─── 用户在组织中创建的 Agent 实例（mock） ────────────────────────────────────
/** 格式：userId -> groupId -> 实例列表 */
const MOCK_USER_GROUP_AGENTS: Record<string, Record<string, Array<{ id: string; name: string }>>> = {
  // 普通模式：fiona 在研发-前端有 1 个实例
  "fiona@acompany.com": {
    "mgrp-rd-fe": [
      { id: "claw-fiona-1", name: "Fiona 的前端助手" },
    ],
  },
  // 普通模式：lucas 在研发-后端有 2 个实例（lucas 兼任前端+后端）
  "lucas@acompany.com": {
    "mgrp-rd-be": [
      { id: "claw-lucas-1", name: "Lucas 的后端服务" },
      { id: "claw-lucas-2", name: "Lucas 的 API 测试" },
    ],
  },
  // OneID 模式：alice 在「前端基础架构与工程效能研发协作组」(og-frontend) 有实例
  // —— 该用户组 alice 确实归属（见 MOCK_USERS）且可改上级组织，故可同时验收 编辑用户①②、改上级⑥
  "alice@acompany.com": {
    "og-frontend": [
      { id: "claw-alice-1", name: "Alice 的代码助手" },
      { id: "claw-alice-2", name: "Alice 的文档生成器" },
      { id: "claw-alice-3", name: "Alice 的测试工具" },
    ],
  },
  // OneID 模式：bob 在「后端研发同学」(og-backend) 有实例（bob 归属该组且可改上级）
  "bob@acompany.com": {
    "og-backend": [
      { id: "claw-bob-1", name: "Bob 的组件库助手" },
    ],
  },
  // 普通模式：ryan 处于「未分配组织」且名下有存量实例（__global__）
  // —— 用于验收弹窗③：未分配组织 → 加入新组织（随用户迁移到新组织，移交不可用）
  "ryan@acompany.com": {
    "__global__": [
      { id: "claw-ryan-1", name: "Ryan 的通用助手" },
      { id: "claw-ryan-2", name: "Ryan 的数据分析工具" },
    ],
  },
};

/**
 * 为「被移除的原组织」构建存量实例处理配置（弹窗①②③ 的 per-group 数据）
 * - migrateTargets：用户的新组织（用于「随用户迁移到新组织」下拉）
 * - migrateToUnassigned：新组织为空 → migrate 改为「回退为未分配组织配置」（弹窗②）
 * - transferTargets：原组织内仍在的其他用户（用于「移交给同组织其他用户」）
 * - disabledModes.transfer：无同组用户可接手时禁用移交
 * - oldGroupIds 为空 + 有 __global__ 实例 + 加入了新组织 → 未分配组织加入新组织（弹窗③）
 */
function mmBuildAffectedGroups(
  targetUserId: string,
  removedGroupIds: string[],
  newGroupIds: string[],
  usersList: Array<{ userId: string; groupIds: string[] }>,
  allGroups: MMUserGroup[],
  oldGroupIds: string[] = [],
): AffectedGroup[] {
  const result: AffectedGroup[] = [];
  const newTargets: MigrateTarget[] = newGroupIds.map((gId) => ({
    id: gId,
    name: mmGetPrimaryDeptPath(gId, allGroups),
  }));
  removedGroupIds.forEach((gId) => {
    const instances = MOCK_USER_GROUP_AGENTS[targetUserId]?.[gId];
    if (!instances || instances.length === 0) return;
    const sameGroupUsers = usersList
      .filter((u) => u.userId !== targetUserId && u.groupIds.includes(gId))
      .map((u) => ({ userId: u.userId }));
    const movingToUnassigned = newGroupIds.length === 0;
    const disabledModes: Partial<Record<HandlingMode, string>> = {};
    if (sameGroupUsers.length === 0) {
      disabledModes.transfer = "原组织无其他同组用户可接手";
    }
    result.push({
      groupId: gId,
      groupName: mmGetPrimaryDeptPath(gId, allGroups),
      instances,
      migrateTargets: newTargets,
      transferTargets: sameGroupUsers,
      migrateToUnassigned: movingToUnassigned,
      disabledModes,
      userSelfOptions: { allowTransfer: sameGroupUsers.length > 0, allowMigrate: true },
    });
  });

  // 弹窗③：用户原本处于「未分配组织」且名下有全局存量实例（__global__），本次加入了新组织
  // —— 实例随用户迁移到新组织；无原组织同组用户 → 移交禁用；自行处理仅允许迁移
  const movingFromUnassigned = oldGroupIds.length === 0;
  const globalInstances = MOCK_USER_GROUP_AGENTS[targetUserId]?.["__global__"];
  if (
    movingFromUnassigned &&
    newGroupIds.length > 0 &&
    globalInstances &&
    globalInstances.length > 0
  ) {
    result.push({
      groupId: "__global__",
      groupName: "未分配组织（全局配置）",
      instances: globalInstances,
      migrateTargets: newTargets,
      transferTargets: [],
      migrateToUnassigned: false,
      disabledModes: { transfer: "无原组织同组用户可接手" },
      userSelfOptions: { allowTransfer: false, allowMigrate: true },
    });
  }
  return result;
}

// ─── 组织数据模型 ─────────────────────────────────────────────────────────────
export interface MemberGroup {
  id: string;
  name: string;
  memberIds: string[];
  createdAt: string;
}

export const MOCK_GROUPS_INIT: MemberGroup[] = [
  { id: "grp-1", name: "产品组", memberIds: ["carol@acompany.com", "david@acompany.com", "eve@acompany.com", "alice@acompany.com"], createdAt: "2025-06-01" },
  { id: "grp-2", name: "研发组", memberIds: ["bob@acompany.com", "frank@acompany.com", "grace@acompany.com", "kate@acompany.com"], createdAt: "2025-06-05" },
  { id: "grp-3", name: "设计组", memberIds: ["iris@acompany.com", "jack@acompany.com"], createdAt: "2025-07-10" },
  { id: "grp-4", name: "产品运营与市场推广团队", memberIds: ["leo@acompany.com", "nina@acompany.com"], createdAt: "2025-08-15" },
];

// ─── 组织关联配置 mock 数据 ──────────────────────────────────────────────────
interface GroupRelatedConfig {
  type: string;
  typePath: string; // 跳转路径
  items: { id: string; name: string }[];
}

const MOCK_GROUP_CONFIGS: Record<string, GroupRelatedConfig[]> = {
  "grp-1": [
    { type: "模型配置", typePath: "/admin/model-config", items: [
      { id: "m1", name: "腾讯云混元 - 混元 TurboS Latest" },
    ]},
    { type: "企业技能", typePath: "/admin/skill-config", items: [
      { id: "s1", name: "Web 搜索" }, { id: "s2", name: "代码解释器" }, { id: "s3", name: "文档分析" },
    ]},
  ],
  "grp-2": [
    { type: "模型配置", typePath: "/admin/model-config", items: [
      { id: "m1", name: "腾讯云混元 - 混元 TurboS Latest" }, { id: "m2", name: "腾讯云 DeepSeek - DeepSeek V3 0324" },
    ]},
    { type: "企业技能", typePath: "/admin/skill-config", items: [
      { id: "s1", name: "Web 搜索" }, { id: "s2", name: "代码解释器" }, { id: "s3", name: "文档分析" },
      { id: "s4", name: "图片生成" }, { id: "s5", name: "数据分析" }, { id: "s6", name: "翻译助手" },
      { id: "s7", name: "知识库问答" }, { id: "s8", name: "邮件撰写" }, { id: "s9", name: "会议纪要" }, { id: "s10", name: "PPT 生成" },
    ]},
    { type: "通道配置", typePath: "/admin/channel-config", items: [
      { id: "c1", name: "默认通道" }, { id: "c2", name: "高级通道" }, { id: "c3", name: "专属通道" },
    ]},
  ],
  "grp-3": [],
  "grp-4": [
    { type: "企业技能", typePath: "/admin/skill-config", items: [
      { id: "s1", name: "Web 搜索" }, { id: "s4", name: "图片生成" },
    ]},
  ],
};
function generatePassword(): string {
  const chars = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789";
  let pwd = "Oc@";
  for (let i = 0; i < 8; i++) {
    pwd += chars[Math.floor(Math.random() * chars.length)];
  }
  return pwd;
}

type AddTokenCycleKind = "natural" | "custom";
type AddTokenNaturalPeriod = "day" | "month" | "year";
type AddTokenRefresh = "daily" | "monthly" | "yearly" | "none";
type AddTokenLimit = number | "unlimited";

interface AddTokenQuotaConfig {
  cycleKind: AddTokenCycleKind;
  naturalPeriod: AddTokenNaturalPeriod;
  start: string;
  end: string | null;
  refresh: AddTokenRefresh;
  limit: AddTokenLimit;
}

function getLocalDateTimeMinute(date = new Date()): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function localStringToISO(s: string | null | undefined): string | null {
  if (!s) return null;
  const d = new Date(s);
  return Number.isNaN(d.getTime()) ? null : d.toISOString();
}

function isoToLocalString(iso: string | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? "" : getLocalDateTimeMinute(d);
}

function startPlusOneMonth(start: string): string {
  const base = start ? new Date(start) : new Date();
  if (Number.isNaN(base.getTime())) return getLocalDateTimeMinute();
  base.setMonth(base.getMonth() + 1);
  return getLocalDateTimeMinute(base);
}

function makeDefaultAddTokenQuota(): AddTokenQuotaConfig {
  return {
    cycleKind: "custom",
    naturalPeriod: "day",
    start: getLocalDateTimeMinute(),
    end: null,
    refresh: "daily",
    limit: DEFAULT_NEW_MEMBER_TOKEN_LIMIT,
  };
}

function resolveTokenLimit(limit: AddTokenLimit): number {
  return limit === "unlimited" ? -1 : limit;
}

function makeEmptyNewMember() {
  const tokenQuota = makeDefaultAddTokenQuota();
  return {
    id: "",
    role: "member",
    clawLimit: DEFAULT_NEW_MEMBER_CLAW_LIMIT,
    tokenLimit: resolveTokenLimit(tokenQuota.limit),
    tokenQuota,
    notificationEmail: "",
    groupIds: [] as string[],
  };
}

const emptyNewMember = makeEmptyNewMember();

const emptyEditForm = {
  id: "", role: "member", clawLimit: LAST_CLAW_LIMIT, tokenLimit: LAST_TOKEN_LIMIT, groupIds: [] as string[],
};

const emptyResetForm = {
  notificationEmail: "",
  newPassword: "",
  confirmPassword: "",
};

// ─── TokenLimit 输入框：默认填数字，右侧「无限制」文字按钮切换 ─────────────────
const TOKEN_UNLIMITED = -1;

function TokenLimitInput({
  value,
  onChange,
}: {
  value: number;
  onChange: (v: number) => void;
}) {
  const isUnlimited = value === TOKEN_UNLIMITED;
  const [inputStr, setInputStr] = React.useState<string>(isUnlimited ? "" : String(value));

  React.useEffect(() => {
    if (!isUnlimited) setInputStr(String(value));
  }, [value, isUnlimited]);

  return (
    <div className="space-y-2">
      <Select
        value={isUnlimited ? "unlimited" : "custom"}
        onValueChange={(v) => {
          if (v === "unlimited") {
            onChange(TOKEN_UNLIMITED);
          } else {
            setInputStr("50000");
            onChange(50000);
          }
        }}
      >
        <SelectTrigger className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="custom">自定义数量</SelectItem>
          <SelectItem value="unlimited">无限制</SelectItem>
        </SelectContent>
      </Select>
      {!isUnlimited && (
        <Input
          type="number"
          value={inputStr}
          onChange={(e) => {
            setInputStr(e.target.value);
            if (e.target.value !== "") onChange(Number(e.target.value));
          }}
          onBlur={() => {
            if (inputStr === "" || isNaN(Number(inputStr))) {
              setInputStr("0");
              onChange(0);
            }
          }}
          placeholder="请输入数量"
        />
      )}
    </div>
  );
}

function AddTokenQuotaEditor({
  value,
  onChange,
}: {
  value: AddTokenQuotaConfig;
  onChange: (v: AddTokenQuotaConfig) => void;
}) {
  const isUnlimited = value.limit === "unlimited";
  const setLimitMode = (mode: "number" | "unlimited") => {
    onChange({ ...value, limit: mode === "unlimited" ? "unlimited" : DEFAULT_NEW_MEMBER_TOKEN_LIMIT });
  };
  const setLimitNumber = (raw: string) => {
    if (raw === "") {
      onChange({ ...value, limit: 0 });
      return;
    }
    const n = Number(raw);
    if (!Number.isNaN(n)) onChange({ ...value, limit: Math.max(0, n) });
  };

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label>周期类型</Label>
        <Select
          value={value.cycleKind}
          onValueChange={(v) => {
            const next = v as AddTokenCycleKind;
            onChange({
              ...value,
              cycleKind: next,
              start: next === "custom" ? value.start || getLocalDateTimeMinute() : value.start,
            });
          }}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="natural">自然周期</SelectItem>
            <SelectItem value="custom">自定义周期</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {value.cycleKind === "natural" ? (
        <div className="space-y-2">
          <Label>周期长度</Label>
          <Select value={value.naturalPeriod} onValueChange={(v) => onChange({ ...value, naturalPeriod: v as AddTokenNaturalPeriod })}>
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="day">每日</SelectItem>
              <SelectItem value="month">每月</SelectItem>
              <SelectItem value="year">每年</SelectItem>
            </SelectContent>
          </Select>
        </div>
      ) : (
        <>
          <div className="space-y-2">
            <Label>起始时间</Label>
            <DateTimePicker
              value={localStringToISO(value.start)}
              onChange={(v) => onChange({ ...value, start: isoToLocalString(v) || getLocalDateTimeMinute() })}
              placeholder="选择起始时间"
            />
          </div>
          <div className="space-y-2">
            <Label>终止时间</Label>
            <RadioGroup
              value={value.end ? "custom" : "none"}
              onValueChange={(v) => onChange({
                ...value,
                end: v === "custom" ? (value.end || startPlusOneMonth(value.start)) : null,
              })}
              className="flex items-center gap-5"
            >
              <div className="flex items-center gap-2">
                <RadioGroupItem value="none" id="add-token-end-none" />
                <Label htmlFor="add-token-end-none" className="font-normal cursor-pointer">无终止时间</Label>
              </div>
              <div className="flex items-center gap-2">
                <RadioGroupItem value="custom" id="add-token-end-custom" />
                <Label htmlFor="add-token-end-custom" className="font-normal cursor-pointer">设置终止时间</Label>
              </div>
            </RadioGroup>
            {value.end && (
              <DateTimePicker
                value={localStringToISO(value.end)}
                onChange={(v) => onChange({ ...value, end: isoToLocalString(v) || startPlusOneMonth(value.start) })}
                placeholder="选择终止时间"
                minDate={value.start ? new Date(value.start) : undefined}
              />
            )}
          </div>
          <div className="space-y-2">
            <Label>刷新方式</Label>
            <Select value={value.refresh} onValueChange={(v) => onChange({ ...value, refresh: v as AddTokenRefresh })}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="daily">每日刷新</SelectItem>
                <SelectItem value="monthly">每月刷新</SelectItem>
                <SelectItem value="yearly">每年刷新</SelectItem>
                <SelectItem value="none">不刷新</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </>
      )}

      <div className="space-y-2">
        <Label>Tokens 上限值</Label>
        <Select value={isUnlimited ? "unlimited" : "custom"} onValueChange={(v) => setLimitMode(v === "unlimited" ? "unlimited" : "number")}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="custom">自定义数量</SelectItem>
            <SelectItem value="unlimited">无限制</SelectItem>
          </SelectContent>
        </Select>
        {!isUnlimited && (
          <Input
            type="number"
            min={0}
            value={typeof value.limit === "number" ? value.limit : ""}
            onChange={(e) => setLimitNumber(e.target.value)}
            placeholder="请输入数量"
          />
        )}
      </div>
    </div>
  );
}

// ─── 添加用户表单（无密码） ────────────────────────
function AddMemberFormFields({
  values,
  onChange,
  existingMemberIds = [],
  groups = [],
  userGroups = [],
  onOpenCreateGroupDialog,
  groupPopoverReopenKey = 0,
}: {
  values: typeof emptyNewMember;
  onChange: (v: typeof emptyNewMember) => void;
  existingMemberIds?: string[];
  groups?: MemberGroup[];
  userGroups?: MMUserGroup[];
  onOpenCreateGroupDialog?: () => void;
  groupPopoverReopenKey?: number;
}) {
  const [clawStr, setClawStr] = React.useState<string>(String(values.clawLimit));
  const [idError, setIdError] = React.useState<string>("");
  const [groupSearchStr, setGroupSearchStr] = React.useState("");
  const [groupPopoverOpen, setGroupPopoverOpen] = React.useState(false);
  const groupReopenMounted = React.useRef(false);
  const groupListRef = React.useRef<HTMLDivElement>(null);

  // 当 groupPopoverReopenKey 变化时（非首次 mount），重新打开 Popover 并滚到底部
  React.useEffect(() => {
    if (!groupReopenMounted.current) { groupReopenMounted.current = true; return; }
    if (groupPopoverReopenKey > 0) {
      setGroupPopoverOpen(true);
      setTimeout(() => {
        if (groupListRef.current) groupListRef.current.scrollTop = groupListRef.current.scrollHeight;
      }, 100);
    }
  }, [groupPopoverReopenKey]);

  React.useEffect(() => {
    setClawStr(String(values.clawLimit));
  }, [values.clawLimit]);

  const handleIdBlur = () => {
    if (values.id.trim() && existingMemberIds.includes(values.id.trim())) {
      setIdError("成员ID已存在，请使用其他ID");
    } else {
      setIdError("");
    }
  };

  // 使用 userGroups 渲染（有层级和 source），fallback 到 groups
  const hasUserGroups = userGroups.length > 0;
  const ugMap = React.useMemo(() => new Map(userGroups.map((g) => [g.id, g])), [userGroups]);
  const getUgPath = (gId: string): string => {
    const chain: string[] = [];
    let node = ugMap.get(gId);
    while (node) { chain.unshift(node.name); node = node.parentId ? ugMap.get(node.parentId) : undefined; }
    return chain.join(" / ");
  };
  // OneID 模式：组织架构 + 用户组；普通模式：全部 manual 组织
  const deptGroups = React.useMemo(() => userGroups.filter((g) => g.source === "oneid-dept"), [userGroups]);
  const ogGroups = React.useMemo(() => userGroups.filter((g) => g.source === "oneid-group"), [userGroups]);
  const manualUGroups = React.useMemo(() => userGroups.filter((g) => g.source === "manual"), [userGroups]);
  // 构建树
  const buildTree = (list: typeof userGroups) => {
    const map = new Map(list.map((g) => [g.id, { ...g, children: [] as typeof list }]));
    const roots: Array<typeof list[0] & { children: typeof list }> = [];
    map.forEach((node) => {
      if (node.parentId && map.has(node.parentId)) {
        map.get(node.parentId)!.children.push(node);
      } else {
        roots.push(node);
      }
    });
    return roots;
  };
  const deptTree = React.useMemo(() => buildTree(deptGroups), [deptGroups]);
  const ogTree = React.useMemo(() => buildTree(ogGroups), [ogGroups]);
  const manualTree = React.useMemo(() => buildTree(manualUGroups), [manualUGroups]);
  // 展开状态
  const [treeExpanded, setTreeExpanded] = React.useState<Set<string>>(new Set());
  const toggleExpand = (id: string) => setTreeExpanded((prev) => {
    const next = new Set(prev);
    if (next.has(id)) next.delete(id); else next.add(id);
    return next;
  });
  // 搜索
  const matchSearch = (g: { id: string; name: string }) => {
    if (!groupSearchStr.trim()) return true;
    const q = groupSearchStr.toLowerCase();
    return g.name.toLowerCase().includes(q) || getUgPath(g.id).toLowerCase().includes(q);
  };

  const filteredGroups = groups.filter((g) =>
    g.name.toLowerCase().includes(groupSearchStr.toLowerCase())
  );

  const toggleGroup = (gId: string) => {
    const next = values.groupIds.includes(gId) ? values.groupIds.filter((x) => x !== gId) : [...values.groupIds, gId];
    onChange({ ...values, groupIds: next });
  };

  // 获取已选组织的显示名称
  const selectedNames = React.useMemo(() => {
    if (hasUserGroups) {
      return values.groupIds.map((id) => getUgPath(id)).filter(Boolean);
    }
    return groups.filter((g) => values.groupIds.includes(g.id)).map((g) => g.name);
  }, [values.groupIds, hasUserGroups, groups, userGroups]);

  // 树形节点渲染
  const renderTreeNode = (node: any, depth: number): React.ReactNode => {
    if (!matchSearch(node) && !(node.children?.length > 0 && node.children.some((c: any) => matchSearch(c)))) return null;
    const hasChildren = node.children && node.children.length > 0;
    const isExpanded = treeExpanded.has(node.id);
    const isSelected = values.groupIds.includes(node.id);
    return (
      <div key={node.id}>
        <div
          className={`w-full flex items-center gap-1.5 px-2 py-1.5 rounded text-xs transition-colors cursor-pointer ${isSelected ? "hover:bg-[#fafafa] text-[#355EF1]" : "hover:bg-[#fafafa] text-[#525252]"}`}
          style={{ paddingLeft: 8 + depth * 16 }}
          onClick={() => toggleGroup(node.id)}
        >
          {hasChildren ? (
            <span
              onClick={(e) => { e.stopPropagation(); toggleExpand(node.id); }}
              className="w-4 h-4 flex items-center justify-center text-[#A3A3A3] hover:text-[#737373] shrink-0"
            >
              {isExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
            </span>
          ) : (
            <span className="w-4 h-4 shrink-0" />
          )}
          <span className={`w-3.5 h-3.5 rounded border shrink-0 flex items-center justify-center ${isSelected ? "bg-[#355EF1] border-[#1447E6]" : "border-[#C8CFDA] bg-white"}`}>
            {isSelected && <Check className="w-2.5 h-2.5 text-white" />}
          </span>
          <span className="truncate">{node.name}</span>
        </div>
        {hasChildren && isExpanded && node.children.map((c: any) => renderTreeNode(c, depth + 1))}
      </div>
    );
  };

  return (
    <div className="py-2 space-y-4">
      <div>
        <p className="text-sm font-medium text-[#0A0A0A] mb-3">用户信息</p>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label className="flex items-center gap-1.5">
              用户 ID <span className="text-[#d42a1e]">*</span>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="cursor-pointer inline-flex">
                    <Info className="w-3.5 h-3.5 text-[#A3A3A3]" />
                  </span>
                </TooltipTrigger>
                <TooltipContent sideOffset={4}>填写企业用户的唯一 ID，例如企业邮箱或企业用户唯一名称，作为企业用户登录用户端的账号</TooltipContent>
              </Tooltip>
            </Label>
            <Input
              placeholder="例如：alice@acompany.com"
              value={values.id}
              onChange={(e) => {
                onChange({ ...values, id: e.target.value });
                setIdError("");
              }}
              onBlur={handleIdBlur}
              className={idError ? "border-red-300 focus:ring-red-500 focus:border-red-500" : ""}
            />
            {idError && <p className="text-xs text-[#d42a1e] font-medium">{idError}</p>}
          </div>

          <div className="space-y-2">
            <Label>用户角色 <span className="text-[#d42a1e]">*</span></Label>
            <Select value={values.role} onValueChange={(v) => onChange({ ...values, role: v })}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="member">用户</SelectItem>
                <SelectItem value="admin">管理员</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* 用户组织 */}
          <div className="space-y-2">
            <Label>用户组织</Label>
            <GroupSelect
              groups={userGroups as any}
              selectedIds={values.groupIds}
              onChange={(ids) => onChange({ ...values, groupIds: ids })}
              sourceFilter={hasUserGroups ? ["oneid-dept", "manual"] : ["manual"]}
              placeholder="请选择组织"
              enableAggregation={false}
            />
          </div>

          <div className="space-y-2">
            <Label className="flex items-center gap-1.5">
              信息发送
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="cursor-pointer inline-flex">
                    <Info className="w-3.5 h-3.5 text-[#A3A3A3]" />
                  </span>
                </TooltipTrigger>
                <TooltipContent sideOffset={4}>信息发送会产生额外的短信/邮件费用，合并到腾讯云账单计费</TooltipContent>
              </Tooltip>
            </Label>
            <Input type="email" placeholder="输入用户接收账号密码的邮箱地址" value={values.notificationEmail} onChange={(e) => onChange({ ...values, notificationEmail: e.target.value })} />
          </div>
        </div>
      </div>

      <div>
        <p className="text-sm font-medium text-[#0A0A0A] mb-3">用户配额</p>
        {values.groupIds.length > 0 ? (
          <div className="space-y-3">
            <SurfaceInner className="overflow-hidden">
              <Table density="compact">
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead>组织</TableHead>
                    <TableHead>Agent 上限</TableHead>
                    <TableHead>每日 Tokens 上限</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {values.groupIds.map((gId) => {
                    const quotaMap = hasUserGroups && userGroups.some((g) => g.source === 'oneid-dept' || g.source === 'oneid-group') ? ONEID_GROUP_POLICY_QUOTAS : GROUP_POLICY_QUOTAS;
                    const ugName = hasUserGroups ? getUgPath(gId) : gId;
                    const quota = quotaMap[gId] ?? { clawLimit: PRESET_POLICY_CLAW_LIMIT, tokenLimit: PRESET_POLICY_TOKEN_LIMIT };
                    return (
                      <TableRow key={gId} className="hover:bg-transparent">
                        <TableCell>{ugName}</TableCell>
                        <TableCell className="tabular-nums">{quota.clawLimit}</TableCell>
                        <TableCell className="tabular-nums">{quota.tokenLimit.toLocaleString()}</TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </SurfaceInner>
            <p className="text-xs text-[#A3A3A3] leading-relaxed">
              该用户已加入组织，配额由平台策略统一管理。如需修改请前往<a href="/admin/platform-policy" className="text-[#355EF1] hover:underline">平台策略</a>页进行配置。
            </p>
          </div>
        ) : (
        <div className="space-y-4">
          <div className="space-y-2">
            <Label className="flex items-center gap-1.5">
              Agent 数量上限 <span className="text-[#d42a1e]">*</span>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="cursor-pointer inline-flex">
                    <Info className="w-3.5 h-3.5 text-[#A3A3A3]" />
                  </span>
                </TooltipTrigger>
                <TooltipContent sideOffset={4}>单个企业用户最多可以创建的 Agent 数量</TooltipContent>
              </Tooltip>
            </Label>
            <div className="rounded-[4px] bg-[#FAFBFD] p-4">
              <Input
                type="number"
                value={clawStr}
                onChange={(e) => {
                  setClawStr(e.target.value);
                  if (e.target.value !== "") onChange({ ...values, clawLimit: Number(e.target.value) });
                }}
                onBlur={() => {
                  if (clawStr === "" || isNaN(Number(clawStr))) {
                    setClawStr("0");
                    onChange({ ...values, clawLimit: 0 });
                  }
                }}
                className="bg-white"
                placeholder="请输入数量"
              />
            </div>
          </div>
          <div className="space-y-2">
            <Label className="flex items-center gap-1.5">
              Tokens 上限
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="cursor-pointer inline-flex">
                    <Info className="w-3.5 h-3.5 text-[#A3A3A3]" />
                  </span>
                </TooltipTrigger>
                <TooltipContent sideOffset={4}>单个企业用户在所配置周期内最多可消耗的 Tokens 数量</TooltipContent>
              </Tooltip>
            </Label>
            <div className="rounded-[4px] bg-[#FAFBFD] p-4">
              <AddTokenQuotaEditor
                value={values.tokenQuota}
                onChange={(tokenQuota) => onChange({
                  ...values,
                  tokenQuota,
                  tokenLimit: resolveTokenLimit(tokenQuota.limit),
                })}
              />
            </div>
          </div>
        </div>
        )}
      </div>
    </div>
  );
}

// ─── 编辑用户表单（无密码、无信息发送，成员ID只读） ──────────────────────────
function EditMemberFormFields({
  values,
  onChange,
  isInitialAdmin = false,
  groups = [],
  userGroups = [],
  onOpenCreateGroupDialog,
  groupPopoverReopenKey = 0,
}: {
  values: typeof emptyEditForm;
  onChange: (v: typeof emptyEditForm) => void;
  isInitialAdmin?: boolean;
  groups?: MemberGroup[];
  userGroups?: MMUserGroup[];
  onOpenCreateGroupDialog?: () => void;
  groupPopoverReopenKey?: number;
}) {
  const [clawStr, setClawStr] = React.useState<string>(String(values.clawLimit));
  const [groupSearchStr, setGroupSearchStr] = React.useState("");
  const [groupPopoverOpen, setGroupPopoverOpen] = React.useState(false);
  const groupReopenMounted = React.useRef(false);
  const groupListRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    setClawStr(String(values.clawLimit));
  }, [values.clawLimit]);

  React.useEffect(() => {
    if (!groupReopenMounted.current) { groupReopenMounted.current = true; return; }
    if (groupPopoverReopenKey > 0) {
      setGroupPopoverOpen(true);
      setTimeout(() => {
        if (groupListRef.current) groupListRef.current.scrollTop = groupListRef.current.scrollHeight;
      }, 100);
    }
  }, [groupPopoverReopenKey]);


  // 使用 userGroups 渲染（有层级和 source），fallback 到 groups
  const hasUserGroups = userGroups.length > 0;
  const ugMap = React.useMemo(() => new Map(userGroups.map((g) => [g.id, g])), [userGroups]);
  const getUgPath = (gId: string): string => {
    const chain: string[] = [];
    let node = ugMap.get(gId);
    while (node) { chain.unshift(node.name); node = node.parentId ? ugMap.get(node.parentId) : undefined; }
    return chain.join(" / ");
  };
  const deptGroups = React.useMemo(() => userGroups.filter((g) => g.source === "oneid-dept"), [userGroups]);
  const ogGroups = React.useMemo(() => userGroups.filter((g) => g.source === "oneid-group"), [userGroups]);
  const manualUGroups = React.useMemo(() => userGroups.filter((g) => g.source === "manual"), [userGroups]);
  const buildTree = (list: typeof userGroups) => {
    const map = new Map(list.map((g) => [g.id, { ...g, children: [] as typeof list }]));
    const roots: Array<typeof list[0] & { children: typeof list }> = [];
    map.forEach((node) => {
      if (node.parentId && map.has(node.parentId)) {
        map.get(node.parentId)!.children.push(node);
      } else {
        roots.push(node);
      }
    });
    return roots;
  };
  const deptTree = React.useMemo(() => buildTree(deptGroups), [deptGroups]);
  const ogTree = React.useMemo(() => buildTree(ogGroups), [ogGroups]);
  const manualTree = React.useMemo(() => buildTree(manualUGroups), [manualUGroups]);
  const [treeExpanded, setTreeExpanded] = React.useState<Set<string>>(new Set());
  const toggleExpand = (id: string) => setTreeExpanded((prev) => {
    const next = new Set(prev);
    if (next.has(id)) next.delete(id); else next.add(id);
    return next;
  });
  const matchSearch = (g: { id: string; name: string }) => {
    if (!groupSearchStr.trim()) return true;
    const q = groupSearchStr.toLowerCase();
    return g.name.toLowerCase().includes(q) || getUgPath(g.id).toLowerCase().includes(q);
  };
  const filteredGroups = groups.filter((g) => g.name.toLowerCase().includes(groupSearchStr.toLowerCase()));
  const toggleGroup = (gId: string) => {
    const next = values.groupIds.includes(gId) ? values.groupIds.filter((x) => x !== gId) : [...values.groupIds, gId];
    onChange({ ...values, groupIds: next });
  };

  const selectedNames = React.useMemo(() => {
    if (hasUserGroups) {
      return values.groupIds.map((id) => getUgPath(id)).filter(Boolean);
    }
    return groups.filter((g) => values.groupIds.includes(g.id)).map((g) => g.name);
  }, [values.groupIds, hasUserGroups, groups, userGroups]);

  const renderTreeNode = (node: any, depth: number): React.ReactNode => {
    if (!matchSearch(node) && !(node.children?.length > 0 && node.children.some((c: any) => matchSearch(c)))) return null;
    const hasChildren = node.children && node.children.length > 0;
    const isExpanded = treeExpanded.has(node.id);
    const isSelected = values.groupIds.includes(node.id);
    return (
      <div key={node.id}>
        <div
          className={`w-full flex items-center gap-1.5 px-2 py-1.5 rounded text-xs transition-colors cursor-pointer ${isSelected ? "hover:bg-[#fafafa] text-[#355EF1]" : "hover:bg-[#fafafa] text-[#525252]"}`}
          style={{ paddingLeft: 8 + depth * 16 }}
          onClick={() => toggleGroup(node.id)}
        >
          {hasChildren ? (
            <span
              onClick={(e) => { e.stopPropagation(); toggleExpand(node.id); }}
              className="w-4 h-4 flex items-center justify-center text-[#A3A3A3] hover:text-[#737373] shrink-0"
            >
              {isExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
            </span>
          ) : (
            <span className="w-4 h-4 shrink-0" />
          )}
          <span className={`w-3.5 h-3.5 rounded border shrink-0 flex items-center justify-center ${isSelected ? "bg-[#355EF1] border-[#1447E6]" : "border-[#C8CFDA] bg-white"}`}>
            {isSelected && <Check className="w-2.5 h-2.5 text-white" />}
          </span>
          <span className="truncate">{node.name}</span>
        </div>
        {hasChildren && isExpanded && node.children.map((c: any) => renderTreeNode(c, depth + 1))}
      </div>
    );
  };
  return (
    <div className="py-2 space-y-4">
      <div>
        
        <div className="space-y-4">
          <div className="space-y-2">
            <Label className="flex items-center gap-1.5">
              用户 ID
              <InfoPopover content="用户 ID 为唯一标识，不可修改" />
            </Label>
            <Input
              value={values.id}
              disabled
              className="bg-[#f5f5f5] cursor-not-allowed opacity-60 disabled:bg-[#f5f5f5] disabled:text-[#A3A3A3] disabled:cursor-not-allowed disabled:opacity-60 disabled:pointer-events-auto"
            />
          </div>

          <div className="space-y-2">
            <Label>用户角色</Label>
            <Select value={values.role} onValueChange={(v) => !isInitialAdmin && onChange({ ...values, role: v })} disabled={isInitialAdmin}>
              <SelectTrigger className={`w-full ${isInitialAdmin ? "bg-[#f5f5f5] cursor-not-allowed opacity-60" : "bg-white"}`}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="member">用户</SelectItem>
                <SelectItem value="admin">管理员</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* 用户组织 */}
          <div className="space-y-2">
            <Label>用户组织</Label>
            <GroupSelect
              groups={userGroups as any}
              selectedIds={values.groupIds}
              onChange={(ids) => onChange({ ...values, groupIds: ids })}
              sourceFilter={hasUserGroups ? ["oneid-dept", "oneid-group", "manual"] : ["manual"]}
              placeholder="搜索组织"
            />
          </div>
        </div>
      </div>

      {/* 第二大块：用户配额 */}
      <div>
        <p className="text-sm font-medium text-[#0A0A0A] mb-3">用户配额</p>
        {values.groupIds.length > 0 ? (
          <div className="space-y-3">
            <SurfaceInner className="overflow-hidden">
              <Table density="compact">
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead>组织</TableHead>
                    <TableHead>Agent 上限</TableHead>
                    <TableHead>每日 Tokens 上限</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {values.groupIds.map((gId) => {
                    const quotaMap = hasUserGroups && userGroups.some((g) => g.source === 'oneid-dept' || g.source === 'oneid-group') ? ONEID_GROUP_POLICY_QUOTAS : GROUP_POLICY_QUOTAS;
                    const ugName = hasUserGroups ? getUgPath(gId) : gId;
                    const quota = quotaMap[gId] ?? { clawLimit: PRESET_POLICY_CLAW_LIMIT, tokenLimit: PRESET_POLICY_TOKEN_LIMIT };
                    return (
                      <TableRow key={gId} className="hover:bg-transparent">
                        <TableCell>{ugName}</TableCell>
                        <TableCell className="tabular-nums">{quota.clawLimit}</TableCell>
                        <TableCell className="tabular-nums">{quota.tokenLimit.toLocaleString()}</TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </SurfaceInner>
            <p className="text-xs text-[#A3A3A3] leading-relaxed">
              该用户已加入组织，配额由平台策略统一管理。如需修改请前往<a href="/admin/platform-policy" className="text-[#355EF1] hover:underline">平台策略</a>页进行配置。
            </p>
          </div>
        ) : (
        <div className="space-y-4">
          <div className="space-y-2">
            <Label className="flex items-center gap-1.5">
              Agent 数量上限
              <InfoPopover content="单个企业用户最多可以创建的 Agent 数量" />
            </Label>
            <Input
              type="number"
              value={clawStr}
              onChange={(e) => {
                setClawStr(e.target.value);
                if (e.target.value !== "") onChange({ ...values, clawLimit: Number(e.target.value) });
              }}
              onBlur={() => {
                if (clawStr === "" || isNaN(Number(clawStr))) {
                  setClawStr("0");
                  onChange({ ...values, clawLimit: 0 });
                }
              }}
              className="bg-white"
              placeholder="请输入数量"
            />
          </div>
          <div className="space-y-2">
            <Label className="flex items-center gap-1.5">
              每日 Tokens 数量上限
              <InfoPopover content="单个企业用户每日最多可消耗的 Tokens 数量" />
            </Label>
            <TokenLimitInput
              value={values.tokenLimit}
              onChange={(v) => onChange({ ...values, tokenLimit: v })}
            />
          </div>
        </div>
        )}
      </div>
    </div>
  );
}

// ─── OneID 编辑用户表单（用户 ID / 角色 / 部门 只读，仅配额可编辑） ──────────
const emptyOneidEditForm = {
  id: "", role: "member", department: "", clawLimit: LAST_CLAW_LIMIT, tokenLimit: LAST_TOKEN_LIMIT, groupIds: [] as string[],
};

function OneidEditMemberFormFields({
  values,
  onChange,
  isUnified = false,
  isInitialAdmin = false,
  groups = [],
  userGroups = [],
  onOpenCreateGroupDialog,
  groupPopoverReopenKey = 0,
}: {
  values: typeof emptyOneidEditForm;
  onChange: (v: typeof emptyOneidEditForm) => void;
  /** unified 模式：允许编辑用户角色（与普通模式对齐） */
  isUnified?: boolean;
  /** 初始管理员：角色不可改，避免误降级 */
  isInitialAdmin?: boolean;
  groups?: MemberGroup[];
  userGroups?: MMUserGroup[];
  onOpenCreateGroupDialog?: () => void;
  groupPopoverReopenKey?: number;
}) {
  const [clawStr, setClawStr] = React.useState<string>(String(values.clawLimit));
  const [groupSearchStr, setGroupSearchStr] = React.useState("");
  const [groupPopoverOpen, setGroupPopoverOpen] = React.useState(false);
  const groupReopenMounted = React.useRef(false);
  const groupListRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    setClawStr(String(values.clawLimit));
  }, [values.clawLimit]);

  React.useEffect(() => {
    if (!groupReopenMounted.current) { groupReopenMounted.current = true; return; }
    if (groupPopoverReopenKey > 0) {
      setGroupPopoverOpen(true);
      setTimeout(() => {
        if (groupListRef.current) groupListRef.current.scrollTop = groupListRef.current.scrollHeight;
      }, 100);
    }
  }, [groupPopoverReopenKey]);

  // userGroups 树形逻辑
  const hasUserGroups = userGroups.length > 0;
  const ugMap = React.useMemo(() => new Map(userGroups.map((g) => [g.id, g])), [userGroups]);
  const getUgPath = (gId: string): string => {
    const chain: string[] = [];
    let node = ugMap.get(gId);
    while (node) { chain.unshift(node.name); node = node.parentId ? ugMap.get(node.parentId) : undefined; }
    return chain.join(" / ");
  };
  const deptGroups = React.useMemo(() => userGroups.filter((g) => g.source === "oneid-dept"), [userGroups]);
  const ogGroups = React.useMemo(() => userGroups.filter((g) => g.source === "oneid-group"), [userGroups]);
  const buildTree = (list: typeof userGroups) => {
    const map = new Map(list.map((g) => [g.id, { ...g, children: [] as typeof list }]));
    const roots: Array<typeof list[0] & { children: typeof list }> = [];
    map.forEach((node) => {
      if (node.parentId && map.has(node.parentId)) {
        map.get(node.parentId)!.children.push(node);
      } else {
        roots.push(node);
      }
    });
    return roots;
  };
  const deptTree = React.useMemo(() => buildTree(deptGroups), [deptGroups]);
  const ogTree = React.useMemo(() => buildTree(ogGroups), [ogGroups]);
  const [treeExpanded, setTreeExpanded] = React.useState<Set<string>>(new Set());
  const toggleExpand = (id: string) => setTreeExpanded((prev) => {
    const next = new Set(prev);
    if (next.has(id)) next.delete(id); else next.add(id);
    return next;
  });
  const matchSearch = (g: { id: string; name: string }) => {
    if (!groupSearchStr.trim()) return true;
    const q = groupSearchStr.toLowerCase();
    return g.name.toLowerCase().includes(q) || getUgPath(g.id).toLowerCase().includes(q);
  };

  // dept 组织 id 集合（不可编辑）
  const deptGroupIds = React.useMemo(() => new Set(deptGroups.map((g) => g.id)), [deptGroups]);
  const filteredGroups = groups.filter((g) => g.name.toLowerCase().includes(groupSearchStr.toLowerCase()));
  const toggleGroup = (gId: string) => {
    if (deptGroupIds.has(gId)) return; // dept 组织不可操作
    const next = values.groupIds.includes(gId) ? values.groupIds.filter((x) => x !== gId) : [...values.groupIds, gId];
    onChange({ ...values, groupIds: next });
  };

  const selectedNames = React.useMemo(() => {
    if (hasUserGroups) {
      return values.groupIds.map((id) => getUgPath(id)).filter(Boolean);
    }
    return groups.filter((g) => values.groupIds.includes(g.id)).map((g) => g.name);
  }, [values.groupIds, hasUserGroups, groups, userGroups]);

  const renderTreeNode = (node: any, depth: number): React.ReactNode => {
    if (!matchSearch(node) && !(node.children?.length > 0 && node.children.some((c: any) => matchSearch(c)))) return null;
    const hasChildren = node.children && node.children.length > 0;
    const isExpanded = treeExpanded.has(node.id);
    const isSelected = values.groupIds.includes(node.id);
    const isDept = deptGroupIds.has(node.id);
    const row = (
      <div key={node.id}>
        <div
          className={`w-full flex items-center gap-1.5 px-2 py-1.5 rounded text-xs transition-colors ${isDept ? "cursor-not-allowed opacity-50" : "cursor-pointer"} ${isSelected ? "hover:bg-[#fafafa] text-[#355EF1]" : "hover:bg-[#fafafa] text-[#525252]"}`}
          style={{ paddingLeft: 8 + depth * 16 }}
          onClick={() => !isDept && toggleGroup(node.id)}
        >
          {hasChildren ? (
            <span
              onClick={(e) => { e.stopPropagation(); toggleExpand(node.id); }}
              className="w-4 h-4 flex items-center justify-center text-[#A3A3A3] hover:text-[#737373] shrink-0"
            >
              {isExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
            </span>
          ) : (
            <span className="w-4 h-4 shrink-0" />
          )}
          <span className={`w-3.5 h-3.5 rounded border shrink-0 flex items-center justify-center ${isSelected ? "bg-[#355EF1] border-[#1447E6]" : "border-[#C8CFDA] bg-white"}`}>
            {isSelected && <Check className="w-2.5 h-2.5 text-white" />}
          </span>
          <span className="truncate">{node.name}</span>
        </div>
        {hasChildren && isExpanded && node.children.map((c: any) => renderTreeNode(c, depth + 1))}
      </div>
    );
    if (isDept) {
      return (
        <Tooltip>
          <TooltipTrigger asChild>{row}</TooltipTrigger>
          <TooltipContent side="right" className="text-xs max-w-[220px]">同步部门的组织不可编辑，如需编辑请前往腾讯统一身份管理平台</TooltipContent>
        </Tooltip>
      );
    }
    return row;
  };

  return (
    <div className="py-2 space-y-4">
      <div>
        
        <div className="space-y-4">
          <div className="space-y-2">
            <Label className="flex items-center gap-1.5">
              用户 ID
              <InfoPopover content="用户 ID 由统一身份平台管理" />
            </Label>
            <Input
              value={values.id}
              disabled
              className="bg-[#f5f5f5] cursor-not-allowed opacity-60 disabled:bg-[#f5f5f5] disabled:text-[#A3A3A3] disabled:cursor-not-allowed disabled:opacity-60 disabled:pointer-events-auto"
            />
          </div>
          <div className="space-y-2">
            <Label>用户角色</Label>
            <Select
              value={values.role}
              onValueChange={(v) => isUnified && !isInitialAdmin && onChange({ ...values, role: v })}
              disabled={!isUnified || isInitialAdmin}
            >
              <SelectTrigger className={`w-full ${(!isUnified || isInitialAdmin) ? "bg-[#f5f5f5] cursor-not-allowed opacity-60" : "bg-white"}`}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="member">用户</SelectItem>
                <SelectItem value="admin">管理员</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>部门</Label>
            <Input
              value={values.department || "—"}
              disabled
              className="bg-[#f5f5f5] cursor-not-allowed opacity-60 disabled:bg-[#f5f5f5] disabled:text-[#A3A3A3] disabled:cursor-not-allowed disabled:opacity-60 disabled:pointer-events-auto"
            />
          </div>

          {/* 用户组织 */}
          <div className="space-y-2">
            <Label>用户组织</Label>
            <GroupSelect
              groups={userGroups as any}
              selectedIds={values.groupIds}
              onChange={(ids) => onChange({ ...values, groupIds: ids })}
              disabledIds={values.groupIds.filter((gid) => deptGroupIds.has(gid))}
              disabledTooltip="部门组织不可手动编辑"
              sourceFilter={["oneid-dept", "oneid-group", "manual"]}
              placeholder="搜索组织"
            />
          </div>
        </div>
      </div>

      {/* 用户配额（可编辑） */}
      <div>
        <p className="text-sm font-medium text-[#0A0A0A] mb-3">用户配额</p>
        {values.groupIds.length > 0 ? (
          <div className="space-y-3">
            <SurfaceInner className="overflow-hidden">
              <Table density="compact">
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead>组织</TableHead>
                    <TableHead>Agent 上限</TableHead>
                    <TableHead>每日 Tokens 上限</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {values.groupIds.map((gId) => {
                    const quotaMap = ONEID_GROUP_POLICY_QUOTAS;
                    const ugName = hasUserGroups ? getUgPath(gId) : gId;
                    const quota = quotaMap[gId] ?? { clawLimit: PRESET_POLICY_CLAW_LIMIT, tokenLimit: PRESET_POLICY_TOKEN_LIMIT };
                    return (
                      <TableRow key={gId} className="hover:bg-transparent">
                        <TableCell>{ugName}</TableCell>
                        <TableCell className="tabular-nums">{quota.clawLimit}</TableCell>
                        <TableCell className="tabular-nums">{quota.tokenLimit.toLocaleString()}</TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </SurfaceInner>
            <p className="text-xs text-[#A3A3A3] leading-relaxed">
              该用户已加入组织，配额由平台策略统一管理。如需修改请前往<a href="/admin/platform-policy" className="text-[#355EF1] hover:underline">平台策略</a>页进行配置。
            </p>
          </div>
        ) : (
        <div className="space-y-4">
          <div className="space-y-2">
            <Label className="flex items-center gap-1.5">
              Agent 数量上限
              <InfoPopover content="单个企业用户最多可以创建的 Agent 数量" />
            </Label>
            <Input
              type="number"
              value={clawStr}
              onChange={(e) => {
                setClawStr(e.target.value);
                if (e.target.value !== "") onChange({ ...values, clawLimit: Number(e.target.value) });
              }}
              onBlur={() => {
                if (clawStr === "" || isNaN(Number(clawStr))) {
                  setClawStr("0");
                  onChange({ ...values, clawLimit: 0 });
                }
              }}
              className="bg-white"
              placeholder="请输入数量"
            />
          </div>
          <div className="space-y-2">
            <Label className="flex items-center gap-1.5">
              每日 Tokens 数量上限
              <InfoPopover content="单个企业用户每日最多可消耗的 Tokens 数量" />
            </Label>
            <TokenLimitInput
              value={values.tokenLimit}
              onChange={(v) => onChange({ ...values, tokenLimit: v })}
            />
          </div>
        </div>
        )}
      </div>
    </div>
  );
}

// ─── 创建/重置成功弹窗 ────────────────────────────────────────────────────────
function CredentialResultDialog({
  open,
  onClose,
  title,
  memberId,
  password,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  memberId: string;
  password: string;
}) {
  const [copied, setCopied] = useState(false);
  // 每次弹窗打开时重置复制状态
  useEffect(() => {
    if (open) setCopied(false);
  }, [open]);
  // 全加密：用 • 替换所有字符
  const maskedPassword = "•".repeat(password.length);

  const handleCopy = () => {
    const text = `账号：${memberId}\n密码：${password}`;
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
    });
  };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <DialogContent className="sm:max-w-[560px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <CheckCircle className="w-5 h-5 text-green-500" />
            {title}
          </DialogTitle>
        </DialogHeader>
        <div className="pt-1 pb-3 space-y-3">
          {/* 账号密码展示 */}
          <div className="bg-[#fafafa] rounded-[4px] border border-[#e5e5e5] p-4 space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-[#A3A3A3] font-medium uppercase tracking-wide">用户 ID</span>
              <span className="text-sm font-mono text-[#0A0A0A] select-all">{memberId}</span>
            </div>
            <div className="border-t border-[#e5e5e5]" />
            <div className="flex items-center justify-between">
              <span className="text-xs text-[#A3A3A3] font-medium uppercase tracking-wide">初始密码</span>
              <span className="text-sm font-mono text-[#0A0A0A] tracking-widest select-none">{maskedPassword}</span>
            </div>
          </div>

          {/* 警示文案 */}
          <div className="flex items-start gap-2 bg-amber-50 border border-amber-100 rounded-[4px] px-3 py-2.5">
            <CircleAlert className="w-4 h-4 text-amber-500 flex-shrink-0 mt-0.5" />
            <p className="text-xs text-amber-700 leading-relaxed">
              关闭弹窗后将无法再次查看此密码，请复制后妥善保存，并通过安全渠道告知用户。
            </p>
          </div>

          {/* 复制按钮 */}
          <Button
            className="w-full"
           
            onClick={handleCopy}
          >
            {copied ? (
              <><CheckCircle className="w-4 h-4 mr-2" />已复制</>
            ) : (
              <><Copy className="w-4 h-4 mr-2" />复制账号密码</>
            )}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ─── Main Component ───────────────────────────────────────────────────────────
/** 获取用户所在组织的配额信息（用于列表和弹窗展示） */
function getMemberGroupQuotas(memberId: string, hasOneid: boolean): Array<{ groupId: string; groupName: string; clawLimit: number; tokenLimit: number }> {
  // 从实际 mock 用户数据中获取用户的 groupIds
  const userOrg = hasOneid
    ? MM_MOCK_USERS.find((u) => u.userId === memberId)
    : MM_MOCK_USERS_MANUAL.find((u) => u.userId === memberId);
  if (!userOrg || userOrg.groupIds.length === 0) return [];

  const quotaMap = hasOneid ? ONEID_GROUP_POLICY_QUOTAS : GROUP_POLICY_QUOTAS;
  const allGroups = hasOneid ? MM_MOCK_GROUPS : MM_MOCK_MANUAL_GROUPS;
  const groupMap = new Map(allGroups.map((g) => [g.id, g]));

  return userOrg.groupIds
    .filter((gId) => groupMap.has(gId))
    .map((gId) => {
      const quota = quotaMap[gId] ?? { clawLimit: PRESET_POLICY_CLAW_LIMIT, tokenLimit: PRESET_POLICY_TOKEN_LIMIT };
      const fullPath = mmGetPrimaryDeptPath(gId, allGroups);
      return { groupId: gId, groupName: fullPath, clawLimit: quota.clawLimit, tokenLimit: quota.tokenLimit };
    });
}

export default function MemberManagement() {
  // 获取 hasOneid / isUnified 状态
  // - hasOneid：standard 或 unified（用于继承 OneID 视图基础：部门列、按部门搜索、手动同步、前往腾讯统一身份等）
  // - isUnified：仅 unified（"统一"模式）；用于在 OneID 基础上叠加"普通模式独有功能"
  // - showCustomExtras：unified 或 custom 都为 true，表示需要展示普通模式独有的功能
  //   （顶栏「下载用户」「添加用户」、操作列完整三按钮、表格列宽走自适应布局等）
  const { hasOneid, isUnified } = useAdminMode();
  const showCustomExtras = isUnified || !hasOneid;

  const [members, setMembers] = useState<typeof MOCK_MEMBERS_BASE>(
    hasOneid ? MOCK_MEMBERS_ONEID_BASE : (MOCK_MEMBERS_MANUAL_BASE as typeof MOCK_MEMBERS_BASE)
  );
  // 监听 hasOneid 切换，members 重置为对应模式的基础数据
  useEffect(() => {
    setMembers(
      hasOneid ? MOCK_MEMBERS_ONEID_BASE : (MOCK_MEMBERS_MANUAL_BASE as typeof MOCK_MEMBERS_BASE)
    );
  }, [hasOneid]);
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [isInitialAdminEdit, setIsInitialAdminEdit] = useState(false);

  // OneID 模式专用状态
  const [deptFilter, setDeptFilter] = useState("");
  const [roleFilter, setRoleFilter] = useState<"all" | "admin" | "member">("all");
  // 「全部用户」表格 — 列头筛选（部门/组织）
  const [groupFilter, setGroupFilter] = useState("");
  const [deptColFilterOpen, setDeptColFilterOpen] = useState(false);
  const [groupColFilterOpen, setGroupColFilterOpen] = useState(false);
  /**
   * OneID 是否同步出了部门数据：用于决定「部门」列、「组织 sheet 中的组织架构段/部门信息」是否展示。
   * 口径：本地 MOCK_DEPARTMENTS 树是否非空（即 OneID 是否同步出了部门元数据）。
   * 后续接入真实接口后，把它替换为「OneID 部门数据是否非空」即可。
   */
  const hasDeptData = MOCK_DEPARTMENTS.length > 0;
  const [oneidEditForm, setOneidEditForm] = useState({ ...emptyOneidEditForm });
  const [isSyncing, setIsSyncing] = useState(false);
  /** 组织架构是否已同步为组织（由 GroupView 回调通知） */
  const [mmDeptSynced, setMmDeptSynced] = useState(false);
  /** 手动同步时产生的异常组织数据，传递给 GroupView 显示红点 */
  const [mmAnomalousGroups, setMmAnomalousGroups] = useState<{ groupId: string; groupName: string; memberCount: number; boundConfigs: string[]; agentInstanceCount: number }[]>([]);

  // OneID 同步结果弹窗：展示因名下有未清理 Agent 而无法删除的用户 + 组织异常
  const [syncResultDialog, setSyncResultDialog] = useState<{
    open: boolean;
    failedUsers: { id: string; clawCount: number; vpcName?: string }[];
    deletedCount: number;
    addedCount: number;
    /** 组织异常：组织架构被删除但仍有配置绑定的组织 */
    anomalousGroups?: { groupId: string; groupName: string; memberCount: number; boundConfigs: string[]; agentInstanceCount: number }[];
  } | null>(null);

  // 排序：管理员置顶（按加入时间升序），普通用户按加入时间降序
  const sortedMembers = [...members].sort((a, b) => {
    if (a.role === "admin" && b.role !== "admin") return -1;
    if (a.role !== "admin" && b.role === "admin") return 1;
    if (a.role === "admin" && b.role === "admin") {
      return new Date(a.joinTime).getTime() - new Date(b.joinTime).getTime();
    }
    return new Date(b.joinTime).getTime() - new Date(a.joinTime).getTime();
  });

  // 管理员账号：所有 role === "admin" 的成员均不允许重置密码 / 禁用 / 删除
  const initialAdminIds = useMemo(
    () => new Set(sortedMembers.filter((m) => m.role === "admin").map((m) => m.id)),
    [sortedMembers]
  );

  const [showAddDialog, setShowAddDialog] = useState(false);
  const [showBatchDialog, setShowBatchDialog] = useState(false);
  const [showAuthSourceDialog, setShowAuthSourceDialog] = useState(false);
  // 数据源弹窗的初始步骤和初始数据源ID（编辑/更换时使用）
  const [authSourceInitialStep, setAuthSourceInitialStep] = useState<1 | 2 | undefined>(undefined);
  const [authSourceInitialId, setAuthSourceInitialId] = useState<string | null>(null);
  const [authSourceInitialFormValues, setAuthSourceInitialFormValues] = useState<Record<string, string> | null>(null);
  // 已配置的数据源列表
  const [configuredAuthSources, setConfiguredAuthSources] = useState<ConfiguredAuthSource[]>([]);
  // 数据源删除二次确认弹窗
  const [deleteAuthSourceConfirm, setDeleteAuthSourceConfirm] = useState<{ open: boolean; source: ConfiguredAuthSource } | null>(null);
  // 批量导入弹窗状态
  const [batchImportStep, setBatchImportStep] = useState<"upload" | "importing" | "done">("upload");
  const [batchImportFile, setBatchImportFile] = useState<File | null>(null);
  const [batchImportProgress, setBatchImportProgress] = useState(0); // 0~100
  const [batchImportResult, setBatchImportResult] = useState<{ success: number; fail: number } | null>(null);
  const [showResetDialog, setShowResetDialog] = useState<string | null>(null);
  const [editMemberId, setEditMemberId] = useState<string | null>(null);

  const [newMember, setNewMember] = useState(makeEmptyNewMember());
  const [editForm, setEditForm] = useState({ ...emptyEditForm });
  const [resetForm, setResetForm] = useState({ ...emptyResetForm });

  useEffect(() => {
    if (showAddDialog) setNewMember(makeEmptyNewMember());
  }, [showAddDialog]);
  // ─── 重置密码（unified 模式手动输入）：行内错误 + 明文显示 ────────────────
  const [resetNewPwdError, setResetNewPwdError] = useState<string | null>(null);
  const [resetConfirmPwdError, setResetConfirmPwdError] = useState<string | null>(null);
  const [showResetNewPwd, setShowResetNewPwd] = useState(false);
  const [showResetConfirmPwd, setShowResetConfirmPwd] = useState(false);

  // 创建/重置成功弹窗
  const [credentialDialog, setCredentialDialog] = useState<{
    open: boolean;
    title: string;
    memberId: string;
    password: string;
  }>({ open: false, title: "", memberId: "", password: "" });

  // 删除检查弹窗
  const [deleteCheckDialog, setDeleteCheckDialog] = useState<{
    open: boolean;
    memberId: string;
    clawCount: number;
    vpcType: "auto" | "custom";
    vpcName: string | null;
    hasVpcResources: boolean | null;
    clawRefreshing: boolean;
    vpcRefreshing: boolean;
  } | null>(null);
  // 二次确认弹窗
  const [deleteConfirmDialog, setDeleteConfirmDialog] = useState<{ open: boolean; memberId: string; vpcType: "auto" | "custom"; vpcName: string | null } | null>(null);
  // 禁用确认弹窗（新：所有用户均可禁用，只需二次确认）
  const [disableConfirmDialog, setDisableConfirmDialog] = useState<{ open: boolean; memberId: string; clawCount: number } | null>(null);
  // 启用确认弹窗
  const [enableConfirmDialog, setEnableConfirmDialog] = useState<{ open: boolean; memberId: string; clawCount: number } | null>(null);
  // API Token 禁用/启用弹窗
  const [apiTokenDisableDialog, setApiTokenDisableDialog] = useState<{ open: boolean; memberId: string } | null>(null);
  const [apiTokenEnableDialog, setApiTokenEnableDialog] = useState<{ open: boolean; memberId: string } | null>(null);
  // 追踪哪些用户的 API Token 已被禁用（互斥展示禁用/启用菜单项）
  const [apiTokenDisabledIds, setApiTokenDisabledIds] = useState<Set<string>>(new Set());

  // ─── 组织相关状态 ─────────────────────────────────────────────────────────
  const [viewMode, setViewMode] = useState<"all" | "group" | "project">("all");

  // 新：部门视图的裁决 state（mock 级持久化）
  const [mmOverrides, setMmOverrides] = useState<Record<string, MMUserOverrideInfo>>(
    () => ({ ...MM_MOCK_OVERRIDES })
  );
  const handleMmResolveConflict = useCallback(
    (userId: string, winnerResourceId: string) => {
      setMmOverrides((prev) => {
        const cur = prev[userId];
        if (!cur) return prev;
        return {
          ...prev,
          [userId]: {
            ...cur,
            winnerResourceId,
            isResolved: true,
          },
        };
      });
    },
    []
  );
  const [groups, setGroups] = useState<MemberGroup[]>(MOCK_GROUPS_INIT);
  const [selectedGroupId, setSelectedGroupId] = useState<string>(MOCK_GROUPS_INIT.length > 0 ? MOCK_GROUPS_INIT[0].id : "__ungrouped__");
  const [showCreateGroupDialog, setShowCreateGroupDialog] = useState(false);
  const [newGroupName, setNewGroupName] = useState("");
  const [newGroupParentId, setNewGroupParentId] = useState<string | null>(null);
  const [editingGroupId, setEditingGroupId] = useState<string | null>(null);
  const [editingGroupName, setEditingGroupName] = useState("");
  const [deleteGroupDialog, setDeleteGroupDialog] = useState<{ open: boolean; groupId: string; groupName: string; memberCount: number; configRefreshing: boolean } | null>(null);
  const [showAddToGroupDialog, setShowAddToGroupDialog] = useState(false);
  const [addToGroupSearch, setAddToGroupSearch] = useState("");
  const [addToGroupSelected, setAddToGroupSelected] = useState<string[]>([]);
  const [addToGroupDeptFilter, setAddToGroupDeptFilter] = useState("");
  const [groupPage, setGroupPage] = useState(1);
  const [groupListSearch, setGroupListSearch] = useState("");
  const [groupPopoverReopenKey, setGroupPopoverReopenKey] = useState(0);
  const [removeFromGroupDialog, setRemoveFromGroupDialog] = useState<{ open: boolean; groupId: string; groupName: string; memberId: string } | null>(null);
  const [configSectionExpanded, setConfigSectionExpanded] = useState(false);

  // 订阅跨页共享的 userStore / groupStore：项目成员归属或项目树变更时，
  // 全部视图的「项目」列需要实时刷新。
  const [, setProjectStoreVersion] = useState(0);
  useEffect(() => {
    const bump = () => setProjectStoreVersion((v) => v + 1);
    const unsubUser = userStore.subscribe(bump);
    const unsubGroup = groupStore.subscribe(bump);
    return () => {
      unsubUser();
      unsubGroup();
    };
  }, []);

  // 存量 Agent 实例处理弹窗（编辑用户组织：弹窗①②③）
  const [agentInstanceDialog, setAgentInstanceDialog] = useState<{
    open: boolean;
    affectedUsers: AffectedUser[];
    pendingAction: () => void; // 确认处理后执行的原始操作
  } | null>(null);

  // 同步后存量 Agent 实例处理弹窗（弹窗⑦：多用户混合 + 末尾上级组织自动迁移块⑥）
  const [syncAgentInstanceDialog, setSyncAgentInstanceDialog] = useState<{
    open: boolean;
    affectedUsers: AffectedUser[];
    parentMigration?: ParentMigration;
  } | null>(null);

  // 查看配置对比弹窗（弹窗⑧）
  const [mmConfigDiffDialog, setMmConfigDiffDialog] = useState<{
    open: boolean;
    fromGroupName: string;
    toGroupName: string;
    instances: MmInstanceConfigCompare[];
  } | null>(null);

  // 筛选逻辑：hasOneid 模式时支持部门和角色筛选
  const filtered = sortedMembers.filter((m) => {
    // 搜索筛选
    if (!m.id.toLowerCase().includes(search.toLowerCase())) return false;
    // OneID 模式：部门筛选（外部下拉 deptFilter，仅 oneid 专用模式可见）
    if (hasOneid && deptFilter) {
      const memberDept = MOCK_MEMBER_DEPARTMENTS[m.id] || "";
      // 根据部门 ID 匹配部门路径
      const findDeptPath = (nodes: DepartmentNode[], id: string): string | undefined => {
        for (const n of nodes) {
          if (n.id === id) return n.path;
          if (n.children) {
            const found = findDeptPath(n.children, id);
            if (found) return found;
          }
        }
        return undefined;
      };
      const selectedPath = findDeptPath(MOCK_DEPARTMENTS, deptFilter);
      if (selectedPath && !memberDept.startsWith(selectedPath)) return false;
    }
    // OneID 模式：角色筛选
    if (hasOneid && roleFilter !== "all" && m.role !== roleFilter) return false;
    return true;
  });

  // ─── 列头筛选叠加（部门/组织） ──────────────────────────────────────
  // 与 Agent 列表（OpenClawMonitor）一致：基于 id 集合（命中节点及其所有子孙）。
  // - 部门列头筛选：复用 deptFilter state（与外部下拉共用同一个 path-based 过滤逻辑，
  //   外部下拉在 unified 模式已隐藏，列头入口只在 hasDeptData 时展示，所以两者不会冲突）。
  // - 组织列头筛选：基于 hasOneid 时的全部 group / 普通模式 manual group。
  const colGroupAllowedIds = React.useMemo(() => {
    if (!groupFilter) return null;
    const allGroups = hasOneid ? MM_MOCK_GROUPS : MM_MOCK_MANUAL_GROUPS;
    const trees = mmBuildGroupTree(allGroups);
    type Tree = { id: string; children: Tree[] };
    const collect = (nodes: Tree[], targetId: string): string[] | null => {
      for (const n of nodes) {
        if (n.id === targetId) {
          const ids = [n.id];
          const dfs = (cs: Tree[]) => cs.forEach((c) => { ids.push(c.id); dfs(c.children); });
          dfs(n.children);
          return ids;
        }
        const found = collect(n.children, targetId);
        if (found) return found;
      }
      return null;
    };
    return collect(trees as Tree[], groupFilter) ?? [];
  }, [hasOneid, groupFilter]);

  // 把组织列头筛选叠加到 filtered 之上（部门列头筛选与外部下拉共享 deptFilter，已在 filtered 里生效）
  const colFiltered = filtered.filter((m) => {
    if (colGroupAllowedIds) {
      let userGroupIds: string[] = [];
      if (hasOneid) {
        userGroupIds = getMmUserGroupItems(m.id).map((g) => g.id);
      } else {
        userGroupIds = getManualUserGroupPaths(m.id).map((g) => g.id);
      }
      if (!userGroupIds.some((id) => colGroupAllowedIds.includes(id))) return false;
    }
    return true;
  });

  const totalPages = Math.max(1, Math.ceil(colFiltered.length / PAGE_SIZE));
  const currentPage = Math.min(page, totalPages);
  const paginated = colFiltered.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE);

  const handleAdd = () => {
    if (!newMember.id.trim()) { toast.error("请输入用户 ID"); return; }
    const pwd = generatePassword();
    setMembers([...members, {
      id: newMember.id, role: newMember.role, status: "active",
      clawLimit: newMember.clawLimit, tokenLimit: newMember.tokenLimit,
      clawCount: 0, joinTime: new Date().toISOString().slice(0, 10),
      vpcType: "auto" as const, vpcName: `agent/${newMember.id.split("@")[0]}`, hasVpcResources: false,
    }]);
    // 将新用户添加到选中的组织
    if (newMember.groupIds.length > 0) {
      setGroups(groups.map((g) =>
        newMember.groupIds.includes(g.id)
          ? { ...g, memberIds: [...g.memberIds, newMember.id] }
          : g
      ));
    }
    setShowAddDialog(false);
    setNewMember(makeEmptyNewMember());
    setCredentialDialog({ open: true, title: "成员已创建", memberId: newMember.id, password: pwd });
  };

  const openEditDialog = (member: typeof MOCK_MEMBERS_BASE[0]) => {
    // 从实际 mock 用户数据获取该用户的 groupIds（用于组织选择框）
    const userOrg = hasOneid
      ? MM_MOCK_USERS.find((u) => u.userId === member.id)
      : MM_MOCK_USERS_MANUAL.find((u) => u.userId === member.id);
    const actualGroupIds = userOrg?.groupIds ?? [];
    if (hasOneid) {
      setOneidEditForm({
        id: member.id,
        role: member.role,
        department: MOCK_MEMBER_DEPARTMENTS[member.id] || "",
        clawLimit: member.clawLimit,
        tokenLimit: member.tokenLimit,
        groupIds: actualGroupIds,
      });
    } else {
      setEditForm({
        id: member.id,
        role: member.role,
        clawLimit: member.clawLimit,
        tokenLimit: member.tokenLimit,
        groupIds: actualGroupIds,
      });
      setIsInitialAdminEdit(initialAdminIds.has(member.id));
    }
    setEditMemberId(member.id);
  };

  const handleEdit = () => {
    const targetId = editMemberId!;
    const newGroupIds = hasOneid ? oneidEditForm.groupIds : editForm.groupIds;

    // 获取用户原来的 groupIds
    const userOrg = hasOneid
      ? MM_MOCK_USERS.find((u) => u.userId === targetId)
      : MM_MOCK_USERS_MANUAL.find((u) => u.userId === targetId);
    const oldGroupIds = userOrg?.groupIds ?? [];
    // 找出被移除的组织
    const removedGroupIds = oldGroupIds.filter((gId) => !newGroupIds.includes(gId));

    // 检查被移除组织中是否有 Agent 实例，构建 per-group 处理配置（弹窗①②③）
    const allGroups = hasOneid ? MM_MOCK_GROUPS : MM_MOCK_MANUAL_GROUPS;
    const usersList = hasOneid ? MM_MOCK_USERS : MM_MOCK_USERS_MANUAL;
    const affectedGroups = mmBuildAffectedGroups(
      targetId,
      removedGroupIds,
      newGroupIds,
      usersList,
      allGroups,
      oldGroupIds,
    );

    const doEdit = () => {
      if (hasOneid) {
        setMembers(members.map((m) =>
          m.id === targetId
            ? { ...m, clawLimit: oneidEditForm.clawLimit, tokenLimit: oneidEditForm.tokenLimit }
            : m
        ));
      } else {
        setMembers(members.map((m) =>
          m.id === targetId
            ? { ...m, role: editForm.role, clawLimit: editForm.clawLimit, tokenLimit: editForm.tokenLimit }
            : m
        ));
      }
      // 同步组织数据：先从所有组中移除该用户，再添加到选中的组
      setGroups(groups.map((g) => {
        const without = g.memberIds.filter((id) => id !== targetId);
        if (newGroupIds.includes(g.id)) {
          return { ...g, memberIds: Array.from(new Set([...without, targetId])) };
        }
        return { ...g, memberIds: without };
      }));

      // 回写底层用户库的 groupIds，保证其它页面（CreateAgentDialog 等）通过
      // useUsers() 读到的是最新态；配合下方 notifyUsersChanged() 触发订阅刷新。
      const source = hasOneid ? MM_MOCK_USERS : MM_MOCK_USERS_MANUAL;
      const idx = source.findIndex((u) => u.userId === targetId);
      if (idx >= 0) {
        source[idx] = { ...source[idx], groupIds: [...newGroupIds] };
        notifyUsersChanged();
      }

      setEditMemberId(null);
      toast.success("用户信息已更新");
    };

    if (affectedGroups.length > 0) {
      // 有存量实例，弹出处理弹窗（按原组织逐项选择处理方式）
      const affectedUser: AffectedUser = {
        userId: targetId,
        originalGroups: oldGroupIds.map((gId) => mmGetPrimaryDeptPath(gId, allGroups)),
        newGroups: newGroupIds.map((gId) => mmGetPrimaryDeptPath(gId, allGroups)),
        affectedGroups,
      };
      setAgentInstanceDialog({
        open: true,
        affectedUsers: [affectedUser],
        pendingAction: doEdit,
      });
    } else {
      doEdit();
    }
  };

  // 手动同步（OneID 模式）
  const handleSync = useCallback(() => {
    setIsSyncing(true);
    // 模拟同步过程：假设 OneID 侧删除了 jack@acompany.com 和 iris@acompany.com
    setTimeout(() => {
      const oneidDeletedUserIds = ["jack@acompany.com", "iris@acompany.com"];
      // 模拟新增用户数量
      const addedCount = 0;

      // 检查每个被删除用户名下的 Agent 数量
      const failedUsers: { id: string; clawCount: number; vpcName?: string }[] = [];
      let deletedCount = 0;
      // 模拟私有网络绑定情况
      const vpcBindings: Record<string, string> = {
        "iris@acompany.com": "openclaw/iris",
      };

      setMembers((prev) => {
        const updated = prev.map((m) => {
          if (!oneidDeletedUserIds.includes(m.id)) return m;
          const hasVpc = !!vpcBindings[m.id];
          if (m.clawCount > 0 || hasVpc) {
            // 有未清理的 Agent 或有私有网络绑定，不能删除，改为禁用
            failedUsers.push({ id: m.id, clawCount: m.clawCount, vpcName: vpcBindings[m.id] });
            return { ...m, status: "disabled" };
          } else {
            // 无 Agent，直接删除
            deletedCount++;
            return { ...m, _deleted: true };
          }
        });
        // 过滤掉直接删除的用户
        return updated.filter((m) => !(m as any)._deleted);
      });

      setIsSyncing(false);

      if (failedUsers.length > 0) {
        // 有无法删除的用户，弹窗提醒（已同步过组织架构时同时展示组织异常）
        const groupAnomalies = mmDeptSynced ? MM_MOCK_SYNC_RESULT.anomalousGroups : [];
        setSyncResultDialog({
          open: true,
          failedUsers,
          deletedCount,
          addedCount,
          anomalousGroups: groupAnomalies,
        });
        // 同步异常组织数据到 GroupView 以显示红点
        if (groupAnomalies.length > 0) {
          setMmAnomalousGroups(groupAnomalies);
          // 模拟：组织架构被删除后，用户从这些组织中被移除
          const deletedGroupIds = new Set(["dept-operation", "dept-operation-1", "dept-operation-2"]);
          let mutated = false;
          MM_MOCK_USERS.forEach((u, idx) => {
            if (u.groupIds.some((gid) => deletedGroupIds.has(gid))) {
              MM_MOCK_USERS[idx] = {
                ...u,
                groupIds: u.groupIds.filter((gid) => !deletedGroupIds.has(gid)),
              };
              mutated = true;
            }
          });
          if (mutated) notifyUsersChanged();
        }
      } else {
        const parts: string[] = [];
        if (addedCount > 0) parts.push(`新增 ${addedCount} 个`);
        if (deletedCount > 0) parts.push(`删除 ${deletedCount} 个`);
        toast.success(`同步完成${parts.length > 0 ? `，${parts.join("，")}用户` : ""}`);
      }
    }, 2000);
  }, [mmDeptSynced]);

  const handleToggleStatus = (id: string) => {
    setMembers(members.map((m) =>
      m.id === id ? { ...m, status: m.status === "active" ? "disabled" : "active" } : m
    ));
    toast.success("状态已更新");
  };

  const openDeleteCheck = (member: typeof MOCK_MEMBERS_BASE[0]) => {
    setDeleteCheckDialog({
      open: true,
      memberId: member.id,
      clawCount: member.clawCount,
      vpcType: member.vpcType,
      vpcName: member.vpcName,
      hasVpcResources: member.hasVpcResources,
      clawRefreshing: false,
      vpcRefreshing: false,
    });
  };

  const openDisableConfirm = (member: typeof MOCK_MEMBERS_BASE[0]) => {
    setDisableConfirmDialog({ open: true, memberId: member.id, clawCount: member.clawCount });
  };

  const openEnableConfirm = (member: typeof MOCK_MEMBERS_BASE[0]) => {
    setEnableConfirmDialog({ open: true, memberId: member.id, clawCount: member.clawCount });
  };

  const handleDisable = (id: string) => {
    setMembers(members.map((m) => m.id === id ? { ...m, status: "disabled" } : m));
    setDisableConfirmDialog(null);
    toast.success("用户已禁用");
  };

  const handleEnable = (id: string) => {
    setMembers(members.map((m) => m.id === id ? { ...m, status: "active" } : m));
    setEnableConfirmDialog(null);
    toast.success("用户已启用");
  };

  const handleDisableApiToken = (id: string) => {
    setApiTokenDisabledIds((prev) => new Set(Array.from(prev).concat(id)));
    setApiTokenDisableDialog(null);
    toast.success(`已禁用 ${id.split("@")[0]} 的 API Token`);
  };

  const handleEnableApiToken = (id: string) => {
    setApiTokenDisabledIds((prev) => { const next = new Set(prev); next.delete(id); return next; });
    setApiTokenEnableDialog(null);
    toast.success(`已启用 ${id.split("@")[0]} 的 API Token`);
  };

  const handleDelete = (id: string) => {
    setMembers(members.filter((m) => m.id !== id));
    setDeleteConfirmDialog(null);
    toast.success("用户已删除");
  };

  const handleReset = () => {
    const memberId = showResetDialog ?? "";

    // ─── unified 模式：管理员直接输入新密码 ─────────────────────────────────
    if (isUnified) {
      const { newPassword, confirmPassword } = resetForm;

      // 1. 非空
      if (!newPassword.trim() || !confirmPassword.trim()) {
        if (!newPassword.trim()) setResetNewPwdError("请输入新密码");
        if (!confirmPassword.trim()) setResetConfirmPwdError("请再次输入新密码");
        return;
      }
      // 2. 强度校验
      const strengthError = validatePasswordStrength(newPassword);
      if (strengthError) {
        setResetNewPwdError(strengthError);
        return;
      }
      // 3. 两次一致
      if (newPassword !== confirmPassword) {
        setResetConfirmPwdError("两次输入的密码需保持一致");
        return;
      }

      setShowResetDialog(null);
      setResetForm({ ...emptyResetForm });
      setResetNewPwdError(null);
      setResetConfirmPwdError(null);
      setShowResetNewPwd(false);
      setShowResetConfirmPwd(false);
      toast.success("密码重置成功");
      return;
    }

    // ─── 其他模式（standard / custom）：保留原\"系统自动生成 + 结果弹窗\"流程 ─
    const pwd = generatePassword();
    setShowResetDialog(null);
    setResetForm({ ...emptyResetForm });
    setCredentialDialog({ open: true, title: "密码已重置", memberId, password: pwd });
  };

  // ─── 组织操作 ─────────────────────────────────────────────────────────────
  const handleCreateGroup = () => {
    if (!newGroupName.trim()) { toast.error("请输入组织名称"); return; }
    if (groups.some((g) => g.name === newGroupName.trim())) { toast.error("组织名称已存在"); return; }
    const newGroup: MemberGroup = { id: `grp-${Date.now()}`, name: newGroupName.trim(), memberIds: [], createdAt: new Date().toISOString().slice(0, 10) };
    setGroups([...groups, newGroup]);
    setNewGroupName("");
    setNewGroupParentId(null);
    setShowCreateGroupDialog(false);
    setSelectedGroupId(newGroup.id);
    // 如果添加用户弹窗打开，自动选中新组织并重新打开 Popover
    if (showAddDialog) {
      setNewMember((prev) => ({ ...prev, groupIds: [newGroup.id] }));
      setTimeout(() => setGroupPopoverReopenKey((k) => k + 1), 150);
    }
    // 如果编辑用户弹窗打开，自动选中新组织并重新打开 Popover
    if (editMemberId) {
      if (hasOneid) {
        setOneidEditForm((prev) => ({ ...prev, groupIds: [newGroup.id] }));
      } else {
        setEditForm((prev) => ({ ...prev, groupIds: [newGroup.id] }));
      }
      setTimeout(() => setGroupPopoverReopenKey((k) => k + 1), 150);
    }
    toast.success("组织已创建");
  };

  const handleRenameGroup = (groupId: string) => {
    if (!editingGroupName.trim()) return;
    if (groups.some((g) => g.id !== groupId && g.name === editingGroupName.trim())) { toast.error("组织名称已存在"); return; }
    setGroups(groups.map((g) => g.id === groupId ? { ...g, name: editingGroupName.trim() } : g));
    setEditingGroupId(null);
    toast.success("组织已重命名");
  };

  const handleDeleteGroup = (groupId: string) => {
    setGroups(groups.filter((g) => g.id !== groupId));
    setDeleteGroupDialog(null);
    if (selectedGroupId === groupId) {
      const remaining = groups.filter((g) => g.id !== groupId);
      setSelectedGroupId(remaining.length > 0 ? remaining[0].id : "__ungrouped__");
    }
    toast.success("组织已删除，用户保留");
  };

  const handleRemoveFromGroup = (groupId: string, memberId: string) => {
    setGroups(groups.map((g) => g.id === groupId ? { ...g, memberIds: g.memberIds.filter((id) => id !== memberId) } : g));
    toast.success("已从组织中移除");
  };

  const handleAddMembersToGroup = () => {
    if (addToGroupSelected.length === 0) return;
    setGroups(groups.map((g) => {
      if (g.id !== selectedGroupId) return g;
      const newIds = Array.from(new Set([...g.memberIds, ...addToGroupSelected]));
      return { ...g, memberIds: newIds };
    }));
    setShowAddToGroupDialog(false);
    setAddToGroupSearch("");
    setAddToGroupSelected([]);
    toast.success(`已添加 ${addToGroupSelected.length} 名用户到组织`);
  };

  // 组织视图的数据
  const getGroupMembers = () => {
    if (selectedGroupId === "__ungrouped__") {
      const allGroupedIds = new Set(groups.flatMap((g) => g.memberIds));
      return sortedMembers.filter((m) => !allGroupedIds.has(m.id));
    }
    const group = groups.find((g) => g.id === selectedGroupId);
    if (!group) return [];
    return sortedMembers.filter((m) => group.memberIds.includes(m.id));
  };

  const groupFiltered = viewMode === "group" ? getGroupMembers() : [];
  const groupTotalPages = Math.max(1, Math.ceil(groupFiltered.length / PAGE_SIZE));
  const groupCurrentPage = Math.min(groupPage, groupTotalPages);
  const groupPaginated = groupFiltered.slice((groupCurrentPage - 1) * PAGE_SIZE, groupCurrentPage * PAGE_SIZE);

  return (
    <>
      <div className="page-enter min-w-0 overflow-hidden">
        <AdminPageHeader
          title="用户管理"
          description={
            <>
              管理企业用户的访问权限和资源配额
              {hasOneid && !isUnified && (
                <>
                  <span className="mx-2">|</span>
                  <button
                    onClick={async () => {
                      window.open(
                        "https://xxx.com/login",
                        "_blank"
                      );
                    }}
                    className="text-[#737373] hover:text-[#355EF1] inline-flex items-center gap-1 transition-colors cursor-pointer bg-transparent border-none p-0"
                  >
                    前往腾讯统一身份管理用户
                    <ExternalLink className="w-3.5 h-3.5" />
                  </button>
                </>
              )}
            </>
          }
          className="mb-8"
        />

        {/* 我的数据源（OneID 模式下不展示） */}
        {!hasOneid && configuredAuthSources.length > 0 && (
          <div className="mb-5">
            <h3 className="text-sm font-semibold text-[#525252] mb-3">我的数据源</h3>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
              {configuredAuthSources.map((source) => (
                <div
                  key={source.id}
                  className="bg-white rounded-xl border border-[#e5e5e5] p-4 transition-all"
                >
                  <div className="flex items-start gap-3 mb-3">
                    <div className="w-10 h-10 rounded-[4px] bg-white border border-[#e5e5e5] flex items-center justify-center overflow-hidden flex-shrink-0">
                      <img
                        src={source.iconUrl}
                        alt={source.name}
                        className="w-7 h-7 object-contain"
                      />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-semibold text-[#09090b]">{source.name}</p>
                      <p className="text-xs text-[#737373] mt-0.5">{source.description}</p>
                    </div>
                  </div>
                  <div className="flex items-center justify-between pt-2 border-t border-[#f5f5f5]">
                    <div className="flex items-center gap-3">
                      <button
                        className="flex items-center gap-1 text-xs text-[#737373] hover:text-[#355EF1] transition-colors"
                        onClick={() => {
                          setAuthSourceInitialStep(2);
                          setAuthSourceInitialId(source.id);
                          setAuthSourceInitialFormValues(source.formValues || null);
                          setShowAuthSourceDialog(true);
                        }}
                      >
                        <Pencil className="w-3 h-3" />
                        编辑
                      </button>
                      <button
                        className="flex items-center gap-1 text-xs text-[#737373] hover:text-[#355EF1] transition-colors"
                        onClick={() => {
                          setAuthSourceInitialStep(1);
                          setAuthSourceInitialId(null);
                          setAuthSourceInitialFormValues(null);
                          setShowAuthSourceDialog(true);
                        }}
                      >
                        <RefreshCw className="w-3 h-3" />
                        更换
                      </button>
                      <button
                        className="flex items-center gap-1 text-xs text-[#737373] hover:text-[#d42a1e] transition-colors"
                        onClick={() => {
                          setDeleteAuthSourceConfirm({ open: true, source });
                        }}
                      >
                        <Trash2 className="w-3 h-3" />
                        删除
                      </button>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className={`text-xs ${source.enabled ? "text-[#355EF1]" : "text-[#A3A3A3]"}`}>
                        {source.enabled ? "已启用" : "已禁用"}
                      </span>
                      <Switch
                        checked={source.enabled}
                        onCheckedChange={(checked) => {
                          setConfiguredAuthSources(configuredAuthSources.map((s) =>
                            s.id === source.id ? { ...s, enabled: checked } : s
                          ));
                          toast.success(checked ? `已启用数据源：${source.name}` : `已禁用数据源：${source.name}`);
                        }}
                      />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Search + Filter + Actions Row
            data-guide：供步骤指引气泡（GuideHighlightBubble）按真实位置贴合标注，避免随机浮层 */}
        <div className="flex items-center justify-between mb-4" data-guide="member-toolbar">
          <div className="flex w-[676px] h-9 items-center gap-2 px-0 py-0 text-base leading-6 text-[#0A0A0A]">
            {/* 视图切换按钮组（最左侧，两种模式通用）
              * 停服态豁免：切换「全部 / 组织 / 项目」视图属于查看类操作，
              * 与全局搜索/翻页同档，需保持可用（不置灰、不拦截点击）。
              * SegmentOption 自身没有 disabled 属性，"停服前已禁用则延续禁用"的
              * 约束通过组件级disabled 传递依然生效（此处均未设置）。 */}
            <SegmentGroup data-guide="member-view-segment" data-billing-exempt>
              <SegmentOption
                active={viewMode === "all"}
                onClick={() => { setViewMode("all"); setPage(1); }}
              >
                全部
              </SegmentOption>
              <SegmentOption
                active={viewMode === "group"}
                onClick={() => { setViewMode("group"); setGroupPage(1); }}
              >
                组织
              </SegmentOption>
              <SegmentOption
                active={viewMode === "project"}
                onClick={() => { setViewMode("project"); }}
              >
                项目
              </SegmentOption>
            </SegmentGroup>
            {/* OneID 模式：部门筛选（统一版隐藏） */}
            {hasOneid && !isUnified && viewMode !== "project" && (
              <TreeSelectFilter
                nodes={MOCK_DEPARTMENTS}
                value={deptFilter}
                onChange={(v) => { setDeptFilter(v); setPage(1); }}
                allLabel="全部部门"
                searchPlaceholder="搜索部门"
              />
            )}
            {/* OneID 专用模式：角色筛选（unified 统一模式不展示） */}
            {hasOneid && !isUnified && viewMode !== "project" && (
              <Select
                value={roleFilter}
                onValueChange={(v) => { setRoleFilter(v as "all" | "admin" | "member"); setPage(1); }}
              >
                <SelectTrigger className="w-[160px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {["all", "admin", "member"].map((v) => (
                    <SelectItem key={v} value={v}>
                      {v === "all" ? "全部角色" : v === "admin" ? "管理员" : "用户"}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
            {/* 搜索框（统一版「组织」视图 + 「项目」视图下隐藏，项目视图有独立搜索） */}
            {!(isUnified && viewMode === "group") && viewMode !== "project" && (
              <div className="relative w-[260px]">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)]" />
                <Input
                  placeholder="搜索用户 ID..."
                  value={search}
                  onChange={(e) => { setSearch(e.target.value); setPage(1); }}
                  className="pl-9"
                />
              </div>
            )}
            {/* 清除筛选按钮 - 当有任何筛选条件时显示（与"手动同步"保持一致的 claw-outline 样式） */}
            {viewMode !== "project" && (deptFilter || search.trim() || roleFilter !== "all") && (
              <Button
                variant="claw-outline"
                onClick={() => {
                  setDeptFilter("");
                  setSearch("");
                  setRoleFilter("all");
                  setPage(1);
                }}
              >
                <X className="w-4 h-4" />
                清除筛选
              </Button>
            )}
          </div>

          {/* 右端操作区：手动同步（普通 OneID） / 下载 + 添加用户 + 同步（统一版·全部视图） */}
          <div className="flex items-center gap-2">
            {/* 普通 OneID 模式：手动同步按钮（统一版移至最右侧，见下方） */}
            {hasOneid && !isUnified && viewMode !== "project" && (
                <Button
                variant="claw-outline"
                onClick={handleSync}
                disabled={isSyncing}
              >
                {isSyncing ? (
                  <><Loader2 className="w-4 h-4 mr-2 animate-spin" />同步中...</>
                ) : (
                  <><RefreshCw className="w-4 h-4 mr-2" />手动同步</>
                )}
              </Button>
            )}

            {/* 普通模式 + 统一模式：导出用户列表 + 添加用户（仅全部视图） */}
            {viewMode === "all" && showCustomExtras && (
              <>
                <Button
                  variant="claw-outline"
                  size="icon"
                  className="w-9 h-9"
                  title="导出用户列表"
                  onClick={() => {
                    const headers = ["用户ID", "姓名", "角色", "状态", "创建时间"];
                    const rows = members.map((m: any) => [m.id || "", m.name || m.username || "", m.role || "", m.status || "", m.createdAt || m.created_at || ""]);
                    const csv = [headers, ...rows].map(r => r.join(",")).join("\n");
                    const blob = new Blob(["﻿" + csv], { type: "text/csv;charset=utf-8;" });
                    const url = URL.createObjectURL(blob);
                    const a = document.createElement("a");
                    a.href = url;
                    a.download = `用户列表_${new Date().toLocaleDateString("zh-CN").replace(/\//g, "-")}.csv`;
                    a.click();
                    URL.revokeObjectURL(url);
                    toast.success("用户列表已导出");
                  }}
                >
                  <Download className="w-4 h-4" />
                </Button>
                {members.length >= 20 ? (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <div className="relative inline-block cursor-not-allowed">
                        <Button className="pointer-events-none select-none text-sm" tabIndex={-1} aria-disabled="true">
                          添加用户<ChevronDown className="w-3.5 h-3.5" />
                        </Button>
                        <div className="absolute inset-0 rounded-[4px] bg-white/50 pointer-events-none" />
                      </div>
                    </TooltipTrigger>
                    <TooltipContent side="bottom" className="max-w-[224px]">
                      当前用户数已达上限，无法再添加
                    </TooltipContent>
                  </Tooltip>
                ) : (
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button className="text-sm">
                        添加用户<ChevronDown className="w-3.5 h-3.5" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => setShowAddDialog(true)}><Plus className="w-4 h-4 mr-2" />单个添加</DropdownMenuItem>
                      <DropdownMenuItem onClick={() => setShowBatchDialog(true)}><Upload className="w-4 h-4 mr-2" />批量导入</DropdownMenuItem>
                      <DropdownMenuItem onClick={() => {
                        if (isUnified) {
                          // 统一模式：跳转到外部数据源管理网页（占位 URL，待后续替换为实际地址）
                          window.open("https://example.com/auth-source-import", "_blank", "noopener,noreferrer");
                          return;
                        }
                        setAuthSourceInitialStep(undefined);
                        setAuthSourceInitialId(null);
                        setAuthSourceInitialFormValues(null);
                        setShowAuthSourceDialog(true);
                      }}><Link2 className="w-4 h-4 mr-2" />数据源导入</DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                )}
              </>
            )}

            {/* 统一版·全部视图：同步按钮（最右侧） */}
            {isUnified && viewMode === "all" && (
              <Button
                variant="claw-outline"
                onClick={handleSync}
                disabled={isSyncing}
              >
                {isSyncing ? (
                  <><Loader2 className="w-4 h-4 mr-2 animate-spin" />同步中...</>
                ) : (
                  <><RefreshCw className="w-4 h-4 mr-2" />同步</>
                )}
              </Button>
            )}
          </div>
        </div>

        {/* Table - 全部视图 */}
        {viewMode === "all" && (
        <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
          <Table variant="white" scrollX={hasOneid ? 1520 : 1380}>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead fixed="left" style={{ width: "220px", minWidth: "220px", maxWidth: "220px" }}>
                  <div className="flex items-center gap-1.5">
                    用户 ID
                    <Tooltip>
                      <TooltipTrigger asChild><span className="cursor-pointer inline-flex"><Info className="w-3.5 h-3.5 text-[#A3A3A3]" /></span></TooltipTrigger>
                      <TooltipContent sideOffset={4}>企业用户的唯一 ID，例如企业邮箱或企业用户唯一名称</TooltipContent>
                    </Tooltip>
                  </div>
                </TableHead>
                {hasOneid && !isUnified && (
                  <>
                    <TableHead style={{ minWidth: "200px" }}>
                      <div className="flex items-center gap-1.5">
                        部门
                        <Tooltip>
                          <TooltipTrigger asChild><span className="cursor-pointer inline-flex"><Info className="w-3.5 h-3.5 text-[#A3A3A3]" /></span></TooltipTrigger>
                          <TooltipContent sideOffset={4}>用户的部门信息来自腾讯统一身份管理平台</TooltipContent>
                        </Tooltip>
                      </div>
                    </TableHead>
                    <TableHead style={{ minWidth: "200px" }}>组织</TableHead>
                  </>
                )}
                {hasOneid && isUnified && (
                  <TableHead style={{ minWidth: "200px" }}>组织</TableHead>
                )}
                {!hasOneid && (
                  <TableHead style={{ minWidth: "200px" }}>组织</TableHead>
                )}
                <TableHead style={{ minWidth: "154px" }}>项目</TableHead>
                <TableHead style={{ minWidth: "80px" }}>角色</TableHead>
                <TableHead style={{ minWidth: "80px" }}>状态</TableHead>
                <TableHead style={{ minWidth: "120px" }}>
                  <div className="flex items-center gap-1.5">
                    Agent 上限
                    <Tooltip>
                      <TooltipTrigger asChild><span className="cursor-pointer inline-flex"><Info className="w-3.5 h-3.5 text-[#A3A3A3]" /></span></TooltipTrigger>
                      <TooltipContent sideOffset={4}>单个企业用户最多可以创建的 Agent 数量</TooltipContent>
                    </Tooltip>
                  </div>
                </TableHead>
                <TableHead>
                  <div className="flex items-center gap-1.5">
                    每日 Tokens 上限
                    <Tooltip>
                      <TooltipTrigger asChild><span className="cursor-pointer inline-flex"><Info className="w-3.5 h-3.5 text-[#A3A3A3]" /></span></TooltipTrigger>
                      <TooltipContent sideOffset={4}>单个企业用户每日最多可消耗的 Tokens 数量</TooltipContent>
                    </Tooltip>
                  </div>
                </TableHead>
                <TableHead style={{ minWidth: "110px" }}>加入时间</TableHead>
                <TableHead fixed="right" style={{ width: "112px", minWidth: "112px" }}>
                  操作
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginated.map((member) => {
                const memberGroups = groups.filter((g) => g.memberIds.includes(member.id));
                const groupNames = memberGroups.map((g) => g.name);
                // OneID 模式：从 MM_MOCK_USERS 获取部门 + 组织
                const mmDeptPaths = hasOneid ? getMmUserDeptPaths(member.id) : [];
                const mmGroupItems = hasOneid ? getMmUserGroupItems(member.id) : [];
                // 普通模式：从 MM_MOCK_USERS_MANUAL 获取组织完整路径
                const manualGroupPaths = !hasOneid ? getManualUserGroupPaths(member.id) : [];
                // 项目：跨页共享 userStore + groupStore（source='project'）
                const projectPaths = getUserProjectPaths(member.id);
                return (
                <TableRow key={member.id}>
                  <TableCell fixed="left" style={{ width: "220px", minWidth: "220px", maxWidth: "220px" }}>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="block max-w-[180px] truncate font-medium text-[var(--text-emphasis)] cursor-pointer">{member.id}</span>
                      </TooltipTrigger>
                      <TooltipContent side="top" className="text-xs max-w-xs break-all">{member.id}</TooltipContent>
                    </Tooltip>
                  </TableCell>
                  {hasOneid && !isUnified && (
                    <>
                      {/* 部门列 */}
                      <TableCell style={{ minWidth: "200px" }}>
                        {mmDeptPaths.length === 0 ? (
                          <span className="text-[var(--text-weak)]">—</span>
                        ) : (
                          <Tooltip delayDuration={200}>
                            <TooltipTrigger asChild>
                              <span className="inline-flex items-center gap-1 max-w-[200px] cursor-default">
                                <span className="truncate text-[var(--text-secondary)]">
                                  {mmDeptPaths[0].path}
                                </span>
                                {mmDeptPaths.length > 1 && (
                                  <span className="shrink-0 tabular-nums text-[var(--text-muted)]">
                                    +{mmDeptPaths.length - 1}
                                  </span>
                                )}
                              </span>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="text-xs max-w-[360px]">
                              <div className="space-y-1">
                                {mmDeptPaths.map((dp, idx) => (
                                  <div key={idx} className="break-all">
                                    {mmDeptPaths.length > 1 && <span className="tabular-nums mr-1">{idx + 1}.</span>}
                                    {dp.path}
                                    {dp.isPrimary && <span className="ml-1 text-[10px] opacity-70">（主部门）</span>}
                                  </div>
                                ))}
                              </div>
                            </TooltipContent>
                          </Tooltip>
                        )}
                      </TableCell>
                      {/* 组织列（OneID 模式：紧跟部门列） */}
                      <TableCell style={{ minWidth: "200px" }}>
                        {mmGroupItems.length === 0 ? (
                          <span className="text-[var(--text-weak)]">—</span>
                        ) : (
                          <Tooltip delayDuration={200}>
                            <TooltipTrigger asChild>
                              <span className="inline-flex items-center gap-1 max-w-[200px] cursor-default">
                                <span className="truncate text-sm text-[var(--text-secondary)]">
                                  {mmGroupItems[0].path}
                                </span>
                                {mmGroupItems.length > 1 && (
                                  <span className="shrink-0 tabular-nums text-[var(--text-muted)]">
                                    +{mmGroupItems.length - 1}
                                  </span>
                                )}
                              </span>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="text-xs max-w-[360px]">
                              <div className="space-y-1">
                                {mmGroupItems.map((gi, idx) => (
                                  <div key={idx} className="break-all">
                                    {mmGroupItems.length > 1 && <span className="tabular-nums mr-1">{idx + 1}.</span>}
                                    {gi.path}
                                  </div>
                                ))}
                              </div>
                            </TooltipContent>
                          </Tooltip>
                        )}
                      </TableCell>
                    </>
                  )}
                  {hasOneid && isUnified && (
                    /* 统一版：仅一列「组织」，内容沿用部门路径（主组织置顶 + 主组织角标） */
                    <TableCell style={{ minWidth: "200px" }}>
                      {mmDeptPaths.length === 0 ? (
                        <span className="text-[var(--text-weak)]">—</span>
                      ) : mmDeptPaths.length === 1 ? (
                        <span
                          className="inline-flex items-center gap-1.5 max-w-[200px] text-[var(--text-secondary)]"
                          title={mmDeptPaths[0].path}
                        >
                          <span className="truncate">{mmDeptPaths[0].path}</span>
                          {mmDeptPaths[0].isPrimary && (
                            <span className="inline-flex items-center text-[10px] font-medium text-[#355EF1] bg-[#EEF2FF] rounded px-1.5 py-0.5 shrink-0">
                              主组织
                            </span>
                          )}
                        </span>
                      ) : (
                        <HoverCard>
                          <HoverCardTrigger asChild>
                            <span className="inline-flex items-center gap-1 max-w-[200px] cursor-pointer">
                              <span className="truncate text-[var(--text-secondary)]">
                                {mmDeptPaths[0].path}
                              </span>
                              <span className="shrink-0 tabular-nums text-[var(--text-muted)]">
                                +{mmDeptPaths.length - 1}
                              </span>
                            </span>
                          </HoverCardTrigger>
                          <HoverCardContent className="text-xs max-w-[360px] p-0">
                            <div className="py-2">
                              {mmDeptPaths.map((dp, idx) => (
                                <div key={idx} className="px-3 py-1.5 text-sm flex items-center">
                                  <span className="text-[var(--text-muted)] mr-1 tabular-nums shrink-0">{idx + 1}.</span>
                                  <span className="text-[var(--text-emphasis)] break-all">{dp.path}</span>
                                  {dp.isPrimary && (
                                    <span className="ml-2 inline-flex items-center text-[10px] font-medium text-[#355EF1] bg-[#EEF2FF] rounded px-1.5 py-0.5 shrink-0">
                                      主组织
                                    </span>
                                  )}
                                </div>
                              ))}
                            </div>
                          </HoverCardContent>
                        </HoverCard>
                      )}
                    </TableCell>
                  )}
                  {!hasOneid && (
                    /* 普通模式组织列：紧跟用户ID，完整路径 + hover tooltip */
                    <TableCell style={{ minWidth: "200px" }}>
                      {manualGroupPaths.length === 0 ? (
                        <span className="text-[var(--text-weak)]">—</span>
                      ) : (
                        <Tooltip delayDuration={200}>
                          <TooltipTrigger asChild>
                            <span className="inline-flex items-center gap-1 max-w-[200px] cursor-default">
                              <span className="truncate text-sm text-[var(--text-secondary)]">
                                {manualGroupPaths[0].path}
                              </span>
                              {manualGroupPaths.length > 1 && (
                                <span className="shrink-0 tabular-nums text-[var(--text-muted)]">
                                  +{manualGroupPaths.length - 1}
                                </span>
                              )}
                            </span>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="text-xs max-w-[360px]">
                            <div className="space-y-1">
                              {manualGroupPaths.map((gp, idx) => (
                                <div key={idx} className="break-all">
                                  {manualGroupPaths.length > 1 && <span className="tabular-nums mr-1">{idx + 1}.</span>}
                                  {gp.path}
                                </div>
                              ))}
                            </div>
                          </TooltipContent>
                        </Tooltip>
                      )}
                    </TableCell>
                  )}
                  {/* 项目列（组织列右侧）：展示用户加入的项目，路径 + hover tooltip */}
                  <TableCell style={{ minWidth: "154px" }}>
                    {projectPaths.length === 0 ? (
                      <span className="text-[var(--text-weak)]">—</span>
                    ) : (
                      <Tooltip delayDuration={200}>
                        <TooltipTrigger asChild>
                          <span className="inline-flex items-center gap-1 max-w-[154px] cursor-default">
                            <span className="truncate text-sm text-[var(--text-secondary)]">
                              {projectPaths[0].path}
                            </span>
                            {projectPaths.length > 1 && (
                              <span className="shrink-0 tabular-nums text-[var(--text-muted)]">
                                +{projectPaths.length - 1}
                              </span>
                            )}
                          </span>
                        </TooltipTrigger>
                        <TooltipContent side="top" className="text-xs max-w-[360px]">
                          <div className="space-y-1">
                            {projectPaths.map((pp, idx) => (
                              <div key={idx} className="break-all">
                                {projectPaths.length > 1 && <span className="tabular-nums mr-1">{idx + 1}.</span>}
                                {pp.path}
                              </div>
                            ))}
                          </div>
                        </TooltipContent>
                      </Tooltip>
                    )}
                  </TableCell>
                  <TableCell>
                    {member.role === "admin" ? (
                      <StatusTag preset="role-admin" />
                    ) : (
                      <StatusTag preset="role-user" />
                    )}
                  </TableCell>
                  <TableCell>
                    {member.status === "active" ? (
                      <StatusTag mode="text" variant="green">正常</StatusTag>
                    ) : (
                      <StatusTag mode="text" variant="red">禁用</StatusTag>
                    )}
                  </TableCell>
                  <TableCell>
                    {(() => {
                      const quotas = getMemberGroupQuotas(member.id, hasOneid);
                      if (quotas.length > 0) {
                        return (
                            <Tooltip>
                              <TooltipTrigger asChild>
                              <span className="cursor-default border-b border-dashed border-[#d4d4d4]">按组织</span>
                              </TooltipTrigger>
                              <TooltipContent side="top" className="text-xs max-w-[240px]">
                              <div className="space-y-1">
                                {quotas.map((q) => (
                                  <div key={q.groupId} className="flex items-center justify-between gap-3">
                                    <span className="text-[#A3A3A3]">{q.groupName}</span>
                                    <span className="text-white font-medium">{q.clawLimit}</span>
                                  </div>
                                ))}
                              </div>
                            </TooltipContent>
                          </Tooltip>
                        );
                      }
                      return <span>{member.clawLimit}</span>;
                    })()}
                  </TableCell>
                  <TableCell>
                    {(() => {
                      const quotas = getMemberGroupQuotas(member.id, hasOneid);
                      if (quotas.length > 0) {
                        return (
                            <Tooltip>
                              <TooltipTrigger asChild>
                              <span className="cursor-default border-b border-dashed border-[#d4d4d4]">按组织</span>
                              </TooltipTrigger>
                              <TooltipContent side="top" className="text-xs max-w-[240px]">
                              <div className="space-y-1">
                                {quotas.map((q) => (
                                  <div key={q.groupId} className="flex items-center justify-between gap-3">
                                    <span className="text-[#A3A3A3]">{q.groupName}</span>
                                    <span className="text-white font-medium">{q.tokenLimit.toLocaleString()}</span>
                                  </div>
                                ))}
                              </div>
                            </TooltipContent>
                          </Tooltip>
                        );
                      }
                      return <span>{member.tokenLimit.toLocaleString()}</span>;
                    })()}
                  </TableCell>
                  <TableCell>
                    <span className="text-[var(--text-muted)]">{member.joinTime}</span>
                  </TableCell>
                  <TableActionCell fixed="right" style={{ width: "112px", minWidth: "112px" }}>
                    <Button
                      variant="link"
                      onClick={() => openEditDialog(member)}
                    >
                      编辑
                    </Button>
                    {showCustomExtras && (
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="link">
                            更多
                          </Button>
                        </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            {initialAdminIds.has(member.id) ? (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  {/* disabled 项不能直接放在 Tooltip 内（Radix 会吞 hover），用 span 包一层透传 */}
                                  <span>
                                    <DropdownMenuItem disabled onSelect={(e) => e.preventDefault()}>
                                      <Key />重置密码
                                    </DropdownMenuItem>
                                  </span>
                                </TooltipTrigger>
                                <TooltipContent side="left" className="max-w-[220px] text-xs leading-relaxed">管理员账号不允许重置密码</TooltipContent>
                              </Tooltip>
                            ) : (
                              <DropdownMenuItem onClick={() => { setShowResetDialog(member.id); setResetForm({ ...emptyResetForm }); setResetNewPwdError(null); setResetConfirmPwdError(null); setShowResetNewPwd(false); setShowResetConfirmPwd(false); }}>
                                <Key />重置密码
                              </DropdownMenuItem>
                            )}
                            {initialAdminIds.has(member.id) ? (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span>
                                    <DropdownMenuItem disabled onSelect={(e) => e.preventDefault()}>
                                      <UserX />禁用
                                    </DropdownMenuItem>
                                  </span>
                                </TooltipTrigger>
                                <TooltipContent side="left">管理员账号不可禁用</TooltipContent>
                              </Tooltip>
                            ) : member.status === "active" ? (
                              <DropdownMenuItem onClick={() => openDisableConfirm(member)}>
                                <UserX />禁用
                              </DropdownMenuItem>
                            ) : (
                              <DropdownMenuItem onClick={() => openEnableConfirm(member)}>
                                <UserCheck />启用
                              </DropdownMenuItem>
                            )}
                            {/* API Token 管理（互斥：Token 启用中 → 显示"禁用"；Token 已禁用 → 显示"启用"） */}
                            {apiTokenDisabledIds.has(member.id) ? (
                              <DropdownMenuItem onClick={() => setApiTokenEnableDialog({ open: true, memberId: member.id })}>
                                <Shield className="w-4 h-4" />启用 API Token
                              </DropdownMenuItem>
                            ) : (
                              <DropdownMenuItem onClick={() => setApiTokenDisableDialog({ open: true, memberId: member.id })}>
                                <ShieldOff className="w-4 h-4" />禁用 API Token
                              </DropdownMenuItem>
                            )}
                            {initialAdminIds.has(member.id) ? (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span>
                                    <DropdownMenuItem disabled variant="destructive" onSelect={(e) => e.preventDefault()}>
                                      <Trash2 />删除
                                    </DropdownMenuItem>
                                  </span>
                                </TooltipTrigger>
                                <TooltipContent side="left">管理员账号不可删除</TooltipContent>
                              </Tooltip>
                            ) : (
                              <DropdownMenuItem variant="destructive" onClick={() => openDeleteCheck(member)}>
                                <Trash2 />删除
                              </DropdownMenuItem>
                            )}
                          </DropdownMenuContent>
                        </DropdownMenu>
                      )}
                  </TableActionCell>
                </TableRow>
                );
              })}
            </TableBody>
          </Table>

          {/* 底部翻页
            * 停服态豁免：底部「共 X 名用户 + 翻页控件」属于查看类导航，
            * 需保持 100% 不透明与正常交互（不置灰、不拦截点击）。
            * Pagination 内部对上一页/下一页按钮的 disabled（如首页/末页）
            * 仍由组件自身控制，符合"停服前已禁用则延续禁用"。 */}
          <div
            className="grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[#f0f0f0]"
            data-billing-exempt
          >
            <span className="justify-self-start text-sm leading-[1.5] text-[#737373]">
              共 {filtered.length} 名用户
            </span>
            <Pagination
              total={filtered.length}
              current={currentPage}
              pageSize={PAGE_SIZE}
              size="default"
              className="justify-self-end justify-end flex-nowrap"
              hideOnSinglePage
              onChange={(page) => { setPage(page); }}
            />
          </div>
        </div>
        )}

        {/* 组织视图（v2.0：多层级树 + 健康圆点 + 配置总览 + 导入组织架构） */}
        {viewMode === "group" && (
          <NewGroupView
            hasOneid={hasOneid}
            hasDeptData={hasDeptData}
            users={MM_MOCK_USERS}
            overrides={mmOverrides}
            onResolveConflict={handleMmResolveConflict}
            onDeptSyncedChange={setMmDeptSynced}
            externalAnomalousGroups={mmAnomalousGroups}
            onShowSyncResult={(anomalousGroups) => {
              // 模拟刷新同步：与手动同步保持一致，返回用户异常 + 组织异常
              // 假设 OneID 侧删除了 jack 和 iris（与 handleSync 一致）
              const oneidDeletedUserIds = ["jack@acompany.com", "iris@acompany.com"];
              const vpcBindings: Record<string, string> = {
                "iris@acompany.com": "openclaw/iris",
              };
              const failedUsers: { id: string; clawCount: number; vpcName?: string }[] = [];
              let deletedCount = 0;
              setMembers((prev) => {
                const updated = prev.map((m) => {
                  if (!oneidDeletedUserIds.includes(m.id)) return m;
                  const hasVpc = !!vpcBindings[m.id];
                  if (m.clawCount > 0 || hasVpc) {
                    failedUsers.push({ id: m.id, clawCount: m.clawCount, vpcName: vpcBindings[m.id] });
                    return { ...m, status: "disabled" as const };
                  } else {
                    deletedCount++;
                    return { ...m, _deleted: true } as typeof m & { _deleted: true };
                  }
                });
                return updated.filter((m) => !(m as { _deleted?: boolean })._deleted);
              });
              setSyncResultDialog({
                open: true,
                failedUsers,
                deletedCount,
                addedCount: 0,
                anomalousGroups,
              });
              // 同步异常组织数据到 GroupView 以显示红点
              if (anomalousGroups.length > 0) {
                setMmAnomalousGroups(anomalousGroups);
              }
            }}
          />
        )}

        {/* 项目视图（单层级项目管理，与项目资产页共享 store） */}
        {viewMode === "project" && <ProjectView />}

        {/* 旧组织视图已由 NewGroupView 替代 */}
      </div>

      <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
        <DialogContent
          className="sm:max-w-[720px]"
          style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
        >
          <DialogHeader>
            <DialogTitle>添加用户</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6 flex-1">
            <AddMemberFormFields
              values={newMember}
              onChange={setNewMember}
              existingMemberIds={members.map((m) => m.id)}
              groups={groups}
              userGroups={hasOneid ? MM_MOCK_GROUPS : MM_MOCK_MANUAL_GROUPS}
              onOpenCreateGroupDialog={() => { setShowCreateGroupDialog(true); setNewGroupName(""); setNewGroupParentId(null); }}
              groupPopoverReopenKey={groupPopoverReopenKey}
            />
          </DialogBody>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddDialog(false)}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleAdd}>确认添加</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Member Dialog */}
      <Dialog open={!!editMemberId} onOpenChange={(open) => { if (!open) setEditMemberId(null); }}>
        <DialogContent
          className="sm:max-w-[720px]"
          style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
          onOpenAutoFocus={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>编辑用户</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6 flex-1">
            {hasOneid ? (
              <OneidEditMemberFormFields
                values={oneidEditForm}
                onChange={setOneidEditForm}
                isUnified={isUnified}
                isInitialAdmin={isInitialAdminEdit}
                groups={groups}
                userGroups={MM_MOCK_GROUPS}
                onOpenCreateGroupDialog={() => { setShowCreateGroupDialog(true); setNewGroupName(""); setNewGroupParentId(null); }}
                groupPopoverReopenKey={groupPopoverReopenKey}
              />
            ) : (
              <EditMemberFormFields
                values={editForm}
                onChange={setEditForm}
                isInitialAdmin={isInitialAdminEdit}
                groups={groups}
                userGroups={hasOneid ? MM_MOCK_GROUPS : MM_MOCK_MANUAL_GROUPS}
                onOpenCreateGroupDialog={() => { setShowCreateGroupDialog(true); setNewGroupName(""); setNewGroupParentId(null); }}
                groupPopoverReopenKey={groupPopoverReopenKey}
              />
            )}
          </DialogBody>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditMemberId(null)}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleEdit}>保存修改</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Batch Import Dialog */}
      <Dialog open={showBatchDialog} onOpenChange={(open) => {
        if (!open) {
          setShowBatchDialog(false);
          // 重置状态
          setTimeout(() => {
            setBatchImportStep("upload");
            setBatchImportFile(null);
            setBatchImportProgress(0);
            setBatchImportResult(null);
          }, 300);
        }
      }}>
        <DialogContent className="sm:max-w-[560px]" onInteractOutside={(e) => {
          if (batchImportStep === "importing") e.preventDefault();
        }}>
          <DialogHeader>
            <DialogTitle className="text-[#0A0A0A]">批量导入用户</DialogTitle>
          </DialogHeader>

          <DialogBody className="px-6">
            {/* ── 上传阶段 ── */}
            {batchImportStep === "upload" && (
              <div className="space-y-4">
                {/* Step 1: 下载模板 */}
                <div className="space-y-2">
                  <p className="text-sm font-medium text-[#0A0A0A]">第一步：下载模板并填写用户信息</p>
                  <p className="text-xs text-[#737373] leading-relaxed">
                    下载 CSV 模板，按格式填写信息后保存。
                    <span className="text-[#d42a1e] font-medium">单次最多导入 1000 个用户。</span>
                  </p>
                  <Button variant="outline" size="sm" className="w-full mt-1" onClick={() => {
                    // 生成模板 CSV 并下载
                    const header = "用户邮箱,姓名,角色(admin/member),每日Tokens上限(-1表示无限制)";
                    const example = "user@example.com,张三,member,100000";
                    const blob = new Blob([header + "\n" + example], { type: "text/csv;charset=utf-8;" });
                    const url = URL.createObjectURL(blob);
                    const a = document.createElement("a");
                    a.href = url; a.download = "批量导入用户模板.csv"; a.click();
                    URL.revokeObjectURL(url);
                    toast.success("模板已下载");
                  }}>
                    <Download className="w-4 h-4 mr-2" />
                    下载导入模板
                  </Button>
                </div>

                {/* Step 2: 上传文件 */}
                <div className="space-y-2">
                  <p className="text-sm font-medium text-[#0A0A0A]">第二步：上传填写好的 CSV 文件</p>
                  {!batchImportFile ? (
                    <label className="flex flex-col items-center justify-center w-full h-28 border border-dashed rounded-[4px] cursor-pointer border-[#e5e5e5] hover:border-[#1447E6] hover:bg-[#F5F8FF] transition-colors">
                      <Upload className="w-5 h-5 text-[#737373] mb-1.5" />
                      <span className="text-sm text-[#0A0A0A]">点击选择 CSV 文件</span>
                      <span className="text-xs text-[#737373] mt-0.5">仅支持 .csv 格式</span>
                      <input type="file" accept=".csv" className="hidden"
                        onChange={(e) => {
                          const file = e.target.files?.[0];
                          if (file) setBatchImportFile(file);
                        }} />
                    </label>
                  ) : (
                    <div className="flex items-center justify-between gap-2 px-3 py-3 rounded-[4px] border border-[#E5E5E5] bg-white">
                      <div className="flex items-center gap-2 flex-1 min-w-0">
                        <span className="w-7 h-7 rounded-full bg-[#F5F5F5] flex items-center justify-center shrink-0">
                          <FileText className="w-4 h-4 text-[#525252]" />
                        </span>
                        <div className="flex items-center gap-2 min-w-0">
                          <p className="text-sm font-normal text-[#0A0A0A] truncate">{batchImportFile.name}</p>
                          <span className="text-xs text-[#737373] shrink-0">{(batchImportFile.size / 1024).toFixed(1)} KB</span>
                        </div>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label="移除文件"
                        className="h-7 w-7 p-0 hover:bg-red-50 hover:text-[#d42a1e] shrink-0"
                        onClick={() => setBatchImportFile(null)}
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* ── 导入中阶段 ── */}
            {batchImportStep === "importing" && (
              <div className="py-8 flex flex-col items-center gap-4">
                <div className="relative">
                  <div className="w-16 h-16 rounded-full border-4 border-[#E8ECFE] flex items-center justify-center">
                    <Loader2 className="w-8 h-8 text-[#1447E6] animate-spin" />
                  </div>
                </div>
                <div className="text-center space-y-1.5">
                  <p className="text-base font-semibold text-[#0A0A0A]">正在导入中...</p>
                  <p className="text-sm text-[#737373]">预计需要 1 ~ 2 分钟，请勿关闭弹窗</p>
                  <p className="text-xs text-[#A3A3A3]">导入完成后将自动显示结果通知</p>
                </div>
              </div>
            )}
          </DialogBody>

          <DialogFooter>
            {batchImportStep === "upload" && (
              <>
                <Button variant="outline" onClick={() => {
                  setShowBatchDialog(false);
                  setTimeout(() => {
                    setBatchImportStep("upload");
                    setBatchImportFile(null);
                    setBatchImportProgress(0);
                    setBatchImportResult(null);
                  }, 300);
                }}>取消</Button>
                <Button
                  variant="dialog-confirm"
                  disabled={!batchImportFile}
                  onClick={() => {
                    // 开始导入
                    setBatchImportStep("importing");
                    setBatchImportProgress(0);
                    // 模拟进度：90 秒内从 0 到 95%，最后跳到 100%
                    const totalMs = 90000;
                    const intervalMs = 1000;
                    const steps = totalMs / intervalMs;
                    let current = 0;
                    const timer = setInterval(() => {
                      current += 1;
                      const pct = Math.min(95, Math.round((current / steps) * 95));
                      setBatchImportProgress(pct);
                      if (current >= steps) {
                        clearInterval(timer);
                        setBatchImportProgress(100);
                        setTimeout(() => {
                          const result = { success: 85, fail: 15 };
                          setBatchImportResult(result);
                          setBatchImportStep("upload");
                          setShowBatchDialog(false);
                          setTimeout(() => {
                            setBatchImportStep("upload");
                            setBatchImportFile(null);
                            setBatchImportProgress(0);
                            setBatchImportResult(null);
                          }, 300);
                          // Toast with download link
                          toast.success(
                            `导入完成：成功 ${result.success} 条，失败 ${result.fail} 条`,
                            {
                              duration: 10000,
                              action: {
                                label: "下载详情报告",
                                onClick: () => {
                                  const rows = ["用户邮箱,导入状态,备注"];
                                  for (let i = 1; i <= result.success; i++) {
                                    rows.push(`success_user_${i}@example.com,成功,`);
                                  }
                                  for (let i = 1; i <= result.fail; i++) {
                                    rows.push(`fail_user_${i}@example.com,失败,邮箱格式错误`);
                                  }
                                  const blob = new Blob([rows.join("\n")], { type: "text/csv;charset=utf-8;" });
                                  const url = URL.createObjectURL(blob);
                                  const a = document.createElement("a");
                                  a.href = url; a.download = "导入详情报告.csv"; a.click();
                                  URL.revokeObjectURL(url);
                                },
                              },
                            }
                          );
                        }, 500);
                      }
                    }, intervalMs);
                  }}
                >
                  确认导入
                </Button>
              </>
            )}

          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Reset Password Dialog */}
      <Dialog open={!!showResetDialog} onOpenChange={(open) => { if (!open) { setShowResetDialog(null); setResetForm({ ...emptyResetForm }); setResetNewPwdError(null); setResetConfirmPwdError(null); setShowResetNewPwd(false); setShowResetConfirmPwd(false); } }}>
        <DialogContent
          className="sm:max-w-md"
          style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
          onOpenAutoFocus={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>重置密码</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6 flex-1">
            <div className="space-y-4">
              {isUnified ? (
                <>
                  <p className="text-sm text-[#404040] leading-relaxed">
                    为用户 <span className="font-semibold text-[#0A0A0A]">{showResetDialog}</span> 设置新密码。
                  </p>

                  {/* 新密码 */}
                  <div className="space-y-1.5">
                    <div className="flex items-center gap-1.5">
                      <Label className="text-sm font-medium text-[#404040]">新密码</Label>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="cursor-help inline-flex">
                            <Info className="w-3 h-3 text-[#A3A3A3]" />
                          </span>
                        </TooltipTrigger>
                        <TooltipContent side="top" className="max-w-[240px] whitespace-normal text-xs leading-relaxed">
                          {PASSWORD_RULES_HINT}
                        </TooltipContent>
                      </Tooltip>
                    </div>
                    <div className="relative">
                      <Input
                        type={showResetNewPwd ? "text" : "password"}
                        placeholder="请输入新密码"
                        value={resetForm.newPassword}
                        onChange={(e) => {
                          setResetForm({ ...resetForm, newPassword: e.target.value });
                          if (resetNewPwdError) setResetNewPwdError(null);
                        }}
                        onBlur={() => {
                          if (!resetForm.newPassword) { setResetNewPwdError(null); return; }
                          setResetNewPwdError(validatePasswordStrength(resetForm.newPassword));
                        }}
                        maxLength={PASSWORD_MAX_LENGTH}
                        className={`pr-10 ${resetNewPwdError ? "border-[#d42a1e] focus-visible:ring-[#d42a1e]/30" : ""}`}
                      />
                      <button type="button" onClick={() => setShowResetNewPwd((v) => !v)} className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-[#A3A3A3] hover:text-[#737373] transition-colors" aria-label={showResetNewPwd ? "隐藏密码" : "显示密码"} tabIndex={-1}>
                        {showResetNewPwd ? <Eye className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
                      </button>
                    </div>
                    {resetNewPwdError && <p className="text-xs text-[#d42a1e] leading-tight">{resetNewPwdError}</p>}
                  </div>

                  {/* 确认新密码 */}
                  <div className="space-y-1.5">
                    <Label className="text-sm font-medium text-[#404040]">确认新密码</Label>
                    <div className="relative">
                      <Input
                        type={showResetConfirmPwd ? "text" : "password"}
                        placeholder="请再次输入新密码"
                        value={resetForm.confirmPassword}
                        onChange={(e) => {
                          setResetForm({ ...resetForm, confirmPassword: e.target.value });
                          if (resetConfirmPwdError) setResetConfirmPwdError(null);
                        }}
                        onBlur={() => {
                          if (!resetForm.confirmPassword) { setResetConfirmPwdError(null); return; }
                          if (resetForm.newPassword && resetForm.confirmPassword !== resetForm.newPassword) {
                            setResetConfirmPwdError("两次输入的密码需保持一致");
                          } else {
                            setResetConfirmPwdError(null);
                          }
                        }}
                        maxLength={PASSWORD_MAX_LENGTH}
                        className={`pr-10 ${resetConfirmPwdError ? "border-[#d42a1e] focus-visible:ring-[#d42a1e]/30" : ""}`}
                      />
                      <button type="button" onClick={() => setShowResetConfirmPwd((v) => !v)} className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-[#A3A3A3] hover:text-[#737373] transition-colors" aria-label={showResetConfirmPwd ? "隐藏密码" : "显示密码"} tabIndex={-1}>
                        {showResetConfirmPwd ? <Eye className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
                      </button>
                    </div>
                    {resetConfirmPwdError && <p className="text-xs text-[#d42a1e] leading-tight">{resetConfirmPwdError}</p>}
                  </div>
                </>
              ) : (
                <Alert variant="info">
                  <AlertInfoIcon />
                  <AlertDescription>
                    确认重置用户「<span className="font-medium">{showResetDialog}</span>」的密码？系统将自动生成新密码。
                  </AlertDescription>
                </Alert>
              )}

              {/* 信息发送地址 */}
              <div className="space-y-2">
                <Label className="flex items-center gap-1.5 text-xs font-medium text-[#525252]">
                  信息发送地址（选填）
                  <InfoPopover
                    content="信息发送会产生额外的短信/邮件费用，合并到腾讯云账单计费"
                    placement="top"
                  >
                    <Info className="w-3.5 h-3.5 text-[#737373]" />
                  </InfoPopover>
                </Label>
                <Input
                  type="email"
                  placeholder="输入用户接收新密码的邮箱地址"
                  value={resetForm.notificationEmail}
                  onChange={(e) => setResetForm({ ...resetForm, notificationEmail: e.target.value })}
                  tabIndex={-1}
                />
              </div>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowResetDialog(null); setResetForm({ ...emptyResetForm }); setResetNewPwdError(null); setResetConfirmPwdError(null); setShowResetNewPwd(false); setShowResetConfirmPwd(false); }}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleReset}>
              确认重置
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Credential Result Dialog (创建成功 / 密码已重置) */}
      <CredentialResultDialog
        open={credentialDialog.open}
        onClose={() => setCredentialDialog((d) => ({ ...d, open: false }))}
        title={credentialDialog.title}
        memberId={credentialDialog.memberId}
        password={credentialDialog.password}
      />

      {/* OneID 同步结果弹窗：展示组织异常 + 用户异常 */}
      <Dialog
        open={!!syncResultDialog?.open}
        onOpenChange={(open) => { if (!open) setSyncResultDialog(null); }}
      >
        <DialogContent className="sm:max-w-[920px] max-h-[85vh] overflow-y-auto" onOpenAutoFocus={(e) => e.preventDefault()}>
          <DialogHeader>
            <DialogTitle className="text-lg font-semibold text-[#09090b]">同步结果</DialogTitle>
          </DialogHeader>
          <div className="py-2 space-y-4">

            {/* ═══ 组织异常区块（上方） ═══ */}
            {(syncResultDialog?.anomalousGroups?.length ?? 0) > 0 && (
              <div>
                <h4 className="text-sm font-semibold text-[#09090b] mb-3">组织异常</h4>

                {/* 组织异常提示 */}
                <div className="flex items-start gap-2.5 bg-red-50 border border-red-100 rounded-[4px] px-4 py-3 mb-3">
                  <Info className="w-4 h-4 text-red-400 mt-0.5 shrink-0" />
                  <p className="text-sm text-red-600 leading-relaxed">
                    以下组织对应的部门已在腾讯统一身份管理平台被删除，组织内用户已被移除。但由于组织仍有专属配置未解绑或存量 Agent 实例未删除，需管理员处理完成后，组织才会被彻底删除。专属配置可前往{" "}
                    <button
                      type="button"
                      onClick={() => {
                        setSyncResultDialog(null);
                        setViewMode("group");
                      }}
                      className="inline font-semibold text-red-700 underline underline-offset-2 hover:text-red-800"
                    >
                      用户管理-组织视图
                    </button>
                    {" "}查看并解绑，Agent 实例可前往{" "}
                    <button
                      type="button"
                      onClick={() => {
                        setSyncResultDialog(null);
                        window.location.href = "/admin/openclaw-monitor";
                      }}
                      className="inline font-semibold text-red-700 underline underline-offset-2 hover:text-red-800"
                    >
                      Agent 列表
                    </button>
                    {" "}页删除。
                  </p>
                </div>

                {/* 组织异常表格 */}
                <div className="rounded-[4px] border border-[#e5e5e5] overflow-hidden">
                  <Table density="compact">
                    <TableHeader>
                      <TableRow>
                        <TableHead>组织名称</TableHead>
                        <TableHead className="text-center">组织总人数</TableHead>
                        <TableHead>组织专属配置</TableHead>
                        <TableHead className="text-center">Agent 实例数</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {syncResultDialog?.anomalousGroups?.map((group) => (
                        <TableRow key={group.groupId}>
                          <TableCell className="font-medium">{group.groupName}</TableCell>
                          <TableCell className="text-center text-gray-500 tabular-nums">{group.memberCount}</TableCell>
                          <TableCell>
                            <div className="flex flex-wrap gap-1.5">
                              {group.boundConfigs.map((config) => (
                                <span key={config} className="inline-flex items-center px-2 py-0.5 bg-red-50 text-red-600 rounded-[4px] border border-red-100">
                                  {config}
                                </span>
                              ))}
                            </div>
                          </TableCell>
                          <TableCell className="text-center tabular-nums">
                            <span className={group.agentInstanceCount > 0 ? "font-semibold text-red-600" : "text-gray-400"}>
                              {group.agentInstanceCount > 0 ? `${group.agentInstanceCount} 个` : "—"}
                            </span>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </div>
            )}

            {/* ═══ 用户异常区块（下方） ═══ */}
            {(syncResultDialog?.failedUsers.length ?? 0) > 0 && (
              <div className="space-y-3">
                <h4 className="text-sm font-semibold text-[#09090b]">用户异常</h4>

                {/* Alert 放在所有正文和表格上方 */}
                <Alert variant="warning">
                  <AlertInfoIcon />
                  <AlertDescription>
                    <ul className="space-y-1 list-disc pl-4">
                      <li>删除用户需要该用户名下没有任何 Agent。可让用户自行删除，或由管理员在 Agent 列表页手动删除。</li>
                      {syncResultDialog?.failedUsers.some(u => !!u.vpcName) && (
                        <li>
                          删除用户需要系统自动分配的私有网络下无关联云资源。请前往{" "}
                          <a href="https://console.cloud.tencent.com/vpc" target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-0.5 underline hover:opacity-80">腾讯云控制台<ExternalLink className="w-3 h-3 inline-block" /></a>
                          {" "}解除后，再刷新检查。
                        </li>
                      )}
                    </ul>
                  </AlertDescription>
                </Alert>

                {/* 同步概要正文 */}
                <BodyText as="p" tone="secondary" className="leading-relaxed">
                  本次同步{[
                    (syncResultDialog?.addedCount ?? 0) > 0 ? <React.Fragment key="added">新增用户 <BodyMedium as="span" tone="primary">{syncResultDialog?.addedCount}</BodyMedium> 个</React.Fragment> : null,
                    (syncResultDialog?.failedUsers.length ?? 0) > 0 ? <React.Fragment key="failed">禁用用户 <BodyMedium as="span" tone="danger">{syncResultDialog?.failedUsers.length}</BodyMedium> 个</React.Fragment> : null,
                    (syncResultDialog?.deletedCount ?? 0) > 0 ? <React.Fragment key="deleted">删除用户 <BodyMedium as="span" tone="primary">{syncResultDialog?.deletedCount}</BodyMedium> 个</React.Fragment> : null,
                  ].filter(Boolean).reduce<React.ReactNode[]>((acc, item, i) => {
                    if (i === 0) return [item];
                    return [...acc, <React.Fragment key={`sep-${i}`}>，</React.Fragment>, item];
                  }, [])}。{(syncResultDialog?.failedUsers.length ?? 0) > 0 && (() => {
                    const clawFailCount = syncResultDialog?.failedUsers.filter(u => u.clawCount > 0).length ?? 0;
                    const vpcFailCount = syncResultDialog?.failedUsers.filter(u => !!u.vpcName).length ?? 0;
                    const parts: React.ReactNode[] = [];
                    if (clawFailCount > 0) parts.push(<React.Fragment key="claw">其中 {clawFailCount} 个用户因名下存在未清理的 Agent 无法直接删除</React.Fragment>);
                    if (vpcFailCount > 0) parts.push(<React.Fragment key="vpc">{vpcFailCount} 个用户因名下存在未解除的私有网络无法直接删除</React.Fragment>);
                    return parts.length > 0 ? <>{parts.reduce<React.ReactNode[]>((acc, item, i) => i === 0 ? [item] : [...acc, "，", item], [])}，状态已自动改为禁用。</> : null;
                  })()}
                </BodyText>

                {/* 无法删除的用户列表 */}
                <div className="rounded-[4px] border border-[#e5e5e5] overflow-hidden">
                  <Table density="compact">
                    <TableHeader>
                      <TableRow>
                        <TableHead>用户 ID</TableHead>
                        <TableHead className="text-center">名下 Agent</TableHead>
                        <TableHead>私有网络</TableHead>
                        <TableHead>当前状态</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {syncResultDialog?.failedUsers.map((user) => (
                        <TableRow key={user.id}>
                          <TableCell className="font-medium">{user.id}</TableCell>
                          <TableCell className="text-center font-semibold text-red-600">{user.clawCount} 个</TableCell>
                          <TableCell>
                            {user.vpcName ? (
                              <div className="flex flex-col gap-0.5">
                                <span className="font-medium text-[#355EF1]">{user.vpcName}</span>
                                <span className="text-red-600">(有关联云资源)</span>
                              </div>
                            ) : (
                              <span className="text-gray-400">—</span>
                            )}
                          </TableCell>
                          <TableCell>
                            <StatusTag mode="text" variant="red">禁用</StatusTag>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </div>
            )}

          </div>
          <DialogFooter>
            <Button
              variant="dialog-confirm"
              onClick={() => {
                setSyncResultDialog(null);
                // 同步后检测存量 Agent 实例（弹窗⑦：多用户混合 + 末尾上级组织自动迁移块⑥）
                const allGroups = hasOneid ? MM_MOCK_GROUPS : MM_MOCK_MANUAL_GROUPS;
                const usersList = hasOneid ? MM_MOCK_USERS : MM_MOCK_USERS_MANUAL;
                
                // 三种场景演示：
                // 1. 用户还有组织（alice 被从 og-frontend 移出但仍在其他组织）→ 4 种处理方式
                // 2. 用户之前没有组织、现在有组织（ryan 名下有未分配存量实例，本次同步分配到新组织）
                //    → 现组织有值，移交/允许移交禁用，其余可选
                // 3. 上级组织变动（只显示 ParentMigrationBlock，不需要选择）
                const scenarios: Array<{
                  userId: string;
                  lookupGroupId: string; // 从本地 MOCK_USER_GROUP_AGENTS 取实例用的 key
                  removedGroupIds: string[]; // 传给 mmBuildAffectedGroups 的被移除原组织
                  newGroupIds: string[]; // 现组织（同步后）
                  originalGroupIds: string[]; // 同步前原组织（[] 表示未分配）
                }> = [
                  // 情况1：alice 从 og-frontend 移出，仍保留其他组织
                  (() => {
                    const cur = usersList.find((u) => u.userId === "alice@acompany.com")?.groupIds ?? [];
                    const newIds = cur.filter((g) => g !== "og-frontend");
                    return {
                      userId: "alice@acompany.com",
                      lookupGroupId: "og-frontend",
                      removedGroupIds: ["og-frontend"],
                      newGroupIds: newIds,
                      originalGroupIds: [...newIds, "og-frontend"],
                    };
                  })(),
                  // 情况2：ryan 原本未分配组织（名下有 __global__ 存量实例），本次同步分配到新组织 dept-fe
                  {
                    userId: "ryan@acompany.com",
                    lookupGroupId: "__global__",
                    removedGroupIds: [], // 未从任何真实组织移出 → 靠弹窗③分支（oldGroupIds=[]）触发
                    newGroupIds: ["dept-fe"],
                    originalGroupIds: [],
                  },
                ];
                const affectedUsers: AffectedUser[] = [];
                scenarios.forEach(({ userId, lookupGroupId, removedGroupIds, newGroupIds, originalGroupIds }) => {
                  const instances = MOCK_USER_GROUP_AGENTS[userId]?.[lookupGroupId];
                  if (!instances || instances.length === 0) return;
                  const affectedGroups = mmBuildAffectedGroups(
                    userId,
                    removedGroupIds,
                    newGroupIds,
                    usersList,
                    allGroups,
                    originalGroupIds,
                  );
                  if (affectedGroups.length === 0) return;
                  affectedUsers.push({
                    userId,
                    originalGroups: originalGroupIds.length > 0
                      ? originalGroupIds.map((g) => mmGetPrimaryDeptPath(g, allGroups))
                      : ["未分配组织"],
                    newGroups: newGroupIds.map((g) => mmGetPrimaryDeptPath(g, allGroups)),
                    affectedGroups,
                  });
                });
                
                // 情况3：上级组织变动（只显示自动迁移块，不需要用户选择）
                const parentMigration: ParentMigration = {
                  groups: [
                    {
                      targetPath: "A公司 / 产品部 / 测试组",
                      fromGroupId: "dept-qa",
                      toGroupId: "dept-qa",
                      rows: [
                        { userId: "grace@acompany.com", instanceId: "claw-grace-1", instanceName: "Grace 的测试助手" },
                      ],
                    },
                  ],
                };
                if (affectedUsers.length > 0 || parentMigration.groups.length > 0) {
                  setSyncAgentInstanceDialog({ open: true, affectedUsers, parentMigration });
                }
              }}
            >
              知道了
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Check Dialog - 第一步：资源情况说明 */}
      <Dialog
        open={!!deleteCheckDialog?.open}
        onOpenChange={(open) => { if (!open) setDeleteCheckDialog(null); }}
      >
        <DialogContent
          className="sm:max-w-md"
          style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
          onOpenAutoFocus={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>删除用户</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6 flex-1">
            <div className="space-y-4">
            {/* Alert 提示放在内容区最上方（§8.7 强制规范 5）：根据条件显示蓝色 info 或红色 destructive */}
            {(() => {
              const clawOk = (deleteCheckDialog?.clawCount ?? 0) === 0;
              const vpcOk = deleteCheckDialog?.vpcType === "custom" || deleteCheckDialog?.hasVpcResources === false;
              const allOk = clawOk && vpcOk;

              if (allOk) {
                // 黄色 warning Alert：条件已满足，提醒用户即将执行删除
                return (
                  <Alert variant="warning">
                    <CircleAlert />
                    <AlertTitle>注意事项</AlertTitle>
                    <AlertDescription>
                      {deleteCheckDialog?.vpcType === "auto"
                        ? "该用户名下没有 Agent，且私有网络无关联资源，可以删除。删除后该用户将无法登录系统，操作不可撤销。"
                        : "该用户名下没有 Agent，可以删除。删除后该用户将无法登录系统，操作不可撤销。"
                      }
                    </AlertDescription>
                  </Alert>
                );
              }

              // 红色 destructive Alert：条件未满足
              const reasons: React.ReactNode[] = [];
              if (!clawOk) {
                reasons.push(
                  <p key="claw">
                    删除用户需要该用户名下没有任何 Agent。可让用户自行删除，或由管理员在 Agent 列表页手动删除。
                  </p>
                );
              }
              if (deleteCheckDialog?.vpcType === "auto" && !vpcOk) {
                reasons.push(
                  <p key="vpc">
                    删除用户需要系统自动分配的私有网络下无关联云资源。请前往{" "}
                    <a href="https://console.cloud.tencent.com/vpc" target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-0.5 underline hover:opacity-80">腾讯云控制台<ExternalLink className="w-3 h-3 inline-block" /></a>
                    {" "}解除后，再刷新检查。
                  </p>
                );
              }

              return (
                <Alert variant="warning">
                  <CircleAlert />
                  <AlertTitle>无法删除该用户</AlertTitle>
                  <AlertDescription>
                    {reasons}
                  </AlertDescription>
                </Alert>
              );
            })()}

            {/* 信息卡片：合并三项内容到统一卡片，行间用细分割线 */}
            <div className="rounded-[4px] border border-[#e5e5e5] bg-white divide-y divide-[#F5F5F5]">
              {/* 用户 ID */}
              <div className="px-4 py-3 flex items-center justify-between gap-4">
                <Label className="text-xs font-medium text-[#525252]">用户 ID</Label>
                <span className="text-sm text-[#0A0A0A] truncate">{deleteCheckDialog?.memberId}</span>
              </div>

              {/* 名下 Agent 数量 */}
              <div className="px-4 py-3 flex items-center justify-between gap-4">
                <Label className="text-xs font-medium text-[#525252]">名下 Agent 数量</Label>
                <div className="flex items-center gap-2">
                  {deleteCheckDialog?.clawRefreshing ? (
                    <Loader2 className="w-4 h-4 animate-spin text-[#A3A3A3]" />
                  ) : (
                    <span className={`text-sm ${(deleteCheckDialog?.clawCount ?? 0) > 0 ? "text-[#DC2626] font-medium" : "text-[#0A0A0A]"}`}>
                      {deleteCheckDialog?.clawCount ?? 0} 个
                    </span>
                  )}
                  <button
                    type="button"
                    aria-label="刷新"
                    className="text-[#A3A3A3] hover:text-[#0A0A0A] transition-colors"
                    onClick={() => {
                      if (!deleteCheckDialog) return;
                      setDeleteCheckDialog({ ...deleteCheckDialog, clawRefreshing: true });
                      setTimeout(() => {
                        const newCount = Math.random() > 0.5 ? deleteCheckDialog.clawCount : 0;
                        setDeleteCheckDialog({ ...deleteCheckDialog, clawCount: newCount, clawRefreshing: false });
                      }, 1200);
                    }}
                  >
                    <RefreshCw className="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>

              {/* 私有网络（仅自动分配 VPC 时展示） */}
              {deleteCheckDialog?.vpcType === "auto" && (
                <div className="px-4 py-3 flex items-center justify-between gap-4">
                  <Label className="text-xs font-medium text-[#525252]">私有网络</Label>
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-[#0A0A0A]">
                      <a
                        href="https://console.cloud.tencent.com/vpc"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-[#1447E6] hover:underline"
                      >
                        {deleteCheckDialog?.vpcName}
                      </a>
                      {deleteCheckDialog?.vpcRefreshing ? (
                        <span className="text-[#A3A3A3] ml-1">(检查中...)</span>
                      ) : deleteCheckDialog?.hasVpcResources ? (
                        <span className="text-[#DC2626] ml-1">(有关联云资源)</span>
                      ) : (
                        <span className="text-[#0A0A0A] ml-1">(无关联资源)</span>
                      )}
                    </span>
                    <button
                      type="button"
                      aria-label="刷新"
                      className="text-[#A3A3A3] hover:text-[#0A0A0A] transition-colors"
                      onClick={() => {
                        if (!deleteCheckDialog) return;
                        setDeleteCheckDialog({ ...deleteCheckDialog, vpcRefreshing: true });
                        setTimeout(() => {
                          const newHasResources = Math.random() > 0.5 ? deleteCheckDialog.hasVpcResources : false;
                          setDeleteCheckDialog({ ...deleteCheckDialog, hasVpcResources: newHasResources, vpcRefreshing: false });
                        }, 1200);
                      }}
                    >
                      <RefreshCw className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>
              )}
            </div>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteCheckDialog(null)}>取消</Button>
            {/* 所有条件满足时才显示确认删除按钮 */}
            {(deleteCheckDialog?.clawCount ?? 0) === 0 &&
              (deleteCheckDialog?.vpcType === "custom" || deleteCheckDialog?.hasVpcResources === false) && (
                <Button
                  variant="destructive"
                  onClick={() => {
                    const d = deleteCheckDialog!;
                    setDeleteCheckDialog(null);
                    setDeleteConfirmDialog({ open: true, memberId: d.memberId, vpcType: d.vpcType, vpcName: d.vpcName });
                  }}
                >
                  确认删除
                </Button>
              )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Disable Confirm - 警示弹窗 */}
      <AlertDialog
        open={!!disableConfirmDialog?.open}
        onOpenChange={(open) => { if (!open) setDisableConfirmDialog(null); }}
      >
        <AlertDialogContent className="sm:max-w-[560px]">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[#0A0A0A]">禁用用户</AlertDialogTitle>
          </AlertDialogHeader>
          <div className="space-y-4">
            <Alert variant="warning">
              <CircleAlert />
              <AlertTitle>禁用后将产生以下影响</AlertTitle>
              <AlertDescription>
                <ul className="list-disc pl-4 space-y-0.5">
                  <li>该用户将<span className="font-medium">无法再登录</span>用户端</li>
                  <li>名下所有 Agent 云服务器<span className="font-medium">关机</span>（数据不删除）</li>
                  <li>用户将<span className="font-medium">无法与 Agent 机器人对话</span></li>
                </ul>
              </AlertDescription>
            </Alert>
            <div className="rounded-[4px] border border-gray-200 bg-white divide-y divide-[#F5F5F5]">
              <div className="px-4 py-3 flex items-center justify-between">
                <MetaText tone="muted">用户 ID</MetaText>
                <BodyMedium>{disableConfirmDialog?.memberId}</BodyMedium>
              </div>
              <div className="px-4 py-3 flex items-center justify-between">
                <MetaText tone="muted">名下 Agent 数量</MetaText>
                <BodyMedium>{disableConfirmDialog?.clawCount ?? 0} 个</BodyMedium>
              </div>
            </div>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setDisableConfirmDialog(null)}>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => handleDisable(disableConfirmDialog!.memberId)}
            >
              确认禁用
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Enable Confirm Dialog */}
      <Dialog
        open={!!enableConfirmDialog?.open}
        onOpenChange={(open) => { if (!open) setEnableConfirmDialog(null); }}
      >
        <DialogContent className="sm:max-w-[560px]" onOpenAutoFocus={(e) => e.preventDefault()}>
          <DialogHeader>
            <DialogTitle>启用用户</DialogTitle>
          </DialogHeader>
          <div className="py-2 space-y-4">
            {/* 启用影响说明 — 蓝色 Alert（放在卡片上方） */}
            <Alert variant="info">
              <AlertInfoIcon />
              <AlertTitle>启用后将产生以下影响</AlertTitle>
              <AlertDescription>
                <ul className="space-y-1 list-disc list-inside">
                  <li>该用户可以<span className="font-semibold">继续登录</span>用户端</li>
                  <li>名下所有 Agent 云服务器将<span className="font-semibold">开机</span>，恢复运行</li>
                  <li>用户可以<span className="font-semibold">恢复与 Agent 机器人对话</span></li>
                </ul>
              </AlertDescription>
            </Alert>

            {/* 用户 ID + 名下 Agent 数量 — 合并为一张卡片，内部用分隔线区分 */}
            <div className="rounded-[4px] bg-white border border-[#e5e5e5] divide-y divide-[#e5e5e5]">
              <div className="px-4 py-3 flex items-center justify-between">
                <span className="text-sm text-[#737373]">用户 ID</span>
                <span className="text-sm font-medium text-[#09090b]">{enableConfirmDialog?.memberId}</span>
              </div>
              <div className="px-4 py-3 flex items-center justify-between">
                <span className="text-sm text-[#737373]">名下 Agent 数量</span>
                <span className="text-sm font-semibold text-[#0A0A0A]">{enableConfirmDialog?.clawCount ?? 0} 个</span>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEnableConfirmDialog(null)}>取消</Button>
            <Button
              variant="dialog-confirm"
              onClick={() => handleEnable(enableConfirmDialog!.memberId)}
            >
              确认启用
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* API Token 禁用确认弹窗 */}
      <AlertDialog
        open={!!apiTokenDisableDialog?.open}
        onOpenChange={(open) => { if (!open) setApiTokenDisableDialog(null); }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>禁用 API Token</AlertDialogTitle>
            <AlertDialogDescription>
              禁用后该用户的所有 API Token 将立即失效，API 调用将被拒绝。请确认是否继续？
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => handleDisableApiToken(apiTokenDisableDialog!.memberId)}
              variant="destructive"
            >
              确认禁用
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* API Token 启用确认弹窗 */}
      <AlertDialog
        open={!!apiTokenEnableDialog?.open}
        onOpenChange={(open) => { if (!open) setApiTokenEnableDialog(null); }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>启用 API Token</AlertDialogTitle>
            <AlertDialogDescription>
              启用后该用户可重新使用 API Token 进行接口调用。请确认是否继续？
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => handleEnableApiToken(apiTokenEnableDialog!.memberId)}
              variant="claw-primary"
            >
              确认启用
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete Confirm - 警示弹窗（二次确认 - 列出将删除的资源） */}
      <AlertDialog
        open={!!deleteConfirmDialog?.open}
        onOpenChange={(open) => { if (!open) setDeleteConfirmDialog(null); }}
      >
        <AlertDialogContent className="sm:max-w-[560px]">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[#0A0A0A]">确认删除用户</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <div className="space-y-3">
              <p className="text-sm text-[#0A0A0A]">以下资源将被删除：</p>
              <Alert variant="warning">
                <CircleAlert />
                <AlertTitle>受影响的资源</AlertTitle>
                <AlertDescription>
                  <ul className="list-disc pl-4 space-y-1">
                    <li>用户账号：<span className="font-medium">{deleteConfirmDialog?.memberId}</span></li>
                    {deleteConfirmDialog?.vpcType === "auto" && deleteConfirmDialog?.vpcName && (
                      <li>私有网络：<span className="font-medium">{deleteConfirmDialog.vpcName}</span></li>
                    )}
                  </ul>
                </AlertDescription>
              </Alert>
              <p className="text-sm text-[#DC2626]">删除后无法恢复，请谨慎确认。</p>
            </div>
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setDeleteConfirmDialog(null)}>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => handleDelete(deleteConfirmDialog!.memberId)}
            >
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 存量 Agent 实例处理弹窗（编辑用户组织：弹窗①②③） */}
      <AgentInstanceHandlingDialog
        open={!!agentInstanceDialog?.open}
        onOpenChange={(open) => { if (!open) setAgentInstanceDialog(null); }}
        scenario="editUser"
        affectedUsers={agentInstanceDialog?.affectedUsers ?? []}
        onConfirm={() => {
          agentInstanceDialog?.pendingAction();
          setAgentInstanceDialog(null);
          toast.success("存量实例处理方式已提交");
        }}
        onViewDiff={(fromGroupId, toGroupId, instances) => {
          const allGroups = hasOneid ? MM_MOCK_GROUPS : MM_MOCK_MANUAL_GROUPS;
          const fromName = fromGroupId === "__global__"
            ? "未分配组织（全局配置）"
            : mmGetPrimaryDeptPath(fromGroupId, allGroups);
          const toName = toGroupId === "unassigned"
            ? "未分配组织"
            : (toGroupId ? mmGetPrimaryDeptPath(toGroupId, allGroups) : "");
          setMmConfigDiffDialog({
            open: true,
            fromGroupName: fromName,
            toGroupName: toName,
            instances: mmBuildInstanceCompare(
              (instances ?? []).map((i) => ({ instanceName: i.name, instanceId: i.id }))
            ),
          });
        }}
      />
      {/* 同步后存量 Agent 实例处理弹窗（弹窗⑦：多用户混合 + 上级组织自动迁移块⑥） */}
      <AgentInstanceHandlingDialog
        open={!!syncAgentInstanceDialog?.open}
        onOpenChange={(open) => { if (!open) setSyncAgentInstanceDialog(null); }}
        scenario="oneidSync"
        affectedUsers={syncAgentInstanceDialog?.affectedUsers ?? []}
        parentMigration={syncAgentInstanceDialog?.parentMigration}
        onConfirm={() => {
          setSyncAgentInstanceDialog(null);
          toast.success("存量实例处理方式已提交");
        }}
        onViewDiff={(fromGroupId, toGroupId, instances) => {
          const allGroups = hasOneid ? MM_MOCK_GROUPS : MM_MOCK_MANUAL_GROUPS;
          const fromName = mmGetPrimaryDeptPath(fromGroupId, allGroups);
          const toName = toGroupId === "unassigned"
            ? "未分配组织"
            : (toGroupId ? mmGetPrimaryDeptPath(toGroupId, allGroups) : "");
          setMmConfigDiffDialog({
            open: true,
            fromGroupName: fromName,
            toGroupName: toName,
            instances: mmBuildInstanceCompare(
              (instances ?? []).map((i) => ({ instanceName: i.name, instanceId: i.id }))
            ),
          });
        }}
      />

      {/* 查看配置对比弹窗（弹窗⑧） */}
      <MmConfigDiffDialog
        open={!!mmConfigDiffDialog?.open}
        onOpenChange={(open) => { if (!open) setMmConfigDiffDialog(null); }}
        newGroupName={mmConfigDiffDialog?.toGroupName ?? ""}
        instances={mmConfigDiffDialog?.instances ?? []}
      />

      {/* 新建组织 Dialog */}
      <Dialog open={showCreateGroupDialog} onOpenChange={setShowCreateGroupDialog}>
        <DialogContent
          className="sm:max-w-md"
          style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
          onOpenAutoFocus={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>新建组织</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6 flex-1">
            <div className="space-y-4">
              <div className="space-y-2">
                <Label className="text-xs font-medium text-[#525252]">上级组织</Label>
                <Select value={newGroupParentId ?? "__root__"} onValueChange={(v) => setNewGroupParentId(v === "__root__" ? null : v)}>
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__root__">无（顶层组织）</SelectItem>
                    {(hasOneid ? MM_MOCK_GROUPS : MM_MOCK_MANUAL_GROUPS).map((g) => (
                      <SelectItem key={g.id} value={g.id}>{g.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label className="text-xs font-medium text-[#525252]">
                  组织名称<span className="text-[#DC2626] ml-0.5">*</span>
                </Label>
                <div
                  className="w-full flex items-center h-9 bg-white border border-[#d3d6db] rounded-[4px] focus-within:border-[#355EF1] transition-colors cursor-text overflow-hidden"
                  onClick={(e) => {
                    const inp = (e.currentTarget as HTMLDivElement).querySelector('input');
                    inp?.focus();
                  }}
                >
                  {newGroupParentId && (
                    <span className="pl-3 text-sm text-[#525252] whitespace-nowrap shrink-0 max-w-[40%] truncate select-none pointer-events-none">
                      {(hasOneid ? MM_MOCK_GROUPS : MM_MOCK_MANUAL_GROUPS).find((g) => g.id === newGroupParentId)?.name} /
                    </span>
                  )}
                  <input
                    placeholder="请输入组织名称"
                    value={newGroupName}
                    onChange={(e) => setNewGroupName(e.target.value)}
                    onKeyDown={(e) => { if (e.key === "Enter") handleCreateGroup(); }}
                    className="flex-1 min-w-0 h-full px-3 text-sm bg-transparent outline-none text-[#020617] placeholder:text-[#b0b6c3]"
                    style={{ paddingLeft: newGroupParentId ? "4px" : undefined }}
                    autoFocus
                  />
                </div>
                <p className="text-xs text-[#737373]">组织名称为唯一标识，不能与已有组织重名，创建后支持修改</p>
              </div>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreateGroupDialog(false)}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleCreateGroup}>确认创建</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 删除组织确认 Dialog */}
      <Dialog open={!!deleteGroupDialog?.open} onOpenChange={(open) => { if (!open) setDeleteGroupDialog(null); }}>
        <DialogContent className="sm:max-w-[560px]" onOpenAutoFocus={(e) => e.preventDefault()}>
          <DialogHeader>
            <DialogTitle>删除组织</DialogTitle>
          </DialogHeader>
          {(() => {
            const configs = deleteGroupDialog ? (MOCK_GROUP_CONFIGS[deleteGroupDialog.groupId] || []) : [];
            const hasRelatedConfigs = configs.some((c) => c.items.length > 0);
            return (
              <div className="py-2 space-y-3">
                <div className="rounded-[4px] bg-[#fafafa] border border-[#e5e5e5] px-4 py-3 flex items-center justify-between">
                  <span className="text-sm text-[#737373]">组织名称</span>
                  <span className="text-sm font-medium text-[#09090b]">{deleteGroupDialog?.groupName}</span>
                </div>
                <div className="rounded-[4px] bg-[#fafafa] border border-[#e5e5e5] px-4 py-3 flex items-center justify-between">
                  <span className="text-sm text-[#737373]">组织内用户数</span>
                  <span className="text-sm font-semibold text-[#0A0A0A]">{deleteGroupDialog?.memberCount ?? 0} 人</span>
                </div>

                {/* 已应用配置 */}
                <div className="rounded-[4px] bg-[#fafafa] border border-[#e5e5e5] px-4 py-3">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm text-[#737373]">已应用配置</span>
                    <button
                      className="text-[#A3A3A3] hover:text-[#355EF1] transition-colors"
                      title="刷新"
                      onClick={() => {
                        if (!deleteGroupDialog) return;
                        setDeleteGroupDialog({ ...deleteGroupDialog, configRefreshing: true });
                        setTimeout(() => {
                          setDeleteGroupDialog((prev) => prev ? { ...prev, configRefreshing: false } : null);
                        }, 1200);
                      }}
                    >
                      {deleteGroupDialog?.configRefreshing ? (
                        <Loader2 className="w-3.5 h-3.5 animate-spin" />
                      ) : (
                        <RefreshCw className="w-3.5 h-3.5" />
                      )}
                    </button>
                  </div>
                  {hasRelatedConfigs ? (
                    <div className="flex items-center gap-1.5 flex-wrap">
                      {configs.filter((c) => c.items.length > 0).map((c) => (
                        <StatusTag mode="fill" key={c.type} variant="gray">{c.type}({c.items.length})</StatusTag>
                      ))}
                    </div>
                  ) : (
                    <span className="text-sm text-green-600">无关联配置</span>
                  )}
                </div>

                {/* 状态提示 */}
                {hasRelatedConfigs ? (
                  <div className="rounded-[4px] bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-600 space-y-2">
                    <p className="flex items-start gap-2"><span className="w-1.5 h-1.5 rounded-full bg-red-400 shrink-0 mt-1.5" />以上配置的应用范围包含该组织，请先前往对应配置页面移除该组织后再执行删除。</p>
                    <p className="flex items-start gap-2"><span className="w-1.5 h-1.5 rounded-full bg-red-400 shrink-0 mt-1.5" />删除组织后，组内用户不会被删除，仅解除组织关联。</p>
                  </div>
                ) : (
                  <div className="rounded-[4px] bg-green-50 border border-green-300 px-4 py-3 text-sm text-green-700">
                    该组织无关联配置，可安全删除。删除后组内用户不会被删除，仅解除组织关联。
                  </div>
                )}
              </div>
            );
          })()}
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteGroupDialog(null)}>取消</Button>
            {(() => {
              const configs = deleteGroupDialog ? (MOCK_GROUP_CONFIGS[deleteGroupDialog.groupId] || []) : [];
              const hasRelatedConfigs = configs.some((c) => c.items.length > 0);
              return !hasRelatedConfigs && (
                <Button variant="destructive" onClick={() => deleteGroupDialog && handleDeleteGroup(deleteGroupDialog.groupId)}>
                  确认删除
                </Button>
              );
            })()}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 从组织中移除确认 AlertDialog —— 遵循项目标准警示弹窗规范：
            - 标题黑色，正文黑色，强调字段告警色
            - 警示信息使用 Alert destructive 变体（位于内容区最上方）
            - 主按钮使用 destructive variant
            - 右上角带 X 关闭按钮
       */}
      <AlertDialog open={!!removeFromGroupDialog?.open} onOpenChange={(open) => { if (!open) setRemoveFromGroupDialog(null); }}>
        <AlertDialogContent className="sm:max-w-[560px]">
          <button
            type="button"
            aria-label="关闭"
            onClick={() => setRemoveFromGroupDialog(null)}
            className="absolute top-5 right-5 flex items-center justify-center size-5 rounded-sm text-[#737373] transition-colors hover:text-[#0A0A0A] focus:outline-none"
          >
            <X className="size-5" />
            <span className="sr-only">关闭</span>
          </button>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[#0A0A0A]">从组织中移除</AlertDialogTitle>
          </AlertDialogHeader>

          {/* 警示提示 - Alert 位于内容区最上方 */}
          <Alert variant="warning">
            <CircleAlert />
            <AlertDescription>
              移除后，该用户在此组织下的可见范围和权限将被收回。用户不会被删除，
              <span className="font-medium">仅解除与该组织的关联</span>。
            </AlertDescription>
          </Alert>

          {/* 信息卡 */}
          <div className="rounded-[4px] border border-[#e5e5e5] bg-white px-4 py-3 space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-[#525252]">用户 ID</span>
              <span className="text-sm font-medium text-[#0A0A0A]">{removeFromGroupDialog?.memberId}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-[#525252]">组织名称</span>
              <span className="text-sm font-medium text-[#0A0A0A]">{removeFromGroupDialog?.groupName}</span>
            </div>
          </div>

          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setRemoveFromGroupDialog(null)}>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (removeFromGroupDialog) {
                  handleRemoveFromGroup(removeFromGroupDialog.groupId, removeFromGroupDialog.memberId);
                  setRemoveFromGroupDialog(null);
                }
              }}
            >
              确认移除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 添加用户到组织 Dialog */}
      <Dialog open={showAddToGroupDialog} onOpenChange={(open) => { if (!open) { setShowAddToGroupDialog(false); setAddToGroupSearch(""); setAddToGroupSelected([]); setAddToGroupDeptFilter(""); } }}>
        <DialogContent
          className="sm:max-w-[720px]"
          style={{ height: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
          onOpenAutoFocus={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>添加用户到「{groups.find((g) => g.id === selectedGroupId)?.name || ""}」</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6 flex-1">
            <div className="space-y-4">
              {/* 单组织规则提示 */}
              <Alert variant="info">
                <AlertInfoIcon />
                <AlertDescription>
                  一个用户支持加入多个组织，可按组织设置不同的配置与权限
                </AlertDescription>
              </Alert>

              <div className="flex items-center gap-2">
                {hasOneid && (
                  <TreeSelectFilter
                    nodes={MOCK_DEPARTMENTS}
                    value={addToGroupDeptFilter}
                    onChange={setAddToGroupDeptFilter}
                    allLabel="全部部门"
                    searchPlaceholder="搜索部门"
                  />
                )}
                <div className="relative flex-1">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)]" />
                  <Input
                    placeholder="搜索用户 ID..."
                    value={addToGroupSearch}
                    onChange={(e) => setAddToGroupSearch(e.target.value)}
                    className="pl-9"
                    autoFocus
                  />
                </div>
              </div>

              {(() => {
                const currentGroup = groups.find((g) => g.id === selectedGroupId);
                let searchFiltered = members.filter((m) => m.id.toLowerCase().includes(addToGroupSearch.toLowerCase()));
                // OneID 模式：部门筛选
                if (hasOneid && addToGroupDeptFilter) {
                  const findDeptPath = (nodes: DepartmentNode[], id: string): string | undefined => {
                    for (const n of nodes) {
                      if (n.id === id) return n.path;
                      if (n.children) { const f = findDeptPath(n.children, id); if (f) return f; }
                    }
                    return undefined;
                  };
                  const selectedPath = findDeptPath(MOCK_DEPARTMENTS, addToGroupDeptFilter);
                  if (selectedPath) {
                    searchFiltered = searchFiltered.filter((m) => (MOCK_MEMBER_DEPARTMENTS[m.id] || "").startsWith(selectedPath));
                  }
                }

                return (
                  <div className="border border-[#e5e5e5] rounded-[4px] overflow-hidden">
                    <Table>
                      <TableHeader>
                        <TableRow className="hover:bg-transparent">
                          <TableHead className="w-10"></TableHead>
                          <TableHead>用户 ID</TableHead>
                          {hasOneid && <TableHead>所属组织</TableHead>}
                          <TableHead>当前组织</TableHead>
                          <TableHead className="w-24">角色</TableHead>
                          <TableHead className="w-20">状态</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {searchFiltered.length === 0 ? (
                          <TableRow className="hover:bg-transparent">
                            <TableCell colSpan={hasOneid ? 6 : 5} className="text-center text-xs text-[var(--text-weak)] py-6">
                              没有可添加的用户
                            </TableCell>
                          </TableRow>
                        ) : (
                          searchFiltered.map((m) => {
                            const isInCurrentGroup = currentGroup?.memberIds.includes(m.id) ?? false;
                            const otherGroup = groups.find((g) => g.id !== selectedGroupId && g.memberIds.includes(m.id));
                            const isInOtherGroup = !!otherGroup;
                            const isDisabled = isInCurrentGroup || isInOtherGroup;
                            const memberGroupNames = groups.filter((g) => g.memberIds.includes(m.id)).map((g) => g.name);
                            const groupDisplay = memberGroupNames.length === 0 ? "未分配组织" : memberGroupNames[0];
                            const tooltipText = isInCurrentGroup ? "该用户已在当前组织" : isInOtherGroup ? "该用户已在其他组织" : "";
                            const isChecked = isInCurrentGroup || addToGroupSelected.includes(m.id);
                            const onToggle = () => {
                              if (isDisabled) return;
                              setAddToGroupSelected((prev) =>
                                prev.includes(m.id) ? prev.filter((id) => id !== m.id) : [...prev, m.id]
                              );
                            };
                            const row = (
                              <TableRow
                                key={m.id}
                                data-state={isChecked && !isDisabled ? "selected" : undefined}
                                onClick={onToggle}
                                className={isDisabled ? "opacity-50 cursor-not-allowed bg-[#FAFAFA] hover:bg-[#FAFAFA]" : "cursor-pointer"}
                              >
                                <TableCell className="w-10">
                                  <Checkbox
                                    checked={isChecked}
                                    disabled={isDisabled}
                                    onCheckedChange={onToggle}
                                    onClick={(e) => e.stopPropagation()}
                                  />
                                </TableCell>
                                <TableCell className="text-sm text-[#0A0A0A]">{m.id}</TableCell>
                                {hasOneid && (
                                  <TableCell className="text-xs text-[#737373]">
                                    {MOCK_MEMBER_DEPARTMENTS[m.id] || "-"}
                                  </TableCell>
                                )}
                                <TableCell className="text-xs text-[#737373]">{groupDisplay}</TableCell>
                                <TableCell className="w-24 text-xs text-[#0A0A0A]">
                                  {m.role === "admin" ? "管理员" : "用户"}
                                </TableCell>
                                <TableCell className="w-20">
                                  {m.status === "active" ? (
                                    <StatusTag mode="text" variant="green">正常</StatusTag>
                                  ) : (
                                    <StatusTag mode="text" variant="red">禁用</StatusTag>
                                  )}
                                </TableCell>
                              </TableRow>
                            );
                            return isDisabled ? (
                              <Tooltip key={m.id}>
                                <TooltipTrigger asChild>{row}</TooltipTrigger>
                                <TooltipContent>{tooltipText}</TooltipContent>
                              </Tooltip>
                            ) : row;
                          })
                        )}
                      </TableBody>
                    </Table>
                  </div>
                );
              })()}

              {addToGroupSelected.length > 0 && (
                <div className="flex items-center justify-between">
                  <span className="text-xs text-[#737373]">已选择 {addToGroupSelected.length} 名用户</span>
                  <Button
                    variant="link"
                    size="sm"
                    className="h-auto p-0 text-xs"
                    onClick={() => setAddToGroupSelected([])}
                  >
                    清除筛选
                  </Button>
                </div>
              )}
            </div>
          </DialogBody>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowAddToGroupDialog(false); setAddToGroupSearch(""); setAddToGroupSelected([]); setAddToGroupDeptFilter(""); }}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleAddMembersToGroup} disabled={addToGroupSelected.length === 0}>
              确认添加
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Auth Source Import Dialog（OneID 专用模式下不渲染；普通/统一模式渲染） */}
      {showCustomExtras && (
        <AuthSourceImportDialog
          open={showAuthSourceDialog}
          onOpenChange={(o) => {
            setShowAuthSourceDialog(o);
            if (!o) {
              // 关闭弹窗时重置初始参数
              setAuthSourceInitialStep(undefined);
              setAuthSourceInitialId(null);
              setAuthSourceInitialFormValues(null);
            }
          }}
          initialStep={authSourceInitialStep}
          initialSourceId={authSourceInitialId}
          initialFormValues={authSourceInitialFormValues}
          onComplete={(source) => {
            // 避免重复添加同一数据源
            setConfiguredAuthSources((prev) => {
              const exists = prev.find((s) => s.id === source.id);
              if (exists) {
                return prev.map((s) => s.id === source.id ? source : s);
              }
              return [...prev, source];
            });
          }}
        />
      )}

      {/* Auth Source Delete Confirm Dialog（OneID 专用模式下不渲染；普通/统一模式渲染） */}
      {showCustomExtras && (
        <Dialog
          open={!!deleteAuthSourceConfirm?.open}
          onOpenChange={(open) => { if (!open) setDeleteAuthSourceConfirm(null); }}
        >
        <DialogContent className="sm:max-w-[560px]" onOpenAutoFocus={(e) => e.preventDefault()}>
          <DialogHeader>
            <DialogTitle>删除数据源</DialogTitle>
          </DialogHeader>
          <div className="py-2 space-y-4">
            <Alert variant="warning">
              <AlertInfoIcon />
              <AlertTitle>确定要删除该数据源吗？</AlertTitle>
              <AlertDescription>删除后，通过该数据源同步的用户数据将不再自动更新，已同步的用户不受影响。</AlertDescription>
            </Alert>
            {deleteAuthSourceConfirm?.source && (
              <div className="flex items-center gap-3 rounded-[4px] bg-[#fafafa] border border-[#e5e5e5] px-4 py-3">
                <div className="w-8 h-8 rounded-[4px] bg-white border border-[#e5e5e5] flex items-center justify-center overflow-hidden flex-shrink-0">
                  <img
                    src={deleteAuthSourceConfirm.source.iconUrl}
                    alt={deleteAuthSourceConfirm.source.name}
                    className="w-6 h-6 object-contain"
                  />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-gray-950 leading-snug truncate">{deleteAuthSourceConfirm.source.name}</p>
                  <p className="mt-0.5 text-xs text-gray-500 leading-relaxed">{deleteAuthSourceConfirm.source.description}</p>
                </div>
              </div>
            )}
            <div className="rounded-[4px] bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-600 space-y-1.5">
              <p className="font-medium">确定要删除该数据源吗？</p>
              <p className="text-xs text-[#d42a1e] leading-relaxed">删除后，通过该数据源同步的用户数据将不再自动更新，已同步的用户不受影响。</p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteAuthSourceConfirm(null)}>取消</Button>
            <Button
              variant="destructive"
              onClick={() => {
                if (deleteAuthSourceConfirm?.source) {
                  setConfiguredAuthSources(configuredAuthSources.filter((s) => s.id !== deleteAuthSourceConfirm.source.id));
                  toast.success(`已删除数据源：${deleteAuthSourceConfirm.source.name}`);
                }
                setDeleteAuthSourceConfirm(null);
              }}
            >
              确认删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      )}
    </>
  );
}
