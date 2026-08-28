# Migration Map

> 这份映射表不是写给 demo 仓自身的，而是写给“要在宿主仓换皮的人”。

## 1. 使用原则

- 优先复用宿主仓已有组件。
- 但视觉标准必须对齐 ClawPro portable spec，而不是沿用宿主仓旧视觉。
- 如果宿主仓没有对应组件，退回到最小 React 或 HTML/CSS fallback。

## 2. 高风险组件映射

| 宿主仓现状 | 应映射到 | 必看文件 | 备注 |
|---|---|---|---|
| 原生 `<table>` + 自写 class | ClawPro Table 标准 | `component-specs/table.md` | 先对齐表头、行高、操作列、空态 |
| `div.bg-white rounded... shadow...` 手写卡片 | `SurfaceCard` / `TenantCard` 语义 | `component-specs/card-surface.md` | 先判断 Admin 还是 Tenant |
| 宿主仓已有 `Button` | 映射到 `claw-*` / `tenant-*` 视觉 | `component-specs/button.md` | 不要求必须引入当前 demo Button |
| 页面内临时空态 | 统一空状态模块 | `component-specs/empty-state.md` | 特别注意页面级与容器级分流 |
| 手写搜索 + 筛选 + 刷新条 | `Search / Filter Bar` 结构 | `component-specs/search-filter-bar.md` | 先统一分组和高度，再映射宿主仓控件 |
| 手写日期触发器 / 旧日期库外壳 | `Date Picker` 结构 | `component-specs/date-picker.md` | 保留逻辑，先统一 trigger 和日历选中态 |
| 手写搜索型对象选择器 | `Combobox` 结构 | `component-specs/combobox.md` | 保留候选项逻辑，统一 trigger、panel、搜索、选中态和空态 |
| 手写表格多选 / 批量操作浮条 | `Batch Actions Bar` 结构 | `component-specs/batch-actions-bar.md` | 明确选中数量、跨页选择、清除入口和危险确认 |
| 手写 Popover / Dropdown / Tooltip 浮层 | `Popover / Dropdown Menu` 结构 | `component-specs/popover-dropdown-menu.md` | 统一 4px、Portal、overlay shadow、危险操作确认 |
| 页面内自建 Admin 侧栏 | `Admin Sidebar` 结构 | `component-specs/admin-sidebar.md` | 240 / 64 / 4px 圆角，active 弱蓝渐变 |
| 页面内自建 Tenant 顶栏 | `Tenant TopNav` 结构 | `component-specs/tenant-topnav.md` | 64px / 三栏 / 32 图标按钮 / 31 圆形头像 |
| 页面内自建面包屑 | `Breadcrumb` 结构 | `component-specs/breadcrumb.md` | 当前页不可点击，分隔符弱灰 |
| 整页 spinner / 临时 loading | `Loading / Progress` 结构 | `component-specs/loading-progress.md` | 分首屏、局部、按钮、进度状态 |
| 图表和统计卡片自定义配色 | `Chart / Stat` 结构 | `component-specs/chart-stat.md` | 主色品牌蓝，数字和 Tooltip 统一 |
| 默认上传区 / 文件列表 | `Upload / File Browser` 结构 | `component-specs/upload-file-browser.md` | 统一 dashed、进度、错误和文件空态 |
| 手写标题 + 操作按钮条 | `AdminPageHeader` 或等效结构 | `component-specs/page-header.md` | 标题、说明、操作区要统一 |
| 自写 canvas / 渐变动态背景（开通页 / 能力 hero） | DarkVeil 动态背景，**默认兜底档 L1（纯 CSS 渐变）**；宿主仓能装 `ogl` 则升 L0 完整移植 | `component-specs/dark-veil.md` | 纯装饰、仅 hero，命中 §0 Auto-Trigger 才用；无 ogl/WebGL → **L1** 纯 CSS 光晕，禁脚本/`reduced-motion` → **L2** 纯色 `#E0EBFE` 或静态截图。单页最多 1 个实例，不扩散到列表/表单/整页 |

## 3. Admin 页面迁移口诀

1. 先保住页面背景和页头骨架。
2. 再统一卡片层级。
3. 再统一表格和筛选区。
4. 最后清理旧色值、旧阴影和旧圆角。

## 4. Tenant 页面迁移口诀

1. 先确认当前页面是否属于 Tenant 差异规则。
2. 按 Tenant 按钮、TenantCard、Tenant 背景执行。
3. 逐步替换散落的文字样式到 Typography 语义。
4. 不把 Admin 的 4px 卡片直接机械套进 Tenant 页面。

## 5. 常见旧写法到新写法

| 旧写法 | 新口径 |
|---|---|
| 旧品牌蓝紫体系 | 统一 Brand Blue / Brand Black 体系 |
| 临时重阴影卡片 | 改为 Surface 层级或阴影 token |
| `rounded-xl/2xl/3xl` 手写业务卡片 | 先判断端，再映射到 `SurfaceCard` / `TenantCard` |
| 页面级 `暂无数据` 自拼样式 | 改为统一 Empty 结构 |
| 表格内红色文字按钮当危险操作 | 优先统一到 Table 操作列模式 + 二次确认 |
| 手写页头标题区 | 改为统一页头结构 |

## 6. 可以暂时兼容，但不建议新增

- 宿主仓已有可用的 Button / Dialog / Input 组件，但视觉未完全一致。
- 部分历史页面继续使用原生 `<table>`，但新改动需优先向 Table spec 靠拢。
- 历史页面未彻底接入 Typography，但新增区域不要继续扩大旧写法。

## 7. 明确禁止新增

- 新增旧品牌色。
- 新增重阴影。
- 在 Admin 页面新增 Tenant 式胶囊卡片。
- 在 Tenant 页面新增 Admin 式 4px 业务卡片。
- 新增只有 demo 仓能运行的页面实现而不写 fallback。
