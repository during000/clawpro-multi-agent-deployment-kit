# ClawPro 周一协作安排与后续工作计划

> 用途：周一回到公司后，用这份文档向同事介绍当前工作、组织复核、推动后续交付开发与组件库化。  
> 日期：2026-06-06  
> 当前交付包主体：`.codebuddy/skills/clawpro-portable-design-skill/`

## 1. 一句话介绍

这轮已经完成的是一套 **ClawPro 新皮肤 portable design pack**。

它不是最终组件库，也不是单纯设计规范文档，而是一套给设计、前端和 AI 共同使用的交付包：

- 设计同事可以用它确认全局、Admin、Tenant、Landing 的视觉口径。
- 前端同事可以用它判断宿主仓怎么换皮、现有组件怎么映射、没有组件时怎么 fallback。
- AI 可以用它作为统一规范来源，继续做页面、审查、补 spec 或生成迁移建议。

## 2. 周一交付包在哪里

正式交付包主体：

```text
.codebuddy/skills/clawpro-portable-design-skill/
```

协作材料入口：

```text
docs/design-audit/README.md
```

设计确认结果：

```text
docs/design-audit/design-confirmation-decisions-2026-06-06.md
```

新对话 / 同事 AI 续接 prompt：

```text
docs/design-audit/new-chat-resume-template-2026-06-06.md
```

## 3. 周一开场可以这样说

```text
我这轮不是直接做组件库，也不是只写一份设计规范，而是整理了一套 ClawPro 新皮肤 portable design pack。

目标是让产品前端、设计同事和 AI 都能基于同一套规则继续做管理端、客户端和落地页换皮，不依赖当前 demo 仓组件一定能被复用。

如果宿主仓不能直接用 demo 组件，这个包里也提供了 React / HTML-CSS fallback、迁移规则和 QA checklist。

今天希望大家帮忙确认三件事：
1. 设计确认结果是否准确；
2. component specs 是否能指导前端落地；
3. 哪些内容可以进入下一阶段组件库化。
```

## 4. 建议会议流程

### 4.1 5 分钟：说明交付目标

说明本轮交付不是：

- 不是完整 npm 组件库。
- 不是要求前端直接迁移到 demo 仓全部组件。
- 不是让前端自己从多份规范中二次仲裁。

而是：

- 先统一设计裁决。
- 先统一组件映射。
- 先提供 portable fallback。
- 先给后续组件库化提供可验证基线。

### 4.2 10 分钟：讲交付包结构

打开：

```text
.codebuddy/skills/clawpro-portable-design-skill/
```

重点说明：

| 目录 / 文件 | 作用 |
|---|---|
| `README.md` | 人类入口 |
| `SKILL.md` | AI 入口 |
| `STATUS.md` | 当前状态与续接板 |
| `references/` | 全局、Admin、Tenant、Landing、页面模板、迁移规则 |
| `component-specs/` | 高风险组件 portable spec |
| `portable/` | React / HTML-CSS fallback 示例 |
| `tokens/` | 颜色、文字、描边、圆角、阴影 token |
| `qa/` | 验收 checklist |
| `scripts/` | 校验和打包脚本 |

### 4.3 10 分钟：讲已确认设计决策

打开：

```text
docs/design-audit/design-confirmation-decisions-2026-06-06.md
```

重点讲已经收口的点：

- 品牌蓝统一为 `#1447E6`。
- 文字色用蓝灰 / slate 语义 token。
- 描边统一蓝灰 token：`--border = #EAEEF4`。
- AdminSidebar：展开 `240px`，收起 `64px`。
- Admin 默认卡片无投影。
- TenantCard 默认有轻投影。
- 表格整体 `12px`。
- Tenant 业务卡使用 `12px`，但数据密集容器按组件 spec 分流。
- Tenant 表单控件双轨：搜索 / 筛选胶囊，普通表单 4px。
- Tenant Text Switch 删除，不进入周一交付包。
- Landing 独立维护，但周一只锁方向和规则。
- portable fallback 不散写业务色 hex，优先 token / CSS variable。

### 4.4 10 分钟：分角色认领复核

| 角色 | 建议负责内容 |
|---|---|
| miekoyychen | 全局口径、Admin、交付包收口、后续路线 |
| 客户端设计同事 | `tenant.md`、Tenant 卡片、TopNav、Tabs、表单、空态 |
| 落地页设计同事 | `landing.md`、Hero、区块、资产规则 |
| 前端同事 | `component-specs/`、`portable/`、`migration-map.md` |
| 组件库 owner | 判断哪些 spec 可升级成组件库 API |
| AI 协作 | 根据 `new-chat-resume-template` 读取规范并辅助补充 / 审查 |

## 5. 周一当天建议产出

周一当天不建议把目标定成“所有问题全部解决”，建议产出这几项：

1. 确认 `.codebuddy/skills/clawpro-portable-design-skill/` 可以作为后续执行基线。
2. 确认 `design-confirmation-decisions-2026-06-06.md` 中哪些项已最终确认，哪些还要 owner 回看。
3. 选出 1-2 个试点页面。
4. 选出最先组件库化的 3-5 个组件。
5. 明确下一轮 owner 和截止时间。

## 6. 建议试点页面

不要一上来全量改，先验证这套 pack 是否真的能指导落地。

建议试点：

| 试点 | 目的 |
|---|---|
| 一个 Admin 列表页 | 验证 Admin 背景、PageHeader、SearchFilterBar、Table、Pagination、Empty |
| 一个 Tenant 卡片列表页 | 验证 Tenant 背景、TenantCard、Button、Tabs、空态 |

试点要回答：

- `references/admin.md` / `references/tenant.md` 是否够用？
- `component-specs/table.md` / `card-surface.md` 是否够落地？
- `portable/` fallback 是否能被前端接受？
- `migration-map.md` 是否能指导宿主仓换皮？

## 7. 后续工作计划

### 阶段 1：确认 portable design pack 正确性

目标：确认这套 pack 能不能作为后续执行基线。

要做：

1. 设计同事复核 `references/`。
2. 前端同事复核 `component-specs/` 和 `portable/`。
3. 把不对的地方回写到：
   - `.codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md`
   - `docs/design-audit/design-confirmation-decisions-2026-06-06.md`
   - 对应 spec 文件

产出：

```text
一版可接受的 ClawPro portable design pack
```

### 阶段 2：产品前端试点落地

目标：用 1-2 个真实页面验证交付包能否指导换皮。

要做：

1. 选 Admin 列表页试点。
2. 选 Tenant 卡片列表页试点。
3. 前端按 spec 和 fallback 落地。
4. 设计按 QA checklist 验收。
5. 记录 spec 缺口和误解点。

产出：

```text
试点页面问题清单
spec 需要补充的地方
```

### 阶段 3：组件库级别交付

目标：把试点验证过的高频 spec 升级为组件库 API。

优先级建议：

1. `Button`
2. `SurfaceCard / TenantCard`
3. `Table`
4. `SearchFilterBar`
5. `DatePicker`
6. `Empty`
7. `StatusTag / Badge`
8. `Pagination`
9. `Dialog / Drawer`
10. `Tabs / Segment`

每个组件库化前需要补：

- API
- variants
- token 映射
- Admin / Tenant 分流
- fallback 策略
- demo / preview
- usage / anti-pattern
- QA checklist

## 8. 可以发给同事的消息

```text
我这轮已经整理了一套 ClawPro 新皮肤的 portable design pack，路径是：

.codebuddy/skills/clawpro-portable-design-skill/

它不是最终组件库，而是一套可交付、可被前端和 AI 消费的设计规则包，里面包含：
- references：全局 / Admin / Tenant / Landing 规范
- component-specs：高风险组件规范
- portable：React / HTML-CSS fallback
- tokens：颜色、文字、描边、圆角、阴影 token
- qa：验收 checklist

请大家先看：
1. docs/design-audit/README.md
2. .codebuddy/skills/clawpro-portable-design-skill/README.md
3. .codebuddy/skills/clawpro-portable-design-skill/STATUS.md
4. docs/design-audit/design-confirmation-decisions-2026-06-06.md
5. docs/design-audit/monday-collaboration-plan-2026-06-06.md

如果要让 AI 协助续接，请直接使用：
docs/design-audit/new-chat-resume-template-2026-06-06.md

这次希望大家帮忙确认三件事：
1. 设计确认结果是否准确；
2. component specs 是否能指导前端落地；
3. 哪些内容可以进入下一阶段组件库化。
```

## 9. 风险和边界

- `C-016`、`C-019` 仍有“暂选 / 后续进一步确认”属性，不要包装成不可变更最终裁决。
- 历史页面 hardcoded style 是存量债务，不是本轮交付包阻塞项。
- 组件库化不要直接从文档跳到全量建设，应先通过试点页面验证。
- 如果同事更新 Tenant / Landing 决策，需要回写 `design-confirmation-decisions-2026-06-06.md` 和对应 `references/` 文件。

## 10. 最终定位

这轮工作的完成状态可以这样定位：

```text
第一阶段已完成：把散落的设计规则、冲突确认、组件规范、fallback 和协作入口整理成一个 portable design pack。

周一要推进第二阶段：让设计、前端和 AI 一起复核这套 pack 是否准确，并选试点页面验证。

验证通过后，再进入第三阶段：把高频组件沉淀成组件库级别交付。
```
