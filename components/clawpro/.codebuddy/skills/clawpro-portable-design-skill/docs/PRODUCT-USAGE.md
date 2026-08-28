# PRODUCT-USAGE — 产品前端快速上手

> 这份文档给产品同事用。如果你要做一个新页面或适配既有页面，看这里。
> 
> 设计规范、组件标准、换皮映射都在这个包里，**你不需要切图、不需要问设计、不需要手抄 Figma**。

---

## 1. 做一个列表页（最常见的起点）

### 你需要读的文件

1. `references/foundation.md` —— Admin 基线（色值、字体、圆角、间距）
2. `references/admin.md` —— Admin 端规则（卡片、按钮、表单的全局约束）
3. `references/page-recipes.md` —— "列表页"的标准骨架（头部 + 工具条 + 表格 + 分页）
4. `component-specs/table.md` —— 表格规范
5. `component-specs/search-filter-bar.md` —— 搜索 / 筛选 / 刷新工具条
6. `component-specs/pagination.md` —— 分页器
7. 如果有"日期筛选"：`component-specs/date-picker.md`
8. 如果有"对象选择"（Agent / 部门 / 标签）：`component-specs/input-select.md`（已合并 Combobox）

### 实施过程

1. 复制 page recipe 的骨架
2. 按 `foundation.md` 对齐颜色、圆角、字号、间距
3. 表格行高、列宽、行操作按钮的尺寸 → 看 `table.md` 的规范表
4. 如果你的宿主仓没有"表格"组件 → 按 `table.md` 的 Portable Fallback 还原
5. 工具条 / 分页 / 日期选择器 同理
6. 完成后对照 `qa/admin-checklist.md` 逐项验收

### 常见问题

**Q：我们宿主仓用的是不同的表格组件，能直接用吗？**  
A：可以，但要对齐 `table.md` 的视觉标准（圆角、高度、行间距、分割线等）。如果没法对齐，用 Portable Fallback 里的 HTML/CSS。

**Q：色值不一样怎么办？**  
A：先读 `references/foundation.md`，它有"Admin 主题色 #1447E6 / 危险红 #DC2626"这种映射。如果你们主题 hex 不同，向设计要对应转译表；不要自己猜。

**Q：字号是多少？**  
A：全部在 `foundation.md` 的"Typography"表。12px / 14px / 16px 三档，加上"semibold / medium / regular"三档。表格数据用 12px Regular，标题用 14px Semibold。

---

## 2. 做一个 Tenant（客户端）页面

### 和 Admin 的差别

Tenant 页面有自己的视觉规范，不能直接用 Admin 的。看两个文件：

1. `references/foundation.md` —— 看 Tenant 部分（不同的色板、圆角可能是 12px、字体可能不同）
2. `references/tenant.md` —— Tenant 端规则（布局、交互、限制）

### 特殊组件

Tenant 有些组件和 Admin 不一样：

| 场景 | Admin | Tenant |
|---|---|---|
| 顶部导航 | Tab（下划线） | 参考 `component-specs/tenant-topnav.md` |
| 模式切换 | Segment（方角） | Segment（胶囊） |
| 按钮 | 黑底白字主按钮 | 可能不同，查 `component-specs/button.md` Tenant 部分 |
| 卡片 | Admin 卡片（4px 圆角） | TenantCard（12px 圆角、可能有不同阴影） |

---

## 3. 页面整体适配核对清单

### 开始前问自己

- [ ] 这是 Admin / Tenant / Landing？
- [ ] 页面类型是"列表 / 详情 / 表单 / 看板 / 引导"中的哪一种？
- [ ] 需要用到的高风险组件有哪些（表格 / 日期选择 / 对象选择 / 多选 / 分页）？

### 逐组件对齐

对每个高风险组件：

1. 找到对应 `component-specs/*.md`
2. 看"Visual Standard"表，对齐尺寸 / 圆角 / 颜色 / 间距
3. 如果宿主仓没有这个组件 → 复制 Portable Fallback（HTML/CSS 或 React）
4. 测试时对照 `qa/admin-checklist.md` 或 `qa/tenant-checklist.md`

### 完成核对

- [ ] 按 foundation 对齐了色值（品牌蓝 #1447E6、文字 #0F172A、背景 #FFFFFF 等）
- [ ] 圆角全部是管控端 4px（或 Tenant 端 12px）
- [ ] 字号 / 行高符合 foundation Typography
- [ ] 按钮变体是对的（主按钮用黑底 / 次按钮用白底描边）
- [ ] 表格行高、分割线、hover 态是对的
- [ ] 空状态不是大插画，用文字 + 可选图标（见 empty-state.md）
- [ ] 表单错误提示不漂浮在远处，贴近字段（见 form-controls.md）

---

## 4. 遇到设计问题怎么办

### 表现不清楚的几个场景

**"多选后怎么展示选中数量？"**  
→ 看 `component-specs/batch-actions-bar.md`

**"表格可以冻结列吗？怎么冻结？"**  
→ 看 `component-specs/table.md` 的"Fixed Column"部分

**"弹窗 footer 的按钮应该怎么排？"**  
→ 看 `component-specs/dialog-drawer.md` 的"Footer"

**"日期选择器支持范围选择吗？"**  
→ 看 `component-specs/date-picker.md` 的"Range"部分

**"StatusTag 和 Badge 怎么选？"**  
→ 看 `component-specs/status-tag.md` vs `component-specs/badge.md`；Status 用于"运行状态"，Badge 用于"标签 / 版本"

### 觉得规范有问题或有例外

1. 先确认自己是不是读漏了（spec 可能在"Migration Rules"或"Edge Cases"里说了例外）
2. 如果确实是新需求或冲突，发到 `references/conflict-log.md` 里的 Issue 渠道（见文件头注释）
3. 不要自己决定改色值 / 改圆角 / 改尺寸，等设计 review

---

## 5. 宿主仓缺"关键组件"怎么办

### 常见情况

| 缺的组件 | 怎么办 |
|---|---|
| 按钮 | 用 `component-specs/button.md` 的 Portable Fallback（`portable/react/button.tsx` 或 HTML/CSS） |
| 表格 | 用 `portable/react/table.tsx` 或 HTML/CSS，再套上 spec 的视觉标准 |
| 表单控件（Input / Select） | 用 `portable/react/input-select.tsx` |
| 日期选择器 | 用 `portable/react/date-picker.tsx` 或 HTML/CSS |
| 对象选择器（SearchableSelect） | 用 `portable/react/input-select.tsx` 的 searchable 模式（旧名 Combobox） |
| 弹窗 / 抽屉 | 用 `portable/react/dialog-drawer.tsx` |
| 分页器 | 用 `portable/react/pagination.tsx` |
| KPI 数字卡 | 用 `portable/react/number-card.tsx` |

### Portable Fallback 是什么

Portable Fallback 是"如果宿主仓没有同构组件"的最小实现。每个 spec 都提供了 React / HTML+CSS 的兜底代码。

**重点**：

- Portable 代码不依赖 shadcn / Radix / 其他库（只用纯 React / HTML/CSS）
- 代码已经 token 化，可以直接复制到宿主仓
- 这些代码的目的不是"长期运维"，而是"应急 fallback"——如果你们后续有自己的组件体系，可以逐步替换

---

## 6. 这套方案和之前有什么不一样

### 旧方案的问题

- 只有 demo 仓代码，宿主仓无法直接用（依赖 @radix-ui / shadcn / 大量内部组件库）
- 只有 Figma 设计稿，数据没有结构化（hex 值分散、规则模糊）
- AI 工具无法可靠执行（只能"参考"，容易自由发挥）

### 新方案的改进

- 提供了"可独立移植的 fallback"，不绑定 demo 仓代码
- 设计规范结构化：foundation.md / admin.md / tenant.md 分别定义基线
- 每个组件 spec 都包含"Visual Standard"表 + Portable Fallback，机器可检验
- 新加了"交付清单 / 迁移映射 / QA checklist"，产品前端可对标验收

---

## 7. FAQ

**Q：能直接从 demo 仓复制组件代码吗？**  
A：可以，但要看 spec 对齐。Demo 仓代码通常还有其他依赖（theme provider / 国际化等），如果你们不需要这些，直接用 Portable Fallback 更快。

**Q：Portable React 代码用了哪些库？**  
A：只用 React 原生 + 纯 CSS（Tailwind）。不用 @radix-ui、shadcn、lucide，避免依赖爆炸。

**Q：能自己改色值吗？**  
A：不能。宿主仓色值由设计决定，前端只负责"贴标准"。如果你的主题色不同，向设计要转译表，不要自己猜。

**Q：圆角能改吗？Admin 4px 太方了。**  
A：不能改 Admin。Admin 页面铁律就是 4px。Tenant 可以是 12px（见 foundation.md）。

**Q：这套规范以后还会改吗？**  
A：会。新的冲突 / 例外会写到 `references/conflict-log.md`；新加的组件会补到 `component-specs/` 和 `portable/`。每次更新会发新版本号（当前 2026-06-06）。

**Q：为什么有 portable/react、portable/html-css 两套？**  
A：前端通常用 React；如果你们用别的框架或纯 HTML，就用 HTML/CSS 版。

---

## 8. 这个包怎么用在 AI 协作中

如果你用 CodeBuddy / Cursor / ChatGPT 做换皮：

1. 给 AI 一整个 `.codebuddy/skills/clawpro-portable-design-skill/` 文件夹
2. 告诉 AI："按 component-specs/ 里的规范做，遇到宿主仓没有的组件就用 portable/ 里的 fallback"
3. AI 会自动拿 Portable Fallback 避免乱编

---

## 9. 周一前最后要做的

- [ ] 读完 foundation.md + admin.md（15 分钟）
- [ ] 读完 page-recipes.md（5 分钟）
- [ ] 按 page recipe 做出第一个列表页骨架（1 小时）
- [ ] 对照 qa/admin-checklist.md 验收（15 分钟）
- [ ] 如果有 Tenant 页面，再读 tenant.md 和对应 component-specs/（1 小时）

---

## 10. 遇到我这里没说的问题

1. 先看 `INDEX.md`（文件索引）
2. 再看 `HANDOFF.md`（交付方式 / 文件角色）
3. 最后看 `references/conflict-log.md`（已知冲突 / 待确认项）

**还是没找到？** 发给设计或 CodeBuddy 团队，他们会补到 conflict-log.md 并回复你。

---

**最后**：这不是一份"美观的设计规范"，而是一份"可执行的交付包"。看每个 spec 最重要的是"Visual Standard"表 + "Portable Fallback"代码，不是它写了多少页。祝周一换皮顺利！
