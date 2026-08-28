/**
 * SessionManagement - 会话管理页面
 * 左侧：筛选面板（OpenClaw 名称、Session ID、标签、范围筛选）
 * 右侧：表格（OpenClaw 名称、ID、创建时间、Input、Output、耗时、Trace 数、Token）
 */
import { useState, useMemo, useEffect } from "react";
import { useLocation, useSearch } from "wouter";
import {
  Search, PanelLeftClose, PanelLeft, ChevronUp, ChevronDown, Settings2,
  RefreshCw, ArrowLeftRight, ArrowUpRight, SlidersHorizontal,
  Download, AlertTriangle, Info, CheckCircle2, ArrowUp, ArrowDown, X,
} from "lucide-react";
import {
  BarChart, Bar, Cell, XAxis, YAxis, CartesianGrid, Tooltip as ReTooltip, ResponsiveContainer,
  PieChart, Pie,
} from "recharts";
import { Tooltip as UITooltip, TooltipContent as UITooltipContent, TooltipTrigger as UITooltipTrigger } from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { StatusTag } from "@/components/ui/status-tag";
import { SearchableSelect } from "@/components/ui/select";
import {
  Dialog, DialogContent, DialogBody, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuCheckboxItem,
} from "@/components/ui/dropdown-menu";
import { GroupSelect } from "@/components/GroupSelect";
import { ScopeSelect } from "@/components/ScopeSelect";
import { TreeSelect, type TreeSelectNode } from "@/components/ui/tree-select";
import { useFilterSections } from "@/lib/clsScopeMock";
import type { FilterSection, TreeNodeData } from "@/components/groupTreeShared";
import type { GroupSource, UserGroup } from "@/pages/admin/MemberManagement/types";
import { Alert, AlertDescription, AlertTitle, AlertOperationInfoIcon } from "@/components/ui/alert";
import { SurfaceCard, SurfaceInner } from "@/components/ui/Surface";
import { StatNumber, PanelTitle, MetaMedium } from "@/components/ui/Typography";
import { Badge } from "@/components/ui/badge";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { DatePicker } from "@/components/ui/date-picker";
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription, EmptyContent } from "@/components/ui/empty";
import { Pagination } from "@/components/ui/pagination";
import { useSessionManagementCalendarBillingExempt } from "./SessionManagement/useSessionManagementCalendarBillingExempt";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  TableActionCell,
} from "@/components/ui/table";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from "@/components/ui/alert-dialog";
import { toast } from "sonner";
import {
  sessionsData, fmtTokens, getSessionTraces,
} from "@/data/trace-data";
import {
  usePluginUpgrade,
  needsPluginUpgradeHint,
  LATEST_PLUGIN_VERSION,
} from "@/contexts/PluginUpgradeContext";

// ─── 旧版会话管理 mock 数据（与 openclaw-enterprise 旧版一致） ───
const LEGACY_STAT_CARDS = [
  { label: "总会话数", value: 11, metric: "total_sessions" },
  { label: "平均轮次", value: "28.6", metric: "avg_rounds" },
  { label: "工具调用", value: 206, metric: "tool_calls" },
  { label: "活跃渠道", value: 0, metric: "active_channels" },
];

/** 4 个设计系统标准 SVG icon（渐变黑→蓝，与 OpsObservation / TokensMonitor 统一风格；复用 design-refresh-2026 实现） */
const LEGACY_STAT_ICONS: React.ReactNode[] = [
  /* 对话气泡 - 总会话数 */
  <svg key="s0" width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M9 1.6875C4.96992 1.6875 1.6875 4.66172 1.6875 8.32031C1.6875 9.83391 2.24297 11.2289 3.18516 12.3539C2.93437 13.3664 2.27109 14.2664 2.26055 14.2805C2.07187 14.4844 2.02266 14.7797 2.13633 15.0398C2.25 15.3 2.50781 15.4688 2.79141 15.4688C4.50703 15.4688 5.79023 14.6531 6.42773 14.1492C7.21992 14.4445 8.085 14.6133 9 14.6133C13.0301 14.6133 16.3125 11.6391 16.3125 7.98047C16.3125 4.32187 13.0301 1.6875 9 1.6875ZM5.20312 9.28125C4.68164 9.28125 4.25391 8.85352 4.25391 8.33203C4.25391 7.81055 4.68164 7.38281 5.20312 7.38281C5.72461 7.38281 6.15234 7.81055 6.15234 8.33203C6.15234 8.85352 5.72461 9.28125 5.20312 9.28125ZM9 9.28125C8.47852 9.28125 8.05078 8.85352 8.05078 8.33203C8.05078 7.81055 8.47852 7.38281 9 7.38281C9.52148 7.38281 9.94922 7.81055 9.94922 8.33203C9.94922 8.85352 9.52148 9.28125 9 9.28125ZM12.7969 9.28125C12.2754 9.28125 11.8477 8.85352 11.8477 8.33203C11.8477 7.81055 12.2754 7.38281 12.7969 7.38281C13.3184 7.38281 13.7461 7.81055 13.7461 8.33203C13.7461 8.85352 13.3184 9.28125 12.7969 9.28125Z" fill="url(#sm_icon_0)"/><defs><radialGradient id="sm_icon_0" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(1.6875 8.578) scale(14.625 720)"><stop stopColor="#202020"/><stop offset="1" stopColor="#0080FF"/></radialGradient></defs></svg>,
  /* 循环 - 平均轮次 */
  <svg key="s1" width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M14.7656 4.40039L13.0078 6.15234C12.8438 6.31641 12.6328 6.39844 12.4219 6.39844C12.2109 6.39844 12 6.31641 11.8359 6.15234C11.5078 5.82422 11.5078 5.30859 11.8359 4.98047L12.1992 4.61719C11.2734 4.13672 10.207 3.86719 9 3.86719C6.16406 3.86719 3.86719 6.16406 3.86719 9C3.86719 9.46172 3.49219 9.83672 3.03047 9.83672C2.56875 9.83672 2.19375 9.46172 2.19375 9C2.19375 5.25 5.25 2.19375 9 2.19375C10.7344 2.19375 12.2812 2.85234 13.4297 3.92812L13.5703 3.78867C13.8984 3.46055 14.4141 3.46055 14.7422 3.78867C15.0703 4.12148 15.0938 4.07344 14.7656 4.40039ZM14.9695 8.16328C14.5078 8.16328 14.1328 8.53828 14.1328 9C14.1328 11.8359 11.8359 14.1328 9 14.1328C7.79297 14.1328 6.72656 13.8633 5.80078 13.3828L6.16406 13.0195C6.49219 12.6914 6.49219 12.1758 6.16406 11.8477C5.83594 11.5195 5.32031 11.5195 4.99219 11.8477L3.23438 13.5996C2.90625 13.9277 2.90625 14.4434 3.23438 14.7715C3.5625 15.0996 4.07813 15.0996 4.40625 14.7715L4.57031 14.6074C5.71875 15.6797 7.26562 16.3383 9 16.3383C12.75 16.3383 15.8062 13.2727 15.8062 9.52266C15.8062 9.06094 15.4312 8.16328 14.9695 8.16328Z" fill="url(#sm_icon_1)"/><defs><radialGradient id="sm_icon_1" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(2.19375 9) scale(13.6125 600)"><stop stopColor="#202020"/><stop offset="1" stopColor="#0080FF"/></radialGradient></defs></svg>,
  /* 闪电 - 工具调用 */
  <svg key="s2" width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M11.1557 0.568474C11.2759 0.547602 11.3997 0.565694 11.5083 0.621208C11.6168 0.676751 11.7039 0.766463 11.7573 0.876091C11.8107 0.985788 11.8275 1.10986 11.8042 1.22961L10.77 6.39172L14.8227 7.91125C14.9089 7.94398 14.9857 7.99716 15.0464 8.06652C15.1071 8.13609 15.1505 8.2197 15.1714 8.30968C15.1922 8.39969 15.1905 8.4939 15.1665 8.58312C15.1425 8.67222 15.0968 8.75406 15.0337 8.8214H15.0366L7.1616 17.2589L7.09421 17.3204C7.0224 17.3757 6.9373 17.4131 6.84714 17.4288L6.7573 17.4366C6.69672 17.4373 6.63627 17.4288 6.57859 17.4103L6.49461 17.3751C6.386 17.3195 6.29798 17.2299 6.24461 17.1202C6.20472 17.0381 6.18625 16.9479 6.18894 16.8575L6.19871 16.7667L7.22996 11.6105L3.17722 10.089C3.11208 10.0646 3.05213 10.0285 3.00046 9.98254L2.95164 9.93273C2.9057 9.8803 2.86992 9.82011 2.84617 9.755L2.82664 9.68859C2.80577 9.59809 2.80709 9.50378 2.83152 9.41418C2.85597 9.32456 2.90234 9.2423 2.96629 9.17492L10.8413 0.737419C10.9247 0.648358 11.0355 0.589437 11.1557 0.568474ZM5.34324 9.09972L9.1655 10.5353L8.63035 13.2111L11.1528 10.5089H11.1401L12.6479 8.89758L8.83445 7.46789L9.37058 4.78527L5.34324 9.09972Z" fill="url(#sm_icon_2)"/><defs><radialGradient id="sm_icon_2" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(2.81201 8.99836) scale(12.3738 747.725)"><stop stopColor="#202020"/><stop offset="1" stopColor="#0080FF"/></radialGradient></defs></svg>,
  /* 地球 - 活跃渠道 */
  <svg key="s3" width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M9 0.84375C4.49648 0.84375 0.84375 4.49648 0.84375 9C0.84375 13.5035 4.49648 17.1562 9 17.1562C13.5035 17.1562 17.1562 13.5035 17.1562 9C17.1562 4.49648 13.5035 0.84375 9 0.84375ZM2.53125 9C2.53125 8.4082 2.60859 7.83633 2.75273 7.29023L4.92188 9.45937V10.4062C4.92188 10.9301 5.34492 11.3531 5.86875 11.3531H7.78125V14.1563H6.83438V12.2438C6.83438 11.7199 6.41133 11.2969 5.8875 11.2969C5.36367 11.2969 4.94063 10.8738 4.94063 10.35V9.6L2.5793 7.23867C3.05156 5.95195 3.94336 4.86562 5.10117 4.15523L5.86875 5.04141V5.625C5.86875 6.14883 6.29179 6.57187 6.81562 6.57187H10.5938C11.1176 6.57187 11.5406 6.99492 11.5406 7.51875V8.46562C11.5406 8.98945 11.9637 9.4125 12.4875 9.4125H13.4344V11.3531H14.1187V14.0414C12.973 15.0676 11.4609 15.4688 9.94688 15.4688H9V12.3094C9 11.7855 8.57695 11.3625 8.05312 11.3625L7.875 11.3531V9.45937L9.94688 9.45937L9.94688 8.51016L7.875 8.51016V6.61875C8.39414 6.10547 9.21758 5.65195 10.0688 5.4082L9 4.05L9.62578 3.42422L11.4844 5.28281V5.625C12.0996 5.625 13.0078 5.625 13.6113 6.1875L13.7836 6.34922L14.85 5.28281C15.5613 6.4125 15.4688 7.76953 15.4688 9C15.4688 12.5719 12.5719 15.4688 9 15.4688C5.42812 15.4688 2.53125 12.5719 2.53125 9Z" fill="url(#sm_icon_3)"/><defs><radialGradient id="sm_icon_3" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(0.84375 9) scale(16.3125 720)"><stop stopColor="#202020"/><stop offset="1" stopColor="#0080FF"/></radialGradient></defs></svg>,
];

const LEGACY_CHANNEL_DIST = [
  { name: "Feishu Dm", count: 4 },
  { name: "QQ Dm", count: 3 },
  { name: "Feishu Group", count: 2 },
  { name: "CLI", count: 1 },
  { name: "Webchat", count: 1 },
];

const LEGACY_MODEL_DIST = [
  { name: "hunyuan-turbos-latest", value: 6, color: "#1447E6" },
  { name: "deepseek-v3.2", value: 5, color: "#020617" },
];

const LEGACY_AGENT_NAMES = [
  "Agent-A", "Agent-B",
  "运行中-长名字示例-Agent-C-用于演示截断",
  "Agent-D", "Agent-E", "Agent-F",
  "运维巡检-Agent-G-Special-Long-Name",
  "Agent-H",
];

const LEGACY_SESSIONS_RAW = [
  { id: "c3b2ac3c", name: "System: [2026-03-09 16:10]", model: "hunyuan-turbos-latest", tokens: "521K", cost: "$0.0742", updatedAt: "2026-03-09 17:49" },
  { id: "81c87c7b", name: "Conversation info (untrus", model: "hunyuan-turbos-latest", tokens: "188K", cost: "$0.0155", updatedAt: "2026-03-09 10:07" },
  { id: "267e462d", name: "Conversation info (untrus", model: "hunyuan-turbos-latest", tokens: "476K", cost: "$0.0691", updatedAt: "2026-03-08 14:17" },
  { id: "7be362c", name: "System: [2026-03-08 12:49]", model: "hunyuan-turbos-latest", tokens: "755K", cost: "$0.1076", updatedAt: "2026-03-08 13:58" },
  { id: "c51c62c7", name: "你是什么模型", model: "hunyuan-turbos-latest", tokens: "29K", cost: "$0.0041", updatedAt: "2026-03-08 12:54" },
  { id: "96c0b225", name: "System: [2026-03-08 12:45]", model: "deepseek-v3.2", tokens: "1.88M", cost: "$0.2700", updatedAt: "2026-03-08 05:14" },
  { id: "a46be688", name: "Conversation info (untrus", model: "deepseek-v3.2", tokens: "965K", cost: "$0.1359", updatedAt: "2026-03-07 15:29" },
  { id: "e4861318", name: "System: [2026-03-06 10:49]", model: "deepseek-v3.2", tokens: "415K", cost: "$0.0685", updatedAt: "2026-03-05 07:21" },
  { id: "6a9b9765", name: "System: [2026-03-04 17:59]", model: "deepseek-v3.2", tokens: "585K", cost: "$0.0829", updatedAt: "2026-03-04 13:08" },
  { id: "7878d832", name: "System: [2026-03-04 13:32]", model: "deepseek-v3.2", tokens: "1.95M", cost: "$0.2743", updatedAt: "2026-03-04 13:06" },
  { id: "a9c7eb8b", name: "[Wed 2026-03-04 12:11 UTC", model: "deepseek-v3.2", tokens: "1.59M", cost: "$0.2242", updatedAt: "2026-03-04 12:23" },
];
const LEGACY_SESSIONS = LEGACY_SESSIONS_RAW.map((s, i) => ({
  ...s,
  agentName: LEGACY_AGENT_NAMES[i % LEGACY_AGENT_NAMES.length],
}));

// 根据"当前安装的版本"判断是否需要红点提示（已迁移到 lib/pluginUpgrade）

// 未开启 CLS 时的引导卡（4 张统一价值主题卡，跨页面共用，样式与"运维观测"一致）
const SESSION_MGMT_ICON_BASE = "/assets/admin-session-management";
const GUIDE_CARDS = [
  {
    id: "global-health",
    title: "全局态势与健康度监控",
    description: "聚合实例规模、会话量、消息吞吐与响应耗时等核心指标，一屏看清系统是否运行正常，及时发现性能下滑与异常风险",
    iconSrc: `${SESSION_MGMT_ICON_BASE}/business-health-monitoring.svg`,
  },
  {
    id: "session-trace-replay",
    title: "会话全链路追溯与根因下钻",
    description: "完整还原每一轮对话的执行过程，遇到异常会话支持逐层下钻分析，快速定位问题根因，缩短排障时间",
    iconSrc: `${SESSION_MGMT_ICON_BASE}/session-detail-analysis.svg`,
  },
  {
    id: "perf-bottleneck",
    title: "响应性能与瓶颈分析",
    description: "拆解大模型推理、工具调用、流程编排各环节耗时与延迟趋势，精准定位卡在哪一步、哪些请求响应过慢",
    iconSrc: `${SESSION_MGMT_ICON_BASE}/app-log-otel-insight.svg`,
  },
  {
    id: "token-tool-insight",
    title: "Token 成本与工具效能洞察",
    description: "看清 Token 花在哪些会话与工具上，识别异常调用与高成本来源，帮助优化 Token 成本与工具链运行质量",
    iconSrc: `${SESSION_MGMT_ICON_BASE}/high-token-session-control.svg`,
  },
];

// 工具函数
function todayStr() {
  return new Date().toISOString().slice(0, 10);
}

// ─── 辅助 ────────────────────────────────────────────────────────────

function getLatestTrace(sessionId: string) {
  const traces = getSessionTraces(sessionId);
  return traces.length > 0 ? traces[traces.length - 1] : null;
}

type SortCol = "createdAt" | "durationSec" | "traces" | "totalTokens";

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

/** 将 FilterSection[] 转为 TreeSelectNode[]（供 TreeSelect 组件使用） */
function sectionsToTreeNodes(sections: FilterSection[]): TreeSelectNode[] {
  const convert = (node: TreeNodeData): TreeSelectNode => ({
    id: node.id,
    name: node.name,
    children: node.children?.map(convert),
  });
  return sections.flatMap((section) => section.roots.map(convert));
}

// 列定义
const ALL_COLUMNS = [
  { key: "openClawName", label: "OpenClaw 名称", defaultVisible: true },
  { key: "id", label: "Session ID", defaultVisible: true },
  { key: "createdAt", label: "创建时间", defaultVisible: true },
  { key: "input", label: "Input（最近一轮）", defaultVisible: true },
  { key: "output", label: "Output（最近一轮）", defaultVisible: true },
  { key: "duration", label: "耗时", defaultVisible: true },
  { key: "traces", label: "Trace 数", defaultVisible: true },
  { key: "totalTokens", label: "Total Token", defaultVisible: true },
] as const;

// ─── 主组件 ──────────────────────────────────────────────────────────

export default function SessionManagement() {
  // 停服态下，本页三处"选择日期" DatePicker 弹出的日历面板需保持 100% 可用。
  // 触发器已通过外层 <div data-billing-exempt> 打标；但面板经 Radix Portal 挂到 <body>，
  // 不在触发器的祖先链上，因此需要独立的页面级 hook 打标 + 注入 CSS 补充规则；
  // 详见 ./SessionManagement/useSessionManagementCalendarBillingExempt.ts 头部注释。
  useSessionManagementCalendarBillingExempt();
  const [, navigate] = useLocation();
  const searchString = useSearch();
  const urlParams = useMemo(() => new URLSearchParams(searchString), [searchString]);

  // 筛选状态
  const [showFilterPanel, setShowFilterPanel] = useState(true);
  const [idSearch, setIdSearch] = useState("");
  const [selectedOpenClaw, setSelectedOpenClaw] = useState("");
  const [minDur, setMinDur] = useState("");
  const [maxDur, setMaxDur] = useState("");
  const [minTraces, setMinTraces] = useState("");
  const [maxTraces, setMaxTraces] = useState("");
  const [minInTok, setMinInTok] = useState("");
  const [maxInTok, setMaxInTok] = useState("");
  const [minOutTok, setMinOutTok] = useState("");
  const [maxOutTok, setMaxOutTok] = useState("");
  const [minTotTok, setMinTotTok] = useState("");
  const [maxTotTok, setMaxTotTok] = useState("");

  // 标签
  const allTags = useMemo(() => Array.from(new Set(sessionsData.flatMap(s => s.tags))), []);
  const [selectedTags, setSelectedTags] = useState<Set<string>>(new Set());

  // OpenClaw 状态筛选
  const [openClawStatus, setOpenClawStatus] = useState<"all" | "normal" | "error">("all");

  // ── 顶栏状态（继承运维观测） ──
  const today = todayStr();
  const [dateFrom, setDateFrom] = useState(today);
  const [dateTo, setDateTo] = useState(today);
  const [refreshing, setRefreshing] = useState(false);
  const [groupFilter, setGroupFilter] = useState<string>(""); // "" = 全部分组
  const [agentFilter, setAgentFilter] = useState<string>(""); // "" = 全部 Agent
  const [scopeFilter, setScopeFilter] = useState<string[]>([]); // [] = 全部用户
  const filterSections = useFilterSections();
  const scopeGroupSelectGroups = useMemo(
    () => toScopeGroupSelectGroups(filterSections),
    [filterSections],
  );
  const groupTreeNodes = useMemo(
    () => sectionsToTreeNodes(filterSections),
    [filterSections],
  );

  // 升级 / 关闭 弹框
  const [showCloseClsConfirm, setShowCloseClsConfirm] = useState(false);
  const [deleteLogTopic, setDeleteLogTopic] = useState(false);

  // 会话管理页面版本：v1 = 旧版（默认），v2 = 新版
  // 默认 v1：用户首次开 CLS 进入旧版；只有点击"升级到新版"且 CLS 采集插件升级到 v2 后才能进新版
  const [sessionPageVersion, setSessionPageVersion] = useState<'v1' | 'v2'>(() => {
    const stored = localStorage.getItem('sessionPageVersion');
    return stored === 'v2' ? 'v2' : 'v1';
  });

  // 当前 CLS 采集插件版本（用 state 触发重新渲染）
  const [currentPluginVersion, setCurrentPluginVersion] = useState<string>(() => localStorage.getItem('clsPluginVersion') || 'v1');

  // 全局升级状态（来自 Context）
  const { status: upgradeStatus, start: startUpgrade } = usePluginUpgrade();

  // 升级成功后同步本页 state（升级完即享受新版）
  useEffect(() => {
    if (upgradeStatus === 'succeeded') {
      setCurrentPluginVersion(LATEST_PLUGIN_VERSION);
      setSessionPageVersion('v2');
    }
  }, [upgradeStatus]);

  // 点击"升级 CLS 采集插件"按钮：直接走升级流程（无确认 / 无版本选择）
  const handleUpgradePlugin = () => {
    if (upgradeStatus === 'running') return;
    if (currentPluginVersion === LATEST_PLUGIN_VERSION) {
      toast.info('CLS 采集插件已是最新版本');
      return;
    }
    startUpgrade();
  };

  // 点击"升级到新版页面"按钮：当前插件不是最新版时直接触发升级；已是最新则直接切版面
  const handleUpgradeToNewPage = () => {
    if (currentPluginVersion !== LATEST_PLUGIN_VERSION) {
      handleUpgradePlugin();
      return;
    }
    localStorage.setItem('sessionPageVersion', 'v2');
    setSessionPageVersion('v2');
    toast.success('已切换到新版会话管理');
  };

  // 从新版"返回旧版"
  const handleBackToOldPage = () => {
    localStorage.setItem('sessionPageVersion', 'v1');
    setSessionPageVersion('v1');
  };

  // CLS 开启状态（与运维观测共用 globalClsEnabled）
  const [clsEnabled, setClsEnabled] = useState(() => {
    const stored = localStorage.getItem("globalClsEnabled");
    return stored === "true";
  });
  const [isEnablingCls, setIsEnablingCls] = useState(false);
  const [isClosingCls, setIsClosingCls] = useState(false);

  // 跨页面同步 globalClsEnabled
  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === "globalClsEnabled") setClsEnabled(e.newValue === "true");
    };
    window.addEventListener("storage", handleStorageChange);
    return () => window.removeEventListener("storage", handleStorageChange);
  }, []);

  const handleOpenCls = () => {
    setIsEnablingCls(true);
    setTimeout(() => {
      setClsEnabled(true);
      localStorage.setItem("globalClsEnabled", "true");
      // 开启 CLS：强制重置为旧版会话 + 旧版插件，让用户走"先升级插件再切新版"的完整流程
      localStorage.setItem('sessionPageVersion', 'v1');
      setSessionPageVersion('v1');
      localStorage.setItem('clsPluginVersion', 'v1');
      setCurrentPluginVersion('v1');
      setIsEnablingCls(false);
      toast.success("CLS 日志服务开启成功");
    }, 1000);
  };

  const handleCloseCls = () => {
    setIsClosingCls(true);
    setTimeout(() => {
      setClsEnabled(false);
      localStorage.setItem("globalClsEnabled", "false");
      // 关闭 CLS 时清理版本状态：下次重新开启回到默认 v1
      localStorage.removeItem('sessionPageVersion');
      localStorage.removeItem('clsPluginVersion');
      setSessionPageVersion('v1');
      setCurrentPluginVersion('v1');
      setIsClosingCls(false);
      setShowCloseClsConfirm(false);
      setDeleteLogTopic(false);
      toast.success(deleteLogTopic ? "CLS 服务已关闭，日志主题已删除" : "CLS 服务已关闭");
    }, 800);
  };

  const handleCloseClsConfirmCancel = () => {
    setShowCloseClsConfirm(false);
    setDeleteLogTopic(false);
  };

  const handleRefresh = () => {
    setRefreshing(true);
    setTimeout(() => setRefreshing(false), 600);
  };

  // 从 URL 参数初始化
  useEffect(() => {
    const urlOpenClawStatus = urlParams.get("openClawStatus");
    if (urlOpenClawStatus === "error") setOpenClawStatus("error");
    else if (urlOpenClawStatus === "normal") setOpenClawStatus("normal");
  }, [urlParams]);

  // 排序
  const [sortCol, setSortCol] = useState<SortCol>("createdAt");
  const [sortAsc, setSortAsc] = useState(false);

  // 列可见性
  const [colVisible, setColVisible] = useState<Record<string, boolean>>(
    Object.fromEntries(ALL_COLUMNS.map(c => [c.key, c.defaultVisible]))
  );

  // 分页
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // 活跃筛选数量
  const activeFilterCount = useMemo(() => {
    let n = 0;
    if (idSearch) n++;
    if (selectedOpenClaw) n++;
    if (openClawStatus !== "all") n++;
    if (selectedTags.size > 0) n++;
    if (minDur || maxDur) n++;
    if (minTraces || maxTraces) n++;
    if (minInTok || maxInTok) n++;
    if (minOutTok || maxOutTok) n++;
    if (minTotTok || maxTotTok) n++;
    if (groupFilter) n++;
    if (agentFilter) n++;
    return n;
  }, [idSearch, selectedOpenClaw, openClawStatus, selectedTags, minDur, maxDur, minTraces, maxTraces, minInTok, maxInTok, minOutTok, maxOutTok, minTotTok, maxTotTok, groupFilter, agentFilter]);

  // 筛选 + 排序
  const filteredSessions = useMemo(() => {
    let result = sessionsData.filter(s => {
      if (idSearch && !s.id.toLowerCase().includes(idSearch.toLowerCase())) return false;
      if (selectedOpenClaw && s.openClawName !== selectedOpenClaw) return false;
      if (openClawStatus === "error" && s.status === "normal") return false;
      if (openClawStatus === "normal" && s.status !== "normal") return false;
      if (minDur && s.durationSec < Number(minDur)) return false;
      if (maxDur && s.durationSec > Number(maxDur)) return false;
      if (minTraces && s.traces < Number(minTraces)) return false;
      if (maxTraces && s.traces > Number(maxTraces)) return false;
      if (minTotTok && s.totalTokens < Number(minTotTok)) return false;
      if (maxTotTok && s.totalTokens > Number(maxTotTok)) return false;
      if (selectedTags.size > 0 && !s.tags.some(t => selectedTags.has(t))) return false;
      return true;
    });

    result.sort((a, b) => {
      let va: string | number, vb: string | number;
      if (sortCol === "createdAt") { va = a.createdAt; vb = b.createdAt; }
      else if (sortCol === "durationSec") { va = a.durationSec; vb = b.durationSec; }
      else if (sortCol === "traces") { va = a.traces; vb = b.traces; }
      else { va = a.totalTokens; vb = b.totalTokens; }
      if (typeof va === "string") return sortAsc ? va.localeCompare(vb as string) : (vb as string).localeCompare(va);
      return sortAsc ? (va as number) - (vb as number) : (vb as number) - (va as number);
    });

    return result;
  }, [idSearch, selectedOpenClaw, openClawStatus, minDur, maxDur, minTraces, maxTraces, minTotTok, maxTotTok, selectedTags, sortCol, sortAsc]);

  // 分页切片：当筛选/排序变化导致超出当前页时，自动回到第 1 页
  useEffect(() => {
    const totalPages = Math.max(1, Math.ceil(filteredSessions.length / pageSize));
    if (page > totalPages) setPage(1);
  }, [filteredSessions.length, pageSize, page]);

  const pagedSessions = useMemo(() => {
    const start = (page - 1) * pageSize;
    return filteredSessions.slice(start, start + pageSize);
  }, [filteredSessions, page, pageSize]);

  const handleSort = (col: SortCol) => {
    if (sortCol === col) setSortAsc(!sortAsc);
    else { setSortCol(col); setSortAsc(false); }
  };

  const toggleTag = (tag: string) => {
    setSelectedTags(prev => {
      const next = new Set(prev);
      if (next.has(tag)) next.delete(tag); else next.add(tag);
      return next;
    });
  };

  const toggleCol = (key: string) => {
    setColVisible(prev => ({ ...prev, [key]: !prev[key] }));
  };

  // 清空所有查询筛选（不包含「开启范围 / 日期 / 升级 / 关闭」等配置项）
  const handleClearAllFilters = () => {
    setIdSearch("");
    setSelectedOpenClaw("");
    setOpenClawStatus("all");
    setSelectedTags(new Set());
    setMinDur(""); setMaxDur("");
    setMinTraces(""); setMaxTraces("");
    setMinInTok(""); setMaxInTok("");
    setMinOutTok(""); setMaxOutTok("");
    setMinTotTok(""); setMaxTotTok("");
    setGroupFilter("");
    setAgentFilter("");
  };

  const SortIcon = ({ col }: { col: SortCol }) => {
    if (sortCol !== col) return <span className="text-gray-300 ml-0.5">↕</span>;
    return sortAsc ? <ChevronUp className="w-3 h-3 inline text-blue-500" /> : <ChevronDown className="w-3 h-3 inline text-blue-500" />;
  };

  const RangeFilter = ({ label, minVal, maxVal, onMinChange, onMaxChange }: {
    label: string; minVal: string; maxVal: string;
    onMinChange: (v: string) => void; onMaxChange: (v: string) => void;
  }) => (
    <div className="space-y-1.5">
      <Label className="text-[11px] font-medium text-[var(--text-secondary)]">{label}</Label>
      <div className="flex items-center gap-1.5">
        {/* 标准 Input：36px 高 / 4px 圆角 / --border / hover-focus #355EF1，不再二次编造样式 */}
        <Input
          type="number"
          placeholder="Min"
          value={minVal}
          onChange={e => onMinChange(e.target.value)}
        />
        <span className="text-[var(--text-weak)] text-xs flex-shrink-0">~</span>
        <Input
          type="number"
          placeholder="Max"
          value={maxVal}
          onChange={e => onMaxChange(e.target.value)}
        />
      </div>
    </div>
  );

  const visibleColCount = Object.values(colVisible).filter(Boolean).length;

  return (
    // data-billing-exempt: 停服态下会话管理保持正常可用（豁免全局停服禁用；元素自身 disabled 不受影响，延续禁用）
    <div className="page-enter" data-billing-exempt>
    {/* ═══════════ 顶栏（对齐运维观测） ═══════════ */}
      <AdminPageHeader
        title="会话管理"
        description="提供会话级检索、Token 下钻与链路溯源能力，为你的多轮对话系统构建可追溯、可分析、可优化的运维底座。"
        titleAccessory={
          <>
            {clsEnabled && sessionPageVersion === 'v2' && (
              <Button
                variant="claw-outline"
                size="claw-sm"
                onClick={handleBackToOldPage}
                title="返回旧版会话管理"
              >
                <ArrowLeftRight className="w-3.5 h-3.5" />
                返回旧版
              </Button>
            )}
            {clsEnabled && sessionPageVersion === 'v1' && (
              <Button
                variant="claw-outline"
                size="claw-sm"
                onClick={handleUpgradeToNewPage}
                title="前往新版会话管理"
              >
                <ArrowLeftRight className="w-3.5 h-3.5" />
                前往新版
              </Button>
            )}
          </>
        }
        actions={
          !clsEnabled ? (
            // 停服态豁免（页面级）：为DatePicker 触发器外层容器打 data-billing-exempt，
            // 命中 AdminDisabledOverlay 的视觉恢复 + 事件放行分支；配合本页
            // useSessionManagementCalendarBillingExempt hook 让面板同样可用。
            // 停服前如 DatePicker 自身传入了 disabled 则依旧生效（此处未传）。
            <div className="flex items-center gap-2" data-billing-exempt>
              <DatePicker value={dateFrom} onChange={setDateFrom} />
              <span className="text-[var(--text-weak)] text-sm">—</span>
              <DatePicker value={dateTo} onChange={setDateTo} />
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
          ) : null
        }
      />

      {/* 筛选行：分组 / Agent / 采集设置 / 升级 / 关闭（已开启 CLS 时显示）
          - v2 模式下"分组 / Agent"已并入左侧筛选面板，此处只保留右侧操作区
          - v1 模式下保留左侧"分组 / Agent"，让旧版用户继续使用 */}
      {clsEnabled && (
        <div className="flex items-end mb-6 gap-4 justify-between">
          {/* 左侧：v1 含分组/Agent + 日期；v2 仅日期 + 刷新（左对齐） */}
          {sessionPageVersion !== 'v2' ? (
            <div className="flex items-end gap-4">
              {/* 日期范围 + 刷新（最左侧） */}
              {/* 停服态豁免（页面级）：外层容器 data-billing-exempt，配合本页 hook。 */}
              <div className="flex items-center gap-2" data-billing-exempt>
                <DatePicker value={dateFrom} onChange={setDateFrom} />
                <span className="text-[var(--text-weak)] text-sm">—</span>
                <DatePicker value={dateTo} onChange={setDateTo} />
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
              <TreeSelect
                nodes={groupTreeNodes}
                value={groupFilter}
                onChange={setGroupFilter}
                allLabel="全部分组"
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
          ) : (
            // 停服态豁免（页面级）：v2 无左侧分组/Agent 时的日期范围容器。
            <div className="flex items-center gap-2" data-billing-exempt>
              <DatePicker value={dateFrom} onChange={setDateFrom} />
              <span className="text-[var(--text-weak)] text-sm">—</span>
              <DatePicker value={dateTo} onChange={setDateTo} />
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
          )}

          {/* 右侧：开启范围 + 升级插件 + 关闭服务 */}
          <div className="flex items-center gap-2">
            {/* 开启范围（CLS 开通后的全局采集范围，不是本页查询筛选） */}
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

      {/* ═══════════ 未开启 CLS 时的引导页（样式对齐运维观测） ═══════════ */}
      {!clsEnabled && (
        <>
          {/* 开通提示条 + 开启范围（Alert info 风格） */}
          <Alert variant="info" className="mb-6">
            <Info className="h-4 w-4" />
            <AlertDescription>
              <div className="flex items-start justify-between gap-6">
                <div className="flex-1">
                  <h3 className="text-sm font-semibold text-[var(--text-title)] mb-1">会话管理需要开启 CLS 日志服务</h3>
                  <p className="text-xs text-[var(--text-muted)]">
                    开启后，为您赠送3个月ClawPro 专属 CLS 日志服务免费额度，预估可覆盖 500台 OpenClaw 机器3个月的日志用量；服务到期后，CLS 将按量计费。
                    <a href="https://cloud.tencent.com/document/product/614/45802" target="_blank" rel="noreferrer" className="text-[var(--text-brand)] hover:underline inline-flex items-center gap-1 ml-1">
                      计费详情 <ArrowUpRight className="w-3 h-3" />
                    </a>
                  </p>
                </div>
                <Button
                  variant="claw-primary"
                  onClick={handleOpenCls}
                  disabled={isEnablingCls}
                  className="ml-4 text-xs h-8 px-4 whitespace-nowrap flex-shrink-0"
                >
                  {isEnablingCls ? '开启中...' : '开启 CLS 日志服务'}
                </Button>
              </div>

              {/* 开启范围（开通前的配置项，不是查询筛选）—— 复用 CLS 开通后顶部工具栏的交互形态 */}
              <div className="mt-4 pt-4 border-t border-[var(--alert-info-border)] flex items-center gap-3 flex-wrap">
                <span className="inline-flex items-center gap-1.5 text-xs font-medium text-[var(--text-title)] flex-shrink-0">
                  <SlidersHorizontal className="w-3.5 h-3.5" />
                  开启范围
                </span>
                <div className="w-60">
                  <GroupSelect
                    groups={scopeGroupSelectGroups}
                    selectedIds={scopeFilter}
                    onChange={setScopeFilter}
                    placeholder="全部用户（默认）"
                    sourceFilter={SCOPE_GROUP_SOURCE_FILTER}
                    sourceLabels={SCOPE_GROUP_SOURCE_LABELS}
                    compactTrigger
                  />
                </div>
                <span className="text-xs text-[var(--text-muted)]">
                  {scopeFilter.length === 0
                    ? "默认为全部用户开启日志服务，可在此限定分组以节省 CLS 配额。"
                    : `当前：将为 ${scopeFilter.length} 个分组开启 CLS 日志服务，其余用户不再上报日志。`}
                </span>
              </div>
            </AlertDescription>
          </Alert>

          {/* 引导卡：4 张统一价值主题卡（SurfaceCard 风格） */}
          <div className="space-y-4 mb-8">
            <SurfaceCard className="px-6 py-5">
              <h4 className="text-[14px] font-medium text-[var(--text-muted)] mb-4">开启 CLS 日志服务后您可以获得以下数据：</h4>
              <div className="grid grid-cols-2 gap-x-6 gap-y-5">
                {GUIDE_CARDS.map((card) => (
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

      {/* ═══════════ 旧版会话管理（v1）—— token 化重构 ═══════════ */}
      {clsEnabled && sessionPageVersion === 'v1' && (
        <div className="space-y-8">
          {/* 顶部 4 张统计卡（复用 design-refresh-2026 风格：SurfaceCard + 渐变 SVG + StatNumber） */}
          <div className="grid grid-cols-4 gap-5">
            {LEGACY_STAT_CARDS.map((card, idx) => (
              <SurfaceCard key={card.metric} className="p-5">
                <div className="flex items-center gap-2 mb-3">
                  {LEGACY_STAT_ICONS[idx]}
                  <span className="text-sm text-[var(--text-muted)]">{card.label}</span>
                </div>
                <StatNumber>{card.value}</StatNumber>
              </SurfaceCard>
            ))}
          </div>

          {/* 会话摘要表格 */}
          <div>
            <div className="mb-4 flex items-end justify-between gap-2">
              <div>
                <h2 className="text-base font-semibold text-[#0A0A0A]">会话摘要一览</h2>
                <p className="text-xs text-[#737373] mt-1">按时间倒序 · 点击查看会话详情</p>
              </div>
              <UITooltip>
                <UITooltipTrigger asChild>
                  <Button
                    variant="claw-outline"
                    size="claw-square"
                    onClick={() => toast.success('正在导出会话列表（mock）')}
                    aria-label="导出列表"
                  >
                    <Download className="w-3.5 h-3.5" />
                  </Button>
                </UITooltipTrigger>
                <UITooltipContent side="top" className="text-xs">导出列表</UITooltipContent>
              </UITooltip>
            </div>
            <SurfaceCard className="overflow-hidden p-0">
              <Table variant="white">
                <TableHeader>
                  <TableRow>
                    <TableHead>Agent 名称</TableHead>
                    <TableHead>会话</TableHead>
                    <TableHead>会话 ID</TableHead>
                    <TableHead>模型</TableHead>
                    <TableHead className="text-right">轮次</TableHead>
                    <TableHead className="text-right">Tokens</TableHead>
                    <TableHead className="text-right">成本</TableHead>
                    <TableHead className="text-right">更新时间 ↓</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {LEGACY_SESSIONS.map((s) => (
                    <TableRow key={s.id}>
                      <TableCell style={{ maxWidth: 180 }}>
                        <UITooltip>
                          <UITooltipTrigger asChild>
                            <div className="truncate">{s.agentName}</div>
                          </UITooltipTrigger>
                          <UITooltipContent side="top" className="max-w-xs break-all">{s.agentName}</UITooltipContent>
                        </UITooltip>
                      </TableCell>
                      <TableCell
                        className="cursor-pointer"
                        style={{ maxWidth: 280 }}
                        onClick={() => navigate(`/admin/session/${s.id}`)}
                      >
                        <UITooltip>
                          <UITooltipTrigger asChild>
                            <div className="font-medium hover:text-[#1447E6] transition-colors truncate">{s.name}</div>
                          </UITooltipTrigger>
                          <UITooltipContent side="top" className="max-w-xs break-all">{s.name}</UITooltipContent>
                        </UITooltip>
                      </TableCell>
                      <TableCell>{s.id}</TableCell>
                      <TableCell>{s.model}</TableCell>
                      <TableCell className="text-right tabular-nums">28</TableCell>
                      <TableCell className="text-right tabular-nums">{s.tokens}</TableCell>
                      <TableCell className="text-right tabular-nums">{s.cost}</TableCell>
                      <TableCell className="text-right tabular-nums text-[#737373]">{s.updatedAt}</TableCell>
                      <TableActionCell fixed="right">
                        <Button
                          variant="link"
                          size="sm"
                          onClick={() => navigate(`/admin/session/${s.id}`)}
                        >
                          查看详情
                        </Button>
                      </TableActionCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              {/* 分页器（mock：单页展示全部） */}
              <div className="px-6 py-3 border-t border-[#E5E5E5]">
                <Pagination
                  total={LEGACY_SESSIONS.length}
                  current={1}
                  pageSize={20}
                  showTotal={(total) => `共 ${total} 条记录`}
                  onChange={() => { /* mock: 旧版 v1 不切换页 */ }}
                  className="w-full justify-between"
                />
              </div>
            </SurfaceCard>
          </div>

          {/* 渠道与模型分布 */}
          <div className="grid grid-cols-2 gap-6">
            <SurfaceCard className="p-6">
              <h3 className="text-sm font-semibold text-[#0A0A0A] mb-4">渠道分布</h3>
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={LEGACY_CHANNEL_DIST}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                  <XAxis dataKey="name" tick={{ fontSize: 11, fill: "#737373" }} />
                  <YAxis tick={{ fontSize: 11, fill: "#737373" }} />
                  <ReTooltip contentStyle={{ fontSize: 12 }} />
                  <Bar dataKey="count" fill="#1447E6" radius={[2, 2, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </SurfaceCard>
            <SurfaceCard className="p-6">
              <h3 className="text-sm font-semibold text-[#0A0A0A] mb-4">模型分布</h3>
              <ResponsiveContainer width="100%" height={250}>
                <PieChart>
                  <Pie
                    data={LEGACY_MODEL_DIST}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={({ name, value }) => `${name}: ${value}`}
                    outerRadius={80}
                    fill="#8884d8"
                    dataKey="value"
                  >
                    {LEGACY_MODEL_DIST.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <ReTooltip contentStyle={{ fontSize: 12 }} />
                </PieChart>
              </ResponsiveContainer>
            </SurfaceCard>
          </div>
        </div>
      )}

      {/* ═══════════ 新版会话管理（v2）—— token 化重构 ═══════════ */}
      {clsEnabled && sessionPageVersion === 'v2' && (
      <>
      <SurfaceCard className="flex p-0 overflow-hidden min-h-[calc(100vh-260px)]">
      {/* ═══ 左侧筛选面板（v2 唯一筛选入口） ═══ */}
      {showFilterPanel && (
        <div className="w-[260px] shrink-0 border-r border-[var(--border)] overflow-y-auto">
          {/* ── 顶部：筛选条件标题 + 已应用 + 清空（高度与右侧工具条一致） ── */}
          <div className="h-[52px] px-4 border-b border-[var(--border)] flex items-center justify-between flex-shrink-0">
            <div className="flex items-center gap-2">
              <PanelTitle as="h2" className="text-sm">筛选条件</PanelTitle>
              {activeFilterCount > 0 && (
                <Badge color="blue" className="h-5 px-2 text-[10px]">已应用 {activeFilterCount}</Badge>
              )}
            </div>
            {activeFilterCount > 0 && (
              <Button
                variant="link"
                size="sm"
                onClick={handleClearAllFilters}
                className="h-auto px-0 text-xs text-[var(--text-muted)] hover:text-[var(--text-danger)]"
              >
                全部清空
              </Button>
            )}
          </div>

          {/* ── 检索 ── */}
          <div className="px-4 py-3 border-b border-[var(--border)] space-y-2.5">
            <span className="text-xs font-semibold text-[var(--text-weak)] uppercase tracking-wider">检索</span>
            <div className="space-y-1.5">
              <Label className="text-[11px] font-medium text-[var(--text-secondary)]">Session ID</Label>
              <div className="relative">
                <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)] pointer-events-none" />
                <Input
                  type="text"
                  placeholder="模糊搜索..."
                  value={idSearch}
                  onChange={e => setIdSearch(e.target.value)}
                  className="h-8 pl-8 pr-3 text-xs"
                />
              </div>
            </div>
          </div>

          {/* ── 实例归属（分组 + Agent，从顶栏并入） ── */}
          <div className="px-4 py-3 border-b border-[var(--border)] space-y-2.5">
            <span className="text-xs font-semibold text-[var(--text-weak)] uppercase tracking-wider">实例归属</span>
            <div className="space-y-1.5">
              <Label className="text-[11px] font-medium text-[var(--text-secondary)]">分组</Label>
              <TreeSelect
                nodes={groupTreeNodes}
                value={groupFilter}
                onChange={setGroupFilter}
                allLabel="全部分组"
                triggerWidth={228}
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px] font-medium text-[var(--text-secondary)]">Agent</Label>
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
                triggerClassName="w-full bg-white"
              />
            </div>
          </div>

          {/* ── 范围筛选 ── */}
          <div className="px-4 py-3 space-y-3">
            <span className="text-xs font-semibold text-[var(--text-weak)] uppercase tracking-wider">范围筛选</span>
            <RangeFilter label="会话时长（秒）" minVal={minDur} maxVal={maxDur} onMinChange={setMinDur} onMaxChange={setMaxDur} />
            <RangeFilter label="Trace 数量" minVal={minTraces} maxVal={maxTraces} onMinChange={setMinTraces} onMaxChange={setMaxTraces} />
            <RangeFilter label="Input Token" minVal={minInTok} maxVal={maxInTok} onMinChange={setMinInTok} onMaxChange={setMaxInTok} />
            <RangeFilter label="Output Token" minVal={minOutTok} maxVal={maxOutTok} onMinChange={setMinOutTok} onMaxChange={setMaxOutTok} />
            <RangeFilter label="Total Token" minVal={minTotTok} maxVal={maxTotTok} onMinChange={setMinTotTok} onMaxChange={setMaxTotTok} />
          </div>
        </div>
      )}

      {/* ═══ 右侧：表格区域（工具条 + 表格 + 状态条；外层大 SurfaceCard 提供卡片边） ═══ */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* 工具条（高度与左侧筛选标题一致） */}
        <div className="h-[52px] border-b border-[var(--border)] px-4 flex items-center justify-between flex-shrink-0">
              <div className="flex items-center gap-2">
                <Button
                  variant="claw-outline"
                  size="sm"
                  onClick={() => setShowFilterPanel(!showFilterPanel)}
                  className="h-8 text-xs px-3 gap-1.5"
                >
                  {showFilterPanel ? <PanelLeftClose className="w-3.5 h-3.5" /> : <PanelLeft className="w-3.5 h-3.5" />}
                  {showFilterPanel ? "隐藏筛选" : "显示筛选"}
                  {activeFilterCount > 0 && (
                    <span className="ml-0.5 px-1.5 py-0.5 rounded-full bg-[var(--alert-info-bg)] text-[var(--text-brand)] text-[10px] font-medium">{activeFilterCount}</span>
                  )}
                </Button>
              </div>

              <div className="flex items-center gap-3">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="claw-outline"
                      size="sm"
                      className="h-8 text-xs px-3 gap-1.5"
                    >
                      <Settings2 className="w-3 h-3" />
                      Columns <span className="font-semibold text-[var(--text-body)]">{visibleColCount}/{ALL_COLUMNS.length}</span>
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="min-w-[180px]">
                    <DropdownMenuLabel className="text-xs text-[var(--text-muted)] font-normal">
                      选择展示列
                    </DropdownMenuLabel>
                    <DropdownMenuSeparator />
                    {ALL_COLUMNS.map(col => (
                      <DropdownMenuCheckboxItem
                        key={col.key}
                        checked={colVisible[col.key]}
                        onCheckedChange={() => toggleCol(col.key)}
                        onSelect={(e) => e.preventDefault() /* 勾选后不关闭菜单 */}
                        className="text-xs"
                      >
                        {col.label}
                      </DropdownMenuCheckboxItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>

            {/* 表格 —— 字号 12px / 行高 py-3 / 数字列统一右对齐 + tabular-nums / 长内容 truncate + Tooltip */}
            <div className="flex-1 overflow-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="bg-[var(--bg-grey-hover)]/50 border-b border-[var(--border)]">
                    {colVisible.openClawName && <th className="text-left py-3 px-4 font-semibold text-[var(--text-muted)] text-[11px] uppercase tracking-wide whitespace-nowrap">OpenClaw 名称</th>}
                    {colVisible.id && <th className="text-left py-3 px-4 font-semibold text-[var(--text-muted)] text-[11px] uppercase tracking-wide whitespace-nowrap">Session ID</th>}
                    {colVisible.createdAt && (
                      <th className="text-left py-3 px-4 font-semibold text-[var(--text-muted)] text-[11px] uppercase tracking-wide whitespace-nowrap cursor-pointer hover:text-[var(--text-brand)] transition-colors" onClick={() => handleSort("createdAt")}>
                        创建时间 <SortIcon col="createdAt" />
                      </th>
                    )}
                    {colVisible.input && <th className="text-left py-3 px-4 font-semibold text-[var(--text-muted)] text-[11px] uppercase tracking-wide whitespace-nowrap">Input（最近一轮）</th>}
                    {colVisible.output && <th className="text-left py-3 px-4 font-semibold text-[var(--text-muted)] text-[11px] uppercase tracking-wide whitespace-nowrap">Output（最近一轮）</th>}
                    {colVisible.duration && (
                      <th className="text-right py-3 px-4 font-semibold text-[var(--text-muted)] text-[11px] uppercase tracking-wide whitespace-nowrap cursor-pointer hover:text-[var(--text-brand)] transition-colors" onClick={() => handleSort("durationSec")}>
                        耗时 <SortIcon col="durationSec" />
                      </th>
                    )}
                    {colVisible.traces && (
                      <th className="text-right py-3 px-4 font-semibold text-[var(--text-muted)] text-[11px] uppercase tracking-wide whitespace-nowrap cursor-pointer hover:text-[var(--text-brand)] transition-colors" onClick={() => handleSort("traces")}>
                        Trace 数 <SortIcon col="traces" />
                      </th>
                    )}
                    {colVisible.totalTokens && (
                      <th className="text-right py-3 px-4 font-semibold text-[var(--text-muted)] text-[11px] uppercase tracking-wide whitespace-nowrap cursor-pointer hover:text-[var(--text-brand)] transition-colors" onClick={() => handleSort("totalTokens")}>
                        Total Token <SortIcon col="totalTokens" />
                      </th>
                    )}
                  </tr>
                </thead>
                <tbody className="divide-y divide-[var(--border)]">
                  {pagedSessions.map(session => {
                    const lt = getLatestTrace(session.id);
                    const ltIn = lt ? (lt.input || "").slice(0, 80) : "";
                    const ltOut = lt ? ((lt.output && lt.output !== "None") ? lt.output : "").slice(0, 80) : "";
                    const dash = <span className="text-[var(--text-weak)]">—</span>;
                    return (
                      <tr
                        key={session.id}
                        className="hover:bg-[var(--alert-info-bg)]/40 transition-colors cursor-pointer group"
                        onClick={() => navigate(`/admin/session/${session.id}`)}
                      >
                        {colVisible.openClawName && (
                          <td className="py-3 px-4 min-w-[140px] max-w-[180px]">
                            <UITooltip>
                              <UITooltipTrigger asChild>
                                <div className="text-[var(--text-emphasis)] font-medium truncate">{session.openClawName}</div>
                              </UITooltipTrigger>
                              <UITooltipContent side="top" className="text-xs max-w-xs break-all">{session.openClawName}</UITooltipContent>
                            </UITooltip>
                          </td>
                        )}
                        {colVisible.id && (
                          <td className="py-3 px-4 max-w-[160px]">
                            <UITooltip>
                              <UITooltipTrigger asChild>
                                <span className="text-[var(--text-brand)] font-mono truncate inline-block max-w-full align-middle group-hover:underline">{session.id.slice(0, 8)}</span>
                              </UITooltipTrigger>
                              <UITooltipContent side="top" className="text-xs font-mono break-all">{session.id}</UITooltipContent>
                            </UITooltip>
                          </td>
                        )}
                        {colVisible.createdAt && <td className="py-3 px-4 text-[var(--text-weak)] font-mono whitespace-nowrap">{session.createdAt}</td>}
                        {colVisible.input && (
                          <td className="py-3 px-4 max-w-[220px]">
                            <UITooltip>
                              <UITooltipTrigger asChild>
                                <span className="truncate block text-[var(--text-secondary)]">{ltIn || dash}</span>
                              </UITooltipTrigger>
                              {ltIn && <UITooltipContent side="top" className="text-xs max-w-md break-all">{ltIn}</UITooltipContent>}
                            </UITooltip>
                          </td>
                        )}
                        {colVisible.output && (
                          <td className="py-3 px-4 max-w-[220px]">
                            <UITooltip>
                              <UITooltipTrigger asChild>
                                <span className="truncate block text-[var(--text-secondary)]">{ltOut || dash}</span>
                              </UITooltipTrigger>
                              {ltOut && <UITooltipContent side="top" className="text-xs max-w-md break-all">{ltOut}</UITooltipContent>}
                            </UITooltip>
                          </td>
                        )}
                        {colVisible.duration && <td className="py-3 px-4 text-right text-[var(--text-secondary)] tabular-nums whitespace-nowrap">{session.durationStr}</td>}
                        {colVisible.traces && <td className="py-3 px-4 text-right font-semibold text-[var(--text-body)] tabular-nums">{session.traces}</td>}
                        {colVisible.totalTokens && <td className="py-3 px-4 text-right text-[var(--text-body)] font-mono tabular-nums">{fmtTokens(session.totalTokens)}</td>}
                      </tr>
                    );
                  })}
                  {filteredSessions.length === 0 && (
                    <tr>
                      <td colSpan={visibleColCount} className="p-0">
                        <Empty className="border-0 py-12">
                          <EmptyHeader>
                            <EmptyMedia />
                            <EmptyTitle>
                              {activeFilterCount > 0 ? "没有匹配的 Session" : "暂无 Session 数据"}
                            </EmptyTitle>
                            <EmptyDescription>
                              {activeFilterCount > 0
                                ? `当前共应用了 ${activeFilterCount} 项筛选条件，可调整或清空后再次查询。`
                                : "新的 Session 将在用户与 OpenClaw 产生对话后展示在此。"}
                            </EmptyDescription>
                          </EmptyHeader>
                          {activeFilterCount > 0 && (
                            <EmptyContent>
                              <Button
                                variant="claw-outline"
                                size="sm"
                                onClick={handleClearAllFilters}
                                className="text-xs h-8 px-3"
                              >
                                清空全部筛选条件
                              </Button>
                            </EmptyContent>
                          )}
                        </Empty>
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>

            {/* 底部分页器 */}
            <div className="px-4 py-3 border-t border-[var(--border)]">
              <Pagination
                total={filteredSessions.length}
                current={page}
                pageSize={pageSize}
                showSizeChanger
                pageSizeOptions={[10, 20, 50, 100]}
                showTotal={(total) => `共 ${total} 个 Sessions`}
                onChange={(p, ps) => { setPage(p); setPageSize(ps); }}
                className="w-full justify-between"
              />
            </div>
      </div>
      {/* end of 内容区（外层 SurfaceCard） */}
      </SurfaceCard>
      </>
      )}

      {/* ═══════════ 升级CLS采集插件：右上角进度 Toast，无 Dialog ═══════════ */}

      {/* ═══════════ 关闭 CLS 服务 AlertDialog（与运维观测保持一致） ═══════════ */}
      <AlertDialog open={showCloseClsConfirm} onOpenChange={setShowCloseClsConfirm}>
        <AlertDialogContent className="sm:max-w-[560px]">
          <AlertDialogHeader>
            <AlertDialogTitle>确定要关闭 CLS 日志服务吗？</AlertDialogTitle>
            <AlertDialogDescription>
              关闭后以下功能将无法使用，
              <span className="text-[var(--text-danger)]">此操作可能影响业务运行</span>。
            </AlertDialogDescription>
          </AlertDialogHeader>

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
