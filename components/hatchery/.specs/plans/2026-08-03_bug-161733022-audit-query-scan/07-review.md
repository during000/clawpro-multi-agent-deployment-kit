# 07. Review — 代码审查（username 精确查询 v4）

## 审查状态

- [x] 用户已授权进入 v4 Review
- [x] AI Review 完成
- [ ] 人工 Review

- 开始时间：2026-08-05 16:32:49
- 结束时间：2026-08-05 16:34:48
- 耗时：1 分 59 秒

## 审查范围与结论

审查以 `origin/master` 到 task worktree 的最终完整差异为准，覆盖生产代码、模型、MySQL schema/迁移、UT、Python IT、API 文档及 SOP 记录。

| 维度 | 结论 | 证据 |
|------|------|------|
| 正确性 | PASS | username 默认等值、仅 `fuzzy=1` 包含匹配；fuzzy 非 1、无 username、user_id 0/非法/溢出和组合筛选均覆盖 |
| 权限与租户 | PASS | 继续使用 `requireAdmin`；Count/列表均从 `model.DB(r.Context())` 创建，保留 identifier 自动过滤 |
| SQL 安全 | PASS | 业务筛选全部使用 GORM 参数绑定；新增 Go diff 无 `Raw` / `Exec` / `Table` 等禁止接口 |
| API 兼容 | 已批准例外 | username 从隐式模糊改为默认精确是有意语义迁移；完整 username 调用形式不变，部分匹配调用方需显式增加 `fuzzy=1`，用户已明确接受并已文档化 |
| 数据库维护 | PASS | GORM tag、SQLite AutoMigrate 测试、`sql/init.sql`、`0804` 增量迁移均包含相同三个联合索引 |
| 迁移安全 | PASS | 迁移仅 ADD INDEX；MySQL 新建、master 升级及重复执行通过，无 DROP INDEX |
| 性能 | PASS（有边界） | user_id/username/resource_id 等值查询命中租户联合索引；fuzzy 与首次租户 Count 的线性边界已明确 |
| 测试与文档 | PASS | UT、真实 MySQL/HTTP、13 项 Python IT、OpenAPI 及新增参数覆盖与 v4 契约一致 |
| 工作区 | PASS | 主工作区为干净最新 master；任务 worktree 只保留本任务未提交变更，无测试产物 |

## 兼容性例外说明

仓库规范原则上禁止改变公开 API 参数语义。本需求明确要求：

1. 已知 API 调用方继续传完整 username，不改变参数名或调用结构；
2. 该默认路径必须从 `LIKE '%username%'` 改为 `username = ?`，避免租户范围扫描；
3. 确需部分匹配的调用方使用新增 `fuzzy=1` 恢复原包含查询；
4. 操作记录前端主流程仍先选用户，再按 `user_id` 查询。

因此本次语义变化属于用户明确批准的需求例外，不是无意回归。剩余风险是未知调用方若依赖部分 username 且不增加 `fuzzy=1`，结果会变化；该风险已在 API 文档、Clarify、Plan 和交付说明中突出记录。

## 发现与处理

| # | 文件 | 问题 | 严重度 | 处理 | 状态 |
|---|------|------|--------|------|------|
| 1 | `.specs/.../01-clarify.md` | v4 结论之后保留的 v3 历史段落未明确声明已被 v4 覆盖，可能被误读为当前契约 | 低 | 增加“冲突时以 v4 为准”的历史说明 | 已修复 |
| 2 | `.specs/.../08-commit.md` | 仍显示上次 v3 已完成提交状态，不能反映 v4 尚未提交的事实 | 低 | 重写为 v4 待授权状态，保留单提交与 force-with-lease 约束 | 已修复 |

未发现本任务引入的、未处理的高或中严重度正确性/安全性问题。公开 API 语义风险按上节作为已批准需求例外保留。

## Review 回归

```text
GO111MODULE=on go test ./controller \
  -run '^(TestHandleAdminAudit|TestAdminAuditHandlers)' -count=1 -race
PASS (2.470s)

GO111MODULE=on go test ./model \
  -run '^(TestLogAudit|TestAuditLog)' -count=1 -race
PASS (1.635s)

GO111MODULE=on go vet ./...
PASS

gofmt -l <本任务 Go 文件>
无输出

Python AST 语法检查
PASS

git diff --check origin/master
PASS
```

- 新增 Go 行禁止接口扫描：PASS。
- 增量迁移 `DROP INDEX` 扫描：PASS，无匹配。
- 明文测试 Token/密码仓库扫描：PASS，无匹配。
- `staticcheck`：本机未安装，未执行。

## CI 门禁说明

`.ci/ci-check-schema.sh` 和 `.ci/ci-check-coverage.sh` 通过 `origin/master...HEAD` 读取已提交内容。当前 v4 尚未提交，直接运行只会看到远端/本地 HEAD 的 v3 内容，不能代表最终 v4，因此不在 Review 阶段伪造通过结论。

等用户授权 Commit、将 v4 整理为最终单提交后，必须运行这两个 HEAD-based 门禁；当前工作树等价验证已由真实 MySQL 新建/迁移/幂等检查、函数覆盖率和真实 API 参数覆盖完成。

## 已知边界

- 全量 race 的 task 全局 DB 竞态已在 UT 单独复现并证明为主线既有问题，不归因于审计改动。
- 首页无筛选 Count 随当前租户记录数线性增长；本次轻量方案接受该首次成本，预发布仍需观察耗时和 examined rows。
- `fuzzy=1` 带前导 `%`，仍可能扫描租户范围；它是显式低频能力，前端主流程应使用 `user_id`。
- TAPD 完整闭环仍需要前端提供「用户候选 → user_id」的 MR/联调证据。
- 未执行 K8s/共享环境和生产等量 P95/P99 验证。

## 审查结论

- [x] 无意外引入的高/中严重度未修复问题
- [x] 权限、租户、SQL 安全和错误处理通过
- [x] 数据库三处维护一致，迁移只增不删且真实 MySQL 验证通过
- [x] 新增 API 参数 `user_id` / `fuzzy` 有 UT、真实 HTTP 和 Python IT 覆盖
- [x] API 语义迁移风险已获用户批准并完整文档化
- [x] 工作区与临时资源清理通过

Review 通过，可以在用户授权后进入 Commit 阶段。
