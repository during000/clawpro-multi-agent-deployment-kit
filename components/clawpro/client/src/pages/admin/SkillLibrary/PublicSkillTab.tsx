/**
 * 公共技能 Tab（单 Skill 列表）
 *
 * 作为「公共技能库」一级 Tab 下的二级 Tab 之一，由 PublicSkillLibraryTab 外壳挂载。
 * 内容来自设计走查后的最新版本（卡片：StatusTag Top 1/2/3、Typography 体系、
 * rounded-[4px]；详情页：版本/文件树/预览 三列布局）。
 */
import { useState, useMemo, useCallback, useRef, lazy, Suspense, useEffect } from 'react';
import { toast } from 'sonner';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { StatusTag } from '@/components/ui/status-tag';
import { Pagination } from '@/components/ui/pagination';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { BackButton } from '@/components/ui/back-button';
import { SegmentedTabs } from '@/components/ui/segmented-tabs';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  TenantPageTitle,
  TenantDocTitle,
  SectionTitle,
  CardTitle,
  BodyText,
  BodyMedium,
  MetaText,
  MetaMedium,
  MiniBodyText,
  CodeText,
  TinyText,
  HelperText,
} from '@/components/ui/Typography';
import {
  Search, Download, Star, Heart, ChevronRight,
  ChevronDown, ChevronRight as ChevronRightIcon, FileText, Folder, FolderOpen, RefreshCw, Package, PackagePlus, Eye, Code, Send, PackageX
} from 'lucide-react';
import { FavoriteButton } from '@/components/ui/favorite-button';
import { MoreActionsDropdown } from '@/components/ui/more-actions-dropdown';
import {
  PUBLIC_SKILLS, PUBLIC_SKILL_CATEGORIES, type PublicSkill, type FavoriteSkill, type PublicSkillFile
} from './publicSkillMockData';
import MDXRenderer from '@/components/MDXRenderer';
import AddToPackageDialog from './AddToPackageDialog';
import { FilterChipGroup } from '@/components/ui/filter-chip';
import { Empty, EmptyHeader, EmptyDescription, EmptyMedia } from '@/components/ui/empty';
import PublicBatchDistributeDialog from './PublicBatchDistributeDialog';
import PublicBatchDeleteDialog from './PublicBatchDeleteDialog';
import { MOCK_OPENCLAW_INSTANCES } from './mockData';
import {
  addDistributionRecord,
  createDistributionRecordId,
  getAllDistributionRecords,
  getCurrentSkillInstalledInstances,
  getInstancesWithSkillDistributionStatus,
  getSkillDistributionSummary,
  type CachedDistributionRecord,
  type SkillDistributionSummary,
  updateDistributionRecord,
} from './distributionCache';
import { type DistributionStatus, DISTRIBUTION_STATUS_MAP } from './types';

// 懒加载 react-syntax-highlighter 减少首屏包体积
const SyntaxHighlighter = lazy(() =>
  import('react-syntax-highlighter').then(mod => ({ default: mod.Light as any }))
) as any;
const _loadedLanguages = new Set<string>();
const registerLanguage = async (lang: string) => {
  if (_loadedLanguages.has(lang)) return;
  _loadedLanguages.add(lang);
  try {
    const mod = await import('react-syntax-highlighter');
    const Light = mod.Light as any;
    const langModules: Record<string, () => Promise<any>> = {
      xml: () => import('react-syntax-highlighter/dist/esm/languages/hljs/xml'),
      json: () => import('react-syntax-highlighter/dist/esm/languages/hljs/json'),
      yaml: () => import('react-syntax-highlighter/dist/esm/languages/hljs/yaml'),
      python: () => import('react-syntax-highlighter/dist/esm/languages/hljs/python'),
      javascript: () => import('react-syntax-highlighter/dist/esm/languages/hljs/javascript'),
      typescript: () => import('react-syntax-highlighter/dist/esm/languages/hljs/typescript'),
      bash: () => import('react-syntax-highlighter/dist/esm/languages/hljs/bash'),
      css: () => import('react-syntax-highlighter/dist/esm/languages/hljs/css'),
      ini: () => import('react-syntax-highlighter/dist/esm/languages/hljs/ini'),
      markdown: () => import('react-syntax-highlighter/dist/esm/languages/hljs/markdown'),
    };
    const loader = langModules[lang];
    if (loader) {
      const langMod = await loader();
      Light.registerLanguage(lang, langMod.default);
    }
  } catch { /* 静默降级 */ }
};

// hljs 亮色主题样式（与企业技能库保持一致）
const hljsStyle: Record<string, React.CSSProperties> = {
  'hljs': { display: 'block', overflowX: 'auto', padding: '1em', background: '#ffffff', color: '#383a42' },
  'hljs-comment': { color: '#a0a1a7', fontStyle: 'italic' },
  'hljs-quote': { color: '#a0a1a7', fontStyle: 'italic' },
  'hljs-keyword': { color: '#a626a4' },
  'hljs-selector-tag': { color: '#a626a4' },
  'hljs-addition': { color: '#50a14f' },
  'hljs-number': { color: '#986801' },
  'hljs-string': { color: '#50a14f' },
  'hljs-meta': { color: '#4078f2' },
  'hljs-literal': { color: '#0184bb' },
  'hljs-doctag': { color: '#a626a4' },
  'hljs-regexp': { color: '#50a14f' },
  'hljs-attr': { color: '#986801' },
  'hljs-attribute': { color: '#50a14f' },
  'hljs-builtin-name': { color: '#e45649' },
  'hljs-name': { color: '#e45649' },
  'hljs-section': { color: '#e45649' },
  'hljs-tag': { color: '#e45649' },
  'hljs-variable': { color: '#e45649' },
  'hljs-template-variable': { color: '#e45649' },
  'hljs-selector-id': { color: '#e45649' },
  'hljs-title': { color: '#4078f2' },
  'hljs-type': { color: '#4078f2' },
  'hljs-symbol': { color: '#4078f2' },
  'hljs-bullet': { color: '#4078f2' },
  'hljs-link': { color: '#4078f2' },
  'hljs-deletion': { color: '#e45649' },
  'hljs-emphasis': { fontStyle: 'italic' },
  'hljs-strong': { fontWeight: 'bold' },
};

function getLanguageFromFilename(filename: string): string {
  const ext = filename.split('.').pop()?.toLowerCase() || '';
  const map: Record<string, string> = {
    json: 'json', xml: 'xml', yaml: 'yaml', yml: 'yaml',
    py: 'python', js: 'javascript', jsx: 'javascript',
    ts: 'typescript', tsx: 'typescript',
    sh: 'bash', bash: 'bash', css: 'css',
    md: 'markdown', html: 'xml', htm: 'xml',
    ini: 'ini', cfg: 'ini', conf: 'ini',
  };
  return map[ext] || 'text';
}

// ─── 技能卡片 ─────────────────────────────────────────────────────────────────

interface SkillCardProps {
  skill: PublicSkill;
  rank: number;
  isFavorited: boolean;
  onFavorite: (skillId: string) => void;
  onAddToPackage: (skillId: string) => void;
  onDistribute: (skillId: string) => void;
  onUninstall: (skillId: string) => void;
  disableActions?: boolean;
  uninstallDisabled?: boolean;
  onClick: () => void;
}

function SkillCard({
  skill,
  rank,
  isFavorited,
  onFavorite,
  onAddToPackage,
  onDistribute,
  onUninstall,
  disableActions = false,
  uninstallDisabled = false,
  onClick,
}: SkillCardProps) {
  const formatCount = (n: number) => {
    if (n >= 10000) {
      const v = n / 10000;
      return `${parseFloat(v.toFixed(1))}万`;
    }
    if (n >= 1000) {
      const v = n / 1000;
      return `${parseFloat(v.toFixed(1))}千`;
    }
    return String(n);
  };

  const handleFavoriteClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    onFavorite(skill.id);
  };

  const handleAddToPackageClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    onAddToPackage(skill.id);
  };

  return (
    <div
      className="relative flex flex-col overflow-hidden rounded-[4px] border border-gray-200 bg-white cursor-pointer transition-all hover:border-[#355EF1] group"
      onClick={onClick}
    >
      <div className="p-4 pl-4 flex flex-col flex-1">
        {/* 技能名称 + Top 标签（前 3 名展示，标签贴右） */}
        <div className="flex items-center justify-between gap-2 mb-1 pl-3">
          <CardTitle as="h3" tone="primary" className="font-semibold group-hover:text-[var(--text-brand)] transition-colors leading-tight truncate min-w-0 flex-1">
            {skill.name}
          </CardTitle>
          {rank === 1 && (
            <StatusTag mode="fill" variant="gray" className="shrink-0 bg-[#F3E8FF] text-[#7E22CE]">Top 1</StatusTag>
          )}
          {rank === 2 && (
            <StatusTag mode="fill" variant="blue" className="shrink-0">Top 2</StatusTag>
          )}
          {rank === 3 && (
            <StatusTag mode="fill" variant="green" className="shrink-0">Top 3</StatusTag>
          )}
        </div>

        {/* 中文简介 - 固定两行高度 */}
        <MetaText as="p" className="line-clamp-2 leading-relaxed pl-3" style={{ minHeight: '2.5rem' }}>
          {skill.descriptionZh}
        </MetaText>

        {/* 统计数据 + 收藏按钮 - 常驻第三行 */}
        <div className="flex items-center justify-between mt-3 pl-3">
          <MetaText as="div" tone="weak" className="flex items-center gap-3">
            <span className="flex items-center gap-1">
              <Download className="w-3 h-3" />
              {formatCount(skill.downloads)}
            </span>
            <span className="flex items-center gap-1">
              <Star className="w-3 h-3" />
              {formatCount(skill.stars)}
            </span>
            <span className="font-mono">v{skill.version}</span>
          </MetaText>
          {/* 右下角操作区 */}
          <div className="flex items-center gap-1">
            <button
              onClick={handleAddToPackageClick}
              className="w-7 h-7 rounded-[4px] flex items-center justify-center text-[var(--text-weak)] transition-colors hover:text-[var(--text-brand)] hover:bg-[var(--bg-brand-selected)]"
              title="加入初始技能包"
              aria-label="加入初始技能包"
            >
              <PackagePlus className="w-3.5 h-3.5" />
            </button>
            <button
              onClick={handleFavoriteClick}
              className={`w-7 h-7 rounded-[4px] flex items-center justify-center transition-colors ${
                isFavorited
                  ? 'text-[var(--text-danger)] bg-red-50 hover:bg-red-100'
                  : 'text-[var(--text-weak)] hover:text-[var(--text-danger)] hover:bg-red-50'
              }`}
              title={isFavorited ? '取消收藏' : '添加到我的收藏'}
            >
              <Heart className={`w-3.5 h-3.5 ${isFavorited ? 'fill-current' : ''}`} />
            </button>
            <MoreActionsDropdown
              triggerType="icon"
              align="end"
              stopPropagation
              disabled={disableActions && uninstallDisabled}
              items={[
                {
                  label: '下发',
                  icon: Send,
                  onClick: () => onDistribute(skill.id),
                  disabled: disableActions,
                },
                {
                  label: '卸载',
                  icon: PackageX,
                  onClick: () => onUninstall(skill.id),
                  disabled: disableActions || uninstallDisabled,
                },
              ]}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── 文件树节点 ───────────────────────────────────────────────────────────────

interface FileTreeNodeProps {
  file: PublicSkillFile;
  depth: number;
  selectedFile: PublicSkillFile | null;
  onSelect: (file: PublicSkillFile) => void;
}

function FileTreeNode({ file, depth, selectedFile, onSelect }: FileTreeNodeProps) {
  const [expanded, setExpanded] = useState(depth === 0);

  if (file.type === 'folder') {
    return (
      <div>
        <button
          className="w-full flex items-center gap-1.5 h-8 px-2 hover:bg-[#f4f4f5] rounded-[4px] transition-colors"
          style={{ paddingLeft: `${8 + depth * 16}px` }}
          onClick={() => setExpanded(!expanded)}
        >
          {expanded ? <ChevronDown className="w-3.5 h-3.5 shrink-0 text-[var(--text-muted)]" /> : <ChevronRightIcon className="w-3.5 h-3.5 shrink-0 text-[var(--text-muted)]" />}
          {expanded ? <FolderOpen className="w-3.5 h-3.5 shrink-0 text-[var(--text-muted)]" /> : <Folder className="w-3.5 h-3.5 shrink-0 text-[var(--text-muted)]" />}
          <BodyText as="span" tone="emphasis" className="font-medium">{file.name}</BodyText>
        </button>
        {expanded && file.children?.map(child => (
          <FileTreeNode
            key={child.path}
            file={child}
            depth={depth + 1}
            selectedFile={selectedFile}
            onSelect={onSelect}
          />
        ))}
      </div>
    );
  }

  const isSelected = selectedFile?.path === file.path;
  return (
    <button
      className={`w-full flex items-center gap-1.5 h-8 px-2 rounded-[4px] transition-colors ${
        isSelected ? 'bg-[#f4f4f5]' : 'hover:bg-[#f4f4f5]'
      }`}
      style={{ paddingLeft: `${8 + depth * 16}px` }}
      onClick={() => onSelect(file)}
    >
      <FileText className="w-3.5 h-3.5 text-[var(--text-muted)] shrink-0" />
      <BodyText as="span" tone="emphasis" className={isSelected ? 'font-medium' : ''}>{file.name}</BodyText>
    </button>
  );
}

// ─── 技能详情页 ───────────────────────────────────────────────────────────────

interface SkillDetailViewProps {
  skill: PublicSkill;
  isFavorited: boolean;
  isInPackage: boolean;
  onFavorite: (skillId: string) => void;
  onAddToPackage: (skillId: string) => void;
  groups: Array<{ id: string; name: string; parentId?: string | null }>;
  onBack: () => void;
}

function SkillDetailView({ skill, isFavorited, isInPackage, onFavorite, onAddToPackage, groups, onBack }: SkillDetailViewProps) {
  const [selectedVersion, setSelectedVersion] = useState(skill.versions[0]);
  const [activeTab, setActiveTab] = useState<'files' | 'distribution' | 'uninstall'>('files');
  const [distributeDialogOpen, setDistributeDialogOpen] = useState(false);
  const [uninstallDialogOpen, setUninstallDialogOpen] = useState(false);
  const [distributionRecords, setDistributionRecords] = useState<CachedDistributionRecord[]>([]);
  const [activeDistributionId, setActiveDistributionId] = useState<string | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [statusFilter, setStatusFilter] = useState<'all' | DistributionStatus>('all');
  const [detailSearchQuery, setDetailSearchQuery] = useState('');

  // 剥离唯一顶层文件夹：如果 files 只有一个 folder，直接展示其 children
  const displayFiles = useMemo(() => {
    if (skill.files.length === 1 && skill.files[0].type === 'folder' && skill.files[0].children) {
      return skill.files[0].children;
    }
    return skill.files;
  }, [skill.files]);

  const [selectedFile, setSelectedFile] = useState<PublicSkillFile | null>(() => {
    // 默认选中 SKILL.md（在 displayFiles 中递归查找）
    const findSkillMd = (files: PublicSkillFile[]): PublicSkillFile | null => {
      for (const f of files) {
        if (f.name === 'SKILL.md') return f;
        if (f.children) {
          const found = findSkillMd(f.children);
          if (found) return found;
        }
      }
      return null;
    };
    return findSkillMd(displayFiles.length > 0 ? displayFiles : skill.files) || skill.files[0] || null;
  });
  const [mdPreviewMode, setMdPreviewMode] = useState<'source' | 'preview'>(
    () => selectedFile?.name.endsWith('.md') ? 'preview' : 'source'
  );
  const skillResourceId = getPublicSkillResourceId(skill.id);
  const formatCount = (n: number) => {
    if (n >= 10000) {
      const v = n / 10000;
      return `${parseFloat(v.toFixed(1))}万`;
    }
    if (n >= 1000) {
      const v = n / 1000;
      return `${parseFloat(v.toFixed(1))}千`;
    }
    return String(n);
  };

  const handleFavoriteClick = () => {
    onFavorite(skill.id);
  };

  const refreshRecords = useCallback(() => {
    setDistributionRecords(getAllDistributionRecords().filter((record) => record.skillId === skillResourceId));
  }, [skillResourceId]);

  useEffect(() => {
    refreshRecords();
    const handleCacheUpdated = () => refreshRecords();
    window.addEventListener('distribution-cache-updated', handleCacheUpdated);
    return () => window.removeEventListener('distribution-cache-updated', handleCacheUpdated);
  }, [refreshRecords]);

  const distributionOnlyRecords = useMemo(
    () => distributionRecords.filter((record) => (record.type || 'distribute') === 'distribute'),
    [distributionRecords],
  );
  const deleteOnlyRecords = useMemo(
    () => distributionRecords.filter((record) => record.type === 'delete'),
    [distributionRecords],
  );
  const hasInProgress = distributionRecords.some((record) => record.status === 'distributing' || record.status === 'deleting');
  const hasDeleting = distributionRecords.some((record) => record.status === 'deleting');
  const hasDistributing = distributionRecords.some((record) => record.status === 'distributing');

  const distributeInstances = useMemo(() => {
    return getInstancesWithSkillDistributionStatus(
      skillResourceId,
      skill.version,
      MOCK_OPENCLAW_INSTANCES,
    );
  }, [distributionRecords, skill.version, skillResourceId]);

  const distributedInstancesForUninstall = useMemo(() => {
    const installedInstances = getCurrentSkillInstalledInstances(
      skillResourceId,
      skill.version,
      MOCK_OPENCLAW_INSTANCES,
    );
    const instanceMap = new Map<string, any>();

    installedInstances.forEach((instance) => {
      const primaryGroupId = instance.groupIds?.[0];
      const groupName = primaryGroupId
        ? groups.find((group) => group.id === primaryGroupId)?.name
        : undefined;

      instanceMap.set(instance.id, {
        id: instance.id,
        name: instance.name,
        createdBy: instance.createdBy || 'admin',
        groupName: groupName || '全部用户',
        distributedVersion: instance.distributedVersion || skill.version,
        distributedTime: instance.createdAt,
        agentType: instance.agentType,
        agentVersion: instance.agentVersion,
        localProduct: instance.localProduct,
        deleteStatus: 'not_deleted' as const,
      });
    });

    distributionRecords
      .filter((record) => record.type === 'delete')
      .forEach((record) => {
        record.instances.forEach((instance) => {
          if (instance.distributionStatus === 'failed' && instanceMap.has(instance.id)) {
            const existing = instanceMap.get(instance.id);
            instanceMap.set(instance.id, {
              ...existing,
              deleteStatus: 'delete_failed' as const,
              deleteFailReason: instance.failReason,
            });
          }
        });
      });

    return Array.from(instanceMap.values());
  }, [distributionRecords, groups, skill.version, skillResourceId]);

  const activeDistribution = useMemo(
    () => distributionRecords.find((record) => record.id === activeDistributionId) ?? null,
    [activeDistributionId, distributionRecords],
  );

  const filteredInstances = useMemo(() => {
    if (!activeDistribution) return [];

    return activeDistribution.instances.filter((instance) => {
      const matchesStatus = statusFilter === 'all' || instance.distributionStatus === statusFilter;
      const searchLower = detailSearchQuery.trim().toLowerCase();
      const matchesSearch = !searchLower
        || instance.name.toLowerCase().includes(searchLower)
        || instance.id.toLowerCase().includes(searchLower);
      return matchesStatus && matchesSearch;
    });
  }, [activeDistribution, detailSearchQuery, statusFilter]);

  const handleDistributionStart = useCallback((selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId: skillResourceId,
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'distributing',
      operator: 'admin',
      instances: selectedInstancesData.map((instance) => ({
        id: instance.id,
        name: instance.name,
        createdBy: instance.createdBy || 'admin',
        agentType: instance.agentType,
        agentVersion: instance.agentVersion,
        localProduct: instance.localProduct,
        distributedVersion: skill.version,
        distributionStatus: 'distributing' as const,
      })),
    };
    addDistributionRecord(newRecord);
    setDistributeDialogOpen(false);
    toast.success('已开始下发流程');

    const totalCount = selectedInstanceIds.length;
    let completed = 0;
    const interval = setInterval(() => {
      completed += Math.floor(Math.random() * 3) + 1;
      if (completed >= totalCount) {
        completed = totalCount;
        clearInterval(interval);
        const failedCount = Math.floor(Math.random() * 2);
        const successCount = totalCount - failedCount;
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount,
          failedCount,
          inProgressCount: 0,
          status: failedCount === 0 ? 'success' : 'failed',
          instances: record.instances.map((instance, index) => ({
            ...instance,
            distributionStatus: index < successCount ? 'success' as const : 'failed' as const,
            failReason: index < successCount ? undefined : '命令下发失败',
          })),
        }));
        toast.success('下发完成');
      } else {
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount: completed,
          inProgressCount: totalCount - completed,
        }));
      }
    }, 800);
  }, [skill.version, skillResourceId]);

  const handleDeleteStart = useCallback((selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId: skillResourceId,
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'deleting',
      type: 'delete',
      operator: 'admin',
      instances: selectedInstancesData.map((instance) => ({
        id: instance.id,
        name: instance.name,
        createdBy: instance.createdBy || 'admin',
        agentType: instance.agentType,
        agentVersion: instance.agentVersion,
        localProduct: instance.localProduct,
        distributionStatus: 'distributing' as const,
      })),
    };
    addDistributionRecord(newRecord);
    setUninstallDialogOpen(false);
    toast.success('已开始卸载流程');

    const totalCount = selectedInstanceIds.length;
    let completed = 0;
    const interval = setInterval(() => {
      completed += Math.floor(Math.random() * 3) + 1;
      if (completed >= totalCount) {
        completed = totalCount;
        clearInterval(interval);
        const results = Array.from({ length: totalCount }, () => Math.random() < 0.9);
        const successCount = results.filter(Boolean).length;
        const failedCount = totalCount - successCount;
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount,
          failedCount,
          inProgressCount: 0,
          status: 'success',
          instances: record.instances.map((instance, index) => ({
            ...instance,
            distributionStatus: (results[index] ? 'success' : 'failed') as 'success' | 'failed',
            failReason: results[index] ? undefined : '实例离线',
          })),
        }));
        toast.success('卸载完成');
      } else {
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount: completed,
          inProgressCount: totalCount - completed,
        }));
      }
    }, 800);
  }, [skillResourceId]);

  return (
    <div className="space-y-4">
      {/* 顶部导航
          停服态下仍允许返回公共技能库：纯导航类操作。 */}
      <div data-billing-exempt>
        <BackButton onClick={onBack}>返回公共技能库</BackButton>
      </div>

      {/* 技能信息头部 */}
      <div className="bg-white rounded-[4px] border border-gray-200 p-5"
       >
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-0.5">
              <TenantDocTitle as="h1" tone="primary">{skill.name}</TenantDocTitle>
              <StatusTag mode="fill" variant="gray" className="font-mono">v{skill.version}</StatusTag>
            </div>
            <MetaText as="p" tone="weak" className="font-mono mb-2">slug：{skill.name}</MetaText>
            <BodyText as="p" tone="secondary" className="mb-3">{skill.descriptionZh}</BodyText>
            <MetaText as="div" tone="weak" className="flex items-center gap-4">
              <span className="flex items-center gap-1.5">
                <Download className="w-4 h-4" />
                {formatCount(skill.downloads)} 次下载
              </span>
              <span className="flex items-center gap-1.5">
                <Star className="w-4 h-4" />
                {formatCount(skill.stars)} 收藏
              </span>
            </MetaText>
          </div>
          <div className="flex items-center gap-2 relative">
            {/* 收藏按钮 */}
            <FavoriteButton
              isFavorited={isFavorited}
              onToggle={handleFavoriteClick}
            />
          </div>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as typeof activeTab)} className="w-full">
        <div className="flex items-center justify-between">
          {/* 文件列表 / 下发记录 / 卸载记录 二级 Tab
              停服态下仍允许切换 Tab 查看内容：纯导航/查看类操作。
              通过 data-billing-exempt 豁免；批量下发/批量卸载按钮不受此豁免影响，继续受停服约束。 */}
          <div data-billing-exempt>
            <SegmentedTabs
              tabs={[
                { id: 'files', label: '文件列表' },
                { id: 'distribution', label: '下发记录' },
                { id: 'uninstall', label: '卸载记录' },
              ]}
              active={activeTab}
              onChange={(value) => setActiveTab(value as typeof activeTab)}
              ariaLabel="公共技能详情 Tab 切换"
            />
          </div>
          {activeTab === 'distribution' && (
            <Button
              variant="claw-primary"
              size="claw"
              onClick={() => setDistributeDialogOpen(true)}
              disabled={hasInProgress}
            >
              {hasDistributing ? '下发中...' : '批量下发'}
            </Button>
          )}
          {activeTab === 'uninstall' && (
            <Button
              variant="claw-primary"
              size="claw"
              onClick={() => setUninstallDialogOpen(true)}
              disabled={hasInProgress || distributedInstancesForUninstall.length === 0}
              className={hasInProgress || distributedInstancesForUninstall.length === 0 ? 'opacity-50 cursor-not-allowed' : ''}
            >
              {hasDeleting ? '卸载中...' : '批量卸载'}
            </Button>
          )}
        </div>
        <TabsList className="hidden">
          <TabsTrigger value="files" />
          <TabsTrigger value="distribution" />
          <TabsTrigger value="uninstall" />
        </TabsList>

        <TabsContent value="files" className="mt-4 p-0">
          {/* 版本列表 + 文件列表 + 预览/源码 整体
              停服态下仍允许查看版本/文件/代码：纯查看类操作。 */}
          <div className="flex h-[47rem] border border-gray-200 rounded-[4px] overflow-hidden bg-white" data-billing-exempt>
            <div className="w-[14%] min-w-[120px] border-r border-gray-200 flex flex-col">
              <div className="h-12 px-3 border-b border-gray-200 flex items-center">
                <BodyMedium as="p" tone="emphasis">版本</BodyMedium>
              </div>
              <div className="flex-1 overflow-y-auto">
                {skill.versions.map((v) => (
                  <button
                    key={v.version}
                    onClick={() => setSelectedVersion(v)}
                    className={`w-full text-left px-3 py-2.5 border-b border-[#f4f4f5] transition-colors ${
                      selectedVersion.version === v.version ? 'bg-[#f4f4f5]' : 'hover:bg-[#f4f4f5] cursor-pointer'
                    }`}
                  >
                    <div className="flex items-center gap-1.5">
                      <BodyMedium as="span" tone="emphasis" className="font-semibold">{v.version}</BodyMedium>
                      {v.isLatest && (
                        <span className="inline-flex h-[18px] items-center justify-center rounded-[2px] border border-[#1447E6] px-1 leading-none">
                          <TinyText as="span" tone="brand">New</TinyText>
                        </span>
                      )}
                    </div>
                    <MetaText as="p" tone="weak" className="mt-0.5">{v.date.slice(0, 10)}</MetaText>
                  </button>
                ))}
              </div>
            </div>
            <div className="w-[22%] min-w-[160px] border-r border-gray-200 flex flex-col">
              <div className="h-12 px-3 border-b border-gray-200 flex items-center">
                <BodyMedium as="p" tone="emphasis">{selectedVersion.version}</BodyMedium>
              </div>
              <div className="flex-1 overflow-y-auto px-3 py-2">
                {displayFiles.map(file => (
                  <FileTreeNode
                    key={file.path}
                    file={file}
                    depth={0}
                    selectedFile={selectedFile}
                    onSelect={(file) => {
                      setSelectedFile(file);
                      setMdPreviewMode(file.name.endsWith('.md') ? 'preview' : 'source');
                    }}
                  />
                ))}
              </div>
            </div>
            <div className="flex-1 flex flex-col bg-white">
              {selectedFile ? (
                <>
                  <div className="h-12 px-3 border-b border-gray-200 flex items-center justify-between">
                    <BodyMedium as="p" tone="emphasis">{selectedFile.name}</BodyMedium>
                    <div className="flex items-center gap-0.5 bg-gray-200/60 rounded p-0.5">
                      <button onClick={() => setMdPreviewMode('preview')} className={`flex items-center gap-1 px-2 py-1 rounded transition-colors ${mdPreviewMode === 'preview' ? 'bg-white shadow-sm' : ''}`}>
                        <Eye className="w-3 h-3" />
                        <MiniBodyText as="span" tone={mdPreviewMode === 'preview' ? 'primary' : 'muted'} className={mdPreviewMode === 'preview' ? 'font-medium' : ''}>预览</MiniBodyText>
                      </button>
                      <button onClick={() => setMdPreviewMode('source')} className={`flex items-center gap-1 px-2 py-1 rounded transition-colors ${mdPreviewMode === 'source' ? 'bg-white shadow-sm' : ''}`}>
                        <Code className="w-3 h-3" />
                        <MiniBodyText as="span" tone={mdPreviewMode === 'source' ? 'primary' : 'muted'} className={mdPreviewMode === 'source' ? 'font-medium' : ''}>源码</MiniBodyText>
                      </button>
                    </div>
                  </div>
                  <div className="flex-1 overflow-y-auto">
                    {(() => {
                      const content = selectedFile.content || '';
                      if (!content) return <div className="flex items-center justify-center h-full"><BodyText as="p" tone="weak">文件内容暂无</BodyText></div>;
                      if (mdPreviewMode === 'source') {
                        const lang = getLanguageFromFilename(selectedFile.name);
                        registerLanguage(lang);
                        return <Suspense fallback={<CodeText as="pre" tone="secondary" className="block overflow-x-auto whitespace-pre-wrap break-words leading-5 bg-gray-50 p-3 m-0">{content}</CodeText>}><SyntaxHighlighter language={lang} style={hljsStyle} showLineNumbers lineNumberStyle={{ color: '#b0b0b0', fontSize: '11px', minWidth: '2.5em', paddingRight: '1em', userSelect: 'none' }} customStyle={{ margin: 0, padding: '12px 0', fontSize: '12px', lineHeight: '1.6', background: '#ffffff', borderRadius: 0 }} wrapLongLines>{content}</SyntaxHighlighter></Suspense>;
                      }
                      if (selectedFile.name.toLowerCase().endsWith('.md') || selectedFile.name.toLowerCase().endsWith('.mdx')) {
                        return <div className="p-4"><MDXRenderer content={content} /></div>;
                      }
                      const previewLang = getLanguageFromFilename(selectedFile.name);
                      registerLanguage(previewLang);
                      return <Suspense fallback={<CodeText as="pre" tone="secondary" className="block overflow-x-auto whitespace-pre-wrap break-words leading-5 bg-gray-50 p-3 m-0">{content}</CodeText>}><SyntaxHighlighter language={previewLang} style={hljsStyle} showLineNumbers lineNumberStyle={{ color: '#b0b0b0', fontSize: '11px', minWidth: '2.5em', paddingRight: '1em', userSelect: 'none' }} customStyle={{ margin: 0, padding: '12px 0', fontSize: '12px', lineHeight: '1.6', background: '#ffffff', borderRadius: 0 }} wrapLongLines>{content}</SyntaxHighlighter></Suspense>;
                    })()}
                  </div>
                </>
              ) : (
                <div className="flex items-center justify-center h-full">
                  <BodyText as="p" tone="muted">选择一个文件查看内容</BodyText>
                </div>
              )}
            </div>
          </div>
        </TabsContent>

        <TabsContent value="distribution" className="mt-4 p-0">
          <div className="space-y-3">
            {distributionOnlyRecords.length === 0 ? (
              <div className="flex flex-col items-center justify-center rounded-[4px] border border-gray-200 bg-white py-12 text-center">
                <HelperText as="p">还没有下发记录</HelperText>
              </div>
            ) : (
              distributionOnlyRecords.map((record, idx) => {
                const progress = record.totalCount > 0 ? Math.round((record.successCount / record.totalCount) * 100) : 0;
                return (
                  <div key={record.id} className="rounded-[4px] border border-gray-200 bg-white p-4">
                    <div className="mb-3 flex items-start justify-between">
                      <BodyMedium as="p" tone="primary" className="font-semibold">#{idx + 1} · v{skill.version} {new Date(record.timestamp).toLocaleString('zh-CN')}</BodyMedium>
                      <div className="flex items-center gap-2">
                        <span className={`inline-block rounded px-3 py-1 text-xs font-medium ${record.status === 'distributing' ? 'bg-blue-50 text-blue-700' : record.successCount === record.totalCount ? 'bg-green-50 text-green-700' : 'bg-yellow-50 text-yellow-700'}`}>
                          {record.status === 'distributing' ? `下发中 ${progress}%` : `下发完成，${record.successCount}个成功，${record.failedCount}个失败`}
                        </span>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => {
                            setActiveDistributionId(record.id);
                            setStatusFilter('all');
                            setDetailSearchQuery('');
                            setDetailsOpen(true);
                          }}
                          className="h-auto px-2 py-1 text-[var(--text-brand)] hover:text-[var(--text-brand)]"
                        >
                          查看详情
                        </Button>
                      </div>
                    </div>
                    {record.status === 'distributing' && (
                      <div className="w-full rounded-full bg-gray-200 h-2">
                        <div className="h-2 rounded-full bg-blue-600 transition-all duration-300" style={{ width: `${progress}%` }} />
                      </div>
                    )}
                  </div>
                );
              })
            )}
          </div>
        </TabsContent>

        <TabsContent value="uninstall" className="mt-4 p-0">
          <div className="space-y-3">
            {deleteOnlyRecords.length === 0 ? (
              <div className="flex flex-col items-center justify-center rounded-[4px] border border-gray-200 bg-white py-12 text-center">
                <HelperText as="p">还没有卸载记录</HelperText>
              </div>
            ) : (
              deleteOnlyRecords.map((record, idx) => {
                const progress = record.totalCount > 0 ? Math.round((record.successCount / record.totalCount) * 100) : 0;
                const isInProgress = record.status === 'deleting' || record.status === 'distributing';
                return (
                  <div key={record.id} className="rounded-[4px] border border-gray-200 bg-white p-4">
                    <div className="mb-3 flex items-start justify-between">
                      <BodyMedium as="p" tone="primary" className="font-semibold">#{idx + 1} · {new Date(record.timestamp).toLocaleString('zh-CN')}</BodyMedium>
                      <div className="flex items-center gap-2">
                        <span className={`inline-block rounded px-3 py-1 text-xs font-medium ${isInProgress ? 'bg-red-100 text-red-700' : record.failedCount === 0 ? 'bg-green-50 text-green-700' : 'bg-yellow-50 text-yellow-700'}`}>
                          {isInProgress ? `卸载中 ${progress}%` : `卸载完成，${record.successCount}个成功，${record.failedCount}个失败`}
                        </span>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => {
                            setActiveDistributionId(record.id);
                            setStatusFilter('all');
                            setDetailSearchQuery('');
                            setDetailsOpen(true);
                          }}
                          className="h-auto px-2 py-1 text-[var(--text-brand)] hover:text-[var(--text-brand)]"
                        >
                          查看详情
                        </Button>
                      </div>
                    </div>
                    {isInProgress && (
                      <div className="w-full rounded-full bg-gray-200 h-2">
                        <div className="h-2 rounded-full bg-red-500 transition-all duration-300" style={{ width: `${progress}%` }} />
                      </div>
                    )}
                  </div>
                );
              })
            )}
          </div>
        </TabsContent>
      </Tabs>

      <PublicBatchDistributeDialog
        open={distributeDialogOpen}
        onOpenChange={setDistributeDialogOpen}
        skillId={skillResourceId}
        skillName={skill.nameZh || skill.name}
        skillVersion={skill.version}
        instances={distributeInstances as any}
        groups={groups}
        onDistributionStart={handleDistributionStart}
        title="批量下发 Skill"
        descriptionNode={(
          <div className="space-y-1">
            <BodyText as="p" tone="muted">
              将 <BodyMedium as="span" tone="primary" className="font-semibold">
                {skill.nameZh || skill.name}
              </BodyMedium> 部署至所选实例。
            </BodyText>
            <BodyText as="p" tone="muted">
              仅支持向 <BodyMedium as="span" tone="secondary">运行中</BodyMedium> 的实例下发技能; 已下发当前版本技能的实例默认不展示，可选择后再次下发。
            </BodyText>
          </div>
        )}
      />
      <PublicBatchDeleteDialog
        open={uninstallDialogOpen}
        onOpenChange={setUninstallDialogOpen}
        skillName={skill.nameZh || skill.name}
        skillVersion={skill.version}
        distributedInstances={distributedInstancesForUninstall}
        groups={groups}
        onDeleteStart={handleDeleteStart}
        resourceLabel="Skill"
        warningText={'通过下发按钮安装的技能可支持移出（包括用户下发和管理端下发）。卸载成功后，该技能在对应实例上恢复为"未下发"状态。'}
        emptyText="暂无已下发的实例"
      />
      <Dialog open={detailsOpen} onOpenChange={setDetailsOpen}>
        <DialogContent className="flex w-[700px] max-h-[min(90vh,780px)] !max-w-[700px] flex-col overflow-hidden">
          <DialogHeader className="-mx-6 shrink-0 px-6 pt-6 pb-3">
            <DialogTitle>{activeDistribution?.type === 'delete' ? '卸载详情' : '下发详情'}</DialogTitle>
          </DialogHeader>
          {activeDistribution && (
            <div className="-mx-6 flex-1 min-h-0 overflow-y-auto px-6 py-0 pb-6">
              <div className="flex min-h-0 flex-col space-y-4 overflow-hidden">
                <div className="flex items-center gap-2">
                  <div className="relative flex-1">
                    <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                    <Input
                      placeholder="搜索实例名称/ID..."
                      value={detailSearchQuery}
                      onChange={(e) => setDetailSearchQuery(e.target.value)}
                      className="h-9 pl-10"
                    />
                  </div>
                  <Select value={statusFilter} onValueChange={(value: any) => setStatusFilter(value)}>
                    <SelectTrigger className="w-28">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">全部</SelectItem>
                      <SelectItem value="success">成功</SelectItem>
                      <SelectItem value="failed">失败</SelectItem>
                      <SelectItem value="distributing">{activeDistribution.type === 'delete' ? '卸载中' : '下发中'}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex-1 min-h-0 overflow-y-auto rounded-[4px] border border-gray-200">
                <Table density="compact" containerClassName="max-h-full overflow-y-auto">
                  <TableHeader>
                    <TableRow>
                      <TableHead>实例名称</TableHead>
                      <TableHead className="min-w-[140px]">实例ID</TableHead>
                      <TableHead>状态</TableHead>
                      <TableHead>失败原因</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredInstances.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={4} className="text-center">
                          <BodyText as="span" tone="muted">没有符合条件的记录</BodyText>
                        </TableCell>
                      </TableRow>
                    ) : (
                      filteredInstances.map((instance) => (
                        <TableRow key={instance.id}>
                          <TableCell>{instance.name}</TableCell>
                          <TableCell className="font-mono text-gray-500">{instance.id}</TableCell>
                          <TableCell>
                            <span className={`inline-block rounded px-2 py-1 text-xs font-medium ${DISTRIBUTION_STATUS_MAP[instance.distributionStatus]?.color || 'bg-gray-50 text-gray-500'}`}>
                              {DISTRIBUTION_STATUS_MAP[instance.distributionStatus]?.label || '未下发'}
                            </span>
                          </TableCell>
                          <TableCell className="text-gray-500">{instance.failReason || '-'}</TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
                </div>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ─── 主组件 ───────────────────────────────────────────────────────────────────

interface PublicSkillTabProps {
  packages: Array<{ id: string; name: string; isActive: boolean; scopeType?: 'all-users' | 'groups'; scopeLabel?: string; groupIds?: string[] }>;
  groups: Array<{ id: string; name: string; parentId?: string | null }>;
  onAddSkillToPackage: (skillId: string, packageId: string) => void;
}

const PAGE_SIZE = 24;
const PUBLIC_SKILL_RESOURCE_PREFIX = 'public-skill';

const getPublicSkillResourceId = (skillId: string) => `${PUBLIC_SKILL_RESOURCE_PREFIX}:${skillId}`;

export default function PublicSkillTab({
  packages,
  groups,
  onAddSkillToPackage,
}: PublicSkillTabProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [activeCategory, setActiveCategory] = useState('featured');
  const [favorites, setFavorites] = useState<FavoriteSkill[]>([]);
  const [inPackageSkills, setInPackageSkills] = useState<Set<string>>(new Set());
  const [addedPackageIdsBySkill, setAddedPackageIdsBySkill] = useState<Record<string, string[]>>({});
  const [selectedSkillId, setSelectedSkillId] = useState<string | null>(null);
  const [addToPackageSkillId, setAddToPackageSkillId] = useState<string | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [distributeDialogOpen, setDistributeDialogOpen] = useState(false);
  const [distributeSkillId, setDistributeSkillId] = useState<string | null>(null);
  const [uninstallDialogOpen, setUninstallDialogOpen] = useState(false);
  const [uninstallSkillId, setUninstallSkillId] = useState<string | null>(null);
  const [distributionSummaries, setDistributionSummaries] = useState<Record<string, SkillDistributionSummary>>({});

  const refreshDistributionSummaries = useCallback(() => {
    const next: Record<string, SkillDistributionSummary> = {};
    PUBLIC_SKILLS.forEach((skill) => {
      const resourceId = getPublicSkillResourceId(skill.id);
      const summary = getSkillDistributionSummary(resourceId);
      if (summary) next[resourceId] = summary;
    });
    setDistributionSummaries(next);
  }, []);

  useEffect(() => {
    refreshDistributionSummaries();
    const handleCacheUpdated = () => refreshDistributionSummaries();
    window.addEventListener('distribution-cache-updated', handleCacheUpdated);
    return () => window.removeEventListener('distribution-cache-updated', handleCacheUpdated);
  }, [refreshDistributionSummaries]);

  // 精选 Top 50：按下载量+收藏量综合排序
  const featuredSkills = useMemo(() => {
    return [...PUBLIC_SKILLS]
      .sort((a, b) => (b.downloads + b.stars) - (a.downloads + a.stars))
      .slice(0, 50);
  }, []);

  // 过滤技能
  const filteredSkills = useMemo(() => {
    let list: PublicSkill[] = [];

    if (activeCategory === 'all') {
      list = [...PUBLIC_SKILLS];
    } else if (activeCategory === 'featured') {
      list = featuredSkills;
    } else if (activeCategory === 'favorites') {
      const favIds = new Set(favorites.map(f => f.skillId));
      list = PUBLIC_SKILLS.filter(s => favIds.has(s.id));
    } else {
      list = PUBLIC_SKILLS.filter(s => s.category === activeCategory);
    }

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      list = list.filter(s =>
        s.name.toLowerCase().includes(q) ||
        s.nameZh.includes(q) ||
        s.descriptionZh.includes(q)
      );
    }

    return list;
  }, [activeCategory, searchQuery, favorites, featuredSkills]);

  // 收藏操作
  const handleFavorite = useCallback((skillId: string) => {
    setFavorites(prev => {
      const exists = prev.find(f => f.skillId === skillId);
      if (exists) {
        return prev.filter(f => f.skillId !== skillId);
      }
      return [...prev, { skillId, tags: [], addedAt: new Date() }];
    });
  }, []);

  // 加入初始技能包
  const handleAddToPackage = useCallback((skillId: string) => {
    setAddToPackageSkillId(skillId);
  }, []);

  const handleDistribute = useCallback((skillId: string) => {
    setDistributeSkillId(skillId);
    setDistributeDialogOpen(true);
  }, []);

  const handleUninstall = useCallback((skillId: string) => {
    setUninstallSkillId(skillId);
    setUninstallDialogOpen(true);
  }, []);

  const handlePackageSelected = useCallback((packageIds: string[]) => {
    if (addToPackageSkillId && packageIds.length > 0) {
      packageIds.forEach((packageId) => onAddSkillToPackage(addToPackageSkillId, packageId));
      setInPackageSkills(prev => { const next = new Set(prev); next.add(addToPackageSkillId); return next; });
      setAddedPackageIdsBySkill((prev) => {
        const existed = prev[addToPackageSkillId] ?? [];
        return {
          ...prev,
          [addToPackageSkillId]: Array.from(new Set([...existed, ...packageIds])),
        };
      });
    }
    setAddToPackageSkillId(null);
  }, [addToPackageSkillId, onAddSkillToPackage]);

  const isFavorited = useCallback(
    (skillId: string) => favorites.some(f => f.skillId === skillId),
    [favorites]
  );

  const getDistributionSummary = useCallback(
    (skillId: string) => distributionSummaries[getPublicSkillResourceId(skillId)] ?? null,
    [distributionSummaries],
  );

  const isDistributionInProgress = useCallback(
    (skillId: string) => getDistributionSummary(skillId)?.hasInProgress ?? false,
    [getDistributionSummary],
  );

  const getUninstallableCount = useCallback((skillId: string) => {
    const skill = PUBLIC_SKILLS.find((item) => item.id === skillId);
    if (!skill) return 0;
    return getCurrentSkillInstalledInstances(
      getPublicSkillResourceId(skillId),
      skill.version,
      MOCK_OPENCLAW_INSTANCES,
    ).length;
  }, []);

  const selectedDistributeSkill = useMemo(
    () => (distributeSkillId ? PUBLIC_SKILLS.find((skill) => skill.id === distributeSkillId) ?? null : null),
    [distributeSkillId],
  );

  const distributeInstances = useMemo(() => {
    if (!selectedDistributeSkill) return MOCK_OPENCLAW_INSTANCES;
    return getInstancesWithSkillDistributionStatus(
      getPublicSkillResourceId(selectedDistributeSkill.id),
      selectedDistributeSkill.version,
      MOCK_OPENCLAW_INSTANCES,
    );
  }, [selectedDistributeSkill, distributionSummaries]);

  const distributedInstancesForUninstall = useMemo(() => {
    if (!uninstallSkillId) return [];
    const skill = PUBLIC_SKILLS.find((item) => item.id === uninstallSkillId);
    if (!skill) return [];

    const installedInstances = getCurrentSkillInstalledInstances(
      getPublicSkillResourceId(uninstallSkillId),
      skill.version,
      MOCK_OPENCLAW_INSTANCES,
    );
    const instanceMap = new Map<string, any>();

    installedInstances.forEach((instance) => {
      const primaryGroupId = instance.groupIds?.[0];
      const groupName = primaryGroupId
        ? groups.find((group) => group.id === primaryGroupId)?.name
        : undefined;

      instanceMap.set(instance.id, {
        id: instance.id,
        name: instance.name,
        createdBy: instance.createdBy || 'admin',
        groupName: groupName || '全部用户',
        distributedVersion: instance.distributedVersion || skill.version,
        distributedTime: instance.createdAt,
        agentType: instance.agentType,
        agentVersion: instance.agentVersion,
        localProduct: instance.localProduct,
        deleteStatus: 'not_deleted' as const,
      });
    });

    getAllDistributionRecords()
      .filter((record) => record.skillId === getPublicSkillResourceId(uninstallSkillId) && (record.type || 'distribute') === 'delete')
      .forEach((record) => {
        record.instances.forEach((instance) => {
          if (instance.distributionStatus === 'failed' && instanceMap.has(instance.id)) {
            const existing = instanceMap.get(instance.id);
            instanceMap.set(instance.id, {
              ...existing,
              deleteStatus: 'delete_failed' as const,
              deleteFailReason: instance.failReason,
            });
          }
        });
      });

    return Array.from(instanceMap.values());
  }, [groups, uninstallSkillId, distributionSummaries]);

  const handleDistributeStart = useCallback((selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    if (!selectedDistributeSkill) return;

    const resourceId = getPublicSkillResourceId(selectedDistributeSkill.id);
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId: resourceId,
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'distributing',
      operator: 'admin',
      instances: selectedInstancesData.map((instance) => ({
        id: instance.id,
        name: instance.name,
        createdBy: instance.createdBy || 'admin',
        agentType: instance.agentType,
        agentVersion: instance.agentVersion,
        localProduct: instance.localProduct,
        distributedVersion: selectedDistributeSkill.version,
        distributionStatus: 'distributing' as const,
      })),
    };

    addDistributionRecord(newRecord);
    setDistributeDialogOpen(false);
    toast.success('已开始下发流程');

    const totalCount = selectedInstanceIds.length;
    let completed = 0;
    const interval = setInterval(() => {
      completed += Math.floor(Math.random() * 3) + 1;
      if (completed >= totalCount) {
        completed = totalCount;
        clearInterval(interval);
        const failedCount = Math.floor(Math.random() * 2);
        const successCount = totalCount - failedCount;
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount,
          failedCount,
          inProgressCount: 0,
          status: failedCount === 0 ? 'success' : 'failed',
          instances: record.instances.map((instance, index) => ({
            ...instance,
            distributionStatus: index < successCount ? 'success' as const : 'failed' as const,
            failReason: index < successCount ? undefined : '命令下发失败',
          })),
        }));
        toast.success('下发完成');
      } else {
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount: completed,
          inProgressCount: totalCount - completed,
        }));
      }
    }, 800);
  }, [selectedDistributeSkill]);

  const handleUninstallStart = useCallback((selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    if (!uninstallSkillId) return;

    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId: getPublicSkillResourceId(uninstallSkillId),
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'deleting',
      type: 'delete',
      operator: 'admin',
      instances: selectedInstancesData.map((instance) => ({
        id: instance.id,
        name: instance.name,
        createdBy: instance.createdBy || 'admin',
        agentType: instance.agentType,
        agentVersion: instance.agentVersion,
        localProduct: instance.localProduct,
        distributionStatus: 'distributing' as const,
      })),
    };

    addDistributionRecord(newRecord);
    setUninstallDialogOpen(false);
    toast.success('已开始卸载流程');

    const totalCount = selectedInstanceIds.length;
    let completed = 0;
    const interval = setInterval(() => {
      completed += Math.floor(Math.random() * 3) + 1;
      if (completed >= totalCount) {
        completed = totalCount;
        clearInterval(interval);
        const results = Array.from({ length: totalCount }, () => Math.random() < 0.9);
        const successCount = results.filter(Boolean).length;
        const failedCount = totalCount - successCount;
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount,
          failedCount,
          inProgressCount: 0,
          status: 'success',
          instances: record.instances.map((instance, index) => ({
            ...instance,
            distributionStatus: (results[index] ? 'success' : 'failed') as 'success' | 'failed',
            failReason: results[index] ? undefined : '实例离线',
          })),
        }));
        toast.success('卸载完成');
      } else {
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount: completed,
          inProgressCount: totalCount - completed,
        }));
      }
    }, 800);
  }, [uninstallSkillId]);

  // 分页计算
  const pagedSkills = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filteredSkills.slice(start, start + PAGE_SIZE);
  }, [filteredSkills, currentPage]);

  // 切换分类或搜索时重置到第一页
  const handleCategoryChange = useCallback((catId: string) => {
    setActiveCategory(catId);
    setCurrentPage(1);
  }, []);
  const handleSearchChange = useCallback((q: string) => {
    setSearchQuery(q);
    setCurrentPage(1);
  }, []);

  // 如果选中了技能，显示详情页
  if (selectedSkillId) {
    const skill = PUBLIC_SKILLS.find(s => s.id === selectedSkillId);
    if (skill) {
      return (
        <>
          <SkillDetailView
            skill={skill}
            isFavorited={isFavorited(skill.id)}
            isInPackage={inPackageSkills.has(skill.id)}
            onFavorite={handleFavorite}
            onAddToPackage={handleAddToPackage}
            groups={groups}
            onBack={() => setSelectedSkillId(null)}
          />
          <AddToPackageDialog
            open={!!addToPackageSkillId}
            itemName={PUBLIC_SKILLS.find(s => s.id === addToPackageSkillId)?.nameZh || ''}
            itemType="skill"
            packages={packages}
            groups={groups}
            addedPackageIds={addToPackageSkillId ? addedPackageIdsBySkill[addToPackageSkillId] ?? [] : []}
            onConfirm={handlePackageSelected}
            onCancel={() => setAddToPackageSkillId(null)}
          />
        </>
      );
    }
  }

  return (
    <>
      <div className="space-y-4">
        {/* 搜索框 + 刷新按钮
            停服态下仍允许搜索与刷新：纯导航/查看类操作。
            通过 data-billing-exempt 豁免 AdminDisabledOverlay 的全局灰化与点击拦截。 */}
        <div className="flex items-center gap-2" data-billing-exempt>
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)]" />
            <Input
              placeholder="搜索技能名称或关键词..."
              value={searchQuery}
              onChange={e => handleSearchChange(e.target.value)}
              className="pl-9 bg-white"
            />
          </div>
          <Button
            variant="claw-outline"
            size="icon"
            onClick={() => {
              setIsRefreshing(true);
              setTimeout(() => {
                handleSearchChange('');
                setCurrentPage(1);
                setTimeout(() => setIsRefreshing(false), 50);
              }, 250);
            }}
            title="刷新"
            className="w-9 h-9"
          >
            <RefreshCw className="w-4 h-4" />
          </Button>
        </div>

        {/* 分类 Tab
            停服态下仍允许分类切换：纯导航/查看类操作。 */}
        <div data-billing-exempt>
          <FilterChipGroup
            items={PUBLIC_SKILL_CATEGORIES.map(cat => ({ id: cat.id, label: cat.name }))}
            value={activeCategory}
            onChange={handleCategoryChange}
          />
        </div>

        {/* 技能卡片网格 */}
        {filteredSkills.length > 0 ? (
          <>
            <div className="grid grid-cols-3 gap-4" style={{ opacity: isRefreshing ? 0 : 1, transition: 'opacity 0.25s ease' }}>
              {pagedSkills.map((skill, index) => {
                const globalRank = (currentPage - 1) * PAGE_SIZE + index + 1;
                const isFeatured = activeCategory === 'featured';
                return (
                  <SkillCard
                    key={skill.id}
                    skill={skill}
                    rank={isFeatured ? globalRank : 0}
                    isFavorited={isFavorited(skill.id)}
                    onFavorite={handleFavorite}
                    onAddToPackage={handleAddToPackage}
                    onDistribute={handleDistribute}
                    onUninstall={handleUninstall}
                    disableActions={isDistributionInProgress(skill.id)}
                    uninstallDisabled={getUninstallableCount(skill.id) === 0}
                    onClick={() => setSelectedSkillId(skill.id)}
                  />
                );
              })}
            </div>
            <div className="pt-3 border-t border-gray-200 mt-2">
              <Pagination
                total={filteredSkills.length}
                current={currentPage}
                pageSize={PAGE_SIZE}
                showTotal={(total) => `共 ${total} 个技能`}
                className="w-full justify-between"
                onChange={(p) => { setCurrentPage(p); window.scrollTo({ top: 0, behavior: 'smooth' }); }}
              />
            </div>
          </>
        ) : (
          <Empty className="border-0 py-16">
            <EmptyHeader>
              <EmptyMedia />
              <EmptyDescription>
                {activeCategory === 'favorites' && favorites.length === 0
                  ? '还没有收藏任何技能'
                  : '没有找到匹配的技能'}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        )}
      </div>
      <AddToPackageDialog
        open={!!addToPackageSkillId}
        itemName={PUBLIC_SKILLS.find(s => s.id === addToPackageSkillId)?.nameZh || ''}
        itemType="skill"
        packages={packages}
        groups={groups}
        addedPackageIds={addToPackageSkillId ? addedPackageIdsBySkill[addToPackageSkillId] ?? [] : []}
        onConfirm={handlePackageSelected}
        onCancel={() => setAddToPackageSkillId(null)}
      />
      <PublicBatchDistributeDialog
        open={distributeDialogOpen}
        onOpenChange={setDistributeDialogOpen}
        skillId={selectedDistributeSkill ? getPublicSkillResourceId(selectedDistributeSkill.id) : undefined}
        skillName={selectedDistributeSkill?.nameZh || selectedDistributeSkill?.name}
        skillVersion={selectedDistributeSkill?.version}
        instances={distributeInstances as any}
        groups={groups}
        onDistributionStart={handleDistributeStart}
        title="批量下发 Skill"
        descriptionNode={(
          <div className="space-y-1">
            <BodyText as="p" tone="muted">
              将 <BodyMedium as="span" tone="primary" className="font-semibold">
                {selectedDistributeSkill?.nameZh || selectedDistributeSkill?.name}
              </BodyMedium> 部署至所选实例。
            </BodyText>
            <BodyText as="p" tone="muted">
              仅支持向 <BodyMedium as="span" tone="secondary">运行中</BodyMedium> 的实例下发技能; 已下发当前版本技能的实例默认不展示，可选择后再次下发。
            </BodyText>
          </div>
        )}
      />
      {uninstallSkillId && (
        <PublicBatchDeleteDialog
          open={uninstallDialogOpen}
          onOpenChange={setUninstallDialogOpen}
          skillName={PUBLIC_SKILLS.find((skill) => skill.id === uninstallSkillId)?.nameZh || PUBLIC_SKILLS.find((skill) => skill.id === uninstallSkillId)?.name || ''}
          skillVersion={PUBLIC_SKILLS.find((skill) => skill.id === uninstallSkillId)?.version || ''}
          resourceLabel="Skill"
          distributedInstances={distributedInstancesForUninstall}
          groups={groups}
          onDeleteStart={handleUninstallStart}
          warningText={'通过下发按钮安装的技能可支持移出（包括用户下发和管理端下发）。卸载成功后，该技能在对应实例上恢复为"未下发"状态。'}
          emptyText="暂无已下发的实例"
        />
      )}
    </>
  );
}
