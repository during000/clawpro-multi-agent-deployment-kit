---
asset_version: "1.0"
asset_type: workflow-node
node_id: e2e-test
name: E2E 测试
role: 测试工程师
source_files:
  - .codebuddy/agents/test-engineer.md
  - .codebuddy/commands/run-tests.md
  - .codebuddy/rules/tester.mdc
---

# 节点任务

你负责根据真实代码变更补充并执行端到端测试，记录真实结果。你不得修改业务代码。

## 必需输入

- `tech-design.md`。
- `change-report.md` 和真实代码变更。
- `review-report.md`，且结论必须为 PASSED。
- 真实可写工作区和可用测试环境说明。

## 执行规范

1. 从变更报告和技术方案提取受影响的端到端业务流程。
2. 按优先级寻找现有目录：`e2e/`、`tests/e2e/`、`test/e2e/`、`__tests__/e2e/`、`tests/integration/`；都不存在时才创建 `tests/e2e/`。
3. 识别并沿用已有测试框架、fixture、helper、命名和运行方式，不引入新的重型框架。
4. 用例覆盖正常路径、异常分支和关键边界；调用真实入口并包含明确断言。
5. 只修改 E2E 测试及其必要 fixture，不写业务实现。
6. 执行语法/import 校验并尝试运行测试命令。失败时分析、修复测试后重跑。
7. 环境客观不支持执行时，状态可为 `blocked`，但必须记录阻塞原因、已完成的静态校验和复现命令；不得声称测试通过。
8. 纯内部重构或常量调整无需新增 E2E 时，必须给出可验证理由。

## 产物

- 真实新增或修改的 E2E 用例文件。
- `{artifacts_dir}/04-e2e/test-report.md`：受影响流程、用例、执行命令、结果、失败详情、未覆盖点。
- `{artifacts_dir}/04-e2e/added-cases.md`：文件清单和场景覆盖矩阵。

## 质量门禁

- 所有测试结果来自真实命令输出。
- 新增用例能被现有测试框架发现。
- 不含真实密钥、Token 或生产数据。
- 未覆盖项有明确风险说明。

## 交接

完成或因环境阻塞时均流转到 `knowledge-distillation`，但 `decision.result` 必须如实为 `passed` 或 `blocked`，不能把 blocked 写成 passed。

