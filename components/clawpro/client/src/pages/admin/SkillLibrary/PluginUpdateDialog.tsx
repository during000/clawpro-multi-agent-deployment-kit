/**
 * PluginUpdateDialog - 更新插件弹窗
 * 参照 SkillUpdateDialog 交互和样式进行优化
 */
import { useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogBody } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Alert, AlertDescription, AlertInfoIcon } from '@/components/ui/alert';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { SurfaceCard } from '@/components/ui/Surface';
import { Checkbox } from '@/components/ui/checkbox';
import { AlertCircle, ChevronDown, ChevronRight, FileText, Loader, Trash2, Search as SearchIcon, Sparkles, Check, Lock } from 'lucide-react';
import JSZip from 'jszip';
import { type Plugin, type PluginScope } from './PluginUploadDialog';
import { type ScopeLockConfig } from './types';
import { compareSemver, isValidSemver } from './downloadUtils';
import { BodyMedium, MetaText, MetaMedium, HelperText, CompactText, CardTitle, BodyText } from '@/components/ui/Typography';
import { UploadRequirementsCard } from './UploadRequirementsCard';
import FileReplaceHelper from './FileReplaceHelper';
import UploadFileCard from './UploadFileCard';
import { DEFAULT_CATEGORIES, MOCK_GROUPS, MOCK_PROJECT_GROUPS } from './mockData';
import { StatusTag } from '@/components/ui/status-tag';
import { SecurityScanCard } from '@/components/SecurityScanCard';
import { ScopeSelect } from '@/components/ScopeSelect';

interface PluginUpdateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  plugin: Plugin;
  onConfirm: (updatedPlugin: Plugin) => void;
  defaultSecurityScan?: boolean;
  onDefaultSecurityScanChange?: (value: boolean) => void;
  securityServiceActive?: boolean;
  /** 当在「项目资产管理」内使用时，将应用范围锁定为指定组织（只读） */
  lockedScope?: ScopeLockConfig;
}

interface UploadedFile {
  name: string;
  size: number;
  status: 'success' | 'error' | 'parsing';
  error?: string;
  pluginJsonFound?: boolean;
  packageJsonFound?: boolean;
  pluginJsonParsed?: { name?: string; description?: string };
  files?: Array<{ name: string; size: number; content?: string }>;
}

const PLUGIN_MANIFEST_FILE = 'openclaw.plugin.json';
const TEXT_EXTENSIONS = ['.md', '.mdx', '.xml', '.json', '.txt', '.yaml', '.yml', '.toml', '.ini', '.cfg', '.conf', '.sh', '.bat', '.py', '.js', '.ts', '.css', '.html', '.htm', '.svg', '.env', '.gitignore', '.dockerfile'];
const isTextFile = (name: string) => {
  const lower = name.toLowerCase();
  if (!lower.includes('.') && !lower.includes('/')) return true;
  return TEXT_EXTENSIONS.some(ext => lower.endsWith(ext));
};

const parseZipFile = async (file: File) => {
  try {
    const zip = new JSZip();
    const loaded = await zip.loadAsync(file);
    const files: Array<{ name: string; size: number; content?: string }> = [];
    let pluginJsonFound = false;
    let packageJsonFound = false;
    const fileEntries: Array<{ relativePath: string; zipEntry: JSZip.JSZipObject }> = [];

    loaded.forEach((relativePath, zipEntry) => {
      if (zipEntry.dir) return;
      if (relativePath.startsWith('__MACOSX/') || relativePath.endsWith('.DS_Store')) return;
      const parts = relativePath.split('/');
      const fileName = parts[parts.length - 1];
      if (fileName === PLUGIN_MANIFEST_FILE && parts.length <= 2) {
        pluginJsonFound = true;
      }
      if (fileName === 'package.json' && parts.length <= 2) {
        packageJsonFound = true;
      }
      fileEntries.push({ relativePath, zipEntry });
    });

    for (const { relativePath, zipEntry } of fileEntries) {
      const size = (zipEntry as any)._data ? (zipEntry as any)._data.uncompressedSize : 0;
      let content: string | undefined;
      if (isTextFile(relativePath)) {
        try { content = await zipEntry.async('text'); } catch { /* ignore */ }
      }
      files.push({ name: relativePath, size, content });
    }

    files.sort((a, b) => {
      if (a.name.toLowerCase().endsWith(PLUGIN_MANIFEST_FILE)) return -1;
      if (b.name.toLowerCase().endsWith(PLUGIN_MANIFEST_FILE)) return 1;
      return a.name.localeCompare(b.name);
    });

    let pluginJsonParsed: { name?: string; description?: string } | undefined;
    if (pluginJsonFound) {
      const pluginJsonFile = files.find(f => f.name.endsWith(PLUGIN_MANIFEST_FILE));
      if (pluginJsonFile?.content) {
        try {
          const parsed = JSON.parse(pluginJsonFile.content);
          pluginJsonParsed = {};
          if (parsed.name && typeof parsed.name === 'string') pluginJsonParsed.name = parsed.name;
          if (parsed.description && typeof parsed.description === 'string') pluginJsonParsed.description = parsed.description;
        } catch { /* JSON 解析失败则不填充 */ }
      }
    }

    return { files, pluginJsonFound, packageJsonFound, pluginJsonParsed };
  } catch (error) {
    return {
      files: [] as Array<{ name: string; size: number; content?: string }>,
      pluginJsonFound: false,
      packageJsonFound: false,
      pluginJsonParsed: undefined,
      error: `ZIP 文件解析失败: ${error instanceof Error ? error.message : '未知错误'}`,
    };
  }
};

export default function PluginUpdateDialog({
  open,
  onOpenChange,
  plugin,
  onConfirm,
  defaultSecurityScan = false,
  onDefaultSecurityScanChange = () => {},
  securityServiceActive = true,
  lockedScope,
}: PluginUpdateDialogProps) {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    version: '',
    updateNote: '',
    categories: [] as string[],
    scope: 'public' as PluginScope,
    groupIds: [] as string[],
  });
  const [uploadedFiles, setUploadedFiles] = useState<UploadedFile[]>([]);
  const [expandedFile, setExpandedFile] = useState<boolean>(false);
  const [versionError, setVersionError] = useState('');
  const [groupSearchQuery, setGroupSearchQuery] = useState('');
  const [enableSecurityScan, setEnableSecurityScan] = useState(false);
  const [localDefaultSecurityScan, setLocalDefaultSecurityScan] = useState(defaultSecurityScan);

  useEffect(() => {
    if (open && plugin) {
      setFormData({
        name: plugin.name,
        description: plugin.description,
        version: '',
        updateNote: '',
        categories: [],
        scope: lockedScope ? 'private' : (plugin.scope || 'public'),
        groupIds: lockedScope ? [lockedScope.lockedGroupId] : [...(plugin.groupIds || [])],
      });
      setGroupSearchQuery('');
      setEnableSecurityScan(false);
      setLocalDefaultSecurityScan(defaultSecurityScan);
      // 回显已有文件
      if (plugin.files && plugin.files.length > 0) {
        setUploadedFiles([{
          name: '当前文件',
          size: 0,
          status: 'success',
          files: plugin.files,
        }]);
      } else {
        setUploadedFiles([]);
      }
      setExpandedFile(false);
      setVersionError('');
    }
  }, [open, plugin, defaultSecurityScan, lockedScope]);

  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      setUploadedFiles([]);
      setExpandedFile(false);
      setVersionError('');
    }
    onOpenChange(newOpen);
  };

  const validateVersion = (version: string): string => {
    const nextVersion = version.trim();
    if (!nextVersion) return '请填写新版本号';
    if (!isValidSemver(nextVersion)) return '版本号格式必须为 x.y.z';
    if (compareSemver(nextVersion, plugin.version) <= 0) return `新版本号需高于上个版本号 v${plugin.version}`;
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
      setUploadedFiles([{ name: file.name, size: file.size, status: 'error', error: '只支持 ZIP 文件' }]);
      return;
    }
    setUploadedFiles([{ name: file.name, size: file.size, status: 'parsing' }]);
    const parseResult = await parseZipFile(file);

    if (parseResult.error) {
      setUploadedFiles([{ name: file.name, size: file.size, status: 'error', error: parseResult.error }]);
    } else if (!parseResult.pluginJsonFound && !parseResult.packageJsonFound) {
      setUploadedFiles([{ name: file.name, size: file.size, status: 'error', error: '不存在 openclaw.plugin.json 和 package.json 文件，请修改后重试', files: parseResult.files }]);
    } else if (!parseResult.pluginJsonFound) {
      setUploadedFiles([{ name: file.name, size: file.size, status: 'error', error: '不存在 openclaw.plugin.json 文件，请修改后重试', files: parseResult.files }]);
    } else if (!parseResult.packageJsonFound) {
      setUploadedFiles([{ name: file.name, size: file.size, status: 'error', error: '不存在 package.json 文件，请修改后重试', files: parseResult.files }]);
    } else {
      setUploadedFiles([{ name: file.name, size: file.size, status: 'success', files: parseResult.files, pluginJsonFound: true, packageJsonFound: true, pluginJsonParsed: parseResult.pluginJsonParsed }]);
    }
  };

  const handleFolderUpload = async (files: FileList) => {
    setUploadedFiles([{ name: '文件夹上传', size: 0, status: 'parsing' }]);
    setTimeout(async () => {
      const fileList: { name: string; size: number; content?: string }[] = [];
      let pluginJsonFound = false;
      let packageJsonFound = false;
      for (let i = 0; i < files.length; i++) {
        const file = files[i];
        const relativePath = file.webkitRelativePath || file.name;
        let content: string | undefined;
        if (isTextFile(relativePath)) {
          try { content = await file.text(); } catch { /* skip */ }
        }
        fileList.push({ name: relativePath, size: file.size, content });
        if (relativePath.toLowerCase().endsWith(PLUGIN_MANIFEST_FILE.toLowerCase())) {
          const pathParts = relativePath.split('/');
          if (pathParts.length === 2 && pathParts[1].toLowerCase() === PLUGIN_MANIFEST_FILE.toLowerCase()) {
            pluginJsonFound = true;
          }
        }
        if (relativePath.toLowerCase().endsWith('package.json')) {
          const pathParts = relativePath.split('/');
          if (pathParts.length === 2 && pathParts[1].toLowerCase() === 'package.json') {
            packageJsonFound = true;
          }
        }
      }
      const displayFiles = fileList.filter(f => f.name.split('/').length > 1).map(f => {
        const pathParts = f.name.split('/');
        return { name: pathParts.slice(1).join('/'), size: f.size, content: f.content };
      });
      
      let pluginJsonParsed: { name?: string; description?: string } | undefined;
      if (pluginJsonFound) {
        const pluginJsonFile = displayFiles.find(f => f.name.toLowerCase() === PLUGIN_MANIFEST_FILE.toLowerCase());
        if (pluginJsonFile?.content) {
          try {
            const parsed = JSON.parse(pluginJsonFile.content);
            pluginJsonParsed = {};
            if (parsed.name && typeof parsed.name === 'string') pluginJsonParsed.name = parsed.name;
            if (parsed.description && typeof parsed.description === 'string') pluginJsonParsed.description = parsed.description;
          } catch { /* JSON 解析失败则不填充 */ }
        }
      }
      
      if (!pluginJsonFound && !packageJsonFound) {
        setUploadedFiles([{ name: '文件夹上传', size: 0, status: 'error', error: '不存在 openclaw.plugin.json 和 package.json 文件，请修改后重试', files: displayFiles }]);
      } else if (!pluginJsonFound) {
        setUploadedFiles([{ name: '文件夹上传', size: 0, status: 'error', error: '不存在 openclaw.plugin.json 文件，请修改后重试', files: displayFiles }]);
      } else if (!packageJsonFound) {
        setUploadedFiles([{ name: '文件夹上传', size: 0, status: 'error', error: '不存在 package.json 文件，请修改后重试', files: displayFiles }]);
      } else {
        setUploadedFiles([{ name: '文件夹上传', size: 0, status: 'success', files: displayFiles, pluginJsonFound: true, packageJsonFound: true, pluginJsonParsed }]);
      }
    }, 0);
  };

  const handleRemoveFile = () => {
    const hasCurrent = uploadedFiles.some(f => f.name === '当前文件');
    if (hasCurrent && uploadedFiles.length > 0 && uploadedFiles[0].name === '当前文件') {
      // 删除当前文件，还原为无文件状态
      setUploadedFiles([]);
    } else {
      // 删除新上传的文件，还原为原始文件
      if (plugin.files && plugin.files.length > 0) {
        setUploadedFiles([{
          name: '当前文件',
          size: 0,
          status: 'success',
          files: plugin.files,
        }]);
      } else {
        setUploadedFiles([]);
      }
    }
    setExpandedFile(false);
  };

  // 一键生成 updateNote
  const handleGenerateUpdateNote = () => {
    const changes: string[] = [];
    let idx = 1;
    if (formData.name !== plugin.name) {
      changes.push(`${idx}、修改名称字段`);
      idx++;
    }
    if (formData.description !== plugin.description) {
      changes.push(`${idx}、修改描述字段`);
      idx++;
    }
    // 检查是否有新上传文件（非回显的原始文件）
    const hasNewUpload = uploadedFiles.length > 0 && uploadedFiles[0].name !== '当前文件' && uploadedFiles[0].status === 'success';
    if (hasNewUpload) {
      changes.push(`${idx}、更新插件文件`);
    }
    if (changes.length === 0) {
      changes.push('无变更');
    }
    setFormData(prev => ({ ...prev, updateNote: changes.join('\n') }));
  };

  const hasNewUpload = uploadedFiles.length > 0 && uploadedFiles[0].name !== '当前文件' && uploadedFiles[0].status === 'success';
  const hasCurrentFile = uploadedFiles.some(f => f.name === '当前文件');

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
    const newFiles = uploadedFiles.length > 0 && uploadedFiles[0].name !== '当前文件' && uploadedFiles[0].status === 'success'
      ? uploadedFiles[0].files || []
      : plugin.files || [];

    const updatedPlugin: Plugin = {
      ...plugin,
      name: formData.name,
      description: formData.description,
      version: formData.version,
      categories: formData.categories,
      scope: formData.scope,
      groupIds: formData.scope === 'public' ? [] : formData.groupIds,
      files: newFiles,
      content: `# ${formData.name}\n\n${formData.description}`,
      uploadTime: new Date(),
      versions: [formData.version, ...(plugin.versions || [])],
      versionHistory: [
        {
          version: formData.version,
          date: new Date().toISOString().split('T')[0],
          changeLog: formData.updateNote || undefined,
          files: newFiles,
        },
        ...(plugin.versionHistory || []),
      ],
      securityInfo: enableSecurityScan
        ? { overallStatus: 'scanning', engines: [] }
        : plugin.securityInfo,
    };

    onConfirm(updatedPlugin);
    toast.success(`插件「${plugin.name}」已更新至 v${formData.version}`);
    handleOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-2xl overflow-visible" style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }} onPointerDownOutside={(e) => e.preventDefault()}>
        <DialogHeader>
          <DialogTitle>更新插件</DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6 flex-1">
        <div className="space-y-4">
          {/* 更新提示 */}
          <Alert variant="warning">
            <AlertInfoIcon />
            <AlertDescription>
              仅更新企业插件库中的插件版本。已下发至 Agent 实例的插件不会同步升级，需手动重新下发。
            </AlertDescription>
          </Alert>

          {/* 文件替换 */}
          <div className="space-y-3">
            <MetaMedium as="label" tone="secondary">文件（可选替换）</MetaMedium>

            <FileReplaceHelper show={uploadedFiles.length > 0 && uploadedFiles[0].status === 'success'} variant="plugin-update" />

            <UploadFileCard
              file={uploadedFiles.length > 0 ? uploadedFiles[0] : null}
              expanded={expandedFile}
              onToggleExpand={() => setExpandedFile(!expandedFile)}
              onZipUpload={handleZipUpload}
              onFolderUpload={handleFolderUpload}
              onRemove={() => handleRemoveFile()}
            />

            {!hasCurrentFile && !hasNewUpload && (
              <UploadRequirementsCard variant="plugin-update" />
            )}
          </div>



          {/* Slug（只读） */}
          <div>
            <MetaMedium as="label" tone="secondary" htmlFor="plugin-update-slug">
              唯一标识 (slug)<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
            </MetaMedium>
            <Tooltip delayDuration={1000}>
                <TooltipTrigger asChild>
                  <Input
                    id="plugin-update-slug"
                    value={plugin.slug}
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
            <MetaMedium as="label" tone="secondary" htmlFor="plugin-update-name">
              显示名称<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
            </MetaMedium>
            <Input
              id="plugin-update-name"
              value={formData.name}
              onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
              className="mt-1"
            />
          </div>

          {/* Description */}
          <div>
            <MetaMedium as="label" tone="secondary" htmlFor="plugin-update-desc">描述</MetaMedium>
            <Textarea
              id="plugin-update-desc"
              value={formData.description}
              onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
              className="mt-1"
              rows={2}
            />
          </div>

          {/* Version */}
          <div>
            <MetaMedium as="label" tone="secondary" htmlFor="plugin-update-version">
              版本号<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
            </MetaMedium>
            <Input
              id="plugin-update-version"
              value={formData.version}
              onChange={(e) => handleVersionChange(e.target.value)}
              placeholder={`新版本号需高于上一版本号 ${plugin.version}`}
              className={`mt-1 ${versionError ? 'border-red-400 focus:ring-red-400' : ''}`}
            />
            {versionError && (
              <MetaText as="p" tone="danger" className="mt-1">{versionError}</MetaText>
            )}
          </div>

          {/* 更新说明 */}
          <div>
            <div className="flex items-center justify-between mb-1">
              <MetaMedium as="label" tone="secondary" htmlFor="plugin-update-note">更新说明</MetaMedium>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleGenerateUpdateNote}
                className="h-7 px-2 text-xs gap-1"
              >
                <Sparkles className="w-3 h-3" />
                一键生成
              </Button>
            </div>
            <Textarea
              id="plugin-update-note"
              value={formData.updateNote}
              onChange={(e) => setFormData(prev => ({ ...prev, updateNote: e.target.value }))}
              placeholder="请填写本次更新内容"
              className="mt-1"
              rows={3}
            />
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

          {/* 安全检测 */}
          <SecurityScanCard
            securityServiceActive={securityServiceActive}
            enableSecurityScan={enableSecurityScan}
            onEnableSecurityScanChange={(checked) => setEnableSecurityScan(checked)}
            defaultSecurityScan={localDefaultSecurityScan}
            onDefaultSecurityScanChange={(checked) => {
              setLocalDefaultSecurityScan(checked);
              onDefaultSecurityScanChange(checked);
            }}
            checkboxId="plugin-default-security-scan"
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
