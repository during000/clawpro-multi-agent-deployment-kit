'use client';
import { useState, useEffect, useMemo, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { BackButton } from '@/components/ui/back-button';
import { SegmentedTabs } from '@/components/ui/segmented-tabs';
import { BodyMedium, BodyText, MetaText, MetaMedium, TenantDocTitle, HelperText } from '@/components/ui/Typography';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { ChevronDown, ChevronRight, Folder, FolderOpen, FileText, Search, Code, Eye, RefreshCw, Trash2, Download, Info, Loader, ShieldCheck, ShieldAlert, ShieldX, ExternalLink, ScanSearch, Send, X } from 'lucide-react';
import { toast } from 'sonner';
import { StatusTag } from '@/components/ui/status-tag';
import { Badge } from '@/components/ui/badge';
import { MOCK_SKILLS, DEFAULT_CATEGORIES, MOCK_GROUPS, MOCK_OPENCLAW_INSTANCES } from './mockData';
import BatchDistributeDialog from './BatchDistributeDialog';
import BatchDeleteDialog from './BatchDeleteDialog';
import SkillUpdateDialog from './SkillUpdateDialog';
import DeleteSkillDialog from './DeleteSkillDialog';
import { Input } from '@/components/ui/input';
import { SurfaceCard } from '@/components/ui/Surface';
import { FileBrowser, type VersionInfo } from '@/components/ui/file-browser';
import MDXRenderer from '@/components/MDXRenderer';
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
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';

import { type Skill, type DistributionStatus, DISTRIBUTION_STATUS_MAP } from './types';
import {
  getDistributionRecords,
  addDistributionRecord,
  updateDistributionRecord,
  createDistributionRecordId,
  type CachedDistributionRecord,
} from './distributionCache';
import { downloadSkillAsZip } from './downloadUtils';
// localStorage 缓存 key（与 SkillListTab 保持一致）
const SKILLS_CACHE_KEY = 'skillhub_enterprise_skills_cache';

interface SkillDetailProps {
  skillId: string;
  onBack: () => void;
  skills?: any[];
  defaultTab?: string;
  onSkillUpdate?: (updatedSkill: Skill) => void;
  onSkillDelete?: (skillId: string) => void;
  securityServiceActive?: boolean;
}

export default function SkillDetail({ skillId, onBack, skills, defaultTab, onSkillUpdate, onSkillDelete, securityServiceActive = false }: SkillDetailProps) {
  const [distributeDialogOpen, setDistributeDialogOpen] = useState(false);
  const [batchDeleteDialogOpen, setBatchDeleteDialogOpen] = useState(false);
  const [updateDialogOpen, setUpdateDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [isDownloading, setIsDownloading] = useState(false);
  const [expandedFile, setExpandedFile] = useState<string | null>('SKILL.md');
  const [activeDistributionId, setActiveDistributionId] = useState<string | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [statusFilter, setStatusFilter] = useState<'all' | DistributionStatus>('all');
  const [detailSearchQuery, setDetailSearchQuery] = useState('');
  const [activeTab, setActiveTab] = useState(defaultTab || 'overview');
  const [selectedVersion, setSelectedVersion] = useState<string>('');
  /** 记录类型筛选：全部 / 下发记录 / 卸载记录 */
  const [recordTypeFilter, setRecordTypeFilter] = useState<'all' | 'distribute' | 'delete'>('all');
  const [fileViewMode, setFileViewMode] = useState<'preview' | 'source'>('preview');

  const [securityScanDialogOpen, setSecurityScanDialogOpen] = useState(false);
  const skillsArray = skills || MOCK_SKILLS;

  // 从缓存读取下发记录
  const [distributionRecords, setDistributionRecords] = useState<CachedDistributionRecord[]>([]);

  const refreshRecords = useCallback(() => {
    setDistributionRecords(getDistributionRecords(skillId));
  }, [skillId]);

  // 首次加载 + 监听缓存更新
  useEffect(() => {
    refreshRecords();
    const handler = () => refreshRecords();
    window.addEventListener('distribution-cache-updated', handler);
    return () => window.removeEventListener('distribution-cache-updated', handler);
  }, [refreshRecords]);

  // 是否有进行中的下发或卸载任务
  const hasInProgress = distributionRecords.some(r => r.status === 'distributing' || r.status === 'deleting');
  
  // 先从 props 传入的 skills 中查找，找不到再从 localStorage 缓存中查找
  const skill = useMemo(() => {
    let found = skillsArray.find((s: any) => s.id === skillId);
    if (!found) {
      try {
        const cached = localStorage.getItem(SKILLS_CACHE_KEY);
        if (cached) {
          const parsed = JSON.parse(cached);
          const cachedSkill = parsed.find((s: any) => s.id === skillId);
          if (cachedSkill) {
            found = {
              ...cachedSkill,
              uploadTime: new Date(cachedSkill.uploadTime),
              lastDistributionTime: cachedSkill.lastDistributionTime ? new Date(cachedSkill.lastDistributionTime) : undefined,
            };
          }
        }
      } catch (e) {
        console.warn('从缓存加载 skill 失败:', e);
      }
    }
    return found;
  }, [skillId, skillsArray]);

  // 本地安全检测状态覆盖（点击检测后立即生效，不依赖父组件 re-render）
  const [localSecurityOverride, setLocalSecurityOverride] = useState<Skill['securityInfo'] | null>(null);
  // 当 skillId 变化时重置本地覆盖
  useEffect(() => {
    setLocalSecurityOverride(null);
  }, [skillId]);

  /*
   * 停服时仍允许「下发和卸载记录 Tab 的记录类型筛选」浮层内容可交互：
   * Radix Select 的浮层渲染在 Portal 中（[data-radix-popper-content-wrapper]），
   * 不在页面 DOM 树内，无法被页面层的 data-billing-exempt 容器覆盖。
   * 这里通过 MutationObserver 监听 document.body 增量挂载的 select-content，
   * 并附加豁免标记，使 AdminDisabledOverlay 对浮层内的 SelectItem 失效。
   * 页面其他 Select 同样受益，符合"只读类筛选"语义。
   */
  useEffect(() => {
    const ensureSelectExempt = (root: ParentNode) => {
      root.querySelectorAll<HTMLElement>(
        '[data-slot="select-content"]'
      ).forEach((el) => {
        if (!el.hasAttribute("data-billing-exempt")) {
          el.setAttribute("data-billing-exempt", "");
        }
      });
    };

    // 兜底：处理已挂载的（页面加载时）
    ensureSelectExempt(document.body);

    const observer = new MutationObserver((mutations) => {
      for (const mutation of mutations) {
        mutation.addedNodes.forEach((node) => {
          if (node instanceof HTMLElement) {
            ensureSelectExempt(node);
          }
        });
      }
    });
    observer.observe(document.body, { childList: true, subtree: true });
    return () => observer.disconnect();
  }, []);

  // 合并后的 skill（本地覆盖优先）
  const effectiveSkill = useMemo(() => {
    if (!skill) return skill;
    if (localSecurityOverride) {
      return { ...skill, securityInfo: localSecurityOverride };
    }
    return skill;
  }, [skill, localSecurityOverride]);

  

  // 根据选中版本获取文件列表（如选中的是非最新版本，则从 versionHistory 中取）
  const currentVersionFiles = useMemo(() => {
    if (!skill) return [];
    // 如果选中的版本是最新版本（versions[0]）或者没有选中版本，使用 skill.files
    if (!selectedVersion || selectedVersion === skill.versions?.[0]) {
      return skill.files || [];
    }
    // 从 versionHistory 中查找对应版本的文件列表
    const versionRecord = skill.versionHistory?.find((v: any) => v.version === selectedVersion);
    if (versionRecord?.files && versionRecord.files.length > 0) {
      return versionRecord.files;
    }
    // 如果历史版本没有文件记录，回退到当前文件
    return skill.files || [];
  }, [skill, selectedVersion]);

  // 剥离唯一顶层文件夹：如果所有文件都在同一个顶层目录下，则去掉该前缀
  const { processedFiles, strippedPrefix } = useMemo(() => {
    const rawFiles = currentVersionFiles;
    if (rawFiles.length === 0) return { processedFiles: rawFiles, strippedPrefix: '' };
    
    const topDirs = new Set<string>();
    let topFileCount = 0;
    for (const f of rawFiles) {
      const parts = f.name.split('/');
      if (parts.length > 1) {
        topDirs.add(parts[0]);
      } else {
        topFileCount++;
      }
    }
    // 所有文件都在同一个顶层目录下，且没有顶层文件
    if (topDirs.size === 1 && topFileCount === 0) {
      const prefix = Array.from(topDirs)[0] + '/';
      return {
        processedFiles: rawFiles.map((f: any) => ({ ...f, name: f.name.slice(prefix.length) })),
        strippedPrefix: prefix,
      };
    }
    return { processedFiles: rawFiles, strippedPrefix: '' };
  }, [currentVersionFiles]);

  // 可展示的文件扩展名（文本类文件）
  const VIEWABLE_EXTENSIONS = ['.md', '.xml', '.json', '.txt', '.yaml', '.yml', '.toml', '.ini', '.cfg', '.conf', '.sh', '.bat', '.py', '.js', '.ts', '.css', '.html', '.htm', '.svg', '.env', '.gitignore', '.dockerfile'];
  
  


  
  // 递归在文件树中查找文件（支持 children 嵌套结构和 path 匹配）
  const findFileInTree = (files: any[], targetName: string): any => {
    for (const f of files) {
      // 同时匹配 name 和 path
      if (f.name === targetName || f.path === targetName) return f;
      if (f.children && f.children.length > 0) {
        const found = findFileInTree(f.children, targetName);
        if (found) return found;
      }
    }
    return null;
  };

  const getFileContent = (fileName: string): string => {
    // 使用当前选中版本的文件列表（而不是始终用最新版本的 skill.files）
    const versionFiles = currentVersionFiles;

    // 对 SKILL.md，也优先从当前版本的文件列表中取
    if (fileName === 'SKILL.md' || fileName.toLowerCase() === 'skill.md') {
                const skillMdFile = versionFiles.find((f: any) => f.name.toLowerCase() === 'skill.md' || f.name.toLowerCase().endsWith('/skill.md'));
      if (skillMdFile?.content) return skillMdFile.content;
      // 如果当前版本是最新版本，回退到 skill.content
      if (!selectedVersion || selectedVersion === skill?.versions?.[0]) {
        return skill?.content || '';
      }
      return '';
    }
    // 如果剥离了顶层文件夹，查找时还原为原始路径
    const originalName = strippedPrefix ? strippedPrefix + fileName : fileName;
    const file = findFileInTree(versionFiles, originalName);
    if (file?.content) return file.content;
    // 也尝试直接用处理后的路径查找
    const file2 = findFileInTree(versionFiles, fileName);
    if (file2?.content) return file2.content;
    return '';
  };
  
  const handleDistributionStart = (selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    // 创建新的分发记录并写入缓存
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId,
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
    
    // 模拟下发进度
    simulateDistribution(recordId, selectedInstanceIds.length);
  };
  
  const simulateDistribution = (recordId: string, totalCount: number) => {
    let completed = 0;
    const interval = setInterval(() => {
      completed += Math.floor(Math.random() * 3) + 1;
      if (completed >= totalCount) {
        completed = totalCount;
        clearInterval(interval);
        
        // 模拟随机失败一些实例
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
        // 更新进度
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount: completed,
          inProgressCount: totalCount - completed,
        }));
      }
    }, 800);
  };
  
  const handleRetry = (recordId: string) => {
    const record = distributionRecords.find(r => r.id === recordId);
    if (!record) return;
    
    const failedInstances = record.instances.filter(inst => inst.distributionStatus === 'failed');
    
    // 重置失败的实例状态
    updateDistributionRecord(recordId, (r) => ({
      ...r,
      status: 'distributing' as DistributionStatus,
      inProgressCount: failedInstances.length,
      instances: r.instances.map(inst => ({
        ...inst,
        distributionStatus: (inst.distributionStatus === 'failed' ? 'distributing' : inst.distributionStatus) as DistributionStatus,
      })),
    }));
    
    simulateDistribution(recordId, failedInstances.length);
  };

  // ========== 批量卸载实例 ==========

  /** 从下发记录中聚合已下发成功的实例列表（用于卸载弹窗） */
  const distributedInstancesForDelete = useMemo(() => {
    // 从所有下发记录中找出成功下发的实例，去重
    const instanceMap = new Map<string, any>();
    distributionRecords
      .filter(r => (r.type || 'distribute') === 'distribute') // 只看下发记录
      .forEach(r => {
        r.instances.forEach(inst => {
          if (inst.distributionStatus === 'success' && !instanceMap.has(inst.id)) {
            // 尝试从 MOCK_OPENCLAW_INSTANCES 获取更多信息
            const fullInst = MOCK_OPENCLAW_INSTANCES.find(i => i.id === inst.id);
            const groupName = fullInst?.groupIds?.[0]
              ? MOCK_GROUPS.find(g => g.id === fullInst.groupIds[0])?.name
              : undefined;
            instanceMap.set(inst.id, {
              id: inst.id,
              name: inst.name,
              createdBy: inst.createdBy || 'admin',
              groupName: groupName || '全部用户',
              distributedVersion: skill?.version,
              distributedTime: r.timestamp,
              deleteStatus: 'not_deleted' as const,
            });
          }
        });
      });
    return Array.from(instanceMap.values());
  }, [distributionRecords, skill?.version]);

  const handleDeleteStart = (selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId,
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'deleting',
      type: 'delete',
      operator: 'yequanzheng',
      instances: selectedInstancesData.map(inst => ({
        id: inst.id,
        name: inst.name,
        createdBy: inst.createdBy || 'admin',
        distributionStatus: 'distributing' as DistributionStatus, // 复用状态表示进行中
      })),
    };

    addDistributionRecord(newRecord);
    setActiveDistributionId(recordId);
    setBatchDeleteDialogOpen(false);

    // 模拟卸载进度
    simulateDeletion(recordId, selectedInstanceIds.length);
  };

  const simulateDeletion = (recordId: string, totalCount: number) => {
    let completed = 0;
    const failReasons = ['实例离线', '权限不足', '技能被占用', '网络超时', '实例已停止'];
    const interval = setInterval(() => {
      completed += Math.floor(Math.random() * 3) + 1;
      if (completed >= totalCount) {
        completed = totalCount;
        clearInterval(interval);

        // 90% 成功，10% 失败
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
      } else {
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount: completed,
          inProgressCount: totalCount - completed,
        }));
      }
    }, 800);
  };

  /** 根据记录类型筛选后的记录列表 */
  const filteredRecords = useMemo(() => {
    if (recordTypeFilter === 'all') return distributionRecords;
    return distributionRecords.filter(r => (r.type || 'distribute') === recordTypeFilter);
  }, [distributionRecords, recordTypeFilter]);

  // 下载 Skill
  const handleDownload = async () => {
    if (!skill) return;
    setIsDownloading(true);
    try {
      await downloadSkillAsZip(skill);
      toast.success(`「${skill.name}」下载完成`);
    } catch {
      toast.error('下载失败，请重试');
    } finally {
      setIsDownloading(false);
    }
  };

  // 提交安全检测（更新本地状态立即生效 + 通知父组件 + 10s后mock完成）
  const handleSecurityScan = () => {
    if (!skill) return;
    setSecurityScanDialogOpen(false);
    toast.success('已提交安全检测，预计 5 分钟后完成');

    const newSecurityInfo = {
      overallStatus: 'scanning' as const,
      engines: [],
    };
    // 本地立即生效
    setLocalSecurityOverride(newSecurityInfo);

    // 同步给父组件（如果传了 onSkillUpdate）
    const updatedSkill: Skill = {
      ...skill,
      securityInfo: newSecurityInfo,
    };
    if (onSkillUpdate) onSkillUpdate(updatedSkill);

    // 10s 后 mock 完成检测，随机生成结果
    setTimeout(() => {
      const rand = Math.random();
      let result: 'safe' | 'suspicious' | 'malicious';
      if (rand < 0.5) result = 'safe';
      else if (rand < 0.8) result = 'suspicious';
      else result = 'malicious';

      const safeDims = [
        { name: '供应链风险', status: 'safe' as const, detail: '未发现可疑的第三方依赖引入或供应链污染行为' },
        { name: '命令执行风险', status: 'safe' as const, detail: '未检测到危险的系统命令调用或子进程执行操作' },
        { name: '网络请求与数据外传', status: 'safe' as const, detail: '未发现未经授权的网络请求或敏感数据外传行为' },
        { name: '文件操作与敏感路径访问', status: 'safe' as const, detail: '未检测到对敏感系统路径或凭证文件的异常访问' },
        { name: 'Prompt 注入风险', status: 'safe' as const, detail: '未发现试图篡改 AI Agent 行为的 Prompt 注入指令' },
        { name: '远程脚本下载执行', status: 'safe' as const, detail: '未检测到从远程服务器下载并执行脚本的行为' },
        { name: '可疑编码/混淆', status: 'safe' as const, detail: '未发现可疑的代码编码混淆或加密逃逸技术' },
        { name: '其他安全风险', status: 'safe' as const, detail: '未检测到其他类别的异常安全风险行为' },
      ];
      const suspiciousDims = [
        { name: '供应链风险', status: 'safe' as const, detail: '未发现可疑的第三方依赖引入或供应链污染行为' },
        { name: '命令执行风险', status: 'suspicious' as const, detail: '检测到潜在的系统命令调用，存在一定风险' },
        { name: '网络请求与数据外传', status: 'safe' as const, detail: '未发现未经授权的网络请求或敏感数据外传行为' },
        { name: '文件操作与敏感路径访问', status: 'safe' as const, detail: '未检测到对敏感系统路径或凭证文件的异常访问' },
        { name: 'Prompt 注入风险', status: 'safe' as const, detail: '未发现试图篡改 AI Agent 行为的 Prompt 注入指令' },
        { name: '远程脚本下载执行', status: 'safe' as const, detail: '未检测到从远程服务器下载并执行脚本的行为' },
        { name: '可疑编码/混淆', status: 'suspicious' as const, detail: '发现部分代码使用了 Base64 编码包裹，需人工确认' },
        { name: '其他安全风险', status: 'safe' as const, detail: '未检测到其他类别的异常安全风险行为' },
      ];
      const maliciousDims = [
        { name: '供应链风险', status: 'malicious' as const, detail: '发现恶意第三方依赖注入，存在供应链污染' },
        { name: '命令执行风险', status: 'malicious' as const, detail: '检测到危险的系统命令调用，执行反弹 shell' },
        { name: '网络请求与数据外传', status: 'malicious' as const, detail: '发现向外部 C2 服务器发送敏感数据' },
        { name: '文件操作与敏感路径访问', status: 'safe' as const, detail: '未检测到对敏感系统路径或凭证文件的异常访问' },
        { name: 'Prompt 注入风险', status: 'suspicious' as const, detail: '发现可能篡改 AI Agent 行为的指令片段' },
        { name: '远程脚本下载执行', status: 'malicious' as const, detail: '检测到从远程服务器下载并执行恶意脚本' },
        { name: '可疑编码/混淆', status: 'malicious' as const, detail: '发现大量代码使用多层编码混淆，隐藏恶意逻辑' },
        { name: '其他安全风险', status: 'safe' as const, detail: '未检测到其他类别的异常安全风险行为' },
      ];

      const dims = result === 'safe' ? safeDims : result === 'suspicious' ? suspiciousDims : maliciousDims;
      const score2 = result === 'safe' ? 85 : result === 'suspicious' ? 55 : 15;

      const completedSecurityInfo = {
        overallStatus: result,
        contentHash: Math.random().toString(36).slice(2, 18),
        engines: [
          { engineName: '科恩实验室', status: 'safe' as const, reportUrl: '#', score: 92, dimensions: safeDims },
          { engineName: '云鼎实验室', status: result, reportUrl: '#', score: score2, dimensions: dims },
        ],
      };

      // 本地更新
      setLocalSecurityOverride(completedSecurityInfo);

      // 同步给父组件
      if (onSkillUpdate) {
        onSkillUpdate({ ...skill, securityInfo: completedSecurityInfo });
      }

      const resultLabel = result === 'safe' ? '安全' : result === 'suspicious' ? '可疑' : '恶意';
      toast.info(`「${skill.name}」安全检测完成：${resultLabel}`);
    }, 10000);
  };

  // 更新 Skill 回调
  const handleSkillUpdate = (updatedSkill: Skill) => {
    if (onSkillUpdate) {
      onSkillUpdate(updatedSkill);
    }
    // 重置版本选择，让 useEffect 自动选中最新版本
    setSelectedVersion('');
    setUpdateDialogOpen(false);
  };

  // 删除 Skill 回调
  const handleSkillDelete = () => {
    if (!skill) return;
    if (onSkillDelete) {
      onSkillDelete(skill.id);
    }
    toast.success(`Skill「${skill.name}」已删除`);
    setDeleteDialogOpen(false);
    onBack();
  };

  if (!skill) {
    return (
      <div className="text-center py-12">
        <BodyText as="p" tone="muted">技能未找到</BodyText>
        <Button onClick={onBack} className="mt-4">返回列表</Button>
      </div>
    );
  }

  const getCategoryName = (catId: string) => {
    return DEFAULT_CATEGORIES.find((cat: any) => cat.id === catId)?.name || catId;
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
    <div className="space-y-4">
      {/* ======== Header（参照 MCPDetail / PluginDetail 卡片风格）======== */}
      <header className="flex flex-col gap-4">
        {/* 返回按钮 — 卡片外，单独成行 */}
        {/*
          停服时仍允许「返回上级」可点击：纯导航操作，不消耗管控台写权限。
          通过 data-billing-exempt 豁免 AdminDisabledOverlay 的全局禁用样式与点击拦截。
        */}
        <BackButton onClick={onBack} className="self-start" data-billing-exempt>返回上级</BackButton>

        {/* 基础信息卡片 — bg + 圆角 + 边框 + p-6 */}
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-start justify-between gap-4">
            <div className="flex flex-col gap-2 min-w-0 flex-1">
            <div className="flex items-center gap-3">
              <TenantDocTitle as="h1">
                {skill.name}
              </TenantDocTitle>
              {/* 安全状态徽标 */}
              {(() => {
                const secStatus = effectiveSkill?.securityInfo?.overallStatus || 'not_scanned';
                if (secStatus === 'not_scanned') {
                  return (
                    <span className="inline-flex items-center gap-1.5">
                      <StatusTag mode="fill" variant="gray">未检测</StatusTag>
                      {securityServiceActive ? (
                        <button
                          onClick={() => setSecurityScanDialogOpen(true)}
                          className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium text-[var(--text-brand)] bg-blue-50 hover:bg-blue-100 rounded-full transition-colors"
                        >
                          <ScanSearch className="w-3 h-3" />
                          检测
                        </button>
                      ) : (
                        <Tooltip delayDuration={300}>
                          <TooltipTrigger asChild>
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium text-[var(--text-weak)] bg-gray-100 rounded-full cursor-not-allowed">
                              <ScanSearch className="w-3 h-3" />
                              检测
                            </span>
                          </TooltipTrigger>
                          <TooltipContent side="bottom" className="max-w-[280px]">
                            <MetaText>安全检测服务尚未开通，请前往技能库列表页右上角免费开通试用（26年6月30日前1000次免费试用）。</MetaText>
                          </TooltipContent>
                        </Tooltip>
                      )}
                    </span>
                  );
                }
                if (secStatus === 'scanning') {
                  return (
                    <StatusTag mode="fill" variant="blue">
                      <Loader className="w-3 h-3 animate-spin" />
                      检测中
                    </StatusTag>
                  );
                }
                const IconComp = secStatus === 'safe' ? ShieldCheck : secStatus === 'suspicious' ? ShieldAlert : ShieldX;
                const reportUrl = effectiveSkill?.securityInfo?.engines?.[0]?.reportUrl;
                const secVariant = secStatus === 'safe' ? 'green' : secStatus === 'suspicious' ? 'orange' : 'red';
                return (
                  <span className="inline-flex items-center gap-1.5">
                    <StatusTag mode="fill" variant={secVariant} icon={<IconComp />}>
                      {secStatus === 'safe' ? '安全' : secStatus === 'suspicious' ? '可疑' : '恶意'}
                    </StatusTag>
                    {reportUrl && (
                      <a
                        href={reportUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-0.5 text-xs text-[#355EF1] hover:text-[#355EF1] transition-colors"
                      >
                        报告
                        <ExternalLink className="w-3 h-3" />
                      </a>
                    )}
                  </span>
                );
              })()}
            </div>
            {/* slug */}
            <MetaText as="p" tone="weak" className="font-mono">slug：{skill.slug}</MetaText>
            {/* 元信息行 */}
            <div className="flex items-center flex-wrap gap-2">
              <StatusTag mode="fill" variant="gray" className="font-mono">v{skill.version}</StatusTag>
              <BodyText as="span" tone="weak">｜</BodyText>
              <BodyText as="span" tone="secondary">分类：{skill.categories.map((catId: string) => getCategoryName(catId)).join('、')}</BodyText>
              <BodyText as="span" tone="weak">｜</BodyText>
              <BodyText as="span" tone="secondary">范围：{skill.scope === 'public' || !skill.groupIds || skill.groupIds.length === 0
                ? '全部用户'
                : skill.groupIds.map((gId: string) => MOCK_GROUPS.find(g => g.id === gId)?.name || gId).join('、')
              }</BodyText>
            </div>
            {skill.description && (
              <BodyText as="p" tone="secondary" className="leading-5 mt-1">
                {skill.description}
              </BodyText>
            )}
            </div>

            {/* 操作按钮组 — 卡片右上角（与标题水平对齐） */}
            <div className="flex flex-wrap items-center justify-end gap-2 shrink-0">
              <TooltipProvider>
                <Tooltip delayDuration={1000}>
                  <TooltipTrigger asChild>
                    <span>
                      <Button
                        variant="claw-outline"
                        size="claw"
                        onClick={() => setUpdateDialogOpen(true)}
                        disabled={hasInProgress}
                      >
                        <RefreshCw className="w-4 h-4" />
                        更新
                      </Button>
                    </span>
                  </TooltipTrigger>
                  {hasInProgress && (
                    <TooltipContent>仅支持状态为正常的 Skill</TooltipContent>
                  )}
                </Tooltip>
              </TooltipProvider>

              <Button
                variant="claw-outline"
                size="claw"
                onClick={handleDownload}
                disabled={isDownloading}
                data-billing-exempt
              >
                {isDownloading ? <Loader className="w-4 h-4 animate-spin" /> : <Download className="w-4 h-4" />}
                下载
              </Button>

              <TooltipProvider>
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
                    <TooltipContent>仅支持状态为正常的 Skill</TooltipContent>
                  )}
                </Tooltip>
              </TooltipProvider>

              <TooltipProvider>
                <Tooltip delayDuration={1000}>
                  <TooltipTrigger asChild>
                    <span>
                      <Button
                        variant="claw-outline"
                        size="claw"
                        onClick={() => setBatchDeleteDialogOpen(true)}
                        disabled={hasInProgress || distributedInstancesForDelete.length === 0}
                      >
                        <Trash2 className="w-4 h-4" />
                        {distributionRecords.some(r => r.status === 'deleting') ? '卸载中...' : '批量卸载'}
                      </Button>
                    </span>
                  </TooltipTrigger>
                  {(hasInProgress || distributedInstancesForDelete.length === 0) && (
                    <TooltipContent>
                      {hasInProgress ? '有任务进行中，请等待完成' : '暂无已下发的实例'}
                    </TooltipContent>
                  )}
                </Tooltip>
              </TooltipProvider>

              <Button
                variant="claw-primary"
                size="claw"
                onClick={() => setDistributeDialogOpen(true)}
                disabled={hasInProgress}
              >
                {hasInProgress ? '下发中...' : '批量下发'}
                <Send className="w-4 h-4" />
              </Button>
            </div>
          </div>
        </div>
      </header>

      {/* ======== 横向 Segmented Tab（§8.6 规范，参照 Agent 详情页）======== */}
      <div className="flex items-center justify-start">
        {/*
          停服时仍允许「概述/文件列表/下发和卸载记录」三个 Tab 可点击：纯导航/查看类操作。
          SegmentedTabs 是通用 UI 组件、未提供 per-tab 豁免能力，
          整组 Tab 统一豁免停服禁用。
        */}
        <div data-billing-exempt>
          <SegmentedTabs
            tabs={[
              { id: "overview", label: "概述" },
              { id: "files", label: "文件列表" },
              { id: "distribution", label: "下发和卸载记录" },
            ]}
            active={activeTab}
            onChange={setActiveTab}
            ariaLabel="技能详情 Tab 切换"
          />
        </div>
        {/* 下发和卸载记录 Tab 时，右侧显示记录类型筛选 */}
        {activeTab === 'distribution' && (
          <div data-billing-exempt>
          <Select
            value={recordTypeFilter}
            onValueChange={(value) => setRecordTypeFilter(value as 'all' | 'distribute' | 'delete')}
          >
            <SelectTrigger className="w-32 h-8 ml-4">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部</SelectItem>
              <SelectItem value="distribute">下发记录</SelectItem>
              <SelectItem value="delete">卸载记录</SelectItem>
            </SelectContent>
          </Select>
          </div>
        )}
      </div>

      {/* ======== Tab 内容 ======== */}
      <div className="pb-6">
        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          {/* 隐藏原始 TabsList，使用上方自定义 Segmented */}
          <TabsList className="hidden">
            <TabsTrigger value="overview">概述</TabsTrigger>
            <TabsTrigger value="files">文件列表</TabsTrigger>
            <TabsTrigger value="distribution">下发和卸载记录</TabsTrigger>
          </TabsList>

          {/* 概述 Tab */}
          <TabsContent value="overview" className="mt-0 p-0">
            <SurfaceCard className="p-6">
              <MDXRenderer content={(() => {
                // 如果选中的是最新版本或未选中，用 skill.content
                if (!selectedVersion || selectedVersion === skill.versions?.[0]) {
                  return skill.content || '';
                }
                // 否则从当前版本文件列表中取 SKILL.md 内容
                const versionFiles = currentVersionFiles;
                const skillMdFile = versionFiles.find((f: any) => f.name.toLowerCase() === 'skill.md' || f.name.toLowerCase().endsWith('/skill.md'));
                return skillMdFile?.content || skill.content || '';
              })()} />
            </SurfaceCard>
          </TabsContent>

          {/* 文件列表 Tab */}
          <TabsContent value="files" className="mt-0 p-0">
          {/*
            停服时仍允许「文件列表」Tab 内的版本切换/文件选择/Preview/Source 等操作正常可用：
            纯导航/查看类操作，不消耗管控台写权限。
            FileBrowser 是通用 UI 组件、不提供 per-area 豁免能力，整块 Tab 容器统一豁免。
          */}
          <div data-billing-exempt>
          <FileBrowser
            versions={(() => {
              return (skill.versions || []).map((v: string, i: number) => ({
                version: v,
                date: (() => {
                  const d = new Date(skill.uploadTime);
                  d.setDate(d.getDate() - i * 14);
                  return d.toISOString().slice(0, 10);
                })(),
                isLatest: i === 0,
              }));
            })()}
            files={processedFiles}
            getFileContent={(fileName) => {
              const versionFiles = currentVersionFiles;
              if (fileName === 'SKILL.md' || fileName.toLowerCase() === 'skill.md') {
                const f = versionFiles.find((f: any) => f.name.toLowerCase() === 'skill.md' || f.name.toLowerCase().endsWith('/skill.md'));
                if (f?.content) return f.content;
                if (!selectedVersion || selectedVersion === skill?.versions?.[0]) return skill?.content || '';
                return '';
              }
              const orig = strippedPrefix ? strippedPrefix + fileName : fileName;
              const find = (files: any[], t: string): any => {
                for (const f of files) {
                  if (f.name === t || f.path === t) return f;
                  if (f.children) { const r = find(f.children as any[], t); if (r) return r; }
                }
                return null;
              };
              const r1 = find(versionFiles, orig);
              if (r1?.content) return r1.content;
              const r2 = find(versionFiles, fileName);
              if (r2?.content) return r2.content;
              return '';
            }}
            showDownload={true}
            onDownload={handleDownload}
            isDownloading={isDownloading}
          />
          </div>
          </TabsContent>

          {/* 下发和卸载记录 Tab */}
          <TabsContent value="distribution" className="mt-0 p-0">
            <div className="space-y-3">
              {filteredRecords.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-12 text-center">
                  <HelperText as="p">还没有下发和卸载记录</HelperText>
                </div>
              ) : (
                <div className="space-y-3">
                  {filteredRecords.map((record, idx) => {
                    const progress = record.totalCount > 0 ? Math.round((record.successCount / record.totalCount) * 100) : 0;
                    const isDeleteRecord = record.type === 'delete';
                    const isInProgress = record.status === 'distributing' || record.status === 'deleting';
                    
                    return (
                      <div key={record.id} className="border border-gray-200 rounded-lg p-4">
                        <div className="flex items-start justify-between mb-3">
                          <div>
                            <BodyMedium as="p" tone="primary" className="font-semibold">
                              #{idx + 1} · {isDeleteRecord ? '卸载' : '下发'} · v{skill.version} {new Date(record.timestamp).toLocaleString('zh-CN')}
                            </BodyMedium>
                          </div>
                          <div className="flex items-center gap-2">
                            <span className={`inline-block px-3 py-1 rounded text-xs font-medium ${
                              isInProgress ? (isDeleteRecord ? 'bg-red-100 text-red-700' : 'bg-blue-50 text-blue-700') :
                              record.successCount === record.totalCount ? 'bg-green-50 text-green-700' :
                              record.successCount === 0 && record.failedCount > 0 ? 'bg-red-50 text-red-700' :
                              'bg-yellow-50 text-yellow-700'
                            }`}>
                              {isInProgress
                                ? `${isDeleteRecord ? '卸载' : '下发'}中 ${progress}%`
                                : `${isDeleteRecord ? '卸载' : '下发'}完成，${record.successCount}个成功，${record.failedCount}个失败`}
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
                                className={`h-2 rounded-full transition-all duration-300 ${
                                  isDeleteRecord ? 'bg-red-600' : 'bg-blue-600'
                                }`}
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
        skillName={skill.name}
        skillVersion={skill.version}
        skillScope={skill.scope}
        skillGroupIds={skill.groupIds}
        onDistributionStart={handleDistributionStart}
        instances={MOCK_OPENCLAW_INSTANCES}
        groups={MOCK_GROUPS}
      />

      {/* 批量卸载实例对话框 */}
      <BatchDeleteDialog
        open={batchDeleteDialogOpen}
        onOpenChange={setBatchDeleteDialogOpen}
        skillName={skill.name}
        skillVersion={skill.version}
        distributedInstances={distributedInstancesForDelete}
        groups={MOCK_GROUPS}
        onDeleteStart={handleDeleteStart}
      />

      {/* 更新对话框 */}
      {skill && (
        <SkillUpdateDialog
          open={updateDialogOpen}
          onOpenChange={setUpdateDialogOpen}
          skill={skill}
          onConfirm={(updatedSkill) => handleSkillUpdate(updatedSkill)}
          defaultSecurityScan={localStorage.getItem('skill_default_security_scan') !== 'false'}
          onDefaultSecurityScanChange={(value) => {
            localStorage.setItem('skill_default_security_scan', String(value));
          }}
          securityServiceActive={securityServiceActive}
        />
      )}

      {/* 删除确认对话框 */}
      <DeleteSkillDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        skillName={skill.name}
        onConfirm={handleSkillDelete}
      />

      {/* 分发详情对话框 */}
      <Dialog open={detailsOpen} onOpenChange={setDetailsOpen}>
        <DialogContent className="sm:max-w-[720px] max-h-[80vh] flex flex-col">
          <DialogHeader>
            <DialogTitle>{activeDistribution && (activeDistribution.type || 'distribute') === 'delete' ? '卸载详情' : '下发详情'}</DialogTitle>
          </DialogHeader>
          
          {activeDistribution && (
            <div className="space-y-4">
              {/* 筛选器 + 搜索框 */}
              <div className="flex items-center gap-2">
                <div className="relative flex-1">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#A3A3A3]" />
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
                    <SelectItem value="distributing">{activeDistribution && (activeDistribution.type || 'distribute') === 'delete' ? '卸载中' : '下发中'}</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {/* 实例列表 */}
              <div className="border border-gray-200 rounded-[4px] overflow-y-auto max-h-64">
                <Table>
                  <TableHeader className="bg-gray-50 border-b border-gray-200 sticky top-0">
                    <TableRow>
                      <TableHead className="text-left">实例名称</TableHead>
                      <TableHead className="text-left min-w-[140px]">实例ID</TableHead>
                      <TableHead className="text-left">状态</TableHead>
                      <TableHead className="text-left">失败原因</TableHead>
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
                          <TableCell><BodyText as="span" tone="primary">{instance.name}</BodyText></TableCell>
                          <TableCell><MetaText className="font-mono whitespace-nowrap">{instance.id}</MetaText></TableCell>
                          <TableCell>
                            <span className={`inline-block px-2 py-1 rounded text-xs font-medium ${
                              DISTRIBUTION_STATUS_MAP[instance.distributionStatus]?.color || 'bg-gray-50 text-[#737373]'
                            }`}>
                              {DISTRIBUTION_STATUS_MAP[instance.distributionStatus]?.label || '未下发'}
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
          )}
        </DialogContent>
      </Dialog>

      {/* 安全检测确认弹窗 (row 47) - 警示弹窗 */}
      <AlertDialog open={securityScanDialogOpen} onOpenChange={setSecurityScanDialogOpen}>
        <AlertDialogContent className="sm:max-w-md">
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              提交安全检测
              <StatusTag mode="fill" variant="blue">限免</StatusTag>
            </AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <BodyText as="p" tone="primary">
              确认对技能「<BodyMedium as="span" tone="primary">{skill.name}</BodyMedium>」提交安全检测？检测将由腾讯云 AI Agent 安全进行，通常几分钟内完成。
            </BodyText>
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              variant="dialog-confirm"
              onClick={handleSecurityScan}
            >
              确认检测
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
