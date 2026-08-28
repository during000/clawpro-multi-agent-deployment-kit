/**
 * ComponentPreview - 筛选面板组件预览页（按数据结构同构性分类）
 *
 * 经过 v2 重构后的最终分类：
 *   ┌──────────────────────────────────────────────────────────────────────────┐
 *   │ 可合并组（数据同构 → 已合并为同一组件的变体）                              │
 *   │  1. Select        — simple / searchable / filter-multi                  │
 *   │     文件：ui/select.tsx                                                  │
 *   │  2. TreeSelect    — button / filter-icon                                │
 *   │     文件：ui/tree-select.tsx                                             │
 *   │  3. ScopeSelect   — instant / confirm                                   │
 *   │     文件：components/ScopeSelect.tsx                                     │
 *   ├──────────────────────────────────────────────────────────────────────────┤
 *   │ 独立组件（数据异构/业务特化）                                              │
 *   │  4. GroupSelect       — source 分桶 + 聚合算法                           │
 *   │  5. TokenValueEditor  — mode + 数值输入（非列表）                         │
 *   │  6. ActionsMenu       — DropdownMenu + MoreActionsDropdown              │
 *   ├──────────────────────────────────────────────────────────────────────────┤
 *   │ 底层骨架                                                                 │
 *   │  A. SelectPanel    — 面板三段式骨架                                      │
 *   │  B. FilterTrigger  — 触发器统一组件                                      │
 *   └──────────────────────────────────────────────────────────────────────────┘
 *
 * 访问路径：/filter-panel-preview
 */
import { useState } from "react";
import { Check, ChevronDown, Download, MoreHorizontal, RefreshCw, Search as SearchIcon, Terminal, Trash2 } from "lucide-react";
import { SelectPanel, SelectPanelItem } from "@/components/ui/select-panel";
import { FilterTrigger } from "@/components/ui/filter-trigger";
import {
  Select, SelectTrigger, SelectValue, SelectContent, SelectItem,
  SearchableSelect, type SearchableSelectOption,
  FilterMultiSelect,
  InstantMultiSelect, type InstantMultiSelectSection, type InstantMultiSelectOption,
} from "@/components/ui/select";
import { TreeSelect, type TreeFilterNode } from "@/components/ui/tree-select";
import { ScopeSelect, type ScopeFilterGroup, type ScopeGroup, type ScopeType } from "@/components/ScopeSelect";
import { GroupSelect } from "@/components/GroupSelect";
import { TokenValueEditor } from "@/components/policy/TokenValueEditor";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { MoreActionsDropdown } from "@/components/ui/more-actions-dropdown";
import { Button } from "@/components/ui/button";
import { MetaText, BodyText } from "@/components/ui/Typography";

// ─── Demo 数据 ───────────────────────────────────────────────────────────────

const DEMO_OPTIONS: SearchableSelectOption[] = [
  { value: "vpc-001", label: "vpc-default（默认）" },
  { value: "vpc-002", label: "vpc-production" },
  { value: "vpc-003", label: "vpc-staging" },
  { value: "vpc-004", label: "vpc-dev-team-alpha" },
  { value: "vpc-005", label: "vpc-isolated" },
];

const DEMO_TREE: TreeFilterNode[] = [
  { id: "dept-root", name: "A公司", children: [
    { id: "dept-tech", name: "技术部", children: [
      { id: "dept-fe", name: "前端组" },
      { id: "dept-be", name: "后端组" },
    ]},
    { id: "dept-product", name: "产品部" },
  ]},
];

const DEMO_FILTER_OPTIONS = [
  { value: "running", label: "运行中" },
  { value: "stopped", label: "已停止" },
  { value: "creating", label: "创建中" },
  { value: "error", label: "异常" },
];

// InstantMultiSelect 组织数据
const DEMO_INSTANT_SECTIONS: InstantMultiSelectSection[] = [
  {
    label: "环境",
    options: [
      { value: "prod", label: "生产环境" },
      { value: "staging", label: "预发环境" },
      { value: "dev", label: "开发环境" },
    ],
  },
  {
    label: "状态",
    options: [
      { value: "running", label: "运行中" },
      { value: "stopped", label: "已停止" },
      { value: "creating", label: "创建中" },
      { value: "error", label: "异常", disabled: true },
    ],
  },
];

const DEMO_SCOPE_GROUPS: ScopeFilterGroup[] = [
  { id: "grp-fe", name: "前端组" },
  { id: "grp-be", name: "后端组" },
  { id: "grp-qa", name: "测试组" },
];

const DEMO_SCOPE_EDIT_GROUPS: ScopeGroup[] = [
  { id: "g1", name: "研发部", parentId: null },
  { id: "g2", name: "前端组", parentId: "g1" },
  { id: "g3", name: "后端组", parentId: "g1" },
  { id: "g4", name: "产品部", parentId: null },
];

const _NOW = new Date().toISOString();
const DEMO_GROUP_SELECT_DATA = [
  { id: "g1", name: "研发部", parentId: null as string | null, source: "manual" as const, readonly: false, createdAt: _NOW },
  { id: "g2", name: "前端组", parentId: "g1" as string | null, source: "manual" as const, readonly: false, createdAt: _NOW },
  { id: "g3", name: "后端组", parentId: "g1" as string | null, source: "manual" as const, readonly: false, createdAt: _NOW },
  { id: "g4", name: "产品部", parentId: null as string | null, source: "manual" as const, readonly: false, createdAt: _NOW },
];

// ─── 工具组件 ────────────────────────────────────────────────────────────────

interface VariantCardProps {
  variant: string;
  desc: string;
  children: React.ReactNode;
  api: string;
  status?: "merge" | "independent" | "skeleton";
  width?: string;
}

function VariantCard({ variant, desc, children, api, status = "merge", width = "w-64" }: VariantCardProps) {
  const statusColors = {
    merge: "bg-blue-50 text-blue-600",
    independent: "bg-amber-50 text-amber-700",
    skeleton: "bg-gray-100 text-gray-600",
  };
  return (
    <div className="space-y-2">
      <div className="flex items-baseline gap-2 flex-wrap">
        <code className={`text-[12px] font-semibold px-1.5 py-0.5 rounded ${statusColors[status]}`}>{variant}</code>
        <MetaText tone="weak" className="text-[12px]">{desc}</MetaText>
      </div>
      <div className={width}>{children}</div>
      <MetaText tone="weak" className="text-[11px]">API：<code className="text-[11px]">{api}</code></MetaText>
    </div>
  );
}

function SectionHeader({ index, title, badge, summary }: { index: string; title: string; badge: string; summary: string }) {
  const badgeColors: Record<string, string> = {
    "可合并": "bg-blue-100 text-blue-700 border-blue-200",
    "独立组件": "bg-amber-100 text-amber-700 border-amber-200",
    "底层骨架": "bg-gray-100 text-gray-600 border-gray-200",
  };
  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-2.5">
        <h3 className="text-base font-semibold text-[var(--text-title)]">
          <span className="text-blue-600 mr-1.5">{index}</span>{title}
        </h3>
        <span className={`text-[11px] font-medium px-2 py-0.5 rounded border ${badgeColors[badge] || "bg-gray-100 text-gray-600 border-gray-200"}`}>
          {badge}
        </span>
      </div>
      <MetaText tone="weak">{summary}</MetaText>
    </div>
  );
}

// ─── 主组件 ──────────────────────────────────────────────────────────────────

export default function ComponentPreview() {
  const [selectVal, setSelectVal] = useState("");
  const [searchableVal, setSearchableVal] = useState("");
  const [treeVal, setTreeVal] = useState("");
  const [treeSelectVal, setTreeSelectVal] = useState("");
  const [filterVals, setFilterVals] = useState<Set<string>>(new Set(["running", "stopped"]));
  const [instantMultiFlatVals, setInstantMultiFlatVals] = useState<Set<string>>(new Set(["vpc-002"]));
  const [instantMultiGroupVals, setInstantMultiGroupVals] = useState<Set<string>>(new Set(["prod", "running"]));
  const [panelSearch, setPanelSearch] = useState("");
  const [panelSelected, setPanelSelected] = useState("vpc-002");
  const [instantSelected, setInstantSelected] = useState("");
  const [scopeKeys, setScopeKeys] = useState<Set<string>>(new Set(["public"]));
  const [scopeEditScope, setScopeEditScope] = useState<ScopeType>("all");
  const [scopeEditIds, setScopeEditIds] = useState<string[]>([]);
  const [groupSelectIds, setGroupSelectIds] = useState<string[]>([]);
  const [tokenMode, setTokenMode] = useState<"custom" | "unlimited">("unlimited");
  const [tokenVal, setTokenVal] = useState("");

  return (
    <div className="p-8 max-w-6xl mx-auto space-y-16">
      {/* ── Header ── */}
      <div>
        <h1 className="text-2xl font-bold text-[var(--text-title)] mb-1">筛选面板组件预览</h1>
        <BodyText tone="secondary">
          按<strong>数据结构同构性</strong>分类——同构的合并为变体，异构的保持独立。
        </BodyText>
        <div className="mt-3 flex items-center gap-2 flex-wrap text-[12px]">
          <span className="px-2 py-0.5 rounded bg-blue-50 text-blue-600 font-medium border border-blue-200">可合并 × 3 组</span>
          <span className="px-2 py-0.5 rounded bg-amber-50 text-amber-700 font-medium border border-amber-200">独立 × 3 组件</span>
          <span className="px-2 py-0.5 rounded bg-gray-100 text-gray-600 font-medium border border-gray-200">骨架 × 2</span>
        </div>
      </div>

      {/* ════════════════════════════════════════════════════════════════════════
          第一部分：可合并组（数据结构同构）
         ════════════════════════════════════════════════════════════════════════ */}
      <div className="space-y-12">
        <h2 className="text-xl font-bold text-[var(--text-title)] border-b-2 border-blue-600 pb-2">
          可合并组（数据结构同构 → 统一为同一组件的变体）
        </h2>

        {/* ─── 1. Select ─── */}
        <section className="space-y-5">
          <SectionHeader
            index="1."
            title="Select（扁平 Option[] 列表）"
            badge="可合并"
            summary="数据结构：{ value: string; label: string }[]。差异仅在「搜索 / 单多选 / 触发器」→ 同一文件 ui/select.tsx 中的不同导出。"
          />
          <div className="flex items-start gap-x-10 gap-y-6 flex-wrap">
            <VariantCard variant="simple" desc="短列表静态单选" api="<Select> + <SelectItem>" width="w-48">
              <Select value={selectVal} onValueChange={setSelectVal}>
                <SelectTrigger><SelectValue placeholder="选择状态" /></SelectTrigger>
                <SelectContent>{DEMO_FILTER_OPTIONS.map((o) => <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>)}</SelectContent>
              </Select>
            </VariantCard>

            <VariantCard variant="searchable" desc="长列表 + 搜索过滤" api="<SearchableSelect>">
              <SearchableSelect options={DEMO_OPTIONS} value={searchableVal} onChange={setSearchableVal} placeholder="选择 VPC" searchPlaceholder="搜索 VPC..." />
            </VariantCard>

            <VariantCard variant="filter-multi" desc="表头多选 + confirm 提交" api="<FilterMultiSelect>" width="w-auto">
              <div className="flex items-center gap-4 h-10 px-4 bg-[var(--bg-grey-normal)] rounded-[4px]">
                <FilterMultiSelect title="状态" options={DEMO_FILTER_OPTIONS} value={filterVals} onChange={setFilterVals} />
                <MetaText tone="weak">模拟表头</MetaText>
              </div>
            </VariantCard>

            <VariantCard variant="instant-multi（扁平）" desc="下拉框触发 + 即时多选" api="<InstantMultiSelect options={...}>" width="w-64">
              <InstantMultiSelect
                options={DEMO_OPTIONS as InstantMultiSelectOption[]}
                value={instantMultiFlatVals}
                onChange={setInstantMultiFlatVals}
                placeholder="选择 VPC（可多选）"
                searchPlaceholder="搜索 VPC..."
              />
            </VariantCard>

            <VariantCard variant="instant-multi（组织）" desc="分段标题 + 三态全选 + 即时多选" api="<InstantMultiSelect sections={...}>" width="w-64">
              <InstantMultiSelect
                sections={DEMO_INSTANT_SECTIONS}
                value={instantMultiGroupVals}
                onChange={setInstantMultiGroupVals}
                placeholder="选择环境与状态"
                searchPlaceholder="搜索..."
              />
            </VariantCard>
          </div>
        </section>

        {/* ─── 2. TreeSelect ─── */}
        <section className="space-y-5">
          <SectionHeader
            index="2."
            title="TreeSelect（树形 TreeNode[] 列表）"
            badge="可合并"
            summary='数据结构：{ id, name, children?, path? }[]。通过 triggerVariant 区分两种触发器。'
          />
          <div className="flex items-start gap-x-10 gap-y-6 flex-wrap">
            <VariantCard variant="button（默认）" desc="toolbar 按钮触发 + confirm" api='<TreeSelect triggerVariant="button">' width="w-56">
              <TreeSelect nodes={DEMO_TREE} value={treeSelectVal} onChange={setTreeSelectVal} allLabel="全部部门" searchPlaceholder="搜索部门" />
            </VariantCard>

            <VariantCard variant="filter-icon" desc="表头漏斗图标触发 + confirm" api='<TreeSelect triggerVariant="filter-icon" title="...">' width="w-auto">
              <div className="flex items-center gap-4 h-10 px-4 bg-[var(--bg-grey-normal)] rounded-[4px]">
                <TreeSelect triggerVariant="filter-icon" title="部门" nodes={DEMO_TREE} value={treeVal} onChange={setTreeVal} allLabel="全部部门" searchPlaceholder="搜索部门" />
                <MetaText tone="weak">模拟表头</MetaText>
              </div>
            </VariantCard>
          </div>
        </section>

        {/* ─── 3. ScopeSelect ─── */}
        <section className="space-y-5">
          <SectionHeader
            index="3."
            title="ScopeSelect（范围选择面板）"
            badge="可合并"
            summary={'数据结构：{ id, name, parentId? }[]，含"全部用户"+"按组织"两段式。通过 mode 区分提交模式。'}
          />
          <div className="flex items-start gap-x-10 gap-y-6 flex-wrap">
            <VariantCard variant="instant（默认）" desc="嵌入式即时多选（无触发器包裹）" api='<ScopeSelect mode="instant">' width="w-56">
              <ScopeSelect groups={DEMO_SCOPE_GROUPS} value={scopeKeys} onChange={setScopeKeys} searchPlaceholder="搜索组织" />
            </VariantCard>

            <VariantCard variant="instant + withTrigger" desc="即时多选 + Popover 触发器（Portal 不被裁切）" api='<ScopeSelect withTrigger groups=... value=... onChange=...>' width="w-auto">
              <ScopeSelect
                withTrigger
                groups={DEMO_SCOPE_GROUPS}
                value={scopeKeys}
                onChange={setScopeKeys}
                searchPlaceholder="搜索组织"
                triggerPlaceholder="选择组织"
                align="start"
              />
            </VariantCard>

            <VariantCard variant="confirm" desc="badge-pencil 触发 + segment + 确认" api='<ScopeSelect mode="confirm" scope=...>' width="w-auto">
              <div className="flex items-center gap-2">
                <MetaText>应用范围：</MetaText>
                <ScopeSelect
                  scope={scopeEditScope}
                  selectedGroupIds={scopeEditIds}
                  groups={DEMO_SCOPE_EDIT_GROUPS}
                  onConfirm={(s, ids) => { setScopeEditScope(s); setScopeEditIds(ids); }}
                />
              </div>
            </VariantCard>
          </div>
        </section>
      </div>

      {/* ════════════════════════════════════════════════════════════════════════
          第二部分：独立组件
         ════════════════════════════════════════════════════════════════════════ */}
      <div className="space-y-12">
        <h2 className="text-xl font-bold text-[var(--text-title)] border-b-2 border-amber-500 pb-2">
          独立组件（数据结构异构 → 不合并）
        </h2>

        <section className="space-y-5">
          <SectionHeader
            index="4."
            title="GroupSelect"
            badge="独立组件"
            summary="UserGroup[] + source 多桶组织 + parentId 树 + 自动聚合/展开算法。数据结构远超简单 TreeNode，不适合合并。"
          />
          <div className="flex items-start gap-x-10 gap-y-6 flex-wrap">
            <VariantCard variant="default" desc="纯选择面板（标签 + 清除）" api="<GroupSelect>" status="independent" width="w-72">
              <GroupSelect groups={DEMO_GROUP_SELECT_DATA} selectedIds={groupSelectIds} onChange={setGroupSelectIds} placeholder="选择组织" sourceFilter={["manual"]} />
            </VariantCard>
          </div>
        </section>

        <section className="space-y-5">
          <SectionHeader
            index="5."
            title="TokenValueEditor"
            badge="独立组件"
            summary="mode('custom'|'unlimited') + valStr 数值输入。非列表选择组件。"
          />
          <div className="flex items-start gap-x-10 gap-y-6 flex-wrap">
            <VariantCard variant="—" desc="SegmentGroup + 数值 Input + confirm" api="<TokenValueEditor>" status="independent" width="w-56">
              <div className="flex items-center gap-4">
                <TokenValueEditor mode={tokenMode} valStr={tokenVal} onCommit={(m, v) => { setTokenMode(m); setTokenVal(v); }} />
                <MetaText tone="weak">{tokenMode === "unlimited" ? "无限制" : tokenVal || "未设置"}</MetaText>
              </div>
            </VariantCard>
          </div>
        </section>

        <section className="space-y-5">
          <SectionHeader
            index="6."
            title="ActionsMenu（DropdownMenu + MoreActionsDropdown）"
            badge="独立组件"
            summary="命令菜单（MenuItem[] 含 icon + onClick + variant），非筛选/选择组件。"
          />
          <div className="flex items-start gap-x-10 gap-y-6 flex-wrap">
            <VariantCard variant="more-icon" desc="强制图标的三点菜单" api="<MoreActionsDropdown>" status="independent" width="w-auto">
              <MoreActionsDropdown items={[
                { label: "安全检测", icon: SearchIcon, onClick: () => {} },
                { label: "下载", icon: Download, onClick: () => {} },
                { label: "删除", icon: Trash2, onClick: () => {}, variant: "destructive" },
              ]} />
            </VariantCard>

            <VariantCard variant="more-text" desc="文字触发" api='<MoreActionsDropdown triggerType="text">' status="independent" width="w-auto">
              <MoreActionsDropdown triggerType="text" items={[
                { label: "重启", icon: RefreshCw, onClick: () => {} },
                { label: "打开终端", icon: Terminal, onClick: () => {} },
                { label: "删除", icon: Trash2, onClick: () => {}, variant: "destructive", separatorBefore: true },
              ]} />
            </VariantCard>

            <VariantCard variant="button-trigger" desc="按钮+箭头触发" api="<DropdownMenu>" status="independent" width="w-auto">
              <DropdownMenu>
                <DropdownMenuTrigger asChild><Button variant="claw-outline" size="claw-sm">更多操作 <ChevronDown className="w-3.5 h-3.5 ml-1" /></Button></DropdownMenuTrigger>
                <DropdownMenuContent align="start">
                  <DropdownMenuItem>批量隔离</DropdownMenuItem>
                  <DropdownMenuItem variant="destructive">批量删除</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </VariantCard>

            <VariantCard variant="icon-trigger" desc="三点图标触发" api="<DropdownMenu>" status="independent" width="w-auto">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <button className="w-8 h-8 rounded-[4px] flex items-center justify-center text-gray-500 hover:text-gray-900 hover:bg-[#F5F5F5] transition-colors"><MoreHorizontal className="w-4 h-4" /></button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start">
                  <DropdownMenuItem>编辑</DropdownMenuItem>
                  <DropdownMenuItem>复制</DropdownMenuItem>
                  <DropdownMenuItem variant="destructive">删除</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </VariantCard>
          </div>
        </section>
      </div>

      {/* ════════════════════════════════════════════════════════════════════════
          第三部分：底层骨架
         ════════════════════════════════════════════════════════════════════════ */}
      <div className="space-y-10">
        <h2 className="text-xl font-bold text-[var(--text-title)] border-b-2 border-gray-400 pb-2">
          底层骨架（组合积木，不参与合并）
        </h2>

        <section className="space-y-5">
          <SectionHeader index="A." title="SelectPanel（面板三段式骨架）" badge="底层骨架" summary="搜索 + 列表 + footer 的可组合壳子。" />
          <div className="flex items-start gap-x-10 gap-y-6 flex-wrap">
            <VariantCard variant="confirm" desc="底部确认/取消" api='<SelectPanel commitMode="confirm">' status="skeleton" width="w-[280px]">
              <div className="rounded-[4px] shadow-[var(--shadow-popover)]">
                <SelectPanel commitMode="confirm" searchPlaceholder="搜索 VPC" searchValue={panelSearch} onSearchChange={setPanelSearch} footerLeft={<MetaText>已选: {panelSelected || "无"}</MetaText>} onConfirm={() => {}} onCancel={() => setPanelSelected("")}>
                  {DEMO_OPTIONS.filter((o) => !panelSearch || String(o.label).toLowerCase().includes(panelSearch.toLowerCase())).map((opt) => (
                    <SelectPanelItem key={opt.value} selected={panelSelected === opt.value} onClick={() => setPanelSelected(opt.value)}>
                      <span className="flex-1 truncate">{opt.label}</span>
                      {panelSelected === opt.value && <span className="absolute right-3"><Check className="size-4 text-blue-500" /></span>}
                    </SelectPanelItem>
                  ))}
                </SelectPanel>
              </div>
            </VariantCard>

            <VariantCard variant="instant" desc="点击即生效" api='<SelectPanel commitMode="instant">' status="skeleton" width="w-[220px]">
              <div className="rounded-[4px] shadow-[var(--shadow-popover)]">
                <SelectPanel commitMode="instant" showSearch={false} showFooter={false} maxHeight={220}>
                  {DEMO_FILTER_OPTIONS.map((opt) => (
                    <SelectPanelItem key={opt.value} selected={instantSelected === opt.value} onClick={() => setInstantSelected(opt.value)}>
                      <span className="flex-1">{opt.label}</span>
                      {instantSelected === opt.value && <span className="absolute right-3"><Check className="size-4 text-blue-500" /></span>}
                    </SelectPanelItem>
                  ))}
                </SelectPanel>
              </div>
            </VariantCard>
          </div>
        </section>

        <section className="space-y-5">
          <SectionHeader index="B." title="FilterTrigger（触发器统一组件）" badge="底层骨架" summary="所有面板的触发器入口。" />
          <div className="flex items-start gap-x-10 gap-y-6 flex-wrap">
            <VariantCard variant="button" desc="Input 式触发器" api='<FilterTrigger variant="button">' status="skeleton" width="w-auto">
              <div className="flex items-center gap-3">
                <div className="w-48"><FilterTrigger variant="button" label="vpc-production" active /></div>
                <div className="w-48"><FilterTrigger variant="button" placeholder="请选择 VPC" /></div>
              </div>
            </VariantCard>

            <VariantCard variant="icon" desc="表头漏斗图标" api='<FilterTrigger variant="icon">' status="skeleton" width="w-auto">
              <div className="flex items-center gap-3">
                <FilterTrigger variant="icon" title="部门" active={false} />
                <FilterTrigger variant="icon" title="组织" active />
              </div>
            </VariantCard>

            <VariantCard variant="badge-pencil" desc="行内徽章+铅笔图标" api='<FilterTrigger variant="badge-pencil">' status="skeleton" width="w-auto">
              <FilterTrigger variant="badge-pencil" />
            </VariantCard>
          </div>
        </section>
      </div>

      {/* ════════════════════════════════════════════════════════════════════════
          第四部分：迁移映射总览
         ════════════════════════════════════════════════════════════════════════ */}
      <div className="space-y-6">
        <h2 className="text-xl font-bold text-[var(--text-title)] border-b-2 border-[var(--text-title)] pb-2">
          迁移映射总览
        </h2>
        <BodyText tone="secondary">
          已完成迁移：旧组件实现移至 <code>components/_internal/</code>（仅供新 wrapper 内部使用），业务侧应仅 import 下表"新 import 路径"。
        </BodyText>

        <div className="overflow-x-auto rounded-[4px] border border-[#EAEEF4]">
          <table className="w-full text-sm border-collapse">
            <thead className="bg-[#F8FAFC]">
              <tr className="border-b border-[#EAEEF4]">
                <th className="text-left py-2.5 px-3 font-semibold w-[12%]">类别</th>
                <th className="text-left py-2.5 px-3 font-semibold w-[18%]">旧组件</th>
                <th className="text-left py-2.5 px-3 font-semibold w-[14%]">新组件</th>
                <th className="text-left py-2.5 px-3 font-semibold w-[24%]">新 import 路径</th>
                <th className="text-left py-2.5 px-3 font-semibold">关键 prop 变化</th>
              </tr>
            </thead>
            <tbody className="text-xs">
              {/* 组1 Select */}
              <tr className="border-b border-[#F5F5F5] bg-blue-50/30"><td className="py-2 px-3 font-medium text-blue-600" rowSpan={3}>合并 →</td><td className="py-2 px-3"><code>Select</code></td><td className="py-2 px-3"><code>Select</code></td><td className="py-2 px-3"><code>@/components/ui/select</code></td><td className="py-2 px-3">无变化</td></tr>
              <tr className="border-b border-[#F5F5F5] bg-blue-50/30"><td className="py-2 px-3"><code>SearchableSelect</code></td><td className="py-2 px-3"><code>SearchableSelect</code></td><td className="py-2 px-3"><code>@/components/ui/select</code></td><td className="py-2 px-3">无变化（已存在）</td></tr>
              <tr className="border-b border-[#F5F5F5] bg-blue-50/30"><td className="py-2 px-3"><code>TableHeaderFilter</code></td><td className="py-2 px-3"><code>FilterMultiSelect</code></td><td className="py-2 px-3"><code>@/components/ui/select</code></td><td className="py-2 px-3"><code>selectedValues</code> → <code>value</code>; <code>onConfirm</code> → <code>onChange</code>（旧名兼容）</td></tr>
              {/* 组2 TreeSelect */}
              <tr className="border-b border-[#F5F5F5] bg-blue-50/30"><td className="py-2 px-3 font-medium text-blue-600" rowSpan={2}>合并 →</td><td className="py-2 px-3"><code>TreeSelectFilter</code></td><td className="py-2 px-3"><code>TreeSelect</code></td><td className="py-2 px-3"><code>@/components/ui/tree-select</code></td><td className="py-2 px-3">默认 <code>triggerVariant="button"</code></td></tr>
              <tr className="border-b border-[#F5F5F5] bg-blue-50/30"><td className="py-2 px-3"><code>TableHeaderTreeFilter</code></td><td className="py-2 px-3"><code>TreeSelect</code></td><td className="py-2 px-3"><code>@/components/ui/tree-select</code></td><td className="py-2 px-3">显式传 <code>triggerVariant="filter-icon"</code> + <code>title</code></td></tr>
              {/* 组3 ScopeSelect */}
              <tr className="border-b border-[#F5F5F5] bg-blue-50/30"><td className="py-2 px-3 font-medium text-blue-600" rowSpan={2}>合并 →</td><td className="py-2 px-3"><code>ScopeFilterDropdown</code></td><td className="py-2 px-3"><code>ScopeSelect</code></td><td className="py-2 px-3"><code>@/components/ScopeSelect</code></td><td className="py-2 px-3">不传 <code>scope</code> 自动 instant; <code>selectedKeys</code> → <code>value</code>（兼容）</td></tr>
              <tr className="border-b border-[#F5F5F5] bg-blue-50/30"><td className="py-2 px-3"><code>ScopeEditPopover</code></td><td className="py-2 px-3"><code>ScopeSelect</code></td><td className="py-2 px-3"><code>@/components/ScopeSelect</code></td><td className="py-2 px-3">传 <code>scope</code> 自动 confirm; props 完全兼容</td></tr>
              {/* 独立 */}
              <tr className="border-b border-[#F5F5F5]"><td className="py-2 px-3 font-medium text-amber-700" rowSpan={3}>保持独立</td><td className="py-2 px-3"><code>GroupSelect</code></td><td className="py-2 px-3"><code>GroupSelect</code></td><td className="py-2 px-3"><code>@/components/GroupSelect</code></td><td className="py-2 px-3">已重命名（原 GroupSelectPanel）</td></tr>
              <tr className="border-b border-[#F5F5F5]"><td className="py-2 px-3"><code>TokenValueEditor</code></td><td className="py-2 px-3">不变</td><td className="py-2 px-3"><code>@/components/policy/TokenValueEditor</code></td><td className="py-2 px-3">—</td></tr>
              <tr className="border-b border-[#F5F5F5]"><td className="py-2 px-3"><code>DropdownMenu / MoreActions</code></td><td className="py-2 px-3">不变</td><td className="py-2 px-3"><code>@/components/ui/dropdown-menu</code></td><td className="py-2 px-3">—</td></tr>
              {/* 骨架 */}
              <tr className="border-b border-[#F5F5F5] bg-gray-50"><td className="py-2 px-3 font-medium text-gray-600" rowSpan={2}>底层骨架</td><td className="py-2 px-3"><code>SelectPanel</code></td><td className="py-2 px-3">不变</td><td className="py-2 px-3"><code>@/components/ui/select-panel</code></td><td className="py-2 px-3">—</td></tr>
              <tr className="bg-gray-50"><td className="py-2 px-3"><code>FilterTrigger</code></td><td className="py-2 px-3">不变</td><td className="py-2 px-3"><code>@/components/ui/filter-trigger</code></td><td className="py-2 px-3">—</td></tr>
            </tbody>
          </table>
        </div>
      </div>

      {/* ════════════════════════════════════════════════════════════════════════
          第五部分：设计规范
         ════════════════════════════════════════════════════════════════════════ */}
      <div className="space-y-6">
        <h2 className="text-xl font-bold text-[var(--text-title)] border-b-2 border-[var(--text-title)] pb-2">设计规范</h2>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div className="p-4 rounded-[4px] border border-[#EAEEF4] space-y-1.5"><p className="font-medium">面板外层</p><code className="block text-xs bg-[#F5F5F5] px-2 py-1 rounded">rounded-[4px] border-none shadow-[var(--shadow-popover)]</code></div>
          <div className="p-4 rounded-[4px] border border-[#EAEEF4] space-y-1.5"><p className="font-medium">选项行</p><code className="block text-xs bg-[#F5F5F5] px-2 py-1 rounded">h-8 px-3 rounded-[6px] · space-y-0.5(2px)</code></div>
          <div className="p-4 rounded-[4px] border border-[#EAEEF4] space-y-1.5"><p className="font-medium">选中态</p><code className="block text-xs bg-[#F5F5F5] px-2 py-1 rounded">bg-[var(--bg-brand-selected)] · 文字保持 secondary · 不变蓝/不加粗</code></div>
          <div className="p-4 rounded-[4px] border border-[#EAEEF4] space-y-1.5"><p className="font-medium">Hover</p><code className="block text-xs bg-[#F5F5F5] px-2 py-1 rounded">hover:bg-[var(--bg-grey-hover)]</code></div>
          <div className="p-4 rounded-[4px] border border-[#EAEEF4] space-y-1.5"><p className="font-medium">全选 Checkbox 三态</p><code className="block text-xs bg-[#F5F5F5] px-2 py-1 rounded">全选 checked / 部分 indeterminate(—) / 无 unchecked</code></div>
          <div className="p-4 rounded-[4px] border border-[#EAEEF4] space-y-1.5"><p className="font-medium">Footer</p><code className="block text-xs bg-[#F5F5F5] px-2 py-1 rounded">mx-2 border-t border-[#EAEEF4] py-2</code></div>
          <div className="p-4 rounded-[4px] border border-[#EAEEF4] space-y-1.5"><p className="font-medium">触发器 hover</p><code className="block text-xs bg-[#F5F5F5] px-2 py-1 rounded">hover:border-blue-500</code></div>
          <div className="p-4 rounded-[4px] border border-[#EAEEF4] space-y-1.5"><p className="font-medium">阴影 Token</p><code className="block text-xs bg-[#F5F5F5] px-2 py-1 rounded leading-relaxed">--shadow-popover: 0 0 2px rgba(0,0,0,.1), 0 4px 16px rgba(0,0,0,.12)</code></div>
        </div>
      </div>
    </div>
  );
}
