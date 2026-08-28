// 管理接口端点详情数据
// 包含所有 /admin/* 路径的接口

import type { EndpointDetail } from "./apiDocsData";

// ────────────────────────────────────────
// 管理接口 — 站点配置
// ────────────────────────────────────────

export const adminSiteEndpoints: EndpointDetail[] = [
  {
    method: "GET", path: "/admin/config",
    description: "获取站点配置信息，包括站点名称、区域、VPC、安全组、模板配置等。",
    auth: "必须（管理员）",
    inputParams: [
      { name: "template_path", type: "string[]", required: "否", description: "要从云服务器模板中提取返回的子模块（Query 参数），可选值：internet_accessible、system_disk、instance_type、instance_charge_type、instance_charge_prepaid。支持多个，未传时默认返回 internet_accessible" },
    ],
    outputParams: [
      { name: "config", type: "object", description: "站点配置对象" },
      { name: "config.name", type: "string", description: "站点名称" },
      { name: "config.has_logo", type: "bool", description: "是否已配置 Logo" },
      { name: "config.cvm_region", type: "string", description: "云服务器所在区域" },
      { name: "config.available_zones", type: "array[string]", description: "当前区域可用区列表" },
      { name: "config.cvm_secret_id", type: "string", description: "云 API 密钥 ID" },
      { name: "config.cvm_template", type: "string", description: "云服务器创建模板 JSON" },
      { name: "config.security_group_id", type: "string", description: "安全组 ID" },
      { name: "config.skillhub", type: "string", description: "技能中心地址" },
      { name: "config.cvm_uin", type: "string", description: "云账号 UIN" },
      { name: "config.domain", type: "string", description: "服务域名" },
      { name: "config.global_token_quota_day", type: "int", description: "全局每日 Token 配额，-1 表示不限" },
      { name: "config.public_image_id", type: "string", description: "公有镜像 ID" },
      { name: "config.vpc_id", type: "string", description: "VPC ID" },
      { name: "config.subnet_ids", type: "string", description: "子网配置 JSON（可用区到子网 ID 的映射）" },
      { name: "config.terminal_enabled", type: "bool", description: "是否开启终端功能" },
      { name: "config.gateway_ui_enable", type: "bool", description: "是否开启 Gateway UI" },
      { name: "config.gateway_ui_port", type: "int", description: "Gateway UI 端口号" },
      { name: "config.internet_accessible", type: "object", description: "公网配置子模块（按 template_path 动态返回）" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/admin/config?template_path=internet_accessible&template_path=system_disk"`,
    responseExample: `{
  "config": {
    "name": "Agent",
    "has_logo": true,
    "cvm_region": "广州",
    "available_zones": ["ap-guangzhou-3", "ap-guangzhou-4"],
    "security_group_id": "sg-xxxxxxxx",
    "global_token_quota_day": -1,
    "terminal_enabled": true,
    "gateway_ui_enable": false,
    "gateway_ui_port": 0
  }
}`,
    errorCodes: [{ code: "400", error: "不支持的 template_path: xxx" }],
  },
  {
    method: "POST", path: "/admin/config",
    description: "更新站点基础配置，包括站点名称、Logo、全局 Token 配额、终端功能开关、Gateway UI 开关等。",
    auth: "必须（管理员）",
    contentType: "multipart/form-data 或 application/x-www-form-urlencoded",
    inputParams: [
      { name: "name", type: "string", required: "否", description: "站点名称" },
      { name: "global_token_quota_day", type: "string", required: "否", description: "全局每日 Token 配额，-1 或非负整数" },
      { name: "terminal_enabled", type: "string", required: "否", description: "终端功能开关，true 或 false" },
      { name: "gateway_ui_enable", type: "string", required: "否", description: "Gateway UI 开关，true 或 false" },
      { name: "logo", type: "file", required: "否", description: "Logo 图片文件（PNG/JPEG/SVG，最大 512KB）" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
      { name: "gateway_ui_port", type: "int", description: "Gateway UI 端口号（仅在开启时返回）" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "name=Agent&global_token_quota_day=-1&terminal_enabled=true" \\
  https://example.com/api/admin/config`,
    responseExample: `{"ok": true}`,
    errorCodes: [
      { code: "400", error: "全局配额必须为 -1 或非负整数 / 仅支持 PNG、JPEG、SVG 格式 / Logo 文件不能超过 512KB / 开启 Gateway UI 需要先配置安全组" },
    ],
  },
  {
    method: "POST", path: "/admin/config/cvm",
    description: "更新云服务器相关配置（VPC、子网、技能中心地址等）。",
    auth: "必须（管理员）",
    contentType: "application/x-www-form-urlencoded",
    inputParams: [
      { name: "skillhub", type: "string", required: "否", description: "技能中心地址" },
      { name: "vpc_id", type: "string", required: "否", description: "VPC ID，清空则同时清空子网" },
      { name: "subnet_ids", type: "string", required: "否", description: "子网配置 JSON（可用区到子网 ID 的映射）" },
    ],
    outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -d "skillhub=https://skillhub.example.com&vpc_id=vpc-xxx" \\
  https://example.com/api/admin/config/cvm`,
    responseExample: `{"ok": true}`,
    errorCodes: [{ code: "400", error: "CVM 模板必须是合法的 JSON / 配置了子网但未配置全局 VPC ID" }],
  },
  {
    method: "POST", path: "/admin/config/template",
    description: "更新云服务器模板的指定子模块。支持的模块：internet_accessible、system_disk、instance_type、instance_charge_type、instance_charge_prepaid。",
    auth: "必须（管理员）",
    contentType: "application/json",
    inputParams: [
      { name: "internet_accessible", type: "object", required: "否", description: "公网配置" },
      { name: "system_disk", type: "object", required: "否", description: "系统盘配置" },
      { name: "instance_type", type: "string", required: "否", description: "实例机型" },
      { name: "instance_charge_type", type: "string", required: "否", description: "实例计费模式" },
      { name: "instance_charge_prepaid", type: "object", required: "否", description: "预付费配置" },
    ],
    outputParams: [
      { name: "ok", type: "bool", description: "操作是否成功" },
      { name: "cvm_template", type: "string", description: "更新后的完整模板 JSON" },
      { name: "message", type: "string", description: "操作结果信息" },
    ],
    requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{"internet_accessible":{"public_ip_assigned":true,"internet_max_bandwidth_out":100}}' \\
  https://example.com/api/admin/config/template`,
    responseExample: `{"ok": true, "cvm_template": "{...}", "message": "模板配置保存成功"}`,
    errorCodes: [
      { code: "400", error: "请求体解析失败 / 请求体不能为空 / 不允许修改的字段: xxx" },
      { code: "405", error: "仅支持 POST 方法" },
    ],
  },
  {
    method: "GET", path: "/admin/vpc/cloud",
    description: "查询当前区域下的云端 VPC 列表。",
    auth: "必须（管理员）",
    inputParams: [],
    outputParams: [
      { name: "vpcs", type: "array", description: "VPC 列表" },
      { name: "vpcs[].vpc_id", type: "string", description: "VPC ID" },
      { name: "vpcs[].name", type: "string", description: "VPC 名称" },
      { name: "vpcs[].cidr_block", type: "string", description: "VPC 网段" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  https://example.com/api/admin/vpc/cloud`,
    responseExample: `{"vpcs": [{"vpc_id": "vpc-xxxxxxxx", "name": "Default-VPC", "cidr_block": "172.16.0.0/16"}]}`,
    errorCodes: [
      { code: "403", error: "需要管理员权限" },
      { code: "500", error: "查询云 VPC 列表失败" },
    ],
  },
  {
    method: "GET", path: "/admin/subnet/cloud",
    description: "查询指定 VPC 和可用区下的云端子网列表。",
    auth: "必须（管理员）",
    inputParams: [
      { name: "vpc_id", type: "string", required: "是", description: "VPC ID（Query 参数）" },
      { name: "zone", type: "string", required: "是", description: "可用区（Query 参数）" },
    ],
    outputParams: [
      { name: "subnets", type: "array", description: "子网列表" },
      { name: "subnets[].subnet_id", type: "string", description: "子网 ID" },
      { name: "subnets[].name", type: "string", description: "子网名称" },
      { name: "subnets[].cidr_block", type: "string", description: "子网网段" },
    ],
    requestExample: `curl -H "Authorization: Bearer hk-xxx" \\
  "https://example.com/api/admin/subnet/cloud?vpc_id=vpc-xxx&zone=ap-guangzhou-3"`,
    responseExample: `{"subnets": [{"subnet_id": "subnet-xxxxxxxx", "name": "Default-Subnet", "cidr_block": "172.16.0.0/24"}]}`,
    errorCodes: [{ code: "400", error: "vpc_id 参数不能为空 / zone 参数不能为空" }],
  },
];

// ────────────────────────────────────────
// 管理接口 — 安全组
// ────────────────────────────────────────

export const adminSecurityEndpoints: EndpointDetail[] = [
  { method: "GET", path: "/admin/config/security-group", description: "查询当前已配置的安全组信息。", auth: "必须（管理员）", inputParams: [], outputParams: [{ name: "Response.SecurityGroupSet", type: "array", description: "安全组列表" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" https://example.com/api/admin/config/security-group`, responseExample: `{"Response": {"SecurityGroupSet": [{"SecurityGroupId": "sg-xxxxxxxx", "SecurityGroupName": "openclaw-sg"}], "TotalCount": 1}}`, errorCodes: [{ code: "404", error: "未配置安全组" }] },
  { method: "POST", path: "/admin/config/security-group", description: "创建新的安全组，并自动绑定到站点配置。", auth: "必须（管理员）", contentType: "application/json", inputParams: [{ name: "GroupName", type: "string", required: "是", description: "安全组名称" }, { name: "GroupDescription", type: "string", required: "否", description: "安全组描述" }], outputParams: [{ name: "Response.SecurityGroup", type: "object", description: "创建的安全组详情" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -H "Content-Type: application/json" -d '{"GroupName":"openclaw-sg"}' https://example.com/api/admin/config/security-group`, responseExample: `{"Response": {"SecurityGroup": {"SecurityGroupId": "sg-xxxxxxxx"}, "RequestId": "xxx"}}`, errorCodes: [{ code: "400", error: "请求参数格式错误" }] },
  { method: "PUT", path: "/admin/config/security-group", description: "修改当前已绑定的安全组属性（名称、描述）。", auth: "必须（管理员）", contentType: "application/json", inputParams: [{ name: "SecurityGroupName", type: "string", required: "否", description: "安全组名称" }, { name: "SecurityGroupDesc", type: "string", required: "否", description: "安全组描述" }], outputParams: [{ name: "Response.RequestId", type: "string", description: "请求 ID" }], requestExample: `curl -X PUT -H "Authorization: Bearer hk-xxx" -H "Content-Type: application/json" -d '{"SecurityGroupName":"new-name"}' https://example.com/api/admin/config/security-group`, responseExample: `{"Response": {"RequestId": "xxx"}}`, errorCodes: [{ code: "400", error: "未配置安全组，请先创建或绑定安全组 / 请求参数格式错误" }] },
  { method: "GET", path: "/admin/config/security-group/policies", description: "查询当前安全组的入站和出站规则列表。", auth: "必须（管理员）", inputParams: [], outputParams: [{ name: "Response.SecurityGroupPolicySet.Ingress", type: "array", description: "入站规则列表" }, { name: "Response.SecurityGroupPolicySet.Egress", type: "array", description: "出站规则列表" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" https://example.com/api/admin/config/security-group/policies`, responseExample: `{"Response": {"SecurityGroupPolicySet": {"Ingress": [{"PolicyIndex": 0, "Protocol": "tcp", "Port": "22", "CidrBlock": "0.0.0.0/0", "Action": "ACCEPT"}], "Egress": []}}}`, errorCodes: [{ code: "404", error: "未配置安全组" }] },
  { method: "POST", path: "/admin/config/security-group/policies", description: "为当前安全组添加入站或出站规则。", auth: "必须（管理员）", contentType: "application/json", inputParams: [{ name: "SecurityGroupPolicySet", type: "object", required: "是", description: "安全组规则集" }, { name: "SecurityGroupPolicySet.Ingress", type: "array", required: "否", description: "入站规则列表" }, { name: "SecurityGroupPolicySet.Egress", type: "array", required: "否", description: "出站规则列表" }], outputParams: [{ name: "Response.RequestId", type: "string", description: "请求 ID" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -H "Content-Type: application/json" -d '{"SecurityGroupPolicySet":{"Ingress":[{"Protocol":"tcp","Port":"443","CidrBlock":"0.0.0.0/0","Action":"ACCEPT"}]}}' https://example.com/api/admin/config/security-group/policies`, responseExample: `{"Response": {"RequestId": "xxx"}}`, errorCodes: [{ code: "400", error: "未配置安全组 / 请求参数格式错误" }] },
  { method: "PUT", path: "/admin/config/security-group/policies", description: "替换当前安全组的指定规则。", auth: "必须（管理员）", contentType: "application/json", inputParams: [{ name: "SecurityGroupPolicySet", type: "object", required: "是", description: "安全组规则集（需包含 PolicyIndex）" }], outputParams: [{ name: "Response.RequestId", type: "string", description: "请求 ID" }], requestExample: `curl -X PUT -H "Authorization: Bearer hk-xxx" -H "Content-Type: application/json" -d '{"SecurityGroupPolicySet":{"Ingress":[{"PolicyIndex":0,"Protocol":"tcp","Port":"8080","CidrBlock":"0.0.0.0/0","Action":"ACCEPT"}]}}' https://example.com/api/admin/config/security-group/policies`, responseExample: `{"Response": {"RequestId": "xxx"}}`, errorCodes: [{ code: "400", error: "未配置安全组 / 请求参数格式错误" }] },
  { method: "DELETE", path: "/admin/config/security-group/policies", description: "删除当前安全组的指定规则。", auth: "必须（管理员）", contentType: "application/json", inputParams: [{ name: "SecurityGroupPolicySet", type: "object", required: "是", description: "安全组规则集（通过 PolicyIndex 指定要删除的规则）" }], outputParams: [{ name: "Response.RequestId", type: "string", description: "请求 ID" }], requestExample: `curl -X DELETE -H "Authorization: Bearer hk-xxx" -H "Content-Type: application/json" -d '{"SecurityGroupPolicySet":{"Ingress":[{"PolicyIndex":0}]}}' https://example.com/api/admin/config/security-group/policies`, responseExample: `{"Response": {"RequestId": "xxx"}}`, errorCodes: [{ code: "400", error: "未配置安全组 / 请求参数格式错误" }] },
];

// ────────────────────────────────────────
// 管理接口 — 用户管理
// ────────────────────────────────────────

export const adminUserEndpoints: EndpointDetail[] = [
  { method: "GET", path: "/admin/users", description: "获取用户列表，支持分页和按用户名搜索。", auth: "必须（管理员）", inputParams: [{ name: "page", type: "string", required: "否", description: "页码，默认 1" }, { name: "page_size", type: "string", required: "否", description: "每页条数，默认 20，最大 100" }, { name: "username", type: "string", required: "否", description: "按用户名过滤" }, { name: "fuzzy", type: "string", required: "否", description: "模糊匹配，1 开启" }], outputParams: [{ name: "users", type: "array", description: "用户列表" }, { name: "users[].ID", type: "uint", description: "用户 ID" }, { name: "users[].Username", type: "string", description: "用户名" }, { name: "users[].Email", type: "string", description: "邮箱" }, { name: "users[].Role", type: "string", description: "角色（admin / user）" }, { name: "users[].InstanceQuota", type: "int", description: "实例配额" }, { name: "users[].TokenQuotaDay", type: "int", description: "每日 Token 配额" }, { name: "users[].VpcId", type: "string", description: "用户自动创建的 VPC ID" }, { name: "users[].CreatedAt", type: "string", description: "创建时间" }, { name: "users[].DeletedAt", type: "object", description: "软删除时间（已封禁时非 null）" }, { name: "page", type: "int", description: "当前页码" }, { name: "page_size", type: "int", description: "每页条数" }, { name: "total", type: "int64", description: "总用户数" }, { name: "total_pages", type: "int", description: "总页数" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/users?page=1&page_size=10"`, responseExample: `{"users": [{"ID": 1, "Username": "admin", "Email": "", "Role": "admin", "InstanceQuota": 1, "TokenQuotaDay": -1, "VpcId": "", "CreatedAt": "2026-03-01T10:00:00Z", "DeletedAt": null}], "page": 1, "page_size": 10, "total": 1, "total_pages": 1}`, errorCodes: [{ code: "403", error: "需要管理员权限" }] },
  { method: "POST", path: "/admin/create", description: "创建新用户。可选发送欢迎邮件。", auth: "必须（管理员）", contentType: "application/x-www-form-urlencoded", inputParams: [{ name: "username", type: "string", required: "是", description: "用户名" }, { name: "password", type: "string", required: "是", description: "密码" }, { name: "email", type: "string", required: "否", description: "邮箱地址" }, { name: "role", type: "string", required: "否", description: "角色，默认 user" }, { name: "instance_quota", type: "string", required: "否", description: "实例配额（0~999），默认 1" }, { name: "token_quota_day", type: "string", required: "否", description: "每日 Token 配额，-1 表示不限" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -d "username=newuser&password=P@ssw0rd&role=user" https://example.com/api/admin/create`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "用户名和密码不能为空 / 实例配额必须为 0~999 的整数 / Token 配额必须为 -1 或非负整数" }, { code: "403", error: "已达到用户数上限（N）" }, { code: "409", error: "创建失败：用户名已存在" }] },
  { method: "POST", path: "/admin/batch-create", description: "批量创建用户，请求体为 JSON 数组。", auth: "必须（管理员）", contentType: "application/json", inputParams: [{ name: "username", type: "string", required: "是", description: "用户名" }, { name: "password", type: "string", required: "是", description: "密码" }, { name: "email", type: "string", required: "否", description: "邮箱" }, { name: "role", type: "string", required: "否", description: "角色，默认 user" }, { name: "instance_quota", type: "int", required: "否", description: "实例配额" }, { name: "token_quota_day", type: "int", required: "否", description: "每日 Token 配额" }], outputParams: [{ name: "results", type: "array", description: "创建结果列表" }, { name: "results[].username", type: "string", description: "用户名" }, { name: "results[].ok", type: "bool", description: "是否创建成功" }, { name: "results[].error", type: "string", description: "失败原因（成功时无此字段）" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -H "Content-Type: application/json" -d '[{"username":"user1","password":"pass1"},{"username":"user2","password":"pass2","role":"admin"}]' https://example.com/api/admin/batch-create`, responseExample: `{"results": [{"username": "user1", "ok": true}, {"username": "user2", "ok": true}]}`, errorCodes: [{ code: "400", error: "请求体格式错误 / 用户列表不能为空，不能超过5000" }, { code: "403", error: "导入后将超过用户数上限（N），当前已有 M 个用户" }] },
  { method: "POST", path: "/admin/delete", description: "软删除指定用户（封禁），同时关停该用户所有运行中的云服务器实例。不能删除初始管理员。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "用户 ID（Query 参数）" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/delete?id=2"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "403", error: "不能删除初始管理员" }, { code: "404", error: "用户不存在" }] },
  { method: "POST", path: "/admin/hard-delete", description: "永久删除指定用户。要求用户名下无实例存在。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "用户 ID（Query 参数）" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/hard-delete?id=2"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "403", error: "不能删除初始管理员" }, { code: "404", error: "用户不存在" }, { code: "409", error: "该用户还有实例存在，请先删除其实例" }] },
  { method: "POST", path: "/admin/restore", description: "恢复已封禁（软删除）的用户，同时尝试启动该用户已停止的云服务器实例。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "用户 ID（Query 参数）" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/restore?id=2"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "403", error: "不能操作初始管理员" }, { code: "404", error: "用户不存在" }] },
  { method: "POST", path: "/admin/reset-password", description: "重置指定用户的密码。初始管理员的密码只能通过管理令牌重置。", auth: "必须（管理员）", contentType: "application/x-www-form-urlencoded", inputParams: [{ name: "id", type: "string", required: "是", description: "用户 ID（Query 参数）" }, { name: "password", type: "string", required: "是", description: "新密码" }, { name: "email", type: "string", required: "否", description: "邮箱地址，设置后会发送通知邮件" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -d "password=NewP@ss123" "https://example.com/api/admin/reset-password?id=2"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "密码不能为空" }, { code: "403", error: "初始管理员密码只能通过 admin-token 重置" }, { code: "404", error: "用户不存在" }] },
  { method: "POST", path: "/admin/update-user", description: "更新指定用户的信息，包括邮箱、角色、实例配额、Token 配额等。", auth: "必须（管理员）", contentType: "application/x-www-form-urlencoded", inputParams: [{ name: "id", type: "string", required: "是", description: "用户 ID（Query 参数）" }, { name: "email", type: "string", required: "否", description: "邮箱" }, { name: "role", type: "string", required: "否", description: "角色" }, { name: "instance_quota", type: "string", required: "否", description: "实例配额（0~999）" }, { name: "token_quota_day", type: "string", required: "否", description: "每日 Token 配额" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -d "role=admin&instance_quota=10" "https://example.com/api/admin/update-user?id=2"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "没有可更新的字段" }, { code: "403", error: "不能修改初始管理员的角色" }, { code: "404", error: "用户不存在" }] },
  { method: "POST", path: "/admin/export-tokens", description: "批量导出所有用户的 API Token。对于尚未生成 Token 的用户，会自动生成。", auth: "必须（管理员）", inputParams: [], outputParams: [{ name: "id", type: "uint", description: "用户 ID" }, { name: "username", type: "string", description: "用户名" }, { name: "token", type: "string", description: "API Token" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" https://example.com/api/admin/export-tokens`, responseExample: `[{"id": 1, "username": "admin", "token": "hk-xxxxxxxxxxxxxxxxxxxx"}, {"id": 2, "username": "user1", "token": "hk-yyyyyyyyyyyyyyyyyyyy"}]`, errorCodes: [{ code: "403", error: "需要管理员权限" }] },
  { method: "GET", path: "/admin/user-limit", description: "查询当前用户数量和用户数上限。", auth: "必须（管理员）", inputParams: [], outputParams: [{ name: "count", type: "int64", description: "当前用户数量（含已封禁）" }, { name: "limit", type: "int", description: "用户数上限，0 表示不限" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" https://example.com/api/admin/user-limit`, responseExample: `{"count": 50, "limit": 1000}`, errorCodes: [{ code: "403", error: "需要管理员权限" }] },
  { method: "GET", path: "/admin/user-vpc", description: "查询指定用户自动创建的 VPC 信息及其资源占用情况。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "用户 ID（Query 参数）" }], outputParams: [{ name: "vpc_id", type: "string", description: "VPC ID" }, { name: "has_resources", type: "bool", description: "VPC 下是否有资源占用" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/user-vpc?id=2"`, responseExample: `{"vpc_id": "vpc-xxxxxxxx", "has_resources": false}`, errorCodes: [{ code: "404", error: "用户不存在" }] },
  { method: "GET", path: "/admin/departments", description: "查询部门列表和部门树结构。", auth: "必须（管理员）", inputParams: [], outputParams: [{ name: "departments", type: "array", description: "主部门名称列表" }, { name: "department_tree", type: "array", description: "部门树结构列表" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" https://example.com/api/admin/departments`, responseExample: `{"departments": ["技术部", "产品部"], "department_tree": [{"id": "100", "name": "技术部", "path": "公司/技术部", "parent_id": "1", "has_child": true}]}`, errorCodes: [{ code: "403", error: "需要管理员权限" }] },
  { method: "POST", path: "/admin/token/disable", description: "管理员禁用指定用户的 API Token。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "用户 ID（Query 参数）" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/token/disable?id=5"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "该用户没有 API Token / 该用户 Token 已处于禁用状态" }, { code: "404", error: "用户不存在" }] },
  { method: "POST", path: "/admin/token/enable", description: "管理员启用指定用户已被禁用的 API Token。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "用户 ID（Query 参数）" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/token/enable?id=5"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "该用户没有 API Token / 该用户 Token 未被禁用" }, { code: "404", error: "用户不存在" }] },
];

// ────────────────────────────────────────
// 管理接口 — 通道管理
// ────────────────────────────────────────

export const adminChannelEndpoints: EndpointDetail[] = [
  { method: "GET", path: "/admin/channels", description: "查询系统中所有 AI 通道列表。", auth: "必须（管理员）", inputParams: [], outputParams: [{ name: "channels", type: "array", description: "通道列表" }, { name: "channels[].ID", type: "int", description: "通道记录 ID" }, { name: "channels[].ChannelID", type: "string", description: "通道标识" }, { name: "channels[].Name", type: "string", description: "通道名称" }, { name: "channels[].Enabled", type: "bool", description: "是否启用" }, { name: "channels[].Custom", type: "bool", description: "是否为自定义通道" }, { name: "channels[].CustomConfig", type: "string", description: "自定义通道配置 JSON（仅自定义通道有值）" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/channels"`, responseExample: `{"channels": [{"ID": 1, "ChannelID": "wechat", "Name": "微信", "Enabled": true, "Custom": false}]}`, errorCodes: [{ code: "403", error: "需要管理员权限" }] },
  { method: "POST", path: "/admin/channels/toggle", description: "切换指定通道的启用/禁用状态。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "通道记录 ID（Query 参数）" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/channels/toggle?id=1"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "403", error: "需要管理员权限" }, { code: "404", error: "通道不存在" }] },
  { method: "POST", path: "/admin/channels/add", description: "添加自定义通道。创建后默认为禁用状态。", auth: "必须（管理员）", contentType: "application/json", inputParams: [{ name: "channel_id", type: "string", required: "是", description: "通道标识" }, { name: "name", type: "string", required: "是", description: "通道显示名称" }, { name: "custom_config", type: "object", required: "是", description: "自定义通道配置" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }, { name: "channel", type: "object", description: "创建的通道对象" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -H "Content-Type: application/json" -d '{"channel_id":"my_channel","name":"自定义通道","custom_config":{"server":{"url":"https://im.example.com"}}}' "https://example.com/api/admin/channels/add"`, responseExample: `{"ok": true, "channel": {"ID": 6, "ChannelID": "my_channel", "Name": "自定义通道", "Enabled": false, "Custom": true}}`, errorCodes: [{ code: "400", error: "Channel ID 不能为空 / 通道名称不能为空" }, { code: "409", error: "Channel ID 已存在" }] },
  { method: "POST", path: "/admin/channels/delete", description: "删除指定的自定义通道。预定义通道不允许删除。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "通道记录 ID（Query 参数）" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/channels/delete?id=6"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "缺少参数 id" }, { code: "403", error: "预定义通道不允许删除" }, { code: "404", error: "通道不存在" }] },
];

// ────────────────────────────────────────
// 管理接口 — 模型管理
// ────────────────────────────────────────

export const adminModelEndpoints: EndpointDetail[] = [
  { method: "GET", path: "/admin/models", description: "查询系统中所有 AI 模型的列表。", auth: "必须（管理员）", inputParams: [], outputParams: [{ name: "models", type: "array", description: "模型列表" }, { name: "models[].ID", type: "int", description: "模型记录 ID" }, { name: "models[].Provider", type: "string", description: "模型提供商" }, { name: "models[].ModelID", type: "string", description: "模型 ID" }, { name: "models[].ModelName", type: "string", description: "模型显示名称" }, { name: "models[].URL", type: "string", description: "API 地址" }, { name: "models[].ModelType", type: "string", description: "接口类型" }, { name: "models[].ContextLen", type: "int", description: "上下文长度" }, { name: "models[].QuotaDay", type: "int", description: "每日 Token 上限，-1 表示不限" }, { name: "models[].Enabled", type: "bool", description: "是否启用" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/models"`, responseExample: `{"models": [{"ID": 1, "Provider": "custom", "ModelID": "gpt-4o", "ModelName": "GPT-4o", "Enabled": true}]}`, errorCodes: [{ code: "403", error: "需要管理员权限" }] },
  { method: "POST", path: "/admin/models/create", description: "创建新的自定义 AI 模型。", auth: "必须（管理员）", contentType: "application/x-www-form-urlencoded", inputParams: [{ name: "model_id", type: "string", required: "是", description: "模型 ID" }, { name: "model_name", type: "string", required: "否", description: "显示名称" }, { name: "api_key", type: "string", required: "是", description: "API Key" }, { name: "url", type: "string", required: "是", description: "API 地址" }, { name: "model_type", type: "string", required: "是", description: "接口类型" }, { name: "quota_day", type: "int", required: "是", description: "每日 Token 上限，-1 不限" }, { name: "context_len", type: "int", required: "否", description: "上下文长度，默认 128000" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -d "model_id=gpt-4o&api_key=sk-xxx&url=https://api.openai.com/v1&model_type=openai-completions&quota_day=-1" "https://example.com/api/admin/models/create"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "模型ID、API Key、URL 和接口类型不能为空" }] },
  { method: "POST", path: "/admin/models/update", description: "更新自定义 AI 模型配置。系统内置记录不可修改。", auth: "必须（管理员）", contentType: "application/x-www-form-urlencoded", inputParams: [{ name: "id", type: "string", required: "是", description: "模型记录 ID（Query 参数）" }, { name: "model_id", type: "string", required: "是", description: "模型 ID" }, { name: "api_key", type: "string", required: "否", description: "API Key" }, { name: "url", type: "string", required: "是", description: "API 地址" }, { name: "model_type", type: "string", required: "是", description: "接口类型" }, { name: "quota_day", type: "int", required: "是", description: "每日 Token 上限" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -d "model_id=gpt-4o&url=https://api.openai.com/v1&model_type=openai-completions&quota_day=-1" "https://example.com/api/admin/models/update?id=2"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "403", error: "系统内置记录不可修改" }, { code: "404", error: "模型不存在" }] },
  { method: "POST", path: "/admin/models/delete", description: "删除指定的自定义 AI 模型。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "模型记录 ID（Query 参数）" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/models/delete?id=2"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "403", error: "系统内置记录不可删除" }, { code: "404", error: "模型不存在" }] },
  { method: "POST", path: "/admin/models/toggle", description: "切换指定 AI 模型的启用/禁用状态。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "模型记录 ID（Query 参数）" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/models/toggle?id=2"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "404", error: "模型不存在" }] },
  { method: "POST", path: "/admin/models/toggle-default", description: "切换指定 AI 模型的默认状态。设为默认时要求模型已启用。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "模型 ID（Query 参数）" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/models/toggle-default?id=3"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "请先开启该模型的「用户可见」后再设为默认" }, { code: "404", error: "模型不存在" }] },
];

// ────────────────────────────────────────
// 管理接口 — 镜像管理
// ────────────────────────────────────────

export const adminImageEndpoints: EndpointDetail[] = [
  { method: "GET", path: "/admin/images", description: "查询系统中所有已导入的镜像列表。", auth: "必须（管理员）", inputParams: [], outputParams: [{ name: "images", type: "array", description: "镜像列表" }, { name: "images[].id", type: "int", description: "镜像记录 ID" }, { name: "images[].image_id", type: "string", description: "云镜像 ID" }, { name: "images[].image_name", type: "string", description: "镜像名称" }, { name: "images[].image_type", type: "string", description: "镜像类型" }, { name: "images[].os_name", type: "string", description: "操作系统名称" }, { name: "images[].image_size", type: "int", description: "镜像大小（GB）" }, { name: "images[].image_state", type: "string", description: "镜像状态" }, { name: "images[].enabled", type: "bool", description: "是否启用" }, { name: "images[].public", type: "bool", description: "是否为公共镜像" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/images"`, responseExample: `{"images": [{"id": 1, "image_id": "img-xxxxxxxx", "image_name": "Ubuntu 22.04", "enabled": true, "public": false}]}`, errorCodes: [{ code: "403", error: "需要管理员权限" }] },
  { method: "GET", path: "/admin/images/cloud", description: "查询云端可导入的私有镜像列表，已导入的镜像会被过滤。", auth: "必须（管理员）", inputParams: [], outputParams: [{ name: "imageId", type: "string", description: "云镜像 ID" }, { name: "imageName", type: "string", description: "镜像名称" }, { name: "osName", type: "string", description: "操作系统名称" }, { name: "imageState", type: "string", description: "镜像状态" }, { name: "public", type: "bool", description: "是否为配置的公共镜像" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/images/cloud"`, responseExample: `[{"imageId": "img-yyyyyyyy", "imageName": "CentOS 7.9", "osName": "CentOS 7.9 64bit", "imageState": "NORMAL"}]`, errorCodes: [{ code: "403", error: "需要管理员权限" }, { code: "500", error: "查询镜像失败" }] },
  { method: "POST", path: "/admin/images/import", description: "从云端导入指定镜像到系统中。镜像大小不能超过 50GB。", auth: "必须（管理员）", contentType: "application/x-www-form-urlencoded", inputParams: [{ name: "image_id", type: "string", required: "是", description: "云镜像 ID" }, { name: "image_name", type: "string", required: "否", description: "自定义镜像名称" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -d "image_id=img-yyyyyyyy" "https://example.com/api/admin/images/import"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "镜像 ID 不能为空 / 镜像大小不能超过 50GB" }, { code: "409", error: "镜像 ID 已存在" }] },
  { method: "POST", path: "/admin/images/delete", description: "删除指定的已导入镜像。启用状态的镜像不能删除，需先禁用。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "镜像记录 ID（Query 参数）" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/images/delete?id=1"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "403", error: "启用状态的镜像不能删除，请先禁用" }, { code: "404", error: "镜像不存在" }] },
  { method: "POST", path: "/admin/images/enable", description: "切换指定镜像的启用/禁用状态。同一时间只能有一个镜像处于启用状态。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "镜像记录 ID（Query 参数）" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/images/enable?id=1"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "404", error: "镜像不存在" }] },
];

// ────────────────────────────────────────
// 管理接口 — 实例监控
// ────────────────────────────────────────

export const adminInstanceEndpoints: EndpointDetail[] = [
  { method: "GET", path: "/admin/instances", description: "分页查询系统中所有用户的实例列表，支持关键词搜索。", auth: "必须（管理员）", inputParams: [{ name: "page", type: "int", required: "否", description: "页码，默认 1" }, { name: "page_size", type: "int", required: "否", description: "每页条数，默认 20，最大 100" }, { name: "keyword", type: "string", required: "否", description: "搜索关键词" }, { name: "fuzzy", type: "string", required: "否", description: "设为 1 启用模糊搜索" }], outputParams: [{ name: "instances", type: "array", description: "实例列表" }, { name: "page", type: "int", description: "当前页码" }, { name: "total", type: "int", description: "总记录数" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/instances?page=1&page_size=20"`, responseExample: `{"instances": [{"ID": 1, "Name": "test", "Username": "user1", "InstanceId": "ins-xxx", "Status": "running"}], "page": 1, "total": 1}`, errorCodes: [{ code: "403", error: "需要管理员权限" }] },
  { method: "POST", path: "/admin/instances/delete", description: "管理员强制删除指定实例，同时销毁关联的云实例。", auth: "必须（管理员）", contentType: "application/x-www-form-urlencoded", inputParams: [{ name: "id", type: "string", required: "是", description: "实例记录 ID" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -d "id=1" "https://example.com/api/admin/instances/delete"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "无效的实例 ID" }, { code: "404", error: "实例不存在" }] },
  { method: "POST", path: "/admin/instances/terminal-url", description: "管理员获取指定实例的终端授权访问 URL。", auth: "必须（管理员）", contentType: "application/x-www-form-urlencoded", inputParams: [{ name: "id", type: "string", required: "是", description: "实例记录 ID" }], outputParams: [{ name: "login_url", type: "string", description: "终端授权登录 URL" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -d "id=1" "https://example.com/api/admin/instances/terminal-url"`, responseExample: `{"login_url": "https://orcaterm.cloud.tencent.com/terminal?token=xxxxx"}`, errorCodes: [{ code: "400", error: "无效的实例 ID / 该实例无关联的 CVM" }, { code: "404", error: "实例不存在" }] },
  { method: "POST", path: "/admin/instances/denied-actions", description: "管理员批量查询指定实例对应云实例的禁用操作。", auth: "必须（管理员）", contentType: "application/json", inputParams: [{ name: "ids", type: "array<int>", required: "是", description: "实例记录 ID 列表" }], outputParams: [{ name: "instances", type: "array", description: "实例禁用操作列表" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -H "Content-Type: application/json" -d '{"ids": [1, 2]}' "https://example.com/api/admin/instances/denied-actions"`, responseExample: `{"instances": [{"id": 1, "denied_actions": []}, {"id": 2, "denied_actions": [{"action": "DescribeInstanceVncUrl", "code": "InstanceInArrear", "message": "实例欠费"}]}]}`, errorCodes: [{ code: "400", error: "请求体格式错误" }] },
  { method: "GET", path: "/admin/instances/status", description: "管理员查询指定实例的云实例运行状态。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "实例记录 ID（Query 参数）" }], outputParams: [{ name: "state", type: "string", description: "云实例运行状态" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/instances/status?id=1"`, responseExample: `{"state": "RUNNING"}`, errorCodes: [{ code: "400", error: "缺少参数 id" }, { code: "500", error: "查询云实例状态失败" }] },
  { method: "GET", path: "/admin/instances/channels", description: "管理员查询指定实例的已配置通道列表。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "实例记录 ID（Query 参数）" }], outputParams: [{ name: "channels", type: "array", description: "通道列表" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/instances/channels?id=1"`, responseExample: `{"channels": [{"channel_id": "wechat", "name": "微信", "enabled": true}]}`, errorCodes: [{ code: "400", error: "缺少参数 id" }, { code: "500", error: "查询实例通道列表失败" }] },
  { method: "GET", path: "/admin/instances/skills", description: "管理员查询指定实例的已安装技能列表。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "实例记录 ID（Query 参数）" }], outputParams: [{ name: "skills", type: "array", description: "技能列表" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/instances/skills?id=1"`, responseExample: `{"skills": [{"skill_id": "web_search", "name": "网页搜索", "enabled": true}]}`, errorCodes: [{ code: "400", error: "缺少参数 id" }, { code: "500", error: "查询实例技能列表失败" }] },
  { method: "POST", path: "/admin/instances/start", description: "管理员开机指定实例的云服务器。", auth: "必须（管理员）", contentType: "application/x-www-form-urlencoded", inputParams: [{ name: "id", type: "string", required: "是", description: "实例 ID" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -d "id=1" https://example.com/api/admin/instances/start`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "无效的实例 ID / 该实例无关联的 CVM" }, { code: "404", error: "实例不存在" }, { code: "409", error: "实例正在执行其他操作" }] },
  { method: "POST", path: "/admin/instances/stop", description: "管理员关机指定实例的云服务器。", auth: "必须（管理员）", contentType: "application/x-www-form-urlencoded", inputParams: [{ name: "id", type: "string", required: "是", description: "实例 ID" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -d "id=1" https://example.com/api/admin/instances/stop`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "无效的实例 ID / 该实例无关联的 CVM" }, { code: "404", error: "实例不存在" }, { code: "409", error: "实例正在执行其他操作" }] },
  { method: "POST", path: "/admin/instances/reboot", description: "管理员重启指定实例的云服务器。", auth: "必须（管理员）", contentType: "application/x-www-form-urlencoded", inputParams: [{ name: "id", type: "string", required: "是", description: "实例 ID" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -d "id=1" https://example.com/api/admin/instances/reboot`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "无效的实例 ID / 该实例无关联的 CVM" }, { code: "404", error: "实例不存在" }, { code: "409", error: "实例正在执行其他操作" }] },
  { method: "POST", path: "/admin/instances/reset", description: "管理员重装指定实例的云服务器系统。", auth: "必须（管理员）", contentType: "application/x-www-form-urlencoded", inputParams: [{ name: "id", type: "string", required: "是", description: "实例 ID" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -d "id=1" https://example.com/api/admin/instances/reset`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "无效的实例 ID / 该实例无关联的 CVM" }, { code: "404", error: "实例不存在" }, { code: "409", error: "实例正在执行其他操作" }, { code: "500", error: "未启用任何镜像，无法重装实例" }] },
];

// ────────────────────────────────────────
// 管理接口 — 技能包管理
// ────────────────────────────────────────

export const adminSkillBundleEndpoints: EndpointDetail[] = [
  { method: "POST", path: "/admin/skill-bundles/create", description: "创建新的技能包。技能包名称不允许重复。", auth: "必须（管理员）", contentType: "application/x-www-form-urlencoded", inputParams: [{ name: "name", type: "string", required: "是", description: "技能包名称" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }, { name: "id", type: "int", description: "创建的技能包 ID" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -d "name=自定义技能包" https://example.com/api/admin/skill-bundles/create`, responseExample: `{"ok": true, "id": 2}`, errorCodes: [{ code: "400", error: "技能包名称不能为空" }, { code: "409", error: "同名技能包已存在" }] },
  { method: "GET", path: "/admin/skill-bundles", description: "查询技能包列表，支持分页。", auth: "必须（管理员）", inputParams: [{ name: "page", type: "int", required: "否", description: "页码，默认 1" }, { name: "page_size", type: "int", required: "否", description: "每页条数，默认 20" }], outputParams: [{ name: "skill_bundles", type: "array", description: "技能包列表" }, { name: "skill_bundles[].id", type: "int", description: "技能包 ID" }, { name: "skill_bundles[].name", type: "string", description: "技能包名称" }, { name: "skill_bundles[].skill_count", type: "int", description: "包内技能数量" }, { name: "skill_bundles[].enabled", type: "bool", description: "是否启用" }, { name: "total", type: "int", description: "总记录数" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/skill-bundles?page=1"`, responseExample: `{"skill_bundles": [{"id": 1, "name": "通用技能包", "skill_count": 7, "enabled": true}], "page": 1, "total": 1}`, errorCodes: [{ code: "403", error: "需要管理员权限" }] },
  { method: "POST", path: "/admin/skill-bundles/delete", description: "删除指定技能包。技能包必须先被禁用才能删除。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "技能包 ID" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/skill-bundles/delete?id=2"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "缺少参数 id" }, { code: "404", error: "技能包不存在" }, { code: "409", error: "技能包正在生效中，需先禁用" }] },
  { method: "POST", path: "/admin/skill-bundles/toggle", description: "启用或禁用指定技能包。同一时间只能有一个技能包处于启用状态。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "技能包 ID" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/skill-bundles/toggle?id=1"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "404", error: "技能包不存在" }, { code: "409", error: "已有其他技能包处于启用状态，请先禁用" }] },
  { method: "GET", path: "/admin/skill-bundles/detail", description: "查询技能包详情及包内技能列表。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "技能包 ID（Query 参数）" }], outputParams: [{ name: "skill_bundle", type: "object", description: "技能包信息" }, { name: "skills", type: "array", description: "包内技能列表" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/skill-bundles/detail?id=1"`, responseExample: `{"skill_bundle": {"id": 1, "name": "通用技能包", "enabled": true}, "skills": [{"id": 1, "name": "openclaw-tavily-search", "slug": "openclaw-tavily-search", "version": "0.1.0"}]}`, errorCodes: [{ code: "400", error: "缺少参数 id" }, { code: "404", error: "技能包不存在" }] },
  { method: "POST", path: "/admin/skill-bundles/update-skills", description: "批量更新技能包内的技能，支持同时添加和移除。", auth: "必须（管理员）", contentType: "application/json", inputParams: [{ name: "id", type: "string", required: "是", description: "技能包 ID（Query 参数）" }, { name: "add", type: "array", required: "否", description: "要添加的技能列表" }, { name: "add[].id", type: "int", required: "是", description: "技能 ID" }, { name: "add[].source", type: "string", required: "是", description: "来源类型：public 或 enterprise" }, { name: "remove", type: "array", required: "否", description: "要移除的技能记录 ID 列表" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }, { name: "skill_count", type: "int", description: "更新后技能总数" }, { name: "added", type: "int", description: "实际添加的技能数量" }, { name: "removed", type: "int", description: "实际移除的技能数量" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -H "Content-Type: application/json" -d '{"add": [{"id": 1, "source": "public"}], "remove": [5]}' "https://example.com/api/admin/skill-bundles/update-skills?id=1"`, responseExample: `{"ok": true, "skill_count": 8, "added": 1, "removed": 1}`, errorCodes: [{ code: "400", error: "缺少参数 id / 请求体格式错误" }, { code: "404", error: "技能包不存在" }] },
];

// ────────────────────────────────────────
// 管理接口 — 技能收藏
// ────────────────────────────────────────

export const adminSkillFavEndpoints: EndpointDetail[] = [
  { method: "POST", path: "/admin/skills/favorite", description: "收藏一个公共技能。", auth: "必须（管理员）", contentType: "application/json", inputParams: [{ name: "name", type: "string", required: "是", description: "技能名称" }, { name: "slug", type: "string", required: "是", description: "技能标识" }, { name: "version", type: "string", required: "否", description: "技能版本" }, { name: "description", type: "string", required: "否", description: "技能描述" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }, { name: "skill_id", type: "int", description: "收藏记录 ID" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" -H "Content-Type: application/json" -d '{"name":"Web Search","slug":"openclaw-tavily-search"}' https://example.com/api/admin/skills/favorite`, responseExample: `{"ok": true, "skill_id": 1}`, errorCodes: [{ code: "400", error: "请求体格式错误 / name 和 slug 不能为空" }] },
  { method: "POST", path: "/admin/skills/unfavorite", description: "取消收藏指定的公共技能。", auth: "必须（管理员）", inputParams: [{ name: "id", type: "string", required: "是", description: "收藏记录 ID" }], outputParams: [{ name: "ok", type: "bool", description: "操作是否成功" }], requestExample: `curl -X POST -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/skills/unfavorite?id=1"`, responseExample: `{"ok": true}`, errorCodes: [{ code: "400", error: "缺少参数 id" }, { code: "404", error: "技能不存在" }] },
  { method: "GET", path: "/admin/skills/favorited", description: "查询已收藏的公共技能列表，支持分页。", auth: "必须（管理员）", inputParams: [{ name: "page", type: "int", required: "否", description: "页码，默认 1" }, { name: "page_size", type: "int", required: "否", description: "每页条数，默认 20" }], outputParams: [{ name: "skills", type: "array", description: "已收藏技能列表" }, { name: "skills[].id", type: "int", description: "收藏记录 ID" }, { name: "skills[].name", type: "string", description: "技能名称" }, { name: "skills[].slug", type: "string", description: "技能标识" }, { name: "total", type: "int", description: "总记录数" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/skills/favorited?page=1"`, responseExample: `{"skills": [{"id": 1, "name": "Web Search", "slug": "openclaw-tavily-search", "version": "0.1.0"}], "page": 1, "total": 1}`, errorCodes: [{ code: "403", error: "需要管理员权限" }] },
];

// ────────────────────────────────────────
// 管理接口 — 使用统计
// ────────────────────────────────────────

export const adminUsageEndpoints: EndpointDetail[] = [
  { method: "GET", path: "/admin/usage/data", description: "查询模型使用量统计数据，支持按日期、用户、模型、实例维度聚合。", auth: "必须（管理员）", inputParams: [{ name: "start_date", type: "string", required: "否", description: "起始日期（YYYY-MM-DD），默认当天" }, { name: "end_date", type: "string", required: "否", description: "结束日期" }, { name: "group_by", type: "string", required: "否", description: "组织维度（逗号分隔）：date, user, model, instance" }, { name: "user_id", type: "int", required: "否", description: "按用户 ID 过滤" }, { name: "order", type: "string", required: "否", description: "desc 按 total_tokens 降序" }], outputParams: [{ name: "start_date", type: "string", description: "查询起始日期" }, { name: "end_date", type: "string", description: "查询结束日期" }, { name: "group_by", type: "array<string>", description: "组织维度" }, { name: "rows", type: "array", description: "统计数据行" }, { name: "global_token_quota_day", type: "int", description: "全局每日 Token 配额" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/usage/data?start_date=2026-03-01&end_date=2026-03-30&group_by=date,model"`, responseExample: `{"start_date": "2026-03-01", "end_date": "2026-03-30", "group_by": ["date", "model"], "rows": [{"date": "2026-03-28", "ai_model_id": 1, "model_name": "gpt-4o", "total_tokens": 23000, "request_count": 50}], "global_token_quota_day": -1}`, errorCodes: [{ code: "403", error: "需要管理员权限" }, { code: "500", error: "查询用量数据失败" }] },
  { method: "GET", path: "/admin/usage/logs", description: "分页查询指定日期范围内的 LLM 使用明细日志。", auth: "必须（管理员）", inputParams: [{ name: "start_date", type: "string", required: "否", description: "起始日期" }, { name: "end_date", type: "string", required: "否", description: "结束日期" }, { name: "page", type: "int", required: "否", description: "页码，默认 1" }, { name: "page_size", type: "int", required: "否", description: "每页条数，默认 50" }, { name: "user_id", type: "int", required: "否", description: "按用户 ID 过滤" }], outputParams: [{ name: "logs", type: "array", description: "使用日志列表" }, { name: "logs[].id", type: "int", description: "日志 ID" }, { name: "logs[].user_name", type: "string", description: "用户名" }, { name: "logs[].model", type: "string", description: "模型名称" }, { name: "logs[].total_tokens", type: "int", description: "总 Token 数" }, { name: "total", type: "int", description: "总记录数" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/usage/logs?page=1&page_size=50"`, responseExample: `{"start_date": "2026-03-28", "page": 1, "page_size": 50, "total": 120, "logs": [{"id": 100, "user_name": "admin", "model": "gpt-4o", "total_tokens": 700, "status_code": 200}]}`, errorCodes: [{ code: "403", error: "需要管理员权限" }, { code: "500", error: "查询使用记录失败" }] },
];

// ────────────────────────────────────────
// 管理接口 — 审计日志
// ────────────────────────────────────────

export const adminAuditEndpoints: EndpointDetail[] = [
  { method: "GET", path: "/admin/audit", description: "分页查询系统操作审计日志，支持按用户名模糊搜索。", auth: "必须（管理员）", inputParams: [{ name: "page", type: "int", required: "否", description: "页码，默认 1" }, { name: "page_size", type: "int", required: "否", description: "每页条数，默认 20，最大 100" }, { name: "username", type: "string", required: "否", description: "按用户名模糊搜索" }], outputParams: [{ name: "logs", type: "array", description: "审计日志列表" }, { name: "logs[].id", type: "int", description: "日志 ID" }, { name: "logs[].started_at", type: "string", description: "操作开始时间" }, { name: "logs[].username", type: "string", description: "操作用户名" }, { name: "logs[].action", type: "string", description: "操作类型" }, { name: "logs[].resource", type: "string", description: "操作资源" }, { name: "logs[].status", type: "string", description: "操作状态" }, { name: "page", type: "int", description: "当前页码" }, { name: "total", type: "int", description: "总记录数" }], requestExample: `curl -H "Authorization: Bearer hk-xxx" "https://example.com/api/admin/audit?page=1&page_size=20"`, responseExample: `{"logs": [{"id": 1, "started_at": "2026-03-28T14:00:00Z", "username": "admin", "action": "创建用户", "resource": "user", "status": "success"}], "page": 1, "total": 50}`, errorCodes: [{ code: "403", error: "需要管理员权限" }] },
];

// ────────────────────────────────────────
// 汇总导出
// ────────────────────────────────────────

export const adminEndpointsBySection: Record<string, EndpointDetail[]> = {
  "admin-site": adminSiteEndpoints,
  "admin-security": adminSecurityEndpoints,
  "admin-user": adminUserEndpoints,
  "admin-channel": adminChannelEndpoints,
  "admin-model": adminModelEndpoints,
  "admin-image": adminImageEndpoints,
  "admin-instance": adminInstanceEndpoints,
  "admin-skill-bundle": adminSkillBundleEndpoints,
  "admin-skill-fav": adminSkillFavEndpoints,
  "admin-usage": adminUsageEndpoints,
  "admin-audit": adminAuditEndpoints,
};
