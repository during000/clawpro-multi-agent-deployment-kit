# 🚨 ClawPro Portable Design Skill - 阶段 A3 事故复盘分析

**执行时间**：2026-06-26  
**审计范围**：portable/react 实现代码 + portable/css 样式文件  
**目标**：发现实际存在或很可能出现的违规案例，分析绕过机制与根本缺陷

---

## 📌 执行摘要

通过对 **portable/react** 与 **portable/css** 的深度扫描，发现了 **7 个高频违规场景**，涉及：

| 场景 | 受影响组件 | 根本原因 | 优先级 |
|------|---------|--------|------|
| 1. 禁 lucide 槽位硬编码内联 SVG | alert.tsx | 文档指导模糊 | P1 |
| 2. 硬编码色值混用 Token | status-tag.css / segment.css / alert.css | 脚本盲区 | P1 |
| 3. 硬编码阴影不走 Token | segment.css | 脚本盲区 | P1 |
| 4. Toast 圆角误用 12px | toast/toast.css | 文档与工程断层 | P1 |
| 5. Alert 内联 SVG 路径混乱 | alert.tsx | 流程缺失自动化 | P2 |
| 6. Token 覆盖率不一致 | admin-sidebar.tsx | 脚本设计不全 | P2 |
| 7. 双份重定义规则 | alert.tsx vs alert/alert.tsx | 流程缺乏审查 | P3 |

**核心发现**：这 7 个违规**都能被当前脚本捕捉**，但 `check-design-usage.mjs` 的规则设计与执行策略存在 3 大缺陷，导致违规频繁滑过。

---

## 🔴 违规案例详解

### 案例 #1：禁 lucide 槽位硬编码内联 SVG（Alert 组件）

**📍 发现位置**

文件：`portable/react/alert.tsx`  
行号：115-125 (AlertInfoIcon)、128-142 (AlertWarningIcon)、145-150 (AlertSuccessIcon)

```tsx
// 第 115-125 行 - AlertInfoIcon
export function AlertInfoIcon(props: IconProps) {
  return (
    <IconShell>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
        <circle cx="12" cy="12" r="10" />
        <path d="M12 16v-4" />
        <path d="M12 8h.01" />
      </svg>
    </IconShell>
  );
}
```

**🚨 问题描述**

- Alert 组件的 6 个图标（Info / OperationInfo / Warning / Success / Error / ProductNews）全部**硬编码为内联 SVG**（`<svg>...</svg>`），而非引用 lucide-react 的 `<Info />` / `<CircleAlert />` 等
- 这是个"禁 lucide 槽位"（九槽禁用 lucide 规则不适用此处，Alert 图标本身是特殊情况）但**缺乏明确标记**

**🚫 绕过的规则**

- §2.8 图标规范：
  > "✅ lucide-react 或已登记 SVG"
- 脚本规则 `emoji-icon`：检查了 emoji 但**未涵盖硬编码 inline SVG 的检测**
- 脚本缺乏对"SVG path 硬编码"的规则（只检查 icon 引用注册表）

**🔴 根本原因**

1. **文档指导不明确**：SKILL.md §2.8 没有明确说明何时允许内联 SVG、何时必须用 lucide
2. **脚本覆盖不全**：`check-design-usage.mjs` 的规则列表（第 32-58 行）**没有对内联 SVG 路径的检测**
3. **流程缺乏自动化**：Self-Audit §8 第 4 项（"Empty 用 Empty 系列"）没有对应的"Alert icon 必须 lucide"的自检项

**💡 B/C 改动方向**

- **B 阶段**（文档）：
  - 在 SKILL.md §2.8 补充：
    ```
    ❌ 业务层不应硬编码 SVG paths（icon path=""）
    ✅ Alert / StatusTag 等框架级组件可内联 lucide 的 SVG path（带 /* allow-inline-svg */ 注释）
    ```

- **C 阶段**（脚本 + 流程）：
  - 在 `check-design-usage.mjs` 新增规则：
    ```javascript
    {
      id: 'inline-svg-path',
      message: '业务层不应硬编码 SVG path；仅框架级组件可用（需 allow-inline-svg 标记）',
      pattern: /<svg[^>]*>[\s\S]*?<(path|circle|line|rect)[^>]*\/?>[\s\S]*?<\/svg>/,
    }
    ```
  - 补充 pre-commit hook 自动验证

**✅ 验收用例**

改后，下列代码应被脚本拦住：

```tsx
// ❌ 业务页面硬编码 SVG path
function MyAlert() {
  return (
    <svg>
      <path d="M12 16v-4" />
    </svg>
  );
}

// ✅ 框架级组件内联（带注释）
export function AlertInfoIcon() {
  /* allow-inline-svg：框架级 Alert 组件内置 lucide Info icon */
  return (
    <svg>
      <circle cx="12" cy="12" r="10" />
    </svg>
  );
}
```

---

### 案例 #2：硬编码色值混用 Token（StatusTag CSS）

**📍 发现位置**

文件：`portable/css/status-tag.css`  
行号：28-32

```css
.cp-status-tag--blue   { color: #1447E6; }  /* 信息 / 进行中 */
.cp-status-tag--green  { color: #008236; }  /* 成功 / 在线 / 已完成 */
.cp-status-tag--red    { color: #DC2626; }  /* 失败 / 离线 / 错误 */
.cp-status-tag--orange { color: #B45309; }  /* 警告 / 即将到期 */
.cp-status-tag--gray   { color: #0A0A0A; }  /* 中性 / 未启用 */
```

**🚨 问题描述**

- StatusTag 的 5 个语义颜色全部**硬编码为 hex 值**（`#1447E6` / `#008236` / `#DC2626` 等）
- 这些值本应来自 `--cp-brand-blue` / `--cp-status-green` 等 token（见 portable/css/tokens.css 第 8-9 行定义了 `--cp-brand-blue: #1447E6`）
- 相同的颜色值在多个文件重复定义：`#1447E6` 出现在 status-tag.css / alert.css / badge.css / table.css / tokens.css

**🚫 绕过的规则**

- SKILL.md §2.1 颜色与 token：
  > "✅ 走 token：`<div className="... bg-[var(--cp-brand-blue)]" />`"
- SKILL.md §8 Self-Audit 第 1 项：
  > "没有硬编码颜色（`#xxxxxx` / `rgb(...)` / `text-gray-*`）"
- 脚本规则 `old-brand-color`（第 34-37 行）：只检查**旧品牌色** `#007AFF / #5856D6`，**不检查现行品牌色 `#1447E6` 的硬编码**

**🔴 根本原因**

1. **脚本设计缺陷**：规则只黑名单"旧色值"，不检查"正确颜色值的硬编码"
   - 脚本假设：新开发的代码"不会再硬编码旧色值"
   - 现实：新代码**硬编码新色值**，脚本无法检测
2. **文档指导模糊**：SKILL.md §2.1 说"走 token"，但没有示例说明"hex 硬编码"算违规
3. **流程缺失**：Self-Audit Checklist 是**手工清单**，没有强制执行

**💡 B/C 改动方向**

- **B 阶段**（文档 + Self-Audit 强制化）：
  - 在 SKILL.md §8 补充：
    ```
    - [ ] 没有硬编码任何 hex 颜色值（包括品牌色 #1447E6、黑 #020617、灰度值等）
        即使是正确的颜色，也必须来自 token（--cp-brand-blue 等）
    ```

- **C 阶段**（脚本增强）：
  - 修改 `check-design-usage.mjs` 规则为白名单模式：
    ```javascript
    {
      id: 'hardcoded-color-value',
      message: '发现硬编码 hex 颜色值；所有颜色必须来自 token（var(--cp-*)）',
      pattern: /(?<!var\(--)#[0-9A-Fa-f]{3,6}(?![0-9A-Fa-f])/,  // 负向断言排除 var(--color-xxx)
    }
    ```
  - 在 CSS 文件编译前运行 `lint:colors` 检查

**✅ 验收用例**

改后，下列代码应被脚本拦住：

```css
/* ❌ 硬编码色值（即使正确） */
.status-tag--blue { color: #1447E6; }

/* ✅ 走 token */
.status-tag--blue { color: var(--cp-brand-blue); }
```

---

### 案例 #3：硬编码阴影不走 Token（Segment CSS）

**📍 发现位置**

文件：`portable/css/segment.css`  
行号：45、98

```css
/* 行 45 - Admin Segment active 态 */
.cp-segment--admin .cp-segment__item--active {
  background: #FFFFFF;
  color: #020617;
  font-weight: 600;
  box-shadow: 0px 1px 2px rgba(0, 0, 0, 0.05);  /* ❌ 硬编码阴影 */
}

/* 行 98 - Tenant Segment active 态 */
.cp-segment--tenant .cp-segment__item--active {
  background: #FFFFFF;
  color: #020617;
  font-weight: 500;
  outline: 1px solid #CDD4DC;
  box-shadow: 0px 1px 4px rgba(0, 0, 0, 0.05);  /* ❌ 硬编码阴影 */
}
```

**🚨 问题描述**

- Segment 组件的激活态使用**硬编码 inline `box-shadow`**（`0px 1px 2px rgba(...)` / `0px 1px 4px rgba(...)`）
- 这些值应该来自 `--shadow-segment` token（见 SKILL.md §2.2 Surface 层级的阴影要求）
- 相同的模式在 admin-sidebar-style.css、input.css 等多处重复

**🚫 绕过的规则**

- SKILL.md §2.2 卡片与层级：
  > "❌ 手写卡片：`<div className="... shadow-md p-6">...</div>`"
- SKILL.md §7.2 Key Patterns：
  > "❌ 业务层 inline boxShadow 需要改用 Surface 层级"
- 脚本规则 `inline-box-shadow`（第 44-47 行）：
  ```javascript
  pattern: /boxShadow\s*:/,  // 只检查 JSX style 属性中的 boxShadow
  ```
  **问题**：这个正则**只匹配 React 的 `boxShadow:` style 属性**，**不能检测 CSS 文件中的 `box-shadow:` 属性**

**🔴 根本原因**

1. **脚本盲区**：规则设计针对 JSX/TSX，不覆盖 CSS 文件
2. **文档指导不完整**：SKILL.md §8 Self-Audit 第 1 项说"没有硬编码色值"，但**没有提到 box-shadow**
3. **流程缺失分层审查**：CSS 文件没有单独的审查流程

**💡 B/C 改动方向**

- **B 阶段**（文档）：
  - 在 SKILL.md §8 Self-Audit 补充第 9 项：
    ```
    - [ ] 没有硬编码阴影值（box-shadow / shadow-* / drop-shadow）
        所有阴影必须来自 token（--shadow-lg 等）或 Surface 组件
    ```

- **C 阶段**（脚本增强 + CSS lint）：
  - 扩展 `check-design-usage.mjs` 规则以覆盖 CSS：
    ```javascript
    {
      id: 'inline-box-shadow-css',
      message: 'CSS 中发现硬编码 box-shadow；请改用 --shadow-* token',
      pattern: /box-shadow\s*:\s*(?!var\(|none|inherit|initial)/,
    }
    ```
  - 补充 CSS lint 配置（stylelint）

**✅ 验收用例**

改后，下列代码应被脚本拦住：

```css
/* ❌ 硬编码阴影 */
.item--active {
  box-shadow: 0px 1px 2px rgba(0, 0, 0, 0.05);
}

/* ✅ 走 token */
.item--active {
  box-shadow: var(--shadow-sm);
}
```

---

### 案例 #4：Toast 圆角误用 12px（Toast CSS）

**📍 发现位置**

文件：`portable/css/toast/toast.css`  
行号：1-3、25

```css
/* toast/TOAST_TOKENS.css 第 1-2 行 */
--toast-radius: 0.75rem; /* rounded-xl = 12px */
/* ✓ 圆角：12px (rounded-xl) - ClawPro 标准圆角 */

/* toast/toast.css 第 25 行 */
.toaster > * {
  border-radius: 0.75rem; /* rounded-xl → 12px */
}
```

**🚨 问题描述**

- Toast 组件使用 **12px 圆角**（`rounded-xl`），这在**管控端（Admin）场景违规**
- SKILL.md §2.4 管控端铁律：
  > "几乎所有面板类元素圆角统一 4px"，"看到 Figma 标 12px 但当前页是管控端 → 不要照抄，落到 4px"
- Toast 是管控端的**跨越 Admin / Tenant 场景的浮层组件**，应该采用**管控端默认 4px**

**🚫 绕过的规则**

- SKILL.md §2.4 圆角规范（铁律）
- SKILL.md §8 Self-Audit 第 2 项：
  > "没有硬编码圆角（`rounded-xl` / `rounded-2xl` / `rounded-[8px]`）；管控端面板类元素必须 4px"
- 脚本规则 `large-radius`（第 49-52 行）：
  ```javascript
  pattern: /(?:^|[\s"'`])rounded-(?:md|lg|xl|2xl|3xl)(?:[\s"'`]|$)|rounded-\[(?:[8-9]|[1-9]\d+)px\]/,
  ```
  **问题**：规则**只检查 Tailwind 类名**（`rounded-xl`），**不检查 CSS 变量中的 rem 值**

**🔴 根本原因**

1. **脚本盲区**：规则针对 JSX className，不检查 CSS `:root` 或 `--radius` 变量的值
2. **文档分端混乱**：toast.css 没有明确标记"这是跨端组件，应该在 Admin/Tenant 之间选择"
3. **流程缺失场景判断**：Self-Audit 没有对"跨端浮层组件"的圆角决策规则

**💡 B/C 改动方向**

- **B 阶段**（文档 + 场景分流）：
  - 在 SKILL.md §0.2 或 references/components.md 中补充：
    ```
    跨端浮层组件（Toast / Tooltip / Popover）的圆角：
    - 在 Admin 页面内弹出 → 4px（--radius-lg）
    - 在 Tenant 页面内弹出 → 12px（--radius-xl）
    - Portable fallback：统一 4px（管控端优先）
    ```
  - 在 toast/TOAST_TOKENS.css 补充注释：
    ```css
    /* ⚠️ Admin 场景应该 4px；本文件默认 12px 供 Tenant 参考 */
    ```

- **C 阶段**（脚本 + 流程）：
  - 新增脚本规则检查 CSS 变量：
    ```javascript
    {
      id: 'css-var-large-radius',
      message: '检测到 CSS 变量中圆角值 ≥ 8px；管控端必须 4px（--radius-lg）',
      pattern: /--(radius|toast-radius|popover-radius)\s*:\s*(?:(?:0\.(?:[89]|[1-9]\d)rem)|(?:[1-9]\d*px))/,
    }
    ```
  - 在 Admin 页面集成时的检查清单中补充

**✅ 验收用例**

改后，下列代码应被脚本拦住：

```css
/* ❌ 管控端场景圆角不能 12px */
:root {
  --toast-radius: 0.75rem; /* rounded-xl = 12px */
}

/* ✅ 改为 4px（管控端默认） */
:root {
  --toast-radius: 4px; /* rounded-lg */
}
```

---

### 案例 #5：Alert 重定义与双份实现（Alert 组件重复）

**📍 发现位置**

文件 1：`portable/react/alert.tsx`（258 行）  
文件 2：`portable/react/alert/alert.tsx`（子目录重复）

```
portable/react/
├── alert.tsx                    ← 主文件
└── alert/
    └── alert.tsx                ← 重复文件
    
portable/css/
├── alert.css                    ← 主文件
└── alert/
    └── alert.css                ← 重复文件
```

**🚨 问题描述**

- Alert 组件存在**双份完整实现**：一份在根目录（`alert.tsx`），另一份在子目录（`alert/alert.tsx`）
- 两份文件几乎完全相同（可能是复制或目录结构重构遗留）
- 导致：
  - 维护困难（修复一个忘记修复另一个）
  - 导入路径混乱（`from "./alert"` 还是 `from "./alert/alert"`？）
  - 引入额外的 TokenBudget 成本

**🚫 绕过的规则**

- 这不是单个文件的违规，而是**流程缺失文件管理审查**
- 脚本 `check-design-usage.mjs` 没有**重复文件检测** 机制

**🔴 根本原因**

1. **流程缺乏自动化**：没有 pre-commit 或 CI 检查来检测同名组件的重复定义
2. **文档分工不明确**：SKILL.md 或 README 没有说明"alert/ 子目录是用于什么"（示例？废弃？）
3. **没有单一真实源**（Single Source of Truth）：设计体系没有明确规定组件放置位置

**💡 B/C 改动方向**

- **A 阶段（现在完成）**：
  - 运行 `find portable/{react,css} -name "*.tsx" -o -name "*.css" | sort | uniq -d` 检查所有重复

- **B 阶段**（流程 + 文档）：
  - 在 README 或 DEVELOPER-USAGE.md 补充：
    ```
    ## 文件组织规范
    
    portable/react/    ← 组件源码（单文件 per 组件）
    portable/css/      ← 样式文件（与 React 同名）
    
    ✅ 允许子目录的情况：
    - demo/            ← 演示页面
    - examples/        ← 使用示例
    
    ❌ 禁止：
    - alert.tsx + alert/alert.tsx（重复定义）
    ```
  - 删除重复文件，选择一个作为**唯一真实源**

- **C 阶段**（脚本 + CI）：
  - 新增脚本检查：
    ```bash
    #!/bin/bash
    # check-duplicates.mjs
    // 扫描同名 .tsx / .css 文件
    // 报警重复定义
    ```
  - 集成到 pre-commit hook

**✅ 验收用例**

改后，执行此命令应该输出为空：

```bash
$ find portable/react -name "*.tsx" | \
  sed 's|portable/react/||; s|/.*||' | \
  sort | uniq -d

# ✓ 无输出 = 无重复
```

---

### 案例 #6：Token 覆盖率不一致（AdminSidebar 与通用 Token）

**📍 发现位置**

文件 1：`portable/css/tokens.css`（全局 token）  
文件 2：`portable/css/admin-sidebar-style.css`（专用 token）  
行号：TOKENS.CSS 第 8-40 行 vs ADMIN-SIDEBAR-STYLE.CSS 第 1-30 行

```css
/* tokens.css - 通用 brand token */
:root {
  --cp-brand-blue: #1447E6;
  --cp-brand-black: #020617;
  /* ... 26 个通用 token ... */
}

/* admin-sidebar-style.css - 专用 token（重复定义同样的色值） */
:root {
  --cp-admin-sidebar-fg: #0A0A0A;          /* 可能应该是 --cp-brand-black？ */
  --cp-admin-sidebar-muted: #94A3B8;
  --cp-admin-sidebar-avatar-fg: #020617;   /* 同 --cp-brand-black！ */
  --cp-admin-sidebar-bg: #FFFFFF;
  /* ... */
}
```

**🚨 问题描述**

- AdminSidebar 的样式使用了**专用的 `--cp-admin-sidebar-*` token 集合**
- 但这些 token 中的许多**重复定义了通用 token 的值**（如 `#020617` 既在 `--cp-brand-black` 又在 `--cp-admin-sidebar-avatar-fg`）
- 导致：
  - 维护重复（修改一处忘记修改另一处）
  - Token 系统割裂（全局 vs 专用不清晰）
  - AI 生成代码时混乱（该用 `--cp-brand-black` 还是 `--cp-admin-sidebar-avatar-fg`？）

**🚫 绕过的规则**

- SKILL.md §1 原则第 3 项：
  > "Token 不硬编码"、"颜色、圆角、阴影、间距走 `--cp-*` token"
- 这里虽然用了 token，但 **token 本身的设计有冗余**

**🔴 根本原因**

1. **文档指导不明确**：SKILL.md 或 references/foundation.md 没有明确"何时创建专用 token vs 复用通用 token"
2. **脚本无法检查**：当前脚本**无法检测 token 值的重复定义**（只检查代码中是否用了 hex 值）
3. **流程缺乏 token 治理**：没有 token 审查流程来检查"新 token 是否与现有 token 重复"

**💡 B/C 改动方向**

- **B 阶段**（文档 + Token 体系治理）：
  - 在 references/foundation.md 中补充 Token 设计原则：
    ```
    ## Token 分层原则
    
    全局 Token（--cp-*）：
    - 品牌色：--cp-brand-blue / --cp-brand-black
    - 文本色：--cp-text-title / --cp-text-body / --cp-text-muted
    - 使用场景：页面通用，所有组件优先引用
    
    组件专用 Token（--cp-[component]-*）：
    - 仅当全局 token 无法满足、或值与全局 token 完全不同时创建
    - 例：--cp-admin-sidebar-active-bg（渐变背景）≠ 任何全局色值
    - 禁止：--cp-admin-sidebar-fg: #0A0A0A（=复用 --cp-brand-black，应该用全局）
    ```
  - 补充 token 命名规范与审查清单

- **C 阶段**（脚本 + Token Linter）：
  - 新增脚本检查 token 重复：
    ```javascript
    // check-token-duplication.mjs
    function detectDuplicateTokenValues(cssFiles) {
      const tokenMap = {}; // { value: [tokenName, ...] }
      // 解析所有 token 定义，找出相同值的重复
      // 报告"--cp-admin-sidebar-avatar-fg 与 --cp-brand-black 值相同，建议复用"
    }
    ```
  - 在 CI 中运行

**✅ 验收用例**

改后，应该满足：

```css
/* ❌ 不允许（重复定义） */
:root {
  --cp-brand-black: #020617;
  --cp-admin-sidebar-avatar-fg: #020617; /* 相同值 */
}

/* ✅ 改为复用 */
:root {
  --cp-brand-black: #020617;
  --cp-admin-sidebar-avatar-fg: var(--cp-brand-black);
}
```

---

### 案例 #7：NumberCard 图标槽位规范模糊（多 Variant 组件的同步缺陷）

**📍 发现位置**

文件：`portable/react/number-card.tsx`  
行号：60-90（Props 定义与实现）

```tsx
export interface PortableNumberCardProps {
  /** 自定义 React icon（与 iconSrc 二选一） */
  icon?: React.ReactNode;
  /** 项目内 SVG 图标路径 */
  iconSrc?: string;
  /** Icon 尺寸（默认 18） */
  iconSize?: number;
  label: React.ReactNode;
  value: React.ReactNode;
  // ...
}

export const PortableNumberCard = React.forwardRef<
  HTMLDivElement,
  PortableNumberCardProps
>(
  ({
    icon,
    iconSrc,
    // ...
  }, ref) => {
    // 两条路径都支持，但文档和 API 设计上"优先级模糊"
    const iconNode = iconSrc ? (
      <img ... />     // ← 优先外部 SVG
    ) : icon ? (
      <span className="cp-number-card__icon">{icon}</span>
    ) : null;
    // ...
  }
);
```

**🚨 问题描述**

- NumberCard 的 icon 槽位提供了**两条路径**：
  1. `iconSrc`（项目内 SVG 路径）
  2. `icon`（React 组件，支持 lucide-react 或渐变 SVG）
- 但**文档指导不明确**：应该优先用哪个？何时用哪个？
- 实现中通过**条件判断** (`iconSrc ? ... : icon ? ...`) 暗示了优先级，但**没有在 TypeScript 类型中约束**（两个都是可选的，都可以同时传）
- 导致调用方混乱：
  ```tsx
  // 这两种用法都是有效的，但不清楚哪个是"推荐"
  <NumberCard iconSrc="/icon.svg" label="..." value="..." />
  <NumberCard icon={<MyIcon />} label="..." value="..." />
  <NumberCard icon={<InputTokensIcon />} label="..." value="..." />  // 内置渐变
  ```

**🚫 绕过的规则**

- SKILL.md §2.9 KPI 概览：
  > "强制行为：一律用 `<NumberCard>`"，但**没有明确 icon 参数的最佳实践**
- 脚本 `check-design-usage.mjs` 没有对"组件 Props 组合"的检查

**🔴 根本原因**

1. **文档指导不完整**：component-specs/number-card.md 应该有"图标使用决策树"
2. **API 设计冗余**：两条 icon 路径（icon / iconSrc）没有约束条件（XOR），导致混淆
3. **流程缺乏变体审查**：组件支持多种 variant，但文档没有逐个说明

**💡 B/C 改动方向**

- **B 阶段**（文档 + API 明确化）：
  - 在 component-specs/number-card.md 中补充 Icon 决策树：
    ```
    ## Icon 选择指南
    
    ✅ 优先级 1：iconSrc（项目真实 SVG）
      - 何时用：有专属设计的业务图标
      - 例：<NumberCard iconSrc="/icons/requests-total.svg" />
      - 缺点：需要维护项目的 /public/icons/ 目录
    
    ✅ 优先级 2：icon={<GradientIcon />}（内置渐变）
      - 何时用：没有专属设计，用系统内置渐变样式
      - 例：<NumberCard icon={<InputTokensIcon />} />
      - 优点：自动彩色、无需外部依赖
    
    ✅ 优先级 3：icon={<LucideIcon />}（lucide 直接传）
      - 何时用：临时 demo、不追求渐变效果
      - 例：<NumberCard icon={<Zap />} />
      - 缺点：没有渐变、与设计稿可能不符
    
    ❌ 禁止组合：同时传 icon + iconSrc
      - 脚本应该拦住：if (icon && iconSrc) throw Error(...)
    ```
  - 更新 SKILL.md §9（高风险组件 Spec）中 NumberCard 的条目

- **C 阶段**（API 约束 + 脚本）：
  - 修改 TypeScript 类型强制 XOR：
    ```typescript
    export type PortableNumberCardProps = 
      | (BaseProps & { iconSrc: string; icon?: never })
      | (BaseProps & { icon: React.ReactNode; iconSrc?: never })
      | (BaseProps & { icon?: never; iconSrc?: never });
    ```
  - 在脚本中检测同时传入的情况
  - 补充 runtime warning（dev 环境）

**✅ 验收用例**

改后，下列代码应被 TypeScript 或脚本拦住：

```tsx
/* ❌ TS 错误：同时传 icon + iconSrc */
<NumberCard
  icon={<InputTokensIcon />}
  iconSrc="/icon.svg"  // ← TS Error: icon and iconSrc are mutually exclusive
  label="..."
  value="..."
/>

/* ✅ 只传 iconSrc */
<NumberCard iconSrc="/icon.svg" label="..." value="..." />

/* ✅ 只传 icon */
<NumberCard icon={<InputTokensIcon />} label="..." value="..." />
```

---

## 📊 3 大根本缺陷总结

| 缺陷 | 表现 | 根本原因 | 影响 | 优先级 |
|------|-----|--------|------|------|
| **文档指导不明确** | SKILL.md 对 portable 自包含策略约束不足；对"何时允许例外"缺乏明确定义 | 设计师未充分参与文档编写；设计 vs 工程的认知不一致 | AI 混淆规则优先级；开发者难以判断；Self-Audit 手工流程容易漏掉 | P1 |
| **脚本覆盖不全** | check-design-usage.mjs 有 7 个盲区（内联 SVG / CSS hex 值 / CSS box-shadow / 双份定义 / token 重复等） | 脚本只针对 JSX/TSX，不全面覆盖 CSS；只检查已知的"旧"问题，不检查"新"代码的同类问题 | 违规无法被自动捕捉；依赖手工 code review | P1 |
| **流程缺乏自动化** | Self-Audit 是手工清单（§8）+ 无 pre-commit hook；token 治理、重复检测、跨端判断都缺自动化 | 文档指导 vs 工程落地的断层；没有建立"持续执行"的审查机制 | 即便规则清楚、脚本完善，也容易被绕过；新人难以理解期望 | P1 |

---

## 🎯 改进优先级（B/C 阶段行动表）

| 优先级 | 阶段 | 改动 | 工期 | 检查项 |
|-------|------|------|------|--------|
| **P1** | **B** | 文档明确化（SKILL.md §2.8 图标、§2.1 颜色、§2.4 圆角、§8 Self-Audit 补充项） | 1-2 天 | 新文档通过设计师 review；AI 生成代码时通过 prompt 验证 |
| **P1** | **C** | 脚本增强（新增规则：inline-svg / hardcoded-color / css-var-radius / duplicate-check / token-duplication） | 2-3 天 | 脚本通过所有 7 个案例验证；CI 集成；pre-commit hook 工作 |
| **P2** | **B/C** | 流程自动化（Self-Audit 强制执行、pre-commit hook、Token 治理指南、API 约束） | 2-3 天 | pre-commit hook 拦住违规；CI 报告详细；开发者 DX 改善 |
| **P2** | **B** | 跨端场景明确化（references/components.md 补充"浮层组件在 Admin/Tenant 的适配"） | 1 天 | Toast / Popover / Tooltip 的圆角/阴影决策清晰 |
| **P3** | **B/C** | 技术债处理（Alert 去重、card 命名澄清、token 重复整理） | 1-2 天 | 文件结构清晰、无重复、导入路径一致 |

---

## 💾 输出清单

- ✅ 7 个违规案例详解（包含代码位置、问题描述、根本原因、改进方向、验收用例）
- ✅ 3 大根本缺陷分析
- ✅ B/C 阶段改进优先级表
- ✅ 每个案例的验收标准

**下一步**：交付 `audit-a3-incidents.json`（结构化数据）供 B/C 阶段计划使用。

---

*本报告为 ClawPro Portable Design Skill A3 事故复盘分析，执行于 2026-06-26。*
