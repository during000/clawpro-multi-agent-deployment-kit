/**
 * StandardUpdateDialog - 企业规范/全局指令更新弹窗（独立可复用版）
 * 供「项目资产管理」更新已选的企业规范；版本按 patch 自动递增，应用范围锁定当前组织。
 */
import { useEffect, useState } from 'react';
import { toast } from 'sonner';
import { Lock } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogBody,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription, AlertInfoIcon } from '@/components/ui/alert';
import { MetaMedium, HelperText } from '@/components/ui/Typography';
import {
  ENTRY_CLIENTS,
  RULE_CLIENTS,
  type AgentConfigAsset,
  type TargetClient,
} from '../SkillLibrary/standardsStore';
import type { ScopeLockConfig } from '../SkillLibrary/types';

const CLIENT_LABELS: Record<TargetClient, string> = {
  claude_code: 'Claude Code',
  codebuddy: 'CodeBuddy',
  codex: 'Codex',
  workbuddy: 'WorkBuddy',
};

const bumpPatchVersion = (version: string) => {
  const parts = version.split('.').map((p) => Number(p));
  const [major = 1, minor = 0, patch = 0] = parts.map((p) => (Number.isFinite(p) ? p : 0));
  return `${major}.${minor}.${patch + 1}`;
};

interface StandardUpdateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  asset: AgentConfigAsset;
  lockedScope: ScopeLockConfig;
  onConfirm: (updatedAsset: AgentConfigAsset) => void;
}

export default function StandardUpdateDialog({
  open,
  onOpenChange,
  asset,
  lockedScope,
  onConfirm,
}: StandardUpdateDialogProps) {
  const [name, setName] = useState('');
  const [clients, setClients] = useState<Set<TargetClient>>(new Set());
  const [contentMd, setContentMd] = useState('');

  useEffect(() => {
    if (open && asset) {
      setName(asset.name);
      setClients(new Set(asset.targetClients));
      setContentMd(asset.contentMd);
    }
  }, [open, asset]);

  const allowedClients = asset.kind === 'entry' ? ENTRY_CLIENTS : RULE_CLIENTS;
  const nextVersion = bumpPatchVersion(asset.version);

  const toggleClient = (client: TargetClient) => {
    setClients((prev) => {
      const next = new Set(prev);
      if (next.has(client)) next.delete(client);
      else next.add(client);
      return next;
    });
  };

  const handleSubmit = () => {
    if (!name.trim()) {
      toast.error('请填写名称');
      return;
    }
    if (clients.size === 0) {
      toast.error('请至少选择一个适用客户端');
      return;
    }
    const updated: AgentConfigAsset = {
      ...asset,
      name: name.trim(),
      targetClients: Array.from(clients),
      contentMd: contentMd.trim() || asset.contentMd,
      version: nextVersion,
      checksum: `sha256:${Math.random().toString(16).slice(2, 8)}`,
      updatedAt: new Date(),
      lastTaskStatus: 'pending',
    };
    onConfirm(updated);
    toast.success(`企业规范「${updated.name}」已更新至 v${nextVersion}`);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-[560px] flex flex-col"
        style={{ maxHeight: 'min(90vh, 720px)' }}
        onPointerDownOutside={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>更新企业规范</DialogTitle>
        </DialogHeader>

        <DialogBody className="flex-1 px-6">
          <div className="space-y-4">
            <Alert variant="warning">
              <AlertInfoIcon />
              <AlertDescription>
                仅更新企业规范库中的版本。已下发至 Agent 实例的规范不会自动升级，需按同步模式重新下发。
              </AlertDescription>
            </Alert>

            {/* 名称 */}
            <div className="space-y-1.5">
              <MetaMedium as="label" tone="secondary" htmlFor="std-upd-name">
                名称<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
              </MetaMedium>
              <Input id="std-upd-name" value={name} onChange={(e) => setName(e.target.value)} />
            </div>

            {/* 版本（自动递增，只读） */}
            <div className="space-y-1.5">
              <MetaMedium as="label" tone="secondary" htmlFor="std-upd-version">版本号</MetaMedium>
              <Input id="std-upd-version" value={nextVersion} disabled />
              <HelperText>版本号在当前 v{asset.version} 基础上自动递增</HelperText>
            </div>

            {/* 适用客户端 */}
            <div className="space-y-1.5">
              <MetaMedium as="label" tone="secondary">适用客户端</MetaMedium>
              <div className="flex flex-wrap gap-2">
                {allowedClients.map((client) => {
                  const isSelected = clients.has(client);
                  return (
                    <button
                      key={client}
                      type="button"
                      onClick={() => toggleClient(client)}
                      className={`h-8 px-3 rounded-[4px] text-sm border transition-colors inline-flex items-center gap-1.5 ${
                        isSelected
                          ? 'border-[var(--cp-brand-blue)] bg-[var(--bg-brand-selected)] text-[var(--text-title)]'
                          : 'bg-white border-[var(--cp-border)] text-[var(--text-title)] hover:border-[var(--cp-brand-blue)]'
                      }`}
                    >
                      <Checkbox checked={isSelected} className="pointer-events-none" tabIndex={-1} />
                      {CLIENT_LABELS[client]}
                    </button>
                  );
                })}
              </div>
            </div>

            {/* 内容 */}
            <div className="space-y-1.5">
              <MetaMedium as="label" tone="secondary" htmlFor="std-upd-content">内容（Markdown）</MetaMedium>
              <Textarea
                id="std-upd-content"
                value={contentMd}
                onChange={(e) => setContentMd(e.target.value)}
                rows={6}
                className="font-mono text-xs"
              />
            </div>

            {/* 应用范围（锁定） */}
            <div className="space-y-1.5">
              <MetaMedium as="label" tone="secondary">应用范围</MetaMedium>
              <div className="flex items-center gap-1.5 h-9 px-3 rounded-[4px] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] text-[var(--text-secondary)]">
                <Lock className="w-3.5 h-3.5" />
                <MetaMedium tone="secondary">{lockedScope.lockedGroupName}（已锁定为该组织）</MetaMedium>
              </div>
            </div>
          </div>
        </DialogBody>

        <DialogFooter>
          <Button variant="claw-outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button variant="claw-primary" onClick={handleSubmit}>保存更新</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
