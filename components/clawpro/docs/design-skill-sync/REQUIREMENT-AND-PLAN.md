# 设计规范改动 → skill 对齐：需求与计划（含 SOP 方案）

> 状态：**讨论已收敛 / 计划阶段（未开工实现）**
> 本文档只落盘需求、目标、范围、方案，**不含任何代码改动**。实现须另起任务。
> 关联：`client/src/index.css`、`client/src/components/ui/*`（真相源）、`SKILL-GLOBAL-COMPONENTS.md`（Tier1）、`.codebuddy/skills/clawpro-portable-design-skill`（Tier2）、`.codebuddy/skills/clawpro-walkthrough`（Tier3）。

---

## 1. 一句话需求

> 设计规范/组件**稳定后**，由人**手动触发**，让 AI 把「代码里的真实改动」沿链路对齐下去：**代码 → `SKILL-GLOBAL-COMPONENTS.md` → 两个 skill**，产出「对不上清单」，逐条经人确认后 reconcile（不覆盖、不私自裁决），并给出以退出码为准的对齐证据。

---

## 2. 真实链路（四层，代码是源头）

工作流：在页面里看到某组件/规范不顺眼或需要新增 → **直接改代码文件**（`index.css` + `components/ui/*.tsx`）。md 文件只是**沉淀文档**，都是下游。

```
真相源（首改地）  client/src/index.css  +  client/src/components/ui/*.tsx     ← 唯一权威，代码为准
   │  ① 沉淀（AI 辅助，本身也是一次可能有损的对齐）        【本任务范围内】
Tier1  SKILL-GLOBAL-COMPONENTS.md（owner addietang/miekoyychen，"冲突以此为准"的规范沉淀文档）
   │  ② 对齐                                             【本任务范围内】
Tier2  clawpro-portable-design-skill（SKILL.md + component-specs×40 + references/* + tokens/* + 页面模板）= "法律"
   │  ③ 对齐                                             【本任务范围内】
Tier3  clawpro-walkthrough（fixtures×6 + §0.A/§0.B/§3 契约表 + detectors）= "警察"
```

**关键推论**：连 Tier1 都是沉淀产物、可能有损，所以 **P1 值/枚举校验一律以代码为准**（组件 `.tsx` 的 union/variant、`index.css` 的 CSS 变量）；文档（Tier1/Tier2）只对「代码里没有的、纯语义/取舍类」规则负责。这正是防住 status-tag 那类事故的根本——源码里 `fill/soft` 一直都在，只要以代码为准就不会被文档带偏。

---

## 3. 需求目标

1. **明确的手动触发方式**：规范稳定后，一句触发语启动对齐流水线；改动期频繁修改**不打扰**。
2. **改动范围靠 `git diff` 机械圈定**，不靠人口述（人记忆有损，是历史漂移诱因之一）。
3. **确定性步骤脚本化**：P1 差集、P2 ghost、MANIFEST 校验、fixtures 重抽等，输出退出码/差集，不靠 AI 口头断言。
4. **语义/契约漂移由 AI 传播**：新增/改名组件、规则文字、章节号 → 同步 Tier2 交叉引用与 Tier3 契约表/detector 覆盖。
5. **每层先出「对不上清单」逐条确认再改**（reconcile 不覆盖），模糊冲突进 `conflict-log.md`。
6. **记录对齐基线**（commit/时间），供下次 `git diff` 参照。

---

## 4. 范围

### 4.1 范围内
- Stage 0：代码改动**沉淀进 `SKILL-GLOBAL-COMPONENTS.md`**（Tier1）。
- Stage 1：对齐 **Tier2 设计 skill**（SKILL.md / component-specs×40 / references / tokens / **页面模板 page-recipes·page-references**）。
- Stage 2：对齐 **Tier3 走查 skill**（fixtures×6 / §0.A·§0.B·§3 契约表 / detectors；`showcase-anchors.json` 仅机械重抽）。

### 4.2 范围外
- **展示台 `client/src/pages/DesignSystemComponents.tsx` 的对齐**：由 owner 单独处理（需兼顾展示效果、且组件/规范不会大规模改）。它**不是真相源**，禁止拿它当比对基准；走查的 `showcase-anchors.json` 只机械重抽镜像其现状，**不修展示台本身**。
- **业务组件源码本身**：它是真相源，本任务不改它。
- **图标资源治理**：另立项。
- **历史存量不回填**：现存无 spec 的组件、重复/冗余组件属历史包袱，本机制**只防增量、不负责补齐**——它们进「历史存量豁免清单」，仅当被改动碰到时才纳入对齐（详见 §10.1）。

---

## 5. 要解决的漂移两类（历史事故，团队上轮已修存量，机制要防增量）

| 类型 | 含义 | 真实实例 | 基准 / 检法 |
|---|---|---|---|
| **P1 值/枚举漂移** | skill/Tier1 声明的 variant/mode/token 与**代码**背离 | `status-tag.md` 曾声称「只剩 text、fill/soft 已删」，而 `status-tag.tsx` 里 `type StatusTagMode = "text"｜"dot"｜"fill"｜"soft"`、fill 还是默认值 | 以**代码**为准：提取 `.tsx` union/variant + `index.css` CSS 变量 vs 文档声明，列差集 |
| **P2 交叉引用不自洽 / ghost** | 文档引用了**不存在的文件**或**错挂组件别名** | 页面参考引用 `component-specs/tree-select.md`（当时只有 `tree.md`）；`combobox.md` 把 TreeSelect 错挂到 `tree.md` | `check-spec-symbols` 类 ghost 检查 + 扩到「页面参考↔spec 文件存在性、组件别名映射」 |

> 现状核实（可信信号）：`tree-select.md` 现已存在；`status-tag.md` 已改为「以组件实现为准」，与 `status-tag.tsx` union 一致。**存量已修，本机制目标是防增量。**

---

## 6. 关键决策（已拍板）

| # | 决策 | 结论 |
|---|---|---|
| 1 | 触发方式 | **只手动触发**，不自动、不 CI 硬卡；自检脚本最多"可选手动运行" |
| 2 | 载体 & 入口 | **SOP 文档为主（大脑）+ 复用/补齐少量脚本（手脚）**；触发**先走「人工入口」（A）**——人贴 kickoff / 说口令，AI 才去读 SOP 执行；**不折叠进现有两个 skill**（法律/警察角色不搅浑）；将来嫌人工烦，再升级为独立薄 `sync-skill` |
| 3 | 触发语约定 | 走人工入口：**增量对齐**说「对齐 skill 规范」；**全量审计**说「全量体检 skill 对齐度」（见 §12）。口令只是**人记的入口语、非自动 hook**；将来薄 skill 化时再固化为触发语/参数 |
| 4 | 真相源 | **代码（`index.css` + `components/ui/*.tsx`）唯一权威**；md 全是下游 |
| 5 | P1 基准 | **以代码为准**；文档只负责代码里没有的语义/取舍规则 |
| 6 | 沉淀 Tier1 | **纳入范围**（必经环节，要管上） |
| 7 | 展示台 | **范围外**，owner 单独处理；`showcase-anchors.json` 仅机械重抽 |
| 8 | reconcile 原则 | 先出「对不上清单」逐条确认再改；**不覆盖 Tier2 独有内容**（Admin/Tenant 场景分流、portable fallback、图标 9 槽位治理、页面模板）；模糊冲突进 `references/conflict-log.md` 不私自裁决 |
| 9 | 改动圈定 | 以 `git diff`（对上次对齐基线）为准，人口述仅作提示 |

---

## 7. 方案：手动触发的分阶段对齐 SOP

> ✅ 已落地为 `docs/design-skill-sync/SOP.md`（供 AI 读取执行，含增量/全量两段流程）；下面是流程骨架。

### Stage 0 — 沉淀（代码 → Tier1 `SKILL-GLOBAL-COMPONENTS.md`）
- S0.1 `git diff` 列出 `index.css` + `components/ui/*.tsx` 自上次基线的**真实改动**（口述仅作提示，不作准）。
- S0.2 分类：新增组件 / 改 variant·mode·union / 改 token(CSS 变量) / 改交互状态。
- S0.3 以**代码事实**为准，沉淀进 `SKILL-GLOBAL-COMPONENTS.md` 对应章节；文档只补语义取舍。
- S0.4 出「Tier1 待更新清单」→ 逐条确认 → 改 → `git diff` 复核落盘。

### Stage 1 — 对齐 Tier2 设计 skill（代码 + Tier1 → portable-design）
- S1.1 **P1 差集（只读脚本）**：组件 union/variant/CSS 变量 vs `component-specs/*` + `SKILL.md` + `tokens/*` 声明，列不一致。
- S1.2 **P2 ghost**：`check-spec-symbols.mjs` + 扩展（页面参考↔spec 存在性、组件别名映射）。
- S1.3 **MANIFEST**：`verify-manifest.mjs`（新增/删除组件是否登记，componentSpecs 计数）。
- S1.4 **component-mapping**：`generate-component-mapping.mjs` 重生成。
- S1.5 **页面模板**：校验 `page-recipes` / `page-references` 引用的 spec/组件**存在且挂对**（不比值）。
- S1.6 出「对不上清单」→ 逐条确认 → reconcile（不覆盖 Tier2 独有内容）→ 复跑脚本至退出码 0。

### Stage 2 — 对齐 Tier3 走查 skill（Tier2 → walkthrough）
- S2.1 **重抽 fixtures**：编排 6 个 extractor（落实 `$walkthrough refresh-fixtures`，DESIGN.md 已规划）；`showcase-anchors.json` 机械重抽镜像展示台现状。
- S2.2 **契约表 §0.B / §3**：新增/改名组件、新规则 → 判断是否需要新 detector / 更新锚点（AI 语义传播）。
- S2.3 **§0.A 阅读清单**：章节号变动同步。
- S2.4 试跑 `walkthrough audit <样例>` 确认 detector 正常加载。
- S2.5 出「契约漂移清单」→ 确认 → 改 → 复跑。

### 收尾
- 记录本次对齐基线（commit hash / 时间），供下次 `git diff` 参照。
- 模糊冲突写入 `references/conflict-log.md`（按 §1.5 模板）。

---

## 8. 工具清单

### 8.1 复用现有
- Tier2：`check-spec-symbols.mjs`（ghost/P2）、`verify-manifest.mjs`、`generate-component-mapping.mjs`、`sync-tokens.mjs`、`verify-portable-skill.mjs`、`check-design-usage.mjs`。
- Tier3：`extractors/` 6 个（tokens/specs/page-recipes/icon-slots/portable/showcase）。

### 8.2 待补（实现阶段）
1. ✅ **`refresh-fixtures` 子命令（已实现）**：`walkthrough.mjs refresh-fixtures [--only=<名>] [--json]`，一键编排 `extractors/` 6 个 extractor 重抽 fixtures，逐个报 ok/fail、退出码 0/1。
2. ✅ **只读 P1 差集脚本（已实现）**：`clawpro-portable-design-skill/scripts/diff-code-vs-docs.mjs`。提取组件 `.tsx` 的 union / 内联联合 / cva variant 键 + `index.css` 的 CSS 变量，与该组件 spec / `tokens`（含 `design-tokens.json`）比对列差集。支持 `--since <ref>`（增量）/ `--full`（全量），退出码 0/1/2；只查"代码→文档覆盖"正方向（反向 & 别名交给 AI / P2）。
3. **扩展 `check-spec-symbols.mjs`（可选）**：覆盖「页面参考↔spec 文件存在性 + 组件别名映射」，直接命中 P2 类。

---

## 9. 已核实事实基线（以事实为准、勿凭记忆推翻；改动前先 `git diff`/`ls`/`grep` 复核）

- 展示台 = `client/src/pages/DesignSystemComponents.tsx` 的 `COMPONENTS` 数组（范围外）。
- 存量已修：`component-specs/tree-select.md` 现存在；`status-tag.md` 已改「以组件实现为准」，与 `status-tag.tsx` union 一致。
- Tier2 `scripts/` 真实 9 个（见上 §8.1）；`MANIFEST.json` 登记 40 个 `component-specs`。
- Tier3 `scripts/` 只有 `walkthrough.mjs` + `detectors/`；`extractors/` 6 个 → `fixtures/` 6 个 JSON。
- **不存在** `verify-consistency.mjs` / `refresh-fixtures.mjs`（脚本）/ `verify-skill-toolchain.mjs` / `MANIFEST.scripts.json`（历史 memory 幻象，已删除相关 memory）；`$walkthrough refresh-fixtures` 只是 `DESIGN.md` 里未实现的 roadmap 子命令。
- `.codebuddy/skills/` 下**无 package.json**；`SKILL.md` 里的 `npm run` 命令依赖宿主仓 `client/src/design-assets`。

---

## 10. 决策记录（原待拍板项，2026-07-02 已拍板）

1. **触发入口** —— **A：人工入口**。SOP 文档为大脑；人贴 kickoff / 说口令，AI 才读 SOP 执行。**不折叠进现有两个 skill**；将来嫌人工烦再升级为独立薄 `sync-skill`。
2. **映射表纳入范围** —— **纳入，但按「增量 allow-list + 历史存量豁免」**：只登记被 sync 碰到的组件；历史 spec-less / 重复组件不回填，进豁免清单（见 §4.2 与下方 §10.1）。
3. **P1 自动化程度** —— **脚本 + AI**：只读脚本机械列差集（退出码为准），AI 逐条判定。
4. **增量 vs 全量** —— 同一套脚本两种 scope：增量默认（`--since 基线`），全量审计 opt-in（`--full`）；全量 findings 只进 backlog、不强制修（详见 §12）。

### 10.1 映射表说明（组件源码 ↔ spec）

- ⚠️ **现有 `references/component-mapping.md`（由 `generate-component-mapping.mjs` 生成）映射的是 spec ↔ skill 内 portable 实现（`portable/react·css`），并不映射真相源 `client/src/components/ui/*.tsx`。** 别把它当成"组件源码↔spec"映射误用。
- 本项要新建的是「**组件源码 ↔ Tier2 spec**」映射，且**懒增长（lazy）**：组件被 sync 碰到时才登记进映射；未登记的历史存量（现 40 spec vs ~80 组件的缺口、重复组件）列入**豁免清单**，不强制补 spec。
- 目的：消除 P2（错挂/ghost）、防增量漂移，同时**不背历史包袱**。

---

## 11. 同事演示 / 教学场景设计（给同事使用教学用）

> 目的：向同事演示「**规范/组件/页面模板一旦改动，一句口令就让 skill 沿链路对齐变动**」。
> **前置依赖**（最小可演示集，✅ 已落地）：`docs/design-skill-sync/SOP.md`（含增量/全量两段流程）+ 人工入口口令（见 §6#3）+ 只读 P1 差集脚本 `diff-code-vs-docs.mjs`（见 §8.2）。三者已就绪，本节场景可实跑。

### 11.1 统一触发流程（所有场景通用）

```
① 改真相源代码（组件/规范/页面模板，三选一或组合）
② git commit（作为 diff 锚点；改动范围靠 git diff 机械圈定，不靠口述）
③ 说口令「对齐 skill 规范」（人工入口，AI 读 SOP 执行；见 §6#3）
④ AI 按 SOP 跑：Stage0 沉淀 Tier1 → Stage1 对齐 Tier2 → Stage2 对齐 Tier3
⑤ 每层先出「对不上清单」→ 逐条确认 → reconcile（不覆盖 Tier2 独有内容）
⑥ 复跑脚本至退出码 0 → 记录本次对齐基线（commit / 时间）
```

> 演示要点：**改动只在代码里做，文档/skill 的更新完全由触发后的流水线产出**，同事看到的是「清单 → 确认 → 落盘 → 退出码 0」这条闭环。

### 11.2 场景一 · P1 值/枚举改动（组件改 variant/mode/union）

- **改什么**：给 `client/src/components/ui/status-tag.tsx` 的 `StatusTagMode` union 增/删一个形态。**演示值选独特、不与既有示例撞词的**（如 `"banner"`；勿用 `"outline"`——它会被 spec 里 `<Badge variant="outline">` 示例掩盖）。或改某 variant 的 CSS 变量。
- **触发后预期**：`node scripts/diff-code-vs-docs.mjs --since <baseline>` 报 `CODE_ENUM_NOT_IN_DOC：status-tag union StatusTagMode = "banner"`（退出码 1）→ 进「对不上清单」。
- **确认后动作**：Stage0 沉淀进 Tier1 对应章节 → Stage1 更新 `status-tag.md` §3.4 形态表 → Stage2 判断走查 detector / 契约表是否需覆盖新形态。
- **教学点**：这正是历史上 status-tag「文档声称只剩 text」事故的防线——**以代码为准**，文档漏声明会被机械揪出。

### 11.3 场景二 · P2 交叉引用 / ghost（新增或改名组件）

- **改什么**：新增一个组件文件，或对现有组件改名（如把 `xxx.tsx` 重命名 / 改导出名）。
- **触发后预期**：`check-spec-symbols` 类 ghost 检查报出「页面参考 / spec 里 import 的标识符在 `client/src` 不存在」或「别名错挂」→ 进「对不上清单」。
- **确认后动作**：更新 spec 的 import 名、MANIFEST 登记、component-mapping 重生成；新增组件补 spec 骨架。
- **教学点**：对应历史上 `combobox.md` 错挂 `tree.md`、页面参考引用不存在 `tree-select.md` 的漂移。

### 11.4 场景三 · 页面模板改动（page-recipes / page-references）

- **改什么**：改 Tier2 页面模板里引用的 spec / 组件（如页面配方换了个组件、或引用了新 spec）。
- **触发后预期**：Stage1 S1.5 校验报出「页面模板引用的 spec/组件是否存在且挂对」（只校验存在性与挂载，不比值）→ 进「对不上清单」。
- **确认后动作**：修正模板引用，使其指向真实存在且正确别名的 spec/组件。
- **教学点**：页面模板是 Tier2 独有资产，reconcile 时**不被覆盖**，只做引用自洽校验。

### 11.5 演示收尾

- 展示 `references/conflict-log.md`：如遇模糊冲突（语义取舍拿不准），演示「不私自裁决、写入冲突日志待人拍板」。
- 展示对齐基线记录：本次 commit hash / 时间，说明「下次触发时 `git diff` 以此为参照，只处理增量」。

---

## 12. 增量对齐 vs 全量审计（两种 scope，别混）

> 同一套脚本（P1 差集 / `check-spec-symbols` 等），靠**范围开关**切换；对应两种意图、两段流程。

| 维度 | 增量对齐（默认，本机制主线） | 全量审计（opt-in，体检用） |
|---|---|---|
| scope | `--since 基线`（只看 `git diff`） | `--full`（全量扫，无 diff 过滤） |
| 触发（人工入口） | 说口令「对齐 skill 规范」 | 说口令「全量体检 skill 对齐度」 |
| findings 处理 | 进「对不上清单」→ 逐条确认 → reconcile | 只进 backlog / 一次性快照，**不强制修** |
| 历史存量 | 绕开（豁免清单，不报） | 会全抖出来（含 spec-less / 重复组件） |

- **默认必须是增量**：日常 sync 绝不自动跑全量，避免一次吐 80 条淹没。
- **全量审计定位**：用于回答"现在 skill 对规范对齐到什么程度"，产出一次性快照，findings 只作 backlog 待办、**不作必修**。
- **两层分开界定，先落进 SOP 文档**：增量流程 / 全量流程各写一段（何时用、扫什么、findings 怎么处理）；将来薄 skill 化时，对应两个触发语或「一个触发语 + `--full` 参数」，本节界定直接沿用。

---

## 13. 演进路线（分三阶段落地，逐步增强、不一步到位）

> 原则：**先轻后重、先人工后自动、先不卡后卡**。每阶段先试跑收集反馈，达到"退出标准"再进下一阶段；任何时候都以「不打扰日常、结论以退出码为准」为底线。

### Phase 1 · 现状：SOP 文档 + skill 脚本 + 人工 prompt 触发（人工入口）
- **形态**：`SOP.md`（大脑）+ 复用/补齐的脚本（`diff-code-vs-docs` / `sync-tokens` / `verify-manifest` / `check-spec-symbols` / `generate-component-mapping` / `refresh-fixtures`，手脚）+ 人贴 kickoff 触发。
- **触发**：改完（并提交）→ 贴 kickoff（增量 `--since 基线` / 全量 `--full`）→ AI 读 SOP 执行 → 出「对不上清单」逐条确认 → reconcile → 复跑退出码 0。
- **目的**：低成本验证「链路、脚本、清单→确认→reconcile 闭环」是否顺手、是否真的防住漂移。
- **退出标准（进 Phase 2 的条件）**：多名成员各自跑过 ≥N 次真实改动；kickoff 措辞/基线选择稳定；脚本无高频误报；SOP 步骤无需频繁临时解释。

### Phase 2 · 沉淀为独立薄 `sync-skill`（便于自动/半自动触发）
- **动因**：Phase 1 人工贴 kickoff、手选基线较繁琐，且新成员上手有门槛。
- **形态**：把 kickoff 固化为薄 skill 的**触发语 + 参数**（如 `对齐 skill 规范` / `全量体检 skill 对齐度` / `--full`），基线自动取 `references/alignment-baseline.json`；SOP 仍是 skill 内部依据，脚本不变。
- **边界**：薄 skill 只做「编排既有脚本 + 读 SOP 执行」，**不搅浑现有两个 skill 的"法律/警察"角色**（决策 §6#2）。仍是人主动触发，**不自动、不 CI 硬卡**。
- **退出标准（进 Phase 3 的条件）**：薄 skill 触发稳定、误触发少；团队形成"改完就对齐"的习惯；有数据表明增量对齐确实拦下了漂移。

### Phase 3 · 按需接入「提交必经校验」（可选、最后一步）
- **动因**：若仍出现"改了不对齐就提交"的漏网漂移，再考虑把增量对齐变成提交流程的硬约束。
- **形态（按需选其一，从轻到重）**：
  1. **pre-commit 提醒**（非阻断）：提交时跑增量差集，有候选给 warning，不拦。
  2. **PR/CI 增量卡口**（阻断）：对 `git diff` 跑差集/走查，命中阻断级 finding 才卡 CI（复用 walkthrough 的 `diff` 退出码机制）。
- **谨慎前提**：只卡**增量**、只卡**阻断级**、历史存量豁免；避免一刀切淹没或误伤。是否落地、卡到多严，由团队在 Phase 2 数据基础上再拍板。
- **退出标准**：达成"漂移增量趋零"且开发者体验可接受。

### 阶段对照

| 维度 | Phase 1（现状） | Phase 2（薄 skill） | Phase 3（按需卡口） |
|---|---|---|---|
| 载体 | SOP 文档 + 脚本 | 独立薄 sync-skill | + pre-commit / CI |
| 触发 | 人贴 kickoff | 触发语/参数（人主动） | 提交/PR 自动触发 |
| 强制度 | 全人工、不卡 | 半自动、不卡 | 增量阻断（按需） |
| 进入条件 | — | Phase 1 试跑稳定 | Phase 2 数据支撑 |
