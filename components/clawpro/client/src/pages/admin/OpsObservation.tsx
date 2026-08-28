/**
 * OpsObservation - 运维观测页面
 * Design: 「流动蓝图」Fluid Blueprint
 * - 标题、副标题、卡片、icon 与其他子页面保持一致
 */
import { useState, useEffect, useMemo } from "react";
import { useLocation } from "wouter";
import { ArrowUpRight, RefreshCw, CheckCircle2, AlertTriangle, Info, Timer } from "lucide-react";
import { Dialog, DialogBody, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction } from "@/components/ui/alert-dialog";
import { Alert, AlertDescription, AlertTitle, AlertOperationInfoIcon } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { SurfaceCard, SurfaceInner } from "@/components/ui/Surface";
import { StatNumber } from "@/components/ui/Typography";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { DatePicker } from "@/components/ui/date-picker";
import { LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, PieChart, Pie, Cell } from "recharts";
import { SearchableSelect } from "@/components/ui/select";
import { GroupSelect } from "@/components/GroupSelect";
import { TreeSelect, type TreeSelectNode } from "@/components/ui/tree-select";
import { ScopeSelect } from "@/components/ScopeSelect";
import { Badge } from "@/components/ui/badge";
import { SlidersHorizontal } from "lucide-react";
import { useFilterSections } from "@/lib/clsScopeMock";
import type { FilterSection, TreeNodeData } from "@/components/groupTreeShared";
import type { GroupSource, UserGroup } from "@/pages/admin/MemberManagement/types";
import { toast } from "sonner";
import { useOpsObservationCalendarBillingExempt } from "./OpsObservation/useOpsObservationCalendarBillingExempt";
import { Tooltip as UITooltip, TooltipContent as UITooltipContent, TooltipTrigger as UITooltipTrigger } from "@/components/ui/tooltip";
import {
  usePluginUpgrade,
  needsPluginUpgradeHint,
  LATEST_PLUGIN_VERSION,
} from "@/contexts/PluginUpgradeContext";


/**
 * 环比指标样式工具
 * - polarity: 'positive' 正向指标（值越大越好，如吞吐量、成功率），'negative' 负向指标（值越小越好，如耗时、错误数、队列深度）
 * - delta: 'up' 上升 / 'down' 下降 / 'flat' 持平
 * 规则：正向↑/负向↓ → 绿色；正向↓/负向↑ → 红色；持平 → 灰色
 */
function getTrendStyle(polarity: 'positive' | 'negative', delta: 'up' | 'down' | 'flat'): { color: string; arrow: string } {
  if (delta === 'flat') return { color: 'text-gray-400', arrow: '' };
  const isGood = (polarity === 'positive' && delta === 'up') || (polarity === 'negative' && delta === 'down');
  return {
    color: isGood ? 'text-emerald-600' : 'text-red-500',
    arrow: delta === 'up' ? '↑' : '↓',
  };
}

/**
 * 环比说明 Tooltip
 * 统一文案：周期定义 = 用户所选时间范围的上一个等长周期；超出 30 天存储则无对比数据
 */
function TrendTooltipContent() {
  return (
    <UITooltipContent side="top" className="max-w-[280px] text-xs leading-relaxed space-y-2 bg-white text-[var(--text-secondary)] border border-[var(--border)] shadow-[0_4px_12px_rgba(0,0,0,0.08)]">
      <p><span className="font-semibold text-[var(--text-title)]">环比说明：</span>与您所选时间范围的上一个等长周期对比。例：选择 5/10–5/19（10 天），将与 4/30–5/9 对比。</p>
      <p>当前数据默认存储 30 天，若上一周期已超出存储范围，将显示「无对比数据」。</p>
    </UITooltipContent>
  );
}

const SCOPE_GROUP_SOURCE_FILTER: GroupSource[] = ["oneid-dept", "manual"];
const SCOPE_GROUP_SOURCE_LABELS: Partial<Record<GroupSource, string>> = {
  "oneid-dept": "部门",
  manual: "自定义分组",
};

function getScopeGroupSource(sectionKey: string): GroupSource {
  return sectionKey === "dept" ? "oneid-dept" : "manual";
}

function flattenScopeGroupNodes(
  nodes: TreeNodeData[],
  source: GroupSource,
  parentId: string | null,
): UserGroup[] {
  return nodes.flatMap((node) => [
    {
      id: node.id,
      name: node.name,
      parentId,
      source,
      readonly: source !== "manual",
      createdAt: "",
    },
    ...flattenScopeGroupNodes(node.children ?? [], source, node.id),
  ]);
}

function toScopeGroupSelectGroups(sections: FilterSection[]): UserGroup[] {
  return sections.flatMap((section) =>
    flattenScopeGroupNodes(section.roots, getScopeGroupSource(section.key), null),
  );
}

/** 把分区树节点转换为 TreeSelect 节点（递归带完整 path 面包屑） */
function treeNodeToSelectNode(node: TreeNodeData, parentPath: string): TreeSelectNode {
  const path = parentPath ? `${parentPath}/${node.name}` : node.name;
  return {
    id: node.id,
    name: node.name,
    path,
    children: node.children?.map((child) => treeNodeToSelectNode(child, path)),
  };
}

/**
 * 把「部门 / 自定义分组」分区数据转换为 TreeSelect 树：
 *   - 多分区时，每个分区作为一个顶层分组节点（保留分区语义与树结构）
 *   - 单分区时直接平铺其根节点
 */
function toGroupTreeSelectNodes(sections: FilterSection[]): TreeSelectNode[] {
  const usableSections = sections.filter((section) => section.roots.length > 0);
  if (usableSections.length === 1) {
    return usableSections[0].roots.map((root) => treeNodeToSelectNode(root, ""));
  }
  return usableSections.map((section) => ({
    id: `__section_${section.key}`,
    name: section.label,
    path: section.label,
    children: section.roots.map((root) => treeNodeToSelectNode(root, section.label)),
  }));
}

/**
 * 带说明 Tooltip 的环比 Badge（hover 后展示周期定义 + 30 天存储说明）
 */
function TrendBadge({ className, children }: { className?: string; children: React.ReactNode }) {
  return (
    <UITooltip>
      <UITooltipTrigger asChild>
        <span
          className={`inline-flex items-center cursor-help ${className ?? ''}`}
        >
          {children}
        </span>
      </UITooltipTrigger>
      <TrendTooltipContent />
    </UITooltip>
  );
}


// 设计系统标准 SVG icon（渐变黑→蓝，与 TokensMonitor 统一风格）
const METRIC_ICONS: React.ReactNode[] = [
  /* 闪电 - AGENT 总数 */
  <svg key="i0" width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M11.1557 0.568474C11.2759 0.547602 11.3997 0.565694 11.5083 0.621208C11.6168 0.676751 11.7039 0.766463 11.7573 0.876091C11.8107 0.985788 11.8275 1.10986 11.8042 1.22961L10.77 6.39172L14.8227 7.91125C14.9089 7.94398 14.9857 7.99716 15.0464 8.06652C15.1071 8.13609 15.1505 8.2197 15.1714 8.30968C15.1922 8.39969 15.1905 8.4939 15.1665 8.58312C15.1425 8.67222 15.0968 8.75406 15.0337 8.8214H15.0366L7.1616 17.2589L7.09421 17.3204C7.0224 17.3757 6.9373 17.4131 6.84714 17.4288L6.7573 17.4366C6.69672 17.4373 6.63627 17.4288 6.57859 17.4103L6.49461 17.3751C6.386 17.3195 6.29798 17.2299 6.24461 17.1202C6.20472 17.0381 6.18625 16.9479 6.18894 16.8575L6.19871 16.7667L7.22996 11.6105L3.17722 10.089C3.11208 10.0646 3.05213 10.0285 3.00046 9.98254L2.95164 9.93273C2.9057 9.8803 2.86992 9.82011 2.84617 9.755L2.82664 9.68859C2.80577 9.59809 2.80709 9.50378 2.83152 9.41418C2.85597 9.32456 2.90234 9.2423 2.96629 9.17492L10.8413 0.737419C10.9247 0.648358 11.0355 0.589437 11.1557 0.568474ZM5.34324 9.09972L9.1655 10.5353L8.63035 13.2111L11.1528 10.5089H11.1401L12.6479 8.89758L8.83445 7.46789L9.37058 4.78527L5.34324 9.09972Z" fill="url(#ops_icon_0)"/><defs><radialGradient id="ops_icon_0" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(2.81201 8.99836) scale(12.3738 747.725)"><stop stopColor="#202020"/><stop offset="1" stopColor="#0080FF"/></radialGradient></defs></svg>,
  /* 下载/入队 - 会话数量 */
  <svg key="i1" width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M5.02805 6.22195C4.86954 6.06344 4.78049 5.84846 4.78049 5.6243C4.78049 5.40013 4.86954 5.18515 5.02805 5.02664C5.18656 4.86813 5.40154 4.77908 5.6257 4.77908C5.84987 4.77908 6.06485 4.86813 6.22336 5.02664L8.15625 6.96094V1.6875C8.15625 1.46372 8.24514 1.24911 8.40338 1.09088C8.56161 0.932645 8.77622 0.84375 9 0.84375C9.22378 0.84375 9.43839 0.932645 9.59662 1.09088C9.75485 1.24911 9.84375 1.46372 9.84375 1.6875V6.96094L11.778 5.02594C11.9366 4.86743 12.1515 4.77838 12.3757 4.77838C12.5999 4.77838 12.8149 4.86743 12.9734 5.02594C13.1319 5.18445 13.2209 5.39943 13.2209 5.62359C13.2209 5.84776 13.1319 6.06274 12.9734 6.22125L9.59836 9.59625C9.51997 9.67491 9.42683 9.73732 9.32427 9.77991C9.22171 9.82249 9.11175 9.84442 9.0007 9.84442C8.88965 9.84442 8.7797 9.82249 8.67714 9.77991C8.57458 9.73732 8.48143 9.67491 8.40305 9.59625L5.02805 6.22195ZM15.75 8.15625H13.2188C12.995 8.15625 12.7804 8.24514 12.6221 8.40338C12.4639 8.56161 12.375 8.77622 12.375 9C12.375 9.22378 12.4639 9.43839 12.6221 9.59662C12.7804 9.75485 12.995 9.84375 13.2188 9.84375H15.4688V13.7812H2.53125V9.84375H4.78125C5.00503 9.84375 5.21964 9.75485 5.37787 9.59662C5.53611 9.43839 5.625 9.22378 5.625 9C5.625 8.77622 5.53611 8.56161 5.37787 8.40338C5.21964 8.24514 5.00503 8.15625 4.78125 8.15625H2.25C1.87704 8.15625 1.51935 8.30441 1.25563 8.56813C0.991908 8.83185 0.84375 9.18954 0.84375 9.5625V14.0625C0.84375 14.4355 0.991908 14.7931 1.25563 15.0569C1.51935 15.3206 1.87704 15.4688 2.25 15.4688H15.75C16.123 15.4688 16.4806 15.3206 16.7444 15.0569C17.0081 14.7931 17.1562 14.4355 17.1562 14.0625V9.5625C17.1563 9.18954 17.0081 8.83185 16.7444 8.56813C16.4806 8.30441 16.123 8.15625 15.75 8.15625ZM14.3438 11.8125C14.3438 11.59 14.2778 11.3725 14.1542 11.1875C14.0305 11.0025 13.8548 10.8583 13.6493 10.7731C13.4437 10.688 13.2175 10.6657 12.9993 10.7091C12.781 10.7525 12.5806 10.8597 12.4233 11.017C12.2659 11.1743 12.1588 11.3748 12.1154 11.593C12.072 11.8113 12.0942 12.0375 12.1794 12.243C12.2645 12.4486 12.4087 12.6243 12.5937 12.7479C12.7787 12.8715 12.9962 12.9375 13.2188 12.9375C13.5171 12.9375 13.8033 12.819 14.0142 12.608C14.2252 12.397 14.3438 12.1109 14.3438 11.8125Z" fill="url(#ops_icon_1)"/><defs><radialGradient id="ops_icon_1" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(0.843749 8.15625) scale(16.3125 647.966)"><stop stopColor="#202020"/><stop offset="1" stopColor="#0080FF"/></radialGradient></defs></svg>,
  /* 星星 - 消息响应 P95 */
  <svg key="i2" width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M16.6322 7.68155L12.2953 6.10444L10.7182 1.76757C10.6198 1.49691 10.4405 1.26309 10.2047 1.09786C9.9688 0.932629 9.68779 0.843994 9.39982 0.843994C9.11184 0.843994 8.83084 0.932629 8.59498 1.09786C8.35912 1.26309 8.17983 1.49691 8.08146 1.76757L6.50435 6.10444L2.16747 7.68155C1.89682 7.77992 1.66299 7.95921 1.49776 8.19507C1.33253 8.43093 1.2439 8.71193 1.2439 8.99991C1.2439 9.28789 1.33253 9.56889 1.49776 9.80475C1.66299 10.0406 1.89682 10.2199 2.16747 10.3183L6.50435 11.8954L8.08146 16.2323C8.17983 16.5029 8.35912 16.7367 8.59498 16.902C8.83084 17.0672 9.11184 17.1558 9.39982 17.1558C9.68779 17.1558 9.9688 17.0672 10.2047 16.902C10.4405 16.7367 10.6198 16.5029 10.7182 16.2323L12.2953 11.8954L16.6322 10.3183C16.9028 10.2199 17.1366 10.0406 17.3019 9.80475C17.4671 9.56889 17.5557 9.28789 17.5557 8.99991C17.5557 8.71193 17.4671 8.43093 17.3019 8.19507C17.1366 7.95921 16.9028 7.77992 16.6322 7.68155ZM11.3489 10.4441C11.2329 10.4863 11.1277 10.5533 11.0404 10.6405C10.9532 10.7278 10.8862 10.833 10.844 10.949L9.39982 14.9209L7.9556 10.949C7.91347 10.833 7.84643 10.7278 7.7592 10.6405C7.67198 10.5533 7.56669 10.4863 7.45075 10.4441L3.4788 8.99991L7.45075 7.55569C7.56669 7.51356 7.67198 7.44653 7.7592 7.3593C7.84643 7.27208 7.91347 7.16679 7.9556 7.05085L9.39982 3.0789L10.844 7.05085C10.8862 7.16679 10.9532 7.27208 11.0404 7.3593C11.1277 7.44653 11.2329 7.51356 11.3489 7.55569L15.3208 8.99991L11.3489 10.4441Z" fill="url(#ops_icon_2)"/><defs><radialGradient id="ops_icon_2" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(1.2439 8.99991) scale(16.3118 722.702)"><stop stopColor="#202020"/><stop offset="1" stopColor="#0080FF"/></radialGradient></defs></svg>,
];


const SESSION_MGMT_ICON_BASE = "/assets/admin-session-management";

// 未开启 CLS 时的引导卡（对齐 main：当前页能力 2 张 + 其他页能力 4 张）
const EXISTING_OBSERVATION_CARDS = [
  {
    id: "health-monitoring",
    title: "业务运行健康度实时监控",
    description: "聚焦消息处理总量、入队效率与卡死会话，保障系统稳定运行",
    iconSrc: `${SESSION_MGMT_ICON_BASE}/business-health-monitoring.svg`,
  },
  {
    id: "log-metrics-insight",
    title: "应用日志与 OTEL 指标全景洞察",
    description: "多维度分析日志级别与模块分布，精细化追踪消息处理、队列状态与执行耗时",
    iconSrc: `${SESSION_MGMT_ICON_BASE}/app-log-otel-insight.svg`,
  },
];

const CLS_NEW_CARDS = [
  {
    id: "high-cost-session",
    title: "高Token会话实时分析与管控",
    description: "聚焦 TOP 会话的 Token 消耗、轮次分布与耗时特征，精准定位高Token交互，优化模型调用成本与资源效率",
    iconSrc: `${SESSION_MGMT_ICON_BASE}/high-token-session-control.svg`,
  },
  {
    id: "single-session-cost",
    title: "单会话全链路Token透视",
    description: "拆解每轮交互的 Token 流量与耗时分布，可视化工具调用与上下文膨胀对成本的影响",
    iconSrc: `${SESSION_MGMT_ICON_BASE}/single-session-token-insight.svg`,
  },
  {
    id: "session-global-monitoring",
    title: "会话全局运行态势监控",
    description: "聚合总会话数、平均轮次与工具调用量，多维度洞察渠道与模型分布，实现会话全生命周期可追溯、可分析",
    iconSrc: `${SESSION_MGMT_ICON_BASE}/session-global-monitoring.svg`,
  },
  {
    id: "session-efficiency",
    title: "会话详情与交互效率精细化分析",
    description: "聚焦单会话 Token 消耗，可视化渠道与模型分布特征，精准定位高Token会话，优化资源配置与调用效率",
    iconSrc: `${SESSION_MGMT_ICON_BASE}/session-detail-analysis.svg`,
  },
];

// CLS 采集插件版本管理已迁移到 lib/pluginUpgrade



// 工具函数
function toDateStr(d: Date) {
  return d.toISOString().slice(0, 10);
}
function todayStr() {
  return toDateStr(new Date());
}
function addDays(base: string, n: number) {
  const d = new Date(base);
  d.setDate(d.getDate() + n);
  return toDateStr(d);
}

export default function OpsObservation() {
  // 停服态下，本页两个"选择日期" DatePicker 弹出的日历面板需保持 100% 可用。
  // 触发器已通过外层 <div data-billing-exempt> 打标；但面板经 Radix Portal 挂到 <body>，
  // 不在触发器的祖先链上，因此需要独立的页面级 hook 打标 + 注入 CSS 补充规则；
  // 详见 ./OpsObservation/useOpsObservationCalendarBillingExempt.ts 头部注释。
  useOpsObservationCalendarBillingExempt();
  const today = todayStr();
  const [, navigate] = useLocation();
  const [dateFrom, setDateFrom] = useState(today);
  const [dateTo, setDateTo] = useState(today);
  const [refreshing, setRefreshing] = useState(false);
  const [clsEnabled, setClsEnabled] = useState(() => {
    const stored = localStorage.getItem("globalClsEnabled");
    return stored === "true";
  });
  const [isEnablingCls, setIsEnablingCls] = useState(false);
  const [showSuccessMessage, setShowSuccessMessage] = useState(false);
  const [showCloseClsConfirm, setShowCloseClsConfirm] = useState(false);
  const [isClosingCls, setIsClosingCls] = useState(false);
  const [deleteLogTopic, setDeleteLogTopic] = useState(false);
  const [showClsAgreementDialog, setShowClsAgreementDialog] = useState(false);
  const [clsAgreed, setClsAgreed] = useState(false);
  const [showAuthDialog, setShowAuthDialog] = useState(false);
  const [isCheckingAuth, setIsCheckingAuth] = useState(false);
  const [authCompleted, setAuthCompleted] = useState(false);
  const [authCheckInterval, setAuthCheckInterval] = useState<NodeJS.Timeout | null>(null);
  const [showFreeQuotaDialog, setShowFreeQuotaDialog] = useState(false);
  const [freeQuotaAgreed, setFreeQuotaAgreed] = useState(false);
  const [selectedOpenClaw, setSelectedOpenClaw] = useState("");
  const [groupFilter, setGroupFilter] = useState<string>(""); // "" = 全部分组
  const [agentFilter, setAgentFilter] = useState<string>(""); // "" = 全部 Agent
  const [activeTab, setActiveTab] = useState<"message" | "perf" | "skill">("message");
  const [scopeFilter, setScopeFilter] = useState<string[]>([]); // [] = 全部实例
  const filterSections = useFilterSections();
  const scopeGroupSelectGroups = useMemo(
    () => toScopeGroupSelectGroups(filterSections),
    [filterSections],
  );
  const groupTreeSelectNodes = useMemo(
    () => toGroupTreeSelectNodes(filterSections),
    [filterSections],
  );
  // 当前 CLS 采集插件版本（用 state 触发红点显示/隐藏）
  const [currentPluginVersion, setCurrentPluginVersion] = useState<string>(
    () => localStorage.getItem("clsPluginVersion") || "v1",
  );

  // 跨页面同步 clsPluginVersion
  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === "clsPluginVersion" && e.newValue) {
        setCurrentPluginVersion(e.newValue);
      }
    };
    window.addEventListener("storage", handleStorageChange);
    return () => window.removeEventListener("storage", handleStorageChange);
  }, []);

  // 全局升级状态（来自 Context）
  const { status: upgradeStatus, start: startUpgrade } = usePluginUpgrade();

  // 升级成功后同步本页插件版本
  useEffect(() => {
    if (upgradeStatus === "succeeded") {
      setCurrentPluginVersion(LATEST_PLUGIN_VERSION);
    }
  }, [upgradeStatus]);

  // 点击"升级 CLS 采集插件"：直接走升级流程（无 Dialog / 无版本选择）
  const handleUpgradePlugin = () => {
    if (upgradeStatus === "running") return;
    if (currentPluginVersion === LATEST_PLUGIN_VERSION) {
      toast.info("CLS 采集插件已是最新版本");
      return;
    }
    startUpgrade();
  };

  const handleFromChange = (value: string) => {
    setDateFrom(value);
  };

  const handleToChange = (value: string) => {
    setDateTo(value);
  };

  const handleRefresh = () => {
    setRefreshing(true);
    setTimeout(() => { setRefreshing(false); }, 1000);
  };

  // 监听 localStorage 变化，实现跨页面同步
  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === "globalClsEnabled") {
        setClsEnabled(e.newValue === "true");
      }
    };
    window.addEventListener("storage", handleStorageChange);
    return () => window.removeEventListener("storage", handleStorageChange);
  }, []);

  // 监听 clsOpenClicked 标记，显示协议弹窗
  useEffect(() => {
    const checkClsOpen = () => {
      if (localStorage.getItem('clsOpenClicked') === 'true') {
        localStorage.removeItem('clsOpenClicked');
        setShowClsAgreementDialog(true);
      }
    };
    
    // 页面加载时检查
    checkClsOpen();
    
    // 监听 focus 事件
    window.addEventListener('focus', checkClsOpen);
    return () => window.removeEventListener('focus', checkClsOpen);
  }, []);

  const handleOpenCLS = () => {
    const isAuthorized = localStorage.getItem('clsAuthorized') === 'true';

    if (!isAuthorized) {
      setShowAuthDialog(true);
      setIsCheckingAuth(false);
      setAuthCompleted(false);
      return;
    }

    proceedWithClsSetup();
  };

  const proceedWithClsSetup = () => {
    // 显示免费额度 Dialog
    setShowFreeQuotaDialog(true);
    setFreeQuotaAgreed(false);
  };

  const handleGoToAuth = () => {
    // Mock 授权流程：5 秒后自动检测授权完成
    // 不真正打开腾讯云页面，而是模拟授权完成
    // 先显示检测状态
    setIsCheckingAuth(true);
    setAuthCompleted(false);
    
    setTimeout(() => {
      localStorage.setItem('clsAuthorized', 'true');
      // 检测完成，显示完成状态
      setIsCheckingAuth(false);
      setAuthCompleted(true);
      // 1秒后自动关闭Dialog并进入下一步
      setTimeout(() => {
        setShowAuthDialog(false);
        setAuthCompleted(false);
        proceedWithClsSetup();
      }, 1000);
    }, 5000);
  };

  const handleConfirmFreeQuota = () => {
    if (!freeQuotaAgreed) return;
    setShowFreeQuotaDialog(false);
    setIsEnablingCls(true);
    setTimeout(() => {
      setClsEnabled(true);
      localStorage.setItem('globalClsEnabled', 'true');
      // 开启 CLS：重置插件版本为 v1，触发"升级 CLS 采集插件"按钮的红点提示
      localStorage.setItem('clsPluginVersion', 'v1');
      setCurrentPluginVersion('v1');
      setIsEnablingCls(false);
      setShowSuccessMessage(true);
      setFreeQuotaAgreed(false);
      setTimeout(() => {
        setShowSuccessMessage(false);
      }, 3000);
    }, 1500);
  };

  const handleGoToCalcDetail = () => {
    window.open('https://cloud.tencent.com/document/product/614/45802', '_blank');
  };

  const handleCancelFreeQuota = () => {
    setShowFreeQuotaDialog(false);
    setFreeQuotaAgreed(false);
  };

  const handleConfirmClsAgreement = () => {
    if (!clsAgreed) return;
    setIsEnablingCls(true);
    // 模拟 loading 1.5 秒
    setTimeout(() => {
      setClsEnabled(true);
      localStorage.setItem('globalClsEnabled', 'true');
      // 开启 CLS：重置插件版本为 v1，触发"升级 CLS 采集插件"按钮的红点提示
      localStorage.setItem('clsPluginVersion', 'v1');
      setCurrentPluginVersion('v1');
      setIsEnablingCls(false);
      setShowSuccessMessage(true);
      setShowClsAgreementDialog(false);
      setClsAgreed(false);
      // 3 秒后隐藏成功提示
      setTimeout(() => {
        setShowSuccessMessage(false);
      }, 3000);
    }, 1500);
  };

  const handleCancelAuth = () => {
    setShowAuthDialog(false);
    setIsCheckingAuth(false);
    setAuthCompleted(false);
    if (authCheckInterval) {
      clearInterval(authCheckInterval);
      setAuthCheckInterval(null);
    }
  };

  const handleCloseCls = () => {
    setIsClosingCls(true);
    setTimeout(() => {
      setClsEnabled(false);
      localStorage.setItem("globalClsEnabled", "false");
      setIsClosingCls(false);
      setShowCloseClsConfirm(false);
      setDeleteLogTopic(false);
      const message = deleteLogTopic ? "CLS 日志服务已关闭，日志主题资源已删除" : "CLS 日志服务已关闭";
      // toast.success(message);
    }, 1000);
  };

  const handleCloseClsConfirmCancel = () => {
    setShowCloseClsConfirm(false);
    setDeleteLogTopic(false);
  };

  return (
    // data-billing-exempt: 停服态下运维观测保持正常可用（豁免全局停服禁用；元素自身 disabled 不受影响，延续禁用）
    <div className="page-enter" data-billing-exempt>
      <AdminPageHeader
        title="运维观测"
        description="提供全景式的 OpenClaw 运行观测能力，覆盖消息处理、队列健康、响应性能与工具调用全链路，为你的 AI Agent 业务构建可视化的稳定性保障。"
        descriptionClassName="leading-relaxed"
        actions={
          <div className="flex items-center gap-2" data-billing-exempt>
            <DatePicker value={dateFrom} onChange={handleFromChange} />
            <span className="text-[var(--text-weak)] text-sm">—</span>
            <DatePicker value={dateTo} onChange={handleToChange} />
            <Button
              variant="claw-outline"
              size="icon"
              onClick={handleRefresh}
              disabled={refreshing}
              title="刷新数据"
            >
              <RefreshCw className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
            </Button>
          </div>
        }
        className="mb-8"
      />

      {/* CLS 日志服务未开启提示 */}
      {!clsEnabled && (
        <>
          {/* CLS 提示弹框 */}
          <Alert variant="info" className="mb-6">
            <Info className="h-4 w-4" />
            <AlertDescription>
              <div className="flex items-start justify-between gap-6">
                <div className="flex-1">
                  <h3 className="text-sm font-semibold text-[var(--text-title)] mb-1">运维观测需要开启 CLS 日志服务</h3>
                  <p className="text-xs text-[var(--text-muted)]">开启后，为您赠送3个月ClawPro 专属 CLS 日志服务免费额度，预估可覆盖 500台 Agent 机器3个月的日志用量；服务到期后，CLS 将按量计费。<a href="https://cloud.tencent.com/document/product/614/45802" target="_blank" className="text-[var(--text-brand)] hover:underline inline-flex items-center gap-1">计费详情 <ArrowUpRight className="w-3 h-3" /></a></p>
                </div>
                <Button
                  variant="claw-primary"
                  onClick={handleOpenCLS}
                  disabled={isEnablingCls}
                  className="ml-4 text-xs h-8 px-4 whitespace-nowrap flex-shrink-0"
                >
                  {isEnablingCls ? "开启中..." : "开启 CLS 日志服务"}
                </Button>
              </div>

              {/* 开启范围选择（未选 = 采集全部实例） */}
              <div className="mt-4 pt-4 border-t border-[var(--alert-info-border)] flex items-center gap-3 flex-wrap">
                <span className="text-xs font-medium text-[var(--text-title)] flex-shrink-0">开启范围</span>
                <div className="w-60">
                  <GroupSelect
                    groups={scopeGroupSelectGroups}
                    selectedIds={scopeFilter}
                    onChange={setScopeFilter}
                    placeholder="选择开启范围"
                    sourceFilter={SCOPE_GROUP_SOURCE_FILTER}
                    sourceLabels={SCOPE_GROUP_SOURCE_LABELS}
                  />
                </div>
                <span className="text-xs text-[var(--text-muted)]">
                  未选择时将采集所有实例的日志，可能消耗较多 CLS 配额。
                </span>
              </div>
            </AlertDescription>
          </Alert>

          {/* CLS 协议确认弹窗 */}
          <Dialog open={showClsAgreementDialog} onOpenChange={setShowClsAgreementDialog}>
            <DialogContent className="sm:max-w-[560px]">
              <DialogHeader>
                <DialogTitle>确认免费额度</DialogTitle>
              </DialogHeader>
              <div className="space-y-4">
                <div className="flex items-start gap-3">
                  <Checkbox
                    id="cls-agreement"
                    checked={clsAgreed}
                    onCheckedChange={(checked) => setClsAgreed(checked === true)}
                    className="mt-1"
                  />
                  <Label htmlFor="cls-agreement" className="text-sm text-[var(--text-secondary)] cursor-pointer flex-1 font-normal leading-relaxed">
                    为您赠送三个月ClawPro 专属 CLS 日志服务免费额度，预估可覆盖 700 台 OpenClaw 机器的日志用量；服务到期后，CLS 将按量计费。<a href="https://cloud.tencent.com/document/product/614/45802" target="_blank" className="text-[var(--text-brand)] hover:underline inline-flex items-center gap-1">计费详情 <ArrowUpRight className="w-3 h-3" /></a>
                  </Label>
                </div>
              </div>
              <DialogFooter>
                <Button
                  variant="claw-outline"
                  onClick={() => {
                    setShowClsAgreementDialog(false);
                    setClsAgreed(false);
                  }}
                >
                  取消
                </Button>
                <Button
                  variant="dialog-confirm"
                  onClick={handleConfirmClsAgreement}
                  disabled={!clsAgreed || isEnablingCls}
                >
                  {isEnablingCls ? "开启中..." : "确认"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          {/* 卡片功能展示 - 当前页能力 + 其他页能力（对齐 main 两块 SurfaceCard 布局） */}
          <div className="space-y-4 mb-8">
            <SurfaceCard className="px-6 py-5">
              <h4 className="text-[14px] font-medium text-[var(--text-muted)] mb-4">开启CLS日志服务后您可以在此处获得以下观测数据：</h4>
              <div className="grid grid-cols-2 gap-x-6 gap-y-5">
                {EXISTING_OBSERVATION_CARDS.map((card) => (
                  <div key={card.id} className="flex items-center gap-[14px] py-5">
                    <img src={card.iconSrc} alt="" className="shrink-0 w-9 h-9" />
                    <div className="flex-1 min-w-0">
                      <h5 className="text-[14px] font-medium tracking-[0.005em] text-[var(--text-emphasis)] leading-[22px]">
                        {card.title}
                      </h5>
                      <p className="text-[12px] leading-[20px] tracking-[0.015em] text-[var(--text-muted)]">
                        {card.description}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            </SurfaceCard>

            <SurfaceCard className="px-6 py-5">
              <h4 className="text-[14px] font-medium text-[var(--text-muted)] mb-4">开启CLS日志服务后您还可以在Tokens监控和会话管理页面中获得以下观测数据：</h4>
              <div className="grid grid-cols-2 gap-x-6 gap-y-5">
                {CLS_NEW_CARDS.map((card) => (
                  <div key={card.id} className="flex items-center gap-[14px] py-5">
                    <img src={card.iconSrc} alt="" className="shrink-0 w-9 h-9" />
                    <div className="flex-1 min-w-0">
                      <h5 className="text-[14px] font-medium tracking-[0.005em] text-[var(--text-emphasis)] leading-[22px]">
                        {card.title}
                      </h5>
                      <p className="text-[12px] leading-[20px] tracking-[0.015em] text-[var(--text-muted)]">
                        {card.description}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            </SurfaceCard>
          </div>
        </>
      )}

      {/* CLS 采集插件升级：右上角进度 Toast，无 Dialog */}

      {/* CLS 开启成功提示 */}
      {showSuccessMessage && (
        <div className="fixed top-4 right-4 bg-[var(--alert-success-bg,#ECFDF5)] border border-[var(--alert-success-border,#A7F3D0)] rounded-[4px] px-4 py-3 shadow-[var(--shadow-overlay)] z-50 animate-in fade-in slide-in-from-top-2 max-w-md">
          <div className="flex items-start gap-3">
            <div className="w-5 h-5 bg-[var(--alert-success-icon,#10B981)] rounded-full flex items-center justify-center text-white flex-shrink-0 mt-0.5">
              <CheckCircle2 className="w-3.5 h-3.5" />
            </div>
            <div>
              <p className="text-sm font-medium text-[var(--alert-success-foreground,var(--foreground))]">CLS 日志服务开启成功</p>
            </div>
          </div>
        </div>
      )}

      {/* 已开启时显示筛选条 + 升级/关闭CLS按钮 */}
      {clsEnabled && (
        <div className="flex items-end justify-between mb-6 gap-4">
          {/* 左侧：分组、Agent */}
          <div className="flex items-end gap-4">
            <TreeSelect
              nodes={groupTreeSelectNodes}
              value={groupFilter}
              onChange={setGroupFilter}
              allLabel="全部分组"
              searchPlaceholder="搜索分组"
              triggerWidth={140}
            />
            <SearchableSelect
              options={[
                { value: "all", label: "全部 Agent" },
                { value: "Agent-A", label: "Agent-A" },
                { value: "Agent-B", label: "Agent-B" },
                { value: "Agent-C", label: "Agent-C" },
                { value: "Agent-D", label: "Agent-D" },
                { value: "Agent-E", label: "Agent-E" },
                { value: "Agent-F", label: "Agent-F" },
                { value: "Agent-G", label: "Agent-G" },
                { value: "Agent-H", label: "Agent-H" },
              ]}
              value={agentFilter || "all"}
              onChange={(v) => setAgentFilter(v === "all" ? "" : v)}
              placeholder="全部 Agent"
              searchPlaceholder="搜索 Agent..."
              showCount={false}
              triggerClassName="w-[160px] bg-white"
            />
          </div>

          {/* 右侧：开启范围 + 升级/关闭CLS按钮 */}
          <div className="flex items-center gap-2">
            <ScopeSelect
              scope={scopeFilter.length === 0 ? "all" : "groups"}
              selectedGroupIds={scopeFilter}
              groups={scopeGroupSelectGroups}
              showBadges={false}
              onConfirm={(scope, groupIds) => {
                setScopeFilter(scope === "all" ? [] : groupIds);
              }}
              trigger={
                <Button
                  variant="claw-outline"
                  className="h-9 px-2.5 gap-1.5 font-normal"
                >
                  <SlidersHorizontal className="w-3.5 h-3.5 text-[var(--text-secondary)]" />
                  <span className="text-xs font-medium text-[var(--text-title)]">开启范围</span>
                  {(() => {
                    if (scopeFilter.length === 0) {
                      return (
                        <Badge variant="secondary" className="ml-1 px-2 py-0.5 text-[10px] rounded-[4px]">
                          全部用户
                        </Badge>
                      );
                    }
                    const firstId = scopeFilter[0];
                    const firstName =
                      scopeGroupSelectGroups.find((g) => g.id === firstId)?.name || firstId;
                    const rest = scopeFilter.length - 1;
                    return (
                      <span className="inline-flex items-center gap-1 ml-1">
                        <Badge
                          variant="secondary"
                          className="px-2 py-0.5 text-[10px] rounded-[4px] max-w-[140px]"
                        >
                          <span className="block truncate max-w-[124px]">{firstName}</span>
                        </Badge>
                        {rest > 0 && (
                          <Badge
                            variant="secondary"
                            className="px-2 py-0.5 text-[10px] rounded-[4px]"
                          >
                            +{rest}
                          </Badge>
                        )}
                      </span>
                    );
                  })()}
                </Button>
              }
            />
            <Button
              onClick={handleUpgradePlugin}
              variant="claw-outline"
              className="relative h-9 text-xs px-3"
            >
              升级CLS采集插件
              {needsPluginUpgradeHint(clsEnabled, currentPluginVersion) && (
                <span className="absolute -top-1 -right-1 flex h-2.5 w-2.5">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-[var(--text-danger)] opacity-75"></span>
                  <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-[var(--text-danger)]"></span>
                </span>
              )}
            </Button>
            <Button
              onClick={() => setShowCloseClsConfirm(true)}
              variant="claw-outline"
              className="h-9 text-xs px-3 text-[var(--text-danger)] hover:bg-[var(--alert-warning-bg)]"
            >
              关闭CLS服务
            </Button>
          </div>
        </div>
      )}

      {/* ════════ 概览 ════════ */}
      {clsEnabled && (
        <>
      {/* ════════ 第一层：SLI 黄金指标（对齐 main：AGENT 总数 / 会话数量 / 消息响应 P95） ════════ */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        {/* AGENT 总数 */}
        <SurfaceCard
          onClick={() => navigate("/admin/openclaw-monitor")}
          className="p-4 cursor-pointer group"
        >
          <div className="flex items-center gap-2 mb-2">
            {METRIC_ICONS[0]}
            <p className="text-xs text-[var(--text-muted)]">AGENT 总数</p>
          </div>
          <StatNumber className="mb-2">22</StatNumber>
          <div className="flex items-center gap-2 text-[11px] text-[var(--text-muted)]">
            {(() => {
              const s = getTrendStyle('positive', 'up');
              return (
                <TrendBadge className={`${s.color} font-medium`}>环比 {s.arrow}2 台</TrendBadge>
              );
            })()}
          </div>
          <p className="text-[10px] text-[var(--text-weak)] mt-2 group-hover:text-[var(--text-brand)] transition-colors">查看 Agent 列表 →</p>
        </SurfaceCard>

        {/* 会话数量 */}
        <SurfaceCard
          onClick={() => navigate("/admin/session-management")}
          className="p-4 cursor-pointer group"
        >
          <div className="flex items-center gap-2 mb-2">
            {METRIC_ICONS[1]}
            <p className="text-xs text-[var(--text-muted)]">会话数量</p>
          </div>
          <StatNumber className="mb-2">1,286</StatNumber>
          <div className="flex items-center gap-2 text-[11px] text-[var(--text-muted)]">
            {(() => {
              const s = getTrendStyle('positive', 'up');
              return (
                <TrendBadge className={`${s.color} font-medium`}>环比 {s.arrow}5.6%</TrendBadge>
              );
            })()}
          </div>
          <p className="text-[10px] text-[var(--text-weak)] mt-2 group-hover:text-[var(--text-brand)] transition-colors">点击查看会话管理 →</p>
        </SurfaceCard>

        {/* 消息响应 P95 */}
        <SurfaceCard
          onClick={() => document.getElementById("section-perf")?.scrollIntoView({ behavior: "smooth" })}
          className="p-4 cursor-pointer group"
        >
          <div className="flex items-center gap-2 mb-2">
            {METRIC_ICONS[2]}
            <p className="text-xs text-[var(--text-muted)]">消息响应 P95</p>
          </div>
          <StatNumber className="mb-2">14.2s</StatNumber>
          <div className="flex items-center gap-2 text-[11px] text-[var(--text-muted)]">
            <span>P50 <span className="font-semibold text-[var(--text-secondary)]">3.8s</span></span>
            <span className="text-[var(--text-weak)]">·</span>
            <TrendBadge className="text-[var(--text-danger)] font-medium">环比 ↑0.8s</TrendBadge>
          </div>
          <p className="text-[10px] text-[var(--text-weak)] mt-2 group-hover:text-[var(--text-brand)] transition-colors">点击查看性能详情 →</p>
        </SurfaceCard>
      </div>

      {/* ════════ Tab 切换（对齐 main：消息概览 / 响应性能 / Skill & 工具） ════════ */}
      <div className="flex items-center gap-6 mb-6 border-b border-[var(--border)] pb-2">
        <button
          className={`text-sm pb-2 -mb-[10px] transition-colors ${activeTab === "message" ? "font-medium text-[var(--text-title)] border-b-2 border-[var(--text-title)]" : "text-[var(--text-muted)] hover:text-[var(--text-title)]"}`}
          onClick={() => setActiveTab("message")}
        >消息概览</button>
        <button
          className={`text-sm pb-2 -mb-[10px] transition-colors ${activeTab === "perf" ? "font-medium text-[var(--text-title)] border-b-2 border-[var(--text-title)]" : "text-[var(--text-muted)] hover:text-[var(--text-title)]"}`}
          onClick={() => setActiveTab("perf")}
        >响应性能</button>
        <button
          className={`text-sm pb-2 -mb-[10px] transition-colors ${activeTab === "skill" ? "font-medium text-[var(--text-title)] border-b-2 border-[var(--text-title)]" : "text-[var(--text-muted)] hover:text-[var(--text-title)]"}`}
          onClick={() => setActiveTab("skill")}
        >Skill & 工具</button>
      </div>

      {/* ════════ 第二层：消息对话 ════════ */}
      {activeTab === "message" && <>
      <div className="mb-6" id="section-message">
        <SurfaceCard className="p-6">
          <div className="flex items-center gap-2 mb-5">
            <p className="text-sm font-bold text-[var(--text-title)]">消息对话</p>
          </div>

          <div className="grid grid-cols-3 gap-4">
            {/* 消息总量（正向指标） */}
            {(() => {
              const s = getTrendStyle('positive', 'up');
              return (
                <SurfaceInner className="px-5 py-4">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-xs text-[var(--text-muted)]">消息总量</span>
                    <TrendBadge className={`text-[10px] font-medium ${s.color}`}>环比 {s.arrow}12.3%</TrendBadge>
                  </div>
                  <StatNumber className="text-3xl">8,432</StatNumber>
                </SurfaceInner>
              );
            })()}
            {/* 处理成功率（正向指标，持平） */}
            {(() => {
              const s = getTrendStyle('positive', 'flat');
              return (
                <SurfaceInner className="px-5 py-4">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-xs text-[var(--text-muted)]">处理成功率</span>
                    <TrendBadge className={`text-[10px] font-medium ${s.color}`}>环比 0%</TrendBadge>
                  </div>
                  <StatNumber className="text-3xl text-[var(--text-success)]">99.6 <span className="text-base font-bold text-[var(--text-muted)]">%</span></StatNumber>
                </SurfaceInner>
              );
            })()}
            {/* 平均处理时长（负向指标） */}
            {(() => {
              const s = getTrendStyle('negative', 'up');
              return (
                <SurfaceInner className="px-5 py-4">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-xs text-[var(--text-muted)]">平均处理时长</span>
                    <TrendBadge className={`text-[10px] font-medium ${s.color}`}>环比 {s.arrow}0.5s</TrendBadge>
                  </div>
                  <StatNumber className="text-3xl">4.8<span className="text-base font-bold text-[var(--text-muted)] ml-0.5">s</span></StatNumber>
                </SurfaceInner>
              );
            })()}
          </div>
        </SurfaceCard>
      </div>

      {/* ════════ 第三层：消息队列状态 ════════ */}
      <div className="mb-6" id="section-queue">
        <SurfaceCard className="p-6">
          <div className="flex items-center gap-2 mb-5">
            <p className="text-sm font-bold text-[var(--text-title)]">消息队列状态</p>
          </div>

          {/* 3 个指标卡片（含环比） */}
          <div className="grid grid-cols-3 gap-4">
            {/* 入队速率（正向） */}
            {(() => {
              const s = getTrendStyle('positive', 'up');
              return (
                <SurfaceInner className="px-5 py-4">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-xs text-[var(--text-muted)]">入队速率/min</span>
                    <TrendBadge className={`text-[10px] font-medium ${s.color}`}>环比 {s.arrow}8.4%</TrendBadge>
                  </div>
                  <StatNumber className="text-3xl">162</StatNumber>
                </SurfaceInner>
              );
            })()}
            {/* 处理速率（正向） */}
            {(() => {
              const s = getTrendStyle('positive', 'up');
              return (
                <SurfaceInner className="px-5 py-4">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-xs text-[var(--text-muted)]">处理速率/min</span>
                    <TrendBadge className={`text-[10px] font-medium ${s.color}`}>环比 {s.arrow}15.2%</TrendBadge>
                  </div>
                  <StatNumber className="text-3xl">191</StatNumber>
                </SurfaceInner>
              );
            })()}
            {/* 平均队列深度（负向：上升为红）— 改回单值卡，去掉 sparkline；下方独立区域展示实时折线 */}
            {(() => {
              const s = getTrendStyle('negative', 'up');
              return (
                <SurfaceInner className="px-5 py-4">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-xs text-[var(--text-muted)]">平均队列深度</span>
                    <TrendBadge className={`text-[10px] font-medium ${s.color}`}>环比 {s.arrow}3</TrendBadge>
                  </div>
                  <StatNumber className="text-3xl">8.2</StatNumber>
                </SurfaceInner>
              );
            })()}
          </div>

          {/* 实时队列深度趋势（00:00 - 12:00） */}
          <div className="mt-6">
            <div className="flex items-center justify-between mb-3">
              <p className="text-xs font-semibold text-[var(--text-secondary)]">实时队列深度</p>
              <span className="text-[10px] text-[var(--text-weak)]">00:00 - 12:00</span>
            </div>
            <ResponsiveContainer width="100%" height={180}>
              <LineChart
                data={(() => {
                  // 00:00 - 12:00 每 5 分钟一个采样点，共 145 个点
                  const data: { time: string; value: number }[] = [];
                  for (let m = 0; m <= 12 * 60; m += 5) {
                    const hh = String(Math.floor(m / 60)).padStart(2, '0');
                    const mm = String(m % 60).padStart(2, '0');
                    // mock 队列深度：早间低谷、上午波动、接近 12 点缓慢上升
                    const hour = m / 60;
                    let base = 4 + Math.sin(hour * 0.7) * 3 + Math.cos(hour * 1.3) * 2;
                    if (hour > 8) base += (hour - 8) * 2.2; // 工作时段开始堆积
                    const noise = (Math.sin(m * 0.13) + Math.cos(m * 0.27)) * 1.5;
                    const value = Math.max(0, Math.round((base + noise) * 10) / 10);
                    data.push({ time: `${hh}:${mm}`, value });
                  }
                  return data;
                })()}
                margin={{ top: 8, right: 16, left: 0, bottom: 0 }}
              >
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" vertical={false} />
                <XAxis
                  dataKey="time"
                  tick={{ fontSize: 10, fill: "#94a3b8" }}
                  axisLine={{ stroke: "#e2e8f0" }}
                  tickLine={false}
                  interval={23}
                />
                <YAxis
                  tick={{ fontSize: 10, fill: "#94a3b8" }}
                  axisLine={false}
                  tickLine={false}
                  width={36}
                  allowDecimals={false}
                />
                <Tooltip
                  contentStyle={{ fontSize: 11, borderRadius: 8, border: "none", boxShadow: "0 4px 12px rgba(0,0,0,0.1)" /* allow-shadow */ }}
                  formatter={(v: number) => [`${v}`, "队列深度"]}
                  labelFormatter={(label) => `时间 ${label}`}
                />
                <Line
                  type="monotone"
                  dataKey="value"
                  stroke="#7C3AED"
                  strokeWidth={1.5}
                  dot={false}
                  isAnimationActive={false}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </SurfaceCard>
      </div>

      {/* ════════ 消息分布 ════════ */}
      <div className="mb-6">
        <SurfaceCard className="p-6">
          <div className="grid grid-cols-2 gap-4">
            {/* 消息来源分布 */}
            <div className="flex flex-col items-center">
              <p className="text-xs font-semibold text-gray-700 mb-2 self-start">消息来源分布</p>
              <ResponsiveContainer width="100%" height={220}>
                <PieChart>
                  <Pie
                    data={[
                      { name: "system", value: 505 },
                      { name: "user", value: 505 },
                    ]}
                    cx="50%" cy="45%" innerRadius={55} outerRadius={80}
                    paddingAngle={3} dataKey="value" strokeWidth={2} stroke="#fff"
                    label={({ name, value, percent, cx: pcx, cy: pcy, midAngle, outerRadius: or }) => {
                      const RADIAN = Math.PI / 180;
                      const radius = (or || 80) + 22;
                      const x = (pcx || 0) + radius * Math.cos(-midAngle * RADIAN);
                      const y = (pcy || 0) + radius * Math.sin(-midAngle * RADIAN);
                      return <text x={x} y={y} fill="#64748b" textAnchor={x > (pcx || 0) ? "start" : "end"} dominantBaseline="central" fontSize={10}>{`${name} ${value} (${(percent! * 100).toFixed(0)}%)`}</text>;
                    }}
                    labelLine={{ stroke: "#cbd5e1", strokeWidth: 1 }}
                  >
                    {["#3B82F6", "#7C3AED"].map((color, i) => (
                      <Cell key={i} fill={color} />
                    ))}
                  </Pie>
                  <Tooltip contentStyle={{ fontSize: 11, borderRadius: 8, border: "none", boxShadow: "0 4px 12px rgba(0,0,0,0.1)" /* allow-shadow */ }} formatter={(v: number, name: string) => [`${v}`, name]} />
                  <Legend iconType="circle" iconSize={8} wrapperStyle={{ fontSize: 11, paddingTop: 8 }} />
                </PieChart>
              </ResponsiveContainer>
            </div>

            {/* 消息渠道分布 */}
            <div className="flex flex-col items-center">
              <p className="text-xs font-semibold text-gray-700 mb-2 self-start">消息渠道分布</p>
              <ResponsiveContainer width="100%" height={220}>
                <PieChart>
                  <Pie
                    data={[
                      { name: "qqbot", value: 523 },
                      { name: "webchat", value: 289 },
                      { name: "lightclawbot", value: 198 },
                      { name: "openclaw-wecom-bot", value: 156 },
                    ]}
                    cx="50%" cy="45%" innerRadius={55} outerRadius={80}
                    paddingAngle={3} dataKey="value" strokeWidth={2} stroke="#fff"
                    label={({ name, value, percent, cx: pcx, cy: pcy, midAngle, outerRadius: or }) => {
                      const RADIAN = Math.PI / 180;
                      const radius = (or || 80) + 22;
                      const x = (pcx || 0) + radius * Math.cos(-midAngle * RADIAN);
                      const y = (pcy || 0) + radius * Math.sin(-midAngle * RADIAN);
                      return <text x={x} y={y} fill="#64748b" textAnchor={x > (pcx || 0) ? "start" : "end"} dominantBaseline="central" fontSize={10}>{`${name} ${value} (${(percent! * 100).toFixed(0)}%)`}</text>;
                    }}
                    labelLine={{ stroke: "#cbd5e1", strokeWidth: 1 }}
                  >
                    {["#3B82F6", "#D97706", "#059669", "#7C3AED"].map((color, i) => (
                      <Cell key={i} fill={color} />
                    ))}
                  </Pie>
                  <Tooltip contentStyle={{ fontSize: 11, borderRadius: 8, border: "none", boxShadow: "0 4px 12px rgba(0,0,0,0.1)" /* allow-shadow */ }} formatter={(v: number, name: string) => [`${v}`, name]} />
                  <Legend iconType="circle" iconSize={8} wrapperStyle={{ fontSize: 11, paddingTop: 8 }} />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </div>
        </SurfaceCard>
      </div>
      </>}

      {/* ════════ 响应性能 — "慢在哪？" ════════ */}
      {activeTab === "perf" && <div className="mt-0" id="section-perf">
        {/* 顶部 KPI 横排（4 张卡：流量 R · 延迟 D · 错误率 E · 稳定性） */}
        <div className="grid grid-cols-4 gap-4 mb-4">
          {([
            {
              label: "Webhook 请求数",
              value: "8,432",
              unit: "",
              trendValue: "12.3%",
              polarity: "positive",
              delta: "up",
              tip: "渠道 Webhook 收到的入站请求总数（含按钮回调、心跳）",
            },
            {
              label: "消息响应耗时",
              value: "4.5",
              unit: "s",
              trendValue: "0.3s",
              polarity: "negative",
              delta: "down",
              tip: "用户消息从入队到处理完成的端到端平均耗时",
            },
            {
              label: "Webhook 错误率",
              value: "0.12",
              unit: "%",
              trendValue: "0.05%",
              polarity: "negative",
              delta: "down",
              tip: "Webhook 处理失败请求数占总入站请求数的比例",
            },
            {
              label: "网关启动次数",
              value: "3",
              unit: "次",
              trendValue: "70.8%",
              polarity: "negative",
              delta: "down",
              tip: "近期网关进程重启次数（重启越少越稳定）",
            },
          ] as const).map((card) => {
            const s = getTrendStyle(card.polarity, card.delta);
            return (
              <SurfaceCard
                key={card.label}
                className="p-5"
                title={card.tip}
              >
                <p className="text-xs text-[var(--text-muted)] mb-2">{card.label}</p>
                <div className="flex items-end justify-between">
                  <StatNumber className="text-2xl">
                    {card.value}
                    {card.unit && <span className="text-sm font-bold text-[var(--text-muted)] ml-0.5">{card.unit}</span>}
                  </StatNumber>
                  <TrendBadge className={`text-[11px] font-medium ${s.color}`}>
                    环比 {s.arrow}{card.trendValue}
                  </TrendBadge>
                </div>
              </SurfaceCard>
            );
          })}
        </div>

        <div className="grid grid-cols-2 gap-4">
          {/* ① Webhook 请求数趋势（柱状）— 与顶部 KPI #1 对应 */}
          <SurfaceCard className="p-5">
            <div className="flex items-center justify-between mb-4">
              <p className="text-xs font-semibold text-gray-700">Webhook 请求数趋势</p>
            </div>
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={[
                { time: "00:00", count: 28 },
                { time: "03:00", count: 14 },
                { time: "06:00", count: 9 },
                { time: "09:00", count: 156 },
                { time: "12:00", count: 248 },
                { time: "15:00", count: 192 },
                { time: "18:00", count: 137 },
                { time: "21:00", count: 86 },
                { time: "23:59", count: 42 },
              ]}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="time" tick={{ fontSize: 11, fill: "#94a3b8" }} axisLine={{ stroke: "#e2e8f0" }} tickLine={false} />
                <YAxis tick={{ fontSize: 11, fill: "#94a3b8" }} axisLine={false} tickLine={false} />
                <Tooltip contentStyle={{ fontSize: 12, borderRadius: 10, border: "none", boxShadow: "0 4px 20px rgba(0,0,0,0.12)" /* allow-shadow */ }} formatter={(v: number) => [`${v} 次`, "Webhook 请求数"]} />
                <Bar dataKey="count" name="Webhook 请求数" fill="#3B82F6" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </SurfaceCard>

          {/* ② 各环节耗时对比 — 与顶部 KPI #2「消息响应耗时」配对（"用户等多久" + "钱花哪儿"） */}
          <SurfaceCard className="p-5">
            <div className="flex items-center justify-between mb-4">
              <p className="text-xs font-semibold text-gray-700">各环节平均耗时</p>
            </div>
            <div className="space-y-3">
              {[
                { name: "Agent 编排", sub: "invoke_agent · 22 次", avg: 59532, color: "#3B82F6" },
                { name: "STEP 单轮", sub: "react step · 127 次", avg: 8010, color: "#7C3AED" },
                { name: "工具执行", sub: "execute_tool · 106 次", avg: 3045, color: "#D97706" },
                { name: "LLM 模型调用", sub: "chat tc-code-latest · 127 次", avg: 44, color: "#059669" },
              ].map((item) => {
                const maxVal = 65000;
                const pct = Math.min((item.avg / maxVal) * 100, 100);
                return (
                  <div key={item.name}>
                    <div className="flex items-center justify-between mb-1">
                      <div>
                        <span className="text-xs text-gray-600">{item.name}</span>
                        <span className="text-[10px] text-gray-400 ml-2">{item.sub}</span>
                      </div>
                      <span className="text-xs font-mono font-semibold text-gray-700">
                        {item.avg >= 1000 ? `${(item.avg / 1000).toFixed(1)}s` : `${item.avg}ms`}
                      </span>
                    </div>
                    <div className="h-2.5 bg-gray-100 rounded-full overflow-hidden">
                      <div className="h-full rounded-full transition-all" style={{ width: `${pct}%`, backgroundColor: item.color }} />
                    </div>
                  </div>
                );
              })}
            </div>
          </SurfaceCard>

          {/* ③ 消息响应耗时趋势（P50 / P95 / P99）— 顶部 KPI #2 的历史展开 */}
          <SurfaceCard className="p-5">
            <div className="flex items-center justify-between mb-4">
              <p className="text-xs font-semibold text-gray-700">消息响应耗时趋势（P50 / P95 / P99）</p>
            </div>
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={[
                { time: "00:00", p50: 3.2, p95: 8.2, p99: 11.5 },
                { time: "03:00", p50: 2.5, p95: 6.5, p99: 9.2 },
                { time: "06:00", p50: 2.2, p95: 5.8, p99: 8.4 },
                { time: "09:00", p50: 4.5, p95: 12.4, p99: 17.6 },
                { time: "12:00", p50: 5.1, p95: 15.6, p99: 22.3 },
                { time: "15:00", p50: 4.8, p95: 13.2, p99: 18.9 },
                { time: "18:00", p50: 3.8, p95: 10.8, p99: 15.4 },
                { time: "21:00", p50: 3.2, p95: 8.9, p99: 12.7 },
                { time: "23:59", p50: 2.9, p95: 7.5, p99: 10.3 },
              ]}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="time" tick={{ fontSize: 11, fill: "#94a3b8" }} axisLine={{ stroke: "#e2e8f0" }} tickLine={false} />
                <YAxis tick={{ fontSize: 11, fill: "#94a3b8" }} axisLine={false} tickLine={false} tickFormatter={(v) => `${v}s`} />
                <Tooltip contentStyle={{ fontSize: 12, borderRadius: 10, border: "none", boxShadow: "0 4px 20px rgba(0,0,0,0.12)" /* allow-shadow */ }} formatter={(v: number) => [`${v}s`]} />
                <Legend wrapperStyle={{ fontSize: 11, color: "#64748b" }} />
                <Line type="monotone" dataKey="p50" name="P50 中位数" stroke="#D97706" strokeWidth={2} dot={false} />
                <Line type="monotone" dataKey="p95" name="P95" stroke="#E11D48" strokeWidth={2} dot={{ r: 2 }} />
                <Line type="monotone" dataKey="p99" name="P99" stroke="#9333EA" strokeWidth={2} dot={{ r: 2 }} />
              </LineChart>
            </ResponsiveContainer>
          </SurfaceCard>

          {/* ④ Webhook 错误率趋势（折线，RED 三件套之 E）— 与顶部 KPI #3 对应 */}
          <SurfaceCard className="p-5">
            <div className="flex items-center justify-between mb-4">
              <p className="text-xs font-semibold text-gray-700">Webhook 错误率趋势</p>
            </div>
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={[
                { time: "00:00", rate: 0.05 },
                { time: "03:00", rate: 0.04 },
                { time: "06:00", rate: 0.03 },
                { time: "09:00", rate: 0.18 },
                { time: "12:00", rate: 0.32 },
                { time: "15:00", rate: 0.21 },
                { time: "18:00", rate: 0.15 },
                { time: "21:00", rate: 0.09 },
                { time: "23:59", rate: 0.06 },
              ]}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="time" tick={{ fontSize: 11, fill: "#94a3b8" }} axisLine={{ stroke: "#e2e8f0" }} tickLine={false} />
                <YAxis tick={{ fontSize: 11, fill: "#94a3b8" }} axisLine={false} tickLine={false} tickFormatter={(v) => `${v}%`} />
                <Tooltip
                  contentStyle={{ fontSize: 12, borderRadius: 10, border: "none", boxShadow: "0 4px 20px rgba(0,0,0,0.12)" /* allow-shadow */ }}
                  formatter={(v: number) => [`${v}%`, "错误率"]}
                />
                <Line type="monotone" dataKey="rate" name="错误率" stroke="#E11D48" strokeWidth={2} dot={{ r: 2, fill: "#E11D48" }} />
              </LineChart>
            </ResponsiveContainer>
          </SurfaceCard>
        </div>
      </div>}

      {/* ════════ Skill & 工具调用 ════════ */}
      {activeTab === "skill" && <div className="mt-0">
        {/* 四个核心数字 */}
        <div className="grid grid-cols-4 gap-4 mb-4">
          {([
            { label: "Skill 访问次数", value: "123", trendValue: "12%", polarity: "positive", delta: "up" },
            { label: "工具执行次数", value: "847", trendValue: "8%", polarity: "positive", delta: "up" },
            { label: "工具执行错误次数", value: "23", trendValue: "5%", polarity: "negative", delta: "down" },
            { label: "工具执行平均耗时", value: "1.8s", trendValue: "0.3s", polarity: "negative", delta: "up" },
          ] as const).map((card) => {
            const s = getTrendStyle(card.polarity, card.delta);
            return (
              <SurfaceCard key={card.label} className="p-4">
                <p className="text-[11px] font-semibold text-[var(--text-muted)] uppercase tracking-wide mb-2">{card.label}</p>
                <div className="flex items-end justify-between">
                  <StatNumber className="text-2xl">{card.value}</StatNumber>
                  <TrendBadge className={`text-xs font-medium ${s.color}`}>
                    环比 {s.arrow}{card.trendValue}
                  </TrendBadge>
                </div>
              </SurfaceCard>
            );
          })}
        </div>

        {/* 四个 Top 排行 */}
        <div className="grid grid-cols-2 gap-4">
          {/* Skill 访问 Top */}
          <SurfaceCard className="p-5">
            <p className="text-xs font-semibold text-gray-700 mb-3">Skill 访问 Top 10</p>
            <div className="space-y-2">
              {[
                { name: "superpowers", count: 48 },
                { name: "tea-d2c", count: 32 },
                { name: "tea-component-mapper", count: 18 },
                { name: "cls需求文档编写助手", count: 12 },
                { name: "product-prototype-design", count: 8 },
                { name: "skill-creator", count: 5 },
              ].map((s, i) => (
                <div key={s.name} className="flex items-center gap-3">
                  <span className="text-[10px] text-gray-400 font-mono w-4 text-right">{i + 1}</span>
                  <div className="flex-1">
                    <div className="flex items-center justify-between mb-0.5">
                      <span className="text-xs text-gray-700 truncate">{s.name}</span>
                      <span className="text-xs font-mono text-gray-500 ml-2">{s.count}</span>
                    </div>
                    <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
                      <div className="h-full bg-violet-400 rounded-full" style={{ width: `${(s.count / 48) * 100}%` }} />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </SurfaceCard>

          {/* 工具执行 Top */}
          <SurfaceCard className="p-5">
            <p className="text-xs font-semibold text-gray-700 mb-3">工具执行 Top 10</p>
            <div className="space-y-2">
              {[
                { name: "exec_command", count: 312 },
                { name: "read_file", count: 198 },
                { name: "write_file", count: 145 },
                { name: "web_fetch", count: 89 },
                { name: "search_content", count: 56 },
                { name: "list_dir", count: 47 },
              ].map((s, i) => (
                <div key={s.name} className="flex items-center gap-3">
                  <span className="text-[10px] text-gray-400 font-mono w-4 text-right">{i + 1}</span>
                  <div className="flex-1">
                    <div className="flex items-center justify-between mb-0.5">
                      <span className="text-xs text-gray-700 font-mono truncate">{s.name}</span>
                      <span className="text-xs font-mono text-gray-500 ml-2">{s.count}</span>
                    </div>
                    <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
                      <div className="h-full bg-blue-400 rounded-full" style={{ width: `${(s.count / 312) * 100}%` }} />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </SurfaceCard>

          {/* 工具错误 Top */}
          <SurfaceCard className="p-5">
            <p className="text-xs font-semibold text-gray-700 mb-3">工具执行发生错误 Top 10</p>
            <div className="space-y-2">
              {[
                { name: "exec_command", count: 12, errorRate: "3.8%" },
                { name: "web_fetch", count: 6, errorRate: "6.7%" },
                { name: "write_file", count: 3, errorRate: "2.1%" },
                { name: "search_content", count: 2, errorRate: "3.6%" },
              ].map((s, i) => (
                <div key={s.name} className="flex items-center gap-3">
                  <span className="text-[10px] text-gray-400 font-mono w-4 text-right">{i + 1}</span>
                  <div className="flex-1">
                    <div className="flex items-center justify-between mb-0.5">
                      <span className="text-xs text-gray-700 font-mono truncate">{s.name}</span>
                      <div className="flex items-center gap-2 ml-2">
                        <span className="text-xs font-mono text-red-600 font-semibold">{s.count}</span>
                        <span className="text-[10px] text-gray-400">({s.errorRate})</span>
                      </div>
                    </div>
                    <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
                      <div className="h-full bg-red-400 rounded-full" style={{ width: `${(s.count / 12) * 100}%` }} />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </SurfaceCard>

          {/* 工具平均耗时趋势 */}
          <SurfaceCard className="p-5">
            <p className="text-xs font-semibold text-gray-700 mb-3">工具执行平均耗时趋势</p>
            <ResponsiveContainer width="100%" height={165}>
              <LineChart data={[
                { time: "03-25", avg: 1.2 }, { time: "03-26", avg: 1.4 },
                { time: "03-27", avg: 1.1 }, { time: "03-28", avg: 1.6 },
                { time: "03-29", avg: 1.3 }, { time: "03-30", avg: 2.1 },
                { time: "03-31", avg: 1.8 },
              ]}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="time" tick={{ fontSize: 10, fill: "#94a3b8" }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 10, fill: "#94a3b8" }} axisLine={false} tickLine={false} tickFormatter={(v) => `${v}s`} />
                <Tooltip contentStyle={{ fontSize: 11, borderRadius: 8, border: "none", boxShadow: "0 4px 12px rgba(0,0,0,0.1)" /* allow-shadow */ }} formatter={(v: number) => [`${v}s`, "平均耗时"]} />
                <Line type="monotone" dataKey="avg" stroke="#7C3AED" strokeWidth={2} dot={{ r: 3 }} />
              </LineChart>
            </ResponsiveContainer>
          </SurfaceCard>
        </div>
      </div>}

        </>
      )}


      {/* CLS 授权 Dialog */}
      <Dialog
        open={showAuthDialog}
        onOpenChange={(open) => {
          if (open) setShowAuthDialog(true);
          else handleCancelAuth();
        }}
      >
        <DialogContent size="md">
          <DialogHeader>
            <DialogTitle>开通服务授权</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6">
            <div className="space-y-4">
              <DialogDescription className="text-sm leading-6 text-[var(--text-secondary)]">
                开启 CLS 日志服务前，需要完成腾讯云服务角色授权。授权完成后系统会自动检测状态，并继续开通服务。
              </DialogDescription>

              <SurfaceInner className="bg-[var(--bg-grey-hover)] px-4 py-4">
                {isCheckingAuth ? (
                  <div className="flex items-center gap-3">
                    <span className="flex w-9 h-9 items-center justify-center rounded-full border border-[var(--border)] bg-white shrink-0">
                      <RefreshCw className="w-4 h-4 animate-spin text-[var(--text-brand)]" />
                    </span>
                    <div className="space-y-1">
                      <p className="text-sm font-medium text-[var(--text-title)]">正在检测授权状态</p>
                      <p className="text-xs leading-5 text-[var(--text-muted)]">请在授权页面完成操作，完成后返回本页面。</p>
                    </div>
                  </div>
                ) : authCompleted ? (
                  <div className="flex items-center gap-3">
                    <span className="flex w-9 h-9 items-center justify-center rounded-full border border-[var(--alert-info-border)] bg-white shrink-0">
                      <CheckCircle2 className="w-5 h-5 text-[var(--text-brand)]" />
                    </span>
                    <div className="space-y-1">
                      <p className="text-sm font-medium text-[var(--text-title)]">检测到已授权</p>
                      <p className="text-xs leading-5 text-[var(--text-muted)]">正在为您继续开通 CLS 日志服务。</p>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center gap-3">
                    <span className="flex w-9 h-9 items-center justify-center rounded-full border border-[var(--alert-info-border)] bg-white shrink-0">
                      <Info className="w-5 h-5 text-[var(--text-brand)]" />
                    </span>
                    <div className="space-y-1">
                      <p className="text-sm font-medium text-[var(--text-title)]">授权后可获取会话与运维观测数据</p>
                      <p className="text-xs leading-5 text-[var(--text-muted)]">点击“前往授权”后，请在新页面完成授权操作。</p>
                    </div>
                  </div>
                )}
              </SurfaceInner>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button variant="claw-outline" onClick={handleCancelAuth}>
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              onClick={handleGoToAuth}
              disabled={isCheckingAuth || authCompleted}
            >
              前往授权
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 免费额度 Dialog */}
      <Dialog open={showFreeQuotaDialog} onOpenChange={setShowFreeQuotaDialog}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>免费额度说明</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 my-4">
            <Alert variant="info">
              <Info className="h-4 w-4" />
              <AlertDescription>
                为您赠送<span className="font-semibold text-[var(--text-brand)]">3个月</span>ClawPro 专属 CLS 日志服务免费额度（共<span className="font-semibold text-[var(--text-brand)]">3000U</span>），预估可覆盖 <span className="font-semibold text-[var(--text-brand)]">500台</span> OpenClaw 机器<span className="font-semibold text-[var(--text-brand)]">3个月</span>的日志用量；超过免费额度达到上限或<span className="font-semibold text-[var(--text-brand)]">3个月</span>到期后，CLS 将按量计费。计费详情请参考{' '}
                <a
                  href="#"
                  onClick={(e) => {
                    e.preventDefault();
                    handleGoToCalcDetail();
                  }}
                  className="text-[var(--text-brand)] hover:underline"
                >
                  计费详情
                </a>
                。
              </AlertDescription>
            </Alert>
            <div className="flex items-center gap-3">
              <Checkbox
                id="free-quota-agreement"
                checked={freeQuotaAgreed}
                onCheckedChange={(checked) => setFreeQuotaAgreed(checked === true)}
              />
              <Label htmlFor="free-quota-agreement" className="text-sm text-[var(--text-secondary)] cursor-pointer font-normal">我已阅读并同意免费额度说明</Label>
            </div>
          </div>
          <DialogFooter className="flex gap-2 justify-end">
            <Button variant="claw-outline" onClick={handleCancelFreeQuota}>
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              onClick={handleConfirmFreeQuota}
              disabled={!freeQuotaAgreed}
            >
              确认
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 关闭CLS确认对话框 */}
      <AlertDialog open={showCloseClsConfirm} onOpenChange={setShowCloseClsConfirm}>
        <AlertDialogContent className="sm:max-w-[560px]">
          <AlertDialogHeader>
            <AlertDialogTitle>确定要关闭 CLS 日志服务吗？</AlertDialogTitle>
            <AlertDialogDescription>
              关闭后以下功能将无法使用，
              <span className="text-[var(--text-danger)]">此操作可能影响业务运行</span>。
            </AlertDialogDescription>
          </AlertDialogHeader>

          {/* Body：与 Header / Footer 平级，不再塞入 Description */}
          <div className="space-y-4 py-2">
            <Alert variant="warning">
              <AlertOperationInfoIcon />
              <AlertTitle>受影响的功能</AlertTitle>
              <AlertDescription>
                <ul className="list-disc pl-4 space-y-1">
                  <li><span className="font-medium">运维观测：</span>支持通过全链路性能监控采集核心运行指标</li>
                  <li><span className="font-medium">会话管理：</span>支持通过会话总览、会话链下钻还原及渠道模型分布分析</li>
                  <li><span className="font-medium">Tokens 监控（按会话）：</span>支持从按会话、消息维度查看 tokens、费用使用情况</li>
                </ul>
              </AlertDescription>
            </Alert>

            <div className="flex items-start gap-2">
              <Checkbox
                id="deleteLogTopic"
                checked={deleteLogTopic}
                onCheckedChange={(checked) => setDeleteLogTopic(checked === true)}
                className="mt-0.5"
              />
              <div className="flex-1 space-y-1">
                <Label htmlFor="deleteLogTopic" className="text-sm font-medium cursor-pointer">
                  同时删除关联的日志主题资源
                </Label>
                <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
                  勾选后将永久删除该日志主题及所有日志数据，
                  <span className="text-[var(--text-danger)]">数据不可恢复</span>
                  ；未删除则会持续产生存储费用。
                </p>
              </div>
            </div>
          </div>

          <AlertDialogFooter>
            <AlertDialogCancel onClick={handleCloseClsConfirmCancel}>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleCloseCls} disabled={isClosingCls}>
              {isClosingCls ? "关闭中…" : "确定关闭"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
