# 03. Implement — 实现记录

---

## 关键实现细节

### Hatchery 后端

- 新增 `PasswordlessLoginToken` 租户模型：仅保存 SHA-256 摘要、目标用户和绝对过期时间；GORM 条件删除以 `RowsAffected == 1` 决定唯一消费成功者。
- 新增 `POST /admin/passwordless-login-link`：复用其它管理接口的 `requireAdmin`，管理员可为当前租户任意有效用户签发；同时校验 `passwordless-login` 功能白名单，返回基于可信 `TenantSnapshot.Domain` 的 HTTPS Fragment 链接。
- `IsFeatureAllowed` 只接收 feature type，并从 model 集中配置读取空表策略：既有 Local Agent 保持空表开放；免登录签发、消费和诊断统一为空表拒绝，调用方不再手动组合 type 与 policy。
- 新增 `POST /auth/passwordless-login`：消费凭证后重新查询有效用户，覆盖建立本地 Session，并清理 OneID 身份字段、旧 `user_id` 和失败计数。
- 账号密码登录与免登录共用 `establishLocalSession`，避免两套 Session 字段语义漂移。
- 两个 POST 路由均接入 `WithAudit`；明文凭证不写数据库、日志或审计记录。
- 新模型已加入 `allModels`；同步 `sql/init.sql` 和 `sql/0721-passwordless-login.sql`。两分钟有效的一次性凭证不参与 SQLite→MySQL 数据迁移，仅登记 migration coverage，切换后未消费链接直接失效。
- 新增模型与 Handler 契约测试，覆盖摘要存储、单次消费、并发、过期、租户隔离、白名单、严格 AdminToken、Session 和重放拒绝。

### OpenClaw Enterprise 前端

- 在独立仓库 `../openclaw-enterprise-fronted` 的 `feature/passwordless-login-link` 分支新增公开路由 `/passwordless-login`。
- 落地页只在函数局部读取 Fragment 凭证，立即通过 `history.replaceState` 清除地址栏；不写 React state、Zustand 或浏览器存储。
- `useRef` 防止组件生命周期内重复消费；Axios 请求使用 `silentError`，失败由页面稳定展示且不自动重试。
- 成功后使用 `window.location.replace("/")` 完整刷新，由现有 `authApi.check()` 同步真实用户；缺失/无效凭证展示现有品牌 Logo、国际化错误和返回首页按钮。
- 免登录页面新增文案均使用 `t()` 标记；按用户确认，不提交前端自动生成的 `i18n/ClawPro/*.json` 翻译产物。
- 保留且未修改前端仓库原有 `Dockerfile` 用户变更。

## 与 Plan 差异

- SQLite→MySQL 迁移明确跳过 `passwordless_login_tokens`；瞬时凭证不值得迁移，也不阻断历史数据库迁移。
- `docs/API.md`、`docs/login-logic.md` 和 IT 脚本按 SOP 分别留到 Docs、IT 步骤，不在 Implement 提前完成。
- 前端全量 `pnpm check` 被仓库既有类型错误阻断；本次四个改动文件的定向 ESLint 为 0 错误/0 警告。

## 实现期验证

- `go test ./controller -run '^TestPasswordlessLogin_FullFlowAndReplay$' -count=1 -v`：通过；签发、摘要落库、Session、首次消费和重放拒绝均实际执行。
- `go vet ./controller ./model`：通过。
- `go vet ./...`：被既有 `skillhubclient/client_test.go:278`「using resp before checking for errors」阻断，本次改动包无该问题。
- Chromium：
  - 直接访问 `/passwordless-login` 展示“登录失败 / 免登录链接无效或已过期 / 返回首页”，且没有发起消费请求。
  - 携带 43 字符伪造 Fragment 刷新后地址栏立即清除 Fragment；后端 404 安全错误由页面展示，无全局 toast。

## 检查项

- [x] `gofmt` 格式化通过
- [x] 变更包 `go vet ./controller ./model` 无错误
- [x] 写接口已添加审计日志
- [x] 数据库变更已同步 `sql/init.sql` + migration SQL
- [x] 使用 `model.DB(r.Context())`，全局白名单仅使用 `DBGlobal`
- [x] 无硬编码密钥/配置
- [x] 后端新增 i18n Key 已同步英文翻译；前端新增文案均使用 `t()` 标记
