# 08. Commit — 提交记录

## 提交前检查

> 执行顺序：① 写完本文件 → ② 更新 `00-overview.md` 状态为已完成 →
> ③ 确认所有 `0N-*.md` 已就绪 → ④ `git add` → ⑤ `git commit` → ⑥ `git push`

- [x] `01-clarify.md` ～ `08-commit.md` 阶段产物均已填写
- [x] `00-overview.md` Meta/Progress/当前步骤已完成
- [x] `gofmt` 已覆盖 Review 触及的 Go 文件
- [x] `go vet ./...` 通过
- [x] `go test ./controller -count=1` 通过
- [x] adjustment、role、MCP、task、model focused race 通过
- [x] OpenAPI generator unittest、Python compile 与 `make openapi BASE_BRANCH=origin/master` 通过
- [x] IT I01～I16 与 Review 确定性真实云 E2E 通过，专用云/K8s 资源已清理
- [x] Review 初审 2 高、5 中问题全部修复；独立复核无剩余 High/Medium
- [x] 临时 AK/SK/admin-token 日志、生成报告和 base spec 已删除
- [x] `AGENTS.md` 是本地 `CODEBUDDY.md` 识别软链接：保持未跟踪，不提交，也不加入 `.gitignore`
- Commit 记录时间：`2026-07-20 10:37:57` ～ `2026-07-20 10:41:10`（`3分13秒`）

## Commit Message

```text
feat(controller): support cloud instance resource adjustments

Add administrator validation and submission APIs for AI2 upgrades and
system-disk expansion, backed by a durable per-instance worker.

Enforce lifecycle/config mutation locks, persist resource and failure state,
and expose resource fields through admin list and status APIs.

Add tenant-aware CVM/CBS gates, migrations, i18n, API/OpenAPI contracts,
unit/race coverage, and deterministic real-cloud integration coverage.
```

## 提交范围

- 管理端 `/admin/instances/adjust-config/validate` 与 `/admin/instances/adjust-config`。
- AI2 族内严格升配、CLOUD 系统盘扩容、完整 CVM/CBS 实时校验，以及仅用于规格升配的价格询价。
- instances 单行持久状态机、RequestId 崩溃恢复、15 分钟边界、原状态恢复和稳定失败码。
- 生命周期、命令、技能、插件、MCP、角色等共享写入口的 adjustment lock。
- 管理端 list/status 资源字段和筛选、GORM/MySQL migration、i18n、审计、API/OpenAPI。
- U01～U41、I01～I16 及 Review 新增的崩溃重放/CAS/TOCTOU/OpenAPI/E2E 回归。

## 推送目标

- 分支：`feature/clawpro-agent-instance-upgrade`
- 远端：`origin`

## 2026-07-22 Commit 补充

升配提交额外包含系统盘询价门禁移除：规格询价保留，系统盘按本地事实与配额规则校验，实际写失败稳定映射；UT、IT 脚本和 API 文档同步更新。
