/**
 * Portable Chart / Stat — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓图表卡片的可移植兜底外壳（卡壳 + 图例 + Tooltip + 各态）。
 *  - 不依赖具体图表库；宿主仓保留自有 recharts / echarts，仅复用本卡壳、
 *    配色（var(--cp-chart-*)）、Tooltip、Legend、空 / 加载 / 错误 / 无权限态。
 *  - 不渲染真实坐标系；把图表实例作为 children 传进 PortableChartCard 即可。
 *  - 视觉规范（component-specs/chart-stat.md §3）：
 *      主色 var(--cp-brand-blue)（chart-1）；辅助线弱灰；网格细弱；
 *      Tooltip 白底 4px overlay shadow 12-14px；Legend 小色块 + 文本；
 *      delta 趋势用语义色且同时给 ↑ / ↓ 符号（不只靠颜色）。
 *  - 顶部 KPI 用 PortableNumberCard（见 number-card.tsx / number-card.css），
 *    不要在此再拼装散装 StatCard。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/chart-stat.css";
 *
 * 用法：
 *   <PortableChartCard
 *     title="请求趋势"
 *     subtitle="近 7 天"
 *     legend={[{ label: "请求数", color: "var(--cp-chart-1)", value: "1,841" }]}
 *     explanation="数据每 5 分钟刷新一次。"
 *     state={loading ? "loading" : data.length ? "default" : "empty"}
 *   >
 *     <MyLineChart data={data} />
 *   </PortableChartCard>
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

/* ───────────── Legend ───────────── */

export interface PortableChartLegendItem {
  label: React.ReactNode;
  /** 色块颜色，默认 var(--cp-chart-1) */
  color?: string;
  value?: React.ReactNode;
  muted?: boolean;
}

export function PortableChartLegend({
  items,
  className = "",
}: {
  items: PortableChartLegendItem[];
  className?: string;
}) {
  const cls = ["cp-chart-legend", className].filter(Boolean).join(" ");
  return (
    <div className={cls}>
      {items.map((it, i) => (
        <span
          key={i}
          className={["cp-chart-legend__item", it.muted && "cp-chart-legend__item--muted"]
            .filter(Boolean)
            .join(" ")}
        >
          <span className="cp-chart-legend__swatch" style={it.color ? { background: it.color } : undefined} />
          <span>{it.label}</span>
          {it.value != null && <span className="cp-chart-legend__value">{it.value}</span>}
        </span>
      ))}
    </div>
  );
}

/* ───────────── Tooltip（供宿主图表库 content 插槽复用） ───────────── */

export interface PortableChartTooltipRow {
  label: React.ReactNode;
  value: React.ReactNode;
  color?: string;
}

export function PortableChartTooltip({
  label,
  rows,
}: {
  label?: React.ReactNode;
  rows: PortableChartTooltipRow[];
}) {
  return (
    <div className="cp-chart-tooltip">
      {label != null && <div className="cp-chart-tooltip__label">{label}</div>}
      {rows.map((r, i) => (
        <div key={i} className="cp-chart-tooltip__row">
          <span className="cp-chart-tooltip__series">
            <span className="cp-chart-tooltip__dot" style={r.color ? { background: r.color } : undefined} />
            {r.label}
          </span>
          <span className="cp-chart-tooltip__num">{r.value}</span>
        </div>
      ))}
    </div>
  );
}

/* ───────────── delta 趋势标记 ───────────── */

export function PortableChartDelta({
  direction,
  children,
}: {
  direction: "up" | "down" | "flat";
  children: React.ReactNode;
}) {
  const sign = direction === "up" ? "↑" : direction === "down" ? "↓" : "→";
  return (
    <span className={`cp-chart-delta cp-chart-delta--${direction}`}>
      <span aria-hidden="true">{sign}</span>
      {children}
    </span>
  );
}

/* ───────────── ChartCard 外壳 ───────────── */

export type PortableChartState = "default" | "loading" | "empty" | "error" | "no-permission";

export interface PortableChartCardProps {
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  /** header 右侧操作（筛选 / 时间范围切换等） */
  actions?: React.ReactNode;
  /** 图例（传则渲染在图表下方） */
  legend?: PortableChartLegendItem[];
  /** 解释文案（chart-stat 强制：图表必须配说明） */
  explanation?: React.ReactNode;
  /** 状态；非 default 时覆盖图表区，保留卡片尺寸 */
  state?: PortableChartState;
  /** 自定义各态文案 / 重试入口 */
  stateContent?: Partial<
    Record<Exclude<PortableChartState, "default">, { title?: React.ReactNode; desc?: React.ReactNode; action?: React.ReactNode }>
  >;
  /** 图表区最小高度，默认走 css 240px */
  chartMinHeight?: number;
  children?: React.ReactNode;
  className?: string;
}

const DEFAULT_STATE_TEXT: Record<
  Exclude<PortableChartState, "default">,
  { title: string; desc: string }
> = {
  loading: { title: "加载中…", desc: "正在拉取图表数据，请稍候。" },
  empty: { title: "暂无数据", desc: "当前时间范围内没有可展示的数据，换个时间范围试试。" },
  error: { title: "加载失败", desc: "数据请求出错，请检查网络后重试。" },
  "no-permission": { title: "无访问权限", desc: "你当前没有查看该数据的权限，请联系管理员开通。" },
};

function ChartSpinner() {
  return (
    <span
      aria-hidden="true"
      style={{
        width: 20,
        height: 20,
        border: "2px solid var(--cp-border-control, #C8CFDA)",
        borderTopColor: "var(--cp-brand-blue)",
        borderRadius: "999px",
        display: "inline-block",
        animation: "cp-spin 0.7s linear infinite",
      }}
    />
  );
}

export function PortableChartCard({
  title,
  subtitle,
  actions,
  legend,
  explanation,
  state = "default",
  stateContent,
  chartMinHeight,
  children,
  className = "",
}: PortableChartCardProps) {
  const cls = ["cp-chart-card", className].filter(Boolean).join(" ");
  const overlay = state !== "default";
  const text = overlay ? DEFAULT_STATE_TEXT[state] : null;
  const custom = overlay ? stateContent?.[state] : undefined;

  return (
    <section className={cls}>
      <header className="cp-chart-card__header">
        <div className="cp-chart-card__heading">
          <span className="cp-chart-card__title">{title}</span>
          {subtitle != null && <span className="cp-chart-card__subtitle">{subtitle}</span>}
        </div>
        {actions != null && <div className="cp-chart-card__actions">{actions}</div>}
      </header>

      <div
        className="cp-chart-card__chart"
        style={chartMinHeight ? { minHeight: chartMinHeight } : undefined}
      >
        {/* 图表实例始终保留在 DOM 中以维持高度；非 default 态用覆盖层遮挡 */}
        {children}
        {overlay && (
          <div className="cp-chart-state" role={state === "error" ? "alert" : "status"}>
            {state === "loading" && <ChartSpinner />}
            <span className="cp-chart-state__title">{custom?.title ?? text!.title}</span>
            <span className="cp-chart-state__desc">{custom?.desc ?? text!.desc}</span>
            {custom?.action != null && <div className="cp-chart-state__action">{custom.action}</div>}
          </div>
        )}
      </div>

      {legend && legend.length > 0 && <PortableChartLegend items={legend} />}
      {explanation != null && <p className="cp-chart-card__explanation">{explanation}</p>}
    </section>
  );
}
