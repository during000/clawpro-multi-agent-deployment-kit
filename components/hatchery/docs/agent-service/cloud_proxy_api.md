# ClawPro独立站腾讯云API通用透传接口

## 1. 概述

Cloud Proxy 是一个**通用的腾讯云 API 透传网关**，前端无需持有云 API 密钥，只需按照腾讯云 API 3.0 标准格式构造请求，后端自动完成凭证注入、签名和转发。

**核心特点：**

- **请求/响应格式与腾讯云 API 完全一致**：前端可直接参照 [腾讯云官方 API 文档](https://cloud.tencent.com/document/api) 构造请求
- **白名单机制**：只有在后端 `cloudProxyRegistry` 中注册的 Action 才能透传，防止越权调用
- **读写分离**：查询接口（`query`）和变更接口（`mutate`）走不同路由，变更接口强制审计
- **零适配扩展**：新增云产品/接口只需后端注册白名单，无需编写额外 handler

---

## 2. 路由总览

| 方法 | 路由 | 说明 | 审计 |
|------|------|------|------|
| `GET` | `/admin/cloud` | 列出所有可用 service 及其 Actions（读/写分类） | 无 |
| `POST` | `/admin/cloud/query/{service}` | **只读查询**：Describe/Inquiry 等查询类 API | 无 |
| `POST` | `/admin/cloud/mutate/{service}` | **变更操作**：Create/Delete/Modify 等写类 API | ✅ 有 |

> `{service}` 为腾讯云产品标识，如 `cvm`、`vpc`、`cls`、`billing`、`csip`、`cwp`、`vdb`、`smh`。

---

## 3. 认证方式

所有 `/admin/` 路由需要管理员身份认证，支持方式：

- **Session 认证**：通过 `/login` 登录获得 Session Cookie

---

## 4. 请求格式

请求格式复用 [腾讯云 API 3.0 公共参数规范](https://cloud.tencent.com/document/api/213/15692)，通过 HTTP Header 传递控制参数：

### 4.1 请求 Header

| Header | 必选 | 说明 | 示例 |
|--------|------|------|------|
| `X-TC-Action` | **是** | 要调用的 API 名称 | `DescribeInstances` |
| `X-TC-Version` | 否 | API 版本号，默认使用注册表中的版本 | `2017-03-12` |
| `Content-Type` | 是 | 固定为 `application/json` | `application/json` |

> 注：无需传入`X-TC-Region` 会使用服务端启动时 `--region` 配置

### 4.2 请求 Body

请求体为标准 JSON，**与腾讯云官方 API 文档中对应接口的请求参数完全一致**。

如果不需要传任何参数，可以发送空对象 `{}`。

### 4.3 响应格式

响应为腾讯云 API 原始返回的 JSON，**原样透传，不做任何包装**。

成功响应示例：
```json
{
  "Response": {
    "TotalCount": 1,
    "InstanceSet": [...],
    "RequestId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
}
```

错误响应示例（HTTP 状态码仍为 200，与腾讯云标准行为一致）：
```json
{
  "Response": {
    "Error": {
      "Code": "InvalidParameterValue",
      "Message": "参数错误描述"
    },
    "RequestId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
}
```

**代理层自身的错误**（如 Action 不在白名单、缺少参数等）以标准 JSON 返回，HTTP 状态码为 4xx：
```json
{
  "error": "Action \"RunInstances\" 不在 cvm 的读接口白名单中, 允许的 Actions: ..."
}
```

---

## 5. 当前已注册的 Actions

### 5.1 CVM（云服务器） — 版本 `2017-03-12`

| 类型 | Action | 说明 |
|------|--------|------|
| 读 | `DescribeInstances` | 查询实例列表 |
| 写 | `AssociateSecurityGroups` | 绑定安全组 |
| 写 | `DisassociateSecurityGroups` | 解绑安全组 |

### 5.2 VPC（私有网络） — 版本 `2017-03-12`

| 类型 | Action | 说明 |
|------|--------|------|
| 读 | `DescribeSecurityGroups` | 查询安全组列表 |
| 读 | `DescribeSecurityGroupPolicies` | 查询安全组规则 |
| 写 | `DeleteSecurityGroupPolicies` | 删除安全组规则 |
| 写 | `DeleteSecurityGroup` | 删除安全组 |
| 写 | `CreateSecurityGroupWithPolicies` | 创建安全组并设置规则 |
| 写 | `CreateSecurityGroupPolicies` | 添加安全组规则 |

### 5.3 CLS（日志服务） — 版本 `2020-10-16`

| 类型 | Action | 说明 |
|------|--------|------|
| 读 | `DescribeLogsets` | 查询日志集列表 |
| 读 | `DescribeTopics` | 查询日志主题列表 |
| 读 | `SearchLog` | 检索日志 |
| 读 | `QueryMetric` | 查询指标 |
| 读 | `QueryRangeMetric` | 查询范围指标 |
| 读 | `GetClsService` | 查询 CLS 服务开通状态 |
| 读 | `DescribeRainbowConfigs` | 查询七彩石（Rainbow）配置列表 |
| 读 | `DescribeTemplates` | 查询模板列表 |
| 写 | `OpenClsService` | 开通 CLS 服务 |
| 写 | `OpenClawService` | 开通 OpenClaw 服务 |
| 写 | `DeleteTopic` | 删除日志主题 |

### 5.4 CSIP（云安全中心） — 版本 `2022-11-21`

| 类型 | Action | 说明 |
|------|--------|------|
| 读 | `DescribeCVMAssets` | 查询 CVM 资产列表 |
| 读 | `DescribeVulRiskList` | 查询漏洞风险列表 |
| 读 | `DescribeExposures` | 查询暴露面列表 |
| 读 | `DescribeExposeRules` | 查询暴露面规则 |
| 读 | `DescribeExposeAssetCategory` | 查询暴露面资产分类 |
| 读 | `DescribeExposePath` | 查询暴露面路径 |
| 读 | `DescribeAIAgentAssetList` | 查询 AI Agent 资产列表 |
| 读 | `DescribeAgentlessVulAssetDetail` | 查询无 Agent 漏洞资产详情 |
| 读 | `DescribeHighBaseLineRiskList` | 查询高危基线风险列表 |
| 读 | `DescribeAssetProcessList` | 查询资产进程列表 |
| 读 | `DescribeABTestConfig` | 查询 ABTest 配置 |
| 读 | `DescribeOrganizationInfo` | 查询组织信息 |
| 读 | `DescribePayInfo` | 查询付费信息 |
| 读 | `DescribeUserAccountInfo` | 查询用户账号信息 |
| 读 | `DescribeUserOperationPermission` | 查询用户操作权限 |
| 读 | `GetLocalStorageItem` | 获取本地存储项 |
| 读 | `DescribeAgentlessVulRiskList` | 查询无 Agent 漏洞风险列表 |
| 读 | `DescribeAgentlessVulAssetList` | 查询无 Agent 漏洞资产列表 |
| 读 | `DescribeKeySandboxCredentialList` | 查询密钥沙箱凭据列表 |
| 读 | `DescribeKeySandboxCredential` | 查询密钥沙箱凭据详情 |
| 读 | `DescribeExportJobDownloadURL` | 获取导出任务下载链接 |
| 读 | `DescribeSkillScanResult` | 获取恶意文件扫描结果 |
| 读 | `DescribeTrialStatus` | 获取试用状态 |
| 读 | `DescribeSkillScanPayInfo` | 获取试用数据 |
| 写 | `SetLocalStorageItem` | 设置本地存储项 |
| 写 | `CreateScanTask` | 创建扫描任务 |
| 写 | `ApplyTrial` | 申请试用 |
| 写 | `CreateKeySandboxCredential` | 创建密钥沙箱凭据 |
| 写 | `ModifyKeySandboxCredential` | 修改密钥沙箱凭据 |
| 写 | `InstallKeySandboxSkill` | 安装密钥沙箱技能 |
| 写 | `UninstallKeySandboxSkill` | 卸载密钥沙箱技能 |
| 写 | `DeleteKeySandboxCredential` | 删除密钥沙箱凭据 |
| 写 | `CreateExposuresExportJob` | 创建暴露面导出任务 |
| 写 | `CreateSkillScan` | 创建扫描任务 |
| 写 | `ModifyTrialStatus` | 开启试用 |

### 5.5 Billing（计费） — 版本 `2018-07-09`

| 类型 | Action | 说明 |
|------|--------|------|
| 读 | `DescribeMeasureResources` | 查询用户购买的套餐包详情列表 |
| 写 | `CreateOrdersAndPay` | 下单并支付 |

### 5.6 CWP（主机安全） — 版本 `2018-02-28`

| 类型 | Action | 说明 |
|------|--------|------|
| 读 | `DescribeMachines` | 查询机器列表 |
| 读 | `DescribeMachineInfo` | 查询机器详情 |
| 读 | `DescribeMachineGeneral` | 查询机器概览 |
| 读 | `DescribeMachineRegionList` | 查询机器地域列表 |
| 读 | `DescribeVersionStatistics` | 查询版本统计 |
| 读 | `DescribeLicenseGeneral` | 查询授权概览 |
| 读 | `DescribeLicenseWhiteConfig` | 查询授权白名单配置 |
| 读 | `DescribeLicenseBindSchedule` | 查询授权绑定进度 |
| 读 | `DescribeOrderList` | 查询订单列表 |
| 读 | `DescribeBashEventsNew` | 查询高危命令事件 |
| 读 | `DescribeBashEventsInfoNew` | 查询高危命令事件详情 |
| 读 | `DescribeBashPolicies` | 查询高危命令策略 |
| 读 | `DescribeRiskDnsEventList` | 查询恶意请求事件列表 |
| 读 | `DescribeRiskDnsEventInfo` | 查询恶意请求事件详情 |
| 读 | `DescribeRiskDnsPolicyList` | 查询恶意请求策略列表 |
| 读 | `DescribeMalWareList` | 查询木马列表 |
| 读 | `DescribeMalwareInfo` | 查询木马详情 |
| 读 | `DescribeRiskBatchStatus` | 查询风险批量处理状态 |
| 读 | `DescribeSkillInfo` | 查询技能信息 |
| 读 | `DescribeImportMachineInfo` | 查询导入机器信息 |
| 读 | `DescribeTags` | 查询标签列表 |
| 读 | `DescribeLogStorageConfig` | 查询日志存储配置 |
| 读 | `DescribeLogHistogram` | 查询日志直方图 |
| 读 | `DescribeLogStorageStatistic` | 查询日志存储统计 |
| 读 | `SearchLog` | 检索日志 |
| 读 | `GetLocalStorageItem` | 获取本地存储项 |
| 读 | `DescribeVulList` | 查询漏洞列表 |
| 读 | `DescribeVulInfoCvss` | 查询漏洞 CVSS 详情 |
| 读 | `DescribeVulIgnoreRule` | 查询漏洞忽略规则 |
| 读 | `DescribeVulEffectHostList` | 查询漏洞影响主机列表 |
| 读 | `DescribeLicenseList` | 查询授权列表 |
| 读 | `DescribeGrayPolicy` | 查询灰度策略 |
| 读 | `DescribeUsersConfig` | 查询用户配置 |
| 读 | `DescribeMachinesSimple` | 查询机器简要列表 |
| 读 | `DescribeLicenseBindList` | 查询授权绑定列表 |
| 读 | `DescribeHostInfo` | 查询主机信息 |
| 读 | `DescribeBaselineItemDetectList` | 查询基线检测项列表 |
| 读 | `DescribeBaselineRuleDetectList` | 查询基线规则检测列表 |
| 读 | `DescribeBaselineHostDetectList` | 查询基线主机检测列表 |
| 读 | `DescribeBaselineItemList` | 查询基线项列表 |
| 读 | `DescribeIgnoreHostAndItemConfig` | 查询忽略主机和检测项配置 |
| 读 | `DescribeBaselineDownloadList` | 查询基线下载列表 |
| 读 | `DescribeBaselineRuleIgnoreList` | 查询基线规则忽略列表 |
| 读 | `DescribeAIAgentAutoOpenConfig` | 查询AI Agent自动设置 |
| 写 | `CreateWhiteListOrder` | 创建白名单订单 |
| 写 | `ModifyLicenseBinds` | 修改授权绑定 |
| 写 | `ModifyLogStorageConfig` | 修改日志存储配置 |
| 写 | `ScanAsset` | 扫描资产 |
| 写 | `SyncAssetScan` | 同步资产扫描 |
| 写 | `ModifyRiskEventsStatus` | 修改风险事件状态 |
| 写 | `SetLocalStorageItem` | 设置本地存储项 |
| 写 | `RemoveLocalStorageItem` | 移除本地存储项 |
| 写 | `ModifyBashPolicyStatus` | 修改高危命令策略状态 |
| 写 | `ModifyBashPolicy` | 修改高危命令策略 |
| 写 | `DeleteBashPolicies` | 删除高危命令策略 |
| 写 | `CheckBashPolicyParams` | 校验高危命令策略参数 |
| 写 | `ModifyRiskDnsPolicy` | 修改恶意请求策略 |
| 写 | `ModifyRiskDnsPolicyStatus` | 修改恶意请求策略状态 |
| 写 | `DeleteRiskDnsPolicy` | 删除恶意请求策略 |
| 写 | `ModifyReverseShellRulesAggregation` | 修改反弹 Shell 聚合规则 |
| 写 | `ScanVulAgain` | 重新扫描漏洞 |
| 写 | `ExportVulList` | 导出漏洞列表 |
| 写 | `ExportTasks` | 导出任务 |
| 写 | `ExportVulEffectHostList` | 导出漏洞影响主机列表 |
| 写 | `AddVulIgnoreRule` | 添加漏洞忽略规则 |
| 写 | `CancelVulIgnoreRule` | 取消漏洞忽略规则 |
| 写 | `ExportRiskDnsPolicyList` | 导出恶意请求策略列表 |
| 写 | `ExportBashPolicies` | 导出高危命令策略 |
| 写 | `StartBaselineDetect` | 启动基线检测 |
| 写 | `SyncBaselineDetectSummary` | 同步基线检测摘要 |
| 写 | `ModifyBaselineRuleIgnore` | 修改基线规则忽略 |
| 写 | `ExportBaselineItemList` | 导出基线项列表 |
| 写 | `ExportBaselineItemDetectList` | 导出基线检测项列表 |
| 写 | `ModifyAIAgentAutoOpenConfig` | 修改AI Agent自动加购设置 |

### 5.7 VDB（向量数据库） — 版本 `2023-06-16`

| 类型 | Action | 说明 |
|------|--------|------|
| 写 | `CreateInstance` | 创建实例 |
| 写 | `ScaleOutInstance` | 水平扩容实例 |
| 写 | `ScaleUpInstance` | 垂直扩容实例 |

### 5.8 SMH（智能媒资托管） — 版本 `2021-07-12`

| 类型 | Action | 说明 |
|------|--------|------|
| 读 | `DescribeLibrarySecret` | 查询媒体库密钥 |
| 读 | `DescribeLibraries` | 查询媒体库列表 |
| 写 | `CreateLibrary` | 创建媒体库 |
| 写 | `ModifyLibrary` | 修改媒体库 |
| 写 | `DeleteLibrary` | 删除媒体库 |

> 运行时可通过 `GET /admin/cloud` 获取最新的完整列表。

---

## 6. 调用示例

### 6.1 查询 CVM 实例列表

```bash
curl -X POST 'https://{host}/admin/cloud/query/cvm' \
  -H 'Content-Type: application/json' \
  -H 'X-TC-Action: DescribeInstances' \
  -d '{
    "Offset": 0,
    "Limit": 20,
    "Filters": [
      {
        "Name": "instance-state",
        "Values": ["RUNNING"]
      }
    ]
  }'
```

**响应：**
```json
{
  "Response": {
    "TotalCount": 2,
    "InstanceSet": [
      {
        "InstanceId": "ins-xxxxxxxx",
        "InstanceName": "my-server",
        "InstanceState": "RUNNING",
        "PublicIpAddresses": ["1.2.3.4"],
        "PrivateIpAddresses": ["10.0.0.1"]
      }
    ],
    "RequestId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  }
}
```

### 6.2 查询安全组列表

```bash
curl -X POST 'https://{host}/admin/cloud/query/vpc' \
  -H 'Content-Type: application/json' \
  -H 'X-TC-Action: DescribeSecurityGroups' \
  -d '{
    "Offset": "0",
    "Limit": "20"
  }'
```

### 6.3 查询安全组规则详情

```bash
curl -X POST 'https://{host}/admin/cloud/query/vpc' \
  -H 'Content-Type: application/json' \
  -H 'X-TC-Action: DescribeSecurityGroupPolicies' \
  -d '{
    "SecurityGroupId": "sg-xxxxxxxx"
  }'
```

### 6.4 创建安全组（写操作）

```bash
curl -X POST 'https://{host}/admin/cloud/mutate/vpc' \
  -H 'Content-Type: application/json' \
  -H 'X-TC-Action: CreateSecurityGroupWithPolicies' \
  -d '{
    "GroupName": "web-sg",
    "GroupDescription": "Web 服务安全组",
    "SecurityGroupPolicySet": {
      "Ingress": [
        {
          "Protocol": "TCP",
          "Port": "80,443",
          "CidrBlock": "0.0.0.0/0",
          "Action": "ACCEPT",
          "PolicyDescription": "允许 HTTP/HTTPS"
        },
        {
          "Protocol": "TCP",
          "Port": "22",
          "CidrBlock": "10.0.0.0/8",
          "Action": "ACCEPT",
          "PolicyDescription": "内网 SSH"
        }
      ],
      "Egress": [
        {
          "Protocol": "ALL",
          "Port": "ALL",
          "CidrBlock": "0.0.0.0/0",
          "Action": "ACCEPT",
          "PolicyDescription": "允许所有出站"
        }
      ]
    }
  }'
```

### 6.5 检索 CLS 日志

```bash
curl -X POST 'https://{host}/admin/cloud/query/cls' \
  -H 'Content-Type: application/json' \
  -H 'X-TC-Action: SearchLog' \
  -d '{
    "TopicId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "From": 1710000000000,
    "To": 1710003600000,
    "Query": "level:ERROR",
    "Limit": 20
  }'
```

### 6.6 查询主机安全机器列表

```bash
curl -X POST 'https://{host}/admin/cloud/query/cwp' \
  -H 'Content-Type: application/json' \
  -H 'X-TC-Action: DescribeMachines' \
  -d '{
    "MachineType": "CVM",
    "MachineRegion": "ap-guangzhou",
    "Offset": 0,
    "Limit": 20
  }'
```

### 6.7 查询 CSIP 暴露面

```bash
curl -X POST 'https://{host}/admin/cloud/query/csip' \
  -H 'Content-Type: application/json' \
  -H 'X-TC-Action: DescribeExposures' \
  -d '{
    "Filter": {}
  }'
```

### 6.8 查询用户套餐包列表

```bash
curl -X POST 'https://{host}/admin/cloud/query/billing' \
  -H 'Content-Type: application/json' \
  -H 'X-TC-Action: DescribeMeasureResources' \
  -d '{}'
```

### 6.9 下单并支付

```bash
curl -X POST 'https://{host}/admin/cloud/mutate/billing' \
  -H 'Content-Type: application/json' \
  -H 'X-TC-Action: CreateOrdersAndPay' \
  -d '{
    "Goods": [...]
  }'
```

### 6.10 获取所有可用 Actions

```bash
curl 'https://{host}/admin/cloud'
```

**响应：**
```json
{
  "data": {
    "cvm": {
      "service": "cvm",
      "version": "2017-03-12",
      "read_actions": ["DescribeInstances"],
      "write_actions": ["AssociateSecurityGroups", "DisassociateSecurityGroups"]
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
      "read_actions": ["DescribeLogsets", "DescribeTopics", "SearchLog", "QueryMetric", "QueryRangeMetric", "GetClsService", "DescribeRainbowConfigs", "DescribeTemplates"],
      "write_actions": ["OpenClsService", "OpenClawService", "DeleteTopic"]
    },
    "billing": {
      "service": "billing",
      "version": "2018-07-09",
      "read_actions": ["DescribeMeasureResources"],
      "write_actions": ["CreateOrdersAndPay"]
    },
    "csip": { "..." },
    "cwp": { "..." },
    "vdb": {
      "service": "vdb",
      "version": "2023-06-16",
      "read_actions": [],
      "write_actions": ["CreateInstance", "ScaleOutInstance", "ScaleUpInstance"]
    },
    "smh": {
      "service": "smh",
      "version": "2021-07-12",
      "read_actions": ["DescribeLibrarySecret", "DescribeLibraries"],
      "write_actions": ["CreateLibrary", "ModifyLibrary", "DeleteLibrary"]
    }
  }
}
```

### 6.11 创建向量数据库实例（VDB 写操作）

```bash
curl -X POST 'https://{host}/admin/cloud/mutate/vdb' \
  -H 'Content-Type: application/json' \
  -H 'X-TC-Action: CreateInstance' \
  -d '{
    "Name": "my-vector-db",
    "DbType": "VECTOR"
  }'
```

### 6.12 查询媒体库列表（SMH）

```bash
curl -X POST 'https://{host}/admin/cloud/query/smh' \
  -H 'Content-Type: application/json' \
  -H 'X-TC-Action: DescribeLibraries' \
  -d '{}'
```

---

## 7. 错误处理

| 场景 | HTTP 状态码 | 说明 |
|------|------------|------|
| 缺少 `X-TC-Action` | 400 | `缺少 X-TC-Action Header 或 Action 参数` |
| 不支持的 service | 400 | `不支持的 service: xxx, 可用: cvm, vpc, ...` |
| Action 不在白名单 | 403 | `Action "xxx" 不在 cvm 的读接口白名单中` |
| 请求体非法 JSON | 400 | `请求体不是合法的 JSON` |
| 云 API 凭证错误 | 500 | `获取云 API 凭证失败: ...` |
| 腾讯云 API 返回错误 | 200 | 返回腾讯云标准错误格式 `Response.Error` |
| 未认证 | 401 | 需要管理员登录 |

---

## 8. 设计原则

1. **协议透明**：请求/响应格式完全复用 [腾讯云 API 3.0 公共参数规范](https://cloud.tencent.com/document/api/213/15692)，前端工程师可直接对照腾讯云官方文档开发
2. **安全隔离**：云 API 凭证（SecretId/SecretKey/STS Token）仅存在于后端，前端无法获取
3. **白名单控制**：只有显式注册的 Action 才能透传，未注册的一律拒绝
4. **读写分离**：查询接口走 `/query/`、变更接口走 `/mutate/`，变更接口强制审计记录
5. **零适配扩展**：新增云产品只需在 `cloudProxyRegistry` 添加配置，无需编写新 handler
