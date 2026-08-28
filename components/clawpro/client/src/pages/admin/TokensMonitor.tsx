/**
 * TokensMonitor - 管控端 Tokens 监控页
 * 设计风格：与整体管控台保持一致，浅色卡片 + 蓝紫渐变强调色
 */
import { useState, useMemo, useEffect } from "react";
import { useLocation } from "wouter";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { RefreshCw, ChevronRight, Info, AlertCircle, ArrowUpRight, BarChart3, Activity, CheckCircle2, ChevronDown, Check, Download, X } from "lucide-react";
import { Pagination } from "@/components/ui/pagination";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableActionCell } from "@/components/ui/table";
import { Dialog, DialogContent, DialogBody, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from "@/components/ui/dialog";
import { AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction } from "@/components/ui/alert-dialog";
import { Alert, AlertDescription, AlertTitle, AlertOperationInfoIcon } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { StatusTag } from "@/components/ui/status-tag";
import { SurfaceCard } from "@/components/ui/Surface";
import { StatNumber } from "@/components/ui/Typography";
import {
  NumberCard,
  RequestsIcon,
  InputTokensIcon,
  OutputTokensIcon,
  TotalTokensIcon,
} from "@/components/ui/number-card";
import {
  Tooltip as UITooltip,
  TooltipContent as UITooltipContent,
  TooltipTrigger as UITooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Popover, PopoverContent, PopoverTrigger,
} from "@/components/ui/popover";
import {
  HoverCard, HoverCardContent, HoverCardTrigger,
} from "@/components/ui/hover-card";
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
} from "recharts";
import { toast } from "sonner";
import { DatePicker } from "@/components/ui/date-picker";
import { MOCK_DEPARTMENTS, MOCK_TOKEN_BY_DEPARTMENT, MOCK_OPENCLAW_LIST, MOCK_CLAWS_WITH_DEPT, type DepartmentNode, type GroupNode, MOCK_GROUP_TREE_MANUAL, MOCK_GROUP_TREE_ONEID, MOCK_TOKEN_BY_GROUP_MANUAL, MOCK_TOKEN_BY_GROUP_ONEID } from "@/lib/mockData";
import { useAdminMode } from "@/contexts/AdminModeContext";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { TreeSelect } from "@/components/ui/tree-select";
import { useTokensMonitorCalendarBillingExempt } from "./TokensMonitor/useTokensMonitorCalendarBillingExempt";
import { SearchableSelect } from "@/components/ui/select";

// CLS 采集插件版本历史
interface CLSPluginVersion {
  version: string;
  releaseDate: string;
  changelog: string;
  status: 'current' | 'available' | 'deprecated';
}

const CLS_PLUGIN_VERSIONS: CLSPluginVersion[] = [
  { version: "v5", releaseDate: "2026-03-24", changelog: "修复会话追踪精度问题，优化 Token 计算算法", status: "available" },
  { version: "v4", releaseDate: "2026-03-17", changelog: "新增会话全局监控功能，支持多渠道分析", status: "available" },
  { version: "v3", releaseDate: "2026-03-10", changelog: "优化日志采集性能，降低 CPU 占用率", status: "current" },
  { version: "v2", releaseDate: "2026-03-03", changelog: "修复 CLS 连接超时问题", status: "deprecated" },
  { version: "v1", releaseDate: "2026-02-24", changelog: "首次发布 CLS 采集插件", status: "deprecated" },
];

// ─── 工具函数 ────────────────────────────────────────────────────────────────
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
function daysBetween(from: string, to: string) {
  return Math.round((new Date(to).getTime() - new Date(from).getTime()) / 86400000);
}
function fmt(n: number) { return n.toLocaleString(); }

// ─── Mock 数据生成 ────────────────────────────────────────────────────────────
// 生成最近 60 天每天的数据
const DAYS_HISTORY = 60;
const BASE_DATE = addDays(todayStr(), -(DAYS_HISTORY - 1));

const MEMBERS = [
  "alice@acompany.com",
  "bob@acompany.com",
  "carol@acompany.com",
  "dave@acompany.com",
  "eve@acompany.com",
  "frank@acompany.com",
  "grace@acompany.com",
  "henry@acompany.com",
  "ivy@acompany.com",
  "jack@acompany.com",
  "karen@acompany.com",
  "leo@acompany.com",
  "longname-user@very-long-domain-example.com",
  "product-ops-admin@enterprise-acompany.com",
];

const MODELS = [
  "腾讯云 DeepSeek (V3 0324)",
  "腾讯云混元 (Turbo)",
  "腾讯云 DeepSeek (R1)",
  "腾讯云混元 (Pro)",
];

// 每天每用户的 mock 数据
function seedRand(seed: number) {
  let s = seed;
  return () => { s = (s * 1664525 + 1013904223) & 0xffffffff; return (s >>> 0) / 0xffffffff; };
}

interface DayRecord {
  date: string;
  memberId: string;
  modelName: string;
  requests: number;
  inputTokens: number;
  outputTokens: number;
}

const ALL_RECORDS: DayRecord[] = [];
for (let i = 0; i < DAYS_HISTORY; i++) {
  const date = addDays(BASE_DATE, i);
  MEMBERS.forEach((memberId, mi) => {
    MODELS.forEach((modelName, moi) => {
      const rand = seedRand(i * 1000 + mi * 100 + moi);
      const requests = Math.floor(rand() * 80 + 5);
      const inputTokens = Math.floor(rand() * 15000 + 2000);
      const outputTokens = Math.floor(rand() * 12000 + 1500);
      ALL_RECORDS.push({ date, memberId, modelName, requests, inputTokens, outputTokens });
    });
  });
}

// 今日全局配额（固定）
// 注意：GLOBAL_LIMIT 为 null 表示无限制
const TODAY_RECORDS = ALL_RECORDS.filter((r) => r.date === todayStr());
const TODAY_TOTAL_TOKENS = TODAY_RECORDS.reduce((s, r) => s + r.inputTokens + r.outputTokens, 0);

// ─── 进度条 ───────────────────────────────────────────────────────────────────
function ProgressBar({ value, max, showTooltip, isUnlimited }: { value: number; max: number | null; showTooltip?: boolean; isUnlimited?: boolean }) {
  if (isUnlimited || max === null) {
    // 无限制时显示浅灰色进度条，不显示进度
    const bar = (
      <div className="w-full bg-[#f5f5f5] rounded-full h-1.5 cursor-default">
        <div className="h-1.5 rounded-full bg-gray-300 transition-all" style={{ width: "0%" }} />
      </div>
    );
    if (!showTooltip) return bar;
    return (
      <UITooltip>
        <UITooltipTrigger asChild>
          {bar}
        </UITooltipTrigger>
        <UITooltipContent side="bottom" className="text-xs font-medium">
          已消耗 {value.toLocaleString()} Tokens（无限制）
        </UITooltipContent>
      </UITooltip>
    );
  }
  const pct = Math.min((value / max) * 100, 100);
  const barColor = pct > 80 ? "bg-red-500" : pct > 60 ? "bg-yellow-500" : "bg-blue-500";
  const bar = (
    <div className="w-full bg-[#f5f5f5] rounded-full h-1.5 cursor-default">
      <div className={`h-1.5 rounded-full ${barColor} transition-all`} style={{ width: `${pct}%` }} />
    </div>
  );
  if (!showTooltip) return bar;
  return (
    <UITooltip>
      <UITooltipTrigger asChild>
        {bar}
      </UITooltipTrigger>
      <UITooltipContent side="bottom" className="text-xs font-medium">
        {value.toLocaleString()} / {max.toLocaleString()} Tokens
      </UITooltipContent>
    </UITooltip>
  );
}

// ─── CSV 导出工具 ────────────────────────────────────────────────────────────
function makeCsvBlob(header: string, rows: string[]): Blob {
  return new Blob(["\uFEFF" + header + "\n" + rows.join("\n")], { type: "text/csv;charset=utf-8" });
}
function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

// ─── 翻页组件 ─────────────────────────────────────────────────────────────────
const PAGE_SIZE = 10;

// 注：原 TokenDepartmentFilter / TokenGroupFilter 及其树节点为死代码（无 JSX 引用），
// 实际部门/分组筛选已统一改用 <TreeSelect />，旧实现已移除。

// ─── 主组件 ───────────────────────────────────────────────────────────────────
export default function TokensMonitor() {
  const [, navigate] = useLocation(); // 在组件顶级调用 useLocation
  const { hasOneid } = useAdminMode();

  // 停服态下，本页两个"选择日期" DatePicker 弹出的日历面板需保持 100% 可用。
  // 触发器已通过外层 <div data-billing-exempt> 打标；但面板经 Radix Portal 挂到 <body>，
  // 不在触发器的祖先链上，因此需要独立的页面级 hook 打标 + 注入 CSS 补充规则；
  // 详见./TokensMonitor/useTokensMonitorCalendarBillingExempt.ts 头部注释。
  useTokensMonitorCalendarBillingExempt();
  const today = todayStr();
  const [dateFrom, setDateFrom] = useState(today);
  const [dateTo, setDateTo] = useState(today);
  const [refreshing, setRefreshing] = useState(false);
  const [instancePage, setInstancePage] = useState(1);
  const [memberPage, setMemberPage] = useState(1);
  const [modelPage, setModelPage] = useState(1);
  const [sessionPage, setSessionPage] = useState(1);
  const [deptPage, setDeptPage] = useState(1);
  const [deptFilter, setDeptFilter] = useState("");
  const [groupPage, setGroupPage] = useState(1);
  const [groupFilter, setGroupFilter] = useState("");
  const [isEnablingCls, setIsEnablingCls] = useState(false);
  const [showSuccessMessage, setShowSuccessMessage] = useState(false);
  const [clsEnabled, setClsEnabled] = useState(() => {
    const stored = localStorage.getItem("globalClsEnabled");
    return stored === "true";
  });
  const [showCloseClsConfirm, setShowCloseClsConfirm] = useState(false);
  const [isClosingCls, setIsClosingCls] = useState(false);
  const [deleteLogTopic, setDeleteLogTopic] = useState(false);
  const [showPluginUpgradeDialog, setShowPluginUpgradeDialog] = useState(false);
  const [selectedPluginVersion, setSelectedPluginVersion] = useState<any>(null);
  const [isUpgradingPlugin, setIsUpgradingPlugin] = useState(false);

  // 当弹窗打开时，自动选中最新版本
  useEffect(() => {
    if (showPluginUpgradeDialog && !selectedPluginVersion) {
      setSelectedPluginVersion(CLS_PLUGIN_VERSIONS[0]); // v5 是最新版本
    }
  }, [showPluginUpgradeDialog]);
  const [showClsAgreementDialog, setShowClsAgreementDialog] = useState(false);
  const [clsAgreed, setClsAgreed] = useState(false);
  const [showAuthDialog, setShowAuthDialog] = useState(false);
  const [isCheckingAuth, setIsCheckingAuth] = useState(false);
  const [authCompleted, setAuthCompleted] = useState(false);
  const [authCheckInterval, setAuthCheckInterval] = useState<NodeJS.Timeout | null>(null);
  const [showFreeQuotaDialog, setShowFreeQuotaDialog] = useState(false);
  const [freeQuotaAgreed, setFreeQuotaAgreed] = useState(false);
  const [selectedAgent, setSelectedAgent] = useState(""); // Agent 名称筛选
  const [globalLimit, setGlobalLimit] = useState<number | null>(() => {
    const mode = localStorage.getItem("globalLimitMode");
    if (mode === "unlimited") return null;
    const value = localStorage.getItem("globalLimit");
    return value ? parseInt(value, 10) : 2000000;
  });
  // 全局 Tokens 时间维度（v2 完整版）—— 与平台策略页同步
  type NaturalPeriod = "daily" | "monthly" | "yearly";
  type CustomRefresh = "none" | NaturalPeriod;
  type TimeDimensionConfig = { type: "natural"; period: NaturalPeriod } | { type: "custom" };
  const parseGlobalTimeDim = (raw: string | null): TimeDimensionConfig => {
    try {
      if (raw) { const p = JSON.parse(raw); if (p && (p.type === "natural" || p.type === "custom")) return p; }
    } catch { /* ignore */ }
    return { type: "natural", period: "daily" };
  };
  const [globalTimeDimV2, setGlobalTimeDimV2] = useState<TimeDimensionConfig>(() => {
    const v2 = localStorage.getItem("admin_global_token_time_dim_v2");
    if (v2) return parseGlobalTimeDim(v2);
    // fallback to v1
    const v1 = localStorage.getItem("admin_global_token_time_dim");
    return { type: "natural", period: v1 === "monthly" ? "monthly" : "daily" };
  });
  // 兼容旧逻辑引用
  const globalTokenTimeDim = (globalTimeDimV2.type === "natural" && globalTimeDimV2.period === "monthly" ? "monthly" : "daily") as "daily" | "monthly";
  // 全局 Tokens 预设策略的 startAt/refresh（模拟开通服务时间）
  const GLOBAL_PRESET_START_AT = "2025-01-01T00:00";
  const GLOBAL_PRESET_REFRESH: CustomRefresh = globalTimeDimV2.type === "custom" ? "monthly" : "none";

  // 单用户 Tokens 时间维度
  const [userTimeDimV2, setUserTimeDimV2] = useState<TimeDimensionConfig>(() =>
    parseGlobalTimeDim(localStorage.getItem("admin_user_token_time_dim_v2"))
  );
  const USER_PRESET_TOKEN_LIMIT = 50000;
  // Mock：标记为「无限制」的用户（用于展示无限制态）
  const UNLIMITED_USERS = new Set<string>(["leo@acompany.com"]);
  // Mock：用户所在分组映射（模拟部分用户有分组，部分没有）
  const MOCK_USER_GROUPS: Record<string, Array<{ groupId: string; groupName: string; tokenLimit: number; startAt?: string; endAt?: string | null; refresh?: string }>> = {
    "alice@acompany.com": [
      { groupId: "dept-tech", groupName: "技术部", tokenLimit: 100000, startAt: "2026-05-01T09:00", endAt: "2026-12-31T23:59", refresh: "monthly" },
    ],
    "bob@acompany.com": [
      { groupId: "mgrp-rd", groupName: "研发组", tokenLimit: 100000, startAt: "2026-05-01T09:00", endAt: "2026-12-31T23:59", refresh: "daily" },
      { groupId: "mgrp-rd-fe", groupName: "研发-前端", tokenLimit: 100000, startAt: "2026-06-01T00:00", endAt: "2026-12-31T23:59", refresh: "daily" },
    ],
    "eve@acompany.com": [
      { groupId: "mgrp-design", groupName: "设计组", tokenLimit: 50000, startAt: "2026-05-15T10:00", endAt: "2026-11-15T10:00", refresh: "none" },
    ],
  };
  // 全局 Tokens 上限的"分组策略"列表（来自平台策略页）
  type GlobalTokenGroupRule = { id: string; groupIds: string[]; value: number | "unlimited"; startAt?: string; endAt?: string | null; refresh?: string };
  const [globalTokenGroupRules, setGlobalTokenGroupRules] = useState<GlobalTokenGroupRule[]>(() => {
    try {
      const raw = localStorage.getItem("admin_global_token_group_rules");
      if (!raw) return [];
      const parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  });
  // 是否启用了"按分组"模式（存在分组策略）
  const IS_GLOBAL_BY_GROUP = globalTokenGroupRules.length > 0;

  // 监听 localStorage 变化
  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === "globalClsEnabled") {
        setClsEnabled(e.newValue === "true");
      } else if (e.key === "globalLimitMode" || e.key === "globalLimit") {
        const mode = localStorage.getItem("globalLimitMode");
        if (mode === "unlimited") {
          setGlobalLimit(null);
        } else {
          const value = localStorage.getItem("globalLimit");
          setGlobalLimit(value ? parseInt(value, 10) : 2000000);
        }
      } else if (e.key === "admin_global_token_time_dim_v2" || e.key === "admin_global_token_time_dim") {
        const v2 = localStorage.getItem("admin_global_token_time_dim_v2");
        if (v2) { setGlobalTimeDimV2(parseGlobalTimeDim(v2)); }
        else { setGlobalTimeDimV2({ type: "natural", period: e.newValue === "monthly" ? "monthly" : "daily" }); }
      } else if (e.key === "admin_global_token_group_rules") {
        try {
          const parsed = e.newValue ? JSON.parse(e.newValue) : [];
          setGlobalTokenGroupRules(Array.isArray(parsed) ? parsed : []);
        } catch {
          setGlobalTokenGroupRules([]);
        }
      } else if (e.key === "admin_user_token_time_dim_v2") {
        setUserTimeDimV2(parseGlobalTimeDim(e.newValue));
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

  const handleRefresh = () => {
    setRefreshing(true);
    setTimeout(() => { setRefreshing(false); toast.success("数据已刷新"); }, 1000);
  };

  const handleOpenCLS = () => {
    // 检查授权状态（从后台缓存数据中获取）
    const isAuthorized = localStorage.getItem('clsAuthorized') === 'true';
    
    if (!isAuthorized) {
      // 未授权，显示授权 Dialog
      setShowAuthDialog(true);
      // 启动自动检测授权状态
      setIsCheckingAuth(true);
      const interval = setInterval(() => {
        const authorized = localStorage.getItem('clsAuthorized') === 'true';
        if (authorized) {
          // 已授权，关闭 Dialog 并继续
          setShowAuthDialog(false);
          setIsCheckingAuth(false);
          clearInterval(interval);
          // 继续开启 CLS 日志服务
          proceedWithClsSetup();
        }
      }, 2000);
      setAuthCheckInterval(interval);
    } else {
      // 已授权，直接继续
      proceedWithClsSetup();
    }
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

  const handleCancelAuth = () => {
    setShowAuthDialog(false);
    setIsCheckingAuth(false);
    setAuthCompleted(false);
    if (authCheckInterval) {
      clearInterval(authCheckInterval);
      setAuthCheckInterval(null);
    }
  };

  const handleConfirmFreeQuota = () => {
    if (!freeQuotaAgreed) return;
    setShowFreeQuotaDialog(false);
    setIsEnablingCls(true);
    setTimeout(() => {
      setClsEnabled(true);
      localStorage.setItem('globalClsEnabled', 'true');
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

  const handleCloseClsConfirmCancel = () => {
    setShowCloseClsConfirm(false);
    setDeleteLogTopic(false);
  };

  const handleConfirmClsAgreement = () => {
    if (!clsAgreed) return;
    setIsEnablingCls(true);
    // 模拟 loading 1.5 秒
    setTimeout(() => {
      setClsEnabled(true);
      localStorage.setItem('globalClsEnabled', 'true');
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

  const handleCloseCls = () => {
    setIsClosingCls(true);
    setTimeout(() => {
      setClsEnabled(false);
      localStorage.setItem("globalClsEnabled", "false");
      setIsClosingCls(false);
      setShowCloseClsConfirm(false);
      setDeleteLogTopic(false);
      const message = deleteLogTopic ? "CLS 日志服务已关闭，日志主题资源已删除" : "CLS 日志服务已关闭";
      toast.success(message);
    }, 1000);
  };

  // 计算全局配额百分比
  const TODAY_GLOBAL_PCT = globalLimit === null ? "0" : ((TODAY_TOTAL_TOKENS / globalLimit) * 100).toFixed(1);
  const IS_GLOBAL_UNLIMITED = globalLimit === null;

  // ─── 新卡片：全局 Tokens 上限展示 + 本周期消耗 ───
  const PERIOD_PREFIX_MAP: Record<NaturalPeriod, string> = { daily: "每日", monthly: "每月", yearly: "每年" };
  const PERIOD_LABEL_MAP: Record<NaturalPeriod, string> = { daily: "今日", monthly: "本月", yearly: "本年" };
  const REFRESH_LABEL_MAP: Record<CustomRefresh, string> = { none: "不刷新", daily: "按日刷新", monthly: "按月刷新", yearly: "按年刷新" };
  const fmtTimeShort = (iso: string | undefined | null): string => {
    if (!iso) return "";
    const d = new Date(iso);
    if (isNaN(d.getTime())) return iso;
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
  };

  // 全局 Tokens 上限：拆为两行展示（line1=起止时间，line2=刷新方式 + 上限值）
  // - 自然周期：line1 为空，line2 = "每月 1,000,000" / "每月 无限制"
  // - 自定义周期：line1 = "2025/01/01 00:00 - 无终止"，line2 = "按月刷新 1,000,000" / "按月刷新 无限制"
  // - 不刷新：line1 = "2025/01/01 00:00 - 无终止"，line2 = "不刷新 1,000,000" / "不刷新 无限制"
  // - 按分组：line1 为空，line2 = "按分组设置"
  const globalLimitInfo = (() => {
    if (IS_GLOBAL_BY_GROUP) return { line1: "", line2: "按分组设置" };
    const valStr = IS_GLOBAL_UNLIMITED ? "无限制" : (globalLimit ?? 0).toLocaleString();
    if (globalTimeDimV2.type === "natural") {
      return { line1: "", line2: `${PERIOD_PREFIX_MAP[globalTimeDimV2.period]} ${valStr}` };
    }
    return {
      line1: `${fmtTimeShort(GLOBAL_PRESET_START_AT)} - 无终止时间`,
      line2: `${REFRESH_LABEL_MAP[GLOBAL_PRESET_REFRESH]} ${valStr}`,
    };
  })();

  // 本周期窗口计算（按分组态返 null；无限制态仍返回窗口，仅消耗值显示「无限制」）
  const currentPeriodWindow = (() => {
    if (IS_GLOBAL_BY_GROUP) return null;
    const now = new Date();
    if (globalTimeDimV2.type === "natural") {
      return { label: PERIOD_LABEL_MAP[globalTimeDimV2.period], startStr: "", endStr: "" };
    }
    // 自定义周期：按 refresh 滚动窗口
    const start = new Date(GLOBAL_PRESET_START_AT);
    const refresh = GLOBAL_PRESET_REFRESH;
    if (refresh === "none") {
      // 不刷新 → 整个窗口
      return { label: "", startStr: fmtTimeShort(GLOBAL_PRESET_START_AT), endStr: "无终止时间" };
    }
    // 按 refresh 计算当前窗口起止
    let windowStart = new Date(start);
    const r = refresh as string;
    while (true) {
      const next = new Date(windowStart);
      if (r === "daily") next.setDate(next.getDate() + 1);
      else if (r === "monthly") next.setMonth(next.getMonth() + 1);
      else if (r === "yearly") next.setFullYear(next.getFullYear() + 1);
      if (next > now) return { label: "", startStr: fmtTimeShort(windowStart.toISOString()), endStr: fmtTimeShort(next.toISOString()) };
      windowStart = next;
    }
  })();

  const currentPeriodPct = IS_GLOBAL_BY_GROUP || IS_GLOBAL_UNLIMITED ? "0" : TODAY_GLOBAL_PCT;
  // 进度条阈值色（< 60 蓝 / 60-<80 橙 / >=80 红 / 按分组+无限制 灰）；百分比文字始终黑
  const periodPctNum = IS_GLOBAL_BY_GROUP || IS_GLOBAL_UNLIMITED ? 0 : Number(TODAY_GLOBAL_PCT);

  // ⓘ Info 图标 hover 文案
  const globalQuotaInfoTooltip = IS_GLOBAL_BY_GROUP
    ? "全局 Tokens 上限已按分组进行设置，请在下方“按分组”Tab 查看具体分组的消耗。"
    : IS_GLOBAL_UNLIMITED
      ? "全局配额已设置为无限制，无需关注消耗占比。"
      : "全局配额消耗 = 全平台所有用户调用公司配置模型产生的 Tokens 总和（不含用户自行配置的自定义模型） ÷ 全局 Tokens 上限。当前数字仅反映当前所属周期的累计占比，进入下一周期会按新周期重新统计。";

  const handleFromChange = (v: string) => {
    setDateFrom(v);
    setInstancePage(1);
    setMemberPage(1);
    setModelPage(1);
    setSessionPage(1);
    setDeptPage(1);
    setGroupPage(1);
  };
  const handleToChange = (v: string) => {
    setDateTo(v);
    setInstancePage(1);
    setMemberPage(1);
    setModelPage(1);
    setSessionPage(1);
    setDeptPage(1);
    setGroupPage(1);
  };

  // 有效时间范围
  const effectiveFrom = dateFrom || today;
  const effectiveTo = dateTo || today;
  const isSingleDay = effectiveFrom === effectiveTo;

  // 筛选范围内的记录
  const rangeRecords = useMemo(
    () => ALL_RECORDS.filter((r) => r.date >= effectiveFrom && r.date <= effectiveTo),
    [effectiveFrom, effectiveTo]
  );

  // 总览指标（随时间联动）
  const totalRequests = rangeRecords.reduce((s, r) => s + r.requests, 0);
  const totalInput = rangeRecords.reduce((s, r) => s + r.inputTokens, 0);
  const totalOutput = rangeRecords.reduce((s, r) => s + r.outputTokens, 0);
  const totalTokens = totalInput + totalOutput;

  // 折线图数据
  const chartData = useMemo(() => {
    if (isSingleDay) {
      // 单日：展示最近 7 天
      return Array.from({ length: 7 }, (_, i) => {
        const date = addDays(today, i - 6);
        const recs = ALL_RECORDS.filter((r) => r.date === date);
        return {
          date: date.slice(5), // MM-DD
          输入Tokens: recs.reduce((s, r) => s + r.inputTokens, 0),
          输出Tokens: recs.reduce((s, r) => s + r.outputTokens, 0),
        };
      });
    } else {
      // 时间段：展示每天
      const days = daysBetween(effectiveFrom, effectiveTo);
      return Array.from({ length: days + 1 }, (_, i) => {
        const date = addDays(effectiveFrom, i);
        const recs = ALL_RECORDS.filter((r) => r.date === date);
        return {
          date: date.slice(5),
          输入Tokens: recs.reduce((s, r) => s + r.inputTokens, 0),
          输出Tokens: recs.reduce((s, r) => s + r.outputTokens, 0),
        };
      });
    }
  }, [isSingleDay, effectiveFrom, effectiveTo, today]);

  // 按实例汇总（随时间联动），按总 token 降序
  // 普通模式用 MOCK_OPENCLAW_LIST，OneID 模式用 MOCK_CLAWS_WITH_DEPT
  const instanceList = hasOneid ? MOCK_CLAWS_WITH_DEPT : MOCK_OPENCLAW_LIST;
  const instanceStats = useMemo(() => {
    // 用实例 id 作为 seed 生成稳定的 mock 消耗数据
    return instanceList.map((inst, idx) => {
      const rand = seedRand(idx * 777 + 42);
      const days = daysBetween(effectiveFrom, effectiveTo) + 1;
      const requests = Math.floor(rand() * 200 * days + 10);
      const inputTokens = Math.floor(rand() * 30000 * days + 5000);
      const outputTokens = Math.floor(rand() * 25000 * days + 3000);
      return {
        id: inst.id,
        instanceId: inst.instanceId,
        name: inst.name,
        creator: (inst as any).creator ?? "",
        department: (inst as any).department ?? "",
        requests,
        inputTokens,
        outputTokens,
        total: inputTokens + outputTokens,
      };
    }).sort((a, b) => b.total - a.total);
  }, [instanceList, effectiveFrom, effectiveTo, hasOneid]);
  const instancePaged = instanceStats.slice((instancePage - 1) * PAGE_SIZE, instancePage * PAGE_SIZE);

  // 按用户汇总（随时间联动），按总请求数降序
  const memberStats = useMemo(() => {
    const map = new Map<string, { requests: number; inputTokens: number; outputTokens: number }>();
    rangeRecords.forEach((r) => {
      const cur = map.get(r.memberId) ?? { requests: 0, inputTokens: 0, outputTokens: 0 };
      map.set(r.memberId, {
        requests: cur.requests + r.requests,
        inputTokens: cur.inputTokens + r.inputTokens,
        outputTokens: cur.outputTokens + r.outputTokens,
      });
    });
    return Array.from(map.entries())
      .map(([id, v]) => ({ id, ...v, total: v.inputTokens + v.outputTokens }))
      .sort((a, b) => b.total - a.total);
  }, [rangeRecords]);

  // 按模型汇总（随时间联动），按总 token 降序
  const modelStats = useMemo(() => {
    const map = new Map<string, { requests: number; inputTokens: number; outputTokens: number }>();
    rangeRecords.forEach((r) => {
      const cur = map.get(r.modelName) ?? { requests: 0, inputTokens: 0, outputTokens: 0 };
      map.set(r.modelName, {
        requests: cur.requests + r.requests,
        inputTokens: cur.inputTokens + r.inputTokens,
        outputTokens: cur.outputTokens + r.outputTokens,
      });
    });
    return Array.from(map.entries())
      .map(([name, v]) => ({ name, ...v, total: v.inputTokens + v.outputTokens }))
      .sort((a, b) => b.total - a.total);
  }, [rangeRecords]);

  // 按会话汇总（高成本 TOP 5），按成本降序
  interface SessionStat {
    sessionId: string;
    sessionName: string;
    channel: string;
    model: string;
    lastActiveTime: string;
    rounds: number;
    tokens: number;
    cost: number;
    duration: string;
  }
  const sessionStats: SessionStat[] = [
    { sessionId: "fb766833", sessionName: "你能干啥 / 你管理一下我在伊朗的局势", channel: "Feishu Dm", model: "deepseek-v3.2", lastActiveTime: "2026-03-04 21:06", rounds: 63, tokens: 1950000, cost: 0.2743, duration: "454m 1s" },
    { sessionId: "06468225", sessionName: "我感觉现在仅表盘可观测细节这人，...", channel: "Feishu Dm", model: "deepseek-v3.2", lastActiveTime: "2026-03-08 13:14", rounds: 51, tokens: 1880000, cost: 0.2700, duration: "28m 52s" },
    { sessionId: "a9c7eb8b", sessionName: "请帮我列出 /etc 目录下所有 .conf ...", channel: "Webchat", model: "deepseek-v3.2", lastActiveTime: "2026-03-04 20:23", rounds: 47, tokens: 1590000, cost: 0.2242, duration: "12m 5s" },
    { sessionId: "a46be600", sessionName: "nihao / 帮我看看你的session-cost...", channel: "QQ Dm", model: "deepseek-v3.2", lastActiveTime: "2026-03-07 23:29", rounds: 35, tokens: 965000, cost: 0.1359, duration: "679m 41s" },
    { sessionId: "7bec562c", sessionName: "你还在吗 / 我是觉得现在 agent 仍...", channel: "Feishu Group", model: "hunyuan-turbos-latest", lastActiveTime: "2026-03-08 21:58", rounds: 28, tokens: 755000, cost: 0.1076, duration: "548m 57s" },
  ];
  const sessionPaged = sessionStats.slice((sessionPage - 1) * PAGE_SIZE, sessionPage * PAGE_SIZE);

  // 导出函数
  const runExport = (buildBlob: () => { blob: Blob; filename: string }) => {
    const tid = toast.loading("正在导出Tokens消耗明细列表");
    setTimeout(() => {
      const { blob, filename } = buildBlob();
      downloadBlob(blob, filename);
      toast.dismiss(tid);
    }, 500);
  };

  const handleExportInstance = () => runExport(() => {
    const header = hasOneid
      ? "实例名称,实例ID,用户ID,所属部门,总请求数,输入Tokens,输出Tokens,总Tokens"
      : "实例名称,实例ID,用户ID,总请求数,输入Tokens,输出Tokens,总Tokens";
    const rows = instanceStats.map((r) =>
      hasOneid
        ? `${r.name},${r.instanceId},${r.creator},${r.department},${r.requests},${r.inputTokens},${r.outputTokens},${r.total}`
        : `${r.name},${r.instanceId},${r.creator},${r.requests},${r.inputTokens},${r.outputTokens},${r.total}`
    );
    return { blob: makeCsvBlob(header, rows), filename: `tokens_by_instance_${effectiveFrom}_${effectiveTo}.csv` };
  });
  const handleExportMember = () => runExport(() => {
    const header = "用户ID,总请求数,输入Tokens,输出Tokens,总Tokens";
    const rows = memberStats.map((r) => `${r.id},${r.requests},${r.inputTokens},${r.outputTokens},${r.total}`);
    return { blob: makeCsvBlob(header, rows), filename: `tokens_by_member_${effectiveFrom}_${effectiveTo}.csv` };
  });
  const handleExportModel = () => runExport(() => {
    const header = "模型名称,总请求数,输入Tokens,输出Tokens,总Tokens";
    const rows = modelStats.map((r) => `${r.name},${r.requests},${r.inputTokens},${r.outputTokens},${r.total}`);
    return { blob: makeCsvBlob(header, rows), filename: `tokens_by_model_${effectiveFrom}_${effectiveTo}.csv` };
  });
  const handleExportDept = () => runExport(() => {
    const header = "部门名称,所属路径,总请求数,输入Tokens,输出Tokens,总Tokens";
    const rows = deptStats.map((r) => `${r.departmentName},${r.path},${r.requests},${r.inputTokens},${r.outputTokens},${r.totalTokens}`);
    return { blob: makeCsvBlob(header, rows), filename: `tokens_by_department_${effectiveFrom}_${effectiveTo}.csv` };
  });
  const handleExportSession = () => runExport(() => {
    const header = "会话ID,会话名称,渠道,模型,最后活动时间,轮次,Tokens,成本($),耗时";
    const rows = sessionStats.map((r) => `${r.sessionId},"${r.sessionName}",${r.channel},${r.model},${r.lastActiveTime},${r.rounds},${r.tokens},${r.cost.toFixed(4)},${r.duration}`);
    return { blob: makeCsvBlob(header, rows), filename: `tokens_by_session_${effectiveFrom}_${effectiveTo}.csv` };
  });

  // 翻页切片
  const memberPaged = memberStats.slice((memberPage - 1) * PAGE_SIZE, memberPage * PAGE_SIZE);
  const modelPaged = modelStats.slice((modelPage - 1) * PAGE_SIZE, modelPage * PAGE_SIZE);

  // 按部门汇总（OneID 模式使用）
  const findDeptAndChildren = (nodes: DepartmentNode[], targetId: string): string[] => {
    const ids: string[] = [];
    const collect = (node: DepartmentNode) => {
      ids.push(node.id);
      node.children?.forEach(collect);
    };
    const find = (list: DepartmentNode[]): boolean => {
      for (const n of list) {
        if (n.id === targetId) { collect(n); return true; }
        if (n.children && find(n.children)) return true;
      }
      return false;
    };
    find(nodes);
    return ids;
  };

  const deptStats = useMemo(() => {
    if (!hasOneid) return [];
    let data = MOCK_TOKEN_BY_DEPARTMENT;
    if (deptFilter) {
      const allowedIds = findDeptAndChildren(MOCK_DEPARTMENTS, deptFilter);
      data = data.filter((d) => allowedIds.includes(d.departmentId));
    }
    return data.sort((a, b) => b.totalTokens - a.totalTokens);
  }, [hasOneid, deptFilter]);
  const deptPaged = deptStats.slice((deptPage - 1) * PAGE_SIZE, deptPage * PAGE_SIZE);

  // 按分组汇总（普通模式和 OneID 模式都可见）
  const groupTree = hasOneid ? MOCK_GROUP_TREE_ONEID : MOCK_GROUP_TREE_MANUAL;
  const findGroupAndChildren = (nodes: GroupNode[], targetId: string): string[] => {
    const ids: string[] = [];
    const collect = (node: GroupNode) => {
      if (!node.id.startsWith("__section_")) ids.push(node.id);
      node.children?.forEach(collect);
    };
    const find = (list: GroupNode[]): boolean => {
      for (const n of list) {
        if (n.id === targetId) { collect(n); return true; }
        if (n.children && find(n.children)) return true;
      }
      return false;
    };
    find(nodes);
    return ids;
  };

  const groupStats = useMemo(() => {
    const rawData = hasOneid ? MOCK_TOKEN_BY_GROUP_ONEID : MOCK_TOKEN_BY_GROUP_MANUAL;
    let data = rawData;
    if (groupFilter) {
      const allowedIds = findGroupAndChildren(groupTree, groupFilter);
      data = data.filter((d) => allowedIds.includes(d.groupId));
    }
    return data.sort((a, b) => b.totalTokens - a.totalTokens);
  }, [hasOneid, groupFilter, groupTree]);
  const groupPaged = groupStats.slice((groupPage - 1) * PAGE_SIZE, groupPage * PAGE_SIZE);

  // ── 分组 → 全局 Tokens 上限映射 ──
  // 把 groupRule.groupIds 展开成"该规则覆盖的所有底层分组ID"
  type GroupQuotaEntry = { value: number | "unlimited"; startAt?: string; endAt?: string | null; refresh?: string };
  const groupLimitMap = useMemo(() => {
    const map = new Map<string, GroupQuotaEntry>();
    for (const rule of globalTokenGroupRules) {
      for (const gid of rule.groupIds) {
        const expanded = findGroupAndChildren(groupTree, gid);
        const ids = expanded.length > 0 ? expanded : [gid];
        for (const id of ids) {
          map.set(id, { value: rule.value, startAt: rule.startAt, endAt: rule.endAt, refresh: rule.refresh });
        }
      }
    }
    return map;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [globalTokenGroupRules, groupTree]);

  // Mock：标记为「无限制」的分组（用于展示无限制态）
  const MOCK_UNLIMITED_GROUP_IDS = new Set<string>(["og-ai-core", "mgrp-design"]);

  // 分组的"全局 Tokens 上限"展示文案
  const getGroupLimitDisplay = (groupId: string): string => {
    const entry = groupLimitMap.get(groupId);
    const isMockUnlimited = MOCK_UNLIMITED_GROUP_IDS.has(groupId);
    const limitVal = isMockUnlimited
      ? "unlimited"
      : (entry?.value ?? (globalLimit === null ? "unlimited" : globalLimit));
    const valStr = limitVal === "unlimited" || limitVal === null ? "无限制" : Number(limitVal).toLocaleString();

    if (globalTimeDimV2.type === "natural") {
      return `${PERIOD_PREFIX_MAP[globalTimeDimV2.period]} ${valStr}`;
    }
    // 自定义周期
    const startAt = entry?.startAt ?? GLOBAL_PRESET_START_AT;
    const endAt = entry?.endAt === undefined ? null : entry?.endAt;
    const refresh = (entry?.refresh ?? GLOBAL_PRESET_REFRESH) as string;
    const refreshStr = REFRESH_LABEL_MAP[(refresh || "daily") as CustomRefresh] ?? refresh;
    const endStr = endAt ? fmtTimeShort(endAt) : "无终止时间";
    return `${fmtTimeShort(startAt)} - ${endStr}，${refreshStr} ${valStr}`;
  };

  // 分组的"本周期 Tokens 消耗"窗口计算
  const getGroupPeriodWindow = (groupId: string): { label: string; startStr: string; endStr: string } | null => {
    if (globalTimeDimV2.type === "natural") {
      return { label: PERIOD_LABEL_MAP[globalTimeDimV2.period], startStr: "", endStr: "" };
    }
    // 自定义周期
    const entry = groupLimitMap.get(groupId);
    const startAt = entry?.startAt ?? GLOBAL_PRESET_START_AT;
    const endAt = entry?.endAt === undefined ? null : entry?.endAt;
    const refresh = ((entry?.refresh ?? GLOBAL_PRESET_REFRESH) || "daily") as string;
    const now = new Date();

    if (refresh === "none") {
      return { label: "", startStr: fmtTimeShort(startAt), endStr: endAt ? fmtTimeShort(endAt) : "无终止时间" };
    }
    // 按 refresh 滚动窗口
    let windowStart = new Date(startAt);
    while (true) {
      const next = new Date(windowStart);
      if (refresh === "daily") next.setDate(next.getDate() + 1);
      else if (refresh === "monthly") next.setMonth(next.getMonth() + 1);
      else if (refresh === "yearly") next.setFullYear(next.getFullYear() + 1);
      if (next > now) return { label: "", startStr: fmtTimeShort(windowStart.toISOString()), endStr: fmtTimeShort(next.toISOString()) };
      windowStart = next;
    }
  };

  // 给每个分组返回"按时间维度的消耗 / 上限"
  // mock：daily 用 totalTokens × 0.3，monthly 用 totalTokens 直接
  const getGroupQuotaInfo = (g: { groupId: string; totalTokens: number }) => {
    const entry = groupLimitMap.get(g.groupId);
    const limit = entry?.value;
    const consumed = globalTokenTimeDim === "daily"
      ? Math.round(g.totalTokens * 0.3)
      : g.totalTokens;
    if (MOCK_UNLIMITED_GROUP_IDS.has(g.groupId)) {
      return { unlimited: true, consumed, limit: null as number | null, pct: 0 };
    }
    if (limit === undefined) {
      // 未配置策略 → 落入兜底
      if (IS_GLOBAL_UNLIMITED) return { unlimited: true, consumed, limit: null as number | null, pct: 0 };
      const pct = globalLimit && globalLimit > 0 ? (consumed / globalLimit) * 100 : 0;
      return { unlimited: false, consumed, limit: globalLimit, pct };
    }
    if (limit === "unlimited" || limit === -1) {
      return { unlimited: true, consumed, limit: null as number | null, pct: 0 };
    }
    const num = Number(limit);
    const pct = num > 0 ? (consumed / num) * 100 : 0;
    return { unlimited: false, consumed, limit: num, pct };
  };

  // ─── 用户 Tab：单用户 Tokens 上限展示 ───
  const getUserTokenLimitDisplay = (quota: { tokenLimit: number; startAt?: string; endAt?: string | null; refresh?: string }): string => {
    const valStr = quota.tokenLimit <= 0 ? "无限制" : quota.tokenLimit.toLocaleString();
    if (userTimeDimV2.type === "natural") {
      return `${PERIOD_PREFIX_MAP[userTimeDimV2.period]} ${valStr}`;
    }
    const startAt = quota.startAt ?? "2025-01-01T00:00";
    const endAt = quota.endAt ? fmtTimeShort(quota.endAt) : "无终止时间";
    const refreshStr = REFRESH_LABEL_MAP[(quota.refresh || "daily") as CustomRefresh] ?? quota.refresh;
    return `${fmtTimeShort(startAt)} - ${endAt}，${refreshStr} ${valStr}`;
  };

  const getUserPeriodWindow = (startAt?: string, refresh?: string): { label: string; startStr: string; endStr: string } | null => {
    if (userTimeDimV2.type === "natural") {
      return { label: PERIOD_LABEL_MAP[userTimeDimV2.period], startStr: "", endStr: "" };
    }
    const sAt = startAt ?? "2025-01-01T00:00";
    const r = (refresh || "daily") as string;
    const now = new Date();
    if (r === "none") {
      return { label: "", startStr: fmtTimeShort(sAt), endStr: "无终止时间" };
    }
    let windowStart = new Date(sAt);
    while (true) {
      const next = new Date(windowStart);
      if (r === "daily") next.setDate(next.getDate() + 1);
      else if (r === "monthly") next.setMonth(next.getMonth() + 1);
      else if (r === "yearly") next.setFullYear(next.getFullYear() + 1);
      if (next > now) return { label: "", startStr: fmtTimeShort(windowStart.toISOString()), endStr: fmtTimeShort(next.toISOString()) };
      windowStart = next;
    }
  };

  const handleExportGroup = () => runExport(() => {
    const header = "分组名称,总请求数,输入Tokens,输出Tokens,总Tokens";
    const rows = groupStats.map((r) => `${r.groupName},${r.requests},${r.inputTokens},${r.outputTokens},${r.totalTokens}`);
    return { blob: makeCsvBlob(header, rows), filename: `tokens_by_group_${effectiveFrom}_${effectiveTo}.csv` };
  });

  return (
      <div className="page-enter">
        {/* Header */}




        <AdminPageHeader
          title="Tokens 监控"
          description={
            <span className="flex items-center gap-2">
              查看企业用户和模型的 Tokens 消耗情况。
              <UITooltip>
                <UITooltipTrigger asChild>
                  <button className="text-sm text-[#355EF1] hover:text-[#355EF1] hover:underline cursor-help transition-colors">
                    查看tokens统计规则
                  </button>
                </UITooltipTrigger>
                <UITooltipContent side="right" className="max-w-sm text-xs">
                  <div className="space-y-1.5">
                    <p>统计数据为模型 API 处理的全量 Token，包含输入 Token(缓存未命中)、输入 Token(缓存命中)、输出 Token。</p>
                    <p>缓存命中 Token 的实际计费价格通常远低于缓存未命中 Token。</p>
                    <p>因此页面展示的总 Token 数不等于等额的实际计费成本。</p>
                    <p>如需了解各模型的缓存输入 Token 定价，请参考对应模型提供商的官方计费文档。</p>
                  </div>
                </UITooltipContent>
              </UITooltip>
            </span>
          }
          actions={
            /* 停服态豁免：日期区间选择 + 刷新 均属「查看类」操作，
               只更新本地 state / 触发前端 mock 刷新，不发写请求，停服时保持可用。
               整组打 data-billing-exempt，overlay 灰化CSS 与点击拦截同时放行；
               「刷新」按钮自身若因 refreshing loading 传入 disabled，仍由原生 disabled 生效。*/
            <div className="flex items-center gap-2" data-billing-exempt>
              <DatePicker
                value={dateFrom}
                onChange={handleFromChange}
              />
              <span className="text-[var(--text-weak)] text-sm">—</span>
              <DatePicker
                value={dateTo}
                onChange={handleToChange}
              />
              <Button
                variant="claw-outline"
                size="icon"
                onClick={handleRefresh}
                disabled={refreshing}
                title="刷新数据"
                className="w-9 h-9"
              >
                <RefreshCw className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
              </Button>
            </div>
          }
        />

        {/* Overview Cards - 始终显示（样式对齐 feature/design-refresh-2026） */}
        <div className="grid grid-cols-5 gap-5 mb-6">
          {/* 随时间联动的四张卡片（迁移：NumberCard） */}
          <NumberCard icon={<RequestsIcon />} label="总请求数" value={fmt(totalRequests)} />
          <NumberCard icon={<InputTokensIcon />} label="输入 Tokens" value={fmt(totalInput)} />
          <NumberCard icon={<OutputTokensIcon />} label="输出 Tokens" value={fmt(totalOutput)} />
          <NumberCard icon={<TotalTokensIcon />} label="总 Tokens" value={fmt(totalTokens)} />

          {/* 当前周期全局配额消耗（按周期统计，功能保留，样式对齐 2026） */}
          <HoverCard openDelay={150} closeDelay={80}>
            <HoverCardTrigger asChild>
              <NumberCard
                className="cursor-default"
                icon={<TotalTokensIcon />}
                label={
                  <>
                    当前周期全局配额消耗
                    <UITooltip>
                      <UITooltipTrigger asChild>
                        <span className="cursor-default">
                          <Info className="w-3 h-3 text-[var(--text-weak)] hover:text-[var(--text-weak)] transition-colors" />
                        </span>
                      </UITooltipTrigger>
                      <UITooltipContent side="top" className="max-w-[280px] text-xs leading-relaxed">
                        {globalQuotaInfoTooltip}
                      </UITooltipContent>
                    </UITooltip>
                  </>
                }
                value={`${(IS_GLOBAL_BY_GROUP || IS_GLOBAL_UNLIMITED) ? "0" : currentPeriodPct}%`}
                extra={
                  IS_GLOBAL_BY_GROUP ? (
                    <span className="text-xs font-semibold text-[#355EF1] bg-[#e0e9ff] px-2.5 py-1.5 rounded-[4px]">按分组</span>
                  ) : IS_GLOBAL_UNLIMITED ? (
                    <span className="text-xs font-semibold text-[#355EF1] bg-[#e0e9ff] px-2.5 py-1.5 rounded-[4px]">无限制</span>
                  ) : (
                    <ProgressBar
                      value={periodPctNum}
                      max={100}
                      showTooltip={false}
                    />
                  )
                }
              />
            </HoverCardTrigger>
            <HoverCardContent side="bottom" align="end" className="w-72 p-4">
              <div className="space-y-3">
                {/* 全局 Tokens 上限 */}
                <div>
                  <p className="text-[10px] text-gray-400 mb-1">全局 Tokens 上限</p>
                  {globalLimitInfo.line1 && (
                    <p className="text-xs text-gray-700 tabular-nums">{globalLimitInfo.line1}</p>
                  )}
                  {globalLimitInfo.line2 && (
                    <p className={`text-xs tabular-nums ${globalLimitInfo.line1 ? "mt-0.5" : ""} text-gray-700`}>
                      {globalLimitInfo.line2}
                    </p>
                  )}
                </div>
                <div className="border-t border-gray-100" />
                {/* 本周期 Tokens 消耗 */}
                <div>
                  <p className="text-[10px] text-gray-400 mb-1">当前周期全局 Tokens 消耗</p>
                  {IS_GLOBAL_BY_GROUP ? (
                    <p className="text-xs text-gray-500">请在下方“按分组”Tab 查看具体分组的消耗</p>
                  ) : (
                    <>
                      <p className="text-xs text-gray-700 tabular-nums">
                        {currentPeriodWindow?.label
                          ? currentPeriodWindow.label
                          : `${currentPeriodWindow?.startStr} - ${currentPeriodWindow?.endStr}`}
                      </p>
                      <p className="text-xs text-gray-700 tabular-nums mt-0.5">
                        {IS_GLOBAL_UNLIMITED
                          ? `${TODAY_TOTAL_TOKENS.toLocaleString()} Tokens（无限制）`
                          : `${TODAY_TOTAL_TOKENS.toLocaleString()} / ${(globalLimit ?? 0).toLocaleString()} Tokens（${currentPeriodPct}%）`}
                      </p>
                    </>
                  )}
                </div>
              </div>
            </HoverCardContent>
          </HoverCard>
        </div>

        {/* Line Chart */}
        <div className="bg-white rounded-[4px] border border-gray-200 p-5 mb-6"
         >
          <p className="text-sm font-medium text-[var(--text-secondary)] mb-4">
            {isSingleDay ? "最近 7 天 Tokens 趋势" : "所选时间段 Tokens 趋势"}
          </p>
          <ResponsiveContainer width="100%" height={220}>
            <LineChart data={chartData} margin={{ top: 4, right: 16, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#F1F5F9" />
              <XAxis dataKey="date" tick={{ fontSize: 11, fill: "#94A3B8" }} axisLine={false} tickLine={false} />
              <YAxis tick={{ fontSize: 11, fill: "#94A3B8" }} axisLine={false} tickLine={false}
                tickFormatter={(v) => v >= 1000 ? `${(v / 1000).toFixed(0)}k` : v} />
              <Tooltip
                contentStyle={{ borderRadius: 10, border: "1px solid #E2E8F0", fontSize: 12, color: "#334155" }}
                labelStyle={{ color: "#64748B" }}
                itemStyle={{ color: "#334155" }}
                formatter={(value: number) => [value.toLocaleString(), ""]}
              />
              <Legend
                wrapperStyle={{ fontSize: 12 }}
                formatter={(value) => (
                  <span style={{ color: "var(--text-title)" }}>{value}</span>
                )}
              />
              <Line type="monotone" dataKey="输入Tokens" stroke="#1447E6" strokeWidth={2} dot={false} activeDot={{ r: 4 }} />
              <Line type="monotone" dataKey="输出Tokens" stroke="#60A5FA" strokeWidth={2} dot={false} activeDot={{ r: 4 }} />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* Detail Tabs */}
        {/* Detail Tabs
            停服态豁免：按实例/按用户/按模型/按部门/按分组/按会话 均为「视图切换」，
            只切换本地 tab state，不发写请求，停服时保持可用。整体挂在 TabsList 上，
            让内部所有 TabsTrigger 通过祖先选择器统一放行。*/}
        <Tabs defaultValue="instance">
          <TabsList data-billing-exempt className="flex items-center justify-start gap-2 border-b border-[#dbe6ff] mb-3 bg-transparent h-auto p-0 rounded-none w-full">
              <TabsTrigger value="instance" className="relative flex-none px-4 py-3 text-[14px] font-medium transition-colors whitespace-nowrap text-[var(--text-muted)] hover:text-[var(--text-title)] data-[state=active]:text-[var(--text-title)] data-[state=active]:border-b-2 data-[state=active]:border-[#0A0A0A] data-[state=active]:-mb-px bg-transparent shadow-none rounded-none h-auto border-0 data-[state=active]:bg-transparent data-[state=active]:shadow-none">按实例</TabsTrigger>
              <TabsTrigger value="member" className="relative flex-none px-4 py-3 text-[14px] font-medium transition-colors whitespace-nowrap text-[var(--text-muted)] hover:text-[var(--text-title)] data-[state=active]:text-[var(--text-title)] data-[state=active]:border-b-2 data-[state=active]:border-[#0A0A0A] data-[state=active]:-mb-px bg-transparent shadow-none rounded-none h-auto border-0 data-[state=active]:bg-transparent data-[state=active]:shadow-none">按用户</TabsTrigger>
              <TabsTrigger value="model" className="relative flex-none px-4 py-3 text-[14px] font-medium transition-colors whitespace-nowrap text-[var(--text-muted)] hover:text-[var(--text-title)] data-[state=active]:text-[var(--text-title)] data-[state=active]:border-b-2 data-[state=active]:border-[#0A0A0A] data-[state=active]:-mb-px bg-transparent shadow-none rounded-none h-auto border-0 data-[state=active]:bg-transparent data-[state=active]:shadow-none">按模型</TabsTrigger>
              {hasOneid && <TabsTrigger value="department" className="relative flex-none px-4 py-3 text-[14px] font-medium transition-colors whitespace-nowrap text-[var(--text-muted)] hover:text-[var(--text-title)] data-[state=active]:text-[var(--text-title)] data-[state=active]:border-b-2 data-[state=active]:border-[#0A0A0A] data-[state=active]:-mb-px bg-transparent shadow-none rounded-none h-auto border-0 data-[state=active]:bg-transparent data-[state=active]:shadow-none">按部门</TabsTrigger>}
              <TabsTrigger value="group" className="relative flex-none px-4 py-3 text-[14px] font-medium transition-colors whitespace-nowrap text-[var(--text-muted)] hover:text-[var(--text-title)] data-[state=active]:text-[var(--text-title)] data-[state=active]:border-b-2 data-[state=active]:border-[#0A0A0A] data-[state=active]:-mb-px bg-transparent shadow-none rounded-none h-auto border-0 data-[state=active]:bg-transparent data-[state=active]:shadow-none">按分组<span className="absolute top-1 right-0.5 w-1.5 h-1.5 bg-red-500 rounded-full" /></TabsTrigger>
               {/* data-billing-exempt: 停服态下「按会话」tab 标签保持可点击 */}
    <TabsTrigger value="session" data-billing-exempt className="relative flex-none px-4 py-3 text-[14px] font-medium transition-colors whitespace-nowrap text-[var(--text-muted)] hover:text-[var(--text-title)] data-[state=active]:text-[var(--text-title)] data-[state=active]:border-b-2 data-[state=active]:border-[#0A0A0A] data-[state=active]:-mb-px bg-transparent shadow-none rounded-none h-auto border-0 data-[state=active]:bg-transparent data-[state=active]:shadow-none">按会话</TabsTrigger>
          </TabsList>

          {/* 按实例 */}
          <TabsContent value="instance">
            <div className="flex items-center justify-between mb-3">
              <p className="text-sm text-[var(--text-secondary)]">汇总所选时间范围内每台实例的 Token 消耗，按总 Tokens 降序排序</p>
              <UITooltip>
                <UITooltipTrigger asChild>
                  <Button
                    variant="claw-outline"
                    size="icon"
                    className="w-9 h-9"
                    onClick={handleExportInstance}
                  >
                    <Download className="w-3.5 h-3.5" />
                  </Button>
                </UITooltipTrigger>
                <UITooltipContent side="top" className="text-xs">导出列表</UITooltipContent>
              </UITooltip>
            </div>
            <div className="bg-white rounded-[4px] border border-gray-200 overflow-hidden"
             >
              <Table variant="white">
                <TableHeader>
                  <TableRow>
                    <TableHead>名称 / ID</TableHead>
                    <TableHead>用户 ID</TableHead>
                    {hasOneid && <TableHead>所属部门</TableHead>}
                    <TableHead>总请求数</TableHead>
                    <TableHead>输入 Tokens</TableHead>
                    <TableHead>输出 Tokens</TableHead>
                    <TableHead>总 Tokens</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {instancePaged.length === 0 ? (
                    <TableRow><TableCell colSpan={hasOneid ? 7 : 6} className="text-center text-sm text-[var(--text-weak)] py-12">暂无数据</TableCell></TableRow>
                  ) : instancePaged.map((inst) => (
                    <TableRow key={inst.id}>
                      <TableCell>
                        <div className="min-w-0">
                          <UITooltip>
                            <UITooltipTrigger asChild>
                              <div className="text-sm font-medium text-[#09090b] truncate max-w-[180px]">{inst.name}</div>
                            </UITooltipTrigger>
                            <UITooltipContent side="top" className="text-xs max-w-xs break-all">{inst.name}</UITooltipContent>
                          </UITooltip>
                          <div className="text-xs font-mono text-[var(--text-muted)]">{inst.instanceId}</div>
                        </div>
                      </TableCell>
                      <TableCell className="text-sm text-[var(--text-muted)]">{inst.creator || "—"}</TableCell>
                      {hasOneid && <TableCell className="text-sm text-[var(--text-muted)]">{inst.department || "—"}</TableCell>}
                      <TableCell>{fmt(inst.requests)}</TableCell>
                      <TableCell>{fmt(inst.inputTokens)}</TableCell>
                      <TableCell>{fmt(inst.outputTokens)}</TableCell>
                      <TableCell className="font-medium">{fmt(inst.total)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              {/*
                停服态豁免（页面级，非组件库改动）：
                本页使用的是 shadcn 版 <Pagination>（@/components/ui/pagination），
                按"不动组件库维度代码"约束，改动落在页面上——为其外层已有的
                <div> 补 data-billing-exempt，触达 AdminDisabledOverlay 的两条恢复分支：
                  1) 视觉：.admin-service-suspended [data-billing-exempt] *恢复
                     opacity/cursor/pointer-events 到正常态；
                  2) 事件：文档级 click/mousedown 拦截通过
                     target.closest('[data-billing-exempt]') 命中后放行。
                "停服前已禁用则延续禁用"：<Pagination> 内部对首/末页按钮自身
                标注的disabled/aria-disabled 依旧生效（CSS 恢复分支包含
                :not([disabled]):not([aria-disabled="true"]) 保护），因此
                "首页时上一页 / 末页时下一页" 依然禁用。
              */}
              <div className="px-4 py-2 border-t border-gray-200" data-billing-exempt>
                <Pagination total={instanceStats.length} current={instancePage} pageSize={PAGE_SIZE} showTotal={(total) => `共 ${total} 条记录`} className="w-full justify-between" onChange={(p) => setInstancePage(p)} />
              </div>
            </div>
          </TabsContent>

          {/* 按用户 */}
          <TabsContent value="member">
            <div className="flex items-center justify-between mb-3">
              <p className="text-sm text-[var(--text-secondary)]">汇总所选时间范围内每个用户使用所有模型的消耗，按总 Tokens 降序排序</p>
              <UITooltip>
                <UITooltipTrigger asChild>
                  <Button
                    variant="claw-outline"
                    size="icon"
                    className="w-9 h-9"
                    onClick={handleExportMember}
                  >
                    <Download className="w-3.5 h-3.5" />
                  </Button>
                </UITooltipTrigger>
                <UITooltipContent side="top" className="text-xs">导出列表</UITooltipContent>
              </UITooltip>
            </div>
            <div className="bg-white rounded-[4px] border border-gray-100 overflow-hidden">
              <Table variant="white">
                <TableHeader>
                  <TableRow>
                    <TableHead className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wide">用户 ID</TableHead>
                    <TableHead className="text-right px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wide">总请求数</TableHead>
                    <TableHead className="text-right px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wide">输入 Tokens</TableHead>
                    <TableHead className="text-right px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wide">输出 Tokens</TableHead>
                    <TableHead className="text-right px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wide">总 Tokens</TableHead>
                                        <TableHead className="text-right px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wide whitespace-nowrap">
                      <span className="inline-flex items-center gap-1">
                        当前周期单用户配额消耗
                        <UITooltip>
                          <UITooltipTrigger asChild>
                            <Info className="w-3 h-3 text-gray-300 hover:text-gray-500 cursor-help" />
                          </UITooltipTrigger>
                          <UITooltipContent side="top" className="max-w-[280px] text-xs leading-relaxed normal-case font-normal tracking-normal">
                            单用户配额消耗 = 该用户调用公司配置模型产生的 Tokens 总和（不含用户自行配置的自定义模型）÷ 单用户 Tokens 上限。当前数字仅反映当前所属周期的累计占比，进入下一周期会按新周期重新统计。
                          </UITooltipContent>
                        </UITooltip>
                      </span>
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {memberPaged.length === 0 ? (
                    <TableRow><TableCell colSpan={6} className="px-6 py-12 text-center text-sm text-gray-400">暂无数据</TableCell></TableRow>
                  ) : memberPaged.map((m) => {
                    const userGroups = MOCK_USER_GROUPS[m.id];
                    const hasGroups = userGroups && userGroups.length > 0;
                    const isUnlimited = !hasGroups && UNLIMITED_USERS.has(m.id);
                    const consumed = m.total;
                    const userLimit = USER_PRESET_TOKEN_LIMIT;
                    const pct = userLimit > 0 ? ((consumed / userLimit) * 100).toFixed(1) : "0.0";
                    const pctNum = Number(pct);
                    const userBarColor = pctNum >= 80 ? "bg-red-500" : pctNum >= 60 ? "bg-orange-500" : "bg-blue-500";
                    return (
                    <TableRow key={m.id}>
                      <TableCell className="px-6 py-4" style={{ width: '220px', minWidth: '220px', maxWidth: '220px' }}>
                        <UITooltip>
                          <UITooltipTrigger asChild>
                            <span className="font-medium truncate block max-w-[180px]">{m.id}</span>
                          </UITooltipTrigger>
                          <UITooltipContent side="top" className="text-xs max-w-xs break-all">{m.id}</UITooltipContent>
                        </UITooltip>
                      </TableCell>
                      <TableCell className="px-6 py-4 text-sm text-gray-600 text-right">{fmt(m.requests)}</TableCell>
                      <TableCell className="px-6 py-4 text-sm text-gray-600 text-right">{fmt(m.inputTokens)}</TableCell>
                      <TableCell className="px-6 py-4 text-sm text-gray-600 text-right">{fmt(m.outputTokens)}</TableCell>
                      <TableCell className="px-6 py-4 text-sm font-medium text-gray-900 text-right">{fmt(m.total)}</TableCell>
                      {/* 单用户当前周期配额消耗（合并列） */}
                      <TableCell className="text-right whitespace-nowrap p-0" style={{ width: '220px' }}>
                        <HoverCard openDelay={150} closeDelay={80}>
                          <HoverCardTrigger asChild>
                            <div className="flex items-center justify-end gap-2 cursor-default w-full h-full px-4 py-4">
                              {hasGroups ? (
                                <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-50 text-blue-600 border border-blue-100">按分组</span>
                              ) : isUnlimited ? (
                                <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-emerald-50 text-emerald-600 border border-emerald-100">无限制</span>
                              ) : (
                                <>
                                  <div className="w-20 bg-gray-100 rounded-full h-1.5">
                                    <div className={`h-1.5 rounded-full ${userBarColor} transition-all`} style={{ width: `${Math.min(pctNum, 100)}%` }} />
                                  </div>
                                  <span className="text-sm font-medium text-gray-900 tabular-nums w-12 text-right">{pct}%</span>
                                </>
                              )}
                            </div>
                          </HoverCardTrigger>
                          <HoverCardContent side="left" align="start" className="w-80 p-4">
                            <div className="space-y-3">
                              <div>
                                <p className="text-[10px] text-gray-400 mb-1.5">单用户 Tokens 上限</p>
                                {hasGroups ? (
                                  <div className="space-y-2">
                                    {userGroups.map((g) => {
                                      const display = getUserTokenLimitDisplay(g);
                                      // 拆出"起止 - 终止" + "刷新策略 + 上限"
                                      const parts = display.split("，");
                                      return (
                                        <div key={g.groupId}>
                                          <p className="text-[11px] text-gray-500 mb-0.5">{g.groupName}</p>
                                          {parts.length === 2 ? (
                                            <>
                                              <p className="text-xs text-gray-700 tabular-nums">{parts[0]}</p>
                                              <p className="text-xs text-gray-700 tabular-nums">{parts[1]}</p>
                                            </>
                                          ) : (
                                            <p className="text-xs text-gray-700 tabular-nums">{display}</p>
                                          )}
                                        </div>
                                      );
                                    })}
                                  </div>
                                ) : isUnlimited ? (() => {
                                  const display = getUserTokenLimitDisplay({ tokenLimit: 0, startAt: "2025-02-15T00:00", endAt: null, refresh: "daily" });
                                  const parts = display.split("，");
                                  return parts.length === 2 ? (
                                    <>
                                      <p className="text-xs text-gray-700 tabular-nums">{parts[0]}</p>
                                      <p className="text-xs text-gray-700 tabular-nums mt-0.5">{parts[1]}</p>
                                    </>
                                  ) : (
                                    <p className="text-xs text-gray-700 tabular-nums">{display}</p>
                                  );
                                })() : (() => {
                                  const display = getUserTokenLimitDisplay({ tokenLimit: userLimit, startAt: "2025-02-15T00:00", endAt: null, refresh: "daily" });
                                  const parts = display.split("，");
                                  return parts.length === 2 ? (
                                    <>
                                      <p className="text-xs text-gray-700 tabular-nums">{parts[0]}</p>
                                      <p className="text-xs text-gray-700 tabular-nums mt-0.5">{parts[1]}</p>
                                    </>
                                  ) : (
                                    <p className="text-xs text-gray-700 tabular-nums">{display}</p>
                                  );
                                })()}
                              </div>
                              <div className="border-t border-gray-100" />
                              <div>
                                <p className="text-[10px] text-gray-400 mb-1.5">当前周期该用户 Tokens 消耗</p>
                                {hasGroups ? (
                                  <div className="space-y-2">
                                    {userGroups.map((g) => {
                                      const gPct = g.tokenLimit > 0 ? ((consumed * 0.5 / g.tokenLimit) * 100).toFixed(1) : "0.0";
                                      const gConsumed = Math.round(consumed * 0.5);
                                      const pw = getUserPeriodWindow(g.startAt, g.refresh);
                                      const periodText = pw?.label || `${pw?.startStr} - ${pw?.endStr}`;
                                      return (
                                        <div key={g.groupId}>
                                          <p className="text-[11px] text-gray-500 mb-0.5">{g.groupName}</p>
                                          <p className="text-xs text-gray-700 tabular-nums">{periodText}</p>
                                          <p className="text-xs text-gray-700 tabular-nums">{gConsumed.toLocaleString()} / {g.tokenLimit.toLocaleString()} Tokens（{gPct}%）</p>
                                        </div>
                                      );
                                    })}
                                  </div>
                                ) : isUnlimited ? (() => {
                                  const pw = getUserPeriodWindow("2025-02-15T00:00", "daily");
                                  const periodText = pw?.label || `${pw?.startStr} - ${pw?.endStr}`;
                                  return (
                                    <>
                                      <p className="text-xs text-gray-700 tabular-nums">{periodText}</p>
                                      <p className="text-xs text-gray-700 tabular-nums mt-0.5">{consumed.toLocaleString()} Tokens（无限制）</p>
                                    </>
                                  );
                                })() : (() => {
                                  const pw = getUserPeriodWindow("2025-02-15T00:00", "daily");
                                  const periodText = pw?.label || `${pw?.startStr} - ${pw?.endStr}`;
                                  return (
                                    <>
                                      <p className="text-xs text-gray-700 tabular-nums">{periodText}</p>
                                      <p className="text-xs text-gray-700 tabular-nums mt-0.5">{consumed.toLocaleString()} / {userLimit.toLocaleString()} Tokens（{pct}%）</p>
                                    </>
                                  );
                                })()}
                              </div>
                            </div>
                          </HoverCardContent>
                        </HoverCard>
                      </TableCell>
                    </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
              {/* 停服态豁免（页面级）：为分页器套一层 data-billing-exempt，
                 命中 AdminDisabledOverlay 的视觉/事件恢复分支。此处原本没有外层
                 包装 div，因此新增一个空 className 的 <div> —— 不产生任何布局/视觉影响。 */}
              <div data-billing-exempt>
                <Pagination current={memberPage} total={memberStats.length} pageSize={PAGE_SIZE} onChange={(p) => setMemberPage(p)} />
              </div>
            </div>
          </TabsContent>

          {/* 按模型 */}
          <TabsContent value="model">
            <div className="flex items-center justify-between mb-3">
              <p className="text-sm text-[var(--text-secondary)]">汇总所选时间范围内每个模型被所有企业用户使用的消耗，按总 Tokens 降序排序</p>
              <UITooltip>
                <UITooltipTrigger asChild>
                  <Button
                    variant="claw-outline"
                    size="icon"
                    className="w-9 h-9"
                    onClick={handleExportModel}
                  >
                    <Download className="w-3.5 h-3.5" />
                  </Button>
                </UITooltipTrigger>
                <UITooltipContent side="top" className="text-xs">导出列表</UITooltipContent>
              </UITooltip>
            </div>
            <div className="bg-white rounded-[4px] border border-gray-200 overflow-hidden"
             >
              <Table variant="white">
                <TableHeader>
                  <TableRow>
                    <TableHead>模型名称</TableHead>
                    <TableHead>总请求数</TableHead>
                    <TableHead>输入 Tokens</TableHead>
                    <TableHead>输出 Tokens</TableHead>
                    <TableHead>总 Tokens</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {modelPaged.length === 0 ? (
                    <TableRow><TableCell colSpan={5} className="text-center text-sm text-[var(--text-weak)] py-12">暂无数据</TableCell></TableRow>
                  ) : modelPaged.map((m) => (
                    <TableRow key={m.name}>
                      <TableCell className="font-medium">{m.name}</TableCell>
                      <TableCell>{fmt(m.requests)}</TableCell>
                      <TableCell>{fmt(m.inputTokens)}</TableCell>
                      <TableCell>{fmt(m.outputTokens)}</TableCell>
                      <TableCell className="font-medium">{fmt(m.total)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              <div className="px-4 py-2 border-t border-gray-200" data-billing-exempt>
                <Pagination total={modelStats.length} current={modelPage} pageSize={PAGE_SIZE} showTotal={(total) => `共 ${total} 条记录`} className="w-full justify-between" onChange={(p) => setModelPage(p)} />
              </div>
            </div>
          </TabsContent>

          {/* 按部门 - 仅 OneID 模式显示 */}
          {hasOneid && (
            <TabsContent value="department">
              <div className="flex items-center justify-between mb-3">
                <p className="text-sm text-[var(--text-secondary)]">汇总所选时间范围内各部门的消耗，按总 Tokens 降序排序</p>
                <div className="flex items-center gap-2">
                  <TreeSelect
                    nodes={MOCK_DEPARTMENTS}
                    value={deptFilter}
                    onChange={(v) => { setDeptFilter(v); setDeptPage(1); }}
                    allLabel="全部部门"
                    searchPlaceholder="搜索部门"
                    triggerWidth={160}
                    align="end"
                  />
                  <UITooltip>
                    <UITooltipTrigger asChild>
                      <Button
                        variant="claw-outline"
                        size="icon"
                        className="w-9 h-9"
                        onClick={handleExportDept}
                      >
                        <Download className="w-3.5 h-3.5" />
                      </Button>
                    </UITooltipTrigger>
                    <UITooltipContent side="top" className="text-xs">导出列表</UITooltipContent>
                  </UITooltip>
                </div>
              </div>
              <div className="bg-white rounded-[4px] border border-gray-200 overflow-hidden"
               >
                <Table variant="white">
                  <TableHeader>
                    <TableRow>
                      <TableHead>部门名称</TableHead>
                      <TableHead>所属路径</TableHead>
                      <TableHead>总请求数</TableHead>
                      <TableHead>输入 Tokens</TableHead>
                      <TableHead>输出 Tokens</TableHead>
                      <TableHead>总 Tokens</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {deptPaged.length === 0 ? (
                      <TableRow><TableCell colSpan={6} className="text-center text-sm text-[var(--text-weak)] py-12">暂无数据</TableCell></TableRow>
                    ) : deptPaged.map((d) => (
                      <TableRow key={d.departmentId}>
                        <TableCell className="font-medium">{d.departmentName}</TableCell>
                        <TableCell className="text-sm text-[var(--text-muted)]">{d.path.replace(/\//g, " / ")}</TableCell>
                        <TableCell>{fmt(d.requests)}</TableCell>
                        <TableCell>{fmt(d.inputTokens)}</TableCell>
                        <TableCell>{fmt(d.outputTokens)}</TableCell>
                        <TableCell className="font-medium">{fmt(d.totalTokens)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                <div className="px-4 py-2 border-t border-gray-200" data-billing-exempt>
                  <Pagination total={deptStats.length} current={deptPage} pageSize={PAGE_SIZE} showTotal={(total) => `共 ${total} 条记录`} className="w-full justify-between" onChange={(p) => setDeptPage(p)} />
                </div>
              </div>
            </TabsContent>
          )}

          {/* 按分组 */}
          <TabsContent value="group">
            <div className="flex items-center justify-between mb-3">
              <p className="text-sm text-[var(--text-secondary)]">汇总所选时间范围内各分组的消耗，按总 Tokens 降序排序</p>
              <div className="flex items-center gap-2">
                <TreeSelect
                  nodes={groupTree}
                  value={groupFilter}
                  onChange={(v) => { setGroupFilter(v); setGroupPage(1); }}
                  allLabel="全部分组"
                  searchPlaceholder="搜索分组"
                  triggerWidth={160}
                />
                <UITooltip>
                  <UITooltipTrigger asChild>
                    <Button
                      variant="claw-outline"
                      size="icon"
                      className="w-9 h-9"
                      onClick={handleExportGroup}
                    >
                      <Download className="w-3.5 h-3.5" />
                    </Button>
                  </UITooltipTrigger>
                  <UITooltipContent side="top" className="text-xs">导出列表</UITooltipContent>
                </UITooltip>
              </div>
            </div>
            <div className="bg-white rounded-[4px] border border-gray-200 overflow-hidden"
             >
              <Table variant="white">
                <TableHeader>
                  <TableRow>
                    <TableHead>分组名称</TableHead>
                    <TableHead>总请求数</TableHead>
                    <TableHead>输入 Tokens</TableHead>
                    <TableHead>输出 Tokens</TableHead>
                    <TableHead>总 Tokens</TableHead>
                    {IS_GLOBAL_BY_GROUP && (
                      <TableHead className="text-right px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wide whitespace-nowrap">
                        <span className="inline-flex items-center gap-1">
                          当前周期分组全局配额消耗
                          <UITooltip>
                            <UITooltipTrigger asChild>
                              <Info className="w-3 h-3 text-gray-300 hover:text-gray-500 cursor-help" />
                            </UITooltipTrigger>
                            <UITooltipContent side="top" className="max-w-[280px] text-xs leading-relaxed normal-case font-normal tracking-normal">
                              分组全局配额消耗 = 该分组下所有 Agent 实例使用公司配置模型产生的 Tokens 总和（不含用户自行配置的自定义模型）÷ 该分组的全局 Tokens 上限。当前数字仅反映当前所属周期的累计占比，进入下一周期会按新周期重新统计。
                            </UITooltipContent>
                          </UITooltip>
                        </span>
                      </TableHead>
                    )}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {groupPaged.length === 0 ? (
                    <TableRow><TableCell colSpan={IS_GLOBAL_BY_GROUP ? 6 : 5} className="text-center text-sm text-[var(--text-weak)] py-12">暂无数据</TableCell></TableRow>
                  ) : groupPaged.map((g) => {
                    const q = IS_GLOBAL_BY_GROUP ? getGroupQuotaInfo(g) : null;
                    const pctNum = q ? q.pct : 0;
                    const groupBarColor = pctNum >= 80 ? "bg-red-500" : pctNum >= 60 ? "bg-orange-500" : "bg-blue-500";
                    return (
                      <TableRow key={g.groupId}>
                        <TableCell className="font-medium">{g.groupName}</TableCell>
                        <TableCell>{fmt(g.requests)}</TableCell>
                        <TableCell>{fmt(g.inputTokens)}</TableCell>
                        <TableCell>{fmt(g.outputTokens)}</TableCell>
                        <TableCell className="font-medium">{fmt(g.totalTokens)}</TableCell>
                        {IS_GLOBAL_BY_GROUP && q && (
                          <TableCell className="text-right whitespace-nowrap p-0" style={{ width: '220px' }}>
                            <HoverCard openDelay={150} closeDelay={80}>
                              <HoverCardTrigger asChild>
                                <div className="flex items-center justify-end gap-2 cursor-default w-full h-full px-4 py-4">
                                  {q.unlimited ? (
                                    <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-emerald-50 text-emerald-600 border border-emerald-100">无限制</span>
                                  ) : (
                                    <>
                                      <div className="w-20 bg-gray-100 rounded-full h-1.5">
                                        <div className={`h-1.5 rounded-full ${groupBarColor} transition-all`} style={{ width: `${Math.min(pctNum, 100)}%` }} />
                                      </div>
                                      <span className="text-sm font-medium text-gray-900 tabular-nums w-14 text-right">{q.pct.toFixed(1)}%</span>
                                    </>
                                  )}
                                </div>
                              </HoverCardTrigger>
                              <HoverCardContent side="left" align="start" className="w-80 p-4">
                                <div className="space-y-3">
                                  <div>
                                    <p className="text-[10px] text-gray-400 mb-1.5">该分组的全局 Tokens 上限</p>
                                    {(() => {
                                      const display = getGroupLimitDisplay(g.groupId);
                                      const parts = display.split("，");
                                      return parts.length === 2 ? (
                                        <>
                                          <p className="text-xs text-gray-700 tabular-nums">{parts[0]}</p>
                                          <p className="text-xs text-gray-700 tabular-nums mt-0.5">{parts[1]}</p>
                                        </>
                                      ) : (
                                        <p className="text-xs text-gray-700 tabular-nums">{display}</p>
                                      );
                                    })()}
                                  </div>
                                  <div className="border-t border-gray-100" />
                                  <div>
                                    <p className="text-[10px] text-gray-400 mb-1.5">当前周期该分组全局 Tokens 消耗</p>
                                    {(() => {
                                      const pw = getGroupPeriodWindow(g.groupId);
                                      const periodText = pw?.label || `${pw?.startStr} - ${pw?.endStr}`;
                                      const consumedStr = q.consumed.toLocaleString();
                                      return (
                                        <>
                                          <p className="text-xs text-gray-700 tabular-nums">{periodText}</p>
                                          <p className="text-xs text-gray-700 tabular-nums mt-0.5">
                                            {q.unlimited
                                              ? `${consumedStr} Tokens（无限制）`
                                              : `${consumedStr} / ${q.limit !== null ? q.limit.toLocaleString() : "—"} Tokens（${q.pct.toFixed(1)}%）`}
                                          </p>
                                        </>
                                      );
                                    })()}
                                  </div>
                                </div>
                              </HoverCardContent>
                            </HoverCard>
                          </TableCell>
                        )}
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
              <div className="px-4 py-2 border-t border-gray-200" data-billing-exempt>
                <Pagination total={groupStats.length} current={groupPage} pageSize={PAGE_SIZE} showTotal={(total) => `共 ${total} 条记录`} className="w-full justify-between" onChange={(p) => setGroupPage(p)} />
              </div>
            </div>
          </TabsContent>

          {/* 按会话 */}
          {/* data-billing-exempt: 停服态下「按会话」内容区保持正常可用（仅此 tab，不影响其他 tab） */}
          <TabsContent value="session" data-billing-exempt>
            {!clsEnabled && (
              <>
                {/* CLS 提示弹框 */}
                <div className="bg-white border border-gray-200 rounded-[4px] p-6 mb-6">
                  <div className="flex items-start justify-between gap-6">
                    <div className="flex-1">
                      <h3 className="text-sm font-semibold text-[var(--text-title)] mb-1">Tokens 监控（按会话）需要开启 CLS 日志服务</h3>
                      <p className="text-xs text-[var(--text-muted)]">开启后，为您赠送3个月ClawPro 专属 CLS 日志服务免费额度，预估可覆盖 500台 Agent 机器3个月的日志用量；服务到期后，CLS 将按量计费。<a href="https://cloud.tencent.com/document/product/614/45802" target="_blank" className="text-[#355EF1] hover:underline inline-flex items-center gap-1">计费详情 <ArrowUpRight className="w-3 h-3" /></a></p>
                    </div>
                    <Button
                      onClick={handleOpenCLS}
                      disabled={isEnablingCls}
                      className="ml-4 text-xs h-8 px-4 whitespace-nowrap flex-shrink-0"
                    >
                      {isEnablingCls ? "开启中..." : "开启 CLS 日志服务"}
                    </Button>
                  </div>
                </div>

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
                        <Label htmlFor="cls-agreement" className="text-sm text-[#525252] cursor-pointer flex-1 font-normal leading-relaxed">
                          为您赠送三个月ClawPro 专属 CLS 日志服务免费额度，预估可覆盖 700 台 Agent 机器的日志用量；服务到期后，CLS 将按量计费。<a href="https://cloud.tencent.com/document/product/614/45802" target="_blank" className="text-[#355EF1] hover:text-[#355EF1] inline-flex items-center gap-1">计费详情 <ArrowUpRight className="w-3 h-3" /></a>
                        </Label>
                      </div>
                    </div>
                    <DialogFooter>
                      <Button
                        variant="outline"
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

                {/* 卡片功能展示 */}
                <div className="space-y-4 mb-8">
                  {/* 第一块：当前页可获得的会话数据 */}
                  <div className="bg-white border border-gray-200 rounded-[4px] px-6 py-5">
                    <h4 className="text-[14px] font-medium text-[var(--text-muted)] mb-4">开启CLS日志服务后您可以在此处获得以下会话数据：</h4>
                    <div className="grid grid-cols-2 gap-x-6 gap-y-5">
                      {[
                        {
                          id: "high-cost-session",
                          title: "高Token会话实时分析与管控",
                          description: "聚焦 TOP 会话的 Token 消耗、轮次分布与耗时特征，精准定位高Token交互，优化模型调用成本与资源效率",
                          iconSrc: "/assets/admin-session-management/high-token-session-control.svg",
                        },
                        {
                          id: "single-session-cost",
                          title: "单会话全链路Token透视",
                          description: "拆解每轮交互的 Token 流量与耗时分布，可视化工具调用与上下文膨胀对成本的影响",
                          iconSrc: "/assets/admin-session-management/single-session-token-insight.svg",
                        },
                      ].map((card) => (
                        <div
                          key={card.id}
                          className="flex items-center gap-[14px] py-5"
                        >
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
                  </div>

                  {/* 第二块：运维观测和会话管理中可获得的数据 */}
                  <div className="bg-white border border-gray-200 rounded-[4px] px-6 py-5">
                    <h4 className="text-[14px] font-medium text-[var(--text-muted)] mb-4">开启CLS日志服务后您还可以在运维观测和会话管理页面中获得以下观测数据：</h4>
                    <div className="grid grid-cols-2 gap-x-6 gap-y-5">
                      {[
                        {
                          id: "log-metrics-insight",
                          title: "应用日志与 OTEL 指标全景洞察",
                          description: "多维度分析日志级别与模块分布，精细化追踪消息处理、队列状态与执行耗时",
                          iconSrc: "/assets/admin-session-management/app-log-otel-insight.svg",
                        },
                        {
                          id: "session-efficiency",
                          title: "会话详情与交互效率精细化分析",
                          description: "聚焦单会话 Token 消耗，可视化渠道与模型分布特征，精准定位高Token会话，优化资源配置与调用效率",
                          iconSrc: "/assets/admin-session-management/session-detail-analysis.svg",
                        },
                        {
                          id: "health-monitoring",
                          title: "业务运行健康度实时监控",
                          description: "聚焦消息处理总量、入队效率与卡死会话，保障系统稳定运行",
                          iconSrc: "/assets/admin-session-management/business-health-monitoring.svg",
                        },
                        {
                          id: "session-global-monitoring",
                          title: "会话全局运行态势监控",
                          description: "聚合总会话数、平均轮次与工具调用量，多维度洞察渠道与模型分布，实现会话全生命周期可追溯、可分析",
                          iconSrc: "/assets/admin-session-management/session-global-monitoring.svg",
                        },
                      ].map((card) => (
                        <div
                          key={card.id}
                          className="flex items-center gap-[14px] py-5"
                        >
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
                  </div>
                </div>
              </>
            )}

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
            {clsEnabled && (
              <>

              {/* 顶部工具栏：左侧描述文案；右侧 Agent 下拉 + 操作按钮 */}
              <div className="flex items-center justify-between mb-3 gap-4">
                {/* 左侧：描述文案 */}
                <p className="text-sm text-[var(--text-secondary)]">全部会话已按 tokens 排序，点击可查看会话详情</p>
                {/* 右侧：Agent 筛选 + 操作按钮 */}
                <div className="flex items-center gap-2 shrink-0">
                  <SearchableSelect
                    options={[
                      { value: "", label: "全部 Agent" },
                      ...["Agent-A","Agent-B","Agent-C","Agent-D","Agent-E","Agent-F","Agent-G","Agent-H"].map((n) => ({ value: n, label: n })),
                    ]}
                    value={selectedAgent}
                    onChange={setSelectedAgent}
                    placeholder="全部 Agent"
                    searchPlaceholder="搜索 Agent 名称..."
                    showCount={false}
                    triggerClassName="w-[160px]"
                    align="end"
                  />
                  <UITooltip>
                    <UITooltipTrigger asChild>
                      <Button
                        variant="claw-outline"
                        size="claw"
                        className="px-2"
                        onClick={handleExportSession}
                      >
                        <Download className="w-3.5 h-3.5" />
                      </Button>
                    </UITooltipTrigger>
                    <UITooltipContent side="top" className="text-xs">导出列表</UITooltipContent>
                  </UITooltip>
                  <Button
                    onClick={() => setShowCloseClsConfirm(true)}
                    variant="claw-outline"
                    size="claw"
                  >
                    关闭CLS服务
                  </Button>
                  <Button
                    onClick={() => setShowPluginUpgradeDialog(true)}
                    variant="claw-primary"
                    size="claw"
                  >
                    升级CLS采集插件
                  </Button>
                </div>
              </div>
              <div className="bg-white rounded-[4px] border border-gray-200 overflow-hidden"
               >
                <Table variant="white" scrollX={1400}>
                  <TableHeader>
                    <TableRow>
                      <TableHead fixed="left">会话</TableHead>
                      <TableHead>渠道</TableHead>
                      <TableHead>模型</TableHead>
                      <TableHead>最后活动时间</TableHead>
                      <TableHead>轮次</TableHead>
                      <TableHead>TOKENS</TableHead>
                      <TableHead>成本</TableHead>
                      <TableHead>耗时</TableHead>
                      <TableHead fixed="right">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {sessionPaged.length === 0 ? (
                      <TableRow><TableCell colSpan={9} className="text-center text-sm text-[var(--text-weak)] py-12">暂无数据</TableCell></TableRow>
                    ) : sessionPaged.map((s) => {
                      return (
                      <TableRow key={s.sessionId} className="cursor-pointer" onClick={() => navigate(`/admin/session/${s.sessionId}`)}>
                        <TableCell fixed="left">
                          <div className="text-sm">{s.sessionName}</div>
                          <div className="text-xs text-[var(--text-weak)] font-mono mt-0.5">{s.sessionId}</div>
                        </TableCell>
                        <TableCell>{s.channel}</TableCell>
                        <TableCell>{s.model}</TableCell>
                        <TableCell className="text-[var(--text-muted)]">{s.lastActiveTime}</TableCell>
                        <TableCell>{s.rounds}</TableCell>
                        <TableCell className="font-mono">{(s.tokens / 1000000).toFixed(2)}M</TableCell>
                        <TableCell className="font-mono">${s.cost.toFixed(4)}</TableCell>
                        <TableCell className="text-[var(--text-muted)]">{s.duration}</TableCell>
                        <TableActionCell fixed="right">
                          <Button
                            variant="link"
                            onClick={(e) => {
                              e.stopPropagation();
                              navigate(`/admin/session/${s.sessionId}`);
                            }}
                          >
                            查看详情
                          </Button>
                        </TableActionCell>
                      </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
                <div className="px-4 py-2 border-t border-gray-200" data-billing-exempt>
                  <Pagination total={sessionStats.length} current={sessionPage} pageSize={PAGE_SIZE} showTotal={(total) => `共 ${total} 条记录`} className="w-full justify-between" onChange={(p) => setSessionPage(p)} />
                </div>
              </div>
              </>
            )}
          </TabsContent>
        </Tabs>

      {/* CLS 授权 Dialog */}
      <Dialog open={showAuthDialog} onOpenChange={setShowAuthDialog}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>开通服务授权</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 my-4">
            {!isCheckingAuth && !authCompleted && (
              <p className="text-sm text-[#525252]">开启CLS日志服务后您可以获取会话数据和观测数据</p>
            )}
            <div className="space-y-3 flex flex-col items-center min-h-16 justify-center">
              {isCheckingAuth ? (
                <>
                  {/* 检测中的旋转动画 */}
                  <div className="w-8 h-8 border-2 border-[#355EF1] border-t-[#355EF1] rounded-full animate-spin"></div>
                  <p className="text-xs text-[var(--text-muted)] text-center">检测中...</p>
                </>
              ) : authCompleted ? (
                <>
                  {/* 检测完成后显示完成 icon */}
                  <CheckCircle2 className="w-8 h-8 text-green-500" />
                  <p className="text-xs text-[var(--text-muted)] text-center">检测到已授权</p>
                </>
              ) : null}
            </div>
          </div>
          <DialogFooter className="flex gap-2 justify-end">
            <Button variant="outline" onClick={handleCancelAuth}>
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              onClick={handleGoToAuth}
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
            <DialogTitle>开启CLS日志服务-免费额度说明</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 my-4">
            <Alert variant="info">
              <AlertOperationInfoIcon />
              <AlertDescription>
                为您赠送<span className="font-semibold text-[#355EF1]">3个月</span>ClawPro 专属 CLS 日志服务免费额度（共<span className="font-semibold text-[#355EF1]">3000U</span>），预估可覆盖 <span className="font-semibold text-[#355EF1]">500台</span> Agent 机器<span className="font-semibold text-[#355EF1]">3个月</span>的日志用量；超过免费额度达到上限或<span className="font-semibold text-[#355EF1]">3个月</span>到期后，CLS 将按量计费。计费详情请参考{' '}
                <a
                  href="#"
                  onClick={(e) => {
                    e.preventDefault();
                    handleGoToCalcDetail();
                  }}
                  className="text-[#355EF1] hover:text-[#355EF1] underline"
                >
                  计费详情
                </a>
                。
              </AlertDescription>
            </Alert>
            <label className="flex items-center gap-2 cursor-pointer">
              <Checkbox
                id="free-quota-agreement"
                checked={freeQuotaAgreed}
                onCheckedChange={(checked) => setFreeQuotaAgreed(checked === true)}
              />
              <Label htmlFor="free-quota-agreement" className="text-sm text-[#525252] cursor-pointer font-normal">我已阅读并同意免费额度说明</Label>
            </label>
          </div>
          <DialogFooter className="flex gap-2 justify-end">
            <Button variant="outline" onClick={handleCancelFreeQuota}>
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

      {/* 关闭CLS服务 - 警示弹窗 */}
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

      {/* CLS 采集插件升级对话框 - 普通弹窗 */}
      <Dialog open={showPluginUpgradeDialog} onOpenChange={setShowPluginUpgradeDialog}>
        <DialogContent
          className="sm:max-w-[720px]"
          style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
        >
          <DialogHeader>
            <DialogTitle>升级 CLS 采集插件</DialogTitle>
            <DialogDescription>选择要升级的版本并查看更新内容</DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6 flex-1">
            <div className="rounded-[4px] border border-gray-200 overflow-hidden">
              <RadioGroup
                value={selectedPluginVersion?.version ?? ""}
                onValueChange={(val) => {
                  const v = CLS_PLUGIN_VERSIONS.find((x) => x.version === val);
                  if (v) setSelectedPluginVersion(v);
                }}
                className="contents"
              >
              <Table density="compact" autoFixedColumns={false}>
                <TableHeader>
                  <TableRow>
                    <TableHead style={{ width: 40 }} />
                    <TableHead style={{ width: 100 }}>版本号</TableHead>
                    <TableHead>更新内容</TableHead>
                    <TableHead style={{ width: 120 }}>状态</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {CLS_PLUGIN_VERSIONS.map((v) => {
                    const isUpgradeable = v.status !== 'current' && v.status !== 'deprecated';
                    return (
                      <TableRow
                        key={v.version}
                        onClick={() => isUpgradeable && setSelectedPluginVersion(v)}
                        className={isUpgradeable ? "cursor-pointer" : "cursor-default"}
                      >
                        <TableCell>
                          <RadioGroupItem
                            value={v.version}
                            disabled={!isUpgradeable}
                            aria-label={`选择版本 ${v.version}`}
                            onClick={(e) => e.stopPropagation()}
                          />
                        </TableCell>
                        <TableCell className="font-medium">{v.version}</TableCell>
                        <TableCell className="text-[#525252]">{v.changelog}</TableCell>
                        <TableCell>
                          {v.status === 'current' && <StatusTag mode="text" variant="green">当前版本</StatusTag>}
                          {v.status === 'deprecated' && <StatusTag mode="text" variant="gray">已弃用</StatusTag>}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
              </RadioGroup>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowPluginUpgradeDialog(false);
                setSelectedPluginVersion(null);
              }}
              disabled={isUpgradingPlugin}
            >
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              onClick={() => {
                setIsUpgradingPlugin(true);
                setTimeout(() => {
                  setIsUpgradingPlugin(false);
                  setShowPluginUpgradeDialog(false);
                  if (selectedPluginVersion) {
                    toast.success(`成功升级到 ${selectedPluginVersion?.version}`);
                  }
                }, 2000);
              }}
              disabled={isUpgradingPlugin || !selectedPluginVersion || selectedPluginVersion?.status === 'current'}
            >
              {isUpgradingPlugin ? "升级中..." : "确认升级"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
