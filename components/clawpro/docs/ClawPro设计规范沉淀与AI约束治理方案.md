# ClawPro 设计规范沉淀与 AI 约束治理方案

> **适用对象**：产品、设计、前端开发、协作 AI  
> **适用范围**：ClawPro / OpenClaw Enterprise 项目 UI 设计系统、组件规范、Vibe Coding 协作流程  
> **文档目的**：明确下周规范交付目标、当前现状、距离理想状态的差距，以及后续如何修正规范、迁移页面、合并资料和补充第三轮运行态审计。  
> **版本**：v2.2 目标导向版  
> **日期**：2026-06-05

---

## 0. 结论先行

下周的核心目标不是再补一份过程报告，而是交付一套后续产品、设计、开发和 AI 都能持续使用的相对完整规范。

当前前两轮审计已经足够支撑规范收敛方向：ClawPro 现在不是缺少规范，而是规范文档、组件源码、业务页面落地三者尚未完全对齐。第三轮运行态审计仍然需要，但它的作用是确认现网 / 预发页面距离修正后的规范还有多远，而不是重新决定规范该怎么收敛。

本报告建议：

1. **下周交付目标**：优先修正并交付 `SKILL.md`、`SKILL-GLOBAL-COMPONENTS.md`、`SKILL-TENANT.md` 三份核心规范。
2. **当前现状**：`client/src/index.css` 与 `client/src/components/ui/*` 已经形成事实代码基线，但文档、注释和业务落地仍有偏差。
3. **主要差距**：三份 Skill 文档职责和优先级需要重整；高风险组件需要裁决；业务页面仍有硬编码、原生表格和组件绕开。
4. **实施方式**：少建新文件，先修现有规范和组件源码；高风险裁决直接写回现有规范与代码；第三轮后再决定是否需要新增整改清单。
5. **不建议现在做的事**：不默认新增 `DESIGN-COMPONENT-DECISIONS.md`、`DESIGN-MIGRATION-BACKLOG.md` 等长期文件；不提前承诺线上已经全部完成组件化和 token 化。

一句话：

> **目标是下周交付一套能真正指导后续工作的规范；现状和差距由审计报告支撑；怎么改、怎么迁移、怎么合并，由本方案给出实施建议。**

---

## 1. 下周交付目标

### 1.1 交付目标

下周应交付的是一套相对完整、可持续使用的 ClawPro 设计系统规范，而不是一组临时过程文件。

这套规范需要回答：

1. 后续团队做页面、组件、PRD 和 AI 协作时，应该以什么为准。
2. Shared / Admin / Tenant 的规范如何分层。
3. 文档和组件源码冲突时，如何仲裁。
4. 高风险组件应按哪套规则使用。
5. 旧页面如何逐步迁移。
6. AI 在 Vibe Coding 中应该读取哪些规范、禁止引用哪些历史资料。

### 1.2 建议交付范围

| 交付物 | 定位 | 下周需要做到什么程度 |
|---|---|---|
| `SKILL.md` | AI 总入口 / 规范路由 / 冲突仲裁 | 删除旧 token、旧品牌色、旧字体、旧圆角等具体数值，保留规范加载和仲裁规则 |
| `SKILL-GLOBAL-COMPONENTS.md` | 共享 / Admin 组件规范 | 修正与源码冲突的组件规则，沉淀高风险组件最终用法 |
| `SKILL-TENANT.md` | Tenant 差异规范 | 收敛为 Tenant 差异补充，不重复共享组件全文 |
| `docs/design-audit/**` | 审计证据与运行态审计材料 | 保留为治理依据，不作为当前 UI 规范源 |

### 1.3 暂不默认新增的文件

如果设计侧会直接修正规范文档，并推动组件源码同步修正，且第三轮运行态审计后还会统一整理最终规范，那么现在不需要默认新增以下长期文件：

```text
docs/DESIGN-COMPONENT-DECISIONS.md
docs/DESIGN-MIGRATION-BACKLOG.md
```

这些文件只在以下情况再考虑新增：

1. 第三轮审计后仍有大量问题无法立即修正。
2. 多人并行治理，需要单独追踪责任人、范围和状态。
3. 某些组件需要跨多个迭代长期迁移。
4. 现有 `SKILL-GLOBAL-COMPONENTS.md` / `SKILL-TENANT.md` 无法承载裁决结果。

更推荐的方式是：

> **高风险组件裁决直接写回现有规范和组件源码；暂时无法修完的遗留问题，第三轮后再决定是否补整改清单。**

---

## 2. 当前现状

### 2.1 总体判断

当前 ClawPro 不是从 0 开始，也不是已经完成规范收敛。更准确的状态是：

> **代码真源已经具备主体，但规范文档和业务落地仍需收口。**

### 2.2 分层现状

| 层级 | 当前现状 | 判断 |
|---|---|---|
| 审计 | 已完成规范冲突审计和全量组件合规审计 | 足够支撑规范收敛方向 |
| 规范文档 | `SKILL.md`、`SKILL-GLOBAL-COMPONENTS.md`、`SKILL-TENANT.md` 都已存在 | 职责和优先级需要重整 |
| 组件源码 | `client/src/index.css` 与 `client/src/components/ui/*` 已形成事实标准 | 可作为当前运行时真源 |
| Tenant / Admin | 差异方向已形成，Tenant 变体、TenantCard、Typography 等已有基础 | 分层规则仍需稳定 |
| 业务页面 | 仍存在硬编码、原生 `<table>`、className 覆盖组件、旧 token 残留 | 需要迁移和触达即同步 |
| 运行态 | 尚未完成第三轮现网 / 预发审计 | 不能确认线上实际差距 |

### 2.3 当前可直接采用的真源

| 范围 | 当前事实源 |
|---|---|
| 全局 token / 品牌色 / 字体 / 圆角 / 阴影 | `client/src/index.css` |
| 共享基础组件 | `client/src/components/ui/*` |
| Button | `client/src/components/ui/button.tsx` |
| Table | `client/src/components/ui/table.tsx` |
| Surface / Card | `client/src/components/ui/Surface.tsx` |
| Typography | `client/src/components/ui/Typography.tsx` |
| StatusTag | `client/src/components/ui/status-tag.tsx` |
| AdminSidebar | `client/src/components/ui/admin-sidebar.tsx` |

原则：

> 当文档与代码冲突时，先以代码作为“当前事实”，再决定是修文档、修组件，还是安排业务迁移。

---

## 3. 距离理想状态的差距

### 3.1 理想状态

理想状态不是“所有文档合成一份”，而是：

1. 规范入口清晰，团队知道该看哪份。
2. 文档与组件源码一致，AI 不会读到过时规则。
3. Tenant / Admin 差异明确，不靠业务页面散落 className 表达。
4. 业务页面优先复用共享组件，不再自造基础样式。
5. 新页面严格按新规范，旧页面触达即同步。
6. 第三轮运行态审计能确认现网与规范之间的真实差距。

### 3.2 当前差距表

| 理想状态 | 当前差距 | 处理方式 |
|---|---|---|
| 规范入口清晰 | 三份 Skill 职责和优先级存在冲突 | 修正 `SKILL.md` 的总入口和仲裁规则 |
| 文档与代码一致 | 部分文档、组件注释与源码冲突 | 回写 `SKILL-GLOBAL-COMPONENTS.md` 和组件注释 |
| 共享组件可信 | 部组织件文档仍混合旧规则、理想态和真实实现 | 高风险组件优先裁决并写回规范 |
| Tenant / Admin 分层稳定 | Tenant 仍有非 Tenant 变体、手写卡片和文本样式 | 收敛 `SKILL-TENANT.md`，触达即同步 |
| 业务页面复用组件 | 原生 `<table>`、硬编码色值、inline `boxShadow`、Badge 改色等残留 | 建立检查规则和 baseline |
| 运行态可判断 | 缺部署环境、账号、截图、构建 commit | 启动第三轮运行态审计 |

### 3.3 不能提前承诺的事项

在第三轮审计完成前，不建议对外承诺：

1. 现网所有页面已经与规范一致。
2. 所有历史模块已经完成换皮。
3. 线上已经不存在硬编码样式。
4. 线上已经完全组件化和 token 化。

更准确的表达是：

> **规范框架已经可以收敛，代码侧已具备事实基线；现网运行态差距需要第三轮审计确认。**

---

## 4. 实施建议：如何修改、迁移、合并

### 4.1 如何修改规范

#### 第一步：修正 `SKILL.md`

`SKILL.md` 应收窄为 AI 执行入口，不再写成设计系统百科。

应保留：

- 规范加载顺序
- Shared / Admin / Tenant 的判断方式
- 文档与代码冲突时的仲裁规则
- 禁止 AI 引用历史资料作为当前规范源
- 修改旧页面时的触达即同步原则
- 高风险组件需要优先检查的提示

应删除或迁出：

- 具体品牌色 hex 值
- 圆角尺寸表
- 阴影数值表
- 旧字体体系
- 组件视觉细节
- 与 `index.css` 或组件源码可能再次分叉的硬编码 token

#### 第二步：修正 `SKILL-GLOBAL-COMPONENTS.md`

这份文档应成为共享组件和 Admin 组件的主要说明文档。

修正原则：

1. 每个组件只保留用途、推荐用法、禁止用法、代码真源。
2. 删除容易漂移的硬数值，改为引用 `index.css` 或组件源码。
3. 对高风险组件直接给出最终规则。
4. 保留必要示例，但避免同一组件出现两套规则。

#### 第三步：修正 `SKILL-TENANT.md`

这份文档只写 Tenant 相对共享基线的差异，不重复完整共享组件规范。

应保留：

- Tenant 页面骨架
- Tenant 专属按钮变体
- Tenant 卡片与 `TenantCard` 使用规则
- Tenant Typography 要求
- Tenant 导航、背景、空态、营销视觉差异
- Tenant 与 Admin 冲突时的仲裁方式

应减少：

- 全局 token 全量表
- 共享组件完整说明
- 与 `SKILL-GLOBAL-COMPONENTS.md` 重复的基础规则
- 与组件源码冲突的旧数值

### 4.2 如何同步修组件源码

重点不是另写一份裁决文档，而是让文档、组件注释、组件 API 三者一致。

优先同步：

1. `Table`
2. `Card / SurfaceCard / TenantCard`
3. `Dialog / Drawer`
4. `Tabs / Segment / LineTabs`
5. `StatusTag`
6. `AdminSidebar`
7. `Typography`

处理原则：

1. 能直接修文档的，直接修文档。
2. 能直接修组件注释的，直接修组件注释。
3. 能通过组件 API 固化的，不继续写成 prose 规则。
4. 暂时无法修完的，先进入第三轮审计后的整改清单。

### 4.3 如何迁移旧页面

| 场景 | 建议 |
|---|---|
| 新页面 | 严格按修正后的规范执行，不允许新增旧写法 |
| 旧页面小改 | 触达即同步，至少不新增违规 |
| 旧页面大改 | 同步迁移组件、token、布局和文案样式 |
| 历史问题过多 | 使用 baseline，不要求一次性清零 |
| 高风险组件 | 优先迁移 Table、Card、Dialog、Tabs、StatusTag |

Baseline 原则：

```text
不要求今天清零所有历史问题；
但不允许新增问题；
改到哪个文件，就顺手降低该文件的违规数量。
```

### 4.4 如何合并 / 归档资料

| 类型 | 建议处理 |
|---|---|
| PRD | 保留为业务需求，不作为 UI 规范源 |
| 审计报告 | 保留为治理证据，不作为当前 UI 规范源 |
| 旧设计探索 | 归档，不再让 AI 引用为规范 |
| 旧迁移计划 | 若已完成或被吸收，归档 |
| 与源码冲突的旧规范 | 修正、删除或标记废弃 |
| 第三轮后新增整改项 | 视规模决定是否单独建整改清单 |

---

## 5. 高风险组件优先级

高风险组件裁决仍然需要做，但不必默认独立成文件。建议直接沉淀到 `SKILL-GLOBAL-COMPONENTS.md`、`SKILL-TENANT.md` 和组件源码中。

| 优先级 | 组件 | 主要问题 | 处理方向 |
|---|---|---|---|
| P0 | `Table` | 文档有两套规则，业务仍有原生 `<table>`，操作列写法不统一 | 统一使用 `@/components/ui/table`；禁止业务页原生表格；操作列使用 `TableActionCell + Button variant="link"` |
| P0 | `Card / SurfaceCard / TenantCard` | 圆角、卡片语义、Tenant / Admin 分流混乱 | 明确 Shared/Admin 与 Tenant 卡片语义，不用散落圆角 class 表达业务语义 |
| P0 | `Dialog / Drawer` | 文档注释与 Admin 实际大量使用共享 Dialog 冲突 | 决定共享 Dialog 是否双端通用，并修正文档和组件注释 |
| P1 | `Tabs / Segment / LineTabs` | 多套切换控件并存，Tenant 胶囊切换有覆盖实现 | 明确页面一级、内容切换、Tenant 胶囊分别使用哪套组件 |
| P1 | `StatusTag` | `text` / `fill` / `dot` 语义边界不清 | 明确状态、信息、分类、角色语义和废弃策略 |
| P1 | `AdminSidebar` | 文档 token 与代码实现不一致 | 以 `index.css` 与 `admin-sidebar.tsx` 为真源回写文档 |
| P1 | `Typography` | 组件已成熟，但 Tenant 落地不足 | Tenant 新页面强制使用语义 Typography，旧页面触达即同步 |

---

## 6. 第三轮审计补什么

### 6.1 第三轮审计定位

第三轮审计不是为了重新决定规范方向，而是为了确认运行态差距。

它要回答：

1. 现网 / 预发实际页面是否接近修正后的规范。
2. 哪些页面仍明显偏旧。
3. 哪些共享组件在运行态被稳定使用。
4. 哪些 token 仍新旧并存。
5. 下周换皮 / 合并后，哪些模块最值得优先治理。

### 6.2 启动前需要的素材

| 素材 | 是否必需 | 用途 |
|---|---:|---|
| 部署环境 URL | 必需 | 锁定运行态基线 |
| 环境说明 | 必需 | 区分现网、预发、demo 部署 |
| 构建 commit / tag / 构建号 | 必需 | 与代码分支准确对应 |
| 可登录账号与权限矩阵 | 必需 | 覆盖 Tenant / Admin 页面 |
| 本轮换皮 / 合并范围清单 | 必需 | 确定审计页面范围 |
| demo 基线 commit | 必需 | 对比 `feature/design-refresh-2026` |
| 规范文档版本 | 必需 | 防止报告期间文档变更 |
| 关键页面清单 | 必需 | 覆盖首页、列表、详情、表单、弹窗、空态 |
| 关键组件清单 | 必需 | 至少覆盖高风险组件 |
| 截图采集规则 | 必需 | 统一分辨率、浏览器、缩放比 |

### 6.3 第三轮后的处理方式

| 第三轮结果 | 建议处理 |
|---|---|
| 规范和组件基本已修正，运行态偏差少 | 直接回写 `SKILL-GLOBAL-COMPONENTS.md` / `SKILL-TENANT.md`，不新增长期文件 |
| 发现少量明确整改项 | 写入第三轮审计报告的整改清单，按触达即同步处理 |
| 发现大量跨模块遗留问题 | 再考虑新增临时整改 backlog |
| 高风险组件仍无法一次裁决 | 再考虑新增单独组件裁决记录 |

---

## 7. 项目文档整理规划

文档整理的目标不是“立刻删除所有旧资料”，而是减少规范入口，避免团队和 AI 继续误引用过时材料。

整理原则：

> **先盘点分类，再降级 / 归档，最后确认无引用后再删除。**

### 7.1 最终保留的规范入口

后续当前 UI 规范只保留以下入口：

| 文档 | 定位 | 处理 |
|---|---|---|
| `SKILL.md` | AI 总入口 / 规范路由 / 冲突仲裁 | 保留并修订 |
| `SKILL-GLOBAL-COMPONENTS.md` | 共享 / Admin 组件规范 | 保留并修订 |
| `SKILL-TENANT.md` | Tenant 差异规范 | 保留并修订 |

其他文档即使保留，也不应作为当前 UI 规范源。

### 7.2 审计资料：保留为证据

`docs/design-audit/**` 建议保留为审计证据和阶段性治理材料，不作为当前 UI 规范源。

| 文档 / 目录 | 建议处理 | 理由 |
|---|---|---|
| `design-spec-conflict-audit-2026-06-05.*` | 保留 | 支撑规范冲突判断 |
| `design-component-compliance-audit-2026-06-05.*` | 保留 | 支撑组件合规和高风险优先级 |
| `design-governance-and-delivery-plan-2026-06-05.*` | 保留 | 支撑阶段性交付判断 |
| `design-audit-round3-materials-and-template-2026-06-05.*` | 保留 | 第三轮运行态审计模板 |
| `component-index/**` | 暂保留 | 可作为组件引用和治理证据 |
| `figma-730_665/**` | 暂保留或后续归档 | 只读设计快照，不是规范源 |
| `ops-card-hierarchy-preview.html` / `toast-preview.html` | 归档候选 | 临时预览材料 |
| `changelog-2026-05-28.md` / `changed-files-2026-05-28.txt` | 归档候选 | 历史过程材料 |

同时建议更新 `docs/design-audit/README.md`，将目录定位改为：

> 本目录为设计审计证据与阶段性治理材料，不作为当前 UI 规范源。当前 UI 规范入口为 `SKILL.md`、`SKILL-GLOBAL-COMPONENTS.md`、`SKILL-TENANT.md`。

### 7.3 PRD / 需求文档：降级保留

PRD 和需求类文档应保留业务追溯价值，但不作为 UI 规范源。

建议降级保留：

- `PRD.md`
- `PRD_CORE_PAGES.md`
- `PRD_SESSION_DETAIL.md`
- `PRD_IMAGE_PUSH_UPDATE.md`
- `docs/PRD-*.md`
- `docs/MCP库需求.md`
- `docs/user_mcp_prd.md`
- `docs/skill-config-requirements.md`
- `docs/del_skill.md`
- `docs/企业技能库功能介绍.md`

建议在这类文档顶部统一加说明：

```md
> 本文档为业务需求文档，不作为 UI 设计系统规范源。
> 若其中的颜色、圆角、组件样式、布局描述与当前设计系统冲突，以 `SKILL.md`、`SKILL-GLOBAL-COMPONENTS.md`、`SKILL-TENANT.md` 和组件源码为准。
```

### 7.4 旧设计规范 / 迁移计划：吸收后归档

以下文档最容易被误认为当前规范，应在关键信息吸收进三份 Skill 后归档：

| 文档 | 建议处理 | 原因 |
|---|---|---|
| `docs/DESIGN_SYSTEM_BUTTON.md` | 吸收后归档 | Button 规范应回到 `SKILL-GLOBAL-COMPONENTS.md` 和 `button.tsx` |
| `docs/DESIGN_SYSTEM_PAGE_LAYOUT.md` | 吸收后归档 | 页面布局规则应回写 Skill，避免单独漂移 |
| `docs/TYPOGRAPHY_COLOR_MIGRATION_PLAN.md/pdf/html` | 吸收后归档 | 迁移计划不是当前规范源 |
| `docs/button-tokens/**` | 归档候选 | 若内容已吸收进 Button 规范，可归档 |
| `docs/admin-sidebar-vibe-coding.md` | 归档候选 | 过程记录，不是当前规范 |
| `docs/ClawPro管控端.md` | 人工确认后归档 | 若含 UI 规则，可能与新规范冲突 |

### 7.5 设计协作与研究类文档：保留但避免作为规范源

| 文档 / 路径 | 建议处理 |
|---|---|
| `docs/DESIGN_COLLAB_GUIDE.md` / `docs/DESIGN_COLLAB_GUIDE_LITE.md` | 保留一个主入口，另一个归档，避免分叉 |
| `docs/component-refs/**` | 保留但更新旧规范引用 |
| `client/public/research/**` | 不作为规范入口；如页面仍引用，先不要移动 |
| `dist/public/research/**` | 构建产物，不作为源文档处理 |
| `client/src/components/topnav/README.md` | 局部组件 README，保留 |
| `client/src/lib/smh-space-drive/README.md` | SDK 接入文档，保留 |
| `client/src/pages/admin/Security/api/*.md` | API 文档，保留 |

### 7.6 临时 / 过程文件：归档或候选删除

| 文件 | 建议处理 |
|---|---|
| `ideas.md` | 归档 |
| `todo.md` | 删除候选或归档 |
| `notice-bar-verification.md` | 归档候选 |
| `TEST_CASES.md` | 保留，属于测试资料 |
| `.codebuddy/**` | 不移动、不删除 |

### 7.7 建议归档结构

建议后续新增：

```text
docs/archive/
├── deprecated-design/
│   ├── DESIGN_SYSTEM_BUTTON.md
│   ├── DESIGN_SYSTEM_PAGE_LAYOUT.md
│   ├── TYPOGRAPHY_COLOR_MIGRATION_PLAN.md
│   └── admin-sidebar-vibe-coding.md
├── process-records/
│   ├── ideas.md
│   ├── notice-bar-verification.md
│   └── todo.md
├── old-collab-guides/
│   └── DESIGN_COLLAB_GUIDE_LITE.md 或 DESIGN_COLLAB_GUIDE.md
└── old-audit-artifacts/
    ├── ops-card-hierarchy-preview.html
    ├── toast-preview.html
    └── changelog-2026-05-28.md
```

如果担心文件移动影响引用，第一步可以不移动，只在顶部加 `Deprecated / Archived` 声明。

### 7.8 推荐清理顺序

1. **先不删除**，先给 PRD / 需求类文档加“非 UI 规范源”声明。
2. 更新 `docs/design-audit/README.md`，明确审计目录只是证据源。
3. 将旧设计规范中的有效内容吸收到三份 Skill。
4. 归档最容易误导的旧设计规范和迁移计划。
5. 修正旧引用，重点检查 `docs/component-refs/README.md` 和三份 Skill。
6. 等三份 Skill 修订完成、第三轮审计完成后，再考虑删除无引用文件。

---

## 8. 详细依据与附录

本报告依据以下材料，不在本文中重复展开全部证据。

| 材料 | 用途 |
|---|---|
| `docs/design-audit/design-spec-conflict-audit-2026-06-05.md` | 说明规范文档之间、文档与代码之间的高风险冲突 |
| `docs/design-audit/design-component-compliance-audit-2026-06-05.md` | 说明全量组件实现、业务调用和绕开共享组件的情况 |
| `docs/design-audit/design-governance-and-delivery-plan-2026-06-05.md` | 说明规范收敛框架和阶段性交付建议 |
| `docs/design-audit/design-audit-round3-materials-and-template-2026-06-05.md` | 说明第三轮运行态审计所需素材、证据口径和报告模板 |

## 9. 最终总结

本报告按“目标、现状、差距、如何做”的顺序给出判断：

1. **目标**：下周交付一套相对完整、后续能用的 ClawPro 规范。
2. **现状**：前两轮审计已足够支撑规范收敛方向，代码侧已有事实真源。
3. **差距**：三份 Skill 文档、组件注释、业务落地、文档入口和运行态审计仍需收口。
4. **如何做**：先修三份核心 Skill 和组件源码；旧页面触达即同步；项目文档先降级 / 归档，确认无引用后再删除。

最终建议：

> **不要把精力放在新增更多过程文件上；先把现有三份 Skill 文档和组件源码修成可信真源，同时清理旧文档入口，再用第三轮审计确认现网差距，并把结果回写到最终规范。**
