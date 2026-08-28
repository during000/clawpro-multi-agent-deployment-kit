/**
 * AgentDetail - Agent 详细配置页
 * Design: 「流动蓝图」Fluid Blueprint
 * - 三栏布局：模型 | 通道 | 技能
 * - 参考图片风格：白色卡片，标题带彩色图标
 * - Header：名称、动态状态 badge（8 种状态）、一键更新、开启 Agent 面板
 * - 基础配置 Tab：模型配置、通道配置、技能配置
 */
import { useState, useEffect, useRef, useCallback } from "react";
import { useRoute, Link } from "wouter";
import TenantLayout from "@/components/TenantLayout";
import { Button } from "@/components/ui/button";
import {
  TenantSegment as Segment,
  TenantSegmentList as SegmentList,
  TenantSegmentItem as SegmentItem,
  TenantSegmentGroup,
  TenantSegmentOption,
} from "@/components/ui/segment";
import { Badge } from "@/components/ui/badge";
import { StatusTag } from "@/components/ui/status-tag";
import { TenantCard } from "@/components/ui/Surface";
import { Empty, EmptyHeader, EmptyMedia, EmptyDescription } from "@/components/ui/empty";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import { Progress } from "@/components/ui/progress";
import { toast } from "sonner";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  ArrowLeft, Trash2, EyeOff, Eye, X,
  Search, ExternalLink, Brain, MessageSquare, Puzzle,
  ChevronRight, ChevronDown, ChevronUp, Info, CheckCircle2, Loader2, CircleAlert, AlertCircle, ArrowUpCircle, Monitor, RotateCcw, XCircle, ArrowUpToLine, ArrowLeftRight,
  Copy, Terminal, Database, Clock, Shield, Lock, Megaphone,
  Plus, Sparkles, Mic, Send, Pencil, Wrench, RefreshCw, Check,
} from "lucide-react";
import { MOCK_OPENCLAW_LIST } from "@/lib/mockData";
import { findClawById, onClawListChange, saveClawList, loadClawList, notifyClawListChange, type AgentItem } from "@/lib/openclawStore";
import { isSharedToMe } from "@/lib/currentUser";
import { getActivePush, compareVersion, type ActivePush } from "@/lib/upgradePushStore";
import FileSpace from "./FileSpace";
import MemoryPreview from "@/components/MemoryPreview";
import { SkillManagementUpdateNotice } from "@/components/agent/SkillManagementUpdateNotice";
import ToolsMcpPanel from "./ToolsMcpPanel";
import { queryDiagnosisStatus, startDiagnosis, endDiagnosis, getCallerId, type InitiatorType } from "@/lib/doctorDiagnosisApi";
import { groupStore } from "../admin/MemberManagement/groupStore";
import { projectAssetStore } from "../admin/project-assets/projectAssetStore";
import { getAssetItemDisplay } from "../admin/project-assets/assetSelectors";

// ─── 实例状态配置（与 MyAgent 保持一致） ──────────────────────────────────────

type AgentStatus = "creating" | "createFail" | "running" | "shutdown" | "loading" | "loadFail" | "maintaining" | "pending";

const INSTANCE_STATUS_CONFIG: Record<AgentStatus, {
  label: string;
  badgeClass: string;
  dotColor?: string;
  spinning?: boolean;
  tooltipText?: string;
}> = {
  creating: {
    label: "创建中",
    badgeClass: "badge-loading",
    dotColor: "#355EF1",
    spinning: true,
    tooltipText: "正在创建中，请稍候",
  },
  createFail: {
    label: "创建失败",
    badgeClass: "badge-stopped",
    dotColor: "#FF3B30",
    tooltipText: "创建失败，可删除后重新创建",
  },
  running: {
    label: "运行中",
    badgeClass: "badge-running",
    dotColor: "#34C759",
  },
  shutdown: {
    label: "已关机",
    badgeClass: "badge-shutdown",
    dotColor: "#9CA3AF",
    tooltipText: "已关机，如需恢复请联系管理员",
  },
  loading: {
    label: "加载中",
    badgeClass: "badge-loading",
    dotColor: "#355EF1",
    spinning: true,
    tooltipText: "加载中，请稍候",
  },
  loadFail: {
    label: "加载失败",
    badgeClass: "badge-stopped",
    dotColor: "#FF3B30",
    tooltipText: "加载失败，可点击重试恢复",
  },
  maintaining: {
    label: "维护中",
    badgeClass: "badge-pending",
    dotColor: "#FF9500",
    tooltipText: "维护中，请稍候",
  },
  pending: {
    label: "待处理",
    badgeClass: "badge-stopped",
    dotColor: "#FF3B30",
    tooltipText: "已停用，请联系管理员处理",
  },
};
import {
  type CustomChannel as AdminCustomChannel,
  loadVisibleCustomChannels,
  onCustomChannelsChange,
  loadBuiltinChannelVisibility,
  onBuiltinChannelVisibilityChange,
} from "@/lib/customChannelStore";

// ─── 通道 / 模型配置共享常量（迁移到 lib/agentConfigConstants） ─────────────────

import {
  MODEL_PROVIDERS,
  getAdminModelProviders,
  getSelfProvidersByCategory,
  MODEL_PROVIDER_GROUP_LABELS,
  SELF_CONFIG_CATEGORY_LABELS,
  SELF_CONFIG_CATEGORY_ORDER,
  CHANNEL_OPTIONS,
  DEFAULT_CUSTOM_JSON,
  type SelfConfigCategory,
  type ChannelField,
  type ChannelConfig,
} from "@/lib/agentConfigConstants";

// ─── 工具函数 ────────────────────────────────────────────────────────────────────

/** 加密显示：保留前3字符，后面用 •••••• 替代 */
function maskSecret(val: string): string {
  if (!val) return "";
  if (val.length <= 3) return val;
  return val.slice(0, 3) + "••••••";
}

// ─── 已接入通道数据结构 ───────────────────────────────────────────────────────────

type AppliedChannel = {
  type: string;       // label
  channelValue: string; // value key
  status: "running";
  fields: ChannelField[];
  fieldValues: Record<string, string>;
  feishuConfigMode?: "quick" | "manual"; // 飞书专用
  weworkConfigMode?: "quick" | "manual"; // 企业微信专用
};

const DEFAULT_LOCAL_AGENT_STANDARDS = ["Agent 安全合规基线", "交付协作规范"];

const getRecentLocalAgentReportedAt = () =>
  new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString().slice(0, 19).replace("T", " ");

const getInstalledStandards = (claw: AgentItem): string[] => {
  const raw = (claw as any).standards;
  if (Array.isArray(raw) && raw.length > 0) {
    return raw
      .map((item) => {
        if (typeof item === "string") return item;
        if (item && typeof item === "object") return item.name || item.title || item.slug;
        return "";
      })
      .filter(Boolean);
  }
  return (claw as any).agentType === "localagent" ? DEFAULT_LOCAL_AGENT_STANDARDS : [];
};

const getLocalAgentDetailFallback = (id?: string): AgentItem | undefined => {
  if (id === "oc-local-workbuddy-demo") {
    return {
      id: "oc-local-workbuddy-demo",
      instanceId: "local-workbuddy-01",
      name: "WorkBuddy-运营笔记本",
      status: "running",
      agentType: "localagent",
      localProduct: "WorkBuddy",
      localConnectionStatus: "connected",
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
    };
  }
  if (id === "oc-local-workbuddy-offline-demo") {
    return {
      id: "oc-local-workbuddy-offline-demo",
      instanceId: "local-workbuddy-02",
      name: "WorkBuddy-离线笔记本",
      status: "running",
      agentType: "localagent",
      localProduct: "WorkBuddy",
      localConnectionStatus: "disconnected",
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
    };
  }
  return undefined;
};

// 演示用「他人共享给我」的实例 fallback。
// 这些卡片（gc-shared-*）在列表页仅内存追加、不落 store，故详情页 findClawById 取不到，
// 此处补一份只读演示数据，保证点进详情能正确命中「被共享 → 只读」链路。
const getSharedDemoDetailFallback = (id?: string): AgentItem | undefined => {
  if (id === "gc-shared-01") {
    return {
      id: "gc-shared-01",
      instanceId: "ins-shared-01",
      name: "团队协作助手",
      status: "running",
      agentType: "openclaw",
      createdAt: "2026-03-25",
      model: "腾讯云 DeepSeek",
      modelVersion: "DeepSeek V3 0324",
      channels: ["飞书"],
      skills: [],
      groupName: "前端组",
      creator: "bob@acompany.com",
      owner: "bob@acompany.com",
      shareScope: "shared",
      shareGroupIds: ["grp-fe"],
      shareGroupNames: ["前端组"],
      billingMode: "payg",
      hourlyRate: 1.5,
      runningMinutes: 1200,
    };
  }
  if (id === "gc-shared-02") {
    return {
      id: "gc-shared-02",
      instanceId: "ins-shared-02",
      name: "客服知识库助手",
      status: "running",
      agentType: "openclaw",
      createdAt: "2026-03-28",
      model: "腾讯云混元",
      modelVersion: "混元 Turbo",
      channels: ["企业微信"],
      skills: [],
      groupName: "运营组",
      creator: "carol@acompany.com",
      owner: "carol@acompany.com",
      shareScope: "shared",
      shareUserIds: ["bob@a.com", "dave@a.com", "grace@a.com"],
      shareUserNames: ["Bob", "Dave", "Grace"],
      billingMode: "subscription",
    };
  }
  return undefined;
};

// ─── 主组件 ──────────────────────────────────────────────────────────────────────

export default function AgentDetail() {
  const [, params] = useRoute("/openclaw/:id");
  const clawId = params?.id;

  // 优先从共享 store 读取（包含动态创建的 claw 及 roleName），fallback 到 mock 数据
  const [clawData, setClawData] = useState<AgentItem>(() =>
    (clawId ? findClawById(clawId) : undefined) ??
    getLocalAgentDetailFallback(clawId) ??
    getSharedDemoDetailFallback(clawId) ??
    (MOCK_OPENCLAW_LIST.find((c) => c.id === clawId) ?? MOCK_OPENCLAW_LIST[0]) as unknown as AgentItem
  );
  useEffect(() => {
    const unsub = onClawListChange(() => {
      if (clawId) {
        const updated = findClawById(clawId);
        if (updated) setClawData(updated);
      }
    });
    return unsub;
  }, [clawId]);
  const claw = clawData;
  const isLocalAgent = (claw as any).agentType === "localagent";
  const installedStandards = getInstalledStandards(claw);

  // 「他人共享给我」→ 详情页全部配置只读（仅可查看，不可修改）。
  const isReadOnly = isSharedToMe(claw);
  const sharedOwnerName = (claw.owner ?? claw.creator ?? "").split("@")[0];

  const clawName = claw.name;
  const clawStatus = (claw.status || "running") as AgentStatus;
  const statusCfg = INSTANCE_STATUS_CONFIG[clawStatus] ?? INSTANCE_STATUS_CONFIG.running;
  const canRenameByStatus = !isReadOnly && (["running", "maintaining", "shutdown"] as AgentStatus[]).includes(clawStatus);


  // ── Inline rename state ──
  const AGENT_NAME_MAX_BYTES = 128;
  const getAgentNameByteLength = (value: string) => new TextEncoder().encode(value).length;
  const [isNameEditing, setIsNameEditing] = useState(false);
  const [nameDraft, setNameDraft] = useState("");
  const [nameError, setNameError] = useState<string>("");
  const [isNameOverflow, setIsNameOverflow] = useState(false);
  const nameEditWrapperRef = useRef<HTMLDivElement | null>(null);
  const nameInputRef = useRef<HTMLInputElement | null>(null);
  const nameTextRef = useRef<HTMLHeadingElement | null>(null);

  const startNameEdit = useCallback(() => {
    if (!canRenameByStatus) return;
    setIsNameEditing(true);
    setNameDraft(clawName);
    setNameError("");
  }, [canRenameByStatus, clawName]);


  const cancelNameEdit = useCallback(() => {
    setIsNameEditing(false);
    setNameDraft(clawName);
    setNameError("");
  }, [clawName]);

  const saveNameEdit = useCallback(() => {
    if (isReadOnly) {
      setIsNameEditing(false);
      return false;
    }
    const trimmed = nameDraft.trim();

    if (!trimmed) {
      cancelNameEdit();
      return true;
    }

    if (getAgentNameByteLength(trimmed) > AGENT_NAME_MAX_BYTES) {
      setNameError(`名称不能超过 ${AGENT_NAME_MAX_BYTES} 字节`);
      return false;
    }

    if (trimmed === clawName) {
      setIsNameEditing(false);
      setNameError("");
      return true;
    }

    try {
      const latest = loadClawList();
      const next = latest.map((item) => (item.id === claw.id ? { ...item, name: trimmed } : item));
      saveClawList(next);
      notifyClawListChange();
      setClawData((prev) => ({ ...prev, name: trimmed }));
      setIsNameEditing(false);
      setNameError("");
      return true;
    } catch {
      setNameError("重命名失败，请重试");
      return false;
    }
  }, [nameDraft, clawName, claw.id, cancelNameEdit, isReadOnly]);

  useEffect(() => {
    if (!isNameEditing) return;
    const frame = requestAnimationFrame(() => {
      if (!nameInputRef.current) return;
      nameInputRef.current.focus();
      nameInputRef.current.select();
    });
    return () => cancelAnimationFrame(frame);
  }, [isNameEditing]);

  useEffect(() => {
    if (!isNameEditing) return;

    const handlePointerDown = (e: MouseEvent) => {
      if (!nameEditWrapperRef.current) return;
      if (nameEditWrapperRef.current.contains(e.target as Node)) return;
      saveNameEdit();
    };

    document.addEventListener("mousedown", handlePointerDown);
    return () => document.removeEventListener("mousedown", handlePointerDown);
  }, [isNameEditing, saveNameEdit]);

  useEffect(() => {
    if (isNameEditing) return;
    const el = nameTextRef.current;
    if (!el) return;

    const updateOverflow = () => {
      setIsNameOverflow(el.scrollWidth > el.clientWidth);
    };

    updateOverflow();
    const resizeObserver = new ResizeObserver(updateOverflow);
    resizeObserver.observe(el);
    window.addEventListener("resize", updateOverflow);

    return () => {
      resizeObserver.disconnect();
      window.removeEventListener("resize", updateOverflow);
    };
  }, [clawName, isNameEditing]);

  // ── Configuration state ──
  const [isConfiguring, setIsConfiguring] = useState(false); // 配置中状态
  const [quickFixState, setQuickFixState] = useState<"idle" | "loading" | "success" | "failed">("idle");
  const [quickFixFailReason, setQuickFixFailReason] = useState<string>("");

  // ─── 一键修复 · MOCK（仅前端 demo 演示用） ────────────────────────────────
  // 真接口接入时删除整段，改为读后端响应。
  //
  // 节奏（按 PM 要求）：每 3 次尝试 → 2 次成功 + 1 次失败，顺序随机洗牌。
  //   做法：维护一个长度为 3 的"批次队列"[success, success, failed]，每次取一个，
  //   取空就重新洗牌。比"按固定次序"更接近真实接口的随机感，又能严格保证 1/3 失败率。
  //
  // 状态停留策略：success / failed 均常驻展示，不自动消失。
  //   设计依据：失败时用户需要充分阅读失败原因再决策"是否开启龙虾医生"，
  //   成功时也希望用户明确感知一次操作的最终结果。
  //   重新触发途径：用户离开当前 OpenClaw 详情页（路由切走 / 刷新页面）再回来时，
  //   组件 state 重置回 idle，主操作区"一键修复"按钮重新出现。
  const quickFixBatchRef = useRef<("success" | "failed")[]>([]);
  const quickFixFailReasonsRef = useRef([
    "API KEY 校验未通过",
    "插件依赖加载超时",
    "通道配置文件解析异常",
  ]);
  const quickFixFailIdxRef = useRef(0);

  const drawQuickFixOutcome = (): "success" | "failed" => {
    if (quickFixBatchRef.current.length === 0) {
      // Fisher-Yates 洗牌一个新批次：[success, success, failed]
      const batch: ("success" | "failed")[] = ["success", "success", "failed"];
      for (let i = batch.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [batch[i], batch[j]] = [batch[j], batch[i]];
      }
      quickFixBatchRef.current = batch;
    }
    return quickFixBatchRef.current.shift()!;
  };

  const runQuickFixMock = useCallback(() => {
    setQuickFixState("loading");
    setQuickFixFailReason("");
    setTimeout(() => {
      const outcome = drawQuickFixOutcome();
      if (outcome === "failed") {
        const reasons = quickFixFailReasonsRef.current;
        const reason = reasons[quickFixFailIdxRef.current % reasons.length];
        quickFixFailIdxRef.current += 1;
        setQuickFixFailReason(reason);
        setQuickFixState("failed");
      } else {
        setQuickFixState("success");
        toast.success("一键修复执行完成");
      }
    }, 3000);
  }, []);

  // 读取管控端「允许用户使用龙虾医生」开关状态（默认关闭）
  const [lobsterDoctorEnabled, setLobsterDoctorEnabled] = useState(
    () => localStorage.getItem("admin_allow_lobster_doctor") === "true"
  );
  // 监听 localStorage 变化，管控端切换开关后用户端实时响应
  // - 跨 tab：原生 storage 事件
  // - 同 tab：管控端派发的 lobster-doctor-policy-changed CustomEvent
  useEffect(() => {
    const handleStorage = (e: StorageEvent) => {
      if (e.key === "admin_allow_lobster_doctor") {
        setLobsterDoctorEnabled(e.newValue === "true");
      }
    };
    const handleCustom = (e: Event) => {
      const detail = (e as CustomEvent<{ enabled: boolean }>).detail;
      if (detail && typeof detail.enabled === "boolean") {
        setLobsterDoctorEnabled(detail.enabled);
      } else {
        setLobsterDoctorEnabled(localStorage.getItem("admin_allow_lobster_doctor") === "true");
      }
    };
    window.addEventListener("storage", handleStorage);
    window.addEventListener("lobster-doctor-policy-changed", handleCustom);
    return () => {
      window.removeEventListener("storage", handleStorage);
      window.removeEventListener("lobster-doctor-policy-changed", handleCustom);
    };
  }, []);

  // 读取管控端「允许用户配置模型」开关状态（默认开启）
  const [userConfigModelEnabled, setUserConfigModelEnabled] = useState(() => {
    const v = localStorage.getItem("admin_allow_user_config_model");
    return v !== null ? v === "true" : true;
  });
  // 读取管控端「允许用户配置通道」开关状态（默认开启）
  const [userConfigChannelEnabled, setUserConfigChannelEnabled] = useState(() => {
    const v = localStorage.getItem("admin_allow_user_config_channel");
    return v !== null ? v === "true" : true;
  });
  useEffect(() => {
    const handleStorage = (e: StorageEvent) => {
      if (e.key === "admin_allow_user_config_model") {
        setUserConfigModelEnabled(e.newValue !== null ? e.newValue === "true" : true);
      }
      if (e.key === "admin_allow_user_config_channel") {
        setUserConfigChannelEnabled(e.newValue !== null ? e.newValue === "true" : true);
      }
    };
    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, []);

  // ── Model state ──
  // 模型来源：admin = 管理员配置的模型；self = 自行配置
  const [modelSource, setModelSource] = useState<"admin" | "self">("admin");
  // 「自行配置」下的二级分类（Coding Plan / 模型 API / 自定义模型）
  const [selfCategory, setSelfCategory] = useState<SelfConfigCategory>("codingPlan");
  const [selectedProvider, setSelectedProvider] = useState(MODEL_PROVIDERS[0].value);
  const [selectedModel, setSelectedModel] = useState(MODEL_PROVIDERS[0].versions[0].value);
  const [customInputMode, setCustomInputMode] = useState<"json" | "form">("json");
  const [customJson, setCustomJson] = useState(DEFAULT_CUSTOM_JSON);
  const [customForm, setCustomForm] = useState({ provider: "", base_url: "", api: "", api_key: "", model_id: "", model_name: "" });
  const [customMultimodal, setCustomMultimodal] = useState(false);
  // ── 自行配置·公开模型 state（无需模型 URL，仅 API Key + 限额）──
  const [publicForm, setPublicForm] = useState({ apiKey: "", dailyLimit: "100000" });
  // 高级配置
  type HeaderEntry = { key: string; value: string };
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [advancedConfig, setAdvancedConfig] = useState<{ contextWindow: string; maxTokens: string; headers: HeaderEntry[] }>({
    contextWindow: "",
    maxTokens: "",
    headers: [{ key: "", value: "" }],
  });
  // 多条模型列表，每条有唯一 id；primary 表示主模型，其余为备选模型，所有模型均为应用中状态
  type AppliedModel = { id: number; providerLabel: string; versionLabel: string; primary: boolean; isCustom: boolean; customName: string; addedAt: number; multimodal?: boolean; };
  const [appliedModels, setAppliedModels] = useState<AppliedModel[]>([
    { id: 1, providerLabel: "腾讯云 DeepSeek", versionLabel: "DeepSeek V3 0324", primary: true, isCustom: false, customName: "", addedAt: Date.now() },
  ]);
  const [modelIdCounter, setModelIdCounter] = useState(2);
  // 模型操作二次确认弹窗
  const [modelConfirmDialog, setModelConfirmDialog] = useState<{
    open: boolean;
    type: "set-primary" | "delete" | "delete-backup";
    modelId: number | null;
  }>({ open: false, type: "set-primary", modelId: null });

  // 连通性检测
  const [connectTesting, setConnectTesting] = useState(false);
  const [connectFailResult, setConnectFailResult] = useState<string | null>(null);

  const handleConnectTest = async () => {
    if (isPublicSelfProvider) {
      if (!selectedModel || !publicForm.apiKey) {
        toast.error("请填写完整的模型配置信息");
        return;
      }
    } else if (customInputMode === "form") {
      if (!customForm.base_url || !customForm.api_key || !customForm.model_id) {
        toast.error("请填写完整的模型配置信息");
        return;
      }
    } else {
      if (!customJson.trim()) {
        toast.error("请填写完整的模型配置信息");
        return;
      }
    }
    setConnectTesting(true);
    await new Promise((r) => setTimeout(r, 1500));
    setConnectTesting(false);
    setConnectFailResult(JSON.stringify({
      error: {
        message: "Invalid API Key",
        param: "Please provide valid API Key",
        code: "401",
        type: "invalid_key",
      }
    }, null, 2));
  };

  const currentProvider = MODEL_PROVIDERS.find(p => p.value === selectedProvider) || MODEL_PROVIDERS[0];
  const currentVersions = currentProvider.versions;
  // 当前生效的厂商候选列表：admin 来源 = 管理员模型；self 来源 = 当前类别下的厂商
  const sourceProviders = modelSource === "admin"
    ? getAdminModelProviders()
    : getSelfProvidersByCategory(selfCategory);
  // 当前选中厂商是否为「自定义模型」/「自行配置的公开模型」
  const isCustomProvider = modelSource === "self" && selfCategory === "custom";
  const isPublicSelfProvider = modelSource === "self" && selfCategory !== "custom";

  const handleProviderChange = (providerValue: string) => {
    setSelectedProvider(providerValue);
    const provider = MODEL_PROVIDERS.find(p => p.value === providerValue);
    if (provider) setSelectedModel(provider.versions[0].value);
  };

  // 切换「来源」（管理员 / 自行配置）：自动定位到该来源下第一个厂商
  const handleModelSourceChange = (source: "admin" | "self") => {
    setModelSource(source);
    const list = source === "admin" ? getAdminModelProviders() : getSelfProvidersByCategory(selfCategory);
    if (list.length > 0) {
      setSelectedProvider(list[0].value);
      setSelectedModel(list[0].versions[0]?.value ?? "");
    }
  };

  // 切换「自行配置」二级分类：自动定位到该分类下第一个厂商
  const handleSelfCategoryChange = (category: SelfConfigCategory) => {
    setSelfCategory(category);
    const list = getSelfProvidersByCategory(category);
    if (list.length > 0) {
      setSelectedProvider(list[0].value);
      setSelectedModel(list[0].versions[0]?.value ?? "");
    }
  };

  // 自定义通道（从管控端 localStorage 读取可见的自定义通道）
  const [visibleCustomChannels, setVisibleCustomChannels] = useState<AdminCustomChannel[]>(() => loadVisibleCustomChannels());

  useEffect(() => {
    const unsub = onCustomChannelsChange(() => {
      setVisibleCustomChannels(loadVisibleCustomChannels());
    });
    return unsub;
  }, []);

  // 内置通道可见性（从管控端 localStorage 读取）
  const [builtinChannelVisibility, setBuiltinChannelVisibility] = useState<Record<string, boolean>>(
    () => loadBuiltinChannelVisibility()
  );

  useEffect(() => {
    const unsub = onBuiltinChannelVisibilityChange(() => {
      setBuiltinChannelVisibility(loadBuiltinChannelVisibility());
    });
    return unsub;
  }, []);
  // ── Channel state ──
  const [selectedChannel, setSelectedChannel] = useState("wework");
  const [channelFields, setChannelFields] = useState<Record<string, string>>({});
  // 密码显示/隐藏状态
  const [visibleSecrets, setVisibleSecrets] = useState<Set<string>>(new Set());
  // 飞书专用：快捷/手动 Tab（默认快捷配置）
  const [feishuConfigMode, setFeishuConfigMode] = useState<"quick" | "manual">("quick");
  // 企业微信专用：快捷/手动 Tab
  const [weworkConfigMode, setWeworkConfigMode] = useState<"quick" | "manual">("quick");
  // 飞书二维码弹窗
  const [showQrModal, setShowQrModal] = useState(false);
  // 飞书弹窗阶段："loading" | "qr" | "configuring" | "done"
  const [feishuModalStage, setFeishuModalStage] = useState<"loading" | "qr" | "configuring" | "done" | "failed">("loading");
  // 飞书配置步骤完成状态
  const [feishuStepsDone, setFeishuStepsDone] = useState<number>(0);
  // 飞书授权次数计数（奇数成功，偶数失败）
  const [feishuToggleCount, setFeishuToggleCount] = useState<number>(0);
  const feishuSteps = [
    "创建应用", "获取应用凭证", "写入配置文件", "开启机器人能力",
    "设置事件模式", "添加消息事件", "配置回调地址", "导入基础权限",
    "发布应用", "导入高级权限", "获取用户信息"
  ];
  // 步骤10（index 9）为高级权限步骤，无法免审批，需橙色标识
  const feishuHighPrivilegeStepIdx = 9;
  // 已接入通道密码显示/隐藏状态
  const [visibleAppliedSecrets, setVisibleAppliedSecrets] = useState<Set<string>>(new Set());
  // 已接入通道
  const [appliedChannels, setAppliedChannels] = useState<AppliedChannel[]>([
    {
      type: "飞书", channelValue: "feishu", status: "running",
      fields: CHANNEL_OPTIONS.find(c => c.value === "feishu")!.fields!,
      fieldValues: { appId: "<your-feishu-app-id>", appSecret: "<your-feishu-app-secret>" },
      feishuConfigMode: "manual",
    },
    {
      type: "QQ", channelValue: "qq", status: "running",
      fields: CHANNEL_OPTIONS.find(c => c.value === "qq")!.fields!,
      fieldValues: { appId: "<your-qq-app-id>", appSecret: "<your-qq-app-secret>" },
    },
  ]);
  // 已接入通道展开状态（手风琴：同一时间只展开一个，用 index | null）
  const [expandedChannelIdx, setExpandedChannelIdx] = useState<number | null>(null);
  // 飞书 pairing code
  const [feishuPairingCode, setFeishuPairingCode] = useState("");
  // 微信二维码弹窗
  const [showWechatQrModal, setShowWechatQrModal] = useState(false);
  // 微信弹窗阶段："checking" | "generating" | "qr"
  const [wechatModalStage, setWechatModalStage] = useState<"checking" | "generating" | "qr">("checking");

  // ── 一键更新状态 ──
  const [showUpdateConfirmDialog, setShowUpdateConfirmDialog] = useState(false);
  const [showUpdateBubble, setShowUpdateBubble] = useState(true);
  const [activeDetailTab, setActiveDetailTab] = useState("basic");

  // ── 平台策略：是否允许用户自助更新（同 key：admin_allow_self_upgrade） ──
  // 默认 true；管理员关闭后按钮置灰，hover 提示联系管理员
  const [allowSelfUpgrade, setAllowSelfUpgrade] = useState<boolean>(() => {
    try {
      const raw = localStorage.getItem("admin_allow_self_upgrade");
      return raw === null ? true : raw === "true";
    } catch {
      return true;
    }
  });
  useEffect(() => {
    const sync = () => {
      try {
        const raw = localStorage.getItem("admin_allow_self_upgrade");
        setAllowSelfUpgrade(raw === null ? true : raw === "true");
      } catch { /* ignore */ }
    };
    window.addEventListener("storage", sync);
    return () => window.removeEventListener("storage", sync);
  }, []);

  // ── 管理员推送的"可更新"徽章 ──
  // 把用户端 agentType（小写 openclaw/hermes/lightclawace）映射到管控端 key
  const imgAgentType = ((): string => {
    const raw = (claw as any).agentType as string | undefined;
    if (!raw) return "OpenClaw";
    const map: Record<string, string> = {
      openclaw: "OpenClaw",
      hermes: "HermesAgent",
      lightclawace: "LightClawACE",
    };
    return map[raw.toLowerCase()] ?? "OpenClaw";
  })();
  const [recommendPush, setRecommendPush] = useState<ActivePush | null>(() => getActivePush(imgAgentType));
  useEffect(() => {
    const refresh = () => setRecommendPush(getActivePush(imgAgentType));
    refresh();
    window.addEventListener("upgrade-push-changed", refresh);
    window.addEventListener("storage", refresh);
    return () => {
      window.removeEventListener("upgrade-push-changed", refresh);
      window.removeEventListener("storage", refresh);
    };
  }, [imgAgentType]);

  // ── 一键更新：目标版本（读取管理员配置的生效镜像版本） ──
  // admin_images_v3 中的 agentType 为 OpenClaw / HermesAgent / LightClawACE
  const getTargetVersion = useCallback((): string | null => {
    try {
      const raw = localStorage.getItem("admin_images_v3");
      if (!raw) return null;
      const imgs: Array<{ agentType: string; agentVersion: string; active: boolean }> = JSON.parse(raw);
      const activeImg = imgs.find(
        (img) => img.agentType === imgAgentType && img.active,
      );
      return activeImg?.agentVersion || null;
    } catch {
      return null;
    }
  }, [imgAgentType]);

  const [targetVersion, setTargetVersion] = useState<string | null>(() => getTargetVersion());

  // 监听 admin_images_v3 变化（管理员切换生效镜像时）
  useEffect(() => {
    const refresh = () => setTargetVersion(getTargetVersion());
    window.addEventListener("storage", refresh);
    return () => window.removeEventListener("storage", refresh);
  }, [getTargetVersion]);

  // 当前 Agent 版本
  const currentVersion: string = (claw as any).agentVersion ?? "2026.4.0";

  // 版本比较：目标版本是否比当前版本更新
  const isTargetNewer = targetVersion ? compareVersion(targetVersion, currentVersion) > 0 : false;

  // ── Memory 状态 ──
  // 当前实例的 Memory 状态：'pro' | 'free' | 'none'
  const [memoryStatus, setMemoryStatus] = useState<'pro' | 'free' | 'none' | 'upgrading'>('none');
  // Pro 版配额是否可用（从管控端获取）
  const [proQuotaAvailable] = useState(true);
  // 记忆数据是否正在加载中（首次进入时可能较慢）
  const [memoryLoading, setMemoryLoading] = useState(false);
  // 标记是否已加载过记忆数据
  const [memoryDataLoaded, setMemoryDataLoaded] = useState(false);

  // 当用户切换到记忆管理 tab 时，首次需要加载数据（Free 版首次可能较慢）
  useEffect(() => {
    if (activeDetailTab === "memory" && !memoryDataLoaded && (memoryStatus === 'free' || memoryStatus === 'pro')) {
      // 首次进入记忆管理，显示加载状态
      setMemoryLoading(true);
      // 模拟后端加载延迟（Free 版用户首次加载较慢，3-5秒）
      const loadTime = memoryStatus === 'free' ? 4000 : 1500;
      const timer = setTimeout(() => {
        setMemoryLoading(false);
        setMemoryDataLoaded(true);
      }, loadTime);
      return () => clearTimeout(timer);
    }
  }, [activeDetailTab, memoryDataLoaded, memoryStatus]);

  // ── Agent 迁移状态 ──
  const [migrationOpen, setMigrationOpen] = useState(false);
  const [migrationStep, setMigrationStep] = useState<"export" | "waitUpload" | "import" | "importing" | "success" | "failed">("export");
  const [migrationCosUrl, setMigrationCosUrl] = useState("");
  const [migrationUploaded, setMigrationUploaded] = useState(false);
  const [migrationChecking, setMigrationChecking] = useState(false);
  const [migrationError, setMigrationError] = useState("");
  const [migrationCommandReady, setMigrationCommandReady] = useState(false);

  // 导入步骤流转（替代进度条）
  type ImportStepStatus = "pending" | "running" | "done" | "failed";
  type ImportStepItem = { label: string; status: ImportStepStatus; error?: string };
  const [importSteps, setImportSteps] = useState<ImportStepItem[]>([
    { label: "下载数据包", status: "pending" },
    { label: "备份当前配置", status: "pending" },
    { label: "解压并覆盖", status: "pending" },
    { label: "重启 Gateway", status: "pending" },
    { label: "验证生效", status: "pending" },
  ]);

  // 导入后验证结果
  type VerifyItem = { label: string; cmd: string; passed: boolean; detail?: string };
  const [verifyResults, setVerifyResults] = useState<VerifyItem[]>([]);

  const [migrationCheckFailed, setMigrationCheckFailed] = useState(false);
  const [migrationCheckCount, setMigrationCheckCount] = useState(0);

  const migrationBatchId = `${clawData?.instanceId || "unknown"}-${Date.now()}`;
  const migrationCosBucket = "clawpro-migrate-1302061491";
  const migrationCosKey = `single/${clawData?.instanceId || "unknown"}-${Math.random().toString(36).substring(2, 8)}.tgz`;
  const migrationPresignedUrl = `https://${migrationCosBucket}.cos.ap-guangzhou.myqcloud.com/${migrationCosKey}?q-sign-algorithm=sha1&q-ak=AKID****&q-sign-time=****&q-signature=****`;

  const migrationExportCommand = `# 在源端 Agent 终端执行以下命令
agent gateway stop
tar -czf /tmp/openclaw-export.tgz -C $HOME .agent
curl -X PUT --upload-file /tmp/openclaw-export.tgz \\
  "${migrationPresignedUrl}"
rm -f /tmp/openclaw-export.tgz
agent gateway start
echo "✅ 导出完成，数据已上传到 COS"`;

  const handleCheckUpload = () => {
    setMigrationChecking(true);
    setMigrationCheckFailed(false);
    setMigrationCheckCount((c) => c + 1);
    setTimeout(() => {
      // 模拟：第一次检测有概率检测不到（用户可能还没上传完）
      const detected = migrationCheckCount >= 1 || Math.random() < 0.6;
      setMigrationChecking(false);
      if (detected) {
        setMigrationUploaded(true);
        setMigrationCheckFailed(false);
        setMigrationStep("import");
        toast.success("检测到已上传的数据包");
      } else {
        setMigrationCheckFailed(true);
        toast.error("未检测到数据包");
      }
    }, 1500);
  };

  const handleStartMigration = () => {
    setMigrationStep("importing");
    const steps: ImportStepItem[] = [
      { label: "下载数据包", status: "pending" },
      { label: "备份当前配置", status: "pending" },
      { label: "解压并覆盖", status: "pending" },
      { label: "重启 Gateway", status: "pending" },
      { label: "验证生效", status: "pending" },
    ];
    setImportSteps(steps);
    setVerifyResults([]);

    const delays = [1200, 1000, 1500, 2000, 1800];
    let current = 0;

    const failStep = (idx: number, reason: string) => {
      setImportSteps((prev) => prev.map((s, i) => i === idx ? { ...s, status: "failed", error: reason } : s));
      setMigrationStep("failed");
      setMigrationError(reason + "，已自动回滚");
    };

    const runStep = () => {
      if (current >= steps.length) return;
      setImportSteps((prev) => prev.map((s, i) => i === current ? { ...s, status: "running" } : s));

      setTimeout(() => {
        const stepIdx = current;
        // 模拟：后端接口返回失败（如解压格式异常）
        const fail = stepIdx === 2 && Math.random() < 0.08;

        if (fail) {
          failStep(stepIdx, "解压失败：数据包格式异常");
          return;
        }

        setImportSteps((prev) => prev.map((s, i) => i === stepIdx ? { ...s, status: "done" } : s));
        current++;

        if (current < steps.length) {
          runStep();
        } else {
          const results: VerifyItem[] = [
            { label: "Agent 进程状态", cmd: "agent gateway status", passed: true, detail: "running" },
            { label: "配置完整性", cmd: "agent doctor check", passed: true, detail: "所有检查项通过" },
            { label: "通道连通性", cmd: "agent gateway ping", passed: Math.random() < 0.85, detail: Math.random() < 0.85 ? "ping 成功" : "IM 通道需重新登录" },
          ];
          setVerifyResults(results);

          const allPassed = results.every((r) => r.passed);
          const criticalFailed = !results[0].passed;

          if (criticalFailed) {
            setMigrationStep("failed");
            setMigrationError("Agent 进程未启动，已触发自动回滚");
          } else {
            setMigrationStep("success");
            toast.success(allPassed ? "迁移成功，已验证生效" : "迁移完成，部分项需手动处理");
          }
        }
      }, delays[current]);
    };

    runStep();
  };

  const resetMigration = () => {
    setMigrationStep("export");
    setMigrationCosUrl("");
    setMigrationUploaded(false);
    setMigrationChecking(false);
    setMigrationCheckFailed(false);
    setMigrationCheckCount(0);
    setMigrationError("");
    setMigrationCommandReady(false);
    setImportSteps([
      { label: "下载数据包", status: "pending" },
      { label: "备份当前配置", status: "pending" },
      { label: "解压并覆盖", status: "pending" },
      { label: "重启 Gateway", status: "pending" },
      { label: "验证生效", status: "pending" },
    ]);
    setVerifyResults([]);
    setTimeout(() => setMigrationCommandReady(true), 1800);
  };

  // 导入失败后重试：不回到导出步骤，直接重新执行导入
  const retryImport = () => {
    setMigrationError("");
    setVerifyResults([]);
    handleStartMigration();
  };

  // 弹窗内容变化时自动滚到底部
  const migrationDialogRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (migrationOpen && migrationDialogRef.current) {
      setTimeout(() => {
        migrationDialogRef.current?.scrollTo({ top: migrationDialogRef.current.scrollHeight, behavior: "smooth" });
      }, 100);
    }
  }, [migrationStep, migrationChecking, migrationCheckFailed, migrationUploaded, migrationCommandReady, migrationOpen, importSteps]);

  const [showUpdateProgressDialog, setShowUpdateProgressDialog] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);
  const [updateStepsDone, setUpdateStepsDone] = useState<number>(0);
  const updateSteps = [
    "环境准备",
    "Agent 安装",
    "Doctor 修复",
    "Gateway 安装",
    "Clawhub 安装",
    "插件安装",
    "Skills 安装",
    "安装收尾",
  ];
  const handleStartUpdate = () => {
    setShowUpdateConfirmDialog(false);
    setIsUpdating(true);
    setUpdateStepsDone(0);
    setShowUpdateProgressDialog(true);
    // 随机间隔逐步完成 8 步
    let done = 0;
    const runNext = () => {
      if (done >= updateSteps.length) {
        setIsUpdating(false);
        setShowUpdateProgressDialog(false);
        // 更新完成后刷新版本号为目标版本
        if (targetVersion) {
          setClawData((prev) => ({ ...prev, agentVersion: targetVersion } as typeof prev));
        }
        toast.success("Agent 已更新至最新版本");
        return;
      }
      const delay = 800 + Math.random() * 2200; // 0.8s ~ 3s 随机
      setTimeout(() => {
        done += 1;
        setUpdateStepsDone(done);
        runNext();
      }, delay);
    };
    runNext();
  };

  // ── WebUI 状态 ──
  const [showWebUIProgressDialog, setShowWebUIProgressDialog] = useState(false);
  const [showWebUIResultDialog, setShowWebUIResultDialog] = useState(false);
  const [webUIStep, setWebUIStep] = useState<0 | 1 | 2>(0); // 0=未开始, 1=放通端口完成, 2=生成链接完成
  // 失败状态："none" | "port" | "link"
  const [webUIFailedStep, setWebUIFailedStep] = useState<"none" | "port" | "link">("none");
  // 打开次数计数（奇数成功，偶数失败）
  const [webUIOpenCount, setWebUIOpenCount] = useState(0);
  const webUIUrl = "http://43.139.137.45:38341/knmnz8?token=<your-webui-token>";
  const webUIToken = "<your-webui-token>";

  // ── Hermes 专属开启面板弹窗状态 ──
  const [showHermesPanelDialog, setShowHermesPanelDialog] = useState(false);
  const [hermesPanelStep, setHermesPanelStep] = useState<0 | 1 | 2>(0); // 0=等待中, 1=加载中, 2=完成
  const hermesPanelUrl = "http://43.139.137.45:38341/knmnz8?token=<your-webui-token>";

  const runHermesPanelFlow = () => {
    setHermesPanelStep(0);
    setShowHermesPanelDialog(true);
    // 1.5秒后进入加载中
    setTimeout(() => {
      setHermesPanelStep(1);
      // 再3秒后完成，自动跳转
      setTimeout(() => {
        setHermesPanelStep(2);
        setTimeout(() => {
          setShowHermesPanelDialog(false);
          window.open(hermesPanelUrl, "_blank");
        }, 500);
      }, 3000);
    }, 1500);
  };

  const runWebUIFlow = (isFail: boolean) => {
    setWebUIStep(0);
    setWebUIFailedStep("none");
    if (isFail) {
      // 失败流程：1.5秒后放通端口失败
      setTimeout(() => {
        setWebUIFailedStep("port");
      }, 1500);
    } else {
      // 成功流程：1.5秒后放通端口完成，再4秒后生成链接完成
      setTimeout(() => {
        setWebUIStep(1);
        setTimeout(() => {
          setWebUIStep(2);
        }, 4000);
      }, 1500);
    }
  };

  // 从 localStorage 读取管理员是否开启了用户端访问权限
  const allowPanelAccess = localStorage.getItem("admin_allow_panel_access") === "true";

  const handleOpenWebUI = () => {
    if (!allowPanelAccess) {
      toast.error("管理员未开启访问权限");
      return;
    }
    // Hermes 类型使用专属弹窗，加载完自动跳转
    if ((claw as any).agentType === "hermes") {
      runHermesPanelFlow();
      return;
    }
    const newCount = webUIOpenCount + 1;
    setWebUIOpenCount(newCount);
    setShowWebUIProgressDialog(true);
    runWebUIFlow(newCount % 2 === 0); // 偶数次失败
  };

  const handleWebUIProgressConfirm = () => {
    setShowWebUIProgressDialog(false);
    setShowWebUIResultDialog(true);
  };

  const handleWebUIRetry = () => {
    const newCount = webUIOpenCount + 1;
    setWebUIOpenCount(newCount);
    runWebUIFlow(newCount % 2 === 0);
  };

  const handleFeishuPairing = () => {
    if (!feishuPairingCode.trim()) {
      toast.error("请输入 pairing code");
      return;
    }
    toast.success("匹配成功");
    setFeishuPairingCode("");
  };

  const toggleExpandChannel = (idx: number) => {
    setExpandedChannelIdx(prev => prev === idx ? null : idx);
  };

  const toggleSecretVisibility = (key: string) => {
    setVisibleSecrets(prev => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  const toggleAppliedSecretVisibility = (channelIdx: number, fieldKey: string) => {
    const uniqueKey = `${channelIdx}-${fieldKey}`;
    setVisibleAppliedSecrets(prev => {
      const next = new Set(prev);
      if (next.has(uniqueKey)) {
        next.delete(uniqueKey);
      } else {
        next.add(uniqueKey);
      }
      return next;
    });
  };

  // ── Skills state ──
  const [skillSearch, setSkillSearch] = useState("");
  const [skillInstallConfirm, setSkillInstallConfirm] = useState<{ open: boolean; skillName: string }>({
    open: false,
    skillName: "",
  });
  const [installedSkills, setInstalledSkills] = useState<string[]>([
    "tavily-search 1.0.0",
    "summarize 1.0.0",
    "agent-browser 0.2.0",
    "find-skills 0.1.0",
    "github 1.0.0",
    "obsidian 1.0.0",
    "notion 1.0.0",
    "weather 1.0.0",
    "tencentcloud-lighthouse-skill 1.0.0",
    "tencent-docs 1.0.3",
    "slack 2.1.0",
    "jira 1.5.2",
    "confluence 1.3.0",
    "gitlab 1.2.1",
    "linear 0.8.0",
    "figma-export 1.0.0",
    "google-calendar 2.0.1",
    "airtable 1.1.0",
    "zapier-webhook 0.5.0",
    "stripe-billing 1.0.0",
    "sendgrid-email 1.2.0",
    "twilio-sms 0.9.0",
    "aws-s3 2.3.0",
    "openai-dalle 1.0.0",
    "huggingface-inference 1.0.0",
    "elasticsearch 2.0.0",
    "redis-cache 1.1.0",
    "mongodb-query 1.4.0",
    "postgres-sql 2.2.0",
    "docker-exec 0.8.0",
    "kubernetes-deploy 1.0.0",
    "terraform-plan 0.5.0",
    "ansible-run 1.2.0",
    "prometheus-alert 1.0.0",
    "grafana-dashboard 0.9.0",
    "datadog-monitor 1.1.0",
    "pagerduty-incident 1.0.0",
    "zoom-meeting 2.0.0",
    "teams-message 1.3.0",
    "discord-bot 0.7.0",
    "telegram-send 1.0.0",
    "wechat-work 2.1.0",
    "dingtalk-notify 1.5.0",
  ]);

  // 技能最新版本表（mock）：仅当已安装版本低于此处版本时，才展示"更新"入口
  const SKILL_LATEST_VERSIONS: Record<string, string> = {
    "tavily-search": "1.2.0",
    "agent-browser": "0.3.1",
    "github": "1.1.0",
    "notion": "1.1.0",
    "slack": "2.2.0",
    "figma-export": "1.0.2",
    "aws-s3": "2.4.0",
    "postgres-sql": "2.3.0",
    "linear": "0.9.0",
  };

  // 解析 "name version" 格式的已安装技能条目
  const parseSkillEntry = (entry: string) => {
    const idx = entry.lastIndexOf(" ");
    if (idx === -1) return { name: entry, version: "" };
    return { name: entry.slice(0, idx), version: entry.slice(idx + 1) };
  };

  // 计算某条已安装技能的更新信息
  const getSkillUpdateInfo = (entry: string) => {
    const { name, version } = parseSkillEntry(entry);
    const latestVersion = SKILL_LATEST_VERSIONS[name];
    const hasUpdate = !!latestVersion && !!version && compareVersion(latestVersion, version) > 0;
    return { name, version, latestVersion, hasUpdate };
  };

  // 卸载技能二次确认
  const [skillUninstallConfirm, setSkillUninstallConfirm] = useState<{ open: boolean; entry: string }>({
    open: false,
    entry: "",
  });

  // 更新技能到最新版本
  const handleUpdateSkill = (entry: string) => {
    const { name, version, latestVersion } = getSkillUpdateInfo(entry);
    if (!latestVersion) return;
    const nextEntry = `${name} ${latestVersion}`;
    setInstalledSkills((prev) => prev.map((s) => (s === entry ? nextEntry : s)));
    toast.success(`${name} 已更新至 ${latestVersion}`, {
      description: version ? `原版本 ${version}` : undefined,
    });
  };

  // 确认卸载技能
  const handleConfirmUninstallSkill = () => {
    const entry = skillUninstallConfirm.entry;
    const { name } = parseSkillEntry(entry);
    setInstalledSkills((prev) => prev.filter((s) => s !== entry));
    setSkillUninstallConfirm({ open: false, entry: "" });
    toast.success(`${name} 已卸载`);
  };

  // ── Handlers ──

  const handleApplyModel = () => {
    let newEntry: AppliedModel;
    if (isCustomProvider) {
      const customName = customInputMode === "json"
        ? (() => { try { const parsed = JSON.parse(customJson); return parsed?.model?.name || ""; } catch { return ""; } })()
        : customForm.model_name;
      newEntry = { id: modelIdCounter, providerLabel: "自定义模型", versionLabel: "", primary: false, isCustom: true, customName: customName || "", addedAt: Date.now(), multimodal: customMultimodal };
    } else if (isPublicSelfProvider) {
      // 自行配置的公开模型：校验 版本 + API Key + 限额
      const provider = MODEL_PROVIDERS.find(p => p.value === selectedProvider);
      const version = currentVersions.find(v => v.value === selectedModel);
      if (!provider || !version) { toast.error("请选择模型版本"); return; }
      if (!publicForm.apiKey) { toast.error("请填写 API Key"); return; }
      if (!publicForm.dailyLimit || Number(publicForm.dailyLimit) <= 0) { toast.error("请填写有效的每日 Tokens 上限"); return; }
      newEntry = { id: modelIdCounter, providerLabel: provider.label, versionLabel: version.label, primary: false, isCustom: false, customName: "", addedAt: Date.now() };
      // 重置公开模型表单
      setPublicForm({ apiKey: "", dailyLimit: "100000" });
      setAdvancedOpen(false);
      setAdvancedConfig({ contextWindow: "", maxTokens: "", headers: [{ key: "", value: "" }] });
    } else {
      const provider = MODEL_PROVIDERS.find(p => p.value === selectedProvider);
      const version = currentVersions.find(v => v.value === selectedModel);
      if (!provider || !version) return;
      newEntry = { id: modelIdCounter, providerLabel: provider.label, versionLabel: version.label, primary: false, isCustom: false, customName: "", addedAt: Date.now() };
    }
    setAppliedModels(prev => {
      // 当前无主模型时（列表为空或全部为备选），新模型直接成为主模型
      const hasPrimary = prev.some(m => m.primary);
      if (!hasPrimary) return [...prev, { ...newEntry, primary: true }];
      return [...prev, newEntry];
    });
    setModelIdCounter(c => c + 1);
    const hasPrimary = appliedModels.some(m => m.primary);
    toast.success(hasPrimary ? "备用模型已添加" : "已设为主模型");
  };

  const handleAddChannel = () => {
    // 先在全部通道选项（包括自定义）中查找
    const ch = allChannelOptions.find((c) => c.value === selectedChannel);
    if (!ch) return;

    // 管控端自定义通道处理
    if (ch.adminCustomMode) {
      const newEntry: AppliedChannel = {
        type: ch.label,
        channelValue: ch.value,
        status: "running",
        fields: ch.fields || [],
        fieldValues: { ...channelFields },
      };
      setAppliedChannels([...appliedChannels, newEntry]);
      setChannelFields({});
      toast.success(`${ch.label} 已添加并应用`);
      return;
    }

    // 飞书快捷配置：点击"前往授权"弹出二维码
    if (ch.feishuMode && feishuConfigMode === "quick") {
      const newCount = feishuToggleCount + 1;
      setFeishuToggleCount(newCount);
      const willSucceed = newCount % 2 === 1; // 奇数成功，偶数失败
      setFeishuModalStage("loading");
      setFeishuStepsDone(0);
      setShowQrModal(true);
      // 5秒后显示二维码
      setTimeout(() => setFeishuModalStage("qr"), 5000);
      // 再5秒后自动进入配置阶段
      setTimeout(() => {
        setFeishuModalStage("configuring");
        // 每步约0.8秒逐步完成
        for (let i = 1; i <= 10; i++) {
          setTimeout(() => {
            setFeishuStepsDone(i);
            if (i === 10) {
              setTimeout(() => setFeishuModalStage(willSucceed ? "done" : "failed"), 600);
            }
          }, i * 800);
        }
      }, 10000);
      return;
    }

    // 企业微信快捷配置：点击"前往授权"弹出提示
    if (ch.weworkMode && weworkConfigMode === "quick") {
      toast.info("即将跳转至企业微信授权页面，此功能即将开放");
      // 快捷配置添加一个企微机器人占位符
      const newEntry: AppliedChannel = {
        type: "企微机器人",
        channelValue: "wework",
        status: "running",
        fields: ch.fields || [],
        fieldValues: { botId: "auto-authorized", secret: "<your-bot-secret>" },
        weworkConfigMode: "quick",
      };
      setAppliedChannels([...appliedChannels, newEntry]);
      toast.success("企微机器人已添加");
      return;
    }

    // 微信：点击"前往授权"弹出二维码（带 loading 流程）
    if (ch.wechatMode) {
      setWechatModalStage("checking");
      setShowWechatQrModal(true);
      // 2秒后切换到"正在生成二维码"
      setTimeout(() => setWechatModalStage("generating"), 2000);
      // 再2秒后显示二维码
      setTimeout(() => {
        setWechatModalStage("qr");
        // 二维码出现后5秒自动关闭并添加通道
        setTimeout(() => {
          setShowWechatQrModal(false);
          setAppliedChannels(prev => {
            const existingIdx = prev.findIndex(c => c.channelValue === "wechat");
            const newEntry: AppliedChannel = {
              type: "微信 ClawBot",
              channelValue: "wechat",
              status: "running",
              fields: [],
              fieldValues: {},
            };
            if (existingIdx >= 0) {
              const next = [...prev];
              next[existingIdx] = newEntry;
              return next;
            }
            return [...prev, newEntry];
          });
          toast.success("微信 ClawBot 已添加");
        }, 5000);
      }, 4000);
      return;
    }

    // 企业微信手动配置：显示为"企微机器人"
    const channelType = ch.weworkMode ? "企微机器人" : ch.label;
    const newEntry: AppliedChannel = {
      type: channelType,
      channelValue: ch.value,
      status: "running",
      fields: ch.fields || [],
      fieldValues: { ...channelFields },
      feishuConfigMode: ch.feishuMode ? feishuConfigMode : undefined,
      weworkConfigMode: ch.weworkMode ? weworkConfigMode : undefined,
    };
    setAppliedChannels([...appliedChannels, newEntry]);
    setChannelFields({});
    toast.success(`${channelType} 已添加并应用`);
  };



  const filteredSkills = installedSkills.filter((s) =>
    s.toLowerCase().includes(skillSearch.toLowerCase())
  );

  // ── Pending skills state ──
  type PendingSkillStatus = "pending" | "installing" | "failed";
  type PendingSkill = { id: string; name: string; status: PendingSkillStatus };
  // ps-3 和 ps-7 模拟安装失败
  const MOCK_FAIL_IDS = new Set(["ps-3", "ps-7"]);

  const [pendingSkills, setPendingSkills] = useState<PendingSkill[]>([
    { id: "ps-1", name: "code-interpreter 1.2.0", status: "pending" },
    { id: "ps-2", name: "image-recognition 0.9.1", status: "pending" },
    { id: "ps-3", name: "data-analysis 2.0.0", status: "pending" },
    { id: "ps-4", name: "text-to-speech 1.0.0", status: "pending" },
    { id: "ps-5", name: "pdf-parser 1.1.0", status: "pending" },
    { id: "ps-6", name: "excel-reader 2.0.0", status: "pending" },
    { id: "ps-7", name: "video-transcribe 0.7.0", status: "pending" },
    { id: "ps-8", name: "sentiment-analysis 1.0.0", status: "pending" },
    { id: "ps-9", name: "ocr-scanner 1.3.0", status: "pending" },
    { id: "ps-10", name: "sql-query 2.1.0", status: "pending" },
    { id: "ps-11", name: "web-scraper 0.6.0", status: "pending" },
    { id: "ps-12", name: "chart-generator 1.0.0", status: "pending" },
  ]);

  // 并行安装：页面加载后（及后续新加入的）所有 pending 技能同时变为 installing
  useEffect(() => {
    const pendingList = pendingSkills.filter(s => s.status === "pending");
    if (pendingList.length === 0) return;
    // 所有 pending 同时变为 installing
    setPendingSkills(prev =>
      prev.map(s => s.status === "pending" ? { ...s, status: "installing" as PendingSkillStatus } : s)
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pendingSkills.filter(s => s.status === "pending").map(s => s.id).join()]);

  // 监听 installing 技能，3秒后一次性批量更新所有结果
  useEffect(() => {
    const installingSkills = pendingSkills.filter(s => s.status === "installing");
    if (installingSkills.length === 0) return;
    const timer = setTimeout(() => {
      const successSkills = installingSkills.filter(s => !MOCK_FAIL_IDS.has(s.id));
      const failedIds = new Set(installingSkills.filter(s => MOCK_FAIL_IDS.has(s.id)).map(s => s.id));
      // 一次性更新 pendingSkills：删除成功的，失败的标记 failed
      setPendingSkills(prev =>
        prev
          .filter(s => !successSkills.some(ss => ss.id === s.id))
          .map(s => failedIds.has(s.id) ? { ...s, status: "failed" as PendingSkillStatus } : s)
      );
      // 一次性批量添加到已安装列表
      if (successSkills.length > 0) {
        setInstalledSkills(prev => [...successSkills.map(s => s.name), ...prev]);
      }
    }, 3000);
    return () => clearTimeout(timer);
  }, [pendingSkills.filter(s => s.status === "installing").map(s => s.id).join()]);

  // 全部重试：所有失败技能同时重新安装
  const handleRetryAllFailed = () => {
    const failedIds = pendingSkills.filter(s => s.status === "failed").map(s => s.id);
    failedIds.forEach(id => MOCK_FAIL_IDS.delete(id));
    setPendingSkills(prev =>
      prev.map(s => s.status === "failed" ? { ...s, status: "installing" as PendingSkillStatus } : s)
    );
  };

  // 全部删除：移除所有失败技能
  const handleDeleteAllFailed = () => {
    setPendingSkills(prev => prev.filter(s => s.status !== "failed"));
  };

  // ── 头部「项目」编辑 ──────────────────────────────────────
  // 项目池来自共享 groupStore（管控端「项目资产管理」创建的项目实时可见，单层级）
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
  const [projectEditOpen, setProjectEditOpen] = useState(false);
  const [projectDraftIds, setProjectDraftIds] = useState<string[]>([]);
  const toggleProjectDraft = (id: string) =>
    setProjectDraftIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));

  const openProjectEdit = useCallback(() => {
    setProjectDraftIds(claw.projectIds ?? []);
    setProjectEditOpen(true);
  }, [claw.projectIds]);

  /** 查询某项目在「项目资产管理」中关联的技能（企业技能 + 公共技能），返回 "名称 版本" 列表 */
  const getProjectSkillNames = useCallback((projectId: string): string[] => {
    const cfg = projectAssetStore.getConfig(projectId);
    const refs = [
      ...cfg.categories.enterpriseSkill.items.map((i) => ({ refId: i.refId, version: i.versionAtBind, cat: "enterpriseSkill" as const })),
      ...cfg.categories.publicSkill.items.map((i) => ({ refId: i.refId, version: i.versionAtBind, cat: "publicSkill" as const })),
    ];
    return refs.map((r) => {
      const d = getAssetItemDisplay(r.cat, r.refId);
      return `${d.name} ${r.version}`;
    });
  }, []);

  const saveProjectEdit = useCallback(() => {
    const prevIds = claw.projectIds ?? [];
    const nextIds = projectDraftIds;
    const nextNames = projectPool.filter((p) => nextIds.includes(p.id)).map((p) => p.name);
    // 新增关联的项目（取消关联不移除已安装技能，故只处理新增）
    const addedIds = nextIds.filter((id) => !prevIds.includes(id));

    // 持久化项目关联到共享 store
    try {
      const latest = loadClawList();
      const updated = latest.map((item) =>
        item.id === claw.id ? { ...item, projectIds: nextIds, projectNames: nextNames } : item,
      );
      saveClawList(updated);
      notifyClawListChange();
    } catch {
      // ignore（演示环境）
    }
    setClawData((prev) => ({ ...prev, projectIds: nextIds, projectNames: nextNames }));

    // 收集新增项目携带的技能，注入待安装队列（复用现有 pending→installing→installed 流程）
    let newSkillCount = 0;
    if (addedIds.length > 0) {
      const skillNames = Array.from(new Set(addedIds.flatMap((pid) => getProjectSkillNames(pid))));
      const toAdd = skillNames.filter(
        (name) =>
          !installedSkills.includes(name) &&
          !pendingSkills.some((p) => p.name === name),
      );
      newSkillCount = toAdd.length;
      if (toAdd.length > 0) {
        setPendingSkills((cur) => [
          ...cur,
          ...toAdd
            .filter((name) => !cur.some((c) => c.name === name))
            .map((name) => ({ id: `proj-skill-${name}`, name, status: "pending" as PendingSkillStatus })),
        ]);
      }
    }

    setProjectEditOpen(false);
    toast.success(newSkillCount > 0 ? `已更新项目关联，正在安装 ${newSkillCount} 个新增技能` : "已更新项目关联");
  }, [claw.id, claw.projectIds, projectDraftIds, projectPool, getProjectSkillNames, installedSkills, pendingSkills]);

  // 内置通道按管控端开关过滤（没有 builtinId 的项目为全局内置，始终显示）
  const visibleBuiltinChannels = CHANNEL_OPTIONS.filter((ch) => {
    if (!ch.builtinId) return true;
    return builtinChannelVisibility[ch.builtinId] !== false;
  });

  // 合并内置通道 + 可见的自定义通道（动态构建 ChannelConfig）
  const allChannelOptions: ChannelConfig[] = [
    ...CHANNEL_OPTIONS,
    ...visibleCustomChannels.map((cc) => ({
      value: `admin_custom_${cc.id}`,
      label: cc.name,
      descText: `企业自定义通道（Channel ID: ${cc.channelId}）`,
      detailUrl: "#",
      adminCustomMode: true as const,
      adminCustomId: cc.id,
      fields: cc.credentialFields.map((f) => ({
        key: f.key || f.id, // 使用管控端配置的 key，写入配置文件
        label: f.label,     // 用户看到的标签
        secret: true,       // 凭证字段默认加密显示
      })),
    } as ChannelConfig & { adminCustomMode: true; adminCustomId: string })),
  ];

  const currentChannelConfig = allChannelOptions.find((c) => c.value === selectedChannel);

  // ─── 渲染通道配置输入区 ───────────────────────────────────────────────────────

  const renderChannelInputs = () => {
    if (!currentChannelConfig) return null;

    // 企业微信快捷/手动配置
    if (currentChannelConfig.weworkMode) {
      return (
        <div className="space-y-3">
          {/* 快捷/手动 Tab（快捷默认选中） */}
          <TenantSegmentGroup className="w-full">
            <TenantSegmentOption
              className="flex-1"
              active={weworkConfigMode === "quick"}
              onClick={() => setWeworkConfigMode("quick")}
            >
              快捷配置
            </TenantSegmentOption>
            <TenantSegmentOption
              className="flex-1"
              active={weworkConfigMode === "manual"}
              onClick={() => setWeworkConfigMode("manual")}
            >
              手动配置
            </TenantSegmentOption>
          </TenantSegmentGroup>

          {weworkConfigMode === "manual" && (
            <div className="space-y-2">
              {currentChannelConfig.fields!.map((field) => (
                <div key={field.key} className="relative">
                  <Input
                    tenant
                    type={field.secret && !visibleSecrets.has(field.key) ? "password" : "text"}
                    placeholder={field.label}
                    value={channelFields[field.key] || ""}
                    onChange={(e) => setChannelFields({ ...channelFields, [field.key]: e.target.value })}
                    className="bg-gray-50 border-gray-200 pr-10"
                  />
                  {field.secret && (
                    <button
                      onClick={() => toggleSecretVisibility(field.key)}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors cursor-pointer"
                      type="button"
                      title={visibleSecrets.has(field.key) ? "隐藏" : "显示"}
                    >
                      {visibleSecrets.has(field.key) ? (
                        <Eye className="w-4 h-4" />
                      ) : (
                        <EyeOff className="w-4 h-4" />
                      )}
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      );
    }

    // 微信：无需额外输入，直接显示"前往授权"按钮（由外部按钮处理）
    if (currentChannelConfig.wechatMode) {
      return null;
    }

    if (currentChannelConfig.feishuMode) {
      return (
        <div className="space-y-3">
          {/* 快捷配置在左，手动配置在右 */}
          <TenantSegmentGroup className="w-full">
            <TenantSegmentOption
              className="flex-1"
              active={feishuConfigMode === "quick"}
              onClick={() => setFeishuConfigMode("quick")}
            >
              快捷配置
            </TenantSegmentOption>
            <TenantSegmentOption
              className="flex-1"
              active={feishuConfigMode === "manual"}
              onClick={() => setFeishuConfigMode("manual")}
            >
              手动配置
            </TenantSegmentOption>
          </TenantSegmentGroup>

          {feishuConfigMode === "manual" && (
            <div className="space-y-2">
              {currentChannelConfig.fields!.map((field) => (
                <div key={field.key} className="relative">
                  <Input
                    tenant
                    type={field.secret && !visibleSecrets.has(field.key) ? "password" : "text"}
                    placeholder={field.label}
                    value={channelFields[field.key] || ""}
                    onChange={(e) => setChannelFields({ ...channelFields, [field.key]: e.target.value })}
                    className="bg-gray-50 border-gray-200 pr-10"
                  />
                  {field.secret && (
                    <button
                      onClick={() => toggleSecretVisibility(field.key)}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors cursor-pointer"
                      type="button"
                      title={visibleSecrets.has(field.key) ? "隐藏" : "显示"}
                    >
                      {visibleSecrets.has(field.key) ? (
                        <Eye className="w-4 h-4" />
                      ) : (
                        <EyeOff className="w-4 h-4" />
                      )}
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      );
    }

    // 管控端自定义通道：渲染管理员定义的凭证字段
    if (currentChannelConfig.adminCustomMode) {
      if (!currentChannelConfig.fields || currentChannelConfig.fields.length === 0) {
        return (
          <div className="rounded-[12px] bg-gray-50 border border-gray-200 px-4 py-3">
            <p className="text-xs text-gray-400">该通道无需额外凭证信息</p>
          </div>
        );
      }
      return (
        <div className="space-y-2">
          {currentChannelConfig.fields.map((field) => (
            <div key={field.key} className="relative">
              <Input
                tenant
                type={field.secret && !visibleSecrets.has(field.key) ? "password" : "text"}
                placeholder={field.label}
                value={channelFields[field.key] || ""}
                onChange={(e) => setChannelFields({ ...channelFields, [field.key]: e.target.value })}
                className="bg-gray-50 border-gray-200 pr-10"
              />
              {field.secret && (
                <button
                  onClick={() => toggleSecretVisibility(field.key)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors cursor-pointer"
                  type="button"
                  title={visibleSecrets.has(field.key) ? "隐藏" : "显示"}
                >
                  {visibleSecrets.has(field.key) ? (
                    <Eye className="w-4 h-4" />
                  ) : (
                    <EyeOff className="w-4 h-4" />
                  )}
                </button>
              )}
            </div>
          ))}
        </div>
      );
    }

    // 普通通道
    return (
      <div className="space-y-2">
        {currentChannelConfig.fields?.map((field) => (
          <div key={field.key} className="relative">
            <Input
              tenant
              type={field.secret && !visibleSecrets.has(field.key) ? "password" : "text"}
              placeholder={field.label}
              value={channelFields[field.key] || ""}
              onChange={(e) => setChannelFields({ ...channelFields, [field.key]: e.target.value })}
              className="bg-gray-50 border-gray-200 pr-10"
            />
            {field.secret && (
              <button
                onClick={() => toggleSecretVisibility(field.key)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors cursor-pointer"
                type="button"
                title={visibleSecrets.has(field.key) ? "隐藏" : "显示"}
              >
                {visibleSecrets.has(field.key) ? (
                  <Eye className="w-4 h-4" />
                ) : (
                  <EyeOff className="w-4 h-4" />
                )}
              </button>
            )}
          </div>
        ))}
      </div>
    );
  };

  // ─── 渲染已接入通道的展开配置项 ─────────────────────────────────────────────

  const renderAppliedChannelDetail = (chIdx: number, ch: AppliedChannel) => {
    // 判断是否是管控端自定义通道（value 以 admin_custom_ 开头）
    const isAdminCustom = ch.channelValue.startsWith("admin_custom_");

    return (
      <div className="mx-2 mb-2 space-y-2">
        {isAdminCustom ? (
          /* 管控端自定义通道：展示字段 key，内容加密 */
          <div className="rounded-[12px] bg-white border border-gray-200 px-4 py-3 space-y-2">
            {ch.fields.length === 0 ? (
              <p className="text-xs text-gray-400">无凭证字段</p>
            ) : (
              ch.fields.map((field) => {
                const val = ch.fieldValues[field.key] || "";
                const displayVal = maskSecret(val);
                return (
                  <div key={field.key} className="flex items-center gap-1 text-sm">
                    <span className="text-gray-500 font-mono shrink-0">{field.key}：</span>
                    <span className="text-gray-800 font-mono break-all flex-1">{displayVal || "—"}</span>
                  </div>
                );
              })
            )}
          </div>
        ) : (
          <div className="rounded-[12px] bg-white border border-gray-200 px-4 py-3 space-y-2">
            {ch.fields.map((field) => {
              const val = ch.fieldValues[field.key] || "";
              const uniqueKey = `${chIdx}-${field.key}`;
              const isVisible = visibleAppliedSecrets.has(uniqueKey);
              const displayVal = field.secret && !isVisible ? maskSecret(val) : val;
              const displayKey = field.key;
              return (
                <div key={field.key} className="flex items-center gap-1 text-sm">
                  <span className="text-gray-500 shrink-0">{displayKey}：</span>
                  <span className="text-gray-800 font-mono break-all flex-1">{displayVal || "—"}</span>
                </div>
              );
            })}
          </div>
        )}
         {/* 子框2：飞书 pairing code */}
        {ch.channelValue === "feishu" && (
          <div className="rounded-[12px] bg-white border border-gray-200 px-4 py-3 flex items-center gap-2">
            <Input
              tenant
              placeholder="（如需）请输入 pairing code"
              value={feishuPairingCode}
              onChange={(e) => setFeishuPairingCode(e.target.value)}
              className="bg-gray-50 border-gray-200 text-sm h-8"
              onKeyDown={(e) => e.key === "Enter" && handleFeishuPairing()}
            />
            <Button
              variant="tenant-outline"
              size="claw-sm"
              className="shrink-0 text-sm"
              onClick={handleFeishuPairing}
            >
              匹配
            </Button>
          </div>
        )}
      </div>
    );
  };

  return (
    <TooltipProvider delayDuration={200}>
    <TenantLayout>
      {isConfiguring && (
        <div className="fixed inset-0 bg-white/80 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="flex flex-col items-center gap-4">
            <Loader2 className="w-8 h-8 text-blue-600 animate-spin" />
            <p className="text-gray-600 font-medium">\u52a0\u8f7d\u4e2d...</p>
          </div>
        </div>
      )}
      {/* SKILL §7.4 用户端通用骨架（以「我的 Agent」为基准）：
            min-w-[1200px] overflow-x-clip 兜底小屏 + max-w-[1920px] mx-auto flex 大屏限宽，
            左右各 w-20（80px）占位带 + 中间 flex-1 min-w-0 px-[42px] py-8 内容区。
            保证切换 Tab 时两侧留白节奏与「我的 Agent」一致。 */}
      <div className="min-w-[1200px] overflow-x-clip">
        <div className="max-w-[1920px] mx-auto flex items-stretch page-enter">
          <div aria-hidden className="shrink-0 w-20 self-stretch" />
          <div className="flex-1 min-w-0 px-[42px] py-8">
        {/* Back */}
        <div className="flex items-center gap-3 mb-6">
          <Link href="/my-openclaw">
            <Button variant="ghost" size="sm" className="text-[var(--text-emphasis)] hover:text-[var(--text-strong)] -ml-2">
              <ArrowLeft className="w-4 h-4 mr-1" />
              返回
            </Button>
          </Link>
        </div>

        {/* Title */}
          <div className="flex items-center justify-between gap-4 mb-8">
          {/* 左侧：图标 + 名称/ID/badge */}
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-[4px] flex items-center justify-center text-xl flex-shrink-0"
              style={{ background: "linear-gradient(135deg, rgba(20,71,230,0.1), rgba(88,86,214,0.1))" }}>
              🦞
            </div>
            <div>
            {/* 第一行：名称 + 状态 badge（8 种状态动态渲染） */}
          <div className="flex items-start gap-2 min-w-0">
                <div ref={nameEditWrapperRef} className="group/name peer/name min-w-0 h-9">
                  {isNameEditing ? (
                    <div className="relative w-full">
                      <Input
                        ref={nameInputRef}
                        value={nameDraft}
                        onChange={(e) => {
                          const value = e.target.value.replace(/[\r\n]/g, "");
                          setNameDraft(value);
                          if (getAgentNameByteLength(value.trim()) > AGENT_NAME_MAX_BYTES) {
                            setNameError(`名称不能超过 ${AGENT_NAME_MAX_BYTES} 字节`);
                          } else if (nameError) {
                            setNameError("");
                          }
                        }}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            e.preventDefault();
                            saveNameEdit();
                          }
                          if (e.key === "Escape") {
                            e.preventDefault();
                            cancelNameEdit();
                          }
                        }}
                        aria-label="编辑 Agent 名称"
                        aria-invalid={!!nameError}
                        className={`h-9 w-full text-2xl font-bold text-[var(--text-strong)] bg-transparent rounded-[4px] ${nameError ? "border-[var(--text-danger)] focus-visible:ring-[var(--text-danger)]" : "border-[var(--border)] focus-visible:ring-[var(--ring)]"}`}
                      />
                      {nameError && <p className="absolute left-0 bottom-full mb-1 text-xs text-[var(--text-danger)]">{nameError}</p>}
                    </div>
                  ) : (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <button
                          type="button"
                          onClick={startNameEdit}
                          disabled={!canRenameByStatus}
                          className={`h-9 inline-flex items-center px-1 -ml-1 rounded-md transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ring)] ${canRenameByStatus ? "hover:bg-[var(--accent)]" : "cursor-not-allowed"} disabled:opacity-40`}
                          aria-label="重命名 Agent"
                        >

                          <h1 ref={nameTextRef} className="text-2xl font-bold text-[var(--text-strong)] leading-tight max-w-[460px] truncate">
                            {clawName}
                          </h1>
                          <span className="h-5 inline-flex items-center justify-center flex-shrink-0 overflow-hidden w-5 opacity-100">
                            <Pencil className="w-3.5 h-3.5 text-[var(--text-muted)]" />
                          </span>
                        </button>
                      </TooltipTrigger>
                      {!canRenameByStatus ? (
                        <TooltipContent side="top" className="text-xs">
                          当前状态不支持重命名
                        </TooltipContent>
                      ) : isNameOverflow ? (
                        <TooltipContent side="top" className="text-xs max-w-[520px] break-all">
                          {clawName}
                        </TooltipContent>
                      ) : null}

                    </Tooltip>
                  )}
                </div>
                <div className="transition-transform duration-150 peer-hover/name:translate-x-2 peer-focus-within/name:translate-x-2">
                  <TooltipProvider delayDuration={200}>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className={statusCfg.badgeClass}>
                          {statusCfg.spinning ? (
                            <Loader2 className="w-3.5 h-3.5 animate-spin" />
                          ) : null}
                          {statusCfg.label}
                        </span>
                      </TooltipTrigger>
                      {statusCfg.tooltipText && (
                        <TooltipContent side="top" className="text-xs">
                          {statusCfg.tooltipText}
                        </TooltipContent>
                      )}
                    </Tooltip>
                  </TooltipProvider>
                </div>
              </div>
              {/* 第二行：角色 → 类型 → ID → 版本 → 组织 → 项目 */}
              <div className="flex items-center flex-wrap gap-x-2 gap-y-1 mt-0.5">
                {/* 角色胶囊标签 */}
                {claw.roleName && (
                  <span className="inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-full flex-shrink-0 text-[var(--text-secondary)]"
                    style={{ background: "linear-gradient(135deg, rgba(20,71,230,0.08), rgba(88,86,214,0.05))", border: "1px solid rgba(20,71,230,0.1)" }}>
                    {claw.roleName}
                  </span>
                )}
                {/* Agent 类型 tag */}
                <span
                  className="inline-flex items-center text-[10px] font-semibold px-2 py-0.5 whitespace-nowrap flex-shrink-0"
                  style={{
                    background: "linear-gradient(135deg, rgba(20,71,230,0.1), rgba(88,86,214,0.1))",
                    color: "rgba(20,71,230,0.5)",
                    borderRadius: "0.375rem",
                  }}
                >
                  {(claw as any).agentType === "hermes"
                    ? "Hermes Agent"
                    : (claw as any).agentType === "lightclawace"
                    ? "LightClaw ACE"
                    : "OpenClaw"}
                </span>
                {/* 实例 ID */}
                <p className="text-xs text-[var(--text-muted)]">{claw.instanceId}</p>
                {/* 当前 Agent 版本号（mock 数据缺字段时兜底） */}
                <span className="text-xs text-[var(--text-weak)]">·</span>
                <span className="text-xs text-[var(--text-muted)] font-mono tabular-nums">
                  v{(claw as any).agentVersion ?? "2026.4.0"}
                </span>
                {/* 组织 / 项目：本地 Agent 不展示（本地无组织/项目归属概念） */}
                {!isLocalAgent && (
                  <>
                    {/* 组织（只读） */}
                    {claw.groupName && (
                      <>
                        <span className="text-xs text-[var(--text-weak)]">·</span>
                        <span className="text-xs text-[var(--text-muted)]">组织：{claw.groupName}</span>
                      </>
                    )}
                    {/* 项目（可编辑：可添加 / 可删除关联） */}
                    <span className="text-xs text-[var(--text-weak)]">·</span>
                    <span className="inline-flex items-center gap-1 text-xs text-[var(--text-muted)]">
                      项目：{claw.projectNames && claw.projectNames.length > 0 ? claw.projectNames.join("、") : "未关联"}
                      <button
                        type="button"
                        onClick={openProjectEdit}
                        className="ml-0.5 inline-flex items-center justify-center w-4 h-4 rounded hover:bg-[var(--bg-grey-hover)] text-[var(--text-muted)] hover:text-[var(--text-emphasis)] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ring)]"
                        aria-label="编辑项目关联"
                      >
                        <Pencil className="w-3 h-3" />
                      </button>
                    </span>
                  </>
                )}
                {recommendPush && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Badge color="blue" className="cursor-help text-[10px] py-0.5 px-1.5">
                        <Megaphone className="w-2.5 h-2.5" />
                        可更新 v{recommendPush.version}
                      </Badge>
                    </TooltipTrigger>
                    <TooltipContent side="top" className="max-w-[320px] text-xs leading-relaxed">
                      <div className="space-y-1">
                        <div className="font-medium">{recommendPush.message ?? `推荐更新到 v${recommendPush.version}`}</div>
                        <div className="opacity-70">
                          来自 {recommendPush.pushedBy} · {recommendPush.pushedAt}
                        </div>
                      </div>
                    </TooltipContent>
                  </Tooltip>
                )}
              </div>
            </div>
          </div>
          {/* 右侧：操作按鈕 */}
          <div className="flex items-center gap-2 flex-shrink-0">
            {/* 一键更新按鈕 + 气泡 */}
            <div className="relative flex items-center">
              {showUpdateBubble && !isUpdating && (
                <div className="absolute bottom-full right-0 mb-2 z-50">
                  <div className="relative bg-[var(--text-brand)] text-white text-xs rounded-[4px] px-3 py-2 shadow-sm leading-none whitespace-nowrap">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => setShowUpdateBubble(false)}
                      aria-label="关闭气泡提示"
                      className="absolute -top-1.5 -right-1.5 h-4 w-4 rounded-full bg-[var(--text-muted)] hover:bg-[var(--text-emphasis)] text-white hover:text-white p-0 [&_svg]:size-2.5"
                    >
                      <X />
                    </Button>
                    重磅来袭！升级版本，一键接入微信！
                    <div
                      className="absolute top-full right-4 w-0 h-0"
                      style={{
                        borderLeft: "5px solid transparent",
                        borderRight: "5px solid transparent",
                        borderTop: "5px solid var(--text-brand)",
                      }}
                    />
                  </div>
                </div>
              )}
              {isReadOnly ? (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span tabIndex={0}>
                      <Button
                        variant="tenant-outline"
                        size="claw-sm"
                        className="text-xs"
                        disabled
                      >
                        <ArrowUpCircle className="w-3.5 h-3.5" />
                        一键更新
                      </Button>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="bottom" className="text-xs max-w-[240px] leading-relaxed">
                    该 Agent 由他人共享，你没有修改权限
                  </TooltipContent>
                </Tooltip>
              ) : isUpdating ? (
                <Button
                  variant="tenant-outline"
                  size="claw-sm"
                  className="text-xs"
                  title="查看更新进度"
                  onClick={() => setShowUpdateProgressDialog(true)}
                >
                  <Loader2 className="w-3.5 h-3.5 animate-spin text-[var(--text-brand)]" />
                  更新中
                </Button>
              ) : !allowSelfUpgrade ? (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span tabIndex={0}>
                      <Button
                        variant="tenant-outline"
                        size="claw-sm"
                        className="text-xs"
                        disabled
                      >
                        <ArrowUpCircle className="w-3.5 h-3.5" />
                        一键更新
                      </Button>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="bottom" className="text-xs max-w-[240px] leading-relaxed">
                    管理员已关闭"自助更新"，请联系管理员开启
                  </TooltipContent>
                </Tooltip>
              ) : (claw as any).agentType && (claw as any).agentType !== "openclaw" ? (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span tabIndex={0}>
                      <Button
                        variant="tenant-outline"
                        size="claw-sm"
                        className="text-xs"
                        disabled
                      >
                        <ArrowUpCircle className="w-3.5 h-3.5" />
                        一键更新
                      </Button>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="bottom" className="text-xs">
                    当前仅 OpenClaw 支持
                  </TooltipContent>
                </Tooltip>
              ) : (
                <Button
                  variant="tenant-outline"
                  size="claw-sm"
                  className="text-xs"
                  onClick={() => {
                    // 检查当前实例的 agentType 是否有对应的生效镜像
                    const agentTypeRaw = (claw as any).agentType as string | undefined;
                    if (agentTypeRaw) {
                      try {
                        // 将实例的 agentType 小写形式映射到镜像管理中的标准形式
                        // 实例 agentType: openclaw/hermes/lightclawace（小写）
                        // 镜像 agentType: OpenClaw/HermesAgent/LightClawACE
                        const typeMap: Record<string, string> = {
                          openclaw: "openclaw",
                          hermes: "hermesagent",
                          lightclawace: "lightclawace",
                        };
                        const normalizedType = typeMap[agentTypeRaw.toLowerCase()] ?? agentTypeRaw.toLowerCase();
                        const raw = localStorage.getItem("admin_images");
                        const imgs: Array<{ agentType: string; active: boolean }> = raw ? JSON.parse(raw) : [];
                        // 将镜像的 agentType 也进行同样的规范化匹配
                        const hasActive = imgs.some((i) => {
                          const imgNorm = typeMap[i.agentType.toLowerCase()] ?? i.agentType.toLowerCase();
                          return imgNorm === normalizedType && i.active;
                        });
                        if (!hasActive) {
                          toast.error("暂无生效的Agent类型镜像，请联系管理员处理");
                          return;
                        }
                      } catch { /* ignore */ }
                    }
                    setShowUpdateConfirmDialog(true);
                  }}
                >
                  <ArrowUpCircle className="w-3.5 h-3.5" />
                  一键更新
                </Button>
              )}
            </div>
            {/* 开启面板按鈕 */}
            <Tooltip>
              <TooltipTrigger asChild>
                <span tabIndex={allowPanelAccess ? -1 : 0}>
                  <Button
                    variant="tenant-outline"
                    size="claw-sm"
                    className="text-xs"
                    onClick={handleOpenWebUI}
                    disabled={!allowPanelAccess}
                  >
                    <Monitor className="w-3.5 h-3.5" />
                    开启Agent面板
                  </Button>
                </span>
              </TooltipTrigger>
              {!allowPanelAccess && (
                <TooltipContent side="top" className="text-xs">
                  管理员未开启访问权限
                </TooltipContent>
              )}
            </Tooltip>
            {activeDetailTab === "basic" && !isReadOnly && (
              <Button
                variant="tenant-outline"
                size="claw-sm"
                className="text-xs"
                onClick={() => { setMigrationOpen(true); setMigrationStep("export"); setMigrationCosUrl(""); setMigrationUploaded(false); setMigrationChecking(false); setMigrationCheckFailed(false); setMigrationCheckCount(0); setMigrationError(""); setMigrationCommandReady(false); setVerifyResults([]); setImportSteps([{ label: "下载数据包", status: "pending" }, { label: "备份当前配置", status: "pending" }, { label: "解压并覆盖", status: "pending" }, { label: "重启 Gateway", status: "pending" }, { label: "验证生效", status: "pending" }]); setTimeout(() => setMigrationCommandReady(true), 1800); }}
              >
                <ArrowLeftRight className="w-3.5 h-3.5" />
                Agent 迁移
              </Button>
            )}
          </div>
        </div>

        {/* 被共享者只读提示 */}
        {isReadOnly && (
          <Alert variant="info" className="mb-4">
            <AlertInfoIcon />
            <AlertDescription>
              该 Agent 由 {sharedOwnerName || "他人"} 共享给你，你可以查看配置，但不能修改。如需调整请联系创建者。
            </AlertDescription>
          </Alert>
        )}

        {/* Segment nav + content area */}
        <Segment value={activeDetailTab} onValueChange={setActiveDetailTab}>
          <SegmentList className="my-4">
            <SegmentItem value="basic">基础配置</SegmentItem>
            <SegmentItem value="tools">工具管理</SegmentItem>
            <SegmentItem value="memory">记忆管理</SegmentItem>
            <SegmentItem value="files">网盘管理</SegmentItem>
            <SegmentItem value="doctor">龙虾医院</SegmentItem>
          </SegmentList>

          {/* 只读模式下：整个配置内容区置灰并屏蔽交互（可切 Tab 查看，但不可编辑） */}
          <div className={`mt-1${isReadOnly ? " opacity-60 pointer-events-none select-none" : ""}`} aria-disabled={isReadOnly}>

          {/* 基础配置 tab */}
          {activeDetailTab === "basic" && (
            <div className="grid grid-cols-3 gap-5" style={{ minHeight: 0, alignItems: "start" }}>

          {/* ===== Model Column ===== */}
          <TenantCard padding="none" state="static" className="overflow-hidden flex flex-col relative h-[749px] [box-shadow:var(--shadow-card)]">
            <div className="p-5 border-b border-[var(--border)]">
              <div className="flex items-center gap-2 justify-center">
                <div className="w-6 h-6 rounded-[4px] bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center">
                  <Brain className="w-3.5 h-3.5 text-white" />
                </div>
                <h2 className="font-semibold text-[var(--text-strong)]">模型 (Models)</h2>
              </div>
            </div>

            {/* 管控端关闭「允许用户配置模型」时的锁定蒙版 */}
            {!userConfigModelEnabled && (
              <div className="absolute inset-0 top-[61px] z-10 bg-white/80 backdrop-blur-sm flex flex-col items-center justify-center gap-3 rounded-b-2xl">
                <Lock className="w-8 h-8 text-[var(--text-weak)]" />
                <p className="text-xs text-[var(--text-muted)] text-center leading-relaxed px-6">模型已由管理员统一配置，无需手动调整</p>
              </div>
            )}

            {/* Scrollable content area */}
            <div className="overflow-y-auto flex-1">
            {/* Upper: config inputs */}
            <div className="p-5 space-y-3">
              {/* 模型来源选择：管理员配置的模型 / 自行配置 */}
              <div className="space-y-1.5">
                <label className="text-xs text-[var(--text-secondary)]">模型来源</label>
                <Select value={modelSource} onValueChange={(v) => handleModelSourceChange(v as "admin" | "self")}>
                  <SelectTrigger tenant className="w-full bg-[var(--accent)] border-[var(--border)]">
                    <SelectValue placeholder="选择模型来源" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="admin">{MODEL_PROVIDER_GROUP_LABELS.admin}</SelectItem>
                    <SelectItem value="self">{MODEL_PROVIDER_GROUP_LABELS.self}</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {/* 自行配置：先选配置类型（Coding Plan / 模型 API / 自定义模型） */}
              {modelSource === "self" && (
                <div className="space-y-1.5">
                  <label className="text-xs text-[var(--text-secondary)]">配置类型</label>
                  <TenantSegmentGroup className="w-full">
                    {SELF_CONFIG_CATEGORY_ORDER.map((cat) => (
                      <TenantSegmentOption
                        key={cat}
                        className="flex-1"
                        active={selfCategory === cat}
                        onClick={() => handleSelfCategoryChange(cat)}
                      >
                        {SELF_CONFIG_CATEGORY_LABELS[cat]}
                      </TenantSegmentOption>
                    ))}
                  </TenantSegmentGroup>
                </div>
              )}

              {/* 模型厂商选择（自定义模型时隐藏） */}
              {!isCustomProvider && (
                <div className="space-y-1.5">
                  <label className="text-xs text-[var(--text-secondary)]">
                    模型厂商
                  </label>
                  <Select value={selectedProvider} onValueChange={handleProviderChange}>
                    <SelectTrigger tenant className="w-full bg-[var(--accent)] border-[var(--border)]">
                      <SelectValue placeholder="选择模型厂商" />
                    </SelectTrigger>
                    <SelectContent>
                      {sourceProviders.map((p) => (
                        <SelectItem key={p.value} value={p.value}>
                          <div className="flex items-center gap-2">
                            <span>{p.label}</span>
                          </div>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}

              {/* 模型版本选择（自定义模型厂商时隐藏） */}
              {!isCustomProvider && (
                <div className="space-y-1.5">
                  <label className="text-xs text-[var(--text-secondary)]">
                    模型版本
                  </label>
                  <Select value={selectedModel} onValueChange={setSelectedModel}>
                    <SelectTrigger tenant className="w-full bg-[var(--accent)] border-[var(--border)]">
                      <SelectValue placeholder="选择模型版本" />
                    </SelectTrigger>
                    <SelectContent>
                      {currentVersions.map((v) => (
                        <SelectItem key={v.value} value={v.value}>
                          <span>{v.label}</span>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}

              {/* ===== 自行配置·公开模型 表单（API Key / 限额 / 高级配置）===== */}
              {isPublicSelfProvider && (
                <div className="space-y-3 pt-1">
                  {/* API Key */}
                  <div className="space-y-1.5">
                    <label className="text-xs text-[var(--text-secondary)]">
                      API Key<span className="text-[var(--text-danger)] ml-0.5">*</span>
                    </label>
                    <Input
                      tenant
                      type="password"
                      placeholder="请输入 API Key"
                      value={publicForm.apiKey}
                      onChange={(e) => setPublicForm({ ...publicForm, apiKey: e.target.value })}
                      className="bg-[var(--accent)] border-[var(--border)] text-sm"
                    />
                  </div>
                  {/* 每日 Tokens 上限 */}
                  <div className="space-y-1.5">
                    <label className="text-xs text-[var(--text-secondary)]">
                      每日 Tokens 数量上限<span className="text-[var(--text-danger)] ml-0.5">*</span>
                    </label>
                    <Input
                      tenant
                      type="number"
                      placeholder="请输入每日 Tokens 上限"
                      value={publicForm.dailyLimit}
                      onChange={(e) => setPublicForm({ ...publicForm, dailyLimit: e.target.value })}
                      className="bg-[var(--accent)] border-[var(--border)] text-sm"
                    />
                  </div>
                  {/* 高级配置：最大输出长度 + 请求头 */}
                  <div className="rounded-[var(--radius-card)] border border-[var(--border)] overflow-hidden">
                    <button
                      type="button"
                      onClick={() => setAdvancedOpen(v => !v)}
                      className="w-full flex items-center justify-between px-4 py-3 bg-[var(--accent)] hover:bg-[var(--muted)] transition-colors"
                    >
                      <span className="text-sm font-medium text-[var(--text-emphasis)]">高级配置</span>
                      <span className="flex items-center gap-1.5 text-xs text-[var(--text-muted)]">
                        <span>非必填</span>
                        {advancedOpen ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
                      </span>
                    </button>
                    {advancedOpen && (
                      <div className="px-4 py-3 space-y-3 bg-[var(--accent)] border-t border-[var(--border)]">
                        {/* 最大输出长度 */}
                        <div className="space-y-1.5">
                          <div className="flex items-center gap-1">
                            <span className="text-xs text-[var(--text-secondary)]">最大输出长度</span>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="cursor-default"><Info className="w-3 h-3 text-[var(--text-muted)]" /></span>
                              </TooltipTrigger>
                              <TooltipContent className="max-w-[240px] text-xs leading-relaxed">
                                maxTokens，即模型单次回复时最多输出的 Token 数。
                              </TooltipContent>
                            </Tooltip>
                          </div>
                          <Input
                            tenant
                            type="number"
                            placeholder="请输入最大输出长度 maxTokens"
                            value={advancedConfig.maxTokens}
                            onChange={(e) => setAdvancedConfig({ ...advancedConfig, maxTokens: e.target.value })}
                            className="bg-white border-[var(--border)] text-sm"
                          />
                        </div>
                        {/* 请求头 */}
                        <div className="space-y-1.5">
                          <div className="flex items-center gap-1">
                            <span className="text-xs text-[var(--text-secondary)]">请求头</span>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="cursor-default"><Info className="w-3 h-3 text-[var(--text-muted)]" /></span>
                              </TooltipTrigger>
                              <TooltipContent className="max-w-[240px] text-xs leading-relaxed">
                                headers，在 HTTP 请求中用于传递认证信息或数据格式等元数据的参数。
                              </TooltipContent>
                            </Tooltip>
                          </div>
                          <div className="space-y-2">
                            {advancedConfig.headers.map((entry, idx) => (
                              <div key={idx} className="flex items-center gap-2">
                                <Input
                                  tenant
                                  placeholder="key"
                                  value={entry.key}
                                  onChange={(e) => {
                                    const next = advancedConfig.headers.map((h, i) => i === idx ? { ...h, key: e.target.value } : h);
                                    setAdvancedConfig({ ...advancedConfig, headers: next });
                                  }}
                                  className="bg-white border-[var(--border)] text-sm w-[36%] shrink-0"
                                />
                                <Input
                                  tenant
                                  placeholder="value"
                                  value={entry.value}
                                  onChange={(e) => {
                                    const next = advancedConfig.headers.map((h, i) => i === idx ? { ...h, value: e.target.value } : h);
                                    setAdvancedConfig({ ...advancedConfig, headers: next });
                                  }}
                                  className="bg-white border-[var(--border)] text-sm flex-1"
                                />
                                <button
                                  type="button"
                                  onClick={() => setAdvancedConfig({ ...advancedConfig, headers: advancedConfig.headers.filter((_, i) => i !== idx) })}
                                  className="shrink-0 text-[var(--text-weak)] hover:text-[var(--text-danger)] transition-colors"
                                  aria-label="删除请求头"
                                >
                                  <Plus className="w-3.5 h-3.5 rotate-45" />
                                </button>
                              </div>
                            ))}
                            <button
                              type="button"
                              onClick={() => setAdvancedConfig({ ...advancedConfig, headers: [...advancedConfig.headers, { key: "", value: "" }] })}
                              className="flex items-center gap-1 text-xs text-[var(--text-brand)] hover:text-[var(--text-brand)] transition-colors mt-1"
                            >
                              <Plus className="w-3 h-3" />
                              添加请求头
                            </button>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}

              {isCustomProvider && (
                <div className="space-y-3 pt-1">
                  <TenantSegmentGroup className="w-full">
                    <TenantSegmentOption
                      className="flex-1"
                      active={customInputMode === "json"}
                      onClick={() => setCustomInputMode("json")}
                    >
                      JSON 输入
                    </TenantSegmentOption>
                    <TenantSegmentOption
                      className="flex-1"
                      active={customInputMode === "form"}
                      onClick={() => setCustomInputMode("form")}
                    >
                      表单输入
                    </TenantSegmentOption>
                  </TenantSegmentGroup>

                  {customInputMode === "json" ? (
                    <Textarea
                      value={customJson}
                      onChange={(e) => setCustomJson(e.target.value)}
                      className="font-mono text-xs bg-[var(--accent)] border-[var(--border)] min-h-[180px] resize-none"
                      spellCheck={false}
                    />
                  ) : (
                    <div className="space-y-2">
                      {[
                        { key: "provider", label: "请输入自定义模型 provider" },
                        { key: "base_url", label: "请输入自定义模型 base_url" },
                        { key: "api", label: "请输入自定义模型 api" },
                        { key: "api_key", label: "请输入自定义模型 api_key" },
                        { key: "model_id", label: "请输入自定义模型 model.id" },
                        { key: "model_name", label: "请输入自定义模型 model.name" },
                      ].map((field) => (
                        <Input
                          tenant
                          key={field.key}
                          placeholder={field.label}
                          value={customForm[field.key as keyof typeof customForm]}
                          onChange={(e) => setCustomForm({ ...customForm, [field.key]: e.target.value })}
                          className="bg-[var(--accent)] border-[var(--border)] text-sm"
                        />
                      ))}
                    </div>
                  )}

                  {/* 高级配置：仅表单模式 */}
                  {customInputMode === "form" && (
                    <div className="rounded-xl border border-[var(--border)] overflow-hidden">
                      <button
                        type="button"
                        onClick={() => setAdvancedOpen(v => !v)}
                        className="w-full flex items-center justify-between px-4 py-3 bg-[var(--accent)] hover:bg-[var(--accent)] transition-colors"
                      >
                        <span className="text-sm font-medium text-[var(--text-emphasis)]">高级配置</span>
                        <span className="flex items-center gap-1.5 text-xs text-[var(--text-muted)]">
                          <span>非必填</span>
                          {advancedOpen ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
                        </span>
                      </button>
                      {advancedOpen && (
                        <div className="px-4 py-3 space-y-3 bg-[var(--card)]">
                          {/* 上下文长度 */}
                          <div className="space-y-1.5">
                            <div className="flex items-center gap-1">
                              <span className="text-xs text-[var(--text-secondary)]">上下文长度</span>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="cursor-default"><Info className="w-3 h-3 text-[var(--text-muted)]" /></span>
                                </TooltipTrigger>
                                <TooltipContent className="max-w-[240px] text-xs leading-relaxed">
                                  contextWindow，指的是模型总上下文窗口大小。
                                </TooltipContent>
                              </Tooltip>
                            </div>
                            <Input
                              type="number"
                              placeholder="请输入上下文长度 contextWindow"
                              value={advancedConfig.contextWindow}
                              onChange={(e) => setAdvancedConfig({ ...advancedConfig, contextWindow: e.target.value })}
                              className="bg-[var(--accent)] border-[var(--border)] text-sm"
                            />
                          </div>
                          {/* 最大输出长度 */}
                          <div className="space-y-1.5">
                            <div className="flex items-center gap-1">
                              <span className="text-xs text-[var(--text-secondary)]">最大输出长度</span>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="cursor-default"><Info className="w-3 h-3 text-[var(--text-muted)]" /></span>
                                </TooltipTrigger>
                                <TooltipContent className="max-w-[240px] text-xs leading-relaxed">
                                  maxTokens，即模型单次回复时最多输出的 Token 数。
                                </TooltipContent>
                              </Tooltip>
                            </div>
                            <Input
                              type="number"
                              placeholder="请输入最大输出长度 maxTokens"
                              value={advancedConfig.maxTokens}
                              onChange={(e) => setAdvancedConfig({ ...advancedConfig, maxTokens: e.target.value })}
                              className="bg-[var(--accent)] border-[var(--border)] text-sm"
                            />
                          </div>
                          {/* 请求头 */}
                          <div className="space-y-1.5">
                            <div className="flex items-center gap-1">
                              <span className="text-xs text-[var(--text-secondary)]">请求头</span>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="cursor-default"><Info className="w-3 h-3 text-[var(--text-muted)]" /></span>
                                </TooltipTrigger>
                                <TooltipContent className="max-w-[240px] text-xs leading-relaxed">
                                  headers，在 HTTP 请求中用于传递认证信息或数据格式等元数据的参数。
                                </TooltipContent>
                              </Tooltip>
                            </div>
                            <div className="space-y-2">
                              {advancedConfig.headers.map((entry, idx) => (
                                <div key={idx} className="flex items-center gap-2">
                                  <Input
                                    placeholder="key"
                                    value={entry.key}
                                    onChange={(e) => {
                                      const next = advancedConfig.headers.map((h, i) => i === idx ? { ...h, key: e.target.value } : h);
                                      setAdvancedConfig({ ...advancedConfig, headers: next });
                                    }}
                                    className="bg-[var(--accent)] border-[var(--border)] text-sm w-[36%] shrink-0"
                                  />
                                  <Input
                                    placeholder="value"
                                    value={entry.value}
                                    onChange={(e) => {
                                      const next = advancedConfig.headers.map((h, i) => i === idx ? { ...h, value: e.target.value } : h);
                                      setAdvancedConfig({ ...advancedConfig, headers: next });
                                    }}
                                    className="bg-[var(--accent)] border-[var(--border)] text-sm flex-1"
                                  />
                                  <button
                                    type="button"
                                    onClick={() => setAdvancedConfig({ ...advancedConfig, headers: advancedConfig.headers.filter((_, i) => i !== idx) })}
                                    className="shrink-0 text-[var(--text-weak)] hover:text-[var(--text-danger)] transition-colors"
                                  >
                                    <Plus className="w-3.5 h-3.5 rotate-45" />
                                  </button>
                                </div>
                              ))}
                              <button
                                type="button"
                                onClick={() => setAdvancedConfig({ ...advancedConfig, headers: [...advancedConfig.headers, { key: "", value: "" }] })}
                                className="flex items-center gap-1 text-xs text-[var(--text-brand)] hover:text-[var(--text-brand)] transition-colors mt-1"
                              >
                                <Plus className="w-3 h-3" />
                                添加请求头
                              </button>
                            </div>
                          </div>
                        </div>
                      )}
                    </div>
                  )}

                  {/* 多模态开关 */}
                  <div className="flex items-center justify-between rounded-[4px] bg-[var(--accent)] border border-[var(--border)] px-3 py-2.5">
                    <div className="flex flex-col">
                      <span className="text-sm font-medium text-[var(--text-emphasis)]">多模态模型</span>
                      <span className="text-xs text-[var(--text-muted)] mt-0.5">支持图片、文字多模态输入</span>
                    </div>
                    <button
                      type="button"
                      role="switch"
                      aria-checked={customMultimodal}
                      onClick={() => setCustomMultimodal(v => !v)}
                      className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 ${
                        customMultimodal ? "bg-[var(--text-brand)]" : "bg-[var(--accent)]"
                      }`}
                    >
                      <span
                        className={`pointer-events-none block h-4 w-4 rounded-full bg-white shadow-sm ring-0 transition-transform ${
                          customMultimodal ? "translate-x-4" : "translate-x-0"
                        }`}
                      />
                    </button>
                  </div>

                  <div className="rounded-[12px] bg-amber-50 border border-amber-100 p-3 text-xs text-amber-700 leading-relaxed">
                    使用自定义模型需自行承担 Tokens 费用，不计入公司提供的大模型 Tokens 范围。
                    <a href="#" className="inline-flex items-center gap-0.5 text-[var(--text-brand)] hover:text-[var(--text-brand)] underline underline-offset-2 ml-1 transition-colors">
                      自定义模型配置指引 <ExternalLink className="w-3 h-3" />
                    </a>
                  </div>

                </div>
              )}

              <div className="flex gap-2">
                {/* 连通性检测：仅自定义模型 / 自行配置的公开模型需要 */}
                {(isCustomProvider || isPublicSelfProvider) && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        className="flex-1 text-sm"
                        variant="tenant-outline"
                        onClick={handleConnectTest}
                        disabled={isConfiguring || connectTesting}
                      >
                        {connectTesting && <Loader2 className="w-4 h-4 animate-spin mr-1" />}
                        {connectTesting ? "检测中…" : "连通性检测"}
                      </Button>
                    </TooltipTrigger>
                    {isConfiguring && (
                      <TooltipContent side="top" className="bg-[var(--text-emphasis)] text-white text-xs">
                        当前TAT状态不在线，无法操作
                      </TooltipContent>
                    )}
                  </Tooltip>
                )}
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      className="flex-1 text-sm"
                      variant="tenant-outline"
                      onClick={handleApplyModel}
                      disabled={isConfiguring}
                    >
                      {appliedModels.some(m => m.primary) ? "添加备用模型" : "设为主模型"}
                    </Button>
                  </TooltipTrigger>
                  {isConfiguring && (
                    <TooltipContent side="top" className="bg-[var(--text-emphasis)] text-white text-xs">
                      当前TAT状态不在线，无法操作
                    </TooltipContent>
                  )}
                </Tooltip>
              </div>

            </div>
            {/* Lower: model list */}
            <div className="px-5 pb-5">
              <div className="pt-2 border-t border-[var(--border)]">
                <div className="flex items-center gap-1.5 mb-2">
                  <p className="text-xs text-[var(--text-muted)]">已应用模型</p>
                </div>
                {/* 主模型组织 */}
                {appliedModels.some(m => m.primary) && (
                  <div className="mb-3">
                    <p className="text-xs text-[var(--text-muted)] mb-1.5">主模型</p>
                    <div className="space-y-1.5">
                      {appliedModels.filter(m => m.primary).map((model) => (
                        <div
                          key={model.id}
                          className="rounded-[12px] border transition-all bg-[var(--accent)] border-[var(--border)] p-2.5"
                        >
                          <div className="flex items-center justify-between">
                            <div className="flex items-center min-w-0 flex-1 overflow-hidden">
                              <div className="flex flex-col min-w-0 overflow-hidden">
                                {model.isCustom ? (
                                  <>
                                    <span className="text-sm font-medium text-[var(--text-strong)] leading-tight truncate block">自定义模型</span>
                                    {model.customName && (
                                      <span className="text-xs text-[var(--text-muted)] leading-tight mt-0.5 truncate block">{model.customName}</span>
                                    )}
                                    {model.multimodal && (
                                      <Badge color="blue" className="mt-1 px-1.5 py-0.5 text-[10px] w-fit">多模态</Badge>
                                    )}
                                  </>
                                ) : (
                                  <>
                                    <span className="text-sm font-medium text-[var(--text-strong)] leading-tight truncate block">{model.providerLabel}</span>
                                    {model.versionLabel && (
                                      <span className="text-xs text-[var(--text-muted)] leading-tight mt-0.5 truncate block">{model.versionLabel}</span>
                                    )}
                                  </>
                                )}
                              </div>
                            </div>
                            <div className="flex items-center gap-2 shrink-0 ml-2">
                              <StatusTag mode="fill" variant="green">主模型</StatusTag>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <button
                                    type="button"
                                    onClick={() => {
                                      setModelConfirmDialog({ open: true, type: "delete", modelId: model.id });
                                    }}
                                    className="p-1 rounded text-[var(--text-weak)] hover:text-[var(--text-danger)] transition-colors"
                                  >
                                    <Trash2 className="w-3.5 h-3.5" />
                                  </button>
                                </TooltipTrigger>
                                <TooltipContent side="top" className="bg-[var(--text-emphasis)] text-white text-xs">
                                  删除模型
                                </TooltipContent>
                              </Tooltip>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
                {/* 备用模型组织 */}
                {appliedModels.some(m => !m.primary) && (
                  <div>
                    <p className="text-xs text-[var(--text-muted)] mb-1">
                      备用模型 ({appliedModels.filter(m => !m.primary).length})
                    </p>
                    <Alert variant="info" className="mb-2 text-xs">
                      <Info />
                      <AlertDescription>主模型不可用时会自动切换备用模型，此时备用模型消耗的token将统计到主模型下</AlertDescription>
                    </Alert>
                    <div className="space-y-1.5">
                      {[...appliedModels.filter(m => !m.primary)].sort((a, b) => b.addedAt - a.addedAt).map((model) => (
                        <div
                          key={model.id}
                          className="rounded-[12px] border transition-all bg-[var(--card)] border-[var(--border)] hover:bg-[var(--accent)] p-2.5"
                        >
                          <div className="flex items-center justify-between">
                            <div className="flex items-center min-w-0 flex-1 overflow-hidden">
                              <div className="flex flex-col min-w-0 overflow-hidden">
                                {model.isCustom ? (
                                  <>
                                    <span className="text-sm font-medium text-[var(--text-strong)] leading-tight truncate block">自定义模型</span>
                                    {model.customName && (
                                      <span className="text-xs text-[var(--text-muted)] leading-tight mt-0.5 truncate block">{model.customName}</span>
                                    )}
                                    {model.multimodal && (
                                      <Badge color="blue" className="mt-1 px-1.5 py-0.5 text-[10px] w-fit">多模态</Badge>
                                    )}
                                  </>
                                ) : (
                                  <>
                                    <span className="text-sm font-medium text-[var(--text-strong)] leading-tight truncate block">{model.providerLabel}</span>
                                    {model.versionLabel && (
                                      <span className="text-xs text-[var(--text-muted)] leading-tight mt-0.5 truncate block">{model.versionLabel}</span>
                                    )}
                                  </>
                                )}
                              </div>
                            </div>
                            <div className="flex items-center gap-2 shrink-0 ml-2">
                              <StatusTag mode="fill" variant="gray">备用</StatusTag>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <button
                                    type="button"
                                    onClick={() => {
                                      setModelConfirmDialog({ open: true, type: "set-primary", modelId: model.id });
                                    }}
                                    className="p-1 rounded opacity-60 hover:opacity-90 transition-opacity focus:outline-none"
                                    aria-label="切换为主模型"
                                  >
                                    <ArrowLeftRight className="w-3.5 h-3.5" />
                                  </button>
                                </TooltipTrigger>
                                <TooltipContent side="top" className="bg-[var(--text-emphasis)] text-white text-xs">
                                  切换为主模型
                                </TooltipContent>
                              </Tooltip>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <button
                                    type="button"
                                    onClick={() => {
                                      setModelConfirmDialog({ open: true, type: "delete-backup", modelId: model.id });
                                    }}
                                    className="p-1 rounded text-[var(--text-weak)] hover:text-[var(--text-danger)] transition-colors"
                                  >
                                    <Trash2 className="w-3.5 h-3.5" />
                                  </button>
                                </TooltipTrigger>
                                <TooltipContent side="top" className="bg-[var(--text-emphasis)] text-white text-xs">
                                  删除模型
                                </TooltipContent>
                              </Tooltip>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
            </div>{/* end scrollable */}
          </TenantCard>

          {/* ===== Channel Column ===== */}
          <TenantCard padding="none" state="static" className="!rounded-[4px] overflow-hidden flex flex-col relative h-[749px] [box-shadow:var(--shadow-card)]">
            <div className="p-5 border-b border-[var(--border)]">
              <div className="flex items-center gap-2 justify-center">
                <div className="w-6 h-6 rounded-[4px] bg-gradient-to-br from-green-500 to-green-600 flex items-center justify-center">
                  <MessageSquare className="w-3.5 h-3.5 text-white" />
                </div>
                <h2 className="font-semibold text-[var(--text-strong)]">通道 (Channels)</h2>
              </div>
            </div>

            {/* 管控端关闭「允许用户配置通道」时的锁定蒙版 */}
            {!userConfigChannelEnabled && (
              <div className="absolute inset-0 top-[61px] z-10 bg-white/80 backdrop-blur-sm flex flex-col items-center justify-center gap-3 rounded-b-2xl">
                <Lock className="w-8 h-8 text-[var(--text-weak)]" />
                <p className="text-xs text-[var(--text-muted)] text-center leading-relaxed px-6">通道已由管理员统一配置，无需手动调整</p>
              </div>
            )}

            {/* Upper: config inputs - fixed */}
            <div className="p-5 space-y-3 flex-shrink-0">
              {/* 通道下拉 - 固定宽度 */}
              <div className="flex items-center gap-2">
                <Select value={selectedChannel} onValueChange={(v) => { setSelectedChannel(v); setChannelFields({}); setFeishuConfigMode("quick"); setWeworkConfigMode("quick"); }}>
                  <SelectTrigger tenant className="w-full bg-[var(--accent)] border-[var(--border)]">
                    <SelectValue placeholder="选择通道类型" />
                  </SelectTrigger>
                  <SelectContent>
                    {visibleBuiltinChannels.map((ch) => (
                      <SelectItem key={ch.value} value={ch.value}>{ch.label}</SelectItem>
                    ))}
                    {visibleCustomChannels.length > 0 && (
                      <>
                        <div className="px-2 py-1.5 text-xs text-[var(--text-muted)] font-medium border-t border-[var(--border)] mt-1 pt-2">自定义通道</div>
                        {visibleCustomChannels.map((cc) => (
                          <SelectItem key={`admin_custom_${cc.id}`} value={`admin_custom_${cc.id}`}>{cc.name}</SelectItem>
                        ))}
                      </>
                    )}
                  </SelectContent>
                </Select>
                {currentChannelConfig?.hasInfoIcon && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button className="shrink-0 text-[var(--text-muted)] hover:text-[var(--text-brand)] transition-colors">
                        <Info className="w-4 h-4" />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent side="right" className="p-0 border-0 shadow-xl bg-transparent" sideOffset={8}>
                      <img
                        src="https://d2xsxph8kpxj0f.cloudfront.net/310519663415970324/bygiZj33T3TUvGMBPvApKE/pasted_file_To1FVK_image_06b2d1cc.png"
                        alt="企业微信通道示意图"
                        className="rounded-[4px] max-w-xs"
                        style={{ width: 320 }}
                      />
                    </TooltipContent>
                  </Tooltip>
                )}
              </div>

              {/* 动态配置输入区 */}
              {renderChannelInputs()}

              {/* 操作按钮 */}
              <Button className="w-full text-sm" variant="tenant-outline" onClick={handleAddChannel}>
                {(currentChannelConfig?.feishuMode && feishuConfigMode === "quick") || (currentChannelConfig?.weworkMode && weworkConfigMode === "quick") || currentChannelConfig?.wechatMode ? "前往授权" : "添加并应用"}
              </Button>

              {/* 底部说明 */}
              <p className="text-xs text-[var(--text-muted)] leading-relaxed">
                {currentChannelConfig?.descText}
                <a href={currentChannelConfig?.detailUrl || "#"} className="inline-flex items-center gap-0.5 text-[var(--text-brand)] hover:text-[var(--text-brand)] underline underline-offset-2 ml-1 transition-colors">
                  配置指引<ExternalLink className="w-3 h-3" />
                </a>
              </p>

            </div>
            {/* Lower: applied channels - scrollable */}
            <div className="px-5 pb-5 overflow-y-auto flex-1">
              <div className="pt-2 border-t border-[var(--border)]">
                <p className="text-xs text-[var(--text-muted)] mb-2">已接入通道（{appliedChannels.length}）</p>
                {appliedChannels.length > 0 && (
                  <div className="space-y-1">
                    {appliedChannels.map((ch, chIdx) => (
                      <div key={chIdx} className="rounded-[4px] bg-[var(--accent)] border border-[var(--border)] overflow-hidden">
                        {/* 折叠行 */}
                        <div className="flex items-center justify-between px-2.5 py-2">
                          {ch.channelValue === "wechat" ? (
                            <span className="text-sm font-medium text-[var(--text-strong)] truncate flex-1 min-w-0 pl-[18px]">{ch.type}</span>
                          ) : (
                            <button
                              className="flex items-center gap-1.5 flex-1 min-w-0 text-left"
                              onClick={() => toggleExpandChannel(chIdx)}
                            >
                              {expandedChannelIdx === chIdx
                                ? <ChevronDown className="w-3 h-3 text-[var(--text-muted)] shrink-0" />
                                : <ChevronRight className="w-3 h-3 text-[var(--text-muted)] shrink-0" />
                              }
                              <span className="text-sm font-medium text-[var(--text-strong)] truncate">{ch.type}</span>
                            </button>
                          )}
                          <div className="flex items-center gap-2 shrink-0">
                            <StatusTag mode="fill" variant="green">运行中</StatusTag>
                            <button
                              onClick={() => {
                                setAppliedChannels(appliedChannels.filter((_, i) => i !== chIdx));
                                if (expandedChannelIdx === chIdx) setExpandedChannelIdx(null);
                              }}
                              className="text-[var(--text-weak)] hover:text-[var(--text-danger)] transition-colors"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          </div>
                        </div>
                        {/* 展开配置项（微信无展开） */}
                        {ch.channelValue !== "wechat" && expandedChannelIdx === chIdx && renderAppliedChannelDetail(chIdx, ch)}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </TenantCard>

          {/* ===== Skills Column ===== */}
          <TenantCard padding="none" state="static" className="overflow-visible flex flex-col h-[749px] [box-shadow:var(--shadow-card)]">
            <div className="p-5 border-b border-[var(--border)]">
              <div className="flex items-center gap-2 justify-center">
                <div className="w-6 h-6 rounded-[4px] bg-gradient-to-br from-purple-500 to-purple-600 flex items-center justify-center">
                  <Puzzle className="w-3.5 h-3.5 text-white" />
                </div>
                <div className="flex items-baseline gap-1 font-semibold text-[var(--text-strong)]">
                  <div className="relative inline-flex">
                    技能
                    <SkillManagementUpdateNotice />
                  </div>
                  <span>(Skills)</span>
                </div>
              </div>
            </div>

            {/* Upper: search + install - fixed */}
            <div className="p-5 space-y-3 flex-shrink-0">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-muted)]" />
                <Input
                  tenant
                  placeholder="请输入准确 Skill 名称"
                  value={skillSearch}
                  onChange={(e) => setSkillSearch(e.target.value)}
                  className="pl-9 text-xs"
                />
              </div>

              {(() => {
                const hasQueueing = pendingSkills.some(s => s.status === "installing" || s.status === "pending");
                const hasDistributingSkills = pendingSkills.some(s => s.status === "installing");
                return (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="w-full block" tabIndex={hasQueueing ? 0 : -1}>
                        <Button
                          className="w-full text-sm"
                          variant="tenant-outline"
                          disabled={hasQueueing}
                          onClick={hasQueueing ? undefined : () => {
                            if (!skillSearch.trim()) {
                              toast.warning("请先输入准确的 Skill 名称");
                              return;
                            }
                            setSkillInstallConfirm({ open: true, skillName: skillSearch.trim() });
                          }}
                        >
                          安装技能
                        </Button>
                      </span>
                    </TooltipTrigger>
                    {hasQueueing && (
                      <TooltipContent side="top" className="text-xs max-w-[220px] text-justify">
                        {isLocalAgent
                          ? hasDistributingSkills
                            ? "本地发起对话，资源自动同步"
                            : "当前有技能待下发，请等待进入下发中状态。"
                          : "当前有技能正在安装队列中，请等待安装完成后再添加新技能，以免影响 Agent 的正常运行。"}
                      </TooltipContent>
                    )}
                  </Tooltip>
                );
              })()}
            </div>
            {/* Lower: two scrollable sections */}
            <div className="px-5 pb-5 flex flex-col flex-1 min-h-0 gap-3">

              {/* 已安装技能 - scrollable */}
              <div className="flex flex-col flex-1 min-h-0 pt-2 border-t border-[var(--border)]">
                <p className="text-xs text-[var(--text-muted)] mb-2 flex-shrink-0">已安装技能（{skillSearch ? filteredSkills.length : installedSkills.length}）</p>
                <div className="overflow-y-auto flex-1 space-y-1">
                  {(skillSearch ? filteredSkills : installedSkills).map((skill) => {
                    const { latestVersion, hasUpdate } = getSkillUpdateInfo(skill);
                    return (
                      <div key={skill}
                        className="group flex items-center justify-between gap-2 px-3 py-2 rounded-[4px] hover:bg-[var(--accent)] transition-colors">
                        <span className="text-sm text-[var(--text-emphasis)] truncate">{skill}</span>
                        <div className="flex items-center gap-0.5 flex-shrink-0 -mr-1.5">
                          {hasUpdate && (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <button
                                  type="button"
                                  aria-label="更新技能"
                                  onClick={() => handleUpdateSkill(skill)}
                                  className="w-7 h-7 inline-flex items-center justify-center rounded-[4px] text-[var(--text-brand)] hover:bg-[var(--accent)] transition-colors"
                                >
                                  <RefreshCw className="w-3.5 h-3.5" />
                                </button>
                              </TooltipTrigger>
                              <TooltipContent side="top" className="text-xs">
                                更新至 {latestVersion}
                              </TooltipContent>
                            </Tooltip>
                          )}
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <button
                                type="button"
                                aria-label="卸载技能"
                                onClick={() => setSkillUninstallConfirm({ open: true, entry: skill })}
                                className="w-7 h-7 inline-flex items-center justify-center rounded-[4px] text-[var(--text-muted)] hover:text-[var(--text-danger)] hover:bg-[var(--accent)] transition-colors"
                              >
                                <X className="w-3.5 h-3.5" />
                              </button>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="text-xs">
                              卸载
                            </TooltipContent>
                          </Tooltip>
                        </div>
                      </div>
                    );
                  })}
                  {skillSearch && filteredSkills.length === 0 && (
                    <p className="text-xs text-[var(--text-muted)] text-center py-4">未找到相关技能</p>
                  )}
                </div>
              </div>

              {isLocalAgent && (
                <div className="flex flex-col flex-none pt-2 border-t border-[var(--border)]">
                  <p className="text-xs text-[var(--text-muted)] mb-2 flex-shrink-0">已安装规范（{installedStandards.length}）</p>
                  {installedStandards.length > 0 ? (
                    <div className="max-h-32 overflow-y-auto space-y-1">
                      {installedStandards.map((standard) => (
                        <div
                          key={standard}
                          className="flex items-center px-3 py-2 rounded-[4px] hover:bg-[var(--accent)] transition-colors"
                        >
                          <span className="text-sm text-[var(--text-emphasis)]">{standard}</span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-xs text-[var(--text-muted)] text-center py-3">暂无已安装规范</p>
                  )}
                </div>
              )}

              {/* 待安装技能 - scrollable */}
              {pendingSkills.length > 0 && (
                <div className="flex flex-col flex-1 min-h-0 pt-2 border-t border-[var(--border)]">
                  {isLocalAgent && pendingSkills.some(s => s.status === "installing") && (
                    <Alert variant="info" className="mb-2 py-2">
                      <AlertInfoIcon />
                      <AlertDescription className="text-xs leading-5">
                        本地发起对话，资源自动同步
                      </AlertDescription>
                    </Alert>
                  )}
                  <div className="flex items-center gap-1 mb-2 flex-shrink-0">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="cursor-default flex items-center">
                          <Info className="w-3 h-3 text-[var(--text-weak)] hover:text-[var(--text-muted)] transition-colors" />
                        </span>
                      </TooltipTrigger>
                      <TooltipContent side="top" className="text-xs max-w-[220px] text-justify">
                        {isLocalAgent
                          ? "待下发技能通常为管理员为您预配置的初始技能。进入本地 Agent 发起一次对话后，插件会拉取并完成下发。"
                          : "待安装技能通常为管理员为您预配置的初始技能，安装过程不影响正常对话。只要模型与通道配置完毕，即可随时开始与 Agent 对话。"}
                      </TooltipContent>
                    </Tooltip>
                    <p className="text-xs text-[var(--text-muted)]">{isLocalAgent ? "待下发技能" : "待安装技能"}（{pendingSkills.length}）</p>
                    {pendingSkills.some(s => s.status === "failed") && (
                      <div className="ml-auto flex items-center gap-2">
                        <Button
                          variant="tenant-primary"
                          size="claw-sm"
                          onClick={handleRetryAllFailed}
                          className="h-7 px-3 text-xs gap-1 [&_svg]:size-3"
                        >
                          <RotateCcw />
                          重试
                        </Button>
                        <Button
                          variant="tenant-primary"
                          size="claw-sm"
                          onClick={handleDeleteAllFailed}
                          className="h-7 px-3 text-xs gap-1 [&_svg]:size-3"
                        >
                          <Trash2 />
                          删除
                        </Button>
                      </div>
                    )}
                  </div>
                  <div className="overflow-y-auto flex-1 space-y-1">
                    {pendingSkills.map((skill) => (
                      <div key={skill.id}
                        className="flex items-center justify-between px-3 py-2 rounded-[4px] hover:bg-[var(--accent)] transition-colors">
                        <span className="text-sm text-[var(--text-emphasis)] truncate flex-1 mr-2">{skill.name}</span>
                        <div className="flex items-center gap-1.5 flex-shrink-0">
                          {skill.status === "installing" && (
                            <>
                              <Loader2 className="w-3.5 h-3.5 text-[var(--text-brand)] animate-spin" />
                              <span className="text-xs text-[var(--text-brand)]">{isLocalAgent ? "下发中" : "安装中"}</span>
                            </>
                          )}
                          {skill.status === "pending" && (
                            <span className="text-xs text-[var(--text-muted)]">{isLocalAgent ? "待下发" : "待安装"}</span>
                          )}
                          {skill.status === "failed" && (
                            <>
                              <XCircle className="w-3.5 h-3.5 text-[var(--text-danger)]" />
                              <span className="text-xs text-[var(--text-danger)]">{isLocalAgent ? "下发失败" : "安装失败"}</span>
                            </>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

            </div>
          </TenantCard>

            </div>
          )}{/* end basic tab */}

          {/* 工具管理 tab */}
          {activeDetailTab === "tools" && (
            <ToolsMcpPanel />
          )}{/* end tools tab */}

          {/* 记忆管理 tab */}
          {activeDetailTab === "memory" && (
            (claw as any).agentType && (claw as any).agentType !== "openclaw" ? (
              <Empty className="border-0 py-20 [box-shadow:var(--shadow-card)] !rounded-[var(--radius-card)] bg-[var(--card)] border border-[var(--border)]">
                <EmptyHeader>
                  <EmptyMedia />
                  <EmptyDescription>当前 Agent 暂不支持此功能，敬请期待</EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : (
            <TenantCard padding="none" state="static" className="p-6 [box-shadow:var(--shadow-card)]">
              <MemoryPreview
                memoryStatus={memoryStatus}
                proQuotaAvailable={proQuotaAvailable}
                showConfidence={false}
                isLoading={memoryLoading}
                onStatusChange={async (newStatus) => {
                  // TODO: 调用后端接口切换 Memory 状态
                  // await api.changeMemoryStatus(clawId, newStatus);
                  await new Promise(resolve => setTimeout(resolve, 1000)); // mock 延迟
                  setMemoryStatus(newStatus);
                }}
              />
            </TenantCard>
            )
          )}

          {/* 网盘管理 tab */}
          {activeDetailTab === "files" && (
            <FileSpace
              clawName={clawName}
              clawId={clawId || ""}
              basePath="https://smh3jsttekkpsoqw.api.tencentsmh.cn"
              libraryId="smh3jsttekkpsoqw"
              spaceId="space232t1yug3w7up"
              getAccessToken={async () => ({
                accessToken: "<your-access-token>",
                expiresAt: Date.now() + 3600 * 24 * 1000,
              })}
            />
          )}

          {/* 龙虾医院 tab
              「一键修复」「龙虾医生」两块功能仅对 OpenClaw 类型 Agent 启用：
                ① 业务原因：这两块底层都依赖 OpenClaw 配置文件结构（agent doctor --fix / 配置快照 / 通道修复）
                   和 OpenClaw 实例化机制，Hermes / LightClawACE 的运行时与配置体系不同，无法适配；
                ② 判定约定：与本页其他 OpenClaw 专属 tab（如 memory，2231-2254 行）保持一致——
                   `agentType` 未设置时默认为 OpenClaw（历史兼容），显式为 "openclaw" 时也启用，
                   其余值（hermes / lightclawace）一律隐藏；
                ③ 空态体验：完全复用 memory tab 的"暂不支持"卡片样式，让用户在不同 tab 看到的
                   "类型限制"反馈视觉一致，不显得是为这一块单独糊的提示。 */}
          {activeDetailTab === "doctor" && (
            (claw as any).agentType && (claw as any).agentType !== "openclaw" ? (
              <Empty className="border-0 py-20 [box-shadow:var(--shadow-card)] !rounded-[var(--radius-card)] bg-[var(--card)] border border-[var(--border)]">
                <EmptyHeader>
                  <EmptyMedia />
                  <EmptyDescription>当前 Agent 暂不支持此功能，敬请期待</EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : (
            <div className="flex flex-col gap-5">

              {/* ===== 一键修复卡片 ===== */}
              <TenantCard padding="none" state="static" className="[box-shadow:var(--shadow-card)]">
                <div className="p-6">
                  <h2 className="text-base font-semibold text-[var(--text-strong)] mb-2">一键修复</h2>
                  <p className="text-sm text-[var(--text-muted)] mb-4">适合龙虾配置文件中 API KEY、插件、通道等配置异常导致无法启动等常见问题，系统自动检测并尝试修复。</p>
                  <ul className="space-y-2 mb-6">
                    <li className="flex items-center gap-2 text-sm text-[var(--text-emphasis)]">
                      自动执行
                      <code className="px-2 py-0.5 rounded bg-[var(--accent)] text-[var(--text-emphasis)] font-mono text-xs">agent doctor --fix</code>
                    </li>
                    <li className="flex items-center gap-2 text-sm text-[var(--text-emphasis)]">
                      自动恢复常见配置问题
                    </li>
                    <li className="flex items-center gap-2 text-sm text-[var(--text-emphasis)]">
                      恢复前会将配置文件备份
                    </li>
                  </ul>
                  <div className="border-t border-[var(--border)] pt-4">
                    {quickFixState === "idle" && (
                      <Button
                        variant="tenant-outline-strong"
                        size="claw-lg"
                        onClick={runQuickFixMock}
                      >
                        <Wrench className="w-4 h-4" />
                        <span>一键修复</span>
                      </Button>
                    )}
                    {quickFixState === "loading" && (
                      <div className="inline-flex items-center gap-2 px-3 h-8 rounded-[4px] bg-[var(--accent)] border border-[var(--border)] text-xs text-[var(--text-muted)]">
                        <span className="w-3 h-3 border-2 border-[var(--border)] border-t-gray-600 rounded-full animate-spin" />
                        正在执行修复
                      </div>
                    )}
                    {quickFixState === "success" && (
                      <div className="flex items-center gap-2.5 flex-wrap">
                        <span className="inline-flex items-center gap-1.5 text-xs font-medium text-[var(--text-secondary)] bg-white border border-[var(--border)] rounded-[4px] px-3 py-1.5 leading-none">
                          <CheckCircle2 className="w-3.5 h-3.5 text-[var(--text-muted)] flex-shrink-0" />
                          修复成功
                        </span>
                        <span className="text-xs text-[var(--text-muted)]">Gateway 已正常启动，请前往 Agent 对话确认问题是否已解决</span>
                      </div>
                    )}
                    {quickFixState === "failed" && (
                      <div className="flex items-center gap-2.5 flex-wrap">
                        <span className="inline-flex items-center gap-1.5 text-xs font-medium text-[var(--text-secondary)] bg-white border border-[var(--border)] rounded-[4px] px-3 py-1.5 leading-none">
                          <AlertCircle className="w-3.5 h-3.5 text-[var(--text-danger)] flex-shrink-0" />
                          修复失败
                        </span>
                        <span className="text-xs text-[var(--text-muted)]">{quickFixFailReason}，建议开启龙虾医生进行深度诊断</span>
                      </div>
                    )}
                  </div>
                </div>
              </TenantCard>

              {/* ===== 龙虾医生对话卡片（受管控端「允许用户使用龙虾医生」开关控制） ===== */}
              {lobsterDoctorEnabled && (
                <div data-doctor-chat-card>
                  <DoctorChatCard instanceId={claw.instanceId} instanceName={claw.name || claw.instanceId} />
                </div>
              )}

            </div>
            )
          )}

          </div>{/* end tab content */}
        </Segment>{/* end segment */}
          </div>{/* end 中间内容区 (flex-1 min-w-0 px-[42px] py-8) */}
          <div aria-hidden className="shrink-0 w-20 self-stretch" />
        </div>{/* end max-w-[1920px] flex */}
      </div>{/* end min-w-[1200px] overflow-x-clip */}

      {/* ===== 飞书授权弹窗（三阶段） ===== */}
      <Dialog open={showQrModal} onOpenChange={(open) => {
        // 仅在 done 阶段或 loading/qr 阶段允许关闭
        if (!open && (feishuModalStage === "done" || feishuModalStage === "loading" || feishuModalStage === "qr")) {
          setShowQrModal(false);
        } else if (!open && feishuModalStage === "configuring") {
          // 配置中不允许关闭
        } else {
          setShowQrModal(open);
        }
      }}>
        <DialogContent className="max-w-lg [&>button]:focus-visible:ring-0 [&>button]:focus-visible:ring-offset-0 [&>button]:outline-none [&>button]:shadow-none [&>button]:border-0 [&>button]:ring-0">

          {/* ── 阶段1&2：loading + qr ── */}
          {(feishuModalStage === "loading" || feishuModalStage === "qr") && (
            <>
              <DialogHeader>
                <DialogTitle className="text-base font-semibold text-gray-900">扫码配置飞书机器人</DialogTitle>
                <DialogDescription className="text-sm text-gray-500 mt-1">
                  请使用飞书账号扫码登录，完成授权后将自动为您创建机器人。
                </DialogDescription>
              </DialogHeader>
              <div className="flex flex-col items-center justify-center bg-gray-50 rounded-[4px] min-h-[240px] mt-1 mb-2">
                {feishuModalStage === "loading" ? (
                  <>
                    <Loader2 className="w-12 h-12 text-gray-300 animate-spin mb-4" />
                    <p className="text-sm text-gray-500">正在生成二维码...</p>
                  </>
                ) : (
                  <svg width="180" height="180" viewBox="0 0 180 180" xmlns="http://www.w3.org/2000/svg">
                    <rect width="180" height="180" fill="white"/>
                    <rect x="10" y="10" width="50" height="50" fill="black"/>
                    <rect x="18" y="18" width="34" height="34" fill="white"/>
                    <rect x="26" y="26" width="18" height="18" fill="black"/>
                    <rect x="120" y="10" width="50" height="50" fill="black"/>
                    <rect x="128" y="18" width="34" height="34" fill="white"/>
                    <rect x="136" y="26" width="18" height="18" fill="black"/>
                    <rect x="10" y="120" width="50" height="50" fill="black"/>
                    <rect x="18" y="128" width="34" height="34" fill="white"/>
                    <rect x="26" y="136" width="18" height="18" fill="black"/>
                    <rect x="70" y="10" width="8" height="8" fill="black"/>
                    <rect x="82" y="10" width="8" height="8" fill="black"/>
                    <rect x="94" y="10" width="8" height="8" fill="black"/>
                    <rect x="106" y="10" width="8" height="8" fill="black"/>
                    <rect x="70" y="22" width="8" height="8" fill="black"/>
                    <rect x="94" y="22" width="8" height="8" fill="black"/>
                    <rect x="70" y="34" width="8" height="8" fill="black"/>
                    <rect x="82" y="34" width="8" height="8" fill="black"/>
                    <rect x="106" y="34" width="8" height="8" fill="black"/>
                    <rect x="70" y="46" width="8" height="8" fill="black"/>
                    <rect x="94" y="46" width="8" height="8" fill="black"/>
                    <rect x="70" y="58" width="8" height="8" fill="black"/>
                    <rect x="82" y="58" width="8" height="8" fill="black"/>
                    <rect x="94" y="58" width="8" height="8" fill="black"/>
                    <rect x="106" y="58" width="8" height="8" fill="black"/>
                    <rect x="10" y="70" width="8" height="8" fill="black"/>
                    <rect x="22" y="70" width="8" height="8" fill="black"/>
                    <rect x="46" y="70" width="8" height="8" fill="black"/>
                    <rect x="58" y="70" width="8" height="8" fill="black"/>
                    <rect x="70" y="70" width="8" height="8" fill="black"/>
                    <rect x="94" y="70" width="8" height="8" fill="black"/>
                    <rect x="118" y="70" width="8" height="8" fill="black"/>
                    <rect x="130" y="70" width="8" height="8" fill="black"/>
                    <rect x="154" y="70" width="8" height="8" fill="black"/>
                    <rect x="166" y="70" width="8" height="8" fill="black"/>
                    <rect x="10" y="82" width="8" height="8" fill="black"/>
                    <rect x="34" y="82" width="8" height="8" fill="black"/>
                    <rect x="58" y="82" width="8" height="8" fill="black"/>
                    <rect x="82" y="82" width="8" height="8" fill="black"/>
                    <rect x="106" y="82" width="8" height="8" fill="black"/>
                    <rect x="130" y="82" width="8" height="8" fill="black"/>
                    <rect x="154" y="82" width="8" height="8" fill="black"/>
                    <rect x="10" y="94" width="8" height="8" fill="black"/>
                    <rect x="22" y="94" width="8" height="8" fill="black"/>
                    <rect x="46" y="94" width="8" height="8" fill="black"/>
                    <rect x="70" y="94" width="8" height="8" fill="black"/>
                    <rect x="94" y="94" width="8" height="8" fill="black"/>
                    <rect x="118" y="94" width="8" height="8" fill="black"/>
                    <rect x="142" y="94" width="8" height="8" fill="black"/>
                    <rect x="166" y="94" width="8" height="8" fill="black"/>
                    <rect x="10" y="106" width="8" height="8" fill="black"/>
                    <rect x="34" y="106" width="8" height="8" fill="black"/>
                    <rect x="58" y="106" width="8" height="8" fill="black"/>
                    <rect x="82" y="106" width="8" height="8" fill="black"/>
                    <rect x="106" y="106" width="8" height="8" fill="black"/>
                    <rect x="130" y="106" width="8" height="8" fill="black"/>
                    <rect x="154" y="106" width="8" height="8" fill="black"/>
                    <rect x="70" y="118" width="8" height="8" fill="black"/>
                    <rect x="82" y="118" width="8" height="8" fill="black"/>
                    <rect x="106" y="118" width="8" height="8" fill="black"/>
                    <rect x="118" y="118" width="8" height="8" fill="black"/>
                    <rect x="142" y="118" width="8" height="8" fill="black"/>
                    <rect x="166" y="118" width="8" height="8" fill="black"/>
                    <rect x="70" y="130" width="8" height="8" fill="black"/>
                    <rect x="94" y="130" width="8" height="8" fill="black"/>
                    <rect x="118" y="130" width="8" height="8" fill="black"/>
                    <rect x="130" y="130" width="8" height="8" fill="black"/>
                    <rect x="154" y="130" width="8" height="8" fill="black"/>
                    <rect x="70" y="142" width="8" height="8" fill="black"/>
                    <rect x="82" y="142" width="8" height="8" fill="black"/>
                    <rect x="94" y="142" width="8" height="8" fill="black"/>
                    <rect x="106" y="142" width="8" height="8" fill="black"/>
                    <rect x="130" y="142" width="8" height="8" fill="black"/>
                    <rect x="142" y="142" width="8" height="8" fill="black"/>
                    <rect x="166" y="142" width="8" height="8" fill="black"/>
                    <rect x="70" y="154" width="8" height="8" fill="black"/>
                    <rect x="94" y="154" width="8" height="8" fill="black"/>
                    <rect x="118" y="154" width="8" height="8" fill="black"/>
                    <rect x="142" y="154" width="8" height="8" fill="black"/>
                    <rect x="70" y="166" width="8" height="8" fill="black"/>
                    <rect x="82" y="166" width="8" height="8" fill="black"/>
                    <rect x="106" y="166" width="8" height="8" fill="black"/>
                    <rect x="130" y="166" width="8" height="8" fill="black"/>
                    <rect x="154" y="166" width="8" height="8" fill="black"/>
                    <rect x="166" y="166" width="8" height="8" fill="black"/>
                  </svg>
                )}
              </div>
            </>
          )}

          {/* ── 阶段3：正在配置 ── */}
          {feishuModalStage === "configuring" && (
            <>
              <DialogHeader>
                <DialogTitle className="text-base font-semibold text-gray-900">正在配置飞书机器人</DialogTitle>
              </DialogHeader>
              <div className="mt-1 space-y-2.5 py-1 pb-3">
                {feishuSteps.map((step, idx) => {
                  const stepNum = idx + 1;
                  const isDone = feishuStepsDone >= stepNum;
                  const isActive = feishuStepsDone === idx;
                  const isHighPrivilege = idx === feishuHighPrivilegeStepIdx;
                  return (
                    <div key={step} className="flex items-center gap-3">
                      {isDone && isHighPrivilege ? (
                        <AlertCircle className="w-5 h-5 text-orange-500 shrink-0" />
                      ) : isDone ? (
                        <CheckCircle2 className="w-5 h-5 text-green-500 shrink-0" />
                      ) : isActive ? (
                        <Loader2 className="w-5 h-5 text-blue-500 animate-spin shrink-0" />
                      ) : (
                        <div className="w-5 h-5 rounded-full border-2 border-gray-200 shrink-0" />
                      )}
                      <span className={`text-xs ${
                        isDone && isHighPrivilege ? "text-orange-500 font-medium" :
                        isDone ? "text-gray-600" : isActive ? "text-blue-600 font-medium" : "text-gray-400"
                      }`}>
                        [步骤{stepNum}] {step}
                      </span>
                    </div>
                  );
                })}
              </div>
            </>
          )}

          {/* ── 阶段4b：配置失败 ── */}
          {feishuModalStage === "failed" && (
            <>
              <DialogHeader>
                <div className="flex items-center gap-3 mb-1">
                  <CircleAlert className="w-5 h-5 text-red-500 shrink-0" />
                  <DialogTitle className="text-base font-semibold text-gray-900">飞书机器人发布失败</DialogTitle>
                </div>
                <DialogDescription className="text-sm text-red-500 mt-1 font-medium">
                  当前用户权限无法免审批发布飞书机器人，请联系管理员审批通过后再进行手动配置。
                </DialogDescription>
              </DialogHeader>
              <div className="mt-3 space-y-1.5 text-sm bg-gray-50 rounded-[4px] p-3 border border-gray-200">
                <div className="flex items-center gap-2">
                  <span className="text-gray-500 shrink-0">机器人名称：</span>
                  <span className="text-gray-800 font-medium">Agent机器人-8791</span>
                </div>
                <div className="flex items-start gap-2">
                  <span className="text-gray-500 shrink-0">管理地址：</span>
                  <a
                    href="https://open.feishu.cn/app/<your-feishu-app-id>"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-500 hover:underline break-all"
                  >
                    https://open.feishu.cn/app/&lt;your-feishu-app-id&gt;
                  </a>
                </div>
                <div className="flex items-start gap-2">
                  <span className="text-gray-500 shrink-0">管理员审批地址：</span>
                  <a
                    href="https://feishu.cn/admin/appCenter/audit"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-500 hover:underline break-all"
                  >
                    https://feishu.cn/admin/appCenter/audit
                  </a>
                </div>
              </div>
              <div className="mt-5 flex justify-center">
                <Button
                  onClick={() => setShowQrModal(false)}
                  className="px-8"
                >
                  完成
                </Button>
              </div>
            </>
          )}

          {/* ── 阶段4：配置完成 ── */}
          {feishuModalStage === "done" && (
            <>
              <DialogHeader>
                <div className="flex items-center gap-3 mb-1">
                  <CheckCircle2 className="w-8 h-8 text-green-500 shrink-0" />
                  <DialogTitle className="text-base font-semibold text-gray-900">飞书机器人授权配置成功</DialogTitle>
                </div>
              </DialogHeader>
              <div className="mt-3 space-y-1.5 text-sm bg-gray-50 rounded-[4px] p-3 border border-gray-200">
                <div className="flex items-center gap-2">
                  <span className="text-gray-500 shrink-0">机器人名称：</span>
                  <span className="text-gray-800 font-medium">Agent机器人-4598</span>
                </div>
                <div className="flex items-start gap-2">
                  <span className="text-gray-500 shrink-0">管理地址：</span>
                  <a
                    href="https://open.feishu.cn/app/<your-feishu-app-id>"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-500 hover:underline break-all"
                  >
                    https://open.feishu.cn/app/&lt;your-feishu-app-id&gt;
                  </a>
                </div>
              </div>
              {/* 审批提示 */}
              <div className="mt-4 p-3 bg-orange-50 rounded-[4px] border border-orange-100">
                <div className="flex items-start gap-2">
                  <CircleAlert className="w-4 h-4 text-orange-500 shrink-0 mt-0.5" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-orange-600 font-medium mb-2">以下高级权限无法免审批发布，已自动为您提交申请：</p>
                    <ol className="text-sm text-orange-600 ml-4 space-y-1 list-decimal">
                      <li>查看、评论和下载云空间中所有文件</li>
                      <li>查看、评论、编辑和管理云空间中所有文件</li>
                    </ol>
                    <div className="mt-2 space-y-0.5">
                      <p className="text-sm text-orange-600">如需启用，请联系管理员前往审批：</p>
                      <a
                        href="https://feishu.cn/admin/appCenter/audit"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sm text-blue-500 hover:underline block"
                      >
                        https://feishu.cn/admin/appCenter/audit
                      </a>
                    </div>
                  </div>
                </div>
              </div>
              <div className="mt-5 flex justify-center">
                <Button
                  onClick={() => {
                    setShowQrModal(false);
                    // 飞书通道唯一性：有则更新，无则新增
                    const feishuConfig = CHANNEL_OPTIONS.find(c => c.value === "feishu");
                    if (feishuConfig) {
                      setAppliedChannels(prev => {
                        const existingIdx = prev.findIndex(c => c.channelValue === "feishu");
                        const updatedEntry: AppliedChannel = {
                          type: "飞书",
                          channelValue: "feishu",
                          status: "running",
                          fields: feishuConfig.fields || [],
                          fieldValues: { appId: "<your-feishu-app-id>", appSecret: "auto-authorized" },
                          feishuConfigMode: "quick",
                        };
                        if (existingIdx >= 0) {
                          const next = [...prev];
                          next[existingIdx] = updatedEntry;
                          return next;
                        }
                        return [...prev, updatedEntry];
                      });
                    }
                    toast.success("飞书机器人已添加并应用");
                  }}
                  className="px-6"
                >
                  完成
                </Button>
              </div>
            </>
          )}

        </DialogContent>
      </Dialog>

      {/* ===== Hermes 开启面板 等待弹窗 ===== */}
      <Dialog open={showHermesPanelDialog} onOpenChange={(open) => {
        if (!open) setShowHermesPanelDialog(false);
      }}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-gray-900">开启 Agent 面板</DialogTitle>
            <DialogDescription className="sr-only">开启 Agent 面板</DialogDescription>
            <div className="mt-2 flex items-start gap-2.5 rounded-[4px] border border-blue-200 bg-blue-50 px-3.5 py-3 text-xs text-blue-700 leading-relaxed">
              <svg xmlns="http://www.w3.org/2000/svg" className="w-4 h-4 mt-0.5 shrink-0 text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
              <span>Hermes Agent 面板是官方提供的浏览器操作界面，加载完成后将自动跳转，请稍候等待。</span>
            </div>
          </DialogHeader>
          <div className="mt-3 space-y-2.5 py-1 pb-3">
            {/* 步骤1：连接服务 */}
            <div className="flex items-center gap-3">
              {hermesPanelStep >= 1 ? (
                <CheckCircle2 className="w-5 h-5 text-green-500 shrink-0" />
              ) : (
                <Loader2 className="w-5 h-5 text-blue-500 animate-spin shrink-0" />
              )}
              <span className={`text-xs ${hermesPanelStep >= 1 ? "text-gray-600" : "text-blue-600 font-medium"}`}>
                {hermesPanelStep >= 1 ? "连接服务：连接成功" : "连接服务：正在连接 Hermes Agent 服务，预计1~2秒..."}
              </span>
            </div>
            {/* 步骤2：加载面板 */}
            <div className="flex items-center gap-3">
              {hermesPanelStep >= 2 ? (
                <CheckCircle2 className="w-5 h-5 text-green-500 shrink-0" />
              ) : hermesPanelStep === 1 ? (
                <Loader2 className="w-5 h-5 text-blue-500 animate-spin shrink-0" />
              ) : (
                <div className="w-5 h-5 rounded-full border-2 border-gray-200 shrink-0" />
              )}
              <span className={`text-xs ${
                hermesPanelStep >= 2 ? "text-gray-600" :
                hermesPanelStep === 1 ? "text-blue-600 font-medium" : "text-gray-400"
              }`}>
                {hermesPanelStep >= 2
                  ? "加载面板：加载完成，正在跳转..."
                  : hermesPanelStep === 1
                  ? "加载面板：正在加载 Hermes Agent 面板，预计3~5秒..."
                  : "加载面板：等待连接完成"}
              </span>
            </div>
          </div>
          <div className="flex justify-center pt-1">
            <Button
              variant="tenant-outline"
              size="claw-sm"
              onClick={() => setShowHermesPanelDialog(false)}
              className="px-6"
            >
              取消
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* ===== Agent 面板 进度弹窗 ===== */}
      <Dialog open={showWebUIProgressDialog} onOpenChange={(open) => {
        if (!open) setShowWebUIProgressDialog(false);
      }}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-gray-900">开启Agent面板</DialogTitle>
            <DialogDescription className="sr-only">开启Agent面板</DialogDescription>
            <div className="mt-2 flex items-start gap-2.5 rounded-[4px] border border-blue-200 bg-blue-50 px-3.5 py-3 text-xs text-blue-700 leading-relaxed">
              <svg xmlns="http://www.w3.org/2000/svg" className="w-4 h-4 mt-0.5 shrink-0 text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
              <span>Agent 面板（WebUI）是官方提供的浏览器操作界面，可直接在浏览器与 AI 对话，并且有查看会话记录、配置定时任务、监控系统日志等高级功能。</span>
            </div>
            <p className="text-sm text-gray-500 mt-3">开启Agent面板将会依次执行以下操作，确定后将自动执行：</p>
          </DialogHeader>
          <div className="mt-1 space-y-2.5 py-1 pb-3">
            {/* 步骤1：放通端口 */}
            <div className="flex items-center gap-3">
              {webUIFailedStep === "port" ? (
                <AlertCircle className="w-5 h-5 text-orange-500 shrink-0" />
              ) : webUIStep >= 1 ? (
                <CheckCircle2 className="w-5 h-5 text-green-500 shrink-0" />
              ) : (
                <Loader2 className="w-5 h-5 text-blue-500 animate-spin shrink-0" />
              )}
              <span className={`text-xs ${
                webUIFailedStep === "port" ? "text-orange-500 font-medium" :
                webUIStep >= 1 ? "text-gray-600" : "text-blue-600 font-medium"
              }`}>
                {webUIFailedStep === "port"
                  ? "放通端口：放通端口失败，请重试"
                  : webUIStep >= 1
                  ? "放通端口：端口38341已放通"
                  : "放通端口：正在放通端口38341...预计1~2秒"}
              </span>
            </div>
            {/* 步骤2：生成链接 */}
            <div className="flex items-center gap-3">
              {webUIFailedStep === "link" ? (
                <AlertCircle className="w-5 h-5 text-orange-500 shrink-0" />
              ) : webUIStep >= 2 ? (
                <CheckCircle2 className="w-5 h-5 text-green-500 shrink-0" />
              ) : webUIStep === 1 ? (
                <Loader2 className="w-5 h-5 text-blue-500 animate-spin shrink-0" />
              ) : (
                <div className="w-5 h-5 rounded-full border-2 border-gray-200 shrink-0" />
              )}
              <span className={`text-xs ${
                webUIFailedStep === "link" ? "text-orange-500 font-medium" :
                webUIStep >= 2 ? "text-gray-600" :
                webUIStep === 1 ? "text-blue-600 font-medium" : "text-gray-400"
              }`}>
                {webUIFailedStep === "link"
                  ? "生成链接：生成链接失败，请重试"
                  : webUIStep >= 2
                  ? "生成链接：链接已生成"
                  : webUIStep === 1
                  ? "生成链接：正在为您生成Agent面板访问链接，预计5~10秒..."
                  : "生成链接：等待放通端口完成"}
              </span>
            </div>
          </div>
          <div className="flex justify-center gap-3 pt-1">
            <Button
              variant="tenant-outline"
              size="claw-sm"
              onClick={() => setShowWebUIProgressDialog(false)}
              className="px-6"
            >
              取消
            </Button>
            <Button
              size="sm"
              disabled={webUIStep < 2 && webUIFailedStep === "none"}
              onClick={webUIFailedStep !== "none" ? handleWebUIRetry : handleWebUIProgressConfirm}
              className={`px-6 ${(webUIStep >= 2 || webUIFailedStep !== "none") ? '' : 'opacity-50 cursor-not-allowed'}`}
            >
              {webUIFailedStep !== "none" ? "重试" : "确定"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* ===== Agent 面板 结果弹窗 ===== */}
      <Dialog open={showWebUIResultDialog} onOpenChange={(open) => {
        if (!open) setShowWebUIResultDialog(false);
      }}>
        <DialogContent className="w-[90vw] max-w-lg overflow-hidden">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-gray-900">开启Agent面板</DialogTitle>
          </DialogHeader>
          {/* 警告文字 */}
          <div className="text-sm text-orange-600 font-medium bg-orange-50 border border-orange-100 rounded-[4px] px-3 py-2.5 leading-relaxed break-all">
            访问链接已生成，该链接含有您的 API Key 和加密配置，请勿分享给第三方，以防隐私泄露或资产损失。
          </div>
          {/* 链接和 Token - 根据 agentType 区分字段文案 */}
          {(claw as any).agentType === "lightclawace" ? (
            /* LightclawACE：面板链接 + 密码（初始密码提示） */
            <div className="mt-2 space-y-2 bg-gray-50 rounded-[4px] border border-gray-200 px-4 py-3 w-full overflow-hidden">
              <div className="flex items-center gap-2 w-full min-w-0">
                <span className="text-xs text-gray-500 shrink-0 w-16">面板链接</span>
                <span className="text-xs text-gray-700 flex-1 truncate font-mono min-w-0">{webUIUrl}</span>
                <button
                  onClick={() => { navigator.clipboard.writeText(webUIUrl); toast.success("已复制链接"); }}
                  className="shrink-0 p-1 hover:bg-gray-200 rounded transition-colors"
                >
                  <svg className="w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                </button>
              </div>
              <div className="flex items-center gap-2 w-full min-w-0">
                <div className="flex items-center gap-1 shrink-0 w-auto">
                  <span className="text-xs text-gray-500 shrink-0">初始密码</span>
                  <TooltipProvider delayDuration={100}>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <svg xmlns="http://www.w3.org/2000/svg" className="w-3.5 h-3.5 text-gray-400 cursor-pointer hover:text-gray-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                          <circle cx="12" cy="12" r="10"/>
                          <line x1="12" y1="16" x2="12" y2="12"/>
                          <line x1="12" y1="8" x2="12.01" y2="8"/>
                        </svg>
                      </TooltipTrigger>
                      <TooltipContent side="top" className="text-xs max-w-[200px] text-center">
                        若已在面板内修改了密码，请使用新密码登录
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                </div>
                <span className="text-xs text-gray-700 flex-1 truncate font-mono min-w-0 ml-1">{webUIToken}</span>
                <button
                  onClick={() => { navigator.clipboard.writeText(webUIToken); toast.success("已复制密码"); }}
                  className="shrink-0 p-1 hover:bg-gray-200 rounded transition-colors"
                >
                  <svg className="w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                </button>
              </div>
            </div>
          ) : (
            /* OpenClaw： WebSocket URL + 网关令牌（保持原样） */
            <div className="mt-2 space-y-2 bg-gray-50 rounded-[4px] border border-gray-200 px-4 py-3 w-full overflow-hidden">
              <div className="flex items-center gap-2 w-full min-w-0">
                <span className="text-xs text-gray-500 shrink-0 w-16">WebSocket URL</span>
                <span className="text-xs text-gray-700 flex-1 truncate font-mono min-w-0">{webUIUrl}</span>
                <button
                  onClick={() => { navigator.clipboard.writeText(webUIUrl); toast.success("已复制链接"); }}
                  className="shrink-0 p-1 hover:bg-gray-200 rounded transition-colors"
                >
                  <svg className="w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                </button>
              </div>
              <div className="flex items-center gap-2 w-full min-w-0">
                <span className="text-xs text-gray-500 shrink-0 w-16">网关令牌</span>
                <span className="text-xs text-gray-700 flex-1 truncate font-mono min-w-0">{webUIToken}</span>
                <button
                  onClick={() => { navigator.clipboard.writeText(webUIToken); toast.success("已复制Token"); }}
                  className="shrink-0 p-1 hover:bg-gray-200 rounded transition-colors"
                >
                  <svg className="w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                </button>
              </div>
            </div>
          )}
          {/* 提示文字 - 根据类型区分 */}
          <p className="text-xs text-gray-500 mt-1">
            {(claw as any).agentType === "lightclawace"
              ? "用浏览器打开面板链接，如面板需要填入密码，则将密码复制并粘贴过去，即可进入面板。"
              : "用浏览器打开 WebSocket URL，如面板需要填入网关令牌，则将网关令牌复制并粘贴过去，即可进入面板。"}
          </p>
          <div className="flex justify-center pt-1">
            <Button
              size="sm"
              onClick={() => { window.open(webUIUrl, "_blank"); }}
              className="px-8"
            >
              立即访问
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* ===== 一键更新 确认弹窗 ===== */}
      {/* 连通性检测失败弹窗 */}
      <Dialog open={!!connectFailResult} onOpenChange={() => setConnectFailResult(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-red-500">
              <CircleAlert className="w-5 h-5" />
              模型连接失败
            </DialogTitle>
            <DialogDescription className="sr-only">模型连接失败详情</DialogDescription>
          </DialogHeader>
          <div className="py-2">
            <pre className="rounded-lg bg-gray-50 border border-gray-200 p-3 text-xs text-gray-700 font-mono whitespace-pre-wrap break-all">
              {connectFailResult}
            </pre>
          </div>
          <div className="flex justify-end pt-1">
            <Button
              style={{ background: "linear-gradient(135deg, #007AFF, #5856D6)" }}
              className="text-white"
              onClick={() => setConnectFailResult(null)}
            >
              我知道了
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={showUpdateConfirmDialog} onOpenChange={setShowUpdateConfirmDialog}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-gray-900">更新确认</DialogTitle>
            <DialogDescription className="sr-only">更新确认</DialogDescription>
          </DialogHeader>
          <div className="text-sm text-gray-700 leading-relaxed space-y-3 py-1">
            {/* 当前版本 vs 目标版本 */}
            <div className="rounded-lg border border-[var(--border)] p-3 space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-[var(--text-muted)]">当前 Agent 版本</span>
                <span className="font-mono text-[var(--text-emphasis)] font-medium">v{currentVersion}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-[var(--text-muted)]">目标 Agent 版本</span>
                <span className="font-mono text-[var(--text-brand)] font-medium">
                  {targetVersion ? `v${targetVersion}` : "暂无可更新版本"}
                </span>
              </div>
            </div>
            {/* 版本校验提示 */}
            {!targetVersion ? (
              <Alert variant="warning">
                <AlertCircle className="w-4 h-4" />
                <AlertDescription>
                  暂未找到该 Agent 类型的生效镜像版本，请联系管理员在镜像管理中配置。
                </AlertDescription>
              </Alert>
            ) : !isTargetNewer ? (
              <Alert variant="warning">
                <AlertCircle className="w-4 h-4" />
                <AlertDescription>
                  目标版本（v{targetVersion}）不比当前版本（v{currentVersion}）更新，不支持向低版本更新。
                </AlertDescription>
              </Alert>
            ) : null}
            <p>Agent版本将会更新至管理员指定生效镜像所对应的版本，且不支持跨Agent类型更新。</p>
            <p>更新版本预计需要 5～10 分钟不等，请您耐心等待。更新期间 Agent 网关服务暂停，面板不可操作。</p>
            <p>更新版本后模型（Models）、通道（Channels）、技能（Skills）和记忆均不会丢失。</p>
          </div>
          <div className="flex justify-end gap-3 pt-2">
            <Button
              variant="tenant-outline"
              size="claw-sm"
              onClick={() => setShowUpdateConfirmDialog(false)}
              className="px-5"
            >
              取消
            </Button>
            <Tooltip>
              <TooltipTrigger asChild>
                <span tabIndex={!isTargetNewer || !targetVersion ? 0 : -1}>
                  <Button
                    size="sm"
                    onClick={handleStartUpdate}
                    className="px-5"
                    disabled={!isTargetNewer || !targetVersion}
                  >
                    确认更新
                  </Button>
                </span>
              </TooltipTrigger>
              {(!targetVersion || !isTargetNewer) && (
                <TooltipContent side="bottom" className="text-xs">
                  {!targetVersion
                    ? "暂无可更新的目标版本"
                    : `目标版本 v${targetVersion} 不比当前版本 v${currentVersion} 更新`}
                </TooltipContent>
              )}
            </Tooltip>
          </div>
        </DialogContent>
      </Dialog>

      {/* ===== 一键更新 进度弹窗 ===== */}
      <Dialog open={showUpdateProgressDialog} onOpenChange={(open) => {
        if (!open) setShowUpdateProgressDialog(false);
      }}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-gray-900">正在更新 Agent</DialogTitle>
            <DialogDescription className="sr-only">更新进度</DialogDescription>
          </DialogHeader>
          <div className="mt-1 space-y-2.5 py-1 pb-3">
            {updateSteps.map((step, idx) => {
              const stepNum = idx + 1;
              const isDone = updateStepsDone >= stepNum;
              const isActive = updateStepsDone === idx;
              return (
                <div key={step} className="flex items-center gap-3">
                  {isDone ? (
                    <CheckCircle2 className="w-5 h-5 text-green-500 shrink-0" />
                  ) : isActive ? (
                    <Loader2 className="w-5 h-5 text-blue-500 animate-spin shrink-0" />
                  ) : (
                    <div className="w-5 h-5 rounded-full border-2 border-gray-200 shrink-0" />
                  )}
                  <span className={`text-xs ${
                    isDone ? "text-gray-600" : isActive ? "text-blue-600 font-medium" : "text-gray-400"
                  }`}>
                    [步骤{stepNum}] {step}
                  </span>
                </div>
              );
            })}
          </div>
        </DialogContent>
      </Dialog>

      {/* ===== 微信扫码登录弹窗 ===== */}
      <Dialog open={showWechatQrModal} onOpenChange={(open) => {
        // 仅在 qr 阶段允许手动关闭（loading 阶段不允许）
        if (!open && wechatModalStage === "qr") setShowWechatQrModal(false);
      }}>
        <DialogContent className="max-w-sm [&>button]:focus-visible:ring-0 [&>button]:focus-visible:ring-offset-0 [&>button]:outline-none [&>button]:shadow-none [&>button]:border-0 [&>button]:ring-0">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-gray-900">微信扫码登录</DialogTitle>
            <DialogDescription className="text-sm text-gray-500 mt-1">
              使用微信（需要 iOS、Android系统 8.0.70 以上版本）"扫一扫"完成接入
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col items-center justify-center bg-gray-50 rounded-[4px] min-h-[220px] mt-1 mb-2">
            {wechatModalStage === "checking" && (
              <>
                <Loader2 className="w-10 h-10 text-gray-300 animate-spin mb-3" />
                <p className="text-sm text-gray-500">正在检查网关…</p>
              </>
            )}
            {wechatModalStage === "generating" && (
              <>
                <Loader2 className="w-10 h-10 text-gray-300 animate-spin mb-3" />
                <p className="text-sm text-gray-500">正在生成二维码…</p>
              </>
            )}
            {wechatModalStage === "qr" && (
              <svg width="180" height="180" viewBox="0 0 180 180" xmlns="http://www.w3.org/2000/svg">
                <rect width="180" height="180" fill="white"/>
                <rect x="10" y="10" width="50" height="50" fill="black"/>
                <rect x="18" y="18" width="34" height="34" fill="white"/>
                <rect x="26" y="26" width="18" height="18" fill="black"/>
                <rect x="120" y="10" width="50" height="50" fill="black"/>
                <rect x="128" y="18" width="34" height="34" fill="white"/>
                <rect x="136" y="26" width="18" height="18" fill="black"/>
                <rect x="10" y="120" width="50" height="50" fill="black"/>
                <rect x="18" y="128" width="34" height="34" fill="white"/>
                <rect x="26" y="136" width="18" height="18" fill="black"/>
                <rect x="82" y="10" width="8" height="8" fill="black"/>
                <rect x="94" y="10" width="8" height="8" fill="black"/>
                <rect x="70" y="22" width="8" height="8" fill="black"/>
                <rect x="106" y="22" width="8" height="8" fill="black"/>
                <rect x="82" y="34" width="8" height="8" fill="black"/>
                <rect x="94" y="34" width="8" height="8" fill="black"/>
                <rect x="70" y="46" width="8" height="8" fill="black"/>
                <rect x="106" y="46" width="8" height="8" fill="black"/>
                <rect x="82" y="58" width="8" height="8" fill="black"/>
                <rect x="10" y="70" width="8" height="8" fill="black"/>
                <rect x="34" y="70" width="8" height="8" fill="black"/>
                <rect x="58" y="70" width="8" height="8" fill="black"/>
                <rect x="82" y="70" width="8" height="8" fill="black"/>
                <rect x="106" y="70" width="8" height="8" fill="black"/>
                <rect x="130" y="70" width="8" height="8" fill="black"/>
                <rect x="154" y="70" width="8" height="8" fill="black"/>
                <rect x="22" y="82" width="8" height="8" fill="black"/>
                <rect x="46" y="82" width="8" height="8" fill="black"/>
                <rect x="70" y="82" width="8" height="8" fill="black"/>
                <rect x="118" y="82" width="8" height="8" fill="black"/>
                <rect x="142" y="82" width="8" height="8" fill="black"/>
                <rect x="166" y="82" width="8" height="8" fill="black"/>
                <rect x="10" y="94" width="8" height="8" fill="black"/>
                <rect x="34" y="94" width="8" height="8" fill="black"/>
                <rect x="94" y="94" width="8" height="8" fill="black"/>
                <rect x="118" y="94" width="8" height="8" fill="black"/>
                <rect x="154" y="94" width="8" height="8" fill="black"/>
                <rect x="22" y="106" width="8" height="8" fill="black"/>
                <rect x="58" y="106" width="8" height="8" fill="black"/>
                <rect x="82" y="106" width="8" height="8" fill="black"/>
                <rect x="130" y="106" width="8" height="8" fill="black"/>
                <rect x="166" y="106" width="8" height="8" fill="black"/>
                <rect x="70" y="118" width="8" height="8" fill="black"/>
                <rect x="94" y="118" width="8" height="8" fill="black"/>
                <rect x="118" y="118" width="8" height="8" fill="black"/>
                <rect x="154" y="118" width="8" height="8" fill="black"/>
                <rect x="82" y="130" width="8" height="8" fill="black"/>
                <rect x="106" y="130" width="8" height="8" fill="black"/>
                <rect x="130" y="130" width="8" height="8" fill="black"/>
                <rect x="70" y="142" width="8" height="8" fill="black"/>
                <rect x="94" y="142" width="8" height="8" fill="black"/>
                <rect x="142" y="142" width="8" height="8" fill="black"/>
                <rect x="166" y="142" width="8" height="8" fill="black"/>
                <rect x="82" y="154" width="8" height="8" fill="black"/>
                <rect x="118" y="154" width="8" height="8" fill="black"/>
                <rect x="142" y="154" width="8" height="8" fill="black"/>
                <rect x="70" y="166" width="8" height="8" fill="black"/>
                <rect x="106" y="166" width="8" height="8" fill="black"/>
                <rect x="130" y="166" width="8" height="8" fill="black"/>
                <rect x="154" y="166" width="8" height="8" fill="black"/>
              </svg>
            )}
          </div>
        </DialogContent>
      </Dialog>

      {/* ===== 模型操作二次确认弹窗 ===== */}
      <Dialog
        open={modelConfirmDialog.open}
        onOpenChange={(open) => !open && setModelConfirmDialog(prev => ({ ...prev, open: false }))}
      >
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle className="text-blue-600">
              {modelConfirmDialog.type === "delete" ? "确认删除主模型" : modelConfirmDialog.type === "delete-backup" ? "确认删除备选模型" : "切换主模型"}
            </DialogTitle>
            <DialogDescription className="text-gray-600 leading-relaxed pt-1">
              {modelConfirmDialog.type === "delete"
                ? "删除后将自动切换备选模型作为主模型，切换过程中将导致相关的 Gateway 服务重启"
                : modelConfirmDialog.type === "delete-backup"
                ? "删除后将导致相关的 Gateway 服务重启，确认删除么"
                : "将此模型设为主模型后，原主模型将降为备选模型。切换过程中会自动重启 Gateway 服务，是否继续？"}
            </DialogDescription>
          </DialogHeader>
          <div className="flex justify-end gap-2 pt-2">
            <Button
              variant="tenant-outline"
              size="claw-sm"
              onClick={() => setModelConfirmDialog(prev => ({ ...prev, open: false }))}
            >
              取消
            </Button>
            <Button
              size="sm"
              variant={
                modelConfirmDialog.type === "delete" ||
                modelConfirmDialog.type === "delete-backup"
                  ? "tenant-destructive"
                  : "tenant-primary"
              }
              onClick={() => {
                const { type, modelId } = modelConfirmDialog;
                setModelConfirmDialog(prev => ({ ...prev, open: false }));
                if (type === "set-primary" && modelId !== null) {
                  setAppliedModels(prev => prev.map(m => ({ ...m, primary: m.id === modelId })));
                  toast.success("已设为主模型");
                } else if (type === "delete-backup" && modelId !== null) {
                  setAppliedModels(prev => prev.filter(m => m.id !== modelId));
                  toast.success("备选模型已删除");
                } else if (type === "delete" && modelId !== null) {
                  setAppliedModels(prev => {
                    const next = prev.filter(m => m.id !== modelId);
                    const wasPrimary = prev.find(m => m.id === modelId)?.primary ?? false;
                    // 删除主模型后自动将列表中第一个升为主模型
                    if (wasPrimary && next.length > 0) {
                      next[0] = { ...next[0], primary: true };
                    }
                    return next;
                  });
                  toast.success("主模型已删除，已自动升级备选模型");
                }
              }}
            >
              {modelConfirmDialog.type === "delete" || modelConfirmDialog.type === "delete-backup" ? "确认删除" : "确认设置"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* ===== 技能安装二次确认弹窗 ===== */}
      <Dialog
        open={skillInstallConfirm.open}
        onOpenChange={(open) => !open && setSkillInstallConfirm(prev => ({ ...prev, open: false }))}
      >
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle className="text-blue-600">确认安装技能</DialogTitle>
            <DialogDescription className="text-gray-600 leading-relaxed pt-1">
              确认安装名称为
              <span className="font-semibold text-gray-900 mx-1">{skillInstallConfirm.skillName}</span>
              的技能？
            </DialogDescription>
          </DialogHeader>
          {/* SkillHub 配置提示 */}
          <div className="flex items-start gap-2 px-3 py-2.5 rounded-[4px] bg-blue-50 border border-blue-200 mt-1">
            <AlertCircle className="w-4 h-4 text-blue-600 flex-shrink-0 mt-0.5" />
            <div className="flex-1 text-xs text-blue-700 leading-relaxed">
              管理员配置了
              <a href="https://skillhub.tencent.com/" target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1 text-blue-600 hover:text-blue-700 underline underline-offset-1 font-medium">
                SkillHub地址
                <ExternalLink className="w-3 h-3 flex-shrink-0" />
              </a>
              ，不支持模糊搜索，请输入准确Skill名称
            </div>
          </div>
          <div className="flex items-start gap-2 px-3 py-2.5 rounded-[4px] bg-amber-50 border border-amber-200 mt-1">
            <CircleAlert className="w-4 h-4 text-amber-500 flex-shrink-0 mt-0.5" />
            <p className="text-xs text-amber-700 leading-relaxed">部分技能(Skills)可能存在安全风险，安装前请确认其安全性。</p>
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button
              variant="tenant-outline"
              size="claw-sm"
              onClick={() => setSkillInstallConfirm(prev => ({ ...prev, open: false }))}
            >
              取消
            </Button>
            <Button
              size="sm"
              onClick={() => {
                const name = skillInstallConfirm.skillName;
                setSkillInstallConfirm({ open: false, skillName: "" });
                setSkillSearch("");
                // 添加到待安装队列
                setPendingSkills(prev => [
                  ...prev,
                  { id: `ps-${Date.now()}`, name, status: "pending" as const },
                ]);
                toast.success(`技能「${name}」已加入安装队列`);
              }}
            >
              确认安装
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* ===== 技能卸载二次确认弹窗 ===== */}
      <AlertDialog
        open={skillUninstallConfirm.open}
        onOpenChange={(open) => !open && setSkillUninstallConfirm({ open: false, entry: "" })}
      >
        <AlertDialogContent className="max-w-sm">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-base font-semibold text-[var(--text-strong)]">卸载技能</AlertDialogTitle>
            <AlertDialogDescription className="text-sm text-[var(--text-muted)] leading-relaxed pt-1">
              确认卸载
              <span className="font-semibold text-[var(--text-strong)] mx-1">
                {parseSkillEntry(skillUninstallConfirm.entry).name}
              </span>
              ？卸载后该 Agent 将无法继续使用此技能。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <Button
              variant="tenant-outline"
              size="claw-sm"
              onClick={() => setSkillUninstallConfirm({ open: false, entry: "" })}
            >
              取消
            </Button>
            <Button
              variant="tenant-destructive"
              size="claw-sm"
              onClick={handleConfirmUninstallSkill}
            >
              确认卸载
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* ==================== 编辑项目关联弹窗 ==================== */}
      <Dialog open={projectEditOpen} onOpenChange={setProjectEditOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold">编辑项目关联</DialogTitle>
            <DialogDescription className="text-xs text-[var(--text-secondary)]">
              从项目池中勾选关联项目。新增关联携带技能的项目，保存后其技能会自动安装到当前 Agent；取消关联不会移除已安装的技能。
            </DialogDescription>
          </DialogHeader>
          <div className="py-2">
            {projectPool.length === 0 ? (
              <p className="text-sm text-[var(--text-weak)] py-6 text-center">暂无可关联的项目</p>
            ) : (
              <div className="flex flex-wrap gap-2 max-h-[280px] overflow-y-auto">
                {projectPool.map((p) => {
                  const active = projectDraftIds.includes(p.id);
                  return (
                    <button
                      key={p.id}
                      type="button"
                      onClick={() => toggleProjectDraft(p.id)}
                      className={`inline-flex items-center gap-1.5 h-8 px-3 rounded-full border text-sm transition-colors ${
                        active
                          ? "border-[var(--text-brand)] bg-[var(--text-brand)] text-white"
                          : "border-[var(--border-control)] bg-[var(--card)] text-[var(--text-secondary)] hover:border-[var(--text-brand)]"
                      }`}
                    >
                      {active && <Check className="w-3.5 h-3.5" />}
                      {p.name}
                    </button>
                  );
                })}
              </div>
            )}
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="tenant-outline" size="claw-sm" onClick={() => setProjectEditOpen(false)}>
              取消
            </Button>
            <Button variant="tenant-primary" size="claw-sm" onClick={saveProjectEdit}>
              保存
            </Button>
          </div>
        </DialogContent>
      </Dialog>
      {/* ==================== Agent 迁移弹窗 ==================== */}
      <Dialog open={migrationOpen} onOpenChange={setMigrationOpen}>
        <DialogContent ref={migrationDialogRef} className="max-w-lg max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold">
              迁移 Agent 至当前实例
            </DialogTitle>
            <DialogDescription className="text-xs text-gray-500">
              将源端 Agent 的配置、通道状态、会话历史导入到「{clawData?.name}」
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-5 mt-2">
            {/* 注意事项 */}
            <div className="rounded-[4px] bg-amber-50 border border-amber-200 p-3 space-y-1.5">
              <p className="text-xs text-amber-800 font-semibold flex items-center gap-1">
                <CircleAlert className="w-3.5 h-3.5" /> 注意事项
              </p>
              <ul className="text-xs text-amber-700 space-y-1 list-disc pl-4 leading-relaxed">
                <li><strong className="text-red-600">源端 Agent 类型必须与当前实例的 Agent 类型一致</strong>（如当前为 {(claw as any).agentType === "hermes" ? "Hermes Agent" : (claw as any).agentType === "lightclawace" ? "LightClaw ACE" : "OpenClaw"}，则源端也须为同类型），否则配置文件将无法兼容，导致迁移失败</li>
                <li>源端 Agent 的配置、通道登录状态、会话历史将完整导入到当前实例，源端仅做读取打包，不影响源端正常运行</li>
                <li>导入将覆盖当前实例的 ~/.agent/ 目录，导入前自动备份，失败自动回滚</li>
              </ul>
            </div>

            {/* Step 1: 导出源端配置 */}
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <div className={`w-5 h-5 rounded-full flex items-center justify-center text-xs font-bold ${
                  migrationStep === "export" || migrationStep === "waitUpload"
                    ? "bg-blue-500 text-white" : "bg-green-500 text-white"
                }`}>
                  {migrationStep !== "export" && migrationStep !== "waitUpload" ? <CheckCircle2 className="w-3 h-3" /> : "1"}
                </div>
                <h3 className="text-sm font-semibold text-gray-900">导出源端 Agent 配置</h3>
              </div>
              <p className="text-xs text-gray-500 ml-7">
                请复制下方命令，在源 Agent 终端或 IM 机器人对话框中执行。
              </p>
              {!migrationCommandReady ? (
                <div className="ml-7 bg-gray-50 border border-gray-200 rounded-[4px] p-6 flex flex-col items-center justify-center gap-2">
                  <Loader2 className="w-5 h-5 text-blue-500 animate-spin" />
                  <p className="text-xs text-gray-500">正在生成迁移命令...</p>
                  <p className="text-xs text-gray-400">正在获取临时上传凭证和 COS 预签名链接</p>
                </div>
              ) : (
              <div className="ml-7 relative bg-gray-50 border border-gray-200 rounded-[4px] p-3">
                <button
                  onClick={() => { navigator.clipboard.writeText(migrationExportCommand); toast.success("命令已复制"); }}
                  className="absolute top-2 right-2 p-1.5 rounded-[4px] bg-gray-100 hover:bg-gray-200 text-gray-500 hover:text-gray-700 transition-colors"
                  title="复制命令"
                >
                  <Copy className="w-3.5 h-3.5" />
                </button>
                <pre className="text-xs text-gray-700 font-mono whitespace-pre-wrap break-all leading-relaxed pr-8">{migrationExportCommand}</pre>
              </div>
              )}
              <div className="ml-7 text-xs text-gray-400 space-y-0.5">
                <p className="flex items-center gap-1"><Clock className="w-3 h-3" /> 上传链接有效期 1 小时，超时请刷新页面重新获取</p>
              </div>
            </div>

            {/* Step 2: 检测上传 & 导入 */}
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <div className={`w-5 h-5 rounded-full flex items-center justify-center text-xs font-bold ${
                  migrationStep === "export" || migrationStep === "waitUpload" ? "bg-gray-300 text-gray-600"
                  : migrationStep === "import" ? "bg-blue-500 text-white"
                  : "bg-green-500 text-white"
                }`}>
                  {migrationStep === "success" || migrationStep === "importing" ? <CheckCircle2 className="w-3 h-3" /> : "2"}
                </div>
                <h3 className="text-sm font-semibold text-gray-900">将源端配置导入当前实例</h3>
              </div>

              {!migrationUploaded && (migrationStep === "export" || migrationStep === "waitUpload") && (
                <div className="ml-7 space-y-2">
                  <p className="text-xs text-gray-500">执行完导出命令后，点击检测上传状态：</p>
                  <button
                    onClick={handleCheckUpload}
                    disabled={migrationChecking || !migrationCommandReady}
                    className="inline-flex items-center gap-1.5 text-xs font-medium border rounded-[4px] px-3 py-1.5 transition-colors text-white border-blue-500 disabled:opacity-50"
                    style={{ background: "#355EF1" }}
                  >
                    {migrationChecking ? <Loader2 className="w-3 h-3 animate-spin" /> : <Search className="w-3 h-3" />}
                    {migrationChecking ? "检测中..." : migrationCheckFailed ? "重新检测" : "检测上传状态"}
                  </button>
                  {migrationCheckFailed && (
                    <div className="rounded-[4px] bg-red-50 border border-red-200 p-2.5 space-y-1">
                      <p className="text-xs text-red-700 font-medium flex items-center gap-1">
                        <XCircle className="w-3.5 h-3.5" /> 未检测到数据包
                      </p>
                      <ul className="text-xs text-red-600 list-disc pl-4 space-y-0.5 leading-relaxed">
                        <li>请确认已在源端执行完导出命令，且命令输出包含 "✅ 导出完成"</li>
                        <li>检查源端网络是否正常，curl 上传是否报错</li>
                        <li>上传链接有效期 1 小时，超时请关闭弹窗重新打开获取新链接</li>
                      </ul>
                    </div>
                  )}
                </div>
              )}

              {migrationStep === "import" && (
                <div className="ml-7 space-y-3">
                  <div className="rounded-[4px] bg-green-50 border border-green-200 p-2.5">
                    <p className="text-xs text-green-700 flex items-center gap-1">
                      <CheckCircle2 className="w-3.5 h-3.5" /> 已检测到上传的数据包
                    </p>
                  </div>
                  <div className="rounded-[4px] bg-red-50 border border-red-200 p-3">
                    <p className="text-xs text-red-700 font-semibold flex items-center gap-1">
                      <CircleAlert className="w-3.5 h-3.5" /> 重要提醒
                    </p>
                    <p className="text-xs text-red-600 mt-1 leading-relaxed">
                      执行导入将<strong>覆盖</strong>当前实例「{clawData?.name}」的全部 Agent 配置（~/.agent/ 目录）。
                      导入前会自动备份，失败时自动回滚。
                    </p>
                  </div>
                  <button
                    onClick={handleStartMigration}
                    className="inline-flex items-center gap-1.5 text-xs font-semibold rounded-[4px] px-4 py-2 transition-colors text-white bg-blue-600 hover:bg-blue-700"
                  >
                    <ArrowLeftRight className="w-3.5 h-3.5" />
                    导入并重启 Agent
                  </button>
                </div>
              )}

              {migrationStep === "importing" && (
                <div className="ml-7 space-y-3">
                  <p className="text-xs text-blue-600 flex items-center gap-1.5 font-medium">
                    <Loader2 className="w-3.5 h-3.5 animate-spin" /> 正在执行导入...
                  </p>
                  <div className="space-y-1.5">
                    {importSteps.map((step, i) => (
                      <div key={i} className={`flex items-center gap-2 px-3 py-1.5 rounded-[4px] text-xs ${
                        step.status === "done" ? "bg-green-50" :
                        step.status === "running" ? "bg-blue-50" :
                        step.status === "failed" ? "bg-red-50" : "bg-gray-50"
                      }`}>
                        {step.status === "done" && <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />}
                        {step.status === "running" && <Loader2 className="w-3.5 h-3.5 text-blue-500 animate-spin flex-shrink-0" />}
                        {step.status === "failed" && <XCircle className="w-3.5 h-3.5 text-red-500 flex-shrink-0" />}
                        {step.status === "pending" && <div className="w-3.5 h-3.5 rounded-full border-2 border-gray-300 flex-shrink-0" />}
                        <span className={
                          step.status === "done" ? "text-green-700" :
                          step.status === "running" ? "text-blue-700 font-medium" :
                          step.status === "failed" ? "text-red-700" : "text-gray-400"
                        }>
                          {step.label}
                        </span>
                        {step.status === "done" && <span className="text-green-500 ml-auto">✓</span>}
                        {step.status === "running" && <span className="text-blue-400 ml-auto">进行中...</span>}
                        {step.status === "failed" && step.error && <span className="text-red-500 ml-auto">{step.error}</span>}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Step 3: Result */}
            {migrationStep === "success" && (
              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <div className="w-5 h-5 rounded-full bg-green-500 text-white flex items-center justify-center">
                    <CheckCircle2 className="w-3 h-3" />
                  </div>
                  <h3 className="text-sm font-semibold text-green-700">
                    {verifyResults.every((r) => r.passed) ? "迁移成功，已验证生效" : "迁移完成，部分项需处理"}
                  </h3>
                </div>
                {/* 验证结果 */}
                {verifyResults.length > 0 && (
                  <div className="ml-7 space-y-1.5">
                    <p className="text-xs text-gray-500 font-medium">导入后验证：</p>
                    {verifyResults.map((v, i) => (
                      <div key={i} className={`flex items-center gap-2 px-3 py-1.5 rounded-[4px] text-xs ${v.passed ? "bg-green-50" : "bg-amber-50"}`}>
                        {v.passed
                          ? <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                          : <CircleAlert className="w-3.5 h-3.5 text-amber-500 flex-shrink-0" />}
                        <span className={v.passed ? "text-green-700" : "text-amber-700"}>{v.label}</span>
                        <code className="text-gray-400 font-mono ml-1">{v.cmd}</code>
                        <span className={`ml-auto ${v.passed ? "text-green-500" : "text-amber-600"}`}>{v.detail}</span>
                      </div>
                    ))}
                  </div>
                )}
                {!verifyResults.every((r) => r.passed) && verifyResults.some((r) => !r.passed) && (
                  <div className="ml-7 rounded-[4px] bg-amber-50 border border-amber-200 p-2.5">
                    <p className="text-xs text-amber-700">
                      <CircleAlert className="w-3 h-3 inline mr-1" />
                      部分验证项未通过，Agent 核心功能已正常运行，未通过项可能需要手动处理（如重新登录 IM 通道）。
                    </p>
                  </div>
                )}
                <div className="ml-7">
                  <button onClick={() => setMigrationOpen(false)}
                    className="text-xs font-medium text-blue-600 hover:underline">
                    关闭
                  </button>
                </div>
              </div>
            )}

            {migrationStep === "failed" && (
              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <div className="w-5 h-5 rounded-full bg-red-500 text-white flex items-center justify-center">
                    <XCircle className="w-3 h-3" />
                  </div>
                  <h3 className="text-sm font-semibold text-red-700">迁移失败</h3>
                </div>
                {/* 显示步骤流转状态，方便定位失败在哪步 */}
                <div className="ml-7 space-y-1.5">
                  {importSteps.map((step, i) => (
                    <div key={i} className={`flex items-center gap-2 px-3 py-1.5 rounded-[4px] text-xs ${
                      step.status === "done" ? "bg-green-50" :
                      step.status === "failed" ? "bg-red-50" : "bg-gray-50"
                    }`}>
                      {step.status === "done" && <CheckCircle2 className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />}
                      {step.status === "failed" && <XCircle className="w-3.5 h-3.5 text-red-500 flex-shrink-0" />}
                      {step.status === "pending" && <div className="w-3.5 h-3.5 rounded-full border-2 border-gray-300 flex-shrink-0" />}
                      <span className={
                        step.status === "done" ? "text-green-700" :
                        step.status === "failed" ? "text-red-700 font-medium" : "text-gray-400"
                      }>{step.label}</span>
                      {step.status === "failed" && step.error && <span className="text-red-500 ml-auto text-xs">{step.error}</span>}
                    </div>
                  ))}
                </div>
                <div className="ml-7 rounded-[4px] bg-red-50 border border-red-200 p-3 space-y-1.5">
                  <p className="text-xs text-red-700">{migrationError}</p>
                  <p className="text-xs text-red-600">已自动回滚至导入前状态，当前实例配置未受影响。</p>
                </div>
                <div className="ml-7 flex gap-2">
                  <button onClick={retryImport}
                    className="text-xs font-medium text-blue-600 hover:underline">
                    重新导入
                  </button>
                  <button onClick={() => setMigrationOpen(false)}
                    className="text-xs font-medium text-gray-500 hover:underline">
                    关闭
                  </button>
                </div>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>

    </TenantLayout>
    </TooltipProvider>
  );
}

// ─── 龙虾医生对话卡片（v2：实例化 + LightClaw SDK 插槽样式）────────────────────
//
// 架构变化（与 v1 的差异）：
// 1. 「开始诊断」= 创建一台龙虾医生 OpenClaw 实例（前端 mock 创建 + 轮询）
// 2. 同一用户同时只能有一个实例（mock 单实例冲突）
// 3. 历史 Session 从 localStorage 还原（mock COS），直接在对话框最上方滑入展示
// 4. 视觉 100% 复用 LightClaw 对话框（参考 ChatView.tsx 同款气泡 / 输入框样式）
// 5. SDK 插槽：「开始修复」(检测完出现)、「结束诊断」(诊断中始终可见，左下角)
// 6. 实例 Active 后自动注入第一条全面检测提示词
//
// MOCK 标注：所有后端调用（实例查询/创建/销毁/轮询/Session COS 还原）均为前端模拟，
//          标注 "// MOCK:" 注释，等接口对齐后替换。
// ────────────────────────────────────────────────────────────────────────────────

// ─── 类型定义 ────────────────────────────────────────────────────────────────

type DiagCheckItem = {
  label: string;
  status: "ok" | "error" | "warn";
  detail?: string;
};

type RepairResult = {
  label: string;
  ok: boolean;
  reason?: string; // 失败原因
};

type DoctorMessageContent =
  | { type: "text"; text: string }
  | { type: "check_list"; items: DiagCheckItem[] }
  | { type: "repair_summary"; results: RepairResult[] };

type DoctorMsg =
  | { kind: "system"; text: string; transient?: boolean }
  | { kind: "assistant"; parts: DoctorMessageContent[]; loading?: boolean; transient?: boolean }
  | { kind: "user"; text: string };

// ─── 系统消息构造器 ──────────────────────────────────────────────────────────
// 所有"流程提示"性质的系统消息默认 transient（不进缓存），刷新后不重复显示。
// 仅在确有"业务追溯价值"时（如：用户主动回滚、用户主动结束）才显式传 persistent=true。
function sysMsg(text: string, persistent = false): DoctorMsg {
  return persistent
    ? { kind: "system", text }
    : { kind: "system", text, transient: true };
}

// 「龙虾医生」第一人称口语化文本气泡——用于流程提示场景，
// 替代过去居中灰胶囊样式的 system 提示。视觉上与 LLM 真实回复完全一致（左对齐纯文字、无装饰），
// 让用户对所有"医生在说话"的内容形成统一心智，避免"系统公告"与"医生回复"两种平行视觉语言。
//
// 持久化策略（默认 transient）：
//   - 默认 transient = true，purgeForPersist 会过滤掉，刷新页面 / 历史会话回看时不会留痕；
//     适用于「创建中 / 已就绪 / 修复中 / 自动结束 / 离场」等纯过程提示——它们在当时有展示价值，
//     但归档到历史 Session 时只会污染回看视图，故不持久化。
//   - 当传 persistent = true 时持久化，适用于「用户主动操作的破坏性确认」（如已回滚到诊断前快照）
//     ——用户日后回看历史会话时，必须能看到当时的关键决策痕迹。
function doctorMsg(text: string, persistent = false): DoctorMsg {
  return persistent
    ? { kind: "assistant", parts: [{ type: "text", text }] }
    : { kind: "assistant", parts: [{ type: "text", text }], transient: true };
}

// ─── 持久化前的清洗：剔除瞬态/加载中消息 ─────────────────────────────────────
// 用于「写入 localStorage 缓存」和「写入历史 Session 快照」前的统一清洗，避免：
// 1) transient 系统提示（"已恢复"/"已就绪"等）被持久化后刷新累积
// 2) transient 助手过程提示（doctorMsg 默认；"创建中"/"修复中"等）污染历史回看
// 3) 流式回复未完成的 "…" 占位被持久化为永久占位
function purgeForPersist(msgs: DoctorMsg[]): DoctorMsg[] {
  return msgs.filter((m) => {
    if (m.kind === "system" && m.transient) return false;
    if (m.kind === "assistant" && m.transient) return false;
    if (m.kind === "assistant" && m.loading) return false;
    return true;
  });
}

// ─── 缓存读取后的结构校验 ────────────────────────────────────────────────────
// 防御老版本写入的脏数据 / 类型缺失字段，确保渲染层永远拿到合法形状。
function sanitizeLoadedMessages(raw: unknown): DoctorMsg[] {
  if (!Array.isArray(raw)) return [];
  const out: DoctorMsg[] = [];
  for (const item of raw) {
    if (!item || typeof item !== "object") continue;
    const m = item as Record<string, unknown>;
    if (m.kind === "user" && typeof m.text === "string") {
      out.push({ kind: "user", text: m.text });
    } else if (m.kind === "assistant" && Array.isArray(m.parts)) {
      // 丢弃 loading=true 的脏占位（应该已经被 purgeForPersist 拦住，这里是兜底）
      if (m.loading) continue;
      // 丢弃 transient=true 的过程提示气泡（兜底；应该已经被 purgeForPersist 拦住）
      if (m.transient) continue;
      out.push({ kind: "assistant", parts: m.parts as DoctorMessageContent[] });
    } else if (m.kind === "system" && typeof m.text === "string") {
      // 丢弃旧版本残留的 "已复用/已恢复" 系统消息（自愈）
      if (m.text === "已复用上次未结束的诊断会话") continue;
      if (m.text === "已恢复上次未结束的诊断会话") continue;
      // 其余 system 消息（如旧版未标 transient 的"已就绪"等）也一并丢弃，
      // 因为它们本质都是流程提示，没有回看价值，留着只会重复展示
      if (!m.transient) continue;
      // 极端情况下仍带 transient 的也丢
      continue;
    }
  }
  return out;
}

// 实例状态机（架构核心）
type InstanceStatus = "none" | "creating" | "active" | "destroying" | "ended";

// 诊断阶段（仅在 active 期间细分）
type DiagPhase =
  | "diagnosing"      // 龙虾医生检测中
  | "summary_ready"   // 已输出问题汇总，等用户点「开始修复」
  | "repairing"       // 修复中
  | "done"            // 修复完成或自由对话中
  | "idle";           // 实例 active 但还没开始诊断（创建中→active 瞬态）

// 历史 Session（mock COS 持久化用）
type SessionSnapshot = {
  endedAt: string;     // 结束时间
  instanceId: string;  // 实例 ID（按 instanceId 区分）
  agentInstanceId: string; // 用户的 agent 实例 ID（按 agent 区分）
  messages: DoctorMsg[];
};

// ─── 第一条自动注入的提示词（产品文档定义） ────────────────────────────────────
const AUTO_FIRST_PROMPT =
  "请对当前 Agent 进行全面检测，覆盖所有可检测项目（包括但不限于网络、模型、通道、技能等运行状态），实时输出每项进度和结果，异常项附简短说明。完成后汇总问题列表，逐项给出原因和修复方案，等待我回复后再执行修复。";

// ─── 子组件：检测结果列表 ─────────────────────────────────────────────────────
function CheckList({ items }: { items: DiagCheckItem[] }) {
  // 字体规范（与 assistant 气泡顶层 text-sm/gray-900 标杆对齐，与 RepairSummary 同源）：
  //   ① 标签字色 → gray-900（之前 gray-700 偏浅）；
  //   ② "正常"详情字色 → gray-900（之前 gray-400 太浅，看不清）；
  //   ③ "异常"详情保留红/橙语义色 + font-medium——这是状态语义着色，
  //      不属于"层级颜色"范畴，必须保留以让用户一眼定位异常项。
  return (
    <div className="space-y-1.5 mt-1">
      {items.map((item, i) => (
        <div key={i} className="flex items-center gap-2 text-sm text-gray-900">
          <span
            className={`w-2 h-2 rounded-full flex-shrink-0 ${
              item.status === "ok" ? "bg-green-500" :
              item.status === "error" ? "bg-red-500" : "bg-orange-400"
            }`}
          />
          <span className="w-28 flex-shrink-0">{item.label}</span>
          <span className={
            item.status === "error" ? "text-red-600 font-medium" :
            item.status === "warn"  ? "text-orange-600 font-medium" :
            ""
          }>
            {item.detail ?? (item.status === "ok" ? "正常" : "异常")}
          </span>
        </div>
      ))}
    </div>
  );
}

// ─── 子组件：修复结果汇总（成功 + 失败组织） ───────────────────────────────────
// 字体规范（与 assistant 气泡顶层 text-sm/gray-900 标杆对齐）：
//   ① 字色统一 text-gray-900——避免子模块看起来比所属气泡"褪色"，
//      之前 gray-500 / gray-700 / gray-400 三档并存导致视觉破碎；
//   ② 主体字号统一 text-sm（14px）——组织标题用 font-semibold 表达层级，
//      不再用更小字号区分；
//   ③ 唯一例外：失败原因（reason）保留 text-xs（12px）——这是"主条目下的辅助说明"，
//      按主流 UX 惯例（LightClaw / Claude / Cursor）小一号有助于扫读，
//      但字色仍归一到 gray-900 以保持视觉同源；
//   ④ ✓ 成功标记保留视觉降权——用 emerald-500 浅色，避免与红色失败图标视觉权重失衡，
//      同时让眼睛优先聚焦在"标签文字"本身而非装饰符号。
function RepairSummary({ results }: { results: RepairResult[] }) {
  const succeed = results.filter((r) => r.ok);
  const failed = results.filter((r) => !r.ok);
  return (
    <div className="space-y-2.5 mt-2">
      {succeed.length > 0 && (
        <div>
          <p className="text-sm font-semibold text-gray-900 mb-1.5">已成功修复 {succeed.length} 项：</p>
          <div className="space-y-1.5">
            {succeed.map((r, i) => (
              <div key={i} className="flex items-center gap-2 text-sm text-gray-900">
                <span className="w-2 h-2 rounded-full flex-shrink-0 bg-green-500" />
                <span>{r.label}</span>
                <span className="text-emerald-500">✓</span>
              </div>
            ))}
          </div>
        </div>
      )}
      {failed.length > 0 && (
        <div>
          <p className="text-sm font-semibold text-gray-900 mb-1.5">{failed.length} 项修复失败：</p>
          <div className="space-y-1.5">
            {failed.map((r, i) => (
              <div key={i} className="flex items-start gap-2 text-sm text-gray-900">
                <span className="w-2 h-2 rounded-full flex-shrink-0 bg-red-500 mt-1.5" />
                <div className="min-w-0">
                  <p>{r.label}</p>
                  {r.reason && <p className="text-xs text-gray-900 mt-0.5">{r.reason}</p>}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// ─── 子组件：打字动画 ─────────────────────────────────────────────────────────
function TypingBubble() {
  return (
    <div className="flex gap-3 py-1">
      <div className="flex items-center gap-1">
        <span className="w-1.5 h-1.5 rounded-full bg-gray-400 animate-bounce" style={{ animationDelay: "0ms" }} />
        <span className="w-1.5 h-1.5 rounded-full bg-gray-400 animate-bounce" style={{ animationDelay: "150ms" }} />
        <span className="w-1.5 h-1.5 rounded-full bg-gray-400 animate-bounce" style={{ animationDelay: "300ms" }} />
      </div>
    </div>
  );
}

// ─── localStorage Keys ────────────────────────────────────────────────────────
// 诊断选项 · 授权（agent 维度）—— 用户自主开关，可随时启用/停用
//   说明：早期版本在此 key 写入 "true" 用作"首次授权后永久免弹"标志；
//   重构后该 key 含义变更为「上次该 agent 是否启用授权选项」，仍兼容旧值。
const getDiagAuthKey = (agentId: string) => `doctor_diag_auth_${agentId}`;
// 诊断选项 · 配置快照（agent 维度）—— 用户自主开关，可随时启用/停用
const getDiagSnapshotKey = (agentId: string) => `doctor_diag_snapshot_${agentId}`;
// MOCK COS：每个 agent 一份"已结束"历史 Session
const getSessionStoreKey = (agentId: string) => `doctor_session_cos_${agentId}`;
// 活跃会话实时缓存（按 doctorInstanceId 分桶；用于刷新/复用时恢复对话）
const getLiveMessagesKey = (doctorId: string) => `doctor_live_msgs_${doctorId}`;
// MOCK 后端：当前用户是否已有 active 实例（全局单实例）
const ACTIVE_INSTANCE_KEY = "doctor_active_instance";

// ─── 龙虾医生 · 指令库 ───────────────────────────────────────────────────────
// 与 ChatView（普通 Agent 对话）通用指令保持完全一致——确保用户在任何 Agent
// 入口看到的指令库都是同一套，降低认知成本。
// 注：诊断业务专属指令（/diagnose、/repair、/end）暂未实现，保留按钮入口走交互。
const DOCTOR_COMMAND_LIST = [
  { command: "/new",      label: "新建会话" },
  { command: "/compact",  label: "压缩上下文" },
  { command: "/status",   label: "查看状态" },
  { command: "/commands", label: "全部指令" },
];

// ─── 诊断选项 · 数据层（authorize / snapshot 两个完全独立的可选项）──────────
// 设计原则：
//   1. 两个选项互不依赖，彼此不知道对方存在；
//   2. 都是「用户可自主开关」的偏好（像设置开关一样）；
//   3. 都按 agentId 维度持久化，下次打开记住上次选择，但用户随时可改；
//   4. 启动诊断时由调用方读取两份偏好分别独立处理，不再做"首次必弹"特殊路径。
//
// 后续接入真后端时，把 localStorage 替换为 GET/PUT /api/agent/{id}/diag-options
// 即可，调用方 API 完全不变。
// ────────────────────────────────────────────────────────────────────────────

type DiagOptionKey = "authorize" | "snapshot";

const DIAG_OPTION_STORAGE: Record<DiagOptionKey, (agentId: string) => string> = {
  authorize: getDiagAuthKey,
  snapshot:  getDiagSnapshotKey,
};

// 默认值：授权 = 关（用户须主动 opt-in）；快照 = 开（推荐勾选）
const DIAG_OPTION_DEFAULTS: Record<DiagOptionKey, boolean> = {
  authorize: false,
  snapshot:  true,
};

function loadDiagOption(agentId: string, key: DiagOptionKey): boolean {
  try {
    const raw = localStorage.getItem(DIAG_OPTION_STORAGE[key](agentId));
    if (raw === null) return DIAG_OPTION_DEFAULTS[key];
    return raw === "true";
  } catch {
    return DIAG_OPTION_DEFAULTS[key];
  }
}

function saveDiagOption(agentId: string, key: DiagOptionKey, enabled: boolean) {
  try {
    localStorage.setItem(DIAG_OPTION_STORAGE[key](agentId), enabled ? "true" : "false");
  } catch {
    // ignore
  }
}

// ─── 已授权标记（agent 级一次性持久化）────────────────────────────────────────
// 与「authorize 偏好」(可随时 opt-out 的开关) 是两个不同语义：
//   - authorize 偏好  —— "我这次愿不愿意授权"，存的是用户当前选择，可来回切；
//   - hasAuthorized  —— "我曾经授权过此 Agent"，存的是历史事实，一旦为 true 就不再问。
// 决定「开始诊断时是否弹授权确认窗」的是 hasAuthorized，而不是 authorize 偏好。
// 后续接入真后端时，替换为 GET /api/agent/{id}/auth-status。
function getDiagAuthorizedKey(agentId: string) {
  return `oc.diag.authorized.${agentId}`;
}

function hasAuthorizedDiag(agentId: string): boolean {
  try {
    return localStorage.getItem(getDiagAuthorizedKey(agentId)) === "true";
  } catch {
    return false;
  }
}

function markAuthorizedDiag(agentId: string) {
  try {
    localStorage.setItem(getDiagAuthorizedKey(agentId), "true");
  } catch {
    // ignore
  }
}

// ─── 已询问授权标记（agent 级一次性持久化）─────────────────────────────────────
// 与 hasAuthorizedDiag 的区别：
//   - hasAuthorizedDiag —— "用户曾经勾选并同意授权" → 决定后端是否能使用诊断记录；
//   - hasAskedAuth      —— "我们曾经向用户询问过授权这件事" → 决定 UI 是否还要再弹问。
// 不论用户当时勾没勾，只要弹窗里出现过授权选项并由用户做出过决定（哪怕是"不勾"），
// 就视作"已问过"，后续不再打扰；这符合「授权一次性」的产品语义——不能因为用户拒绝就反复弹。
// 后续接入真后端时，替换为 GET /api/agent/{id}/auth-status 中的 "asked" 字段（或独立接口）。
function getDiagAuthAskedKey(agentId: string) {
  return `oc.diag.authAsked.${agentId}`;
}

function hasAskedAuth(agentId: string): boolean {
  try {
    return localStorage.getItem(getDiagAuthAskedKey(agentId)) === "true";
  } catch {
    return false;
  }
}

function markAskedAuth(agentId: string) {
  try {
    localStorage.setItem(getDiagAuthAskedKey(agentId), "true");
  } catch {
    // ignore
  }
}

/**
 * useDiagnosisOptions —— 诊断启动选项的统一管理 hook。
 * 返回每个独立选项的 [value, setValue]，调用方对它们一视同仁，
 * 不感知"授权"和"快照"在业务上的差异。
 */
function useDiagnosisOptions(agentId: string) {
  const [authorize, setAuthorizeRaw] = useState<boolean>(() => loadDiagOption(agentId, "authorize"));
  const [snapshot,  setSnapshotRaw]  = useState<boolean>(() => loadDiagOption(agentId, "snapshot"));

  const setAuthorize = (v: boolean) => { setAuthorizeRaw(v); saveDiagOption(agentId, "authorize", v); };
  const setSnapshot  = (v: boolean) => { setSnapshotRaw(v);  saveDiagOption(agentId, "snapshot",  v); };

  // 切换 agent 时重新读取偏好（避免跨 agent 串值）
  useEffect(() => {
    setAuthorizeRaw(loadDiagOption(agentId, "authorize"));
    setSnapshotRaw(loadDiagOption(agentId, "snapshot"));
  }, [agentId]);

  return { authorize, setAuthorize, snapshot, setSnapshot } as const;
}

// ─── 诊断选项 · UI 层（完全通用的勾选卡片，不感知授权/快照差异）──────────────
type DiagOptionCardProps = {
  checked: boolean;
  onChange: (next: boolean) => void;
  title: string;
  description: string;
  /** 可选补充说明，作为独立段落在 description 下方灰色显示（仅 DiagOptionRow 使用） */
  extra?: string;
};

function DiagOptionCard({ checked, onChange, title, description }: DiagOptionCardProps) {
  return (
    <label
      className="flex items-start gap-2.5 cursor-pointer select-none rounded-[4px] px-3 py-2.5 transition-colors"
      style={{ border: "1px solid #EDEFF5", background: "#FFFFFF" }}
    >
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="sr-only"
      />
      {/* allow-inline-gradient: 16×16 自定义 Checkbox 选中态色块（非按钮，SKILL §8.1 白名单） */}
      <span
        aria-hidden
        className="mt-0.5 flex-shrink-0 inline-flex items-center justify-center rounded-[5px] transition-all"
      >
        {checked && (
          <svg width="10" height="10" viewBox="0 0 12 12" fill="none">
            <path d="M2.5 6.2L4.8 8.5L9.5 3.5" stroke="#FFFFFF" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        )}
      </span>
      <div className="flex-1 min-w-0">
        <p className="text-sm text-gray-700 leading-snug">{title}</p>
        <p className="text-xs text-gray-400 mt-0.5 leading-relaxed">{description}</p>
      </div>
    </label>
  );
}

/**
 * DiagOptionRow —— 用于多个独立选项「合并到同一个气泡卡片内」时的行版本。
 * 与 DiagOptionCard 共享同套交互/勾选样式，但不带外层 border/background，
 * 由父级容器统一提供卡片外壳。
 * 字号规范：标题 text-sm / 描述 text-xs，与「结束诊断弹窗 · 回滚到诊断前快照」
 * 复选框对齐（统一全站「勾选项 · 标题/描述」字体层级，避免前后弹窗大小不一致）。
 */
function DiagOptionRow({ checked, onChange, title, description, extra }: DiagOptionCardProps) {
  return (
    <label className="flex items-start gap-2.5 cursor-pointer select-none px-3 py-2.5">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="sr-only"
      />
      {/* allow-inline-gradient: 16×16 自定义 Checkbox（非按钮，SKILL §8.1 白名单）
          未选中：白底 + 浅灰边框；选中：品牌蓝实心 + 白色对勾 */}
      <span
        aria-hidden
        className="mt-0.5 flex-shrink-0 inline-flex items-center justify-center rounded-[4px] transition-all"
        style={{
          width: 16,
          height: 16,
          background: checked ? "#1447E6" : "#FFFFFF",
          border: checked ? "1px solid #1447E6" : "1px solid #D0D5DD",
        }}
      >
        {checked && (
          <svg width="10" height="10" viewBox="0 0 12 12" fill="none">
            <path d="M2.5 6.2L4.8 8.5L9.5 3.5" stroke="#FFFFFF" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        )}
      </span>
      <div className="flex-1 min-w-0">
        <p className="text-sm text-gray-700 leading-snug">{title}</p>
        <p className="text-xs text-gray-400 mt-0.5 leading-relaxed">{description}</p>
        {extra && (
          <p className="text-xs text-gray-400 mt-1.5 leading-relaxed">{extra}</p>
        )}
      </div>
    </label>
  );
}

function loadHistorySession(agentId: string): SessionSnapshot | null {
  try {
    const raw = localStorage.getItem(getSessionStoreKey(agentId));
    return raw ? (JSON.parse(raw) as SessionSnapshot) : null;
  } catch {
    return null;
  }
}

function saveHistorySession(agentId: string, snap: SessionSnapshot) {
  try {
    localStorage.setItem(getSessionStoreKey(agentId), JSON.stringify(snap));
  } catch {
    // ignore
  }
}

// ─── 活跃会话实时缓存（按 doctorInstanceId 分桶） ─────────────────────────────
// 用于「刷新页面 / 复用 active 实例」时无缝恢复对话历史
function loadLiveMessages(doctorId: string): DoctorMsg[] | null {
  if (!doctorId) return null;
  try {
    const raw = localStorage.getItem(getLiveMessagesKey(doctorId));
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    const cleaned = sanitizeLoadedMessages(parsed);
    return cleaned.length > 0 ? cleaned : null;
  } catch {
    return null;
  }
}

function saveLiveMessages(doctorId: string, msgs: DoctorMsg[]) {
  if (!doctorId) return;
  // 统一在写入入口做一次清洗（防御性：调用方忘记过滤也不会污染缓存）
  const persistable = purgeForPersist(msgs);
  if (persistable.length === 0) {
    // 没有可持久化内容时，主动清掉缓存，避免残留旧脏数据
    try {
      localStorage.removeItem(getLiveMessagesKey(doctorId));
    } catch {
      /* ignore */
    }
    return;
  }
  try {
    localStorage.setItem(getLiveMessagesKey(doctorId), JSON.stringify(persistable));
  } catch {
    // ignore（超额等异常静默；后续考虑 LRU/截断）
  }
}

function clearLiveMessages(doctorId: string) {
  if (!doctorId) return;
  try {
    localStorage.removeItem(getLiveMessagesKey(doctorId));
  } catch {
    // ignore
  }
}

// ─── 活跃会话的诊断阶段持久化（按 doctorInstanceId 分桶）──────────────────────
// 用于「切换页面 / 刷新」后还原 SDK 插槽（如「开始修复」按钮）的可见性
// MOCK 标注：实际应由后端 Session 状态机维护，前端从轮询接口反推
function getLivePhaseKey(doctorId: string) {
  return `oc.doctor.live.phase.${doctorId}`;
}

function loadLivePhase(doctorId: string): DiagPhase | null {
  if (!doctorId) return null;
  try {
    const raw = localStorage.getItem(getLivePhaseKey(doctorId));
    if (!raw) return null;
    // 仅接受白名单值，避免脏数据导致状态错乱
    if (raw === "diagnosing" || raw === "summary_ready" || raw === "repairing" || raw === "done" || raw === "idle") {
      return raw;
    }
    return null;
  } catch {
    return null;
  }
}

function saveLivePhase(doctorId: string, phase: DiagPhase) {
  if (!doctorId) return;
  try {
    localStorage.setItem(getLivePhaseKey(doctorId), phase);
  } catch {
    // ignore
  }
}

function clearLivePhase(doctorId: string) {
  if (!doctorId) return;
  try {
    localStorage.removeItem(getLivePhaseKey(doctorId));
  } catch {
    // ignore
  }
}

// ─── 活跃会话的「快照已创建」标志持久化（按 doctorInstanceId 分桶）────────────
// 用于「切换页面 / 刷新」后还原结束诊断弹窗里的「回滚到诊断前快照」勾选项可见性
// MOCK 标注：实际应由后端 Session 元数据返回（hasSnapshot 字段）
function getLiveSnapshotKey(doctorId: string) {
  return `oc.doctor.live.snapshot.${doctorId}`;
}

function loadLiveSnapshotCreated(doctorId: string): boolean {
  if (!doctorId) return false;
  try {
    return localStorage.getItem(getLiveSnapshotKey(doctorId)) === "true";
  } catch {
    return false;
  }
}

function saveLiveSnapshotCreated(doctorId: string, created: boolean) {
  if (!doctorId) return;
  try {
    localStorage.setItem(getLiveSnapshotKey(doctorId), created ? "true" : "false");
  } catch {
    // ignore
  }
}

function clearLiveSnapshotCreated(doctorId: string) {
  if (!doctorId) return;
  try {
    localStorage.removeItem(getLiveSnapshotKey(doctorId));
  } catch {
    // ignore
  }
}

// ─── 活跃会话的「最后活跃时间戳」持久化（按 doctorInstanceId 分桶）──────────
// 用途：把"10 分钟无操作自动结束"的判定基线从瞬态 setTimeout 迁移到时间戳。
// 关键：组件卸载会清掉 setTimeout，导致用户切走后实例永远不会自动销毁（僵尸实例）。
// 切回时按 lastActiveAt 计算剩余时间，已超时则立即触发结束流程。
// MOCK 标注：实际应由后端 Session lastActiveAt 字段维护，前端轮询。
const AUTO_END_MS = 10 * 60 * 1000;

function getLiveLastActiveKey(doctorId: string) {
  return `oc.doctor.live.lastActive.${doctorId}`;
}

function loadLiveLastActive(doctorId: string): number | null {
  if (!doctorId) return null;
  try {
    const raw = localStorage.getItem(getLiveLastActiveKey(doctorId));
    if (!raw) return null;
    const n = Number(raw);
    return Number.isFinite(n) && n > 0 ? n : null;
  } catch {
    return null;
  }
}

function saveLiveLastActive(doctorId: string, ts: number) {
  if (!doctorId) return;
  try {
    localStorage.setItem(getLiveLastActiveKey(doctorId), String(ts));
  } catch {
    // ignore
  }
}

function clearLiveLastActive(doctorId: string) {
  if (!doctorId) return;
  try {
    localStorage.removeItem(getLiveLastActiveKey(doctorId));
  } catch {
    // ignore
  }
}

// ─── 输入框草稿持久化（按 doctorInstanceId 分桶）──────────────────────────────
// 用户输入到一半切走后切回，输入框应保留草稿。
function getLiveDraftKey(doctorId: string) {
  return `oc.doctor.live.draft.${doctorId}`;
}

function loadLiveDraft(doctorId: string): string {
  if (!doctorId) return "";
  try {
    return localStorage.getItem(getLiveDraftKey(doctorId)) ?? "";
  } catch {
    return "";
  }
}

function saveLiveDraft(doctorId: string, draft: string) {
  if (!doctorId) return;
  try {
    if (draft) {
      localStorage.setItem(getLiveDraftKey(doctorId), draft);
    } else {
      // 空草稿直接清掉，避免长期残留
      localStorage.removeItem(getLiveDraftKey(doctorId));
    }
  } catch {
    // ignore
  }
}

function clearLiveDraft(doctorId: string) {
  if (!doctorId) return;
  try {
    localStorage.removeItem(getLiveDraftKey(doctorId));
  } catch {
    // ignore
  }
}

// ─── 主组件 DoctorChatCard ────────────────────────────────────────────────────
// export 供 OpenClawDetailGuide 复用（引导页需要在管控端"允许"时展示同款完整交互）。
export function DoctorChatCard({ instanceId, instanceName }: { instanceId: string; instanceName: string }) {
  // ─── 状态：实例与诊断阶段 ────────────────────────────────────────────────────
  const [instanceStatus, setInstanceStatus] = useState<InstanceStatus>("none");
  const [diagPhase, setDiagPhase] = useState<DiagPhase>("idle");
  const [doctorInstanceId, setDoctorInstanceId] = useState<string>(""); // 龙虾医生实例 ID
  const [snapshotCreated, setSnapshotCreated] = useState(false);        // 本次是否创建了快照

  // ─── 状态：消息与历史 Session ────────────────────────────────────────────────
  const [messages, setMessages] = useState<DoctorMsg[]>([]);
  const [historySession, setHistorySession] = useState<SessionSnapshot | null>(() =>
    loadHistorySession(instanceId)
  );

  // ─── 状态：输入框与 UI ───────────────────────────────────────────────────────
  const [input, setInput] = useState("");
  const [isTyping, setIsTyping] = useState(false);
  const [isStreaming, setIsStreaming] = useState(false);
  const abortControllerRef = useRef<AbortController | null>(null);
  const [showCommands, setShowCommands] = useState(false); // 指令库下拉
  const commandsRef = useRef<HTMLDivElement | null>(null);

  // ─── 状态：弹窗 ──────────────────────────────────────────────────────────────
  const [showStartModal, setShowStartModal] = useState(false);  // 开始诊断弹窗（仅承担渲染独立选项）
  // 诊断启动选项：授权 / 快照 —— 两个完全独立的可选项，由 useDiagnosisOptions 统一管理
  const diagOptions = useDiagnosisOptions(instanceId);
  const [showEndModal, setShowEndModal] = useState(false);      // 结束诊断弹窗
  const [rollbackChecked, setRollbackChecked] = useState(false);// 结束弹窗内的回滚勾选项
  const [conflictInfo, setConflictInfo] = useState<{
    instanceId: string;
    instanceName: string;
  } | null>(null);   // 单实例冲突信息（冲突中的实例）

  // ─── 状态：是否被管控端/他人占用（从服务端查询）─────────────────────────────
  const [occupiedByOther, setOccupiedByOther] = useState(false);
  const [occupiedInitiatorType, setOccupiedInitiatorType] = useState<InitiatorType | null>(null);

  // 查询当前 Agent 是否被他人诊断中（服务端）
  useEffect(() => {
    let cancelled = false;
    const checkOccupied = async () => {
      const callerId = getCallerId("user");
      const result = await queryDiagnosisStatus(instanceId, callerId);
      if (cancelled) return;
      if (result.active && !result.isMine) {
        setOccupiedByOther(true);
        setOccupiedInitiatorType(result.session?.initiatorType ?? null);
      } else {
        setOccupiedByOther(false);
        setOccupiedInitiatorType(null);
      }
    };
    // 只在没有本地 active 实例时检查
    if (instanceStatus === "none" || instanceStatus === "ended") {
      checkOccupied();
    }
    return () => { cancelled = true; };
  }, [instanceId, instanceStatus]);

  // ─── 状态：自动结束计时器 ────────────────────────────────────────────────────
  const autoEndTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // ─── 守卫：本次组件生命周期内是否已自动发送过首条提示词 ─────────────────────
  // 用于防止「刚创建实例 → autoSend → 用户刷新页面 → 恢复时再次 autoSend」导致重复
  const didAutoSendRef = useRef(false);

  // ─── refs ────────────────────────────────────────────────────────────────────
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // ─── 滚动到底部 ──────────────────────────────────────────────────────────────
  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [messages, isTyping, scrollToBottom]);

  // ─── 实时缓存对话历史：active 期间每次 messages 变更都同步到 localStorage ───
  // 用于刷新 / 重进页面时无缝恢复对话；ended/none 时不缓存（避免污染下一轮）
  // 写入侧由 saveLiveMessages 内部统一过滤 transient 系统消息和 loading 占位，
  // 这里只负责"何时写"（active 状态 + 有 doctorId）
  useEffect(() => {
    if (instanceStatus !== "active" || !doctorInstanceId) return;
    if (messages.length === 0) return;
    saveLiveMessages(doctorInstanceId, messages);
  }, [messages, instanceStatus, doctorInstanceId]);

  // 持久化 diagPhase（与 messages 同生命周期）
  // 关键：切换页面后组件会卸载，diagPhase 会丢失；持久化后还原可恢复 SDK 插槽
  useEffect(() => {
    if (instanceStatus !== "active" || !doctorInstanceId) return;
    saveLivePhase(doctorInstanceId, diagPhase);
  }, [diagPhase, instanceStatus, doctorInstanceId]);

  // 持久化 snapshotCreated（与 diagPhase 同生命周期）
  // 关键：切换页面后丢失会导致结束诊断弹窗里的「回滚到诊断前快照」勾选项消失
  useEffect(() => {
    if (instanceStatus !== "active" || !doctorInstanceId) return;
    saveLiveSnapshotCreated(doctorInstanceId, snapshotCreated);
  }, [snapshotCreated, instanceStatus, doctorInstanceId]);

  // 持久化输入框草稿（用户切走再回来不丢内容）
  // 注意：仅在 active 时持久化，避免污染下一轮；空字符串会自动清掉
  useEffect(() => {
    if (instanceStatus !== "active" || !doctorInstanceId) return;
    saveLiveDraft(doctorInstanceId, input);
  }, [input, instanceStatus, doctorInstanceId]);

  // ─── 页面初始化：MOCK 查询当前用户是否已有 active 实例 ────────────────────────
  useEffect(() => {
    // MOCK: 实际应该是 GET /api/doctor/instance?agentInstanceId=xxx
    const raw = localStorage.getItem(ACTIVE_INSTANCE_KEY);
    if (raw) {
      try {
        const cur = JSON.parse(raw) as { agentInstanceId: string; doctorInstanceId: string; status: InstanceStatus };
        if (cur.agentInstanceId === instanceId && cur.status === "active") {
          // 复用已有 active 实例：不弹授权，直接进入对话状态
          setDoctorInstanceId(cur.doctorInstanceId);
          setInstanceStatus("active");
          // 还原诊断阶段：从持久化缓存读取，未命中则按"自由对话"对待
          // 注意：repairing/diagnosing 是中间瞬态，刷新/切换后无法恢复后台轮询，统一降级为 done
          const cachedPhase = loadLivePhase(cur.doctorInstanceId);
          const restoredPhase: DiagPhase =
            cachedPhase === "summary_ready" ? "summary_ready" :
            cachedPhase === "diagnosing" || cachedPhase === "repairing" ? "done" :
            cachedPhase === "done" || cachedPhase === "idle" ? cachedPhase :
            "done";
          setDiagPhase(restoredPhase);
          // 还原快照已创建标志（决定结束诊断弹窗里的「回滚」勾选项是否可见）
          setSnapshotCreated(loadLiveSnapshotCreated(cur.doctorInstanceId));

          // 守卫：复用场景下视为"自动首发已完成"，避免任何分支再次注入首条提示词
          didAutoSendRef.current = true;

          // 加载实时缓存对话历史（loadLiveMessages 内部已做 sanitize 清洗）
          const cached = loadLiveMessages(cur.doctorInstanceId);
          const hasRealHistory =
            !!cached &&
            cached.some((m) => m.kind === "user" || m.kind === "assistant");

          // 「已恢复 / 已复用上次未结束的诊断会话」这类系统提示已下架：
          //   ① 当前 mock 的恢复链路并不可靠（消息缓存、phase、快照标志、lastActive 多源容易不同步），
          //      用文字承诺"已恢复"会让用户产生"会话状态完全等于离开前"的错误心智模型，
          //      一旦实际表现有差异（如缺一条流式回复 / 草稿未还原 / SDK 插槽未出现），用户会觉得被误导；
          //   ② 真实历史本身就是最强的视觉信号——对话气泡直接出现 = 用户立刻知道"上次的对话还在"，
          //      不需要再用一条系统提示重复说明；无历史时则保持对话区为空，用户输入即开始新会话。
          // 待恢复链路真正稳定（接入后端 Session 状态机）后再考虑加回提示。
          if (hasRealHistory && cached) {
            setMessages(cached);
          } else {
            setMessages([]);
          }

          // 还原输入框草稿（用户切走时未发送的内容）
          const draft = loadLiveDraft(cur.doctorInstanceId);
          if (draft) setInput(draft);

          // 自动结束计时续时：按 lastActiveAt 计算剩余时间
          // - 已超时：立即触发结束流程（与超时一致体验，不让僵尸实例存活）
          // - 未超时：按剩余时间重启 setTimeout
          // - 无记录：按完整 AUTO_END_MS 重置（兼容老缓存）
          const lastActive = loadLiveLastActive(cur.doctorInstanceId);
          if (lastActive !== null) {
            const elapsed = Date.now() - lastActive;
            const remaining = AUTO_END_MS - elapsed;
            if (remaining <= 0) {
              // 已超时：异步触发，确保状态先就位（setInstanceStatus active 已上面 set 过）
              setTimeout(() => {
                setMessages((prev) => [
                  ...prev,
                  doctorMsg("您已超过 10 分钟未操作，本次诊断已自动结束。如需继续排查，请重新点击「开始诊断」。", true),
                ]);
                const snap: SessionSnapshot = {
                  endedAt: new Date().toLocaleString("zh-CN", { hour12: false }),
                  instanceId: cur.doctorInstanceId,
                  agentInstanceId: instanceId,
                  messages: [],  // messages 还在异步还原中，留空（实际可在 callback 内取最新）
                };
                saveHistorySession(instanceId, snap);
                setHistorySession(snap);
                localStorage.removeItem(ACTIVE_INSTANCE_KEY);
                clearLiveMessages(cur.doctorInstanceId);
                clearLivePhase(cur.doctorInstanceId);
                clearLiveSnapshotCreated(cur.doctorInstanceId);
                clearLiveLastActive(cur.doctorInstanceId);
                clearLiveDraft(cur.doctorInstanceId);
                setInstanceStatus("ended");
              }, 100);
            } else {
              restartAutoEndTimer(remaining);
            }
          } else {
            // 兼容场景：之前已是 active 但没记 lastActiveAt（老版本缓存）
            // 视为"刚刚活跃"，写入当前时间并按完整时长计时
            saveLiveLastActive(cur.doctorInstanceId, Date.now());
            restartAutoEndTimer();
          }
          return;
        }
        if (cur.agentInstanceId === instanceId && cur.status === "creating") {
          // 仍在创建中：继续轮询
          setDoctorInstanceId(cur.doctorInstanceId);
          setInstanceStatus("creating");
          // 创建中状态恢复时，也展示一条 transient 提示，避免空白
          setMessages([doctorMsg("正在为您创建专属龙虾医生 Agent，预计 3-5 分钟，准备就绪后将立即开始检测，请稍候…")]);
          startCreatePolling(cur.doctorInstanceId);
          return;
        }
      } catch { /* ignore */ }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [instanceId]);

  // ─── 卸载：清理计时器 ────────────────────────────────────────────────────────
  useEffect(() => {
    return () => {
      if (autoEndTimerRef.current) clearTimeout(autoEndTimerRef.current);
    };
  }, []);

  // ─── 指令库：外点关闭 ────────────────────────────────────────────────────────
  useEffect(() => {
    if (!showCommands) return;
    const onDocClick = (e: MouseEvent) => {
      if (commandsRef.current && !commandsRef.current.contains(e.target as Node)) {
        setShowCommands(false);
      }
    };
    document.addEventListener("mousedown", onDocClick);
    return () => document.removeEventListener("mousedown", onDocClick);
  }, [showCommands]);

  // ─── 「开始诊断」入口：弹授权+快照确认弹窗 ────────────────────────────────────
  const handleStartDiagnosisClick = () => {
    // MOCK 单实例冲突检测
    const raw = localStorage.getItem(ACTIVE_INSTANCE_KEY);
    if (raw) {
      try {
        const cur = JSON.parse(raw) as {
          agentInstanceId: string;
          agentInstanceName?: string;
          status: InstanceStatus;
        };
        if (cur.status === "active" && cur.agentInstanceId !== instanceId) {
          setConflictInfo({
            instanceId: cur.agentInstanceId,
            instanceName: cur.agentInstanceName || cur.agentInstanceId,
          });
          return;
        }
      } catch { /* ignore */ }
    }
    // 授权 vs 快照 的生命周期是不同的：
    //   - 授权：Agent 级一次性事实（hasAuthorizedDiag + hasAskedAuth）—— 对应后端
    //     POST /openclaw/doctor/authorize 首次授权幂等语义，在当前 Agent 上
    //     授权过一次后永不再问；
    //   - 快照：每次诊断的独立决策（对应 /start 请求体的 snapshot 字段），
    //     必须每次都让用户重新选择。
    // 因此「是否弹窗」的判定不能简单按授权状态短路——只要"快照"这个每次需选的项存在，
    // 弹窗就必须出现，只是弹窗内部按授权状态决定要不要显示「授权」那一行。
    // 弹窗内的条件渲染见下方 ConfirmDialog 的渲染分支。
    //
    // ── 默认勾选策略 ──────────────────────────────────────────────────────
    // 每次打开「开始诊断」弹窗时，将「创建配置快照」「同意使用龙虾医生功能」两个勾选项
    // 强制重置为已勾选（覆盖之前持久化的用户偏好）。
    // 设计理由：
    //   ① 两项均为「保护性 / 合规性」操作（快照=出问题能回滚、授权=首次功能同意），
    //      默认选中等价于"安全优先"的引导；
    //   ② 用户仍可手动取消勾选——只是默认从"上次选过什么"变成"安全态启动"；
    //   ③ 授权项若未勾选，下方「确认开始」按钮灰化，用户必须同意才能继续
    //      （对应后端未授权时 /start 会被拒的硬约束）。
    diagOptions.setAuthorize(true);
    diagOptions.setSnapshot(true);
    setShowStartModal(true);
  };

  // ─── 弹窗内点「确认开始」：读取两个独立选项后启动 ─────────────────────────
  // 授权 / 快照 在此**完全平行**地处理，互不依赖；任何一个的取值都不影响另一个的执行路径。
  const handleStartConfirm = async () => {
    setShowStartModal(false);
    // 选项 1：配置快照（独立）—— 决定本次诊断是否创建可回滚快照
    setSnapshotCreated(diagOptions.snapshot);
    // 选项 2：授权诊断记录（独立）—— 当前为前端偏好，未来此处对接 PUT /api/agent/{id}/auth
    //                                    无论开关如何都不阻断诊断启动
    // 一次性授权语义：用户在弹窗里勾选了授权 → 写入"已授权"标记；
    // 不论勾没勾，只要弹窗显示过授权选项并点了确认，就写入"已问过"标记，
    // 后续不再就授权这件事打扰用户（即便他这次选择了不勾）。
    if (diagOptions.authorize) {
      markAuthorizedDiag(instanceId);
    }
    markAskedAuth(instanceId);
    await doCreateInstance();
  };

  // ─── 创建龙虾医生实例（写入共享 DB + 本地标记）──────────────────────────────
  const doCreateInstance = async () => {
    // 调用共享 DB 的 startDiagnosis，实现跨端互斥
    const callerId = getCallerId("user");
    const result = await startDiagnosis({
      agentInstanceId: instanceId,
      initiatorId: callerId,
      initiatorType: "user",
      snapshot: diagOptions.snapshot,
    });

    if (!result.success) {
      if (result.reason === "conflict") {
        setOccupiedByOther(true);
        setOccupiedInitiatorType("admin");
      }
      return;
    }

    const newDoctorId = result.doctorInstanceId!;
    setDoctorInstanceId(newDoctorId);
    setInstanceStatus("creating");
    setDiagPhase("idle");
    // 新建实例 → 重置自动首发守卫（允许本次 active 后触发一次 autoSend）
    didAutoSendRef.current = false;
    // 新建实例时清掉可能残留的旧缓存（极端场景：上次实例 ID 与新 ID 不同但旧 key 还在）
    clearLiveMessages(newDoctorId);
    clearLivePhase(newDoctorId);
    clearLiveSnapshotCreated(newDoctorId);
    clearLiveLastActive(newDoctorId);
    clearLiveDraft(newDoctorId);
    setMessages([
      doctorMsg("正在为您创建专属龙虾医生 Agent，预计 3-5 分钟，准备就绪后将立即开始检测，请稍候…"),
    ]);
    // 同时写入本地"全局当前实例"标记（用于本地单实例冲突检测）
    localStorage.setItem(
      ACTIVE_INSTANCE_KEY,
      JSON.stringify({ agentInstanceId: instanceId, agentInstanceName: instanceName, doctorInstanceId: newDoctorId, status: "creating" })
    );
    startCreatePolling(newDoctorId);
  };

  // ─── MOCK 轮询：1.5s 后实例变 active ─────────────────────────────────────────
  const startCreatePolling = (doctorId: string) => {
    setTimeout(() => {
      setInstanceStatus("active");
      localStorage.setItem(
        ACTIVE_INSTANCE_KEY,
        JSON.stringify({ agentInstanceId: instanceId, agentInstanceName: instanceName, doctorInstanceId: doctorId, status: "active" })
      );
      // 实例 Active → 自动注入第一条全面检测提示词（仅在本次生命周期未触发过时）
      // 守卫场景：用户在 creating 状态刷新页面 → 恢复时也不会重复 autoSend
      if (!didAutoSendRef.current) {
        // 正常首发路径：不再单独插一条「龙虾医生已就绪」提示——
        // 紧接着 autoSendFirstPrompt 会发出「您好，我是龙虾医生 🦞 收到，开始全面检测…」，
        // 那条本身就是最强的"已就绪"信号（既自我介绍、又表明已开始干活），
        // 单独再插一条会语义重复、视觉冗余。
        setTimeout(() => autoSendFirstPrompt(), 400);
      } else {
        // 刷新恢复 creating 态 → 守卫跳过 autoSend：
        // 这种用户上一次已经走过开场流程，但本次"创建中→已就绪"的转折仍需明确告知，
        // 用一条简短的医生口吻文本承接，让对话不至于在"准备中…"卡住后无声变成可输入态。
        setMessages((prev) => [
          ...prev,
          doctorMsg("龙虾医生已准备就绪，您可以继续与我对话了。"),
        ]);
      }
      // 启动 10 分钟无操作计时
      restartAutoEndTimer();
    }, 1500); // MOCK: 实际应轮询 GET /api/doctor/instance/{id}
  };

  // ─── 自动注入第一条提示词（不展示输入过程，直接显示为已发送的用户消息）──────
  const autoSendFirstPrompt = () => {
    // 二次保护：防止任何路径下重复触发
    if (didAutoSendRef.current) return;
    didAutoSendRef.current = true;
    setMessages((prev) => [...prev, { kind: "user", text: AUTO_FIRST_PROMPT }]);
    setDiagPhase("diagnosing");
    setIsTyping(true);
    // MOCK 龙虾医生检测响应
    setTimeout(() => {
      setMessages((prev) => [
        ...prev,
        {
          kind: "assistant",
          parts: [
            { type: "text", text: "您好，我是龙虾医生 🦞 收到，开始全面检测，将逐项实时输出结果…" },
          ],
        },
      ]);
    }, 600);
    setTimeout(() => {
      setIsTyping(false);
      setMessages((prev) => [
        ...prev,
        {
          kind: "assistant",
          parts: [
            { type: "text", text: "全部检测项执行完成，共 5 项，发现 2 项异常：" },
            {
              type: "check_list",
              items: [
                { label: "网络连通性", status: "ok" },
                { label: "模型接口", status: "ok" },
                { label: "飞书通道", status: "error", detail: "认证 Token 已过期" },
                { label: "QQ 通道", status: "ok" },
                { label: "tavily-search", status: "error", detail: "进程崩溃" },
              ],
            },
            { type: "text", text: "异常项原因与修复方案：\n1. 飞书通道 —— Token 已过期；建议刷新通道授权 Token。\n2. tavily-search —— 进程已崩溃；建议重启技能进程并校验本地端口占用。\n\n请回复确认是否执行修复，或直接点击下方「开始修复」。" },
          ],
        },
      ]);
      setDiagPhase("summary_ready");
    }, 2400);
  };

  // ─── 「开始修复」按钮（SDK 插槽注入）──────────────────────────────────────────
  const handleStartRepair = () => {
    // 防御：会话已结束（10 分钟无操作自动结束 / 用户主动结束）时直接拦截
    // 视觉层已在渲染条件里隐藏按钮，这里再兜一层，防止 race / 缓存还原 / 外部触发
    if (instanceStatus !== "active") {
      toast.error("当前诊断已结束，请重新开始诊断");
      return;
    }
    restartAutoEndTimer();
    // 发送提示词为用户消息
    setMessages((prev) => [
      ...prev,
      { kind: "user", text: "请按问题列表依次修复每个问题，每完成一项告知具体执行结果（成功则说明已修复内容，失败则说明失败原因）。若某项修复失败，不要中断整个流程，请继续修复后续问题。全部处理完毕后，汇总输出本次修复的成功项和失败项列表，并对失败项给出后续建议或排查方向。" },
    ]);
    setDiagPhase("repairing");
    setIsTyping(true);
    setMessages((prev) => [
      ...prev,
      doctorMsg("好的，我现在开始依次执行修复。即使中途某项失败，我也会继续执行后续项，请放心等待结果。"),
    ]);

    // MOCK 修复结果：飞书成功，tavily 失败
    setTimeout(() => {
      setIsTyping(false);
      const results: RepairResult[] = [
        { label: "飞书通道 Token 刷新", ok: true },
        { label: "tavily-search 进程重启", ok: false, reason: "技能依赖的本地端口被占用；后续建议：登录实例执行 lsof -i:<端口> 排查占用进程，释放端口后再次重试。" },
      ];
      setMessages((prev) => [
        ...prev,
        {
          kind: "assistant",
          parts: [
            { type: "text", text: "本次修复执行完毕，逐项结果如下：" },
            { type: "repair_summary", results },
            { type: "text", text: "汇总：成功 1 项 / 失败 1 项。失败项已附后续排查方向，如需我协助进一步定位，可继续在下方提问。" },
          ],
        },
      ]);
      setDiagPhase("done");
    }, 3000);
  };

  // ─── 「结束诊断」按钮：弹窗 ──────────────────────────────────────────────────
  // 结束诊断弹窗内的「回滚到诊断前快照」保留 checkbox（默认不勾选）——用户如需
  // 回滚到诊断前配置，需显式勾选后再点「确认结束」，避免误操作触发网关重启。
  const handleEndClick = () => {
    setRollbackChecked(false);
    setShowEndModal(true);
  };

  // ─── 弹窗内点「确认结束」：立即销毁会话组件 + 异步销毁实例 ──────────────────
  const handleEndConfirm = () => {
    setShowEndModal(false);
    if (autoEndTimerRef.current) clearTimeout(autoEndTimerRef.current);

    // 持久化历史 Session（MOCK COS）——同样要清洗，避免历史快照里残留 transient/loading
    const snap: SessionSnapshot = {
      endedAt: new Date().toLocaleString("zh-CN", { hour12: false }),
      instanceId: doctorInstanceId,
      agentInstanceId: instanceId,
      messages: purgeForPersist(messages),
    };
    saveHistorySession(instanceId, snap);
    setHistorySession(snap);

    // 立即前端销毁
    setInstanceStatus("destroying");

    // 通知共享 DB 结束诊断（让管控端感知到诊断已结束）
    endDiagnosis({ agentInstanceId: instanceId, doctorInstanceId, rollback: rollbackChecked });

    // MOCK 后端异步销毁：1s 后清理状态
    setTimeout(() => {
      localStorage.removeItem(ACTIVE_INSTANCE_KEY);
      clearLiveMessages(doctorInstanceId);
      clearLivePhase(doctorInstanceId);
      clearLiveSnapshotCreated(doctorInstanceId);
      clearLiveLastActive(doctorInstanceId);
      clearLiveDraft(doctorInstanceId);
      setInstanceStatus("ended");
      setMessages((prev) => [
        ...prev,
        // 「已回滚到诊断前快照」是用户主动勾选的破坏性操作，必须持久化到历史 Session，
        // 让用户日后回看历史会话时仍能看到当时确实回滚了——这是审计/追溯诉求，非过程提示。
        ...(rollbackChecked ? [doctorMsg("已为您回滚到诊断前的配置快照。", true)] : []),
        // 「下线 / 道别」气泡：transient（默认）——历史 Session 视图无需重复展示。
        doctorMsg("本次诊断已结束，感谢您的使用。期待下次为您服务！"),
      ]);
    }, 1000);
  };

  // ─── 10 分钟无操作自动结束 ───────────────────────────────────────────────────
  // 关键设计：判定基线是持久化的 lastActiveAt 时间戳，而非瞬态 setTimeout。
  // 否则用户切走后 setTimeout 被 unmount 清掉，实例会变成永不释放的僵尸。
  // setTimeout 仅作为本次组件生命周期内的"剩余时间到点提醒器"。
  // remainingMs 可选：用于挂载时按"已用时间"接续计时，未传则按完整 AUTO_END_MS 重置。
  const restartAutoEndTimer = (remainingMs?: number) => {
    if (autoEndTimerRef.current) clearTimeout(autoEndTimerRef.current);
    // 主动调用（用户操作）→ 刷新 lastActiveAt；挂载续时 → 不刷新
    if (remainingMs === undefined && doctorInstanceId) {
      saveLiveLastActive(doctorInstanceId, Date.now());
    }
    const wait = remainingMs !== undefined ? Math.max(0, remainingMs) : AUTO_END_MS;
    autoEndTimerRef.current = setTimeout(() => {
      // MOCK: 实际由后端定时任务销毁，前端通过轮询发现实例不再 active 时提示
      if (instanceStatus !== "active") return;
      setMessages((prev) => [
        ...prev,
        doctorMsg("您已超过 10 分钟未操作，本次诊断已自动结束。如需继续排查，请重新点击「开始诊断」。", true),
      ]);
      // 通知共享 DB 结束诊断
      endDiagnosis({ agentInstanceId: instanceId, doctorInstanceId, rollback: false });
      // 自动持久化 Session（清洗后）
      const snap: SessionSnapshot = {
        endedAt: new Date().toLocaleString("zh-CN", { hour12: false }),
        instanceId: doctorInstanceId,
        agentInstanceId: instanceId,
        messages: purgeForPersist(messages),
      };
      saveHistorySession(instanceId, snap);
      setHistorySession(snap);
      localStorage.removeItem(ACTIVE_INSTANCE_KEY);
      clearLiveMessages(doctorInstanceId);
      clearLivePhase(doctorInstanceId);
      clearLiveSnapshotCreated(doctorInstanceId);
      clearLiveLastActive(doctorInstanceId);
      clearLiveDraft(doctorInstanceId);
      setInstanceStatus("ended");
    }, wait);
  };

  // ─── AI 自由对话（修复完成后）─────────────────────────────────────────────────
  const callAI = async (userText: string) => {
    setIsTyping(false);
    setIsStreaming(true);
    const controller = new AbortController();
    abortControllerRef.current = controller;

    const history = messages
      .filter((m) => m.kind === "user" || m.kind === "assistant")
      .map((m) => {
        if (m.kind === "user") return { role: "user" as const, content: m.text };
        const text = m.parts.filter((p) => p.type === "text").map((p) => (p as { type: "text"; text: string }).text).join("\n");
        return { role: "assistant" as const, content: text };
      });

    const systemPrompt = `你是龙虾医生，ClawPro 平台的 AI 运维助手。用简洁中文回复，不超过 150 字。`;

    try {
      // 加 loading: true 标记，避免在响应到达前刷新页面时把 "…" 占位永久持久化
      setMessages((prev) => [...prev, { kind: "assistant", parts: [{ type: "text", text: "…" }], loading: true }]);
      const resp = await fetch("/api/ai/v1/chat/completions", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          model: "gpt-4.1-mini",
          messages: [
            { role: "system", content: systemPrompt },
            ...history,
            { role: "user", content: userText },
          ],
          max_tokens: 400,
        }),
        signal: controller.signal,
      });
      if (!resp.ok) throw new Error(`API error ${resp.status}`);
      const data = await resp.json();
      const text = data.choices?.[0]?.message?.content ?? "（无回复）";
      setMessages((prev) => {
        const updated = [...prev];
        const last = updated[updated.length - 1];
        if (last.kind === "assistant") {
          // 响应到达：去掉 loading 标记，正式落盘到缓存
          updated[updated.length - 1] = { kind: "assistant", parts: [{ type: "text", text }] };
        }
        return updated;
      });
    } catch (err: unknown) {
      if (!(err instanceof Error && err.name === "AbortError")) {
        // 替换掉之前的 loading 占位（而不是再 append 一条），避免出现"…"和错误提示并存
        setMessages((prev) => {
          const updated = [...prev];
          const last = updated[updated.length - 1];
          if (last && last.kind === "assistant" && last.loading) {
            updated[updated.length - 1] = {
              kind: "assistant",
              parts: [{ type: "text", text: "抱歉，我暂时无法回复，请稍后再试。" }],
            };
            return updated;
          }
          return [
            ...prev,
            { kind: "assistant", parts: [{ type: "text", text: "抱歉，我暂时无法回复，请稍后再试。" }] },
          ];
        });
      } else {
        // 用户主动 abort：清掉 loading 占位
        setMessages((prev) => {
          const last = prev[prev.length - 1];
          if (last && last.kind === "assistant" && last.loading) {
            return prev.slice(0, -1);
          }
          return prev;
        });
      }
    } finally {
      setIsStreaming(false);
      abortControllerRef.current = null;
    }
  };

  // ─── 指令处理（与 ChatView 的 4 条通用指令保持等价语义）
  // 设计原则：
  //   1. 所有指令都消耗 user 输入（即把指令本身作为 user 气泡上屏），与 ChatView 行为一致；
  //   2. 4 条指令均为前端 mock 反馈，不发请求；
  //   3. 「/new」清空消息 + 重新跑首条全面检测提示词（语义=新会话）。
  // 反馈 UI 统一：以「龙虾医生 assistant 气泡」呈现回复（左对齐自然语言文字 / 模块级 doctorMsg），
  //   而非居中灰色胶囊的 system 系统提示——参考 LightClaw / Claude / Cursor 等真实助手对话体验，
  //   让指令反馈与上下文里的对话气泡同源同质，给用户「就是医生本人在回应」的统一心智。
  // 持久化：四条反馈均为「过程性回复」，无需归档到历史 Session（doctorMsg 默认 transient）。
  // 返回值 true = 已被指令分发处理，调用方不要再走 LLM 路径。
  const handleDoctorCommand = (cmd: string): boolean => {
    // 1) /new：清空当前消息 + 重新跑首条全面检测提示词
    //    复用 autoSendFirstPrompt（重置守卫位 didAutoSendRef 才能再次触发）
    //    给一条简短反馈再交棒给 autoSend——参照 LightClaw「好的，已新建会话…」的真实表现。
    if (cmd === "/new") {
      setMessages([
        doctorMsg("好的，已为您新建会话，之前的对话上下文已清空，我们重新开始吧！"),
      ]);
      didAutoSendRef.current = false;
      setTimeout(() => autoSendFirstPrompt(), 200);
      return true;
    }
    // 2) /compact
    if (cmd === "/compact") {
      setMessages((prev) => [
        ...prev,
        doctorMsg("好的，我已为您压缩了对话上下文，仅保留最近 10 条对话作为后续推理依据。"),
      ]);
      return true;
    }
    // 3) /status：上报当前实例状态 + 诊断阶段（这里取的是已有 state，不发请求）
    if (cmd === "/status") {
      const statusLabel = ({
        none: "未启动", creating: "创建中", active: "运行中",
        destroying: "销毁中", ended: "已结束",
      } as const)[instanceStatus];
      const phaseLabel = ({
        idle: "待诊断", diagnosing: "检测中", summary_ready: "等待开始修复",
        repairing: "修复中", done: "自由对话中",
      } as const)[diagPhase];
      setMessages((prev) => [
        ...prev,
        doctorMsg(`当前我的运行状态是「${statusLabel}」，诊断阶段处于「${phaseLabel}」。`),
      ]);
      return true;
    }
    // 4) /commands：打印全部指令清单
    if (cmd === "/commands") {
      const list = DOCTOR_COMMAND_LIST.map((x) => `${x.command} — ${x.label}`).join("\n");
      setMessages((prev) => [
        ...prev,
        doctorMsg(`这是您当前可以使用的指令清单：\n${list}`),
      ]);
      return true;
    }
    return false;
  };

  const handleSend = () => {
    const text = input.trim();
    if (!text || inputDisabled) return;
    setInput("");
    if (textareaRef.current) textareaRef.current.style.height = "auto";
    // 指令分发：以 "/" 开头且首词命中 7 条指令之一时，走专属处理而非 LLM
    // 注意：先把 user 气泡上屏，与 ChatView 的"指令也是用户消息"语义一致
    const firstWord = text.split(/\s+/)[0];
    const isCommand = DOCTOR_COMMAND_LIST.some((x) => x.command === firstWord);
    setMessages((prev) => [...prev, { kind: "user", text }]);
    restartAutoEndTimer();
    if (isCommand && handleDoctorCommand(firstWord)) return;
    callAI(text);
  };

  const handleStopStreaming = () => {
    abortControllerRef.current?.abort();
    setIsStreaming(false);
  };

  // ─── 渲染单条消息（视觉与 ChatView 对齐）─────────────────────────────────────
  const renderMsg = (msg: DoctorMsg, idx: number) => {
    if (msg.kind === "system") {
      return (
        <div key={idx} className="flex justify-center">
          <span className="px-3 py-1 rounded-full bg-gray-100 text-xs text-gray-400">{msg.text}</span>
        </div>
      );
    }
    if (msg.kind === "user") {
      // 与 ChatView 同款：max-w-[78%] px-4 py-2.5 rounded-[4px] bg-gray-100
      return (
        <div key={idx} className="flex justify-end">
          <div className="max-w-[78%] px-4 py-2.5 rounded-[4px] bg-gray-100 text-sm text-gray-900 leading-relaxed whitespace-pre-wrap">
            {msg.text}
          </div>
        </div>
      );
    }
    return (
      <div key={idx} className="flex justify-start">
        {/*
          字体 / 颜色 / 行高 / 换行 / 宽度 全部对齐 LightClaw ChatView 标杆样式：
            text-sm + text-gray-900 + leading-relaxed + whitespace-pre-wrap + max-w-[90%]
          关键修正：
            ① 字色之前是 text-gray-800（更浅），改回 text-gray-900 与 LightClaw 完全一致；
            ② 补 whitespace-pre-wrap，让 doctorMsg 里的 "\n" 正确换行（如 /commands 指令清单）；
            ③ 限宽 max-w-[90%]，避免长文本横跨整个面板观感发散；
            ④ 单 text part 不再外包 <p>——直接落字，让 whitespace-pre-wrap 在 div 上一次性生效；
               仅当存在多个 part（text + check_list / repair_summary 复合内容）时才用 <p mt-2> 分段，
               以保证视觉层级清晰。
        */}
        <div className="max-w-[90%] text-sm text-gray-900 leading-relaxed whitespace-pre-wrap">
          {msg.parts.length === 1 && msg.parts[0].type === "text" ? (
            msg.parts[0].text
          ) : (
            msg.parts.map((part, pi) => {
              if (part.type === "text") return <p key={pi} className={pi > 0 ? "mt-2" : ""}>{part.text}</p>;
              if (part.type === "check_list") return <CheckList key={pi} items={part.items} />;
              if (part.type === "repair_summary") return <RepairSummary key={pi} results={part.results} />;
              return null;
            })
          )}
        </div>
      </div>
    );
  };

  // ─── 派生：输入框与按钮状态 ──────────────────────────────────────────────────
  const isActive = instanceStatus === "active";
  const isCreating = instanceStatus === "creating";
  const isEnded = instanceStatus === "ended";
  const isDestroying = instanceStatus === "destroying";

  const inputDisabled =
    !isActive ||
    isStreaming ||
    isTyping ||
    diagPhase === "diagnosing" ||
    diagPhase === "repairing";

  const inputPlaceholder =
    instanceStatus === "none" ? "点击「开始诊断」创建龙虾医生 Agent" :
    isCreating ? "龙虾医生正在就位…" :
    isEnded ? "当前诊断已结束，点击「开始诊断」开启新会话" :
    isDestroying ? "龙虾医生正在离场…" :
    diagPhase === "diagnosing" ? "龙虾医生检测中，请稍候…" :
    diagPhase === "repairing" ? "正在执行修复…" :
    "向龙虾医生提问，或描述您遇到的问题…";

  // 主按钮（页面顶部）：开始诊断 / 再次诊断
  const showStartButton = instanceStatus === "none" || isEnded;
  const startButtonLabel = "开始诊断";

  // 是否显示对话区（active / destroying / ended 都需要展示已有消息）
  // 关键：必须把「historySession」也纳入显示条件——否则
  //   instanceStatus === "none" + messages.length === 0 + 有历史 Session 的场景下
  //   （即"上次诊断已结束并归档、本次还没点开始"），历史折叠卡片会因为
  //   外层对话区被屏蔽而无法渲染，造成"历史诊断记录消失"的体验回归。
  const showChatArea = instanceStatus !== "none" || messages.length > 0 || !!historySession;

  return (
    <div
      className="bg-white rounded-[4px] border border-gray-200 relative"
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      {/* ─── 开始诊断弹窗（仅承担渲染独立选项 + 启动按钮）─────────────────────
          授权 / 配置快照 在此完全独立、并列展示：
          - 都是用户自主 opt-in 的偏好（与"配置快照"对等）；
          - 关闭弹窗 / 取消时不会改变已保存的偏好；
          - 启动诊断时由 handleStartConfirm 分别独立处理两份取值。 */}
      <Dialog open={showStartModal} onOpenChange={(open) => { if (!open) setShowStartModal(false); }}>
        <DialogContent className="max-w-sm">
          <DialogHeader className="pb-1">
            <DialogTitle className="text-base font-semibold text-gray-900">
              开始诊断
            </DialogTitle>
          </DialogHeader>
          <div className="pt-0 pb-2 space-y-3">
            {/* 引导文案：字号 sm，比下方选项 xs 更大，形成层级 */}
            <p className="text-sm text-gray-700 leading-relaxed">
              即将创建龙虾医生 Agent，使用平台配置的诊断模型对当前 Agent 进行全面检测和修复。
            </p>

            {/* 授权 + 配置快照 合并到同一个气泡框中：
                外层一个卡片容器，内部两行勾选并列展示，仍保持数据/逻辑独立。
                授权行使用 hasAskedAuth 判断：本 Agent 已经被问过一次就不再询问，
                符合后端 POST /openclaw/doctor/authorize 的「首次授权、记录到实例、
                后续不再问」的幂等语义；
                快照行始终显示，因为快照是每次诊断的独立决策（对接 /start 请求体的
                snapshot 字段）。

                文案重写：原"授权平台使用本次诊断记录 / ... 用于平台优化龙虾医生能力"
                与后端接口语义不匹配（后端并无"数据使用同意"相关功能），调整为
                "同意使用龙虾医生功能"——对应后端 authorize 接口的真实语义：
                首次创建诊断节点前的功能使用同意书。 */}
            <div
              className="rounded-[4px]"
              style={{ border: "1px solid #EDEFF5", background: "#FFFFFF" }}
            >
              {!hasAskedAuth(instanceId) && (
                <DiagOptionRow
                  checked={diagOptions.authorize}
                  onChange={diagOptions.setAuthorize}
                  title="同意使用龙虾医生功能"
                  description="龙虾医生将在当前 Agent 上创建临时诊断节点，诊断结束后自动销毁。"
                />
              )}
              <DiagOptionRow
                checked={diagOptions.snapshot}
                onChange={diagOptions.setSnapshot}
                title="为本次诊断创建配置快照"
                description="勾选后，结束诊断时可一键回滚至 Agent 开始诊断前的状态。"
                extra="创建快照会触发网关重启，期间该 Agent 对外服务将短暂中断。"
              />
            </div>

            <p className="text-xs text-gray-400">初始化约需 3-5 分钟，请稍作等待。</p>
          </div>
          <div className="flex gap-2 pt-2 pb-6 justify-end">
            {/*
              按钮排布：按目标视觉——次要按钮「取消」在左、主按钮「确认开始」在右，
              整组靠右对齐（justify-end）。

              「确认开始」按钮的启用条件：
              ① 首次进入此 Agent（hasAskedAuth 为 false，授权行可见）：
                 必须先勾选「同意使用龙虾医生功能」才能进入下一步，否则灰化。
                 对应后端 POST /openclaw/doctor/authorize 的"首次必须授权"语义——
                 未授权的情况下 /start 接口也会拒绝（error: 'unauthorized'
                 之类）。
              ② 非首次（hasAskedAuth 为 true，授权行已隐藏）：
                 此时仅剩快照选项，而快照是可选项（不勾也能开始诊断），
                 故按钮永远可点。
              视觉：灰化采用 disabled 属性 + opacity 降低，悬停提示（title 属性）
              告知用户"需先同意使用龙虾医生功能"，避免用户困惑"按钮为什么点不动"。
            */}
            <Button
              variant="tenant-outline"
              size="claw-sm"
              className="text-xs"
              onClick={() => setShowStartModal(false)}
            >
              取消
            </Button>
            {(() => {
              const needAuth = !hasAskedAuth(instanceId);
              const confirmDisabled = needAuth && !diagOptions.authorize;
              return (
                <Button
                  variant="tenant-primary"
                  size="claw-sm"
                  className="text-xs"
                  onClick={handleStartConfirm}
                  disabled={confirmDisabled}
                  title={confirmDisabled ? "请先勾选「同意使用龙虾医生功能」" : undefined}
                >
                  确认开始
                </Button>
              );
            })()}
          </div>
        </DialogContent>
      </Dialog>

      {/* ─── 结束诊断弹窗（含可选回滚勾选） ───────────────────────────────── */}
      <Dialog open={showEndModal} onOpenChange={(open) => { if (!open) setShowEndModal(false); }}>
        <DialogContent className="max-w-sm">
          <DialogHeader className="pb-1">
            <DialogTitle className="text-base font-semibold text-gray-900">结束诊断</DialogTitle>
          </DialogHeader>
          <div className="pt-0 pb-2 space-y-3">
            <p className="text-sm text-gray-600 leading-relaxed">
              即将结束本次诊断，临时龙虾医生节点将被销毁。
            </p>
            {snapshotCreated && (
              <label
                className="flex items-start gap-2.5 cursor-pointer select-none rounded-[4px] px-3 py-2.5 transition-colors"
                style={{
                  border: "1px solid #EDEFF5",
                  background: "#FFFFFF",
                }}
              >
                <input
                  type="checkbox"
                  checked={rollbackChecked}
                  onChange={(e) => setRollbackChecked(e.target.checked)}
                  className="sr-only"
                />
                <span
                  aria-hidden
                  className="mt-0.5 flex-shrink-0 inline-flex items-center justify-center rounded-[3px] transition-all"
                  style={{
                    width: 14,
                    height: 14,
                    border: rollbackChecked ? "1px solid #1447E6" : "1px solid #C7CBD6",
                    background: rollbackChecked ? "#1447E6" : "#FFFFFF",
                  }}
                >
                  {rollbackChecked && (
                    <svg width="10" height="10" viewBox="0 0 12 12" fill="none">
                      <path d="M2.5 6.2L4.8 8.5L9.5 3.5" stroke="#FFFFFF" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                  )}
                </span>
                <div className="flex-1 min-w-0 space-y-1">
                  <p className="text-sm text-gray-700 leading-snug">回滚到诊断前快照</p>
                  <p className="text-xs text-gray-400 leading-relaxed">勾选后，将把 Agent 配置恢复到诊断开始前的状态。</p>
                  <p className="text-xs text-gray-400 leading-relaxed">回滚会触发网关重启，期间该 Agent 对外服务将短暂中断。</p>
                </div>
              </label>
            )}
          </div>
          <div className="flex gap-2 pt-2 pb-6 justify-end">
            <Button
              variant="tenant-outline"
              size="claw-sm"
              className="text-xs"
              onClick={() => setShowEndModal(false)}
            >
              取消
            </Button>
            <Button
              variant="tenant-primary"
              size="claw-sm"
              className="text-xs"
              onClick={handleEndConfirm}
            >
              确认结束
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* ─── 单实例冲突提示 ─────────────────────────────────────────────────── */}
      <AlertDialog open={!!conflictInfo} onOpenChange={(open) => { if (!open) setConflictInfo(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-base font-semibold text-gray-900">诊断会话冲突</AlertDialogTitle>
            <AlertDialogDescription asChild>
              <div className="text-sm text-gray-600 leading-relaxed">
                {conflictInfo ? (
                  <>
                    {/*
                      关键：把整段文本拼成一个连续字符串、避免 JSX 文本节点之间的换行/空格被渲染成
                      可折行空白——浏览器看到「Agent」后紧跟中文「（」就不会在这里折，从而把第一行
                      的剩余宽度真正利用起来去塞 ins-id；只有当行宽不够时，才会在中文/空格处折到下一行，
                      不会再出现「第一行只到 Agent 就空着 → 整组括号被压到第二行」的视觉浪费。

                      ID 与实例名之间用「 ｜ 」(全角竖线 U+FF5C) 分隔：
                        ① 视觉上明确区分「ID 是 ID、实例名是实例名」，避免读者误以为是一段连续字符串；
                        ② 两侧普通空格给浏览器一个明确的可折点——行宽不够时优先在这里折。
                      ID 自身加 whitespace-nowrap：英文 ID 串若被拆成 "ins-light\nclaw02" 阅读体验差，
                      作为最小不可拆单元处理；实例名（含中文）则保持默认折行行为。
                    */}
                    {`同一时间仅支持一个 Agent 进行诊断。Agent（`}
                    <span className="font-mono text-xs text-gray-900 whitespace-nowrap">{conflictInfo.instanceId}</span>
                    {` ｜ `}
                    <span className="text-gray-700">{conflictInfo.instanceName}</span>
                    {`）当前正在诊断中，请先结束其会话后再开始新的诊断。`}
                  </>
                ) : (
                  "同一时间仅支持一个 Agent 进行诊断。"
                )}
              </div>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <Button
              variant="tenant-primary"
              onClick={() => setConflictInfo(null)}
            >
              我知道了
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* ─── 标题栏 ──────────────────────────────────────────────────────────
          说明：整个 DoctorChatCard 已被外层 {lobsterDoctorEnabled && ...} 守卫，
          能渲染到这里即代表管控端已允许 → 无需再展示"未开启" Badge（截图2 设计稿）。 */}
      <div className="px-6 pt-5 pb-0">
        <div className="flex items-center gap-2">
          <h2 className="text-base font-semibold text-gray-900">龙虾医生</h2>
        </div>
      </div>

      {/* ─── 副标题 + 主按钮 ─────────────────────────────────────────────────── */}
      <div className="px-6 pt-3 pb-4">
        {/*
          副标题文案补充「回滚操作路径」说明——技术上「创建快照」走开始诊断接口、
          「回滚」走结束诊断接口，二者不能在诊断中途单独触发，导致用户在诊断进行中
          想"撤销改动"时找不到入口。把这条规则用单一静态文案前置告知，避免事后困惑。

          采用「用户立场 + 可选语气」而非「系统立场 + 限制语气」：
            "若诊断开始前已勾选..." → 用户视角的条件
            "结束诊断后可选择配置回滚" → 用户的可选动作（强调"可"，非"必须"）
          引号「创建配置快照」与开始诊断弹窗的勾选项标题完全一致，
          帮助用户把这句话与那个勾选项一一对应。
          有意采用静态文案而非按 snapshotCreated 条件分支：
            ① 已勾快照的用户读到 → 知道「结束诊断时可勾选回滚」；
            ② 未勾快照的用户读到 → 自然意识到本次无法回滚、下次记得勾；
          一句话覆盖两种场景，避免引入额外条件分支与状态读取。
        */}
        <p className="text-sm text-gray-500 mb-3">
          AI 智能诊断，帮助您发现并修复 Agent 运行问题。若诊断开始前已勾选「创建配置快照」，结束诊断后可选择配置回滚。
        </p>
        {showStartButton && !occupiedByOther && (
          <Button
            variant="tenant-primary"
            size="claw-sm"
            className="text-xs"
            onClick={handleStartDiagnosisClick}
          >
            <span>{startButtonLabel}</span>
          </Button>
        )}
        {showStartButton && occupiedByOther && (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="inline-block">
                  <Button
                    variant="tenant-primary"
                    size="claw-sm"
                    className="text-xs"
                    disabled
                  >
                    <span>{startButtonLabel}</span>
                  </Button>
                </span>
              </TooltipTrigger>
              <TooltipContent>
                {occupiedInitiatorType === "admin"
                  ? "管理员正在为该 Agent 进行诊断"
                  : "该 Agent 当前正在诊断中"}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )}
        {isCreating && (
          <div className="inline-flex items-center gap-2 px-3 h-8 rounded-[4px] bg-gray-50 border border-gray-200 text-xs text-gray-500">
            <span className="w-3 h-3 border-2 border-gray-300 border-t-gray-600 rounded-full animate-spin" />
            正在为您创建龙虾医生 Agent…
          </div>
        )}
        {(isActive || isDestroying) && (
          <Button
            variant="tenant-primary"
            size="claw-sm"
            className="text-xs"
            onClick={handleEndClick}
            disabled={isDestroying}
          >
            <span>{isDestroying ? "正在结束…" : "结束诊断"}</span>
          </Button>
        )}
      </div>

      {/* ─── 空态：还没创建过任何实例且无历史 ─────────────────────────────── */}
      {instanceStatus === "none" && !historySession && (
        <div className="px-6 pb-8">
          <div className="rounded-[4px] border border-dashed border-gray-200 bg-gray-50 flex flex-col items-center justify-center py-6 gap-1.5">
            <p className="text-sm font-medium text-gray-600">龙虾医生待命中</p>
            <p className="text-xs text-gray-400 text-center px-4">
              点击「开始诊断」后，将为您创建一台龙虾医生 Agent，对当前 Agent 进行全面检测和修复，初始化约需 3-5 分钟。诊断结束后该 Agent 会自动销毁。
            </p>
          </div>
        </div>
      )}

      {/* ─── 对话区（active / destroying / ended 都展示）────────────────────── */}
      {(showChatArea) && (
        <>
          {/* 消息列表（历史 Session 展示在最上方，向上滑动可见） */}
          <div
            className="overflow-y-auto px-6 space-y-4 pb-2 pt-2"
            style={{ minHeight: "300px", maxHeight: "440px" }}
          >
            {/* 历史 Session（如果有） */}
            {historySession && (
              <>
                <div className="flex items-center gap-2 py-1">
                  <div className="flex-1 h-px bg-gray-100" />
                  <span className="text-xs text-gray-400">历史诊断 · {historySession.endedAt}</span>
                  <div className="flex-1 h-px bg-gray-100" />
                </div>
                {historySession.messages.map((m, i) => renderMsg(m, -1000 - i))}
                <div className="flex items-center gap-2 py-1">
                  <div className="flex-1 h-px bg-gray-100" />
                  <span className="text-xs text-gray-400">当前诊断</span>
                  <div className="flex-1 h-px bg-gray-100" />
                </div>
              </>
            )}

            {/* 当前会话消息 */}
            {messages.map((msg, idx) => renderMsg(msg, idx))}
            {isTyping && <TypingBubble />}
            <div ref={messagesEndRef} />
          </div>

          {/* SDK 插槽：操作按钮区（在输入框上方）───────────────────────────
              注意：必须同时校验 instanceStatus === "active"。
              「10 分钟无操作自动结束」会把 instanceStatus 切到 "ended" 但不动 diagPhase
              （diagPhase 是会话内阶段、instanceStatus 是会话生命周期），仅判 diagPhase
              会让会话已死时按钮仍可点 → 触发无效修复请求。 */}
          {diagPhase === "summary_ready" && instanceStatus === "active" && (
            <div className="px-5 pt-1 pb-2 flex items-center gap-2">
              <Button
                variant="tenant-primary"
                onClick={handleStartRepair}
                className="h-7 text-xs px-3 gap-1"
              >
                开始修复
              </Button>
            </div>
          )}

          {/* 输入区（与 ChatView 同款样式）─────────────────────────────────── */}
          <div className="px-5 pb-5 pt-2">
            <div
              className={`rounded-[4px] border bg-white transition-colors ${
                inputDisabled ? "border-gray-200 opacity-60" : "border-gray-200 focus-within:border-gray-300"
              }`}
              style={{ boxShadow: "0 1px 3px rgba(0,0,0,0.04)" }}
            >
              <textarea
                ref={textareaRef}
                value={input}
                rows={1}
                onChange={(e) => {
                  setInput(e.target.value);
                  e.target.style.height = "auto";
                  e.target.style.height = `${Math.min(e.target.scrollHeight, 120)}px`;
                }}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    if (!isStreaming && !inputDisabled) handleSend();
                  }
                }}
                placeholder={inputPlaceholder}
                disabled={inputDisabled}
                className="w-full px-4 pt-3 pb-1 text-sm bg-transparent focus:outline-none disabled:cursor-not-allowed resize-none overflow-hidden leading-relaxed text-gray-800 placeholder:text-gray-400"
                style={{ minHeight: "44px" }}
              />
              {/* 底部工具栏：与 ChatView 一致 ── 左侧 +、指令库；右侧 麦克风、发送 */}
              <div className="flex items-center justify-between px-2 pb-2">
                <div className="flex items-center gap-0.5">
                  <button
                    type="button"
                    disabled={inputDisabled}
                    className="w-7 h-7 rounded-[4px] flex items-center justify-center text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-gray-400"
                    title="附件"
                  >
                    <Plus className="w-4 h-4" />
                  </button>
                  <div className="w-px h-4 bg-gray-200 mx-1" />
                  <div className="relative" ref={commandsRef}>
                    <button
                      type="button"
                      disabled={inputDisabled}
                      onClick={() => setShowCommands((prev) => !prev)}
                      className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-[4px] text-gray-500 hover:text-gray-700 hover:bg-gray-100 transition-colors text-xs font-medium disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-gray-500"
                    >
                      <Sparkles className="w-3.5 h-3.5" />
                      指令库
                      {showCommands ? <ChevronDown className="w-3 h-3" /> : <ChevronUp className="w-3 h-3" />}
                    </button>
                    {showCommands && (
                      <div
                        className="absolute bottom-full left-0 mb-2 w-48 bg-white rounded-[4px] border border-gray-200 py-1.5 z-50"
                        style={{ boxShadow: "var(--shadow-overlay)" }}
                      >
                        {DOCTOR_COMMAND_LIST.map((item) => (
                          <button
                            key={item.command}
                            type="button"
                            onClick={() => {
                              setShowCommands(false);
                              setInput(item.command + " ");
                              textareaRef.current?.focus();
                            }}
                            className="w-full flex items-center gap-2 px-3 py-1.5 hover:bg-gray-50 transition-colors text-left"
                          >
                            <span className="text-xs font-mono text-gray-900">{item.command}</span>
                            <span className="text-xs text-gray-400">{item.label}</span>
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-0.5">
                  <button
                    type="button"
                    disabled={inputDisabled}
                    className="w-7 h-7 rounded-[4px] flex items-center justify-center text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-gray-400"
                    title="语音输入"
                  >
                    <Mic className="w-4 h-4" />
                  </button>
                  {isStreaming ? (
                    <button
                      onClick={handleStopStreaming}
                      // allow-inline-gradient: 圆形停止按钮（27×27 rounded-full，非标准矩形按钮，SKILL §8.1 白名单）

                      className="w-7 h-7 rounded-full flex items-center justify-center text-white transition-all hover:opacity-90"
                      title="暂停输出"
                    >
                      <svg width="10" height="10" viewBox="0 0 12 12" fill="white">
                        <rect x="2" y="1.5" width="3" height="9" rx="1" />
                        <rect x="7" y="1.5" width="3" height="9" rx="1" />
                      </svg>
                    </button>
                  ) : (
                    <button
                      onClick={handleSend}
                      disabled={inputDisabled || !input.trim()}
                      className="w-7 h-7 rounded-full flex items-center justify-center text-white transition-all duration-150 disabled:opacity-30"
                      // allow-inline-gradient: 圆形发送按钮（27×27 rounded-full，非标准矩形按钮，SKILL §8.1 白名单）}
                      title="发送（Enter）"
                    >
                      <Send className="w-3 h-3" />
                    </button>
                  )}
                </div>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
