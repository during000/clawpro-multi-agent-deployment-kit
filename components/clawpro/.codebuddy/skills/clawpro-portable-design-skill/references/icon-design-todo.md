# Icon Design Confirmation TODO

根据 `needs-design-confirmation` 标记自动生成。

**最后更新时间**：2026-06-26 06:00 UTC  
**生成方式**：`scripts/check-design-usage.mjs` + `scripts/generate-icon-todo.mjs`

---

## Summary

- **Total items**: 12
- **Status distribution**:
  - 未启动: 8
  - 进行中: 3
  - 已完成: 1
- **Last reviewed**: 2026-06-26
- **Priority distribution**:
  - P0 (阻塞发布): 2
  - P1 (高优先): 5
  - P2 (中优先): 4
  - P3 (低优先): 1

---

## Items

### 1. NumberCard - 梯度图标待设计

- **位置**: `portable/react/number-card.tsx:45`
- **需要**: 5 种梯度颜色的 KPI 卡片图标
  - InputTokensIcon（蓝→紫梯度）
  - OutputTokensIcon（绿→蓝梯度）
  - CostIcon（红→橙梯度）
  - LatencyIcon（黄→红梯度）
  - ThroughputIcon（蓝→绿梯度）
- **当前**: 临时用单色 lucide 图标
- **优先级**: P0
- **负责人**: @design-system
- **状态**: 未启动
- **描述**: NumberCard 槽位设计规范要求梯度多彩，但目前没有合适的梯度图标设计。需要与设计沟通确认5个图标的配色方案。

---

### 2. StatusTag - 文案变体图标

- **位置**: `portable/react/status-tag.tsx:38`
- **需要**: 为以下文案补充对应图标
  - "进行中" → 时钟/加载图标
  - "已完成" → 勾选/完成图标
  - "失败" → 错误/叉号图标
  - "等待中" → 暂停/等待图标
- **当前**: 空
- **优先级**: P1
- **负责人**: @design-system
- **状态**: 未启动
- **描述**: StatusTag 的文案需要配套图标来提升可读性，但目前大部分文案还没有对应的图标。

---

### 3. CardLeftIcon - 业务对象图标集

- **位置**: `portable/react/card-surface.tsx:82`
- **需要**: 为以下业务对象类型补充图标
  - Agent（智能体）→ 机器人/AI图标
  - Model（模型）→ 神经网络/芯片图标
  - Skill（技能）→ 工具/魔杖图标
  - 知识库 → 文档/书籍图标
  - 提示词 → 星号/灵感图标
- **当前**: 占位符（空或 placeholder）
- **优先级**: P1
- **负责人**: @design-system
- **状态**: 进行中（已交付设计稿，等待 SVG 转换）
- **描述**: Card 左侧的业务对象图标需要统一的、可识别的设计。已有 Figma 设计稿。

---

### 4. RunStatus - 执行状态图标

- **位置**: `portable/react/run-status.tsx:28`
- **需要**: 6 个执行状态对应的图标
  - "running" → 运行/旋转图标
  - "success" → 成功/勾选
  - "failed" → 失败/错误
  - "cancelled" → 取消/停止
  - "queued" → 队列/排队
  - "timeout" → 超时/沙漏
- **当前**: 使用 lucide 的通用图标（不够具体）
- **优先级**: P0
- **负责人**: @design-system
- **状态**: 进行中（正在与设计讨论细节）
- **描述**: 执行流程的状态需要更清晰的视觉标识。当前 lucide 图标不够业务特化。

---

### 5. FeatureCard - 功能卡片装饰图标

- **位置**: `portable/react/feature-card.tsx:55`
- **需要**: 3 种功能卡片类型的装饰图标
  - "intelligence"（智能）→ 大脑/闪电图标
  - "reliability"（可靠）→ 盾牌/钻石图标
  - "efficiency"（高效）→ 闪电/火箭图标
- **当前**: 临时用空白区域
- **优先级**: P2
- **负责人**: @design-system
- **状态**: 未启动
- **描述**: Landing 页的功能卡片需要对应的装饰性图标来提升视觉层次。

---

### 6. AdminSidebarIcon - 导航菜单图标集

- **位置**: `portable/react/admin-sidebar.tsx:120`
- **需要**: Admin 页面完整导航菜单的图标集（>20 个）
  - Dashboard（仪表板）
  - Models（模型）
  - Agents（智能体）
  - Skills（技能）
  - Users（用户）
  - Settings（设置）
  - ...等
- **当前**: 使用 lucide-react 的通用图标
- **优先级**: P1
- **负责人**: @design-system
- **状态**: 进行中（已有初稿，等待优化）
- **描述**: Admin 导航需要一套统一的、高识别度的菜单图标。

---

### 7. BatchActionIcon - 批量操作工具栏图标

- **位置**: `portable/react/batch-action-bar.tsx:45`
- **需要**: 批量操作相关的图标
  - "delete"（删除）
  - "export"（导出）
  - "import"（导入）
  - "move"（移动）
  - "copy"（复制）
  - "assign"（指派）
- **当前**: 混用 lucide 和自定义 SVG
- **优先级**: P2
- **负责人**: @design-system
- **状态**: 未启动
- **描述**: 工具栏的动作图标需要统一的设计语言。

---

### 8. ToggleItemIcon - 切换项目图标

- **位置**: `portable/react/toggle-group.tsx:95`
- **需要**: 为常用的切换选项补充图标
  - "list"（列表视图）
  - "grid"（网格视图）
  - "compact"（紧凑视图）
  - "timeline"（时间线）
  - "kanban"（看板）
- **当前**: 纯文字或 lucide 通用图标
- **优先级**: P2
- **负责人**: @design-system
- **状态**: 未启动
- **描述**: 视图切换需要图标来快速识别。

---

### 9. ChartLegendIcon - 图表图例图标

- **位置**: `portable/react/chart-wrapper.tsx:178`
- **需要**: 图表图例中的数据系列图标
  - success_rate（成功率）→ 上升趋势
  - error_rate（错误率）→ 下降趋势
  - latency_p95（延迟）→ 时钟
  - throughput（吞吐）→ 流量
- **当前**: 仅有色块，无图标
- **优先级**: P2
- **负责人**: @design-system / @data-viz
- **状态**: 未启动
- **描述**: 图表图例可以通过图标提升数据可读性。

---

### 10. FormFieldHint - 表单字段提示图标

- **位置**: `portable/react/input.tsx:112`
- **需要**: 表单字段的 hint/tooltip 图标
  - "info"（信息）
  - "warning"（警告）
  - "error"（错误）
  - "success"（成功）
- **当前**: 使用 lucide 的通用 info / alert 图标
- **优先级**: P3
- **负责人**: @design-system
- **状态**: 未启动
- **描述**: 表单反馈可以用更明确的图标。

---

### 11. EmptyState - 空态装饰图标

- **位置**: `portable/react/empty-state.tsx:42`
- **需要**: 不同场景的空态装饰图标
  - "no-data"（无数据）
  - "no-permission"（无权限）
  - "not-found"（未找到）
  - "loading"（加载中）
- **当前**: 文字或简单占位图
- **优先级**: P2
- **负责人**: @design-system
- **状态**: 进行中（已有 Figma 设计稿）
- **描述**: 空态需要友好的视觉表达。

---

### 12. DialogIcon - 对话框头部图标

- **位置**: `portable/react/dialog.tsx:28`
- **需要**: Dialog 的不同类型头部图标
  - "confirm"（确认）
  - "warning"（警告）
  - "info"（信息）
  - "error"（错误）
- **当前**: 可选，大部分使用标题文字
- **优先级**: P3
- **负责人**: @design-system
- **状态**: 未启动
- **描述**: 对话框可选的视觉强化。

---

## Statistics

### 按槽位分布

| 槽位 | 数量 | 优先级 |
|-----|-----|-------|
| NumberCard | 1 | P0 |
| StatusTag | 1 | P1 |
| CardLeftIcon | 1 | P1 |
| RunStatus | 1 | P0 |
| FeatureCard | 1 | P2 |
| AdminSidebarIcon | 1 | P1 |
| BatchActionIcon | 1 | P2 |
| ToggleItemIcon | 1 | P2 |
| ChartLegendIcon | 1 | P2 |
| FormFieldHint | 1 | P3 |
| EmptyState | 1 | P2 |
| DialogIcon | 1 | P3 |
| **TOTAL** | **12** | - |

### 按优先级分布

| 优先级 | 数量 | 状态 |
|-------|-----|------|
| P0（阻塞发布） | 2 | 1 未启动，1 进行中 |
| P1（高优先） | 5 | 2 未启动，2 进行中，1 其他 |
| P2（中优先） | 4 | 3 未启动，1 进行中 |
| P3（低优先） | 1 | 1 未启动 |

### 按状态分布

| 状态 | 数量 | 项目 |
|-----|-----|------|
| 未启动 | 8 | 1,2,5,7,8,9,10,12 |
| 进行中 | 3 | 3,4,6,11 |
| 已完成 | 1 | - |

---

## 维护说明

### 自动更新

此文件可通过以下命令自动生成/更新：

```bash
node scripts/generate-icon-todo.mjs --output references/icon-design-todo.md
```

### 手工维护

如需手工维护项目，请：

1. 在代码中添加 `needs-design-confirmation: [描述]` 标记
2. 运行脚本提取
3. 编辑此文件补充详情（优先级、负责人、截止日期等）

### 标记格式

```typescript
// 在 .tsx / .tsx 文件中：
/* needs-design-confirmation: 描述此处需要什么图标/设计 */
const icon = undefined;

// 或作为注释行：
// needs-design-confirmation: 待设计的图标类型和要求
```

### 更新流程

1. **发现问题** → 在代码中添加标记
2. **汇总清单** → 运行脚本生成此文件
3. **设计评审** → 更新状态和优先级
4. **设计交付** → 状态改为"已完成"
5. **代码审查** → 移除标记

---

**最后生成**：2026-06-26 06:00 UTC  
**预计下次更新**：2026-06-27（或手工触发时）
