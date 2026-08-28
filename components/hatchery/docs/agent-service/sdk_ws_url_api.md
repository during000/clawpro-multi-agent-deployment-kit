# SDK 获取实例内网连接 URL 接口

## 1. 概述

提供给 SDK 调用的接口，用户通过 API Token 鉴权后，传入实例的 CVM Instance ID，返回一个内网连接地址。调用方需在同 VPC 内网环境中使用该地址建立连接。

支持两种 Agent 类型：
- **OpenClaw** — 返回 WebSocket 连接地址（`ws://`），通过 `/ws` 端点双向通信
- **Hermes** — 返回 HTTP API 地址（`http://`），通过 OpenAI 兼容的 SSE 流式接口双向通信

**核心特点：**

- **仅内网可达**：返回的 URL 使用 CVM 私网 IP，调用方必须在同 VPC 内网才能连接
- **程序化调用**：面向 SDK/代码使用，非浏览器页面
- **Token 鉴权**：使用用户 API Token（`hk-` 前缀）进行 Bearer 认证
- **归属校验**：用户只能获取自己名下实例的连接地址
- **多协议支持**：按 agent 类型返回不同协议（WebSocket / HTTP SSE），通过 `protocol` 字段区分

---

## 2. 接口定义

### 请求

| 项目 | 值 |
|------|------|
| **方法** | `POST` |
| **路径** | `/openclaw/ws-url` |
| **认证** | `Authorization: Bearer hk-xxxxxxxx`（用户 API Token） |

### 请求参数

| 参数 | 位置 | 必选 | 类型 | 说明 |
|------|------|------|------|------|
| `instance_id` | JSON Body | 是 | string | 腾讯云 CVM 实例 ID，格式为 `ins-xxxxxxxx` |

### 请求示例

```bash
curl -X POST 'https://{hatchery-host}/openclaw/ws-url' \
  -H 'Authorization: Bearer hk-xxxxxxxxxxxxxxxxxxxx' \
  -H 'Content-Type: application/json' \
  -d '{"instance_id": "ins-xxxxxxxx"}'
```

---

## 3. 响应

### 成功响应（HTTP 200）

**OpenClaw（WebSocket 协议）：**

```json
{
  "url": "ws://10.0.1.5:8080/ws?token=xxxxxx",
  "protocol": "websocket",
  "token": "xxxxxx"
}
```

**Hermes（HTTP SSE 协议）：**

```json
{
  "url": "http://10.0.1.5:8642",
  "protocol": "sse",
  "token": "xxxxxx"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `url` | string | 内网连接地址。OpenClaw 为完整 WebSocket URL（含 token），Hermes 为 API Server 基础地址 |
| `protocol` | string | 连接协议类型：`websocket`（双向 WebSocket）或 `sse`（HTTP SSE 流式） |
| `token` | string | 连接鉴权 token。OpenClaw 为 gateway authToken，Hermes 为 API Server Key（用作 Bearer Token） |

### SDK 使用方式

**OpenClaw (protocol=websocket)**：
```javascript
const ws = new WebSocket(response.url); // url 已包含 token
```

**Hermes (protocol=sse)**：
```bash
curl -X POST "${response.url}/v1/chat/completions" \
  -H "Authorization: Bearer ${response.token}" \
  -H "Content-Type: application/json" \
  -d '{"model": "hermes-agent", "messages": [{"role": "user", "content": "Hello"}], "stream": true}'
```

### 错误响应

错误响应统一格式：

```json
{"error": "错误描述信息", "detail": "详细信息（可选）", "request_id": "腾讯云请求ID（可选）"}
```

| HTTP 状态码 | 场景 | 示例 |
|-------------|------|------|
| 401 | Token 无效或未提供 | `{"error": "未登录或 Token 无效"}` |
| 400 | 缺少 instance_id 参数 | `{"error": "缺少 instance_id 参数"}` |
| 400 | instance_id 格式无效 | `{"error": "instance_id 格式无效，应为 ins-xxxxxxxx"}` |
| 400 | 实例状态不是 running | `{"error": "实例当前状态为 stopped，仅 running 状态可获取连接地址"}` |
| 400 | 不支持的 agent 类型 | `{"error": "当前实例类型（xxx）暂不支持获取连接地址"}` |
| 403 | 实例不存在或无权访问 | `{"error": "实例不存在或无权访问"}` |
| 500 | 配置不完整或服务异常 | `{"error": "Hermes API Server 配置不完整（port 或 key 为空）"}` |
| 500 | CVM 查询失败或无内网 IP | `{"error": "实例无可用内网IP", "request_id": "abc-123-def"}` |
| 405 | 请求方法错误 | `{"error": "Method not allowed"}` |

---

## 4. 认证说明

使用用户 API Token 进行 Bearer Token 认证：

- Token 前缀：`hk-`
- 获取方式：在 Hatchery 平台「API Token 管理」页面创建
- 传递方式：HTTP Header `Authorization: Bearer hk-xxxxxxxx`

---

## 5. 各 Agent 类型行为差异

| | OpenClaw | Hermes |
|---|---|---|
| **协议** | WebSocket (`ws://`) | HTTP SSE (`http://`) |
| **初始化脚本** | `get_ws_url.sh` | `get_hermes_api.sh` |
| **端口来源** | 实例配置文件 `openclaw.json` | `.env` 中 `API_SERVER_PORT`（默认 8642） |
| **鉴权方式** | `gateway.auth.token`（URL 参数） | `API_SERVER_KEY`（Bearer Token） |
| **URL 格式** | `ws://ip:port/ws?token=xxx` | `http://ip:port` |
| **SDK 调用方式** | WebSocket 连接 | `POST /v1/chat/completions` (stream=true) |
| **首次耗时** | ~30-40s（改配置+重启） | ~30-60s（配置 .env + 启动 gateway） |
| **幂等** | 是（配置已正确则跳过重启） | 是（已启用则直接返回） |

---

## 6. Hermes API Server 端点说明

当 `protocol=sse` 时，SDK 使用返回的 `url` 作为基础地址，调用 OpenAI 兼容的 API：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/v1/chat/completions` | POST | Chat Completions（支持 stream=true 流式） |
| `/v1/responses` | POST | Responses API（支持多轮会话） |
| `/v1/models` | GET | 获取可用模型列表 |
| `/health` | GET | 健康检查 |

所有请求需携带 `Authorization: Bearer <token>`。

---

## 7. 使用约束

| 约束 | 说明 |
|------|------|
| **网络要求** | 调用方必须与 CVM 实例在同一 VPC 内网中 |
| **实例状态** | 仅 `running` 状态的实例可获取连接地址 |
| **归属限制** | 用户只能获取自己名下的实例地址 |
| **连接协议** | OpenClaw 返回 `ws://`，Hermes 返回 `http://`（均为内网非加密） |
| **安全组** | CVM 安全组需放通对应端口的内网入站规则 |

---

## 8. 注意事项

1. **内网连通性**：确保调用客户端与目标 CVM 在同一 VPC，或通过对等连接/VPN 互通
2. **安全组规则**：CVM 安全组需放通对应端口的内网入站规则（源地址限制为 VPC 网段即可）
3. **首次调用较慢**：新建实例首次调用时，接口会自动初始化服务，耗时 30-60 秒不等
4. **URL 有效期**：返回的 URL 中 IP/端口/token 在实例生命周期内不变，但实例重启后可能变化，建议每次连接前重新获取
5. **Token 安全**：URL 中的 token 是实例的鉴权凭证，请妥善保管，勿泄露到公网
6. **protocol 字段**：SDK 应根据 `protocol` 字段选择连接方式，而非硬编码协议

---

## 9. 实现说明

### 后端流程

1. `WithOpenAPI` + `WithAudit` 中间件：启用 API Token 鉴权 + 审计
2. `requireLogin` 验证用户身份
3. 通过 `instance_id`（ins-xxx）查询数据库，校验实例归属
4. 校验 `last_stable_state == "running"`
5. 查询 CVM DescribeInstances 获取内网 IP
6. 按 `AgentType` 分发到对应的初始化逻辑：
   - **OpenClaw**：执行 `get_ws_url.sh`（确保 bind=lan + 读取 port/token）
   - **Hermes**：执行 `get_hermes_api.sh`（确保 API Server 启用 + 读取 port/key）
7. 检查安全组端口放通（仅日志告警）
8. 拼接连接地址返回

### 相关文件

| 文件 | 说明 |
|------|------|
| `controller/openclaw_ws_url.go` | Handler 实现（含两种类型分发） |
| `scripts/get_ws_url.sh` | OpenClaw 专用 TAT 脚本（确保内网可访问 + 读取配置） |
| `scripts/get_hermes_api.sh` | Hermes 专用 TAT 脚本（确保 API Server 启用 + 读取配置） |
| `main.go` | 路由注册 |
