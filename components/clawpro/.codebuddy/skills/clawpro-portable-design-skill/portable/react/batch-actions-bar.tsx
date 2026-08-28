import * as React from "react";

/**
 * Configuration for a single batch action button.
 * Each action requires a unique key, display label, and click handler.
 * Optional properties control danger state, disabled state, and help text.
 */
export interface PortableBatchActionsBarAction {
  /** Unique identifier for the action */
  key: string;
  /** Display label for the button */
  label: string;
  /** Callback fired when action button is clicked */
  onClick: () => void;
  /** Whether this is a destructive/danger action (requires confirmation) */
  danger?: boolean;
  /** Whether the action button is disabled */
  disabled?: boolean;
  /** Reason why the action is disabled (shown in tooltip/helper) */
  disabledReason?: string;
  /** Whether the action is in loading/processing state */
  loading?: boolean;
}

/**
 * Props for PortableBatchActionsBar component.
 * Handles selection summary, cross-page selection prompts, and batch actions.
 */
export interface PortableBatchActionsBarProps {
  /** Number of currently selected items */
  selectedCount: number;
  /** Total number of items available (for "select all" calculation) */
  totalCount?: number;
  /** Whether all items on the current page are selected */
  isPageAllSelected?: boolean;
  /** Whether all filtered items across all pages are selected */
  isAllFilteredSelected?: boolean;
  /** Callback to select all filtered items (cross-page selection) */
  onSelectAllFiltered?: () => void;
  /** Callback to clear all selections */
  onClearSelection: () => void;
  /** Array of batch action configurations */
  actions: PortableBatchActionsBarAction[];
  /** Additional CSS class names to append to root container */
  className?: string;
}

/**
 * PortableBatchActionsBar Component
 *
 * A unified batch selection feedback and action bar for data tables and lists.
 * Displays:
 * - Selection count with clear number display
 * - Cross-page selection prompt (when applicable)
 * - Batch action buttons (normal and danger variants)
 * - Clear selection button
 *
 * Visibility:
 * - Hidden when selectedCount is 0
 * - Uses grey background when all filtered items are selected
 *
 * Design system:
 * - Follows ClawPro portable design specifications
 * - Uses CSS custom properties for theming (--cp-*, --bg-*)
 * - Responsive design for mobile viewports
 *
 * @example
 * ```tsx
 * <PortableBatchActionsBar
 *   selectedCount={5}
 *   totalCount={36}
 *   isPageAllSelected={true}
 *   isAllFilteredSelected={false}
 *   onSelectAllFiltered={() => setSelectedCount(36)}
 *   onClearSelection={() => setSelectedCount(0)}
 *   actions={[
 *     {
 *       key: "delete",
 *       label: "批量删除",
 *       danger: true,
 *       onClick: handleBatchDelete,
 *     },
 *     {
 *       key: "export",
 *       label: "批量导出",
 *       onClick: handleBatchExport,
 *     },
 *   ]}
 * />
 * ```
 */
export function PortableBatchActionsBar(
  props: PortableBatchActionsBarProps
): React.ReactElement | null {
  const {
    selectedCount,
    totalCount = 0,
    isPageAllSelected = false,
    isAllFilteredSelected = false,
    onSelectAllFiltered,
    onClearSelection,
    actions,
    className,
  } = props;

  // Hide when no items are selected
  if (selectedCount === 0) {
    return null;
  }

  // Determine if we should show the "select all filtered" prompt
  const shouldShowSelectAllPrompt =
    isPageAllSelected &&
    !isAllFilteredSelected &&
    totalCount > selectedCount &&
    onSelectAllFiltered;

  // Build root container class names
  const rootClasses = [
    "cp-batch-bar",
    isAllFilteredSelected && "cp-batch-bar--all",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  // Summary text based on selection state
  const summaryText = isAllFilteredSelected
    ? `已选择全部 ${totalCount} 项（跨页）`
    : `已选择 ${selectedCount} 项`;

  return (
    <div className={rootClasses}>
      {/* Selection Summary */}
      <div className="cp-batch-summary">
        <strong>{summaryText}</strong>
        {shouldShowSelectAllPrompt && (
          <button
            type="button"
            className="cp-batch-summary__link"
            onClick={onSelectAllFiltered}
            aria-label={`选择全部 ${totalCount} 项`}
          >
            选择全部 {totalCount} 项
          </button>
        )}
      </div>

      {/* Action Buttons */}
      <div className="cp-batch-actions">
        {actions.map((action) => {
          const actionClasses = [
            "cp-batch-action",
            action.danger && "cp-batch-action--danger",
            action.loading && "is-loading",
          ]
            .filter(Boolean)
            .join(" ");

          return (
            <button
              key={action.key}
              type="button"
              className={actionClasses}
              onClick={action.onClick}
              disabled={action.disabled || action.loading}
              title={action.disabledReason}
              aria-busy={action.loading}
              aria-disabled={action.disabled}
            >
              {action.loading ? "处理中..." : action.label}
            </button>
          );
        })}

        {/* Clear Selection Button */}
        <button
          type="button"
          className="cp-batch-action cp-batch-action--clear"
          onClick={onClearSelection}
          aria-label="清除选择"
        >
          清除选择
        </button>
      </div>
    </div>
  );
}

PortableBatchActionsBar.displayName = "PortableBatchActionsBar";
