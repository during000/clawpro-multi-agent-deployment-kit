'use client';
import { useState, useEffect, useMemo, useCallback, lazy, Suspense } from 'react';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { Search, Code, Eye, Trash2, RefreshCw, PackageX, Info } from 'lucide-react';
import { toast } from 'sonner';
import MDXRenderer from '@/components/MDXRenderer';
import { Input } from '@/components/ui/input';
import { BackButton } from '@/components/ui/back-button';
import { FileTree } from '@/components/ui/tree';
import { FileBrowser } from '@/components/ui/file-browser';
import { SegmentedTabs } from '@/components/ui/segmented-tabs';
import {
  TenantDocTitle,
  PanelTitle,
  BodyText,
  BodyMedium,
  MetaText,
  MetaMedium,
  TinyText,
  HelperText,
} from '@/components/ui/Typography';
import { StatusTag } from '@/components/ui/status-tag';
import { Badge } from '@/components/ui/badge';
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
import BatchDistributeDialog from './BatchDistributeDialog';
import BatchDeleteDialog from './BatchDeleteDialog';
import DeleteSkillDialog from './DeleteSkillDialog';
import PluginUpdateDialog from './PluginUpdateDialog';
import { type DistributionStatus, DISTRIBUTION_STATUS_MAP } from './types';
import { MOCK_GROUPS, MOCK_OPENCLAW_INSTANCES } from './mockData';
import {
  getDistributionRecords,
  getInstancesWithPluginDistributionStatus,
  getCurrentPluginInstalledInstances,
  addDistributionRecord,
  updateDistributionRecord,
  createDistributionRecordId,
  type CachedDistributionRecord,
} from './distributionCache';
import { type Plugin } from './PluginUploadDialog';

// 懒加载 react-syntax-highlighter
const SyntaxHighlighter = lazy(() =>
  import('react-syntax-highlighter').then(mod => ({ default: mod.Light as React.ComponentType<any> }))
);
const loadedLanguages = new Set<string>();
const registerLanguage = async (lang: string) => {
  if (loadedLanguages.has(lang)) return;
  loadedLanguages.add(lang);
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

interface PluginDetailProps {
  plugin: Plugin;
  onBack: () => void;
  onPluginDelete?: (pluginId: string) => void;
  onPluginUpdate?: (updatedPlugin: Plugin) => void;
}

const VIEWABLE_EXTENSIONS = ['.md', '.mdx', '.xml', '.json', '.txt', '.yaml', '.yml', '.toml', '.ini', '.cfg', '.conf', '.sh', '.bat', '.py', '.js', '.ts', '.css', '.html', '.htm', '.svg', '.env', '.gitignore', '.dockerfile'];

const isViewableFile = (name: string) => {
  const lower = name.toLowerCase();
  if (!lower.includes('.') && !lower.includes('/')) return true;
  return VIEWABLE_EXTENSIONS.some(ext => lower.endsWith(ext));
};

const isMarkdownFile = (name: string) => {
  const lower = name.toLowerCase();
  return lower.endsWith('.md') || lower.endsWith('.mdx');
};

const getFileLanguage = (name: string): string => {
  const ext = name.split('.').pop()?.toLowerCase() || '';
  const langMap: Record<string, string> = {
    json: 'json', xml: 'xml', yaml: 'yaml', yml: 'yaml',
    toml: 'toml', py: 'python', js: 'javascript', ts: 'typescript',
    css: 'css', html: 'html', htm: 'html', sh: 'bash', bat: 'batch',
    svg: 'xml', ini: 'ini', cfg: 'ini', conf: 'ini',
  };
  return langMap[ext] || 'text';
};

export default function PluginDetail({ plugin, onBack, onPluginDelete, onPluginUpdate }: PluginDetailProps) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [updateDialogOpen, setUpdateDialogOpen] = useState(false);
  const [distributeDialogOpen, setDistributeDialogOpen] = useState(false);
  const [batchDeleteDialogOpen, setBatchDeleteDialogOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('files');
  const [selectedVersion, setSelectedVersion] = useState<string>(plugin.version);
  const [expandedFile, setExpandedFile] = useState<string | null>(null);
  const [fileViewMode, setFileViewMode] = useState<'preview' | 'source'>('source');

  // 下发记录
  const [distributionRecords, setDistributionRecords] = useState<CachedDistributionRecord[]>([]);
  const [activeDistributionId, setActiveDistributionId] = useState<string | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [statusFilter, setStatusFilter] = useState<'all' | DistributionStatus>('all');
  const [detailSearchQuery, setDetailSearchQuery] = useState('');

  const refreshRecords = useCallback(() => {
    setDistributionRecords(getDistributionRecords(plugin.id));
  }, [plugin.id]);

  useEffect(() => {
    refreshRecords();
    const handler = () => refreshRecords();
    window.addEventListener('distribution-cache-updated', handler);
    return () => window.removeEventListener('distribution-cache-updated', handler);
  }, [refreshRecords]);

  const distributionOnlyRecords = useMemo(
    () => distributionRecords.filter(r => (r.type || 'distribute') === 'distribute'),
    [distributionRecords]
  );
  const deleteOnlyRecords = useMemo(
    () => distributionRecords.filter(r => r.type === 'delete'),
    [distributionRecords]
  );
  const hasInProgress = distributionRecords.some(r => r.status === 'distributing' || r.status === 'deleting');
  const hasDeleting = distributionRecords.some(r => r.status === 'deleting');
  const hasDistributing = distributionRecords.some(r => r.status === 'distributing');
  const isPluginIdentifierMissing = !plugin.id || !plugin.slug?.trim();
  const updateDisabled = hasInProgress || isPluginIdentifierMissing;
  const distributeInstances = useMemo(
    () => getInstancesWithPluginDistributionStatus(plugin.id, plugin.version, MOCK_OPENCLAW_INSTANCES),
    [plugin.id, plugin.version, distributionRecords]
  );
  const distributedInstancesForDelete = useMemo(() => {
    return getCurrentPluginInstalledInstances(plugin.id, plugin.version, MOCK_OPENCLAW_INSTANCES)
      .map(inst => {
        const groupName = inst.groupIds?.[0]
          ? MOCK_GROUPS.find(g => g.id === inst.groupIds[0])?.name
          : undefined;
        return {
          id: inst.id,
          name: inst.name,
          createdBy: inst.createdBy || 'admin',
          groupName: groupName || '全部用户',
          distributedVersion: inst.distributedVersion || plugin.version,
          deleteStatus: inst.failReason ? 'delete_failed' as const : 'not_deleted' as const,
          deleteFailReason: inst.failReason,
        };
      });
  }, [plugin.id, plugin.version, distributionRecords]);



  const pluginVersions = useMemo(() => {
    const versions = plugin.versions?.length ? plugin.versions : [plugin.version];
    return versions.includes(plugin.version) ? versions : [plugin.version, ...versions];
  }, [plugin.version, plugin.versions]);

  useEffect(() => {
    setSelectedVersion(plugin.version);
  }, [plugin.version]);

  // 构建 FileBrowser 所需的 versions 数据（包含 changeLog）
  const versionsForBrowser = useMemo(() => {
    const latestVersion = pluginVersions.includes(plugin.version) ? plugin.version : pluginVersions[0];
    return pluginVersions.map((ver: string, idx: number) => {
      const isLatest = ver === latestVersion;
      const versionRecord = plugin.versionHistory?.find((v: any) => v.version === ver);
      const dateStr = versionRecord?.date || (() => {
        const versionDate = new Date(plugin.uploadTime);
        const latestIndex = pluginVersions.findIndex((v: string) => v === latestVersion);
        versionDate.setDate(versionDate.getDate() - Math.abs(idx - latestIndex) * 14);
        return `${versionDate.getFullYear()}-${String(versionDate.getMonth() + 1).padStart(2, '0')}-${String(versionDate.getDate()).padStart(2, '0')}`;
      })();
      return {
        version: ver,
        date: dateStr,
        isLatest,
        changeLog: versionRecord?.changeLog || '暂无更新说明',
      };
    });
  }, [pluginVersions, plugin.uploadTime, plugin.versionHistory]);

  // 根据选中版本获取文件列表（历史版本优先从 versionHistory 中取）
  const currentFiles = useMemo(() => {
    const latestVersion = pluginVersions.includes(plugin.version) ? plugin.version : pluginVersions[0];
    if (!selectedVersion || selectedVersion === latestVersion) {
      return plugin.files || [];
    }
    const versionRecord = plugin.versionHistory?.find(v => v.version === selectedVersion);
    if (versionRecord?.files && versionRecord.files.length > 0) {
      return versionRecord.files;
    }
    return plugin.files || [];
  }, [plugin, pluginVersions, selectedVersion]);

  // 剥离唯一顶层文件夹
  const { processedFiles, strippedPrefix } = useMemo(() => {
    if (currentFiles.length === 0) return { processedFiles: currentFiles, strippedPrefix: '' };
    const topDirs = new Set<string>();
    let topFileCount = 0;
    for (const f of currentFiles) {
      const parts = f.name.split('/');
      if (parts.length > 1) {
        topDirs.add(parts[0]);
      } else {
        topFileCount++;
      }
    }
    if (topDirs.size === 1 && topFileCount === 0) {
      const prefix = Array.from(topDirs)[0] + '/';
      return {
        processedFiles: currentFiles.map(f => ({ ...f, name: f.name.slice(prefix.length) })),
        strippedPrefix: prefix,
      };
    }
    return { processedFiles: currentFiles, strippedPrefix: '' };
  }, [currentFiles]);

  // 默认选中 openclaw.plugin.json
  useEffect(() => {
    if (processedFiles.length > 0) {
      const pluginJson = processedFiles.find(f => f.name.endsWith('openclaw.plugin.json'));
      if (pluginJson) {
        setExpandedFile(pluginJson.name);
        setFileViewMode('source');
      } else {
        const first = processedFiles.find(f => !f.name.endsWith('/') && isViewableFile(f.name));
        if (first) {
          setExpandedFile(first.name);
          setFileViewMode(isMarkdownFile(first.name) ? 'preview' : 'source');
        }
      }
    }
  }, [processedFiles]);

  // 初始化展开顶层文件夹（已统一交由共享 FileTree 组件管理）

  const getFileContent = (fileName: string): string => {
    const originalName = strippedPrefix ? strippedPrefix + fileName : fileName;
    const file = currentFiles.find(f => f.name === originalName);
    if (file?.content) return file.content;
    const file2 = currentFiles.find(f => f.name === fileName);
    if (file2?.content) return file2.content;
    return '';
  };

  // 下发逻辑
  const handleDistributionStart = (selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId: plugin.id,
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'distributing',
      type: 'distribute',
      instances: selectedInstancesData.map(inst => ({
        id: inst.id,
        name: inst.name,
        createdBy: inst.createdBy || 'admin',
        distributionStatus: 'distributing' as DistributionStatus,
      })),
    };
    addDistributionRecord(newRecord);
    setActiveDistributionId(recordId);
    setDistributeDialogOpen(false);
    simulateDistribution(recordId, selectedInstanceIds.length);
  };

  const simulateDistribution = (recordId: string, totalCount: number) => {
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
          status: (failedCount === 0 ? 'success' : 'failed') as DistributionStatus,
          instances: record.instances.map((inst, idx) => ({
            ...inst,
            distributionStatus: (idx < successCount ? 'success' : 'failed') as DistributionStatus,
            distributedVersion: idx < successCount ? plugin.version : undefined,
            failReason: idx < successCount ? undefined : '命令下发失败',
          })),
        }));
      } else {
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount: completed,
          inProgressCount: totalCount - completed,
        }));
      }
    }, 800);
  };

  const handleDeleteStart = (selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId: plugin.id,
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'deleting',
      type: 'delete',
      operator: 'admin',
      instances: selectedInstancesData.map(inst => ({
        id: inst.id,
        name: inst.name,
        createdBy: inst.createdBy || 'admin',
        distributionStatus: 'distributing' as DistributionStatus,
      })),
    };
    addDistributionRecord(newRecord);
    setActiveDistributionId(recordId);
    setBatchDeleteDialogOpen(false);
    toast.success('已开始卸载流程');
    simulateDeletion(recordId, selectedInstanceIds.length);
  };

  const simulateDeletion = (recordId: string, totalCount: number) => {
    let completed = 0;
    const failReasons = ['实例离线', '权限不足', '插件被占用', '网络超时', '实例已停止'];
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
          status: 'success' as DistributionStatus,
          instances: record.instances.map((inst, idx) => ({
            ...inst,
            distributionStatus: (results[idx] ? 'success' : 'failed') as DistributionStatus,
            failReason: results[idx] ? undefined : failReasons[Math.floor(Math.random() * failReasons.length)],
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
  };

  const handlePluginDelete = () => {
    if (onPluginDelete) {
      onPluginDelete(plugin.id);
    }
    toast.success(`插件「${plugin.name}」已删除`);
    setDeleteDialogOpen(false);
    onBack();
  };

  const handlePluginUpdate = (updatedPlugin: Plugin) => {
    onPluginUpdate?.(updatedPlugin);
  };

  const activeDistribution = distributionRecords.find(r => r.id === activeDistributionId);
  const filteredInstances = activeDistribution
    ? activeDistribution.instances.filter(inst => {
        const matchesStatus = statusFilter === 'all' || inst.distributionStatus === statusFilter;
        const searchLower = detailSearchQuery.toLowerCase();
        const matchesSearch = !detailSearchQuery ||
          inst.name.toLowerCase().includes(searchLower) ||
          inst.id.toLowerCase().includes(searchLower);
        return matchesStatus && matchesSearch;
      })
    : [];

  return (
    <div className="space-y-6">
      {/* 返回按钮 */}
      <BackButton onClick={onBack}>返回列表</BackButton>

      {/* 基础信息卡片 */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <TenantDocTitle as="h1" tone="primary" className="mb-1">{plugin.name}</TenantDocTitle>
            <MetaText as="p" tone="weak" className="font-mono mb-2">slug：{plugin.slug}</MetaText>
            <div className="flex items-center gap-2 flex-wrap">
              <StatusTag mode="fill" variant="gray" className="font-mono">v{plugin.version}</StatusTag>
            </div>
          </div>
          <div className="flex items-center gap-2 ml-4 flex-shrink-0">
            <Tooltip delayDuration={1000}>
              <TooltipTrigger asChild>
                <span>
                  <Button
                    variant="claw-outline"
                    size="claw"
                    onClick={() => {
                      if (!updateDisabled) setUpdateDialogOpen(true);
                    }}
                    disabled={updateDisabled}
                    className={updateDisabled ? 'opacity-50 cursor-not-allowed' : ''}
                  >
                    <RefreshCw className="w-4 h-4 mr-1.5" />
                    更新
                  </Button>
                </span>
              </TooltipTrigger>
              {updateDisabled && (
                <TooltipContent>有下发任务进行中</TooltipContent>
              )}
            </Tooltip>

            <Tooltip delayDuration={1000}>
              <TooltipTrigger asChild>
                <span>
                  <Button
                    variant="outline-destructive"
                    size="claw"
                    onClick={() => setDeleteDialogOpen(true)}
                    disabled={hasInProgress}
                    className={hasInProgress ? 'opacity-50 cursor-not-allowed' : ''}
                  >
                    <Trash2 className="w-4 h-4 mr-1.5" />
                    删除
                  </Button>
                </span>
              </TooltipTrigger>
              {hasInProgress && (
                <TooltipContent>有下发任务进行中</TooltipContent>
              )}
            </Tooltip>
          </div>
        </div>
        {plugin.description && (
          <BodyText as="p" tone="secondary" className="mt-3">{plugin.description}</BodyText>
        )}
      </div>

      {/* Tab 区域 */}
      <div>
        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <div className="flex items-center justify-between">
            <SegmentedTabs
              tabs={[
                { id: 'files', label: '文件列表' },
                { id: 'distribution', label: '下发记录' },
                { id: 'uninstall', label: '卸载记录' },
              ]}
              active={activeTab}
              onChange={setActiveTab}
              ariaLabel="插件详情 Tab 切换"
            />
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
                onClick={() => setBatchDeleteDialogOpen(true)}
                disabled={hasInProgress || distributedInstancesForDelete.length === 0}
                variant="claw-primary"
                size="claw"
                className={hasInProgress || distributedInstancesForDelete.length === 0 ? 'opacity-50 cursor-not-allowed' : ''}
                title={hasInProgress ? '有任务进行中' : distributedInstancesForDelete.length === 0 ? '暂无可卸载实例' : undefined}
              >
                <PackageX className="w-4 h-4 mr-1.5" />
                {hasDeleting ? '卸载中...' : '批量卸载'}
              </Button>
            )}
          </div>
          <TabsList className="hidden">
            <TabsTrigger value="files" />
            <TabsTrigger value="distribution" />
            <TabsTrigger value="uninstall" />
          </TabsList>

          {/* 文件列表 Tab */}
          <TabsContent value="files" className="mt-4 p-0">
            <FileBrowser
              versions={versionsForBrowser}
              files={processedFiles}
              getFileContent={(fileName: string) => {
                const originalName = strippedPrefix ? strippedPrefix + fileName : fileName;
                const file = currentFiles.find((f: any) => f.name === originalName);
                if (file?.content) return file.content;
                const file2 = currentFiles.find((f: any) => f.name === fileName);
                if (file2?.content) return file2.content;
                // SKILL.md 特殊处理
                if (fileName === 'SKILL.md' || fileName.toLowerCase() === 'skill.md') {
                  const f = currentFiles.find((f: any) => f.name.toLowerCase() === 'skill.md' || f.name.toLowerCase().endsWith('/skill.md'));
                  if (f?.content) return f.content;
                  if (!selectedVersion || selectedVersion === pluginVersions[0]) return plugin?.content || '';
                  return '';
                }
                return '';
              }}
              height="47rem"
              defaultVersion={selectedVersion}
              onVersionChange={(ver: string) => setSelectedVersion(ver)}
            />
          </TabsContent>

          {/* 下发记录 Tab */}
          <TabsContent value="distribution" className="mt-4 p-0">
            <div className="space-y-3">
              {distributionOnlyRecords.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-12 text-center">
                  <HelperText as="p">还没有下发记录</HelperText>
                </div>
              ) : (
                <div className="space-y-3">
                  {distributionOnlyRecords.map((record, idx) => {
                    const progress = record.totalCount > 0 ? Math.round((record.successCount / record.totalCount) * 100) : 0;
                    return (
                      <div key={record.id} className="border border-gray-200 rounded-lg p-4">
                        <div className="flex items-start justify-between mb-3">
                          <div>
                            <BodyMedium as="p" tone="primary" className="font-semibold">
                              #{idx + 1} · v{plugin.version} {new Date(record.timestamp).toLocaleString('zh-CN')}
                            </BodyMedium>
                          </div>
                          <div className="flex items-center gap-2">
                            <span className={`inline-block px-3 py-1 rounded text-xs font-medium ${
                              record.status === 'distributing' ? 'bg-blue-50 text-blue-700' :
                              record.successCount === record.totalCount ? 'bg-green-50 text-green-700' :
                              record.successCount === 0 && record.failedCount > 0 ? 'bg-red-50 text-red-700' :
                              'bg-yellow-50 text-yellow-700'
                            }`}>

                              {record.status === 'distributing'
                                ? `下发中 ${progress}%`
                                : `下发完成，${record.successCount}个下发成功，${record.failedCount}个失败`}
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
                              className="text-blue-600 hover:text-blue-700 h-auto py-1 px-2"
                            >
                              查看详情
                            </Button>
                          </div>
                        </div>
                        {record.status === 'distributing' && (
                          <div className="mb-2">
                            <div className="w-full bg-gray-200 rounded-full h-2">
                              <div
                                className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                                style={{ width: `${progress}%` }}
                              />
                            </div>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </TabsContent>

          {/* 卸载记录 Tab */}
          <TabsContent value="uninstall" className="mt-4 p-0">
            <div className="space-y-3">
              {deleteOnlyRecords.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-12 text-center">
                  <HelperText as="p">还没有卸载记录</HelperText>
                </div>
              ) : (
                <div className="space-y-3">
                  {deleteOnlyRecords.map((record, idx) => {
                    const progress = record.totalCount > 0 ? Math.round((record.successCount / record.totalCount) * 100) : 0;
                    const isInProgress = record.status === 'deleting' || record.status === 'distributing';
                    return (
                      <div key={record.id} className="border border-gray-200 rounded-lg p-4">
                        <div className="flex items-start justify-between mb-3">
                          <div>
                            <BodyMedium as="p" tone="primary" className="font-semibold">
                              #{idx + 1} · {new Date(record.timestamp).toLocaleString('zh-CN')}
                            </BodyMedium>
                          </div>
                          <div className="flex items-center gap-2">
                            <span className={`inline-block px-3 py-1 rounded text-xs font-medium ${
                              isInProgress ? 'bg-red-100 text-red-700' :
                              record.successCount === 0 ? 'bg-red-50 text-red-700' :
                              record.failedCount === 0 ? 'bg-green-50 text-green-700' :
                              'bg-yellow-50 text-yellow-700'
                            }`}>
                              {isInProgress
                                ? `卸载中 ${progress}%`
                                : `卸载完成，${record.successCount}个卸载成功，${record.failedCount}个失败`}
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
                              className="text-blue-600 hover:text-blue-700 h-auto py-1 px-2"
                            >
                              查看详情
                            </Button>
                          </div>
                        </div>
                        {isInProgress && (
                          <div className="mb-2">
                            <div className="w-full bg-gray-200 rounded-full h-2">
                              <div
                                className="bg-red-500 h-2 rounded-full transition-all duration-300"
                                style={{ width: `${progress}%` }}
                              />
                            </div>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </TabsContent>
        </Tabs>
      </div>

      {/* 更新插件弹窗 */}
      <PluginUpdateDialog
        open={updateDialogOpen}
        onOpenChange={setUpdateDialogOpen}
        plugin={plugin}
        onConfirm={handlePluginUpdate}
        defaultSecurityScan={false}
        onDefaultSecurityScanChange={() => {}}
        securityServiceActive={true}
      />

      {/* 批量下发对话框 */}
      <BatchDistributeDialog
        open={distributeDialogOpen}
        onOpenChange={setDistributeDialogOpen}
        skillId={plugin.id}
        skillName={plugin.name}
        skillVersion={plugin.version}
        onDistributionStart={handleDistributionStart}
        title="批量下发插件"
        showScopeFilter
        instances={distributeInstances}
        groups={MOCK_GROUPS}
      />

      {/* 批量卸载对话框 */}
      <BatchDeleteDialog
        open={batchDeleteDialogOpen}
        onOpenChange={setBatchDeleteDialogOpen}
        skillName={plugin.name}
        skillVersion={plugin.version}
        distributedInstances={distributedInstancesForDelete}
        groups={MOCK_GROUPS}
        onDeleteStart={handleDeleteStart}
        resourceLabel="插件"
        warningText="卸载成功后，该插件在对应实例上恢复为“未下发”状态。"
        emptyText="暂无可卸载实例"
      />

      {/* 删除确认对话框 */}
      <DeleteSkillDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        skillName={plugin.name}
        onConfirm={handlePluginDelete}
      />

      {/* 下发详情对话框 */}
      <Dialog open={detailsOpen} onOpenChange={setDetailsOpen}>
        <DialogContent className="max-w-3xl max-h-96">
          <DialogHeader>
            <DialogTitle>{activeDistribution && activeDistribution.type === 'delete' ? '卸载详情' : '下发详情'}</DialogTitle>
          </DialogHeader>
          {activeDistribution && (
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <div className="relative flex-1">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <Input
                    placeholder="搜索实例名称/ID..."
                    value={detailSearchQuery}
                    onChange={(e) => setDetailSearchQuery(e.target.value)}
                    className="pl-10 h-9"
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
                    <SelectItem value="distributing">{activeDistribution && activeDistribution.type === 'delete' ? '卸载中' : '下发中'}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="border border-gray-200 rounded-lg overflow-hidden">
                <Table density="compact" containerClassName="max-h-64 overflow-y-auto">
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
                          <TableCell className="text-gray-500 font-mono">{instance.id}</TableCell>
                          <TableCell>
                            <span className={`inline-block px-2 py-1 rounded text-xs font-medium ${
                              DISTRIBUTION_STATUS_MAP[instance.distributionStatus]?.color || 'bg-gray-50 text-gray-500'
                            }`}>
                              {DISTRIBUTION_STATUS_MAP[instance.distributionStatus]?.label || '未下发'}
                            </span>
                          </TableCell>
                          <TableCell className="text-gray-500">
                            {(instance as any).failReason || '-'}
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
