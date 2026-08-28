> ⚠️ **本文件为长期路线图（DESIGN.md），不是当前实现说明。**
>
> - 当前**实际可跑**的能力 → 看 `clawpro-walkthrough/SKILL.md`（v0.6 audit-only：6 类 detector + `audit` / `diff` / `explain`）
> - 当前**fixtures / extractors** → 看 `clawpro-walkthrough/SKILL.md` + `README.md`
> - 本文件是 **v1.0 / v1.5 的全景蓝图**，描述的是「未来要长成的样子」，包含 critique / polish / 健康分 / 趋势 / 复核 webUI 等尚未实现的能力
> - 实施进度对照表见下方「实施现状对照」

---

## 实施现状对照（v0.6 · 2026-06-30）

| 蓝图中的能力 | 在本文档对应章节 | 现状 | 物理位置 |
|---|---|---|---|
| 五大参照源 → fixtures 抽取 | §2.3 | ✅ 已实现 5 个 extractor | `clawpro-walkthrough/extractors/` |
| `tokens.json` / `component-spec-index.json` / `page-recipes.json` / `portable-impl-index.json` / `showcase-anchors.json` | §2.3 | ✅ 已生成 | `clawpro-walkthrough/fixtures/` |
| `audit` 子命令 + 10 类 detector | §4.2 / §5.1 | ✅ v0.8 已实现 9 内置 + 1 外部（radius / color / icon-slot / shadow / component-drift / page-recipe-match / **text-color / surface-nesting / spacing-grouping** + external/*） | `clawpro-walkthrough/scripts/detectors/` |
| `diff` 子命令 | §5.2 | ✅ 已实现 | `clawpro-walkthrough/scripts/walkthrough.mjs` |
| `explain` 子命令 | §5.3 | ✅ 已实现 | 同上 |
| `audit-report.csv` | §6.1 | ✅ 已实现（9 列简化版，未上 13 列闭枚举） | `_walkthrough/<ts>/audit-report.csv` |
| `critique` 子命令（设计走查） | §1.5 / §5.1 | ✅ v0.8（评分脚手架，AI 视觉视角独立填） | `walkthrough.mjs#runCritique` |
| `polish` 子命令 + `design-todo.csv` | §1.5 / §5.1 / §6.2 | ✅ v0.8（合并 audit+critique，出总分 + design-todo） | `walkthrough.mjs#runPolish` |
| 健康分模型（critique 24 + audit 32） | §1.5.2 | ✅ v0.8（audit 自动算；critique 由 AI 填回 critique.json） | `walkthrough.mjs#computeAuditHealth` |
| 走查快照 + `trend.json` 趋势曲线 | §1.5.3 / §6.4 | ✅ v0.8（`snapshots/<slug>/trend.json` + `trend` 命令） | `walkthrough.mjs#appendTrend / runTrend` |
| 双独立评估（critique / audit 互盲） | §1.5.1 | 🟡 v0.8（critique 不跑 detector，结构互盲；子 agent 隔离待 v1.5） | `walkthrough.mjs#runCritique` |
| 复核 webUI（`review_viewer.py`） | §6.3 | ❌ 待 v1.0 | — |
| tenant skill（客户端走查） | §3.1 | ❌ 待 v1.0 | — |
| MR 阻断（CI 卡口，P0 不清零不让合） | §9 v1.5 | 🟡 方案已定，待接入蓝盾 Stream（工蜂） | 见 `clawpro-walkthrough/ci/` |
| 误报清单沉淀（`reference/常见误报模式.md`） | §3.2 / §10 | ❌ 待 v1.0 | — |

### 设计 skill ↔ 走查 skill 的耦合现状（2026-06-30 加固）

| 耦合层 | 现状 | 物理位置 |
|---|---|---|
| **L1 数值层（token / spec 索引）** | ✅ 已通过 fixtures 抽取耦合 | `core/fixtures/{tokens,component-spec-index,page-recipes}.json` |
| **L2 槽位事实（9 类不可回退槽位）** | ✅ 抽出 `icon-slots.json`，admin/icon-slot detector + 设计 skill 老脚本 `check-design-usage.mjs` 共用 | `core/fixtures/icon-slots.json` + `core/extractors/extract-icon-slots.mjs` |
| **L3 外部 detector（复用老脚本）** | ✅ admin/audit 自动调 `check-design-usage --json`（8 个 check）+ `check-spec-symbols --json`，输出合并进同一份 audit-report.csv | `admin/scripts/detectors/external-design-skill.mjs` |
| **L4 语义层（规则 ↔ 设计 skill 章节锚点）** | ✅ admin/SKILL.md §0.A/§0.B 写明审查 SOP + 锚点对照表 | `clawpro-walkthrough/SKILL.md` 头部 |
| **L5 冲突铁律（§1.5 / §12 落地到走查链路）** | ✅ admin/SKILL.md §0.C 写明冲突上报流程；外部 detector 自动放过 `needs-design-confirmation` 标记 | `clawpro-walkthrough/SKILL.md §0.C` + `external-design-skill.mjs` 注释段 |
| **L6 critique / polish 段（视觉视角）** | ✅ v0.8（critique 脚手架 + polish 合并 + 健康分 + trend；与 audit 结构互盲） | `admin/scripts/walkthrough.mjs#runCritique/runPolish/runTrend` |

> 同事建议（2026-06-30）："让审查 skill 的脚本挂在设计 skill 的规范原文上，规范是法律，脚本是警察。" 当前 L1–L6 已全部落地。

### 同事 (miekoyychen) MR !123 / !129 / !133 的治理产出（2026-06-30 同步进走查 skill）

| 产出 | 类型 | 走查 skill 怎么用 |
|---|---|---|
| 设计 skill 新增 §1.5「冲突处理铁律」+ §12「冲突处理完整版」 | SKILL.md 章节 | admin/SKILL.md §0.A 列为**最高优先级必读**；§0.C 补齐冲突上报流程，落实"冲突回用户"原则 |
| `check-design-usage.mjs` 升级到 **8 个检查函数**（新增 `collectDesignConfirmationMarkers`） | 脚本能力 | wrapper 注释 + walkthrough.mjs explain 全部对齐到 8；wrapper 不把 confirmations 映射成 finding（避免重复报警） |
| 4 个新治理脚本：`generate-design-todo` / `generate-icon-todo` / `generate-component-mapping` / `verify-manifest` | 脚本 | 目前走查链路不直接调用（产物在设计 skill 内闭环），但 admin/SKILL.md §0.A 把生成出的 `icon-design-todo.md` / `component-mapping.md` 列入必读 |
| `references/component-mapping.md`（42 组件 spec/react/css 三层映射） | 参考文档 | admin/SKILL.md §0.A 列为跑 `component-drift` 之前的必读，给 AI 提供组件全景图 |
| `references/icon-design-todo.md`（图标冲突待办清单，12 项） | 参考文档 | admin/SKILL.md §0.A 列为跑 `icon-slot` 之前的必读，AI 看见的图标问题如果已在该清单里，应建议进 design-todo 而不是 audit-report |
| 图标槽位"禁回退 lucide"策略细化（MR !129） | 规则细化 | admin/SKILL.md §0.B 第 1 行更新为「图标槽位用对、无违规 lucide、**禁回退**」；icon-slot detector + external/prohibited-lucide-slot 都已自动覆盖 |
| MANIFEST.json 补登记 5→9 / 9→11（MR !133） | 索引 | 走查 skill 不直接消费，但提示：未来 v1.0 critique 时要按 MANIFEST 完整度评分（"包齐 = 1 分基线"） |

> **v0.8 修复（2026-06-30）**：同事 MR 给 `check-design-usage.mjs --json` 在写完大块 JSON 后 `process.exit`，管道场景会被 Node 截断，导致 admin/external detector **静默全失效**（只剩 1 条"无法解析 JSON"错误，2900+ 条覆盖归零）。已在 `external-design-skill.mjs` 的 wrapper 侧绕过：stdout 改重定向到临时文件 fd 再读回（文件写同步、不被 exit 截断），未改动同事脚本。

> 节奏建议：v0.8 已推进到 roadmap 0.5 minimum-loop（9+1 detector / design-todo 分流 / critique→audit→polish 三段闭环 / 健康分 + trend）；剩下 **CI 接入 + 每周巡检 automation**（见 `clawpro-walkthrough/ci/`）+ v1.0 复核 webUI / tenant skill / 误报清单沉淀。

---

# ClawPro 走查 Skill 方案（clawpro-walkthrough）

> 状态：长期路线图 · 起草日期：2026-06-30
> 目标：把"产品用 skill 生成页面 / 换皮后，必须找用户审核"这件事从**人力审核**改成**机器走查 + 用户只做最终复核**，把用户人力从"100% 看代码"降到"只看 AI 拿不准的清单"。

---

## 0. 背景与定位

1. **痛点**：当前流程下，产品每次用 `clawpro-portable-design-skill` 1) 生成新页面、2) 对已有页面换皮，都需要用户重新走查一遍，人力成本无法降低。
2. **新增能力**：做一个**走查后先出清单、逐条经用户确认才改代码**的走查 skill `clawpro-walkthrough`，对 PR / 本地代码做静态扫描，输出**结构化报告 + 待用户裁决清单**。
3. **与"ClawPro Skill 优化计划"的关系**：本 skill 是优化计划**阶段 C 的工程化承载**——把 C1（脚本拦截）、C2（待设计清单）变成可复用的独立 skill；同时也是阶段 A（审计取证）的工具雏形。
4. **不做什么**：
   - 不静默 / 不批量改业务代码（扫描只读；发现问题先出清单，逐条经用户确认后才改）
   - 不替代用户的最终判断（"AI 拿不准的"必须留给用户裁决，不私自下结论）
   - 不依赖真实浏览器（**优先静态代码扫描**，截图作为可选可视化辅助）

---

## 1. 核心理念（一句话）

> **静态代码扫描为主，把"页面里所有可走查点"先盘存成全集（inventory），再逐条标记处理结果（coverage），coverage.length == inventory.length 是硬校验**——把"我查过了"从口头承诺变成机器可校验的事实。

方法论参考（三条来源，融合为我们自己的方法论）：
- **内部工程框架沉淀**：inventory + coverage 双盘存、CSV 强枚举、三级合并（文件 → 页面 → PR）、复核 webUI 等已经在我们其他静态检查项目里跑通的工程做法
- **业界通用的"设计审查 / 技术审计 / 上线打磨"方法论**（提炼出 critique → audit → polish 三段闭环、双独立评估、健康分量表、严重度分级、持久化趋势——纳入我们自己的语言体系）
- **本项目 `clawpro-portable-design-skill`**（项目真相：9 槽位 / 158 候选 / token / spec 索引）

三者关系：**工程框架出脚手架，方法论出审查智慧，ClawPro 出项目真相**——本 skill 把三者合一成 ClawPro 自有的走查标准。

---

## 1.5 三段闭环：critique → audit → polish（本方案的方法论骨架）

> 这是把"AI 生成 / 换皮 → 用户裁决"的闭环切成三段，每一段职责单一、产物单一、可被独立调用。三段加起来才是一次完整走查。

| 段 | 角色 | 关注什么 | 主产出 | 主参照源 |
|---|---|---|---|---|
| **critique** · 设计走查 | "设计师视角" | 视觉手感、信息层级、AI slop 痕迹、文案、组件选型是否符合 ClawPro 设计语言 | `critique-report.md`（设计健康分 + 优先问题） | ① 本地预览 + ② 全局样式展示台 + ④ page-references |
| **audit** · 技术审计 | "前端 reviewer 视角" | token / className / 内联样式 / 圆角 / 阴影 / 图标槽位 / 组件实现是否漂离 portable 正版 | `audit-report.csv`（10 类 detector 的结构化结果） | ③ component-specs + ⑤ portable/css + portable/react |
| **polish** · 上线打磨 | "用户 + AI 的合并视角" | 把 critique + audit 的所有 P0 / P1 收口为可执行清单，对接 design-todo.csv | `polish-plan.md` + `design-todo.csv` | 综合 ①~⑤ + 前两步报告 |

### 1.5.1 双独立评估（critique vs audit 必须互盲）

借鉴成熟方法论的关键设计：**critique 与 audit 不能互相看见对方的输出，否则会被锚定**。本 skill 把这一条作为硬约束：

- `$walkthrough critique <target>` 与 `$walkthrough audit <target>` 默认在独立上下文里跑（子 agent 或两次独立调用）；
- 综合在 `$walkthrough polish <target>` 这一步才发生；
- polish 阶段必须明确标注三类信号：①两边都命中 → 强证据；②只 audit 命中 → 静态规则可定性；③只 critique 命中 → "AI 拿不准"，**强制进 design-todo.csv 而非 audit-report.csv**。

> 工程意义：把"AI 视觉判断"和"机器静态规则"分离，避免互相污染；同时让用户只在 polish 阶段介入，看到的就是最终收口清单。

### 1.5.2 健康分模型（让"有没有变好"可度量）

每段产出一个 0~4 分制的健康分：

- **critique 健康分**：覆盖"AI slop / 视觉层级 / 信息架构 / 一致性 / 文案 / 状态完备性"6 项，总分 24。
- **audit 健康分**：覆盖"圆角 / 颜色 token / 阴影 / 组件复用 / 图标槽位 / 内联样式漂移 / 页面骨架 / portable 一致性"8 项，总分 32。
- **总分 = critique + audit**，满分 56。

**评分带**：
| 总分区间 | 评级 | 含义 |
|---|---|---|
| 50–56 | Excellent | 仅细节打磨，可直接合入 |
| 40–49 | Good | 个别维度需修，主体可用 |
| 28–39 | Acceptable | 需走完整 polish 才能合入 |
| 16–27 | Poor | 必须返工，不应进入复核 |
| 0–15 | Critical | 生成失败，重新跑 skill |

### 1.5.3 持久化 + 趋势（让"有没有改好"可看见）

每次 `$walkthrough` 跑完都写一份带时间戳 + 目标 slug 的快照到 `_walkthrough/snapshots/`：

```
_walkthrough/snapshots/
└── admin-tokens-monitor/
    ├── 20260628-1620.md          # 完整报告（包含三段产出）
    ├── 20260629-1130.md
    └── trend.json                # 历史分数曲线
```

跑 `$walkthrough trend <slug>` 直接打印最近 5 次的总分趋势：

```
admin-tokens-monitor 最近 5 次走查: 28 → 33 → 41 → 39 → 47 ↑
```

> 工程意义：把"我这次改了什么"从主观感受变成可对账的数字；老板 / 用户只需看趋势曲线就能判断这条页面是不是在持续好转。

### 1.5.4 严重度分级（P0–P3，统一所有产出）

| 等级 | 名称 | 判据 | 处理 |
|---|---|---|---|
| **P0** | 阻断级 | 违反 ClawPro 设计铁律（如管控端非 4px 圆角 / 9 槽位用 lucide / 完全没用规范组件） | PR 阻断，不清零不能合入 |
| **P1** | 主要 | 偏离规范但不至于阻断（如硬编码颜色 / 组件未走 portable 正版 className） | 当前 PR 必修 |
| **P2** | 次要 | 体感小瑕疵（如间距 token 漂移 / 文案不一致） | 下一轮统一修 |
| **P3** | 打磨 | 锦上添花（如动效缓动曲线 / 微交互） | 有时间再做 |

P0 / P1 进 `audit-report.csv` 由 AI 直接修；critique 出的 P0 / P1 进 `design-todo.csv` 给用户；P2 / P3 默认进 `polish-plan.md` 不打扰任何人。

---

## 2. 走查参照源（Ground Truth）

> 这是本 skill 整套规则的事实底座。所有 detector 必须挂在这五份参照源的某一条事实上，没有挂点的规则禁止落地。

### 2.1 五大参照源

| 序号 | 参照源 | 角色 | 物理位置 | 形态 |
|---|---|---|---|---|
| ① | **本地项目预览** | 真实渲染结果（视觉对账） | `http://localhost:3002/` | dev-server 实时渲染 |
| ② | **全局样式展示台** | 组件标准实现（"应该长这样"） | `client/src/pages/DesignSystemComponents.tsx` | 单文件 React，覆盖全部已规范化组件 |
| ③ | **component-specs/** | 单组件规范（含 ✅/❌ 代码对照） | `.codebuddy/skills/clawpro-portable-design-skill/component-specs/` | 39 份 md，每份含 Purpose / Scope / Visual Standard / Anatomy / Do & Don't |
| ④ | **assets/page-references/** | 真实页面快照 + 组合 recipe | `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/` | 8 份典型页面（md + png）：列表、看板、配置、空态、能力开通等 |
| ⑤ | **portable/css + portable/react** | 跨仓 fallback 的原子实现 | `.codebuddy/skills/clawpro-portable-design-skill/portable/css/` + `portable/react/` | 33 份 CSS + 32 份 React 组件，含 `tokens.css`（设计 token 单一真相源） |

### 2.2 各参照源的查法（detector 怎么用）

| 走查动作 | 主参照 | 辅参照 |
|---|---|---|
| 判某段 className 是否在用合法 token | `portable/css/tokens.css` ⑤ | `component-specs/<comp>.md` ③ |
| 判某个组件用法是否符合规范 | `component-specs/<comp>.md` ③ | `portable/react/<comp>.tsx` ⑤（看正版实现） |
| 判某个页面骨架是否走对组合 | `assets/page-references/<page>.md` ④ | 同 slug 的 `.png` 视觉对照 |
| 判组件视觉细节（圆角 / border / 阴影） | `portable/css/<comp>.css` ⑤ | `DesignSystemComponents.tsx` ② 渲染态 |
| 判生成 / 换皮后的页面是否还原 | 启动 `localhost:3002` ① | 与 `assets/page-references/*.png` ④ 视觉 diff |

### 2.3 参照源 → fixtures（机器可读快照）

走查 skill 不直接读 md，而是从这五份参照源**抽取成机器可读 fixtures**，缓存在 skill 内的 `fixtures/` 目录：

```
clawpro-walkthrough/fixtures/
├── tokens.json                  # 抽自 portable/css/tokens.css ⑤
│   ├── colors: { "--cp-brand-blue": "#1447E6", ... }
│   ├── radius: { "--radius": "4px", ... }
│   └── shadows: { "--cp-shadow-card-hover": "...", ... }
├── component-spec-index.json    # 抽自 component-specs/ ③
│   ├── card-surface: { aliases: [...], specPath, showcaseAnchor, doDont: [...] }
│   └── ...39 组件
├── page-recipes.json            # 抽自 assets/page-references/ ④
│   ├── admin-tokens-monitor: { skeleton: [...], requiredComponents: [...], referenceImage }
│   └── ...8 个典型页面
├── portable-impl-index.json     # 抽自 portable/react + portable/css ⑤
│   └── 每个组件 → { reactPath, cssPath, requiredClasses, forbiddenInline }
└── showcase-anchors.json        # 抽自 DesignSystemComponents.tsx ②
    └── 每个组件在展示台的锚点 id（生成报告时给"对照看正版"链接）
```

**关键设计**：
- fixtures 是**自动生成 + 加哈希**的，源文件一改就 CI 报错，杜绝漂移。
- 所有 finding 的 `evidence` 字段必须指回 fixtures 里的具体路径（如 `tokens.json#colors.--cp-brand-blue`），没有 evidence 的 finding 直接阻断。
- 这一步也是优化计划 A 阶段（审计取证）的第一份产出物。

---

## 3. 整体框架

### 3.1 Skill 结构（管控端走查合并为单一 skill）

```
.codebuddy/skills/
└── clawpro-walkthrough/        # 管控端走查 + 事实底座合并（fixtures / extractors / tools / ci / CSV 规范）
```

> v0.9 起，原 `clawpro-walkthrough-admin`（走查入口）与 `clawpro-walkthrough-core`（fixtures 事实底座）**合并为单一 `clawpro-walkthrough`**，侧边栏只保留一个 skill。客户端走查的 UI 铁律不同（圆角、token、组件参考都不一样），其规则口径另见 portable-design-skill 的 `references/tenant.md`。

### 3.2 目录结构（走查入口部分）

```
clawpro-walkthrough/
├── SKILL.md                          # 入口：触发条件 + 子命令路由 + 红线
├── reference/
│   ├── 走查标准-管控端.md             # 4px 圆角、9 槽位、token、组件参考的检查口径
│   ├── 问题严重等级定义.md            # P0 / P1 / P2 三级
│   ├── 问题类型枚举.md               # 闭枚举：图标槽位违规 / token 漂离 / 圆角误用 / …
│   ├── 常见误报模式.md               # ClawPro 特有误报（alias 命中、聚合 spec 等）
│   ├── CSV走查表规范.md              # 13 列报告 CSV 的列定义与示例
│   └── 待设计裁决清单规范.md          # design-todo.csv 的格式与字段
└── scripts/
    ├── walkthrough.mjs               # 总入口（子命令路由）
    ├── inventory.mjs                 # 候选全集盘存
    ├── coverage.mjs                  # 跑检测器，输出 coverage
    ├── detectors/                    # ★ 静态代码扫描核心
    │   ├── icon-slot.mjs             # 9 槽位 + 不可回退槽位 lucide 违规
    │   ├── radius.mjs                # rounded-xl/2xl/[8px]/[12px] 误用
    │   ├── color.mjs                 # 硬编码 #1447E6 / text-gray-*
    │   ├── shadow.mjs                # inline boxShadow / 非 token 阴影
    │   ├── token-drift.mjs           # 漂离 design token 的内联样式
    │   ├── component-drift.mjs       # 没用 Surface/Card/Button 等基础组件
    │   ├── emoji.mjs                 # 跨行 / 符号区 emoji 盲区
    │   ├── spec-binding.mjs          # 组件实现 ↔ spec 是否能稳定命中
    │   ├── page-recipe-match.mjs     # ★ 换皮后页面骨架是否仍命中 page-references 配方
    │   └── portable-impl-diff.mjs    # ★ 组件实现是否漂离 portable/react 的正版实现
    ├── merge-reports.mjs             # 三级合并：文件 → 页面 → PR
    └── render-report.mjs             # coverage.json → CSV / markdown
```

### 3.3 内置事实底座（原 core，已并入本 skill）

```
clawpro-walkthrough/
├── fixtures/                         # ★ ClawPro 项目真相（机器可读快照，对应 §2.3）
│   ├── tokens.json                   # 抽自 portable/css/tokens.css
│   ├── component-spec-index.json     # 抽自 component-specs/
│   ├── page-recipes.json             # 抽自 assets/page-references/
│   ├── portable-impl-index.json      # 抽自 portable/react + portable/css
│   ├── showcase-anchors.json         # 抽自 client/src/pages/DesignSystemComponents.tsx
│   └── resource-skill-map.json       # 9 槽位 / 158 候选（既有）
├── extractors/                       # ★ 参照源 → fixtures 的抽取脚本
│   ├── extract-tokens.mjs
│   ├── extract-specs.mjs
│   ├── extract-page-recipes.mjs
│   ├── extract-portable.mjs
│   └── extract-showcase.mjs
├── reviewer/                         # 复核 webUI（用户用）
│   ├── review_viewer.py
│   └── review_viewer.html
└── tools/
    └── walkthrough_report_csv.py     # CSV 规范化 + 枚举校验（写完必跑 --fix）
```

---

## 4. 静态代码扫描设计（本方案最核心部分）

### 4.1 为什么是静态扫描

| 维度 | 真实浏览器方案 | 静态代码扫描（本方案） |
|---|---|---|
| 触发时机 | 页面上线后 | **PR 提交前 / 换皮完成立刻** |
| 速度 | 秒级到分钟级（要起浏览器） | **毫秒级**（grep + AST） |
| 取证准确度 | 看到的就是真的 | 取决于规则覆盖度 |
| 覆盖面 | 看得见的视觉问题 | **看得见 + 看不见**（token 漂离、组件参考错位） |
| 成本 | 每次都要起浏览器 | 一次写好规则，永久免费 |

**结论**：你的场景（产品换皮 / 生成新页面）刚好是**改完代码立刻要查**的场景，静态扫描是天然最优解。截图复核只作为 P0 级的可选可视化辅助。

### 4.2 十类 detector（首批必做）

每个 detector 都是独立 mjs 脚本，输入文件 / 目录，输出标准化 finding[]。**每条规则都挂在 §2.1 五大参照源里**，没有挂点禁止落地。

| Detector | 检查什么 | 数据来源（→ fixture） | 参照源 | 等级 |
|---|---|---|---|---|
| **icon-slot** | 9 类不可回退槽位是否仍在用 lucide / 未登记 SVG | `resource-skill-map.json` | 既有 ADR | P0 |
| **radius** | 管控端非 4px 圆角（`rounded-xl/2xl/[8px]/[12px]`） | `tokens.json#radius` | ⑤ `tokens.css` | P0 |
| **color** | 硬编码色值 `#1447E6` / `text-gray-*` 等应走 token 的写法 | `tokens.json#colors` | ⑤ `tokens.css` | P1 |
| **shadow** | inline boxShadow / 非 token 阴影 | `tokens.json#shadows` | ⑤ `tokens.css` | P1 |
| **token-drift** | 内联样式漂离 token（margin / padding / font-size 等） | `tokens.json` | ⑤ `tokens.css` | P2 |
| **component-drift** | 直接用 `<div>` / `<button>` 而非规范组件（Surface / Card / Button …） | `component-spec-index.json` | ③ `component-specs/` | P1 |
| **spec-binding** | 组件实现能否在 spec 索引里稳定命中（不能 → "待设计裁决"） | `component-spec-index.json` | ③ `component-specs/` | 待裁决 |
| **page-recipe-match** ★ | **换皮后页面骨架是否仍命中典型配方**（结构 / 必备组件 / 组合顺序） | `page-recipes.json` | ④ `assets/page-references/` | P1 |
| **portable-impl-diff** ★ | 组件用法是否漂离 portable 正版实现（必需 className / 禁用内联样式） | `portable-impl-index.json` | ⑤ `portable/react` + `portable/css` | P1 |
| **emoji** | 跨行 emoji / 符号区 emoji（补已知盲区） | 内置规则 | — | P2 |

> ★ 标记的两条是**换皮场景**的关键 detector：
> - `page-recipe-match`：产品换皮最容易"换着换着把骨架换没了"——少了 AlertBanner、NumberCard 数量不对、Tabs 顺序错位。这条规则以 ④ `page-references` 的 8 份典型页面为模板做骨架对账。
> - `portable-impl-diff`：换皮时最容易"看着像、其实把 className 都改成了内联样式"。这条规则以 ⑤ `portable/react/<comp>.tsx` 为正版，对比当前实现里 `Card / Button / Table` 等组件的 className / 内联 style 是否漂离。

### 4.3 检测器输出 schema（统一约束）

每个 detector 必须输出符合下面 schema 的 finding：

```json
{
  "ruleId": "icon-slot-lucide-in-metric-card",
  "severity": "P0",
  "type": "图标槽位违规",
  "file": "client/src/pages/admin/tokens-monitor.tsx",
  "line": 268,
  "column": 12,
  "snippet": "<Zap className=\"h-5 w-5\" />",
  "message": "metric-card 槽位禁止使用 lucide 图标",
  "evidence": "resource-skill-map.json#metric-card",
  "suggestion": "改用 assets/icons/metric/quota-zap.svg",
  "showcaseAnchor": "http://localhost:3002/design-system#card-surface",
  "referenceImage": ".codebuddy/.../page-references/admin-tokens-monitor.png",
  "needsDesign": false,
  "autoFix": null
}
```

**关键设计**：
- `ruleId` 全局唯一 → 可被 `$walkthrough explain <ruleId>` 反查原文
- `evidence` 必须指向 `fixtures/` 里的某一条事实 → 杜绝 AI "凭印象判违规"
- `showcaseAnchor` 直接跳转到 ② 全局样式展示台对应组件（http://localhost:3002/design-system#...），让 AI / 用户"对照看正版"零成本
- `referenceImage` 关联 ④ `page-references/*.png`，给用户视觉对账
- `needsDesign: true` 的 finding **不进 audit-report.csv，直接进 design-todo.csv**

### 4.4 Inventory + Coverage 双盘存（硬约束）

```
1. inventory.mjs 扫一遍目标文件，列出"所有可走查点"
   - 每个 className、每个 import、每个组件使用、每个内联 style
   - 输出 inventory.json（候选全集）

2. coverage.mjs 跑所有 detector
   - 每个候选点必须被某个 detector 标记为：
     - covered（命中规则 → 生成 finding）
     - skipped-pass（命中规则但符合规范 → 不入报告但留痕）
     - skipped-out-of-scope（不在任何 detector 关心范围 → 留痕）

3. 硬校验：coverage.length === inventory.length
   - 不通过 → 阻断报告生成
```

这一条让"AI 漏查"从可能变成不可能。

---

## 5. 子命令设计

按"三段闭环"组织，每段一组命令，外加共用工具命令。

### 5.1 三段闭环命令（主用法）

| 命令 | 用法 | 产出 | 是否独立 |
|---|---|---|---|
| `$walkthrough critique <target>` | 设计走查（AI 视觉视角） | `critique-report.md` + 设计健康分 | 独立 |
| `$walkthrough audit <target>` | 技术审计（10 类 detector） | `audit-report.csv` + 技术健康分 | 独立（与 critique 互盲） |
| `$walkthrough polish <target>` | 综合 + 收口 | `polish-plan.md` + `design-todo.csv` + 总分 | 依赖前两步 |
| `$walkthrough <target>` | 一键跑三段 | 上述三份全产出 + `trend.json` | 默认入口 |

### 5.2 增量与单规则（高频用法）

| 命令 | 用法 | 产出 |
|---|---|---|
| `$walkthrough diff` | 对当前 `git diff` 走完 audit 一段 | 增量 audit 报告（**PR 提交前最高频**） |
| `$walkthrough icon <file>` | 只跑 icon-slot detector | 图标违规清单 |
| `$walkthrough radius <file>` | 只跑 radius detector | 圆角违规清单 |
| `$walkthrough recipe <page>` | 只跑 page-recipe-match | 页面骨架对账结果 |

### 5.3 工具命令

| 命令 | 用法 | 产出 |
|---|---|---|
| `$walkthrough explain <ruleId>` | 解释某条规则为什么 | 规则原文 + 来源（SKILL.md / 参照源锚点） |
| `$walkthrough merge` | 合并多份子报告 | PR 级合并报告 |
| `$walkthrough trend <slug>` | 打印某条页面最近 5 次走查总分曲线 | `28 → 33 → 41 → 39 → 47 ↑` |
| `$walkthrough refresh-fixtures` | 重抽 fixtures（参照源有更新时跑） | 新版 fixtures + 哈希记录 |

---

## 6. 产出物（两张表 + 一个 webUI）

### 6.1 表 1：`walkthrough-report.csv`（AI 直接修的）

13 列闭枚举 CSV（沿用我们内部已经验证过的成熟设计）：

| 列名 | 类型 | 必填 |
|---|---|---|
| 报告 ID | 文本 | ✓ |
| 所属页面 | 文本 | ✓ |
| 文件路径 | 文本 | ✓ |
| 行号 | 数字 | ✓ |
| 问题类型 | 枚举 | ✓ |
| 严重等级 | P0 / P1 / P2 | ✓ |
| 问题描述 | 文本 | ✓ |
| 命中规则 ID | 文本 | ✓ |
| 证据来源 | 文本（fixtures 锚点） | ✓ |
| 修复建议 | 文本 | ✓ |
| 自动修复 | bool | ✓ |
| 解决状态 | 枚举（待修复 / 已修复 / 误报） | ✓ |
| 备注 | 文本 |  |

未知枚举 → `walkthrough_report_csv.py --fix` 直接阻断。

### 6.2 表 2：`design-todo.csv`（★ 给用户拍板的，本方案独有）

```csv
所属页面,槽位/位置,问题描述,AI 当前处理,建议,展示台对照,真实页面参照,用户裁决
admin/tokens-monitor,metric-card 图标槽位,资源映射里没匹配候选,降级到 lucide Zap,补 metric-card 专用 SVG,localhost:3002/design-system#number-card,page-references/admin-tokens-monitor.png,待裁决
admin/notice-bar,阴影未命中 token,inline boxShadow,新增 --shadow-notice 档位,localhost:3002/design-system#alert,page-references/admin-ops-observation.png,待裁决
```

用户只看这一张表，不用看代码、不用看 PR、不用进 Figma。**每条都附"展示台对照"链接 + "真实页面参照"图**，30 秒做完决策。

### 6.3 复核 webUI：`review_viewer.py`

- 缩略图列表 + 一键改 P0/P1/P2
- "AI 建议采纳 / 驳回 / 补充意图" 下拉
- 内嵌 iframe 直接打开"展示台对照"地址，左右对比
- 保存即合并、即覆写 design-todo.csv
- 顶部固定显示**健康分趋势曲线**（取自 `_walkthrough/snapshots/<slug>/trend.json`），让用户/老板第一眼看到这条页面在变好还是变差

### 6.4 走查快照：`_walkthrough/snapshots/<slug>/`

每次走查写一份带时间戳的完整快照，沉淀历史，支撑趋势曲线：

```
_walkthrough/snapshots/admin-tokens-monitor/
├── 20260628-1620.md           # critique + audit + polish 的完整合并报告
├── 20260629-1130.md
└── trend.json                 # 历次健康分（critique / audit / total）
```

- 快照只读、按时间归档，不参与 PR 合并。
- `$walkthrough trend` / 复核 webUI 都从这里读，**这是健康分系统的唯一真实来源**。

---

## 7. 红线（脚本层硬拦）

| 红线 | 实现方式 |
|---|---|
| 不擅自改业务代码 | detector 扫描只读；任何改动必须先出清单、逐条经用户确认，报告产物只写到 `_walkthrough/` 目录 |
| 不私自下设计结论 | `needsDesign: true` 的 finding 强制进 design-todo.csv |
| 不引入规则漂移 | 所有规则必须挂在 `fixtures/` 的某一条事实上，`evidence` 字段为空直接阻断 |
| 不产出脏 CSV | 写完必跑 `walkthrough_report_csv.py --fix`，未知枚举阻断 |
| 不脱离参照源 | fixtures 每次抽取记录源文件哈希，参照源变了不重抽就阻断报告生成 |

---

## 8. 与"ClawPro Skill 优化计划"的衔接

| 优化计划阶段 | 本 skill 怎么承载 |
|---|---|
| 阶段 A · 审计取证 | 先做 `fixtures/` 抽取 + 跑 `$walkthrough --audit-only`，产出 A1 / A2 / A3 的差异表 |
| 阶段 B · 消除自相矛盾 | 不直接承载，但 fixtures 是 B 阶段改 SKILL.md 的事实底座 |
| 阶段 C · 补机制 | **C1（脚本拦截）= detectors/，C2（待设计清单）= design-todo.csv，C3（组件参考路由）= component-spec-index** |
| 阶段 D · 固化纪律 | SKILL.md §1 / §12 增一句：「PR 合并前必须跑 `$walkthrough diff`，P0 必须清零」 |

> ⚠️ 关键约束：本 skill 的 `fixtures/` 必须以优化计划**阶段 A 的真实产出**为准。在 A 完成前，先做 `0.1-audit-only` 版本，只跑扫描出报告、不下结论、不做拦截。

---

## 9. 版本规划

| 版本 | 能力 | 时间预估 |
|---|---|---|
| **0.1 audit-only** | core fixtures 抽取（5 个 extractor）+ radius / color / icon-slot 三个 detector + `$walkthrough audit` + `audit-report.csv` | 1 周 |
| **0.5 minimum-loop** | 全部 10 个 detector + `$walkthrough critique`（视觉视角）+ `$walkthrough polish`（综合）+ `design-todo.csv` + `$walkthrough diff` | 2 周 |
| **1.0 full-loop** | 复核 webUI + 三级合并 + tenant skill + 误报清单沉淀 + 展示台 iframe 对照 + **快照持久化 + 健康分趋势曲线** | 3 周 |
| **1.5 enforce** | PR 阻断（P0 不清零不让合并）+ `$walkthrough explain` + **双独立评估（critique / audit 子 agent 互盲）** + 截图 diff（与 page-references/*.png 对账） | 后续迭代 |

---

## 10. 风险与对策

| 风险 | 对策 |
|---|---|
| 误报率高 → 用户反而要纠误报，没省力 | 1) `reference/常见误报模式.md` 沉淀已知模式 2) 复核 webUI 的"驳回"动作要回流成新误报规则 |
| fixtures 与代码现状漂移 | A 阶段产出的 fixtures 加版本号 + CI 检查 fixtures 哈希是否与源文件一致 |
| 规则越加越多最后没人维护 | 每条规则必须挂 `evidence` 字段；定期跑 `--unused` 删掉 30 天未命中的规则 |
| 静态扫描覆盖不到的视觉问题（如对比度、间距视觉手感） | 1.5 版本引入可选截图复核，但不强制 |

---

## 11. 一句话总结

> **clawpro-walkthrough 是一个走查后先出清单、逐条经用户确认才改代码的设计走查 skill：以"本地预览 + 展示台 + component-specs + page-references + portable 实现"这五份参照源为事实底座，抽成机器可读 fixtures；按 critique → audit → polish 三段闭环、双独立评估的方法论组织流程，用 10 类 detector + AI 视觉双轨打分，结果落成 audit-report.csv（AI 直接修）+ design-todo.csv（给用户）+ 健康分趋势曲线（给老板）。把用户从"看代码"解放到"只看清单"，把"有没有改好"从主观感受变成可对账的数字。**
