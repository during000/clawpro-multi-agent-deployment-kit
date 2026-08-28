# 场景 C · 修改/新增规范（只演「skill 如何同步修改」）

> 日常触发语：「我定了一条新语义规范——卡片次级说明文字统一用 `text-[var(--text-brand)]`（#1447E6 品牌蓝）。」
> 重点：这类规则**代码枚举表达不了**（纯语义取舍），看机制怎么把它沿链路落到各层，用 `git diff` 展示多文件联动。

## 0. 人工触发（真实入口）📸 `C0-trigger.png`
> 顺序：先做 §1（在 Tier1 定这条新规则），再说口令 **`对齐 skill 规范`**；AI 读 `SOP.md` A 段后，把规则沿 Tier2（§2）→ Tier3（§3）传播。

## 1. 在 Tier1 加语义规则（源头）
`SKILL-GLOBAL-COMPONENTS.md` §0.6 增一条：
```
- **卡片次级说明文字统一用 `text-[var(--text-brand)]`（#1447E6 品牌蓝）**；不再用 `text-gray-500` / `muted` 等中性灰表达卡片内次级说明文字（本条为新增语义规范，下游 spec 与走查须对齐）。
```

## 2. 联动 Tier2 设计 skill
`component-specs/card-surface.md` §3.1「子元素字号/色 token」表：把「次级说明文字」一行的颜色从 `--cp-text-muted` 改为 `text-[var(--text-brand)]`，并注明「对齐 Tier1 §0.6 新增语义规范」。

## 3. 联动 Tier3 走查 skill 契约表
`clawpro-walkthrough/SKILL.md` §0.B 契约对照表新增一行：新规范由既有 `text-color` detector 承接，锚点写 `Tier1 §0.6 + card-surface.md §3.1`。
> 契约表硬约束：新规范必须能挂到某个 detector + 设计 skill 章节锚点，否则「没立法、禁止落地」。本条复用 v0.8 的 text-color，成立。

## 4. 展示多文件联动（证据）
```
git status --short
git --no-pager diff --stat
```
**实测证据**：一条语义规范落到 **3 个文件、跨三层**：
```
 .../component-specs/card-surface.md            | 3 ++-
 .codebuddy/skills/clawpro-walkthrough/SKILL.md | 1 +
 SKILL-GLOBAL-COMPONENTS.md                     | 1 +
 3 files changed, 4 insertions(+), 1 deletion(-)
```

## 5. 亮点话术（PPT）
- **语义规范也能沿链路传播**：代码抓不到的规则，由 AI 按 SOP 落到 Tier1→Tier2→Tier3。
- **每层各司其职**：Tier1 立规范、Tier2 落到具体 spec、Tier3 指定哪个 detector 承接（法律 vs 警察）。
- **契约不空挂**：Tier3 只有能挂 detector + 锚点的规范才允许落地。

## 6. 录制取帧（截图存本文件夹）
| 文件名 | 截什么 |
|---|---|
| `C1-tier1-rule.png` | Tier1 §0.6 新增规则那一行 |
| `C4-diff-stat.png` | `git diff --stat`：3 文件跨三层联动 |
| `C-recording.mp4` | 本场景全程录屏 |
