---
name: clawpro-walkthrough
description: ClawPro 管控端设计走查 skill。对管控端代码做静态扫描，输出 audit-report.csv + design-todo.csv。扫描只读；发现问题后列出清单，逐条向用户确认后再修改业务代码。当用户提交管控端 PR、生成新管控端页面、对管控端页面换皮，或显式说"走查 / 审查 / 审计 / 设计走查 / audit / walkthrough"时触发。关键词：管控端、admin、PR 走查、换皮检查、设计走查、PR 检查、token 漂离、圆角检查、9 槽位图标、ClawPro 走查、$walkthrough。
location: project
---

# clawpro-walkthrough（v0.8 · critique→audit→polish 三段闭环）

> 📌 **审查 SOP（必读，先于一切子命令）**：本 skill 是「警察」，`clawpro-portable-design-skill` 才是「法律」。任何 audit / critique 都必须以设计 skill 的规范原文为准，脚本只负责机械命中。下面 §0.A / §0.B / §0.C 是两边的契约。
>
> **v0.8（2026-06-30）更新** —— 推进到 roadmap 0.5 minimum-loop：
> - **补齐 3 个缺位 detector**：`text-color`（§2.5 文字色）/ `surface-nesting`（§7.4 Surface 套娃）/ `spacing-grouping`（§2.7 间距成组）—— §0.B 表里 3 个 ❌ 全部转 ✅；
> - **落地 design-todo.csv 分流**：finding 带 `stream`（audit/todo）+ `conflictType`；确定违规进 `audit-report.csv`，待裁决进 `design-todo.csv`（含 `needs-design-confirmation` 标记、间距成组判断题、critique-only 视觉项），闭环 §0.C；
> - **新增 critique / polish / trend 命令** + 健康分模型（audit 8 维 ×4=32 / critique 6 维 ×4=24 / 总分 56），见 §2 / §8；
> - **修复外部 detector 静默失效**：`check-design-usage.mjs --json` 写大块 JSON 后 `process.exit` 在管道场景会被截断 → wrapper 改走临时文件 fd 读取，外部 2900+ 条覆盖恢复（详见 `external-design-skill.mjs` 注释）。
>
> **v0.7（2026-06-30）基线**：同步同事 miekoyychen MR !123 / !129 / !133 的设计 skill 治理（A/B/C/D 四阶段）：§1.5 冲突铁律 + §12 完整版、4 个治理脚本、2 个参考、`check-design-usage.mjs` 升到 8 个 check（第 8 个 `collectDesignConfirmationMarkers` 收集 `needs-design-confirmation`）。

## 安装与依赖（对外交付必读）

- **配套强依赖**：本 skill 必须与 `clawpro-portable-design-skill`（设计规范"法律原文" + 复用其检测脚本）**一起安装**。两个 skill 并列放进同一个 `.codebuddy/skills/` 目录即可，默认按同级目录相互发现。安装步骤见 `INSTALL.md`。
  - 未安装设计 skill 时：内置 9 类 detector 仍可跑（规则快照已固化在 `fixtures/`），但 `external/*` 一类会**静默跳过**，走查覆盖面下降。
- **运行环境**：Node ≥ 18（`scripts/`）；可选 Python 3（`tools/walkthrough_report_csv.py`，用于 CSV 校验）。
- **可配置项（环境变量）**：
  - `WALKTHROUGH_PROJECT_ROOT`：覆盖被扫描工程的仓库根（默认按标准安装位置 `<repo>/.codebuddy/skills/clawpro-walkthrough` 反推）。
  - `WALKTHROUGH_DESIGN_SKILL_DIR`：设计 skill 装到别处 / 改名时，指向其根目录。
  - `WALKTHROUGH_OUT_DIR`：覆盖产物输出目录（默认 `<repo>/_walkthrough`）。
  - `WALKTHROUGH_SKIP_EXTERNAL=1`：整体跳过 `external/*` detector。
- **扫描目标必填**：`audit / critique / polish / <单规则>` 均需显式传入目标文件或目录，**不再有 `client/src` 默认值**。

## 0.A 强制阅读清单（跑 `$walkthrough` 之前 AI 必须做的事）

| 时机 | 必读 | 用途 |
|---|---|---|
| **任何 audit / critique 之前** | `clawpro-portable-design-skill/SKILL.md` §1.5「冲突处理铁律」+ §12「冲突处理完整版」 | **最高优先级**：理解"冲突回用户"原则，避免 AI 私自裁决 spec vs 现状的冲突 |
| 跑 `audit` / `diff` 之前 | `clawpro-portable-design-skill/SKILL.md` §0 Scope、§2 Critical Rules（✅/❌ 对照）、§8 Self-Audit Checklist | 让 AI 用"用户视角"看代码，而不是只看 detector 命中行 |
| 跑 `radius` / `color` / `shadow` 之前 | 同上的 §2.1 / §2.4 / §7.2 + `tokens.css` | 知道每条规则**为什么是这条值**，避免机械建议 |
| 跑 `icon-slot` 之前 | 同上的 §2.8 + `references/assets-icons.md §5.5` + `fixtures/icon-slots.json` + `references/icon-design-todo.md` | 知道 9 槽位的尺寸 / 候选 / 不可降级理由 + 进行中的图标冲突清单 |
| 跑 `component-drift` 之前 | 同上的 §2.3 / §2.6 + 对应组件 `component-specs/<comp>.md` + `references/component-mapping.md` | 知道 variant 白名单背后的视觉差异 + 42 个组件的 spec/react/css 三层映射 |
| 写 critique 报告之前（v1.0 后） | 上述全部 + `assets/page-references/<page>.md` | 让 critique 的语言对齐设计 skill 的术语 |

## 0.B 走查规则 ↔ 设计规范锚点对照表（admin walkthrough ↔ portable-design 的契约）

| Self-Audit (设计 skill §8) | 走查 detector（本 skill） | 设计 skill 章节锚点 | 实现状态 |
|---|---|---|---|
| 1️⃣ 图标槽位用对、无违规 lucide、禁回退 | `icon-slot` + `external/prohibited-lucide-slot` | §2.8（含禁回退策略，2026-06-30 细化）+ `assets-icons.md §5.5` + `fixtures/icon-slots.json` + `references/icon-design-todo.md` | ✅ v0.7 |
| 2️⃣ 没有硬编码颜色 | `color` + `external/css-hardcoded-color-hex` + `external/semantic-tailwind-color` | §2.1 + §7.2 + `tokens.css` | ✅ v0.6 |
| 3️⃣ 没有硬编码圆角 / 管控端 4px | `radius` + `external/css-large-radius` | §2.4（铁律）+ §0.1 | ✅ v0.6 |
| 4️⃣ 间距用 flex+gap | `spacing-grouping`（→ design-todo） | §2.7 | ✅ v0.8（判断题进 design-todo） |
| 5️⃣ Empty 用 Empty 系列 | `component-drift`（部分） | §2.6 + `component-specs/empty-state.md` | 🟡 部分 v0.6 |
| 6️⃣ Surface 没有套娃 | `surface-nesting` | §7.4 | ✅ v0.8 |
| 7️⃣ 场景识别正确（Admin/Tenant 不混） | `component-drift`（部分）| §0 Scope + §7.3 | 🟡 部分 v0.6 |
| 8️⃣ 按钮 variant 不借用 shadcn outline | `component-drift` | §2.3 + `component-specs/button.md` | ✅ v0.6 |
| 9️⃣ 文字色走 Typography | `text-color` | §2.5 | ✅ v0.8 |
| ✱ spec 与代码无 ghost reference | `external/spec-symbol-ghost` | `component-specs/*.md` + `client/src/` | ✅ v0.6（外部 detector） |
| ✱ 阴影走 token | `shadow` | §2 + `tokens.css#shadows` | ✅ v0.6 |
| ✱ 页面骨架完整 | `page-recipe-match` | `assets/page-references/*.md` | ✅ v0.6 |
| ✱ 冲突标记 `needs-design-confirmation` 不视作违规 | `external/*` 系列（由 `collectDesignConfirmationMarkers` 收口） | §1.5 + §12 + `references/conflict-log.md` + `references/icon-design-todo.md` | ✅ v0.7（外部 detector 第 8 个 check） |

> 这张表的硬约束：**新增 detector 必须填一行**；不能填的（无对应 Self-Audit / 无设计 skill 章节锚点）说明规则没立法，禁止落地。

## 0.C 冲突上报流程（落实设计 skill §1.5 / §12）

走查脚本扫到"可能违规但不确定是 spec 漂离还是业务需求"时，**AI 不能私自定性**，必须按下面流程把决策交回用户：

1. **这一条拿不准的不自动改**：只把它写进 `_walkthrough/` 的清单，交用户裁决，不擅自定性或改动。
2. **进 design-todo.csv**：走查报告里这一条不在 audit-report.csv（确定违规），而是放到 design-todo.csv（待裁决）。
3. **挂冲突类型**：参照设计 skill §12.3 的 5 类（Spec vs 现状 / 需求超范围 / 图标无候选 / Token 模糊 / 宿主仓兼容），在 design-todo 的 `conflict_type` 字段标明。
4. **建议同步写 `conflict-log.md`**：critique 报告里提示用户「请在 `clawpro-portable-design-skill/references/conflict-log.md` 追加冲突条目，按 §1.5 模板填写」。
5. **代码里见 `needs-design-confirmation` 标记 → 放过**：该标记表示"用户已知晓，正在等裁决"，走查脚本（含外部 detector 第 8 个 check `collectDesignConfirmationMarkers`）会自动跳过，不视作违规。

## 0. 角色与红线

- **先确认再改**：扫描本身只读；发现问题先出清单，逐条向用户确认"是否修改"，用户同意后才改业务代码；报告产物统一写到 `_walkthrough/` 目录。
- **静态扫描为主**：不依赖浏览器，毫秒级出报告。
- **挂点强制**：每条 detector 必须挂在 `clawpro-walkthrough/fixtures/` 里的具体事实，没有挂点的规则禁止落地。
- **不下结论**：AI 拿不准的问题，必须进 `design-todo.csv` 留给用户裁决，**严禁私自定性**。

## 1. 触发场景

| 场景 | 用户原话样例 | 跑什么 |
|---|---|---|
| PR 提交前增量走查 | "审一下这个 PR" / "走查一下我刚改的" | `$walkthrough diff` |
| 换皮后全量走查 | "换完皮帮我审一遍" / "audit 一下" | `$walkthrough audit <target>` |
| 单条规则定点查 | "查一下圆角" / "看下硬编码色值" | `$walkthrough radius <file>` / `$walkthrough color <file>` |
| 设计视觉视角复核 | "用用户视角审一遍" / "打个设计健康分" | `$walkthrough critique <target>` |
| 综合收口 + 健康分 | "把这次走查收口成清单" | `$walkthrough polish <target>` |
| 走查规则解释 | "为什么这条算违规" | `$walkthrough explain <ruleId>` |

> v0.8 已实现 audit 轨 9 类 detector（含 v0.8 新增 3 类）+ critique / polish / trend + diff / explain。

## 2. 子命令（v0.8）

**audit 轨**（机器静态规则 → `audit-report.csv` + `design-todo.csv`）：

| 命令 | 用途 | 产物 |
|---|---|---|
| `$walkthrough audit <target>` | 跑全部 detector | `audit-report.csv` + `design-todo.csv` + `meta.json` |
| `$walkthrough radius <target>` | 只跑 radius | 同上 + 过滤 ruleId=radius |
| `$walkthrough color  <target>` | 只跑 color | 同上 |
| `$walkthrough icon-slot <target>` | 只跑 icon-slot | 同上 |
| `$walkthrough shadow <target>` | 只跑 shadow | 同上 |
| `$walkthrough component-drift <target>` | 只跑 component-drift | 同上 |
| `$walkthrough page-recipe-match <target>` | 只跑 page-recipe-match | 同上 |
| `$walkthrough text-color <target>` | 只跑 文字色 Typography（v0.8） | 同上 |
| `$walkthrough surface-nesting <target>` | 只跑 Surface 套娃（v0.8） | 同上 |
| `$walkthrough spacing-grouping <target>` | 只跑 间距成组（v0.8，→ design-todo） | 同上 |
| `$walkthrough diff` | 对当前 git diff 跑全部 detector | 同上，只覆盖已改动文件 |

**critique / polish 轨**（视觉视角 + 综合收口，与 audit 互盲，DESIGN §1.5.1）：

| 命令 | 用途 | 产物 |
|---|---|---|
| `$walkthrough critique <target>` | 生成 AI 视觉视角评分脚手架（6 维 ×4=24，须独立于 audit 填写） | `critique-report.md` + `critique.json` |
| `$walkthrough polish <target>` | 合并最近一次同 slug 的 audit + critique，出收口清单 + 总分 | `polish-plan.md` + 合并版 `design-todo.csv` |
| `$walkthrough trend [slug]` | 打印最近 5 次健康分趋势 | 终端输出（读 `snapshots/<slug>/trend.json`） |
| `$walkthrough explain <ruleId>` | 打印规则定义 + 参照源锚点 | 终端输出 |

## 3. audit 轨 detector（v0.8 共 9 类内置 + 1 类外部）

| ruleId | 检查什么 | 数据来源 | 严重等级 | 落表 |
|---|---|---|---|---|
| `radius` | 管控端非 4px 圆角（`rounded-xl/2xl/3xl/[8px]/[12px]` / `border-radius: 8px` 等） | `fixtures/tokens.json#radius` | P0 | audit |
| `color`  | 硬编码颜色（`#1447E6` / `#FFFFFF` / `rgba(...)`）应走 token | `fixtures/tokens.json#colors` | P1 | audit |
| `icon-slot` | 在 9 类不可回退槽位仍在用 `lucide-react` 图标 | 内置 9 槽位枚举（来自 SKILL.md §0） | P0 | audit |
| `shadow` | 阴影必须走 `--cp-shadow-*`，禁止硬编码 / 自创 var / Tailwind 框架级 `shadow-md/lg/...` | `fixtures/tokens.json#shadows` | P1 / P2 | audit |
| `component-drift` | tsx 中 `<X variant="y">` 的 `y` 必须落在该组件 spec 的 variants 白名单（兜底放行 shadcn 内置 variant） | `fixtures/component-spec-index.json#<id>.variants` | P2 | audit |
| `page-recipe-match` | page entry + 同目录 bundle 必须 import recipe 登记的全部 `required_components` | `fixtures/page-recipes.json#<id>.required_components` | P2 | audit |
| `text-color`（v0.8） | 文字色用了 Tailwind 内置中性灰阶（`text-gray-500` 等），应走 Typography 语义 / `--cp-text-*` | 设计 skill §2.5 + `tokens.json#colors(--cp-text-*)` | P2 | audit |
| `surface-nesting`（v0.8） | Surface 容器套娃（`SurfaceCard` 套 `SurfaceCard` / `TenantCard` 自嵌套），内层须降 `SurfaceInner` | 设计 skill §7.4 | P2 | audit |
| `spacing-grouping`（v0.8） | 相邻控件各自加水平 margin 疑似未成组，应改 flex+gap（**判断题**） | 设计 skill §2.7 | P2 | **design-todo** |
| `external/*` | 复用 `check-design-usage.mjs`（8 check）+ `check-spec-symbols.mjs`，含 `needs-design-confirmation` 收集 | 设计 skill `scripts/` | P0~P2 | audit + design-todo |

## 4. 产物结构

```
_walkthrough/
├── 20260630-1757/                  # 每次跑一份带时间戳的快照
│   ├── audit-report.csv            # 确定违规清单（AI 直接修，stream=audit）
│   ├── audit-report.json           # 结构化原始数据
│   ├── design-todo.csv             # 待裁决清单（给用户，stream=todo，含 conflict_type）
│   ├── design-todo.json            # 待裁决结构化数据（polish 合并用）
│   ├── critique-report.md          # critique 评分脚手架（critique 命令产）
│   ├── critique.json               # critique 6 维分数 + critique-only todos
│   ├── polish-plan.md              # 收口清单 + 健康分（polish 命令产）
│   └── meta.json                   # 入参 + bySeverity + auditHealth + slug
└── snapshots/
    └── <slug>/
        └── trend.json              # 历次健康分（audit / critique / total），trend 命令读这里
```

## 5. CSV 列定义

### 5.1 `audit-report.csv`（AI 直接修，stream=audit）

```
ruleId, severity, file, line, col, snippet, message, evidence, suggestion
```

- `ruleId`：枚举值，见 §3 表
- `severity`：`P0` / `P1` / `P2`
- `file` / `line` / `col`：定位
- `snippet`：原文片段（≤80 字符）
- `message`：一句话说清出了什么问题
- `evidence`：指回 fixture 的锚点（如 `tokens.json#radius.--radius=4px`）
- `suggestion`：可执行修复建议（如 `改为 rounded-[var(--radius)] 或 rounded`）

### 5.2 `design-todo.csv`（给用户拍板，stream=todo；DESIGN §6.2 + §0.C conflict_type）

```
冲突类型, 所属页面, 槽位/位置, 问题描述, AI 当前处理, 建议, 展示台对照, 真实页面参照, 用户裁决
```

- `冲突类型`：设计 skill §12.3 的 5 类（Spec vs 现状 / 需求超范围 / 图标无候选 / Token 模糊 / 宿主仓兼容），判不准 → `待分类`
- 来源三类：① `spacing-grouping` 间距成组判断题；② 代码里的 `needs-design-confirmation` 标记；③ critique-only 视觉项（polish 合并）
- `用户裁决`：默认 `待裁决`，裁决后请在 `clawpro-portable-design-skill/references/conflict-log.md` 留痕

> **铁律**：AI 拿不准的问题只进 design-todo.csv，**不进 audit-report.csv，也不参与退出码 / 健康分扣分**。

### 5.3 健康分（DESIGN §1.5.2）

- **audit 健康分**：8 维 × 0~4 = 32，按各维命中数评分（维度内有 P0 封顶 1 分）；只统计 stream=audit，待裁决不扣分。
- **critique 健康分**：6 维 × 0~4 = 24，由 AI 视觉视角独立评分（与 audit 互盲），填回 `critique.json`。
- **总分** = audit + critique（满分 56），评分带：50–56 Excellent / 40–49 Good / 28–39 Acceptable / 16–27 Poor / 0–15 Critical。
- `polish` 才把两轨合并出总分；`trend` 读 `snapshots/<slug>/trend.json` 看曲线。

## 6. 跑法

```bash
cd .codebuddy/skills/clawpro-walkthrough

# 全量审计某个页面 / 目录（目标必填，路径按你的工程结构；默认不阻断退出码）
node scripts/walkthrough.mjs audit <你的工程>/src/pages/Some.tsx

# 当前 git diff 增量审计（默认 P0 命中 exit 1，可用作 CI）
node scripts/walkthrough.mjs diff

# critique→audit→polish 三段闭环（critique 与 audit 互盲）
node scripts/walkthrough.mjs audit    <你的工程>/src/pages/Some.tsx
node scripts/walkthrough.mjs critique <你的工程>/src/pages/Some.tsx   # 生成脚手架
# → 按 §0.A 读规范，填 critique-report.md 分数并回填 critique.json
node scripts/walkthrough.mjs polish   <你的工程>/src/pages/Some.tsx   # 合并出总分 + design-todo.csv
node scripts/walkthrough.mjs trend    Some                          # 看健康分趋势

# 若 skill 未装在标准位置（<repo>/.codebuddy/skills/），显式指定被扫描仓库根：
# WALKTHROUGH_PROJECT_ROOT=/path/to/your/repo node scripts/walkthrough.mjs audit /path/to/your/repo/src/pages/Some.tsx
```

## 7. 退出码（CI 用）

| 子命令 | 退出码 | 默认行为 |
|---|---|---|
| `diff` | 0 全绿 / 1 命中 ≥ 阻断阈值 / 2 异常 | **P0 命中阻断**（CI 主用） |
| `audit <target>` | 默认始终 0（不卡历史存量） | 设 `WALKTHROUGH_BLOCK_ON_AUDIT=1` 才启用退出码 |
| 单规则（`radius` / `color` / ...） | 同 `diff` | 命中即 exit 1 |
| `explain` | 0 / 1（未知 ruleId） | — |

环境变量：
- `WALKTHROUGH_BLOCK_LEVEL`：`P0` (默认) / `P1` / `P2` / `NONE`
- `WALKTHROUGH_BLOCK_ON_AUDIT`：`1` 时全量 audit 也启用退出码

完整 CI 接入方案见 `.codebuddy/skills/clawpro-walkthrough/ci/README.md`。
