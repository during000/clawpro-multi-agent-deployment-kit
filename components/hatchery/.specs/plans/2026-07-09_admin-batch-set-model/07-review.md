# 07. Review — 代码审查

---

## 审查方式

- [x] AI 自动 Review：`CodeReview`
- [x] AI 自动复核：`FinalReview`
- [x] AI 关闭复核：`ClosureReview`
- [ ] 人工 Review

## 发现的问题

| # | 文件 | 行 | 问题 | 严重度 | 状态 |
|---|------|---|------|-------|------|
| 1 | `controller/openclaw_model.go` | `978-979` | 批量 primary + fallback 逐个 TAT 下发时，前序模型下发成功、后续模型失败会只回滚 DB，CVM 可能保留新 primary/fallback 配置，导致 DB 与运行时分叉。 | 高 | 已修：DB 回滚后存在旧绑定时调用 `restoreInstanceModelBindingsToCVM` 重放旧 primary/fallback。新增 `TestHandleAdminBatchSetModel_TATPartialFailureRollsBackAndRestoresCVM`。 |
| 2 | `controller/openclaw_model.go` | `1033-1034` | 全新实例无旧 `instance_models` 时，前序 TAT 成功、后续失败会回滚 DB 到无绑定，但 CVM 仍可能保留本次新 provider/defaults。 | 高 | 已修：无旧绑定时调用 `cleanupDesiredModelBindingsFromCVM`，用 `remove_model_provider.sh` 清理本次新 provider。新增 `TestHandleAdminBatchSetModel_FreshInstanceCleanCVMNewProviders`。 |

## 复核结果

```text
ClosureReview: No blocking issues found.
Inspected controller/openclaw_model.go rollback helpers/provider key paths and controller/admin_instances_test.go rollback regression tests.
```

## 审查通过确认

- [x] 无高严重度未修复问题
- [x] 代码风格一致（`gofmt` 已执行；LSP diagnostics OK）
- [x] 安全基线检查通过（未新增密钥、未记录 AK/SK；IT 文档只记录变量名）
- [x] 验证通过：`go vet ./...`、`go test ./controller -run TestHandleAdminBatchSetModel -count=1`、`go test ./... -count=1`
