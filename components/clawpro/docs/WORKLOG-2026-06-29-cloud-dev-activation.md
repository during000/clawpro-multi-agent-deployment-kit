# WORKLOG · DarkVeil 动态背景 + 开通页模板沉淀进 design-skill（2026-06-29）

> 本文是「把**已实现**的管控端开通云开发能力页 DarkVeil 动态 hero + 开通页模板**沉淀进 `clawpro-portable-design-skill`**」一轮交付的工作台账，记录 14 项交付的任务名 / 产出文件 / 状态，供日后人或 AI 溯源背景、与同事对齐 skill 口径、避免误改误删。
> ⚠️ **范围澄清**：页面侧动态背景本体（`client/src/pages/admin/CloudDevActivation.tsx` + `client/src/components/ui/DarkVeil.tsx`）**此前已实现并上线，本轮不改页面实现**；本轮 14 项交付全部是 **skill 侧** 的新增 / 修改（spec、配方、兜底样例、登记、文档），目标是把这套既有实现沉淀为可复用的设计治理资产。
> Owner：miekoyychen　|　日期：2026-06-29
> 配套真相源：`.codebuddy/skills/clawpro-portable-design-skill/component-specs/dark-veil.md`（§0 Auto-Trigger、§4 hero 配方、§9 L0/L1/L2 兜底分级），本台账措辞均以该 spec 为准，**不重新定义 L0/L1/L2**。

---

## 1. 这次要解决什么（一句话）

管控端「开通云开发能力」页（`/admin/cloud-dev` 未开通态）的 **hero 区动态背景 `DarkVeil` 早已在 `CloudDevActivation.tsx` 中实现并上线**——本轮**不动这套页面实现**。本轮要解决的是：把这套「DarkVeil 动态 hero + 开通页模板」**沉淀进设计治理 skill**，使其成为**可门控、可跨仓兜底、可复现**的全局能力，并形成可与同事对齐的统一口径——而不是让后续每个开通页各写一套 canvas / 渐变、各自为政。

定位说明：DarkVeil 是「锦上添花」而非「页面默认」。它只用于命中 Auto-Trigger 的开通页 / 能力 hero，普通列表 / 表单 / 详情 / Dashboard / 设置页与整页背景**禁止**滥用（详见 `dark-veil.md` §0 / §2）。

---

## 2. 关键事实（改动前必读）

- **DarkVeil = 纯装饰**：永远 `pointer-events-none`、永远在内容层 `z-10` 之下、永远配「基底 + 蒙版 + 收束」三件套保证文字可读；单页最多 1 个 WebGL 实例。
- 组件本体：`client/src/components/ui/DarkVeil.tsx`（默认导出 `DarkVeil`，依赖 `ogl`）。
- hero 实际用法：`client/src/pages/admin/CloudDevActivation.tsx`（参数配方 speed 1.1 / warp 1.1 / noise 0.05 / tint #B2C3FF，基底 #E0EBFE，顶部 mask 22% 淡出）。
- **Auto-Trigger 门控**：仅当 A. 场景 = 管控端开通页 / 能力介绍 hero / 首次引导空态顶部 hero 区，且 B. 设计意图明确要动态流动光效且设计师已拍板，二者同时满足才用；不命中即不用，模糊时记 `conflict-log.md` 标 `needs-design-confirmation`。
- **跨仓兜底分档（口径以 `dark-veil.md` §9 为准，禁止重定义）**：
  - **L0 完整移植**（首选）：`ogl` + WebGL，1:1 动态；直接复制 `DarkVeil.tsx` 整文件，不改 shader。
  - **L1 静态 CSS 兜底**（`migration-map.md` **默认档**）：纯 CSS 径向渐变光晕 + 可选极慢动画，神似蓝紫飘带，零依赖；宿主仓**最少应做到 L1**。
  - **L2 纯色 / 截图兜底**（最低）：纯色基底或一张静态截图，用于禁脚本 / 静态导出 / 低端 / `prefers-reduced-motion`。

---

## 3. 14 项交付清单（任务名 / 产出文件 / owner / 状态）

> 状态口径：`✅完成` = 已落地；`进行中` = owner 仍在产出。
> 本清单以 team-lead（main）下发的**权威 14 项交付计划**为准（2026-06-29）；agent-c 负责第 7 / 8 / 9 / 14 项。

| # | 任务名 | 产出文件 | owner | 状态 |
|---|---|---|---|---|
| 1 | 页面 spec（开通页 hero 区 + 核心能力卡骨架） | `.codebuddy/skills/clawpro-portable-design-skill/references/admin-cloud-dev-activation.md` | main | ✅完成 |
| 2 | 组件真相源 spec（L0/L1/L2 分级 + Auto-Trigger + hero 配方，唯一口径） | `.codebuddy/skills/clawpro-portable-design-skill/component-specs/dark-veil.md` | main | ✅完成 |
| 3 | 全局组件登记（§33 DarkVeil） | `SKILL-GLOBAL-COMPONENTS.md` | agent-b | 进行中 |
| 4 | 展示台注册（DarkVeil 条目，group `foundation` / platform「Admin 管控端」+ DarkVeilPreview） | `client/src/pages/DesignSystemComponents.tsx` | agent-b | 进行中 |
| 5 | Portable 兜底样例（L1 静态 CSS 渐变） | `.codebuddy/skills/clawpro-portable-design-skill/portable/css/dark-veil.css` + `portable/html-css/dark-veil.html` | agent-b | 进行中 |
| 6 | Portable 登记（portable/README + qa/QA-CHECKLIST 登记） | `.codebuddy/skills/clawpro-portable-design-skill/portable/README` + `qa/QA-CHECKLIST` | agent-b | 进行中 |
| 7 | 冲突日志 C-018（范围 + Auto-Trigger 门控）/ C-019（L0/L1/L2 + 归 L1 口径） | `.codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md` | **agent-c** | ✅完成 |
| 8 | 工作台账（本文件） | `docs/WORKLOG-2026-06-29-cloud-dev-activation.md` | **agent-c** | ✅完成 |
| 9 | 样本登记 + 选样拆两款（L0 完整 hero 款 / L1·L2 静态兜底款） | `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/README.md` | **agent-c** | ✅完成 |
| 10 | SKILL.md ×4 编辑（§4 第 8 项 WebGL/ogl 检测→L0/L1/L2、§9 高风险 Spec 表加 DarkVeil、§10 References Map 加开通页/动态背景 hero 行、§2.4 圆角后加 hero 装饰玻璃元素例外引用块） | `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md` | main | ✅完成 |
| 11 | 配方（新增「6. 能力开通页 Hero（动态背景·专项配方）」并顺延原 6/7/8/9） | `.codebuddy/skills/clawpro-portable-design-skill/references/page-recipes.md` | main | ✅完成 |
| 12 | 迁移映射（加 DarkVeil 行，默认兜底档 L1） | `.codebuddy/skills/clawpro-portable-design-skill/references/migration-map.md` | main | ✅完成 |
| 13 | 页面截图（dev 真截，逻辑 1440×900 / 2x retina 2880×1800） | `.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/admin-cloud-dev-activation.png` | main | ✅完成 |
| 14 | 发布文档主仓改 `openclaw-enterprise`（选项 1，只改源码主仓侧，保留展示仓 / oa-pages / CNAME / 域名不动） | `client/public/research/clawpro-design-system-showcase-guide.md` | **agent-c** | ✅完成 |

> 编号对齐说明：「样本 spec(.md) + 截图(.png)」并入第 9 项 README 登记 + 第 13 项截图；`client/src/components/ui/DarkVeil.tsx` 组件本体为既有源码、只读引用，不属本次新增交付，故不单列。

---

## 4. agent-c 本轮已落地明细（任务 7 / 8 / 9 / 14）

### 任务 7 · conflict-log.md 追加 C-018 / C-019
- 文件：`.codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md`
- **C-018**：DarkVeil 引入的**范围边界与 Auto-Trigger 门控**——锁定纯装饰定位（`pointer-events-none` + 三件套 + 内容 `z-10`），仅命中「开通页 / 能力 hero + 设计已拍板」才用，普通功能页 / 整页 / Tenant / Landing 禁止滥用，单页最多 1 个实例。状态 `resolved`。
- **C-019**：DarkVeil **跨仓兜底分档 L0/L1/L2** 与 `migration-map.md` 归 **L1 默认档**的口径——三档是同一视觉的不同保真度，宿主仓最少做到 L1，静态兜底必做不留白板；分级以 `dark-veil.md` §9 为准、禁止重定义。状态 `resolved`。
- 均按既有条目格式（状态 / 背景 / 现象 / 风险 / 裁决 / 执行）书写；状态用 `resolved`（已约定交付，未私自裁决美学）。

### 任务 8 · 本台账
- 文件：`docs/WORKLOG-2026-06-29-cloud-dev-activation.md`（本文件）。
- 参照 `docs/ClawPro资源库-阶段9决策溯源(ADR).md`、`docs/ClawPro资源库-新资源接入SOP.md` 的口吻与结构。

### 任务 9 · page-references 样本登记 + 选样拆两款
- 文件：`.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/README.md`
- §2 样本清单新增一行「能力开通页（动态 hero）· 开通云开发能力 · `/admin/cloud-dev`」，关键组合写 DarkVeil 三件套 hero + 核心能力卡，详细 spec 指向 `admin-cloud-dev-activation.md`；表头由「7 类」更新为「8 类」。
- §6 选样指引按既有「目标页面 → 优先看样本」维度**拆两款**：① hero 要动态流动光效（命中 Auto-Trigger / 设计已拍板）→ L0 完整动态 hero 款；② 同场景但宿主仓无 ogl/WebGL 或需静态导出 → L1/L2 静态兜底款（指向 `dark-veil.md` §9）。

### 任务 14 · 发布文档主仓改为 openclaw-enterprise
- 文件：`client/public/research/clawpro-design-system-showcase-guide.md`
- 把所有指向「源码主仓 / 日常开发仓」的仓名与路径由 `clawpro`（`/Users/miekoyychen/CodeBuddy/clawpro`）改为 `openclaw-enterprise`（`/Users/miekoyychen/openclaw-enterprise`）：§2 源码位置、§4 标题与内文、§5.1、§8.1、§8.2（`cd` 命令）、§8.3 发布话术、§8.4 第 1/9 步、§9 两仓关系、§10 第 2 条、§11 速查表「主仓库」行。
- **保留不动**（发布产物侧）：展示仓 `clawpro-design-system-showcase`、本地路径 `/Users/miekoyychen/CodeBuddy/clawpro-design-system-showcase`、远程 git 地址、`oa-pages` 分支、`CNAME` / 域名 `clawpro-design-system.pages.woa.com`、OA Pages 管理 / 日志链接、线上访问地址；产品名「ClawPro 全局组件展示台 / ClawPro 设计系统」属命名非仓名，亦保留。
- 改完全文 grep 复核，未把展示仓误改。

---

## 5. 进行中 / 收尾门禁

> 第 3 节 14 项已对齐 main 权威清单。截至本台账，agent-c 负责的 7/8/9/14 与 main 负责的 1/2/10/11/12/13 均 ✅完成；剩余进行中项与收尾门禁如下。

1. **agent-b 负责项（第 3~6）**：§33 SKILL-GLOBAL-COMPONENTS.md、展示台 DarkVeil 条目（group `foundation` / platform「Admin 管控端」+ DarkVeilPreview）、portable L1 静态兜底（`portable/css/dark-veil.css` + `portable/html-css/dark-veil.html`，并补 `portable/react/dark-veil/dark-veil-static.tsx` 即 §9.2 的 `DarkVeilStaticFallback` React 等价包装）、portable/README + qa/QA-CHECKLIST 登记。React 包装已补完，三个 portable 文件均已登记进 `MANIFEST.portableExamples`。
2. **MANIFEST / verify 收口（整轮门禁）**：`component-specs/dark-veil.md` 进 `MANIFEST.json` `componentSpecs`、3 个 portable 文件进 `portableExamples` 后，`scripts/verify-portable-skill.mjs` 须跑出绿色（无 missing / unlisted）才算整轮闭环。agent-b 已登记并跑 verify 通过为绿色。
3. **发布文档构建产物目录路径（任务 14 旁支，main 已拍板）**：`§5.3 / §8.4 / §11` 的构建产物目录 `/Users/miekoyychen/CodeBuddy/clawpro-deploy/clawpro-design-system-only` 属**发布产物侧**（非源码主仓、非展示仓，与展示仓 / oa-pages / CNAME / 域名同侧）。main 已拍板（2026-06-29）：**保持现状、不迁移**——选项 1 只迁源码主仓 `clawpro → openclaw-enterprise`，该产物目录维持旧布局 `/Users/miekoyychen/CodeBuddy/` 不动；agent-c「保留未动」的判断已确认正确。此点已收口，无遗留待办。
