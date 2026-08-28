# 08. Commit — 提交记录（username 精确查询 v4）

## 当前状态

- [x] 用户授权进入 v4 Commit 阶段（2026-08-05 16:36:55）
- [x] 再次拉取并核对最新 `origin/master`
- [x] 将最终 v4 变更整理为相对 `origin/master` 的一个提交
- [x] 重跑提交后验证与 HEAD-based CI schema/coverage 门禁
- [x] 用户授权更新远端任务分支（2026-08-05 16:44:57），使用显式 `--force-with-lease`

本地 v4 Commit 阶段及远端更新均已获授权。更新前远端任务分支指向上次 v3 单提交 `ce2887aa756b31307e6f59bc3cd3376df2299345`，推送使用该哈希作为显式 lease 保护。

## 用户指定 Commit Message

必须逐字使用，不添加前后缀：

```text
--bug=161733022 【clawpro bug 单】/admin/audit 接口全表扫描
```

## 提交约束

- 最终相对 `origin/master` 只能有一个提交。
- 迁移文件保持 `sql/0804-add-audit-query-indexes.sql`。
- 最终迁移只幂等新增三个租户联合索引，不包含 `DROP INDEX`。
- 远端同名分支已有 v3 提交，v4 单提交需要重写后以旧远端哈希为 lease 执行 `--force-with-lease`；禁止普通 `--force`。
- 不创建或修改现有 MR `!1070`，除非用户另行授权。
- 不修改 TAPD 状态、字段或评论，除非用户另行授权。

## Commit 前必须重新确认

| 检查 | 要求 |
|------|------|
| 主工作区 | 干净 `master`，HEAD 等于最新 `origin/master` |
| 任务分支基线 | 最新 `origin/master` 是最终 task HEAD 的父提交 |
| 提交数量 | `origin/master..HEAD` 恰好 1 个提交 |
| 提交标题 | 与用户指定文本逐字一致 |
| schema 门禁 | 基线 init + 0804 migration 与当前 init 一致 |
| 增量覆盖 | 最终提交中的新增 Go 行满足门禁，新增 API 参数 `user_id` / `fuzzy` 均有 IT 覆盖 |
| 定向回归 | controller/model race、vet、release build、diff check 通过 |
| 远端更新 | 本地 HEAD 与远端任务分支哈希一致 |

## 本地提交后验证

| 检查 | 结果 |
|------|------|
| 最新 `origin/master` | `9f2b2120c713301738866398e17b2e0b3562c684`，主工作区已同步且干净 |
| 相对 `origin/master` 提交数 | 1 |
| 提交标题逐字核对 | PASS |
| HEAD-based schema 门禁 | PASS；master init + 0804 migration = 当前 init |
| HEAD-based 覆盖率门禁 | PASS；全量 71.9%，增量 98.6%（68/69，阈值 60%） |
| controller 审计定向 race | PASS，2.452s |
| model 审计索引定向 race | PASS，1.534s |
| `go vet ./...` | PASS |
| release build | PASS |
| OpenAPI 9 参数契约 | PASS |
| Python AST | PASS |
| `git diff --check origin/master..HEAD` | PASS |
| `staticcheck` | 本机未安装，未执行 |

覆盖率文件、HTML 报告、OpenAPI 临时文件和 schema 检查容器均已清理。最终提交哈希在完成最后一次文档状态 amend 后由交付信息报告，避免在提交内容中记录会因自引用而变化的哈希。

## 远端状态

- 用户已单独授权 push；使用 `ce2887aa...` 作为显式 lease，以 `--force-with-lease` 替换远端旧 v3 单提交，未使用普通 `--force`。
- 最终仍保持相对 `origin/master` 只有一个提交，提交信息逐字不变。
- 未创建或修改现有 MR `!1070`，未修改 TAPD。

## 历史远端说明

- v3 已按相同提交信息推送为单提交，迁移前缀已从 0803 修正为 0804。
- v3 后续已移除未上线任务版的 username 索引清理逻辑，远端当前迁移只新增 user_id/resource_id 联合索引。
- v4 将在该基础上加入 username 默认精确、`fuzzy=1` 显式模糊及 `(identifier,username)` 联合索引，并重新整理为一个提交。
