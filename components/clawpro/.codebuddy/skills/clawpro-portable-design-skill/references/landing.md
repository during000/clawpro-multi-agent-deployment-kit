# Landing Reference

> 官网 / 落地页规范。适用：`client/src/pages/landing/**`、官网首页、营销介绍区块、未登录宣传页。

## 1. 定位

落地页是 Brand + Product 的混合表面：

- Brand：建立 ClawPro 的可信感、效率感、企业级安全感。
- Product：清楚说明用户能完成什么任务，避免空泛营销词。

## 2. 加载顺序

1. `foundation.md`
2. `components.md`
3. 本文件
4. 涉及资产时读取 `assets-icons.md`
5. Landing 当前确认使用独立导航，不复用 Tenant TopNav；如后续改为复用，再读取 `tenant.md` 回写差异。

## 3. 视觉原则

- 保持 ClawPro 品牌蓝 `#1447E6` 和黑→蓝 CTA 渐变。
- 不使用泛 SaaS 套路：巨大无意义指标、满屏玻璃拟态、渐变文字、重复三卡片网格。
- 图像 / 插画必须服务于具体能力：Agent、模型、技能、通道、企业治理，而不是抽象科技背景。
- Hero 文案具体，避免“赋能、重塑、下一代、无缝、世界级”等空泛词。

## 4. Hero

推荐结构：

1. 一句具体主张：说明 ClawPro 帮谁把什么工作做得更稳。
2. 两行以内副文案：交代 Agent 管理、模型额度、技能分发、企业治理中的核心价值。
3. 主 CTA + 次 CTA。
4. 可信证据：真实功能截图 / 模块入口 / 架构示意，不用装饰性大数字。

## 5. 区块节奏

| 区块 | 作用 |
|---|---|
| Hero | 建立主张和入口 |
| Capability strip | 3-5 个具体能力，不做泛功能卡 |
| Workflow | 展示从创建 Agent 到配置技能 / 模型 / 通道的流程 |
| Governance | 展示权限、配额、安全、审计能力 |
| Scenario | 以使用场景分组，而不是堆功能名 |
| CTA | 明确下一步 |

## 6. 组件用法

- CTA 按钮仍使用项目 Button 变体，主按钮保持品牌渐变 token。
- Landing 使用独立导航，不复用 Tenant TopNav；若后续设计改为轻改版，需新增导航差异 spec。
- 功能图标优先 `lucide-react`，若需要品牌图形用 `assets/icon-registry.example.json` 或宿主仓 registry 已登记资产。
- 周一前不锁定 Landing 区块卡片体系，只保留内容结构与治理边界；内容卡片后续补 anatomy。

## 7. 图片与资产

- 优先使用项目真实截图、已有 SVG、已登记 icon。
- 不引用未经确认授权的外链图。
- 新增 raster 图片需压缩，并提供 WebP / PNG fallback 策略。
- SVG 文件命名使用 kebab-case 或现有中文命名体系，新增后登记。

## 8. 可访问性

- Hero 主标题不超过 65-75ch 的阅读宽度。
- 文本对比度正文 ≥ 4.5:1，大字号 ≥ 3:1。
- 图片必须有有意义的 `alt`，装饰图 `alt=""`。
- 动画遵守 `prefers-reduced-motion`。

## 9. 禁止事项

- 禁止渐变文字。
- 禁止 emoji 作为核心视觉。
- 禁止大面积 glassmorphism 默认风格。
- 禁止“hero + 三个指标 + 三张功能卡”的模板化首屏。
- 禁止无真实信息的“企业级 / 智能化 / 全链路”堆词。
