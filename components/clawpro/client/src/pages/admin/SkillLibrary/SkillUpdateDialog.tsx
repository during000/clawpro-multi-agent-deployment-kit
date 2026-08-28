/**
 * 更新 Skill 对话框（F-02）
 * - 预填当前 Skill 信息
 * - 可编辑 name、description、version、changeLog、categories
 * - changeLog 支持一键生成模板
 * - 可选上传新 ZIP 替换文件
 * - 版本号格式校验 + 必须高于当前版本
 */
import { useState, useEffect } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogBody } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Alert, AlertDescription, AlertInfoIcon } from '@/components/ui/alert';
import { ChevronDown, Sparkles, Search as SearchIcon, Check, Lock } from 'lucide-react';
import JSZip from 'jszip';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { SurfaceCard } from '@/components/ui/Surface';
import { StatusTag } from '@/components/ui/status-tag';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { Checkbox } from '@/components/ui/checkbox';
import { Switch } from '@/components/ui/switch';
import { BodyMedium, MetaText, MetaMedium, HelperText, CompactText, CardTitle, BodyText } from '@/components/ui/Typography';
import { type Skill, type SkillScope, type UploadedFile, type ScopeLockConfig } from './types';
import { DEFAULT_CATEGORIES, MOCK_GROUPS, MOCK_PROJECT_GROUPS } from './mockData';
import { isValidSemver, compareSemver } from './downloadUtils';
import { UploadRequirementsCard } from './UploadRequirementsCard';
import FileReplaceHelper from './FileReplaceHelper';
import UploadFileCard from './UploadFileCard';
import { SecurityScanCard } from '@/components/SecurityScanCard';
import { ScopeSelect } from '@/components/ScopeSelect';

interface SkillUpdateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  skill: Skill;
  onConfirm: (updatedSkill: Skill, changeLog: string) => void;
  defaultSecurityScan?: boolean;
  onDefaultSecurityScanChange?: (value: boolean) => void;
  securityServiceActive?: boolean;
  /** 当在「项目资产管理」内使用时，将应用范围锁定为指定组织（只读） */
  lockedScope?: ScopeLockConfig;
}

// 复用 UploadFileCard 的 UploadedFile 类型

// 解析 SKILL.md 文件内容
const parseSkillMd = (content: string): { name?: string; description?: string } | null => {
  const lines = content.split('\n').map(line => line.trim());
  if (lines[0] !== '---') return null;
  const result: { name?: string; description?: string } = {};
  for (let i = 1; i < lines.length; i++) {
    const line = lines[i];
    if (line.startsWith('---')) break;
    if (line.startsWith('name:')) result.name = line.substring(5).trim();
    else if (line.startsWith('description:')) result.description = line.substring(12).trim();
  }
  return Object.keys(result).length > 0 ? result : null;
};

// 可读取内容的文本文件扩展名
const TEXT_EXTENSIONS = ['.md', '.mdx', '.xml', '.json', '.txt', '.yaml', '.yml', '.toml', '.ini', '.cfg', '.conf', '.sh', '.bat', '.py', '.js', '.ts', '.css', '.html', '.htm', '.svg', '.env', '.gitignore', '.dockerfile'];
const isTextFile = (name: string) => {
  const lower = name.toLowerCase();
  if (!lower.includes('.') && !lower.includes('/')) return true;
  return TEXT_EXTENSIONS.some(ext => lower.endsWith(ext));
};

const parseZipFile = async (file: File): Promise<{
  files: Array<{ name: string; size: number; content?: string }>;
  skillmdContent?: string;
  skillmdParsed?: { name?: string; description?: string };
  error?: string;
}> => {
  try {
    const zip = new JSZip();
    const loaded = await zip.loadAsync(file);
    const files: Array<{ name: string; size: number; content?: string }> = [];
    let skillmdContent: string | undefined;
    let skillmdFound = false;
    const fileEntries: Array<{ relativePath: string; zipEntry: JSZip.JSZipObject }> = [];

    loaded.forEach((relativePath, zipEntry) => {
      if (zipEntry.dir) return;
      if (relativePath.startsWith('__MACOSX/') || relativePath.endsWith('.DS_Store')) return;
      if (relativePath.toLowerCase().endsWith('skill.md')) skillmdFound = true;
      fileEntries.push({ relativePath, zipEntry });
    });

    for (const { relativePath, zipEntry } of fileEntries) {
      const size = (zipEntry as any)._data ? (zipEntry as any)._data.uncompressedSize : 0;
      let content: string | undefined;
      if (isTextFile(relativePath)) {
        try { content = await zipEntry.async('text'); } catch { /* skip */ }
      }
      files.push({ name: relativePath, size, content });
    }

    files.sort((a, b) => {
      if (a.name.toLowerCase() === 'skill.md') return -1;
      if (b.name.toLowerCase() === 'skill.md') return 1;
      return a.name.localeCompare(b.name);
    });

    if (skillmdFound) {
      const skillmdFile = files.find(f => f.name.toLowerCase().endsWith('skill.md'));
      if (skillmdFile?.content) skillmdContent = skillmdFile.content;
    }

    const skillmdParsed = skillmdContent ? parseSkillMd(skillmdContent) : undefined;
    return { files, skillmdContent, skillmdParsed: skillmdParsed || undefined };
  } catch (error) {
    return { files: [], error: `ZIP 文件解析失败: ${error instanceof Error ? error.message : '未知错误'}` };
  }
};

export default function SkillUpdateDialog({ open, onOpenChange, skill, onConfirm, defaultSecurityScan = false, onDefaultSecurityScanChange = () => {}, securityServiceActive = true, lockedScope }: SkillUpdateDialogProps) {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    version: '',
    changeLog: '',
    categories: [] as string[],
    scope: 'public' as SkillScope,
    groupIds: [] as string[],
  });
  const [uploadedFile, setUploadedFile] = useState<UploadedFile | null>(null);
  const [fileExpanded, setFileExpanded] = useState(false);
  const [versionError, setVersionError] = useState('');
  const [groupSearchQuery, setGroupSearchQuery] = useState('');
  const [enableSecurityScan, setEnableSecurityScan] = useState(false);

  // 初始化 - 回显已有文件和当前 Skill 信息
  useEffect(() => {
    if (open && skill) {
      setFormData({
        name: skill.name,
        description: skill.description,
        version: '',
        changeLog: '',
        categories: [...skill.categories],
        scope: lockedScope ? 'private' : (skill.scope || 'public'),
        groupIds: lockedScope ? [lockedScope.lockedGroupId] : [...(skill.groupIds || [])],
      });
      setGroupSearchQuery('');
      setEnableSecurityScan(false);
      // 回显已有文件
      if (skill.files && skill.files.length > 0) {
        setUploadedFile({
          name: '当前文件',
          size: 0,
          status: 'success',
          files: skill.files,
        });
      } else {
        setUploadedFile(null);
      }
      setFileExpanded(false);
      setVersionError('');
    }
  }, [open, skill, lockedScope]);

  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      setUploadedFile(null);
      setFileExpanded(false);
      setVersionError('');
    }
    onOpenChange(newOpen);
  };

  // 版本号校验
  const validateVersion = (version: string): string => {
    if (!version) return '';
    if (!isValidSemver(version)) return '版本号格式必须为 x.y.z';
    if (compareSemver(version, skill.version) <= 0) return `新版本号需高于上个版本号 ${skill.version}`;
    return '';
  };

  const handleVersionChange = (value: string) => {
    setFormData(prev => ({ ...prev, version: value }));
    setVersionError(validateVersion(value));
  };

  // 新上传处理函数（供 UploadFileCard 调用）
  const handleZipUpload = async (files: FileList) => {
    const file = files[0];
    if (!file) return;
    if (!file.name.endsWith('.zip')) {
      setUploadedFile({ name: file.name, size: file.size, status: 'error', error: '只支持 ZIP 文件' });
      return;
    }
    setUploadedFile({ name: file.name, size: file.size, status: 'parsing' });
    const parseResult = await parseZipFile(file);
    const hasSKILLMd = parseResult.files.some(f => f.name.toLowerCase().endsWith('skill.md'));
    if (parseResult.error) {
      setUploadedFile({ name: file.name, size: file.size, status: 'error', error: parseResult.error });
    } else if (!hasSKILLMd) {
      setUploadedFile({ name: file.name, size: file.size, status: 'error', error: '不存在 SKILL.md 文件，请修改后重试', files: parseResult.files });
    } else {
      setUploadedFile({ name: file.name, size: file.size, status: 'success', files: parseResult.files, skillmdContent: parseResult.skillmdContent, skillmdParsed: parseResult.skillmdParsed });
    }
  };

  const handleFolderUpload = async (files: FileList) => {
    setUploadedFile({ name: '文件夹上传', size: 0, status: 'parsing' });
    setTimeout(async () => {
      const fileList: { name: string; size: number; content?: string }[] = [];
      let skillmdContent: string | undefined;
      let skillmdFound = false;
      for (let i = 0; i < files.length; i++) {
        const file = files[i];
        const relativePath = file.webkitRelativePath || file.name;
        let content: string | undefined;
        if (isTextFile(relativePath)) {
          try { content = await file.text(); } catch { /* skip */ }
        }
        fileList.push({ name: relativePath, size: file.size, content });
        if (relativePath.toLowerCase().endsWith('skill.md')) {
          const pathParts = relativePath.split('/');
          if (pathParts.length === 2 && pathParts[1].toLowerCase() === 'skill.md') {
            skillmdFound = true;
            skillmdContent = content || await file.text();
          }
        }
      }
      const displayFiles = fileList.filter(f => f.name.split('/').length > 1).map(f => {
        const pathParts = f.name.split('/');
        return { name: pathParts.slice(1).join('/'), size: f.size, content: f.content };
      });
      const skillmdParsed = skillmdContent ? parseSkillMd(skillmdContent) : undefined;
      if (!skillmdFound) {
        setUploadedFile({ name: '文件夹上传', size: 0, status: 'error', error: '不存在 Skill.md 文件或者不在根目录下，请修改后重试', files: displayFiles });
      } else {
        setUploadedFile({ name: '文件夹上传', size: 0, status: 'success', files: displayFiles, skillmdContent, skillmdParsed: skillmdParsed || undefined });
      }
    }, 0);
  };

  const handleFileRemove = () => {
    if (uploadedFile && uploadedFile.name !== '当前文件') {
      if (skill.files && skill.files.length > 0) {
        setUploadedFile({ name: '当前文件', size: 0, status: 'success', files: skill.files });
      } else {
        setUploadedFile(null);
      }
    } else {
      setUploadedFile(null);
    }
    setFileExpanded(false);
  };

  // 一键生成 changeLog
  const handleGenerateChangeLog = () => {
    const changes: string[] = [];
    let idx = 1;
    if (formData.name !== skill.name) {
      changes.push(`${idx}、修改名称字段`);
      idx++;
    }
    if (formData.description !== skill.description) {
      changes.push(`${idx}、修改描述字段`);
      idx++;
    }
    // 检查是否有新上传文件（非回显的原始文件）
    const hasNewUpload = uploadedFile && uploadedFile.name !== '当前文件' && uploadedFile.status === 'success';
    if (hasNewUpload) {
      changes.push(`${idx}、更新SKILL文件`);
    }
    if (changes.length === 0) {
      changes.push('无变更');
    }
    setFormData(prev => ({ ...prev, changeLog: changes.join('\n') }));
  };

  const handleSave = () => {
    if (!formData.version) {
      toast.error('请填写新版本号');
      return;
    }
    const verErr = validateVersion(formData.version);
    if (verErr) {
      setVersionError(verErr);
      toast.error(verErr);
      return;
    }

    // 获取新的文件列表
    const newFiles = uploadedFile && uploadedFile.name !== '当前文件' && uploadedFile.status === 'success'
      ? uploadedFile.files || []
      : skill.files || [];
    const newContent = uploadedFile && uploadedFile.name !== '当前文件' && uploadedFile.status === 'success'
      ? uploadedFile.skillmdContent || skill.content
      : skill.content;

    const updatedSkill: Skill = {
      ...skill,
      name: formData.name,
      description: formData.description,
      version: formData.version,
      categories: formData.categories,
      scope: formData.scope,
      groupIds: formData.scope === 'public' ? [] : formData.groupIds,
      files: newFiles,
      content: newContent,
      versions: [formData.version, ...(skill.versions || [])],
      versionHistory: [
        {
          version: formData.version,
          date: new Date().toISOString().split('T')[0],
          changeLog: formData.changeLog || undefined,
          files: newFiles,
        },
        ...(skill.versionHistory || []),
      ],
      uploadTime: new Date(),
      securityInfo: enableSecurityScan
        ? { overallStatus: 'scanning', engines: [] }
        : skill.securityInfo,
    };

    onConfirm(updatedSkill, formData.changeLog);
    toast.success(`Skill「${skill.name}」已更新至 v${formData.version}`);
    handleOpenChange(false);
  };

  const hasNewUpload = uploadedFile != null && uploadedFile.name !== '当前文件' && uploadedFile.status === 'success';
  const hasCurrentFile = uploadedFile != null && uploadedFile.name === '当前文件';

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-2xl max-h-[min(90vh,780px)] flex flex-col overflow-visible" onPointerDownOutside={(e) => e.preventDefault()}>
        <DialogHeader>
          <DialogTitle>更新 Skill</DialogTitle>
        </DialogHeader>

        <DialogBody className="flex-1 px-6">
          <div className="space-y-4">
          {/* 更新提示 */}
          <Alert variant="warning">
            <AlertInfoIcon />
            <AlertDescription>
              仅更新企业技能库中的技能版本。已下发至 Agent 实例的技能不会同步升级，需手动重新下发。
            </AlertDescription>
          </Alert>

          {/* 文件替换 */}
          <div className="space-y-3">
            <MetaMedium as="label" tone="secondary">文件（可选替换）</MetaMedium>

            <FileReplaceHelper show={!!uploadedFile && uploadedFile.status === 'success'} variant="skill" />

            <UploadFileCard
              file={uploadedFile}
              expanded={fileExpanded}
              onToggleExpand={() => setFileExpanded(!fileExpanded)}
              onZipUpload={handleZipUpload}
              onFolderUpload={handleFolderUpload}
              onRemove={handleFileRemove}
            />

            {!uploadedFile && (
              <UploadRequirementsCard variant="skill" />
            )}
          </div>

          {/* Slug（只读） */}
          <div>
            <MetaMedium as="label" tone="secondary" htmlFor="update-slug">
              唯一标识 (slug)<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
            </MetaMedium>
            <Tooltip delayDuration={1000}>
                <TooltipTrigger asChild>
                  <Input
                    id="update-slug"
                    value={skill.slug}
                    disabled
                    className="mt-1"
                  />
                </TooltipTrigger>
                <TooltipContent side="right" sideOffset={8}>
                  <p>slug 不允许修改</p>
                </TooltipContent>
              </Tooltip>
          </div>

          {/* Name */}
          <div>
            <MetaMedium as="label" tone="secondary" htmlFor="update-name">
              显示名称<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
            </MetaMedium>
            <Input
              id="update-name"
              value={formData.name}
              onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
              className="mt-1"
            />
          </div>

          {/* Description */}
          <div>
            <MetaMedium as="label" tone="secondary" htmlFor="update-desc">描述</MetaMedium>
            <Textarea
              id="update-desc"
              value={formData.description}
              onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
              className="mt-1"
              rows={2}
            />
          </div>

          {/* Version */}
          <div>
            <MetaMedium as="label" tone="secondary" htmlFor="update-version">
              版本号<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
            </MetaMedium>
            <Input
              id="update-version"
              value={formData.version}
              onChange={(e) => handleVersionChange(e.target.value)}
              placeholder={`新版本号需高于上一版本号 ${skill.version}`}
              className={`mt-1 ${versionError ? 'border-red-400 focus:ring-red-400' : ''}`}
            />
            {versionError && (
              <MetaText as="p" tone="danger" className="mt-1">{versionError}</MetaText>
            )}
          </div>

          {/* 分类 */}
          <div>
            <MetaMedium as="label" tone="secondary">分类</MetaMedium>
            <div className="flex flex-wrap gap-2 mt-2">
              {DEFAULT_CATEGORIES.map(cat => {
                const isSelected = formData.categories.includes(cat.id);
                return (
                  <button
                    key={cat.id}
                    onClick={() => {
                      setFormData(prev => ({
                        ...prev,
                        categories: prev.categories.includes(cat.id)
                          ? prev.categories.filter(id => id !== cat.id)
                          : [...prev.categories, cat.id]
                      }));
                    }}
                    className={`h-8 px-4 rounded-[4px] text-sm leading-[22px] tracking-[0.07px] border transition-colors whitespace-nowrap inline-flex items-center gap-1.5 ${
                      isSelected
                        ? 'border-blue-500 bg-[rgba(20,71,230,0.06)] text-[#020617]'
                        : 'bg-white border-[#EAEEF4] text-[#020617] hover:border-blue-500'
                    }`}
                  >
                    <Checkbox
                      checked={isSelected}
                      className="pointer-events-none"
                      tabIndex={-1}
                    />
                    {cat.name}
                  </button>
                );
              })}
            </div>
          </div>

          {/* 应用范围 */}
          <div>
            <MetaMedium as="label" tone="secondary">应用范围</MetaMedium>
            <div className="mt-2">
              {lockedScope ? (
                <div className="flex items-center gap-1.5 h-9 px-3 rounded-[4px] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] text-[var(--text-secondary)]">
                  <Lock className="w-3.5 h-3.5" />
                  <MetaMedium tone="secondary">{lockedScope.lockedGroupName}（已锁定为该组织）</MetaMedium>
                </div>
              ) : (
                <ScopeSelect
                  scope={formData.scope === 'public' ? 'all' : 'groups'}
                  selectedGroupIds={formData.groupIds}
                  groups={MOCK_GROUPS}
                  projects={MOCK_PROJECT_GROUPS}
                  onConfirm={(s, ids) => {
                    if (s === 'all') {
                      setFormData(prev => ({ ...prev, scope: 'public', groupIds: [] }));
                    } else {
                      setFormData(prev => ({ ...prev, scope: 'private', groupIds: ids }));
                    }
                  }}
                />
              )}
            </div>
          </div>

          {/* 更新说明 */}
          <div>
            <div className="flex items-center justify-between mb-1">
              <MetaMedium as="label" tone="secondary" htmlFor="update-changelog">更新说明</MetaMedium>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleGenerateChangeLog}
                className="h-7 px-2 text-xs gap-1"
              >
                <Sparkles className="w-3 h-3" />
                一键生成
              </Button>
            </div>
            <Textarea
              id="update-changelog"
              value={formData.changeLog}
              onChange={(e) => setFormData(prev => ({ ...prev, changeLog: e.target.value }))}
              placeholder="请填写本次更新内容"
              className="mt-1"
              rows={3}
            />
          </div>

          {/* 安全检测 */}
          <SecurityScanCard
            securityServiceActive={securityServiceActive}
            enableSecurityScan={enableSecurityScan}
            onEnableSecurityScanChange={(checked) => setEnableSecurityScan(checked)}
            defaultSecurityScan={defaultSecurityScan}
            onDefaultSecurityScanChange={onDefaultSecurityScanChange}
          />

        </div>
        </DialogBody>

        <DialogFooter className="flex-shrink-0">
          <Button variant="outline" onClick={() => handleOpenChange(false)}>
            取消
          </Button>
          <Button
            variant="dialog-confirm"
            onClick={handleSave}
            disabled={!formData.version || !!versionError}
          >
            保存更新
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
