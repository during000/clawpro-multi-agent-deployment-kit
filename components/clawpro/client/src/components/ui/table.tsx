import * as React from "react";

import { cn } from "@/lib/utils";
import { Checkbox } from "@/components/ui/checkbox";
import { Pagination, type PaginationProps } from "@/components/ui/pagination";
import {
  Empty,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";

/* ════════════════════════════════════════════════════════════════════
 * Table 组件 · 设计规范（v2026.05）
 *
 * 本组件是企业管控端表格的"权威标准"。一切表格场景必须使用本组件 + Pagination
 * 组合，禁止直接写原生 <table> / <thead> / <tbody> / <tr> / <th> / <td>。
 *
 * 规范对应的 CSS 落点：client/src/index.css 「表格字号一致性规则」段。
 * ════════════════════════════════════════════════════════════════════
 *
 * §1. 密度 density（default | compact）
 * ──────────────────────────────────────────
 *   - default → 行高 54px，纵向 padding 12px
 *   - compact → 行高 40px，纵向 padding 8px
 *   两种密度仅在行高 / 纵向 padding 上区分，字号 / 横向 padding / 颜色保持一致。
 *
 *   Table 组件只负责表格结构与密度；分页器不属于 Table 内部能力。
 *   约定上，页面级标准表格通常搭配 Pagination 默认尺寸 `size="default"`；
 *   `size="small"` 更适合 Dialog / Drawer 等空间受限浮层中的表格分页。
 *
 * §2. 字号一致性（!important 全局强制）
 * ──────────────────────────────────────────
 *   表格相关所有元素字号统一 **12px / text-xs**，不区分密度。
 *   ⚠️ 最小字号规则：12px 是表格内的硬性下限，不会随页面缩放或响应式断点缩小。
 *      通过 !important + text-size-adjust: none 双重保障。
 *
 *   覆盖范围（全部 12px）：
 *     ① 表格单元格自身（5 类 data-slot）
 *        [data-slot="table"]
 *        [data-slot="table-head"]
 *        [data-slot="table-cell"]
 *        [data-slot="table-action-cell"]
 *        [data-slot="table-footer"]
 *
 *     ② 表格内任意后代元素（table[data-density] *）：
 *        Button / Input / Select / Switch / Checkbox / Label / Tooltip /
 *        code / pre / div / span / p / a / strong ... 全部强制 12px。
 *        业务侧即便手写 `text-sm` / `text-base` / inline style fontSize 也会被 !important 覆盖。
 *
 *     ③ 分页器（[data-slot="pagination"]）：
 *        Pagination 组件 simple / default 两种模式、size="default" / "small"
 *        都强制 12px。两种 size 仅按钮尺寸（32px / 24px）不同，字号一致。
 *
 *     ④ 数量统计 / 摘要文字：
 *        SurfaceCard 内、与 [data-slot="table-shell"] 同级的兄弟元素
 *        （例如「共 N 条记录」「最后更新于 ...」等表格底部说明文字）。
 *
 *   唯一豁免：**Badge** [data-slot="badge"] 始终保持自身尺寸，不被强制 12px。
 *
 *   ⚠️ 业务侧规范：
 *     - 不要在 TableCell 上手写 `text-sm` / `text-[14px]` / `text-gray-700` 来"调字号"，
 *       不仅冗余（被 !important 覆盖），还会让代码层不一致。
 *     - 字号要变化时，请改 index.css 的全局规则，不要在 TableCell 局部硬写。
 *
 * §3. 字色规范
 * ──────────────────────────────────────────
 *   - TableHead（表头）：
 *       default 密度 → #171717（gray-900）
 *       compact 密度 → #737373（gray-500，参考 Figma MetaMedium）
 *   - TableCell / TableActionCell（数据行）：
 *       默认强制 #0A0A0A（纯黑），即 Tailwind gray-950 / project foreground。
 *
 *   业务可在单列上覆盖为辅助灰（#737373 / #525252）来表达"次要信息"，
 *   但表头之外**默认全部纯黑** —— 不再像 v1 那样按"主/次列"切灰色。
 *
 * §4. 字体（PingFang SC）
 * ──────────────────────────────────────────
 *   全站字体已通过 index.css 的 `*:not(svg):not(svg *) { font-family: 'PingFang SC' ... !important }`
 *   强制统一为 PingFang SC，因此：
 *     - 表格内 monospace ID（例如 `ins-hermes01`）也会渲染为 PingFang SC，
 *       不需要在业务侧用 `font-mono` 类名维持等宽（且 font-mono 也会被字体规则覆盖）。
 *     - 极少数确实需要等宽的位置（例如纯英文/数字代码块）建议放在 SVG 或独立 inline style 中处理。
 *
 * §5. 操作列 TableActionCell
 * ──────────────────────────────────────────
 *   - 业务按钮统一使用 `<Button variant="link">` 文字按钮（品牌蓝），
 *     连"删除"等危险操作也用 link 蓝色，红/黑语义差异由文案 + 二次确认 Dialog 承载，
 *     **禁止再加 text-red-600 / text-red-700 / disabled:text-red-300 等覆盖**。
 *   - 内置 flex wrapper：项间距固定 24px (gap-6)，对齐 Figma 操作列规范。
 *   - 在横向滚动表格中，操作列必须 `fixed="right"` 钉在最右侧。
 *
 * §6. 固定列 / Fixed Columns（参考 Ant Design）
 * ──────────────────────────────────────────
 *   <Table> 组件默认 props：
 *     - scrollX={undefined}        → 默认按容器宽度自适应；内容放得下不出现横滚条
 *     - autoFixedColumns={true}    → 自动 sticky 首列（第一个 th/td）+ 操作列（TableActionCell）
 *
 *   ⚠️ 何时显式开启横滚兜底？
 *     列数较多 / 内容长度不可控的表格，**必须**传 `scrollX={1500}` 或 `scrollX="max-content"`，
 *     这样在窄屏 / 大表格时才会出现横向滚动条，并触发自动固定列的视觉效果。
 *     若表格列数固定且内容能放下（如「内置通道」7 行简单列表），无需传 scrollX，
 *     避免出现"内容明明能放下却出现横滚条"的尴尬。
 *
 *   显式 API：
 *     - <Table scrollX={1500}> 或 <Table scrollX="max-content">  开启横滚兜底
 *     - <Table autoFixedColumns={false}>                          关闭自动固定列
 *     - <TableHead fixed="left"> / <TableHead fixed="right">
 *     - <TableCell fixed="left"> / <TableCell fixed="right">
 *     - <TableActionCell fixed="right">
 *       业务显式声明 fixed 的列优先级更高，自动固定不会覆盖。
 *
 *   多列同侧固定（如复选框列 + 名称列同时 fixed="left"）：
 *     - 偏移自动化：组件在挂载/resize 时按 DOM 顺序自动累加同侧固定列宽度，
 *       写入各 cell 的 left/right，无需业务侧手写 style={{ left: 56 }}（任意非首列固定亦支持）。
 *     - 阴影：仅在最右侧的左固定列（或最左侧的右固定列）保留 `fixedShadow`，其余设 `fixedShadow={false}`。
 *
 *   阴影分隔线：自动固定与显式固定通用同一套规则
 *     - 最左：仅在已向右滚动时显示
 *     - 最右：仅在右侧仍有内容时显示
 *     - 无横滚：阴影全部隐藏
 *
 *   规则定义位置：
 *     - JS：本文件 Table 组件（scrollX / autoFixedColumns）
 *     - CSS：client/src/index.css「表格自动固定列规则（v2026.05）」段
 *
 * §7. 选中行 data-state="selected"
 * ──────────────────────────────────────────
 *   选中行**不再有背景高亮**（v2026.06 起全局移除蓝底）。
 *   选中状态请通过复选框（Checkbox）勾选态本身来表达，不要依赖行背景色。
 *   仍可给 <TableRow data-state="selected"> 标记语义，但默认不产生任何视觉背景；
 *   如需自定义选中视觉，请在业务侧自行通过 className 处理。
 *
 * §8. 与 Pagination 的搭配规范
 * ──────────────────────────────────────────
 *   推荐结构：
 *     <SurfaceCard>
 *       <Table>...</Table>
 *       <div className="px-4 py-3 border-t border-gray-200">
 *         <Pagination total={...} showTotal={(t) => `共 ${t} 条记录`} ... />
 *       </div>
 *     </SurfaceCard>
 *
 *   - Pagination 字号自动跟随表格（12px），无需在调用侧覆盖。
 *   - showTotal 文案统一「共 N 条记录」（中文逗号），不要写 "Total: N"。
 * ════════════════════════════════════════════════════════════════════ */

type TableDensity = "default" | "compact";

/**
 * 表格视觉变体（v2026.06 起收敛为 2 种）：
 *   - "white"        → 整体白色（表头 + body 全白）+ rounded-xl + 白描边浮起。
 *                      用于**非白色背景**（蓝色渐变 Hero / 灰色页面底）。
 *   - "gray-header"  → 灰色表头 var(--bg-grey-normal) + 白色 body。**默认值**。
 *                      用于**白色背景容器**（SurfaceCard 内、白底 Dialog 内等）。
 *
 * sticky 表头颜色随 variant 自动跟随（通过 var(--table-head-bg)）。
 */
type TableVariant = "white" | "gray-header";

/**
 * 已废弃的 variant 名（仅作内部 normalize 兼容，新代码禁止使用）：
 *   - "default"        → "gray-header"
 *   - "elevated-white" → "white"
 *   - "collapsible"    → "white"
 * 计划于下个版本移除。
 * @deprecated
 */
type LegacyTableVariant = "default" | "elevated-white" | "collapsible";

function normalizeVariant(v: TableVariant | LegacyTableVariant | undefined): TableVariant {
  if (v === "white" || v === "gray-header") return v;
  if (v === "elevated-white" || v === "collapsible") return "white";
  // "default" / undefined / 任何意外值 → 回落到 "gray-header"
  return "gray-header";
}

type FixedSide = "left" | "right";

type TableScrollState = {
  scrollableX: boolean;
  scrollLeft: boolean;
  scrollRight: boolean;
  scrollLeftValue: number;
};

const TableDensityContext = React.createContext<TableDensity>("default");
const TableVariantContext = React.createContext<TableVariant>("gray-header");

function useTableDensity() {
  return React.useContext(TableDensityContext);
}

function useTableVariant() {
  return React.useContext(TableVariantContext);
}

function assignRef<T>(ref: React.Ref<T> | undefined, value: T | null) {
  if (!ref) return;
  if (typeof ref === "function") {
    ref(value);
    return;
  }
  ref.current = value;
}

type TableProps = React.ComponentProps<"table"> & {
  containerClassName?: string;
  containerRef?: React.Ref<HTMLDivElement>;
  containerStyle?: React.CSSProperties;
  density?: TableDensity;
  /**
   * 视觉风格变体（v2026.06 起收敛为 2 种）：
   *   - "gray-header"（默认）：灰底表头 var(--bg-grey-normal) + 白色 body。
   *     适用于白色背景容器内（SurfaceCard / 白底 Dialog 等）。
   *   - "white"：整体白色（表头 + body 全白）+ rounded-xl + 白描边浮起。
   *     适用于非白色背景（蓝色渐变 Hero / 灰色页面底）。
   *
   * @deprecated 旧值 "default" / "elevated-white" / "collapsible" 仍受内部 normalize 兼容，
   * 但请尽快迁移到新值；下个版本将移除。
   *
   * ⚠️ variant="white" 禁止在 Dialog / AlertDialog / Sheet 等弹窗内使用，
   *    也禁止在白色背景容器上使用（白上加白看不见）。
   */
  variant?: TableVariant | LegacyTableVariant;
  /**
   * 与 Ant Design Table 的 scroll.x 一致：
   *   - 数字：表格最小宽度（px）；超出容器宽度即出现横向滚动条
   *   - 字符串：直接作为 min-width，例如 "max-content" / "1200px"
   *   - 不传：表格按容器宽度自适应（默认）—— 内容放得下时不出现横滚条
   *
   * 列数较多 / 可能溢出的表格请显式传 `scrollX={1500}` 或 `scrollX="max-content"` 启用横滚兜底。
   */
  scrollX?: number | string;
  /**
   * 是否自动固定首列与操作列（默认 true）。
   * 仅在表格触发横向滚动（即传入了 scrollX 且内容溢出）时视觉上有意义：
   *   - 每行第一个 TableHead / TableCell 自动 sticky 在左侧
   *   - 每行的 TableActionCell 自动 sticky 在右侧
   * 业务侧已显式声明 `fixed="left"` / `fixed="right"` 的列优先级更高，不被覆盖。
   * 若特殊场景需要关闭自动固定（如卡片型不滚动表），传 autoFixedColumns={false}。
   */
  autoFixedColumns?: boolean;
};

function Table({
  className,
  containerClassName,
  containerRef,
  containerStyle,
  density = "default",
  variant: variantInput,
  scrollX,
  autoFixedColumns = true,
  ...props
}: TableProps) {
  // 兼容历史 variant 名（"default" / "elevated-white" / "collapsible"），统一收敛到 "white" | "gray-header"
  const variant = normalizeVariant(variantInput);
  const tableMinWidth =
    typeof scrollX === "number" ? `${scrollX}px` : scrollX ?? undefined;
  const outerContainerRef = React.useRef<HTMLDivElement | null>(null);
  const containerNodeRef = React.useRef<HTMLDivElement | null>(null);
  const [scrollState, setScrollState] = React.useState<TableScrollState>({
    scrollableX: false,
    scrollLeft: false,
    scrollRight: false,
    scrollLeftValue: 0,
  });

  const setContainerNode = React.useCallback((node: HTMLDivElement | null) => {
    containerNodeRef.current = node;
    assignRef(containerRef, node);
  }, [containerRef]);

  React.useEffect(() => {
    const el = containerNodeRef.current;
    if (!el) return;

    let raf = 0;
    const measure = () => {
      raf = 0;
      const maxScrollLeft = Math.max(0, el.scrollWidth - el.clientWidth);
      const next: TableScrollState = {
        scrollableX: maxScrollLeft > 1,
        scrollLeft: el.scrollLeft > 1,
        scrollRight: el.scrollLeft < maxScrollLeft - 1,
        scrollLeftValue: el.scrollLeft,
      };

      /* ──────────────────────────────────────────────────────────────
       * 自动累加 fixed 列 offset（多列同侧固定的关键）
       * ────────────────────────────────────────────────────────────
       * 思路：对每一个 <tr> 单独处理（thead / tbody / tfoot 的 tr 都算）。
       * 同一行内：
       *   - 从左到右遍历，遇到 [data-fixed="left"] 的 cell：写入 style.left = 当前累加宽度，
       *     累加自身 offsetWidth 到下一列的偏移。
       *   - 从右到左遍历，遇到 [data-fixed="right"] 的 cell：写入 style.right = 当前累加宽度，
       *     累加自身 offsetWidth。
       * 这样业务侧只需写 fixed="left" / fixed="right"，无需手算 left:48px 等偏移；
       * 也支持任意非首列固定（之前 §6 写的"必须靠 CSS hack"问题），自动累加。
       * ────────────────────────────────────────────────────────────── */
      const tableEl = el.querySelector<HTMLTableElement>("table[data-slot='table']");
      if (tableEl) {
        const rows = tableEl.querySelectorAll<HTMLTableRowElement>("tr");
        rows.forEach((row) => {
          const cells = Array.from(row.children) as HTMLElement[];
          // 左侧累加
          let leftOffset = 0;
          for (const cell of cells) {
            if (cell.dataset.fixed === "left") {
              cell.style.left = `${leftOffset}px`;
              leftOffset += cell.offsetWidth;
            }
          }
          // 右侧累加（反向）
          let rightOffset = 0;
          for (let i = cells.length - 1; i >= 0; i--) {
            const cell = cells[i];
            if (cell.dataset.fixed === "right") {
              cell.style.right = `${rightOffset}px`;
              rightOffset += cell.offsetWidth;
            }
          }
        });
      }

      setScrollState((prev) => (
        prev.scrollableX === next.scrollableX &&
        prev.scrollLeft === next.scrollLeft &&
        prev.scrollRight === next.scrollRight &&
        Math.abs(prev.scrollLeftValue - next.scrollLeftValue) < 0.5
          ? prev
          : next
      ));
    };

    const requestMeasure = () => {
      if (raf) cancelAnimationFrame(raf);
      raf = requestAnimationFrame(measure);
    };

    requestMeasure();
    el.addEventListener("scroll", requestMeasure, { passive: true });
    window.addEventListener("resize", requestMeasure);

    const ro = new ResizeObserver(requestMeasure);
    ro.observe(el);
    if (el.firstElementChild) ro.observe(el.firstElementChild);

    return () => {
      if (raf) cancelAnimationFrame(raf);
      el.removeEventListener("scroll", requestMeasure);
      window.removeEventListener("resize", requestMeasure);
      ro.disconnect();
    };
  }, [tableMinWidth, autoFixedColumns]);

  return (
    <TableDensityContext.Provider value={density}>
    <TableVariantContext.Provider value={variant}>
      <div
        ref={outerContainerRef}
        data-slot="table-shell"
        className={cn(
          "relative isolate w-full",
          // variant="white"：整体白色卡片（thead 单独有 bg；body / 行间隙的白底由这里的 bg 提供，
          // 否则放在渐变 / 灰底容器上 body 区域会透出底色，呈现"空白"假象）
          variant === "white" && "rounded-xl border border-[var(--bg-white)] overflow-hidden bg-[var(--bg-white)]"
        )}
      >
        <div
          ref={setContainerNode}
          data-density={density}
          data-variant={variant}
          data-slot="table-container"
          data-scrollable-x={scrollState.scrollableX ? "true" : "false"}
          data-scroll-left={scrollState.scrollLeft ? "true" : "false"}
          data-scroll-right={scrollState.scrollRight ? "true" : "false"}
          className={cn(
            "relative w-full overflow-x-auto",
            // 横向滚动模式下：滚动条默认隐藏，hover 表格区域或正在滚动时才出现（复用全局 .scrollbar-on-hover 工具类）
            tableMinWidth && "scrollbar-on-hover",
            containerClassName
          )}
          style={containerStyle}
        >
          <table
            data-density={density}
            data-slot="table"
            data-auto-fixed={autoFixedColumns ? "true" : "false"}
            className={cn(
              "w-full caption-bottom font-sans leading-[1.5] text-gray-900 text-xs",
              // 固定列要求 table 不能使用 collapse，否则 sticky 单元格的边框/背景会出现间隙
              tableMinWidth ? "border-separate border-spacing-0" : "",
              className
            )}
            style={tableMinWidth ? { minWidth: tableMinWidth } : undefined}
            {...props}
          />
        </div>
      </div>
    </TableVariantContext.Provider>
    </TableDensityContext.Provider>
  );
}

function TableHeader({ className, style, ...props }: React.ComponentProps<"thead">) {
  const variant = useTableVariant();
  // 通过 CSS 变量 --table-head-bg 把 thead 底色暴露给后代固定列单元格，
  // 让 sticky 表头单元格的不透明背景能自动跟随 variant，而不是写死白色（修复"一块白一块灰"色块割裂 bug）。
  //   - "white"        → var(--bg-white)
  //   - "gray-header"  → var(--bg-grey-normal)
  const headBg = variant === "white" ? "var(--bg-white)" : "var(--bg-grey-normal)";
  return (
    <thead
      data-slot="table-header"
      data-variant={variant}
      style={{ ["--table-head-bg" as any]: headBg, ...style }}
      className={cn(
        "bg-[var(--table-head-bg)]",
        "[&_tr]:border-b [&_tr]:border-gray-200",
        className
      )}
      {...props}
    />
  );
}

function TableBody({ className, ...props }: React.ComponentProps<"tbody">) {
  return (
    <tbody
      data-slot="table-body"
      className={cn("[&_tr:last-child]:border-0", className)}
      {...props}
    />
  );
}

function TableFooter({ className, ...props }: React.ComponentProps<"tfoot">) {
  return (
    <tfoot
      data-slot="table-footer"
      className={cn(
        "bg-[var(--bg-grey-normal)] border-t border-gray-200 font-sans font-medium leading-[1.5] text-gray-900 text-xs [&>tr]:last:border-b-0",
        className
      )}
      {...props}
    />
  );
}

/* ────────────────────────────────────────────────────────────────────
 * TableRow
 * 加 `group` class 是为了让固定单元格通过 group-hover / group-data-[state=selected]
 * 同步行的 hover / selected 背景色，避免固定列出现"白条不变色"问题。
 * ──────────────────────────────────────────────────────────────────── */
function TableRow({ className, ...props }: React.ComponentProps<"tr">) {
  return (
    <tr
      data-slot="table-row"
      className={cn(
        "group border-b border-gray-200 transition-colors hover:bg-[var(--bg-grey-hover-subtle)] [thead_&]:hover:bg-transparent",
        className
      )}
      {...props}
    />
  );
}

/**
 * TableExpandedRow - 展开行（用于行展开/折叠场景的表格，常配合 variant="white"）
 *
 * 特点：白色背景、禁用 hover 态，用于展示行展开后的子内容。
 */
function TableExpandedRow({ className, ...props }: React.ComponentProps<"tr">) {
  return (
    <tr
      data-slot="table-expanded-row"
      className={cn(
        "border-b border-gray-200 [&_td]:!bg-[var(--bg-white)] [&_td]:group-hover:!bg-[var(--bg-white)]",
        className
      )}
      style={{ backgroundColor: 'var(--bg-white)' }}
      {...props}
    />
  );
}

/* ────────────────────────────────────────────────────────────────────
 * 固定列样式 token
 *   - left:  z-index 较高、靠左 sticky；右侧仅以单元格锚定投影提示边界
 *   - right: 同上对称；左侧以投影提示边界
 *
 *  边界视觉使用与 SKILL §15 一致的 token：
 *    before（CSS）→ 6px 渐变投影（rgba(0,0,0,0.06) → transparent），由 index.css 锚定在
 *                  固定单元格的 ::before 上（top:0 / bottom:-1px 满高跨行相连），
 *                  纵向连续不分段、整列首尾无空白。不再使用容器级游离 overlay（旧实现会
 *                  错位留缝或叠加变重）。参考 Ant Design fixed columns。
 *
 *  ⚠️ v2026.06：固定列**不再画 1px 硬分隔线**（旧实现用 ::after 画 #EAEEF4 竖线）。
 *     表格列与列之间本就无竖线，固定列额外加硬线会显得突兀（像多出来的边框），
 *     故只保留 ::before 柔和投影作为边界提示。
 *
 *  注：投影根据横向滚动状态显示（left 仅在已向右滚动时出现；right 仅在右侧仍有内容时出现）。
 *
 *  多列固定（如复选框列 + 名称列同时 fixed="left"）：
 *      只在最右侧那个左固定列（或最左侧那个右固定列）保留阴影。
 *      通过 `fixedShadow={false}` 关闭中间列的阴影（同时不写 data-fixed-shadow，故无投影）。
 * ──────────────────────────────────────────────────────────────────── */
const FIXED_BASE = "sticky";
// 表头固定列：z-50 必须高于业务表头里常见的 `relative z-40`（如带筛选 Popover 的列）以及任何 body cell
// 背景不写死 —— 通过 var(--table-head-bg) 自动跟随 TableHeader 的 variant：
//   - gray-header → var(--bg-grey-normal)
//   - white       → var(--bg-white)
// 这样首列表头不会再出现"一块白一块灰"色块割裂。
// fallback 用 var(--bg-grey-normal)（默认 gray-header 表头色），与普通列 TableHead 的背景表达式
// 完全一致 —— 保证即便 thead 未注入 --table-head-bg（如业务直接写原生 <thead>），
// 普通列与固定列表头颜色仍统一，不会割裂。
const FIXED_LEFT_CLS = "left-0 z-50 bg-[var(--table-head-bg,var(--bg-grey-normal))]";
const FIXED_RIGHT_CLS = "right-0 z-50 bg-[var(--table-head-bg,var(--bg-grey-normal))]";
// 固定列边界**只保留柔和渐变阴影**（由 index.css 的 [data-fixed-shadow] ::before 锚定渲染），
// 不再额外画 1px 硬分隔线 —— 表格内部其它列之间本就无竖线，那条硬线会显得突兀（像多出来的边框）。
// data-fixed-shadow 属性仍由各 cell 单独设置以驱动 ::before 阴影，故此处仅留空字符串占位。
// 参考 Ant Design 固定列：仅阴影、无分隔线。
const FIXED_LEFT_SHADOW_CLS = "";
const FIXED_RIGHT_SHADOW_CLS = "";

// body 单元格的固定列样式：白底（var(--bg-white)）+ 跟随行 hover
// z-20 高于普通 body cell（z auto），避免横向滚动时被相邻列内容穿透
// 注：选中态行背景已全局移除（不再有蓝底），固定列只需跟随 hover
const FIXED_LEFT_CELL_CLS =
  "left-0 z-20 bg-[var(--bg-white)] transition-colors " +
  "group-hover:bg-[var(--bg-grey-hover-subtle)]";
const FIXED_RIGHT_CELL_CLS =
  "right-0 z-20 bg-[var(--bg-white)] transition-colors " +
  "group-hover:bg-[var(--bg-grey-hover-subtle)]";
// body 边界列同样不画 1px 分隔线，仅由 index.css 的 ::before 提供柔和渐变阴影（见上方说明）。
const FIXED_LEFT_CELL_SHADOW_CLS = "";
const FIXED_RIGHT_CELL_SHADOW_CLS = "";

/**
 * TableHead - 表头单元格（强制样式）
 *
 * 规范（严格遵循 §1 / §2 / §3）：
 * - 背景：bg-[var(--bg-grey-normal)]，继承 TableHeader
 * - 字号：12px / Medium（不区分密度，全局 !important 强制 §2）
 * - 字色：default → #171717；compact → #737373
 * - 表头高度：default 54px / compact 40px（§1）
 * - 横向 padding：统一 px-4（16px）
 * - 默认对齐：text-left align-middle，可按列覆盖 text-right
 * - 不换行：whitespace-nowrap
 *
 * Props：
 *   - fixed?: "left" | "right"     固定该列；必须配合 <Table scrollX={...}>（§6）
 *   - fixedShadow?: boolean        是否允许边界分隔线 + 滚动阴影，默认 true
 *
 * className 主要用于控制宽度（w-[xx%]）、sticky 偏移和必要的列对齐。
 * 每列标题和内容必须统一左对齐。
 */
type TableHeadProps = React.ComponentProps<"th"> & {
  fixed?: FixedSide;
  fixedShadow?: boolean;
};

function TableHead({ className, fixed, fixedShadow = true, style, ...props }: TableHeadProps) {
  const density = useTableDensity();

  // 显式 fixed 列：用 inline style 兜底注入 position/zIndex，
  // 避免上层用户传入的 className（如 w-[xxx]px / !bg-... / Tailwind 工具类）
  // 与 cn() 合并时被 tailwind-merge 误判覆盖（曾出现 sticky 类被丢失的现象）。
  // left/right 偏移由 Table 组件的 measure useEffect 在挂载后写入 style.left/right，
  // 支持多列同侧固定时按 DOM 顺序自动累加宽度（无需业务侧手算 left:48px 等偏移）。
  const fixedStyle: React.CSSProperties | undefined = fixed
    ? { position: "sticky", zIndex: 50 }
    : undefined;

  return (
    <th
      data-slot="table-head"
      data-fixed={fixed}
      data-fixed-shadow={fixed && fixedShadow ? fixed : undefined}
      style={{ ...fixedStyle, ...style }}
      className={cn(
        "text-left align-middle font-sans whitespace-nowrap text-xs font-medium leading-[1.5] bg-[var(--table-head-bg,var(--bg-grey-normal))] [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-[2px]",
        density === "compact"
          ? "h-9 px-4 py-0 text-gray-700"
          : "h-10 px-4 py-0 text-gray-700",
        // separate 模式下 <tr> border-b 会失效，由单元格自身补一条下分隔线（仅在 separate 模式下生效）
        "[table.border-separate_&]:border-b [table.border-separate_&]:border-gray-200",
        fixed === "left" && [FIXED_BASE, FIXED_LEFT_CLS],
        fixed === "right" && [FIXED_BASE, FIXED_RIGHT_CLS],
        fixed === "left" && fixedShadow && FIXED_LEFT_SHADOW_CLS,
        fixed === "right" && fixedShadow && FIXED_RIGHT_SHADOW_CLS,
        className
      )}
      {...props}
    />
  );
}

type TableCellProps = React.ComponentProps<"td"> & {
  fixed?: FixedSide;
  fixedShadow?: boolean;
};

function TableCell({ className, fixed, fixedShadow = true, style, ...props }: TableCellProps) {
  const density = useTableDensity();

  // 见 TableHead 同处注释：用 inline style 兜底注入 sticky 行为，
  // left/right 由 Table 组件 measure 后注入，支持多列同侧固定自动累加偏移。
  const fixedStyle: React.CSSProperties | undefined = fixed
    ? { position: "sticky", zIndex: 20 }
    : undefined;

  return (
    <td
      data-slot="table-cell"
      data-fixed={fixed}
      data-fixed-shadow={fixed && fixedShadow ? fixed : undefined}
      // 注意：不要再为 fixed cell 注入 inline `backgroundColor: 'white'`！
      // 它的优先级会盖过 :hover / :data-[state=selected] 的类，导致 sticky 列在
      // 行 hover / 选中时不变色（出现"白条不跟随行变色"现象）。
      // 默认白底已由 FIXED_LEFT_CELL_CLS / FIXED_RIGHT_CELL_CLS 中的
      // `bg-[var(--bg-white)]` 提供，并由 group-hover / group-data-[state=selected]
      // 同步到 hover / selected 配色（与 §15 描述一致）。
      style={{ ...fixedStyle, ...style }}
      className={cn(
        "text-left align-middle whitespace-nowrap font-sans font-normal leading-[1.5] text-gray-950 text-xs transition-colors group-hover:bg-[var(--bg-grey-hover-subtle)] [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-[2px]",
        density === "compact" ? "h-10 px-4 py-2" : "h-[54px] px-4 py-3",
        // separate 模式下补下分隔线（默认 collapse 模式由 <tr> border-b 接管）
        "[table.border-separate_&]:border-b [table.border-separate_&]:border-gray-200",
        // separate 模式下，tbody 最后一行单元格不画底边，避免与外层卡片底边重合（与 collapse 模式 `[&_tr:last-child]:border-0` 行为对齐）
        "[table.border-separate_tbody_tr:last-child_&]:border-b-0",
        fixed === "left" && [FIXED_BASE, FIXED_LEFT_CELL_CLS],
        fixed === "right" && [FIXED_BASE, FIXED_RIGHT_CELL_CLS],
        fixed === "left" && fixedShadow && FIXED_LEFT_CELL_SHADOW_CLS,
        fixed === "right" && fixedShadow && FIXED_RIGHT_CELL_SHADOW_CLS,
        className
      )}
      {...props}
    />
  );
}

/**
 * TableActionCell - 表格操作列专用单元格
 *
 * 规范（参见顶部 §5 + §3 + §6）：
 *   - 字号 12px / 字色 #0A0A0A（被 §2 全局 !important 强制覆盖）
 *   - 内置 flex wrapper：项间距固定 24px (gap-6)，对齐 Figma 操作列规范
 *   - 业务按钮**必须**显式声明 `variant="link"`（品牌蓝文字按钮）：
 *       连「删除」等危险操作也用 link 蓝色，不再以红/黑区分语义；
 *       语义差异由文案 + 二次确认 Dialog 承载。
 *       ❌ 禁止 `text-red-600` / `text-red-700` / `disabled:text-red-300` 等覆盖。
 *   - 横向滚动表格中操作列必须 `fixed="right"`
 *
 * Props：
 *   - fixed?: "left" | "right"
 *   - fixedShadow?: boolean        默认 true
 *   - rawChildren?: boolean        关闭内置 flex wrapper（默认 false）
 *   - actionsClassName?: string    flex wrapper 的额外 className
 *
 * 用法：
 *   <TableActionCell>
 *     <Button variant="link" onClick={onEdit}>编辑</Button>
 *     <Button variant="link" onClick={onDelete}>删除</Button>
 *   </TableActionCell>
 */
type TableActionCellProps = React.ComponentProps<"td"> & {
  fixed?: FixedSide;
  fixedShadow?: boolean;
  /** 关闭内置 flex wrapper，直接渲染 children（默认 false） */
  rawChildren?: boolean;
  /** 内置 flex wrapper 的额外 className（如 h-5 / whitespace-nowrap） */
  actionsClassName?: string;
};

function TableActionCell({
  className,
  fixed,
  fixedShadow = true,
  rawChildren = false,
  actionsClassName,
  children,
  style,
  ...props
}: TableActionCellProps) {
  const density = useTableDensity();

  // 见 TableHead 同处注释：用 inline style 兜底注入 sticky 行为，
  // left/right 由 Table 组件 measure 后注入。
  const fixedStyle: React.CSSProperties | undefined = fixed
    ? { position: "sticky", zIndex: 20 }
    : undefined;

  return (
    <td
      data-slot="table-action-cell"
      data-fixed={fixed}
      data-fixed-shadow={fixed && fixedShadow ? fixed : undefined}
      style={{ ...fixedStyle, ...style }}
      // 见 TableCell 同处注释：fixed cell 的白底由 FIXED_*_CELL_CLS 提供，
      // 禁止注入 inline backgroundColor，否则会覆盖 hover / selected 配色。
      className={cn(
        "align-middle whitespace-nowrap font-sans font-normal leading-[1.5] text-gray-950 text-xs [&:has([role=checkbox])]:pr-0",
        density === "compact" ? "h-10 px-4 py-2" : "h-[54px] px-4 py-3",
        // separate 模式下补下分隔线（默认 collapse 模式由 <tr> border-b 接管）
        "[table.border-separate_&]:border-b [table.border-separate_&]:border-gray-200",
        // separate 模式下，tbody 最后一行单元格不画底边，避免与外层卡片底边重合
        "[table.border-separate_tbody_tr:last-child_&]:border-b-0",
        fixed === "left" && [FIXED_BASE, FIXED_LEFT_CELL_CLS],
        fixed === "right" && [FIXED_BASE, FIXED_RIGHT_CELL_CLS],
        fixed === "left" && fixedShadow && FIXED_LEFT_CELL_SHADOW_CLS,
        fixed === "right" && fixedShadow && FIXED_RIGHT_CELL_SHADOW_CLS,
        className
      )}
      {...props}
    >
      {rawChildren ? (
        children
      ) : (
        // 内置 flex 容器：项间距固定 24px (gap-6)，与 Figma 操作列规范对齐
        <div className={cn("relative z-10 flex items-center gap-6 whitespace-nowrap", actionsClassName)}>
          {children}
        </div>
      )}
    </td>
  );
}

function TableCaption({
  className,
  ...props
}: React.ComponentProps<"caption">) {
  return (
    <caption
      data-slot="table-caption"
      className={cn("mt-4 font-sans text-xs font-normal leading-[1.5] text-gray-500", className)}
      {...props}
    />
  );
}

export {
  Table,
  TableHeader,
  TableBody,
  TableFooter,
  TableHead,
  TableRow,
  TableExpandedRow,
  TableCell,
  TableActionCell,
  TableCaption,
};

/* ════════════════════════════════════════════════════════════════════
 * §9. DataTable · 数据驱动表格壳子（v2026.06）
 *
 * 在上面这套组合式底座之上再封一层"标准列表页"声明式 API，对齐 Ant
 * Design `<Table columns dataSource>` 风格，把以下 7 个状态机一次性内聚：
 *
 *   1. selectedKeys + 全选 + 跨页保留
 *   2. pagination + border-t 内嵌
 *   3. loading mask（不锁页）
 *   4. 空态自动 colSpan 渲染
 *   5. rowKey 取值
 *   6. 选中行 data-state="selected"
 *   7. 操作/选择列 fixed
 *
 * 与底座的关系（双层 API）：
 *   - 90% 列表页用 <DataTable>（强制）
 *   - 嵌套表 / colSpan 合并 / 行内编辑 / 单元格 mini 图表 等特殊场景
 *     回落到 <Table><TableRow><TableCell> 底座（逃生口）
 *
 * 视觉资产 100% 继承 Table（12px 强制 / autoFixedColumns / 容器级阴影
 * / --bg-brand-selected 换皮）。本组件不引入任何新视觉变量。
 *
 * 详见 .codebuddy/skills/clawpro-portable-design-skill/component-specs/data-table.md
 * ════════════════════════════════════════════════════════════════════ */

/* ─── DataTable Types ───────────────────────────────────────────────── */

export type DataTableRowKey<T> = keyof T | ((record: T, index: number) => string);

export type DataTableSize = "default" | "compact";
export type DataTableVariant = "white" | "gray-header";

export interface DataTableColumn<T> {
  /** 唯一 key，必填 */
  key: string;
  /** 表头内容 */
  title?: React.ReactNode;
  /** 取值字段；与 render 二选一 */
  dataIndex?: keyof T;
  /** 自定义渲染 */
  render?: (value: unknown, record: T, index: number) => React.ReactNode;
  /** 列宽，传给 <th>/<td> 的 width */
  width?: number | string;
  /** 列对齐，默认 left */
  align?: "left" | "center" | "right";
  /** 固定列；不传则依赖 Table 的 autoFixedColumns 自动钉首列 */
  fixed?: "left" | "right";
  /** 表头与单元格通用 className */
  className?: string;
  /** 单元格级别 className 函数 */
  cellClassName?: (record: T, index: number) => string;
}

export interface DataTableRowSelection<T> {
  /** 受控选中 keys */
  selectedKeys: string[];
  /** 选中变化回调；rows 仅包含当前页能匹配到的行（跨页保留时无法还原全部） */
  onChange: (keys: string[], rows: T[]) => void;
  /** 是否跨页保留选中，默认 true */
  preserveSelectedKeys?: boolean;
  /** 行级禁用控制 */
  getCheckboxProps?: (record: T) => { disabled?: boolean };
  /** 选择类型，默认 checkbox */
  type?: "checkbox" | "radio";
  /** 选择列宽度，默认 48px */
  columnWidth?: number;
  /** 选择列固定方向，默认 "left"；传 false 关闭固定 */
  columnFixed?: "left" | "right" | false;
}

export interface DataTableProps<T> {
  /** 列定义 */
  columns: DataTableColumn<T>[];
  /** 数据源 */
  dataSource: T[];
  /**
   * 行 key（必填）。强约束以避免 React key 散在各处导致选中态错位。
   *   - string：按字段名取值，如 "id"
   *   - function：自定义计算
   */
  rowKey: DataTableRowKey<T>;
  /** 整表 loading 遮罩（不锁分页/表头） */
  loading?: boolean;
  /** 分页：传对象时内部渲染 <Pagination> 在 border-t 下；false / 不传则不渲染分页 */
  pagination?: false | PaginationProps;
  /** 行选择 */
  rowSelection?: DataTableRowSelection<T>;
  /** 密度，透传到 Table */
  size?: DataTableSize;
  /** 视觉变体，透传到 Table */
  variant?: DataTableVariant;
  /** 横向滚动兜底，透传到 Table */
  scrollX?: number | string;
  /** 自动固定首列 / 操作列，透传到 Table */
  autoFixedColumns?: boolean;
  /** 空态自定义内容；不传时渲染默认 EmptyState */
  emptyText?: React.ReactNode;
  /** 自带 1px 描边 + 圆角外壳（不需要 SurfaceCard 时使用） */
  bordered?: boolean;
  /** 外层 className */
  className?: string;
  /** 行级事件 / 类名挂载点 */
  onRow?: (
    record: T,
    index: number
  ) => Omit<React.HTMLAttributes<HTMLTableRowElement>, "key">;
  /** 行级 className */
  rowClassName?: string | ((record: T, index: number) => string);
}

/* ─── DataTable Helpers ─────────────────────────────────────────────── */

function resolveRowKey<T>(rowKey: DataTableRowKey<T>, record: T, index: number): string {
  if (typeof rowKey === "function") return rowKey(record, index);
  return String(record[rowKey]);
}

function isCheckedTrue(v: boolean | "indeterminate"): v is true {
  return v === true;
}

/* ─── DataTable Component ───────────────────────────────────────────── */

function DataTable<T>({
  columns,
  dataSource,
  rowKey,
  loading = false,
  pagination,
  rowSelection,
  size = "default",
  variant = "gray-header",
  scrollX,
  autoFixedColumns = true,
  emptyText,
  bordered = false,
  className,
  onRow,
  rowClassName,
}: DataTableProps<T>) {
  const isEmpty = dataSource.length === 0 && !loading;

  /* ─ Selection ────────────────────────────────── */
  const selectionType = rowSelection?.type ?? "checkbox";
  const selectedKeys = rowSelection?.selectedKeys ?? [];
  const selectedSet = React.useMemo(() => new Set(selectedKeys), [selectedKeys]);
  const preserve = rowSelection?.preserveSelectedKeys ?? true;

  // 当前页未禁用的行 keys（用于全选/部分选状态计算）
  const currentPageEnabledKeys = React.useMemo(() => {
    if (!rowSelection) return [];
    return dataSource
      .filter((r) => !rowSelection.getCheckboxProps?.(r).disabled)
      .map((r, i) => resolveRowKey(rowKey, r, i));
  }, [dataSource, rowKey, rowSelection]);

  const headerChecked =
    selectionType === "checkbox" &&
    currentPageEnabledKeys.length > 0 &&
    currentPageEnabledKeys.every((k) => selectedSet.has(k));
  const headerIndeterminate =
    selectionType === "checkbox" &&
    !headerChecked &&
    currentPageEnabledKeys.some((k) => selectedSet.has(k));

  const handleHeaderToggle = (next: boolean) => {
    if (!rowSelection) return;
    if (next) {
      const merged = preserve
        ? Array.from(new Set([...selectedKeys, ...currentPageEnabledKeys]))
        : [...currentPageEnabledKeys];
      rowSelection.onChange(merged, dataSource);
    } else {
      const pageSet = new Set(currentPageEnabledKeys);
      const removed = selectedKeys.filter((k) => !pageSet.has(k));
      rowSelection.onChange(removed, []);
    }
  };

  const handleRowToggle = (record: T, index: number, next: boolean) => {
    if (!rowSelection) return;
    const k = resolveRowKey(rowKey, record, index);
    if (selectionType === "radio") {
      if (next) rowSelection.onChange([k], [record]);
      else rowSelection.onChange([], []);
      return;
    }
    if (next) {
      const merged = Array.from(new Set([...selectedKeys, k]));
      rowSelection.onChange(merged, [record]);
    } else {
      const removed = selectedKeys.filter((x) => x !== k);
      rowSelection.onChange(removed, []);
    }
  };

  /* ─ Render ────────────────────────────────────── */
  const totalColSpan = columns.length + (rowSelection ? 1 : 0);
  const selectionFixed =
    rowSelection?.columnFixed === false
      ? undefined
      : rowSelection?.columnFixed ?? "left";
  const selectionWidth = rowSelection?.columnWidth ?? 48;

  return (
    <div
      data-slot="data-table"
      className={cn(
        "relative",
        bordered &&
          "rounded-[4px] border border-gray-200 overflow-hidden bg-white",
        className
      )}
    >
      <div className="relative">
        <Table
          variant={variant}
          density={size}
          scrollX={scrollX}
          autoFixedColumns={autoFixedColumns}
        >
          <TableHeader>
            <TableRow>
              {rowSelection && (
                <TableHead
                  fixed={selectionFixed}
                  fixedShadow={false}
                  style={{ width: selectionWidth }}
                >
                  {selectionType === "checkbox" && (
                    <Checkbox
                      checked={
                        headerChecked
                          ? true
                          : headerIndeterminate
                          ? "indeterminate"
                          : false
                      }
                      onCheckedChange={(v) => handleHeaderToggle(v === true)}
                      disabled={currentPageEnabledKeys.length === 0}
                      aria-label="全选当前页"
                    />
                  )}
                </TableHead>
              )}

              {columns.map((col) => (
                <TableHead
                  key={col.key}
                  fixed={col.fixed}
                  className={cn(
                    col.align === "right" && "text-right",
                    col.align === "center" && "text-center",
                    col.className
                  )}
                  style={col.width ? { width: col.width } : undefined}
                >
                  {col.title}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>

          <TableBody>
            {!isEmpty &&
              dataSource.map((record, index) => {
                const k = resolveRowKey(rowKey, record, index);
                const isSelected = selectedSet.has(k);
                const checkboxDisabled =
                  rowSelection?.getCheckboxProps?.(record).disabled ?? false;
                const rowProps = onRow?.(record, index) ?? {};
                const rowCls =
                  typeof rowClassName === "function"
                    ? rowClassName(record, index)
                    : rowClassName;

                return (
                  <TableRow
                    key={k}
                    data-state={isSelected ? "selected" : undefined}
                    className={rowCls}
                    {...rowProps}
                  >
                    {rowSelection && (
                      <TableCell
                        fixed={selectionFixed}
                        fixedShadow={false}
                        style={{ width: selectionWidth }}
                        onClick={(e) => e.stopPropagation()}
                      >
                        {selectionType === "radio" ? (
                          <input
                            type="radio"
                            className="size-4 accent-[#355EF1] cursor-pointer disabled:cursor-not-allowed disabled:opacity-50"
                            checked={isSelected}
                            disabled={checkboxDisabled}
                            onChange={(e) =>
                              handleRowToggle(record, index, e.target.checked)
                            }
                            aria-label="选择该行"
                          />
                        ) : (
                          <Checkbox
                            checked={isSelected}
                            disabled={checkboxDisabled}
                            onCheckedChange={(v) =>
                              handleRowToggle(record, index, isCheckedTrue(v))
                            }
                            aria-label="选择该行"
                          />
                        )}
                      </TableCell>
                    )}

                    {columns.map((col) => {
                      const value =
                        col.dataIndex !== undefined
                          ? record[col.dataIndex]
                          : undefined;
                      const content = col.render
                        ? col.render(value, record, index)
                        : (value as React.ReactNode);
                      const cellCls =
                        typeof col.cellClassName === "function"
                          ? col.cellClassName(record, index)
                          : undefined;

                      return (
                        <TableCell
                          key={col.key}
                          fixed={col.fixed}
                          className={cn(
                            col.align === "right" && "text-right",
                            col.align === "center" && "text-center",
                            col.className,
                            cellCls
                          )}
                          style={col.width ? { width: col.width } : undefined}
                        >
                          {content as React.ReactNode}
                        </TableCell>
                      );
                    })}
                  </TableRow>
                );
              })}

            {isEmpty && (
              <TableRow className="hover:!bg-transparent">
                <TableCell
                  colSpan={totalColSpan}
                  className="!h-auto !p-0 hover:!bg-transparent"
                >
                  {emptyText ?? (
                    <Empty className="border-0 py-10">
                      <EmptyHeader>
                        <EmptyMedia />
                        <EmptyTitle>暂无数据</EmptyTitle>
                      </EmptyHeader>
                    </Empty>
                  )}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>

        {loading && (
          <div
            data-slot="data-table-loading"
            aria-busy="true"
            className={cn(
              "absolute inset-0 z-40 flex items-center justify-center",
              "bg-white/60 backdrop-blur-[1px]"
            )}
          >
            <div
              role="status"
              aria-label="加载中"
              className="size-6 animate-spin rounded-full border-2 border-[var(--border-control)] border-t-[#355EF1]"
            />
          </div>
        )}
      </div>

      {pagination !== false && pagination && (
        <div className="px-4 py-2 border-t border-gray-200">
          <Pagination {...pagination} />
        </div>
      )}
    </div>
  );
}

export { DataTable };
