/**
 * DateTimePicker Preview · Spec Verification
 * --------------------------------------------------------------
 * 路由：/preview/date-time-picker
 * 用途：对照 `clawpro-portable-design-skill/component-specs/date-picker.md §13`
 *      （DateTimePicker 变体）逐项验证日期 + 时分（秒）多列、草稿态提交、
 *      Admin / Tenant 圆角分流，以及 demo 仓
 *      `client/src/components/ui/date-time-picker.tsx` 的现状审计。
 *
 * 重点：可视化暴露「秒」列接入后的 spec 与 demo 实现的对齐情况（含硬编码 → token 偏差）。
 */
import * as React from "react";
import { DateTimePicker } from "@/components/ui/date-time-picker";
import { MetaText, SectionTitle } from "@/components/ui/Typography";
import { Check, AlertTriangle } from "lucide-react";

/* spec §3 / portable/css/tokens.css，预览页内联兜底（实际已在 client/src/index.css 暴露） */
const SPEC_TOKENS: React.CSSProperties = {
  ["--cp-brand-blue" as never]: "#1447E6",
  ["--cp-border" as never]: "#EAEEF4",
  ["--cp-border-control" as never]: "#C8CFDA",
  ["--cp-surface" as never]: "#FFFFFF",
  ["--cp-text-title" as never]: "#0F172A",
  ["--cp-text-weak" as never]: "#94A3B8",
  ["--cp-text-muted" as never]: "#64748B",
  ["--cp-text-brand" as never]: "#1447E6",
  ["--bg-grey-normal" as never]: "#FAFBFD",
  ["--bg-grey-hover" as never]: "#F5F6FA",
  ["--cp-shadow-overlay" as never]:
    "0px 4px 16px -2px rgba(0,0,0,0.08), 0px 2px 6px rgba(0,0,0,0.06)",
};

/* ------------------------------- Layout ------------------------------------ */
function DemoBlock({
  title,
  desc,
  children,
}: {
  title: string;
  desc?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-[8px] border border-[#e5e5e5] overflow-hidden bg-white">
      <header className="flex items-baseline justify-between px-5 py-3 border-b border-[#f0f0f0] bg-[#fafafa]">
        <SectionTitle as="h3" className="!text-sm">
          {title}
        </SectionTitle>
        {desc ? <MetaText tone="weak">{desc}</MetaText> : null}
      </header>
      <div className="p-5">{children}</div>
    </section>
  );
}

function Swatch({ color }: { color: string }) {
  return (
    <span
      className="inline-block w-3.5 h-3.5 rounded-sm border border-[#e5e5e5] align-middle mr-1.5"
      style={{ background: color }}
    />
  );
}

/* -------------------------------------------------------------------------- */
/*  spec §13.3 视觉标准对照（增量：在 DatePicker 基础上仅时间列相关项）              */
/* -------------------------------------------------------------------------- */
type SpecRow = {
  item: string;
  expect: string;
  actual: string;
  status: "ok" | "diff";
  note?: string;
};

const SPEC_ROWS: SpecRow[] = [
  {
    item: "Trigger 高度",
    expect: "h-9 (36px) · 同 DatePicker",
    actual: "h-9 (36px)",
    status: "ok",
    note: "与 Input / Select / DatePicker 对齐，唯一差异是显示值带时间",
  },
  {
    item: "Trigger 圆角 (Admin)",
    expect: "rounded-[4px]",
    actual: "rounded-[4px]",
    status: "ok",
  },
  {
    item: "Trigger 圆角 (Tenant)",
    expect: "tenant → rounded-full（胶囊）",
    actual: "tenant prop → rounded-full",
    status: "ok",
    note: "与 DatePicker 同口径（spec §5 / SKILL-TENANT.md）",
  },
  {
    item: "Time Column 选中态",
    expect: "品牌蓝实底白字 var(--cp-brand-blue) #1447E6",
    actual: "bg-[#1447E6] text-white font-medium · 硬编码",
    status: "ok",
    note: "与日历选中日同色，色对；建议替换为 var(--cp-brand-blue)",
  },
  {
    item: "Time Column hover",
    expect: "弱蓝提示，不抢选中态",
    actual: "bg-[#eff4ff] · 硬编码 brand-tint",
    status: "diff",
    note: "与日历 day hover 同口径，未走 token；可考虑 --bg-grey-normal",
  },
  {
    item: "Time Column 数字",
    expect: "等宽对齐、两位补零（00–59）",
    actual: "tabular-nums + padStart(2,'0')",
    status: "ok",
  },
  {
    item: "时 / 分 / 秒 列数",
    expect: "默认 2 列；showSeconds → 3 列",
    actual: "showSeconds 条件渲染第三列，复用同一 TimeColumn",
    status: "ok",
    note: "三列共享选中态 / 滚动定位 / hover 样式",
  },
  {
    item: "列分隔线",
    expect: "列间细分隔，与日历区分隔同体系",
    actual: "border-l border-gray-100（列间）/ border-gray-200（日历↔时间）· 硬编码",
    status: "diff",
    note: "色阶接近但未走 --cp-border token",
  },
  {
    item: "选中项滚动定位",
    expect: "打开时选中项滚到列内可视区，不影响整页",
    actual: "scrollIntoView({ block: 'nearest' })",
    status: "ok",
  },
  {
    item: "Footer 预览文本",
    expect: "弱灰文字，显示草稿态完整值",
    actual: "text-gray-500（有值）/ text-gray-400（空）· 硬编码",
    status: "ok",
    note: "格式随 showSeconds 自动带 :ss",
  },
  {
    item: "Footer 确定按钮",
    expect: "黑底白字（Button 默认变体）",
    actual: "Button size=sm（默认变体）",
    status: "ok",
    note: "未选日期时 disabled",
  },
  {
    item: "CalendarIcon 占位",
    expect: "保留日历 icon（spec §9 Don't）",
    actual: "16×16 inline svg + shrink-0",
    status: "ok",
  },
];

/* -------------------------------------------------------------------------- */
/*  spec §13.1 / 状态完整性                                                     */
/* -------------------------------------------------------------------------- */
type StateRow = {
  state: string;
  desc: string;
  covered: boolean;
};

const STATE_ROWS: StateRow[] = [
  { state: "default", desc: "未选择，显示 placeholder（弱灰）", covered: true },
  {
    state: "selected (HH:mm)",
    desc: "显示 YYYY-MM-DD HH:mm（标题色）",
    covered: true,
  },
  {
    state: "selected (HH:mm:ss)",
    desc: "showSeconds 时显示 YYYY-MM-DD HH:mm:ss",
    covered: true,
  },
  { state: "open", desc: "Trigger 描边强调，日历 + 时间列展开", covered: true },
  {
    state: "draft",
    desc: "选日期 / 时分秒只更新预览，点「确定」才提交",
    covered: true,
  },
  { state: "disabled", desc: "灰底灰字、cursor-not-allowed", covered: true },
  { state: "tenant", desc: "tenant=true → rounded-full；普通表单仍 4px", covered: true },
];

/* -------------------------------------------------------------------------- */
/*  spec §13.6 QA Checklist                                                    */
/* -------------------------------------------------------------------------- */
const QA_ITEMS: { label: string; passed: boolean; note?: string }[] = [
  {
    label: "触发器与 DatePicker / Input / Select 同高 h-9",
    passed: true,
    note: "四者对齐",
  },
  {
    label: "时 / 分 /（秒）列选中态使用品牌蓝，与日历选中日同色",
    passed: true,
    note: "色值 #1447E6 正确，但**未走 --cp-brand-blue token**（建议跟进）",
  },
  {
    label: "showSeconds 关闭时值格式不含秒，开启后为 HH:mm:ss",
    passed: true,
    note: "parseDateTimeString / formatDateTime 按 showSeconds 分流",
  },
  {
    label: "草稿态：选完点「确定」才 onChange",
    passed: true,
    note: "draftDate/Hour/Minute/Second 暂存，handleConfirm 提交",
  },
  {
    label: "展示台已接入：/design-system/components → DateTimePicker",
    passed: true,
    note: "FormPreview 默认版 + showSeconds 版可交互示例",
  },
];

/* -------------------------------------------------------------------------- */
/*  Demo Helpers                                                                */
/* -------------------------------------------------------------------------- */
const NOW = (() => {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
})();

/** 计算 N 天后的 ISO date */
function offsetDate(base: string, days: number) {
  const [y, m, d] = base.split("-").map(Number);
  const dt = new Date(y, m - 1, d);
  dt.setDate(dt.getDate() + days);
  const yy = dt.getFullYear();
  const mm = String(dt.getMonth() + 1).padStart(2, "0");
  const dd = String(dt.getDate()).padStart(2, "0");
  return `${yy}-${mm}-${dd}`;
}

/* -------------------------------------------------------------------------- */
/*  Page                                                                        */
/* -------------------------------------------------------------------------- */
export default function DateTimePickerPreview() {
  // Demo 1: 默认时分版（受控）
  const [dtDefault, setDtDefault] = React.useState<string>("");

  // Demo 2: showSeconds 时分秒版
  const [dtSeconds, setDtSeconds] = React.useState<string>(`${NOW} 09:30:00`);

  // Demo 3: secondStep / minuteStep 步长
  const [dtStep, setDtStep] = React.useState<string>(`${NOW} 09:30:00`);

  // Demo 4: admin vs tenant
  const [adminDt, setAdminDt] = React.useState<string>("");
  const [tenantDt, setTenantDt] = React.useState<string>("");

  // Demo 5: min / max
  const minBound = offsetDate(NOW, -3);
  const maxBound = offsetDate(NOW, 14);
  const [boundedDt, setBoundedDt] = React.useState<string>("");

  // Demo 6: disabled
  const [disabledDt] = React.useState<string>(`${NOW} 12:00`);

  // 偏差统计
  const diffCount = SPEC_ROWS.filter((r) => r.status === "diff").length;
  const okCount = SPEC_ROWS.length - diffCount;

  return (
    <div className="min-h-screen bg-[#fafafa] py-10 px-6" style={SPEC_TOKENS}>
      <div className="max-w-[960px] mx-auto space-y-5">
        {/* 顶部说明 */}
        <header className="rounded-[8px] border border-[#e5e5e5] bg-white p-6">
          <div className="text-xs text-[#94A3B8] tracking-wide uppercase mb-2">
            clawpro-portable-design-skill · component-specs · date-picker §13
          </div>
          <SectionTitle as="h1">DateTimePicker · Spec Verification</SectionTitle>
          <p className="mt-2 text-sm text-[#64748B] leading-relaxed">
            按 <code>date-picker.md §13（DateTimePicker 变体）</code> 验证{" "}
            <code>client/src/components/ui/date-time-picker.tsx</code> 的对齐情况。共{" "}
            <strong className="text-[#0F172A]">{SPEC_ROWS.length}</strong> 项视觉标准，
            <span className="text-[#16A34A] font-medium">{okCount} 项通过</span>，
            <span className="text-[#D97706] font-medium">{diffCount} 项硬编码偏差</span>
            （色对但未走 token，建议后续统一）。
          </p>

          <ul className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-[#64748B]">
            <li>
              <Swatch color="#1447E6" />
              时间列选中 / 品牌蓝 --cp-brand-blue
            </li>
            <li>
              <Swatch color="#eff4ff" />
              时间列 hover 弱蓝
            </li>
            <li>
              <Swatch color="#EAEEF4" />
              列分隔 --cp-border
            </li>
            <li>
              <Swatch color="#94A3B8" />
              Placeholder --cp-text-weak
            </li>
            <li>
              <Swatch color="#0F172A" />
              已选值 --cp-text-title
            </li>
            <li>
              <Swatch color="#FAFAFA" />
              Disabled 底
            </li>
          </ul>
        </header>

        {/* Demo 1: 默认时分 + showSeconds 时分秒 */}
        <DemoBlock
          title="① 时分 vs 时分秒（spec §13.4 showSeconds）"
          desc="左默认 2 列（HH:mm）；右 showSeconds 3 列（HH:mm:ss）"
        >
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-8">
            <div className="space-y-2">
              <MetaText tone="weak">默认 · 时 / 分两列 · 值格式 HH:mm</MetaText>
              <DateTimePicker
                value={dtDefault}
                onChange={setDtDefault}
                placeholder="选择日期时间"
                className="w-[260px]"
              />
              <p className="text-xs text-[#64748B]">
                当前值：
                <code className="text-[#0F172A]">{dtDefault || "（未选）"}</code>
                ，右侧仅时 / 分两列。
              </p>
            </div>

            <div className="space-y-2">
              <MetaText tone="weak">showSeconds · 时 / 分 / 秒三列 · 值格式 HH:mm:ss</MetaText>
              <DateTimePicker
                showSeconds
                value={dtSeconds}
                onChange={setDtSeconds}
                placeholder="选择日期时间"
                className="w-[260px]"
              />
              <p className="text-xs text-[#64748B]">
                当前值：<code className="text-[#0F172A]">{dtSeconds}</code>
                ，新增的「秒」列与时 / 分列样式完全一致。
              </p>
            </div>
          </div>
        </DemoBlock>

        {/* Demo 2: 步长 */}
        <DemoBlock
          title="② 步长 minuteStep / secondStep（spec §13.4）"
          desc="分钟按 5 步长、秒按 10 步长生成列表"
        >
          <div className="flex flex-wrap items-center gap-3">
            <DateTimePicker
              showSeconds
              minuteStep={5}
              secondStep={10}
              value={dtStep}
              onChange={setDtStep}
              className="w-[260px]"
            />
            <span className="text-sm text-[#0F172A]">
              已选：<code>{dtStep}</code>
            </span>
          </div>
          <p className="mt-3 text-xs text-[#64748B]">
            分列只出现 <code>00,05,10…55</code>，秒列只出现{" "}
            <code>00,10,20…50</code>；步长仅控制候选项，不限制手动传入的精确值。
          </p>
        </DemoBlock>

        {/* Demo 3: admin vs tenant */}
        <DemoBlock
          title="③ Admin vs Tenant · 圆角分流（spec §13.3 / SKILL-TENANT.md）"
          desc="同一组件，仅圆角差异；tenant 走 rounded-full，admin 走 rounded-[4px]"
        >
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-8">
            <div className="space-y-2">
              <MetaText tone="weak">Admin · rounded-[4px]</MetaText>
              <DateTimePicker
                value={adminDt}
                onChange={setAdminDt}
                placeholder="选择执行时间"
                className="w-[260px]"
              />
              <p className="text-xs text-[#64748B]">
                场景：定时任务执行时间、模型配额生效时间（管理端）。
              </p>
            </div>

            <div className="space-y-2">
              <MetaText tone="weak">Tenant · rounded-full（胶囊）</MetaText>
              <DateTimePicker
                tenant
                value={tenantDt}
                onChange={setTenantDt}
                placeholder="选择日期时间"
                className="w-[260px]"
              />
              <p className="text-xs text-[#64748B]">
                场景：tenant 端筛选 / 搜索；
                <strong>普通表单仍用 4px，不一律 full</strong>。
              </p>
            </div>
          </div>
        </DemoBlock>

        {/* Demo 4: min / max */}
        <DemoBlock
          title="④ min / max 限制 · 越界禁用置灰"
          desc={`日期只能选 [${minBound}, ${maxBound}] 范围内`}
        >
          <div className="flex flex-wrap items-center gap-3">
            <DateTimePicker
              showSeconds
              value={boundedDt}
              onChange={setBoundedDt}
              placeholder="选择日期时间"
              min={minBound}
              max={maxBound}
              className="w-[260px]"
            />
            {boundedDt && (
              <span className="text-sm text-[#0F172A]">
                已选：<code>{boundedDt}</code>
              </span>
            )}
          </div>
          <p className="mt-3 text-xs text-[#64748B]">
            点开后越界日期 <code>opacity-50</code> + 不可点击；复用 react-day-picker 的{" "}
            <code>disabled</code> matcher（与 DatePicker 同源）。
          </p>
        </DemoBlock>

        {/* Demo 5: disabled */}
        <DemoBlock
          title="⑤ disabled · 灰底灰字"
          desc="不可点击，hover 不变描边"
        >
          <DateTimePicker
            value={disabledDt}
            onChange={() => {}}
            disabled
            className="w-[260px]"
          />
          <p className="mt-3 text-xs text-[#64748B]">
            <code>cursor-not-allowed</code> + <code>bg-[#FAFAFA]</code>，与 DatePicker
            disabled 同口径。
          </p>
        </DemoBlock>

        {/* §13.3 视觉标准对照表 */}
        <DemoBlock
          title={`§13.3 视觉标准对照（${SPEC_ROWS.length} 项 · ${okCount} 通过 / ${diffCount} 偏差）`}
          desc="左列 spec 期望，右列 demo 仓 date-time-picker.tsx 实现，硬编码 → token 偏差用色块标注"
        >
          <div className="overflow-hidden rounded-[6px] border border-[#e5e5e5]">
            <table className="w-full text-sm">
              <thead className="bg-[#fafafa] text-left text-xs text-[#64748B]">
                <tr>
                  <th className="px-3 py-2 w-[24%] font-medium">Item</th>
                  <th className="px-3 py-2 w-[28%] font-medium">Expect (spec)</th>
                  <th className="px-3 py-2 w-[30%] font-medium">Actual (demo)</th>
                  <th className="px-3 py-2 w-[18%] font-medium">Status</th>
                </tr>
              </thead>
              <tbody>
                {SPEC_ROWS.map((row) => (
                  <tr key={row.item} className="border-t border-[#f0f0f0] align-top">
                    <td className="px-3 py-2 text-[#0F172A]">{row.item}</td>
                    <td className="px-3 py-2 text-[#0F172A] font-mono text-xs">
                      {row.expect}
                    </td>
                    <td className="px-3 py-2 text-[#334155] font-mono text-xs">
                      {row.actual}
                    </td>
                    <td className="px-3 py-2">
                      {row.status === "ok" ? (
                        <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-[3px] bg-[#ECFDF5] text-[#16A34A] text-xs">
                          <Check className="w-3 h-3" /> OK
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-[3px] bg-[#FEF3C7] text-[#D97706] text-xs">
                          <AlertTriangle className="w-3 h-3" /> Diff
                        </span>
                      )}
                      {row.note && (
                        <p className="mt-1 text-xs text-[#64748B] leading-snug">
                          {row.note}
                        </p>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </DemoBlock>

        {/* 状态完整性 */}
        <DemoBlock
          title={`§13.1 States 完整性（${STATE_ROWS.length} / ${STATE_ROWS.length} 全覆盖）`}
          desc="default / selected(HH:mm) / selected(HH:mm:ss) / open / draft / disabled / tenant"
        >
          <ul className="space-y-2 text-sm">
            {STATE_ROWS.map((row) => (
              <li key={row.state} className="flex items-start gap-2">
                <span className="inline-flex items-center justify-center w-5 h-5 rounded-full bg-[#ECFDF5] text-[#16A34A] mt-0.5 shrink-0">
                  <Check className="w-3 h-3" />
                </span>
                <span>
                  <code className="text-[#1447E6]">{row.state}</code>
                  <span className="text-[#64748B] ml-2">— {row.desc}</span>
                </span>
              </li>
            ))}
          </ul>
        </DemoBlock>

        {/* §13.6 QA */}
        <DemoBlock
          title="§13.6 QA Checklist"
          desc="5 项硬约束 · 全部 PASS（其中时间列色值正确但未走 token）"
        >
          <ul className="space-y-2 text-sm">
            {QA_ITEMS.map((q) => (
              <li key={q.label} className="flex items-start gap-2">
                <span
                  className={
                    q.passed
                      ? "inline-flex items-center justify-center w-5 h-5 rounded-full bg-[#ECFDF5] text-[#16A34A] mt-0.5 shrink-0"
                      : "inline-flex items-center justify-center w-5 h-5 rounded-full bg-[#FEF2F2] text-[#DC2626] mt-0.5 shrink-0"
                  }
                >
                  <Check className="w-3 h-3" />
                </span>
                <span>
                  <span className="text-[#0F172A]">{q.label}</span>
                  {q.note && (
                    <span className="block text-xs text-[#64748B] mt-0.5">
                      {q.note}
                    </span>
                  )}
                </span>
              </li>
            ))}
          </ul>
        </DemoBlock>

        {/* 现状审计 */}
        <DemoBlock
          title="§13 Demo Repo 现状审计"
          desc="结论：在 DatePicker 基础上扩展时间列，复用日历与品牌蓝选中态；剩余优化点同 DatePicker 的硬编码 → token 替换"
        >
          <ol className="space-y-1.5 text-sm text-[#334155] list-decimal list-inside">
            <li>
              ✅ 复用 <code>Calendar</code>（react-day-picker）日历全部能力：键盘、月份切换、
              min/max 禁用、本地时区解析，与 DatePicker 同源
            </li>
            <li>
              ✅ 时 / 分 /（秒）三列共用同一 <code>TimeColumn</code>，选中态 / 滚动定位 /
              hover 完全一致，<code>showSeconds</code> 仅条件渲染第三列
            </li>
            <li>
              ✅ 草稿态：<code>draftDate / draftHour / draftMinute / draftSecond</code>{" "}
              暂存，点「确定」才 <code>onChange</code>，避免半成品值外泄
            </li>
            <li>
              ✅ 值格式按 <code>showSeconds</code> 分流：
              <code>YYYY-MM-DD HH:mm</code> / <code>YYYY-MM-DD HH:mm:ss</code>，
              解析侧缺省秒为 0 向后兼容
            </li>
            <li>
              ⚠️ Time Column hover <code>bg-[#eff4ff]</code> 是硬编码 brand-tint，未走
              token（与 DatePicker day hover 同口径，建议统一为{" "}
              <code>--bg-grey-normal</code> 或 brand-tint token）
            </li>
            <li>
              ⚠️ 选中态 <code>bg-[#1447E6]</code> 色对但建议替换为{" "}
              <code>bg-[var(--cp-brand-blue)]</code>，宿主仓覆盖品牌蓝时同步漂移
            </li>
            <li>
              ⚠️ Trigger 描边 / hover / open 复用 DatePicker 的{" "}
              <code>border-gray-200</code> / <code>border-blue-500</code>，同样存在
              硬编码 → token 偏差（统一修复时一并处理）
            </li>
          </ol>
        </DemoBlock>

        {/* footer */}
        <footer className="text-xs text-[#94A3B8] text-center py-6">
          spec source ·{" "}
          <code>
            .codebuddy/skills/clawpro-portable-design-skill/component-specs/date-picker.md
            §13
          </code>
          ／demo · <code>client/src/components/ui/date-time-picker.tsx</code>
        </footer>
      </div>
    </div>
  );
}
