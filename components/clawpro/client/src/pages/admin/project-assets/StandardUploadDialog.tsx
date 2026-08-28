/**
 * StandardUploadDialog - 项目内新增企业规范/Hook 配置。
 * 交互与「企业工具库 > 企业规范库 > 新增资源」保持一致，
 * 仅将应用范围锁定为当前项目。
 */
import { useEffect, useState } from 'react';
import { CheckCircle2 } from 'lucide-react';
import { toast } from 'sonner';
import { Alert, AlertDescription, AlertInfoIcon } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { MetaMedium, MetaText } from '@/components/ui/Typography';
import FileReplaceHelper from '../SkillLibrary/FileReplaceHelper';
import HookSettingsForm, {
  EMPTY_HOOK_FORM,
  buildHookManifestYaml,
  getHookFormError,
  type HookFormValue,
} from '../SkillLibrary/HookSettingsForm';
import UploadFileCard from '../SkillLibrary/UploadFileCard';
import { UploadRequirementsCard } from '../SkillLibrary/UploadRequirementsCard';
import type { ScopeLockConfig, UploadedFile } from '../SkillLibrary/types';
import {
  type AgentConfigAsset,
  type AssetKind,
  type TargetClient,
} from '../SkillLibrary/standardsStore';

const KIND_META: Record<AssetKind, { label: string; files: string[]; desc: string }> = {
  entry: {
    label: 'CLAUDE.md',
    files: ['CLAUDE.md', 'AGENTS.md', 'CODEBUDDY.md'],
    desc: '定义 Agent 启动时加载的全局说明、身份和工作准则。',
  },
  rule: {
    label: 'rules',
    files: ['rules'],
    desc: '存放编码、安全、交付、评审等企业规范文件。',
  },
  hook: {
    label: 'Hook 配置',
    files: ['hooks.yaml'],
    desc: '定义任务执行过程中按事件触发的本地自动化操作。',
  },
};

const KIND_CLIENTS: Record<AssetKind, TargetClient[]> = {
  entry: ['claude_code', 'codebuddy', 'codex'],
  rule: ['claude_code', 'codebuddy', 'workbuddy'],
  hook: ['claude_code', 'codebuddy', 'codex', 'workbuddy'],
};

interface StandardUploadDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  lockedScope: ScopeLockConfig;
  onConfirm: (asset: AgentConfigAsset) => void;
}

const emptyForm = () => ({
  slug: '',
  name: '',
  description: '',
  version: '1.0.0',
});

const createSlug = (name: string) =>
  name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '') || `asset-${Date.now()}`;

const isValidSemver = (version: string) => /^\d+\.\d+\.\d+$/.test(version.trim());

export default function StandardUploadDialog({
  open,
  onOpenChange,
  lockedScope,
  onConfirm,
}: StandardUploadDialogProps) {
  const [draftKind, setDraftKind] = useState<AssetKind>('rule');
  const [draftFile, setDraftFile] = useState<UploadedFile | null>(null);
  const [draftFileExpanded, setDraftFileExpanded] = useState(false);
  const [hookForm, setHookForm] = useState<HookFormValue>({ ...EMPTY_HOOK_FORM });
  const [formData, setFormData] = useState(emptyForm);

  const hasSuccessfulDraft = draftKind === 'hook' || draftFile?.status === 'success';

  const resetDraft = () => {
    setDraftKind('rule');
    setDraftFile(null);
    setDraftFileExpanded(false);
    setHookForm({ ...EMPTY_HOOK_FORM });
    setFormData(emptyForm());
  };

  useEffect(() => {
    if (open) resetDraft();
  }, [open]);

  const handleKindChange = (kind: AssetKind) => {
    if (kind === draftKind) return;
    setDraftKind(kind);
    setDraftFile(null);
    setDraftFileExpanded(false);
    setHookForm({ ...EMPTY_HOOK_FORM });
    setFormData(emptyForm());
  };

  const handleFileUpload = async (files: FileList) => {
    const file = files?.[0];
    if (!file) return;
    if (!file.name.toLowerCase().endsWith('.md')) {
      toast.error('企业规范仅支持 .md 文件');
      return;
    }

    const content = await file.text();
    const name = file.name.replace(/\.md$/i, '');
    setDraftFile({
      name: file.name,
      size: file.size,
      status: 'success',
      files: [{ name: file.name, size: file.size, content }],
    });
    setDraftFileExpanded(true);
    setFormData((prev) => ({
      ...prev,
      name: prev.name || name,
      slug: prev.slug || createSlug(name),
      description: prev.description || `${name} 企业规范配置`,
    }));
    toast.success(`已选择文件「${file.name}」`);
  };

  const handleSubmit = () => {
    if (draftKind !== 'hook' && draftFile?.status !== 'success') {
      toast.error('请上传企业规范文件');
      return;
    }
    if (draftKind === 'hook') {
      const hookError = getHookFormError(hookForm);
      if (hookError) {
        toast.error(hookError);
        return;
      }
    }
    if (!formData.slug.trim() || !formData.name.trim() || !formData.version.trim()) {
      toast.error('请填写完整的资源信息');
      return;
    }
    if (!/^[a-z0-9-]+$/.test(formData.slug.trim())) {
      toast.error('唯一标识仅支持小写字母、数字和连字符');
      return;
    }
    if (!isValidSemver(formData.version)) {
      toast.error('版本号格式必须为 x.y.z');
      return;
    }

    const content = draftKind === 'hook'
      ? buildHookManifestYaml(hookForm, {
          id: formData.slug,
          description: formData.description || formData.name,
        })
      : draftFile?.files?.[0]?.content || '';
    const asset: AgentConfigAsset = {
      id: `asset-${Date.now()}`,
      tenantId: 'tenant-openclaw',
      name: formData.name.trim(),
      slug: formData.slug.trim(),
      kind: draftKind,
      targetClients: KIND_CLIENTS[draftKind],
      contentMd: content,
      fileName: draftKind === 'hook' ? 'hooks.yaml' : draftFile?.name,
      hookCount: draftKind === 'hook' ? 1 : undefined,
      description: formData.description.trim(),
      version: formData.version.trim(),
      visibilityType: 'group',
      scope: 'private',
      groupIds: [lockedScope.lockedGroupId],
      enabled: true,
      alwaysApply: true,
      pathGlobs: [],
      checksum: `sha256:${Math.random().toString(16).slice(2, 8)}`,
      createdBy: '当前用户',
      updatedAt: new Date(),
      lastTaskStatus: 'pending',
    };

    onConfirm(asset);
    toast.success(`${KIND_META[draftKind].label}「${asset.name}」已创建`);
    onOpenChange(false);
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        onOpenChange(nextOpen);
        if (!nextOpen) resetDraft();
      }}
    >
      <DialogContent
        size="lg"
        className="max-h-[min(90vh,780px)] flex flex-col"
        onPointerDownOutside={(event) => event.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>新增企业规范或 Hook 配置</DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6">
          <div className="space-y-4">
            {draftKind === 'hook' ? (
              <Alert variant="default">
                <AlertInfoIcon />
                <AlertDescription>填写表单创建 Hook，发布后自动生成 hooks.yaml。</AlertDescription>
              </Alert>
            ) : !draftFile && (
              <Alert variant="warning">
                <AlertInfoIcon />
                <AlertDescription>请先上传 Markdown 企业规范，然后填写资源信息。</AlertDescription>
              </Alert>
            )}

            <div className="space-y-2">
              <div>
                <MetaText as="label" tone="secondary">
                  1. 资源类型 <span className="text-[var(--text-danger)]">*</span>
                </MetaText>
                <MetaText as="p" tone="weak" className="mt-1">
                  资源会根据类型自动适配可下发的 Agent。
                </MetaText>
              </div>
              <div className="grid gap-2 sm:grid-cols-3">
                {(Object.keys(KIND_META) as AssetKind[]).map((kind) => {
                  const active = draftKind === kind;
                  return (
                    <button
                      key={kind}
                      type="button"
                      onClick={() => handleKindChange(kind)}
                      className={`flex min-h-[82px] min-w-0 items-start gap-2 overflow-hidden rounded-[4px] border p-3 text-left transition-colors ${
                        active
                          ? 'border-[#C7D7FE] bg-[#E8ECFE] text-[#1447E6]'
                          : 'border-[var(--cp-border)] bg-white text-[var(--text-secondary)] hover:border-[var(--cp-brand-blue)]'
                      }`}
                    >
                      {active ? (
                        <CheckCircle2 className="mt-0.5 size-4 shrink-0" />
                      ) : (
                        <span className="mt-0.5 size-4 shrink-0" />
                      )}
                      <span className="min-w-0 flex-1 overflow-hidden">
                        <span className="flex min-h-[48px] min-w-0 flex-wrap content-start gap-1.5">
                          {KIND_META[kind].files.map((fileName) => (
                            <span
                              key={fileName}
                              className={`min-w-0 rounded-[4px] border px-1 py-0.5 text-center font-mono text-[11px] leading-5 whitespace-nowrap ${
                                active
                                  ? 'border-[#C7D7FE] bg-white/70 text-[#1447E6]'
                                  : 'border-[var(--cp-border)] bg-[#F8FAFC] text-[var(--text-secondary)]'
                              }`}
                            >
                              {fileName}
                            </span>
                          ))}
                        </span>
                        <MetaText as="span" tone="weak" className="mt-1 block leading-relaxed">
                          {KIND_META[kind].desc}
                        </MetaText>
                      </span>
                    </button>
                  );
                })}
              </div>
            </div>

            {draftKind === 'hook' ? (
              <HookSettingsForm
                value={hookForm}
                onChange={setHookForm}
                manifestId={formData.slug}
                manifestDescription={formData.description || formData.name}
              />
            ) : (
              <div className="space-y-3">
                <MetaMedium as="label" tone="secondary">
                  {draftFile ? '文件' : '选择上传方式'}
                </MetaMedium>
                <FileReplaceHelper show={hasSuccessfulDraft} variant="standard">
                  已上传文件，如需重新上传请先删除
                </FileReplaceHelper>
                <UploadFileCard
                  file={draftFile}
                  expanded={draftFileExpanded}
                  onToggleExpand={() => setDraftFileExpanded((expanded) => !expanded)}
                  onZipUpload={handleFileUpload}
                  onRemove={() => {
                    setDraftFile(null);
                    setDraftFileExpanded(false);
                    setFormData(emptyForm());
                  }}
                  variant="standard"
                  accept=".md"
                  uploadHint="点击或拖拽 Markdown 文件上传"
                  uploadButtonLabel="上传 Markdown"
                />
                {!draftFile && <UploadRequirementsCard variant="standard" />}
              </div>
            )}

            <div className={`space-y-4 transition-opacity ${hasSuccessfulDraft ? '' : 'opacity-60'}`}>
              <div>
                <MetaMedium as="h4" tone="secondary">资源信息</MetaMedium>
                <MetaText as="p" tone="weak" className="mt-1">
                  {draftKind === 'hook'
                    ? '填写资源名称，Hook 配置由企业规范库统一管理。'
                    : '根据上传文件填写基础信息，资源由企业规范库统一管理。'}
                </MetaText>
              </div>

              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary" htmlFor="project-standard-slug">
                  唯一标识 (slug)<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                </MetaMedium>
                <Input
                  id="project-standard-slug"
                  value={formData.slug}
                  disabled={!hasSuccessfulDraft}
                  onChange={(event) => setFormData((prev) => ({
                    ...prev,
                    slug: event.target.value.toLowerCase().replace(/\s+/g, '-'),
                  }))}
                  placeholder="e.g., frontend-react-rules"
                />
                <MetaText tone="weak">仅支持小写字母、数字和连字符。</MetaText>
              </div>

              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary" htmlFor="project-standard-name">
                  显示名称<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                </MetaMedium>
                <Input
                  id="project-standard-name"
                  value={formData.name}
                  disabled={!hasSuccessfulDraft}
                  onChange={(event) => setFormData((prev) => ({ ...prev, name: event.target.value }))}
                  placeholder="e.g., 前端 React 规范"
                />
              </div>

              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary" htmlFor="project-standard-description">描述</MetaMedium>
                <Textarea
                  id="project-standard-description"
                  value={formData.description}
                  disabled={!hasSuccessfulDraft}
                  onChange={(event) => setFormData((prev) => ({ ...prev, description: event.target.value }))}
                  placeholder="资源的简要描述"
                  rows={2}
                />
              </div>

              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary" htmlFor="project-standard-version">
                  版本号<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                </MetaMedium>
                <Input
                  id="project-standard-version"
                  value={formData.version}
                  disabled={!hasSuccessfulDraft}
                  onChange={(event) => setFormData((prev) => ({ ...prev, version: event.target.value }))}
                  placeholder="e.g., 1.0.0"
                />
              </div>

              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary">应用范围</MetaMedium>
                <div className="flex h-9 items-center rounded-[4px] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] px-3">
                  <MetaText tone="secondary">{lockedScope.lockedGroupName}</MetaText>
                </div>
                <MetaText tone="weak">在项目内新增的资源，应用范围固定为当前项目。</MetaText>
              </div>
            </div>
          </div>
        </DialogBody>

        <DialogFooter>
          <Button variant="claw-outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button
            variant="dialog-confirm"
            onClick={handleSubmit}
            disabled={
              !hasSuccessfulDraft ||
              (draftKind === 'hook' && !!getHookFormError(hookForm)) ||
              !formData.slug.trim() ||
              !formData.name.trim() ||
              !formData.version.trim()
            }
          >
            <CheckCircle2 className="size-4" />
            发布资源
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
