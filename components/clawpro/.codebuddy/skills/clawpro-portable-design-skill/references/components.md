# Components

> 组件决策表。目标是让 AI 和人类在写页面时先选对组件，而不是用 `div + className` 拼视觉。

## 1. 决策顺序

1. 已有业务组件能表达语义：直接复用。
2. 底层 UI 已有变体能表达语义：调用变体。
3. 只有结构不同：在业务层组合，不改底层组件。
4. 视觉 token 缺失：先提出新增 token / variant，再实现。
5. 不允许为了局部页面绕过全局组件规则。

## 2. 卡片与层级

| 需求 | 使用 | 禁止 |
|---|---|---|
| Admin 普通页面主卡 / 统计卡 / 列表容器 | `SurfaceCard` | 手写 `div.bg-white` + shadow / border |
| Tenant 业务对象卡 / Agent 卡 / 技能卡 | `TenantCard` | 用 Admin `SurfaceCard` 机械替代 |
| 卡片内部表格 / 分组容器 | `SurfaceInner` | 嵌套多个 `SurfaceCard` |
| Dialog / Sheet / Popover 自定义浮层 | `SurfaceOverlay` 或宿主仓浮层组件 | 自己写临时重阴影浮层 |
| 管控端引导 / Pro 推荐 / 高亮配置 | `SurfaceConfig` | 普通卡片硬加大阴影 |
| Segment active 滑块 | `--shadow-segment` | 手写阴影数值 |

## 3. Button

| 场景 | Admin / 全局 | Tenant |
|---|---|---|
| 主操作 | `variant="claw-primary"` | `variant="tenant-primary"` |
| 次级操作 | `variant="claw-outline"` | `variant="tenant-outline"` |
| 危险操作 | `variant="destructive"` 或 `tenant-destructive` | `variant="tenant-destructive"` |
| 仅图标方按钮 | `size="claw-square"` | 使用 tenant 对应变体 + 已定义 size |

硬性规则：

- 主按钮不在页面里 inline 写渐变，除非组件体系没有覆盖且有注释说明。
- 业务次级按钮不使用 shadcn `variant="outline"` 伪装；`outline` 只留给 SearchableSelect / Popover trigger 等控件外壳，并加 `allow-shadcn-outline`。
- 不用 `className` 覆盖变体里的核心颜色、圆角、边框、阴影。

## 4. Typography

新增用户端页面 / 组件优先使用语义文字组件：

| 语义 | 组件 |
|---|---|
| 页面标题 | `TenantPageTitle` / `TenantHeroTitle` |
| 大模块标题 | `SectionTitle` |
| Dialog / Sheet / 卡片区块标题 | `PanelTitle` |
| 卡片对象名 | `CardTitle` |
| 正文 / 表格 | `BodyText` |
| Label / Tab / 按钮文字 | `BodyMedium` |
| 时间 / 描述 / 空状态 | `MetaText` |
| 统计数字 | `StatNumber` |
| 表格内数字 | `InlineNumber` |
| Token / 路径 / 命令 | `CodeText` |

管控端老页面可渐进迁移，但触达文件时要清理明显 `text-gray-*` 老色阶。

## 5. Table

- 默认使用宿主仓已有表格或原生 `<table>` 结构，但视觉必须对齐 portable Table spec，不引入额外表格体系。
- 表头：`px-6 py-3 text-xs font-medium text-[var(--text-muted)] bg-gray-50/50`。
- 单元格：`px-6 py-4 text-xs text-[var(--text-body)]`；表格整体按 12px 口径执行。
- 空状态：表格内 `td colSpan` 居中提示，不额外包大卡片。

## 6. Dialog / Sheet / Popover

- 圆角跟随分端规则。
- 危险确认使用 `AlertDialog`。
- 弹窗中的 Info 图标：默认使用 `HoverCard`；内容可点击时使用 `Popover`；禁止用 `Tooltip` 承载长说明。
- 长表单加 `max-h-[90vh] overflow-y-auto`。

## 7. Badge / Status

| 状态 | 推荐 |
|---|---|
| 运行中 | `badge-running` 或 `StatusTag` success |
| 停用 / 错误 | `badge-stopped` / danger |
| 待处理 | `badge-pending` / warning |
| New / Beta | `TinyText` + 2px 圆角 / token 化 Badge |

不要发明新状态色；需要新增业务状态时先映射语义。

## 8. Input / Select / Filter

- 搜索框统一左侧 `Search` 图标，图标使用 `--text-weak`。
- 输入框、选择器、日期触发器描边统一使用 `--border` / `#EAEEF4` 蓝灰 token，焦点态使用品牌蓝弱边 / ring。
- 筛选区用 `gap-3`，不要每个控件单独写 margin。
- Popover / Select 内容通过 Portal 逃逸 overflow 裁剪。

## 9. 禁用写法速查

| 禁止 | 替代 |
|---|---|
| 自定义重阴影 | Surface 层级或 shadow token |
| `rounded-xl/2xl/3xl` 直接写业务卡片 | 读取分端规则，优先组件变体 |
| 自定义旧品牌主色 | `Brand Blue` / `Brand Black` |
| emoji 当 icon | `lucide-react` 或登记 SVG |
| 页面内任意新增 `fontFamily` | Typography / font token |
| 多层嵌套卡片 | 内部分组、分割线、`SurfaceInner` |
| 一页内重复自定义按钮样式 | Button variant |

## 10. 触达即同步

修改任意页面 / 业务组件时，顺手检查：

- 是否有旧色值、旧渐变。
- 是否有手写卡片 / 阴影。
- 是否有错用 `outline` 按钮。
- 是否有未登记图标路径。
- 是否能把标题、正文、Meta、数字迁移到 Typograp