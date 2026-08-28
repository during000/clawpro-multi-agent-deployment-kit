/**
 * ScopeSelect - 应用范围选择面板（统一封装）
 *
 * 数据结构：ScopeGroup[]（{ id, name, parentId? }）+ "全部用户" / "按组织" 两段式语义
 *
 * 通过 mode 区分两种提交模式：
 *   - "instant"：嵌入式即时多选，无触发器（承载原 ScopeFilterDropdown）
 *     - withTrigger=true 时自动用 Popover 包裹面板 + 渲染触发器按钮
 *   - "confirm"：badge-pencil 触发 + Segment 切换 + 确认提交（承载原 ScopeEditPopover）
 *
 * 设计：内部委托给 ScopeFilterDropdown / ScopeEditPopover 两个旧实现，
 * 待业务全部迁移完成后，再把实现内联进本文件并删除旧文件。
 *
 * 边界兼容：
 *   - 同时支持新 API（value/onChange）和旧 API（selectedKeys/onConfirm/selectedGroupIds）
 *   - mode 默认推断：传 scope/selectedGroupIds 就走 confirm，否则走 instant
 *   - 透传所有原 ScopeEditPopover 边界 prop（trigger / showBadges / scopeLabels / maxVisibleBadges）
 */
import * as React from "react";
import { ChevronDown } from "lucide-react";
import { ScopeFilterDropdown, type ScopeFilterGroup } from "@/components/_internal/ScopeFilterDropdown";
import { ScopeEditPopover, type ScopeGroup, type ScopeType } from "@/components/_internal/ScopeEditPopover";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { normalizeGroupsForUnifiedTree } from "@/pages/admin/MemberManagement/health";

// ─── 类型 ────────────────────────────────────────────────────────────────────

export type { ScopeFilterGroup, ScopeGroup, ScopeType };

export type ScopeSelectMode = "instant" | "confirm";

/**
 * 统一 props（结构性 union），同时承载 instant 和 confirm 两种 API。
 * 字段全部可选 + 运行时根据 mode/scope 分发，避免 TS discriminated-union 在 callback 参数推断上的不友好。
 */
export interface ScopeSelectProps {
  /** 显式指定模式；未传时根据是否提供 scope 推断（提供 scope → confirm，否则 → instant） */
  mode?: ScopeSelectMode;

  // ── instant 模式相关 ──
  /** instant 模式：组织列表（扁平） */
  groups?: ScopeFilterGroup[] | ScopeGroup[];
  /** instant 模式：选中 keys（新 API） */
  value?: Set<string>;
  /** instant 模式：选中变化回调（新 API） */
  onChange?: (value: Set<string>) => void;
  /** @deprecated instant 模式旧 prop，请改用 value */
  selectedKeys?: Set<string>;
  /** instant 模式：搜索框 placeholder */
  searchPlaceholder?: string;
  /** instant 模式："全部用户"对应的 key（默认 "public"） */
  publicKey?: string;
  /** instant 模式："全部应用范围"文案 */
  allLabel?: string;
  /** instant 模式："全部用户"组织标题 */
  publicGroupLabel?: string;
  /** instant 模式："按组织"标题 */
  groupSectionLabel?: string;
  /** instant 模式：已选计数模板 */
  selectedCountTemplate?: string;
  /** instant 模式：面板宽度 className */
  widthClass?: string;
  /** instant 模式：隐藏「全部用户」分区（publicKey 在外部已作为伪组织注入时使用） */
  hidePublicGroup?: boolean;

  // ── instant + withTrigger 相关 ──
  /**
   * instant 模式：是否自动包裹 Popover 触发器。
   * - false（默认）：渲染纯面板（兼容嵌入式使用）
   * - true：用 Popover + PopoverTrigger + PopoverContent 包裹，自动管理开关
   */
  withTrigger?: boolean;
  /** instant + withTrigger：触发器按钮显示的文字（默认根据选中状态自动生成） */
  triggerLabel?: React.ReactNode;
  /** instant + withTrigger：触发器 placeholder（未选中时显示，默认 "请选择"） */
  triggerPlaceholder?: string;
  /** instant + withTrigger：触发器额外 className */
  triggerClassName?: string;
  /** instant + withTrigger：自定义触发器渲染（覆盖默认 button） */
  trigger?: React.ReactNode;
  /** instant + withTrigger：Popover 对齐方式（默认 "start"） */
  align?: "start" | "center" | "end";
  /** instant + withTrigger：受控的 open 状态 */
  open?: boolean;
  /** instant + withTrigger：open 变化回调 */
  onOpenChange?: (open: boolean) => void;

  // ── confirm 模式相关 ──
  /** confirm 模式：当前应用范围（提供时自动切到 confirm） */
  scope?: ScopeType;
  /** confirm 模式：当前选中组织 id */
  selectedGroupIds?: string[];
  /**
   * confirm 模式：可选项目列表。传入且非空时，「按组织」面板会额外渲染「项目」分组小标题，
   * 支持同时选择组织与项目；不传保持原有仅选组织行为。
   */
  projects?: ScopeGroup[];
  /** confirm 模式：确认提交回调 */
  onConfirm?: (scope: ScopeType, groupIds: string[]) => void;
  /** confirm 模式：是否显示范围徽章 */
  showBadges?: boolean;
  /** confirm 模式：已选范围的展示标签 */
  scopeLabels?: string[];
  /** confirm 模式：最多展示几个组织 tag */
  maxVisibleBadges?: number;
}

// ─── 实现 ────────────────────────────────────────────────────────────────────

export function ScopeSelect(props: ScopeSelectProps) {
  // 推断 mode：显式传 mode 优先，否则按 scope 字段是否存在推断
  const resolvedMode: ScopeSelectMode = props.mode ?? (props.scope !== undefined ? "confirm" : "instant");

  if (resolvedMode === "confirm") {
    const {
      scope = "all",
      selectedGroupIds = [],
      groups = [],
      projects,
      onConfirm,
      trigger,
      align,
      showBadges,
      scopeLabels,
      maxVisibleBadges,
    } = props;
    // 在传给底层 ScopeEditPopover 之前，把 OneID 已同步部门时的"顶层 oneid-group / manual"
    // 重映射到 dept-root 下，使 A公司 成为唯一顶层组织。底层只看 parentId，因此规范化数据足矣。
    // 注意：传入原 groups（含 source 字段），保证 normalize 能识别"是否已同步部门"。
    // 项目（projects）为独立分组，不参与组织归一化，原样透传。
    const normalizedGroups = normalizeGroupsForUnifiedTree(
      groups as Array<ScopeGroup & { source?: "oneid-dept" | "oneid-group" | "manual" }>
    );
    return (
      <ScopeEditPopover
        scope={scope}
        selectedGroupIds={selectedGroupIds}
        groups={normalizedGroups as ScopeGroup[]}
        projects={projects}
        onConfirm={onConfirm ?? (() => {})}
        trigger={trigger}
        align={align}
        showBadges={showBadges}
        scopeLabels={scopeLabels}
        maxVisibleBadges={maxVisibleBadges}
      />
    );
  }

  // instant 变体
  const {
    groups = [],
    value,
    onChange,
    selectedKeys,
    searchPlaceholder,
    publicKey,
    allLabel,
    publicGroupLabel,
    groupSectionLabel,
    selectedCountTemplate,
    widthClass,
    hidePublicGroup,
    withTrigger = false,
    triggerLabel,
    triggerPlaceholder = "请选择",
    triggerClassName,
    trigger,
    align = "start",
    open,
    onOpenChange,
  } = props;

  const panel = (
    <ScopeFilterDropdown
      groups={groups as ScopeFilterGroup[]}
      selectedKeys={value ?? selectedKeys ?? new Set()}
      onChange={onChange ?? (() => {})}
      searchPlaceholder={searchPlaceholder}
      publicKey={publicKey}
      allLabel={allLabel}
      publicGroupLabel={publicGroupLabel}
      groupSectionLabel={groupSectionLabel}
      selectedCountTemplate={selectedCountTemplate}
      widthClass={widthClass}
      hidePublicGroup={hidePublicGroup}
    />
  );

  if (!withTrigger) {
    return panel;
  }

  // withTrigger=true：用 Popover 包裹面板
  return (
    <ScopeInstantPopover
      panel={panel}
      triggerLabel={triggerLabel}
      triggerPlaceholder={triggerPlaceholder}
      triggerClassName={triggerClassName}
      trigger={trigger}
      align={align}
      open={open}
      onOpenChange={onOpenChange}
    />
  );
}

// ─── instant + withTrigger 内部 Popover 封装 ─────────────────────────────────

interface ScopeInstantPopoverProps {
  panel: React.ReactNode;
  triggerLabel?: React.ReactNode;
  triggerPlaceholder: string;
  triggerClassName?: string;
  trigger?: React.ReactNode;
  align: "start" | "center" | "end";
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

function ScopeInstantPopover({
  panel,
  triggerLabel,
  triggerPlaceholder,
  triggerClassName,
  trigger,
  align,
  open: controlledOpen,
  onOpenChange,
}: ScopeInstantPopoverProps) {
  const [internalOpen, setInternalOpen] = React.useState(false);
  const isControlled = controlledOpen !== undefined;
  const isOpen = isControlled ? controlledOpen : internalOpen;
  const setOpen = (v: boolean) => {
    if (!isControlled) setInternalOpen(v);
    onOpenChange?.(v);
  };

  return (
    <Popover open={isOpen} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        {trigger ?? (
          <button
            type="button"
            data-state={isOpen ? "open" : "closed"}
            className={
              triggerClassName ??
              "flex items-center justify-between gap-1 min-w-[8rem] max-w-[16rem] h-9 px-3 border border-border rounded-[4px] bg-white text-sm text-[#020617] hover:border-blue-500 data-[state=open]:border-blue-500 transition-colors outline-none"
            }
          >
            <span className="truncate text-left">
              {triggerLabel ?? triggerPlaceholder}
            </span>
            <ChevronDown
              className={`w-4 h-4 text-[var(--text-weak)] flex-shrink-0 transition-transform ${isOpen ? "rotate-180" : ""}`}
            />
          </button>
        )}
      </PopoverTrigger>
      <PopoverContent
        align={align}
        sideOffset={4}
        className="p-0 border-none w-auto"
      >
        {panel}
      </PopoverContent>
    </Popover>
  );
}

export default ScopeSelect;
