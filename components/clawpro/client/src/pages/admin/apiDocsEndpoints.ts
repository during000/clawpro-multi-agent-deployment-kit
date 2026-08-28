// 接口端点详情数据 — 完整的 98 个接口
// 按接口分类组织，每个端点包含：描述、认证、输入参数、输出参数、请求示例、响应示例、错误码

import type { EndpointDetail } from "./apiDocsData";

// ────────────────────────────────────────
// 公共接口
// ────────────────────────────────────────

const publicEndpoints: EndpointDetail[] = [
  {
    method: "GET",
    path: "/site",
    description: "获取站点基础信息。未登录时仅返回站点名称和 Logo 信息，已登录时额外返回功能开关配置。",
    auth: "可选",
    inputParams: [],
    outputParams: [
      { name: "name", type: "string", description: "站点名称" },
      { name: "has_logo", type: "bool", description: "是否已配置 Logo" },
      { name: "skillhub", type: "string", description: "技能中心地址（已登录时）" },
      { name: "terminal_enabled", type: "bool", description: "是否开启终端功能（已登录时）" },
      { name: "gateway_ui_enable", type: "bool", description: "是否开启 Gateway UI 功能（已登录时）" },
    ],
    requestExample: `curl https://example.com/api/site`,
    responseExample: `{
  "name": "Agent",
  "has_logo": true
}`,
    errorCodes: [],
  },
];

// ────────────────────────────────────────
// 实例管理
// ────────────────────────────────────────

const instanceEndpoints: EndpointDetail[] = [
  {
    method: "GET",
    path: "/openclaw/list",
    description: "获取当前用户的实例列表。",
    auth: "必须",
    inputParams: [],
    outputParams: [
      { name: "instances", type: "array", description: "实例列表" },
      { name: "instances[].ID", type: "uint", description: "实例 ID" },
      { name: "instances[].Name", type: "string", description: "实例名称" },
      { name: "instances[].InstanceId", type: "string", description: "云服务器实例 ID" },
      { name: "instances[].UserID", type: "uint", description: "所属用户 ID" },
      { name: "instances[].CreatedAt", type: "string", description: "创建时间" },
      { name: "instances[].UpdatedAt", type: "string", description: "更新时间" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  https://example.com/api/openclaw/list`,
    responseExample: `{
  "instances": [
    {
      "ID": 1,
      "Name": "my-claw",
      "InstanceId": "ins-xxxxxxxx",
      "UserID": 1,
      "CreatedAt": "2026-03-27T10:00:00Z",
      "UpdatedAt": "2026-03-27T10:00:00Z"
    }
  ]
}`,
    errorCodes: [{ code: "401", error: "未认证" }],
  },
  {
    method: "POST",
    path: "/openclaw/create",
    description: "创建一个新实例。",
    auth: "必须",
    contentType: "application/x-www-form-urlencoded",
    inputParams: [
      { name: "name", type: "string", required: "是", description: "实例名称（1~128 字符）" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
      { name: "redirect", type: "string", description: "跳转路径" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "name=my-claw" \\
  https://example.com/api/openclaw/create`,
    responseExample: `{"ok": true, "redirect": "/openclaw"}`,
    errorCodes: [
      { code: "400", error: "实例名称不能为空且不能超过128个字符" },
      { code: "403", error: "已达到实例配额上限（N）" },
    ],
  },
  {
    method: "POST",
    path: "/openclaw/delete",
    description: "删除指定实例，同时销毁关联的云服务器。",
    auth: "必须",
    contentType: "application/x-www-form-urlencoded",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "id=1" \\
  https://example.com/api/openclaw/delete`,
    responseExample: `{"ok": true}`,
    errorCodes: [
      { code: "400", error: "Bad request" },
      { code: "404", error: "Not found" },
    ],
  },
  {
    method: "GET",
    path: "/openclaw/status",
    description: "查询指定实例的云服务器运行状态。",
    auth: "必须",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID（Query 参数）" },
    ],
    outputParams: [
      { name: "state", type: "string", description: "实例状态，可选值：PENDING、RUNNING、STOPPED、REBOOTING、REINSTALLING、TERMINATING、RELEASED 等" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/openclaw/status?id=1"`,
    responseExample: `{"state": "RUNNING"}`,
    errorCodes: [{ code: "404", error: "实例不存在" }],
  },
  {
    method: "POST",
    path: "/openclaw/reboot",
    description: "重启指定实例的云服务器。",
    auth: "必须",
    contentType: "application/x-www-form-urlencoded",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "id=1" \\
  https://example.com/api/openclaw/reboot`,
    responseExample: `{"ok": true}`,
    errorCodes: [
      { code: "400", error: "Bad request / 该实例无关联的云服务器" },
      { code: "404", error: "Not found" },
    ],
  },
  {
    method: "POST",
    path: "/openclaw/reset",
    description: "重装指定实例的云服务器系统。将使用当前启用的镜像重新安装操作系统，实例的模型配置将被重置。",
    auth: "必须",
    contentType: "application/x-www-form-urlencoded",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "id=1" \\
  https://example.com/api/openclaw/reset`,
    responseExample: `{"ok": true}`,
    errorCodes: [
      { code: "400", error: "Bad request / 该实例无关联的云服务器" },
      { code: "404", error: "Not found" },
      { code: "500", error: "未启用任何镜像，无法重装实例" },
    ],
  },
  {
    method: "POST",
    path: "/openclaw/approve",
    description: "在实例上执行通道审批操作，用于提交通道的验证码进行审批确认。",
    auth: "必须",
    contentType: "application/x-www-form-urlencoded",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID" },
      { name: "module", type: "string", required: "否", description: "通道模块名，默认为 feishu" },
      { name: "code", type: "string", required: "是", description: "验证码" },
    ],
    outputParams: [
      { name: "ok", type: "string", description: '操作结果，值为 "true"' },
      { name: "output", type: "string", description: "脚本执行输出" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "id=1&module=feishu&code=123456" \\
  https://example.com/api/openclaw/approve`,
    responseExample: `{"ok": "true", "output": "审批成功"}`,
    errorCodes: [
      { code: "400", error: "缺少参数 code / 缺少参数 id" },
    ],
  },
  {
    method: "GET",
    path: "/openclaw/service-status",
    description: "查询指定实例上的服务运行状态，返回实例内部各服务组件的状态信息。",
    auth: "必须",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID（Query 参数）" },
    ],
    outputParams: [
      { name: "services", type: "array", description: "服务状态列表（取决于实例内部脚本输出）" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/openclaw/service-status?id=1"`,
    responseExample: `{
  "services": [
    {"name": "agent", "status": "running"},
    {"name": "proxy", "status": "running"}
  ]
}`,
    errorCodes: [
      { code: "400", error: "缺少参数 id" },
      { code: "404", error: "实例不存在" },
      { code: "500", error: "查询服务状态失败" },
    ],
  },
  {
    method: "POST",
    path: "/openclaw/terminal-url",
    description: "获取指定实例的 Web 终端登录 URL。需要管理员开启终端功能后才可使用。",
    auth: "必须",
    contentType: "application/x-www-form-urlencoded",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID" },
    ],
    outputParams: [
      { name: "login_url", type: "string", description: "Web 终端登录 URL" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "id=1" \\
  https://example.com/api/openclaw/terminal-url`,
    responseExample: `{"login_url": "https://terminal.example.com/session?token=xxx"}`,
    errorCodes: [
      { code: "400", error: "无效的实例 ID / 该实例无关联的云服务器" },
      { code: "403", error: "终端功能未开启，请联系管理员" },
      { code: "404", error: "实例不存在" },
    ],
  },
  {
    method: "POST",
    path: "/openclaw/denied-actions",
    description: "批量查询实例的禁用操作列表，用于判断实例是否可执行特定操作。",
    auth: "必须",
    contentType: "application/json",
    inputParams: [
      { name: "ids", type: "array[uint]", required: "是", description: "实例 ID 列表" },
    ],
    outputParams: [
      { name: "instances", type: "array", description: "实例禁用操作列表" },
      { name: "instances[].id", type: "uint", description: "实例 ID" },
      { name: "instances[].denied_actions", type: "array", description: "禁用操作列表" },
      { name: "instances[].denied_actions[].action", type: "string", description: "操作名称" },
      { name: "instances[].denied_actions[].code", type: "string", description: "错误码" },
      { name: "instances[].denied_actions[].message", type: "string", description: "错误信息" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{"ids": [1, 2]}' \\
  https://example.com/api/openclaw/denied-actions`,
    responseExample: `{
  "instances": [
    {"id": 1, "denied_actions": []},
    {"id": 2, "denied_actions": [{"action": "DescribeInstanceVncUrl", "code": "UnsupportedOperation", "message": "实例不支持此操作"}]}
  ]
}`,
    errorCodes: [
      { code: "400", error: "请求体格式错误" },
      { code: "401", error: "未认证" },
      { code: "500", error: "查询实例禁用操作失败" },
    ],
  },
  {
    method: "GET",
    path: "/openclaw/zones",
    description: "查询当前地域下的可用区列表。",
    auth: "必须",
    inputParams: [],
    outputParams: [
      { name: "Response.TotalCount", type: "int", description: "可用区总数" },
      { name: "Response.ZoneSet", type: "array", description: "可用区列表" },
      { name: "Response.ZoneSet[].Zone", type: "string", description: "可用区标识" },
      { name: "Response.ZoneSet[].ZoneName", type: "string", description: "可用区名称" },
      { name: "Response.ZoneSet[].ZoneState", type: "string", description: "可用区状态" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  https://example.com/api/openclaw/zones`,
    responseExample: `{
  "Response": {
    "TotalCount": 5,
    "ZoneSet": [
      {"Zone": "ap-guangzhou-3", "ZoneName": "广州三区", "ZoneState": "AVAILABLE"}
    ],
    "RequestId": "xxx"
  }
}`,
    errorCodes: [
      { code: "401", error: "未认证" },
      { code: "500", error: "查询可用区失败" },
    ],
  },
  {
    method: "POST",
    path: "/openclaw/set-gateway-ui",
    description: "为指定实例开启 Gateway UI 面板并返回访问地址。需要管理员先在后台开启 Gateway UI 功能。",
    auth: "必须",
    contentType: "application/x-www-form-urlencoded",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID" },
    ],
    outputParams: [
      { name: "gatewayUI", type: "string", description: "Gateway UI 访问地址（含认证 Token）" },
      { name: "token", type: "string", description: "Gateway UI 认证 Token" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "id=1" \\
  https://example.com/api/openclaw/set-gateway-ui`,
    responseExample: `{
  "gatewayUI": "http://1.2.3.4:18080/?token=abc123",
  "token": "abc123"
}`,
    errorCodes: [
      { code: "400", error: "缺少参数 id / Gateway UI 端口未分配，请先在管理后台配置" },
      { code: "403", error: "Gateway UI 功能未开启，请先在管理后台开启" },
    ],
  },
  {
    method: "POST",
    path: "/openclaw/retry",
    description: "重试加载失败的实例。仅当实例处于 load_failed 状态时可调用，根据之前的操作类型自动选择重启或重装方式。",
    auth: "必须",
    contentType: "application/x-www-form-urlencoded",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "id=1" \\
  https://example.com/api/openclaw/retry`,
    responseExample: `{"ok": true}`,
    errorCodes: [
      { code: "400", error: "无效的实例 ID / 当前状态为 xxx，只有 load_failed 状态才能重试" },
      { code: "404", error: "实例不存在" },
      { code: "409", error: "实例正在执行其他操作" },
    ],
  },
  {
    method: "GET",
    path: "/openclaw/check-agent-port",
    description: "检查指定实例上的服务端口是否正在运行。",
    auth: "必须",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID（Query 参数）" },
    ],
    outputParams: [
      { name: "running", type: "bool", description: "服务端口是否正在运行" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/openclaw/check-agent-port?id=1"`,
    responseExample: `{"running": true}`,
    errorCodes: [{ code: "400", error: "缺少参数 id" }],
  },
  {
    method: "GET",
    path: "/openclaw/check-gateway-access",
    description: "检查指定实例绑定的安全组入站规则中，Gateway UI 面板端口是否可访问。遍历实例实际绑定的所有安全组，只要有一个安全组放通了面板端口即视为可访问。",
    auth: "必须",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID（Query 参数）" },
    ],
    outputParams: [
      { name: "accessible", type: "bool", description: "面板端口是否可访问" },
      { name: "port", type: "int", description: "Gateway UI 端口号" },
      { name: "securityGroupIds", type: "array", description: "实例绑定的安全组 ID 列表" },
      { name: "message", type: "string", description: "检查结果描述" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/openclaw/check-gateway-access?id=1"`,
    responseExample: `{
  "accessible": true,
  "port": 18080,
  "securityGroupIds": ["sg-xxxxxx"],
  "message": "面板端口可正常访问"
}`,
    errorCodes: [
      { code: "400", error: "缺少参数 id / 该实例无关联的 CVM / Gateway UI 端口未分配，请先在管理后台配置" },
      { code: "403", error: "Gateway UI 功能未开启，请先在管理后台开启" },
    ],
  },
  {
    method: "POST",
    path: "/openclaw/set-env",
    description: "为指定实例批量设置或删除环境变量。采用增量更新模式——只修改传入的 key，保留已有的其他 key。设置成功后会自动重启 agent-gateway 服务使变量生效。",
    auth: "必须",
    contentType: "application/json",
    inputParams: [
      { name: "id", type: "uint", required: "二选一", description: "实例数据库 ID，与 instance_id 二选一，优先使用" },
      { name: "instance_id", type: "string", required: "二选一", description: "CVM 实例 ID（如 ins-abc123）" },
      { name: "env", type: "object", required: "是", description: "环境变量键值对，最多 50 个。value 为 string 表示设置，为 null 表示删除" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{"id": 123, "env": {"OPENAI_API_KEY": "sk-xxx", "DEBUG": "", "OLD_VAR": null}}' \\
  "https://example.com/api/openclaw/set-env"`,
    responseExample: `{"ok": true}`,
    errorCodes: [
      { code: "400", error: "请求体格式错误 / 缺少参数 id 或 instance_id / env 不能为空 / env 数量不能超过 50 / 无效的环境变量名 / 环境变量值必须是字符串或 null" },
      { code: "500", error: "设置环境变量失败" },
    ],
  },
  {
    method: "GET",
    path: "/openclaw/env",
    description: "查询指定实例当前的环境变量列表。通过 TAT 远程执行脚本读取实例上已配置的自定义环境变量。",
    auth: "必须",
    inputParams: [
      { name: "id", type: "uint", required: "二选一", description: "实例数据库 ID，与 instance_id 二选一，优先使用（Query 参数）" },
      { name: "instance_id", type: "string", required: "二选一", description: "CVM 实例 ID，如 ins-abc123（Query 参数）" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
      { name: "env", type: "object", description: "当前已配置的环境变量键值对，未设置过时为空对象 {}" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/openclaw/env?id=123"`,
    responseExample: `{
  "ok": true,
  "env": {
    "OPENAI_API_KEY": "sk-xxx",
    "DEBUG": ""
  }
}`,
    errorCodes: [
      { code: "400", error: "缺少参数 id 或 instance_id / 实例不存在" },
      { code: "500", error: "查询环境变量失败" },
    ],
  },
];

// ────────────────────────────────────────
// 通知管理
// ────────────────────────────────────────

const notificationEndpoints: EndpointDetail[] = [
  {
    method: "GET",
    path: "/openclaw/notifications",
    description: "获取当前用户的通知列表，支持分页。",
    auth: "必须",
    inputParams: [
      { name: "page", type: "int", required: "否", description: "页码，默认 1（Query 参数）" },
      { name: "page_size", type: "int", required: "否", description: "每页条数，默认 20，最大 100（Query 参数）" },
    ],
    outputParams: [
      { name: "notifications", type: "array", description: "通知列表" },
      { name: "notifications[].ID", type: "int", description: "通知 ID" },
      { name: "notifications[].CreatedAt", type: "string", description: "通知创建时间" },
      { name: "notifications[].UserID", type: "int", description: "接收用户 ID" },
      { name: "notifications[].InstanceID", type: "int", description: "关联实例 ID" },
      { name: "notifications[].InstanceName", type: "string", description: "实例名称" },
      { name: "notifications[].Type", type: "string", description: "通知类型（admin_delete / external_destroy）" },
      { name: "notifications[].Title", type: "string", description: "通知标题" },
      { name: "notifications[].Message", type: "string", description: "通知详情" },
      { name: "notifications[].IsRead", type: "bool", description: "是否已读" },
      { name: "notifications[].ReadAt", type: "string", description: "阅读时间（未读时为 null）" },
      { name: "page", type: "int", description: "当前页码" },
      { name: "page_size", type: "int", description: "每页条数" },
      { name: "total", type: "int", description: "总记录数" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/openclaw/notifications?page=1&page_size=10"`,
    responseExample: `{
  "notifications": [
    {
      "ID": 1,
      "CreatedAt": "2026-04-01T10:00:00Z",
      "UserID": 1,
      "InstanceID": 5,
      "InstanceName": "my-claw",
      "Type": "admin_delete",
      "Title": "实例已被管理员删除",
      "Message": "您的实例 my-claw 已被管理员删除",
      "IsRead": false,
      "ReadAt": null
    }
  ],
  "page": 1,
  "page_size": 10,
  "total": 1
}`,
    errorCodes: [],
  },
  {
    method: "POST",
    path: "/openclaw/notifications/read",
    description: "标记通知为已读。当 id 为 0 时标记当前用户的所有通知为已读，id 大于 0 时标记指定通知为已读。",
    auth: "必须",
    contentType: "application/json",
    inputParams: [
      { name: "id", type: "int", required: "是", description: "通知 ID，0 表示全部已读，大于 0 表示指定通知" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{"id": 1}' \\
  https://example.com/api/openclaw/notifications/read`,
    responseExample: `{"ok": true}`,
    errorCodes: [{ code: "400", error: "请求体格式错误" }],
  },
  {
    method: "GET",
    path: "/openclaw/notifications/count",
    description: "获取当前用户的未读通知数量。",
    auth: "必须",
    inputParams: [],
    outputParams: [
      { name: "unread_count", type: "int", description: "未读通知数量" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  https://example.com/api/openclaw/notifications/count`,
    responseExample: `{"unread_count": 3}`,
    errorCodes: [],
  },
];

// ────────────────────────────────────────
// 通道管理
// ────────────────────────────────────────

const channelEndpoints: EndpointDetail[] = [
  {
    method: "GET",
    path: "/openclaw/channels",
    description: "获取通道列表。不传 id 时返回所有可用通道及其参数定义；传 id 时返回指定实例上已配置的通道状态。",
    auth: "必须",
    inputParams: [
      { name: "id", type: "string", required: "否", description: "实例 ID（Query 参数），不传则返回全局通道列表" },
    ],
    outputParams: [
      { name: "ChannelID", type: "string", description: "通道标识" },
      { name: "Name", type: "string", description: "通道名称" },
      { name: "Enabled", type: "bool", description: "是否启用" },
      { name: "Custom", type: "bool", description: "是否为自定义通道" },
      { name: "CustomConfig", type: "string", description: "自定义通道配置 JSON" },
      { name: "params", type: "array", description: "通道参数定义列表" },
      { name: "params[].key", type: "string", description: "参数键名" },
      { name: "params[].label", type: "string", description: "参数标签" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  https://example.com/api/openclaw/channels`,
    responseExample: `[
  {
    "ChannelID": "feishu",
    "Name": "飞书",
    "Enabled": true,
    "Custom": false,
    "CustomConfig": "",
    "params": [
      {"key": "app_id", "label": "应用App ID"},
      {"key": "app_secret", "label": "应用App Secret"}
    ]
  }
]`,
    errorCodes: [
      { code: "400", error: "缺少参数 id" },
      { code: "401", error: "未认证" },
      { code: "404", error: "实例不存在" },
    ],
  },
  {
    method: "POST",
    path: "/openclaw/set-channel",
    description: "为指定实例配置通道凭证。通过提交通道标识和键值对形式的配置参数来设置通道。",
    auth: "必须",
    contentType: "application/x-www-form-urlencoded",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID" },
      { name: "channel", type: "string", required: "是", description: "通道标识，如 feishu、qqbot" },
      { name: "key[]", type: "string[]", required: "是", description: "配置参数键名数组" },
      { name: "value[]", type: "string[]", required: "是", description: "配置参数值数组（与 key 一一对应）" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
      { name: "output", type: "string", description: "脚本执行输出" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "id=1&channel=feishu&key[]=app_id&value[]=cli_xxx&key[]=app_secret&value[]=xxx" \\
  https://example.com/api/openclaw/set-channel`,
    responseExample: `{"ok": true, "output": "通道配置成功"}`,
    errorCodes: [
      { code: "400", error: "缺少参数 / 缺少配置参数 / 配置参数不能为空" },
    ],
  },
  {
    method: "POST",
    path: "/openclaw/del-channel",
    description: "删除指定实例上的通道配置。",
    auth: "必须",
    contentType: "application/x-www-form-urlencoded",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID" },
      { name: "channel", type: "string", required: "是", description: "通道标识" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
      { name: "output", type: "string", description: "脚本执行输出" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "id=1&channel=feishu" \\
  https://example.com/api/openclaw/del-channel`,
    responseExample: `{"ok": true, "output": "通道已移除"}`,
    errorCodes: [
      { code: "400", error: "缺少参数 / 缺少参数 id" },
    ],
  },
  {
    method: "GET",
    path: "/openclaw/auto-channel",
    description: "自动配置通道，通过 SSE（Server-Sent Events）流式返回配置进度。支持 QQ、飞书、微信三种通道的自动化配置。",
    auth: "必须",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID（Query 参数）" },
      { name: "channel", type: "string", required: "否", description: "通道类型，可选值：qqbot（默认）、feishu、agent-weixin" },
    ],
    outputParams: [
      { name: "event: qrcode", type: "SSE", description: '展示二维码 {"action":"show_qrcode","url":"..."}' },
      { name: "event: log", type: "SSE", description: '日志信息 {"action":"log","message":"..."}' },
      { name: "event: progress", type: "SSE", description: '进度更新 {"action":"progress","percent":50}' },
      { name: "event: finish", type: "SSE", description: '配置完成 {"action":"finish","message":"配置成功"}' },
      { name: "event: done", type: "SSE", description: '流程结束 {"message":"配置完成"}' },
      { name: "event: fail", type: "SSE", description: '配置失败 {"message":"错误描述"}' },
    ],
    requestExample: `curl -N -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/openclaw/auto-channel?id=1&channel=feishu"`,
    responseExample: `event: log
data: {"action":"log","message":"正在初始化..."}

event: qrcode
data: {"action":"show_qrcode","url":"https://example.com/qr/xxx"}

event: finish
data: {"action":"finish","message":"配置成功"}`,
    errorCodes: [
      { code: "400", error: "缺少参数 id / 不支持自动配置的通道类型: xxx" },
      { code: "404", error: "实例不存在" },
    ],
  },
];

// ────────────────────────────────────────
// 模型管理
// ────────────────────────────────────────

const modelEndpoints: EndpointDetail[] = [
  {
    method: "GET",
    path: "/openclaw/models",
    description: "获取所有已启用的 AI 模型列表。",
    auth: "必须",
    inputParams: [],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
      { name: "models", type: "array", description: "模型列表" },
      { name: "models[].id", type: "uint", description: "模型 ID" },
      { name: "models[].provider", type: "string", description: "模型提供商" },
      { name: "models[].model_id", type: "string", description: "模型标识" },
      { name: "models[].model_type", type: "string", description: "接口类型" },
      { name: "models[].context_len", type: "int", description: "上下文长度" },
      { name: "models[].model_name", type: "string", description: "模型显示名称" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  https://example.com/api/openclaw/models`,
    responseExample: `{
  "ok": true,
  "models": [
    {
      "id": 1,
      "provider": "custom",
      "model_id": "gpt-4",
      "model_type": "openai-completions",
      "context_len": 128000,
      "model_name": "GPT-4"
    }
  ]
}`,
    errorCodes: [
      { code: "400", error: "缺少参数 id" },
      { code: "401", error: "未认证" },
      { code: "404", error: "实例不存在" },
    ],
  },
  {
    method: "POST",
    path: "/openclaw/set-model",
    description: "为指定实例绑定 AI 模型。支持选择预配置模型或自定义模型。",
    auth: "必须",
    contentType: "application/x-www-form-urlencoded",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID" },
      { name: "ai_model_id", type: "string", required: "是", description: "模型 ID，0 表示自定义模型" },
      { name: "model_id", type: "string", required: "条件必填", description: "自定义模型标识（ai_model_id=0 时必填）" },
      { name: "model_name", type: "string", required: "否", description: "自定义模型显示名称，默认使用 model_id" },
      { name: "api_key", type: "string", required: "条件必填", description: "自定义模型 API Key（ai_model_id=0 时必填）" },
      { name: "url", type: "string", required: "条件必填", description: "自定义模型 API URL（ai_model_id=0 时必填）" },
      { name: "model_type", type: "string", required: "条件必填", description: "自定义模型接口类型（ai_model_id=0 时必填）" },
      { name: "context_len", type: "string", required: "否", description: "自定义模型上下文长度，默认 128000" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
      { name: "provider", type: "string", description: "模型提供商" },
      { name: "model_id", type: "string", description: "模型标识" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "id=1&ai_model_id=2" \\
  https://example.com/api/openclaw/set-model`,
    responseExample: `{"ok": true, "provider": "custom", "model_id": "gpt-4"}`,
    errorCodes: [
      { code: "400", error: "缺少参数 / 模型不存在或已禁用 / 模型ID、API Key、URL、接口类型不能为空" },
      { code: "403", error: "自定义模型功能未开启" },
    ],
  },
];

// ────────────────────────────────────────
// 技能管理
// ────────────────────────────────────────

const skillEndpoints: EndpointDetail[] = [
  {
    method: "GET",
    path: "/openclaw/skills",
    description: "获取指定实例上已安装的技能列表。",
    auth: "必须",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID（Query 参数）" },
    ],
    outputParams: [
      { name: "(array)", type: "array", description: '技能名称列表，如 ["skill-web-search", "skill-code-interpreter"]' },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/openclaw/skills?id=1"`,
    responseExample: `["skill-web-search", "skill-code-interpreter"]`,
    errorCodes: [
      { code: "400", error: "缺少参数 id" },
      { code: "401", error: "未认证" },
      { code: "404", error: "实例不存在" },
    ],
  },
  {
    method: "POST",
    path: "/openclaw/add-skill",
    description: "为指定实例安装技能。",
    auth: "必须",
    contentType: "application/x-www-form-urlencoded",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID" },
      { name: "skill_name", type: "string", required: "是", description: "技能名称" },
    ],
    outputParams: [
      { name: "(string)", type: "string", description: '返回字符串 "技能安装成功"' },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "id=1&skill_name=web-search" \\
  https://example.com/api/openclaw/add-skill`,
    responseExample: `"技能安装成功"`,
    errorCodes: [
      { code: "400", error: "skill_name 不能为空 / 缺少参数 id" },
    ],
  },
  {
    method: "GET",
    path: "/openclaw/install-skills",
    description: "查询指定实例的技能安装状态列表，返回所有技能包中的技能安装记录及其当前安装状态。",
    auth: "必须",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID（Query 参数）" },
    ],
    outputParams: [
      { name: "instance_id", type: "int", description: "实例 ID" },
      { name: "skills", type: "array", description: "技能安装记录列表" },
      { name: "skills[].id", type: "int", description: "安装记录 ID" },
      { name: "skills[].name", type: "string", description: "技能名称" },
      { name: "skills[].slug", type: "string", description: "技能标识" },
      { name: "skills[].version", type: "string", description: "技能版本" },
      { name: "skills[].install_status", type: "int", description: "安装状态（0=未安装, 1=安装中, 2=成功, 3=失败, 4=已取消）" },
      { name: "skills[].error_message", type: "string", description: "错误信息（安装失败时非空）" },
      { name: "total", type: "int", description: "技能总数" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/openclaw/install-skills?id=1"`,
    responseExample: `{
  "instance_id": 1,
  "skills": [
    {
      "id": 1,
      "created_at": "2026-04-01T10:00:00Z",
      "updated_at": "2026-04-01T10:05:00Z",
      "instance_id": 1,
      "name": "web-search",
      "slug": "openclaw-tavily-search",
      "version": "0.1.0",
      "install_status": 2,
      "error_message": ""
    }
  ],
  "total": 1
}`,
    errorCodes: [{ code: "400", error: "缺少参数 id" }],
  },
  {
    method: "POST",
    path: "/openclaw/retry-failed-skills",
    description: "重试指定实例上安装失败的技能。将所有 install_status=3（失败）的技能重置为待安装状态并异步重新执行安装。",
    auth: "必须",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID（Query 参数）" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
      { name: "retry_count", type: "int", description: "被重试的技能数量" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/openclaw/retry-failed-skills?id=1"`,
    responseExample: `{"ok": true, "retry_count": 2}`,
    errorCodes: [
      { code: "400", error: "缺少参数 id / 该实例无关联的 CVM" },
    ],
  },
  {
    method: "POST",
    path: "/openclaw/cancel-failed-skills",
    description: "取消指定实例上安装失败的技能。将所有 install_status=3（失败）的技能标记为已取消状态。",
    auth: "必须",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID（Query 参数）" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
      { name: "cancel_count", type: "int", description: "被取消的技能数量" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/openclaw/cancel-failed-skills?id=1"`,
    responseExample: `{"ok": true, "cancel_count": 1}`,
    errorCodes: [{ code: "400", error: "缺少参数 id" }],
  },
];

// ────────────────────────────────────────
// 插件管理
// ────────────────────────────────────────

const pluginEndpoints: EndpointDetail[] = [
  {
    method: "POST",
    path: "/openclaw/add-plugin",
    description: "为指定实例安装插件。",
    auth: "必须",
    contentType: "application/x-www-form-urlencoded",
    inputParams: [
      { name: "id", type: "string", required: "是", description: "实例 ID" },
      { name: "plugin", type: "string", required: "是", description: "插件名称，格式：@scope/name 或 name（仅允许字母、数字、点、连字符）" },
    ],
    outputParams: [
      { name: "(string)", type: "string", description: '返回字符串 "插件安装成功"' },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "id=1&plugin=@example/my-plugin" \\
  https://example.com/api/openclaw/add-plugin`,
    responseExample: `"插件安装成功"`,
    errorCodes: [
      { code: "400", error: "plugin 不能为空 / 插件名称格式不合法" },
    ],
  },
];

// ────────────────────────────────────────
// 用户配额
// ────────────────────────────────────────

const quotaEndpoints: EndpointDetail[] = [
  {
    method: "GET",
    path: "/quota/data",
    description: "查询当前用户的模型用量统计数据。支持按日期、模型、实例等维度组织聚合。",
    auth: "必须",
    inputParams: [
      { name: "start_date", type: "string", required: "否", description: "起始日期（格式 YYYY-MM-DD），默认今天" },
      { name: "end_date", type: "string", required: "否", description: "结束日期（格式 YYYY-MM-DD），默认今天" },
      { name: "group_by", type: "string", required: "否", description: "组织维度（逗号分隔），可选值：date、model、instance，默认 date,model" },
      { name: "ai_model_id", type: "string", required: "否", description: "按模型 ID 过滤" },
      { name: "instance_id", type: "string", required: "否", description: "按实例 ID 过滤" },
      { name: "order", type: "string", required: "否", description: "排序方式，desc 按总 Token 数降序" },
    ],
    outputParams: [
      { name: "quota_day", type: "int", description: "用户每日 Token 配额，-1 表示不限" },
      { name: "start_date", type: "string", description: "查询起始日期" },
      { name: "end_date", type: "string", description: "查询结束日期" },
      { name: "group_by", type: "array[string]", description: "实际使用的组织维度" },
      { name: "rows", type: "array", description: "统计数据行" },
      { name: "rows[].date", type: "string", description: "日期（按 date 组织时）" },
      { name: "rows[].ai_model_id", type: "uint", description: "模型 ID（按 model 组织时）" },
      { name: "rows[].model_name", type: "string", description: "模型名称（按 model 组织时）" },
      { name: "rows[].prompt_tokens", type: "int64", description: "输入 Token 数" },
      { name: "rows[].completion_tokens", type: "int64", description: "输出 Token 数" },
      { name: "rows[].total_tokens", type: "int64", description: "总 Token 数" },
      { name: "rows[].request_count", type: "int64", description: "请求次数" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/quota/data?start_date=2026-03-01&end_date=2026-03-27&group_by=date,model"`,
    responseExample: `{
  "quota_day": -1,
  "start_date": "2026-03-01",
  "end_date": "2026-03-27",
  "group_by": ["date", "model"],
  "rows": [
    {
      "date": "2026-03-27",
      "ai_model_id": 1,
      "model_name": "custom/gpt-4",
      "prompt_tokens": 1000,
      "completion_tokens": 500,
      "total_tokens": 1500,
      "request_count": 10
    }
  ]
}`,
    errorCodes: [
      { code: "401", error: "未认证" },
      { code: "500", error: "查询配额数据失败" },
    ],
  },
  {
    method: "GET",
    path: "/quota/logs",
    description: "查询当前用户的模型调用明细日志，支持分页和过滤。",
    auth: "必须",
    inputParams: [
      { name: "start_date", type: "string", required: "否", description: "起始日期，默认今天" },
      { name: "end_date", type: "string", required: "否", description: "结束日期，默认今天" },
      { name: "page", type: "string", required: "否", description: "页码，从 1 开始" },
      { name: "page_size", type: "string", required: "否", description: "每页条数，默认 50" },
      { name: "ai_model_id", type: "string", required: "否", description: "按模型 ID 过滤" },
      { name: "instance_id", type: "string", required: "否", description: "按实例 ID 过滤" },
    ],
    outputParams: [
      { name: "start_date", type: "string", description: "查询起始日期" },
      { name: "end_date", type: "string", description: "查询结束日期" },
      { name: "page", type: "int", description: "当前页码" },
      { name: "page_size", type: "int", description: "每页条数" },
      { name: "total", type: "int64", description: "总记录数" },
      { name: "logs", type: "array", description: "日志记录列表" },
      { name: "logs[].id", type: "uint", description: "记录 ID" },
      { name: "logs[].provider", type: "string", description: "模型提供商" },
      { name: "logs[].model", type: "string", description: "模型标识" },
      { name: "logs[].prompt_tokens", type: "int", description: "输入 Token 数" },
      { name: "logs[].completion_tokens", type: "int", description: "输出 Token 数" },
      { name: "logs[].total_tokens", type: "int", description: "总 Token 数" },
      { name: "logs[].status_code", type: "int", description: "HTTP 状态码" },
      { name: "logs[].latency", type: "int", description: "请求延迟（毫秒）" },
      { name: "logs[].created_at", type: "string", description: "记录时间" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/quota/logs?page=1&page_size=10"`,
    responseExample: `{
  "start_date": "2026-03-27",
  "end_date": "2026-03-27",
  "page": 1,
  "page_size": 10,
  "total": 50,
  "logs": [
    {
      "id": 100,
      "provider": "custom",
      "model": "gpt-4",
      "prompt_tokens": 100,
      "completion_tokens": 50,
      "total_tokens": 150,
      "status_code": 200,
      "latency": 1200,
      "created_at": "2026-03-27T10:30:00Z"
    }
  ]
}`,
    errorCodes: [
      { code: "401", error: "未认证" },
      { code: "500", error: "查询使用记录失败" },
    ],
  },
];

// ────────────────────────────────────────
// 导出按 sectionId 索引的接口数据
// ────────────────────────────────────────

export const endpointsBySection: Record<string, EndpointDetail[]> = {
  public: publicEndpoints,
  instance: instanceEndpoints,
  notification: notificationEndpoints,
  channel: channelEndpoints,
  model: modelEndpoints,
  skill: skillEndpoints,
  plugin: pluginEndpoints,
  quota: quotaEndpoints,
};
