/**
 * StandardBasicInfo - 管控端基础信息配置页（OneID 专用模式）
 * Design: 「流动蓝图」Fluid Blueprint - Admin Side (浅灰背景)
 *
 * 视觉基线：与 BasicInfo（普通模式）完全一致
 *
 * 与普通模式的业务差异（仅以下三处）：
 *   - 步骤 1 多一个「同步企业信息」按钮（同步腾讯统一身份的网站名）
 *   - 步骤 3 「导入企业用户」跳转腾讯统一身份外链，而非站内用户管理
 *   - 新增步骤 4「设置用户登录方式」（多选 Popover），后续步骤顺延
 */
import { useState, useRef } from "react";
import { useLocation } from "wouter";
import { Button } from "@/components/ui/button";
import { StatusTag } from "@/components/ui/status-tag";
import { Input } from "@/components/ui/input";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Checkbox } from "@/components/ui/checkbox";
import { TokenValueEditor } from "@/components/policy";
import { toast } from "sonner";
import {
  Upload,
  CircleAlert,
  ChevronRight,
  ExternalLink,
  RefreshCw,
  Loader2,
  ChevronDown,
  X,
} from "lucide-react";
import {
  SITE_CONFIG,
  MOCK_SSO_IM_TYPE_OPTIONS,
  type SsoImTypeOption,
} from "@/lib/mockData";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { SurfaceCard } from "@/components/ui/Surface";
import { BodyMedium, HelperText, InlineNumber, MetaText, PanelTitle } from "@/components/ui/Typography";

// ─── 类型 ────────────────────────────────────────────────────────────────────

type TokenLimit = number | "unlimited";

// ─── Mock 完成状态 ───────────────────────────────────────────────────────────

const MOCK_STEP_STATUS: Record<number, boolean> = {
  1: true,  // 平台名称与品牌 — 已完成
  2: true,  // 用户默认配额 — 已完成
  3: false, // 导入企业用户 — 未完成
  4: false, // 用户登录方式 — 未完成（OneID 专属）
  5: true,  // 配置模型 — 已完成
  6: false, // 配置通道 — 未完成
  7: true,  // 配置镜像 — 已完成（默认有公共镜像）
  8: true,  // 配置私有网络 — 已完成（默认有预设VPC）
  9: false, // 配置安全组 — 未完成
};

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

// ─── 子组件：步骤序号徽章 ─────────────────────────────────────────────────────

function StepBadge({ step }: { step: number; done: boolean }) {
  return (
    <span className="self-center inline-flex h-6 items-center translate-y-[1px] font-din text-[18px] font-semibold leading-none tabular-nums text-[var(--text-brand)] shrink-0">
      {String(step).padStart(2, '0')}
    </span>
  );
}

// ─── 子组件：步骤卡片外壳 ─────────────────────────────────────────────────────

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
        <StepBadge step={step} done={done} />
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

// ─── 子组件：配额卡片字段（卡片级编辑模式） ─────────────────────────────────

function QuotaCardField({
  label,
  value,
  type,
  editing,
  draft,
  inputStr,
  onDraftChange,
  onInputChange,
}: {
  label: string;
  value: TokenLimit | number;
  type: "integer" | "token";
  editing: boolean;
  draft: TokenLimit | number;
  inputStr: string;
  onDraftChange: (v: TokenLimit | number) => void;
  onInputChange: (str: string) => void;
}) {
  const displayValue =
    value === "unlimited" || value === -1
      ? "无限制"
      : Number(value).toLocaleString();

  // 数字 → DIN 数字字体（加粗）；单位 → 文本字体（不加粗）
  const unitText =
    type === "token" && value !== "unlimited"
      ? "Tokens / 天"
      : type === "integer"
        ? "个"
        : "";

  const blockInvalidKeys = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (["-", "+", ".", "e", "E"].includes(e.key)) {
      e.preventDefault();
    }
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value.replace(/[^0-9]/g, "");
    onInputChange(val);
    onDraftChange(val ? Number(val) : 0);
  };

  // TokenValueEditor 的 mode/valStr 适配
  const tokenMode: "custom" | "unlimited" = draft === "unlimited" ? "unlimited" : "custom";
  const tokenValStr = draft === "unlimited" ? "" : inputStr;

  return (
    <div className="space-y-2">
      {!editing ? (
        <div className="flex items-baseline gap-0">
          <MetaText as="span">{label}：</MetaText>
          <InlineNumber as="span" tone="emphasis" className="font-bold">
            {displayValue}
          </InlineNumber>
          {unitText && (
            <BodyMedium as="span" tone="emphasis" className="ml-1 font-normal">
              {unitText}
            </BodyMedium>
          )}
        </div>
      ) : type === "integer" ? (
        <>
          <MetaText as="p">{label}</MetaText>
          <Input
            type="number"
            value={inputStr}
            min={0}
            onKeyDown={blockInvalidKeys}
            onChange={handleInputChange}
            className="bg-white border-gray-200 text-sm h-9 min-w-0 flex-1 max-w-[360px]"
            placeholder="0-999"
            autoFocus
          />
        </>
      ) : (
        <>
          <MetaText as="p">{label}</MetaText>
          <TokenValueEditor
            mode={tokenMode}
            valStr={tokenValStr}
            triggerClassName="group relative w-full max-w-[360px] h-9 px-3 pr-8 rounded-[4px] border border-gray-200 bg-white hover:border-[#1447E6] data-[state=open]:border-[#1447E6] transition-colors cursor-pointer flex items-center text-left text-sm"
            onCommit={(nextMode, nextValStr) => {
              if (nextMode === "unlimited") {
                onDraftChange("unlimited");
                onInputChange("");
              } else {
                onDraftChange(nextValStr ? Number(nextValStr) : 0);
                onInputChange(nextValStr);
              }
            }}
          />
        </>
      )}
    </div>
  );
}

// ─── 子组件：配额步骤卡片（卡片级编辑，按钮在右上角） ─────────────────────

function QuotaStepCard({
  step,
  done,
  clawLimit,
  tokenLimit,
  onSaveClawLimit,
  onSaveTokenLimit,
}: {
  step: number;
  done: boolean;
  clawLimit: number;
  tokenLimit: TokenLimit;
  onSaveClawLimit: (v: TokenLimit | number) => void;
  onSaveTokenLimit: (v: TokenLimit | number) => void;
}) {
  const [cardEditing, setCardEditing] = useState(false);
  const [clawDraft, setClawDraft] = useState<TokenLimit | number>(clawLimit);
  const [clawInputStr, setClawInputStr] = useState(String(clawLimit));
  const [tokenDraft, setTokenDraft] = useState<TokenLimit | number>(tokenLimit);
  const [tokenInputStr, setTokenInputStr] = useState(
    tokenLimit === "unlimited" ? "" : String(tokenLimit)
  );

  const startEdit = () => {
    setClawDraft(clawLimit);
    setClawInputStr(String(clawLimit));
    setTokenDraft(tokenLimit);
    setTokenInputStr(tokenLimit === "unlimited" ? "" : String(tokenLimit));
    setCardEditing(true);
  };

  const cancelEdit = () => {
    setCardEditing(false);
  };

  const saveEdit = () => {
    const clawVal = parseInt(clawInputStr, 10);
    if (isNaN(clawVal) || clawVal < 0 || clawVal > 999) {
      toast.error("请输入 0-999 之间的整数");
      return;
    }
    if (tokenDraft !== "unlimited") {
      const tokenVal = parseInt(tokenInputStr, 10);
      if (isNaN(tokenVal) || tokenVal < 0) {
        toast.error("请输入大于等于 0 的整数");
        return;
      }
      onSaveTokenLimit(tokenVal);
    } else {
      onSaveTokenLimit("unlimited");
    }
    onSaveClawLimit(clawVal);
    setCardEditing(false);
    toast.success("配额已保存");
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
        <StepBadge step={step} done={done} />
        <div className="min-w-0 flex items-center gap-2.5 flex-wrap pr-36">
          <PanelTitle as="p" className="leading-6">配置用户默认配额</PanelTitle>
          {done ? (
            <StatusTag mode="fill" variant="green">已完成</StatusTag>
          ) : (
            <StatusTag mode="fill" variant="gray">待完成</StatusTag>
          )}
        </div>
        <div className="col-start-2 mt-1 min-w-0 pr-36">
          <MetaText as="p" tone="secondary">设置新用户创建时自动应用的 Agent 数量上限和每日 Tokens 上限，有效控制企业成本</MetaText>
          <div className="mt-5 space-y-4">
            <QuotaCardField
              label="单用户 Agent 数量上限"
              value={clawLimit}
              type="integer"
              editing={cardEditing}
              draft={clawDraft}
              inputStr={clawInputStr}
              onDraftChange={setClawDraft}
              onInputChange={setClawInputStr}
            />
            <QuotaCardField
              label="单用户每日 Tokens 上限"
              value={tokenLimit}
              type="token"
              editing={cardEditing}
              draft={tokenDraft}
              inputStr={tokenInputStr}
              onDraftChange={setTokenDraft}
              onInputChange={setTokenInputStr}
            />
          </div>
        </div>
      </div>
    </SurfaceCard>
  );
}

// ─── 主页面 ───────────────────────────────────────────────────────────────────

export default function StandardBasicInfo() {
  const [, navigate] = useLocation();

  // ── 步骤 1：平台名称与品牌 ──
  const [siteName, setSiteName] = useState("A公司企业版Agent");
  const [logo, setLogo] = useState<File | null>(null);
  const [logoError, setLogoError] = useState<string | null>(null);
  const [syncing, setSyncing] = useState(false);
  const [step1Editing, setStep1Editing] = useState(false);
  const [step1DraftName, setStep1DraftName] = useState(siteName);
  const MAX_FILE_SIZE = 512 * 1024;

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

  const handleSaveClawLimit = (v: TokenLimit | number) => {
    const n = Number(v);
    setClawLimit(n);
    localStorage.setItem("policy_claw_limit", String(n));
  };

  const handleSaveTokenLimit = (v: TokenLimit | number) => {
    setTokenLimit(v);
    if (v === "unlimited") {
      localStorage.setItem("policy_token_limit_mode", "unlimited");
    } else {
      localStorage.setItem("policy_token_limit_mode", "custom");
      localStorage.setItem("policy_token_limit", String(v));
    }
  };

  // ── 步骤 4：用户登录方式（OneID 专属，卡片级编辑模式：参考步骤 1/2 交互） ──
  const [ssoImTypes, setSsoImTypes] = useState<string[]>([]);
  const ssoImTypeOptions: SsoImTypeOption[] = MOCK_SSO_IM_TYPE_OPTIONS;
  const originalSsoImTypesRef = useRef<string[]>([]);
  const [step4Editing, setStep4Editing] = useState(false);
  const [step4Draft, setStep4Draft] = useState<string[]>([]);

  // ── 同步企业信息（OneID 专属） ──
  const handleSyncEnterprise = () => {
    setSyncing(true);
    setTimeout(() => {
      setSyncing(false);
      setSiteName("A公司企业版Agent");
      toast.success("企业信息已同步成功");
    }, 1500);
  };

  return (
    <div className="page-enter">
      <AdminPageHeader
        title="基础信息配置"
        description="以下为必要的初始化配置，全部完成后用户端方可正常使用，更多高级配置可随时前往对应功能页调整"
      />

      {/* 双栏主体 */}
        <div className="flex gap-6 items-start">
        {/* ── 左侧：分步引导 ── */}
        <div className="min-w-0 space-y-5" style={{ flex: "1 1 0" }} data-guide="basic-steps">

          {/* 步骤 1：平台名称与品牌（参考 PolicyEditCard 卡片级编辑交互） */}
          <SurfaceCard className="relative px-6 pt-5 pb-6 overflow-hidden">
            <div className="absolute top-5 right-6 z-10">
              {!step1Editing ? (
                <Button
                  variant="claw-outline"
                  size="claw-sm"
                  onClick={() => {
                    setStep1DraftName(siteName);
                    setLogoError(null);
                    setStep1Editing(true);
                  }}
                >
                  编辑
                </Button>
              ) : (
                <div className="flex items-center gap-2">
                  <Button
                    variant="claw-outline"
                    size="claw-sm"
                    onClick={() => {
                      setStep1Editing(false);
                      setLogoError(null);
                    }}
                  >
                    取消
                  </Button>
                  <Button
                    variant="dialog-confirm"
                    size="claw-sm"
                    onClick={() => {
                      setSiteName(step1DraftName);
                      setStep1Editing(false);
                      toast.success("平台名称与品牌已保存");
                    }}
                  >
                    保存
                  </Button>
                </div>
              )}
            </div>
            <div className="grid grid-cols-[auto_1fr] gap-x-3.5">
              <StepBadge step={1} done={MOCK_STEP_STATUS[1]} />
              <div className="min-w-0 flex items-center gap-2.5 flex-wrap pr-36">
                <PanelTitle as="p" className="leading-6">设置平台名称与品牌</PanelTitle>
                {MOCK_STEP_STATUS[1] ? (
                  <StatusTag mode="fill" variant="green">已完成</StatusTag>
                ) : (
                  <StatusTag mode="fill" variant="gray">待完成</StatusTag>
                )}
              </div>
              <div className="col-start-2 mt-1 min-w-0 pr-36">
                <MetaText as="p" tone="secondary">配置展示在用户端的网站名称和logo</MetaText>

                {step1Editing ? (
                  /* ── 编辑态 ── */
                  <div className="mt-5 flex flex-col gap-2">
                    <div className="flex flex-col gap-2 mb-3">
                      <MetaText as="p">网站名称（将展示在用户端左上角常驻和首页）</MetaText>
                      <Input
                        id="siteName"
                        value={step1DraftName}
                        onChange={(e) => setStep1DraftName(e.target.value)}
                        placeholder="例如：A公司企业版Agent"
                        className="w-full max-w-[360px] h-9 border-gray-200 rounded-[4px] text-sm text-[var(--text-emphasis)]"
                      />
                    </div>
                    <div className="flex flex-col gap-2">
                      <MetaText as="p">网站logo（建议尺寸200x200px，不超过512kb）</MetaText>
                      <div className="flex items-center gap-3">
                        <img src="/icon/上传图片默认icon.svg" alt="" className="w-14 h-14 shrink-0 rounded-[6px]" />
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
                    {/* OneID 专属：同步企业信息 */}
                    <div className="mt-2">
                      <button
                        onClick={handleSyncEnterprise}
                        disabled={syncing}
                        className="h-8 px-3 inline-flex items-center gap-1.5 border border-gray-200 rounded-[4px] bg-white text-sm text-[#020617] hover:bg-[#f5f5f5] transition-colors disabled:opacity-60"
                      >
                        {syncing ? (
                          <Loader2 className="w-3.5 h-3.5 animate-spin" />
                        ) : (
                          <RefreshCw className="w-3.5 h-3.5" />
                        )}
                        同步企业信息
                      </button>
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
                      <MetaText as="p">网站logo</MetaText>
                      <img src="/icon/上传图片默认icon.svg" alt="" className="w-14 h-14 shrink-0 rounded-[6px]" />
                    </div>
                  </div>
                )}
              </div>
            </div>
          </SurfaceCard>

          {/* 步骤 2：用户默认配额（卡片级编辑模式） */}
          <QuotaStepCard
            step={2}
            done={MOCK_STEP_STATUS[2]}
            clawLimit={clawLimit}
            tokenLimit={tokenLimit}
            onSaveClawLimit={handleSaveClawLimit}
            onSaveTokenLimit={handleSaveTokenLimit}
          />

          {/* 步骤 3：导入企业用户（OneID 专属：跳转腾讯统一身份） */}
          <StepCard
            step={3}
            done={MOCK_STEP_STATUS[3]}
            title="导入企业用户"
            description="前往腾讯统一身份平台导入企业用户，导入完成后可在用户管理页进行管理"
          >
            <Button
              size="sm"
              variant="claw-outline"
              onClick={() => {
                window.open(
                  "https://ci-741.account.tencentcs.com/?redirectUrl=https%3A%2F%2Fe17himtkr0083u.ci-741.workspace.tencentcs.com%2Fadmin%2F%23%2Fusers#/login",
                  "_blank"
                );
              }}
              className="text-xs flex items-center gap-1.5 "
            >
              <ExternalLink className="w-3.5 h-3.5" />
              前往腾讯统一身份
            </Button>
          </StepCard>

          {/* 步骤 4：设置用户登录方式（OneID 专属，卡片级编辑模式） */}
          <SurfaceCard className="relative px-6 pt-5 pb-6 overflow-hidden">
            <div className="absolute top-5 right-6 z-10">
              {!step4Editing ? (
                <Button
                  variant="claw-outline"
                  size="claw-sm"
                  onClick={() => {
                    setStep4Draft([...ssoImTypes]);
                    setStep4Editing(true);
                  }}
                >
                  编辑
                </Button>
              ) : (
                <div className="flex items-center gap-2">
                  <Button
                    variant="claw-outline"
                    size="claw-sm"
                    onClick={() => {
                      setStep4Editing(false);
                    }}
                  >
                    取消
                  </Button>
                  <Button
                    variant="dialog-confirm"
                    size="claw-sm"
                    onClick={() => {
                      setSsoImTypes(step4Draft);
                      originalSsoImTypesRef.current = [...step4Draft];
                      setStep4Editing(false);
                      toast.success("用户登录方式已保存");
                    }}
                  >
                    保存
                  </Button>
                </div>
              )}
            </div>
            <div className="grid grid-cols-[auto_1fr] gap-x-3.5">
              <StepBadge step={4} done={MOCK_STEP_STATUS[4]} />
              <div className="min-w-0 flex items-center gap-2.5 flex-wrap pr-36">
                <PanelTitle as="p" className="leading-6">设置用户登录方式</PanelTitle>
                {MOCK_STEP_STATUS[4] ? (
                  <StatusTag mode="fill" variant="green">已完成</StatusTag>
                ) : (
                  <StatusTag mode="fill" variant="gray">待完成</StatusTag>
                )}
              </div>
              <div className="col-start-2 mt-1 min-w-0 pr-36">
                <MetaText as="p" tone="secondary">设置当前平台用户的默认登录方式，需与腾讯统一身份平台保持一致</MetaText>

                {step4Editing ? (
                  /* ── 编辑态 ── */
                  <div className="mt-5 flex flex-col gap-2">
                    <MetaText as="p">登录方式（可多选）</MetaText>
                    <Popover>
                      <PopoverTrigger asChild>
                        <button
                          type="button"
                          className="w-full max-w-[360px] flex items-center justify-between gap-2 rounded-[4px] border border-gray-200 bg-white px-3 text-sm text-left hover:border-blue-500 focus:border-blue-500 transition-colors h-9 outline-none data-[state=open]:border-blue-500"
                        >
                          <div className="flex items-center gap-1.5 flex-1 min-w-0 overflow-hidden">
                            {step4Draft.length === 0 ? (
                              <span className="text-[#A3A3A3]">请选择登录方式</span>
                            ) : (
                              step4Draft.map((val) => {
                                const opt = ssoImTypeOptions.find((o) => o.value === val);
                                return (
                                  <span
                                    key={val}
                                    className="inline-flex items-center gap-1 bg-[#F5F5F5] text-[#09090b] rounded-[4px] px-2 py-0.5 text-xs font-medium shrink-0"
                                  >
                                    {opt?.label ?? val}
                                    <X
                                      className="w-3 h-3 cursor-pointer hover:text-[#09090b] text-[#a1a1aa] shrink-0"
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        setStep4Draft((prev) => prev.filter((t) => t !== val));
                                      }}
                                    />
                                  </span>
                                );
                              })
                            )}
                          </div>
                          <ChevronDown className="w-4 h-4 text-[#A3A3A3] shrink-0" />
                        </button>
                      </PopoverTrigger>
                      <PopoverContent
                        className="p-0 w-[var(--radix-popover-trigger-width)]"
                        align="start"
                        sideOffset={4}
                      >
                        <div className="max-h-60 overflow-y-auto py-1">
                          {ssoImTypeOptions.map((opt) => {
                            const checked = step4Draft.includes(opt.value);
                            return (
                              <label
                                key={opt.value}
                                className={`flex items-center gap-3 px-3 py-2.5 cursor-pointer hover:bg-gray-50 transition-colors ${
                                  checked ? "bg-blue-50/50" : ""
                                }`}
                              >
                                <Checkbox
                                  checked={checked}
                                  onCheckedChange={(v) => {
                                    if (v) {
                                      setStep4Draft((prev) => [...prev, opt.value]);
                                    } else {
                                      setStep4Draft((prev) => prev.filter((t) => t !== opt.value));
                                    }
                                  }}
                                  className="shrink-0"
                                />
                                <span className="text-sm text-[#334155]">{opt.label}</span>
                              </label>
                            );
                          })}
                          {ssoImTypeOptions.length === 0 && (
                            <div className="px-3 py-4 text-center"><HelperText>暂无可选登录方式</HelperText></div>
                          )}
                        </div>
                      </PopoverContent>
                    </Popover>
                    {/* OneID 辅助入口：前往腾讯统一身份 */}
                    <div className="mt-3">
                      <button
                        onClick={() =>
                          window.open(
                            "https://console.cloud.tencent.com/cam/oneid",
                            "_blank",
                            "noopener,noreferrer"
                          )
                        }
                        className="h-8 px-3 inline-flex items-center gap-1.5 border border-gray-200 rounded-[4px] bg-white text-sm text-[#020617] hover:bg-[#f5f5f5] transition-colors"
                      >
                        <ExternalLink className="w-3.5 h-3.5" />
                        前往腾讯统一身份
                      </button>
                    </div>
                  </div>
                ) : (
                  /* ── 视图态 ── */
                  <div className="mt-5 flex flex-col gap-1">
                    <MetaText as="p">登录方式</MetaText>
                    {ssoImTypes.length === 0 ? (
                      <p className="text-sm text-[#A3A3A3]">未设置</p>
                    ) : (
                      <div className="flex items-center gap-1.5 flex-wrap">
                        {ssoImTypes.map((val) => {
                          const opt = ssoImTypeOptions.find((o) => o.value === val);
                          return (
                            <span
                              key={val}
                              className="inline-flex items-center bg-[#F5F5F5] text-[#09090b] rounded-[4px] px-2 py-0.5 text-xs font-medium"
                            >
                              {opt?.label ?? val}
                            </span>
                          );
                        })}
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          </SurfaceCard>

          {/* 步骤 5：配置模型 */}
          <StepCard
            step={5}
            done={MOCK_STEP_STATUS[5]}
            title="配置至少一个模型"
            description="为用户端配置至少一个全部用户可见的 AI 模型，用户创建 OpenClaw 时将从中选择"
          >
            <div className="space-y-3">
              <Button
                size="sm"
                variant="claw-outline"
                onClick={() => navigate("/admin/model-config")}
                className="text-xs flex items-center gap-1.5 "
              >
                前往模型配置
                <ChevronRight className="w-3.5 h-3.5" />
              </Button>
              <div className="pt-3 mt-4 border-t border-dashed border-gray-200"><MetaText as="p">如企业按组织配置，请前往<button onClick={() => navigate("/admin/members?view=group")} className="text-[#355EF1] hover:underline">用户管理 - 组织视图</button>查看各组织配置情况，未完成初始化的组织会有黄点标记</MetaText></div>
            </div>
          </StepCard>

          {/* 步骤 6：配置通道
              停服态下 6-9 步骤的「前往 xxx」跳转按钮与「用户管理 - 组织视图」文字链
              均属于纯导航/查看类操作，保持正常可用（data-billing-exempt 豁免全局停服禁用；
              元素自身 disabled 属性仍生效，若停服前已禁用则延续禁用） */}
          <StepCard
            step={6}
            done={MOCK_STEP_STATUS[6]}
            title="配置至少一个通道"
            description="为用户端配置至少一个全部用户启用的通道，用户创建 OpenClaw 时可选择对话平台"
          >
            <div className="space-y-3">
              <Button
                size="sm"
                variant="claw-outline"
                onClick={() => navigate("/admin/channel-config")}
                className="text-xs flex items-center gap-1.5 "
                data-billing-exempt
              >
                前往通道配置
                <ChevronRight className="w-3.5 h-3.5" />
              </Button>
              <div className="pt-3 mt-4 border-t border-dashed border-gray-200"><MetaText as="p">如企业按组织配置，请前往<button onClick={() => navigate("/admin/members?view=group")} className="text-[#355EF1] hover:underline" data-billing-exempt>用户管理 - 组织视图</button>查看各组织配置情况，未完成初始化的组织会有黄点标记</MetaText></div>
            </div>
          </StepCard>

          {/* 步骤 7：配置镜像 */}
          <StepCard
            step={7}
            done={MOCK_STEP_STATUS[7]}
            title="配置至少一个镜像"
            description="为用户端配置至少一个全部用户启用的镜像，系统默认已启用公共镜像"
          >
            <div className="space-y-3">
              <Button
                size="sm"
                variant="claw-outline"
                onClick={() => navigate("/admin/image-management")}
                className="text-xs flex items-center gap-1.5 "
                data-billing-exempt
              >
                前往镜像管理
                <ChevronRight className="w-3.5 h-3.5" />
              </Button>
              <div className="pt-3 mt-4 border-t border-dashed border-gray-200"><MetaText as="p">如企业按组织配置，请前往<button onClick={() => navigate("/admin/members?view=group")} className="text-[#355EF1] hover:underline" data-billing-exempt>用户管理 - 组织视图</button>查看各组织配置情况，未完成初始化的组织会有黄点标记</MetaText></div>
            </div>
          </StepCard>

          {/* 步骤 8：配置私有网络 */}
          <StepCard
            step={8}
            done={MOCK_STEP_STATUS[8]}
            title="配置私有网络"
            description="配置私有网络的预设策略，系统默认已创建一个预设 VPC"
          >
            <div className="space-y-3">
              <Button
                size="sm"
                variant="claw-outline"
                onClick={() => navigate("/admin/security-group?tab=vpc")}
                className="text-xs flex items-center gap-1.5 "
                data-billing-exempt
              >
                前往私有网络管理
                <ChevronRight className="w-3.5 h-3.5" />
              </Button>
              <div className="pt-3 mt-4 border-t border-dashed border-gray-200"><MetaText as="p">如企业按组织配置，请前往<button onClick={() => navigate("/admin/members?view=group")} className="text-[#355EF1] hover:underline" data-billing-exempt>用户管理 - 组织视图</button>查看各组织配置情况，未完成初始化的组织会有黄点标记</MetaText></div>
            </div>
          </StepCard>

          {/* 步骤 9：配置安全组 */}
          <StepCard
            step={9}
            done={MOCK_STEP_STATUS[9]}
            title="配置安全组"
            description="为 OpenClaw 云设备配置安全组规则，确保访问安全"
          >
            <Button
              size="sm"
              variant="claw-outline"
              onClick={() => navigate("/admin/security-group")}
              className="text-xs flex items-center gap-1.5 "
              data-billing-exempt
            >
              前往安全组管理
              <ChevronRight className="w-3.5 h-3.5" />
            </Button>
          </StepCard>
        </div>

        {/* ── 右侧：基础信息 + API文档 + 产品动态 ── */}
        <div className="shrink-0 flex flex-col gap-5" style={{ width: "352px" }}>

          {/* 平台基础信息 */}
          <SurfaceCard className="p-5">
            <PanelTitle as="p" className="mb-5">平台基础信息</PanelTitle>
            <div className="flex flex-col gap-5">
              <div className="flex gap-4 items-center">
                <img src="/icon/所在地域.svg" alt="" className="w-9 h-9 shrink-0 rounded-[4px]" />
                <div className="flex flex-col gap-0.5">
                  <MetaText as="p">所在地域</MetaText>
                  <p className="text-sm font-medium text-[var(--text-emphasis)]">{SITE_CONFIG.region}</p>
                </div>
              </div>
              <div className="flex gap-4 items-center">
                <img src="/icon/域名.svg" alt="" className="w-9 h-9 shrink-0 rounded-[4px]" />
                <div className="flex flex-col gap-0.5 min-w-0">
                  <MetaText as="p">域名</MetaText>
                  <p className="text-sm font-medium text-[var(--text-emphasis)] truncate">https://nmyy3n7z.clawpro.cloud/</p>
                </div>
              </div>
              <div className="flex gap-4 items-center">
                <img src="/icon/关联腾讯云账号.svg" alt="" className="w-9 h-9 shrink-0 rounded-[4px]" />
                <div className="flex flex-col gap-0.5">
                  <MetaText as="p">关联腾讯云账号</MetaText>
                  <p className="text-sm font-medium text-[var(--text-emphasis)]">{SITE_CONFIG.tencentUin}</p>
                </div>
              </div>
            </div>
          </SurfaceCard>

          {/* API 文档 */}
          <SurfaceCard
            className="p-5 cursor-pointer group transition-colors"
            onClick={() => window.open("/admin/api-docs", "_blank")}
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <img src="/icon/api文档-icon.svg" alt="" className="w-9 h-9 shrink-0 rounded-[4px]" />
                <div className="flex flex-col gap-0.5">
                  <p className="text-sm font-medium text-[var(--text-emphasis)] group-hover:text-[var(--text-brand)] transition-colors">API 文档</p>
                  <MetaText as="p" className="group-hover:text-[var(--text-brand)] transition-colors">查阅开放接口与调用示例</MetaText>
                </div>
              </div>
              <img src="/icon/arrow-left-stroke.svg" alt="" className="w-4 h-4 opacity-30 group-hover:opacity-100 group-hover:[filter:invert(32%)_sepia(98%)_saturate(1497%)_hue-rotate(215deg)_brightness(96%)_contrast(95%)] transition-all rotate-[-45deg]" />
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
                        src={item.type === "feature" ? "/icon/功能上新icon.svg" : "/icon/体验优化-icon.svg"}
                        alt=""
                        className="w-[18px] h-[18px]"
                      />
                      <StatusTag mode="fill" variant={item.type === "feature" ? "green" : "blue"}>
                        {item.type === "feature" ? "功能上新" : "体验优化"}
                      </StatusTag>
                    </div>
                    {/* 标题 */}
                    <p className="text-xs font-medium text-[var(--text-emphasis)] leading-5">{item.title}</p>
                    {/* 描述 */}
                    <MetaText as="p" className="line-clamp-2 h-10 overflow-hidden">{item.summary}</MetaText>
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
  );
}
