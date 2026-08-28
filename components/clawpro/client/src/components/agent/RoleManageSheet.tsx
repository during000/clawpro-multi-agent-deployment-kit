/**
 * RoleManageSheet — 角色管理抽屉（共享组件）
 *
 * 由「我的 Agent」(MyOpenClaw) 与「实例详情/设置页」(OpenClawDetailGuide) 共用，
 * 保证两处角色管理抽屉视觉与行为 100% 一致（此前两页各自内联复刻，改一处不同步另一处）。
 *
 * 设计原则（自包含 + 极简 props）：
 *  - 组件内部自持所有「抽屉内 UI 态」：editSlots / switchRoleTargets / 逐行技能子集 /
 *    展开态 / 逐行确认态 / 删除确认 / 改名弹窗 等，页面无需关心。
 *  - 通过 props 接收：数据源（dialog / visibleRoles）、进度 hook、后台态（switchingRoleAgents /
 *    addingRoleAgents）、以及「落库 / 打开子弹窗 / 跳转设置」回调。
 *  - 「新增角色」「切换角色（批量）」两个独立大弹窗仍留在各自页面，本组件仅通过
 *    onOpenAddRole / onOpenBatchSwitch 触发打开（两页行为一致，避免把落库耦合搬进组件）。
 */
import { useEffect, useRef, useState, Fragment } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogBody,
  DialogDescription,
} from "@/components/ui/dialog";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  TableActionCell,
} from "@/components/ui/table";
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";
import { SelectPanel, SelectPanelItem } from "@/components/ui/select-panel";
import { Select, SelectTrigger, SelectContent, SelectItem } from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { SurfaceInner } from "@/components/ui/Surface";
import { Separator } from "@/components/ui/separator";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  BodyText,
  MetaText,
  HelperText,
  BodyMedium,
  MetaMedium,
  PanelTitle,
  CompactText,
} from "@/components/ui/Typography";
import { Plus, Trash2, Loader2, Pencil, ChevronDown, ArrowLeftRight, Star } from "lucide-react";
import { AgentAvatar } from "@/components/agent/AgentAvatar";
import { RoleTypeRadioGroup } from "@/components/agent/RoleTypeRadioGroup";
import { SwitchRoleDialog } from "@/components/agent/SwitchRoleDialog";
import { useRoleConfigProgress } from "@/components/agent/RoleConfigProgress";
import type { Role, AgentRoleSlot } from "@/lib/mockData";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

/** 角色名称长度上限（与 BatchSwitchRoleDialog 保持一致） */
const NAME_MAX_LEN = 25;

/**
 * 「切换为」卡片内的角色名称输入框：受控组件，限制最多 NAME_MAX_LEN 字并显示字符计数。
 * 每当默认名称（角色类型切换）变化时，通过 key 重置内部状态回填最新默认名。
 */
function RoleNameInput({ defaultName }: { defaultName: string }) {
  const [value, setValue] = useState(defaultName);
  return (
    <>
      <div className="flex items-center justify-between">
        <MetaMedium tone="primary">角色名称</MetaMedium>
        <HelperText as="span">{value.length}/{NAME_MAX_LEN}</HelperText>
      </div>
      <Input
        tenant
        className="h-9 text-sm"
        value={value}
        maxLength={NAME_MAX_LEN}
        onChange={(e) => setValue(e.target.value.slice(0, NAME_MAX_LEN))}
      />
    </>
  );
}

/** 打开抽屉时携带的目标 Agent 数据（null=关闭） */
export interface RoleManageDialogData {
  id: string;
  name: string;
  roleName: string;
  roleCount?: number;
  allowedRoleNames?: string[];
  roles?: AgentRoleSlot[];
}

export interface RoleManageSheetProps {
  /** 打开数据；null=关闭 */
  dialog: RoleManageDialogData | null;
  /** Sheet open 变化（关闭时会先落库多角色改动，再置 null） */
  onOpenChange: (open: boolean) => void;
  /** 可选真实角色（含 soul / skills） */
  visibleRoles: Role[];
  /** 配置进度 hook（页面创建后透传，两页共用同一套进度 UI） */
  roleConfigProgress: ReturnType<typeof useRoleConfigProgress>;
  /** 「角色切换中」的 Agent：agentId → 数量（后台态提示） */
  switchingRoleAgents: Record<string, number>;
  /** 「角色新增中」的 Agent：agentId → 数量（可选） */
  addingRoleAgents?: Record<string, number>;
  /** 清除某 Agent 的「切换中」提示 */
  clearRoleSwitching: (agentId: string) => void;
  /** 技能名 → 描述（用于逐行技能编辑面板） */
  getSkillDescription: (name: string) => string;
  /** 多角色统一提交（新增/删除/切换三合一） */
  onApplyRoleChanges: (
    id: string,
    name: string,
    nextSlots: AgentRoleSlot[],
    targets: Record<string, Role | "__general__" | null>,
    originalSlots: AgentRoleSlot[],
  ) => void;
  /** 单角色 / 单行切换落库 */
  onSwitchRole: (
    id: string,
    name: string,
    targetRole: string,
    targetSlotId?: string,
    previousRoleName?: string,
  ) => void;
  /** 打开「新增角色」独立弹窗（弹窗本体在页面内） */
  onOpenAddRole: (payload: { id: string; name: string; roles: AgentRoleSlot[]; allowedRoleNames?: string[] }) => void;
  /** 打开「切换角色（批量）」独立弹窗（弹窗本体在页面内） */
  onOpenBatchSwitch: (payload: { id: string; name: string; roles: AgentRoleSlot[]; allowedRoleNames?: string[] }) => void;
  /** 「修改设置」跳转到该 Agent 的设置页 */
  onNavigateSettings: (id: string) => void;
}

export function RoleManageSheet({
  dialog,
  onOpenChange,
  visibleRoles,
  roleConfigProgress,
  switchingRoleAgents,
  addingRoleAgents,
  clearRoleSwitching,
  getSkillDescription,
  onApplyRoleChanges,
  onSwitchRole,
  onOpenAddRole,
  onOpenBatchSwitch,
  onNavigateSettings,
}: RoleManageSheetProps) {
  // ── 抽屉内 UI 态（全部内聚，页面无需关心） ──
  const [switchRoleTarget, setSwitchRoleTarget] = useState<Role | "__general__" | null>(null);
  const [switchRoleTargets, setSwitchRoleTargets] = useState<Record<string, Role | "__general__" | null>>({});
  const [expandedSlotId, setExpandedSlotId] = useState<string | null>(null);
  const [confirmingSlotId, setConfirmingSlotId] = useState<string | null>(null);
  const [confirmedSlotIds, setConfirmedSlotIds] = useState<Set<string>>(new Set());
  const [deleteSlotConfirm, setDeleteSlotConfirm] = useState<{ slotId: string; roleName: string } | null>(null);
  const [renameSlotTarget, setRenameSlotTarget] = useState<{ slotId: string; roleName: string; baseRoleName: string } | null>(null);
  const [renameValue, setRenameValue] = useState("");
  const [renameError, setRenameError] = useState("");
  const [editSlots, setEditSlots] = useState<AgentRoleSlot[]>([]);
  // 逐行「切换角色」弹窗内选中的预装技能子集：slotId → 技能名集合
  const [switchSlotSkillNames, setSwitchSlotSkillNames] = useState<Record<string, Set<string>>>({});
  const [switchSlotSkillPopoverSlot, setSwitchSlotSkillPopoverSlot] = useState<string | null>(null);
  const [switchSlotSkillsBackup, setSwitchSlotSkillsBackup] = useState<Set<string>>(new Set());
  const switchSlotSkillPanelRef = useRef<HTMLDivElement>(null);
  const [skillPanelWidth, setSkillPanelWidth] = useState<Record<string, number>>({});
  // 已单独提示过「切换成功」的 slotId（避免关闭时汇总提示重复）
  const announcedSwitchSlotsRef = useRef<Set<string>>(new Set());

  const currentSlotRoleName = dialog?.roleName ?? "通用助手";

  // 是否以「多设计师协作助手」同款映射表格式（抽屉）渲染：
  //   只要携带了角色位数据（roles.length >= 1）即走映射表抽屉，含以下三类：
  //     1) 真·多角色实例（roles.length > 1）；
  //     2) 受限分组实例（allowedRoleNames 存在，含空数组 []）——仅切换/新增受限；
  //     3) 单角色实例（roles.length === 1）——与多角色统一使用同款抽屉。
  // 调用方（MyOpenClaw / OpenClawDetailGuide）已对所有实例（含单角色 / 受限）合成 roles。
  const isMappingDialog = (
    d: { allowedRoleNames?: string[]; roles?: AgentRoleSlot[] } | null | undefined
  ): boolean => {
    const s = d?.roles;
    return !!s && s.length >= 1;
  };

  // 打开抽屉时初始化多角色映射表；关闭时清理全部 UI 态
  useEffect(() => {
    if (!dialog) {
      setSwitchRoleTargets({}); setExpandedSlotId(null); setEditSlots([]);
      setConfirmingSlotId(null); setConfirmedSlotIds(new Set());
      setDeleteSlotConfirm(null); setRenameSlotTarget(null);
      return;
    }
    const slots = dialog.roles;
    if (slots && isMappingDialog(dialog)) {
      const init: Record<string, Role | "__general__" | null> = {};
      slots.forEach((s) => { init[s.slotId] = null; });
      setSwitchRoleTargets(init);
      setExpandedSlotId(null);
      setConfirmingSlotId(null);
      setConfirmedSlotIds(new Set());
      // baseRoleName 锚定头像身份：缺省时回退到 roleName（打开时二者一致）
      setEditSlots(slots.map((s) => ({ ...s, baseRoleName: s.baseRoleName ?? s.roleName })));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dialog]);

  // 单角色实例：打开时给出默认目标，避免「确认切换」默认禁用（映射表格式跳过，走 switchRoleTargets）
  useEffect(() => {
    if (!dialog) return;
    if (isMappingDialog(dialog)) return;
    const allowedRoleNames = dialog.allowedRoleNames;
    const candidateRoles = allowedRoleNames
      ? visibleRoles.filter((r) => allowedRoleNames.includes(r.name))
      : visibleRoles;
    const switchableRoles = candidateRoles.filter((r) => r.name !== currentSlotRoleName);
    const canSwitchToGeneral = currentSlotRoleName !== "通用助手";
    setSwitchRoleTarget(canSwitchToGeneral ? "__general__" : switchableRoles[0] ?? null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dialog]);

  // 计算某角色位可切换的目标候选
  const computeRowOptions = (slotRoleName: string) => {
    const allowedRoleNames = dialog?.allowedRoleNames;
    const candidateRoles = allowedRoleNames
      ? visibleRoles.filter((r) => allowedRoleNames.includes(r.name))
      : visibleRoles;
    const switchableRoles = candidateRoles.filter((r) => r.name !== slotRoleName);
    const canSwitchToGeneral = slotRoleName !== "通用助手";
    return { switchableRoles, canSwitchToGeneral };
  };

  // 角色介绍（技能 + 风格）
  const getRoleIntro = (target: Role | "__general__" | null) => {
    const generalIntro = {
      name: "通用助手",
      skills: "web-search、file-reader、code-runner",
      soul: "无固定行业偏好的通用 AI 伙伴，擅长日常问答、信息检索与轻量创作，按需切换专业度",
    };
    if (!target || target === "__general__") return generalIntro;
    return {
      name: target.name,
      skills: target.skills.map((s) => s.name).join("、"),
      soul: target.soul,
    };
  };

  const closeAll = () => {
    setSwitchRoleTarget(null); setSwitchRoleTargets({}); setExpandedSlotId(null);
    setSwitchSlotSkillPopoverSlot(null); setSwitchSlotSkillNames({});
    onOpenChange(false);
  };

  return (
    <Sheet
      open={!!dialog}
      onOpenChange={(open) => {
        if (!open) {
          // 多角色 / 受限映射表抽屉：关闭即保存本次角色位增删改（抽屉模式无底部保存按钮）
          const slots = dialog?.roles;
          if (dialog && slots && isMappingDialog(dialog)) {
            onApplyRoleChanges(dialog.id, dialog.name, editSlots, switchRoleTargets, slots);
            return; // 落库回调内部负责置 null / 清理
          }
          closeAll();
        }
      }}
    >
      <SheetContent side="right" className="w-full sm:max-w-2xl flex flex-col p-0 gap-0">
        {(() => {
          const slots = dialog?.roles;
          // 映射表格式：真·多角色实例，或受限分组单角色实例（复用「多设计师协作助手」同款抽屉）
          const hasMultiSlots = isMappingDialog(dialog);
          const allowedRoleNames = dialog?.allowedRoleNames;
          const candidateRoles = allowedRoleNames
            ? visibleRoles.filter((r) => allowedRoleNames.includes(r.name))
            : visibleRoles;

          const singleSwitchable = candidateRoles.filter((r) => r.name !== currentSlotRoleName);
          const singleCanGeneral = currentSlotRoleName !== "通用助手";
          const hasAnyOption = hasMultiSlots ? true : singleCanGeneral || singleSwitchable.length > 0;

          // ═══ 空态：无可切换角色 ═══
          if (!hasAnyOption) {
            return (
              <>
                <DialogHeader>
                  <DialogTitle>
                    角色管理
                    <HelperText as="span" className="ml-1.5">{dialog?.name}</HelperText>
                  </DialogTitle>
                </DialogHeader>
                <div className="space-y-3">
                  <BodyText as="p" tone="secondary">
                    当前 Agent 仅开放「{currentSlotRoleName}」，暂无可切换角色。
                  </BodyText>
                  <BodyText as="p" tone="secondary">如需调整请联系管理员。</BodyText>
                </div>
                <DialogFooter>
                  <Button variant="tenant-primary" onClick={closeAll}>知道了</Button>
                </DialogFooter>
              </>
            );
          }

          // ═══ 多角色实例 / 受限分组实例：N→N 映射表 ═══
          if (hasMultiSlots) {
            const originalSlots = slots!;
            const originalIds = new Set(originalSlots.map((s) => s.slotId));
            // 受限分组能力判定：allowedRoleNames 存在（含空数组）即受限
            const isRestricted = allowedRoleNames !== undefined;
            // 可新增：受限时白名单内可选角色数 > 0
            const canAddRole = isRestricted ? allowedRoleNames!.length > 0 : true;
            // 可切换：受限时白名单排除当前角色 +（当前非通用助手可切回通用助手）> 0
            const canBatchSwitch = isRestricted
              ? (candidateRoles.some((r) => r.name !== currentSlotRoleName) || singleCanGeneral)
              : true;
            // 工具栏「批量切换角色」：仅当存在多个角色位时才有意义；单角色实例禁用（走行内「切换角色」即可）
            const canBatchSwitchToolbar = canBatchSwitch && editSlots.length > 1;
            const addDisabledTip = "该 Agent 属受限分组，管理员未开放可新增的角色。如需调整请联系管理员。";
            const switchDisabledTip = "该 Agent 属受限分组，管理员未开放其它角色，暂不可切换。如需调整请联系管理员。";
            // 工具栏批量切换禁用提示：受限无候选沿用受限文案，单角色则提示只有一个角色无需批量
            const batchSwitchDisabledTip = !canBatchSwitch
              ? switchDisabledTip
              : "当前仅有一个角色，无需批量切换。可直接使用该角色的「切换角色」。";

            return (
              <>
                <SheetHeader className="border-b border-[var(--border)] p-4 gap-1">
                  <SheetTitle className="text-[16px] leading-6 text-[var(--text-emphasis)]">
                    角色管理（{editSlots.length}）
                    <HelperText as="span" className="ml-1.5">当前Agent：{dialog?.name}</HelperText>
                  </SheetTitle>
                  <SheetDescription className="sr-only">{dialog?.name}</SheetDescription>
                </SheetHeader>
                <div className="flex-1 min-h-0 p-4">
                  {(switchingRoleAgents[dialog!.id] ?? 0) > 0 && (
                    <Alert variant="info" className="mb-3">
                      <Loader2 className="size-4 animate-spin" />
                      <AlertDescription>
                        正在为你切换 {switchingRoleAgents[dialog!.id]} 个角色，切换完成前请勿重复操作。
                      </AlertDescription>
                    </Alert>
                  )}
                  {(addingRoleAgents?.[dialog!.id] ?? 0) > 0 && (
                    <Alert variant="info" className="mb-3">
                      <Loader2 className="size-4 animate-spin" />
                      <AlertDescription>
                        正在为你新增 {addingRoleAgents![dialog!.id]} 个角色，新增完成前请勿重复操作。
                      </AlertDescription>
                    </Alert>
                  )}
                  {/* 受限分组提示：该 Agent 仅可使用管理员开放的角色（含空数组 = 仅通用助手） */}
                  {isRestricted && (
                    <Alert variant="warning" className="mb-3">
                      <AlertInfoIcon />
                      <AlertDescription>
                        {allowedRoleNames!.length > 0
                          ? `该 Agent 属受限分组，仅可使用管理员开放的角色（${allowedRoleNames!.join("、")}）。如需调整请联系管理员。`
                          : "该 Agent 属受限分组，管理员仅开放「通用助手」，暂不可切换或新增其它角色。如需调整请联系管理员。"}
                      </AlertDescription>
                    </Alert>
                  )}
                  {/* 列表上方工具栏：新增角色（主按钮） + 切换角色（受限无候选时禁用并给出说明） */}
                  <div className="flex items-center gap-2">
                    <Tooltip delayDuration={200}>
                      <TooltipTrigger asChild>
                        <span className={canAddRole ? undefined : "inline-block cursor-not-allowed"}>
                          <Button
                            type="button"
                            variant="tenant-primary"
                            size="sm"
                            disabled={!canAddRole}
                            onClick={() => {
                              if (!dialog || !canAddRole) return;
                              onOpenAddRole({ id: dialog.id, name: dialog.name, roles: dialog.roles ?? [], allowedRoleNames: dialog.allowedRoleNames });
                            }}
                          >
                            <Plus className="size-4" />
                            新增角色
                          </Button>
                        </span>
                      </TooltipTrigger>
                      {!canAddRole && <TooltipContent side="top" className="text-xs">{addDisabledTip}</TooltipContent>}
                    </Tooltip>
                    <Tooltip delayDuration={200}>
                      <TooltipTrigger asChild>
                        <span className={canBatchSwitchToolbar ? undefined : "inline-block cursor-not-allowed"}>
                          <Button
                            type="button"
                            variant="tenant-outline"
                            size="sm"
                            disabled={!canBatchSwitchToolbar}
                            onClick={() => {
                              if (!dialog || !canBatchSwitchToolbar) return;
                              onOpenBatchSwitch({ id: dialog.id, name: dialog.name, roles: dialog.roles ?? [], allowedRoleNames: dialog.allowedRoleNames });
                            }}
                          >
                            <ArrowLeftRight className="size-4" />
                            批量切换角色
                          </Button>
                        </span>
                      </TooltipTrigger>
                      {!canBatchSwitchToolbar && <TooltipContent side="top" className="text-xs">{batchSwitchDisabledTip}</TooltipContent>}
                    </Tooltip>
                  </div>
                  <div className="py-2">
                    <Table
                      density="compact"
                      autoFixedColumns={false}
                      containerClassName="rounded-[8px] border border-gray-200"
                    >
                      <TableHeader>
                        <TableRow>
                          <TableHead className="w-1/3">角色类型/名称</TableHead>
                          <TableHead className="w-1/3">创建时间</TableHead>
                          <TableHead className="w-1/3">操作</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {editSlots.map((slot, slotIndex) => {
                          const isNewSlot = !originalIds.has(slot.slotId);
                          const { switchableRoles, canSwitchToGeneral } = computeRowOptions(slot.roleName);
                          const newSlotOptions: { value: string; label: string }[] = [
                            { value: "__general__", label: "通用助手" },
                            ...candidateRoles.map((r) => ({ value: r.id, label: r.name })),
                          ];
                          const target = switchRoleTargets[slot.slotId] ?? null;
                          const options: { value: string; label: string }[] = isNewSlot
                            ? newSlotOptions
                            : [
                                ...(canSwitchToGeneral ? [{ value: "__general__", label: "通用助手" }] : []),
                                ...switchableRoles.map((r) => ({ value: r.id, label: r.name })),
                              ];
                          const selectValue = target === "__general__" ? "__general__" : target?.id ?? "";
                          const expanded = expandedSlotId === slot.slotId;
                          const intro = getRoleIntro(target);
                          const currentRoleForIntro = isNewSlot
                            ? null
                            : (visibleRoles.find((r) => r.name === slot.roleName) ?? null);
                          const displayIntro = target ? intro : getRoleIntro(currentRoleForIntro);
                          const rowLoading = confirmingSlotId === slot.slotId;
                          const rowConfirmed = confirmedSlotIds.has(slot.slotId);
                          const targetName = target === "__general__" ? "通用助手" : target?.name;
                          const displayRoleName = rowConfirmed && targetName ? targetName : slot.roleName;
                          // 头像身份名：与显示名解耦。「修改名称」只改显示名（roleName），
                          // 头像应锚定 baseRoleName 保持不变；「切换角色」确认后才随新角色换头像。
                          const avatarRoleName = rowConfirmed && targetName
                            ? targetName
                            : (slot.baseRoleName ?? slot.roleName);
                          // 「创建时间」列：优先取 slot.createdAt；缺失时按 slotId 生成确定性回退时间。
                          // 第二行（slotIndex===1）作为「创建时间为空」的 mock 数据，返回空 → 由渲染处统一展示占位符「-」；
                          // 新增行 / 时间无效等空值场景同样为空并展示「-」。
                          const displayCreatedAt = (() => {
                            if (slotIndex === 1) return "";
                            if (isNewSlot) return "";
                            const iso = slot.createdAt;
                            let d: Date;
                            if (iso) {
                              d = new Date(iso);
                            } else {
                              let hash = 0;
                              for (let i = 0; i < slot.slotId.length; i++) {
                                hash = (hash * 31 + slot.slotId.charCodeAt(i)) >>> 0;
                              }
                              const base = new Date("2026-08-03T14:34:00").getTime();
                              d = new Date(base - (hash % 90) * 24 * 60 * 60 * 1000 - (hash % 1440) * 60 * 1000);
                            }
                            if (Number.isNaN(d.getTime())) return "";
                            const pad = (n: number) => String(n).padStart(2, "0");
                            return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
                          })();
                          const changed = isNewSlot
                            ? !!target
                            : !!target && (target === "__general__" ? "通用助手" : target.name) !== slot.roleName;
                          const canDelete = editSlots.length > 1;
                          const newSlotDisplayName = target
                            ? (target === "__general__" ? "通用助手" : target.name)
                            : "";
                          return (
                            <Fragment key={slot.slotId}>
                              <TableRow>
                                <TableCell>
                                  {isNewSlot ? (
                                    <div className="flex items-center gap-2 min-w-0">
                                      <AgentAvatar roleName={newSlotDisplayName || "通用助手"} size={24} />
                                      {newSlotDisplayName ? (
                                        <span className="truncate">{newSlotDisplayName}</span>
                                      ) : (
                                        <HelperText as="span">待选择</HelperText>
                                      )}
                                      <span className="shrink-0 px-1.5 py-0.5 rounded-[4px] bg-[var(--muted)] text-[var(--text-weak)] leading-none">新增</span>
                                    </div>
                                  ) : (
                                    <div className="flex items-center gap-2 min-w-0">
                                      <AgentAvatar roleName={avatarRoleName} size={28} />
                                      <div className="min-w-0 flex-1">
                                        <div className="flex items-center gap-1">
                                          <span className="text-sm truncate">{displayRoleName}</span>
                                          {rowLoading && (
                                            <span className="shrink-0 inline-flex items-center gap-1 text-[var(--text-weak)]">
                                              <Loader2 className="size-3.5 animate-spin" />
                                              <MetaText as="span" tone="secondary">切换中</MetaText>
                                            </span>
                                          )}

                                        </div>
                                        <div className="text-xs text-[var(--text-muted)] truncate">
                                          {slot.name || displayRoleName}
                                        </div>
                                      </div>
                                    </div>
                                  )}
                                </TableCell>
                                <TableCell>
                                  {displayCreatedAt ? (
                                    <MetaText as="p" tone="secondary" className="whitespace-nowrap">{displayCreatedAt}</MetaText>
                                  ) : (
                                    <HelperText as="span">-</HelperText>
                                  )}
                                </TableCell>
                                <TableActionCell rawChildren>
                                  <div className="flex items-center justify-start gap-3">
                                    {/* 修改设置：跳转到该 Agent 的设置页 */}
                                    <Button
                                      type="button"
                                      variant="link"
                                      size="sm"
                                      onClick={() => {
                                        if (!dialog) return;
                                        onNavigateSettings(dialog.id);
                                      }}
                                    >
                                      修改设置
                                    </Button>
                                    {/* 切换角色：打开该角色位的单行切换弹窗（受限无候选时禁用并给出说明） */}
                                    <Tooltip delayDuration={200}>
                                      <TooltipTrigger asChild>
                                        <span className={canBatchSwitch ? undefined : "inline-block cursor-not-allowed"}>
                                          <Button
                                            type="button"
                                            variant="link"
                                            size="sm"
                                            disabled={!canBatchSwitch}
                                            onClick={() => {
                                              if (!canBatchSwitch) return;
                                              // 打开弹窗时默认选中「角色类型」第一项（若该行尚未选择目标）
                                              if (!switchRoleTargets[slot.slotId] && options.length > 0) {
                                                const first = options[0].value;
                                                let chosen: Role | "__general__" | null = null;
                                                if (first === "__general__") chosen = "__general__";
                                                else chosen = visibleRoles.find((r) => r.id === first) ?? null;
                                                if (chosen && chosen !== "__general__") {
                                                  setSwitchSlotSkillNames((prev) => ({ ...prev, [slot.slotId]: new Set(chosen.skills.map((s) => s.name)) }));
                                                } else {
                                                  setSwitchSlotSkillNames((prev) => { const n = { ...prev }; delete n[slot.slotId]; return n; });
                                                }
                                                setSwitchRoleTargets((prev) => ({ ...prev, [slot.slotId]: chosen }));
                                              }
                                              setExpandedSlotId(slot.slotId);
                                            }}
                                          >
                                            切换角色
                                          </Button>
                                        </span>
                                      </TooltipTrigger>
                                      {!canBatchSwitch && <TooltipContent side="top" className="text-xs">{switchDisabledTip}</TooltipContent>}
                                    </Tooltip>
                                    <SwitchRoleDialog
                                      open={expanded}
                                      onClose={() => setExpandedSlotId(null)}
                                      slot={slot}
                                      target={target}
                                      selectValue={selectValue}
                                      options={options}
                                      displayIntro={displayIntro}
                                      displayRoleName={displayRoleName}
                                      changed={changed}
                                      visibleRoles={visibleRoles}
                                      switchSlotSkillNames={switchSlotSkillNames}
                                      switchSlotSkillPopoverSlot={switchSlotSkillPopoverSlot}
                                      setSwitchRoleTargets={setSwitchRoleTargets}
                                      setSwitchSlotSkillNames={setSwitchSlotSkillNames}
                                      setSwitchSlotSkillPopoverSlot={setSwitchSlotSkillPopoverSlot}
                                      setConfirmingSlotId={setConfirmingSlotId}
                                      setConfirmedSlotIds={setConfirmedSlotIds}
                                      roleConfigProgress={roleConfigProgress}
                                      getRoleIntro={getRoleIntro}
                                      clearRoleSwitching={clearRoleSwitching}
                                      agentId={dialog?.id}
                                      agentName={dialog?.name}
                                      announcedSwitchSlotsRef={announcedSwitchSlotsRef}
                                      switchSlotSkillPanelRef={switchSlotSkillPanelRef}
                                    />
                                    {/* 删除按钮：受限分组 / 仅剩一个角色位时禁用并给出说明 */}
                                    {!canDelete ? (
                                      <Tooltip delayDuration={200}>
                                        <TooltipTrigger asChild>
                                          <span className="inline-block cursor-not-allowed">
                                            <Button
                                              type="button"
                                              variant="link"
                                              size="sm"
                                              disabled
                                            >
                                              删除
                                            </Button>
                                          </span>
                                        </TooltipTrigger>
                                        <TooltipContent side="top" className="text-xs max-w-[240px]">
                                          {isRestricted
                                            ? "该 Agent 属受限分组，暂不支持删除等角色位管理操作。如需调整请联系管理员。"
                                            : "至少需保留一个角色位。"}
                                        </TooltipContent>
                                      </Tooltip>
                                    ) : (
                                      <Button
                                        type="button"
                                        variant="link"
                                        size="sm"
                                        className="!text-red-500 hover:!text-red-600"
                                        disabled={!canDelete}
                                        onClick={() => setDeleteSlotConfirm({ slotId: slot.slotId, roleName: displayRoleName })}
                                      >
                                        删除
                                      </Button>
                                    )}
                                  </div>
                                </TableActionCell>
                              </TableRow>
                            </Fragment>
                          );
                        })}
                      </TableBody>
                    </Table>
                  </div>
                </div>
                {/* 删除角色位二次确认 */}
                <AlertDialog open={!!deleteSlotConfirm} onOpenChange={(open) => { if (!open) setDeleteSlotConfirm(null); }}>
                  <AlertDialogContent className="sm:max-w-[420px]">
                    <AlertDialogHeader>
                      <AlertDialogTitle asChild>
                        <PanelTitle>确认删除角色？</PanelTitle>
                      </AlertDialogTitle>
                    </AlertDialogHeader>
                    <AlertDialogDescription asChild>
                      <BodyText as="p" tone="primary">
                        确定要删除角色 <BodyMedium tone="primary">{deleteSlotConfirm?.roleName}</BodyMedium> 吗？
                        <BodyText as="span" tone="danger">删除后该角色位配置将被移除，此操作不可撤销。</BodyText>
                      </BodyText>
                    </AlertDialogDescription>
                    <AlertDialogFooter>
                      <Button variant="tenant-outline" onClick={() => setDeleteSlotConfirm(null)}>取消</Button>
                      <Button
                        variant="tenant-destructive"
                        onClick={() => {
                          if (!deleteSlotConfirm) return;
                          const sid = deleteSlotConfirm.slotId;
                          setEditSlots((prev) => {
                            const removed = prev.find((s) => s.slotId === sid);
                            const rest = prev.filter((s) => s.slotId !== sid);
                            // 若删除的是主角色且仍有剩余角色位，则将剩余第一个提升为主角色，保证始终存在主角色
                            if (removed?.isMain && rest.length > 0 && !rest.some((s) => s.isMain)) {
                              return rest.map((s, i) => (i === 0 ? { ...s, isMain: true } : s));
                            }
                            return rest;
                          });
                          setSwitchRoleTargets((prev) => { const next = { ...prev }; delete next[sid]; return next; });
                          setConfirmedSlotIds((prev) => {
                            if (!prev.has(sid)) return prev;
                            const next = new Set(prev);
                            next.delete(sid);
                            return next;
                          });
                          if (expandedSlotId === sid) setExpandedSlotId(null);
                          if (confirmingSlotId === sid) setConfirmingSlotId(null);
                          setDeleteSlotConfirm(null);
                        }}
                      >
                        删除
                      </Button>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
                {/* 修改角色名称 */}
                <Dialog open={!!renameSlotTarget} onOpenChange={(open) => { if (!open) { setRenameSlotTarget(null); setRenameValue(""); setRenameError(""); } }}>
                  <DialogContent className="sm:max-w-[420px]">
                    <DialogHeader>
                      <DialogTitle>修改角色名称</DialogTitle>
                    </DialogHeader>
                    <DialogBody className="px-6">
                      <div className="space-y-1.5">
                        <MetaMedium as="label" tone="secondary" htmlFor="role-manage-rename-input">
                          角色名称
                          <HelperText as="span" className="ml-1.5">{renameValue.length}/{NAME_MAX_LEN}</HelperText>
                        </MetaMedium>
                        <Input
                          id="role-manage-rename-input"
                          value={renameValue}
                          maxLength={NAME_MAX_LEN}
                          placeholder="请输入角色名称"
                          className={renameError ? "border-red-400" : ""}
                          onChange={(e) => {
                            const val = e.target.value;
                            setRenameValue(val);
                            const trimmed = val.trim();
                            if (!trimmed) {
                              setRenameError("");
                            } else if (trimmed.length > NAME_MAX_LEN) {
                              setRenameError(`角色名称不超过 ${NAME_MAX_LEN} 个字`);
                            } else if (
                              editSlots.some((s) => s.slotId !== renameSlotTarget?.slotId && s.roleName === trimmed)
                            ) {
                              setRenameError(`同名角色「${trimmed}」已存在，请使用其他名称`);
                            } else {
                              setRenameError("");
                            }
                          }}
                          onKeyDown={(e) => { if (e.key === "Enter") e.preventDefault(); }}
                        />
                        {renameError && <MetaText as="p" tone="danger">{renameError}</MetaText>}
                      </div>
                    </DialogBody>
                    <DialogFooter>
                      <Button
                        variant="tenant-outline"
                        onClick={() => { setRenameSlotTarget(null); setRenameValue(""); setRenameError(""); }}
                      >
                        取消
                      </Button>
                      <Button
                        variant="tenant-dialog-confirm"
                        disabled={
                          !renameValue.trim() ||
                          !!renameError ||
                          renameValue.trim() === renameSlotTarget?.roleName
                        }
                        onClick={() => {
                          if (!renameSlotTarget) return;
                          const trimmed = renameValue.trim();
                          if (!trimmed || trimmed.length > NAME_MAX_LEN) return;
                          if (editSlots.some((s) => s.slotId !== renameSlotTarget.slotId && s.roleName === trimmed)) {
                            setRenameError(`同名角色「${trimmed}」已存在，请使用其他名称`);
                            return;
                          }
                          const sid = renameSlotTarget.slotId;
                          // 只改显示名 roleName；baseRoleName 锚定当前头像身份保持不变，确保改名后头像不变
                          const keepBase = renameSlotTarget.baseRoleName;
                          setEditSlots((prev) => prev.map((s) => (s.slotId === sid ? { ...s, roleName: trimmed, baseRoleName: keepBase } : s)));
                          toast.success(`角色名称已修改为「${trimmed}」`);
                          setRenameSlotTarget(null);
                          setRenameValue("");
                          setRenameError("");
                        }}
                      >
                        确认
                      </Button>
                    </DialogFooter>
                  </DialogContent>
                </Dialog>
              </>
            );
          }

          // ═══ 单角色实例：Pill 单选 + 角色介绍卡 ═══
          const switchableRoles = singleSwitchable;
          const canSwitchToGeneral = singleCanGeneral;
          return (
            <>
              <DialogHeader>
                <DialogTitle>
                  角色管理
                  <HelperText as="span" className="ml-1.5">{dialog?.name}</HelperText>
                </DialogTitle>
              </DialogHeader>
              <DialogDescription>
                为「{dialog?.name}」选择新角色（当前：{currentSlotRoleName}）。切换角色不会删除已有的技能配置，并将自动安装新角色的专属技能。
              </DialogDescription>
              <DialogBody className="px-6">
                <div className="py-2 space-y-3">
                  <Alert variant="info">
                    <AlertInfoIcon />
                    <AlertDescription>
                      切换角色会转变角色风格，同时自动安装新角色技能，但不会删除已有技能。
                    </AlertDescription>
                  </Alert>
                  {/* 当前角色展示卡片 */}
                  <div className="rounded-[8px] border border-gray-200 bg-[var(--bg-grey-normal)] overflow-hidden">
                    <div className="flex items-center h-10 px-4 border-b border-gray-200 bg-[var(--bg-grey-normal)] text-xs text-[var(--text-muted)]">
                      当前角色
                    </div>
                    <div className="flex items-center gap-4 px-4 py-3">
                      <div className="flex items-center gap-1.5 text-sm">
                        <MetaMedium tone="primary" as="span">角色类型</MetaMedium>
                        <AgentAvatar roleName={currentSlotRoleName} size={20} className="shrink-0" />
                        <MetaText as="span" tone="secondary">{currentSlotRoleName}</MetaText>
                      </div>
                      <div className="flex items-center gap-1.5 text-sm">
                        <MetaMedium tone="primary" as="span">角色名称</MetaMedium>
                        <MetaText as="span" tone="secondary">{dialog?.name || "通用助手"}</MetaText>
                      </div>
                    </div>
                  </div>
                  {/* 切换为卡片 */}
                  <div className="rounded-[8px] border border-gray-200 bg-[var(--bg-grey-normal)] overflow-hidden">
                    <div className="flex items-center h-10 px-4 border-b border-gray-200 bg-[var(--bg-grey-normal)] text-xs text-[var(--text-muted)]">
                      切换为
                    </div>
                    <div className="px-4 py-4">
                  {(() => {
                    const display = getRoleIntro(switchRoleTarget);
                    return (
                      <div className="flex flex-col gap-4">
                        <div className="space-y-2.5">
                          <MetaMedium tone="primary">角色类型</MetaMedium>
                          <RoleTypeRadioGroup
                            value={switchRoleTarget === "__general__" ? "__general__" : (switchRoleTarget as Role)?.id ?? ""}
                            onValueChange={(value) => {
                              if (value === "__general__") { setSwitchRoleTarget("__general__"); return; }
                              const role = visibleRoles.find((r) => r.id === value);
                              setSwitchRoleTarget(role ?? null);
                            }}
                            idPrefix="switch-role"
                            options={[
                              ...(canSwitchToGeneral ? [{ value: "__general__", label: "通用助手" }] : []),
                              ...switchableRoles.map((role) => ({ value: role.id, label: role.name })),
                            ]}
                          />
                        </div>
                        <div className="space-y-2.5">
                          <RoleNameInput key={switchRoleTarget === "__general__" ? "__general__" : (switchRoleTarget as Role)?.id ?? ""} defaultName={display.name} />
                        </div>
                        <div className="space-y-2.5">
                          <MetaMedium tone="primary">角色技能</MetaMedium>
                          <MetaText as="p" tone="secondary" className="leading-relaxed">{display.skills}</MetaText>
                        </div>
                        <div className="space-y-2.5">
                          <MetaMedium tone="primary">角色风格</MetaMedium>
                          <MetaText as="p" tone="secondary" className="leading-relaxed">{display.soul}</MetaText>
                        </div>
                      </div>
                    );
                  })()}
                    </div>
                  </div>
                </div>
              </DialogBody>
              <DialogFooter>
                <Button
                  variant="tenant-outline"
                  onClick={() => { setSwitchRoleTarget(null); onOpenChange(false); }}
                >
                  取消
                </Button>
                <Button
                  variant="tenant-primary"
                  disabled={!switchRoleTarget}
                  onClick={() => {
                    if (!dialog || !switchRoleTarget) return;
                    const targetName = switchRoleTarget === "__general__" ? "通用助手" : (switchRoleTarget as Role).name;
                    onSwitchRole(dialog.id, dialog.name, targetName, undefined, currentSlotRoleName);
                  }}
                >
                  确认切换
                </Button>
              </DialogFooter>
            </>
          );
        })()}
      </SheetContent>
    </Sheet>
  );
}

export default RoleManageSheet;
