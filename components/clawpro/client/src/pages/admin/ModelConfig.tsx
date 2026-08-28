/**
 * ModelConfig - 管控端模型配置页
 * Design: 「流动蓝图」Fluid Blueprint - Admin Side
 *
 * 长度治理：各 Dialog 与工具条拆到 ./ModelConfig/* 子目录，页面只保留列表骨架。
 */
import { useState, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableActionCell } from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Pagination } from "@/components/ui/pagination";
import { Badge } from "@/components/ui/badge";
import {
  PanelTitle,
  BodyText,
  BodyMedium,
  CardTitle,
  MetaText,
  InlineNumber,
  UrlText,
} from "@/components/ui/Typography";
import { SurfaceCard } from "@/components/ui/Surface";
import {
  NumberCard,
  RequestsIcon,
  InputTokensIcon,
  OutputTokensIcon,
  TotalTokensIcon,
} from "@/components/ui/number-card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { toast } from "sonner";
import { Plus, Info, Pencil, Search } from "lucide-react";
import type { UserGroup } from "./MemberManagement/types";
import {
  MOCK_GROUPS as MOCK_ONEID_GROUPS,
  MOCK_MANUAL_GROUPS,
  MOCK_PROJECTS,
} from "./MemberManagement/mock";
import {
  CUSTOM_PROVIDER_VALUE,
  setDefaultModelStorage,
  useAdminModelsState,
  type ModelRow,
} from "@/lib/modelConfigStore";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { ScopePopover } from "./ModelConfig/ScopePopover";
import { EditQuotaDialog } from "./ModelConfig/EditQuotaDialog";
import { DeleteModelDialog } from "./ModelConfig/DeleteModelDialog";
import {
  MultimodalConfirmDialog,
  type MultimodalConfirmState,
} from "./ModelConfig/MultimodalConfirmDialog";
import { EditModelDialog } from "./ModelConfig/EditModelDialog";
import { AddModelDialog } from "./ModelConfig/AddModelDialog";
import { RouteStrategyPanel } from "./ModelConfig/RouteStrategyPanel";
import { useModelConfigPortalBillingExempt } from "./ModelConfig/useModelConfigPortalBillingExempt";

// 模型配置页不区分 OneID/普通模式，合并展示所有组织；项目独立传入 ScopeSelect，
// 与企业工具库的「按组织/项目」范围选择保持一致。
const ALL_GROUPS: UserGroup[] = [...MOCK_ONEID_GROUPS, ...MOCK_MANUAL_GROUPS];
const ALL_PROJECTS: UserGroup[] = MOCK_PROJECTS;

// Tab 状态值（仅前端 UI 切换用）
const MODEL_TAB_VALUE = "models";
const ROUTE_TAB_VALUE = "routing";

// 模型列表分页每页条数（对齐 Agent 列表规范）
const PAGE_SIZE = 10;

// 厂商筛选「全部」选项值
const ALL_PROVIDERS_VALUE = "__all__";

export default function ModelConfig() {
  const [models, setModels] = useAdminModelsState();

  // 停服态下把本页面交互触发的所有 Radix Portal 浮层（SelectContent / PopoverContent /
  // DropdownMenuContent / DialogContent / DrawerContent / SheetContent / TooltipContent 等）
  // 补打 data-billing-exempt，让"下拉展开的选项面板/ 弹窗 / Popover 内容"也一并恢复为正常态。
  // 触发器本身已通过就近的 data-billing-exempt 打标豁免（TabsList / SelectTrigger 等）；
  // 详见 ./ModelConfig/useModelConfigPortalBillingExempt.ts 头部注释（作用域 / 幂等 /
  // 延续原生 disabled 的保证）。
  useModelConfigPortalBillingExempt();
  const [showAddDialog, setShowAddDialog] = useState(false);
  // 删除二次确认弹窗
  const [deleteConfirmModel, setDeleteConfirmModel] = useState<ModelRow | null>(null);

  // 编辑模型弹窗
  const [editModel, setEditModel] = useState<ModelRow | null>(null);
  const [showEditDialog, setShowEditDialog] = useState(false);

  // 编辑配额弹窗
  const [editQuotaModel, setEditQuotaModel] = useState<ModelRow | null>(null);
  const [showEditQuota, setShowEditQuota] = useState(false);

  // 多模态切换二次确认弹窗
  const [multimodalConfirm, setMultimodalConfirm] = useState<MultimodalConfirmState | null>(null);

  // ─── 检索 & 分页状态 ───────────────────────────────────────────────────────
  // 关键词：按模型名称 / 版本模糊匹配
  const [keyword, setKeyword] = useState("");
  // 厂商筛选：按模型厂商名精确匹配（__all__ 表示全部厂商）
  const [providerFilter, setProviderFilter] = useState<string>(ALL_PROVIDERS_VALUE);
  // 当前页码（从 1 开始）
  const [page, setPage] = useState(1);

  // 厂商下拉选项：取非自定义模型中出现的厂商名去重，按中文排序；「自定义模型」固定置底
  const providerOptions = useMemo(() => {
    const names = Array.from(
      new Set(models.filter((m) => m.provider !== CUSTOM_PROVIDER_VALUE).map((m) => m.name)),
    );
    return names.sort((a, b) => a.localeCompare(b, "zh-Hans-CN"));
  }, [models]);

  const selectedProviderLabel =
    providerFilter === ALL_PROVIDERS_VALUE
      ? "全部厂商"
      : providerFilter === CUSTOM_PROVIDER_VALUE
        ? "自定义模型"
        : providerFilter;

  // 按厂商 + 关键词（名称 / 版本）过滤
  const filteredModels = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    return models.filter((m) => {
      if (providerFilter === CUSTOM_PROVIDER_VALUE && m.provider !== CUSTOM_PROVIDER_VALUE) return false;
      if (
        providerFilter !== ALL_PROVIDERS_VALUE &&
        providerFilter !== CUSTOM_PROVIDER_VALUE &&
        m.name !== providerFilter
      ) {
        return false;
      }
      if (kw) {
        const haystack = `${m.name} ${m.version}`.toLowerCase();
        if (!haystack.includes(kw)) return false;
      }
      return true;
    });
  }, [models, providerFilter, keyword]);

  // 分页：切片当前页数据（页码越界时回退到有效范围）
  const totalPages = Math.max(1, Math.ceil(filteredModels.length / PAGE_SIZE));
  const safePage = Math.min(page, totalPages);
  const paginatedModels = useMemo(
    () => filteredModels.slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE),
    [filteredModels, safePage],
  );

  const openAddDialog = () => setShowAddDialog(true);

  // 筛选条件变化时回到第一页
  const handleKeywordChange = (value: string) => {
    setKeyword(value);
    setPage(1);
  };
  const handleProviderChange = (value: string) => {
    setProviderFilter(value);
    setPage(1);
  };

  const openEditQuota = (model: ModelRow) => {
    setEditQuotaModel(model);
    setShowEditQuota(true);
  };

  const openEditDialog = (model: ModelRow) => {
    setEditModel(model);
    setShowEditDialog(true);
  };

  // 设置默认模型：单选，同时将其他模型的 isDefault 置为 false
  // 仅允许对「用户可见」的模型设为默认
  const handleSetDefault = (id: string, enable: boolean) => {
    const target = models.find((m) => m.id === id);
    if (!target) return;
    if (enable) {
      if (target.userProvidedKey) {
        toast.error("该模型由用户端自行填写 Key，不支持设为默认");
        return;
      }
      if (!target.visible) {
        toast.error("请先开启该模型的「用户可见」后再设为默认");
        return;
      }
      const updated = models.map((m) => ({ ...m, isDefault: m.id === id }));
      setModels(updated);
      setDefaultModelStorage(id);
      toast.success(`已将「${target.name} · ${target.version}」设为默认模型`, {
        description: "授权范围内的新建 Agent 将默认应用该模型。",
      });
    } else {
      const updated = models.map((m) => ({ ...m, isDefault: false }));
      setModels(updated);
      setDefaultModelStorage(null);
      toast.success("已取消默认模型");
    }
  };

  // 切换多模态属性（仅自定义模型）
  const handleToggleMultimodal = (id: string, value: boolean) => {
    const target = models.find((m) => m.id === id);
    if (!target) return;
    setModels(models.map((m) => m.id === id ? { ...m, isMultimodal: value } : m));
    toast.success(value ? `已为「${target.name}」开启多模态` : `已为「${target.name}」关闭多模态`);
  };

  // 当「用户可见」关闭时，若该模型是默认模型则自动取消默认
  const handleToggleVisible = (id: string, visible: boolean) => {
    const target = models.find((m) => m.id === id);
    if (!target) return;
    let updated = models.map((m) => m.id === id ? { ...m, visible } : m);
    if (!visible && target.isDefault) {
      updated = updated.map((m) => m.id === id ? { ...m, isDefault: false } : m);
      setDefaultModelStorage(null);
      toast.warning(`「${target.name}」已隐藏，默认模型已自动取消`);
    } else {
      toast.success(visible ? "已对用户可见" : "已对用户隐藏");
    }
    setModels(updated);
  };

  const visibleModelCount = useMemo(() => models.filter((model) => model.visible).length, [models]);
  const defaultModel = useMemo(() => models.find((model) => model.isDefault), [models]);
  const scopedModelCount = useMemo(() => models.filter((model) => model.visibilityScope === "groups").length, [models]);

  return (
    <>
      <div className="page-enter space-y-10">
        <div className="space-y-6">
          <AdminPageHeader
            title="模型配置"
            description="统一管理平台可用模型、接入地址、每日配额与应用范围；模型授权和实例同步请在资产管理中配置。"
          />

          <div className="grid grid-cols-4 gap-4">
          <NumberCard
            icon={<RequestsIcon />}
            label="已配置模型"
            value={models.length}
          />
          <NumberCard
            icon={<InputTokensIcon />}
            label="用户可见"
            value={visibleModelCount}
          />
          <NumberCard
            icon={<OutputTokensIcon />}
            label="默认模型"
            value={
              <BodyMedium as="span" className="block truncate">
                {defaultModel ? `${defaultModel.name} · ${defaultModel.version}` : "未设置"}
              </BodyMedium>
            }
          />
          <NumberCard
            icon={<TotalTokensIcon />}
            label="按组织/项目"
            value={scopedModelCount}
          />
          </div>
        </div>

        <Tabs defaultValue={MODEL_TAB_VALUE} className="w-full">
          {/* 顶部 Tab 切换
            * 停服态豁免：切换「模型列表 / 路由策略」属于查看类操作（不产生变更），
            * 与其他视图切换同档，需保持 100% 不透明与正常交互。
            * TabsTrigger 自身未设置 disabled，"停服前已禁用则延续禁用"约束
            * 通过 Radix 的 disabled 属性依然生效（此处无）。 */}
          <TabsList data-billing-exempt>
            <TabsTrigger value={MODEL_TAB_VALUE}>模型列表</TabsTrigger>
            <TabsTrigger value={ROUTE_TAB_VALUE}>路由策略</TabsTrigger>
          </TabsList>

          <TabsContent value={MODEL_TAB_VALUE} className="mt-4">
            <section className="space-y-4">
              <div>
                <PanelTitle>模型列表</PanelTitle>
                <BodyText as="p" tone="muted" className="mt-1">
                  集中管理模型接入、配额、用户可见性与默认配置；资产管理负责将模型应用到对应组织、项目及实例。
                </BodyText>
              </div>

              {/* 检索工具条：按厂商筛选 + 按名称/版本关键词搜索 */}
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex flex-wrap items-center gap-3">
                  <Select value={providerFilter} onValueChange={handleProviderChange}>
                    {/* 厂商筛选下拉
                      * 停服态豁免：按厂商筛选属于查看类操作（不产生变更），
                      * 与关键词搜索同档，需保持 100% 不透明与正常交互。
                      * 页面未给 Select 传 disabled，"停服前已禁用则延续禁用"
                      * 约束通过 Radix 的 disabled 属性依然生效（此处无）。*/}
                    <SelectTrigger
                      className="w-[112px] [&>span]:min-w-0 [&>span]:truncate"
                      title={selectedProviderLabel}
                      data-billing-exempt
                    >
                      <SelectValue placeholder="全部厂商" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={ALL_PROVIDERS_VALUE}>全部厂商</SelectItem>
                      {providerOptions.map((name) => (
                        <SelectItem key={name} value={name}>
                          {name}
                        </SelectItem>
                      ))}
                      <SelectItem value={CUSTOM_PROVIDER_VALUE}>自定义模型</SelectItem>
                    </SelectContent>
                  </Select>
                  <div className="relative w-[280px]">
                    <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-[var(--text-weak)]" />
                    <Input
                      value={keyword}
                      onChange={(e) => handleKeywordChange(e.target.value)}
                      placeholder="搜索模型名称、版本"
                      className="pl-9"
                    />
                  </div>
                </div>
                <Button variant="claw-primary" size="claw-sm" onClick={openAddDialog}>
                  <Plus className="w-3.5 h-3.5" />
                  添加模型
                </Button>
              </div>

        <SurfaceCard className="overflow-hidden">
          <Table variant="white" scrollX={1400}>
            <TableHeader>
              <TableRow>
                <TableHead
                  fixed="left"
                  style={{ width: 220, minWidth: 220, maxWidth: 220 }}
                >
                  模型信息
                </TableHead>
                <TableHead className="w-[280px]">接入地址</TableHead>
                <TableHead className="w-[150px]">每日配额</TableHead>
                <TableHead className="w-[120px]">
                  <div className="flex items-center gap-1">
                    用户可见
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="cursor-default">
                          <Info className="w-3 h-3 text-[var(--text-weak)]" />
                        </span>
                      </TooltipTrigger>
                      <TooltipContent className="max-w-[240px]">
                        <MetaText tone="inherit">开启后，该模型会展示在用户端的模型选项列表中。</MetaText>
                      </TooltipContent>
                    </Tooltip>
                  </div>
                </TableHead>
                <TableHead className="w-[120px]">
                  <div className="flex items-center gap-1">
                    默认配置
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="cursor-default">
                          <Info className="w-3 h-3 text-[var(--text-weak)]" />
                        </span>
                      </TooltipTrigger>
                      <TooltipContent className="max-w-[260px]">
                        <MetaText tone="inherit">
                          在资产管理授权范围内，新建 Agent 默认应用该模型。
                        </MetaText>
                      </TooltipContent>
                    </Tooltip>
                  </div>
                </TableHead>
                <TableHead className="w-[160px]">是否启用多模态</TableHead>
                <TableHead className="w-[220px]">
                  <div className="flex items-center gap-1">
                    应用范围
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="cursor-default">
                          <Info className="w-3 h-3 text-[var(--text-weak)]" />
                        </span>
                      </TooltipTrigger>
                      <TooltipContent className="max-w-[260px]">
                        <MetaText tone="inherit">
                          应用范围决定哪些组织或项目可在资产管理中添加该模型，以及哪些用户可以看到该模型。
                        </MetaText>
                      </TooltipContent>
                    </Tooltip>
                  </div>
                </TableHead>
                <TableHead
                  fixed="right"
                  style={{ width: 120, minWidth: 120, maxWidth: 120 }}
                >
                  操作
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginatedModels.length === 0 && (
                <TableRow>
                  <TableCell colSpan={8}>
                    <div className="flex min-h-[120px] items-center justify-center">
                      <BodyText as="span" tone="muted">未找到匹配的模型</BodyText>
                    </div>
                  </TableCell>
                </TableRow>
              )}
              {paginatedModels.map((model) => {
                const canToggleMultimodal = model.provider === CUSTOM_PROVIDER_VALUE;

                return (
                  <TableRow key={model.id}>
                    <TableCell fixed="left" style={{ width: 220, minWidth: 220, maxWidth: 220 }}>
                      <div className="min-w-0 space-y-2">
                        <div className="flex min-w-0 items-center gap-2">
                          <CardTitle as="p" className="truncate">{model.name}</CardTitle>
                          {model.isDefault && <Badge color="blue">默认</Badge>}
                          {model.userProvidedKey && <Badge color="purple">用户端自备Key</Badge>}
                        </div>
                        <MetaText as="p" className="truncate">{model.version}</MetaText>
                      </div>
                    </TableCell>
                    <TableCell className="w-[280px]">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <UrlText className="block max-w-[248px] cursor-default truncate">
                            {model.modelUrl}
                          </UrlText>
                        </TooltipTrigger>
                        <TooltipContent className="max-w-[360px]">
                          <UrlText tone="inherit">{model.modelUrl}</UrlText>
                        </TooltipContent>
                      </Tooltip>
                    </TableCell>
                    <TableCell className="w-[150px]">
                      {model.userProvidedKey ? (
                        <MetaText tone="weak">无限制</MetaText>
                      ) : (
                        <div className="flex items-center gap-1.5">
                          <InlineNumber tone="emphasis">{model.dailyLimit.toLocaleString()}</InlineNumber>
                          <button
                            type="button"
                            className="inline-flex items-center text-[var(--text-weak)] transition-colors hover:text-[var(--text-brand)]"
                            onClick={() => openEditQuota(model)}
                            aria-label="编辑每日配额"
                          >
                            <Pencil className="w-3 h-3" />
                          </button>
                        </div>
                      )}
                    </TableCell>
                    <TableCell className="w-[120px]">
                      <div className="flex min-h-9 items-center">
                        <Switch
                          checked={model.visible}
                          onCheckedChange={(v) => handleToggleVisible(model.id, v)}
                          aria-label={model.visible ? "关闭用户可见" : "开启用户可见"}
                        />
                      </div>
                    </TableCell>
                    <TableCell className="w-[120px]">
                      <div className="flex min-h-9 items-center">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span className="inline-flex">
                              <Switch
                                checked={model.isDefault}
                                onCheckedChange={(v) => handleSetDefault(model.id, v)}
                                disabled={(!model.visible && !model.isDefault) || model.userProvidedKey}
                                aria-label={model.isDefault ? "当前默认模型" : "设为默认模型"}
                              />
                            </span>
                          </TooltipTrigger>
                          <TooltipContent>
                            <MetaText tone="inherit">
                              {model.isDefault
                                ? "当前默认模型"
                                : model.userProvidedKey
                                  ? "该模型由用户端自行填写 Key，不支持设为默认"
                                  : model.visible
                                    ? "点击设为默认模型"
                                    : "需先开启「用户可见」才可设为默认"}
                            </MetaText>
                          </TooltipContent>
                        </Tooltip>
                      </div>
                    </TableCell>
                    <TableCell className="w-[160px]">
                      <div className="flex min-h-9 items-center">
                        {canToggleMultimodal ? (
                          <Switch
                            checked={model.isMultimodal}
                            onCheckedChange={(value) => setMultimodalConfirm({ model, enable: value })}
                            aria-label={model.isMultimodal ? "关闭多模态" : "开启多模态"}
                          />
                        ) : (
                          <BodyText as="span" tone="muted">-</BodyText>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="w-[220px]">
                      <ScopePopover
                        model={model}
                        groups={ALL_GROUPS}
                        projects={ALL_PROJECTS}
                        onSave={(id, scope, groupIds) => {
                          setModels((prev) =>
                            prev.map((m) =>
                              m.id === id ? { ...m, visibilityScope: scope, visibilityGroupIds: groupIds } : m
                            )
                          );
                        }}
                      />
                    </TableCell>
                    <TableActionCell
                      fixed="right"
                      style={{ width: 120, minWidth: 120, maxWidth: 120 }}
                      actionsClassName="justify-start"
                    >
                      <Button variant="link" size="sm" onClick={() => openEditDialog(model)}>
                        编辑
                      </Button>
                      <Button variant="link" size="sm" onClick={() => setDeleteConfirmModel(model)}>
                        删除
                      </Button>
                    </TableActionCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </SurfaceCard>

        <Pagination
          total={filteredModels.length}
          current={safePage}
          pageSize={PAGE_SIZE}
          showTotal={(total) => `共 ${total} 条记录`}
          className="w-full justify-between"
          hideOnSinglePage
          onChange={(p) => setPage(p)}
        />
        </section>
          </TabsContent>

          <TabsContent value={ROUTE_TAB_VALUE} className="mt-4">
            <RouteStrategyPanel models={models} groups={ALL_GROUPS} />
          </TabsContent>
        </Tabs>


      </div>

      <AddModelDialog open={showAddDialog} onOpenChange={setShowAddDialog} />

      {/* Delete Confirm Dialog */}
      <DeleteModelDialog
        model={deleteConfirmModel}
        onClose={() => setDeleteConfirmModel(null)}
        onConfirm={(model) => {
          if (model.isDefault) {
            setDefaultModelStorage(null);
          }
          setModels(models.filter((m) => m.id !== model.id));
          setDeleteConfirmModel(null);
          toast.success("模型已删除");
        }}
      />

      {/* Edit Quota Dialog */}
      <EditQuotaDialog
        model={editQuotaModel}
        open={showEditQuota}
        onClose={() => setShowEditQuota(false)}
        onSave={(id, limit) => {
          setModels(models.map((m) => m.id === id ? { ...m, dailyLimit: limit } : m));
          toast.success("配额已更新");
        }}
      />

      {/* 多模态切换二次确认弹窗（与删除模型 AlertDialog 保持一致语言） */}
      <MultimodalConfirmDialog
        state={multimodalConfirm}
        onClose={() => setMultimodalConfirm(null)}
        onConfirm={(state) => {
          handleToggleMultimodal(state.model.id, state.enable);
          setMultimodalConfirm(null);
        }}
      />

      {/* 编辑模型弹窗 */}
      <EditModelDialog
        model={editModel}
        open={showEditDialog}
        onClose={() => setShowEditDialog(false)}
        onSave={(id, updates) => {
          setModels(models.map((m) => m.id === id ? { ...m, ...updates } : m));
          toast.success("模型已更新");
        }}
      />
    </>
  );
}
