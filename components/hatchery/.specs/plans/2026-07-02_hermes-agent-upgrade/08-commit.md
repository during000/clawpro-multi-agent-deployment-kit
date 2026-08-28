---
task_id: hermes-agent-upgrade
stage: 08-commit
author: AI
date: 2026-07-20
run_mode: manual
---

# Commit

## 本次新增/修改文件清单

| 文件 | 类型 | 说明 |
|------|------|------|
| `.specs/plans/2026-07-02_hermes-agent-upgrade/00-overview.md` ~ `07-review.md` | 新增 | 反向补建的 SOP 文档（8 个文件） |
| `controller/agent_plugin_preserve_paths_consistency_test.go` | 新增 | preserve_paths JSON 与脚本一致性守卫测试 |
| `test/scripts/openclaw_instance/test_instance_upgrade_hermes.py` | 新增 | Hermes 升级链路专属集成测试（契约级，9 用例） |

不涉及对分支已有生产代码（`controller/openclaw_upgrade.go`、`model/agent_type.go` 等）的任何修改，仅新增测试与文档，风险为零。

## 验证结果

- `go build ./...`：通过
- `go vet ./controller/...`：通过
- `go test ./controller/... -run 'Hermes|Upgrade|Restore|Backup|PreservePaths' -count=1`：全部 PASS
- 新增守卫测试负向验证：临时引入配置漂移后测试正确 FAIL，还原后重新 PASS，`git diff` 确认脚本文件无残留改动
- `test_instance_upgrade_hermes.py`：`py_compile` 语法检查通过；Python 3.11 + 补齐依赖后 import 检查通过（9 个测试函数全部正确发现）；未接入真实服务端运行，需要在有 Hermes 镜像的环境中做最终确认

## Commit Message

```
test: add hermes upgrade preserve_paths guard test + IT contract tests; docs: backfill SOP artifacts
```

## 提交状态

已通过 gongfeng MCP `batch_modify_files` 提交至远端 `feature/hermes_agent_upgrade` 分支。提交后需在本地执行：

```bash
git fetch origin feature/hermes_agent_upgrade
git checkout -- <本次改动文件>  # 若本地有同名旧文件残留
git merge --ff-only origin/feature/hermes_agent_upgrade
```

以同步本地 git HEAD。

## 后续建议（提 MR 前）

1. 在具备 Hermes 镜像的测试环境实跑一次 `test_instance_upgrade_hermes.py`。
2. 补跑一次 `BASE_BRANCH=origin/master bash .ci/ci-check-coverage.sh` 增量覆盖率检查。
