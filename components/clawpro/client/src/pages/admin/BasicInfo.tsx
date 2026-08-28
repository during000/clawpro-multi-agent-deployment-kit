/**
 * BasicInfo - 管控端基础信息配置页（custom / unified 模式共用）
 *
 * 视觉基线：与 OneID 模式 (StandardBasicInfo.tsx) 完全对齐
 *  - SurfaceCard / SurfaceInner / StatusTag / Typography / claw-outline / dialog-confirm
 *  - DIN 步骤号 + 卡片级编辑（编辑/取消/保存）
 *
 * 功能差异（保留 main 之上的 D1–D10）：
 *  - 三模式分流（useAdminMode().isUnified）+ displayStep 角标顺延
 *  - unified 模式插入步骤 4「设置用户登录方式」（外链跳腾讯统一身份）
 *  - 步骤 2 用 TokenTimeDimensionEditor 承载时间维度（OneID 没有）
 *  - 与平台策略页双向同步：policy_claw_limit / policy_token_limit_mode /
 *    policy_token_limit / admin_user_token_time_dim_v2 + storage 事件互通
 *
 * 布局：
 *   左侧：分步引导
 *     - custom 模式：8 步（步骤 1-2 内嵌表单，步骤 3-8 跳转引导）
 *     - unified 模式：9 步（在第 3 步「导入企业用户」后插入「设置用户登录方式」作为第 4 步，
 *                          原第 4-8 步顺延为 5-9）
 *   右侧上：平台基础信息（只读）+ API 文档
 *   右侧下：产品动态时间轴
 */
import { useEffect, useRef, useState } from "react";
import { useLocation } from "wouter";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { StatusTag } from "@/components/ui/status-tag";
import { SurfaceCard } from "@/components/ui/Surface";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import {
  BodyMedium,
  InlineNumber,
  MetaText,
  PanelTitle,
} from "@/components/ui/Typography";
import { toast } from "sonner";
import {
  Upload,
  CircleAlert,
  ChevronRight,
  ExternalLink,
} from "lucide-react";
import { SITE_CONFIG } from "@/lib/mockData";
import { useAdminMode } from "@/contexts/AdminModeContext";
import TokenTimeDimensionEditor, {
  type TimeDimensionConfig,
} from "@/components/TokenTimeDimensionEditor";
import type { TokenTimeDimensionEditorRef } from "@/components/TokenTimeDimensionEditor";

// ─── 类型 ────────────────────────────────────────────────────────────────────

type TokenLimit = number | "unlimited";

// ─── Mock 完成状态（模拟部分完成、部分未完成） ────────────────────────────────
//
// 注：以下 1-8 为「原始业务步骤」编号（custom 模式下即页面所见的角标）。
// 在 unified 模式下，会在原第 3 步「导入企业用户」之后插入「设置用户登录方式」
// 作为第 4 步，原第 4-8 步顺延为第 5-9 步（仅角标顺延，业务键 1-8 维持不变）。
// 新增的「登录方式」步骤完成态用独立的 key `loginMethod` 标识。

const MOCK_STEP_STATUS: Record<number, boolean> = {
  1: true,  // 平台名称与品牌 — 已完成（有默认值）
  2: true,  // 用户默认配额 — 已完成（有默认值）
  3: false, // 导入企业用户 — 未完成
  4: true,  // 配置模型 — 已完成
  5: false, // 配置通道 — 未完成
  6: true,  // 配置镜像 — 已完成（默认有公共镜像）
  7: true,  // 配置私有网络 — 已完成（默认有预设VPC）
  8: false, // 配置安全组 — 未完成
};

// unified 模式新增步骤：设置用户登录方式（默认未完成）
const MOCK_LOGIN_METHOD_DONE = false;

// 腾讯统一身份平台外链（unified 模式下「设置用户登录方式」步骤跳转目标）
const TENCENT_ONEID_URL = "https://console.cloud.tencent.com/cam/oneid";

// ─── 产品动态 Mock 数据 ───────────────────────────────────────────────────────

const PRODUCT_UPDATES = [
  {
    version: "",
    date: "2026-03-28",
    type: "feature" as const,
    title: "内置大模型支持多模态",
    summary: "内置大模型现已支持文本与图片解析，提升对话理解能力。自定义模型暂不支持多模态。",
  },
  {
    version: "",
    date: "2026-03-15",
    type: "feature" as const,
    title: "记忆管理功能上线",
    summary: '记忆管理功能直击"失忆"助理痛点，让 AI Agent记住你、理解你，更有企业级记忆增强版孵化中。',
  },
  {
    version: "",
    date: "2026-03-01",
    type: "improvement" as const,
    title: "模型支持设为默认",
    summary: "管理员可在模型配置页将模型设为默认，用户端新建 OpenClaw 时直接应用，无需手动添加。",
  },
  {
    version: "",
    date: "2026-02-15",
    type: "feature" as const,
    title: "公共技能库上线",
    summary: "管控端支持在技能配置页直接浏览上万个精选公共技能，自由挑选市场上的优质技能为龙虾赋能。",
  },
  {
    version: "",
    date: "2026-02-01",
    type: "feature" as const,
    title: "初始技能包上线，搭配免费 50G 存储",
    summary: "管理员可在技能配置页自由配置初始技能包并加入专有存储空间，OpenClaw 创建时极速下载预装技能。",
  },
  {
    version: "",
    date: "2026-01-15",
    type: "feature" as const,
    title: "ClawPro 新增法兰克福地域",
    summary: "ClawPro 法兰克福地域上线，支持欧洲区域就近部署（仅后端支持）。",
  },
  {
    version: "",
    date: "2025-12-20",
    type: "improvement" as const,
    title: "所有用户默认共用 1 个私有网络",
    summary: "企业内所有用户统一使用平台自动分配的 1 个 VPC，建议将安全组规则设置为内网不互通，以实现 OpenClaw 云服务器间隔离。",
  },
];

// ─── 子组件：步骤序号徽章（OneID 基线：DIN 数字，--text-brand） ──────────────

function StepBadge({ step }: { step: number }) {
  return (
    <span className="self-center inline-flex h-6 items-center translate-y-[1px] font-din text-[18px] font-semibold leading-none tabular-nums text-[var(--text-brand)] shrink-0">
      {String(step).padStart(2, "0")}
    </span>
  );
}

// ─── 子组件：步骤卡片外壳（视图态） ───────────────────────────────────────────

function StepCard({
  step,
  done,
  title,
  description,
  children,
}: {
  step: number;
  done: boolean;
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <SurfaceCard className="px-6 pt-5 pb-6 overflow-hidden">
      <div className="grid grid-cols-[auto_1fr] gap-x-3.5">
        <StepBadge step={step} />
        <div className="min-w-0 flex items-center gap-2.5 flex-wrap">
          <PanelTitle as="p" className="leading-6">{title}</PanelTitle>
          {done ? (
            <StatusTag mode="fill" variant="green">已完成</StatusTag>
          ) : (
            <StatusTag mode="fill" variant="gray">待完成</StatusTag>
          )}
        </div>
        <div className="col-start-2 mt-1 min-w-0">
          <MetaText as="p" tone="secondary">{description}</MetaText>
          <div className="mt-5">{children}</div>
        </div>
      </div>
    </SurfaceCard>
  );
}

// ─── 子组件：配额步骤卡片（卡片级编辑，按钮在右上角） ─────────────────────
// OneID 基线 QuotaStepCard 的 custom 模式扩展版：
//   - Agent 数量上限：与 OneID 一致的 InlineNumber 视图 + Input 编辑
//   - Tokens 上限：复用 TokenTimeDimensionEditor（自带视图/编辑切换 + 周期维度 + 二次确认）
// 卡片级"编辑/保存"按钮仅控制 Agent 数量；Tokens 上限由 TokenTimeDimensionEditor 自管。

function QuotaStepCard({
  step,
  done,
  clawLimit,
  tokenLimit,
  timeDimension,
  onSaveClawLimit,
  onSaveTokenDimension,
}: {
  step: number;
  done: boolean;
  clawLimit: number;
  tokenLimit: TokenLimit;
  timeDimension: TimeDimensionConfig;
  onSaveClawLimit: (v: number) => void;
  onSaveTokenDimension: (config: TimeDimensionConfig, limit: TokenLimit) => void;
}) {
  const [cardEditing, setCardEditing] = useState(false);
  const [clawDraft, setClawDraft] = useState<string>(String(clawLimit));
  const tokenEditorRef = useRef<TokenTimeDimensionEditorRef>(null);

  const blockInvalidKeys = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (["-", "+", ".", "e", "E"].includes(e.key)) {
      e.preventDefault();
    }
  };

  const startEdit = () => {
    setClawDraft(String(clawLimit));
    setCardEditing(true);
  };

  const cancelEdit = () => {
    setCardEditing(false);
  };

  const saveEdit = () => {
    // 验证 Agent 数量
    const n = parseInt(clawDraft, 10);
    if (isNaN(n) || n < 0 || n > 999) {
      toast.error("请输入 0-999 之间的整数");
      return;
    }
    // 保存 Token 配额（可能触发二次确认弹窗）
    const tokenOk = tokenEditorRef.current?.save();
    // 保存 Agent 数量
    onSaveClawLimit(n);
    if (tokenOk !== false) {
      setCardEditing(false);
      toast.success("配额已保存");
    }
  };

  return (
    <SurfaceCard className="relative px-6 pt-5 pb-6 overflow-hidden">
      <div className="absolute top-5 right-6 z-10">
        {!cardEditing ? (
          <Button variant="claw-outline" size="claw-sm" onClick={startEdit}>
            编辑
          </Button>
        ) : (
          <div className="flex items-center gap-2">
            <Button variant="claw-outline" size="claw-sm" onClick={cancelEdit}>
              取消
            </Button>
            <Button variant="dialog-confirm" size="claw-sm" onClick={saveEdit}>
              保存
            </Button>
          </div>
        )}
      </div>
      <div className="grid grid-cols-[auto_1fr] gap-x-3.5">
        <StepBadge step={step} />
        <div className="min-w-0 flex items-center gap-2.5 flex-wrap pr-36">
          <PanelTitle as="p" className="leading-6">配置用户默认配额</PanelTitle>
          {done ? (
            <StatusTag mode="fill" variant="green">已完成</StatusTag>
          ) : (
            <StatusTag mode="fill" variant="gray">待完成</StatusTag>
          )}
        </div>
        <div className="col-start-2 mt-1 min-w-0 pr-36">
          <MetaText as="p" tone="secondary">
            设置新用户创建时自动应用的 Agent 数量上限和每日 Tokens 上限，有效控制企业成本
          </MetaText>

          <div className="mt-5 space-y-4">
            {/* Agent 数量上限：视图/编辑切换 */}
            <div className="space-y-2">
              {!cardEditing ? (
                <div className="flex items-baseline gap-0">
                  <MetaText as="span">
                    单用户 Agent 数量上限
                    <span className="text-[var(--text-weak)] ml-1">
                      此处对应管控端平台策略页的预设策略
                    </span>
                    ：
                  </MetaText>
                  <InlineNumber as="span" tone="emphasis" className="font-bold">
                    {Number(clawLimit).toLocaleString()}
                  </InlineNumber>
                  <BodyMedium as="span" tone="emphasis" className="ml-1 font-normal">
                    个
                  </BodyMedium>
                </div>
              ) : (
                <>
                  <MetaText as="p">
                    单用户 Agent 数量上限
                    <span className="text-[var(--text-weak)] ml-1">
                      此处对应管控端平台策略页的预设策略
                    </span>
                  </MetaText>
                  <Input
                    type="number"
                    value={clawDraft}
                    min={0}
                    onKeyDown={blockInvalidKeys}
                    onChange={(e) =>
                      setClawDraft(e.target.value.replace(/[^0-9]/g, ""))
                    }
                    className="bg-white border-gray-200 text-sm h-9 min-w-0 max-w-[360px] rounded-[4px]"
                    placeholder="0-999"
                    autoFocus
                  />
                </>
              )}
            </div>

            {/* Tokens 上限 + 时间维度：受卡片编辑按钮控制 */}
            <TokenTimeDimensionEditor
              ref={tokenEditorRef}
              timeDimension={timeDimension}
              tokenLimit={tokenLimit}
              onSave={onSaveTokenDimension}
              editing={cardEditing}
              hideActions
            />
          </div>
        </div>
      </div>
    </SurfaceCard>
  );
}

// ─── 子组件：步骤 1 卡片（平台名称 + Logo，卡片级编辑） ─────────────────────

function PlatformBrandStepCard({
  step,
  done,
  siteName,
  onSaveSiteName,
}: {
  step: number;
  done: boolean;
  siteName: string;
  onSaveSiteName: (name: string) => void;
}) {
  const [editing, setEditing] = useState(false);
  const [draftName, setDraftName] = useState(siteName);
  const [logo, setLogo] = useState<File | null>(null);
  const [logoError, setLogoError] = useState<string | null>(null);
  const MAX_FILE_SIZE = 512 * 1024;

  const startEdit = () => {
    setDraftName(siteName);
    setLogoError(null);
    setEditing(true);
  };
  const cancelEdit = () => {
    setEditing(false);
    setLogoError(null);
  };
  const saveEdit = () => {
    onSaveSiteName(draftName);
    setEditing(false);
    toast.success("平台名称与品牌已保存");
  };

  return (
    <SurfaceCard className="relative px-6 pt-5 pb-6 overflow-hidden">
      <div className="absolute top-5 right-6 z-10">
        {!editing ? (
          <Button variant="claw-outline" size="claw-sm" onClick={startEdit}>
            编辑
          </Button>
        ) : (
          <div className="flex items-center gap-2">
            <Button variant="claw-outline" size="claw-sm" onClick={cancelEdit}>
              取消
            </Button>
            <Button variant="dialog-confirm" size="claw-sm" onClick={saveEdit}>
              保存
            </Button>
          </div>
        )}
      </div>
      <div className="grid grid-cols-[auto_1fr] gap-x-3.5">
        <StepBadge step={step} />
        <div className="min-w-0 flex items-center gap-2.5 flex-wrap pr-36">
          <PanelTitle as="p" className="leading-6">设置平台名称与品牌</PanelTitle>
          {done ? (
            <StatusTag mode="fill" variant="green">已完成</StatusTag>
          ) : (
            <StatusTag mode="fill" variant="gray">待完成</StatusTag>
          )}
        </div>
        <div className="col-start-2 mt-1 min-w-0 pr-36">
          <MetaText as="p" tone="secondary">配置展示在用户端的网站名称和 Logo</MetaText>

          {editing ? (
            /* ── 编辑态 ── */
            <div className="mt-5 flex flex-col gap-2">
              <div className="flex flex-col gap-2 mb-3">
                <MetaText as="p">网站名称（将展示在用户端左上角常驻和首页）</MetaText>
                <Input
                  id="siteName"
                  value={draftName}
                  onChange={(e) => setDraftName(e.target.value)}
                  placeholder="例如：A公司企业版Agent"
                  className="w-full max-w-[360px] h-9 border-gray-200 rounded-[4px] text-sm text-[var(--text-emphasis)]"
                />
              </div>
              <div className="flex flex-col gap-2">
                <MetaText as="p">网站 Logo（建议尺寸 200×200px，不超过 512KB）</MetaText>
                <div className="flex items-center gap-3">
                  <img
                    src="/icon/上传图片默认icon.svg"
                    alt=""
                    className="w-14 h-14 shrink-0 rounded-[6px]"
                  />
                  <label className="w-14 h-14 flex flex-col items-center justify-center gap-1 border border-dashed border-[#ddd] rounded-[6px] text-xs text-[rgba(0,0,0,0.9)] hover:border-blue-500 cursor-pointer transition-colors bg-white">
                    <Upload className="w-4 h-4" />
                    更换
                    <input
                      type="file"
                      accept="image/*"
                      className="hidden"
                      onChange={(e) => {
                        if (e.target.files?.[0]) {
                          const file = e.target.files[0];
                          if (file.size > MAX_FILE_SIZE) {
                            setLogoError("Logo 文件不能超过 512KB，请压缩后重试");
                            setLogo(null);
                          } else {
                            setLogoError(null);
                            setLogo(file);
                            toast.success("Logo 已上传");
                          }
                        }
                      }}
                    />
                  </label>
                  {logo && (
                    <span className="text-xs text-green-600 font-medium">{logo.name}</span>
                  )}
                </div>
                {logoError && (
                  <div className="relative w-full rounded-[4px] border border-amber-200 bg-amber-50 text-amber-950 px-4 py-3 text-sm flex items-center gap-3">
                    <CircleAlert className="w-4 h-4 text-amber-500 shrink-0" />
                    {logoError}
                  </div>
                )}
              </div>
            </div>
          ) : (
            /* ── 视图态 ── */
            <div className="mt-5 flex flex-col gap-4">
              <div className="flex flex-col gap-1">
                <MetaText as="p">网站名称</MetaText>
                <p className="text-sm text-[var(--text-emphasis)]">{siteName}</p>
              </div>
              <div className="flex flex-col gap-1">
                <MetaText as="p">网站 Logo</MetaText>
                <img
                  src="/icon/上传图片默认icon.svg"
                  alt=""
                  className="w-14 h-14 shrink-0 rounded-[6px]"
                />
              </div>
            </div>
          )}
        </div>
      </div>
    </SurfaceCard>
  );
}

// ─── 主页面 ───────────────────────────────────────────────────────────────────

export default function BasicInfo() {
  const [, navigate] = useLocation();
  const { isUnified } = useAdminMode();

  // 角标顺延：unified 模式下，原始业务步骤 4-8 在页面上展示为 5-9
  // （因为新插入的「设置用户登录方式」占据了第 4 位）
  const displayStep = (origin: number) => (isUnified && origin >= 4 ? origin + 1 : origin);

  // ── 步骤 1：平台名称与品牌 ──
  const [siteName, setSiteName] = useState("A公司企业版OpenClaw");

  // ── 步骤 2：用户默认配额（与平台策略页共享 localStorage） ──
  const [clawLimit, setClawLimit] = useState<number>(() => {
    const v = localStorage.getItem("policy_claw_limit");
    return v !== null ? Number(v) : 3;
  });
  const [tokenLimit, setTokenLimit] = useState<TokenLimit>(() => {
    const mode = localStorage.getItem("policy_token_limit_mode");
    if (mode === "unlimited") return "unlimited";
    const v = localStorage.getItem("policy_token_limit");
    return v !== null ? Number(v) : 500000;
  });
  const [timeDimension, setTimeDimension] = useState<TimeDimensionConfig>(() => {
    try {
      const raw = localStorage.getItem("admin_user_token_time_dim_v2");
      if (raw) {
        const parsed = JSON.parse(raw);
        if (parsed && (parsed.type === "natural" || parsed.type === "custom")) {
          return parsed as TimeDimensionConfig;
        }
      }
    } catch { /* ignore */ }
    return { type: "natural", period: "daily" };
  });

  const handleSaveClawLimit = (n: number) => {
    setClawLimit(n);
    localStorage.setItem("policy_claw_limit", String(n));
    window.dispatchEvent(new StorageEvent("storage", { key: "policy_claw_limit", newValue: String(n), storageArea: localStorage }));
  };

  const handleSaveTokenDimension = (config: TimeDimensionConfig, limit: TokenLimit) => {
    setTimeDimension(config);
    setTokenLimit(limit);
    const dimSerialized = JSON.stringify(config);
    localStorage.setItem("admin_user_token_time_dim_v2", dimSerialized);
    window.dispatchEvent(new StorageEvent("storage", { key: "admin_user_token_time_dim_v2", newValue: dimSerialized, storageArea: localStorage }));
    if (limit === "unlimited") {
      localStorage.setItem("policy_token_limit_mode", "unlimited");
      window.dispatchEvent(new StorageEvent("storage", { key: "policy_token_limit_mode", newValue: "unlimited", storageArea: localStorage }));
    } else {
      localStorage.setItem("policy_token_limit_mode", "custom");
      localStorage.setItem("policy_token_limit", String(limit));
      window.dispatchEvent(new StorageEvent("storage", { key: "policy_token_limit_mode", newValue: "custom", storageArea: localStorage }));
      window.dispatchEvent(new StorageEvent("storage", { key: "policy_token_limit", newValue: String(limit), storageArea: localStorage }));
    }
  };

  // 监听 storage：在平台策略页修改后反向同步
  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key === "policy_claw_limit" && e.newValue !== null) {
        setClawLimit(Number(e.newValue));
      } else if (e.key === "policy_token_limit_mode" || e.key === "policy_token_limit") {
        const mode = localStorage.getItem("policy_token_limit_mode");
        if (mode === "unlimited") {
          setTokenLimit("unlimited");
        } else {
          const raw = localStorage.getItem("policy_token_limit");
          setTokenLimit(raw !== null ? Number(raw) : 500000);
        }
      } else if (e.key === "admin_user_token_time_dim_v2" && e.newValue) {
        try {
          const parsed = JSON.parse(e.newValue);
          if (parsed && (parsed.type === "natural" || parsed.type === "custom")) {
            setTimeDimension(parsed as TimeDimensionConfig);
          }
        } catch { /* ignore */ }
      }
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);

  // ── 汇总：未完成步骤数（unified 模式额外计入「设置用户登录方式」） ──
  // （保留逻辑，供未来在右栏徽章使用；当前 UI 不展示）
  const _incompleteCount =
    Object.values(MOCK_STEP_STATUS).filter((v) => !v).length +
    (isUnified && !MOCK_LOGIN_METHOD_DONE ? 1 : 0);
  // 引用以避免 TS unused 警告
  void _incompleteCount;

  return (
    <>
      <div className="page-enter">
        <AdminPageHeader
        title="基础信息配置"
        description="以下为必要的初始化配置，全部完成后用户端方可正常使用，更多高级配置可随时前往对应功能页调整"
      />

      {/* 双栏主体 */}
      <div className="flex gap-6 items-start">
        {/* ── 左侧：分步引导 ── */}
        <div className="min-w-0 flex flex-col gap-5" style={{ flex: "1 1 0" }} data-guide="basic-steps">
          {/* 步骤 1：平台名称与品牌 */}
          <PlatformBrandStepCard
            step={displayStep(1)}
            done={MOCK_STEP_STATUS[1]}
            siteName={siteName}
            onSaveSiteName={setSiteName}
          />

          {/* 步骤 2：用户默认配额 */}
          <QuotaStepCard
            step={displayStep(2)}
            done={MOCK_STEP_STATUS[2]}
            clawLimit={clawLimit}
            tokenLimit={tokenLimit}
            timeDimension={timeDimension}
            onSaveClawLimit={handleSaveClawLimit}
            onSaveTokenDimension={handleSaveTokenDimension}
          />

          {/* 步骤 3：导入企业用户 */}
          <StepCard
            step={displayStep(3)}
            done={MOCK_STEP_STATUS[3]}
            title="导入企业用户"
            description="前往用户管理页添加企业用户，添加后即可使用平台"
          >
            <Button
              variant="claw-outline"
              size="claw-sm"
              onClick={() => navigate("/admin/members")}
              className="text-xs flex items-center gap-1.5"
              data-billing-exempt
            >
              前往用户管理
              <ChevronRight className="w-3.5 h-3.5" />
            </Button>
          </StepCard>

          {/* 步骤 4（仅 unified 模式）：设置用户登录方式 */}
          {isUnified && (
            <StepCard
              step={4}
              done={MOCK_LOGIN_METHOD_DONE}
              title="设置用户登录方式"
              description="前往腾讯统一身份平台设置当前平台用户的登录方式"
            >
              <Button
                variant="claw-outline"
                size="claw-sm"
                onClick={() => window.open(TENCENT_ONEID_URL, "_blank", "noopener,noreferrer")}
                className="text-xs flex items-center gap-1.5"
              >
                <ExternalLink className="w-3.5 h-3.5" />
                前往腾讯统一身份
              </Button>
            </StepCard>
          )}

          {/* 步骤 4 / 5：配置模型 */}
          <StepCard
            step={displayStep(4)}
            done={MOCK_STEP_STATUS[4]}
            title="配置至少一个模型"
            description="为用户端配置至少一个全部用户可见的 AI 模型，用户创建 OpenClaw 时将从中选择"
          >
            <div className="space-y-3">
              <Button
                variant="claw-outline"
                size="claw-sm"
                onClick={() => navigate("/admin/model-config")}
                className="text-xs flex items-center gap-1.5"
                data-billing-exempt
              >
                前往模型配置
                <ChevronRight className="w-3.5 h-3.5" />
              </Button>
              <div className="pt-3 mt-4 border-t border-dashed border-gray-200">
                <MetaText as="p">
                  如企业按组织配置，请前往
                  <button
                    onClick={() => navigate("/admin/members?view=group")}
                    className="text-[#355EF1] hover:underline"
                    data-billing-exempt
                  >
                    用户管理 - 组织视图
                  </button>
                  查看各组织配置情况，未完成初始化的组织会有黄点标记
                </MetaText>
              </div>
            </div>
          </StepCard>

          {/* 步骤 5 / 6：配置通道
              停服态下 5-8 步骤的「前往 xxx」跳转按钮与「用户管理 - 组织视图」文字链
              均属于纯导航/查看类操作，保持正常可用（data-billing-exempt 豁免全局停服禁用；
              元素自身 disabled 属性仍生效，若停服前已禁用则延续禁用） */}
          <StepCard
            step={displayStep(5)}
            done={MOCK_STEP_STATUS[5]}
            title="配置至少一个通道"
            description="为用户端配置至少一个全部用户启用的通道，用户创建 OpenClaw 时可选择对话平台"
          >
            <div className="space-y-3">
              <Button
                variant="claw-outline"
                size="claw-sm"
                onClick={() => navigate("/admin/channel-config")}
                className="text-xs flex items-center gap-1.5"
                data-billing-exempt
              >
                前往通道配置
                <ChevronRight className="w-3.5 h-3.5" />
              </Button>
              <div className="pt-3 mt-4 border-t border-dashed border-gray-200">
                <MetaText as="p">
                  如企业按组织配置，请前往
                  <button
                    onClick={() => navigate("/admin/members?view=group")}
                    className="text-[#355EF1] hover:underline"
                    data-billing-exempt
                  >
                    用户管理 - 组织视图
                  </button>
                  查看各组织配置情况，未完成初始化的组织会有黄点标记
                </MetaText>
              </div>
            </div>
          </StepCard>

          {/* 步骤 6 / 7：配置镜像 */}
          <StepCard
            step={displayStep(6)}
            done={MOCK_STEP_STATUS[6]}
            title="配置至少一个镜像"
            description="为用户端配置至少一个全部用户启用的镜像，系统默认已启用公共镜像"
          >
            <div className="space-y-3">
              <Button
                variant="claw-outline"
                size="claw-sm"
                onClick={() => navigate("/admin/image-management")}
                className="text-xs flex items-center gap-1.5"
                data-billing-exempt
              >
                前往镜像管理
                <ChevronRight className="w-3.5 h-3.5" />
              </Button>
              <div className="pt-3 mt-4 border-t border-dashed border-gray-200">
                <MetaText as="p">
                  如企业按组织配置，请前往
                  <button
                    onClick={() => navigate("/admin/members?view=group")}
                    className="text-[#355EF1] hover:underline"
                    data-billing-exempt
                  >
                    用户管理 - 组织视图
                  </button>
                  查看各组织配置情况，未完成初始化的组织会有黄点标记
                </MetaText>
              </div>
            </div>
          </StepCard>

          {/* 步骤 7 / 8：配置私有网络 */}
          <StepCard
            step={displayStep(7)}
            done={MOCK_STEP_STATUS[7]}
            title="配置私有网络"
            description="配置私有网络的预设策略，系统默认已创建一个预设 VPC"
          >
            <div className="space-y-3">
              <Button
                variant="claw-outline"
                size="claw-sm"
                onClick={() => navigate("/admin/security-group?tab=vpc")}
                className="text-xs flex items-center gap-1.5"
                data-billing-exempt
              >
                前往私有网络管理
                <ChevronRight className="w-3.5 h-3.5" />
              </Button>
              <div className="pt-3 mt-4 border-t border-dashed border-gray-200">
                <MetaText as="p">
                  如企业按组织配置，请前往
                  <button
                    onClick={() => navigate("/admin/members?view=group")}
                    className="text-[#355EF1] hover:underline"
                    data-billing-exempt
                  >
                    用户管理 - 组织视图
                  </button>
                  查看各组织配置情况，未完成初始化的组织会有黄点标记
                </MetaText>
              </div>
            </div>
          </StepCard>

          {/* 步骤 8 / 9：配置安全组 */}
          <StepCard
            step={displayStep(8)}
            done={MOCK_STEP_STATUS[8]}
            title="配置安全组"
            description="为 OpenClaw 云设备配置安全组规则，确保访问安全"
          >
            <Button
              variant="claw-outline"
              size="claw-sm"
              onClick={() => navigate("/admin/security-group")}
              className="text-xs flex items-center gap-1.5"
              data-billing-exempt
            >
              前往安全组管理
              <ChevronRight className="w-3.5 h-3.5" />
            </Button>
          </StepCard>
        </div>

        {/* ── 右侧：基础信息 + API 文档 + 产品动态 ── */}
        <div className="shrink-0 flex flex-col gap-5" style={{ width: "352px" }}>
          {/* 平台基础信息 */}
          <SurfaceCard className="p-5">
            <PanelTitle as="p" className="mb-5">平台基础信息</PanelTitle>
            <div className="flex flex-col gap-5">
              <div className="flex gap-4 items-center">
                <img
                  src="/icon/所在地域.svg"
                  alt=""
                  className="w-9 h-9 shrink-0 rounded-[4px]"
                />
                <div className="flex flex-col gap-0.5">
                  <MetaText as="p">所在地域</MetaText>
                  <p className="text-sm font-medium text-[var(--text-emphasis)]">
                    {SITE_CONFIG.region}
                  </p>
                </div>
              </div>
              <div className="flex gap-4 items-center">
                <img
                  src="/icon/域名.svg"
                  alt=""
                  className="w-9 h-9 shrink-0 rounded-[4px]"
                />
                <div className="flex flex-col gap-0.5 min-w-0">
                  <MetaText as="p">域名</MetaText>
                  <p className="text-sm font-medium text-[var(--text-emphasis)] truncate">
                    https://nmyy3n7z.clawpro.cloud/
                  </p>
                </div>
              </div>
              <div className="flex gap-4 items-center">
                <img
                  src="/icon/关联腾讯云账号.svg"
                  alt=""
                  className="w-9 h-9 shrink-0 rounded-[4px]"
                />
                <div className="flex flex-col gap-0.5">
                  <MetaText as="p">关联腾讯云账号</MetaText>
                  <p className="text-sm font-medium text-[var(--text-emphasis)]">
                    {SITE_CONFIG.tencentUin}
                  </p>
                </div>
              </div>
            </div>
          </SurfaceCard>

          {/* API 文档 —— 查看类跳转，停服态下保持可用 */}
          <SurfaceCard
            className="p-5 cursor-pointer group transition-colors"
            onClick={() => window.open("/admin/api-docs", "_blank")}
            data-billing-exempt
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <img
                  src="/icon/api文档-icon.svg"
                  alt=""
                  className="w-9 h-9 shrink-0 rounded-[4px]"
                />
                <div className="flex flex-col gap-0.5">
                  <p className="text-sm font-medium text-[var(--text-emphasis)] group-hover:text-[var(--text-brand)] transition-colors">
                    API 文档
                  </p>
                  <MetaText as="p" className="group-hover:text-[var(--text-brand)] transition-colors">
                    查阅开放接口与调用示例
                  </MetaText>
                </div>
              </div>
              <img
                src="/icon/arrow-left-stroke.svg"
                alt=""
                className="w-4 h-4 opacity-30 group-hover:opacity-100 group-hover:[filter:invert(32%)_sepia(98%)_saturate(1497%)_hue-rotate(215deg)_brightness(96%)_contrast(95%)] transition-all rotate-[-45deg]"
              />
            </div>
          </SurfaceCard>

          {/* 产品动态 */}
          <SurfaceCard className="p-5">
            <PanelTitle as="p" className="mb-6">产品动态</PanelTitle>
            <div className="flex flex-col gap-3">
              {PRODUCT_UPDATES.map((item, idx) => (
                <div key={idx}>
                  {/* 分隔线 */}
                  {idx > 0 && (
                    <div className="w-2 h-3 mb-3 border-l border-black/9 ml-[1px]" />
                  )}
                  {/* 条目 */}
                  <div className="flex flex-col gap-1">
                    {/* 图标 + 标签行 */}
                    <div className="flex items-center justify-between">
                      <img
                        src={
                          item.type === "feature"
                            ? "/icon/功能上新icon.svg"
                            : "/icon/体验优化-icon.svg"
                        }
                        alt=""
                        className="w-[18px] h-[18px]"
                      />
                      <StatusTag mode="fill" variant={item.type === "feature" ? "green" : "blue"}>
                        {item.type === "feature" ? "功能上新" : "体验优化"}
                      </StatusTag>
                    </div>
                    {/* 标题 */}
                    <p className="text-xs font-medium text-[var(--text-emphasis)] leading-5">
                      {item.title}
                    </p>
                    {/* 描述 */}
                    <MetaText as="p" className="line-clamp-2 h-10 overflow-hidden">
                      {item.summary}
                    </MetaText>
                    {/* 日期 */}
                    <MetaText as="p" tone="weak">{item.date}</MetaText>
                  </div>
                </div>
              ))}
            </div>
            {/* 查看全部更新按钮
                停服态下属于纯查看类操作，保持正常可用（data-billing-exempt 豁免全局停服禁用） */}
            <button
              className="mt-6 border border-gray-200 rounded-[4px] px-3 py-2 flex items-center gap-1 text-xs text-[var(--text-emphasis)] leading-5 hover:bg-[#f5f5f5] transition-colors"
              data-billing-exempt
            >
              查看全部更新
              <img src="/icon/arrow-left-stroke.svg" alt="" className="w-3 h-3" />
            </button>
          </SurfaceCard>
        </div>
      </div>
      </div>
    </>
  );
}
