# 01. Clarify — 需求澄清

> AI 以产品经理角色进行 Discovery + Challenge，确保需求清晰、边界明确。

---

## 背景

SkillHub 是技能管理的新平台，Hatchery 需要将技能列表等 API 逐步迁移到 SkillHub。Phase 1 已完成基础迁移工具和方案设计（根目录 `migration-plan.md`），Phase 2 需要实现技能列表 API 的灰度代理。

**核心挑战**：
- Hatchery 不持有 OneID 私钥，无法直接签发 JWT assertion 获取 access_token
- 企业技能列表 API 非公开，需要租户+用户级别的 access_token
- 需要无侵入地切换路由，不能修改现有 `admin_skills.go` 逻辑
- 多 pod 副本下的 token 缓存管理

---

## 目标

- [x] 技能列表 API 支持灰度切换：按租户控制是否走 SkillHub
- [x] 通过 Gateway 代理获取 OneID access_token
- [x] 装饰器模式实现无侵入路由切换
- [x] 缓存 access_token 和 OrgID，减少重复请求
- [x] 提供 `/admin/skillhub-status` 端点查询灰度状态
- [x] 单元测试覆盖率 > 60%

---

## 范围

| 包含 | 不包含 |
|------|--------|
| 技能列表 API（GET /admin/skills）灰度代理 | 其他技能 API（创建/删除/更新等 Phase 3） |
| Gateway `/api/access-token` 端点（openclaw-oneid-gateway 仓库） | SkillHub 前端改造 |
| `site_configs` 新增灰度字段 | 数据迁移 |
| SkillHub API client + 格式适配层 | SkillHub 本身功能开发 |
| `/admin/skillhub-status` 状态查询端点 | 运维部署脚本 |

---

## 待确认问题

| # | 问题 | 状态 | 结论 |
|---|------|------|------|
| 1 | 灰度判断只看 `site_configs.skill_hub_enabled`？ | 已确认 | 是，仅看此字段 |
| 2 | 是否需要新增 `skill_hub_oneid_app_type` 字段？ | 已确认 | 不需要，`aud_app_type` 在 Gateway 写死为 `"skillhub"` |
| 3 | `skill_hub` 字段会被用户修改，如何区分 API 地址？ | 已确认 | 新增 `skill_hub_api_url` 字段（admin 控制），前端 URL 由 API URL 推导 |
| 4 | access_token 缓存如何管理？是否持久化到 DB？ | 已确认 | 内存缓存 `sync.Mutex+map`（与 `getOneIDAppToken` 同模式），不落 DB |
| 5 | 是否需要缓存失效逻辑（私钥轮换后）？ | 已确认 | 不需要，token 有效期短（~1800s），自动过期刷新 |

---

## 约束与依赖

- **依赖 Gateway**：Hatchery 通过 Gateway `POST /api/access-token` 获取 access_token，Gateway 需部署对应端点
- **依赖 SkillHub**：SkillHub 需提供 `GET /api/v1/auth/me` 和 `GET /api/v1/orgs/{orgId}/skills` 接口
- **数据库约束**：`skill_hub_api_url` 使用 `TEXT` 类型（非 `varchar(512)`），避免 MySQL row size 超限
- **Go 版本**：1.21+，使用标准库 `net/http`
- **安全**：access_token 不落日志，缓存仅存内存
