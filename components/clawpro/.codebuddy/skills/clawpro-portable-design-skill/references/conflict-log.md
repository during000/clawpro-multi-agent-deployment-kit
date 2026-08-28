# Conflict Log

> 当前冲突与已确认裁决记录。遇到规范冲突时先查本文件；新增冲突时补充条目，不要在页面实现中隐式裁决。

## 状态定义

| 状态 | 含义 |
|---|---|
| `resolved` | 已有明确裁决，可回写到规范 |
| `needs-design-confirmation` | 需要设计 / owner 确认 |
| `accepted-legacy` | 暂时接受存量，触达即同步 |

> 本文件是设计冲突与已确认裁决的唯一活账本。新增冲突 / 新裁决直接补在本文件，不再回查历史 audit 快照；标注“暂选 / 后续进一步确认”的条目写入规范时保留当前口径，不包装成不可变更最终裁决。

## C-001：项目根 `SKILL.md` 与当前运行时实现不一致

- 状态：`resolved`
- 现象：项目根 `SKILL.md` 仍包含旧色值、旧圆角和旧阴影描述；当前运行时实现更接近 `client/src/index.css`、`client/src/components/ui/*` 与现有设计刷新资料。
- 裁决：周一先交付 `clawpro-portable-design-skill`；周一后把根目录 `SKILL.md` 收敛成短路由入口，指向新版 references。
- 执行：本 portable design pack 以 `references/foundation.md` 和当前组件实现为基线；旧入口不在本轮强改。

## C-002：用户端业务卡片圆角 12px vs 全局 v2 最大 4px

- 状态：`resolved`
- 现象：全局 v2 规定最大圆角 4px；用户端业务卡片已有 12px 三态系统。
- 裁决：Tenant 业务对象卡、Agent 卡、技能卡等展示型业务卡使用 12px `TenantCard`；表格容器、弹窗、Popover、数据密集容器按对应组件 spec 分流。
- 执行：Admin / Shared 继续 4px `SurfaceCard`；Tenant 不把所有容器都套成 12px。

## C-003：用户端背景与点阵装饰规则冲突

- 状态：`resolved`
- 现象：旧规范含点阵和贯穿线；当前客户端确认白底 + 极淡蓝雾。
- 裁决：Tenant 页面背景采用白底 + 极淡蓝雾；点阵、贯穿线、装饰线废弃，不新增。
- 执行：蓝雾具体参数后续可补 token 表；周一交付先锁方向。

## C-004：全局组件 owner 与修改权限

- 状态：`resolved`
- 现象：`SKILL-GLOBAL-COMPONENTS.md` 声明 addietang / miekoyychen 维护基础组件。
- 裁决：业务页面不得随意修改 `client/src/components/ui/**` 核心样式；确需新增 variant 时先在规范中声明，再改组件。

## C-005：图标资产源目录重复

- 状态：`accepted-legacy`
- 现象：`icon/` 与 `client/public/icon/` 存在部分重复资产，且命名中英混合。
- 风险：引用路径不统一、重复维护。
- 当前建议：新增使用先查 `assets/icon-registry.example.json` 或宿主仓 registry；后续逐步统一来源，不做一次性全量迁移。

## C-006：硬编码色治理边界

- 状态：`resolved`
- 现象：历史页面和部分 fallback 示例存在业务色 hex / Tailwind 色阶散写。
- 裁决：只有 token 定义文件、基础 token 表和资产源文件允许出现必要 hex；业务规范、组件 spec、portable fallback、page recipe 优先引用 token / CSS variable。
- 执行：历史页面 hex 是存量债务，不是新增例外；新增禁止扩大，触达页面顺手迁移。

## C-007：表格字号密度

- 状态：`resolved`
- 现象：旧文档中表头 12px、单元格 14px 与当前确认口径不一致。
- 裁决：管理端表格整体使用 12px，包括单元格正文。
- 执行：`components.md` 与 `component-specs/table.md` 按 12px 口径回写。

## C-008：暂选项记录

- 状态：`needs-design-confirmation`
- 现象：`C-016` Dialog / Drawer / Popover 仍为暂选；`D-03` 已由同事确认删除 Tenant Text Switch。
- 当前口径：周一可先按确认表执行；后续如果设计 owner 更新 `C-016` 结论，再回写 `component-specs/dialog-drawer.md`。
- 待确认：浮层圆角 / 宽度细节。

## C-009：C-019 表单页复杂结构暂选

- 状态：`needs-design-confirmation`
- 现象：`design-confirmation-decisions-2026-06-06.md` 中 `C-019` 选择 C，并标注“暂定，后续进一步确认；周一前只确认基础结构”。
- 当前口径：周一前表单页只定义基础结构，不把左右双栏、步骤卡、Sticky footer 写成强制标准。
- 执行：`references/page-recipes.md` 保留基础结构和后续确认说明；具体复杂页面如需升级结构，先回到设计侧确认。
- 待确认：复杂表单页的左右双栏、步骤卡、Sticky footer 是否进入下一批强标准。

## C-010：Tenant Text Switch 删除

- 状态：`resolved`
- 现象：确认表曾暂选拆分 Tenant Capsule Tabs 与 Tenant Text Switch；同事反馈当前没有引用 Text Switch 样式。
- 裁决：Tenant Text Switch 不进入周一交付包，不再作为独立组件规范或迁移目标。
- 执行：`references/tenant.md` 与 `component-specs/segment.md` 只保留 Tenant 胶囊 Segment 与 Admin 方角 Segment 分流；page header 下方一级 Tab 全部走 `component-specs/tabs.md`（LineTabs）；低密度局部切换复用已有基础控件或胶囊体系。

## C-011：StatNumber 数字字体 DIN vs 全局 PingFang SC `!important`

- 状态：`resolved`
- 现象：spec（`component-specs/number-card.md` 4 处口径、`tokens/typography.md`）规定数字使用 DIN Alternate；但 `client/src/index.css` 早期为压制业务侧 inline `fontFamily` 与第三方组件污染，加了 `*:not(svg):not(svg *) { font-family: PingFang SC ... !important }` 全局规则，导致 `font-din` / `font-din-stat` 都被覆盖回 PingFang SC。同时 `font-din-stat` 这个 utility 在 `@theme` 里没有对应 token，class 实际为空。
- 裁决：恢复 spec 口径，让 StatNumber 真实渲染 DIN。
- 执行：
  1. `client/src/index.css` `@theme` 块新增 `--font-din-stat: "DIN Alternate", "DIN", "Helvetica Neue", sans-serif`，让 Tailwind v4 自动产出 `.font-din-stat` utility。
  2. 全局 `!important` 规则改为白名单豁免 —— `*:not(svg):not(svg *):not(.font-din):not(.font-din-stat):not(.font-mono):not(.font-en)`，仅这 4 个显式字体 utility 能透过去；裸元素仍走 PingFang SC，行为与原状一致，不会触发非预期字体回写。
- Portable Fallback：宿主仓如沿用 PingFang SC 全局 `!important` 模式，要么按本仓做白名单豁免，要么至少保证 StatNumber 容器加 `font-variant-numeric: tabular-nums` 以维持数字对齐。

## C-012：页面级请求被误降级成组件级预览

- 状态：`resolved`
- 现象：当用户说“生成一个管控端页面 / 列表页”时，容易直接套 `page-recipes.md` 的主内容骨架，漏掉 `AdminSidebar` / `AdminLayout`，产出只有中间内容区的局部预览。
- 风险：页面级预览和真实 Admin 到达页结构不一致，用户会误以为 skill 本身没有把页面骨架定义清楚。
- 裁决：对 Admin 场景，默认把“页面”理解为完整页面壳；除非用户明确说“只做组件级 demo / 只看局部区域 / 只验证表格容器”，否则必须带 `AdminSidebar + AdminSidebarInset`。
- 执行：已回写到 `SKILL.md` workflow、`references/page-recipes.md`、`references/admin.md` 和 `qa/admin-checklist.md`，后续按该口径执行。

## C-013：navigation-sidebar.md 拆解 + 聚合 spec 反向索引补齐

- 状态：`resolved`
- 背景：盘点展示台（`client/src/pages/DesignSystemComponents.tsx`，81 个组件 id）↔ `component-specs/`（36 份 spec）的对照矩阵时，发现两类结构问题。
- 问题 1：`navigation-sidebar.md` 内 §3.1 Admin Sidebar 与 `admin-sidebar.md` 重复、§3.3 Breadcrumb 与 `breadcrumb.md` 重复，唯一独占内容是 §3.2 Tenant TopNav。`admin-sidebar.md` 是后期补的高密度版本（含 §14 ✅/❌ 代码对照），权威性更强，保留它会让 `navigation-sidebar.md` 形成内容真冗余。
- 问题 2：`card-surface.md` / `selection-controls.md` / `popover-dropdown-menu.md` / `dialog-drawer.md` / `input-select.md` 5 份聚合 spec 单 spec 覆盖展示台 4~6 个 id，但 spec 顶部没有反向索引，读者从展示台 id 找不到对应 spec 锚点。
- 裁决：
  1. 拆 `navigation-sidebar.md` —— 抽 §3.2 / §12.4 出来独立成 `tenant-topnav.md`，删除 `navigation-sidebar.md`。
  2. 5 份聚合 spec 顶部统一加一行 `> **Showcase mapping**: ...` metadata，列出对应展示台 id。
- 执行：
  1. 新增 `component-specs/tenant-topnav.md`（包含 §1 Purpose ~ §12 ✅/❌ 代码对照）。
  2. 删除 `component-specs/navigation-sidebar.md`。
  3. 同步更新 14 处引用：`MANIFEST.json` / `INDEX.md` / `SKILL.md` / `STATUS.md` / `STRUCTURE.md` / `DEVELOPER-USAGE.md` / `references/migration-map.md`、以及 `component-specs/admin-sidebar.md` / `breadcrumb.md` / `tabs.md` / `segment.md` / `tree.md` 内部交叉引用。
  4. `card-surface.md` / `selection-controls.md` / `popover-dropdown-menu.md` / `dialog-drawer.md` / `input-select.md` 5 份顶部加 Showcase mapping。
  5. `node scripts/verify-portable-skill.mjs` 通过，无 missing / unlisted。
- 未处理（留待后续 PR）：
  - B 类：`file-browser.md` ↔ `upload-file-browser.md` 文件名相似导致语义混淆（资产只读 vs 上传写入），改名 `asset-browser.md` / `upload.md` 涉及面更广（HTML/CSS demo 文件名 + verify 脚本），单独 PR 处理。
  - D 类：展示台 14 个 shadcn 原态 id（`kbd` / `accordion` / `slider` / `separator` / `scroll-area` / `collapsible` / `carousel` / `input-otp` / `aspect-ratio` / `navigation-menu` / `menubar` / `resizable` / `command` / `select-panel`）+ 3 个业务封装 id（`back-button` / `favorite-button` / `tree-select`）暂不强制写 spec，触达时按需补。

## C-014：把表格标题区和表格本体绑定在一个容器内

- 状态：`resolved`
- 现象：页面实现时容易把“标题、描述、右上角按钮”塞进表格卡头，导致表格组件边界被扩大。
- 风险：页面结构与标准后台列表页不一致，也会让 `Table` / `DataTable` 组件职责变得含混。
- 裁决：表格默认只展示表格本身；表格外若有标题、描述、统计说明、刷新/创建/更多按钮，一律单独放到表格外。
- 执行：已回写到 `references/page-recipes.md`、`references/admin.md`、`component-specs/table.md`、`component-specs/data-table.md` 与 `qa/admin-checklist.md`。

## C-015：HeaderAction (前往用户端 / 收起态图标按钮) hover 反馈反转 —— bg / border 不变

- 状态：`resolved`
- 背景：旧 spec（`component-specs/admin-sidebar.md` §3.7 / §11 Don't / §14.7）规定 HeaderAction hover 时 bg 走 `--admin-sidebar-action-hover-bg` (`rgba(180,191,225,0.14)`) + border 走 `--admin-sidebar-action-hover-border` (`#D6DDEA`)，理由是"防止 link 被浏览器渲染成失控的灰色"。设计标准页 (`https://clawprodesign.devcloud.woa.com/admin/skill-config`) 实际表现为：
  - 展开态 "前往用户端" 按钮 hover 时 bg / border 不变，**仅靠右侧 → arrow 滑入**（`width 0→14px` + `opacity 0→1` + `translate-x -6px→0`，`group-hover` 控制）提供反馈。
  - 收起态图标按钮 hover 时 bg / border 不变，**靠 Tooltip "前往用户端" / "展开导航"** 提供反馈。
- 风险：旧 hover bg / border 与 Sidebar 白底 + 1px 边的轻量调性冲突，且与设计标准实测不一致。
- 裁决：HeaderAction hover 时 bg / border 完全不动；展开态靠 → arrow 滑入，收起态靠 Tooltip。`--admin-sidebar-action-hover-bg` / `--admin-sidebar-action-hover-border` 两个 token 保留以兼容历史 css override，但**新代码不再引用**。
- 执行：
  1. 组件源 `client/src/components/ui/admin-sidebar.tsx` `AdminSidebarHeaderAction` className 移除 `hover:bg-[var(--admin-sidebar-action-hover-bg)] hover:border-[var(--admin-sidebar-action-hover-border)]`，transition 收窄为 `transition-[color,box-shadow]`。
  2. 装配 `client/src/components/AdminLayout.tsx` 收起态手写 `<a>` 移除 `hover:bg-[#f5f5f5] transition-colors`。
  3. 展示台 `client/src/pages/DesignSystemComponents.tsx` 展开态 line 3142 / 收起态 line 3101 移除外层 `hover:!bg-[#F5F5F5]` 强制覆写，统一回归组件 default。
  4. spec `admin-sidebar.md` 反转 §3.4 / §3.7 / §5 / §11 Don't / §14.7（旧 ✅ 转 ❌；新 ✅ 示例 = 组件 default + arrow 滑入 + Tooltip 三者组合）。
  5. portable HTML/CSS 样例 `admin-sidebar.html` / `admin-control-page.html` / `admin-list-page.html` 三处合并 hover bg 选择器拆出 `.sidebar-action-btn:hover, .sidebar-action-btn--collapsed:hover`，仅保留 `.footer-action:hover` 等其他类的 `background: #f5f5f5`。
  6. `node scripts/verify-portable-skill.mjs` 通过。

## C-016：展开态 Header 折叠按钮 Tooltip 反转 —— 必须挂「收起导航」

- 状态：`resolved`
- 背景：旧 spec（`component-specs/admin-sidebar.md` §3.4 / §6 / §11 Don't / §14.6）规定**展开态 Header 折叠按钮禁止挂 Tooltip**，理由是"`delayDuration={0}` 黑色浮层会越界遮挡主内容区告警条 / 面包屑"。设计标准页 (`https://clawprodesign.devcloud.woa.com/admin/skill-config`) 实测：展开态收起按钮 hover 时**有 Tooltip「收起导航」**，与收起态「展开导航」文案对称。
- 风险：仅靠 `aria-label` 屏幕阅读器之外的鼠标用户拿不到文本反馈，对于"这个图标点了之后是收起还是展开"语义模糊；且与设计标准实测不一致。
- 裁决：展开态折叠按钮 + 收起态折叠按钮**两态都必须挂 Tooltip**，文案对称（展开态 = "收起导航"，收起态 = "展开导航"），统一 `side="right" sideOffset={8} delayDuration={0}`。Tooltip 浮层朝右弹到 sidebar 外侧空白 / `AdminSidebarInset` 留白，不会越界遮挡内容。
- 执行：
  1. 装配 `client/src/components/AdminLayout.tsx:391-397` 展开态 `<button>` 包 `<Tooltip>` + `<TooltipContent side="right" sideOffset={8}>收起导航</TooltipContent>`。
  2. 展示台 `client/src/pages/DesignSystemComponents.tsx:3138-3140` 同步包 Tooltip。
  3. spec `admin-sidebar.md` 反转 §3.4 行 73（"禁止挂 Tooltip" → "必须挂 Tooltip"）、§6 行 164（措辞反转 + 解释浮层方向）、§11 Don't 行 491（"不要挂 Tooltip" → "不要省 Tooltip"）、§14.6 整段重写（旧 ✅ 转 ❌；新 ✅ = 展开态 + 收起态都包 Tooltip）。
  4. `node scripts/verify-portable-skill.mjs` 通过。

## C-017：HeaderAction hover bg 收尾 —— 删掉 index.css 残留覆写规则

- 状态：`resolved`
- 背景：C-015 已把 `AdminSidebarHeaderAction` 组件 className 中的 `hover:bg-* hover:border-*` 拆掉，但 `client/src/index.css` 在 `@layer components` 内仍残留两段 raw CSS 选择器把 hover bg/border 强行改成 `--admin-sidebar-action-hover-bg` / `--admin-sidebar-action-hover-border`（其中第一段还带 `!important`）。结果：装配层删掉了 className，CSS 兜底层却继续覆盖 → 实际页面 hover 时 "前往用户端" 按钮仍然出现轻微蓝灰底 (`rgba(180,191,225,0.14)`)，与设计标准 `https://clawprodesign.devcloud.woa.com/admin/skill-config` 不一致。
- 风险：CSS 兜底覆写绕过组件 className 路径，导致 spec / 组件 / 装配三层的 hover 行为对外仍呈现"半反转"状态。
- 裁决：彻底删除 `index.css` 内两段 `[data-slot="admin-sidebar-header-action"]:hover` 规则（行 435-438 + 行 502-505）。HeaderAction hover 必须直接继承 base 样式（`bg = #fff`、`border = --admin-sidebar-action-border` = `#EAEEF4`），与 C-015 保持一致。`--admin-sidebar-action-hover-bg` / `--admin-sidebar-action-hover-border` 两个 token 仍保留以兼容外部消费方，但 ClawPro 内部三层 (spec / 组件 / 装配 / CSS) 全线断开引用。
- 执行：
  1. `client/src/index.css` 行 435-438 / 502-505 两段 hover 规则整段删除，原位补上注释提醒后续不要再加。
  2. spec `component-specs/admin-sidebar.md` §6 行 105 已说明 token 不再被引用，本轮无需再改文案，但本日志作为依据。
  3. `node scripts/verify-portable-skill.mjs` 通过。

## C-018：知识库连接器 logo 外框圆角 8px vs Admin 控件/面板 4px 铁律

- 状态：`needs-design-confirmation`
- 现象：`KnowledgeDataSources.tsx`「接入企业知识库」弹窗内，连接器卡片 logo 外框（白底 + `--cp-border` 描边的 40px 图标容器）原按 Admin 规范用 `--radius-lg` = 4px。用户要求与「通道配置」页面（`ChannelConfig.tsx`）的 logo 框视觉保持一致——该页 logo 框圆角目测约 8px（实为图标 PNG 资源自带的白底圆角框，代码侧 logo 为无 CSS 外框的 `h-8 w-8`，圆角不来自 CSS token）。
- 风险：8px 圆角超出 Admin「控件/面板圆角统一 ≤4px、禁止 [8px]」铁律；且 8px 在当前 token 体系（`--radius-sm/md/lg/xl` 最大 4px，`--radius-card` 12px 仅 Tenant 业务卡消费）无对应 token，只能硬编码 `rounded-[8px]`。
- 当前处理：依用户提供截图的视觉为准，将该 logo 外框改为 `rounded-[8px]`（仅限此 logo 图标容器，归类为图标/头像类展示元素，非面板/控件本体），其余面板、卡片、按钮仍保持 4px。
- 待确认：(1) logo 图标容器是否归入「头像/图标」例外档，允许 >4px 圆角；(2) 若允许，是否补一个 `--radius-icon`（如 8px）token，避免散写 `rounded-[8px]`；(3) 通道配置页面图标框圆角的权威取值（当前依赖 PNG 资源，无 CSS 口径）。

## C-019：知识管理内置连接器「乐享知识库 / 腾讯文档企业版」品牌 logo 未登记

- 状态：`resolved`（采用官方远程 logo；建议后续沉淀进 canonical-assets）
- 现象：`client/src/pages/admin/KnowledgeManagement.tsx`（知识管理 - 内置连接器视图）需要展示「乐享知识库」「腾讯文档企业版」两个产品的品牌 logo，但 `client/src/design-assets/canonical-assets.ts` 目前仅登记了渠道图标（微信 / QQ / 企业微信 / 钉钉 / 飞书），未收录这两个知识库产品的品牌 SVG。
- 风险：品牌 logo 属 §2.8「不可回退槽位」，无候选时不得用 lucide 顶替，也不应手搓 inline SVG。
- 处理结论（2026-07-03，产品确认）：改用官方品牌 logo 远程图 —— 乐享 `https://identity.tencent.com/public/images/logo/lexiang.png`、腾讯文档企业版 `https://identity.tencent.com/public/images/logo/doc.png`；组件升级为 `ConnectorLogo`，容器沿用 `--radius-lg` = 4px + `object-contain`，加载失败自动回退到「产品名首字 + token 色方块」中性占位（兜底不留白板，非 lucide 顶替，符合 §2.8）。
- 遗留待办（非阻塞）：建议后续由设计将两枚 logo 沉淀进 `canonical-assets.ts`（新增 `knowledge.*` 槽位），实现离线可用与统一管理，届时把远程 URL 替换为登记资产；logo 外框圆角取值关联 C-018 图标容器例外档结论，如后续统一为 `--radius-icon` 一并调整。

## 新冲突记录模板

```md
## C-XXX：标题

- 状态：`needs-design-confirmation`
- 现象：
- 风险：
- 当前建议：
- 待确认：
```
