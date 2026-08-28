# 08. Commit — 提交记录

---

## 提交前检查

> **执行顺序（强制）**：
> ① 写完本文件 → ② 更新 `00-overview.md` 状态为已完成 → ③ 确认所有 `0N-*.md` 已就绪 → ④ `git add` → ⑤ `git commit` → ⑥ `git push`

- [x] 所有 `0N-*.md` 产物已完成（00-08 共 9 个文件）
- [x] `00-overview.md` Progress 全部勾选
- [x] 修改代码格式与 `git diff --check` 通过
- [x] `go vet ./model ./controller ./task` 通过
- [x] 8/8 Doctor 目标用例在 race 下通过
- [x] Doctor 相关 `controller`、`task` 回归在 race 下通过
- [x] 新增生产代码增量覆盖率 100%（20/20）
- [x] MySQL 增量迁移、全量 schema、OpenAPI 生成通过

全仓非阻断基线问题已分别记录在 UT/Review：

- `go vet ./...`：`skillhubclient/client_test.go:278` 既有告警。
- `go test ./... -race`：AgentChecker 既有测试存在后台 goroutine 与全局测试 DB cleanup 的 data race。

## 提交范围

- DoctorSession 新增独立激活时间及 MySQL schema/migration。
- 激活流程原子写入 active 状态与激活时间。
- STS 周期刷新不再改变 `updated_at`。
- NoFiles 超时按激活时间判断，存量记录回退创建时间。
- 补充成功、兼容及数据库失败路径测试。
- 更新龙虾医生自动结束策略文档和完整 SOP 产物。

## Commit Message

```
--bug=161160730 【clawpro bug 单】【高优】进行诊断的龙虾在12小时内不会触发兜底策略进行销毁
```

## 推送目标

- 分支：`feature/bug-161160730-doctor-12h-fallback-0730`
- 远端：`origin`
- 命令：`git push -u origin feature/bug-161160730-doctor-12h-fallback-0730`
