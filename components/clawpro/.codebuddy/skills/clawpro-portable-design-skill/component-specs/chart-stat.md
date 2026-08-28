# Chart / Stat

## 1. Purpose

- 统一 Dashboard、监控页、额度页中的统计数字、趋势图、图例、Tooltip 和空 / 加载 / 错误态。
- 避免图表色彩、统计卡片阴影、数字字体和解释文案在不同页面散开。

## 2. Scope

- 适用端：Admin / Tenant / Shared。
- 必用场景：Dashboard、监控、额度、运营概览、资源统计、趋势图、饼图 / 柱状图 / 折线图。
- 不适用场景：营销插画、Landing 装饰图形、复杂 BI 报表。

## 3. Visual Standard

| Item | Default | Notes |
|---|---|---|
| KPI Card | 顶部概览使用 `NumberCard` | 四张指标卡不要再手写散装 `StatCard` |
| Stat Surface | `SurfaceCard` / `TenantCard` 分端 | Admin 4px；Tenant 业务卡 12px |
| Primary Metric | `StatNumber` / DIN / tabular nums | 数字对齐稳定 |
| Chart Primary | `--cp-brand-blue` | 主趋势 / 主序列 |
| Auxiliary Lines | 弱灰 / 低透明 | 不抢主数据 |
| Grid | 细弱线 | 不使用高对比网格 |
| Tooltip | 白底、4px、overlay shadow | 文本 12px-14px |
| Legend | 小色块 + 文本 | 色彩必须可区分 |
| Empty | 说明无数据原因 | 不只写“暂无数据” |
| Loading | 图表骨架或局部 spinner | 保留卡片尺寸 |

## 4. Anatomy

```text
NumberCard
  Icon
  Label
  Value
  Extra optional
  Footer optional

ChartCard
  Header
  Chart
  Legend / Tooltip
  Explanation
```

## 5. States

- default: 有数据，图表和说明完整。
- loading: 保留图表区域高度，显示 skeleton / spinner。
- empty: 无数据，解释原因和下一步。
- error: 显示失败原因和重试。
- no-permission: 说明需要权限。
- positive / negative / neutral delta: 趋势变化使用语义色，不只靠颜色表达。
- dense: 小图表只保留必要轴和 Tooltip。

## 6. Demo Repo Usage

- 图表底层：`client/src/components/ui/chart.tsx`
- 顶部 KPI 卡：`client/src/components/ui/number-card.tsx`
- 数字字重与字体：`client/src/components/ui/Typography.tsx` 里的 `StatNumber`
- 典型数据页：`client/src/pages/admin/OpenClawMonitor.tsx`、`client/src/pages/admin/TokensMonitor.tsx`
- 组件预览页：`client/src/pages/DesignSystemComponents.tsx` 的 NumberCard / Chart Stat 示例
- 页面 recipe：`references/page-recipes.md` 的 Dashboard 部分

## 7. Portable Fallback

### 7.1 If host repo already has chart library

- 保留宿主仓图表库和数据逻辑。
- 只要求对齐色彩、卡片层级、Tooltip、图例、数字字体和空 / 加载 / 错误态。
- 如果图表库不支持 token，建立颜色映射层，不在页面散写旧色。

### 7.2 Minimal React fallback

```tsx
export function PortableNumberCard() {
  return (
    <section className="rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] p-5">
      <div className="mb-3 flex items-center gap-2">
        <span aria-hidden="true" className="inline-flex h-[18px] w-[18px] shrink-0 items-center justify-center">
          <svg viewBox="0 0 18 18" width="18" height="18" fill="none">
            <defs>
              <radialGradient id="portable-number-card-grad" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(2.81 9) scale(13.5 720)">
                <stop stopColor="#202020" />
                <stop offset="1" stopColor="#0080FF" />
              </radialGradient>
            </defs>
            <path d="M10.7 1.8 6.8 9h2.6l-1.1 7.2 4.9-8H10.6l.1-6.4Z" fill="url(#portable-number-card-grad)" />
          </svg>
        </span>
        <span className="text-sm font-medium text-[var(--cp-text-title)]">总请求数</span>
      </div>
      <div className="font-[var(--cp-font-din)] text-2xl font-semibold leading-none tabular-nums text-[var(--cp-text-emphasis)]">1,841</div>
    </section>
  );
}
```

### 7.3 Minimal HTML/CSS fallback

```html
<section class="cp-number-card">
  <div class="cp-number-card__header">
    <span class="cp-number-card__icon" aria-hidden="true"></span>
    <span class="cp-number-card__label">总请求数</span>
  </div>
  <strong class="cp-number-card__value">1,841</strong>
</section>
```

```css
.cp-number-card { border: 1px solid var(--cp-border); border-radius: 4px; background: var(--cp-surface); padding: 20px; }
.cp-number-card__header { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; }
.cp-number-card__icon { width: 18px; height: 18px; border-radius: 999px; background: linear-gradient(135deg, #202020 0%, #0080FF 100%); }
.cp-number-card__label { font-size: 14px; line-height: 1.5; font-weight: 500; color: var(--cp-text-title); }
.cp-number-card__value { display: block; font-family: var(--cp-font-din); font-size: 24px; line-height: 1; font-weight: 600; color: var(--cp-text-emphasis); font-variant-numeric: tabular-nums; }
```

## 8. Migration Rules

- 旧写法：每个图表页面自行配色、顶部 KPI 手写卡片、统计数字普通字体、无空态说明。
- 新口径：顶部 KPI 先统一成 `NumberCard`，再统一 ChartCard 并映射宿主仓图表库。
- 主色使用品牌蓝，辅助线弱化；不要继续引入旧蓝紫渐变。
- 图表必须配解释文案，不只给线图。
- 无数据 / 无权限 / 加载失败分别处理。

## 9. Do / Don't

Do:

- 顶部 KPI 优先复用 `NumberCard`。
- 数字使用 DIN + tabular nums。
- 主趋势使用品牌蓝。
- 图表旁提供解释或指标说明。

Don't:

- 不要为顶部概览继续拼装自定义 KPI 卡。
- 不要用多套高饱和色抢主数据。
- 不要在图表空态只留空白。
- 不要让统计卡片使用重阴影。

## 10. QA Checklist

- [ ] 顶部 KPI 使用 `NumberCard`
- [ ] 统计卡片按 Admin / Tenant 分端
- [ ] 数字使用 DIN 且对齐稳定
- [ ] 图表主色和辅助色符合 token
- [ ] Tooltip / Legend 可读
- [ ] 空 / 加载 / 错误 / 无权限态完整
- [ ] 图表有解释文案
- [ ] 宿主仓 fallback 可执行

## 11. References

- Demo code: `client/src/components/ui/chart.tsx`
- Demo code: `client/src/components/ui/number-card.tsx`
- Demo code: `client/src/components/ui/Typography.tsx`
- Demo page: `client/src/pages/admin/OpenClawMonitor.tsx`
- Demo page: `client/src/pages/admin/TokensMonitor.tsx`
- Related recipe: `references/page-recipes.md`

## 代码对照（✅/❌）

### ❌ 错误：KPI 写成自由文字块
```tsx
<div className="flex gap-6">
  <div>
    <div className="text-gray-500 text-sm">总实例</div>
    <div className="text-3xl font-bold">128</div>
  </div>
  <div>
    <div className="text-gray-500 text-sm">运行中</div>
    <div className="text-3xl font-bold">112</div>
  </div>
</div>
```
**为什么错**：与 NumberCard 视觉差异大；多份页面之间 KPI 排版不统一。

### ✅ 正确：NumberCard 行
```tsx
<div className="grid grid-cols-4 gap-4">
  <NumberCard title="总实例"   value={128} />
  <NumberCard title="运行中"   value={112} tone="success" />
  <NumberCard title="告警中"   value={3}   tone="danger"  />
  <NumberCard title="本月新增" value={9}   />
</div>
```

---

### ❌ 错误：图表用 5 种花哨色
```tsx
<LineChart data={data}>
  <Line dataKey="cpu"  stroke="#ff6b6b" />
  <Line dataKey="mem"  stroke="#4ecdc4" />
  <Line dataKey="disk" stroke="#ffe66d" />
  <Line dataKey="net"  stroke="#a8e6cf" />
</LineChart>
```
**为什么错**：彩虹配色破坏 ClawPro 克制风格；多线难以分辨主次。

### ✅ 正确：品牌蓝主色 + chart-palette
```tsx
<LineChart data={data}>
  <Line dataKey="cpu"  stroke="var(--cp-chart-1)" /> {/* 品牌蓝主色 */}
  <Line dataKey="mem"  stroke="var(--cp-chart-2)" />
  <Line dataKey="disk" stroke="var(--cp-chart-3)" />
  <Line dataKey="net"  stroke="var(--cp-chart-4)" />
</LineChart>
{/* 调色板严格按 token 顺序使用，主指标固定 chart-1 */}
```

---

### ❌ 错误：轴文字硬编码 gray-400
```tsx
<XAxis tick={{ fontSize: 12, fill: '#9ca3af' }} />
<YAxis tick={{ fontSize: 12, fill: '#9ca3af' }} />
```
**为什么错**：颜色未走 token，深浅模式失效；数字未启用 tabular-nums。

### ✅ 正确：ChartAxis 预设
```tsx
<XAxis {...chartAxisProps} />
<YAxis {...chartAxisProps} tickFormatter={fmtNumber} />
{/* chartAxisProps 内置 12px / --cp-text-weak / tabular-nums */}
```

---

### ❌ 错误：自定义 Tooltip 弹层样式
```tsx
<Tooltip
  contentStyle={{
    background: 'rgba(0,0,0,0.8)',
    color: '#fff',
    borderRadius: 8,
  }}
/>
```
**为什么错**：与系统 Tooltip 风格不一致；黑底过重。

### ✅ 正确：ChartTooltip
```tsx
<Tooltip content={<ChartTooltip />} />
{/* 内置白底 + 阴影 / 12px / token 化颜色 */}
```

---

### ❌ 错误：图表容器写死 height
```tsx
<div style={{ width: 600, height: 240 }}>
  <LineChart {...} />
</div>
```
**为什么错**：响应式失效；移动端被压扁；多卡片高度不一致。

### ✅ 正确：ChartContainer
```tsx
<ChartContainer height={240}>
  <LineChart data={data}>...</LineChart>
</ChartContainer>
{/* 自动 ResponsiveContainer + 标题/图例/空态 */}
```
