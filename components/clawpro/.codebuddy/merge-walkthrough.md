# 合并后走查清单（main 功能 vs 分支设计系统样式）

> 逐模块精确核查（state/function 差集 + 业务关键词）结论。
> 核查日期见 git 提交。tsc 错误 18 个均为合并前既有问题（详见末尾）。

## 一、取 main 版（分支版缺失 main 功能）— 功能 100% 保留，样式回退待改造

这些文件分支版基于较早基线，缺 main 后续新增的核心功能，已取 main 保功能。
**样式回退**：设计系统组件数 = 待设计同学按分支设计语言（Typography/Surface/StatusTag 等）重新改造的工作量。

| 文件 | 缺失的 main 功能 | 样式回退(设计系统组件) |
|---|---|---|
| SecurityGroupManagement | 拖拽排序(DndContext/useSortable)+平台必需规则 | 182 |
| OpenClawMonitor | 计费模式(getClawBillingMode)+分组迁移/移交 | 85 |
| BasicInfo | 时间维度 Token 配额(TokenTimeDimensionEditor) | 45 |
| MemberManagement | 时间维度配额(TokenPeriodEditor)+Agent实例处理v2.0 | 13 |
| OpsObservation | 环比趋势(TrendBadge/环比指标) | 13 |
| SessionManagement | 高级筛选(token/延迟/trace范围+筛选面板) | 10 |
| PlatformPolicy | 分组配额策略树编辑(getCheckState/renderGroupRule) | 5 |
| TokensMonitor | 配额监控(时间维度) | (取main) |

## 二、保留分支样式（功能完整 / 等价重构）— 无需走查

| 文件/模块 | 说明 |
|---|---|
| InstanceTable | 分组列筛选功能在(分支基线较新) + 修了 DialogType 比较 |
| ModelConfig | ScopeSelect 组件等价替代 main 内联树形选择器 |
| ImageManagement 系列 | StatusBadge→ImageStatusBadge 等组件化重构，功能在 |
| SkillLibrary 模块 | 设计系统组件重构(FileBrowser/ScopeSelect 封装)，功能等价 |
| MemoryManagement 模块 | 级联多选/分组策略等价重构 |
| SessionDetail | 深度融合：保留 main 的 trace 数据架构 + 套用分支设计系统 UI |
| tenant 模块全部 | 无功能缺失 |
| 全局组件全部 | 无功能缺失 |

## 三、待定决策

- **MCPAddDialog**：凭据托管两套设计不兼容——
  - main：多凭据字段(credentialFields)+hosted/placeholder 模式
  - 分支：单 Token+IP 白名单+结构化 CONFIG_REFERENCE+设计系统样式
  - 待产品确认哪套为目标设计。

## 四、需额外确认

- **导航入口**：`adminNav.ts` 为分支新增的导航配置(main 无)，TenantLayout 取 main。需确认所有 main 页面(含取 main 的页面)在导航中均有入口。

## 五、合并前既有 tsc 错误（18 个，非本次合并引入，建议单独治理）

- Set 迭代 TS2802(tsconfig downlevelIteration)：PushUpgradeDialog/PushUpgradePopover/MCPAddDialog/SkillInitialPackageTab/SkillSquare
- SyntaxHighlighter JSX 类型：PublicSkillLibraryTab/SkillSquare
- 其他：App.tsx 路由、CvmSelectComponent(lodash-lite)、ComponentPreview(JSX namespace)、UploadFileCard、PublicSkillTab
