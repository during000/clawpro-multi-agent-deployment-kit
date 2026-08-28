/**
 * 存量 Agent 实例处理弹窗（v3.0）
 *
 * 统一承载以下触发场景下的弹窗形态：
 *   editUser  —— 编辑用户组织 / 从组织移除用户（单/多用户，按原组织选择处理方式）
 *               · 组织间移动（弹窗①）
 *               · 移出到未分配组织（弹窗②，migrate → 回退为未分配组织配置）
 *               · 未分配组织加入新组织（弹窗③，移交禁用、自行处理仅迁移）
 *   oneidSync —— OneID 同步处理（多用户多状态混合，弹窗⑦）
 *               · 末尾可附「修改上级组织」自动迁移块（与弹窗⑥同构）
 *   editParent—— 修改上级组织（弹窗⑥，按新组织路径聚合，随组织自动迁移，无需逐项选择）
 *
 * 处理方式（按原组织选择）：
 *   - 随用户迁移到新组织 / 回退为未分配组织配置（migrate）
 *   - 移交给同组织其他用户（transfer）
 *   - 允许用户自行处理（userSelf）
 *   - 保留并关机（archive）
 *
 * 支持 per-group 覆盖：禁用项与原因、迁移目标、移交用户、未分配组织回退、默认选项。
 *
 * 视觉规范：遵循 ClawPro 可移植设计规范（管控端）：
 *   - 颜色走 --cp-* token，面板/控件圆角统一 --radius-lg(4px)，禁止硬编码 hex / 渐变 / 非 4px 圆角；
 *   - 弹窗采用冻结布局：DialogHeader / DialogFooter 固定，中间内容用 DialogBody 滚动（spec §12.4）；
 *   - 处理方式选项卡用 RadioGroup + RadioCard，下拉用 Select，勾选用 Checkbox（禁止原生控件）；
 *   - 主操作按钮 Button variant="dialog-confirm"，次级按钮 variant="claw-outline"。
 */
import React, { useMemo, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import { RadioGroup } from "@/components/ui/radio-group";
import { RadioCard } from "@/components/ui/radio-card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import { ChevronDown, ChevronRight, User, Eye } from "lucide-react";
import { toast } from "sonner";

// ─── Types ─────────────────────────────────────────────────────────────────

export type HandlingMode =
  | "migrate" // 随用户迁移到新组织 / 回退为未分配组织配置
  | "transfer" // 移交给同组织其他用户
  | "userSelf" // 允许用户自行处理
  | "archive"; // 保留并关机

export type TriggerScenario =
  | "editUser" // 编辑用户组织 / 移除用户
  | "oneidSync" // OneID 同步处理
  | "editParent"; // 修改上级组织

export interface AgentInstance {
  id: string;
  name: string;
}

export interface MigrateTarget {
  id: string;
  name: string;
}

export interface TransferTarget {
  userId: string;
  displayName?: string;
}

export interface UserSelfOptions {
  allowTransfer: boolean;
  allowMigrate: boolean;
}

export interface AffectedGroup {
  groupId: string;
  groupName: string;
  instances: AgentInstance[];
  /** 该原组织迁移可选目标（覆盖全局 migrateTargets） */
  migrateTargets?: MigrateTarget[];
  /** 该原组织移交可选用户（覆盖全局 transferTargets[groupId]） */
  transferTargets?: TransferTarget[];
  /** 该原组织各处理方式禁用原因 */
  disabledModes?: Partial<Record<HandlingMode, string>>;
  /** 迁移目标为「未分配组织」：migrate 文案改为「回退为未分配组织配置」，无目标下拉 */
  migrateToUnassigned?: boolean;
  /** 默认选中的处理方式 */
  defaultMode?: HandlingMode;
  /** 允许用户自行处理的子选项可用性 */
  userSelfOptions?: UserSelfOptions;
}

export interface AffectedUser {
  userId: string;
  originalGroups: string[]; // 原组织名列表
  newGroups: string[]; // 新组织名列表（为空表示未分配组织/已离职）
  affectedGroups: AffectedGroup[];
}

/** 「修改上级组织」自动迁移：按新组织路径聚合 */
export interface MigrationPathGroup {
  /** 目标路径名，如 "产品二部 / 研发组" */
  targetPath: string;
  fromGroupId?: string;
  toGroupId?: string;
  rows: Array<{ userId: string; instanceId: string; instanceName: string }>;
}

export interface ParentMigration {
  /** 变更说明 */
  note: string;
  groups: MigrationPathGroup[];
}

export interface GroupDecision {
  userId: string;
  groupId: string;
  mode: HandlingMode;
  migrateTargetId?: string;
  transferTargetUserId?: string;
  userSelfAllowTransfer?: boolean;
  userSelfAllowMigrate?: boolean;
}

export interface AgentInstanceHandlingDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  scenario: TriggerScenario;
  affectedUsers?: AffectedUser[];
  /** 全局迁移可选目标（可被各原组织覆盖） */
  migrateTargets?: MigrateTarget[];
  /** 全局移交可选用户（按原组织 groupId 提供，可被各原组织覆盖） */
  transferTargets?: Record<string, TransferTarget[]>;
  /** 全局禁用项（可被各原组织覆盖） */
  disabledModes?: Partial<Record<HandlingMode, string>>;
  /** 全局 userSelf 子选项可用性 */
  userSelfOptions?: UserSelfOptions;
  /** editParent 场景主体 / oneidSync 末尾追加的上级组织自动迁移块 */
  parentMigration?: ParentMigration;
  onConfirm?: (decisions: GroupDecision[]) => void;
  /** 查看配置对比（instances 为该原组织命中的存量实例，用于逐实例卡片） */
  onViewDiff?: (fromGroupId: string, toGroupId: string, instances?: AgentInstance[]) => void;
}

// ─── 文案 ───────────────────────────────────────────────────────────────────

const SCENARIO_TEXT: Record<TriggerScenario, string> = {
  editUser: "由于用户的组织发生变更，用户在原组织创建的存量 Agent 实例需要处理，请您选择处理方式。",
  oneidSync: "由于用户的组织发生变更，用户在原组织创建的存量 Agent 实例需要处理，请您选择处理方式。",
  editParent: "本次上级组织变更涉及以下存量 Agent 实例，将随组织自动迁移到新路径。",
};

const MIGRATE_DESC_LINES = [
  "Agent 实例的平台策略会应用新组织配置，其他已配置项保留不变，后续修改只能改为新组织配置。",
  "如有需要，管理员可后续到 Agent 列表页查看实例与新组织的配置对比，调整不符合的配置项。",
];
const MIGRATE_UNASSIGNED_DESC_LINES = [
  "Agent 实例的平台策略会应用未分配组织的默认配置，其他已配置项保留不变，后续修改只能改为未分配组织的默认配置。",
  "如有需要，管理员可后续到 Agent 列表页查看实例与未分配组织配置的对比，调整不符合的配置项。",
];

const MODE_TITLE: Record<HandlingMode, string> = {
  migrate: "随用户迁移到新组织",
  transfer: "移交给同组织其他用户",
  userSelf: "允许用户自行处理",
  archive: "保留并关机",
};

const ALL_MODES: HandlingMode[] = ["migrate", "transfer", "userSelf", "archive"];

// ─── 组件 ───────────────────────────────────────────────────────────────────

interface ResolvedGroupConfig {
  migrateTargets: MigrateTarget[];
  transferTargets: TransferTarget[];
  disabledModes: Partial<Record<HandlingMode, string>>;
  userSelfOptions: UserSelfOptions;
  migrateToUnassigned: boolean;
  defaultMode: HandlingMode;
}

interface DecisionState {
  mode: HandlingMode;
  migrateTargetId: string;
  transferTargetUserId: string;
  userSelfAllowTransfer: boolean;
  userSelfAllowMigrate: boolean;
}

export default function AgentInstanceHandlingDialog({
  open,
  onOpenChange,
  scenario,
  affectedUsers = [],
  migrateTargets = [],
  transferTargets = {},
  disabledModes = {},
  userSelfOptions = { allowTransfer: true, allowMigrate: true },
  parentMigration,
  onConfirm,
  onViewDiff,
}: AgentInstanceHandlingDialogProps) {
  const isEditParent = scenario === "editParent";

  // 各原组织（userId::groupId）解析后的可用配置
  const groupConfigMap = useMemo(() => {
    const map = new Map<string, ResolvedGroupConfig>();
    affectedUsers.forEach((user) => {
      user.affectedGroups.forEach((ag) => {
        const resolvedDisabled = ag.disabledModes ?? disabledModes;
        const resolvedMigrateTargets = ag.migrateTargets ?? migrateTargets;
        const resolvedTransferTargets =
          ag.transferTargets ?? transferTargets[ag.groupId] ?? [];
        const resolvedUserSelf = ag.userSelfOptions ?? userSelfOptions;
        const defaultMode =
          ag.defaultMode ??
          ALL_MODES.find((m) => !resolvedDisabled[m]) ??
          "archive";
        map.set(`${user.userId}::${ag.groupId}`, {
          migrateTargets: resolvedMigrateTargets,
          transferTargets: resolvedTransferTargets,
          disabledModes: resolvedDisabled,
          userSelfOptions: resolvedUserSelf,
          migrateToUnassigned: !!ag.migrateToUnassigned,
          defaultMode,
        });
      });
    });
    return map;
  }, [affectedUsers, migrateTargets, transferTargets, disabledModes, userSelfOptions]);

  const [decisions, setDecisions] = useState<Map<string, DecisionState>>(new Map());
  const [collapsedUsers, setCollapsedUsers] = useState<Set<string>>(new Set());

  const getKey = (userId: string, groupId: string) => `${userId}::${groupId}`;

  const getDecision = (userId: string, groupId: string): DecisionState => {
    const key = getKey(userId, groupId);
    const existing = decisions.get(key);
    if (existing) return existing;
    const cfg = groupConfigMap.get(key);
    return {
      mode: cfg?.defaultMode ?? "migrate",
      migrateTargetId: cfg?.migrateTargets[0]?.id ?? "",
      // 默认选中第一个可选接手用户，不留空白
      transferTargetUserId: cfg?.transferTargets[0]?.userId ?? "",
      userSelfAllowTransfer: cfg?.userSelfOptions.allowTransfer ?? true,
      userSelfAllowMigrate: cfg?.userSelfOptions.allowMigrate ?? true,
    };
  };

  const setDecisionField = (
    userId: string,
    groupId: string,
    field: keyof DecisionState,
    value: unknown,
  ) => {
    const key = getKey(userId, groupId);
    setDecisions((prev) => {
      const next = new Map(prev);
      const current = next.get(key) ?? getDecision(userId, groupId);
      next.set(key, { ...current, [field]: value });
      return next;
    });
  };

  const toggleUser = (userId: string) => {
    setCollapsedUsers((prev) => {
      const next = new Set(prev);
      if (next.has(userId)) next.delete(userId);
      else next.add(userId);
      return next;
    });
  };

  const handleConfirm = () => {
    if (isEditParent) {
      onConfirm?.([]);
      return;
    }
    const result: GroupDecision[] = [];
    affectedUsers.forEach((user) => {
      user.affectedGroups.forEach((ag) => {
        const d = getDecision(user.userId, ag.groupId);
        result.push({
          userId: user.userId,
          groupId: ag.groupId,
          mode: d.mode,
          migrateTargetId: d.mode === "migrate" ? d.migrateTargetId : undefined,
          transferTargetUserId: d.mode === "transfer" ? d.transferTargetUserId : undefined,
          userSelfAllowTransfer: d.mode === "userSelf" ? d.userSelfAllowTransfer : undefined,
          userSelfAllowMigrate: d.mode === "userSelf" ? d.userSelfAllowMigrate : undefined,
        });
      });
    });
    onConfirm?.(result);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-2xl max-h-[85vh] flex flex-col"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>存量 Agent 实例处理</DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6">
          <div className="py-2 space-y-4">
            {isEditParent ? (
              <ParentMigrationBlock
                migration={parentMigration}
                onViewDiff={onViewDiff}
              />
            ) : (
              <>
                {/* 场景文案 */}
                <Alert variant="info">
                  <AlertInfoIcon />
                  <AlertDescription>{SCENARIO_TEXT[scenario]}</AlertDescription>
                </Alert>

                {affectedUsers.map((user) => (
                  <UserSection
                    key={user.userId}
                    user={user}
                    collapsed={collapsedUsers.has(user.userId)}
                    onToggle={() => toggleUser(user.userId)}
                    groupConfigMap={groupConfigMap}
                    getDecision={getDecision}
                    setDecisionField={setDecisionField}
                    onViewDiff={onViewDiff}
                  />
                ))}

                {/* oneidSync 末尾追加：修改上级组织自动迁移块 */}
                {parentMigration && parentMigration.groups.length > 0 && (
                  <div className="pt-2 space-y-4 border-t border-dashed border-[var(--cp-border)]">
                    <ParentMigrationBlock
                      migration={parentMigration}
                      onViewDiff={onViewDiff}
                    />
                  </div>
                )}
              </>
            )}
          </div>
        </DialogBody>

        <DialogFooter className="shrink-0">
          <Button variant="claw-outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button variant="dialog-confirm" onClick={handleConfirm}>
            确认处理
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── 修改上级组织：自动迁移块（按新路径聚合） ────────────────────────────────

function ParentMigrationBlock({
  migration,
  onViewDiff,
}: {
  migration?: ParentMigration;
  onViewDiff?: (fromGroupId: string, toGroupId: string) => void;
}) {
  if (!migration) return null;
  return (
    <div className="space-y-4">
      <Alert variant="info">
        <AlertInfoIcon />
        <AlertDescription>
          <div className="space-y-1.5">
            <div>
              1. 由于上级组织发生变更，原组织及其所有子组织的存量 Agent 实例将随组织自动迁移到新路径。
            </div>
            <div>
              2. Agent 实例的平台策略会应用新组织配置，其他已配置项保留不变，后续修改只能改为新组织配置。
            </div>
            <div>
              3. 如有需要，管理员可后续到 Agent 列表页查看实例与新组织的配置对比，调整不符合的配置项。
            </div>
          </div>
        </AlertDescription>
      </Alert>

      {migration.groups.map((g, idx) => (
        <div
          key={idx}
          className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] overflow-hidden"
        >
          <div className="flex items-center justify-between px-3 py-2 bg-[var(--bg-grey-normal)] border-b border-[var(--cp-border)]">
            <span className="text-xs font-medium text-[var(--cp-text-muted)]">迁移至：{g.targetPath}</span>
            <div className="flex items-center gap-3">
              {onViewDiff && (
                <button
                  type="button"
                  className="text-[11px] text-[var(--cp-text-brand)] hover:underline inline-flex items-center gap-1"
                  onClick={() =>
                    onViewDiff(
                      g.fromGroupId ?? "",
                      g.toGroupId ?? "",
                      g.rows.map((r) => ({ id: r.instanceId, name: r.instanceName })),
                    )
                  }
                >
                  <Eye className="w-3 h-3" />
                  查看配置对比
                </button>
              )}
              <span className="text-xs text-[var(--cp-text-weak)]">{g.rows.length} 个实例</span>
            </div>
          </div>
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-[var(--bg-grey-normal)] border-b border-[var(--cp-border)]">
                <th className="text-left px-3 py-2 font-medium text-[var(--cp-text-weak)] uppercase tracking-wide w-[30%]">
                  用户 ID
                </th>
                <th className="text-left px-3 py-2 font-medium text-[var(--cp-text-weak)] uppercase tracking-wide">
                  Agent 实例名称 / ID
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--cp-border)]">
              {g.rows.map((r) => (
                <tr key={r.instanceId}>
                  <td className="px-3 py-2 text-[var(--cp-text-secondary)]">{r.userId}</td>
                  <td className="px-3 py-2 text-[var(--cp-text-secondary)]">
                    {r.instanceName} <span className="text-[var(--cp-text-weak)]">({r.instanceId})</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ))}
    </div>
  );
}

// ─── 用户分区 ───────────────────────────────────────────────────────────────

interface UserSectionProps {
  user: AffectedUser;
  collapsed: boolean;
  onToggle: () => void;
  groupConfigMap: Map<string, ResolvedGroupConfig>;
  getDecision: (userId: string, groupId: string) => DecisionState;
  setDecisionField: (
    userId: string,
    groupId: string,
    field: keyof DecisionState,
    value: unknown,
  ) => void;
  onViewDiff?: (fromGroupId: string, toGroupId: string) => void;
}

function UserSection({
  user,
  collapsed,
  onToggle,
  groupConfigMap,
  getDecision,
  setDecisionField,
  onViewDiff,
}: UserSectionProps) {
  const removed = user.newGroups.length === 0;
  return (
    <div>
      <div
        className="flex items-center gap-2 cursor-pointer select-none"
        onClick={onToggle}
      >
        {collapsed ? (
          <ChevronRight className="w-3.5 h-3.5 text-[var(--cp-text-weak)]" />
        ) : (
          <ChevronDown className="w-3.5 h-3.5 text-[var(--cp-text-weak)]" />
        )}
        <User className="w-3.5 h-3.5 text-[var(--cp-text-muted)]" />
        <span className="text-sm font-semibold text-[var(--cp-text-title)]">{user.userId}</span>
      </div>

      {!collapsed && (
        <>
          <div className="mt-2 ml-5 text-xs text-[var(--cp-text-muted)] bg-[var(--cp-surface)] border border-[var(--cp-border)] rounded-[var(--radius-lg)] px-3 py-2 leading-relaxed">
            <div>
              原组织：
              <span className="text-[var(--cp-text-title)] font-medium">
                {user.originalGroups.join("、") || "未分配组织"}
              </span>
            </div>
            <div>
              现组织：
              <span className={`font-medium ${removed ? "text-[var(--cp-text-danger)]" : "text-[var(--cp-text-title)]"}`}>
                {user.newGroups.join("、") || "未分配组织"}
              </span>
            </div>
          </div>

          <div className="mt-3 ml-5 space-y-3">
            {user.affectedGroups.map((ag) => {
              const cfg = groupConfigMap.get(`${user.userId}::${ag.groupId}`);
              return (
                <GroupCard
                  key={ag.groupId}
                  userId={user.userId}
                  group={ag}
                  cfg={cfg}
                  decision={getDecision(user.userId, ag.groupId)}
                  setField={(field, value) =>
                    setDecisionField(user.userId, ag.groupId, field, value)
                  }
                  onViewDiff={onViewDiff}
                />
              );
            })}
          </div>
        </>
      )}
    </div>
  );
}

// ─── 原组织卡片 ─────────────────────────────────────────────────────────────

interface GroupCardProps {
  userId: string;
  group: AffectedGroup;
  cfg?: ResolvedGroupConfig;
  decision: DecisionState;
  setField: (field: keyof DecisionState, value: unknown) => void;
  onViewDiff?: (fromGroupId: string, toGroupId: string) => void;
}

function GroupCard({ userId, group, cfg, decision, setField, onViewDiff }: GroupCardProps) {
  const disabledModes = cfg?.disabledModes ?? {};
  const migrateToUnassigned = cfg?.migrateToUnassigned ?? false;

  return (
    <div className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-[var(--cp-surface)] overflow-hidden">
      <div className="flex items-center justify-between px-3 py-2 bg-[var(--bg-grey-normal)] border-b border-[var(--cp-border)]">
        <span className="text-xs font-medium text-[var(--cp-text-muted)]">原组织：{group.groupName}</span>
        <span className="text-xs text-[var(--cp-text-weak)]">{group.instances.length} 个实例</span>
      </div>

      <div className="p-3 space-y-3">
        <div>
          <div className="text-[10px] text-[var(--cp-text-weak)] uppercase tracking-wide mb-1">
            Agent 实例名称 / ID
          </div>
          {group.instances.map((inst) => (
            <div key={inst.id} className="text-xs text-[var(--cp-text-secondary)] py-0.5">
              {inst.name} <span className="text-[var(--cp-text-weak)]">({inst.id})</span>
            </div>
          ))}
        </div>

        <div className="pt-2 border-t border-dashed border-[var(--cp-border)] space-y-2">
          <div className="text-[10px] text-[var(--cp-text-weak)] uppercase tracking-wide">处理方式</div>

          <RadioGroup
            className="gap-2"
            value={decision.mode}
            onValueChange={(v) => {
              if (!disabledModes[v as HandlingMode]) setField("mode", v as HandlingMode);
            }}
          >
            {ALL_MODES.map((mode) => (
              <ModeOption
                key={mode}
                mode={mode}
                userId={userId}
                groupId={group.groupId}
                selected={decision.mode === mode}
                disabled={!!disabledModes[mode]}
                disabledReason={disabledModes[mode]}
                migrateTargets={cfg?.migrateTargets ?? []}
                transferTargets={cfg?.transferTargets ?? []}
                userSelfOptions={cfg?.userSelfOptions ?? { allowTransfer: true, allowMigrate: true }}
                migrateToUnassigned={migrateToUnassigned}
                decision={decision}
                onFieldChange={setField}
                onViewDiff={onViewDiff}
                instances={group.instances}
              />
            ))}
          </RadioGroup>
        </div>
      </div>
    </div>
  );
}

// ─── 处理方式选项 ───────────────────────────────────────────────────────────

interface ModeOptionProps {
  mode: HandlingMode;
  userId: string;
  groupId: string;
  selected: boolean;
  disabled: boolean;
  disabledReason?: string;
  migrateTargets: MigrateTarget[];
  transferTargets: TransferTarget[];
  userSelfOptions: UserSelfOptions;
  migrateToUnassigned: boolean;
  decision: DecisionState;
  onFieldChange: (field: keyof DecisionState, value: unknown) => void;
  onViewDiff?: (fromGroupId: string, toGroupId: string, instances?: AgentInstance[]) => void;
  /** 该原组织命中的存量实例（用于配置对比逐实例卡片） */
  instances: AgentInstance[];
}

function ModeOption({
  mode,
  userId,
  groupId,
  selected,
  disabled,
  disabledReason,
  migrateTargets,
  transferTargets,
  userSelfOptions,
  migrateToUnassigned,
  decision,
  onFieldChange,
  onViewDiff,
  instances,
}: ModeOptionProps) {
  const title =
    mode === "migrate" && migrateToUnassigned ? "回退为未分配组织配置" : MODE_TITLE[mode];
  const id = `${userId}::${groupId}::${mode}`;

  return (
    <RadioCard id={id} value={mode} checked={selected} disabled={disabled} title={title} size="sm">
      {/* 禁用原因（红色） */}
      {disabled && disabledReason && (
        <p className="text-[11px] text-[var(--cp-text-danger)] mt-1 leading-relaxed">
          {disabledReason}
        </p>
      )}

      {/* 选中态展开详情 */}
      {!disabled && selected && (
        <div className="mt-2 space-y-2">
          {mode === "migrate" && (
            <>
              {!migrateToUnassigned && migrateTargets.length > 0 && (
                <Select
                  value={decision.migrateTargetId}
                  onValueChange={(v) => onFieldChange("migrateTargetId", v)}
                >
                  <SelectTrigger size="sm" className="w-full max-w-[280px] text-xs">
                    <SelectValue placeholder="选择新组织" />
                  </SelectTrigger>
                  <SelectContent>
                    {migrateTargets.map((t) => (
                      <SelectItem key={t.id} value={t.id} className="text-xs">
                        {t.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
              <div className="text-[11px] text-[var(--cp-text-muted)] leading-relaxed space-y-1">
                {(migrateToUnassigned ? MIGRATE_UNASSIGNED_DESC_LINES : MIGRATE_DESC_LINES).map(
                  (line, i) => (
                    <div key={i}>{line}</div>
                  ),
                )}
              </div>
              {onViewDiff && (
                <button
                  type="button"
                  className="text-[11px] text-[var(--cp-text-brand)] hover:underline inline-flex items-center gap-1"
                  onClick={() =>
                    onViewDiff(
                      groupId,
                      migrateToUnassigned ? "unassigned" : decision.migrateTargetId,
                      instances,
                    )
                  }
                >
                  <Eye className="w-3 h-3" />
                  查看配置对比
                </button>
              )}
            </>
          )}

          {mode === "transfer" && (
            <>
              <Select
                value={decision.transferTargetUserId || undefined}
                onValueChange={(v) => onFieldChange("transferTargetUserId", v)}
              >
                <SelectTrigger size="sm" className="w-full max-w-[280px] text-xs">
                  <SelectValue placeholder="选择接手用户" />
                </SelectTrigger>
                <SelectContent>
                  {transferTargets.map((t) => (
                    <SelectItem key={t.userId} value={t.userId} className="text-xs">
                      {t.displayName ?? t.userId}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-[11px] text-[var(--cp-text-muted)] leading-relaxed">
                Agent 实例将移交给原组织内的其他用户，实例全部配置保持不变。
              </p>
            </>
          )}

          {mode === "userSelf" && (
            <>
              <div className="flex flex-col gap-2">
                <button
                  type="button"
                  disabled={!userSelfOptions.allowMigrate}
                  className={cn(
                    "flex items-center gap-2 text-left",
                    !userSelfOptions.allowMigrate && "opacity-40 cursor-not-allowed",
                  )}
                  onClick={() => {
                    const currentChecked =
                      decision.userSelfAllowMigrate && userSelfOptions.allowMigrate;
                    const otherChecked =
                      decision.userSelfAllowTransfer && userSelfOptions.allowTransfer;
                    // 至少保留一个，禁止取消最后一个勾选
                    if (currentChecked && !otherChecked) {
                      toast.error("请至少选择一种允许用户自行处理的方式");
                      return;
                    }
                    onFieldChange("userSelfAllowMigrate", !currentChecked);
                  }}
                >
                  <Checkbox
                    className="pointer-events-none shrink-0"
                    checked={decision.userSelfAllowMigrate && userSelfOptions.allowMigrate}
                    disabled={!userSelfOptions.allowMigrate}
                  />
                  <span className="text-[11px] text-[var(--cp-text-secondary)]">
                    {migrateToUnassigned ? "允许回退为未分配组织配置" : "允许随用户迁移到新组织"}
                  </span>
                </button>
                <button
                  type="button"
                  disabled={!userSelfOptions.allowTransfer}
                  className={cn(
                    "flex items-center gap-2 text-left",
                    !userSelfOptions.allowTransfer && "opacity-40 cursor-not-allowed",
                  )}
                  onClick={() => {
                    const currentChecked =
                      decision.userSelfAllowTransfer && userSelfOptions.allowTransfer;
                    const otherChecked =
                      decision.userSelfAllowMigrate && userSelfOptions.allowMigrate;
                    // 至少保留一个，禁止取消最后一个勾选
                    if (currentChecked && !otherChecked) {
                      toast.error("请至少选择一种允许用户自行处理的方式");
                      return;
                    }
                    onFieldChange("userSelfAllowTransfer", !currentChecked);
                  }}
                >
                  <Checkbox
                    className="pointer-events-none shrink-0"
                    checked={decision.userSelfAllowTransfer && userSelfOptions.allowTransfer}
                    disabled={!userSelfOptions.allowTransfer}
                  />
                  <span className="text-[11px] text-[var(--cp-text-secondary)]">
                    允许移交给同组织其他用户
                  </span>
                </button>
              </div>
              <p className="text-[11px] text-[var(--cp-text-muted)] leading-relaxed">
                Agent 实例将自动关机，保留在原用户的原组织下，等待用户自行完成
                {migrateToUnassigned ? "回退或移交" : "迁移或移交"}后自动开机。
              </p>
            </>
          )}

          {mode === "archive" && (
            <p className="text-[11px] text-[var(--cp-text-muted)] leading-relaxed">
              Agent 实例将自动关机，保留在原用户的原组织下，等待管理员后续前往 Agent 列表页处理组织迁移或移交给其他用户。
            </p>
          )}
        </div>
      )}
    </RadioCard>
  );
}
