# Admin Reference

> 管控端页面规范。适用：`client/src/pages/admin/**`、`AdminLayout`、`AdminSidebar`、管理后台业务组件。

## 1. 加载顺序

1. `foundation.md`
2. `components.md`
3. 本文件
4. 若遇到冲突，查 `conflict-log.md`

用户端专属规则不要套到管控端。

## 2. 布局

```text
Sidebar fixed white | Main flex-1 p-8 | bg #FFFFFF + url(/admin_content_bg.png) blue gradient image
```

- Sidebar：使用 `AdminSidebar` 组件与 `--admin-sidebar-*` token；展开宽度 `240px`，收起宽度 `64px`，`fixed`、`bg-white`、右侧 `border-r #EAEEF4`。
- Main：通过 `AdminSidebarInset` 统一承载浅蓝系渐变底图：`#FFFFFF` 兜底 + `url(/admin_content_bg.png)`，`cover` / `center top` / `no-repeat` / `fixed`；左侧导航 `ADMIN_NAV_GROUPS` 的所有到达页不得在页面根覆盖为纯白背景。
- Main：默认 `p-8`，铺满主内容区，不做用户端 1200/1920 响应式骨架。
- 页面根：保留 `page-enter`。
- 管控端暂不做小屏适配；不要引入整体横滚骨架。

## 3. 导航

| 元素 | 规格 |
|---|---|
| 分组标题 | `text-xs` / `--text-muted`，当前 `#64748B` |
| 菜单项 | `h-[34px] px-2 rounded-[4px] text-sm gap-2 #0A0A0A` |
| Hover | `--admin-sidebar-item-hover-bg`，当前 `rgba(180,191,225,0.14)` |
| Active | `--admin-sidebar-item-active-bg` 蓝灰渐变，当前代码值 `linear-gradient(90deg, #e9f3ff 0%, #e3eaff 100%)` |
| Icon | `lucide-react` 或登记 SVG，`w-4 h-4`；active 时 SVG 资产转纯黑 |
| Badge | 使用 `--admin-sidebar-badge-*` 专属 token，不套普通 Badge 视觉 |

活跃导航统一使用 `--admin-sidebar-item-active-bg` 蓝灰渐变背景；active 不是“文字蓝”规则，不再附加额外强调条。

## 4. 页面结构

推荐顺序：

0. 页面骨架：完整管控端页面默认先落在 `AdminSidebar + AdminSidebarInset` 内，不要把组件级 demo 误当成页面级交付。
1. Page header：标题、说明、主操作。
2. 筛选区：搜索、选择器、刷新。
3. 数据区：表格 / 卡片列表 / 配置区。
4. 反馈区：空状态、加载态、错误态。

页面标题：`Heading L` 或当前页面既有标题组件。操作按钮在标题右侧，主操作使用 `claw-primary`。

## 5. 配置卡与表单

- 管控端配置卡：`SurfaceConfig`，用于策略、模型、通道、计费、引导类配置块。
- 普通统计 / 列表卡：`SurfaceCard`。
- 卡内子区域：`SurfaceInner` 或分割线，不嵌套 L1 卡。
- 表单字段间距：`space-y-4` 或 `space-y-6`，Label 到控件 `space-y-2`。

## 6. 表格

- 表格视觉优先对齐 `component-specs/table.md`；宿主仓可保留原生 `<table>` 结构或既有表格组件，但不要自由发挥表头、行高和操作列样式。
- 表格仅展示表格本身；表格外若有标题、描述、统计说明、刷新/创建/更多操作按钮，统一放在表格容器外，不做成表格卡头。
- 表头固定浅灰，不要强烈色块。
- 行 hover `bg-gray-50/50`。
- 批量操作需有选中数量反馈。
- 删除、重置、停用等危险操作必须二次确认。

## 7. 常见页面模板

| 页面 | 模板 |
|---|---|
| 列表页 | Header + Filters + Data Surface + Pagination；普通列表可同卡，复杂筛选区可独立 |
| 配置页 | Header + `SurfaceCard` / `SurfaceConfig` + Form sections + Footer actions；左右双栏、步骤卡、Sticky footer 后续按具体页面补 recipe |
| 详情页 | Header + Summary + Tabs / 分区卡片 |
| Dashboard | Header + Stats grid + Chart + Recent table |

详见 `page-recipes.md`。

## 8. 禁止事项

- 不把“组件局部预览”当成“页面级预览”交付；只要用户要的是 Admin 页面，默认要带 `AdminSidebar`。
- 不使用用户端 `tenant-*` 按钮变体。
- 不使用用户端全圆角卡片 / 胶囊 Tab，除非组件本身为跨端共享且已明确支持。
- 不把落地页大图、装饰背景、营销动效引入管控端。
- 不在页面内自行新增大面积渐变背景；管控端页面背景统一继承 `AdminSidebarInset` 的浅蓝系渐变底图。
- 不新增非登记图标资产；优先 `lucide-react`。
