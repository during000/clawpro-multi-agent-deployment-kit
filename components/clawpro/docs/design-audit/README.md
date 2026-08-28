# ClawPro design-audit 协作入口

> 用途：这是 `docs/design-audit/` 的总入口路由。给同事、同事 AI、后续维护者看时，先从这里开始，不要在目录里零散找文件。

## 0. 当前最重要的结论

周一交付包主体不是本目录，而是：

```text
.codebuddy/skills/clawpro-portable-design-skill/
```

`docs/design-audit/` 是协作材料、确认记录、审计过程和续接 prompt 的归档。  
如果只想知道“周一交什么、从哪看、怎么协作”，优先读本文件和下方 P0 文件。

## 1. 如果只读 5 个文件

| 顺序 | 文件 | 用途 |
|---:|---|---|
| 1 | `.codebuddy/skills/clawpro-portable-design-skill/README.md` | 交付包人类入口，说明 portable design pack 怎么用 |
| 2 | `.codebuddy/skills/clawpro-portable-design-skill/DEVELOPER-USAGE.md` | 产品前端拿到交付包后如何执行换皮 |
| 3 | `.codebuddy/skills/clawpro-portable-design-skill/DESIGN-AUDIT-PLAYBOOK.md` | 结合 impeccable 反向审查设计仓库页面 |
| 4 | `docs/design-audit/design-confirmation-decisions-2026-06-06.md` | 当前已经确认的设计决策结果 |
| 5 | `docs/design-audit/monday-collaboration-plan-2026-06-06.md` | 周一到公司如何介绍、如何分工、后续怎么推进 |

## 2. 按角色阅读路线

### 2.1 设计同事

先读：

```text
docs/design-audit/design-confirmation-decisions-2026-06-06.md
.codebuddy/skills/clawpro-portable-design-skill/references/foundation.md
.codebuddy/skills/clawpro-portable-design-skill/references/admin.md
.codebuddy/skills/clawpro-portable-design-skill/references/tenant.md
.codebuddy/skills/clawpro-portable-design-skill/references/landing.md
.codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md
```

重点确认：

- 全局 token、文字、描边、阴影是否准确。
- Admin / Tenant / Landing 分端规则是否合理。
- `C-016`、`C-019` 这类暂选项是否需要继续确认。
- 客户端、落地页专项内容是否要由对应 owner 回写。

### 2.2 前端 / 产品开发同事

先读：

```text
.codebuddy/skills/clawpro-portable-design-skill/README.md
.codebuddy/skills/clawpro-portable-design-skill/SKILL.md
.codebuddy/skills/clawpro-portable-design-skill/references/components.md
.codebuddy/skills/clawpro-portable-design-skill/references/migration-map.md
.codebuddy/skills/clawpro-portable-design-skill/component-specs/
.codebuddy/skills/clawpro-portable-design-skill/portable/
.codebuddy/skills/clawpro-portable-design-skill/tokens/design-tokens.json
```

重点确认：

- 宿主仓现有组件能否映射到 spec。
- fallback 示例是否足够可执行。
- 哪些内容适合页面级换皮，哪些适合进入组件库。
- 试点页面应优先选哪个 Admin 页面 / Tenant 页面。

### 2.3 同事 AI / 新对话 AI

直接读：

```text
docs/design-audit/new-chat-resume-template-2026-06-06.md
```

里面已经把常用 prompt 和项目相对路径整理好了。

## 3. 当前协作核心文件

| 文件 | 作用 | 推荐阅读对象 |
|---|---|---|
| `docs/design-audit/README.md` | 本目录总入口路由 | 所有人 |
| `docs/design-audit/new-chat-resume-template-2026-06-06.md` | 新对话 / 同事 AI 续接 prompt | 同事、AI |
| `docs/design-audit/design-confirmation-decisions-2026-06-06.md` | 已确认设计决策结果 | 设计、前端、AI |
| `.codebuddy/skills/clawpro-portable-design-skill/DEVELOPER-USAGE.md` | 产品前端换皮使用说明 | 前端、AI |
| `.codebuddy/skills/clawpro-portable-design-skill/DESIGN-AUDIT-PLAYBOOK.md` | 结合 impeccable 的统一设计审查与换皮验收流程 | 设计、AI、前端 |
| `docs/design-audit/p0-conflict-confirmation-page-2026-06-06.html` | 设计确认选择页源码 | 设计、协作 owner |
| `docs/design-audit/monday-collaboration-plan-2026-06-06.md` | 周一介绍话术、会议流程、后续计划 | 所有人 |
| `docs/design-audit/monday-delivery-collab-summary-2026-06-06.md` | 为什么采用 portable design pack 的协作结论 | 设计、管理、前端 |
| `docs/design-audit/monday-delivery-package-structure-2026-06-06.md` | 交付包目录结构建议 | 设计、AI、前端 |
| `docs/design-audit/monday-delivery-work-split-2026-06-06.md` | 早期分工建议 | 设计协作者 |
| `docs/design-audit/tenant-landing-design-confirmations-2026-06-06.md` | Tenant / Landing 早期待确认项 | 客户端 / 落地页设计 |

## 4. 参考 / 审计归档文件

这些文件是过程证据和参考材料，不是周一协作的第一入口：

| 文件 / 目录 | 作用 |
|---|---|
| `docs/design-audit/design-spec-conflict-audit-2026-06-05.md` | 早期规范冲突审计 |
| `docs/design-audit/design-component-compliance-audit-2026-06-05.md` | 组件合规审计 |
| `docs/design-audit/design-governance-and-delivery-plan-2026-06-05.md` | 早期治理与交付路线分析 |
| `docs/design-audit/design-audit-round3-materials-and-template-2026-06-05.md` | 审计材料模板 |
| `docs/design-audit/changelog-2026-05-28.md` | 历史变更记录 |
| `docs/design-audit/changed-files-2026-05-28.txt` | 历史变更文件列表 |
| `docs/design-audit/component-index/` | 组件 ↔ 页面交叉索引工具与报告 |
| `docs/design-audit/figma-730_665/` | Figma 静态快照归档 |
| `docs/design-audit/*.html` | 辅助预览页面或确认页 |

## 5. 周一建议怎么开始

1. 先打开 `.codebuddy/skills/clawpro-portable-design-skill/README.md`，说明交付包是什么。
2. 再打开 `docs/design-audit/design-confirmation-decisions-2026-06-06.md`，说明哪些设计点已确认。
5. 再打开 `docs/design-audit/monday-collaboration-plan-2026-06-06.md`，按里面的会议流程和分工推进。
6. 最后按角色进入具体文件：设计看 `references/`，前端看 `DEVELOPER-USAGE.md` + `component-specs/` + `portable/`，页面评审 / 换皮验收看 `DESIGN-AUDIT-PLAYBOOK.md`，AI 看 `new-chat-resume-template`。

## 6. 重要提醒

- 不要把 `docs/design-audit/` 当成正式交付包主体；正式交付包主体是 `.codebuddy/skills/clawpro-portable-design-skill/`。
- `design-confirmation-decisions-2026-06-06.md` 是当前设计确认依据；标注“暂选 / 后续进一步确认”的项不能写成不可变更最终裁决。
- `portable/` fallback 示例优先使用 token / CSS variable，不直接散写业务色 hex。
- 历史页面 hardcoded style 是存量债务，不作为新增例外。
- 后续如果要组件库化，应先用 1-2 个试点页面验证 portable pack，再沉淀组件 API。
