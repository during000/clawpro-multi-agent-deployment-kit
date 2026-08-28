/**
 * SwitchRoleDialog —— 角色管理抽屉内「切换角色」独立弹窗（单个角色位切换）
 *
 * 由 RoleManageSheet 表格每行「切换角色」按钮唤起。此前该弹窗以内联 JSX 深嵌在
 * RoleManageSheet 的 editSlots.map() 闭包里，难以单独维护 / 复用；现抽取为独立命名组件，
 * 通过 props 注入所处行的渲染期计算值（slot / target / options / displayIntro 等）与回写 setter，
 * 后续任何该弹窗 UI 改动只需改本文件。
 *
 * 交互与原内联实现保持一致：
 *   - 顶部信息 Alert；
 *   - 「当前角色」只读卡片（类型 + 名称）；
 *   - 「切换为」卡片：角色类型（RoleTypeRadioGroup）+ 角色名称输入框 + 角色技能 + 角色风格；
 *   - 底部「取消 / 确认」，确认后启动配置进度动画并落库。
 */
import { useState } from "react";
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
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import { MetaText, HelperText, MetaMedium } from "@/components/ui/Typography";
import { AgentAvatar } from "@/components/agent/AgentAvatar";
import { RoleTypeRadioGroup } from "@/components/agent/RoleTypeRadioGroup";
import type { useRoleConfigProgress } from "@/components/agent/RoleConfigProgress";
import type { Role, AgentRoleSlot } from "@/lib/mockData";

/** 角色名称长度上限（与 RoleManageSheet / BatchSwitchRoleDialog 保持一致） */
const NAME_MAX_LEN = 25;

/** 角色介绍（名称 + 技能 + 风格），用于右栏展示 */
export interface SwitchRoleIntro {
  name: string;
  skills: string;
  soul: string;
}

export interface SwitchRoleDialogProps {
  /** 是否打开 */
  open: boolean;
  /** 请求关闭弹窗（取消 / 关闭按钮 / 点遮罩） */
  onClose: () => void;
  /** 当前操作的角色位 */
  slot: AgentRoleSlot;
  /** 当前行已选目标：Role / "__general__" / null（未选） */
  target: Role | "__general__" | null;
  /** RoleTypeRadioGroup 当前值：目标 id / "__general__" / ""（未选） */
  selectValue: string;
  /** 可切换目标选项（value + label） */
  options: { value: string; label: string }[];
  /** 展示用角色介绍（已根据 target / 当前角色计算好） */
  displayIntro: SwitchRoleIntro;
  /** 当前角色显示名（用于「当前角色」卡片） */
  displayRoleName: string;
  /** 是否发生有效变更（未变更时禁用「确认」） */
  changed: boolean;
  /** 可选真实角色（含 soul / skills） */
  visibleRoles: Role[];
  /** 逐行选中的预装技能子集：slotId → 技能名集合 */
  switchSlotSkillNames: Record<string, Set<string>>;
  /** 当前打开技能面板的 slotId（用于禁用「确认」） */
  switchSlotSkillPopoverSlot: string | null;
  /** 更新行目标选择 */
  setSwitchRoleTargets: React.Dispatch<React.SetStateAction<Record<string, Role | "__general__" | null>>>;
  /** 更新行技能子集 */
  setSwitchSlotSkillNames: React.Dispatch<React.SetStateAction<Record<string, Set<string>>>>;
  /** 更新技能面板打开态 */
  setSwitchSlotSkillPopoverSlot: React.Dispatch<React.SetStateAction<string | null>>;
  /** 标记某行进入「配置中」 */
  setConfirmingSlotId: React.Dispatch<React.SetStateAction<string | null>>;
  /** 更新「已确认」行集合 */
  setConfirmedSlotIds: React.Dispatch<React.SetStateAction<Set<string>>>;
  /** 配置进度 hook */
  roleConfigProgress: ReturnType<typeof useRoleConfigProgress>;
  /** 生成某角色（或通用助手）的技能 / 风格介绍 */
  getRoleIntro: (target: Role | "__general__" | null) => SwitchRoleIntro;
  /** 清除某 Agent 的「角色切换中」标记 */
  clearRoleSwitching: (id: string) => void;
  /** 所属 Agent id */
  agentId?: string;
  /** 所属 Agent 名称 */
  agentName?: string;
  /** 已播报过「切换成功」的 slotId 集合（避免重复 toast） */
  announcedSwitchSlotsRef: React.MutableRefObject<Set<string>>;
  /** 角色技能面板容器 ref（外部点击关闭面板用） */
  switchSlotSkillPanelRef: React.RefObject<HTMLDivElement>;
}

/**
 * 「切换为」卡片内的角色名称输入框：受控组件，限制最多 NAME_MAX_LEN 字并显示字符计数。
 * 每当默认名称（角色类型切换）变化时，通过 key 重置内部状态回填最新默认名。
 */
function RoleNameInput({ defaultName }: { defaultName: string }) {
  const [value, setValue] = useState(defaultName);
  return (
    <div className="space-y-1">
      <div className="flex items-center">
        <MetaMedium tone="primary">角色名称</MetaMedium>
        <HelperText as="span" className="ml-2">{value.length}/{NAME_MAX_LEN}</HelperText>
      </div>
      <Input
        tenant
        className="h-9 text-sm"
        value={value}
        maxLength={NAME_MAX_LEN}
        onChange={(e) => setValue(e.target.value.slice(0, NAME_MAX_LEN))}
      />
    </div>
  );
}

export function SwitchRoleDialog({
  open,
  onClose,
  slot,
  target,
  selectValue,
  options,
  displayIntro,
  displayRoleName,
  changed,
  visibleRoles,
  switchSlotSkillNames,
  switchSlotSkillPopoverSlot,
  setSwitchRoleTargets,
  setSwitchSlotSkillNames,
  setSwitchSlotSkillPopoverSlot,
  setConfirmingSlotId,
  setConfirmedSlotIds,
  roleConfigProgress,
  getRoleIntro,
  clearRoleSwitching,
  agentId,
  agentName,
  announcedSwitchSlotsRef,
  switchSlotSkillPanelRef,
}: SwitchRoleDialogProps) {
  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      {/* 固定弹窗高度：保证不同目标角色（角色风格长短不一）下弹窗高度恒定，切换角色类型时不跳变。
          高度取 min(680px, 视口可用高)——680px 刚好可完整展示「行业分析师」这类角色的整卡内容且底部无多余留白
          （Alert + 当前角色卡 + 切换为卡：角色类型两排选项卡 + 角色名称 + 角色技能 + 一行角色风格），
          风格更长的角色超出部分由 DialogBody（flex-1 + overflow-y-auto）内部滚动。 */}
      <DialogContent className="sm:max-w-[720px] h-[min(680px,calc(100dvh-4rem))]">
        <DialogHeader>
          <DialogTitle>切换角色</DialogTitle>
        </DialogHeader>
        <DialogBody className="px-6 overflow-y-auto min-h-0">
          <div className="flex flex-col gap-4">
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
                  <AgentAvatar roleName={slot.baseRoleName || slot.roleName || "通用助手"} size={20} className="shrink-0" />
                  <MetaText as="span" tone="secondary">{slot.baseRoleName || slot.roleName || "通用助手"}</MetaText>
                </div>
                <div className="w-px h-4 bg-gray-200 shrink-0" aria-hidden />
                <div className="flex items-center gap-1.5 text-sm">
                  <MetaMedium tone="primary" as="span">角色名称</MetaMedium>
                  <MetaText as="span" tone="secondary">{displayRoleName || "通用助手"}</MetaText>
                </div>
              </div>
            </div>
            {/* 切换为卡片 */}
            <div className="rounded-[8px] border border-gray-200 bg-[var(--bg-grey-normal)] overflow-hidden">
              <div className="flex items-center h-10 px-4 border-b border-gray-200 bg-[var(--bg-grey-normal)] text-xs text-[var(--text-muted)]">
                切换为
              </div>
              <div className="px-4 py-4">
                <div className="flex flex-col gap-4">
                  <div>
                    <MetaMedium tone="primary">角色类型</MetaMedium>
                    <RoleTypeRadioGroup
                      className="mt-1"
                      value={selectValue}
                      onValueChange={(value) => {
                        let chosen: Role | "__general__" | null = null;
                        if (value === "__general__") chosen = "__general__";
                        else if (value) chosen = visibleRoles.find((r) => r.id === value) ?? null;
                        if (!chosen || chosen === "__general__") {
                          setSwitchSlotSkillNames((prev) => { const n = { ...prev }; delete n[slot.slotId]; return n; });
                          setSwitchSlotSkillPopoverSlot((id) => (id === slot.slotId ? null : id));
                        } else {
                          setSwitchSlotSkillNames((prev) => ({ ...prev, [slot.slotId]: new Set(chosen.skills.map((s) => s.name)) }));
                        }
                        setSwitchRoleTargets((prev) => ({ ...prev, [slot.slotId]: chosen }));
                      }}
                      idPrefix={`switch-slot-${slot.slotId}`}
                      options={options.map((opt) => ({ value: opt.value, label: opt.label }))}
                    />
                  </div>
                  <RoleNameInput key={selectValue} defaultName={displayIntro.name} />
                  <div className="space-y-2.5" ref={switchSlotSkillPanelRef}>
                    <MetaMedium tone="primary">角色技能</MetaMedium>
                    <MetaText as="p" tone="secondary" className="leading-relaxed">{displayIntro.skills}</MetaText>
                  </div>
                  <div className="space-y-2.5">
                    <MetaMedium tone="primary">角色风格</MetaMedium>
                    <MetaText as="p" tone="secondary" className="leading-relaxed">{displayIntro.soul}</MetaText>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </DialogBody>
        <DialogFooter>
          <Button
            type="button"
            variant="tenant-outline"
            onClick={() => {
              setSwitchSlotSkillPopoverSlot(null);
              onClose();
            }}
          >
            取消
          </Button>
          <Button
            type="button"
            variant="tenant-dialog-confirm"
            disabled={!changed || switchSlotSkillPopoverSlot === slot.slotId}
            onClick={() => {
              const sid = slot.slotId;
              const introConfirm = getRoleIntro(target);
              const skillCount =
                target && target !== "__general__"
                  ? (switchSlotSkillNames[sid]?.size ?? target.skills.length)
                  : 5;
              setSwitchSlotSkillPopoverSlot(null);
              onClose();
              setConfirmingSlotId(sid);
              setConfirmedSlotIds((prev) => {
                const next = new Set(prev);
                next.delete(sid);
                return next;
              });
              roleConfigProgress.start({
                roleName: introConfirm.name,
                roleSoul: introConfirm.soul,
                skillCount,
                mode: "switch",
                agentId,
                agentName,
                items: [
                  {
                    slotId: sid,
                    fromName: slot.roleName || "通用助手",
                    toName: introConfirm.name,
                    // 类型信息（进度明细项「类型 + 名称」两行）：
                    //  · 原类型锚定 baseRoleName（改名后类型不变），回退到 roleName；
                    //  · 目标类型为所选 Role 名 / 通用助手。
                    fromType: slot.baseRoleName || slot.roleName || "通用助手",
                    toType: target === "__general__" ? "通用助手" : (target?.name ?? introConfirm.name),
                    skillCount,
                  },
                ],
                apply: () => {
                  setConfirmedSlotIds((prev) => new Set(prev).add(sid));
                  setConfirmingSlotId((cur) => (cur === sid ? null : cur));
                  if (agentId) clearRoleSwitching(agentId);
                  announcedSwitchSlotsRef.current.add(sid);
                  toast.success(
                    `「${agentName ?? ""}」角色切换成功：${slot.roleName || "通用助手"} → ${introConfirm.name}`
                  );
                },
              });
            }}
          >
            确认
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default SwitchRoleDialog;
