# Button

## 1. Purpose

- 统一主操作、次级操作、危险操作、Tenant 差异按钮和轻量文本动作。
- 重点不是强绑 demo 仓 Button API，而是让宿主仓也能对齐按钮视觉与语义。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景：页头主操作、表单提交、次级操作、弹窗确认、Tenant 业务卡操作
- 不适用场景：纯文本链接、表格结构单元格、导航菜单项

## 3. Visual Standard

| Variant | Radius | Background | Border | Text | Scope |
|---|---|---|---|---|---|
| `claw-primary` | 4px | `--cp-brand-black` | none | white | Admin 主操作 |
| `claw-outline` | 4px | `--cp-surface` | `--cp-border` | `--cp-text-emphasis` | Admin 次级 |
| `dialog-confirm` | 4px | `--cp-brand-black` | none | white | 通用弹窗确认 |
| `tenant-primary` | full | `--cp-brand-black` | none | white | Tenant 主操作 |
| `tenant-outline` | full | `--cp-surface` | `--cp-border` | `--cp-text-emphasis` | Tenant 次级 |
| `tenant-destructive` | full | `--cp-text-danger` | none | white | Tenant 危险 |
| `tenant-outline-strong` | full | `--cp-surface` | `#cbcbcb` | `--cp-text-emphasis` font-medium | Tenant 中等强调 |
| `tenant-ghost` | full | transparent | none | `--cp-text-emphasis` | Tenant 低权重 |

### 3.1 Token 适用性矩阵（claw vs tenant 隔离规则）

> 本表是规范级约束，不是 demo 仓现状描述。宿主仓接入时必须严格遵守。

| 维度 | `claw-*` / `dialog-confirm`（共享） | `tenant-*` |
|---|---|---|
| 品牌黑底 | **必须**走 `var(--cp-brand-black)` | **允许**沿用稿件 hex（如 `#0A0A0A`），不强制对齐 token |
| 描边色 | **必须**走 `var(--cp-border)` (`#EAEEF4`) | **允许**沿用稿件 hex（如 `#E5E7EB` / `#cbcbcb`），不强制对齐 token |
| 危险底色 | **必须**走 `var(--cp-text-danger)` (`#DC2626`) | **允许**沿用稿件 hex（如 `#D42A1E`），不强制对齐 token |
| 文字强调色 | **必须**走 `var(--cp-text-emphasis)` 或 `var(--cp-brand-black)` | **允许**沿用 `text-gray-950` 等 Tailwind 实色 |
| 圆角 | 4px（不可改） | 9999px / `rounded-full`（不可改） |
| disabled 半透 | **必须**用同一 token 加透明度（如 `bg-[var(--cp-brand-black)]/40`），保证宿主仓覆盖品牌色时同步漂移 | 允许独立硬编码 |

**隔离意图**：

- `claw-*` / `dialog-confirm` 是 ClawPro 共享视觉资产，受 spec §3 约束，宿主仓覆盖品牌色时必须能整体迁移；
- `tenant-*` 视觉以 Figma 0522 设计稿为最终口径，hex 沿用稿件，避免随宿主仓主题色（admin 侧）漂移；
- 后续如需把 tenant 也并入 token 体系，必须走单独 spec PR 并由设计走查复核，不允许在常规迭代中顺手对齐。

### 3.2 黑底主按钮 hover / active 颜色（claw / shared 统一口径）

> 2026-06-10 起规范级约束。所有"黑底实心主按钮"hover 必须为 `#404040`，禁止再用更深的 `#1a1a1a`（与 active 区分度不足）。

| Variant | normal | hover | active | disabled |
|---|---|---|---|---|
| `default`（shadcn 默认 / 黑底主按钮） | `#0A0A0A` | **`#404040`** | `#000000` | `#0A0A0A/40` + 文字 50% |
| `claw-primary` | `var(--cp-brand-black)` | **`#404040`** | `#000000` | `var(--cp-brand-black)/40` + 文字 50% |
| `dialog-confirm` | `var(--cp-brand-black)` | **`#404040`** | `#262626` | `#A3A3A3` 白字 |
| `tenant-primary` | `#0A0A0A` | `#333333` | — | `#0A0A0A/50` |
| `tenant-dialog-confirm` | `#0A0A0A` | `#404040` | `#1a1a1a` | `#0A0A0A/50` |

**口径要点**：

1. **claw / shared 三件套（default / claw-primary / dialog-confirm）hover 必须是 `#404040`**，宿主仓不得回落到 `#1a1a1a`，否则与 active 几乎无差。
2. `tenant-primary` 沿用 `#333333`（稿件口径，§3.1 隔离），不与 claw 系列同步。
3. active 态保持各 variant 现状，不强求一致；本 spec 只锁 hover。
4. 颜色仍是硬编码 hex（不进 token 体系），但写进 spec 后宿主仓接入时**必须照抄**，不视为"页面级实现细节"。

## 4. Anatomy

```text
Button
  Icon optional
  Label
```

## 5. States

- default: 颜色、圆角、边框按端别区分。
- hover: 背景轻微变化，Tenant outline 可带轻阴影。
- active: 保持结构稳定，不做夸张压深。
- disabled: 降对比度，但不丢可读性。
- loading: 按钮内 loading，不锁全页。

## 6. Demo Repo Usage

- 当前 demo 仓组件：`client/src/components/ui/button.tsx`
- 典型 Admin 用法：`claw-primary`、`claw-outline`
- 典型 Tenant 用法：`tenant-primary`、`tenant-outline`

```tsx
import { Button } from "@/components/ui/button";

<Button variant="claw-primary" size="claw">创建</Button>
<Button variant="claw-outline" size="claw">取消</Button>

<Button variant="tenant-primary" size="claw">创建 Agent</Button>
<Button variant="tenant-outline" size="claw">详细配置</Button>
<Button variant="tenant-outline-strong" size="claw">管理通道</Button>
```

### 6.1 Demo 仓现状审计（截至 2026-06-09）

> 记录 demo 仓 `client/src/components/ui/button.tsx` 与本 spec 的对齐情况，给宿主仓接入时做参考。

**已对齐 spec（claw 侧 / 通用）**：

1. `claw-primary` / `dialog-confirm` 底色已改用 `bg-[var(--cp-brand-black)]`（原硬编码 `#0A0A0A`）。
2. `claw-primary` 的 disabled 态已改为 `disabled:bg-[var(--cp-brand-black)]/40`，宿主仓覆盖品牌色时 disabled 态可跟随。
3. `claw-outline` 文字色由 `text-gray-950`（#030712）改为 `text-[var(--cp-brand-black)]`（#020617），视觉无差但语义统一。
4. 本仓 `client/src/index.css` `:root` 已暴露 `--cp-brand-black: #020617;`，是 §7.2 fallback 的入口 token。

**故意保留独立 hex（tenant 侧 · §3.1 允许）**：

1. `tenant-primary` / `tenant-dialog-confirm` 底色保留 `#0A0A0A`，不走 `--cp-brand-black`。
2. `tenant-destructive` 底色保留 `#D42A1E`，hover `#B91C1C`，不走 `--cp-text-danger`。
3. `tenant-outline` 边框保留 `border-gray-200`（`#E5E7EB`），与 `claw-outline` 的 `#EAEEF4` 解耦。
4. 所有 `tenant-*` 文字色保留 `text-gray-950`（`#030712`），不走 `--cp-text-emphasis`。

**hover / active 态实色**（已按 §3.2 对齐 · 2026-06-10）：

1. `default` / `claw-primary` / `dialog-confirm` 的 hover 已**全部统一为 `#404040`**，原 `default` / `claw-primary` 的 `#1a1a1a` 已不再使用（视觉过深、易与 active 混淆）。
2. `tenant-primary` hover 保持 `#333333`、`tenant-dialog-confirm` hover 保持 `#404040`（§3.1 隔离）。
3. active 态各 variant 沿用现状（`claw-primary` / default → `#000000`、`dialog-confirm` → `#262626`），本轮不动。
4. 这些 hex 仍未纳入 token 体系，宿主仓如需主题化可自行覆盖；但**口径已写进 §3.2，不再视为"页面级硬编码"**。

## 7. Portable Fallback

### 7.1 If host repo already has Button

- 允许继续用宿主仓按钮组件。
- 但要用主题或 class 覆盖出 ClawPro 变体，不要求必须迁到 demo 仓的 `variant` 名字。
- 必须先分 Admin 与 Tenant，不得把一套 4px 按钮通吃所有端。

### 7.2 Minimal React fallback

```tsx
export function AdminPrimaryButton(props: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      {...props}
      className="inline-flex h-9 items-center justify-center gap-2 rounded-[4px] bg-[var(--cp-brand-black)] px-6 text-sm text-white"
    />
  );
}

export function TenantPrimaryButton(props: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      {...props}
      className="inline-flex h-9 items-center justify-center gap-2 rounded-full bg-[var(--cp-brand-black)] px-6 text-sm text-white"
    />
  );
}
```

### 7.3 Minimal HTML/CSS fallback

```html
<button class="cp-btn cp-btn-admin-primary">创建</button>
<button class="cp-btn cp-btn-tenant-outline">详细配置</button>
```

```css
.cp-btn { display:inline-flex; align-items:center; justify-content:center; gap:8px; height:36px; padding:0 24px; font-size:14px; }
.cp-btn-admin-primary { border-radius:4px; background:var(--cp-brand-black); color:white; border:0; }
.cp-btn-tenant-outline { border-radius:9999px; background:var(--cp-surface); color:var(--cp-text-emphasis); border:1px solid var(--cp-border); }
```

## 8. Migration Rules

- 旧写法：页面里手写一个黑按钮、白按钮、幽灵按钮。
- 新口径：先映射成 Admin / Tenant 语义，再决定宿主仓怎么实现。
- 可以暂时兼容：宿主仓旧按钮逻辑不动，仅覆盖视觉。
- 不允许新增：Admin 页面继续误用 Tenant 胶囊按钮；Tenant 页面继续用 Admin 4px 主按钮。
- **tenant 侧 hex 变更须走独立 spec PR**：在常规业务 / 视觉迭代中，不得顺手把 `tenant-*` 的 `#0A0A0A` / `#D42A1E` / `#E5E7EB` 等 hex 改为 `var(--cp-*)`，理由见 §3.1 隔离意图；如确需统一，须单独提 PR 并附设计走查记录。

## 9. Do / Don't

Do:

- 先判断端别。
- 主次操作层级明确。
- 用一个变体解决一类问题，不在页面里重复造按钮样式。

Don't:

- 不要用 `outline` 通用样式冒充业务次级按钮。
- 不要在页面内继续 inline 写核心颜色、圆角、阴影。
- 不要让 Tenant 和 Admin 按钮风格混用。

## 10. QA Checklist

**claw / shared 侧（必须全绿）**：

- [ ] 主次危险操作语义明确
- [ ] Admin 圆角为 4px
- [ ] `claw-primary` / `dialog-confirm` 底色走 `var(--cp-brand-black)`，**未硬编码** `#0A0A0A` / `#020617`
- [ ] `claw-outline` 边框走 `var(--cp-border)` (`#EAEEF4`)，文字走 `var(--cp-brand-black)` 或 `var(--cp-text-emphasis)`
- [ ] disabled 态半透明走同一 token（如 `bg-[var(--cp-brand-black)]/40`），不另行硬编码
- [ ] 宿主仓覆盖 `--cp-brand-black` 时，主按钮 / dialog-confirm / portable §7.2 fallback 视觉同步漂移
- [ ] `default` / `claw-primary` / `dialog-confirm` 的 hover 必须是 `#404040`（§3.2 锁定），**不**回落 `#1a1a1a`

**tenant 侧（隔离验收）**：

- [ ] Tenant 圆角为 `rounded-full`（9999px）
- [ ] hover 与 disabled 状态完整、可读
- [ ] tenant-* 的 hex 与设计稿（Figma 0522）一致，**不**因常规迭代被改成 `var(--cp-*)`

## 11. References

- Demo code: `client/src/components/ui/button.tsx`
- Related rules: `references/components.md`
- Related rules: `references/admin.md`
- Related rules: `references/tenant.md`


## 12. 代码对照（✅/❌）

> 与 SKILL.md §2 同口径。每组对照覆盖一种典型误用 → ClawPro 正确写法。

### 12.1 颜色 token 隔离（claw vs tenant）

```tsx
// ❌ Admin 主按钮硬编码 hex，宿主仓覆盖品牌色时无法漂移
<button className="rounded-[4px] bg-[#0A0A0A] text-white">创建</button>

// ❌ Tenant 按钮强行套到 --cp-* token，违反 §3.1 隔离规则
<Button variant="tenant-primary" className="bg-[var(--cp-brand-black)]">创建 Agent</Button>

// ✅ Admin / Shared 走 token，宿主仓主题可整体迁移
<Button variant="claw-primary" size="claw">创建</Button>

// ✅ Tenant 走稿件 hex（写在 button.tsx 里），不跟随 admin 主题
<Button variant="tenant-primary" size="claw">创建 Agent</Button>
```

### 12.2 圆角端别分流（Admin 4px / Tenant 胶囊）

```tsx
// ❌ Admin 页面用胶囊按钮（混了 Tenant 视觉语言）
<Button className="rounded-full bg-black text-white">保存</Button>

// ❌ Tenant 业务对象用 4px 直角按钮（破坏 Tenant 端识别度）
<Button variant="claw-primary">创建 Agent</Button>

// ✅ Admin 主操作：4px
<Button variant="claw-primary" size="claw">保存</Button>

// ✅ Tenant 业务对象主操作：rounded-full
<Button variant="tenant-primary" size="claw">创建 Agent</Button>
```

### 12.3 variant 选择（禁止 outline 借用做次级业务按钮）

```tsx
// ❌ 拿 shadcn 通用 outline 冒充业务次级，圆角/边框/字色都不对
<Button variant="outline">详细配置</Button>

// ✅ Admin 次级
<Button variant="claw-outline" size="claw">取消</Button>

// ✅ Tenant 次级（中等强调时用 outline-strong）
<Button variant="tenant-outline-strong" size="claw">管理通道</Button>
```

### 12.4 disabled 半透同 token（保证主题漂移）

```tsx
// ❌ disabled 自己写一个独立颜色，宿主仓换品牌色时 disabled 不跟随
<button className="rounded-[4px] bg-[#666] text-white opacity-100" disabled>保存</button>

// ✅ 走同一 token + 透明度，主题色变了 disabled 自动同步
<Button variant="claw-primary" size="claw" disabled>保存</Button>
// 内部实现：disabled:bg-[var(--cp-brand-black)]/40
```

### 12.5 危险确认（Admin 走二次确认 / Tenant 用 destructive variant）

```tsx
// ❌ Admin 列表里直接放红字"删除"按钮当主操作
<Button className="bg-red-600 text-white">删除</Button>

// ❌ Tenant 危险按钮硬编码 hex（应走 variant，不走 className）
<Button className="bg-[#D42A1E] rounded-full text-white">解除绑定</Button>

// ✅ Admin：操作列用蓝色文字按钮 + 二次确认弹窗承担危险语义（见 table.md §5）
<button className="text-[var(--cp-text-brand)] hover:underline text-sm">删除</button>

// ✅ Tenant：危险操作用 destructive variant，不在外层叠 className
<Button variant="tenant-destructive" size="claw">解除绑定</Button>
```
