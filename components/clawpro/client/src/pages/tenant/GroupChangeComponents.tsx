/**
 * 用户端 - 组织变更相关组件
 *
 * 包含：
 *   - GroupChangeBadge：实例卡片上的组织变更标记
 *   - MigrateDialog：迁移到新组织弹窗
 *   - TransferDialog：移交给同组织其他用户弹窗
 *   - TransferReceiveOverlay：接收方蒙版确认
 *
 * 说明：此前的「组织变更通知」主动提醒弹窗（登录时自动弹出）已下线，
 * 组织变更相关信息统一通过顶部导航「消息通知」以纯文字通知呈现。
 */
import React, { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { StatusTag } from "@/components/ui/status-tag";
import {
  BodyText,
  CodeText,
  MetaText,
} from "@/components/ui/Typography";
import { Info } from "lucide-react";

// ─── Types ─────────────────────────────────────────────────────────────────

export type InstanceHandlingStatus =
  | "migrated"           // 已迁移至新组织
  | "transferred"        // 已移交给 xxx
  | "archived"           // 管理员已归档关机
  | "deleted"            // 管理员已删除
  | "pending"            // 待处理
  | "pendingConfirm"     // 待对方确认
  | "transferring"       // 移交中
  | "rejected"           // 对方已拒绝
  | "expired"            // 对方超时未处理
  | "migrating";         // 迁移至新组织中

export interface TransferRequest {
  instanceId: string;
  instanceName: string;
  fromUser: string;
  group: string;
}

// ─── 1. 组织变更标记（实例卡片用） ──────────────────────────────────────────

interface GroupChangeBadgeProps {
  status: InstanceHandlingStatus;
  originalGroup: string;
  transferTarget?: string;
  /**
   * 版本号插槽：由 AgentCard 复用普通卡完全相同的版本渲染逻辑（含可更新蓝胶囊）后传入，
   * 在「组织：」前展示，保证与其他卡片版本信息一致。
   */
  versionSlot?: React.ReactNode;
}

export function GroupChangeBadge({ status, originalGroup, transferTarget, versionSlot }: GroupChangeBadgeProps) {
  const mismatchBadge = (
    <StatusTag mode="soft" variant="gray">您已不在该组织</StatusTag>
  );

  const getStatusTag = () => {
    switch (status) {
      case "pendingConfirm":
        return <StatusTag mode="soft" variant="blue">待对方确认</StatusTag>;
      case "transferring":
        return <StatusTag mode="soft" variant="blue">移交中</StatusTag>;
      case "rejected":
        return <StatusTag mode="soft" variant="red">对方已拒绝</StatusTag>;
      case "expired":
        return <StatusTag mode="soft" variant="red">对方超时未处理</StatusTag>;
      case "migrating":
        return <StatusTag mode="soft" variant="blue">迁移至新组织中</StatusTag>;
      default:
        return null;
    }
  };

  return (
    // 版本与组织各占一行；用紧凑行距（gap 2px + leading-[18px]）保持卡片整体高度基本不变
    <div className="flex flex-col gap-0.5 text-xs leading-[18px] text-[var(--text-muted)]">
      {versionSlot && (
        // 版本号容器：统一为普通卡同款 secondary 文字色，保证视觉一致
        <div className="flex items-center" style={{ color: "var(--text-secondary)" }}>
          {versionSlot}
        </div>
      )}
      <div className="flex items-center flex-wrap gap-1">
        <span>组织：<span className="text-[var(--text-title)]">{originalGroup}</span></span>
        {mismatchBadge}
        {getStatusTag()}
      </div>
    </div>
  );
}

// ─── 2. 迁移到新组织弹窗 ───────────────────────────────────────────────────

interface MigrateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  instanceName: string;
  availableGroups: Array<{ id: string; name: string }>;
  onConfirm: (targetGroupId: string) => void;
  onViewDiff?: () => void;
}

export function MigrateDialog({
  open,
  onOpenChange,
  instanceName,
  availableGroups,
  onConfirm,
  onViewDiff,
}: MigrateDialogProps) {
  const [targetId, setTargetId] = useState(availableGroups[0]?.id ?? "");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md" onOpenAutoFocus={(e) => e.preventDefault()}>
        <DialogHeader>
          <DialogTitle>迁移到新组织</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <BodyText as="p" tone="secondary">
            Agent「<BodyText as="span" tone="primary" className="font-medium">{instanceName}</BodyText>」将跟随您从原组织迁移至新组织，迁移后 Agent 将自动开机。
          </BodyText>
          <div className="space-y-2">
            <MetaText as="label" className="block">目标组织</MetaText>
            <Select value={targetId} onValueChange={setTargetId}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="请选择" />
              </SelectTrigger>
              <SelectContent>
                {availableGroups.map((g) => (
                  <SelectItem key={g.id} value={g.id}>{g.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {onViewDiff && (
            <Button variant="link" className="h-auto px-0 py-0 text-xs" onClick={onViewDiff}>查看配置差异</Button>
          )}
          {/* 迁移说明块——样式参考管控端 OpenClawMonitor 的「迁移说明」 */}
          <div className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] px-3 py-2.5">
            <div className="text-xs font-medium text-[var(--text-secondary)] mb-1">迁移说明：</div>
            <ul className="space-y-1 text-xs leading-relaxed text-[var(--text-secondary)] list-disc pl-4">
              <li>Agent 迁移至您的新组织后，实例的平台策略会立即应用新组织配置（包括您可创建的 Agent 数量上限、您的单用户 Tokens 上限、功能权限等），其他已配置项保留不变。</li>
              <li>您后续修改 Agent 配置时只能改为新组织的配置。</li>
              <li>迁移成功后，Agent 将自动开机，恢复运行中状态。</li>
            </ul>
          </div>
        </div>
        <DialogFooter>
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button
            variant="tenant-primary"
            onClick={() => { onConfirm(targetId); onOpenChange(false); }}
          >
            确认迁移
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── 3. 移交给同组织其他用户弹窗 ─────────────────────────────────────────

interface TransferDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  instanceName: string;
  originalGroupName: string;
  availableUsers: Array<{ userId: string; displayName?: string }>;
  onConfirm: (targetUserId: string) => void;
}

export function TransferDialog({
  open,
  onOpenChange,
  instanceName,
  originalGroupName,
  onConfirm,
}: TransferDialogProps) {
  const [targetUserId, setTargetUserId] = useState("");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md" onOpenAutoFocus={(e) => e.preventDefault()}>
        <DialogHeader>
          <DialogTitle>移交给同组织其他用户</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <BodyText as="p" tone="secondary">
            Agent「<BodyText as="span" tone="primary" className="font-medium">{instanceName}</BodyText>」将移交给原组织「<BodyText as="span" tone="primary" className="font-medium">{originalGroupName}</BodyText>」内的其他用户，移交后 Agent 将转移到对方的 Agent 列表。
          </BodyText>
          <div className="space-y-2">
            <div className="flex items-center gap-1">
              <MetaText as="label" className="block">接手用户 ID</MetaText>
              <TooltipProvider delayDuration={150}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      type="button"
                      aria-label="接手用户 ID 说明"
                      className="inline-flex h-4 w-4 items-center justify-center text-[var(--text-muted)] hover:text-[var(--text-secondary)] focus:outline-none"
                    >
                      <Info className="h-3.5 w-3.5" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent side="top" align="start" className="max-w-[240px]">
                    用户 ID 指的是接收方在页面右上角展示的当前账号名
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
            <Input
              value={targetUserId}
              onChange={(e) => setTargetUserId(e.target.value)}
              placeholder="请输入对方的用户 ID"
            />
          </div>
          {/* 移交说明块——样式参考「迁移到新组织」弹窗的迁移说明 */}
          <div className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] px-3 py-2.5">
            <div className="text-xs font-medium text-[var(--text-secondary)] mb-1">移交说明：</div>
            <ul className="space-y-1 text-xs leading-relaxed text-[var(--text-secondary)] list-disc pl-4">
              <li>发起移交后，对方将在其 Agent 列表选择确认接收或拒绝，对方确认接收后才会正式转移到对方的 Agent 列表。</li>
              <li>在对方还未确认接收或拒绝之前，您可以随时取消此次移交。</li>
              <li>Agent 移交给对方后，已配置项保留不变，如 Agent 内有敏感信息请您先提前清除。</li>
            </ul>
          </div>
        </div>
        <DialogFooter>
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button
            variant="tenant-primary"
            disabled={!targetUserId.trim()}
            onClick={() => { onConfirm(targetUserId.trim()); onOpenChange(false); }}
          >
            发起移交
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── 4. 接收方蒙版确认 ──────────────────────────────────────────────────────

interface TransferReceiveOverlayProps {
  fromUser: string;
  instanceName: string;
  onAccept: () => void;
  onReject: () => void;
}

export function TransferReceiveOverlay({
  fromUser,
  instanceName,
  onAccept,
  onReject,
}: TransferReceiveOverlayProps) {
  return (
    <div className="absolute inset-0 bg-white/[0.92] z-10 flex flex-col items-center justify-center p-4 rounded-[var(--radius-card)]">
      <BodyText as="p" tone="primary" className="font-medium mb-1">实例移交确认</BodyText>
      <MetaText as="p" className="text-center mb-3">
        {fromUser} 将 Agent「{instanceName}」移交给您，<br />是否确认接收？
      </MetaText>
      <div className="flex items-center gap-2">
        <Button variant="tenant-outline" size="sm" onClick={onReject}>拒绝</Button>
        <Button
          variant="tenant-primary"
          size="sm"
          onClick={onAccept}
        >
          确认接收
        </Button>
        {/* 信息 icon——hover 展示移交接收的说明文案 */}
        <TooltipProvider delayDuration={150}>
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                aria-label="移交接收说明"
                className="inline-flex h-5 w-5 items-center justify-center rounded-full text-[var(--text-muted)] hover:text-[var(--text-secondary)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--tenant-ring,rgba(0,0,0,0.12))]"
              >
                <Info className="h-4 w-4" />
              </button>
            </TooltipTrigger>
            <TooltipContent side="top" align="center" className="max-w-[280px]">
              <ul className="list-disc pl-4 space-y-1 text-xs leading-relaxed">
                <li>Agent 移交给您后，此前已配置项保留不变，您可后续再进行修改。</li>
                <li>Agent 移交成功后，Agent 将自动开机，恢复运行中状态。</li>
              </ul>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </div>
  );
}

// ─── Mock 数据 ──────────────────────────────────────────────────────────────

export const MOCK_TRANSFER_REQUESTS: TransferRequest[] = [
  { instanceId: "claw-transfer-001", instanceName: "文档生成器", fromUser: "alice@a.com", group: "前端组" },
];
