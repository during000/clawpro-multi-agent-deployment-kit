# SOP — 设计规范改动 → skill 对齐（供 AI 读取执行）

> 本文件是「设计规范/组件稳定后，手动触发把代码真实改动沿链路对齐下去」的**执行脚本（大脑）**。
> 需求/决策背景见同目录 `REQUIREMENT-AND-PLAN.md`；本文件只讲**怎么做**。
> 触发方式（决策 §6#2）：**人工入口**——人说口令，AI 读本文件并执行；不折叠进现有两个 skill。

---

## 0. 触发口令与两种模式（先分清再动手）

| 口令（人说） | 模式 | scope | findings 处理 |
|---|---|---|---|
| **「对齐 skill 规范」** | 增量对齐（默认主线） | 自上次基线的 `git diff` | 进「对不上清单」→ 逐条确认 → reconcile |
| **「全量体检 skill 对齐度」** | 全量审计（opt-in） | 全量扫描 | 只进 backlog / 一次性快照，**不强制修** |

- **默认永远是增量**。除非人明确说「全量体检」，否则按增量走，绝不自动跑全量（避免一次吐几十上百条淹没）。
- 两种模式**共用同一套脚本**，只是 scope 开关不同（`--since` vs `--full`）。

---

## 链路（代码是唯一真相源）

```
真相源  client/src/index.css + client/src/components/ui/*.tsx      ← 唯一权威，以代码为准
   │  ① 沉淀
Tier1  SKILL-GLOBAL-COMPONENTS.md（"冲突以此为准"的规范文档）
   │  ② 对齐
Tier2  .codebuddy/skills/clawpro-portable-design-skill（SKILL.md + component-specs + tokens + 页面模板）
   │  ③ 对齐
Tier3  .codebuddy/skills/clawpro-walkthrough（fixtures + 契约表 + detectors）
```

**铁律**：
- **P1 值/枚举一律以代码为准**（组件 `.tsx` 的 union/variant、`index.css` 的 CSS 变量）；文档只对「代码里没有的、纯语义/取舍规则」负责。
- **reconcile 不覆盖 Tier2 独有内容**：Admin/Tenant 场景分流、portable fallback、图标 9 槽位治理、页面模板。
- **每层先出「对不上清单」，逐条经人确认再改**；模糊冲突进 `references/conflict-log.md`，不私自裁决。
- **改动范围靠 `git diff` 机械圈定**，人口述仅作提示。
- **历史存量不回填**：无 spec 组件 / 重复组件属历史包袱，只在被改动碰到时才纳入；否则进 `references/historical-exemption.json` 豁免。

---

# A. 增量对齐流程（口令「对齐 skill 规范」）

> 前提：人已改完代码并 **commit**（作为 diff 锚点）。若没记住上次基线，取 `references/alignment-baseline.json` 里的 commit；没有该文件则问人用哪个基线 ref。

## Stage 0 — 沉淀（代码 → Tier1 `SKILL-GLOBAL-COMPONENTS.md`）

- **S0.1** 圈定改动：
  ```
  git diff --stat <baseline> -- client/src/index.css client/src/components/ui
  ```
  以输出为准，不听口述。
- **S0.2** 分类每处改动：新增组件 / 改 variant·mode·union / 改 token(CSS 变量) / 改交互状态。
- **S0.3** 以**代码事实**为准，把改动沉淀进 `SKILL-GLOBAL-COMPONENTS.md` 对应章节；文档只补「代码表达不了的语义取舍」。
- **S0.4** 出「Tier1 待更新清单」→ 逐条经人确认 → 改 → `git diff SKILL-GLOBAL-COMPONENTS.md` 复核确已落盘。

## Stage 1 — 对齐 Tier2 设计 skill（代码 + Tier1 → portable-design）

- **S1.1 P1 差集（只读脚本，退出码为准）**：
  ```
  node .codebuddy/skills/clawpro-portable-design-skill/scripts/diff-code-vs-docs.mjs --since <baseline>
  ```
  - 退出码 0 = 无差集；1 = 有差集候选（读输出逐条判定）；2 = 用法/环境错。
  - finding 类型：
    - `CODE_ENUM_NOT_IN_DOC` = 代码有枚举值、spec 未声明 → **以代码为准**，判定「有效 API 就补进 spec，内部实现细节可放行」。
    - `CSS_VAR_NOT_IN_DOC` = 新增 CSS 变量未进 token 文档 → 补 `tokens/*.md`，并（若需要）重跑 `sync-tokens.mjs` 刷新 `design-tokens.json`。
    - `CHANGED_NO_SPEC` = 被改动碰到的组件无 spec → 补 spec 骨架，或（历史存量）加入 `references/historical-exemption.json`。
- **S1.2 P2 ghost / 交叉引用**：
  ```
  node .codebuddy/skills/clawpro-portable-design-skill/scripts/check-spec-symbols.mjs
  ```
  报「spec import 了 client/src 不存在的标识符」→ 改名/删除/迁反例。
- **S1.3 MANIFEST**：`node scripts/verify-manifest.mjs`（新增/删除组件是否登记、componentSpecs 计数）。
- **S1.4 component-mapping**：`node scripts/generate-component-mapping.mjs` 重生成。
  > 注意：该表映射的是 **spec↔portable 实现**，不是 spec↔源码；spec↔源码映射按需在本流程里懒登记。
- **S1.5 页面模板**：校验 `page-recipes` / `page-references` 引用的 spec/组件**存在且挂对**（只校验存在性与挂载，不比值）。
- **S1.6** 出「Tier2 对不上清单」→ 逐条确认 → reconcile（**不覆盖 Tier2 独有内容**）→ 复跑上述脚本至相关退出码 0。

## Stage 2 — 对齐 Tier3 走查 skill（Tier2 → walkthrough）

- **S2.1 重抽 fixtures**：一键重抽——
  ```
  node .codebuddy/skills/clawpro-walkthrough/scripts/walkthrough.mjs refresh-fixtures
  ```
  编排 `extractors/` 6 个 extractor（tokens/specs/page-recipes/icon-slots/portable/showcase）重写 `fixtures/*.json`；`showcase-anchors.json` 仅机械重抽镜像展示台现状（**不修展示台本身**）。逐个报 ok/fail，退出码 0=全成功。
- **S2.2 契约表 §0.B / §3**：新增/改名组件、新规则 → 判断是否需要新 detector / 更新锚点（AI 语义传播）。
- **S2.3 §0.A 阅读清单**：章节号变动同步。
- **S2.4** 试跑 `walkthrough audit <样例>` 确认 detector 正常加载。
- **S2.5** 出「契约漂移清单」→ 确认 → 改 → 复跑。

## 收尾

- 记录本次对齐基线到 `references/alignment-baseline.json`：
  ```json
  { "commit": "<最新 commit hash>", "at": "<ISO 时间>", "note": "本轮对齐范围简述" }
  ```
  供下次 `git diff` 参照。
- 模糊冲突写入 `references/conflict-log.md`（按 REQUIREMENT-AND-PLAN §1.5 / 冲突模板）。

---

# B. 全量审计流程（口令「全量体检 skill 对齐度」）

> 用途：回答"现在 skill 对规范对齐到什么程度"，产出一次性快照。**findings 是 backlog，不是必修清单。**

- **B.1 P1 全量差集**：
  ```
  node .codebuddy/skills/clawpro-portable-design-skill/scripts/diff-code-vs-docs.mjs --full
  ```
  - 会一次性抖出全部候选，含大量**历史存量**（`NO_SPEC_BACKLOG` = 无 spec 的组件、组件级私有 CSS 变量等）。
  - 处理原则：**分类归档进 backlog，不逐条强改**。真正要处理的由人挑选立项。
- **B.2 P2 全量**：`check-spec-symbols.mjs`（本就是全量）。
- **B.3 汇总快照**：把 by-type 计数 + 关键清单写成一份报告（如 `references/audit-snapshot-<date>.md`），标注「含历史存量，仅供参考」。
- **B.4** 不写基线（全量审计不改变增量基线）。

---

## 附：diff-code-vs-docs.mjs 用法速查

```
# 增量（二选一必填其一）
node scripts/diff-code-vs-docs.mjs --since <git-ref>
# 全量
node scripts/diff-code-vs-docs.mjs --full
# 机读
node scripts/diff-code-vs-docs.mjs --since <ref> --json
# 指定源码根（自动定位失败时）
node scripts/diff-code-vs-docs.mjs --full --src=/abs/path/client/src
```

- 退出码：`0` 无差集 / `1` 有差集候选 / `2` 用法或环境错。
- **`<git-ref>` 基线怎么选（关键）**：改动**未提交**（留在工作区）→ 用 `HEAD`；改动**已提交** → 用「改动**之前**的基线 commit」——刚提交一次用 `HEAD~1`，距上次对齐提交了多次用 `references/alignment-baseline.json` 记录的 commit（没有该文件则问人用哪个 ref）。注意：改动已提交时若仍用 `HEAD` 会「检查 0 个」漏掉，因为工作区相对 HEAD 已干净。
- 已知边界（LIMITATIONS，见脚本头注释）：只查「代码→文档覆盖」正方向；组件↔spec 靠同名匹配（改名/别名属 P2）；不提取大对象 keyof（如色板 map）；覆盖判定是字面量字符串匹配、不理解语义——**输出是候选，不是判决**，AI/人逐条定夺。
- **演示取值提示**：教学演示新增枚举时，选一个**独特、不与既有示例撞词**的值（如 `banner`），避免用 `outline` 这类会被别的组件示例（`<Badge variant="outline">`）掩盖的常见词。
