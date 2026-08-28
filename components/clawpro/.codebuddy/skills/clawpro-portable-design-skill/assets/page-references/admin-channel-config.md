# 配置页 · 通道配置 (admin/channel-config)

> 类别：**配置页（Tabs + 行内开关 Table）**
> 路由：`/admin/channel-config`
> 源码：`client/src/pages/admin/ChannelConfig.tsx`（705 行）
> 截图：`./admin-channel-config.png`（1440×900）

## 1. 视觉骨架（自上而下）

```
┌──────────────────────────────────────────────────────────────────┐
│ AdminLayout                                                      │
│  ├─ Sidebar (left, 14% 默认折叠区可展开)                          │
│  └─ Main                                                         │
│      ├─ Top Banner Alert（"3 项基础配置未完成"）                  │
│      ├─ AdminPageHeader · 标题"通道配置"                          │
│      ├─ LineTabs：内置通道 / 自定义通道                           │
│      ├─ HelperText 描述段（一行说明 1-2 句）                      │
│      └─ Table（无 SurfaceCard 包裹，直接平铺）                    │
│          列：产品（Logo + 名称） / 应用范围（链接+笔图标）/ 用户可见（Switch） │
└──────────────────────────────────────────────────────────────────┘
```

## 2. 组件清单与 spec 对照

| 区域 | 组件 | import | spec |
|---|---|---|---|
| 顶部告警 | `Alert` + `AlertDescription` | `@/components/ui/alert` | `component-specs/alert.md` |
| 页头 | `AdminPageHeader` | `@/components/ui/admin-page-header` | `component-specs/page-header.md` |
| 分组切换 | `LineTabs` | `@/components/ui/line-tabs` | `component-specs/tabs.md` |
| 描述文案 | `HelperText` / `BodyText` | `@/components/ui/Typography` | （Typography 通用规范） |
| 表格 | `Table` 系列 + `TableActionCell` | `@/components/ui/table` | `component-specs/table.md` |
| 行内开关 | `Switch` | `@/components/ui/switch` | `component-specs/selection-controls.md` §Switch |
| 应用范围编辑 | `ScopeSelect`（业务包装） | `@/components/ScopeSelect` | — |
| 自定义通道增删 | `Dialog` + `AlertDialog` | `@/components/ui/dialog` / `alert-dialog` | `component-specs/dialog-drawer.md` |
| 删除确认 | `AlertDialog` | 同上 | 同上 |
| 通知反馈 | `toast`（sonner） | `sonner` | `component-specs/toast.md` |

## 3. 关键 token / 规范要点

- 表格 **不包 SurfaceCard**（属于"轻量配置型表"），直接坐在 PageHeader 下方留白上
- LineTabs 与表头之间留 `description` 段（HelperText），不要直接贴表头
- 行内 Switch 用 `Switch` + 行尾对齐，**不要在最右列再加"操作"列**——可见性即唯一切换
- 应用范围列用 `链接文本 + 编辑铅笔` 复合点击区，不用"全部用户"+"编辑"两列
- 顶部 Banner Alert 是 AdminLayout 全局提供的（基础信息未完成 banner），**业务页不要重复实现**

## 4. 为何典型 (why-typical)

- 配置页里**最常被复用**的骨架：分组（Tabs）→ 描述 → 表格
- 用一份 mock 数据驱动 6 行展示，覆盖了"启用 / 禁用 / 编辑入口"三种态
- 自定义通道 Tab 还演示了 **EmptyState + 添加按钮** 与 **Dialog/AlertDialog** 的标准联动（同源码）

## 5. 易错点 / 反例

| ❌ 反例 | ✅ 正例 |
|---|---|
| 用 `Tabs`（pills）做"内置/自定义"切换 | 用 `LineTabs`，下划线驱动 |
| 表格外面套一层 `SurfaceCard` 又给 `bg-white` 卡片 | 直接平铺，让 PageHeader+Tabs+表保持统一空间 |
| 行尾再加"编辑"按钮 + Switch 双控件 | 仅用 Switch，编辑入口放"应用范围"列内联 |
| Alert 用硬编码蓝底 + emoji | 用标准 `Alert` 组件 + `AlertInfoIcon` |
| 描述文案用 `<p className="text-sm text-gray-500">` | 用 `HelperText` 或 `BodyText tone="muted"` 走语义 token |
