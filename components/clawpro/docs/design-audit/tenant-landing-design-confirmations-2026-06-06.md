# ClawPro Tenant / Landing 待设计确认清单

> 用途：这份文档不是让前端直接实现，而是给 Tenant / Landing 设计 owner 或其 AI 做设计确认与规范收口。
> 日期：2026-06-06
> 背景：当前 `clawpro-portable-design-skill` 已能作为周一交付包骨架，但 Tenant 和 Landing 仍有若干视觉与组件口径需要设计侧拍板。

## 1. 当前结论

- Admin 端的骨架、Table、Card、Button、Empty、Page Header 已经相对能收口到一版可执行规范。
- Tenant 端和 Landing 端仍有一些核心视觉规则处于“代码实现、历史规范、设计稿差异说明”并存状态。
- 这些问题不适合由实现侧自行裁决，否则会继续导致换皮后返工。

## 2. 当前 3 个非阻塞 warning

这 3 个 warning 已知，不是当前最主要风险，但需要说明：

1. `component-specs/card-surface.md` 和 `portable/react/card.tsx` 里 12px TenantCard fallback 会被脚本识别为“大圆角”。
2. 这不是实现错误，而是脚本还无法区分“Tenant 合法 12px”与“可疑大圆角”。
3. `references/assets-icons.md` 里的 `/icon/*.svg` 通配示例会被脚本识别为未登记图标引用，属于文档扫描误报。

建议：这三项暂不需要设计拍板，后续由我继续收口脚本规则即可。

## 3. Tenant 端待确认问题

以下项优先级最高，建议你同事先看。

### T-01 页面背景最终口径

当前状态：

- Tenant 规范已经明显从“白到灰渐变 / 点阵装饰”转向“白底 + 极淡蓝雾”。
- 但 portable pack 目前还没把这件事锁成一版足够可移交的最终参数。

需要确认：

- 最终是否明确采用“纯白底 + 两团极淡蓝雾”的方案。
- 蓝雾是否要在交付包里写死位置、透明度、范围。
- 点阵、贯穿线、装饰横竖线是否彻底废弃。

建议设计输出：

- 一版可交付参数表：颜色、位置、透明度、尺寸范围。
- 一句清晰裁决：`点阵装饰是否彻底废弃`。

影响文件：

- `.codebuddy/skills/clawpro-portable-design-skill/references/tenant.md`
- `.codebuddy/skills/clawpro-portable-design-skill/references/foundation.md`

### T-02 Tenant 业务卡片 12px 的适用范围

当前状态：

- 大方向已经明确是 `TenantCard = 12px 三态`。
- 但“哪些容器必须 12px、哪些容器仍保留 4px 或普通 Surface”这件事，仍容易在实现时混乱。

需要确认：

- Tenant 端哪些内容必须视为“业务卡片”。
- 表格容器、Tabs 容器、文章详情容器、配置卡这类外框，是否都统一进 TenantCard 体系。
- 纯内容承载容器与真正业务卡片的边界是什么。

建议设计输出：

- 一张“Tenant 页面容器分流表”：
  - TenantCard
  - SurfaceCard
  - SurfaceInner
  - 无卡片外框

影响文件：

- `.codebuddy/skills/clawpro-portable-design-skill/references/tenant.md`
- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/card-surface.md`

### T-03 Tenant 顶部导航最终规格

当前状态：

- 已知方向是半透明白底 + blur。
- 但 portable pack 还没把背景透明度、blur 强度、左右结构和最小宽度固定成一版最终交付值。

需要确认：

- TopNav 背景是否固定为 `rgba(255,255,255,0.4) + backdrop-blur`。
- Logo 区、中部切换区、右操作区的最终 spacing。
- 最小宽度与小屏策略是否就是“保持横向滚动，不做折行”。

建议设计输出：

- 一版 TopNav spec。
- 最好补一个 anatomy 截图或标注图。

影响文件：

- `.codebuddy/skills/clawpro-portable-design-skill/references/tenant.md`

### T-04 Tenant Tabs / Segment / Text Switch 分流规则

当前状态：

- Tenant 至少存在三类切换：主胶囊切换、较弱文字切换、历史误用的 Admin Segment。
- 现在最容易混乱的是：什么场景必须用胶囊滑块，什么场景应该用弱切换文本。

需要确认：

- 页面主分类切换是否统一强制用胶囊滑块。
- 轻量辅助切换是否统一使用 text switch，而不是胶囊。
- Tenant 页面内是否彻底禁止使用 Admin 4px 矩形 Segment。

建议设计输出：

- 一张切换器分流表：
  - 主导航切换
  - 页面主分类切换
  - 轻量双态开关
  - 详情页子标签

影响文件：

- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tabs-segment.md`
- `.codebuddy/skills/clawpro-portable-design-skill/references/tenant.md`

### T-05 Tenant 表单控件圆角口径

当前状态：

- 这是当前最容易冲突的一项。
- 一部分资料和组件实现指向 Tenant 表单控件胶囊化。
- 另一部分设计说明又保留“Input / Select / DatePicker 4px”的说法。

需要确认：

- Tenant 的 Input / Select / DatePicker 最终到底是 `4px` 还是 `full`。
- 是否存在“搜索 / 筛选 = 胶囊，普通表单输入 = 4px”的双轨口径。
- 弹窗内表单控件与页面筛选控件是否允许分流。

建议设计输出：

- 直接给一个表：
  - 搜索框
  - 普通 Input
  - Select
  - DatePicker
  - 弹窗表单控件
  - 筛选条控件

影响文件：

- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/form-controls.md`
- `.codebuddy/skills/clawpro-portable-design-skill/references/tenant.md`

### T-06 Tenant 空状态层级

当前状态：

- 当前交付包已区分页面级和容器级空状态。
- 但 Tenant 端到底要不要比 Admin 更轻、更少插画，目前还缺明确设计拍板。

需要确认：

- Tenant 页面级空状态是否也使用通用兔子 / PNG 插画。
- Tenant 容器级空态是否统一降级为纯文字说明。
- Tenant 端空态文案是否需要更偏引导式。

影响文件：

- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/empty-state.md`
- `.codebuddy/skills/clawpro-portable-design-skill/references/tenant.md`

### T-07 Tenant Typography 推广边界

当前状态：

- 方向上已经希望 Tenant 走语义 Typography。
- 但像 TopNav、Hero、卡片标题、辅助文案，是否全部强制用 Typography 语义，还没彻底收口。

需要确认：

- Tenant 页面哪些文字必须语义化。
- 哪些区域允许保留特殊排版，不强行进 Typography。

影响文件：

- `.codebuddy/skills/clawpro-portable-design-skill/references/tenant.md`

## 4. Landing 端待确认问题

Landing 优先级低于 Tenant / Admin，但如果周一之后也可能换皮或新做，现在最好先让设计同事给出方向性裁决。

### L-01 Landing 是“新做”还是“已有页面换皮”

需要确认：

- 当前 landing 是纯新做，还是存在已有版本要换皮。
- 如果是已有页面换皮，哪些历史模块必须保留结构，哪些可以重做。

影响文件：

- `.codebuddy/skills/clawpro-portable-design-skill/references/landing.md`

### L-02 Hero 的最终视觉方向

当前状态：

- 交付包里只收了方向性原则，还没有锁 Hero 的精确视觉语言。

需要确认：

- Hero 是更产品截图导向，还是更品牌视觉导向。
- CTA 用黑到蓝渐变是否固定。
- Hero 中是否必须带真实产品截图 / 架构图，而不是抽象插画。

建议设计输出：

- 一版 Hero anatomy。
- CTA 样式与主副按钮关系。

### L-03 Landing 与 Tenant TopNav 的关系

需要确认：

- Landing 是否复用 Tenant TopNav。
- 如果复用，是完全复用还是做轻改版。
- 若不复用，差异点是什么。

影响文件：

- `.codebuddy/skills/clawpro-portable-design-skill/references/landing.md`
- `.codebuddy/skills/clawpro-portable-design-skill/references/tenant.md`

### L-04 Landing 区块节奏与卡片体系

需要确认：

- Landing 的能力卡、场景卡、流程卡，是否沿用产品卡片体系，还是单独一套 marketing card。
- 区块间距、背景层次、截图与文字比例是否有固定框架。

建议设计输出：

- 一套区块模板：Hero / Capability / Workflow / Governance / CTA。

### L-05 Landing 资产来源

需要确认：

- Landing 首选真实产品截图，还是允许额外插画。
- 如果需要视觉资产，哪些来自现有仓，哪些需要新出。
- 新资产如何进 registry。

影响文件：

- `.codebuddy/skills/clawpro-portable-design-skill/references/landing.md`
- `.codebuddy/skills/clawpro-portable-design-skill/references/assets-icons.md`

## 5. 跟 Tenant / Landing 强相关的组件待确认

### C-01 Dialog / Drawer 分端差异

需要确认：

- Tenant 弹窗 / 抽屉是否沿用 Admin 4px 浮层口径。
- 还是 Tenant 也应该有更柔和的浮层圆角 / footer 按钮风格。

影响文件：

- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/dialog-drawer.md`

### C-02 Form Controls 分端差异

需要确认：

- Tenant 表单控件是统一胶囊，还是页面筛选与普通输入分流。

影响文件：

- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/form-controls.md`

### C-03 Tabs / Segment 分端差异

需要确认：

- Tenant 强切换 / 弱切换是否要拆成两种明确组件规范。

影响文件：

- `.codebuddy/skills/clawpro-portable-design-skill/component-specs/tabs-segment.md`

## 6. 建议同事确认后的回写顺序

建议同事确认完后，按下面顺序回写：

1. 先改 `tenant.md` 和 `landing.md`
2. 再改 `foundation.md` 里受影响的全局描述
3. 再改相关 `component-specs/*.md`
4. 最后更新 `qa/*.md` 和 `STATUS.md`

## 7. 给同事 AI 的最短任务口径

可以把下面这段直接发给同事的 AI：

```text
请先读取：
1. docs/design-audit/tenant-landing-design-confirmations-2026-06-06.md
2. .codebuddy/skills/clawpro-portable-design-skill/references/tenant.md
3. .codebuddy/skills/clawpro-portable-design-skill/references/landing.md
4. .codebuddy/skills/clawpro-portable-design-skill/component-specs/tabs-segment.md
5. .codebuddy/skills/clawpro-portable-design-skill/component-specs/form-controls.md
6. .codebuddy/skills/clawpro-portable-design-skill/component-specs/dialog-drawer.md

目标：不要直接写实现，先从设计规范角度确认 Tenant 和 Landing 仍未拍板的视觉与组件口径，并输出明确裁决建议。
```

