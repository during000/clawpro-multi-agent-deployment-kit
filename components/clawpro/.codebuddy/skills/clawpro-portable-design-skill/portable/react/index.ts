/**
 * ClawPro Portable Design — React Components Index
 * ───────────────────────────────────────────────────────────────────────────
 * 统一导出所有 portable React 组件。
 *
 * 用法（外部仓）：
 *   1. 装 Tailwind（推荐）。本包组件大量使用 Tailwind utility class
 *      + 任意值语法 `[var(--cp-*)]`，Tailwind 默认支持。
 *   2. 在你的入口 CSS 引入：
 *        @import "clawpro-portable/css/portable.css";
 *      （这一行会按顺序拉入 tokens.css → globals.css → 各组件 CSS）
 *   3. import 组件：
 *        import { PortableButton, PortableDataTable } from "clawpro-portable/react";
 *
 * ⚠️ 命名前缀统一 `Portable*`，避免与宿主仓自有组件碰撞。
 * ───────────────────────────────────────────────────────────────────────────
 */

/* ─── Foundation ─── */
export {
  PortableSurfaceCard,
  PortableSurfaceInner,
  PortableSurfaceConfig,
  PortableTenantCard,
} from "./card";
export type {
  PortableSurfaceCardProps,
  PortableSurfaceInnerProps,
  PortableSurfaceConfigProps,
  PortableTenantCardProps,
} from "./card";

/* ─── Action ─── */
export { PortableButton } from "./button";
export type {
  PortableButtonProps,
  PortableButtonVariant,
  PortableButtonSize,
} from "./button";

/* ─── Form ─── */
export {
  PortableInput,
  PortableSelect,
  PortableField,
} from "./input-select";
export type {
  PortableInputProps,
  PortableSelectProps,
  PortableSelectOption,
  PortableFieldProps,
} from "./input-select";

export { PortableSearchableSelect } from "./searchable-select";
export type {
  PortableSearchableSelectProps,
  PortableSearchableSelectOption,
} from "./searchable-select";

export {
  PortableSwitch,
  PortableCheckbox,
  PortableRadio,
  PortableRadioGroup,
} from "./selection-controls";
export type {
  PortableSwitchProps,
  PortableCheckboxProps,
  PortableRadioProps,
  PortableRadioGroupProps,
} from "./selection-controls";

export { PortableDatePicker } from "./date-picker";

export {
  PortableLabel,
  PortableFieldGroup,
  PortableFieldRow,
  PortableHelperText,
  PortableFieldError,
} from "./form-controls";
export type {
  PortableLabelProps,
  PortableFieldGroupProps,
  PortableFieldRowProps,
  PortableHelperTextProps,
  PortableFieldErrorProps,
} from "./form-controls";

/* ─── Data ─── */
export {
  PortableTable,
  PortableTableHeader,
  PortableTableBody,
  PortableTableRow,
  PortableTableHead,
  PortableTableCell,
  PortableTableActions,
  PortableTableStatus,
  PortableDataTable,
  PortableAdminTable,
} from "./table";
export type {
  PortableTableHeadProps,
  PortableTableCellProps,
  PortableTableStatusProps,
  PortableTableStatusTone,
  PortableDataTableColumn,
  PortableDataTableProps,
} from "./table";

export { PortablePagination } from "./pagination";
export type { PortablePaginationProps } from "./pagination";

export { PortableEmpty, PortableTableEmpty } from "./empty-state";
export type {
  PortableEmptyProps,
  PortableTableEmptyProps,
} from "./empty-state";

export {
  PortableNumberCard,
  PortableStatNumber,
  PortableGradientIcon,
  PortableRequestsIcon,
  PortableInputTokensIcon,
  PortableOutputTokensIcon,
  PortableTotalTokensIcon,
  PORTABLE_ICONS,
} from "./number-card";
export type {
  PortableNumberCardProps,
  PortableGradientIconProps,
} from "./number-card";

export { PortableSearchFilterBar } from "./search-filter-bar";
export type { PortableSearchFilterBarProps } from "./search-filter-bar";

export { PortableBatchActionsBar } from "./batch-actions-bar";
export type {
  PortableBatchActionsBarProps,
  PortableBatchActionsBarAction,
} from "./batch-actions-bar";

export { PortableTransfer } from "./transfer";
export type {
  PortableTransferProps,
  PortableTransferItem,
} from "./transfer";

export { PortableFileBrowser } from "./file-browser";
export type {
  PortableFileBrowserProps,
  VersionInfo,
  FileEntry,
} from "./file-browser";

/* ─── Chart / Stat ─── */
export {
  PortableChartCard,
  PortableChartLegend,
  PortableChartTooltip,
  PortableChartDelta,
} from "./chart-stat";
export type {
  PortableChartCardProps,
  PortableChartState,
  PortableChartLegendItem,
  PortableChartTooltipRow,
} from "./chart-stat";

/* ─── Feedback / Loading ─── */
export {
  PortableSpinner,
  PortableSkeleton,
  PortableProgress,
  PortableTableSkeleton,
} from "./loading-progress";
export type {
  PortableSpinnerProps,
  PortableSkeletonProps,
  PortableProgressProps,
  PortableTableSkeletonProps,
} from "./loading-progress";

export { PortableTooltip } from "./tooltip";
export type {
  PortableTooltipProps,
  PortableTooltipSide,
} from "./tooltip";

export {
  PortableAlert,
  PortableAlertTitle,
  PortableAlertDescription,
  PortableAlertInfoIcon,
  PortableAlertOperationInfoIcon,
  PortableAlertProductNewsIcon,
  PortableAlertWarningIcon,
  PortableAlertSuccessIcon,
  PortableAlertErrorIcon,
  PortableAdminNoticeAlert,
} from "./alert";
export type {
  PortableAlertProps,
} from "./alert";

/* ─── Feedback ─── */
export {
  PortableDialog,
  PortableAlertDialog,
  PortableDrawer,
  PortableDialogFooter,
} from "./dialog-drawer";
export type {
  PortableDialogProps,
  PortableAlertDialogProps,
  PortableDrawerProps,
} from "./dialog-drawer";

/* ─── Navigation ─── */
export {
  PortableTabs,
  PortableTabsList,
  PortableTabsTrigger,
  PortableTabsContent,
  PortableLineTabs,
  PortableLineTabsList,
  PortableLineTabsTrigger,
  PortableLineTabsContent,
} from "./tabs";
export type {
  PortableTabsProps,
  PortableTabsListProps,
  PortableTabsTriggerProps,
  PortableTabsContentProps,
} from "./tabs";

export {
  PortableAdminSegment,
  PortableAdminSegmentItem,
  PortableAdminSegmentGroup,
  PortableAdminSegmentOption,
  PortableTenantSegment,
  PortableTenantSegmentItem,
  PortableTenantSegmentGroup,
  PortableTenantSegmentOption,
} from "./segment";
export type {
  PortableAdminSegmentProps,
  PortableAdminSegmentItemProps,
  PortableAdminSegmentGroupProps,
  PortableAdminSegmentOptionProps,
  PortableTenantSegmentProps,
  PortableTenantSegmentItemProps,
  PortableTenantSegmentGroupProps,
  PortableTenantSegmentOptionProps,
} from "./segment";

export { PortableBreadcrumb } from "./breadcrumb";
export type {
  PortableBreadcrumbProps,
  PortableBreadcrumbItem,
} from "./breadcrumb";

export { PortablePageHeader } from "./page-header";
export type { PortablePageHeaderProps } from "./page-header";

export { PortableTree } from "./tree";
export type {
  PortableTreeProps,
  PortableTreeItem,
} from "./tree";

export { PortableTreeSelect } from "./tree-select";
export type {
  PortableTreeSelectProps,
  PortableTreeSelectNode,
  PortableTreeSelectSection,
  PortableTreeSelectAlign,
} from "./tree-select";

/* ─── Overlay（Popover / DropdownMenu） ─── */
export {
  PortableDropdownMenu,
  PortablePopover,
} from "./popover-menu";
export type {
  PortableDropdownMenuProps,
  PortablePopoverProps,
  PortableMenuItem,
} from "./popover-menu";

/* ─── Media ─── */
export { PortableAvatar, PortableAvatarGroup } from "./avatar";
export type {
  PortableAvatarProps,
  PortableAvatarSize,
  PortableAvatarGroupProps,
} from "./avatar";

/* ─── Tag / Status ─── */
export { PortableBadge } from "./badges";
export type {
  PortableBadgeProps,
  PortableBadgeColor,
  PortableBadgeVariant,
} from "./badges";

export { PortableStatusTag } from "./status-tag";
export type {
  PortableStatusTagProps,
  PortableStatusVariant,
} from "./status-tag";

/* ─── Admin Sidebar（已含独立 CSS） ─── */
export * from "./admin-sidebar";
