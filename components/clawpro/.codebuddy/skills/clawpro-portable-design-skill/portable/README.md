# ClawPro Portable Design — 使用指南

> **Owner**: addietang  
> **版本**: v0.1（Wave 1，已覆盖 18 个核心组件 + 完整 Token / 全局规则）  
> **目标**: 任何外部仓拿到本目录，3 步集成即可 1:1 还原 ClawPro 管控端视觉。

---

## 📦 目录结构

```
portable/
├── README.md              ← 本文件
├── QA-CHECKLIST.md        ← 35+ 组件交付清单（哪些 Ready / Partial / Missing）
├── css/
│   ├── tokens.css         ← 设计 token（所有 --cp-* CSS 变量）
│   ├── globals.css        ← 字体 / 重置 / 表格 12px 强制 / 滚动条工具类
│   ├── portable.css       ← 总入口（按顺序 @import 上述三个文件 + 各组件 CSS）
│   ├── button.css         ← Button 独立 CSS
│   ├── admin-sidebar.css  ← AdminSidebar 独立 CSS
│   └── dark-veil.css      ← DarkVeil L1 静态兜底（无 ogl / WebGL 时的纯 CSS 飘带）
├── react/
│   ├── index.ts           ← 统一导出
│   ├── button.tsx
│   ├── card.tsx           ← Surface / SurfaceInner / SurfaceConfig / TenantCard
│   ├── table.tsx          ← Table / DataTable
│   ├── pagination.tsx
│   ├── empty-state.tsx
│   ├── number-card.tsx    ← 含 4 枚渐变 KPI 图标
│   ├── input-select.tsx   ← Input / Select / Field
│   ├── selection-controls.tsx ← Switch / Checkbox / Radio / RadioGroup
│   ├── date-picker.tsx
│   ├── dialog-drawer.tsx  ← Dialog / AlertDialog / Drawer
│   ├── tabs.tsx
│   ├── badges.tsx         ← Badge
│   ├── status-tag.tsx     ← StatusTag
│   ├── search-filter-bar.tsx
│   └── admin-sidebar.tsx  ← 21 子组件全量补齐
├── dark-veil/
│   └── dark-veil-static.tsx ← DarkVeil L1 静态兜底的 React 等价包装（无 ogl 时用）
├── html-css/
│   └── *.html             ← 18 个静态参考页（不依赖 React，直接浏览器打开预览）
└── demo.html              ← 双击即可预览所有现有组件（CDN 加载 React + Tailwind）
```

---

## 🚀 3 步集成

### Step 1 · 复制目录

把 `portable/` 整个目录拷到你项目（例如放在 `vendor/clawpro-portable/` 或 `src/clawpro-portable/`）。

### Step 2 · 引入 CSS

在你项目入口（`main.tsx` / `index.tsx` / `app.tsx`）顶部加一行：

```ts
import "./vendor/clawpro-portable/css/portable.css";
```

这一行会按顺序加载：
1. **tokens.css** — `:root { --cp-* }` 完整色板与阴影
2. **globals.css** — PingFang SC 字体、box-sizing、表格 12px 全局 `!important`、scrollbar-on-hover、page-enter 动画
3. **button.css / admin-sidebar.css** — 已抽离到独立 CSS 的组件样式

### Step 3 · 用组件

```tsx
import {
  PortableButton,
  PortableSurfaceCard,
  PortableDataTable,
  PortableSearchFilterBar,
  PortablePagination,
} from "./vendor/clawpro-portable/react";

export function MyPage() {
  return (
    <PortableSurfaceCard padding="lg">
      <PortableSearchFilterBar
        searchValue={search}
        onSearchChange={setSearch}
        actions={
          <>
            <PortableButton variant="claw-outline">刷新</PortableButton>
            <PortableButton variant="claw-primary">创建</PortableButton>
          </>
        }
      />
      <PortableDataTable
        rowKey="id"
        columns={[
          { key: "name", title: "名称" },
          { key: "status", title: "状态" },
        ]}
        dataSource={list}
        total={245}
        footer={
          <PortablePagination
            current={page}
            pageSize={10}
            total={245}
            onChange={setPage}
          />
        }
      />
    </PortableSurfaceCard>
  );
}
```

---

## ⚠️ 依赖说明

| 依赖 | 必需？ | 说明 |
|---|---|---|
| **React 17+** | ✅ 必需 | 所有组件均为 React 函数组件 |
| **Tailwind CSS 3+** | ✅ 必需 | 组件使用大量 Tailwind utility + 任意值语法 `[var(--cp-*)]` |
| **react-dom** | ✅ 必需 | Dialog / Drawer 使用 `createPortal` |
| **lucide-react** | ❌ 可选 | demo.html 用 CDN 自动注入；React 组件**不依赖** |
| **shadcn / @radix-ui / cva** | ❌ 不依赖 | 已主动剥离 |

---

## 🎯 视觉一致性铁律

1. **管控端圆角统一 4px**（`var(--radius)`），禁止 `rounded-xl` / `rounded-2xl`
2. **管控端表格字号统一 12px**（已用全局 `!important` 强制，写 `text-sm` 也会被覆盖）
3. **管控端按钮 variant** 只用 `claw-primary` / `claw-outline` / `destructive` / `outline-destructive` / `link` / `dialog-confirm`
4. **页面根部不要写 `bg-white`**（让背景继承 `AdminSidebarInset` 的渐变底图）
5. **表格操作列**只用品牌蓝文字按钮，间距 24px（gap-6），删除按钮**禁红色**

完整规则参见同级目录的 `../SKILL.md` § 2。

---

## 🧪 预览

**双击 `demo.html`** 即可在浏览器中看到所有现有组件 1:1 还原。

如果用的是 React 开发环境，运行：

```bash
# 在你项目内安装 Tailwind 后
npm run dev
```

然后访问任意页面即可。

---

## 📋 已覆盖组件（v0.1）

| Group | 组件 | 状态 |
|---|---|---|
| Foundation | `PortableSurfaceCard` / `PortableSurfaceInner` / `PortableSurfaceConfig` / `PortableTenantCard` | ✅ Ready |
| Action | `PortableButton`（6 variant × 3 size，独立 CSS） | ✅ Ready |
| Form | `PortableInput` / `PortableSelect` / `PortableField` | ✅ Ready |
|  | `PortableSwitch` / `PortableCheckbox` / `PortableRadio` / `PortableRadioGroup` | ✅ Ready |
|  | `PortableDatePicker` | ✅ Ready |
| Data | `PortableTable` / `PortableDataTable` / `PortableTableActions` | ✅ Ready |
|  | `PortablePagination` | ✅ Ready |
|  | `PortableEmpty` / `PortableTableEmpty` | ✅ Ready |
|  | `PortableNumberCard` + 4 枚 KPI 图标 | ✅ Ready |
|  | `PortableSearchFilterBar` | ✅ Ready |
| Feedback | `PortableDialog` / `PortableAlertDialog` / `PortableDrawer` | ✅ Ready |
| Navigation | `PortableTabs` 套件 | ✅ Ready |
|  | `AdminSidebar` 全套 21 子组件 | ✅ Ready（独立 CSS） |
| Tag | `PortableBadge` / `PortableStatusTag` | ✅ Ready |
| Decoration | `DarkVeil`（管控端 hero 动态背景，L1 静态 CSS 兜底 + React 包装；L0 完整版需 ogl） | ✅ Ready（L1） |

剩余约 20 个组件（Tooltip / Popover / Accordion / DropdownMenu / Form / Transfer / TreeSelect ...）将在 Wave 2/3/4 补齐。详见 `QA-CHECKLIST.md`。

---

## 💬 反馈

发现 1:1 还原不一致或样式 bug，请联系 **addietang**。
