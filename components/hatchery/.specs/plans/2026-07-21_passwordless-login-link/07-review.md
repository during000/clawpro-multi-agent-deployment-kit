# 07. Review — 代码审查

---

## 审查方式

- [x] AI 自动 Review
- [ ] 人工 Review

## 发现并修复的问题

| # | 文件 | 问题 | 严重度 | 状态 |
|---|------|------|-------|------|
| 1 | `model/feature_allowlist.go` | 直接复用空表全开会导致免登录功能发布后默认全量开放，调用方手动组合 type 与 policy 还可能错配 | 高 | 已修：空表策略按 feature type 在 model 集中配置，免登录默认拒绝，Local Agent 行为不变 |
| 2 | `controller/auth_passwordless.go` | 将普通管理 API 错误收窄为进程 AdminToken 专用接口 | 高 | 已修：与其它 `/admin` API 一致，仅使用 `requireAdmin` |
| 3 | 前端入口与 Aegis | Navigation Timing 会保留含一次性凭证的原始 Fragment 并被 RUM 上报 | 高 | 已修：模块加载前清理 + `urlHandler`/`beforeRequest` 脱敏，浏览器验证无泄漏 |
| 4 | IT 脚本 | 默认 HTTP 帧日志会输出一次性凭证请求体 | 中 | 已修：该脚本默认 `QUIET=1` |

## 审查结论

- [x] 无高严重度未修复问题
- [x] 标准管理员鉴权、租户白名单和目标用户查询边界一致
- [x] 凭证仅存 SHA-256 摘要，首次消费使用条件删除保证原子单次成功
- [x] 后端用户文案中英文翻译完整；前端新增文案均使用 `t()` 标记且不提交生成翻译 JSON
- [x] 数据库模型、初始化 SQL、增量迁移和 SQLite→MySQL 覆盖同步
- [x] `git diff --check` 与前端目标文件 ESLint 通过
