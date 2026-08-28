# Hatchery API 接口文档

## 概览

Hatchery 使用标准 Go `net/http` 路由。除少量特殊接口（SSE 流、图片、静态资源、WebSocket 等）外，接口均返回 JSON 响应（HTML/HTMX 前端已废弃，`forceJSONMiddleware` 会强制将每个请求的 `Accept` 设为 `application/json`）。

**认证方式：**

1. **Cookie Session**（`hatchery-session`）— 通过 `POST /login` 获取，网页登录用户使用
2. **用户 API Token** — 用户通过网页管理自己的 Token（`GET /api-token`、`POST /api-token/create`、`POST /api-token/reset`、`POST /api-token/revoke`），请求时携带 `Authorization: Bearer hk-xxx` Header。**仅在开放 API 白名单路由中生效**，非白名单路由会忽略用户 Token 回退到 Session 认证
3. **AdminToken** — 启动时通过 `--admin-token` 参数设置，请求时携带 `Authorization: Bearer <token>` Header。**不受白名单限制**，可访问所有接口。Token 用户不拥有实例（`user_id=0`），实例相关查询会返回空列表

**权限等级：** 公开 / 登录用户 / 管理员（`requireAdmin`）

**开放 API（支持 API Token 访问的接口）：**

以下接口支持用户 API Token 认证（标记为 🔓 的接口需管理员角色的 API Token）。API Token 请求会自动设置 `Accept: application/json`，确保返回 JSON 格式响应。

| 分类 | 路由 | 方法 |
|------|------|------|
| 公共 | `/site` | GET |
| 公共 | `/health` | GET |
| 实例管理 | `/openclaw/list` | GET |
| 实例管理 | `/openclaw/current-image` | GET |
| 实例管理 | `/openclaw/status` | GET |
| 实例管理 | `/openclaw/service-status` | GET |
| 实例管理 | `/openclaw/zones` | ANY |
| 实例管理 | `/openclaw/agent-types` | GET |
| 实例管理 | `/openclaw/images/update-notices` | GET |
| 实例管理 | `/openclaw/channels` | GET |
| 实例管理 | `/openclaw/models` | GET |
| 实例管理 | `/openclaw/skills` | GET |
| 实例管理 | `/openclaw/create` | POST |
| 实例管理 | `/openclaw/delete` | POST |
| 实例管理 | `/openclaw/start` | POST |
| 实例管理 | `/openclaw/stop` | POST |
| 实例管理 | `/openclaw/reboot` | POST |
| 实例管理 | `/openclaw/restart-gateway` | POST |
| 实例管理 | `/openclaw/reset` | POST |
| 实例管理 | `/openclaw/retry` | POST |
| 实例管理 | `/openclaw/rename` | POST |
| 实例管理 | `/openclaw/notifications` | GET |
| 实例管理 | `/openclaw/notifications/read` | POST |
| 实例管理 | `/openclaw/notifications/count` | GET |
| 实例管理 | `/openclaw/notifications/delete` | POST |
| 实例管理 | `/openclaw/proxy/prepare` | POST |
| 实例管理 | `/openclaw/set-channel` | POST |
| 实例管理 | `/openclaw/del-channel` | POST |
| 实例管理 | `/openclaw/set-model` | POST |
| 实例管理 | `/openclaw/add-model` | POST |
| 实例管理 | `/openclaw/switch-primary-model` | POST |
| 实例管理 | `/openclaw/del-model` | POST |
| 实例管理 | `/openclaw/instance-models` | GET |
| 实例管理 | `/openclaw/version` | GET |
| 实例管理 | `/openclaw/agent-count` | GET |
| 实例管理 | `/openclaw/add-skill` | POST |
| 实例管理 | `/openclaw/update-skill` | POST |
| 实例管理 | `/openclaw/uninstall-skill` | POST |
| 实例管理 | `/openclaw/upgrade` | POST |
| 实例管理 | `/openclaw/upgrade/retry` | POST |
| 实例管理 | `/openclaw/migration/export` | POST |
| 实例管理 | `/openclaw/migration/status` | GET |
| 实例管理 | `/openclaw/migration/progress` | GET |
| 实例管理 | `/openclaw/migration/import` | POST |
| 实例管理 | `/openclaw/add-plugin` | POST |
| 实例管理 | `/openclaw/install-skills` | GET |
| 本地 agent reporter | `/local-agent/report` | POST |
| 本地 agent reporter | `/local-agent/sync` | POST |
| 本地 agent reporter | `/local-agent/commands/ack` | POST |
| 本地 agent reporter | `/local-agent/wake-ticket` | POST |
| 本地 agent reporter | `/local-agent/wake` | GET（WebSocket） |
| 本地 agent | `/local-agent/availability` | GET |
| 本地 agent | `/local-agent/get-config` | GET |
| Agent 协作 | `/agent-tasks` | GET |
| Agent 协作 | `/agent-tasks/create` | POST |
| 本地 agent 三期 | `/local-agent/remove` | POST |
| 本地 agent 三期 | `/admin/local-agent/remove` | POST |
| 本地 agent 二期 | `/openclaw/local/user-group` | POST |
| 实例管理 | `/openclaw/retry-failed-skills` | POST |
| 实例管理 | `/openclaw/cancel-failed-skills` | POST |
| 实例管理 | `/openclaw/set-gateway-ui` | POST |
| 实例管理 | `/openclaw/ws-url` | POST |
| 实例管理 | `/openclaw/check-gateway-access` | GET |
| 实例管理 | `/openclaw/set-env` | POST |
| 实例管理 | `/openclaw/env` | GET |
| 实例管理 | `/openclaw/terminal-url` | POST |
| 实例管理 | `/openclaw/approve` | POST |
| 实例管理 | `/openclaw/denied-actions` | POST |
| 实例管理 | `/openclaw/memory-tdai-status` | GET |
| 实例管理 | `/openclaw/auto-channel` | GET |
| 实例管理 | `/openclaw/check-openclaw-port` | GET |
| 实例管理 | `/openclaw/smh-status` | GET |
| 实例管理 | `/openclaw/smh-token` | GET |
| VNC 云端浏览器 | `/openclaw/browser-vnc-check` | GET |
| VNC 云端浏览器 | `/openclaw/browser-vnc-install` | POST |
| VNC 云端浏览器 | `/openclaw/browser-vnc-access` | GET |
| VNC 云端浏览器 | `/openclaw/browser-status` | GET |
| VNC 云端浏览器 | `/openclaw/browser-takeover` | POST |
| Agent-Bridge | `/agent-bridge/audit` | POST |
| 用户配额 | `/quota/data` | GET |
| 用户配额 | `/quota/logs` | GET |
| 管理 🔓 | `/admin/users` | GET |
| 管理 🔓 | `/admin/config` | GET |
| 管理 🔓 | `/admin/config` | POST |
| 管理 🔓 | `/admin/config/cvm` | POST |
| 管理 🔓 | `/admin/config/template` | POST |
| 管理 🔓 | `/admin/resource-policies` | GET |
| 管理 🔓 | `/admin/resource-policies/create` | POST |
| 管理 🔓 | `/admin/resource-policies/update` | POST |
| 管理 🔓 | `/admin/resource-policies/delete` | POST |
| 管理 🔓 | `/admin/resource-policies/options/instance-types` | GET |
| 管理 🔓 | `/admin/resource-policies/options/system-disks` | GET |
| 管理 🔓 | `/admin/config/security-group` | GET/POST/PUT |
| 管理 🔓 | `/admin/config/security-group/list` | GET |
| 管理 🔓 | `/admin/config/security-group/required-rules` | GET |
| 管理 🔓 | `/admin/config/security-group/check-rules` | GET |
| 管理 🔓 | `/admin/config/security-group/policies` | GET |
| 管理 🔓 | `/admin/config/security-group/cloud-policies` | GET |
| 管理 🔓 | `/admin/config/security-group/ruleset` | GET |
| 管理 🔓 | `/admin/config/security-group/rulesets` | POST |
| 管理 🔓 | `/admin/config/security-group/ruleset/rules` | POST |
| 管理 🔓 | `/admin/config/security-group/ruleset/rules/reorder` | POST |
| 管理 🔓 | `/admin/config/security-group/ruleset/import-from-sg` | POST |
| 管理 🔓 | `/admin/vpc/cloud` | GET |
| 管理 🔓 | `/admin/subnet/cloud` | GET |
| 管理 🔓 | `/admin/create` | POST |
| 管理 🔓 | `/admin/batch-create` | POST |
| 管理 🔓 | `/admin/delete` | POST |
| 管理 🔓 | `/admin/hard-delete` | POST |
| 管理 🔓 | `/admin/restore` | POST |
| 管理 🔓 | `/admin/reset-password` | POST |
| 管理 🔓 | `/admin/update-user` | POST |
| 管理 🔓 | `/admin/user-token` | GET |
| 管理 🔓 | `/admin/token/disable` | POST |
| 管理 🔓 | `/admin/token/enable` | POST |
| 管理 🔓 | `/admin/user-limit` | GET |
| 管理 🔓 | `/admin/user-vpc` | GET |
| 管理 🔓 | `/admin/channels` | GET |
| 管理 🔓 | `/admin/channels/toggle` | POST |
| 管理 🔓 | `/admin/channels/add` | POST |
| 管理 🔓 | `/admin/channels/delete` | POST |
| 管理 🔓 | `/admin/models` | GET |
| 管理 🔓 | `/admin/models/create` | POST |
| 管理 🔓 | `/admin/models/update` | POST |
| 管理 🔓 | `/admin/models/delete` | POST |
| 管理 🔓 | `/admin/models/toggle` | POST |
| 管理 🔓 | `/admin/models/toggle-enabled` | POST |
| 管理 🔓 | `/admin/models/toggle-default` | POST |
| 管理 🔓 | `/admin/models/visibility` | POST |
| 管理 🔓 | `/admin/user-groups/associated-models` | GET |
| 管理 🔓 | `/admin/images` | GET |
| 管理 🔓 | `/admin/images/cloud` | GET |
| 管理 🔓 | `/admin/images/import` | POST |
| 管理 🔓 | `/admin/images/delete` | POST |
| 管理 🔓 | `/admin/images/enable` | POST |
| 管理 🔓 | `/admin/images/update` | POST |
| 管理 🔓 | `/admin/images/history/publish` | POST |
| 管理 🔓 | `/admin/images/history/update` | POST |
| 管理 🔓 | `/admin/images/history/delete` | POST |
| 管理 🔓 | `/admin/images/history/restore` | POST |
| 管理 🔓 | `/admin/images/update-notice` | POST |
| 管理 🔓 | `/admin/images/history` | GET |
| 管理 🔓 | `/admin/images/set-default-type` | POST |
| 管理 🔓 | `/admin/agent-types` | GET |
| 管理 🔓 | `/admin/agent-types/enabled` | POST |
| 管理 🔓 | `/admin/agent-types/create` | POST |
| 管理 🔓 | `/admin/agent-types/delete` | POST |
| 管理 🔓 | `/admin/local-agent-types` | GET |
| 管理 🔓 | `/admin/feature-allowlist/check` | GET |
| 管理 🔓 | `/admin/instances` | GET |
| 管理 🔓 | `/admin/instances/adjust-config/validate` | POST |
| 管理 🔓 | `/admin/instances/adjust-config` | POST |
| 管理 🔓 | `/admin/instances/create` | POST |
| 管理 🔓 | `/admin/instances/delete` | POST |
| 管理 🔓 | `/admin/instances/start` | POST |
| 管理 🔓 | `/admin/instances/stop` | POST |
| 管理 🔓 | `/admin/instances/reboot` | POST |
| 管理 🔓 | `/admin/instances/restart-gateway` | POST |
| 管理 🔓 | `/admin/instances/reset` | POST |
| 管理 🔓 | `/admin/instances/refresh-version` | POST |
| 管理 🔓 | `/admin/instances/batch-upgrade` | POST |
| 管理 🔓 | `/admin/instances/terminal-url` | POST |
| 管理 🔓 | `/admin/instances/denied-actions` | POST |
| 管理 🔓 | `/admin/instances/cam-role` | POST |
| 管理 🔓 | `/admin/instances/channels` | GET |
| 管理 🔓 | `/admin/instances/skills` | GET |
| 管理 🔓 | `/admin/instances/models` | GET |
| 管理 🔓 | `/admin/instances/available-models` | GET |
| 管理 🔓 | `/admin/instances/available-channels` | GET |
| 管理 🔓 | `/admin/instances/set-model` | POST |
| 管理 🔓 | `/admin/instances/batch-set-model` | POST |
| 管理 🔓 | `/admin/instances/add-model` | POST |
| 管理 🔓 | `/admin/instances/switch-primary-model` | POST |
| 管理 🔓 | `/admin/instances/del-model` | POST |
| 管理 🔓 | `/admin/instances/proxy/prepare` | POST |
| 管理 🔓 | `/admin/instances/set-channel` | POST |
| 管理 🔓 | `/admin/instances/del-channel` | POST |
| 管理 🔓 | `/admin/instances/detect-install` | POST |
| 管理 🔓 | `/admin/instances/status` | GET |
| 管理 🔓 | `/admin/cls/open` | POST |
| 管理 🔓 | `/admin/cls/close` | POST |
| 管理 🔓 | `/admin/cls/status` | GET |
| 管理 🔓 | `/admin/cls/scope` | GET/POST |
| 管理 🔓 | `/admin/cls/update` | GET/POST |
| 管理 🔓 | `/admin/notices` | GET |
| 管理 🔓 | `/admin/departments` | GET |
| 管理 🔓 | `/admin/export-tokens` | POST |
| 管理 🔓 | `/admin/memory-tdai/config` | GET/PUT |
| 管理 🔓 | `/admin/usage/data` | GET |
| 管理 🔓 | `/admin/usage/logs` | GET |
| 管理 🔓 | `/admin/audit` | GET |
| 管理 🔓 | `/admin/skill-categories` | GET |
| 管理 🔓 | `/admin/skill-categories/create` | POST |
| 管理 🔓 | `/admin/skill-categories/update` | POST |
| 管理 🔓 | `/admin/skill-categories/delete` | POST |
| 管理 🔓 | `/admin/skills` | GET |
| 管理 🔓 | `/admin/skills/create` | POST |
| 管理 🔓 | `/admin/skills/update` | POST |
| 管理 🔓 | `/admin/skills/delete` | POST |
| 管理 🔓 | `/admin/skills/detail` | GET |
| 管理 🔓 | `/admin/skills/files` | GET |
| 管理 🔓 | `/admin/skills/references` | GET |
| 管理 🔓 | `/admin/skills/instances` | GET/POST |
| 管理 🔓 | `/admin/skills/tasks` | GET |
| 管理 🔓 | `/admin/skills/distribute` | POST |
| 管理 🔓 | `/admin/skills/uninstall` | POST |
| 管理 🔓 | `/admin/skillhub-status` | GET |
| 管理 🔓 | `/admin/skill-bundles` | GET |
| 管理 🔓 | `/admin/skill-bundles/create` | POST |
| 管理 🔓 | `/admin/skill-bundles/delete` | POST |
| 管理 🔓 | `/admin/skill-bundles/toggle` | POST |
| 管理 🔓 | `/admin/skill-bundles/detail` | GET |
| 管理 🔓 | `/admin/skill-bundles/update-skills` | POST |
| 管理 🔓 | `/admin/skill-bundles/batch-add-skills` | POST |
| 管理 🔓 | `/admin/skill-bundles/update-visibility` | POST |
| 管理 🔓 | `/admin/skills/favorite` | POST |
| 管理 🔓 | `/admin/skills/unfavorite` | POST |
| 管理 🔓 | `/admin/skills/favorited` | GET |
| 管理 🔓 | `/admin/skillsets/favorite` | POST |
| 管理 🔓 | `/admin/skillsets/unfavorite` | POST |
| 管理 🔓 | `/admin/skillsets/favorited` | GET |
| 管理 🔓 | `/admin/skills/scan-trigger` | POST |
| 管理 🔓 | `/admin/skills/scan-config` | GET/POST |
| 管理 🔓 | `/admin/rules` | GET |
| 管理 🔓 | `/admin/rules/detail` | GET |
| 管理 🔓 | `/admin/rules/create` | POST |
| 管理 🔓 | `/admin/rules/delete` | POST |
| 管理 🔓 | `/admin/rules/update` | POST |
| 管理 🔓 | `/admin/rules/files` | GET |
| 管理 🔓 | `/admin/rules/tasks` | GET |
| 管理 🔓 | `/admin/rules/instances` | GET |
| 管理 🔓 | `/admin/rules/distribute` | POST |
| 管理 🔓 | `/admin/rules/uninstall` | POST |
| 管理 🔓 | `/admin/projects` | GET |
| 管理 🔓 | `/admin/projects/create` | POST |
| 管理 🔓 | `/admin/projects/update` | POST |
| 管理 🔓 | `/admin/projects/delete-impact` | GET |
| 管理 🔓 | `/admin/projects/delete` | POST |
| 管理 🔓 | `/admin/projects/members` | GET |
| 管理 🔓 | `/admin/projects/members/set` | POST |
| 管理 🔓 | `/admin/projects/members/add` | POST |
| 管理 🔓 | `/admin/projects/members/remove` | POST |
| 管理 🔓 | `/admin/projects/projects-by-users` | GET |
| 管理 🔓 | `/admin/projects/config-overview` | GET |
| 管理 🔓 | `/admin/projects/instances` | GET |
| 管理 🔓 | `/admin/user-groups/instances` | GET |
| 管理 🔓 | `/admin/assets/detail` | GET |
| 管理 🔓 | `/admin/assets/candidates` | GET |
| 管理 🔓 | `/admin/assets/save` | POST |
| 管理 🔓 | `/admin/assets/versions` | GET |
| 用户 🔒 | `/projects/mine` | GET |
| 管理 🔓 | `/admin/instances/rules` | GET |
| 管理 🔓 | `/admin/roles` | GET |
| 管理 🔓 | `/admin/roles/create` | POST |
| 管理 🔓 | `/admin/roles/update` | POST |
| 管理 🔓 | `/admin/roles/delete` | POST |
| 管理 🔓 | `/admin/roles/toggle-visible` | POST |
| 管理 🔓 | `/admin/roles/reorder` | POST |
| 管理 🔓 | `/admin/roles/detail` | GET |
| 管理 🔓 | `/admin/roles/distribute` | POST |
| 管理 🔓 | `/admin/roles/instances` | GET |
| 管理 🔓 | `/admin/roles/records` | GET |
| 用户组管理 🔓 | `/admin/user-groups` | GET |
| 用户组管理 🔓 | `/admin/user-groups/create` | POST |
| 用户组管理 🔓 | `/admin/user-groups/update` | POST |
| 用户组管理 🔓 | `/admin/user-groups/delete` | POST |
| 用户组管理 🔓 | `/admin/user-groups/members` | GET |
| 用户组管理 🔓 | `/admin/user-groups/members/set` | POST |
| 用户组管理 🔓 | `/admin/user-groups/members/add` | POST |
| 用户组管理 🔓 | `/admin/user-groups/members/remove` | POST |
| 用户组管理 🔓 | `/admin/user-groups/groups-by-users` | GET |
| 用户组管理 | `/user-groups/mine` | GET |
| 插件管理 🔓 | `/admin/plugin-categories` | GET |
| 插件管理 🔓 | `/admin/plugin-categories/create` | POST |
| 插件管理 🔓 | `/admin/plugin-categories/update` | POST |
| 插件管理 🔓 | `/admin/plugin-categories/delete` | POST |
| 插件管理 🔓 | `/admin/plugins` | GET |
| 插件管理 🔓 | `/admin/plugins/create` | POST |
| 插件管理 🔓 | `/admin/plugins/update` | POST |
| 插件管理 🔓 | `/admin/plugins/delete` | POST |
| 插件管理 🔓 | `/admin/plugins/detail` | GET |
| 插件管理 🔓 | `/admin/plugins/files` | GET |
| 插件管理 🔓 | `/admin/plugins/tasks` | GET |
| 插件管理 🔓 | `/admin/plugins/instances` | GET |
| 插件管理 🔓 | `/admin/plugins/distribute` | POST |
| 插件管理 🔓 | `/admin/plugins/favorite` | POST |
| 插件管理 🔓 | `/admin/plugins/unfavorite` | POST |
| 插件管理 🔓 | `/admin/plugins/favorited` | GET |
| 插件管理 🔓 | `/admin/plugin-bundles` | GET |
| 插件管理 🔓 | `/admin/plugin-bundles/create` | POST |
| 插件管理 🔓 | `/admin/plugin-bundles/delete` | POST |
| 插件管理 🔓 | `/admin/plugin-bundles/toggle` | POST |
| 插件管理 🔓 | `/admin/plugin-bundles/detail` | GET |
| 插件管理 🔓 | `/admin/plugin-bundles/update-plugins` | POST |
| SMH 存储 🔓 | `/admin/smh/instances` | GET |
| SMH 存储 🔓 | `/admin/smh/personal-spaces` | GET |
| SMH 存储 🔓 | `/admin/smh/personal-spaces/token` | GET |
| SMH 存储 🔓 | `/admin/smh/stat` | GET |
| SMH 存储 🔓 | `/admin/smh/instance-space` | POST |
| SMH 存储 🔓 | `/admin/smh/personal-space-auto-provision` | POST |
| 标签管理 🔓 | `/admin/tags` | GET |
| 标签管理 🔓 | `/admin/tags/create` | POST |
| 标签管理 🔓 | `/admin/tags/update` | POST |
| 标签管理 🔓 | `/admin/tags/replace-all` | POST |
| 标签管理 🔓 | `/admin/tags/delete` | POST |
| 标签管理 🔓 | `/api/tags/keys` | GET |
| 标签管理 🔓 | `/api/tags/values` | GET |
| 实例管理 | `/openclaw/roles` | GET |
| 实例管理 | `/openclaw/remove-role` | POST |
| 实例管理 | `/openclaw/switch-role` | POST |
| 记忆管理 | `/openclaw/memory/library/detail` | GET |
| 管理 🔓 | `/admin/memory/overview` | GET |
| 管理 🔓 | `/admin/memory/pro/activate` | POST |
| 管理 🔓 | `/admin/memory/pro/release` | POST |
| 管理 🔓 | `/admin/memory/plan/switch` | POST |
| 管理 🔓 | `/admin/memory/instances` | GET |
| 管理 🔓 | `/admin/memory/default-plan` | GET/PUT |

> **注意：** `POST /admin/config/cvm` 通过 API Token 调用时，CVMSecretId、CVMSecretKey、CVMTemplate、PublicImageId 四个敏感字段会被静默忽略，仅可通过网页修改。

**路径前缀：** 本文档中所有接口路径均为应用层原始路由（如 `/openclaw/list`）。实际部署时，若使用了反向代理或网关，可能需要添加路径前缀（如 `/api`），此时实际请求路径为 `/api/openclaw/list`。请根据具体部署环境确认是否需要添加前缀。

## 响应格式

除少量特殊接口外，接口均返回 JSON 响应（详见概览中的废弃说明）。特殊接口：SSE 流（`GET /openclaw/auto-channel`、`POST /v1/chat/completions` 的 `stream: true` 模式）、图片（`GET /captcha/{id}.png`、`GET /logo`、`GET /favicon.ico`）、静态资源（`GET /static/*`）、WebSocket（`GET /openclaw/vnc-ws-proxy`）。

JSON 响应格式约定：

- 成功：`{"ok": true}` 或 `{"ok": true, "redirect": "/path"}`（需跳转时）或直接返回数据对象
- 失败：`{"error": "错误信息", "detail": "详细信息", "request_id": "xxxx"}`，HTTP 状态码反映错误类型，其中只有error是必定会存在的，其他两个字段为可选字段，request_id在调用腾讯API的时候会存在。

```json
{"error": "创建实例失败", "detail": "余额不足", "request_id": "abc-123-def"}
```

以下接口**始终返回 JSON**，不受 Accept Header 影响：

- `GET /openclaw/notifications`
- `POST /openclaw/notifications/read`
- `GET /openclaw/notifications/count`
- `POST /openclaw/notifications/delete`
- `POST /openclaw/denied-actions`
- `POST /openclaw/approve`
- `GET /openclaw/service-status`
- `GET /openclaw/check-openclaw-port`
- `POST /openclaw/set-model`
- `GET /openclaw/models`
- `POST /openclaw/set-channel`
- `POST /openclaw/del-channel`
- `GET /openclaw/channels`
- `GET /openclaw/skills`
- `GET /openclaw/auto-channel`（SSE）
- `POST /openclaw/terminal-url`
- `POST /openclaw/set-gateway-ui`
- `POST /openclaw/ws-url`
- `GET /openclaw/check-gateway-access`
- `POST /openclaw/set-env`
- `GET /openclaw/env`
- `GET /openclaw/smh-status`
- `GET /openclaw/smh-token`
- `GET /openclaw/memory-tdai-status`
- `POST /admin/instances/denied-actions`
- `GET /admin/departments`
- `GET /admin/cls/service`
- `GET /admin/cloud`
- `GET /admin/skill-categories`
- `POST /admin/skill-categories/create`
- `POST /admin/skill-categories/update`
- `POST /admin/skill-categories/delete`
- `GET /admin/skills`
- `POST /admin/skills/create`
- `POST /admin/skills/update`
- `POST /admin/skills/delete`
- `GET /admin/skills/references`
- `GET /admin/skills/detail`
- `GET /admin/skills/files`
- `GET /admin/skills/tasks`
- `GET /admin/skills/instances`
- `POST /admin/skills/instances`
- `POST /admin/skills/distribute`
- `POST /admin/skills/uninstall`
- `GET /admin/smhinfo`
- `POST /admin/smh/personal-space-auto-provision`
- `GET /admin/smh/instances`
- `POST /admin/smh/instance-space`
- `GET /admin/smh/personal-spaces`
- `GET /admin/smh/personal-spaces/token`
- `GET /admin/smh/stat`
- `GET /openclaw/install-skills`
- `POST /openclaw/retry-failed-skills`
- `POST /openclaw/cancel-failed-skills`
- `POST /admin/skill-bundles/create`
- `GET /admin/skill-bundles`
- `POST /admin/skill-bundles/delete`
- `POST /admin/skill-bundles/toggle`
- `GET /admin/skill-bundles/detail`
- `POST /admin/skill-bundles/update-skills`
- `POST /admin/skill-bundles/batch-add-skills`
- `POST /admin/skill-bundles/update-visibility`
- `POST /admin/skills/favorite`
- `POST /admin/skills/unfavorite`
- `GET /admin/skills/favorited`
- `POST /admin/skillsets/favorite`
- `POST /admin/skillsets/unfavorite`
- `GET /admin/skillsets/favorited`
- `POST /admin/skills/scan-trigger`
- `GET /admin/skills/scan-config`
- `POST /admin/skills/scan-config`
- `GET /admin/roles`
- `POST /admin/roles/create`
- `POST /admin/roles/update`
- `POST /admin/roles/delete`
- `POST /admin/roles/toggle-visible`
- `POST /admin/roles/reorder`
- `GET /admin/roles/detail`
- `POST /admin/roles/distribute`
- `GET /admin/roles/instances`
- `GET /admin/roles/records`
- `GET /openclaw/roles`
- `POST /openclaw/remove-role`
- `POST /openclaw/switch-role`
- `GET /openclaw/lightclaw/token`
- `POST /openclaw/lightclaw/auth`
- `POST /openclaw/lightclaw/describe-invocations`
- `POST /openclaw/lightclaw/describe-invocation-tasks`
- `POST /openclaw/lightclaw/run-command`

---

## 实例状态准入规则（写操作 409）

对实例执行写操作时，后端会先校验实例当前语义状态（与 `GET /openclaw/status` 返回的 `status` 一致）。非允许状态下返回 `HTTP 409`，`error` 字段为面向用户的中文文案，前端可直接 toast 展示。

**409 文案规则：**

| 当前状态 | error 字段内容 |
|----------|----------------|
| stopped | 实例已关机，请先开机并等待实例恢复运行中后再操作 |
| creating | 实例当前为创建中，请等待实例恢复运行中后再操作 |
| loading | 实例当前为加载中，请等待实例恢复运行中后再操作 |
| upgrading | 实例当前为升级中，请等待实例恢复运行中后再操作 |
| maintaining | 实例当前为维护中，请等待实例恢复运行中后再操作 |
| pending | 实例当前为待处理，请等待实例恢复运行中后再操作 |
| load_failed | 实例当前为加载失败，无法执行该操作 |
| create_failed | 实例当前为创建失败，无法执行该操作 |
| upgrade_failed | 实例当前为升级失败，无法执行该操作 |
| destroyed | 实例当前为已销毁，无法执行该操作 |

**适用范围：** 所有对实例下发指令、修改配置、改变状态的写接口。部分特殊恢复/清理类操作有各自的允许状态白名单（如 `retry` 仅允许 `load_failed`、`delete` 允许多个非过渡态、`start` 仅允许 `stopped`），具体见各接口文档。

---

## 一、认证相关

### `GET /`

首页 / 登录页。已登录用户重定向到 `/my-openclaw`，未登录用户显示登录表单。

- **权限：** 公开
- **JSON 响应：**
  - 已登录：`{"ok": true, "redirect": "/my-openclaw", "username": "张三", "role": "admin", "user_id": 1}`
  - 未登录：`{"authenticated": false}`（需要验证码时额外返回 `"captchaId": "xxx"`）

  已登录响应字段说明：

  | 字段 | 类型 | 说明 |
  |------|------|------|
  | ok | bool | 固定 `true` |
  | redirect | string | 重定向地址，固定 `/my-openclaw` |
  | username | string | 当前登录用户名 |
  | role | string | 用户角色，`"admin"` 或 `"user"` |
  | user_id | uint | 用户 ID |

### `POST /login`

用户登录。

- **权限：** 公开
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |
| captchaId | string | 条件 | 验证码 ID（登录失败后需要） |
| captchaValue | string | 条件 | 验证码值 |

- **JSON 响应：**
  - 成功：`{"ok": true, "redirect": "/", "role": "user"}` （`role` 为用户角色，值为 `"admin"` 或 `"user"`）
  - 失败：`401 {"error": "用户名或密码错误", "captchaId": "xxx"}` 或 `{"error": "验证码错误", "captchaId": "xxx"}`（每次失败都返回新的 `captchaId`）

### `POST /admin/passwordless-login-link`

为当前租户的指定本地用户签发两分钟有效、仅可成功使用一次的免登录跳转链接。
> **前置条件：** 使用本接口前，租户必须先申请并开通免登录功能白名单；未开通时，签发和消费接口均返回 `403`。

- **权限：** 管理员；与其它 `/admin` API 一致，支持管理员用户 API Token、管理员 Cookie Session 和进程 AdminToken
  - 缺失或错误的认证凭证返回 `401`
  - 已认证但角色不是管理员返回 `403`
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | uint | 是 | 当前租户中未删除的目标本地用户 ID，必须大于 0 |

- **请求示例：**

```json
{
  "user_id": 123
}
```

- **成功响应：**

```json
{
  "link": "https://tenant.example.com/passwordless-login#passwordless_login_token=<opaque-token>",
  "expires_in": 120,
  "expires_at": "2026-07-22T08:15:00Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| link | string | 当前租户域名下的 HTTPS 绝对链接；凭证只出现在 URL Fragment |
| expires_in | int | 固定 `120` 秒 |
| expires_at | string | RFC3339 格式的绝对过期时间 |

- **错误：**

| HTTP | 触发条件 | 说明 |
|------|----------|------|
| 400 | JSON 非法、缺少或非法 `user_id` | 不创建凭证 |
| 401 | 缺失或错误的认证凭证 | 未认证 |
| 403 | 调用者不是管理员，或当前租户未开通免登录功能白名单 | 不创建凭证 |
| 404 | 用户不存在或已被删除 | 不创建凭证 |
| 415 | `Content-Type` 不是 `application/json` | 不创建凭证 |
| 500 | 服务暂不可用 | 不返回可用链接 |

**安全约束：**

- 凭证是不透明字符串，调用方不得解析、记录或持久化。
- 同一用户可以同时持有多个未过期链接；每条链接独立过期、独立单次使用。
- 签发和消费均要求租户已开通免登录功能白名单；取消开通后，未使用的链接立即失效。

### `POST /auth/passwordless-login`

消费一次性凭证并建立 `hatchery-session`。请求前不要求已有登录态；一次性凭证本身是认证因子。

- **权限：** 公开
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| token | string | 是 | 从 `link` 的 `passwordless_login_token` Fragment 中提取的 43 字符不透明凭证 |

- **请求示例：**

```json
{
  "token": "<opaque-token>"
}
```

- **成功响应：** 同时设置 `hatchery-session` Cookie。

```json
{
  "ok": true,
  "redirect": "/",
  "role": "user"
}
```

- **错误：**

| HTTP | 触发条件 | 说明 |
|------|----------|------|
| 400 | JSON 非法、缺少凭证或凭证长度非法 | 凭证未消费 |
| 401 | 凭证伪造、过期、已消费、属于其它租户，或目标用户已删除 | 统一返回“免登录凭证无效或已过期”，不泄露具体状态 |
| 403 | 当前租户未开通免登录功能白名单 | 凭证未消费 |
| 415 | `Content-Type` 不是 `application/json` | 凭证未消费 |
| 500 | 服务暂不可用 | 不建立新登录态 |

消费成功后会覆盖浏览器已有登录态。前端必须：

1. 从 URL Fragment 读取凭证后立即通过 `history.replaceState` 清除 Fragment；
2. 仅在 POST JSON Body 中提交凭证，不得放入 Query、日志、DOM 或浏览器持久化存储；
3. 成功后执行完整页面刷新，由 `GET /` 重新同步当前用户；
4. 失败后不得自动重试一次性凭证。

### `GET /logout`

退出登录。清除本地 session，Gateway 模式下同步跳转 OneID 完成单点登出。

**⚠️ 必须使用 `window.location.href` 跳转，不能用 `fetch`/`axios`。**

- **权限：** 公开
- **行为说明：**
  1. 清除本地 session cookie
  2. 若配置了 Gateway（`GATEWAY_URL` 非空）：`302` 跳转到 oneid-gateway `/auth/logout?tenant={tenant_id}`，由 Gateway 构造 OneID 前端登出 URL，OneID 清除 SSO session 后跳回租户首页
  3. 若未配置 Gateway：直接 `302` 跳回 `/`
- **JSON 响应：**
  - 配置了 Gateway：`{"ok": true, "redirect": "https://oneid-gateway/auth/logout?tenant=..."}`
  - 未配置 Gateway：`{"ok": true, "redirect": "/"}`

### `POST /change-password`

提交修改密码。

- **权限：** 登录用户
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| old_password | string | 是 | 原密码 |
| new_password | string | 是 | 新密码 |
| captchaId | string | 是 | 验证码 ID |
| captchaValue | string | 是 | 验证码值 |

- **JSON 响应：**
  - **空参数调用（不传 `old_password` 和 `new_password`）：** 返回 `{"captchaId": "xxx"}`，用于 API 客户端获取验证码 ID
  - 成功：`{"ok": true}`（Session 同时被清除，需重新登录）
  - 失败：`400 {"error": "原密码错误"}` 等
  - 未登录：`401 {"error": "未登录"}`

> **API 客户端修改密码流程：**
>
> 1. `POST /login` 登录获取 Session Cookie
> 2. `POST /change-password`（空参数 + `Accept: application/json`）获取 `captchaId`
> 3. `GET /captcha/{captchaId}.png` 下载验证码图片
> 4. `POST /change-password` 带完整参数提交修改

### `GET /captcha/{id}.png`

获取验证码图片（由 `dchest/captcha` 库提供）。

- **权限：** 公开
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| reload | string | 否 | 传任意非空值（如 `1` 或时间戳）刷新验证码，为同一 `id` 重新生成数字。可用于"看不清？换一张"功能。建议每次传不同的值以避免浏览器缓存 |
| lang | string | 否 | 音频验证码语言（仅 `.wav` 格式有效），如 `ru`、`zh`。PNG 图片可忽略 |

- **响应：** PNG 图片（不支持 JSON），带 `Cache-Control: no-cache` Header
- **示例：**

```
GET /captcha/LBm5vMjHDtdUfaWYXiQX.png            # 获取验证码图片
GET /captcha/LBm5vMjHDtdUfaWYXiQX.png?reload=1    # 刷新验证码（重新生成数字）
```

> **注意：** 加 `?reload=x` 后会调用 `captcha.Reload(id)` 为同一 ID 重新生成随机数字，之前的验证码答案将失效。不加 `reload` 时，同一 ID 多次请求返回的图片内容完全相同。

### `GET /api-token`

查看当前用户的 API Token。

- **权限：** 登录用户
- **认证：** Session Cookie（需网页登录）
- **Content-Type：** 无需请求体
- **JSON 响应：**
  - 已有 Token：`{"token": "hk-...", "exists": true}`
  - 无 Token：`{"token": "", "exists": false}`
  - 未登录：`401 {"error": "未登录"}`

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| token | string | API Token，格式为 `hk-` 前缀；无 Token 时为空字符串 |
| exists | bool | `true` 表示已有 Token，`false` 表示尚未创建 |

### `POST /api-token/create`

创建用户的 API Token。若已有 Token 则返回错误。

- **权限：** 登录用户
- **认证：** Session Cookie（需网页登录）
- **Content-Type：** 无需请求体
- **JSON 响应：**
  - 成功：`{"token": "hk-..."}`
  - 已有 Token：`409 {"error": "Token 已存在，如需更换请使用重置功能"}`
  - 未登录：`401 {"error": "未登录"}`
  - 服务端错误：`500 {"error": "生成 Token 失败"}`

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| token | string | 新生成的 API Token |

### `POST /api-token/reset`

重置当前用户的 API Token。生成新 Token 并替换旧 Token，旧 Token 立即失效。

- **权限：** 登录用户
- **认证：** Session Cookie（需网页登录）
- **Content-Type：** 无需请求体
- **JSON 响应：**
  - 成功：`{"token": "hk-..."}`
  - 未登录：`401 {"error": "未登录"}`
  - 服务端错误：`500 {"error": "重置 Token 失败"}`

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| token | string | 新生成的 API Token |

### `POST /api-token/revoke`

销毁当前用户的 API Token。Token 立即失效，后续使用该 Token 的请求将返回 401。

- **权限：** 登录用户
- **认证：** Session Cookie（需网页登录）
- **Content-Type：** 无需请求体
- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 未登录：`401 {"error": "未登录"}`
  - 服务端错误：`500 {"error": "销毁 Token 失败"}`

> **注意：** 这三个接口仅支持 Session Cookie 认证，不支持 Bearer Token 调用。这是为了防止 Token 通过 API 被恶意替换或销毁。用户需要通过网页登录后操作自己的 Token。

---

### `GET /auth/oneid`

发起 OneID SSO 登录跳转（Gateway 模式）。浏览器访问此地址后经由 oneid-gateway 完成 OIDC 认证，认证成功后自动建立 session 并跳转到应用首页。

**⚠️ 必须使用 `window.location.href` 跳转，不能用 `fetch`/`axios`。**

- **权限：** 公开
- **Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| acr_values | string | 否 | 指定 OneID 登录页展示的认证方式，多个值以空格分隔（需 URL encode 为 `%20`）。不传则使用平台全局配置，平台未配置则由 OneID 租户后台默认配置决定 |
| idp | string | 否 | 直接指定 OneID 认证源 `aliasId`，跳过 OneID 中间选择页直达对应 IdP 登录页（如飞书、企微）。`aliasId` 通过 `GET /auth/sso-providers` 获取 |

`acr_values` 可选值：

| 值 | 效果 |
|----|------|
| sms | 手机短信验证码 |
| email | 邮箱验证码 |
| sso:preferred_email | 企业 SSO（按邮箱域名匹配认证源） |
| sso:preferred_domain | 企业 SSO（按企业域名匹配认证源） |

- **响应：**
  - 成功：`302` 跳转至 oneid-gateway `/auth/oneid?tenant=xxx`，后续由 Gateway 完成 OIDC 流程，最终落地到 `/auth/internal-login` 建立 session 后跳转 `/openclaw`
  - Gateway 未配置：`400` 文本错误 `Gateway 未配置，请联系管理员`
  - 租户 ID 未找到：`400` 文本错误 `OneID 租户未配置，无法跳转 Gateway`

- **完整调用示例：**

```javascript
// 使用平台默认配置
window.location.href = '/auth/oneid'

// 强制邮箱验证码
window.location.href = '/auth/oneid?acr_values=email'

// 手机 + 邮箱 + 企业SSO 三种方式
window.location.href = '/auth/oneid?acr_values=sms%20email%20sso%3Apreferred_email'

// 直达指定 IdP（飞书/企微等），跳过 OneID 中间选择页
// aliasId 通过 GET /auth/sso-providers 获取
window.location.href = '/auth/oneid?idp=17opbbpf8pj8m'
```

### `GET /auth/sso-providers`

返回当前租户在 OneID 上配置的认证方式，包括 SSO 认证源列表（所有 `aliasId` 非空的可直达 IdP，含飞书、企业微信、OpenLDAP 等）和密码登录开关。前端打开登录弹窗时调用，用于渲染 SSO 卡片和判断是否展示密码登录入口。点击 SSO 卡片后带 `idp=<alias_id>` 参数跳转到 `/auth/oneid` 实现直达对应认证源。

调用链路：Hatchery → Gateway `/api/sso-providers` → OneID `/v1/authn/sso`（Hatchery Pod 不直连 OneID）。

- **权限：** 公开（无需登录）
- **Query 参数：** 无
- **响应：** `200`

```json
{
  "providers": [
    {
      "id": "1433036790747155734",
      "name": "飞书",
      "logo_url": "https://identity.tencent.com/permanent/account_user_logo/...",
      "alias_id": "17opbbpf8pj8m",
      "unique_name": "FEISHU_V0",
      "flow_type": "Redirect"
    },
    {
      "id": "1453193256183302593",
      "name": "OpenLDAP",
      "logo_url": "https://identity.tencent.com/permanent/account_user_logo/...",
      "alias_id": "18am7i4ocp2e1",
      "unique_name": "OPENLDAP_V1",
      "flow_type": "Delegation"
    }
  ],
  "password_enabled": true
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| providers | object[] | SSO 认证源列表，包含所有 `aliasId` 非空的可直达 IdP（飞书、企微、OpenLDAP 等）。`PASSWORD_V0` 等 `aliasId` 为空的项不包含在此列表 |
| providers[].id | string | OneID 认证源唯一 ID。同一类型（如飞书）可配置多个认证源，`unique_name` 会重复，需用此 `id` 作为列表渲染 key 和唯一标识 |
| providers[].name | string | IdP 显示名称（如「飞书」），可能为空，前端可用 `unique_name` 兜底 |
| providers[].logo_url | string | IdP 图标 URL，可能为空，前端可用 `unique_name` 兜底内置图标 |
| providers[].alias_id | string | 传给 `/auth/oneid?idp=` 的标识 |
| providers[].unique_name | string | OneID 内部唯一名（如 `FEISHU_V0`、`WECOM_V0`、`OPENLDAP_V1`），同类型多认证源时会重复 |
| providers[].flow_type | string | 认证流程类型：`Redirect`（跳转外部 IdP 登录页，如飞书）/ `Delegation`（在 OneID 页面内完成，如 OpenLDAP 账密委托）。前端跳转动作两者一致，差异由 OneID 在 `?idp=` 后处理 |
| password_enabled | bool | 是否配置了密码登录（OneID 返回的 `uniqueName=PASSWORD_V0`），前端据此决定是否展示密码登录入口 |

容错策略：当 OneID 接口失败、Gateway 不可用、租户未配置 OneID 时，统一返回 `{"providers":[],"password_enabled":false}`。保守关闭密码登录入口，避免在 OneID 未真正配置密码登录的情况下误导用户。

- **典型用法：**

```javascript
// 弹窗打开时拉取认证源列表
const { providers, password_enabled } = await fetch('/api/auth/sso-providers').then(r => r.json())

// 渲染 SSO 卡片，点击直达 IdP
providers.forEach(idp => {
  showCard(idp.name, idp.logo_url, () => {
    window.location.href = `/auth/oneid?idp=${encodeURIComponent(idp.alias_id)}`
  })
})

// 根据 password_enabled 决定是否展示密码登录入口
if (password_enabled) {
  showPasswordLoginEntry()
}
```

### `GET /auth/oneid-code`

IDP 发起登录（IDP-initiated login）的入口接口。当页面内嵌了 OneID 的 `select_account` 组件，组件在用户认证完成后会直接将 `code` 返回给前端 JS（而不是通过浏览器跳转），前端需将 code 转发给此接口以完成登录流程。

- **权限：** 公开
- **Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| code | string | 是 | OneID `select_account` 组件返回的授权码 |

- **响应：**
  - 成功：`302` 跳转至 oneid-gateway `/auth/idp-callback?code=xxx&tenant=xxx`，后续由 Gateway 完成 token 交换，最终落地到 `/auth/internal-login` 建立 session 后跳转 `/openclaw`
  - `code` 参数缺失：`400`
  - Gateway 未配置：`400` 文本错误 `Gateway 未配置，请联系管理员`
  - 租户 ID 未找到：`400` 文本错误 `OneID 租户未配置，无法跳转 Gateway`

- **完整调用示例：**

```javascript
// OneID select_account 组件回调示例
// 组件返回 { code: "xxx", client_id: "xxx" }
oneIDComponent.on('success', ({ code }) => {
  window.location.href = `/auth/oneid-code?code=${encodeURIComponent(code)}`
})
```

- **与 `/auth/oneid` 的区别：**

| | `/auth/oneid` | `/auth/oneid-code` |
|---|---|---|
| 触发方式 | 前端主动跳转，浏览器经由 Gateway → OneID 完整 OIDC 流程 | OneID 嵌入组件将 code 返回给 JS，再由前端调用 |
| CSRF 保护 | 有（state cookie） | 无（code 由 OneID 组件直接颁发给当前浏览器会话，等价保护） |
| 适用场景 | 独立登录页、登录按钮点击 | 嵌入 OneID 选账组件（`select_account`） |

---

### `GET /auth/oneid/jump`

管理员跳转到 OneID 管理后台/个人中心/工作台的免登链接接口。通过 oneid-gateway 的 `/api/sso-link` 代理获取带时效的免登链接，返回给前端供跳转。

- **权限：** 需登录且 role = admin
- **Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| module | string | 否 | 跳转目标模块，默认 `admin`，可选值：`admin`（管理后台）、`portal`（工作台）、`profile`（个人中心） |
| route | string | 否 | 模块内路由路径，如 `users`、`departments` |

- **响应：**
  - 成功：`200` `{"link": "https://...", "expires_in": 300}`
  - 未登录：`401`
  - 非管理员：`403`
  - Gateway 未配置：`503`
  - 用户未关联 OneID：`400`

---

### `GET /auth/internal-login`

### `GET /auth/oneid-register`

OneID 注册回调接口。前端无需直接调用，由 OneID 登录流程自动触发。行为与 `GET /auth/oneid-code` 类似，将 `code` 参数转发给 Gateway 的 `/auth/idp-callback` 接口。

- **权限：** 公开（由 OneID 回调）
- **Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| code | string | 是 | OneID 授权码 |

- **响应：** `302` 重定向到 Gateway `/auth/idp-callback`
- **失败：** `400`（缺少 code 参数或 Gateway 未配置）

### `GET /auth/internal-login`

Gateway 认证完成后的落地接口，由 oneid-gateway 重定向调用，**前端无需直接调用**。

- **权限：** 公开（内部，由 oneid-gateway 调用）
- **Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| token | string | oneid-gateway 签发的 HMAC-SHA256 内部令牌 |

- **响应：**
  - 成功：建立 session，`302` 重定向到 `/openclaw`，写入审计日志（action=`internal_login`）
  - token 无效/过期：`401`
  - `INTERNAL_SECRET` 未配置：`500`

- **管理员角色实时同步：** 若启动参数 `--oneid-admin-role-id` 或环境变量 `ONEID_ADMIN_ROLE_ID` 不为空，每次登录会调用 OneID `POST /openapi/v3/permissions/roles/{role_id}/has_member` 接口实时查询当前用户是否属于指定角色，并同步更新本地 `role`（`admin` / `user`）。

### `POST /spi/logout`

接收 OneID 被动登出通知（由 oneid-gateway 转发）。

- **权限：** 公开（内部服务调用）
- **Content-Type：** `application/json`
- **请求体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| sub | string | OneID 用户唯一标识 |
| enterprise_id | string | 企业 ID |

- **响应：** `200 {"ok": true}`
- **说明：** 本期不强制清除 session（CookieStore 限制），仅记录日志。写入审计日志（action=`oneid_logout`）。

### `POST /spi/event`

接收 OneID 组织架构 Webhook 事件推送。

- **权限：** 公开（内部服务调用）
- **Content-Type：** `application/json`
- **请求体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | 事件类型，见下表 |
| sub | string | OneID 用户唯一标识 |
| enterprise_id | string | 企业 ID |
| name | string | 用户显示名（member.created / member.updated） |
| asset_action | string | 资产处置方式：`keep`（默认）/ `delete` / `transfer`（member.deleted） |
| transfer_to_sub | string | 转移目标用户 sub（asset_action=transfer 时有效） |

支持的事件类型：

| type | 说明 |
|------|------|
| member.created | 成员入职，创建本地用户（已存在则跳过） |
| member.updated | 成员信息变更，更新用户名 |
| member.deleted | 成员离职，按 asset_action 处理资产并软删除用户 |
| admin.added | 将用户 role 更新为 `admin` |
| admin.removed | 将用户 role 更新为 `user` |
| 其他 | 记录 WARN 日志，返回 200（不影响 OneID 重试） |

- **响应：** `200 {"ok": true}`
- **审计：** 写入审计日志（action=`oneid_event`）

### `POST /admin/oneid-sync-enterprise`

从 OneID 拉取企业信息（名称和 Logo），覆写本地 SiteConfig 的 `name`（网站名称）、`logo`/`logo_mime`（网站 Logo 二进制）。

- **权限：** 管理员
- **请求体：** 无
- **响应：** `200 {"ok": true}`
- **说明：**
  - 通过 OneID 开放平台 `GET /openapi/v3/accounts/{enterprise_id}` 拉取企业信息
  - 企业名称非空时覆写 `SiteConfig.Name`
  - 企业 Logo URL 非空时下载图片二进制存入 `SiteConfig.Logo` / `SiteConfig.LogoMIME`，之后 `GET /logo` 直接返回该二进制，无需依赖外部 URL
  - 依赖 `SiteConfig.OneIDAPIBaseURL`、`OneIDClientID`、`OneIDPrivateKeyJWK` 配置正确

### `POST /admin/oneid-sync-users`

手动触发 OneID 全量用户同步。拉取 OneID 通讯录中的全量部门和成员，自动创建/更新/禁用/删除本地用户，并同步用户画像和管理员角色。

- **权限：** 管理员
- **请求体：** 无
- **响应：** `200`

```json
{
  "ok": true,
  "message": "同步完成",
  "profile_count": 11,
  "dept_count": 2,
  "user_count": 7,
  "affected_users": [
    {
      "username": "张三",
      "instance_count": 3,
      "action": "disabled",
      "vpc_id": "vpc-abc123",
      "vpc_has_resources": true
    },
    {
      "username": "李四",
      "instance_count": 0,
      "action": "hard_deleted",
      "vpc_id": null
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| profile_count | number | 同步后的用户画像总数 |
| dept_count | number | 同步后的部门总数 |
| user_count | number | 同步后的活跃用户总数（不含已禁用） |
| affected_users | array \| null | 本次同步中被禁用或删除的用户列表，无受影响用户时为 `null` |

`affected_users` 数组中每个元素：

| 字段 | 类型 | 说明 |
|------|------|------|
| username | string | 用户名 |
| instance_count | number | 该用户名下的 OpenClaw 实例数量 |
| action | string | 处理方式：`disabled`（软删除/禁用）或 `hard_deleted`（物理删除） |
| vpc_id | string \| null | 用户关联的 VPC ID，无 VPC 时为 `null` |
| vpc_has_resources | boolean | VPC 下是否仍有资源占用（仅 `vpc_id` 非空时有意义） |

- **前端弹框逻辑：**
  - `affected_users` 为 `null` 或空数组 → **不弹框**
  - `affected_users` 中所有用户的 `instance_count` 均为 `0` 且 `vpc_has_resources` 均为 `false` → **不弹框**（用户已被彻底清理或无资源，无需人工介入）
  - `affected_users` 中存在 `instance_count > 0` 或 `vpc_has_resources == true` 的用户 → **弹框提示**，展示有资源残留的用户列表，提醒管理员手动处理

- **处理规则说明：**

| OneID 侧状态 | 本地无资源 | 本地有资源 |
|-------------|-----------|-----------|
| 已删除（不在通讯录中） | `hard_deleted`（物理删除） | `disabled`（软删除） |
| 已停用（Suspended/Disabled/LockedOut） | `disabled`（软删除） | `disabled`（软删除） |

- **安全保护：** 若本次同步中要删除/禁用的用户数超过本地 OneID 用户总数的 50%，视为数据源异常，自动禁止硬删除，仅做软删除（可恢复）

### `GET /admin/oneid-sync-users/status`

查询 OneID 用户同步的当前状态。

- **权限：** 管理员
- **参数：** 无
- **响应：** 始终 JSON

```json
{
  "running": false,
  "last_sync": "2026-04-20T10:30:00+08:00",
  "profile_count": 11,
  "dept_count": 2,
  "oneid_user_count": 7
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| running | bool | 是否正在同步中 |
| last_sync | string | 上次同步完成时间（ISO 8601），未同步过时为空字符串 |
| profile_count | int | 已同步的 OneID 用户画像总数 |
| dept_count | int | 已同步的部门总数 |
| oneid_user_count | int | 已关联 OneID 的本地用户数 |

---

## 二、实例管理（OpenClaw）

### `POST /openclaw/denied-actions`

批量查询当前用户的 claw 实例对应 CVM 的禁用操作。仅返回 `DescribeInstanceVncUrl` 相关的 DeniedAction，用于前端判断实例是否可打开终端。

- **权限：** 登录用户（仅可查询自己的实例）
- **Content-Type：** `application/json`
- **响应：** 始终返回 JSON
- **请求体：**

```json
{"ids": [1, 2, 3]}
// 或使用 instance_ids
{"instance_ids": ["ins-aaa", "ins-bbb"]}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | uint[] | 条件 | claw 实例数据库 ID 列表；与 `instance_ids` 二选一，`ids` 优先 |
| instance_ids | string[] | 条件 | CVM 实例 ID 列表；与 `ids` 二选一 |

> `ids` 与 `instance_ids` 至少传一个，同时存在时以 `ids` 为准。

- **成功响应：**

```json
{
  "instances": [
    {
      "id": 1,
      "denied_actions": [
        {
          "action": "DescribeInstanceVncUrl",
          "code": "UnsupportedOperation.InstanceStateStopped",
          "message": "不支持`已关机`的实例 (b2a82b28)"
        }
      ]
    },
    {
      "id": 2,
      "denied_actions": []
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| instances[].id | uint | claw 实例数据库 ID |
| instances[].denied_actions | array | 被禁用的操作列表（仅 `DescribeInstanceVncUrl`），为空数组表示无限制 |
| instances[].denied_actions[].action | string | 操作名称，固定为 `DescribeInstanceVncUrl` |
| instances[].denied_actions[].code | string | 禁用原因错误码 |
| instances[].denied_actions[].message | string | 禁用原因描述 |

失败时返回错误信息：

- `401 {"error": "未登录"}`
- `405 {"error": "Method not allowed, use POST"}`
- `400 {"error": "请求体格式错误，需要 JSON {\"ids\": [...]} 或 {\"instance_ids\": [...]}"}`
- `500 {"error": "获取云 API 凭证失败: ..."}`
- `500 {"error": "查询实例禁用操作失败: ..."}`

注意：当不提供 `ids` 和 `instance_ids` 参数时，返回 `200 {"instances": []}`（空结果）而不是错误。

### `GET /openclaw/list`

获取当前用户的实例列表，JSON 模式支持分页。

- **权限：** 登录用户
- **参数（仅 JSON 模式生效）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 否 | 实例主键 ID，传入时按 ID 精准搜索（优先级最高） |
| instance_id | string | 否 | CVM 实例 ID，传入时按 instance_id 精准搜索（id 不为 0 时忽略此参数） |
| keyword | string | 否 | 模糊搜索关键词，命中 `name` 或 `instance_id`（任一），最多 50 字符。`id` / `instance_id` 任一存在时此参数被忽略 |
| agent_type | string | 否 | 按智能体类型过滤，逗号分隔多值（OR），如 `openclaw,hermes` |
| page | int | 否 | 页码，默认 `1` |
| page_size | int | 否 | 每页条数，默认 `30`，最大 `100` |

- **JSON 响应：**

```json
{
  "instances": [...],
  "total": 50,
  "page": 1,
  "page_size": 30,
  "total_pages": 2
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| instances | array | 实例对象数组（当前页） |
| total | int | 用户实例总数 |
| page | int | 当前页码 |
| page_size | int | 每页条数 |
| total_pages | int | 总页数（向上取整） |

**实例对象新增字段（原有字段保留）：**

```typescript
// 新增字段（用于前端首屏预判，减少 /status 调用）
CurrentOperation: '' | 'create' | 'reboot' | 'reinstall' | 'delete';
CurrentOperationState: '' | 'processing' | 'success' | 'failed';
CurrentOperationUpdatedAt: string | null;  // ISO 8601 时间字符串
LastStableState: '' | 'RUNNING' | 'STOPPED';
LastCVMState: string;  // 最近一次 CVM 状态
instance_charge_type: string; // CVM 实例实际计费类型
os_name: string;       // CVM 实例的操作系统名称；本地实例为空字符串
AgentReady: number;    // 0 | 1
RoleID: number;        // 关联角色 ID，0 = 通用助手
role_name: string;     // 角色名称（"通用助手" / "行业分析师" 等），由后端批量查询填充
model_provider: string; // 模型提供商，仅当 AIModelID > 1 时返回
model_name: string;     // 模型名称，仅当 AIModelID > 1 时返回
agent_type: string;     // 智能体类型（内置：openclaw/hermes/lightclawace；也可能是自定义类型 code）
agent_version: string;  // 智能体版本号
light_claw_user_id: string; // LightClaw 用户 ID，用于 LightClaw 对接
group_id: number;       // 实例绑定的分组 ID，0 = 未指定
group_full_path: string; // 实例绑定分组的完整路径，group_id=0 时为空串

// 🆕 本地 agent 实例相关字段（由 clawpro 接管本地 agent 一期引入）
source: 'cvm' | 'local';        // 实例来源；CVM 实例为 'cvm'，本地 agent 实例为 'local'
host_name: string;              // local 实例主机名（如 'alex-mbp'）；CVM 实例为空字符串
os: string;                     // local 实例的 OS（如 'darwin/arm64'/'linux/amd64'/'windows/amd64'）；CVM 实例为空字符串
last_report_at: string | null;  // local 实例最近一次 reporter 上报时间（RFC3339）；CVM 实例为 null

// 🆕 二期：本地 agent 资源视图（仅 source='local' 且精准查询 id 时返回）
local_agent_resources: {
  user_level: {
    group_id: number;
    group_name: string;
    group_active: boolean;       // 用户是否仍在该分组（被移出则 false）
    skills: Array<{
      slug: string;
      version: string;
      display_name: string;
      source: 'enterprise' | 'public' | 'local';
      install_status: 'distributed' | 'distributing' | 'failed';
    }>;
    rules: Array<{               // 用户级企业规范（scope=user）
      slug: string;
      version: string;
      display_name: string;
      type: 'prompt' | 'rule';
      source: 'enterprise' | 'local';
      install_status: 'distributed' | 'distributing' | 'failed';
    }>;
  };
  workspaces: Array<{
    path: string;
    name: string;
    ide_type: string;
    project_id: number;
    skills: Array<{
      slug: string;
      version: string;
      display_name: string;
      source: 'enterprise' | 'public' | 'local';
      install_status: 'distributed' | 'distributing' | 'failed';
    }>;
    rules: Array<{
      slug: string;
      version: string;
      display_name: string;
      type: 'prompt' | 'rule';
      source: 'enterprise' | 'local';
      install_status: 'distributed' | 'distributing' | 'failed';
    }>;
  }>;
} | null;  // CVM 实例或非精准查询时为 null
```


**`source` 取值（🆕）：** `cvm` / `local`。

**`status` 在本地实例（`source='local'`）上的取值（🆕）：** 与 CVM 实例复用同一套状态枚举，本地实例只会出现 `running`（reporter 心跳活，最近一次 `last_report_at` 距今未超过阈值）和 `stopped`（reporter 心跳丢失，机器关机/网断/进程挂）两种值。语义提醒：本地实例的 `running` 仅表示心跳活着，hatchery 无法主动启停本地机器；`stopped` 状态下 admin 端不会出现 `start`/`reboot`/`reinstall` 等动作（actions 列表会按本地实例特性裁剪，仅保留 `delete`）。

**智能体类型字段说明（🆕 新增）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| agent_type | string | 智能体类型：内置 `openclaw`/`hermes`/`lightclawace`，或管理员通过 `/admin/agent-types/create` 创建的自定义类型 code |
| agent_version | string | 智能体版本号（如 `2026.4.11`、`0.9.0`）；自定义类型可为空 |

> **使用场景**：前端可根据 `agent_type` 对实例列表进行分组展示，并根据类型判断实例支持的功能（如模型配置、通道配置、技能安装等仅 OpenClaw 类型支持）。自定义类型若声明了 `compatible_with`，其能力位等同于兼容目标；未声明 `compatible_with` 的自定义类型仅支持最小操作集（不支持模型/通道/技能/插件/Chatbot/SMH/Memory/Reinstall/BrowserVNC/Approve）。

**存量实例分组归属处理字段（stale-instances v1.0，与 `/admin/instances` 同构）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| flags | string[] | 实例标记数组（`stale_group` / `pending_user_action` / `allow_migrate` / `allow_same_group_handover`）；无标记时为 `[]` |
| handover_target_user_id | uint | 同组移交目标用户 ID，`0` = 无 pending 移交 |
| handover_rejected_by_user_id | uint | 最近一次拒绝移交的用户 ID，`0` = 无 |
| handover_initiated_at | string \| null | 移交发起时间（RFC 3339），`null` = 未发起 |

**模型信息字段说明（新增）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| model_provider | string | 模型提供商，仅当 `AIModelID > 1` 时返回（预置模型） |
| model_name | string | 模型名称，仅当 `AIModelID > 1` 时返回（预置模型） |

> **使用场景**：当管理员修改模型可见范围后，用户已应用的模型可能不在其 `/openclaw/models` 返回的可见列表中。此时前端无法从模型列表中匹配到已应用模型的信息。通过在实例列表中直接返回模型信息，前端可以正确显示"已应用模型"，不受可见性过滤影响。
>
> **判断逻辑**：
> - `AIModelID = 0`：自定义模型（旧逻辑），前端从 `CustomModelConfig` 解析
> - `AIModelID = 1`：`hatchery/custom` 内置记录，前端从 `CustomModelConfig` 解析
> - `AIModelID > 1`：预置模型，使用后端返回的 `model_provider`、`model_name`

**前端首屏预判策略：**

| 条件 | 前端行为 |
|------|--------|
| `CurrentOperation != ""` | 立即轮询 `/status` |
| `CurrentOperation=="" + AgentReady==1 + LastCVMState=="RUNNING"` | 先显示运行中，用户刷新时再查 |
| 其他 | 调一次 `/status` |

> 预判仅优化首屏性能，不替代 `/status` 的准确判断。

**`local_agent_resources` 字段（🆕 二期）：** 仅当 `source='local'` 且精准查询（传 `id` 参数）时返回，其他情况为 `null`。包含用户级 + 项目级的 skill 和 rule 安装状态。

| 字段 | 类型 | 说明 |
|------|------|------|
| user_level.group_id | number | 服务端维护的用户级主分组 ID；TeamAI 上报值不用于切换 |
| user_level.group_name | string | 分组名 |
| user_level.group_active | boolean | 用户是否仍在该分组 |
| user_level.skills[] | array | 用户级已装 skill 列表 |
| user_level.rules[] | array | 用户级企业规范列表（scope=user，含下发中/失败状态） |
| workspaces[] | array | 项目级 workspace 列表 |
| workspaces[].skills[] | array | 项目级已装 skill 列表 |
| workspaces[].rules[] | array | 项目级企业规范列表（scope=workspace） |

**`install_status` 取值：**

| 状态 | 含义 |
|------|------|
| `distributed` | 已下发（ack 成功 / report 上报已装） |
| `distributing` | 下发中（pending record 已写，agent 未 ack） |
| `failed` | 下发失败（agent ack 失败） |

> 详细接口文档见本文“本地 agent reporter 接入（clawpro 一期）”章节。

### `POST /openclaw/local/user-group`（🆕 二期）

前端切换用户级分组，触发 diffAndQueue 下发新分组技能。

- **权限：** 登录用户
- **请求体：**

```json
{
  "group_id": 2,
  "instance_id": 1
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_id | number | 是 | 切换到的新分组 ID |
| instance_id | number | 是 | 实例主键 ID |

- **响应：**

```json
{
  "ok": true,
  "new_pending_count": 3
}
```

> 行为：清理旧分组 distributing/failed 记录 → 算 catalog − installed 差集 → 写 pending record。

### `POST /local-agent/report`（本地 Agent Reporter）

本地 agent 上报实例状态、用户级资源与 Workspace 项目级资源。

- **权限：** 登录用户（Session Cookie 或 API Token）
- **请求体：** 见本文“本地 agent reporter 接入（clawpro 一期）”中的 report 接口说明
- **响应：**

```json
{
  "ok": true,
  "instance_id": "local-workbuddy-456789",
  "instance_pk": 1,
  "received_at": "2026-07-05T10:00:01+08:00",
  "rules_synced": 1,
  "user_level_synced": 1,
  "project_synced": 1
}
```

### `POST /local-agent/sync`（本地 Agent Reporter）

本地 agent 拉取待执行命令（skill + rule 合并返回）。三期新增 `cmds` 统一字段列表，与 `commands` 并存（数据一致），供新版本 reporter 使用。

- **权限：** 登录用户（Session Cookie 或 API Token）
- **请求体：** 见本文“本地 agent reporter 接入（clawpro 一期）”中的 sync 接口说明
- **响应：**

```json
{
  "ok": true,
  "commands": [
    {"id": 11, "type": "install_skill", "skill_slug": "...", "skill_version": "...", "scope": "user"},
    {"id": 21, "type": "install_prompt_rule", "rule_slug": "...", "rule_version": "...", "rule_type": "prompt", "scope": "workspace", "workspace_path": "/Users/alice/code/repo-x", "project_id": 101},
    {"id": 303, "type": "install_hook_rule", "rule_slug": "hook-abc", "rule_version": "1.0.0", "rule_type": "hook", "event": "SessionStart", "cmd": "echo hello", "scope": "user"},
    {"id": 456, "type": "uninstall_teamai", "cmd": "teamai uninstall --force --agent codebuddy"},
    {"id": 501, "type": "execute_agent_task", "agent_type": "codebuddy", "project_id": 101, "workspace_path": "/Users/alice/code/repo-x", "prompt": "修复登录失败问题"}
  ],
  "cmds": [
    {"id": 11, "type": "install_skill", "slug": "...", "version": "...", "scope": "user"},
    {"id": 21, "type": "install_prompt_rule", "slug": "...", "version": "...", "handle_type": "prompt", "scope": "workspace", "workspace_path": "/Users/alice/code/repo-x", "project_id": 101},
    {"id": 303, "type": "install_hook_rule", "slug": "hook-abc", "version": "1.0.0", "handle_type": "hook", "event": "SessionStart", "cmd": "echo hello", "scope": "user"},
    {"id": 456, "type": "uninstall_teamai", "cmd": "teamai uninstall --force --agent codebuddy"},
    {"id": 501, "type": "execute_agent_task", "agent_type": "codebuddy", "project_id": 101, "workspace_path": "/Users/alice/code/repo-x", "prompt": "修复登录失败问题"}
  ]
}
```

> 三期协议演进：`commands[]` 保持现状（异构数组，skill 字段前缀 `skill_`，rule 字段前缀 `rule_`），供老版本 reporter 使用；新增 `cmds[]` 为统一字段列表（所有类型共享 `slug` / `version` / `handle_type` / `event` / `cmd`，按需要 omitempty）。两列表数据完全一致。`type` 区分具体操作（含本期新增的 `install_hook_rule` / `uninstall_hook_rule` / `uninstall_teamai`）。老 reporter 全量升级后 `commands` 移除。

> 本地 Agent 下发只读取资产绑定，不读取可见范围：`scope=user` 只使用用户当前分组（含祖先继承）绑定的技能和规范；`scope=workspace` 只使用 Workspace 项目直接绑定的技能和规范。**但 report 以本地 Agent 上报为安装快照真相**：无论上报资源是否属于当前分组/项目资产，均按其 `user` 或 `workspace` scope 写入 `local_instance_skills` / `local_instance_rules`；资产绑定只影响后续下发命令，不会过滤上报事实。
>
> Workspace 可在 report/sync 的 `workspaces[]` 中携带可选 `project_id`。缺失该字段时保持现有绑定；传 `0` 显式解除项目；传非零值时必须属于当前用户。report 只写本地事实快照，紧随其后的 sync 才按项目直接绑定资产生成待执行命令；用户本地删除的已绑定资产会在 sync 再次下发。项目同步命令额外返回 `scope: "workspace"`、`workspace_path` 与 `project_id`，只包含该项目直接绑定的技能和规范；已删除项目保留原 ID 与已装资源快照，不报错也不再返回项目资产。

### `POST /local-agent/commands/ack`（本地 Agent Reporter）

agent 回报任务执行结果。`type` 字段区分 skill / rule 记录。

- **权限：** 登录用户（Session Cookie 或 API Token）
- **请求体：**

```json
{
  "id": 11,
  "type": "install_skill",
  "status": "success",
  "error": "",
  "version": "1.0.0"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | number | 是 | = sync 返回的 command.id |
| type | string | 是 | = sync 返回的 command.type，空串向后兼容按 skill 处理 |
| status | string | 是 | 资产命令使用 `success` / `failed`；`execute_agent_task` 还支持 `running` |
| error | string | 否 | failed 时填 |
| version | string | 否 | success 时回报实际版本 |
| result | string | 否 | `execute_agent_task` 截至当前的完整输出快照，最多 2 MiB；重复上报会覆盖旧快照，避免网络重试导致内容重复 |
| session_id | string | 否 | 本地 Agent 会话标识，便于后续继续对话 |

> `type` 取值：`install_skill` / `uninstall_skill` / `install_prompt_rule` / `install_rule_rule` / `uninstall_prompt_rule` / `uninstall_rule_rule` / `install_hook_rule` / `uninstall_hook_rule` / `uninstall_teamai` / `execute_agent_task` / `""`
>
> 三期新增类型语义：
> - `install_hook_rule` / `uninstall_hook_rule`：企业规范库 Hook 资源的安装/卸载，走 rule ack 管道（ack `id` = rule_distribution_records.id）。`install_hook_rule` 的 `version` 回报实际 hook 版本。
> - `uninstall_teamai`：移除本地 Agent（卸载 clawpro-teamai 插件 + 解绑）。ack `id` = local_agent_tasks.id；`success` 后服务端软删该本地实例（关联 skill/rule 数据保留）；`failed` 保留任务可重试。
> - `execute_agent_task`：TeamAI/Edge Runtime 领取任务后先回报 `running`，执行期间可多次回报截至当前的完整 `result` 快照，完成后回报 `success` 或 `failed`。ClawPro 通过 `GET /agent-tasks` 轮询状态与结果。

### `POST /agent-tasks/create`（ClawPro → 本地 Agent）

在 ClawPro 创建本地 Agent 执行任务。目标工作区必须已经由该本地 Agent 上报，并绑定到当前用户所属项目；服务端不会向未登记的任意本地路径下发任务。

- **权限：** 登录用户（仅能选择自己的 `source=local` 实例）
- **Content-Type：** `application/json`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_id | uint | 是 | 目标本地 Agent 的数据库实例 ID |
| project_id | uint | 否 | 项目 ID；传入时必须与工作区绑定一致，不传时由绑定关系确定 |
| workspace_path | string | 是 | 已由 TeamAI 上报并绑定项目的工作区绝对路径 |
| prompt | string | 是 | 要交给本地 Agent 执行的任务描述，最多 32000 个字符 |
| delivery_mode | string | 否 | `wss` 使用常驻通道即时唤醒；`hook` 由旧 Hook 触发；`poll` 仅等待下次同步。默认 `poll` |

当 `delivery_mode=wss` 时，服务端先持久化任务，再向目标实例的在线 TeamAI 连接发送仅包含 `task_id` 的 `task_available`。响应顶层 `wake_delivered` 表示是否存在在线连接接收了即时通知；即使为 `false`，任务仍保留为 `pending`，TeamAI 重连后可通过 `/local-agent/sync` 补拉。

- **响应：**

```json
{
  "ok": true,
  "task": {
    "id": 501,
    "instance_id": 12,
    "instance_c_id": "local-codebuddy-456789",
    "project_id": 101,
    "workspace_path": "/Users/alice/code/repo-x",
    "agent_type": "codebuddy",
    "prompt": "修复登录失败问题",
    "status": "pending",
    "created_at": "2026-08-07T10:00:00+08:00",
    "updated_at": "2026-08-07T10:00:00+08:00"
  }
}
```

### `GET /agent-tasks`（查询任务与结果）

查询当前用户创建的本地 Agent 任务。前端可按短周期轮询该接口，实现 ClawPro 页面上的近实时状态和结果展示。

- **权限：** 登录用户

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 否 | 任务 ID |
| project_id | uint | 否 | 项目 ID |
| status | string | 否 | `pending` / `running` / `success` / `failed` / `cancelled` |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，最大 100 |

- **响应：** `tasks[]` 字段与创建接口的 `task` 一致；执行中会持续更新 `result`，终态包含 `finished_at`，失败时包含 `error`。

### `POST /local-agent/wake-ticket`（常驻 TeamAI 建连票据）

TeamAI 先通过带 Bearer 身份认证的 HTTPS 请求获取 60 秒有效、一次性使用的随机票据。请求体沿用 sync 的设备身份字段：`agent_type` 和 `local_agent_id`。Bearer Token 不进入 WebSocket URL。

### `GET /local-agent/wake?ticket=...`（WebSocket 轻量唤醒）

使用一次性票据升级为 WebSocket。服务端只发送 `sync_required`、`task_available` 和心跳确认，不发送完整 Prompt、执行参数或本地凭据。收到唤醒后，TeamAI 必须调用带认证的 `/local-agent/sync` 领取完整任务；连接断开时，任务继续保存在数据库中。

### `POST /local-agent/remove`（本地 Agent 三期）

用户端移除自己的本地 Agent：创建一个 `uninstall_teamai` 本地任务，并**立即将实例 `last_known_status` 写为 `destroying`**（前端展示「销毁中」，不引入 CVM 删除状态机）；不立即删除实例，由 reporter 下次 sync 拉取命令后本地执行插件卸载 + 解绑。任务持久化，离线场景下次连接自动执行。

- **权限：** 登录用户（仅能移除自己的 source=local 实例）
- **Content-Type：** `application/json`
- **请求体：** `application/json`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_id | uint | 是 | 目标实例主键（instances.id，「我的 Agent」列表中获取） |

**示例 Request：**

```json
POST /local-agent/remove
Authorization: Bearer hk-xxx
Content-Type: application/json

{ "instance_id": 789 }
```

- **响应：** `application/json`

**成功响应：**

```json
{ "ok": true, "task_id": 789, "status": "pending" }
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| task_id | uint | 创建（或幂等命中已有 pending）的 `local_agent_tasks.id` |
| status | string | 任务状态，创建后为 `pending` |

> **幂等：** 同一实例已存在 pending 的 `uninstall_teamai` 任务时，不重复创建，直接返回已有 `task_id`，重复调用无副作用。

- **错误响应：**
  - `400 {"error": "<参数缺失或非法>"}`
  - `404 {"error": "<实例不存在或无权限>"}` — 实例不存在、非本地实例、或不属于当前用户
  - `403 {"error": "<本地 Agent 未开放>"}` — 白名单/开关未通过

### `POST /admin/local-agent/remove`（本地 Agent 三期）

管控端移除指定本地 Agent（管理员操作）。创建 `uninstall_teamai` 任务，并**立即将实例 `last_known_status` 写为 `destroying`**（与用户端一致）；写接口注册审计。

- **权限：** 管理员（requireAdmin，注册 auditRules + WithAudit）
- **Content-Type：** `application/json`
- **请求体：** `application/json`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_id | uint | 是 | 目标实例主键（instances.id，管控端 Agent 列表已有） |

**示例 Request：**

```json
POST /admin/local-agent/remove
Authorization: Bearer <admin-token>
Content-Type: application/json

{ "instance_id": 789 }
```

- **响应：** 同用户端（ok / task_id / status）。

- **错误响应：**
  - `400 {"error": "<参数缺失>"}`
  - `404 {"error": "<实例不存在或非本地实例>"}`

> 执行链路：创建任务即把实例 `last_known_status` 写为 `destroying`（前端展示「销毁中」）并标记 `current_operation=uninstall_local_agent` 防重入 → reporter 下次 sync 拉到 `uninstall_teamai` 命令 → 本地执行 `teamai uninstall --force --agent <agent_type>` → ack `success` 后软删实例四表（instances 软删；local_instance_skills / local_instance_rules 硬删；local_instance_infos 软删），关联数据清理干净 → ack `failed` 保留任务记录，实例退出卸载中（`last_known_status` 恢复 `running`、`current_operation` 清空），管控端可重试 重新 report 同一本地 Agent 时通过 `Unscoped` 查询重新激活已软删实例。

### `GET /local-agent/availability`

普通用户查询本地 Agent 是否可用（聚合判定，内部决策因子不外泄）。

- **权限：** 登录用户
- **响应：**

```json
{
  "enabled": true
}
```



### `GET /openclaw/current-image`

获取当前后台启用的镜像信息，前端据此判断是否展示"一键更新"按钮。支持按实例的智能体类型查询对应的启用镜像。

- **权限：** 登录用户
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| agent_type | string | 否 | 智能体类型（内置 `openclaw`/`hermes`/`lightclawace`，或自定义类型 code）。传入时返回该类型的启用镜像；不传则回退到任意启用镜像。**自定义类型不会回退到兼容内置类型的镜像，必须有自身启用镜像才会返回非 null** |

- **响应：** 始终 JSON

**响应结构：**

```typescript
interface CurrentImageResponse {
  image: AIImageWithPublic | null;  // 启用的镜像，未配置时为 null
}

interface AIImageWithPublic {
  id: number;
  image_id: string;      // CVM 镜像 ID，如 "img-xxx"
  image_name: string;    // 镜像名称
  image_type: string;    // "PUBLIC_IMAGE" | "PRIVATE_IMAGE"
  os_name: string;       // 操作系统名称
  image_size: number;    // 镜像要求的系统盘容量（GB）；0 表示未知
  image_state: string;   // 镜像状态
  enabled: boolean;      // 恒为 true
  public?: boolean;      // true = image_type 为 PUBLIC_IMAGE
  agent_type: string;    // 智能体类型
  agent_version: string; // 智能体版本号
}
```

**前端判断逻辑：**

| 条件 | 前端行为 |
|------|---------|
| `image` 不为 null 且 `image.public === true` | **显示**"一键更新"按钮 |
| `image` 为 null 或 `image.public !== true` | **不显示**"一键更新"按钮 |

> **注意**：一键升级支持 OpenClaw / Hermes 类型实例（按 agent_type 分派备份/恢复脚本）。LightclawACE 类型暂不支持。

**响应示例：**

```json
// 启用了公共镜像（传入 agent_type=openclaw）
{"image": {"id": 1, "image_id": "img-xxx", "image_name": "OpenClaw on TencentOS Server 4", "image_type": "PUBLIC_IMAGE", "public": true, "agent_type": "openclaw", "agent_version": "2026.4.11", ...}}

// 启用了自定义镜像
{"image": {"id": 2, "image_id": "img-yyy", "image_name": "Custom Image", "image_type": "PRIVATE_IMAGE", "agent_type": "hermes", "agent_version": "0.9.0", ...}}

// 未配置启用镜像
{"image": null}
```

### `POST /openclaw/create`

创建新实例（调用腾讯云 CVM API 开机）。创建前会检查用户实例配额，超过配额则拒绝创建。

- **权限：** 登录用户
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 实例名称（不超过 128 字符） |
| role_id | uint | 否 | 角色 ID，不传或传 0 则为「通用助手」。后端会校验角色存在且 `visible=true`。仅 OpenClaw 类型有效 |
| agent_type | string | 否 | 智能体类型（内置 `openclaw`/`hermes`/`lightclawace`，或自定义类型 code），默认为系统首选类型（`default_agent_type`）。必须有对应的启用镜像 |
| group_id | uint | 否 | 实例绑定的分组 ID。省略或传 0 表示未指定分组；传正整数时服务端校验分组存在性与角色可见性，写入 `instances.group_id` |
| disk_type | string | 否 | 兼容系统盘类型参数，应用在生效资源策略和本次 `resource_config` 之后。不传则保留前述层级计算出的类型；支持 `CLOUD_SSD`、`CLOUD_PREMIUM`、`CLOUD_BSSD`、`CLOUD_HSSD`。若与本次 `resource_config.system_disk.disk_type` 不同则返回 400 |
| resource_config | string | 否 | 本次创建的资源覆盖 JSON object（snake_case；schema 见资源策略管理接口）。未知字段忽略，预期字段会 Normalize/Validate；非 object、`null`、数组、多个 JSON 值或尾随内容返回 400 |
| user_data | string | 否 | 用户自定义 UserData（base64 编码）。实例启动时通过 cloud-init 执行。需管理员开启 `user_data_enabled` 站点开关后方可使用（base64 字符串长度不超过 12KB） |
| tags | string | 否 | 自定义 CVM 标签的 JSON 数组字符串，例如 `[{"key":"env","value":"prod"},{"key":"team","value":"sales"}]`。数组元素只接受 `key`、`value` 字段 |

- **JSON 响应：**
  - 成功：`{"ok": true, "redirect": "/openclaw", "instance_id": ins-5p****t}`（`instance_id` 为新建实例的CVM ID）
  - 失败：`400 {"error": "实例名称不能为空且不能超过128个字符"}` / `400 {"error": "所选角色不存在或不可用"}` / `400 {"error": "disk_type 不支持的系统盘类型: ..."}` / `400 {"error": "当前 ClawPro 未配置安全组，请联系企业管理员配置安全组后重试。"}` / `400 {"error": "user_data 必须是合法的 base64 字符串"}` / `400 {"error": "user_data 不能超过 12KB（base64 字符串长度）"}` / `400 {"error": "请求参数格式错误"}`（tags JSON 非法）/ `403 {"error": "UserData 功能未开启，请联系管理员在后台开启"}` / `403 {"error": "已达到实例配额上限（N）"}` / `500 {"error": "..."}`

> **自定义标签：** `tags` 可一次传入多个标签。标签 key 必须已经存在于 ClawPro 绑定的腾讯云账号中，key/value 及配额规则由 CVM `RunInstances` 统一校验。创建流程同时存在系统默认标签时，最终标签为“自定义标签 + 未被覆盖的默认标签”；相同 key 以本次请求的自定义值为准。请求内重复 key 不会被服务端静默去重。请求传入至少一个自定义标签时，腾讯云标签校验错误返回 400；请求未传自定义标签时沿用原有行为，RunInstances 标签错误返回 500。标签仅随本次 CVM 创建请求下发，不写入独立的请求标签字段；实例表只缓存云端最终标签供展示和过滤。

> **资源配置优先级与冲突：** CVMTemplate 是基础请求；服务端按“当前组直接策略 → 最近祖先策略 → 企业默认资源策略”选择一条独立 `ResourcePolicy` 并覆盖其明确提供的字段；随后应用本请求的 `resource_config`；最后应用兼容参数 `disk_type`。策略未提供的字段回退 CVMTemplate，不与其他策略做字段级合并。若 `disk_type` 与请求 `resource_config.system_disk.disk_type` 同时存在且值不同，返回 `400 {"error":"磁盘类型错误"}`。
>
> **镜像与系统盘：** `AIImage.image_size` 与 `system_disk.disk_size` 均以 GB 为单位。`image_size > 0` 且最终明确选择的系统盘容量更小时，创建在 VPC、安全组和 RunInstances 之前失败，例如 `400 {"error":"所选镜像要求系统盘至少 100GB，当前选择为 50GB"}`。服务端不会静默扩大磁盘。`image_size=0`、未设置系统盘或未明确 `disk_size` 时跳过本地容量比较，继续交由 CVM 判断。
>
> **创建前机型预检：** 最终可用区和机型确定后，服务端先使用租户隔离的 options 缓存；缓存未命中时查询腾讯云 CVM。目标机型不在 SELL 结果中或云查询失败时 fail closed，返回 400，并且不会进入安全组选择或 RunInstances。

> **默认模型可见性检查**：若系统配置了默认模型（`DefaultModelID`），创建实例时会检查用户对该模型的可见性。如果默认模型对用户不可见（`visibility_type=group` 且用户不在关联分组中），则**跳过默认模型应用**，新实例不会自动绑定任何模型，用户需手动选择一个可见的模型。

> **默认模型自动注入仅限 OpenClaw**：即便通过可见性检查，也**只有 `agent_type=openclaw` 的实例**会异步注入站点默认模型（由 `AgentTypeSupportsDefaultModelInjection` guard 控制）。Hermes / LightclawACE 实例创建后 `ai_model_id=0`，用户需进入详情页手动调用 `POST /openclaw/set-model` 选择模型。

> **UserData 持久化与复用**：创建时传入的 `user_data` 会持久化到 `instances.user_data` 字段（不通过任何列表/详情 API 暴露，因为可能含密钥等敏感信息）。后续对该实例执行 `POST /openclaw/reset` 或 `POST /openclaw/upgrade` 时，后端会自动读取该字段并与系统初始化脚本合并后传给 CVM，确保用户脚本在重装/升级后再次执行，无需用户重新传入。
>
> 这样设计的原因：
> - 用户手动 `set-model` 对三端均开放（`AgentTypeSupportsModel=true`），脚本按 agent_type 分派（见该接口说明）；
> - 但站点默认模型的 provider/baseUrl 契约基于 OpenClaw 设计，若无脑推给 Hermes(harness) / ACE(lightclaw)，其运行时 API 形态可能不兼容，会出现"首次 TAT 脚本静默失败、但 `ai_model_id` 已写 DB"的不一致状态；
> - 因此 Hermes / ACE 的模型配置一律由用户显式选择。

### `POST /openclaw/delete`

删除实例（同时销毁腾讯云 CVM）。

- **权限：** 登录用户（仅可删除自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **前置校验：** 后端会先查询当前 OpenClaw 状态，以下状态禁止删除（返回 `409`）：`creating`、`loading`、`destroying`、`pending`、`upgrading`、`upgrade_failed`
- **行为说明：**
  - 本地 Agent 实例（`source=local`）：**不支持本接口删除**，返回 `400 unsupported operation`。本地实例删除请走 `/local-agent/remove`（用户端）或 `/admin/local-agent/remove`（管控端），走「下发 uninstall_teamai 任务 → reporter 执行 → ack 异步软删」语义。
  - 创建失败（`create_failed`）或无 CVM 实例 ID：仅删除数据库记录，不调用 CVM API
  - 正常实例：调用 `TerminateInstances` + `PurgeInstances` 确认销毁，CVM 消失后清除 DB 记录
  - 前端乐观更新：删除请求成功后卡片立即消失，失败时回滚
- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`400/404/500 {"error": "..."}`
  - 状态拒绝：`409 {"error": "实例当前为创建中，请等待实例恢复运行中后再操作"}`，文案按实际状态变化，完整规则见「实例状态准入规则」章节

### `POST /openclaw/start`

用户端开机实例，支持单个和批量开机，仅可操作自己的实例。

- **权限：** 登录用户
- **Content-Type：** `application/x-www-form-urlencoded` 或 `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 单个实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | 单个 CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |
| ids | uint[] | 条件 | 批量实例数据库 ID；仅 JSON 支持；与 `instance_ids` 二选一，`ids` 优先 |
| instance_ids | string[] | 条件 | 批量 CVM 实例 ID；仅 JSON 支持；与 `ids` 二选一 |

> `id` / `instance_id` 支持 Query、Form、JSON；`ids` / `instance_ids` 仅支持 JSON。若请求包含单个与批量参数，批量参数优先。

- **前置条件：** 实例状态为 `stopped`
- **本地实例拒绝：** `source='local'` 的实例会被返 `400 {"error": "本地实例不支持此操作"}`。batch 模式下仅拒绝 local target，其它 CVM target 正常处理（results 数组里对应项返 `ok=false`）。
- **前置条件：** 实例状态为 `stopped`；且实例未携带 stale-instances v1.0 拦截标：
    - `pending_user_action`：待用户处理分组归属的实例，需先在弹窗完成迁移或移交（返回 409）
    - `stale_group`：分组归属异常，需管理员在管控端处理（返回 409）；管理员在 `POST /admin/instances/start` 成功开机后自动清除该实例的**全部** stale-instances 标记（`stale_group` / `pending_user_action` / `allow_migrate` / `allow_same_group_handover`）
- **JSON 响应：**
  - 单个成功：`{"ok": true}`
  - 批量成功/部分失败：`{"ok": true, "results": [{"id":1,"instance_id":"ins-a","name":"agent-a","status":"started","message":"开机已下发"},{"id":2,"instance_id":"ins-b","name":"agent-b","status":"skipped","message":"实例当前为运行中，无法执行该操作"}]}`
  - 失败：`400/404/409/500 {"error": "..."}`

### `POST /openclaw/stop`

用户端关机实例，支持单个和批量关机，仅可操作自己的实例。

- **权限：** 登录用户
- **Content-Type：** `application/x-www-form-urlencoded` 或 `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 单个实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | 单个 CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |
| ids | uint[] | 条件 | 批量实例数据库 ID；仅 JSON 支持；与 `instance_ids` 二选一，`ids` 优先 |
| instance_ids | string[] | 条件 | 批量 CVM 实例 ID；仅 JSON 支持；与 `ids` 二选一 |

> `id` / `instance_id` 支持 Query、Form、JSON；`ids` / `instance_ids` 仅支持 JSON。若请求包含单个与批量参数，批量参数优先。

- **前置条件：** 实例状态为 `running`
- **行为：** 关机；按量计费实例关机后停止计算资源计费
- **本地实例拒绝：** `source='local'` 的实例会被返 `400 {"error": "本地实例不支持此操作"}`。batch 模式下仅拒绝 local target，其它 CVM target 正常处理。
- **JSON 响应：**
  - 单个成功：`{"ok": true}`
  - 批量成功/部分失败：`{"ok": true, "results": [{"id":1,"instance_id":"ins-a","name":"agent-a","status":"started","message":"关机已下发"},{"id":2,"instance_id":"ins-b","name":"agent-b","status":"skipped","message":"实例当前为已关机，无法执行该操作"}]}`
  - 失败：`400/404/409/500 {"error": "..."}`

### `POST /openclaw/reboot`

重启实例（调用腾讯云 CVM 重启接口）。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`400/404/500 {"error": "..."}`
  - 状态拒绝：`409 {"error": "实例已关机，请先开机并等待实例恢复运行中后再操作"}`，文案按实际状态变化，完整规则见「实例状态准入规则」章节

### `POST /openclaw/restart-gateway`

重启 Agent（仅重启实例内对应 agent 类型的 gateway 服务，不重启腾讯云 CVM 实例）。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **前置条件：** 实例状态为 `running`
- **行为：** 通过 TAT 按 agent 类型执行 gateway 重启脚本（OpenClaw / Hermes / LightclawACE）；不调用 CVM `RebootInstances`，不写入 `current_operation`，不重置 `agent_ready`
- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`400/404/500 {"error": "..."}`
  - 状态拒绝：`409 {"error": "实例已关机，请先开机并等待实例恢复运行中后再操作"}`，文案按实际状态变化，完整规则见「实例状态准入规则」章节

### `POST /openclaw/reset`

重装实例（调用腾讯云 CVM ResetInstance 接口，使用最新配置的镜像 ID 重装系统）。重装会丢失所有数据并更新镜像到最新版本，同时重置模型绑定（AIModelID 清零）。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`400/404/500 {"error": "..."}`
  - 状态拒绝：`409 {"error": "实例已关机，请先开机并等待实例恢复运行中后再操作"}`，文案按实际状态变化，完整规则见「实例状态准入规则」章节

### `POST /openclaw/upgrade`

升级实例到最新镜像版本。与 `/openclaw/reset` 不同，本接口会在重装前自动备份实例数据到 SMH，重装完成后自动恢复数据，实现无损升级。

> **类型限制：** 一键升级支持 OpenClaw / Hermes 类型实例。LightclawACE 及其他类型调用此接口会返回 `400 {"error": "xxx 类型实例暂不支持一键升级"}`。

**接口行为：** 异步执行。接口立即返回，升级流程在后台异步进行。升级进度通过 `GET /openclaw/status` 查询。

**升级流程（后台异步执行）：**
1. 检查实例当前镜像是否与后台配置的默认镜像一致，若已是最新版本则直接返回成功
2. 设置操作锁（`currentOperation=upgrade`），此时 `/openclaw/status` 返回 `upgrading` 状态
3. 执行重装前数据备份（按 agent_type 分派 `backup_pre_reinstall*.sh`，超时 10 分钟）
4. 将备份包分块上传到 SMH
5. 调用腾讯云 CVM `ResetInstance` 接口重装系统（使用后台配置的默认镜像）
6. 等待实例重装完成并恢复 RUNNING 状态（最长等待 15 分钟）
7. 执行重装后数据恢复（按 agent_type 分派 `restore_post_reinstall*.sh`，超时 40 分钟，失败自动重试最多 5 次）
8. 删除 SMH 上的临时备份文件
9. 升级成功后清除操作锁，状态恢复为 `running`；升级失败则状态变为 `upgrade_failed`（实例仍为 RUNNING，数据不丢失）

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |

- **JSON 响应：**
  - 无需升级：`200 {"ok": true, "message": "实例已是最新版本，无需升级"}`
  - 升级已开始：`200 {"ok": true, "message": "升级已开始"}`
  - 失败：`400/500 {"error": "..."}`
  - 操作冲突：`409 {"error": "实例正在进行 xxx 操作，请稍后再试"}`
  - 状态拒绝：`409 {"error": "实例已关机，请先开机并等待实例恢复运行中后再操作"}`，文案按实际状态变化，完整规则见「实例状态准入规则」章节

**升级状态跟踪：** 调用本接口后，通过 `GET /openclaw/status?id=<id>` 轮询实例状态：
  - `upgrading`（升级中）：`transient=true`，前端应自动轮询
  - `upgrade_failed`（升级失败）：升级失败但实例仍可用。此时 `CurrentOperation=upgrade`、`CurrentOperationState=failed` 均保留，前端可据此展示"重试"按钮并调用 `POST /openclaw/upgrade/retry` 触发重试
  - `running`（运行中）：升级完成

---

### `POST /openclaw/upgrade/retry`

升级失败后的重试入口。仅当实例处于"升级失败"状态时可调用，若 SMH 上仍保留上次的备份文件则复用该备份、跳过备份+上传阶段，直接重装并恢复数据；否则退化为完整的 `/openclaw/upgrade` 流程。

> **类型限制：** 与 `/openclaw/upgrade` 相同，支持 OpenClaw / Hermes 类型实例。

**前置条件：**
- 实例的 `CurrentOperation == "upgrade"` 且 `CurrentOperationState == "failed"`（即上一次升级失败后，`current_operation` 字段会保留为 `upgrade` 不被清空，便于识别失败态）
- 实例类型支持一键升级（OpenClaw / Hermes）
- 后台已配置生效的对应类型镜像

**接口行为：** 异步执行，与 `/openclaw/upgrade` 一致。

**重试流程（后台异步执行）：**
1. 校验失败状态、类型限制和镜像配置
2. 查询 SMH common space 下 `backups/{instanceId}/` 目录中是否存在历史备份：
   - **存在**（快速路径）：取最新备份（按文件名中的时间戳字典序），直接执行**重装 → 等待就绪 → 数据恢复 → 清理 SMH 备份**（跳过耗时的 CVM 备份+SMH 分块上传环节）
   - **不存在**（完整路径）：走与 `/openclaw/upgrade` 完全一致的流程（备份 → 上传 → 重装 → 恢复 → 清理）
3. 通过 `setOperationForRetry` 覆盖失败的操作锁，状态重置为 `upgrade` + `processing`；重试结果最终通过 `current_operation_state` 反映（`success` 或 `failed`）

**备份复用说明：**
- 不设时效限制：只要 SMH 上存在 `backups/{instanceId}/openclaw-state-*.tgz` 均可复用
- 文件命名形如 `openclaw-state-YYYYMMDD_HHMMSS.tgz`，时间戳字典序与时间序一致，自动取最新
- 若查询 SMH 失败，降级走完整升级流程（不阻断重试）

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |

- **JSON 响应：**
  - 重试已开始（命中历史备份）：`200 {"ok": true, "message": "升级重试已开始（复用 SMH 历史备份）"}`
  - 重试已开始（无历史备份）：`200 {"ok": true, "message": "升级重试已开始"}`
  - 当前不是升级失败状态：`400 {"error": "当前不是升级失败状态，无法重试（current_operation=xxx, state=xxx）"}`
  - 不支持升级的类型：`400 {"error": "xxx 类型实例暂不支持一键升级"}`
  - 未配置生效镜像：`400 {"error": "暂无生效的xxx类型镜像，请联系管理员处理"}`
  - 操作冲突：`409 {"error": "设置升级重试操作锁失败: ..."}`
  - 失败：`500 {"error": "..."}`

**状态跟踪：** 与 `/openclaw/upgrade` 一致，通过 `GET /openclaw/status?id=<id>` 轮询。重试期间 `CurrentOperation=upgrade` + `CurrentOperationState=processing`；重试成功后 `CurrentOperation=""` + `CurrentOperationState=success`；重试失败后 `CurrentOperation=upgrade` + `CurrentOperationState=failed`（可再次重试）。

---

### `POST /openclaw/migration/export`

为目标实例生成 agent 数据迁移导出脚本。目标实例无需处于 running 状态。

返回一段可在**源实例**终端直接执行的 shell 脚本，脚本内嵌 SMH 上传凭证，执行后将源实例 agent 数据目录打包上传到 SMH common space。

**接口行为：** 幂等。同一实例若已有未过期的迁移记录，重新调用会返回同一记录（刷新 access token）。

**导出稳定性与日志：**

- 导出命令会将终端输出同步写入源实例 `<agent_home>/logs/migration_export_<时间>_<进程号>.log`，失败时明确打印阶段、退出码和日志路径；日志不记录 SMH token、header 或上传 URL。
- Hermes Agent 面板保持开启时，Dashboard 对已排除日志/锁文件的写入仍可能使 `.hermes` 根目录元数据发生变化。此兼容仅对 Hermes 生效；导出脚本只在 tar 状态码为 1、唯一诊断为归档根目录 `file changed as we read it`、打包前后未排除业务树指纹一致且归档完整性校验通过时继续上传。
- 若变化发生在 `config.yaml` 等普通业务文件、业务树出现新增/删除/修改/替换、tar 出现其他错误或归档校验失败，导出仍会中止，避免上传不一致或损坏的数据。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 目标实例数据库 ID |

- **响应（200）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| migration_id | uint | 迁移记录 ID |
| script | string | 可直接在源实例终端执行的 shell 导出脚本 |
| file_key | string | SMH common space 中的文件路径 |
| expire_at | string (RFC3339) | SMH 返回的上传凭证有效期，超过后分块上传 URL 和确认密钥均失效，需重新调用本接口获取新凭证；秒传时（文件已存在）不返回此字段 |

```json
{
  "migration_id": 1,
  "script": "PART_URL_TEMPLATE='...' \\\nPART_HEADERS_B64='...' \\\n... bash <<'BASH'\n...\nBASH",
  "file_key": "migrations/ins-xxx/agent-export.tgz",
  "expire_at": "2026-04-30T03:44:52Z"
}
```

- **错误：**
  - `403`：SMH 服务未启用
  - `400`：实例不存在或无权限
  - `500`：初始化 SMH 上传失败

---

### `GET /openclaw/migration/status`

查询目标实例的迁移导出文件是否已就绪（用于判断是否可以调用 import）。

- **权限：** 登录用户
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 目标实例数据库 ID |

- **响应（200）：**

```json
{
  "has_migration": true,
  "migration_id": 1,
  "file_key": "migrations/ins-xxx/agent-export.tgz",
  "file_ready": true,
  "file_size": 52428800,
  "expires_at": "2026-04-28T11:00:00+08:00",
  "can_import": true
}
```

当无有效迁移记录时：`{"has_migration": false}`

---

### `GET /openclaw/migration/progress`

查询迁移 import 的执行进展（纯 DB 查询，无 SMH 调用）。import 触发后轮询此接口获取结果。

- **权限：** 登录用户
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 目标实例数据库 ID |

- **响应（200）：**

```json
{
  "has_migration": true,
  "migration_id": 1,
  "status": "importing",
  "steps": [
    {"step": "downloading",    "status": "success", "ts": "2026-04-28T10:00:01Z", "detail": null},
    {"step": "backing_up",     "status": "success", "ts": "2026-04-28T10:00:45Z", "detail": null},
    {"step": "extracting",     "status": "success", "ts": "2026-04-28T10:01:05Z", "detail": null},
    {"step": "restarting",     "status": "running", "ts": "2026-04-28T10:01:20Z", "detail": null},
    {"step": "syncing_models", "status": "pending", "ts": null, "detail": null},
    {"step": "syncing_smh",    "status": "pending", "ts": null, "detail": null}
  ]
}
```

`status` 可能值：`pending_upload` / `importing` / `done` / `failed`

`steps` 字段仅在有 import 记录时返回（`has_migration: true`），`status` 为 `pending_upload` 时为 `null`。每个 step 的 `status` 可能值：

| 值 | 说明 |
|----|------|
| pending | 尚未开始 |
| running | 进行中 |
| success | 已完成 |
| failed | 失败（不一定导致整体失败，如 `syncing_smh`） |

step 顺序固定：`downloading` → `backing_up` → `extracting` → `restarting`（含等待就绪）→ `syncing_models` → `syncing_smh`（仅支持 SMH 的实例类型）

`syncing_models` 完成时 `detail` 包含 `is_primary_model_valid`（`bool`），表示是否成功提取到 primary 模型（`true` 时用户可直接使用，`false` 时需要手动配置模型）：

```json
{"step": "syncing_models", "status": "success", "ts": "...", "detail": {"is_primary_model_valid": true}}
```

失败时附带 `fail_reason` 字段：

```json
{ "has_migration": true, "migration_id": 1, "status": "failed", "fail_reason": "恢复 agent 数据失败: ...", "steps": [...] }
```

无记录时：`{"has_migration": false}`

---

### `POST /openclaw/migration/import`

触发目标实例从 SMH 下载并恢复 agent 数据。目标实例须处于 running 状态（需 TAT 执行恢复脚本）。

**接口行为：** 异步执行。接口立即返回，恢复流程在后台异步进行。

**恢复流程（后台异步执行，可通过 `/migration/progress` 的 `steps` 字段追踪进度）：**
1. 设置操作锁（`currentOperation=migrate`），初始化全部阶段为 `pending`
2. 在目标实例上执行 `restore_from_migration.sh`：
   - `downloading`：下载迁移包
   - `backing_up`：mv 现有 agent 目录到 /tmp 备份
   - `extracting`：解压覆盖 agent 目录、修复配置（如 gateway.port）
   - `restarting`：重启 gateway，**持续到 agent 就绪**（最长 10 分钟）
3. `syncing_models`：从 agent 配置提取源实例模型，写入 `instance_models`；`detail.is_primary_model_valid` 表示是否有可用 primary 模型
4. `syncing_smh`：同步 SMH 个人空间配置（仅支持 SMH 的实例类型）
5. 重置版本缓存、内存插件状态、技能/插件/MCP 安装记录
6. 删除 SMH 上的迁移文件

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 目标实例数据库 ID |

- **响应：**
  - 迁移已开始：`200 "迁移已开始"`
  - 文件未就绪：`400 {"error": "迁移文件尚未上传，请先在源实例终端运行导出脚本"}`
  - 无有效记录：`400 {"error": "未找到有效的迁移导出记录，请先运行导出脚本"}`
  - 操作冲突：`409 {"error": "实例正在进行 xxx 操作，请稍后再试"}`
  - 状态拒绝：`409 {"error": "实例已关机，请先开机并等待实例恢复运行中后再操作"}`，文案按实际状态变化，完整规则见「实例状态准入规则」章节
  - `403`：SMH 服务未启用

### `GET /openclaw/status` ⚠️ 重大变更

查询实例 OpenClaw 语义状态（已从原始 CVM 状态升级为语义状态）。

- **权限：** 登录用户
- **查询范围：** 「自有实例 ∪ 待我接收的移交实例」并集（与 `/openclaw/list` 口径一致）。移交接收方可查询待接收实例的状态，但写操作（reboot/stop/reinstall 等）仍需 accept 后才能执行。
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID（Query 参数）；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`，Query 参数）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。未传任何参数时返回空状态对象。

- **JSON 响应：** 返回 OpenClaw 语义状态对象（见下方）。未传 id 或 id 无效时返回 `{"status": "", "label": "", "tooltip": "", "actions": [], "transient": false}`。

**响应格式（JSON）：**

```typescript
interface InstanceStatusResponse {
  status: OpenClawStatus;  // 12 种语义状态
  label: string;           // 中文标签（如 "运行中"）
  tooltip: string;         // Hover 提示文字
  actions: ActionType[];   // 当前可执行操作列表
  transient: boolean;      // 是否过渡态（true 时前端应自动轮询）
  instance_charge_type: string; // CVM 实例实际计费类型
}

type OpenClawStatus = 'creating' | 'create_failed' | 'running' | 'stopped'
  | 'loading' | 'load_failed' | 'maintaining' | 'pending'
  | 'destroying' | 'destroyed'
  | 'upgrading' | 'upgrade_failed';
type ActionType = 'start' | 'stop' | 'reboot' | 'reinstall' | 'delete' | 'terminal' | 'retry';
```

**响应示例：**

```json
// 创建中
{ "status": "creating", "label": "创建中", "tooltip": "正在创建中，请稍候", "actions": [], "transient": true }

// 创建失败
{ "status": "create_failed", "label": "创建失败", "tooltip": "创建失败，可删除后重新创建", "actions": ["delete"], "transient": false }

// 运行中
{ "status": "running", "label": "运行中", "tooltip": "", "actions": ["stop","reboot","reinstall","delete","terminal"], "transient": false }

// 已关机
{ "status": "stopped", "label": "已关机", "tooltip": "已关机，可开机恢复", "actions": ["start","delete"], "transient": false }

// 加载中
{ "status": "loading", "label": "加载中", "tooltip": "加载中，请稍候", "actions": [], "transient": true }

// 加载失败
{ "status": "load_failed", "label": "加载失败", "tooltip": "加载失败，可点击重试恢复", "actions": ["retry","delete"], "transient": false }

// 维护中
{ "status": "maintaining", "label": "维护中", "tooltip": "维护中，请稍候", "actions": ["delete"], "transient": true }

// 待处理（欠费/到期等平台隔离）
{ "status": "pending", "label": "待处理", "tooltip": "已停用，请联系管理员处理", "actions": [], "transient": true }

// 销毁中（currentOp=delete 且 CVM 仍存在，删除流程进行中）
{ "status": "destroying", "label": "销毁中", "tooltip": "正在销毁中，请稍候", "actions": [], "transient": true }

// 已销毁（CVM 状态 TERMINATING/TERMINATED，或 currentOp=delete 乐观更新场景）
{ "status": "destroyed", "label": "已销毁", "tooltip": "", "actions": ["delete"], "transient": false }

// 升级中（调用 /openclaw/upgrade 后的过渡态）
{ "status": "upgrading", "label": "升级中", "tooltip": "正在升级，请稍候", "actions": [], "transient": true }

// 升级失败（升级流程异常，但实例仍为 RUNNING 可用）
{ "status": "upgrade_failed", "label": "升级失败", "tooltip": "升级失败，请重试或联系管理员", "actions": [], "transient": false }
```

**完整 status → actions / tooltip 映射：**

| status | actions | tooltip |
|--------|---------|--------|
| creating | `[]` | 正在创建中，请稍候 |
| create_failed | `["delete"]` | 创建失败，可删除后重新创建 |
| running | `["stop","reboot","reinstall","delete","terminal"]` | (空) |
| stopped | `["start","delete"]` | 已关机，可开机恢复 |
| loading | `[]` | 加载中，请稍候 |
| load_failed | `["retry","delete"]` | 加载失败，可点击重试恢复 |
| maintaining | `["delete"]` | 维护中，请稍候 |
| pending | `[]` | 已停用，请联系管理员处理 |
| destroying | `[]` | 正在销毁中，请稍候 |
| destroyed | `["delete"]` | (空) |
| upgrading | `[]` | 正在升级，请稍候 |
| upgrade_failed | `[]` | 升级失败，请重试或联系管理员 |

**后端副作用（对前端透明）：**

| 副作用 | 触发条件 | 行为 |
|--------|---------|------|
| 删除确认 + Purge | `currentOp=delete` + CVM 状态变化 | `ISOLATED` → `PurgeInstances`；CVM 消失 → 清 DB |
| Agent 检测 | `CVM=RUNNING` + `agent_ready=0` | 后台异步检测（5s 周期批量），就绪后写 DB，不阻塞响应 |
| 操作超时 | 超过阈值 | 标记 `failed` 状态 |
| 操作收敛 | CVM 非 RUNNING 稳定态 + `currentOp` 非空 | 清除 `currentOp` |

**前端适配说明：**

- 响应格式已从 `{ state }` 变更为 `{ status, label, tooltip, actions, transient }`
- 用 `label` 替代原来的 `STATE_LABEL[state]` 映射
- 用 `actions` 数组渲染操作按钮，替代硬编码逻辑
- `transient=true` 时前端应启动 3s 递归轮询

### `POST /openclaw/retry`

对加载失败（`load_failed`）的实例发起重试。后端根据上一次操作类型自动选择重试方式。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **前置条件：** 当前状态必须为 `load_failed`，否则返回 `400`
- **重试策略（后端自动判断）：**

| 上一次 `currentOperation` | 重试行为 |
|--------------------------|--------|
| create / reboot / "" | 调用 `RebootInstances` |
| reinstall | 调用 `ResetInstance`（使用当前启用镜像） |

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 当前状态不支持重试：`400 {"error": "当前状态为 xxx，只有 load_failed 状态才能重试"}`
  - 操作冲突：`409 {"error": "..."}`
  - 失败：`404/500 {"error": "..."}`

### `POST /openclaw/rename`

修改实例名称（同步更新腾讯云 CVM InstanceName + 本地数据库）。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 二选一 | 实例数据库 ID（优先使用） |
| instance_id | string | 二选一 | 腾讯云 CVM InstanceId（id 缺失时 fallback） |
| name | string | 是 | 新实例名称（1~128 字符，前后空格自动去除） |

> `id` 和 `instance_id` 至少提供一个；同时提供时以 `id` 为准。两者均支持 URL query 参数和 POST body 传参。

- **前置条件：**
  - 实例必须已关联 CVM（`InstanceId` 非空），否则返回 `409`
  - CVM 状态必须为 `RUNNING` 或 `STOPPED`，否则返回 `409`
- **行为说明：**
  - 先调用腾讯云 `ModifyInstancesAttribute` 修改 CVM 名称
  - CVM 改名成功后再更新本地数据库 `name` 字段
  - 如 CVM 改名失败则整体失败，本地数据库不变
- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 参数缺失：`400 {"error": "缺少参数 id 或 instance_id"}`
  - 参数错误：`400 {"error": "实例名称不能为空且不能超过128个字符"}`
  - 实例未就绪：`409 {"error": "实例尚未就绪，无法修改名称"}`
  - 状态冲突：`409 {"error": "当前实例状态为 PENDING，无法修改名称"}`
  - 失败：`500 {"error": "..."}`

### `GET /openclaw/roles`

获取员工端可选角色列表（仅返回 `visible=true` 的角色）。

- **权限：** 登录用户
- **响应：** 始终 JSON

成功：

```json
{
  "roles": [
    {
      "id": 1,
      "name": "行业分析师",
      "description": "结构化分析，输出高质量行业洞察",
      "soul": "你是一位具备麦肯锡级别分析能力的行业研究顾问...",
      "skills": [
        {
          "id": 1,
          "openclaw_role_id": 1,
          "name": "Data Analysis",
          "slug": "data-analysis",
          "version": "1.0.0",
          "source": "public"
        }
      ]
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| roles | array | 可选角色列表，按管控端排序顺序返回 |
| roles[].id | uint | 角色 ID，创建实例时传入 `role_id` |
| roles[].name | string | 角色名称 |
| roles[].description | string | 角色描述 |
| roles[].soul | string | 角色灵魂（System Prompt） |
| roles[].skills | array | 角色技能列表 |
| roles[].skills[].source | string | 技能来源：`public`（公共）或 `enterprise`（企业） |

> **注意：** 不暴露 `sort_order`、`visible` 等管理字段。

### `POST /openclaw/remove-role`

移除实例角色（回退为「通用助手」）。仅解除角色关联，不会删除已安装的技能。

> 建议新前端使用 `POST /openclaw/switch-role`（`role_id=0` 等价于 `remove-role`），保留此接口仅为旧前端兼容。

- **权限：** 登录用户（仅可操作自己的实例）
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID（Query 参数或 Form 参数） |

- **响应：** 始终 JSON
  - 成功：`{"ok": true}`
  - 失败：`400/404/500 {"error": "..."}`
  - 状态拒绝：`409 {"error": "实例当前为创建中，请等待实例恢复运行中后再操作"}`，文案按实际状态变化，完整规则见「实例状态准入规则」章节

---

### `POST /openclaw/switch-role`

用户端切换单个实例的角色（含 `role_id=0` 即移除角色）。所有 Agent 类型（含通用助手）均可调用。

切换后行为：
1. 实例 `role_id` 更新为目标角色 ID（0 表示移除）
2. 实例 `distributed_role_version` 更新为目标角色当前版本号（移除时清空）
3. 实例 `soul_set_at` 置为 NULL，触发周期任务重新下发 `SOUL.md`
4. 异步装载新角色技能（按 slug 比对覆盖装；用户已装且新角色不含的 slug 保留不变）

- **权限：** 登录用户（仅可操作自己的实例；owner 校验失败时按 `not_found` 静默跳过，不返回 403/404）
- **Content-Type：** `application/json`
- **审计动作：** `instance_switch_role`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_id | uint | 是 | 实例数据库 ID |
| role_id | uint | 是 | 目标角色 ID；`0` 表示移除角色 |

- **请求示例：**

```json
{
  "instance_id": 100,
  "role_id": 5
}
```

- **响应：** 始终 JSON

```json
{
  "accepted": 1,
  "skipped": []
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| accepted | int | 实际通过校验并触发切换的实例数量（单实例场景下为 0 或 1） |
| skipped | array | 被跳过的实例列表，每项包含 `instance_id` 和 `reason`，详见前端联调文档的 reason 枚举表 |

> 注意：单实例被跳过时仍返回 `200 OK + accepted=0`，前端按 `accepted` 判断成功/失败并 toast 提示。

---

### `GET /openclaw/smh-status`

查询个人空间功能状态及指定实例是否已绑定个人空间。

- **权限：** 登录用户（仅可操作自己的实例）
- **响应：** 始终返回 JSON
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 实例数据库 ID（Query 参数） |

- **成功响应：**

```json
{
  "enabled": true,
  "has_space": true
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| enabled | bool | 个人空间功能是否全局启用 |
| has_space | bool | 该实例是否已绑定个人空间 |

- **失败响应：**
  - `404 {"error": "实例不存在"}`

---

### `GET /openclaw/smh-token`

获取实例对应的个人空间的临时访问 Token。返回的 Token 可用于直接访问该实例绑定的 SMH 个人空间。

- **权限：** 登录用户（仅可操作自己的实例）
- **响应：** 始终返回 JSON
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 实例数据库 ID（Query 参数） |

- **成功响应：**

```json
{
  "token": "...",
  "space_id": "spacexxxxxxxx",
  "library_id": "smhxxxxxxxx",
  "endpoint": "https://smhxxx.cos.tencentsmh.cn",
  "expires_at": 1743562200000
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| token | string | 临时访问 Token |
| space_id | string | 个人空间 ID |
| library_id | string | SMH 媒体库 ID |
| endpoint | string | SMH 访问域名 |
| expires_at | int | Token 过期时间（毫秒时间戳） |

- **失败响应：**
  - `400 {"error": "该实例未绑定个人空间"}`
  - `400 {"error": "个人空间功能未启用"}`
  - `404 {"error": "实例不存在"}`

---

### `GET /openclaw/memory-tdai-status`

查询实例的记忆插件（Memory TDAI）安装状态及全局开关。

- **权限：** 登录用户（仅可操作自己的实例）
- **响应：** 始终返回 JSON
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 实例数据库 ID（Query 参数） |

- **成功响应：**

```json
{
  "memory_tdai_enable": true,
  "status": "ENABLED",
  "last_error": "..."
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| memory_tdai_enable | bool | 记忆功能全局开关 |
| status | string | 该实例的插件状态，可选值见下表 |
| last_error | string | 最近一次错误信息（仅在有错误时返回） |

**status 可选值：**

| 值 | 说明 |
|------|------|
| NOT_INSTALLED | 未安装 |
| ENABLING | 启用中 |
| ENABLED | 已启用 |
| DISABLING | 禁用中 |
| DISABLED | 已禁用 |
| FAILED | 失败 |
| UNSUPPORTED_VERSION | 不支持的版本 |

- **失败响应：**
  - `404 {"error": "实例不存在"}`
  - `403 {"error": "Hermes 类型实例不支持记忆功能"}`

---

## 三、消息通知接口

### `GET /openclaw/notifications`

获取当前用户的通知列表（分页），支持按已读状态和消息类别过滤。

- **权限：** 登录用户
- **响应：** 始终返回 JSON
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码（默认 1） |
| page_size | int | 否 | 每页条数（默认 20） |
| is_read | string | 否 | 已读状态过滤：`true`=仅已读，`false`=仅未读，不传=全部 |
| category | string | 否 | 消息类别过滤：`success`/`error`/`notice`，不传=全部 |

- **JSON 响应：**

```json
{
  "notifications": [
    {
      "ID": 1,
      "Type": "admin_delete",
      "Category": "notice",
      "Title": "实例已被管理员删除",
      "Message": "my-instance 已被管理员删除",
      "InstanceName": "my-instance",
      "ErrorDetail": "",
      "IsRead": false,
      "CreatedAt": "2026-03-26T10:00:00Z",
      "ReadAt": null
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 5
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| notifications[].ID | uint | 通知 ID |
| notifications[].Type | string | 通知类型（见下方触发场景表） |
| notifications[].Category | string | 消息类别：`success`（成功）/ `error`（错误）/ `notice`（通知） |
| notifications[].Title | string | 通知标题 |
| notifications[].Message | string | 通知内容文本 |
| notifications[].InstanceName | string | 关联实例名称 |
| notifications[].ErrorDetail | string | 错误详情 JSON（仅 `error` 类有值），含 `error`/`detail`/`request_id`/`instance_id` |
| notifications[].IsRead | bool | 是否已读 |
| notifications[].CreatedAt | string | 创建时间（ISO 8601） |
| notifications[].ReadAt | string\|null | 已读时间（ISO 8601），未读时为 `null` |
| page | int | 当前页码 |
| page_size | int | 每页条数 |
| total | int | 总条数 |

失败时返回错误信息：

- `400 {"error": "category 参数值无效，仅支持 success/error/notice"}`
- `401 {"error": "未登录"}`
- `500 {"error": "查询通知失败: ..."}`

### `POST /openclaw/notifications/read`

标记通知为已读。支持单条已读、全部已读、按类别批量已读。

- **权限：** 登录用户
- **Content-Type：** `application/json`
- **响应：** 始终返回 JSON
- **请求体：**

```json
// 单条已读
{"id": 1}

// 全部已读
{"id": 0}

// 按类别批量已读（仅标记 error 类全部已读）
{"id": 0, "category": "error"}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 否 | 通知 ID。不传或传 `0` 时标记全部已读；传具体 ID 时标记单条已读 |
| category | string | 否 | 消息类别，仅 `id=0` 时生效。`success`/`error`/`notice`，不传=所有类别 |

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`400 {"error": "请求体格式错误"}` / `400 {"error": "category 参数值无效..."}` / `401 {"error": "未登录"}` / `500 {"error": "..."}`

### `GET /openclaw/notifications/count`

获取当前用户的未读通知数量，含按类别分类计数。

- **权限：** 登录用户
- **响应：** 始终返回 JSON
- **JSON 响应：**

```json
{
  "unread_count": 5,
  "unread_by_category": {
    "success": 1,
    "error": 3,
    "notice": 1
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| unread_count | int | 未读通知总数 |
| unread_by_category | object | 按类别分类的未读计数，key 为 `success`/`error`/`notice` |

失败时返回错误信息：

- `401 {"error": "未登录"}`
- `500 {"error": "查询未读数量失败: ..."}`

**消息类别说明：**

| 类别 | 值 | 说明 |
|------|---|------|
| 成功 | `success` | 操作成功：实例创建/删除/升级/重装完成 |
| 错误 | `error` | 操作失败：创建/重装/升级失败、模型/通道配置失败、配额超限 |
| 通知 | `notice` | 系统通知：管理员删除实例、实例被外部销毁 |

**通知触发场景：**

| 场景 | type | category | 触发位置 |
|------|------|----------|--------|
| 实例创建成功 | `instance_create_success` | `success` | Agent 就绪（AgentChecker） |
| 实例删除成功 | `instance_delete_success` | `success` | `POST /openclaw/delete` |
| 实例升级完成 | `instance_upgrade_success` | `success` | `POST /openclaw/upgrade` 异步流程 |
| 实例重装完成 | `instance_reinstall_success` | `success` | Agent 就绪（AgentChecker） |
| 实例创建失败 | `instance_create_failed` | `error` | `POST /openclaw/create` |
| 实例重装失败 | `instance_reinstall_failed` | `error` | `POST /openclaw/reset` |
| 实例升级失败 | `instance_upgrade_failed` | `error` | `POST /openclaw/upgrade` 异步流程 |
| 模型配置失败 | `model_config_failed` | `error` | `POST /openclaw/set-model` |
| 通道配置失败 | `channel_config_failed` | `error` | `POST /openclaw/set-channel` |
| Token 配额超限 | `quota_exceeded` | `error` | `POST /v1/chat/completions`（自然日去重） |
| 管理员删除实例 | `admin_delete` | `notice` | `POST /admin/instances/delete` |
| 实例被外部销毁 | `external_destroy` | `notice` | `GET /openclaw/status` 副作用 |

> **配额超限通知**：同一用户每自然日最多收到一条，次日配额重置后如再次超限会收到新通知。
>
> **数据保留策略**：通知记录保留 30 天，超期后由后台定时任务物理删除（以 `created_at` 为准）。

### `POST /openclaw/notifications/delete`

删除通知。支持三种模式：单条删除、批量删除、全部删除。

- **权限：** 登录用户（仅可删除自己的通知）
- **响应：** 始终返回 JSON
- **请求体：**

```json
// 单条删除
{"id": 42}

// 批量删除（最多 100 条）
{"ids": [42, 43, 44]}

// 全部删除
{}

// 按类别全部删除
{"category": "notice"}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 否 | 通知 ID，指定时删除单条 |
| ids | []uint | 否 | 通知 ID 列表，指定时批量删除（最多 100 条） |
| category | string | 否 | 消息类别过滤，仅全部删除时生效，可选 `success`/`error`/`notice` |

**优先级**：`id` > `ids` > 全部删除。即 `id > 0` 时忽略 `ids`；`ids` 非空时忽略 `category`。

- **成功响应：**

```json
{"ok": true, "deleted": 3}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 操作是否成功 |
| deleted | int | 实际删除的通知数量 |

- **失败响应：**
  - `400 {"error": "请求体格式错误"}`
  - `400 {"error": "category 参数值无效，仅支持 success/error/notice"}`
  - `400 {"error": "单次最多删除 100 条通知"}`
  - `401 {"error": "未登录"}`
  - `500 {"error": "删除通知失败: ..."}`

---

### `POST /openclaw/approve`

在实例上执行配对审批操作（通过 TAT 远程执行 `approve.sh`）。

- **权限：** 登录用户
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |
| module | string | 否 | 审批模块，缺省值 `feishu` |
| code | string | 是 | 审批码 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **响应：** `application/json`（始终 JSON），脚本输出的 JSON 或 `{"error":"..."}`

### `GET /openclaw/service-status`

查询实例上的服务运行状态（通过 TAT 远程执行 `check_service.sh`）。

- **权限：** 登录用户
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID（Query 参数）；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`，Query 参数）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **响应：** `application/json`（始终 JSON），脚本输出的 JSON 或 `{"error":"..."}`

### `GET /openclaw/check-openclaw-port`

检查实例上 Agent 运行状态。按实例 `agent_type` 分派不同脚本：OpenClaw 走 `check_openclaw_ready.sh`（从 openclaw 配置解析端口、ss 验证监听、健康探测），Hermes/LightclawACE 走对应的 `check_*_ready.sh`。自定义类型见下文。

- **权限：** 登录用户
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID（Query 参数）；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`，Query 参数）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **响应：** `application/json`（始终 JSON）

> **自定义类型分派规则：**
> - 声明了 `compatible_with`（兼容 `openclaw`/`hermes`/`lightclawace`）的自定义类型：走兼容目标的 `check_*_ready.sh` 脚本，脚本成败由对应运行时的端口/进程布局决定。
> - 未声明 `compatible_with` 的自定义类型：Hatchery 不定义 `check_ready` 业务语义，直接以后台 `agent_checker`（TAT `DescribeAutomationAgentStatus`）同步到 DB 的 `agent_ready` 字段为准——`agent_ready==1` 返回 `{"running": true}`，否则返回 `{"running": false}`。

运行正常：

```json
{"running": true}
```

未运行（多种原因）：

```json
{"running": false, "reason": "config file not found"}
```

```json
{"running": false, "reason": "port not configured in openclaw.json"}
```

```json
{"running": false, "reason": "health check failed on port 8080"}
```

命令下发失败时也返回：

```json
{"running": false}
```

### `GET /openclaw/skills`

获取实例当前实际存在的技能列表。支持 OpenClaw、Hermes 和 LightClaw ACE；后端通过安装目录名识别稳定 slug，保留 Agent 返回的展示 `name`，并明确返回是否支持卸载。

- **权限：** 登录用户（仅可查看自己的实例）
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID（Query 参数） |

- **响应：** `application/json`（始终 JSON）

```json
[
  {
    "slug": "self-improving-agent",
    "name": "self-improvement",
    "description": "Admin-distributed skill",
    "eligible": true,
    "can_uninstall": true,
    "version": "1.0.0",
    "latest_version": "2.0.0",
    "update_available": true
  },
  {
    "slug": "manual-skill",
    "name": "manual-skill",
    "description": "Not managed by Hatchery",
    "eligible": true,
    "can_uninstall": true
  },
  {
    "slug": "cron",
    "name": "cron",
    "description": "Runtime builtin skill",
    "eligible": true,
    "can_uninstall": false
  }
]
```

`slug` 和布尔值 `can_uninstall` 出现在所有实时技能项中。`can_uninstall=true` 表示技能命中当前 Agent 的用户可管理技能目录；仅由运行时 CLI 暴露、没有用户目录的内建技能为 `false`。`version`、`latest_version`、`update_available` 仅出现在由 Admin 成功下发、当前未被成功卸载且仍存在于 Agent 实时列表中的 Public/Enterprise 技能上；数据库中存在但 Agent 实时列表缺失的技能不会被补造。

当前版本按同实例同 slug 的成功下发/卸载事件顺序还原，pending/failed 记录不改变版本。`update_available` 仅在 `latest_version` 严格高于 `version` 时为 `true`。Public 最新版本查询失败时，有过期缓存则降级使用；无缓存时返回空 `latest_version` 和 `false`。

`can_uninstall` 表示正常情况下的卸载能力，不代表本次操作一定成功；权限、文件系统或远程执行异常仍可能使卸载脚本失败。

### `POST /openclaw/add-skill`

为实例安装技能。默认安装公共源 Skill；仅当显式指定 `source=enterprise` 时安装企业 Skill（从 SMH 下载安装包）。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| skill_name | string | 是 | 技能名称/slug；`source=public` 时按公共源安装，`source=enterprise` 时按企业 Skill slug 查找 |
| source | string | 否 | 技能来源：`public` 或 `enterprise`，默认 `public` |
| version | string | 否 | 企业 Skill 版本；仅 `source=enterprise` 时有效，不传则安装该 slug 最新版本 |
| agent_id | string | 否 | OpenClaw 多 Agent ID；须匹配 OpenClaw 规则 `[A-Za-z0-9][A-Za-z0-9_-]{0,63}`，传入时安装到对应已存在的 Agent 工作区 |

- **响应：** `application/json`（始终 JSON）

**CVM 实例分支（不变）：**

```json
{"ok": true, "message": "技能安装成功"}
```

**本地 agent 实例分支（🆕）：**

```json
{"ok": true, "message": "已下发，请等待客户端拉取"}
```

- **本地实例失败响应：**
  - `400 {"error": "本地实例未接入"}` — local 实例 `status=stopped`（reporter 心跳丢失）

### `POST /openclaw/update-skill`

同步更新实例上由 Admin 下发的 Public/Enterprise 技能。仅支持 CVM OpenClaw、Hermes 和 LightClaw ACE 实例。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| slug | string | 是 | Admin 下发技能的稳定 slug；须匹配 `[A-Za-z0-9][A-Za-z0-9._-]{0,127}` 且不能包含 `..` |

- **前置条件：**
  - 实例必须为 running，且 Agent 类型支持技能管理
  - Local Agent、无 Admin 下发记录的技能和已卸载技能不支持更新
  - Enterprise 技能更新时重新校验最新版本对当前用户的可见性
  - Public 技能必须成功刷新或命中新鲜版本缓存，更新不会使用过期缓存

更新成功：

```json
{
  "slug": "self-improving-agent",
  "updated": true,
  "old_version": "1.0.0",
  "version": "2.0.0"
}
```

`old_version` 是操作前版本，`version` 是同步操作完成后的实际版本。最新可用版本由技能列表接口的 `latest_version` 返回，更新响应不重复返回。

已经是最新版时不创建 task/record，也不执行 Agent 脚本：

```json
{
  "slug": "self-improving-agent",
  "updated": false,
  "old_version": "2.0.0",
  "version": "2.0.0"
}
```

- **主要错误：**
  - `400`：缺少参数、实例不属于 CVM
  - `403`：Agent 类型不支持技能功能
  - `404`：技能未安装、没有有效 Admin 下发记录，或 Enterprise 最新版本对用户不可见
  - `409`：实例非 running，或同技能正在被其他操作处理
  - `502`：Public 仓库最新版本查询失败且不可降级
  - `500`：制品准备、Agent 安装脚本或任务状态持久化失败

### `POST /openclaw/uninstall-skill`

同步物理删除指定 slug 的技能。支持 Admin 下发的 Public/Enterprise 技能，以及 `can_uninstall=true` 的用户自行安装或手工复制技能；仅支持 CVM OpenClaw、Hermes 和 LightClaw ACE 实例。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| slug | string | 是 | 运行时技能的稳定目录 slug；须匹配 `[A-Za-z0-9][A-Za-z0-9._-]{0,127}` 且不能包含 `..` |

卸载成功：

```json
{
  "slug": "self-improving-agent",
  "uninstalled": true,
  "version": "2.0.0"
}
```

Admin 下发技能成功卸载后写入 task/record，并返回已知 `version`。其他运行时技能只执行物理卸载，不创建 Admin 分发记录，也不补造未知版本：

```json
{
  "slug": "manual-skill",
  "uninstalled": true
}
```

无 Admin 下发记录的技能不查询 Agent 实时列表；获取 runtime+slug 技能锁后直接执行一次对应 Agent 卸载脚本。脚本自身按目录不存在幂等成功，因此首次和重复请求在脚本成功时都返回：

```json
{
  "slug": "manual-skill",
  "uninstalled": true
}
```

主要错误与更新接口的实例准入、锁冲突和执行失败口径一致；卸载不依赖 Public 仓库版本查询。

### `POST /openclaw/add-plugin`

为实例安装插件（通过 TAT 远程执行 `add_plugin.sh`）。仅支持 OpenClaw 类型实例。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| plugin | string | 是 | 插件名称（如 `openclaw-weixin`），格式需匹配 `^[a-zA-Z0-9_-]+$` |

- **前置校验：**
  - 实例必须支持插件安装（`checkInstanceSupportsPlugin`），非 OpenClaw 类型返回 `403`
  - 插件名称格式校验，不合法返回 `400`

- **响应：** `application/json`（始终 JSON）
  - 成功：`{"ok": "插件安装成功"}`
  - 失败：`400 {"error": "plugin 不能为空"}` / `400 {"error": "插件名称格式不合法"}` / `403 {"error": "..."}` / `500 {"error": "..."}`

> **微信插件版本**：当插件名包含 `openclaw-weixin` 时，后端会从环境变量 `WEIXIN_VERSION` 获取版本号并拼接到安装目标中。

### `GET /openclaw/install-skills`

查询实例的技能包安装状态。同时支持 CVM 实例与本地 agent 实例（响应结构按 `source` 不同）。

- **权限：** 登录用户（仅可查看自己的实例）
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID（Query 参数） |

- **响应：** `application/json`（始终 JSON）

#### CVM 实例分支响应（不变）

```json
{
  "instance_id": 5,
  "skills": [
    {
      "id": 1,
      "instance_id": 5,
      "name": "openclaw-tavily-search",
      "slug": "openclaw-tavily-search",
      "version": "0.1.0",
      "cos_zip_key": "skill-bundles/通用技能包/openclaw-tavily-search/openclaw-tavily-search-0.1.0.zip",
      "install_status": 2,
      "error_message": "",
      "created_at": "2026-03-31T10:00:00Z",
      "updated_at": "2026-03-31T10:05:00Z"
    }
  ],
  "total": 1
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| instance_id | uint | 实例数据库 ID |
| skills | array | 技能安装记录列表 |
| skills[].install_status | int | 安装状态：`0`=未安装、`1`=安装中、`2`=安装成功、`3`=安装失败、`4`=已取消 |
| skills[].error_message | string | 失败原因（仅 install_status=3 时有值） |
| total | int | 技能总数 |

#### 本地 agent 实例分支响应（🆕）

```json
{
  "instance_id": 101,
  "skills": [
    {
      "slug": "weather",
      "name": "Weather",
      "version": "1.2.3",
      "install_status": "success",
      "error_message": "",
      "source": "public",
      "installed_at": "2026-06-23T10:00:00Z"
    },
    {
      "slug": "trace-helper",
      "name": "Trace Helper",
      "version": "0.4.0",
      "install_status": "installing",
      "error_message": "",
      "source": "enterprise",
      "installed_at": null
    }
  ],
  "total": 2
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| skills[].slug | string | skill 唯一标识 |
| skills[].name | string | 展示名（取 `local_instance_skills.display_name`，缺省回查 `skills.name`） |
| skills[].version | string | 目标版本 |
| skills[].install_status | string | 状态枚举：`installing` / `success` / `failed` |
| skills[].error_message | string | 失败原因（仅 `install_status=failed` 时非空） |
| skills[].source | string | skill 来源：`public` / `enterprise` / `local` |
| skills[].installed_at | string \| null | 安装成功时间（RFC3339）；其他状态为 `null` |

  - `400 {"error": "本接口仅适用于本地实例"}` — CVM 实例调用
  - `400 {"error": "参数 record_id 不能为空"}`
  - `400 {"error": "参数 record_id 无效"}` — 非数字 / 0
  - `403 {"error": "无权限访问"}` — 实例不属于当前用户
  - `404 {"error": "实例不存在"}`

### `POST /openclaw/retry-failed-skills`

重试安装失败的技能。将该实例所有 install_status=3（失败）的记录重置为 0（未安装），然后异步重新安装。

- **权限：** 登录用户（仅可操作自己的实例）
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID（Query 参数） |

- **响应：** `application/json`（始终 JSON）

```json
{"ok": true, "retry_count": 3}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| retry_count | int | 重试的技能数量，为 0 表示没有失败的技能需要重试 |

失败时返回错误信息：

- `400 {"error": "该实例无关联的 CVM"}`
- `405 {"error": "请求方法不允许"}`

### `POST /openclaw/cancel-failed-skills`

取消安装失败的技能。将该实例所有 install_status=3（失败）的记录标记为 4（已取消）。

- **权限：** 登录用户（仅可操作自己的实例）
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID（Query 参数） |

- **响应：** `application/json`（始终 JSON）

```json
{"ok": true, "cancel_count": 2}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| cancel_count | int | 取消的技能数量 |

失败时返回错误信息：

- `405 {"error": "请求方法不允许"}`

为实例安装插件（通过 TAT 远程执行 `add_plugin.sh`）。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| plugin | string | 是 | 插件名称 |

- **响应：** `application/json`（始终 JSON），`{"ok":"插件安装成功"}` 或 `{"error":"..."}`

### `GET /openclaw/channels`

获取渠道列表。不传 `id` 时返回数据库中所有渠道（AIChannel）列表（含启用和禁用的），包括预定义通道和自定义通道；传 `id` 时通过 TAT 远程执行 `list_channels.sh` 获取实例上已配置的渠道。

- **权限：** 登录用户
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 否 | 实例数据库 ID（Query 参数）。不传则返回启用的渠道列表 |
| agent_id | uint | 否 | 实例 ID（需属于当前用户）。不传 `id` 时生效，按实例的 group_id 过滤可见通道 |

- **响应：** `application/json`（始终 JSON）
  - 不传 `id`：返回所有渠道数组（含启用和禁用的，前端根据 `Enabled` 字段自行过滤）。预定义通道的 `params` 来自内置定义，自定义通道的 `params` 来自 `CustomConfig` 中的 `cred_fields`：

```json
[
  {
    "ID": 1,
    "CreatedAt": "...",
    "UpdatedAt": "...",
    "DeletedAt": null,
    "ChannelID": "qqbot",
    "Name": "QQ",
    "Enabled": true,
    "Custom": false,
    "CustomConfig": "",
    "params": [
      { "key": "app_id", "label": "机器人App ID" },
      { "key": "app_secret", "label": "机器人App Secret" }
    ],
    "AgentTypes": ["openclaw", "hermes"]
  },
  {
    "ID": 5,
    "CreatedAt": "...",
    "UpdatedAt": "...",
    "DeletedAt": null,
    "ChannelID": "hworktalk",
    "Name": "海尔工作通",
    "Enabled": true,
    "Custom": true,
    "CustomConfig": "{\"server\":{\"url\":\"wss://example.com/ws\"},\"cred_fields\":[{\"key\":\"token\",\"label\":\"接入Token\"}]}",
    "params": [
      { "key": "token", "label": "接入Token" }
    ],
    "AgentTypes": ["openclaw"]
  }
]
```

  - 传 `id`：返回实例上通道配置，响应被包装为结构化对象：

```json
{
  "agent_type": "openclaw",
  "agent_type_supported_channels": ["wecom", "feishu", "qqbot", "dingtalk", "openclaw-weixin", "wecom_app"],
  "channels": {
    "wecom": { "enabled": true, "accounts": { "default": { "corp_id": "xxx" } } },
    "qqbot": { "enabled": false }
  }
}
```

  | 字段 | 类型 | 说明 |
  |------|------|------|
  | `agent_type` | string | 实例的智能体类型（`openclaw`/`hermes`/`lightclawace`） |
  | `agent_type_supported_channels` | string[] | 当前 Agent 类型支持的通道列表 |
  | `channels` | object | 各通道的配置详情，key 为通道 ID |

  > **注意**：后端会对原始输出进行 `normalizeWecomShape` 归一化处理，将 `agent` 字段重命名为 `wecom_app`，将 ACE 类型的通道结构归一化为标准格式。

  - 失败：`400 {"error": "无效的 id"}`、`400 {"error": "缺少参数 id 或 instance_id"}`、`404 {"error": "实例不存在"}`、`500 {"error": "解析 list_channels 脚本失败: ..."}`、`500 {"error": "解析通道列表失败: ..."}`

### `POST /openclaw/proxy/prepare`

为实例创建或刷新通用反向代理入口。当前用于 Microsoft Teams / LINE webhook endpoint 预生成。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| kind | string | 否 | 代理类型；当前支持 `teams`、`line`，默认 `teams` |

- **响应：** `application/json`

```json
{
  "ok": true,
  "kind": "teams",
  "route_id": "opaque-route-id",
  "endpoint": "https://tenant.example.com/api/proxy/opaque-route-id/api/messages"
}
```

### `POST /openclaw/set-channel`

为实例配置渠道（通过 TAT 远程执行 `set_channel.sh`）。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| channel | string | 是 | 渠道类型（预定义通道 ID 或自定义通道 ID） |
| key | string[] | 是 | 配置参数名数组（可重复，与 value 一一对应） |
| value | string[] | 是 | 配置参数值数组（可重复，与 key 一一对应） |

**预定义通道参数定义：**

- **qqbot**：`app_id`（机器人App ID）、`app_secret`（机器人App Secret）
- **wecom**：`bot_id`（机器人botId）、`secret`（机器人secret）
- **feishu**：`app_id`（应用App ID）、`app_secret`（应用App Secret）
- **slack**：`app_token`（App-Level Token）、`bot_token`（Bot User OAuth Token）
- **ddingtalk**：`client_id`（应用Client ID）、`client_secret`（应用Client Secret）
- **discord**: `bot_token`（机器人 Token）、`user_id`（用户 ID）
- **msteams**：`app_id`（Azure App Client ID）、`app_secret`（Azure Client Secret）、`tenant_id`（Azure Tenant ID）。后端会自动创建/刷新通用代理入口，返回 `teams_endpoint`，用户应将其作为 Teams Bot Messaging Endpoint。
- **line**：`channel_access_token`（Channel Access Token，必填）、`channel_secret`（Channel Secret，必填）。后端会自动创建/刷新通用代理入口，返回 `proxy_endpoint`，用户应将其作为 LINE Webhook URL。缺少 `channel_access_token` 或 `channel_secret` 时返回 `400`。

**自定义通道：** 用户仅需提交 `cred_fields` 中定义的凭证参数（key/value）。后端从数据库读取该通道的 `CustomConfig.server`，**将其作为 openclaw.json 中该通道配置的 JSON 模板**：把模板中的 `{{key}}` 占位符替换为用户提交的凭证值（自动做 JSON 转义，防注入），渲染结果即为最终的 `channel_config`（管理员写什么结构就写入什么结构，不额外包装）。若模板顶层无 `enabled` 字段，后端自动补 `"enabled": true`。渲染后若不是合法 JSON 或存在未替换的占位符，返回 `400`。前端无需关心服务器地址等配置细节。

- **响应：** `application/json`（始终 JSON），`{"ok":true,"output":"..."}` 或 `{"error":"..."}`。当 `channel=msteams` 时，成功响应额外包含 `proxy_route_id`、`proxy_endpoint`、`teams_endpoint`。当 `channel=line` 时，成功响应额外包含 `proxy_route_id`、`proxy_endpoint`。

### `POST /openclaw/del-channel`

删除实例上已配置的渠道（通过 TAT 远程执行脚本）。自定义通道会根据 DB 中 `CustomConfig.server.deleteFeature` 选择脚本，预定义通道使用 `del_channel.sh`。`openclaw_whatsapp`（自定义通道配对码模式）专用 `del_channel_whatsapp.sh`（仅 OpenClaw 支持）。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| channel | string | 是 | 渠道类型。预定义通道 ID（如 `whatsapp` 内置扫码通道）或自定义通道 ID（如 `openclaw_whatsapp` 配对码模式）。两者是独立的 channel_id，互不映射、互不覆盖 |

- **响应：** `application/json`（始终 JSON），`{"ok":true,"output":"..."}` 或 `{"error":"..."}`

### `GET /openclaw/auto-channel`

自动配置渠道（SSE 流式）。支持飞书机器人、微信机器人、WhatsApp 内置扫码通道、WhatsApp 配对码通道，以及任意配置了 `pairingMode=true` 的自定义通道，按实例 `agent_type` 分派到不同脚本。

> **WhatsApp 两种独立接入方式：** WhatsApp 存在两套互不冲突、可同时使用的接入方式，通过不同 `channel_id` 区分：
> - `whatsapp`：**内置扫码登录通道**（预定义在 `autoChannelFeature` map 中），基于扫码登录，无需手机号，走 `whatsapp_bot_creator.sh`。
> - `openclaw_whatsapp`：**自定义通道配对码模式**（管理员在自定义通道中配置，DB 记录为 Custom 通道），需要手机号，由 `CustomConfig.server` 驱动，走 `set_channel_whatsapp.sh` / `del_channel_whatsapp.sh`。
>
> 两者 `channel_id` 不同，后端不做别名映射，各自独立生效。

> **自定义通道准入：** 对于自定义通道，后端从 `CustomConfig.server` 解析 `ServerConfig`（并按 `DefaultsForChannel` 填充预设），仅当 `pairingMode=true` 时才允许走 auto-channel；未配置或 `pairingMode=false` 时返回 `400`。相关语义字段：`pairingMode`（是否配对码模式）、`autoFeature`（自动配置脚本 feature）、`autoTimeout`（超时秒数）、`phoneRequired`（是否需要 phone 参数）、`phonePattern`（手机号校验正则）、`egressRequired`（是否需要出站网络诊断）。

- **权限：** 登录用户（仅可操作自己的实例）
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID（Query 参数） |
| channel | string | 否 | 渠道类型：`feishu`、`lark`、`openclaw-weixin`、`whatsapp`（内置扫码通道）、`qqbot`（已废弃），或任意配置了 `pairingMode=true` 的自定义通道 channel_id（如 `openclaw_whatsapp` 配对码模式）。缺省为 `qqbot` |
| phone | string | 否 | 手机号（仅 `openclaw_whatsapp` 等配对码模式自定义通道必填，要求 1-9 开头后跟 6-14 位数字，不含 + 号，如 `85266803489`；内置 `whatsapp` 扫码通道无需此参数） |

- **响应：** `text/event-stream`（SSE，始终流式）

**QQ 渠道事件（已废弃）：**

- **事件 `qrcode`：** `{"url":"二维码链接"}`
- **事件 `done`：** 配置结果 JSON
- **事件 `fail`：** `{"message":"错误信息"}`

**飞书 / 微信 / WhatsApp 渠道事件：**

脚本输出结构化 JSON 行，后端按 `action` 字段分发为不同 SSE 事件，data 为脚本原始 JSON（含新增 `mode` 字段）：

| SSE 事件名 | 触发条件 | data 格式 |
|-----------|---------|----------|
| qrcode | `action=show_qrcode` | 见下方 show_qrcode 说明 |
| log | `action=log` | `{"action":"log","level":"info\|warn\|error","step":"...","message":"..."}` |
| progress | `action=progress` | `{"action":"progress","level":"info","step":"...","message":"..."}` |
| finish | `action=finish` | `{"action":"finish","level":"success\|error","message":"...","data":{...}}` |
| fail | 后端诊断失败（如安全组出站规则封锁） | `{"message":"错误信息"}` |

**WhatsApp 配对码事件：**

脚本输出 `action=show_pairing_code`，后端路由为 `pairing_code` 事件：

| SSE 事件名 | 触发条件 | data 格式 |
|-----------|---------|----------|
| pairing_code | `action=show_pairing_code` | `{"action":"show_pairing_code","code":"S3QADVEG","expires_in":60,"message":"请在手机 WhatsApp 输入配对码"}` |
| progress | `action=progress` | 同上 |
| finish | `action=finish` | 同上 |
| fail | 后端诊断失败（如安全组出站规则封锁） | `{"message":"错误信息"}` |

**`show_qrcode` 事件格式（`qrcode` SSE 事件的 data）：**

新增 `mode` 字段（增量，不影响原有 `action`/`content` 字段），供前端判断如何渲染二维码：

| mode | content 格式 | 渲染方式 | 适用场景 |
|------|------------|---------|---------|
| qrlogin | JSON 字符串 `{"qrlogin":{"token":"<short_token>"}}` | 将整个 content 字符串编码为二维码 | OpenClaw 飞书（飞书 App 识别 qrlogin 协议） |
| url | 裸 URL 字符串（`https://...`） | 将 URL 直接编码为二维码（`QRCodeCanvas`） | Hermes 飞书、Hermes/ACE 微信 |
| ascii_art | UTF8 字符画字符串 | `<pre>` 标签渲染 | OpenClaw 微信 |

示例：
```json
// Hermes 飞书
{"action": "show_qrcode", "mode": "url", "content": "https://open.feishu.cn/page/launcher?from=hermes&user_code=XXXX"}

// OpenClaw 飞书
{"action": "show_qrcode", "mode": "qrlogin", "content": "{\"qrlogin\":{\"token\":\"abc123\"}}"}

// Hermes/ACE 微信
{"action": "show_qrcode", "mode": "url", "content": "https://liteapp.weixin.qq.com/q/7GiQu1?qrcode=XXXX&bot_type=3"}

// OpenClaw 微信
{"action": "show_qrcode", "mode": "ascii_art", "content": "▄▀█▀▄ ...（UTF8字符画）"}

// WhatsApp
{"action": "show_qrcode", "mode": "url", "content": "https://wa.me/settings/linked_devices#2@dGHmxxxxx"}
```

飞书成功的完整事件序列示例（OpenClaw）：
```json
{"action": "log", "level": "info", "step": "init", "message": "检查并安装依赖...", "ts": 1773910740}
{"action": "log", "level": "success", "step": "init", "message": "依赖就绪", "ts": 1773910741}
{"action": "log", "level": "info", "step": "login", "message": "启动浏览器，获取二维码...", "ts": 1773910741}
{"action": "show_qrcode", "mode": "qrlogin", "level": "info", "step": "login", "message": "请扫码登录飞书", "content": "{\"qrlogin\": {\"token\": \"token1\"}}"}
{"action": "log", "level": "success", "step": "login", "message": "扫码登录成功!", "ts": 1773909693}
{"action": "finish", "level": "success", "step": "finish", "message": "创建完成", "data": {"app_id": "xxxxx", "app_secret": "xxxx", "bot_name": "xxxx"}}
```

WhatsApp 配对码成功的完整事件序列示例（OpenClaw）：
```json
{"action":"progress","message":"正在连接 WhatsApp..."}
{"action":"show_pairing_code","code":"S3QADVEG","expires_in":60,"message":"请在手机输入配对码"}
{"action":"progress","message":"正在完成关联..."}
{"action":"finish","success":true,"message":"WhatsApp 通道关联成功"}
```

### `GET /openclaw/models`

获取可用的 AI 模型列表（已启用的模型）。

- **权限：** 登录用户
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| agent_id | uint | 否 | 实例 ID（需属于当前用户）。传入后按实例的 group_id 过滤可见模型 |

- **JSON 响应：**

```json
{
  "ok": true,
  "models": [
    {
      "id": 1,
      "provider": "openai",
      "model_id": "gpt-4o",
      "model_name": "gpt-4o",
      "model_type": "openai-completions",
      "input_types": ["text", "image"],
      "context_len": 128000,
      "max_tokens": 8192,
      "custom_http_headers": {
        "key": "value"
      },
      "default": true
    }
  ]
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| ok | bool | 固定 `true` |
| models | array | 模型列表 |
| models[].id | uint | 模型数据库 ID |
| models[].provider | string | 提供商 |
| models[].model_id | string | 模型标识 |
| models[].model_name | string | 模型显示名称 |
| models[].model_type | string | 接口类型 |
| models[].input_types | string[] | 支持的输入类型（`text`、`image`） |
| models[].context_len | int | 上下文长度 |
| models[].max_tokens | int | Agent 单次请求模型最大输出 Token 数 |
| models[].custom_http_headers | object | 自定义 HTTP 请求头（键值对） |
| models[].default | bool | 是否为默认模型，默认模型返回 `true`，非默认返回 `false` |

> **可见性过滤：** 返回的模型列表会根据当前用户的分组进行过滤。`visibility_type=all` 的模型对所有用户可见；`visibility_type=group` 的模型仅对关联分组中的用户可见。用户不在任何可见分组中的模型不会出现在列表中。

> **自定义模型占位项（`model_id=custom`）：** 当实例允许使用自定义模型时，列表会返回该占位项；不允许时则不返回。前端据此决定是否展示「自定义模型」配置入口。建议查询时传 `agent_id` 以按实例判定。

### `POST /openclaw/models/connectivity`

- **权限：** 登录用户
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 否 | 模型 ID（Query 参数） |

- **请求体：**

```json
{
    "url": "https://api.openai.com/v1",
    "api_key": "<KEY>",
    "model_type": "openai-completions",
    "model": "gpt-4o"
}
```

| 请求体字段 | 类型 | 必填 | 说明 |
|----------|------|-------|------|
| url | string | 是 | API 地址 |
| api_key | string | 是 |  API 密钥 |
| model_type | string | 是 |  接口类型，枚举值：`openai-completions` 或 `anthropic-messages` |
| model | string | 是 |  模型名称 |

- **请求说明：**
  1. 按已保存模型 ID 探测——用于已存在模型的健康检查：POST /openclaw/models/connectivity?id=42

  2. 用临时凭证探测——常用于新增/编辑模型表单未保存即试连：POST /openclaw/models/connectivity Content-Type: application/json {"url":"https://api.openai.com/v1","api_key":"sk-...","model_type":"openai-completions","model":"gpt-4o"}

  > **探活方式：** 使用 chat 探活（发送极简对话请求，max_tokens=1），同时验证 API 地址可达性、API Key 有效性及模型 ID 正确性。

- **JSON 响应：**

模型无法连通的响应：

```json
{
  "ok":false, 
  "kind":"invalid_api_key", 
  "message":"...",
	"status_code":401, 
  "snippet":"...", 
  "latency_ms":120
}
```

模型可以连通的响应

```json
{
  "ok":true, 
  "latency_ms":234
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| ok | bool | 模型是否可以连通 |
| kind | string | 错误类型 |
| message | string | 错误详细信息 |
| status_code | int | 后端检测模型连通性时 provider 返回的 HTTP 状态码 |
| snippet | string | 响应片段 |
| latency_ms | int | 延迟（毫秒） |

错误类型与信息对应：
| 类型 | 信息 |
|-------|-------|
| network_unreachable | 网络不通或上游地址无法访问 |
| invalid_api_key | API Key 无效 |
| forbidden | 凭证有效但被拒绝（账户/区域/模型未授权） |
| rate_limited | 上游限流，请稍后重试 |
| upstream_server_error | 上游服务异常 |
| upstream_client_error | 上游拒绝请求 |

- **错误响应：**

- `400 {"error": "URL 格式无效"}`
- `400 {"error": "URL 必须以 http:// 或 https:// 开头"}`
- `400 {"error": "接口类型无效，仅支持 openai-completions 或 anthropic-messages"}`
- `400 {"error": "请求体格式错误，应为 JSON"}`
- `400 {"error": "id 参数非法"}`
- `400 {"error": "URL 不能为空"}`
- `400 {"error": "model_type 不能为空"}`
- `400 {"error": "api_key 不能为空"}`
- `400 {"error": "model 不能为空"}`
- `403 {"error": "无权限访问"}`
- `404 {"error": "模型不存在"}`
- `405 {"error": "method not allowed"}`

### `POST /openclaw/set-model`

为实例配置 AI 模型（通过 TAT 远程执行 `set_model` 脚本，**按 agent_type 分派**）。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数（预配置模型）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| ai_model_id | uint | 是 | AI 模型数据库 ID（非零） |
| instance_model_id | uint | 否 | 指定要更新的 `instance_models.id`。缺省时保持旧行为：更新 primary；传入时精确更新该绑定（OpenClaw 可用于更新某个 fallback） |

- **参数（自定义模型，`ai_model_id=0`）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|----|------|
| id | uint | 是  | 实例数据库 ID |
| ai_model_id | uint | 是  | 固定为 `0` |
| instance_model_id | uint | 否 | 指定要更新的 `instance_models.id`。缺省时保持旧行为：更新 primary；传入时精确更新该绑定（OpenClaw 可用于更新某个 fallback） |
| provider | string | 否  | 模型提供商名称，未传时默认为 `自定义模型` |
| model_id | string | 是  | 模型 ID |
| model_name | string | 否  | 模型显示名称 |
| api_key | string | 是  | API Key |
| url | string | 否  | API 地址 |
| model_type | string | 是  | 接口类型，枚举值：`openai-completions` 或 `anthropic-messages` |
| input_types | string | 否  | 支持的输入类型，仅允许 `text`、`image`，可通过重复传递该参数来指定多个值（如 `input_types=text&input_types=image`）。缺省为 `text` |
| context_len | int | 否  | 上下文长度（默认 128000） |
| max_tokens | int | 否  | Agent 单次请求模型最大输出 Token 数（传入值小于等于 0 时默认 8192） |
| custom_http_headers| string | 否 | Agent 请求模型时的自定义 HTTP 头部，格式为 JSON 字符串，如 `{"Header-Key": "Header-Value"}`|

> **注意：** 使用自定义模型需管理后台开启「开放自定义模型」开关，且当前实例被允许使用自定义模型，否则提交自定义模型配置（`ai_model_id=0`）返回 `403`。自定义配置以 JSON 保存到 `Instance.CustomModelConfig` 字段，`provider` 字段未传时默认为 `自定义模型`，也可由前端传入自定义值。切换回预配置模型时，已保存的自定义配置不会被清除。

> **可见性校验：** 绑定模型时会检查用户对该模型的可见性。如果模型设置了按分组可见（`visibility_type=group`），而用户不在任何关联分组中，则返回 `403 {"error": "您无权使用该模型"}`。已应用到实例的模型不受可见范围变更影响。

> **Agent 类型支持：** OpenClaw / Hermes / LightclawACE 三种类型均支持手动配置模型（`AgentTypeSupportsModel=true`）。前置 `checkInstanceSupportsModel` guard 若命中不支持的类型，返回 `403 {"error": "<类型显示名> 类型实例不支持模型配置"}`。

> **预配置模型的代理改写：** 预配置模型（`ai_model_id>0`）在下发前会被改写为走 hatchery 代理，`provider="hatchery"`、`model_type="openai-completions"`、`baseUrl="${Domain}/v1"`、`apiKey` 用 `Instance.ProxyToken`（若存在）。自定义模型（`ai_model_id=0`）原样透传。

> **脚本按 agent_type 分派：** `ResolveScript("set_model", agent_type)` 的映射表：
>
> | agent_type | TAT 脚本 | 落地方式 |
> |---|---|---|
> | `openclaw` | `set_model.sh` | `jq` 写 `~/.openclaw/openclaw.json`，重启 `openclaw-gateway`（用户态 systemctl） |
> | `hermes` | `set_model_hermes.sh` | 调 `harness model set` CLI（harness 内部原子写 `~/.hermes/config.yaml` 并重启 gateway）；自定义模型时 `--name` 强制为 `custom` |
> | `lightclawace` | `set_model_ace.sh` | `jq` 写 `~/.lightclaw/lightclaw.json`（`activeLlm` 用冒号分隔 `provider:model`），重启 `lightclaw`（系统 systemctl） |
>
> TAT 参数契约三端统一：`{"value": <完整 provider JSON>, "provider": "...", "model": "..."}`。

> **指定绑定更新：** `instance_model_id` 为可选参数。不传时兼容旧语义，更新实例 primary 模型；传入时后端按 `instance_id + instance_model_id` 精确定位 `instance_models` 记录并保留原 `role`：目标是 `primary` 时同步更新 `instances.ai_model_id`，目标是 `fallback` 时只替换该 fallback，不影响 primary。该参数主要用于 OpenClaw 多模型实例；Hermes / LightclawACE 通常只有一条 primary 绑定，缺省即可。

- **响应：** `application/json`（始终 JSON），默认 primary 更新返回 `{"ok":true,"provider":"...","model_id":"..."}`；传 `instance_model_id` 更新指定绑定时返回 `{"ok":true,"role":"primary|fallback","instance_model_id":12,"binding_id":"...","provider":"...","model_id":"...","model_name":"..."}`；失败返回 `{"error":"..."}`。
- **错误响应：**
  - `400 {"error": "缺少或无效的 instance_model_id 参数"}` — `instance_model_id` 非正整数
  - `400 {"error": "目标模型记录不存在或不属于该实例"}` — 指定绑定不存在或不属于当前实例
  - `403 {"error": "您无权使用该模型"}` — 用户尝试绑定不在可见范围内的模型
  - `403 {"error": "自定义模型功能未开启"}` — 当前实例不允许使用自定义模型（`ai_model_id=0`）
  - `403 {"error": "<类型显示名> 类型实例不支持模型配置"}` — 实例类型不支持模型配置
  - `409 {"error": "该模型已绑定到此实例"}` — 指定绑定要改成的模型已绑定在同一实例的其他记录上
  - `400 {"error": "解析 set_model 脚本失败: ..."}` — agent_type 未在脚本分派表中
  - `500 {"error": "TAT 执行失败: ..."}` — 脚本执行失败（同时异步写入"模型配置失败"通知 `model_config_failed`）

---

### `POST /openclaw/add-model`

为实例添加 AI 模型绑定。OpenClaw 兼容运行时支持多模型：首个模型为 `primary`，后续模型为 `fallback`。Hermes / LightclawACE 只有单个激活模型，仅允许在实例尚无模型绑定时通过该接口设置第一个模型；已有绑定后再次调用返回 409。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数（预配置模型）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| ai_model_id | uint | 是 | AI 模型数据库 ID（非零） |

- **参数（自定义模型，`ai_model_id=0`）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| ai_model_id | uint | 是 | 固定为 `0` |
| provider | string | 否 | 模型提供商名称，未传时默认为 `自定义模型` |
| model_id | string | 是 | 模型 ID（业务键，与 `instance_id` 组成联合唯一索引） |
| model_name | string | 否 | 模型显示名称 |
| api_key | string | 是 | API Key |
| url | string | 否 | API 地址 |
| model_type | string | 是 | 接口类型，枚举值：`openai-completions` 或 `anthropic-messages` |
| input_types | string | 否 | 支持的输入类型（`text`、`image`），可重复传递；缺省为 `text` |
| context_len | int | 否 | 上下文长度（默认 128000） |
| max_tokens | int | 否 | Agent 单次请求模型最大输出 Token 数（传入值小于等于 0 时默认 8192） |
| custom_http_headers| string | 否 | Agent 请求模型时的自定义 HTTP 头部，格式为 JSON 字符串，如 `{"Header-Key": "Header-Value"}`|

- **响应：** `application/json`

```json
{
  "ok": true,
  "role": "primary",
  "instance_model_id": 11,
  "binding_id": "hatchery-glm-4-plus/glm-4-plus",
  "provider": "智谱AI",
  "model_id": "glm-4-plus",
  "model_name": "GLM-4 Plus"
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| ok | bool | 固定 `true` |
| role | string | `primary`（首个添加；Hermes / LightclawACE 仅有此值）或 `fallback`（仅 OpenClaw 兼容运行时） |
| instance_model_id | uint | `instance_models.id`，可直接用于后续 `del-model` / `switch-primary-model` 的 `instance_model_id` 参数 |
| binding_id | string | 绑定引用标识，格式 `providerKey/modelId`（预置为 `hatchery-{model_id}/{model_id}`，自定义为 `custom-{model_id}/{model_id}`） |
| provider | string | 模型提供商展示名 |
| model_id | string | 模型 ID |
| model_name | string | 模型显示名称 |

- **错误响应：**
  - `400 {"error": "缺少或无效的 ai_model_id 参数"}`
  - `400 {"error": "自定义模型缺少 model_id"}`
  - `403 {"error": "您无权使用该模型"}` — 用户对预置模型无可见性
  - `403 {"error": "自定义模型功能未开启"}` — 当前实例不允许使用自定义模型（`ai_model_id=0`）
  - `403 {"error": "...实例类型不支持..."}` — 实例 Agent 类型不支持模型配置
  - `409 {"error": "...仅支持单模型..."}` — Hermes / LightclawACE 等单模型运行时已有模型绑定，不能再添加 fallback
  - `409 {"error": "Agent 3.28 暂不支持多模型 fallback 功能，请升级到更高版本后再使用"}` — OpenClaw Agent 版本为 3.28.x 时被拦截（已知兼容性问题）
  - `409 {"error": "该模型已绑定到此实例"}` — 联合唯一键冲突（同一实例下相同 `ai_model_id` + `custom_model_id` 已存在）
  - `409 {"error": "该自定义模型已绑定到此实例"}` — 同一实例下相同 `model_id` 的自定义模型已存在
  - `500 {"error": "TAT 执行失败: ..."}`

> **联合唯一键：** `(instance_id, ai_model_id, custom_model_id)`。预置模型 `custom_model_id=""`，自定义模型 `ai_model_id=0` + `custom_model_id=<用户提交的 model_id>`。

---

### `POST /openclaw/switch-primary-model`

**多模型 Fallback v2.0**：将指定模型切换为主模型，原主模型自动降级为备选（`fallback`）。通过 TAT 远程执行 `switch_model.sh` 重写 `primary`/`fallbacks` 并重启 gateway。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| instance_model_id | uint | 是 | `instance_models.id`（目标模型的绑定记录 ID） |

- **响应：** `application/json`

```json
{
  "ok": true,
  "new_primary": {
    "binding_id": "hatchery-qwen-max/qwen-max",
    "instance_model_id": 42,
    "provider": "通义千问",
    "model_id": "qwen-max",
    "model_name": "Qwen Max",
    "role": "primary"
  },
  "demoted_to_fallback": {
    "binding_id": "hatchery-glm-4-plus/glm-4-plus",
    "instance_model_id": 41,
    "provider": "智谱AI",
    "model_id": "glm-4-plus",
    "model_name": "GLM-4 Plus",
    "role": "fallback"
  }
}
```

- **错误响应：**
  - `400 {"error": "缺少或无效的 instance_model_id 参数"}`
  - `400 {"error": "目标模型记录不存在或不属于该实例"}`
  - `400 {"error": "无法切换到自身：目标模型已是当前主模型"}`

> **原子性保证：** DB 层在事务中完成"旧 primary → fallback、目标 → primary、同步 `instances.ai_model_id`" 三步。TAT 调用在事务外执行，若 TAT 失败会**回滚 DB** 至事务前状态（重新将目标还原为 fallback，将旧 primary 还原），并记录日志和发送通知。

---

### `POST /openclaw/del-model`

**多模型 Fallback v2.0**：从实例删除指定的模型绑定。若删除的是主模型且存在其他绑定，自动将 `sort_order` 最小（最早添加）的 fallback 提升为 primary。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| instance_model_id | uint | 是 | `instance_models.id`（要删除的绑定记录 ID） |

- **响应：** `application/json`
  - **场景 A（删除备选模型）**：

    ```json
    {
      "ok": true,
      "was_primary": false,
      "current_primary": {
        "binding_id": "hatchery-glm-4-plus/glm-4-plus",
        "instance_model_id": 41,
        "provider": "智谱AI",
        "model_id": "glm-4-plus",
        "model_name": "GLM-4 Plus",
        "role": "primary"
      }
    }
    ```

  - **场景 B（删除主模型，存在其他备选）**：

    ```json
    {
      "ok": true,
      "was_primary": true,
      "promoted_model": {
        "binding_id": "hatchery-qwen-max/qwen-max",
        "instance_model_id": 42,
        "provider": "通义千问",
        "model_id": "qwen-max",
        "model_name": "Qwen Max",
        "role": "primary"
      }
    }
    ```

  - **场景 C（删除最后一个模型）**：

    ```json
    { "ok": true, "was_primary": true, "promoted_model": null }
    ```

    此时 `instances.ai_model_id` 会被置 0。

- **错误响应：**
  - `400 {"error": "缺少或无效的 instance_model_id 参数"}`
  - `400 {"error": "目标模型记录不存在或不属于该实例"}`

> **原子性保证：** DB 层在事务中完成"删除目标记录、提升新主模型 role、同步 `instances.ai_model_id`"三步；TAT 调用在事务外执行，失败仅记录日志，不回滚 DB。

> **TAT 行为：**
> - 删除备选：调用 `remove_model_provider.sh`，传入 `provider`、`primary`、`fallbacksb64`，从 `openclaw.json` 的 `models.providers` 移除对应 key 并同步更新 `fallbacks` 列表。
> - 删除主模型并提升新主：调用 `switch_model.sh` 传入新的 `primary` + `fallbacksb64`；同时调用 `remove_model_provider.sh` 清理被删除主模型的 provider key。
> - 删除最后一个模型：调用 `remove_model_provider.sh` 仅传 `provider`，清理 provider 配置。
> - TAT 失败不回滚 DB，仅记录日志。

---

### `GET /openclaw/instance-models`

**多模型 Fallback v2.0**：返回实例的所有模型绑定列表（按 `sort_order DESC` 排序，最新添加的在前）。

- **权限：** 登录用户（仅可查看自己的实例）
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID（URL query 或 form） |

- **响应：** `application/json`

```json
{
  "ok": true,
  "models": [
    {
      "instance_model_id": 12,
      "binding_id": "hatchery-qwen-max/qwen-max",
      "ai_model_id": 10,
      "role": "fallback",
      "provider": "通义千问",
      "model_id": "qwen-max",
      "model_name": "Qwen Max",
      "model_type": "openai-completions",
      "context_len": 128000,
      "max_tokens": 8192,
      "custom_http_headers": {
        "key": "value"
      },
      "is_custom": false
    },
    {
      "instance_model_id": 11,
      "binding_id": "hatchery-glm-4-plus/glm-4-plus",
      "ai_model_id": 8,
      "role": "primary",
      "provider": "智谱AI",
      "model_id": "glm-4-plus",
      "model_name": "GLM-4 Plus",
      "model_type": "openai-completions",
      "context_len": 128000,
      "max_tokens": 8192,
      "custom_http_headers": {
        "key": "value"
      },
      "is_custom": false
    }
  ]
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| instance_model_id | uint | `instance_models.id`，删除时传给 `del-model` 的 `instance_model_id` 参数 |
| binding_id | string | 绑定引用标识 `providerKey/modelId` |
| ai_model_id | uint | 预置模型数据库 ID，自定义模型为 `0` |
| role | string | `primary` / `fallback` |
| provider | string | 提供商展示名 |
| model_id | string | 模型 ID |
| model_name | string | 模型显示名称 |
| model_type | string | 接口类型（`openai-completions` / `anthropic-messages`） |
| context_len | int | 上下文长度 |
| max_tokens | int | Agent 单次请求模型最大输出 Token 数 |
| custom_http_headers | object | 自定义 HTTP 请求头（键值对） |
| is_custom | bool | 是否为自定义模型（`ai_model_id == 0` 时为 `true`） |

---

### `GET /openclaw/version`

查询实例的 OpenClaw 版本信息（通过 TAT 远程执行 `detect_openclaw_install.sh`）。与截图一致，用于前端"关于" / "版本信息"面板展示。

- **权限：** 登录用户（仅可查询自己的实例）
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID（URL query 或 form） |

- **响应：** `application/json`

```json
{
  "ok": true,
  "instance_id": "ins-abcd1234",
  "version": "2.16.0",
  "runtime_user": "openclaw"
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| ok | bool | 固定 `true` |
| instance_id | string | 腾讯云 CVM 实例 ID |
| version | string | 当前安装的 OpenClaw 版本号（如 `2.16.0`） |
| runtime_user | string | 运行 OpenClaw 的 OS 用户名 |

- **错误响应：**
  - `400 {"error": "缺少参数 id"}` — 未传 id 参数
  - `400 {"error": "实例不存在"}` — 实例不存在或不属于当前用户
  - `500 {"error": "TAT 执行失败: ..."}`
  - `500 {"error": "版本信息解析失败"}` — 脚本输出不是合法 JSON

> **复用脚本：** 后端调用同一个 `detect_openclaw_install.sh`（本来用于检测 openclaw 安装目录与用户），脚本同时输出 `runtime_user` 和 `openclaw_version` 两个字段。

---

### `GET /openclaw/agent-count`

查询用户实例内部的 agent 数量。对于不支持统计多个 agent 的实例类型，返回 `count=1`。

- **权限：** 登录用户（仅可查询自己的实例）
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 二选一 | 实例数据库 ID，与 instance_id 二选一，优先使用 |
| instance_id | string | 二选一 | CVM 实例 ID（如 `ins-abc123`） |

- **响应：** `application/json`

```json
{
  "ok": true,
  "count": 1
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| ok | bool | 固定 `true` |
| count | int | 实例内部 agent 数量 |

- **错误响应：**
  - `400 {"error": "缺少参数 id 或 instance_id"}` — 未传实例标识
  - `400 {"error": "实例不存在"}` — 实例不存在或不属于当前用户
  - `500 {"error": "multi-agent 查询失败", "detail": "..."}`
  - `500 {"error": "multi-agent 查询结果解析失败: ..."}`

---

### `GET/POST /openclaw/zones`

透传腾讯云 CVM DescribeZones 接口，返回当前 Region 下的可用区列表。

- **权限：** 登录用户
- **请求体（可选）：** 腾讯云 [DescribeZones](https://cloud.tencent.com/document/api/213/15707) 请求 JSON，可为空
- **响应：** 始终 JSON，直接透传腾讯云 SDK 响应

失败时返回错误信息：

- `400 {"error": "读取请求体失败: ..."}`
- `400 {"error": "请求参数格式错误: ..."}`
- `500 {"error": "创建 CVM 客户端失败: ..."}`
- `500 {"error": "查询可用区失败: ..."}`

### `POST /openclaw/terminal-url`

获取当前用户实例的 OrcaTerm 终端登录 URL。通过查询 CVM 实例信息自动判断操作系统和登录用户名，然后调用 OrcaTerm GenerateAuthLoginUrl 获取临时终端访问链接。

- **权限：** 登录用户（仅可操作自己的实例）
- **前置条件：** 管理员需开启终端功能（`TerminalEnabled = true`），否则返回 `403`
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **响应：** 始终 JSON

成功：

```json
{"login_url": "https://orcaterm.cloud.tencent.com/terminal?..."}
```

失败时返回错误信息：

- `401 {"error": "未登录"}`
- `403 {"error": "终端功能未开启，请联系管理员"}`
- `405 {"error": "Method not allowed, use POST"}`
- `400 {"error": "缺少参数 id 或 instance_id"}` / `400 {"error": "无效的 id"}`
- `404 {"error": "实例不存在"}`
- `400 {"error": "该实例无关联的 CVM"}`
- `500 {"error": "创建 CVM 客户端失败: ..."}`
- `500 {"error": "查询 CVM 实例失败: ..."}`
- `404 {"error": "CVM 实例 ins-xxx 不存在"}`
- `500 {"error": "获取云 API 凭证失败: ..."}`
- `500 {"error": "获取终端登录 URL 失败: ..."}`

### `POST /openclaw/set-gateway-ui`

为实例配置并获取 WebUI 面板地址。根据实例的 AgentType 自动选择对应的配置脚本：

- **OpenClaw**：通过 TAT 执行 `set_gateway_ui.sh`，更新 `~/.openclaw/openclaw.json` 网关配置并重启 `openclaw-gateway` 服务
- **LightclawACE**：通过 TAT 执行 `set_lightclaw_ui.sh`，重启 `lightclaw` 服务并绑定到管理后台分配的端口，从 `$HOME/lightclaw-login.txt` 读取 password
- **Hermes**：通过 TAT 执行 `set_hermes_ui.sh`，启动 `hermes dashboard --insecure` 并绑定到管理后台分配的端口，无鉴权（token 为空）
- **其他类型**：返回 400 错误，暂不支持

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |
| network_type | string | 否 | 网络类型：`"public"`（公网 IP，默认）或 `"private"`（私网 IP）。决定返回的 Gateway UI 地址使用公网还是私网 IP |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **响应：** `application/json`（始终 JSON）

成功（OpenClaw）：

```json
{
  "gatewayUI": "http://<公网IP>:<port><basePath>?token=<authToken>",
  "token": "<authToken>"
}
```

成功（LightclawACE）：

```json
{
  "gatewayUI": "http://<公网IP>:<port>",
  "token": "<password>"
}
```

成功（Hermes）：

```json
{
  "gatewayUI": "http://<公网IP>:<port>",
  "token": ""
}
```

> **API 网关域名化访问（软功能）**：当站点配置 `api_gateway_config.enable=true`、当前用户为 OneID 用户、且实例类型为 OpenClaw 时，后端会在本请求中先后调用云 API 网关 `DeleteSignOnAgentService`（清理旧路径）和 `CreateSignOnAgentService`（按当前 basePath 建新路径），随后把响应中的 `gatewayUI` 替换为 `<scheme>://<instance_id>.<base_domain><basePath>`，其中 `<scheme>` 取自 `api_gateway_config.scheme`（默认 `http`，仅支持 `http` / `https`）。任何异常都会降级为原 `ip:port` 返回，响应结构不变。此策略确保实例**重装/升级导致 basePath 变化后，下一次开启 WebUI 会自动重建网关路由，无需业务方在删除/重装流程中插入清理逻辑**。
> 若云 API 调用失败 / 超时 / 配置格式错误 / 非 OneID 用户，一律自动降级为 `http://<ip>:<port>/...` 原格式返回，**不会向用户报错**。详见 site_config 配置文档和 OpenSpec change `webui-apigateway`。

失败时返回错误信息：

- `401 {"error": "未登录"}`
- `405 {"error": "请求方法不允许"}`
- `400 {"error": "当前实例类型（...）暂不支持 WebUI"}`（不支持的实例类型）
- `400 {"error": "..."}`（实例不存在或不属于当前用户）
- `500 {"error": "创建CVM客户端失败"}`
- `404 {"error": "未找到实例"}`
- `500 {"error": "实例无可用IP"}`
- `500 {"error": "解析脚本输出失败: ..."}`
- `500 {"error": "脚本返回数据不完整"}`

### `POST /openclaw/ws-url`

获取实例的内网 WebSocket 连接信息，供 SDK 直接发起长连接。根据实例的 AgentType 自动选择对应的协议分支：

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **响应：** `application/json`（始终 JSON）

成功（OpenClaw）：

```json
{
  "url": "ws://<私网IP>:<port>/ws?token=<authToken>",
  "protocol": "websocket",
  "token": "<authToken>",
  "basePath": "/<随机化路径>"
}
```

成功（Hermes）：

```json
{
  "url": "http://<私网IP>:<port>",
  "protocol": "sse",
  "token": "<apiKey>"
}
```

失败时返回错误信息：

- `405 {"error": "请求方法不允许"}`
- `400 {"error": "无效的 JSON 格式"}`
- `400 {"error": "缺少参数: id / instance_id"}`
- `400 {"error": "instance_id 格式无效，应为 ins-xxxxxxxx"}`
- `403 {"error": "实例不存在或无权访问"}`
- `400 {"error": "实例当前状态为 XXX，仅 RUNNING 状态可获取连接地址"}`
- `400 {"error": "Gateway UI 端口未分配，请先在管理后台开启 Gateway UI"}`
- `400 {"error": "当前实例类型（...）暂不支持获取连接地址"}`（不支持的实例类型）
- `500 {"error": "查询 CVM 实例失败"}`
- `500 {"error": "获取 WS 连接信息失败: ..."}`
- `500 {"error": "解析脚本输出失败: ..."}`

### `GET /openclaw/check-gateway-access`

独立检查 WebUI 面板端口是否可访问（基于 RuleSet + ACTIVE SG 池）。用于前端在打开面板前预检端口连通性。支持 OpenClaw、LightclawACE 和 Hermes 类型，其他类型返回 400。

**新模型语义（v2 / migrate-port-open-to-ruleset 之后）：** 本接口通过实例绑定的 SG 反查所属 RuleSet，判断端口是否已在 Rules 中放通。若 `managed_sg_pool.rule_version < rule_sets.version`（规则同步中），会降级走云 API 读云端实际规则，此时响应的 `drifting` 字段为 `true`。实例不再以 `SiteConfig.SecurityGroupId` 为检查对象，老版本"当前配置的安全组 sg-xxx 未绑定在实例上"错误已不会再出现。

- **权限：** 登录用户
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 条件 | 实例的数据库 ID（Query 参数）；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（Query 参数，如 `ins-xxxxxxxx`）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **响应：** `application/json`（始终 JSON）

成功（端口可访问）：

```json
{
  "accessible": true,
  "port": 12345,
  "securityGroupIds": ["sg-xxx"],
  "drifting": false,
  "message": "面板端口可正常访问"
}
```

端口不可访问（规则未放通）：

```json
{
  "accessible": false,
  "port": 12345,
  "securityGroupIds": ["sg-xxx"],
  "drifting": false,
  "message": "面板端口 12345 尚未放通，请联系管理员在规则编辑页追加规则"
}
```

规则同步中（drift 状态；云 API 结果已考虑）：

```json
{
  "accessible": true,
  "port": 12345,
  "securityGroupIds": ["sg-xxx"],
  "drifting": true,
  "message": "面板端口可正常访问（安全组规则同步中，显示结果来自云端实际配置）"
}
```

全新租户未初始化安全组：

```json
{
  "accessible": false,
  "port": 12345,
  "securityGroupIds": ["sg-xxx"],
  "drifting": false,
  "message": "ClawPro 安全组尚未初始化，请联系管理员完成初始化后再使用本功能"
}
```

实例未绑定安全组（默认不可访问）：

```json
{
  "accessible": false,
  "port": 12345,
  "securityGroupIds": [],
  "drifting": false,
  "message": "实例未绑定任何安全组，无法检查入站规则，默认不可访问"
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| accessible | bool | 面板端口是否可访问 |
| port | int | Gateway UI 端口号 |
| securityGroupIds | []string | 实例当前绑定的 SG ID 列表（新模型下实例绑单个 SG，列表长度 ≤ 1）；未绑定时为空数组 |
| drifting | bool | `true` = 规则同步中（结果来自云 API，可能稍后改变）；`false` = 稳定态（结果来自 DB 快路径） |
| message | string | 检查结果描述 |

> **兼容性（向后兼容）：** 响应结构新增 `drifting` 字段；`securityGroupIds` 字段保留但语义调整为"实例当前绑定的 SG"（新模型下即 `instance.SecurityGroupId`），不再是"DescribeInstances 返回的云端 SG 列表"。老版本前端忽略 `drifting` 字段即可正常工作。

失败时返回错误信息：

- `401 {"error": "未登录"}`
- `405 {"error": "请求方法不允许"}`
- `403 {"error": "Gateway UI 功能未开启，请先在管理后台开启"}`
- `400 {"error": "Gateway UI 端口未分配，请先在管理后台配置"}`
- `400 {"error": "缺少参数 id"}`
- `400 {"error": "该实例无关联的 CVM"}`
- `500 {"error": "检查安全组入站规则失败: ..."}`

### `POST /openclaw/set-env`

为实例批量设置/删除环境变量（通过 TAT 远程执行 `set_env.sh`）。增量更新，只修改传入的 key，保留已有的其他 key。

- **权限：** 登录用户（仅可操作自己的实例）
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "id": 123,
  "instance_id": "ins-abc123",
  "env": {
    "OPENAI_API_KEY": "sk-xxx",
    "DEBUG": "",
    "OLD_VAR": null
  }
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 二选一 | 实例数据库 ID，与 instance_id 二选一，优先使用 |
| instance_id | string | 二选一 | CVM 实例 ID（如 `ins-abc123`） |
| env | object | 是 | 环境变量键值对，最多 50 个 |

env 字段说明：
- key 必须匹配 `^[A-Za-z_][A-Za-z0-9_]*$`（标准 bash 变量名规则）
- value 为 string：设置该环境变量（空字符串 `""` 是合法值）
- value 为 null：删除该环境变量

> **注意：** 设置环境变量后会自动重启 openclaw-gateway 服务使其生效。环境变量存储在 systemd drop-in 目录下，不受 service 文件覆盖影响。

- **响应：** `application/json`（始终 JSON）

成功：

```json
{"ok": true}
```

失败时返回错误信息：

- `400 {"error": "请求体格式错误"}`
- `400 {"error": "缺少参数 id 或 instance_id"}`
- `400 {"error": "实例不存在"}`
- `400 {"error": "env 不能为空"}`
- `400 {"error": "env 数量不能超过 50"}`
- `400 {"error": "无效的环境变量名: FOO BAR（仅允许字母、数字、下划线，且不能以数字开头）"}`
- `400 {"error": "环境变量 KEY 的值必须是字符串或 null"}`
- `500 {"error": "设置环境变量失败", "detail": "..."}`

### `GET /openclaw/env`

查看实例当前环境变量（通过 TAT 远程执行 `get_env.sh`）。

- **权限：** 登录用户（仅可操作自己的实例）
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 二选一 | 实例数据库 ID，与 instance_id 二选一，优先使用 |
| instance_id | string | 二选一 | CVM 实例 ID（如 `ins-abc123`） |

- **响应：** `application/json`（始终 JSON）

成功：

```json
{
  "ok": true,
  "env": {
    "OPENAI_API_KEY": "sk-xxx",
    "DEBUG": ""
  }
}
```

未设置过环境变量时：

```json
{"ok": true, "env": {}}
```

失败时返回错误信息：

- `400 {"error": "缺少参数 id 或 instance_id"}`
- `400 {"error": "实例不存在"}`
- `500 {"error": "查询环境变量失败", "detail": "..."}`

---

## 四、管理后台

所有管理后台接口需要 `admin` 角色。未登录返回 `401`，非管理员返回 `403 Forbidden`。

### `GET /admin/feature-allowlist/check`

查询当前登录管理员所属租户是否已开通指定功能。该接口只用于诊断。

- **权限：** 管理员 Session 或管理员用户 API Token
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 是 | 功能类型；免登录功能固定为 `passwordless-login` |

- **成功响应：**

```json
{
  "in_allowlist": true
}
```

#### 免登录功能白名单

使用免登录链接签发和消费接口前，租户必须先申请并开通 `passwordless-login` 功能白名单。

### `GET /admin/notices`

获取管控端通知栏数据，包括基础配置完成状态、腾讯云资源配额告警和产品动态。配额查询结果缓存 5 分钟，产品动态缓存 3 分钟。

- **权限：** 管理员
- **参数（Query String，均为可选）：**

| 参数 | 类型 | 说明 |
|------|------|------|
| limit | int | 产品动态返回条数，不传返回全部（最多 20 条） |
| offset | int | 产品动态偏移量，默认 0 |

- **响应：** 始终返回 JSON
- **JSON 响应（标准模式，6 步）：**

```json
{
  "config_steps": [
    { "key": "brand",          "label": "设置平台名称与品牌", "done": true  },
    { "key": "default_quota",  "label": "配置用户默认配额",   "done": true  },
    { "key": "users",          "label": "导入企业用户",       "done": false },
    { "key": "model",          "label": "配置至少一个模型",   "done": true  },
    { "key": "channel",        "label": "配置至少一个通道",   "done": true  },
    { "key": "security_group", "label": "配置安全组",         "done": false }
  ],
  "quota_alerts": [
    {
      "id": "vpc",
      "level": "critical",
      "message": "您的腾讯云账号在华南地区（广州）地域下的私有网络配额已耗尽，用户端将无法创建 Agent，请",
      "action": {
        "label": "前往腾讯云控制台提交工单",
        "href": "https://console.cloud.tencent.com/workorder/category",
        "external": true
      },
      "detail": {
        "region": "ap-guangzhou",
        "total": 200,
        "used": 200,
        "remaining": 0,
        "usage_percent": 100
      }
    },
    {
      "id": "subnet",
      "level": "critical",
      "message": "您的子网的可用 IP 已耗尽，用户端将无法创建 Agent，请",
      "action": {
        "label": "前往网络管理更换有可用 IP 的子网",
        "href": "/admin/security-group",
        "external": false
      },
      "detail": {
        "subnets": [
          { "subnet_id": "subnet-xxx", "zone": "ap-guangzhou-6", "available_ip_count": 0, "total_ip_count": 252 }
        ]
      }
    },
    {
      "id": "account_arrears",
      "level": "critical",
      "message": "您的腾讯云账号 3205597606 已欠费，将影响用户端 Agent 的正常创建，请尽快充值，",
      "action": {
        "label": "前往处理",
        "href": "https://console.cloud.tencent.com/expense/recharge",
        "external": true
      },
      "detail": {
        "uin": "3205597606",
        "is_owed": true,
        "is_over_credit": false
      }
    }
  ]
}
```

- **JSON 响应（OneID 模式，7 步）：**

OneID 模式（环境变量 `ONEID_ACCOUNT_ID` 非空时）在第 3 步之后插入"设置用户登录方式"步骤：

```json
{
  "config_steps": [
    { "key": "brand",          "label": "设置平台名称与品牌", "done": true  },
    { "key": "default_quota",  "label": "配置用户默认配额",   "done": true  },
    { "key": "users",          "label": "导入企业用户",       "done": false },
    { "key": "sso_login",      "label": "设置用户登录方式",   "done": false },
    { "key": "model",          "label": "配置至少一个模型",   "done": true  },
    { "key": "channel",        "label": "配置至少一个通道",   "done": true  },
    { "key": "security_group", "label": "配置安全组",         "done": false }
  ],
  "quota_alerts": []
}
```

#### `config_steps` 字段

根据部署模式动态返回配置步骤（标准模式 6 步，OneID 模式 7 步），每次请求实时查询数据库：

| 字段 | 类型 | 说明 |
|------|------|------|
| config_steps[].key | string | 步骤标识，固定值见下表 |
| config_steps[].label | string | 步骤中文名称 |
| config_steps[].done | bool | 是否已完成 |

| key | 判断条件 | 模式 |
|-----|---------|------|
| brand | 平台名称非空 | 两者 |
| default_quota | 默认实例配额或 Token 配额大于 0 | 两者 |
| users | 用户总数 > 1（初始管理员始终存在，有导入用户则总数超过 1） | 两者 |
| sso_login | 已配置至少一种 SSO 登录方式（`sso_im_type` 非空） | 仅 OneID |
| model | 存在已启用的 AI 模型 | 两者 |
| channel | 存在已启用的通道 | 两者 |
| security_group | 安全组 ID 已配置 | 两者 |

#### `quota_alerts` 字段

腾讯云资源配额告警，结果缓存 5 分钟：

| 字段 | 类型 | 说明 |
|------|------|------|
| quota_alerts[].id | string | 告警标识：`vpc`、`subnet`、`security_group` 或 `account_arrears` |
| quota_alerts[].level | string | 告警级别：`info`（正常）或 `critical`（已耗尽/欠费） |
| quota_alerts[].message | string | 告警文案前半段，前端拼接 `action.label` 作为可点击链接 |
| quota_alerts[].action | object | 跳转操作 |
| quota_alerts[].action.label | string | 操作链接文案（拼接在 message 之后） |
| quota_alerts[].action.href | string | 跳转链接（外部链接为完整 URL，内部链接为前端路由路径） |
| quota_alerts[].action.external | bool | `true` 用 `target="_blank"` 新窗口打开，`false` 为前端路由跳转 |
| quota_alerts[].detail | object | 配额详情，结构因 id 不同而异 |

**文案拼接规则：** `message` + `action.label`（带链接）组成完整的告警文案。例如：

> 您的腾讯云账号在华南地区（广州）地域下的云服务器 Ai2 机型购买配额已耗尽，用户端将无法创建 Agent，请[前往腾讯云配额中心提升配额](https://console.cloud.tencent.com/cvm/quota/index?...)

#### 告警级别判定

| level | 含义 | 触发条件 |
|-------|------|---------|
| info | 正常 | 资源配额未耗尽 / 账号未欠费 |
| critical | 已耗尽 | 资源配额 100% 耗尽 / 账号已欠费或超额 |

#### 告警类型一览

| id | 说明 | 数据来源 | critical 条件 | action |
|----|------|---------|--------------|--------|
| vpc | 私有网络配额 | VPC `DescribeVpcLimits` + `DescribeVpcs` | VPC 已用数 ≥ 配额上限 | 前往腾讯云控制台提交工单（外部链接） |
| subnet | 子网可用 IP | VPC `DescribeSubnets` | 当前使用的子网可用 IP 数为 0 | 前往网络管理更换子网（内部跳转 `/admin/security-group`） |
| security_group | 安全组关联实例数 | VPC `DescribeSecurityGroupLimits` + `DescribeSecurityGroupAssociationStatistics` | 关联 CVM 数 ≥ 上限（默认 2000，可通过工单提升） | 前往腾讯云控制台提交工单（外部链接） |
| account_arrears | 账号欠费/超额 | billing `CheckAccountBalance`（CommonClient，endpoint=billing.tencentcloudapi.com，Region=ap-guangzhou） | `IsOwed=true` 或 `IsOverCredit=true` | 前往腾讯云充值（外部链接） |

#### `detail` 结构（id=`vpc`）

| 字段 | 类型 | 说明 |
|------|------|------|
| region | string | 查询的地域，如 `ap-guangzhou` |
| total | uint | VPC 配额上限 |
| used | uint | 已使用的 VPC 数量 |
| remaining | uint | 剩余可用的 VPC 数量 |
| usage_percent | float | 使用率百分比 |

| 项目 | 说明 |
|------|------|
| 数据来源 | VPC `DescribeVpcLimits`（appid-max-vpcs）+ `DescribeVpcs`（TotalCount） |
| 跳转链接 | 前往腾讯云控制台提交工单（外部链接） |

VPC 告警文案：

| level | message |
|-------|---------|
| info | （无特殊文案，正常状态） |
| critical | 您的腾讯云账号在{地域名称}地域下的私有网络配额已耗尽，用户端将无法创建 Agent，请 |

#### `detail` 结构（id=`subnet`）

| 字段 | 类型 | 说明 |
|------|------|------|
| subnets | array | 可用 IP 耗尽的子网列表 |
| subnets[].subnet_id | string | 子网 ID |
| subnets[].zone | string | 子网所在可用区 |
| subnets[].available_ip_count | uint | 可用 IP 数 |
| subnets[].total_ip_count | uint | IP 总数 |

| 项目 | 说明 |
|------|------|
| 数据来源 | VPC `DescribeSubnets`（查询当前配置的子网） |
| 跳转链接 | 前往网络管理（内部路由 `/admin/security-group`） |

子网告警文案：

| level | message |
|-------|---------|
| info | （无特殊文案，正常状态） |
| critical | 您的子网的可用 IP 已耗尽，用户端将无法创建 Agent，请 |

#### `detail` 结构（id=`security_group`）

| 字段 | 类型 | 说明 |
|------|------|------|
| security_group_id | string | 安全组 ID |
| cvm_count | uint | 当前安全组关联的 CVM 实例数 |
| limit | uint | 安全组关联实例数上限（默认 2000，查询失败时回退到默认值） |

| 项目 | 说明 |
|------|------|
| 数据来源 | VPC `DescribeSecurityGroupLimits` + `DescribeSecurityGroupAssociationStatistics` |
| 跳转链接 | 前往腾讯云控制台提交工单（外部链接） |

安全组告警文案：

| level | message |
|-------|---------|
| info | （无特殊文案，正常状态） |
| critical | 您的安全组 {sg_id} 关联云服务器实例数已达上限，用户端将无法创建 Agent，请提交腾讯云工单申请提升配额， |

#### `detail` 结构（id=`account_arrears`）

| 字段 | 类型 | 说明 |
|------|------|------|
| uin | string | 腾讯云账号 UIN |
| is_owed | bool | 是否欠费 |
| is_over_credit | bool | 是否超额 |

| 项目 | 说明 |
|------|------|
| 数据来源 | billing `CheckAccountBalance`（CommonClient 接入，endpoint=`billing.tencentcloudapi.com`，Version=2018-07-09，Region 固定 `ap-guangzhou`） |
| 跳转链接 | 前往腾讯云充值（外部链接） |

账号欠费告警文案：

| level | message |
|-------|---------|
| info | （无特殊文案，正常状态） |
| critical | 您的腾讯云账号 {uin} 已欠费，将影响用户端 Agent 的正常创建，请尽快充值， |

- **降级处理：**
  - CVM 密钥未配置（`CVMSecretId` 为空）时 `quota_alerts` 返回空数组 `[]`，`config_steps` 正常返回
  - 腾讯云 API 调用失败时对应的告警项不返回，错误记录到服务端日志
- **失败响应：**
  - `401 {"error": "未登录"}`
  - `403 {"error": "需要管理员权限"}`
  - `403 {"error": "无权限访问"}`

#### `product_news` 字段

产品动态列表（已发布，按 SortOrder 降序 → PublishDate 降序排列）。后端 5 分钟内存缓存，数据来源为腾讯云 `DescribeClawProProductNews` 接口。API 调用失败时返回空数组 `[]`，前端自行 fallback 到本地硬编码数据。

| 字段 | 类型 | 说明 |
|------|------|------|
| product_news[].id | string | 产品动态 ID（腾讯云 ClawProNewsId） |
| product_news[].title | string | 标题 |
| product_news[].summary | string | 详细描述 |
| product_news[].type | string | 类型：`feature`（功能上线）/ `improvement`（体验优化）|
| product_news[].publish_date | string | 发布日期，格式 `YYYY-MM-DD` |
| product_news[].link | string | 产品动态详情链接，为空表示无外链 |
| product_news[].show_banner | bool | 是否在顶部 Banner 栏展示 |
| product_news[].banner_text | string | Banner 展示文案（`show_banner=true` 时使用） |

### `GET /admin`

重定向到 `/admin/config`。

### `GET /admin/user-limit`

获取当前用户数及用户数上限。

- **权限：** 管理员
- **JSON 响应：** `{"ok": true, "count": 42, "limit": 100}`（`limit` 为 `0` 表示不限制）

### `GET /admin/user-vpc`

获取用户自动创建的关联 VPC 信息及是否有阻断删除的资源。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 用户 ID（Query 参数） |

- **JSON 响应：**
  - 自动创建的关联 VPC 不存在（用户使用自定义 VPC、未创建过实例，或关联 VPC 已在云上销毁）：`{"vpc_id": null}`
  - 自动创建的关联 VPC 存在且无阻断资源：`{"vpc_id": "vpc-xxx", "has_resources": false}`
  - 自动创建的关联 VPC 存在且有阻断资源：`{"vpc_id": "vpc-xxx", "has_resources": true}`
  - 失败：`404 {"error": "用户不存在"}` / `500 {"error": "..."}`

### `GET /admin/users`

用户管理页面，分页显示所有用户（含已软删除的）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码（默认 1） |
| page_size | int | 否 | 每页条数（默认 20，最大 100） |
| username | string | 否 | 按用户名搜索（默认精确匹配） |
| fuzzy | string | 否 | 传 `1` 时对 `username` 启用模糊匹配（`LIKE %username%`），不传或传其他值则精确匹配 |
| department_id | string | 否 | 按部门 ID 筛选用户（兼容旧参数名 `department`） |
| role | string | 否 | 按角色筛选（`admin` 或 `user`） |
| has_personal_space | string | 否 | 传 `1` 时仅返回已绑定个人空间的用户，传 `0` 时仅返回未绑定个人空间的用户，不传则不过滤 |
| group_ids | string | 否 | 按用户组过滤，英文逗号分隔的用户组 ID，如 `1,2,3`；返回属于这些组中**任意一个**的用户（OR 语义） |
| ungrouped | string | 否 | 传 `1` 或 `true` 时只返回未加入任何用户组的用户；优先于 `group_ids` |

- **JSON 响应：** `{"users": [...], "page": 1, "page_size": 20, "total": 100, "total_pages": 5}`（密码字段已清空）

  `users` 数组中每个元素包含用户基础字段及以下部门相关字段（仅配置 OneID 时有值）：

  | 字段 | 类型 | 说明 |
  |------|------|------|
  | department | string | 主部门名称（兼容旧字段） |
  | departments | array | 完整部门列表（含层级），`omitempty` |
  | department_path | string | 主部门完整路径，如 `"公司/技术部/后端组"`，`omitempty` |
  | token_quota_rules | string | 用户 Token 配额规则 JSON 字符串。兼容旧数据：为空时会从 `token_quota_day` 转换；无限制返回 `[]` |
  | groups | array | 用户所属的用户组列表，每项包含 `id`（uint）、`name`（string）、`full_path`（string）、`source`（string）、`is_main`（bool）、`created_at`（string）、`instance_quota`（int）、`token_quota_day`（int）、`token_quota_rules`（string）、`instance_count`（int），不属于任何组时为空数组 |
  | projects | array | 用户所属项目列表，每项包含 `id`、`name`、`joined_at`；按加入项目时间升序，不属于任何项目时为 `[]` |

### `POST /admin/create`

创建新用户。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded` 或 `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |
| email | string | 否 | 邮箱（填写则通过邮件发送密码给用户） |
| role | string | 否 | 角色（`admin` 或 `user`，默认 `user`） |
| instance_quota | int | 否 | 实例配额（范围 0~999，默认取站点配置中的用户默认实例上限） |
| token_quota_day | int | 否 | 每日 Token 配额（-1=不限，默认取站点配置中的用户默认 Token 配额）。若同时传 `token_quota_rules` 则以后者为准（兼容旧客户端） |
| token_quota_rules | array | 否 | Token 配额规则数组（仅 JSON 请求体支持）。优先于 `token_quota_day`，格式见下方「Token 配额 rules 格式」 |
| group_ids | uint[] | 否 | 用户组 ID 列表，支持多组归属（覆盖式）；创建用户后将其加入这些用户组；传空数组 `[]` 时不加入任何用户组；不传或传 `null` 时同样不加入任何用户组（仅 JSON 请求体支持） |

**Token 配额 rules 格式：**

`token_quota_rules`、`default_token_quota_rules`、`global_token_quota_rules` 使用同一套 rules 数组格式。数组为空 `[]` 表示显式无限制；同一数组内不允许重复 `mode`，因此最多只能有一条 `custom` 规则。

| 写法 | 含义 |
|------|------|
| `{"mode":"day","limit":100000}` | 自然日配额，按业务时区当天 00:00 到次日 00:00 统计 |
| `{"mode":"day","limit":-1}` | 自然日窗口但不限制，用于保留规则并展示该窗口内真实用量 |
| `{"mode":"month","limit":3000000}` | 自然月配额，按业务时区当月 1 日 00:00 到下月 1 日 00:00 统计 |
| `{"mode":"year","limit":30000000}` | 自然年配额，按业务时区 1 月 1 日 00:00 到次年 1 月 1 日 00:00 统计 |
| `{"mode":"custom","limit":5000000,"start":1748736000,"refresh":"none"}` | 自定义固定窗口，从 `start` 起统计；`end` 可选，未传表示无截止 |
| `{"mode":"custom","limit":5000000,"start":1748736000,"refresh":"daily"}` | 自定义周期窗口，从 `start` 起每固定 24 小时刷新 |
| `{"mode":"custom","limit":5000000,"start":1748736000,"refresh":"monthly"}` | 自定义周期窗口，从 `start` 起每固定 31 天刷新 |
| `{"mode":"custom","limit":5000000,"start":1748736000,"refresh":"yearly"}` | 自定义周期窗口，从 `start` 起每固定 365 天刷新 |

`limit` 必须为 `-1` 或非负整数。`limit=-1` 表示该规则不限制，但 `/quota/data` 与 `/admin/usage/data` 仍会按该规则窗口返回真实用量。`custom` 的 `start` 为 Unix 秒；未传时后端会自动填当前时间。`custom` 的 `refresh` 可省略，默认 `none`。非 `custom` 模式会忽略 `start` / `end` / `refresh`。

- **JSON 响应：**
  - 成功：`{"ok": true, "id": 42}`（`id` 为新创建用户的数据库 ID）
  - 失败：`400 {"error": "用户名和密码不能为空"}` / `400 {"error": "实例配额必须为 0~999 的整数"}` / `400 {"error": "Token 配额必须为 -1 或非负整数"}` / `403 {"error": "已达到用户数上限（N）"}` / `409 {"error": "创建失败：用户名已存在"}` / `400 {"error": "用户已创建但用户组归属设置失败：存在不合法的用户组 ID"}` / `400 {"error": "用户已创建但用户组归属设置失败：目标用户组成员数量已达上限（10000 人）"}`

### `POST /admin/batch-create`

批量创建用户。仅支持 JSON（`Content-Type: application/json`）。最多支持一次创建5000个用户。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：** 用户数组

```json
[
  {"username": "u1", "password": "p1", "role": "user", "email": "", "instance_quota": 1, "token_quota_day": -1, "group_ids": [1]},
  {"username": "u2", "password": "p2"}
]
```

每个用户对象的字段：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |
| role | string | 否 | 角色（`admin` 或 `user`，默认 `user`） |
| email | string | 否 | 邮箱（填写则通过邮件发送密码给用户） |
| instance_quota | int | 否 | 实例配额（范围 0~999，默认取站点配置中的用户默认实例上限） |
| token_quota_day | int | 否 | 每日 Token 配额（-1=不限，默认取站点配置中的用户默认 Token 配额）。若同时传 `token_quota_rules` 则以后者为准 |
| token_quota_rules | array | 否 | Token 配额规则数组，优先于 `token_quota_day`；数组元素结构与创建单用户接口一致 |
| group_ids | uint[] | 否 | 用户组 ID 列表，支持多组归属（覆盖式）；创建用户后将其加入这些用户组；传空数组 `[]` 或不传时不加入任何用户组 |

- **响应：** 始终 JSON，HTTP 状态码始终 200（具体错误在 results 中），部分失败不影响其他用户创建

```json
{
  "results": [
    {"username": "u1", "ok": true, "id": 42},
    {"username": "u2", "ok": false, "error": "用户名已存在"}
  ]
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| username | string | 用户名 |
| ok | bool | 是否创建成功 |
| id | uint | 新创建用户的数据库 ID（仅 `ok=true` 时返回） |
| error | string | 错误信息（仅 `ok=false` 时返回） |

- **整体失败：** `400 {"error": "请求体格式错误：..."}` / `400 {"error": "用户列表不能为空，不能超过5000"}` / `403 {"error": "导入后将超过用户数上限（N），当前已有 M 个用户"}`
- **单条用户组失败（`ok=false`）：** `"每个用户最多只能加入一个用户组"` / `"用户已创建但用户组归属设置失败：存在不合法的用户组 ID"` / `"用户已创建但用户组归属设置失败：目标用户组成员数量已达上限（10000 人）"`

### `POST /admin/export-tokens`

批量为所有用户生成并导出 API Token。已有 Token 的用户保留不变，没有 Token 的用户自动生成。仅支持 JSON。

- **权限：** 管理员
- **参数：** 无
- **响应：** 始终 JSON

```json
[
  {"id": 1, "username": "admin", "token": "hk-abcdef..."},
  {"id": 2, "username": "user1", "token": "hk-123456..."}
]
```

- **失败：** `500 {"error": "生成 Token 失败：..."}`

### `GET /admin/user-token`

管理员根据用户 ID 查询指定用户的 API Token。仅支持 JSON。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 用户 ID（Query 参数） |

- **响应：** 始终 JSON

**用户未创建 Token：**

```json
{
  "exists": false
}
```

**用户已创建 Token：**

```json
{
  "exists": true,
  "token": "hk-abcdef1234567890abcdef1234567890abcdef1234567890",
  "disabled": false,
  "created_at": "2026-03-28T10:00:00+08:00"
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| exists | bool | 是否已创建 Token |
| token | string | API Token 明文（仅 `exists=true` 时返回） |
| disabled | bool | 是否被管理员禁用（仅 `exists=true` 时返回） |
| created_at | string \| null | Token 创建/重置时间（ISO 8601）（仅 `exists=true` 时返回） |

- **失败：** `404 {"error": "用户不存在"}`

### `POST /admin/token/disable`

管理员禁用指定用户的 API Token。禁用后该用户的 API Token 将无法用于认证。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 用户 ID（Query 参数） |

- **响应：** 始终 JSON
  - 成功：`{"ok": true}`
  - 失败：`404 {"error": "用户不存在"}` / `400 {"error": "该用户没有 API Token"}` / `400 {"error": "该用户 Token 已处于禁用状态"}`

### `POST /admin/token/enable`

管理员启用指定用户的 API Token（解除禁用）。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 用户 ID（Query 参数） |

- **响应：** 始终 JSON
  - 成功：`{"ok": true}`
  - 失败：`404 {"error": "用户不存在"}` / `400 {"error": "该用户没有 API Token"}` / `400 {"error": "该用户 Token 未被禁用"}`

### `POST /admin/delete`

软删除用户。同时关闭该用户名下所有 CVM 实例。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 用户 ID（Query 参数） |

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`404 {"error": "用户不存在"}` / `403 {"error": "不能删除初始管理员"}` / `500 {"error": "关机失败：..."}`

### `POST /admin/hard-delete`

永久删除用户。删除后不可恢复，要求该用户名下没有实例。若用户有自动创建的关联 VPC 且无阻断资源，将自动删除该关联 VPC。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 用户 ID（Query 参数） |

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`404 {"error": "用户不存在"}` / `403 {"error": "不能删除初始管理员"}` / `409 {"error": "该用户还有实例存在，请先删除其实例"}` / `409 {"error": "用户自动创建的关联 VPC（vpc-xxx）下仍有资源占用，请先清理后再删除用户"}` / `500 {"error": "删除 VPC 失败：..."}`

### `POST /admin/restore`

恢复已软删除的用户。同时启动该用户名下所有 CVM 实例。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 用户 ID（Query 参数） |

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`404 {"error": "用户不存在"}` / `403 {"error": "不能操作初始管理员"}`

### `POST /admin/reset-password`

重置用户密码。支持两种定位方式：按用户 ID 或按 `init_user=true` 自动定位初始管理员。初始管理员（角色为 admin、ID 最小的用户）只能通过 admin-token 认证重置密码，其他用户无额外限制。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件必填 | 用户 ID（Query 参数）；与 `init_user` 二选一 |
| init_user | string | 条件必填 | 传 `true` 时自动定位初始管理员（角色为 admin 且 ID 最小），无需传 `id`；与 `id` 二选一 |
| password | string | 是 | 新密码（Form 参数） |
| email | string | 否 | 邮箱（Form 参数，填写则通过邮件发送新密码给用户） |

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`400 {"error": "密码不能为空"}` / `404 {"error": "用户不存在"}` / `404 {"error": "初始管理员不存在"}` / `403 {"error": "初始管理员密码只能通过 admin-token 重置"}`

### `POST /admin/update-user`

更新用户属性（用户名和密码除外）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 用户 ID（Query 参数） |
| email | string | 否 | 邮箱 |
| role | string | 否 | 角色（`admin` 或 `user`） |
| instance_quota | int | 否 | 实例配额（-1=无限制，0~999，-1 表示不限制实例数量，0 表示无配额） |
| token_quota_day | int | 否 | 每日 Token 配额（-1=不限）。若同时传 `token_quota_rules` 则以后者为准 |
| token_quota_rules | array | 否 | Token 配额规则数组（仅 JSON 请求体支持），优先于 `token_quota_day`；数组元素结构与创建用户接口一致 |
| group_ids | uint[] | 否 | 用户组 ID 列表，支持多组归属（覆盖式）；覆盖更新该用户的用户组归属；传空数组 `[]` 时清空所有用户组；不传或传 `null` 时不修改用户组归属（仅 JSON 请求体支持） |

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`404 {"error": "用户不存在"}` / `403 {"error": "不能修改初始管理员的角色"}` / `400 {"error": "配额必须为 0~999 的整数"}` / `400 {"error": "Token 配额必须为 -1 或非负整数"}` / `400 {"error": "没有可更新的字段"}` / `400 {"error": "存在不合法的用户组 ID"}` / `400 {"error": "目标用户组成员数量已达上限（10000 人）"}`

> **Token 配额兼容规则：**
>
> - `token_quota_rules` 是新的配额字段，`token_quota_day` 是旧字段。两者共存以保证向后兼容。
> - **写入优先级**：传 `token_quota_rules` 时忽略 `token_quota_day`；仅传 `token_quota_day` 时自动转换为等价的 rules（`[{"mode":"day","limit":N}]`）。
> - **响应中**：`token_quota_rules` 返回兼容后的有效 rules JSON 字符串。未迁移的旧用户会从 `token_quota_day` 转换；无规则且无限制时返回 `[]`。
> - **配额执行**：
>   - Agent 有分组时：运行时从组策略解析 `token_quota_rules`（优先），fallback 到组的 `token_quota_day` 策略，再 fallback 到用户字段。管理员修改组策略后即时生效，无需重新烙印。
>   - Agent 无分组时：读取用户字段 `token_quota_rules`（优先），fallback 到 `token_quota_day`。
> - **自然迁移**：用户下次被显式修改配额（通过 API）时，配额自动写入 `token_quota_rules`，`token_quota_day` 置为 -1。无需手动迁移。
> - **custom 模式 start 时间**：
>   - 设置组策略/站点默认值时：未指定 start 则自动填充为配置时间（组内所有用户共享同一计时起点）。
>   - 创建用户烙印时：start 强制覆盖为用户创建时间（每个用户独立计时）。

### `GET /admin/departments`

获取部门列表（仅限配置了 OneID 的环境）。始终返回 JSON，不受 Accept Header 影响。

- **权限：** 管理员
- **参数：** 无
- **响应：** 始终 JSON

```json
{
  "departments": ["技术部", "产品部", "运营部"],
  "department_tree": [
    {
      "id": "dept_001",
      "name": "后端组",
      "path": "公司/技术部/后端组",
      "parent_id": "dept_tech",
      "has_child": false
    }
  ]
}
```

  | 字段 | 类型 | 说明 |
  |------|------|------|
  | departments | string[] | 主部门名称去重列表（兼容旧格式） |
  | department_tree | array | 新格式部门树，含 id/name/path/parent_id/has_child |

  `department_tree` 每项字段说明：

  | 字段 | 类型 | 说明 |
  |------|------|------|
  | id | string | 部门 ID |
  | name | string | 部门名称 |
  | path | string | 完整层级路径，如 `"公司/技术部/后端组"` |
  | parent_id | string | 父部门 ID |
  | has_child | bool | 是否有子部门 |

### `GET /admin/config`

站点配置页面。

- **权限：** 管理员
- **参数（Query）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| template_path | string | 否 | 指定要返回的资源配置子模块（snake_case），可多次传递。返回默认 ResourcePolicy 覆盖 `cvm_template` 后的实际生效值；未传时默认返回 `internet_accessible` |

  示例：`?template_path=internet_accessible&template_path=system_disk`

- **JSON 响应：** `{"config": {"name": "...", "has_logo": true, "cvm_region": "...", "cvm_region_id": "...", "available_zones": ["ap-beijing-6", ...], "cvm_secret_id": "...", "cvm_template": "...", "instance_charge_type": "PREPAID", "security_group_id": "...", "skillhub": "...", "cvm_uin": "...", "domain": "...", "global_token_quota_day": 0, "global_token_quota_period": "day", "public_image_id": "", "vpc_id": "", "subnet_ids": "", "terminal_enabled": false, "chat_view_enabled": true, "gateway_ui_enable": false, "gateway_ui_port": 0, "gateway_ui_addr_type": "private", "browser_vnc_enable": false, "doctor_enabled": false, "agent_cam_role_secret_id": "", "has_oneid": false, "sso_im_types": [], "sso_im_type_options": [...], "internet_accessible": {"public_ip_assigned": true, ...}}}`
  - `cvm_region` 当前地域的中文显示名称（如 `华北地区（北京）`、`华东地区（上海）`、`华南地区（广州）`、`华东地区（南京）`），未配置 region 时为空字符串
  - `cvm_region_id` 当前地域的实际代号（如 `ap-guangzhou`），未配置 region 时为空字符串
  - `cvm_uin` 来自启动参数 `-uin`，未配置时为空字符串
  - `domain` 来自启动参数 `-domain`，未配置时为空字符串
  - `available_zones` 当前 region 下的可用区列表，未配置 region 时为 null
  - `instance_charge_type` 默认 ResourcePolicy 中实际生效的新建 Agent CVM 计费类型：`PREPAID` 包年包月、`POSTPAID_BY_HOUR` 按量计费；仅影响后续新建实例，不修改已有实例
  - `global_token_quota_day` 全局 Token 配额值，-1 表示不限制。字段名历史保留 `day`，实际按日或按月由 `global_token_quota_period` 决定；配置了 `global_token_quota_rules` 时会从兼容的 day/month 规则反推
  - `global_token_quota_period` 全局 Token 配额兼容周期：`day` 每日本地自然日，`month` 每月本地自然月。配置了 `global_token_quota_rules` 且只有 day/month 兼容规则时会从规则反推；仅有 year/custom 规则时旧字段无法表达，返回保存的 day/month 值
  - `global_token_quota_rules` 站点全局 Token 配额规则数组，数组元素结构与用户创建接口的 `token_quota_rules` 一致。为 null 时 fallback 到 `global_token_quota_day` + `global_token_quota_period`
  - `default_instance_quota` 用户默认实例上限，系统初始值 `3`
  - `default_token_quota_day` 用户默认每日 Token 配额，-1 表示不限制，系统初始值 `500000`。若设置了 `default_token_quota_rules` 则以后者为准
  - `default_token_quota_rules` 用户默认 Token 配额规则数组（JSON），数组元素结构与用户创建接口的 `token_quota_rules` 一致。为 null 时 fallback 到 `default_token_quota_day`
  - `public_image_id` 公共镜像 ID，未配置时为空字符串
  - `vpc_id` 全局 VPC ID，未配置时为空字符串
  - `subnet_ids` 全局子网配置 JSON 字符串，格式为 `zone → []subnetId`（每个可用区可配置多个子网），未配置时为空字符串。示例：`{"ap-guangzhou-6":["subnet-a","subnet-b"]}`。读取时兼容旧单值格式 `{"ap-guangzhou-6":"subnet-a"}`，保存时自动升级为数组格式
  - `terminal_enabled` 是否开启终端查看功能，默认 `false`
  - `chat_view_enabled` 是否允许前端加载对话界面，默认 `true`
  - `gateway_ui_enable` 是否开启 Gateway UI 面板功能，默认 `false`
  - `gateway_ui_port` Gateway UI 面板分配的端口，默认 `0`（未分配）
  - `gateway_ui_addr_type` Gateway UI 地址类型（`private`/`public`），默认 `private`
  - `browser_vnc_enable` 是否开启云端浏览器（VNC）功能，默认 `false`
  - `doctor_enabled` 是否允许用户使用龙虾医生，默认 `false`。作为分组策略 `lobster_doctor` 的全局兜底值
  - `local_agent_enabled` 是否允许本租户接入本地 Agent（reporter 第 ② 层守卫；SiteConfig 全局预设），默认 `false`。下期叠加分组策略 `local_agent`，作为该策略的全局兜底值
  - `user_data_enabled` 是否允许用户在创建实例时提交 UserData，默认 `false`。开启后用户可通过 `POST /openclaw/create` 的 `user_data` 参数传入自定义脚本
  - `user_config_model_enabled` 是否允许用户查看与配置模型，默认 `true`
  - `user_config_channel_enabled` 是否允许用户查看与配置通道，默认 `true`
  - `model_quota_enabled` 是否允许用户查看模型额度，默认 `true`
  - `has_oneid` 是否已配置 OneID（当 TenantID 非空时为 `true`）
  - `sso_im_types` 已配置的 IM 类型数组，如 `["wecom","feishu"]`，未配置时为 `[]`
  - `sso_im_type_options` 全量 IM 类型枚举，每项含 `value`、`label`、`logo` 三个字段，固定值
  - `internet_accessible`、`system_disk` 等：根据 `template_path` 参数动态返回默认 ResourcePolicy 覆盖 `cvm_template` 后的资源子模块，字段名为 snake_case 格式。支持的 `template_path` 值：`internet_accessible`、`system_disk`、`instance_type`、`instance_charge_type`、`instance_charge_prepaid`
  - `default_tags` 全局默认标签 JSON 字符串，格式 `[{"Key":"env","Value":"prod"}]`

### `POST /admin/config`

更新站点配置（站点名称、Logo、全局配额、默认用户配额、终端开关、Gateway UI 开关）。

- **权限：** 管理员
- **Content-Type：** `multipart/form-data`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 站点名称 |
| logo | file | 否 | Logo 图片（PNG/JPEG/SVG，≤512KB） |
| global_token_quota_day | int | 否 | 全局 Token 配额值（-1=不限）。字段名历史保留 `day`，实际周期由 `global_token_quota_period` 决定 |
| global_token_quota_period | string | 否 | 全局 Token 配额统计周期：`day` 每日本地自然日，`month` 每月本地自然月；默认 `day` |
| global_token_quota_rules | string | 否 | 站点全局 Token 配额规则（JSON 数组字符串）；数组元素结构与用户创建接口的 `token_quota_rules` 一致 |
| default_instance_quota | int | 否 | 用户默认实例上限（0~999，系统初始值 `3`） |
| default_token_quota_day | int | 否 | 用户默认每日 Token 配额（-1=不限，系统初始值 `500000`）。若同时传 `default_token_quota_rules` 则以后者为准 |
| default_token_quota_rules | string | 否 | 用户默认 Token 配额规则（JSON 数组字符串）；数组元素结构与用户创建接口的 `token_quota_rules` 一致 |
| terminal_enabled | string | 否 | `"true"` 开启 / `"false"` 关闭终端查看功能 |
| chat_view_enabled | string | 否 | `"true"` 开启 / `"false"` 关闭前端对话界面加载开关 |
| gateway_ui_enable | string | 否 | `"true"` 开启 / `"false"` 关闭 Gateway UI 面板功能。设为 `"true"` 时，若当前 `gateway_ui_port` 为 0，将自动随机分配一个端口（范围 10000~40000）；设为 `"false"` 时，已分配的端口保持不变 |
| gateway_ui_addr_type | string | 否 | Gateway UI 地址类型：`"private"` 或 `"public"`，决定用户侧 `POST /openclaw/set-gateway-ui` 默认使用私网还是公网 IP |
| browser_vnc_enable | string | 否 | `"true"` 开启 / `"false"` 关闭云端浏览器（VNC）功能。开启时会自动放通安全组 6080 端口 |
| doctor_enabled | string | 否 | `"true"` 开启 / `"false"` 关闭龙虾医生功能（全局兜底开关，分组策略可覆盖） |
| local_agent_enabled | string | 否 | `"true"` 开启 / `"false"` 关闭本租户本地 Agent 接入。reporter 三接口双层守卫之二，与 `feature_allowlist` 取 AND；详见 `POST /local-agent/report` |
| user_data_enabled | string | 否 | `"true"` 允许 / `"false"` 禁止用户在创建实例时提交 UserData。开启后用户可通过 `POST /openclaw/create` 传入 `user_data` 参数 |
| user_config_model_enabled | string | 否 | `"true"` 允许 / `"false"` 禁止用户查看与配置模型，默认 `"true"` |
| user_config_channel_enabled | string | 否 | `"true"` 允许 / `"false"` 禁止用户查看与配置通道，默认 `"true"` |
| model_quota_enabled | string | 否 | `"true"` 允许 / `"false"` 禁止用户查看模型额度，默认 `"true"` |
| instance_charge_type | string | 否 | 新建 Agent 的 CVM 计费类型：`PREPAID`=包年包月，`POSTPAID_BY_HOUR`=按量计费。同步修改 `cvm_template` 与默认 ResourcePolicy，仅影响后续新建实例 |
| default_tags | string | 否 | 全局默认标签 JSON 字符串，格式 `[{"Key":"env","Value":"prod"}]`。提交后替换所有全局默认标签；传空字符串或 `[]` 可清除全局默认标签 |
| sso_im_types | string | 否 | IM 类型数组 JSON 字符串，如 `["wecom","feishu"]`，传 `[]` 清空；可选值：`wecom` / `feishu` / `dingtalk` / `aad` / `saml` / `ad` / `wework_private` / `oidc` / `jwt` / `openldap` / `cas` / `oauth2` |
| api_gateway_config | string | 否 | WebUI 接入云 API 网关的 JSON 配置字符串，格式 `{"enable":true,"gateway_instance_id":"ins-xxx","base_domain":"xxx.com","scheme":"http"}`。默认 `{}`（即禁用）。启用时（`enable=true`）`gateway_instance_id` 和 `base_domain` 必填；`scheme` 可选，只接受 `http` / `https`，未传/非法值时回落到 `http`。这是软功能：云 API 调用异常/超时/格式错误一律降级为原 `ip:port` 返回，不影响用户正常开 WebUI |

- **JSON 响应：**
  - 成功：`{"ok": true}`（当 `gateway_ui_enable` 为 `true` 时额外返回 `"gateway_ui_port": <port>`）
  - 成功但有警告：`{"ok": true, "warning": "云端浏览器功能已开启，但安全组端口放通失败: ..."}`（当 `browser_vnc_enable` 开启但安全组 6080 端口放通失败时）
  - 失败：`400 {"error": "仅支持 PNG、JPEG、SVG 格式的图片"}`、`400 {"error": "当前 ClawPro 未配置安全组，请先在安全组管理中创建或绑定安全组后再开启 Gateway UI"}`、`400 {"error": "gateway_ui_addr_type 仅支持 \"private\" 或 \"public\""}` 等

> `global_token_quota_rules` / `default_token_quota_rules` 的参数值是 rules 数组的 JSON 字符串，rules 支持的写法见 `POST /admin/create` 的「Token 配额 rules 格式」。
>
> `global_token_quota_day` / `default_token_quota_day` 为兼容旧客户端保留的写入字段。仅传旧字段时，后端会写入对应的规则数组；同时传旧字段和 rules 时，以 rules 为准。

### `POST /admin/config/cvm`

更新 CVM 云服务器配置。仅覆盖请求中包含的参数，未传递的参数保持不变。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cvm_secret_id | string | 否 | 腾讯云 SecretId，传空字符串可清除 |
| cvm_secret_key | string | 否 | 腾讯云 SecretKey，传空字符串不会覆盖（防误清） |
| cvm_template | string | 否 | CVM 创建模板（JSON），传空字符串可清除。提交时以其中五类资源字段完整替换默认 ResourcePolicy，其余 CVM 字段仍仅保存在模板中 |
| skillhub | string | 否 | SkillHub 地址，传空字符串可清除 |
| public_image_id | string | 否 | 公共镜像 ID，传空字符串可清除 |
| vpc_id | string | 否 | 全局 VPC ID，传空字符串可清除。配置后创建实例使用全局 VPC |
| subnet_ids | string | 否 | 全局子网配置 JSON，格式为 `{"<zone>": ["<subnet-id>", ...]}`（每可用区多子网）。创建实例时按 `AvailableIpAddressCount` 加权随机挑选，跳过已满子网。为兼容旧数据，读取时同时接受旧单值格式 `{"<zone>": "<subnet-id>"}`。vpc_id 未配置时，该字段被忽略 |
| agent_cam_role_secret_id | string | 否 | Agent CAM 角色 SecretId，传空字符串可清除 |
| agent_cam_role_secret_key | string | 否 | Agent CAM 角色 SecretKey，传空字符串不会覆盖（防误清） |

- **JSON 响应：** 成功 `{"ok": true}`

### `POST /admin/config/template`

通用 CVM 模板分模块修改接口。以 Patch 方式合并到 `cvm_template`，并将请求中出现的资源模块同步到默认 ResourcePolicy；未传字段保持不变。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| internet_accessible | object | 否 | 公网访问配置（子字段：`public_ip_assigned`、`internet_charge_type`、`internet_max_bandwidth_out`） |
| system_disk | object | 否 | 系统盘配置（子字段：`disk_type`、`disk_size`），详见下方校验规则 |
| instance_type | string | 否 | 实例机型，如 `"Ai2.MEDIUM4"`，详见下方校验规则 |
| instance_charge_type | string | 否 | 计费模式，如 `"POSTPAID_BY_HOUR"` |
| instance_charge_prepaid | object | 否 | 预付费配置（子字段：`period`、`renew_flag`） |

  **合并规则：**
  - 传值：替换对应字段
  - 传 `null`：删除对应字段
  - 未传：保持不变

  请求示例：

```json
{
  "internet_accessible": {
    "public_ip_assigned": true,
    "internet_charge_type": "TRAFFIC_POSTPAID_BY_HOUR",
    "internet_max_bandwidth_out": 10
  }
}
```

- **字段名转换：** 后端自动将请求中的 snake_case 字段名转换为 CVM 模板内部的大驼峰（PascalCase）格式存储。例如 `public_ip_assigned` → `PublicIpAssigned`，`internet_accessible` → `InternetAccessible`。

- **分模块校验：**

  1. **internet_accessible**：包含时会进行公网配置业务规则校验（带宽范围、计费模式合法性等）。

  2. **instance_type**（白名单校验）：
     - 允许的机型：`Ai2.MEDIUM2`、`Ai2.MEDIUM4`、`Ai2.LARGE8`
     - 传入不支持的机型将返回错误，并提示当前可选机型列表

  3. **system_disk**：
     - **disk_type**：白名单校验，允许的类型：`CLOUD_SSD`（SSD云硬盘）、`CLOUD_PREMIUM`（高性能云硬盘）、`CLOUD_BSSD`（通用型SSD云硬盘）、`CLOUD_HSSD`（增强型SSD云硬盘）
     - **disk_size**：必须为整数（不接受小数），且不能小于 50GB

- **JSON 响应：**
  - 成功：`{"ok": true, "cvm_template": "{...}", "message": "模板配置保存成功", "internet_accessible": {...}}`
    - `cvm_template`：更新后的完整 CVM 模板 JSON 字符串
    - `internet_accessible`：当修改了公网配置时，返回 Normalize 后的公网配置
  - 失败（通用）：
    - `400 {"error": "请求体不能为空"}`
    - `400 {"error": "不允许修改的字段: xxx"}`
    - `500 {"error": "当前 CVM 模板格式错误: ..."}`
  - 失败（instance_type 校验）：
    - `400 {"error": "实例规格 S5.MEDIUM2 不可用，当前可选机型: Ai2.MEDIUM4, Ai2.LARGE8"}`
    - `400 {"error": "实例规格 Ai2.MEDIUM4 不可用，当前 Region（ap-guangzhou）下白名单机型均不可用"}`（极端情况：所有白名单机型均不可用）
  - 失败（system_disk 校验）：
    - `400 {"error": "不支持的系统盘类型: LOCAL_SSD，允许的类型: CLOUD_SSD, CLOUD_PREMIUM, CLOUD_BSSD, CLOUD_HSSD"}`
    - `400 {"error": "SystemDisk.DiskSize 必须为整数，当前值: 50.5"}`
    - `400 {"error": "系统盘大小不能小于 50GB，当前值: 30GB"}`
  - 失败（internet_accessible 校验）：
    - `400 {"error": "InternetAccessible 校验失败信息"}`

### `GET /admin/resource-policies`

分页返回当前租户的独立资源策略。首次访问时会并发安全地懒创建企业默认策略；默认策略始终置顶。默认策略的持久化名称保持稳定，但所有展示读取按请求语言返回“企业默认资源策略”或 `Enterprise Default Resource Policy`；普通策略名称始终按管理员输入原样返回，不参与翻译。

- **权限：** 管理员
- **Query：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1，必须大于 0 |
| page_size | int | 否 | 每页数量，默认 10，范围 1～100 |

- **JSON 响应：**

```json
{
  "items": [
    {
      "id": 12,
      "name": "研发资源策略",
      "is_default": false,
      "resource_config": {
        "instance_charge_type": "POSTPAID_BY_HOUR",
        "instance_type": "Ai2.MEDIUM4",
        "system_disk": {"disk_type": "CLOUD_SSD", "disk_size": 100}
      },
      "groups": [
        {"id": 3, "name": "研发组", "full_path": "总部/研发组"}
      ],
      "created_at": "2026-07-21T10:00:00+08:00",
      "updated_at": "2026-07-21T10:00:00+08:00"
    }
  ],
  "total": 2,
  "page": 1,
  "page_size": 10
}
```

`groups` 只包含直接应用该策略的分组，不展开继承后代。应用范围通过 `GroupConfigBinding(config_type=resource_policy, config_key=<policy_id>)` 索引查询；策略配置只存于 `resource_policies.config_json`。

### `POST /admin/resource-policies/create`

原子创建普通资源策略及其应用分组。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **审计 action：** `resource_policy_create`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 租户内唯一，trim 后 1～128 字符；不能使用固定保留名“企业默认资源策略”或其他已存在名称 |
| resource_config | object | 是 | ResourceConfig；未知字段忽略，已知字段规范化并校验 |
| group_ids | uint[] | 是 | 非空；分组必须存在，且不能已被其他资源策略直接绑定 |

成功返回 `{"id":12}`。名称冲突或分组已被占用返回 409；任一校验或写入失败时策略及全部绑定一起回滚。

### `POST /admin/resource-policies/update`

原子更新策略配置和应用范围。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **审计 action：** `resource_policy_update`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 资源策略 ID |
| name | string | 否 | 普通策略必填且不能改为固定保留名“企业默认资源策略”；默认策略可省略，或原样回传当前请求语言下读取到的默认展示名称；其他值返回 409 |
| resource_config | object | 是 | 规范化并校验后的 ResourceConfig |
| group_ids | uint[] | 否 | 普通策略必须非空；默认策略必须省略或为空，传非空数组返回 409 |

普通策略请求字段与创建接口相同，并额外要求 `id`。企业默认资源策略只要求 `id` 和 `resource_config`：名称固定且没有应用分组；同一请求语言下原样回传读取到的本地化默认名称视为未改名，其他名称或非空 `group_ids` 返回 409，不会覆盖此前已保存的配置。

成功返回 `{"ok":true}`；策略不存在返回 404，名称冲突、分组占用或默认策略保护冲突返回 409。

### `POST /admin/resource-policies/delete`

删除普通策略并清理其全部直接分组绑定。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **审计 action：** `resource_policy_delete`
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 要删除的普通资源策略 ID |

- **请求：** `{"id":12}`

成功返回 `{"ok":true}`。企业默认资源策略不可删除，尝试删除返回 409。

### ResourceConfig schema

| 字段 | 类型 | 说明 |
|------|------|------|
| instance_charge_type | string | `PREPAID` 或 `POSTPAID_BY_HOUR`；保存前 trim 并转大写 |
| instance_charge_prepaid | object | 当计费类型为 `PREPAID` 时必填；`period` 必须大于 0，可带 `renew_flag` |
| instance_type | string | 固定 allowlist：`Ai2.MEDIUM2`、`Ai2.MEDIUM4`、`Ai2.LARGE8` |
| system_disk | object | `disk_type` 为允许的云盘类型；`disk_size` 不应用旧 CVMTemplate 的 50GB 下限，实际容量范围按系统盘选项接口返回值 |
| internet_accessible | object | 可包含 `public_ip_assigned`、`internet_charge_type`、`internet_max_bandwidth_out` |

策略可以只覆盖部分字段；未提供字段继续使用 CVMTemplate。未知字段不会进入规范化后的持久化 JSON。

默认策略首次物化时从当前租户 `CVMTemplate` 提取资源字段；后续访问不会覆盖管理员已经编辑的默认策略。不在启动时扫描全部租户。

### `GET /admin/resource-policies/options/instance-types`

查询目标可用区中处于 `SELL` 状态且属于固定 Ai2 allowlist 的实例规格。

- **权限：** 管理员
- **参数（Query）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| zone | string | 是 | 腾讯云可用区，例如 `ap-guangzhou-6` |
| instance_charge_type | string | 否 | `PREPAID` 或 `POSTPAID_BY_HOUR`；省略时不添加计费类型过滤 |
| refresh | string | 否 | 传 `1` 绕过已有缓存并刷新；其他值按普通缓存请求处理 |

- **JSON 响应：**

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 成功时为 `true` |
| source | string | 本次调用腾讯云时为 `tencent_cloud`；缓存命中或 inflight waiter 复用结果时为 `cache` |
| refreshed_at | string | 缓存生成时间，UTC RFC3339 |
| instance_types | array | 可用机型数组 |

`instance_types[]`：

| 字段 | 类型 | 说明 |
|------|------|------|
| instance_type | string | allowlist 中的 SELL 机型 |
| cpu | int | vCPU 数量 |
| memory | int | 内存大小，单位由腾讯云接口定义 |
| unit_price | number | 腾讯云返回价格时出现；无价格时省略 |

```json
{
  "ok": true,
  "source": "tencent_cloud",
  "refreshed_at": "2026-07-17T06:00:00Z",
  "instance_types": [
    {
      "instance_type": "Ai2.MEDIUM4",
      "cpu": 2,
      "memory": 4
    }
  ]
}
```

### `GET /admin/resource-policies/options/system-disks`

先通过 CVM 查询目标机型的实例族、CPU 和内存，再通过 CBS `DescribeDiskConfigQuota` 查询可作为系统盘的磁盘类型和容量范围。

- **权限：** 管理员
- **参数（Query）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| zone | string | 是 | 腾讯云可用区，例如 `ap-guangzhou-6` |
| instance_charge_type | string | 否 | `PREPAID` 或 `POSTPAID_BY_HOUR`；省略时按后付费磁盘查询 |
| instance_type | string | 是 | 必须是固定 allowlist 中的机型 |
| refresh | string | 否 | 传 `1` 绕过已有缓存并刷新；其他值按普通缓存请求处理 |

- **JSON 响应：**

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 成功时为 `true` |
| source | string | 本次调用腾讯云时为 `tencent_cloud`；缓存命中或 inflight waiter 复用结果时为 `cache` |
| refreshed_at | string | 缓存生成时间，UTC RFC3339 |
| system_disk_options | array | 可用系统盘数组 |

`system_disk_options[]`：

| 字段 | 类型 | 说明 |
|------|------|------|
| disk_type | string | 后端磁盘 allowlist 中的可用系统盘类型 |
| min_disk_size | int | 最小容量 GB；直接使用 CBS `DescribeDiskConfigQuota` 返回值 |
| max_disk_size | int | 最大容量 GB；腾讯云未返回时可能为 0 |
| step_size | int | 容量步长 GB；腾讯云未返回时可能为 0 |

仅保留 `available=true`、用途为 `SYSTEM_DISK`、类型在 allowlist 中且容量区间有效的结果。`max_disk_size > 0` 且小于最终 `min_disk_size` 的结果会被丢弃。

```json
{
  "ok": true,
  "source": "cache",
  "refreshed_at": "2026-07-17T06:00:00Z",
  "system_disk_options": [
    {
      "disk_type": "CLOUD_BSSD",
      "min_disk_size": 50,
      "max_disk_size": 1000,
      "step_size": 10
    }
  ]
}
```

两个动态 options 接口共享以下缓存契约：

- TTL 为 5 分钟；key 包含租户 identifier、region、endpoint 类型及 zone/charge type/instance type 等 scope，不跨租户复用云账号结果。
- `refresh=1` 绕过已有 cache；同 key 并发 miss/refresh 仍只有一个 winner 调用云 API，waiter 复用 winner 写入的缓存。
- 云 API 失败不会写入坏缓存；waiter 无可用结果时返回受控 500。
- 缺少必填参数或枚举/allowlist 非法返回 400；CVM/CBS client 或 SDK 查询失败返回 500。

### `GET /admin/config/security-group`

查询当前配置的安全组详情。用保存的安全组 ID 调用腾讯云 VPC [DescribeSecurityGroups](https://cloud.tencent.com/document/product/215/15808) 接口，响应直接透传。

- **权限：** 管理员
- **响应：** 始终 JSON，透传腾讯云 `DescribeSecurityGroups` 响应。
  - 未配置安全组时：`404 {"error": "未配置安全组"}`
  - 查询失败：`500 {"error": "..."}`

### `POST /admin/config/security-group`

创建新安全组并自动绑定到站点配置。请求体为 JSON，支持腾讯云 VPC [CreateSecurityGroup](https://cloud.tencent.com/document/product/215/15806) 标准参数，并扩展了 `quick_rules` 字段用于快速添加常用规则。创建成功后自动将所有实例迁移到新安全组（异步执行）。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| GroupName | string | 是 | 安全组名称 |
| GroupDescription | string | 否 | 安全组描述 |
| quick_rules | string[] | 否 | 快速规则组标识（key）列表，见下方说明 |

`quick_rules` 字段说明：

字符串数组，每个元素为规则组标识（key）。支持以下标识：

- `"restrict_vpc_access"`（限制OpenClaw互访）：内置特殊规则，入站拒绝 VPC CIDR（自动从 `siteconfig.VpcId` 查询），限制 OpenClaw 实例互访
- 其他标识：从 `config/clawpro_required_sg_rules.json` 的所有分类（`builtin`/`recommended`）中按 `key` 匹配。新增规则只需在 JSON 配置中添加 `rule_group`，无需修改后端代码

当前可用的规则组标识：

| 分类 | key | 名称 | 说明 |
|------|-----|------|------|
| 内置规则 | `allow_ssh` | 允许LinuxSSH登录 | 入站放通 TCP 22 |
| 内置规则 | `allow_internet` | 允许公网访问 | 出站放通所有流量 |
| 内置规则 | `allow_http` | 允许HTTP访问 | 入站放通 TCP 80 |
| 内置规则 | `allow_https` | 允许HTTPS访问 | 入站放通 TCP 443 |
| ClawPro 推荐 | `allow_rdp` | 允许Windows远程桌面 | 入站放通 TCP 3389 |
| ClawPro 推荐 | `allow_icmp` | 允许Ping | 入站放通 ICMP/ICMPv6 |
| ClawPro 推荐 | `allow_dns` | 允许DNS解析 | 出站放通 UDP 53 |

请求体示例：

```json
{
  "GroupName": "my-security-group",
  "GroupDescription": "安全组描述",
  "quick_rules": ["restrict_vpc_access", "allow_ssh", "allow_internet"]
}
```

- **响应：** 始终 JSON，透传腾讯云 `CreateSecurityGroup` 响应。创建成功后自动将安全组 ID 保存到站点配置，并异步将所有实例的安全组替换为仅包含新安全组（解绑之前绑定的所有旧安全组）。
  - 失败：`400/500 {"error": "..."}`

### `PUT /admin/config/security-group`

修改当前绑定的安全组属性。请求体为 JSON，直接透传给腾讯云 VPC [ModifySecurityGroupAttribute](https://cloud.tencent.com/document/product/215/15805) 接口。`SecurityGroupId` 由服务端自动填充为当前配置的安全组 ID。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：** 腾讯云 `ModifySecurityGroupAttribute` 请求参数 JSON（无需传 `SecurityGroupId`），例如：

```json
{
  "GroupName": "new-name",
  "GroupDescription": "新描述"
}
```

- **响应：** 始终 JSON，透传腾讯云 `ModifySecurityGroupAttribute` 响应。
  - 未配置安全组时：`400 {"error": "未配置安全组，请先创建或绑定安全组"}`
  - 失败：`400/500 {"error": "..."}`

### `GET /admin/config/security-group/policies`

查询当前安全组的规则列表。调用腾讯云 VPC [DescribeSecurityGroupPolicies](https://cloud.tencent.com/document/product/215/15804) 接口，`SecurityGroupId` 由服务端自动填充，响应直接透传。

- **权限：** 管理员
- **响应：** 始终 JSON，透传腾讯云 `DescribeSecurityGroupPolicies` 响应。
  - 未配置安全组时：`404 {"error": "未配置安全组"}`
  - 查询失败：`500 {"error": "..."}`

> **历史 POST / PUT / DELETE 已合并**：sg-ruleset-projection 方案下规则真相源在 DB（`rule_sets` 表），所有写操作统一走 `POST /admin/config/security-group/ruleset/rules`（整包提交）。

### `GET /admin/config/security-group/ruleset`

查询当前规则组（RuleSet）及其投影到的 ACTIVE 云安全组列表。

sg-ruleset-projection 方案下，RuleSet 是规则真相源，云端 SG 只是投影容器。管理员编辑规则时读本接口，保存时走 `POST /admin/config/security-group/ruleset/rules`。

- **权限：** 管理员
- **响应：** JSON

**已初始化（租户已经有 RuleSet）：**

```json
{
  "initialized": true,
  "id": 1,
  "name": "clawpro-default",
  "description": "Agent 默认安全组",
  "version": 7,
  "user_group_ids": [],
  "is_default": true,
  "rules": [
    {
      "direction": "INGRESS",
      "protocol": "TCP",
      "port": "22",
      "cidr_block": "0.0.0.0/0",
      "action": "ACCEPT",
      "policy_description": "允许 SSH",
      "is_required": true
    }
  ],
  "projected_to": [
    { "sg_id": "sg-xxxxxxxx", "sg_name": "clawpro-sg-acme-clawpro-default-01", "cvm_count": 1523 },
    { "sg_id": "sg-yyyyyyyy", "sg_name": "clawpro-sg-acme-clawpro-default-02", "cvm_count": 877 }
  ]
}
```

**未初始化（全新租户从未配过云安全组 / 未调用 POST rulesets）：**

```json
{ "initialized": false }
```

前端据此展示"创建规则组"引导卡片，而不是报错。

> **`cidr_block` 字段语义**：不仅支持 IPv4/IPv6 CIDR，还支持安全组 ID（`sg-*`）、IP 地址模板（`ipm-*`）、IP 地址模板组（`ipmg-*`）。详见下方 `POST /ruleset/rules` 的「规则字段约束」。

| 字段 | 说明 |
|------|------|
| initialized | 当前租户是否已有 RuleSet；`false` 时其他字段不返回 |
| id | RuleSet 主键 |
| name | 规则组名称。在 `(identifier, name)` 维度唯一（同租户下不可重名）。本期单一 RuleSet 固定 `clawpro-default`；未来支持多规则组按用户组分流时用于区分 |
| description | 管理员自定义备注，UI 展示用，不影响业务逻辑。长度上限 256 字符 |
| version | RuleSet 当前版本号（每次保存 +1），Guardian 判断漂移也用它 |
| user_group_ids | 预留字段：本规则组作用的用户组 ID 列表。本期恒为 `[]`，未来支持多规则组按用户组分流时填充 |
| is_default | 预留字段：是否为默认规则组。本期每租户只有一行且恒为 `true` |
| rules | 规则数组。`is_required=true` 的规则由 ClawPro 平台维护，前端只读 |
| projected_to | 当前 ACTIVE 状态的云安全组列表。每项含 `sg_id` / `sg_name` / `cvm_count`；`sg_name` 由 Guardian 每 5 分钟从云 API 反向同步，反映云端真实名称（管理员在云控制台改名后 ≤5min 自动对齐）；`cvm_count` 同样由 Guardian 巡检纠偏，可能有秒级延迟 |

### `POST /admin/config/security-group/rulesets`

创建规则组：建 RuleSet 行 + 云端建第一个 ACTIVE SG + 按 `auto_fix_rules` 与 SiteConfig 决定是否合并 ClawPro 必需规则 + 下发规则。幂等：同 `(identifier, name)` 的 RuleSet 已存在则直接返回现状，不新建。

全新租户（从未配过云安全组）首次使用时由前端"创建规则组"弹窗触发；日常不会重复调用。

> 后续支持多 RuleSet 按用户组分流时，本接口的 `name` 参数自然延展为"创建不同名称的规则组"；当前版本固定单规则组。

- **权限：** 管理员
- **审计 action：** `create_ruleset`
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "name": "clawpro-default",
  "description": "Agent 默认安全组",
  "rules": [
    {
      "direction": "INGRESS",
      "protocol": "TCP",
      "port": "22",
      "cidr_block": "0.0.0.0/0",
      "action": "ACCEPT",
      "policy_description": "允许 SSH"
    }
  ],
  "auto_fix_rules": false
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 规则组名称，默认 `clawpro-default`。同一租户下唯一（由 `(identifier, name)` UNIQUE 索引保证）。格式：**3-32 字符，必须以字母开头，仅允许字母、数字、短横线**（不允许下划线）。本期每租户仅允许一行 RuleSet |
| description | string | 否 | 管理员自定义备注，纯 UI 展示用，不影响业务逻辑。长度上限 256 字符。Bootstrap 存量迁移时默认填 `Agent 默认安全组` |
| rules | array | 否 | 管理员在弹窗里配置的自定义规则；`is_required` 由服务端合并必需规则时自动标记，请求体可不带 |
| auto_fix_rules | bool | 否 | 是否主动注入 ClawPro 必需规则，默认 `false`。**SiteConfig 启用任何 recommended 规则（gateway_ui_enable / browser_vnc_enable）时仍会兜底注入，无视本字段**。仅当 SiteConfig 全关 recommended 时本字段生效：`true` 注入 builtin（SSH / 出站）；`false` 不注入，rules 原样落盘 |
| import_from_sg_id | string | 否 | 若提供，则忽略 `rules` 字段，从该云端 SG 读取规则作为初始规则；不能是 ClawPro 自建 SG |

- **响应：** JSON

响应体与 `GET /admin/config/security-group/ruleset` 已初始化场景结构一致（`initialized: true` + id / name / description / version / rules / projected_to 等）。幂等调用时返回现有 RuleSet 的当前状态而不是 409。

- **错误：**
  - `name` 不合法：`400 {"error": "规则组名称不合法..."}`
  - 云 API 失败（建 SG 或下发规则失败，已回滚云端资源）：`500 {"error": "...", "drift_errors": [...]}`

### `POST /admin/config/security-group/ruleset/rules`

整包提交规则：**两相提交**语义 —— 先并发 fan-out 到目标 RuleSet 所有 ACTIVE 云 SG 全部成功，再 `UPDATE rule_sets.rules + version++`；任一 SG 失败则尝试回滚其他已更新的云 SG 并返回 `drift_errors`，本地 DB 不变。

**清空规则支持**：传 `rules: []` 时，云端规则会被真清空（走 `DescribeSecurityGroupPolicies` + `DeleteSecurityGroupPolicies` 路径，按 PolicyIndex 删除所有 ingress / egress；DB 也存 `[]`）。腾讯云 `ModifySecurityGroupPolicies` 不接受空 PolicySet，本接口内部自动选择正确的云 API 路径。

- **权限：** 管理员
- **审计 action：** `update_rule_set`
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "name": "clawpro-default",
  "rules": [
    {
      "direction": "INGRESS",
      "protocol": "TCP",
      "port": "22",
      "cidr_block": "0.0.0.0/0",
      "action": "ACCEPT",
      "policy_description": "允许 SSH"
    }
  ],
  "auto_fix_rules": false
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 要更新的规则组名称。缺省时 fallback 到 `clawpro-default`，与老客户端向后兼容；未来多 RuleSet 场景下前端必须显式传。格式：3-32 字符，字母开头，仅字母/数字/短横线 |
| rules | array | 是 | 完整规则数组（整包覆盖语义，不是增量）。空数组 `[]` 表示清空所有规则。`is_required` 字段由服务端合并必需规则时自动标记，请求体可以不带 |
| auto_fix_rules | bool | 否 | 是否主动注入 ClawPro 必需规则，默认 `false`。**SiteConfig 启用任何 recommended（gateway_ui_enable / browser_vnc_enable）时仍会兜底注入，无视本字段**。仅当 SiteConfig 全关时本字段生效 |

**规则字段约束**：
- `protocol` 取值 `TCP` / `UDP` / `ICMP` / `ICMPV6` / `GRE` / `ALL`（大小写不敏感）
- `protocol` 为 `ALL` / `ICMP` / `ICMPV6` / `GRE` / 空时，`port` 必须为 `ALL` 或留空（这些协议没有端口概念，混传会被腾讯云 API 拒绝，后端会前置 422 拦截）
- `cidr_block` 是"规则来源/目的"标识，支持 **5 种取值**：
  - IPv4 CIDR：如 `0.0.0.0/0` / `10.0.0.0/8` / `1.2.3.4/32`（也接受裸 IP `1.2.3.4`，服务端自动补 `/32`）
  - IPv6 CIDR：如 `::/0` / `2001:db8::/64`（含冒号即识别为 IPv6）
  - 腾讯云安全组 ID：以 `sg-` 开头（源/目的为另一个安全组，典型用途：跨服务授权）
  - 腾讯云 IP 地址模板 ID：以 `ipm-` 开头
  - 腾讯云 IP 地址模板组 ID：以 `ipmg-` 开头（⚠️ 注意与 `ipm-` 的前缀顺序，`ipmg-` 是地址组）
  - 服务端按前缀自动路由到腾讯云 SDK 对应字段（`CidrBlock` / `Ipv6CidrBlock` / `SecurityGroupId` / `AddressTemplate.AddressId` / `AddressTemplate.AddressGroupId`），这 5 种字段在腾讯云 API 层面互斥

- **响应：** JSON（成功）

```json
{
  "version": 8,
  "synced": 2,
  "drifted": 0,
  "drift_errors": []
}
```

| 字段 | 说明 |
|------|------|
| version | 保存后 RuleSet 的新版本号 |
| synced | 成功同步规则的云 SG 数 |
| drifted | 成功路径下恒为 0；保留兼容字段 |
| drift_errors | 成功路径下为空数组 |

- **错误：**
  - 规则格式非法（CIDR / 端口解析失败 / `protocol=ALL` 配具体端口等）：`400 {"error": "...", "drift_errors": [{"sg_id": "", "error": "..."}]}`
  - 云 API fan-out 失败（两相提交回滚语义）：`500 {"error": "apply rules failed on N SGs", "drift_errors": [{"sg_id": "sg-xxx", "error": "..."}, ...]}`；此时 RuleSet `version` 不变、云端已尽力回滚
  - 前端应将 `drift_errors` 展示给管理员定位是哪条规则 / 哪个 SG 出问题

### `POST /admin/config/security-group/ruleset/import-from-sg`

从云账号下任一云安全组读规则 → 覆盖当前 RuleSet → fan-out。内部复用 `POST /ruleset/rules` 的两相提交路径。

运行时主动触发，用于"从其他安全组导入规则"场景。校验：源 SG 不能是 ClawPro 自建（名称以 `clawpro-sg-` 开头会被拒绝，避免循环依赖）。

- **权限：** 管理员
- **审计 action：** `import_rules_from_sg`
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "name": "clawpro-default",
  "source_sg_id": "sg-abcd1234",
  "auto_fix_rules": false
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 要导入规则的目标规则组名称。缺省时 fallback 到 `clawpro-default`。格式：3-32 字符，字母开头，仅字母/数字/短横线 |
| source_sg_id | string | 是 | 要作为规则模板的源安全组 ID，必须在当前账号/地域下存在 |
| auto_fix_rules | bool | 否 | 是否主动注入 ClawPro 必需规则，默认 `false`。**SiteConfig 启用任何 recommended 时仍会兜底注入，无视本字段**。仅当 SiteConfig 全关时本字段生效：`true` 在源 SG 规则上 merge builtin（SSH / 出站）；`false` 严格保留源 SG 规则原貌（"纯导入"）|

- **响应：** JSON（成功）

```json
{
  "version": 9,
  "synced": 2,
  "drifted": 0,
  "drift_errors": [],
  "imported_from": "sg-abcd1234"
}
```

| 字段 | 说明 |
|------|------|
| version | 导入后 RuleSet 的新版本号 |
| synced | 成功同步到云端的 SG 数 |
| drifted | 成功路径下恒为 0 |
| drift_errors | 成功路径下为空数组 |
| imported_from | 导入源 SG ID |

- **错误：**
  - 源 SG 为空：`400 {"error": "source_sg_id 不能为空"}`
  - 源 SG 是 ClawPro 自建：`400 {"error": "不允许从 ClawPro 自建安全组导入（会造成循环依赖）"}`
  - 源 SG 读取失败：`500 {"error": "describe source sg policies: ..."}`
  - fan-out 失败（两相提交回滚语义同上）：`500 {"error": "...", "drift_errors": [...]}`

### `POST /admin/config/security-group/ruleset/rules/reorder`

调整出入站规则在规则组内的匹配顺序。腾讯云安全组采用「自上而下匹配」语义（位置越靠前优先级越高），本接口让管理员能精确控制每条规则的相对顺序，而不会无意中改动任何规则的内容（Direction/Protocol/Port/CidrBlock/Action/Description）。

设计要点：

1. **只动顺序，不动内容**：仅按 `ordered_fingerprints` 重排 `RuleSet.Rules` 数组，不会修改任何规则字段。
2. **不能用于删除规则**：未在 `ordered_fingerprints` 中列出的规则会按其在原 `RuleSet.Rules` 中的相对顺序追加到末尾——即「漏列等价于放最后」，不会丢规则。如需删除某条规则，请改用 `POST /admin/config/security-group/ruleset/rules` 整包提交。
3. **必需规则可参与排序**：管理员可决定 ClawPro 必需规则（`is_required=true`）的位置；但若 SiteConfig 启用了 recommended 必需规则注入，缺失的必需规则在底层 `MergeRequiredRules` 仍会被兜底追加到末尾，保证安全。
4. **复用整包提交的两相提交语义**：内部走 `UpdateRuleSetRulesInternal(autoFixRules=false)` → 先写 DB → 向所有 ACTIVE SG fan-out → 任一 SG 失败即整体回滚（DB 与所有云端 SG 全部恢复到旧顺序）。失败响应 HTTP **409**，DB 不变。
5. **不再额外注入 recommended 规则**：reorder 是「管理员明确控制顺序」的语义，不应在内部再被合入用户未感知的新规则。

- **权限：** 管理员
- **审计 action：** `security_group_ruleset_reorder`
- **Content-Type：** `application/json`

#### 请求体

```json
{
  "name": "ClawPro-Default",
  "ordered_fingerprints": [
    "INGRESS|TCP|443|0.0.0.0/0|ACCEPT",
    "INGRESS|TCP|22|0.0.0.0/0|ACCEPT",
    "EGRESS|ALL|ALL|0.0.0.0/0|ACCEPT"
  ]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 要排序的规则组名称。缺省（或为空白字符串）时走默认 RuleSet ClawPro-Default。格式：3-32 字符，字母开头，仅字母/数字/短横线 |
| ordered_fingerprints | string[] | 是 | 期望的规则顺序（首位 = 第 0 条）。每个 fingerprint 必须从 `GET /admin/config/security-group/ruleset` 响应里每条 rule 的 fingerprint 字段直接复用，前端无需自己拼。不允许重复，不允许空字符串；漏列的规则按原相对顺序追加到末尾 |

####  fingerprint 拼接算法（需要独立计算时参考）

正常调用路径上前端不需要自己算 fingerprint（直接读 GET ruleset 响应里的 `fingerprint` 字段即可）。以下算法仅在需要本地预计算 / 调试比对时参考。

后端 `Rule.Fingerprint()` 用 5 个字段拼接：

```
DIRECTION|PROTOCOL|PORT|CIDR_BLOCK|ACTION
```

但**每个字段都需要按下表先归一化再拼**，否则会被服务端识别为「不存在的指纹」（返回 400）：

| 字段 | 取值来源（GET ruleset 响应字段） | 归一化规则 |
|------|----------------------------------|-----------|
| `DIRECTION` | `direction` | TrimSpace → 大写。允许 `INGRESS` / `EGRESS`；同时容忍 `IN` → `INGRESS`、`OUT` → `EGRESS` |
| `PROTOCOL` | `protocol` | TrimSpace → 大写。例：`TCP` / `UDP` / `ICMP` / `ALL` |
| `PORT` | `port` | TrimSpace；空串和 `ALL`（任意大小写）→ `ALL`；`"22-22"` 这类首尾相同的范围 → `"22"`；其余原样保留（如 `"80-443"`） |
| `CIDR_BLOCK` | `cidr_block` | TrimSpace；裸 IPv4（如 `192.168.1.1`）补 `/32`；裸 IPv6（如 `::1`）补 `/128`；带 `/` 的 CIDR 走 `net.ParseCIDR` 规范化（IPv6 大小写归一）；`sg-xxx` / `ipm-xxx` / `ipmg-xxx` 这类资源标识**原样保留**，不补前缀 |
| `ACTION` | `action` | TrimSpace → 大写。`ACCEPT` / `DROP` |

> 🚨 **重要提醒**：`GET /admin/config/security-group/ruleset` 的响应中每条 rule 都会额外携带一个后端计算好的 `fingerprint` 字段（`Rule` 本体表示字段为 `direction` / `protocol` / `port` / `cidr_block` / `action` / `policy_description` / `is_required`）。前端不需要手拼，直接取用即可；上面的归一化表与下面 TS 参考实现仅为需要本地预计算 / 调试比对时参考。

TypeScript 参考实现：

```ts
type Rule = {
  direction: string;
  protocol: string;
  port: string;
  cidr_block: string;
  action: string;
  policy_description?: string;
  is_required?: boolean;
};

function normalizeDirection(d: string): string {
  const u = d.trim().toUpperCase();
  if (u === "INGRESS" || u === "IN") return "INGRESS";
  if (u === "EGRESS" || u === "OUT") return "EGRESS";
  return u;
}

function normalizePort(p: string): string {
  const t = p.trim();
  if (t === "" || t.toUpperCase() === "ALL") return "ALL";
  const i = t.indexOf("-");
  if (i > 0 && t.slice(0, i).trim() === t.slice(i + 1).trim()) {
    return t.slice(0, i).trim();
  }
  return t;
}

function normalizeCIDR(c: string): string {
  const t = c.trim();
  if (t === "") return "";
  // 资源标识（sg-xxx / ipm-xxx / ipmg-xxx）原样保留
  if (/^(sg-|ipm-|ipmg-)/i.test(t)) return t;
  // 已含 / 直接返回（前端无 net.ParseCIDR，IPv6 大小写差异极少出现，可视情况自行 toLowerCase）
  if (t.includes("/")) return t;
  // 裸 IP 补前缀
  return t.includes(":") ? `${t}/128` : `${t}/32`;
}

export function ruleFingerprint(r: Rule): string {
  return [
    normalizeDirection(r.direction),
    r.protocol.trim().toUpperCase(),
    normalizePort(r.port),
    normalizeCIDR(r.cidr_block),
    r.action.trim().toUpperCase(),
  ].join("|");
}
```

#### 端到端调用示例（前端拖拽排序后保存）

**Step 1**：拉取当前 RuleSet（返回的每条 rule 都带 `fingerprint` 字段）

```bash
curl -sS -H "Authorization: Bearer <ADMIN_TOKEN>" \
  http://<host>/admin/config/security-group/ruleset
```

响应（节选）：

```json
{
  "initialized": true,
  "name": "ClawPro-Default",
  "version": 11,
  "rules": [
    {
      "direction": "INGRESS", "protocol": "TCP", "port": "22",
      "cidr_block": "0.0.0.0/0", "action": "ACCEPT", "is_required": true,
      "fingerprint": "INGRESS|TCP|22|0.0.0.0/0|ACCEPT"
    },
    {
      "direction": "INGRESS", "protocol": "TCP", "port": "443",
      "cidr_block": "0.0.0.0/0", "action": "ACCEPT",
      "fingerprint": "INGRESS|TCP|443|0.0.0.0/0|ACCEPT"
    },
    {
      "direction": "EGRESS", "protocol": "ALL", "port": "ALL",
      "cidr_block": "0.0.0.0/0", "action": "ACCEPT", "is_required": true,
      "fingerprint": "EGRESS|ALL|ALL|0.0.0.0/0|ACCEPT"
    }
  ]
}
```

> `fingerprint` 是**响应派生字段**，只出现在 GET 响应中；不会被写入数据库，也不在 `POST /ruleset/rules` 等入参中生效（即使传了也会被忽略）。

**Step 2**：前端拖拽排序后，直接取每条 rule 的 `fingerprint` 字段拼成数组：

```ts
// rules 是拖拽排序后的状态数组
const orderedFingerprints = rules.map(r => r.fingerprint);
```

**Step 3**：POST 提交新顺序（下面把第二条 443 提到最前）

```bash
curl -sS -X POST \
  -H "Authorization: Bearer <ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  http://<host>/admin/config/security-group/ruleset/rules/reorder \
  -d '{
    "ordered_fingerprints": [
      "INGRESS|TCP|443|0.0.0.0/0|ACCEPT",
      "INGRESS|TCP|22|0.0.0.0/0|ACCEPT",
      "EGRESS|ALL|ALL|0.0.0.0/0|ACCEPT"
    ]
  }'
```

> 不传 `name` 即走 `ClawPro-Default`。本期只有一个 RuleSet，前端通常无需关心 `name`。

#### 响应（成功）

```json
{
  "version": 12,
  "synced": 2,
  "drifted": 0,
  "drift_errors": []
}
```

| 字段 | 说明 |
|------|------|
| `version` | 排序后 RuleSet 的新版本号（与 `POST /ruleset/rules` 一致地递增） |
| `synced` | 成功同步到云端的 ACTIVE SG 数 |
| `drifted` | 成功路径下恒为 0；失败路径下携带在 409 错误响应中 |
| `drift_errors` | 成功路径下为空数组；失败路径下携带每个失败 SG 的错误详情 |

#### 错误

| HTTP | 触发条件 | 响应示例 |
|------|----------|----------|
| 400 | 请求体非 JSON | `{"error": "请求参数格式错误: ..."}` |
| 400 | `name` 不合法（不满足 `^[A-Za-z][A-Za-z0-9-]{2,31}$`） | `{"error": "规则组名称不合法：需 3~32 个字符，必须以字母开头，仅允许字母、数字、短横线（当前=...)"}` |
| 400 | `ordered_fingerprints` 为空数组或未传 | `{"error": "ordered_fingerprints 不能为空"}` |
| 400 | `ordered_fingerprints` 含空字符串 | `{"error": "ordered_fingerprints 含空字符串"}` |
| 400 | `ordered_fingerprints` 含重复 fingerprint | `{"error": "ordered_fingerprints 含重复 fingerprint: ..."}` |
| 400 | 入参 fingerprint 在当前 RuleSet 中找不到（前端拼错或 RuleSet 已被他人改动） | `{"error": "fingerprint 不存在于规则组中: ...（请先 GET /admin/config/security-group/ruleset 重新获取最新指纹列表）"}` |
| 400 | RuleSet 存在但 `rules` 为空数组 | `{"error": "规则组当前无任何规则，无需排序"}` |
| 404 | 指定 `name` 的 RuleSet 不存在 | `{"error": "规则组不存在 (name=...)"}` |
| 409 | fan-out 任一 SG 失败（已自动回滚，DB 不变，version 保留旧值） | `{"error": "...", "drift_errors": [{"sg_id":"sg-xxx","error":"..."}], "drifted": 1, "synced": 0, "version": 11}` |

> 拿到 400「fingerprint 不存在于规则组中」时的标准恢复路径：重新 GET `/admin/config/security-group/ruleset` 拉到最新 `rules`，重新取其 `fingerprint` 字段后让用户重新排序再提交（一般是别的管理员在你打开页面期间改了规则）。

### `GET /admin/config/security-group/cloud-policies`

查询任一云端安全组的规则明细，供"从其他安全组导入规则"弹窗预览使用。

透传腾讯云 VPC [DescribeSecurityGroupPolicies](https://cloud.tencent.com/document/product/215/15804) 接口。和 `/policies` 不同：`/policies` 读本租户已配置 SG 的规则（内部从 site_configs 取 sg_id），`/cloud-policies` 读请求方指定的任意 SG。

- **权限：** 管理员
- **参数（Query）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| sg_id | string | 是 | 目标安全组 ID，必须在当前账号/地域下可见 |

- **响应：** 始终 JSON，透传腾讯云 `DescribeSecurityGroupPolicies` 响应
  - `sg_id` 为空：`400 {"error": "sg_id 不能为空"}`
  - 查询失败：`500 {"error": "..."}`

### `GET /admin/config/security-group/list`

列出当前账号/地域下的安全组，支持分页和过滤查询。

主要用途：**"从其他安全组导入规则"弹窗的模板候选列表**。因此响应中会过滤掉如下两类安全组，避免管理员误选导致循环依赖：

1. 已在 `managed_sg_pool` 表中的 SG（任意状态：ACTIVE / FROZEN / DRAINING）
2. 名称以 `clawpro-sg-` 前缀命名的 SG（即使未入 `managed_sg_pool`，也视为 ClawPro 管理的 SG）

`total_count` 会相应扣减过滤掉的条数，使前端分页一致。

- **权限：** 管理员
- **参数（Query）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| offset | string | 否 | 偏移量，默认 `0` |
| limit | string | 否 | 每页数量，默认 `20`，最大 `100` |
| keyword | string | 否 | 模糊搜索关键字（按安全组名称过滤） |
| security_group_id | string | 否 | 精确匹配安全组 ID（多个用英文逗号分隔） |

- **响应：** 始终 JSON

```json
{
  "ok": true,
  "security_groups": [
    {
      "security_group_id": "sg-xxxxxxxx",
      "security_group_name": "my-sg",
      "security_group_desc": "描述",
      "is_default": false
    }
  ],
  "total_count": 50
}
```

  - 查询失败：`500 {"error": "..."}`

### `GET /admin/config/security-group/required-rules`

查询内部配置的 ClawPro 所需安全组规则列表（来自 `config/clawpro_required_sg_rules.json`）。

规则按分类组织，目前包含两类：
- `builtin`（内置规则）：平台运行所需的基础安全组规则
- `recommended`（ClawPro 推荐规则）：ClawPro 推荐的安全组规则

- **权限：** 管理员
- **参数（Query）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 否 | 规则分类类型，可选值：`builtin`（内置规则）、`recommended`（推荐规则）、`all`（返回所有分类），默认 `builtin` |

- **响应：** 始终 JSON

```json
{
  "ok": true,
  "data": {
    "categories": [
      {
        "type": "builtin",
        "label": "内置规则",
        "rule_groups": [
          {
            "key": "allow_ssh",
            "name": "允许LinuxSSH登录",
            "rules": [
              {
                "direction": "ingress",
                "protocol": "TCP",
                "port": "22",
                "cidr_block": "0.0.0.0/0",
                "ipv6_cidr": "",
                "action": "ACCEPT",
                "description": "放通 Linux SSH 登录（IPv4）"
              }
            ]
          }
        ]
      },
      {
        "type": "recommended",
        "label": "ClawPro 推荐规则",
        "rule_groups": [
          {
            "key": "allow_rdp",
            "name": "允许Windows远程桌面",
            "rules": [
              {
                "direction": "ingress",
                "protocol": "TCP",
                "port": "3389",
                "cidr_block": "0.0.0.0/0",
                "ipv6_cidr": "",
                "action": "ACCEPT",
                "description": "放通 Windows 远程桌面（IPv4）"
              }
            ]
          }
        ]
      }
    ]
  }
}
```

### `GET /admin/config/security-group/check-rules`

检查指定安全组是否满足 ClawPro 所需规则。满足时返回空的 `missing_rules` 列表；不满足时返回缺少的规则列表。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| security_group_id | string | 是 | 要检查的安全组 ID |

- **响应：** 始终 JSON

```json
{
  "ok": true,
  "data": {
    "missing_rules": [
      {
        "direction": "ingress",
        "protocol": "TCP",
        "port": "22",
        "cidr_block": "0.0.0.0/0",
        "ipv6_cidr": "",
        "action": "ACCEPT",
        "description": "放通Linux SSH登录"
      }
    ]
  }
}
```

  - 缺少参数：`400 {"error": "security_group_id 参数不能为空"}`
  - 查询失败：`500 {"error": "..."}`

### `GET /admin/vpc/cloud`

从腾讯云获取当前 Region 下的 VPC 列表，支持分页查询和过滤。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| offset | string | 否 | 偏移量，默认 `"0"` |
| limit | string | 否 | 每页返回数量，默认 `"20"`，最大 `"100"` |
| vpc_name | string | 否 | 按 VPC 名称过滤（映射为腾讯云 Filter `vpc-name`） |
| vpc_id | string | 否 | 按 VPC ID 过滤（映射为腾讯云 Filter `vpc-id`） |

- **响应：** 始终 JSON

```json
{
  "ok": true,
  "vpcs": [
    {"vpc_id": "vpc-xxx", "name": "my-vpc", "cidr_block": "172.16.0.0/16"}
  ],
  "total_count": 50
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| vpcs | array | 当前页的 VPC 列表 |
| vpcs[].vpc_id | string | VPC ID |
| vpcs[].name | string | VPC 名称 |
| vpcs[].cidr_block | string | CIDR 地址段 |
| total_count | uint64 | 符合条件的 VPC 总数，用于判断是否还有下一页 |

> **分页遍历示例：** 第一次请求 `?offset=0&limit=100`，若 `total_count=250`，则继续请求 `?offset=100&limit=100`，再请求 `?offset=200&limit=100`，直到 `offset >= total_count`。

失败时返回错误信息：

- `500 {"error": "创建 VPC 客户端失败: ..."}`
- `500 {"error": "查询 VPC 列表失败: ..."}`

### `GET /admin/subnet/cloud`

获取指定 VPC 指定可用区下的子网列表，包含剩余 IP 容量（供"按剩余 IP 加权挑选子网"功能使用）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| vpc_id | string | 是 | VPC ID（Query 参数） |
| zone | string | 是 | 可用区（Query 参数） |

- **响应：** 始终 JSON

```json
{
  "ok": true,
  "subnets": [
    {
      "subnet_id": "subnet-xxx",
      "name": "my-subnet",
      "cidr_block": "172.16.1.0/24",
      "available_ip_count": 201,
      "total_ip_count": 252
    }
  ]
}
```

- `available_ip_count` uint：子网当前可用 IP 数（来自腾讯云 VPC `DescribeSubnets.AvailableIpAddressCount`）
- `total_ip_count` uint：子网 IP 总数（含网关、保留地址等）

失败时返回错误信息：

- `400 {"error": "vpc_id 参数不能为空"}`
- `400 {"error": "zone 参数不能为空"}`
- `500 {"error": "创建 VPC 客户端失败: ..."}`
- `500 {"error": "查询子网列表失败: ..."}`

### `GET /admin/models`

模型配置页面。

- **权限：** 管理员
- **HTML 响应：** HTML 页面
- **JSON 响应：** `{"models": [...], "default_model_id": 0}`（`APIKey` 仅返回脱敏值，`default_model_id` 为当前默认模型 ID，0 表示无默认）

模型对象字段（按大写字段名输出，部分字段在 `AIModel.MarshalJSON` / `modelWithVisibility.MarshalJSON` 中做了语义包装）：

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint | 模型记录 ID |
| Provider | string | 模型提供商名称 |
| ModelID | string | 原始模型 ID（保真，可含 `/`、`:` 等） |
| ModelName | string | 模型显示名称（留空时与 `ModelID` 相同） |
| APIKey | string | 脱敏后的 API Key：长度 `>= 12` 时保留前 4 / 后 4 字符，中间按原长度用 `*` 替换（至少隐藏 4 个字符）；长度 `1-11` 时全部替换为等长 `*`；空值返回空字符串 |
| URL | string | 上游 API 地址 |
| ModelType | string | 接口类型：`openai-completions` 或 `anthropic-messages` |
| InputTypes | string | 支持的输入类型（JSON array 字符串，如 `["text"]`） |
| ContextLen | int | 上下文长度 |
| MaxTokens | int | 单请求最大输出 token 数，`0` 表示不限 |
| CustomHTTPHeaders | string | 自定义 HTTP 请求头（JSON 对象字符串） |
| QuotaDay | int | 每日 token 上限，`-1` 表示不限 |
| Enabled | bool | **兼容字段，值 = 真实 `Visible`**（旧前端把它当作"用户可见"开关读取，新前端不应再用此字段判断"是否启用"） |
| EnabledStatus | bool | 真实的"是否启用（开启/关闭）"状态。新前端读取此字段判断模型是否可用于 LLM 路由 |
| VisibilityType | string | 可见范围类型：`all`（全部用户）或 `group`（按分组），默认为 `all` |
| visibility_groups | array | 可见分组列表，仅当 `VisibilityType=group` 时有值；其余情况为空数组 `[]` |
| visibility_groups[].group_id | uint | 分组 ID |
| visibility_groups[].group_name | string | 分组名称 |

> **不再输出**：`Visible`（已被 `MarshalJSON` 显式删除）。`APIKey` 只输出脱敏展示值，禁止返回明文。
>
> **前端字段读取约定**：
> - 旧前端：`model.Enabled` 表达"用户可见"语义（实际由后端把 `Visible` 值回填到 `Enabled` 字段）。
> - 新前端：判断"是否启用 / 开启关闭"读 `EnabledStatus`；判断"用户可见"读 `Enabled`（语义即原 `Visible`）。

```json
{
  "models": [
    {
      "ID": 1,
      "Provider": "腾讯云 DeepSeek",
      "ModelID": "DeepSeek V3 0324",
      "ModelName": "DeepSeek V3 0324",
      "APIKey": "sk-1**********abcd",
      "URL": "https://api.example.com/v1",
      "ModelType": "openai-completions",
      "InputTypes": "[\"text\"]",
      "ContextLen": 128000,
      "MaxTokens": 8192,
      "CustomHTTPHeaders": "",
      "QuotaDay": -1,
      "Enabled": true,
      "EnabledStatus": true,
      "VisibilityType": "group",
      "visibility_groups": [
        { "group_id": 1, "group_name": "高层管理" },
        { "group_id": 3, "group_name": "设计团队" }
      ]
    },
    {
      "ID": 2,
      "Provider": "OpenAI",
      "ModelID": "gpt-4o",
      "ModelName": "gpt-4o",
      "APIKey": "sk-p************wxyz",
      "URL": "https://api.openai.com/v1",
      "ModelType": "openai-completions",
      "InputTypes": "[\"text\",\"image\"]",
      "ContextLen": 128000,
      "MaxTokens": 8192,
      "CustomHTTPHeaders": "",
      "QuotaDay": -1,
      "Enabled": true,
      "EnabledStatus": true,
      "VisibilityType": "all",
      "visibility_groups": []
    }
  ],
  "default_model_id": 2
}
```

> **内置占位记录：** `ai_models` 表中包含一条 `provider='hatchery', model_id='custom'` 的内置记录，由 `SeedModels()` 在启动时自动创建。该记录不是实际模型，其 `Enabled` 字段控制管理后台的"开放自定义模型"开关。此记录不可删除、不可编辑（仅可通过 toggle 接口切换启用状态），在实例模型选择列表中会被过滤掉。

### `POST /admin/models/create`

创建新的 AI 模型配置。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| provider | string | 否 | 模型提供商名称，未传时默认为 `自定义模型` |
| model_id | string | 是 | 模型标识 |
| model_name | string | 否 | 模型显示名称（留空则使用 `model_id`） |
| api_key | string | 是 | API 密钥 |
| url | string | 是 | API 地址（必须为合法的 `http://` 或 `https://` 地址） |
| model_type | string | 是 | 接口类型，枚举值：`openai-completions` 或 `anthropic-messages` |
| input_types | string | 否 | 支持的输入类型，仅允许 `text`、`image`，可通过重复传递该参数来指定多个值（如 `input_types=text&input_types=image`）。缺省为 `text` |
| quota_day | int | 否 | 每日配额（-1=不限） |
| context_len | int | 否 | 上下文长度 |
| max_tokens | int | 否 | Agent 单次请求模型最大输出 Token 数（传入值小于等于 0 时默认 8192） |
| custom_http_headers| string | 否 | Agent 请求模型时的自定义 HTTP 头部，格式为 JSON 字符串，如 `{"Header-Key": "Header-Value"}`|

> **注意：** `provider` 字段为可选参数，未传时默认为 `自定义模型`。

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`400 {"error": "模型ID、API Key、URL 和接口类型不能为空"}` / `400 {"error": "URL 格式无效: ..."}` / `400 {"error": "URL 必须以 http:// 或 https:// 开头"}` / `400 {"error": "接口类型无效，仅支持 openai-completions 或 anthropic-messages"}` / `400 {"error": "输入类型无效，仅支持 text、image"}` / `400 {"error": "每日Token上限必须为 -1 或非负整数"}` / `500 {"error": "创建失败"}`

### `POST /admin/models/update`

更新 AI 模型配置。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 模型 ID（Query 参数） |
| model_id | string | 是 | 模型标识 |
| api_key | string | 否 | API 密钥（留空则不更新） |
| url | string | 是 | API 地址（必须为合法的 `http://` 或 `https://` 地址） |
| model_type | string | 是 | 接口类型，枚举值：`openai-completions` 或 `anthropic-messages` |
| input_types | string | 否 | 支持的输入类型，仅允许 `text`、`image`，可通过重复传递该参数来指定多个值（如 `input_types=text&input_types=image`）。缺省为 `text` |
| quota_day | int | 否 | 每日配额（-1=不限） |
| context_len | int | 否 | 上下文长度 |
| max_tokens | int | 否 | Agent 单次请求模型最大输出 Token 数。**未传则保留原值**；**显式传值（含 `0`）则更新，`0` 表示不限输出**；传负数返回 400 |
| custom_http_headers| string | 否 | Agent 请求模型时的自定义 HTTP 头部，JSON 字符串，如 `{"Header-Key": "Header-Value"}`。**未传则保留原值**；**显式传空对象 `{}` 则清空自定义头** |

> **注意：** `provider` 字段固定为 `自定义模型`，无需传递。
>
> **max_tokens / custom_http_headers 的显式语义：** 这两个字段区分「未传」与「显式传值」——请求中不带该参数时保留数据库原值；带了该参数则按值更新。因此 `max_tokens=0` 会显式写入 0（下发到实例时不设输出上限，即不限输出），`custom_http_headers={}` 会显式清空已配置的自定义头。
>
> **实例同步：** 更新成功后，**自动异步同步**变更到所有绑定该 model 的 OpenClaw 实例（含 primary 与 fallback），下发 `set_model.sh` 全量重写实例 CVM 侧 openclaw.json 的 provider 配置 + primary/fallback 链，确保实例配置与 DB 一致。同步为异步执行（并发上限 10），失败仅记日志，不影响 DB 更新结果，也不阻塞 HTTP 响应。

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`404 {"error": "模型不存在"}` / `400 {"error": "模型ID、URL 和接口类型不能为空"}` / `400 {"error": "URL 格式无效: ..."}` / `400 {"error": "URL 必须以 http:// 或 https:// 开头"}` / `400 {"error": "接口类型无效，仅支持 openai-completions 或 anthropic-messages"}` / `400 {"error": "输入类型无效，仅支持 text、image"}` / `400 {"error": "每日Token上限必须为 -1 或非负整数"}` / `400 {"error": "最大输出 Token 数必须为非负整数"}` / `403 {"error": "系统内置记录不可修改"}`

### `POST /admin/models/delete`

删除 AI 模型配置。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 模型 ID（Query 参数） |

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`404 {"error": "模型不存在"}` / `403 {"error": "系统内置记录不可删除"}`

### `POST /admin/models/toggle`

切换模型「用户可见」状态（`visible` 字段）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 模型 ID（Query 参数） |

- **逻辑：**
  - 仅翻转 `visible` 字段，不影响 `enabled`，也不影响已绑定该模型的 agent 在 LLM 路由上的可用性。
  - 若关闭可见（`visible` 由 `true` 翻为 `false`）后该模型为站点默认模型，则联动清除站点 `default_model_id`，避免新建实例自动注入用户不可见的模型。
- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`404 {"error": "模型不存在"}`

### `POST /admin/models/toggle-enabled` 🆕

切换模型「启用」状态（`enabled` 字段）。`enabled` 控制模型是否可用于 LLM 路由——关闭后已绑定该模型的 agent 在调用上游 API 时也会被拒绝，同时该模型也无法被新用户绑定。

- **权限：** 管理员
- **方法：** POST（GET 返回 `405 Method Not Allowed`）
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 模型 ID（Query 参数） |

- **逻辑：**
  - 仅翻转 `enabled` 字段，不影响 `visible`。
  - 若关闭启用（`enabled` 由 `true` 翻为 `false`）后该模型为站点默认模型，则联动清除站点 `default_model_id`。
  - 与 `/admin/models/toggle` 的语义解耦：`toggle` 仅控制用户端是否可见，`toggle-enabled` 才真正影响 LLM 路由可用性。
- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`404 {"error": "模型不存在"}` / `405 {"error": "Method Not Allowed"}`

### `POST /admin/models/toggle-default`

切换模型的默认配置状态。默认模型会在用户创建新 OpenClaw 实例时自动注入。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 模型 ID（Query 参数） |

- **逻辑：** Toggle 语义——若该模型已是默认则取消，否则设为默认（单选，同时只有一个默认模型）。仅用户可见（`enabled=true`）的模型可设为默认。
- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`404 {"error": "模型不存在"}` / `400 {"error": "请先开启该模型的「用户可见」后再设为默认"}`

### `POST /admin/models/visibility`

更新模型的可见范围配置。支持设置模型对"全部用户"或"按分组"可见。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 模型 ID（Query 参数） |

- **请求体：**

```json
{
    "visibility_type": "group",
    "group_ids": [1, 3, 5]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| visibility_type | string | 是 | 可见范围类型：`all`（全部用户）或 `group`（按分组） |
| group_ids | uint[] | 条件必填 | 分组 ID 列表，当 `visibility_type=group` 时必填且非空 |

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：
    - `400 {"error": "visibility_type 必须为 all 或 group"}`
    - `400 {"error": "按分组可见时必须选择至少一个分组"}`
    - `400 {"error": "分组不存在: [5, 9]"}`
    - `400 {"error": "请求参数格式错误"}`
    - `404 {"error": "模型不存在"}`
    - `500 {"error": "更新可见范围失败"}`

### `POST /admin/models/connectivity`

测试模型的连通性。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 否 | 模型 ID（Query 参数） |

- **请求体：**

```json
{
    "url": "https://api.openai.com/v1",
    "api_key": "<KEY>",
    "model_type": "openai-completions",
    "model": "gpt-3.5-turbo"
}
```

| 请求体字段 | 类型 | 必填 | 说明 |
|----------|------|-------|------|
| url | string | 是 | API 地址 |
| api_key | string | 是 |  API 密钥 |
| model_type | string | 是 |  接口类型，枚举值：`openai-completions` 或 `anthropic-messages` |
| model | string | 是 |  模型名称 |

- **请求说明：**
  1. 按已保存模型 ID 探测——用于已存在模型的健康检查：POST /admin/models/connectivity?id=42

  2. 用临时凭证探测——常用于新增/编辑模型表单未保存即试连：POST /admin/models/connectivity Content-Type: application/json {"url":"https://api.openai.com/v1","api_key":"sk-...","model_type":"openai-completions","model":"gpt-3.5-turbo"}

  > **探活方式：** 使用 chat 探活（发送极简对话请求，max_tokens=1），同时验证 API 地址可达性、API Key 有效性及模型 ID 正确性。

- **JSON 响应：**

模型无法连通的响应：

```json
{
  "ok":false, 
  "kind":"invalid_api_key", 
  "message":"...",
	"status_code":401, 
  "snippet":"...", 
  "latency_ms":120
}
```

模型可以连通的响应

```json
{
  "ok":true, 
  "latency_ms":234
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| ok | bool | 模型是否可以连通 |
| kind | string | 错误类型 |
| message | string | 错误详细信息 |
| status_code | int | 后端检测模型连通性时 provider 返回的 HTTP 状态码 |
| snippet | string | 响应片段 |
| latency_ms | int | 延迟（毫秒） |

错误类型与信息对应：
| 类型 | 信息 |
|-------|-------|
| network_unreachable | 网络不通或上游地址无法访问 |
| invalid_api_key | API Key 无效 |
| forbidden | 凭证有效但被拒绝（账户/区域/模型未授权） |
| rate_limited | 上游限流，请稍后重试 |
| upstream_server_error | 上游服务异常 |
| upstream_client_error | 上游拒绝请求 |

- **错误响应：**

- `400 {"error": "URL 格式无效"}`
- `400 {"error": "URL 必须以 http:// 或 https:// 开头"}`
- `400 {"error": "接口类型无效，仅支持 openai-completions 或 anthropic-messages"}`
- `400 {"error": "请求体格式错误，应为 JSON"}`
- `400 {"error": "id 参数非法"}`
- `400 {"error": "URL 不能为空"}`
- `400 {"error": "model_type 不能为空"}`
- `400 {"error": "api_key 不能为空"}`
- `400 {"error": "model 不能为空"}`
- `404 {"error": "模型不存在"}`
- `405 {"error": "method not allowed"}`

### `GET /admin/user-groups/associated-models`

查询指定用户组关联的模型列表。用于删除用户组前提示用户该组关联了哪些模型。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_id | uint | 是 | 用户组 ID（Query 参数） |

- **JSON 响应：**

```json
{
    "ok": true,
    "count": 3,
    "models": [
        { "id": 1, "provider": "腾讯云 DeepSeek", "model_id": "DeepSeek V3 0324" },
        { "id": 2, "provider": "OpenAI", "model_id": "gpt-4o" },
        { "id": 5, "provider": "Claude", "model_id": "claude-3-opus" }
    ]
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| ok | bool | 固定 `true` |
| count | int | 关联的模型数量 |
| models | array | 关联的模型列表 |
| models[].id | uint | 模型 ID |
| models[].provider | string | 模型提供商 |
| models[].model_id | string | 模型标识 |

- **错误响应：**
  - `400 {"error": "缺少 group_id 参数"}`
  - `400 {"error": "group_id 格式错误"}`
  - `500 {"error": "查询关联模型失败"}`

> **使用场景：** 前端在删除用户组前调用此接口，如果 `count > 0`，则弹窗提示用户该组关联了哪些模型。

### `GET /admin/channels`

通道配置页面，展示所有通道（预定义通道和自定义通道）及其启用状态。

- **权限：** 管理员
- **JSON 响应：** `{"channels": [...]}`，每个通道对象包含以下字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint | 通道数据库 ID |
| ChannelID | string | 通道标识（唯一） |
| Name | string | 通道显示名称 |
| Enabled | bool | 是否启用 |
| Custom | bool | `true` 为自定义通道，`false` 为预定义通道 |
| CustomConfig | string | 自定义通道配置 JSON（预定义通道为空字符串） |
| agent_types | string[] | 支持该通道的 agent_type 列表（见下）。前端据此置灰不支持的 channel 卡片 |

> **`agent_types` 字段语义：**
> - **预定义通道**（`Custom=false`）：返回所有 `AgentTypeChannelAllowed(t, channel_id) == true` 的 agent_type；自定义类型按其 `compatible_with` 解析为兼容的内置运行时类型再查白名单，无 `compatible_with` 的自定义类型不会出现。顺序按 `GetAllAgentTypes()` 稳定返回（内置在前，自定义按创建顺序）。未被任何类型支持则返回空数组 `[]`。
> - **自定义通道**（`Custom=true`）：返回所有 `SupportsChannel=true` 的 agent_type，即所有内置类型 + 兼容内置类型的自定义类型；不兼容任何内置类型的自定义类型 `SupportsChannel=false`，不会出现。
> - 字段为 JSON 数组（非 `null`），即使为空也序列化为 `[]`。

```json
{
  "channels": [
    {
      "ID": 1, "ChannelID": "wecom", "Name": "企业微信",
      "Enabled": true, "Custom": false, "CustomConfig": "",
      "agent_types": ["openclaw", "lightclawace"]
    },
    {
      "ID": 5, "ChannelID": "hworktalk", "Name": "海尔工作通",
      "Enabled": true, "Custom": true,
      "CustomConfig": "{\"server\":{\"url\":\"wss://example.com/ws\"},\"cred_fields\":[{\"key\":\"token\",\"label\":\"接入Token\"}]}",
      "agent_types": ["openclaw", "hermes", "lightclawace"]
    }
  ]
}
```

### `POST /admin/channels/toggle`

切换通道启用/禁用状态。适用于预定义通道和自定义通道。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 通道 ID（Query 参数） |

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`404 {"error": "通道不存在"}`

### `POST /admin/channels/add`

添加自定义通道。仅支持 JSON 请求体。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channel_id | string | 是 | 通道标识，仅允许英文字母、数字和下划线（`^[a-zA-Z0-9_]+$`），不可与已有通道重复 |
| name | string | 是 | 通道显示名称 |
| custom_config | object | 是 | 自定义通道配置，包含 `server` 和 `cred_fields` 两个字段 |

`custom_config` 对象结构：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| server | object | 否 | IM 服务器配置，任意 JSON 对象，**同时作为 openclaw.json 中该通道配置的模板**（可含 `{{key}}` 占位符，key 对应 cred_fields）。可为空或 `{}`（运行时由 `DefaultsForChannel` 填充默认值）；如提供则必须是合法 JSON 对象。原样存储不做业务字段解析 |
| cred_fields | array | 否 | 用户凭证字段定义数组，每项包含 `key`（仅允许英文字母、数字和下划线）和 `label`（显示名称），key 不可重复 |

- **请求示例：**

```json
{
  "channel_id": "hworktalk",
  "name": "海尔工作通",
  "custom_config": {
    "server": {
      "url": "wss://example.com/ws",
      "protocol": "websocket"
    },
    "cred_fields": [
      { "key": "app_key", "label": "应用Key" },
      { "key": "app_secret", "label": "应用Secret" }
    ]
  }
}
```

- **JSON 响应：**
  - 成功：`{"ok": true, "channel": {...}}`（`channel` 为创建后的完整通道对象）
  - 失败：
    - `400 {"error": "请求格式错误"}`
    - `400 {"error": "Channel ID 不能为空"}`
    - `400 {"error": "Channel ID 仅允许英文字母、数字和下划线"}`
    - `400 {"error": "通道名称不能为空"}`
    - `400 {"error": "缺少自定义通道配置"}`
    - `400 {"error": "自定义通道配置格式错误"}`
    - `400 {"error": "IM 服务器配置必须为 JSON 对象"}`
    - `400 {"error": "凭证字段 key 仅允许英文字母、数字和下划线"}`
    - `400 {"error": "凭证字段 label 不能为空"}`
    - `400 {"error": "凭证字段 key 重复"}`
    - `409 {"error": "Channel ID 已存在"}`
    - `500 {"error": "创建通道失败"}`

### `POST /admin/channels/delete`

删除自定义通道。仅允许删除自定义通道（`Custom=true`），预定义通道不可删除。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 通道 ID（Query 参数） |

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：
    - `400 {"error": "缺少参数 id"}`
    - `404 {"error": "通道不存在"}`
    - `403 {"error": "预定义通道不允许删除"}`

### `GET /admin/usage/data`

统一用量查询 JSON API，支持灵活的聚合维度和筛选。

- **权限：** 管理员
- **Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 起始日期（`YYYY-MM-DD`），默认今天 |
| end_date | string | 否 | 终止日期（`YYYY-MM-DD`），默认今天 |
| group_by | string | 否 | 聚合维度，逗号分隔：`date`/`user`/`model`/`instance`/`department`/`group`。默认 `date,model` |
| user_id | uint | 否 | 筛选指定用户 |
| ai_model_id | uint | 否 | 筛选指定模型 |
| id | uint | 否 | 筛选指定实例（`instances.id` 主键）；优先级高于 `instance_id` |
| instance_id | string | 否 | 筛选指定实例（CVM ID 字符串，如 `ins-abc123`）；为兼容旧调用方，也接受纯数字（此时同 `id`） |
| department_id | string | 否 | 按部门 ID 筛选（兼容旧参数名 `department`） |
| group_id | uint | 否 | 按用户组筛选；`group_by=user` 时同时用于解析该组上下文下的用户 Token 配额规则；`group_by=group` 时筛选该组及所有后代组 |
| order_by | string | 否 | 排序字段：`total_tokens`（默认）或 `request_count` |
| order | string | 否 | `desc` 按 `order_by` 字段降序排列，默认不排序 |

- **响应顶层字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| start_date | string | 起始日期（`YYYY-MM-DD`） |
| end_date | string | 终止日期（`YYYY-MM-DD`） |
| group_by | string[] | 实际使用的聚合维度 |
| global_token_quota_day | int | 兼容旧字段：站点全局 Token 配额值（-1=不限）。实际周期由全局规则决定 |
| global_token_quota_rules | object[] | 站点全局 Token 配额规则数组 |
| global_token_quota_usages | object[] | 站点全局各规则当前周期内用量，元素为 `{rule_index, used, period_start, period_end, active}` |
| rows | object[] | 数据行，字段见下表 |

- **响应字段（每条 row）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| date | string | 日期（`YYYY-MM-DD`），`group_by` 含 `date` 时出现 |
| user_id | uint | 用户 ID，`group_by` 含 `user` 时出现 |
| user_name | string | 用户名，`group_by` 含 `user` 时出现 |
| user_email | string | 用户邮箱，`group_by` 含 `user` 时出现（`omitempty`） |
| ai_model_id | uint | 模型 ID，`group_by` 含 `model` 时出现 |
| model_name | string | 模型名称，`group_by` 含 `model` 时出现 |
| instance_id | uint | 实例 ID，`group_by` 含 `instance` 时出现 |
| instance_name | string | 实例名称，`group_by` 含 `instance` 时出现 |
| instance_cvm_id | string | CVM 实例 ID（如 `ins-abc123`），`group_by` 含 `instance` 时出现 |
| department_id | string | 部门 ID，`group_by` 含 `department` 时出现（`omitempty`） |
| department_name | string | 部门名称，`group_by` 含 `department` 时出现（`omitempty`） |
| department_path | string | 部门完整路径，`group_by` 含 `department` 时出现（`omitempty`） |
| group_id | uint | 分组 ID，`group_by=group` 时出现 |
| group_name | string | 分组名称，`group_by=group` 时出现 |
| group_full_path | string | 分组完整路径，`group_by=group` 时出现 |
| prompt_tokens | int64 | 输入 tokens |
| completion_tokens | int64 | 输出 tokens |
| total_tokens | int64 | 总 tokens |
| prompt_cache_read_tokens | int64 | 命中/读取 prompt cache 的 tokens |
| prompt_cache_write_tokens | int64 | 创建/写入 prompt cache 的 tokens |
| request_count | int64 | 请求次数 |
| token_quota_rules | object[] | 用户 Token 配额规则数组，`group_by` 含 `user` 时出现。传 `group_id` 时按 LLM proxy 的分组上下文解析；否则返回用户自身有效规则 |
| token_quota_usages | object[] | 用户各 Token 配额规则当前周期内用量，元素为 `{rule_index, used, period_start, period_end, active}`，`group_by` 含 `user` 时出现 |
| token_quota_groups | object[] | 用户所属各分组下的 Token 配额和当前周期用量，`group_by` 含 `user` 且未传 `group_id` 时出现；每项包含 `group_id`、`group_name`、`group_full_path`、`token_quota_rules`、`token_quota_usages` |
| global_token_quota_rules | object[] | 分组有效全局 Token 配额规则数组，`group_by=group` 时出现；分组有显式全局规则时返回分组规则，否则返回站点全局规则 |
| global_token_quota_usages | object[] | 分组有效全局规则当前周期内用量，元素为 `{rule_index, used, period_start, period_end, active}`，`group_by=group` 时出现；分组 row 始终按该分组统计 |

- **JSON 响应：**

```json
{
  "start_date": "2026-03-05",
  "end_date": "2026-03-12",
  "group_by": ["user", "model"],
  "global_token_quota_day": -1,
  "global_token_quota_rules": [
    {"mode": "day", "limit": 1000000}
  ],
  "global_token_quota_usages": [
    {"rule_index": 0, "used": 62000, "period_start": 1772640000, "period_end": 1772726400, "active": true}
  ],
  "rows": [
    {
      "user_id": 1,
      "user_name": "alice",
      "user_email": "alice@example.com",
      "ai_model_id": 2,
      "model_name": "openai/gpt-4",
      "prompt_tokens": 50000,
      "completion_tokens": 12000,
      "total_tokens": 62000,
      "prompt_cache_read_tokens": 30000,
      "prompt_cache_write_tokens": 2000,
      "request_count": 150,
      "token_quota_rules": [
        {"mode": "day", "limit": 100000}
      ],
      "token_quota_usages": [
        {"rule_index": 0, "used": 62000, "period_start": 1772640000, "period_end": 1772726400, "active": true}
      ],
      "token_quota_groups": [
        {
          "group_id": 7,
          "group_name": "研发一组",
          "group_full_path": "根组/研发组/研发一组",
          "token_quota_rules": [
            {"mode": "month", "limit": 3000000}
          ],
          "token_quota_usages": [
            {"rule_index": 0, "used": 62000, "period_start": 1772294400, "period_end": 1774972800, "active": true}
          ]
        }
      ]
    }
  ]
}
```

> 响应中仅包含 `group_by` 指定的维度字段，未指定的维度字段省略（`omitempty`）。
>
> `*_quota_usages[].rule_index` 对应同名 `*_quota_rules` 数组下标；`used` 为该规则当前生效窗口内的 `total_tokens` 汇总；`period_start` / `period_end` 为该规则当前生效窗口的 Unix 秒时间范围，消耗占比由前端计算。`active=false` 表示当前无生效窗口，兼容返回 `period_start=0`、`period_end=0`；无终止的生效窗口返回 `period_end=null`。

- **错误响应：**
  - `400 {"error": "order_by 参数无效，仅支持 total_tokens 或 request_count"}`
  - `500 {"error": "查询用量数据失败"}`

### `GET /admin/usage/logs`

分页查询 LLM 使用明细记录。

- **权限：** 管理员
- **Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 起始日期（`YYYY-MM-DD`），默认今天 |
| end_date | string | 否 | 终止日期（`YYYY-MM-DD`），默认今天 |
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页数量，默认 50，最大 200 |
| user_id | uint | 否 | 筛选指定用户 |
| ai_model_id | uint | 否 | 筛选指定模型 |
| id | uint | 否 | 筛选指定实例（`instances.id` 主键）；优先级高于 `instance_id` |
| instance_id | string | 否 | 筛选指定实例（CVM ID 字符串，如 `ins-abc123`）；为兼容旧调用方，也接受纯数字（此时同 `id`） |

- **响应字段（每条 log）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 记录 ID |
| user_name | string | 用户名 |
| provider | string | 模型提供商 |
| model | string | 模型名称 |
| prompt_tokens | int | 输入 tokens |
| completion_tokens | int | 输出 tokens |
| total_tokens | int | 总 tokens |
| prompt_cache_read_tokens | int | 命中/读取 prompt cache 的 tokens |
| prompt_cache_write_tokens | int | 创建/写入 prompt cache 的 tokens |
| status_code | int | HTTP 状态码 |
| latency | int | 耗时（毫秒） |
| created_at | string | 请求时间（RFC 3339，如 `2026-03-14T14:30:00Z`） |

- **JSON 响应：**

```json
{
  "start_date": "2026-03-12",
  "end_date": "2026-03-12",
  "page": 1,
  "page_size": 50,
  "total": 128,
  "logs": [
    {
      "id": 1,
      "user_name": "alice",
      "provider": "openai",
      "model": "gpt-4",
      "prompt_tokens": 500,
      "completion_tokens": 120,
      "total_tokens": 620,
      "prompt_cache_read_tokens": 200,
      "prompt_cache_write_tokens": 50,
      "status_code": 200,
      "latency": 1230,
      "created_at": "2026-03-12 10:30:00"
    }
  ]
}
```

---

## 四、技能管理（管理员）

所有技能管理接口始终返回 JSON，不受 Accept Header 影响。

> **前置条件：** 所有技能及技能分类接口均要求 SMH 服务已启用（`smh_enabled = 1`）。若未启用，返回 `403 {"error": "SMH 服务未启用，请先在管理后台开通 SMH 服务"}`。

### 4.1 技能分类管理

### `GET /admin/skill-categories`

查询技能分类列表（分页）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "categories": [
    {
      "id": 1,
      "name": "基础技能",
      "description": "系统内置基础技能",
      "created_at": "2026-03-26T10:00:00Z",
      "skill_count": 3
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

### `POST /admin/skill-categories/create`

创建技能分类。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 分类名称（唯一） |
| description | string | 否 | 分类描述 |

- **成功响应：** `{"ok": true, "id": 3}`
- **失败响应：**
  - `400 {"error": "分类名称不能为空"}`
  - `400 {"error": "分类名称已存在"}`

### `POST /admin/skill-categories/update`

更新技能分类。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 分类 ID |
| name | string | 否 | 新名称 |
| description | string | 否 | 新描述 |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "缺少分类 ID"}`
  - `400 {"error": "分类名称已存在"}`
  - `404 {"error": "分类不存在"}`

### `POST /admin/skill-categories/delete`

删除技能分类。删除时会自动清理该分类与技能的关联关系，技能本身不受影响。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 分类 ID |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `404 {"error": "分类不存在"}`

### 4.2 技能 CRUD

### `GET /admin/skills`

查询技能列表。每个 slug 只返回最新版本（按语义化版本号排序）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 否 | 按名称、描述或 slug 模糊搜索（同时匹配 name、description、slug，满足其一即命中） |
| name | string | 否 | 按名称模糊搜索 |
| slug | string | 否 | 按 slug 精确搜索（唯一字段，精确匹配） |
| description | string | 否 | 按描述模糊搜索 |
| category_ids | string | 否 | 按分类 ID 筛选，多个用逗号分隔，如 `1,3` |
| visibility_type | string | 否 | 按可见范围类型筛选：`all` 或 `group`。单独传时仅返回该类型的技能 |
| group_id | string | 否 | 按分组 ID 筛选，多个用逗号分隔如 `1,3`。单独传时仅返回关联了这些分组的技能；同时传 `visibility_type=all` 时返回全局可见 + 匹配分组的技能 |
| project_id | string | 否 | 按项目 ID 筛选，多个用逗号分隔如 `1,3`。返回应用范围关联了任一项目的技能；与 `group_id` 同时传时取两者并集 |
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "skills": [
    {
      "id": 5,
      "slug": "skill-a",
      "name": "技能A",
      "version": "2.0.0",
      "description": "这是一个基础技能",
      "file_size": 102400,
      "visibility_type": "group",
      "created_at": "2026-03-26T10:00:00Z",
      "categories": [
        {"id": 1, "name": "基础技能"}
      ],
      "visibility_groups": [
        {"group_id": 1, "group_name": "研发组"},
        {"group_id": 3, "group_name": "测试组"}
      ],
      "visibility_projects": [
        {"id": 2, "name": "智能助手项目"}
      ],
      "last_task": {
        "task_id": 12,
        "status": "completed",
        "total": 10,
        "success": 9,
        "failed": 1,
        "version": "2.0.0",
        "created_at": "2026-03-28T15:30:00Z"
      },
      "security_scan": {
        "scan_status": "safe",
        "risk_level": "benign",
        "security_score": 100,
        "scanned_at": "2026-04-02T09:13:23+08:00",
        "report_url": "https://..."
      }
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

> `last_task` 为该技能最近一次下发任务的执行状态，未下发过时为 `null`。`status` 取值：`running`（进行中）、`completed`（已完成）。

> `security_scan` 为该技能最新安全检测状态，未检测时为 `null`。`scan_status` 取值见文末枚举说明。

### `POST /admin/skills/create`

创建技能（上传 zip 压缩包）。服务端会校验 zip 中的所有文件名均为 UTF-8，自动注入 `_meta.json` 到压缩包中，然后上传到 SMH 存储。非 UTF-8 文件名无法可靠判断原始编码，因此不会自动转码，而是返回明确的 400 错误，提示重新打包。

支持两种 zip 包结构：

- **规范结构（推荐）：** `{slug}/SKILL.md`、`{slug}/src/...` — 包含一个以 slug 命名的顶级目录
- **简化结构：** `SKILL.md`、`src/...` — 根目录直接放文件，服务端自动转换为规范结构

- **权限：** 管理员
- **Content-Type：** `multipart/form-data`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | binary | 是 | zip 压缩包（最大 100MB） |
| slug | string | 是 | 唯一标识。格式：小写字母、数字和连字符，长度 3-50，不能以连字符开头或结尾 |
| name | string | 是 | 显示名称 |
| version | string | 是 | 版本号（语义化版本，如 `1.0.0`） |
| description | string | 否 | 技能描述 |
| category_ids | string | 否 | 分类 ID 列表，多个用逗号分隔，如 `1,3` |
| visibility_type | string | 否 | 应用范围：`all`（全部用户，默认）或 `group`（按分组）。不传则继承同 slug 旧版本的设置；传入范围时必传 |
| group_ids | string | 条件 | 分组 ID 列表，多个用逗号分隔，如 `1,3`。传入范围时必须同时传该字段；空字符串表示清空全部已有分组绑定 |
| project_ids | string | 条件 | 项目 ID 列表，多个用逗号分隔。传入范围时必须同时传该字段；空字符串表示清空全部已有项目绑定 |
| changelog | string | 否 | 版本更新说明 |
| submit_scan | string | 否 | `"true"` 提交安全检测，不传不触发。文件超过 7MB 时即使传 true 也不会触发 |

**Zip 包校验规则：**

1. 必须包含 `SKILL.md` 文件（不区分大小写）
2. 不允许包含 `..` 路径（防止 zip slip）
3. 解压后总大小不超过 200MB
4. zip 中所有文件名必须是合法 UTF-8；服务端不猜测或转换 GBK、GB18030、Big5、Shift-JIS 等本地编码

**事务保证：** 先写入数据库，再上传 SMH。SMH 上传失败时自动回滚数据库并清理已上传的文件，确保不会出现 DB 和 SMH 不一致的情况。

> **重新上传：** 如果同 slug+version 的记录已被删除（软删除），再次上传会自动物理删除旧记录并重新创建，无需手动干预。

- **成功响应：** `{"ok": true, "id": 6, "slug": "my-skill", "version": "1.0.0", "scan_submitted": true}`

> **版本递增校验：** 新版本号必须大于该 slug 的现有最高版本号。首次发布不受限制。

> **安全检测字段：** 传入 `submit_scan=true` 时响应额外包含 `scan_submitted`（bool）和 `scan_skip_reason`（string，仅 scan_submitted=false 时有值）。跳过原因如：`"file_too_large"`、`"limit_exceeded"` 等。

- **失败响应：**
  - `400 {"error": "slug、name、version 为必填字段"}`
  - `400 {"error": "slug 格式不合法，只允许小写字母、数字和连字符，长度 3-50，不能以连字符开头或结尾"}`
  - `400 {"error": "新版本号 1.0.0 必须大于现有最高版本 2.0.0"}`
  - `400 {"error": "该技能版本已存在（slug=my-skill, version=1.0.0），请修改后重试"}`
  - `400 {"error": "不存在 SKILL.md 文件，请修改后重试"}`
  - `400 {"error": "zip 中存在非 UTF-8 编码的文件名，请将文件名统一转换为 UTF-8 后重新打包上传"}`
  - `500 {"error": "存储服务不可用: ..."}`
  - `500 {"error": "上传 zip 到 COS 失败: ..."}`
  - `500 {"error": "上传文件到 COS 失败: ..."}`

### `POST /admin/skills/update`

更新技能元信息（名称、描述、分类），不涉及文件变更。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能唯一标识 |
| version | string | 是 | 版本号，定位要更新的记录 |
| name | string | 否 | 新名称 |
| description | string | 否 | 新描述 |
| category_ids | string | 否 | 新分类 ID 列表，多个用逗号分隔。传空字符串清空分类；不传则不修改 |
| visibility_type | string | 否 | 应用范围：`all` 或 `group`。不传则不修改；传入范围时必传 |
| group_ids | string | 条件 | 分组 ID 列表，多个用逗号分隔。更新范围时必须同时传该字段；空字符串清空已有分组绑定 |
| project_ids | string | 条件 | 项目 ID 列表，多个用逗号分隔。更新范围时必须同时传该字段；空字符串清空已有项目绑定 |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "slug 和 version 为必填字段"}`
  - `404 {"error": "技能不存在"}`

### `POST /admin/skills/delete`

删除技能。支持删除指定版本或所有版本。可选级联删除技能包和角色中的引用。有进行中的下发任务时禁止删除。同时清理 SMH 存储中对应的文件。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能唯一标识 |
| version | string | 否 | 要删除的版本号。不传则删除该 slug 的所有版本 |
| cascade | string | 否 | 传 `true` 时，级联删除技能包（BundleSkill）和角色（OpenClawRoleSkill）中引用该技能的记录，并清理 SMH 文件 |

- **成功响应：** `{"ok": true, "deleted_skills": 3, "cascade_deleted": {"bundle_skills": 2, "role_skills": 1}}`
- **失败响应：**
  - `400 {"error": "该版本有进行中的下发任务，无法删除"}`
  - `404 {"error": "技能不存在"}`

> **级联删除说明：** 不传 cascade 参数时，只删除 Skill 本身，技能包和角色中的悬挂引用由调用方自行处理。建议先调用 `GET /admin/skills/references` 查看影响范围，确认后再传入 cascade 参数。

### `GET /admin/skills/references`

查询技能在技能包和角色中的关联引用。用于删除前的影响评估。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能唯一标识 |
| version | string | 否 | 版本号。不传则查询所有版本的引用 |

- **成功响应：**

```json
{
    "slug": "my-skill",
    "references": {
        "bundle_skills": [
            {"id": 1, "skill_bundle_id": 1, "bundle_name": "通用技能包", "version": "1.0.0"}
        ],
        "role_skills": [
            {"id": 1, "openclaw_role_id": 2, "role_name": "客服助手", "version": "1.0.0"}
        ]
    }
}
```

> `references.bundle_skills` 和 `references.role_skills` 分别返回技能包和角色中的引用记录。建议删除前先调用此接口评估影响范围。

### `GET /admin/skills/detail`

查询技能详情。默认返回最新版本，可指定历史版本。同时返回该 slug 的所有可用版本列表。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能唯一标识 |
| version | string | 否 | 版本号。`latest` 或不传返回最新版本；传具体版本号返回指定版本 |

- **成功响应：**

```json
{
  "skill": {
    "id": 5,
    "slug": "my-skill",
    "name": "我的技能",
    "version": "2.0.0",
    "description": "技能描述",
    "categories": [{"id": 1, "name": "基础技能"}],
    "visibility_type": "group",
    "visibility_groups": [{"group_id": 1, "group_name": "研发组"}],
    "file_size": 102400,
    "cos_zip_key": "my-skill/my-skill-2.0.0.zip",
    "cos_dir_key": "my-skill/my-skill-2.0.0/",
    "created_at": "2026-03-26T10:00:00Z",
    "updated_at": "2026-03-26T11:00:00Z"
  },
  "versions": ["2.0.0", "1.0.0"],
  "security_scan": {
    "scan_status": "malicious",
    "scan_id": 456,
    "risk_level": "malicious",
    "security_score": 0,
    "report_url": "https://...",
    "scanned_at": "2026-04-02T09:13:23+08:00",
    "risk_description": "该 Skill 存在命令注入、凭证窃取与数据外传等多项高危行为",
    "mitigation": "建议立即停止使用该 Skill...",
    "scan_items": [...],
    "capability_tags": [{"ID": "file_read", "Name": "文件读取"}]
  }
}
```

> `security_scan` 为完整版安全检测详情，未检测时为 `null`。包含 `scan_items`（检测引擎明细）和 `capability_tags`（能力标签）等详细信息。

- **失败响应：**
  - `400 {"error": "缺少 slug 参数"}`
  - `404 {"error": "技能不存在"}`
  - `404 {"error": "版本不存在（slug=skill-a, version=3.0.0）"}`

### `GET /admin/skills/files`

查询技能文件列表（所有版本的文件目录结构）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能唯一标识 |

- **成功响应：**

```json
{
  "slug": "skill-a",
  "versions": [
    {
      "version": "2.0.0",
      "files": ["skill-a/skill-a-2.0.0/src/index.js", "skill-a/skill-a-2.0.0/SKILL.md", "skill-a/skill-a-2.0.0/_meta.json"],
      "security_scan": {
        "scan_status": "safe",
        "security_score": 100,
        "scanned_at": "2026-04-02T09:13:23+08:00"
      }
    },
    {
      "version": "1.0.0",
      "files": ["skill-a/skill-a-1.0.0/src/index.js", "skill-a/skill-a-1.0.0/SKILL.md", "skill-a/skill-a-1.0.0/_meta.json"],
      "security_scan": null
    }
  ]
}
```

> `security_scan` 仅最新版本（第一个元素）有值，其余为 `null`。

- **失败响应：**
  - `404 {"error": "技能不存在"}`

### 4.3 技能下发与安装

### `GET /admin/skills/tasks`

查询技能下发/卸载任务列表。默认按企业技能 `slug` 查询；公共技能可传 `source=public`，也可按 `source_skillset_slug` 查询公共技能包记录。返回按时间倒序，可用于轮询进度。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| source | string | 否 | 技能来源：`enterprise`（默认）或 `public` |
| slug | string | 条件必填 | 技能唯一标识；传 `source_skillset_slug` 或 `batch_id` 时可不传 |
| source_skillset_slug | string | 否 | 公共技能包 slug，仅 `source=public` 时有效 |
| batch_id | string | 否 | 批量请求 ID |
| type | string | 否 | 任务类型：`distribute`、`uninstall`、`all` |
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "tasks": [
    {
      "id": 1,
      "created_at": "2026-03-26T10:00:00Z",
      "operator": "admin",
      "version": "2.0.0",
      "source": "enterprise",
      "slug": "my-skill",
      "source_skillset_slug": "",
      "batch_id": "",
      "total": 3,
      "success": 2,
      "failed": 1,
      "pending": 0,
      "status": "completed",
      "type": "distribute",
      "records": [
        {
          "instance_id": 1,
          "cvm_instance_id": "ins-xxx",
          "instance_name": "user1的实例",
          "username": "user1",
          "status": "success",
          "error": ""
        }
      ]
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

> 按 `source_skillset_slug` 查询公共技能包记录时，`tasks[]` 会按批量请求聚合，返回 `task_ids`；`records[].skill_statuses` 返回该实例下各技能的状态。

- **状态枚举值：**

  - **Task 级别 `status`：** `running`（任务进行中）、`completed`（所有实例处理完毕）。
  - **Task 级别 `type`：** `distribute`（下发/更新）、`uninstall`（卸载）、`all`（聚合行包含多种任务类型）。
  - **Record 级别 `status`：** `pending`、`success`、`failed`、`upgrade_failed`、`uninstall_failed_old`。

- **失败响应：**
  - `400 {"error": "缺少 slug 参数"}`
  - `400 {"error": "不支持的来源类型: xxx"}`
  - `404 {"error": "技能不存在"}`

### `GET /admin/skills/instances`

查询实例安装情况。支持企业技能和公共技能，仅返回实例语义状态为 `running` 的实例（通过实时 CVM API 判定），支持按状态、实例类型、用户组等条件筛选。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| source | string | 否 | 技能来源：`enterprise`（默认）或 `public` |
| slug | string | 是 | 技能唯一标识 |
| version | string | 否 | 目标版本。企业技能不传或传 `latest` 时使用最新版本；公共技能不传或传 `latest` 时使用默认版本 |
| status | string | 否 | 按安装状态筛选：`installed`、`outdated`、`uninstalled`、`installing`、`failed`、`upgrade_failed`、`uninstalling`、`uninstall_failed`、`uninstall_failed_old`。支持逗号分隔多状态，如 `uninstalled,failed` |
| search | string | 否 | 按实例名称或 CVM 实例 ID 模糊搜索 |
| instance_type | string | 否 | 按实例类型筛选，如 `openclaw`、`hermes`、`lightclawace`。支持逗号分隔多类型 |
| group_id | string | 否 | 按用户组筛选实例。支持逗号分隔多个 group_id；传 `0` 表示未分组用户的实例，可与正常 group_id 组合使用 |
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "instances": [
    {
      "instance_id": 1,
      "cvm_instance_id": "ins-xxx",
      "instance_name": "user1的实例",
      "instance_type": "openclaw",
      "user_id": 1,
      "username": "user1",
      "last_cvm_state": "RUNNING",
      "status": "installed",
      "version": "2.0.0",
      "latest_version": "2.0.0",
      "user_groups": [
        {"group_id": 1, "group_name": "研发组"},
        {"group_id": 3, "group_name": "测试组"}
      ],
      "instance_status": "running",
      "instance_status_label": "运行中",
      "transient": false
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

### `POST /admin/skills/instances`

| 枚举值 | 说明 |
|--------|------|
| openclaw | OpenClaw，功能最完整的智能体类型（CVM 实例） |
| hermes | Hermes，轻量级智能体（CVM 实例） |
| lightclawace | LightclawACE，轻量级智能体（CVM 实例） |
| workbuddy 🆕 | WorkBuddy 本地 agent（local 实例，user PC 上） |
| codebuddy 🆕 | CodeBuddy 本地 agent（local 实例，user PC 上） |
| （空字符串） | 存量实例，未设置类型 |
批量查询公共技能包在实例上的安装情况。

- **权限：** 管理员
- **Content-Type：** `application/json`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| source_skillset_slug | string | 是 | 公共技能包 slug |
| skills | array | 是 | 公共技能列表，1～50 项 |
| skills[].slug | string | 是 | 公共技能 slug |
| skills[].version | string | 否 | 版本，`latest` 或不传使用默认版本 |
| status | string | 否 | 按安装状态筛选 |
| search | string | 否 | 按实例名称或 CVM 实例 ID 模糊搜索 |
| instance_type | string | 否 | 按实例类型筛选 |
| group_id | string | 否 | 按用户组筛选，语义同 GET |

分页通过 URL query 参数 `page` / `page_size` 传入。

| 枚举值 | 说明 | 判断逻辑（CVM） | 判断逻辑（local 🆕） |
|--------|------|---------|---------|
| installed | 已安装（最新版） | `skill_distribution_records.status='success'` 且已安装版本 = 技能最新版本 | `local_instance_skills.install_status='success'` 且 `version` = 技能最新版本 |
| outdated | 待更新 | `skill_distribution_records.status='success'` 但已安装版本 < 技能最新版本 | `local_instance_skills.install_status='success'` 但 `version` < 技能最新版本 |
| installing | 下发中 | `skill_distribution_records.status='pending'` | `local_instance_skills.install_status='pending'` |
| failed | 下发失败 | `skill_distribution_records.status='failed'` | `local_instance_skills.install_status='failed'` |
| uninstalled | 未下发 | 没有下发记录 | `local_instance_skills` 中没有该 (instance, slug) 的记录 |

> `upgrade_failed`/`uninstall_failed`/`uninstall_failed_old`/`uninstalling` 等状态仅 CVM 实例会出现，本地实例不会。
- **成功响应：** 结构与 `GET /admin/skills/instances` 一致，额外包含 `skill_statuses`（每条实例下各技能的安装状态列表）。

- **失败响应：**
  - `400 {"error": "参数 source_skillset_slug 不能为空"}`
  - `400 {"error": "参数 skills 不能为空"}`


### `POST /admin/skills/distribute`

批量下发技能到实例。支持企业技能和公共技能；可通过 `instance_ids` 显式指定实例，或通过 `select_all=true` 按目标版本的安装状态和用户组选择全部匹配实例。传 `skills[]` 时可一次提交多个技能，每个技能独立解析目标实例。异步执行，接口同步返回 task_id 或 batch_id。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "slug": "my-skill",
  "version": "latest",
  "instance_ids": [1, 2, 3]
}
```

按状态和用户组全选：

```json
{
  "slug": "my-skill",
  "version": "2.0.0",
  "select_all": true,
  "statuses": ["uninstalled", "failed"],
  "group_ids": [12, 0],
  "search": "alice"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| source | string | 否 | 技能来源：`enterprise`（默认）或 `public` |
| slug | string | 是 | 技能唯一标识 |
| version | string | 否 | 版本号。`latest` 或不传使用默认版本；传具体版本号下发指定版本 |
| source_skillset_slug | string | 否 | 公共技能包 slug，仅 `source=public` 时有效 |
| instance_ids | int[] | 条件必填 | 显式目标实例 ID；与 `select_all=true` 严格二选一 |
| select_all | bool | 条件必填 | 传 `true` 时由后端选择全部匹配实例；全选模式不限制目标数量 |
| statuses | string[] | 否 | 仅全选模式。可选 `uninstalled/installed/outdated/failed/upgrade_failed/uninstall_failed/uninstall_failed_old`；相对于目标版本计算；空数组/省略表示以上稳定状态全集；禁止 `installing/uninstalling` |
| group_ids | uint[] | 否 | 仅全选模式。多个组取并集；`0` 表示未分组用户；空数组/省略表示不限制用户组 |
| search | string | 否 | 仅全选模式。模糊匹配实例名称、实例 ID 或创建人用户名；最长 50 个字符；省略表示不限制 |
| skills | array | 否 | 批量技能列表，1～50 项；传入时不传顶层 `source/slug/version/source_skillset_slug` |
| skills[].source | string | 否 | `enterprise` 或 `public`，不传默认 `enterprise` |
| skills[].slug | string | 是 | 技能唯一标识 |
| skills[].version | string | 否 | 版本语义同顶层 `version` |
| skills[].source_skillset_slug | string | 否 | 公共技能包 slug，仅公共技能可传 |

- **成功响应：**
  - 单技能：`{"ok": true, "task_id": 1, "version": "2.0.0", "total": 3}`
  - 批量：`{"ok": true, "batch_id": "skilldist-...", "task_ids": [1, 2], "total": 2, "submitted": 2, "failed": 0, "results": [{"status": "submitted", "instance_count": 3}]}`

> 全选目标在请求受理时固化为 task records；不要求实例处于 running，但仍按 Agent 技能能力过滤。筛选后无目标返回 400，且不创建空任务。

- **失败响应：**
  - `400 {"error": "slug 不能为空"}`
  - `400 {"error": "instance_ids 不能为空"}`
  - `400 {"error": "skills 数量不能超过 50"}`
  - `400 {"error": "instance_ids 与 select_all=true 必须二选一"}`
  - `400 {"error": "不能按过渡状态 installing 全选下发"}`
  - `400 {"error": "技能不存在"}`
  - `400 {"error": "版本不存在（slug=my-skill, version=3.0.0）"}`

> 下发进度可通过 `GET /admin/skills/tasks?slug=X` 轮询；公共技能可传 `source=public`，公共技能包可传 `source_skillset_slug`。

### `POST /admin/skills/uninstall`

批量卸载技能（从实例上移除已安装的技能）。支持企业技能和公共技能；可通过 `instance_ids` 显式指定实例，或通过 `select_all=true` 按安装状态和用户组选择全部匹配实例。传 `skills[]` 时可一次提交多个技能。异步执行，接口同步返回 task_id 或 batch_id。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "slug": "my-skill",
  "select_all": true,
  "statuses": ["installed", "uninstall_failed"],
  "group_ids": [12, 0],
  "search": "alice"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| source | string | 否 | 技能来源：`enterprise`（默认）或 `public` |
| slug | string | 是 | 技能唯一标识 |
| source_skillset_slug | string | 否 | 公共技能包 slug，仅 `source=public` 时有效 |
| instance_ids | int[] | 条件必填 | 显式目标实例 ID 列表；与 `select_all=true` 严格二选一 |
| select_all | bool | 条件必填 | 传 `true` 时选择全部匹配实例；全选模式不限制目标数量 |
| statuses | string[] | 否 | 仅全选模式。可选 `installed/outdated/upgrade_failed/uninstall_failed/uninstall_failed_old`；空数组/省略表示以上状态全集 |
| group_ids | uint[] | 否 | 仅全选模式。多个组取并集；`0` 表示未分组用户；空数组/省略表示不限制用户组 |
| search | string | 否 | 仅全选模式。模糊匹配实例名称、实例 ID 或创建人用户名；最长 50 个字符；省略表示不限制 |
| skills | array | 否 | 批量技能列表，1～50 项；传入时不传顶层 `source/slug/source_skillset_slug` |
| skills[].source | string | 否 | `enterprise` 或 `public`，不传默认 `enterprise` |
| skills[].slug | string | 是 | 技能唯一标识 |
| skills[].source_skillset_slug | string | 否 | 公共技能包 slug，仅公共技能可传 |

- **成功响应：**
  - 单技能显式 ID：`{"ok": true, "task_id": 1}`
  - 单技能全选：`{"ok": true, "task_id": 1, "total": 1200}`
  - 批量：`{"ok": true, "batch_id": "skilldist-...", "task_ids": [1, 2], "total": 2, "submitted": 2, "failed": 0, "results": []}`

- **失败响应：**
  - `400 {"error": "slug 不能为空"}`
  - `400 {"error": "instance_ids 不能为空"}`
  - `400 {"error": "instance_ids 与 select_all=true 必须二选一"}`
  - `400 {"error": "状态 uninstalled 不支持全选操作"}`
  - `400 {"error": "skills 数量不能超过 50"}`
  - `400 {"error": "没有符合条件的实例，所选实例类型不支持技能"}`
  - `404 {"error": "技能不存在"}`
  - `409 {"error": "该技能正在被其他操作处理，请稍后重试"}`

> 卸载进度可通过 `GET /admin/skills/tasks?slug=X&type=uninstall` 轮询；公共技能可传 `source=public`，公共技能包可传 `source_skillset_slug`。

#### 实例安装状态矩阵

`GET /admin/skills/instances` 和 `GET /openclaw/skillstore/instances` 返回的 `status` 字段可能的值及其在下发/卸载弹窗中的可见性：

| status 值 | 含义 | 下发弹窗可选？ | 卸载弹窗可选？ |
|-----------|------|:-:|:-:|
| uninstalled | 从未安装 / 已成功卸载 | ✅ | ❌ |
| installed | 已安装最新版 | ❌ | ✅ |
| outdated | 已安装但版本旧 | ✅（升级） | ✅（可卸载） |
| installing | 正在下发中 | ❌ | ❌ |
| failed | 首次安装失败（实例上无技能） | ✅（重试） | ❌ |
| upgrade_failed | 升级失败（旧版本仍在实例上） | ✅（重试升级） | ✅（卸载旧版本） |
| uninstalling | 正在卸载中 | ❌ | ❌ |
| uninstall_failed | 卸载最新版失败（最新版仍在） | ❌ | ✅（重试卸载） |
| uninstall_failed_old | 卸载旧版本失败（旧版本仍在） | ✅（升级覆盖） | ✅（重试卸载） |

**前端筛选参数：**

下发弹窗：

| 筛选项 | status 参数 |
|--------|-------------|
| 未下发 | `uninstalled` |
| 待更新 | `outdated,uninstall_failed_old` |
| 下发失败 | `failed,upgrade_failed` |

卸载弹窗：

| 筛选项 | status 参数 |
|--------|-------------|
| 未卸载 | `installed,outdated,upgrade_failed` |
| 卸载失败 | `uninstall_failed,uninstall_failed_old` |

### 4.4 SMH 存储信息

### `GET /admin/smh/config`

查询 SMH 存储配置信息（API 接口，始终返回 JSON）。

- **权限：** 管理员
- **参数：** 无
- **响应：** 始终 JSON

```json
{
  "smh_enabled": true,
  "smh_library_id": "lib-xxx",
  "smh_library_secret": "abcd****efgh",
  "smh_endpoint": "https://smh.example.com",
  "smh_common_space": "space-common-xxx",
  "smh_skillhub_space": "space-skillhub-xxx",
  "smh_auto_provision_on_create": false,
  "is_configured": true,
  "provision_error": ""
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| smh_enabled | bool | SMH 服务是否启用 |
| smh_library_id | string | SMH Library ID |
| smh_library_secret | string | SMH Library Secret（脱敏显示，仅前4位和后4位） |
| smh_endpoint | string | SMH 访问域名 |
| smh_common_space | string | 公共空间 ID |
| smh_skillhub_space | string | 技能市场空间 ID |
| smh_auto_provision_on_create | bool | 创建实例时是否自动创建个人空间 |
| is_configured | bool | 配置是否完整 |
| provision_error | string | 最近一次自动开通的错误信息，无错误时为空 |

### `POST /admin/config/smh`

更新 SMH 存储配置。仅覆盖请求中包含的参数，未传递的参数保持不变。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| smh_library_id | string | 否 | SMH Library ID |
| smh_library_secret | string | 否 | SMH Library Secret（敏感字段） |
| smh_endpoint | string | 否 | SMH 访问域名 |
| smh_enabled | string | 否 | `"0"` 关闭 / `"1"` 开启 SMH 服务 |
| smh_auto_provision_on_create | string | 否 | `"0"` 关闭 / `"1"` 开启创建实例时自动创建个人空间 |

- **响应：** 始终 JSON
  - 成功：`{"ok": true}`
  - 失败：`500 {"error": "update SiteConfig: ..."}`

### `GET /admin/smhinfo`

查询 SMH 存储配置信息、开通状态和各 Space 的 AccessToken。前端进入企业技能库页面时应先调用此接口判断 SMH 状态。每个 Space 维护独立的 Token（创建时绑定 SpaceId），不可互用。返回的 `access_token` 为只读（`download_only`）权限，仅可用于下载，不可用于上传或删除。

- **权限：** 管理员
- **响应：** 始终返回 JSON
- **成功响应（已开通且配置完整）：**

```json
{
  "smh_enabled": 1,
  "provision_error": "",
  "endpoint": "https://smh.example.com",
  "library_id": "lib-xxx",
  "auto_provision_on_create": false,
  "common_space": {
    "space_id": "space-common-xxx",
    "access_token": "token-for-common"
  },
  "skillhub_space": {
    "space_id": "space-skills-xxx",
    "access_token": "token-for-skillhub"
  }
}
```

- **未开通或配置不完整时的响应：**

```json
{
  "smh_enabled": 0,
  "provision_error": "INSUFFICIENT_BALANCE",
  "endpoint": "",
  "library_id": "",
  "auto_provision_on_create": false
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| smh_enabled | int | SMH 服务开通状态：`0`=未开通，`1`=已开通 |
| provision_error | string | SMH 开通失败时的错误码，开通成功后为空字符串。完整错误码清单见下方表格 |
| endpoint | string | SMH 媒体库访问域名 |
| library_id | string | SMH 媒体库 ID |
| auto_provision_on_create | bool | 创建实例时是否自动创建个人空间 |
| common_space | object | 公共空间信息（仅已开通且配置完整时返回） |
| skillhub_space | object | 技能空间信息（仅已开通且配置完整时返回） |

> 如果某个 Space 的 Token 获取失败，对应对象中会包含 `"error"` 字段而非 `"access_token"`。`access_token` 为 `download_only` 只读 Token，写操作（上传/删除）由后端使用独立的 `space_admin` Token 完成。

> **`provision_error` 错误码清单：**
>
> | 错误码 | 说明 | 前端建议提示 |
> |--------|------|-------------|
> | `INSUFFICIENT_BALANCE` | 账户余额不足 | 账户余额不足，无法开通 SMH 服务，请充值后等待自动重试 |
> | `STS_ROLE_NOT_FOUND` | STS 角色不存在，凭证获取失败 | STS 角色不存在，请检查 CAM 角色配置后等待自动重试 |
> | `PROVISION_IN_PROGRESS` | 另一个实例正在执行开通 | SMH 服务正在由其他节点开通中，请稍候... |
> | `CREATE_LIBRARY_FAILED` | 创建媒体库失败 | SMH 媒体库创建失败，后台将自动重试 |
> | `UPDATE_LIBRARY_FAILED` | 更新媒体库配置失败 | SMH 媒体库配置更新失败，后台将自动重试 |
> | `DESCRIBE_SECRET_FAILED` | 获取媒体库密钥失败 | SMH 媒体库密钥获取失败，后台将自动重试 |
> | `CREATE_SPACE_FAILED` | 创建空间失败 | SMH 空间创建失败，后台将自动重试 |
> | `INTERNAL_ERROR` | 内部错误（含网络错误、未知错误等） | 服务内部异常，后台将自动重试，如持续失败请联系管理员 |

> **前端判断逻辑：**
> - `smh_enabled == 0 && provision_error != ""` → 根据错误码展示对应提示（后台每五分钟自动重试）
> - `smh_enabled == 0 && provision_error == ""` → 展示「SMH 服务正在开通中，请稍候...」
> - `smh_enabled == 1` → 正常展示技能列表

### `GET /admin/smh/personal-spaces`

查询实例个人空间列表（分页）。

- **权限：** 管理员
- **响应：** 始终返回 JSON
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|---------|
| page | int | 否 | 页码（默认 1） |
| page_size | int | 否 | 每页条数（默认 20） |
| instance_deleted_only | string | 否 | 传 `1` 时仅返回实例已删除（回收站）的个人空间 |
| instance_not_deleted_only | string | 否 | 传 `1` 时仅返回实例未删除（激活）的个人空间 |
| user | int | 否 | 按用户 ID 过滤 |

- **成功响应：**

```json
{
  "items": [
    {
      "id": 1,
      "space_id": "spacexxxxxxxx",
      "user_id": 1,
      "username": "alice",
      "instance_id": 42,
      "instance_name": "my-instance",
      "cvm_instance_id": "ins-xxxxxxxx",
      "storage_quota": 53687091200,
      "free_storage_quota": 53687091200,
      "used_storage": 1073741824,
      "bound_at": "2026-04-01T00:00:00Z",
      "expires_at": "2026-07-01T00:00:00Z",
      "instance_deleted": false,
      "to_be_deleted_at": null
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

> - `storage_quota` / `free_storage_quota` / `used_storage`：单位为字节。`used_storage` 为实时查询 SMH API 获取。
> - `expires_at`：个人空间过期时间（ISO 8601 格式），当前默认为绑定后 3 个月，后续可能根据策略调整。
> - `to_be_deleted_at`：仅在 `instance_deleted` 为 `true`（实例在回收站中）时有值，表示空间计划删除的时间。

### `GET /admin/smh/personal-spaces/token`

获取指定个人空间的临时访问 Token。

- **权限：** 管理员
- **响应：** 始终返回 JSON
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 个人空间数据库 ID（Query 参数） |

- **成功响应：**

```json
{
  "token": "...",
  "space_id": "...",
  "library_id": "...",
  "endpoint": "...",
  "expires_at": 1743562200000
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| token | string | 临时访问 Token |
| space_id | string | 个人空间 ID |
| library_id | string | SMH 媒体库 ID |
| endpoint | string | SMH 访问域名 |
| expires_at | int | Token 过期时间（毫秒时间戳） |

- **失败响应：**
  - `404 {"error": "个人空间不存在"}`

### `GET /admin/smh/stat`

查询 SMH 存储的整体统计信息。

- **权限：** 管理员
- **参数：** 无
- **响应：** 始终返回 JSON
- **成功响应：**

```json
{
  "space_count": 130,
  "used_storage": 56.3,
  "storage_quota": 1024.0,
  "public_space": {
    "used_storage": 6.3,
    "storage_quota": 100.0
  },
  "common_space": {
    "used_storage": 3.2,
    "storage_quota": 50.0
  },
  "skillhub_space": {
    "used_storage": 3.1,
    "storage_quota": 50.0
  },
  "personal_space": {
    "count": 128,
    "used_storage": 50.0,
    "storage_quota": 924.0
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| space_count | int | 空间总数（含公共空间和个人空间） |
| used_storage | float | 总已使用存储量（单位：GB） |
| storage_quota | float | 总存储配额（单位：GB） |
| public_space.used_storage | float | 公共空间已使用存储量（common + skillhub，单位：GB） |
| public_space.storage_quota | float | 公共空间存储配额（common + skillhub，单位：GB） |
| common_space.used_storage | float | Common 空间已使用存储量（单位：GB） |
| common_space.storage_quota | float | Common 空间存储配额（单位：GB） |
| skillhub_space.used_storage | float | Skillhub 空间已使用存储量（单位：GB） |
| skillhub_space.storage_quota | float | Skillhub 空间存储配额（单位：GB） |
| personal_space.count | int | 个人空间数量 |
| personal_space.used_storage | float | 个人空间已使用存储量（单位：GB） |
| personal_space.storage_quota | float | 个人空间存储配额（单位：GB） |

### `POST /admin/smh/personal-space-auto-provision`

设置创建实例时是否自动创建个人空间。

- **权限：** 管理员
- **响应：** 始终返回 JSON
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| enabled | string | 是 | `1` 开启，`0` 关闭 |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "参数 enabled 必须为 0 或 1"}`

### `GET /admin/smh/instances`

查询实例及个人空间绑定状态列表（分页）。

- **权限：** 管理员
- **响应：** 始终返回 JSON
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码（默认 1） |
| page_size | int | 否 | 每页条数（默认 20） |
| user | int | 否 | 按用户 ID 过滤 |
| agent_type | string | 否 | 按实例类型过滤（如 `openclaw`、`lightclaw-ace`、`hermes`） |
| group_id | int | 否 | 按分组 ID 过滤 |
| exclude_recycling | string | 否 | 传 `1` 时过滤掉 space 处于回收站中的实例 |

- **成功响应：**

```json
{
  "items": [
    {
      "instance_id": 42,
      "instance_name": "my-instance",
      "cvm_instance_id": "ins-xxxxxxxx",
      "agent_type": "openclaw",
      "user_id": 1,
      "username": "alice",
      "group_id": 3,
      "group_full_path": "研发中心/后端组",
      "space_status": "active",
      "space_id": 1,
      "smh_space_id": "spacexxxxxxxx",
      "storage_quota": 53687091200,
      "free_storage_quota": 53687091200,
      "used_storage": 1073741824,
      "bound_at": "2026-04-01T00:00:00Z",
      "expires_at": "2026-07-01T00:00:00Z",
      "to_be_deleted_at": null
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| agent_type | string | 实例类型（`openclaw`、`lightclaw-ace`、`hermes`），空值默认显示为 `openclaw` |
| username | string | 实例所属用户名，始终返回 |
| group_id | int | 实例所属分组 ID，0 表示未分组 |
| group_full_path | string | 分组全路径（如 "研发中心/后端组"），`group_id` 为 0 时为空串 |
| space_status | string | `none`（未绑定）、`active`（已绑定）、`recycling`（回收站中） |
| space_id | int | 个人空间数据库 ID，`space_status` 为 `none` 时不存在 |
| smh_space_id | string | SMH 空间 ID，`space_status` 为 `none` 时不存在 |
| used_storage | int | 实时查询 SMH API 获取，单位为字节 |

### `POST /admin/smh/instance-space`

批量开启或关闭实例的个人空间，逐个处理，部分失败不影响其他。

- **权限：** 管理员
- **响应：** 始终返回 JSON
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "instance_ids": [1, 2, 3],
  "action": "enable"
}
```

或全选所有实例：

```json
{
  "select_all": true,
  "action": "enable"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_ids | int[] | 否 | 实例数据库 ID 列表，与 `select_all` 二选一 |
| select_all | bool | 否 | 为 `true` 时忽略 `instance_ids`，对全部实例操作 |
| action | string | 是 | `enable` 或 `disable` |

- **enable 行为：**
  - 无空间记录：同步创建 SMH 空间并写入 DB，异步触发环境初始化（skill 安装 + token 注入）
  - 在回收站中：移除回收站标记，重置 env_initialized，异步触发环境初始化
  - 已开启：幂等返回成功

- **disable 行为：**
  - 将活跃空间标记进回收站，并立即异步触发环境卸载（skill 卸载 + token 清除）
  - 回收站保留 15 天后由后台任务彻底删除 SMH 空间

- **成功响应：**

```json
{
  "results": [
    {"instance_id": 1, "ok": true},
    {"instance_id": 2, "ok": false, "error": "该实例无活跃个人空间"}
  ]
}
```

- **失败响应：**
  - `400 {"error": "..."}` — 请求体格式错误、action 无效、instance_ids 为空

### 4.5 技能包管理

技能包（Skill Bundle）用于批量管理要预装到新实例的技能。同一时间只能有一个技能包处于启用状态。创建/重装实例时，系统会快照当前启用技能包中的技能，异步安装到 CVM 实例。

> **前置条件：** 创建、删除技能包及更新技能包内技能接口均要求 SMH 服务已启用。

### `GET /admin/skillhub-status`

查询当前租户的 SkillHub 迁移灰度状态。前端据此决定是否显示"前往 SkillHub"跳转入口。

- **响应：**

| 字段 | 类型 | 说明 |
|------|------|------|
| enabled | bool | 是否启用 SkillHub 迁移 |
| skillhub_url | string | SkillHub 前端地址（由 skill_hub_api_url 去掉 api. 前缀推导） |

### `GET /admin/skill-bundles`

查询技能包列表（分页）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |
| keyword | string | 否 | 按技能包名称模糊搜索 |
| id | uint | 否 | 按技能包 ID 精确筛选 |
| visibility_type | string | 否 | 按可见范围类型筛选：`all` 或 `group`。单独传时仅返回该类型的技能包 |
| group_id | string | 否 | 按分组 ID 筛选，多个用逗号分隔如 `1,3`。单独传时仅返回关联了这些分组的技能包；同时传 `visibility_type=all` 时返回全局可见 + 匹配分组的技能包 |
| skill_slug | string | 否 | 按技能 slug 反查所在初始技能包 |
| skill_source | string | 否 | 技能来源：`public` 或 `enterprise`，需与 `skill_slug` 一起使用；不传表示不限来源 |
| skill_version | string | 否 | 技能版本，需与 `skill_slug` 一起使用 |
| source_skillset_slug | string | 否 | 按来源公共技能包 slug 反查所在初始技能包 |

- **成功响应：**

```json
{
  "skill_bundles": [
    {
      "id": 1,
      "name": "通用技能包",
      "skill_count": 8,
      "enabled": true,
      "visibility_type": "all",
      "visible_groups": [],
      "created_at": "2026-03-26T10:00:00Z",
      "updated_at": "2026-03-31T10:00:00Z"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1,
  "total_pages": 1
}
```

传入 `skill_slug` 或 `source_skillset_slug` 时，`skill_bundles[]` 额外返回：

| 字段 | 类型 | 说明 |
|------|------|------|
| matched_skill_count | int | 命中的技能数量 |
| matched_skills | array | 命中的技能列表 |

### `POST /admin/skill-bundles/create`

创建技能包。同时在 SMH common space 创建对应目录。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 技能包名称（唯一） |
| visibility_type | string | 否 | 可见范围：`all`（默认，全部用户）或 `group`（按分组） |
| group_ids | string | 条件必填 | 分组 ID，逗号分隔。`visibility_type=group` 时必填 |

- **成功响应：** `{"ok": true, "id": 2}`
- **失败响应：**
  - `400 {"error": "技能包名称不能为空"}`
  - `400 {"error": "visibility_type 必须为 all 或 group"}`
  - `400 {"error": "按分组可见时必须选择至少一个分组"}`
  - `409 {"error": "同名技能包已存在"}`
  - `403 {"error": "SMH 服务未启用，请先在管理后台开通 SMH 服务"}`

### `POST /admin/skill-bundles/delete`

删除技能包。只能删除未启用的技能包。同时清理 SMH 文件和级联删除技能包内技能记录。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 技能包 ID（Query 参数或 Form 参数） |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `404 {"error": "技能包不存在"}`
  - `409 {"error": "技能包正在生效中，需先禁用"}`

### `POST /admin/skill-bundles/toggle`

启用/禁用技能包。`visibility_type=all` 的全局技能包之间互斥（同时只能启用一个），`visibility_type=group` 的分组技能包可以多个同时启用。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 技能包 ID（Query 参数或 Form 参数） |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `404 {"error": "技能包不存在"}`
  - `409 {"error": "已有其他应用范围为「全部用户」的技能包处于启用状态，请先禁用"}`

### `GET /admin/skill-bundles/detail`

查询技能包详情，包含技能包内的所有技能列表。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 技能包 ID（Query 参数） |

- **成功响应：**

```json
{
  "skill_bundle": {
    "id": 1,
    "name": "通用技能包",
    "skill_count": 8,
    "enabled": true,
    "created_at": "2026-03-26T10:00:00Z",
    "updated_at": "2026-03-31T10:00:00Z"
  },
  "skills": [
    {
      "id": 1,
      "skill_bundle_id": 1,
      "name": "openclaw-tavily-search",
      "slug": "openclaw-tavily-search",
      "version": "0.1.0",
      "source": "public",
      "source_skillset_slug": "finance-risk-assessment",
      "source_skillset_name": "金融风控技能包",
      "cos_zip_key": "skill-bundles/通用技能包/openclaw-tavily-search/openclaw-tavily-search-0.1.0.zip",
      "created_at": "2026-03-26T10:00:00Z",
      "updated_at": "2026-03-31T10:00:00Z"
    }
  ],
  "visible_groups": [
    {"group_id": 1, "group_name": "研发组"}
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| skill_bundle | object | 技能包基本信息 |
| skills | array | 技能包内技能列表 |
| visible_groups | array | 技能包可见分组列表；全局可见或未配置分组时为空数组 |
| skills[].source | string | 技能来源：`public`（公共技能）或 `enterprise`（企业技能） |
| skills[].source_skillset_slug | string | 来源公共技能包 slug，空表示无来源公共技能包 |
| skills[].source_skillset_name | string | 来源公共技能包名称，供前端展示和筛选 |
| skills[].cos_zip_key | string | SMH common space 中的 zip 文件路径，为空表示尚未完成同步 |

- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `404 {"error": "技能包不存在"}`

### `POST /admin/skill-bundles/update-skills`

批量更新技能包内的技能（添加/移除）。添加技能时会从 SkillHub 下载 zip 并上传到 SMH common space；移除技能时会清理 SMH 文件。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 技能包 ID（Query 参数） |

- **请求体：**

```json
{
  "add": [
    {"slug": "riskofficer", "name": "RiskOfficer", "source": "public", "source_skillset_slug": "finance-risk-assessment", "source_skillset_name": "金融风控技能包"},
    {"id": 5, "source": "enterprise"}
  ],
  "remove": [3, 4]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| add | array | 要添加的技能列表 |
| add[].id | uint | 可选。`source=public` 且未传 `slug` 时为 `public_skills` 表 ID；`source=enterprise` 时为 `skills` 表 ID |
| add[].source | string | 技能来源：`public` 或 `enterprise` |
| add[].slug | string | 可选。公共技能 slug；传入后可添加未收藏的公共技能 |
| add[].name | string | 可选。公共技能名称，未传时默认使用 slug |
| add[].version | string | 可选。公共技能版本；`latest` 或不传时使用默认版本 |
| add[].source_skillset_slug | string | 可选，来源公共技能包 slug，保存后随详情接口返回 |
| add[].source_skillset_name | string | 可选，来源公共技能包名称，保存后随详情接口返回 |
| remove | uint[] | 要移除的 `bundle_skills` 表记录 ID 列表 |

- **成功响应：**

```json
{"ok": true, "skill_count": 10, "added": 2, "removed": 1}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| skill_count | int | 更新后技能包内技能总数 |
| added | int | 本次添加的技能数 |
| removed | int | 本次移除的技能数 |

- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `400 {"error": "请求体格式错误"}`
  - `400 {"error": "add[].source 不能为空"}`
  - `400 {"error": "企业技能 ID=N 不存在"}`
  - `400 {"error": "公共技能 ID=N 不存在"}`
  - `404 {"error": "技能包不存在"}`
  - `409 {"error": "技能 slug-version 已存在于该技能包中"}`
  - `500 {"error": "SkillHub 地址未配置"}`

### `POST /admin/skill-bundles/batch-add-skills`

将公共技能批量加入多个初始技能包。添加技能时会按 slug 下载 zip 并上传到 SMH common space。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "bundle_ids": [1, 2],
  "skills": [
    {"slug": "riskofficer", "name": "RiskOfficer", "version": "3.0.0", "source_skillset_slug": "finance-risk-assessment", "source_skillset_name": "金融风控技能包"},
    {"slug": "openclaw-tavily-search", "name": "Tavily Search"}
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| bundle_ids | uint[] | 目标初始技能包 ID 列表 |
| skills | array | 要添加的公共技能列表 |
| skills[].slug | string | 公共技能 slug |
| skills[].name | string | 可选，公共技能名称；保存后随详情接口返回，不传时使用 slug |
| skills[].version | string | 可选，公共技能版本；`latest` 或不传时使用默认版本 |
| skills[].source_skillset_slug | string | 可选，来源公共技能包 slug，保存后随详情接口返回 |
| skills[].source_skillset_name | string | 可选，来源公共技能包名称，保存后随详情接口返回 |

- **成功响应：**

```json
{
  "ok": true,
  "added": 4,
  "bundle_results": [
    {"bundle_id": 1, "bundle_name": "通用技能包", "added": 2, "skill_count": 8},
    {"bundle_id": 2, "bundle_name": "研发技能包", "added": 2, "skill_count": 5}
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| added | int | 本次添加的技能总数 |
| bundle_results | array | 各初始技能包的添加结果 |
| bundle_results[].bundle_id | uint | 初始技能包 ID |
| bundle_results[].bundle_name | string | 初始技能包名称 |
| bundle_results[].added | int | 该初始技能包本次添加的技能数 |
| bundle_results[].skill_count | int | 添加后该初始技能包的技能总数 |

- **失败响应：**
  - `400 {"error": "bundle_ids 不能为空"}`
  - `400 {"error": "skills 不能为空"}`
  - `400 {"error": "参数 slug 不能为空"}`
  - `500 {"error": "读取公共技能 zip 失败 (status=N)"}`
  - `404 {"error": "技能包不存在"}`
  - `409 {"error": "技能 slug-version 已存在于该技能包中"}`

### `POST /admin/skill-bundles/update-visibility`

更新技能包的可见范围。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 技能包 ID（Query 参数或 Form 参数） |
| visibility_type | string | 是 | 可见范围：`all`（全部用户）或 `group`（按分组） |
| group_ids | string | 条件必填 | 分组 ID，逗号分隔。`visibility_type=group` 时必填 |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `400 {"error": "visibility_type 必须为 all 或 group"}`
  - `400 {"error": "按分组可见时必须选择至少一个分组"}`
  - `404 {"error": "技能包不存在"}`

### 4.6 公共技能收藏

管理员可以从 SkillHub 收藏公共技能到本地 `public_skills` 表，收藏后的技能可用于添加到技能包。

### `POST /admin/skills/favorite`

收藏公共技能。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "name": "搜索引擎",
  "slug": "openclaw-tavily-search",
  "version": "0.1.0",
  "description": "基于 Tavily 的网页搜索技能"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 技能名称 |
| slug | string | 是 | 技能唯一标识 |
| version | string | 否 | 版本号 |
| description | string | 否 | 技能描述 |

- **成功响应：** `{"ok": true, "skill_id": 1}`
- **失败响应：**
  - `400 {"error": "name 和 slug 不能为空"}`
  - `400 {"error": "请求体格式错误"}`

### `POST /admin/skills/unfavorite`

取消收藏公共技能。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 公共技能 ID（Query 参数或 Form 参数） |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `404 {"error": "技能不存在"}`

### `GET /admin/skills/favorited`

获取已收藏的公共技能列表（分页）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "skills": [
    {
      "id": 1,
      "name": "搜索引擎",
      "slug": "openclaw-tavily-search",
      "version": "0.1.0",
      "description": "基于 Tavily 的网页搜索技能",
      "total_downloads": 0,
      "total_favorites": 0,
      "created_at": "2026-03-31T10:00:00Z",
      "updated_at": "2026-03-31T10:00:00Z"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1,
  "total_pages": 1
}
```

**技能管理相关环境变量：**

| 环境变量 | 必填 | 说明 |
|---------|------|------|
| SMH_ENDPOINT | 是 | SMH 专属域名，如 `https://smhxxx.api.tencentsmh.cn` |
| SMH_LIBRARY_ID | 是 | SMH 媒体库 ID |
| SMH_LIBRARY_SECRET | 是 | SMH 媒体库密钥 |
| SMH_COMMON_SPACE | 否 | 公共空间 SpaceId |
| SMH_SKILLHUB_SPACE | 是 | 技能空间 SpaceId（技能上传/下发必需） |

### 4.7 公共技能包收藏

管理员可以从 SkillHub 收藏公共技能包（Skillset）到本地 `public_skillsets` 表，收藏后前端解包时将 slug 传给 SkillHub API 实时获取技能列表。

### `POST /admin/skillsets/favorite`

收藏公共技能包。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "slug": "finance-risk-assessment"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能包唯一标识 |

- **成功响应：** `{"ok": true, "skillset_id": 1}`
- **失败响应：**
  - `400 {"error": "slug 不能为空"}`
  - `400 {"error": "请求体格式错误"}`

### `POST /admin/skillsets/unfavorite`

取消收藏公共技能包。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 技能包 ID；与 `slug` 二选一 |
| slug | string | 条件 | 技能包唯一标识；与 `id` 二选一 |

> `id` 与 `slug` 至少传一个，不能同时指定。

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "缺少参数 id 或 slug"}`
  - `400 {"error": "id 和 slug 不能同时指定"}`
  - `404 {"error": "技能包不存在"}`

### `GET /admin/skillsets/favorited`

获取已收藏的公共技能包列表（分页）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "skillsets": [
    {
      "id": 1,
      "slug": "finance-risk-assessment",
      "created_at": "2026-05-26T10:00:00Z",
      "updated_at": "2026-05-26T10:00:00Z"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1,
  "total_pages": 1
}
```

### 4.8 角色管理

管理员可以预置角色（包含技能包 + 灵魂设定），员工创建 OpenClaw 时可一键选择角色。

### `GET /admin/roles`

查询角色列表（含每个角色的技能列表）。

- **权限：** 管理员
- **查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| visibility_type | string | 否 | 按可见范围类型筛选：`all` 或 `group`。单独传时仅返回该类型的角色 |
| group_id | string | 否 | 按分组 ID 筛选，多个用逗号分隔如 `1,3`。单独传时仅返回关联了这些分组的角色；同时传 `visibility_type=all` 时返回全局可见 + 匹配分组的角色 |

- **响应：** 始终 JSON

成功：

```json
{
  "roles": [
    {
      "id": 1,
      "name": "行业分析师",
      "description": "结构化分析，输出高质量行业洞察",
      "soul": "你是一位具备麦肯锡级别分析能力的行业研究顾问...",
      "visible": true,
      "visibility_type": "group",
      "visible_groups": [
        {"group_id": 1, "group_name": "研发组"}
      ],
      "sort_order": 0,
      "created_at": "2026-04-01T10:00:00Z",
      "updated_at": "2026-04-01T10:00:00Z",
      "skills": [
        {
          "id": 1,
          "openclaw_role_id": 1,
          "name": "Data Analysis",
          "slug": "data-analysis",
          "version": "1.0.0",
          "source": "public",
          "created_at": "2026-04-01T10:00:00Z"
        }
      ]
    }
  ],
  "total": 6
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| roles | array | 角色列表，按 `sort_order ASC, id ASC` 排序 |
| roles[].name | string | 角色名称，最多 8 个字 |
| roles[].description | string | 角色描述 |
| roles[].soul | string | 角色灵魂（System Prompt） |
| roles[].visible | bool | 是否对员工可见 |
| roles[].visibility_type | string | 可见范围类型：`all`（所有人）或 `group`（按分组） |
| roles[].visible_groups | array | 可见分组列表（`visibility_type=group` 时有值），含 `group_id` 和 `group_name` |
| roles[].sort_order | int | 排序序号，越小越靠前 |
| roles[].version | string | 角色版本号，`X.Y` 两段式格式（如 `1.0`、`2.0`） |
| roles[].skills | array | 角色关联的技能列表，空时为 `[]` |
| roles[].skills[].source | string | 技能来源：`public`（公共）或 `enterprise`（企业） |
| total | int | 角色总数 |

### `POST /admin/roles/create`

新增角色。新角色排在列表最前面（`sort_order=0`，其他角色 `sort_order + 1`）。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "name": "测试角色",
  "description": "角色描述",
  "soul": "角色灵魂文本",
  "visible": true,
  "visibility_type": "all",
  "group_ids": [],
  "skills": [
    {
      "name": "技能名",
      "slug": "skill-slug",
      "version": "1.0.0",
      "source": "public"
    }
  ]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 角色名称，最多 8 个字（按 Unicode 字符计数） |
| description | string | 否 | 角色描述 |
| soul | string | 否 | 角色灵魂（System Prompt） |
| visible | bool | 否 | 是否对员工可见，默认 `true` |
| visibility_type | string | 否 | 可见范围：`all`（默认，全部用户）或 `group`（按分组） |
| group_ids | uint[] | 条件 | 分组 ID 列表。`visibility_type=group` 时必填 |
| version | string | 否 | 角色版本号，`X.Y` 两段式（如 `1.0`、`2.0`），默认 `1.0` |
| skills | array | 否 | 角色技能列表 |
| skills[].name | string | 是 | 技能名称 |
| skills[].slug | string | 是 | 技能标识（安装用） |
| skills[].version | string | 否 | 版本号 |
| skills[].source | string | 否 | 技能来源，默认 `"public"` |

- **成功响应：** `{"ok": true, "id": 7}`
- **失败响应：**
  - `400 {"error": "角色名称不能为空"}`
  - `400 {"error": "角色名称不能超过 30 个字"}`
  - `400 {"error": "请求体格式错误"}`
  - `409 {"error": "同名角色已存在"}`
  - `500 {"error": "创建角色失败: ..."}`

### `POST /admin/roles/update`

编辑角色。技能采用全量替换策略（删除旧技能 → 创建新技能）。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 角色 ID（Query 参数） |

- **请求体：** 与创建相同；额外字段：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| version | string | 否 | 角色版本号，`X.Y` 两段式。传则必须严格大于旧版本号；不传则保留旧版本号不变 |

> `visible` 未传时不更新可见性；`version` 未传时不更新版本号。

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `400 {"error": "角色名称不能为空"}`
  - `400 {"error": "角色名称不能超过 30 个字"}`
  - `400 {"error": "版本号格式必须为 X.Y"}`
  - `400 {"error": "新版本号需高于上个版本号 X.Y"}`
  - `400 {"error": "请求体格式错误"}`
  - `404 {"error": "角色不存在"}`
  - `409 {"error": "同名角色已存在"}`

### `POST /admin/roles/delete`

删除角色（硬删除）。级联删除关联技能。已选择该角色的 OpenClaw 实例不受影响（`RoleID` 保留原值，查询时优雅降级为"通用助手"）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 角色 ID（Query 参数或 Form 参数） |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `404 {"error": "角色不存在"}`
  - `500 {"error": "删除角色失败: ..."}`

### `POST /admin/roles/toggle-visible`

切换角色可见性（取反）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 角色 ID（Query 参数或 Form 参数） |

- **成功响应：** `{"ok": true, "visible": false}`（返回切换后的新值）
- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `404 {"error": "角色不存在"}`
  - `500 {"error": "切换可见性失败: ..."}`

### `POST /admin/roles/reorder`

批量更新角色排序。前端拖拽后传入新的 ID 顺序数组。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "ids": [3, 1, 5, 2, 4, 6]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | uint[] | 是 | 角色 ID 数组，按期望顺序排列。`ids[0]` → `sort_order=0`，`ids[1]` → `sort_order=1`，以此类推 |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "ids 不能为空"}`
  - `400 {"error": "请求体格式错误"}`
  - `500 {"error": "更新排序失败 id=N: ..."}`

### `GET /admin/roles/detail`

查询角色详情（含技能列表）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 角色 ID（Query 参数） |

- **成功响应：**

```json
{
  "role": {
    "id": 1,
    "name": "行业分析师",
    "description": "结构化分析，输出高质量行业洞察",
    "soul": "你是一位具备麦肯锡级别分析能力的行业研究顾问...",
    "visible": true,
    "sort_order": 0,
    "version": "1.0",
    "created_at": "2026-04-01T10:00:00Z",
    "updated_at": "2026-04-01T10:00:00Z"
  },
  "skills": [
    {
      "id": 1,
      "openclaw_role_id": 1,
      "name": "Data Analysis",
      "slug": "data-analysis",
      "version": "1.0.0",
      "source": "public",
      "created_at": "2026-04-01T10:00:00Z"
    }
  ]
}
```

- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `404 {"error": "角色不存在"}`

---

### `POST /admin/roles/distribute`

管理端把角色当前最新版本批量推送到选中的实例（更新 `distributed_role_version` + 重下发 `SOUL.md` + 增量装新角色技能）。

校验管线（按顺序）：

1. 实例必须存在且属于当前租户（`identifier` 匹配），否则跳过 `not_found`
2. 实例 `agent_type` 必须支持角色配置，否则跳过 `agent_type_unsupported`
3. 实例必须 `running` 状态，否则跳过 `not_running`
4. 实例 `role_sync_status` 不能为 `updating`，否则跳过 `updating_in_progress`
5. 实例 `role_id` 必须等于路径参数 `id`，否则跳过 `role_mismatch`
6. 实例 `distributed_role_version` >= 角色 `version` 且 `role_sync_status='updated'`，否则跳过 `already_updated`

通过校验的实例：写入 `distributed_role_version` + 异步下发 `SOUL.md` + 异步装载新角色技能。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **审计动作：** `role_distribute`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 角色 ID（Query 参数） |
| instance_ids | uint[] | 是 | 要推送的实例 ID 列表（Body 字段），长度范围 `[1, 500]` |

- **请求示例：**

```json
{
  "instance_ids": [12, 25, 33, 47]
}
```

- **响应：** 始终 JSON

```json
{
  "accepted": 2,
  "skipped": [
    {"instance_id": 33, "reason": "already_updated"},
    {"instance_id": 47, "reason": "not_running"}
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| accepted | int | 通过校验并触发推送的实例数量 |
| skipped | array | 被跳过的实例列表，每项含 `instance_id` 和 `reason`（详见前端联调文档的 reason 枚举表） |

- **失败响应：**
  - `400 {"error": "role_id 参数必填"}`
  - `400 {"error": "instance_ids 不能为空"}`
  - `400 {"error": "批量推送实例数量超过上限 500"}`
  - `500 {"error": "..."}`

---

### `GET /admin/roles/instances`

查询绑定指定角色的实例列表（管理端「更新弹窗」数据源）。返回每个实例的角色版本同步状态，供前端展示「未更新 / 待更新 / 更新中 / 已更新 / 更新失败」标签。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| role_id | uint | 是 | 角色 ID |
| role_sync_status | string | 否 | 同步状态过滤：`pending` / `updating` / `updated` / `failed` / `all`（默认 `all`）；**支持逗号分隔多值**（如 `pending,failed`） |
| search | string | 否 | 模糊匹配 instance_name 或 cvm_instance_id |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20，上限 100 |

- **成功响应：**

```json
{
  "role": {
    "id": 5,
    "name": "行业分析师",
    "version": "2.0"
  },
  "page": 1,
  "page_size": 20,
  "total": 87,
  "items": [
    {
      "instance_id": 123,
      "cvm_instance_id": "ins-xxx",
      "instance_name": "agent-1",
      "user_id": 42,
      "username": "alice",
      "user_groups": [
        {"group_id": 7, "group_name": "技术部"}
      ],
      "group_id": 0,
      "group_name": "",
      "role_version": "1.0",
      "role_sync_status": "pending"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| role.version | string | 角色当前版本号，`X.Y` 两段式（如 `2.0`） |
| items[].role_version | string | 实例上记录的最近一次成功推送版本号；空字符串表示从未推送过 |
| items[].role_sync_status | string | 状态字段：`pending`（待更新 v旧版）/ `updating`（更新中）/ `updated`（已更新）/ `failed`（更新失败） |
| items[].user_groups | array | 该实例所有者所属的所有用户分组列表 |
| items[].group_id / group_name | uint / string | 实例创建时绑定的分组（创建时指定的分组） |

- **失败响应：**
  - `400 {"error": "role_id 参数必填"}`
  - `400 {"error": "role_sync_status 参数无效"}`
  - `404 {"error": "角色不存在"}`

---

### `GET /admin/roles/records`

分页查询角色下发记录（仅管理员 distribute 操作，排除用户 switch/create）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_ids | string | 否 | 实例数据库 ID，逗号分隔多值（如 `1,2,3`）；不传则查全部实例 |
| role_ids | string | 否 | 角色 ID，逗号分隔多值（如 `7,8`）；不传则查全部角色 |
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20，上限 100。`page_size=1` 可取最新一条 |

- **成功响应：**

```json
{
  "ok": true,
  "data": {
    "page": 1,
    "page_size": 20,
    "total": 2,
    "items": [
      {
        "id": 45,
        "instance_id": 123,
        "instance_cid": "ins-xxx",
        "role_id": 7,
        "role_name": "行业分析师",
        "version": "2.0",
        "operator_id": 1,
        "operator_username": "admin",
        "source": "distribute",
        "status": "updated",
        "soul_status": "success",
        "soul_error": "",
        "soul_set_at": "2026-07-03T10:00:00Z",
        "skill_status": "success",
        "skill_error": "",
        "skill_set_at": "2026-07-03T10:02:00Z",
        "skill_installation_ids": "[10,11,12]",
        "created_at": "2026-07-03T10:00:00Z",
        "updated_at": "2026-07-03T10:02:00Z"
      },
      {
        "id": 42,
        "instance_id": 123,
        "instance_cid": "ins-xxx",
        "role_id": 7,
        "role_name": "行业分析师",
        "version": "1.0",
        "operator_id": 1,
        "operator_username": "admin",
        "source": "distribute",
        "status": "failed",
        "soul_status": "success",
        "soul_error": "",
        "soul_set_at": "2026-07-02T14:58:00Z",
        "skill_status": "failed",
        "skill_error": "技能包 data-analysis 尚未完成 SMH 同步，请稍后重试",
        "skill_set_at": null,
        "skill_installation_ids": "[8,9]",
        "created_at": "2026-07-02T14:55:00Z",
        "updated_at": "2026-07-02T15:00:00Z"
      }
    ]
  }
}
```

- **错误响应：**
  - `400 {"error": "参数 instance_ids 无效"}` （传了但全部值无效）
  - `400 {"error": "参数 role_ids 无效"}` （传了但全部值无效）
  - `401` 未登录

---

### 4.9 安全检测

### `POST /admin/skills/scan-trigger`

对已上传的技能手动发起安全检测。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "skill_id": 123
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| skill_id | int | 是 | 技能 ID |

- **成功响应：**

```json
{
  "ok": true,
  "scan_id": 456,
  "status": "scanning",
  "message": "已提交安全检测，预计 5 分钟后完成"
}
```

- **错误响应：**
  - `400 {"error": "技能文件超过安全检测大小限制（7MB），无法进行安全检测"}`
  - `402 {"error": "SkillScan trial not activated, please activate trial first", "code": "LimitExceeded"}`（试用未开通/额度不足）
  - `404 {"error": "技能不存在"}`
  - `409 {"error": "该技能已有正在进行的安全检测，请等待完成"}`

### `GET /admin/skills/scan-config`

获取上传/更新技能时「提交安全检测」勾选框的默认值。

- **权限：** 管理员
- **响应：**

```json
{
  "skill_scan_default_enabled": false
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| skill_scan_default_enabled | bool | `true`=默认勾选, `false`=默认不勾选 |

### `POST /admin/skills/scan-config`

管理员设置上传技能时安全检测勾选框的默认行为。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "skill_scan_default_enabled": true
}
```

- **响应：**

```json
{
  "ok": true,
  "skill_scan_default_enabled": true
}
```

#### `scan_status` 枚举说明

| 值 | 说明 |
|---|------|
| not_scanned | 未检测（含检测失败） |
| scanning | 检测中 |
| safe | 安全（RiskLevel=benign） |
| suspicious | 可疑（RiskLevel=suspicious） |
| malicious | 恶意（RiskLevel=malicious） |

---

## 四（再续）、企业规范库管理（管理员）

所有企业规范接口始终返回 JSON，不受 Accept Header 影响。企业规范只下发本地 agent 实例（`instances.source = 'local'`），对 CVM 实例的下发/卸载/查询接口层拒绝。

**关键约束：**

- 规范文件为**单个 markdown 文件**，`≤ 1 MiB`，UTF-8，无 `\x00`；不打 zip
- 类型 `type` ∈ {`prompt`, `rule`}；同 slug 首次上传决定 type，后续版本必须一致
- 元信息编辑走 `POST /admin/rules/update`（对齐 `/admin/skills/update`），支持修改 `name` / `description` / `visibility_type` / `group_ids`
- 「更新规范内容」= 上传新版本 → 重新 `POST /admin/rules/distribute`，reporter 幂等覆写
- 分发流程与 skill 完全对齐：REST 只有 `distribute` / `uninstall` 两个动作入口

### R.1 规范 CRUD

### `GET /admin/rules`

查询规范列表。每个 slug 只返回最新版本（按语义化版本号排序）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 否 | 按名称或描述模糊搜索（同时匹配 name 和 description，满足其一即命中） |
| name | string | 否 | 按名称模糊搜索 |
| description | string | 否 | 按描述模糊搜索 |
| type | string | 否 | 类型筛选：`prompt` / `rule`；不传返回全部（两类混排） |
| source | string | 否 | 来源筛选：`enterprise` / `local`；不传返回全部 |
| visibility_type | string | 否 | 按可见范围类型筛选：`all` 或 `group`。单独传时仅返回该类型的规范 |
| group_id | string | 否 | 按分组 ID 筛选，多个用逗号分隔如 `1,3`。单独传时仅返回关联了这些分组的规范；同时传 `visibility_type=all` 时返回全局可见 + 匹配分组的规范 |
| project_id | string | 否 | 按项目 ID 筛选，多个用逗号分隔如 `1,3`。返回应用范围关联了任一项目的规范；与 `group_id` 同时传时取两者并集 |
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "rules": [
    {
      "id": 12,
      "slug": "coding-standards",
      "name": "编码规范",
      "description": "Go 代码 review 强制要求",
      "type": "rule",
      "source": "enterprise",
      "version": "1.2.0",
      "cos_key": "enterprise-rules/coding-standards/1.2.0.md",
      "file_size": 8421,
      "content_sha256": "abc...",
      "changelog": "新增 Go 代码规范",
      "distribute_count": 5,
      "visibility_type": "all",
      "visibility_groups": [],
      "visibility_projects": [
        {"id": 2, "name": "智能助手项目"}
      ],
      "created_at": "2026-07-05T10:00:00Z",
      "updated_at": "2026-07-05T10:00:00Z",
      "last_task": {
        "task_id": 34,
        "status": "completed",
        "type": "distribute",
        "total": 3,
        "success": 3,
        "failed": 0,
        "version": "1.2.0",
        "created_at": "2026-07-05T10:15:00Z"
      }
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

> `last_task` 为该规范最近一次下发任务的执行状态，未下发过时为 `null`。`status` 取值：`running` / `completed`。

### `GET /admin/rules/detail`

查询规范详情，返回该 slug 全部版本号。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 规范唯一标识 |
| version | string | 否 | 版本号；为空或 `latest` 表示取该 slug 最高版本 |

- **成功响应：**

```json
{
  "rule": {
    "id": 12,
    "slug": "coding-standards",
    "name": "编码规范",
    "description": "...",
    "type": "rule",
    "source": "enterprise",
    "version": "1.2.0",
    "cos_key": "enterprise-rules/coding-standards/1.2.0.md",
    "file_size": 8421,
    "content_sha256": "abc...",
    "changelog": "新增 Go 代码规范",
    "distribute_count": 5,
    "created_at": "2026-07-05T10:00:00Z",
    "updated_at": "2026-07-05T10:00:00Z",
    "event": "",
    "cmd": ""
  },
  "versions": ["1.2.0", "1.1.0", "1.0.0"]
}
```

> 🆕 **三期 Hook 资源**：当 `type=hook` 时，响应额外返回 `event`（触发时机）与 `cmd`（执行命令）字段，示例如下；`type=prompt` / `rule` 时这两个字段为空字符串 `""`。
>
> ```json
> {
>   "rule": {
>     "id": 88,
>     "slug": "hook-abc",
>     "name": "会话开始提醒",
>     "type": "hook",
>     "version": "1.0.0",
>     "event": "SessionStart",
>     "cmd": "echo hello",
>     "distribute_count": 0
>   },
>   "versions": ["1.0.0"]
> }
> ```

- **失败响应：**
  - `400 {"error": "slug 不能为空"}`
  - `404 {"error": "规范不存在"}`

### `POST /admin/rules/create`

上传新规范（首次插入）或新版本（同 slug）。规范文件为**单个 markdown 文件**。

- **权限：** 管理员
- **Content-Type：** `multipart/form-data`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | binary | 条件 | markdown 文件（`.md`，最大 1 MiB）。`type=prompt` / `rule` 时必填；`type=hook` 时忽略 |
| slug | string | 是 | 唯一标识。格式：小写字母、数字和连字符，长度 3-50，不能以连字符开头或结尾 |
| name | string | 是 | 显示名称 |
| type | string | 是 | 类型：`prompt` / `rule` / `hook`。同 slug 首次插入决定 type，后续版本必须与首版一致 |
| version | string | 是 | 版本号（语义化版本，如 `1.0.0`） |
| description | string | 否 | 规范描述 |
| event | string | 条件 | 触发时机（仅 `type=hook` 必填）。取值：`SessionStart` / `UserPromptSubmit` / `PreToolUse` / `PostToolUse` / `Stop` |
| cmd | string | 条件 | 执行命令（仅 `type=hook` 必填，非空）。管理员填写的原始命令，由本地插件在对应 event 触发时执行 |
| visibility_type | string | 否 | 应用范围：`all`（全部用户，默认）或 `group`（按分组）。不传则继承同 slug 旧版本的设置；传入范围时必传 |
| group_ids | string | 条件 | 分组 ID 列表，多个用逗号分隔，如 `1,3`。传入范围时必须同时传该字段；空字符串表示清空全部已有分组绑定 |
| project_ids | string | 条件 | 项目 ID 列表，多个用逗号分隔。传入范围时必须同时传该字段；空字符串表示清空全部已有项目绑定 |

> 🆕 **三期 Hook 资源**：`type=hook` 时无需上传文件，改为通过表单字段 `event` + `cmd` 定义触发时机与执行命令，单次提交仅支持一种触发时机 + 一条命令。slug 由系统或管理员指定（唯一标识）。其余应用范围校验与 prompt/rule 一致。

**文件校验规则：**

1. 文件名后缀 `.md`（大小写不敏感）
2. 文件大小 ≤ 1 MiB
3. 内容非空且为合法 UTF-8
4. 不含 `\x00` 字节（防上传二进制）

**事务保证：** 先写入数据库，再上传 SMH。SMH 上传失败时自动回滚数据库并清理已上传的文件。

- **成功响应：** `{"ok": true, "id": 12, "slug": "coding-standards", "version": "1.2.0"}`

> 🆕 **三期 Hook 资源**：`type=hook` 时成功响应同样返回 `id` / `slug` / `version`，其余 `event` / `cmd` 字段已落库，可通过 `GET /admin/rules/detail` 查询。示例：`{"ok": true, "id": 88, "slug": "hook-abc", "version": "1.0.0"}`

> **版本递增校验：** 新版本号必须严格大于该 slug 的现有最高版本号。首次发布不受限制。
> **类型一致性校验：** 同 slug 后续版本 `type` 必须与首版一致，否则拒绝。

- **失败响应：**
  - `400 {"error": "slug、name、type、version 为必填字段"}`
  - `400 {"error": "type 必须为 prompt 或 rule 或 hook"}`
  - `400 {"error": "hook 类型必须提供 event 和 cmd"}` — `type=hook` 但缺 event 或 cmd
  - `400 {"error": "event 必须为 SessionStart / UserPromptSubmit / PreToolUse / PostToolUse / Stop 之一"}`
  - `400 {"error": "type 与首版本不一致（slug=coding-standards, 首版 type=rule）"}`
  - `400 {"error": "slug 格式不合法，只允许小写字母、数字和连字符，长度 3-50，不能以连字符开头或结尾"}`
  - `400 {"error": "新版本号 1.0.0 必须大于现有最高版本 2.0.0"}`
  - `400 {"error": "文件必须为 .md，且大小不超过 1 MiB"}`
  - `400 {"error": "文件内容非法：非 UTF-8 或包含 \\x00"}`
  - `500 {"error": "存储服务不可用: ..."}`

### `POST /admin/rules/delete`

删除规范。**不级联卸载已装实例**（对齐 `POST /admin/skills/delete` 语义）；如需下架并从实例卸载，须另行调用 `POST /admin/rules/uninstall`。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 规范唯一标识 |
| version | string | 否 | 版本号；不传或为空表示删除该 slug 全部版本 |

- **前置检查：** 若该 rule 存在 `RuleDistributionTask.status='running'` 的下发任务，直接 400 拒绝。
- **副作用：**
  - 事务内软删 `enterprise_rules` 记录
  - 不触及 `RuleDistributionRecord`（保留历史）、不触及 `LocalInstanceRule`（本地实例上的 md 不会被动清理）
  - 事务外异步删除 SMH 上对应版本的 md 文件；失败仅告警，不影响 DB 结果

- **成功响应：** `{"ok": true, "deleted_rules": 2}`

- **失败响应：**
  - `400 {"error": "slug 不能为空"}`
  - `400 {"error": "规范有正在运行的下发任务，请等待任务结束后重试"}`
  - `404 {"error": "规范不存在"}`

### R.2 规范文件与任务查询

### `POST /admin/rules/update`

更新规范元信息（name/description/visibility）。对齐 `/admin/skills/update`。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 规范标识 |
| version | string | 是 | 规范版本 |
| name | string | 否 | 新名称（不传不变）|
| description | string | 否 | 新描述（不传不变）|
| visibility_type | string | 否 | `all` / `group`；不传不变；传入范围时必传 |
| group_ids | string | 条件 | 分组 ID 列表，多个用逗号分隔，如 `1,3`。更新范围时必须同时传该字段；空字符串清空已有分组绑定 |
| project_ids | string | 条件 | 项目 ID 列表，多个用逗号分隔。更新范围时必须同时传该字段；空字符串清空已有项目绑定 |

- **成功响应：** `{"ok": true}`

- **失败响应：**
  - `400 {"error": "slug 和 version 为必填参数"}`
  - `404 {"error": "规范不存在"}`
  - `500 {"error": "更新规范失败: ..."}`

### `GET /admin/rules/files`

查询某 slug 全部版本的下载 URL 与元信息。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 规范唯一标识 |

- **成功响应：**

```json
{
  "slug": "coding-standards",
  "versions": [
    {
      "version": "1.2.0",
      "download_url": "https://smh.example.com/enterprise-rules/coding-standards/1.2.0.md?...",
      "content_sha256": "abc...",
      "file_size": 8421
    },
    {
      "version": "1.1.0",
      "download_url": "https://...",
      "content_sha256": "...",
      "file_size": 8100
    }
  ]
}
```

> `download_url` 通过 `buildSMHDownloadURL(cosKey, false)` 生成，走公网域名（reporter 在用户机器上下载）。

- **失败响应：**
  - `400 {"error": "slug 不能为空"}`
  - `404 {"error": "规范不存在"}`

### `GET /admin/rules/tasks`

查询规范下发任务列表（分页），含每个 task 的 records 明细。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 条件必填 | 规范 slug（`slug` 与 `batch_id` 至少填一个） |
| batch_id | string | 条件必填 | 精确定位 batch |
| type | string | 否 | 任务类型：`distribute` / `uninstall` / `all`（默认全部） |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "tasks": [
    {
      "id": 34,
      "created_at": "2026-07-05T10:15:00Z",
      "operator": "alice",
      "slug": "coding-standards",
      "rule_type": "rule",
      "version": "1.2.0",
      "batch_id": "",
      "total": 3,
      "success": 3,
      "failed": 0,
      "pending": 0,
      "status": "completed",
      "type": "distribute",
      "records": [
        {
          "instance_id": 42,
          "cvm_instance_id": "local-agent-abc",
          "instance_name": "dev-macbook",
          "username": "bob",
          "status": "success",
          "error": ""
        }
      ]
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 5
}
```

- **失败响应：**
  - `400 {"error": "slug 或 batch_id 至少填一个"}`

### `GET /admin/rules/instances`

查询某规范的实例分发/安装状态列表。**仅返回 `source=local` 的实例**（企业规范库不下发 CVM）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 规范唯一标识 |
| version | string | 否 | 目标版本；未传取该 slug 最新版本 |
| status | string | 否 | 安装状态过滤，多个用逗号分隔，如 `installed,outdated`。取值同 skill 侧 |
| search | string | 否 | 匹配 `instances.name` 或 `instance_id` |
| group_id | string | 否 | 用户组过滤，多个用逗号分隔；`0` 表示未分组 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20，上限 500 |

- **成功响应：**

```json
{
  "instances": [
    {
      "instance_id": 42,
      "cvm_instance_id": "local-agent-abc",
      "instance_name": "dev-macbook",
      "instance_type": "workbuddy",
      "user_id": 8,
      "username": "bob",
      "user_groups": [{"group_id": 1, "group_name": "研发"}],
      "status": "installed",
      "version": "1.2.0",
      "latest_version": "1.2.0",
      "instance_status": "running",
      "instance_status_label": "运行中",
      "transient": false
    }
  ],
  "page": 1,
  "page_size": 500,
  "total": 12
}
```

> 结果只保留 `instance_status == running` 的本地实例；`total` 是过滤后的总数。`status` 字段取值与 skill 一致，参见 `GET /admin/skills/instances` 附近的实例安装状态矩阵。

- **失败响应：**
  - `400 {"error": "slug 不能为空"}`
  - `404 {"error": "规范不存在"}`

### R.3 规范下发与卸载

### `POST /admin/rules/distribute`

批量下发规范到本地 agent 实例。异步执行，接口同步返回 `task_id`。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "slug": "coding-standards",
  "version": "1.2.0",
  "instance_ids": [42, 43, 44]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 规范唯一标识 |
| version | string | 否 | 版本号；`latest` 或不传使用最新版本 |
| instance_ids | int[] | 是 | 目标实例 ID 列表（服务端会去重，并过滤掉非本地实例） |

**行为说明：**

- 服务端过滤 `instance_ids` 中的非 `source=local` 实例；过滤后为空则 400
- 加分布式锁 `enterprise_rule:distribute:{slug}` 30 分钟，防并发重复下发
- 建 `RuleDistributionTask`（`type=distribute`, `rule_type=<主表 type>`）+ 每实例一条 `RuleDistributionRecord`（`type=distribute`, `status=pending`）
- 本地实例的 record 保留 `pending`，由 reporter 通过 `POST /local-agent/sync` 拉取并处理
- 「更新」= 上传新版本后再次调用本接口。reporter 端幂等覆写同名文件，服务端不区分「首装」与「重装」

- **成功响应：** `{"ok": true, "task_id": 34, "slug": "coding-standards", "version": "1.2.0"}`

- **失败响应：**
  - `400 {"error": "slug 不能为空"}`
  - `400 {"error": "instance_ids 不能为空"}`
  - `400 {"error": "没有有效的本地实例（企业规范不下发 CVM 实例）"}`
  - `404 {"error": "规范不存在"}`
  - `409 {"error": "该规范正在被其他操作处理，请稍后重试"}`

> 下发进度可通过 `GET /admin/rules/tasks?slug=X` 轮询。

### `POST /admin/rules/uninstall`

批量从本地 agent 实例上卸载规范。异步执行，接口同步返回 `task_id`。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "slug": "coding-standards",
  "instance_ids": [42]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 规范唯一标识 |
| instance_ids | int[] | 是 | 目标实例 ID 列表 |

**行为说明：**

- 加分布式锁 `enterprise_rule:uninstall:{slug}` 30 分钟
- 建 `RuleDistributionTask`（`type=uninstall`）+ 每实例 `RuleDistributionRecord`（`type=uninstall`, `status=pending`）
- 本地实例保留 `pending` 由 reporter 通过 sync 拉取

- **成功响应：** `{"ok": true, "task_id": 35}`

- **失败响应：**
  - `400 {"error": "slug 不能为空"}`
  - `400 {"error": "instance_ids 不能为空"}`
  - `400 {"error": "没有有效的本地实例"}`
  - `404 {"error": "规范不存在"}`
  - `409 {"error": "该规范正在被其他操作处理，请稍后重试"}`

> 卸载进度可通过 `GET /admin/rules/tasks?slug=X&type=uninstall` 轮询。

### R.4 实例已下发规范

### `GET /admin/instances/rules`

查询指定本地实例上的已下发规范列表。**仅本地实例可用**，CVM 实例调用返回 400。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 二选一 | 实例数据库 ID |
| instance_id | string | 二选一 | 实例 CID（本地 agent CID 字符串） |

- **成功响应：**

```json
{
  "instance_id": 42,
  "rules": [
    {
      "slug": "coding-standards",
      "name": "编码规范",
      "type": "rule",
      "version": "1.2.0",
      "distributed_at": "2026-07-05T12:34:56Z",
      "source": "enterprise"
    },
    {
      "slug": "codebuddy-prompt",
      "name": "CodeBuddy Prompt",
      "type": "prompt",
      "version": "0.3.1",
      "distributed_at": "2026-07-04T10:00:00Z",
      "source": "enterprise"
    }
  ],
  "total": 2
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| rules[].slug | string | 规范唯一标识 |
| rules[].name | string | 规范展示名 |
| rules[].type | string | 规范类型：`prompt` / `rule` |
| rules[].version | string | 已装版本 |
| rules[].distributed_at | string | 首次下发成功时间（RFC3339） |
| rules[].source | string | 来源：`enterprise` / `local` |

> 数据来源：`local_instance_rules`（reporter 上报后的本地事实快照）。

- **失败响应：**
  - `400 {"error": "本接口仅适用于本地实例"}`
  - `400 {"error": "缺少参数 id 或 instance_id"}`
  - `404 {"error": "实例不存在"}`

### R.5 reporter sync 命令扩展

`POST /local-agent/sync` 响应中的 `commands[]` 数组新增 4 种规范相关命令。命名规则 `<action>_<rule_type>_rule`：

| 命令 type | records.type | rule_type | reporter 语义 |
|---|---|---|---|
| `install_prompt_rule`   | distribute | prompt | 落 prompt 类 md（reporter 侧决定 fixed 落地路径；已存在则幂等覆盖） |
| `uninstall_prompt_rule` | uninstall  | prompt | 移除 prompt 类 md |
| `install_rule_rule`     | distribute | rule   | 落 rule 类 md（reporter 侧写 rules 目录；已存在则幂等覆盖） |
| `uninstall_rule_rule`   | uninstall  | rule   | 移除 rule 类 md |

**服务端不下发独立的 `update` 命令**（对齐 `install_skill` 现有做法，`local_agent.go:536` 已有明确注释：`update_skill 不单独区分，reporter 端幂等覆盖`）。

命令携带字段：

```json
{
  "id": 202,
  "type": "install_rule_rule",
  "rule_slug": "coding-standards",
  "rule_version": "1.2.0",
  "rule_type": "rule",
  "download_url": "https://smh.../....md",
  "content_sha256": "abc..."
}
```

- `install_*` 命令携带 `download_url`（走公网）+ `content_sha256`
- `uninstall_*` 命令仅携带 `rule_slug` / `rule_version` / `rule_type`
- 老 reporter 收到未知 type 应跳过（一期已实现该保护）


---

## 四（续）、插件管理（管理员）

所有插件管理接口始终返回 JSON，不受 Accept Header 影响。

> **前置条件：** 所有插件及插件分类接口均要求 SMH 服务已启用（`smh_enabled = 1`）。若未启用，返回 `403 {"error": "SMH 服务未启用，请先在管理后台开通 SMH 服务"}`。

### P.1 插件分类管理

### `GET /admin/plugin-categories`

查询插件分类列表（分页）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "categories": [
    {
      "id": 1,
      "name": "基础插件",
      "description": "系统内置基础插件",
      "created_at": "2026-04-10T10:00:00Z",
      "plugin_count": 3
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

### `POST /admin/plugin-categories/create`

创建插件分类。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 分类名称（唯一，最长 100 字符） |
| description | string | 否 | 分类描述 |

- **成功响应：** `{"ok": true, "id": 3}`
- **失败响应：**
  - `400 {"error": "分类名称不能为空"}`
  - `400 {"error": "分类名称不能超过 100 个字符"}`
  - `400 {"error": "分类名称已存在"}`

### `POST /admin/plugin-categories/update`

更新插件分类。支持清空 `description` 为空字符串。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 分类 ID |
| name | string | 否 | 新名称 |
| description | string | 否 | 新描述（传空字符串可清空） |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "缺少分类 ID"}`
  - `400 {"error": "分类名称已存在"}`
  - `404 {"error": "分类不存在"}`

### `POST /admin/plugin-categories/delete`

删除插件分类。删除时会自动清理该分类与插件的关联关系，插件本身不受影响。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 分类 ID |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `404 {"error": "分类不存在"}`

### P.2 插件 CRUD

### `GET /admin/plugins`

查询插件列表。每个 slug 只返回最新版本（按语义化版本号排序）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 否 | 按名称或描述模糊搜索 |
| category_ids | string | 否 | 按分类 ID 筛选，多个用逗号分隔，如 `1,3` |
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "plugins": [
    {
      "id": 5,
      "slug": "system-info-plugin",
      "name": "系统信息插件",
      "version": "1.0.0",
      "description": "获取系统信息",
      "plugin_id": "@anthropic/system-info",
      "plugin_format": "openclaw",
      "kind": "",
      "npm_package": "",
      "file_size": 102400,
      "created_at": "2026-04-10T10:00:00Z",
      "updated_at": "2026-04-10T10:00:00Z",
      "categories": [
        {"id": 1, "name": "基础插件"}
      ],
      "last_task": {
        "task_id": 12,
        "status": "completed",
        "type": "distribute",
        "total": 10,
        "success": 9,
        "failed": 1,
        "version": "1.0.0",
        "created_at": "2026-04-12T15:30:00Z"
      },
      "changelog": "首次发布",
      "visibility_type": "all",
      "visibility_groups": [],
      "installed_count": 9,
      "has_running_task": false
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| plugins[].last_task | object/null | 最近一次下发/卸载任务，无任务时为 `null` |
| plugins[].last_task.status | string | 任务状态：`running`（进行中）、`completed`（已完成） |
| plugins[].last_task.type | string | 任务类型：`distribute`（下发）、`uninstall`（卸载） |
| plugins[].changelog | string | 最新版本更新说明 |
| plugins[].visibility_type | string | 应用范围：`all`（全部可见）、`group`（按分组可见） |
| plugins[].visibility_groups | array | 关联分组列表，每项含 `group_id` 和 `group_name` |
| plugins[].installed_count | int | 当前已安装实例数 |
| plugins[].has_running_task | bool | 是否有进行中的任务（前端据此禁用操作按钮） |

### `POST /admin/plugins/create`

创建插件（上传 zip 压缩包）。服务端会校验 zip 包中的 `openclaw.plugin.json` 或 Bundle 目录，提取元数据后上传到 SMH 存储。

支持两种 zip 包结构：

- **规范结构（推荐）：** `{slug}/openclaw.plugin.json`、`{slug}/src/...` — 包含一个以 slug 命名的顶级目录
- **简化结构：** `openclaw.plugin.json`、`src/...` — 根目录直接放文件，服务端自动转换为规范结构

支持两种插件格式：

- **OpenClaw 原生格式：** 包含 `openclaw.plugin.json`，其中 `id` 字段为必填
- **Bundle 格式：** 包含 `.codex-plugin/`、`.claude-plugin/` 或 `.cursor-plugin/` 目录

- **权限：** 管理员
- **Content-Type：** `multipart/form-data`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | binary | 是 | zip 压缩包（最大 200MB） |
| slug | string | 是 | 唯一标识。格式：小写字母、数字和连字符，长度 3-50，不能以连字符开头或结尾 |
| name | string | 是 | 显示名称 |
| version | string | 是 | 版本号（语义化版本，如 `1.0.0`，各段不超过 999） |
| description | string | 否 | 插件描述 |
| npm_package | string | 否 | npm 包名（可选） |
| category_ids | string | 否 | 分类 ID 列表，多个用逗号分隔，如 `1,3` |
| visibility_type | string | 否 | 应用范围：`all`（全部用户可见，默认）或 `group`（按分组可见） |
| group_ids | string | 否 | 逗号分隔的分组 ID，仅 `visibility_type=group` 时有效 |

**Zip 包校验规则：**

1. 必须包含 `openclaw.plugin.json` 文件（原生格式）或 Bundle 目录（`.codex-plugin/` 等）
2. `openclaw.plugin.json` 中 `id` 字段为必填，`kind` 仅支持 `memory` / `context-engine` / 空
3. 不允许包含 `..` 路径（防止 zip slip）
4. 解压后总大小不超过 200MB
5. 内置 ZIP 炸弹防护（实际读取字节数限制）

- **成功响应：** `{"ok": true, "id": 6, "slug": "my-plugin", "version": "1.0.0", "plugin_id": "@scope/my-plugin"}`
- **失败响应：**
  - `400 {"error": "slug、name、version 为必填字段"}`
  - `400 {"error": "slug 格式不合法，只允许小写字母、数字和连字符，长度 3-50，不能以连字符开头或结尾"}`
  - `400 {"error": "该插件版本已存在（slug=my-plugin, version=1.0.0），请修改后重试"}`
  - `400 {"error": "zip 中未找到 openclaw.plugin.json 或 Bundle 目录"}`
  - `400 {"error": "openclaw.plugin.json 中缺少 id 字段"}`
  - `400 {"error": "zip 解压后总大小超过 200MB 限制"}`
  - `500 {"error": "SMH 存储服务不可用: ..."}`

### `POST /admin/plugins/update`

更新插件版本。创建基于当前最新版本的新版本记录，支持上传新 ZIP 包或沿用旧版本文件。未传的元信息字段自动从当前版本继承。

- **权限：** 管理员
- **Content-Type：** `multipart/form-data`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 插件 slug |
| version | string | 是 | 新版本号（格式 x.y.z，必须高于当前最新版本） |
| name | string | 否 | 显示名称（不传则继承当前版本） |
| description | string | 否 | 描述（不传则继承当前版本） |
| changelog | string | 否 | 版本更新说明 |
| file | binary | 否 | 新版本 ZIP 文件（不传则沿用当前版本的文件） |
| npm_package | string | 否 | npm 包名（不传则继承当前版本） |
| category_ids | string | 否 | 分类 ID 逗号分隔（不传则继承当前版本） |
| visibility_type | string | 否 | 应用范围：`all` 或 `group`（不传则继承当前版本） |
| group_ids | string | 否 | 逗号分隔的分组 ID，仅 `visibility_type=group` 时有效 |

- **成功响应：**

```json
{"ok": true, "id": 456, "slug": "my-plugin", "version": "1.1.0", "plugin_id": "@scope/my-plugin"}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 是否成功 |
| id | uint | 新创建的插件版本记录 ID |
| slug | string | 插件唯一标识 |
| version | string | 新版本号 |
| plugin_id | string | 插件 ID（从新 ZIP 解析或继承自旧版本） |

- **失败响应：**
  - `400 {"error": "slug 和 version 为必填字段"}`
  - `400 {"error": "版本号必须高于当前最新版本 x.y.z"}`
  - `404 {"error": "插件不存在"}`
  - `409 {"error": "该插件有进行中的任务，请稍后重试"}`

### `POST /admin/plugins/delete`

删除指定 slug+version 的插件。有进行中的下发任务时不允许删除。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 插件 slug |
| version | string | 是 | 插件版本号 |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "slug 和 version 为必填字段"}`
  - `400 {"error": "该版本有进行中的下发任务，无法删除"}`
  - `404 {"error": "插件不存在"}`

### P.3 插件详情与文件

### `GET /admin/plugins/detail`

查询插件详情，包含所有版本列表和分类信息。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 插件 slug |
| version | string | 否 | 版本号，不传或传 `latest` 返回最新版本 |

- **成功响应：**

```json
{
  "plugin": {
    "id": 5,
    "slug": "system-info-plugin",
    "name": "系统信息插件",
    "version": "1.0.0",
    "description": "获取系统信息",
    "plugin_id": "@anthropic/system-info",
    "plugin_format": "openclaw",
    "kind": "",
    "npm_package": "",
    "config_schema": "{}",
    "providers": "[]",
    "channels": "[]",
    "categories": [{"id": 1, "name": "基础插件"}],
    "file_size": 102400,
    "cos_zip_key": "plugins/system-info-plugin/system-info-plugin-1.0.0.zip",
    "cos_dir_key": "plugins/system-info-plugin/system-info-plugin-1.0.0/",
    "created_at": "2026-04-10T10:00:00Z",
    "updated_at": "2026-04-10T10:00:00Z",
    "changelog": "首次发布",
    "distribute_count": 50,
    "has_running_task": false,
    "installed_count": 42,
    "visibility_type": "all",
    "visibility_groups": []
  },
  "versions": ["1.0.0"]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| plugin.changelog | string | 当前版本更新说明 |
| plugin.distribute_count | int | 累计下发成功数 |
| plugin.has_running_task | bool | 是否有进行中的任务（前端据此禁用操作按钮） |
| plugin.installed_count | int | 当前已安装实例数 |
| plugin.visibility_type | string | 应用范围：`all`（全部可见）、`group`（按分组可见） |
| plugin.visibility_groups | array | 关联分组列表，每项含 `group_id` 和 `group_name` |

- **失败响应：**
  - `400 {"error": "缺少 slug 参数"}`
  - `404 {"error": "插件不存在"}`

### `GET /admin/plugins/files`

查询插件所有版本的文件列表。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 插件 slug |

- **成功响应：**

```json
{
  "slug": "system-info-plugin",
  "versions": [
    {
      "version": "1.0.0",
      "files": [
        "plugins/system-info-plugin/system-info-plugin-1.0.0/openclaw.plugin.json",
        "plugins/system-info-plugin/system-info-plugin-1.0.0/src/index.js"
      ],
      "changelog": "初始版本",
      "created_at": "2026-05-17 10:00:00"
    }
  ]
}
```

- **失败响应：**
  - `400 {"error": "缺少 slug 参数"}`
  - `404 {"error": "插件不存在"}`

### P.4 插件下发与卸载

### `GET /admin/plugins/tasks`

查询插件下发/卸载任务列表（分页），包含每个任务的实例级执行记录。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 插件 slug |
| type | string | 是 | 任务类型：`distribute`（下发）、`uninstall`（卸载） |
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "tasks": [
    {
      "id": 1,
      "type": "distribute",
      "created_at": "2026-04-12T15:30:00Z",
      "operator": "admin",
      "version": "1.0.0",
      "total": 10,
      "success": 9,
      "failed": 1,
      "pending": 0,
      "status": "completed",
      "records": [
        {
          "instance_id": 1,
          "cvm_instance_id": "ins-abc123",
          "instance_name": "开发机-01",
          "username": "zhangsan",
          "status": "success",
          "error": ""
        },
        {
          "instance_id": 2,
          "cvm_instance_id": "ins-def456",
          "instance_name": "开发机-02",
          "username": "lisi",
          "status": "failed",
          "error": "脚本执行超时"
        }
      ]
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| tasks[].type | string | 任务类型：`distribute`（下发）、`uninstall`（卸载） |
| tasks[].status | string | 任务状态：`running`（进行中）、`completed`（已完成） |
| tasks[].total | int | 目标实例总数 |
| tasks[].success | int | 成功数（实时聚合） |
| tasks[].failed | int | 失败数（实时聚合） |
| tasks[].pending | int | 待执行数（实时聚合） |
| tasks[].records[].status | string | 实例级状态：`pending`（等待中）、`success`（成功）、`failed`（失败）、`skipped`（跳过） |
| tasks[].records[].error | string | 失败原因（仅 status=failed 时有值） |

- **失败响应：**
  - `400 {"error": "缺少 slug 参数"}`
  - `400 {"error": "type 参数必须为 distribute 或 uninstall"}`
  - `404 {"error": "插件不存在"}`

### `GET /admin/plugins/instances`

查询实例的插件安装情况（分页），支持按安装状态、实例类型、用户组和关键词筛选。仅返回智能体类型支持插件（当前仅 OpenClaw）且实时状态为运行中的实例。

> **实时状态判断：** 接口通过批量调用 CVM API 获取实时运行状态，结合 `ResolveInstanceStatus` 语义状态引擎计算，只返回 `instance_status=running` 的实例。相比数据库缓存的 `last_cvm_state`，状态判断更精确（秒级）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 插件 slug |
| status | string | 否 | 按安装状态筛选，多个用逗号分隔。取值：`uninstalled`、`installed`、`installing`、`failed`、`outdated`、`upgrade_failed`、`uninstalling`、`uninstall_failed` |
| search | string | 否 | 按实例名称或 CVM 实例 ID 模糊搜索 |
| instance_type | string | 否 | 按实例类型筛选，支持逗号分隔多类型，如 `openclaw,hermes`。不传返回所有支持插件的类型 |
| group_id | string | 否 | 按用户组筛选，支持逗号分隔多个用户组 ID，如 `1,2,3`。`0` 表示未分组用户的实例，可与正常 ID 组合使用，如 `0,1,3` |
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "instances": [
    {
      "instance_id": 1,
      "cvm_instance_id": "ins-abc123",
      "instance_name": "开发机-01",
      "instance_type": "openclaw",
      "user_id": 5,
      "username": "zhangsan",
      "last_cvm_state": "RUNNING",
      "status": "installed",
      "version": "1.0.0",
      "user_groups": [
        {"group_id": 1, "group_name": "研发组"}
      ],
      "instance_status": "running",
      "instance_status_label": "运行中",
      "transient": false
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 10
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| instances[].instance_id | uint | 实例数据库 ID |
| instances[].cvm_instance_id | string | CVM 实例 ID |
| instances[].instance_name | string | 实例名称 |
| instances[].instance_type | string | 智能体类型（如 `openclaw`） |
| instances[].user_id | uint | 所属用户 ID |
| instances[].username | string | 所属用户名 |
| instances[].last_cvm_state | string | 数据库缓存的 CVM 状态 |
| instances[].status | string | 安装状态：`uninstalled`（未安装）、`installed`（已安装当前版本）、`installing`（安装中）、`failed`（安装失败）、`outdated`（已安装旧版本，需更新）、`upgrade_failed`（升级失败，旧版本仍在）、`uninstalling`（卸载中）、`uninstall_failed`（卸载失败） |
| instances[].version | string | 已安装的版本号 |
| instances[].user_groups | array | 用户所属分组列表 |
| instances[].user_groups[].group_id | uint | 用户组 ID |
| instances[].user_groups[].group_name | string | 用户组名称 |
| instances[].instance_status | string | 实时语义状态（固定为 `running`，因为接口只返回运行中的实例） |
| instances[].instance_status_label | string | 状态中文标签（如 `运行中`） |
| instances[].transient | bool | 是否为过渡态（固定为 `false`，运行中不是过渡态） |

> **排序规则：** `installed` 和 `installing` 状态的实例排在末尾，其余按创建时间倒序。

- **失败响应：**
  - `400 {"error": "缺少 slug 参数"}`
  - `404 {"error": "插件不存在"}`

### `POST /admin/plugins/distribute`

将插件部署至所选实例。可通过 `instance_ids` 显式指定实例，或通过 `select_all=true` 按目标版本的安装状态和用户组选择全部匹配实例。通过 TAT 远程执行 `install_plugin_from_smh.sh`。

> **后端兜底过滤：** 仅对支持插件能力的 Agent 实例执行下发；不新增 running 状态限制。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "slug": "system-info-plugin",
  "version": "1.0.0",
  "instance_ids": [1, 2, 3]
}
```

全选示例：

```json
{
  "slug": "system-info-plugin",
  "version": "1.0.0",
  "select_all": true,
  "statuses": ["uninstalled", "outdated"],
  "group_ids": [12],
  "search": "alice"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 插件 slug |
| version | string | 否 | 版本号，不传或传 `latest` 使用最新版本 |
| instance_ids | uint[] | 条件必填 | 显式目标实例 ID（最多 500 个）；与 `select_all=true` 严格二选一 |
| select_all | bool | 条件必填 | 传 `true` 时选择全部匹配实例；全选模式不受 500 个限制 |
| statuses | string[] | 否 | 仅全选模式。可选 `uninstalled/installed/outdated/failed/upgrade_failed/uninstall_failed/uninstall_failed_old`；空数组/省略表示以上稳定状态全集；禁止 `installing/uninstalling` |
| group_ids | uint[] | 否 | 仅全选模式。多个组取并集；`0` 表示未分组用户 |
| search | string | 否 | 仅全选模式。模糊匹配实例名称、实例 ID 或创建人用户名；最长 50 个字符；省略表示不限制 |

- **成功响应：**

```json
{"ok": true, "task_id": 1, "version": "1.0.0", "total": 3}
```

> 下发任务异步执行，通过 `GET /admin/plugins/tasks` 查询进度。

- **失败响应：**
  - `400 {"error": "slug 不能为空"}`
  - `400 {"error": "instance_ids 不能为空"}`
  - `400 {"error": "单次下发实例数不能超过 500"}`
  - `400 {"error": "instance_ids 与 select_all=true 必须二选一"}`
  - `400 {"error": "不能按过渡状态 installing 全选下发"}`
  - `400 {"error": "插件不存在"}`
  - `400 {"error": "没有符合条件的实例，所选实例类型不支持插件安装"}`
  - `409 {"error": "该插件版本正在被其他操作处理，请稍后重试"}`

### `POST /admin/plugins/uninstall`

批量从实例中卸载插件。可通过 `instance_ids` 显式指定实例，或通过 `select_all=true` 按安装状态和用户组选择全部匹配实例。通过 TAT 远程执行插件卸载脚本。与下发操作互斥（同一插件同时只能有一个下发或卸载任务）。

> **后端兜底过滤：** 仅对支持插件能力的 Agent 实例执行卸载；不新增 running 状态限制。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "slug": "system-info-plugin",
  "instance_ids": [1, 2, 3]
}
```

全选示例：

```json
{
  "slug": "system-info-plugin",
  "select_all": true,
  "statuses": ["installed", "uninstall_failed"],
  "group_ids": [12],
  "search": "alice"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 插件 slug |
| instance_ids | uint[] | 条件必填 | 显式目标实例 ID；与 `select_all=true` 严格二选一 |
| select_all | bool | 条件必填 | 传 `true` 时选择全部匹配实例；全选模式不受显式 ID 数量限制 |
| statuses | string[] | 否 | 仅全选模式。可选 `installed/outdated/upgrade_failed/uninstall_failed/uninstall_failed_old`；空数组/省略表示以上五种状态 |
| group_ids | uint[] | 否 | 仅全选模式。多个组取并集；`0` 表示未分组用户 |
| search | string | 否 | 仅全选模式。模糊匹配实例名称、实例 ID 或创建人用户名；最长 50 个字符；省略表示不限制 |

- **成功响应：**

显式 ID 模式：

```json
{"ok": true, "task_id": 15, "message": "已开始卸载流程"}
```

全选模式：

```json
{"ok": true, "task_id": 15, "message": "已开始卸载流程", "total": 37}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 是否成功 |
| task_id | uint | 卸载任务 ID，可通过 `GET /admin/plugins/tasks?type=uninstall` 查询进度 |
| message | string | 提示信息 |
| total | int | 全选模式的目标实例数；显式 ID 模式不返回 |

> 卸载任务异步执行，通过 `GET /admin/plugins/tasks?slug=xxx&type=uninstall` 查询进度。

- **失败响应：**
  - `400 {"error": "slug 不能为空"}`
  - `400 {"error": "instance_ids 不能为空"}`
  - `400 {"error": "instance_ids 与 select_all=true 必须二选一"}`
  - `400 {"error": "statuses 和 group_ids 仅可在 select_all=true 时使用"}`
  - `400 {"error": "不支持的安装状态: uninstalled"}`
  - `400 {"error": "插件不存在"}`
  - `400 {"error": "没有符合条件的实例，所选实例类型不支持插件安装"}`
  - `409 {"error": "该插件正在被其他操作处理，请稍后重试"}`

### P.5 插件包管理

插件包（Plugin Bundle）用于批量管理要预装到新实例的插件。同一时间只能有一个插件包处于启用状态。创建/重装实例时，系统会快照当前启用插件包中的插件，异步安装到 CVM 实例。

> **前置条件：** 创建、删除插件包及更新插件包内插件接口均要求 SMH 服务已启用。

### `GET /admin/plugin-bundles`

查询插件包列表（分页）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "plugin_bundles": [
    {
      "id": 1,
      "name": "通用插件包",
      "plugin_count": 5,
      "enabled": true,
      "created_at": "2026-04-10T10:00:00Z",
      "updated_at": "2026-04-12T10:00:00Z"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1,
  "total_pages": 1
}
```

### `POST /admin/plugin-bundles/create`

创建插件包。同时在 SMH common space 创建对应目录。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 插件包名称（唯一，最长 100 字符） |

- **成功响应：** `{"ok": true, "id": 2}`
- **失败响应：**
  - `400 {"error": "插件包名称不能为空"}`
  - `400 {"error": "插件包名称不能超过 100 个字符"}`
  - `409 {"error": "同名插件包已存在"}`
  - `403 {"error": "SMH 服务未启用，请先在管理后台开通 SMH 服务"}`

### `POST /admin/plugin-bundles/delete`

删除插件包。只能删除未启用的插件包。同时清理 SMH 文件和级联删除插件包内插件记录。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 插件包 ID（Query 参数或 Form 参数） |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `404 {"error": "插件包不存在"}`
  - `409 {"error": "插件包正在生效中，需先禁用"}`

### `POST /admin/plugin-bundles/toggle`

启用/禁用插件包。同一时间只能有一个插件包处于启用状态，启用某个插件包时若已有其他插件包启用则拒绝。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 插件包 ID（Query 参数或 Form 参数） |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `404 {"error": "插件包不存在"}`
  - `409 {"error": "已有其他插件包处于启用状态，请先禁用"}`

### `GET /admin/plugin-bundles/detail`

查询插件包详情，包含插件包内的所有插件列表。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 插件包 ID（Query 参数） |

- **成功响应：**

```json
{
  "plugin_bundle": {
    "id": 1,
    "name": "通用插件包",
    "plugin_count": 5,
    "enabled": true,
    "created_at": "2026-04-10T10:00:00Z",
    "updated_at": "2026-04-12T10:00:00Z"
  },
  "plugins": [
    {
      "id": 1,
      "plugin_bundle_id": 1,
      "name": "系统信息插件",
      "slug": "system-info-plugin",
      "plugin_id": "@anthropic/system-info",
      "version": "1.0.0",
      "source": "enterprise",
      "cos_zip_key": "plugin-bundles/通用插件包/system-info-plugin/system-info-plugin-1.0.0.zip",
      "npm_package": "",
      "install_mode": "smh",
      "kind": "",
      "created_at": "2026-04-10T10:00:00Z",
      "updated_at": "2026-04-12T10:00:00Z"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| plugin_bundle | object | 插件包基本信息 |
| plugins | array | 插件包内插件列表 |
| plugins[].source | string | 插件来源：`enterprise`（企业插件） |
| plugins[].install_mode | string | 安装方式：`smh`（SMH 下载安装）或 `npm`（npm 安装） |
| plugins[].kind | string | 插件类型：`memory`（记忆插件）、`context-engine`（上下文引擎）或空 |

- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `404 {"error": "插件包不存在"}`

### `POST /admin/plugin-bundles/update-plugins`

批量更新插件包内的插件（添加/移除）。添加企业插件时会从 SkillHub space 下载 zip 并上传到 SMH common space；移除插件时会清理 SMH 文件。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 插件包 ID（Query 参数） |

- **请求体：**

```json
{
  "add": [
    {"id": 5, "source": "enterprise"}
  ],
  "remove": [3, 4]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| add | array | 要添加的插件列表 |
| add[].id | uint | 企业插件 ID（`plugins` 表 ID） |
| add[].source | string | 插件来源，目前仅支持 `enterprise` |
| remove | uint[] | 要移除的 `bundle_plugins` 表记录 ID 列表 |

- **成功响应：**

```json
{"ok": true, "plugin_count": 10, "added": 1, "removed": 2}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| plugin_count | int | 更新后插件包内插件总数 |
| added | int | 本次添加的插件数 |
| removed | int | 本次移除的插件数 |

- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `400 {"error": "请求体格式错误"}`
  - `400 {"error": "企业插件 ID=N 不存在"}`
  - `400 {"error": "不支持的来源类型: xxx"}`
  - `404 {"error": "插件包不存在"}`
  - `409 {"error": "插件 slug-version 已存在于该插件包中"}`

### P.6 公共插件收藏

管理员可以收藏公共插件到本地 `public_plugins` 表，收藏后的插件可用于添加到插件包。

### `POST /admin/plugins/favorite`

收藏公共插件。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{
  "name": "系统信息插件",
  "slug": "system-info-plugin",
  "plugin_id": "@anthropic/system-info",
  "version": "1.0.0",
  "description": "获取系统信息",
  "npm_package": ""
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 插件名称 |
| slug | string | 是 | 插件唯一标识 |
| plugin_id | string | 否 | 插件运行时 ID |
| version | string | 否 | 版本号 |
| description | string | 否 | 插件描述 |
| npm_package | string | 否 | npm 包名 |

- **成功响应：** `{"ok": true, "plugin_id": 1}`
- **失败响应：**
  - `400 {"error": "name 和 slug 不能为空"}`
  - `400 {"error": "请求体格式错误"}`
  - `409 {"error": "该插件已收藏"}`

### `POST /admin/plugins/unfavorite`

取消收藏公共插件。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 公共插件 ID（Query 参数或 Form 参数） |

- **成功响应：** `{"ok": true}`
- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `404 {"error": "插件不存在"}`

### `GET /admin/plugins/favorited`

获取已收藏的公共插件列表（分页）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "plugins": [
    {
      "id": 1,
      "name": "系统信息插件",
      "slug": "system-info-plugin",
      "plugin_id": "@anthropic/system-info",
      "version": "1.0.0",
      "description": "获取系统信息",
      "npm_package": "",
      "total_downloads": 0,
      "total_favorites": 0,
      "created_at": "2026-04-10T10:00:00Z",
      "updated_at": "2026-04-10T10:00:00Z"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1,
  "total_pages": 1
}
```

- **失败响应：** 无特殊失败场景

---

## 五、镜像管理（管理员）

### `GET /admin/images`

镜像管理列表页面。

- **权限：** 管理员
- **JSON 响应：**

```json
{
  "images": [
    {
      "id": 1,
      "image_id": "img-xxx",
      "image_name": "云服务器 OpenClaw 镜像",
      "image_type": "...",
      "os_name": "CentOS 7.9 64位",
      "agent_type": "openclaw",
      "agent_version": "2026.3.28",
      "enabled": true,
      "update_notice_enabled": false,
      "public": true,
      "is_legacy": false,
      "can_enable": true,
      "enable_block_reason": ""
    }
  ],
  "enabled_images_by_type": {
    "openclaw": { "id": 1, "image_name": "...", "agent_version": "2026.3.28" }
  },
  "default_agent_type": "openclaw"
}
```

字段说明：

| 字段 | 说明 |
|------|------|
| agent_type | 智能体类型（openclaw/hermes/lightclawace） |
| agent_version | 智能体版本号 |
| update_notice_enabled | 是否提示该官方镜像有更新 |
| is_legacy | 是否存量镜像（无类型或无版本） |
| can_enable | 是否可启用 |
| enable_block_reason | 不可启用原因（如"请先设置 Agent 版本后再启用"） |
| agent_name | 智能体名称，仅公共镜像（`public=true`）返回，当前固定为 `"OpenClaw"` |
| enabled_images_by_type | 各类型当前启用镜像 |
| default_agent_type | 用户端首选智能体类型 |

### `GET /admin/images/cloud`

获取腾讯云私有镜像列表（排除已导入到本地的镜像），用于导入镜像时的下拉选择。同时会补充查询内置候选镜像列表（`CandidateImages`）中尚未导入的镜像（公共镜像不在私有镜像列表中，需单独查询）。

- **权限：** 管理员
- **响应：** 始终 JSON

成功时返回镜像数组，属于候选镜像列表的镜像额外包含 `"public": true`：

```json
[
  {"imageId": "img-xxx", "imageName": "镜像名", "osName": "Ubuntu 22.04", "imageState": "NORMAL"},
  {"imageId": "img-idzg74s9", "imageName": "OpenClaw on Ubuntu 24.04", "osName": "Ubuntu 24.04", "imageState": "NORMAL", "public": true}
]
```

失败时返回错误信息：

- `500 {"error": "未配置腾讯云 CVM 凭证"}`
- `500 {"error": "查询镜像失败: ..."}`

### `POST /admin/images/import`

从腾讯云导入镜像信息。通过 `DescribeImages` API 查询镜像详情并保存到数据库。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| image_id | string | 是 | 腾讯云镜像 ID（如 img-xxx） |
| image_name | string | 否 | 自定义镜像名称，留空则使用腾讯云返回的名称 |
| agent_type | string | 是 | 智能体类型：内置 `openclaw`/`hermes`/`lightclawace`，或已存在的自定义类型 code |
| agent_version | string | 视情况 | 内置类型必填（OpenClaw: `YYYY.M.D`，Hermes/LightclawACE: `X.Y.Z`）；**自定义类型可为空**（自定义类型无版本概念） |

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`400 {"error": "agent_type 不能为空"}` / `400 {"error": "agent_version 不能为空"}` / `404 {"error": "未找到镜像: img-xxx"}` / `500 {"error": "..."}`

### `POST /admin/images/delete`

删除镜像记录（仅删除本地数据库记录，不影响腾讯云镜像）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 镜像记录 ID（Query 参数） |

- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`404 {"error": "镜像不存在"}`

### `POST /admin/images/enable`

切换镜像启用状态。同一 agent_type 下最多一个镜像处于启用状态，启用某个镜像会自动禁用同类型其他已启用的镜像。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 镜像记录 ID（Query 参数） |

- **启用约束：**
  - 内置类型镜像若 `agent_type` 不为空但 `agent_version` 为空，不可启用，返回 `400 {"error": "请先设置 Agent 版本后再启用"}`（自定义类型无此限制，自定义类型镜像 `agent_version` 允许为空）
  - 无效 agent_type 的镜像不可启用
- **禁用约束：**
  - 首选类型（default_agent_type）的唯一启用镜像不可禁用，返回 `400 {"error": "该类型为用户端首选，不可取消启用"}`
- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`400 {"error": "..."}` / `404 {"error": "镜像不存在"}`

### `POST /admin/images/update`

更新镜像的智能体类型和版本。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 镜像记录 ID |
| agent_type | string | 否 | 新的智能体类型 |
| agent_version | string | 否 | 新的版本号 |

- **公共镜像保护：** 若目标镜像命中内置候选镜像列表（`hcommon.IsCandidateImage(img.ImageId)`，即腾讯云公共镜像/官方镜像），**拒绝编辑**，其类型和版本由系统维护，不允许管理员手动覆盖。
- **响应：** 始终 JSON
  - 成功：`{"ok": true}`
  - 公共镜像保护：`403 {"error": "公共镜像不支持编辑，其类型和版本由系统维护"}`
  - 失败：`400 {"error": "..."}` / `404 {"error": "镜像不存在"}`

### `POST /admin/images/history/publish`

发布官方镜像更新历史。如果发布后该镜像的最新历史发生变化，会同步更新所有租户中该镜像的当前版本，并清除该镜像的通知提示状态。

- **权限：** 仅 `admin-token`（`Authorization: Bearer <admin-token>`）
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| image_id | string | 是 | 官方镜像 ID |
| agent_version | string | 是 | 发布后的 Agent 版本号，按镜像对应类型校验格式 |
| published_at | string | 是 | 发布日期/时间，支持 RFC3339 或 `YYYY-MM-DD` |

- **响应：** 始终 JSON
  - 成功：`{"ok": true, "id": 123, "latest_changed": true, "updated_images": 3}`
  - 失败：`400 {"error": "仅官方镜像支持发布更新通知"}` / `403 {"error": "仅 admin-token 可以发布镜像更新动态"}`

### `POST /admin/images/history/update`

修改指定官方镜像更新历史。如果修改后该镜像的最新历史发生变化，会同步更新所有租户中该镜像的当前版本，并清除该镜像的通知提示状态。

- **权限：** 仅 `admin-token`（`Authorization: Bearer <admin-token>`）
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 历史记录 ID |
| agent_version | string | 否 | 新 Agent 版本号 |
| published_at | string | 否 | 发布日期/时间，支持 RFC3339 或 `YYYY-MM-DD` |

`agent_version` 和 `published_at` 至少填写一个。

- **响应：** 始终 JSON
  - 成功：`{"ok": true, "latest_changed": true, "updated_images": 3}`

### `POST /admin/images/history/delete`

删除官方镜像更新历史。默认使用软删除；默认查询和用户侧提示不会再使用已删除记录。传 `hard=true` 时物理删除记录。如果删除后该镜像的最新历史发生变化，会同步更新所有租户中该镜像的当前版本，并清除该镜像的通知提示状态；如果删除后没有剩余历史，则回退到内置初始版本。

- **权限：** 仅 `admin-token`（`Authorization: Bearer <admin-token>`）
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 否 | 历史记录 ID |
| image_id | string | 否 | 官方镜像 ID；未传 `id` 时删除该镜像最新一条未删除历史 |
| hard | bool | 否 | 是否物理删除，默认 `false` |

`id` 和 `image_id` 至少填写一个；同时填写时优先使用 `id`。`hard=true` 且传 `id` 时，可物理删除已软删除记录。

- **响应：** 始终 JSON
  - 成功：`{"ok": true, "deleted_id": 123, "hard": false, "latest_changed": true, "updated_images": 3}`

### `POST /admin/images/history/restore`

启用已删除的官方镜像更新历史。如果启用后该镜像的最新历史发生变化，会同步更新所有租户中该镜像的当前版本，并清除该镜像的通知提示状态。

- **权限：** 仅 `admin-token`（`Authorization: Bearer <admin-token>`）
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 历史记录 ID |

- **响应：** 始终 JSON
  - 成功：`{"ok": true, "restored_id": 123, "latest_changed": true, "updated_images": 3}`

### `POST /admin/images/update-notice`

开启或关闭当前租户内某个官方镜像的更新提示。

- **权限：** 管理员
- **Content-Type：** `application/json` 或 `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| image_id | string | 是 | 官方镜像 ID |
| enabled | bool | 是 | 是否开启提示 |

- **响应：** 始终 JSON
  - 成功：`{"ok": true}`
  - 失败：`400 {"error": "仅官方镜像支持更新通知开关"}` / `404 {"error": "镜像不存在"}`

### `GET /admin/images/history`

获取官方镜像更新历史。

- **权限：** 管理员
- **参数（Query String，均为可选）：**

| 参数 | 类型 | 说明 |
|------|------|------|
| image_id | string | 按官方镜像 ID 过滤 |
| agent_type | string | 按智能体类型过滤 |
| include_deleted | bool | 是否包含已删除记录，默认 `false` |
| enabled_only | bool | 是否只返回当前租户有效启用镜像的历史，默认 `false` |
| page | int | 页码，默认 1 |
| page_size | int | 每页条数，默认 20，最大 100 |

- **响应：** 始终 JSON

```json
{
  "items": [
    {
      "id": 123,
      "image_id": "img-idzg74s9",
      "image_name": "OpenClaw on Ubuntu 24.04",
      "agent_type": "openclaw",
      "agent_version": "2026.5.25",
      "published_at": "2026-05-25T00:00:00Z",
      "created_at": "2026-05-25T00:00:00Z",
      "updated_at": "2026-05-25T00:00:00Z",
      "can_set_notice": true,
      "update_notice_enabled": true,
      "image_enabled": true,
      "outdated_running_instance_count": 3
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

`can_set_notice=true` 表示该记录的 `published_at` 是同一 `image_id` 下未删除历史中的最新发布时间，可用于展示“设置更新通知”操作。

`update_notice_enabled=true` 表示当前租户已对该 `image_id` 开启更新通知；该字段只会在同一 `image_id` 的最新未删除历史记录上为 `true`，旧历史记录始终为 `false`。

`image_enabled=true` 表示当前租户中该 `image_id` 对应镜像处于有效启用状态。

`outdated_running_instance_count` 表示当前租户中同一 `agent_type` 下正在运行且 Agent 已就绪、但 `agent_version` 与该最新历史版本不一致的实例数量；该字段只在每个 `image_id` 的最新未删除历史记录上返回，旧历史记录不返回该字段。

### `POST /admin/images/set-default-type`

设置用户端首选智能体类型。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| agent_type | string | 是 | 智能体类型 |

- **约束：** 首选类型必须有已启用镜像
- **响应：** 始终 JSON
  - 成功：`{"ok": true}`
  - 失败：`400 {"error": "该类型（OpenClaw）没有已启用镜像，无法设为首选"}`

### `GET /admin/agent-types`

获取所有智能体类型列表（只读），包含各类型的功能支持情况、启用镜像信息和镜像统计。

- **权限：** 管理员
- **响应：** 始终 JSON

```json
{
  "agent_types": [
    {
      "code": "openclaw",
      "name": "OpenClaw",
      "description": "功能最完整的智能体类型",
      "is_builtin": true,
      "compatible_with": "",
      "supports_role": true,
      "supports_model": true,
      "supports_channel": true,
      "supports_skill": true,
      "supports_plugin": true,
      "supports_chatbot": true,
      "supports_smh": true,
      "supports_memory": true,
      "supports_reinstall": true,
      "supports_upgrade": true,
      "sort_order": 1,
      "enabled": true,
      "has_enabled_image": true,
      "enabled_image": { "id": 1, "image_name": "...", "agent_version": "2026.3.28" },
      "is_default": true,
      "user_selectable": true,
      "image_count": 3,
      "instance_count": 12
    },
    {
      "code": "my-openclaw-fork",
      "name": "my-openclaw-fork",
      "description": "自定义智能体类型，兼容 OpenClaw",
      "is_builtin": false,
      "compatible_with": "openclaw",
      "supports_role": true,
      "supports_model": true,
      "supports_channel": true,
      "supports_skill": true,
      "supports_plugin": true,
      "supports_chatbot": true,
      "supports_smh": true,
      "supports_memory": true,
      "supports_reinstall": true,
      "supports_upgrade": true,
      "sort_order": 1001,
      "enabled": true,
      "has_enabled_image": false,
      "is_default": false,
      "user_selectable": false,
      "image_count": 0,
      "instance_count": 0
    },
    {
      "code": "lone-custom",
      "name": "lone-custom",
      "description": "自定义智能体类型，不兼容内置类型，仅支持最小操作集",
      "is_builtin": false,
      "supports_role": false,
      "supports_model": false,
      "supports_channel": false,
      "supports_skill": false,
      "supports_plugin": false,
      "supports_chatbot": false,
      "supports_smh": false,
      "supports_memory": false,
      "supports_reinstall": false,
      "supports_upgrade": false,
      "sort_order": 1002,
      "enabled": true,
      "has_enabled_image": false,
      "is_default": false,
      "user_selectable": false,
      "image_count": 0,
      "instance_count": 0
    }
  ],
  "default_agent_type": "openclaw"
}
```

> **能力字段语义：**
> - 内置类型（`is_builtin=true`）的能力位由后端硬编码（`model/agent_type.go::agentTypesMap`）。
> - 自定义类型若声明 `compatible_with`（仅允许 `openclaw`/`hermes`/`lightclawace`），其能力位、脚本分派、通道白名单等同于该兼容目标，但**镜像不复用**——必须为该自定义类型自己导入并启用镜像。
> - 自定义类型未声明 `compatible_with` 时，所有能力位为 `false`，仅支持最小操作集（创建/删除/启停实例等）。
> - 声明了 `compatible_with` 的自定义类型会参与版本同步（使用兼容目标的脚本）；未声明 `compatible_with` 的自定义类型不参与版本同步任务。
> - 该接口响应中暴露的能力位为 `supports_role/model/channel/skill/plugin/chatbot/smh/memory/reinstall/upgrade`。其他后端能力位（`supports_browser_vnc/approve/default_model_injection/api_gateway`）仅供后端内部分派使用，未对外下发。
> - `instance_count` 表示当前租户中该 `agent_type` 下未删除的实例数量；历史空 `agent_type` 实例按 `openclaw` 统计。

### `POST /admin/agent-types/enabled`

启用、禁用或切换用户端可选智能体类型。内部更新 `site_configs.disabled_agent_types`，不改变镜像启用状态；被禁用类型仍会在管理端列表中返回，但不会出现在用户端 `/openclaw/agent-types` 中，也不能用于创建实例。

- **权限：** 管理员
- **参数：** Query string 或表单均可

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| agent_type | string | 是 | 智能体类型 code |
| enabled | boolean | 否 | 直接设置启用状态；`true`=启用，`false`=禁用 |
| toggle | boolean | 否 | 传 `true` 时切换当前启用状态 |

示例：`POST /admin/agent-types/enabled?agent_type=hermes&enabled=false`

- **约束：** `enabled` 和 `toggle` 二者互斥，必须且只能传一个；`toggle=false` 无意义并返回 400；当前 `default_agent_type` 不允许禁用。
- **响应：** 始终 JSON
  - 成功：`{"ok": true, "agent_type": "hermes", "operation": "disable", "previous_enabled": true, "enabled": false, "disabled_agent_types": ["hermes"], "default_agent_type": "openclaw"}`
  - 失败：`400 {"error": "enabled 和 toggle 不能同时传"}` / `400 {"error": "该类型是用户端首选，不可禁用"}`

### `POST /admin/agent-types/create`

创建自定义智能体类型。自定义类型只有一个 `name`，可选择兼容一个内置类型；兼容只影响能力和脚本分派，不复用内置类型镜像。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 自定义类型名称，不能为空，最多 20 个字符，不能包含首尾空格，不能与内置类型或已有自定义类型重复 |
| compatible_with | string | 否 | 兼容的内置类型：`openclaw` / `hermes` / `lightclawace`；为空表示不兼容 |

- **响应：** 始终 JSON
  - 成功：`{"ok": true}`
  - 失败：`400 {"error": "名称不能为空"}` / `400 {"error": "名称不能包含首尾空格"}` / `400 {"error": "名称不能超过 20 个字符"}` / `400 {"error": "名称不能与内置智能体类型重复: <name>"}` / `400 {"error": "兼容类型必须是内置智能体类型"}` / `400 {"error": "自定义智能体类型已存在: <name>"}` / `500 {"error": "..."}`

> **语义说明：**
> - `name` 严格按用户输入存储，不做大小写或空白归一化；冲突判定使用精确匹配。
> - `compatible_with` 仅影响**能力位**和**脚本分派**：兼容类型的实例继承兼容目标的所有 `Supports*` 标志、通道白名单和脚本映射。
> - `compatible_with` **不复用镜像**：自定义类型必须为自身导入并启用镜像，才能被用户在 `/openclaw/create` 创建实例时选中。
> - 自定义类型没有版本概念：导入镜像时 `agent_version` 可为空，且不参与 `version_sync` 任务。

### `POST /admin/agent-types/delete`

删除自定义智能体类型。内置类型不能删除；不能是当前首选类型，且该类型下不能存在实例。该类型被禁用，或该类型下没有启用镜像时才允许删除；删除时会自动删除该类型下的所有本地镜像记录。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 自定义类型名称 |

- **响应：** 始终 JSON
  - 成功：`{"ok": true}`
  - 失败：`400 {"error": "name 不能为空"}` / `400 {"error": "内置智能体类型不能删除"}` / `404 {"error": "自定义智能体类型不存在: <name>"}` / `400 {"error": "请先禁用该智能体类型或取消启用该类型下的镜像后再删除"}` / `400 {"error": "该类型下存在实例，不能删除"}` / `400 {"error": "该类型是用户端首选类型，不能删除"}` / `500 {"error": "..."}`

> **删除语义：** 先删除该类型下的本地镜像记录，再通过 `Unscoped().Delete` 物理删除自定义类型；删除后 `name` 可被重新创建（不保留软删除占位）。

### `GET /admin/local-agent-types` 🆕

列举本期支持的**本地 agent 类型**（一期 codebuddy / workbuddy）。与 `/admin/agent-types` **故意分开**：后者面向管控 hatchery CVM 实例的 agent_type（含 30+ 个 `supports_*` 能力位 / 镜像信息 / 启用状态）；本接口面向本地 agent type，它们没有镜像概念、不走 hatchery 创实例流程、能力位全不适用，独立接口 schema 更干净。

- **权限：** 管理员
- **入参：** 无
- **响应：** `200 application/json`

```json
{
  "local_agent_types": [
    {
      "code": "codebuddy",
      "name": "CodeBuddy",
      "description": "本机代码助手 agent，由用户在本地安装并运维"
    },
    {
      "code": "workbuddy",
      "name": "WorkBuddy",
      "description": "本机工作流 agent，由用户在本地安装并运维"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `local_agent_types[].code` | string | reporter 上报与 DB 落库的原始值（`instances.agent_type`） |
| `local_agent_types[].name` | string | 展示名 |
| `local_agent_types[].description` | string | 简介，供管控页 tooltip / 列表副标题使用 |

**顺序稳定**：数组顺序与后端 `localAgentTypes` 同源，与 reporter 接口 `agent_type` 校验白名单同一数据源。二期扩类型只需改一处，本接口 + reporter 校验同时生效。

**不带 `instance_count` 等聚合字段**：本接口语义是「我支持哪些 type」，不是「我有哪些实例」。前端要表示「某 type 当前接入 N 台」这类 dashboard 数字，应调 `GET /admin/instances?source=local&agent_type=<code>` 自行 group by。

- **错误响应：**
  - `401 {"error": "未登录"}`
  - `405 {"error": "method not allowed"}` — 非 GET

### `GET /local-agent/get-config` 🆕

本地 agent（reporter / agent 自身）在需要往 CLS 公网上报日志/指标/Trace 前，主动拉取接入配置。CVM 实例的 CLS 配置由 hatchery 经 TAT 脚本 + CAM Role 临时凭证下发；本地 agent 不在 VPC 内、无法用 CAM Role，因此本接口直接返回**公网 endpoint + 永久 AK/SK + topic**，本地 agent 用这些凭据直连 CLS 公网 endpoint 上报。

本期 GET 只读查询，不写实例状态。

- **权限：** 登录用户（且通过本地 Agent 两层白名单，见 `POST /local-agent/report`）
- **Content-Type：** `application/json`
- **Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config_type | string | 否 | 配置类型筛选。一期仅支持 `cls`；不传则返回全量配置（当前仅 `cls`）。传不支持的类型返回 400 |

- **响应：** `200 application/json`

```json
{
  "cls": {
    "endpoint": "ap-guangzhou.cls.tencentcs.com",
    "topic_id": "f7e11f3b-668b-4756-ab1e-2c5eac5b83c6",
    "secret_id": "AKIDxx…xxxx",
    "secret_key": "xxxxxxxxxxxxxxxxxxxxxxxx",
    "user_id": 12345,
    "user_name": "alexwhwang",
    "install_cmd": "npm install -g tencentcloud-cls-sdk-codebuddy-test --registry https://mirrors.tencentyun.com/npm/",
    "run_cmd": "cls-codebuddy setup --endpoint ap-guangzhou.cls.tencentcs.com --topic-id f7e11f3b-668b-4756-ab1e-2c5eac5b83c6 --secret-id AKIDxx…xxxx --secret-key xxxx --service-name ${local_agent_id} --user-name alexwhwang --user-id 12345",
    "update_cmd": "npm install -g tencentcloud-cls-sdk-codebuddy-test --registry https://mirrors.tencentyun.com/npm/",
    "uninstall_cmd": "cls-codebuddy uninstall-all"
  }
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `cls` | object | CLS 配置明细对象（key 即配置类型） |
| `cls.endpoint` | string | **公网** CLS 接入域名，格式 `<region>.cls.tencentcs.com`。与 CVM 实例内网域名 `<region>.cls.tencentyun.com` 区分 |
| `cls.topic_id` | string | CLS 日志主题 ID（如 `clawpro-topic-xxx`）。**实时查询** CLS `OpenClawService` 返回，**不落库**；CLS 服务未开通则 4xx |
| `cls.secret_id` | string | 用于本地 agent 直连 CLS 公网上报的 AK |
| `cls.secret_key` | string | 同上，SK。敏感字段，仅在本接口按需返回，不出现在列表/日志明文 |
| `cls.user_id` | uint | 当前调用用户的 ID，本地 agent 用于上报时标记归属 |
| `cls.user_name` | string | 当前调用用户的用户名，本地 agent 用于上报时标记归属 |
| `cls.install_cmd` | string | 本地 agent 安装命令（固定值：`npm install -g tencentcloud-cls-sdk-codebuddy-test --registry https://mirrors.tencentyun.com/npm/`） |
| `cls.run_cmd` | string | 本地 agent 启动/配置命令（固定值：`cls-codebuddy setup --endpoint ${endpoint} --topic-id ${topic_id} --secret-id ${secret_id} --secret-key ${secret_key} --service-name ${local_agent_id} --user-name ${user_name} --user-id ${user_id}`，原样返回，占位符由本地 agent 自行替换） |
| `cls.update_cmd` | string | 本地 agent 更新命令（固定值：`npm install -g tencentcloud-cls-sdk-codebuddy-test --registry https://mirrors.tencentyun.com/npm/`） |
| `cls.uninstall_cmd` | string | 本地 agent 卸载命令（固定值：`cls-codebuddy uninstall-all`） |

- **错误响应：**
  - `401 {"error": "未登录"}`
  - `403 {"error": "本地 Agent 功能未开放"}` — 未通过两层白名单（跨租户白名单或租户 `LocalAgentEnabled`）
  - `400 {"error": "config_type 无效: <type>"}` — config_type 传了但不支持（一期仅 cls）
  - `400 {"error": "CLS 服务未开启，请先开启 CLS 服务"}` — CLS 服务未开通或 topic 为空
  - `500 {"error": "CLS 凭据未配置，请联系管理员"}` — 凭据表无当前租户数据或读取失败
  - `405 {"error": "method not allowed"}` — 非 GET

> **安全约束：** `secret_key` 仅在响应里出现，禁止写入任何 access_log / 普通业务日志。凭据按租户隔离，不同租户持有各自的 CLS AK/SK，互不可见。

### `GET /openclaw/agent-types`

获取用户可选的智能体类型。

- **权限：** 登录用户
- **响应：** 始终 JSON

```json
{
  "agent_types": [
    {
      "code": "openclaw",
      "name": "OpenClaw",
      "description": "功能最完整的智能体类型",
      "is_builtin": true,
      "is_default": true
    },
    {
      "code": "hermes",
      "name": "Hermes",
      "description": "轻量级智能体，支持终端和 WebUI",
      "is_builtin": true,
      "is_default": false
    },
    {
      "code": "my-openclaw-fork",
      "name": "my-openclaw-fork",
      "description": "自定义智能体类型，兼容 OpenClaw",
      "is_builtin": false,
      "compatible_with": "openclaw",
      "is_default": false
    }
  ],
  "default_agent_type": "openclaw"
}
```
> **过滤规则：** 仅返回**该类型自身**有启用镜像、未被 `disabled_agent_types` 禁用、且当前用户命中应用范围的条目。自定义类型即便声明了 `compatible_with`，也不会复用兼容内置类型的镜像；必须为该自定义类型导入并启用镜像后才会出现。

### `GET /openclaw/images/update-notices`

获取当前登录用户可见的官方镜像更新提示列表。前端可据此判断是否提醒用户更新镜像。

- **权限：** 登录用户
- **响应：** 始终 JSON
- **过滤规则：**
  - 仅返回当前租户内 `update_notice_enabled=true` 的镜像；
  - 镜像必须是官方镜像；
  - 仅返回当前用户有权使用的 `agent_type`。若镜像类型被配置为按组可见，则用户属于对应组或其子组时才返回；未分组用户只能看到全局可见的类型。

```json
{
  "items": [
    {
      "image_id": "img-idzg74s9",
      "image_name": "OpenClaw on Ubuntu 24.04",
      "agent_type": "openclaw",
      "agent_version": "2026.5.25",
      "published_at": "2026-05-25T00:00:00Z"
    }
  ]
}
```

### `POST /admin/instances/create`

管理员为指定用户创建 CVM Agent。实例创建、配额、镜像、网络、安全组、角色、技能/插件初始化等主流程与用户端 `POST /openclaw/create` 共用；本接口额外支持在创建前指定目标用户、目标分组，以及初始模型、多个手动通道和追加 public/enterprise 技能。

- **权限：** 管理员；已登录非管理员返回 403，未登录返回 401
- **Content-Type：** `application/json`
- **请求体上限：** 1 MiB
- **JSON 约束：** 未知字段、多 JSON 值或尾随非空内容会被拒绝
- **审计 action：** `instance_admin_create`
- **请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | uint | 是 | Agent 所属用户 ID；用户必须存在且未软删除 |
| name | string | 是 | 实例名称；去除首尾空白后长度为 1～128 字节 |
| group_id | uint | 条件必填 | 目标用户没有有效分组时自动使用 0；只有一个有效分组时自动选择；存在多个有效分组时必须指定。指定值必须属于目标用户，且分组不能处于 `to_be_deleted` |
| agent_type | string | 是 | 智能体类型；支持内置类型和已注册的自定义类型，按类型能力校验 models/channels/skills |
| role_id | uint | 否 | 初始角色 ID；沿用普通创建流程的角色校验与初始化逻辑 |
| disk_type | string | 否 | 系统盘类型；沿用普通创建接口支持的磁盘类型校验 |
| tags | object[] | 否 | 自定义 CVM 标签列表；可传多个。与系统默认标签同 key 时本次自定义值优先 |
| tags[].key | string | 是 | 标签键；必须已存在于 ClawPro 绑定的腾讯云账号中，具体规则由 CVM `RunInstances` 校验 |
| tags[].value | string | 是 | 标签值；具体规则由 CVM `RunInstances` 校验 |
| models | object | 否 | 初始模型配置；传入后替换站点默认模型，不传则沿用站点默认模型注入逻辑。所有支持模型配置的类型均可指定单个模型；只有 OpenClaw 兼容运行时支持 fallbacks |
| models.primary | uint | models 存在时必填 | 初始激活模型 ID；模型必须 enabled、visible 且对目标分组可见。OpenClaw 内部作为 primary；Hermes / LightclawACE 作为唯一激活模型 |
| models.fallbacks | uint[] | 否 | 仅 OpenClaw 兼容运行时可用，按数组顺序配置；不能为 0，不能与 primary 或其他 fallback 重复。OpenClaw 3.28.x 镜像不支持 |
| channels | object[] | 否 | 初始手动通道列表；同一请求中 channel 不能重复 |
| channels[].channel | string | 是 | 通道 ID；必须在当前站点范围内、由 agent_type 支持、已启用且对目标分组可见 |
| channels[].config | object | 是 | `string -> string` 的完整手动配置；键和值均不能为空，且必须包含该通道定义的所有必填参数 |
| skills | object[] | 否 | 追加安装的技能；不会替换角色技能或技能包技能 |
| skills[].source | string | 否 | `public` 或 `enterprise`；省略时与用户手动添加一致，默认为 `public` |
| skills[].slug | string | 是 | 技能 slug；public 直接按公共源安装，enterprise 必须存在且对目标分组可见 |
| skills[].version | string | 否 | 仅 enterprise 使用；精确版本，省略或空字符串时选择该 slug 的最新语义版本 |

> **Resource Config Policy：** 本接口与用户创建共用站点/最近用户组资源策略、镜像与最终系统盘容量校验、机型可售性预检及安全组前 fail-closed 行为。当前 JSON schema **不接受**直接 `resource_config` 字段；管理员创建若要改变本次资源配置，应先更新站点/组策略，或使用已支持的兼容 `disk_type`。未知顶层字段（包括 `resource_config`）会因本接口严格 JSON 解码返回 400。

请求示例：

```json
{
  "user_id": 1001,
  "name": "sales-agent-01",
  "group_id": 12,
  "agent_type": "openclaw",
  "role_id": 3,
  "disk_type": "CLOUD_BSSD",
  "tags": [
    {"key": "env", "value": "prod"},
    {"key": "team", "value": "sales"}
  ],
  "models": {
    "primary": 10,
    "fallbacks": [11, 12]
  },
  "channels": [
    {
      "channel": "feishu",
      "config": {
        "app_id": "cli_xxx",
        "app_secret": "secret"
      }
    },
    {
      "channel": "wecom",
      "config": {
        "bot_id": "bot-xxx",
        "secret": "secret"
      }
    }
  ],
  "skills": [
    {
      "source": "enterprise",
      "slug": "sales-assistant",
      "version": "1.2.0"
    }
  ]
}
```

- **成功响应：** HTTP 200。`instance_id` 是腾讯云 CVM 实例 ID。响应不额外返回 Hatchery 数据库主键，也不回显 presets：预设已在创建前完成校验，但异步下发没有可查询的终态，返回固定的 `queued` 没有可操作意义。

```json
{
  "ok": true,
  "instance_id": "ins-abcdefgh"
}
```

> HTTP 200 只表示 CVM 创建成功，且非空预设已安排在当前进程中等待 Agent ready；不表示模型、通道或技能已经下发完成。创建期预设不提供任务查询、自动重试或服务重启恢复。

#### 预设处理语义

1. 用户、分组、模型、通道和技能均在申请 CVM 前完成存在性、状态、能力和可见性校验；校验失败不会分配 CVM。
2. models 在 Agent ready 后逐个执行。Hermes / LightclawACE 等单模型运行时只允许 `primary` 一个初始模型；OpenClaw 兼容运行时可按 `primary -> fallbacks[]` 顺序配置，第一个成功模型成为 primary，后续成功模型成为 fallback。OpenClaw 3.28.x 不接受 fallbacks。单次下发失败会回滚该模型绑定并停止后续模型。指定 models 时不再注入站点默认模型；未指定时普通默认模型行为不变。
3. channels 在 Agent ready 后按请求顺序逐个执行，与用户手动 `set-channel` 共用校验、参数组装和脚本下发逻辑。配置只保存在当前 goroutine 内存中，不写数据库；失败不重试，进程退出时尚未执行的配置会丢失。
4. 通道凭据不会出现在响应、审计详情或结构化日志中。服务不会记录本接口的请求 body；调用方也不得记录完整请求体。
5. skills 在 Agent ready 后逐个执行，与用户手动 `add-skill` 共用来源语义：source 省略时为 public；public 调用公共源安装脚本，enterprise 查询可见企业技能并从 SMH 安装。两者均不创建安装任务、失败不重试。角色技能和技能包技能仍沿用原有持久化安装流程。
6. 创建期预设失败不会删除已经创建的 Agent。模型失败会发送模型配置失败通知；通道和追加技能失败只记录不含凭据的标识信息，管理员可在实例 ready 后按普通手动接口重新设置。
7. tags 与用户端创建接口共用合并逻辑：自定义标签与系统默认标签合并，同 key 时自定义值优先；请求内重复 key 交由腾讯云校验，不在本地静默去重。请求传入至少一个自定义标签时，腾讯云标签校验错误返回 400；请求未传自定义标签时保持原有 500 行为。标签只随本次 CVM 创建下发，不写入独立的请求标签字段。

#### 错误响应

| HTTP 状态 | 场景 |
|-----------|------|
| 400 | JSON 非法/超限/含未知字段；必填参数缺失；目标用户、分组、模型、通道或技能无效；重复模型/通道；预设能力不受 agent_type 支持；请求传入至少一个自定义标签时发生腾讯云标签校验错误 |
| 401 | 未登录 |
| 403 | 已登录但不是管理员；目标用户或目标分组实例配额已满 |
| 405 | 请求方法不是 POST |
| 500 | CVM、网络、安全组或数据库等创建流程失败；请求未传自定义标签时发生腾讯云标签校验错误 |

---


### `GET /admin/instances`

OpenClaw 实例监控页面，分页列出所有用户的实例，并内联返回每个实例的实时 OpenClaw 状态、操作进度和管控端可用操作。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码（默认 1） |
| page_size | int | 否 | 每页条数（默认 20，最大 1000） |
| status | string | 否 | 按 OpenClaw 状态筛选。不传=全部；支持 `running`/`stopped`/`other`（展开为所有非 running/stopped 状态）或具体状态值。本地实例（`source=local`）也只会出现 `running`/`stopped`，与 CVM 实例共用同一套筛选语义 |
| keyword | string | 否 | 模糊搜索实例名称、实例 ID、用户名 |
| creator | string | 否 | 按创建人精确筛选 |
| date_from | string | 否 | 创建时间起始（`YYYY-MM-DD`，含当天），不传不限 |
| date_to | string | 否 | 创建时间截止（`YYYY-MM-DD`，含当天），不传不限 |
| department_id | string | 否 | 按部门 ID 筛选（筛选该部门下用户的实例，兼容旧参数名 `department`） |
| agent_type | string | 否 | 按智能体类型过滤，支持逗号分隔多值（OR），如 `openclaw` 或 `openclaw,hermes`，不传不限 |
| cvm_instance_type | string | 否 | 按缓存的 CVM 规格精确过滤，支持逗号分隔多值（OR），如 `Ai2.MEDIUM4,Ai2.LARGE8` |
| system_disk_size | string | 否 | 按缓存的系统盘容量（GiB）精确过滤；支持逗号分隔正整数多值（OR），如 `50,100` |
| system_disk_size_lt | int64 | 否 | 按缓存的系统盘容量严格小于指定正整数过滤，如 `system_disk_size_lt=50` 不包含 50 GiB |
| system_disk_size_gt | int64 | 否 | 按缓存的系统盘容量严格大于指定正整数过滤，如 `system_disk_size_gt=50` 不包含 50 GiB |
| ids | string | 否 | 按实例主键 ID 精确过滤，逗号分隔；最多 1000 项 |
| instance_ids | string | 否 | 按 CVM 实例 ID 精确过滤（如 `ins-xxx,ins-yyy`），逗号分隔；最多 1000 项 |
| tag_keys | string | 否 | 按标签键过滤：实例需带任一指定 key（多键 OR），逗号分隔；最多 100 项。与 `tag_key`+`tag_values` 互斥，若同时传则被忽略 |
| tag_key | string | 否 | 标签键，需配合 `tag_values` 使用；与 `tag_keys` 互斥（同时传时优先采用本组） |
| tag_values | string | 否 | 单一 `tag_key` 下要匹配的多个值（OR），逗号分隔；最多 100 项；只传 `tag_key` 不传 `tag_values` 不会触发标签过滤 |

> 所有筛选条件为 AND 关系。时间筛选按实例创建时间（`created_at`）过滤，支持只传 `date_from` 或只传 `date_to`。
>
> **标签过滤实现细节**：标签数据来自 CVM `DescribeInstances` 实时返回（与列表接口本就需要的调用复用，不会增加额外的腾讯云 API 调用），在内存中过滤。CVM 调用失败标记为 `API_ERROR` 的实例没有可信标签信息，启用标签过滤时会被排除。可先通过 `GET /api/tags/keys` 与 `GET /api/tags/values?key=xxx` 拉取下拉选项。
>
> **资源过滤实现细节**：`cvm_instance_type` 与系统盘容量条件在 SQL 查询阶段执行，影响列表、`total`、分页和统计。`system_disk_size`、`system_disk_size_lt`、`system_disk_size_gt` 三者互斥，同时提供、包含非正整数或非数字时返回 400；与其他筛选条件取 AND。

- **JSON 响应：**

```json
{
  "instances": [
    {
      "ID": 42,
      "Name": "my-instance",
      "Username": "zhangsan",
      "agent_type": "openclaw",
      "agent_version": "2026.4.11",
      "GroupID": 3,
      "GroupFullPath": "根组/研发组",
      "department": "市场二组",
      "department_path": "OpenClaw企业版体验/新组/市场组/市场二组",
      "departments": [
        {
          "department_id": "d4",
          "department_name": "市场二组",
          "department_parent_id": "d3",
          "is_main_department": true,
          "department_path": "OpenClaw企业版体验/新组/市场组/市场二组"
        }
      ],
      "status": "loading",
      "label": "加载中",
      "tooltip": "加载中，请稍候",
      "actions": ["monitor"],
      "transient": true,
      "current_operation": "reboot",
      "current_operation_state": "processing",
      "current_operation_updated_at": "2026-03-26T10:00:00Z",
      "instance_charge_type": "PREPAID",
      "agent_version": "2026.3.28",
      "agent_type": "openclaw",
      "plugin_version_status": [
        {
          "slug": "openclaw-qqbot",
          "display_name": "QQ 机器人",
          "installed_version": "1.6.7",
          "latest_version": "1.7.0",
          "need_upgrade": true,
          "not_installed": false
        }
      ],
      "version_fetched_at": "2026-04-10T08:30:00Z",
      "is_official_image": true,
      "tags": [
        { "key": "env", "value": "production" },
        { "key": "managed-by", "value": "openclaw" }
      ],
      "cvm_instance_type": "Ai2.MEDIUM4",
      "cpu": 4,
      "memory_gb": 8,
      "system_disk_type": "CLOUD_BSSD",
      "system_disk_size": 50,
      "public_ip": "203.0.113.10",
      "internet_charge_type": "TRAFFIC_POSTPAID_BY_HOUR",
      "internet_max_bandwidth_out": 100,
      "adjustment_status": "processing",
      "adjustment_type": "instance_type",
      "adjustment_updated_at": "2026-07-19T12:00:00Z",
      "source": "cvm",
      "host_name": "",
      "os": "",
      "last_report_at": null
    },
    {
      "ID": 101,
      "InstanceId": "local-workbuddy-6a7b8",
      "Name": "alex-mbp",
      "Username": "alice",
      "agent_type": "workbuddy",
      "agent_version": "1.2.3",
      "source": "local",
      "host_name": "alex-mbp",
      "os": "darwin/arm64",
      "last_report_at": "2026-06-24T03:15:00Z",
      "projects": [
        {"project_id": 12, "project_name": "智能助手项目"}
      ],
      "status": "running",
      "label": "运行中",
      "tooltip": "",
      "actions": ["delete"],
      "transient": false
    }
  ],
  "stats": {
    "total": 100,
    "running": 60,
    "stopped": 20,
    "other": 20,
    "other_detail": {
      "need_attention": {
        "count": 12,
        "label": "⚠ 需关注",
        "items": { "create_failed": 2, "load_failed": 5, "maintaining": 3, "pending": 2 }
      },
      "in_progress": {
        "count": 8,
        "label": "◎ 处理中",
        "items": { "creating": 3, "loading": 5 }
      }
    }
  },
  "page": 1,
  "page_size": 20,
  "total": 100,
  "total_pages": 5
}
```

**实例对象字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| agent_type | string | 智能体类型：`openclaw`/`hermes`/`lightclawace` |
| agent_version | string | 智能体版本号（如 `2026.4.11`、`0.9.0`） |
| GroupID | uint | CVM 为创建时指定的分组 ID；本地 Agent 在 report 时同步当前用户级主组织，0 = 未指定 |
| GroupFullPath | string | 实例绑定分组的完整路径（如 `根组/研发组`）；`GroupID=0` 时为空串 |
| instance_charge_type | string | CVM 实例实际计费类型 |
| department | string | 创建用户的 OneID 主部门名（采用 `/admin/users` 的 department 字段口径）；用户无 OneID 画像时为空字符串 |
| departments | object[] | 创建用户的 OneID 完整部门列表；每项含 `department_id` / `department_name` / `department_parent_id` / `is_main_department` 基础字段 + `department_path`（本部门从根到当前部门的全路径）；用户无画像或反序列化失败时省略 |
| department_path | string | 主部门完整路径，按"父→子"用 `/` 拼接（如 `OpenClaw企业版体验/新组/市场组/市场二组`）；主部门 ID 在 `oneid_departments` 表中无法解析时省略。**与 `/admin/users` 同字段同口径，前端可复用同一套类型** |
| status | string | OpenClaw 状态（`running`/`stopped`/`loading` 等） |
| label | string | 状态中文标签 |
| tooltip | string | 状态说明提示文本 |
| actions | string[] | 管控端可执行操作列表 |
| transient | boolean | 是否为临时状态（需前端轮询） |
| cvm_instance_type | string | 当前 CVM 规格；仅 CVM 实例返回，值来自资源缓存 |
| cpu | int | 当前 CPU 核数；仅 CVM 实例返回 |
| memory_gb | int | 当前内存大小（GiB）；仅 CVM 实例返回 |
| system_disk_type | string | 当前系统盘介质类型；仅 CVM 实例返回 |
| system_disk_size | int | 当前系统盘容量（GiB）；仅 CVM 实例返回 |
| public_ip | string | 公网 IP；仅 CVM 实例返回，值来自后台状态缓存。未分配时为空字符串 |
| internet_charge_type | string | 公网计费模式；仅 CVM 实例返回，如 `TRAFFIC_POSTPAID_BY_HOUR` |
| internet_max_bandwidth_out | int | 公网带宽上限（Mbps）；仅 CVM 实例返回，未开通时为 0 |
| adjustment_status | string | 资源调整状态：空字符串、`processing` 或 `failed`；空值因 `omitempty` 省略 |
| adjustment_type | string | 当前或最近失败的调整类型：`instance_type` 或 `system_disk`；没有处理中或失败的调整时省略 |
| adjustment_error_code | string | 最近失败的稳定产品错误码；仅失败时返回 |
| adjustment_error_message | string | 根据当前请求语言从 `adjustment_error_code` 渲染的安全文案；仅失败时返回，不透传腾讯云原始错误 |
| adjustment_updated_at | string | 调整状态最近更新时间（RFC3339）；没有处理中或失败的调整时省略 |
| source 🆕 | string | 实例来源：`cvm`（CVM 实例）/ `local`（本地 agent 实例）；存量实例视为 `cvm` |
| host_name 🆕 | string | local 实例主机名（来自 `local_instance_infos.host_name`）；CVM 实例为空字符串 |
| os 🆕 | string | local 实例 OS（如 `darwin/arm64`/`linux/amd64`/`windows/amd64`）；CVM 实例为空字符串 |
| last_report_at 🆕 | string \| null | local 实例最近一次 reporter 上报时间（RFC3339）；CVM 实例为 `null` |
| projects 🆕 | array | local 实例所有 Workspace 绑定的有效项目摘要；每项含 `project_id`、`project_name`。CVM 实例不返回；已删除项目不返回 |

> `department_path` 计算规则：以 `MainDeptID` 为起点，沿 `oneid_departments.department_parent_id` 链反推到根，反转后用 `/` 拼接。同样规则适用于 `departments[i].department_path`。当系统未开通 OneID（所有用户 `OneIDSub` 均为空）时，本接口对部门字段做短路处理，不会触发 `oneid_user_profiles` / `oneid_departments` 表的查询。

**管控端 actions 映射：**

| status | actions |
|--------|---------|
| running | `["terminal", "stop", "delete", "reboot", "reinstall", "monitor"]` |
| stopped | `["start", "delete", "reinstall", "monitor"]` |
| creating / loading / destroying | `["monitor"]` |
| load_failed / maintaining | `["delete", "monitor"]` |
| pending | `["monitor"]` |
| create_failed | `["delete"]` |
| 资源调整处理中 | `[]`；主 `status` 保持调整前的 `running` 或 `stopped`，同时返回 `current_operation=adjust_instance_type` 或 `adjust_system_disk`、`adjustment_status=processing` |

> 本地实例（`source=local`）复用 `running`/`stopped` 两个状态，但 actions 列表会按本地实例特性裁剪（hatchery 无法主动启停本地机器）：用户端 / admin 端均仅保留 `delete`。

> `monitor` 为前端跳转，不调后端接口。管控端列表已内联实时状态，**不需要对每个实例单独轮询 `/status`**。
>
> **🆕 本地 agent 实例（`source='local'`）的 `status` 取值**：`running` / `stopped`（与 CVM 实例一致）。心跳超过 `LocalInstanceOfflineThreshold` 未上报会被刷为 `stopped`；`actions` 裁剪为 `[]`（hatchery 无法远程控制本地机器）。

### `POST /admin/instances/adjust-config/validate`

管理员批量实时校验 CVM 实例能否升配规格或扩容系统盘。该接口只读取实时云事实，不锁定实例、不创建任务、不记录审计日志；规格升配会额外询价，系统盘扩容按实例/磁盘状态、CBS 配额和请求模式本地判定，不调用磁盘询价接口。腾讯云没有 DryRun、库存预留或订单预留能力，因此校验通过不等于后续提交必然成功。

- **权限：** 管理员；已登录非管理员返回 403，未登录返回 401
- **Content-Type：** `application/json`
- **请求体上限：** 1 MiB
- **JSON 约束：** 未知字段、多 JSON 值、尾随非空内容或错误字段类型会被拒绝
- **请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | uint[] | 条件 | Hatchery 实例数据库 ID；与 `instance_ids` 必须且只能提供一种。移除 0、去重后 1～100 项，保留首次出现顺序 |
| instance_ids | string[] | 条件 | CVM 实例 ID；与 `ids` 必须且只能提供一种。去除空白、移除空值、去重后 1～100 项，保留首次出现顺序 |
| adjustment_type | string | 是 | `instance_type`（规格升配）或 `system_disk`（系统盘扩容） |
| target_instance_type | string | 条件 | `adjustment_type=instance_type` 时必填；仅支持固定 AI2 升配序列中的更高档规格 |
| target_system_disk_size | int64 | 条件 | `adjustment_type=system_disk` 时必填；绝对目标容量（GiB），必须大于当前容量且满足账号实时配额的最小值、最大值和步长 |
| resize_mode | string | 条件 | `adjustment_type=system_disk` 时必填；`online` 或 `offline` |

规格升配请求示例：

```json
{
  "ids": [42, 43],
  "adjustment_type": "instance_type",
  "target_instance_type": "Ai2.LARGE8"
}
```

系统盘扩容请求示例：

```json
{
  "instance_ids": ["ins-abcdefgh"],
  "adjustment_type": "system_disk",
  "target_system_disk_size": 100,
  "resize_mode": "online"
}
```

**逐实例准入：**

1. 只允许 `source=cvm` 的非龙虾医生实例；Hatchery 与腾讯云侧均不得有其他活动操作，实例必须处于 `RUNNING` 或 `STOPPED`，且不处于受限、停机不收费或不支持的计费状态。
2. 规格只允许严格向上调整：`Ai2.MEDIUM2` → `Ai2.MEDIUM4` → `Ai2.LARGE8` → `Ai2.2XLARGE16`。同档、降配、未知规格和跨族调整均拒绝；系统盘和全部数据盘必须是就绪的云硬盘，目标规格必须在实例可用区及计费模式下处于 `SELL`。
3. 系统盘只允许扩容，不允许缩容；必须是已挂载、非可携带、状态正常的云硬盘。目标容量必须满足实时 CBS 配额区间和步长。
4. 两种调整都会执行 DeniedAction 补充检查。规格升配继续调用 `InquiryPriceResetInstancesType` 排除余额、订单和售卖阻断；系统盘扩容不调用 `InquiryPriceResizeInstanceDisks`，因为该价格接口不是在线扩容能力接口。系统盘是否可提交由实时实例状态、`DescribeDisks`、`DescribeDiskConfigQuota` 及 `resize_mode` 确定；实际 `ResizeInstanceDisks` 若拒绝在线扩容，异步任务以稳定 `online_resize_not_supported` 失败，不静默降级。
5. 停机行为由目标和实时状态直接确定：运行中实例升配规格时自动停机；运行中系统盘扩容按 `resize_mode=online|offline` 决定在线执行或停机执行；原本已关机的实例保持关机。接口不接受额外停机确认字段。

- **成功响应：** 请求 envelope 合法时固定 HTTP 200；不可调整项进入 `results[]`，不会导致整批 4xx。

```json
{
  "adjustable_count": 1,
  "non_adjustable_count": 1,
  "results": [
    {
      "id": 42,
      "instance_id": "ins-abcdefgh",
      "current_instance_type": "Ai2.MEDIUM4",
      "current_system_disk_type": "CLOUD_BSSD",
      "current_system_disk_size": 50,
      "current_status": "running",
      "adjustable": true,
      "reason_code": "",
      "reason_message": ""
    },
    {
      "id": 43,
      "instance_id": "ins-ijklmnop",
      "current_instance_type": "Ai2.LARGE8",
      "current_system_disk_type": "CLOUD_BSSD",
      "current_system_disk_size": 50,
      "current_status": "running",
      "adjustable": false,
      "reason_code": "instance_type_unchanged",
      "reason_message": "已是目标规格"
    }
  ]
}
```

**results 字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | Hatchery 实例数据库 ID；按 `instance_ids` 查询但实例不存在时可能省略 |
| instance_id | string | CVM 实例 ID；按 `ids` 查询但实例不存在时可能省略 |
| current_instance_type | string | 实时 CVM 规格 |
| current_system_disk_type | string | 实时系统盘介质类型 |
| current_system_disk_size | int64 | 实时系统盘容量（GiB） |
| current_status | string | 实时 CVM 状态的小写形式 |
| adjustable | boolean | 当前实时事实下是否可调整 |
| reason_code | string | 不可调整时的稳定产品错误码；可调整时为空字符串 |
| reason_message | string | 按当前请求语言渲染的安全展示文案；不含腾讯云原始错误 |
| min_disk_size | int64 | 系统盘扩容时当前可选的最小有效容量；不适用或无有效容量时省略 |
| max_disk_size | int64 | 系统盘扩容时账号实时配额允许的最大容量；不适用时省略 |
| step_size | int64 | 系统盘扩容容量步长；不适用时省略 |

**请求级错误：**

| HTTP 状态 | 场景 |
|-----------|------|
| 400 | JSON 非法/超限/含未知字段；`ids`/`instance_ids` 二选一或数量不合法；调整类型、必填目标或扩容模式不合法 |
| 401 | 未登录 |
| 403 | 已登录但不是管理员 |
| 405 | 请求方法不是 POST |
| 500 | 数据库查询或创建腾讯云客户端失败，或无法取得完成整批校验所需的云事实 |

---

### `POST /admin/instances/adjust-config`

管理员批量提交 CVM 规格升配或系统盘扩容。服务会再次执行与校验接口相同的完整实时校验，并按实例独立受理；响应成功只表示任务已持久化受理，不表示腾讯云调整已经完成。**该接口记录审计日志。**

- **权限：** 管理员；已登录非管理员返回 403，未登录返回 401
- **Content-Type：** `application/json`
- **请求体上限：** 1 MiB
- **JSON 约束：** 未知字段、多 JSON 值、尾随非空内容或错误字段类型会被拒绝
- **审计 action：** `instance_adjust_config`
- **请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | uint[] | 条件 | Hatchery 实例数据库 ID；与 `instance_ids` 必须且只能提供一种。移除 0、去重后 1～100 项，保留首次出现顺序 |
| instance_ids | string[] | 条件 | CVM 实例 ID；与 `ids` 必须且只能提供一种。去除空白、移除空值、去重后 1～100 项，保留首次出现顺序 |
| adjustment_type | string | 是 | `instance_type`（规格升配）或 `system_disk`（系统盘扩容） |
| target_instance_type | string | 条件 | `adjustment_type=instance_type` 时必填；只允许固定 AI2 序列内严格升配 |
| target_system_disk_size | int64 | 条件 | `adjustment_type=system_disk` 时必填；绝对目标容量（GiB），必须满足实时配额 |
| resize_mode | string | 条件 | `adjustment_type=system_disk` 时必填；`online` 或 `offline` |

请求示例：

```json
{
  "instance_ids": ["ins-abcdefgh", "ins-ijklmnop"],
  "adjustment_type": "system_disk",
  "target_system_disk_size": 100,
  "resize_mode": "offline"
}
```

- **成功响应：** 请求 envelope 合法时固定 HTTP 200。每个目标的 `status` 独立为 `accepted`、`rejected` 或 `already_processing`。

```json
{
  "accepted_count": 1,
  "rejected_count": 0,
  "already_processing_count": 1,
  "results": [
    {
      "id": 42,
      "instance_id": "ins-abcdefgh",
      "current_instance_type": "Ai2.MEDIUM4",
      "current_system_disk_type": "CLOUD_BSSD",
      "current_system_disk_size": 50,
      "current_status": "running",
      "adjustable": true,
      "status": "accepted",
      "accepted": true,
      "reason_code": "",
      "reason_message": ""
    },
    {
      "id": 43,
      "instance_id": "ins-ijklmnop",
      "current_instance_type": "Ai2.MEDIUM4",
      "current_system_disk_type": "CLOUD_BSSD",
      "current_system_disk_size": 50,
      "current_status": "running",
      "adjustable": false,
      "status": "already_processing",
      "already_processing": true,
      "reason_code": "",
      "reason_message": ""
    }
  ]
}
```

完成实时复验的 `accepted`/`rejected` 结果沿用校验接口的当前值、原因和磁盘配额字段。命中同一持久化任务的 `already_processing` 在调用云接口前直接返回，只使用实例缓存的当前值，且可能省略实时磁盘配额字段。提交结果另增加：

| 字段 | 类型 | 说明 |
|------|------|------|
| status | string | `accepted`、`rejected` 或 `already_processing` |
| accepted | boolean | 仅 `true` 时返回；表示该实例已通过 CAS 锁定并进入持久化异步队列 |
| already_processing | boolean | 仅 `true` 时返回；表示同一实例已经在执行完全相同的目标调整 |

**幂等与异步语义：**

1. 同一实例、相同调整类型、相同目标和相同扩容模式重复提交时返回 `already_processing`；不同目标与任何其他活动操作冲突时返回 `rejected + operation_in_progress`。
2. worker 在首次腾讯云写入前第三次执行完整实时校验；通过后立即用同一规范化参数执行。每租户最多同时推进 5 台，总超时 15 分钟。
3. 运行中实例规格升配会自动停机；系统盘扩容严格按 `resize_mode` 执行，`online` 不停机、`offline` 自动停机。调整 worker 复用既有云轮询，在同一次数据库写中同步最近观察到的稳定 `RUNNING/STOPPED`，不增加云调用；过渡态保留最近稳定状态，`actions=[]`。调整完成后恢复提交前的状态。
4. 写请求 RequestId 尚未持久化就发生进程崩溃时，worker 至少进行 3 次成功云状态观察，每次间隔不少于 5 秒；读取失败不计入观察次数，避免重复云写。
5. 成功完成后清空 `adjustment_status`；失败保留 `adjustment_status=failed` 和稳定错误码，直到该实例下一次真正受理生命周期或配置写操作时清除。

请求级 400/401/403/405/500 语义与校验接口一致；实例级准入和云产品拒绝均通过 HTTP 200 的 `results[]` 返回。

#### 资源调整稳定原因码

| reason_code | 说明 |
|-------------|------|
| instance_not_found | Hatchery 实例不存在 |
| cloud_instance_required | 本地 Agent 不支持云资源调整 |
| doctor_node_not_allowed | 龙虾医生临时节点不允许调整 |
| operation_in_progress | 实例已有其他生命周期、配置或资源调整操作 |
| instance_status_not_supported | 实例不是可调整的运行/关机稳定态 |
| cvm_instance_not_found | 腾讯云侧未找到 CVM |
| cvm_restricted | CVM 处于受限状态 |
| cvm_operation_in_progress | 腾讯云侧已有操作进行中 |
| cvm_query_failed | 无法取得 CVM 实时事实 |
| stop_charging_not_supported | 实例处于停机不收费模式 |
| invalid_target | 目标参数不满足云产品要求 |
| unsupported_instance_type | 目标规格不在固定 AI2 支持序列 |
| instance_type_not_upgrade | 当前规格不在固定 AI2 升配序列 |
| instance_type_unchanged | 当前规格已经是目标规格 |
| instance_type_downgrade_not_supported | 目标规格低于当前规格，本期不支持降配 |
| cloud_disk_required | 规格调整要求所有关联盘均为云硬盘 |
| system_disk_type_not_supported | 当前系统盘介质不支持规格调整 |
| target_instance_type_unavailable | 目标规格在当前可用区/计费模式下不可售 |
| disk_quota_unavailable | 无法取得可用的系统盘扩容配额 |
| unsupported_charge_type | 实例或系统盘计费类型不受支持 |
| disk_not_ready | 系统盘或数据盘状态、挂载关系或实时容量不一致 |
| cloud_disk_unavailable | 腾讯云侧未找到所需云硬盘 |
| instance_network_incompatible | 当前网络配置不允许该调整 |
| instance_resource_limit_exceeded | 调整会超过实例资源限制 |
| resource_quota_exceeded | 账号资源配额不足 |
| instance_image_not_supported | 当前镜像不支持目标调整 |
| instance_feature_not_supported | 当前实例特性不支持目标调整 |
| promotion_restricted | 促销或活动实例限制调整 |
| invalid_disk_size | 目标容量不满足配额区间或步长 |
| disk_size_unchanged | 当前系统盘已经是目标容量 |
| disk_shrink_not_supported | 目标容量低于当前容量，系统盘不支持缩容 |
| online_resize_not_supported | 当前实例/系统盘不支持在线扩容 |
| insufficient_balance | 账号余额不足 |
| unpaid_order | 账号存在未支付订单 |
| resource_sold_out | 目标资源已售罄 |
| internal_error | 内部异常 |
| cloud_adjustment_failed | 未归类的腾讯云调整失败 |
| adjustment_timeout | 异步调整超过 15 分钟 |
| adjustment_restore_failed | 调整后恢复原开关机状态失败 |

---

### `POST /admin/instances/terminal-url`

管理员获取任意实例的 OrcaTerm 终端登录 URL。通过查询 CVM 实例信息自动判断操作系统和登录用户名，然后调用 OrcaTerm GenerateAuthLoginUrl 获取临时终端访问链接。与用户侧 `POST /openclaw/terminal-url` 不同，管理员可操作任意用户的实例。

- **权限：** 管理员
- **前置条件：** 管理员需开启终端功能（`TerminalEnabled = true`），否则返回 `403`
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **响应：** 始终 JSON

成功：

```json
{"login_url": "https://orcaterm.cloud.tencent.com/terminal?..."}
```

失败时返回错误信息：

- `401 {"error": "未登录"}`
- `403 {"error": "需要管理员权限"}`
- `403 {"error": "无权限访问"}`
- `403 {"error": "终端功能未开启，请在管理后台开启终端配置"}`
- `405 {"error": "请求方法不允许"}`
- `400 {"error": "缺少参数 id 或 instance_id"}` / `400 {"error": "无效的 id"}`
- `404 {"error": "实例不存在"}`
- `400 {"error": "该实例无关联的 CVM"}`
- `500 {"error": "创建 CVM 客户端失败: ..."}`
- `500 {"error": "查询 CVM 实例失败: ..."}`
- `404 {"error": "CVM 实例 ins-xxx 不存在"}`
- `500 {"error": "获取云 API 凭证失败: ..."}`
- `500 {"error": "获取终端登录 URL 失败: ..."}`

### `POST /admin/instances/denied-actions`

管理员批量查询 claw 实例对应 CVM 的禁用操作。仅返回 `DescribeInstanceVncUrl` 相关的 DeniedAction，用于判断实例是否可打开终端。与用户侧 `POST /openclaw/denied-actions` 不同，管理员可查询任意用户的实例。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **响应：** 始终返回 JSON
- **请求体：**

```json
{"ids": [1, 2, 3]}
// 或使用 instance_ids
{"instance_ids": ["ins-aaa", "ins-bbb"]}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | uint[] | 条件 | claw 实例数据库 ID 列表；与 `instance_ids` 二选一，`ids` 优先 |
| instance_ids | string[] | 条件 | CVM 实例 ID 列表；与 `ids` 二选一 |

> `ids` 与 `instance_ids` 至少传一个，同时存在时以 `ids` 为准。

- **成功响应：**

```json
{
  "instances": [
    {
      "id": 1,
      "denied_actions": [
        {
          "action": "DescribeInstanceVncUrl",
          "code": "UnsupportedOperation.InstanceStateStopped",
          "message": "不支持`已关机`的实例 (b2a82b28)"
        }
      ]
    },
    {
      "id": 2,
      "denied_actions": []
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| instances[].id | uint | claw 实例数据库 ID |
| instances[].denied_actions | array | 被禁用的操作列表（仅 `DescribeInstanceVncUrl`），为空数组表示无限制 |
| instances[].denied_actions[].action | string | 操作名称，固定为 `DescribeInstanceVncUrl` |
| instances[].denied_actions[].code | string | 禁用原因错误码 |
| instances[].denied_actions[].message | string | 禁用原因描述 |

失败时返回错误信息：

- `401/403 {"error": "..."}`
- `405 {"error": "Method not allowed, use POST"}`
- `400 {"error": "请求体格式错误，需要 JSON {\"ids\": [...]} 或 {\"instance_ids\": [...]}"}`
- `500 {"error": "查询实例禁用操作失败: ..."}`

注意：当不提供 `ids` 和 `instance_ids` 参数时，返回 `200 {"instances": []}`（空结果）而不是错误。

### `POST /admin/instances/cam-role`

批量为 CVM 实例绑定 ClawPro Agent 服务角色。包装腾讯云 CVM [ModifyInstancesAttribute](https://cloud.tencent.com/document/api/213/15739) 接口，固定设置 `CamRoleName=CVM_QCSLinkedRoleInClawProAgent`、`CamRoleType=service_linked`。**该接口记录审计日志。**

- **权限：** 管理员
- **Content-Type：** `application/json`
- **响应：** 始终返回 JSON
- **请求体：**

```json
{"instance_ids": ["ins-xxx", "ins-yyy"]}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_ids | string[] | 是 | CVM 实例 ID 列表 |

- **成功响应：**

```json
{
  "ok": true,
  "message": "已为 2 台实例绑定角色 CVM_QCSLinkedRoleInClawProAgent",
  "request_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "instance_ids": ["ins-xxx", "ins-yyy"],
  "cam_role_name": "CVM_QCSLinkedRoleInClawProAgent"
}
```

失败时返回错误信息：

- `401/403 {"error": "..."}`
- `400 {"error": "请求体格式错误: ..."}`
- `400 {"error": "instance_ids 不能为空"}`
- `500 {"error": "获取凭证失败: ..."}`
- `500 {"error": "ModifyInstancesAttribute 失败 (requestId=xxx): Code - Message"}`

### `GET /admin/instances/channels`

获取实例上已配置的 IM 通道列表。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **响应：** 始终 JSON
  - 成功：

```json
{
  "feishu": { "enabled": true, "accounts": { "default": { "app_id": "xxx", "app_secret": "xxx" } } },
  "qqbot":  { "enabled": true, "accounts": { "default": { "app_id": "xxx", "app_secret": "xxx" } } },
  "wecom":  { "bot": {}, "wecom_app": {} }
}
```

  - 失败：`400 {"error": "缺少参数 id 或 instance_id"}`、`404 {"error": "实例不存在"}`、`500 {"error": "解析 list_channels 脚本失败: ..."}`、`500 {"error": "解析通道列表失败: ..."}`

### `GET /admin/instances/skills`

获取实例上的技能列表。同时支持 CVM 与本地 agent 实例（响应结构按 `source` 不同）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **响应：** 始终 JSON

#### CVM 实例分支响应（不变）

```json
[
  { "name": "skill-name", "version": "1.0.0", "eligible": true }
]
```

#### 本地 agent 实例分支响应（🆕）

> 本接口只列出当前已成功安装（`install_status=success`）的 skill；安装中 / 安装失败的下发任务不在此接口返回。

```json
{
  "instance_id": 101,
  "skills": [
    {
      "slug": "weather",
      "name": "Weather",
      "version": "1.2.3",
      "install_status": "success",
      "error_message": "",
      "source": "public",
      "installed_at": "2026-06-23T10:00:00Z"
    },
    {
      "slug": "user-local-tool",
      "name": "User Local Tool",
      "version": "0.0.1",
      "install_status": "success",
      "error_message": "",
      "source": "local",
      "installed_at": "2026-06-20T14:32:00Z"
    }
  ],
  "total": 2
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| instance_id | uint | 实例数据库 ID |
| skills | array | 已安装技能列表 |
| skills[].slug | string | skill 唯一标识 |
| skills[].name | string | 展示名（取 `local_instance_skills.display_name`） |
| skills[].version | string | 已安装版本 |
| skills[].install_status | string | 固定为 `success`（本表只保存「成功装着」的事实） |
| skills[].error_message | string | 固定为空字符串（保留字段以保持响应结构对外稳定） |
| skills[].source | string | skill 来源：`public`（公共技能库）/ `enterprise`（企业技能库）/ `local`（用户在本地手动安装，hatchery 从未下发过） |
| skills[].installed_at | string \| null | 安装成功时间（RFC3339 UTC） |
| total | int | 已安装 skill 数量 |

- **失败响应：**
  - `400 {"error": "缺少参数 id 或 instance_id"}`
  - `404 {"error": "实例不存在"}`
  - `500 {"error": "..."}`

### `GET /admin/instances/models`

管控端查询任意实例的所有模型绑定列表（多模型 Fallback v2.0），与用户侧 `GET /openclaw/instance-models` 能力对齐：返回 `instance_models` 表中归属该实例的全部记录，按 `sort_order DESC` 排序，同时区分内置模型（关联 `ai_models`）与自定义模型（解析 `CustomModelConfig` JSON）。

与用户侧的差异：

- 鉴权方式：管理员（session 中 `role == admin` 或 Bearer admin-token）；
- 不限制实例所有者，可查询任意用户名下的实例。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID（URL query 或 form） |

- **响应：** `application/json`

```json
{
  "ok": true,
  "models": [
    {
      "instance_model_id": 12,
      "binding_id": "hatchery-qwen-max/qwen-max",
      "ai_model_id": 10,
      "role": "fallback",
      "provider": "通义千问",
      "model_id": "qwen-max",
      "model_name": "Qwen Max",
      "model_type": "openai-completions",
      "context_len": 128000,
      "max_tokens": 8192,
      "custom_http_headers": {
        "key": "value"
      },
      "is_custom": false
    },
    {
      "instance_model_id": 11,
      "binding_id": "hatchery-glm-4-plus/glm-4-plus",
      "ai_model_id": 8,
      "role": "primary",
      "provider": "智谱AI",
      "model_id": "glm-4-plus",
      "model_name": "GLM-4 Plus",
      "model_type": "openai-completions",
      "context_len": 128000,
      "max_tokens": 8192,
      "custom_http_headers": {
        "key": "value"
      },
      "is_custom": false
    }
  ]
}
```

响应字段说明同 `GET /openclaw/instance-models`：

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| instance_model_id | uint | `instance_models.id` |
| binding_id | string | 绑定引用标识 `providerKey/modelId`（自定义模型形如 `custom-<id>/<id>`，内置形如 `hatchery-<modelID>/<modelID>`） |
| ai_model_id | uint | 预置模型数据库 ID，自定义模型为 `0` |
| role | string | `primary` / `fallback` |
| provider | string | 提供商展示名 |
| model_id | string | 模型 ID |
| model_name | string | 模型显示名称 |
| model_type | string | 接口类型（`openai-completions` / `anthropic-messages`） |
| context_len | int | 上下文长度 |
| max_tokens | int | Agent 单次请求模型最大输出 Token 数 |
| custom_http_headers| object | 否 | Agent 请求模型时的自定义 HTTP 头部，格式为 JSON 字符串，如 `{"Header-Key": "Header-Value"}`|
| is_custom | bool | 是否为自定义模型（`ai_model_id == 0` 时为 `true`） |

- **错误响应：**
  - `401 {"error": "未登录"}`
  - `403 {"error": "需要管理员权限"}`
  - `403 {"error": "无权限访问"}`
  - `400 {"error": "缺少参数 id"}`
  - `400 {"error": "实例不存在"}`

### `GET /admin/instances/available-models`

返回实例可配置的模型列表（已启用且对该实例的分组可见），与用户端 `GET /openclaw/models` 对齐。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID（URL query 或 form） |

- **响应：** `application/json`

```json
{
  "ok": true,
  "models": [
    {
      "id": 10,
      "provider": "openai",
      "model_id": "gpt-4o",
      "model_type": "openai-completions",
      "input_types": ["text", "image"],
      "context_len": 128000,
      "max_tokens": 8192,
      "custom_http_headers": {
        "key": "value"
      },
      "model_name": "GPT-4o",
      "default": true
    }
  ]
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| id | uint | 模型数据库 ID（ai_models.id） |
| provider | string | 提供商名称 |
| model_id | string | 模型 ID |
| model_type | string | 接口类型 |
| input_types | []string | 支持的输入类型 |
| context_len | int | 上下文长度 |
| max_tokens | int | Agent 单次请求模型最大输出 Token 数 |
| custom_http_headers| object | 否 | 自定义 HTTP 请求头部 (键值对) |
| model_name | string | 模型显示名称 |
| default | bool | 是否为站点默认模型 |

- **错误响应：**
  - `403 {"error": "<agent_type> 类型实例不支持模型配置"}`
  - `400 {"error": "缺少参数 id"}` / `"实例不存在"}`

### `GET /admin/instances/available-channels`

返回实例可配置的通道列表（已启用、对该实例分组可见、且 agent_type 支持）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID（URL query 或 form） |

- **响应：** `application/json`

```json
{
  "ok": true,
  "channels": [
    {
      "ChannelID": "feishu",
      "Name": "飞书",
      "Enabled": true,
      "Custom": false,
      "CustomConfig": "",
      "visibility_type": "all",
      "params": [
        {"key": "app_id", "label": "应用App ID"},
        {"key": "app_secret", "label": "应用App Secret"}
      ],
      "agent_types": ["openclaw", "lightclawace"]
    }
  ]
}
```

### `POST /admin/instances/set-model`

管控端设置/替换实例模型。与用户端 `POST /openclaw/set-model` 能力对齐，但管理员可操作任意实例。缺省更新 primary；传 `instance_model_id` 时精确更新指定绑定（OpenClaw 可用于更新某个 fallback）。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| ai_model_id | uint | 是 | 模型数据库 ID（0 表示自定义模型） |
| instance_model_id | uint | 否 | 指定要更新的 `instance_models.id`。缺省时更新 primary；传入时保留目标绑定原 `role`，只替换这条记录 |
| provider | string | 否 | 自定义模型提供商 |
| model_id | string | 否 | 自定义模型 ID |
| model_name | string | 否 | 自定义模型显示名称 |
| api_key | string | 否 | 自定义模型 API Key |
| url | string | 否 | 自定义模型 URL |
| model_type | string | 否 | 自定义模型接口类型 |
| context_len | int | 否 | 自定义模型上下文长度 |
| max_tokens | int | 否 | 自定义模型 Agent 单次请求模型最大输出 Token 数（设置值小于等于 0 时默认 8192） |
| custom_http_headers | string | 否 | Agent 请求模型时的自定义 HTTP 头部，格式为 JSON 字符串，如 `{"Header-Key": "Header-Value"}` |
| input_types[] | string[] | 否 | 自定义模型输入类型 |

*`ai_model_id` 为 0 时需填写自定义模型字段。

不传 `instance_model_id` 时保持旧行为：OpenClaw 更新 primary，Hermes / LightclawACE 更新其唯一模型。传 `instance_model_id` 时按 `id + instance_model_id` 精确定位 `instance_models` 记录；目标是 fallback 时只改该 fallback，不修改 primary / `instances.ai_model_id`。

- **响应：** 默认 primary 更新返回 `{"ok": true, "provider": "...", "model_id": "..."}`；指定绑定更新返回 `{"ok": true, "role": "primary|fallback", "instance_model_id": 12, "binding_id": "...", "provider": "...", "model_id": "...", "model_name": "..."}`

- **错误响应：**
  - `400 {"error": "缺少或无效的 instance_model_id 参数"}`
  - `400 {"error": "目标模型记录不存在或不属于该实例"}`
  - `403 {"error": "该模型对当前实例不可见"}`
  - `403 {"error": "<agent_type> 类型实例不支持模型配置"}`
  - `409 {"error": "该模型已绑定到此实例"}`
  - `409 {"error": "实例状态不支持此操作"}`

### `POST /admin/instances/batch-set-model`

管控端批量覆盖实例模型配置。请求体为 JSON；顶层模型字段与单实例 `POST /admin/instances/set-model` 不传 `instance_model_id` 时一致，用作 primary；`fallbacks` 是同一模型字段结构的数组。接口的最终效果是**目标实例的模型集合等于顶层 primary + `fallbacks`**：未出现在请求里的旧 fallback 会被删除，本接口不是 add-model。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **限制：**
  - 单次最多 20 个目标实例；并发执行上限为 5。
  - 仅 running 实例可配置；本地实例不支持。
  - 所有已解析目标必须是同一个标准化 `agent_type`；空值按 `openclaw` 处理。不同内置类型、自定义类型与其兼容内置类型、自定义类型之间混选，均返回请求级 `400`。
  - 自定义 Agent 类型按其兼容的内置运行时执行脚本；未配置兼容内置类型时按单项失败返回“不支持模型配置”。
  - OpenClaw 及兼容 OpenClaw 的自定义类型支持 primary + fallback；Hermes / LightclawACE 及兼容它们的自定义类型仅支持一个模型，带 `fallbacks` 的请求会按单项失败返回。
  - OpenClaw Agent 版本为 `3.28.x` 时不支持 fallback；带 `fallbacks` 的请求会按单项失败返回。
- **目标选择：** `ids` 和 `instance_ids` 至少提供一个。两者同时提供时只使用 `ids`，忽略 `instance_ids`。
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | uint[] | 条件 | 实例数据库 ID 列表；会过滤 0 并按首次出现顺序去重；与 `instance_ids` 二选一，`ids` 优先 |
| instance_ids | string[] | 条件 | CVM 实例 ID 列表；会 trim 空白、过滤空字符串并按首次出现顺序去重；与 `ids` 二选一 |
| ai_model_id | uint | 是 | 顶层 primary 模型数据库 ID；`0` 表示使用自定义模型配置 |
| provider | string | 否 | 自定义模型提供商，空值默认 `custom` |
| model_id | string | 否 | 自定义模型 ID；`ai_model_id=0` 时必填 |
| model_name | string | 否 | 自定义模型显示名称，空值默认等于 `model_id` |
| api_key | string | 否 | 自定义模型 API Key；`ai_model_id=0` 时必填 |
| url | string | 否 | 自定义模型 URL；`ai_model_id=0` 时必填，必须为 `http://` 或 `https://` |
| model_type | string | 否 | 自定义模型接口类型；`ai_model_id=0` 时必填，仅支持 `openai-completions`、`anthropic-messages` |
| input_types | string[] | 否 | 自定义模型输入类型，空值默认 `["text"]` |
| context_len | int | 否 | 自定义模型上下文长度，值小于等于 0 时默认 `128000` |
| max_tokens | int | 否 | 自定义模型 Agent 单次请求模型最大输出 Token 数，值小于等于 0 时默认 `8192` |
| custom_http_headers | object | 否 | Agent 请求模型时的自定义 HTTP 头部对象，如 `{"Header-Key": "Header-Value"}` |
| fallbacks | array | 否 | fallback 模型数组。每个元素使用同一组模型字段：`ai_model_id`、`provider`、`model_id`、`model_name`、`api_key`、`url`、`model_type`、`input_types`、`context_len`、`max_tokens`、`custom_http_headers` |

说明：`ids` 与 `instance_ids` 至少一个非空。`instance_model_id` 不属于本接口；本接口总是覆盖目标实例的 primary + fallback 集合。

- **请求示例（内置 primary + 一个内置 fallback）：**

```json
{
  "ids": [1, 2, 3],
  "ai_model_id": 123,
  "fallbacks": [
    {"ai_model_id": 456}
  ]
}
```

- **请求示例（自定义 primary，无 fallback，清空旧 fallbacks）：**

```json
{
  "instance_ids": ["ins-aaa", "ins-bbb"],
  "ai_model_id": 0,
  "provider": "custom",
  "model_id": "gpt-4o",
  "model_name": "GPT-4o",
  "api_key": "sk-...",
  "url": "https://api.example.com/v1",
  "model_type": "openai-completions",
  "input_types": ["text", "image"],
  "context_len": 128000,
  "max_tokens": 8192,
  "custom_http_headers": {
    "X-Org": "team-a"
  },
  "fallbacks": []
}
```

- **响应：** 请求级校验通过时始终返回 `200`，包括全部目标不存在、全部已解析目标执行失败；每个目标的结果按去重后的请求顺序返回。调用方需通过现有实例模型列表接口刷新最终模型状态。

```json
{
  "ok": true,
  "results": [
    {
      "id": 1,
      "instance_id": "ins-aaa",
      "name": "agent-a",
      "status": "ok",
      "message": "模型配置已覆盖"
    },
    {
      "id": 999,
      "status": "failed",
      "message": "实例不存在"
    }
  ]
}
```

- **`results[]` 字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 实例数据库 ID；按 `ids` 查询且目标不存在时保留请求中的 ID |
| instance_id | string | CVM 实例 ID；按 `instance_ids` 查询且目标不存在时保留请求中的 instance_id |
| name | string | 实例名称 |
| status | string | `ok` 或 `failed` |
| message | string | 成功或失败的本地化说明 |

- **请求级错误响应：**
  - `400 {"error": "无效的 JSON 格式"}`
  - `400 {"error": "缺少参数 ids 或 instance_ids"}`
  - `400 {"error": "ids 数量超过上限 20"}`
  - `400 {"error": "instance_ids 数量超过上限 20"}`
  - `400 {"error": "缺少参数"}`
  - `400 {"error": "主模型和备选模型不能重复，备选模型之间也不能重复"}`
  - `400 {"error": "所选 Agent 包含 openclaw 与非 openclaw 两种类型，不能一起批量配置模型"}`
  - `405 {"error": "方法不允许"}`

### `POST /admin/instances/add-model`

管控端添加模型（首个自动 primary，后续 fallback）。与用户端 `POST /openclaw/add-model` 能力对齐。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| ai_model_id | uint | 是 | 要绑定的模型 ID |

- **响应：**

```json
{
  "ok": true,
  "role": "primary",
  "instance_model_id": 15,
  "binding_id": "hatchery-gpt-4o/gpt-4o",
  "provider": "openai",
  "model_id": "gpt-4o",
  "model_name": "GPT-4o"
}
```

### `POST /admin/instances/switch-primary-model`

管控端切换主备模型。与用户端 `POST /openclaw/switch-primary-model` 能力对齐。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| instance_model_id | uint | 是 | 目标 instance_models.id |

- **响应：**

```json
{
  "ok": true,
  "new_primary": {
    "binding_id": "hatchery-gpt-4o/gpt-4o",
    "instance_model_id": 15,
    "provider": "openai",
    "model_id": "gpt-4o",
    "model_name": "GPT-4o",
    "role": "primary"
  },
  "demoted_to_fallback": {
    "binding_id": "hatchery-claude-3.5/claude-3.5",
    "instance_model_id": 12,
    "provider": "anthropic",
    "model_id": "claude-3.5",
    "model_name": "Claude 3.5",
    "role": "fallback"
  }
}
```

### `POST /admin/instances/del-model`

管控端删除模型绑定。与用户端 `POST /openclaw/del-model` 能力对齐。删除 primary 时自动提升最早 fallback。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| instance_model_id | uint | 是 | 目标 instance_models.id |

- **响应：**

```json
{
  "ok": true,
  "was_primary": true,
  "promoted_model": {
    "binding_id": "hatchery-gpt-4o/gpt-4o",
    "instance_model_id": 15,
    "provider": "openai",
    "model_id": "gpt-4o",
    "model_name": "GPT-4o",
    "role": "primary"
  }
}
```

### `POST /admin/instances/proxy/prepare` 🆕

管控端为指定实例创建或刷新通用反向代理入口。当前用于 Microsoft Teams / LINE webhook endpoint 预生成。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| kind | string | 否 | 代理类型；当前支持 `teams`、`line`，默认 `teams` |

- **响应：** `{"ok": true, "kind": "teams", "route_id": "...", "endpoint": "https://.../api/proxy/.../api/messages"}`（LINE 的 endpoint 格式为 `https://.../api/proxy/.../line/webhook`）

### `POST /admin/instances/set-channel`

管控端设置/编辑通道配置。与用户端 `POST /openclaw/set-channel` 能力对齐。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| channel | string | 是 | 通道标识（如 feishu / wecom / msteams / line） |
| key[] | []string | 是 | 配置参数 key 列表 |
| value[] | []string | 是 | 配置参数 value 列表（与 key[] 一一对应） |

- **响应：** `{"ok": true, "output": "..."}`；当 `channel=msteams` 时额外包含 `proxy_route_id`、`proxy_endpoint`、`teams_endpoint`；当 `channel=line` 时额外包含 `proxy_route_id`、`proxy_endpoint`
- **错误响应：**
  - `403 {"error": "通道 xxx 对当前实例不可见"}`
  - `400 {"error": "agent_type xxx 不支持通道 xxx"}`

### `POST /admin/instances/del-channel`

管控端删除已配置通道。与用户端 `POST /openclaw/del-channel` 能力对齐。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例数据库 ID |
| channel | string | 是 | 通道标识 |

- **响应：** `{"ok": true, "output": "..."}`

### `POST /admin/instances/delete`

管理员删除实例（同时销毁对应的腾讯云 CVM）。与用户侧 `POST /openclaw/delete` 不同，管理员可删除任意用户的实例，并向被删除实例的所属用户发送通知。支持单删和批量删除。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded` 或 `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID（单删模式） |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`，单删模式）；与 `id` 二选一 |
| ids | uint[] | 条件 | 实例数据库 ID 数组（批量模式，上限 100）；`ids` 不传时以 `id`/`instance_id` 为准 |
| instance_ids | string[] | 条件 | CVM 实例 ID 数组（批量模式，上限 100）；与 `ids` 二选一 |

> 优先级：`ids` > `instance_ids` > `id` > `instance_id`，至少传其中一个。

- **行为说明：**
  - **批量模式（`ids` 非空）：** 一次腾讯云 `TerminateInstances` 调用，响应带 `results[]`，强制 JSON
  - **单删模式（仅 `id`）：** 行为与原逻辑一致（HTML/JSON 双形态）
  - 创建失败（`create_failed`）或无 CVM 实例 ID：仅删除数据库记录
  - 正常实例：调用 `TerminateInstances` + `PurgeInstances` 确认销毁，CVM 消失后清除 DB 记录
  - 删除成功后向实例所属用户发送 `admin_delete` 类型通知
- **JSON 响应（批量模式）：**

```json
{
  "ok": true,
  "results": [
    { "id": 11, "instance_id": "ins-aaa", "name": "agent-1", "status": "started", "message": "销毁中" },
    { "id": 12, "instance_id": "ins-bbb", "name": "agent-2", "status": "deleted", "message": "已释放" },
    { "id": 13, "instance_id": "",        "name": "agent-3", "status": "failed",  "message": "释放失败" }
  ]
}
```

- **JSON 响应（单删模式）：**
  - 成功：`{"ok": true}`
  - 失败：`400 {"error": "无效的实例 ID"}` / `404 {"error": "实例不存在"}` / `500 {"error": "..."}`
- **错误（批量模式）：**
  - `400` `ids 数量超过上限 100`
  - `400` `ids 不能为空列表`
  - `400` `instance_ids 数量超过上限 100`
  - `400` `instance_ids 对应的实例不存在`
  - `400` `缺少参数 id 或 instance_id`

### `POST /admin/instances/start`

管理员开机实例，支持单个和批量开机。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded` 或 `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 单个实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | 单个 CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |
| ids | uint[] | 条件 | 批量实例数据库 ID；仅 JSON 支持；与 `instance_ids` 二选一，`ids` 优先 |
| instance_ids | string[] | 条件 | 批量 CVM 实例 ID；仅 JSON 支持；与 `ids` 二选一 |

> `id` / `instance_id` 支持 Query、Form、JSON；`ids` / `instance_ids` 仅支持 JSON。若请求包含单个与批量参数，批量参数优先。

- **前置条件：** CVM 状态为 `STOPPED`
- **行为：** 开机
- **本地实例拒绝：** `source='local'` 的实例会被返 `400 {"error": "本地实例不支持此操作"}`。batch 模式下仅拒绝 local target，其它 CVM target 正常处理（results 数组里对应项返 `ok=false`）。
- **HTML 响应：** HTML 片段（`admin_instance_list`）
- **JSON 响应：**
  - 单个成功：`{"ok": true}`
  - 批量成功/部分失败：`{"ok": true, "results": [{"id":1,"instance_id":"ins-a","name":"agent-a","status":"started","message":"开机已下发"},{"id":2,"instance_id":"ins-b","name":"agent-b","status":"skipped","message":"实例当前为运行中，无法执行该操作"}]}`
  - 失败：`400 {"error": "缺少参数 id 或 instance_id"}` / `404 {"error": "实例不存在"}` / `500 {"error": "..."}`

### `POST /admin/instances/stop`

管理员关机实例，支持单个和批量关机。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded` 或 `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 单个实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | 单个 CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |
| ids | uint[] | 条件 | 批量实例数据库 ID；仅 JSON 支持；与 `instance_ids` 二选一，`ids` 优先 |
| instance_ids | string[] | 条件 | 批量 CVM 实例 ID；仅 JSON 支持；与 `ids` 二选一 |

> `id` / `instance_id` 支持 Query、Form、JSON；`ids` / `instance_ids` 仅支持 JSON。若请求包含单个与批量参数，批量参数优先。

- **前置条件：** CVM 状态为 `RUNNING`
- **行为：** 关机；按量计费实例关机后停止计算资源计费。不发送通知。
- **本地实例拒绝：** `source='local'` 的实例会被返 `400 {"error": "本地实例不支持此操作"}`。batch 模式下仅拒绝 local target，其它 CVM target 正常处理。
- **HTML 响应：** HTML 片段（`admin_instance_list`）
- **JSON 响应：**
  - 单个成功：`{"ok": true}`
  - 批量成功/部分失败：`{"ok": true, "results": [{"id":1,"instance_id":"ins-a","name":"agent-a","status":"started","message":"关机已下发"},{"id":2,"instance_id":"ins-b","name":"agent-b","status":"skipped","message":"实例当前为已关机，无法执行该操作"}]}`
  - 失败：`400 {"error": "缺少参数 id 或 instance_id"}` / `404 {"error": "实例不存在"}` / `500 {"error": "..."}`

### `POST /admin/instances/reboot`

管理员重启实例（调用腾讯云 CVM RebootInstances 接口）。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **前置条件：** CVM 状态为 `RUNNING`
- **行为：** 调用 `RebootInstances`，同时将 `agent_ready` 重置为 `0`
- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`400 {"error": "缺少参数 id 或 instance_id"}` / `404 {"error": "实例不存在"}` / `500 {"error": "..."}`

### `POST /admin/instances/restart-gateway`

管理员重启实例内对应 agent 类型的 gateway 服务，不重启腾讯云 CVM 实例。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded` 或 `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID（单实例模式） |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`，单实例模式）；与 `id` 二选一 |
| ids | uint[] | 条件 | 实例数据库 ID 数组（批量模式，上限 50）；`ids` 不传时以 `id`/`instance_id` 为准 |
| instance_ids | string[] | 条件 | CVM 实例 ID 数组（批量模式，上限 50）；与 `ids` 二选一 |

> 优先级：`ids` > `instance_ids` > `id` > `instance_id`，至少传其中一个。

- **前置条件：** 实例状态为 `RUNNING`
- **行为：** 通过 TAT 按 agent 类型执行 gateway 重启脚本（OpenClaw / Hermes / LightclawACE）；不调用 CVM `RebootInstances`，不写入 `current_operation`，不重置 `agent_ready`
- **JSON 响应（单实例模式）：**
  - 成功：`{"ok": true}`
  - 失败：`400 {"error": "..."}` / `404 {"error": "实例不存在"}` / `409 {"error": "..."}` / `500 {"error": "..."}`
- **JSON 响应（批量模式）：**

```json
{
  "ok": true,
  "results": [
    { "id": 11, "instance_id": "ins-aaa", "name": "agent-1", "status": "ok", "message": "gateway 已重启" },
    { "id": 12, "instance_id": "ins-bbb", "name": "agent-2", "status": "failed", "message": "实例已关机，请先开机并等待实例恢复运行中后再操作" }
  ]
}
```

### `POST /admin/instances/reset`

管理员重装实例（调用腾讯云 CVM ResetInstance 接口，使用当前启用镜像）。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **前置条件：** CVM 状态为 `RUNNING` 或 `STOPPED`
- **行为：** 调用 `ResetInstance`，同时将 `agent_ready`、`cls_agent_status` 重置为 `0`，`ai_model_id` 清零，`agent_version`、`agent_type`、`plugin_versions_json` 清空，`version_fetched_at` 置空（重装后需重新拉取版本信息）
- **JSON 响应：**
  - 成功：`{"ok": true}`
  - 失败：`400 {"error": "缺少参数 id 或 instance_id"}` / `404 {"error": "实例不存在"}` / `500 {"error": "..."}`

### `POST /admin/instances/refresh-version`

管理员手动触发单个实例的版本信息刷新。同步等待 TAT 脚本执行完成（最长 60s），返回最新版本信息。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **前置条件：** 实例 Agent 已就绪（`agent_ready=1`）
- **行为：** 通过 TAT 在实例上执行 `get_version_info.sh` 脚本，获取 openclaw 主程序版本和已安装插件版本，写入数据库
- **JSON 响应：**
  - 成功：
    ```json
    {
      "ok": true,
      "agent_version": "2026.3.28",
      "agent_type": "openclaw",
      "plugin_version_status": [
        {
          "slug": "openclaw-qqbot",
          "display_name": "QQ 机器人",
          "installed_version": "1.6.7",
          "latest_version": "1.7.0",
          "need_upgrade": true,
          "not_installed": false
        }
      ],
      "version_fetched_at": "2026-04-10T08:30:00Z"
    }
    ```
  - 失败：`400 {"error": "缺少参数 id 或 instance_id"}` / `400 {"error": "实例 Agent 未就绪，无法拉取版本信息"}` / `404 {"error": "实例不存在"}` / `500 {"error": "版本信息拉取失败: ..."}`

### `POST /admin/instances/batch-upgrade`

管理员批量升级实例镜像。升级前会检查所有实例是否使用官方公共镜像，如果存在非官方镜像的实例，整个请求将被拒绝。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

```json
{"ids": [1, 2, 3]}
// 或使用 instance_ids
{"instance_ids": ["ins-aaa", "ins-bbb"]}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | uint[] | 条件 | 实例数据库 ID 列表（最多 20 个）；与 `instance_ids` 二选一，`ids` 优先 |
| instance_ids | string[] | 条件 | CVM 实例 ID 列表（最多 20 个）；与 `ids` 二选一 |

> `ids` 与 `instance_ids` 至少传一个，同时存在时以 `ids` 为准。

- **前置条件：**
  1. 后台已配置并启用默认镜像，且该镜像为官方公共镜像
  2. 所有目标实例均关联了 CVM，且当前使用的镜像为官方公共镜像
  3. 实例当前状态为 `running`（非运行中的实例会被跳过）

- **行为：** 逐个检查实例状态，对满足条件的实例异步启动升级流程（复用 `POST /openclaw/upgrade` 的升级逻辑）。每个实例的升级结果独立返回。

- **JSON 响应：**
  - 成功：
    ```json
    {
      "ok": true,
      "results": [
        {"id": 1, "name": "instance-a", "status": "started", "message": "升级已开始"},
        {"id": 2, "name": "instance-b", "status": "skipped", "message": "实例已是最新版本，无需升级"},
        {"id": 3, "name": "instance-c", "status": "skipped", "message": "实例正在进行 reboot 操作，跳过"},
        {"id": 4, "name": "instance-d", "status": "skipped", "message": "实例当前状态为 stopped（已关机），仅运行中的实例支持升级"},
        {"id": 5, "name": "instance-e", "status": "failed", "message": "检查升级状态失败: ..."}
      ]
    }
    ```
  - 参数错误：`400 {"error": "缺少参数 ids 或 instance_ids"}` / `400 {"error": "单次批量升级最多支持 20 个实例"}`
  - 镜像校验失败：`400 {"error": "以下实例使用的不是官方公共镜像，无法批量升级：instance-a(ID=1, 镜像=img-xxx)"}`
  - 无默认镜像：`500 {"error": "无法获取默认镜像，请先在后台开启一个镜像"}`

**results 中每个实例的 status 枚举：**

| status | 说明 |
|--------|------|
| started | 升级已异步启动 |
| skipped | 跳过（已是最新版本 / 正在进行其他操作 / 非运行状态） |
| failed | 检查或启动升级失败 |

### `GET /admin/instances/status`

管理员查询单个实例的 CVM 原始状态（非 OpenClaw 语义状态）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 实例数据库 ID（Query 参数）；与 `instance_id` 二选一，`id` 优先 |
| instance_id | string | 条件 | CVM 实例 ID（如 `ins-xxxxxxxx`，Query 参数）；与 `id` 二选一 |

> `id` 与 `instance_id` 至少传一个，同时存在时以 `id` 为准。

- **响应：** 始终 JSON。资源字段优先使用本次实时 `DescribeInstances` 结果；腾讯云已释放实例时 `state=RELEASED`，资源字段回退到最近缓存。调整错误文案按当前请求语言渲染。

```json
{
  "state": "RUNNING",
  "cvm_instance_type": "Ai2.MEDIUM4",
  "cpu": 4,
  "memory_gb": 8,
  "system_disk_type": "CLOUD_BSSD",
  "system_disk_size": 50,
  "public_ip": "203.0.113.10",
  "internet_charge_type": "TRAFFIC_POSTPAID_BY_HOUR",
  "internet_max_bandwidth_out": 100,
  "adjustment_status": "processing",
  "adjustment_type": "system_disk",
  "adjustment_error_code": "",
  "adjustment_error_message": "",
  "adjustment_updated_at": "2026-07-19T12:00:00Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| state | string | CVM 原始状态，如 `RUNNING`、`STOPPED`、`PENDING` 或 `RELEASED` |
| cvm_instance_type | string | 当前实时 CVM 规格，实时结果缺失时使用最近缓存 |
| cpu | int | CPU 核数 |
| memory_gb | int | 内存大小（GiB） |
| system_disk_type | string | 系统盘介质类型 |
| system_disk_size | int | 系统盘容量（GiB） |
| public_ip | string | 公网 IP；未分配时为空字符串 |
| internet_charge_type | string | 公网计费模式，如 `TRAFFIC_POSTPAID_BY_HOUR` |
| internet_max_bandwidth_out | int | 公网带宽上限（Mbps）；未开通时为 0 |
| adjustment_status | string | 空字符串、`processing` 或 `failed` |
| adjustment_type | string | `instance_type` 或 `system_disk` |
| adjustment_error_code | string | 最近失败的稳定产品错误码 |
| adjustment_error_message | string | 根据错误码和当前请求语言渲染的安全文案 |
| adjustment_updated_at | string \| null | 调整状态最近更新时间（RFC3339） |

- **失败：** `400` 缺少 `id`/`instance_id` 或目标为本地 Agent；`404` 实例不存在；`500` 查询腾讯云失败。

> **注意**：此接口返回的是 CVM 原始状态字符串，与用户侧 `GET /openclaw/status` 返回的 Agent 语义状态不同。本地 Agent 仍按既有契约拒绝，不返回云资源或调整字段。

### `POST /admin/instances/detect-install`

管理员检测实例上的 OpenClaw/Hermes/ACE 安装状态。支持单实例和批量检测。通过 TAT 远程执行探测脚本，按 `agent_type` 自动分派对应的探测脚本。

- **权限：** 管理员
- **Content-Type：** `application/x-www-form-urlencoded`（单实例）或 `application/json`（批量）
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | 单实例模式：实例数据库 ID（Form 参数）；与 `instance_id` 二选一 |
| instance_id | string | 条件 | 单实例模式：CVM 实例 ID（Form 参数）；与 `id` 二选一 |
| ids | uint[] | 条件 | 批量模式：JSON body `{"ids": [1, 2, 3]}`，最多 50 个；与 `instance_ids` 二选一 |
| instance_ids | string[] | 条件 | 批量模式：JSON body `{"instance_ids": ["ins-aaa", ...]}`，最多 50 个；与 `ids` 二选一 |

> 优先级：`id`/`instance_id`（Form 参数）> JSON body `ids` > JSON body `instance_ids`。至少传其中一个。

- **响应：** 始终 JSON

```json
[
  {
    "id": 42,
    "instance_id": "ins-xxxxx",
    "status": "ok",
    "output": { "runtime_user": "openclaw", "service_status": "active", ... }
  },
  {
    "id": 43,
    "instance_id": "",
    "status": "skip",
    "error": "无关联 CVM"
  }
]
```

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 实例数据库 ID |
| instance_id | string | CVM 实例 ID |
| status | string | `ok` / `skip` / `error` |
| output | object | 探测脚本输出的 JSON（`status=ok` 时） |
| error | string | 错误信息（`status=skip` 或 `error` 时） |

### `GET /admin/feature-allowlist/check` 🆕

查询当前登录管理员所属租户是否已开通指定功能。免登录功能使用前必须先申请并开通白名单。

- **权限：** 管理员
- **参数（Query）：**

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `type` | string | 是 | 功能类型，如 `local-agent` 或 `passwordless-login` |

- **响应：** `200 application/json`

```json
{
  "in_allowlist": true
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `in_allowlist` | bool | 当前租户是否已开通指定功能 |

`passwordless-login` 返回 `false` 时，需要先申请并开通免登录功能白名单。

**与 `GET /local-agent/availability` 的区别**：

| 维度 | `/local-agent/availability`（用户视角） | `/admin/feature-allowlist/check`（管控视角） |
|---|---|---|
| 鉴权 | 登录用户 | 管理员 |
| 决策因子 | 功能开通状态 + 站点开关 | 功能开通状态 |
| 返字段 | `enabled` 单一 bool | `in_allowlist` 单一 bool |
| 用途 | 前端是否展示本地 Agent 入口 | 管控页诊断「当前租户在白名单里吗」 |

- **错误响应：**
  - `400 {"error": "参数 type 不能为空"}` / `400 {"error": "不支持的白名单类型: <type>"}`
  - `401 {"error": "未登录"}`
  - `405 {"error": "method not allowed"}` — 非 GET
  - `500 {"error": "查询白名单失败"}`

### `GET /admin/audit`

分页查询审计日志，支持按用户、操作类型、资源 ID 和时间范围筛选。接口保持原有同步总数响应；操作记录页面应先通过用户列表模糊选择具体用户，再使用稳定的 `user_id` 精确查询审计记录。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码（默认 1） |
| page_size | int | 否 | 每页条数（默认 20，最大 1000） |
| user_id | uint | 否 | 按操作者用户 ID 精确筛选；显式传 `0` 时筛选 Token 认证等 user_id 为 0 的记录 |
| username | string | 否 | 按完整用户名精确筛选；默认使用等值查询，可利用 `(identifier, username)` 联合索引 |
| fuzzy | string | 否 | 传 `1` 时将 `username` 改为包含匹配（`LIKE %username%`）；不传或传其他值时保持精确匹配 |
| action | string | 否 | 按操作类型前缀筛选（如 `agent_bridge_` 可筛选所有 Agent-Bridge 审计记录） |
| resource_id | string | 否 | 按资源 ID 精确筛选（如按实例 ID 查询某台机器的所有审计记录） |
| start_time | int64 | 否 | 起始时间（Unix 秒时间戳）。不传则不限制起始时间 |
| end_time | int64 | 否 | 终止时间（Unix 秒时间戳）。不传则不限制终止时间 |

- **响应：** `{"logs": [...], "page": 1, "page_size": 20, "total": 100, "total_pages": 5}`。每次请求先按相同筛选条件同步统计精确总数，再返回当前页记录。

同时传 `user_id`、`username` 和其他筛选时使用 AND 语义。既有 API 调用方继续以 `username=<完整用户名>` 调用即可，参数形式不变且查询由模糊改为精确；此前依赖部分用户名匹配的调用方需显式增加 `fuzzy=1`。单独传 `fuzzy=1` 而不传 `username` 时不增加筛选条件。

前端操作人选择契约：

1. 页面首次进入时调用 `GET /admin/audit?page=1&page_size=<N>`，直接使用响应中的 `logs`、`total` 和 `total_pages`。
2. 操作人搜索框输入变化时，调用 `GET /admin/users?username=<输入>&fuzzy=1&page=1&page_size=20` 获取候选用户；建议约 300 ms 防抖并忽略过期响应。
3. 候选用户对象的实际字段为 `ID` / `Username`；下拉框展示 `Username`，用户选定后保存 `ID`。
4. 选定用户后调用 `GET /admin/audit?user_id=<候选对象.ID>&page=1&page_size=<N>`，查询该用户全部历史用户名下的操作记录及精确总数。
5. 清空选中用户时移除 `user_id`，恢复全部记录；导出流程复用同一个 `user_id` 和其他筛选条件。
6. 页面字段应标为“操作人”或“用户名”，不要将展示的用户名标为“用户 ID”。`user_id=0` 表示 Token/系统主体，不应作为普通用户候选项。

性能边界：

- 首页无用户筛选时，总数查询使用当前租户的 `identifier` 条件，耗时随该租户记录数增长。
- 选定用户后，列表和总数查询可使用 `(identifier, user_id)` 联合索引，只扫描该用户范围；完整用户名查询可使用 `(identifier, username)` 联合索引。
- 显式使用 `username=<keyword>&fuzzy=1` 时仍可能扫描当前租户的大量记录，因此新页面必须使用候选选择后的 `user_id`，不应以 audit 模糊查询驱动主流程。

- **错误响应：**
  - `400`：`user_id` 不是合法无符号整数或发生溢出
  - `401` / `403`：未登录或非管理员
  - `500`：总数或列表数据库查询失败

审计日志对象字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 日志 ID |
| started_at | string | 操作开始时间（ISO 8601） |
| created_at | string | 操作结束时间（ISO 8601） |
| user_id | uint | 操作者用户 ID（Token 认证为 0） |
| username | string | 操作者用户名（Token 认证为 `admin-token`） |
| action | string | 操作类型 |
| resource | string | 资源类型 |
| resource_id | string | 资源标识 |
| status | string | `success` 或 `failed` |

操作类型（action）枚举：

- **用户管理：** `user_create`、`user_batch_create`、`user_delete`、`user_hard_delete`、`user_restore`、`user_reset_password`、`user_update`、`user_export_tokens`、`user_change_password`
- **配置管理：** `config_update`、`cvm_config_update`、`template_update`、`security_group_update`、`security_group_policy_update`
- **模型管理：** `model_create`、`model_update`、`model_delete`、`model_toggle`、`model_visibility_update`
- **通道管理：** `channel_toggle`、`channel_add`、`channel_delete`
- **镜像管理：** `image_import`、`image_delete`、`image_toggle`
- **实例操作：** `instance_create`、`instance_delete`、`instance_reboot`、`instance_reset`、`instance_set_model`、`instance_set_channel`、`instance_del_channel`、`instance_auto_channel`、`instance_add_skill`、`instance_approve`、`instance_set_gateway_ui`
- **OneID SSO：** `oneid_login`、`oneid_logout`、`oneid_event`
- **云 API 透传：** `cloud_proxy_{Action}`（动态生成，如 `cloud_proxy_OpenClsService`、`cloud_proxy_RunInstances`）
- **Agent-Bridge TAT 审计：** `agent_bridge_*`（由 Agent-Bridge 回调写入，如 `agent_bridge_desktop_install`、`agent_bridge_script_exec`）

### `GET /admin/cls/service`

> **❗ 此接口已废弃，代码中不存在此路由。** CLS 服务状态请使用 `GET /admin/cls/status` 接口查询。

- **权限：** 管理员
- **参数：** 无
- **响应：** 始终 JSON
  - 已开通：`{"activated": true, "total_logsets": 5}`
  - 未开通/查询失败：`{"activated": false, "error": "ServiceNotActivated: ..."}`
  - 客户端创建失败：`500 {"error": "创建 CLS 客户端失败: ..."}`

### `GET /admin/cls/status`

查询 CLS ClawPro 相关主题的详细状态信息。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| biz_type | uint | 否 | 主题业务类型：`0`=日志主题（默认），`1`=指标主题 |

- **响应：** `application/json`（始终 JSON）

成功：

```json
{
  "ok": true,
  "biz_type": 0,
  "total_count": 1,
  "topics": [
    {
      "TopicId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "TopicName": "主题名称",
      "LogsetId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "BizType": 0,
      "PartitionCount": 1,
      "Index": true,
      "Status": "ok",
      "AssumerUin": 100000000001,
      "AssumerName": "ClawPro",
      "CreateTime": "2026-03-01 12:00:00",
      "RoleName": "CVM_QCSLinkedRoleInClawProAgent",
      "AutoSplit": true,
      "MaxSplitPartitions": 50,
      "StorageType": "hot",
      "Period": 30,
      "SubAssumerName": "",
      "Describes": "主题描述",
      "HotPeriod": 30,
      "IsWebTracking": false,
      "Tags": []
    }
  ]
}
```

失败时返回错误信息：

- `500 {"error": "创建 CLS Client 失败: ..."}`
- `500 {"error": "查询 CLS 主题失败: ..."}`

### `GET /admin/check-role`

检查当前账号是否存在 `CVM_QCSLinkedRoleInClawProAgent` 服务角色。通过 STS AssumeRole 尝试扮演该角色，成功则角色存在，失败则角色不存在。

- **权限：** 管理员
- **响应：** `application/json`（始终 JSON）

成功（角色存在）：

```json
{
  "has_role": true,
  "message": "角色 CVM_QCSLinkedRoleInClawProAgent 存在"
}
```

角色不存在：

```json
{
  "has_role": false,
  "message": "AssumeRole 错误信息"
}
```

失败时返回错误信息：

- `500 {"error": "创建 STS 客户端失败: ..."}`

### `POST /admin/cls/open`

> 内部测试用，实际不会调用！

确认当前账号是否已开通 CLS 日志服务，若未开通则自动开通，并调用 `OpenClawService` 获取 Topic 信息。**该接口记录审计日志。**

- **权限：** 管理员
- **请求体：** `application/json`（可选，空 body 时等同于全量模式）
- **响应：** `application/json`（始终 JSON）

**请求体字段：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| scope_type | string | 可选 | 采集范围模式，枚举值：`"all"`=全量模式，`"group"`=分组模式；未传时根据 `group_ids` 自动推断：有值则为 `"group"`，否则为 `"all"` |
| group_ids | []uint | 可选 | 分组 ID 列表，仅 `scope_type="group"` 时生效；`scope_type="all"` 时忽略此字段 |

请求示例（空 body，等同于全量模式）：

```json
{}
```

请求示例（全量模式）：

```json
{
  "scope_type": "all"
}
```

请求示例（分组模式，显式指定）：

```json
{
  "scope_type": "group",
  "group_ids": [1, 2, 3]
}
```

请求示例（分组模式，省略 scope_type，自动推断）：

```json
{
  "group_ids": [1, 2, 3]
}
```

流程：
1. 若 `scope_type` 为空，根据 `group_ids` 是否有值自动推断（有值→`"group"`，否则→`"all"`）
2. 校验 `scope_type`，不合法时返回 400
3. 调用 `GetClsService` 查询 CLS 服务开通状态
4. 如果未开通，调用 `OpenClsService` 开通服务
5. 调用 `OpenClawService`（Tag=ClawPro）获取日志主题和指标主题 ID
6. 更新 `SiteConfig` 的 `CLSEnabled=1`、`CLSScopeMode=scope_type` 字段
7. `scope_type="group"` 时，将 `group_ids` 写入 CLS 采集范围，并标记目标实例为待安装

成功：

```json
{
  "opened": true,
  "message": "CLS 日志服务已开通",
  "request_id": "xxx",
  "topic_id": "日志主题 ID",
  "metric_topic_id": "指标主题 ID"
}
```

失败时返回错误信息：

- `400 {"error": "scope_type 必须为 \"all\" 或 \"group\""}`
- `400 {"error": "分组数量不能超过 N"}`
- `400 {"error": "分组不存在: ..."}`
- `500 {"error": "创建 CLS CommonClient 失败: ..."}`
- `500 {"error": "查询 CLS 服务状态失败: ..."}`
- `500 {"error": "开通 CLS 服务失败: ..."}`
- `500 {"error": "调用 OpenClawService 失败: ..."}`

### `GET /admin/cls/scope`

查询当前 CLS 采集范围配置，包含各分组的实例数量及安装状态统计。

- **权限：** 管理员
- **参数：** 无
- **响应：** `application/json`（始终 JSON）

**响应字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| cls_enabled | bool | CLS 服务是否已开启 |
| scope_type | string | 采集范围模式：`"all"`=全量、`"group"`=分组、`"off"`=未开启 |
| group_ids | []uint | 当前配置的分组 ID 列表 |
| groups | []object | 各分组详情（见下表），未配置分组时为空数组 |
| total_instance_count | number | scope 范围内实例总数（跨分组去重） |
| total_install_stats | object | 全量安装状态统计 |

**groups 子对象字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | number | 分组 ID |
| name | string | 分组名称 |
| full_path | string | 分组完整路径（含父级） |
| source | string | 分组来源 |
| descendant_count | number | 子孙分组数量（不含自身） |
| instance_count | number | 该分组（含子孙）下的实例数量 |
| install_stats | object | 该分组实例的安装状态统计 |

**install_stats 字段：**

| 字段 | 说明 |
|------|------|
| not_installed | 未安装 |
| installing | 安装中 |
| installed | 已安装 |
| uninstalling | 卸载中 |
| skipped | 已跳过 |

成功（CLS 未开启或未配置分组）：

```json
{
  "ok": true,
  "cls_enabled": false,
  "scope_type": "off",
  "group_ids": [],
  "groups": []
}
```

成功（分组模式）：

```json
{
  "ok": true,
  "cls_enabled": true,
  "scope_type": "group",
  "group_ids": [1, 2],
  "groups": [
    {
      "id": 1,
      "name": "研发组",
      "full_path": "根组/研发组",
      "source": "manual",
      "descendant_count": 3,
      "instance_count": 12,
      "install_stats": {
        "not_installed": 2,
        "installing": 1,
        "installed": 8,
        "uninstalling": 0,
        "skipped": 1
      }
    }
  ],
  "total_instance_count": 12,
  "total_install_stats": {
    "not_installed": 2,
    "installing": 1,
    "installed": 8,
    "uninstalling": 0,
    "skipped": 1
  }
}
```

失败时返回错误信息：

- `500 {"error": "查询 CLS 采集范围失败: ..."}`

### `POST /admin/cls/scope`

更新 CLS 采集范围，自动对新增/移除分组下的实例触发安装或卸载流程。**该接口记录审计日志。**

- **权限：** 管理员
- **请求体：** `application/json`（必须携带，不可为空）
- **响应：** `application/json`（始终 JSON）

**请求体字段：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| scope_type | string | **必填** | 采集范围模式：`"all"`=全量模式（忽略 group_ids），`"group"`=分组模式 |
| group_ids | []uint | 可选 | 分组 ID 列表，`scope_type="group"` 时生效；可为空数组（不安装任何机器） |

请求示例（全量模式）：

```json
{
  "scope_type": "all"
}
```

请求示例（分组模式）：

```json
{
  "scope_type": "group",
  "group_ids": [1, 2, 3]
}
```

成功：

```json
{
  "ok": true,
  "added_groups": [3],
  "removed_groups": [1],
  "added_instance_count": 5,
  "removed_instance_count": 2
}
```

**响应字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| added_groups | []number | 本次新增的分组 ID 列表 |
| removed_groups | []number | 本次移除的分组 ID 列表 |
| added_instance_count | number | 新增分组下触发安装的实例数量（已跳过活跃状态实例） |
| removed_instance_count | number | 移除分组下将被卸载的实例数量（仅统计不再属于任何 scope 分组的独占实例，后台异步处理） |
| warnings | []string | （可选）非致命警告信息 |

> **注意：**
> - 新增分组的实例安装为**异步**操作，标记待安装后由后台定时任务拾取执行，`added_instance_count` 仅供展示参考。
> - 移除分组的实例卸载同样为**异步**操作，由后台 `runCLSAgentScopeUninstall` 定时任务处理，`removed_instance_count` 仅供展示参考。
> - 多归属实例（同时属于多个 scope 分组）不会被误卸载。

失败时返回错误信息：

- `400 {"error": "CLS 服务未开启，请先开启 CLS 服务"}`
- `400 {"error": "scope_type 必须为 \"all\" 或 \"group\""}`
- `400 {"error": "分组数量不能超过 N"}`
- `400 {"error": "分组不存在: ..."}`
- `500 {"error": "更新 CLS 采集范围失败: ..."}`

### `GET /admin/cls/update`

查询已安装 CLS Agent 的实例的插件版本分布统计，以及各实例的版本明细。doctor 节点和未安装实例不计入结果。

- **权限：** 管理员
- **参数：** 无
- **响应：** `application/json`（始终 JSON）

**响应字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| v1_count | number | 插件版本为 `1.0` 或 `"updating"` 的实例数量（旧版或升级中，均视为未完成） |
| v2_count | number | 插件版本为 `2.0`（新版，含 trace 配置）的实例数量 |
| instances | []object | 各实例版本明细（仅含已安装 CLS Agent 的非 doctor 节点实例） |

**instances 子对象字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| instance_id | string | CVM 实例 ID |
| name | string | 实例名称 |
| cls_plugin_version | string | CLS 插件版本，取值见下表；数据库为空时归一化为 `"1.0"` |

**cls_plugin_version 取值说明：**

| 值 | 说明 |
|----|------|
| `"1.0"` | 旧版插件，无 trace 配置，待升级 |
| `"updating"` | 正在升级中（升级任务已下发，尚未完成）；前端可展示"升级中"状态 |
| `"2.0"` | 新版插件，含 trace 配置，已完成升级 |

成功：

```json
{
  "ok": true,
  "v1_count": 3,
  "v2_count": 7,
  "instances": [
    {
      "instance_id": "ins-aabbccdd",
      "name": "生产服务器-01",
      "cls_plugin_version": "1.0"
    },
    {
      "instance_id": "ins-bbccddee",
      "name": "生产服务器-02",
      "cls_plugin_version": "updating"
    },
    {
      "instance_id": "ins-eeffgghh",
      "name": "生产服务器-03",
      "cls_plugin_version": "2.0"
    }
  ]
}
```

失败时返回错误信息：

- `401 {"error": "未授权"}`
- `500 {"error": "查询实例版本信息失败: ..."}`

### `POST /admin/cls/update`

对已安装 CLS Agent 的实例批量执行插件升级。无论 `scope_type` 为何值，均升级当前用户下**全部**已安装 CLS Agent 的机器。根据实例当前的 trace 配置状态，决定仅更新版本号还是重新安装插件。**该接口记录审计日志。**

- **权限：** 管理员
- **请求体：** `application/json`（必须携带，不可为空）
- **响应：** `application/json`（始终 JSON）

> ⚠️ **注意：** 该接口为**同步**操作，会等待所有实例处理完成后才返回。实例数量较多时请求耗时可能较长，建议前端设置较长的超时时间（如 10 分钟）。

**请求体字段：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| scope_type | string | **必填** | `"all"` 或 `"group"`，当前版本两者效果相同，均升级全部机器 |

> **注意：** `group_ids` 字段保留兼容性，但不影响升级范围，始终升级全部已安装 CLS Agent 的机器。

请求示例：

```json
{
  "scope_type": "all"
}
```

成功：

```json
{
  "ok": true,
  "total": 10,
  "skipped": 3,
  "upgraded": 5,
  "reinstalled": 2,
  "failed": 0
}
```

无需处理时：

```json
{
  "ok": true,
  "total": 0,
  "skipped": 0,
  "upgraded": 0,
  "reinstalled": 0,
  "failed": 0,
  "message": "没有需要更新的实例"
}
```

**响应字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| total | number | 本次查询到的待处理实例总数（已排除 `2.0` 和 `updating` 状态） |
| skipped | number | 跳过的实例数：非 RUNNING 状态，或已被并发请求抢先处理 |
| upgraded | number | trace 已配置，仅更新版本号为 `2.0` 的实例数 |
| reinstalled | number | trace 未配置，执行 npx 插件重新安装后更新为 `2.0` 的实例数 |
| failed | number | 处理失败的实例数（版本已回滚为 `1.0`，可重试） |
| message | string | （可选）无需处理时的提示信息 |

**升级逻辑：**

```
查询全部已安装 CLS Agent 且版本非 "2.0"/"updating" 的实例（过滤 doctor 节点）
  │
  ├─ 通过 CVM API 过滤非 RUNNING 实例 → 直接跳过（skipped +1）
  │
  └─ 对每个 RUNNING 实例（最大并发数 5）：
       ├─ CAS：将版本从 "1.0" → "updating"（WHERE cls_plugin_version = "1.0"）
       │    └─ RowsAffected = 0 → 已被并发请求抢先处理，跳过（skipped +1）
       │
       ├─ 下发 cls_check_trace.sh，读取 ~/.openclaw/openclaw.json
       │    ├─ trace.enabled = true 且 traceTopicId 非空（configured = true）
       │    │    → 更新数据库版本为 "2.0"（upgraded +1）
       │    └─ 否则（configured = false）
       │         → 通过 npx 卸载并重新安装 clawpro-diagnostics-metrics-cls-onboard-cli 插件
       │              ├─ 成功 → 更新版本为 "2.0"（reinstalled +1）
       │              └─ 失败 → 回滚版本为 "1.0"（failed +1，可重试）
       │
       └─ 任意步骤失败 → 回滚版本为 "1.0"（failed +1）
```

**防重复触发机制：**

- 查询时过滤 `"updating"` 状态，连续调用不会重复处理同一批实例
- 每个实例处理前通过 CAS 原子性地将版本标记为 `"updating"`，即使并发请求同时查到同一实例，也只有一个能成功标记
- 失败时版本回滚为 `"1.0"`，允许下次重新触发升级

失败时返回错误信息：

- `400 {"error": "CLS 服务未开启，请先开启 CLS 服务"}`
- `400 {"error": "scope_type 必须为 \"all\" 或 \"group\""}`
- `500 {"error": "查询目标实例失败: ..."}`
- `500 {"error": "获取 CLS 服务信息失败: ..."}`

### `POST /admin/cls/close`

关闭 CLS 日志服务（查询并清理相关资源）。**该接口记录审计日志。**

- **权限：** 管理员
- **响应：** `application/json`（始终 JSON）

流程：
1. 调用 `DescribeTopics`（Filter: `assumerName=ClawPro`）分别以 `BizType=0`（日志主题）和 `BizType=1`（指标主题）查询 ClawPro 相关主题
2. 如果两次查询都没有查到主题，说明服务已关闭，清空 DB 中的 `CLSEnabled`，直接返回
3. 如果查到日志主题（`BizType=0`），提取 `TopicId`，调用 `DeleteTopic` 删除
4. 如果查到指标主题（`BizType=1`），提取 `MetricTopicId`，调用 `DeleteTopic` 删除
5. 更新 `SiteConfig.CLSEnabled` 为 0

服务已关闭（无需操作）：

```json
{
  "message": "CLS ClawPro 服务未开通，无需关闭"
}
```

成功删除主题：

```json
{
  "message": "已删除 2 个主题",
  "deleted_topics": ["topic-id-1", "topic-id-2"]
}
```

失败时返回错误信息：

- `500 {"error": "创建 CLS Client 失败: ..."}`
- `500 {"error": "查询 CLS 日志主题失败: ..."}`
- `500 {"error": "查询 CLS 指标主题失败: ..."}`
- `500 {"error": "删除日志主题 xxx 失败: ..."}`
- `500 {"error": "删除指标主题 xxx 失败: ..."}`

---

### Memory TDAI 记忆插件管理

### `GET /admin/memory-tdai/config`

查询 Memory TDAI 配置与实例插件状态统计。

- **权限：** 管理员
- **响应：** 始终 JSON

成功：

```json
{
  "config": {
    "memory_tdai_enable": true,
    "memory_tdai_supported_versions": ["1.0.0", "1.1.0"],
    "stats": {
      "ENABLED": 10,
      "NOT_INSTALLED": 3,
      "FAILED": 1
    }
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| config.memory_tdai_enable | bool | 总开关 |
| config.memory_tdai_supported_versions | string[] | 支持版本列表（站点维度，前端用于判断是否展示功能入口）。空数组表示未配置 |
| config.stats | object | 实例插件状态统计（按 status 分组计数），key 为状态枚举，value 为数量 |

状态枚举值：`NOT_INSTALLED`、`ENABLING`、`ENABLED`、`DISABLING`、`DISABLED`、`FAILED`

### `PUT /admin/memory-tdai/config`

更新 Memory TDAI 总开关。只更新开关落库，不同步触发任务（后台定时轮询自动处理不一致实例）。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **请求体：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| memory_tdai_enable | bool | 是 | 总开关 |

请求示例：

```json
{"memory_tdai_enable": true}
```

- **JSON 响应：**
  - 成功：`{"ok": true, "memory_tdai_enable": true}`
  - 失败：`400 {"error": "memory_tdai_enable 为必填字段"}` / `400 {"error": "请求体格式错误: ..."}` / `500 {"error": "更新 memory_tdai_enable 失败: ..."}`

> **后台行为说明：** 开关更新后，后台定时轮询任务（`StartMemoryTDAITask`，默认每 60 秒）会自动检测不一致实例并执行启用或禁用操作。每个实例最多重试 3 次，超限标记为 `FAILED` 不再处理。

---

### `GET /admin/cloud`

列出所有已注册的腾讯云 API 透传服务及其可用 Actions（按读/写分类）。

- **权限：** 管理员
- **响应：** 始终 JSON

```json
{
  "data": {
    "cvm": {
      "service": "cvm",
      "version": "2017-03-12",
      "read_actions": ["DescribeInstances", "CheckOpenClawRole"],
      "write_actions": ["AssociateSecurityGroups", "DisassociateSecurityGroups", "StopInstances"]
    },
    "vpc": {
      "service": "vpc",
      "version": "2017-03-12",
      "read_actions": ["DescribeSecurityGroups", "DescribeSecurityGroupPolicies"],
      "write_actions": ["DeleteSecurityGroupPolicies", "DeleteSecurityGroup", "CreateSecurityGroupWithPolicies", "CreateSecurityGroupPolicies"]
    },
    "cls": {
      "service": "cls",
      "version": "2020-10-16",
      "read_actions": ["DescribeLogsets", "DescribeTopics", "SearchLog", "QueryMetric", "QueryRangeMetric", "GetClsService", "DescribeRainbowConfigs"],
      "write_actions": ["OpenClsService", "OpenClawService", "DeleteTopic"]
    },
    "billing": {
      "service": "billing",
      "version": "2018-07-09",
      "read_actions": ["DescribeMeasureResources"],
      "write_actions": ["CreateOrdersAndPay"]
    },
    "csip": {
      "service": "csip",
      "version": "2022-11-21",
      "read_actions": [
        "DescribeAIAgentAssetList", "DescribeABTestConfig", "DescribeOrganizationInfo",
        "DescribePayInfo", "DescribeUserAccountInfo", "DescribeUserOperationPermission",
        "GetLocalStorageItem", "DescribeAgentlessVulAssetDetail", "DescribeExposeRules",
        "DescribeExposeAssetCategory", "DescribeExposePath", "DescribeCVMAssets",
        "DescribeAssetProcessList", "DescribeVulRiskList", "DescribeHighBaseLineRiskList",
        "DescribeExposures"
      ],
      "write_actions": ["SetLocalStorageItem"]
    },
    "cwp": {
      "service": "cwp",
      "version": "2018-02-28",
      "read_actions": [
        "DescribeLicenseWhiteConfig", "DescribeOrderList", "DescribeLogStorageConfig",
        "DescribeLicenseBindSchedule", "DescribeBashEventsNew", "DescribeRiskDnsEventList",
        "DescribeVersionStatistics", "DescribeBashEventsInfoNew", "DescribeMachines",
        "DescribeRiskDnsEventInfo", "DescribeBashPolicies", "DescribeMalWareList",
        "DescribeRiskDnsPolicyList", "DescribeSkillInfo", "DescribeImportMachineInfo",
        "DescribeRiskBatchStatus", "DescribeTags", "DescribeMachineRegionList",
        "DescribeLicenseGeneral", "GetLocalStorageItem", "DescribeMachineGeneral",
        "DescribeLogHistogram", "DescribeLogStorageStatistic", "SearchLog",
        "DescribeMachineInfo", "DescribeMalwareInfo"
      ],
      "write_actions": [
        "CreateWhiteListOrder", "ModifyLicenseBinds", "ModifyLogStorageConfig",
        "ScanAsset", "SyncAssetScan", "ModifyRiskEventsStatus", "SetLocalStorageItem",
        "RemoveLocalStorageItem", "ModifyBashPolicyStatus", "DeleteBashPolicies",
        "ModifyBashPolicy", "CheckBashPolicyParams", "ModifyRiskDnsPolicy",
        "ModifyRiskDnsPolicyStatus", "DeleteRiskDnsPolicy", "ModifyReverseShellRulesAggregation"
      ]
    }
  }
}
```

> 详细说明参见 [Cloud Proxy 透传接口文档](cloud_proxy_api.md)

### `POST /admin/cloud/query/{service}`

腾讯云 API **只读查询**透传接口。将请求透传到指定云产品的查询类 API（如 Describe/Inquiry），后端自动完成凭证注入与签名。不记录审计日志。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **路径参数：** `{service}` — 云产品标识，如 `cvm`、`vpc`、`cls`、`billing`、`csip`、`cwp`
- **请求 Header：**

| Header | 必选 | 说明 |
|--------|------|------|
| `X-TC-Action` | 是 | 要调用的 API 名称（必须在该 service 的读白名单中） |
| `X-TC-Version` | 否 | API 版本号，默认使用注册表中的版本 |

- **请求体：** 与腾讯云官方 API 对应接口的请求参数完全一致的 JSON，无参数时可传 `{}`
- **响应：** 原样透传腾讯云 API 响应 JSON
- **错误响应：**
  - `400 {"error": "缺少 X-TC-Action Header 或 Action 参数"}`
  - `400 {"error": "不支持的 service: xxx, 可用: cvm, vpc, ..."}`
  - `403 {"error": "Action \"xxx\" 不在 {service} 的读接口白名单中"}`

> 详细说明、已注册 Actions 列表及调用示例参见 [Cloud Proxy 透传接口文档](cloud_proxy_api.md)

### `POST /admin/cloud/mutate/{service}`

腾讯云 API **变更操作**透传接口。将请求透传到指定云产品的写类 API（如 Create/Delete/Modify/Open），后端自动完成凭证注入与签名。**所有调用均记录审计日志**。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **路径参数：** `{service}` — 云产品标识，如 `cvm`、`vpc`、`cls`、`billing`、`csip`、`cwp`
- **请求 Header：**

| Header | 必选 | 说明 |
|--------|------|------|
| `X-TC-Action` | 是 | 要调用的 API 名称（必须在该 service 的写白名单中） |
| `X-TC-Version` | 否 | API 版本号，默认使用注册表中的版本 |

- **请求体：** 与腾讯云官方 API 对应接口的请求参数完全一致的 JSON，无参数时可传 `{}`
- **响应：** 原样透传腾讯云 API 响应 JSON
- **错误响应：**
  - `400 {"error": "缺少 X-TC-Action Header 或 Action 参数"}`
  - `400 {"error": "不支持的 service: xxx, 可用: cvm, vpc, ..."}`
  - `403 {"error": "Action \"xxx\" 不在 {service} 的写接口白名单中"}`

> 详细说明、已注册 Actions 列表及调用示例参见 [Cloud Proxy 透传接口文档](cloud_proxy_api.md)

---

## 六、用户用量查询

### `GET /quota/data`

当前用户的用量查询 JSON API。参数与 `/admin/usage/data` 类似，但 `user_id` 强制为当前登录用户，`group_by=user` 被忽略。

- **权限：** 已登录用户（Session Cookie）
- **Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 起始日期（`YYYY-MM-DD`），默认今天 |
| end_date | string | 否 | 终止日期（`YYYY-MM-DD`），默认今天 |
| group_by | string | 否 | 聚合维度，逗号分隔：`date`/`model`/`instance`（不支持 `user`）。默认 `date,model` |
| ai_model_id | uint | 否 | 筛选指定模型 |
| id | uint | 否 | 筛选指定实例（`instances.id` 主键）；优先级高于 `instance_id` |
| instance_id | string | 否 | 筛选指定实例（CVM ID 字符串，如 `ins-abc123`）；为兼容旧调用方，也接受纯数字（此时同 `id`）。仅可指向当前登录用户名下的实例 |
| group_id | uint | 否 | 按 agent 绑定的分组过滤用量数据，并按该分组上下文解析用户 Token 配额规则 |
| order_by | string | 否 | 排序字段：`total_tokens`（默认）或 `request_count` |
| order | string | 否 | `desc` 按 `order_by` 字段降序排列，默认不排序 |

- **JSON 响应：**

```json
{
  "quota_day": 50000,
  "quota_period": "day",
  "token_quota_rules": [
    {"mode": "day", "limit": 50000}
  ],
  "token_quota_usages": [
    {"rule_index": 0, "used": 620, "period_start": 1773244800, "period_end": 1773331200, "active": true}
  ],
  "global_token_quota_rules": [
    {"mode": "day", "limit": 1000000}
  ],
  "global_token_quota_usages": [
    {"rule_index": 0, "used": 620, "period_start": 1773244800, "period_end": 1773331200, "active": true}
  ],
  "start_date": "2026-03-12",
  "end_date": "2026-03-12",
  "group_by": ["instance"],
  "rows": [
    {
      "instance_id": 3,
      "instance_name": "my-instance",
      "instance_cvm_id": "ins-abc123",
      "prompt_tokens": 500,
      "completion_tokens": 120,
      "total_tokens": 620,
      "prompt_cache_read_tokens": 200,
      "prompt_cache_write_tokens": 50,
      "request_count": 5
    }
  ]
}
```

> `quota_day` 为兼容旧字段，由当前有效 `token_quota_rules` 中的 `day` 规则反推；没有 day 规则时为 `-1`。`quota_period` 为站点全局 Token 配额兼容周期字段，会从站点全局 day/month 规则反推；仅有 year/custom 规则时旧字段无法表达，返回保存的 day/month 值。
>
> `token_quota_rules` 表示当前用户实际 Token 配额策略：未传 `group_id` 时使用用户自身有效规则；传 `group_id` 时与 LLM proxy 一致，优先使用该组/祖先组的分组用户 Token 规则，未命中时回退到站点默认用户规则。
>
> `token_quota_usages` 与 `global_token_quota_usages` 均按规则当前生效窗口汇总 `LLMUsageLog.total_tokens`，元素的 `rule_index` 对应同名 rules 数组下标；`period_start` / `period_end` 为当前窗口的 Unix 秒时间范围。`active=false` 表示当前无生效窗口，兼容返回 `period_start=0`、`period_end=0`；无终止的生效窗口返回 `period_end=null`。

- **错误响应：**
  - `400 {"error": "order_by 参数无效，仅支持 total_tokens 或 request_count"}`
  - `500 {"error": "查询用量数据失败"}`

### `GET /quota/logs`

当前用户的使用明细记录查询。参数与 `/admin/usage/logs` 类似，但 `user_id` 强制为当前登录用户。

- **权限：** 已登录用户（Session Cookie）
- **Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 起始日期（`YYYY-MM-DD`），默认今天 |
| end_date | string | 否 | 终止日期（`YYYY-MM-DD`），默认今天 |
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页数量 |
| ai_model_id | uint | 否 | 筛选指定模型 |
| id | uint | 否 | 筛选指定实例（`instances.id` 主键）；优先级高于 `instance_id` |
| instance_id | string | 否 | 筛选指定实例（CVM ID 字符串，如 `ins-abc123`）；为兼容旧调用方，也接受纯数字（此时同 `id`）。仅可指向当前登录用户名下的实例 |

- **响应字段（每条 log）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 记录 ID |
| provider | string | 模型提供商 |
| model | string | 模型名称 |
| prompt_tokens | int | 输入 tokens |
| completion_tokens | int | 输出 tokens |
| total_tokens | int | 总 tokens |
| prompt_cache_read_tokens | int | 命中/读取 prompt cache 的 tokens |
| prompt_cache_write_tokens | int | 创建/写入 prompt cache 的 tokens |
| status_code | int | HTTP 状态码 |
| latency | int | 耗时（毫秒） |
| created_at | string | 请求时间（RFC 3339，如 `2026-03-14T14:30:00Z`） |

- **JSON 响应：**

```json
{
  "start_date": "2026-03-12",
  "end_date": "2026-03-12",
  "page": 1,
  "page_size": 50,
  "total": 28,
  "logs": [
    {
      "id": 1,
      "provider": "openai",
      "model": "gpt-4",
      "prompt_tokens": 500,
      "completion_tokens": 120,
      "total_tokens": 620,
      "prompt_cache_read_tokens": 200,
      "prompt_cache_write_tokens": 50,
      "status_code": 200,
      "latency": 1230,
      "created_at": "2026-03-12 10:30:00"
    }
  ]
}
```

---

## 七、LLM 代理

OpenAI 兼容的 LLM 代理接口。使用实例的 `proxy_token` 进行 Bearer Token 认证（非 Session Cookie、非 admin-token）。

### `POST /v1/chat/completions`

Chat Completions 代理。将请求转发到实例绑定的 AI 模型后端，并记录 token 用量。

- **权限：** 实例 Bearer Token（`Authorization: Bearer <proxy_token>`）
- **Content-Type：** `application/json`
- **请求体：** OpenAI Chat Completions 格式，支持 `stream: true`
- **配额检查：**
  1. **全局每日配额**（`SiteConfig.GlobalTokenQuotaDay`）：全站所有用户所有模型当日总用量超限时返回 `429`
  2. **全局模型配额**（`AIModel.QuotaDay`）：该模型所有实例当日总用量超限时返回 `429`
  3. **用户每日配额**（`User.TokenQuotaDay`）：该用户所有模型当日总用量超限时返回 `429`
- **max_tokens 限制：** 若实例设置了 `MaxTokens`，请求中的 `max_tokens` 将被强制限制为不超过该值
- **响应：**
  - 成功：透传上游 AI 模型的 JSON 响应（非流式）或 SSE 流（流式）
  - `401 {"error": {"message": "Missing or invalid API key", ...}}`
  - `400 {"error": {"message": "No active model configured for this instance", ...}}`
  - `429 {"error": {"message": "Global daily token quota exceeded", ...}}` — 全局配额超限
  - `429 {"error": {"message": "Model daily token quota exceeded", ...}}` — 模型配额超限
  - `429 {"error": {"message": "User daily token quota exceeded", ...}}` — 用户配额超限
  - `502 {"error": {"message": "Failed to connect to LLM backend: ...", ...}}`

> 错误响应遵循 OpenAI 格式：`{"error": {"message": "...", "type": "error", "code": <status>}}`

### `GET /v1/models`

列出实例可用的模型。

- **权限：** 实例 Bearer Token
- **响应：** OpenAI Models List 格式

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4",
      "object": "model",
      "created": 1700000000,
      "owned_by": "openai"
    }
  ]
}
```

> 仅返回当前实例绑定的已启用模型（0 或 1 个）。

---

## 八、用户分组管理

用户分组管理接口分为两类：
- **管理员接口**（`/admin/user-groups/*`）：需要管理员角色，支持 API Token 访问（🔓）
- **普通用户接口**（`/user-groups/mine`、`/openclaw/config-overview`）：需要登录

**容量限制：**
- 单平台最多 **1000** 个用户组
- 单用户组最多 **10000** 名成员
- 树最大深度 **10** 层

**架构要点：**
- 加法型资源（model/channel/skill/mcp/image）：祖先链并集，用户可见 = 所属组 + 所有祖先组绑定的资源
- 策略型（policy）：最近祖先覆盖，未配置时回退 site_config 全局默认值

#### 1. 分组树

### `GET /admin/user-groups/tree`

- **权限:** 管理员
- **参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| q | string | 否 | 按 `full_path` / `name` 模糊过滤 |
| with_user_counts | bool | 否 | 是否计算 `direct_user_count` / `descendant_user_count`。**默认 `true`**;显式传 `false` 时全部置 0(追求极致速度) |
| with_health | bool | 否 | 是否计算 `health` 节点配置健康度。默认 `false` |
| with_resource_policy | bool | 否 | 是否返回节点直接绑定的 `direct_resource_policy`；默认 `false`，不返回继承策略 |
| sources | string | 否 | 英文逗号分隔的 source 白名单(`manual,oneid_dept`)。空=全部 |

> `with_user_counts` 曾因 `parseBoolQuery` 默认返回 `false` 与文档不一致。本次 feature 分支已修正为 `parseBoolQueryDefault(..., true)`。

- **响应:**

```json
{
  "ok": true,
  "summary": {
    "total_groups": 8,
    "manual_groups": 6,
    "oneid_dept_groups": 2,
    "to_be_deleted_count": 0,
    "multi_group_users_count": 2,
    "ungrouped_users_count": 3
  },
  "org_tree": [
    {
      "id": 10,
      "name": "技术中心",
      "full_path": "技术中心",
      "parent_id": null,
      "depth": 0,
      "source": "oneid_dept",
      "source_ref": "dept_123",
      "readonly": true,
      "to_be_deleted": false,
      "direct_user_count": 0,
      "descendant_user_count": 12,
      "descendant_count": 3,
      "children": []
    }
  ],
  "user_groups": [
    {
      "id": 2,
      "name": "根组",
      "full_path": "根组",
      "parent_id": null,
      "depth": 0,
      "source": "manual",
      "readonly": false,
      "to_be_deleted": false,
      "direct_user_count": 0,
      "descendant_user_count": 14,
      "descendant_count": 6,
      "direct_resource_policy": {"id": 12, "name": "研发资源策略"},
      "health": { "healthy": true, "missing": [] },
      "children": []
    }
  ],
  "ungrouped": { "user_count": 3 }
}
```

> - `org_tree`:`source=oneid_dept` 的组织架构树
> - `user_groups`:`source=manual` 的自建分组树
> - `ungrouped`:未加入任何组的用户数
> - `health` 仅 `with_health=true` 时返回;检查 4 项:`model / channel / network / imageType`
>   - 任一项全局兜底满足(如模型有 `visibility=all` + `enabled=true` 的记录)→ 对所有组标记通过
>   - 全局不满足时,走闭包表批量查祖先链绑定

---

#### 2. 分组 CRUD

### `GET /admin/user-groups`

- **权限:** 管理员
- **参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| parent_id | uint | 否 | 精确匹配父组 ID(0=根组);省略=不过滤 |
| source | string | 否 | `manual` / `oneid_dept`;省略=不过滤 |
| q | string | 否 | 按 name 模糊 |
| page | int | 否 | 默认 1 |
| page_size | int | 否 | 默认 20,上限 200 |

- **响应:**

```json
{
  "ok": true,
  "total": 48,
  "groups": [
    {
      "id": 9,
      "name": "1组",
      "description": "",
      "parent_id": 4,
      "depth": 2,
      "full_path": "OpenClaw企业版体验/后台组/1组",
      "source": "oneid_dept",
      "source_ref": "1442679816254923756",
      "to_be_deleted": false,
      "readonly": true,
      "member_count": 0,
      "instance_count": 7,
      "created_at": "2026-05-04T12:54:43Z"
    }
  ]
}
```

- **`groups[]` 字段:**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 分组主键 |
| name | string | 分组名称 |
| description | string | 描述 |
| parent_id | uint | 父组 ID,0 = 根组 |
| depth | int | 0 = 根组,上限 10 |
| full_path | string | 父→子拼接(`/` 分隔) |
| source | string | `manual` / `oneid_dept` |
| source_ref | string | OneID 部门 ID(manual 空串);`omitempty` |
| to_be_deleted | bool | OneID 已删但本地仍有绑定时置 `true`(只读标记) |
| readonly | bool | 是否只读(oneid_dept + 未待删时 = true) |
| member_count | int64 | 本组**直属**成员数 |
| instance_count | int64 | 本组 + 所有子孙组下创建的 Agent 总数(通过 `group_closure` 闭包表一次 JOIN 聚合,`ancestor_id=descendant_id` 自指行覆盖"本组"场景) |
| created_at | string | UTC RFC3339 |

---

### `POST /admin/user-groups/create`

- **权限:** 管理员(审计)
- **请求体:**

```json
{ "name": "研发组", "description": "", "parent_id": 2 }
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 64 字符以内,不可含 `/`,不可全空白 |
| description | string | 否 | |
| parent_id | uint / null / 省略 | 否 | null 或省略=根组;uint=挂到该组下 |

- **响应:**

```json
{
  "ok": true,
  "group": {
    "id": 20, "name": "研发组", "description": "",
    "parent_id": 2, "depth": 1, "full_path": "根组/研发组", "source": "manual"
  }
}
```

- **错误:** 400(name 非法 / parent_id 格式错误 / 名字冲突 / 深度超限 / full_path 超 512) / 404(父组不存在) / 403(平台分组上限 2000)

---

### `POST /admin/user-groups/update`

- **权限:** 管理员(审计)
- **请求体(name / description / parent_id 均为可选):**

```json
{ "id": 20, "name": "研发一组", "parent_id": null }
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | |
| name | string (指针) | 否 | 缺失或 null 时不改 |
| description | string (指针) | 否 | 同上 |
| parent_id | uint / null / **省略** | 否 | **三态**:**省略**=不换父;`null`=移到根;uint=挂到该父下 |

> **换父场景用分布式锁 `TryLock("group_move")` 串行化**,冲突时 409。

- **响应:** 同 create 形态,返回更新后 `depth / full_path`
- **错误:** 400(同 create)/ 404(组不存在 / parent 不存在)/ 409(换父锁占用 / name 与同父兄弟冲突)/ 403(只读 oneid_dept 组除 OneID 同步外不可改)

---

### `POST /admin/user-groups/delete`

- **权限:** 管理员(审计)
- **请求体:** `{"id": 20}`
- **行为:** 软删 user_groups 行 + 清成员 + 清 closure。**调用前务必先查 delete-impact,否则 409**。
- **响应:** `{"ok": true}`
- **错误:**
  - 409 `user_group_has_dependencies` - 有 6 张旧表绑定 / 配置绑定表 / manual 子组 / 直属 Agent
  - 404 分组不存在
  - 403 oneid_dept 组由同步维护,不允许通过此接口删除

---

#### 3. 分组成员管理

### `GET /admin/user-groups/members`

- **权限:** 管理员
- **参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 分组 ID；**id=0 代表查询未分组用户** |
| page | int | 否 | 默认 1 |
| page_size | int | 否 | 默认 20,上限 200 |
| include_descendants | bool | 否 | 是否展开子孙(沿闭包表)。默认 false |
| q | string | 否 | 按 `username` 模糊 |

- **响应:**

```json
{
  "ok": true, "total": 5, "page": 1, "page_size": 20,
  "members": [
    {
      "user_id": 19,
      "username": "张三",
      "role": "user",
      "deleted_at": null,
      "direct_groups": [
        { "id": 70, "name": "基础架构组", "full_path": "OpenClaw企业/技术部/后端中心/基础架构组", "source": "oneid_dept", "is_main": true, "instance_count": 2 },
        { "id": 65, "name": "后端中心",   "full_path": "OpenClaw企业/技术部/后端中心",         "source": "oneid_dept", "is_main": false, "instance_count": 0 },
        { "id": 3,  "name": "前端组",     "full_path": "研发中心/前端组",                     "source": "manual",     "is_main": false, "instance_count": 1 }
      ],
      "is_main": true,
      "source": "oneid_dept",
      "joined_at": "2026-05-04T12:54:43Z",
      "from_descendant": false
    }
  ]
}
```

| 响应字段 | 类型 | 说明 |
|---|---|---|
| members[].direct_groups | array | 用户直属的全部组织/分组；子组织带出的成员可从这里取得实际归属的子组织 |
| members[].from_descendant | bool | 用户仅由子孙组织带出且不直属当前节点时为 `true` |

> **`deleted_at` 字段:** null 表示用户正常,非 null(ISO 8601 时间字符串)表示用户已被禁用。与 `/admin/users` 接口的用户状态字段对齐。
>
> **`instance_count` 字段:** `direct_groups[]` 每项新增，表示该用户在该分组下创建的 Agent 数量（`instances.user_id + group_id` 聚合，含所有状态）。
>
> **`from_descendant` 字段:** 仅当 `include_descendants=true` 且用户不是当前组直属成员、而是由子孙组带出时为 `true`。

> **`direct_groups` 排序规则**:
> 1. `source='oneid_dept'` 在前
> 2. 同为 oneid_dept 时,`is_main=true` 的主部门排最前
> 3. 其他按 `full_path` ASC

---

### `POST /admin/user-groups/members/set`

- **权限:** 管理员(审计)
- **请求体:** `{"id": 20, "user_ids": [11, 12, 13]}`
- **行为:** 全量替换组内成员。上限 `MaxMembersPerUserGroup=10000`。
- **约束:** `id` 必填非 0;`oneid_dept` 分组不允许通过此接口变更(只读)。

---

### `POST /admin/user-groups/members/add`

- **权限:** 管理员(审计)
- **请求体:** `{"id": 20, "user_ids": [11]}`
- **行为:** 幂等追加。
- **错误:** 400(user_ids 存在非法 ID 或导致组员数超上限);403(oneid_dept 只读)

---

### `POST /admin/user-groups/members/remove`

- **权限:** 管理员(审计)
- **请求体:** `{"id": 20, "user_ids": [11]}`
- **行为:** 批量移除。

---

### `GET /admin/user-groups/groups-by-users`

- **权限:** 管理员
- **参数:** `user_ids=1,2,3`(上限 100)
- **响应:**

```json
{
  "ok": true,
  "data": {
    "1": [
      { "id": 3, "name": "前端组", "description": "", "full_path": "研发中心/前端组", "source": "manual" }
    ],
    "2": []
  }
}
```

---

### `GET /admin/user-groups/associated-models`

- **权限:** 管理员
- **参数:** `group_id=N`
- **响应:** `{"ok":true, "count":2, "models":[{"id":3,"provider":"tencentcodingplan","model_id":"minimax-m2.5"}]}`

---

#### 4. 未分组用户

> 原 `/admin/user-groups/ungrouped/detail` 和 `/admin/user-groups/ungrouped/members` 已废弃删除。
> 未分组用户统一通过 `/admin/user-groups/members?id=0` 查询，配置总览通过 `/admin/user-groups/config-overview?group_ids=0` 查询全局默认。

---

#### 5. 配置总览

### `GET /admin/user-groups/config-overview`

- **权限:** 管理员
- **参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_ids | string | 是 | 逗号分隔用户组 ID(如 `3,7,8`)。校验存在性 |
| keys | string | 否 | 逗号分隔 category key 过滤;不传=全部 13 个 category |

- **可用 keys:**

| key | 说明 |
|-----|------|
| chargeType | 计费模式 |
| resourcePolicy | 生效资源策略及其具体 ResourceConfig |
| model | 模型 |
| channel | 通道 |
| skill | 技能(含技能包 + 角色 + 技能安装来源) |
| agentTool | Agent 工具(企业插件 + 企业 MCP) |
| memory | 记忆 |
| drive | 网盘 |
| imageType | 镜像 |
| network | 网络(VPC/子网/安全组/公网) |
| cls | CLS 日志服务 |
| aiAgentSecurity | AI Agent 安全 |
| platformPolicy | 平台策略(用户配额 + 模型配额 + 功能权限开关) |

- **响应:**

```json
{
  "ok": true,
  "results": [
    {
      "group_id": 3,
      "categories": [
        {
          "key": "resourcePolicy", "label": "资源策略", "description": "...", "icon": "ServerCog",
          "entries": [
            {
              "id": "12",
              "label": "研发资源策略",
              "source": { "type": "local", "group_id": 3, "full_path": "根组/研发组" },
              "meta": {
                "policy_id": 12,
                "is_default": false,
                "value": {
                  "instance_charge_type": "POSTPAID_BY_HOUR",
                  "instance_type": "Ai2.LARGE8",
                  "system_disk": { "disk_type": "CLOUD_SSD", "disk_size": 100 }
                },
                "resource_config": {
                  "instance_charge_type": "POSTPAID_BY_HOUR",
                  "instance_type": "Ai2.LARGE8",
                  "system_disk": { "disk_type": "CLOUD_SSD", "disk_size": 100 }
                }
              }
            }
          ]
        },
        {
          "key": "model", "label": "模型", "description": "...", "icon": "Brain",
          "entries": [
            {
              "id": "2", "label": "Tencent HY 2.0 Instruct",
              "source": { "type": "local", "group_id": 3, "full_path": "根组/研发组" }
            }
          ]
        },
        {
          "key": "channel", "label": "通道", "description": "...", "icon": "MessageSquare",
          "entries": [
            { "id": "1", "label": "微信",   "source": { "type": "local", "group_id": 3, "full_path": "根组/研发组" } },
            { "id": "2", "label": "企业微信", "source": { "type": "all_users" } }
          ]
        }
      ]
    }
  ]
}
```

- **`source.type` 枚举:**

| 值 | 说明 |
|----|------|
| local | 本组直接配置(附带 group_id + full_path) |
| inherited | 继承自祖先组(附带 group_id + full_path) |
| all_users | 资源 visibility_type='all',全部用户可见 |
| site_default | 回退到 site_configs 全局默认值(仅策略类) |
| global | 全局配置项(网络/记忆/CLS 等,不按组配置) |
| unset | 未配置 |

- **`entry` 字段:**
  - `id` - 条目标识
  - `label` - 主标签
  - `sub_label` - 副标签/子分类(skill/agentTool/network/platformPolicy 用于前端分板块)
  - `source` - 配置来源
  - `meta` - 额外元数据。`resourcePolicy` 中 `value` 是向前端兼容的具体 ResourceConfig object，格式与旧版一致；`resource_config` 是内容相同的显式别名。策略配置仍只存于 `resource_policies.config_json`，分组绑定仅存策略 ID。其他 category 使用 `{"enabled":bool}` / `{"value":int}` / `{"type":"vpc","zone":"..."}` 等结构

---

#### 6. 删除影响报告

### `GET /admin/user-groups/delete-impact`

- **权限:** 管理员
- **参数:** `group_ids=3,4`(逗号分隔,校验存在性)

> 路径由 `/admin/user-groups/{id}/delete-impact` 改为 `/admin/user-groups/delete-impact?group_ids=`,支持批量。
> `blockers` 新增 `instances[]`(直属 Agent）。任意分组下存在 `instances.group_id = X` 的记录时,该分组不可删除,必须先迁移或销毁这些 Agent。

- **响应:**

```json
{
  "ok": true,
  "results": [
    {
      "group": { "id": 3, "name": "研发组", "full_path": "根组/研发组", "source": "manual" },
      "blockers": {
        "manual_descendants": [
          { "id": 7, "name": "研发一组", "full_path": "根组/研发组/研发一组", "source": "manual" }
        ],
        "resource_bindings": {
          "model": [{ "resource_id": 1, "resource_name": "Tencent HY" }],
          "skill": [], "skill_bundle": [], "role": [],
          "channel": [{ "resource_id": 1, "resource_name": "微信" }],
          "plugin_bundle": [{ "resource_id": 1, "resource_name": "通用插件包" }],
          "mcp": [],
          "image_type": [{ "resource_id": 0, "resource_name": "openclaw" }]
        },
        "instances": [
          { "instance_id": "ins-xxxx", "name": "my-agent-1" },
          { "instance_id": "",         "name": "my-agent-占位" }
        ],
        "policy_configs": [],
        "security_group_configs": [],
        "scoped_configs": []
      },
      "non_blocking_info": {
        "direct_members_count": 5,
        "total_member_count": 12,
        "note": "成员不阻塞删除;只属于此组的用户将变为游离用户。"
      },
      "hint": "该分组下存在直属创建的 Agent,请先迁移或销毁这些 Agent 再重试。"
    }
  ]
}
```

- **`blockers.instances[]` 字段:**

| 字段 | 类型 | 说明 |
|------|------|------|
| instance_id | string | 腾讯云 CVM 实例 ID(`ins-xxx`);占位记录阶段可能为空 |
| name | string | 实例展示名 |

- **`hint` 文案分支:**
  - 所有阻塞项均为空 → `"此组当前无阻塞项,可直接删除。"`
  - `instances` 非空 → `"该分组下存在直属创建的 Agent,请先迁移或销毁这些 Agent 再重试。"`
  - 其他阻塞项非空 → `"请先解除资源绑定 / 删除子分组,再重试。"`

> `policy_configs` 可返回 `[{config_key, value_json}]`(从 `group_config_bindings` 读 `policy` 类型);`security_group_configs` / `scoped_configs` 仍固定空数组(P2 实装)

---

#### 7. 加法型资源可见性

> 加法型资源使用 **Union** 语义:
> `用户可见资源 = 自身所有组 ∪ 所有祖先组 的绑定并集 ∪ visibility_type='all' 的资源`

### `POST /admin/models/visibility`

> **Release 已有**,此处为对齐描述。

- **权限:** 管理员(审计)
- **查询参数:** `?id=N`(模型主表 ID)
- **请求体:**

```json
{ "visibility_type": "group", "group_ids": [9] }
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| visibility_type | string | 是 | `all` / `group` |
| group_ids | []uint | 条件 | `visibility_type=group` 时必填 |

- **响应:** `{"ok": true}`
- **错误:** 400(visibility_type 无效 / group_ids 为空或有非法 ID)/ 404(模型不存在)

---

### `POST /admin/channels/visibility`

- **请求体:**

```json
{ "channel_id": "openclaw-weixin", "visibility_type": "group", "group_ids": [10] }
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channel_id | string 或 uint | 是 | 通道标识符(如 `"openclaw-weixin"`)或数据库 uint ID |
| visibility_type | string | 是 | `all` / `group` |
| group_ids | []uint | 条件 | |

- **错误:** 400 / 422(通道不存在)

---

### `POST /admin/mcp/visibility`

- **请求体:** `{"mcp_id": 1, "visibility_type": "group", "group_ids": [3]}`

---

### `POST /admin/images/type-visibility`

- **请求体:** `{"agent_type": "openclaw", "visibility_type": "group", "group_ids": [3, 7]}`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| agent_type | string | 是 | 镜像类型(`openclaw` / `hermes` / `lightclawace`) |

> 镜像类型可见性控制的是 `agent_type` 级别。设为 `group` 后,仅绑定组的用户才能创建该类型的实例。

---

#### 8. 平台策略

> 策略型配置使用 **最近祖先覆盖** 语义:从本组向上遍历祖先链,第一个有配置的组胜出;全链无配置回退 `site_configs` 全局默认。

### `POST /admin/group-config/policy`

- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_id | uint | 是 | 目标用户组 ID；不能为 0，且分组必须存在 |
| config_key | string | 是 | 平台策略 key，必须属于下方合法列表 |
| value_json | string | 是 | 策略值 JSON 字符串；格式由 `config_key` 决定 |

普通布尔策略请求示例：`{"group_id":3,"config_key":"user_config_channel","value_json":"{\"enabled\":false}"}`

- **合法 `config_key`:**

| config_key | value_json | 说明 |
|------------|-----------|------|
| token_quota_day | `{"value": 100000}` | 用户每日 Token 配额。若同组设置了 `token_quota_rules` 则以后者为准 |
| token_quota_rules | `{"value": "[{\"mode\":\"day\",\"limit\":100000}]"}` | 用户 Token 配额规则（JSON 数组字符串），优先于 `token_quota_day`。支持 mode: day/month/year/custom |
| instance_quota | `{"value": 3}` | 用户实例上限 |
| global_token_quota_day | `{"value": -1}` | 全局 Token 上限值(-1=无限)。`day` 为历史 key 名,实际按日或按月统计由 `site_configs.global_token_quota_period` 决定 |
| global_token_quota_rules | `{"value": "[{\"mode\":\"day\",\"limit\":500000}]"}` | 分组全局 Token 配额规则（JSON 数组字符串），优先于 `global_token_quota_day`。支持 mode: day/month/year/custom |
| user_config_model | `{"enabled": true}` | 允许用户配置模型 |
| user_config_channel | `{"enabled": true}` | 允许用户配置通道 |
| custom_model | `{"enabled": false}` | 允许用户添加自定义模型 |
| agent_terminal | `{"enabled": true}` | 允许用户进入终端 |
| gateway_ui | `{"enabled": false}` | 允许用户访问 Gateway 面板 |
| chat_view | `{"enabled": true}` | 允许用户使用对话视图 |
| browser_vnc | `{"enabled": false}` | 允许用户访问云端浏览器 |
| lobster_doctor | `{"enabled": false}` | 允许用户使用龙虾医生 |
| model_quota | `{"enabled": true}` | 允许用户查看模型额度 |
| smh_auto_provision | `{"enabled": true}` | 创建实例时自动开启网盘 |

> 写入 `token_quota_rules` / `global_token_quota_rules` 时，`value_json` 的外层固定为 `{"value":"<rules-json-string>"}`，其中 `<rules-json-string>` 是 rules 数组序列化后的字符串；rules 支持的写法见 `POST /admin/create` 的「Token 配额 rules 格式」。
>
> `global_token_quota_rules` / `global_token_quota_day` 是分组显式全局 Token 配额策略；未配置时 fallback 到站点全局规则，但用量仍按当前分组统计。
>
> 旧字段 `token_quota_day` / `global_token_quota_day` 仅用于兼容；写入 `*_day` 时会同步生成同组对应的 rules 配置并清除既有 `*_day` 绑定，写入 `*_rules` 时也会清除同组的 `*_day` 绑定。同时传 day 与 rules 时以 rules 为准。旧字段只能表达 day 或站点当前全局兼容周期，不能表达 `year` 或 `custom/yearly`。
>

- **错误:** 400(config_key 无效 / value_json 非法 JSON / group_id=0)/ 422(分组不存在)

---

### `POST /admin/group-config/policy/delete`

- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_id | uint | 是 | 目标用户组 ID；分组必须存在 |
| config_key | string | 是 | 要删除的平台策略 key |

- **请求体示例:** `{"group_id":3,"config_key":"user_config_channel"}`
- **行为:** 删除该组对应的策略绑定行 → 该组重新继承祖先或回退全局默认
- **配额 key 兼容:** 删除 `token_quota_rules` / `global_token_quota_rules` / `token_quota_day` / `global_token_quota_day` 时，若目标 key 绑定不存在，会自动删除配对 key 绑定（例如组仅有 `token_quota_day` 绑定时，删 `token_quota_rules` 会清掉 `token_quota_day`）
- **资源策略删除示例：** `{"group_id":3,"config_key":"resource_config"}`；删除后该组重新继承最近祖先资源策略，全链无命中时回退站点配置
- **错误:** 400 / 422(分组不存在 / 目标 key 与配对 key 均未配置)

---

#### 9. 配置查询

### `GET /admin/group-config/groups`

- **参数:** `queries=<JSON encoded>`

```json
[
  {"config_type": "channel", "config_key": "1"},
  {"config_type": "policy",  "config_key": "user_config_channel"}
]
```

查询用户 Token rules 与分组显式全局 Token rules：

```json
[
  {"config_type": "policy", "config_key": "token_quota_rules"},
  {"config_type": "policy", "config_key": "global_token_quota_rules"}
]
```

- **响应:**

```json
{
  "results": [
    {
      "config_type": "channel", "config_key": "1",
      "groups": [ { "group_id": 3, "group_name": "研发组" } ]
    },
    {
      "config_type": "policy", "config_key": "user_config_channel",
      "groups": [ { "group_id": 3, "group_name": "研发组", "value": {"enabled": false} } ]
    }
  ]
}
```

- **`config_type` 枚举:** `channel` / `plugin_bundle` / `mcp` / `image_type` / `policy`
- `value` 仅 policy 类型返回

---

#### 10. 多归属 & 未分组(users 视角)

---

### `POST /admin/instances/group-check`

- **权限:** 管理员
- **说明:** 批量查询指定实例的分组关系检查数据（`user_group_mismatch` 和 `has_config_drift`）。
  前端在渲染 Agent 列表后异步调用，获取红黄点指示器数据，与主列表接口（`GET /admin/instances`）解耦。
- **Content-Type：** `application/json`
- **请求体：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | uint[] | 是 | 实例主键列表；自动去重并过滤 0；上限 500 |
| check_user_group | bool | 否 | true 时计算 user_group_mismatch（用户已不在实例分组）；默认 false |
| check_config_drift | bool | 否 | true 时计算 has_config_drift（实例配置与目标组存在 different 行）；默认 false；有额外 CVM API 调用开销 |

- **响应：**

```json
{
  "ok": true,
  "results": [
    {"id": 1, "user_group_mismatch": false, "has_config_drift": true},
    {"id": 2, "user_group_mismatch": true,  "has_config_drift": false}
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| results | array | 每个有效实例一条，顺序与 DB 查询顺序一致（传入的 ID 不在 DB 中则不出现） |
| results[].id | uint | 实例主键 |
| results[].user_group_mismatch | bool | true = 用户 user_id 已不在实例的 group_id（或 group_id=0 但 user 已加入某分组） |
| results[].has_config_drift | bool | true = 实例配置相对 group_id 组配置存在 different 行（跳过 not_check；同 config-diff 语义） |

- **错误：**
  - 400 `请求体格式错误`（ids 为空或超限 500）
  - 405 非 POST 请求

- **性能说明：**
  - `check_user_group=false, check_config_drift=false` → 仅一次按 IDs 查 DB，极轻量
  - `check_config_drift=true` → 额外一次批量 CVM API（拉公网三字段）+ 各 unique group 一次配置查询，适合页面渲染后延迟调用

---

### `POST /admin/instances/by-user-group`

- **权限:** 管理员
- **说明:** 按 (user_id, group_id) 精确对或 group_ids 子树批量查询实例清单。**不分页,一次返回全量**。用于管控端"按分组 → 所属用户 → 其所有 Agent"的快速筛选。
- **请求体:**

```json
{
  "user_group_ids": [{"user_id": 1, "group_id": 3}, {"user_id": 2, "group_id": 3}],
  "group_ids":      [7]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_group_ids | [{user_id, group_id}] | 条件 | 直接 (user_id, group_id) 精确对数组(用户和分组 id 对);自动过滤含 0 值的项 |
| group_ids | []uint | 条件 | 分组根集合;每个根展开**子孙(含自身,通过 `group_closure`)** × 子孙下的直属成员,拼成更多 (user_id, group_id) 对 |

> `user_group_ids` 和 `group_ids` **至少传一个**(否则返回 400);两者可同时传,最终去重合并。

> ⚠️ **命名变更:** 该字段曾命名为 `user_groups`,现统一改为 `user_group_ids`,语义更明确——表示"用户和分组 id 对"。**旧字段名 `user_groups` 服务端不再识别**,客户端必须升级。

- **上限:** 合并去重后的 (user_id, group_id) 对数量 ≤ **2000**(`adminInstancesByUserGroupMaxPairs`),超限返回 400。
- **响应:**

```json
{
  "ok": true,
  "instances": [
    {
      "id": 1,
      "instance_id": "ins-xxx",
      "name": "my-agent",
      "user_id": 19,
      "user_name": "张三",
      "group_id": 3,
      "group_full_path": "根组/研发组",
      "status": "running",
      "created_at": "2026-05-08T10:00:00Z"
    }
  ]
}
```

- **响应字段(对应 `controller.instanceByUserGroupItem`):**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 实例主键 |
| instance_id | string | 腾讯云 CVM 实例 ID |
| name | string | 实例展示名 |
| user_id | uint | 所属用户 ID |
| user_name | string | 所属用户的用户名;被软删或找不到时为空串(查询用 `Unscoped` 兼容历史 Agent) |
| group_id | uint | 所属分组 ID |
| group_full_path | string | 所属分组完整路径;`group_id=0` 时为空串(此处一般不会出现 0,因为查询条件已排除) |
| status | string | 实时状态(来自 `ResolveInstanceStatus`,会调用腾讯云 CVM `DescribeInstances` 取实时态) |
| created_at | string | UTC RFC3339 时间(如 `2026-05-08T10:00:00Z`) |

- **去重保证:**
  - `user_group_ids` 中重复传 `(user_id, group_id)` → 服务端用 `pairSet := map[userGroupPair]struct{}` 自动去重
  - `group_ids` 同时传 parent + child(子树展开重叠) → DB 层 `Distinct("descendant_id")` + 应用层 `pairSet` 双重去重
  - `user_group_ids` 与 `group_ids` 子树展开后的对若有交叉 → `pairSet` 合并去重
  - 实例级:SQL 一次查 + 内存按 pair 精确过滤,每个实例 ID 至多入选一次,**响应中 `instances[]` 不会出现重复行**

- **软删 instance 过滤:**
  - 实例查询使用 `Find(&[]model.Instance{})`,GORM 自动注入 `deleted_at IS NULL`
  - **已销毁的 instance 不会出现在响应中**,与 `/admin/users.groups[].instance_count` / `/admin/user-groups.instance_count` / `/admin/user-groups/members.direct_groups[].instance_count` 行为一致

- **SQL 展开策略:**
  1. `user_group_ids` → 直接得到一组精确 `(user_id, group_id)` 对(去重、过滤含 0 值的项)
  2. `group_ids` → 通过 `group_closure` 查子树(含自身)→ 对每个子孙组查 `user_group_members` 直属成员 → 拼 `(member.user_id, group.id)` 对
  3. 合并去重 → 按 `instance.user_id IN (...) AND instance.group_id IN (...)` 筛 `instances`(GORM 自动过滤软删)
  4. 批量回填 `group_full_path` 和 `user_name`(各一次 SQL),CVM 实时状态批量调腾讯云 API
  5. 内存按 pair 精确过滤 + 组装响应

- **错误:**
  - 400 `user_group_ids 和 group_ids 至少传一个`
  - 400 `请求体格式错误`
  - 400 `展开后的 (user_id, group_id) 对数量超过上限 2000`
  - 405 非 POST 请求

- **典型用法:**
  - 管控端"按分组查 Agent"列表页,前端先调 `/admin/user-groups/tree` 获取分组,选中后再调本接口拉对应实例
  - 与 `/admin/instances` 的区别:本接口走"用户 × 分组"精确匹配,无分页;`/admin/instances` 是全量分页 + 状态/agent_type 过滤维度

---

### `GET /admin/users/multi-group-stats`

- **响应:**

```json
{
  "ok": true,
  "total_users": 48,
  "multi_group_users": 3,
  "ungrouped_users": 3,
  "top_examples": [
    { "user_id": 19, "username": "张三", "group_count": 3, "groups": ["..."] }
  ]
}
```

用于驱动前端"多归属 Banner"。

---

> `GET /admin/users/ungrouped` 已废弃删除，未分组用户通过 `/admin/user-groups/members?id=0` 查询。

---

#### 11. OneID 同步

### `POST /admin/oneid-sync-users`

> **Release 已有**;本次新增 body 字段 + 响应字段。
> 1. `affected_dept_groups` 字段类型由 `[]string`(full_path) 升级为 `[]{group_id, full_path}`,前端拿 group_id 可直接发起后续操作。
> 2. 新增 `change_parent_group_ids`、`move_group_user_ids` 两个事件数组,分别对应本次同步发生**父节点切换的组**和**用户被从旧组迁出**的事件。

- **权限:** 管理员(审计)
- **请求体(可选):** `{"sync_dept": true}`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| sync_dept | bool | 否 | `true`=强制本次进入部门落地(Step 1.8 + 2.6);省略/false=仅当本地已有 oneid_dept 组才继续维护 |

- **响应:**

```json
{
  "ok": true,
  "message": "同步完成",
  "profile_count": 15,
  "dept_count": 12,
  "user_count": 9,
  "affected_users": [
    { "username": "张三", "action": "disabled", "instance_count": 0, "vpc_has_resources": false }
  ],
  "dept_group_count": 12,
  "affected_dept_groups": [
    { "group_id": 71, "full_path": "OpenClaw企业/运营部" }
  ],
  "change_parent_group_ids": [65, 70],
  "move_group_user_ids": [
    { "user_id": 19, "from_group_id": 70 }
  ],
  "landing_failures": [
    {
      "department_id": "1442680221525345068",
      "department_name": "二组",
      "stage": "create",
      "err": "constraint failed: UNIQUE constraint failed: user_groups.identifier, user_groups.name"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| profile_count | int64 | `oneid_user_profiles` 行数 |
| dept_count | int64 | `oneid_departments` 行数(OneID 原始部门拉取结果) |
| user_count | int64 | 带 `one_id_sub` 的本地用户数 |
| affected_users | [] | 被禁用/删除的用户列表(`action` 为 `disabled` / `hard_deleted`) |
| dept_group_count | int64 | 落地后 `user_groups` 中 `source=oneid_dept AND to_be_deleted=0` 的数量;`sync_dept=false` 且本地无 oneid_dept 组时为 0 |
| affected_dept_groups | `[{group_id, full_path}]?` | 本次新打 `to_be_deleted=1` 的分组明细(含 group_id + full_path)。前端可直接拿 group_id 发起 `/admin/user-groups/delete`。空时省略(JSON null) |
| change_parent_group_ids | []uint? | 本次同步中检测到 `existing.ParentID != 新解析 parentGroupID` 且 `UpdateUserGroupExtForOneIDDept` 成功落盘的 oneid_dept 组 ID 列表。仅改名不换父不计入 |
| move_group_user_ids | `[{user_id, from_group_id}]?` | 本次同步把 `user_group_members` 物理删除的事件列表(事务提交成功后才写入),标记"用户在 OneID 侧被调出了 from_group_id 对应部门"。前端可据此做"你的成员已被移出 xxx 组"的提醒 |
| landing_failures | [] | 本次未落地的部门明细 |

> **空值行为:** `affected_dept_groups` / `change_parent_group_ids` / `move_group_user_ids` 均为 `omitempty` + `nilIfEmpty*` 序列化,空时等价于不出现(JSON null 或字段省略),前端应同时兼容"缺字段"和"字段为 null/空数组"两种情况。

> `landing_failures[].stage` 可能值:
>   `lookup` / `create` / `update` / `rename_parent` / `clear_tbd`
>
> **常见 landing 失败根因:**
>   - 老库残留唯一索引 `idx_ug_identifier_name`(全库 name 唯一),冲突时会报 `UNIQUE constraint failed: user_groups.identifier, user_groups.name` → 需执行 `sql/0504-drop-stale-user-groups-index.sql`

---

### `GET /admin/oneid-sync-users/status`

- **权限:** 管理员
- **响应:**

```json
{
  "running": false,
  "last_sync": "2026-05-04T13:18:55Z",
  "profile_count": 15,
  "dept_count": 12,
  "oneid_user_count": 9,
  "oneid_dept_group_count": 12
}
```

| 字段 | 说明 |
|------|------|
| running | 是否正在同步 |
| last_sync | 最近一次完成时间(RFC3339) |
| profile_count | `oneid_user_profiles` 行数 |
| dept_count | `oneid_departments` 行数 |
| oneid_user_count | 带 `one_id_sub` 的本地用户数 |
| oneid_dept_group_count | - 本地 `user_groups` 中 `source=oneid_dept AND to_be_deleted=0` 的数量 |

> 本次两端点均加上 `WithOpenAPI` 中间件,支持 API Token(`Authorization: Bearer hk-...`)认证。

---

#### 12. 用户端 API

### `GET /user-groups/mine`

> `groups[]` 每项新增 `full_path` / `source` / `is_main` / `created_at` 四个字段。前端据此区分 manual / oneid_dept、显示完整路径并标注组织主部门。

- **权限:** 登录用户
- **行为:** 返回当前用户所在的所有分组(含完整成员列表);`is_main` 批量在一条 SQL 内取出当前用户在所有 groupIDs 中命中 `is_main=true` 的组,避免 N+1。
- **响应:**

```json
{
  "ok": true,
  "groups": [
    {
      "id": 3,
      "name": "研发组",
      "full_path": "根组/研发组",
      "source": "manual",
      "is_main": false,
      "created_at": "2026-05-01T10:00:00Z",
      "description": "",
      "member_count": 8,
      "members": [
        { "user_id": 19, "username": "张三" }
      ]
    },
    {
      "id": 70,
      "name": "基础架构组",
      "full_path": "OpenClaw企业/技术部/后端中心/基础架构组",
      "source": "oneid_dept",
      "is_main": true,
      "created_at": "2026-05-04T12:54:43Z",
      "description": "",
      "member_count": 4,
      "members": [
        { "user_id": 19, "username": "张三" }
      ]
    }
  ]
}
```

- **`groups[]` 字段:**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 分组 ID |
| name | string | 分组名称 |
| full_path | string | 分组全路径(如"研发中心/后端组") |
| source | string | 分组来源 `manual` / `oneid_dept` |
| is_main | bool | 是否是当前用户的 oneid_dept **主部门**(manual 分组恒 `false`) |
| created_at | string | 分组创建时间(UTC RFC3339,如 `2026-05-08T12:00:00Z`) |
| description | string | 分组描述 |
| member_count | int64 | 该分组的直属成员数 |
| members | `[{user_id, username}]` | 该分组完整直属成员列表(批量查询,避免 N+1) |

---

### `GET /openclaw/config-overview`

> 新增 `group_ids` 查询模式,用于**创建实例前尚无 `agent_id` 的场景**(如创建弹窗中先选分组、再根据分组预览可用模型/通道/镜像等配置)。与原 `ids` 模式二选一,**`group_ids` 优先**。

- **权限:** 登录用户(Session Cookie)
- **说明:** 查询分组配置总览,支持两种模式:
  - **模式一(新增) `group_ids`**:直接按分组 ID 查询。无需有实例,适用于创建实例前的分组预览场景。
  - **模式二 `ids`**:按实例 ID 查询其绑定分组的配置(实例必须属于当前登录用户)。相同 group_id 的 agent 共享结果(内部去重查询)。group_id=0 的 agent 返回全局默认配置。

- **参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_ids | string | 二选一 | 逗号分隔的分组 ID(如 `3,4`)。**优先级高于 `ids`**;`0` 代表全局默认配置;服务端自动对 group_id 去重 |
| ids | string | 二选一 | 逗号分隔的实例 ID(如 `1,2,3`),必须属于当前登录用户;仅当未传 `group_ids` 时生效 |
| keys | string | 否 | 逗号分隔的 category key 过滤;不传返回全部 category |

> `group_ids` 与 `ids` 必须至少传一个,否则返回 400 `ids 或 group_ids 参数必须传其一`。

- **可用的 keys:** 同 §5 `GET /admin/user-groups/config-overview`(model / channel / skill / agentTool / memory / drive / imageType / network / cls / aiAgentSecurity / platformPolicy)

- **错误:**
  - 400 `ids 或 group_ids 参数必须传其一` - 两个参数都未传
  - 400 `group_ids 格式错误` - `group_ids` 不是合法的 uint CSV
  - 400 `ids 格式错误` - `ids` 不是合法的 uint CSV
  - 400 `未找到属于当前用户的实例` - `ids` 模式下查不到任何归属当前用户的实例
  - 400 `无效的 key: xxx` - `keys` 含非法值
  - 401: 未登录
  - 500 `解析分组祖先链失败` / `查询实例失败` - 内部错误

- **响应(模式一 `group_ids`):** 每项以 `group_id` 为主键,不含实例维度。

```json
{
  "ok": true,
  "results": [
    {
      "group_id": 3,
      "categories": [
        {
          "key": "model",
          "label": "模型",
          "description": "用户能使用哪些模型",
          "icon": "Brain",
          "entries": [
            { "id": "2", "label": "Tencent HY 2.0 Instruct", "source": { "type": "local", "group_id": 3, "full_path": "根组/研发组" } }
          ]
        }
      ]
    },
    {
      "group_id": 0,
      "categories": [...]
    }
  ]
}
```

- **响应(模式二 `ids`):** 每项额外含 `id`(实例 ID),其余结构一致。

```json
{
  "ok": true,
  "results": [
    {
      "id": 1,
      "group_id": 3,
      "categories": [
        {
          "key": "model",
          "label": "模型",
          "description": "用户能使用哪些模型",
          "icon": "Brain",
          "entries": [
            { "id": "2", "label": "Tencent HY 2.0 Instruct", "source": { "type": "local", "group_id": 3, "full_path": "根组/研发组" } }
          ]
        }
      ]
    },
    {
      "id": 2,
      "group_id": 3,
      "categories": [...]
    }
  ]
}
```

- **字段差异说明:**
  - 模式一返回项**不包含 `id`**(没有实例上下文);模式二每项必有 `id`。
  - 两种模式的 `categories` 结构完全一致(复用同一个 resolver)。
  - 模式一的 `group_ids` 会在响应中保持**首次出现顺序**并去重;模式二按实例行一一对应(同 group 的实例命中同一份缓存,但仍会重复返回)。

---

### `GET /quota/data` - group_id 增强

- **权限:** 登录用户
- **响应新增字段:** `quota_period`、`token_quota_rules`、`token_quota_usages`、`global_token_quota_rules`、`global_token_quota_usages`。
- **新增可选参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_id | uint | 否 | 按 agent 绑定的分组过滤用量数据，并按该分组上下文解析用户 Token 配额规则 |

- **行为变化:**
  - 传 `group_id` 时:查询条件加 `group_id = ?` 过滤 `DailyUsageSummary`;`token_quota_rules` 与 LLM proxy 一致，优先使用该组/祖先组的分组用户 Token 规则，未命中时回退到站点默认用户规则
  - 不传时:`token_quota_rules` 返回当前用户自身有效规则；`quota_day` 为兼容旧字段，由 `token_quota_rules` 中的 day 规则反推，没有 day 规则时为 `-1`
  - `token_quota_usages` 返回当前用户各规则当前生效窗口内用量；传 `group_id` 时只统计该分组下的 `LLMUsageLog`，不传时统计该用户全部分组用量
  - `global_token_quota_rules` / `global_token_quota_usages` 返回站点全局 Token 规则及当前周期内全站用量；分组显式全局规则仍由 LLM proxy 执行阶段校验，不在 `/quota/data` 顶层替换站点全局规则

---

### `GET /quota/logs` - group_id 增强

- **权限:** 登录用户
- **新增可选参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_id | uint | 否 | 按 agent 绑定的分组过滤 LLMUsageLog 明细记录 |

- **行为变化:**
  - 传 `group_id` 时:查询条件加 `group_id = ?` 过滤
  - 不传时:保持原逻辑

---

### `GET /admin/usage/data` - group_by=group 增强

- **权限:** 管理员
- **说明:** 新增 `group_by=group` 维度,按 agent 绑定的分组聚合 Token 用量。支持 `group_id` 参数筛选指定分组及其所有后代组。
- **参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 起始日期(默认今天) |
| end_date | string | 否 | 结束日期(默认今天) |
| group_by | string | 否 | 聚合维度,逗号分隔:`date` / `user` / `model` / `instance` / `department` / `group`。不传默认 `date,model` |
| group_id | uint | 否 | 筛选指定分组及其所有后代组(通过 closure 表展开)。例如传研发组 ID,返回研发组+研发一组+研发二组的数据 |
| user_id | uint | 否 | 按用户过滤 |
| ai_model_id | uint | 否 | 按模型过滤 |
| instance_id | uint | 否 | 按实例过滤 |
| order_by | string | 否 | 排序字段:`total_tokens` / `request_count`。不传则不排序 |
| order | string | 否 | `desc` 为降序。需配合 `order_by` 使用 |

- **`group_by=group` 时的响应:**

```json
{
  "start_date": "2026-05-01",
  "end_date": "2026-05-08",
  "group_by": ["group"],
  "rows": [
    {
      "group_id": 3,
      "group_name": "研发组",
      "group_full_path": "根组/研发组",
      "prompt_tokens": 120000,
      "completion_tokens": 80000,
      "total_tokens": 200000,
      "prompt_cache_read_tokens": 80000,
      "prompt_cache_write_tokens": 10000,
      "request_count": 150,
      "global_token_quota_rules": [
        {"mode": "day", "limit": 300000}
      ],
      "global_token_quota_usages": [
        {"rule_index": 0, "used": 200000, "period_start": 1773244800, "period_end": 1773331200, "active": true}
      ]
    },
    {
      "group_id": 7,
      "group_name": "研发一组",
      "group_full_path": "根组/研发组/研发一组",
      "prompt_tokens": 50000,
      "completion_tokens": 30000,
      "total_tokens": 80000,
      "prompt_cache_read_tokens": 20000,
      "prompt_cache_write_tokens": 5000,
      "request_count": 60
    }
  ],
  "global_token_quota_day": 500000,
  "global_token_quota_rules": [
    {"mode": "day", "limit": 500000}
  ],
  "global_token_quota_usages": [
    {"rule_index": 0, "used": 280000, "period_start": 1773244800, "period_end": 1773331200, "active": true}
  ]
}
```

- **行为说明:**
  - 仅聚合 `group_id > 0` 的记录(有分组的 agent)
  - `group_id` 筛选会通过 `group_closure` 表展开所有后代组,每个后代组独立一行返回
  - `group_by=group` 与其他维度互斥(类似 `group_by=department`)
  - 各分组维度(按用户/模型/实例/部门/分组)的排序行为一致:前端传 `order=desc&order_by=total_tokens` 时按指定字段降序,不传则返回自然顺序
  - 顶层 `global_token_quota_rules` / `global_token_quota_usages` 表示站点全局规则及全站当前周期用量
  - `group_by=user` 时，每个用户 row 额外返回 `token_quota_rules` / `token_quota_usages`；未传 `group_id` 时还会返回 `token_quota_groups`，列出该用户所属各分组下的规则和当前周期用量；传 `group_id` 时按该组上下文解析用户 Token 规则
  - `group_by=group` 时，每个分组 row 返回有效 `global_token_quota_rules` / `global_token_quota_usages`；分组有显式全局 Token 规则时返回分组规则，否则 fallback 到站点全局规则，用量始终按该分组统计
  - `*_quota_usages[].rule_index` 对应同名 `*_quota_rules` 数组下标，`used` 为该规则当前生效窗口内的 `total_tokens` 汇总，`period_start` / `period_end` 为该窗口的 Unix 秒时间范围；`active=false` 表示当前无生效窗口，兼容返回 `period_start=0`、`period_end=0`；无终止的生效窗口返回 `period_end=null`

---

#### 13. 基础配置检查(admin/notices)

### `GET /admin/notices` - `config_steps` 部分

- **权限:** 管理员
- **响应中 `config_steps` 字段(标准模式 8 步,OneID 模式 9 步):**

```json
{
  "config_steps": [
    { "key": "brand",          "label": "设置平台名称与品牌",  "done": true },
    { "key": "default_quota",  "label": "配置用户默认配额",    "done": true },
    { "key": "users",          "label": "导入企业用户",        "done": true },
    { "key": "model",          "label": "配置至少一个模型",    "done": true },
    { "key": "channel",        "label": "配置至少一个通道",    "done": true },
    { "key": "vpc",            "label": "配置私有网络",        "done": true },
    { "key": "security_group", "label": "配置安全组",          "done": true },
    { "key": "image",          "label": "配置至少一个启用镜像", "done": true }
  ]
}
```

- **各步骤判断条件:**

| key | 判断条件 |
|-----|---------|
| brand | `site_config.Name != ""` |
| default_quota | `DefaultInstanceQuota >= 1` 或 `DefaultTokenQuotaDay > 0` |
| users | 用户总数 > 1 |
| sso_login | `sso_im_types` 非空(仅 OneID 模式显示) |
| model | 存在 `enabled=true && visibility_type='all'` 的模型 |
| channel | 存在 `enabled=true && visibility_type='all'` 的通道 |
| vpc | `site_config.VpcId != ""` |
| security_group | `site_config.SecurityGroupId != ""` |
| image | 存在 `enabled=true` 的镜像 |

> `model` / `channel` 判断必须 `visibility_type='all'`,确保未分组用户也能使用。仅绑定到特定分组的资源不计入基础配置检查。

---

#### 14. VPC 分组网络配置

> 支持管理员为不同用户组配置独立的 VPC 和子网。每条配置记录代表一个完整的绑定关系（VPC/子网策略 + 关联的分组），对应前端列表中的一行数据。
>
> **数据模型**：新增独立 `vpc_configs` 表存储 VPC 资源信息，通过 `group_config_bindings`（`config_type="vpc"`）加法型绑定关联分组。
>
> **预设策略**（原全局配置）仍通过 `POST /admin/config/cvm` 编辑，本章接口仅管理分组级别的 VPC 配置。

---

### `GET /admin/group-vpc-configs`

- **权限:** 管理员
- **说明:** 返回所有 VPC 分组配置列表。每条 `vpc_configs` 记录为一行，含策略名称、VPC/子网信息和已绑定的分组列表。

- **响应字段:**

| 字段 | 类型 | 说明 |
|------|------|------|
| data[].id | uint | vpc_configs 表主键 |
| data[].vpc_id | string | 腾讯云 VPC ID |
| data[].subnet_ids | object | 子网配置，格式 `{"zone": ["subnet-id", ...]}` |
| data[].strategy_name | string | 策略名称（可能为空） |
| data[].visibility_type | string | 应用范围：`all` 或 `group` |
| data[].visibility_groups | []object | 绑定的分组列表（visibility_type=group 时有值） |
| data[].visibility_groups[].group_id | uint | 分组 ID |
| data[].visibility_groups[].group_name | string | 分组名称（叶子节点名） |
| data[].visibility_groups[].group_full_path | string | 分组全路径名（如 "A公司 / 产品部 / 设计组"） |
| data[].created_at | string | 创建时间（ISO 8601） |
| data[].updated_at | string | 更新时间（ISO 8601） |

- **响应示例:**

```json
{
  "data": [
    {
      "id": 1,
      "vpc_id": "vpc-9lyx5t8h",
      "subnet_ids": {"ap-guangzhou-6": ["subnet-aaa"], "ap-guangzhou-7": ["subnet-bbb"]},
      "strategy_name": "研发组网络",
      "visibility_type": "group",
      "visibility_groups": [
        {"group_id": 5, "group_name": "人力资源", "group_full_path": "A公司 / 人力资源"},
        {"group_id": 8, "group_name": "财务部", "group_full_path": "A公司 / 财务部"}
      ],
      "created_at": "2026-05-08T10:00:00Z",
      "updated_at": "2026-05-08T10:00:00Z"
    },
    {
      "id": 2,
      "vpc_id": "vpc-9lyx5t8h",
      "subnet_ids": {"ap-guangzhou-6": ["subnet-aaa"]},
      "strategy_name": "",
      "visibility_type": "group",
      "visibility_groups": [
        {"group_id": 12, "group_name": "产品部", "group_full_path": "A公司 / 产品部"}
      ],
      "created_at": "2026-05-09T10:00:00Z",
      "updated_at": "2026-05-09T10:00:00Z"
    }
  ]
}
```

> 注：id=1 和 id=2 的 vpc_id 相同，但它们是独立的绑定关系，分别关联不同的分组，列表中分两行展示。

---

### `POST /admin/group-vpc-configs/create`

- **权限:** 管理员(审计)
- **说明:** 新增一条绑定关系（VPC + 子网 + 分组）。数据存储在独立的 `vpc_configs` 表中，绑定关系通过 `group_config_bindings`（`config_type="vpc"`）管理。

- **请求字段:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| vpc_id | string | 是 | 腾讯云 VPC ID |
| subnet_ids | string | 是 | JSON 字符串，格式 `{"zone": ["subnet-id", ...]}` |
| strategy_name | string | 否 | 策略名称，最长 20 字符，用户填写时才保存 |
| group_ids | []uint | 否 | 绑定的分组 ID 列表（为空则 visibility_type=all） |

- **请求示例:**

```json
{
  "vpc_id": "vpc-9lyx5t8h",
  "subnet_ids": "{\"ap-guangzhou-6\": [\"subnet-aaa\"]}",
  "strategy_name": "研发组专用",
  "group_ids": [5, 8]
}
```

- **校验规则:**
  - `vpc_id` 必须在腾讯云存在（调用 DescribeVpcs 校验）
  - `subnet_ids` 中的子网必须属于对应 VPC 和可用区（调用 DescribeSubnets 校验）
  - `group_ids` 中每个分组不能已绑定其他 vpc_configs 记录（单向约束）
  - `strategy_name` 长度不超过 20 字符

- **处理逻辑:**
  1. 创建 `vpc_configs` 记录（visibility_type 根据 group_ids 是否为空设为 group/all）
  2. 若 group_ids 非空，创建对应的 `group_config_bindings` 绑定记录

- **响应字段:**

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 是否成功 |

- **响应示例:** `{"ok": true}`
- **错误:** 400(vpc_id 无效 / subnet_ids 格式错误 / 分组冲突 / strategy_name 超长) / 422(分组不存在)

---

### `POST /admin/group-vpc-configs/update`

- **权限:** 管理员(审计)
- **说明:** 更新一条绑定关系（VPC + 子网 + 策略名 + 分组，全量更新）。

- **请求字段:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | vpc_configs 表主键 |
| vpc_id | string | 是 | VPC ID |
| subnet_ids | string | 是 | JSON 子网配置 |
| strategy_name | string | 否 | 策略名称，最长 20 字符 |
| group_ids | []uint | 否 | 绑定的分组 ID 列表（全量替换，为空则 visibility_type=all） |

- **请求示例:**

```json
{
  "id": 1,
  "vpc_id": "vpc-9lyx5t8h",
  "subnet_ids": "{\"ap-guangzhou-6\": [\"subnet-aaa\", \"subnet-bbb\"]}",
  "strategy_name": "研发组专用-更新",
  "group_ids": [5, 8, 10]
}
```

- **校验规则:**
  - 同 create（VPC/子网云端校验 + strategy_name 长度 + 分组单向约束）
  - 分组单向约束校验时排除当前记录自身已有的绑定（允许保留原分组）

- **处理逻辑（事务内）:**
  1. 更新 `vpc_configs` 记录（vpc_id、subnet_ids、strategy_name、visibility_type）
  2. 全量替换 `group_config_bindings` 绑定（先删旧的，再插新的）

- **响应字段:**

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 是否成功 |

- **响应示例:** `{"ok": true}`
- **错误:** 400(校验失败) / 404(id 不存在)

---

### `POST /admin/group-vpc-configs/delete`

- **权限:** 管理员(审计)
- **说明:** 删除一条绑定关系（同时清理 `group_config_bindings` 中对应的所有绑定记录）。删除后原绑定的所有分组下新建实例将回退到上级或预设策略。

- **请求字段:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | vpc_configs 表主键 |

- **请求示例:**

```json
{
  "id": 1
}
```

- **响应字段:**

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 是否成功 |

- **响应示例:** `{"ok": true}`
- **错误:** 400(id 无效) / 404(记录不存在)

---

## 九、其他

### `GET /site`

获取站点基本信息（名称、是否有 Logo、终端开关状态、SSO 配置等）。**无需登录**，前端在加载登录页前调用，用于渲染站点名称、Logo、SSO 按钮等。已登录时额外返回 `skillhub`、`gateway_ui_enable`、`gateway_ui_addr_type` 等字段。

- **权限：** 公开
- **响应：** 始终返回 JSON

未登录：

```json
{
  "name": "ClawPro",
  "has_logo": true,
  "chat_view_enabled": true,
  "has_oneid": true,
  "oneid_account_id": "enterprise-xxx",
  "sso_im_types": ["wecom", "feishu"],
  "sso_im_type_options": [
    {"value": "wecom",         "label": "企业微信",       "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/wework-v2-logo.png"},
    {"value": "feishu",        "label": "飞书",           "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/lark-v2-logo.png"},
    {"value": "dingtalk",      "label": "钉钉",           "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/dd-v2-logo.png"},
    {"value": "aad",           "label": "微软 Entra ID",  "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/aad-v2-logo.png"},
    {"value": "saml",          "label": "SAML 2.0",      "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/saml-v2-logo.png"},
    {"value": "ad",            "label": "Windows AD",    "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/ad-v2-logo.png"},
    {"value": "wework_private","label": "私有化企微",      "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/wework-logo.png"},
    {"value": "oidc",          "label": "OIDC",          "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/oidc-v2-logo.png"},
    {"value": "jwt",           "label": "JWT",           "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/jwt-v2-logo.png"},
    {"value": "openldap",      "label": "OpenLDAP",      "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/openldap-v2-logo.png"},
    {"value": "cas",           "label": "CAS",           "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/cas-v2-logo.png"},
    {"value": "oauth2",        "label": "OAuth 2.0",     "logo": ""}
  ],
  "is_overseas": true,
}
```

已登录：

```json
{
  "ok": true,
  "name": "ClawPro",
  "has_logo": true,
  "terminal_enabled": true,
  "chat_view_enabled": true,
  "gateway_ui_enable": false,
  "gateway_ui_addr_type": "private",
  "browser_vnc_enable": false,
  "cvm_region_id": "ap-guangzhou",
  "has_oneid": true,
  "sso_im_types": ["wecom"],
  "sso_im_type_options": [
    {"value": "wecom",         "label": "企业微信",       "logo": "https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/wework-v2-logo.png"},
    {"value": "feishu",        "label": "飞书",           "logo": "https://..."},
    "..."
  ],
  "is_overseas": true,
  "skillhub": "https://skillhub.example.com"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 站点名称 |
| has_logo | bool | 是否已上传 Logo，`true`/`false` 均可调用 `GET /logo`（未上传时返回默认 SVG） |
| chat_view_enabled | bool | 是否允许前端加载对话界面（**登录和未登录时均返回**） |
| has_oneid | bool | 是否开启 SSO（`TenantID` 非空），`true` 时在登录页显示 SSO 按钮 |
| oneid_account_id | string | OneID 企业 account_id，**仅 `has_oneid=true` 时返回**，用于前端登录组件的 `select_account` 参数 |
| is_unified_account | bool | 是否为统一账号模式（`OneIDAppID` 非空） |
| is_universe | bool | 是否为多租户 universe 模式（`true` = 多租户，`false` = 单租户） |
| sso_im_types | string[] | 管理员已配置的 IM 类型列表，决定登录页显示几个 SSO 按钮及其文案，空数组时显示通用「OneID 登录」按钮 |
| sso_im_type_options | object[] | 全量 IM 类型枚举，每项含 `value`、`label`、`logo` 三个字段，管理员配置页多选框使用 |
| is_overseas | bool | 是否为海外站点 |
| terminal_enabled | bool | 是否开启终端功能，**仅登录后返回** |
| gateway_ui_enable | bool | 是否开启 Gateway UI 面板，**仅登录后返回** |
| gateway_ui_addr_type | string | Gateway UI 地址类型（`private`/`public`），**仅登录后返回** |
| browser_vnc_enable | bool | 是否开启云端浏览器（VNC）功能，**仅登录后返回** |
| user_config_model_enabled | bool | 是否允许用户查看与配置模型，默认 `true`，**仅登录后返回** |
| user_config_channel_enabled | bool | 是否允许用户查看与配置通道，默认 `true`，**仅登录后返回** |
| model_quota_enabled | bool | 是否允许用户查看模型额度，默认 `true`，**仅登录后返回** |
| cvm_region_id | string | CVM 地域 ID（如 `ap-guangzhou`），**仅登录后返回** |
| skillhub | string | 技能市场地址，**仅登录后返回**，未登录时字段不存在 |

`sso_im_type_options` 每项结构：

| 字段 | 类型 | 说明 |
|------|------|------|
| value | string | 类型标识，存入数据库的值 |
| label | string | 显示名称 |
| logo | string | 图标 URL，可直接用于 `<img src>`，空字符串表示暂无图标 |

全量枚举（固定值，由后端维护）：

| value | label | 备注 |
|-------|-------|------|
| wecom | 企业微信 | |
| feishu | 飞书 | |
| dingtalk | 钉钉 | |
| aad | 微软 Entra ID | 原 Azure AD |
| saml | SAML 2.0 | |
| ad | Windows AD | |
| wework_private | 私有化企微 | |
| oidc | OIDC | |
| jwt | JWT | |
| openldap | OpenLDAP | |
| cas | CAS | |
| oauth2 | OAuth 2.0 | 暂无图标 |

`sso_im_types` 与登录按钮对应关系：

| 值 | 建议按钮文案 |
|----|-------------|
| wecom | 企业微信登录 |
| feishu | 飞书登录 |
| dingtalk | 钉钉登录 |
| aad | 微软 Entra ID 登录 |
| saml | SAML 2.0 登录 |
| ad | Windows AD 登录 |
| wework_private | 私有化企微登录 |
| oidc | OIDC 登录 |
| jwt | JWT 登录 |
| openldap | OpenLDAP 登录 |
| cas | CAS 登录 |
| oauth2 | OAuth 2.0 登录 |
| 空数组 | OneID 登录 |

**前端典型用法：**

```javascript
const site = await fetch('/site').then(r => r.json())

// 渲染 SSO 按钮
const labelMap = { wecom: '企业微信登录', feishu: '飞书登录', dingtalk: '钉钉登录' }
if (site.has_oneid) {
  const types = site.sso_im_types.length > 0 ? site.sso_im_types : ['']
  types.forEach(type => {
    showSSOButton(labelMap[type] ?? 'OneID 登录', () => {
      window.location.href = '/auth/oneid'
    })
  })
}
```

### `GET /favicon.ico`

站点图标，返回站点 Logo 图片。

- **权限：** 公开
- **响应：** 图片二进制数据，不支持 JSON。未上传 Logo 时返回内嵌默认 SVG 图标。

### `GET /logo`

获取站点 Logo 图片。

- **权限：** 公开
- **响应：** 图片二进制数据，不支持 JSON。
  - 已上传 Logo：返回上传的图片（`Content-Type` 为上传时的 MIME 类型），`Cache-Control: public, max-age=3600`
  - 未上传 Logo：返回内嵌默认 SVG 图标（`Content-Type: image/svg+xml`），`Cache-Control: public, max-age=60`（短缓存，上传后快速生效）

### `GET /static/*`

静态资源文件（CSS、JS 等）。

- **权限：** 公开

### `GET /health`

健康检查接口，用于负载均衡器或监控系统探活。

- **权限：** 公开
- **响应：** 始终返回 JSON `{"status":"ok"}`

---

## 九、LightClaw 对接接口

### `GET /openclaw/lightclaw/token`

获取实例 ProxyToken 及 HMAC-SHA256 签名信息，供前端传递给 LightClaw 组件完成鉴权。

- **权限：** 登录用户
- **响应：** 始终返回 JSON
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 实例数据库 ID（需属于当前用户） |

- **JSON 响应：**

```json
{
  "businessCode": "clawpro",
  "callbackUrl": "http://example.com/openclaw/lightclaw/auth",
  "proxyToken": "sk-xxx...",
  "sign": "14219914fc88ea021f566be8...",
  "timestamp": 1775127171
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| businessCode | string | 固定值 `"clawpro"`，产品标识 |
| callbackUrl | string | LightClaw 回调鉴权地址，由服务启动参数 `--domain` 拼接而成 |
| proxyToken | string | 实例的代理 Token，用于 LightClaw 鉴权 |
| timestamp | int | Unix 时间戳（秒） |
| sign | string | HMAC-SHA256 签名，签名算法见下方说明 |

> **签名算法：**
>
> 1. 取 `businessCode`、`callbackUrl`、`proxyToken`、`timestamp` 四个参数，按 key 字母序排序拼接为 `k1=v1&k2=v2&...` 形式
> 2. 末尾追加 `&secretId=<LightClawSecretId>&secretKey=<LightClawSecretKey>`
> 3. 以 `secretKey` 为密钥，对整个字符串做 HMAC-SHA256，输出 hex 编码

失败时返回错误信息：

- `405 {"error": "method not allowed"}`
- `404 {"error": "实例不存在"}`
- `500 {"error": "签名服务未配置"}` / `500 {"error": "服务域名未配置"}` / `500 {"error": "ProxyToken 未配置"}`

---

### `POST /openclaw/lightclaw/auth`

LightClaw 后端回调鉴权接口。LightClaw 服务使用此接口验证 ProxyToken 有效性并获取用户信息。

- **权限：** 公开（供 LightClaw 后端调用，不需要用户登录）
- **Content-Type：** `application/json`
- **响应：** 始终返回 JSON
- **请求体：**

```json
{
  "product": "clawpro",
  "accessToken": "sk-xxx..."
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| product | string | 是 | 产品标识，必须为 `"clawpro"` |
| accessToken | string | 是 | 实例的 ProxyToken |

- **JSON 响应：**

```json
{
  "code": 0,
  "data": {
    "user_id": 2,
    "username": "kyloadmin",
    "id": 64,
    "instance_id": "ins-da2d8hvm"
  },
  "message": "OK"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| data.user_id | int | 用户 ID |
| data.username | string | 用户名 |
| data.id | int | 实例数据库 ID |
| data.instance_id | string | CVM 实例 ID |

失败时返回错误信息：

- `400 {"error": "product 不匹配"}` / `400 {"error": "请求体格式错误"}`
- `401 {"error": "accessToken 无效或不存在"}`
- `404 {"error": "关联用户数据异常"}`
- `405 {"error": "method not allowed"}`

---

### `POST /openclaw/lightclaw/run-command`

向指定 CVM 实例下发 Shell/PowerShell 命令（通过腾讯云 TAT RunCommand API），有审计日志。

- **权限：** 登录用户
- **Content-Type：** `application/json`
- **响应：** 始终返回 JSON
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 实例数据库 ID（Query 参数，需属于当前用户） |

- **请求体：**

```json
{
  "Content": "bHMgLWxh",
  "InstanceIds": ["ins-da2d8hvm"],
  "CommandType": "SHELL"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| Content | string | 是 | Shell 命令内容（Base64 编码） |
| InstanceIds | string[] | 是 | CVM 实例 ID 数组，长度必须为 1 且与 `id` 对应的实例匹配 |
| CommandType | string | 否 | 命令类型，默认 `"SHELL"`，可选 `"POWERSHELL"` |

- **JSON 响应：**

```json
{
  "code": 0,
  "data": {
    "Response": {
      "CommandId": "cmd-3ljjauix",
      "InvocationId": "inv-w1u99t0ckk",
      "RequestId": "05b17ad8-4f4b-4820-96d0-650181340b3e"
    }
  },
  "message": "OK",
  "tid": "...",
  "timestamp": "..."
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| data.Response.CommandId | string | 命令 ID |
| data.Response.InvocationId | string | 执行 ID，可用于后续查询 |
| data.Response.RequestId | string | 腾讯云请求 ID |

失败时返回错误信息：

- `400 {"error": "Content 为空"}` / `400 {"error": "请求体格式错误"}`
- `403 {"error": "InstanceIds 与实例不匹配"}`
- `404 {"error": "实例不存在"}`
- `405 {"error": "method not allowed"}`
- TAT Agent 未在线时返回 `code: "InvalidInstance.NotRunning"`
- 云 API 错误返回对应的 `code`（如 `"InvalidParameterValue"`）

---

### `POST /openclaw/lightclaw/describe-invocations`

查询命令执行记录（封装腾讯云 TAT DescribeInvocations），会校验返回结果中的 InstanceId 是否与当前实例匹配。

- **权限：** 登录用户
- **Content-Type：** `application/json`
- **响应：** 始终返回 JSON
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 实例数据库 ID（Query 参数，需属于当前用户） |

- **请求体：**

```json
{
  "InvocationIds": ["inv-w1u99t0ckk"]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| InvocationIds | string[] | 否 | 执行 ID 列表，为空时返回所有记录 |

- **JSON 响应：**

```json
{
  "code": 0,
  "data": {
    "Response": {
      "TotalCount": 1,
      "InvocationSet": [
        {
          "InvocationId": "inv-w1u99t0ckk",
          "CommandId": "cmd-3ljjauix",
          "InvocationStatus": "SUCCESS",
          "CommandContent": "bHMgLWxh",
          "InvocationTaskBasicInfoSet": [
            {
              "InvocationTaskId": "invt-u1u9br05dd",
              "InstanceId": "ins-da2d8hvm",
              "TaskStatus": "SUCCESS"
            }
          ]
        }
      ],
      "RequestId": "..."
    }
  },
  "message": "OK",
  "tid": "...",
  "timestamp": "..."
}
```

失败时返回错误信息：

- `404 {"error": "实例不存在"}`
- `405 {"error": "method not allowed"}`
- InstanceId 不匹配时返回 `code: "InvalidInstance"`
- 云 API 错误返回对应的 `code`

---

### `POST /openclaw/lightclaw/describe-invocation-tasks`

查询命令执行任务详情及输出（封装腾讯云 TAT DescribeInvocationTasks），会校验返回结果中的 InstanceId 是否与当前实例匹配。

- **权限：** 登录用户
- **Content-Type：** `application/json`
- **响应：** 始终返回 JSON
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 实例数据库 ID（Query 参数，需属于当前用户） |

- **请求体：**

```json
{
  "InvocationTaskIds": ["invt-u1u9br05dd"],
  "HideOutput": false
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| InvocationTaskIds | string[] | 否 | 任务 ID 列表，为空时返回所有记录 |
| HideOutput | bool | 否 | 是否隐藏输出内容，默认 `false` |

- **JSON 响应：**

```json
{
  "code": 0,
  "data": {
    "Response": {
      "TotalCount": 1,
      "InvocationTaskSet": [
        {
          "InvocationTaskId": "invt-u1u9br05dd",
          "InstanceId": "ins-da2d8hvm",
          "TaskStatus": "SUCCESS",
          "ExitCode": 0,
          "Output": "dG90YWwgNjQK...",
          "WorkingDirectory": "/root",
          "Username": "root"
        }
      ],
      "RequestId": "..."
    }
  },
  "message": "OK",
  "tid": "...",
  "timestamp": "..."
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| Output | string | 命令输出内容（Base64 编码），`HideOutput=true` 时为空 |
| ExitCode | int | 命令退出码，`0` 为成功 |
| TaskStatus | string | 任务状态（`SUCCESS`、`FAILED`、`RUNNING` 等） |

失败时返回错误信息：

- `404 {"error": "实例不存在"}`
- `405 {"error": "method not allowed"}`
- InstanceId 不匹配时返回 `code: "InvalidInstance"`
- 云 API 错误返回对应的 `code`

---

## 十、Memory Pro 记忆管理

> Memory Pro 版为实例提供基于腾讯云向量数据库（VDB）的企业级记忆服务。以下接口覆盖管控端管理 + 用户端查询两部分。

### 接口总览

| # | 方法 | 路径 | 权限 | 状态 |
|---|------|------|------|------|
| 1 | GET | `/admin/memory/overview` | 管理员 | ✅ |
| 2 | POST | `/admin/memory/pro/activate` | 管理员 | ✅ |
| 3 | POST | `/admin/memory/pro/release` | 管理员 | ✅ |
| 4 | POST | `/admin/memory/plan/switch` | 管理员 | ✅ |
| 5 | GET | `/admin/memory/instances` | 管理员 | ✅ |
| 6 | GET | `/admin/memory/default-plan` | 管理员 | ✅ |
| 7 | PUT | `/admin/memory/default-plan` | 管理员 | ✅ |
| 8 | GET | `/admin/memory/group-policies` | 管理员 | ✅ |
| 9 | POST | `/admin/memory/group-policy` | 管理员 | ✅ |
| 10 | PUT | `/admin/memory/group-policy` | 管理员 | ✅ |
| 11 | POST | `/admin/memory/group-policy/delete` | 管理员 | ✅ |
| 12 | GET | `/openclaw/memory/library/detail` | 登录用户 | ✅ |
| 13 | POST | `/openclaw/memory/plan/switch` | 登录用户 | ⏸️ 已屏蔽 |
| 14 | GET | `/openclaw/memory/config` | 登录用户 | ⏸️ 已屏蔽 |
| 15 | GET | `/openclaw/memory/task` | 登录用户 | ⏸️ 已屏蔽 |

> 旧版兼容接口（`GET /openclaw/memory-tdai-status`、`GET/PUT /admin/memory-tdai/config`）见上方 [二、实例管理](#二实例管理openclaw) 和 [四、管理后台](#四管理后台) 章节。

---

### 10.1 服务概览

```
GET /admin/memory/overview
```

- **权限：** 管理员

**curl 示例**：
```bash
curl -s http://localhost:9999/admin/memory/overview \
  -H "Authorization: Bearer clawpro-dev-token"
```

**响应**：
```json
{
  "total_instances": 100,
  "plan_stats": {
    "OFF": 50,
    "FREE": 30,
    "PRO": 20
  },
  "pro_capacity": {
    "total": 1000,
    "used": 200,
    "status": "online",
    "memory_pro_id": "mp-xxxxx"
  },
  "memory_default_plan": "free"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| total_instances | int | 有 instance_id 的实例总数 |
| plan_stats | object | 按 current_plan 分组的实例数 |
| `pro_capacity.total` | int | Pro VDB 记忆空间上限 |
| `pro_capacity.used` | int | Pro VDB 已使用空间 |
| `pro_capacity.status` | string | Pro 实例状态（未开通时不返回） |
| `pro_capacity.memory_pro_id` | string | Pro 实例 ID（未开通时不返回） |
| memory_default_plan | string | 增量实例默认计划：`off`/`free`/`pro` |

---

### 10.2 开通 Pro 服务

```
POST /admin/memory/pro/activate
```

- **权限：** 管理员
- **Content-Type：** `application/json`

**请求体**：
```json
{"memory_limit": 1000}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| memory_limit | int | 是 | 记忆空间上限，必须 > 0 |

**curl 示例**：
```bash
curl -s -X POST http://localhost:9999/admin/memory/pro/activate \
  -H "Authorization: Bearer clawpro-dev-token" \
  -H "Content-Type: application/json" \
  -d '{"memory_limit": 1000}'
```

**成功响应**：
```json
{
  "memory_pro_id": "mp-xxxxx",
  "vdb_instance_id": "vdb-xxxxx"
}
```

**错误响应**：
- `400` — `memory_limit 必须大于 0`
- `409` — `Pro 服务已开通，请勿重复创建`
- `500` — `创建 VDB 实例失败: ...`

---

### 10.3 关闭 Pro 服务

```
POST /admin/memory/pro/release
```

- **权限：** 管理员
- **前置条件**：所有实例的 Pro 记忆库必须已关闭（`current_plan != PRO` 且 `pool_id` 为空）。

**curl 示例**：
```bash
curl -s -X POST http://localhost:9999/admin/memory/pro/release \
  -H "Authorization: Bearer clawpro-dev-token"
```

**成功响应**：
```json
{"ok": true}
```

**错误响应**：
- `404` — `未找到 Pro 服务实例`
- `409` — `仍有 N 个实例在使用 Pro 记忆库（ins-xxx, ins-yyy...），请先将这些实例切换到 OFF 后再关闭 Pro 服务`

---

### 10.4 批量切换记忆计划

```
POST /admin/memory/plan/switch
```

- **权限：** 管理员
- **Content-Type：** `application/json`

**请求体**：
```json
{
  "instance_ids": ["ins-aaa", "ins-bbb"],
  "target_plan": "pro"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_ids | string[] | 是 | CVM 实例 ID 列表 |
| target_plan | string | 是 | 目标计划：`off` / `free` / `pro` |

**curl 示例**：
```bash
curl -s -X POST http://localhost:9999/admin/memory/plan/switch \
  -H "Authorization: Bearer clawpro-dev-token" \
  -H "Content-Type: application/json" \
  -d '{"instance_ids": ["ins-aaa"], "target_plan": "free"}'
```

**响应**：
```json
{
  "target_plan": "pro",
  "results": [
    {
      "instance_id": "ins-aaa",
      "status": "accepted",
      "task_id": 123
    },
    {
      "instance_id": "ins-bbb",
      "status": "rejected",
      "reason": "network_unreachable",
      "message": "Agent所在CVM (ins-bbb) 到 记忆空间所在VDB (vdb-pxcnites, http://10.0.0.5:80) 网络不通，无法切换到 Pro 版。请检查 CVM 与 VDB 所在 VPC 是否连通，以及 CVM 与 VDB 的安全组规则是否放通。",
      "detail": {
        "cvm_id": "ins-bbb",
        "vdb_instance_id": "vdb-pxcnites"
      },
      "error": "Agent所在CVM (ins-bbb) 到 记忆空间所在VDB (vdb-pxcnites, http://10.0.0.5:80) 网络不通，无法切换到 Pro 版。请检查 CVM 与 VDB 所在 VPC 是否连通，以及 CVM 与 VDB 的安全组规则是否放通。"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `results[].instance_id` | string | 实例 ID |
| `results[].status` | string | 单实例处理状态：`accepted`=已创建切换任务；`rejected`=前置校验失败，未创建任务 |
| `results[].task_id` | uint | 任务 ID（`status=accepted` 时返回，可用于轮询任务状态） |
| `results[].reason` | string | 拒绝原因枚举（`status=rejected` 时返回），例如 `network_unreachable` |
| `results[].message` | string | 面向用户展示的拒绝原因文案（`status=rejected` 时返回） |
| `results[].detail` | object | 拒绝原因的结构化补充信息（`status=rejected` 时返回） |
| `results[].detail.cvm_id` | string | 发生拒绝的 CVM 实例 ID |
| `results[].detail.vdb_instance_id` | string | 记忆空间所在 VDB 实例 ID（`vdb-xxx`） |
| `results[].error` | string | 兼容老字段，内容同 `message` 或其它错误原因（失败时返回） |

**全部实例均被拒绝时的响应**：

当请求中的实例全部未创建任务（例如均因 CVM 到 VDB 网络不通被拒绝）时，接口返回 HTTP `422`，并在顶层 `error` 字段中返回聚合后的错误文案：

```json
{
  "error": "以下 Agent 所在 CVM (ins-aaa, ins-bbb) 到 VDB (vdb-pxcnites, http://10.0.0.5:80) 网络不通，无法切换到 Pro 版。请检查 CVM 与 VDB 所在 VPC 是否连通，以及 CVM 与 VDB 的安全组规则是否放通。",
  "target_plan": "pro",
  "results": [
    {
      "instance_id": "ins-aaa",
      "status": "rejected",
      "reason": "network_unreachable",
      "message": "...",
      "detail": {"cvm_id": "ins-aaa", "vdb_instance_id": "vdb-pxcnites"},
      "error": "..."
    }
  ]
}
```

> **说明**：切换是异步的。`status=accepted` 的实例会返回 `task_id`，后端通过任务调度器（3 秒轮询）执行实际切换；`status=rejected` 的实例不会创建任务。切换到 Pro 版前会执行 CVM 到 VDB 的网络连通性预检，网络不通时返回 `network_unreachable`。

---

### 10.5 实例列表

```
GET /admin/memory/instances
```

- **权限：** 管理员

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20，最大 100 |
| keyword | string | 否 | 搜索关键词（匹配 instance_id 或实例名称） |
| plan | string | 否 | 计划过滤：`OFF` / `FREE` / `PRO` |

**curl 示例**：
```bash
# 不带过滤
curl -s "http://localhost:9999/admin/memory/instances?page=1&page_size=20" \
  -H "Authorization: Bearer clawpro-dev-token"

# 按计划过滤
curl -s "http://localhost:9999/admin/memory/instances?plan=FREE" \
  -H "Authorization: Bearer clawpro-dev-token"

# 搜索
curl -s "http://localhost:9999/admin/memory/instances?keyword=ins-abc" \
  -H "Authorization: Bearer clawpro-dev-token"
```

**响应**：
```json
{
  "total": 100,
  "page": 1,
  "page_size": 20,
  "items": [
    {
      "instance_id": "ins-xxxxx",
      "instance_name": "my-claw",
      "agent_type": "openclaw",
      "creator_name": "admin",
      "current_plan": "PRO",
      "switch_status": "",
      "last_switched_at": "2024-01-01T00:00:00Z",
      "group_id": 28,
      "group_full_path": "A公司/技术部"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `items[].instance_id` | string | CVM 实例 ID |
| `items[].instance_name` | string | 实例名称 |
| `items[].agent_type` | string | Agent 类型（如 `openclaw` / `hermes` 等），用于前端区分实例形态 |
| `items[].creator_name` | string | 创建者用户名 |
| `items[].current_plan` | string | 当前计划：`OFF`/`FREE`/`PRO` |
| `items[].switch_status` | string | 切换中状态（空字符串 = 无进行中切换） |
| `items[].last_switched_at` | string\|null | 最后切换时间 |
| `items[].group_id` | int\|null | 用户所属分组 ID（未分组时返回 null） |
| `items[].group_full_path` | string\|null | 用户所属分组完整路径（如 `A公司/技术部`，未分组时返回 null） |

---

### 10.6 默认记忆计划（查询）

```
GET /admin/memory/default-plan
```

- **权限：** 管理员

**curl 示例**：
```bash
curl -s http://localhost:9999/admin/memory/default-plan \
  -H "Authorization: Bearer clawpro-dev-token"
```

**响应**：
```json
{"memory_default_plan": "free"}
```

> 兼容逻辑：若 `memory_default_plan` 为空，根据旧开关 `memory_tdai_enable` 推导（`true` → `free`，`false` → `off`）。

---

### 10.7 默认记忆计划（更新）

```
PUT /admin/memory/default-plan
```

- **权限：** 管理员
- **Content-Type：** `application/json`

**请求体**：
```json
{"memory_default_plan": "pro"}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| memory_default_plan | string | 是 | `off` / `free` / `pro` |
| clear_policies | bool | 否 | 是否清空分组策略表，默认 `false` |

> **新增说明**：`clear_policies` 为本迭代新增字段。管理员在「记忆管理」页面手动切换预设策略时（如滑块切换 off→free），应传 `clear_policies: true` 以清空已有的分组策略。开通 Pro 服务时联动修改预设策略无需传此字段。

**curl 示例**：
```bash
curl -s -X PUT http://localhost:9999/admin/memory/default-plan \
  -H "Authorization: Bearer clawpro-dev-token" \
  -H "Content-Type: application/json" \
  -d '{"memory_default_plan": "pro"}'

# 切换预设策略时同时清空分组策略
curl -s -X PUT http://localhost:9999/admin/memory/default-plan \
  -H "Authorization: Bearer clawpro-dev-token" \
  -H "Content-Type: application/json" \
  -d '{"memory_default_plan": "off", "clear_policies": true}'
```

**成功响应**：
```json
{"ok": true, "memory_default_plan": "pro"}
```

**错误响应**：
- `400` — `memory_default_plan 必须是 off/free/pro，得到 "xxx"`

---

### 10.8 记忆库数据查询

```
GET /openclaw/memory/library/detail
```

- **权限：** 登录用户（普通用户仅能查自己的实例，admin 可查所有）

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_id | string | 是 | CVM 实例 ID |
| type | string | 是 | 记录类型：`persona` / `scene` / `memory` / `conversation` |
| sub_type | string | 否 | 原子记忆子类型（仅 `type=memory` 时有效）：`persona`（偏好） / `episodic`（事实） / `instruction`（指令） |
| page | int | 否 | 页码，默认 1（`memory` / `conversation` 需要） |
| page_size | int | 否 | 每页条数，默认 20（Pro 最大 100，Free 最大 10） |
| start_time | string | 否 | 起始时间，格式 `2026-04-14 20:00:00`（仅 `type=conversation`） |
| end_time | string | 否 | 结束时间，格式 `2026-04-14 21:00:00`（仅 `type=conversation`） |

**curl 示例**：
```bash
# 查询人设摘要（persona）
curl -s "http://localhost:9999/openclaw/memory/library/detail?instance_id=ins-xxx&type=persona" \
  -H "Authorization: Bearer clawpro-dev-token"

# 查询场景记忆（scene）
curl -s "http://localhost:9999/openclaw/memory/library/detail?instance_id=ins-xxx&type=scene" \
  -H "Authorization: Bearer clawpro-dev-token"

# 查询原子记忆（memory），按子类型过滤
curl -s "http://localhost:9999/openclaw/memory/library/detail?instance_id=ins-xxx&type=memory&page=1&page_size=20&sub_type=persona" \
  -H "Authorization: Bearer clawpro-dev-token"

# 查询对话记忆（conversation），带时间范围
curl -s "http://localhost:9999/openclaw/memory/library/detail?instance_id=ins-xxx&type=conversation&page=1&page_size=20&start_time=2026-04-01+00:00:00&end_time=2026-04-16+23:59:59" \
  -H "Authorization: Bearer clawpro-dev-token"
```

#### 10.8.1 type=persona（个性化记忆 / 人设摘要）

返回单条文档，字段为 `document`（非 `documents`）。

**响应（Pro 版）**：
```json
{
  "instance_id": "ins-xxxxx",
  "space_id": "space-xxxxx",
  "type": "persona",
  "total_count": 1,
  "document": { "id": "...", "content": "用户是一名软件工程师..." }
}
```

**响应（Free 版）**：
```json
{
  "instance_id": "ins-xxxxx",
  "type": "persona",
  "source": "local",
  "total_count": 1,
  "document": { "content": "# 人设摘要\n用户是一名软件工程师..." }
}
```

#### 10.8.2 type=scene（场景记忆）

返回场景 Markdown 文件列表。

**响应（Pro 版）**：
```json
{
  "instance_id": "ins-xxxxx",
  "space_id": "space-xxxxx",
  "type": "scene",
  "total_count": 13,
  "documents": [{ "id": "...", "content": "..." }]
}
```

**响应（Free 版）**：

| 字段 | 类型 | 说明 |
|------|------|------|
| fileName | string | 文件名（含 `.md` 后缀），建议去掉后缀作为标题 |
| summary | string | 场景摘要 |
| heat | int | 热度值 |
| created | string | 创建时间（ISO8601） |
| updated | string | 更新时间（ISO8601） |
| body | string | Markdown 正文（超 24KB 返回占位文本） |

```json
{
  "instance_id": "ins-xxxxx",
  "type": "scene",
  "source": "local",
  "total_count": 13,
  "documents": [
    {
      "fileName": "技术架构与工程实践.md",
      "summary": "**BATCH32关键决策点**: 微服务架构演进...",
      "heat": 85,
      "created": "2026-04-13T03:37:21.141Z",
      "updated": "2026-04-15T18:33:00.000Z",
      "body": "## 技术架构概述\n\n### 微服务拆分策略\n- 按业务域拆分..."
    }
  ]
}
```

#### 10.8.3 type=memory（原子记忆）

支持 `sub_type` 过滤：`persona`（偏好）/ `episodic`（事实）/ `instruction`（指令）。不传则返回全部。

**响应（Pro 版）**：
```json
{
  "instance_id": "ins-xxxxx",
  "space_id": "space-xxxxx",
  "type": "memory",
  "total_count": 42,
  "documents": [
    { "id": "...", "content": "用户偏好使用 VS Code", "type": "persona", "priority": 85, "timestamp": "2026-04-13T10:00:00Z" }
  ]
}
```

**响应（Free 版）**：
```json
{
  "instance_id": "ins-xxxxx",
  "type": "memory",
  "source": "local",
  "total_count": 25,
  "documents": [
    {
      "id": "rec_001", "content": "用户偏好使用 VS Code", "type": "persona",
      "priority": 85, "scene_name": "技术架构与工程实践",
      "timestamps": ["2026-04-13T10:00:00Z"],
      "createdAt": "2026-04-13T03:37:21Z", "updatedAt": "2026-04-15T18:33:00Z"
    }
  ]
}
```

#### 10.8.4 type=conversation（对话记录）

支持 `start_time` / `end_time` 时间范围过滤，按时间倒序返回。

**响应（Pro 版）**：
```json
{
  "instance_id": "ins-xxxxx",
  "space_id": "space-xxxxx",
  "type": "conversation",
  "total_count": 100,
  "documents": [
    { "id": "...", "role": "user", "content": "帮我看看这个报错", "timestamp": "2026-04-15T10:30:00Z" }
  ]
}
```

**响应（Free 版）**：
```json
{
  "instance_id": "ins-xxxxx",
  "type": "conversation",
  "source": "local",
  "total_count": 50,
  "documents": [
    {
      "id": "rec_001", "role": "user", "message": "帮我看看这个报错",
      "sessionKey": "session_20260415", "sessionId": "abc123",
      "recordedAt": "2026-04-15T10:30:00Z", "timestamp": 1744684200
    }
  ]
}
```

**记忆库查询通用错误响应**：
- `400` — `实例未开通记忆服务（当前为 OFF）`
- `400` — `type 必须是 persona/scene/memory/conversation`
- `403` — `无权访问该实例`
- `404` — `实例 xxx 未找到记忆配置`
- `500` — `本地记忆数据量过大（超过 TAT 24KB 输出限制），请缩小查询范围或使用分页`

---

### 10.9 已屏蔽接口（待联调后开放）

> 以下接口代码已实现，但路由注册被注释。联调完成后取消注释即可启用。

#### 10.9.1 用户端切换记忆计划

```
POST /openclaw/memory/plan/switch
```

- **权限：** 登录用户
- **Content-Type：** `application/json`
- **请求体**：`{"instance_id": "ins-xxxxx", "target_plan": "pro"}`
- **响应**：`{"task_id": 123, "job_type": "SWITCH_TO_PRO", "state": "PENDING", ...}`
- **错误**：`409` — 有进行中的切换操作

#### 10.9.2 查询实例记忆配置

```
GET /openclaw/memory/config?instance_id=ins-xxxxx
```

- **权限：** 登录用户
- **响应**：返回 `current_plan`、`switch_status`、`last_task` 等字段

#### 10.9.3 查询任务详情

```
GET /openclaw/memory/task?task_id=123
```

- **权限：** 登录用户
- **响应**：返回完整的 TdaiJob 结构体（`job_type`、`state`、`progress`、`attempt` 等）

---

### 10.10 枚举值参考

#### 记忆计划 (current_plan / desired_plan)

| 值 | 说明 |
|------|------|
| OFF | 关闭记忆 |
| FREE | 免费版（本地 sqlite） |
| PRO | Pro 版（远端 VDB） |

#### 切换状态 (switch_status)

| 值 | 说明 |
|------|------|
| "" (空) | 无进行中切换 |
| SWITCHING_TO_OFF | 正在切换到 OFF |
| SWITCHING_TO_FREE | 正在切换到 FREE |
| SWITCHING_TO_PRO | 正在切换到 PRO |

#### 任务类型 (job_type)

| 值 | 说明 |
|------|------|
| SWITCH_TO_FREE | 切换到 Free |
| SWITCH_TO_OFF | 切换到 Off |
| SWITCH_TO_PRO | 切换到 Pro |

#### 任务状态 (state)

| 值 | 说明 |
|------|------|
| PENDING | 等待执行 |
| RUNNING | 执行中 |
| SUCCEEDED | 成功 |
| FAILED | 失败（已用完重试次数，或不可重试错误） |
| CANCELED | 已取消 |

---

### 10.11 状态迁移约束

```
OFF ──→ FREE     ✅
OFF ──→ PRO      ✅
FREE ──→ OFF     ✅
FREE ──→ PRO     ✅
PRO ──→ OFF      ✅
PRO ──→ FREE     ❌（需先 PRO → OFF，再 OFF → FREE）
```

**任务重试机制**：
- 最多 3 次重试，指数退避（5s → 30s → 180s）
- 不可重试错误（参数非法、状态冲突）直接标记 FAILED
- 任务失败后自动回滚 `switch_status` 为空

---

### 10.12 查询分组策略列表

```
GET /admin/memory/group-policies
```

- **权限：** 管理员

**curl 示例**：
```bash
curl -s http://localhost:9999/admin/memory/group-policies \
  -H "Authorization: Bearer clawpro-dev-token"
```

**响应**：
```json
{
  "ok": true,
  "policies": [
    {
      "priority": 1,
      "plan": "free",
      "groups": [
        { "group_id": 28, "group_name": "技术部", "full_path": "A公司/技术部" },
        { "group_id": 29, "group_name": "产品部", "full_path": "A公司/产品部" }
      ]
    },
    {
      "priority": 2,
      "plan": "pro",
      "groups": [
        { "group_id": 30, "group_name": "行政部", "full_path": "A公司/行政部" }
      ]
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `policies[].priority` | int | 策略优先级：1（低优先）/ 2（高优先） |
| `policies[].plan` | string | 记忆版本：`off` / `free` / `pro` |
| `policies[].groups[].group_id` | int | 分组 ID |
| `policies[].groups[].group_name` | string | 分组名称 |
| `policies[].groups[].full_path` | string | 分组完整路径（如 `A公司/技术部`） |

**说明**：
- `policies` 数组最多 2 条（priority=1 和 priority=2）
- 已删除的分组会被自动过滤，不会出现在返回结果中
- 预设策略（`default_plan`）请从 `GET /admin/memory/overview` 的 `memory_default_plan` 字段获取

---

### 10.13 创建分组策略

```
POST /admin/memory/group-policy
```

- **权限：** 管理员
- **Content-Type：** `application/json`

**请求体**：
```json
{ "priority": 1, "plan": "free", "group_ids": [28, 29] }
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| priority | int | 是 | 策略优先级：1 或 2 |
| plan | string | 是 | 记忆版本：`off` / `free` / `pro` |
| group_ids | []int | 是 | 选中的分组 ID 列表（不能为空） |

**curl 示例**：
```bash
curl -s -X POST http://localhost:9999/admin/memory/group-policy \
  -H "Authorization: Bearer clawpro-dev-token" \
  -H "Content-Type: application/json" \
  -d '{"priority": 1, "plan": "free", "group_ids": [28, 29]}'
```

**成功响应**：
```json
{ "ok": true }
```

**错误响应**：
- `400` — plan 无效 / plan 与预设策略相同 / priority 无效 / group_ids 为空 / 部分 group_id 不存在
- `409` — 该 priority 已存在策略（应用 PUT 修改）/ 部分分组已被其他策略占用

**前端传参折叠规则**：

假设分组树如下：
```
- 技术部 (id=28)
  - 大模型组 (id=33)
    - 模型一组 (id=38)
    - 模型二组 (id=39)
```

| 用户操作 | 前端应传的 group_ids | 说明 |
|---------|---------------------|------|
| 只勾选「模型一组」 | `[38]` | 只传被选中的节点 |
| 勾选「模型一组」+「模型二组」 | `[33]` | 大模型组的所有直接子节点都被选中 → 折叠为父节点 |
| 直接勾选「大模型组」 | `[33]` | 直接选父节点 |
| 勾选「大模型组」+「技术部下其他所有子节点」 | `[28]` | 技术部全选 → 折叠为技术部 |

> **规则**：同一条策略内，如果某节点的所有直接子节点都被选中，前端应折叠为该父节点 ID 再传给后端。后端只存前端传来的 ID，不做展开也不做折叠。

---

### 10.14 修改分组策略（全量替换）

```
PUT /admin/memory/group-policy
```

- **权限：** 管理员
- **Content-Type：** `application/json`
- **幂等性：** 同样参数调多次结果一致

**请求体**：
```json
{ "priority": 1, "plan": "pro", "group_ids": [28, 29, 30] }
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| priority | int | 是 | 修改哪条策略：1 或 2 |
| plan | string | 是 | 新的记忆版本 |
| group_ids | []int | 是 | 新的分组 ID 列表（全量替换） |

**curl 示例**：
```bash
curl -s -X PUT http://localhost:9999/admin/memory/group-policy \
  -H "Authorization: Bearer clawpro-dev-token" \
  -H "Content-Type: application/json" \
  -d '{"priority": 1, "plan": "pro", "group_ids": [28, 29, 30]}'
```

**成功响应**：
```json
{ "ok": true }
```

**说明**：
- 全量替换：每次提交完整的 priority + plan + group_ids，后端直接覆盖旧数据
- 幂等：同样参数调多次结果一样

**错误响应**：
- `400` — plan 无效 / plan 与预设策略相同 / plan 与另一条策略冲突 / group_ids 为空 / 部分 group_id 不存在
- `404` — 该 priority 不存在策略（应先 POST 创建）
- `409` — 部分分组已被其他策略占用

---

### 10.15 删除分组策略

```
POST /admin/memory/group-policy/delete
```

- **权限：** 管理员
- **Content-Type：** `application/json`

**请求体**：
```json
{ "priority": 1 }
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| priority | int | 是 | 要删除的策略：1 或 2 |

**curl 示例**：
```bash
curl -s -X POST http://localhost:9999/admin/memory/group-policy/delete \
  -H "Authorization: Bearer clawpro-dev-token" \
  -H "Content-Type: application/json" \
  -d '{"priority": 1}'
```

**成功响应**：
```json
{ "ok": true }
```

**说明**：
- 删除 priority=1 时，如果 priority=2 还有数据，会自动降级为 priority=1
- 硬删除，不可恢复（后端有日志记录删除前的数据）

---

### 10.16 分组策略匹配规则与业务约束

#### 匹配规则

创建 Agent 时，根据用户选定的分组（`group_id`）决定记忆版本：

1. 查该分组的祖先链（含自身），通过 `group_closure` 表获取
2. 在 `memory_plan_group_policies` 表中查找祖先链中的分组
3. 命中 → 使用该策略的 plan
4. 未命中 → 使用预设策略（`memory_default_plan`）

**继承语义**：选中某个节点 = 覆盖该节点及其所有子孙节点（含未来新增的子节点）。后端只存被选中的节点 ID，不展开子节点。

#### 业务约束

**plan 可选值**（取决于 Pro 服务是否开通）：

| Pro 状态 | 预设策略可选 | 分组策略 plan 可选 | 最多几条分组策略 |
|---------|------------|-----------------|--------------|
| 未开通 | off / free | free（预设=off时）或 off（预设=free时） | 1 条 |
| 已开通 | off / free / pro | 预设之外的另外两个值 | 2 条 |

**核心规则**：预设策略占一个值，每条分组策略各占一个值，三者不能重复。

**互斥约束**（后端 UNIQUE 索引 + 前端校验）：
1. **同一分组不能出现在两条策略中**
2. **父子节点不能跨策略**：策略 1 选了「技术部」→ 策略 2 不能选「后台组」（技术部的子节点）；反之亦然
3. **分组策略的 plan 不能与预设策略相同**
4. **两条策略的 plan 不能相同**

**折叠规则**（前端负责）：
- 同一条策略内，某节点的所有直接子节点都被选中 → 自动折叠为父节点
- 折叠只在同一条策略内生效，跨策略不触发

---

## 十一、云端浏览器（Browser VNC）

云端浏览器功能允许用户通过 noVNC 在浏览器中实时查看和操作 CVM 实例上运行的 Chrome 浏览器。AI Agent 通过 CDP（Chrome DevTools Protocol）控制浏览器，用户可以实时观看 AI 操作，也可以手动接管浏览器控制权。

**版本说明：**
- **v1（旧版 - 仅浏览器模式）**：使用 supervisor 管理进程，仅安装 Chrome + noVNC + openbox，无桌面环境
- **v2（新版 - 云桌面模式）**：使用 systemd 管理进程，安装完整 XFCE4 桌面 + Chrome + noVNC + fcitx5 中文输入法

check 接口会返回 `desktop_mode` 和 `upgrade_available` 字段，前端据此判断是否需要升级。升级操作复用 install 接口（自动完成 supervisor → systemd 迁移 + 桌面组件安装）。

**前置条件：**
- 管理员已在后台开启云端浏览器功能（`POST /admin/config` 设置 `browser_vnc_enable=true`）
- CVM 实例处于 RUNNING 状态
- CVM 实例已安装 VNC 环境（通过 `POST /openclaw/browser-vnc-install`）

### 11.1 检查 VNC 环境

### `GET /openclaw/browser-vnc-check`

检查指定实例的 VNC 云端浏览器环境是否已安装并正常运行。通过 TAT 在 CVM 上执行检查脚本，返回各组件的就绪状态。兼容 supervisor（旧版）和 systemd（新版）两种服务管理模式。

- **权限：** 登录用户（仅可查看自己的实例）
- **超时：** TAT 脚本执行超时 30 秒
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID（Query 参数） |

- **成功响应（新版完整云桌面）：**

```json
{
  "ok": true,
  "data": {
    "ready": true,
    "desktop_mode": "full",
    "upgrade_available": false,
    "service_manager": "systemd",
    "os_name": "Ubuntu Server 24.04 LTS 64bit",
    "elapsed_ms": 1523,
    "checks": {
      "packages": "ok",
      "novnc": "ok",
      "chrome": "ok",
      "systemd_config": "ok",
      "services": "ok",
      "ports": "ok",
      "cdp_owner": "ok",
      "ssl_cert": "ok",
      "fcitx5": "ok",
      "cjk_fonts": "ok",
      "novnc_patch": "ok"
    }
  }
}
```

- **成功响应（旧版仅浏览器，可升级）：**

```json
{
  "ok": true,
  "data": {
    "ready": true,
    "desktop_mode": "browser_only",
    "upgrade_available": true,
    "service_manager": "supervisor",
    "os_name": "Ubuntu Server 24.04 LTS 64bit",
    "elapsed_ms": 980,
    "checks": {
      "packages": "ok",
      "novnc": "ok",
      "chrome": "ok",
      "systemd_config": "ok_supervisor",
      "services": "ok",
      "ports": "ok",
      "cdp_owner": "ok",
      "ssl_cert": "not_configured",
      "fcitx5": "missing",
      "cjk_fonts": "missing",
      "novnc_patch": "missing"
    }
  }
}
```

- **成功响应（未安装）：**

```json
{
  "ok": true,
  "data": {
    "ready": false,
    "desktop_mode": "none",
    "upgrade_available": true,
    "service_manager": "unknown",
    "elapsed_ms": 320,
    "checks": {
      "packages": "missing",
      "novnc": "missing",
      "chrome": "missing",
      "systemd_config": "missing",
      "services": "not_running",
      "ports": "not_listening",
      "cdp_owner": "no_listener",
      "ssl_cert": "missing",
      "fcitx5": "missing",
      "cjk_fonts": "missing",
      "novnc_patch": "missing"
    },
    "missing": ["package:tigervnc-standalone-server", "package:openbox", "noVNC", "websockify", "google-chrome-stable", "no-service-manager", "service:no-manager", "port:5900", "port:6080", "port:9222", "ssl:cert-missing"]
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ready | bool | 环境是否完全就绪（浏览器功能可用）。旧版 supervisor 用户只要浏览器正常也返回 `true` |
| desktop_mode | string | 安装模式：`"full"`=完整云桌面（新版），`"browser_only"`=仅浏览器（旧版），`"none"`=未安装 |
| upgrade_available | bool | 是否可升级到云桌面。`true` 时前端可展示"一键升级"按钮 |
| service_manager | string | 服务管理器类型：`"systemd"`（新版）、`"supervisor"`（旧版）、`"unknown"`（未安装） |
| os_name | string | CVM 操作系统名称（通过 DescribeInstances API 获取，与 TAT 并行执行） |
| elapsed_ms | int | 检查脚本执行耗时（毫秒） |
| checks | object | 各组件检查结果详情 |
| missing | string[] | 未就绪时返回缺失组件列表（`ready=false` 时） |

**checks 字段详细说明：**

| 字段 | 可能的值 | 说明 |
|------|---------|------|
| packages | `"ok"` / `"missing"` | 核心系统包（tigervnc、openbox、dbus 等） |
| novnc | `"ok"` / `"missing"` | noVNC + websockify 安装状态 |
| chrome | `"ok"` / `"missing"` | Google Chrome 安装状态 |
| systemd_config | `"ok"` / `"ok_supervisor"` / `"missing"` | 服务配置：systemd unit 文件或 supervisor 配置 |
| services | `"ok"` / `"not_running"` | 服务进程运行状态 |
| ports | `"ok"` / `"not_listening"` | 端口监听状态（5900/6080/9222） |
| cdp_owner | `"ok"` / `"conflict"` / `"no_listener"` | CDP 端口归属检查（防止 headless/playwright 冲突） |
| ssl_cert | `"ok"` / `"expired"` / `"missing"` / `"not_configured"` | SSL 证书状态（旧版无证书为 `not_configured`，不影响 ready） |
| fcitx5 | `"ok"` / `"installed_not_running"` / `"missing"` | 中文输入法（信息性，不影响 ready） |
| cjk_fonts | `"ok"` / `"missing"` | CJK 字体（信息性，不影响 ready） |
| novnc_patch | `"ok"` / `"missing"` | noVNC 剪贴板/中文输入补丁（信息性，不影响 ready） |

- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `400 {"error": "该实例无关联的 CVM"}`
  - `403 {"error": "云端浏览器功能未开启，请联系管理员在后台开启"}`
  - `404 {"error": "实例不存在"}`
  - `500 {"error": "检查 VNC 环境失败: ..."}`

### 11.2 安装 VNC 环境

### `POST /openclaw/browser-vnc-install`

在指定实例上安装 VNC 云端浏览器环境（完整云桌面模式）。通过 TAT 执行安装脚本，安装 Xvnc、noVNC、websockify、Google Chrome、XFCE4 桌面、fcitx5 中文输入法等组件，并配置 systemd 服务自启动。

对于旧版 supervisor 用户，此接口会自动完成升级：停止旧 supervisor 服务 → 清理旧配置 → 安装桌面组件 → 创建 systemd unit → 启动新服务。

- **权限：** 登录用户（仅可操作自己的实例）
- **审计：** ✅ `browser_vnc_install` / `instance`
- **超时：** TAT 脚本执行超时 600 秒（10 分钟）
- **注意：** CLB 空闲连接超时需 ≥ 900 秒，否则长时间安装会触发 504
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID（Query 参数或 Form 参数） |

- **成功响应：**

```json
{
  "ok": true,
  "data": {
    "installed": true
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| installed | bool | 是否安装成功 |
| error | string | 安装失败时的错误信息（`installed=false` 时） |

- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `400 {"error": "该实例无关联的 CVM"}`
  - `403 {"error": "云端浏览器功能未开启，请联系管理员在后台开启"}`
  - `404 {"error": "实例不存在"}`
  - `409 {"error": "该实例正在安装中，请勿重复操作"}`
  - `500 {"error": "安装 VNC 环境失败: ..."}`

### 11.3 获取 VNC 连接信息

### `GET /openclaw/browser-vnc-access`

获取指定实例的 VNC 连接信息，包括公网 IP、端口、noVNC URL 和 WebSocket 代理路径。

- **权限：** 登录用户（仅可查看自己的实例）
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID（Query 参数） |

- **成功响应（端口已放通）：**

```json
{
  "ok": true,
  "data": {
    "host": "42.194.245.131",
    "port": 5900,
    "ws_port": 6080,
    "accessible": true,
    "vnc_url": "vnc://42.194.245.131:5900",
    "novnc_url": "https://42.194.245.131:6080/vnc.html?autoconnect=true&resize=scale&reconnect=true&reconnect_delay=3000",
    "ws_proxy_path": "/openclaw/vnc-ws-proxy?id=14"
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| host | string | CVM 公网 IP |
| port | int | VNC 端口（5900，绑定 localhost，仅供内部使用） |
| ws_port | int | websockify 端口（6080） |
| accessible | bool | 安全组是否已放通 6080 端口 |
| vnc_url | string | VNC 客户端直连 URL（`accessible=true` 时） |
| novnc_url | string | noVNC Web 直连 URL（v1.x 回退模式，需信任自签名证书） |
| ws_proxy_path | string | **推荐** WebSocket 代理路径（v2.0 模式），前端拼接 `wss://当前域名/api{ws_proxy_path}` |
| message | string | 端口未放通时的错误描述（`accessible=false` 时） |

- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `400 {"error": "该实例无关联的 CVM"}`
  - `403 {"error": "云端浏览器功能未开启，请联系管理员在后台开启"}`
  - `404 {"error": "实例不存在"}`
  - `404 {"error": "未找到 CVM 实例"}`
  - `409 {"error": "实例当前状态为 STOPPED，云端浏览器仅在实例运行中（RUNNING）时可用"}`
  - `500 {"error": "实例无公网 IP，无法使用云端浏览器"}`

### 11.4 查询 AI 状态

### `GET /openclaw/browser-status`

查询指定实例的 AI 任务状态和用户接管状态。前端每 3 秒轮询此接口，用于实时显示 AI 是否正在操作浏览器。接口极轻量（仅内存读取 + 一次 DB 查询校验功能开关）。

- **权限：** 登录用户（仅可查看自己的实例）
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID（Query 参数） |

- **成功响应：**

```json
{
  "ok": true,
  "data": {
    "ai_active": true,
    "takeover": false
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ai_active | bool | AI 是否正在活跃（LLM 请求进行中 或 宽限期内 或 超时兜底内） |
| takeover | bool | 用户是否已接管浏览器控制权 |

**状态组合含义：**

| ai_active | takeover | 含义 |
|:---------:|:--------:|------|
| true | false | AI 正在操作浏览器，用户只能观看 |
| false | false | 空闲状态，用户可以点击"进入操作"接管 |
| false | true | 用户已接管浏览器，可以直接操作 |
| true | true | AI 重新开始操作，自动退出用户接管 |

- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `403 {"error": "云端浏览器功能未开启，请联系管理员在后台开启"}`
  - `404 {"error": "实例不存在"}`

### 11.5 接管浏览器控制

### `POST /openclaw/browser-takeover`

用户手动接管或释放浏览器控制权。接管后用户可以直接在 noVNC 画面中操作浏览器（鼠标、键盘），AI 的浏览器操作会被暂停。

- **权限：** 登录用户（仅可操作自己的实例）
- **审计：** ✅ `browser_takeover` / `instance`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID（Query 参数或 Form 参数） |
| action | string | 是 | 操作类型：`start`（接管）或 `stop`（释放） |

- **成功响应：**

```json
{
  "ok": true,
  "data": {
    "takeover": true
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| takeover | bool | 当前接管状态（`start` → `true`，`stop` → `false`） |

- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `400 {"error": "action 参数无效，可选值: start, stop"}`
  - `403 {"error": "云端浏览器功能未开启，请联系管理员在后台开启"}`
  - `404 {"error": "实例不存在"}`

### 11.6 WebSocket 代理

### `GET /openclaw/vnc-ws-proxy`

WebSocket 反向代理，将前端的 WebSocket 连接代理到 CVM 上的 websockify（6080 端口）。采用**双段独立握手**机制：Hatchery 先与 CVM websockify 完成 WebSocket 握手，再自己计算 `Sec-WebSocket-Accept` 返回给浏览器，完全绕过 Nginx/Go 头名称规范化问题。

- **权限：** 登录用户（通过 Cookie 认证）
- **协议：** WebSocket（`Upgrade: websocket`）
- **子协议：** `binary`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID（Query 参数） |

- **连接流程：**

```
浏览器 ──WSS──→ Nginx ──WS──→ Hatchery ──TLS──→ CVM:6080 (websockify)
                                  │
                          双段独立握手：
                          1. Hatchery ↔ CVM 完成 WS 握手
                          2. Hatchery 自己计算 Accept 返回给浏览器
                          3. 双向透传 VNC 数据帧
```

- **成功：** 返回 `101 Switching Protocols`，建立 WebSocket 双向数据通道
- **限制：** 每实例最大 3 个并发 WebSocket 连接
- **失败响应：**
  - `400 {"error": "缺少参数 id"}`
  - `400 {"error": "非 WebSocket 升级请求"}`
  - `403 {"error": "云端浏览器功能未开启，请联系管理员在后台开启"}`
  - `404 {"error": "实例不存在"}`
  - `409 {"error": "实例当前状态为 STOPPED，云端浏览器仅在实例运行中（RUNNING）时可用"}`
  - `429 {"error": "该实例 VNC 代理连接数已达上限(3)"}`
  - `500 {"error": "实例无公网 IP"}`
  - `502 {"error": "连接 CVM VNC 服务失败"}`

**前端使用示例：**

```typescript
import RFB from "@novnc/novnc/lib/rfb";

const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
const wsUrl = `${protocol}//${window.location.host}/api/openclaw/vnc-ws-proxy?id=${instanceId}`;

const rfb = new RFB(containerElement, wsUrl, {
  wsProtocols: ["binary"],
});
rfb.scaleViewport = true;
rfb.resizeSession = true;
rfb.viewOnly = true; // 查看模式，接管后设为 false
```

### 11.7 管理员配置

云端浏览器功能开关通过 `POST /admin/config` 接口控制，参数 `browser_vnc_enable`（`"true"` / `"false"`）。开启时自动在安全组中放通 6080 端口。

详见 [四、管理后台](#四管理后台) 中的 `POST /admin/config` 接口。

`GET /site` 响应中已登录时包含 `browser_vnc_enable` 字段，前端据此决定是否显示云端浏览器入口。

---

## 十一、标签管理（Tag）

标签分两类能力：

- 默认标签管理：配置创建实例时自动注入的腾讯云 CVM 标签，支持全局范围和分组范围。
- 腾讯云标签查询：`/api/tags/keys` 与 `/api/tags/values` 用于管理界面的标签键/值下拉选项。凭证复用系统统一的 STS 临时凭证 / 永久 AKSK（与 CVM 接口共享）。

默认标签按 `key/value` 管理，每条标签可设置应用范围：

- `visibility_type = "all"`：全局标签，所有新建实例都会注入。
- `visibility_type = "group"`：分组标签，仅实例创建时选择的分组命中该标签范围时注入。

分组标签范围与模型应用范围一致：创建实例时按实例选择的分组判断，父组配置对子组实例生效。

`GET /admin/config` 与 `POST /admin/config` 保持兼容，只管理全局默认标签。分组范围标签请使用本章的标签管理接口。

### `GET /admin/tags`

查询默认标签列表。

- **权限：** 管理员
- **参数：** 无

- **JSON 响应：**

```json
{
  "tags": [
    {
      "id": 1,
      "key": "env",
      "value": "prod",
      "visibility_type": "all",
      "group_ids": [],
      "created_at": "2026-06-04T10:00:00+08:00",
      "updated_at": "2026-06-04T10:00:00+08:00"
    },
    {
      "id": 2,
      "key": "team",
      "value": "rd",
      "visibility_type": "group",
      "group_ids": [10, 11],
      "created_at": "2026-06-04T10:00:00+08:00",
      "updated_at": "2026-06-04T10:00:00+08:00"
    }
  ]
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| tags | object[] | 标签列表 |
| tags[].id | int | 标签 ID |
| tags[].key | string | 标签键 |
| tags[].value | string | 标签值 |
| tags[].visibility_type | string | 应用范围：`all` / `group` |
| tags[].group_ids | int[] | `visibility_type=group` 时绑定的分组 ID 列表 |
| tags[].created_at | string | 创建时间 |
| tags[].updated_at | string | 更新时间 |

---

### `POST /admin/tags/create`

新增一条默认标签。

- **权限：** 管理员
- **Content-Type：** `application/json`

```json
{
  "key": "team",
  "value": "rd",
  "visibility_type": "group",
  "group_ids": [10, 11]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| key | string | 是 | 标签键 |
| value | string | 否 | 标签值 |
| visibility_type | string | 否 | 应用范围：`all` / `group`，默认 `all` |
| group_ids | int[] | 条件 | `visibility_type=group` 时必填 |

- **JSON 响应：**

```json
{
  "tag": {
    "id": 2,
    "key": "team",
    "value": "rd",
    "visibility_type": "group",
    "group_ids": [10, 11],
    "created_at": "2026-06-04T10:00:00+08:00",
    "updated_at": "2026-06-04T10:00:00+08:00"
  }
}
```

---

### `POST /admin/tags/update`

修改一条默认标签。

- **权限：** 管理员
- **Content-Type：** `application/json`
- **Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 标签 ID |

- **请求体：**

```json
{
  "key": "team",
  "value": "rd-platform",
  "visibility_type": "group",
  "group_ids": [10, 11]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| key | string | 是 | 标签键 |
| value | string | 否 | 标签值 |
| visibility_type | string | 否 | 应用范围：`all` / `group`，默认 `all` |
| group_ids | int[] | 条件 | `visibility_type=group` 时必填；`visibility_type=all` 时会被忽略 |

- **JSON 响应：**

```json
{
  "tag": {
    "id": 2,
    "key": "team",
    "value": "rd-platform",
    "visibility_type": "group",
    "group_ids": [10, 11],
    "created_at": "2026-06-04T10:00:00+08:00",
    "updated_at": "2026-06-04T10:05:00+08:00"
  }
}
```

---

### `POST /admin/tags/replace-all`

全量覆盖默认标签列表。适用于前端弹窗中编辑完整列表后一次性保存。

- **权限：** 管理员
- **Content-Type：** `application/json`

- **请求体：**

```json
{
  "tags": [
    {
      "key": "env",
      "value": "prod",
      "visibility_type": "all",
      "group_ids": []
    },
    {
      "key": "team",
      "value": "rd",
      "visibility_type": "group",
      "group_ids": [10, 11]
    }
  ]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| tags | object[] | 是 | 完整标签列表，整包覆盖语义；空数组 `[]` 表示清空所有标签 |
| tags[].key | string | 是 | 标签键 |
| tags[].value | string | 否 | 标签值 |
| tags[].visibility_type | string | 否 | 应用范围：`all` / `group`，默认 `all` |
| tags[].group_ids | int[] | 条件 | `visibility_type=group` 时必填；`visibility_type=all` 时会被忽略 |

- **JSON 响应：**

```json
{
  "ok": true
}
```

---

### `POST /admin/tags/delete`

删除一条默认标签。

- **权限：** 管理员
- **Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 标签 ID |

- **JSON 响应：**

```json
{
  "ok": true
}
```

---

### `GET /api/tags/keys`

查询当前账号下所有标签键列表。

- **权限：** 管理员
- **参数：** 无

- **JSON 响应：**

```json
{
  "keys": ["env", "team", "所属产品", "负责人"]
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| keys | string[] | 标签键列表 |

---

### `GET /api/tags/values`

查询指定标签键的所有值列表。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| key | string | 是 | 标签键 |

- **JSON 响应：**

```json
{
  "key": "env",
  "values": ["production", "staging", "dev"]
}
```

| 响应字段 | 类型 | 说明 |
|----------|------|------|
| key | string | 查询的标签键 |
| values | string[] | 该键下所有值列表 |

---

## 企业 MCP 库

管理员可通过以下接口管理企业级 MCP（Model Context Protocol）服务器配置，并将其批量下发到 OpenClaw 实例。

### `GET /admin/mcp` — MCP 列表

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| q | string | 否 | 搜索关键词（匹配 name / description / service_id） |
| transport | string | 否 | 按连接方式筛选（sse / streamable-http / stdio） |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20，最大 500 |

- **JSON 响应：**

```json
{
  "total": 10,
  "page": 1,
  "page_size": 20,
  "items": [
    {
      "id": 1,
      "service_id": "github-copilot",
      "name": "GitHub Copilot",
      "description": "...",
      "transport_type": "sse",
      "latest_version": "1.2.0",
      "distribution_summary": { "total": 50, "success": 48, "failed": 1, "pending": 1 },
      "created_at": "2026-04-21 10:00:00",
      "updated_at": "2026-04-21 12:00:00"
    }
  ]
}
```

### `POST /admin/mcp/create` — 新增 MCP

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| service_id | string | 是 | 服务唯一标识（英文、数字、中划线、下划线，≤48 字符） |
| name | string | 否 | 显示名称（默认 = service_id） |
| description | string | 否 | 描述 |
| transport_type | string | 是 | 连接方式（sse / streamable-http / stdio） |
| config_json | string | 是 | 服务配置 JSON 字符串 |
| usage_doc_md | string | 否 | 使用说明 Markdown |
| tool_doc_md | string | 否 | 工具说明 Markdown |
| key_hosted | bool | 否 | 是否开启密钥托管（默认 false）。开启时 config_json 的 headers 和 url query 中的占位符（如 `<token>`）将由安全网关代理注入 |
| hosted_defaults | map[string]string | 否 | 托管字段的管理员默认值。key 为占位符名（去掉 `<>`），value 为默认值。有默认值的字段用户无需填写 |
| ip_whitelist | string | 否 | IP 白名单（逗号分隔的 IP/CIDR），仅 `key_hosted=true` 时生效。空字符串=不限制。示例：`"10.0.0.1,192.168.1.0/24"` |

- **成功响应（201）：**

```json
{
  "id": 1,
  "service_id": "github-copilot",
  "latest_version": "1.0.0",
  "version_id": 1,
  "warnings": []
}
```

- **失败响应：**
  - `400 {"error": "请输入服务ID"}` / `{"error": "服务ID 仅支持英文、数字、中划线、下划线"}`
  - `400 {"error": "开启密钥托管时，config_json 中至少需要一个占位符字段（如 <your-token>）"}`
  - `409 {"error": "该服务ID 已存在，请使用其他标识"}`

### `POST /admin/mcp/meta` — 修改元数据

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| service_id | string | 是 | 服务唯一标识 |
| name | string | 否 | 新名称（null 表示不修改） |
| description | string | 否 | 新描述（null 表示不修改） |
| ip_whitelist | string | 否 | IP 白名单（null 表示不修改，空字符串 `""` = 取消限制） |

- **成功响应：** `{"id": 1, "service_id": "...", "name": "...", "description": "...", "ip_whitelist": "...", "updated_at": "..."}`
- **失败响应：**
  - `404 {"error": "MCP 不存在"}`

### `GET /admin/mcp/detail` — MCP 详情

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| service_id | string | 是 | 服务唯一标识 |
| version | string | 否 | 指定版本号（默认返回最新版本，传 "latest" 等效） |

- **JSON 响应：**

```json
{
  "id": 1,
  "service_id": "github-copilot",
  "name": "GitHub Copilot",
  "description": "...",
  "created_at": "...",
  "created_by": "admin",
  "key_hosted": true,
  "ip_whitelist": "10.0.0.1,192.168.1.0/24",
  "current_version": {
    "version": "1.0.0",
    "transport_type": "sse",
    "config_json": "{\"url\":\"http://...\"}",
    "usage_doc_md": "...",
    "tool_doc_md": "...",
    "created_at": "...",
    "created_by": "admin"
  },
  "latest_version": "1.2.0"
}
```

### `POST /admin/mcp/delete` — 删除 MCP

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| service_id | string | 是 | 服务唯一标识 |

> 硬删除（server + versions + installations 全部清除）。有正在运行的下发任务或安装中的实例时返回 409。

- **成功响应：** `{"message": "删除成功"}`
- **失败响应：**
  - `404 {"error": "MCP 不存在"}`
  - `409 {"error": "有下发任务正在进行中，请等待完成后再删除"}`
  - `409 {"error": "有实例正在安装中，请等待完成后再删除"}`

### `POST /admin/mcp/update` — 新增版本

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| service_id | string | 是 | 服务唯一标识 |
| version | string | 否 | 版本号（格式 x.y.z）。传入时必须大于当前最高版本，否则返回 400；不传时后端自动递增（PATCH +1） |
| transport_type | string | 是 | 连接方式 |
| config_json | string | 是 | 服务配置 JSON 字符串 |
| usage_doc_md | string | 否 | 使用说明 Markdown |
| tool_doc_md | string | 否 | 工具说明 Markdown |
| hosted_defaults | map[string]string | 否 | 托管字段默认值（仅 `key_hosted=true` 时有效）。key 必须是 config_json 中存在的占位符名 |

> 版本号规则：前端传入 version 时使用指定版本号，必须大于当前最高版本（否则返回 400），已存在返回 409；不传时后端自动在最高版本基础上 PATCH +1（例如 1.0.0 → 1.0.1，1.2.3 → 1.2.4），同时更新 server 的 latest_version_id 和 transport_type。
>
> **密钥托管：** 如果该 MCP 已开启 `key_hosted`，新版本的 config_json 必须包含占位符，后端会自动同步 `mcp_hosted_keys` 表。`hosted_defaults` 中的 key 必须是 config_json 中有效的占位符名，否则返回 400。

- **成功响应（201）：** `{"version": "1.0.1", "warnings": []}`
- **失败响应：**
  - `400 {"error": "版本号格式不合法，需要 x.y.z 格式"}`
  - `400 {"error": "版本号必须大于当前最高版本 x.y.z"}`
  - `400 {"error": "该 MCP 已开启密钥托管，config_json 中必须包含占位符"}`
  - `400 {"error": "hosted_defaults 中的 \"xxx\" 不是 config_json 中的占位符"}`
  - `404 {"error": "MCP 不存在"}`
  - `409 {"error": "版本 2.0.0 已存在"}`

### `GET /admin/mcp/versions` — 版本列表

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| service_id | string | 是 | 服务唯一标识 |

- **JSON 响应：**

```json
{
  "versions": [
    { "version": "1.1.0", "transport_type": "streamable-http", "created_at": "...", "created_by": "admin", "is_latest": true },
    { "version": "1.0.0", "transport_type": "sse", "created_at": "...", "created_by": "admin", "is_latest": false }
  ]
}
```

### `POST /admin/mcp/distribute` — 批量下发

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| service_id | string | 是 | 服务唯一标识 |
| version | string | 是 | 版本号（如 v1） |
| instance_ids | uint[] | 条件必填 | 显式目标实例 ID（最多 500）；与 `select_all=true` 严格二选一 |
| select_all | bool | 条件必填 | 传 `true` 时选择全部匹配实例；全选模式不受 500 个限制 |
| statuses | string[] | 否 | 仅全选模式。可选 `uninstalled/installed/outdated/failed`，按目标 `version` 计算；空数组/省略表示以上稳定状态全集；禁止 `installing` |
| group_ids | uint[] | 否 | 仅全选模式。多个组取并集；`0` 表示未分组用户 |
| search | string | 否 | 仅全选模式。模糊匹配实例名称、实例 ID 或创建人用户名；最长 50 个字符；省略表示不限制 |

> 异步执行。全选目标在请求受理时固化为 task records，不新增 running 状态限制，但仍过滤不支持 MCP 的 Agent。使用分布式锁防止同一 MCP+版本并发下发。

- **成功响应（202）：**

```json
{
  "task_id": 1,
  "total": 10,
  "per_instance": [
    { "instance_id": 1, "record_id": 1, "status": "pending" }
  ],
  "warnings": [
    { "instance_id": 99, "reason": "not_found", "detail": "instance not found" }
  ]
}
```

全选模式仅返回汇总，不展开逐实例明细：

```json
{
  "task_id": 2,
  "total": 1200
}
```

- **失败响应：**
  - `404 {"error": "MCP 不存在"}` / `{"error": "版本 v1 不存在"}`
  - `400 {"error": "instance_ids 与 select_all=true 必须二选一"}`
  - `400 {"error": "不能按过渡状态 installing 全选下发"}`
  - `400 {"error": "没有符合条件的实例可以下发"}`
  - `409 {"error": "有下发任务正在进行中，请稍后重试"}`

### `GET /admin/mcp/tasks` — 下发任务列表

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| service_id | string | 是 | 服务唯一标识 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20，最大 100 |

> 每个 task 内嵌 `records` 数组，包含该任务下所有实例的执行结果。与技能的 `tasks` 接口结构一致。

- **JSON 响应：**

```json
{
  "tasks": [
    {
      "id": 1,
      "created_at": "...",
      "operator": "admin",
      "version": "1.0.0",
      "total": 50,
      "success": 48,
      "failed": 1,
      "pending": 1,
      "status": "running",
      "records": [
        { "instance_id": 1, "cvm_instance_id": "ins-abc123", "instance_name": "my-instance", "username": "john", "status": "success", "error": "" }
      ]
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

### `GET /admin/mcp/instances` — 实例安装情况

查询实例安装情况。查询所有实例对该 MCP 的安装状态，仅返回实例语义状态为 `running` 的实例（通过实时 CVM API 判定），支持按状态、实例类型等条件筛选。当实例已安装的 MCP 版本低于最新版本时，状态为 `outdated`（待更新）。

- **权限：** 管理员
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| service_id | string | 是 | MCP 服务唯一标识 |
| status | string | 否 | 按安装状态筛选：`installed`（已安装且为最新版）、`outdated`（已安装但有新版本可更新）、`uninstalled`（未安装）、`installing`（安装中）、`failed`（安装失败）。支持逗号分隔多状态，如 `uninstalled,failed`。不传返回全部 |
| search | string | 否 | 按实例名称或 CVM 实例 ID 模糊搜索 |
| instance_type | string | 否 | 按实例类型筛选，如 `openclaw`、`hermes`、`lightclawace`。支持逗号分隔多类型，如 `openclaw,hermes`。不传返回全部 |
| group_id | string | 否 | 按用户组筛选实例，只显示该分组内用户的实例。支持逗号分隔多个 group_id（取并集）。传 `0` 表示未分组用户的实例，可与正常 group_id 组合使用，如 `group_id=0,1,3` 表示未分组 + 分组 1 或 3 的用户实例 |
| agent_version_min | string | 否 | 按实例版本筛选：大于等于指定版本（闭区间下界），格式 `YYYY.M.D`。精确匹配可令 min=max |
| agent_version_max | string | 否 | 按实例版本筛选：小于等于指定版本（闭区间上界），格式 `YYYY.M.D` |
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **成功响应：**

```json
{
  "instances": [
    {
      "instance_id": 1,
      "cvm_instance_id": "ins-abc123",
      "instance_name": "my-instance",
      "instance_type": "openclaw",
      "user_id": 1,
      "username": "user1",
      "last_cvm_state": "RUNNING",
      "agent_version": "2026.4.11",
      "status": "installed",
      "version": "1.0.0",
      "latest_version": "1.0.0",
      "user_groups": [
        {"group_id": 1, "group_name": "研发组"}
      ],
      "instance_status": "running",
      "instance_status_label": "运行中",
      "transient": false
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

- **响应字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| instance_id | uint | 实例 DB ID |
| cvm_instance_id | string | CVM 实例 ID |
| instance_name | string | 实例名称 |
| instance_type | string | 实例类型（如 `openclaw`） |
| user_id | uint | 用户 ID |
| username | string | 用户名 |
| last_cvm_state | string | CVM 最后记录的状态（如 `RUNNING`） |
| agent_version | string | 实例版本号，格式 `YYYY.M.D`（如 `2026.4.11`），未上报时为空 |
| status | string | MCP 安装状态：`installed`/`outdated`/`uninstalled`/`installing`/`failed` |
| version | string | 已安装的 MCP 版本号，未安装时为空 |
| latest_version | string | 该 MCP 的最新版本号 |
| user_groups | array | 用户所属分组列表，含 `group_id` 和 `group_name` |
| instance_status | string | 实例语义状态（当前固定为 `running`） |
| instance_status_label | string | 实例状态中文标签（如运行中） |
| transient | bool | 是否过渡态（当前固定为 `false`） |

- **`status` 枚举值（MCP 安装状态）：**

| 枚举值 | 说明 | 判断逻辑 |
|--------|------|---------|
| installed | 已安装（最新版） | install_status=2（成功）且版本 = 最新版本 |
| outdated | 待更新 | install_status=2（成功）但版本 ≠ 最新版本 |
| installing | 安装中 | install_status=1 |
| failed | 安装失败 | install_status=3（失败）或 4（已取消） |
| uninstalled | 未安装 | 无安装记录或 install_status=0 |

- **失败响应：**
  - `400 {"error": "service_id 不能为空"}`
  - `404 {"error": "MCP 不存在"}`

---

## 龙虾医生（Dragon Doctor）

### 诊断会话自动结束策略

后台每 5 分钟扫描一次 `active` 诊断会话：

- 用户尚未开始对话（远端没有 session 文件）时，从会话进入 `active` 的时间开始计算；超过 12 小时后自动转为 `ending`。升级前创建、没有独立激活时间的存量会话回退使用创建时间。
- 已产生对话文件时，以最新 session 文件的修改时间计算空闲时长；连续 12 小时没有新活动后自动转为 `ending`。
- STS 临时凭证的周期刷新属于节点维护行为，不视为用户活动，也不会重置上述 12 小时窗口。
- `ending` 会话由每 1 分钟运行一次的清理任务销毁临时诊断节点并最终转为 `ended`。

### POST /openclaw/doctor/quick-fix

在目标实例上异步下发 `openclaw doctor --fix --yes` 命令，立即返回 `invocation_id`。
前端通过 `GET /openclaw/doctor/quick-fix/status` 轮询执行结果。

- **请求参数（query）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 DB ID |

- **成功响应：**
  - `200 {"ok": true, "invocation_id": "inv-xxx"}`

- **失败响应：**
  - `200 {"ok": false, "error": "fix_failed", "message": "...", "output": "..."}`
  - `400 {"message": "..."}`
  - `401` 未登录

---

### GET /openclaw/doctor/quick-fix/status

查询一键修复命令的执行状态，前端轮询使用。

- **请求参数（query）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 DB ID |
| invocation_id | string | 是 | 由 POST /openclaw/doctor/quick-fix 返回的执行 ID |

- **成功响应：**
  - `200 {"ok": true, "status": "RUNNING", "output": "...", "exit_code": 0, "finished": false}`
  - `status` 枚举值：`PENDING` | `DELIVERING` | `RUNNING` | `SUCCESS` | `FAILED` | `TIMEOUT` | `DELIVER_FAILED` | `START_FAILED`
  - `finished` 为 `true` 时表示命令已执行结束，前端可停止轮询

- **失败响应：**
  - `200 {"ok": false, "error": "query_failed", "message": "..."}`
  - `400 {"message": "缺少参数 invocation_id"}`
  - `401` 未登录

---

### GET /openclaw/doctor/feature

查询龙虾医生功能状态和用户授权情况。

- **请求参数（query）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 DB ID |

- **成功响应：**
  - `200 {"ok": true, "doctor_enabled": true, "authorized": false}`

---

### POST /openclaw/doctor/authorize

用户首次使用龙虾医生时确认授权（幂等）。

- **请求参数（query）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 DB ID |

- **成功响应：**
  - `200 {"ok": true, "message": "授权成功"}` 或 `{"ok": true, "message": "已授权"}`

---

### POST /openclaw/doctor/start

创建龙虾医生诊断会话，异步创建临时 CVM 节点。可选同时创建配置快照（快照在诊断节点激活后由定时任务异步执行）。

- **请求参数（query）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 目标实例 DB ID |

- **请求体（JSON）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| snapshot | bool | 否 | 是否在激活后自动创建配置快照，默认 false |

- **成功响应：**
  - `200 {"ok": true}`

- **失败响应：**
  - `200 {"ok": false, "error": "doctor_disabled", "message": "龙虾医生功能未开启"}`
  - `200 {"ok": false, "error": "security_group_not_set", "message": "安全组未配置"}`
  - `200 {"ok": false, "error": "active_session_exists", "message": "已有进行中的诊断会话"}`
  - `200 {"ok": false, "error": "unsupported_agent_type", "message": "龙虾医生暂仅支持 OpenClaw 类型实例"}`

---

### GET /openclaw/doctor/status

查询指定实例最新的诊断会话状态。

- **请求参数（query）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 DB ID |

- **成功响应（有诊断记录）：**
  - `200 {"ok": true, "session_id": 1, "status": "active", "has_snapshot": false}`
  - `status` 枚举值：`creating` | `active` | `ending` | `ended` | `failed`

- **成功响应（无诊断记录）：**
  - `200 {"ok": true, "has_active_session": false}`

- **失败响应：**
  - `400 {"message": "缺少参数 id"}`

---

### POST /openclaw/doctor/end

结束诊断会话（异步）。接口立刻将会话标记为 `ending` 状态并返回，实际的回滚和资源清理由后台定时任务（每 1 分钟）完成。前端通过轮询 `GET /openclaw/doctor/status` 等待 `status` 从 `ending` 变为 `ended`。

- **请求参数（query）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 DB ID |

- **请求体（JSON）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| rollback | bool | 否 | 是否回滚到快照，默认 false |

- **成功响应：**
  - `200 {"ok": true, "status": "ending"}`

- **失败响应：**
  - `200 {"ok": false, "error": "session_not_found", "message": "会话不存在"}`
  - `200 {"ok": false, "error": "session_already_ended", "message": "会话已结束"}`

## 一键升级（Plugin Upgrade）

### `GET /admin/memory/plugin-upgrade/candidates` — 查询待升级实例列表

```
GET /admin/memory/plugin-upgrade/candidates
```

- **权限：** 管理员
- **说明：** 实时查询所有 Pro 版 OpenClaw 实例的插件版本和 Offload 状态，返回需要升级（版本低于最低要求或 Offload 未开启）的实例列表。响应时间取决于实例数量（TAT 最大并发 15）。
- **限制：** 仅支持 `agent_type = openclaw` 的实例，Hermes 类型实例不在查询范围内。

**响应**：
```json
{
  "min_version": "0.3.3",
  "total": 2,
  "instances": [
    {
      "instance_id": "ins-7m3jsb9q",
      "instance_name": "my-openclaw",
      "creator_name": "admin123",
      "memory_plugin_version": "0.3.0",
      "offload_enabled": false
    },
    {
      "instance_id": "ins-0o8p6l1g",
      "instance_name": "test-openclaw",
      "creator_name": "admin123",
      "memory_plugin_version": "0.3.3",
      "offload_enabled": false
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| min_version | string | 系统要求的最低插件版本 |
| total | int | 待升级实例数量 |
| `instances[].instance_id` | string | CVM 实例 ID |
| `instances[].instance_name` | string | 实例名称 |
| `instances[].creator_name` | string | 创建者用户名 |
| `instances[].memory_plugin_version` | string | 当前插件版本（空字符串=未安装） |
| `instances[].offload_enabled` | bool | Offload 是否已开启 |

---

### `POST /admin/memory/plugin-upgrade/execute` — 触发插件升级

```
POST /admin/memory/plugin-upgrade/execute
```

- **权限：** 管理员
- **Content-Type：** `application/json`
- **说明：** 异步触发批量插件升级 + 开启 Offload + 重启 OpenClaw。最大并发 5。
- **限制：** 仅支持 `agent_type = openclaw` 的实例，传入非 OpenClaw 实例会返回 `failed`。

**请求体**：
```json
{
  "instance_ids": ["ins-7m3jsb9q", "ins-0o8p6l1g"]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_ids | string[] | 是 | 待升级的实例 ID 列表（来自 candidates 接口） |

**响应**：
```json
{
  "min_version": "0.3.3",
  "submitted": 2,
  "results": [
    {"instance_id": "ins-7m3jsb9q", "status": "submitted"},
    {"instance_id": "ins-0o8p6l1g", "status": "submitted"}
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| min_version | string | 系统要求的最低插件版本 |
| submitted | int | 成功提交的实例数量 |
| `results[].instance_id` | string | 实例 ID |
| `results[].status` | string | `submitted`=已提交 / `skipped`=无需操作 / `failed`=校验失败 |
| `results[].message` | string | 失败/跳过原因（可选） |

> **说明**：接口立即返回，升级在后台异步执行（并发度 5）。每台实例依次执行：升级插件 → 开启 Offload → 重启 OpenClaw。升级完成后可再次调用 candidates 确认列表是否清空。

## 技能广场（Tenant Skill Marketplace）

用户端技能浏览与下发接口。鉴权方式为 Cookie Session（需已登录），与管控端技能库的核心差异：

- 列表自动按用户可见范围过滤（`visibility_type=all` 全部可见；`visibility_type=group` 时用户所属分组与技能分组有交集即可见）
- 实例查询和下发仅限当前用户的实例
- 不暴露创建/编辑/删除等管理能力

### `GET /openclaw/skillstore` — 技能列表

获取当前用户可见的技能列表（每个 slug 仅返回最新版本）。

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 否 | 模糊搜索（匹配名称或描述） |
| category_ids | string | 否 | 分类 ID，逗号分隔，如 `1,3` |
| sort | string | 否 | 排序方式：`newest`（默认，按发布时间倒序）/ `downloads`（按下发成功数倒序） |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **响应：**

```json
{
  "skills": [
    {
      "id": 1,
      "slug": "aippt-maker",
      "name": "PPT 生成助手",
      "description": "一键生成精美 PPT",
      "version": "1.2.0",
      "distribute_count": 128,
      "created_at": "2025-03-02T10:00:00Z",
      "categories": [
        { "id": 1, "name": "通用办公" }
      ],
      "last_task": {
        "task_id": 10,
        "status": "completed",
        "total": 5,
        "success": 4,
        "failed": 1,
        "version": "1.2.0",
        "created_at": "2025-03-05T14:30:00Z"
      },
      "security_status": "safe"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 42
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `skills[].last_task` | object\|null | 最近一次下发任务信息，`null` 表示从未下发 |
| `skills[].security_status` | string | 安全检测状态：`not_scanned` / `scanning` / `safe` / `suspicious` / `malicious` |
| `last_task.status` | string | `"running"` 下发中 / `"completed"` 已完成 |
| `last_task.total` | int | 本次下发实例总数 |
| `last_task.success` | int | 成功数 |
| `last_task.failed` | int | 失败数 |

**前端状态映射**：
- `last_task == null` → 无状态图标
- `status == "running"` → 蓝色旋转图标，hover 显示进度百分比 `success / total`
- `status == "completed" && failed == 0` → 绿色 ✓
- `status == "completed" && failed > 0` → 红色 ✗

---

### `GET /openclaw/skillstore/detail` — 技能详情

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能标识 |
| version | string | 否 | 指定版本，不传或 `latest` 返回最新版本 |

- **响应：**

```json
{
  "skill": {
    "id": 1,
    "slug": "aippt-maker",
    "name": "PPT 生成助手",
    "description": "一键生成精美 PPT，支持多种模板风格...",
    "version": "1.2.0",
    "changelog": "- 新增 3 种模板\n- 修复导出 bug",
    "created_at": "2025-03-02T10:00:00Z",
    "categories": [
      { "id": 1, "name": "通用办公" }
    ],
    "file_list": [
      "aippt-maker/SKILL.md",
      "aippt-maker/src/index.ts",
      "aippt-maker/src/templates.json"
    ],
    "file_size": 102400,
    "distribute_count": 128
  },
  "versions": [
    { "version": "1.2.0", "created_at": "2025-03-02T10:00:00Z" },
    { "version": "1.1.0", "created_at": "2025-02-15T08:00:00Z" },
    { "version": "1.0.0", "created_at": "2025-01-20T12:00:00Z" }
  ],
  "security_scan": {
    "scan_status": "safe",
    "risk_level": "benign",
    "security_score": 100,
    "scanned_at": "2026-04-02T09:13:23+08:00",
    "report_url": "https://..."
  },
  "smh": {
    "access_token": "xxx-read-only-token-xxx",
    "space_id": "spaceXXX",
    "library_id": "libXXX",
    "endpoint": "https://smh.tencentcs.com"
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `skill.distribute_count` | int | 累计下发成功数 |
| security_scan | object\|null | 安全检测精简版状态，未检测时为 `null` |
| smh | object\|null | SMH 只读访问信息，SMH 未配置时为 `null` |
| `smh.access_token` | string | skillhub 空间只读 Token（有效期 24h） |
| `smh.space_id` | string | SMH 空间 ID |
| `smh.library_id` | string | SMH 媒体库 ID |
| `smh.endpoint` | string | SMH API 端点 |

> **文件访问 URL 拼接**：`{endpoint}/api/v1/file/{library_id}/{space_id}/{file_path}?access_token={access_token}`
> 例如：`https://smh.tencentcs.com/api/v1/file/libXXX/spaceXXX/aippt-maker/SKILL.md?access_token=xxx`

---

### `GET /openclaw/skillstore/categories` — 分类列表

- **参数：** 无

- **响应：**

```json
{
  "categories": [
    { "id": 1, "name": "通用办公" },
    { "id": 2, "name": "研发工具" },
    { "id": 3, "name": "系统运维" },
    { "id": 4, "name": "质量测试" },
    { "id": 5, "name": "需求设计" },
    { "id": 6, "name": "信息检索" },
    { "id": 7, "name": "项目管理" },
    { "id": 8, "name": "数据分析" },
    { "id": 9, "name": "安全合规" },
    { "id": 10, "name": "其他" }
  ]
}
```

---

### `GET /openclaw/skillstore/instances` — 用户实例安装状态

查询当前用户的实例对某技能的安装情况。**仅返回状态为「运行中」的实例**。

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能标识 |
| status | string | 否 | 安装状态筛选，逗号分隔。可选值：`uninstalled` / `installed` / `outdated` / `installing` / `failed` |
| search | string | 否 | 按实例名称或 CVM ID 模糊搜索 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **响应：**

```json
{
  "instances": [
    {
      "instance_id": 1,
      "cvm_instance_id": "ins-abc123",
      "instance_name": "Alice的工作助手",
      "instance_type": "openclaw",
      "status": "uninstalled",
      "version": "",
      "latest_version": "1.2.0",
      "instance_status": "running",
      "instance_status_label": "运行中"
    },
    {
      "instance_id": 3,
      "cvm_instance_id": "ins-def456",
      "instance_name": "Alice的分析助手",
      "instance_type": "openclaw",
      "status": "failed",
      "version": "1.1.0",
      "latest_version": "1.2.0",
      "instance_status": "running",
      "instance_status_label": "运行中"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 2
}
```

| status 值 | 含义 | 下发弹窗中是否可选 |
|-----------|------|-------------------|
| uninstalled | 未下发 | 是 |
| installed | 已安装（版本一致） | 否 |
| outdated | 待更新（已安装旧版本） | 是 |
| installing | 下发中 | 否 |
| failed | 下发失败 | 是 |

---

### `GET /openclaw/skillstore/tasks` — 下发记录

查询某技能对当前用户实例的下发历史。`total`/`success`/`failed` 只统计当前用户的实例。

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能标识 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **响应：**

```json
{
  "tasks": [
    {
      "id": 10,
      "created_at": "2025-03-05T14:30:00Z",
      "operator": "alice",
      "version": "1.2.0",
      "total": 2,
      "success": 1,
      "failed": 1,
      "pending": 0,
      "status": "completed",
      "records": [
        {
          "instance_id": 1,
          "cvm_instance_id": "ins-abc123",
          "instance_name": "Alice的工作助手",
          "status": "success",
          "error": ""
        },
        {
          "instance_id": 3,
          "cvm_instance_id": "ins-def456",
          "instance_name": "Alice的分析助手",
          "status": "failed",
          "error": "timeout: script execution exceeded 120s"
        }
      ]
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 3
}
```

---

### `POST /openclaw/skillstore/distribute` — 下发技能

将技能部署到用户自己选择的实例。下发为异步操作，接口立即返回 `task_id`。

- **请求体（JSON）：**

```json
{
  "slug": "aippt-maker",
  "version": "1.2.0",
  "instance_ids": [1, 3]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能标识 |
| version | string | 否 | 版本号，不传或 `"latest"` 使用最新版本 |
| instance_ids | number[] | 是 | 实例 ID 列表（必须为当前用户的实例） |

- **响应（成功）：**

```json
{
  "ok": true,
  "task_id": 123,
  "version": "1.2.0"
}
```

- **错误场景：**

| HTTP 状态码 | error | 场景 |
|-------------|-------|------|
| 400 | `"slug 不能为空"` | 参数缺失 |
| 400 | `"instance_ids 不能为空"` | 未选择实例 |
| 400 | `"技能不存在"` | slug 不存在 |
| 400 | `"版本不存在（slug=x, version=y）"` | 指定版本不存在 |
| 403 | `"包含非本人实例"` | instance_ids 包含非当前用户的实例 |
| 409 | `"该技能版本正在被其他操作处理，请稍后重试"` | 同版本正在下发中 |

> **说明**：下发后可通过轮询 `GET /openclaw/skillstore/tasks` 或 `GET /openclaw/skillstore` 的 `last_task` 字段获取进度，建议 5-10 秒轮询一次直到 `status != "running"`。
>
> **🆕** 本接口同时支持 CVM 与本地 agent 实例，请求/响应字段不变。

---

### `POST /openclaw/skillstore/uninstall` — 卸载技能

从当前用户自己的实例上移除已安装的技能。异步执行。

- **请求体：**

```json
{
  "slug": "my-skill",
  "instance_ids": [1, 2, 3]
}
```

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能唯一标识 |
| instance_ids | int[] | 是 | 目标实例 ID 列表（必须属于当前登录用户） |

- **成功响应：**

```json
{"ok": true, "task_id": 1}
```

- **失败响应：**
  - `400` — slug 或 instance_ids 为空
  - `403` — instance_ids 包含非当前用户的实例
  - `404` — 技能不存在或不可见
  - `409` — 该技能正在被其他操作处理

> **说明**：卸载后可通过轮询 `GET /openclaw/skillstore/tasks?type=uninstall` 获取进度。卸载成功后实例恢复为"未下发"状态，可再次下发。

---

### `GET /openclaw/skillstore/download` — 下载技能 zip 包

302 跳转到 SMH 下载地址，同时递增下载计数。

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能标识 |
| version | string | 否 | 版本号，不传或 `latest` 使用最新版本 |

- **响应：** HTTP 302 重定向到 SMH 带签名的 zip 下载地址。

- **错误场景：**

| HTTP 状态码 | error | 场景 |
|-------------|-------|------|
| 400 | `"缺少 slug 参数"` | 参数缺失 |
| 404 | `"技能不存在"` | slug 不存在或对当前用户不可见 |
| 404 | `"版本不存在（slug=x, version=y）"` | 指定版本不存在 |

---

### `GET /admin/skills/download` — 下载技能 zip 包（管控端）

管控端版本，需要 admin 权限，无可见性限制。参数和行为与 `/openclaw/skillstore/download` 一致。

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能标识 |
| version | string | 否 | 版本号，不传或 `latest` 使用最新版本 |

- **响应：** HTTP 302 重定向到 SMH 带签名的 zip 下载地址。

---

### 前端集成注意事项

1. **下发弹窗默认筛选**：调用 instances 接口时建议默认传 `status=uninstalled,failed`（未下发 + 下发失败），与管控端仅默认「未下发」不同
2. **可选实例**：接口已只返回 `instance_status=running` 的实例，前端只需控制 `status` 为 `uninstalled`/`failed`/`outdated` 的才可勾选
3. **分类筛选**：支持单选或多选（逗号分隔 ID），与管控端一致
4. **排序**：`sort=newest`（默认）按发布时间倒序，`sort=downloads` 按下发成功数倒序

---

## 技能共建审核

员工可提交技能和新版本，由管理员审核通过后自动上架。员工也可申请下架自己上传的技能，管理员审核通过后自动下架。

**互斥规则**：同一 slug 同时只能存在一个 `status=pending` 的申请。

---

### `POST /openclaw/skills/contribute` — 提交技能/新版本

员工提交技能（含 zip 包），创建待审核的 Skill 记录（`status=pending_review`）和审核申请单。参数与 `POST /admin/skills/create` 一致。

- **权限**：已登录用户
- **Content-Type：** `multipart/form-data`

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能标识（3-50 字符，小写字母+数字+连字符） |
| name | string | 是 | 技能名称 |
| version | string | 是 | 版本号（x.y.z 格式，须大于同 slug 现有最高版本） |
| description | string | 否 | 技能描述 |
| changelog | string | 否 | 版本更新说明 |
| file | file | 是 | 技能 zip 包（最大 100MB，须含 SKILL.md） |
| category_ids | string | 否 | 分类 ID 列表（逗号分隔） |
| visibility_type | string | 否 | 可见范围：`all`（默认）或 `group` |
| group_ids | string | 否 | 可见分组 ID 列表（逗号分隔，`visibility_type=group` 时有效） |
| project_ids | string | 否 | 可见项目 ID 列表（逗号分隔） |
| submit_scan | string | 否 | 是否触发安全检测：`true` 触发，其他值或不传不触发。文件超过 7MB 自动跳过 |

- **响应：** `200`

```json
{
  "ok": true,
  "skill_id": 42,
  "request_id": 7,
  "scan_submitted": false,
  "scan_skip_reason": ""
}
```

- **错误：**

| 状态码 | 说明 |
|--------|------|
| 400 | 参数缺失 / slug 格式不合法 / 版本号未递增 / 版本已存在 / 已有进行中的申请 |
| 401 | 未登录 |
| 403 | SMH 未开启 |

---

### `POST /openclaw/skills/takedown` — 申请下架

员工申请下架自己上传的技能。下架语义针对**整个 slug**（与 `POST /admin/skills/offline` 一致）：管理员审核通过后，该 slug 下所有 `published` 版本变为 `offline`。

申请单会写入 `review_requests`：`action_type=takedown`，`resource_id` 绑定该 slug 当前**最新** `published` 版本的 skill id（冗余字段，便于详情展示）；互斥与列表展示均按 **slug**。

归属校验：该 slug 下存在 `uploader_id=当前用户` 且 `status=published` 的版本即可申请（不要求绑定到某一固定版本行）。

- **权限**：已登录用户
- **Content-Type：** `application/json`

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 要下架的技能 slug |
| reason | string | 是 | 下架理由 |

- **响应：** `200`

```json
{
  "ok": true,
  "request_id": 8
}
```

- **错误：**

| 状态码 | 说明 |
|--------|------|
| 400 | 缺少 slug / 缺少 reason / 已有进行中的申请 |
| 401 | 未登录 |
| 403 | 只能下架自己上传的技能（该 slug 无本人 published 版本，或仅为管理员上传） |
| 404 | 技能不存在或无任何 `published` 版本 |

---

### `GET /openclaw/skills/contributions` — 我的申请列表

查看当前用户的审核申请列表，支持按状态、操作类型筛选和关键词搜索。

- **权限**：已登录用户

- **查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| status | string | 否 | 筛选状态：`pending` / `approved` / `rejected` / `withdrawn` |
| action_type | string | 否 | 筛选操作类型：`publish` / `takedown` |
| keyword | string | 否 | 按技能 slug 或名称模糊搜索 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **响应：** `200`

```json
{
  "requests": [
    {
      "id": 7,
      "requester_id": 5,
      "resource_type": "skill",
      "resource_id": 42,
      "action_type": "publish",
      "slug": "my-skill",
      "status": "pending",
      "reason": "",
      "reviewer_id": 0,
      "reviewed_at": null,
      "review_comment": "",
      "skill_name": "My Skill",
      "created_at": "2026-07-27T15:00:00Z",
      "updated_at": "2026-07-27T15:00:00Z"
    }
  ],
  "total": 1,
  "pending_total": 1,
  "page": 1,
  "page_size": 20
}
```

> `skill_name`：关联技能名称，Skill 已软删除时为空。`pending_total`：当前用户 pending 申请总数，不受筛选影响。

---

### `GET /openclaw/skills/contributions/detail` — 申请详情

查看审核申请详情。申请人本人或管理员可查看。

- **权限**：已登录用户（申请人本人或管理员）

- **查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 申请 ID |

- **响应：** `200`

```json
{
  "request": {
    "id": 7,
    "requester_id": 5,
    "resource_type": "skill",
    "resource_id": 42,
    "action_type": "publish",
    "slug": "my-skill",
    "status": "approved",
    "reviewer_id": 1,
    "reviewed_at": "2026-07-27T16:00:00Z",
    "review_comment": ""
  },
  "skill": {
    "id": 42,
    "slug": "my-skill",
    "name": "My Skill",
    "version": "1.0.0",
    "status": "published",
    "uploader_id": 5
  }
}
```

- **错误：**

| 状态码 | 说明 |
|--------|------|
| 400 | 缺少 id |
| 401 | 未登录 |
| 403 | 非申请人且非管理员 |
| 404 | 申请不存在 |

---

### `GET /admin/contributions` — 管理员审核申请列表

管理员查看所有审核申请，支持按资源类型、操作类型、状态筛选和关键词搜索。

- **权限**：管理员

- **查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| resource_type | string | 否 | 筛选资源类型：`skill`（默认全部） |
| action_type | string | 否 | 筛选操作类型：`publish` / `takedown` |
| status | string | 否 | 筛选状态：`pending` / `approved` / `rejected` / `withdrawn` |
| keyword | string | 否 | 按技能 slug 或名称模糊搜索 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **响应：** `200`

```json
{
  "requests": [
    {
      "id": 7,
      "requester_id": 5,
      "requester_name": "employee",
      "resource_type": "skill",
      "resource_id": 42,
      "action_type": "publish",
      "slug": "my-skill",
      "status": "pending",
      "reason": "",
      "reviewer_id": 0,
      "reviewed_at": null,
      "review_comment": "",
      "skill_name": "My Skill",
      "created_at": "2026-07-27T15:00:00Z",
      "updated_at": "2026-07-27T15:00:00Z"
    }
  ],
  "total": 1,
  "pending_total": 3,
  "page": 1,
  "page_size": 20
}
```

> `skill_name`：关联技能名称，Skill 已软删除时为空。`pending_total`：全系统 pending 申请总数，不受筛选影响。

---

### `GET /admin/contributions/detail` — 管理员审核申请详情

管理员查看审核申请详情，含关联 Skill 信息。

- **权限**：管理员

- **查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 申请 ID |

- **响应：** `200`

```json
{
  "request": {
    "id": 7,
    "requester_id": 5,
    "resource_type": "skill",
    "resource_id": 42,
    "action_type": "publish",
    "slug": "my-skill",
    "status": "pending",
    "reason": ""
  },
  "requester_name": "employee",
  "skill": {
    "id": 42,
    "slug": "my-skill",
    "name": "My Skill",
    "version": "1.0.0",
    "status": "pending_review",
    "uploader_id": 5
  }
}
```

---

### `POST /admin/contributions/approve` — 审核通过

管理员审核通过申请。根据 `action_type` 自动执行对应操作：

- `publish`：对应 Skill 记录状态 `pending_review` → `published`（上架该版本）
- `takedown`：将该申请 `slug` 下**所有** `status=published` 的版本更新为 `offline`（整技能下架，与 `POST /admin/skills/offline` 一致）。即使历史申请单的 `resource_id` 指向旧版本，也按 slug 批量生效。

- **权限**：管理员
- **Content-Type：** `application/json`

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 申请 ID |

- **响应：** `200`

```json
{
  "ok": true
}
```

- **错误：**

| 状态码 | 说明 |
|--------|------|
| 400 | 缺少 id / 申请已审核（非 pending 状态） / 版本冲突 |
| 403 | 非管理员 |
| 404 | 申请不存在 |

---

### `POST /admin/contributions/reject` — 审核拒绝

管理员审核拒绝申请。根据 `action_type` 自动执行对应操作：

- `publish`：软删除 Skill 记录（技能不上架）
- `takedown`：Skill 不变（保持 `published`）

- **权限**：管理员
- **Content-Type：** `application/json`

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 申请 ID |
| review_comment | string | 是 | 拒绝理由 |

- **响应：** `200`

```json
{
  "ok": true
}
```

- **错误：**

| 状态码 | 说明 |
|--------|------|
| 400 | 缺少 id / 缺少 review_comment / 申请已审核 |
| 403 | 非管理员 |
| 404 | 申请不存在 |

---

### `POST /openclaw/skills/contributions/withdraw` — 撤回申请

员工撤回自己的待审核申请。撤回后置为 `withdrawn` 终态（不删除记录）。

- 撤回 `publish` 申请：软删除关联的 Skill 记录（技能不会上架）
- 撤回 `takedown` 申请：Skill 状态不变，仅 ReviewRequest 变为 `withdrawn`
- 只有 `status=pending` 的申请可以撤回
- 只有申请人本人可以撤回

- **权限**：已登录用户
- **Content-Type：** `application/json`

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 申请 ID |

- **响应：** `200` —— `{"ok": true}`

- **错误：**

| 状态码 | 说明 |
|--------|------|
| 400 | 缺少 id / 申请非 pending 状态 |
| 401 | 未登录 |
| 403 | 非申请人本人 |
| 404 | 申请不存在 |

---

### `POST /admin/skills/offline` — 管理员下架技能

管理员直接下架技能（不走审核流程，立即生效）。仅对 `status=published` 的技能有效。

- **权限**：管理员
- **Content-Type：** `application/x-www-form-urlencoded`

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能 slug |

- **响应：** `200` —— `{"ok": true}`

- **错误：**

| 状态码 | 说明 |
|--------|------|
| 400 | 缺少 slug |
| 403 | 非管理员 |
| 404 | 技能不存在或非 `published` 状态 |

---

### `POST /admin/skills/online` — 管理员上架技能

管理员直接上架技能（不走审核流程，立即生效）。仅对 `status=offline` 的技能有效。反向操作，非破坏性，无需二次确认。

- **权限**：管理员
- **Content-Type：** `application/x-www-form-urlencoded`

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 技能 slug |

- **响应：** `200` —— `{"ok": true}`

- **错误：**

| 状态码 | 说明 |
|--------|------|
| 400 | 缺少 slug |
| 403 | 非管理员 |
| 404 | 技能不存在或非 `offline` 状态 |

---

### `GET /admin/skills` — 技能列表（字段扩展）

管理员技能列表响应中每个 skill 对象新增以下字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| status | string | 技能状态：`published` / `pending_review` / `offline` |
| uploader_id | uint | 上传者 ID：`0`=管理员上传，`>0`=员工 user_id |
| uploader_name | string | 上传者用户名（`uploader_id=0` 时为空字符串） |
| pending_review | object \| null | 待审核申请信息，无待审核申请时为 `null` |

`pending_review` 对象：

| 字段 | 类型 | 说明 |
|------|------|------|
| request_id | uint | 申请 ID |
| action_type | string | `publish` / `takedown` |
| requester_id | uint | 申请人 user_id |
| requester_name | string | 申请人用户名 |

> **关联规则：** 列表每个 slug 只返回最新版本行；`pending_review` 按 **slug** 关联 `review_requests` 中 `status=pending` 的申请（而非按当前行 `skill.id == resource_id`）。因此即使历史下架申请的 `resource_id` 指向旧版本，最新行仍会显示 `pending_review`。

支持查询参数 `?status=published` 按状态筛选。

---

## 用户端 MCP 管理

用户端 MCP 管理接口允许实例用户自助查看、添加、编辑、删除、开关企业 MCP，并支持连通性探测。

**准入条件**：所有接口需用户登录，实例 `agent_type = openclaw` 且 `agent_version >= 2026.3.28`。

---

### `GET /openclaw/mcp/available` — 企业可选 MCP 列表

- **说明：** 返回当前用户可见的企业 MCP 列表，排除该实例已添加的。对于开启密钥托管的 MCP，config_json 中有管理员默认值的占位符字段会被自动移除（用户无需填写）。
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID |
| q | string | 否 | 搜索关键词（匹配名称/描述） |

- **响应：**

```json
{
  "items": [
    {
      "id": 1,
      "service_id": "github-mcp",
      "name": "GitHub MCP",
      "description": "GitHub 集成工具",
      "transport_type": "streamable-http",
      "config_json": "{\"transportType\":\"streamable-http\",\"url\":\"https://mcp.github.com\",\"headers\":{\"Authorization\":\"<token>\"}}"
    }
  ],
  "total": 1
}
```

> **密钥托管说明：** config_json 中形如 `<xxx>` 的字段为占位符，用户需要替换为真实值后提交。有管理员默认值的占位符不会出现在此处（已由管理员预设）。

---

### `POST /openclaw/mcp/add` — 添加 MCP 到实例

- **说明：** 将指定 MCP 下发到用户的一个或多个实例。支持单实例和批量（通过 `instance_ids` 数组控制）。逐实例校验条件，不满足的标记 `skipped`。下发成功后异步探测连通性，结果通过 `/openclaw/mcp/list` 的 `connection_status` 字段查看。
- **请求体：**

```json
{
  "instance_ids": [123, 456],
  "service_id": "github-mcp",
  "config_json": "{\"transportType\":\"streamable-http\",\"url\":\"https://mcp.github.com\",\"headers\":{\"Authorization\":\"Bearer ghp_xxx\"}}",
  "restart": true
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_ids | []uint | 是 | 实例 ID 列表（最多 50 个），单实例时传 `[123]` |
| service_id | string | 是 | MCP 的 service_id |
| config_json | string | 是 | 前端替换占位符后的完整 JSON 配置，必须包含 `url` 或 `command` 字段。**不允许包含未替换的占位符（`<xxx>`）**，有默认值的字段可以不传（不出现在 config_json 中） |
| restart | bool | 否 | 是否在下发成功后重启 openclaw-gateway，默认 `false` |

> **密钥托管处理：** 对于 `key_hosted=true` 的 MCP，后端会比对用户提交的 config_json 和原始模板，提取用户填写的值存入 DB，安全网关转发时自动注入。URL path 和非占位符字段不允许修改。

- **响应：**

```json
{
  "total": 2,
  "success": 2,
  "failed": 0,
  "skipped": 0,
  "items": [
    {"instance_id": 123, "status": "success"},
    {"instance_id": 456, "status": "success"}
  ]
}
```

| 字段 | 说明 |
|------|------|
| total | 总处理数 |
| success | 下发成功数 |
| failed | 下发失败数 |
| skipped | 跳过数（不满足条件） |
| items[].instance_id | 实例 ID |
| items[].status | `success` / `failed` / `skipped` |
| items[].error | 失败/跳过原因 |

- **失败响应：**
  - `400 {"error": "配置中存在未填写的占位符: token"}`
  - `400 {"error": "不允许修改 URL 路径"}`
  - `400 {"error": "不允许修改 header X-Trace-Id"}`
  - `400 {"error": "请填写以下托管字段: token"}`

---

### `GET /openclaw/mcp/list` — 实例已添加的 MCP 列表

- **说明：** 返回实例上所有已配置的 MCP（含成功和失败状态），包括连接状态、工具列表、完整配置。
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID |
| q | string | 否 | 搜索关键词（匹配名称/service_id） |

- **响应：**

```json
{
  "items": [
    {
      "id": 1,
      "service_id": "github-mcp",
      "name": "GitHub MCP",
      "description": "GitHub 集成工具",
      "transport_type": "streamable-http",
      "source": "user",
      "version": "1.0.0",
      "install_status": 2,
      "error_message": "",
      "config_json": "{...}",
      "connection_status": "connected",
      "connection_error": "",
      "tools": ["create_issue", "list_repos"],
      "probed_at": "2026-05-13 10:30:00",
      "updated_at": "2026-05-13 09:00:00"
    }
  ],
  "total": 1
}
```

| 字段 | 说明 |
|------|------|
| install_status | 2=成功, 3=失败 |
| error_message | 操作失败时的错误原因 |
| connection_status | `connected` / `failed` / `unsupported` / `unconfigured` / `""`（未探测） |
| tools | 连接成功时返回的工具名称列表 |
| source | `admin`=管理端下发, `user`=用户自选 |

---

### `POST /openclaw/mcp/refresh-status` — 刷新 MCP 连接状态

- **说明：** 对实例上的 MCP 发起连通性探测（仅 HTTP 公网可达的 MCP），更新连接状态和工具列表。STDIO 类型返回 `unsupported`，含 `<xxx>` 占位符的返回 `unconfigured`。
- **请求体：**

```json
{
  "id": 123,
  "service_ids": ["github-mcp"]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID |
| service_ids | []string | 否 | 指定刷新的 MCP，为空则刷新全部 |

- **响应：**

```json
{
  "items": [
    {
      "service_id": "github-mcp",
      "connection_status": "connected",
      "tools": ["create_issue", "list_repos"],
      "error": ""
    }
  ]
}
```

---

### `POST /openclaw/mcp/update-config` — 编辑 MCP 配置

- **说明：** 修改实例上某个 MCP 的 JSON 配置并下发。下发失败时标记 `install_status=Failed`。
- **请求体：**

```json
{
  "id": 123,
  "service_id": "github-mcp",
  "config_json": "{...}",
  "restart": true
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID |
| service_id | string | 是 | MCP service_id |
| config_json | string | 是 | 新的完整 JSON 配置 |
| restart | bool | 否 | 是否立即重启 openclaw-gateway（默认 false） |

- **响应：**

```json
{
  "ok": true,
  "sync_status": "success",
  "restarted": true
}
```

---

### `POST /openclaw/mcp/delete` — 删除 MCP

- **说明：** 从实例移除某个 MCP。删除失败时不删 DB 记录，标记 `install_status=Failed`。
- **请求体：**

```json
{
  "id": 123,
  "service_id": "github-mcp",
  "restart": true
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID |
| service_id | string | 是 | MCP service_id |
| restart | bool | 否 | 是否立即重启（默认 false） |

- **响应：**

```json
{
  "ok": true,
  "restarted": true
}
```

---

### `POST /openclaw/mcp/toggle` — 开启/关闭 MCP

- **说明：** 在 config_json 中注入或移除 `"disabled": true`，然后下发到实例。下发失败时标记 `install_status=Failed`。
- **请求体：**

```json
{
  "id": 123,
  "service_id": "github-mcp",
  "disabled": true,
  "restart": true
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID |
| service_id | string | 是 | MCP service_id |
| disabled | bool | 是 | true=关闭, false=开启 |
| restart | bool | 否 | 是否立即重启（默认 false） |

- **响应：**

```json
{
  "ok": true,
  "restarted": true
}
```

## Agent 命令执行（管控端）

> 来源：feature/agent_command_execution（OpenSpec change agent-command-execution）。
> 详细需求与流程见 `tat-requirements/requirements.md` + `tat-requirements/api.md`。
> 本期共 7 个端点，全部需要管理员权限（`requireAdmin`），4 个写端点全部走 `WithAudit()`。

### 数据层级

```
[ClawPro 用户视角] dispatch（dispatch_slug = task-xxxxxxxx）  ← 用户从前端"下发"一次
   ↓ 1:1~2
[TAT 概念] invocation（tat_invocation_id = inv-xxxxxxxx）     ← 一次 RunCommand 调用
   ↓ 1:N
[TAT 概念] task（tat_invocation_task_id = invt-xxxxxxxx）     ← 每台 instance 一条
```

物理表（v2 数据模型）：

| 表 | 说明 |
|----|------|
| agent_command_dispatch | 一次用户视角的下发 = 1 行；持有 command_snapshot / param_values_json / triggered_by_user_id / status 等顶层字段 |
| agent_command_invocations | 一次 RunCommand = 1 行；`dispatch_id` FK 关联 dispatch 表，`dispatch_slug` 冗余便于反查 |
| agent_command_tasks | 每台 instance 一行；`dispatch_id` / `invocation_id` 双 FK，`dispatch_slug` 冗余 |

- 一次用户 dispatch 最多 2 个 invocation：测试机 = 1 次 RunCommand（1 台）+ 生产 = 1 次 RunCommand（≤ 199 台）。`instance_ids` 上限 200 与 TAT 单批 200 对齐（详见 [TAT RunCommand 文档](https://cloud.tencent.com/document/api/1340/52676)），本期生产阶段不分批。
- `dispatch_slug` 由后端系统生成（`task-{8 位随机}`），同租户内全局唯一。
- `tat_invocation_id` / `tat_invocation_task_id` 直接来自 TAT API 返回值，不二次包装。
- `agent_commands.content` 与 `command_snapshot.content` 一律存 raw 文本，base64 仅在 `controller/tat.go` 调 TAT SDK 前那一刻完成。
- `dispatch.status` 是显式持久字段（`in_progress` / `awaiting_confirmation` / `success` / `partial` / `failed` / `cancelled`），由「事件式更新 + 后台 reconcile 兜底」推进；列表/详情接口直接读该字段。

### 审计动作命名

| 端点 | action | resource_type |
|------|--------|---------------|
| POST /admin/agent-commands/create | `agent_command_create` | `agent_command` |
| POST /admin/agent-commands/update | `agent_command_update` | `agent_command` |
| POST /admin/agent-commands/delete | `agent_command_delete` | `agent_command` |
| POST /admin/agent-commands/dispatch | `agent_command_dispatch` | `agent_command_dispatch` |
| POST /admin/agent-commands/schedules/create | `agent_command_schedule_create` | `agent_command_schedule` |
| POST /admin/agent-commands/schedules/delete | `agent_command_schedule_delete` | `agent_command_schedule` |
| POST /admin/agent-commands/schedules/toggle | `agent_command_schedule_toggle` | `agent_command_schedule` |

### `GET /admin/agent-commands` — 命令模板列表

仅返回当前租户内活跃命令（`deleted_at IS NULL`）。

- **权限：** 管理员
- **Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| q | string | 否 | 模糊匹配 slug / 名称 / 内容 / 创建人用户名（4 字段任一命中即返回） |
| page | uint | 否 | 页码，默认 1 |
| page_size | uint | 否 | 每页数量，默认 20，上限 100 |
| sort | string | 否 | `updated_at_desc`（默认）/ `name_asc` |

- **响应：** `200` —— `commands[]` 每项含 id / slug / name / type / content（raw）/ params / created_by_username / can_edit / last_executed_at / executed_count。

### `POST /admin/agent-commands/create` — 创建命令模板

slug 由后端生成 `cmd-{8位随机}`，请求体不传。

- **权限：** 管理员
- **审计：** `agent_command_create`
- **请求体字段：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 中英文/数字/下划线/`-`/`.`，≤ 60 字符 |
| description | string | 否 | 0–512 字符，默认 "" |
| type | string | 否 | 当前枚举仅 `SHELL`，默认 `SHELL` |
| content | string | 是 | 1–8192 字符；可含 `{{变量名}}` |
| timeout_sec | uint | 否 | 1–86400（秒），上限 1 天，与 TAT API 对齐，默认 60 |
| run_user | string | 否 | TAT 执行用户，长度 ≤ 64 字符，默认 `root` |
| workdir | string | 否 | TAT 工作目录，长度 ≤ 255 字符，默认 `/root` |
| params | array | 否 | 0–10 项，每项 `{name, default, description}`（name ≤ 32，default ≤ 128，description ≤ 200），默认 `[]` |

- **响应：** `201` —— 同列表 item 结构。
- **错误：** `name_invalid_chars` / `name_too_long` / `description_too_long` / `content_required` / `content_too_long` / `timeout_out_of_range` / `run_user_too_long` / `workdir_too_long` / `invalid_type` / `params_too_many` / `param_name_invalid` / `param_name_duplicated` / `param_default_too_long` / `param_description_too_long` / `409 name_already_exists`（同租户下命令名称重复，软删行不计入） / `409 quota_exceeded`（活跃 ≥ 500）。

### `POST /admin/agent-commands/update` — 编辑命令模板

仅创建者本人或初始管理员可编辑。`type` / `slug` 字段不可修改。`params` 整体替换语义。

- **审计：** `agent_command_update`
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 命令模板主键 ID |
| name | string | 否 | 中英文/数字/下划线/`-`/`.`，≤ 60 字符 |
| description | string | 否 | 0–512 字符 |
| content | string | 否 | 1–8192 字符；可含 `{{变量名}}` |
| timeout_sec | uint | 否 | 1–86400（秒），默认 60 |
| run_user | string | 否 | TAT 执行用户，长度 ≤ 64 字符，默认 `root` |
| workdir | string | 否 | TAT 工作目录，长度 ≤ 255 字符，默认 `/root` |
| params | array | 否 | 0–10 项，每项 `{name, default, description}` |

- **错误：** 创建端点全部错误 + `id_required` / `403 permission_denied` / `404 command_not_found`。

### `POST /admin/agent-commands/delete` — 软删命令模板

GORM 软删（`deleted_at = NOW()`）；保留 fk 引用便于历史 dispatch 反查。

- **审计：** `agent_command_delete`
- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 命令模板主键 ID |

- **响应：** `200 {"ok":true,"id":12,"deleted_at":"..."}`
- **错误：** `403 permission_denied` / `404 command_not_found` / `409 command_in_use`。

`command_in_use` 错误体携带阻塞 dispatch 列表：

```json
{
  "code": 409,
  "error": "command_in_use",
  "message": "命令仍有进行中的执行任务，无法删除",
  "blocking_tasks": [{ "dispatch_slug": "task-x9k2m4n7" }]
}
```

### `POST /admin/agent-commands/dispatch` — 下发命令（启动 / 续跑 / 终止 三模式单端点）

接口按入参分流为 **A 启动 / B 续跑 / C 终止** 三种模式，**全部走同一个端点 + 同一个审计 action**。第 1 次调用启动 dispatch（最多到测试机阶段），后续由前端在用户点【继续下发剩余 N 台】或【终止下发】时再调一次。

- **权限：** 管理员
- **审计：** `agent_command_dispatch`

#### 模式 A — 启动新 dispatch

事务内预创建 1+ 条 invocation + 对应 task，立即返回 `dispatch_slug`，TAT 调用与状态推进异步执行。
当 `test_first=true` 时，后端**仅触发测试机的 RunCommand**；测试机终态后 dispatch 进入 `awaiting_confirmation` 状态等待用户决策。

**请求体：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| command_id | uint | 是 | 命令模板主键 |
| instance_ids | array\<uint\> | 是 | 目标 Agent ID，去重后 1 ≤ N ≤ 200（与 TAT 单次 RunCommand 上限对齐，不分批） |
| test_first | bool | 否 | 是否启用测试机优先 |
| test_target_instance_id | uint | 否 | 测试机；test_first=true 时**必填**且必须在 instance_ids |
| param_values | object | 否 | `{name: value}`；按命令 params 校验 |

参数取值规则：
- key **存在** 于 `param_values` 中 → 使用用户值（即使是空字符串 `""` 也透传，不回退 default）
- key **不存在** → 用 param 的 `default`（声明了的话）兜底；都没有则返回 `param_value_required`
- 多余的 key（命令未声明） → 返回 `param_unknown`

RunCommand 分批：测试机阶段 = 1 invocation（1 台）；生产阶段 = 剩余 M 台拆 `ceil(M/200)` 个 invocation（本期 M ≤ 199，固定 1 个 invocation）。生产 invocation 在 A 模式下仅预创建为 `pending` 状态，不发出 RunCommand。

**响应：** `200`

```json
{
  "dispatch_slug": "task-x9k2m4n7",
  "status": "in_progress",
  "target_count": 3,
  "test_first": true,
  "test_target_instance_id": 123,
  "param_values": {"name": "alice"},
  "invocations_planned": 2,
  "started_at": "2026-05-21T09:50:00+08:00"
}
```

> 响应不含 TAT IDs（异步返回）；前端走详情接口拉真实 ID。
> 详情接口的 `status` 字段在测试机终态后会变为 `awaiting_confirmation`，前端据此弹出「继续 / 终止」按钮。

#### 模式 B — 续跑（用户点【继续下发剩余 N 台】）

**请求体：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| dispatch_slug | string | 是 | 第 1 次调用返回的 slug |

A 模式独有字段（`command_id` / `instance_ids` / `test_first` / `test_target_instance_id` / `param_values`）必须**全部为零值**，否则返回 `400 dispatch_slug_with_extra_params`。

后端从 DB 中找出所有 `is_test_run=false AND status='pending' AND tat_invocation_id=''` 的 invocation，逐个异步触发 RunCommand。脚本内容、参数从 `command_snapshot` + `param_values_json` 还原。

**响应：** `200`

```json
{
  "dispatch_slug": "task-x9k2m4n7",
  "status": "in_progress",
  "invocations_planned": 1,
  "started_at": "2026-05-21T09:55:00+08:00"
}
```

#### 模式 C — 终止（用户点【终止下发】）

**请求体：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| dispatch_slug | string | 是 | 第 1 次调用返回的 slug |
| abort | bool | 是 | 必须为 `true` |

A 模式独有字段必须为零值（同 B 模式约束）。

后端把所有 `is_test_run=false AND status='pending'` 的 invocation 与 task 状态置为 `cancelled`；已发出（`tat_invocation_id != ''`）的 invocation 不动 —— 本期不调 TAT `CancelInvocation`。

**响应：** `200`

```json
{
  "dispatch_slug": "task-x9k2m4n7",
  "status": "cancelled",
  "cancelled_count": 4
}
```

#### 错误码（按模式标注）

| 错误码 | HTTP | 触发模式 | 说明 |
|--------|------|---------|------|
| command_required | 400 | A | 未传 command_id |
| targets_required | 400 | A | instance_ids 为空 |
| too_many_targets | 400 | A | instance_ids 去重后 > 200 |
| test_target_required | 400 | A | test_first=true 但未传 test_target_instance_id |
| test_target_invalid | 400 | A | test_target 不在 instance_ids |
| param_value_required | 400 | A | 缺少必填参数 |
| param_unknown | 400 | A | 提供命令未声明的参数 |
| command_not_found | 404 | A | command_id 在当前租户下不存在或已软删 |
| instance_not_found | 404 | A | instance_ids 含未知 ID |
| target_offline | 409 | A | 实例非 RUNNING 状态 |
| dispatch_slug_with_extra_params | 400 | B/C | 携带 dispatch_slug 时同时带了 A 模式字段 |
| dispatch_slug_required | 400 | C | 携带 abort=true 但未带 dispatch_slug |
| dispatch_not_found | 404 | B/C | slug 在当前租户下不存在 |
| permission_denied | 403 | B/C | 调用者既非原 triggered_by 也非初始管理员 |
| test_phase_in_progress | 409 | B/C | 测试机仍未终态 |
| test_phase_failed | 409 | B/C | 测试机失败，dispatch 已 failed |
| already_continued | 409 | B/C | 生产批已发出（`tat_invocation_id != ''`） |
| already_completed | 409 | B/C | dispatch 已是 success/partial/failed/cancelled 任一终态 |
| nothing_to_abort | 409 | C | 没有可取消的 pending 行（并发：另一边已先一步动作） |

### `GET /admin/agent-commands/tasks` — 执行记录列表

按 `dispatch_slug` 聚合。仅查 ClawPro DB 缓存。

- **Query：** `triggered_by` / `q`（模糊匹配 dispatch_slug / 命令名 / 命令内容 / 操作人用户名）/ `status` / `started_after` / `started_before` / `page` / `page_size`
- **响应：** `tasks[]` 每项含 dispatch_slug / command_name / command_content_preview / triggered_by_username / triggered_by_email / **test_first**（v2 新增，bool，是否走灰度模式）/ status / target_count（含测试机）/ success_count / failed_count / invocation_count（1 或 2）/ started_at / finished_at。

### `GET /admin/agent-commands/tasks/detail` — 执行记录详情

主体走 DB；stdout/stderr 实时调 TAT `DescribeInvocationTasks` 批量拉取。

- **Query：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| dispatch_slug | string | 是 | 用户视角下发 ID |
| with_output | bool | 否 | 是否拉取 stdout/stderr，默认 `true` |

- **响应：** 嵌套结构 `invocations[].tasks[]`，含 command_snapshot / param_values / test_summary / 全聚合计数。stdout/stderr 单字段超 64KB 自动截断并设 `*_truncated=true`；TAT 端过期 task 设 `output_expired=true`、stdout/stderr=null，整体仍 200。每条 task 还含 `error_info`：来自 TAT 顶层 `ErrorInfo` 字段，启动阶段失败（`status=unreachable`，对应 TAT `DELIVER_FAILED` / `START_FAILED`）时承载具体原因（如 "user xxx does not exist"），便于前端定位 stdout/stderr 都为空的问题；正常执行的 task 该字段为空字符串。
- **错误：** `400 dispatch_slug_required` / `404 dispatch_not_found` / `502 tat_unavailable`（仅 with_output=true 且 TAT 整体不可用时）。

### `POST /admin/agent-commands/agent-status` — 查询 TAT Agent 客户端状态

实时调腾讯云 [DescribeAutomationAgentStatus](https://cloud.tencent.com/document/api/1340/52682) 接口，返回每个 Agent 实例上 TAT 客户端的在线状态、版本、上次心跳时间等。前端可在「下发命令」前用此接口检查目标实例是否可用。

- **请求 body：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_ids | string[] | 是 | 实例 ID 列表，最多 200 个，自动去空 / 去重 |

- **响应：** `200`

```json
{
  "agents": [
    {
      "instance_id": "ins-aaa",
      "agent_status": "Online",
      "version": "1.0.0",
      "last_heartbeat_time": "2026-05-22T10:00:00Z",
      "environment": "Linux"
    },
    {
      "instance_id": "ins-bbb",
      "agent_status": "Offline",
      "environment": "Linux"
    },
    {
      "instance_id": "ins-ccc",
      "agent_status": "Unknown"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| agents[].instance_id | string | 实例 ID（按入参顺序，去空去重后保留首次出现） |
| agents[].agent_status | string | `Online` / `Offline` / `Unknown`（TAT 未返回时补 Unknown） |
| agents[].version | string | TAT Agent 版本号，可空 |
| agents[].last_heartbeat_time | string | TAT 原样字符串（UTC，未做时区转换），可空 |
| agents[].environment | string | `Linux` / `Windows`，可空 |

- **错误：**
  - `400 instance_ids_required` — `instance_ids` 缺失或全部为空
  - `400 too_many_instance_ids` — 超过 100 上限
  - `502 tat_describe_agent_failed` — 上游 TAT API 调用失败

### 定时任务（schedule）

> 定时任务 = 「何时触发一次 dispatch」的配置；到期由后台 runner 调用 `startDispatch` 触发，执行链路完全复用上述 dispatch 体系。完整字段说明与 curl 联调示例见 `docs/AGENT_COMMAND_SCHEDULE_API.md`。
>
> 约定：时间一律按**服务器本地时区**解释（不支持自定义时区）；单租户上限 **1000** 条；对外资源 ID 为 `sch-{8 位随机}`（租户内唯一，所有接口的 `id`/`schedule_id` 均用此字符串，不暴露数据库主键）；定时触发为**尽力而为**（部分目标离线只对在线机器下发、离线记 `unreachable`，全部离线才整轮失败）。早期的 `update`（编辑）/ `run-now`（立即触发）端点已移除，编辑请「删除后重建」。

**调度表达式 `schedule`**（关键字大小写不敏感，存储/回显为 canonical 小写、`HH:MM` 补零）：

| 形态 | 含义 | 示例 |
|------|------|------|
| `once(<YYYY-MM-DD HH:MM>)` | 一次性，精确到分钟 | `once(2026-06-30 15:00)` |
| `every(d, at=<HH:MM>)` | 每天 | `every(d, at=02:00)` |
| `every(w, on=<1-7>, at=<HH:MM>)` | 每周；`on` 1=周一..7=周日 | `every(w, on=1, at=09:00)` |
| `every(m, on=<1-31>, at=<HH:MM>)` | 每月几号；无该日的月份整月跳过 | `every(m, on=1, at=09:00)` |
| `cron(<分 时 日 月 周>)` | 标准 5 字段 cron；周 0-6（0=周日..6=周六，不接受 7） | `cron(*/5 * * * *)`、`cron(0 9 * * 1-5)` |
| `interval(<n><m\|h\|d>, begin=<YYYY-MM-DD HH:MM>[, end=<...>])` | 从 `begin` 起每隔 N 触发（单位 m/h/d）；可选 `end` 截止，超过不再触发 | `interval(1m, begin=2026-06-30 15:00)`、`interval(2h, begin=2026-06-30 15:00, end=2026-07-30 15:00)` |

> 不支持周期数字（`every(3d,...)`、`every(1w,...)` 均非法）；`once` 时刻过期、`interval` 超过 `end` 时创建/启用返回 `400 schedule_spec_invalid`。
>
> ⚠️ `every` 与 `cron` 是两套独立的解析体系，**周字段约定不同**：`every` 的 `on` 为 **1-7（1=周一..7=周日）**；`cron` 沿用标准 crontab 的 **0-6（0=周日..6=周六，不接受 7）**。请勿混用。

**合成状态 `status`**（优先级从高到低，可用于列表 `?status=` 筛选）：`completed`（once 已执行终态）> `running`（有 dispatch 在执行）> `paused`（被停用）> `pending`（从未触发）> `waiting`（执行过等待下次）。

#### `GET /admin/agent-commands/schedules` — 列表

- **Query：** `q`（按名称模糊匹配）/ `status`（状态筛选，非法值 `400 invalid_status`）/ `page` / `page_size`（默认 20，上限 100）
- **响应 `200`：** `schedules[]`（每项含 id（资源 ID `sch-xxxx`）/ name / description / command_id / command_name / instance_ids / param_values / schedule / enabled / is_running / status / next_run_at / first_run_at / last_run_at / last_dispatch_slug / created_by_user_id / created_by_username / can_edit / created_at / updated_at）/ total / page / page_size。

#### `POST /admin/agent-commands/schedules/create` — 创建

- **请求体：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 名称，≤32 字符 |
| description | string | 否 | 描述，≤64 字符 |
| command_id | uint | 是 | 命令模板 ID（须存在于当前租户） |
| instance_ids | uint[] | 是 | 目标实例 ID，至少 1 台 |
| param_values | map<string,string> | 否 | 命令参数值 |
| schedule | string | 是 | 调度表达式，见上表 |

- **响应：** `201` + 单条 schedule 视图（结构同列表项，`can_edit=true`）。
- **错误：** `400 invalid_body` / `400 schedule_spec_invalid`（超长 / 命令不存在 / 无实例 / 表达式非法 / once 已过期）/ `409 quota_exceeded`（达 1000 上限）。

#### `POST /admin/agent-commands/schedules/toggle` — 启停

- **请求体：** `{ "id": "sch-xxxx", "enabled": <bool> }`，启用时重算 `next_run_at`。
- **响应：** `200` + 更新后的 schedule 视图。
- **错误：** `400 id_required` / `400 schedule_spec_invalid`（启用时触发时刻已过期）/ `403 permission_denied`（非创建者且非初始管理员）/ `404 schedule_not_found` / `409 schedule_completed`（已完成的一次性任务禁止启停）。

#### `POST /admin/agent-commands/schedules/delete` — 软删

- **请求体：** `{ "id": "sch-xxxx" }`
- **响应 `200`：** `{ "ok": true, "id": "sch-xxxx" }`
- **错误：** `400 id_required` / `403 permission_denied` / `404 schedule_not_found`。

#### `GET /admin/agent-commands/schedules/records` — 执行记录

记录仅存 `dispatch_slug`，本接口实时批量查 dispatch 表拼装状态/计数（避免 N+1）。

- **Query：** `schedule_id`（必填，资源 ID `sch-xxxx`，缺失 `400 schedule_id_required`，不存在 `404 schedule_not_found`）/ `page` / `page_size`
- **响应 `200`：** `records[]`（每项含 id / dispatch_slug / status / target_count / success_count / failed_count / started_at / finished_at / created_at）/ total / page / page_size。`status` 为 dispatch 状态枚举。

---

## 统一账号模式（OneID 登录代理）

> 仅在统一账号模式（`is_unified_account=true`）下生效。前端通过 `/site` 接口的 `is_unified_account` 字段判断是否展示 OneID 登录界面。

### `GET /oneid/encrypt_setting`

获取 OneID 登录加密公钥配置。透明代理到 OneID base API。

- **认证**：无需登录
- **响应**：原样返回 OneID 响应

---

### `GET /oneid/login-name?username={username}`

查询本地用户对应的 OneID 登录名。无需登录态，前端在登录前调用此接口获取 OneID 登录名。

- **认证**：无需登录
- **Query 参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 本地用户名（可能是中文姓名） |

- **成功响应**：

```json
{
  "username": "张三",
  "oneid_login_name": "zhangsan"
}
```

- **错误**：
  - `404` — 用户不存在
  - `400` — 缺少 username 参数

---

### `POST /oneid/enterprise?username={username}`

OneID 企业登录验证。透明代理到 OneID，成功后建立本地 hatchery session。

- **认证**：无需登录
- **Query 参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 本地用户名，用于建立 session |

- **请求体**：原样转发给 OneID（JSON，含 credential、accountId、captchaVerification）

- **成功响应**（OneID 验证通过且无后续步骤）：

```json
{
  "...OneID 原始字段...",
  "ok": true,
  "redirect": "/",
  "role": "admin"
}
```

- **需要额外步骤**（如首次登录强制改密码）：原样返回 OneID 响应（含 `next.type`）

- **Cookie 行为**：
  - 返回 OneID 的 `state_token` Set-Cookie（domain 转为租户域名）
  - 登录成功时额外设置 `hatchery-session` cookie

---

### `POST /oneid/password-reset`

首次登录强制重置密码（登录流程中 `next.type=ACCOUNT_RESET_PASSWORD` 时使用）。

- **认证**：需要 `state_token` cookie（登录流程中返回）
- **请求体**：原样转发给 OneID（JSON，含 credential）
- **成功后行为**：清除本地 hatchery session + 清除 `state_token` cookie

---

### `POST /oneid/password-verify`

已登录用户修改密码前验证当前密码。成功后 OneID 返回新的 `state_token`。

- **认证**：需要 `session_token` cookie
- **请求体**：原样转发给 OneID（JSON，含 credential）
- **响应**：原样返回 OneID 响应 + 转域后的 `state_token` Set-Cookie

---

### `POST /oneid/password-change`

已登录用户修改密码（先调 password:verify 获取 state_token 后使用）。

- **认证**：需要 `state_token` + `session_token` cookie
- **请求体**：原样转发给 OneID（JSON，含 credential）
- **成功后行为**：清除本地 hatchery session + 清除 `state_token` cookie

---

## 租户初始化（统一账号扩展）

### `POST /tenants/init`

创建新租户。统一账号模式需额外传入 OneID 自建应用凭证。

- **认证**：AdminToken
- **请求体（JSON）**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| identifier | string | 是 | 租户标识 |
| domains | []string | 是 | 绑定域名列表 |
| primary_domain | string | 是 | 主域名（带协议） |
| uin | string | 否 | 腾讯云 UIN |
| init_user | string | 否 | 初始管理员用户名 |
| init_pass | string | 否 | 初始管理员密码 |
| internal_secret | string | 否 | Gateway 内部鉴权密钥 |
| oneid_account_id | string | 否 | OneID 企业 ID |
| oneid_app_id | string | 否 | OneID 自建应用 ID（传入即开启统一账号模式） |
| oneid_client_id | string | 否 | OneID 自建应用 client_id |
| oneid_client_secret | string | 否 | OneID 自建应用 client_secret |
| oneid_token_endpoint | string | 否 | OneID Token 获取端点 URL |
| secret_id | string | 否 | CVM SecretId |
| secret_key | string | 否 | CVM SecretKey |
| agent_cam_role_secret_id | string | 否 | CAM Role SecretId |
| agent_cam_role_secret_key | string | 否 | CAM Role SecretKey |
| default_lang | string | 否 | "zh" 或 "en"，如果为不设置则默认为 "zh" |
| security_policies | []string | 否 | 当前仅支持 "SSRF"，如果为不设置则默认不启用任何安全策略 | 

- **统一账号模式开通**：传入 `oneid_app_id` 非空即开启统一账号功能，此时 `oneid_client_id`、`oneid_client_secret`、`oneid_token_endpoint` 也应一并传入。

---

## Agent-Bridge 回调接口

Agent-Bridge 是运行在 CVM 实例上的代理服务，通过 TAT（腾讯云自动化助手）执行脚本后，异步回调 Hatchery 写入审计日志。

**认证方式：** 与其他 Agent-Bridge 回调接口一致，通过 `WithOpenAPI` + `resolveAgentBridgeIdentity` 进行 Bearer Token 鉴权，支持以下认证方式：
- `hk-` 用户 API Token
- `sk-` 实例代理 Token（ProxyToken，绑定特定实例）
- AdminToken
- Cookie Session

### `POST /agent-bridge/audit`

Agent-Bridge 在通过 TAT 执行脚本后，异步回调此接口将审计信息写入 Hatchery 的 `audit_logs` 表，使管理员可以在统一的审计日志页面（`GET /admin/audit`）查看所有 TAT 操作。

- **认证：** Bearer Token（hk- / sk- / AdminToken / Session）
- **请求体大小限制：** 4KB
- **Content-Type：** `application/json`

**请求头：**

```
Authorization: Bearer sk-xxx
```

**请求体（JSON）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| platform_id | string | 否 | 平台标识（如 `hatchery`），用于日志追踪 |
| user_id | string | 否 | 用户 ID（仅用于日志记录，实际 user_id 从 Token 中获取，不可伪造） |
| action | string | 是 | 操作类型，**必须以 `agent_bridge_` 前缀开头**（如 `agent_bridge_desktop_install`） |
| resource | string | 否 | 资源类型（如 `instance`） |
| resource_id | string | 否 | 资源标识（通常为 CVM 实例 ID，如 `ins-xxx`） |
| invocation_id | string | 否 | TAT invocation ID，用于关联 TAT 执行记录 |
| script_name | string | 否 | 执行的脚本名称（用于详细追踪） |
| status | string | 是 | 执行结果，必须为以下值之一：`success`、`failed`、`timeout`、`dispatched` |
| trace_id | string | 否 | 调用链追踪 ID |
| started_at | int64 | 否 | 操作开始时间（Unix 秒时间戳）。为 0 或不传时使用当前时间；超出合理范围（早于 24 小时前或晚于当前时间 + 5 分钟）时静默修正为当前时间 |

**请求示例：**

```json
{
  "platform_id": "hatchery",
  "action": "agent_bridge_desktop_install",
  "resource": "instance",
  "resource_id": "ins-xxx",
  "invocation_id": "inv-xxx",
  "script_name": "install_browser_vnc.sh",
  "status": "success",
  "trace_id": "trace-xxx",
  "started_at": 1716192000
}
```

**成功响应：**

```json
{"ok": true}
```

**错误响应：**

| HTTP 状态码 | 错误信息 | 说明 |
|------------|---------|------|
| 400 | `bad request` | 请求体解析失败或超过 4KB 限制 |
| 400 | `action is required` | 缺少 action 字段 |
| 400 | `status is required` | 缺少 status 字段 |
| 400 | `action must start with 'agent_bridge_' prefix` | action 不以 `agent_bridge_` 开头 |
| 400 | `status must be one of: success, failed, timeout, dispatched` | status 值不合法 |
| 403 | `resource_id mismatch with bound instance` | sk- 模式下 resource_id 与绑定实例不一致 |
| 405 | `method not allowed` | 非 POST 方法 |

**安全校验规则：**

1. **action 前缀强制**：action 必须以 `agent_bridge_` 开头，防止伪造其他类型的审计记录
2. **status 白名单**：仅允许 `success`、`failed`、`timeout`、`dispatched` 四种状态
3. **sk- 实例绑定校验**：使用 sk-（ProxyToken）认证时，若请求中携带 resource_id，必须与 Token 绑定的实例 ID 一致
4. **用户身份不可伪造**：user_id 和 username 从 Token 中获取，请求体中的 user_id 仅用于日志记录
5. **请求体大小限制**：最大 4KB，防止 OOM 攻击

**写入行为：**

- 审计日志异步写入数据库（与 `WithAudit` 中间件行为一致）
- `invocation_id`、`script_name`、`trace_id` 通过结构化日志（slog）记录，不额外持久化到 `audit_logs` 表，保持表结构不变
- 写入后可通过 `GET /admin/audit?action=agent_bridge_` 筛选查看所有 Agent-Bridge 审计记录

## 本地 agent reporter 接入（clawpro 一期）

> 本节是 clawpro 接管用户**本地 agent**（WorkBuddy / CodeBuddy 等）的对接接口。reporter 是嵌在用户本机 agent 进程里的轻量模块，由 agent 的 hook 触发（不跑定时任务），通过用户的 hk- API Token 调用以下三个接口与 hatchery 同步状态。
>
> 设计文档：[iwiki 4022150701](https://iwiki.woa.com/p/4022150701)

**触发模型（hook 驱动，不跑定时任务）：**

- 用户本地 agent 提供各自的 hook 点，reporter 在这些 hook 里发起 HTTP 调用：
  - **新建会话 hook**（如 `OnSessionCreate`）：先调一次 `report`，再调一次 `sync`
  - **用户发送消息 hook**（如 `OnUserMessage`）：只调一次 `sync`
- 本地不跑 cron / 心跳 / 轮询；只要用户从不使用 agent，reporter 就不产生流量

**鉴权：**

- 三个接口均为开放 API，仅支持 hk- 用户 API Token（`Authorization: Bearer hk-xxx`）
- 同一 hk- Token 通过 `(user_id, instance_id)` 复合键做多租户隔离

**当前白名单：** `agent_type ∈ {workbuddy, codebuddy, claude}`。其他取值返回 400。

### `POST /local-agent/report` 🆕

agent「新建会话」hook 触发。上报本地实例元数据、`user_level`（用户级资源）和 `workspaces`（项目级资源）；资源只按 `user` / `workspace` scope 对齐。

- **权限：** 登录用户（hk- API Token）
- **Content-Type：** `application/json`
- **请求体：** `application/json`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| local_agent_id | string | 是 | reporter 在本机派生的 16 位 hex（不含敏感主机信息） |
| agent_type | string | 是 | agent 类型，一期仅 `workbuddy` / `codebuddy` |
| agent_version | string | 是 | agent 自身版本号 |
| host_name | string | 是 | 主机名（如 `alex-mbp`），用作实例默认名 |
| os | string | 是 | `<goos>/<goarch>` 格式，如 `darwin/arm64` / `linux/amd64` / `windows/amd64` |
| started_at | string | 否 | agent 进程启动时间（RFC3339） |
| last_status | string | 否 | 运行状态：`running` / `error` |
| user_level | object | 否 | 二期：用户级资源（scope=user）。不传则不处理 |
| user_level.group_id | uint | 否 | 服务端维护的当前主分组；TeamAI 无需传入，即使传入也不作为切换依据。report 会按服务端用户关系检测主组织被动变化 |
| user_level.skills | array | 否 | 用户级已装 skill 列表 |
| workspaces | array | 否 | 二期：项目级 workspace 列表（scope=workspace） |
| workspaces[].path | string | 是 | workspace 唯一标识（路径） |
| workspaces[].name | string | 否 | workspace 名称 |
| workspaces[].ide_type | string | 否 | IDE 类型：`codebuddy` / `workbuddy` / `claude_code` / `codex` |
| workspaces[].project_id | uint | 否 | workspace 绑定的项目 ID；省略时保持现有绑定，传 `0` 清空绑定 |
| workspaces[].skills | array | 否 | 项目级已装 skill 列表 |

> ⚠️ 请求体中**没有 `install_path`、`machine_id`** 等本机敏感字段；hatchery 不感知 reporter 的本地路径与机器标识。

**示例 Request：**

```json
POST /local-agent/report
Authorization: Bearer hk-xxx
Content-Type: application/json

{
  "local_agent_id": "a1b2c3d4e5f6a7b8",
  "agent_type": "workbuddy",
  "agent_version": "1.2.3",
  "host_name": "alex-mbp",
  "os": "darwin/arm64",
  "started_at": "2026-06-16T10:00:00Z",
  "user_level": {
    "skills": [
      {"slug": "user-skill", "version": "1.0.0", "display_name": "User Skill", "source": "enterprise"}
    ]
  },
  "workspaces": [
    {
      "path": "/Users/alex/code/repo-x",
      "name": "repo-x",
      "ide_type": "codebuddy",
      "project_id": 2,
      "skills": [
        {"slug": "project-skill", "version": "1.1", "display_name": "Project Skill", "source": "enterprise"}
      ]
    }
  ]
}
```

- **响应：** `application/json`

**成功响应：**

```json
{
  "ok": true,
  "instance_id": "local-workbuddy-6a7b8",
  "instance_pk": 1,
  "received_at": "2026-07-05T10:00:01+08:00",
  "user_level_synced": 1,
  "project_synced": 1
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| instance_id | string | 派生出的 hatchery 实例 ID（形如 `local-workbuddy-6a7b8`） |
| instance_pk | uint | 实例主键 ID |
| received_at | string | 接收时间（RFC3339） |
| rules_synced | int | 用户级与项目级规范同步数量 |
| user_level_synced | int | 二期：用户级 skill 同步数量 |
| project_synced | int | 二期：项目级 workspace 同步数量 |

- **错误响应：**
  - `400 {"error": "<参数缺失或格式错误>"}` — `local_agent_id` 不是 16 位 hex / `agent_type` 不在白名单 / 必填字段缺失等
  - `401 {"error": "未登录"}` — Token 无效
  - `500 {"error": "服务器内部错误"}`

### `POST /local-agent/sync` 🆕

agent「新建会话」hook（紧跟 `report` 后）/「用户发送消息」hook 触发。拉取待执行的 commands（pending + failed records）+ 同步刷新 `last_report_at`。二期新增 `workspaces` 上报和 commands 的 `scope` / `workspace_path` 字段。

- **权限：** 登录用户（hk- API Token）
- **Content-Type：** `application/json`
- **请求体：** `application/json`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| local_agent_id | string | 是 | reporter 派生的 16 位 hex（与 report 一致） |
| agent_type | string | 是 | agent 类型，与 report 一致（用于派生 `instance_id`） |
| status | string | 是 | 上报时的运行状态：`running` / `error` |
| workspaces | array | 否 | 二期：agent 上报当前 workspace 列表 |
| workspaces[].path | string | 是 | workspace 唯一标识（路径） |
| workspaces[].name | string | 否 | workspace 名称 |
| workspaces[].ide_type | string | 否 | IDE 类型 |
| workspaces[].project_id | uint | 否 | workspace 绑定的项目 ID；省略时保持现有绑定，传 `0` 清空绑定 |

**示例 Request：**

```json
POST /local-agent/sync
Authorization: Bearer hk-xxx
Content-Type: application/json

{
  "local_agent_id": "a1b2c3d4e5f6a7b8",
  "agent_type": "workbuddy",
  "status": "running",
  "workspaces": [
    {
      "path": "/Users/alex/code/repo-x",
      "name": "repo-x",
      "ide_type": "codebuddy",
      "project_id": 2
    }
  ]
}
```

- **响应：** `application/json`

**成功响应：**

```json
{
  "ok": true,
  "commands": [
    {
      "id": 582,
      "type": "install_skill",
      "skill_slug": "weather",
      "skill_version": "1.2.3",
      "download_url": "https://smh.../weather-1.2.3.zip",
      "scope": "user"
    },
    {
      "id": 583,
      "type": "install_skill",
      "skill_slug": "project-skill",
      "skill_version": "1.1",
      "download_url": "https://smh.../project-skill-1.1.zip",
      "scope": "workspace",
      "workspace_path": "/Users/alex/code/repo-x",
      "project_id": 2
    },
    {
      "id": 303,
      "type": "install_hook_rule",
      "rule_slug": "hook-abc",
      "rule_version": "1.0.0",
      "rule_type": "hook",
      "event": "SessionStart",
      "cmd": "echo hello",
      "scope": "user"
    },
    {
      "id": 456,
      "type": "uninstall_teamai",
      "cmd": "teamai uninstall --force --agent codebuddy"
    }
  ],
  "cmds": [
    {
      "id": 582,
      "type": "install_skill",
      "slug": "weather",
      "version": "1.2.3",
      "download_url": "https://smh.../weather-1.2.3.zip",
      "scope": "user"
    },
    {
      "id": 583,
      "type": "install_skill",
      "slug": "project-skill",
      "version": "1.1",
      "download_url": "https://smh.../project-skill-1.1.zip",
      "scope": "workspace",
      "workspace_path": "/Users/alex/code/repo-x",
      "project_id": 2
    },
    {
      "id": 303,
      "type": "install_hook_rule",
      "slug": "hook-abc",
      "version": "1.0.0",
      "handle_type": "hook",
      "event": "SessionStart",
      "cmd": "echo hello",
      "scope": "user"
    },
    {
      "id": 456,
      "type": "uninstall_teamai",
      "cmd": "teamai uninstall --force --agent codebuddy"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| commands | array | 待执行命令列表（老版本协议，异构数组，skill 字段前缀 `skill_`、rule 字段前缀 `rule_`）。reporter 按顺序执行后 ack |
| cmds | array | 🆕 三期：统一字段列表，所有类型共享 `slug` / `version` / `handle_type` / `event` / `cmd`（按需要 omitempty），与 `commands` 数据完全一致。新版本 reporter 读取此列表 |
| {commands,cmds}[].id | int | 命令 ID（skill/rule = `skill_distribution_records.id` / `rule_distribution_records.id`；卸载插件 = `local_agent_tasks.id`），ack 时回传 |
| {commands,cmds}[].type | string | 命令类型：`install_skill` / `uninstall_skill` / `install_prompt_rule` / `install_rule_rule` / `uninstall_prompt_rule` / `uninstall_rule_rule` / `install_hook_rule` / `uninstall_hook_rule` / `uninstall_teamai` |
| {commands,cmds}[].scope | string | 二期：`user` / `workspace`；不会返回已废弃的 `instance`，无法关联二期 scope 的历史实例任务按 `user` 兼容 |
| {commands,cmds}[].workspace_path | string | 二期：仅 `scope=workspace` 时有，标识目标 workspace |
| {commands,cmds}[].project_id | uint | 二期：仅 `scope=workspace` 时有，命令所属项目 ID |
| commands[].skill_slug / commands[].skill_version | string | 目标 skill 的 slug / 版本（`install_skill` / `uninstall_skill`） |
| commands[].rule_slug / commands[].rule_version / commands[].rule_type | string | 目标 rule 的 slug / 版本 / 处理类型 prompt/rule/hook（`install_*_rule` / `uninstall_*_rule`） |
| cmds[].slug / cmds[].version | string | 统一：目标资产 slug / 版本（skill / rule / hook 共用；`uninstall_teamai` 无） |
| cmds[].handle_type | string | 统一：处理类型 `prompt` / `rule` / `hook`（skill 命令不返回或按场景） |
| cmds[].event | string | 🆕 Hook 专属：触发时机 `SessionStart` / `UserPromptSubmit` / `PreToolUse` / `PostToolUse` / `Stop`（`install_hook_rule` 返回） |
| cmds[].cmd | string | 🆕 执行命令：Hook 的 `install_hook_rule` 返回管理员填写的命令；`uninstall_teamai` 返回 `teamai uninstall --force --agent <agent_type>` |
| {commands,cmds}[].download_url | string | skill / rule 安装包临时下载 URL（`install_skill` / `install_*_rule` 必填，由 SMH 签发，限时）；Hook 与 `uninstall_teamai` 无此字段 |

> 🆕 三期协议演进（双列表兼容）：`commands[]` 为老版本协议（异构字段名），`cmds[]` 为新版本统一协议。两列表**数据完全相同**，同一份命令同时出现在两个数组。`uninstall_teamai` 命令仅在数组中携带 `id` / `type` / `cmd` 字段，无 slug/version/download_url。`install_hook_rule` 携带 `event` / `cmd` / `slug` / `version` / `handle_type`，无 `download_url`。老 reporter 遇到不认识的 `type`（如 `install_hook_rule` / `uninstall_teamai`）会跳过，不受影响。老 reporter 全量升级后 `commands` 字段移除（届时另行通知）。

> sync 重复拉到同一条 pending/failed 命令是允许的——reporter 端自行幂等。命令的状态机仅由 ack 推动（pending → success/failed），sync 不修改 records 表。

> TeamAI 只需在 Workspace 上报可选 `project_id`。`user_level.group_id` 由 Hatchery 服务端维护：report 时以已保存的分组与服务端当前主组织进行比对，发生被动变化时触发用户级对账；上报的 `user_level.group_id` 不会改变该结果。`workspaces[].group_id` 不属于当前协议。

- **错误响应：**
  - `400 {"error": "<参数缺失或格式错误>"}`
  - `401 {"error": "未登录"}` — Token 无效
  - `500 {"error": "服务器内部错误"}`

### `POST /local-agent/commands/ack` 🆕

reporter 在执行完命令后回写结果。**`id` + `type` 通过 Request Body 传入**，`type` 用于区分 skill/rule 记录（与 sync 返回的 `command.type` 一致）。

- **权限：** 登录用户（hk- API Token）
- **Content-Type：** `application/json`
- **请求体：** `application/json`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 命令 ID（= sync 返回的 `commands[].id` 或 `cmds[].id`） |
| type | string | 否 | 命令类型（= sync 返回的 `commands[].type` / `cmds[].type`）：`install_skill` / `uninstall_skill` / `install_prompt_rule` / `install_rule_rule` / `uninstall_prompt_rule` / `uninstall_rule_rule` / `install_hook_rule` / `uninstall_hook_rule` / `uninstall_teamai`。空串向后兼容按 skill 处理 |
| status | string | 是 | 执行结果：`success` / `failed` |
| error | string | 否 | 失败原因（`status=failed` 时填） |
| version | string | 否 | `success` 时回报的实际安装版本（可选；Hook 可不传） |

> **二期变更**：ack 请求新增 `type` 字段。一期不传 `type` → 空串默认按 skill 处理（向后兼容）。
> **三期变更**：`type` 枚举扩展 `install_prompt_rule` / `install_rule_rule` / `uninstall_prompt_rule` / `uninstall_rule_rule` / `install_hook_rule` / `uninstall_hook_rule` / `uninstall_teamai`（原本的 `install_rule` / `uninstall_rule` 别名仍向后兼容路由到对应 rule 路径）。

**示例 Request（成功）：**

```json
POST /local-agent/commands/ack
Authorization: Bearer hk-xxx
Content-Type: application/json

{
  "id": 582,
  "type": "install_skill",
  "status": "success",
  "version": "1.2.3"
}
```

**示例 Request（失败）：**

```json
POST /local-agent/commands/ack
Authorization: Bearer hk-xxx
Content-Type: application/json

{
  "id": 582,
  "type": "install_skill",
  "status": "failed",
  "error": "下载失败: connection timeout"
}
```

- **响应：** `application/json`

**成功响应：**

```json
{
  "ok": true,
  "record_id": 582,
  "status": "success"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| record_id | uint | 记录 ID |
| status | string | 当前 record 的最终状态：`success` / `failed` |

> **幂等：** 重复 ack 同一条 record（命中 `status != 'pending'` 且 `!= 'failed'`）会直接返回当前状态，不会重复写库。reporter 重试是安全的。

**ack 处理行为：**

- 校验 `type`：
  - `install_skill` / `uninstall_skill` 走 skill 路径
  - `install_prompt_rule` / `install_rule_rule` / `uninstall_prompt_rule` / `uninstall_rule_rule`（含旧别名 `install_rule` / `uninstall_rule`）走 rule 路径
  - `install_hook_rule` / `uninstall_hook_rule` 走 rule 路径（Hook 复用 rule 下发/记录管道，`id` = `rule_distribution_records.id`）
  - `uninstall_teamai` 走本地任务路径（`id` = `local_agent_tasks.id`）
- **skill 路径**：`pending → success`：record.status=success，`local_instance_skills.install_status=distributed`；`pending → failed`：record.status=failed，install_status=failed；`failed → success` 允许重试；终态幂等返回
- **rule / hook 路径**：`pending → success/failed` 更新 `rule_distribution_records` + 实例安装状态；`failed → success` 允许重试；终态幂等返回
- **uninstall_teamai 路径**：
  - `pending → success`：更新 `local_agent_tasks.status=success`，**软删该本地实例**（关联 `local_instance_skills` / `local_instance_rules` 数据保留）；reporter 下次 `report` 时通过 `Unscoped` 查询重新激活该实例（恢复 `deleted_at`）
  - `pending → failed`：更新 `local_agent_tasks.status=failed` + `error`，保留任务记录支持管控端重试
  - `failed → success` 允许重试成功；终态幂等返回

- **错误响应：**
  - `400 {"error": "参数 id 不能为空"}`
  - `400 {"error": "参数 status 无效"}` — 非 success / failed
  - `400 {"error": "参数 type 无效"}` — 非 install_skill / uninstall_skill / install_prompt_rule / install_rule_rule / uninstall_prompt_rule / uninstall_rule_rule / install_hook_rule / uninstall_hook_rule / uninstall_teamai
  - `401 {"error": "未登录"}` — Token 无效
  - `404` — record 不存在或不属于当前用户
  - `405 {"error": "method not allowed"}` — 非 POST
  - `501 {"error": "规范（rule）分发功能本期未实现"}` — type=install_prompt_rule 等

### `POST /local-agent/remove` 🆕（本地 Agent 三期）

用户端移除自己的本地 Agent：创建一个 `uninstall_teamai` 本地任务，**不立即删除实例**，由 reporter 下次 sync 拉取命令后本地执行插件卸载 + 解绑。任务持久化，离线场景下次连接自动执行。

- **权限：** 登录用户（仅能移除自己的 source=local 实例）
- **Content-Type：** `application/json`
- **请求体：** `application/json`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_id | uint | 是 | 目标实例主键（instances.id，「我的 Agent」列表中获取） |

**示例 Request：**

```json
POST /local-agent/remove
Authorization: Bearer hk-xxx
Content-Type: application/json

{ "instance_id": 789 }
```

- **响应：** `application/json`

**成功响应：**

```json
{ "ok": true, "task_id": 789, "status": "pending" }
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| task_id | uint | 创建（或幂等命中已有 pending）的 `local_agent_tasks.id` |
| status | string | 任务状态，创建后为 `pending` |

> **幂等：** 同一实例已存在 pending 的 `uninstall_teamai` 任务时，不重复创建，直接返回已有 `task_id`，重复调用无副作用。

- **错误响应：**
  - `400 {"error": "<参数缺失或非法>"}`
  - `404 {"error": "<实例不存在或无权限>"}` — 实例不存在、非本地实例、或不属于当前用户
  - `403 {"error": "<本地 Agent 未开放>"}` — 白名单/开关未通过

### `POST /admin/local-agent/remove` 🆕（本地 Agent 三期）

管控端移除指定本地 Agent（管理员操作）。创建 `uninstall_teamai` 任务，语义与用户端一致；写接口注册审计（auditRules + WithAudit）。

- **权限：** 管理员（requireAdmin）
- **Content-Type：** `application/json`
- **请求体：** `application/json`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_id | uint | 是 | 目标实例主键（instances.id，管控端 Agent 列表已有） |

**示例 Request：**

```json
POST /admin/local-agent/remove
Authorization: Bearer <admin-token>
Content-Type: application/json

{ "instance_id": 789 }
```

- **响应：** 同用户端（ok / task_id / status）。

- **错误响应：**
  - `400 {"error": "<参数缺失>"}`
  - `404 {"error": "<实例不存在或非本地实例>"}`

> 执行链路：创建任务即把实例 `last_known_status` 写为 `destroying`（前端展示「销毁中」）并标记 `current_operation=uninstall_local_agent` 防重入 → reporter 下次 sync 拉到 `uninstall_teamai` 命令 → 本地执行 `teamai uninstall --force --agent <agent_type>` → ack `success` 后软删实例四表（instances 软删；local_instance_skills / local_instance_rules 硬删；local_instance_infos 软删），关联数据清理干净 → ack `failed` 保留任务记录，实例退出卸载中（`last_known_status` 恢复 `running`、`current_operation` 清空），管控端可重试 重新 report 同一本地 Agent 时通过 `Unscoped` 查询重新激活已软删实例。

### `GET /local-agent/availability` 🆕

普通用户视角查询「本地 Agent 是否可用」。前端在「+ Agent」按钮、引导页等位置据此决定是否展示本地 agent 入口。

与 reporter 三接口（`/local-agent/report` / `sync` / `commands/ack`）的双层守卫共享判定公式：

```
enabled = feature_allowlist 通过 AND SiteConfig.local_agent_enabled
```

下期会在公式末尾叠加分组策略 `local_agent`（用 AND）。**响应 schema 永远是 200 + 单一 `enabled` 布尔**——前端无需感知公式扩展。

- **权限：** 登录用户
- **入参：** 无
- **响应：** `200 application/json`

```json
{
  "enabled": true
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| enabled | bool | true=可用；false=任一守卫层未通过 |

**为什么不复用 reporter 拒绝路径**：reporter 接口未通过守卫时返 403 + 错文案——给前端的可选性查询用不上 403，需要单独接口 200 返 bool 让前端逻辑简单。

**与 `GET /admin/feature-allowlist/check` 的差别**：

| 维度 | `/local-agent/availability`（用户视角） | `/admin/feature-allowlist/check`（管控视角） |
|---|---|---|
| 鉴权 | 登录用户 | 管理员 |
| 决策因子 | feature_allowlist + SiteConfig（一期）；下期叠加分组策略 | 仅 feature_allowlist 一层 |
| 返字段 | `enabled` 单一 bool | `in_allowlist`（含 `empty_table_means_allow` 短路） |
| 用途 | 前端是否展示本地 Agent 入口 | 管控页诊断「该租户在白名单里吗」 |

- **错误响应：**
  - `401 {"error": "未登录"}`
  - `405 {"error": "method not allowed"}` — 非 GET

### `POST /openclaw/local/user-group` 🆕

前端切换用户级分组。切换后 hatchery 算差集（catalog − installed），写 pending record，等本地 agent 主动 sync 拉走。

- **权限：** 登录用户
- **Content-Type：** `application/json`
- **请求体：** `application/json`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_id | uint | 是 | 切换到的新分组 ID |
| instance_id | uint | 是 | 实例主键 ID |

**示例 Request：**

```json
POST /openclaw/local/user-group
Content-Type: application/json

{
  "group_id": 2,
  "instance_id": 1
}
```

- **响应：** `application/json`

**成功响应：**

```json
{
  "ok": true,
  "new_pending_count": 3
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 固定 `true` |
| new_pending_count | int | 本次切换新增的 pending 任务数 |

> **行为：** 清理旧分组 failed/distributing 记录 → 算新分组 catalog 差集 → 写 pending record + local_instance_skills(install_status=distributing)

- **错误响应：**
  - `400 {"error": "参数 group_id 不能为空"}`
  - `400 {"error": "分组不存在"}` — group_id 不属于当前用户
  - `401 {"error": "未登录"}`
  - `404 {"error": "实例不存在或无权访问"}`
  - `405 {"error": "method not allowed"}` — 非 POST

### `GET /openclaw/list?id=N`（EXTEND — 精准查询返回 local_agent_resources）

精准查询时（带 `id` 参数），返回 `local_agent_resources` 字段，包含用户级分组和 Workspace 项目绑定 + 已装 skill 状态。仅 `source=local` 的实例返回此字段。

> 非精准查询（列表）时不返回 `local_agent_resources`（性能考虑）。

**新增返回字段（在 instances[] 元素内）：**

```json
{
  "local_agent_resources": {
    "user_level": {
      "group_id": 1,
      "group_name": "CVM团队",
      "group_active": true,
      "skills": [
        {
          "slug": "skill-a",
          "version": "1.0.0",
          "display_name": "Skill A",
          "source": "enterprise",
          "install_status": "distributed"
        },
        {
          "slug": "skill-b",
          "version": "2.0.0",
          "display_name": "Skill B",
          "source": "enterprise",
          "install_status": "distributing"
        }
      ]
    },
    "workspaces": [
      {
        "path": "/Users/alice/code/repo-x",
        "name": "repo-x",
        "ide_type": "codebuddy",
        "project_id": 2,
        "project_name": "运营项目",
        "project_exists": true,
        "skills": [
          {
            "slug": "project-skill",
            "version": "1.1",
            "display_name": "Project Skill",
            "source": "enterprise",
            "install_status": "distributed"
          }
## 存量实例分组归属处理（stale-instances v1.0）

当用户分组发生变更（编辑分组 / 添加未分组用户 / 修改分组父级 / OneID 同步）后，部分 Agent 实例的 `group_id` 与所有人当前分组归属不一致。本组接口提供「拉差异 → 决策处理 → 跟踪记录」的完整流程。前端使用现有 `POST /admin/instances/by-user-group` 拉受影响实例列表（已放开 `group_id=0` 过滤，可查询某用户的未分组实例）。

### `POST /admin/stale-instances/action-options` — 操作选项

给定受影响的用户-分组对或分组列表，返回 3 个 UI 弹窗场景中包含哪些实例、以及支持哪些操作选项。
入参格式与 `POST /admin/instances/by-user-group` 相同；`user_group_ids` 和 `group_ids` 均为可选，不传时 `no_group` 场景仍会全量扫描。
**已打 `stale_group` 标记的实例会被过滤掉**，避免重复处理（如 OneID 同步已自动标记的实例不会再次出现在操作选项中）。

场景说明：
- **`no_group`**（全量扫描，不依赖请求参数）：系统内所有 `user_id` 当前有分组成员关系、但实例 `group_id=0` 的实例 —— 不支持移交
- **`user_removed`**（来自 `user_group_ids`）：精确匹配 `(user_id, group_id)` 对的实例 —— 移交是否可用取决于该分组当前是否有其他成员
- **`subtree`**（来自 `group_ids`）：`group_ids` 子树展开后的成员实例，按 `group_id` 聚合 —— 仅支持迁移

- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_group_ids | array | 否 | `[{"user_id":N,"group_id":N},...]` 用户-分组对列表；为空时 user_removed 场景返回空数组 |
| group_ids | uint[] | 否 | 根分组 ID 列表，后端展开完整子树后查询成员实例；为空时 subtree 场景返回空数组 |

- **响应：**

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 是否成功 |
| no_group | object | 全量扫描 — 用户当前有分组成员关系、但实例 `group_id=0` 的全系统实例，按 user_id 聚合 |
| no_group.options | array | 按 user_id 聚合的操作项列表，顺序按行首次出现顺序 |
| no_group.options[].user_id | uint | 用户 ID |
| no_group.options[].username | string | 用户名 |
| no_group.options[].user_groups | array | 该用户当前所属的分组列表（group_id + group_full_path） |
| no_group.options[].user_groups[].group_id | uint | 用户当前所属分组 ID |
| no_group.options[].user_groups[].group_full_path | string | 用户当前所属分组全路径 |
| no_group.options[].instances | array | 该用户归入 no_group 的实例列表 |
| no_group.options[].instances[].id | uint | 实例主键 ID |
| no_group.options[].instances[].instance_id | string | CVM 实例 ID（如 `ins-db1zmpsy`） |
| no_group.options[].instances[].name | string | 实例名 |
| no_group.options[].instances[].user_id | uint | 所有者 user_id |
| no_group.options[].instances[].username | string | 所有者用户名 |
| no_group.options[].migrate | bool | 是否支持「随用户迁移到新分组」（固定 `true`） |
| no_group.options[].handover | bool | 是否支持「移交给其他用户」（固定 `false`） |
| no_group.options[].pending_user | bool | 是否支持「允许用户自行处理」（固定 `true`） |
| no_group.options[].pending_user_allow_migrate | bool | 用户自行处理中：是否允许随用户迁移（固定 `true`） |
| no_group.options[].pending_user_allow_handover | bool | 用户自行处理中：是否允许移交（固定 `false`） |
| no_group.options[].archive_stop | bool | 是否支持「关机」（固定 `true`） |
| user_removed | object | 用户被移出某分组，实例仍持有 `group_id`，按 user_id 聚合，再按 agent 的 group_id 二级聚合 |
| user_removed.options | array | 按 user_id 聚合的操作项列表 |
| user_removed.options[].user_id | uint | 用户 ID |
| user_removed.options[].username | string | 用户名 |
| user_removed.options[].user_groups | array | 该用户当前所属的分组列表（group_id + group_full_path） |
| user_removed.options[].user_groups[].group_id | uint | 用户当前所属分组 ID |
| user_removed.options[].user_groups[].group_full_path | string | 用户当前所属分组全路径 |
| user_removed.options[].groups | array | 按 agent 的 group_id 二级聚合的分组列表 |
| user_removed.options[].groups[].group_id | uint | 实例当前 group_id |
| user_removed.options[].groups[].group_full_path | string | 实例当前分组全路径 |
| user_removed.options[].groups[].instances | array | 该分组下的实例列表 |
| user_removed.options[].groups[].instances[].id | uint | 实例主键 ID |
| user_removed.options[].groups[].instances[].instance_id | string | CVM 实例 ID（如 `ins-db1zmpsy`） |
| user_removed.options[].groups[].instances[].name | string | 实例名 |
| user_removed.options[].groups[].instances[].user_id | uint | 所有者 user_id |
| user_removed.options[].groups[].instances[].username | string | 所有者用户名 |
| user_removed.options[].groups[].handover_available | bool | 该分组当前是否有其他成员（per-group，同组实例共享） |
| user_removed.options[].migrate | bool | 是否支持「随用户迁移到新分组」（固定 `true`） |
| user_removed.options[].handover | bool | 是否支持「移交给其他用户」；`OR(groups[].handover_available)` |
| user_removed.options[].pending_user | bool | 是否支持「允许用户自行处理」（固定 `true`） |
| user_removed.options[].pending_user_allow_migrate | bool | 用户自行处理中：是否允许随用户迁移（固定 `true`） |
| user_removed.options[].pending_user_allow_handover | bool | 用户自行处理中：是否允许移交；等于 `handover` |
| user_removed.options[].archive_stop | bool | 是否支持「关机」（固定 `true`） |
| subtree | object | 分组父节点变更，按 group_id 聚合 |
| subtree.groups | array | 按 group_id 聚合的分组列表，顺序与请求 ID 首次出现顺序一致 |
| subtree.groups[].group_id | uint | 分组 ID |
| subtree.groups[].group_full_path | string | 分组路径 |
| subtree.groups[].instances | array | 该分组下的实例列表（字段同 no_group 实例：`id` / `instance_id` / `name` / `user_id` / `username`） |

> `subtree` 无 `options` 字段，前端根据产品规格直接展示「迁移到新分组路径」操作。

- **响应示例：**

```json
// 请求体示例
{
  "user_group_ids": [{"user_id": 8, "group_id": 42}],
  "group_ids": [99]
}

// 响应
{
  "ok": true,
  "no_group": {
    "options": [
      {
        "user_id": 5, "username": "alice",
        "user_groups": [
          {"group_id": 3, "group_full_path": "产品组/产品一组"}
        ],
        "instances": [
          {"id": 101, "instance_id": "ins-db1zmpsy", "name": "my-agent", "user_id": 5, "username": "alice"}
        ],
        "migrate": true, "handover": false, "pending_user": true,
        "pending_user_allow_migrate": true, "pending_user_allow_handover": false,
        "archive_stop": true
      }
    ]
  },
  "user_removed": {
    "options": [
      {
        "user_id": 8, "username": "bob",
        "user_groups": [
          {"group_id": 3, "group_full_path": "产品组/产品一组"}
        ],
        "groups": [
          {
            "group_id": 42, "group_full_path": "研发部/AI 小组",
            "instances": [
              {"id": 202, "instance_id": "ins-abc12345", "name": "team-agent", "user_id": 8, "username": "bob"}
            ],
            "handover_available": true
          },
          {
            "group_id": 55, "group_full_path": "研发部/基础架构组",
            "instances": [
              {"id": 203, "instance_id": "ins-def67890", "name": "infra-agent", "user_id": 8, "username": "bob"}
            ],
            "handover_available": false
          }
        ],
        "migrate": true, "handover": true, "pending_user": true,
        "pending_user_allow_migrate": true, "pending_user_allow_handover": true,
        "archive_stop": true
      }
    ]
  },
  "subtree": {
    "groups": [
      {
        "group_id": 99, "group_full_path": "新部门/子组",
        "instances": [
          {"id": 303, "instance_id": "ins-xyz99999", "name": "infra-agent", "user_id": 12, "username": "carol"}
        ]
      }
    ]
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| local_agent_resources | object \| null | 本地 agent 资源信息（仅 source=local 且精准查询时返回） |
| local_agent_resources.user_level | object | 用户级资源绑定 |
| user_level.group_id | uint | 服务端维护的用户级主分组 ID |
| user_level.group_name | string | 分组名称 |
| user_level.group_active | bool | 分组是否有效（用户是否仍在该分组内） |
| user_level.skills | array | 用户级已装 skill 列表 |
| skills[].slug | string | skill 唯一标识 |
| skills[].version | string | skill 版本 |
| skills[].display_name | string | 展示名 |
| skills[].source | string | 来源：`enterprise` / `public` / `local` |
| skills[].install_status | string | 安装状态：`distributing`（下发中）/ `distributed`（已下发）/ `failed`（下发失败） |
| user_level.rules | array | 用户级已装规范列表（scope=user） |
| rules[].slug | string | 规范唯一标识 |
| rules[].version | string | 规范版本 |
| rules[].display_name | string | 展示名 |
| rules[].type | string | 规范类型：`prompt` / `rule` |
| rules[].source | string | 来源：`enterprise` / `local` |
| local_agent_resources.workspaces | array | 项目级 workspace 列表 |
| workspaces[].path | string | workspace 唯一标识（路径） |
| workspaces[].name | string | workspace 名称 |
| workspaces[].ide_type | string | IDE 类型 |
| workspaces[].project_id | uint | workspace 绑定的项目 ID；删除项目后仍保留原 ID |
| workspaces[].project_name | string | 项目名称；项目已删除时为空字符串 |
| workspaces[].project_exists | bool | 项目是否仍存在 |
| workspaces[].skills | array | 项目级已装 skill 列表（同 user_level.skills 格式） |
| workspaces[].rules | array | 项目级已装规范列表（scope=workspace） |

**install_status 展示规则：**

| 状态 | 含义 | 前端展示 |
|------|------|---------|
| `distributing` | 下发中（pending record 已写，agent 未 ack） | 蓝色 spinner + "下发中" |
| `distributed` | 已下发（agent ack 成功 / report 上报已装） | 正常 Chip |
| `failed` | 下发失败（agent ack 失败） | 红色边框 + "下发失败" |

**项目删除兼容：**

- Workspace 保留原 `project_id`，但 `project_name` 为空且 `project_exists=false`
- 已下发资源仍可查看；后续 report/sync 不再从已删除项目读取、对账或下发资产
### `POST /admin/stale-instances/config-diff` — 配置差异

对比「实例自身存储的配置」 vs 「目标分组配置集合」。响应里 target 配置和 instance 配置**分离**：
顶层 `target_config.categories[]` 给出目标组的所有行（**仅一份，所有实例共享**），
`instance_configs[].categories[]` 给出每个实例独立的左侧值 + 集合关系 status。
两者通过共享的 `key` 字段一一对应（前端按 `key` join）。

**左侧 `instance_values` 语义**：实例本身存储的配置（实例级覆盖），不回退到当前组。
仅以下 category 在实例上存有字段，其他 category 一律返回空集合：

| Category / SubLabel | Instance 字段 | 空/零值时 |
|------|------|------|
| `model` | `AIModelID`（uint） | `==0` → 空集 |
| `skill`（`sub_label=""`） | `RoleID`（uint） | `==0` → 空集 |
| `imageType` | `AgentType`（string） | 空字符串 → 空集（实例正常情况下必有） |
| `network`（`sub_label="私有网络与子网"`） | `VpcId` + `SubnetId` | 空字符串显示为"自动分配"（与组视图一致） |
| `network`（`sub_label="安全组"`） | `SecurityGroupId` | 空字符串 → 空集 |
| `network`（`sub_label="公网"`） | — | 实例无对应字段，固定为空集 |
| 其他 category（`channel` / `agentTool` / `memory` / `drive` / `cls` / `aiAgentSecurity` / `platformPolicy`） | — | 实例无对应字段，固定为空集 |

- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_ids | uint[] | 是 | 待对比的实例 ID 列表，单次最多 100 条；超过返回 400 |
| target_group_id | uint | 是 | 目标分组 ID；`0` 表示未分组（全局默认）。未传字段 → 400 |

- **响应：**

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 是否成功 |
| target_group_id | uint | 同入参 |
| target_group_path | string | 目标分组的 full_path |
| target_config | object | 目标分组配置视图（所有实例共享一份） |
| target_config.categories | array | 行集合，按 ConfigCategoryList 顺序铺开 |
| target_config.categories[].key | string | 行唯一 key（如 `model` / `skill.0` / `platformPolicy.instance_quota`） |
| target_config.categories[].category_key | string | 所属 category |
| target_config.categories[].category_label | string | category 显示名 |
| target_config.categories[].sub_label | string | 子标签（按 SubLabel 拆分时存在） |
| target_config.categories[].values | array | 该行目标分组取值集合 `[{id, name, value}]`；`name` 仅公网子项（`name` = 子项标签，如 "公网 IP"）和平台策略条目非空，其余为空串 |
| instance_configs | array | 每个被查询 instance 一条；顺序与请求 instance_ids 一致 |
| instance_configs[].instance_id | uint | 实例 ID |
| instance_configs[].current_group_id | uint | 实例当前 group_id |
| instance_configs[].current_group_path | string | 当前 group 的 full_path |
| instance_configs[].categories | array | 行集合，row keys 与 `target_config.categories` 一一对应 |
| instance_configs[].categories[].key | string | 行 key（与 target_config 同 key 表示同一行） |
| instance_configs[].categories[].category_key | string | 所属 category |
| instance_configs[].categories[].category_label | string | category 显示名 |
| instance_configs[].categories[].sub_label | string | 子标签 |
| instance_configs[].categories[].instance_values | array | 实例自身存储的值 `[{id, name, value, status}]`（按上表语义）；`name` 仅公网子项和平台策略条目非空 |
| instance_configs[].categories[].instance_values[].status | string | 单条值与 target 的对比状态：`same` / `different` / `contained_in_target` / `not_check`（逻辑同行级 status，`different` 行下逐值判断是否出现在 target_values 中） |
| instance_configs[].categories[].status | string | `same`：集合相等；`contained_in_target`：实例集合 ⊊ 目标集合；`different`：实例有项不在目标内 |
| not_found_instance_ids | uint[] | 未找到的实例 ID 列表（不存在的实例不会静默丢弃） |

- **行展示策略**：
    - `single_row`：整个 category 一行（model / channel / memory / drive / imageType / cls / aiAgentSecurity）
    - `by_sub_label`：按 entry.SubLabel 拆多行（skill 拆 role（sub_label=""）/ skillhub；agentTool 拆 plugin/mcp；network 拆 vpc_subnet/security_group/internet）；"初始技能包"和"企业技能"行已过滤，不出现在输出中
    - `by_entry`：每个 entry 一行（platformPolicy，每条策略独立配置项）

- **响应示例（节选）**：

```json
{
  "ok": true,
  "target_group_id": 193,
  "target_group_path": "AI 实验室",
  "target_config": {
    "categories": [
      {
        "key": "model", "category_key": "model", "category_label": "模型",
        "values": [
          {"id": "1", "name": "", "value": "混元 TurboS Latest"},
          {"id": "5", "name": "", "value": "Claude Sonnet 4"},
          {"id": "12", "name": "", "value": "DeepSeek V3"}
        ]
      },
      {
        "key": "skill.0", "category_key": "skill", "category_label": "技能", "sub_label": "",
        "values": [{"id": "1", "name": "", "value": "通用助手"}, {"id": "3", "name": "", "value": "AI 研究员"}]
      },
      {
        "key": "platformPolicy.instance_quota", "category_key": "platformPolicy", "category_label": "平台策略", "sub_label": "单用户 Agent 数量上限",
        "values": [{"id": "instance_quota", "name": "单用户 Agent 数量上限", "value": "10"}]
      }
    ]
  },
  "instance_configs": [
    {
      "instance_id": 12137,
      "current_group_id": 7,
      "current_group_path": "技术部/前端组",
      "categories": [
        {
          "key": "model", "category_key": "model", "category_label": "模型",
          "instance_values": [{"id": "5", "name": "", "value": "Claude Sonnet 4", "status": "same"}],
          "status": "contained_in_target"
        },
        {
          "key": "skill.0", "category_key": "skill", "category_label": "技能", "sub_label": "",
          "instance_values": [{"id": "2", "name": "", "value": "代码助手", "status": "different"}],
          "status": "different"
        },
        {
          "key": "platformPolicy.instance_quota", "category_key": "platformPolicy", "category_label": "平台策略", "sub_label": "单用户 Agent 数量上限",
          "instance_values": [],
          "status": "contained_in_target"
        }
      ]
    }
  ],
  "not_found_instance_ids": []
}
```

### `POST /admin/stale-instances/apply` — 应用处理

弹窗「确认处理」或 Agent 列表页「分组处理」按钮。一个请求可同时混合多种 action（OneID 同步必需）。

- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| trigger_source | string | 是 | `user_edit` / `user_added_to_group` / `group_parent_change` / `oneid_sync` / `list_page_followup`。`list_page_followup` + `migrate` 时跳过 noop 短路：即使用户-分组关系恰好一致也允许执行迁移 |
| actions | array | 是 | 每条 action 一条记录；任一 action 枚举非法或必填字段缺失立即 400 |
| actions[].action | string | 是 | `migrate` / `handover` / `pending_user` / `archive_stop`，**不**支持 `delete` |
| actions[].instance_ids | uint[] | 是 | 该 action 作用的实例 ID 列表，**不能为空**；所有 actions 的 instance_ids 总数不得超过 500，超过返回 400 |
| actions[].target_group_id | uint | 条件 | `migrate`：**必填**（场景 B 显式传 0；其他场景传 > 0；JSON 缺字段一律 400）。`handover`：可选——**省略（JSON 不传该字段）时为「同分组移交」**：实例 `group_id` 保持不变，目标用户必须在当前分组中（当前分组为 0 时目标用户也必须无分组），否则 `failed: 目标用户不在当前实例所属的分组中，无法进行同分组移交`。**显式传值时**：目标用户单分组可传 0 自动选中；多分组必须显式指定一个 ∈ 目标用户分组列表；目标用户无分组必须为 0。**> 0 时 handler 会预先校验该 group 在 `user_groups` 表中存在；不存在 → 该实例 `failed: target_group_not_found`，不会被 noop 吞掉** |
| actions[].target_user_id | uint | 条件 | `handover` **必填且 > 0**（任意用户，不限同分组；不能是当前 owner）。实例的 `group_id` 会跟随目标用户的分组重新设定。**handler 会预先校验该用户存在；不存在 → 该实例 `failed: target_user_not_found`** |
| actions[].allow_migrate | bool | 否 | `pending_user` 用：允许用户自行迁移到新分组 |
| actions[].allow_same_group_handover | bool | 否 | `pending_user` 用：允许用户同组移交（场景 B/C 强制 false） |

- **响应：**

| 字段 | 类型 | 说明 |
|------|------|------|
| ok | bool | 是否成功 |
| results | array | 每个 (instance_id, action) 一条记录 |
| results[].instance_id | uint | 实例 ID |
| results[].action | string | 同入参 |
| results[].status | string | `success` / `failed` / `noop` |
| results[].error | string | 失败原因（i18n 完整句子，如「目标分组不存在」「目标用户与实例当前所有者相同，无法移交给本人」「当前场景下不允许执行此操作」等；DB 系统错误原样返回） |

### `GET /admin/stale-instances/records` — 处理记录列表

按时间倒序返回 `instance_change_group_records`。

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| instance_id | string | 否 | 数字按 instance_pk 过滤；非数字按 CVM ins-xxx 过滤 |
| user_id | uint | 否 | 按 user_id_before/after 过滤 |
| group_id | uint | 否 | 按 group_id_before/after 过滤 |
| action | string | 否 | 操作类型 |
| actor_type | string | 否 | `admin` / `user` / `oneid_sync` |
| trigger_source | string | 否 | 同 apply 入参 |
| from | string | 否 | RFC3339 时间下界 |
| to | string | 否 | RFC3339 时间上界 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页大小，默认 20，最大 100 |

- **响应：** `{ok, total, page, page_size, records[]}`，records 元素是 InstanceChangeGroupRecord 行。

### `POST /openclaw/stale-instances/rebind` — 用户自迁分组

仅当实例携带 `pending_user_action` + `allow_migrate` 时可用。成功后开机 + 清标。

- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID |
| target_group_id | uint | 是 | 目标分组（必须 ∈ 当前用户的 user_group_ids；用户已无任何分组时必须为 0） |

- **响应：** `{ok: true}`

### `POST /openclaw/stale-instances/initiate` — 发起同组移交

仅当实例携带 `pending_user_action` + `allow_same_group_handover` 时可用。

- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID |
| target_username | string | 是 | 目标用户名（后端解析为 user_id；必须是实例当前 group 的直属成员，且 ≠ 自己） |

- **响应：** `{ok: true}`。实例运行态保持 STOPPED。

### `POST /openclaw/stale-instances/cancel` — 原 owner 取消移交

- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID |

- **响应：** `{ok: true}`。运行态保持 STOPPED；实例回到 `pending_user_action` 状态。

### `POST /openclaw/stale-instances/accept` — 接收方同意

调用方必须 `= instance.handover_target_user_id`。成功后换 `user_id`、清掉所有 stale-instances 标记，异步开机。

- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID |

- **响应：** `{ok: true}`

### `POST /openclaw/stale-instances/reject` — 接收方拒绝

调用方必须 `= instance.handover_target_user_id`。保持关机；写 `handover_rejected_by_user_id`。

- **Content-Type：** `application/json`
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 实例 ID |

- **响应：** `{ok: true}`

### 关联接口扩展（既有接口在本期增强字段）

- `POST /admin/instances/by-user-group`：放开对 `(user_id, group_id=0)` 对的过滤，允许查询某用户的未分组实例（用于场景 C：从未分组加入新分组）。
- `GET /admin/instances`：响应每条实例新增 `flags` 数组、`handover_target_user_id`、`handover_rejected_by_user_id`、`handover_initiated_at` 四个字段。
- `GET /openclaw/list`：返回结果扩展为「自有实例 ∪ 待我接收的移交实例」并集；每条实例新增 `list_role` 字段（`owner` / `handover_incoming`）；`handover_incoming` 实例的 `custom_model_config` / `user_data` 字段被屏蔽。新增 `flags`、`handover_target_user_id`、`handover_rejected_by_user_id`、`handover_initiated_at`、`user_name` 五个字段（前四个与 `/admin/instances` 同构；`user_name` 为 `user_id` 对应的用户名）。
- `GET /openclaw/status`：查询范围扩展为「自有实例 ∪ 待我接收的移交实例」并集（与 `/openclaw/list` 口径一致），移交接收方可查询待接收实例的状态。

## 项目与统一资产管理

项目是扁平的租户内实体。本期只支持本地 Agent Workspace 绑定项目；不创建云端实例项目关系，也不进行 CVM/TAT 下发。项目删除为物理删除：成员关系随之清理，Workspace 保留原 `project_id` 以便兼容展示，但不会再获取资产。

### `GET /admin/projects`

管理员查询项目。管理页保持分页；应用范围选择器传 `with_counts=false` 时与分组树一致，一次返回全部项目。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| q | string | 否 | 按项目名称模糊搜索 |
| page | int | 否 | 管理页页码，默认 1；`with_counts=false` 时忽略 |
| page_size | int | 否 | 管理页每页条数，默认 20；`with_counts=false` 时忽略 |
| with_counts | bool | 否 | 是否返回成员、本地 Agent、Workspace 与资产计数；默认 true。传 false 时不计算计数并返回全部项目 |

- **响应：** `{ok: true, projects: [{id, name, description, member_count, cloud_instance_count, local_agent_count, workspace_count, asset_count, created_at, updated_at}], total, page, page_size}`。本期 `cloud_instance_count` 固定为 0。

### `POST /admin/projects/create` 与 `POST /admin/projects/update`

- **权限：** 管理员
- **Content-Type：** `application/json`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 条件 | update 时必填 |
| name | string | 条件 | create 必填；update 传入时租户内唯一且不能空 |
| description | string | 否 | 项目说明 |

- **响应：** `{ok: true, project}`。

### `GET /admin/projects/delete-impact` 与 `POST /admin/projects/delete`

删除前可用 `delete-impact` 查询阻塞项；只有项目工具应用范围（`skill` / `rule`）阻塞删除。直接资产绑定（`asset_skill` / `asset_rule`）在删除事务中清理，成员与本地 Workspace 均为非阻塞关联。`delete` 会在事务中二次检查，冲突响应含 `reason=has_dependencies`。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| project_ids | uint[] | 条件 | delete-impact 使用逗号分隔的 query 参数 |
| id | uint | 条件 | delete 使用 JSON 单个项目 ID |

- **响应：** delete-impact 返回 `{results: [{project_id, can_delete, blockers}]}`；delete 返回 `{ok: true}`。

### 项目成员

- `GET /admin/projects/members?id=<id>&q=&page=&page_size=`：返回脱敏成员 DTO `{user_id, username, role, deleted_at, joined_at, projects, cloud_instance_count, local_workspace_count}`；`projects` 为该成员所属的全部项目简要列表 `{id, name, joined_at}`。
- `POST /admin/projects/members/set`：全量替换成员。
- `POST /admin/projects/members/add`：幂等增加成员。
- `POST /admin/projects/members/remove`：幂等移除成员。
- `GET /admin/projects/projects-by-users?user_ids=1,2`：批量查询用户所在项目，返回 `{ok: true, data: {"1": [{id, name}]}}`；列表按加入项目时间升序。

后三个成员写接口均使用 JSON 请求体：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 项目 ID |
| user_ids | uint[] | 是 | 用户 ID 列表；set 允许空数组清空 |

`GET /admin/projects/config-overview?project_ids=1,2` 返回与组织配置总览一致的单个 `agentTool` 分类：其中仅有本期支持的企业技能和企业规范。每个条目使用 `{id, label, sub_label, source}`；`source.type=all_users` 表示全员可见，`local` 表示当前项目直接配置的应用范围。不返回资产选择绑定，直接资产绑定由 `/admin/assets/detail` 查询。

### `GET /projects/mine`

当前登录用户（含 API Token）查询自己仍有效的项目，供 TeamAI 选择 Workspace 项目。

- **响应：** `{projects: [{id, name, description}]}`。

### 通用分组/项目资产接口

`GET /admin/assets/detail?target_type=group|project&target_id=<id>` 返回 `{current_version, assets}`。`assets` 是扁平数组，按本地绑定（`source=local`）在前、继承绑定（`source=inherited`）在后排序；继承项携带 `source_target` 分组摘要。版本记录模块尚未接入时 `current_version=0`。
该接口只返回已经写入资产绑定的项目/分组资产；全员可见或命中可见范围但尚未绑定的资源，只由 candidates 接口返回，也不返回 `effective_assets` 或 `eligible`。

`GET /admin/assets/candidates` 支持 `target_type`、`target_id`、`asset_type`、`q`、`selected`、分页；候选只包括全员可见或目标（分组含祖先）可见的技能和规范。每个候选项始终返回 `selected` 和 `source`：`selected` 表示当前目标是否已直接选择该资产；`source.type` 复用配置来源枚举 `all_users`、`local`、`inherited`，其中 `inherited` 仅用于分组，并携带来源祖先的 `group_id` 和 `full_path`。

`POST /admin/assets/save` 使用 `{target_type, target_id, sync_mode, assets}` 全量替换目标直接资产。`target_type=group` 的资产保存到 `group_config_bindings` 的 `asset_skill` / `asset_rule`；项目保存到 `project_config_bindings` 的同名类型。`sync_mode` 为必填请求参数（`initial_only` 或 `continuous`；项目只允许 `continuous`，分组两种均可）。保存时若资产或同步模式有变化，会在同一事务内调用本模块 `RecordAssetSave` 生成版本记录，并按同步模式决定是否触发存量实例下发（详见 `docs/project-asset-api.md`）。

### `GET /admin/assets/versions`

分页查询某组织/项目的资产版本更新记录（倒序，最新在前）。

请求（query string）：

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| target_type | string | 是 | `project` / `group` |
| target_id | uint | 是 | 项目 ID 或分组 ID |
| page | int | 否 | 页码，从 1 开始，默认 1 |
| page_size | int | 否 | 每页条数，默认 20，最大 100 |

响应字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| total | int | 符合条件的总记录数 |
| page | int | 当前页码 |
| page_size | int | 每页条数 |
| data | array | 版本记录列表 |
| data[].record_id | uint | 记录 ID |
| data[].version | int | 版本号（自增序列） |
| data[].trigger_type | string | `manual` / `system` |
| data[].trigger_reason | string | `manual_save` / `asset_version_published` / `asset_deleted` / `asset_scope_changed` |
| data[].operator | object | 仅 `{type}`：`admin`(手动) / `system`(自动) |
| data[].segments | array | 变更分段（后端只返回结构化关键信息，文案由前端拼接） |
| data[].segments[].type | string | `added` / `removed` / `sync_mode` / `version_published` / `deleted` / `scope_changed` |
| data[].segments[].items | array | 该段变更项列表 |
| data[].segments[].items[].asset_type | string | `skill` / `rule` |
| data[].segments[].items[].name | string | 资产名称 |
| data[].segments[].items[].old_version | string | 仅 `version_published` 段；空字符串表示无 |
| data[].segments[].items[].new_version | string | 仅 `version_published` 段；空字符串表示无 |
| data[].segments[].value | string | 仅 `sync_mode` 段承载新模式值；为空表示本次未修改同步模式 |
| data[].created_at | string | ISO8601 时间 |

> 不返回 `summary` / `changes`：后端只返回 `segments[].items` 结构化字段，文案由前端拼接。`sync_mode` 段仅在同步模式发生变更时返回（`value` 非空），未变更则不展示。

响应示例：

```json
{
  "total": 5,
  "page": 1,
  "page_size": 20,
  "data": [
    {
      "record_id": 102,
      "version": 5,
      "trigger_type": "manual",
      "trigger_reason": "manual_save",
      "operator": {"type": "admin"},
      "segments": [
        {"type": "added", "items": [
          {"asset_type": "skill", "name": "API 文档生成器"},
          {"asset_type": "rule", "name": "代码审查工具"}
        ]},
        {"type": "sync_mode", "items": [], "value": "continuous"}
      ],
      "created_at": "2026-07-17 19:20:00"
    },
    {
      "record_id": 101,
      "version": 4,
      "trigger_type": "system",
      "trigger_reason": "asset_version_published",
      "operator": {"type": "system"},
      "segments": [
        {"type": "version_published", "items": [
          {"asset_type": "skill", "name": "代码审查工具", "old_version": "v2.0.0", "new_version": "v2.1.0"}
        ]}
      ],
      "created_at": "2026-07-16 19:20:00"
    }
  ]
}
```

### `GET /admin/projects/instances`

仅通过 `local_agent_scope_bindings` 查询使用该项目的本地 Agent Workspace，不扫描 `instances.local_agent_resources` JSON。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| project_id | uint | 是 | 项目 ID |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **响应：** `{instances: [{instance, bound_workspaces}], total, page, page_size}`；同一 Agent 聚合为一行。`instance` 保留原始实例字段，并新增 `status`（本地 Agent 为 `running` / `stopped`）和 `username`。每项 `bound_workspaces` 保留原 scope 字段并新增 `project_name`。

### `GET /admin/user-groups/instances`

查询用户级绑定到指定分组及其全部下级分组的本地 Agent。该接口与项目实例查询保持分开，避免把 `scope=user` 的分组绑定和 `scope=workspace` 的项目绑定混在同一请求参数中。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | uint | 是 | 分组 ID |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

- **响应：** `{instances: [{instance, bound_user_levels}], total, page, page_size}`；返回 `scope=user` 且 `group_id` 属于当前分组子树（含自身）的绑定，同一 Agent 聚合为一行。`instance` 保留原始实例字段，并新增 `status`（本地 Agent 为 `running` / `stopped`）和 `username`。每项 `bound_user_levels` 保留原 scope 字段，并新增其实际所属分组的 `group_name` 与 `group_full_path`（分组全路径）。

### 技能与规范应用范围扩展

`POST /admin/skills/create`、`POST /admin/skills/update`、`POST /admin/rules/create` 和 `POST /admin/rules/update` 的应用范围采用覆盖式更新：传入 `visibility_type` 时必须同时传 `group_ids` 和 `project_ids`，任一空字符串都会清空对应绑定；`visibility_type=group` 可以没有分组 ID（用于仅绑定项目或清空分组）；`visibility_type=all` 无条件清空两类绑定。不传三个范围字段时，创建接口继承同 slug 旧版本范围，更新接口不修改范围。范围缩小时，系统会删除被移出目标上该资产的直接绑定；仅原本直接绑定过该资产的目标会新增一条 `asset_scope_changed` 版本记录，且不会下发卸载命令。技能和规范列表、详情的 `visibility_groups` 使用 `[{group_id, group_name}]`，`visibility_projects` 使用 `[{project_id, project_name}]`。
