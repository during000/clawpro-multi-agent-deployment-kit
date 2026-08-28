# 01. Clarify — get-config 接口需求澄清

> 规格主源：iWiki 技术方案 `https://iwiki.woa.com/p/4022150701` §5.A.4（已定稿，定义阶段完成）。
> 本文件做 Discovery + 待确认问题梳理。规格已较完整，Clarify 重点在**对齐实现层面的歧义点**，而非重新定义需求。

---

## 一、背景与目标

本地 agent（reporter / agent 自身）在需要往 CLS 公网上报日志/指标/Trace 前，主动拉取接入配置。CVM 实例的 CLS 配置由 hatchery 经 TAT 脚本 + CAM Role 临时凭证下发；本地 agent 不在 VPC 内、无法用 CAM Role，因此本接口直接返回**公网域名 + 永久 AK/SK + topic**，本地 agent 用这些凭据直连 CLS 公网 endpoint 上报。

- **Path**：`GET /local-agent/get-config`
- **Query**：`config_type=cls`（一期仅 cls，其他类型 400）
- **鉴权**：Bearer user API token（与 §5.A.1 report 同源），前置过 `ensureLocalAgentAllowed` 两层白名单
- **性质**：GET 只读查询，不写实例状态

---

## 二、代码现状（Discovery 结果）

| 项 | 现状 | 结论 |
|---|---|---|
| `ensureLocalAgentAllowed` | 已存在 `controller/local_agent.go:136`，report/sync/ack 三处复用 | get-config **直接复用**，拒绝返 403 + i18n `openclaw.local_agent.not_allowed` |
| `CheckCLSClawServiceOpened` | 已存在 `controller/admin_cls.go:1174`，返回 `CLSClawServiceResult{TopicId, MetricTopicId, TraceTopicId, RequestId}`；服务未开通时返回 `(nil, nil)` | get-config 复用，取 `result.TopicId` 作为 `topic_id`；返回 nil 时按「未开通」处理 |
| `IsFeatureAllowed` | 已存在 `model/feature_allowlist.go:42` | `ensureLocalAgentAllowed` 内部已调用，无需新写 |
| `local_agent_cls_credentials` 表 | **不存在**（grep 无命中） | 需新建 model + `allModels` 注册 + `sql/init.sql` + 增量 migration + `MigrateFromSQLite` |
| 路由风格 | GET 只读接口用 `WithOpenAPI`（不带 `WithAudit`）；写接口才 `WithAudit` | get-config 用 `WithOpenAPI` 即可（只读查询） |
| `CVMRegion` | `controller.auth.go` 全局 `CVMRegion` | endpoint 拼 `<CVMRegion>.cls.tencentcs.com` |

---

## 三、Response 结构（iWiki 定稿，一期 config_type=cls）

扁平结构，最外层以 `config_type` 为 key，无额外包裹层：

```json
{
  "cls": {
    "endpoint": "ap-guangzhou.cls.tencentcs.com",
    "topic_id": "clawpro-topic-ap-guangzhou-open-1254139626",
    "secret_id": "AKIDxx…xxxx",
    "secret_key": "xxxxxxxxxxxxxxxxxxxxxxxx",
    "user_id": 12345,
    "user_name": "alexwhwang",
    "install_cmd": "<待定值>",
    "update_cmd": "<待定值>"
  }
}
```

> `install_cmd` / `update_cmd` 为用户 2026-07-14 补充的**固定值字段**，值待用户提供后定稿。

字段来源：
- `endpoint`：`fmt.Sprintf("%s.cls.tencentcs.com", controller.CVMRegion)`（**公网** tencentcs.com，非内网 tencentyun.com）
- `topic_id`：`CheckCLSClawServiceOpened(ctx).TopicId`（实时查，不落库）
- `secret_id` / `secret_key`：从 `local_agent_cls_credentials` 表按 `(identifier, config_type=cls)` 查一行
- `user_id` / `user_name`：当前调用用户（`user.ID` / `user.Username`，token 反查）
- `install_cmd` / `update_cmd`：固定值（待定）

---

## 四、后端逻辑（iWiki 定稿）

```
1. 鉴权 + 两层白名单（复用 ensureLocalAgentAllowed）：
   ① feature_allowlist type='local-agent' 无记录=全开，有记录=仅表内 identifier 放行
   ② SiteConfig.LocalAgentEnabled == true
   任一层拒绝 → 403 (openclaw.local_agent.not_allowed)
2. 校验 config_type（一期仅 "cls"），否则 400 (openclaw.get_config.unsupported_type)
3. endpoint = fmt.Sprintf("%s.cls.tencentcs.com", CVMRegion)
4. topic_id = CheckCLSClawServiceOpened(ctx).TopicId
   —— 返回 nil / TopicId 为空 → 4xx (openclaw.cls.not_opened)
5. secret_id/secret_key = 按 (identifier, config_type) 查 local_agent_cls_credentials
   —— 查不到 / 读取失败 → 5xx (openclaw.get_config.credential_not_ready)
6. user_id = user.ID; user_name = user.Username
7. 组装返回 { cls: {...} }（install_cmd/update_cmd 待定值加入）
```

---

## 五、待确认问题（Challenge）

| # | 问题 | 我的默认建议 | 需你拍板 |
|---|------|------------|---------|
| Q1 | `CheckCLSClawServiceOpened` 未开通返回 `(nil, nil)`，如何区分「未开通」与「调用失败」？ | 返回 nil 即认为未开通 → 4xx `openclaw.cls.not_opened`；err != nil 才 5xx | 建议 OK？ |
| Q2 | `topic_id` 取 `CLSClawServiceResult.TopicId`（日志主题）还是 `MetricTopicId`？iWiki 原文用 `TopicId` | 用 `TopicId`（日志主题），与 iWiki 一致 | 确认 |
| Q3 | secret 表 `local_agent_cls_credentials` 为空时返 **5xx**（iWiki 原文），但业务上更像「配置未就绪」的 4xx。是否按 iWiki 原文 5xx？ | 按 iWiki 原文 5xx `openclaw.get_config.credential_not_ready` | 确认 OK？ |
| Q4 | `install_cmd` / `update_cmd` 固定值从哪来？硬编码在 handler / 常量 / site_config？运维按租户不同还是全局统一？ | 一期**全局固定常量**（与 endpoint 同级，非租户维度）；值待你给 | 值 + 来源？ |
| Q5 | 是否需要频率限制 / 仅本人 token？iWiki 草稿提过，定稿未要求 | 一期不做限流（复用 user token 反查 user，天然仅本人可拉自己的 user_id） | 确认不做？ |
| Q6 | `endpoint` 用 `controller.CVMRegion` 全局变量（与 report 接口同 region 源）是否 OK | OK | 确认 |

---

## 六、范围边界（一期）

- **做**：GET 只读查询 + 两层白名单 + 新表 model/migration + i18n + 单测 + 集成测试 + API.md 文档
- **不做**：凭据的写入/轮换 API（运维按租户 SQL 写入）、agent_type 维度区分（定稿已去掉）、service_name 字段（定稿已去掉）、密钥加密（定稿明文）

---

## 七、结论

规格清晰，主要风险在 Q1（nil vs error 区分）、Q4（install_cmd/update_cmd 值）。**Q4 的具體值用户晚点提供**，在值到达前不进 Implement 阶段。其余问题按默认建议推进至 Plan。
