# 08. Commit — 提交记录

---

## 提交前检查

> **执行顺序（强制）**：
> ① 写完本文件 → ② 更新 `00-overview.md` 状态为已完成 → ③ 确认所有 `0N-*.md` 已就绪 → ④ `jj squash` 形成单一提交 → ⑤ 设置 bookmark → ⑥ `jj git push`

- [x] 所有 `0N-*.md` 产物已完成
- [x] `00-overview.md` Progress 全部勾选
- [x] `gofmt` + `go vet` 通过（功能新增文件/区块已格式化，`go vet ./...` 通过；基线中的 `controller/audit.go`、`i18n/keys.go` 存在整文件既有 gofmt 对齐差异）
- [x] 单元测试通过

## Commit Message

```text
--story=135676012 【clawpro】管控端的agent列表新增管理员可创建并分配agent功能
```

模板的 conventional commit 占位格式由本仓库 TAPD 提交规范覆盖；本需求使用强制的 story 短 ID 前缀。

提交 bookmark：`feature/clawpro-agent-admin-create-assign`

基线：`Release/2026_07_15@origin`（`ac61396e7565`）。功能提交已通过 jj rebase、单提交收敛并推送；全部 spec 模板修正均折叠到同一功能提交，不保留额外文档提交。
