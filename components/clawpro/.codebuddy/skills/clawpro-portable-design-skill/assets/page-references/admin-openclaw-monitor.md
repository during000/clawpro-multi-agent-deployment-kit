# Agent 列表页（富功能版） · admin-openclaw-monitor

> **类别**：列表页 · 富功能版（NumberCard 状态统计 + Search + 批量操作 + 列头筛选 + 行操作 + 监控抽屉）
> **路由**：`/admin/openclaw-monitor`
> **源码**：`client/src/pages/admin/OpenClawMonitor.tsx`
> **截图**：`admin-agent-datatable.png`（1440×900）

---

## 1. 视觉骨架

```
ConsoleShell（左导航）
└─ ContentArea
   ├─ AlertBanner（warning：私有网络 VPC 配额已耗尽 · 含「前往腾讯云控制台提交工单」链接）
   ├─ AdminPageHeader
   │  ├─ Title「Agent 列表」
   │  ├─ Subtitle「查看和管理由企业用户创建的 Agent 云服务器。」
   │  └─ Right: DateRange + RefreshIcon
   │
   ├─ NumberCard Group ×4  （grid-cols-4 gap=12）
   │  · 总数 24      (CountIcon · 蓝 · selected 态：实色边框 + 浅蓝底)
   │  · 运行中 18    (RunningIcon · 绿)
   │  · 已关机 1     (PowerOffIcon · 灰)
   │  · 其他 5       (InfoIcon · 灰)
   │
   ├─ Toolbar Row（白底无卡）
   │  ├─ Left: Input「搜索名称、ID 或创建人」（带 Search icon · 280px）
   │  └─ Right Buttons:
   │     ├─ Badge「3 个新版本」+ ChevronRight  (ghost · 触发版本面板)
   │     ├─ 批量更新       (ghost · disabled 态)
   │     ├─ 批量销毁       (ghost · disabled 态)
   │     ├─ 命令下发 ▾     (DropdownMenu)
   │     ├─ 配置默认标签
   │     └─ 智能体迁移
   │
   └─ Table（卡内）
      cols:
       □ 名称/ID                        (Checkbox + 双行 · ID 用 CodeText)
       当前状态 ▼   （列头带 Filter，状态值用 StatusTag · 绿点+"运行中"）
       创建人        (email + truncate + Tooltip)
       分组 ▼        （列头筛选）
       创建时间
       Agent 类型 ▼  （列头筛选 · 取值：Hermes Agent / OpenClaw / LightClaw ACE）
       Agent 版本    (CodeText)
       操作          (内联：终端 关机 删除 更多 ▾)
```

---

## 2. 组件清单与 spec 对照

| 区域 | 组件 | import | spec |
|---|---|---|---|
| 资源告警 | `Alert` warning + `AlertOperationInfoIcon` | `@/components/ui/alert` | `component-specs/alert.md` |
| 页头 | `AdminPageHeader` | `@/components/ui/admin-page-header` | `component-specs/admin-page-header.md` |
| 状态卡 | `NumberCard`（**支持 selected 态**） | `@/components/ui/number-card` | `component-specs/number-card.md` |
| 搜索 | `Input` + `Search` lucide icon | `@/components/ui/input` | `component-specs/input.md` |
| Badge | `Badge`（"3 个新版本"红点） | `@/components/ui/badge` | `component-specs/badge.md` |
| Dropdown | `DropdownMenu` 全套 | `@/components/ui/dropdown-menu` | `component-specs/dropdown-menu.md` |
| 列头筛选 | `Popover` + `Checkbox` 自定义组合（**列头点击触发**） | `@/components/ui/popover` | `component-specs/popover.md` |
| 状态值 | `StatusTag` | `@/components/ui/status-tag` | `component-specs/status-tag.md` |
| 表格 | `Table` 全套 + `TableActionCell` | `@/components/ui/table` | `component-specs/table.md` |
| 行操作下拉 | `DropdownMenu` | 同上 | 同上 |
| 抽屉 | `Drawer`（监控面板，本截图未展开） | `@/components/ui/drawer` | `component-specs/drawer.md` |
| 删除确认 | `AlertDialog` | `@/components/ui/alert-dialog` | `component-specs/alert-dialog.md` |
| 分组筛选 | `TreeSelect` | `@/components/ui/tree-select` | `component-specs/tree-select.md` |

---

## 3. 关键 token / 规范要点

- **NumberCard selected 态**（截图中"总数"卡）：`border: 1.5px solid tokens.color.brand.primary` + `bg: brand.primary/4`，**点击切换**会刷新表格筛选——所以视觉必须是按钮态而不是装饰。
- **Toolbar 区不是卡**：直接 `flex justify-between` 嵌在 ContentArea 上，与下方 Table 卡视觉分离 8~12px。
- **"3 个新版本" Badge**：左侧用 `Bell` icon + 数字 + 文字 + ChevronRight，整体 `Button variant="ghost"`，**红点**用 `bg-danger size-1.5 rounded-full absolute -top-0.5 -right-0.5`。
- **批量按钮 disabled 态**：在没有勾选行时 ghost + 50% 透明度，**不要直接隐藏**——保留位置感。
- **列头筛选交互**：列名后跟 `▼` 小箭头（`ChevronDown` 12px），点击弹 `Popover`，里面是带 Checkbox 的多选 + 底部"重置/确定"。激活时列名右侧显示蓝点。
- **行操作列**：终端、关机、删除是高频高优先级 → 直接平铺；其余收进"更多 ▾"。**链接色**用 `tokens.color.brand.primary`，hover 不下划线。
- **状态列优先级**：用 `StatusTag size="sm"`（绿点+文字 inline），不要用大号 Badge——会让行高被撑开。
- **email 列**：必 truncate，hover 显完整 Tooltip。

---

## 4. why-typical

- 是 ClawPro **最完整的列表页"全家桶"模板**：状态统计 / 搜索 / 批量 / 命令面板 / 列头筛选 / 行操作 / 抽屉详情 七大模式齐全。
- 比 `admin-members.md` 更丰富——后者只有 Segment + Search + Table，是"基础列表"；这页是"运维型富功能列表"。生成"管控类列表页"时**优先参考此页**。
- "NumberCard 可点击切筛选" 是 ClawPro 管控端独有交互模式，能复用到很多场景。

---

## 5. ❌反例 / ✅正例

| 场景 | ❌ 反例 | ✅ 正例 |
|---|---|---|
| Toolbar 与 Table 关系 | Toolbar 嵌在 Table 卡顶部内边距里 | Toolbar 在卡外，与卡间距 8~12px |
| 多个批量按钮 | 没选时全部隐藏 | 全部 disabled + 半透明保留 |
| 列头筛选 | 在表格上方加一行 FilterBar | **列头点 ▼ 弹 Popover**，与列绑定 |
| 行操作过多 | 全部平铺 6 个按钮 | 高优先 3 个 + "更多 ▾" |
| 状态列 | 用 Badge `<Badge variant="success">运行中</Badge>` | `<StatusTag tone="success">运行中</StatusTag>`（更轻量，行高不变） |
| email 列 | 完整显示挤占 200px+ | truncate + Tooltip，列宽 160 |
| NumberCard 数量 selected | 仅文字加粗 | 实色边框 + 浅底色（按钮态） |
| Alert banner 提示 | 用 toast 一闪而过 | **常驻 Alert**——配额耗尽是阻塞性提示 |
