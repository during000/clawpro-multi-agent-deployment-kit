# Page Recipes

> 常见页面模板。用于快速判断页面结构，而不是替代分端规范。

## 1. 列表页

适用：实例列表、技能列表、用户列表、审计日志、模型 / 通道列表。

Admin 完整页面默认前提：如果用户要的是“管控端页面 / Admin 页面 / 列表页页面”，且未明确说明只看局部组件预览，则必须放进 `AdminLayout` 语境，包含左侧 `AdminSidebar` + 主内容区；只有在用户明确要求“只生成表格 / 只验证列表容器 / 只做组件级 demo”时，才允许省略侧栏只做局部 Surface。

```text
PageHeader
Filters / Search
Data Surface
  Table / CardGrid
Pagination / BatchActions
Empty / Loading / Error
```

检查点：

- Header 主操作在右侧，文案为“动词 + 对象”。
- 搜索、筛选、刷新成组，使用 `gap-3`。涉及列表顶部工具条时优先对齐 `component-specs/search-filter-bar.md`。
- 普通列表可将筛选区、表格、分页放在同一数据 Surface；复杂筛选区可独立在表格 Surface 之外。
- 表格容器默认只承载批量操作条、表头、表体、空态、加载态和分页；标题、描述、主次操作按钮应放在表格外的页面区或独立区块，不与表格本体绑定。
- 分页紧跟数据区底部，不漂到页面底部或远离数据区域。
- 表格空状态放在表体内部；卡片列表空状态放在列表容器内。
- 批量操作必须显示选中数量和取消入口。

## 2. 表单页

适用：创建 / 编辑配置、策略、模型、通道、用户。

```text
PageHeader
SurfaceCard / SurfaceConfig
  Form sections
  Inline help / validation
Footer actions
```

检查点：

- 周一前只定义基础结构，不把左右双栏、步骤卡、Sticky footer 写成强制标准。
- 左右双栏、步骤卡、Sticky footer 后续按具体复杂页面补 recipe；本轮不作为前端执行硬要求。
- 字段按业务分组，不把所有字段平铺。
- 必填、只读、不可编辑原因清楚。
- Info 图标说明用 `HoverCard` 或 `Popover`。
- 危险选项与普通配置分离。
- 提交按钮禁用态说明原因。

## 3. 详情页

适用：Agent 详情、实例详情、技能详情、租户详情。

```text
Back / Breadcrumb
Summary header
Status / Key metadata
Tabs or Sections
Primary content + side info
Danger zone optional
```

检查点：

- 首屏必须能看出对象名称、状态、关键 ID、更新时间。
- Tab 不超过 5 个；更多内容用二级分组。
- 危险区放到底部并明确确认。
- 复杂详情避免多层嵌套卡片。

## 4. Dashboard

适用：监控、额度、统计、运营概览。

```text
PageHeader
Stats grid
Primary chart
Secondary charts / tables
Recent activity / alerts
```

检查点：

- 统计数字使用 `tabular-nums` / `StatNumber`。
- 图表主色 `#1447E6`，辅助线弱化。
- 数据为空 / 无权限 / 加载失败分别处理。
- 图表旁必须有解释，不只给线图。

## 5. 设置页

适用：平台策略、配额、企业配置、集成配置。

```text
PageHeader
Section nav optional
Config sections
Save / Reset actions
Audit hint optional
```

检查点：

- “保存”与“重置 / 取消”分组明确。
- 对影响范围大的配置展示提示横幅。
- 修改后未保存状态要可见。
- 权限不足时禁用并说明。

## 6. 能力开通页 Hero（动态背景 · 专项配方）

适用：管控端某能力「未开通 / 首次引导」整页（如云开发开通页）。**专项配方，不属于上面 5 类功能页型**——普通列表 / 表单 / 详情 / Dashboard / 设置页**不要**套动态背景 hero。命中条件见 `component-specs/dark-veil.md` §0 Auto-Trigger（场景 = 开通/能力 hero + 设计师拍板）。

```text
page-enter
  AdminPageHeader              ← 标题 + 描述（不在 hero 内）
  SurfaceCard (overflow-hidden)
    Hero 区 (relative overflow-hidden)
      背景三层（基底 + DarkVeil + 收束叠层，全部 pointer-events-none）
      内容层 (relative z-10)：图标徽章 + 主标题 + 描述 + 主操作 + 权益卡
    核心能力区：SectionTitle + SurfaceInner ×N
    底部说明
```

hero 背景骨架（参数 / 算法以 `component-specs/dark-veil.md` §4 为准，勿在此重定义）：

```tsx
import DarkVeil from "@/components/ui/DarkVeil"; // 依赖 ogl，宿主仓需先 `npm i ogl`（无 ogl 见下方兜底）

<div className="relative overflow-hidden px-[60px] py-12">
  <div className="pointer-events-none absolute inset-0 bg-[#E0EBFE]" />          {/* 基底 */}
  <DarkVeil speed={1.1} warpAmount={1.1} noiseIntensity={0.05} tintColor="#B2C3FF"
    className="pointer-events-none absolute inset-0 h-full w-full"
    style={{ transform: "translateY(72px)", maskImage: "linear-gradient(to bottom, transparent 0%, #000 22%)", WebkitMaskImage: "linear-gradient(to bottom, transparent 0%, #000 22%)" }} />
  <div className="pointer-events-none absolute inset-0 bg-gradient-to-b from-transparent via-white/10 to-[#E0EBFE]" /> {/* 收束 */}
  <div className="relative z-10 ...">{/* 内容 */}</div>
</div>
```

检查点：

- hero 是 `SurfaceCard(overflow-hidden)` 内顶部局部背景，**不是整页背景**；背景三层全部 `pointer-events-none`，内容 `relative z-10`。
- **ogl 注入提示**：DarkVeil 依赖 `ogl`（唯一新依赖）。宿主仓能装 → L0 完整动态；不便引 ogl/WebGL → 降 **L1** 纯 CSS 渐变（`portable/css/dark-veil.css`）；禁脚本/`reduced-motion` → **L2** 纯色或静态截图。分档见 `component-specs/dark-veil.md` §9 与 `references/migration-map.md`（DarkVeil 归 L1）。
- hero 内玻璃拟态装饰元素（图标徽章 / 权益卡）允许非 4px 圆角，属 hero 装饰例外（`SKILL.md` §2.4 / `references/conflict-log.md` C-018）；hero 以外一律 4px。
- 完整页面结构、图标资产、分区口径见 `references/admin-cloud-dev-activation.md`。

## 7. 空状态

唯一标准：页面级空状态优先使用 `client/src/components/ui/empty.tsx` 的 `Empty` 系列组件；弹窗、抽屉、表格、Popover 等容器内空态按下方容器场景降级，不要在页面内临时拼样式。

页面级结构：

1. `EmptyMedia`：轻量 icon 或已登记插画；默认居中。轻量 icon 容器 `size-10 bg-muted rounded-[4px]`，图标 `size-6`；PNG 空状态插画展示尺寸固定 `100×100`，使用 `h-[100px] w-[100px] object-contain`。
2. `EmptyHeader`：标题与描述成组，`max-w-sm`，居中，标题与描述间距 `gap-2`。
3. `EmptyTitle`：说明当前为空的对象，不只写“暂无数据”。
4. `EmptyDescription`：解释为什么为空，并给出下一步判断；描述文字可包含文档链接。
5. `EmptyContent`：放主操作或次级操作；最多一个主按钮 + 一个次级链接，避免把空态变成操作面板。

页面级版式：

- 容器：`flex min-w-0 flex-1 flex-col items-center justify-center gap-6 rounded-[4px] border-dashed p-6 text-center md:p-12`。
- 边框：`border border-dashed border-[var(--cp-border)]` 或宿主仓同义 token；Upload 上传区域与 Empty 共享 dashed 边框，但禁止使用默认 Upload 图标。
- 文本：标题使用 `EmptyTitle` / `SectionTitle` 语义，描述使用 `EmptyDescription` / `MetaText tone="weak"`。
- 插图：页面级默认使用已登记 PNG 空状态插画 `/assets/admin-resource-management/empty-no-data.png`（源文件 `client/public/assets/admin-resource-management/empty-no-data.png`，源图 250×250，展示尺寸固定 100×100）；业务强相关页面可复用其他已登记 empty-state 资产或 `lucide-react` 大图标；只有页面级 Empty 才允许装饰性图标 / 插画。

容器场景速查：

| 场景 | 推荐写法 | 关键规则 |
|---|---|---|
| 页面整块无数据 / 初始引导 | `Empty` + `EmptyMedia` + `EmptyHeader` + 可选 `EmptyContent` | 可配 icon / 插画；必须说明原因和下一步 |
| 卡片列表空状态 | 放在列表容器内的 `Empty` 或轻量 `EmptyHeader` | 不额外再包一层大卡片 |
| 表格空状态 | 表体内 `td colSpan` 居中提示 | 不另起大卡片；可用单行弱提示 |
| Dialog 内区块空态 | 纯文字 `HelperText`，`text-center py-12 space-y-1` | 禁止装饰图标；标题 / 描述同字号 |
| Drawer 分组空态 | `MetaText tone="weak"` 或 `HelperText` + `border border-dashed` | 添加入口默认放分组标题右侧，非必要不放框内 |
| Popover / 搜索结果空态 | 单行 `HelperText` | 只提示“没有匹配结果”等即时反馈，不放图标 |

单行 / 双行规则：

- 单行：用于表格、Popover、搜索无结果、紧凑分组等低权重空态，只写一个具体对象，如“暂无符合条件的实例”。
- 双行：用于 Dialog / Drawer 内嵌区块空态，第一行说明对象为空，第二行给出下一步，如“该角色还没有技能 / 可从公共技能库或企业技能库添加”。两行都用 `HelperText`，间距 `space-y-1`。
- 页面级：允许标题 + 描述 + 操作，但标题必须具体到业务对象，描述必须解释原因或下一步，不要只写“暂无数据”。

禁止：

- 不要只写“暂无数据”。
- 不用 emoji 作为空状态图标。
- 不在 Dialog / Drawer 内嵌区块加装饰性大图标。
- 不手写 `text-gray-*` / `text-[#...]` 表达空态文字色，优先使用 `EmptyDescription`、`MetaText tone="weak"` 或 `HelperText`。
- 不把大尺寸复杂插画用于表格、Popover、Drawer 分组等紧凑空态。

## 8. 错误态

| 场景 | 处理 |
|---|---|
| 网络失败 | 重试按钮 + 简短原因 |
| 权限不足 | 说明需要什么权限 / 找谁申请 |
| 配置缺失 | 指向具体配置入口 |
| 资源不存在 | 返回列表 + 刷新建议 |

## 9. 加载态

- 表格：骨架行或局部 spinner。
- 卡片列表：骨架卡片数量与最终布局一致。
- 按钮操作：按钮内部 spinner + disabled，不锁死整个页面。

## 10. 触达检查表

- 页面根是否 `page-enter`。
- 主次行动是否清楚。
- 是否有空 / 加载 / 错误 / 权限态。
- 是否使用正确分端组件变体。
- 是否使用已登记图标资产。
- 是否有旧 token 可顺手清理。
