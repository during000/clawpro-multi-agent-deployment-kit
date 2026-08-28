# 演示总纲 · 设计规范对齐机制端到端

> 本目录是「设计规范对齐机制」团队演示的产物区。每个场景一个子文件夹，内含结构化文案（SCRIPT.md）+ 截图 + 录屏。
> 干跑已验证全部命令与退出码（node v22.14.0，基线 commit `68e7df32`，工作树 clean）。

## 一句话价值
代码是唯一真相源。设计规范/组件稳定后，一句口令触发，AI 把「代码里的真实改动」沿 `代码 → Tier1 沉淀 → Tier2 设计skill → Tier3 走查skill` 对齐下去，产出「对不上清单」逐条确认后 reconcile，全程以**退出码 / git diff / grep 计数**为证据。

## 录制顺序（固定 A→B→C→D→彩蛋）
| 段 | 文件夹 | 主题 | 核心证据 |
|---|---|---|---|
| S0 | `00-setup` | 开场 + 基线自证 | `git status` clean |
| A | `A-modify-component` | 改现有组件（StatusTag +banner + token） | 双通道差集 exit 1→0 |
| B | `B-new-component` | 新增组件（rating-stars onboarding） | CHANGED_NO_SPEC 1→0 + MANIFEST FAILED→PASSED |
| C | `C-spec-rule` | 改语义规范（Tier1→下游联动） | 一条规范落 3 文件 diff |
| D | `D-page-template` | 页面模板引用校验 | 存在性校验 MISS→exit1→修复→exit0 |
| E | `E-full-audit` | 彩蛋 · 全量体检 | 豁免 62 / 候选 32 |

## 人工触发（真实使用入口，每场景必演）
真实链路不是「直接跑脚本」，而是 **人改完代码 → 贴 kickoff → AI 读 SOP → 自动跑流水线**（SOP §0 / REQUIREMENT §11.1）。

> ⚠️ 重要：`对齐 skill 规范` 只是**人记的入口口令**，不是系统注册的自动命令（现阶段走人工入口，将来才薄 skill 化）。因此**新会话 / 教同事时，要贴下面完整的 kickoff prompt**——口令 + 明确让 AI 读 SOP 并按对应段执行——才可复制照用。

**增量对齐 · kickoff（A/B/C/D 用，改完代码后贴）：**
```
对齐 skill 规范。请阅读 docs/design-skill-sync/SOP.md 的 A 段（增量对齐流程），
针对我刚才对真相源（client/src/index.css + components/ui/*.tsx）的改动，
按 Stage0→Stage1→Stage2 执行：先跑 P1 差集 diff-code-vs-docs.mjs --since HEAD 出「对不上清单」，
逐条经我确认后 reconcile（以代码为准，不覆盖 Tier2 独有内容），
再重抽 fixtures，复跑脚本至退出码 0。
```

**全量审计 · kickoff（E 用）：**
```
全量体检 skill 对齐度。请阅读 docs/design-skill-sync/SOP.md 的 B 段（全量审计流程），
跑 diff-code-vs-docs.mjs --full 出一次性 backlog 快照，按类型汇总，findings 只作 backlog 不强制修。
```

- 触发截图点 `X0-trigger.png`：框住「人贴的 kickoff + AI 回复开头（打开/引用 SOP、开始执行）」。
- 被触发后 AI 会先读 `SOP.md` 再执行，体现「SOP 是大脑、kickoff 是人工入口」。

## 退出码语义
`0` 无差集 · `1` 有差集候选 · `2` 用法/环境错。

## 通用铁律（可作 PPT 开场页）
- **P1 值/枚举一律以代码为准**；文档只对代码表达不了的语义规则负责。
- **每层先出「对不上清单」→ 逐条确认 → reconcile（不覆盖 Tier2 独有内容）**。
- **改动范围靠 `git diff` 机械圈定**，不靠口述。
- **历史存量只豁免、不回填**；机制只防增量。

## 环境 / 基线（录制前自证用）
```
node -v                 # v22.14.0
git status              # nothing to commit, working tree clean
git log --oneline -1    # 68e7df32 feat(tokens): 添加引导气泡与模块浮层设计令牌
```

## 演示收尾 · 一键还原（录制结束后执行，确保 git 干净）
```
# 场景 A / C 改的 tracked 文件
git checkout -- client/src/components/ui/status-tag.tsx client/src/index.css \
  SKILL-GLOBAL-COMPONENTS.md \
  .codebuddy/skills/clawpro-portable-design-skill/component-specs/status-tag.md \
  .codebuddy/skills/clawpro-portable-design-skill/component-specs/card-surface.md \
  .codebuddy/skills/clawpro-portable-design-skill/tokens/design-tokens.json \
  .codebuddy/skills/clawpro-portable-design-skill/MANIFEST.json \
  .codebuddy/skills/clawpro-portable-design-skill/references/component-mapping.md \
  .codebuddy/skills/clawpro-walkthrough/SKILL.md \
  .codebuddy/skills/clawpro-walkthrough/fixtures \
  .codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-members.md
# 场景 B 新增文件
git rm --cached client/src/components/ui/rating-stars.tsx 2>/dev/null
rm -f client/src/components/ui/rating-stars.tsx \
  .codebuddy/skills/clawpro-portable-design-skill/component-specs/rating-stars.md
# 走查产物
rm -rf _walkthrough
git status   # 期望：working tree clean（demo/ 目录内的截图录屏属产物，可保留）
```
