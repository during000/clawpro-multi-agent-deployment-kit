# ClawPro 新对话续接模板

> 用途：后续新开对话 / 切对话时，直接复制对应 prompt 给 AI，让它在不依赖旧线程记忆的情况下继续交付包、组件维护或页面审查工作。

## 0. 协作相关文档清单

如果是给同事 / 同事 AI 续接，通常不只看本文件，还建议按任务读取这些项目相对路径：

| 文件 | 作用 |
|---|---|
| `docs/design-audit/new-chat-resume-template-2026-06-06.md` | 新对话 prompt 模板入口 |
| `docs/design-audit/design-confirmation-decisions-2026-06-06.md` | 当前已选择的设计确认结果 |
| `docs/design-audit/p0-conflict-confirmation-page-2026-06-06.html` | 设计确认选择页源码 |
| `docs/design-audit/monday-delivery-collab-summary-2026-06-06.md` | 协作口径总结 |
| `docs/design-audit/monday-delivery-package-structure-2026-06-06.md` | 交付包结构说明 |
| `docs/design-audit/monday-delivery-work-split-2026-06-06.md` | 分工建议 |
| `.codebuddy/skills/clawpro-portable-design-skill/STATUS.md` | 当前交付包续接状态 |
| `.codebuddy/skills/clawpro-portable-design-skill/README.md` | portable design pack 人类入口 |
| `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` | portable design pack AI 入口 |
| `.codebuddy/skills/clawpro-portable-design-skill/DEVELOPER-USAGE.md` | 产品前端换皮使用说明 |
| `.codebuddy/skills/clawpro-portable-design-skill/DESIGN-AUDIT-PLAYBOOK.md` | 结合 impeccable 的统一设计审查与换皮验收流程 |
| `PRODUCT.md` | impeccable product register 项目上下文 |

下面所有 prompt 均使用项目相对路径，默认从仓库根目录读取。

## 1. 推荐完整版：继续维护交付包

```text
项目根目录：当前仓库根目录

继续维护 ClawPro 周一交付包 / portable design pack。

请先读取这些文件：

1. .codebuddy/skills/clawpro-portable-design-skill/README.md
2. .codebuddy/skills/clawpro-portable-design-skill/SKILL.md
3. .codebuddy/skills/clawpro-portable-design-skill/STATUS.md
4. .codebuddy/skills/clawpro-portable-design-skill/DEVELOPER-USAGE.md
5. docs/design-audit/design-confirmation-decisions-2026-06-06.md
6. .codebuddy/skills/clawpro-portable-design-skill/references/foundation.md
7. .codebuddy/skills/clawpro-portable-design-skill/references/components.md
8. .codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md

目标：
- 继续维护 / 收口 clawpro-portable-design-skill
- 不要重做已有内容
- 以 design-confirmation-decisions-2026-06-06.md 为当前设计确认依据
- 已确认项可以回写规范
- 标注“暂选 / 后续进一步确认”的项，不要写成不可变更最终裁决
- portable fallback 示例优先使用 token / CSS variable，不直接散写业务色 hex
- 历史页面 hardcoded style 是存量债务，不作为新增例外
- 如涉及 Admin / Tenant / Landing，请按端读取对应 reference：
  - Admin: references/admin.md
  - Tenant: references/tenant.md
  - Landing: references/landing.md
- 如涉及具体组件，请读取 component-specs/ 下对应 spec

先总结：
1. 当前交付包状态
2. 已确认的关键设计决策
3. 仍需后续确认的事项
4. 接下来建议做什么

然后再按我的具体任务继续推进。
```

## 2. 短版：使用这套规范做页面 / 审查页面

```text
项目根目录：当前仓库根目录

请使用 ClawPro portable design pack 作为唯一设计规范来源：

.codebuddy/skills/clawpro-portable-design-skill/

先读：
1. .codebuddy/skills/clawpro-portable-design-skill/README.md
2. .codebuddy/skills/clawpro-portable-design-skill/SKILL.md
3. .codebuddy/skills/clawpro-portable-design-skill/STATUS.md
4. .codebuddy/skills/clawpro-portable-design-skill/DEVELOPER-USAGE.md
5. docs/design-audit/design-confirmation-decisions-2026-06-06.md
6. .codebuddy/skills/clawpro-portable-design-skill/references/foundation.md
7. .codebuddy/skills/clawpro-portable-design-skill/references/components.md

然后根据任务端别继续读：
- Admin 读 .codebuddy/skills/clawpro-portable-design-skill/references/admin.md
- Tenant 读 .codebuddy/skills/clawpro-portable-design-skill/references/tenant.md
- Landing 读 .codebuddy/skills/clawpro-portable-design-skill/references/landing.md
- 页面模板读 .codebuddy/skills/clawpro-portable-design-skill/references/page-recipes.md
- 高风险组件读 .codebuddy/skills/clawpro-portable-design-skill/component-specs/ 对应文件

执行时遵守：
- 不重做已有内容
- 以确认表结果为准
- fallback 必须使用 token / CSS variable
- 不新增旧色值、旧圆角、旧阴影
- 不让 AI 自行裁决未确认设计问题
```

## 3. 交付准备版：检查是否可以打包交付

```text
项目根目录：当前仓库根目录

继续完成 ClawPro portable design pack 的交付准备。

先读：
1. .codebuddy/skills/clawpro-portable-design-skill/README.md
2. .codebuddy/skills/clawpro-portable-design-skill/SKILL.md
3. .codebuddy/skills/clawpro-portable-design-skill/STATUS.md
4. .codebuddy/skills/clawpro-portable-design-skill/DEVELOPER-USAGE.md
5. .codebuddy/skills/clawpro-portable-design-skill/DELIVERY-CHECKLIST.md
6. .codebuddy/skills/clawpro-portable-design-skill/INDEX.md
7. docs/design-audit/design-confirmation-decisions-2026-06-06.md

然后：
1. 检查交付包是否还缺必需文件
2. 检查 STATUS.md 里剩余 P0
3. 跑 verify-portable-skill.mjs
4. 如需，跑 check-design-usage.mjs，但注意业务代码 warning 是存量债务，不阻塞 portable pack
5. 总结是否可以交付，以及剩余风险
```

## 4. 如果要让 AI 自动分工

```text
我允许你按明确写域拆分 2 到 3 个 agent 并行推进，但你要先自己给出主计划，再分工。

要求：
- 每个 agent 的写域不能重叠。
- 不要把需要我做设计裁决的事情丢给 agent 自行判断。
- 所有不确定的 token / 颜色 / 视觉规则，先回到主线程问我。
```

## 5. 如果只想继续主流程，不开 agent

```text
这次不要开 agent，主流程就在当前对话里继续推进。
先根据 STATUS.md 识别当前最该做的 P0，再直接动手。
```

## 6. 如果要把同事也拉进来

把下面这段发给同事或同事的 AI：

```text
我们在推进 ClawPro 周一交付包，请先读：
1. docs/design-audit/monday-delivery-collab-summary-2026-06-06.md
2. docs/design-audit/monday-delivery-package-structure-2026-06-06.md
3. docs/design-audit/monday-delivery-work-split-2026-06-06.md
4. docs/design-audit/monday-collaboration-plan-2026-06-06.md
5. docs/design-audit/design-confirmation-decisions-2026-06-06.md
6. .codebuddy/skills/clawpro-portable-design-skill/STATUS.md

目标不是继续补厚总规范，而是把 .codebuddy/skills/clawpro-portable-design-skill 做成周一可交付的 portable design pack。
```
