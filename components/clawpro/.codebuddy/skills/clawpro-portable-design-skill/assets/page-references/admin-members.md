# 列表页 · 用户管理 (admin/members)

> 类别：**列表页（Segment + Search + Table + Pagination）**
> 路由：`/admin/members`
> 源码：`client/src/pages/admin/MemberManagement.tsx`（4225 行，含 Drawer/Dialog/Group 等丰富子模块）
> 截图：`./admin-members.png`（1440×900）

## 1. 视觉骨架（自上而下）

```
┌──────────────────────────────────────────────────────────────────┐
│ Top Banner Alert（同 AdminLayout 全局）                            │
│ AdminPageHeader · 标题"用户管理" + 副标题"管理企业用户的访问权限和资源配置" │
│ Toolbar Row（同行三段）                                            │
│  ├─ 左：Segment（全部 / 分组）+ Search                              │
│  └─ 右：IconButton(Download) + Button(添加用户) ▾                  │
│ Table                                                            │
│  列：用户 ID(?) · 分组 · 角色(Badge+icon) · 状态(StatusTag) ·       │
│       Agent 上限 · 每日 Tokens 上限 · 加入时间 · 操作（编辑/更多）   │
│ Footer Row                                                        │
│  ├─ 左：HelperText"共 19 名用户"                                    │
│  └─ 右：Pagination                                                  │
└──────────────────────────────────────────────────────────────────┘
```

## 2. 组件清单与 spec 对照

| 区域 | 组件 | import | spec |
|---|---|---|---|
| 页头 | `AdminPageHeader` | `@/components/ui/admin-page-header` | `component-specs/page-header.md` |
| 分类切换 | `SegmentGroup` + `SegmentOption` | `@/components/ui/segment` | `component-specs/segment.md` |
| 搜索 | `Input` (with `Search` icon) | `@/components/ui/input` | `component-specs/input-select.md` |
| 工具栏右侧 | `Button`（含 `DropdownMenu` 拆分按钮） | `@/components/ui/button` / `dropdown-menu` | `component-specs/button.md`、`component-specs/popover-dropdown-menu.md` |
| 表格主体 | `Table` 系列 + `TableActionCell` | `@/components/ui/table` | `component-specs/table.md` |
| 角色 Badge | `Badge` + `lucide` 图标 | `@/components/ui/badge` | `component-specs/badge.md` |
| 状态 | `StatusTag`（正常 / 禁用） | `@/components/ui/status-tag` | `component-specs/status-tag.md` |
| 弹层操作 | `Dialog` + `AlertDialog` + Drawer | `@/components/ui/dialog` / `alert-dialog`、`MemberDrawer.tsx` | `component-specs/dialog-drawer.md` |
| 选择/筛选 | `Select` / `Popover` / `HoverCard` / `Tooltip` | `@/components/ui/*` | 对应 spec |
| 反馈 | `Alert` / `toast`(sonner) | `@/components/ui/alert` / `sonner` | `component-specs/alert.md` / `toast.md` |
| 分页 | `Pagination` | `@/components/ui/pagination` | `component-specs/pagination.md` |

## 3. 关键 token / 规范要点

- Toolbar 行严格"两端对齐"：左 Segment+Search 一组，右 Toolbar 按钮一组，**不要换行**
- Segment 选中态 = `--cp-bg-strong`（白底）+ `font-medium`，**不要**用蓝色描边
- Search 占位文案统一 `搜索<对象> ID...` 格式
- Table 内联操作：**最多 2 个外露 + "更多"折叠**（这里就是"编辑 / 更多"）
- 角色单元格示例 `<Badge><UserIcon/>管理员</Badge>` —— 复合 icon+label
- 状态单元格用 `StatusTag`：正常=蓝、禁用=红——严格按 status-tag.md 调色板
- 表格底部 footer 行：**左数量 + 右分页**，不要把"共 X 条"放分页内
- "添加用户 ▾" 是 split-button：主按钮直接动作，▾ 弹 `DropdownMenu` 提供"批量导入"等次选项

## 4. 为何典型 (why-typical)

- 标准管理列表页骨架的"完整版"：覆盖 PageHeader / Segment 切换 / 搜索 / 工具栏右侧主操作 / 表格行内 Action / 分页
- 用户实体常见的全部列类型都出现：ID / 文本 / Badge / Tag / 数字 / 时间 / 操作
- 同源码内还展示了 Drawer 详情、AlertDialog 删除确认、批量操作 Bar 的组合用法

## 5. 易错点 / 反例

| ❌ 反例 | ✅ 正例 |
|---|---|
| 用 `Tabs`/`LineTabs` 做"全部/分组"切换 | 用 `SegmentGroup`，因为是**互斥过滤态**而非"内容章节" |
| 把"添加用户"放页头右上角 | 放在 Toolbar 行右侧（与表格语义紧贴） |
| 角色列写文本 `"管理员"` | 用 `Badge` + 角色 icon |
| 状态列写自定义颜色文本 | 用 `StatusTag` 走规范色板 |
| 操作列罗列 `编辑 / 删除 / 重置密码 / 转让`（4 个外露） | 保留主操作"编辑" + `更多` 收起，否则信息过载 |
| 分页放表格上方 | 分页固定在表底 footer 右侧 |
