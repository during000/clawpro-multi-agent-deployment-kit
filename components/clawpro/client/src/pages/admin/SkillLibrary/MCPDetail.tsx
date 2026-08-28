import React from 'react';
/**
 * MCPDetail - MCP 服务详情页
 * 展示基本信息 + 两个 Tab（文件列表 / 下发记录）
 * 文件列表 Tab 内三栏布局：版本列表 | 文件列表 | 内容展示
 * 样式参考 PluginDetail
 */
import { useState, useEffect, useCallback, useMemo, lazy, Suspense } from 'react';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { Trash2, Search, Eye, Code, Loader } from 'lucide-react';
import { toast } from 'sonner';
import MDXRenderer from '@/components/MDXRenderer';
import { Input } from '@/components/ui/input';
import { BackButton } from '@/components/ui/back-button';
import { FileTree } from '@/components/ui/tree';
import { FileBrowser } from '@/components/ui/file-browser';
import { SegmentedTabs } from '@/components/ui/segmented-tabs';
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
import DeleteSkillDialog from './DeleteSkillDialog';
import { type MCPService, type DistributionStatus, DISTRIBUTION_STATUS_MAP } from './types';
import { MOCK_GROUPS, MOCK_OPENCLAW_INSTANCES } from './mockData';
import {
  TenantDocTitle,
  PanelTitle,
  BodyText,
  BodyMedium,
  MetaText,
  MetaMedium,
  MiniBodyText,
  TinyText,
  CodeText,
  HelperText,
} from '@/components/ui/Typography';
import { StatusTag } from '@/components/ui/status-tag';
import { Badge } from '@/components/ui/badge';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table';
import {
  getDistributionRecords,
  addDistributionRecord,
  updateDistributionRecord,
  createDistributionRecordId,
  type CachedDistributionRecord,
} from './distributionCache';

// 懒加载语法高亮
const SyntaxHighlighter = lazy(() =>
  import('react-syntax-highlighter').then(mod => ({ default: mod.Light as any as React.ComponentType<any> }))
);
const loadedLanguages = new Set<string>();
const registerJsonLanguage = async () => {
  if (loadedLanguages.has('json')) return;
  loadedLanguages.add('json');
  try {
    const mod = await import('react-syntax-highlighter');
    const Light = mod.Light as any;
    const jsonMod = await import('react-syntax-highlighter/dist/esm/languages/hljs/json');
    Light.registerLanguage('json', jsonMod.default);
  } catch { /* 静默降级 */ }
};
const registerMarkdownLanguage = async () => {
  if (loadedLanguages.has('markdown')) return;
  loadedLanguages.add('markdown');
  try {
    const mod = await import('react-syntax-highlighter');
    const Light = mod.Light as any;
    const mdMod = await import('react-syntax-highlighter/dist/esm/languages/hljs/markdown');
    Light.registerLanguage('markdown', mdMod.default);
  } catch { /* 静默降级 */ }
};

const hljsStyle: Record<string, React.CSSProperties> = {
  'hljs': { display: 'block', overflowX: 'auto', padding: '1em', background: '#ffffff', color: '#383a42' },
  'hljs-comment': { color: '#a0a1a7', fontStyle: 'italic' },
  'hljs-keyword': { color: '#a626a4' },
  'hljs-number': { color: '#986801' },
  'hljs-string': { color: '#50a14f' },
  'hljs-attr': { color: '#986801' },
  'hljs-literal': { color: '#0184bb' },
  'hljs-name': { color: '#e45649' },
  'hljs-title': { color: '#4078f2' },
  'hljs-type': { color: '#4078f2' },
  'hljs-punctuation': { color: '#383a42' },
  'hljs-section': { color: '#e45649' },
  'hljs-bullet': { color: '#4078f2' },
  'hljs-link': { color: '#4078f2' },
  'hljs-emphasis': { fontStyle: 'italic' },
  'hljs-strong': { fontWeight: 'bold' },
};

/** MCP 固定的三个文件 */
interface MCPFile {
  name: string;
  label: string;
  language: 'markdown' | 'json';
}

const MCP_FILES: MCPFile[] = [
  { name: '使用说明.md', label: '使用说明.md', language: 'markdown' },
  { name: '工具说明.md', label: '工具说明.md', language: 'markdown' },
  { name: '服务配置.json', label: '服务配置.json', language: 'json' },
];

interface MCPDetailProps {
  mcp: MCPService;
  onBack: () => void;
  onMCPDelete?: (mcpId: string) => void;
}

export default function MCPDetail({ mcp, onBack, onMCPDelete }: MCPDetailProps) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [distributeDialogOpen, setDistributeDialogOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('files');
  const [selectedVersion, setSelectedVersion] = useState<string>(
    mcp.versions?.[mcp.versions.length - 1] || mcp.version
  );
  const [selectedFile, setSelectedFile] = useState<string>('使用说明.md');
  const [fileViewMode, setFileViewMode] = useState<'preview' | 'source'>('preview');

  // 下发记录
  const [distributionRecords, setDistributionRecords] = useState<CachedDistributionRecord[]>([]);
  const [activeDistributionId, setActiveDistributionId] = useState<string | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [statusFilter, setStatusFilter] = useState<'all' | DistributionStatus>('all');
  const [detailSearchQuery, setDetailSearchQuery] = useState('');

  const refreshRecords = useCallback(() => {
    setDistributionRecords(getDistributionRecords(mcp.name));
  }, [mcp.name]);

  useEffect(() => {
    refreshRecords();
    const handler = () => refreshRecords();
    window.addEventListener('distribution-cache-updated', handler);
    return () => window.removeEventListener('distribution-cache-updated', handler);
  }, [refreshRecords]);

  const hasInProgress = distributionRecords.some(r => r.status === 'distributing');

  // 注册语法高亮语言
  useEffect(() => {
    registerJsonLanguage();
    registerMarkdownLanguage();
  }, []);

  // 版本列表（从新到旧）
  const versions = [...(mcp.versions || [mcp.version])].reverse();

  // 构建 FileBrowser 所需的 versions 数据（包含 changeLog）
  const versionsForBrowser = useMemo(() => {
    return versions.map((ver: string, idx: number) => {
      const isLatest = idx === 0;
      // 模拟版本日期（从最新往前推，每个版本间隔 15 天）
      const baseDate = mcp.updatedAt || mcp.createdAt;
      const versionDate = new Date(baseDate);
      versionDate.setDate(versionDate.getDate() - idx * 15);
      return {
        version: ver,
        date: versionDate.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' }).replace(/\//g, '-'),
        isLatest,
        changeLog: `MCP 服务 ${mcp.name} 的 ${ver} 版本更新`,
      };
    });
  }, [versions, mcp.updatedAt, mcp.createdAt, mcp.name, mcp.version]);

  // 获取文件内容
  const getFileContent = (fileName: string): string => {
    switch (fileName) {
      case '使用说明.md':
        return mcp.usageDoc?.trim() || '';
      case '工具说明.md':
        return mcp.toolDoc?.trim() || '';
      case '服务配置.json': {
        try {
          return JSON.stringify(JSON.parse(mcp.configJson), null, 2);
        } catch {
          return mcp.configJson;
        }
      }
      default:
        return '';
    }
  };

  // 获取文件对应的语法高亮语言
  const getFileLanguage = (fileName: string): string => {
    const file = MCP_FILES.find(f => f.name === fileName);
    return file?.language || 'text';
  };

  // 判断是否为 Markdown 文件
  const isMarkdownFile = (fileName: string): boolean => {
    return fileName.endsWith('.md') || fileName.endsWith('.mdx');
  };

  // 下发逻辑
  const handleDistributionStart = (selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId: mcp.name,
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'distributing',
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

  const handleMCPDelete = () => {
    if (onMCPDelete) onMCPDelete(mcp.name);
    toast.success(`MCP「${mcp.displayName || mcp.name}」已删除`);
    setDeleteDialogOpen(false);
    onBack();
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

  // 渲染右侧内容区
  const renderFileContent = () => {
    if (!selectedFile) {
      return (
        <div className="flex items-center justify-center h-full">
          <BodyText as="p" tone="muted">选择一个文件查看内容</BodyText>
        </div>
      );
    }

    const content = getFileContent(selectedFile);
    const isMd = isMarkdownFile(selectedFile);

    if (!content) {
      return (
        <>
          <div className="h-12 px-3 border-b border-gray-200 flex items-center justify-between">
            <BodyMedium as="p" tone="emphasis">{selectedFile}</BodyMedium>
            {isMd && renderViewModeSwitch()}
          </div>
          <div className="flex items-center justify-center h-full">
            <BodyText as="p" tone="weak">文件内容暂无</BodyText>
          </div>
        </>
      );
    }

    return (
      <>
        <div className="h-12 px-3 border-b border-gray-200 flex items-center justify-between">
          <BodyMedium as="p" tone="emphasis">{selectedFile}</BodyMedium>
          {isMd && renderViewModeSwitch()}
        </div>
        <div className="flex-1 overflow-y-auto">
          {isMd && fileViewMode === 'preview' ? (
            renderPreviewView(content, selectedFile)
          ) : (
            renderSourceView(content, getFileLanguage(selectedFile))
          )}
        </div>
      </>
    );
  };

  // 源码模式
  const renderSourceView = (content: string, lang: string) => {
    return (
      <Suspense fallback={
        <CodeText as="pre" tone="secondary" className="block overflow-x-auto whitespace-pre leading-5 bg-gray-50 p-3 m-0">
          {content}
        </CodeText>
      }>
        <SyntaxHighlighter
          language={lang}
          style={hljsStyle}
          showLineNumbers
          lineNumberStyle={{ color: '#b0b0b0', fontSize: '11px', minWidth: '2.5em', paddingRight: '1em', userSelect: 'none' }}
          customStyle={{ margin: 0, padding: '12px 0', fontSize: '12px', lineHeight: '1.6', background: '#ffffff', borderRadius: 0, overflowX: 'auto' }}
          wrapLongLines={false}
        >
          {content}
        </SyntaxHighlighter>
      </Suspense>
    );
  };

  // 预览模式
  const renderPreviewView = (content: string, fileName: string) => {
    if (isMarkdownFile(fileName)) {
      return (
        <div className="p-4">
          <MDXRenderer content={content} />
        </div>
      );
    }
    // JSON 等非 Markdown 文件，预览模式也使用 MDXRenderer 渲染代码块
    const lang = getFileLanguage(fileName);
    return (
      <div className="p-4">
        <MDXRenderer content={`\`\`\`${lang}\n${content}\n\`\`\``} />
      </div>
    );
  };

  // 预览/源码 切换按钮
  const renderViewModeSwitch = () => (
    <div className="flex items-center gap-0.5 bg-gray-200/60 rounded p-0.5">
      <button
        onClick={() => setFileViewMode('preview')}
        className={`flex items-center gap-1 px-2 py-1 rounded transition-colors ${
          fileViewMode === 'preview'
            ? 'bg-white shadow-sm'
            : ''
        }`}
      >
        <Eye className="w-3 h-3" />
        <MiniBodyText
          as="span"
          tone={fileViewMode === 'preview' ? 'primary' : 'muted'}
          className={fileViewMode === 'preview' ? 'font-medium' : ''}
        >
          预览
        </MiniBodyText>
      </button>
      <button
        onClick={() => setFileViewMode('source')}
        className={`flex items-center gap-1 px-2 py-1 rounded transition-colors ${
          fileViewMode === 'source'
            ? 'bg-white shadow-sm'
            : ''
        }`}
      >
        <Code className="w-3 h-3" />
        <MiniBodyText
          as="span"
          tone={fileViewMode === 'source' ? 'primary' : 'muted'}
          className={fileViewMode === 'source' ? 'font-medium' : ''}
        >
          源码
        </MiniBodyText>
      </button>
    </div>
  );

  return (
    <div className="space-y-6">
      {/* 返回按钮 */}
      <BackButton onClick={onBack}>返回列表</BackButton>

      {/* 基础信息卡片 */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <TenantDocTitle as="h1" tone="primary" className="mb-1">{mcp.displayName || mcp.name}</TenantDocTitle>
            <MetaText as="p" tone="weak" className="mb-2">slug：{mcp.name}</MetaText>
            <div className="flex items-center gap-2 flex-wrap">
              <StatusTag mode="fill" variant="blue">
                {mcp.transport === 'stdio' ? '本地命令' : '远程服务'}
              </StatusTag>
              <StatusTag mode="fill" variant="gray" className="">v{mcp.version}</StatusTag>
              <MetaText as="span" tone="weak">
                创建于 {mcp.createdAt.toLocaleDateString('zh-CN')}
              </MetaText>
            </div>
          </div>
          <div className="flex items-center gap-2 ml-4 flex-shrink-0">
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
        {mcp.description && (
          <BodyText as="p" tone="secondary" className="mt-3">{mcp.description}</BodyText>
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
              ]}
              active={activeTab}
              onChange={setActiveTab}
              ariaLabel="MCP 详情 Tab 切换"
            />
            {activeTab === 'distribution' && (
              <Button
                variant="claw-primary"
                size="claw"
                onClick={() => setDistributeDialogOpen(true)}
                disabled={hasInProgress}
              >
                {hasInProgress ? '下发中...' : '批量下发'}
              </Button>
            )}
          </div>
          <TabsList className="hidden">
            <TabsTrigger value="files" />
            <TabsTrigger value="distribution" />
          </TabsList>

          {/* 文件列表 Tab — 使用 FileBrowser 组件 */}
          <TabsContent value="files" className="mt-4 p-0">
            <FileBrowser
              versions={versionsForBrowser}
              files={MCP_FILES.map((f) => ({ name: f.name }))}
              getFileContent={(fileName: string) => {
                switch (fileName) {
                  case '使用说明.md':
                    return mcp.usageDoc?.trim() || '';
                  case '工具说明.md':
                    return mcp.toolDoc?.trim() || '';
                  case '服务配置.json': {
                    try {
                      return JSON.stringify(JSON.parse(mcp.configJson), null, 2);
                    } catch {
                      return mcp.configJson;
                    }
                  }
                  default:
                    return '';
                }
              }}
              height="47rem"
              defaultVersion={selectedVersion}
              onVersionChange={(ver: string) => setSelectedVersion(ver)}
            />
          </TabsContent>

          {/* 下发记录 Tab */}
          <TabsContent value="distribution" className="mt-4 p-0">
            <div className="space-y-3">
              {distributionRecords.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-12 text-center">
                  <HelperText as="p">还没有下发记录</HelperText>
                </div>
              ) : (
                  <div className="space-y-3">
                    {distributionRecords.map((record, idx) => {
                      const progress = record.totalCount > 0 ? Math.round((record.successCount / record.totalCount) * 100) : 0;
                      return (
                        <div key={record.id} className="border border-gray-200 rounded-lg p-4">
                          <div className="flex items-start justify-between mb-3">
                            <div>
                              <BodyMedium as="p" tone="primary" className="font-semibold">
                                #{idx + 1} · {new Date(record.timestamp).toLocaleString('zh-CN')}
                              </BodyMedium>
                            </div>
                            <div className="flex items-center gap-2">
                              <span className={`inline-block px-3 py-1 rounded ${
                                record.status === 'distributing' ? 'bg-blue-50' :
                                record.successCount === record.totalCount ? 'bg-green-50' :
                                'bg-yellow-50'
                              }`}>
                                <MetaMedium
                                  as="span"
                                  tone="inherit"
                                  className={
                                    record.status === 'distributing' ? 'text-blue-700' :
                                    record.successCount === record.totalCount ? 'text-green-700' :
                                    'text-yellow-700'
                                  }
                                >
                                  {record.status === 'distributing'
                                    ? `下发中 ${progress}%`
                                    : `下发完成，${record.successCount}个下发成功，${record.failedCount}个失败`}
                                </MetaMedium>
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
                                className="text-[var(--text-brand)] hover:text-[var(--text-brand)] h-auto py-1 px-2"
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
        </Tabs>
      </div>

      {/* 批量下发对话框 */}
      <BatchDistributeDialog
        open={distributeDialogOpen}
        onOpenChange={setDistributeDialogOpen}
        skillName={mcp.displayName || mcp.name}
        onDistributionStart={handleDistributionStart}
        title="批量下发 MCP 配置"
        showScopeFilter
        instances={MOCK_OPENCLAW_INSTANCES}
        groups={MOCK_GROUPS}
        singleStatusFilter
        showConfirmDialog
        descriptionNode={
          <>
            <p>将 <span className="font-semibold text-gray-900">「{mcp.displayName || mcp.name}」</span> 部署至所选实例。</p>
            <p className="mt-1">仅支持智能体类型为 <span className="font-medium text-gray-700">OpenClaw</span> 且状态为 <span className="font-medium text-gray-700">运行中</span> 的实例。默认展示 <span className="font-medium text-gray-700">未下发</span> 和 <span className="font-medium text-gray-700">下发失败</span> 的实例，已下发成功实例可通过状态筛选查看。</p>
            <p className="mt-1"><span className="font-medium text-gray-700">26.3.28 以下版本</span>暂不支持 MCP 服务。</p>
          </>
        }
      />

      {/* 删除确认对话框 */}
      <DeleteSkillDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        skillName={mcp.displayName || mcp.name}
        onConfirm={handleMCPDelete}
      />

      {/* 下发详情对话框 */}
      <Dialog open={detailsOpen} onOpenChange={setDetailsOpen}>
        <DialogContent className="!max-w-[700px] max-h-[80vh] flex flex-col w-[700px]">
          <DialogHeader>
            <DialogTitle>下发详情</DialogTitle>
          </DialogHeader>
          {activeDistribution && (
            <div className="space-y-4 overflow-hidden flex flex-col">
              <div className="flex items-center gap-2">
                <div className="relative flex-1">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <Input
                    placeholder="搜索实例名称/ID..."
                    value={detailSearchQuery}
                    onChange={(e) => setDetailSearchQuery(e.target.value)}
                    className="pl-10 h-9 focus-visible:ring-0 focus-visible:border-blue-400"
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
                    <SelectItem value="distributing">下发中</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="border border-gray-200 rounded-lg overflow-hidden">
                <Table density="compact" containerClassName="max-h-64 overflow-y-auto">
                  <TableHeader>
                    <TableRow>
                      <TableHead>实例名称</TableHead>
                      <TableHead>实例ID</TableHead>
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
                          <TableCell className="text-gray-500">{instance.id}</TableCell>
                          <TableCell>
                            <span className={`inline-block px-2 py-0.5 rounded ${
                              DISTRIBUTION_STATUS_MAP[instance.distributionStatus]?.color || 'bg-gray-50 text-gray-500'
                            }`}>
                              <MetaMedium as="span" tone="inherit">
                                {DISTRIBUTION_STATUS_MAP[instance.distributionStatus]?.label || '未下发'}
                              </MetaMedium>
                            </span>
                          </TableCell>
                          <TableCell className="max-w-[200px] text-gray-500">
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
