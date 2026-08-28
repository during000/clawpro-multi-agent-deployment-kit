'use client';
import { useState, useEffect, useMemo } from 'react';
import { Button } from '@/components/ui/button';
import { DialogPagination } from '@/components/ui/pagination';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { ScopeSelect } from '@/components/ScopeSelect';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
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
} from '@/components/ui/select';
import { Search, AlertTriangle, CircleAlert } from 'lucide-react';
import type { Group } from './types';

/** 卸载状态筛选选项（单选） */
type UninstallFilterOption = 'all' | 'not_deleted' | 'delete_failed';

const UNINSTALL_FILTER_OPTIONS: { key: UninstallFilterOption; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'not_deleted', label: '未卸载' },
  { key: 'delete_failed', label: '卸载失败' },
];

interface PublicBatchDeleteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  skillName: string;
  skillVersion: string;
  resourceLabel?: string;
  introNode?: React.ReactNode;
  warningText?: string;
  emptyText?: string;
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
  /** 是否显示组织筛选，默认 true */
  showScopeFilter?: boolean;
  /** 隐藏实例中的创建人、组织信息 */
  hideCreatorAndGroup?: boolean;
  /** 是否显示右侧版本号，默认 true */
  showDistributedVersion?: boolean;
}

export default function PublicBatchDeleteDialog({
  open,
  onOpenChange,
  skillName,
  skillVersion,
  resourceLabel = '技能',
  introNode,
  warningText = '通过下发按钮安装的技能可支持移出（包括用户下发和管理端下发）。卸载成功后，该技能在对应实例上恢复为"未下发"状态。',
  emptyText = '暂无已下发的实例',
  distributedInstances,
  onDeleteStart,
  showScopeFilter = true,
  hideCreatorAndGroup = false,
  showDistributedVersion = true,
}: PublicBatchDeleteDialogProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedInstances, setSelectedInstances] = useState<string[]>([]);
  /** 是否处于“选择全部”模式（跨页全选） */
  const [selectAllMode, setSelectAllMode] = useState(false);
  /** 状态单选筛选 */
  const [statusFilter, setStatusFilter] = useState<UninstallFilterOption>('all');
  /** 组织筛选：空数组=全部, 否则为选中的组织名列表（多选） */
  const [scopeFilters, setScopeFilters] = useState<string[]>([]);
  const [confirmDialogOpen, setConfirmDialogOpen] = useState(false);
  /** 全选模式卸载二次确认弹窗 */
  const [selectAllConfirmOpen, setSelectAllConfirmOpen] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 20;

  // 打开时重置状态
  useEffect(() => {
    if (open) {
      setSearchQuery('');
      setSelectedInstances([]);
      setSelectAllMode(false);
      setStatusFilter('all');
      setScopeFilters([]);
      setConfirmDialogOpen(false);
      setSelectAllConfirmOpen(false);
      setCurrentPage(1);
    }
  }, [open]);

  // 提取所有可用的组织名（去重）
  const availableGroupNames = useMemo(() => {
    const names = new Set<string>();
    distributedInstances.forEach(inst => {
      if (inst.groupName && inst.groupName !== '全部用户') {
        names.add(inst.groupName);
      }
    });
    return Array.from(names);
  }, [distributedInstances]);
  const scopeGroups = useMemo(
    () => availableGroupNames.map((name) => ({ id: name, name })),
    [availableGroupNames],
  );
  const selectedScopeKeys = useMemo(
    () => new Set(scopeFilters.length === 0 ? availableGroupNames : scopeFilters),
    [availableGroupNames, scopeFilters],
  );

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
      // 组织过滤（多选）
      if (scopeFilters.length > 0) {
        const instGroup = inst.groupName || '全部用户';
        if (!scopeFilters.includes(instGroup)) return false;
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
  const selectedCount = selectAllMode ? allFilteredIds.length : selectedInFilterCount;
  const isAllFilteredSelected = selectAllMode || (allFilteredIds.length > 0 && allFilteredIds.every(id => selectedInstances.includes(id)));
  const isIndeterminate = !selectAllMode && selectedInFilterCount > 0 && !isAllFilteredSelected;

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
    if (selectAllMode) {
      setSelectAllConfirmOpen(true);
      return;
    }
    setConfirmDialogOpen(true);
  };

  const handleConfirmDelete = () => {
    const selectedData = selectAllMode
      ? filteredInstances
      : filteredInstances.filter(inst => selectedInstances.includes(inst.id));
    const selectedIds = selectedData.map(inst => inst.id);
    onDeleteStart(selectedIds, selectedData);
    setConfirmDialogOpen(false);
    setSelectAllConfirmOpen(false);
    onOpenChange(false);
  };

  const displayResourceLabel = resourceLabel === 'Skill' ? '技能' : resourceLabel;
  const uninstallIntroText =
    resourceLabel === 'Skill' ? '已下发实例中卸载技能' : `从已下发实例中卸载${displayResourceLabel}`;

  const scopeFilterLabel = useMemo(() => {
    if (scopeFilters.length === 0 || (availableGroupNames.length > 0 && availableGroupNames.every((name) => scopeFilters.includes(name)))) {
      return '全部分组';
    }
    return scopeFilters.join('、');
  }, [availableGroupNames, scopeFilters]);

  const handleScopeChange = (next: Set<string>) => {
    const arr = Array.from(next).filter((key) => availableGroupNames.includes(key));
    if (arr.length === 0 || (availableGroupNames.length > 0 && availableGroupNames.every((key) => arr.includes(key)))) {
      setScopeFilters([]);
    } else {
      setScopeFilters(arr);
    }
    setCurrentPage(1);
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="flex max-h-[min(90vh,720px)] flex-col sm:max-w-[720px]">
          <DialogHeader>
            <DialogTitle>批量卸载实例</DialogTitle>
          </DialogHeader>
          <DialogBody className="flex flex-1 flex-col px-6">
            <div className="flex min-h-0 flex-1 flex-col space-y-4">
              <p className="text-sm text-[var(--text-secondary)]">
                {introNode || (
                  <>
                    {uninstallIntroText} <span className="font-medium text-[var(--text-title)]">{skillName}</span>
                  </>
                )}
              </p>
              <Alert variant="warning">
                <CircleAlert />
                <AlertDescription>{warningText}</AlertDescription>
              </Alert>

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

                {showScopeFilter && (
                  <ScopeSelect
                    withTrigger
                    groups={scopeGroups}
                    selectedKeys={selectedScopeKeys}
                    onChange={handleScopeChange}
                    searchPlaceholder="搜索分组..."
                    allLabel="全部分组"
                    groupSectionLabel="按分组"
                    selectedCountTemplate="已选 {count} 个分组"
                    hidePublicGroup
                    triggerLabel={scopeFilterLabel}
                    triggerClassName="flex h-9 w-full min-w-[8rem] max-w-[12rem] items-center justify-between gap-2 whitespace-nowrap rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 py-[5px] text-sm font-normal outline-none transition-colors hover:border-[var(--cp-brand-blue)] data-[state=open]:border-[var(--cp-brand-blue)] disabled:cursor-not-allowed disabled:border-[var(--border)] disabled:bg-[var(--bg-grey-normal)] disabled:text-[var(--text-weak)]"
                    align="end"
                  />
                )}

                <Select
                  value={statusFilter}
                  onValueChange={(value) => {
                    setStatusFilter(value as UninstallFilterOption);
                    setCurrentPage(1);
                  }}
                >
                  <SelectTrigger className="w-28">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {UNINSTALL_FILTER_OPTIONS.map((opt) => (
                      <SelectItem key={opt.key} value={opt.key}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

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
                    <span className="text-sm font-medium text-[var(--text-title)]">
                      全选
                    </span>
                  </div>
                  {selectedCount > 0 && (
                    <span className="text-sm text-[var(--text-weak)]">
                      已选 {selectedCount} 条
                    </span>
                  )}
                </div>

            {/* 实例项 */}
              <div className="max-h-[260px] overflow-y-auto">
                {pagedInstances.length === 0 ? (
                  <div className="py-12 text-center">
                    <p className="text-xs text-[var(--text-muted)]">{emptyText}</p>
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
                          isRowDisabled ? 'cursor-not-allowed' : 'cursor-pointer hover:bg-[var(--bg-grey-hover-subtle)]'
                        } ${isSelected ? 'bg-[var(--bg-brand-selected)]' : ''}`}
                      >
                        <div className="flex-shrink-0 self-center">
                          <Checkbox
                            checked={isSelected}
                            disabled={isRowDisabled}
                            onCheckedChange={() => toggleInstance(inst.id)}
                            onClick={(e) => e.stopPropagation()}
                            aria-label={`选择 ${inst.name}`}
                          />
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-baseline gap-3">
                            <span className="truncate text-sm font-medium text-[var(--text-title)]">{inst.name}</span>
                            <span className="flex-shrink-0 font-mono text-xs text-[var(--text-weak)]">{inst.id}</span>
                          </div>
                          <div className="mt-0.5 flex items-center gap-3">
                            {!hideCreatorAndGroup && (
                              <>
                                <span className="text-xs text-[var(--text-weak)]">创建人：{inst.createdBy}</span>
                                <span className="text-xs text-[var(--text-weak)]">组织：{inst.groupName || '全部用户'}</span>
                              </>
                            )}
                          </div>
                        </div>
                        <div className="flex-shrink-0 text-right">
                          {deleteStatus === 'delete_failed' ? (
                            <Tooltip delayDuration={300}>
                              <TooltipTrigger asChild>
                                <span className="inline-flex cursor-help items-center justify-center rounded-full border border-[var(--cp-border)] bg-[var(--cp-surface)] px-2.5 py-0.5 text-[10px] font-normal text-[var(--text-danger)]">
                                  卸载失败
                                </span>
                              </TooltipTrigger>
                              <TooltipContent side="top">
                                <span className="text-xs">{inst.deleteFailReason || '未知原因'}</span>
                              </TooltipContent>
                            </Tooltip>
                          ) : (
                            <span className="inline-flex items-center justify-center rounded-full border border-[var(--cp-border)] bg-[var(--cp-surface)] px-2.5 py-0.5 text-[10px] font-normal text-[var(--text-muted)]">
                              未卸载
                            </span>
                          )}
                          {showDistributedVersion && (inst.distributedVersion || skillVersion) ? (
                            <div className="mt-0.5 text-center text-[11px] text-[var(--text-weak)]">
                              v{inst.distributedVersion || skillVersion}
                            </div>
                          ) : null}
                        </div>
                      </div>
                    );
                  })
                )}
              </div>
            </div>

            <div className="pt-3">
              {selectAllMode && selectedCount > 0 && (
                <p className="mb-1 text-xs text-[var(--cp-text-muted)]">已选择全部符合条件的实例</p>
              )}
              <DialogPagination
                total={totalCount}
                currentPage={safeCurrentPage}
                totalPages={totalPages}
                onPrevPage={() => {
                  setCurrentPage((p) => Math.max(1, p - 1));
                }}
                onNextPage={() => {
                  setCurrentPage((p) => Math.min(totalPages, p + 1));
                }}
              />
            </div>
          </div>
          </DialogBody>

          <DialogFooter className="items-center gap-2 justify-end">
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
        <AlertDialogContent className="sm:max-w-sm">
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-[var(--text-danger)]" />
              确认卸载
            </AlertDialogTitle>
            <AlertDialogDescription>
              确定要从 <span className="font-semibold text-[var(--text-emphasis)]">{selectedCount}</span> 个实例中卸载{displayResourceLabel}「{skillName}」吗？卸载后该{displayResourceLabel}将恢复为未下发状态。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmDelete}>
              确认卸载
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 全选模式卸载二次确认弹窗 */}
      <AlertDialog open={selectAllConfirmOpen} onOpenChange={setSelectAllConfirmOpen}>
        <AlertDialogContent className="sm:max-w-sm">
          <AlertDialogHeader>
            <AlertDialogTitle>确认批量卸载</AlertDialogTitle>
            <AlertDialogDescription>
              当前已选择全部符合筛选条件的 {selectedCount} 个实例，确认从这些实例中卸载{displayResourceLabel}「{skillName}」吗？
            </AlertDialogDescription>
          </AlertDialogHeader>
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
