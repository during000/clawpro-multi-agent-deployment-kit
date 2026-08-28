# Assets & Icons

> 图标、插画、空状态、视觉资产使用规则。
> - **跨仓 / 可移植语境**：`assets/icon-registry.example.json` 是 skill 自带的「可移植身份样例」，记录品牌 / 业务语义资产的身份样例，供移交和宿主仓建立正式 registry 时参照；它**不是**某个具体项目的资源真相，也**不**作为候选准入闸门。
> - **当前项目页面语境**：本项目落地时，业务 / 自有 SVG 资源的真相与候选以 `client/src/design-assets/resource-skill-map.json`（由 inventory 审计确定性派生）为准；红线资源（品牌 / 渠道 / 头像）经 `canonicalAssets` 引用、跨仓由宿主注入。
> - 新增或替换资产时，当前项目同步更新资源库（重跑 `npm run build:skill-map`），跨仓同步更新宿主仓的正式 registry。

## 1. 资产优先级

1. `lucide-react`：通用 UI 图标，如导航、操作、状态、表单辅助（**默认首选，不被下列规则改变**）。
2. 业务 / 自有 SVG / 品牌语义资产、空状态插画（如 AI Agent、Token、风险、地域、通用无数据）：
   - 在**当前项目页面**生成时，从 `client/src/design-assets/resource-skill-map.json` 按组件槽位选择候选（详见 §5.5）；
   - 在**跨仓 / 可移植**语境，参照 `assets/icon-registry.example.json` 身份样例，由宿主仓 registry 提供或宿主注入。
3. `client/public/icon/**`、`icon/**`、`client/public/assets/**`、`client/public/landing-assets/**` 现有资产（当前项目；已被 §2 inventory 收录的优先经 skill-map / canonical 引用）。
4. 新增资产：仅当以上都无法表达，并需登记来源、用途、路径、状态。

## 2. 目录约定

| 目录 | 用途 |
|---|---|
| `icon/` | 根目录业务图标源文件 |
| `client/public/icon/` | 前端 public 可访问业务图标 |
| `client/public/assets/` | 前端 public 可访问业务图片 / PNG 插画 |
| `client/public/landing-assets/` | 官网 / 落地页视觉资产 |
| `client/src/assets/` | 需要被 bundler import 的源码资产 |
| `assets/icon-registry.example.json` | 资产清单示例，供移交和宿主仓映射 |

## 3. 命名规则

- 新增 SVG 优先 kebab-case：`icon-agent-risk.svg`。
- 延续现有中文名时保持语义清楚，不写“图标1 / 未命名”。
- 不同状态用后缀：`-active`、`-disabled`、`-empty`。
- 不在文件名中包含版本随机串。

## 4. SVG 要求

- 优先 `currentColor`，除非品牌插画需要固定颜色。
- 移除无用 metadata、编辑器注释、隐藏图层。
- 不内联外链图片、脚本、foreignObject。
- 可交互图标使用 `aria-label` 或配套文字；纯装饰加 `aria-hidden`。
- 不把大尺寸复杂插画当作小 icon 使用。

## 5. Icon 使用决策

| 场景 | 推荐 |
|---|---|
| 导航 / 按钮 / 表格行操作 | `lucide-react` |
| 业务指标，如输入 Token / 输出 Token / 风险 / 地域 | 已登记 SVG |
| 空状态 | 默认 PNG：`/assets/admin-resource-management/empty-no-data.png`；业务强相关时使用其他已登记 empty-state 资产或 lucide 大图标 |
| Landing Hero 视觉 | 真实产品截图 / landing-assets，少量自定义 SVG |
| 状态点 | CSS 圆点，不用 SVG |

## 5.5 当前项目页面：resource-skill-map 槽位选图

> 仅适用于「当前项目页面层」。共享组件源码 / 跨仓页面不读本映射（用 props / lucide / 宿主注入）。事实源：`client/src/design-assets/resource-skill-map.json`（由 `npm run build:skill-map` 从 inventory 审计确定性派生，请勿手改）。

选图流程：先判断是否命中下列**组件槽位**；命中则在该槽位的 `candidates` 中按视觉风格 / 尺寸 / 场景挑选；未命中或无合适候选时按 §1 优先级回落（优先 `lucide-react`）。

| 槽位 slot | 用途 | 红线 | usageScope | 允许 lucide 回退 | 引用方式 |
|---|---|---|---|---|---|
| `admin-sidebar` | 侧栏菜单图标 | 否 | current-project-only | 是 | 页面经 webPath / import 传入组件 img/svg 槽位 |
| `card-left-icon` | 卡片左侧多彩渐变图标 | 否 | current-project-only | 否（渐变语义 lucide 难等价） | 页面经 webPath / import 引用 |
| `number-card` | 统计卡片图标 | 否 | current-project-only | 否（22 枚渐变候选已足，扁平 lucide 破坏渐变家族） | 页面经 icon prop 传入渐变候选；内置 4 枚为已固化常用项·非上限 |
| `file-type` | 文件类型图标 | 否 | current-project-only | 是 | 页面经 webPath / import；如抽资源须单独立项 |
| `run-status` | 运行状态图标 | 否 | current-project-only | 否（动画语义为组件私有） | 状态动画随组件 inline；静态文件为设计源 |
| `feature-card` | 企业特性卡片图标 | 否 | current-project-only | 否 | 经页面层注入；勿在组件源码写死 /assets |
| `agent-avatar` | Agent 头像 | **是** | host-injected | 否 | 当前项目经 `canonicalAssets.avatars.*`；跨仓由宿主注入 |
| `channel-icon` | 渠道图标 | **是** | host-injected | 否 | 当前项目经 `canonicalAssets.channels.*`；跨仓由宿主注入 |
| `brand-logo` | 品牌 Logo | **是** | host-injected | 否 | 当前项目经 `canonicalAssets.brands.*`；跨仓由宿主注入 |

规则要点：

- **红线资源（agent-avatar / channel-icon / brand-logo）**：品牌固定色，禁当普通 UI 图标改色、禁进普通 icon 槽位；当前项目页面只经 `canonicalAssets.<group>.<key>` 引用，跨仓一律由宿主注入。
- **usageScope 边界**：`current-project-only` 候选只能用于当前项目页面层，禁止进共享组件源码或跨仓代码；`host-injected` 即上述红线，跨仓注入。
- **lucide 回退**：`allowLucideFallback=否` 的槽位（多彩渐变 `card-left-icon` / 动画 `run-status` / 特性卡 `feature-card` / 统计卡 `number-card`）lucide 难等价，无合适候选时标 `needs-design-confirmation` 交设计补绘，不私自画 inline SVG、不回退扁平 lucide。其中 `number-card` 候选已达 22 枚渐变图标，组件内置的 4 枚 inline 渐变仅为其中已固化常用项（非上限），不要因「只有 4 枚」而退回 lucide。
- **不读页面、不臆造候选**：skill 只读 `resource-skill-map.json`；候选来自 inventory 已审计字段，不手填 slot。

## 6. 新增资产流程

1. 确认 `assets/icon-registry.example.json` 或宿主仓 registry 没有可复用项。
2. 将 SVG / PNG 放入合适目录；页面级 PNG 空状态插画优先放在 `client/public/assets/**`。
3. 在 `assets/icon-registry.example.json` 或宿主仓 registry 中新增记录：`id`、`name`、`path`、`category`、`usage`、`status`。
4. 若宿主仓需要画廊或预览页，可额外生成，不作为周一最低交付前提。
5. 如设计未确认，`status` 标记为 `pending-design-review`。

## 7. 禁止事项

- 不使用 emoji 作为产品 UI 图标。
- 不使用未授权外链图标库 CDN。
- 不在 React 组件内粘贴超长 SVG path，除非确认为组件级图标且已压缩。
- 不复制同一语义的多个近似图标；需要替换时更新登记状态。
- 不用旧路径引用已迁移资产。

## 8. 当前资产状态

- 根目录 `icon/` 已有 27 个 SVG，已登记为初始清单。
- 通用页面级空状态 PNG 已登记为 `empty-no-data`，源文件 `client/public/assets/admin-resource-management/empty-no-data.png`，前端引用 `/assets/admin-resource-management/empty-no-data.png`。
- `client/public/icon/` 存在部分重复 / 英文规范化命名资产，后续可统一来源。
- `client/public/landing-assets/` 数量较多，建议按落地页区块逐步登记，不一次性全量搬运。

## 9. 检查脚本覆盖

`check-design-usage.mjs`（skill 自带，跨仓可移植）会检查：

- 旧品牌色。
- Tailwind 重阴影。
- 可疑大圆角。
- emoji 图标倾向。
- 未登记的业务 SVG 资源引用。

默认只 warning；加 `--strict` 失败退出。

当前项目另有资源库专项校验（落地建设计划 §6.5）：

- `npm run build:skill-map`：从 inventory 确定性重建 `resource-skill-map.json`。
- `npm run check:skill-map`：校验候选 id 在 inventory、status=normal、未被治理移除、slot 合法、磁盘文件存在、usageScope 合法、红线落对槽位、槽位约束不漂移（任一不符即失败退出）。
- `npm run check:component-resources`：组件源码层防新增违规（基线模式）。
