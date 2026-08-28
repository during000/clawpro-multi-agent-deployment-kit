/**
 * 加入初始技能包弹窗
 * 用于从公共技能库或企业技能库将技能/技能包加入到初始技能包
 */
import { useEffect, useMemo, useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { ScopeSelect } from '@/components/ScopeSelect';
import { BodyMedium, BodyText, HelperText, MetaText } from '@/components/ui/Typography';
import { Search, X } from 'lucide-react';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { toast } from 'sonner';

const STATUS_FILTERS = [
  { id: 'all', label: '全部' },
  { id: 'active', label: '生效中' },
  { id: 'inactive', label: '未生效' },
] as const;

type StatusFilter = (typeof STATUS_FILTERS)[number]['id'];
type ScopeFilterKey = string;

interface AgentGroupOption {
  id: string;
  name: string;
  parentId?: string | null;
}

interface PackageOption {
  id: string;
  name: string;
  isActive: boolean;
  scopeType?: 'all-users' | 'groups';
  scopeLabel?: string;
  groupIds?: string[];
}

type AddToPackageItemType = 'skill' | 'package';

interface AddToPackageDialogProps {
  open: boolean;
  itemName: string;
  itemType?: AddToPackageItemType;
  packages: PackageOption[];
  groups?: AgentGroupOption[];
  addedPackageIds?: string[];
  successMessage?: (packageCount: number) => string;
  onConfirm: (packageIds: string[]) => void;
  onCancel: () => void;
}

const getPackageScopeType = (pkg: PackageOption) => pkg.scopeType ?? 'all-users';

export default function AddToPackageDialog({
  open,
  itemName,
  itemType = 'skill',
  packages,
  groups = [],
  addedPackageIds = [],
  successMessage,
  onConfirm,
  onCancel,
}: AddToPackageDialogProps) {
  const [selectedPackageIds, setSelectedPackageIds] = useState<string[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [scopeFilters, setScopeFilters] = useState<ScopeFilterKey[]>([]);
  const [scopeDropdownOpen, setScopeDropdownOpen] = useState(false);

  const addedPackageIdSet = useMemo(() => new Set(addedPackageIds), [addedPackageIds]);
  const groupNameMap = useMemo(() => new Map(groups.map((group) => [group.id, group.name])), [groups]);
  const allGroupKeys = useMemo(() => groups.map((group) => group.id), [groups]);
  const selectedCount = selectedPackageIds.length;
  const itemTypeLabel = itemType === 'package' ? '技能包' : '技能';
  const availablePackageCount = useMemo(
    () => packages.filter((pkg) => !addedPackageIdSet.has(pkg.id)).length,
    [addedPackageIdSet, packages],
  );
  const isAllAdded = packages.length > 0 && availablePackageCount === 0;

  const selectedScopeKeys = useMemo(() => new Set<string>(
    scopeFilters.length === 0 ? allGroupKeys : scopeFilters,
  ), [allGroupKeys, scopeFilters]);

  const scopeFilterLabel = useMemo(() => {
    if (scopeFilters.length === 0 || (allGroupKeys.length > 0 && allGroupKeys.every((key) => scopeFilters.includes(key)))) return '全部分组';
    return scopeFilters
      .map((groupId) => groupNameMap.get(groupId) ?? groupId)
      .join('、');
  }, [allGroupKeys, groupNameMap, scopeFilters]);

  const getPackageScopeLabel = (pkg: PackageOption) => {
    if (getPackageScopeType(pkg) !== 'groups') return '全部分组';
    if (pkg.scopeLabel) return pkg.scopeLabel;
    const names = (pkg.groupIds ?? []).map((groupId) => groupNameMap.get(groupId) ?? groupId);
    return names.length > 0 ? names.join('、') : '指定用户分组';
  };

  const filteredPackages = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    return packages
      .filter((pkg) => {
        const matchesSearch = !q || pkg.name.toLowerCase().includes(q) || pkg.id.toLowerCase().includes(q);
        const matchesStatus = statusFilter === 'all'
          || (statusFilter === 'active' && pkg.isActive)
          || (statusFilter === 'inactive' && !pkg.isActive);
        const matchesScope = scopeFilters.length === 0
          ? true
          : getPackageScopeType(pkg) === 'all-users'
            ? true
            : (pkg.groupIds ?? []).some((groupId) => scopeFilters.includes(groupId));
        return matchesSearch && matchesStatus && matchesScope;
      })
      .sort((a, b) => Number(addedPackageIdSet.has(a.id)) - Number(addedPackageIdSet.has(b.id)));
  }, [addedPackageIdSet, packages, searchQuery, scopeFilters, statusFilter]);

  useEffect(() => {
    if (open) {
      setSelectedPackageIds([]);
      setSearchQuery('');
      setStatusFilter('all');
      setScopeFilters([]);
      setScopeDropdownOpen(false);
    }
  }, [open, itemName]);

  const handleClose = () => {
    setSelectedPackageIds([]);
    onCancel();
  };

  const handleTogglePackage = (packageId: string) => {
    if (addedPackageIdSet.has(packageId)) return;
    setSelectedPackageIds((prev) => (
      prev.includes(packageId)
        ? prev.filter((id) => id !== packageId)
        : [...prev, packageId]
    ));
  };

  const handleScopeChange = (next: Set<string>) => {
    const arr = Array.from(next).filter((key) => groups.some((group) => group.id === key));
    if (arr.length === 0 || (allGroupKeys.length > 0 && allGroupKeys.every((key) => arr.includes(key)))) {
      setScopeFilters([]);
    } else {
      setScopeFilters(arr);
    }
  };

  const handleConfirm = () => {
    if (selectedPackageIds.length === 0) return;
    onConfirm(selectedPackageIds);
    toast.success(successMessage?.(selectedPackageIds.length) ?? `已添加到 ${selectedPackageIds.length} 个初始技能包`, { duration: 2000 });
    handleClose();
  };

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen) handleClose(); }}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>将&quot;{itemName}&quot;添加到初始技能包</DialogTitle>
          <DialogDescription>
            请选择要加入的初始技能包，支持按名称 / ID、状态和分组快速筛选。
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="grid grid-cols-[minmax(0,1fr)_auto_auto] gap-3">
            <div className="relative min-w-0">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--text-weak)]" />
              <Input
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="按名称 / ID 搜索初始技能包"
                className="bg-white pl-9 pr-8"
              />
              {searchQuery && (
                <button
                  type="button"
                  onClick={() => setSearchQuery('')}
                  className="absolute right-2 top-1/2 flex h-5 w-5 -translate-y-1/2 items-center justify-center rounded-[4px] text-[var(--text-weak)] transition-colors hover:bg-[var(--bg-grey-hover)] hover:text-[var(--text-title)]"
                  aria-label="清除搜索"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              )}
            </div>

            <ScopeSelect
              withTrigger
              groups={groups}
              selectedKeys={selectedScopeKeys}
              onChange={handleScopeChange}
              searchPlaceholder="搜索分组..."
              allLabel="全部分组"
              groupSectionLabel="按分组"
              selectedCountTemplate="已选 {count} 个分组"
              hidePublicGroup
              triggerLabel={scopeFilterLabel}
              triggerClassName="flex h-9 w-48 items-center justify-between gap-2 rounded-[4px] border border-border bg-white px-3 text-sm font-normal whitespace-nowrap transition-colors outline-none hover:border-blue-500 data-[state=open]:border-blue-500 data-[placeholder]:text-[var(--text-weak)]"
              align="end"
              open={scopeDropdownOpen}
              onOpenChange={setScopeDropdownOpen}
            />

            <Tabs value={statusFilter} onValueChange={(v) => setStatusFilter(v as StatusFilter)}>
              <TabsList>
                {STATUS_FILTERS.map((filter) => (
                  <TabsTrigger key={filter.id} value={filter.id}>{filter.label}</TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between gap-3">
              <BodyText as="p" tone="primary">初始技能包列表</BodyText>
              <MetaText tone="weak">共 {filteredPackages.length} / {packages.length} 个</MetaText>
            </div>

            {packages.length === 0 ? (
              <div className="rounded-[4px] border border-dashed border-[var(--cp-border)] py-10 text-center">
                <HelperText>暂无初始技能包，请先创建初始技能包</HelperText>
              </div>
            ) : (
              <>
                {isAllAdded && (
                  <div className="rounded-[4px] border border-[var(--cp-border)] bg-[var(--bg-brand-selected)] px-3 py-2.5">
                    <BodyText tone="primary">当前{itemTypeLabel}已加入全部初始技能包，无需重复添加。</BodyText>
                  </div>
                )}

                {filteredPackages.length > 0 ? (
                  <div className="max-h-[320px] space-y-2 overflow-y-auto pr-1">
                    {filteredPackages.map((pkg) => {
                      const isAdded = addedPackageIdSet.has(pkg.id);
                      const isSelected = selectedPackageIds.includes(pkg.id);
                      return (
                        <button
                          key={pkg.id}
                          type="button"
                          disabled={isAdded}
                          onClick={() => handleTogglePackage(pkg.id)}
                          className={`w-full rounded-[4px] border px-3 py-2.5 text-left transition-colors ${
                            isAdded
                              ? 'cursor-not-allowed border-[var(--cp-border)] bg-[var(--bg-grey-hover)] opacity-70'
                              : isSelected
                                ? 'border-[var(--text-brand)] bg-[var(--bg-brand-selected)]'
                                : 'border-[var(--cp-border)] bg-white hover:border-[var(--text-brand)] hover:bg-[var(--bg-grey-hover-subtle)]'
                          }`}
                        >
                          <span className="flex items-start justify-between gap-3">
                            <span className="flex min-w-0 items-start gap-2">
                              <Checkbox
                                checked={isSelected || isAdded}
                                disabled={isAdded}
                                tabIndex={-1}
                                className="pointer-events-none mt-0.5"
                              />
                              <span className="min-w-0 space-y-1">
                                <BodyMedium as="span" tone={isAdded ? 'muted' : 'primary'} className="block truncate">
                                  {pkg.name}
                                </BodyMedium>
                                <MetaText as="span" tone="weak" className="block truncate">
                                  ID：{pkg.id} · 应用范围：{getPackageScopeLabel(pkg)}
                                </MetaText>
                              </span>
                            </span>
                            <span className="flex shrink-0 flex-wrap justify-end gap-1.5">
                              <Badge variant="outline" className="text-[10px]">
                                {pkg.isActive ? '生效中' : '未生效'}
                              </Badge>
                              {isAdded && (
                                <Badge variant="outline" className="text-[10px]">
                                  已加入
                                </Badge>
                              )}
                            </span>
                          </span>
                        </button>
                      );
                    })}
                  </div>
                ) : (
                  <div className="rounded-[4px] border border-dashed border-[var(--cp-border)] py-10 text-center">
                    <HelperText>未找到符合筛选条件的初始技能包</HelperText>
                  </div>
                )}
              </>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button variant="claw-outline" onClick={handleClose}>取消</Button>
          <Button
            variant="dialog-confirm"
            onClick={handleConfirm}
            disabled={selectedCount === 0}
          >
            确认添加（{selectedCount}）
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
