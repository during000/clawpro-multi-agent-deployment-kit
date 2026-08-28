# ClawPro 设计规范冲突与实现偏差审计报告

## 检测基线

- 检测对象分支：`feature/design-refresh-2026`
- 基线提交：`4a49d2a3214c3d7edfb6139a47facb3a0c1e005e`
- 基线提交时间：`2026-06-03 14:08:54 +0800`
- 检测时间：`2026-06-05 15:49:44 CST`
- 当前本地工作分支：`feature/design-miekoyychen`
- 检测方式：所有结论均以 `feature/design-refresh-2026` 分支快照为准，通过 `git show` / `git grep` 对文档、共享组件、Tenant 页面、Admin 页面进行比对，不以当前工作分支文件内容替代基线
- 排除项：仓库当前存在未纳入本次基线的未跟踪文件 `docs/ClawPro设计规范沉淀与AI约束治理方案.md`、`docs/ClawPro设计规范沉淀与AI约束治理方案.pdf`

## 核心结论

1. 当前项目并不存在单一、无冲突的“设计规范唯一真源”。从实际落地情况看，运行中的有效基线更接近 `client/src/index.css`、`client/src/components/ui/*` 与 `SKILL-GLOBAL-COMPONENTS.md` 的部分内容，而不是 `SKILL.md`。
2. 现有规范的首要问题不是“某一份文档细节不够”，而是“规范优先级图本身互相冲突”。`SKILL.md`、`SKILL-GLOBAL-COMPONENTS.md`、`SKILL-TENANT.md` 对谁是最高优先级给出了不同答案，这会直接导致 AI 和人工实现都无法稳定判断该对齐哪一份规则。
3. Tenant / Admin 的视觉分流已经在代码中部分成型，但仍处于“组件化方案”和“业务侧手写样式”并存的状态。最典型的分裂点包括：品牌色、圆角/卡片体系、Dialog 所有权、Typography 语义化落地、Tenant 按钮变体迁移。
4. 如果要高效收敛规范，最应优先对齐的不是继续补充 `SKILL.md`，而是先冻结“代码级真实标准”与“文档级说明标准”的边界：令 `index.css + ui 组件` 成为运行时真源，令 `SKILL-GLOBAL-COMPONENTS.md` 成为共享/Admin 说明文档，令 `SKILL-TENANT.md` 仅保留 Tenant 差异规则，`SKILL.md` 退回总入口与治理说明，不再承载具体 token 数值与组件尺寸。
5. 本报告已经覆盖 Tenant 与 Admin 两端，但它对比的是“目标分支代码实现态”与“仓库内规范文档”，不是线上运行截图审计。对管理层判断规范收敛方向已经足够；若后续需要验证线上视觉结果，还应再补一轮页面截图或实际运行态走查。

## 审计范围与方法

本次审计聚焦两类问题：

- 文档之间是否互相冲突，导致无法稳定下达“对齐规范”的指令。
- 实际代码是否遵循文档，还是已经形成另一套隐性标准。

重点检查对象：

- 规范文档：`SKILL.md`、`SKILL-GLOBAL-COMPONENTS.md`、`SKILL-TENANT.md`
- 设计真源候选：`client/src/index.css`
- 共享 UI：`client/src/components/ui/Typography.tsx`、`button.tsx`、`Surface.tsx`、`input.tsx`、`dialog.tsx`
- Tenant 实现：`client/src/pages/tenant/**`、`client/src/components/topnav/**`
- Admin 实现：`client/src/pages/admin/**`

说明：

- 文中出现的计数，如 Tenant 页面对硬编码样式命中数、Tenant 按钮变体使用量，均为基于 `git grep` 的仓库静态证据，适合作为风险量级与趋势判断，不应视为像素级视觉审计结论。

## 总体判断

### 1. 当前最接近“真实标准”的不是单一文档，而是代码本身

证据链显示，多个最关键设计决策已经由代码明确表达：

- 颜色、字体、圆角和阴影基线集中在 `client/src/index.css`
- 卡片分流规则集中在 `client/src/components/ui/Surface.tsx`
- Tenant 按钮变体集中在 `client/src/components/ui/button.tsx`
- Typography 语义层集中在 `client/src/components/ui/Typography.tsx`

与之相反，`SKILL.md` 在品牌色、字体、圆角等高频规则上明显滞后，因此它不适合继续被当作“具体组件规范真源”。

### 2. 当前最严重的问题是治理冲突，其次才是实现不统一

如果团队现在直接下达“对齐组件规范”“对齐 token”的要求，执行层会首先卡在以下问题：

- Tenant 页面到底是以 `SKILL-TENANT.md` 为最高优先级，还是以 `SKILL-GLOBAL-COMPONENTS.md` 为组件最高优先级。
- `rounded-xl` 在文档里到底代表 12px，还是在本项目里已经被压缩成 4px。
- Dialog 是否有 Admin 专用实现，如果没有，为何共享 `DialogContent` 已被 Admin 大量使用。
- 文字、按钮、卡片究竟应该走语义组件，还是允许业务页继续手写原子 class。

这类问题不先收敛，继续补文档只会继续制造冲突。

## 冲突矩阵

| 主题 | `SKILL.md` | `SKILL-GLOBAL-COMPONENTS.md` | `SKILL-TENANT.md` | 实际代码 / 实现态 | 结论 |
|------|------------|------------------------------|-------------------|-------------------|------|
| 规范优先级 | 指定 `SKILL-GLOBAL-COMPONENTS.md` 为组件样式最高优先级；Tenant 冲突再以 `SKILL-TENANT.md` 为准 | 明确 Admin 场景为“本文件 > SKILL.md”，Tenant 场景为“Tenant > 本文件” | 明确写出 `SKILL-TENANT.md > SKILL.md > SKILL-GLOBAL-COMPONENTS.md` | 实现层同时存在 shared/global 组件与 tenant 差异扩展 | 三份文档形成了不一致的优先级图，是第一治理风险 |
| 品牌色 | 仍以 `#007AFF` / `#5856D6` 及旧渐变为品牌色 | 多数品牌说明已转向 `#1447E6` / `#355EF1` | Tenant 规则基本围绕较新的蓝色体系展开 | `index.css` 核心品牌 token 为 `#1447E6` / `#355ef1`，但 Admin 与 Tenant 业务页仍保留旧蓝紫渐变 | 文档与代码均存在新旧品牌并行，`SKILL.md` 已明显过时 |
| Typography | 仍写 `Inter` / `DM Mono` | 明确要求 Tenant 使用 `Typography.tsx` 语义组件 | Tenant 范围接受共享 Typography 体系 | `index.css` 实际为 `PingFang SC` / `Open Sans` / `Menlo`；`Typography.tsx` 已实现语义层，但 Tenant 落地很低 | 文档目标已更新，但 `SKILL.md` 与实现态不一致，推广也未完成 |
| 圆角 / 卡片 | 仍描述较旧的大圆角体系，如 `rounded-2xl` 卡片、`rounded-lg` 输入框 | 表格写 `Card(SurfaceCard) = rounded-xl(12px)` | Tenant 需要更柔和的业务卡体系 | `index.css` 明确 `--radius-xl = 4px`、`--radius-card = 12px`；`Surface.tsx` 明确 `SurfaceCard=4px`、`TenantCard=12px` | 文档内部与代码直接矛盾，尤其 `rounded-xl` 在本仓库不再等于 12px |
| Button 体系 | 总体规则仍偏旧 | 共享组件约束强调不要覆盖默认样式 | Tenant 要通过 `tenant-*` 变体扩展，不得改默认 Admin 变体 | `button.tsx` 已实现完整 `tenant-*` 变体，但 Tenant 页面仍混用 `outline` / `ghost` / `secondary` | 正确方向已在代码中存在，但迁移未完成 |
| Input 体系 | 无法提供当前真实规则 | 倾向统一控件圆角 | Tenant 需要差异控件形态 | `input.tsx` 注释写 Tenant 应为 12px，但实际实现是 `rounded-full` | 组件注释本身与实现冲突，说明规范已进入“文档滞后于代码”状态 |
| Dialog 所有权 | 未定义真实现状 | 无法单独解释 Admin 实现 | 未覆盖 Admin | `dialog.tsx` 注释写“当前 rounded-[12px] Dialog 仅限 Tenant，Admin 禁止使用”，但 Admin 页面大量使用该组件，甚至通过 className 覆盖成 4px | 这是最典型的“组件注释失真”问题，现状应以代码使用事实为准 |
| 禁止业务页手写样式 | 多处强调不要覆盖组件 | 强调统一通过组件和 token | Tenant 也主张通过共享组件扩展 | Tenant/Admin 仍有大量 `rounded-[4px]`、`rounded-[12px]`、hex 色值、`boxShadow` 手写 | 组件化方向正确，但执行层未完全进入规范化状态 |

## 关键证据

### A. 规范优先级发生直接冲突

1. `SKILL.md` 将 `SKILL-GLOBAL-COMPONENTS.md` 设为“组件样式最高优先级”，并要求 Tenant 工作同时加载三份文档，冲突时再以 `SKILL-TENANT.md` 为准。

- `SKILL.md:18-28`
- `SKILL.md:32-42`

2. `SKILL-TENANT.md` 又明确写出：`本文件 > SKILL.md > SKILL-GLOBAL-COMPONENTS.md`。

- `SKILL-TENANT.md:15-17`

3. `SKILL-GLOBAL-COMPONENTS.md` 对 Admin 场景又写成“本文件 > SKILL.md”。

- `SKILL-GLOBAL-COMPONENTS.md:23-31`

结论：团队当前并不是“缺少规范”，而是“规范仲裁关系不一致”。在这种状态下，任何“请对齐规范”的指令都存在多解。

### B. 品牌色规范与实现已经分裂为新旧两套

1. `SKILL.md` 仍将品牌色定义为：

- Brand Blue：`#007AFF`
- Brand Purple：`#5856D6`
- 品牌渐变：`linear-gradient(135deg, #007AFF, #5856D6)`

证据：`SKILL.md:46-58`

2. 实际代码基线已经切换到新蓝色体系：

- `client/src/index.css:67-69`
- `client/src/index.css:160-203` 中语义 token 使用 `#1447E6`

其中可直接确认：

- `--color-blue-500: #355ef1`
- `--brand-blue: #1447e6`
- `--brand-purple: #1447e6`
- `--text-brand: #1447e6`
- `--ring: #1447E6`

3. `SKILL-GLOBAL-COMPONENTS.md` 的品牌色说明已大体转向 `#1447E6` / `#355EF1`，与 `index.css` 更接近。

- `SKILL-GLOBAL-COMPONENTS.md:49-61`

4. 但旧蓝紫体系并未清理干净，仍在 Admin 与 Tenant 业务页中被实际使用。

Admin 例子：

- `client/src/components/AdminModeToggle.tsx:12`
- `client/src/pages/admin/MemoryManagement/components/EnableConfirmDialog.tsx:50`
- `client/src/pages/admin/MemoryManagement/components/EnableConfirmDialog.tsx:110`
- `client/src/pages/admin/MemoryManagement/components/FreeVersionCard.tsx:93`
- `client/src/pages/admin/MemoryManagement/components/FreeVersionCard.tsx:181`
- `client/src/pages/admin/MemoryManagement/components/GroupColumn.tsx:314`
- `client/src/pages/admin/MemoryManagement/components/OverviewStats.tsx:39`
- `client/src/pages/admin/ModelConfig.tsx:898`
- `client/src/pages/admin/SkillLibrary/PublicSkillTab.tsx:149`
- `client/src/pages/admin/SkillLibrary/PublicSkillPackageTab.tsx:179`

Tenant 例子：

- `client/src/pages/tenant/ChatView.tsx:2860`
- `client/src/pages/tenant/ChatView.tsx:2888`
- `client/src/pages/tenant/OpenClawDetail.tsx:3085`

结论：当前真实状态不是“项目已经统一到新 token”，而是“核心 token 已迁移，但业务层残留旧品牌写法”。

### C. 圆角与卡片体系是当前最典型的文档失真区域

1. `SKILL.md` 仍保留旧圆角体系，包括卡片 `rounded-2xl (16px)`、输入/导航 `rounded-lg (8px)`、按钮 `rounded-md (6px)`、Dialog `rounded-lg (8px)`。

- `SKILL.md:175-184`

2. `index.css` 已明确改写全站圆角语义：

- `--radius-lg = 4px`
- `--radius-xl = 4px`
- `--radius-card = 12px`
- 注释明确说明“不要把 rounded-xl 等同于 12px；用户端要 12px 必须用 TenantCard”

证据：`client/src/index.css:71-84`

3. `Surface.tsx` 进一步把分流写死为组件语义：

- `SurfaceCard`：Admin / 全局卡片，4px
- `TenantCard`：Tenant 业务卡，12px

证据：

- `client/src/components/ui/Surface.tsx:12-29`
- `client/src/components/ui/Surface.tsx:44-53`
- `client/src/components/ui/Surface.tsx:149-232`

4. `SKILL-GLOBAL-COMPONENTS.md` 在“圆角规范”中却仍写：`Card（SurfaceCard） | rounded-xl（12px）`。

- `SKILL-GLOBAL-COMPONENTS.md:253-260`

这与 `index.css` 和 `Surface.tsx` 形成直接矛盾。该点非常适合作为对外说明“规范互相打架”的核心案例。

5. Tenant 实现态并未完全收敛到 `TenantCard`。

- `git grep` 统计到 `<TenantCard` 使用 `21` 次
- `git grep` 统计到 `<SurfaceCard` 使用 `76` 次（覆盖共享、Admin、部分 Tenant 场景）

Tenant 中同时存在大量手写卡片：

- `client/src/pages/tenant/OpenClawDetail.tsx:1793` 直接写 `bg-white rounded-[12px] border ...`，并手写 `boxShadow: "var(--shadow-card)"`
- `client/src/pages/tenant/ToolsMcpPanel.tsx:730-738` 直接写 `rounded-[12px] border transition-all` 并继续手写 `boxShadow`
- `client/src/pages/tenant/OpenClawDetailGuide.tsx` 多处混用 `rounded-[12px]` 卡片与 `rounded-[4px]` 局部控件
- `client/src/pages/tenant/TenantIconAudit.tsx` 在 Tenant 页面中使用 `<SurfaceCard className="rounded-[4px] ...">`，说明 Tenant/Global 卡片语义仍未彻底隔离

结论：代码中已经存在正确的卡片语义 API，但业务实现仍处于“语义组件 + 手写样式”双轨并存状态。

### D. Typography 规则方向明确，但 Tenant 落地率较低

1. `SKILL.md` 仍写旧字体：`Inter` / `DM Mono`。

- `SKILL.md:108-117`

2. 实际主题与 Typography 组件已切到新体系：

- `client/src/index.css:47-69` 定义 `font-sans = PingFang SC`、`font-en = Open Sans`、`font-mono = Menlo/Consolas`
- `client/src/components/ui/Typography.tsx:5-31` 绑定 `--text-*` 语义 token
- `client/src/components/ui/Typography.tsx:74-120` 等定义 `TenantHeroTitle`、`TenantPageTitle`、`BodyText` 等语义组件

3. `SKILL-GLOBAL-COMPONENTS.md` 已明确要求 Tenant 走 Typography 语义化。

- `SKILL-GLOBAL-COMPONENTS.md:35-47`
- `SKILL-GLOBAL-COMPONENTS.md:63-134`
- `SKILL-GLOBAL-COMPONENTS.md:192-219`

4. 但目标分支中的 Tenant 页面落地率很低。

- Tenant 页面 `.tsx` 共 `12` 个
- `components/topnav` `.tsx` 共 `7` 个
- 实际导入 `Typography` 的 Tenant 页面只有 `3` 个：
  - `client/src/pages/tenant/FileSpace.tsx`
  - `client/src/pages/tenant/HelpDocs.tsx`
  - `client/src/pages/tenant/ModelQuota.tsx`
- `components/topnav` 导入 `Typography` 的文件数为 `0`

5. 典型未对齐例子：

- `client/src/pages/tenant/SkillSquare.tsx:350-355` 直接手写 Hero 标题和副标题 class，而不是 `TenantHeroTitle` / `BodyText`
- `client/src/pages/tenant/ModelQuota.tsx:316-327` 标题仍手写 class，Tooltip Trigger 使用 `Button variant="ghost"` 并通过 className 直接写蓝色文本
- `client/src/pages/tenant/MyOpenClaw.tsx:1065-1090` 在 `SurfaceInner` 内继续使用大量手写文字样式

结论：Typography 已经具备成为 Tenant 文字真源的条件，但执行面还未完成迁移，当前不能宣称“Tenant 已全面按 Typography 规范执行”。

### E. Tenant 按钮分流方案已存在，但迁移尚未闭环

1. `button.tsx` 已实现完整 Tenant 按钮变体族：

- `tenant-primary`
- `tenant-outline`
- `tenant-outline-r20`
- `tenant-destructive`
- `tenant-ghost`
- `tenant-plain`
- `tenant-dialog-confirm`

证据：`client/src/components/ui/button.tsx:163-260`

2. 目标分支中 Tenant 页面对 `tenant-*` 变体的实际使用量说明这条路线已被采用，而非停留在文档中：

- `tenant-outline`: `48`
- `tenant-primary`: `23`
- `tenant-destructive`: `5`
- `tenant-dialog-confirm`: `4`
- `tenant-plain`: `3`
- `tenant-ghost`: `2`
- 合计 `85`

3. 但 Tenant 业务页仍混用非 Tenant 变体，至少命中 `21` 次：

- `ghost`
- `outline`
- `secondary`（主要出现在 Badge）

典型例子：

- `client/src/pages/tenant/ChatView.tsx:2857` 使用 `variant="outline"`
- `client/src/pages/tenant/MyOpenClaw.tsx:711` 使用 `variant="outline"`
- `client/src/pages/tenant/OpenClawDetail.tsx:1494` 使用 `variant="ghost"`
- `client/src/pages/tenant/OpenClawDetailGuide.tsx:684-686` 使用 `variant="outline"`
- `client/src/pages/tenant/ModelQuota.tsx:325-327` 使用 `variant="ghost"` 并重写颜色

结论：Tenant 按钮组件标准已经存在，但业务代码未完全迁移，仍保留“共享默认变体 + 业务覆盖”的旧习惯。

### F. Input 和 Dialog 的“注释标准”已落后于真实实现

1. `input.tsx` 注释写 Tenant 形态应为 `rounded-xl（12px）`，但真实实现是：

- `tenant ? "rounded-full" : "rounded-[4px]"`

证据：`client/src/components/ui/input.tsx:15-20` 与 `client/src/components/ui/input.tsx:65-71`

这说明组件注释本身已不能被直接当作标准。

2. `dialog.tsx` 注释写“当前 rounded-[12px] 圆角弹窗仅限 Tenant，Admin 禁止使用此弹窗组件”。

- `client/src/components/ui/dialog.tsx:92-102`

3. 但 Admin 对共享 `DialogContent` 的使用非常广泛，已不是偶发例外，而是体系性依赖。

`git grep` 在 `client/src/pages/admin/**` 中命中大量 `<DialogContent` 使用，覆盖成员管理、文件管理、镜像管理、监控、Memory 管理、安全组等多处业务。

4. 更直接的矛盾是，Admin 甚至会显式覆盖共享 Dialog 的圆角，说明团队默认接受“共享 Dialog + Admin 局部修正”的现实模式。

- `client/src/pages/admin/OpenClawMonitor.tsx:2682-2683` 使用 `<DialogContent className="rounded-[4px] sm:max-w-[680px]">`

结论：Dialog 的“Tenant 专用”说法在当前代码库中不成立。若继续保留这段注释，只会误导后续实现。

### G. 业务侧硬编码样式仍然大量存在

静态 grep 粗计结果：

- Tenant 页面中，`#[hex]` / `boxShadow:` / `shadow-[` 等硬编码样式相关命中约 `668` 次
- Admin 页面中，同类命中约 `2592` 次

说明：这是风险量级指标，不代表 668/2592 个独立问题，但足以证明“规范已经完全组件化、token 化”的说法不成立。

高风险样本：

- `client/src/pages/tenant/OpenClawDetail.tsx` 同时大量出现 `rounded-[12px]`、`rounded-[4px]`、hex 色值、`boxShadow`
- `client/src/pages/tenant/ChatView.tsx` 存在旧品牌渐变、手写卡片、手写按钮容器
- `client/src/pages/tenant/OpenClawDetailGuide.tsx` 大量以业务层 class 拼接实现局部规则
- `client/src/pages/admin/MemoryManagement/**` 仍残留旧品牌蓝紫渐变与自定义确认弹窗视觉

## Tenant / Admin 实现态判断

### Tenant 端

Tenant 当前状态不是“没有规范”，而是“规范结构已经出现，但业务实现仍高度混合”：

- 已有明确的 Tenant 专属按钮变体体系
- 已有明确的 TenantCard 与 Typography 方案
- 但大量页面仍在手写圆角、按钮、文本、阴影、颜色
- 同一页面内同时出现 12px 卡片、4px 控件、旧梯度、新品牌蓝、原子类文本写法

因此，Tenant 当前最适合的判断不是“完全失控”，而是“已有正确方向，但未完成规范收敛”。

### Admin 端

Admin 的共享组件依赖程度高于 Tenant，但也没有真正达到严格统一：

- `SurfaceCard` 和共享 `DialogContent` 已被广泛使用
- Admin 页面仍保留大量旧品牌色、旧渐变和局部手写覆盖
- 部分页面通过 className 覆盖共享组件，使共享组件注释和真实使用方式出现偏差

因此，Admin 的问题更像“共享组件已成为主路径，但文档与历史遗留样式尚未清理”。

## 风险分级

### P0：必须先定仲裁规则，否则无法稳定执行

- 三份规范文档的优先级图互相冲突
- `SKILL.md` 与 `SKILL-GLOBAL-COMPONENTS.md` / `SKILL-TENANT.md` 的职责边界不清

### P1：文档已失真，会直接误导设计走查与 AI 实现

- `SKILL.md` 品牌色仍是旧蓝紫体系
- `SKILL.md` Typography 仍是 `Inter` / `DM Mono`
- `SKILL-GLOBAL-COMPONENTS.md` 把 `SurfaceCard rounded-xl` 写成 12px
- `dialog.tsx` 注释宣称 Admin 禁止使用共享 Dialog
- `input.tsx` 注释宣称 Tenant Input 为 12px，代码实际为全圆角

### P2：实现收敛未完成，后续改动仍容易继续制造偏差

- Tenant Typography 落地率偏低
- Tenant 按钮变体迁移未闭环
- Tenant/Admin 仍有大量业务侧硬编码样式
- 新旧品牌色在业务页共存

## 建议的收敛路径

以下建议不是抽象流程建议，而是基于当前证据推导出的最低成本收敛路径。

### 1. 先冻结“真源分工”，再改正文档

建议分工如下：

- 运行时真源：`client/src/index.css` + `client/src/components/ui/*`
- 共享 / Admin 文档真源：`SKILL-GLOBAL-COMPONENTS.md`
- Tenant 差异文档真源：`SKILL-TENANT.md`
- 总入口与治理说明：`SKILL.md`

其中 `SKILL.md` 应去掉具体 token 值、圆角数值、旧品牌梯度、旧字体体系，避免再次与代码分叉。

### 2. 将“可执行规范”尽量下沉到组件 API，而不是继续写 prose

优先级最高的几个点：

- Card：只保留 `SurfaceCard` / `TenantCard` 两套明确语义，不再在文档中让 `rounded-xl` 同时表示两个数值
- Dialog：要么补一个 Admin 专用 Dialog，要么明确共享 Dialog 就是双端通用组件，并修正文档与注释
- Input / Select / Button：组件 props 比文档说明更可信，文档应反向引用组件 API，而不是重复手抄规则
- Typography：若决定推进语义化，就应明确 Tenant 新增页面必须使用 Typography，且在 lint / review 中执行

### 3. 文档合并顺序建议

建议不是“把三份文档机械拼接”，而是按失真风险处理：

1. 先把 `SKILL.md` 中已经失真的 token、品牌色、字体、圆角数值移除或标记废弃。
2. 再把 `SKILL-GLOBAL-COMPONENTS.md` 修正为与 `index.css` / `Surface.tsx` / `button.tsx` / `dialog.tsx` 一致。
3. 最后保留 `SKILL-TENANT.md` 作为 Tenant 差异补充，不再重复共享规则全文。

### 4. 第一批最值得治理的实现点

如果要选最能立刻降低混乱度的治理对象，建议按以下顺序：

1. 圆角 / 卡片体系：先统一 `SurfaceCard`、`TenantCard`、Dialog、Input 的真实标准。
2. 品牌色：清理旧 `#007AFF / #5856D6` 及旧梯度残留。
3. Dialog 所有权：修正“Tenant only”失真注释，并决定是否拆 Admin Dialog。
4. Tenant Typography：从 Hero 标题、页面标题、说明文本、Tooltip 文本开始收敛。
5. Tenant 按钮：把剩余 `outline` / `ghost` / `secondary` 的旧用法迁走。

## 核心摘要

如果需要快速说明当前规范问题，可直接使用以下表述：

- 当前 ClawPro 设计规范的问题，不是缺文档，而是文档优先级互相冲突，导致无法稳定判断该对齐哪份规则。
- 从仓库实现态看，真正起作用的标准更接近 `index.css + shared ui 组件 + SKILL-GLOBAL-COMPONENTS.md`，而不是 `SKILL.md`。
- Tenant / Admin 差异已经部分产品化，但仍有大量业务层手写样式和旧规范残留，所以目前属于“方向已形成，收敛未完成”。
- 后续文档治理不应继续堆新规范，而应先确定真源分工，再清理失真文档和历史样式。

## 附录：核心证据索引

### 文档优先级

- `SKILL.md:18-28`
- `SKILL.md:32-42`
- `SKILL-TENANT.md:15-17`
- `SKILL-GLOBAL-COMPONENTS.md:23-31`

### 品牌色与 token

- `SKILL.md:46-58`
- `client/src/index.css:55-69`
- `SKILL-GLOBAL-COMPONENTS.md:49-61`
- `client/src/components/AdminModeToggle.tsx:12`
- `client/src/pages/admin/MemoryManagement/components/EnableConfirmDialog.tsx:50`
- `client/src/pages/admin/MemoryManagement/components/FreeVersionCard.tsx:93`
- `client/src/pages/admin/ModelConfig.tsx:898`
- `client/src/pages/tenant/ChatView.tsx:2860`
- `client/src/pages/tenant/OpenClawDetail.tsx:3085`

### 圆角 / 卡片

- `SKILL.md:175-184`
- `client/src/index.css:71-84`
- `SKILL-GLOBAL-COMPONENTS.md:253-260`
- `client/src/components/ui/Surface.tsx:12-29`
- `client/src/components/ui/Surface.tsx:44-53`
- `client/src/pages/tenant/OpenClawDetail.tsx:1793`
- `client/src/pages/tenant/ToolsMcpPanel.tsx:730-738`

### Typography

- `SKILL.md:108-117`
- `client/src/index.css:47-69`
- `client/src/components/ui/Typography.tsx:5-31`
- `client/src/components/ui/Typography.tsx:74-120`
- `client/src/pages/tenant/SkillSquare.tsx:350-355`
- `client/src/pages/tenant/ModelQuota.tsx:316-327`

### Button / Input / Dialog

- `client/src/components/ui/button.tsx:163-260`
- `client/src/pages/tenant/ChatView.tsx:2857`
- `client/src/pages/tenant/MyOpenClaw.tsx:711`
- `client/src/pages/tenant/OpenClawDetailGuide.tsx:684-686`
- `client/src/components/ui/input.tsx:15-20`
- `client/src/components/ui/input.tsx:65-71`
- `client/src/components/ui/dialog.tsx:92-102`
- `client/src/pages/admin/OpenClawMonitor.tsx:2682-2683`
