'use client';
import { useState, useEffect, useMemo } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
  DialogFooter,
} from '@/components/ui/dialog';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  InstantMultiSelect,
} from '@/components/ui/select';
import { Pagination } from '@/components/ui/pagination';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { CircleAlert, Search } from 'lucide-react';
import {
  BodyText,
  BodyMedium,
  HelperText,
  MetaText,
} from '@/components/ui/Typography';

import type { Group } from './types';

/** 卸载状态筛选选项（单选） */
type UninstallFilterOption = 'all' | 'not_deleted' | 'delete_failed';

const UNINSTALL_FILTER_OPTIONS: { value: string; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'not_deleted', label: '未卸载' },
  { value: 'delete_failed', label: '卸载失败' },
];

const PAGE_SIZE_OPTIONS = [20, 50, 100, 200, 500] as const;

interface BatchDeleteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  skillName: string;
  skillVersion: string;
  /** 已下发成功过的实例列表（从下发记录中聚合） */
  distributedInstances: Array<{
    id: string;
    name: string;
    createdBy: string;
    groupName?: string;
    distributedVersion?: string;
    distributedTime?: string;
    /** 是否曾经卸载失败 */
    deleteStatus?: 'not_deleted' | 'delete_failed';
    deleteFailReason?: string;
  }>;
  groups: Group[];
  onDeleteStart: (selectedInstanceIds: string[], selectedInstancesData: any[]) => void;
  /** 是否显示分组筛选，默认 true */
  showScopeFilter?: boolean;
  /** 隐藏实例中的创建人、分组信息 */
  hideCreatorAndGroup?: boolean;
  /** 资源名称标签，如"插件""企业规范"等 */
  resourceLabel?: string;
  /** 顶部警告文案，如"卸载后将删除该技能的所有本地数据..." */
  warningText?: string;
  /** 空数据时文案 */
  emptyText?: string;
}

export default function BatchDeleteDialog({
  open,
  onOpenChange,
  skillName,
  skillVersion,
  distributedInstances,
  groups,
  onDeleteStart,
  showScopeFilter = true,
  hideCreatorAndGroup = false,
  resourceLabel = '技能',
  warningText = '通过下发按钮安装的技能可支持移出（包括用户下发和管理端下发）。卸载成功后，该技能在对应实例上恢复为"未下发"状态。',
  emptyText = '暂无已下发的实例',
}: BatchDeleteDialogProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedInstances, setSelectedInstances] = useState<string[]>([]);
  /** 是否处于"选择全部"模式（跨页全选） */
  const [selectAllMode, setSelectAllMode] = useState(false);
  /** 状态单选筛选 */
  const [statusFilter, setStatusFilter] = useState<UninstallFilterOption>('all');
  /** 分组筛选：空 Set=全部 */
  const [scopeFilters, setScopeFilters] = useState<Set<string>>(new Set());
  const [confirmDialogOpen, setConfirmDialogOpen] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState<number>(20);

  // 打开时重置状态
  useEffect(() => {
    if (open) {
      setSearchQuery('');
      setSelectedInstances([]);
      setSelectAllMode(false);
      setStatusFilter('all');
      setScopeFilters(new Set());
      setConfirmDialogOpen(false);
      setCurrentPage(1);
      setPageSize(20);
    }
  }, [open]);

  // 提取所有可用的分组名（去重）
  const availableGroupNames = useMemo(() => {
    const names = new Set<string>();
    distributedInstances.forEach(inst => {
      if (inst.groupName && inst.groupName !== '全部用户') {
        names.add(inst.groupName);
      }
    });
    return Array.from(names);
  }, [distributedInstances]);

  // 筛选后的实例列表
  const filteredInstances = useMemo(() => {
    return distributedInstances.filter(inst => {
      // 搜索过滤
      if (searchQuery) {
        const q = searchQuery.toLowerCase();
        if (!inst.name.toLowerCase().includes(q) && !inst.id.toLowerCase().includes(q)) {
          return false;
        }
      }
      // 分组过滤（多选）
      if (scopeFilters.size > 0) {
        const instGroup = inst.groupName || '全部用户';
        const hasUngrouped = scopeFilters.has('__ungrouped__');
        const groupNames = Array.from(scopeFilters).filter(n => n !== '__ungrouped__');
        const matchesGroup = groupNames.includes(instGroup);
        const matchesUngrouped = hasUngrouped && (instGroup === '全部用户' || !instGroup);
        if (!matchesGroup && !matchesUngrouped) return false;
      }
      // 状态过滤（单选）
      if (statusFilter !== 'all') {
        const instStatus = (inst.deleteStatus || 'not_deleted') as string;
        if (instStatus !== statusFilter) return false;
      }
      return true;
    });
  }, [distributedInstances, searchQuery, scopeFilters, statusFilter]);

  // 分页计算
  const totalCount = filteredInstances.length;
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));
  const safeCurrentPage = Math.min(currentPage, totalPages);
  const startIndex = (safeCurrentPage - 1) * pageSize;
  const pagedInstances = filteredInstances.slice(startIndex, startIndex + pageSize);

  // 全选判断：跨页筛选范围
  const allFilteredIds = filteredInstances.map(inst => inst.id);
  const allFilteredIdsKey = allFilteredIds.join('\u0000');
  const selectedInFilterCount = allFilteredIds.filter(id => selectedInstances.includes(id)).length;
  const isAllFilteredSelected = selectAllMode || (allFilteredIds.length > 0 && allFilteredIds.every(id => selectedInstances.includes(id)));
  const isIndeterminate = !selectAllMode && selectedInFilterCount > 0 && !isAllFilteredSelected;
  const selectedCount = selectAllMode ? allFilteredIds.length : selectedInFilterCount;

  // 全选模式下，筛选条件变化时自动同步 selectedInstances
  useEffect(() => {
    if (!selectAllMode) return;
    setSelectedInstances(prev => {
      const unchanged = prev.length === allFilteredIds.length && prev.every((id, index) => id === allFilteredIds[index]);
      return unchanged ? prev : allFilteredIds;
    });
  }, [selectAllMode, allFilteredIdsKey]);

  const toggleAll = () => {
    if (selectAllMode || isAllFilteredSelected) {
      setSelectedInstances([]);
      setSelectAllMode(false);
      return;
    }
    setSelectedInstances(allFilteredIds);
    setSelectAllMode(true);
  };

  const toggleInstance = (id: string) => {
    if (selectAllMode) return;
    setSelectedInstances(prev => {
      if (prev.includes(id)) {
        return prev.filter(x => x !== id);
      }
      return [...prev, id];
    });
  };

  const handleDelete = () => {
    if (selectedCount === 0) return;
    setConfirmDialogOpen(true);
  };

  const handleConfirmDelete = () => {
    const selectedData = selectAllMode
      ? filteredInstances
      : filteredInstances.filter(inst => selectedInstances.includes(inst.id));
    const selectedIds = selectedData.map(inst => inst.id);
    onDeleteStart(selectedIds, selectedData);
    setConfirmDialogOpen(false);
    onOpenChange(false);
  };

  const displayResourceLabel = resourceLabel === 'Skill' ? '技能' : resourceLabel;
  const uninstallIntroText = `从已下发实例中卸载${displayResourceLabel}`;

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent size="lg" className="flex max-h-[min(90vh,720px)] flex-col">
          <DialogHeader>
            <DialogTitle>批量卸载实例</DialogTitle>
          </DialogHeader>

          <DialogBody className="flex flex-1 flex-col px-6">
            <div className="flex min-h-0 flex-1 flex-col space-y-4">
              {/* 副标题描述 */}
              <BodyText tone="secondary">
                {uninstallIntroText}{' '}
                <BodyMedium tone="primary">{skillName}</BodyMedium>
              </BodyText>

              {/* Alert 提示 — 必须在内容区最上方 */}
              <Alert variant="warning">
                <CircleAlert className="size-4" />
                <AlertDescription>
                  {warningText}
                </AlertDescription>
              </Alert>

              {/* 搜索框 + 筛选 */}
              <div className="flex items-center gap-2">
                <div className="relative flex-1">
                  <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[var(--text-weak)]" />
                  <Input
                    placeholder="搜索实例名称/ID..."
                    value={searchQuery}
                    onChange={(e) => {
                      setSearchQuery(e.target.value);
                      setCurrentPage(1);
                    }}
                    className="pl-9"
                  />
                </div>

                {/* 分组筛选 — InstantMultiSelect（Portal，不被裁切） */}
                {showScopeFilter && (
                  <InstantMultiSelect
                    options={[
                      ...availableGroupNames.map(name => ({ value: name, label: name })),
                      { value: '__ungrouped__', label: '未分组' },
                    ]}
                    value={scopeFilters}
                    onChange={(next) => {
                      setScopeFilters(next);
                      setCurrentPage(1);
                    }}
                    placeholder="全部分组"
                    selectAllLabel="全部分组"
                    align="start"
                    triggerClassName="min-w-[8rem] max-w-[12rem]"
                  />
                )}

                {/* 状态筛选 — Select（Portal，不被裁切） */}
                <Select
                  value={statusFilter}
                  onValueChange={(v) => {
                    setStatusFilter(v as UninstallFilterOption);
                    setCurrentPage(1);
                  }}
                >
                  <SelectTrigger className="w-28">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent align="end">
                    {UNINSTALL_FILTER_OPTIONS.map(opt => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* 实例列表 */}
              <div className="min-h-0 flex-1 overflow-hidden rounded-[4px] border border-[var(--border)]">
                {/* 全选复选框 — 跨页全选当前筛选结果 */}
                <div className="sticky top-0 z-10 flex items-center justify-between border-b border-[var(--border)] bg-[var(--bg-grey-normal)] px-3 py-2.5">
                  <div className="flex items-center gap-3">
                    <Checkbox
                      checked={isAllFilteredSelected ? true : isIndeterminate ? 'indeterminate' : false}
                      onCheckedChange={toggleAll}
                      disabled={allFilteredIds.length === 0}
                      aria-label={isAllFilteredSelected ? '取消全选' : '全选'}
                    />
                    <BodyMedium tone="primary">全选</BodyMedium>
                  </div>
                  {selectedCount > 0 && (
                    <MetaText>已选 {selectedCount} 条</MetaText>
                  )}
                </div>

                {/* 实例项 */}
                <div className="max-h-[260px] overflow-y-auto">
                  {pagedInstances.length === 0 ? (
                    <div className="py-12 text-center">
                      <HelperText>{emptyText}</HelperText>
                    </div>
                  ) : (
                    pagedInstances.map(inst => {
                      const isSelected = selectAllMode || selectedInstances.includes(inst.id);
                      const isRowDisabled = selectAllMode;
                      const deleteStatus = inst.deleteStatus || 'not_deleted';
                      return (
                        <div
                          key={inst.id}
                          onClick={() => !isRowDisabled && toggleInstance(inst.id)}
                          className={`flex items-center gap-3 border-b border-[var(--border)] px-3 py-3 transition-colors last:border-b-0 ${
                            isRowDisabled ? 'cursor-not-allowed' : 'cursor-pointer'
                          } ${
                            isSelected ? 'bg-[var(--bg-brand-selected)]' : 'hover:bg-[var(--bg-grey-hover-subtle)]'
                          }`}
                        >
                          <div className="shrink-0 self-center">
                            <Checkbox
                              checked={isSelected}
                              disabled={isRowDisabled}
                              onCheckedChange={() => toggleInstance(inst.id)}
                              onClick={(e) => e.stopPropagation()}
                              aria-label={`选择 ${inst.name}`}
                            />
                          </div>
                          <div className="min-w-0 flex-1">
                            <div className="flex items-baseline gap-3">
                              <BodyMedium tone="primary" className="truncate">
                                {inst.name}
                              </BodyMedium>
                              <MetaText tone="weak" className="shrink-0 font-mono">
                                {inst.id}
                              </MetaText>
                            </div>
                            {!hideCreatorAndGroup && (
                              <div className="mt-0.5 flex items-center gap-3">
                                <MetaText>创建人：{inst.createdBy}</MetaText>
                                <MetaText>分组：{inst.groupName || '全部用户'}</MetaText>
                              </div>
                            )}
                          </div>
                          {/* 右侧：卸载状态 + 版本号 */}
                          <div className="shrink-0 text-right">
                            {deleteStatus === 'delete_failed' ? (
                              <Tooltip delayDuration={300}>
                                <TooltipTrigger asChild>
                                  <span className="inline-block rounded-full bg-[var(--bg-grey-normal)] px-2 py-0.5 text-xs font-medium text-[var(--cp-text-danger)]">
                                    卸载失败
                                  </span>
                                </TooltipTrigger>
                                <TooltipContent side="top">
                                  <MetaText tone="inherit">{inst.deleteFailReason || '未知原因'}</MetaText>
                                </TooltipContent>
                              </Tooltip>
                            ) : (
                              <span className="inline-block rounded-full bg-[var(--bg-grey-normal)] px-2 py-0.5 text-xs font-medium text-[var(--cp-text-muted)]">
                                未卸载
                              </span>
                            )}
                            <MetaText tone="weak" className="mt-0.5 block text-center">
                              v{inst.distributedVersion || skillVersion}
                            </MetaText>
                          </div>
                        </div>
                      );
                    })
                  )}
                </div>
              </div>

              {/* 分页控件 — 弹窗场景用 simple 模式（规范 §12.5） */}
              <div className="pt-3">
                {selectAllMode && selectedCount > 0 && (
                  <p className="mb-1 text-xs text-[var(--cp-text-muted)]">已选择全部符合条件的实例</p>
                )}
                <Pagination
                  total={totalCount}
                  current={safeCurrentPage}
                  pageSize={pageSize}
                  mode="simple"
                  showTotal={() => `共 ${totalCount} 条，第 ${safeCurrentPage} / ${totalPages} 页`}
                  className="w-full justify-between"
                  onChange={(page) => {
                    setCurrentPage(page);
                  }}
                />
              </div>
            </div>
          </DialogBody>

          {/* 底部操作 — 警示弹窗主按钮使用 destructive */}
          <DialogFooter>
            <Button variant="claw-outline" onClick={() => onOpenChange(false)}>
              取消
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={selectedCount === 0}
            >
              确认卸载（{selectedCount}）
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 二次确认弹窗 */}
      <AlertDialog open={confirmDialogOpen} onOpenChange={setConfirmDialogOpen}>
        <AlertDialogContent className="sm:max-w-[420px]">
          <AlertDialogHeader>
            <AlertDialogTitle>确认卸载</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <BodyText as="p" tone="primary">
              确定要从 <BodyMedium tone="primary">{selectedCount}</BodyMedium> 个实例中卸载{displayResourceLabel}「{skillName}」吗？
              <BodyText tone="danger">卸载后该{displayResourceLabel}将恢复为未下发状态。</BodyText>
            </BodyText>
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmDelete}>
              确认卸载
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

    </>
  );
}
