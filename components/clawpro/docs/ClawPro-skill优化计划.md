# ClawPro Skill 优化计划

> 状态：待团队评审 · 起草 2026-06-26 · 本版为"系统性升级版"
>
> **目标（升级后）**：不再靠零散经验单点对照组件，而是**从本源建立 skill 的自体检能力**——把已知痛点升维成可枚举的「问题模式分类」，建立覆盖全部资产层的「系统性审计方法」，并沉淀一套可复用的「检查与修复原则」。使**未遇到过的同类问题也能被主动发现、被脚本拦截、被记录上报**，而不是等踩坑后再补。

---

## 0. 背景与本次升级动机

**背景输入（前几轮对齐）**
1. 组件源码存在重复，83+ 不是真实需要数量；spec（37 份）少于组件总数可能合理，但**不等于整理到位**。客观事实：存在多份文件对不上、skill 找组件参考"找不对 / 找不到"。
2. 图标资源已治理：9 类组件槽位禁止回退 lucide（依据 `docs/ClawPro资源库-阶段9决策溯源(ADR).md`）。但实际仍出现"用 skill 优化样式时仍用 lucide / 图标用错"。
3. 样式优化常"只改外层不改子元素"，例：表格容器改了，但表格内字体仍 14px，没落到组件规范的 12px。

**本次升级动机**
> 上述都是**单点经验**。靠单点对照组件，覆盖不全、永远漏。真正要解决的是：**为什么 skill 会反复产出与预期不符的结果？根在哪？用什么方法能把同类问题一次性扫出来、并防止再发生？**

---

## 1. 本源诊断：skill 是「多层文档系统」，缺单一事实源与层间一致性契约

skill 实际由 **8 个资产层**组成，各层都在描述同一套设计事实：

| 层 | 内容 | 角色 |
|---|---|---|
| `SKILL.md` | 主文 / 总规则 / workflow / self-audit | AI 的主入口与引导 |
| `component-specs/` (37) | 单组件规范（字号、间距、色、结构） | 组件级事实源 |
| `references/` (9) | admin / tenant / foundation / assets-icons / **conflict-log** / migration-map / page-recipes 等 | 场景与跨域规则 |
| `tokens/` | design-tokens.json / radius-shadow | 设计令牌事实源 |
| `portable/` (37 tsx) | 可移植实现参考 | 代码级事实源 |
| `scripts/` | `check-design-usage.mjs` 等 | 机检 |
| `docs/` (12) | INDEX / COMMON-ERRORS / **DESIGN-AUDIT-PLAYBOOK** / EVALUATION / QA-checklist 等 | 索引与方法 |
| `MANIFEST.json` | 资产索引 | 索引层 |

**本源结论**：问题不在"某个组件写错了"，而在 **层与层之间没有约定"谁是事实源 + 如何保持一致"**。于是必然出现：
- 同一事实在多层各写各的、且互相打架（→ AI 读到哪层就听哪层）；
- 主文的引导和已建好的治理机制脱节（→ 治理白做，AI 绕过）；
- 索引层和实体层对不上（→ 找不对 / 找不到）；
- 大量规则只写在文档里、没有脚本兜底（→ 违规一路放行）。

> 三个痛点（spec 对不上、用 lucide、表格 14px）都是这一本源缺陷在不同层的**表现**，不是孤立 bug。

---

## 2. 问题模式分类（Failure Modes）— 把单点痛点升维成可枚举的类

> 用途：以后任何"与预期不符"的现象，先归到下面某一类，就能用该类的「全量发现手法」（见 §4）一次性扫出同类，而不是逐个对照。每类都附**已核实实例**作为证据与验收用例。

| 编号 | 问题模式 | 本源机制 | 已核实实例（证据） |
|---|---|---|---|
| **M1** | 同一事实多层冲突 | 无单一事实源，多层各写 | 表格字号：`table.md`=12px vs `typography.md` 把 `BodyText`(14px)/`InlineNumber`(14px) 用途标为"表格内容/数字" ➜ AI 选到 14px。图标登记源：`SKILL.md §2.8` 指 `example.json`(已降格) vs ADR 决策应指 `resource-skill-map` |
| **M2** | 主文引导与治理机制脱节 | 主文没把 AI 导向已建机制 | `SKILL.md §9.2` 一张「页面→lucide」16 项映射表诱导直接配 lucide，绕过 ADR 的 9 槽位 / 不可回退判断；§3 Workflow 无"选图查槽位"步；§8 Self-Audit 无"图标槽位/子元素贯彻"项 |
| **M3** | 索引与实体脱节 | 索引不随实体校验 | `MANIFEST.json` 36 条 vs `component-specs/` 37 份；spec↔组件 alias 混淆（`combobox.md` 自述 alias、`data-table.md` vs `table.tsx`）；有组件无 spec（`tree-select`/`stepper`/`line-tabs`） |
| **M4** | 局部优化不贯彻子元素 | 只改容器、不回贯内部 | 表格优化只动圆角/边框/表头底色，漏改单元格字号 / 行高 / 颜色 token，字号停在 14px |
| **M5** | 规则无机检 / 脚本盲区 | 规则只在文档、脚本不覆盖 | `check-design-usage.mjs` 只查 5 条正则 + 未登记 SVG；不拦"9 槽位用 lucide"、不拦"表格内 14px"、不拦硬编码 `#1447E6`、emoji 跨行/符号区漏检 |
| **M6** | 拿不准无统一出口 / 无记录 | 缺"上报设计"的落点 | `needs-design-confirmation` 规则要求标，但没规定标到哪、无产出清单 ➜ 无法汇报设计（已有 `references/conflict-log.md` 可能是出口，待核对复用） |
| **M7** | skill 自身内部硬编码违规 | 自身违反自身规则 | `§9.3` 图标色直接写 `text-gray-600/green-600`、`§19` inline boxShadow，违反 §8「禁硬编码颜色 / 阴影走 token」 |

---

## 3. 检查与修复原则（可复用，覆盖未来同类问题）

> 这是本计划的核心沉淀。任何修复都按这 7 条原则裁决；以后新问题也用它判断对错。最终写进 `SKILL.md`。

- **P1 单一事实源（SSOT）**：每类事实只有一个权威层；其他层引用、不复写。字号/间距/色 → `tokens` + 组件级 spec；图标登记 → `resource-skill-map`；资产索引 → `MANIFEST`。
- **P2 组件级 > 通用，近场 > 远场**：同一对象口径冲突时，组件级 spec（`table.md`）覆盖通用 spec（`typography.md`）。
- **P3 改容器必贯彻子元素**：任何样式优化，改了容器就必须回头核对内部字号 / 行高 / 颜色 token 是否同步到位，不留外层与内部脱节。
- **P4 引导即机制**：主文 / Workflow / Self-Audit 必须显式指向已建治理机制（9 槽位、不可回退、token），不得给出绕过机制的"捷径表"。
- **P5 机检优先**：凡能用脚本判定的规则，必须有脚本兜底；文档规则与脚本规则保持一一对应，避免"只写不查"。
- **P6 冲突 / 缺失回设计师 + 强制留痕**：现状真相=组件源码/inventory，意图真相=设计师；AI 不私自裁决，拿不准统一写入登记清单上报。
- **P7 索引随实体**：`MANIFEST` / spec↔组件映射必须有一致性校验脚本，实体增删即报索引漂移。

---

## 4. 系统性审计方法（替代"单点对照"）

### 4.1 层间一致性矩阵（逐格扫，而非等 case）
把 8 层两两之间"应满足的一致性契约"列成矩阵，逐格审计。重点格子：

| 契约（左→右） | 应保证 | 对应模式 |
|---|---|---|
| 主文 → specs/ADR | 主文引导不与组件级/治理机制冲突 | M1/M2 |
| specs ↔ specs | 同一对象多 spec 口径一致（或有优先级） | M1 |
| specs/主文 → tokens | 不出现硬编码、全部引 token | M7 |
| 文档规则 ↔ scripts | 每条可机检规则都有脚本 | M5 |
| MANIFEST ↔ 实体目录 | 数量/命名/alias 一致 | M3 |
| spec ↔ portable 实现 | 规范与示例代码口径一致 | M1/M4 |

### 4.2 各模式的"全量发现手法"（一次扫出同类）
- **M1**：枚举"被多层描述的对象"（字号、间距、色、图标、圆角），对每个对象抽取各层取值做 diff。
- **M2**：扫主文所有"给出具体选型/映射"的段落，逐一核对是否绕过治理机制；核对 Workflow/Self-Audit 是否覆盖治理关键步。
- **M3**：脚本遍历 `component-specs/` vs `MANIFEST` vs 真实组件目录，三方 diff。
- **M4**：定义"容器类组件"清单，检查每个的子元素属性是否在 spec 中明确且与 token 对齐。
- **M5**：把文档里所有"禁止/必须"规则列清单，逐条标"是否已被脚本覆盖"。
- **M6/M7**：全局正则扫硬编码（颜色/阴影/字号）+ 扫 `needs-design-confirmation` 是否有落点。

### 4.3 统一产出：问题登记表
所有审计结果汇入一张表：`问题模式 | 所在层/文件 | 证据 | 建议处置 | 裁决方(设计/负责人/AI可直接修)`。这张表既是修复 backlog，也是给设计的汇报材料。

> 先决动作：A 阶段须**先核对已有机制**（`references/conflict-log.md`、`docs/DESIGN-AUDIT-PLAYBOOK.md`、`docs/COMMON-ERRORS.md`、`docs/EVALUATION.md`、`qa/*-checklist.md`），能复用就接驳，避免重复造轮子。

---

## 5. 优化计划（阶段；严格"先审计后改"）

> 纪律：A 阶段不动 skill 一字；B/C/D 每次改动前先 `git diff`/原文比对，确认不丢原有合理规则；一切以 A 的真实产出为准，不凭印象改。

### 阶段 A · 全量体检（不改 skill，只产出「问题登记表」）
- **A0** 核对并梳理已有机制（conflict-log / DESIGN-AUDIT-PLAYBOOK / COMMON-ERRORS / EVALUATION / qa-checklist），判定复用还是新建。
- **A1** 按 §4.1 矩阵 + §4.2 手法逐格 / 逐模式扫描，所有命中归入 §4.3 登记表（含 M1–M7）。
- **A2** 登记表交设计 / 负责人裁决：哪些是"故意聚合/合理"，哪些是"真冲突/真漏/盲区"，定优先级。

### 阶段 B · 按原则收敛与修复（依据 A 产出 + §3 原则）
- **B1（P1/P2）** 建立 SSOT 与优先级，把多层冲突收敛到单一入口（含图标口径、表格字号、登记源）。
- **B2（P4）** 主文引导对齐机制：§9.2 lucide 映射表降级为"仅未登记 admin-sidebar 槽位的语义占位"并加显著警示；Workflow 增"选图查槽位"步；Self-Audit 增"图标槽位 + 子元素贯彻"项。
- **B3（P3/P7）** 修过时引用（§2.8 example.json→resource-skill-map）、修 typography 表格用途指向、修 §9.3/§19 硬编码。

### 阶段 C · 补机制（让规则可拦截 + 可记录 + 可校验）
- **C1（P5）** 增强 `check-design-usage.mjs`：9 槽位 lucide、表格内 14px/`BodyText`/`InlineNumber`、硬编码 `#1447E6`、emoji 跨行盲区。
- **C2（P6）** 落地"待设计确认清单"出口（优先复用 `conflict-log.md`），AI 拿不准强制追加一条。
- **C3（P7）** spec↔组件↔MANIFEST 一致性校验脚本，解决"找不对/找不到"与索引漂移。

### 阶段 D · 固化原则（写进 `SKILL.md §1/§12`）
把 §3 的 P1–P7 写成 skill 的最高约束；明确 SSOT、组件级>通用、改容器必贯彻子元素、冲突回设计师 + 留痕。

**执行顺序**：A → B/C 并行 → D。

---

## 6. 验收标准

1. 产出覆盖 M1–M7 的「问题登记表」，每条有证据 + 裁决，不再依赖临时经验喂 case。
2. §3 的 P1–P7 原则写入 `SKILL.md`，成为后续判断对错的统一依据。
3. 同一对象多层口径冲突清零或有明确优先级（表格字号确定 12px，无 14px/`BodyText`/`InlineNumber` 残留）。
4. 9 类不可回退槽位零 lucide；主文不再有诱导绕过机制的"捷径表"；脚本能拦住以上违规。
5. 每条可机检规则都有脚本覆盖（文档规则↔脚本一一对应）。
6. 拿不准 / 缺图 / 冲突统一落入登记清单，可直接汇报设计。
7. `MANIFEST` ↔ spec ↔ 组件三方一致，参考可经索引稳定命中。

---

## 附录 A · 阶段 A 审计产出：问题登记表（2026-06-26，5 agent 并行只读体检）

> 执行方式：按 §4 审计矩阵，分 5 个只读 agent 并行扫描（M1 多层冲突 / M2+M7 引导脱节与自身硬编码 / M3 索引脱节 / M4 子元素贯彻 / M5+M6+A0 机检与机制）。全程未改 skill 一字，所有证据挂真实文件行号。
> 裁决方含义：**[设计]** 需设计师拍口径 / **[AI可直修]** 事实性错误或过期引用，确认后 AI 可改 / **[补机制]** 需脚本或索引支撑。

### A.1 M1 同一事实多层冲突（已核实，超出原预期）

> 关键升级：**冲突远不止表格 14px**。整个文字色体系与圆角 token 命名都在打架。

| # | 对象 | 层 A | 层 B | 处置建议 | 裁决方 |
|---|---|---|---|---|---|
| M1-1 | 文字色 primary | `typography.md:27` text-gray-900 **#171717** | `tokens/colors.md:34`/`foundation.md:49` `--text-title` **#0F172A** | 统一为 token slate 系，typography 改引用不复写 | [设计] |
| M1-2 | 文字色 body | `typography.md:29` **#171717** | `colors.md:34` `--text-body` **#1E293B** | 同上 | [设计] |
| M1-3 | 文字色 muted（表头色） | `typography.md:31` text-gray-500 **#737373** | `colors.md:36` `--text-muted` **#64748B** | 同上（表头色须唯一） | [设计] |
| M1-4 | 文字色 weak/emphasis/secondary | `typography.md:28/30/32` gray 系 | `colors.md:32/35/37` slate 系 | 同上，整张色表收敛到 tokens | [设计] |
| M1-5 | 表格 body 字号 | `table.md:25/44/311` 全表 **12px** | `typography.md:47/56` `BodyText`/`InlineNumber` **14px**，用途写"表格内容/数字" | 按 P2 组件级优先：表格 12px，typography 用途列移除"表格"指向 | [AI可直修] |
| M1-6 | 表格"紧凑才 12px"歧义 | `typography.md:50` 仅 `MiniBodyText`=12px"紧凑表格" | `table.md:44` 所有表格一律 12px | 明确"全表 12px"为准，消歧义 | [设计] |
| M1-7 | 圆角 4px 的 token 名 | `SKILL.md:59/161/170`+`design-tokens.json:24` = **`radius-lg`** | `radius-shadow.md:9`/`foundation.md:81` = **`radius-md`** | 统一一个名，全仓引用对齐 | [设计] |
| M1-8 | 圆角 radius-sm 取值 | `radius-shadow.md:8` **3px** | `design-tokens.json:22` **2px**（md=3px） | 统一 sm/md/xs 数值与名 | [设计] |
| M1-9 | 图标登记源 | `SKILL.md §2.8` 仅指 `assets/icon-registry.example.json`（已降格） | ADR + `assets-icons.md` 以 `resource-skill-map` 为准 | §2.8 改指 resource-skill-map；registry 内残留 `status:approved` 与"已降格"矛盾，同步修 | [AI可直修] |

### A.2 M2 主文引导与机制脱节 / M7 skill 自身硬编码

| # | 模式 | 文件:行 | 证据 | 处置建议 | 裁决方 |
|---|---|---|---|---|---|
| M2-1 | M2 | `SKILL.md:774/778-795` | §9.2 一张 16 行「页面→lucide」捷径表（模型配置→Brain、通道配置→MessageSquare…），表头 776 行虽有免责声明，但表体仍诱导抄表配 lucide，绕过 9 槽位判断 | 映射表降级为"仅未登记 admin-sidebar 槽位语义占位"并强警示，或拆除业务槽位项 | [设计] |
| M2-2 | M2 | `SKILL.md:293-308`（第4步302） | §3 Workflow 选组件步无"判图标槽位/查 resource-skill-map"环节，治理在主流程断链 | Workflow 增"选图查槽位"步 | [AI可直修] |
| M2-3 | M2 | `SKILL.md:491-504` | §8 Self-Audit 8 项无任何图标槽位/违规 lucide/子元素贯彻项 | Self-Audit 增"图标槽位用对 + 子元素贯彻"项 | [AI可直修] |
| M7-1 | M7 | `SKILL.md:797-814`（§9.3） | 图标色示例 4 行全硬编码 `text-gray-600/green-600/red-600/yellow-600`，违反 §8(495) 禁 text-gray-*、§1原则3(105) 禁硬编码 | 改走语义色 token | [AI可直修] |
| M7-2 | M7 | `SKILL.md:963`（§19） | checklist 要求"统一 boxShadow（inline style）"，违反 `foundation.md §7:101` 禁未带 allow-shadow 的 inline boxShadow | 改为走阴影 token | [AI可直修] |
| M7-3 | M7 | `SKILL.md:897/923-924/932-946`（§18.3-18.5） | 危险按钮手写 `bg-red-600`、骨架屏 `bg-gray-200`、横幅 `bg-blue-50/amber-50/red-50`+硬编码色阶，违反 §5 决策表(343/345/348)"用 destructive/Skeleton/Alert" | 反馈示例改用组件，去硬编码 | [AI可直修] |
| M7-4 | M7 | `SKILL.md:604-605/643`（§15 图表） | 轴标签 `fill:"#64748B"`、数据标签 `fill:'#0F172A'` 硬编码 hex | 走 token | [AI可直修] |

> 说明：§2.1/§7.2 等处的 hex 为"❌反例示范"、foundation 等处 `#1447E6`/`bg-gray-50` 为 token 定义/已拍板例外，**不计违规**（agent 已甄别）。

### A.3 M3 索引与实体脱节（三方对账：specs 37 / MANIFEST 36 / 组件源码 83）

| # | 问题项 | 证据 | 处置建议 | 裁决方 |
|---|---|---|---|---|
| M3-1 | `datetime-display.md` 未登记 MANIFEST | 36 vs 37 的**唯一缺口**：`MANIFEST.json` componentSpecs[] 搜 datetime-display 命中 0，文件实存 | 补登记进 MANIFEST | [AI可直修] |
| M3-2 | spec↔实现命名错位（alias） | `card-surface.md`→`card.tsx`；`combobox.md`→无实现(实为 SearchableSelect)；`data-table.md`→`table.tsx`；`badge.md`→`badges.tsx`；`popover-dropdown-menu.md`→多组件；`tabs.md`→实为 `line-tabs.tsx` | 建机器可读 alias 映射表（C3） | [补机制] |
| M3-3 | 有组件无 spec | `tree-select.tsx`/`stepper.tsx`/`line-tabs.tsx` 等，83 组件仅 ~37 有 spec | 列"真漏 vs 故意不沉淀"清单交裁决 | [设计] |
| M3-4 | spec 冗余/承诺缺口 | `file-browser.md`+`upload-file-browser.md` 共指一组件；`transfer.md:18` TreeTransfer"暂未沉淀" | 合并冗余 spec / 标注缺口 | [设计] |

### A.4 M4 局部优化不贯彻子元素 / spec↔实现口径不一致

| # | 容器 | 证据 | 处置建议 | 裁决方 |
|---|---|---|---|---|
| M4-1 | **盲区集中：card-surface / dialog-drawer / form-controls / search-filter-bar** | 4 个 spec 的 Visual Standard 表**只列容器属性（圆角/边框/阴影/高度/间距），不列子元素字号/行高/颜色 token**，留"改容器不改子元素"漏洞 | 4 份 spec 补"子元素字号/色 token"强制条目 | [AI可直修] |
| M4-2 | table（正面样板） | `table.md:25/44` 强制 12px，`table.css:46/63/80` 实现一致——盲区已堵，可作改造范式 | 以 table 模式补齐其它容器 | — |
| M4-3 | spec 颜色注释过期 | `table.md:23/25` 注 muted=#737373/body=#171717，但 `tokens.css:21/23` 实际 #64748B/#1E293B | 修正 spec 注释为实际 token 值（同 M1-3） | [AI可直修] |
| M4-4 | 实现侧硬编码 | `table.css:113-117` 状态色、`input.css`/`search-filter-bar.css:62` 边框 `#355EF1`、图表 hex 等硬编码，spec 要求走 var(--cp-*) | 实现去硬编码走 token | [AI可直修] |

### A.5 M5 机检盲区 / M6 出口 / A0 已有机制（可复用）

| # | 项 | 证据 | 处置建议 | 裁决方 |
|---|---|---|---|---|
| M5-1 | 脚本仅 5 条单行正则 | `check-design-usage.mjs:32-58` 逐行匹配，无作用域概念 | C1 增强为多规则+作用域 | [补机制] |
| M5-2 | `#1447E6` 漏检 | `old-brand-color` 正则只含旧色 `#007AFF/#5856D6`，**不含现行 #1447E6**、不含 `text-gray-*`/`bg-yellow-50` | 补硬编码现行色正则 | [补机制] |
| M5-3 | 9 槽位 lucide 漏检 | 脚本无槽位概念；`check:skill-map` 等专项**不在本 skill 5 脚本内、不可移植** | 把槽位校验纳入可移植脚本 | [补机制] |
| M5-4 | 表格 14px 漏检 | 无表格作用域识别；且 `table.md` 连显式 `14px/text-sm` 反例文本都没有 | 脚本加表格作用域规则 | [补机制] |
| M5-5 | emoji 漏检 | 正则 `1F300-1FAFF` **漏 ✅(2705)❌(274C)⚠️(26A0)** 及 §9.2 表内 ⚙️✏️⬇️；单行匹配漏跨行 | 扩 emoji 字符区+跨行 | [补机制] |
| M6-1 | 待确认无汇总出口 | 出口文件已有（`conflict-log.md`，`SKILL.md:107/573` 指定），但**无"本轮 needs-design-confirmation 汇总清单"、无脚本扫描汇总**；与 registry `pending-design-review` 双出口未统一 | C2 复用 conflict-log + 补汇总脚本，统一双出口 | [补机制] |
| A0-1 | 已有机制可复用，无需新建 | `conflict-log.md`/`DESIGN-AUDIT-PLAYBOOK.md`/`COMMON-ERRORS.md`/`EVALUATION.md`/`qa/*-checklist.md` 五类均已存在 | 全部**复用**；各自补：汇总清单 / 章节号 / 与脚本联动 / 修订工具链评分 / 补量化项 | [AI可直修] |
| A0-2 | EVALUATION 工具链评分偏乐观 | `EVALUATION.md` 给工具链 5.0 满分，与本次盲区（M5-2~M5-5）矛盾 | 修订该项评分并标注盲区 | [AI可直修] |

### A.6 裁决汇总（给设计/负责人快速决策）

- **必须设计拍口径（9 项，B 阶段前置）**：M1-1~M1-4 文字色 gray vs slate 体系二选一 → M1-6 表格 12px 歧义 → M1-7/M1-8 圆角 token 命名与数值 → M2-1 §9.2 lucide 表去留 → M3-3/M3-4 有组件无 spec 哪些是真漏。
- **AI 可直修（确认后即改，不涉设计取舍）**：M1-5/M1-9、M2-2/M2-3、M7-1~M7-4、M3-1、M4-1/M4-3/M4-4、A0-1/A0-2。
- **补机制（C 阶段脚本/索引）**：M3-2、M5-1~M5-5、M6-1。

> **当前进度停在此处（确认点）**：以上为阶段 A 全部产出，未改 skill。请评审本表并裁决上述 9 项设计口径、圈定 B/C/D 起点；确认后我再按 P1–P7 动手改 skill，绝不擅自开干。

---

## 附录 B · 优化工作日志（交接 / 续接专用）

> **这一章的用途**：本计划的「单一交接入口」。任何人（包括新开对话的 AI）想接手这件事，**只看这一章就能知道：做到哪了、谁拍了什么板、改了哪些文件、下一步从哪起、铁律是什么**。
> **维护约定**：每完成一个动作（一次审计 / 一批改 skill / 一次设计裁决）就在 §B.3 追加一条流水，并同步更新 §B.1 当前状态。**先记录、后动手**。

### B.0 一句话现状

> **截至 2026-06-26 18:15**：阶段 A 已完成、**D1–D5 全部裁决**；**阶段 B 已全部完成**。已完成：**B1（D1+D2+D3）**、**StatusTag 16:50 卡点关闭**（组件不动、12px 为应用层）、**SKILL.md 主文批次**（D4 §9.2 lucide 表降级 + M2-2/M2-3 + M7-1~4 自身硬编码 + M1-9 登记源指向；M1-9 status:approved 经核实被脚本消费→驳回不改）、**M3-1**（datetime-display 补进 MANIFEST）、**M4-1/M4-3**（4 份容器 spec 补子元素 token + table.md 过期注释纠正）、**D5**（新建 `stepper.md`/`tree-select.md` 两份 spec、`upload-file-browser.md` 去冗余、`transfer.md` TreeTransfer 缺口标注；全量同步索引）、**M4-4**（实现层 `#355EF1` 去硬编码——新增 `--cp-control-accent` token 承载「控件交互蓝」语义，`input.css`/`search-filter-bar.css`/3 个 html demo 裸 hex 全 token 化，保真零视觉变化）。**B 阶段已清零**。**剩余工作**：进 **C**（机检脚本增强 M3-2/M5-1~5/M6-1）/ **D**（P1–P7 固化进 SKILL.md），另有 **tree-select L68 / selection-controls L20 三种蓝口径文字冲突**单列待设计裁决（见 §B.2 D6）。改前逐处 `git diff`/原文比对、改后追 §B.3 流水。
>
> **【2026-06-30 重大更新，覆盖上方旧口径】** 经核实：同事 **addietang** 已在集成分支落地 **C 阶段**（`check-design-usage.mjs` 7 检查函数 + 4 新脚本 + `icon-design-todo.md`/`component-mapping.md` 产出）与 **D 阶段**（`SKILL.md` §1.5/§12 冲突铁律），上文「剩余进 C/D」口径**已过时**；本计划 B 成果与同事 C/D **已在集成分支融合共存、无丢失**。本次仅修复 verify 报红（13 文档补登记进 `MANIFEST.rootFiles`，FAIL→PASS）。遗留：MANIFEST `scripts`/`references` 段部分新文件待登记；C/D 完整度待专项验收。详见 §B.3 2026-06-30 流水。

### B.1 阶段进度看板

| 阶段 | 内容 | 状态 | 产出落点 |
|---|---|---|---|
| **A · 全量体检** | 5 个只读 agent 并行扫 M1–M7，产出问题登记表 | ✅ 已完成 | 本文档 附录 A |
| **确认点** | 设计/负责人裁决 §A.6 的 9 项设计口径 + 圈定 B/C/D 起点 | ✅ 已完成（D1–D5 见 §B.2） | 见 §B.2 待裁决清单 |
| **B · 收敛修复** | 按 P1–P7 改 skill（先 AI 可直修，再设计拍板项） | ✅ 已完成 | skill 各层文件 |
| **C · 补机制** | 增强 `check-design-usage.mjs`、补 alias 映射、统一待确认出口 | 🟡 **同事 addietang 已落地**（7 检查函数 + 4 脚本 + 产出物；实现路径与本计划设计有差异，待验收）见 §B.3 2026-06-30 | `scripts/`、`MANIFEST.json` 等 |
| **D · 固化** | 把 P1–P7 原则写进 `SKILL.md` | 🟡 **同事已落地冲突铁律**（`SKILL.md` §1.5/§12）；P1–P7 是否逐条固化待核 | `SKILL.md` |

### B.2 待裁决清单（卡点 → 设计/负责人拍板，B 阶段前置）

> 拍完板请直接在本表「裁决结论」列填写，并在 §B.3 追一条日志。

| # | 待决问题 | 选项 | 裁决结论（待填） |
|---|---|---|---|
| D1 | 文字色体系统一走哪套？（影响全仓，对应 M1-1~M1-4） | A. gray 系（#171717…） / B. slate 系（#0F172A…，token 现状） | **B（slate 系）**✅2026-06-26 ｜ typography 改"引用 token、不复写"，整张文字色表收敛到 tokens slate 系 |
| D2 | 表格正文字号口径（M1-5/M1-6） | A. 全表一律 12px / B. 仅紧凑表格 12px | **A（全表一律 12px）**✅2026-06-26 ｜ typography 的 `BodyText/InlineNumber` 用途列移除"表格"指向，消歧义 |
| D3 | 圆角 token 命名与数值（M1-7/M1-8） | 4px 叫 `radius-lg` 还是 `radius-md`；`radius-sm`=2px 还是 3px | **方案①（纯文档层·不碰运行时）**✅2026-06-26 ｜ ①**设计语义 token** 维持 `radius-xs=2px / radius-sm=3px / radius-md=4px`（foundation/radius-shadow 现状已对，无需改）；②**运行时 CSS 变量**保持不动（`--radius-sm=2px / --radius-md=3px / --radius-lg=--radius-xl=4px / --radius-card=12px`，生成物禁手改）；③**关键纠偏**：SKILL.md 里 `var(--radius-lg)`=4px 是**正确写法不得改成 `var(--radius-md)`**（否则渲染 3px）；④**2px 数值保障**（statustag 类诉求）：2px 的唯一正确代码写法是 **`var(--radius-sm)`**——运行时无 `--radius-xs` 变量，禁止按语义名写 `var(--radius-xs)`；**【2026-06-26 16:50 纠错】**StatusTag **并非无圆角**——真实组件 `status-tag.tsx` 中 `mode="soft"` 用 `rounded-[4px]`、`fill`/`preset` 用 `rounded-full`，仅 `text` 模式无圆角；前一版误信 spec 写成「StatusTag 不消费圆角」已删除。2px 的真实消费者是小 Badge/状态徽章；⑤**分端沿用现状**（admin 控件 4px / tenant 业务卡 12px），不新增端级圆角 token。skill 仅在 radius-shadow/foundation/SKILL.md 补「语义↔CSS 变量」对照，不动任何能正确渲染的引用 |
| D4 | `SKILL.md §9.2` 那张「页面→lucide」16 项捷径表（M2-1） | A. 拆除业务槽位项 / B. 降级为占位+强警示 / C. 保留 | **A + 警示注脚（混合）**✅2026-06-26 ｜ 拆除 11 项 `admin-sidebar` 槽位业务图标（改为"先查 resource-skill-map 候选"指引）、删中间列 emoji、保留通用动作图标（下载/删除/编辑/搜索/刷新）、表头加强警示 |
| D5 | 「有组件无 spec」哪些是真漏需补、哪些故意不沉淀（M3-3/M3-4） | 逐项圈定（tree-select / stepper / line-tabs …） | **同意建议**✅2026-06-26 ｜ ①真漏补 spec：`stepper`、`tree-select` 两份；②`line-tabs` 归 C3 alias（`tabs.md` 已覆盖）不另写；③合并冗余 `file-browser.md`+`upload-file-browser.md`、标注 `transfer.md` 的 TreeTransfer 缺口；④shadcn/radix 基线件统一标注"采用 shadcn 基线、无 ClawPro 专属规范"，不逐个补 spec |
| D6 | **「三种蓝」口径文字冲突**（M4-4 副产品，spec 内部自相矛盾） | A. 统一品牌蓝（`tree-select.md:68` 立场：三种蓝并存是不一致，方向品牌蓝 `#1447E6`） / B. 控件交互蓝独立成色（`selection-controls.md:20` 立场：`#1447E6` 边框 + `#355EF1` 填充是合法既定语义） | **待裁**｜M4-4 已用 `--cp-control-accent` 实现层承载 B 立场（保真）；但 `tree-select.md:68` 仍把 `#355EF1` 列为「待统一为品牌蓝」的不一致项，二者文字层面冲突。**不阻塞**（实现已落地），但 spec 文字需择一：A=接受三蓝、把 `#355EF1` 正名为控件交互蓝并改 tree-select 措辞；B=维持「应统一品牌蓝」、把 `--cp-control-accent` 标为过渡 token。建议 A（与已落地实现一致）|

### B.3 流水日志（倒序，最新在上）

| 时间 | 执行方 | 动作 | 影响文件 | 结果 / 备注 |
|---|---|---|---|---|
| 2026-06-30 | AI（核实 + 收尾） | **重大状态核实 + verify 红灯修复**：①经 git 比对，本分支 `feature/design-miekoyychen-refresh` 与目标分支 `feature/design-integration-2026` 的 skill 目录 **243 文件完全一致**，成果零丢失；②**核实发现 C/D 实为「已落地」非「未开始」**——同事 **addietang** 已在集成分支完成 C 阶段（`check-design-usage.mjs` 增强为 7 检查函数含 9 槽位禁 lucide + 新增 `generate-design-todo`/`generate-icon-todo`/`generate-component-mapping`/`verify-manifest` 4 脚本 + `icon-design-todo.md`/`component-mapping.md` 产出）与 D 阶段（`SKILL.md` §1.5 冲突铁律 + §12 完整版），本台账「C/D 未开始」口径**已过时**；本计划 B 成果（40 specs 含 stepper/tree-select、token 化）与同事 C/D **已在集成分支融合共存**，非二选一；③修复 `verify-portable-skill.mjs` 报红——13 个阶段 C 交付文档未登记，已补进 `MANIFEST.rootFiles`（3→16），校验 FAIL→PASS | `…/MANIFEST.json` | 仅补登记、零删除、**未动任何 skill 规则与图标口径**；**遗留**：MANIFEST `scripts` 段（4 新脚本）/`references` 段（`component-mapping.md`/`icon-design-todo.md`）尚未登记（verify 不强校验该两段故未报红）；C/D 实现路径与原计划设计有差异，完整度/质量**待专项验收**；同事 `FINAL_COMPLETION_SUMMARY_FULL.md` 自述 B（仅改 2 文件）与本台账 B（多文件）为两套并行叙述，实体已合并，**以当前 skill 实体为准** |
| 2026-06-26 18:15 | AI | **M4-4 完成（实现层 `#355EF1` 去硬编码，[AI可直修]）**：先精确盘点 skill 全量 `#355EF1` 分布并做**语义分层**（铁律不机械替换）：①实现层裸 hex → token；②注释/文档表格的 hex 标注 → 保留；③现状描述/迁移映射/Do-Don't 反例 → 保留原文。**关键澄清**：`#355EF1` 非品牌蓝 `#1447E6` 误用，而是 input/select hover·focus 边框、选中态文字/勾号、checkbox/radio/switch 填充、focus ring 的「控件交互蓝」既有设计语义，故 `tokens.css` **新增 `--cp-control-accent: #355EF1`（带语义注释）** 专门承载，而非并入品牌蓝。据此：`input.css`(9 处)+`search-filter-bar.css`(3 处)+3 个 html demo（`input-select-table` 5 / `admin-control-page` 5 / `admin-list-page` 4，含 checkbox 填充与 status-filter 激活态）实现层裸 hex 全部 → `var(--cp-control-accent)`，保真零视觉变化。react/segment.tsx 仅注释含 hex（保留），无实际裸 hex 样式 | `…/portable/css/{tokens,input,search-filter-bar}.css`、`…/portable/html-css/{input-select-table,admin-control-page,admin-list-page}.html` | 改前逐处盘点行号 + 原文比对；两个 admin demo 经核 `:root` 本就悬空引用 `var(--cp-brand-blue/font-sans)`（依赖外部注入、非独立自洽），故改 token 同口径同保真。实现层裸 hex 清零，残留 8 处全为注释/文档标注；CSS/HTML lint=0。**B 阶段全部完成**，下一步进 C（机检）/D（固化） |
| 2026-06-26 17:40 | AI | **D5 批次（补 spec + 去冗余 + 标缺口，全 [设计] 已裁项）**：①**D5①新建 `stepper.md`**——值全取自 `client/src/components/ui/stepper.tsx`（24px 圆圈 / `ChevronRight` 分隔 / `current` 自动推 completed·active·pending）；组件硬编码 `bg-blue-500`/`gray-*` 与 token 偏差**如实列「§3.4 对齐缺口」**、不改组件；②**D5①新建 `tree-select.md`**——button + filter-icon 两变体，值取自 `_internal/TreeSelectFilter.tsx`+`TableHeaderTreeFilter.tsx`（触发器/面板/节点行/footer/commitMode），**三种蓝并存（`blue-500`/`#355EF1`/品牌 `#1447E6`）+ 6px 行圆角**如实标对齐缺口；③两份均补登记 `MANIFEST.json`（37→39）+ 补进 `SKILL.md` 两处决策表 + `INDEX/DEVELOPER-USAGE/STRUCTURE/STATUS`（顺手补回 INDEX 漏列的 `datetime-display`）；④**D5③去冗余**：`upload-file-browser.md` 加顶部职责边界声明、收敛为纯上传（Anatomy `FileBrowser`→`UploadedFileList`、§6/§11 去 `file-browser.tsx` 误指、双向交叉引用 `file-browser.md`）——**未改名**（遵 `conflict-log` 既定「改名留单独 PR」）；⑤**D5③标缺口**：`transfer.md §2` `TreeTransfer` 死注释强化为可执行出口（标 `needs-design-confirmation`+记 conflict-log+禁误用 TreeSelect 单选），`conflict-log` 同步登记 TreeTransfer(C类)、移出已补的 tree-select(D类) | `…/component-specs/{stepper,tree-select,upload-file-browser,transfer}.md`、`…/MANIFEST.json`、`…/SKILL.md`、`…/references/conflict-log.md`、`…/docs/{INDEX,DEVELOPER-USAGE,STRUCTURE,STATUS}.md` | 改前逐处读真实组件/原文比对；**组件代码零改动**；md/json lint=0；`verify-portable-skill.mjs` 通过（无 missing/unlisted）。`file-browser.md`↔`upload` 经核 `conflict-log` 判为职责正交→去重而非删并 |
| 2026-06-26 17:25 | AI | **M3-1 + M4 批次（索引 / 容器子元素，全 [AI可直修]）**：①**M3-1** `datetime-display.md` 补登记进 `MANIFEST.json` componentSpecs[]（按字母序插 date-picker 与 dialog-drawer 之间），MANIFEST↔spec 缺口补齐（36→37）；②**M4-3** `table.md` §3.1 过期颜色注释 `#737373`/`#171717`→实际 token 值 `#64748B`/`#1E293B`；③**M4-1** 以 `table.md`（M4-2 正面样板）为范式，给 4 份容器 spec（`card-surface` / `dialog-drawer` / `form-controls` / `search-filter-bar`）各补「§3.1 子元素字号 / 色 token（改容器必贯彻 P3）」强制条目——值全部取自各 spec 自身 fallback 代码与 `colors.md`，未杜撰 | `…/MANIFEST.json`、`…/component-specs/{table,card-surface,dialog-drawer,form-controls,search-filter-bar}.md` | 改前逐处 read_file 比对原文；JSON / md lint=0。**M4-4（实现侧 CSS 去硬编码 `#355EF1` 等）未做**：属 portable/css 实现层、且需先核 token 映射是否存在，留作单独评估 |①**D4/M2-1** §9.2 lucide 表降级——拆 11 项业务/侧栏槽位、删中间列 emoji、仅保留 5 个通用动作图标、表头加强警示并指向 `assets-icons.md §5.5` resource-skill-map；②**M2-2** §3 Workflow 第4步并入「选图查槽位（命中禁回退 lucide）」+ 交付物标注图标来源；③**M2-3** §8 Self-Audit 增「图标槽位用对」「改容器必贯彻子元素(P3)」两项；④**M7-1** §9.3 图标色 `text-gray/green/red/yellow-600`→`--cp-text-muted/success/danger`+`--text-warning`（cp 集合无 warning 文本变量，已核实用 `--text-warning`）；⑤**M7-4** §7 图表 `fill:#64748B/#0F172A`→`var(--cp-text-muted/title)`（对齐 skill 自身 L614 已用 var 约定）；⑥**M7-3** §10.3 删除按钮→`--cp-text-danger`（对齐 dialog-drawer.md）、§10.4 骨架屏→`Skeleton` 组件、§10.5 横幅→`Alert variant`（严格对齐 alert.md）；⑦**M7-2** §19 checklist「inline boxShadow」→阴影 token；⑧**M1-9 文档部分** §2.8/§9/Reference 表图标登记源补「当前项目以 resource-skill-map.json 为准、跨仓参照 example.json 样例」 | `…/SKILL.md` | 改前逐处 read_file 比对原文；组件代码零改动；lint=0。Skeleton/Alert API 均核对真实组件 spec |
| 2026-06-26 17:25 | AI（核实驳回） | **M1-9 第二子项「修 registry 残留 status:approved」驳回**：审计认为 `icon-registry.example.json` 每条 `status:approved` 与「已降格」矛盾。核 `client/src/design-assets/scripts/classify-resources.mjs` L77/L235/L285——该 `status` 字段**被构建脚本实际消费**（作分类 status/category 提示输入，脚本注释明确「非候选准入闸门」）。「降格」降的是**准入闸门角色**而非该字段，二者不矛盾；删字段会破坏 `build:skill-map` 分类。按铁律2 保留不动。 | （未改文件，仅核实） | M1-9 仅文档指向部分有效（已做）；JSON status 字段不动 |①方向定 **A（以真实组件为准、收敛 spec）——StatusTag 组件一字不动**（`text`=`text-sm`/14px、`soft`=`rounded-[4px]`、`fill`/`preset`=`rounded-full` 全保持）；②查实表格状态列「12px」**非组件字号**，而是应用层用 `className="text-xs"` 压出来的（实例 `NodeContentPanel.tsx`）；③立规：**字号 `text-xs` 覆盖=允许**（表格密度对齐）、**形态 `bg/border/rounded` 覆盖=禁止**；④另查实组件不传 `mode` 时默认渲染 `fill` 胶囊（L220），表格要纯文字须显式 `mode="text"`——如实写明、不改组件；⑤据此精修 `status-tag.md`（§3.1/§3.5新增/§4/§5/§8 Demo/§10/§11/§12/§14.1），清除"组件=12px"误导措辞 | `…/component-specs/status-tag.md` | 改前 git/原文比对；**组件代码零改动**；lint=0。16:50 卡点关闭，与 D3④ StatusTag 圆角纠错口径一致 | **重大纠错：StatusTag 圆角结论被证伪**。设计以 StatusTag 全模式截图质疑「不消费圆角」结论。比对真实组件 `status-tag.tsx`：`mode="soft"` = `rounded-[4px]`（写死 4px，非 CSS 变量）、`fill`/`preset` = `rounded-full`，仅 `text` 无圆角；demo 页 2393/2825/2826 行真在用 `mode="fill"`。根因＝AI 前一轮**只信 spec `status-tag.md`（已把 fill/soft 误判「已删除」）未核对真实组件**，违反铁律1。已删除 `radius-shadow.md §1.1` / `foundation.md §6` / 本文档 §B.2 D3④ / §B.3 三处错误表述。**遗留方向待裁**：spec 与真实组件三方不一致（spec 称唯一 text 形态 vs 组件 soft/fill 在用），需定 A（以组件为准改 spec）/B（以 spec 为准收敛组件）。 | `…/tokens/radius-shadow.md`、`…/references/foundation.md`、`docs/…优化计划.md` | 错误结论已全部纠正；lint=0。`status-tag.md` 与真实组件的对齐方向待设计裁决后再动 |
| 2026-06-26 16:40 | AI | **B1 完成（D3 文档对齐执行）**：①`radius-shadow.md` §1 加「设计语义↔运行时 CSS 变量」对照(§1.1)+2px/4px 红线；②`foundation.md §6` 表补 CSS 变量列+红线，引导 §1.1；③`SKILL.md §2.4` 铁律下补对照引导，`var(--radius-lg)` 等正确引用**一律未动** | `…/tokens/radius-shadow.md`、`…/references/foundation.md`、`…/SKILL.md` | 改前已 git/原文比对：foundation/radius-shadow 语义命名本就正确，仅缺与运行时对照→补；运行时变量(生成物)零改动。4 文件 lint=0。**B1 全部完成（D1+D2+D3）** |
| 2026-06-26 16:35 | 设计/负责人 + AI | **D3 终裁（方案①）+ 关键纠偏**：圆角走纯文档层；查实运行时与设计语义是两套命名且数值错位（运行时 `--radius-sm=2px/md=3px/lg=4px`，语义 `xs=2px/sm=3px/md=4px`）。**纠正上一轮口头方案①的危险建议**——SKILL.md `var(--radius-lg)`=4px 是对的，不得改成 `var(--radius-md)`（会渲染 3px）。回填 §B.2 D3。 | `docs/…优化计划.md`（仅 docs） | 2px 唯一正确写法 `var(--radius-sm)`；运行时无 `--radius-xs`；~~StatusTag 已无圆角~~（**此结论后续证伪**，见 16:50 纠错行）2px 实属小 Badge/状态徽章；分端沿用现状。下一步只在 radius-shadow/foundation/SKILL.md 补对照、不动可正确渲染的引用 |
| 2026-06-26 16:25 | AI | **B1 部分执行（D1+D2）**：`component-specs/typography.md` §4 文字色表 gray→引用 slate `--text-*`（改引用不复写）；§5/§8 移除 14px 组件的「表格」指向、表格正文/数字统一指向 12px `MiniBodyText`、消「紧凑才 12px」歧义 | `…/component-specs/typography.md` | D1=运行时 Typography.tsx/colors.md/foundation.md/tokens-typography.md 早已是 slate，仅此表过期，已对齐。**D3（圆角）暂缓**：发现与运行时存在命名冲突，需先回设计确认，未动任何 radius |
| 2026-06-26 16:10 | 设计/负责人 + AI | **确认点通过**：裁决 §B.2 的 D1–D5 五项设计口径，回填结论、看板转入阶段 B | `docs/ClawPro-skill优化计划.md`（仅 docs） | D1=slate / D2=全表12px / D3=圆角分端(admin小·client大,4px=radius-md,sm=2px) / D4=拆业务项+警示 / D5=补 stepper+tree-select 等。未碰 skill；下一步进 B 阶段，改前逐处 git diff |
| 2026-06-26 15:48 | AI | 新增 §B.6 续作 prompt（断点续作 / 接手对齐用） | `docs/ClawPro-skill优化计划.md`（仅 docs） | 未碰 skill；含安全续作 + B 阶段执行两个变体 |
| 2026-06-26 15:36 | AI | 抽出独立审计报告并转 PDF（供团队同步） | 新增 `docs/ClawPro-skill审计报告-2026-06-26.md` + 同名 `.pdf` | 内容与附录 A 一致；未碰 skill；PDF 用本机 Chrome 无头打印生成 |
| 2026-06-26 15:25 | AI | 建立本「优化工作日志」交接章节 | `docs/ClawPro-skill优化计划.md`（仅 docs） | 未碰 skill；确立"先记录后动手"维护约定 |
| 2026-06-26 | AI（5 只读 agent 并行） | 阶段 A 全量体检：扫 M1–M7，汇总问题登记表 | 仅写 `docs/ClawPro-skill优化计划.md` 附录 A | 未碰 skill 一字；新发现：文字色 gray↔slate 全面冲突、圆角 token 命名冲突、MANIFEST 缺口=`datetime-display`、脚本 `#1447E6` 漏检、A0 五类机制均可复用无需新建 |

### B.4 铁律（接手者必读，不可违反）

1. **审计前不得凭猜测改 skill**：一切改动以阶段 8 组件审计的真实产出（附录 A）为准。
2. **改 skill 前必先比对原文**：用 `git diff` / 原文比对确认 skill 之前怎么写，**不得丢失原有合理规则**。skill 原始立场已是「默认 lucide + 业务/品牌用已登记 registry SVG」，不得擅自推翻。
3. **设计口径未裁决前，不动设计取舍类改动**：§B.2 的 D1–D5 必须先有结论。
4. **先记录、后动手**：每个动作先在 §B.3 留痕，再执行。

### B.5 关键依据文件索引（接手必读）

| 文件 | 作用 |
|---|---|
| `docs/ClawPro-skill优化计划.md`（本文） | 计划全文 + 附录 A 问题登记表 + 附录 B 工作日志（本入口） |
| `docs/ClawPro资源库-阶段9决策溯源(ADR).md` | 9 类组件槽位禁回退 lucide 的决策依据 |
| `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` | 被优化对象主文 |
| `.../references/conflict-log.md` | 待设计确认 / 冲突的既有出口（M6 复用） |
| `.../scripts/check-design-usage.mjs` | 待增强的机检脚本（C 阶段） |
| `.../MANIFEST.json` | 资产索引（M3 三方对账对象） |

### B.6 续作 prompt（新对话 / 同事接手时直接复制粘贴）

> **用途**：新开对话或同事接手时，把下面整段贴给 AI 即可对齐状态、套上铁律，**不会让 AI 乱动 skill**。本身不含待办决策，谁接手都安全。

```text
你接手「ClawPro Skill 设计治理」任务。请严格按以下交接信息续作，不要凭空发挥。

【单一事实源】
docs/ClawPro-skill优化计划.md —— 必须先完整读它的「附录 A 问题登记表」和「附录 B 工作日志」，
那是这件事的全部状态。另有独立审计报告 docs/ClawPro-skill审计报告-2026-06-26.md（内容同附录 A）。

【当前进度】
- 阶段 A（全量只读体检）已完成，问题登记表见附录 A，覆盖 M1–M7。
- skill 至今一字未改，正卡在「确认点」：等设计/负责人裁决 §B.2 的 D1–D5 五项设计口径。
- 阶段 B（改 skill）/ C（补机检脚本）/ D（固化 SKILL.md）均未开始。

【被治理对象】
.codebuddy/skills/clawpro-portable-design-skill/，主文 SKILL.md，
原始立场是「默认 lucide + 业务/品牌用已登记 registry SVG」。

【铁律（不可违反）】
1. 审计前不得凭猜测改 skill，一切以附录 A 真实产出为准。
2. 改 skill 前必先 git diff / 原文比对，确认不丢失原有合理规则，不得擅自推翻原立场。
3. D1–D5 未裁决前，不动设计取舍类改动。
4. 先记录后动手：每个动作先在 §B.3 流水追一条、并同步更新 §B.1 看板，再执行。

【请你先做】
先读上述两个附录，复述当前卡点与下一步，等我确认后再动手，禁止直接改 skill。
```

> **变体**：若 D1–D5 已裁决、要让 AI 直接进入 B 阶段，把【请你先做】换成：
> `D1–D5 裁决结论见 §B.2，请按结论从阶段 B 第 P__ 项开始改 skill，改前先比对原文、改后追 §B.3 流水`。
