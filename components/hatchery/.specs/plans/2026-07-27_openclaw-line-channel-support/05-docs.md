# 05. Docs — 文档更新

## 已更新

### `docs/API.md`

以下接口文档已同步更新 LINE 支持说明：

| 接口 | 更新内容 |
|------|---------|
| `POST /openclaw/proxy/prepare` | `kind` 参数新增 `line` 支持 |
| `POST /openclaw/set-channel` | 新增 LINE 凭证说明（`channel_token`、`channel_secret`）；响应说明新增 `proxy_route_id`、`proxy_endpoint` |
| `POST /admin/instances/proxy/prepare` | `kind` 参数新增 `line` 支持 |
| `POST /admin/instances/set-channel` | `channel` 参数新增 `line` 示例；响应说明同步更新 |

### 无需更新

- `docs/INDEX.md`：无需新增功能文档模块（LINE 通道属于已有通道体系的扩展）
- `docs/i18n.md`：本次变更未新增 i18n key
- `docs/testing.md`：无需更新

## 增量说明（2026-07-30）

- 本次新增 `~/.hermes/config.yaml` 的 `gateway.platforms.line.enabled` 写入/删除逻辑，属 Hermes Agent 运行时内部配置，不影响对外 API 契约，`docs/API.md` 无需再次更新。
