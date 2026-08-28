# 02. Plan — 方案设计

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `controller/openclaw_create_service.go` | 修改 | 定义共享创建标签结构，并在 `createInstanceOptions` 中承载管理员端已解析标签 |
| `controller/openclaw.go` | 修改 | 解析用户端 `tags`；合并默认/自定义标签并注入 RunInstances |
| `controller/admin_instance_create.go` | 修改 | 管理员 JSON 请求体新增 `tags` 并传给共享创建流程 |
| `controller/custom_instance_tags.go` | 新增 | 集中实现 form JSON 解析、默认/自定义标签合并及 RunInstances 注入 |
| `controller/custom_instance_tags_test.go` | 新增 | 覆盖解析、合并、冲突和云端透传语义 |
| `controller/openclaw_handler_writectx_test.go` | 修改 | 用户端成功链路断言 RunInstances 收到自定义+默认标签 |
| `controller/admin_instance_create_test.go` | 修改 | 管控端成功链路和严格 JSON 断言 |
| `docs/API.md` | 修改 | 更新两个创建接口的 `tags` 参数、示例和合并/校验语义 |
| `test/scripts/openclaw_instance/test_instance_create_validation.py` | 修改 | 增加用户端非法 `tags` 的无计费校验 |
| `test/scripts/openclaw_instance/test_admin_instance_create.py` | 修改 | 完整请求加入 `tags`，增加 tags 未知字段校验 |

## 调用链 / 数据流

```text
POST /openclaw/create
  form.tags(JSON string) → parseCreateInstanceTags
                                   ┐
POST /admin/instances/create       │
  JSON tags[] → createInstanceOptions.CustomTags
                                   ┘
          → createInstance
          → model.ResolveTagsForGroup
          → mergeCreateInstanceTags（custom 同 key 覆盖 default）
          → RunInstancesRequest.TagSpecification[instance]
          → CVM RunInstances（存在性/长度/数量/字符规则的权威校验）
```

## 数据库变更

无。

## 测试用例设计（自然语言描述）

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 用户端解析多个标签 | `tags=[{"key":"env","value":"prod"},...]` | 保留顺序和原始键值 | P0 |
| 2 | 用户端非法 JSON | object、未知字段、尾随 JSON、语法错误 | 创建前返回 400，不创建占位实例 | P0 |
| 3 | 仅默认标签 | custom 为空，default 两项 | RunInstances 收到两项默认标签 | P0 |
| 4 | 仅自定义标签 | custom 两项，default 为空 | RunInstances 收到两项自定义标签 | P0 |
| 5 | 自定义+默认标签 | 两边 key 不冲突 | RunInstances 收到并集 | P0 |
| 6 | 同 key 冲突 | custom `env=dev`，default `env=prod` | 只保留 `env=dev` | P0 |
| 7 | 自定义内部重复 key | custom 两个 `env` | 不在本地静默去重，交由 RunInstances | P1 |
| 8 | 用户端完整成功链路 | form 携带 tags，DB 有默认标签 | 捕获的 RunInstances 请求包含合并结果 | P0 |
| 9 | 管控端完整成功链路 | JSON 携带 tags，DB 有默认标签 | 捕获的 RunInstances 请求包含合并结果 | P0 |
| 10 | 管控端严格 JSON | `tags[]` 含未知字段或错误类型 | 返回 400 | P0 |
| 11 | 向后兼容 | 两个接口均不传 tags | 现有默认标签行为与响应不变 | P0 |

### 集成测试（IT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 用户端非法 tags | 已登录用户 + 非数组 JSON | 400，且在 RunInstances 前结束 | P0 |
| 2 | 管控端完整参数解析 | 缺失目标用户 + 合法 tags | 400 用户不存在，而不是未知字段/JSON 错误 | P0 |
| 3 | 管控端 tags 严格 JSON | `tags[]` 含未知字段 | 400 | P0 |

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| 1 | 本地校验与腾讯云规则不一致 | 中 | 高 | 只解析结构；业务校验统一交给 RunInstances |
| 2 | 默认标签覆盖用户本次选择 | 中 | 高 | 自定义同 key 优先并加单测 |
| 3 | 管控端复用 form 时标签丢失 | 中 | 中 | 通过 `createInstanceOptions` 显式传递，不做 JSON→form→JSON 往返 |
| 4 | 标签请求内容意外进入日志 | 低 | 中 | 不新增标签值日志，只记录数量；管理员请求体仍沿用既有整体不记录策略 |
| 5 | 全量测试耗时较长或受基线波动影响 | 中 | 中 | 先跑聚焦测试，再跑 `go test ./...`、vet、staticcheck，分别记录结果 |
