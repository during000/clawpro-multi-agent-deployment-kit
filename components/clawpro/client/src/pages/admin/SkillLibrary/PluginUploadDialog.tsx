/**
 * PluginUploadDialog - 发布插件弹窗
 * 只支持上传 ZIP，校验 agent.plugin.json + package.json
 *
 * 视觉与交互对齐 SkillUploadDialog（发布 Skill 弹窗）：
 *  - 使用 UploadFileCard 组件处理文件上传和显示
 *  - DialogBody 管理滚动 + DialogFooter 主按钮
 *  - 上传要求卡片 + 下载样例
 *  - 表单字段：使用 MetaMedium + HelperText 规范
 *  - DialogFooter 主按钮使用 variant="dialog-confirm"
 *
 * 业务差异（保留）：
 *  - 校验 agent.plugin.json + package.json 而非 SKILL.md
 *  - 自动从 agent.plugin.json 填充 name / description
 *  - 不含分类 / 安全检测
 */
import { useState, useEffect } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogBody } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { CircleAlert } from 'lucide-react';
import JSZip from 'jszip';
import { type SkillScope, type UploadedFile, type ScopeLockConfig } from './types';
import { UploadRequirementsCard } from './UploadRequirementsCard';
import UploadFileCard from './UploadFileCard';
import FileReplaceHelper from './FileReplaceHelper';
import { MetaMedium, HelperText } from '@/components/ui/Typography';
import { ScopeSelect } from '@/components/ScopeSelect';
import { MOCK_GROUPS, MOCK_PROJECT_GROUPS } from './mockData';

export interface Plugin {
  id: string;
  slug: string;
  name: string;
  description: string;
  version: string;
  scope: SkillScope;
  groupIds: string[];
  uploadTime: Date;
  versions: string[];
  files: Array<{ name: string; size: number; content?: string }>;
  content?: string;
  categories?: string[];
  versionHistory?: Array<{
    version: string;
    date: string;
    changeLog?: string;
    files?: Array<{ name: string; size: number; content?: string }>;
  }>;
  securityInfo?: {
    overallStatus: string;
    engines: any[];
  };
}

/** @deprecated Use SkillScope from './types' instead */
export type PluginScope = SkillScope;

interface PluginUploadDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (plugin: Plugin) => void;
  existingSlugs?: string[];
  /** 当在「项目资产管理」内使用时，将应用范围锁定为指定组织（只读） */
  lockedScope?: ScopeLockConfig;
}

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
      // 根目录或单层文件夹下
      if (fileName === 'openclaw.plugin.json' && parts.length <= 2) {
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
      if (a.name.toLowerCase().endsWith('openclaw.plugin.json')) return -1;
      if (b.name.toLowerCase().endsWith('openclaw.plugin.json')) return 1;
      return a.name.localeCompare(b.name);
    });

    // 解析 agent.plugin.json 内容
    let pluginJsonParsed: { name?: string; description?: string } | undefined;
    if (pluginJsonFound) {
      const pluginJsonFile = files.find(f => f.name.endsWith('openclaw.plugin.json'));
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
    return { files: [] as Array<{ name: string; size: number; content?: string }>, pluginJsonFound: false, packageJsonFound: false, pluginJsonParsed: undefined, error: `ZIP 文件解析失败: ${error instanceof Error ? error.message : '未知错误'}` };
  }
};

const emptyForm = () => ({
  slug: '',
  name: '',
  description: '',
  version: '1.0.0',
  scope: 'public' as SkillScope,
  groupIds: [] as string[],
});

export default function PluginUploadDialog({ open, onOpenChange, onConfirm, existingSlugs = [], lockedScope }: PluginUploadDialogProps) {
  const [uploadedFiles, setUploadedFiles] = useState<UploadedFile[]>([]);
  const [expandedFile, setExpandedFile] = useState<string | null>(null);
  const [formData, setFormData] = useState(emptyForm());

  // 锁定场景：强制应用范围为指定组织
  useEffect(() => {
    if (open && lockedScope) {
      setFormData(prev => ({ ...prev, scope: 'private', groupIds: [lockedScope.lockedGroupId] }));
    }
  }, [open, lockedScope]);

  const hasSuccessfulUpload = uploadedFiles.some(f => f.status === 'success');

  const freshForm = () => (lockedScope
    ? { ...emptyForm(), scope: 'private' as SkillScope, groupIds: [lockedScope.lockedGroupId] }
    : emptyForm());

  const resetAll = () => {
    setUploadedFiles([]);
    setExpandedFile(null);
    setFormData(freshForm());
  };

  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) resetAll();
    onOpenChange(newOpen);
  };

  const handleRemoveFile = (name: string) => {
    setUploadedFiles(prev => prev.filter(f => f.name !== name));
    if (expandedFile === name) setExpandedFile(null);
    // 删除文件时也清空表单数据，与 SkillUploadDialog 行为一致
    setFormData(freshForm());
  };

  const handleZipUpload = (files: FileList) => {
    const event = {
      target: { files },
    } as unknown as React.ChangeEvent<HTMLInputElement>;
    handleFileSelect(event);
  };

  const handleFileSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    setUploadedFiles([]);
    setExpandedFile(null);
    const files = event.target.files;
    if (!files) return;

    const newFiles: UploadedFile[] = [];
    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      if (!file.name.endsWith('.zip')) {
        newFiles.push({ name: file.name, size: file.size, status: 'error', error: '只支持 ZIP 文件' });
        continue;
      }
      newFiles.push({ name: file.name, size: file.size, status: 'parsing' });
    }
    setUploadedFiles(newFiles);

    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      if (!file.name.endsWith('.zip')) continue;
      const parseResult = await parseZipFile(file);

      setUploadedFiles(prev => {
        const updated = [...prev];
        const idx = updated.findIndex(f => f.name === file.name);
        if (idx !== -1) {
          if (parseResult.error) {
            updated[idx] = { name: file.name, size: file.size, status: 'error', error: parseResult.error };
          } else if (!parseResult.pluginJsonFound && !parseResult.packageJsonFound) {
            updated[idx] = { name: file.name, size: file.size, status: 'error', error: '不存在 agent.plugin.json 和 package.json 文件，请修改后重试', files: parseResult.files };
          } else if (!parseResult.pluginJsonFound) {
            updated[idx] = { name: file.name, size: file.size, status: 'error', error: '不存在 agent.plugin.json 文件，请修改后重试', files: parseResult.files };
          } else if (!parseResult.packageJsonFound) {
            updated[idx] = { name: file.name, size: file.size, status: 'error', error: '不存在 package.json 文件，请修改后重试', files: parseResult.files };
          } else {
            updated[idx] = { name: file.name, size: file.size, status: 'success', files: parseResult.files, pluginJsonFound: true, packageJsonFound: true, pluginJsonParsed: parseResult.pluginJsonParsed };
            // 自动填充表单
            if (parseResult.pluginJsonParsed?.name && !formData.name) {
              setFormData(prev => ({ ...prev, name: parseResult.pluginJsonParsed!.name! }));
            }
            if (parseResult.pluginJsonParsed?.description && !formData.description) {
              setFormData(prev => ({ ...prev, description: parseResult.pluginJsonParsed!.description! }));
            }
          }
        }
        return updated;
      });
    }
  };

  const handlePublish = () => {
    const successFiles = uploadedFiles.filter(f => f.status === 'success');
    if (successFiles.length === 0) { toast.error('请先上传有效的插件 ZIP 文件'); return; }
    if (!formData.slug || !formData.name || !formData.version) { toast.error('请填写所有必填字段'); return; }
    if (!/^[a-z0-9-]+$/.test(formData.slug)) { toast.error('slug 仅支持小写字母/数字/连字符 -'); return; }
    if (existingSlugs.includes(formData.slug)) { toast.error('该 slug 已存在，请修改后重试'); return; }

    const successFile = uploadedFiles.find(f => f.status === 'success');
    const newPlugin: Plugin = {
      id: `plugin-${Date.now()}`,
      slug: formData.slug,
      name: formData.name,
      description: formData.description,
      version: formData.version,
      scope: formData.scope,
      groupIds: formData.scope === 'public' ? [] : formData.groupIds,
      uploadTime: new Date(),
      content: `# ${formData.name}\n\n${formData.description}`,
      versions: [formData.version],
      files: successFile?.files || [],
    };

    onConfirm(newPlugin);
    toast.success('插件发布成功');
    resetAll();
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[720px]" style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }} onPointerDownOutside={(e) => e.preventDefault()}>
        <DialogHeader>
          <DialogTitle>发布新插件</DialogTitle>
        </DialogHeader>

        <DialogBody className="flex-1 px-6">
          <div className="space-y-4">
            {/* 顶部提示：未上传文件时提示先上传 */}
            {uploadedFiles.length === 0 && (
              <Alert variant="warning">
                <CircleAlert />
                <AlertDescription>请先上传插件文件，然后再填写下方出现的插件信息。</AlertDescription>
              </Alert>
            )}

            {/* 文件上传区域 */}
            <div className="space-y-3">
              <MetaMedium as="label" tone="secondary">
                {uploadedFiles.length > 0 ? '文件' : '选择上传方式'}
              </MetaMedium>

              <FileReplaceHelper show={uploadedFiles.length > 0 && uploadedFiles[0].status === 'success'} variant="plugin-upload">
                已上传文件，如需重新上传请先删除
              </FileReplaceHelper>

              <UploadFileCard
                file={uploadedFiles.length > 0 ? uploadedFiles[0] : null}
                expanded={uploadedFiles.length > 0 ? expandedFile === uploadedFiles[0].name : false}
                onToggleExpand={() => {
                  if (uploadedFiles.length === 0) return;
                  const name = uploadedFiles[0].name;
                  setExpandedFile(expandedFile === name ? null : name);
                }}
                onZipUpload={handleZipUpload}
                onRemove={() => {
                  setUploadedFiles([]);
                  setExpandedFile(null);
                  setFormData(freshForm());
                }}
                variant="plugin"
              />

              {uploadedFiles.length === 0 && (
                <UploadRequirementsCard variant="plugin-upload" />
              )}
            </div>

            {/* 插件信息表单 - 上传成功后才显示 */}
            {hasSuccessfulUpload && (
              <div className="space-y-4">
                <div className="space-y-1.5">
                  <MetaMedium as="label" tone="secondary" htmlFor="p-slug">
                    唯一标识 (slug)<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                  </MetaMedium>
                  <Input
                    id="p-slug"
                    value={formData.slug}
                    onChange={(e) => setFormData({ ...formData, slug: e.target.value.toLowerCase().replace(/\s+/g, '-') })}
                    placeholder="e.g., my-plugin-1"
                  />
                  <HelperText>仅支持小写字母/数字/连字符 - 。企业内唯一，发布后不可修改。</HelperText>
                </div>

                <div className="space-y-1.5">
                  <MetaMedium as="label" tone="secondary" htmlFor="p-name">
                    显示名称<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                  </MetaMedium>
                  <Input
                    id="p-name"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="e.g., 我的自定义插件"
                  />
                </div>

                <div className="space-y-1.5">
                  <MetaMedium as="label" tone="secondary" htmlFor="p-desc">
                    描述
                  </MetaMedium>
                  <Textarea
                    id="p-desc"
                    value={formData.description}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    placeholder="插件的简要描述"
                    rows={2}
                  />
                </div>

                <div className="space-y-1.5">
                  <MetaMedium as="label" tone="secondary" htmlFor="p-version">
                    版本号<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                  </MetaMedium>
                  <Input
                    id="p-version"
                    value={formData.version}
                    onChange={(e) => setFormData({ ...formData, version: e.target.value })}
                    placeholder="e.g., 1.0.0"
                  />
                </div>

                <div className="space-y-1.5">
                  <MetaMedium as="label" tone="secondary">应用范围</MetaMedium>
                  {lockedScope ? (
                    <div className="flex items-center h-9 px-3 rounded-[4px] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] text-[var(--text-secondary)]">
                      <MetaMedium tone="secondary">{lockedScope.lockedGroupName}</MetaMedium>
                    </div>
                  ) : (
                    <div className="mt-1">
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
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </DialogBody>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)}>取消</Button>
          <Button
            variant="dialog-confirm"
            onClick={handlePublish}
            disabled={!hasSuccessfulUpload}
          >
            发布插件
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
