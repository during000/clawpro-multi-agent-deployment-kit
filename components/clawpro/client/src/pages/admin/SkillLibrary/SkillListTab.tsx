import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { StatusTag } from '@/components/ui/status-tag';
import { SurfaceCard } from '@/components/ui/Surface';
import { SegmentGroup, SegmentOption } from '@/components/ui/segment';

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableActionCell } from '@/components/ui/table';

import { Search, Grid3x3, List, Send, Download, Trash2, RefreshCw, Loader, Check, Edit2, ShieldCheck, ShieldAlert, ShieldX, ScanSearch, ExternalLink, Info, Settings2, X, PackageX, ArrowDownToLine, ArrowUpFromLine } from 'lucide-react';
import { Dialog, DialogContent, DialogBody, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog';
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card';
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip';
import { BodyText, BodyMedium, MetaText, MetaMedium, CardTitle as CardHeading } from '@/components/ui/Typography';

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
import { MoreActionsDropdown } from '@/components/ui/more-actions-dropdown';
import { useLocation } from 'wouter';
import { Badge } from '@/components/ui/badge';
import { DEFAULT_CATEGORIES, MOCK_OPENCLAW_INSTANCES, MOCK_GROUPS, MOCK_PROJECT_GROUPS } from './mockData';
import { skillStore } from './skillStore';
import { projectAssetStore } from '../project-assets/projectAssetStore';
import { ScopeSelect } from '@/components/ScopeSelect';
import SkillUploadDialog from './SkillUploadDialog';
import SkillDetail from './SkillDetail';
import BatchDistributeDialog from './BatchDistributeDialog';
import EditCategoriesDialog from './EditCategoriesDialog';

import SkillUpdateDialog from './SkillUpdateDialog';
import DeleteSkillDialog from './DeleteSkillDialog';
import OfflineSkillDialog from './OfflineSkillDialog';
import CategoryManagementDialog from './CategoryManagementDialog';
import BatchDeleteDialog from './BatchDeleteDialog';
import SkillReviewDialog from './SkillReviewDialog';
import { Skill, type SkillScope, SECURITY_STATUS_MAP, type SecurityStatus } from './types';
import {
  getSkillDistributionSummary,
  getAllDistributionRecords,
  addDistributionRecord,
  updateDistributionRecord,
  createDistributionRecordId,
  type CachedDistributionRecord,
  type SkillDistributionSummary,
} from './distributionCache';
import { downloadSkillAsZip } from './downloadUtils';

// localStorage 缓存与 mock 数据统一由 skillStore 提供（localStorage + CustomEvent 共享）
const loadCachedSkills = (): Skill[] => skillStore.getAll();
const saveCachedSkills = (skills: Skill[]) => skillStore.replaceAll(skills);

interface SkillListTabProps {
  onSelectSkill?: (skillId: string) => void;
  securityServiceActive?: boolean;
}

/** 仅当子元素文本溢出（出现 ...）时，hover 1s 后才显示 Tooltip */
function OverflowTooltip({ content, children }: { content: React.ReactNode; children: React.ReactElement }) {
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLElement | null>(null);

  return (
    <Tooltip
      delayDuration={1000}
      open={open}
      onOpenChange={(next) => {
        if (next) {
          const el = triggerRef.current;
          if (el && el.scrollWidth > el.clientWidth) {
            setOpen(true);
          }
          // 没溢出时 open 保持 false，tooltip 不弹
        } else {
          setOpen(false);
        }
      }}
    >
      <TooltipTrigger asChild ref={triggerRef as any}>
        {children}
      </TooltipTrigger>
      <TooltipContent side="top">{content}</TooltipContent>
    </Tooltip>
  );
}

export default function SkillListTab({ onSelectSkill, securityServiceActive: securityServiceActiveProp }: SkillListTabProps) {
  const [, setLocation] = useLocation();
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false);
  const [skills, setSkills] = useState<Skill[]>(loadCachedSkills);
  const [selectedSkillId, setSelectedSkillId] = useState<string | null>(null);
  const [defaultTabForDetail, setDefaultTabForDetail] = useState<string>('overview');
  const [viewMode, setViewMode] = useState<'card' | 'list'>('list');
  const [distributeDialogOpen, setDistributeDialogOpen] = useState(false);
  const [distributeSkillId, setDistributeSkillId] = useState<string | null>(null);
  const [editCategoryDialogOpen, setEditCategoryDialogOpen] = useState(false);
  const [editingSkillId, setEditingSkillId] = useState<string | null>(null);
  const [editingSkillCategories, setEditingSkillCategories] = useState<string[]>([]);
  const [categories, setCategories] = useState(DEFAULT_CATEGORIES);
  // 标签分类管理弹窗
  const [categoryManageDialogOpen, setCategoryManageDialogOpen] = useState(false);
  const [updateDialogOpen, setUpdateDialogOpen] = useState(false);
  const [updateSkillId, setUpdateSkillId] = useState<string | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteSkillId, setDeleteSkillId] = useState<string | null>(null);
  // 下架二次确认（管控端 MoreActions → 下架）
  const [offlineDialogOpen, setOfflineDialogOpen] = useState(false);
  const [offlineSkillId, setOfflineSkillId] = useState<string | null>(null);
  const [uninstallDialogOpen, setUninstallDialogOpen] = useState(false);
  const [uninstallSkillId, setUninstallSkillId] = useState<string | null>(null);
  const [downloadingSkillId, setDownloadingSkillId] = useState<string | null>(null);
  // 审核弹窗（对齐管控端 Demo #reviewOverlay）
  const [reviewDialogOpen, setReviewDialogOpen] = useState(false);
  const [reviewSkillId, setReviewSkillId] = useState<string | null>(null);
  // 安全检测确认弹窗
  const [securityScanDialogOpen, setSecurityScanDialogOpen] = useState(false);
  const [securityScanSkillId, setSecurityScanSkillId] = useState<string | null>(null);
  // 安全检测服务开通状态：优先使用 prop，否则从 localStorage 读取
  const [securityServiceActiveLocal, setSecurityServiceActiveLocal] = useState<boolean>(() => {
    const saved = localStorage.getItem('skill_security_service_active');
    return saved === 'true';
  });
  const securityServiceActive = securityServiceActiveProp !== undefined ? securityServiceActiveProp : securityServiceActiveLocal;
  const setSecurityServiceActive = (val: boolean) => {
    setSecurityServiceActiveLocal(val);
    localStorage.setItem('skill_security_service_active', String(val));
  };
  const [securityApplyDialogOpen, setSecurityApplyDialogOpen] = useState(false);
  const [securitySuccessDialogOpen, setSecuritySuccessDialogOpen] = useState(false);
  const [securityServiceUsed, setSecurityServiceUsed] = useState(156); // mock 已用额度
  // 默认行为设置（默认不勾选）
  const [defaultSecurityScan, setDefaultSecurityScan] = useState<boolean>(() => {
    const saved = localStorage.getItem('skill_default_security_scan');
    return saved !== null ? saved === 'true' : false;
  });
  // 应用范围筛选：含 'public'=全部用户, 含 'grp-xxx'=特定分组（多选）
  // 空 Set = 未选任何范围（按钮显示"选择应用范围"）；全选时包含 public + 所有 groupId
  const allScopeKeys = useMemo(() => ['public', ...MOCK_GROUPS.map(g => g.id)], []);
  const [selectedScopes, setSelectedScopes] = useState<Set<string>>(new Set());
  const [scopeDropdownOpen, setScopeDropdownOpen] = useState(false);
  // 保存编辑弹窗打开前的滚动位置（含表格水平滚动），关闭后恢复
  const scrollPositionRef = useRef<{ x: number; y: number; tableScrollLeft?: number } | null>(null);
  // 表格水平滚动容器 ref（用于保存/恢复滚动位置）
  const tableScrollRef = useRef<HTMLDivElement>(null);

  // 下发状态缓存：key 是 skillId，value 是摘要
  const [distributionSummaries, setDistributionSummaries] = useState<Record<string, SkillDistributionSummary>>({});

  // 标记自身写入，避免订阅回调造成循环
  const isSelfWritingSkills = useRef(false);
  // skills 变化时同步到共享 skillStore（localStorage + 广播）
  useEffect(() => {
    isSelfWritingSkills.current = true;
    saveCachedSkills(skills);
    isSelfWritingSkills.current = false;
  }, [skills]);

  // 订阅 skillStore：其他模块（含「项目资产管理」联动）变更时刷新列表
  useEffect(() => skillStore.subscribe(() => {
    if (isSelfWritingSkills.current) return;
    setSkills(skillStore.getAll());
  }), []);

  // 追踪已启动检测计时器的 skill ID，避免重复
  const scanTimersRef = useRef<Set<string>>(new Set());

  // 对所有处于 scanning 状态的 skill，mock 10s 后自动随机完成检测（实际提示为预计 5 分钟）
  useEffect(() => {
    const scanningSkills = skills.filter(
      s => s.securityInfo?.overallStatus === 'scanning' && !scanTimersRef.current.has(s.id)
    );
    if (scanningSkills.length === 0) return;

    const timers = scanningSkills.map(s => {
      scanTimersRef.current.add(s.id);
      return setTimeout(() => {
        const rand = Math.random();
        let result: SecurityStatus;
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
        const engine2Status = result as 'safe' | 'suspicious' | 'malicious';

        setSkills(prev => {
          const target = prev.find(sk => sk.id === s.id);
          // 如果已不是 scanning（已被其他地方完成），跳过
          if (!target || target.securityInfo?.overallStatus !== 'scanning') {
            scanTimersRef.current.delete(s.id);
            return prev;
          }
          const updated = prev.map(sk =>
            sk.id === s.id
              ? {
                  ...sk,
                  securityInfo: {
                    overallStatus: result,
                    contentHash: Math.random().toString(36).slice(2, 18),
                    engines: [
                      { engineName: '科恩实验室', status: 'safe' as const, reportUrl: '#', score: 92, dimensions: safeDims },
                      { engineName: '云鼎实验室', status: engine2Status, reportUrl: '#', score: score2, dimensions: dims },
                    ],
                  },
                }
              : sk
          );
          saveCachedSkills(updated);
          const resultLabel = result === 'safe' ? '安全' : result === 'suspicious' ? '可疑' : '恶意';
          toast.info(`「${s.name}」安全检测完成：${resultLabel}`);
          return updated;
        });
        scanTimersRef.current.delete(s.id);
      }, 10000);
    });

    return () => timers.forEach(t => clearTimeout(t));
  }, [skills]);


  // 从缓存加载所有 skill 的下发摘要
  const refreshDistributionSummaries = useCallback(() => {
    const summaries: Record<string, SkillDistributionSummary> = {};
    skills.forEach(s => {
      const summary = getSkillDistributionSummary(s.id);
      if (summary) summaries[s.id] = summary;
    });
    setDistributionSummaries(summaries);
  }, [skills]);

  // 首次加载 + 监听缓存更新事件
  useEffect(() => {
    refreshDistributionSummaries();
    const handler = () => refreshDistributionSummaries();
    window.addEventListener('distribution-cache-updated', handler);
    return () => window.removeEventListener('distribution-cache-updated', handler);
  }, [refreshDistributionSummaries]);

  /*
   * 停服时仍允许「选择应用范围」下拉浮层内容可交互：
   * ScopeSelect 的浮层渲染在 Radix Portal 中（[data-radix-popper-content-wrapper]），
   * 不在页面 DOM 树内，无法被页面层的 data-billing-exempt 容器覆盖。
   * 这里在 dropdown 展开后定位最近挂载的 popover-content 并附加豁免标记，
   * 使 AdminDisabledOverlay 对浮层内的 button / [role="checkbox"] 等失效。
   * ScopeSelect 组件库本身未做修改，所有逻辑在当前页面内。
   */
  useEffect(() => {
    if (!scopeDropdownOpen) return;
    const timer = requestAnimationFrame(() => {
      const popovers = document.querySelectorAll<HTMLElement>(
        '[data-radix-popper-content-wrapper] [data-slot="popover-content"]'
      );
      const last = popovers[popovers.length - 1];
      if (last) {
        last.setAttribute("data-billing-exempt", "");
      }
    });
    return () => cancelAnimationFrame(timer);
  }, [scopeDropdownOpen]);

  /*
   * 停服时仍允许「更多」菜单中的「下载」项可点击：
   * 1. 触发按钮已在外层用 data-billing-exempt 包裹，可正常打开下拉；
   * 2. 但下拉浮层（Radix DropdownMenu Portal）中的菜单项不在页面 DOM 树内，
   *    且组件库 MoreActionsDropdown 未提供 per-item 豁免能力；
   * 3. 这里通过 MutationObserver 监听 document.body 增量挂载的 dropdown-menu-content，
   *    找出其中 label === "下载" 的菜单项并附加豁免标记，仅放行该单个菜单项，
   *    其余写操作项（卸载/上架/下架/删除）继续受停服约束。
   */
  useEffect(() => {
    const exemptDownloadMenuItem = (root: ParentNode) => {
      root.querySelectorAll<HTMLElement>(
        '[data-slot="dropdown-menu-content"] [data-slot="dropdown-menu-item"]'
      ).forEach((item) => {
        if (item.textContent?.trim() === "下载") {
          item.setAttribute("data-billing-exempt", "");
        }
      });
    };

    // 初始化时处理已挂载的（兜底）
    exemptDownloadMenuItem(document.body);

    const observer = new MutationObserver((mutations) => {
      for (const mutation of mutations) {
        mutation.addedNodes.forEach((node) => {
          if (node instanceof HTMLElement) {
            exemptDownloadMenuItem(node);
          }
        });
      }
    });
    observer.observe(document.body, { childList: true, subtree: true });
    return () => observer.disconnect();
  }, []);

  const getCategoryName = (catId: string) => {
    return DEFAULT_CATEGORIES.find((cat: any) => cat.id === catId)?.name || catId;
  };

  const getGroupName = (groupId: string) => {
    return MOCK_GROUPS.find(g => g.id === groupId)?.name || groupId;
  };

  /** 获取 Skill 的应用范围显示标签数组 */
  const getScopeLabels = (skill: Skill): string[] => {
    if (skill.scope === 'public' || !skill.groupIds || skill.groupIds.length === 0) {
      return ['全部用户'];
    }
    return skill.groupIds.map(id => getGroupName(id));
  };

  /** 获取 Skill 的应用范围显示文本（用于卡片等单行场景） */
  const getScopeDisplay = (skill: Skill) => {
    return getScopeLabels(skill).join('、');
  };

  const filteredSkills = skills.filter((skill: any) => {
    const matchesSearch = skill.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      skill.description.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCategory = selectedCategory === null ||
      skill.categories.some((catId: string) => catId === selectedCategory);
    // 应用范围筛选（多选）
    let matchesScope = true;
    if (selectedScopes.size === 0) {
      // 没有选中任何范围 → 不筛选，显示全部
      matchesScope = true;
    } else {
      const hasPublic = selectedScopes.has('public');
      const groupScopes = Array.from(selectedScopes).filter(s => s !== 'public');
      // 满足任一选中条件即匹配
      const matchPublic = hasPublic && (skill.scope === 'public' || !skill.groupIds || skill.groupIds.length === 0);
      const matchGroup = groupScopes.length > 0 && skill.scope === 'private' && skill.groupIds?.some((gid: string) => selectedScopes.has(gid));
      matchesScope = !!(matchPublic || matchGroup);
    }
    return matchesSearch && matchesCategory && matchesScope;
  });

  const sortedSkills = [...filteredSkills].sort((a, b) => {
    // 待审核 Skill 永远置顶，避免管理员遗漏审批
    const aPending = a.reviewStatus === 'pending' ? 1 : 0;
    const bPending = b.reviewStatus === 'pending' ? 1 : 0;
    if (aPending !== bPending) return bPending - aPending;
    return b.uploadTime.getTime() - a.uploadTime.getTime();
  });

  const handleUploadSkill = (skillData: any) => {
    // skillData 已经是 SkillUploadDialog 中构造好的完整 Skill 对象
    // 确保必要字段存在
    const newSkill: Skill = {
      ...skillData,
      id: skillData.id || `skill-${Date.now()}`,
      uploadTime: skillData.uploadTime instanceof Date ? skillData.uploadTime : new Date(),
      versions: skillData.versions || [skillData.version || '1.0.0'],
      files: skillData.files || [],
    };
    setSkills(prev => {
      const updated = [...prev, newSkill];
      // 立即同步缓存，确保不丢数据
      saveCachedSkills(updated);
      return updated;
    });
  };

  // 安全检测提交确认
  const handleSecurityScanConfirm = () => {
    if (!securityScanSkillId) return;
    setSkills(prev => prev.map(s =>
      s.id === securityScanSkillId
        ? { ...s, securityInfo: { overallStatus: 'scanning' as SecurityStatus, engines: [] } }
        : s
    ));
    toast.success('已提交安全检测，预计 5 分钟后完成');
    setSecurityScanDialogOpen(false);
    setSecurityScanSkillId(null);
    // 模拟：10秒后随机变为安全/可疑/恶意（mock 模拟，实际预计 5 分钟）
    const targetId = securityScanSkillId;
    setTimeout(() => {
      const rand = Math.random();
      let result: SecurityStatus;
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
      const score = result === 'safe' ? 92 : result === 'suspicious' ? 55 : 15;
      const engineStatus = result as 'safe' | 'suspicious' | 'malicious';

      setSkills(prev => prev.map(s =>
        s.id === targetId && s.securityInfo?.overallStatus === 'scanning'
          ? {
              ...s,
              securityInfo: {
                overallStatus: result,
                contentHash: Math.random().toString(36).slice(2, 18),
                engines: [
                  { engineName: '腾讯云 AI Agent 安全', status: engineStatus, reportUrl: '#', score, dimensions: dims },
                ],
              },
            }
          : s
      ));
      const resultLabel = result === 'safe' ? '安全' : result === 'suspicious' ? '可疑' : '恶意';
      toast.info(`安全检测完成：${resultLabel}`);
    }, 10000);
  };

  const handleViewDetail = (skillId: string) => {
    if (onSelectSkill) {
      onSelectSkill(skillId);
    } else {
      setSelectedSkillId(skillId);
    }
  };

  const handleDistribute = (skillId: string) => {
    setDistributeSkillId(skillId);
    setDistributeDialogOpen(true);
  };

  const handleDistributeStart = (selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    if (!distributeSkillId) return;
    
    // 创建下发记录并写入缓存
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId: distributeSkillId,
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
        distributionStatus: 'distributing' as const,
      })),
    };
    addDistributionRecord(newRecord);

    // 关闭对话框
    setDistributeDialogOpen(false);
    
    // 显示下发开始通知
    toast.success('已开始下发流程');

    // 模拟进度更新
    const totalCount = selectedInstanceIds.length;
    let completed = 0;
    const interval = setInterval(() => {
      completed += Math.floor(Math.random() * 3) + 1;
      if (completed >= totalCount) {
        completed = totalCount;
        clearInterval(interval);
        // 模拟随机失败
        const failedCount = Math.floor(Math.random() * 2);
        const successCount = totalCount - failedCount;
        // 完成下发 - 更新缓存
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount,
          failedCount,
          inProgressCount: 0,
          status: failedCount === 0 ? 'success' : 'failed',
          instances: record.instances.map((inst, idx) => ({
            ...inst,
            distributionStatus: idx < successCount ? 'success' as const : 'failed' as const,
          })),
        }));
        toast.success('下发完成');
      } else {
        // 更新进度 - 更新缓存
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount: completed,
          inProgressCount: totalCount - completed,
        }));
      }
    }, 800);
  };

  const handleViewDistributeProgress = () => {
    // 跳转到详情页的下发记录 Tab
    if (distributeSkillId) {
      setSelectedSkillId(distributeSkillId);
      setDistributeDialogOpen(false);
      setDefaultTabForDetail('distribution');
      // 设置默认 Tab 为下发记录
      setTimeout(() => {
        const tabTrigger = document.querySelector('[value="distribution"]') as HTMLElement;
        if (tabTrigger) tabTrigger.click();
      }, 100);
    }
  };

  // 更新 Skill
  const handleUpdate = (skillId: string) => {
    setUpdateSkillId(skillId);
    setUpdateDialogOpen(true);
  };

  const handleSkillUpdated = (updatedSkill: Skill) => {
    setSkills(prev => {
      const updated = prev.map(s => s.id === updatedSkill.id ? updatedSkill : s);
      saveCachedSkills(updated);
      return updated;
    });
    setUpdateDialogOpen(false);
    setUpdateSkillId(null);
  };


  // 删除 Skill
  const handleDelete = (skillId: string) => {
    setDeleteSkillId(skillId);
    setDeleteDialogOpen(true);
  };

  const handleSkillDeleted = () => {
    if (!deleteSkillId) return;
    const skillName = skills.find(s => s.id === deleteSkillId)?.name || '';
    const deletedId = deleteSkillId;
    setSkills(prev => {
      const updated = prev.filter(s => s.id !== deleteSkillId);
      saveCachedSkills(updated);
      return updated;
    });
    projectAssetStore.onLibraryItemDeleted('enterpriseSkill', deletedId);
    toast.success(`Skill「${skillName}」已删除`);
    setDeleteDialogOpen(false);
    setDeleteSkillId(null);
  };

  // 审核（对齐管控端 Demo #reviewOverlay）
  const handleReview = (skillId: string) => {
    setReviewSkillId(skillId);
    setReviewDialogOpen(true);
  };

  const handleApproveSkill = (skillId: string, reviewType: 'publish' | 'offshelf') => {
    const target = skills.find(s => s.id === skillId);
    setSkills(prev => {
      const updated = prev.map(s => {
        if (s.id !== skillId) return s;
        if (reviewType === 'offshelf') {
          // 下架申请通过 → offlined，清理审核态字段
          return {
            ...s,
            reviewStatus: 'offlined' as const,
            reviewType: undefined,
            applicant: undefined,
            submittedAt: undefined,
            offshelfReason: undefined,
          };
        }
        // 发布申请通过 → normal，清理审核态字段（不再改动 categories，尊重提交时的选择）
        return {
          ...s,
          reviewStatus: 'normal' as const,
          reviewType: undefined,
          applicant: undefined,
          submittedAt: undefined,
        };
      });
      saveCachedSkills(updated);
      return updated;
    });
    toast.success(
      reviewType === 'offshelf'
        ? `Skill「${target?.name ?? ''}」已审核通过并下架`
        : `Skill「${target?.name ?? ''}」已通过审核并发布`
    );
    setReviewSkillId(null);
  };

  const handleRejectSkill = (skillId: string, reviewType: 'publish' | 'offshelf', _reason: string) => {
    const target = skills.find(s => s.id === skillId);
    setSkills(prev => {
      if (reviewType === 'offshelf') {
        // 下架驳回 → 回到 normal，Skill 保留可用
        const updated = prev.map(s =>
          s.id === skillId
            ? {
                ...s,
                reviewStatus: 'normal' as const,
                reviewType: undefined,
                applicant: undefined,
                submittedAt: undefined,
                offshelfReason: undefined,
              }
            : s
        );
        saveCachedSkills(updated);
        return updated;
      }
      // 发布驳回 → 从列表移除
      const updated = prev.filter(s => s.id !== skillId);
      saveCachedSkills(updated);
      return updated;
    });
    toast.success(
      reviewType === 'offshelf'
        ? `已驳回下架申请，Skill「${target?.name ?? ''}」已恢复为正常`
        : `已驳回 Skill「${target?.name ?? ''}」的发布申请`
    );
    setReviewSkillId(null);
  };

  // 管控端直接下架（不经过审核，即时生效）
  const handleOfflineSkill = (skillId: string) => {
    const target = skills.find(s => s.id === skillId);
    setSkills(prev => {
      const updated = prev.map(s =>
        s.id === skillId ? { ...s, reviewStatus: 'offlined' as const } : s
      );
      saveCachedSkills(updated);
      return updated;
    });
    toast.success(`Skill「${target?.name ?? ''}」已下架`);
  };

  // 管控端上架（把 offlined 恢复为 normal）
  const handleOnlineSkill = (skillId: string) => {
    const target = skills.find(s => s.id === skillId);
    setSkills(prev => {
      const updated = prev.map(s =>
        s.id === skillId ? { ...s, reviewStatus: 'normal' as const } : s
      );
      saveCachedSkills(updated);
      return updated;
    });
    toast.success(`Skill「${target?.name ?? ''}」已上架`);
  };

  // 卸载 Skill（从实例上移除）
  const handleUninstall = (skillId: string) => {
    setUninstallSkillId(skillId);
    setUninstallDialogOpen(true);
  };

  /** 从下发记录中聚合某 Skill 已下发成功的实例列表（用于卸载弹窗） */
  const distributedInstancesForUninstall = useMemo(() => {
    if (!uninstallSkillId) return [];
    const allRecords = getAllDistributionRecords();
    const instanceMap = new Map<string, any>();
    allRecords
      .filter(r => r.skillId === uninstallSkillId && (r.type || 'distribute') === 'distribute')
      .forEach(r => {
        r.instances.forEach(inst => {
          if (inst.distributionStatus === 'success' && !instanceMap.has(inst.id)) {
            const fullInst = MOCK_OPENCLAW_INSTANCES.find(i => i.id === inst.id);
            const groupName = fullInst?.groupIds?.[0]
              ? MOCK_GROUPS.find(g => g.id === fullInst.groupIds[0])?.name
              : undefined;
            const skill = skills.find(s => s.id === uninstallSkillId);
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
  }, [uninstallSkillId, skills]);

  const handleUninstallStart = (selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    if (!uninstallSkillId) return;
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId: uninstallSkillId,
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'distributing',
      type: 'delete',
      operator: 'admin',
      instances: selectedInstancesData.map(inst => ({
        id: inst.id,
        name: inst.name,
        createdBy: inst.createdBy || 'admin',
        distributionStatus: 'distributing' as const,
      })),
    };
    addDistributionRecord(newRecord);
    setUninstallDialogOpen(false);
    toast.success('已开始卸载流程');

    const totalCount = selectedInstanceIds.length;
    let completed = 0;
    const failReasons = ['实例离线', '权限不足', '技能被占用', '网络超时', '实例已停止'];
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
          instances: record.instances.map((inst, idx) => ({
            ...inst,
            distributionStatus: (results[idx] ? 'success' : 'failed') as 'success' | 'failed',
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

  // 下载 Skill
  const handleDownload = async (skill: Skill) => {
    setDownloadingSkillId(skill.id);
    try {
      await downloadSkillAsZip(skill);
      toast.success(`「${skill.name}」下载完成`);
    } catch {
      toast.error('下载失败，请重试');
    } finally {
      setDownloadingSkillId(null);
    }
  };

  /** 检查某个 skill 是否有进行中的下发或删除（用于禁用按钮） */
  const isDistributing = (skillId: string): boolean => {
    const summary = distributionSummaries[skillId];
    return summary?.hasInProgress || false;
  };

  // 如果选中了 Skill，显示详情页
  if (selectedSkillId) {
    return (
      <SkillDetail
        skillId={selectedSkillId}
        skills={skills}
        onBack={() => {
          setSelectedSkillId(null);
          setDefaultTabForDetail('overview');
        }}
        defaultTab={defaultTabForDetail}
        onSkillUpdate={(updatedSkill) => {
          setSkills(prev => {
            const updated = prev.map(s => s.id === updatedSkill.id ? updatedSkill : s);
            saveCachedSkills(updated);
            return updated;
          });
        }}
        onSkillDelete={(id) => {
          setSkills(prev => {
            const updated = prev.filter(s => s.id !== id);
            saveCachedSkills(updated);
            return updated;
          });
          projectAssetStore.onLibraryItemDeleted('enterpriseSkill', id);
          setSelectedSkillId(null);
        }}
        securityServiceActive={securityServiceActive}
      />
    );
  }

  return (
    <div className="space-y-4">
      {/* 搜索和工具栏 */}
      <div className="flex items-center gap-2">
        {/*
          停服时仍允许「搜索框 + 选择应用范围」正常可用：纯导航/筛选类操作，不消耗管控台写权限。
          通过 data-billing-exempt 容器豁免 AdminDisabledOverlay 的全局禁用样式与点击拦截。
          应用范围下拉浮层（Popover Portal）的内容另由下方 useEffect 在展开时单独豁免。
        */}
        <div data-billing-exempt className="flex items-center gap-2 flex-1 min-w-0">
          {/* 搜索框 */}
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-[var(--text-weak)] w-4 h-4" />
            <Input
              placeholder="搜索技能名称或描述..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10 bg-white border border-gray-200"
            />
          </div>

          {/* 应用范围下拉筛选 — ScopeSelect withTrigger（Portal 实现，不被裁切） */}
          <ScopeSelect
            withTrigger
            groups={MOCK_GROUPS}
            value={selectedScopes}
            onChange={setSelectedScopes}
            triggerLabel={
              selectedScopes.size === 0
                ? undefined
                : selectedScopes.size === allScopeKeys.length && allScopeKeys.every(k => selectedScopes.has(k))
                  ? '全部应用范围'
                  : Array.from(selectedScopes).map(s => s === 'public' ? '全部用户' : MOCK_GROUPS.find(g => g.id === s)?.name || s).join('、')
            }
            triggerPlaceholder="选择应用范围"
            triggerClassName="flex items-center justify-between gap-1 min-w-[10rem] max-w-[20rem] h-9 px-3 border border-gray-200 rounded-[4px] bg-white text-sm text-[#020617] hover:border-blue-500 data-[state=open]:border-blue-500 transition-colors outline-none"
            align="end"
            open={scopeDropdownOpen}
            onOpenChange={setScopeDropdownOpen}
          />
        </div>

        {/*
          停服时仍允许「视图切换（卡片/列表）」正常可用：纯展示类切换，不消耗管控台写权限。
          「+ 发布 Skill」写操作按钮不豁免，延续停服禁用。
        */}
        <div data-billing-exempt>
          <SegmentGroup>
            <SegmentOption active={viewMode === 'card'} onClick={() => setViewMode('card')} title="卡片视图">
              <Grid3x3 className="w-4 h-4" />
            </SegmentOption>
            <SegmentOption active={viewMode === 'list'} onClick={() => setViewMode('list')} title="列表视图">
              <List className="w-4 h-4" />
            </SegmentOption>
          </SegmentGroup>
        </div>

        <Button variant="claw-primary" size="claw-sm" onClick={() => setUploadDialogOpen(true)}>
          + 发布 Skill
        </Button>
      </div>

      {/* 分类筛选 */}
      <div className="flex items-start gap-2 mb-4 border-t border-gray-200 pt-4">
        {/*
          停服时仍允许技能分类筛选按钮正常可用：纯筛选类操作，不消耗管控台写权限。
          「标签分类管理」写操作按钮不豁免，延续停服禁用。
        */}
        <div data-billing-exempt className="flex items-center gap-2 flex-wrap flex-1 min-w-0">
          <Button variant="plain" size="sm" data-state={selectedCategory === null ? "active" : undefined} onClick={() => setSelectedCategory(null)}>
            全部
          </Button>
          {categories.map((cat: any) => (
            <Button key={cat.id} variant="plain" size="sm" data-state={selectedCategory === cat.id ? "active" : undefined} onClick={() => setSelectedCategory(cat.id)}>
              {cat.name}
            </Button>
          ))}
        </div>
        <Button
          variant="claw-outline"
          size="claw-sm"
          onClick={() => setCategoryManageDialogOpen(true)}
          className="flex-shrink-0"
        >
          <Settings2 className="w-4 h-4" />
          标签分类管理
        </Button>
      </div>

      {/* 空状态 */}
      {sortedSkills.length === 0 && (
        <div className="text-center py-12">
          <BodyText as="p" tone="muted">还没有发布任何 SKILL</BodyText>
        </div>
      )}

      {/* 卡片视图 */}
      {viewMode === 'card' && sortedSkills.length > 0 && (
        <div className="grid grid-cols-3 gap-4">
          {sortedSkills.map(skill => {
            const summary = distributionSummaries[skill.id];
            const distributing = isDistributing(skill.id);
            const isPending = skill.reviewStatus === 'pending';
            const isOfflined = skill.reviewStatus === 'offlined';
            return (
              <div
                key={skill.id}
                onClick={() => isPending ? handleReview(skill.id) : handleViewDetail(skill.id)}
                className={
                  isPending
                    ? 'relative flex flex-col overflow-hidden rounded-[4px] border border-[#B7D3FF] bg-[#F0F6FF] p-4 cursor-pointer transition-all hover:border-[#355EF1] group'
                    : 'relative flex flex-col overflow-hidden rounded-[4px] border border-gray-200 bg-white p-4 cursor-pointer transition-all hover:border-[#355EF1] group'
                }
              >
                {isPending && (
                  <div className="absolute right-3 top-3">
                    <StatusTag mode="text" variant="blue">待审核</StatusTag>
                  </div>
                )}
                {isOfflined && (
                  <div className="absolute right-3 top-3">
                    <StatusTag mode="text" variant="gray">已下架</StatusTag>
                  </div>
                )}
                {/* 头部：名称 + 安全检测图标 + 版本（右上） */}
                <div className="flex items-start justify-between gap-2 mb-3">
                  <div className="flex items-center gap-1.5 flex-1 min-w-0">
                    <CardHeading as="h3" className="truncate group-hover:text-[var(--text-brand)] transition-colors">{skill.name}</CardHeading>
                    {/* 安全检测小图标 */}
                    {(() => {
                      const secStatus = skill.securityInfo?.overallStatus || 'not_scanned';
                      if (secStatus === 'not_scanned') {
                        return (
                          <Tooltip delayDuration={300}>
                            <TooltipTrigger asChild>
                              <span className="inline-flex flex-shrink-0 cursor-default" onClick={(e) => e.stopPropagation()}>
                                <ShieldCheck className="w-3.5 h-3.5 text-[var(--text-weak)]" />
                              </span>
                            </TooltipTrigger>
                            <TooltipContent side="top">
                              <MetaText>未检测</MetaText>
                            </TooltipContent>
                          </Tooltip>
                        );
                      }
                      if (secStatus === 'scanning') {
                        return (
                          <Tooltip delayDuration={300}>
                            <TooltipTrigger asChild>
                              <span className="inline-flex flex-shrink-0 cursor-default" onClick={(e) => e.stopPropagation()}>
                                <Loader className="w-3.5 h-3.5 text-[var(--text-brand)] animate-spin" />
                              </span>
                            </TooltipTrigger>
                            <TooltipContent side="top">
                              <MetaText>安全检测中</MetaText>
                            </TooltipContent>
                          </Tooltip>
                        );
                      }
                      const statusInfo = SECURITY_STATUS_MAP[secStatus];
                      const IconComp = secStatus === 'safe' ? ShieldCheck : secStatus === 'suspicious' ? ShieldAlert : ShieldX;
                      return (
                        <Tooltip delayDuration={300}>
                          <TooltipTrigger asChild>
                            <span
                              className="inline-flex flex-shrink-0 cursor-pointer"
                              onClick={(e) => {
                                e.stopPropagation();
                                setDefaultTabForDetail('overview');
                                setSelectedSkillId(skill.id);
                              }}
                            >
                              <IconComp className={`w-3.5 h-3.5 ${statusInfo.color}`} />
                            </span>
                          </TooltipTrigger>
                          <TooltipContent side="top">
                            <MetaText>安全检测：{statusInfo.label}</MetaText>
                          </TooltipContent>
                        </Tooltip>
                      );
                    })()}
                  </div>
                  <Badge variant="secondary" className="tabular-nums shrink-0">
                    v{skill.version}
                  </Badge>
                </div>

                {/* 分类 — 标准 Badge variant="outline"，最多 3 个 + +N */}
                <div className="flex flex-wrap gap-1 mb-3 items-center">
                  {(() => {
                    const maxVisible = 3;
                    const total = skill.categories.length;
                    const visible = skill.categories.slice(0, maxVisible);
                    const overflow = total - maxVisible;
                    return (
                      <>
                        {visible.map((catId: string) => (
                          <Badge key={catId} variant="outline">
                            {getCategoryName(catId)}
                          </Badge>
                        ))}
                        {overflow > 0 && (
                          <Tooltip delayDuration={300}>
                            <TooltipTrigger asChild>
                              <span className="inline-flex" onClick={(e) => e.stopPropagation()}>
                                <Badge variant="outline" className="cursor-default">
                                  +{overflow}
                                </Badge>
                              </span>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-[320px]">
                              <div className="flex flex-wrap gap-1">
                                {skill.categories.map((catId: string) => (
                                  <MetaText as="span" key={catId}>
                                    {getCategoryName(catId)}
                                  </MetaText>
                                )).reduce<React.ReactNode[]>((acc, cur, idx) => {
                                  if (idx > 0) acc.push(<MetaText as="span" key={`sep-${idx}`}>,&nbsp;</MetaText>);
                                  acc.push(cur);
                                  return acc;
                                }, [])}
                              </div>
                            </TooltipContent>
                          </Tooltip>
                        )}
                        <Tooltip delayDuration={1000}>
                          <TooltipTrigger asChild>
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                scrollPositionRef.current = { x: window.scrollX, y: window.scrollY };
                                setEditingSkillId(skill.id);
                                setEditingSkillCategories(skill.categories);
                                setEditCategoryDialogOpen(true);
                              }}
                              className="p-0.5 text-[var(--text-weak)] hover:text-[var(--text-title)] rounded transition-colors"
                            >
                              <Edit2 className="w-3 h-3" />
                            </button>
                          </TooltipTrigger>
                          <TooltipContent side="top">编辑分类</TooltipContent>
                        </Tooltip>
                      </>
                    );
                  })()}
                </div>

                {/* 描述 — 两行截断 */}
                <Tooltip delayDuration={1000}>
                  <TooltipTrigger asChild>
                    <MetaText
                      as="p"
                      tone="muted"
                      className="line-clamp-2 mb-3 cursor-default leading-relaxed min-h-[34px]"
                    >
                      {skill.description || '-'}
                    </MetaText>
                  </TooltipTrigger>
                  {skill.description && skill.description.length > 60 && (
                    <TooltipContent side="bottom" className="max-w-[320px]">
                      <MetaText as="p" className="whitespace-pre-wrap">{skill.description}</MetaText>
                    </TooltipContent>
                  )}
                </Tooltip>

                {/* 应用范围 — 直接展示 outline badge（去掉冗余前缀文字） */}
                <div className="flex items-center gap-1 mb-3 flex-wrap" onClick={(e) => e.stopPropagation()}>
                  <MetaText tone="weak" className="mr-1">应用范围</MetaText>
                  <ScopeSelect
                    scope={(!skill.scope || skill.scope === 'public' || !skill.groupIds || skill.groupIds.length === 0) ? 'all' : 'groups'}
                    selectedGroupIds={skill.groupIds || []}
                    groups={MOCK_GROUPS}
                    projects={MOCK_PROJECT_GROUPS}
                    maxVisibleBadges={3}
                    onConfirm={(scope, groupIds) => {
                      setSkills(prev => prev.map(s =>
                        s.id === skill.id ? { ...s, scope: scope === 'all' ? 'public' : 'groups', groupIds } : s
                      ));
                      toast.success('应用范围修改成功');
                    }}
                  />
                </div>

                {/* 操作 — 左端下发+更新，右端更多；待审核 Skill 仅显示「审核」 */}
                <div className="flex items-center gap-2 pt-3 border-t border-[#F5F5F5]" onClick={(e) => e.stopPropagation()}>
                  {isPending ? (
                    <Button
                      variant="claw-primary"
                      size="sm"
                      onClick={() => handleReview(skill.id)}
                      className="h-8"
                    >
                      审核
                    </Button>
                  ) : (
                    <>
                      <Button
                        variant="claw-outline"
                        size="sm"
                        onClick={() => handleDistribute(skill.id)}
                        disabled={distributing}
                        className="h-8"
                      >
                        <Send className="size-3.5" />
                        {distributing ? (summary?.lastDistributionStatus === ('deleting' as any) ? '卸载中' : '下发中') : '下发'}
                      </Button>
                      <Button
                        variant="claw-outline"
                        size="sm"
                        onClick={() => handleUpdate(skill.id)}
                        disabled={distributing}
                        className="h-8"
                      >
                        <RefreshCw className="size-3.5" />
                        更新
                      </Button>
                      <div className="ml-auto" data-billing-exempt>
                        <MoreActionsDropdown
                          triggerType="icon"
                          align="end"
                          items={[
                            {
                              label: "下载",
                              icon: downloadingSkillId === skill.id ? Loader : Download,
                              onClick: () => handleDownload(skill),
                              disabled: downloadingSkillId === skill.id,
                            },
                            {
                              label: "卸载",
                              icon: PackageX,
                              onClick: () => handleUninstall(skill.id),
                              disabled: distributing,
                            },
                            isOfflined
                              ? {
                                  label: "上架",
                                  icon: ArrowUpFromLine,
                                  onClick: () => handleOnlineSkill(skill.id),
                                  disabled: distributing,
                                }
                              : {
                                  label: "下架",
                                  icon: ArrowDownToLine,
                                  onClick: () => {
                                    setOfflineSkillId(skill.id);
                                    setOfflineDialogOpen(true);
                                  },
                                  disabled: distributing,
                                },
                            {
                              label: "删除",
                              icon: Trash2,
                              onClick: () => handleDelete(skill.id),
                              disabled: distributing,
                              variant: "destructive" as const,
                            },
                          ]}
                        />
                      </div>
                    </>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* 表格视图 — 名称列固定左侧、操作列固定右侧，中间列可水平滚动 */}
      {viewMode === 'list' && sortedSkills.length > 0 && (
        <SurfaceCard className="overflow-hidden">
          <Table variant="white" containerRef={tableScrollRef} scrollX={1520}>
            <TableHeader>
              <TableRow>
                <TableHead fixed="left" className="w-[260px]" style={{ width: 260 }}>
                  技能信息
                </TableHead>
                <TableHead className="w-[100px]" style={{ width: 100 }}>状态</TableHead>
                <TableHead className="w-[160px]" style={{ width: 160 }}>下发</TableHead>
                <TableHead className="w-[80px]" style={{ width: 80 }}>版本</TableHead>
                <TableHead className="w-[360px]" style={{ width: 360 }}>描述</TableHead>
                <TableHead className="min-w-[160px]">分类</TableHead>
                <TableHead className="w-[190px]" style={{ width: 190 }}>应用范围</TableHead>
                <TableHead className="w-[130px]" style={{ width: 130 }}>最后更新</TableHead>
                <TableHead fixed="right" className="w-[168px]" style={{ width: 168 }}>
                  操作
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
                {sortedSkills.map(skill => {
                  const summary = distributionSummaries[skill.id];
                  const distributing = isDistributing(skill.id);
                  const isPending = skill.reviewStatus === 'pending';
                  const isOfflined = skill.reviewStatus === 'offlined';

                  // 下发/删除状态显示：两行结构
                  const hasDistribution = summary && summary.lastDistributionStatus !== 'not_distributed';
                  let statusLine1 = '正常'; // 第一行：状态
                  let statusLine2 = '未下发'; // 第二行：下发进度
                  let statusVariant: 'green' | 'blue' | 'red' | 'gray' = 'green';
                  let distributionVariant: 'green' | 'blue' | 'red' = 'red';
                  if (isPending) {
                    statusLine1 = '待审核';
                    statusVariant = 'blue';
                    statusLine2 = '—';
                  } else if (isOfflined) {
                    statusLine1 = '已下架';
                    statusVariant = 'gray';
                  } else if (summary) {
                    if (summary.lastDistributionStatus === 'deleting' as any) {
                      statusLine1 = '卸载中';
                      statusLine2 = `${summary.lastDistributionProgress || 0}%`;
                      statusVariant = 'red';
                      distributionVariant = 'red';
                    } else if (summary.lastDistributionStatus === 'distributing') {
                      statusLine1 = '下发中';
                      statusLine2 = `${summary.lastDistributionProgress || 0}%`;
                      statusVariant = 'blue';
                      distributionVariant = 'blue';
                    } else if (hasDistribution) {
                      const total = summary.lastDistributionInstanceCount || 0;
                      const success = summary.lastDistributionSuccessCount ?? total;
                      statusLine2 = `已下发（${success}/${total}成功）`;
                      distributionVariant = success === total ? 'green' : 'red';
                    }
                  }

                  return (
                    <TableRow
                      key={skill.id}
                      onClick={() => isPending ? handleReview(skill.id) : handleViewDetail(skill.id)}
                      className={isPending ? 'cursor-pointer bg-[#F0F6FF] hover:bg-[#E6EFFF]' : 'cursor-pointer'}
                    >
                      {/* 技能信息 — 固定左侧 */}
                      <TableCell
                        fixed="left"
                        className=""
                        style={{ width: 260 }}
                      >
                        <div className="min-w-0">
                          <OverflowTooltip content={skill.name}>
                            <BodyMedium as="div" tone="primary" className="truncate max-w-[230px]">{skill.name}</BodyMedium>
                          </OverflowTooltip>
                          <OverflowTooltip content={skill.slug}>
                            <MetaText as="div" className="mt-0.5 truncate max-w-[230px]">{skill.slug}</MetaText>
                          </OverflowTooltip>
                        </div>
                      </TableCell>
                      {/* 状态 — StatusTag 文本模式 */}
                      <TableCell className="">
                        <StatusTag mode="text" variant={statusVariant}>
                          {statusLine1}
                        </StatusTag>
                      </TableCell>
                      {/* 下发状态 — 纯文字 */}
                      <TableCell className="">
                        {hasDistribution ? (
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              setDefaultTabForDetail('distribution');
                              setSelectedSkillId(skill.id);
                            }}
                            className="text-sm text-[var(--text-secondary)] hover:text-[var(--text-brand)] transition-colors"
                            title={statusLine2}
                          >
                            {statusLine2}
                          </button>
                        ) : (
                          <BodyText as="span" tone="weak">{statusLine2}</BodyText>
                        )}
                      </TableCell>
                      {/* 版本号 */}
                      <TableCell className="">
                        <BodyText as="span" tone="secondary">v{skill.version}</BodyText>
                      </TableCell>
                      {/* 描述 */}
                      <TableCell className="" style={{ width: 360, overflow: 'hidden' }}>
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
                            >{skill.description || '-'}</BodyText>
                          </TooltipTrigger>
                          {skill.description && skill.description.length > 40 && (
                            <TooltipContent side="bottom" className="max-w-[400px]">
                              <MetaText as="p" className="whitespace-pre-wrap">{skill.description}</MetaText>
                            </TooltipContent>
                          )}
                        </Tooltip>
                      </TableCell>
                      {/* 分类 — 纯文本展示（用「/」分隔），自适应列宽，hover 展示全部，可点击编辑 */}
                      <TableCell className="" onClick={(e) => e.stopPropagation()}>
                        {(() => {
                          const names = skill.categories.map((catId: string) => getCategoryName(catId));
                          const fullText = names.join(' / ');
                          return (
                            <div className="flex items-center gap-1 min-w-0">
                              {names.length > 0 ? (
                                <Tooltip delayDuration={500}>
                                  <TooltipTrigger asChild>
                                    <BodyText as="span" tone="primary" className="whitespace-nowrap">
                                      {fullText}
                                    </BodyText>
                                  </TooltipTrigger>
                                  <TooltipContent side="top" className="max-w-[320px]">
                                    <MetaText>{fullText}</MetaText>
                                  </TooltipContent>
                                </Tooltip>
                              ) : (
                                <BodyText as="span" tone="weak">—</BodyText>
                              )}
                              <Tooltip delayDuration={1000}>
                                <TooltipTrigger asChild>
                                  <button
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      scrollPositionRef.current = { x: window.scrollX, y: window.scrollY, tableScrollLeft: tableScrollRef.current?.scrollLeft };
                                      setEditingSkillId(skill.id);
                                      setEditingSkillCategories(skill.categories);
                                      setEditCategoryDialogOpen(true);
                                    }}
                                    className="p-0.5 text-[var(--text-weak)] hover:text-[var(--text-title)] rounded transition-colors flex-shrink-0"
                                  >
                                    <Edit2 className="w-3 h-3" />
                                  </button>
                                </TooltipTrigger>
                                <TooltipContent side="top">
                                  编辑分类
                                </TooltipContent>
                              </Tooltip>
                            </div>
                          );
                        })()}
                      </TableCell>
                      {/* 应用范围 — 使用标准 ScopeSelect */}
                      <TableCell className="" onClick={(e) => e.stopPropagation()}>
                        <ScopeSelect
                          scope={(!skill.scope || skill.scope === 'public' || !skill.groupIds || skill.groupIds.length === 0) ? 'all' : 'groups'}
                          selectedGroupIds={skill.groupIds || []}
                          groups={MOCK_GROUPS}
                          projects={MOCK_PROJECT_GROUPS}
                          maxVisibleBadges={3}
                          onConfirm={(scope, groupIds) => {
                            setSkills(prev => prev.map(s =>
                              s.id === skill.id ? { ...s, scope: scope === 'all' ? 'public' : 'groups', groupIds } : s
                            ));
                            toast.success('应用范围修改成功');
                          }}
                        />
                      </TableCell>
                      {/* 最后更新时间 */}
                      <TableCell className="">
                        <BodyText as="span" tone="secondary" className="tabular-nums">
                          {skill.uploadTime.toLocaleDateString('zh-CN')}
                        </BodyText>
                      </TableCell>
                      {/* 操作 — 固定右侧：
                          - 待审核 Skill：仅「审核」入口
                          - 正常 Skill：下发 / 更新 / 更多(下载、删除) */}
                      <TableActionCell
                        fixed="right"
                        className=""
                        style={{ width: 168 }}
                        actionsClassName="h-5"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {isPending ? (
                          <Button
                            variant="link"
                            onClick={() => handleReview(skill.id)}
                          >
                            审核
                          </Button>
                        ) : (
                          <>
                            <Button
                              variant="link"
                              onClick={() => handleDistribute(skill.id)}
                              disabled={distributing}
                            >
                              {distributing ? (summary?.lastDistributionStatus === ('deleting' as any) ? '卸载中' : '下发中') : '下发'}
                            </Button>
                            <Button
                              variant="link"
                              onClick={() => handleUpdate(skill.id)}
                              disabled={distributing}
                            >
                              更新
                            </Button>
                            <span data-billing-exempt>
                            <MoreActionsDropdown
                              triggerType="text"
                              align="end"
                              items={[
                                ...((skill.securityInfo?.overallStatus === 'not_scanned' || !skill.securityInfo) ? [{
                                  label: "安全检测",
                                  icon: ScanSearch,
                                  onClick: () => {
                                    setSecurityScanSkillId(skill.id);
                                    setSecurityScanDialogOpen(true);
                                  },
                                }] : []),
                                {
                                  label: "下载",
                                  icon: downloadingSkillId === skill.id ? Loader : Download,
                                  onClick: () => handleDownload(skill),
                                  disabled: downloadingSkillId === skill.id,
                                },
                                {
                                  label: "卸载",
                                  icon: PackageX,
                                  onClick: () => handleUninstall(skill.id),
                                  disabled: distributing,
                                },
                                isOfflined
                                  ? {
                                      label: "上架",
                                      icon: ArrowUpFromLine,
                                      onClick: () => handleOnlineSkill(skill.id),
                                      disabled: distributing,
                                    }
                                  : {
                                      label: "下架",
                                      icon: ArrowDownToLine,
                                      onClick: () => {
                                        setOfflineSkillId(skill.id);
                                        setOfflineDialogOpen(true);
                                      },
                                      disabled: distributing,
                                    },
                                {
                                  label: "删除",
                                  icon: Trash2,
                                  onClick: () => handleDelete(skill.id),
                                  disabled: distributing,
                                  variant: "destructive" as const,
                                },
                              ]}
                            />
                            </span>
                          </>
                        )}
                      </TableActionCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
        </SurfaceCard>
      )}

      <SkillUploadDialog
        open={uploadDialogOpen}
        onOpenChange={setUploadDialogOpen}
        onConfirm={handleUploadSkill}
        existingSlugs={skills.map(s => s.slug)}
        defaultSecurityScan={defaultSecurityScan}
        onDefaultSecurityScanChange={(value) => {
          setDefaultSecurityScan(value);
          localStorage.setItem('skill_default_security_scan', String(value));
          toast.success('默认行为已保存');
        }}
        securityServiceActive={securityServiceActive}
      />

      {distributeSkillId && (
        <BatchDistributeDialog
          open={distributeDialogOpen}
          onOpenChange={setDistributeDialogOpen}
          skillName={skills.find(s => s.id === distributeSkillId)?.name || ''}
          skillVersion={skills.find(s => s.id === distributeSkillId)?.version}
          skillScope={skills.find(s => s.id === distributeSkillId)?.scope}
          skillGroupIds={skills.find(s => s.id === distributeSkillId)?.groupIds}
          onDistributionStart={handleDistributeStart}
          instances={MOCK_OPENCLAW_INSTANCES}
          groups={MOCK_GROUPS}
        />
      )}

      {/* 批量卸载对话框 */}
      {uninstallSkillId && (
        <BatchDeleteDialog
          open={uninstallDialogOpen}
          onOpenChange={(open) => {
            setUninstallDialogOpen(open);
            if (!open) setUninstallSkillId(null);
          }}
          skillName={skills.find(s => s.id === uninstallSkillId)?.name || ''}
          skillVersion={skills.find(s => s.id === uninstallSkillId)?.version || ''}
          distributedInstances={distributedInstancesForUninstall}
          groups={MOCK_GROUPS}
          onDeleteStart={handleUninstallStart}
        />
      )}

      {/* 更新对话框 */}
      {updateSkillId && (() => {
        const updateSkill = skills.find(s => s.id === updateSkillId);
        return updateSkill ? (
          <SkillUpdateDialog
            open={updateDialogOpen}
            onOpenChange={(open) => {
              setUpdateDialogOpen(open);
              if (!open) setUpdateSkillId(null);
            }}
            skill={updateSkill}
            onConfirm={handleSkillUpdated}
            defaultSecurityScan={defaultSecurityScan}
            onDefaultSecurityScanChange={(value) => {
              setDefaultSecurityScan(value);
              localStorage.setItem('skill_default_security_scan', String(value));
              toast.success('默认行为已保存');
            }}
            securityServiceActive={securityServiceActive}
          />
        ) : null;
      })()}

      {/* 删除确认对话框 */}
      {deleteSkillId && (
        <DeleteSkillDialog
          open={deleteDialogOpen}
          onOpenChange={(open) => {
            setDeleteDialogOpen(open);
            if (!open) setDeleteSkillId(null);
          }}
          skillName={skills.find(s => s.id === deleteSkillId)?.name || ''}
          onConfirm={handleSkillDeleted}
        />
      )}

      {/* 下架确认对话框（管控端 MoreActions → 下架） */}
      {offlineSkillId && (
        <OfflineSkillDialog
          open={offlineDialogOpen}
          onOpenChange={(open) => {
            setOfflineDialogOpen(open);
            if (!open) setOfflineSkillId(null);
          }}
          skillName={skills.find(s => s.id === offlineSkillId)?.name || ''}
          onConfirm={() => {
            if (offlineSkillId) {
              handleOfflineSkill(offlineSkillId);
            }
            setOfflineDialogOpen(false);
            setOfflineSkillId(null);
          }}
        />
      )}

      {/* 标签分类管理弹窗 */}
      <CategoryManagementDialog
        open={categoryManageDialogOpen}
        onOpenChange={setCategoryManageDialogOpen}
        categories={categories}
        setCategories={setCategories}
        skills={skills}
      />

      {/* 审核弹窗（对齐管控端 Demo #reviewOverlay） */}
      <SkillReviewDialog
        open={reviewDialogOpen}
        onOpenChange={(open) => {
          setReviewDialogOpen(open);
          if (!open) setReviewSkillId(null);
        }}
        skill={reviewSkillId ? skills.find(s => s.id === reviewSkillId) ?? null : null}
        categories={categories}
        onApprove={handleApproveSkill}
        onReject={handleRejectSkill}
      />

       {/* 编辑分类弹窗 */}
      <EditCategoriesDialog
        open={editCategoryDialogOpen}
        onOpenChange={(open) => {
          setEditCategoryDialogOpen(open);
          if (!open) {
            setEditingSkillId(null);
            setEditingSkillCategories([]);
            // 恢复弹窗打开前的滚动位置
            if (scrollPositionRef.current) {
              const saved = scrollPositionRef.current;
              requestAnimationFrame(() => {
                window.scrollTo(saved.x, saved.y);
                if (saved.tableScrollLeft !== undefined && tableScrollRef.current) {
                  tableScrollRef.current.scrollLeft = saved.tableScrollLeft;
                }
                scrollPositionRef.current = null;
              });
            }
          }
        }}
        categories={categories}
        selectedCategoryIds={editingSkillCategories}
        skillName={editingSkillId ? skills.find(s => s.id === editingSkillId)?.name : undefined}
        onConfirm={(selectedCategoryIds) => {
          if (editingSkillId) {
            setSkills(prev => prev.map(skill => 
              skill.id === editingSkillId 
                ? { ...skill, categories: selectedCategoryIds }
                : skill
            ));
            toast.success('分类修改成功');
            setEditCategoryDialogOpen(false);
            setEditingSkillId(null);
            setEditingSkillCategories([]);
            // 恢复弹窗打开前的滚动位置
            if (scrollPositionRef.current) {
              const saved = scrollPositionRef.current;
              requestAnimationFrame(() => {
                window.scrollTo(saved.x, saved.y);
                if (saved.tableScrollLeft !== undefined && tableScrollRef.current) {
                  tableScrollRef.current.scrollLeft = saved.tableScrollLeft;
                }
                scrollPositionRef.current = null;
              });
            }
          }
        }}
      />

      {/* 安全检测确认弹窗 */}
      <AlertDialog open={securityScanDialogOpen} onOpenChange={setSecurityScanDialogOpen}>
        <AlertDialogContent className="sm:max-w-[420px]">
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              提交安全检测
              <Badge variant="secondary" className="rounded-full bg-[#F0F2F8] text-[var(--text-brand)] text-[10px] px-2 py-0.5 border-0">限免</Badge>
            </AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription>
            {securityServiceUsed >= 1000 ? (
              '免费试用额度已用完，请前往官网提交工单提额，产品可选择 云安全中心。'
            ) : (
              <>确认对技能「{securityScanSkillId ? skills.find(s => s.id === securityScanSkillId)?.name : ''}」提交安全检测？检测将由腾讯云 AI Agent 安全进行，通常几分钟内完成。</>
            )}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => { setSecurityScanDialogOpen(false); setSecurityScanSkillId(null); }}>取消</AlertDialogCancel>
            <AlertDialogAction
              variant="dialog-confirm"
              onClick={handleSecurityScanConfirm}
              disabled={securityServiceUsed >= 1000}
              className="disabled:cursor-not-allowed"
            >
              确认检测
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 安全检测服务 — 申请开通弹窗 (row 49) - 普通弹窗 */}
      <Dialog open={securityApplyDialogOpen} onOpenChange={setSecurityApplyDialogOpen}>
        <DialogContent
          className="sm:max-w-md"
          style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
        >
          <DialogHeader>
            <DialogTitle>申请免费试用（Skills 风险检测 API）</DialogTitle>
          </DialogHeader>
          <DialogBody className="flex-1">
            <div className="rounded-[4px] border border-gray-200 bg-white px-4 py-3 space-y-2.5">
              <div className="flex items-center justify-between">
                <MetaMedium tone="muted">试用有效期</MetaMedium>
                <BodyText as="span" tone="primary">有效期至 2026 年 6 月 30 日</BodyText>
              </div>
              <div className="flex items-center justify-between">
                <MetaMedium tone="muted">调用额度</MetaMedium>
                <div className="text-right">
                  <BodyText as="span" tone="primary">1000 次</BodyText>
                  <MetaText as="p">有效期到期后，剩余未使用的额度将清空</MetaText>
                </div>
              </div>
              <div className="flex items-center justify-between">
                <MetaMedium tone="muted">操作指引</MetaMedium>
                <a
                  href="https://cloud.tencent.com/document/api/664/131590"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-[var(--text-brand)] hover:underline flex items-center gap-1 text-sm"
                >
                  说明文档
                  <ExternalLink className="w-3.5 h-3.5" />
                </a>
              </div>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button variant="outline" onClick={() => setSecurityApplyDialogOpen(false)}>
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              onClick={() => {
                setSecurityServiceActive(true);
                localStorage.setItem('skill_security_service_active', 'true');
                setSecurityApplyDialogOpen(false);
                setSecuritySuccessDialogOpen(true);
              }}
            >
              立即领取
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 安全检测服务 — 开通成功弹窗 */}
      <Dialog open={securitySuccessDialogOpen} onOpenChange={setSecuritySuccessDialogOpen}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-base">
              <span className="w-6 h-6 rounded-full bg-green-100 flex items-center justify-center">
                <Check className="w-4 h-4 text-[var(--text-success)]" />
              </span>
              试用额度已开通
            </DialogTitle>
            <DialogDescription className="pt-2">
              1000次调用额度，有效期至 2026-06-30
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <div>
              <BodyMedium as="p" tone="primary" className="mb-1">使用 API</BodyMedium>
              <BodyText as="p" tone="muted">
                您可以前往查看{' '}
                <a
                  href="https://cloud.tencent.com/document/api/664/131590"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-[var(--text-brand)] hover:text-[var(--text-brand)] inline-flex items-center gap-0.5"
                >
                  说明文档
                  <ExternalLink className="w-3 h-3" />
                </a>
                ，基于说明文档调用并测试 API。
              </BodyText>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="dialog-confirm"
              onClick={() => setSecuritySuccessDialogOpen(false)}
            >
              我知道了
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

    </div>
  );
}
