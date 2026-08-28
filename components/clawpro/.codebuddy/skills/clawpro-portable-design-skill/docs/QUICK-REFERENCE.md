# ⚡ 快速查询表 — 产品快速决策指南

> 做页面时卡住了？这里有决策表和快速答案。不要从 spec 开始读，先用这个表格找到你的问题。

---

## 1. 我想快速判断"这是什么组件"

### 按需求特征识别

| 我要做... | 对应组件 | 查看文档 |
|---|---|---|
| 显示一个数字（总用户数、总 Token、收入等） | **NumberCard**（KPI 数字卡） | `component-specs/number-card.md` |
| 显示一个没有数据的空白页 | **Empty**（空状态） | `component-specs/empty-state.md` |
| 让用户选一个日期 / 日期范围 | **DatePicker** | `component-specs/date-picker.md` |
| 让用户从下拉列表选一个（支持搜索） | **SearchableSelect**（原 Combobox） | `component-specs/input-select.md` |
| 显示一个表格 | **Table** | `component-specs/table.md` |
| 显示多选后的操作条（批量删除、批量审核） | **BatchActionsBar** | `component-specs/batch-actions-bar.md` |
| 显示一个二级菜单 / 右键菜单 | **DropdownMenu** / **Popover** | `component-specs/popover-dropdown-menu.md` |
| 让用户选"开启 / 关闭" | **Switch** / **Checkbox** / **Radio** | `component-specs/selection-controls.md` |
| 显示"成功 / 进行中 / 失败"等状态 | **StatusTag** | `component-specs/status-tag.md` |
| 显示"版本 / 类别 / 标签"标记 | **Badge** | `component-specs/badge.md` |
| 显示一个"弹窗 / 对话框 / 侧滑" | **Dialog** / **AlertDialog** / **Drawer** | `component-specs/dialog-drawer.md` |
| 显示一个表单（输入框 + 提交） | `form-controls.md` + `component-specs/button.md` | `component-specs/form-controls.md` |

---

## 2. 颜色怎么选

### Admin（管控端）配色速查表

| 用途 | 颜色值 | CSS Token | 说明 |
|---|---|---|---|
| 主操作按钮 / 链接 / 品牌 | #1447E6（品牌蓝） | `--cp-brand-blue` | 所有 CTA 都用这个 |
| 危险操作（删除 / 清空） | #DC2626（红） | `--text-danger` | 删除按钮、销毁确认 |
| 成功 / 通过 | #16A34A（绿） | `--text-success` | 成功提示、通过状态 |
| 警告 / 待处理 | #EA8C00（橙） | `--text-warning` | 待审核、超期 |
| 常规文字 | #0F172A（接近黑） | `--text-title` / `--cp-text-title` | 大标题、重要文字 |
| 次要文字 | #64748B（中灰） | `--text-weak` / `--text-secondary` | 辅助说明、灰色文字 |
| 不可用 / 禁用 | #A0AEC0（浅灰） | `--text-muted` / `--text-disabled` | 禁用按钮、不可操作项 |
| 背景（主） | #FFFFFF（白） | `--cp-surface` | 卡片、面板背景 |
| 背景（次级） | #F8FAFC（超浅灰） | `--bg-grey-normal` / `--cp-surface-muted` | 表格行、分组背景 |
| 背景（Hover） | #F1F5F9（浅灰） | `--bg-grey-hover` | 表格行 hover、菜单项 hover |
| 描边 | #E2E8F0（极浅灰） | `--cp-border` | 所有线条、边框 |

**核心规则**：
- ❌ 不要硬编码 hex（如 `#1447E6`）
- ✅ 用 CSS token：`var(--cp-brand-blue)` 或 `var(--text-danger)`
- 如果找不到 token，向设计要

---

## 3. 尺寸和圆角速查

### Admin 端规范值

| 元素 | 尺寸 | Token / 说明 |
|---|---|---|
| 按钮高度 | 36px | 标准按钮（大多数场景） |
| 按钮高度（Small） | 32px | 表格行内按钮、紧凑工具条 |
| 输入框高度 | 36px | 和按钮一致 |
| 表格行高 | 40px | 数据行；表头可能 48px |
| 卡片 / 表格 / 弹窗圆角 | **4px** | `--radius-lg` 或 `rounded-[4px]` |
| 头像 / 状态点 / Switch 圆角 | full | 100%（胶囊形） |
| 左侧栏宽度 | 240px | 固定，不可调 |
| 页面内边距（左右） | 24px | 列表页 / 详情页顶级 |
| 卡片内边距 | 16px | 卡片内部 padding |
| 间距（元素与元素） | 8px / 12px / 16px / 24px | 按层级递进 |

**核心规则**：
- ❌ 不要写 `rounded-xl` / `rounded-[8px]` / `rounded-2xl`
- ✅ Admin 几乎所有面板 = 4px（`--radius-lg`）
- ❌ 不要改左侧栏宽度
- ✅ 统一用 8/12/16/24 px 做间距

---

## 4. 文字层级速查

### Typography 规范值

| 用途 | 字号 | 粗细 | Token / 说明 |
|---|---|---|---|
| 页面大标题 | 16px | Semibold | `heading-3` |
| 卡片标题 / 弹窗标题 | 14px | Semibold | `heading-4` |
| 表格列头 | 14px | Semibold | - |
| 表格数据 | 12px | Regular | - |
| 普通正文 | 14px | Regular | `body` |
| 辅助说明 | 12px | Regular | `caption` |
| 按钮文字 | 14px | Medium | - |

**核心规则**：
- ❌ 不要用 11px / 13px / 15px / 18px / 20px
- ✅ 只用三档：12px / 14px / 16px
- ❌ 不要用 light / 300 粗细
- ✅ 只用：Regular / Medium / Semibold

---

## 5. 页面布局速查

### 三种典型页面骨架

#### 5.1 列表页（最常见）

```
┌─────────────────────────────────┐
│  [PageHeader]                   │  ← 标题 + 按钮
├─────────────────────────────────┤
│  [SearchFilterBar]              │  ← 搜索 / 筛选 / 刷新
├─────────────────────────────────┤
│  [Table]                        │  ← 数据表格
│                                 │
│                                 │
├─────────────────────────────────┤
│  [Pagination]                   │  ← 分页
└─────────────────────────────────┘
```

| 区域 | 组件 | 说明 |
|---|---|---|
| PageHeader | 标题 + 新增/导出/... 按钮 | 见 `component-specs/page-header.md` |
| SearchFilterBar | 搜索框 + 筛选条件 + 刷新按钮 | 见 `component-specs/search-filter-bar.md` |
| Table | 数据表格 | 见 `component-specs/table.md` |
| Pagination | 分页器 | 见 `component-specs/pagination.md` |

#### 5.2 详情页

```
┌─────────────────────────────────┐
│  [PageHeader]                   │  ← 返回 + 标题 + 编辑/删除 按钮
├─────────────────────────────────┤
│  [Tabs]                         │  ← 可选：多 Tab 分组
├─────────────────────────────────┤
│  [SurfaceCard]                  │  ← 基本信息卡
│  [SurfaceCard]                  │  ← 其他卡片
└─────────────────────────────────┘
```

#### 5.3 表单页

```
┌─────────────────────────────────┐
│  [PageHeader]                   │  ← 标题 + 说明
├─────────────────────────────────┤
│  [FormSection]                  │  ← 基本信息
│    Label: [Input]               │
│    Label: [Select]              │
│    Label: [DatePicker]          │
│                                 │
│  [FormSection]                  │  ← 其他部分
│    [Radio] 选项 A               │
│    [Radio] 选项 B               │
│                                 │
├─────────────────────────────────┤
│  [取消] [提交]                  │  ← Footer 按钮
└─────────────────────────────────┘
```

---

## 6. 常见错误 & 快速诊断

### 我的页面看起来不对，怎么排查？

#### 症状 1：按钮太大 / 太小 / 形状奇怪

| 症状 | 原因 | 修复 |
|---|---|---|
| 按钮是 48px 高 | 尺寸错误 | 改成 36px（标准）或 32px（小） |
| 按钮是全圆角（胶囊）| 这是客户端规则 | Admin 用方角（4px） |
| 按钮有渐变背景 | 这是客户端规则 | Admin 只有纯色 |
| 按钮用了"蓝底白字" | 正确 | — |
| 按钮用了"黑底白字" | 可能主操作用对了 | 次按钮应该白底黑字 |

**快速判断**：看 `component-specs/button.md` 的"Visual Standard"表

---

#### 症状 2：颜色看起来不对

| 症状 | 原因 | 修复 |
|---|---|---|
| 品牌蓝是 #007AFF（苹果蓝） | 错了，应该 #1447E6 | 找设计要转译表或改色值 |
| 删除按钮是蓝色 | 错了，删除用红色 #DC2626 | 改成 `variant="destructive"` |
| 文字是亮绿色 #00FF00 | 太亮了，应该深绿 #16A34A | 改 token 或找设计确认 |
| 表格行背景没有阴影 | 无阴影是对的，Admin 没有行阴影 | — |

**快速判断**：看本表"第 2 节 颜色怎么选"

---

#### 症状 3：圆角看起来奇怪

| 症状 | 原因 | 修复 |
|---|---|---|
| 卡片是 12px 圆角 | 这是客户端 | Admin 改成 4px（`--radius-lg`） |
| 按钮是 8px 圆角 | 还是太圆 | 改成 4px |
| 表格行是 12px 圆角 | 表格行一般不单独圆角 | 去掉或改 4px |
| 表格本身没有描边 | 对的，表格只有分割线 | — |

**快速判断**：看本表"第 3 节 尺寸和圆角"

---

#### 症状 4：文字大小不对

| 症状 | 原因 | 修复 |
|---|---|---|
| 标题是 18px | 太大了 | 改成 16px（`heading-3`） |
| 表格数据是 14px | 太大了 | 改成 12px |
| 按钮文字是 12px | 太小了 | 改成 14px |
| 所有文字都不一样 | 没有遵循 Typography 层级 | 查 PRODUCT-USAGE.md §4 |

**快速判断**：看本表"第 4 节 文字层级"

---

#### 症状 5：表格看起来不对

| 症状 | 原因 | 修复 |
|---|---|---|
| 表格列头背景是白色 | 应该是超浅灰 | 改成 `--bg-grey-normal` |
| 表格行没有分割线 | 有分割线是对的 | — |
| 表格行高是 48px | 太高了 | 改成 40px |
| 表格没有 hover 态 | 应该有 | 加 hover 时背景改 `--bg-grey-hover` |

**快速判断**：看 `component-specs/table.md` §Visual Standard

---

### 我还是不确定，怎么办？

1. **截一张图**，找设计或 CodeBuddy 团队看一眼（通常几分钟）
2. **贴到 `references/conflict-log.md`** Issue 区，标 `[URGENT]` 如果很着急
3. **临时方案**：用 `portable/react/*` 或 `portable/html-css/*` 里的代码替换，通常 100% 对标规范

---

## 7. "我要改XXX"的决策树

```
我要改配色
    ↓
问"能改吗"
    ↙           ↘
   不能           可能可以
   ↓             ↓
找设计          找设计要映射表
要转译表         (比如从蓝改绿)
   ↓             ↓
改 token         改完要 review
值             由设计确认
   ↓
全量替换

我要改圆角
    ↓
这是 Admin 还是 Tenant？
   ↙              ↘
 Admin          Tenant
   ↓              ↓
  不能            可能可以
  改             问设计
   ↓             (12px 固定吗？)
保持 4px            ↓
                  改后用新值
                  全量替换

我要改尺寸（高度/宽度/间距）
    ↓
这是核心组件还是自定义区域？
   ↙              ↘
核心组件      自定义区域
(Button/     (页面间距)
Table/Input)   ↓
   ↓       可以改，用
 不能改      8/12/16/24 px
   ↓       其中之一
问设计
   ↓
如果必须改
向设计要
新值确认

我要改文字大小
    ↓
这是按钮/表格/标题吗？
   ↙           ↘
  是          否
   ↓           ↓
不能改      看 Typography
   ↓       (12/14/16px 之一)
问设计        ↓
           不在列表里？
             ↓
           问设计加新尺寸
```

---

## 8. 我应该读哪个文档（按紧急度排序）

| 我现在要做... | 立即读 | 然后读 | 有疑问再读 |
|---|---|---|---|
| 做一个列表页 | `PRODUCT-USAGE.md` §1 | `page-recipes.md` + `component-specs/table.md` | `references/admin.md` |
| 做一个详情页 | `page-recipes.md` | `references/admin.md` | `component-specs/dialog-drawer.md` |
| 做一个表单页 | `page-recipes.md` | `component-specs/form-controls.md` | `component-specs/dialog-drawer.md` |
| 改一个按钮 | `component-specs/button.md` | — | `references/admin.md` |
| 改一个颜色 | 本表"§2 颜色怎么选" | `references/foundation.md` | `references/conflict-log.md` |
| 改一个尺寸 | 本表"§3 尺寸和圆角" | `component-specs/*.md` | 设计 |
| 宿主仓缺一个组件 | `PRODUCT-USAGE.md` §5 | `portable/react/*` 或 `portable/html-css/*` | — |
| 不知道该用哪个组件 | 本表"§1 我想快速判断" | `references/components.md` | — |

---

**最后**：这个表把最高频的问题都覆盖了。还有问题？贴 `references/conflict-log.md` 或找 CodeBuddy 团队。
