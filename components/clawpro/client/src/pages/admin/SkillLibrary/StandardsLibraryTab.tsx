/**
 * StandardsLibraryTab - 企业规范库
 * 统一管理企业入口文件、企业规范 Markdown 与用户级 Hook 配置。
 */
import { useEffect, useMemo, useRef, useState } from 'react';
import { toast } from 'sonner';
import {
  CheckCircle2,
  Download,
  Grid3x3,
  List,
  PackageX,
  RefreshCw,
  Search,
  Send,
  Sparkles,
  Trash2,
  Upload,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription, AlertInfoIcon } from '@/components/ui/alert';
import { BackButton } from '@/components/ui/back-button';
import { Dialog, DialogBody, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty';
import { Input } from '@/components/ui/input';
import { MoreActionsDropdown } from '@/components/ui/more-actions-dropdown';
import { ScopeSelect } from '@/components/ScopeSelect';
import { SegmentedTabs } from '@/components/ui/segmented-tabs';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { SegmentGroup, SegmentOption } from '@/components/ui/segment';
import { StatusTag } from '@/components/ui/status-tag';
import { SurfaceCard } from '@/components/ui/Surface';
import {
  Table,
  TableActionCell,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Textarea } from '@/components/ui/textarea';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { FileBrowser } from '@/components/ui/file-browser';
import { BodyMedium, BodyText, CardTitle as CardHeading, HelperText, MetaMedium, MetaText, TenantDocTitle } from '@/components/ui/Typography';
import MDXRenderer from '@/components/MDXRenderer';
import BatchDeleteDialog from './BatchDeleteDialog';
import BatchDistributeDialog from './BatchDistributeDialog';
import EditScopePopover from './EditScopeDialog';
import FileReplaceHelper from './FileReplaceHelper';
import HookSettingsForm, {
  EMPTY_HOOK_FORM,
  buildHookManifestYaml,
  getHookFormError,
  type HookFormValue,
} from './HookSettingsForm';
import { MOCK_GROUPS, MOCK_PROJECT_GROUPS, MOCK_OPENCLAW_INSTANCES } from './mockData';
import { DISTRIBUTION_STATUS_MAP, type AgentInstance, type DistributionStatus, type SkillScope, type UploadedFile } from './types';
import UploadFileCard from './UploadFileCard';
import { UploadRequirementsCard } from './UploadRequirementsCard';
import {
  standardsStore,
  type AgentConfigAsset,
  type AssetKind,
  type TargetClient,
  type DeliveryTaskStatus,
} from './standardsStore';
import { projectAssetStore } from '../project-assets/projectAssetStore';

type StatusVariant = 'green' | 'blue' | 'red';

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

const TYPE_TEXT: Record<AssetKind, string> = {
  entry: 'CLAUDE.md',
  rule: 'rules',
  hook: 'hooks.yaml',
};

// 资产数据统一由 standardsStore 提供（localStorage + CustomEvent 共享）

const createSlug = (name: string) =>
  name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '') || `asset-${Date.now()}`;

const summarizeMarkdown = (contentMd: string) => {
  const lines = contentMd.split('\n').map((line) => line.trim()).filter(Boolean);
  const heading = lines.find((line) => /^#{1,6}\s+/.test(line));
  const source = heading || lines[0] || '';
  const summary = source
    .replace(/^#{1,6}\s+/gm, '')
    .replace(/^\s*[-*+]\s+/gm, '')
    .replace(/\*\*([^*]+)\*\*/g, '$1')
    .trim();
  return summary.length > 80 ? `${summary.slice(0, 80)}...` : summary;
};

const isYamlFile = (fileName: string) => /\.ya?ml$/i.test(fileName);

const inspectHookManifestYaml = (content: string) => {
  const emptyResult = { hookKeys: [] as string[], hookCount: 0 };
  if (!/^hooks:\s*$/m.test(content)) {
    return { ...emptyResult, error: 'YAML 顶层必须包含 hooks 数组' };
  }
  const hookBlocks = content.split(/(?=^ {2}- id:\s*)/m).slice(1);
  if (hookBlocks.length === 0) {
    return { ...emptyResult, error: 'hooks 数组至少需要包含一个 Hook' };
  }
  const hookKeys: string[] = [];
  for (const block of hookBlocks) {
    const id = block.match(/^ {2}- id:\s*["']?([^"'\n]+)["']?\s*$/m)?.[1]?.trim();
    const description = block.match(/^ {4}description:\s*(.+)$/m)?.[1]?.trim();
    const event = block.match(/^ {4}event:\s*["']?([^"'\n]+)["']?\s*$/m)?.[1]?.trim();
    const command = block.match(/^ {4}command:\s*(.*)$/m)?.[1]?.trim();
    if (!id || !/^[a-z0-9-]+$/.test(id)) {
      return { ...emptyResult, error: '每个 Hook 的 id 仅支持小写字母、数字和连字符' };
    }
    if (!description) return { ...emptyResult, error: `Hook「${id}」缺少 description` };
    if (!event) return { ...emptyResult, error: `Hook「${id}」缺少 event` };
    if (command === undefined) return { ...emptyResult, error: `Hook「${id}」缺少 command` };
    if ((command === '|' || command === '>') && !/^ {6}\S/m.test(block)) {
      return { ...emptyResult, error: `Hook「${id}」的 command 不能为空` };
    }
    if (!command && !/^ {6}\S/m.test(block)) {
      return { ...emptyResult, error: `Hook「${id}」的 command 不能为空` };
    }
    hookKeys.push(`${event} #${hookKeys.length + 1}`);
  }
  return { error: '', hookKeys, hookCount: hookKeys.length };
};

const getAssetDescription = (asset: AgentConfigAsset) =>
  asset.description?.trim() || summarizeMarkdown(asset.contentMd) || '-';

const getAssetFileName = (asset: AgentConfigAsset) => {
  if (asset.kind === 'hook') return asset.fileName || 'hooks.yaml';
  if (asset.fileName) return asset.fileName;
  if (asset.kind === 'entry') {
    return KIND_META.entry.files.find((fileName) => asset.name.includes(fileName)) || 'CLAUDE.md';
  }
  return `${asset.slug}.md`;
};

const getUserHookSettingsPaths = (asset: AgentConfigAsset) => {
  const paths = asset.targetClients.flatMap((client) => {
    if (client === 'claude_code') return ['~/.claude/settings.json'];
    if (client === 'codebuddy') return ['~/.codebuddy/settings.json'];
    if (client === 'codex') return ['~/.codex/hooks.json'];
    if (client === 'workbuddy') return ['~/.workbuddy/settings.json'];
    return [];
  });
  return paths.length > 0 ? paths : ['目标 Agent 用户配置'];
};

const getAssetFilePath = (asset: AgentConfigAsset) => {
  const fileName = getAssetFileName(asset);
  if (asset.kind === 'hook') return `hooks/${fileName}`;
  if (asset.kind === 'rule') return `rules/${fileName}`;
  return fileName;
};

const formatContentSize = (content: string) => {
  const bytes = new TextEncoder().encode(content).byteLength;
  return bytes < 1024 ? `${bytes} B` : `${(bytes / 1024).toFixed(2)} KB`;
};

const isValidSemver = (version: string) => /^\d+\.\d+\.\d+$/.test(version);

const compareSemver = (left: string, right: string) => {
  const leftParts = left.split('.').map(Number);
  const rightParts = right.split('.').map(Number);
  for (let index = 0; index < 3; index += 1) {
    const diff = (leftParts[index] || 0) - (rightParts[index] || 0);
    if (diff !== 0) return diff;
  }
  return 0;
};

const getScopeLabels = (asset: AgentConfigAsset): string[] => {
  if (asset.scope === 'public' || asset.groupIds.length === 0) return ['全部用户'];
  return asset.groupIds.map(
    (groupId) =>
      MOCK_GROUPS.find((group) => group.id === groupId)?.name ||
      MOCK_PROJECT_GROUPS.find((project) => project.id === groupId)?.name ||
      groupId
  );
};

const getTriggerLabel = (selectedScopes: Set<string>, allScopeKeys: string[]) => {
  if (selectedScopes.size === 0) return undefined;
  if (selectedScopes.size === allScopeKeys.length && allScopeKeys.every((key) => selectedScopes.has(key))) {
    return '全部应用范围';
  }
  return Array.from(selectedScopes)
    .map((scope) => (scope === 'public' ? '全部用户' : MOCK_GROUPS.find((group) => group.id === scope)?.name || scope))
    .join('、');
};

const buildAssetInstances = (asset?: AgentConfigAsset): AgentInstance[] => {
  if (!asset) return MOCK_OPENCLAW_INSTANCES;
  const candidates = asset.kind === 'hook'
    ? MOCK_OPENCLAW_INSTANCES.filter((instance) => {
        if (instance.agentType !== 'LocalAgent' || !instance.localProduct) return false;
        return asset.targetClients.includes(instance.localProduct.toLowerCase() as TargetClient);
      })
    : MOCK_OPENCLAW_INSTANCES;
  const previousVersion = `${Math.max(Number(asset.version) - 1, 0)}`;
  return candidates.map((instance, index) => {
    if (index % 5 === 0) {
      return {
        ...instance,
        distributionStatus: 'failed',
        distributedVersion: undefined,
        failReason: '目标 Agent 暂不满足下发条件，本次未完成下发',
      };
    }
    if (index % 3 === 0) {
      return {
        ...instance,
        distributionStatus: 'success',
        distributedVersion: previousVersion,
        failReason: undefined,
      };
    }
    return {
      ...instance,
      distributionStatus: 'not_distributed',
      distributedVersion: undefined,
      failReason: undefined,
    };
  });
};

const getAssetDistributionSummary = (asset: AgentConfigAsset) => {
  let statusLine1 = asset.enabled ? '正常' : '停用';
  let statusLine2 = '未下发';
  let statusVariant: StatusVariant = asset.enabled ? 'green' : 'red';
  let hasDistribution = false;

  if (asset.lastTaskStatus === 'running') {
    statusLine1 = '下发中';
    statusLine2 = '0%';
    statusVariant = 'blue';
    hasDistribution = true;
  } else if (asset.lastTaskStatus === 'installed' || asset.lastTaskStatus === 'failed') {
    const instances = buildAssetInstances(asset);
    const total = instances.length;
    const success = asset.lastTaskStatus === 'failed'
      ? 0
      : instances.filter((instance) => instance.distributionStatus === 'success').length;
    statusLine2 = `已下发（${success}/${total}成功）`;
    hasDistribution = true;
  }

  return {
    statusLine1,
    statusLine2,
    statusVariant,
    hasDistribution,
  };
};

interface StandardsAssetDetailProps {
  asset: AgentConfigAsset;
  onBack: () => void;
  onDistribute: () => void;
  onUpdate: () => void;
  onDownload: () => void;
  onUninstall: () => void;
  onDelete: () => void;
}

function StandardsAssetDetail({
  asset,
  onBack,
  onDistribute,
  onUpdate,
  onDownload,
  onUninstall,
  onDelete,
}: StandardsAssetDetailProps) {
  const [activeTab, setActiveTab] = useState('overview');
  const [recordTypeFilter, setRecordTypeFilter] = useState<'all' | 'distribute' | 'delete'>('all');
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [activeRecordId, setActiveRecordId] = useState<string | null>(null);
  const [detailSearchQuery, setDetailSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | DistributionStatus>('all');
  const description = getAssetDescription(asset);
  const assetFileName = getAssetFileName(asset);
  const assetFilePath = getAssetFilePath(asset);
  const content = asset.contentMd.startsWith('已上传文件') || asset.contentMd.startsWith('已更新文件')
    ? `# ${asset.name}\n\n${description}`
    : asset.contentMd;
  const detailInstances = buildAssetInstances(asset).slice(0, 8);
  const distributedInstances = detailInstances.filter((instance) => instance.distributionStatus === 'success');
  const hasInProgress = asset.lastTaskStatus === 'running';
  const distributionRecords = [
    ...(asset.lastTaskStatus !== 'pending' && asset.lastTaskStatus !== 'skipped' && asset.lastTaskStatus !== 'unsupported'
      ? [{
          id: `${asset.id}-distribute-latest`,
          type: 'distribute' as const,
          status: asset.lastTaskStatus === 'running' ? 'distributing' : 'completed',
          timestamp: asset.updatedAt,
          instances: detailInstances,
        }]
      : []),
    ...(distributedInstances.length > 0
      ? [{
          id: `${asset.id}-delete-preview`,
          type: 'delete' as const,
          status: 'completed',
          timestamp: new Date(asset.updatedAt.getTime() - 2 * 60 * 60 * 1000),
          instances: distributedInstances,
        }]
      : []),
  ];
  const filteredRecords = distributionRecords.filter((record) => recordTypeFilter === 'all' || record.type === recordTypeFilter);
  const activeRecord = distributionRecords.find((record) => record.id === activeRecordId);
  const filteredRecordInstances = activeRecord
    ? activeRecord.instances.filter((instance) => {
        const keyword = detailSearchQuery.trim().toLowerCase();
        const matchesSearch = !keyword || instance.name.toLowerCase().includes(keyword) || instance.id.toLowerCase().includes(keyword);
        const matchesStatus = statusFilter === 'all' || instance.distributionStatus === statusFilter;
        return matchesSearch && matchesStatus;
      })
    : [];

  return (
    <div className="space-y-4">
      <header className="flex flex-col gap-4">
        <BackButton onClick={onBack} className="self-start">返回上级</BackButton>

        <div className="rounded-xl border border-gray-200 bg-white p-6">
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-3">
                <TenantDocTitle as="h1">{asset.name}</TenantDocTitle>
                <StatusTag mode="fill" variant={asset.enabled ? 'green' : 'gray'}>
                  {asset.enabled ? '正常' : '停用'}
                </StatusTag>
              </div>
              <MetaText as="p" tone="weak" className="mt-1 font-mono">slug：{asset.slug}</MetaText>
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <StatusTag mode="fill" variant="gray" className="font-mono">v{asset.version}</StatusTag>
                <BodyText as="span" tone="weak">｜</BodyText>
                <BodyText as="span" tone="secondary">范围：{getScopeLabels(asset).join('、')}</BodyText>
              </div>
              <BodyText as="p" tone="secondary" className="mt-2 leading-5">
                {description}
              </BodyText>
            </div>

            <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">
              <Button variant="claw-outline" size="claw" onClick={onUpdate} disabled={hasInProgress}>
                <RefreshCw className="size-4" />
                更新
              </Button>
              <Button variant="claw-outline" size="claw" onClick={onDownload}>
                <Download className="size-4" />
                下载
              </Button>
              <Button variant="outline-destructive" size="claw" onClick={onDelete} disabled={hasInProgress}>
                <Trash2 className="size-4" />
                删除
              </Button>
              <Button
                variant="claw-outline"
                size="claw"
                onClick={onUninstall}
                disabled={hasInProgress || distributedInstances.length === 0}
                className={hasInProgress || distributedInstances.length === 0 ? 'opacity-50 cursor-not-allowed' : ''}
              >
                <Trash2 className="size-4" />
                批量卸载
              </Button>
              <Button variant="claw-primary" size="claw" onClick={onDistribute} disabled={hasInProgress}>
                {hasInProgress ? '下发中...' : '批量下发'}
                <Send className="size-4" />
              </Button>
            </div>
          </div>
        </div>
      </header>

      <div className="flex items-center justify-start">
        <SegmentedTabs
          tabs={[
            { id: 'overview', label: '概述' },
            { id: 'files', label: '文件列表' },
            { id: 'distribution', label: '下发和卸载记录' },
          ]}
          active={activeTab}
          onChange={setActiveTab}
          ariaLabel="企业规范详情 Tab 切换"
        />
        {activeTab === 'distribution' && (
          <Select value={recordTypeFilter} onValueChange={(value) => setRecordTypeFilter(value as 'all' | 'distribute' | 'delete')}>
            <SelectTrigger className="ml-4 h-8 w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部</SelectItem>
              <SelectItem value="distribute">下发记录</SelectItem>
              <SelectItem value="delete">卸载记录</SelectItem>
            </SelectContent>
          </Select>
        )}
      </div>

      {activeTab === 'overview' && (
        <SurfaceCard className="p-6">
          {asset.kind === 'hook' ? (
            <div className="space-y-4">
              <Alert variant="default">
                <AlertInfoIcon />
                <AlertDescription>
                  下发范围：指定用户或用户组。本地组件会将 hooks.yaml 转换为目标 Agent 支持的用户级 Hook 配置，并保留已有设置。
                </AlertDescription>
              </Alert>
              <div className="flex flex-wrap gap-2">
                {getUserHookSettingsPaths(asset).map((path) => (
                  <Badge key={path} variant="outline" className="font-mono">{path}</Badge>
                ))}
                <Badge variant="outline">{asset.hookCount || inspectHookManifestYaml(content).hookCount} 个 Hook</Badge>
                {asset.targetClients.map((client) => <Badge key={client} variant="secondary">{client}</Badge>)}
              </div>
              <pre className="max-h-[520px] overflow-auto rounded-lg border border-gray-200 bg-[#F8FAFC] p-4 text-xs leading-6 text-[#334155]"><code>{content}</code></pre>
            </div>
          ) : (
            <MDXRenderer content={content} />
          )}
        </SurfaceCard>
      )}

      {activeTab === 'files' && (
        <div className="space-y-3">
          <SurfaceCard className="grid gap-4 p-4 sm:grid-cols-2 xl:grid-cols-5">
            <div className="min-w-0">
              <MetaText as="p" tone="weak">文件名</MetaText>
              <BodyMedium as="p" tone="primary" className="mt-1 truncate font-mono">{assetFileName}</BodyMedium>
            </div>
            <div className="min-w-0 sm:col-span-2 xl:col-span-1">
              <MetaText as="p" tone="weak">清单路径</MetaText>
              <BodyText as="p" tone="secondary" className="mt-1 truncate font-mono">{assetFilePath}</BodyText>
            </div>
            <div>
              <MetaText as="p" tone="weak">文件类型</MetaText>
              <BodyText as="p" tone="secondary" className="mt-1">{asset.kind === 'hook' ? 'YAML' : 'Markdown'}</BodyText>
            </div>
            <div>
              <MetaText as="p" tone="weak">文件大小</MetaText>
              <BodyText as="p" tone="secondary" className="mt-1 tabular-nums">{formatContentSize(asset.contentMd)}</BodyText>
            </div>
            <div className="min-w-0">
              <MetaText as="p" tone="weak">校验和</MetaText>
              <BodyText as="p" tone="secondary" className="mt-1 truncate font-mono">{asset.checksum}</BodyText>
            </div>
          </SurfaceCard>

          <FileBrowser
            versions={[{
              version: asset.version,
              date: asset.updatedAt.toISOString().slice(0, 10),
              isLatest: true,
            }]}
            files={[{ name: asset.kind === 'hook' ? assetFileName : assetFilePath }]}
            getFileContent={() => asset.contentMd}
            height="36rem"
            defaultVersion={asset.version}
            defaultFile={asset.kind === 'hook' ? assetFileName : assetFilePath}
            showDownload
            onDownload={onDownload}
          />
        </div>
      )}

      {activeTab === 'distribution' && (
        <div className="space-y-3">
          {filteredRecords.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <HelperText as="p">还没有下发和卸载记录</HelperText>
            </div>
          ) : (
            filteredRecords.map((record, index) => {
              const totalCount = record.instances.length;
              const successCount = record.instances.filter((instance) => instance.distributionStatus === 'success').length;
              const failedCount = record.instances.filter((instance) => instance.distributionStatus === 'failed').length;
              const progress = totalCount > 0 ? Math.round((successCount / totalCount) * 100) : 0;
              const isDeleteRecord = record.type === 'delete';
              const isInProgress = record.status === 'distributing';

              return (
                <div key={record.id} className="rounded-lg border border-gray-200 bg-white p-4">
                  <div className="mb-3 flex items-start justify-between gap-3">
                    <BodyMedium as="p" tone="primary" className="font-semibold">
                      #{index + 1} · {isDeleteRecord ? '卸载' : '下发'} · v{asset.version} {record.timestamp.toLocaleString('zh-CN')}
                    </BodyMedium>
                    <div className="flex items-center gap-2">
                      <span className={`inline-block rounded px-3 py-1 text-xs font-medium ${
                        isInProgress ? 'bg-blue-50 text-blue-700' :
                        failedCount === 0 ? 'bg-green-50 text-green-700' :
                        successCount === 0 ? 'bg-red-50 text-red-700' :
                        'bg-yellow-50 text-yellow-700'
                      }`}>
                        {isInProgress
                          ? `${isDeleteRecord ? '卸载' : '下发'}中 ${progress}%`
                          : `${isDeleteRecord ? '卸载' : '下发'}完成，${successCount}个成功，${failedCount}个失败`}
                      </span>
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => {
                          setActiveRecordId(record.id);
                          setDetailSearchQuery('');
                          setStatusFilter('all');
                          setDetailsOpen(true);
                        }}
                        className="h-auto px-2 py-1 text-blue-600 hover:text-blue-700"
                      >
                        查看详情
                      </Button>
                    </div>
                  </div>
                  {isInProgress && (
                    <div className="h-2 w-full rounded-full bg-gray-200">
                      <div className="h-2 rounded-full bg-blue-600 transition-all duration-300" style={{ width: `${progress}%` }} />
                    </div>
                  )}
                </div>
              );
            })
          )}
        </div>
      )}

      <Dialog open={detailsOpen} onOpenChange={setDetailsOpen}>
        <DialogContent className="flex max-h-[80vh] flex-col sm:max-w-[720px]">
          <DialogHeader>
            <DialogTitle>{activeRecord?.type === 'delete' ? '卸载详情' : '下发详情'}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[#A3A3A3]" />
                <Input
                  value={detailSearchQuery}
                  onChange={(event) => setDetailSearchQuery(event.target.value)}
                  placeholder="搜索实例名称/ID..."
                  className="h-9 pl-10"
                />
              </div>
              <Select value={statusFilter} onValueChange={(value) => setStatusFilter(value as 'all' | DistributionStatus)}>
                <SelectTrigger className="w-28">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部</SelectItem>
                  <SelectItem value="success">成功</SelectItem>
                  <SelectItem value="failed">失败</SelectItem>
                  <SelectItem value="distributing">{activeRecord?.type === 'delete' ? '卸载中' : '下发中'}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="max-h-64 overflow-y-auto rounded-[4px] border border-gray-200">
              <Table>
                <TableHeader className="sticky top-0 border-b border-gray-200 bg-gray-50">
                  <TableRow>
                    <TableHead className="text-left">实例名称</TableHead>
                    <TableHead className="min-w-[140px] text-left">实例ID</TableHead>
                    <TableHead className="text-left">状态</TableHead>
                    <TableHead className="text-left">失败原因</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredRecordInstances.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={4} className="text-center">
                        <BodyText as="span" tone="muted">没有符合条件的记录</BodyText>
                      </TableCell>
                    </TableRow>
                  ) : (
                    filteredRecordInstances.map((instance) => (
                      <TableRow key={instance.id}>
                        <TableCell><BodyText as="span" tone="primary">{instance.name}</BodyText></TableCell>
                        <TableCell><MetaText className="font-mono whitespace-nowrap">{instance.id}</MetaText></TableCell>
                        <TableCell>
                          <span className={`inline-block rounded px-2 py-1 text-xs font-medium ${
                            DISTRIBUTION_STATUS_MAP[instance.distributionStatus || 'not_distributed']?.color || 'bg-gray-50 text-gray-500'
                          }`}>
                            {DISTRIBUTION_STATUS_MAP[instance.distributionStatus || 'not_distributed']?.label || '未下发'}
                          </span>
                        </TableCell>
                        <TableCell>
                          <BodyText as="span" tone="muted">{instance.failReason || '-'}</BodyText>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

export default function StandardsLibraryTab() {
  const [assets, setAssets] = useState<AgentConfigAsset[]>(() => standardsStore.getAll());
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedScopes, setSelectedScopes] = useState<Set<string>>(new Set());
  const [scopeDropdownOpen, setScopeDropdownOpen] = useState(false);
  const [viewMode, setViewMode] = useState<'card' | 'list'>('list');
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<AgentConfigAsset | null>(null);
  const [instanceDialogOpen, setInstanceDialogOpen] = useState(false);
  const [uninstallDialogOpen, setUninstallDialogOpen] = useState(false);
  const [activeAssetId, setActiveAssetId] = useState<string | null>(null);
  const [updateTarget, setUpdateTarget] = useState<AgentConfigAsset | null>(null);
  const [selectedAssetId, setSelectedAssetId] = useState<string | null>(null);
  const [draftKind, setDraftKind] = useState<AssetKind>('rule');
  const [draftFile, setDraftFile] = useState<UploadedFile | null>(null);
  const [draftFileExpanded, setDraftFileExpanded] = useState(false);
  const [hookForm, setHookForm] = useState<HookFormValue>({ ...EMPTY_HOOK_FORM });
  const [updateFile, setUpdateFile] = useState<UploadedFile | null>(null);
  const [updateFileExpanded, setUpdateFileExpanded] = useState(false);
  const [createFormData, setCreateFormData] = useState({
    slug: '',
    name: '',
    description: '',
    version: '1.0.0',
    scope: 'public' as SkillScope,
    groupIds: [] as string[],
  });
  const [updateFormData, setUpdateFormData] = useState({
    name: '',
    description: '',
    version: '',
    changeLog: '',
    scope: 'public' as SkillScope,
    groupIds: [] as string[],
  });
  const [updateVersionError, setUpdateVersionError] = useState('');
  const allScopeKeys = useMemo(() => ['public', ...MOCK_GROUPS.map((group) => group.id)], []);

  // 标记自身写入，避免订阅回调造成循环
  const isSelfWritingAssets = useRef(false);
  // assets 变化时同步到共享 standardsStore（localStorage + 广播）
  useEffect(() => {
    isSelfWritingAssets.current = true;
    standardsStore.replaceAll(assets);
    isSelfWritingAssets.current = false;
  }, [assets]);

  // 订阅 standardsStore：其他模块（含「项目资产管理」联动）变更时刷新
  useEffect(() => standardsStore.subscribe(() => {
    if (isSelfWritingAssets.current) return;
    setAssets(standardsStore.getAll());
  }), []);

  const activeAsset = useMemo(
    () => assets.find((asset) => asset.id === activeAssetId),
    [activeAssetId, assets],
  );
  const selectedAsset = useMemo(
    () => assets.find((asset) => asset.id === selectedAssetId),
    [assets, selectedAssetId],
  );

  const activeAssetInstances = useMemo(
    () => buildAssetInstances(activeAsset),
    [activeAsset],
  );
  const hasSuccessfulDraft = draftKind === 'hook' || draftFile?.status === 'success';

  const uninstallInstances = useMemo(
    () => activeAssetInstances
      .filter((instance) => instance.distributionStatus === 'success')
      .map((instance) => {
        const groupName = instance.groupIds?.[0]
          ? MOCK_GROUPS.find((group) => group.id === instance.groupIds[0])?.name
          : undefined;
        return {
          id: instance.id,
          name: instance.name,
          createdBy: instance.createdBy || 'admin',
          groupName: groupName || '全部用户',
          distributedVersion: instance.distributedVersion || activeAsset?.version,
          deleteStatus: instance.failReason ? 'delete_failed' as const : 'not_deleted' as const,
          deleteFailReason: instance.failReason,
        };
      }),
    [activeAsset?.version, activeAssetInstances],
  );

  const sortedAssets = useMemo(() => {
    const keyword = searchQuery.trim().toLowerCase();
    return assets
      .filter((asset) => {
        const matchesSearch =
          !keyword ||
          asset.name.toLowerCase().includes(keyword) ||
          asset.slug.toLowerCase().includes(keyword) ||
          (asset.description || '').toLowerCase().includes(keyword) ||
          asset.contentMd.toLowerCase().includes(keyword);
        let matchesScope = true;
        if (selectedScopes.size > 0) {
          const hasPublic = selectedScopes.has('public');
          const groupScopes = Array.from(selectedScopes).filter((scope) => scope !== 'public');
          const matchPublic = hasPublic && (asset.scope === 'public' || asset.groupIds.length === 0);
          const matchGroup = groupScopes.length > 0 && asset.scope === 'private' && asset.groupIds.some((groupId) => selectedScopes.has(groupId));
          matchesScope = matchPublic || matchGroup;
        }
        return matchesSearch && matchesScope;
      })
      .sort((a, b) => b.updatedAt.getTime() - a.updatedAt.getTime());
  }, [assets, searchQuery, selectedScopes]);

  const resetDraft = () => {
    setDraftKind('rule');
    setDraftFile(null);
    setDraftFileExpanded(false);
    setHookForm({ ...EMPTY_HOOK_FORM });
    setCreateFormData({
      slug: '',
      name: '',
      description: '',
      version: '1.0.0',
      scope: 'public',
      groupIds: [],
    });
  };

  const handleDraftKindChange = (kind: AssetKind) => {
    if (kind === draftKind) return;
    setDraftFile(null);
    setDraftFileExpanded(false);
    setHookForm({ ...EMPTY_HOOK_FORM });
    setCreateFormData({
      slug: '',
      name: '',
      description: '',
      version: '1.0.0',
      scope: 'public',
      groupIds: [],
    });
    setDraftKind(kind);
  };

  const handleDraftFileUpload = async (files: FileList) => {
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
    setCreateFormData((prev) => ({
      ...prev,
      name: prev.name || name,
      slug: prev.slug || createSlug(name),
      description: prev.description || `${name} 企业规范配置`,
    }));
    toast.success(`已选择文件「${file.name}」`);
  };

  const handleUpdateFileUpload = async (files: FileList) => {
    const file = files?.[0];
    if (!file) return;
    const updatingHook = updateTarget?.kind === 'hook';
    if (updatingHook ? !isYamlFile(file.name) : !file.name.toLowerCase().endsWith('.md')) {
      toast.error(updatingHook ? 'Hook 配置只支持 .yaml 或 .yml 文件' : '该资源只支持上传 .md 文件');
      return;
    }
    const content = await file.text();
    if (updatingHook) {
      const inspection = inspectHookManifestYaml(content);
      if (inspection.error) {
        toast.error(inspection.error);
        return;
      }
    }
    setUpdateFile({
      name: file.name,
      size: file.size,
      status: 'success',
      files: [{ name: file.name, size: file.size, content }],
    });
    setUpdateFileExpanded(true);
    if (!updatingHook && !updateFormData.description.trim()) {
      setUpdateFormData((prev) => ({ ...prev, description: summarizeMarkdown(content) || `${file.name.replace(/\.md$/i, '')} 企业规范配置` }));
    }
    toast.success(updatingHook
      ? `已选择「${file.name}」，保存后作为 Hook 配置资源管理`
      : `已选择更新文件「${file.name}」`);
  };

  const handleUpdateFileRemove = () => {
    setUpdateFile(null);
    setUpdateFileExpanded(false);
  };

  const validateUpdateVersion = (version: string, currentVersion: string) => {
    if (!version) return '';
    if (!isValidSemver(version)) return '版本号格式必须为 x.y.z';
    if (compareSemver(version, currentVersion) <= 0) return `新版本号需高于上个版本号 ${currentVersion}`;
    return '';
  };

  const handleUpdateVersionChange = (version: string) => {
    setUpdateFormData((prev) => ({ ...prev, version }));
    setUpdateVersionError(updateTarget ? validateUpdateVersion(version, updateTarget.version) : '');
  };

  const handleCreateAsset = () => {
    if (draftKind !== 'hook' && (!draftFile || draftFile.status !== 'success')) {
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
    if (!createFormData.slug.trim() || !createFormData.name.trim() || !createFormData.version.trim()) {
      toast.error('请填写完整的资源信息');
      return;
    }
    if (!/^[a-z0-9-]+$/.test(createFormData.slug.trim())) {
      toast.error('唯一标识仅支持小写字母、数字和连字符');
      return;
    }
    if (!isValidSemver(createFormData.version)) {
      toast.error('版本号格式必须为 x.y.z');
      return;
    }

    const uploadedContent = draftKind === 'hook'
      ? buildHookManifestYaml(hookForm, {
          id: createFormData.slug,
          description: createFormData.description || createFormData.name,
        })
      : draftFile?.files?.[0]?.content || '';
    const hookInspection = draftKind === 'hook' ? inspectHookManifestYaml(uploadedContent) : null;
    if (hookInspection?.error) {
      toast.error(hookInspection.error);
      return;
    }
    const asset: AgentConfigAsset = {
      id: `asset-${Date.now()}`,
      tenantId: 'tenant-openclaw',
      name: createFormData.name,
      slug: createFormData.slug,
      kind: draftKind,
      targetClients: KIND_CLIENTS[draftKind],
      contentMd: uploadedContent || createFormData.description || `已上传文件：${draftFile?.name || 'hooks.yaml'}`,
      fileName: draftKind === 'hook' ? 'hooks.yaml' : (draftFile?.name || ''),
      hookCount: hookInspection?.hookCount,
      description: createFormData.description,
      version: createFormData.version,
      visibilityType: createFormData.scope === 'public' ? 'all' : 'group',
      scope: createFormData.scope,
      groupIds: createFormData.scope === 'public' ? [] : createFormData.groupIds,
      enabled: true,
      alwaysApply: true,
      pathGlobs: [],
      checksum: `sha256:${Math.random().toString(16).slice(2, 8)}`,
      createdBy: '当前管理员',
      updatedAt: new Date(),
      lastTaskStatus: 'pending',
    };
    setAssets((prev) => [asset, ...prev]);
    setCreateDialogOpen(false);
    resetDraft();
    toast.success(`${KIND_META[draftKind].label}「${createFormData.name}」已创建，可选择本地 Agent 下发`);
  };

  const handleDeleteAsset = () => {
    if (!deleteTarget) return;
    const deletedId = deleteTarget.id;
    setAssets((prev) => prev.filter((asset) => asset.id !== deleteTarget.id));
    projectAssetStore.onLibraryItemDeleted('enterpriseStandard', deletedId);
    toast.success(`资产「${deleteTarget.name}」已删除`);
    if (selectedAssetId === deleteTarget.id) setSelectedAssetId(null);
    setDeleteTarget(null);
  };

  const openInstanceDialog = (assetId: string) => {
    setActiveAssetId(assetId);
    setInstanceDialogOpen(true);
  };

  const openUninstallDialog = (assetId: string) => {
    setActiveAssetId(assetId);
    setUninstallDialogOpen(true);
  };

  const handleDistributionStart = (selectedInstanceIds: string[]) => {
    if (!activeAsset) return;
    const distributingAssetId = activeAsset.id;
    setAssets((prev) =>
      prev.map((asset) =>
        asset.id === distributingAssetId
          ? {
              ...asset,
              lastTaskStatus: 'running',
              updatedAt: new Date(),
            }
          : asset,
      ),
    );
    toast.success(activeAsset.kind === 'hook'
      ? '已按用户应用范围发起 Hook 配置下发'
      : `已开始向 ${selectedInstanceIds.length} 个实例执行下发`);
    window.setTimeout(() => {
      setAssets((prev) =>
        prev.map((asset) =>
          asset.id === distributingAssetId
            ? {
                ...asset,
                lastTaskStatus: 'installed',
                updatedAt: new Date(),
              }
            : asset,
        ),
      );
      toast.success(activeAsset.kind === 'hook' ? 'Hook 配置已下发并适配到目标 Agent 的用户级配置' : '下发成功');
    }, 1200);
  };

  const handleOpenUpdateDialog = (asset: AgentConfigAsset) => {
    setUpdateTarget(asset);
    setUpdateFormData({
      name: asset.name,
      description: getAssetDescription(asset),
      version: '',
      changeLog: '',
      scope: asset.scope,
      groupIds: [...asset.groupIds],
    });
    setUpdateVersionError('');
    setUpdateFile(null);
    setUpdateFileExpanded(false);
  };

  const handleConfirmUpdateFile = () => {
    if (!updateTarget) return;
    if (!updateFormData.version) {
      toast.error('请填写新版本号');
      return;
    }
    const versionError = validateUpdateVersion(updateFormData.version, updateTarget.version);
    if (versionError) {
      setUpdateVersionError(versionError);
      return;
    }
    const hasNewUpload = updateFile && updateFile.status === 'success';
    const uploadedContent = hasNewUpload ? updateFile.files?.[0]?.content : undefined;
    const hookInspection = updateTarget.kind === 'hook' && uploadedContent ? inspectHookManifestYaml(uploadedContent) : null;
    if (hookInspection?.error) {
      toast.error(hookInspection.error);
      return;
    }
    setAssets((prev) =>
      prev.map((asset) =>
        asset.id === updateTarget.id
          ? {
              ...asset,
              name: updateFormData.name,
              contentMd: uploadedContent || asset.contentMd,
              fileName: hasNewUpload ? updateFile.name : asset.fileName,
              hookCount: hookInspection?.hookCount || asset.hookCount,
              description: updateFormData.description,
              version: updateFormData.version,
              visibilityType: updateFormData.scope === 'public' ? 'all' : 'group',
              scope: updateFormData.scope,
              groupIds: updateFormData.scope === 'public' ? [] : updateFormData.groupIds,
              checksum: `sha256:${Math.random().toString(16).slice(2, 8)}`,
              updatedAt: new Date(),
              lastTaskStatus: 'pending',
            }
          : asset,
      ),
    );
    toast.success(`「${updateTarget.name}」已更新至 v${updateFormData.version}`);
    setUpdateTarget(null);
    setUpdateFile(null);
    setUpdateVersionError('');
  };

  const handleUninstallStart = (selectedInstanceIds: string[]) => {
    if (!activeAsset) return;
    toast.success(`资产「${activeAsset.name}」已开始从 ${selectedInstanceIds.length} 个实例卸载`);
  };

  const handleScopeUpdate = (assetId: string, scope: SkillScope, groupIds: string[]) => {
    setAssets((prev) => prev.map((asset) =>
      asset.id === assetId
        ? { ...asset, scope, visibilityType: scope === 'public' ? 'all' : 'group', groupIds }
        : asset,
    ));
    toast.success('应用范围修改成功');
  };

  const handleDownload = (asset: AgentConfigAsset) => {
    const fileName = asset.kind === 'hook' ? (asset.fileName || 'hooks.yaml') : (asset.fileName || `${asset.slug}.md`);
    const blob = new Blob([asset.contentMd], { type: asset.kind === 'hook' ? 'application/yaml;charset=utf-8' : 'text/markdown;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = fileName;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
    toast.success(`资产「${asset.name}」已下载`);
  };

  return (
    <div className="space-y-4">
      {selectedAsset ? (
        <StandardsAssetDetail
          asset={selectedAsset}
          onBack={() => setSelectedAssetId(null)}
          onDistribute={() => openInstanceDialog(selectedAsset.id)}
          onUpdate={() => handleOpenUpdateDialog(selectedAsset)}
          onDownload={() => handleDownload(selectedAsset)}
          onUninstall={() => openUninstallDialog(selectedAsset.id)}
          onDelete={() => setDeleteTarget(selectedAsset)}
        />
      ) : (
      <>
      <div className="flex items-center gap-3">
        <div className="relative min-w-0 flex-1">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[var(--text-weak)]" />
          <Input
            placeholder="搜索规范名称、标识或文件名..."
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            className="pl-10"
          />
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <ScopeSelect
            withTrigger
            groups={MOCK_GROUPS}
            value={selectedScopes}
            onChange={setSelectedScopes}
            triggerLabel={getTriggerLabel(selectedScopes, allScopeKeys)}
            triggerPlaceholder="选择应用范围"
            triggerClassName="flex flex-none items-center justify-between gap-1 w-[12rem] h-9 px-3 border border-[var(--cp-border)] rounded-[var(--radius-lg)] bg-[var(--cp-surface)] text-sm text-[var(--text-emphasis)] hover:border-[var(--cp-brand-blue)] data-[state=open]:border-[var(--cp-brand-blue)] transition-colors outline-none"
            align="end"
            open={scopeDropdownOpen}
            onOpenChange={setScopeDropdownOpen}
          />
          {/* 视图切换
            * 停服态豁免：卡片/列表视图切换属于查看类操作（不产生变更），
            * 需保持 100% 不透明与正常交互。
            * SegmentOption 自身未设置 disabled，"停服前已禁用则延续禁用"
            * 约束通过组件 disabled 属性依然生效（此处无）。 */}
          <SegmentGroup data-billing-exempt>
            <SegmentOption active={viewMode === 'card'} onClick={() => setViewMode('card')} title="卡片视图">
              <Grid3x3 className="size-4" />
            </SegmentOption>
            <SegmentOption active={viewMode === 'list'} onClick={() => setViewMode('list')} title="列表视图">
              <List className="size-4" />
            </SegmentOption>
          </SegmentGroup>
          <Button variant="claw-primary" size="claw-sm" onClick={() => setCreateDialogOpen(true)}>
            <Upload className="size-4" />
            新增资源
          </Button>
        </div>
      </div>

      {sortedAssets.length === 0 && (
        <Empty className="border border-[var(--cp-border)] bg-[var(--cp-surface)]">
          <EmptyHeader>
            <EmptyTitle>暂无企业规范资源</EmptyTitle>
            <EmptyDescription>上传 Markdown 企业规范或通过表单创建 Hook 配置，并按用户应用范围下发到 Agent。</EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button variant="claw-primary" size="claw-sm" onClick={() => setCreateDialogOpen(true)}>
              <Upload className="size-4" />
              新增资源
            </Button>
          </EmptyContent>
        </Empty>
      )}

      {viewMode === 'card' && sortedAssets.length > 0 && (
        <div className="grid grid-cols-3 gap-4">
          {sortedAssets.map((asset) => {
            const description = getAssetDescription(asset);
            return (
              <div
                key={asset.id}
                onClick={() => setSelectedAssetId(asset.id)}
                className="relative flex cursor-pointer flex-col overflow-hidden rounded-[4px] border border-gray-200 bg-white p-4 transition-all hover:border-[#355EF1] group"
              >
                <div className="flex items-start justify-between gap-2 mb-3">
                  <div className="flex items-center gap-1.5 flex-1 min-w-0">
                    <CardHeading as="h3" className="truncate group-hover:text-[var(--text-brand)] transition-colors">{asset.name}</CardHeading>
                  </div>
                  <Badge variant="secondary" className="tabular-nums shrink-0">
                    v{asset.version}
                  </Badge>
                </div>

                <div className="flex flex-wrap gap-1 mb-3 items-center">
                  <Badge variant="outline">{KIND_META[asset.kind].label}</Badge>
                </div>

                <Tooltip delayDuration={1000}>
                  <TooltipTrigger asChild>
                    <MetaText
                      as="p"
                      tone="muted"
                      className="line-clamp-2 mb-3 cursor-default leading-relaxed min-h-[34px]"
                    >
                      {description}
                    </MetaText>
                  </TooltipTrigger>
                  {description.length > 60 && (
                    <TooltipContent side="bottom" className="max-w-[320px]">
                      <MetaText as="p" className="whitespace-pre-wrap">{description}</MetaText>
                    </TooltipContent>
                  )}
                </Tooltip>

                <div className="flex items-center gap-1 mb-3 flex-wrap" onClick={(event) => event.stopPropagation()}>
                  <MetaText tone="weak" className="mr-1">应用范围</MetaText>
                  <ScopeSelect
                    scope={asset.scope === 'public' || asset.groupIds.length === 0 ? 'all' : 'groups'}
                    selectedGroupIds={asset.groupIds}
                    groups={MOCK_GROUPS}
                    projects={MOCK_PROJECT_GROUPS}
                    scopeLabels={getScopeLabels(asset)}
                    maxVisibleBadges={3}
                    onConfirm={(scope, groupIds) => handleScopeUpdate(asset.id, scope === 'all' ? 'public' : 'groups', groupIds)}
                  />
                </div>

                <div className="mt-auto flex items-center gap-2 pt-3 border-t border-[#F5F5F5]" onClick={(event) => event.stopPropagation()}>
                  <Button variant="claw-outline" size="sm" className="h-8" onClick={() => openInstanceDialog(asset.id)}>
                    <Send className="size-3.5" />
                    下发
                  </Button>
                  <Button variant="claw-outline" size="sm" className="h-8" onClick={() => handleOpenUpdateDialog(asset)}>
                    <RefreshCw className="size-3.5" />
                    更新
                  </Button>
                  <div className="ml-auto">
                    <MoreActionsDropdown
                      triggerType="icon"
                      align="end"
                      items={[
                        {
                          label: '下载',
                          icon: Download,
                          onClick: () => handleDownload(asset),
                        },
                        {
                          label: '卸载',
                          icon: PackageX,
                          onClick: () => openUninstallDialog(asset.id),
                        },
                        {
                          label: '删除',
                          icon: Trash2,
                          onClick: () => setDeleteTarget(asset),
                          variant: 'destructive',
                        },
                      ]}
                    />
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {viewMode === 'list' && sortedAssets.length > 0 && (
        <SurfaceCard className="overflow-hidden">
          <Table variant="white" scrollX={1420}>
            <TableHeader>
              <TableRow>
                <TableHead fixed="left" style={{ width: 260, minWidth: 260 }}>资产信息</TableHead>
                <TableHead style={{ width: 100, minWidth: 100 }}>状态</TableHead>
                <TableHead style={{ width: 100, minWidth: 100 }}>类型</TableHead>
                <TableHead style={{ width: 140, minWidth: 140 }}>下发</TableHead>
                <TableHead style={{ width: 90, minWidth: 90 }}>版本</TableHead>
                <TableHead style={{ width: 360, minWidth: 360 }}>描述</TableHead>
                <TableHead style={{ width: 160, minWidth: 160 }}>应用范围</TableHead>
                <TableHead style={{ width: 130, minWidth: 130 }}>最后更新</TableHead>
                <TableHead fixed="right" style={{ width: 170, minWidth: 170 }}>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sortedAssets.map((asset) => {
                const description = getAssetDescription(asset);
                const distributionSummary = getAssetDistributionSummary(asset);
                return (
                  <TableRow key={asset.id} className="cursor-pointer" onClick={() => setSelectedAssetId(asset.id)}>
                    <TableCell fixed="left" style={{ width: 260 }}>
                      <div className="min-w-0">
                        <BodyMedium as="p" tone="primary" className="truncate">{asset.name}</BodyMedium>
                        <div className="mt-1 flex min-w-0 items-center gap-1.5">
                          <MetaText as="p" tone="weak" className="truncate font-mono">{asset.slug}</MetaText>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <StatusTag mode="text" variant={distributionSummary.statusVariant}>
                        {distributionSummary.statusLine1}
                      </StatusTag>
                    </TableCell>
                    <TableCell>
                      <BodyText as="span" tone="secondary">{TYPE_TEXT[asset.kind]}</BodyText>
                    </TableCell>
                    <TableCell>
                      {distributionSummary.hasDistribution ? (
                        <button
                          type="button"
                          onClick={(event) => {
                            event.stopPropagation();
                            setSelectedAssetId(asset.id);
                          }}
                          className="text-sm text-[var(--text-secondary)] transition-colors hover:text-[var(--text-brand)]"
                          title={distributionSummary.statusLine2}
                        >
                          {distributionSummary.statusLine2}
                        </button>
                      ) : (
                        <BodyText as="span" tone="weak">{distributionSummary.statusLine2}</BodyText>
                      )}
                    </TableCell>
                    <TableCell>
                      <BodyText as="span" tone="secondary">{asset.version}</BodyText>
                    </TableCell>
                    <TableCell style={{ width: 360, overflow: 'hidden' }}>
                      <Tooltip delayDuration={1000}>
                        <TooltipTrigger asChild>
                          <BodyText
                            as="span"
                            tone="secondary"
                            className="block cursor-default leading-relaxed"
                            style={{
                              display: '-webkit-box',
                              WebkitLineClamp: 2,
                              WebkitBoxOrient: 'vertical',
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              wordBreak: 'break-all',
                            }}
                          >
                            {description}
                          </BodyText>
                        </TooltipTrigger>
                        {description.length > 40 && (
                          <TooltipContent side="bottom" className="max-w-[400px]">
                            <MetaText as="p" className="whitespace-pre-wrap">{description}</MetaText>
                          </TooltipContent>
                        )}
                      </Tooltip>
                    </TableCell>
                    <TableCell onClick={(event) => event.stopPropagation()}>
                      <EditScopePopover
                        groups={MOCK_GROUPS}
                        projects={MOCK_PROJECT_GROUPS}
                        currentScope={asset.scope}
                        currentGroupIds={asset.groupIds}
                        scopeLabels={getScopeLabels(asset)}
                        isPublic={asset.scope === 'public' || asset.groupIds.length === 0}
                        onConfirm={(scope, groupIds) => handleScopeUpdate(asset.id, scope, groupIds)}
                      />
                    </TableCell>
                    <TableCell>
                      <BodyText as="span" tone="secondary" className="tabular-nums">
                        {asset.updatedAt.toLocaleDateString('zh-CN')}
                      </BodyText>
                    </TableCell>
                    <TableActionCell fixed="right" style={{ width: 170 }} onClick={(event) => event.stopPropagation()}>
                      <Button variant="link" onClick={() => openInstanceDialog(asset.id)}>
                        下发
                      </Button>
                      <Button variant="link" onClick={() => handleOpenUpdateDialog(asset)}>
                        更新
                      </Button>
                      <MoreActionsDropdown
                        triggerType="text"
                        align="end"
                        items={[
                          {
                            label: '下载',
                            icon: Download,
                            onClick: () => handleDownload(asset),
                          },
                          {
                            label: '卸载',
                            icon: PackageX,
                            onClick: () => openUninstallDialog(asset.id),
                          },
                          {
                            label: '删除',
                            icon: Trash2,
                            onClick: () => setDeleteTarget(asset),
                            variant: 'destructive',
                          },
                        ]}
                      />
                    </TableActionCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </SurfaceCard>
      )}
      </>
      )}

      <Dialog open={createDialogOpen} onOpenChange={(open) => {
        setCreateDialogOpen(open);
        if (!open) resetDraft();
      }}>
        <DialogContent size="lg" className="max-h-[min(90vh,780px)] flex flex-col" onPointerDownOutside={(event) => event.preventDefault()}>
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
                <MetaText as="label" tone="secondary">1. 资源类型 <span className="text-[var(--text-danger)]">*</span></MetaText>
                <MetaText as="p" tone="weak" className="mt-1">资源会根据类型自动适配可下发的 Agent。</MetaText>
              </div>
              <div className="grid gap-2 sm:grid-cols-3">
                {(Object.keys(KIND_META) as AssetKind[]).map((kind) => {
                  const active = draftKind === kind;
                  return (
                    <button
                      key={kind}
                      type="button"
                      onClick={() => handleDraftKindChange(kind)}
                      className={`flex min-h-[82px] min-w-0 items-start gap-2 overflow-hidden rounded-[4px] border p-3 text-left transition-colors ${
                        active
                          ? 'border-[#C7D7FE] bg-[#E8ECFE] text-[#1447E6]'
                          : 'border-[var(--cp-border)] bg-white text-[var(--text-secondary)] hover:border-[var(--cp-brand-blue)]'
                      }`}
                    >
                      {active ? <CheckCircle2 className="mt-0.5 size-4 shrink-0" /> : <span className="mt-0.5 size-4 shrink-0" />}
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
                        <MetaText as="span" tone="weak" className="mt-1 block leading-relaxed">{KIND_META[kind].desc}</MetaText>
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
                manifestId={createFormData.slug}
                manifestDescription={createFormData.description || createFormData.name}
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
                  onZipUpload={handleDraftFileUpload}
                  onRemove={() => {
                    setDraftFile(null);
                    setDraftFileExpanded(false);
                    setCreateFormData({
                      slug: '',
                      name: '',
                      description: '',
                      version: '1.0.0',
                      scope: 'public',
                      groupIds: [],
                    });
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
                    ? '填写资源名称和应用范围，Hook 配置由企业规范库统一管理。'
                    : '根据上传文件填写基础信息，资源由企业规范库统一管理。'}
                </MetaText>
              </div>

              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary" htmlFor="asset-create-slug">
                  唯一标识 (slug)<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                </MetaMedium>
                <Input
                  id="asset-create-slug"
                  value={createFormData.slug}
                  disabled={!hasSuccessfulDraft}
                  onChange={(event) => setCreateFormData((prev) => ({ ...prev, slug: event.target.value.toLowerCase().replace(/\s+/g, '-') }))}
                  placeholder="e.g., frontend-react-rules"
                />
                <MetaText tone="weak">
                  仅支持小写字母、数字和连字符；Hook 配置会将其写入 hooks.yaml 的 id。
                </MetaText>
              </div>

              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary" htmlFor="asset-create-name">
                  显示名称<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                </MetaMedium>
                <Input
                  id="asset-create-name"
                  value={createFormData.name}
                  disabled={!hasSuccessfulDraft}
                  onChange={(event) => setCreateFormData((prev) => ({ ...prev, name: event.target.value }))}
                  placeholder="e.g., 前端 React 规范"
                />
              </div>

              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary" htmlFor="asset-create-description">描述</MetaMedium>
                <Textarea
                  id="asset-create-description"
                  value={createFormData.description}
                  disabled={!hasSuccessfulDraft}
                  onChange={(event) => setCreateFormData((prev) => ({ ...prev, description: event.target.value }))}
                  placeholder="资源的简要描述"
                  rows={2}
                />
              </div>

              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary" htmlFor="asset-create-version">
                  版本号<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                </MetaMedium>
                <Input
                  id="asset-create-version"
                  value={createFormData.version}
                  disabled={!hasSuccessfulDraft}
                  onChange={(event) => setCreateFormData((prev) => ({ ...prev, version: event.target.value }))}
                  placeholder="e.g., 1.0.0"
                />
              </div>

              <div>
                <MetaMedium as="label" tone="secondary">应用范围</MetaMedium>
                <div className={`mt-2 ${hasSuccessfulDraft ? '' : 'pointer-events-none opacity-60'}`} aria-disabled={!hasSuccessfulDraft}>
                  <ScopeSelect
                    scope={createFormData.scope === 'public' ? 'all' : 'groups'}
                    selectedGroupIds={createFormData.groupIds}
                    groups={MOCK_GROUPS}
                    onConfirm={(scope, groupIds) => {
                      if (!hasSuccessfulDraft) return;
                      if (scope === 'all') {
                        setCreateFormData((prev) => ({ ...prev, scope: 'public', groupIds: [] }));
                      } else {
                        setCreateFormData((prev) => ({ ...prev, scope: 'private', groupIds }));
                      }
                    }}
                  />
                </div>
              </div>

            </div>

          </div>
          </DialogBody>
          <DialogFooter>
            <Button variant="claw-outline" onClick={() => setCreateDialogOpen(false)}>取消</Button>
            <Button
              variant="dialog-confirm"
              onClick={handleCreateAsset}
              disabled={
                !hasSuccessfulDraft ||
                (draftKind === 'hook' && !!getHookFormError(hookForm)) ||
                !createFormData.slug.trim() ||
                !createFormData.name.trim() ||
                !createFormData.version.trim()
              }
            >
              <CheckCircle2 className="size-4" />
              发布资源
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader>
            <DialogTitle>删除配置资产</DialogTitle>
          </DialogHeader>
          <BodyText as="p" tone="muted" className="py-2">
            确定要删除「<BodyMedium as="span" tone="primary">{deleteTarget?.name}</BodyMedium>」吗？删除后已下发实例不会自动卸载。
          </BodyText>
          <DialogFooter>
            <Button variant="claw-outline" onClick={() => setDeleteTarget(null)}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleDeleteAsset}>删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!updateTarget} onOpenChange={(open) => {
        if (!open) {
          setUpdateTarget(null);
          setUpdateFile(null);
          setUpdateFileExpanded(false);
          setUpdateVersionError('');
        }
      }}>
        <DialogContent className="sm:max-w-2xl overflow-visible" style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }} onPointerDownOutside={(event) => event.preventDefault()}>
          <DialogHeader>
            <DialogTitle>更新企业规范或 Hook 配置</DialogTitle>
          </DialogHeader>

          <DialogBody className="px-6 flex-1">
          <div className="space-y-4">
            <Alert variant="warning">
              <AlertInfoIcon />
              <AlertDescription>
                仅更新企业规范库中的文件版本。已下发至 Agent 实例的文件不会同步升级，需手动重新下发。
              </AlertDescription>
            </Alert>

            <div className="space-y-3">
              <MetaMedium as="label" tone="secondary">文件（可选替换）</MetaMedium>

              <UploadFileCard
                file={updateFile}
                expanded={updateFileExpanded}
                onToggleExpand={() => setUpdateFileExpanded((expanded) => !expanded)}
                onZipUpload={handleUpdateFileUpload}
                onRemove={handleUpdateFileRemove}
                variant="standard"
                accept={updateTarget?.kind === 'hook' ? '.yaml,.yml' : '.md'}
                uploadHint={updateTarget?.kind === 'hook' ? '点击或拖拽新的 YAML 文件替换' : '点击或拖拽 Markdown 文件替换'}
                uploadButtonLabel={updateTarget?.kind === 'hook' ? '上传 YAML' : '上传 Markdown'}
              />

              {!updateFile && <UploadRequirementsCard variant={updateTarget?.kind === 'hook' ? 'hook-manifest' : 'standard'} />}
            </div>

            <div>
              <MetaMedium as="label" tone="secondary" htmlFor="asset-update-slug">
                唯一标识 (slug)<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
              </MetaMedium>
              <Tooltip delayDuration={1000}>
                <TooltipTrigger asChild>
                  <Input
                    id="asset-update-slug"
                    value={updateTarget?.slug || ''}
                    disabled
                    className="mt-1"
                  />
                </TooltipTrigger>
                <TooltipContent side="right" sideOffset={8}>
                  <p>slug 不允许修改</p>
                </TooltipContent>
              </Tooltip>
            </div>

            <div>
              <MetaMedium as="label" tone="secondary" htmlFor="asset-update-name">
                显示名称<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
              </MetaMedium>
              <Input
                id="asset-update-name"
                value={updateFormData.name}
                onChange={(event) => setUpdateFormData((prev) => ({ ...prev, name: event.target.value }))}
                className="mt-1"
              />
            </div>

            <div>
              <MetaMedium as="label" tone="secondary" htmlFor="asset-update-description">描述</MetaMedium>
              <Textarea
                id="asset-update-description"
                value={updateFormData.description}
                onChange={(event) => setUpdateFormData((prev) => ({ ...prev, description: event.target.value }))}
                className="mt-1"
                rows={2}
              />
            </div>

            <div>
              <MetaMedium as="label" tone="secondary" htmlFor="asset-update-version">
                版本号<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
              </MetaMedium>
              <Input
                id="asset-update-version"
                value={updateFormData.version}
                onChange={(event) => handleUpdateVersionChange(event.target.value)}
                placeholder={`新版本号需高于上一版本号 ${updateTarget?.version || ''}`}
                className={`mt-1 ${updateVersionError ? 'border-red-400 focus:ring-red-400' : ''}`}
              />
              {updateVersionError && (
                <MetaText as="p" tone="danger" className="mt-1">{updateVersionError}</MetaText>
              )}
            </div>

            <div>
              <MetaMedium as="label" tone="secondary">应用范围</MetaMedium>
              <div className="mt-2">
                <ScopeSelect
                  scope={updateFormData.scope === 'public' ? 'all' : 'groups'}
                  selectedGroupIds={updateFormData.groupIds}
                  groups={MOCK_GROUPS}
                  onConfirm={(scope, groupIds) => {
                    if (scope === 'all') {
                      setUpdateFormData((prev) => ({ ...prev, scope: 'public', groupIds: [] }));
                    } else {
                      setUpdateFormData((prev) => ({ ...prev, scope: 'private', groupIds }));
                    }
                  }}
                />
              </div>
            </div>

            <div>
              <div className="mb-1 flex items-center justify-between">
                <MetaMedium as="label" tone="secondary" htmlFor="asset-update-changelog">更新说明</MetaMedium>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    const logs: string[] = [];
                    if (updateTarget && updateFormData.name !== updateTarget.name) logs.push(`名称调整为「${updateFormData.name}」`);
                    if (updateTarget && updateFormData.description !== getAssetDescription(updateTarget)) logs.push('更新资源描述');
                    if (updateFile) logs.push(`替换文件为 ${updateFile.name}`);
                    if (logs.length === 0) logs.push('更新企业规范资源版本');
                    setUpdateFormData((prev) => ({ ...prev, changeLog: logs.join('；') }));
                  }}
                  className="h-7 gap-1 px-2 text-xs"
                >
                  <Sparkles className="size-3" />
                  一键生成
                </Button>
              </div>
              <Textarea
                id="asset-update-changelog"
                value={updateFormData.changeLog}
                onChange={(event) => setUpdateFormData((prev) => ({ ...prev, changeLog: event.target.value }))}
                placeholder="请填写本次更新内容"
                className="mt-1"
                rows={3}
              />
            </div>

          </div>
          </DialogBody>

          <DialogFooter className="flex-shrink-0">
            <Button variant="claw-outline" onClick={() => {
              setUpdateTarget(null);
              setUpdateFile(null);
              setUpdateFileExpanded(false);
              setUpdateVersionError('');
            }}>
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              onClick={handleConfirmUpdateFile}
              disabled={!updateFormData.version || !!updateVersionError}
            >
              保存更新
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <BatchDistributeDialog
        open={instanceDialogOpen}
        onOpenChange={(open) => {
          setInstanceDialogOpen(open);
          if (!open && !uninstallDialogOpen) setActiveAssetId(null);
        }}
        skillName={activeAsset?.name || ''}
        skillVersion={activeAsset?.version}
        skillScope={activeAsset?.scope}
        skillGroupIds={activeAsset?.groupIds}
        onDistributionStart={handleDistributionStart}
        title="批量下发配置资产"
        descriptionNode={
          <>
            <BodyText as="p" tone="muted">
              将 <BodyMedium as="span" tone="primary" className="font-semibold">{activeAsset?.name || ''}</BodyMedium>
              {activeAsset?.kind === 'hook'
                ? ' 按用户应用范围下发，并合并到目标 Agent 的用户级配置。'
                : ' 下发至所选实例。'}
            </BodyText>
          </>
        }
        showScopeFilter
        showAgentType={activeAsset?.kind === 'hook'}
        instances={activeAssetInstances}
        groups={MOCK_GROUPS}
      />

      <BatchDeleteDialog
        open={uninstallDialogOpen}
        onOpenChange={(open) => {
          setUninstallDialogOpen(open);
          if (!open && !instanceDialogOpen) setActiveAssetId(null);
        }}
        skillName={activeAsset?.name || ''}
        skillVersion={activeAsset?.version || ''}
        distributedInstances={uninstallInstances}
        groups={MOCK_GROUPS}
        onDeleteStart={handleUninstallStart}
        resourceLabel="配置资产"
        warningText="卸载成功后，该资产在对应实例上恢复为未下发状态；本地备份文件由后端任务保留。"
        emptyText="暂无可卸载实例"
      />
    </div>
  );
}
