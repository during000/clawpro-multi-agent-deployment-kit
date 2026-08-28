# 按钮组件规范（ClawPro Button）

> 与 Figma 「按钮」ComponentSet `317:1051` 严格对齐。
> 所有按钮统一通过 `@/components/ui/button` 中的 `Button` 组件 + `claw-*` 变体来实现，**禁止**再手写 `border / 圆角 / 高度 / 颜色` 的 inline style 或 className。
> 适用仓库：`openclaw-enterprise/client/src`　最近更新：2026-05-14

---

## 一、为什么需要这份规范

历史代码里到处出现这种重复样式（同一个"线性描边按钮"被各处复制）：

```tsx
// ❌ 不要这样写
<Button
  variant="outline"
  className="h-9 px-6 text-sm font-normal rounded-[4px]"
  style={{ borderColor: "#E5E5E5", color: "#020617" }}
>
  详细配置
</Button>
```

带来的问题：
- 设计稿改 token（如悬浮态加了浅蓝渐变），需要全局搜索替换；
- 不同页面会因为复制时漏几个属性而出现细微不一致；
- AI 在阅读现有代码时容易复用错样式（比如把 ant 风格 outline 当成 Figma 风格）。

把这一切一次性收敛到 `Button` 组件的 variant 里，调用方只需要写 `variant + size`，不再传任何样式覆盖。

---

## 二、Token 表（来自 Figma 317:1051）

| Token | 值 |
|---|---|
| **claw-outline / 背景** | `#FFFFFF` |
| **claw-outline / hover 背景** | `#F5F5F5` |
| **claw-outline / 边框** | `1px solid #E5E5E5` |
| **claw-outline / hover 边框** | `1px solid #E3E3E3` |
| **claw-outline / 文字色** | `#020617` |
| **claw-primary / 背景** | `linear-gradient(90deg, #020617 70%, #1447E6 100%)` |
| **claw-primary / hover 背景** | `linear-gradient(90deg, #020617 70%, #0A226F 100%)` |
| **claw-primary / 文字色** | `#FFFFFF` |
| **圆角** | `4px` |
| **icon 尺寸** | `16×16` |

---

## 三、变体（variant）

| variant | 用途 | 视觉描述 |
|---|---|---|
| `claw-outline` | 次级按钮（卡片底部、对话框右下次操作） | 白底灰描边，hover 浅灰底 `#F5F5F5` + 灰描边 `#E3E3E3` |
| `claw-primary` | 主操作按钮（创建 Agent、提交表单等） | 黑→蓝渐变 + 白字，hover 加深 |

> shadcn 自带的 `default / outline / destructive / secondary / ghost / link` 仍然保留，与 Figma 规范并存，但**新代码不要再用 `outline` 表达 ClawPro 设计稿里的次级按钮**——请用 `claw-outline`。

---

## 四、尺寸（size）

| size | 维度 | 用途 |
|---|---|---|
| `claw` | h36 / px24 / py8 / gap8 | 卡片底部「详细配置」「重试」等带 icon+文字的次级按钮 |
| `claw-sm` | h32 / px16 / py4 / gap6 | 紧凑场景：Dialog footer / 行内主按钮（与历史 `size="sm"` 视觉对齐） |
| `claw-square` | 48×36 / p0 | 仅图标的方形次级按钮（卡片角落「刷新」） |
| `claw-lg` | h40 / px18 / py1 / gap16 | 主操作按钮「创建 Agent」 |

shadcn 自带的 `default / sm / lg / icon / icon-sm / icon-lg` 不变。

---

## 五、调用示例

### 1. 卡片底部：详细配置（次级 + 文字 icon）

```tsx
<Button variant="claw-outline" size="claw">
  <Settings className="w-3.5 h-3.5" />
  详细配置
</Button>
```

### 2. 卡片角落：刷新（次级 + 仅图标）

```tsx
<Button
  variant="claw-outline"
  size="claw-square"
  onClick={handleRefresh}
  aria-label="刷新状态"
>
  <RefreshCw className="w-3.5 h-3.5" style={{ color: "#737373" }} />
</Button>
```

### 3. 页面顶部：创建 Agent（主操作）

```tsx
<Button variant="claw-primary" size="claw-lg">
  <Plus />
  创建 Agent
</Button>
```

### 4. 配合 `<Link>` 跳转

shadcn `Button` 默认渲染为 `<button>`。若需要跳转，外层包 `<Link>` 即可：

```tsx
<Link href={`/openclaw/${id}`}>
  <Button variant="claw-outline" size="claw">
    <Settings className="w-3.5 h-3.5" />
    详细配置
  </Button>
</Link>
```

---

## 六、禁止 / 正确 对照

| 场景 | ❌ 旧写法 | ✅ 新写法 |
|---|---|---|
| 详细配置按钮 | `<Button variant="outline" className="h-9 px-6 rounded-[4px]" style={{borderColor:"#E5E5E5"...}}>` | `<Button variant="claw-outline" size="claw">` |
| 仅图标方按钮 | `<Button variant="outline" className="h-9 w-12 p-0 rounded-[4px]" style={{borderColor:"#E5E5E5"}}>` | `<Button variant="claw-outline" size="claw-square">` |
| 创建 Agent 主按钮 | 自己写一个 `<button className="bg-gradient-to-r from-black ...">` | `<Button variant="claw-primary" size="claw-lg">` |
| icon 与文字间距 | `<Icon className="mr-2" />` | 不需要 `mr-2`，size 已自动 `gap-2` 或 `gap-4` |

---

## 七、扩展规则

后续 Figma 如果新增按钮变体（例如「危险操作 / 文字按钮 / loading 态」），**先扩展 `Button` 的 variant**，再让业务方调用，**不要直接在业务页面手写一组样式覆盖**。修改流程：

1. 在 Figma 拿到新变体的 token；
2. 编辑 `client/src/components/ui/button.tsx` 增加新的 variant key；
3. 在本文档「Token 表」「变体」「调用示例」三个表里同步追加；
4. 找 1-2 个业务调用点先迁移过去验证；
5. 老调用点逐步收敛。

---

## 八、当前已迁移的页面

### 8.1 用户端（`pages/tenant/**` + `components/**`）— 已全量收口

下列文件中所有"业务按钮"（主按钮 + 次级按钮）均已统一到 `<Button variant="claw-primary | claw-outline | destructive">`，业务层零残留 inline 主渐变 / 零残留 shadcn `outline`：

**`components/`**
- ✅ `agent/AgentCard.tsx` —— 卡片底部 3 个按钮（详细配置 / 重试 / 刷新）。
- ✅ `ScopePopover.tsx` —— 确认（claw-primary）+ 取消（claw-outline）。
- ✅ `EnableMemoryDialog.tsx` —— 取消（claw-outline）+ 确认启用（claw-primary，去掉自创紫色渐变）。
- ✅ `DisableMemoryDialog.tsx` —— 取消（claw-outline）+ 确认禁用（destructive）。
- 🟡 `OpenClawCombobox.tsx` —— Combobox trigger 保留 shadcn `outline`，已加 `// allow-shadcn-outline` 白名单注释。

**`pages/tenant/`**
- ✅ `FileSpace.tsx` —— 7 处迁移（4 主按钮 + 2 工具栏次级 + 1 删除按钮 → destructive）。
- ✅ `SkillSquare.tsx` —— 3 处（下发主按钮 + 返回列表 + 下载次级）。
- ✅ `ResetPassword.tsx` —— 1 处（取消次级）。
- ✅ `ModelQuota.tsx` —— 1 处（刷新次级）。
- ✅ `ChatView.tsx` —— 6 个 Button（4 个 + 2 个次级取消/重试恢复）+ 2 个 AlertDialogAction（通过 `buttonVariants({ variant: "claw-primary", size: "claw-sm" })` 复用样式）+ 圆形发送/进度条白名单注释。
- ✅ `ToolsMcpPanel.tsx` —— 4 处次级（取消×2 / 取消修改 / 暂不重启）。
- ✅ `MyOpenClaw.tsx` —— 14 处迁移（5 主按钮：创建 Agent×2 / 立即访问 / 下一步 / 创建×2 + 8 outline 取消/上一步/空状态 + Logo 容器白名单）。橙色重启/重装/移除角色按钮**保留 className**（属于"警示但非破坏"语义，未来若新增统一沿用 `bg-orange-500 text-white`）。
- ✅ `OpenClawDetail.tsx` —— 18 处迁移（7 主按钮 + 11 次级，含 3 处 `w-full` 全宽表单按钮）+ 5 处自定义 Checkbox / 圆形按钮白名单注释。

### 8.2 待迁移

- `pages/admin/**`（管理端）：按用户范围限定，未在本批次内。后续触达时按 SKILL.md §8.1 铁律执行。
- 已查：用户端 `pages/tenant/` 与 `components/` 中 `variant="outline"`、`linear-gradient(90deg, #020617 70%, #1447E6 100%)` 残留命中均为 0（除 `OpenClawCombobox` 1 处合法白名单）。

### 8.3 合法保留场景（必须带白名单注释）

#### inline 主渐变 `// allow-inline-gradient`

| 场景 | 文件 | 说明 |
|---|---|---|
| 进度条填充色 | `ChatView.tsx` 1916 | 非按钮，水平进度条 |
| 圆形发送按钮（27×27 rounded-full） | `ChatView.tsx` 2398 / `OpenClawDetail.tsx` 5135 | 非标准矩形按钮，沿用 ChatGPT-like 圆形发送 |
| 圆形停止按钮（27×27 rounded-full） | `OpenClawDetail.tsx` 5119 | 同上 |
| 自定义 Checkbox 选中态色块（16×16） | `OpenClawDetail.tsx` `DiagOptionCard` / `DiagOptionRow` / rollback 行内 | 非按钮，表单控件 |
| Logo 图标容器（7×7） | `MyOpenClaw.tsx` 974 | 非按钮，🦞 logo |

#### shadcn outline `// allow-shadcn-outline`

| 场景 | 文件 | 说明 |
|---|---|---|
| Combobox / Popover trigger | `OpenClawCombobox.tsx` 68 | shadcn 内置交互模式，换 claw-outline 会破坏选择器外观 |

#### 警示色按钮（橙色，非破坏性但需提醒）

| 场景 | 文件 | 说明 |
|---|---|---|
| 重启 / 重装 / 移除角色 | `MyOpenClaw.tsx` 798/828/902 | 用 `className="bg-orange-500 hover:bg-orange-600 text-white"` 单色样式（非渐变，不算违规）。**未来新增此类按钮统一沿用此规范**，不要换成红色 `destructive`（与"删除"语义冲突）。 |

> 自检脚本（计划接入）将通过扫描 `linear-gradient(90deg, #020617 70%, #1447E6 100%)` + `variant="outline"` 命中点，只对**没有相邻 `// allow-*` 注释**的命中报错。
