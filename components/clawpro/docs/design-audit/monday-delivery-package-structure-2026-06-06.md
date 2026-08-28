# ClawPro 周一交付包目录结构建议

> 目标：给产品前端、设计同事、AI 工具一套都能直接消费的设计交付包。
> 日期：2026-06-06

## 1. 推荐目录

```text
clawpro-portable-design-skill/
├── SKILL.md
├── README.md
├── STRUCTURE.md
├── references/
│   ├── foundation.md
│   ├── admin.md
│   ├── tenant.md
│   ├── landing.md
│   ├── components.md
│   ├── page-recipes.md
│   ├── migration-map.md
│   ├── assets-icons.md
│   └── conflict-log.md
├── component-specs/
│   ├── table.md
│   ├── card-surface.md
│   ├── button.md
│   ├── empty-state.md
│   ├── page-header.md
│   ├── dialog-drawer.md
│   ├── tabs-segment.md
│   └── form-controls.md
├── portable/
│   ├── html-css/
│   │   ├── table.html
│   │   ├── empty-state.html
│   │   └── card.html
│   ├── react/
│   │   ├── table.tsx
│   │   ├── empty-state.tsx
│   │   └── card.tsx
│   └── css/
│       └── tokens.css
├── tokens/
│   ├── design-tokens.json
│   ├── colors.md
│   ├── typography.md
│   ├── radius-shadow.md
│   └── spacing.md
├── qa/
│   ├── admin-checklist.md
│   ├── tenant-checklist.md
│   ├── landing-checklist.md
│   └── component-review-checklist.md
├── assets/
│   └── icon-registry.example.json
└── scripts/
    └── check-design-usage.mjs
```

## 2. 每层的职责

### 2.1 根目录文件

`README.md`

- 给人类看的总入口。
- 说明这包东西是干什么的、谁该先看什么、如何按场景使用。

`SKILL.md`

- 给 AI 的总入口。
- 只做路由、加载顺序、冲突仲裁，不堆大段细节规范。

`STRUCTURE.md`

- 解释整包目录的组织方式。
- 帮助外部 AI 或外部协作者快速找到正确文件。

## 3. `references/` 层

这层放稳定规则，不放单个组件的长篇细则。

`foundation.md`

- 品牌 token、字体、色彩、圆角、阴影、动效基线。

`admin.md`

- 管理端布局、导航、页面骨架、主区块风格。

`tenant.md`

- 客户端差异规则、骨架、卡片、按钮、背景、导航。

`landing.md`

- 落地页的品牌感和视觉方向。

`components.md`

- 组件决策表，先告诉人“该用谁”，不展开所有视觉细节。

`page-recipes.md`

- 列表页、表单页、详情页、Dashboard、空页面等常见页面模板。

`migration-map.md`

- 宿主仓已有组件 / 原生写法，应该映射到什么视觉标准。
- 这是前端换皮非常需要的一层。

`assets-icons.md`

- 图标、插画、空状态资产的来源、优先级和使用规则。

`conflict-log.md`

- 记录暂未裁决的冲突，不让执行人隐式裁决。

## 4. `component-specs/` 层

这层是 portable spec 的核心。

特点：

- 只收高风险组件。
- 每个文件都要能被“不是 demo 仓开发的人”直接使用。
- 必须包含 portable fallback。

建议周一前至少先有：

- `table.md`
- `card-surface.md`
- `button.md`
- `empty-state.md`
- `page-header.md`

## 5. `portable/` 层

这层是“脱离 demo 仓还能落地”的最强保障。

`portable/html-css/`

- 给没有 React 组件或不愿接你们组件体系的前端看。
- 提供最小可复现 HTML/CSS 示例。

`portable/react/`

- 给 React 项目看。
- 不强依赖你们完整项目结构，只保留最小结构和语义类名。

`portable/css/tokens.css`

- 给对方快速对齐视觉 token 的最小 CSS 变量版本。

这一层不需要周一就做得很全，但至少给几个关键组件做样例，会极大提升“前端愿不愿跟”。

## 6. `tokens/` 层

目的不是做设计系统大工程，而是把最关键的视觉常量抽出来，降低误读。

建议最少交付：

- `design-tokens.json`
- `colors.md`
- `typography.md`
- `radius-shadow.md`

如果时间不够，`spacing.md` 可后补。

## 7. `qa/` 层

这是交付包里最容易被低估，但最能减少返工的一层。

建议按端拆：

- `admin-checklist.md`
- `tenant-checklist.md`
- `landing-checklist.md`

再加一个组件通用验收：

- `component-review-checklist.md`

## 8. 周一现实版最小交付结构

如果时间只有两三天，最小化到下面这版也可以：

```text
clawpro-portable-design-skill/
├── SKILL.md
├── README.md
├── references/
│   ├── foundation.md
│   ├── admin.md
│   ├── tenant.md
│   ├── components.md
│   ├── page-recipes.md
│   ├── migration-map.md
│   └── conflict-log.md
├── component-specs/
│   ├── table.md
│   ├── card-surface.md
│   ├── button.md
│   └── empty-state.md
├── qa/
│   ├── admin-checklist.md
│   └── tenant-checklist.md
└── tokens/
    └── design-tokens.json
```

这版已经足够形成一套能被前端和 AI 直接消费的交付骨架。

## 9. 组件 portable spec 统一模板

下面这套模板建议所有 `component-specs/*.md` 统一套用。

```md
# {Component Name}

## 1. Purpose

- 这个组件解决什么问题。
- 适用在哪些页面 / 场景。

## 2. Scope

- 适用端：Admin / Tenant / Landing / Shared
- 必用场景：
- 不适用场景：

## 3. Visual Standard

| Item | Value | Notes |
|---|---|---|
| Background |  |  |
| Border |  |  |
| Radius |  |  |
| Shadow |  |  |
| Text |  |  |
| Spacing |  |  |

## 4. Anatomy

```text
Component
  Slot A
  Slot B
  Slot C
```

## 5. States

- default:
- hover:
- active / selected:
- disabled:
- loading:
- empty:
- error:

## 6. Demo Repo Usage

- 当前 demo 仓对应组件：
- 对应文件：
- 推荐调用方式：

```tsx
// 最小调用示例
```

## 7. Portable Fallback

### 7.1 React fallback

```tsx
// 不依赖 demo 仓业务组件的最小 React 结构
```

### 7.2 HTML/CSS fallback

```html
<!-- 最小 HTML 结构 -->
```

```css
/* 最小视觉还原 CSS */
```

### 7.3 If host repo already has components

- 如果宿主仓已有 Table / Card / Button 组件，应如何映射。
- 哪些可以复用。
- 哪些视觉必须覆盖。

## 8. Migration Rules

- 旧页面常见旧写法：
- 应迁移到的新写法：
- 可以暂时兼容的写法：
- 不允许新增的写法：

## 9. Do / Don't

Do:

- 
- 

Don't:

- 
- 

## 10. QA Checklist

- [ ] 视觉层级正确
- [ ] 状态完整
- [ ] 空态 / 加载态 / 错误态完整
- [ ] 跨仓 fallback 可执行
- [ ] 没有使用被禁止的旧写法

## 11. References

- Figma:
- Demo code:
- Related tokens:
- Related recipes:
```

## 10. 建议周一先完成的组件 spec 优先级

P0：

- `table.md`
- `card-surface.md`
- `button.md`
- `empty-state.md`

P1：

- `page-header.md`
- `dialog-drawer.md`
- `tabs-segment.md`

P2：

- `form-controls.md`
- `status-badge.md`
- `pagination.md`

