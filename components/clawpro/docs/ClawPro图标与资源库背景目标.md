# ClawPro 图标与资源库背景目标

> 适用项目：`/Users/miekoyychen/openclaw-enterprise`  
> 文档定位：说明为什么要建设 ClawPro 图标与资源库、资源范围、分类原则、治理目标和边界。  
> 配套计划：实施步骤见 `docs/ClawPro图标与资源库建设计划.md`。

---

# 一、背景

当前项目中的图标和图片资源来源较多，主要分布在：

| 位置 | 资源类型 | 说明 |
|---|---|---|
| `client/public/assets/**` | SVG / PNG | 管控端业务资源、侧栏资源、管理页资源等 |
| `client/public/assets/avatars/**` | PNG | Agent 头像资源 |
| `client/public/assets/admin-channel-icons/**` | SVG | 企微、飞书、钉钉、QQ 等渠道图标 |
| `client/public/icon/**` | SVG / PNG | 业务图标，命名和尺寸不完全统一 |
| `client/src/assets/**` | SVG / PNG | 源码内通过 Vite import 使用的资源 |
| `icon/**` | SVG | 根目录旧素材或源素材 |
| `assets/**` | SVG / PNG | Figma / CodeBuddy 导出资源，稳定性不一 |
| TSX 文件内 | inline SVG | 多个组件和页面内存在内联 SVG |
| `lucide-react` | React Icon | 项目大量使用的通用图标来源 |

目前资源使用方式并存：

```tsx
<img src="/assets/avatars/avatar-developer.png" />
```

```tsx
import iconUrl from "@/assets/icons/example.svg";
```

```tsx
function ExampleIcon() {
  return <svg>{/* ... */}</svg>;
}
```

```tsx
import { Settings } from "lucide-react";
```

这导致团队难以快速判断：

1. 项目里到底有哪些资源。
2. 哪些资源可以复用。
3. 哪些资源是重复的。
4. 哪些资源适合组件使用。
5. skill 生成页面时应该选择哪个资源。
6. 修改某个图标时，会影响哪些使用处。

---

# 二、本阶段目标

本阶段不是建设完整资产平台，而是完成一次可落地、可合入 `main` 的资源库建设：

```text
资源扫描
+ 资源分类
+ 自有 SVG 使用场景 / 视觉类型细分
+ 资源库单页
+ 重复资源实际治理
+ canonical 统一入口
+ 组件槽位对齐
+ skill-map 建立
+ 资源使用校验
```

核心目标：

1. 帮产品设计团队看清当前项目有多少图标和图片资源。
2. 让资源可以按类型、场景、视觉风格、来源目录、使用情况进行浏览。
3. 对重复资源做一次实际治理，而不只是页面告警。
4. 对高频 canonical 资源建立统一入口，逐步具备“一改多处生效”能力。
5. 让高频组件通过资源槽位与资源库对齐。
6. 为后续页面设计、组件开发和 vibe coding 提供可查询的资源参考。
7. 建立 `clawpro-portable-design-skill` 与资源库之间的机器可读连接。

---

# 三、本阶段明确不做

| 不做项 | 原因 |
|---|---|
| 不做私有 npm 包 | 当前目标是盘点、分类、治理和展示，不是跨项目分发 |
| 不做全量资源迁移 | 会扩大 diff 和回归风险，不利于本次安全合入 `main` |
| 不做强注册体系 | 资源库先作为资源清单和治理视图，不强制所有业务先注册再使用 |
| 不做在线上传 / 删除 / SVG 编辑器 | 避免权限、XSS、路径、误删等风险 |
| 不做复杂审批流 | 本阶段以设计团队确认 + PR Review 为治理流程 |
| 不做“推荐复用”独立视图 | 复用建议作为资源标签展示，不单独维护一个视图 |
| 不做 Landing 大图 / 页面截图管理 | 不是本次资源库重点，避免资源库膨胀 |
| 不做全量组件重构 | 只对齐高频组件槽位和已治理重复资源涉及的组件引用 |
| 不承诺所有历史引用自动同步 | 只有接入同一物理文件、`canonicalAssets` 或专属组件的资源才具备一改多处生效能力 |

---

# 四、资源范围

## 4.1 纳入范围

| 类型 | 是否纳入 | 用途 |
|---|---:|---|
| `lucide-react` 已使用图标 | 是 | 展示项目当前实际使用的通用图标 |
| ClawPro 自有 SVG 图标 | 是 | 本次分类和治理重点 |
| inline SVG | 是，作为治理对象 | 判断是否抽为资源、替换为已有图标或保留组件私有 |
| Agent 头像 | 是 | 方便设计和业务复用参考 |
| 渠道图标 | 是 | 统一展示企微、飞书、钉钉等渠道资源 |
| 品牌 Logo | 是 | 明确品牌资源边界和使用限制 |
| 高频业务 PNG / SVG | 是 | 纳入复用价值较高的业务资源 |
| Figma / CodeBuddy 临时导出资源 | 仅扫描，不默认纳入 | 标注资源来源（非独立类别）；被实际引用的会随常规扫描覆盖、不会遗漏，主要用于识别重复副本或污染资源 |

## 4.2 不纳入范围

| 类型 | 是否纳入 | 原因 |
|---|---:|---|
| Landing page 大图 | 不纳入 | 页面专用、体积大、复用低 |
| onboarding 视频 | 不纳入 | 非图标 / 图片资源库重点 |
| 页面参考截图 | 不纳入 | 属于设计资料，不是运行时资源 |
| 设计审查素材 | 不纳入 | 不应进入资源消费体系 |
| 单页面大型插画 | 默认不纳入 | 复用价值低，容易污染分类 |
| 空状态大插画 | 默认不纳入 | 继续由现有 Empty 体系或页面私有资源处理 |

---

# 五、资源分类体系

资源库页面需要让设计团队能快速理解资源，因此分类不只按目录或文件类型，而是按“资源类型 + 使用场景 + 视觉类型”组织。

## 5.1 一级分类

```text
全部资源
Lucide 图标
自有 SVG 图标
Agent 头像
渠道图标
品牌 Logo
业务图片
重复治理
未分类 / 待确认
```

说明：

- `Lucide 图标`：展示项目已使用的 `lucide-react` 图标，以及后续被加入 skill-map 的图标。
- `自有 SVG 图标`：重点细分对象，需要按使用场景和视觉类型进一步分类。
- `Agent 头像`：展示头像资源，不混入普通图标。
- `渠道图标`：展示第三方渠道图标，保持品牌固定色限制。
- `品牌 Logo`：展示 ClawPro 等品牌资源，禁止当普通图标改色。
- `业务图片`：只放复用价值较高的业务图片，不包含 Landing 大图。
- `重复治理`：按重复组展示治理进度。
- `未分类 / 待确认`：承接扫描后无法自动判断的资源。

## 5.2 自有 SVG：按使用场景细分

| 使用场景 | 说明 | 示例用途 |
|---|---|---|
| `navigation` 导航入口 | 用于 Sidebar、TopNav、菜单入口 | 管理后台菜单、产品模块入口 |
| `action` 操作行为 | 表示用户操作 | 上传、下载、复制、刷新、编辑、删除 |
| `search-filter` 搜索筛选 | 搜索、筛选、过滤条件 | 搜索框、筛选器、工具栏 |
| `status` 状态反馈 | 表示运行状态或结果 | 成功、失败、警告、处理中、禁用 |
| `metric` 数据指标 | 用于统计卡、用量卡、监控卡 | Tokens、请求数、余额、成功率 |
| `agent` Agent 业务 | Agent、角色、能力、技能、智能体 | AgentCard、Agent 详情、Agent 创建页 |
| `model-ai` 模型 / AI | 模型、推理、Prompt、Token、AI 能力 | 模型配置、AI 能力说明 |
| `channel` 渠道业务 | 通信渠道、第三方平台 | 企微、飞书、钉钉、QQ |
| `security` 安全权限 | 权限、安全、审计、风险 | 安全组、审计日志、权限设置 |
| `file-resource` 文件资源 | 文件、目录、存储、上传资源 | 文件浏览器、资源管理 |
| `billing-quota` 计费配额 | 用量、额度、账单、余额 | 配额页、账单页、用量监控 |
| `brand-product` 品牌产品 | 产品 Logo、品牌标识、产品专属符号 | ClawPro Logo、ClawPro 产品标识 |
| `empty-hint` 轻量提示 | 小型引导、轻装饰图标 | 非大插画级别的提示图标 |

分类原则：

1. 一个资源只能有一个主使用场景。
2. 一个资源可以有多个辅助使用场景。
3. `channel`、`brand-product` 类资源必须保留品牌固定色，不允许被当作普通 UI 图标改色。
4. `empty-hint` 只纳入小图标级提示资源，大型空状态插画不进入本次重点。

## 5.3 自有 SVG：按视觉类型细分

| 视觉类型 | 说明 | 使用建议 |
|---|---|---|
| `line` | 单色描边，接近 lucide 风格 | 操作、导航、表格动作 |
| `solid` | 填充图形 | 状态、模块入口、强调场景 |
| `duotone` | 两种主色层级 | 业务对象、能力说明 |
| `gradient` | 带品牌渐变或装饰渐变 | NumberCard、能力卡、重点模块 |
| `brand-fixed` | 品牌固定色 | Logo、渠道图标、第三方品牌，不允许随意改色 |
| `avatar-like` | 头像类或角色形象 | Agent 相关场景 |
| `badge-emblem` | 徽章、盾牌、角标类 | 安全、认证、状态强调 |
| `illustrative-icon` | 插画化小图标，但不是大插画 | 引导、轻量空提示、功能说明 |
| `monochrome-currentColor` | 可通过 `currentColor` 控色 | 通用 UI 图标，适合组件内复用 |
| `asset-fixed-color` | 固定色业务资源 | 不建议改色，只按资源原貌使用 |

视觉类型原则：

1. `monochrome-currentColor` 优先进入通用组件槽位。
2. `gradient` 更适合 `NumberCard.icon`、能力说明卡，不适合普通按钮图标。
3. `brand-fixed`、`asset-fixed-color` 禁止进入普通操作图标槽位。
4. `avatar-like` 不作为普通 icon 使用。
5. `illustrative-icon` 可以用于轻量提示，但不替代 Empty 大插画体系。

## 5.4 资源状态

```text
normal        正常
preferred     优先使用
duplicate     重复资源
resolved       已治理
needs-review   待确认
avoid          不建议使用
deprecated     已废弃
```

说明：

- 不设“推荐复用”独立视图，`preferred` 只作为资源标签出现。
- `avoid`、`deprecated` 资源不得进入 skill-map。
- `needs-review` 资源可以展示，但不能作为 skill 自动选择候选。

---

# 六、图标选择决策模型

> 阶段 8 组件资源审计 + 阶段 9 skill 连接已落地，本节据真实产出补全（原占位的「四道筛预设」以审计结果据实化）。完整规则见 skill `references/assets-icons.md` §1 优先级 + §5.5 槽位选图，与《建设计划》§5.5 / §6.3 一致、不漂移。

## 6.1 选图判断顺序（据实）

1. **判断上下文与端别**：当前项目页面 / 页面级非组件代码 / 组件源码 / 跨仓页面；Admin / Tenant / Landing / Global。
2. **通用 UI**（导航 / 操作 / 状态 / 表单辅助）：默认 `lucide-react`，不因资源库建设强行替换（保留原始 skill 立场）。
3. **业务 / 自有 SVG**：判断是否命中组件槽位；命中则查 `resource-skill-map.json` 对应槽位 `candidates`，按视觉风格 / 尺寸 / 场景挑选。
4. **回退策略**：槽位 `allowLucideFallback=true`（`admin-sidebar` / `file-type`）无合适候选可回退 lucide；`=false`（`card-left-icon` 多彩渐变 / `run-status` 动画 / `feature-card` 特性卡 / `number-card` 统计卡渐变）lucide 难等价，无候选时标 `needs-design-confirmation`，不私自画 inline SVG、不回退扁平 lucide（会破坏渐变家族）。
   > 修订（2026-06-22）：`number-card` 由 `true` 移入 `=false` 组——阶段 9 派生候选已扩至 22 枚渐变图标（槽位充足），扁平 lucide 反而破坏渐变家族，故据实撤回兜底、对齐 `card-left-icon`。详见 ADR §4 修订补注。
5. **红线资源**（品牌 Logo / 渠道图标 / Agent 头像）：硬约束不变——保留品牌固定色、禁当普通 UI 图标改色、禁进普通 icon 槽位；当前项目页面经 `canonicalAssets.<group>.<key>` 引用，跨仓由宿主注入（`usageScope=host-injected`）。

## 6.2 槽位选图约束（阶段 8/9 真实产出）

`resource-skill-map.json` 收录 **9 槽位、155 候选**（槽位 141 + 红线 14）。每槽位的 `allowLucideFallback` / `recommendedResourceType` / `redline` / 风险等级取自阶段 8 `component-resource-map.json`，`candidates` 由 inventory 审计字段确定性派生（`status=normal` + `componentSlot` 或红线 `canonicalKey` + 未排除 / 未被治理移除）。具体槽位表见 skill `assets-icons.md` §5.5。

> 不变的硬边界以 §5 资源分类体系为准：`brand-fixed` / `asset-fixed-color` 禁进普通操作图标槽位、`avatar-like` 不作普通 icon（见 §5.2 / §5.3）。

---

# 七、与现有 icon-registry.example.json 的关系，及使用保证机制

## 7.1 与现有 registry 的关系

`clawpro-portable-design-skill` 已自带一份示例 registry：

```text
.codebuddy/skills/clawpro-portable-design-skill/assets/icon-registry.example.json
```

本资源库的产出**不另起炉灶、不与它漂移**，但经阶段 8 组件资源审计，分工据实调整如下：

| 文件 | 角色 | 事实源 |
|---|---|---|
| `icon-registry.example.json` | skill 自带的**可移植身份样例**，供跨仓移交时参照建立宿主仓正式 registry | 跨仓样例：**不是**本项目资源真相 |
| `resource-inventory.generated.json` | 本项目资源**真相**（扫描 + 分类 + 治理 + canonical 审计字段） | **是**：本项目资源身份与候选准入事实源 |
| `resource-skill-map.json` | 已审计资产**如何被槽位选用**（槽位、风格、是否可回退 lucide、usageScope） | 否：由 inventory 确定性派生，slot 约束取自阶段 8 `component-resource-map` |
| `generated/*.json` | 资源库单页展示数据（清单 / 使用 / 重复 / 治理） | 否：扫描产物 |

单一事实源原则（阶段 9 据真实审计修正）：

1. **本项目**资源的 `path` / `status` / 候选准入以 inventory 为准（`status=normal`、`componentSlot` / `canonicalKey`、治理状态）。
2. `resource-skill-map.json` 候选由 inventory 确定性派生，不引用 `needs-review` / `avoid` / `deprecated` / 已治理移除资源。
3. `icon-registry.example.json` 降格为 skill 可移植身份样例，**不**作为本项目候选准入闸门；跨仓移交时供宿主仓建立正式 registry 参照。
4. 检查脚本（`npm run check:skill-map`）校验 skill-map 候选都能在 inventory 命中、状态合法、磁盘存在、槽位与 `usageScope` 正确，防 skill-map 与 inventory / 阶段 8 槽位规格漂移。

> **口径修正说明（阶段 9）**：原设想「registry 是资产身份唯一事实源、skill-map 只引用其 `approved` 资产」。阶段 8 审计表明本项目带确认槽位资源 141 项、红线 14 项，与那份 28 条样例几乎无交叉关联（inventory 848 项中仅 1 项 `registry.registered`），故据实改为「以 inventory 为本项目真相、registry 为可移植样例」。品牌 / 渠道 / Agent 头像红线硬约束不变。

## 7.2 使用保证机制（三层）

“资源库建好”不等于“vibe coding 时一定会用到”。需要三层共同保证：

| 层 | 机制 | 作用 |
|---|---|---|
| 第一层 触发 & 引导 | skill 随页面任务加载，给出选图判断流程 | 让 AI 写图标前有明确判断流程，而非默认 reach for lucide |
| 第二层 数据保证 | 高频槽位在 `resource-skill-map.json` 中以槽位定义约束候选 | 关键位置不依赖 AI 临场判断，照表执行 |
| 第三层 脚本兜底 | `check-design-usage.mjs` 检测**确定性错误**：品牌/渠道/头像被 lucide 替代、未登记 SVG、内联手搓 SVG、emoji | 引导失效时兜底告警 |

> 第一、二层的“判断顺序”与“槽位字段的具体取值”依赖阶段 8 组件资源审计产出，不在背景阶段预设；第三层脚本只拦确定性错误，不对“业务语义用了 lucide”告警。

---

# 八、关键结论

ClawPro 资源库本阶段应定位为：

> **设计团队能看懂、项目能治理、skill 能读取的轻量资源体系。**

核心取舍：

1. 不以 npm 包作为本次目标。
2. 不做全量资源迁移。
3. 不做强注册体系。
4. 单页重点是分类呈现、资源详情、使用位置、重复治理。
5. 自有 SVG 必须同时按使用场景和视觉类型细分。
6. 重复资源治理必须有实际处理结果，不只停留在页面告警。
7. 高频 canonical 资源需要建立统一入口，逐步具备“一改多处生效”能力。
8. 高频组件需要通过资源槽位与资源库对齐，但不做全量组件重构。
9. skill 通过 `resource-skill-map.json` 与资源库建立连接，并优先使用 canonical 资源。
10. 具体的图标选择规则（语义 / 风格 / 尺寸 / 槽位的判断与回退策略）待阶段 8 组件资源审计后据实补充，不在背景阶段预设；在此之前品牌、渠道、Agent 头像仍为硬约束，必须用已登记 registry 资产且禁止改色（见 §5）。
11. `resource-skill-map.json` 与现有 `icon-registry.example.json` 不漂移；经阶段 8 审计，本项目资源真相为 inventory，skill-map 候选由 inventory 确定性派生，`icon-registry.example.json` 降格为 skill 可移植身份样例、不作 `approved` 准入闸门（详见 §七 口径修正）。品牌 / 渠道 / Agent 头像红线硬约束不变。
12. 通过“引导 + 槽位数据约束 + 检查脚本兜底”三层机制，保证 vibe coding 时真正用到资源库，而非只把资源库当展示页；其中槽位数据的具体字段取值以阶段 8 审计产出为准。
13. 通过检查脚本和规则保证组件和 skill 不乱用资源。
14. 本阶段优先保证安全、低侵入、可 review、可合入 `main`。
