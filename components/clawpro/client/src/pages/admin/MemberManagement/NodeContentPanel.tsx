/**
 * 右侧节点内容面板：用户列表 Tab + 配置总览 Tab
 *
 * 配置总览（v2）：
 *   - 三大核心初始化检查：可见模型 / 可见通道 / 安全组
 *     每项显示：✅ 正常 / ⚠️ 异常；异常时给出「前往对应页配置」跳转
 *   - 按 12 种配置项聚合展示（模型/通道/安全组/技能/Agent工具/记忆/网盘/镜像/VPC/公网/CLS/平台策略）
 *     每条标注来源：本组织 / 继承自某组织 / 平台默认
 *   - 初始化校验仅模型/通道/安全组三项
 */
import React, { useMemo, useState, useRef, useEffect, useCallback } from "react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Pagination } from "@/components/ui/pagination";
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";
import { StatusTag } from "@/components/ui/status-tag";
import { HelperText, MetaText, MetaMedium } from "@/components/ui/Typography";
import {
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell, TableActionCell,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Search,
  ChevronRight,
  ChevronLeft,
  CircleAlert,
  ExternalLink,
  CheckCircle2,
  Lock,
  Plus,
  ChevronDown,
  X,
  Pencil
} from "lucide-react";
import { Link } from "wouter";
import type {
  UserOrg,
  UserOverrideInfo,
  UserGroup,
  ConfigCategory,
  ConfigEntry,
} from "./types";
import { getPrimaryDeptPath, getConfigEntries, CONFIG_CATEGORY_META } from "./mock";
import {
  getGroupHealth,
  getGroupInitHealth,
  MISSING_LABEL,
  INIT_MISSING_LABEL,
  INIT_MISSING_TO_CATEGORY,
  hasNetworkOutdated,
} from "./health";
import { AddUsersToGroupDialog } from "./AddUsersToGroupDialog";
import { RemoveUserFromGroupDialog } from "./RemoveUserFromGroupDialog";

const PAGE_SIZE = 10;

type Tab = "members" | "config";

interface NodeContentPanelProps {
  /** 当前组织 id */
  nodeId: string;
  nodeName: string;
  /** 当前节点所属来源（oneid-dept / oneid-group / manual） */
  nodeSource: "oneid-dept" | "oneid-group" | "manual";
  nodeReadonly: boolean;
  /** 节点所在组织全集（用于祖先继承判定 + 展示主部门路径） */
  groups: UserGroup[];
  /** 本节点路径（面包屑） */
  nodePath: string;
  /** 已过滤到本节点的用户列表 */
  users: UserOrg[];
  /** 保留但当前不在此面板展示（用户列表已精简掉覆盖状态列与查看配置按钮）。
   *  未来需要在此面板恢复冲突裁决入口时可再使用。 */
  overrides?: Record<string, UserOverrideInfo>;
  onResolveConflict?: (userId: string, winnerResourceId: string) => void;
  /** 是否为 OneID 模式 */
  hasOneid?: boolean;
  /**
   * OneID 中是否存在部门数据。仅在 `hasOneid=true` 时生效；为 false 时：
   * 隐藏「部门」列与"部门："信息。默认 true。
   */
  hasDeptData?: boolean;
  /** 是否为普通模式 */
  isManualMode?: boolean;
  /** 全部用户（添加用户到组织弹窗用） */
  allUsers?: UserOrg[];
  /** 添加用户到组织的回调 */
  onAddUsersToGroup?: (userIds: string[]) => void;
  /** 从组织中移除用户的回调 */
  onRemoveFromGroup?: (userId: string) => void;
  /** 编辑用户组织的回调 */
  onEditUserGroups?: (userId: string, groupIds: string[]) => void;
  /** 是否为异常组织（配置未解绑，需显示红点+告警条） */
  isAnomalous?: boolean;
  /** 异常组织绑定的配置名称（用于告警条展示） */
  anomalousBoundConfigs?: string[];
  /** 是否初始化未完成（缺少模型/通道/镜像/网络中的某项，需显示橙色点+黄色告警条） */
  isUninitialized?: boolean;
  /** 是否为项目节点（项目场景：文案用"项目"、配置总览仅展示 agent 工具） */
  isProject?: boolean;
}

// ─── 核心维度 meta（初始化检查卡用） ─────────────────────
const CORE_CHECK_META = {
  model: {
    label: "配置至少一个可见模型",
    path: "/admin/model-config",
    desc: "当前缺失，建议前往配置",
  },
  channel: {
    label: "配置至少一个可见通道",
    path: "/admin/channel-config",
    desc: "当前缺失，建议前往配置",
  },
  securityGroup: {
    label: "配置安全组",
    path: "/admin/security-group",
    desc: "当前缺失，建议前往配置",
  },
} as const;

// ─── 配置总览卡片图标（使用场景命名，对应 public/assets/admin-member-config-overview） ─────
const CONFIG_OVERVIEW_ICON_BASE = "/assets/admin-member-config-overview";
const CATEGORY_ICON_SRC: Record<ConfigCategory, string> = {
  model: `${CONFIG_OVERVIEW_ICON_BASE}/model.svg`,
  channel: `${CONFIG_OVERVIEW_ICON_BASE}/channel.svg`,
  skill: `${CONFIG_OVERVIEW_ICON_BASE}/skill.svg`,
  agentTool: `${CONFIG_OVERVIEW_ICON_BASE}/agent-tool.svg`,
  memory: `${CONFIG_OVERVIEW_ICON_BASE}/memory.svg`,
  drive: `${CONFIG_OVERVIEW_ICON_BASE}/file-drive.svg`,
  image: `${CONFIG_OVERVIEW_ICON_BASE}/image.svg`,
  cloudResource: `${CONFIG_OVERVIEW_ICON_BASE}/resource-config.svg`,
  network: `${CONFIG_OVERVIEW_ICON_BASE}/network.svg`,
  cls: `${CONFIG_OVERVIEW_ICON_BASE}/cls-log.svg`,
  aiAgentSecurity: `${CONFIG_OVERVIEW_ICON_BASE}/ai-agent-security.svg`,
  platformPolicy: `${CONFIG_OVERVIEW_ICON_BASE}/platform-policy.svg`,
  cloudDev: `${CONFIG_OVERVIEW_ICON_BASE}/cloud-dev.svg`,
};

// 配置项展示顺序
const CATEGORY_ORDER: ConfigCategory[] = [
  "model", "channel", "skill", "agentTool", "memory",
  "drive", "image", "cloudResource", "network",
  "cls", "aiAgentSecurity", "platformPolicy",
  "cloudDev",
];

// 配置项导航短名称
const CATEGORY_NAV_LABEL: Record<ConfigCategory, string> = {
  model: "模型",
  channel: "通道",
  skill: "技能",
  agentTool: "工具",
  memory: "记忆",
  drive: "网盘",
  image: "镜像",
  cloudResource: "资源",
  network: "网络",
  cls: "日志",
  aiAgentSecurity: "安全",
  platformPolicy: "策略",
  cloudDev: "云开发",
};

// ─── 组织标签选择器（简化版，用于添加/编辑用户弹窗） ───────
function GroupTagSelect({
  groups,
  selectedIds,
  onChange,
}: {
  groups: UserGroup[];
  selectedIds: string[];
  onChange: (ids: string[]) => void;
}) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");

  // 只显示 manual 组织（普通模式）
  const manualGroups = useMemo(
    () => groups.filter((g) => g.source === "manual"),
    [groups]
  );
  const groupMap = useMemo(
    () => new Map(manualGroups.map((g) => [g.id, g])),
    [manualGroups]
  );

  // 获取组织全路径
  const getPath = (gId: string): string => {
    const chain: string[] = [];
    let node = groupMap.get(gId);
    while (node) {
      chain.unshift(node.name);
      node = node.parentId ? groupMap.get(node.parentId) : undefined;
    }
    return chain.join(" / ");
  };

  // 搜索过滤
  const filtered = useMemo(() => {
    if (!search.trim()) return manualGroups;
    const q = search.trim().toLowerCase();
    return manualGroups.filter(
      (g) => g.name.toLowerCase().includes(q) || getPath(g.id).toLowerCase().includes(q)
    );
  }, [manualGroups, search]);

  const toggleGroup = (id: string) => {
    if (selectedIds.includes(id)) {
      onChange(selectedIds.filter((x) => x !== id));
    } else {
      onChange([...selectedIds, id]);
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div className="relative w-full min-h-[36px] px-2 py-1.5 rounded-xl border border-[#e5e5e5] bg-white hover:border-[#1447E6] transition-colors cursor-pointer flex items-center flex-wrap gap-1 pr-7">
          {selectedIds.length === 0 ? (
            <span className="text-xs text-[#A3A3A3] px-1">选择组织…</span>
          ) : (
            selectedIds.map((id) => (
              <span
                key={id}
                className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-[#E8ECFE] text-[#1447E6] text-[11px] max-w-full"
              >
                <span className="truncate">{getPath(id)}</span>
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    onChange(selectedIds.filter((x) => x !== id));
                  }}
                  className="text-[#1447E6]/60 hover:text-[#1447E6] shrink-0"
                >
                  <X className="w-3 h-3" />
                </button>
              </span>
            ))
          )}
          {selectedIds.length > 0 && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onChange([]);
              }}
              className="absolute right-1.5 top-1/2 -translate-y-1/2 w-4 h-4 rounded-full bg-[#A3A3A3] hover:bg-[#737373] flex items-center justify-center shrink-0"
              title="清空"
            >
              <X className="w-2.5 h-2.5 text-white" />
            </button>
          )}
        </div>
      </PopoverTrigger>
      <PopoverContent
        className="p-0"
        style={{ width: "var(--radix-popover-trigger-width)" }}
        align="start"
        sideOffset={4}
      >
        <div className="p-2.5 border-b border-[#e5e5e5]">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[#A3A3A3]" />
            <input
              type="text"
              placeholder="搜索组织…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-8 pr-7 py-1.5 text-xs border border-[#e5e5e5] rounded-xl bg-[#fafafa] outline-none focus:border-[#1447E6] focus:ring-1 focus:ring-[#BFCFFE] transition-colors"
            />
            {search && (
              <button onClick={() => setSearch("")} className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[#A3A3A3] hover:text-[#525252]">
                <X className="w-3 h-3" />
              </button>
            )}
          </div>
        </div>
        <div className="max-h-[240px] overflow-y-auto p-1.5">
          {filtered.length === 0 ? (
            <div className="text-center py-4"><HelperText>暂无组织</HelperText></div>
          ) : (
            filtered.map((g) => {
              const isSelected = selectedIds.includes(g.id);
              return (
                <button
                  key={g.id}
                  type="button"
                  onClick={() => toggleGroup(g.id)}
                  className={`w-full flex items-center gap-2 px-2.5 py-1.5 rounded-xl text-left text-xs transition-colors ${
                    isSelected ? "bg-[#E8ECFE] text-[#1447E6]" : "hover:bg-[var(--bg-grey-hover)] text-[#404040]"
                  }`}
                >
                  <span
                    className={`w-3.5 h-3.5 rounded border shrink-0 flex items-center justify-center transition-colors ${
                      isSelected ? "bg-[#355EF1] border-[#1447E6]" : "border-[#C8CFDA] bg-white"
                    }`}
                  >
                    {isSelected && (
                      <svg className="w-2.5 h-2.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                      </svg>
                    )}
                  </span>
                  <span className="truncate">{getPath(g.id)}</span>
                </button>
              );
            })
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}

// ─── 获取用户所有 oneid-dept 类型部门的完整路径 ────────────
function getUserDeptPaths(
  user: UserOrg,
  groups: UserGroup[]
): Array<{ path: string; isPrimary: boolean }> {
  const deptGroupIds = user.groupIds.filter((gid) => {
    const g = groups.find((g) => g.id === gid);
    return g?.source === "oneid-dept";
  });
  if (deptGroupIds.length === 0) return [];
  return deptGroupIds.map((gid) => ({
    path: getPrimaryDeptPath(gid, groups),
    isPrimary: gid === user.primaryGroupId,
  })).sort((a, b) => (a.isPrimary ? -1 : b.isPrimary ? 1 : 0));
}

export default function NodeContentPanel({
  nodeId,
  nodeName,
  nodeSource,
  nodeReadonly,
  groups,
  nodePath,
  users,
  hasOneid = false,
  hasDeptData = true,
  isManualMode = false,
  allUsers = [],
  onAddUsersToGroup,
  onRemoveFromGroup,
  onEditUserGroups,
  isAnomalous = false,
  anomalousBoundConfigs = [],
  isUninitialized = false,
  isProject = false,
}: NodeContentPanelProps) {
  // 是否显示「部门」相关 UI（OneID 模式 + 存在部门数据）
  const showDept = hasOneid && hasDeptData;
  // 节点称谓：项目场景用"项目"，否则"组织"
  const term = isProject ? "项目" : "组织";
  const [tab, setTab] = useState<Tab>(isAnomalous ? "config" : "members");
  const [page, setPage] = useState(1);

  // 用户列表表格横向滚动检测（操作列 sticky right 阴影）
  const ncpTableScrollRef = useRef<HTMLDivElement>(null);
  const [ncpTableCanScrollRight, setNcpTableCanScrollRight] = useState(false);
  useEffect(() => {
    const el = ncpTableScrollRef.current;
    if (!el) return;
    const check = () => {
      const canScroll = el.scrollWidth > el.clientWidth;
      const isAtRight = el.scrollLeft + el.clientWidth >= el.scrollWidth - 1;
      setNcpTableCanScrollRight(canScroll && !isAtRight);
    };
    check();
    el.addEventListener("scroll", check, { passive: true });
    const ro = new ResizeObserver(check);
    ro.observe(el);
    return () => { el.removeEventListener("scroll", check); ro.disconnect(); };
  }, []);

  // 切换节点时根据是否异常重置默认 tab
  useEffect(() => {
    setTab(isAnomalous ? "config" : "members");
    setPage(1);
  }, [nodeId, isAnomalous]);

  /**
   * 网络配置待更新（VPC 整删 / 某可用区所有子网均被删除）
   * 用于「配置总览」Tab 旁的红色小圆点提示。
   * 与各资源行的逐项「配置已失效，请及时更新」保持同源判定。
   */
  const networkOutdatedForTab = useMemo(
    () => hasNetworkOutdated(nodeId, groups),
    [nodeId, groups]
  );

  // 添加用户到组织弹窗
  const [showAddDialog, setShowAddDialog] = useState(false);

  // 编辑用户组织弹窗
  const [editUserDialog, setEditUserDialog] = useState<{ userId: string; displayName: string; groupIds: string[] } | null>(null);
  const [editGroupIds, setEditGroupIds] = useState<string[]>([]);

  // 从组织 / 项目中移除确认弹窗（存量 Agent 实例二次处理逻辑封装在共享组件内部）
  const [removeUserId, setRemoveUserId] = useState<string | null>(null);

  React.useEffect(() => {
    setPage(1);
  }, [nodeId]);

  const total = users.length;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const currentPage = Math.min(page, totalPages);
  const pagedUsers = users.slice(
    (currentPage - 1) * PAGE_SIZE,
    currentPage * PAGE_SIZE
  );

  const health = getGroupHealth(nodeId, groups);

  // 节点来源文案
  const sourceLabel =
    nodeSource === "oneid-dept"
      ? "OneID 部门节点"
      : nodeSource === "oneid-group"
      ? "OneID 用户组"
      : "自建组织";

  const groupMap = new Map(groups.map((g) => [g.id, g]));
  const groupName = (id: string) => groupMap.get(id)?.name ?? id;

  return (
    <div className="flex flex-col h-full">
      {/* 节点头：名称 + 人数 + 组织名称路径 + 添加按钮 */}
      <div className="flex items-center justify-between px-6 pt-5 pb-3 border-b border-[#e5e5e5]">
        <div>
          <div className="flex items-center gap-3 mb-1 flex-wrap">
            <h2 className="text-lg font-semibold text-[var(--text-title)]">{nodeName}</h2>
            <span className="text-sm text-[var(--text-muted)] tabular-nums">
              · {users.length} 人
            </span>
          </div>
          <div className="text-xs text-[var(--text-muted)]">{term}名称：{nodePath}</div>
        </div>
        {nodeId !== "__unassigned__" && (isManualMode || nodeSource !== "oneid-dept") && (() => {
          const totalUserCount = allUsers?.length ?? 0;
          const isAtLimit = totalUserCount >= 20;
          return (
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="shrink-0">
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-8 text-xs gap-1.5 border-[#e5e5e5] shrink-0"
                    onClick={() => setShowAddDialog(true)}
                    disabled={isAtLimit}
                  >
                    <Plus className="w-3.5 h-3.5" />
                    添加用户到{term}
                  </Button>
                </span>
              </TooltipTrigger>
              {isAtLimit && (
                <TooltipContent className="text-xs">
                  已达用户人数上限（{totalUserCount}/{20}）
                </TooltipContent>
              )}
            </Tooltip>
          );
        })()}
      </div>

      {/* Tab 切换
        * 停服态豁免：切换「用户列表 / 配置总览」属于查看类操作（不产生变更），
        * 与「全部/组织/项目」视图切换同档，需保持 100% 不透明与正常交互。
        * SegmentOption 自身未设置 disabled，"停服前已禁用则延续禁用"约束
        * 通过组件 disabled 属性依然生效（此处无）。 */}
      <div className="px-6 pt-3">
        <SegmentGroup data-billing-exempt>
          <SegmentOption active={tab === "members"} onClick={() => setTab("members")}>
            用户列表
          </SegmentOption>
          <SegmentOption active={tab === "config"} onClick={() => setTab("config")} className="relative">
            配置总览
            {isAnomalous && (
              <span className="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-red-500" />
            )}
            {!isAnomalous && (isUninitialized || networkOutdatedForTab) && (
              <span className="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-[var(--alert-warning-icon)]" />
            )}
          </SegmentOption>
        </SegmentGroup>
      </div>

      {/* Tab 内容 */}
      <div className={`flex-1 overflow-y-auto px-6 pb-4 ${tab === "config" ? "pt-0" : "pt-4"}`}>
        {tab === "members" && (
          <>
            {/* 卡片 */}
            <div
              className="bg-white rounded-xl border border-[#e5e5e5] overflow-hidden"
            >
              {/* 表格 */}
              <div className="overflow-x-auto" style={{ width: 0, minWidth: "100%" }} ref={ncpTableScrollRef}>
                <Table style={{ width: "max-content", minWidth: "100%" }} autoFixedColumns={false}>
                  <TableHeader>
                    <TableRow>
                      <TableHead style={{ minWidth: "160px" }}>
                        用户 ID
                      </TableHead>
                      <TableHead style={{ minWidth: "180px" }}>
                        {term}
                      </TableHead>
                      <TableHead>
                        角色
                      </TableHead>
                      <TableHead>
                        状态
                      </TableHead>
                      {isManualMode && nodeId !== "__unassigned__" && (
                        <TableHead>
                          操作
                        </TableHead>
                      )}
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {pagedUsers.length === 0 ? (
                      <TableRow>
                        <TableCell
                          colSpan={
                            4 +
                            (isManualMode && nodeId !== "__unassigned__" ? 1 : 0)
                          }
                          className="px-6 py-12 text-center text-[var(--text-weak)]"
                        >
                          暂无用户
                        </TableCell>
                      </TableRow>
                    ) : (
                      pagedUsers.map((u) => {
                        const userGroups = u.groupIds
                          .map((gid) => groupMap.get(gid))
                          .filter(Boolean) as UserGroup[];
                        const manualGroups = userGroups.filter((g) => g.source === "manual");
                        return (
                          <TableRow
                            key={u.userId}
                          >
                            {/* 用户 ID */}
                            <TableCell>
                              <span className="font-medium text-[var(--text-emphasis)]">
                                {u.userId}
                              </span>
                            </TableCell>

                            {/* 组织 */}
                            <TableCell>
                              {(() => {
                                // 项目场景：只显示项目（source='project'）
                                // OneID 模式：显示部门 + 用户组；普通模式：只显示自建组织
                                // 当 OneID 中无部门数据时（!showDept），组织列只显示 oneid-group
                                const rawDisplayGroups = isProject
                                  ? userGroups.filter((g) => g.source === "project")
                                  : hasOneid
                                  ? showDept
                                    ? userGroups.filter((g) => g.source === "oneid-dept" || g.source === "oneid-group")
                                    : userGroups.filter((g) => g.source === "oneid-group")
                                  : manualGroups;
                                if (rawDisplayGroups.length === 0)
                                  return <span className="text-[var(--text-weak)]">—</span>;

                                // 主条目（主部门/主组织）置顶
                                const displayGroups = [...rawDisplayGroups].sort((a, b) => {
                                  const aPrim = a.id === u.primaryGroupId ? 1 : 0;
                                  const bPrim = b.id === u.primaryGroupId ? 1 : 0;
                                  return bPrim - aPrim;
                                });

                                // 统一使用完整路径（OneID 模式：部门/用户组；普通模式：自建组织层级）
                                const getDisplayName = (g: UserGroup) =>
                                  getPrimaryDeptPath(g.id, groups);

                                const firstName = getDisplayName(displayGroups[0]);

                                return (
                                  <div className="flex items-center gap-1 max-w-[180px]">
                                    <Tooltip>
                                      <TooltipTrigger asChild>
                                        <span className="inline-flex items-center gap-1 cursor-default max-w-full text-[var(--text-secondary)]">
                                          <span className="truncate max-w-[200px]">
                                            {firstName}
                                          </span>
                                          {displayGroups.length > 1 && (
                                            <span className="shrink-0 tabular-nums text-[var(--text-muted)]">
                                              +{displayGroups.length - 1}
                                            </span>
                                          )}
                                        </span>
                                      </TooltipTrigger>
                                      <TooltipContent side="bottom" align="start" className="max-w-[380px] p-0">
                                        <div className="py-2">
                                          {displayGroups.map((g, idx) => {
                                            const isPrimary = g.id === u.primaryGroupId;
                                            return (
                                              <div key={g.id} className="px-3 py-1.5 text-sm flex items-center">
                                                <span className="text-[var(--text-muted)] mr-1 tabular-nums shrink-0">{idx + 1}.</span>
                                                <span className="text-white break-all">{getDisplayName(g)}</span>
                                                {isPrimary && (
                                                  <span className="ml-2 inline-flex items-center text-[10px] font-medium text-[#355EF1] bg-[#EEF2FF] rounded px-1.5 py-0.5 shrink-0">
                                                    主组织
                                                  </span>
                                                )}
                                              </div>
                                            );
                                          })}
                                        </div>
                                      </TooltipContent>
                                    </Tooltip>
                                  </div>
                                );
                              })()}
                            </TableCell>

                            {/* 角色 */}
                            <TableCell>
                              <span className="text-[var(--text-body)]">
                                {u.role === "admin" ? "管理员" : "用户"}
                              </span>
                            </TableCell>

                            {/* 状态 */}
                            <TableCell>
                              {u.status === "active" ? (
                                <StatusTag mode="text" variant="green">正常</StatusTag>
                              ) : (
                                <StatusTag mode="text" variant="red">禁用</StatusTag>
                              )}
                            </TableCell>

                            {/* 操作（仅普通模式且非未分配组织） */}
                            {isManualMode && nodeId !== "__unassigned__" && (
                              <TableActionCell actionsClassName="justify-start">
                                <Button
                                  variant="link"
                                  onClick={() => setRemoveUserId(u.userId)}
                                >
                                  移除
                                </Button>
                              </TableActionCell>
                            )}
                          </TableRow>
                        );
                      })
                    )}
                  </TableBody>
                </Table>
              </div>

              {/* 底部：共 N 名用户 + 分页 */}
              <div className="grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[#e5e5e5]">
                <span className="justify-self-start text-xs leading-[18px] text-[var(--text-muted)]">
                  共 {total} 名用户
                </span>
                <Pagination
                  total={total}
                  current={page}
                  pageSize={PAGE_SIZE}
                  className="justify-self-end justify-end flex-nowrap"
                  hideOnSinglePage
                  onChange={(p) => { setPage(() => p); }}
                />
              </div>
            </div>
          </>
        )}

        {tab === "config" && (
          <ConfigOverviewTab
            nodeId={nodeId}
            groups={groups}
            health={health}
            isAnomalous={isAnomalous}
            anomalousBoundConfigs={anomalousBoundConfigs}
            isUninitialized={isUninitialized}
            isProject={isProject}
          />
        )}
      </div>

      {/* 添加用户到组织弹窗（普通模式） */}
      <AddUsersToGroupDialog
        open={showAddDialog}
        onOpenChange={setShowAddDialog}
        nodeName={nodeName}
        nodeId={nodeId}
        allUsers={allUsers}
        groups={groups}
        showDept={showDept}
        hasOneid={hasOneid}
        term={term}
        onConfirm={(userIds) => onAddUsersToGroup?.(userIds)}
      />

      {/* 从组织 / 项目中移除确认弹窗（含存量 Agent 实例处理，与资产管理页共享同一组件） */}
      <RemoveUserFromGroupDialog
        userId={removeUserId}
        nodeId={nodeId}
        nodeName={nodeName}
        groups={groups}
        isProject={isProject}
        onClose={() => setRemoveUserId(null)}
        onConfirm={(userId) => onRemoveFromGroup?.(userId)}
      />

      {/* 编辑用户组织弹窗 */}
      <Dialog
        open={!!editUserDialog}
        onOpenChange={(open) => {
          if (!open) {
            setEditUserDialog(null);
            setEditGroupIds([]);
          }
        }}
      >
        <DialogContent
          className="sm:max-w-[560px]"
          onOpenAutoFocus={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>编辑用户组织</DialogTitle>
          </DialogHeader>
          <div className="py-2 space-y-4">
            <div className="rounded-xl bg-[#fafafa] border border-[#e5e5e5] px-4 py-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-[#737373]">用户 ID</span>
                <span className="text-sm font-medium text-[#0A0A0A]">
                  {editUserDialog?.userId}
                </span>
              </div>
            </div>
            <div className="space-y-2">
              <MetaMedium as="label" tone="secondary">用户组织</MetaMedium>
              <GroupTagSelect
                groups={groups}
                selectedIds={editGroupIds}
                onChange={setEditGroupIds}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setEditUserDialog(null); setEditGroupIds([]); }}>
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              onClick={() => {
                if (editUserDialog) {
                  onEditUserGroups?.(editUserDialog.userId, editGroupIds);
                }
                setEditUserDialog(null);
                setEditGroupIds([]);
              }}
              disabled={editGroupIds.length === 0}
            >
              确认修改
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ─── 配置总览 Tab ───────────────────────────────────────
interface ConfigOverviewTabProps {
  nodeId: string;
  groups: UserGroup[];
  health: { healthy: boolean; missing: Array<"model" | "channel" | "securityGroup"> };
  isAnomalous?: boolean;
  anomalousBoundConfigs?: string[];
  /** 是否初始化未完成（缺少模型/通道/镜像/网络中的某项） */
  isUninitialized?: boolean;
  /** 是否为项目节点：仅展示 agent 工具模块，且隐藏顶部锚点导航条 */
  isProject?: boolean;
}

/** 来源标签 */
function SourceBadge({ source, isProject = false }: { source: ConfigEntry["source"]; isProject?: boolean }) {
  if (source.type === "local") {
    return <Badge color="blue">{isProject ? "本项目" : "本组织"}</Badge>;
  }
  if (source.type === "platformDefault") {
    return <Badge variant="secondary">全部用户</Badge>;
  }
  if (source.type === "presetPolicy") {
    return <Badge variant="secondary">预设策略</Badge>;
  }
  // inherited
  return <Badge variant="secondary">继承自 {source.groupName}</Badge>;
}

/** 异常组织：本组织配置条目后的红色提示标签 */
function LocalAnomalyHint() {
  return (
    <StatusTag mode="dot" variant="red" className="text-xs shrink-0">
      请前往对应配置页解绑或删除
    </StatusTag>
  );
}

/**
 * VPC / 子网被云端删除时的「请前往网络管理页面更新配置」轻量提示。
 * 黄色（amber）配色，用于表达「需要前往更新」的待处理状态，与红色异常胶囊形成色阶区分。
 * 注：StatusTag 当前未提供 yellow 变体，沿用 Badge 自定义样式（amber tokens 与 Alert variant="warning" 对齐）。
 */
function ConfigOutdatedHint() {
  return (
    <Badge
      variant="outline"
      className="gap-1 border-[#FED7AA] bg-[#FFF7ED] text-[#FF6900] shrink-0"
    >
      <span className="w-1 h-1 rounded-full bg-[#FF6900]" />
      请前往网络管理更新配置
    </Badge>
  );
}

/** 公网配置项的特殊展示（三项信息 + 来源标签跟在后面） */
function PublicNetworkDetail({ meta, source }: { meta: Record<string, string | number | boolean>; source: ConfigEntry["source"] }) {
  return (
    <div className="flex items-center gap-3 text-xs text-[var(--text-muted)] flex-wrap">
      <span>
        公网 IP：
        <StatusTag mode="text" variant={meta.allocated ? "green" : "gray"} className="ml-1 text-xs">
          {meta.allocated ? "已分配" : "未分配"}
        </StatusTag>
      </span>
      <span className="text-[var(--text-weak)]">|</span>
      <span>计费模式：<span className="font-medium text-[var(--text-emphasis)]">{String(meta.billingMode)}</span></span>
      <span className="text-[var(--text-weak)]">|</span>
      <span>带宽上限：<span className="font-medium text-[var(--text-emphasis)] tabular-nums">{String(meta.bandwidthCap)} Mbps</span></span>
      <SourceBadge source={source} />
    </div>
  );
}

/** 平台策略条目的特殊展示 */
function PolicyEntryValue({ entry }: { entry: ConfigEntry }) {
  if (!entry.meta) return null;
  if ("enabled" in entry.meta) {
    return (
      <StatusTag mode="dot" variant={entry.meta.enabled ? "green" : "gray"} className="text-xs">
        {entry.meta.enabled ? "已开启" : "已关闭"}
      </StatusTag>
    );
  }
  if ("value" in entry.meta) {
    const val = entry.meta.value as number;
    return (
      <span className="text-xs font-medium text-[var(--text-emphasis)] tabular-nums">
        {val}
      </span>
    );
  }
  return null;
}

/** 云资源策略摘要（参照网络模块「私有网络与子网」分组标题 + 内容样式） */
function CloudResourceDetail({ entry }: { entry: ConfigEntry }) {
  const meta = entry.meta as Record<string, unknown> | undefined;
  if (!meta) return null;

  const policyName = String(meta.policyName ?? "");
  const billingModeLabel = meta.billingMode === "payAsYouGo" ? "按量计费" : "包年包月";
  const instanceSpec = String(meta.instanceSpec ?? "");
  const systemDiskType = String(meta.systemDiskType ?? "");
  const systemDiskSize = String(meta.systemDiskSize ?? "");
  const assignPublicIp = Boolean(meta.assignPublicIp);
  const bandwidthBillingModeLabel = meta.bandwidthBillingMode === "traffic" ? "按流量计费" : "包月带宽";
  const bandwidthLimit = meta.bandwidthLimit ? String(meta.bandwidthLimit) : "";

  // 行内用 ｜ 连接的辅助渲染
  const Sep = () => <span className="text-[var(--text-weak)] mx-1">｜</span>;
  const Label = ({ children }: { children: React.ReactNode }) => (
    <span className="text-[var(--text-muted)]">{children}</span>
  );
  const Value = ({ children }: { children: React.ReactNode }) => (
    <span className="font-medium text-[var(--text-emphasis)]">{children}</span>
  );

  return (
    <div>
      {/* 分组标题：参照网络「私有网络与子网」灰底栏 */}
      <div className="px-6 py-2 bg-[#fafafa]/80 border-b border-[#f5f5f5] flex items-center gap-2">
        <MetaText>资源配置策略</MetaText>
      </div>
      {/* 策略内容 */}
      <div className="px-6 py-3 space-y-2.5">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs font-medium text-[var(--text-emphasis)]">{policyName}</span>
          <SourceBadge source={entry.source} />
        </div>
        {/* 第 1 行：计费 / 规格 / 系统盘 */}
        <div className="text-xs leading-relaxed flex items-center flex-wrap gap-x-0">
          <Label>计费模式：</Label>
          <Value>{billingModeLabel}</Value>
          <Sep />
          <Label>实例规格：</Label>
          <Value>{instanceSpec}</Value>
          <Sep />
          <Label>系统盘类型：</Label>
          <Value>{systemDiskType}</Value>
          <Sep />
          <Label>系统盘容量：</Label>
          <Value>{systemDiskSize}GiB</Value>
        </div>
        {/* 第 2 行：公网 IP */}
        <div className="text-xs leading-relaxed flex items-center flex-wrap gap-x-0">
          <Label>公网 IP：</Label>
          <Value>{assignPublicIp ? "分配" : "不分配"}</Value>
          {assignPublicIp && (
            <>
              <Sep />
              <Label>公网计费模式：</Label>
              <Value>{bandwidthBillingModeLabel}</Value>
              <Sep />
              <Label>带宽上限：</Label>
              <Value>{bandwidthLimit}Mbps</Value>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function ConfigOverviewTab({
  nodeId,
  groups,
  health,
  isAnomalous = false,
  anomalousBoundConfigs = [],
  isUninitialized = false,
  isProject = false,
}: ConfigOverviewTabProps) {
  // 项目节点：配置总览仅展示 agent 工具模块，其余类别与锚点导航条一律不显示
  const displayOrder: ConfigCategory[] = isProject ? ["agentTool"] : CATEGORY_ORDER;
  // 获取当前节点的全部配置条目
  const configEntries = useMemo(() => getConfigEntries(nodeId, groups), [nodeId, groups]);

  // 按 category 组织
  const byCategory = useMemo(() => {
    const map = new Map<ConfigCategory, ConfigEntry[]>();
    configEntries.forEach((e) => {
      const list = map.get(e.category) ?? [];
      list.push(e);
      map.set(e.category, list);
    });
    return map;
  }, [configEntries]);

  // 异常组织：统计有「本组织」(local) 配置的类别集合，用于显示红点
  const anomalousLocalCategories = useMemo(() => {
    if (!isAnomalous) return new Set<ConfigCategory>();
    const set = new Set<ConfigCategory>();
    configEntries.forEach((e) => {
      if (e.source.type === "local") {
        set.add(e.category);
      }
    });
    return set;
  }, [configEntries, isAnomalous]);

  // 初始化未完成：计算缺失的配置类别集合（用于导航栏+标题橙色点）
  const uninitializedCategories = useMemo(() => {
    if (!isUninitialized || isAnomalous) return new Set<ConfigCategory>();
    const initHealth = getGroupInitHealth(nodeId, groups);
    const set = new Set<ConfigCategory>();
    initHealth.missing.forEach((m) => {
      set.add(INIT_MISSING_TO_CATEGORY[m]);
    });
    return set;
  }, [nodeId, groups, isUninitialized, isAnomalous]);

  /**
   * 网络配置待更新（VPC / 子网被云端删除）
   *
   * 当前组织的网络配置中存在任意一个 VPC 或子网待更新时为 true，
   * 用于：顶部配置总览条「网络」节点 + 网络卡片标题旁的红色小圆点。
   * 与各资源行的逐项「⚠ 配置待更新」保持同源判定。
   */
  const networkOutdated = useMemo(
    () => hasNetworkOutdated(nodeId, groups),
    [nodeId, groups]
  );

  // 折叠状态：默认全部展开
  const [collapsed, setCollapsed] = useState<Set<ConfigCategory>>(new Set());
  const toggleCollapse = (cat: ConfigCategory) => {
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(cat)) {
        next.delete(cat);
      } else {
        next.add(cat);
      }
      return next;
    });
  };

  // ─── 锚点导航 ───
  const sectionRefs = useRef<Map<ConfigCategory, HTMLDivElement>>(new Map());
  const navRef = useRef<HTMLDivElement>(null);
  const [activeCat, setActiveCat] = useState<ConfigCategory>(CATEGORY_ORDER[0]);

  const setSectionRef = useCallback((cat: ConfigCategory, el: HTMLDivElement | null) => {
    if (el) {
      sectionRefs.current.set(cat, el);
    } else {
      sectionRefs.current.delete(cat);
    }
  }, []);

  // 滚动监听：判断哪个 section 在视口中
  useEffect(() => {
    // 找到最近的可滚动祖先容器
    const nav = navRef.current;
    if (!nav) return;
    const scrollContainer = nav.closest<HTMLElement>(".overflow-y-auto");
    if (!scrollContainer) return;

    const handleScroll = () => {
      const containerTop = scrollContainer.getBoundingClientRect().top;
      let current: ConfigCategory = CATEGORY_ORDER[0];
      for (const cat of CATEGORY_ORDER) {
        const el = sectionRefs.current.get(cat);
        if (el) {
          const rect = el.getBoundingClientRect();
          if (rect.top - containerTop <= 80) {
            current = cat;
          }
        }
      }
      setActiveCat(current);
    };
    scrollContainer.addEventListener("scroll", handleScroll, { passive: true });
    return () => scrollContainer.removeEventListener("scroll", handleScroll);
  }, []);

  const scrollToSection = (cat: ConfigCategory) => {
    const el = sectionRefs.current.get(cat);
    if (el) {
      el.scrollIntoView({ behavior: "instant", block: "start" });
    }
  };

  // 三大核心检查卡
  const checks: Array<{
    key: "model" | "channel" | "securityGroup";
    status: "ok" | "missing";
  }> = (["model", "channel", "securityGroup"] as const).map((k) => ({
    key: k,
    status: health.missing.includes(k) ? "missing" : "ok",
  }));

  return (
    <div className="relative">
      {/* 锚点导航条 — 时间轴风格（项目节点仅有单一模块，无需锚点导航） */}
      {!isProject && (
      <div ref={navRef} className="sticky top-0 z-10 bg-white -mx-6 px-6 pt-3 pb-3 border-b border-[#e5e5e5]">
        <div className="flex items-center w-full">
          {CATEGORY_ORDER.map((cat, idx) => {
            const isActive = activeCat === cat;
            const activeIdx = CATEGORY_ORDER.indexOf(activeCat);
            const isPast = idx < activeIdx;
            const isLast = idx === CATEGORY_ORDER.length - 1;
            const catMeta = CONFIG_CATEGORY_META[cat];
            const hasAnomaly = anomalousLocalCategories.has(cat);
            const hasUninitWarning = uninitializedCategories.has(cat);
            return (
              <React.Fragment key={cat}>
                {/* 导航项：圆点 + 文字 */}
                <button
                  type="button"
                  onClick={() => scrollToSection(cat)}
                  className="flex flex-col items-center gap-1 shrink-0 group"
                >
                  <span
                    className={`w-2 h-2 rounded-full transition-colors duration-200 ${
                      isActive
                        ? "bg-[#1447E6] ring-4 ring-blue-50"
                        : isPast
                        ? "bg-[#94A3B8]"
                        : "bg-[#CBD5E1] group-hover:bg-[#94A3B8]"
                    }`}
                  />
                  <span className="relative inline-flex">
                    <span
                      className={`text-xs font-medium transition-colors duration-200 whitespace-nowrap ${
                        isActive
                          ? "text-[var(--text-brand)]"
                          : isPast
                          ? "text-[var(--text-brand)]"
                          : "text-[var(--text-weak)] group-hover:text-[var(--text-muted)]"
                      }`}
                    >
                      {CATEGORY_NAV_LABEL[cat]}
                    </span>
                    {hasAnomaly && (
                      <span className="absolute -top-0.5 -right-1.5 w-1.5 h-1.5 rounded-full bg-red-500" />
                    )}
                    {!hasAnomaly && hasUninitWarning && (
                      <span className="absolute -top-0.5 -right-1.5 w-1.5 h-1.5 rounded-full bg-[var(--alert-warning-icon)]" />
                    )}
                    {/* 网络配置待更新（VPC/子网被云端删除）：仅「网络」节点展示明黄色小圆点，
                        与异常点互斥，归类为初始化未完成 */}
                    {!hasAnomaly && !hasUninitWarning && cat === "network" && networkOutdated && (
                      <span className="absolute -top-0.5 -right-1.5 w-1.5 h-1.5 rounded-full bg-[var(--alert-warning-icon)]" />
                    )}
                  </span>
                </button>
                {/* 连接线 */}
                {!isLast && (
                  <div
                    className={`flex-1 h-px mx-1 mt-[-14px] transition-colors duration-200 ${
                      idx < activeIdx ? "bg-[#BFDBFE]" : "bg-[#E2E8F0]"
                    }`}
                  />
                )}
              </React.Fragment>
            );
          })}
        </div>
      </div>
      )}

      {/* 异常组织告警条 */}
      {isAnomalous && (
        <Alert variant="warning" className="mt-3">
          <CircleAlert />
          <AlertTitle>该组织的专属配置未解绑</AlertTitle>
          <AlertDescription>
            该组织对应的部门已在腾讯统一身份管理平台被删除。请前往对应配置页面将专属于「本组织」的配置与本组织解绑或删除，处理完成后刷新组织列表，组织即可被清除。来自「全部用户」、「继承自上级组织」和「预设策略」的配置项无需处理。
          </AlertDescription>
        </Alert>
      )}

      {/* 初始化未完成黄色告警条（优先级低于异常组织，不同时展示） */}
      {!isAnomalous && isUninitialized && (
        <Alert variant="warning" className="mt-3">
          <CircleAlert />
          <AlertTitle>该组织初始化配置未完成</AlertTitle>
          <AlertDescription>
            当前组织缺少必要的初始化配置（{Array.from(uninitializedCategories).map((cat) => CATEGORY_NAV_LABEL[cat]).join("、")}），可能影响组织内用户在用户端的正常使用，请尽快完成配置。
          </AlertDescription>
        </Alert>
      )}

      <div className="space-y-3 pt-3">
        {displayOrder.map((cat) => {
          const entries = byCategory.get(cat) ?? [];
          const catMeta = CONFIG_CATEGORY_META[cat];
          const iconSrc = CATEGORY_ICON_SRC[cat];
          const hasAnomaly = anomalousLocalCategories.has(cat);
          const hasUninitWarning = uninitializedCategories.has(cat);
          return (
            <div
              key={cat}
              ref={(el) => setSectionRef(cat, el)}
              className="bg-white rounded-xl border border-[#e5e5e5] overflow-hidden scroll-mt-[3.75rem]"
            >
              {/* 配置项 header */}
              <div
                className={`flex items-center justify-between px-6 py-3.5 cursor-pointer select-none hover:bg-[var(--bg-grey-hover)]/50 transition-colors ${
                  collapsed.has(cat) ? "" : "border-b border-[#f5f5f5]"
                }`}
                onClick={() => toggleCollapse(cat)}
              >
                <div className="flex items-center gap-3 min-w-0">
                  <img
                    src={iconSrc}
                    alt=""
                    aria-hidden="true"
                    className="w-9 h-9 shrink-0"
                  />
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="relative inline-flex text-sm font-semibold leading-5 text-[var(--text-title)]">
                        {catMeta.label}
                        {hasAnomaly && (
                          <span className="absolute -top-0.5 -right-2 w-1.5 h-1.5 rounded-full bg-red-500" />
                        )}
                        {!hasAnomaly && hasUninitWarning && (
                          <span className="absolute -top-0.5 -right-2 w-1.5 h-1.5 rounded-full bg-[var(--alert-warning-icon)]" />
                        )}
                        {/* 网络配置待更新（VPC/子网被云端删除）：仅「网络」卡片标题旁展示明黄色小圆点 */}
                        {!hasAnomaly && !hasUninitWarning && cat === "network" && networkOutdated && (
                          <span className="absolute -top-0.5 -right-2 w-1.5 h-1.5 rounded-full bg-[var(--alert-warning-icon)]" />
                        )}
                      </span>
                      {/* 仅模型、通道、镜像在标题旁显示数量；技能/Agent工具在子类别显示；其余不显示 */}
                      {(cat === "model" || cat === "channel" || cat === "image") && (
                        <span className="text-xs font-semibold leading-5 text-[var(--text-emphasis)] tabular-nums">
                          {entries.length} 个
                        </span>
                      )}
                    </div>
                    <div className="text-xs leading-[18px] text-[var(--text-muted)] mt-0.5">
                      {catMeta.description}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Link
                    href={catMeta.path}
                    className="inline-flex items-center gap-1 text-xs text-[var(--text-brand)] hover:underline"
                    onClick={(e: React.MouseEvent) => e.stopPropagation()}
                  >
                    管理 <ExternalLink className="w-3 h-3" />
                  </Link>
                  <ChevronDown
                    className={`w-4 h-4 text-[#A3A3A3] transition-transform duration-200 ${
                      collapsed.has(cat) ? "-rotate-90" : ""
                    }`}
                  />
                </div>
              </div>

              {/* 条目列表（可折叠） */}
              {!collapsed.has(cat) && (
              <div className="divide-y divide-[#f5f5f5]">
                {entries.length === 0 ? (
                  <div className="px-6 py-6 text-center">
                    <span className="text-xs text-[var(--text-weak)]">暂未配置</span>
                  </div>
                ) : cat === "cloudResource" ? (
                  entries.map((entry) => (
                    <CloudResourceDetail key={entry.id} entry={entry} />
                  ))
                ) : (cat === "skill" || cat === "agentTool" || cat === "platformPolicy" || cat === "network") ? (
                  (() => {
                    // 按 subLabel 组织
                    const grouped = new Map<string, ConfigEntry[]>();
                    entries.forEach((e) => {
                      const key = e.subLabel || "其他";
                      const list = grouped.get(key) ?? [];
                      list.push(e);
                      grouped.set(key, list);
                    });
                    return Array.from(grouped.entries()).map(([groupLabel, groupEntries]) => (
                      <div key={groupLabel}>
                        {/* 大类标题 */}
                        <div className="px-6 py-2 bg-[#fafafa]/80 border-b border-[#f5f5f5] flex items-center gap-2">
                          <MetaText>
                            {groupLabel}
                          </MetaText>
                          {/* 技能和Agent工具在子类别标题旁显示数量 */}
                          {(cat === "skill" || cat === "agentTool") && (
                            <MetaText className="tabular-nums">
                              {groupEntries.length} 个
                            </MetaText>
                          )}
                        </div>
                        {/* 大类下的条目 */}
                        {groupEntries.map((entry) => (
                          <div key={entry.id} className="px-6 py-3 border-b border-[#f5f5f5] last:border-b-0">
                            {/* 公网特殊展示 */}
                            {entry.subLabel === "公网" && entry.meta ? (
                              <PublicNetworkDetail meta={entry.meta} source={entry.source} />
                            ) : entry.subLabel === "私有网络与子网" && entry.meta ? (
                              /* VPC + 子网结构化展示
                               * 用户管理页只展示「可用」的 VPC / 子网，不展示已删除资源；
                               * 仅在以下两种场景提示「⚠ 配置待更新」：
                               *   1) VPC 整个被删除
                               *   2) 某个可用区下所有子网均被删除（无法用于实例创建）
                               */
                              <div className="space-y-2">
                                {/* 私有网络
                                 * 与线上保持一致：仅展示 VPC ID（不展示名称 / CIDR）。
                                 * 触发条件：vpcId 存在，但 vpcName / vpcCidr 缺失 → 视为 VPC 已被云端删除。
                                 * 预设策略来源：VPC 由平台自动重建，不展示真实 ID，统一展示「自动分配」。 */}
                                {(() => {
                                  const vpcId = entry.meta.vpcId ? String(entry.meta.vpcId) : "";
                                  const vpcName = entry.meta.vpcName ? String(entry.meta.vpcName) : "";
                                  const vpcCidr = entry.meta.vpcCidr ? String(entry.meta.vpcCidr) : "";
                                  const vpcOutdated = !!vpcId && (!vpcName || !vpcCidr);
                                  const isPreset = entry.source.type === "presetPolicy";
                                  return (
                                    <div className="flex items-center gap-2 flex-wrap">
                                      <span className="text-xs text-[var(--text-muted)] shrink-0">私有网络：</span>
                                      <span className="text-xs font-semibold text-[var(--text-emphasis)]">
                                        {isPreset ? <>自动分配</> : <>{vpcId}</>}
                                      </span>
                                      <SourceBadge source={entry.source} />
                                      {isAnomalous && entry.source.type === "local" && <LocalAnomalyHint />}
                                      {!isPreset && vpcOutdated && <ConfigOutdatedHint />}
                                    </div>
                                  );
                                })()}
                                {/* 子网列表
                                 * - 预设策略来源：按可用区逐条展示「子网：[可用区] 自动分配」
                                 * - VPC 已整体删除：不展示任何子网行（已无意义）
                                 * - VPC 健在：
                                 *     · 仅展示可用子网（已删除子网不进入此列表）
                                 *     · 某可用区下所有子网均被删除时（zonesAllDeleted），
                                 *       追加一行「子网：[可用区] 无可用子网 ⚠ 配置待更新」 */}
                                {(() => {
                                  const isPreset = entry.source.type === "presetPolicy";
                                  if (isPreset) {
                                    const zones = Array.isArray(entry.meta.zones) ? (entry.meta.zones as string[]) : [];
                                    return zones.map((zone, idx) => (
                                      <div key={`preset-zone-${zone}-${idx}`} className="flex items-center gap-2 flex-wrap pl-4">
                                        <span className="text-xs text-[var(--text-muted)] shrink-0">子网：</span>
                                        <Badge variant="secondary">{zone}</Badge>
                                        <span className="text-xs font-semibold text-[var(--text-emphasis)]">自动分配</span>
                                      </div>
                                    ));
                                  }
                                  const vpcId = entry.meta.vpcId ? String(entry.meta.vpcId) : "";
                                  const vpcName = entry.meta.vpcName ? String(entry.meta.vpcName) : "";
                                  const vpcCidr = entry.meta.vpcCidr ? String(entry.meta.vpcCidr) : "";
                                  const vpcOutdated = !!vpcId && (!vpcName || !vpcCidr);
                                  // VPC 整体删除时，隐藏所有子网行
                                  if (vpcOutdated) return null;
                                  const subnets = Array.isArray(entry.meta.subnets)
                                    ? (entry.meta.subnets as Array<{ zone: string; subnetId: string; subnetName?: string; subnetCidr: string }>)
                                    : [];
                                  const zonesAllDeleted = Array.isArray(entry.meta.zonesAllDeleted)
                                    ? (entry.meta.zonesAllDeleted as string[])
                                    : [];
                                  return (
                                    <>
                                      {subnets.map((subnet, idx) => (
                                        <div key={`${subnet.subnetId}-${idx}`} className="flex items-center gap-2 flex-wrap pl-4">
                                          <span className="text-xs text-[var(--text-muted)] shrink-0">子网：</span>
                                          <Badge variant="secondary">{subnet.zone}</Badge>
                                          {/* 与线上保持一致：仅展示 subnetId（不展示子网名 / CIDR） */}
                                          <span className="text-xs font-semibold text-[var(--text-emphasis)]">
                                            {subnet.subnetId}
                                          </span>
                                        </div>
                                      ))}
                                      {zonesAllDeleted.map((zone, idx) => (
                                        <div key={`zone-empty-${zone}-${idx}`} className="flex items-center gap-2 flex-wrap pl-4">
                                          <span className="text-xs text-[var(--text-muted)] shrink-0">子网：</span>
                                          <Badge variant="secondary">{zone}</Badge>
                                          <span className="text-xs text-[var(--text-weak)]">无可用子网</span>
                                          <ConfigOutdatedHint />
                                        </div>
                                      ))}
                                    </>
                                  );
                                })()}
                              </div>
                            ) : (
                              <div className="flex items-center justify-between gap-3">
                                <div className="flex-1 min-w-0">
                                  <div className="flex items-center gap-2 flex-wrap">
                                    <span className="text-xs font-medium leading-[18px] text-[var(--text-emphasis)] truncate">
                                      {entry.label}
                                    </span>
                                    <SourceBadge source={entry.source} isProject={isProject} />
                                    {isAnomalous && entry.source.type === "local" && <LocalAnomalyHint />}
                                  </div>
                                </div>
                                <div className="shrink-0">
                                  {cat === "platformPolicy" && <PolicyEntryValue entry={entry} />}
                                </div>
                              </div>
                            )}
                          </div>
                        ))}
                      </div>
                    ));
                  })()
                ) : (
                  entries.map((entry) => (
                    <div key={entry.id} className="flex items-center justify-between px-6 py-3 gap-3">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="text-xs font-medium leading-[18px] text-[var(--text-emphasis)] truncate">
                            {entry.label}
                          </span>
                          <SourceBadge source={entry.source} isProject={isProject} />
                          {isAnomalous && entry.source.type === "local" && <LocalAnomalyHint />}
                        </div>
                      </div>
                      <div className="shrink-0">
                        {(cat === "platformPolicy" || cat === "cloudDev") && <PolicyEntryValue entry={entry} />}
                      </div>
                    </div>
                  ))
                )}
              </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
