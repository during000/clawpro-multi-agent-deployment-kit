/**
 * BatchSwitchRoleDialog —— 「切换角色」独立批量切换弹窗（卡片页 / 设置详情页共用）
 *
 * 单一数据源：卡片页（MyOpenClaw）与设置详情页（OpenClawDetailGuide）此前各维护了一份
 * 结构不同的切换角色弹窗，导致「卡片页改了详情页不跟着改」。现将UI 与内部交互逻辑统一
 * 抽取到本组件，两处仅通过 props 注入「打开数据源 / 落库回调 / 后台标记回调」即可，
 * 未来任何弹窗 UI 改动只需改本文件，两个页面自动联动保持完全一致。
 *
 * 组件内部自管理：目标选择、右栏激活态、技能勾选、配置加载动画（含到 100% 自动落库）。
 * 落库与「后台切换中」标记的业务差异由外部通过 onCommit / onBackgrounded 注入。
 */
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogBody,
} from "@/components/ui/dialog";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { TruncatedTooltip } from "@/components/ui/truncated-tooltip";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectSeparator } from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import { MetaMedium, MetaText, HelperText } from "@/components/ui/Typography";
import { cn } from "@/lib/utils";
import { X, RotateCcw, ChevronRight, Plus, Pencil, ArrowLeftRight, Loader2 } from "lucide-react";
import { RoleConfigProgressContent, type RoleConfigProgressItem } from "@/components/agent/RoleConfigProgress";
import { AgentAvatar } from "@/components/agent/AgentAvatar";
import { RoleTypeRadioGroup } from "@/components/agent/RoleTypeRadioGroup";
import type { Role, AgentRoleSlot } from "@/lib/mockData";

/** 角色名称长度上限（与 RoleManageSheet「修改角色名称」保持一致） */
const NAME_MAX_LEN = 25;

/**
 * 生成不与 taken 集合重复、且不超过 maxLen 的名称：base 未被占用则原样返回（超长先截断）；
 * 否则依次尝试 base2、base3…… 直到不重复（用于切换类型时自动填充避免重名）。
 * 追加序号时为序号预留字符位，保证结果始终满足长度上限。
 */
function makeUniqueName(base: string, taken: Set<string>, maxLen: number = NAME_MAX_LEN): string {
  const b = base.trim().slice(0, maxLen);
  if (!b || !taken.has(b)) return b;
  for (let i = 2; i <= 999; i += 1) {
    const suffix = String(i);
    const stem = b.slice(0, Math.max(1, maxLen - suffix.length));
    const candidate = `${stem}${suffix}`;
    if (!taken.has(candidate)) return candidate;
  }
  return b.slice(0, maxLen);
}

/**
 * 溢出省略文本的 Tooltip 包裹器：仅当子元素文本真的被 truncate 截断（scrollWidth > clientWidth）
 * 时才挂 Tooltip 展示全文，未截断时不加任何浮层（避免无意义 tooltip）。
 * 与 select.tsx 内的 TruncatedText 同一实现口径；此处以 children 形式包裹，
 * 便于保留 Typography（MetaText 等）语义组件与原有 class。
 */
/** 角色介绍（技能 + 风格），供右栏角色卡片展示 */
export interface RoleIntro {
  name: string;
  skills: string;
  soul: string;
}

/** 弹窗打开数据源：目标 Agent + 其全部角色位 + 分组可切换范围限制 */
export interface BatchSwitchSource {
  id: string;
  name: string;
  slots: AgentRoleSlot[];
  /** 分组限制：undefined=无限制；[]=仅通用助手；非空=白名单角色名 */
  allowedRoleNames?: string[];
}

/** 落库回调载荷：本次点击「确认切换」时的最终角色位与目标映射 */
export interface BatchSwitchCommitPayload {
  id: string;
  name: string;
  slots: AgentRoleSlot[];
  targets: Record<string, Role | "__general__" | null>;
  items: RoleConfigProgressItem[];
}

interface BatchSwitchRoleDialogProps {
  /** 打开数据源；null 表示关闭 */
  source: BatchSwitchSource | null;
  /** 请求关闭弹窗（点击遮罩 / 取消 / 关闭按钮，配置加载中会被组件内部拦截） */
  onClose: () => void;
  /** 当前可见角色列表 */
  visibleRoles: Role[];
  /** 生成某角色（或通用助手）的技能 / 风格介绍 */
  getRoleIntro: (target: Role | "__general__" | null) => RoleIntro;
  /** 配置加载动画走完（100%）后落库：由各页面注入自己的 apply 实现 */
  onCommit: (payload: BatchSwitchCommitPayload) => void;
  /** 「我知道了」把切换转入后台时，标记该 Agent「角色切换中（N）」 */
  onBackgrounded: (agentId: string, count: number, agentName: string, items: RoleConfigProgressItem[]) => void;
  /** 交互方案：1=纯切换（无新增/删除）；2=左表格右详情（默认）；3=左文字目录右编辑 */
  scheme?: 1 | 2 | 3;
  /** 方案3：点击「添加」按钮时回调（由外部打开添加角色弹窗） */
  onAddRole?: () => void;
  /** 正在新增角色的数量（>0 时在新增按钮旁显示胶囊提示） */
  roleAddingCount?: number;
  /** 删除角色位回调：传入被删除的 slotId 和角色名 */
  onDeleteSlot?: (slotId: string, roleName: string) => void;
  /** 正在运行的 Agent 名称集合：切换后的角色名称不能与之重复 */
  runningAgentNames?: string[];
}

export function BatchSwitchRoleDialog({
  source,
  onClose,
  visibleRoles,
  getRoleIntro,
  onCommit,
  onBackgrounded,
  scheme = 2,
  onAddRole,
  roleAddingCount = 0,
  onDeleteSlot,
  runningAgentNames = [],
}: BatchSwitchRoleDialogProps) {
  // ── 弹窗内部交互态 ──
  const [targets, setTargets] = useState<Record<string, Role | "__general__" | null>>({});
  // 目标角色自定义显示名：slotId → 用户在右栏「角色名称」输入框改写的名称（覆盖默认角色名）。
  const [targetNames, setTargetNames] = useState<Record<string, string>>({});
  const [activeSlotId, setActiveSlotId] = useState<string | null>(null);
  const [skillNames, setSkillNames] = useState<Record<string, Set<string>>>({});
  const [skillPopoverSlot, setSkillPopoverSlot] = useState<string | null>(null);
  // 方案3：右栏编辑模式（默认只读，点击编辑按钮进入编辑态）
  const [scheme3Editing, setScheme3Editing] = useState(false);
  // 方案3：删除角色二次确认
  const [deleteSlotConfirm, setDeleteSlotConfirm] = useState<{ slotId: string; roleName: string } | null>(null);
  // 编辑角色名称弹窗
  const [editNameSlot, setEditNameSlot] = useState<{ slotId: string; name: string } | null>(null);
  const [editNameValue, setEditNameValue] = useState("");
  // 配置加载态：确认切换后进入进度动画，走完 100% 再落库
  const [loading, setLoading] = useState<{
    id: string;
    name: string;
    slots: AgentRoleSlot[];
    targets: Record<string, Role | "__general__" | null>;
    primaryRoleName: string;
    skillCount: number;
    items: RoleConfigProgressItem[];
    percent: number;
    stepIndex: number;
  } | null>(null);

  // 弹窗关闭时重置内部态（source 变为 null）
  useEffect(() => {
    if (!source) {
      setTargets({});
      setTargetNames({});
      setActiveSlotId(null);
      setSkillNames({});
      setSkillPopoverSlot(null);
      setScheme3Editing(false);
    }
  }, [source]);

  // 打开即默认激活第一个角色位，直接进入右侧编辑区。
  useEffect(() => {
    const slots = source?.slots ?? [];
    if (source && slots.length > 0) {
      setActiveSlotId(slots[0].slotId);
    }
  }, [source, scheme]);

  // 配置加载动画：递增进度，到 100% 模拟结果（部分可能失败）并落库
  useEffect(() => {
    if (!loading) return;
    if (loading.percent >= 100 && !loading.items.some((it) => it.resultStatus)) {
      // 模拟部分技能安装失败：根据 Agent id 决定结果
      const done = setTimeout(() => {
        const resolvedItems = loading.items.map((item, idx) => {
          // "角色切换部分成功+重试失败"(oc-role-011) 和 "角色切换部分成功+重试成功"(oc-role-012)
          // 永远让第二项部分失败；其它agent全部成功
          if (idx === 1 && (loading.id === "oc-role-011" || loading.id === "oc-role-012")) {
            return { ...item, resultStatus: "partial" as const, failedSkills: ["legacy-plugin", "deprecated-tool"] };
          }
          return { ...item, resultStatus: "success" as const };
        });
        const hasPartial = resolvedItems.some((it) => it.resultStatus === "partial");
        // 无论是否有异常都落库（角色已切换成功，仅技能安装有问题）
        onCommit({
          id: loading.id,
          name: loading.name,
          slots: loading.slots,
          targets: loading.targets,
          items: resolvedItems,
        });
        if (hasPartial) {
          // 有部分失败：弹窗不关闭，展示结果让用户确认
          setLoading({ ...loading, items: resolvedItems });
        } else {
          // 全部成功：正常关闭
          setTargets({});
          setTargetNames({});
          setActiveSlotId(null);
          setLoading(null);
          onClose();
        }
      }, 420);
      return () => clearTimeout(done);
    }
    const timer = setTimeout(() => {
      setLoading((prev) => {
        if (!prev) return prev;
        const count = Math.max(1, prev.items.length);
        // 随机步幅 1~6，越多角色越慢；模拟真实快慢变化
        const base = Math.floor(Math.random() * 6) + 1;
        const step = Math.max(1, Math.round(base / count));
        const next = Math.min(100, prev.percent + step);
        const stepIndex = next < 35 ? 0 : next < 70 ? 1 : 2;
        return { ...prev, percent: next, stepIndex };
      });
    }, 60 + Math.floor(Math.random() * 80));
    return () => clearTimeout(timer);
  }, [loading, onCommit, onClose]);

  // 弹窗尺寸自适应：当全部角色位都没有任何可切换目标（既无可切换角色，也不能切通用助手）时，
  // 右栏（角色详情）永远不会被激活，弹窗恒为左侧单栏 —— 收窄为规范内小一号尺寸（lg）；
  // 否则使用规范内最大尺寸（xl），给左右两栏更充裕的展示空间。
  const hasNoOptions = (() => {
    const slots = source?.slots ?? [];
    if (slots.length === 0) return false;
    const allowed = source?.allowedRoleNames;
    return slots.every((slot) => {
      const candidates =
        allowed !== undefined ? visibleRoles.filter((r) => allowed.includes(r.name)) : visibleRoles;
      const switchable = candidates.filter((r) => r.name !== slot.roleName);
      const canGeneral = slot.roleName !== "通用助手";
      return switchable.length === 0 && !canGeneral;
    });
  })();

  // 右栏（目标角色编辑区）是否展示，需同时满足：
  //  1) 该 Agent 存在可切换选项（hasNoOptions 为 false）；
  //  2) 当前激活了某一角色位（点击左栏行即进入右栏编辑：类型下拉 + 名称输入框）。
  // 右栏是编辑区（非只读卡片），故只要激活了行就展示；未激活任何行（初始态）不展示。
  // 用于同步驱动「列表宽度固定 + 编辑区从右侧新增」：
  //  · 无编辑区 → 弹窗 md(560px)，右栏不渲染（右栏宽 0），左栏 flex-1 占满内容区（512px）；
  //  · 有编辑区 → 弹窗 xl(920px)，右栏固定 360px，左栏 flex-1 = 内容区 - 360px（仍 512px）。
  // 因 md 与 xl 内容区宽度差恰好等于右栏固定宽度 360px，两种状态下左栏 flex-1 实际宽度相同，
  // 即「列表面板宽度固定不变」，且两档弹窗宽度均在设计规范内（md / xl）。
  const showDetail =
    !hasNoOptions && (source?.slots ?? []).some((s) => s.slotId === activeSlotId);
  // 单角色实例：直接展示单栏「目标角色设置」编辑表单（不走「左列表 + 点击展开」双栏交互）。
  const isSingle = (source?.slots?.length ?? 0) === 1;
  // 是否展示右侧编辑区（单角色恒展示；多角色需激活某行且有可切换选项）。
  const showEditor = isSingle ? !hasNoOptions : showDetail;

  // 多角色双栏固定高度分 2 档：根据 Agent 角色位数量选择，使弹窗高度适配内容、避免大面积空白，
  // 同一档内为固定高度（不随右栏激活/切换选中行的技能·风格长文变化而跳变）。
  //  · ≤6 个角色 → 400px；· 7 个及以上 → 500px（超出档位高度时两栏各自内部滚动）。
  const slotCount = source?.slots?.length ?? 0;
  const dualPaneHeightClass = slotCount <= 6 ? "h-[400px]" : "h-[500px]";

  // 全局重名校验：计算每个角色位的最终显示名，若同 Agent 内出现重复名称则存在冲突，用于禁用「确认切换」。
  // 显示名规则与右栏/落库一致：名称必填但允许留空，留空即回退到「当前角色名称」（首位取 Agent 名，其余取 slot.roleName）。
  const slots = source?.slots ?? [];
  const finalNames = slots.map((s, idx) => {
    const curName = (idx === 0 ? source?.name || "通用助手" : s.roleName || "通用助手").trim();
    const t = targets[s.slotId] ?? null;
    if (t) {
      return (targetNames[s.slotId]?.trim() || curName).trim();
    }
    return curName;
  });
  const hasNameConflict = finalNames.some(
    (name, i) => name.length > 0 && finalNames.indexOf(name) !== i,
  );
  // 运行中 Agent 重名校验：切换后的角色名称不能与「正在运行的其它 Agent」名称重复。
  // 排除当前 Agent 自身名称（切换作用于当前 Agent，其主角色名保持/回退到自身名不算冲突）。
  const runningNameSet = new Set(
    runningAgentNames.map((n) => n.trim()).filter((n) => n && n !== (source?.name || "").trim()),
  );
  const hasRunningNameConflict = finalNames.some(
    (name) => name.length > 0 && runningNameSet.has(name),
  );

  return (
    <>
    <Dialog
      open={!!source}
      onOpenChange={(open) => {
        if (!open) {
          // 配置加载进行中：点右上角 X / 按 ESC 与底部「我知道了」一致——
          // 关闭弹窗并清除加载状态，下次打开重新开始。
          if (loading) {
            const allResolved = loading.items.every((it) => !!it.resultStatus);
            if (!allResolved) {
              onBackgrounded(loading.id, loading.items.length, loading.name, loading.items);
            }
            setLoading(null);
            setTargets({});
            setTargetNames({});
            setActiveSlotId(null);
            onClose();
            return;
          }
          onClose();
        }
      }}
    >
      {/* 尺寸档位：单角色（上下布局，单列表单）→ md(560px)，比多角色双栏小一号；
          多角色 → xl(920px)。
          进度加载态（loading）宽度须与对应编辑态一致（单角色 md / 多角色 xl），
          使「切换进度弹窗」与「批量切换角色弹窗」同宽，避免进入进度后弹窗宽度跳变。
          高度：单角色场景收窄最大高度上限（默认 100dvh-64px 会被角色风格长文撑到近满屏），
          取 min(740px, 视口可用高)；多角色双栏已由内部固定高度各自滚动，保持默认上限。 */}
      <DialogContent
        size={isSingle ? "md" : "xl"}
        // 右上角关闭按钮（X）在所有阶段都保留：
        // · 编辑态点 X → onClose 关闭弹窗；
        // · 加载态点 X → 与底部「我知道了」一致，转入后台继续配置（见上方 onOpenChange 对 loading 的处理）。
        showCloseButton
        className={
          isSingle && !loading
            ? "h-[min(740px,calc(100dvh-4rem))]"
            : undefined
        }
      >
        <DialogHeader>
          <DialogTitle>
            <span className="inline-flex items-center gap-2">
              {loading ? (loading.items.every((it) => it.resultStatus) ? `切换角色完成（共 ${loading.items.length} 个）` : `正在切换角色（共 ${loading.items.length} 个）`) : isSingle ? "切换角色" : "批量切换角色"}
              {!loading && roleAddingCount > 0 && (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-normal bg-blue-50 text-blue-600 border border-blue-200">
                  <Loader2 className="size-3 animate-spin" />正在新增角色…
                </span>
              )}
            </span>
          </DialogTitle>
        </DialogHeader>
        {/* 多角色：内容不自身滚动（滚动交给内部两栏），用 scrollbarGutter:auto 覆盖默认 stable，
            使内容区宽度恒为「弹窗宽 − px-6(48px)」，保证 md / xl 下左栏严格等宽（列表宽度固定）。
            单角色：上下两卡片均不内部滚动，整体由 DialogBody 滚动，按规范使用 stable。 */}
        <DialogBody className={cn("px-6", isSingle && !loading && "pb-2")} style={{ scrollbarGutter: isSingle ? "stable" : "auto" }}>
          {loading ? (
            /* ===== 配置加载态：批量明细进度（Agent + 逐角色位 from → to + 各自三步子进度） ===== */
            <RoleConfigProgressContent
              roleName={loading.primaryRoleName || "通用助手"}
              roleSoul={
                getRoleIntro(
                  loading.primaryRoleName === "通用助手"
                    ? "__general__"
                    : (visibleRoles.find((r) => r.name === loading.primaryRoleName) ?? "__general__")
                ).soul
              }
              skillCount={loading.skillCount}
              mode="switch"
              agentName={loading.name}
              items={loading.items}
              percent={loading.percent}
              stepIndex={loading.stepIndex}
              onRetry={(idx) => {
                // "角色切换部分成功+重试成功"(oc-role-012): 重试后变为成功
                if (loading.id === "oc-role-012") {
                  setLoading({
                    ...loading,
                    items: loading.items.map((it, i) =>
                      i === idx ? { ...it, resultStatus: "success" as const, failedSkills: undefined } : it
                    ),
                  });
                } else {
                  // 其他agent（如 oc-role-011）：重试失败，弹toast
                  toast.error("技能安装失败", {
                    description: `以下技能安装失败：${loading.items[idx]?.failedSkills?.join("、") || "未知技能"}，请稍后重试或联系管理员处理。`,
                  });
                }
              }}
            />
          ) : (
            <>
              {/* 受限分组 Alert：置于内容区最上方，说明当前 Agent 所属分组的可切换角色范围受管理员限制。
                  · 空数组 [] → 仅可切换为通用助手；· 非空白名单 → 仅可切换为白名单内角色。 */}
              {source?.allowedRoleNames !== undefined && (
                <Alert variant="warning" className="mt-2">
                  <AlertInfoIcon />
                  <AlertDescription>
                    {source.allowedRoleNames.length > 0
                      ? `该 Agent 所在分组已被管理员限制角色范围，仅可切换为以下角色：${[...source.allowedRoleNames, "通用助手"].join("、")}。`
                      : "该 Agent 所在分组已被管理员限制角色范围，仅可切换为通用助手。"}
                  </AlertDescription>
                </Alert>
              )}
              {(scheme === 1 || scheme === 2) && (
                <Alert variant="info" className="mt-2">
                  <AlertInfoIcon />
                  <AlertDescription>
                    切换角色会转变角色风格，同时自动安装新角色技能，但不会删除已有技能。
                  </AlertDescription>
                </Alert>
              )}

              {/* 添加角色 + 切换角色按钮：表格框外（方案1纯切换不显示） */}
              {!isSingle && scheme === 2 && (
                <div className="flex items-center gap-2 mt-4 mb-2">
                  <Button variant="tenant-dialog-confirm" size="sm" className="h-7 px-3 text-xs gap-1" onClick={() => onAddRole?.()}>
                    <Plus className="size-3.5" />
                    新增角色
                  </Button>
                  {roleAddingCount > 0 && (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium bg-blue-50 text-blue-600 border border-blue-200">
                      <Loader2 className="size-3 animate-spin" />正在新增角色…
                    </span>
                  )}
                </div>
              )}

              {/* 左右两栏：左=角色列表；右=该行的角色卡片。固定高度以保证两栏各自独立滚动、避免高度跳变。
                  列表宽度固定的关键（严格等宽推导）：
                  · 右栏固定 360px（box-border，含 border-l），左栏 flex-1 占剩余；
                  · DialogBody 传 scrollbarGutter:auto 且内容不自身滚动（滚动交给内部两栏），
                    故内容区宽度=弹窗宽 − px-6(48px)，不受滚动条gutter 影响；
                  · md(560) 内容区 512、xl(920) 内容区 872，差值恰好 = 右栏 360px；
                  → 无卡片(md，右栏0) 左栏 512；有卡片(xl，右栏360) 左栏 872−360=512。两状态左栏完全等宽。*/}
              <div
                className={cn(
                  "flex",
                  "mt-4",
                  // 单角色：上下布局——上方「当前角色」精简卡片（仅类型 / 名称），下方「请选择目标角色」设置卡片；
                  //          两卡片均为自然高度、内部不滚动，超出由弹窗内容区整体滚动。
                  // 多角色：左右双栏，固定高度（按角色数量分 3 档，见 dualPaneHeightClass），
                  //          同档内高度恒定，避免激活右栏/切换选中行时因右栏长文把容器撑高导致跳变；
                  //          角色少时用小档减少底部空白，角色多时用大档容纳更多行。两栏各自内部滚动。
                  isSingle
                    ? "flex-col gap-4"
                    : cn(
                        "items-stretch rounded-[8px] border border-gray-200 overflow-hidden",
                        dualPaneHeightClass,
                      ),
                )}
              >
                {isSingle ? (
                  /* 单角色：上方「当前角色」卡片——与下方目标角色卡片同构（h-10 表头 + 内容区），
                     内容为单行压缩型（头像 + 角色类型 / 角色名称两组字段），不滚动 */
                  (() => {
                    const only = (source?.slots ?? [])[0];
                    const curType = only?.baseRoleName || only?.roleName || "通用助手";
                    const curName = source?.name || only?.roleName || "通用助手";
                    return (
                      <div className="rounded-[8px] border border-gray-200 bg-[var(--bg-grey-normal)] overflow-hidden">
                        {/* 表头：与「请选择目标角色」表头等高同色 */}
                        <div className="flex items-center h-10 px-4 border-b border-gray-200 bg-[var(--bg-grey-normal)] text-xs text-[var(--text-muted)]">
                          当前角色
                        </div>
                        {/* 内容行：两组字段上下两行（label 与值左右相接，label 定宽保证两行值左对齐） */}
                        <div className="flex items-center gap-4 px-4 py-4">
                          <AgentAvatar roleName={curType} size={36} className="shrink-0" />
                          <div className="min-w-0 flex-1 flex flex-col gap-1.5">
                            <div className="min-w-0 flex items-center gap-2">
                              <MetaMedium tone="primary" className="shrink-0 w-12">角色类型</MetaMedium>
                              <TruncatedTooltip text={curType}>
                                <MetaText as="span" tone="secondary" className="min-w-0 block truncate">
                                  {curType}
                                </MetaText>
                              </TruncatedTooltip>
                            </div>
                            <div className="min-w-0 flex items-center gap-2">
                              <MetaMedium tone="primary" className="shrink-0 w-12">角色名称</MetaMedium>
                              <TruncatedTooltip text={curName}>
                                <MetaText as="span" tone="secondary" className="min-w-0 block truncate">
                                  {curName}
                                </MetaText>
                              </TruncatedTooltip>
                            </div>
                          </div>
                        </div>
                      </div>
                    );
                  })()
                ) : scheme === 3 ? (
                /* 方案3：三栏布局——左栏（序号+当前角色类型/名称+删除）、中栏（目标角色下拉）、右栏（目标角色详情） */
                <>
                {/* 左栏：当前角色列表 */}
                <div className={cn("flex flex-col min-h-0", "w-[240px] shrink-0 border-r border-gray-200")}>
                  <div className="flex items-center justify-between h-10 px-3 border-b border-gray-200 bg-[var(--bg-grey-normal)] text-xs text-[var(--text-muted)] shrink-0">
                    <span>当前角色类型/名称</span>
                    <button type="button" className="inline-flex items-center gap-0.5 text-xs text-[var(--text-primary)] hover:text-[var(--text-emphasis)] transition-colors" onClick={() => onAddRole?.()}>
                      <Plus className="size-3.5" />
                      新增
                    </button>
                  </div>
                  <div className="overflow-y-auto overscroll-contain flex-1">
                    {(source?.slots ?? []).map((slot, index) => {
                      const isActive = activeSlotId === slot.slotId;
                      const curName = index === 0 ? (source?.name || "通用助手") : (slot.name || slot.roleName || "通用助手");
                      const curType = slot.baseRoleName || slot.roleName || "通用助手";
                      return (
                        <div
                          key={slot.slotId}
                          className={cn(
                            "group relative flex items-center gap-2 px-3 h-[60px] border-b border-gray-100 transition-colors cursor-pointer",
                            isActive ? "bg-[var(--bg-brand-selected)]" : "hover:bg-[var(--bg-grey-hover)]",
                          )}
                          onClick={() => setActiveSlotId(slot.slotId)}
                        >
                          <span className="text-[var(--text-weak)] tabular-nums text-xs w-4 shrink-0">{index + 1}</span>
                          <AgentAvatar roleName={curType} size={24} className="shrink-0" />
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-1">
                              <span className={cn("text-sm truncate text-[var(--text-primary)]", isActive && "font-medium")}>{curType}</span>
                            </div>
                            <div className="text-xs text-[var(--text-muted)] truncate">{curName}</div>
                          </div>
                          {/* 删除按钮（主角色不可删除） */}
                          {!slot.isMain && (
                            <button
                              type="button"
                              className="hidden group-hover:inline-flex shrink-0 items-center px-1.5 py-0.5 rounded text-xs text-red-500 hover:bg-red-50 transition-colors"
                              onClick={(e) => { e.stopPropagation(); setDeleteSlotConfirm({ slotId: slot.slotId, roleName: curName }); }}
                            >
                              删除
                            </button>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </div>
                {/* 中栏：目标角色选择 */}
                <div className={cn("flex flex-col min-h-0", "w-[240px] shrink-0 border-r border-gray-200")}>
                  <div className="flex items-center justify-between h-10 px-3 border-b border-gray-200 bg-[var(--bg-grey-normal)] text-xs text-[var(--text-muted)] shrink-0">
                    <span className="flex items-center">
                      切换角色为
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="inline-flex items-center justify-center ml-0.5 cursor-pointer text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors">
                            <AlertInfoIcon className="size-3.5" />
                          </span>
                        </TooltipTrigger>
                        <TooltipContent side="top" className="text-xs max-w-[260px]">
                          切换角色会转变角色风格，同时自动安装新角色技能，但不会删除已有技能。
                        </TooltipContent>
                      </Tooltip>
                    </span>
                    <button
                      type="button"
                      className="inline-flex items-center gap-0.5 text-xs text-[var(--text-primary)] hover:text-[var(--text-emphasis)] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                      disabled={!(source?.slots ?? []).some((s) => targets[s.slotId])}
                      onClick={() => {
                        const slots = source?.slots ?? [];
                        const cleared: Record<string, Role | "__general__" | null> = {};
                        slots.forEach((s) => { cleared[s.slotId] = null; });
                        setTargets(cleared);
                        setTargetNames({});
                        setSkillNames({});
                        if (slots.length > 0) setActiveSlotId(slots[0].slotId);
                      }}
                    >
                      <RotateCcw className="size-3.5" />
                      重置
                    </button>
                  </div>
                  <div className="overflow-y-auto overscroll-contain flex-1">
                    {(source?.slots ?? []).map((slot, index) => {
                      const target = targets[slot.slotId] ?? null;
                      const isActive = activeSlotId === slot.slotId;
                      const curRoleName = slot.baseRoleName || slot.roleName || "通用助手";
                      const curName = index === 0 ? (source?.name || "通用助手") : (slot.roleName || "通用助手");
                      const curType = curRoleName;
                      return (
                        <div
                          key={slot.slotId}
                          className={cn(
                            "flex items-center px-3 h-[60px] border-b border-gray-100 transition-colors cursor-pointer",
                            isActive ? "bg-[var(--bg-brand-selected)]" : "hover:bg-[var(--bg-grey-hover)]",
                          )}
                          onClick={() => setActiveSlotId(slot.slotId)}
                        >
                          <div className="flex-1 min-w-0" onClick={(e) => e.stopPropagation()}>
                            <Select
                              value={target === "__general__" ? "__general__" : target ? target.id : ""}
                              onValueChange={(v) => {
                                if (!v) return;
                                if (v === "__clear__") {
                                  setTargets((prev) => ({ ...prev, [slot.slotId]: null }));
                                  setTargetNames((prev) => { const n = { ...prev }; delete n[slot.slotId]; return n; });
                                  return;
                                }
                                const nextRole = v === "__general__" ? "__general__" as const : (visibleRoles.find((r) => r.id === v) ?? null);
                                setTargets((prev) => ({ ...prev, [slot.slotId]: nextRole }));
                                if (nextRole && nextRole !== "__general__") {
                                  setSkillNames((prev) => ({ ...prev, [slot.slotId]: new Set(nextRole.skills.map((s) => s.name)) }));
                                }
                                const newTypeName = nextRole === "__general__" ? "通用助手" : (nextRole?.name ?? curType);
                                const otherSlotNames = new Set(
                                  (source?.slots ?? [])
                                    .filter((s) => s.slotId !== slot.slotId)
                                    .map((s, i) => targetNames[s.slotId] ?? (i === 0 ? (source?.name || "通用助手") : (s.roleName || "通用助手")))
                                );
                                setTargetNames((prev) => ({ ...prev, [slot.slotId]: makeUniqueName(newTypeName, otherSlotNames) }));
                                setActiveSlotId(slot.slotId);
                              }}
                            >
                              <SelectTrigger tenant className="h-auto min-h-[40px] px-3 py-2 text-sm gap-1.5 rounded-[12px]">
                                <span className="flex items-center gap-1.5 min-w-0 flex-1 text-left">
                                  {target ? (
                                    <>
                                      <AgentAvatar roleName={target === "__general__" ? "通用助手" : target.name} size={24} className="shrink-0" />
                                      <span className="min-w-0 text-left">
                                        <span className="block truncate text-sm">{target === "__general__" ? "通用助手" : target.name}</span>
                                        {targetNames[slot.slotId] && (
                                          <span className="block truncate text-xs text-[var(--text-muted)]">{targetNames[slot.slotId]}</span>
                                        )}
                                      </span>
                                    </>
                                  ) : (
                                    <span className="truncate text-[var(--text-muted)]">默认保持不变</span>
                                  )}
                                </span>
                              </SelectTrigger>
                              <SelectContent tenant>
                                {curType !== "通用助手" && <SelectItem value="__general__">通用助手</SelectItem>}
                                {(source?.allowedRoleNames !== undefined
                                  ? visibleRoles.filter((r) => source!.allowedRoleNames!.includes(r.name))
                                  : visibleRoles
                                ).filter((r) => r.name !== curType).map((r) => (
                                  <SelectItem key={r.id} value={r.id}>{r.name}</SelectItem>
                                ))}
                                {target && (
                                  <>
                                    <SelectSeparator />
                                    <SelectItem value="__clear__" className="text-[var(--text-muted)]">清除选项</SelectItem>
                                  </>
                                )}
                              </SelectContent>
                            </Select>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
                </>
                ) : (
                <div className={cn("flex flex-col min-h-0", "flex-1 min-w-0")}>
                  <div className="flex items-center h-10 px-3 border-b border-gray-200 bg-[var(--bg-grey-normal)] text-xs text-[var(--text-muted)] shrink-0">
                    <span className="w-8 shrink-0" aria-hidden />
                    <span className="flex-1 min-w-0">当前角色类型/名称</span>
                    <span className="w-3 shrink-0" aria-hidden />
                    <span className="flex-1 min-w-0">目标角色类型/名称</span>
                    {scheme !== 1 && <span className="w-[60px] shrink-0" aria-hidden />}
                  </div>
                  <div className="overflow-y-auto overscroll-contain flex-1">
                    {(source?.slots ?? []).map((slot, index) => {
                      const target = targets[slot.slotId] ?? null;
                      const isActive = activeSlotId === slot.slotId;
                      // 当前角色类型：优先 baseRoleName（改名后类型不变）
                      const curRoleName = slot.baseRoleName || slot.roleName || "通用助手";
                      // 当前角色名称/类型：首行取 Agent 显示名；类型锚定 baseRoleName（改名后类型不变）
                      const curName = index === 0 ? (source?.name || "通用助手") : (slot.roleName || "通用助手");
                      const curType = curRoleName;
                      return (
                        <div
                          key={slot.slotId}
                          className={cn(
                            "flex items-center px-3 py-2.5 border-b border-gray-100 transition-colors cursor-pointer",
                            isActive ? "bg-[var(--bg-brand-selected)]" : "hover:bg-[var(--bg-grey-hover)]",
                          )}
                          onClick={() => setActiveSlotId(slot.slotId)}
                        >
                          {/* 序号 */}
                          <div className="w-8 shrink-0 flex items-center">
                            <span className="text-[var(--text-weak)] tabular-nums text-sm">{index + 1}</span>
                          </div>
                          {/* 当前角色类型/名称：类型为主文字，名称为次级文字；主角色在类型后加tag */}
                          <div className="flex-1 min-w-0 flex items-center gap-1.5">
                            <AgentAvatar roleName={curType} size={24} className="shrink-0" />
                            <div className="min-w-0">
                              <div className="flex items-center gap-1">
                                <span className="text-sm text-[var(--text-primary)] truncate">{curType}</span>
                              </div>
                              <div className="text-xs text-[var(--text-muted)] truncate">{curName}</div>
                            </div>
                          </div>
                          {/* 间距 */}
                          <div className="w-3 shrink-0" />
                          {/* 目标角色类型/名称：下拉选择 + 名称次级文字 */}
                          <div className="flex-1 min-w-0" onClick={(e) => e.stopPropagation()}>
                            <Select
                              value={target === "__general__" ? "__general__" : target ? target.id : ""}
                              onValueChange={(v) => {
                                if (!v) return;
                                if (v === "__clear__") {
                                  setTargets((prev) => ({ ...prev, [slot.slotId]: null }));
                                  setTargetNames((prev) => { const n = { ...prev }; delete n[slot.slotId]; return n; });
                                  return;
                                }
                                const nextRole = v === "__general__" ? "__general__" as const : (visibleRoles.find((r) => r.id === v) ?? null);
                                setTargets((prev) => ({ ...prev, [slot.slotId]: nextRole }));
                                if (nextRole && nextRole !== "__general__") {
                                  setSkillNames((prev) => ({ ...prev, [slot.slotId]: new Set(nextRole.skills.map((s) => s.name)) }));
                                }
                                // 自动更新角色名称为新类型名（重名则加序号）
                                const newTypeName = nextRole === "__general__" ? "通用助手" : (nextRole?.name ?? curType);
                                const otherSlotNames = new Set(
                                  (source?.slots ?? [])
                                    .filter((s) => s.slotId !== slot.slotId)
                                    .map((s, i) => targetNames[s.slotId] ?? (i === 0 ? (source?.name || "通用助手") : (s.roleName || "通用助手")))
                                );
                                setTargetNames((prev) => ({ ...prev, [slot.slotId]: makeUniqueName(newTypeName, otherSlotNames) }));
                                setActiveSlotId(slot.slotId);
                              }}
                            >
                              <SelectTrigger tenant className="h-auto min-h-[40px] px-3 py-1.5 text-sm gap-1.5 rounded-[12px]">
                                <span className="flex items-center gap-1.5 min-w-0 flex-1 text-left">
                                  {target ? (
                                    <>
                                      <AgentAvatar roleName={target === "__general__" ? "通用助手" : target.name} size={24} className="shrink-0" />
                                      <span className="min-w-0 text-left">
                                        <span className="block truncate">{target === "__general__" ? "通用助手" : target.name}</span>
                                        {targetNames[slot.slotId] && (
                                          <span className="block truncate text-xs text-[var(--text-muted)]">{targetNames[slot.slotId]}</span>
                                        )}
                                      </span>
                                    </>
                                  ) : (
                                    <span className="truncate text-[var(--text-muted)]">默认保持不变</span>
                                  )}
                                </span>
                              </SelectTrigger>
                              <SelectContent tenant>
                                {curType !== "通用助手" && <SelectItem value="__general__">通用助手</SelectItem>}
                                {(source?.allowedRoleNames !== undefined
                                  ? visibleRoles.filter((r) => source!.allowedRoleNames!.includes(r.name))
                                  : visibleRoles
                                ).filter((r) => r.name !== curType).map((r) => (
                                  <SelectItem key={r.id} value={r.id}>{r.name}</SelectItem>
                                ))}
                                {target && (
                                  <>
                                    <SelectSeparator />
                                    <SelectItem value="__clear__" className="text-[var(--text-muted)]">清除选项</SelectItem>
                                  </>
                                )}
                              </SelectContent>
                            </Select>
                          </div>
                          {/* 操作列：删除（主角色禁用+tooltip）— 方案1不显示 */}
                          {scheme !== 1 && <div className="w-[60px] shrink-0 flex items-center justify-end pr-1">
                            {slot.isMain ? (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="text-xs text-gray-300 cursor-not-allowed">删除</span>
                                </TooltipTrigger>
                                <TooltipContent side="top" className="text-xs">
                                  主角色不可删除
                                </TooltipContent>
                              </Tooltip>
                            ) : (
                              <button
                                type="button"
                                className="text-xs text-[var(--text-muted)] hover:text-red-500 transition-colors"
                                onClick={(e) => { e.stopPropagation(); setDeleteSlotConfirm({ slotId: slot.slotId, roleName: curName }); }}
                              >
                                删除
                              </button>
                            )}
                          </div>}
                        </div>
                      );
                    })}
                  </div>
                </div>
                )}

                {/* 右侧「目标角色設置」编辑区：多角色时激活某行才出现（xl 双栏）；单角色时默认展示。
                    「角色类型」用下拉选择目标，「角色名称」用输入框自定义显示名；两者的修改实时同步到左栏文字。 */}
                {(() => {
                  if (!showEditor) return null;
                  // 单角色：始终锚定唯一角色位（即使 activeSlotId 被 toggle 清空，编辑区也不消失）。
                  const activeSlot = isSingle
                    ? (source?.slots ?? [])[0]
                    : (source?.slots ?? []).find((s) => s.slotId === activeSlotId);
                  if (!activeSlot) return null;
                  const activeTarget = targets[activeSlot.slotId] ?? null;
                  // 当前角色类型（未选目标时右栏默认展示它，可直接编辑）
                  const activeCurType = activeSlot.baseRoleName || activeSlot.roleName || "通用助手";
                  const curTypeRole = visibleRoles.find((r) => r.name === activeCurType) ?? null;
                  // 当前角色名称（该角色位的原显示名）：首位取 Agent 名，其余取 slot.roleName。
                  // 用作「角色名称」留空时的默认值（必填但允许留空，留空即保持当前名称）。
                  const activeIndex = (source?.slots ?? []).findIndex((s) => s.slotId === activeSlot.slotId);
                  const curNameForActive =
                    (activeIndex === 0 ? source?.name || "通用助手" : activeSlot.roleName || "通用助手").trim();
                  // 未选目标时用哨兵值：右栏名称框据此禁用并提示先在左栏选类型。
                  const RIGHT_UNSET = "__unset__";
                  // 目标标识值：已选目标 → 其 id / __general__；未选 → 哨兵。（类型选择已移至左栏目标列下拉）
                  const rightTypeValue =
                    activeTarget === "__general__"
                      ? "__general__"
                      : activeTarget
                        ? activeTarget.id
                        : RIGHT_UNSET;
                  // intro（技能/风格）：已选目标 → 跟随目标类型；未选 → 展示当前角色信息作为对照。
                  const introTarget: Role | "__general__" | null =
                    rightTypeValue === "__general__"
                      ? "__general__"
                      : rightTypeValue === RIGHT_UNSET
                        ? (curTypeRole ?? "__general__")
                        : (visibleRoles.find((r) => r.id === rightTypeValue) ?? null);
                  const intro = getRoleIntro(introTarget);
                  // 重名校验基准：计算「同 Agent 其它角色位」的最终显示名集合。
                  //  · 其它位已选目标 → 优先其自定义名（targetNames），否则目标类型名；
                  //  · 其它位未选目标 → 原显示名（首位取 Agent 名，其余取 slot.roleName）。
                  const slotDisplayName = (s: AgentRoleSlot, idx: number): string => {
                    const curName = (idx === 0 ? source?.name || "通用助手" : s.roleName || "通用助手").trim();
                    const t = targets[s.slotId] ?? null;
                    if (t) {
                      return (targetNames[s.slotId]?.trim() || curName).trim();
                    }
                    return curName;
                  };
                  const otherNames = new Set(
                    (source?.slots ?? [])
                      .map((s, idx) => ({ s, idx }))
                      .filter(({ s }) => s.slotId !== activeSlot.slotId)
                      .map(({ s, idx }) => slotDisplayName(s, idx)),
                  );
                  // 名称输入框值：未选目标类型（哨兵）时为空（禁用输入，引导先选类型）；
                  // 已选类型时优先用户自定义，否则回落到目标类型名（去重，避免与其它角色位重复）。
                  const rightName =
                    rightTypeValue === RIGHT_UNSET
                      ? ""
                      : (targetNames[activeSlot.slotId] ?? makeUniqueName(intro.name, otherNames));
                  const trimmedName = rightName.trim();
                  const nameError =
                    trimmedName.length > 0 && otherNames.has(trimmedName)
                      ? "该名称与当前 Agent 内的其它角色重复，请换一个名称"
                      : trimmedName.length > 0 && runningNameSet.has(trimmedName)
                        ? "该名称与正在运行的 Agent 重复，请换一个名称"
                        : "";
                  return (
                    <div
                      className={cn(
                        "flex flex-col bg-[var(--bg-grey-normal)]",
                        // 单角色：上下两卡片的「下」——独立卡片（圆角+边框），自然高度、内部不滚动，与上方卡片由外层 gap 留间距；
                        // 多角色方案3：右栏 flex-1 占满剩余宽度；方案2：右栏固定 360px + 左边框
                        isSingle
                          ? "rounded-[8px] border border-gray-200 overflow-hidden"
                          : scheme === 3
                            ? "flex-1 min-w-0 min-h-0"
                            : "w-[360px] shrink-0 min-h-0 border-l border-gray-200",
                      )}
                    >
                      {/* 表头行：与左栏表头等高（h-10）对齐，标注「角色详情」 */}
                      <div className="flex items-center h-10 px-4 border-b border-gray-200 bg-[var(--bg-grey-normal)] text-xs text-[var(--text-muted)] shrink-0">
                        目标角色详情
                      </div>
                      {/* 内容：角色名称(输入框) + 角色技能 + 角色风格（「角色类型」已移至左栏目标列下拉）。
                          单角色 → 自然高度、卡片内不滚动（超出由弹窗内容区整体滚动）；多角色 → 右栏内部纵向滚动。 */}
                      <div
                        className={cn(
                          "px-4 pt-4 pb-4 flex flex-col gap-4",
                          !isSingle && "flex-1 min-h-0 overflow-y-auto overscroll-contain",
                        )}
                        style={isSingle ? undefined : { scrollbarGutter: "stable" }}
                      >
                        {/* 角色类型：仅单角色场景在右栏保留（单角色为上下布局、无左栏目标列可放）；
                            多角色的类型选择已移到左栏「目标角色类型/名称」列的常驻下拉。
                            使用包含头像的大尺寸白底黑字选项卡组件（横向可滚动）。 */}
                        {isSingle && (() => {
                          const allowedS = source?.allowedRoleNames;
                          const singleCandidates = (
                            allowedS !== undefined ? visibleRoles.filter((r) => allowedS.includes(r.name)) : visibleRoles
                          ).filter((r) => r.name !== activeCurType);
                          // 构建全部选项：通用助手（当当前不是通用助手时）+ 候选角色列表
                          const allOptions: { value: string; label: string }[] = [];
                          if (activeCurType !== "通用助手") {
                            allOptions.push({ value: "__general__", label: "通用助手" });
                          }
                          singleCandidates.forEach((r) => allOptions.push({ value: r.id, label: r.name }));

                          const handleSelect = (v: string) => {
                            if (v === rightTypeValue) return;
                            if (v === RIGHT_UNSET) {
                              setTargets((prev) => ({ ...prev, [activeSlot.slotId]: null }));
                              setTargetNames((prev) => { const n = { ...prev }; delete n[activeSlot.slotId]; return n; });
                              setSkillNames((prev) => { const n = { ...prev }; delete n[activeSlot.slotId]; return n; });
                              return;
                            }
                            const nextRole = v === "__general__" ? "__general__" as const : (visibleRoles.find((r) => r.id === v) ?? null);
                            setTargets((prev) => ({ ...prev, [activeSlot.slotId]: nextRole }));
                            const nextTypeName = nextRole === "__general__" ? "通用助手" : (nextRole?.name ?? "");
                            const uniqueName = makeUniqueName(nextTypeName, otherNames);
                            setTargetNames((prev) => ({ ...prev, [activeSlot.slotId]: uniqueName }));
                            if (nextRole === "__general__" || !nextRole) {
                              setSkillNames((prev) => { const n = { ...prev }; delete n[activeSlot.slotId]; return n; });
                            } else {
                              setSkillNames((prev) => ({ ...prev, [activeSlot.slotId]: new Set(nextRole.skills.map((s) => s.name)) }));
                            }
                          };

                          return (
                            <div>
                              <MetaMedium tone="primary">角色类型</MetaMedium>
                              <RoleTypeRadioGroup
                                idPrefix={`batch-switch-${activeSlot.slotId}`}
                                className="mt-4"
                                value={rightTypeValue}
                                onValueChange={handleSelect}
                                options={allOptions}
                              />
                            </div>
                          );
                        })()}
                        {/* 多角色未选目标类型：整个「目标角色详情」仅展示一句引导文案 */}
                        {!isSingle && rightTypeValue === RIGHT_UNSET ? (
                          <div className="flex-1 min-h-0 flex items-center justify-center px-2">
                            <p className="text-center text-xs text-[var(--text-muted)] leading-relaxed">
                              {scheme === 3
                                ? "请先选择要切换的目标角色"
                                : <>请先在左侧「目标角色类型/名称」列<br />选择目标角色类型</>
                              }
                            </p>
                          </div>
                        ) : (
                        <>
                        {/* 角色类型（多角色）：只读展示当前所选目标类型 */}
                        {!isSingle && (
                          <div className="space-y-1.5">
                            <MetaMedium tone="primary">角色类型</MetaMedium>
                            <MetaText as="p" tone="secondary" className="leading-relaxed">
                              {rightTypeValue === "__general__" ? "通用助手" : intro.name}
                            </MetaText>
                          </div>
                        )}
                        {/* 角色名称：输入框可编辑 */}
                        <div className="space-y-0.5">
                          <div className="flex items-center">
                            <MetaMedium tone="primary">角色名称</MetaMedium>
                            <HelperText as="span" className="ml-2 shrink-0">{(targetNames[activeSlot.slotId] ?? curNameForActive).length}/{NAME_MAX_LEN}</HelperText>
                          </div>
                          <Input
                            tenant
                            className={cn("h-9 text-sm", nameError && "border-[var(--border-danger,#d42a1e)] focus-visible:ring-[var(--border-danger,#d42a1e)]")}
                            value={targetNames[activeSlot.slotId] ?? curNameForActive}
                            onChange={(e) => setTargetNames((prev) => ({ ...prev, [activeSlot.slotId]: e.target.value }))}
                            maxLength={NAME_MAX_LEN}
                            aria-invalid={!!nameError}
                          />
                          {nameError && (
                            <MetaText as="span" tone="danger" className="leading-snug">{nameError}</MetaText>
                          )}
                        </div>
                        {/* 角色技能 */}
                        <div className="space-y-1.5">
                          <MetaMedium tone="primary">角色技能</MetaMedium>
                          <MetaText as="p" tone="secondary" className="leading-relaxed">
                            {intro.skills}
                          </MetaText>
                        </div>
                        {/* 角色风格 */}
                        <div className="space-y-1.5">
                          <MetaMedium tone="primary">角色风格</MetaMedium>
                          <MetaText as="p" tone="secondary" className="leading-relaxed">
                            {intro.soul}
                          </MetaText>
                        </div>
                        </>
                        )}
                      </div>
                    </div>
                  );
                })()}
              </div>
            </>
          )}
        </DialogBody>
        {loading && (
          <DialogFooter className="justify-center">
            <Button
              variant="tenant-dialog-confirm"
              onClick={() => {
                // 仅在进度未完成时标记后台切换中（切换完成后不再显示胶囊）
                const allResolved = loading.items.every((it) => !!it.resultStatus);
                if (!allResolved) {
                  onBackgrounded(loading.id, loading.items.length, loading.name, loading.items);
                }
                setLoading(null);
                setTargets({});
                setTargetNames({});
                setActiveSlotId(null);
                onClose();
              }}
            >
              我知道了
            </Button>
          </DialogFooter>
        )}
        {!loading && (
          <DialogFooter className={isSingle || scheme === 3 ? "justify-end" : "justify-between"}>
            {/* 左下角「重置」：清空所有角色位已选的目标角色选项，回到「全部保持不变」初始态。无任何选择时禁用。
                仅多角色批量场景提供（单角色只有一行选择，直接改下拉即可，无需重置）。方案3重置已在中栏表头。 */}
            {!isSingle && scheme !== 3 && (
              <Button
                variant="tenant-outline"
                disabled={!(source?.slots ?? []).some((s) => targets[s.slotId])}
                onClick={() => {
                  const slots = source?.slots ?? [];
                  const cleared: Record<string, Role | "__general__" | null> = {};
                  slots.forEach((s) => {
                    cleared[s.slotId] = null;
                  });
                  setTargets(cleared);
                  setTargetNames({});
                  setSkillNames({});
                  setSkillPopoverSlot(null);
                  // 重置后保持选中第一行（右侧面板不消失）
                  if (slots.length > 0) {
                    setActiveSlotId(slots[0].slotId);
                  }
                }}
              >
                <RotateCcw />
                重置目标角色
              </Button>
            )}
            <div className="flex items-center gap-2">
              <Button variant="tenant-outline" onClick={onClose}>
                取消
              </Button>
              {(() => {
                // 确认切换按钮禁用理由：hover 时通过 Tooltip 提示用户为何不可点击。
                //  · 技能选择弹窗未收起 → 请先关闭技能选择面板；
                //  · 存在重名冲突 → 请先解决角色名称重复；
                //  · 未选择任何目标 → 请先为至少一个角色位选择目标角色。
                const hasSelection = (source?.slots ?? []).some((s) => targets[s.slotId]);
                const isDisabled = skillPopoverSlot !== null || hasNameConflict || hasRunningNameConflict || !hasSelection;
                const disabledReason = skillPopoverSlot !== null
                  ? "请先完成并关闭角色技能选择面板"
                  : hasNameConflict
                    ? "存在重复的角色名称，请修改后再切换"
                    : hasRunningNameConflict
                      ? "角色名称与正在运行的 Agent 重复，请修改后再切换"
                      : !hasSelection
                        ? "请先为至少一个角色位选择目标角色"
                        : "";
                const confirmButton = (
                  <Button
                    variant="tenant-dialog-confirm"
                    disabled={isDisabled}
                    // Button 基类为 disabled:pointer-events-auto（禁用仍吞指针事件），会挡住 hover 冒泡到
                    // 外层 TooltipTrigger(span)，导致 tooltip 不弹。禁用时用 !pointer-events-none 让事件穿透到 span。
                    className={isDisabled ? "!pointer-events-none" : undefined}
                    onClick={() => {
                      if (!source) return;
                      const slots = source.slots;
                      // 本次已选择目标选项的角色位明细（原角色 → 目标角色 + 目标技能数）
                      const items: RoleConfigProgressItem[] = slots.flatMap((s, idx) => {
                        const t = targets[s.slotId];
                        if (!t) return [];
                        // 角色名称必填但允许留空：留空则默认与「当前角色名称」保持一致（首位取 Agent 名，其余取 slot.roleName）。
                        const curName = (idx === 0 ? source.name : s.roleName) || "通用助手";
                        const toName = targetNames[s.slotId]?.trim() || curName;
                        const skillCount = t === "__general__" ? 5 : (skillNames[s.slotId]?.size ?? t.skills.length);
                        // 类型信息（进度明细项展示「类型 + 名称」两行）：
                        //  · 原角色类型锚定 baseRoleName（改名后类型不变），回退到 roleName；
                        //  · 目标类型为所选 Role 名 / 通用助手。
                        const fromType = s.baseRoleName || s.roleName || "通用助手";
                        const toType = t === "__general__" ? "通用助手" : t.name;
                        // 原角色名称：首位取 Agent 名、其余取 slot.roleName（与 curName 口径一致），
                        // 保证明细项「类型 + 名称」两行的名称行始终有值且准确。
                        return [{ slotId: s.slotId, fromName: curName, toName, fromType, toType, skillCount, isPrimary: s.isMain }];
                      });
                      if (items.length === 0) return;
                      // 不直接落库：进入配置加载动画，动画走完（100%）后统一落库并关闭弹窗
                      setLoading({
                        id: source.id,
                        name: source.name,
                        slots,
                        targets,
                        primaryRoleName: items[0].toName,
                        skillCount: items[0].skillCount,
                        items,
                        percent: 0,
                        stepIndex: 0,
                      });
                    }}
                  >
                    {(() => {
                      // 单角色只有一行选择，不带计数；多角色批量场景保留「（N）」提示本次改动条数
                      if (isSingle) return "确认";
                      const cnt = (source?.slots ?? []).filter((s) => targets[s.slotId]).length;
                      return `确认${cnt > 0 ? `（${cnt}）` : ""}`;
                    })()}
                  </Button>
                );
                // 未禁用时直接渲染按钮，避免多余浮层；禁用时用 span 包裹作为 Tooltip 触发器
                // （禁用按钮已被置为 pointer-events-none，hover 事件落到本 span 上触发 tooltip）。
                if (!isDisabled) return confirmButton;
                return (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="inline-flex cursor-not-allowed [pointer-events:auto]">{confirmButton}</span>
                    </TooltipTrigger>
                    <TooltipContent side="top" className="text-xs">
                      {disabledReason}
                    </TooltipContent>
                  </Tooltip>
                );
              })()}
            </div>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>

    {/* 编辑角色名称弹窗 */}
    <Dialog open={!!editNameSlot} onOpenChange={(o) => { if (!o) setEditNameSlot(null); }}>
      <DialogContent size="sm" showCloseButton>
        <DialogHeader>
          <DialogTitle>编辑角色名称</DialogTitle>
        </DialogHeader>
        <DialogBody className="px-6 py-4">
          <div className="space-y-2">
            <MetaMedium as="label" tone="secondary">角色名称</MetaMedium>
            <Input
              value={editNameValue}
              onChange={(e) => setEditNameValue(e.target.value)}
              maxLength={NAME_MAX_LEN}
              placeholder="请输入角色名称"
            />
            <HelperText as="span">{editNameValue.length}/{NAME_MAX_LEN}</HelperText>
          </div>
        </DialogBody>
        <DialogFooter className="px-6 pb-6">
          <Button variant="tenant-outline" onClick={() => setEditNameSlot(null)}>取消</Button>
          <Button
            variant="tenant-primary"
            disabled={!editNameValue.trim()}
            onClick={() => {
              if (editNameSlot) {
                setTargetNames((prev) => ({ ...prev, [editNameSlot.slotId]: editNameValue.trim() }));
              }
              setEditNameSlot(null);
            }}
          >
            确认
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    {/* 删除角色二次确认弹窗 */}
    <Dialog open={!!deleteSlotConfirm} onOpenChange={(o) => { if (!o) setDeleteSlotConfirm(null); }}>
      <DialogContent size="sm" showCloseButton>
        <DialogHeader>
          <DialogTitle>删除角色</DialogTitle>
        </DialogHeader>
        <DialogBody className="px-6 py-4">
          <MetaText as="p" tone="secondary" className="leading-relaxed">
            确定要删除角色「<MetaText as="span" tone="primary" className="font-medium">{deleteSlotConfirm?.roleName}</MetaText>」吗？删除后该角色位将从当前 Agent 中移除，<MetaText as="span" tone="danger" className="font-medium">相关对话数据也将被一并清除，且无法恢复</MetaText>。
          </MetaText>
        </DialogBody>
        <DialogFooter>
          <Button variant="tenant-outline" onClick={() => setDeleteSlotConfirm(null)}>
            取消
          </Button>
          <Button
            variant="tenant-destructive"
            onClick={() => {
              if (deleteSlotConfirm) {
                onDeleteSlot?.(deleteSlotConfirm.slotId, deleteSlotConfirm.roleName);
                toast.success(`角色「${deleteSlotConfirm.roleName}」已删除`);
                // 如果删除的是当前选中行，清空选中态
                if (activeSlotId === deleteSlotConfirm.slotId) {
                  setActiveSlotId(null);
                }
              }
              setDeleteSlotConfirm(null);
            }}
          >
            确认删除
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
    </>
  );
}