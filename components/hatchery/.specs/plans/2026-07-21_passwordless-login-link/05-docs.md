# 05. Docs — 文档更新

---

## 更新清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `docs/API.md` | 新增 | 记录 AdminToken 签发接口、公开消费接口、请求/响应、状态码、Session 覆盖和凭证安全约束 |
| `docs/API.md` | 新增 | 记录 `passwordless-login` 空表默认关闭策略、诊断 API、运维 SQL，以及删除最后一条会关闭全部租户 |
| `../openclaw-enterprise-fronted/docs/login-logic.md` | 修改 | 登录模式扩展为四种，新增公开落地页、Fragment 清理、Axios 调用、完整刷新及安全时序 |
| `../openclaw-enterprise-fronted/docs/login-logic.md` | 修正 | 按当前 `request.ts` 实现修正错误拦截说明：`silentError` 抑制 toast，401 不由拦截器自动跳转 |

## 契约核对

- 后端路由与文档一致：
  - `POST /admin/passwordless-login-link`
  - `POST /auth/passwordless-login`
- 链接契约一致：`/passwordless-login#passwordless_login_token=<opaque-token>`。
- TTL 与实现一致：`expires_in=120`。
- 前端公开路由、`authApi.passwordlessLogin()`、Fragment 清理和 `silentError` 均与源码逐项核对。
- 所有新增请求参数表使用“参数 / 类型 / 必填 / 说明”四列格式。
- 未新增独立模块文档，因此无需修改 `docs/INDEX.md` 或 `.specs/docs/`。

## 检查项

- [x] `docs/API.md` 已更新
- [x] 前端 `docs/login-logic.md` 已同步
- [x] 功能白名单的集中空表策略与运维影响已说明
- [x] 参数表格式符合 CLAUDE.md 要求（4 列、参数名无反引号）
- [x] 文档中的路由、字段名、TTL、状态码与代码契约核对一致
