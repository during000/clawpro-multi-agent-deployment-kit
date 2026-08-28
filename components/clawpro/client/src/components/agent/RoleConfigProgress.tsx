import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { BodyMedium, MetaText, MetaMedium, HelperText } from "@/components/ui/Typography";
import { AgentAvatar } from "@/components/agent/AgentAvatar";
import { TruncatedTooltip } from "@/components/ui/truncated-tooltip";
import { StatusTag } from "@/components/ui/status-tag";
import { Loader2, Check, ArrowRight, ChevronDown, ChevronUp, RotateCw } from "lucide-react";
import { cn } from "@/lib/utils";

/**
 * 角色配置进度：切换角色 / 批量切换角色 / 新增角色三类操作确认后的统一加载态。
 * 三步进度：更新（新增）角色身份 → 配置人设 → 安装 Agent Skills。
 */
/** 批量切换明细的单项：一个角色位的「从 → 到」 */
export interface RoleConfigProgressItem {
  /** 角色位 id（列表 key，可选） */
  slotId?: string;
  /** 切换前的角色名 */
  fromName: string;
  /** 切换后的目标角色名 */
  toName: string;
  /** 切换前的角色类型（可选；传入后明细项展示「类型 + 名称」两行，否则仅名称） */
  fromType?: string;
  /** 切换后的目标角色类型（可选；传入后明细项展示「类型 + 名称」两行，否则仅名称） */
  toType?: string;
  /** 是否为主角色位 */
  isPrimary?: boolean;
  /** 目标角色的预装技能数量 */
  skillCount: number;
  /** 配置结果状态：success=全部成功 / partial=部分成功（有技能安装失败） */
  resultStatus?: "success" | "partial";
  /** 安装失败的技能名列表（仅 partial 时有值） */
  failedSkills?: string[];
}

export interface RoleConfigProgressPayload {
  /** 目标角色名：用于头像与全部进度文案 */
  roleName: string;
  /** 角色风格简述：顶部副标题（单行截断） */
  roleSoul: string;
  /** 预装技能数量：第三步标题 */
  skillCount: number;
  /** 场景：switch=切换角色 / add=新增角色，决定首步文案 */
  mode: "switch" | "add";
  /** 所属 Agent id：用于「我知道了」关闭弹窗后在卡片上标记「角色切换中」 */
  agentId?: string;
  /** 所属 Agent 名称：批量切换时顶部展示「Agent：xxx」，让用户明确是哪个实例在配置 */
  agentName?: string;
  /**
   * 批量切换明细：传入后进度弹窗改为「Agent + 逐角色位 from → to」的详细展示，
   * 总进度按明细项数量均分，逐项推进（等待中 / 配置中 / 已完成）。
   */
  items?: RoleConfigProgressItem[];
  /** 进度走完 100% 后执行的真正落库动作 */
  apply?: () => void;
}

export interface RoleConfigProgressState extends RoleConfigProgressPayload {
  percent: number;
  stepIndex: number;
  /** 用户点了「我知道了」：弹窗关闭但进度继续推进，完成时仍会落库 */
  dismissed: boolean;
}

/** 进度条推进速率：80ms 递增 1~4%，总时长约 4 秒，与批量切换弹窗保持一致 */
const TICK_MS = 80;
/** 到达 100% 后的停留时长，让用户看到「已完成」再落库收尾 */
const SETTLE_MS = 420;

/**
 * 角色配置进度状态机：start() 启动进度，到 100% 后延迟执行 payload.apply() 并自动清理。
 * dismiss() 仅隐藏弹窗，不中断后台进度（与批量切换弹窗「我知道了」行为一致）。
 */
export function useRoleConfigProgress() {
  const [progress, setProgress] = useState<RoleConfigProgressState | null>(null);
  // apply 存在 state 里会进入 effect 依赖，用 ref 持有避免重复触发
  const applyRef = useRef<(() => void) | undefined>(undefined);

  const start = useCallback((payload: RoleConfigProgressPayload) => {
    applyRef.current = payload.apply;
    setProgress({ ...payload, percent: 0, stepIndex: 0, dismissed: false });
  }, []);

  const dismiss = useCallback(() => {
    setProgress((prev) => (prev ? { ...prev, dismissed: true } : prev));
  }, []);

  useEffect(() => {
    if (!progress) return;
    if (progress.percent >= 100) {
      const done = setTimeout(() => {
        applyRef.current?.();
        applyRef.current = undefined;
        setProgress(null);
      }, SETTLE_MS);
      return () => clearTimeout(done);
    }
    const timer = setTimeout(() => {
      setProgress((prev) => {
        if (!prev) return prev;
        // 批量场景下角色越多，单 tick 增幅越小，保证每个角色位都有可感知的配置时长
        const count = Math.max(1, prev.items?.length ?? 1);
        const step = Math.max(1, Math.round((Math.floor(Math.random() * 4) + 1) / count));
        const next = Math.min(100, prev.percent + step);
        // 三步：0-35% 更新身份 / 35-70% 配置人设 / 70-100% 安装 Skills
        const stepIndex = next < 35 ? 0 : next < 70 ? 1 : 2;
        return { ...prev, percent: next, stepIndex };
      });
    }, TICK_MS);
    return () => clearTimeout(timer);
  }, [progress]);

  return { progress, start, dismiss };
}

/**
 * 单个角色配置的三步文案（更新/新增身份 → 配置人设 → 安装 Skills）
 */
function buildSteps(roleName: string, skillCount: number, mode: "switch" | "add") {
  return [
    mode === "add"
      ? { title: `新增角色 “${roleName}”`, desc: "创建角色身份信息" }
      : { title: `更新角色 “${roleName}”`, desc: "更新角色身份信息" },
    { title: "配置人设", desc: "加载人设配置与个性化参数" },
    { title: `安装 Agent Skills（${skillCount} 个）`, desc: "加载并激活所有预装技能模块" },
  ];
}

/** 三步骤列表：已完成品牌色带勾 / 进行中品牌色旋转 Loader / 未开始灰色带勾占位 */
function StepList({
  steps,
  stepIndex,
  compact = false,
}: {
  steps: { title: string; desc: string }[];
  stepIndex: number;
  compact?: boolean;
}) {
  return (
    <div className={compact ? "space-y-2" : "space-y-4"}>
      {steps.map((step, i) => {
        const isActive = i === stepIndex;
        const isDone = i < stepIndex;
        return (
          <div key={i} className="flex items-center gap-3">
            {isActive ? (
              <span
                className={cn(
                  "shrink-0 flex items-center justify-center rounded-full bg-[#355EF1] text-white",
                  compact ? "size-4" : "size-5"
                )}
              >
                <Loader2 className={cn("animate-spin", compact ? "size-2.5" : "size-3")} />
              </span>
            ) : (
              <span
                className={cn(
                  "shrink-0 flex items-center justify-center rounded-full",
                  compact ? "size-4" : "size-5",
                  isDone ? "bg-[#355EF1] text-white" : "bg-[var(--muted)] text-[var(--text-weak)]"
                )}
              >
                <Check className={compact ? "size-2.5" : "size-3"} strokeWidth={3} />
              </span>
            )}
            <div className="min-w-0 space-y-0.5">
              <MetaMedium className={cn(!isActive && !isDone && "text-[var(--text-weak)]")}>
                {step.title}
              </MetaMedium>
              {!compact && (
                <TruncatedTooltip text={step.desc}>
                  <HelperText as="p" className="truncate">{step.desc}</HelperText>
                </TruncatedTooltip>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}

/** 批量切换进度卡片列表：默认不展示配置过程，点击展开 */
function ProgressCardList({ items, total, percent, innerStep, agentName, mode, onRetry }: {
  items: RoleConfigProgressItem[];
  total: number;
  percent: number;
  innerStep: number;
  agentName?: string;
  mode: "switch" | "add";
  onRetry?: (index: number) => void;
}) {
  const [expandedIds, setExpandedIds] = useState<Set<number>>(new Set());
  // 记录已经自动展开过的 partial 卡片，避免重复自动展开
  const autoExpandedRef = useRef<Set<number>>(new Set());

  // partial 卡片首次出现时自动展开；从 partial 变为 success 时自动收起
  const prevStatusRef = useRef<(string | undefined)[]>([]);
  useEffect(() => {
    items.forEach((item, i) => {
      const prevStatus = prevStatusRef.current[i];
      if (item.resultStatus === "partial" && !autoExpandedRef.current.has(i)) {
        autoExpandedRef.current.add(i);
        setExpandedIds((prev) => new Set(prev).add(i));
      }
      // 从 partial 变为 success：自动收起
      if (prevStatus === "partial" && item.resultStatus === "success") {
        setExpandedIds((prev) => {
          const next = new Set(prev);
          next.delete(i);
          return next;
        });
      }
    });
    prevStatusRef.current = items.map((it) => it.resultStatus);
  }, [items]);

  const toggleExpand = (idx: number) => {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      next.has(idx) ? next.delete(idx) : next.add(idx);
      return next;
    });
  };

  // 为每个卡片生成固定的进度偏移量（模拟快慢不一）
  const itemOffsetsRef = useRef<number[]>([]);
  if (itemOffsetsRef.current.length !== items.length) {
    itemOffsetsRef.current = items.map(() => Math.floor(Math.random() * 20) - 10);
  }

  const isDone = items.every((it) => !!it.resultStatus);

  return (
    <div className="py-2 space-y-4">
      <div className="space-y-0.5">
        <BodyMedium className="truncate">
          Agent：{agentName || "—"}
          {!isDone && <span className="text-[var(--text-muted)] font-normal ml-2">正在进行角色切换，请耐心等待...</span>}
        </BodyMedium>
      </div>

      <div className="space-y-2 max-h-[320px] overflow-y-auto [scrollbar-gutter:stable]">
        {items.map((item, i) => {
          const isPartial = isDone && item.resultStatus === "partial";
          const isSuccess = isDone && item.resultStatus === "success";
          // 每张卡片自己的进度（带偏移，模拟快慢差异）
          const itemProgress = isDone ? 100 : Math.min(100, Math.max(0, percent + (itemOffsetsRef.current[i] || 0)));
          // 部分成功的卡片首次自动展开（通过 useEffect），用户可手动收起
          const isExpanded = expandedIds.has(i);
          return (
            <div
              key={item.slotId ?? `${item.fromName}-${item.toName}-${i}`}
              className={cn(
                "rounded-[4px] border bg-white px-3 py-2.5 space-y-2 cursor-pointer transition-colors",
                isPartial ? "border-[#EAEEF4] hover:border-orange-400" : "border-[#EAEEF4] hover:border-[#355EF1]",
              )}
              onClick={() => toggleExpand(i)}
            >
              {/* 切换信息行：grid 布局保证箭头精确居中于原角色和目标角色之间 */}
              <div className="grid grid-cols-[1.5rem_1fr_1.5rem_1fr_auto_auto] items-center gap-x-2">
                <MetaText as="span" tone="secondary" className="shrink-0 tabular-nums text-center">
                  {i + 1}
                </MetaText>
                <span className="flex items-center gap-1.5 min-w-0">
                  <AgentAvatar roleName={item.fromType || item.fromName} size={28} className="shrink-0" />
                  <span className="min-w-0 flex flex-col text-left">
                    <span className="flex items-center gap-1 min-w-0">
                      <MetaMedium as="span" tone="emphasis" className="truncate text-sm">{item.fromType || item.fromName}</MetaMedium>
                    </span>
                    <HelperText as="span" className="block truncate">{item.fromName || item.fromType}</HelperText>
                  </span>
                </span>
                <span className="flex items-center justify-center">
                  <ArrowRight className="size-3.5 text-[var(--text-weak)]" />
                </span>
                <span className="flex items-center gap-1.5 min-w-0">
                  <AgentAvatar roleName={item.toType || item.toName} size={28} className="shrink-0" />
                  <span className="min-w-0 flex flex-col text-left">
                    <span className="flex items-center gap-1 min-w-0">
                      <MetaMedium as="span" tone="emphasis" className="truncate text-sm">{item.toType || item.toName}</MetaMedium>
                    </span>
                    <HelperText as="span" className="block truncate">{item.toName || item.toType}</HelperText>
                  </span>
                </span>
                <span className="shrink-0">
                  {isPartial ? (
                    <StatusTag mode="fill" variant="orange">部分成功</StatusTag>
                  ) : isSuccess ? (
                    <StatusTag mode="fill" variant="green">切换成功</StatusTag>
                  ) : (
                    <StatusTag mode="fill" variant="blue" icon={<Loader2 className="animate-spin" />}>
                      配置中
                    </StatusTag>
                  )}
                </span>
                {/* 展开/收起按钮 */}
                <span className="shrink-0 flex items-center justify-center size-5 rounded text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-grey-hover)] transition-colors">
                  {isExpanded ? <ChevronUp className="size-3.5" /> : <ChevronDown className="size-3.5" />}
                </span>
              </div>
              {/* 配置中：显示进度条（收起状态也可见） */}
              {!isDone && (
                <Progress
                  value={itemProgress}
                  className="h-1 [&>[data-slot=progress-indicator]]:bg-[linear-gradient(90deg,#93B4FF_0%,#355EF1_100%)]"
                />
              )}
              {/* 展开时显示配置明细 */}
              {isExpanded && (
                <>
                  <div className="border-t border-gray-200" />
                  <div className="pl-6">
                    {isPartial ? (
                      /* 部分成功：展示三步结果，第三步标红 + 失败技能列表 */
                      <div className="space-y-1.5 py-1">
                        <div className="flex items-center gap-2">
                          <Check className="size-3.5 text-green-500 shrink-0" />
                          <MetaText tone="secondary">更新角色「{item.toName}」</MetaText>
                        </div>
                        <div className="flex items-center gap-2">
                          <Check className="size-3.5 text-green-500 shrink-0" />
                          <MetaText tone="secondary">配置人设</MetaText>
                        </div>
                        <div className="flex items-center gap-2">
                          <span className="size-3.5 shrink-0 flex items-center justify-center text-orange-500 font-bold text-xs">!</span>
                          <MetaText as="span" tone="secondary">
                            安装 Agent Skills（{item.skillCount} 个）
                          </MetaText>
                        </div>
                        <div className="pl-[1.375rem] space-y-0.5">
                          <div className="flex items-center gap-2">
                            <span className="text-xs text-[var(--text-danger)] cursor-default">
                              部分技能安装失败，请重试或联系管理员处理
                            </span>
                            <button
                              type="button"
                              className="inline-flex items-center gap-1 text-xs text-[#355EF1] hover:text-[#2845c5] cursor-pointer bg-transparent border-none p-0 m-0"
                              onClick={(e) => {
                                e.stopPropagation();
                                if (onRetry) {
                                  onRetry(i);
                                } else {
                                  toast.error("技能安装失败", {
                                    description: `以下技能安装失败：${item.failedSkills?.join("、") || "未知技能"}，请稍后重试或联系管理员处理。`,
                                  });
                                }
                              }}
                            >
                              <RotateCw className="size-3" />
                              重试
                            </button>
                          </div>
                          {item.failedSkills && item.failedSkills.length > 0 && (
                            <HelperText as="p" className="leading-snug">
                              失败技能：{item.failedSkills.join("、")}
                            </HelperText>
                          )}
                        </div>
                      </div>
                    ) : (
                      <StepList
                        steps={buildSteps(item.toName, item.skillCount, mode)}
                        stepIndex={isDone ? 3 : innerStep}
                        compact
                      />
                    )}
                  </div>
                </>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

/**
 * 进度内容区（不含 Dialog 外壳）：供已有弹窗内切换加载态复用（如批量切换弹窗）。
 * 传入 items 时切换为「批量明细」形态：顶部展示所属 Agent 名称 + 整体进度，
 * 下方逐个角色位展示「原角色 → 目标角色」及各自状态（等待中 / 配置中 / 已完成），
 * 当前配置中的那一项展开三步子进度。
 */
export function RoleConfigProgressContent({
  roleName,
  roleSoul,
  skillCount,
  mode,
  percent,
  stepIndex,
  agentName,
  items,
  onRetry,
}: Omit<RoleConfigProgressState, "dismissed" | "apply"> & { onRetry?: (index: number) => void }) {
  // ===== 批量明细形态（传入 items 时启用；1 个也走同一形态，保证「从什么切换成什么」始终可见） =====
  if (items && items.length > 0) {
    const total = items.length;
    // 并行模式：所有角色位同时进行配置，共享同一个进度百分比
    const innerStep = percent < 35 ? 0 : percent < 70 ? 1 : 2;

    return (
      <ProgressCardList
        items={items}
        total={total}
        percent={percent}
        innerStep={innerStep}
        agentName={agentName}
        mode={mode}
        onRetry={onRetry}
      />
    );
  }

  // ===== 单角色形态（新增角色 / 单个角色位切换） =====
  const steps = buildSteps(roleName, skillCount, mode);

  return (
    <div className="py-2">
      {/* 顶部：目标角色头像 + 类型（上）+ 名称/风格简述（下） */}
      <div className="flex items-center gap-3">
        <AgentAvatar roleName={roleName} size={48} />
        <div className="min-w-0 space-y-0.5">
          <TruncatedTooltip text={roleName}>
            <BodyMedium className="truncate">{roleName}</BodyMedium>
          </TruncatedTooltip>
          <TruncatedTooltip text={roleSoul}>
            <HelperText as="p" className="truncate">{roleSoul}</HelperText>
          </TruncatedTooltip>
        </div>
      </div>

      {/* 进度：仅进度条（右侧百分比）+ 进度条下方的配置过程步骤，去掉标题/说明文字与分割线 */}
      <div className="mt-4 space-y-3">
        <div className="flex items-center gap-4">
          {/* 进度条：走项目 Progress 组件，覆盖为品牌渐变填充（浅蓝 → 品牌蓝） */}
          <Progress
            value={percent}
            className="flex-1 h-2 [&>[data-slot=progress-indicator]]:bg-[linear-gradient(90deg,#93B4FF_0%,#355EF1_100%)]"
          />
          <MetaText as="span" tone="secondary" className="tabular-nums shrink-0">
            {percent}%
          </MetaText>
        </div>
        <StepList steps={steps} stepIndex={stepIndex} />
      </div>
    </div>
  );
}

/**
 * 独立进度弹窗：配合 useRoleConfigProgress 使用。
 * 加载中不可关闭（无右上角关闭按钮、点击遮罩/ESC 无效），仅底部居中「我知道了」可隐藏弹窗，
 * 隐藏后进度仍在后台推进并在完成时落库。
 */
export function RoleConfigProgressDialog({
  progress,
  onDismiss,
}: {
  progress: RoleConfigProgressState | null;
  onDismiss: () => void;
}) {
  const open = !!progress && !progress.dismissed;
  const isBatch = !!progress?.items?.length;
  // 宽度独立于 isBatch 判定：仅「多角色批量切换」（items > 1）才用更宽的 xl(920px) 容纳多行明细；
  // 单角色切换（items 仅 1 项，如从 SwitchRoleDialog 触发）与「切换角色」弹窗同宽（720px），
  // 避免单角色进度弹窗过宽、与来源弹窗宽度不一致。标题 / 内容文案仍沿用 isBatch，不受影响。
  const isWideBatch = (progress?.items?.length ?? 0) > 1;
  return (
    <Dialog open={open} onOpenChange={() => { /* 加载中禁止关闭，避免中断落库 */ }}>
      {/* 宽度：多角色批量切换 xl(920px)，单角色 / 无明细 720px（与「切换角色」弹窗一致）。 */}
      <DialogContent size={isWideBatch ? "xl" : undefined} className={isWideBatch ? "" : "sm:max-w-[720px]"} showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>
            {isBatch
              ? (progress && progress.items?.every((it) => it.resultStatus)
                ? `切换角色完成（共 ${progress?.items?.length} 个）`
                : `正在切换角色（共 ${progress?.items?.length} 个）`)
              : "正在配置角色"}
          </DialogTitle>
        </DialogHeader>
        <DialogBody className="px-6">
          {progress && <RoleConfigProgressContent {...progress} />}
        </DialogBody>
        <DialogFooter className="justify-center">
          <Button variant="tenant-dialog-confirm" onClick={onDismiss}>
            我知道了
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
