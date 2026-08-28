# 产品动态文案示例

> 从 AutoSync 录入中心 156 条已发布产品动态中提炼的典型示例，用于 skill 生成文案时的 few-shot 参考。

---

## 一、功能上线（Feature）

### 示例 1：全新能力上线

```yaml
id: "feat-local-agent-20260701"
title: "ClawPro 支持托管本地 Agent（一期）"
type: "功能上线"
date: "2026-07-01"
endpoint: "管控端"
description: "在管控端 Agent 管理中新增「本地 Agent」类型，支持将用户本地机器注册为 Agent 实例，实现本地与云端统一管理。您可在管控端创建本地 Agent 并获取注册 Token，在本地执行注册脚本即可完成接入。"
display_components:
  banner:
    enabled: false
  floating_window:
    enabled: false
needs_guide: true
guide:
  doc_type: "operation_guide"
  feature_name: "托管本地 Agent"
  feature_url: "https://clawpro.woa.com/admin/agents?type=local"
  endpoint: "管控端"
```

### 示例 2：工具/能力升级

```yaml
id: "feat-agent-tool-library-20260628"
title: "Agent 工具库实现管理升级"
type: "功能上线"
date: "2026-06-28"
endpoint: "管控端"
description: "管控端新增「工具库」模块，支持按工具类型分类浏览、启用/停用工具，以及按 Agent 粒度配置工具的可见范围。管理员可以灵活控制每个 Agent 实例可使用的工具集合。"
display_components:
  banner:
    enabled: false
  floating_window:
    enabled: false
needs_guide: false
```

### 示例 3：配置能力增强

```yaml
id: "feat-model-advanced-params-20260625"
title: "模型配置新增高级参数设置"
type: "功能上线"
date: "2026-06-25"
endpoint: "管控端"
description: "管控端模型配置页面新增 temperature、top_p、max_tokens 等高级参数设置入口，支持按模型粒度覆盖全局默认值。您还可以为不同 Agent 分配不同的模型参数组合。"
display_components:
  banner:
    enabled: false
  floating_window:
    enabled: false
version: "v2.6.5"
needs_guide: false
```

### 示例 4：计费/商业模式

```yaml
id: "feat-billing-pay-as-you-go-20260620"
title: "新增按量计费模式与配额管理"
type: "功能上线"
date: "2026-06-20"
endpoint: "管控端"
description: "ClawPro 新增按量计费模式，支持按 Token 消耗量实时计费。管控端新增配额仪表盘，可查看全局和用户级的 Token 消耗趋势、费用分布，并设置消耗上限告警。"
display_components:
  banner:
    enabled: true
    duration_days: 14
  floating_window:
    enabled: true
    duration_days: 14
needs_guide: true
guide:
  doc_type: "config_integration"
  feature_name: "按量计费配额管理"
  feature_url: "https://clawpro.woa.com/admin/billing"
  endpoint: "管控端"
```

---

## 二、体验优化（Improvement）

### 示例 5：交互体验改善

```yaml
id: "impr-agent-card-conversation-btn-20260622"
title: "用户端 Agent 卡片新增快捷对话按钮"
type: "体验优化"
date: "2026-06-22"
endpoint: "用户端"
description: "在用户端 Agent 列表卡片中新增「开始对话」快捷按钮，用户无需进入详情页即可直接发起对话。该按钮在移动端和桌面端均已适配。"
display_components:
  banner:
    enabled: false
  floating_window:
    enabled: false
needs_guide: false
```

### 示例 6：视觉效果增强

```yaml
id: "impr-model-card-redesign-20260618"
title: "模型选择页面视觉交互焕新"
type: "体验优化"
date: "2026-06-18"
endpoint: "管控端"
description: "管控端模型选择页面进行了视觉升级，模型卡片增加模型类型标签和能力标签展示，模型筛选器支持多条件组合筛选，查找目标模型更加便捷。"
display_components:
  banner:
    enabled: false
  floating_window:
    enabled: false
needs_guide: false
```

### 示例 7：性能/稳定性优化

```yaml
id: "impr-session-loading-speed-20260615"
title: "会话列表加载速度全面优化"
type: "体验优化"
date: "2026-06-15"
endpoint: "用户端"
description: "优化了会话列表的分页加载逻辑和缓存策略，处于大型项目中的会话列表首屏加载时间缩短约 60%，滚动加载更加流畅。"
display_components:
  banner:
    enabled: false
  floating_window:
    enabled: false
needs_guide: false
```

---

## 三、典型句式提炼

### 功能上线常用句式

| 结构 | 示例 |
|---|---|
| `{产品/模块} 新增/上线 {能力名称}` | ClawPro 支持托管本地 Agent（一期） |
| `{功能模块} 实现 {能力描述}` | Agent 工具库实现管理升级 |
| `{产品} 新增 {模式/机制}` | 新增按量计费模式与配额管理 |

### 体验优化常用句式

| 结构 | 示例 |
|---|---|
| `{页面} 新增/优化 {交互细节}` | 用户端 Agent 卡片新增快捷对话按钮 |
| `{模块} 视觉/交互焕新` | 模型选择页面视觉交互焕新 |
| `{功能} 体验全面升级` | 会话列表加载速度全面优化 |

---

## 四、描述的黄金结构

> **做了什么 + 在哪里 + 对用户有什么价值/影响**

```
✅ 管控端新增「工具库」模块，支持按工具类型分类浏览、启用/停用工具。
   → [做了什么] [在哪里] [用户能做什么]

✅ 优化了会话列表的分页加载逻辑和缓存策略，首屏加载时间缩短约 60%。
   → [做了什么] [用户获得什么好处]

❌ 新增接口 /api/v1/tools/list，支持 GET 请求。
   → [太偏技术实现，不是产品动态文案]
```

---

## 五、常见错误与纠正

| 错误 | 纠正 |
|---|---|
| `新增支持xxx功能` | `xxx 新增/上线` 或 `xxx 支持yyy` |
| `修复了xxx bug` | 以用户视角改写为「优化了xxx体验」或归入体验优化 |
| `现在您可以xxx了` | 去掉口语化，改为「支持xxx」 |
| `更新xxx依赖到v2.3` | 去掉技术实现细节 |
| 描述未以句号结尾 | 必须加「。」 |
