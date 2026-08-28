/**
 * AddAssetsDialog - 统一的「添加项目资产」弹窗（多 Tab 勾选）
 * - Tab：模型配置 / 企业技能 / 企业插件 / 企业 MCP / 企业规范 / 公共技能
 * - 组织列出当前组织/上级组织/全部用户范围的资产；项目模型列出当前项目/全部用户范围的模型
 * - 企业 4 类支持内嵌「上传新资产」，上传后自动加入该组织并勾选
 * - 确认后一次性返回各大类的最终勾选集合（覆盖式）
 */
import { useEffect, useMemo, useState } from 'react';
import { Search, Check, Plus, Upload } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogBody,
} from '@/components/ui/dialog';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { StatusTag } from '@/components/ui/status-tag';
import { Empty, EmptyHeader, EmptyDescription } from '@/components/ui/empty';
import { BodyMedium, MetaText } from '@/components/ui/Typography';
import type { UserGroup } from '../MemberManagement/types';
import { ASSET_CATEGORY_MAP, type AssetCategory } from './types';
import {
  getAssetVersionLabel,
  getSelectableItems,
  getItemScopeTag,
  getScopeAncestorIds,
} from './assetSelectors';
import { skillStore } from '../SkillLibrary/skillStore';
import { pluginStore } from '../SkillLibrary/pluginStore';
import { mcpStore } from '../SkillLibrary/mcpStore';
import { standardsStore } from '../SkillLibrary/standardsStore';
import type { Skill, MCPService } from '../SkillLibrary/types';
import type { Plugin } from '../SkillLibrary/PluginUploadDialog';
import type { AgentConfigAsset } from '../SkillLibrary/standardsStore';
import SkillUploadDialog from '../SkillLibrary/SkillUploadDialog';
import PluginUploadDialog from '../SkillLibrary/PluginUploadDialog';
import MCPAddDialog from '../SkillLibrary/MCPAddDialog';
import StandardUploadDialog from './StandardUploadDialog';
import { AddModelDialog } from '../ModelConfig/AddModelDialog';

/** Tab 顺序（模型配置在前，企业 4 类与公共技能在后） */
const TAB_ORDER: AssetCategory[] = [
  'modelConfig',
  'enterpriseSkill',
  'enterprisePlugin',
  'enterpriseMcp',
  'enterpriseStandard',
  'publicSkill',
];

export type CheckedMap = Record<AssetCategory, Set<string>>;

interface AddAssetsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  groupId: string;
  groupName: string;
  groups: UserGroup[];
  /** 各大类当前已选中的 refId 列表（草稿态） */
  selectedRefIds: Record<AssetCategory, string[]>;
  /** 确认后返回各大类最终选中集合（覆盖式） */
  onConfirm: (result: Record<AssetCategory, string[]>) => void;
}

export default function AddAssetsDialog({
  open,
  onOpenChange,
  groupId,
  groupName,
  groups,
  selectedRefIds,
  onConfirm,
}: AddAssetsDialogProps) {
  const [activeTab, setActiveTab] = useState<AssetCategory>('modelConfig');
  const [query, setQuery] = useState('');
  const [checked, setChecked] = useState<CheckedMap>(() => emptyCheckedMap());
  const [uploadCategory, setUploadCategory] = useState<AssetCategory | null>(null);
  const [showAddModelDialog, setShowAddModelDialog] = useState(false);
  // 上传后触发的库刷新，用于重算可选列表
  const [libTick, setLibTick] = useState(0);

  // 当前节点是组织还是项目：项目模型范围独立，其他资产沿用既有直属组织继承规则。
  const isProject = useMemo(
    () => groups.find((g) => g.id === groupId)?.source === 'project',
    [groups, groupId],
  );
  const term = isProject ? '项目' : '组织';
  const scopeAncestorIds = useMemo(
    () => getScopeAncestorIds(groupId, groups, isProject, activeTab),
    [groupId, groups, isProject, activeTab],
  );

  useEffect(() => {
    if (open) {
      const next = emptyCheckedMap();
      TAB_ORDER.forEach((cat) => {
        next[cat] = new Set(selectedRefIds[cat] || []);
      });
      setChecked(next);
      setQuery('');
      setActiveTab('modelConfig');
    }
  }, [open, selectedRefIds]);

  const items = useMemo(
    () => getSelectableItems(activeTab, groupId, groups, isProject),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [activeTab, groupId, groups, isProject, libTick],
  );

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return items;
    return items.filter(
      (i) => i.name.toLowerCase().includes(q) || (i.description || '').toLowerCase().includes(q),
    );
  }, [items, query]);

  const toggle = (category: AssetCategory, refId: string) => {
    setChecked((prev) => {
      const nextSet = new Set(prev[category]);
      if (nextSet.has(refId)) nextSet.delete(refId);
      else nextSet.add(refId);
      return { ...prev, [category]: nextSet };
    });
  };

  const addChecked = (category: AssetCategory, refId: string) => {
    setChecked((prev) => {
      const nextSet = new Set(prev[category]);
      nextSet.add(refId);
      return { ...prev, [category]: nextSet };
    });
  };

  const totalChecked = useMemo(
    () => TAB_ORDER.reduce((sum, cat) => sum + checked[cat].size, 0),
    [checked],
  );

  const handleConfirm = () => {
    const result = {} as Record<AssetCategory, string[]>;
    TAB_ORDER.forEach((cat) => {
      result[cat] = Array.from(checked[cat]);
    });
    onConfirm(result);
    onOpenChange(false);
  };

  const lockedScope = { lockedGroupId: groupId, lockedGroupName: groupName };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent
          className="sm:max-w-[680px] flex flex-col"
          style={{ maxHeight: 'min(92vh, 860px)' }}
          onPointerDownOutside={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>添加{term}资产</DialogTitle>
          </DialogHeader>

          <DialogBody className="flex-1 min-h-0 px-6">
            <Tabs
              value={activeTab}
              onValueChange={(v) => {
                setActiveTab(v as AssetCategory);
                setQuery('');
              }}
              className="gap-3"
            >
              <TabsList className="w-full">
                {TAB_ORDER.map((cat) => {
                  const count = checked[cat].size;
                  return (
                    <TabsTrigger key={cat} value={cat} className="flex-1">
                      {ASSET_CATEGORY_MAP[cat].label}
                      {count > 0 && (
                        <span className="ml-0.5 text-xs tabular-nums text-[var(--text-brand)]">
                          {count}
                        </span>
                      )}
                    </TabsTrigger>
                  );
                })}
              </TabsList>

              {TAB_ORDER.map((cat) => (
                <TabsContent key={cat} value={cat} className="space-y-3">
                  <div className="flex items-start justify-between gap-3">
                    <MetaText tone="secondary" className="min-w-0 leading-[1.5]">
                      {cat === 'publicSkill'
                        ? `从公共技能市场勾选加入该${term}资产`
                        : isProject && cat === 'modelConfig'
                          ? `仅展示应用范围为当前项目或全部用户的${ASSET_CATEGORY_MAP[cat].label}`
                          : isProject
                            ? `仅展示应用范围为当前项目、所属组织或全部用户的${ASSET_CATEGORY_MAP[cat].label}`
                          : `仅展示应用范围为当前组织、上级组织或全部用户的${ASSET_CATEGORY_MAP[cat].label}`}
                    </MetaText>
                    {cat === 'modelConfig' ? (
                      <Button
                        variant="claw-outline"
                        size="sm"
                        className="shrink-0"
                        onClick={() => setShowAddModelDialog(true)}
                      >
                        <Plus className="w-4 h-4" />
                        添加模型
                      </Button>
                    ) : cat !== 'publicSkill' && (
                      <Button
                        variant="claw-outline"
                        size="sm"
                        className="shrink-0"
                        onClick={() => setUploadCategory(cat)}
                      >
                        <Upload className="w-4 h-4" />
                        上传新{ASSET_CATEGORY_MAP[cat].label}
                      </Button>
                    )}
                  </div>

                  <div className="relative">
                    <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[var(--text-weak)]" />
                    <Input
                      placeholder={`搜索${ASSET_CATEGORY_MAP[cat].label}...`}
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                      className="pl-10"
                    />
                  </div>

                  {/* 固定高度列表区：无论条目多少，弹窗高度保持稳定 */}
                  <div className="h-[440px] overflow-y-auto pr-0.5">
                    {filtered.length === 0 ? (
                      <Empty className="h-full justify-center">
                        <EmptyHeader>
                          <EmptyDescription>
                            {items.length === 0
                              ? `暂无可选的${ASSET_CATEGORY_MAP[cat].label}`
                              : '没有匹配的结果'}
                          </EmptyDescription>
                        </EmptyHeader>
                      </Empty>
                    ) : (
                      <div className="space-y-1.5">
                        {filtered.map((item) => {
                          const isChecked = checked[cat].has(item.refId);
                          const scope = getItemScopeTag(item, groupId, groups, scopeAncestorIds);
                          return (
                            <div
                              key={item.refId}
                              className={`w-full flex items-start gap-3 px-3 py-2.5 rounded-[4px] border text-left transition-colors ${
                                isChecked
                                  ? 'border-[var(--cp-brand-blue)] bg-[var(--bg-brand-selected)]'
                                  : 'border-[var(--cp-border)] bg-white hover:border-[var(--cp-brand-blue)]'
                              }`}
                            >
                              <Checkbox
                                checked={isChecked}
                                onCheckedChange={() => toggle(cat, item.refId)}
                                className="mt-0.5"
                                aria-label={`选择 ${item.name}`}
                              />
                              <button
                                type="button"
                                onClick={() => toggle(cat, item.refId)}
                                className="min-w-0 flex-1 text-left"
                              >
                                <div className="flex items-center gap-2">
                                  <BodyMedium className="truncate min-w-0">{item.name}</BodyMedium>
                                  <div className="flex items-center gap-1.5 shrink-0">
                                    <StatusTag variant="gray" mode="soft">
                                      {getAssetVersionLabel(cat, item.version)}
                                    </StatusTag>
                                    <StatusTag
                                      variant={scope.level === 'self' ? 'blue' : 'gray'}
                                      mode="soft"
                                      className="max-w-[200px]"
                                    >
                                      <span className="truncate">{scope.label}</span>
                                    </StatusTag>
                                  </div>
                                </div>
                                {item.description && (
                                  <MetaText tone="secondary" className="line-clamp-1 mt-0.5">
                                    {item.description}
                                  </MetaText>
                                )}
                              </button>
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </div>
                </TabsContent>
              ))}
            </Tabs>
          </DialogBody>

          <DialogFooter className="flex items-center justify-between">
            <MetaText tone="secondary">已选 {totalChecked} 项</MetaText>
            <div className="flex items-center gap-2">
              <Button variant="claw-outline" onClick={() => onOpenChange(false)}>
                取消
              </Button>
              <Button variant="claw-primary" onClick={handleConfirm}>
                <Check className="w-4 h-4" />
                确定
              </Button>
            </div>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AddModelDialog
        open={showAddModelDialog}
        onOpenChange={setShowAddModelDialog}
        onAdded={(model) => {
          addChecked('modelConfig', model.id);
          setLibTick((value) => value + 1);
        }}
      />

      {/* 内嵌上传弹窗（企业 4 类，应用范围锁定当前组织/项目；上传后自动勾选） */}
      <SkillUploadDialog
        open={uploadCategory === 'enterpriseSkill'}
        onOpenChange={(o) => !o && setUploadCategory(null)}
        lockedScope={lockedScope}
        onConfirm={(skill: Skill) => {
          skillStore.add(skill);
          addChecked('enterpriseSkill', skill.id);
          setLibTick((v) => v + 1);
        }}
      />
      <PluginUploadDialog
        open={uploadCategory === 'enterprisePlugin'}
        onOpenChange={(o) => !o && setUploadCategory(null)}
        lockedScope={lockedScope}
        onConfirm={(plugin: Plugin) => {
          pluginStore.add(plugin);
          addChecked('enterprisePlugin', plugin.id);
          setLibTick((v) => v + 1);
        }}
      />
      <MCPAddDialog
        open={uploadCategory === 'enterpriseMcp'}
        onOpenChange={(o) => !o && setUploadCategory(null)}
        existingNames={mcpStore.getAll().map((m) => m.name)}
        lockedScope={lockedScope}
        onConfirm={(mcp: MCPService) => {
          mcpStore.add(mcp);
          addChecked('enterpriseMcp', mcp.name);
          setLibTick((v) => v + 1);
        }}
      />
      <StandardUploadDialog
        open={uploadCategory === 'enterpriseStandard'}
        onOpenChange={(o) => !o && setUploadCategory(null)}
        lockedScope={lockedScope}
        onConfirm={(asset: AgentConfigAsset) => {
          standardsStore.add(asset);
          addChecked('enterpriseStandard', asset.id);
          setLibTick((v) => v + 1);
        }}
      />
    </>
  );
}

function emptyCheckedMap(): CheckedMap {
  return {
    modelConfig: new Set(),
    publicSkill: new Set(),
    enterpriseSkill: new Set(),
    enterprisePlugin: new Set(),
    enterpriseMcp: new Set(),
    enterpriseStandard: new Set(),
  };
}
