# ClawPro Skill 设计治理 · 阶段 A 审计报告

> **版本**：v1.0 · 出具日期 2026-06-26
> **审计对象**：`.codebuddy/skills/clawpro-portable-design-skill`（ClawPro 可移植设计 skill）
> **执行方式**：5 个只读 agent 并行扫描，按问题模式分工互不重叠，全程未改 skill 一字，所有证据挂真实文件行号。
> **当前状态**：阶段 A（全量只读体检）已完成；**skill 未做任何修改**，等待设计/负责人裁决 9 项口径后进入修复阶段。

---

## 一、结论摘要（TL;DR）

本次不是逐个对照组件，而是从本源把 skill 当作「多层文档系统」做全量体检。核心结论：

1. **问题根因是层间缺单一事实源**：同一设计事实（字号/颜色/圆角/图标登记源）被 `SKILL.md`、`component-specs/`、`references/`、`tokens/`、`portable/` 多层各写各的，互相打架，AI 读到哪层就听哪层。三个已知痛点（spec 对不上、用 lucide、表格 14px）都是这一缺陷的不同表现。
2. **冲突范围比预期大得多**：不止"表格字号"，整套**文字色体系**在 `typography.md`（gray 系）与 token（slate 系）两套之间全面冲突；**圆角 token**同一个 4px 一处叫 `radius-lg`、一处叫 `radius-md`。
3. **机检几乎是空的**：脚本只有 5 条单行正则，**连现行品牌色 `#1447E6` 硬编码都拦不住**，9 类图标槽位、表格字号、emoji 等全是盲区。
4. **已有治理机制可复用、无需新建**：conflict-log / PLAYBOOK / COMMON-ERRORS / EVALUATION / QA 清单五类均已存在，补强即可。

**待拍板**：9 项设计口径（见第六节），是进入修复阶段的前置条件。

---

## 二、阶段进度

| 阶段 | 内容 | 状态 |
|---|---|---|
| **A · 全量体检** | 5 个只读 agent 并行扫 M1–M7，产出问题登记表 | ✅ 已完成（本报告） |
| **确认点** | 设计/负责人裁决 9 项设计口径 + 圈定修复起点 | ⏳ 等待裁决（当前卡点） |
| **B · 收敛修复** | 按原则改 skill（先 AI 可直修，再设计拍板项） | ⛔ 未开始（需先过确认点） |
| **C · 补机制** | 增强机检脚本、补 alias 映射、统一待确认出口 | ⛔ 未开始 |
| **D · 固化** | 把检查/修复原则写进 `SKILL.md` | ⛔ 未开始 |

**裁决方含义**：**[设计]** 需设计师拍口径 · **[AI可直修]** 事实性错误或过期引用，确认后可直接改 · **[补机制]** 需脚本或索引支撑。

---

## 三、问题登记表

### 3.1 M1 · 同一事实多层冲突（已核实，超出原预期）

> 关键升级：**冲突远不止表格 14px**，整个文字色体系与圆角 token 命名都在打架。

| # | 对象 | 层 A | 层 B | 处置建议 | 裁决方 |
|---|---|---|---|---|---|
| M1-1 | 文字色 primary | `typography.md:27` text-gray-900 **#171717** | `tokens/colors.md:34`/`foundation.md:49` `--text-title` **#0F172A** | 统一为 token slate 系，typography 改引用不复写 | [设计] |
| M1-2 | 文字色 body | `typography.md:29` **#171717** | `colors.md:34` `--text-body` **#1E293B** | 同上 | [设计] |
| M1-3 | 文字色 muted（表头色） | `typography.md:31` text-gray-500 **#737373** | `colors.md:36` `--text-muted` **#64748B** | 同上（表头色须唯一） | [设计] |
| M1-4 | 文字色 weak/emphasis/secondary | `typography.md:28/30/32` gray 系 | `colors.md:32/35/37` slate 系 | 同上，整张色表收敛到 tokens | [设计] |
| M1-5 | 表格 body 字号 | `table.md:25/44/311` 全表 **12px** | `typography.md:47/56` `BodyText`/`InlineNumber` **14px**，用途写"表格内容/数字" | 表格 12px，typography 用途列移除"表格"指向 | [AI可直修] |
| M1-6 | 表格"紧凑才 12px"歧义 | `typography.md:50` 仅 `MiniBodyText`=12px"紧凑表格" | `table.md:44` 所有表格一律 12px | 明确"全表 12px"为准，消歧义 | [设计] |
| M1-7 | 圆角 4px 的 token 名 | `SKILL.md:59/161/170`+`design-tokens.json:24` = **`radius-lg`** | `radius-shadow.md:9`/`foundation.md:81` = **`radius-md`** | 统一一个名，全仓引用对齐 | [设计] |
| M1-8 | 圆角 radius-sm 取值 | `radius-shadow.md:8` **3px** | `design-tokens.json:22` **2px**（md=3px） | 统一 sm/md/xs 数值与名 | [设计] |
| M1-9 | 图标登记源 | `SKILL.md §2.8` 仅指 `assets/icon-registry.example.json`（已降格） | ADR + `assets-icons.md` 以 `resource-skill-map` 为准 | §2.8 改指 resource-skill-map；同步修 registry 内残留 `status:approved` | [AI可直修] |

### 3.2 M2 · 主文引导与机制脱节 / M7 · skill 自身硬编码

| # | 模式 | 文件:行 | 证据 | 处置建议 | 裁决方 |
|---|---|---|---|---|---|
| M2-1 | M2 | `SKILL.md:774/778-795` | §9.2 一张 16 行「页面→lucide」捷径表（模型配置→Brain、通道配置→MessageSquare…），表头虽有免责声明，表体仍诱导抄表配 lucide，绕过 9 槽位判断 | 降级为"仅未登记 admin-sidebar 槽位语义占位"并强警示，或拆除业务槽位项 | [设计] |
| M2-2 | M2 | `SKILL.md:293-308` | §3 Workflow 选组件步无"判图标槽位/查 resource-skill-map"环节，治理在主流程断链 | Workflow 增"选图查槽位"步 | [AI可直修] |
| M2-3 | M2 | `SKILL.md:491-504` | §8 Self-Audit 8 项无任何图标槽位/违规 lucide/子元素贯彻项 | Self-Audit 增"图标槽位用对 + 子元素贯彻"项 | [AI可直修] |
| M7-1 | M7 | `SKILL.md:797-814`（§9.3） | 图标色示例 4 行全硬编码 `text-gray-600/green-600/red-600/yellow-600`，违反 §8 禁 text-gray-*、§1 原则3 禁硬编码 | 改走语义色 token | [AI可直修] |
| M7-2 | M7 | `SKILL.md:963`（§19） | checklist 要求"统一 boxShadow（inline style）"，违反 `foundation.md §7` 禁未带 allow-shadow 的 inline boxShadow | 改为走阴影 token | [AI可直修] |
| M7-3 | M7 | `SKILL.md:897/923-946`（§18.3-18.5） | 危险按钮手写 `bg-red-600`、骨架屏 `bg-gray-200`、横幅 `bg-blue-50/amber-50/red-50` 硬编码色阶，违反 §5 决策表"用 destructive/Skeleton/Alert" | 反馈示例改用组件，去硬编码 | [AI可直修] |
| M7-4 | M7 | `SKILL.md:604-605/643`（§15 图表） | 轴标签 `fill:"#64748B"`、数据标签 `fill:'#0F172A'` 硬编码 hex | 走 token | [AI可直修] |

> 说明：§2.1/§7.2 等处的 hex 为"❌反例示范"、foundation 等处 `#1447E6`/`bg-gray-50` 为 token 定义/已拍板例外，**不计违规**（agent 已甄别）。

### 3.3 M3 · 索引与实体脱节（三方对账：specs 37 / MANIFEST 36 / 组件源码 83）

| # | 问题项 | 证据 | 处置建议 | 裁决方 |
|---|---|---|---|---|
| M3-1 | `datetime-display.md` 未登记 MANIFEST | 36 vs 37 的**唯一缺口**：`MANIFEST.json` componentSpecs[] 搜 datetime-display 命中 0，文件实存 | 补登记进 MANIFEST | [AI可直修] |
| M3-2 | spec↔实现命名错位（alias） | `card-surface.md`→`card.tsx`；`combobox.md`→无实现(实为 SearchableSelect)；`data-table.md`→`table.tsx`；`badge.md`→`badges.tsx`；`tabs.md`→实为 `line-tabs.tsx` | 建机器可读 alias 映射表 | [补机制] |
| M3-3 | 有组件无 spec | `tree-select.tsx`/`stepper.tsx`/`line-tabs.tsx` 等，83 组件仅 ~37 有 spec | 列"真漏 vs 故意不沉淀"清单交裁决 | [设计] |
| M3-4 | spec 冗余/承诺缺口 | `file-browser.md`+`upload-file-browser.md` 共指一组件；`transfer.md:18` TreeTransfer"暂未沉淀" | 合并冗余 spec / 标注缺口 | [设计] |

### 3.4 M4 · 局部优化不贯彻子元素 / spec↔实现口径不一致

| # | 容器 | 证据 | 处置建议 | 裁决方 |
|---|---|---|---|---|
| M4-1 | **盲区集中：card-surface / dialog-drawer / form-controls / search-filter-bar** | 4 个 spec 的 Visual Standard 表**只列容器属性（圆角/边框/阴影/高度/间距），不列子元素字号/行高/颜色 token**，留"改容器不改子元素"漏洞 | 4 份 spec 补"子元素字号/色 token"强制条目 | [AI可直修] |
| M4-2 | table（正面样板） | `table.md:25/44` 强制 12px，`table.css:46/63/80` 实现一致——盲区已堵，可作改造范式 | 以 table 模式补齐其它容器 | — |
| M4-3 | spec 颜色注释过期 | `table.md:23/25` 注 muted=#737373/body=#171717，但 `tokens.css:21/23` 实际 #64748B/#1E293B | 修正 spec 注释为实际 token 值 | [AI可直修] |
| M4-4 | 实现侧硬编码 | `table.css:113-117` 状态色、`input.css`/`search-filter-bar.css:62` 边框 `#355EF1` 等硬编码，spec 要求走 var(--cp-*) | 实现去硬编码走 token | [AI可直修] |

### 3.5 M5 · 机检盲区 / M6 · 出口 / A0 · 已有机制（可复用）

| # | 项 | 证据 | 处置建议 | 裁决方 |
|---|---|---|---|---|
| M5-1 | 脚本仅 5 条单行正则 | `check-design-usage.mjs:32-58` 逐行匹配，无作用域概念 | 增强为多规则+作用域 | [补机制] |
| M5-2 | `#1447E6` 漏检 | `old-brand-color` 正则只含旧色 `#007AFF/#5856D6`，**不含现行 #1447E6**、不含 `text-gray-*`/`bg-yellow-50` | 补硬编码现行色正则 | [补机制] |
| M5-3 | 9 槽位 lucide 漏检 | 脚本无槽位概念；`check:skill-map` 等专项**不在本 skill 5 脚本内、不可移植** | 把槽位校验纳入可移植脚本 | [补机制] |
| M5-4 | 表格 14px 漏检 | 无表格作用域识别；`table.md` 连显式 `14px/text-sm` 反例文本都没有 | 脚本加表格作用域规则 | [补机制] |
| M5-5 | emoji 漏检 | 正则 `1F300-1FAFF` **漏 ✅(2705)❌(274C)⚠️(26A0)** 及 §9.2 表内 ⚙️✏️⬇️；单行匹配漏跨行 | 扩 emoji 字符区+跨行 | [补机制] |
| M6-1 | 待确认无汇总出口 | 出口文件已有（`conflict-log.md`），但**无"本轮 needs-design-confirmation 汇总清单"、无脚本扫描汇总**；与 registry `pending-design-review` 双出口未统一 | 复用 conflict-log + 补汇总脚本，统一双出口 | [补机制] |
| A0-1 | 已有机制可复用，无需新建 | `conflict-log.md`/`DESIGN-AUDIT-PLAYBOOK.md`/`COMMON-ERRORS.md`/`EVALUATION.md`/`qa/*-checklist.md` 五类均已存在 | 全部**复用**，各自补强 | [AI可直修] |
| A0-2 | EVALUATION 工具链评分偏乐观 | `EVALUATION.md` 给工具链 5.0 满分，与本次盲区（M5-2~M5-5）矛盾 | 修订该项评分并标注盲区 | [AI可直修] |

---

## 四、问题分类统计

| 裁决方 | 条目 | 数量 |
|---|---|---|
| **[设计] 需拍口径** | M1-1~M1-4、M1-6、M1-7、M1-8、M2-1、M3-3、M3-4 | 9 项关键口径 |
| **[AI可直修] 确认后即改** | M1-5、M1-9、M2-2、M2-3、M7-1~M7-4、M3-1、M4-1、M4-3、M4-4、A0-1、A0-2 | 13 条 |
| **[补机制] 脚本/索引** | M3-2、M5-1~M5-5、M6-1 | 7 条 |

---

## 五、修复路线（待确认点通过后执行）

- **阶段 B · 收敛修复**：先处理 [AI可直修] 13 条（事实性纠错、过期引用、自身硬编码），再按设计裁决处理 [设计] 项。改 skill 前必先 `git diff` 比对原文，不丢失原有合理规则。
- **阶段 C · 补机制**：增强 `check-design-usage.mjs`（补 `#1447E6`/表格作用域/9 槽位/emoji），建 spec↔实现 alias 映射表，统一待设计确认出口。
- **阶段 D · 固化**：把检查/修复原则写进 `SKILL.md`，使同类问题可被主动发现、被脚本拦截、被记录上报。

---

## 六、待裁决清单（需设计/负责人拍板 — 修复前置）

| # | 待决问题 | 选项 | 裁决结论 |
|---|---|---|---|
| **D1** | 文字色体系统一走哪套？（影响全仓，对应 M1-1~M1-4） | A. gray 系（#171717…） / B. slate 系（#0F172A…，token 现状） | ☐ 待填 |
| **D2** | 表格正文字号口径（M1-5/M1-6） | A. 全表一律 12px / B. 仅紧凑表格 12px | ☐ 待填 |
| **D3** | 圆角 token 命名与数值（M1-7/M1-8） | 4px 叫 `radius-lg` 还是 `radius-md`；`radius-sm`=2px 还是 3px | ☐ 待填 |
| **D4** | `SKILL.md §9.2` 那张「页面→lucide」16 项捷径表（M2-1） | A. 拆除业务槽位项 / B. 降级为占位+强警示 / C. 保留 | ☐ 待填 |
| **D5** | 「有组件无 spec」哪些是真漏需补、哪些故意不沉淀（M3-3/M3-4） | 逐项圈定（tree-select / stepper / line-tabs …） | ☐ 待填 |

---

## 七、关键依据文件

| 文件 | 作用 |
|---|---|
| `docs/ClawPro-skill优化计划.md` | 计划全文 + 附录 A 问题登记表 + 附录 B 工作日志（交接入口） |
| `docs/ClawPro资源库-阶段9决策溯源(ADR).md` | 9 类组件槽位禁回退 lucide 的决策依据 |
| `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` | 被优化对象主文 |
| `.../references/conflict-log.md` | 待设计确认 / 冲突的既有出口 |
| `.../scripts/check-design-usage.mjs` | 待增强的机检脚本 |
| `.../MANIFEST.json` | 资产索引（三方对账对象） |

> 本报告为只读体检产物，未对 skill 做任何修改。后续每一步治理动作记录在 `docs/ClawPro-skill优化计划.md` 附录 B 工作日志。
