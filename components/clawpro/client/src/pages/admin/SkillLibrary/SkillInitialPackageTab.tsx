/**
 * 技能初始包 Tab
 * 设计风格：浅色主题，草稿+发布分离，生效开关
 */
import { useState, useRef, useEffect, useMemo, type KeyboardEvent } from 'react';
import { Pagination } from '@/components/ui/pagination';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { StatusTag } from '@/components/ui/status-tag';
import { SkillSelectCard } from '@/components/ui/skill-select-card';
import { AllUsersTag } from '@/components/ui/all-users-tag';
import { Switch } from '@/components/ui/switch';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogBody,
} from '@/components/ui/dialog';
import {
  Empty,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
  EmptyDescription,
  EmptyContent,
} from '@/components/ui/empty';
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
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  InstantMultiSelect,
} from '@/components/ui/select';
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
  TableActionCell,
} from '@/components/ui/table';
import {
  Plus, Trash2, ArrowLeft, Package, Globe, CircleAlert,
  CheckCircle2, Clock, ChevronRight, X, AlertCircle, Sparkles,
  Search, RefreshCw, ChevronDown, Check, Edit2, Filter, Users, Pin, Info
} from 'lucide-react';
import { INITIAL_SKILL_PACKAGES_DEFAULT, PUBLIC_SKILLS, PUBLIC_SKILL_CATEGORIES, type PublicSkill, type SkillInitialPackage, type PackageSkillItem } from './publicSkillMockData';
import {
  PUBLIC_SKILL_PACKAGES,
  PUBLIC_SKILL_PACKAGE_CATEGORIES,
  type PackageSkillRef,
  type PublicSkillPackage,
} from './publicSkillPackageMockData';
import { Heart, Download } from 'lucide-react';
import { Star } from 'lucide-react';
import { MOCK_SKILLS, DEFAULT_CATEGORIES, MOCK_GROUPS } from './mockData';
import { FilterChipGroup } from '@/components/ui/filter-chip';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { ScopeSelect, type ScopeType } from '@/components/ScopeSelect';
import { MOCK_GROUPS as MOCK_ONEID_GROUPS, MOCK_MANUAL_GROUPS } from '../MemberManagement/mock';
import type { UserGroup } from '../MemberManagement/types';
import { type SkillScope } from './types';
import { SurfaceCard } from '@/components/ui/Surface';
import {
  BodyText,
  BodyMedium,
  CompactText,
  HelperText,
  MetaText,
  MetaMedium,
  PanelTitle,
  CardTitle,
} from '@/components/ui/Typography';

// ─── 公共技能库添加弹窗 ──────────────────────────────────────────────────────────

const PUBLIC_SKILL_DIALOG_PAGE_SIZE = 6;
const PUBLIC_PACKAGE_DIALOG_PAGE_SIZE = 4;
const PUBLIC_SKILL_DIALOG_CATEGORIES = PUBLIC_SKILL_CATEGORIES
  .map((category) => ({ id: category.id, label: category.name }));
const PUBLIC_PACKAGE_DIALOG_CATEGORIES = PUBLIC_SKILL_PACKAGE_CATEGORIES
  .map((category) => ({ id: category.id, label: category.name }));
const DEFAULT_FAVORITE_PUBLIC_SKILL_IDS = PUBLIC_SKILLS.slice(0, 5).map((skill) => skill.id);
const DEFAULT_FAVORITE_PUBLIC_PACKAGE_IDS = PUBLIC_SKILL_PACKAGES.slice(0, 4).map((pkg) => pkg.id);

type AddPublicSkillDialogTab = 'public-skill' | 'public-package';

interface AddPublicSkillDialogProps {
  open: boolean;
  existingSkillIds: string[];
  onConfirm: (skills: PackageSkillItem[]) => void;
  onCancel: () => void;
}

function formatPublicSkillCount(n: number) {
  if (n >= 10000) { return `${parseFloat((n / 10000).toFixed(1))}万`; }
  if (n >= 1000) { return `${parseFloat((n / 1000).toFixed(1))}千`; }
  return String(n);
}

function AddPublicSkillCard({ skill, rank, isSelected, isAlreadyAdded, isFavorited, onToggle, onFavorite }: {
  skill: PublicSkill; rank: number; isSelected: boolean; isAlreadyAdded: boolean; isFavorited: boolean; onToggle: () => void; onFavorite: () => void;
}) {
  const handleCardKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (isAlreadyAdded) return;
    if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); onToggle(); }
  };
  return (
    <div role="button" tabIndex={isAlreadyAdded ? -1 : 0} aria-disabled={isAlreadyAdded} aria-pressed={isSelected}
      onClick={() => { if (!isAlreadyAdded) onToggle(); }} onKeyDown={handleCardKeyDown}
      className={`group relative flex min-w-0 flex-col overflow-hidden rounded-[4px] border bg-white p-4 text-left transition-all ${
        isAlreadyAdded ? 'cursor-not-allowed border-gray-200 bg-[#FAFAFA] opacity-40'
          : isSelected ? 'cursor-pointer border-blue-500 bg-[rgba(20,71,230,0.06)]'
            : 'cursor-pointer border-gray-200 hover:border-blue-500 hover:shadow-sm'
      }`}
    >
      <div className="mb-1 flex min-w-0 items-center gap-2">
        <CardTitle as="h3" tone="primary" className="min-w-0 flex-1 truncate font-semibold leading-tight transition-colors group-hover:text-[var(--text-brand)]" title={skill.name}>
          {skill.name}
        </CardTitle>
        <div className="flex shrink-0 items-center gap-1.5">
          {rank === 1 && <StatusTag mode="fill" variant="gray">Top 1</StatusTag>}
          {rank === 2 && <StatusTag mode="fill" variant="blue">Top 2</StatusTag>}
          {rank === 3 && <StatusTag mode="fill" variant="green">Top 3</StatusTag>}
          {isAlreadyAdded && <StatusTag mode="fill" variant="gray">已添加</StatusTag>}
          {isSelected && !isAlreadyAdded && (
            <span className="inline-flex size-5 items-center justify-center rounded-full bg-blue-500 shrink-0">
              <Check className="size-3 text-white" />
            </span>
          )}
        </div>
      </div>
      <MetaText as="p" className="line-clamp-2 break-words leading-relaxed" style={{ minHeight: '2.5rem' }}>{skill.descriptionZh}</MetaText>
      <div className="mt-3 flex items-center justify-between gap-3">
        <MetaText as="div" tone="weak" className="flex min-w-0 items-center gap-3">
          <span className="flex items-center gap-1"><Download className="size-3" />{formatPublicSkillCount(skill.downloads)}</span>
          <span className="flex items-center gap-1"><Star className="size-3" />{formatPublicSkillCount(skill.stars)}</span>
          <span className="font-mono">v{skill.version}</span>
        </MetaText>
        <button type="button" onClick={(e) => { e.stopPropagation(); onFavorite(); }}
          className={`flex size-7 shrink-0 items-center justify-center rounded-[4px] transition-colors ${isFavorited ? 'bg-red-50 text-[var(--text-danger)] hover:bg-red-100' : 'text-[var(--text-weak)] hover:bg-red-50 hover:text-[var(--text-danger)]'}`}
          title={isFavorited ? '取消收藏' : '添加到我的收藏'} aria-label={isFavorited ? '取消收藏' : '添加到我的收藏'}
        ><Heart className={`size-3.5 ${isFavorited ? 'fill-current' : ''}`} /></button>
      </div>
    </div>
  );
}

function DialogPackageSkillChip({ skill }: { skill: PackageSkillRef }) {
  return (
    <span className="inline-flex max-w-[140px] items-center gap-1 overflow-hidden rounded-[4px] border border-[var(--cp-border)] bg-[var(--bg-grey-hover-subtle)] px-2 py-0.5">
      <Package className="size-3 shrink-0 text-[var(--text-weak)]" />
      <span className="min-w-0 truncate text-[11px] font-medium text-[var(--text-secondary)]">{skill.name}</span>
    </span>
  );
}

function AddPublicPackageCard({ pkg, isSelected, isFavorited, existingSkillCount, onToggle, onFavorite }: {
  pkg: PublicSkillPackage; isSelected: boolean; isFavorited: boolean; existingSkillCount: number; onToggle: () => void; onFavorite: () => void;
}) {
  const visibleSkills = pkg.skills.slice(0, 4);
  const overflowCount = Math.max(0, pkg.skills.length - visibleSkills.length);
  return (
    <div role="button" tabIndex={0} aria-pressed={isSelected} onClick={onToggle}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onToggle(); } }}
      className={`group relative flex min-w-0 cursor-pointer flex-col overflow-hidden rounded-[4px] border bg-white p-4 text-left transition-all ${
        isSelected ? 'border-blue-500 bg-[rgba(20,71,230,0.06)]' : 'border-gray-200 hover:border-blue-500 hover:shadow-sm'
      }`}
    >
      <div className="mb-1 flex min-w-0 items-center gap-2">
        <CardTitle as="h3" tone="primary" className="min-w-0 flex-1 truncate font-semibold leading-tight transition-colors group-hover:text-[var(--text-brand)]" title={pkg.name}>{pkg.name}</CardTitle>
        {isSelected && (
          <span className="inline-flex size-5 items-center justify-center rounded-full bg-blue-500 shrink-0">
            <Check className="size-3 text-white" />
          </span>
        )}
      </div>
      <Tooltip><TooltipTrigger asChild><MetaText as="p" className="line-clamp-2 break-words leading-relaxed" style={{ minHeight: '2.5rem' }}>{pkg.description}</MetaText></TooltipTrigger>
        <TooltipContent side="top" className="max-w-[360px] leading-relaxed"><MetaText tone="inherit">{pkg.descriptionLong || pkg.description}</MetaText></TooltipContent></Tooltip>
      <div className="mt-3 flex items-center gap-3">
        <MetaText as="div" tone="weak" className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1"><span>包含 {pkg.skills.length} 个技能</span><span>已存在 {existingSkillCount} 个</span></MetaText>
      </div>
      {/* 底部行：左侧 tags + 右侧操作按钮 */}
      <div className="mt-3 flex items-end justify-between gap-2">
        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-1.5 overflow-hidden">
          {visibleSkills.map((s) => <DialogPackageSkillChip key={s.slug} skill={s} />)}
          {overflowCount > 0 && <MetaText as="span" tone="weak" className="shrink-0 text-[11px]">+{overflowCount}</MetaText>}
        </div>
        <button type="button" onClick={(e) => { e.stopPropagation(); onFavorite(); }}
          className={`flex size-7 shrink-0 items-center justify-center rounded-[4px] transition-colors ${isFavorited ? 'bg-red-50 text-[var(--text-danger)] hover:bg-red-100' : 'text-[var(--text-weak)] hover:bg-red-50 hover:text-[var(--text-danger)]'}`}
          title={isFavorited ? '取消收藏' : '添加到我的收藏'} aria-label={isFavorited ? '取消收藏' : '添加到我的收藏'}
        ><Heart className={`size-3.5 ${isFavorited ? 'fill-current' : ''}`} /></button>
      </div>
    </div>
  );
}

function getDialogPaginationItems(totalPages: number, currentPage: number): Array<number | 'ellipsis-left' | 'ellipsis-right'> {
  if (totalPages <= 7) return Array.from({ length: totalPages }, (_, i) => i + 1);
  const pages = Array.from(new Set([1, totalPages, currentPage - 1, currentPage, currentPage + 1])).filter(p => p >= 1 && p <= totalPages).sort((a, b) => a - b);
  const items: Array<number | 'ellipsis-left' | 'ellipsis-right'> = [];
  pages.forEach((page, idx) => { const prev = pages[idx - 1]; if (prev && page - prev > 1) items.push(prev === 1 ? 'ellipsis-left' : 'ellipsis-right'); items.push(page); });
  return items;
}

function AddPublicSkillDialog({ open, existingSkillIds, onConfirm, onCancel }: AddPublicSkillDialogProps) {
  const [activeDialogTab, setActiveDialogTab] = useState<AddPublicSkillDialogTab>('public-skill');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [selectedPackageIds, setSelectedPackageIds] = useState<string[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [packageSearchQuery, setPackageSearchQuery] = useState('');
  const [activeCategory, setActiveCategory] = useState('featured');
  const [activePackageCategory, setActivePackageCategory] = useState('all');
  const [currentPage, setCurrentPage] = useState(1);
  const [packageCurrentPage, setPackageCurrentPage] = useState(1);
  const [favoriteIds, setFavoriteIds] = useState<Set<string>>(() => new Set(DEFAULT_FAVORITE_PUBLIC_SKILL_IDS));
  const [favoritePackageIds, setFavoritePackageIds] = useState<Set<string>>(() => new Set(DEFAULT_FAVORITE_PUBLIC_PACKAGE_IDS));
  const existingSkillIdSet = useMemo(() => new Set(existingSkillIds), [existingSkillIds]);
  const featuredSkills = useMemo(() => [...PUBLIC_SKILLS].sort((a, b) => (b.downloads + b.stars) - (a.downloads + a.stars)).slice(0, 50), []);
  const filteredSkills = useMemo(() => {
    let list: PublicSkill[];
    if (activeCategory === 'all') list = [...PUBLIC_SKILLS];
    else if (activeCategory === 'favorites') list = PUBLIC_SKILLS.filter(s => favoriteIds.has(s.id));
    else if (activeCategory === 'featured') list = featuredSkills;
    else list = PUBLIC_SKILLS.filter(s => s.category === activeCategory);
    const q = searchQuery.trim().toLowerCase();
    if (!q) return list;
    return list.filter(s => s.name.toLowerCase().includes(q) || s.nameZh.toLowerCase().includes(q) || s.description.toLowerCase().includes(q) || s.descriptionZh.toLowerCase().includes(q) || s.tags.some(t => t.toLowerCase().includes(q)));
  }, [activeCategory, favoriteIds, featuredSkills, searchQuery]);
  const filteredPkgs = useMemo(() => {
    let list: PublicSkillPackage[];
    if (activePackageCategory === 'all') list = [...PUBLIC_SKILL_PACKAGES];
    else if (activePackageCategory === 'favorites') list = PUBLIC_SKILL_PACKAGES.filter(p => favoritePackageIds.has(p.id));
    else list = PUBLIC_SKILL_PACKAGES.filter(p => p.category === activePackageCategory);
    const q = packageSearchQuery.trim().toLowerCase();
    if (!q) return list;
    return list.filter(p => p.name.toLowerCase().includes(q) || p.description.toLowerCase().includes(q) || p.skills.some(s => s.name.toLowerCase().includes(q)));
  }, [activePackageCategory, favoritePackageIds, packageSearchQuery]);
  const totalPages = Math.max(1, Math.ceil(filteredSkills.length / PUBLIC_SKILL_DIALOG_PAGE_SIZE));
  const pagedSkills = useMemo(() => filteredSkills.slice((currentPage - 1) * PUBLIC_SKILL_DIALOG_PAGE_SIZE, currentPage * PUBLIC_SKILL_DIALOG_PAGE_SIZE), [currentPage, filteredSkills]);
  const paginationItems = useMemo(() => getDialogPaginationItems(totalPages, currentPage), [currentPage, totalPages]);
  const packageTotalPages = Math.max(1, Math.ceil(filteredPkgs.length / PUBLIC_PACKAGE_DIALOG_PAGE_SIZE));
  const pagedPackages = useMemo(() => filteredPkgs.slice((packageCurrentPage - 1) * PUBLIC_PACKAGE_DIALOG_PAGE_SIZE, packageCurrentPage * PUBLIC_PACKAGE_DIALOG_PAGE_SIZE), [filteredPkgs, packageCurrentPage]);
  const packagePaginationItems = useMemo(() => getDialogPaginationItems(packageTotalPages, packageCurrentPage), [packageCurrentPage, packageTotalPages]);

  useEffect(() => { if (open) { setActiveDialogTab('public-skill'); setSelectedIds([]); setSelectedPackageIds([]); setSearchQuery(''); setPackageSearchQuery(''); setActiveCategory('featured'); setActivePackageCategory('all'); setCurrentPage(1); setPackageCurrentPage(1); } }, [open]);
  useEffect(() => { if (currentPage > totalPages) setCurrentPage(totalPages); }, [currentPage, totalPages]);
  useEffect(() => { if (packageCurrentPage > packageTotalPages) setPackageCurrentPage(packageTotalPages); }, [packageCurrentPage, packageTotalPages]);

  const toggleSkill = (id: string) => { if (existingSkillIdSet.has(id)) return; setSelectedIds(prev => prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]); };
  const togglePackage = (id: string) => setSelectedPackageIds(prev => prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]);
  const handleCategoryChange = (id: string) => { setActiveCategory(id); setCurrentPage(1); };
  const handleSearchChange = (v: string) => { setSearchQuery(v); setCurrentPage(1); };
  const handlePackageCategoryChange = (id: string) => { setActivePackageCategory(id); setPackageCurrentPage(1); };
  const handlePackageSearchChange = (v: string) => { setPackageSearchQuery(v); setPackageCurrentPage(1); };
  const handleFavorite = (id: string) => setFavoriteIds(prev => { const n = new Set(prev); if (n.has(id)) n.delete(id); else n.add(id); return n; });
  const handlePackageFavorite = (id: string) => setFavoritePackageIds(prev => { const n = new Set(prev); if (n.has(id)) n.delete(id); else n.add(id); return n; });
  const getExistingSkillCountInPackage = (pkg: PublicSkillPackage) => pkg.skills.filter(ref => { const s = PUBLIC_SKILLS.find(x => x.slug === ref.slug); return existingSkillIdSet.has(s?.id ?? ref.slug) || existingSkillIdSet.has(ref.slug); }).length;

  const handleConfirm = () => {
    const map = new Map<string, PackageSkillItem>();
    const skipped = new Set<string>();
    selectedIds.forEach(id => { const skill = PUBLIC_SKILLS.find(s => s.id === id); if (!skill) return; if (existingSkillIdSet.has(skill.id)) { skipped.add(skill.id); return; } map.set(skill.id, { skillId: skill.id, skillName: skill.slug, skillNameZh: skill.nameZh, source: 'public', version: skill.version, addedAt: new Date() }); });
    selectedPackageIds.forEach(pkgId => { const pkg = PUBLIC_SKILL_PACKAGES.find(p => p.id === pkgId); if (!pkg) return; pkg.skills.forEach(ref => { const skill = PUBLIC_SKILLS.find(x => x.slug === ref.slug); const sid = skill?.id ?? ref.slug; if (existingSkillIdSet.has(sid)) { skipped.add(sid); return; } const existed = map.get(sid); const sp = { id: pkg.id, name: pkg.name }; if (existed) { const eps = existed.sourcePackages ?? []; if (!eps.some(x => x.id === sp.id)) map.set(sid, { ...existed, sourcePackages: [...eps, sp] }); return; } map.set(sid, { skillId: sid, skillName: skill?.slug ?? ref.slug, skillNameZh: skill?.nameZh ?? ref.name, source: 'public', sourcePackages: [sp], version: skill?.version ?? '1.0.0', addedAt: new Date() }); }); });
    const newSkills = Array.from(map.values());
    onConfirm(newSkills);
    toast.success(`已添加 ${newSkills.length} 个技能，跳过 ${skipped.size} 个已存在技能`);
    setSelectedIds([]); setSelectedPackageIds([]);
  };
  const handleCancel = () => { setSelectedIds([]); setSelectedPackageIds([]); onCancel(); };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) handleCancel(); }}>
      <DialogContent size="xl" className="flex max-h-[min(90vh,780px)] flex-col overflow-hidden">
        <DialogHeader><DialogTitle>从公共技能库添加</DialogTitle></DialogHeader>
        <Tabs value={activeDialogTab} onValueChange={(v) => setActiveDialogTab(v as AddPublicSkillDialogTab)} className="flex min-h-0 flex-1 flex-col gap-0 overflow-hidden">
          <div className="shrink-0 px-0 pb-3"><TabsList><TabsTrigger value="public-skill">公共技能</TabsTrigger><TabsTrigger value="public-package">公共技能包</TabsTrigger></TabsList></div>
          <TabsContent value="public-skill" className="flex min-h-0 flex-1 flex-col overflow-hidden">
            <DialogBody className="flex min-h-0 flex-1 flex-col overflow-y-hidden px-6 pb-4">
              <div className="flex h-full min-h-0 flex-col gap-4">
                <div className="relative shrink-0"><Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[var(--text-weak)]" /><Input placeholder="搜索技能名称或关键词..." value={searchQuery} onChange={e => handleSearchChange(e.target.value)} className="bg-white pl-9 pr-8" />{searchQuery && <button type="button" onClick={() => handleSearchChange('')} className="absolute right-2 top-1/2 flex size-5 -translate-y-1/2 items-center justify-center rounded-[4px] text-[var(--text-weak)] transition-colors hover:bg-[var(--bg-grey-hover)] hover:text-[var(--text-title)]" aria-label="清除搜索"><X className="size-3.5" /></button>}</div>
                <FilterChipGroup items={PUBLIC_SKILL_DIALOG_CATEGORIES} value={activeCategory} onChange={handleCategoryChange} className="shrink-0" />
                <div className="flex shrink-0 items-center justify-between gap-3"><BodyText as="p" tone="secondary">共 {filteredSkills.length} 个技能，已选 {selectedIds.length} 个</BodyText></div>
                <div className="min-h-0 flex-1 overflow-y-scroll pr-2 [&::-webkit-scrollbar]:w-[6px] [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-gray-300 [&::-webkit-scrollbar-track]:bg-transparent">
                  {filteredSkills.length > 0 ? (
                    <div className="grid min-w-0 grid-cols-1 gap-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
                      {pagedSkills.map((skill, idx) => { const rank = activeCategory === 'featured' ? (currentPage - 1) * PUBLIC_SKILL_DIALOG_PAGE_SIZE + idx + 1 : 0; return <AddPublicSkillCard key={skill.id} skill={skill} rank={rank} isSelected={selectedIds.includes(skill.id)} isAlreadyAdded={existingSkillIdSet.has(skill.id)} isFavorited={favoriteIds.has(skill.id)} onToggle={() => toggleSkill(skill.id)} onFavorite={() => handleFavorite(skill.id)} />; })}
                    </div>
                  ) : <Empty className="h-full min-h-[220px] border-0 py-16"><EmptyHeader><EmptyMedia /><EmptyDescription>{activeCategory === 'favorites' && favoriteIds.size === 0 ? '还没有收藏任何公共技能' : '没有找到匹配的公共技能'}</EmptyDescription></EmptyHeader></Empty>}
                </div>
                {filteredSkills.length > 0 && (
                  <div className="flex shrink-0 flex-wrap items-center justify-between gap-3 pt-3">
                    <MetaText tone="weak">第 {currentPage} / {totalPages} 页</MetaText>
                    <div className="flex flex-wrap items-center justify-end gap-1">
                      <Button type="button" variant="claw-outline" size="claw-sm" disabled={currentPage === 1} onClick={() => setCurrentPage(p => Math.max(1, p - 1))}>上一页</Button>
                      {paginationItems.map(item => typeof item === 'number' ? <Button key={item} type="button" variant={item === currentPage ? 'dialog-confirm' : 'claw-outline'} size="icon-sm" onClick={() => setCurrentPage(item)}>{item}</Button> : <span key={item} className="flex size-8 items-center justify-center text-[var(--text-weak)]">...</span>)}
                      <Button type="button" variant="claw-outline" size="claw-sm" disabled={currentPage === totalPages} onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}>下一页</Button>
                    </div>
                  </div>
                )}
              </div>
            </DialogBody>
          </TabsContent>
          <TabsContent value="public-package" className="flex min-h-0 flex-1 flex-col overflow-hidden">
            <DialogBody className="flex min-h-0 flex-1 flex-col overflow-y-hidden px-6 pb-4">
              <div className="flex h-full min-h-0 flex-col gap-4">
                <div className="relative shrink-0"><Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[var(--text-weak)]" /><Input placeholder="搜索技能包名称、描述或包含的技能..." value={packageSearchQuery} onChange={e => handlePackageSearchChange(e.target.value)} className="bg-white pl-9 pr-8" />{packageSearchQuery && <button type="button" onClick={() => handlePackageSearchChange('')} className="absolute right-2 top-1/2 flex size-5 -translate-y-1/2 items-center justify-center rounded-[4px] text-[var(--text-weak)] transition-colors hover:bg-[var(--bg-grey-hover)] hover:text-[var(--text-title)]" aria-label="清除搜索"><X className="size-3.5" /></button>}</div>
                <FilterChipGroup items={PUBLIC_PACKAGE_DIALOG_CATEGORIES} value={activePackageCategory} onChange={handlePackageCategoryChange} className="shrink-0" />
                <div className="flex shrink-0 items-center justify-between gap-3"><BodyText as="p" tone="secondary">共 {filteredPkgs.length} 个技能包，已选 {selectedPackageIds.length} 个</BodyText></div>
                <div className="min-h-0 flex-1 overflow-y-scroll pr-2 [&::-webkit-scrollbar]:w-[6px] [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-gray-300 [&::-webkit-scrollbar-track]:bg-transparent">
                  {filteredPkgs.length > 0 ? (
                    <div className="grid min-w-0 grid-cols-1 gap-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
                      {pagedPackages.map(pkg => <AddPublicPackageCard key={pkg.id} pkg={pkg} isSelected={selectedPackageIds.includes(pkg.id)} isFavorited={favoritePackageIds.has(pkg.id)} existingSkillCount={getExistingSkillCountInPackage(pkg)} onToggle={() => togglePackage(pkg.id)} onFavorite={() => handlePackageFavorite(pkg.id)} />)}
                    </div>
                  ) : <Empty className="h-full min-h-[220px] border-0 py-16"><EmptyHeader><EmptyMedia /><EmptyDescription>{activePackageCategory === 'favorites' && favoritePackageIds.size === 0 ? '还没有收藏任何公共技能包' : '没有找到匹配的公共技能包'}</EmptyDescription></EmptyHeader></Empty>}
                </div>
                {filteredPkgs.length > 0 && (
                  <div className="flex shrink-0 flex-wrap items-center justify-between gap-3 pt-3">
                    <MetaText tone="weak">第 {packageCurrentPage} / {packageTotalPages} 页</MetaText>
                    <div className="flex flex-wrap items-center justify-end gap-1">
                      <Button type="button" variant="claw-outline" size="claw-sm" disabled={packageCurrentPage === 1} onClick={() => setPackageCurrentPage(p => Math.max(1, p - 1))}>上一页</Button>
                      {packagePaginationItems.map(item => typeof item === 'number' ? <Button key={item} type="button" variant={item === packageCurrentPage ? 'dialog-confirm' : 'claw-outline'} size="icon-sm" onClick={() => setPackageCurrentPage(item)}>{item}</Button> : <span key={item} className="flex size-8 items-center justify-center text-[var(--text-weak)]">...</span>)}
                      <Button type="button" variant="claw-outline" size="claw-sm" disabled={packageCurrentPage === packageTotalPages} onClick={() => setPackageCurrentPage(p => Math.min(packageTotalPages, p + 1))}>下一页</Button>
                    </div>
                  </div>
                )}
              </div>
            </DialogBody>
          </TabsContent>
        </Tabs>
        <DialogFooter>
          <Button variant="claw-outline" onClick={handleCancel}>取消</Button>
          <Button variant="dialog-confirm" onClick={handleConfirm} disabled={selectedIds.length === 0 && selectedPackageIds.length === 0}>
            确认添加{(selectedIds.length > 0 || selectedPackageIds.length > 0) ? `（${selectedIds.length + selectedPackageIds.length} 个）` : ''}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── 企业技能库添加弹窗 ──────────────────────────────────────────────────────────

interface AddEnterpriseSkillDialogProps {
  open: boolean;
  existingSkillIds: string[];
  onConfirm: (skills: PackageSkillItem[]) => void;
  onCancel: () => void;
  /** 当前技能包的应用范围类型 */
  pkgScopeType?: 'public' | 'private';
  /** 当前技能包关联的分组 ID 列表 */
  pkgGroupIds?: string[];
}

function AddEnterpriseSkillDialog({ open, existingSkillIds, onConfirm, onCancel, pkgScopeType, pkgGroupIds }: AddEnterpriseSkillDialogProps) {
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [activeCategory, setActiveCategory] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [refreshKey, setRefreshKey] = useState<number>(0);
  /** 应用范围多选筛选：空数组=全部, ['__none__']=全不选, ['__public__']=全部用户, ['grp-x']=特定分组 */
  const [scopeFilters, setScopeFilters] = useState<string[]>([]);
  const [scopeDropdownOpen, setScopeDropdownOpen] = useState(false);
  const [scopeSearchQuery, setScopeSearchQuery] = useState('');

  // 打开时根据技能包应用范围设置默认筛选
  // 规则：【全部用户】默认必勾选
  // 如果技能包是【全部用户】的，只勾选【全部用户】，不再多勾其他
  // 如果技能包不是【全部用户】的，勾选【全部用户】+ 该包关联的分组
  useEffect(() => {
    if (open) {
      if (pkgScopeType === 'public') {
        // 全部用户的初始技能包：只勾选【全部用户】
        setScopeFilters(['__public__']);
      } else if (pkgScopeType === 'private' && pkgGroupIds && pkgGroupIds.length > 0) {
        // 非全部用户的初始技能包：勾选【全部用户】+ 关联分组
        setScopeFilters(['__public__', ...pkgGroupIds]);
      } else {
        setScopeFilters(['__public__']);
      }
      setScopeDropdownOpen(false);
      setScopeSearchQuery('');
    }
  }, [open, pkgScopeType, pkgGroupIds]);

  const toggleSkill = (skillId: string) => {
    setSelectedIds(prev =>
      prev.includes(skillId) ? prev.filter(id => id !== skillId) : [...prev, skillId]
    );
  };

  const handleConfirm = () => {
    const newSkills: PackageSkillItem[] = selectedIds.map(id => {
      const skill = MOCK_SKILLS.find(s => s.id === id)!;
      return {
        skillId: skill.id,
        skillName: skill.slug,
        skillNameZh: skill.name,
        source: 'enterprise' as const,
        version: skill.version,
        addedAt: new Date(),
      };
    });
    onConfirm(newSkills);
    setSelectedIds([]);
    setActiveCategory('all');
    setSearchQuery('');
  };

  const handleCancel = () => {
    setSelectedIds([]);
    setActiveCategory('all');
    setSearchQuery('');
    setScopeFilters([]);
    setScopeDropdownOpen(false);
    setScopeSearchQuery('');
    onCancel();
  };

  const handleRefresh = () => {
    setRefreshKey(k => k + 1);
    setSearchQuery('');
    setActiveCategory('all');
    setSelectedIds([]);
    setScopeFilters([]);
    setScopeDropdownOpen(false);
    setScopeSearchQuery('');
  };

  const filteredSkills = MOCK_SKILLS.filter(s => {
    const matchCategory = activeCategory === 'all' || s.categories.includes(activeCategory);
    const q = searchQuery.trim().toLowerCase();
    const matchSearch = q === '' || s.name.toLowerCase().includes(q) || (s.description ?? '').toLowerCase().includes(q);
    // 应用范围筛选（未选或全选时不过滤）
    let matchScope = true;
    if (scopeFilters.length > 0) {
      const allIds = ['__public__', ...MOCK_GROUPS.map(g => g.id)];
      const allSelected = allIds.every(id => scopeFilters.includes(id));
      if (!allSelected) {
        matchScope = false;
        if (scopeFilters.includes('__public__') && s.scope === 'public') {
          matchScope = true;
        }
        // 检查技能是否关联了选中的分组
        const selectedGroupIds = scopeFilters.filter(f => f !== '__public__' && f !== '__none__');
        if (selectedGroupIds.length > 0 && s.groupIds) {
          if (selectedGroupIds.some(gid => s.groupIds.includes(gid))) {
            matchScope = true;
          }
        }
      }
    }
    return matchCategory && matchSearch && matchScope;
  });

  const renderSkillCard = (skill: typeof MOCK_SKILLS[0]) => {
    const isAlreadyAdded = existingSkillIds.includes(skill.id);
    const isSelected = selectedIds.includes(skill.id);

    // 应用范围标签
    const scopeLabelsArr: string[] = (skill.scope === 'public' || !skill.groupIds || skill.groupIds.length === 0)
      ? ['全部用户']
      : skill.groupIds.map(id => MOCK_GROUPS.find(g => g.id === id)?.name || id);
    const isPublicScope = skill.scope === 'public' || !skill.groupIds || skill.groupIds.length === 0;

    const scopeExtra = (
      <div className="flex items-center gap-1 shrink-0">
        {isPublicScope ? (
          <AllUsersTag />
        ) : (
          <Tooltip delayDuration={300}>
            <TooltipTrigger asChild>
              <span className="inline-flex items-center gap-1 cursor-default">
                <StatusTag mode="fill" variant="gray" className="whitespace-nowrap">
                  {scopeLabelsArr[0]}
                </StatusTag>
                {scopeLabelsArr.length > 1 && (
                  <StatusTag mode="fill" variant="gray">
                    +{scopeLabelsArr.length - 1}
                  </StatusTag>
                )}
              </span>
            </TooltipTrigger>
            <TooltipContent side="top" className="max-w-[280px] leading-relaxed">
              <MetaText tone="inherit">{scopeLabelsArr.join('，')}</MetaText>
            </TooltipContent>
          </Tooltip>
        )}
      </div>
    );

    return (
      <SkillSelectCard
        key={skill.id}
        name={skill.name}
        version={skill.version}
        description={skill.description}
        state={isAlreadyAdded ? "disabled" : isSelected ? "selected" : "default"}
        extra={scopeExtra}
        onClick={() => toggleSkill(skill.id)}
      />
    );
  };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) handleCancel(); }}>
      <DialogContent size="xl" style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }} onOpenAutoFocus={e => e.preventDefault()}>
        <DialogHeader>
          <DialogTitle>从企业技能库添加</DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6 flex-1">
          {/* 搜索框 + 应用范围筛选 + 刷新按钮 */}
          <div className="pb-3 flex items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)] pointer-events-none" />
              <Input
                type="text"
                placeholder="搜索技能名称或描述..."
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>
            {/* 应用范围多选下拉 — InstantMultiSelect（分组模式） */}
            <InstantMultiSelect
              sections={[
                {
                  label: '全部用户',
                  options: [{ value: '__public__', label: '全部用户' }],
                },
                {
                  label: '按分组',
                  options: MOCK_GROUPS.map(g => ({ value: g.id, label: g.name })),
                },
              ]}
              value={new Set(scopeFilters)}
              onChange={(next) => setScopeFilters([...next])}
              placeholder="全部应用范围"
              selectAllLabel="全部应用范围"
              align="end"
              triggerClassName="min-w-[10rem] max-w-[16rem]"
            />
            <Tooltip delayDuration={300}>
              <TooltipTrigger asChild>
                <Button
                  variant="claw-outline"
                  size="claw-square"
                  onClick={handleRefresh}
                  aria-label="刷新"
                >
                  <RefreshCw />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="top">
                <MetaText tone="inherit">刷新</MetaText>
              </TooltipContent>
            </Tooltip>
          </div>

          {/* 分类标签 */}
          <FilterChipGroup
            items={[{ id: 'all', label: '全部' }, ...DEFAULT_CATEGORIES.map(cat => ({ id: cat.id, label: cat.name }))]}
            value={activeCategory}
            onChange={setActiveCategory}
            className="pb-3"
          />

          {/* 技能卡片列表 */}
          {filteredSkills.length > 0 ? (
            <div className="max-h-[420px] overflow-y-auto">
              <div className="grid grid-cols-2 gap-3 pr-1">
              {filteredSkills.map(skill => renderSkillCard(skill))}
              </div>
            </div>
          ) : (
            <Empty className="border-0">
              <EmptyHeader>
                <EmptyMedia />
                <EmptyTitle>暂无匹配的技能</EmptyTitle>
                <EmptyDescription>请尝试调整搜索关键词或筛选条件</EmptyDescription>
              </EmptyHeader>
            </Empty>
          )}
        </DialogBody>

        <DialogFooter>
          <Button variant="outline" onClick={handleCancel}>取消</Button>
          <Button
            variant="dialog-confirm"
            onClick={handleConfirm}
            disabled={selectedIds.length === 0}
          >
            确认添加{selectedIds.length > 0 ? `（${selectedIds.length} 个）` : ''}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── 新建技能包对话框 ──────────────────────────────────────────────────────────

const CREATE_DIALOG_ALL_GROUPS: UserGroup[] = [...MOCK_ONEID_GROUPS, ...MOCK_MANUAL_GROUPS];

interface CreatePackageDialogProps {
  open: boolean;
  existingNames: string[];
  onConfirm: (name: string, scopeType: 'public' | 'private', groupIds: string[]) => void;
  onCancel: () => void;
}

function CreatePackageDialog({ open, existingNames, onConfirm, onCancel }: CreatePackageDialogProps) {
  const [name, setName] = useState('');
  const [scopeType, setScopeType] = useState<'public' | 'private'>('public');
  const [groupIds, setGroupIds] = useState<string[]>([]);

  const trimmed = name.trim();

  const handleConfirm = () => {
    if (!trimmed) return;
    if (existingNames.includes(trimmed)) {
      toast.error('初始技能包名称不可重复');
      return;
    }
    if (scopeType === 'private' && groupIds.length === 0) {
      toast.error('请至少选择一个分组');
      return;
    }
    onConfirm(trimmed, scopeType, groupIds);
    resetForm();
  };

  const resetForm = () => {
    setName('');
    setScopeType('public');
    setGroupIds([]);
  };

  /** ScopeSelect 确认回调 */
  const handleScopeConfirm = (scope: ScopeType, ids: string[]) => {
    if (scope === 'all') {
      setScopeType('public');
      setGroupIds([]);
    } else {
      setScopeType('private');
      setGroupIds(ids);
    }
  };

  const scopeLabels = scopeType === 'public'
    ? ['全部用户']
    : groupIds.map((gid) => CREATE_DIALOG_ALL_GROUPS.find((g) => g.id === gid)?.name || gid);

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) { resetForm(); onCancel(); } }}>
      <DialogContent size="sm" className="flex flex-col">
        <DialogHeader>
          <DialogTitle>新建初始技能包</DialogTitle>
        </DialogHeader>
        <DialogBody className="px-6">
          <div className="space-y-4">
            <div className="space-y-1.5">
              <MetaMedium as="label" tone="secondary">
                技能包名称<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
              </MetaMedium>
              <Input
                placeholder="例如：全员通用技能包"
                value={name}
                onChange={e => setName(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && handleConfirm()}
                autoFocus
              />
            </div>
            {/* 应用范围 */}
            <div>
              <MetaMedium as="label" tone="secondary">应用范围</MetaMedium>
              <div className="mt-1.5">
                <ScopeSelect
                  scope={scopeType === 'public' ? 'all' : 'groups'}
                  selectedGroupIds={groupIds}
                  groups={CREATE_DIALOG_ALL_GROUPS}
                  scopeLabels={scopeLabels}
                  maxVisibleBadges={3}
                  onConfirm={handleScopeConfirm}
                />
              </div>
            </div>
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={() => { resetForm(); onCancel(); }}>取消</Button>
          <Button variant="dialog-confirm" onClick={handleConfirm} disabled={!trimmed || (scopeType === 'private' && groupIds.length === 0)}>创建</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── 发布确认对话框 ────────────────────────────────────────────────────────────

interface PublishConfirmDialogProps {
  open: boolean;
  packageName: string;
  isActive: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

function PublishConfirmDialog({ open, packageName, isActive, onConfirm, onCancel }: PublishConfirmDialogProps) {
  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onCancel(); }}>
      <DialogContent
        className="sm:max-w-md"
        style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
      >
        <DialogHeader>
          <DialogTitle>确认保存修改</DialogTitle>
        </DialogHeader>
        <DialogBody className="flex-1">
          <BodyText as="p" tone="primary">
            本次修改将<BodyMedium tone="primary">应用于新创建的 Agent</BodyMedium>，已创建的 Agent 保持原有初始配置不受影响。
          </BodyText>
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={onCancel}>取消</Button>
          <Button variant="dialog-confirm" onClick={onConfirm}>
            确认保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── 删除确认对话框 ────────────────────────────────────────────────────────────

interface DeleteConfirmDialogProps {
  open: boolean;
  packageName: string;
  onConfirm: () => void;
  onCancel: () => void;
}

function DeleteConfirmDialog({ open, packageName, onConfirm, onCancel }: DeleteConfirmDialogProps) {
  return (
    <AlertDialog open={open} onOpenChange={(o) => { if (!o) onCancel(); }}>
      <AlertDialogContent className="sm:max-w-[420px]">
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除</AlertDialogTitle>
        </AlertDialogHeader>
        <AlertDialogDescription asChild>
          <div>
            <BodyText as="p" tone="primary">
              确定要删除「<BodyMedium tone="primary">{packageName}</BodyMedium>」吗？
            </BodyText>
            <BodyText as="p" tone="danger">删除后不可恢复。</BodyText>
          </div>
        </AlertDialogDescription>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onCancel}>取消</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm}>确认删除</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

// ─── 辅助函数 ─────────────────────────────────────────────────────────────────

/** 获取技能包的应用范围显示标签数组 */
function getScopeLabels(pkg: SkillInitialPackage): string[] {
  if (pkg.scopeType === 'public' || !pkg.groupIds || pkg.groupIds.length === 0) {
    return ['全部用户'];
  }
  return pkg.groupIds.map(id => MOCK_GROUPS.find(g => g.id === id)?.name || id);
}

/** 判断技能包是否为全员范围 */
function isPublicScope(pkg: SkillInitialPackage): boolean {
  return pkg.scopeType === 'public' || !pkg.groupIds || pkg.groupIds.length === 0;
}

const SKILL_PACKAGE_ICON_BY_ID: Record<string, string> = {
  'pkg-1': '/assets/admin-skill-packages/general-skill-package.svg',
  'pkg-2': '/assets/admin-skill-packages/advanced-dev-skill-package.svg',
  'pkg-3': '/assets/admin-skill-packages/ops-team-skill-package.svg',
};

function getSkillPackageIconSrc(pkg: SkillInitialPackage): string {
  return SKILL_PACKAGE_ICON_BY_ID[pkg.id] ?? SKILL_PACKAGE_ICON_BY_ID['pkg-1'];
}

// ─── 版本比对辅助函数 ─────────────────────────────────────────────────────────

/** 获取源库中技能的最新版本 */
function getLatestVersion(skill: PackageSkillItem): string | null {
  if (skill.source === 'public') {
    const pub = PUBLIC_SKILLS.find(s => s.id === skill.skillId);
    return pub?.version ?? null;
  } else {
    const ent = MOCK_SKILLS.find(s => s.id === skill.skillId);
    return ent?.version ?? null;
  }
}

/** 公共技能 mock 更新说明 */
const PUBLIC_SKILL_CHANGELOGS: Record<string, Record<string, string>> = {
  'pub-1': { '1.0.0': '首次发布' },
  'pub-2': { '2.1.0': '1、新增 gh api 高级查询功能\n2、修复 PR 合并冲突检测问题', '2.0.0': '重构核心模块，支持多仓库管理', '1.5.0': '新增 CI/CD 流水线触发功能' },
  'pub-3': { '3.2.1': '1、优化多源聚合排序算法\n2、新增内容摘要提取\n3、修复特定编码下的解析异常', '3.1.0': '新增搜索结果缓存机制' },
  'pub-4': { '1.4.0': '1、新增 TypeScript 深度检查\n2、优化安全漏洞扫描规则\n3、支持自定义审查规则模板', '1.3.0': '新增 Python 类型提示检查' },
  'pub-7': { '1.2.0': '1、新增容器健康检查增强\n2、优化镜像层缓存策略', '1.0.0': '首次发布' },
  'pub-8': { '2.3.0': '1、新增多语言模板库（日/韩/法）\n2、优化语气分析准确率\n3、新增邮件签名管理', '2.1.0': '新增回复建议功能' },
  'pub-9': { '1.6.0': '1、新增多集群统一管理面板\n2、优化 Pod 调试日志实时流\n3、支持 HPA 自动伸缩配置', '1.5.0': '新增 Helm Chart 管理' },
  'pub-10': { '1.1.0': '1、新增图表自动生成\n2、优化主题模板引擎', '1.0.0': '首次发布' },
};

/** 获取企业技能的更新说明（changeLog） */
function getChangeLog(skill: PackageSkillItem, targetVersion: string): string {
  if (skill.source === 'enterprise') {
    const ent = MOCK_SKILLS.find(s => s.id === skill.skillId);
    const vh = ent?.versionHistory?.find(v => v.version === targetVersion);
    return vh?.changeLog || '-';
  }
  // 公共技能也返回 mock 更新说明
  const pubLogs = PUBLIC_SKILL_CHANGELOGS[skill.skillId];
  if (pubLogs && pubLogs[targetVersion]) {
    return pubLogs[targetVersion];
  }
  return '-';
}

/** 判断技能是否有可用更新 */
function hasUpdate(skill: PackageSkillItem): boolean {
  const latest = getLatestVersion(skill);
  return !!latest && latest !== skill.version;
}

// ─── 批量刷新弹窗 ──────────────────────────────────────────────────────────────

interface BatchRefreshDialogProps {
  open: boolean;
  skills: PackageSkillItem[];
  onConfirm: (selectedSkillIds: string[]) => void;
  onCancel: () => void;
}

function BatchRefreshDialog({ open, skills, onConfirm, onCancel }: BatchRefreshDialogProps) {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [currentPage, setCurrentPage] = useState(1);
  const PAGE_SIZE_OPTIONS = [20, 50, 100, 200, 500] as const;
  const [pageSize, setPageSize] = useState<number>(20);

  // 只展示有更新的技能
  const updatableSkills = skills.filter(s => hasUpdate(s));
  const totalPages = Math.max(1, Math.ceil(updatableSkills.length / pageSize));
  const pagedSkills = updatableSkills.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  // 打开弹窗时默认全选当前页
  useEffect(() => {
    if (open) {
      const firstPageSkills = updatableSkills.slice(0, pageSize);
      setSelectedIds(new Set(firstPageSkills.map(s => s.skillId)));
      setCurrentPage(1);
    }
  }, [open]);

  // 当前页全选
  const currentPageIds = pagedSkills.map(s => s.skillId);
  const allPageSelected = currentPageIds.length > 0 && currentPageIds.every(id => selectedIds.has(id));

  const toggleAll = () => {
    setSelectedIds(prev => {
      const next = new Set(prev);
      if (allPageSelected) {
        currentPageIds.forEach(id => next.delete(id));
      } else {
        currentPageIds.forEach(id => next.add(id));
      }
      return next;
    });
  };

  const toggleOne = (id: string) => {
    setSelectedIds(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const handleConfirm = () => {
    onConfirm([...selectedIds]);
    setSelectedIds(new Set());
    setCurrentPage(1);
  };

  const handleCancel = () => {
    setSelectedIds(new Set());
    setCurrentPage(1);
    onCancel();
  };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) handleCancel(); }}>
      {/*
        停服时仍允许「批量刷新技能版本」弹窗内所有功能正常使用：
        刷新技能版本属于只读类操作，不消耗管控台写权限。
        通过 data-billing-exempt 容器豁免 AdminDisabledOverlay 的全局禁用样式与点击拦截。
      */}
      <DialogContent
        size="lg"
        style={{ maxHeight: 'min(90vh, 720px)', display: 'flex', flexDirection: 'column' }}
        data-billing-exempt
      >
        <DialogHeader>
          <DialogTitle>批量刷新技能版本</DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6 flex-1">
        {updatableSkills.length === 0 ? (
          <Empty className="border-0">
            <EmptyHeader>
              <EmptyMedia />
              <EmptyTitle>所有技能均为最新版本</EmptyTitle>
              <EmptyDescription>当前没有可更新的技能</EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className="flex flex-col gap-3">
            {/* 弹窗内压缩表格：直接用 Table 自带 containerClassName 描边，禁止再套 div（参考 Table §1 / §8 + DispatchCommandDialog 规范案例） */}
            <div className="max-h-[420px] overflow-y-auto">
            <Table
              density="compact"
              autoFixedColumns={false}
              containerClassName="border border-gray-200 rounded-[4px] overflow-hidden bg-white"
            >
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[44px]">
                    <Checkbox
                      checked={allPageSelected}
                      onCheckedChange={toggleAll}
                      aria-label={allPageSelected ? '取消全选' : '全选当前页'}
                    />
                  </TableHead>
                  <TableHead className="w-[26%]">技能名称</TableHead>
                  <TableHead className="w-[8%]">类型</TableHead>
                  <TableHead className="w-[12%]">新版本</TableHead>
                  <TableHead className="w-[12%]">原版本</TableHead>
                  <TableHead className="w-[34%]">更新说明</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {pagedSkills.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6}>
                      <div className="text-center py-10">
                        <HelperText>暂无可更新的技能</HelperText>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : (
                  pagedSkills.map((skill) => {
                    const latest = getLatestVersion(skill)!;
                    const checked = selectedIds.has(skill.skillId);
                    const changeLog = getChangeLog(skill, latest);
                    return (
                      <TableRow
                        key={skill.skillId}
                        data-state={checked ? 'selected' : undefined}
                        onClick={() => toggleOne(skill.skillId)}
                        className="cursor-pointer"
                      >
                        <TableCell>
                          <Checkbox
                            checked={checked}
                            onCheckedChange={() => toggleOne(skill.skillId)}
                            onClick={(e) => e.stopPropagation()}
                          />
                        </TableCell>
                        <TableCell className="font-medium">
                          <span className="block truncate">
                            {skill.source === 'enterprise' && skill.skillNameZh ? skill.skillNameZh : skill.skillName}
                          </span>
                        </TableCell>
                        <TableCell>
                          <StatusTag mode="fill" variant={skill.source === 'public' ? 'blue' : 'gray'}>
                            {skill.source === 'public' ? '公共' : '企业'}
                          </StatusTag>
                        </TableCell>
                        <TableCell>v{latest}</TableCell>
                        <TableCell className="text-gray-500">v{skill.version}</TableCell>
                        <TableCell className="whitespace-normal">{changeLog}</TableCell>
                      </TableRow>
                    );
                  })
                )}
              </TableBody>
            </Table>
            </div>

            {/* 分页控件 — 与设计分支主流页面统一使用默认尺寸（28px）*/}
            <Pagination
              total={updatableSkills.length}
              current={currentPage}
              pageSize={pageSize}
              showSizeChanger
              pageSizeOptions={PAGE_SIZE_OPTIONS}
              showTotal={(total) => `共 ${total} 条`}
              size="default"
              className="w-full justify-between"
              onChange={(page, size) => {
                if (size !== pageSize) {
                  setPageSize(size);
                  setCurrentPage(1);
                } else {
                  setCurrentPage(page);
                }
              }}
            />
          </div>
        )}
        </DialogBody>

        <DialogFooter className="gap-2 sm:gap-2">
          <Button variant="outline" onClick={handleCancel}>取消</Button>
          <Button
            variant="dialog-confirm"
            onClick={handleConfirm}
            disabled={selectedIds.size === 0}
          >
            确认刷新{selectedIds.size > 0 ? `（${selectedIds.size} 个）` : ''}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── 技能包详情页 ─────────────────────────────────────────────────────────────

interface PackageDetailViewProps {
  pkg: SkillInitialPackage;
  onBack: () => void;
  onPublish: (pkgId: string) => void;
  onRemoveSkill: (pkgId: string, skillId: string) => void;
}

function PackageDetailView({ pkg, onBack, onPublish, onRemoveSkill }: PackageDetailViewProps) {
  const [showPublishConfirm, setShowPublishConfirm] = useState(false);
  const [isDirty, setIsDirty] = useState(false);
  const [localSkills, setLocalSkills] = useState(pkg.skills);
  const [showAddEnterpriseDialog, setShowAddEnterpriseDialog] = useState(false);
  const [showAddPublicDialog, setShowAddPublicDialog] = useState(false);
  const [showBatchRefreshDialog, setShowBatchRefreshDialog] = useState(false);

  // 当 pkg 变化时同步本地技能列表（例如切换包）
  const handleRemoveLocal = (skillId: string) => {
    setLocalSkills(prev => prev.filter(s => s.skillId !== skillId));
    setIsDirty(true);
  };

  const doSave = () => {
    // 找出被删除的技能并逐一调用 onRemoveSkill
    pkg.skills.forEach(s => {
      if (!localSkills.find(ls => ls.skillId === s.skillId)) {
        onRemoveSkill(pkg.id, s.skillId);
      }
    });
    // 清除 originalVersion 标记（保存后正式生效）
    setLocalSkills(prev => prev.map(s => ({ ...s, originalVersion: undefined })));
    setIsDirty(false);
    setShowPublishConfirm(false);
    toast.success('保存成功');
  };

  const handleSave = () => {
    if (pkg.isActive) {
      // 已生效的技能包：弹出二次确认
      setShowPublishConfirm(true);
    } else {
      // 未生效的技能包：直接保存
      doSave();
    }
  };

  const handleDiscard = () => {
    setLocalSkills(pkg.skills);
    setIsDirty(false);
  };

  const handleAddPublicSkills = (skills: PackageSkillItem[]) => {
    setLocalSkills(prev => [...prev, ...skills]);
    setIsDirty(true);
    setShowAddPublicDialog(false);
  };

  const handleAddEnterpriseSkills = (skills: PackageSkillItem[]) => {
    setLocalSkills(prev => [...prev, ...skills]);
    setIsDirty(true);
    setShowAddEnterpriseDialog(false);
  };

  /** 单个技能刷新到最新版本 */
  const handleRefreshSingle = (skillId: string) => {
    const skill = localSkills.find(s => s.skillId === skillId);
    if (!skill) return;
    const latest = getLatestVersion(skill);
    if (!latest || latest === skill.version) {
      toast.info('当前已是最新版本');
      return;
    }
    setLocalSkills(prev => prev.map(s =>
      s.skillId === skillId
        ? { ...s, originalVersion: s.originalVersion || s.version, version: latest }
        : s
    ));
    setIsDirty(true);
    toast.success(`已刷新至 v${latest}`);
  };

  /** 批量刷新确认 */
  const handleBatchRefreshConfirm = (selectedSkillIds: string[]) => {
    setLocalSkills(prev => prev.map(s => {
      if (!selectedSkillIds.includes(s.skillId)) return s;
      const latest = getLatestVersion(s);
      if (!latest || latest === s.version) return s;
      return { ...s, originalVersion: s.originalVersion || s.version, version: latest };
    }));
    setIsDirty(true);
    setShowBatchRefreshDialog(false);
    toast.success(`已刷新 ${selectedSkillIds.length} 个技能`);
  };

  const scopeLabels = getScopeLabels(pkg);

  // 统计有更新的技能数量
  const updatableCount = localSkills.filter(s => hasUpdate(s)).length;

  return (
    <div className="space-y-4">
      {/* 顶部导航 */}
      <div className="flex items-center justify-between">
        {/*
          停服时仍允许「返回初始技能包列表」可点击：纯导航操作，不消耗管控台写权限。
          通过 data-billing-exempt 豁免 AdminDisabledOverlay 的全局禁用样式与点击拦截。
        */}
        <button
          onClick={onBack}
          className="flex items-center gap-1.5 transition-colors text-[#525252] hover:text-[#0A0A0A]"
          data-billing-exempt
        >
          <ArrowLeft className="w-4 h-4" />
          <BodyText tone="inherit">返回初始技能包列表</BodyText>
        </button>
      </div>

      {/* 技能包信息 + 操作栏 — 同一行，左右两端对齐，使用 SurfaceCard 包裹 */}
      <SurfaceCard className="flex items-center justify-between gap-4 px-5 py-4">
        {/* 左侧：图标 + 标题 + 标签 */}
        <div className="flex items-center gap-3 min-w-0">
          <img
            src={getSkillPackageIconSrc(pkg)}
            alt=""
            aria-hidden="true"
            className="h-10 w-10 shrink-0"
          />
          <div className="flex flex-col gap-2 min-w-0">
            <CardTitle as="h2" className="leading-none truncate">{pkg.name}</CardTitle>
            <div className="flex items-center gap-2">
              {pkg.isActive && (
                <StatusTag mode="fill" variant="green">生效中</StatusTag>
              )}
              {isPublicScope(pkg) ? (
                <AllUsersTag />
              ) : (
                <StatusTag mode="fill" variant="gray">{scopeLabels.join('、')}</StatusTag>
              )}
            </div>
          </div>
        </div>

        {/* 右侧：操作按钮组 */}
        <div className="flex items-center gap-2 shrink-0">
          <Button variant="claw-outline" size="claw-sm" onClick={() => setShowAddPublicDialog(true)}>
            <Plus className="w-3.5 h-3.5" />
            从公共技能库添加
          </Button>
          <Button variant="claw-outline" size="claw-sm" onClick={() => setShowAddEnterpriseDialog(true)}>
            <Plus className="w-3.5 h-3.5" />
            从企业技能库添加
          </Button>
          <Tooltip delayDuration={300}>
            <TooltipTrigger asChild>
              {/*
                停服时仍允许「批量刷新」可点击：仅刷新技能版本信息（只读类操作，不消耗管控台写权限）。
                通过 data-billing-exempt 豁免 AdminDisabledOverlay 的全局禁用样式与点击拦截。
              */}
              <Button
                variant="claw-outline"
                size="claw-sm"
                onClick={() => setShowBatchRefreshDialog(true)}
                data-billing-exempt
              >
                <RefreshCw className="w-3.5 h-3.5" />
                批量刷新{updatableCount > 0 ? `（${updatableCount}）` : ''}
              </Button>
            </TooltipTrigger>
            <TooltipContent side="top">
              检查并批量刷新技能到最新版本
            </TooltipContent>
          </Tooltip>
          {isDirty && (
            <>
              <Button
                variant="claw-outline"
                size="claw-sm"
                onClick={handleDiscard}
              >
                取消
              </Button>
              <Button
                variant="claw-primary"
                size="claw-sm"
                onClick={handleSave}
              >
                保存
              </Button>
            </>
          )}
        </div>
      </SurfaceCard>

      {/* 技能列表 */}
      <div className="bg-white rounded-xl border border-[#EAEEF4] overflow-hidden">
        {localSkills.length > 0 ? (
          <div className="max-h-[420px] overflow-y-auto">
            <Table variant="white">
              <TableHeader>
                <TableRow>
                  <TableHead>技能名称</TableHead>
                  <TableHead className="w-[120px]">来源</TableHead>
                  <TableHead className="w-[140px]">版本</TableHead>
                  <TableHead className="w-[160px]">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {localSkills.map(skill => {
                const canUpdate = hasUpdate(skill);
                const wasRefreshed = !!skill.originalVersion;
                const skillName = skill.source === 'enterprise' && skill.skillNameZh
                  ? skill.skillNameZh
                  : skill.skillName;
                return (
                  <TableRow key={skill.skillId}>
                    {/* 技能名称 */}
                    <TableCell>
                      <div className="flex items-center gap-3 min-w-0">
                        <div className="w-8 h-8 rounded-xl bg-gray-100 flex items-center justify-center shrink-0">
                          <Package className="w-4 h-4 text-[var(--text-muted)]" />
                        </div>
                        <BodyText tone="primary" className="truncate">
                          {skillName}
                        </BodyText>
                      </div>
                    </TableCell>

                    {/* 来源 */}
                    <TableCell className="w-[120px]">
                      <StatusTag mode="fill" variant="gray">
                        {skill.source === 'public' ? '公共' : '企业'}
                      </StatusTag>
                    </TableCell>

                    {/* 版本 */}
                    <TableCell className="w-[140px]">
                      {wasRefreshed ? (
                        <BodyText as="span">
                          <BodyMedium className="text-[var(--text-success)]">v{skill.version}</BodyMedium>
                          <MetaText tone="weak" className="ml-1">(原v{skill.originalVersion})</MetaText>
                        </BodyText>
                      ) : (
                        <BodyText tone="primary">v{skill.version}</BodyText>
                      )}
                    </TableCell>

                    {/* 操作：刷新 + 移除 */}
                    <TableActionCell className="w-[160px]">
                      {canUpdate ? (
                        <Tooltip delayDuration={300}>
                          <TooltipTrigger asChild>
                            <span className="inline-flex">
                              {/*
                                停服时仍允许单条「刷新」可点击：仅刷新技能版本信息（只读类操作）。
                                通过 data-billing-exempt 豁免 AdminDisabledOverlay 的全局禁用样式与点击拦截。
                              */}
                              <Button
                                variant="link"
                                size="sm"
                                onClick={() => handleRefreshSingle(skill.skillId)}
                                data-billing-exempt
                              >
                                刷新
                              </Button>
                            </span>
                          </TooltipTrigger>
                          <TooltipContent side="top">
                            {`有新版本 v${getLatestVersion(skill)}，点击刷新`}
                          </TooltipContent>
                        </Tooltip>
                      ) : (
                        <MetaText tone="weak" className="select-none">已是最新</MetaText>
                      )}
                      <Button
                        variant="link"
                        size="sm"
                        onClick={() => handleRemoveLocal(skill.skillId)}
                      >
                        移除
                      </Button>
                    </TableActionCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
          </div>
        ) : (
          <Empty className="border-0">
            <EmptyHeader>
              <EmptyMedia />
              <EmptyTitle>该技能包还没有技能</EmptyTitle>
              <EmptyDescription>可从公共技能库或企业技能库添加</EmptyDescription>
            </EmptyHeader>
          </Empty>
        )}

        {/* 表格底部：计数说明（共 N 个技能） */}
        {localSkills.length > 0 && (
          <div className="px-4 py-2.5 border-t border-[#EAEEF4]">
            <MetaText tone="weak">共 {localSkills.length} 个技能</MetaText>
          </div>
        )}
      </div>

      {/* 公共技能库添加弹窗 */}
      <AddPublicSkillDialog
        open={showAddPublicDialog}
        existingSkillIds={localSkills.map(s => s.skillId)}
        onConfirm={handleAddPublicSkills}
        onCancel={() => setShowAddPublicDialog(false)}
      />

      {/* 企业技能库添加弹窗 */}
      <AddEnterpriseSkillDialog
        open={showAddEnterpriseDialog}
        existingSkillIds={localSkills.map(s => s.skillId)}
        onConfirm={handleAddEnterpriseSkills}
        onCancel={() => setShowAddEnterpriseDialog(false)}
        pkgScopeType={pkg.scopeType}
        pkgGroupIds={pkg.groupIds}
      />

      {/* 批量刷新弹窗 */}
      <BatchRefreshDialog
        open={showBatchRefreshDialog}
        skills={localSkills}
        onConfirm={handleBatchRefreshConfirm}
        onCancel={() => setShowBatchRefreshDialog(false)}
      />

      {/* 保存确认弹窗（仅已生效技能包触发） */}
      <PublishConfirmDialog
        open={showPublishConfirm}
        packageName={pkg.name}
        isActive={pkg.isActive}
        onConfirm={doSave}
        onCancel={() => setShowPublishConfirm(false)}
      />
    </div>
  );
}


// ─── 主组件 ───────────────────────────────────────────────────────────────────

interface SkillInitialPackageTabProps {
  onPackagesChange?: (packages: Array<{ id: string; name: string; isActive: boolean }>) => void;
}

export default function SkillInitialPackageTab({ onPackagesChange }: SkillInitialPackageTabProps) {
  const [packages, setPackages] = useState<SkillInitialPackage[]>(INITIAL_SKILL_PACKAGES_DEFAULT);
  const [selectedPackageId, setSelectedPackageId] = useState<string | null>(null);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  // 筛选状态：多选 Set，空=全部, 含'public'=全部用户, 含'group-xxx'=特定分组
  const allScopeKeys = useMemo(() => ['public', ...MOCK_GROUPS.map(g => g.id)], []);
  const [selectedScopes, setSelectedScopes] = useState<Set<string>>(new Set());
  const [scopeDropdownOpen, setScopeDropdownOpen] = useState(false);
  const [scopeSearchQuery, setScopeSearchQuery] = useState('');
  const scopeDropdownRef = useRef<HTMLDivElement>(null);

  // 点击外部关闭应用范围下拉
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (scopeDropdownRef.current && !scopeDropdownRef.current.contains(e.target as Node)) {
        setScopeDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // 找到当前已生效的「全部用户」的技能包
  const activeGlobalPackage = packages.find(p => p.isActive && isPublicScope(p));

  // 通知父组件 packages 变化
  const updatePackages = (newPackages: SkillInitialPackage[]) => {
    setPackages(newPackages);
    onPackagesChange?.(newPackages.map(p => ({ id: p.id, name: p.name, isActive: p.isActive })));
  };

  // 新建技能包
  const handleCreate = (name: string, scopeType: 'public' | 'private', groupIds: string[]) => {
    const scopeDisplay = scopeType === 'public'
      ? '全部用户'
      : groupIds.map(id => MOCK_GROUPS.find(g => g.id === id)?.name || id).join('、');
    const newPkg: SkillInitialPackage = {
      id: `pkg-${Date.now()}`,
      name,
      scope: scopeDisplay,
      scopeType,
      groupIds,
      isActive: false,
      hasDraft: false,
      skills: [],
      createdAt: new Date(),
      updatedAt: new Date(),
    };
    updatePackages([...packages, newPkg]);
    setShowCreateDialog(false);
  };

  // 切换生效开关
  const handleToggleActive = (pkgId: string, value: boolean) => {
    const pkg = packages.find(p => p.id === pkgId);
    if (!pkg) return;

    if (!value) {
      // 关闭生效：直接关闭
      updatePackages(packages.map(p => p.id === pkgId ? { ...p, isActive: false } : p));
      return;
    }

    // 打开生效
    if (isPublicScope(pkg)) {
      // 全员范围：需要检查是否已有其他全员生效的
      if (activeGlobalPackage && activeGlobalPackage.id !== pkgId) {
        // 已有其他全员生效的技能包，提示错误，阻止启用
        toast.error('已有其他应用范围为「全部用户」的技能包处于启用状态，请先禁用');
        return;
      }
      // 没有全员生效的，直接启用
      updatePackages(packages.map(p => p.id === pkgId ? { ...p, isActive: true } : p));
    } else {
      // 按分组范围：直接启用（可同时启用任意多个非全员的）
      updatePackages(packages.map(p => p.id === pkgId ? { ...p, isActive: true } : p));
    }
  };

  // 发布修改
  const handlePublish = (pkgId: string) => {
    updatePackages(packages.map(p => p.id === pkgId ? { ...p, hasDraft: false, updatedAt: new Date() } : p));
  };

  // 删除技能包
  const handleDelete = (pkgId: string) => {
    updatePackages(packages.filter(p => p.id !== pkgId));
    setDeleteTarget(null);
  };

  // 从技能包中移除技能
  const handleRemoveSkill = (pkgId: string, skillId: string) => {
    updatePackages(packages.map(p =>
      p.id === pkgId
        ? { ...p, skills: p.skills.filter(s => s.skillId !== skillId), hasDraft: true, updatedAt: new Date() }
        : p
    ));
  };

  // 修改技能包应用范围
  // 默认不改变生效状态，但切换为"全部用户"时默认设为失效
  const handleScopeChange = (pkgId: string, scope: SkillScope, groupIds: string[]) => {
    updatePackages(packages.map(p => {
      if (p.id !== pkgId) return p;
      const scopeType = scope === 'public' ? 'public' : 'private';
      const scopeDisplay = scopeType === 'public'
        ? '全部用户'
        : groupIds.map(id => MOCK_GROUPS.find(g => g.id === id)?.name || id).join('、');
      // 切换为全部用户时，默认设为失效
      const isActive = scopeType === 'public' ? false : p.isActive;
      return { ...p, scopeType, groupIds, scope: scopeDisplay, isActive, updatedAt: new Date() };
    }));
    toast.success('应用范围修改成功');
  };

  // 筛选逻辑 + 排序：置顶生效且全员的技能包，其余按新增时间倒序
  const filteredPackages = packages
    .filter(pkg => {
      if (selectedScopes.size === 0) return true;
      const hasPublic = selectedScopes.has('public');
      const groupScopes = [...selectedScopes].filter(s => s !== 'public');
      const matchPublic = hasPublic && isPublicScope(pkg);
      const matchGroup = groupScopes.length > 0 && pkg.scopeType === 'private' && pkg.groupIds.some(gid => selectedScopes.has(gid));
      return !!(matchPublic || matchGroup);
    })
    .sort((a, b) => {
      const aPinned = a.isActive && isPublicScope(a);
      const bPinned = b.isActive && isPublicScope(b);
      if (aPinned && !bPinned) return -1;
      if (!aPinned && bPinned) return 1;
      return b.createdAt.getTime() - a.createdAt.getTime();
    });

  // 如果选中了技能包，显示详情页
  const selectedPackage = packages.find(p => p.id === selectedPackageId);
  if (selectedPackage) {
    return (
      <PackageDetailView
        pkg={selectedPackage}
        onBack={() => setSelectedPackageId(null)}
        onPublish={handlePublish}
        onRemoveSkill={handleRemoveSkill}
      />
    );
  }

  const deleteTargetPkg = packages.find(p => p.id === deleteTarget);

  return (
    <div className="space-y-4">
      {/* 顶部操作栏 */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <PanelTitle as="h3" className="shrink-0">初始技能包列表</PanelTitle>
          <Tooltip delayDuration={200}>
            <TooltipTrigger asChild>
              <button
                type="button"
                aria-label="服务说明"
                className="inline-flex items-center justify-center text-[var(--text-weak)] hover:text-[var(--text-muted)] transition-colors"
              >
                <Info className="w-4 h-4" />
              </button>
            </TooltipTrigger>
            <TooltipContent side="right" className="max-w-[320px] leading-relaxed">
              <MetaText tone="inherit">由腾讯云存储 Agent Storage 提供服务，ClawPro 用户独享初始技能包和企业技能库各 50GB 免费空间</MetaText>
            </TooltipContent>
          </Tooltip>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {/*
            停服时仍允许「选择应用范围」下拉筛选正常可用：纯导航/筛选类操作，不消耗管控台写权限。
            包装层（含触发按钮 + 内联下拉浮层）整体豁免停服禁用。
          */}
          <div className="relative" ref={scopeDropdownRef} data-billing-exempt>
            <Tooltip delayDuration={1000} open={scopeDropdownOpen ? false : undefined}>
              <TooltipTrigger asChild>
                <button
                  type="button"
                  onClick={() => setScopeDropdownOpen(prev => !prev)}
                  className="flex items-center justify-between gap-1 min-w-[10rem] max-w-[20rem] h-9 px-3 border border-gray-200 rounded-xl bg-white hover:bg-gray-50 transition-colors"
                >
                  <BodyText tone="secondary" className="truncate text-left">
                    {selectedScopes.size === 0
                      ? '选择应用范围'
                      : selectedScopes.size === allScopeKeys.length && allScopeKeys.every(k => selectedScopes.has(k))
                        ? '全部应用范围'
                        : [...selectedScopes].map(s => s === 'public' ? '全部用户' : MOCK_GROUPS.find(g => g.id === s)?.name || s).join('、')}
                  </BodyText>
                  <ChevronDown className={`w-4 h-4 text-[var(--text-weak)] flex-shrink-0 transition-transform ${scopeDropdownOpen ? 'rotate-180' : ''}`} />
                </button>
              </TooltipTrigger>
              <TooltipContent side="bottom" className="max-w-[280px]">
                <MetaText as="p" tone="inherit" className="break-words">
                  {selectedScopes.size === 0
                    ? '选择应用范围'
                    : selectedScopes.size === allScopeKeys.length && allScopeKeys.every(k => selectedScopes.has(k))
                      ? '全部应用范围'
                      : [...selectedScopes].map(s => s === 'public' ? '全部用户' : MOCK_GROUPS.find(g => g.id === s)?.name || s).join('、')}
                </MetaText>
              </TooltipContent>
            </Tooltip>
            {scopeDropdownOpen && (() => {
              const filteredGroups = MOCK_GROUPS.filter(g => g.name.toLowerCase().includes(scopeSearchQuery.toLowerCase()));
              const showPublic = !scopeSearchQuery || '全部用户'.includes(scopeSearchQuery);
              const showGroupSection = !scopeSearchQuery || '按分组'.includes(scopeSearchQuery) || filteredGroups.length > 0;
              const isAllSelected = allScopeKeys.length > 0 && allScopeKeys.every(k => selectedScopes.has(k));

              const toggleScope = (key: string) => {
                setSelectedScopes(prev => {
                  const next = new Set(prev);
                  if (next.has(key)) next.delete(key); else next.add(key);
                  return next;
                });
              };

              return (
                <div className="absolute right-0 top-full mt-1 w-56 bg-white rounded-[4px] shadow-[var(--shadow-popover)] z-50 pt-2 px-2 pb-0">
                  {/* 搜索框 */}
                  <div className="mb-1">
                    <div className="relative">
                      <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400 pointer-events-none" />
                      <Input
                        type="text"
                        placeholder="搜索..."
                        value={scopeSearchQuery}
                        onChange={(e) => setScopeSearchQuery(e.target.value)}
                        className="h-8 pl-8 pr-2 text-sm"
                        onClick={(e) => e.stopPropagation()}
                      />
                    </div>
                  </div>
                  {/* 全部应用范围 — 全选/全不选切换 */}
                  {(!scopeSearchQuery || '全部应用范围'.includes(scopeSearchQuery)) && (
                    <button
                      type="button"
                      onClick={() => {
                        if (isAllSelected) {
                          setSelectedScopes(new Set());
                        } else {
                          setSelectedScopes(new Set(allScopeKeys));
                        }
                        setScopeSearchQuery('');
                      }}
                      className={`flex items-center gap-2 w-full h-8 px-3 rounded-[6px] transition-colors ${isAllSelected ? 'bg-[var(--bg-brand-selected)]' : 'hover:bg-[#FAFAFA]'}`}
                    >
                      <Checkbox
                        checked={isAllSelected}
                        className="pointer-events-none"
                      />
                      <BodyText tone="secondary" className="truncate text-left">全部应用范围</BodyText>
                    </button>
                  )}
                  {/* 全部用户 区域 */}
                  {showPublic && (
                    <>
                      <div className="px-3 pt-2 pb-1 select-none">
                        <MetaMedium tone="weak">全部用户</MetaMedium>
                      </div>
                      <button
                        type="button"
                        onClick={() => toggleScope('public')}
                        className={`flex items-center gap-2 w-full h-8 px-3 rounded-[6px] transition-colors ${selectedScopes.has('public') ? 'bg-[var(--bg-brand-selected)]' : 'hover:bg-[#FAFAFA]'}`}
                      >
                        <Checkbox
                          checked={selectedScopes.has('public')}
                          className="pointer-events-none"
                        />
                        <BodyText tone="secondary" className="truncate text-left">全部用户</BodyText>
                      </button>
                    </>
                  )}
                  {/* 按分组 区域 */}
                  {showGroupSection && (
                    <>
                      <div className="px-3 pt-2.5 pb-1 select-none">
                        <MetaMedium tone="weak">按分组</MetaMedium>
                      </div>
                      <div className="max-h-44 overflow-y-auto space-y-0.5">
                        {filteredGroups.map(group => {
                          const checked = selectedScopes.has(group.id);
                          return (
                            <button
                              key={group.id}
                              type="button"
                              onClick={() => toggleScope(group.id)}
                              className={`flex items-center gap-2 w-full h-8 px-3 rounded-[6px] transition-colors ${checked ? 'bg-[var(--bg-brand-selected)]' : 'hover:bg-[#FAFAFA]'}`}
                            >
                              <Checkbox
                                checked={checked}
                                className="pointer-events-none"
                              />
                              <BodyText tone="secondary" className="truncate text-left" title={group.name}>{group.name}</BodyText>
                            </button>
                          );
                        })}
                        {filteredGroups.length === 0 && !showPublic && scopeSearchQuery && (
                          <MetaText as="p" tone="weak" className="py-2 text-center">没有匹配的结果</MetaText>
                        )}
                      </div>
                    </>
                  )}
                  {/* 底部：已选数量 + 清除筛选 */}
                  {selectedScopes.size > 0 && (
                    <div className="border-t border-[#EAEEF4] mt-1 px-1 h-9 flex items-center justify-between">
                      <MetaText>已选 {selectedScopes.size} 个应用范围</MetaText>
                      <Button
                        variant="link"
                        className="text-xs"
                        onClick={() => {
                          setSelectedScopes(new Set());
                          setScopeSearchQuery('');
                        }}
                      >
                        清除
                      </Button>
                    </div>
                  )}
                </div>
              );
            })()}
          </div>
          <Button size="sm" onClick={() => setShowCreateDialog(true)} className="gap-1.5">
            <Plus className="w-4 h-4" />
            新建
          </Button>
        </div>
      </div>

      {/* 技能包列表 */}
      {filteredPackages.length > 0 ? (
        <div className="bg-white rounded-xl border border-[#EAEEF4] overflow-hidden">
          <div className="max-h-[420px] overflow-y-auto">
          <Table variant="white">
            <TableHeader>
              <TableRow>
                <TableHead style={{ width: 360, minWidth: 280 }}>技能包名称</TableHead>
                <TableHead className="w-[120px]">技能数</TableHead>
                <TableHead className="w-[200px]">应用范围</TableHead>
                <TableHead className="w-[140px]">设为生效</TableHead>
                <TableHead className="w-[80px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredPackages.map(pkg => {
                const isPub = isPublicScope(pkg);
                const isPinned = pkg.isActive && isPub;
                const packageIconSrc = getSkillPackageIconSrc(pkg);
                return (
                  <TableRow
                    key={pkg.id}
                    className="cursor-pointer hover:bg-[#FAFAFA] group"
                    onClick={() => setSelectedPackageId(pkg.id)}
                  >
                    {/* 名称列 */}
                    <TableCell style={{ width: 360, minWidth: 280 }}>
                      <div className="flex items-center gap-3 min-w-0">
                        <img src={packageIconSrc} alt="" aria-hidden="true" className="h-9 w-9 shrink-0" />
                        <div className="flex items-center gap-1.5 min-w-0">
                          {isPinned && (
                            <Tooltip delayDuration={300}>
                              <TooltipTrigger asChild>
                                <span className="text-[var(--text-title)] shrink-0">
                                  <Pin className="w-3.5 h-3.5" />
                                </span>
                              </TooltipTrigger>
                              <TooltipContent side="top" className="max-w-[240px] text-center">
                                <MetaText tone="inherit">默认置顶应用范围为全部用户且生效中的初始技能包。</MetaText>
                              </TooltipContent>
                            </Tooltip>
                          )}
                          <BodyMedium className="group-hover:text-[var(--text-brand)] transition-colors truncate">
                            {pkg.name}
                          </BodyMedium>
                        </div>
                      </div>
                    </TableCell>

                    {/* 技能数列 */}
                    <TableCell className="w-[120px]">
                      <BodyText className="tabular-nums">{pkg.skills.length} 个技能</BodyText>
                    </TableCell>

                    {/* 应用范围列 — 复用 ScopeSelect */}
                    <TableCell className="w-[200px]">
                      <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                        <ScopeSelect
                          scope={pkg.scopeType === 'public' ? 'all' : 'groups'}
                          selectedGroupIds={pkg.groupIds || []}
                          groups={CREATE_DIALOG_ALL_GROUPS}
                          scopeLabels={getScopeLabels(pkg)}
                          onConfirm={(scope, groupIds) =>
                            handleScopeChange(
                              pkg.id,
                              scope === 'all' ? 'public' : 'private',
                              groupIds,
                            )
                          }
                        />
                      </div>
                    </TableCell>

                    {/* 生效开关列 */}
                    <TableCell className="w-[140px]" onClick={(e) => e.stopPropagation()}>
                      <Switch
                        checked={pkg.isActive}
                        onCheckedChange={(v) => handleToggleActive(pkg.id, v)}
                      />
                    </TableCell>

                    {/* 操作列 */}
                    <TableActionCell className="w-[80px]" onClick={(e) => e.stopPropagation()}>
                      {pkg.isActive ? (
                        <Tooltip delayDuration={1000}>
                          <TooltipTrigger asChild>
                            <Button variant="link" disabled>
                              删除
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="max-w-[200px] text-center">
                            <MetaText tone="inherit">生效中的技能包不可删除</MetaText>
                          </TooltipContent>
                        </Tooltip>
                      ) : (
                        <Button variant="link" onClick={() => setDeleteTarget(pkg.id)}>
                          删除
                        </Button>
                      )}
                    </TableActionCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
          </div>
        </div>
      ) : (
        <div className="bg-white rounded-xl border border-[#EAEEF4]">
          <Empty className="border-0 py-12">
            <EmptyHeader>
              <EmptyMedia />
              {selectedScopes.size > 0 ? (
                <EmptyDescription>没有匹配的初始技能包</EmptyDescription>
              ) : (
                <>
                  <EmptyTitle>还没有初始技能包</EmptyTitle>
                  <EmptyDescription>点击「新建」创建第一个初始技能包</EmptyDescription>
                </>
              )}
            </EmptyHeader>
            {selectedScopes.size === 0 && (
              <EmptyContent>
                <Button onClick={() => setShowCreateDialog(true)}>
                  <Plus className="w-4 h-4" />
                  新建初始技能包
                </Button>
              </EmptyContent>
            )}
          </Empty>
        </div>
      )}

      {/* 新建对话框 */}
      <CreatePackageDialog
        open={showCreateDialog}
        existingNames={packages.map(p => p.name)}
        onConfirm={handleCreate}
        onCancel={() => setShowCreateDialog(false)}
      />

      {/* 删除确认 */}
      {deleteTargetPkg && (
        <DeleteConfirmDialog
          open={!!deleteTarget}
          packageName={deleteTargetPkg.name}
          onConfirm={() => handleDelete(deleteTarget!)}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  );
}
