/**
 * Portable Table — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 Table 时的可移植兜底实现。
 *  - 不依赖 shadcn / Radix / cva / Tailwind。
 *  - 颜色 / 尺寸 / 状态 / 空态 / 页脚全部由 portable/css/table.css 提供。
 *  - 视觉规范（spec/component-specs/table.md §3）：
 *      容器：圆角 4px + 蓝灰描边 + 白底
 *      表头：bg var(--cp-bg-subtle) / 12px / Medium / var(--cp-text-muted) / 高 40px
 *      表体：12px / 行高 54px（compact 40px）/ 横向 padding 16px / hover 弱灰
 *      操作列：纯蓝色文字按钮，间距 24px，禁红色
 *      状态列：纯文字变色，无底色无圆点（禁胶囊）
 *      空态：colSpan 内纯文字双行（禁止插画）
 *      字号：12px 全局 !important（已由 globals.css 注入）
 *
 *  - 业务调用方推荐用 <PortableDataTable> 声明式 API（columns + dataSource）。
 *  - 进阶场景用底座组件（<PortableTable>/<PortableTableHead>/<PortableTableCell>）。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/table.css";
 *
 * 用法：
 *   <PortableDataTable
 *     rowKey="id"
 *     columns={[
 *       { key: "name", title: "名称" },
 *       { key: "status", title: "状态",
 *         render: (_, row) => (
 *           <PortableTableStatus tone="success">已启用</PortableTableStatus>
 *         )
 *       },
 *       { key: "actions", title: "操作",
 *         render: (_, row) => (
 *           <PortableTableActions>
 *             <button className="cp-table-action">编辑</button>
 *             <button className="cp-table-action">删除</button>
 *           </PortableTableActions>
 *         )
 *       }
 *     ]}
 *     dataSource={list}
 *     loading={isLoading}
 *     size="default"
 *     rowSelection={{ selectedKeys, onChange: setSelected }}
 *     pagination={{ current: 1, pageSize: 10, total: 100, onChange: setPage }}
 *     onRow={(record, index) => ({
 *       className: "custom-row"
 *     })}
 *   />
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

type Align = "left" | "right" | "center";

function cx(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(" ");
}

function alignClass(align?: Align): string | undefined {
  if (align === "right") return "cp-col-right";
  if (align === "center") return "cp-col-center";
  return undefined;
}

/* ════════════ 底座：组合式 Table 套件 ════════════ */

export const PortableTable = React.forwardRef<
  HTMLTableElement,
  React.TableHTMLAttributes<HTMLTableElement> & { compact?: boolean }
>(({ className = "", compact = false, ...props }, ref) => (
  <div className="cp-table-scroll">
    <table
      ref={ref}
      data-slot="table"
      className={cx("cp-table", compact && "cp-table--compact", className)}
      {...props}
    />
  </div>
));
PortableTable.displayName = "PortableTable";

export const PortableTableHeader = React.forwardRef<
  HTMLTableSectionElement,
  React.HTMLAttributes<HTMLTableSectionElement>
>(({ className = "", ...props }, ref) => (
  <thead ref={ref} data-slot="table-header" className={className} {...props} />
));
PortableTableHeader.displayName = "PortableTableHeader";

export const PortableTableBody = React.forwardRef<
  HTMLTableSectionElement,
  React.HTMLAttributes<HTMLTableSectionElement>
>(({ className = "", ...props }, ref) => (
  <tbody ref={ref} data-slot="table-body" className={className} {...props} />
));
PortableTableBody.displayName = "PortableTableBody";

export const PortableTableRow = React.forwardRef<
  HTMLTableRowElement,
  React.HTMLAttributes<HTMLTableRowElement>
>(({ className = "", ...props }, ref) => (
  <tr ref={ref} data-slot="table-row" className={className} {...props} />
));
PortableTableRow.displayName = "PortableTableRow";

export interface PortableTableHeadProps
  extends React.ThHTMLAttributes<HTMLTableCellElement> {
  align?: Align;
}

export const PortableTableHead = React.forwardRef<
  HTMLTableCellElement,
  PortableTableHeadProps
>(({ align = "left", className = "", ...props }, ref) => (
  <th
    ref={ref}
    data-slot="table-head"
    className={cx(alignClass(align), className)}
    {...props}
  />
));
PortableTableHead.displayName = "PortableTableHead";

export interface PortableTableCellProps
  extends React.TdHTMLAttributes<HTMLTableCellElement> {
  align?: Align;
}

export const PortableTableCell = React.forwardRef<
  HTMLTableCellElement,
  PortableTableCellProps
>(({ align = "left", className = "", ...props }, ref) => (
  <td
    ref={ref}
    data-slot="table-cell"
    className={cx(alignClass(align), className)}
    {...props}
  />
));
PortableTableCell.displayName = "PortableTableCell";

/* ───────────── PortableTableActions ─────────────
 * 操作列容器：内置 flex + gap 24px；调用方放 .cp-table-action 纯文字蓝色按钮。
 */
export const PortableTableActions = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className = "", children, ...props }, ref) => (
  <div ref={ref} className={cx("cp-table-actions", className)} {...props}>
    {children}
  </div>
));
PortableTableActions.displayName = "PortableTableActions";

/* ───────────── PortableTableStatus ─────────────
 * 状态列：纯文字变色（禁胶囊 / 禁圆点）。
 */
export type PortableTableStatusTone =
  | "info"
  | "success"
  | "danger"
  | "warning"
  | "muted";

export interface PortableTableStatusProps
  extends React.HTMLAttributes<HTMLSpanElement> {
  tone?: PortableTableStatusTone;
}

export const PortableTableStatus = React.forwardRef<
  HTMLSpanElement,
  PortableTableStatusProps
>(({ tone = "muted", className = "", ...props }, ref) => (
  <span
    ref={ref}
    className={cx("cp-table-status", `cp-table-status--${tone}`, className)}
    {...props}
  />
));
PortableTableStatus.displayName = "PortableTableStatus";

/* ════════════ 声明式 DataTable API ════════════ */

export interface PortableDataTableColumn<T> {
  key: string;
  title?: React.ReactNode;
  dataIndex?: keyof T;
  render?: (value: unknown, record: T, index: number) => React.ReactNode;
  width?: number | string;
  align?: Align;
  className?: string;
}

export interface PortableRowSelectionProps<T> {
  /** 选中的行 key 数组 */
  selectedKeys: string[];
  /** 选择变化回调 */
  onChange: (keys: string[], records: T[]) => void;
  /** 获取单行禁用状态 */
  getCheckboxProps?: (record: T) => { disabled?: boolean };
  /** 选择类型 */
  type?: "checkbox" | "radio";
}

export interface PortablePaginationProps {
  /** 当前页 */
  current: number;
  /** 每页条数 */
  pageSize: number;
  /** 总条数 */
  total: number;
  /** 分页变化回调 */
  onChange: (page: number, pageSize: number) => void;
}

export interface PortableDataTableProps<T> {
  columns: PortableDataTableColumn<T>[];
  dataSource: T[];
  rowKey: keyof T | ((record: T) => string);
  /** 自带卡片边框（默认 true）。设为 false 时调用方自行包外层 SurfaceCard */
  bordered?: boolean;
  /** 紧凑密度（行高 40px）。建议用 size="compact" 替代 */
  compact?: boolean;
  /** 行高尺寸。默认 "default"（54px），"compact" 为 40px */
  size?: "default" | "compact";
  /** 总数文案（不传则不显示页脚） */
  total?: number;
  /** 自定义页脚（如分页器） */
  footer?: React.ReactNode;
  /** 分页配置。false 表示不分页 */
  pagination?: false | PortablePaginationProps;
  /** 行选择配置 */
  rowSelection?: PortableRowSelectionProps<T>;
  /** 表格加载状态。true 时显示掩层 + 中心 spinner */
  loading?: boolean;
  /** 空态文案（首行主文案） */
  emptyText?: React.ReactNode;
  /** 空态副文案（第二行） */
  emptyDescription?: React.ReactNode;
  /** 行级事件和属性 */
  onRow?: (record: T, index: number) => React.HTMLAttributes<HTMLTableRowElement>;
  /** 行级类名。支持字符串或函数 */
  rowClassName?: string | ((record: T, index: number) => string);
  className?: string;
}

function resolveKey<T>(
  rowKey: PortableDataTableProps<T>["rowKey"],
  record: T,
  index: number
): string {
  if (typeof rowKey === "function") return rowKey(record);
  return String(record[rowKey] ?? index);
}

export function PortableDataTable<T>({
  columns,
  dataSource,
  rowKey,
  bordered = true,
  compact,
  size = "default",
  total,
  footer,
  pagination,
  rowSelection,
  loading = false,
  emptyText = "暂无数据",
  emptyDescription,
  onRow,
  rowClassName,
  className = "",
}: PortableDataTableProps<T>) {
  const isEmpty = dataSource.length === 0;

  // 向后兼容：compact prop 优先级高于 size prop
  const finalSize = compact !== undefined ? (compact ? "compact" : "default") : size;
  const isCompact = finalSize === "compact";

  // 重新计算列数（需考虑选择列）
  const columnCount = columns.length + (rowSelection ? 1 : 0);

  // 构建选中行记录的映射表（便于查询）
  const selectedRecordMap = new Map<string, T>();
  if (rowSelection && rowSelection.selectedKeys.length > 0) {
    rowSelection.selectedKeys.forEach((key) => {
      dataSource.forEach((record, idx) => {
        if (resolveKey(rowKey, record, idx) === key) {
          selectedRecordMap.set(key, record);
        }
      });
    });
  }

  // 判断全选状态（仅在非空表格时考虑）
  const isAllSelected =
    !isEmpty &&
    rowSelection &&
    dataSource.every((record, idx) => {
      const k = resolveKey(rowKey, record, idx);
      const props = rowSelection.getCheckboxProps?.(record);
      return props?.disabled || rowSelection.selectedKeys.includes(k);
    });

  // 获取行级属性（onRow 回调）
  const getRowProps = (record: T, index: number) => {
    const attrs = onRow?.(record, index);
    const key = resolveKey(rowKey, record, index);
    const isSelected = rowSelection?.selectedKeys.includes(key);

    // 计算行级类名
    const classNames: string[] = [];
    if (rowClassName) {
      if (typeof rowClassName === "function") {
        classNames.push(rowClassName(record, index));
      } else {
        classNames.push(rowClassName);
      }
    }

    // 应用 onRow 的类名
    if (attrs?.className) {
      classNames.push(attrs.className);
    }

    return {
      ...attrs,
      className: classNames.filter(Boolean).join(" "),
    };
  };

  // 处理全选
  const handleSelectAll = (checked: boolean) => {
    if (!rowSelection) return;

    let newKeys: string[] = [];
    if (checked) {
      // 全选：包括所有未禁用的行
      newKeys = dataSource
        .map((record, idx) => {
          const k = resolveKey(rowKey, record, idx);
          const props = rowSelection.getCheckboxProps?.(record);
          return props?.disabled ? null : k;
        })
        .filter(Boolean) as string[];
    }
    // 不检查 preserveSelectedKeys；这里只处理当前页

    const selectedRecords = dataSource.filter((record, idx) => {
      const k = resolveKey(rowKey, record, idx);
      return newKeys.includes(k);
    });

    rowSelection.onChange(newKeys, selectedRecords);
  };

  // 处理单行选择
  const handleRowSelect = (record: T, index: number, checked: boolean) => {
    if (!rowSelection) return;

    const k = resolveKey(rowKey, record, index);
    let newKeys = [...rowSelection.selectedKeys];

    if (rowSelection.type === "radio") {
      // 单选模式
      newKeys = checked ? [k] : [];
    } else {
      // 复选模式
      if (checked) {
        if (!newKeys.includes(k)) {
          newKeys.push(k);
        }
      } else {
        newKeys = newKeys.filter((key) => key !== k);
      }
    }

    const selectedRecords = dataSource.filter((record, idx) => {
      const rk = resolveKey(rowKey, record, idx);
      return newKeys.includes(rk);
    });

    rowSelection.onChange(newKeys, selectedRecords);
  };

  // 渲染 Checkbox/Radio
  const renderCheckbox = (record: T, index: number) => {
    if (!rowSelection) return null;

    const k = resolveKey(rowKey, record, index);
    const isChecked = rowSelection.selectedKeys.includes(k);
    const props = rowSelection.getCheckboxProps?.(record);
    const isDisabled = props?.disabled ?? false;
    const isRadio = rowSelection.type === "radio";

    return (
      <input
        type={isRadio ? "radio" : "checkbox"}
        checked={isChecked}
        disabled={isDisabled}
        onChange={(e) => handleRowSelect(record, index, e.target.checked)}
        style={{ cursor: isDisabled ? "not-allowed" : "pointer" }}
      />
    );
  };

  // 渲染表头 Checkbox（仅在复选模式）
  const renderHeaderCheckbox = () => {
    if (!rowSelection || rowSelection.type === "radio" || isEmpty) {
      return null;
    }

    // 部分选中状态：有选中但不全选
    const hasPartial =
      rowSelection.selectedKeys.length > 0 && !isAllSelected;

    return (
      <input
        type="checkbox"
        checked={isAllSelected}
        ref={(input) => {
          if (input) {
            input.indeterminate = hasPartial;
          }
        }}
        onChange={(e) => handleSelectAll(e.target.checked)}
        style={{ cursor: "pointer" }}
      />
    );
  };

  return (
    <section className={cx(bordered && "cp-table-shell", className)}>
      {/* 表格容器 with loading mask */}
      <div style={{ position: "relative" }}>
        <PortableTable compact={isCompact}>
          <PortableTableHeader>
            <PortableTableRow>
              {rowSelection && (
                <PortableTableHead
                  style={{ width: "40px", textAlign: "center" }}
                >
                  {renderHeaderCheckbox()}
                </PortableTableHead>
              )}
              {columns.map((col) => (
                <PortableTableHead
                  key={col.key}
                  align={col.align}
                  className={col.className}
                  style={col.width ? { width: col.width } : undefined}
                >
                  {col.title}
                </PortableTableHead>
              ))}
            </PortableTableRow>
          </PortableTableHeader>
          <PortableTableBody>
            {isEmpty ? (
              <tr className="cp-table-empty-row">
                <td colSpan={columnCount} className="cp-table-empty">
                  <p>{emptyText}</p>
                  {emptyDescription ? <p>{emptyDescription}</p> : null}
                </td>
              </tr>
            ) : (
              dataSource.map((record, idx) => {
                const k = resolveKey(rowKey, record, idx);
                const rowAttrs = getRowProps(record, idx);

                return (
                  <PortableTableRow key={k} {...rowAttrs}>
                    {rowSelection && (
                      <PortableTableCell style={{ textAlign: "center" }}>
                        {renderCheckbox(record, idx)}
                      </PortableTableCell>
                    )}
                    {columns.map((col) => {
                      const val = col.dataIndex
                        ? (record as Record<string, unknown>)[col.dataIndex as string]
                        : undefined;
                      const content = col.render
                        ? col.render(val, record, idx)
                        : (val as React.ReactNode);
                      return (
                        <PortableTableCell
                          key={col.key}
                          align={col.align}
                          className={col.className}
                          style={col.width ? { width: col.width } : undefined}
                        >
                          {content as React.ReactNode}
                        </PortableTableCell>
                      );
                    })}
                  </PortableTableRow>
                );
              })
            )}
          </PortableTableBody>
        </PortableTable>

        {/* Loading mask */}
        {loading && !isEmpty && (
          <div
            style={{
              position: "absolute",
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              background: "rgba(255, 255, 255, 0.7)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              zIndex: 10,
            }}
          >
            {/* 简单 spinner 实现 */}
            <div
              style={{
                width: "32px",
                height: "32px",
                border: "3px solid rgba(20, 71, 230, 0.2)",
                borderTop: "3px solid #1447E6",
                borderRadius: "50%",
                animation: "spin 0.8s linear infinite",
              }}
            />
            <style>{`
              @keyframes spin {
                to { transform: rotate(360deg); }
              }
            `}</style>
          </div>
        )}
      </div>

      {/* 分页与页脚 */}
      {(footer || pagination || typeof total === "number") && (
        <div className="cp-table-footer">
          {typeof total === "number" ? (
            <span className="cp-table-footer__total">共 {total} 条记录</span>
          ) : (
            <span />
          )}
          <div className="cp-table-footer__pager">
            {pagination && pagination !== false ? (
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "12px",
                  fontSize: "12px",
                }}
              >
                <button
                  onClick={() => pagination.onChange(pagination.current - 1, pagination.pageSize)}
                  disabled={pagination.current <= 1}
                  style={{
                    padding: "4px 8px",
                    border: "1px solid #d9d9d9",
                    background: "#fff",
                    cursor: "pointer",
                    borderRadius: "2px",
                  }}
                >
                  上一页
                </button>
                <span>
                  {pagination.current} / {Math.ceil(pagination.total / pagination.pageSize)}
                </span>
                <button
                  onClick={() => pagination.onChange(pagination.current + 1, pagination.pageSize)}
                  disabled={pagination.current >= Math.ceil(pagination.total / pagination.pageSize)}
                  style={{
                    padding: "4px 8px",
                    border: "1px solid #d9d9d9",
                    background: "#fff",
                    cursor: "pointer",
                    borderRadius: "2px",
                  }}
                >
                  下一页
                </button>
              </div>
            ) : (
              footer && <div>{footer}</div>
            )}
          </div>
        </div>
      )}
    </section>
  );
}

/* ─── PortableAdminTable：示例渲染（保留向后兼容） ─── */

export function PortableAdminTable() {
  return (
    <PortableDataTable
      rowKey="id"
      total={36}
      columns={[
        { key: "name", title: "名称" },
        {
          key: "status",
          title: "状态",
          render: (_v, row: { status: string }) =>
            row.status === "已启用" ? (
              <PortableTableStatus tone="success">已启用</PortableTableStatus>
            ) : (
              <PortableTableStatus tone="muted">已停用</PortableTableStatus>
            ),
        },
        { key: "updated", title: "更新时间" },
        {
          key: "actions",
          title: "操作",
          render: () => (
            <PortableTableActions>
              <button className="cp-table-action">查看</button>
              <button className="cp-table-action">编辑</button>
            </PortableTableActions>
          ),
        },
      ]}
      dataSource={[
        { id: 1, name: "企业技能库", status: "已启用", updated: "2026-06-06 10:20" },
        { id: 2, name: "运维 Agent", status: "已停用", updated: "2026-06-05 16:42" },
        { id: 3, name: "通用语义模型", status: "已启用", updated: "2026-06-04 09:10" },
      ]}
    />
  );
}
