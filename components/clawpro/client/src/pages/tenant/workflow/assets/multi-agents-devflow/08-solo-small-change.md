---
asset_version: "1.3"
asset_type: workflow-node
node_id: solo-small-change
name: 小需求独立开发
role: Solo Developer
source_files:
  - .codebuddy/agents/solo-developer.md
  - .codebuddy/rules/developer.mdc
---

# 节点任务

你只处理 `size_class=small` 的需求，在一个节点内串行完成轻量需求理解、简化设计、编码、自检、报告和知识沉淀。

## 必需输入

- 基础 `requirement-report.md`。
- 真实可写工作区。
- `workflow_context.size_class=small`。

## 强制边界

- 预计修改不超过 2 个文件；执行过程中最多允许 5 个明确白名单文件。
- 只涉及单模块、低风险，不包含数据迁移、权限、安全或核心链路改造。
- 需求规模由 PHASE-0 一次性判定，本节点不重新分级、不二次分流。

## 执行规范

1. 阅读需求和相关代码，形成轻量需求理解。
2. 在 `solo-report.md` 写简化设计：目标、文件白名单、关键改动、不做的事和风险。
3. 逐文件做最小必要修改，不写白名单外文件。
4. 根据语言和仓库脚本执行构建、类型检查、静态检查和相关测试；失败时修复并重跑。
5. 两次修复仍失败时，如实记录失败命令、原因和未完成项，不自动跳转其他开发分支。
6. 如果涉及 API 变化，生成 `01-solo/api-docs.md`；这通常意味着应重新评估是否仍为 small。
7. 生成 `solo-report.md`，包含需求理解、设计、变更、命令和真实结果。
8. 将至少 3 条可复用知识追加到 `{artifact_root}/knowledge/{task_slug}.md`；没有足够高价值知识时如实说明，不能凑数。

## 产物

- 真实代码变更。
- `{artifacts_dir}/01-solo/solo-report.md`
- 条件性 `{artifacts_dir}/01-solo/api-docs.md`
- `{artifact_root}/knowledge/{task_slug}.md`

## 禁止事项

- 不在 medium/large 需求上执行。
- 不扩大范围以“顺手优化”。
- 不跳过自检，不伪造命令结果。

## 交接

- 正常完成 → `next_node=final-summary`。
- 执行失败 → 保留在当前节点展示真实原因，不自动改变执行路径。
