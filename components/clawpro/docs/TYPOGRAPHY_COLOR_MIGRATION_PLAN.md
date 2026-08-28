# ClawPro 文字色体系迁移计划

> 状态：评审版，待团队确认后实施  
> 适用范围：`client/src/**` 前端页面与组件  
> 目标读者：设计、前端同事，以及协作 AI  
> 本文档只描述“文字 + 文字色体系”的迁移，不包含背景色、边框色、状态色、图表色的系统性重构。

---

## 1. 背景与目标

当前项目中存在多种文字颜色来源：

- `Typography` 语义组件内的 `text-gray-*`
- 页面和组件中散落的 `text-[#...]`
- Tailwind 灰色类，如 `text-gray-500`
- 少量蓝灰色写法，如 `#334155`、`#020617`

这会导致三个问题：

1. 全站文字层级不容易统一调整；
2. 页面之间的标题、正文、描述、辅助文字颜色不够体系化；
3. 后续如果要从中灰色切换到蓝灰色，需要逐文件判断，维护成本高。

本次迁移目标是：

> 将全站文字颜色从零散的 `gray / hex` 写法，逐步收口到统一的 `--text-*` 语义 token，并以 `Typography` 组件作为主要文字入口。

---

## 2. 已确认的文字色映射

本次建议基于“蓝灰色 Slate”建立文字语义色。

| 文字语义 | CSS Token | 色值 | Slate 对应 | 用途 |
|---|---:|---:|---|---|
| 强强调 | `--text-emphasis` | `#020617` | `slate-950` | 强强调、关键数字、强标题 |
| 标题 | `--text-title` | `#0F172A` | `slate-900` | 页面标题、模块标题 |
| 正文 | `--text-body` | `#1E293B` | `slate-800` | 普通正文 |
| 次级正文 | `--text-secondary` | `#334155` | `slate-700` | 描述、补充说明、表格次要字段 |
| 辅助文字 | `--text-muted` | `#64748B` | `slate-500` | 时间、备注、辅助信息 |
| 极弱文字 | `--text-weak` | `#94A3B8` | `slate-400` | 占位、空状态、极弱提示 |
| 品牌文字 | `--text-brand` | `#1447E6` | - | 链接、活跃态、品牌强调 |
| 危险文字 | `--text-danger` | `#DC2626` | - | 删除、错误、危险操作 |

### 2.1 为什么不是简单同阶替换

不建议直接把 `gray-900` 替换为 `slate-900`、`gray-700` 替换为 `slate-700` 后结束，而是先建立语义层：

- 标题、正文、描述、辅助文字不应该只按色阶编号理解；
- 后续如果正文太重，只需要调整 `--text-body`，不需要搜索全站；
- 页面开发时应选择“语义”，而不是记忆某个色号。

---

## 3. 第一阶段实施范围

第一阶段只做两个代码入口改造。

### 3.1 新增文字色 token

在 `client/src/index.css` 的 `:root` 中新增：

```css
--text-emphasis: #020617;
--text-title: #0F172A;
--text-body: #1E293B;
--text-secondary: #334155;
--text-muted: #64748B;
--text-weak: #94A3B8;
--text-brand: #1447E6;
--text-danger: #DC2626;
```

### 3.2 Typography 组件换入口

修改 `client/src/components/ui/Typography.tsx`：

| Typography token | 当前 | 调整为 |
|---|---|---|
| `primary` | `text-gray-900` | `text-[var(--text-title)]` |
| `emphasis` | `text-gray-950` | `text-[var(--text-emphasis)]` |
| `body` | `text-gray-900` | `text-[var(--text-body)]` |
| `secondary` | `text-gray-700` | `text-[var(--text-secondary)]` |
| `muted` | `text-gray-500` | `text-[var(--text-muted)]` |
| `weak` | `text-gray-400` | `text-[var(--text-weak)]` |
| `brand` | `text-[var(--brand-blue)]` | `text-[var(--text-brand)]` |
| `danger` | `text-red-600` | `text-[var(--text-danger)]` |
| `inherit` | `text-inherit` | `text-inherit` |

建议目标代码：

```ts
export const typographyColorTokens = {
  primary: "text-[var(--text-title)]",
  emphasis: "text-[var(--text-emphasis)]",
  body: "text-[var(--text-body)]",
  secondary: "text-[var(--text-secondary)]",
  muted: "text-[var(--text-muted)]",
  weak: "text-[var(--text-weak)]",
  brand: "text-[var(--text-brand)]",
  danger: "text-[var(--text-danger)]",
  inherit: "text-inherit",
} as const;
```

### 3.3 展示台同步

在 `全局组件展示台` 中同步更新：

1. `Color` 菜单：新增 `Text 文字语义色` 组织，展示 `--text-*` 8 个 token，并标注 `Typography 使用中`；
2. `Typography` 菜单：`Tone 色阶示例` 应展示新的 `--text-*` 实际效果。

---

## 4. 第一阶段不会做什么

第一阶段不迁移页面和组件里散落的颜色写法，例如：

- `text-[#737373]`
- `text-[#334155]`
- `text-[#020617]`
- `text-gray-*`
- `text-slate-*`

第一阶段也不迁移：

- 背景色；
- 边框色；
- 图标色；
- 状态色；
- 图表色；
- 阴影；
- 圆角；
- 间距；
- 布局。

原因：先确认文字语义色和 `Typography` 入口效果，再推进全站迁移，风险最低。

---

## 5. 第一阶段会直接影响哪些内容

所有使用以下 `Typography` 组件，且没有通过 `className` 覆盖颜色的地方，会自动跟随变化：

- `TenantHeroTitle`
- `TenantPageTitle`
- `TenantDocTitle`
- `SectionTitle`
- `PanelTitle`
- `CardTitle`
- `BodyText`
- `BodyMedium`
- `CompactText`
- `MiniBodyText`
- `MetaText`
- `MetaMedium`
- `SmallBodyText`
- `TinyText`
- `StatNumber`
- `InlineNumber`
- `CodeText`
- `StepText`
- `UrlText`

### 5.1 不会自动变化的情况

以下情况不会被第一阶段自动影响：

```tsx
<p className="text-[#737373]">说明文字</p>
<span className="text-gray-500">辅助信息</span>
<BodyText className="text-[#020617]">被 className 覆盖颜色</BodyText>
```

只有“使用了 `Typography` 组件，并且没有额外覆盖文字颜色”的地方，会自动使用新 token。

---

## 6. 后续迁移原则

后续迁移分为“组件迁移”和“页面迁移”。

### 6.1 优先使用 Typography 组件

推荐：

```tsx
<TenantHeroTitle>模型额度与用量总览</TenantHeroTitle>
<BodyText>这里展示正文内容。</BodyText>
<BodyText tone="secondary">这里展示描述说明。</BodyText>
<MetaText>更新于 2026-06-02</MetaText>
```

不推荐：

```tsx
<h1 className="text-[#0A0A0A]">标题</h1>
<p className="text-gray-500">说明</p>
<span className="text-[#737373]">更新时间</span>
```

### 6.2 组件内部无法使用 Typography 时，使用 CSS token

有些底层组件不适合直接引入 `Typography`，可以用 `var(--text-*)`：

```tsx
<span className="text-[var(--text-secondary)]">说明文字</span>
```

### 6.3 不要把非文字颜色迁移到 text token

以下颜色不属于 `--text-*` 范畴：

- 背景：应使用 `--surface-*` 或现有背景 token；
- 边框：应使用 `--border-*` 或现有 border token；
- 状态：应使用状态色 token；
- 图标：需要按具体语义判断，不默认跟随文字；
- 图表：继续使用 chart token；
- 品牌图形：继续使用 brand token。

---

## 7. 团队分工建议

### A. 全局 UI 组件迁移

范围：

```txt
client/src/components/ui/**
```

优先组件：

1. `admin-page-header.tsx`
2. `button.tsx`
3. `table.tsx`
4. `dialog.tsx`
5. `alert-dialog.tsx`
6. `alert.tsx`
7. `input.tsx`
8. `select.tsx`
9. `pagination.tsx`
10. `empty.tsx`
11. `status-tag.tsx`
12. `admin-sidebar.tsx`

迁移要求：

- 只迁移文字颜色；
- 不改布局、间距、圆角、阴影；
- 能用 `Typography` 就用 `Typography`；
- 底层组件不适合引入 `Typography` 时，使用 `text-[var(--text-*)]`；
- 背景、边框、图标、状态色暂不迁移。

### B. 管控端页面迁移

范围：

```txt
client/src/pages/admin/**
```

优先页面：

1. `PlatformPolicy`
2. `BasicInfo`
3. `ModelConfig`
4. `ChannelConfig`
5. `MemberManagement`
6. `OpenClawMonitor`
7. `TokensMonitor`
8. `SessionManagement`

迁移建议：

- 页面标题统一使用 `AdminPageHeader`；
- 模块标题使用 `SectionTitle` / `PanelTitle`；
- 卡片标题使用 `CardTitle`；
- 正文使用 `BodyText`；
- 描述说明使用 `BodyText tone="secondary"`；
- 时间、备注、弱信息使用 `MetaText`；
- 路径、ID、代码类文本使用 `CodeText`；
- 数字使用 `StatNumber` / `InlineNumber`。

### C. 用户端页面迁移

范围：

```txt
client/src/pages/tenant/**
client/src/components/tenant/**
client/src/components/agent/**
```

优先页面：

1. `MyOpenClaw`
2. `OpenClawDetailGuide`
3. `ModelQuota`
4. `SkillSquare`
5. `HelpDocs`

迁移建议：

- Hero 标题使用 `TenantHeroTitle`；
- 页面标题使用 `TenantPageTitle`；
- 文章标题使用 `TenantDocTitle`；
- 模块标题使用 `SectionTitle` / `PanelTitle`；
- 卡片标题使用 `CardTitle`；
- 正文使用 `BodyText`；
- 描述说明使用 `BodyText tone="secondary"`；
- 时间、状态说明、弱信息使用 `MetaText`；
- 不改用户端骨架、不改卡片圆角、不改按钮样式。

### D. 巡检与收口

任务：搜索残留文字色，判断是否应迁移。

重点搜索：

```txt
text-gray-
text-slate-
text-[#0A0A0A]
text-[#020617]
text-[#334155]
text-[#737373]
text-[#A3A3A3]
text-[#404040]
text-[#171717]
#0A0A0A
#020617
#334155
#737373
#A3A3A3
#404040
#171717
```

判断规则：

- 如果是文字颜色：迁移为 `Typography` 或 `var(--text-*)`；
- 如果是边框、背景、图标、状态色：不迁移，并记录原因；
- 如果是历史特殊设计稿要求：添加说明，纳入豁免清单。

---

## 8. 按分工给同事或 AI 的可复制指令

> 使用方式：负责人可以直接复制对应任务包的指令给自己的 AI。每个任务包都包含范围、禁止事项、迁移规则和交付物，避免不同同事理解不一致。

### 8.1 任务包 A：全局 UI 组件迁移

适合负责人：负责 `components/ui` 基础组件的前端同事。

建议范围：

```txt
client/src/components/ui/**
```

优先文件：

```txt
admin-page-header.tsx
button.tsx
table.tsx
dialog.tsx
alert-dialog.tsx
alert.tsx
input.tsx
select.tsx
pagination.tsx
empty.tsx
status-tag.tsx
admin-sidebar.tsx
```

可复制指令：

```txt
请迁移 client/src/components/ui 下指定全局 UI 组件的文字色体系。

目标：
1. 只迁移文字颜色，不改布局、间距、圆角、阴影、交互逻辑；
2. 组件展示效果要继续符合全局组件展示台；
3. 迁移后组件内部文字色应使用 Typography 或 var(--text-*)。

迁移规则：
1. 标题、正文、描述、辅助信息优先使用 Typography 语义组件；
2. 底层组件不适合引入 Typography 时，使用 text-[var(--text-*)]；
3. text-[#020617] → text-[var(--text-emphasis)]；
4. text-[#0A0A0A] / text-[#171717] → 按语义改为 text-title 或 text-body；
5. text-[#334155] / text-[#404040] → text-[var(--text-secondary)]；
6. text-[#737373] → text-[var(--text-muted)]；
7. text-[#A3A3A3] → text-[var(--text-weak)]；
8. 品牌文字使用 text-[var(--text-brand)]；
9. 危险文字使用 text-[var(--text-danger)]。

禁止事项：
1. 不迁移背景色；
2. 不迁移边框色；
3. 不迁移图标色，除非图标明确跟随文字 currentColor；
4. 不迁移状态色，如运行中、失败、警告；
5. 不顺手重构组件结构。

交付物：
1. 列出修改了哪些文件；
2. 列出仍保留的颜色豁免及原因；
3. 检查 lint；
4. 在全局组件展示台确认该组件视觉正常。
```

---

### 8.2 任务包 B：管控端页面迁移

适合负责人：负责 Admin 管控端页面的前端同事。

建议范围：

```txt
client/src/pages/admin/**
```

优先页面：

```txt
PlatformPolicy
BasicInfo
ModelConfig
ChannelConfig
MemberManagement
OpenClawMonitor
TokensMonitor
SessionManagement
```

可复制指令：

```txt
请迁移当前管控端页面的文字体系。

目标：
1. 让页面标题、模块标题、正文、描述、辅助信息进入统一 Typography 体系；
2. 只迁移文字和文字颜色；
3. 不改变业务逻辑、布局结构、间距、背景、边框、图标和状态色。

迁移规则：
1. 页面标题统一使用 AdminPageHeader；
2. 模块标题使用 SectionTitle / PanelTitle；
3. 卡片标题使用 CardTitle；
4. 普通正文使用 BodyText；
5. 描述、补充说明使用 BodyText tone="secondary"；
6. 时间、备注、弱提示、表格辅助字段使用 MetaText；
7. 路径、ID、代码、命令类文本使用 CodeText；
8. 统计数字使用 StatNumber / InlineNumber；
9. 如果不适合换成 Typography，文字色使用 text-[var(--text-*)]。

禁止事项：
1. 不迁移表格边框、卡片边框、背景色；
2. 不改按钮 variant 和 size；
3. 不改页面布局和数据结构；
4. 不把状态色迁移为 --text-*。

交付物：
1. 列出迁移页面；
2. 列出页面中仍保留的文字 hex；
3. 说明每个保留项是文字、图标、边框、背景还是状态色；
4. 检查 lint；
5. 给出需要设计复核的截图页面路径。
```

---

### 8.3 任务包 C：用户端页面迁移

适合负责人：负责 Tenant 用户端页面和 Agent 卡片相关组件的前端同事。

建议范围：

```txt
client/src/pages/tenant/**
client/src/components/tenant/**
client/src/components/agent/**
```

优先页面：

```txt
MyOpenClaw
OpenClawDetailGuide
ModelQuota
SkillSquare
HelpDocs
```

可复制指令：

```txt
请迁移当前用户端页面的文字体系。

目标：
1. 用户端标题、卡片标题、正文、说明、时间等文字进入 Typography 体系；
2. 保持用户端页面骨架、卡片圆角、按钮样式和响应式策略不变；
3. 只迁移文字和文字颜色。

迁移规则：
1. Hero 标题使用 TenantHeroTitle；
2. 页面标题使用 TenantPageTitle；
3. 文档或文章标题使用 TenantDocTitle；
4. 模块标题使用 SectionTitle / PanelTitle；
5. 卡片标题使用 CardTitle；
6. 正文说明使用 BodyText；
7. 描述、补充说明使用 BodyText tone="secondary"；
8. 时间、状态说明、弱信息使用 MetaText；
9. 路径、ID、代码类文本使用 CodeText；
10. 数字使用 StatNumber / InlineNumber。

禁止事项：
1. 不改用户端 1200 / 1920 响应式骨架；
2. 不改 TenantCard 圆角；
3. 不改 TopNav 结构；
4. 不改按钮样式；
5. 不迁移背景、边框、图标和状态色。

交付物：
1. 列出迁移页面和组件；
2. 标记仍保留的颜色豁免；
3. 检查 lint；
4. 给出需要视觉复核的页面路径。
```

---

### 8.4 任务包 D：全站巡检与收口

适合负责人：负责 Code Review、质量巡检或专项收口的同事。

建议范围：

```txt
client/src/**
```

重点搜索：

```txt
text-gray-
text-slate-
text-[#0A0A0A]
text-[#020617]
text-[#334155]
text-[#737373]
text-[#A3A3A3]
text-[#404040]
text-[#171717]
#0A0A0A
#020617
#334155
#737373
#A3A3A3
#404040
#171717
```

可复制指令：

```txt
请巡检全站文字色残留并输出收口建议。

目标：
1. 找出仍然散落的文字颜色；
2. 判断哪些应迁移为 Typography 或 var(--text-*)；
3. 区分文字色和非文字色，避免误迁移。

判断规则：
1. 如果颜色用于标题、正文、描述、时间、备注、路径、ID，建议迁移；
2. 如果颜色用于边框、背景、图标、状态、图表、Logo，不迁移；
3. 如果颜色来自特殊设计稿，记录为豁免；
4. 如果组件已经通过 Typography 间接使用 token，不重复处理。

交付物：
1. 按文件输出残留颜色清单；
2. 给出建议迁移方式；
3. 给出不迁移原因；
4. 汇总需要设计确认的问题。
```

---

### 8.5 任务包 E：设计复核与验收

适合负责人：设计同事、页面负责人或最终验收人。

建议复核页面：

```txt
/design-system/components
/admin/platform-policy
/admin/basic-info
/admin/model-config
/admin/members
/my-openclaw
/openclaw/1
/model-quota
/skill-square
```

可复制指令：

```txt
请复核文字色体系迁移后的页面视觉。

重点检查：
1. 标题、正文、描述、辅助信息层级是否清晰；
2. 正文是否过黑或过浅；
3. 描述文字是否与正文有足够层级差；
4. 表格正文、表头、辅助字段是否可读；
5. 用户端和管控端是否保持各自风格；
6. 品牌色、危险色、状态色是否未被误改；
7. 是否存在局部文字颜色突兀、不一致。

输出结果：
1. 通过 / 不通过；
2. 需要调整的 token；
3. 需要回退或豁免的页面；
4. 截图或页面路径。
```

---

## 9. 验收标准

第一阶段验收：

- `client/src/index.css` 已新增 `--text-*` token；
- `client/src/components/ui/Typography.tsx` 不再直接依赖 `text-gray-*`；
- `全局组件展示台 > Color` 可看到 `Text 文字语义色`；
- `全局组件展示台 > Typography` 能看到新文字色效果；
- 已使用 `Typography` 的页面视觉层级仍然清晰。

后续迁移验收：

- 已迁移文件不再出现核心文字 hex；
- 已迁移文件中的标题、正文、描述、辅助文字有明确语义；
- 非文字颜色未被误迁移到 `--text-*`；
- 页面布局、间距、圆角、阴影未被无关修改；
- 迁移后展示台和核心页面效果正常。

---

## 10. 豁免规则

以下情况可以暂不迁移，但需要记录原因：

- 图标颜色与文字颜色不同步时；
- 状态色，如成功、失败、警告、运行中；
- 边框和分割线；
- 背景色、卡片底色、Hover 底色；
- 图表颜色；
- 品牌图形、Logo、装饰色；
- 设计稿明确指定且暂不纳入文字体系的特殊颜色。

---

## 11. 推荐执行节奏

1. 第一阶段：建立 `--text-*` token，Typography 换入口，展示台验证；
2. 第二阶段：迁移全局 UI 组件；
3. 第三阶段：迁移管控端核心页面；
4. 第四阶段：迁移用户端核心页面；
5. 第五阶段：全站巡检，输出豁免清单；
6. 第六阶段：把文字色规则补入设计规范和 Code Review Checklist。

---

## 12. 当前结论

本计划建议先用最小闭环验证文字色方向：

- 先改 token；
- 再改 `Typography` 入口；
- 通过展示台和已使用 `Typography` 的页面确认效果；
- 确认后再让团队分工迁移组件和页面。

这样既能快速看到视觉变化，又能避免一次性全站大范围改动带来的风险。
