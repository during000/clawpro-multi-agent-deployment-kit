# 01. Clarify — 需求澄清

> AI 以产品经理角色进行 Discovery + Challenge，确保需求清晰、边界明确。

---

## 背景

- 客户：法大大。
- TAPD 需求 1020422209135809941 要求 ClawPro 对外提供指定用户免登录能力：调用方使用 admin token 申请链接，用户打开链接后由前端调用鉴权接口建立登录态。
- 现有系统支持账号密码、OneID/Gateway 内部登录、Session Cookie 和 admin Bearer Token；尚无“管理员为指定本地用户签发、两分钟有效、单次消费”的外部登录凭证，也没有按免登录会话限制功能范围的机制。
- TAPD 需求无评论、附件或原型补充；仓库已有按 tenant identifier 控制功能开放范围的全局 `FeatureAllowlist`，当前唯一正式类型为 `local-agent`，因此“功能白名单”更可能指同类灰度白名单，但仍需产品确认。
## 目标

- [ ] 仅受信任调用方可为一个现存且未封禁的用户申请免登录链接。
- [ ] 凭证自签发起 2 分钟内有效，并通过数据库原子消费保证并发场景最多成功一次。
- [ ] 前端从 URL Fragment 读取凭证并调用鉴权接口，避免凭证进入服务端访问日志、Referer 和查询参数；成功后跳转首页 `/`。
- [ ] 鉴权成功后建立与普通登录一致的 `hatchery-session`；若沿用现有功能白名单语义，不向 Session 引入额外功能权限。
- [ ] 失败场景覆盖：调用方未授权、用户不存在或已封禁、当前租户未获功能白名单放行、凭证伪造、过期、重复消费、租户不匹配。
- [ ] 申请和消费行为均可审计，日志和 API 错误不得泄露明文凭证。
- [ ] 国内站、国际站使用相同接口语义；所有用户可见文案支持中英文。
- [ ] 更新 API/功能文档，并由集成测试覆盖两个新增接口及全部新增请求参数。
## 范围

| 包含 | 不包含 |
|------|--------|
| admin token 申请指定用户免登录链接的后端接口 | 改造现有账号密码或 OneID 登录协议 |
| 免登录凭证的安全生成、哈希存储、两分钟过期和单次原子消费 | 向第三方暴露用户密码、API Token 或 Session Cookie |
| 前端免登录落地页、鉴权调用、成功后首页跳转 | 跨站点或跨租户复用同一凭证 |
| 复用全局 `FeatureAllowlist`，按 tenant identifier 控制免登录功能开放范围 | 新建会话级功能权限或通用 RBAC/权限管理平台 |
| 数据模型、初始化 SQL、增量 migration、SQLite 迁移覆盖 | 修改 Gateway/OneID 服务 |
| i18n、审计、UT、IT 和 API 文档 | 与本需求无关的登录页或后台 UI 重构 |
## 待确认问题

| # | 问题 | 状态 | 建议结论 |
|---|------|------|----------|
| 1 | “功能白名单”是否沿用现有 `FeatureAllowlist`，按 tenant identifier 控制哪些租户可使用免登录功能？ | 已确认 | 是；新增 `type='passwordless-login'`，不把它解释为免登录后的页面/API 权限列表。 |
| 2 | 申请接口如何指定目标用户：`user_id`、`username`，还是两者均支持？ | 待确认 | 只接收稳定且与现有管理 API 一致的 `user_id`，避免用户名变更和歧义。 |
| 3 | “admin token”是否严格指启动参数配置的超级 `AdminToken`，还是也允许管理员 Session/用户 API Token？ | 已调整 | 用户确认这是普通 `/admin` API，直接复用 `requireAdmin`；管理员用户 API Token、管理员 Session 和进程 AdminToken 与其它管理接口保持一致。 |
| 4 | 申请接口应返回完整绝对 URL 还是站内相对路径？绝对 URL 的可信域名来源是什么？ | 待确认 | 返回完整 URL；只使用 TenantSnapshot 中可信的 `Domain`，不使用请求 Host Header，凭证放在 Fragment 中。 |
| 5 | 新类型 `passwordless-login` 没有任何白名单记录时，是否沿用现有“空表=全开”语义？ | 已调整 | 不沿用；用户在 IT 前置审查中确认空表默认关闭。`IsFeatureAllowed` 按 feature type 读取 model 集中配置的空表策略，Local Agent 保持原语义，免登录必须命中 identifier。 |
| 6 | 浏览器已有登录态时消费免登录链接，是否覆盖为目标用户？ | 待确认 | 覆盖现有 Session，并清除 OneID 身份字段，行为与账号密码登录保持一致。 |
| 7 | 功能白名单由运维直接维护数据库，还是需要新增管理 API/页面？ | 已确认 | 沿用现状，由运维维护 `feature_allowlists`；本需求只复用查询和守卫，不额外扩展白名单 CRUD。 |
## 约束与依赖

- 安全：使用密码学安全随机不透明凭证；数据库只保存摘要；常量时间比较；消费成功前不得建立 Session。
- 并发：部署可能有多个 Pod，不能依赖进程内 map/定时器；消费通过按摘要和有效期匹配的条件删除抢占，只有 `RowsAffected == 1` 的请求可建立 Session。
- 多租户：全部查询使用 `model.DB(r.Context())`，凭证绑定当前 tenant identifier，禁止跨租户消费。
- 鉴权：申请接口先按需求确认 `AdminToken` 边界；消费接口只验证一次性凭证，不要求预先登录。
- Session：复用 `hatchery-session` 的 `username`、`role`、`user_id`、`login_at`、identifier 语义；除非产品明确要求会话级权限，否则不新增功能范围字段。
- 数据库：新增 GORM model 时必须同步 `model/db.go`、`sql/init.sql`、目标 Release 日期命名的增量 migration，以及 SQLite→MySQL 迁移覆盖。
- 接口：两个写操作均注册审计规则并使用 `WithAudit`；所有外部输入校验，响应不包含密码、用户 API Token 等敏感数据。
- 前端：Hatchery 自身 UI 已停用；免登录落地页在相邻 `../openclaw-enterprise-fronted` SPA 新增公开路由 `/passwordless-login`，通过 `/api/auth/passwordless-login` 建立 Cookie Session。
- 时间：TAPD 期望完成时间为 2026-07-23；需求状态“待启动”，优先级“高”。
