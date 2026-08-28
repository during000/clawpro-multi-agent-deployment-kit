# ClawPro 走查 Skill 使用教程（clawpro-walkthrough）

> 一句话：把"用设计 skill 生成 / 换皮页面后，必须人工逐页审一遍"这件事，从**纯人力审核**改成**机器走查 + 人只看 AI 拿不准的那一小撮**。

---

## 一、这是什么

`clawpro-walkthrough` 是 ClawPro 管控端的**设计走查 skill**。它对管控端代码（`.tsx` / `.css`）做**毫秒级静态扫描**，自动找出"圆角不对、硬编码色值、图标槽位用错、Surface 套娃、组件漂离规范……"这类问题，产出两份结构化清单和一个可追踪的设计健康分。

它和设计规范的关系是一句话原则：

> **规范是法律，脚本是警察。**
> `clawpro-portable-design-skill` 是"法律"（规范原文），`clawpro-walkthrough` 是"警察"（只负责机械命中、取证、出清单）。任何一条走查结论都能回溯到规范里的具体章节。

---

## 二、解决什么痛点

| 旧流程（纯人力） | 用走查 skill 之后 |
|---|---|
| 每次生成新页面 / 换皮，都要人肉逐页对照规范审一遍 | 一条命令毫秒级出全量清单，人只复核"AI 拿不准的" |
| "我审过了"是口头承诺，没法验证 | inventory + coverage 双盘存，**"查过"变成机器可校验的事实** |
| 问题散在脑子里 / 聊天记录里 | 沉淀为带定位、带证据、带修复建议的 CSV |
| 有没有越改越好？说不清 | 设计健康分 + 趋势曲线，量化"有没有变好" |

核心收益：**把人力从"100% 看代码"降到"只看清单里 AI 标红的争议项"**。

---

## 三、核心价值点（对外介绍重点）

1. **静态扫描，毫秒级、零依赖**
   不启动浏览器、不跑构建，纯代码静态分析。适合 PR 卡口、本地随手查、CI 常态巡检。

2. **规范驱动，每条结论可回溯**
   detector 全部挂在 `fixtures/`（从 `tokens.css` / `component-specs` / 9 图标槽位等参照源抽出的机器快照）上。没有规范锚点的规则**禁止落地**——不存在"AI 拍脑袋的规则"。

3. **"先确认再改"，不静默动业务代码**
   扫描本身**只读**；发现问题先出清单，逐条向用户确认"是否修改"，用户同意后才改业务代码。报告产物统一写到 `_walkthrough/` 目录，不污染工程。

4. **确定违规 / 待裁决两条流分开**
   - 机器能确定的违规 → `audit-report.csv`（AI 直接修）
   - AI 拿不准（规范漂离 or 业务需求？）→ `design-todo.csv`（**交用户裁决，严禁 AI 私自定性**）
   这条铁律保证工具不会"越权拍板"。

5. **critique → audit → polish 三段闭环**
   把一次完整走查拆成三段、职责单一、可独立调用，且 critique（视觉视角）与 audit（技术规则）**互盲**，避免互相锚定。

6. **健康分 + 趋势，可度量、可追踪**
   audit 32 分 + critique 24 分 = 总分 56，分档（Excellent / Good / Acceptable / Poor / Critical），并留存历次快照看曲线。

7. **能进 CI 做卡口**
   `diff` 命令默认 P0 命中即阻断（exit 1），可直接接蓝盾 / 工蜂流水线，把"设计红线"变成合码前的硬门槛。

---

## 四、快速上手

### 4.1 怎么触发（对话式）

直接对 AI 说下面这类话即可命中：

| 你想干嘛 | 说这句 | 实际跑 |
|---|---|---|
| 审刚改的 PR | "审一下这个 PR" / "走查一下我刚改的" | `$walkthrough diff` |
| 换皮后全量审 | "换完皮帮我审一遍" / "audit 一下" | `$walkthrough audit <目标>` |
| 只查一条规则 | "查一下圆角" / "看下硬编码色值" | `$walkthrough radius/color <文件>` |
| 视觉视角复核 | "用用户视角审一遍" / "打个设计健康分" | `$walkthrough critique <目标>` |
| 收口成清单 + 总分 | "把这次走查收口成清单" | `$walkthrough polish <目标>` |
| 问某条为什么违规 | "为什么这条算违规" | `$walkthrough explain <ruleId>` |

关键词：`管控端 / admin / PR 走查 / 换皮检查 / 设计走查 / token 漂离 / 圆角检查 / 9 槽位图标 / $walkthrough`。

### 4.2 怎么跑（命令行）

```bash
cd .codebuddy/skills/clawpro-walkthrough

# 1) 全量审计某个页面 / 目录（目标必填，路径按你的工程结构；默认不阻断退出码）
node scripts/walkthrough.mjs audit <你的工程>/src/pages/Some.tsx

# 2) 只审当前 git diff（默认 P0 命中 exit 1，用于 CI）
node scripts/walkthrough.mjs diff

# 3) 完整三段闭环
node scripts/walkthrough.mjs audit    <你的工程>/src/pages/Some.tsx
node scripts/walkthrough.mjs critique <你的工程>/src/pages/Some.tsx   # 生成评分脚手架
#   → 按 §0.A 读规范，独立填 critique-report.md 分数并回填 critique.json
node scripts/walkthrough.mjs polish   <你的工程>/src/pages/Some.tsx   # 合并出总分 + design-todo.csv
node scripts/walkthrough.mjs trend    Some                          # 看健康分趋势
```

> 目标路径必填：为便于对外交付，已移除 `client/src` 默认目标。若本 skill 未装在标准位置
> `<repo>/.codebuddy/skills/`，用 `WALKTHROUGH_PROJECT_ROOT=<repo>` 指定被扫描仓库根。

---

## 五、三段闭环怎么理解

> 一次完整走查 = critique + audit + polish 三段相加。每段职责单一、产物单一、可独立调用。

| 段 | 视角 | 关注什么 | 主产出 |
|---|---|---|---|
| **critique** · 设计走查 | 设计师视角 | 视觉手感、信息层级、AI slop 痕迹、文案、组件选型 | `critique-report.md`（6 维 ×4 = 24 分） |
| **audit** · 技术审计 | 前端 reviewer 视角 | token / 圆角 / 阴影 / 图标槽位 / 组件是否漂离正版 | `audit-report.csv`（9+1 类 detector） |
| **polish** · 上线打磨 | 用户 + AI 合并视角 | 把两段的 P0/P1 收口成可执行清单 | `polish-plan.md` + 合并版 `design-todo.csv` |

**关键设计：critique 与 audit 互盲。** 两者在独立上下文里跑，互相看不到对方结果，直到 polish 阶段才合并。这样避免"AI 的视觉判断"被"机器规则"锚定。polish 会区分三类信号：
- 两边都命中 → 强证据，直接修
- 只 audit 命中 → 静态规则可定性 → 进 `audit-report.csv`
- 只 critique 命中 → AI 拿不准 → **强制进 `design-todo.csv`**

---

## 六、audit 轨在查什么（9 内置 + 1 外部）

| ruleId | 查什么 | 等级 | 落表 |
|---|---|---|---|
| `radius` | 管控端非 4px 圆角（`rounded-xl/2xl` / `border-radius:8px` 等） | P0 | audit |
| `color` | 硬编码颜色（`#1447E6` / `rgba(...)`）应走 token | P1 | audit |
| `icon-slot` | 9 类不可回退槽位仍在用 `lucide-react` 图标 | P0 | audit |
| `shadow` | 阴影未走 `--cp-shadow-*` | P1/P2 | audit |
| `component-drift` | `<X variant="y">` 的 y 不在组件 spec 白名单 | P2 | audit |
| `page-recipe-match` | 页面骨架缺 recipe 登记的必需组件 | P2 | audit |
| `text-color` | 文字色用了 Tailwind 中性灰（`text-gray-500`），应走 Typography | P2 | audit |
| `surface-nesting` | Surface 容器套娃（§7.4 设计铁律） | P2 | audit |
| `spacing-grouping` | 相邻控件疑似未成组，应改 flex+gap（**判断题**） | P2 | **design-todo** |
| `external/*` | 复用设计 skill 老脚本（8 个 check），含 `needs-design-confirmation` 收集 | P0~P2 | audit + design-todo |

---

## 七、产物怎么读

跑完后所有产物落在 `_walkthrough/<时间戳>/`：

- **`audit-report.csv`（AI 直接修）** — 确定违规清单，9 列：
  `ruleId, severity, file, line, col, snippet, message, evidence, suggestion`
  每行带精确定位、指回规范的 `evidence` 锚点、可直接执行的 `suggestion`。

- **`design-todo.csv`（给用户拍板）** — 待裁决清单，9 列中文：
  `冲突类型, 所属页面, 槽位/位置, 问题描述, AI 当前处理, 建议, 展示台对照, 真实页面参照, 用户裁决`
  `用户裁决` 默认 `待裁决`，用户拍板后建议在设计 skill 的 `conflict-log.md` 留痕。

- **`polish-plan.md` / `meta.json` / `trend.json`** — 收口清单、健康分、历次趋势。

> 铁律：**AI 拿不准的问题只进 `design-todo.csv`，不进 `audit-report.csv`，也不参与退出码 / 健康分扣分。**

---

## 八、接 CI 做卡口

```bash
node scripts/walkthrough.mjs diff   # P0 命中 → exit 1，阻断合码
```

- `diff`：退出码 `0 全绿 / 1 命中阻断阈值 / 2 异常`，默认 **P0 阻断**（CI 主用）。
- `audit <target>`：默认始终 0（不卡历史存量），设 `WALKTHROUGH_BLOCK_ON_AUDIT=1` 才启用退出码。
- 阻断级别可调：`WALKTHROUGH_BLOCK_LEVEL=P0|P1|P2|NONE`。

完整接入方案见 `ci/README.md`。

---

## 九、边界（不做什么）

- **不静默 / 不批量改业务代码**：扫描只读，任何改动都先出清单、逐条经用户确认。
- **不替代用户判断**：AI 拿不准的一律进 `design-todo.csv`，绝不私自定性。
- **不依赖真实浏览器**：以静态代码扫描为主，截图仅作可选可视化辅助。
- **不脱离规范立法**：没有设计规范章节锚点的规则，禁止落地。

---

## 十、一句话总纲

> **clawpro-walkthrough 是一个走查后先出清单、逐条经用户确认才改代码的设计走查 skill：机器负责毫秒级找问题、取证、出清单，人只负责在清单上拍板。**
