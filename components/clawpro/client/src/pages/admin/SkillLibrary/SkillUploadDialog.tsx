import { useState, useEffect } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogBody } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { CircleAlert, X, Check } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { StatusTag } from '@/components/ui/status-tag';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { Checkbox } from '@/components/ui/checkbox';
import { Switch } from '@/components/ui/switch';
import { MetaText, MetaMedium, HelperText, CardTitle } from '@/components/ui/Typography';
import { SecurityScanCard } from '@/components/SecurityScanCard';
import JSZip from 'jszip';
import { Skill, type SkillScope, type UploadedFile, type ScopeLockConfig } from './types';
import { DEFAULT_CATEGORIES, MOCK_GROUPS, MOCK_PROJECT_GROUPS } from './mockData';
import { ScopeSelect } from '@/components/ScopeSelect';
import { UploadRequirementsCard } from './UploadRequirementsCard';
import UploadFileCard from './UploadFileCard';
import FileReplaceHelper from './FileReplaceHelper';

interface SkillUploadDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (skill: Skill) => void;
  existingSlugs?: string[];
  defaultSecurityScan?: boolean;
  onDefaultSecurityScanChange?: (value: boolean) => void;
  securityServiceActive?: boolean;
  /** 当在「项目资产管理」内使用时，将应用范围锁定为指定组织（只读） */
  lockedScope?: ScopeLockConfig;
  /**
   * 发布成功后的 toast 文案。默认 `技能发布成功`（管控端行为不变）；
   * 用户端 SkillSquare 传入 `Skill 已提交，等待管理员审核`。
   */
  successMessage?: string;
  /**
   * 是否隐藏安全检测卡片里的「设置上传/更新时默认提交安全检测」勾选框。
   * 该开关属于全局默认设置，仅管控端有意义；员工端发布 Skill 时应传 true。
   * 默认 false，管控端行为完全不变。
   */
  hideDefaultSecuritySetting?: boolean;
}

// 解析 SKILL.md 文件内容
const parseSkillMd = (content: string): { name?: string; description?: string } | null => {
  const lines = content.split('\n').map(line => line.trim());
  
  // 检查第一行是否为 ---
  if (lines[0] !== '---') {
    return null;
  }

  const result: { name?: string; description?: string } = {};
  
  // 从第二行开始解析 name 和 description
  for (let i = 1; i < lines.length; i++) {
    const line = lines[i];
    
    if (line.startsWith('---')) {
      break;
    }
    
    if (line.startsWith('name:')) {
      result.name = line.substring(5).trim();
    } else if (line.startsWith('description:')) {
      result.description = line.substring(12).trim();
    }
  }

  return Object.keys(result).length > 0 ? result : null;
};

// 真实 ZIP 文件解析
// 可读取内容的文本文件扩展名
const TEXT_EXTENSIONS = ['.md', '.mdx', '.xml', '.json', '.txt', '.yaml', '.yml', '.toml', '.ini', '.cfg', '.conf', '.sh', '.bat', '.py', '.js', '.ts', '.css', '.html', '.htm', '.svg', '.env', '.gitignore', '.dockerfile'];

const isTextFile = (name: string) => {
  const lower = name.toLowerCase();
  if (!lower.includes('.') && !lower.includes('/')) return true; // Dockerfile, Makefile 等
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

    // 遍历 ZIP 中的所有文件
    loaded.forEach((relativePath, zipEntry) => {
      // 跳过文件夹
      if (zipEntry.dir) {
        return;
      }

      // 跳过系统文件（Mac 系统文件）
      if (relativePath.startsWith('__MACOSX/') || relativePath.endsWith('.DS_Store')) {
        return;
      }

      // 检查是否是 SKILL.md（不区分大小写）
      if (relativePath.toLowerCase().endsWith('skill.md')) {
        skillmdFound = true;
      }

      fileEntries.push({ relativePath, zipEntry });
    });

    // 异步读取所有文本文件的内容
    for (const { relativePath, zipEntry } of fileEntries) {
      const size = (zipEntry as any)._data ? (zipEntry as any)._data.uncompressedSize : 0;
      let content: string | undefined;

      // 对文本文件读取内容
      if (isTextFile(relativePath)) {
        try {
          content = await zipEntry.async('text');
        } catch {
          // 读取失败则不填充 content
        }
      }

      files.push({ name: relativePath, size, content });
    }

    // 排序文件列表，SKILL.md 放第一个
    files.sort((a, b) => {
      if (a.name.toLowerCase() === 'skill.md') return -1;
      if (b.name.toLowerCase() === 'skill.md') return 1;
      return a.name.localeCompare(b.name);
    });

    // 从已读取的文件中获取 SKILL.md 内容
    if (skillmdFound) {
      const skillmdFile = files.find(f => f.name.toLowerCase().endsWith('skill.md'));
      if (skillmdFile?.content) {
        skillmdContent = skillmdFile.content;
      }
    }

    const skillmdParsed = skillmdContent ? parseSkillMd(skillmdContent) : undefined;

    return {
      files,
      skillmdContent,
      skillmdParsed: skillmdParsed || undefined,
    };
  } catch (error) {
    return {
      files: [],
      error: `ZIP 文件解析失败: ${error instanceof Error ? error.message : '未知错误'}`,
    };
  }
};

export default function SkillUploadDialog({ open, onOpenChange, onConfirm, existingSlugs = [], defaultSecurityScan = false, onDefaultSecurityScanChange = () => {}, securityServiceActive = true, lockedScope, successMessage = '技能发布成功', hideDefaultSecuritySetting = false }: SkillUploadDialogProps) {
  const [uploadedFiles, setUploadedFiles] = useState<UploadedFile[]>([]);
  const [expandedFile, setExpandedFile] = useState<string | null>(null);

  // 当对话框关闭时，清空上传状态
  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      setUploadedFiles([]);
      setExpandedFile(null);
      setFormData({
        slug: '',
        name: '',
        description: '',
        version: '1.0.0',
        categories: [],
        scope: lockedScope ? 'private' : 'public',
        groupIds: lockedScope ? [lockedScope.lockedGroupId] : [],
      });
      setEnableSecurityScan(false);
    }
    onOpenChange(newOpen);
  };
  const [formData, setFormData] = useState({
    slug: '',
    name: '',
    description: '',
    version: '1.0.0',
    categories: [] as string[],
    scope: 'public' as SkillScope,
    groupIds: [] as string[],
  });
  const [enableSecurityScan, setEnableSecurityScan] = useState(false);

  // 锁定场景：强制应用范围为指定组织
  useEffect(() => {
    if (open && lockedScope) {
      setFormData(prev => ({ ...prev, scope: 'private', groupIds: [lockedScope.lockedGroupId] }));
    }
  }, [open, lockedScope]);

  const hasSuccessfulUpload = uploadedFiles.some(f => f.status === 'success');

  const handleFileSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    // 新上传时删除旧的
    setUploadedFiles([]);
    setExpandedFile(null);
    const files = event.target.files;
    if (!files) return;

    const newFiles: UploadedFile[] = [];

    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      const fileName = file.name;

      if (!fileName.endsWith('.zip')) {
        newFiles.push({
          name: fileName,
          size: file.size,
          status: 'error',
          error: '只支持 ZIP 文件',
        });
        continue;
      }

      // 创建解析中的文件项
      const uploadedFile: UploadedFile = {
        name: fileName,
        size: file.size,
        status: 'parsing',
      };

      newFiles.push(uploadedFile);
    }

    setUploadedFiles([...uploadedFiles, ...newFiles]);

    // 异步解析文件
    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      if (!file.name.endsWith('.zip')) continue;

      const parseResult = await parseZipFile(file);
      
      // 检查是否存在 SKILL.md
      const hasSKILLMd = parseResult.files.some(f => f.name.toLowerCase().endsWith('skill.md'));
      
      setUploadedFiles(prev => {
        const updated = [...prev];
        const fileIndex = updated.findIndex(f => f.name === file.name);
        
        if (fileIndex !== -1) {
          if (parseResult.error) {
            // ZIP 解析失败
            updated[fileIndex] = {
              name: file.name,
              size: file.size,
              status: 'error',
              error: parseResult.error,
            };
          } else if (!hasSKILLMd) {
            // 没有 SKILL.md
            updated[fileIndex] = {
              name: file.name,
              size: file.size,
              status: 'error',
              error: '不存在 SKILL.md 文件，请修改后重试',
              files: parseResult.files,
            };
          } else {
            // 解析成功
            updated[fileIndex] = {
              name: file.name,
              size: file.size,
              status: 'success',
              files: parseResult.files,
              skillmdContent: parseResult.skillmdContent,
              skillmdParsed: parseResult.skillmdParsed,
            };

            // 自动填充表单数据
            if (parseResult.skillmdParsed?.name && !formData.name) {
              setFormData(prev => ({ ...prev, name: parseResult.skillmdParsed!.name! }));
            }
            if (parseResult.skillmdParsed?.description && !formData.description) {
              setFormData(prev => ({ ...prev, description: parseResult.skillmdParsed!.description! }));
            }
          }
        }
        
        return updated;
      });
    }
  };

  const handleFolderSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = event.target.files;
    if (!files) return;

    // 新上传时删除旧的
    setUploadedFiles([]);
    setExpandedFile(null);

    const folderName = '文件夹上传';
    
    // 创建解析中的文件夹项
    setUploadedFiles([{
      name: folderName,
      size: 0,
      status: 'parsing',
    }]);

    // 异步处理文件夹上传
    setTimeout(async () => {
      const fileList: { name: string; size: number; content?: string }[] = [];
      let skillmdContent: string | undefined;
      let skillmdFound = false;

      for (let i = 0; i < files.length; i++) {
        const file = files[i];
        const relativePath = file.webkitRelativePath || file.name;
        let content: string | undefined;

        // 对文本文件读取内容
        if (isTextFile(relativePath)) {
          try {
            content = await file.text();
          } catch {
            // 读取失败则不填充 content
          }
        }

        fileList.push({
          name: relativePath,
          size: file.size,
          content,
        });

        // 检查是否是 SKILL.md
        if (relativePath.toLowerCase().endsWith('skill.md')) {
          const pathParts = relativePath.split('/');
          if (pathParts.length === 2 && pathParts[0] && pathParts[1].toLowerCase() === 'skill.md') {
            skillmdFound = true;
            skillmdContent = content || await file.text();
          }
        }
      }

      // 保留所有文件，但不显示根目录本身
      const displayFiles = fileList.filter(f => {
        const pathParts = f.name.split('/');
        return pathParts.length > 1; // 排除根目录
      }).map(f => {
        const pathParts = f.name.split('/');
        return {
          name: pathParts.slice(1).join('/'), // 移除根目录前缀
          size: f.size,
          content: f.content,
        };
      });

      const skillmdParsed = skillmdContent ? parseSkillMd(skillmdContent) : undefined;

      if (!skillmdFound) {
        setUploadedFiles(prev => prev.map(f => 
          f.name === folderName 
            ? {
                name: folderName,
                size: 0,
                status: 'error',
                error: '不存在 Skill.md 文件或者不在根目录下，请修改后重试',
                files: displayFiles,
              }
            : f
        ));
      } else {
        setUploadedFiles(prev => prev.map(f => 
          f.name === folderName 
            ? {
                name: folderName,
                size: 0,
                status: 'success',
                files: displayFiles,
                skillmdContent,
                skillmdParsed: skillmdParsed || undefined,
              }
            : f
        ));

        // 自动填充表单数据
        if (skillmdParsed?.name && !formData.name) {
          setFormData(prev => ({ ...prev, name: skillmdParsed.name! }));
        }
        if (skillmdParsed?.description && !formData.description) {
          setFormData(prev => ({ ...prev, description: skillmdParsed.description! }));
        }
      }
    }, 0);
  };

  const handlePublish = () => {
    const successFiles = uploadedFiles.filter(f => f.status === 'success');
    if (successFiles.length === 0) {
      toast.error('请先上传有效的 Skill ZIP 文件');
      return;
    }

    if (!formData.slug || !formData.name || !formData.version) {
      toast.error('请填写所有必填字段');
      return;
    }

    // 校验 slug 格式
    if (!/^[a-z0-9-]+$/.test(formData.slug)) {
      toast.error('slug 仅支持小写字母/数字/连字符 -');
      return;
    }

    // 校验 slug 是否重复
    if (existingSlugs.includes(formData.slug)) {
      toast.error('该 slug 已存在，请修改后重试');
      return;
    }

    const successFile = uploadedFiles.find(f => f.status === 'success');

    const newSkill: Skill = {
      id: `skill-${Date.now()}`,
      slug: formData.slug,
      name: formData.name,
      description: formData.description,
      version: formData.version,
      categories: formData.categories,
      scope: formData.scope,
      groupIds: formData.scope === 'public' ? [] : formData.groupIds,
      uploadTime: new Date(),
      content: successFile?.skillmdContent || `# ${formData.name}\n\n${formData.description}`,
      versions: [formData.version],
      files: successFile?.files || [],
      securityInfo: enableSecurityScan
        ? { overallStatus: 'scanning', engines: [] }
        : { overallStatus: 'not_scanned', engines: [] },
    };

    onConfirm(newSkill);

    // 显示成功提示（默认「技能发布成功」；调用方可通过 successMessage 覆盖）
    toast.success(successMessage);

    // 重置表单
    setUploadedFiles([]);
    setFormData({
      slug: '',
      name: '',
      description: '',
      version: '1.0.0',
      categories: [],
      scope: lockedScope ? 'private' : 'public',
      groupIds: lockedScope ? [lockedScope.lockedGroupId] : [],
    });
    onOpenChange(false);
  };

  const handleZipUpload = (files: FileList) => {
    const event = {
      target: { files },
    } as unknown as React.ChangeEvent<HTMLInputElement>;
    handleFileSelect(event);
  };

  const handleFolderUpload = (files: FileList) => {
    const event = {
      target: { files },
    } as unknown as React.ChangeEvent<HTMLInputElement>;
    handleFolderSelect(event);
  };

  return (
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent size="lg" className="max-h-[min(90vh,780px)] flex flex-col" onPointerDownOutside={(e) => e.preventDefault()}>
          <DialogHeader>
            <DialogTitle>发布新技能</DialogTitle>
          </DialogHeader>

          <DialogBody className="px-6">
            <div className="space-y-4">
              {/* 提示文字 - 只有在没有上传文件时显示 */}
              {uploadedFiles.length === 0 && (
                <Alert variant="warning">
                  <CircleAlert />
                  <AlertDescription>请先上传 Skill 文件，然后填写技能信息。</AlertDescription>
                </Alert>
              )}

              {/* 文件上传区域 */}
              <div className="space-y-3">
                <MetaMedium as="label" tone="secondary">
                  {uploadedFiles.length > 0 ? '文件' : '选择上传方式'}
                </MetaMedium>

                <FileReplaceHelper show={uploadedFiles.length > 0 && uploadedFiles[0].status === 'success'} variant="skill">
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
                  onFolderUpload={handleFolderUpload}
                  onRemove={() => {
                    setUploadedFiles([]);
                    setExpandedFile(null);
                    setFormData({
                      slug: '',
                      name: '',
                      description: '',
                      version: '1.0.0',
                      categories: [],
                      scope: lockedScope ? 'private' : 'public',
                      groupIds: lockedScope ? [lockedScope.lockedGroupId] : [],
                    });
                  }}
                />

                {uploadedFiles.length === 0 && (
                  <UploadRequirementsCard />
                )}
              </div>

              {/* 技能信息表单 - 只有在上传成功后才显示 */}
              {hasSuccessfulUpload && (
                <div className="space-y-4">
                  <div className="space-y-2">
                    <MetaMedium as="label" tone="secondary" htmlFor="slug">
                      唯一标识 (slug)<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                    </MetaMedium>
                    <Input
                      id="slug"
                      disabled={!hasSuccessfulUpload}
                      value={formData.slug}
                      onChange={(e) => setFormData({ ...formData, slug: e.target.value.toLowerCase().replace(/\s+/g, '-') })}
                      placeholder="e.g., doc-summarizer-1"
                    />
                    <HelperText>仅支持小写字母/数字/连字符 - 。企业内唯一，发布后不可修改。</HelperText>
                  </div>

                  <div className="space-y-2">
                    <MetaMedium as="label" tone="secondary" htmlFor="name">
                      显示名称<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                    </MetaMedium>
                    <Input
                      id="name"
                      disabled={!hasSuccessfulUpload}
                      value={formData.name}
                      onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      placeholder="e.g., 文档总结助手"
                    />
                  </div>

                  <div className="space-y-2">
                    <MetaMedium as="label" tone="secondary" htmlFor="description">
                      描述
                    </MetaMedium>
                    <Textarea
                      id="description"
                      disabled={!hasSuccessfulUpload}
                      value={formData.description}
                      onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                      placeholder="技能的简要描述"
                      rows={2}
                    />
                  </div>

                  <div className="space-y-2">
                    <MetaMedium as="label" tone="secondary" htmlFor="version">
                      版本号<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                    </MetaMedium>
                    <Input
                      id="version"
                      disabled={!hasSuccessfulUpload}
                      value={formData.version}
                      onChange={(e) => setFormData({ ...formData, version: e.target.value })}
                      placeholder="e.g., 1.0.0"
                    />
                  </div>

                  <div className="space-y-2">
                    <MetaMedium as="label" tone="secondary">分类</MetaMedium>
                    <div className="flex flex-wrap gap-2">
                      {DEFAULT_CATEGORIES.map(cat => {
                        const isSelected = formData.categories.includes(cat.id);
                        return (
                          <button
                            key={cat.id}
                            disabled={!hasSuccessfulUpload}
                            onClick={() => {
                              if (!hasSuccessfulUpload) return;
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
                            } ${!hasSuccessfulUpload ? 'cursor-not-allowed' : ''}`}
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
                        <div className="flex items-center h-9 px-3 rounded-[4px] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] text-[var(--text-secondary)]">
                          <MetaMedium tone="secondary">{lockedScope.lockedGroupName}</MetaMedium>
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
                    defaultSecurityScan={defaultSecurityScan}
                    onDefaultSecurityScanChange={onDefaultSecurityScanChange}
                    switchDisabled={!hasSuccessfulUpload || !securityServiceActive}
                    variant="border"
                    hideDefaultSetting={hideDefaultSecuritySetting}
                  />
                </div>
              )}
            </div>
          </DialogBody>

          <DialogFooter>
            <Button variant="outline" onClick={() => handleOpenChange(false)}>
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              onClick={handlePublish}
              disabled={!hasSuccessfulUpload || !formData.slug.trim() || !formData.name.trim() || !formData.version.trim()}
            >
              发布 Skill
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
  )
}