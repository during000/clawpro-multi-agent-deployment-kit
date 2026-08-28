/**
 * MCPUpdateDialog - 企业 MCP 更新弹窗（项目资产管理新增能力）
 * 工具库侧原本没有 MCP 更新功能；此弹窗用于在「项目资产管理」内更新 MCP 版本与配置，
 * 版本号必须高于当前版本，应用范围锁定当前组织。
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
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription, AlertInfoIcon } from '@/components/ui/alert';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { MetaMedium, MetaText, HelperText } from '@/components/ui/Typography';
import { isValidSemver, compareSemver } from '../SkillLibrary/downloadUtils';
import type { MCPService, ScopeLockConfig } from '../SkillLibrary/types';

interface MCPUpdateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mcp: MCPService;
  lockedScope: ScopeLockConfig;
  onConfirm: (updatedMCP: MCPService) => void;
}

export default function MCPUpdateDialog({
  open,
  onOpenChange,
  mcp,
  lockedScope,
  onConfirm,
}: MCPUpdateDialogProps) {
  const [displayName, setDisplayName] = useState('');
  const [description, setDescription] = useState('');
  const [version, setVersion] = useState('');
  const [configJson, setConfigJson] = useState('');
  const [usageDoc, setUsageDoc] = useState('');
  const [toolDoc, setToolDoc] = useState('');
  const [versionError, setVersionError] = useState('');

  useEffect(() => {
    if (open && mcp) {
      setDisplayName(mcp.displayName || mcp.name);
      setDescription(mcp.description);
      setVersion('');
      setConfigJson(mcp.configJson);
      setUsageDoc(mcp.usageDoc || '');
      setToolDoc(mcp.toolDoc || '');
      setVersionError('');
    }
  }, [open, mcp]);

  const validateVersion = (v: string): string => {
    const next = v.trim();
    if (!next) return '请填写新版本号';
    if (!isValidSemver(next)) return '版本号格式必须为 x.y.z';
    if (compareSemver(next, mcp.version) <= 0) return `新版本号需高于上个版本号 v${mcp.version}`;
    return '';
  };

  const handleVersionChange = (v: string) => {
    setVersion(v);
    setVersionError(validateVersion(v));
  };

  const handleSubmit = () => {
    const verErr = validateVersion(version);
    if (verErr) {
      setVersionError(verErr);
      toast.error(verErr);
      return;
    }
    try {
      JSON.parse(configJson);
    } catch {
      toast.error('服务配置 JSON 格式错误');
      return;
    }
    const updated: MCPService = {
      ...mcp,
      displayName: displayName.trim() || mcp.name,
      description: description.trim(),
      version: version.trim(),
      versions: [version.trim(), ...(mcp.versions || [])],
      configJson,
      usageDoc: usageDoc.trim() || undefined,
      toolDoc: toolDoc.trim() || undefined,
      updatedAt: new Date(),
    };
    onConfirm(updated);
    toast.success(`MCP「${updated.displayName}」已更新至 v${version.trim()}`);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-[600px] flex flex-col"
        style={{ maxHeight: 'min(90vh, 780px)' }}
        onPointerDownOutside={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>更新 MCP 服务</DialogTitle>
        </DialogHeader>

        <DialogBody className="flex-1 px-6">
          <div className="space-y-4">
            <Alert variant="warning">
              <AlertInfoIcon />
              <AlertDescription>
                仅更新企业 MCP 库中的版本。已下发至 Agent 实例的 MCP 不会自动升级，需按同步模式重新下发。
              </AlertDescription>
            </Alert>

            {/* 服务标识（只读） */}
            <div className="space-y-1.5">
              <MetaMedium as="label" tone="secondary">服务标识（不可修改）</MetaMedium>
              <Tooltip delayDuration={1000}>
                <TooltipTrigger asChild>
                  <Input value={mcp.name} disabled />
                </TooltipTrigger>
                <TooltipContent side="right">服务标识创建后不可修改</TooltipContent>
              </Tooltip>
            </div>

            {/* 名称 */}
            <div className="space-y-1.5">
              <MetaMedium as="label" tone="secondary" htmlFor="mcp-upd-name">名称</MetaMedium>
              <Input id="mcp-upd-name" value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
            </div>

            {/* 描述 */}
            <div className="space-y-1.5">
              <MetaMedium as="label" tone="secondary" htmlFor="mcp-upd-desc">描述</MetaMedium>
              <Textarea id="mcp-upd-desc" value={description} onChange={(e) => setDescription(e.target.value)} rows={2} className="resize-none" />
            </div>

            {/* 版本号 */}
            <div className="space-y-1.5">
              <MetaMedium as="label" tone="secondary" htmlFor="mcp-upd-version">
                版本号<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
              </MetaMedium>
              <Input
                id="mcp-upd-version"
                value={version}
                onChange={(e) => handleVersionChange(e.target.value)}
                placeholder={`新版本号需高于上一版本号 ${mcp.version}`}
                className={versionError ? 'border-red-400 focus:ring-red-400' : ''}
              />
              {versionError && <MetaText as="p" tone="danger">{versionError}</MetaText>}
            </div>

            {/* 服务配置 JSON */}
            <div className="space-y-1.5">
              <MetaMedium as="label" tone="secondary" htmlFor="mcp-upd-config">服务配置（JSON）</MetaMedium>
              <Textarea
                id="mcp-upd-config"
                value={configJson}
                onChange={(e) => setConfigJson(e.target.value)}
                rows={6}
                className="font-mono text-xs"
              />
            </div>

            {/* 使用说明 */}
            <div className="space-y-1.5">
              <MetaMedium as="label" tone="secondary" htmlFor="mcp-upd-usage">使用说明（Markdown）</MetaMedium>
              <Textarea id="mcp-upd-usage" value={usageDoc} onChange={(e) => setUsageDoc(e.target.value)} rows={4} className="font-mono text-xs" />
            </div>

            {/* 工具说明 */}
            <div className="space-y-1.5">
              <MetaMedium as="label" tone="secondary" htmlFor="mcp-upd-tool">工具说明（Markdown）</MetaMedium>
              <Textarea id="mcp-upd-tool" value={toolDoc} onChange={(e) => setToolDoc(e.target.value)} rows={4} className="font-mono text-xs" />
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
          <Button variant="claw-primary" onClick={handleSubmit} disabled={!version || !!versionError}>
            保存更新
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
