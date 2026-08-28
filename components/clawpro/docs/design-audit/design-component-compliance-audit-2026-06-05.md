# ClawPro 全量组件规范与使用合规审计报告

## 检测基线

- 检测对象分支：`feature/design-refresh-2026`
- 基线提交：`4a49d2a3214c3d7edfb6139a47facb3a0c1e005e`
- 基线提交时间：`2026-06-03 14:08:54 +0800`
- 检测时间：`2026-06-05 15:49:44 CST`
- 当前本地工作分支：`feature/design-miekoyychen`
- 审计对象：`SKILL-GLOBAL-COMPONENTS.md` 中列出的组件与规则、`client/src/components/ui/*` 中的共享实现、`client/src/pages/admin/**`、`client/src/pages/tenant/**`、`client/src/components/**` 中的实际使用情况
- 方法说明：本报告以 `git show` / `git grep` 为主，给出“规范文本 vs 共享组件实现 vs 实际业务使用”的三方证据链，目的是为后续规范收敛、文档归并、组件治理提供足够依据
- 排除项：仓库当前未跟踪文件 `docs/ClawPro设计规范沉淀与AI约束治理方案.md`、`docs/ClawPro设计规范沉淀与AI约束治理方案.pdf` 未纳入本次基线

## 核心结论

1. 上一版报告不是“没检查组件”，而是只优先报告了最影响仲裁和治理方向的高风险冲突点。本版补充为按 `SKILL-GLOBAL-COMPONENTS.md` 逐组件展开的全量检查。
2. 结论很明确：当前仓库已经不是“部组织件不合规”这么简单，而是同时存在三层问题。
3. 第一层是规范文档自身互相冲突，尤其 `SKILL.md`、`SKILL-GLOBAL-COMPONENTS.md`、`SKILL-TENANT.md` 的优先级图、颜色、圆角、组件归属不一致。
4. 第二层是共享组件实现与文档本身不一致。典型如 `SurfaceCard`、`Dialog`、`Input`、`Table`、`AdminSidebar`、`StatusTag`、`Pagination`。
5. 第三层是业务使用并没有完全收敛到共享组件。大量页面仍通过 className 或原生结构重写按钮、表格、Tab、Tooltip、Badge、树、卡片与状态标签。
6. 从“当前真正的运行标准”看，优先级应当是：`client/src/index.css` 与 `client/src/components/ui/*` 的现行实现，高于 `SKILL.md`；`SKILL-GLOBAL-COMPONENTS.md` 只能视为部分有效、部分过时的说明文档，尚不能直接作为完全可信的唯一标准。
7. 如果团队目标是“后续规范收敛、文件合并归档”，这份报告支持一个更严格的判断：不应再把所有文档直接合并，而应先逐组件判定“到底以代码为准，还是以文档为准”，再进行归档。

## 审计口径

本报告不是简单列文件，也不是只看共享组件是否存在，而是按以下口径审计每个组件：

- 文档是否定义了明确规则
- 共享实现是否与文档一致
- Tenant / Admin 是否真实使用该组件
- 是否存在被业务层大量覆盖、绕开或重写的情况
- 该组件应归类为哪种状态：`已基本对齐` / `部分对齐但存在偏差` / `文档与代码冲突` / `业务大量绕开`

## 总体盘点

### 1. `SKILL-GLOBAL-COMPONENTS.md` 涵盖范围很广，但不是每一项都已形成可执行闭环

该文档实际覆盖了至少以下组件或组件族：

- Typography
- Button / SmallIconStateButton
- Input
- Select
- Dialog
- Drawer
- Checkbox
- Switch
- DatePicker
- Tabs / Tab 切换卡 / LineTabs
- Table
- Alert / AdminNoticeAlert
- Tree / FileTree
- Badge
- StatusTag
- DropdownMenu
- Tooltip
- Popover
- Card / SurfaceCard / TenantCard
- RadioGroup
- Avatar
- Label
- Empty
- Segment
- Pagination
- Toast
- AdminSidebar

这意味着如果要“彻底知道所有规范和组件使用情况”，确实不能只看几个高风险点，而要逐组件比对。你这个要求是合理的。

### 2. 当前仓库对共享组件的依赖度其实不低，但依赖并不等于对齐

基于目标分支静态统计，以下共享组件已被较广泛使用：

- `<Button`：`925` 次
- `<DialogContent`：`182` 次
- `<StatusTag`：`174` 次
- `<Input`：`197` 次
- `<TooltipContent`：`306` 次
- `<Alert`：约 `70` 次（不含 `AlertDialog`）
- `<Badge`：`88` 次
- `<TableActionCell`：`39` 次
- `<Pagination`：`46` 次
- `<AdminSidebar*`：`48` 次
- `<SurfaceCard`：`79` 次
- `<TenantCard`：`24` 次

这说明项目并不是“没有组件体系”，而是“组件体系已经存在，但标准执行不稳定，且仍被大量业务侧局部重写”。

### 3. 仍存在大量绕开共享组件的原生实现

静态扫描发现，在排除 `client/src/components/ui/table.tsx` 本身和 `MDXRenderer` 这种内容渲染例外后，仓库内仍有至少 `24` 处原生 `<table>` 使用，集中于 Admin 业务页与部分共享组件。这直接冲突于 `SKILL-GLOBAL-COMPONENTS.md` 对 Table 的强约束。

典型例子：

- `client/src/components/MemoryPreview.tsx:530-564`
- `client/src/pages/admin/MemberManagement.tsx:3511-3538`
- `client/src/pages/admin/MemberManagement.tsx:3584-3612`
- `client/src/pages/admin/MemberManagement/NodeContentPanel.tsx:1086-1099`
- `client/src/pages/admin/MemoryManagement/components/BenchmarkTable.tsx:39-62`
- `client/src/pages/admin/MemoryManagement/components/InstanceTable.tsx:920-1201`

这类情况说明：很多“规范中已经定义的组件”并没有真正覆盖业务实现。

## 组件全量审计矩阵

| 组件 | 文档状态 | 共享实现状态 | 实际使用状态 | 审计判断 |
|------|----------|--------------|--------------|----------|
| Typography | 文档完整，但与 `SKILL.md` 冲突 | 已实现语义层 | Tenant 落地不足 | `部分对齐但推广未完成` |
| Button | 文档较完整 | 已实现 admin + tenant 变体 | 使用广泛，但 tenant/admin 均有绕行 | `部分对齐` |
| SmallIconStateButton | 文档明确 | 已实现 | 使用很少 | `对齐但未普及` |
| Input | 文档与实现冲突 | 已实现 tenant prop | 大量使用 | `文档与代码冲突` |
| Select | 文档与实现大体接近 | 已实现 tenant prop | 使用较多 | `部分对齐` |
| Dialog | 文档与实现冲突 | 实现为共享组件 | Admin 大量使用并覆盖 | `文档失真` |
| Drawer | 文档规则较多 | 已实现共享 Drawer | 使用量不高但有效 | `部分对齐` |
| Checkbox | 文档较接近 | 已实现 | Admin 为主使用 | `基本对齐` |
| Switch | 文档与实现有差异 | 已实现 | 使用较多 | `部分对齐` |
| DatePicker | 文档与实现有差异 | 已实现 tenant prop | 使用较少 | `部分对齐` |
| Tabs / Tab 切换卡 / LineTabs | 文档拆成三套 | 代码存在多种实现 | 业务大量 className 覆盖 | `高混乱区` |
| Table | 文档内部互相打架 | 共享组件强约束已存在 | 仍有大量原生表格 | `高混乱区` |
| Alert / AdminNoticeAlert | 文档较完整 | 共享实现较成熟 | 使用广，但仍有手写提示条 | `部分对齐` |
| Tree / FileTree | 文档明确 | 已有共享 FileTree | 业务仍自定义树节点 | `未完全收敛` |
| Badge | 文档较完整 | 实现较清晰 | 被频繁 className 改色 | `部分对齐` |
| StatusTag | 文档与代码存在废弃策略冲突 | 已实现多模式 | `fill` 滥用、`dot` 仍残留 | `高混乱区` |
| DropdownMenu | 文档较清晰 | 实现基本一致 | 使用稳定 | `基本对齐` |
| Tooltip | 文档清晰 | 实现清晰 | 使用非常多，但被当 Popover 用 | `部分对齐` |
| Popover | 文档清晰 | 实现清晰 | 使用较多 | `基本对齐` |
| Card / SurfaceCard / TenantCard | 文档互相冲突 | 共享实现已分流 | Tenant 仍大量手写 | `高混乱区` |
| RadioGroup | 文档较清晰 | 实现基本一致 | 使用适中 | `基本对齐` |
| Avatar | 文档简单 | 实现简单 | 使用少 | `基本对齐` |
| Label | 文档较清晰 | 实现基本一致 | 使用较多 | `基本对齐` |
| Empty | 文档较清晰 | 实现已接 Typography | 使用少量 | `基本对齐` |
| Segment | 文档与使用范围冲突 | 代码含 admin/tenant 双套 | 实际 tenant 仍误用 admin Segment | `文档与使用冲突` |
| Pagination | 文档与实现有细节偏差 | 共享实现成熟 | 使用较多 | `部分对齐` |
| Toast | 文档较清晰 | 已通过 sonner 统一 | 使用经 API 间接触发 | `基本对齐` |
| AdminSidebar | 文档与 token 数值冲突 | 实现成熟 | 已在 AdminLayout 统一使用 | `文档与代码冲突` |

## 逐组件结论

### 1. Typography

文档定义很完整，是 `SKILL-GLOBAL-COMPONENTS.md` 最成熟的一个章节之一，但它和 `SKILL.md` 的字体定义冲突。

证据：

- `SKILL-GLOBAL-COMPONENTS.md:35-219`
- `SKILL.md:108-117`
- `client/src/components/ui/Typography.tsx:5-120`
- `client/src/index.css:160-173`

实现判断：

- `Typography.tsx` 已完整实现语义文字层，绑定 `--text-*` token。
- Tenant 页面共 `12` 个，`components/topnav` 共 `7` 个，但直接导入 Typography 的 Tenant 页面只有 `3` 个，topnav 为 `0` 个。
- Tenant 中仍大量直接书写标题、正文和说明文字 class。

典型偏差：

- `client/src/pages/tenant/SkillSquare.tsx:350-355`
- `client/src/pages/tenant/ModelQuota.tsx:316-327`

结论：`Typography` 应被视为未来 Tenant 文本规范真源，但目前还不能当作“已完全落地”的既成事实。

### 2. Button / SmallIconStateButton

共享 Button 体系已经很成熟，且被全仓大量使用。

证据：

- `client/src/components/ui/button.tsx:53-289`
- `SKILL-GLOBAL-COMPONENTS.md:279-443`

使用统计：

- `<Button` 使用 `925` 次
- `Button` 相关导入文件约 `166` 个
- `<SmallIconStateButton` 使用 `5` 次

关键问题：

1. Tenant 侧虽然已有完整 `tenant-*` 体系，但仍残留非 Tenant 变体。
2. 文档要求 TableActionCell 内必须使用 `variant="link"`，但旧 `link-dark` 仍在 Admin 使用。

证据：

- `client/src/pages/tenant/ChatView.tsx:2857`
- `client/src/pages/tenant/MyOpenClaw.tsx:711`
- `client/src/pages/admin/ImageManagement/UpdateRecordsDrawer.tsx:348-352`
- `client/src/pages/admin/MemoryManagementRedesign/MemoryManagementRedesign.tsx:195`
- `client/src/pages/admin/PlatformPolicy.tsx:641`

结论：Button 体系本身是当前较可信的代码真源，但业务使用尚未完全遵守最新规则。

### 3. Input

这是一个标准文档与共享实现直接冲突的组件。

证据：

- 文档：`SKILL-GLOBAL-COMPONENTS.md:445-457`
- 组件注释：`client/src/components/ui/input.tsx:15-20`
- 实现：`client/src/components/ui/input.tsx:65-71`

冲突点：

- 文档整体写法默认是 4px 控件体系。
- 组件注释写 Tenant 形态应为 `rounded-xl（12px）`。
- 真实实现却是 `tenant ? "rounded-full" : "rounded-[4px]"`。

结论：Input 的标准不需要再从文档推断，当前只能以组件实现为准；文档和注释均需要回写修正。

### 4. Select

Select 相比 Input 更接近实际实现，但仍存在“文档写 4px，Tenant 代码走胶囊”的分流。

证据：

- `SKILL-GLOBAL-COMPONENTS.md:460-475`
- `client/src/components/ui/select.tsx:25-64`
- `client/src/components/ui/select.tsx:86-157`

实现判断：

- Admin 默认是 `rounded-[4px]`，Tenant 通过 `tenant` prop 切到 `rounded-full`。
- Dropdown 面板实现与文档大体一致：4px、白底、无明显自定义边框、统一阴影。

结论：Select 属于“共享实现已经分流，但文档尚未完整表达代码现状”的组件。

### 5. Dialog

Dialog 是共享实现与文档最失真的组件之一。

证据：

- 文档：`SKILL-GLOBAL-COMPONENTS.md:478-510`
- 组件注释：`client/src/components/ui/dialog.tsx:92-102`
- 实现：`client/src/components/ui/dialog.tsx:147-167`
- Admin 真实使用：`client/src/pages/admin/OpenClawMonitor.tsx:2682-2683`

核心问题：

1. 文档写圆角 `8px`。
2. 组件实现是 `rounded-[12px]`。
3. 组件注释写“Admin 禁止使用此弹窗组件”。
4. 实际上 Admin 里 `<DialogContent` 至少使用 `138` 次，并且很多地方直接通过 className 覆盖圆角和宽度。

结论：Dialog 当前不能再由文档仲裁，必须先决定“共享 Dialog 双端通用”还是“拆出 AdminDialog”。在此之前，文档结论都不可靠。

### 6. Drawer

Drawer 文档写得很细，实际也已有共享实现，但落地范围仍偏窄。

证据：

- `SKILL-GLOBAL-COMPONENTS.md:512-617`
- `client/src/components/ui/drawer.tsx:46-133`

使用情况：

- `<DrawerContent` 使用 `10` 次，几乎全部在 Admin。

偏差点：

- 文档强调 Header/Footer 结构、Typography 映射、ghost 头部按钮和隐藏滚动条规则。
- 组件层只提供基础容器，很多视觉规范仍依赖业务使用者自行遵守。

结论：Drawer 不是“组件不存在”，而是“组件基础存在，但规范主要停留在使用约束层，自动化约束不够强”。

### 7. Checkbox / Switch / RadioGroup / Avatar / Label / Empty

这一组基础组件整体相对健康。

证据：

- `checkbox.tsx:12-31`
- `switch.tsx:11-29`
- `radio-group.tsx:12-39`
- `avatar.tsx:11-47`
- `label.tsx:11-18`
- `empty.tsx:6-108`

判断：

- Checkbox 与 RadioGroup 与文档较为接近。
- Switch 文档写 disabled 去掉透明度，但实现仍保留 `disabled:opacity-50`，属于轻微偏差。
- Empty 已较好接入 Typography，状态健康。

结论：这些组件可作为相对容易先收敛的一组。

### 8. DatePicker

DatePicker 已实现 Tenant 胶囊分流，但文档仍描述统一 4px Trigger。

证据：

- `SKILL-GLOBAL-COMPONENTS.md:647-655`
- `client/src/components/ui/date-picker.tsx:31-36`
- `client/src/components/ui/date-picker.tsx:97-145`

结论：与 Input / Select 同类，属于“代码已经先分流，文档未同步跟上”的组件。

### 9. Tabs / Tab 切换卡 / LineTabs / Segment

这是当前第二个高混乱区。

证据：

- `tabs.tsx:19-48`
- `segment.tsx:6-320`
- `SKILL-GLOBAL-COMPONENTS.md:659-741`
- `SKILL-GLOBAL-COMPONENTS.md:1828-1868`

冲突与混用现状：

1. 文档同时存在 `Tabs`、Tab 切换卡、`LineTabs`、`Segment` 多套切换规范。
2. Tenant 实际大量通过覆盖共享 `TabsList` / `TabsTrigger` 去模拟胶囊。
3. `segment.tsx` 明确区分 Admin Segment 与 TenantSegment，但 Tenant 业务页仍在使用 Admin `SegmentList`。

典型证据：

- Tenant 使用共享 Tabs 覆盖胶囊：`client/src/pages/tenant/SkillSquare.tsx:1297-1312`
- Tenant 使用 Admin Segment：`client/src/pages/tenant/OpenClawDetail.tsx:1777-1784`
- Admin 使用 Tabs 并整体改造成 LineTabs：`client/src/pages/admin/TokensMonitor.tsx:1193-1200`
- Admin 手动覆盖 Tabs 容器/trigger：`client/src/pages/admin/AgentMigration.tsx:565-568`

结论：当前“切换控件”并没有单一清晰标准。至少需要先决定三件事：

- 页面一级导航是否统一用 `LineTabs`
- 内容区切换是否统一用 `Segment`
- Tenant 胶囊切换是否必须统一落到 `TenantSegment`，而不是继续覆盖 Tabs

### 10. Table

这是当前最需要彻底治理的组件之一。

证据：

- 文档第一版：`SKILL-GLOBAL-COMPONENTS.md:753-857`
- 文档第二版：`SKILL-GLOBAL-COMPONENTS.md:1199-1479`
- 组件实现：`client/src/components/ui/table.tsx:1-620`

核心冲突：

1. 同一份文档里，Table 被写了两套不同规则。
2. 文档前段写“标准版 14px / 紧凑版 12px”；组件头注释却明确说明“表格相关所有元素统一 12px”。
3. 文档要求禁止原生表格，但仓库仍有至少 `24` 处原生 `<table>` 实现。
4. 文档要求操作列必须统一 `variant="link"`，但旧实现和原生表格仍大量存在。

共享实现成熟度：

- `TableActionCell`、固定列、density、variant、pagination 协同都做得比较完整。
- 例如 `client/src/pages/admin/ModelConfig.tsx:600-603` 已是比较标准的使用方式。

但绕开情况严重：

- `client/src/components/MemoryPreview.tsx:530-564`
- `client/src/pages/admin/MemberManagement.tsx:3511-3538`
- `client/src/pages/admin/MemoryManagement/components/BenchmarkTable.tsx:39-62`
- `client/src/pages/admin/MemoryManagement/components/InstanceTable.tsx:920-1201`

结论：Table 不只是“有点不统一”，而是典型的“共享组件已很强，但旧业务页和文档均未跟上”。这应当是后续规范收敛的第一批对象。

### 11. Alert / AdminNoticeAlert

Alert 体系整体比 Table 健康，但也存在业务层手写提示条与 `AlertDialog` 混淆统计的问题。

证据：

- `SKILL-GLOBAL-COMPONENTS.md:860-1017`
- `client/src/components/ui/alert.tsx:9-102`
- `client/src/components/ui/admin-notice-alert.tsx:14-98`

实现判断：

- Alert variant 已较系统化。
- Typography 已接入 `AlertTitle` / `AlertDescription`。
- `AdminNoticeAlert` 也已单独沉淀。

偏差：

- 文档要求禁止手写 `bg-blue-50` / `bg-amber-50` 提示条，但业务层仍有大量原生提示盒存在，尤其 Tenant 页中的说明区。

结论：Alert 体系方向是对的，但离“禁止业务再手写提示条”还有明显差距。

### 12. Tree / FileTree

Tree 文档和共享实现存在，但业务仍保留自定义树节点实现。

证据：

- `SKILL-GLOBAL-COMPONENTS.md:1020-1134`
- `client/src/components/ui/tree.tsx:1-205`
- `client/src/pages/admin/SkillLibrary/PublicSkillTab.tsx:279-319`

问题：

- 文档强调使用共享 FileTree / shadcn 风格。
- 但 SkillLibrary 页面仍自定义 `FileTreeNode`，并且配色、hover、selected 都与规范不同。

结论：Tree 组件不是没做，而是尚未完成对具体业务的替换。

### 13. Badge

Badge 文档与实现总体清晰，但业务大量通过 className 改色，削弱了 `variant` / `color` 约束。

证据：

- `SKILL-GLOBAL-COMPONENTS.md:1136-1195`
- `client/src/components/ui/badge.tsx:23-87`
- `client/src/pages/admin/DocManagement.tsx:86-88`

关键问题：

- 文档明确禁止用 `className` 自改颜色。
- 但 `DocManagement` 等页面仍直接写 `border-blue-200 text-[#355EF1] bg-blue-50`。

结论：Badge 组件本身没问题，但业务使用并未严格遵守规则。

### 14. StatusTag

这是第三个高混乱区。

证据：

- 文档：`SKILL-GLOBAL-COMPONENTS.md:1482-1638`
- 实现：`client/src/components/ui/status-tag.tsx:5-247`
- 使用统计：`mode="text"` 约 `35` 次，`mode="fill"` 约 `115` 次，`mode="dot"` 约 `13` 次

问题：

1. 文档要求“全场景状态默认统一使用 `mode="text"`”。
2. 实际使用里 `mode="fill"` 明显更多。
3. 文档说 `mode="dot"` 已废弃，但实际业务与设计系统页仍有残留。

典型残留：

- `client/src/pages/admin/MemberManagement/NodeContentPanel.tsx:1236`
- `client/src/pages/admin/MemoryManagementRedesign/MemoryManagementRedesign.tsx:106-110`
- `client/src/pages/admin/Security/AIAgent/Assets/AgentAssetsList.tsx:624`

结论：StatusTag 的组件能力已经丰富，但团队对“状态、信息、分类、角色”四种语义的使用边界并未真正稳定下来。

### 15. DropdownMenu / Tooltip / Popover

这组三个浮层组件，底层实现相对健康，但 Tooltip 被当做轻量 Popover 使用的情况较多。

证据：

- `dropdown-menu.tsx:32-236`
- `tooltip.tsx:36-56`
- `popover.tsx:18-37`
- `client/src/components/ui/file-browser.tsx:126-129`

问题：

- Tooltip 文档明确说“大块多行内容应改用 Popover”。
- 但 `file-browser.tsx` 里 `TooltipContent` 已被覆盖成白底、描边、shadow-lg 的卡片式结构，本质是在把 Tooltip 当 Popover 用。

结论：DropdownMenu 和 Popover 较稳；Tooltip 的使用边界仍需要治理。

### 16. Card / SurfaceCard / TenantCard

这是第一版报告已经指出的高混乱区，本版结论不变，而且证据更充分。

证据：

- `card.tsx:5-82`
- `Surface.tsx:12-29,44-53,149-232`
- `SKILL-GLOBAL-COMPONENTS.md:1711-1726`
- `SKILL-GLOBAL-COMPONENTS.md:253-260`
- `index.css:179-205`

问题：

1. `Card` 与 `SurfaceCard` / `TenantCard` 三套卡片语义并存。
2. 文档对 `rounded-xl` 的解释前后矛盾。
3. Tenant 仍大量手写 12px 业务卡，没有完全收敛到 `TenantCard`。

统计：

- `<SurfaceCard`：`79` 次
- `<TenantCard`：`24` 次
- `<Card`：`51` 次

结论：后续归并文档时，卡片体系必须单独梳理“哪类业务该用哪种卡片”，否则只会继续扩散混用。

### 17. Pagination

Pagination 组件整体已经可用，但文档和实现仍存在尺寸、圆角等细节差异。

证据：

- `SKILL-GLOBAL-COMPONENTS.md:1872-1968`
- `client/src/components/ui/pagination.tsx:20-457`

差异点：

- 文档写 default 32px、small 24px、`rounded-lg`。
- 实现中 item 高度是 default `h-7 min-w-[28px]`、small `h-6 min-w-[24px]`，并且字号被统一压到 `12px`。

结论：Pagination 已接近可作为标准，但文档需要跟代码重新核对一遍数值。

### 18. Toast

Toast 的共享实现比较清晰，通过 sonner 已统一。

证据：

- `SKILL-GLOBAL-COMPONENTS.md:1984-2030`
- `client/src/components/ui/sonner.tsx:20-57`

偏差点：

- 文档说关闭按钮在右侧垂直居中；实现里 closeButton 仍通过绝对定位调位置，不完全等于“右侧普通流式布局”。

结论：Toast 可以归入相对稳定组件，但仍有少量视觉实现与文档描述差异。

### 19. AdminSidebar

AdminSidebar 已经形成独立体系，但文档 token 与代码 token 数值不一致。

证据：

- 文档：`SKILL-GLOBAL-COMPONENTS.md:2064-2155`
- 实现：`client/src/components/ui/admin-sidebar.tsx:113-567`
- token：`client/src/index.css:248-280`

冲突点：

- 文档写 `--admin-sidebar-width = 232px`，代码 token 为 `240px`。
- 文档写 header height `72px`，代码 token 为 `104px`。
- 文档写 hover 背景 `#f5f5f5`，代码 token 为 `rgba(180, 191, 225, 0.14)`。
- 文档写 badge 圆角 `2px`，实现为 `rounded-full`。
- 文档写底部用户 Avatar 为 `rounded-md` 绿渐变，代码实现为 `rounded-full` 且依赖 sidebar avatar token。

结论：AdminSidebar 不是“没标准”，而是“代码已经演化成了独立真实标准，文档明显滞后”。

## 最值得向团队说明的冲突类型

### A. 文档内部自相矛盾

- Table 在 `SKILL-GLOBAL-COMPONENTS.md` 中出现两套不同规范
- Card / SurfaceCard 在圆角章节和 Card 章节解释不一致
- StatusTag 一边强调 `mode="text"` 为统一标准，一边仍保留大量历史兼容描述

### B. 文档与共享组件代码冲突

- Input：文档/注释 vs `rounded-full`
- Dialog：文档/注释 vs 共享实现和 Admin 实际使用
- SurfaceCard：文档 12px vs 代码 4px
- Pagination：文档数值 vs 实现尺寸
- AdminSidebar：文档 token vs `index.css` token

### C. 共享组件已存在，但业务仍绕开

- Table：原生 `<table>` 仍大量存在
- Tree：共享 `FileTree` 已存在，但业务仍自写 `FileTreeNode`
- Tabs/Segment：共享实现已存在，但业务仍通过 className 重写
- Badge：共享组件已存在，但业务继续手写蓝色 outline 样式

### D. 共享组件被大量覆盖，导致“看似在用组件，实际没对齐规范”

- `TabsList` / `TabsTrigger`
- `DialogContent`
- `TooltipContent`
- `Badge`
- `Button`

## 哪些规范已经足够成熟，可以优先收敛为真源

更接近“可直接保留并作为今后规范真源”的组件：

- Button 基础体系
- Select 基础体系
- Checkbox / RadioGroup / Label / Empty
- DropdownMenu / Popover
- Toast

这些组件并非完全无问题，但共享实现整体清晰，修订成本相对低。

## 哪些组件必须先开专题治理，再谈文档合并

必须专题治理的高风险组件：

1. Table
2. Card / SurfaceCard / TenantCard
3. Tabs / Segment / LineTabs / TenantSegment
4. Dialog / Drawer
5. StatusTag
6. AdminSidebar
7. Typography（主要是推广落地，不是重写组件）

## 归档与收敛建议

### 1. 不建议直接把 `SKILL-GLOBAL-COMPONENTS.md` 当成现行标准全文保留

原因不是这份文档没价值，而是它已经混合了：

- 仍有效的规则
- 已经落到代码但文档数值过时的规则
- 与当前共享实现直接冲突的规则
- 业务层尚未完成迁移的理想态规则

它更像“设计与实现混合历史稿”，而不是完全可信的最终基线。

### 2. 建议按组件分层定真源

- token / radius / alert / sidebar 等基础值：以 `client/src/index.css` 为真源
- 共享视觉实现：以 `client/src/components/ui/*` 为真源
- Tenant 差异：以 `SKILL-TENANT.md` 补充说明，不重复共享规则
- 总体治理与加载说明：以 `SKILL.md` 承载
- `SKILL-GLOBAL-COMPONENTS.md` 改造成“组件目录 + 链接式说明”，不再手写大量可能漂移的数值

### 3. 更适合做归档的方式

不是把所有文档合并成一个巨文档，而是：

1. 保留一个总入口文档说明优先级与职责。
2. 每个高风险组件只保留一份短规范，并明确“代码真源文件”。
3. 将容易漂移的视觉数值尽量从 prose 文档中删掉，改为引用代码 token / 组件 API。

## 核心摘要

- 当前项目不是缺少组件规范，而是组件规范、共享实现、业务使用三者还没有完全对齐。
- `SKILL-GLOBAL-COMPONENTS.md` 覆盖面很广，但其中不少组件规则已经与当前代码实现不一致，不能直接视为完全可信的唯一标准。
- 共享组件体系已经存在，且使用量不低，但业务层仍大量绕开或覆盖共享组件，导致“看似统一、实际不统一”。
- 若后续要做规范收敛和文档归档，应先逐组件判定真源，再合并文档；不能先合文档再期待统一实现。

## 附录：高价值证据索引

### 文档总目录

- `SKILL-GLOBAL-COMPONENTS.md:35-2155`

### 高冲突组件

- Input：`client/src/components/ui/input.tsx:15-20`、`client/src/components/ui/input.tsx:65-71`
- Dialog：`client/src/components/ui/dialog.tsx:92-102`、`client/src/components/ui/dialog.tsx:147-167`
- Surface/Card：`client/src/index.css:179-205`、`client/src/components/ui/Surface.tsx:12-29`
- Table：`client/src/components/ui/table.tsx:24-67`、`client/src/components/ui/table.tsx:470-620`
- AdminSidebar：`client/src/index.css:248-280`、`client/src/components/ui/admin-sidebar.tsx:113-567`
- StatusTag：`client/src/components/ui/status-tag.tsx:17-247`

### 业务绕开样本

- 原生 Table：`client/src/components/MemoryPreview.tsx:530-564`
- 原生 Table：`client/src/pages/admin/MemberManagement.tsx:3511-3538`
- 自定义 FileTreeNode：`client/src/pages/admin/SkillLibrary/PublicSkillTab.tsx:279-319`
- Tabs 重写胶囊：`client/src/pages/tenant/SkillSquare.tsx:1300-1306`
- Tenant 使用 Admin Segment：`client/src/pages/tenant/OpenClawDetail.tsx:1777-1784`
- Badge 直接改色：`client/src/pages/admin/DocManagement.tsx:86-88`
- Tooltip 当 Popover 用：`client/src/components/ui/file-browser.tsx:126-129`
- TableActionCell 正确用法样本：`client/src/pages/admin/ModelConfig.tsx:600-603`
- `link-dark` 残留：`client/src/pages/admin/ImageManagement/UpdateRecordsDrawer.tsx:348-352`
