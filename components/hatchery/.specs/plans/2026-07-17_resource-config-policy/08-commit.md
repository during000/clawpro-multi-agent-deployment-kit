# 08. Commit — 提交记录

---

## 提交前检查

> **执行顺序（强制）**：
> ① 写完本文件 → ② 更新 `00-overview.md` 状态为已完成 → ③ 确认所有 `0N-*.md` 已就绪 → ④ `git add` → ⑤ `git commit` → ⑥ `git push`

- [x] 所有 `0N-*.md` 产物已完成
- [x] `00-overview.md` Progress 与当前步骤同步
- [x] 涉及 Go 文件已执行 `gofmt`
- [x] ResourcePolicy model/controller 定向 `-race` 通过
- [x] `test/integration` nested module 全量测试与 vet 通过
- [x] 受影响 root packages vet 通过
- [x] 3 个 Resource Config Python IT 脚本语法检查通过
- [x] gopls diagnostics 通过
- [x] OpenAPI 生成与新增 operation/参数断言通过
- [x] model / `sql/init.sql` / `sql/0728-resource-management.sql` schema 对齐
- [x] EKS 隔离 IT 3/3 脚本、116/116 HTTP、真实 CVM 与资源清理通过
- [x] Docs、UT、IT、Review 和 migration 产物已纳入待提交 worktree

## 已知全仓验证限制

- Review 阶段 root `go test ./...` 的 controller 包在非资源策略用例 `TestCreateSkillInstallTasks_RolePlusAllBundle` 超时；该用例单独复跑通过。
- task 包既有 doctor cleanup 后台 goroutine 在测试 DB 清理后 panic；与资源策略路径无关。
- root `go vet ./...` 仅保留既有 `skillhubclient/client_test.go` 在检查 error 前使用 response 的告警。
- 上述限制未记为通过；资源策略定向竞态、受影响包 vet、nested integration 与 smoke/IT 均有独立通过证据。

## 最终提交范围

- 独立 `ResourcePolicy` 实体，配置只存 `ConfigJSON`；
- 通过可双向索引的 `GroupConfigBinding` 表达策略应用范围；
- 并发安全懒创建、固定 canonical 名称和受保护的企业默认策略；
- 本组 → 最近祖先 → 企业默认的资源策略 resolver；
- 管理端策略 CRUD、实例类型/系统盘 options、用户组树直接策略元数据；
- 默认策略读取时 i18n，普通策略名称保持原值，本地化名称可 round-trip update；
- 创建路径资源配置覆盖、镜像/系统盘容量 fail-fast 和目标机型 SELL 预检；
- GORM model、MySQL init schema、`0715` migration、审计与 i18n；
- R-U01–R-U12、Review 回归、真实腾讯云 I01–I12、integration runner 日志脱敏；
- API/OpenAPI、IT README 与 SOP 01–08 产物；
- 分组 `config-overview` 解析策略 ID，并在 `meta.value` 返回与旧版同格式的具体 ResourceConfig；`meta.resource_config` 作为显式别名，存储模型不变；
- 与本功能无关的 `docs/basic/platform-policy.md`、`docs/INDEX.md` 保持基线不变。

本地 `.omp/`、`docs/openapi_base.json`、`test-report/` 和覆盖率报告不提交。

## Commit Message

```text
feat(resource-policy): redesign resource policy management

Store policies independently and bind their application scope through indexed group bindings.
Add protected lazy enterprise defaults, inheritance resolution, admin APIs, and localized display.
Update schema, tests, integration coverage, and API documentation.
```

## 覆盖策略

- 分支：`feat/resource-config-policy`
- 上游：`origin/feat/resource-config-policy`
- 保留父提交：`3c27393e^`
- 按用户 2026-07-22 明确要求，将本任务此前 3 个已发布提交压缩为一个最终提交。
- 2026-07-22 追加总览响应兼容后再次按用户要求覆盖；推送使用 `--force-with-lease`，仅在远端仍指向当前已知 `63911821` 时覆盖，避免吞掉并发提交。
- 不纳入本地验证产物或凭证。
