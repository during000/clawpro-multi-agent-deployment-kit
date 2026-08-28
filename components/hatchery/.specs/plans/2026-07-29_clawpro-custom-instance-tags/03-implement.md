# 03. Implement — 实现记录

## 关键实现细节

- 新增共享 `createInstanceTag`，两个 API 都使用 `{key,value}`，避免用户端 form 和管控端 JSON 产生两套标签模型。
- 用户端 `tags` 由 `parseCreateInstanceTags` 按严格 JSON 数组解析：拒绝对象、未知字段、尾随值和语法错误；空值保持原行为。
- 管控端请求体新增 `Tags []createInstanceTag`，经 `createInstanceOptions.CustomTags` 直接传入共享流程，避免 JSON → form → JSON 往返。
- `mergeCreateInstanceTags` 保留自定义标签顺序，自定义同 key 覆盖命中的默认标签；自定义内部重复 key 原样交给 RunInstances 验证。
- `applyCreateInstanceTags` 统一写入 `TagSpecification(ResourceType=instance)`；没有自定义或默认标签时不改变请求。
- 创建成功后 `instances.cvm_tags_json` 缓存与 `RunInstances` 使用同一份最终合并标签，避免创建期列表只显示默认标签。
- 腾讯云标签错误按请求是否包含自定义标签映射：至少一个自定义标签时返回 400；没有自定义标签时沿用原有 500 行为。
- 新增日志只记录自定义、默认和最终标签数量，不记录 key/value。

## 与 Plan 差异

- 共享标签结构与实现集中在新增的 `controller/custom_instance_tags.go`，而非放入 `openclaw_create_service.go`；后者只保留流程参数。
- 基于腾讯云 RunInstances 文档补充标签专属 SDK 错误分类，并结合请求是否传入自定义标签决定 400/500，Plan 原始清单未单列该辅助函数。
- 真实云 IT 发现原实现仅把默认标签写入创建期缓存；新增 `createInstanceTagItemsForCache` 复用合并规则，并补充两个入口的缓存断言。
- 兼容性审查发现仅按 SDK 错误码映射会把失效的管理员默认标签误报为调用方 400；改为仅在请求实际传入自定义标签时映射 400。

## 检查项

- [x] `gofmt` 格式化通过
- [ ] `go vet ./...` 无错误（被既有 `skillhubclient/client_test.go:278` 警告挡住；`go vet ./controller/...` 通过）
- [ ] `staticcheck ./...` 无错误（当前环境未安装 `staticcheck`）
- [x] 写接口已添加审计日志（复用 `main.go` 中两个入口的既有 `WithAudit`；管控端 action 为 `instance_admin_create`）
- [x] 数据库变更为 N/A，无 model/schema/migration 改动
- [x] 生产代码未新增裸全局 DB 访问
- [x] 无硬编码密钥/配置
- [x] 未新增用户可见文案或 i18n Key；复用 `MsgInvalidJSON` 和既有 RunInstances 错误包装
