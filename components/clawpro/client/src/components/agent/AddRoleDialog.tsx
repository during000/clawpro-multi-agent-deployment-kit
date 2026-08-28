/**
 * AddRoleDialog —— 「新增角色」独立弹窗（卡片页 / 设置详情页共用）
 *
 * 单一数据源：卡片页（MyOpenClaw）与设置详情页（OpenClawDetailGuide）此前各维护了一份
 * 结构不同的「新增角色」弹窗（一处用 RoleTypeRadioGroup + 角色名称输入框 + 重名校验，
 * 另一处用 PillRadioOption + 技能编辑 Popover），导致两页视觉/能力不一致且「改一处不同步另一处」。
 * 现统一抽取到本组件，两处仅通过 props 注入「打开数据源 / 校验用运行中 Agent 名 / 落库回调」即可，
 * 未来任何弹窗 UI 改动只需改本文件，两个页面自动联动保持完全一致。
 *
 * 组件内部自管理：选中角色、角色名称输入、重名校验态。
 * 落库业务差异由外部通过 onConfirm 注入（页面各自决定 handleApplyRoleChanges / applyDetailRoleChanges）。
 */
import { useEffect, useState } from "react";
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
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import { MetaMedium, MetaText, HelperText } from "@/components/ui/Typography";
import { RoleTypeRadioGroup } from "@/components/agent/RoleTypeRadioGroup";
import type { Role, AgentRoleSlot } from "@/lib/mockData";
import { cn } from "@/lib/utils";

/** 角色名称长度上限（与 BatchSwitchRoleDialog / RoleManageSheet 保持一致） */
const NAME_MAX_LEN = 25;

/**
 * 生成不与 taken 集合重复、且不超过 maxLen 的名称（与 BatchSwitchRoleDialog.makeUniqueName 同口径）：
 * base 未被占用则原样返回（超长先截断）；否则依次尝试 base2、base3…… 直到不重复。
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

/** 弹窗打开数据源：目标 Agent + 其现有角色位 + 分组可新增范围限制（null=关闭） */
export interface AddRoleSource {
  id: string;
  name: string;
  roles: AgentRoleSlot[];
  /** 分组限制：undefined=无限制；[]=仅通用助手（无预设角色可新增）；非空=白名单角色名 */
  allowedRoleNames?: string[];
}

/**
 * 汇总「本次新增角色名称不可与之重复」的已占用名集合：
 *  · 当前 Agent 内已存在的角色位名称（优先 slot.name，回退 slot.roleName）；
 *  · 运行中的其它 Agent 名称（排除当前 Agent 自身）。
 * 用于打开弹窗 / 切换角色类型时生成默认名的去重基准（makeUniqueName 的 taken）。
 */
function buildTakenNames(source: AddRoleSource, runningAgentNames: string[]): Set<string> {
  const selfName = (source.name || "").trim();
  const taken = new Set<string>();
  (source.roles ?? []).forEach((s) => {
    const n = (s.name?.trim() || s.roleName?.trim() || "");
    if (n) taken.add(n);
  });
  runningAgentNames
    .map((n) => n.trim())
    .filter((n) => n && n !== selfName)
    .forEach((n) => taken.add(n));
  return taken;
}

/** 确认新增回调载荷 */
export interface AddRoleConfirmPayload {
  /** 目标 Agent */
  target: AddRoleSource;
  /** 选中的预设角色 */
  role: Role;
  /** 追加新角色位后的完整角色位列表 */
  nextSlots: AgentRoleSlot[];
  /** 本次新增角色位的 slotId */
  newSlotId: string;
  /** 预装技能数量（用于配置进度） */
  skillCount: number;
  /** 用户填写的最终角色名称（已 trim） */
  roleName: string;
}

export interface AddRoleDialogProps {
  /** 打开数据源；null 表示关闭 */
  source: AddRoleSource | null;
  /** 请求关闭弹窗 */
  onClose: () => void;
  /** 可选真实角色（含 soul / skills） */
  visibleRoles: Role[];
  /** 生成某角色（或通用助手）的技能 / 风格介绍 */
  getRoleIntro: (target: Role | "__general__" | null) => { name: string; skills: string; soul: string };
  /** 正在运行的 Agent 名称集合：新增的角色名称不能与之重复（排除当前 Agent 自身由外部处理或组件内处理） */
  runningAgentNames?: string[];
  /** 确认新增（校验通过后触发，由页面注入落库 + 配置进度启动） */
  onConfirm: (payload: AddRoleConfirmPayload) => void;
}

export function AddRoleDialog({
  source,
  onClose,
  visibleRoles,
  getRoleIntro,
  runningAgentNames = [],
  onConfirm,
}: AddRoleDialogProps) {
  // ── 弹窗内部交互态 ──
  const [selected, setSelected] = useState<Role | null>(null);
  const [roleName, setRoleName] = useState("");

  // 受限分组判定与可新增候选
  const allowed = source?.allowedRoleNames;
  const isRestricted = allowed !== undefined;
  const addCandidates = isRestricted
    ? visibleRoles.filter((r) => allowed!.includes(r.name))
    : visibleRoles;
  const allowedLabel = allowed && allowed.length > 0 ? allowed.join("、") : "";

  // 打开弹窗时默认选中第一个候选角色 + 回填其名称（自动去重：若默认类型名已与本 Agent 现有角色 /
  // 运行中 Agent 重名，则追加序号得到唯一名，避免一打开就命中「名称重复」报错）
  useEffect(() => {
    if (source) {
      const first = addCandidates[0] ?? null;
      setSelected(first);
      setRoleName(first ? makeUniqueName(first.name, buildTakenNames(source, runningAgentNames)) : "");
    } else {
      setSelected(null);
      setRoleName("");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [source]);

  // 运行中 Agent 名称集合（排除当前 Agent 自身，新增作用于当前 Agent）
  const selfName = (source?.name || "").trim();
  const runningNameSet = new Set(
    runningAgentNames.map((n) => n.trim()).filter((n) => n && n !== selfName),
  );
  // 当前 Agent 已有角色位名称集合：用于校验新增角色名称不能与本 Agent 内已存在的角色重名
  const existingRoleNameSet = new Set(
    (source?.roles ?? [])
      .map((s) => (s.name?.trim() || s.roleName?.trim() || ""))
      .filter((n) => n.length > 0),
  );

  const trimmedName = roleName.trim();
  const nameError =
    trimmedName.length === 0
      ? "请输入角色名称"
      : existingRoleNameSet.has(trimmedName)
        ? "该名称与当前 Agent 已有角色重复，请换一个名称"
        : runningNameSet.has(trimmedName)
          ? "该名称与正在运行的 Agent 重复，请换一个名称"
          : "";

  const handleConfirm = () => {
    if (!selected || !source) return;
    const finalRoleName = trimmedName;
    // 名称二次校验：空 / 与运行中 Agent 重名 / 与当前 Agent 已有角色重名时阻止落库（与按钮禁用逻辑一致，双保险）
    if (!finalRoleName || runningNameSet.has(finalRoleName) || existingRoleNameSet.has(finalRoleName)) return;
    const originalSlots = source.roles;
    const newSlotId = `slot-new-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const nextSlots: AgentRoleSlot[] = [
      ...originalSlots,
      { slotId: newSlotId, roleName: selected.name, name: finalRoleName, isMain: false },
    ];
    const skillCount = selected.skills.length;
    onConfirm({ target: source, role: selected, nextSlots, newSlotId, skillCount, roleName: finalRoleName });
    onClose();
  };

  const intro = selected ? getRoleIntro(selected) : null;
  const confirmDisabled = !selected || !source || trimmedName.length === 0 || !!nameError;

  return (
    <Dialog open={!!source} onOpenChange={(open) => { if (!open) onClose(); }}>
      {/* 固定弹窗高度（约可容纳 3 行角色风格 + 其它字段）：保证长短角色风格文案下弹窗高度一致，
          内容超出时由 DialogBody（flex-1 + overflow-y-auto）整体滚动 */}
      <DialogContent className="sm:max-w-[720px] h-[520px]">
        <DialogHeader>
          <DialogTitle>新增角色</DialogTitle>
        </DialogHeader>
        <DialogBody className="px-6">
          <div className="flex flex-col gap-4">
            {/* 受限分组 Alert：置于内容区最上方，说明当前 Agent 所属分组的角色范围受管理员限制 */}
            {isRestricted && (
              <Alert variant="warning">
                <AlertInfoIcon />
                <AlertDescription>
                  {allowedLabel
                    ? `该 Agent 所在分组已被管理员限制角色范围，仅可新增以下角色：${allowedLabel}。`
                    : "该 Agent 所在分组已被管理员限制角色范围，暂无可新增的预设角色，仅保留通用助手。"}
                </AlertDescription>
              </Alert>
            )}
            <div className="flex flex-col gap-4">
              {addCandidates.length === 0 ? (
                <div className="text-center py-12 space-y-1">
                  <HelperText>暂无可新增的预设角色</HelperText>
                  <HelperText>当前分组仅开放通用助手，如需更多角色请联系管理员调整分组权限。</HelperText>
                </div>
              ) : (
                <>
                  <div>
                    <MetaMedium tone="primary">角色类型</MetaMedium>
                    <RoleTypeRadioGroup
                      idPrefix="add-role"
                      className="mt-1"
                      value={selected?.id ?? ""}
                      onValueChange={(value) => {
                        const role = visibleRoles.find((r) => r.id === value) ?? null;
                        setSelected(role);
                        // 切换角色类型时同步回填默认名，并自动去重（重名则追加序号），避免命中「名称重复」报错
                        setRoleName(
                          role && source
                            ? makeUniqueName(role.name, buildTakenNames(source, runningAgentNames))
                            : (role?.name ?? ""),
                        );
                      }}
                      options={addCandidates.map((role) => ({ value: role.id, label: role.name }))}
                    />
                  </div>

                  {selected && intro && (
                    <>
                      <div className="space-y-1">
                        <div className="flex items-center">
                          <MetaMedium tone="primary">角色名称</MetaMedium>
                          <HelperText as="span" className="ml-2">{roleName.length}/{NAME_MAX_LEN}</HelperText>
                        </div>
                        <Input
                          tenant
                          className={cn("h-9 text-sm", nameError && trimmedName.length > 0 && "border-[var(--border-danger,#d42a1e)] focus-visible:ring-[var(--border-danger,#d42a1e)]")}
                          value={roleName}
                          onChange={(e) => setRoleName(e.target.value)}
                          maxLength={NAME_MAX_LEN}
                          aria-invalid={!!nameError && trimmedName.length > 0}
                        />
                        {nameError && trimmedName.length > 0 && (
                          <MetaText as="span" tone="danger" className="leading-snug">{nameError}</MetaText>
                        )}
                      </div>
                      <div className="space-y-1.5">
                        <MetaMedium tone="primary">角色技能</MetaMedium>
                        <MetaText as="p" tone="secondary" className="leading-relaxed">{intro.skills}</MetaText>
                      </div>
                      <div className="space-y-1.5">
                        <MetaMedium tone="primary">角色风格</MetaMedium>
                        <MetaText as="p" tone="secondary" className="leading-relaxed">{intro.soul}</MetaText>
                      </div>
                    </>
                  )}
                </>
              )}
            </div>
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="tenant-outline" onClick={onClose}>
            取消
          </Button>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="tenant-dialog-confirm"
                disabled={confirmDisabled}
                className={confirmDisabled ? "!pointer-events-none" : undefined}
                onClick={handleConfirm}
              >
                确认新增
              </Button>
            </TooltipTrigger>
            {selected && trimmedName.length === 0 ? (
              <TooltipContent side="top">请输入角色名称</TooltipContent>
            ) : selected && existingRoleNameSet.has(trimmedName) ? (
              <TooltipContent side="top">角色名称与当前 Agent 已有角色重复，请修改后再新增</TooltipContent>
            ) : selected && runningNameSet.has(trimmedName) ? (
              <TooltipContent side="top">角色名称与正在运行的 Agent 重复，请修改后再新增</TooltipContent>
            ) : null}
          </Tooltip>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default AddRoleDialog;
