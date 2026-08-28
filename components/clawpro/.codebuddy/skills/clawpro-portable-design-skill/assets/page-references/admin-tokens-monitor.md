# Tokens 监控页 · admin-tokens-monitor

> **类别**：数据看板 / 监控页（NumberCard 大盘 + 折线图 + Segment + 明细表）
> **路由**：`/admin/tokens-monitor`
> **源码**：`client/src/pages/admin/TokensMonitor.tsx`
> **截图**：`admin-token-monitor.png`（1440×900）

---

## 1. 视觉骨架

```
ConsoleShell（左导航）
└─ ContentArea
   ├─ AlertBanner（待配置：3 项基础配置未完成）   ← 全局提醒条
   ├─ AdminPageHeader
   │  ├─ Title「Tokens 监控」
   │  ├─ Subtitle「查看企业用户和模型的 Tokens 消耗情况。查看tokens统计规则」
   │  └─ Right: DateRange [from] — [to]  +  RefreshIcon
   │
   ├─ NumberCard Group ×5  （等宽 grid，gap=12）
   │  · 总请求数 1,841            (RequestsIcon · 蓝)
   │  · 输入 Tokens 533,112        (InputTokensIcon · 蓝紫)
   │  · 输出 Tokens 419,040        (OutputTokensIcon · 紫)
   │  · 总 Tokens 952,152          (TotalTokensIcon · 蓝)
   │  · 本月全局配额消耗 95.2%     (含警示条 · ProgressBar 红色)
   │
   ├─ SurfaceCard「最近 7 天 Tokens 趋势」
   │  └─ LineChart（双线：输入 Tokens · 输出 Tokens · 蓝/灰）
   │     - X: 06-04 ... 06-10
   │     - Y: 0 ~ 600k
   │     - Legend 居中底部
   │
   └─ SurfaceCard
      ├─ LineTabs：按实例 | 按用户 | 按模型 | 按分组 ● | 按会话
      │  （"按分组"右上角带红点 · 表示新功能或差异提示）
      ├─ 副标题「汇总所选时间范围内每台实例的 Token 消耗，按总 Tokens 降序排序」
      │  └─ 右：DownloadIcon（导出）
      └─ Table
         · cols: 名称/ID  用户ID  总请求数  输入 Tokens  输出 Tokens  总 Tokens
         · 双行表头单元（ins-loadfail01 这类副 ID 用 CodeText 灰）
         · 长名称用 truncate + Tooltip
```

---

## 2. 组件清单与 spec 对照

| 区域 | 组件 | import | spec |
|---|---|---|---|
| 全局提醒 | `Alert`（warning 调）+ AlertOperationInfoIcon | `@/components/ui/alert` | `component-specs/alert.md` |
| 页头 | `AdminPageHeader` | `@/components/ui/admin-page-header` | `component-specs/admin-page-header.md` |
| 日期 | `DatePicker` ×2（区间） | `@/components/ui/date-picker` | `component-specs/date-picker.md` |
| 大盘卡 | `NumberCard` + 内置 `RequestsIcon` / `InputTokensIcon` / `OutputTokensIcon` / `TotalTokensIcon` | `@/components/ui/number-card` | `component-specs/number-card.md` |
| 配额卡 | `NumberCard` + 自定义 `ProgressBar`（红色警示） | 同上 | 同上 |
| 容器 | `SurfaceCard` / `SurfaceInner` | `@/components/ui/Surface` | `component-specs/surface.md` |
| 折线图 | `recharts` `LineChart` | `recharts` | — (chart 用法，参考此页源码 100~) |
| Tabs | `Tabs` / `TabsList` / `TabsTrigger`（**LineTabs 风格**：底部下划线） | `@/components/ui/tabs` | `component-specs/tabs.md` |
| 表格 | `Table` 全套 | `@/components/ui/table` | `component-specs/table.md` |
| 数字 | `StatNumber` | `@/components/ui/Typography` | `component-specs/typography.md` |
| 分页 | `Pagination` | `@/components/ui/pagination` | `component-specs/pagination.md` |

---

## 3. 关键 token / 规范要点

- **NumberCard 数列**：5 列等宽 `grid-cols-5 gap-3`，最后一张「配额消耗」结构变体（百分比 + 警示进度条），**不要拆成两张卡**——保持 5 列对齐。
- **LineTabs vs Segment**：此页用 LineTabs（下划线指示，靠左），适合"切维度查看同一类数据"；Segment 适合"切互斥的视图模式"（参考 `admin-members.md`）。
- **新功能红点**：TabsTrigger 右上角红点用 `relative` + `::after` 或 absolute span，不要用 `Badge`（会撑高 trigger）。
- **Chart Y 轴**：刻度 `0/150k/300k/450k/600k` —— 写成 `[0, 150_000, 300_000, 450_000, 600_000]` ticks，避免 recharts 自动取值不齐。
- **Chart 配色**：`stroke=tokens.color.brand.primary`（输入）、`stroke=tokens.color.text.secondary`（输出），保持冷色调。
- **导出按钮**：`Button variant="ghost" size="icon"` + `Download` icon，**右上角对齐 Table 副标题**而非卡 header。
- **AlertBanner**：顶部全局提醒在 NumberCard **之上**，不在 SurfaceCard 里。

---

## 4. why-typical

- 是 ClawPro 管控端**最复杂的数据看板模板**：大盘卡 + 时序图 + 多维度切换明细表 三段式齐全。
- 涵盖 LineTabs / NumberCard / recharts / Pagination / 全局 AlertBanner / DateRange 6 类常见控件，新页面套此结构改字段即可。
- 配额消耗那张"变体卡"是**真实业务中很常见的需求**——卡片网格中要混搭一张特殊卡时的对齐方案。

---

## 5. ❌反例 / ✅正例

| 场景 | ❌ 反例 | ✅ 正例 |
|---|---|---|
| 5 张 NumberCard 等宽 | `flex` 自适应导致最后一张被挤窄 | `grid grid-cols-5 gap-3`（明确等分） |
| 配额卡进度条 | 用 `<progress>` 原生标签 | 自定义 `<div class="h-1 bg-…/20"><div style={{width:'95.2%'}} class="bg-danger"/></div>` |
| LineTabs 红点 | 在 trigger 文字后加 `<Badge>` | `relative` + `<span class="absolute -top-0.5 -right-1.5 size-1.5 rounded-full bg-danger"/>` |
| Chart 双线配色 | 蓝 + 红（红会被误读为"异常"） | 蓝（强调主指标）+ 灰（次指标） |
| 表格 ID 列 | 把 ID 与 name 拼在一行 | 两行：name 主体 + ID 用 `CodeText` 灰色 12px |
| AlertBanner 位置 | 嵌在 SurfaceCard 顶部 | **页面顶部 ContentArea 第一行**，与 Header 分离 |
