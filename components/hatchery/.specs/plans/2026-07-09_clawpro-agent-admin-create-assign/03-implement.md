# 03. Implement — 实现记录

---

## 关键实现细节

1. `createInstance(..., createInstanceOptions)` 是用户端和管理员端共用的 CVM 创建主流程，返回 `(createInstanceResult, bool)` 且不写成功响应。两个 handler 分别维护自己的输入协议和 HTTP 成功响应，避免管理员字段污染普通用户 API。
2. `HandleAdminCreateInstance` 使用 1 MiB 上限和严格 JSON 解码；目标用户、有效分组、配额、agent_type 能力、role、model、channel、skill 均在调用 CVM 前校验。
3. 指定 models 时跳过站点默认模型。Agent ready 后依次调用 `applyBuiltinModel`；第一个模型为 primary，后续模型为 fallback。单次下发失败回滚本次绑定并停止后续模型。
4. 创建期通道和用户 `set-channel` 共用 `validateManualChannelConfig` / `applyManualChannelConfig`。多个通道按请求顺序执行；凭据只存在于当前 goroutine，不持久化、不重试。
5. 创建期技能和用户 `add-skill` 共用 public/enterprise 来源语义。source 省略时默认 public；enterprise 复用可见性校验和 `applyEnterpriseSkill`。追加技能不创建 `SkillInstallation`。
6. 管理员创建接口不记录 request body；TAT 参数值和 set-channel 脚本失败输出不进入结构化日志。成功响应只包含 `ok` 和 CVM `instance_id`。
7. 通道共享实现合并到既有 `openclaw_channel.go`，创建期等待和调度保留在 `openclaw_create_service.go`；管理员创建、审计和 i18n 测试合并到 `admin_instance_create_test.go`。

## 与 Plan 差异

相对最终 Plan 无行为差异。实现过程中曾评估持久化通道预设表、恢复 worker 和状态查询接口，但该方案在多副本下存在重复下发风险，也超出“与用户手动操作一致”的需求；最终删除整套持久化设计，采用当前进程内一次性异步下发。

文件组织有两项收敛：

- `openclaw_channel_preset.go` 的共享通道逻辑并入 `openclaw_channel.go`，创建期编排并入 `openclaw_create_service.go`。
- 管理员路由审计和 i18n 测试并入 `admin_instance_create_test.go`，不保留单独的小测试文件。

## 检查项

- [x] `gofmt` 格式化通过（功能新增文件和新增区块；`controller/audit.go`、`i18n/keys.go` 的整文件 `gofmt -l` 命中来自基线既有对齐差异，未引入整文件格式化噪声）
- [x] `go vet ./...` 无错误
- [x] 写接口已添加审计日志
- [x] 数据库变更已同步 `sql/init.sql` + migration SQL（N/A：无数据库变更）
- [x] 使用 `model.DB(r.Context())` 而非 `model.DB`
- [x] 无硬编码密钥/配置
- [x] 用户可见文案使用 `i18n.T()`，新增 Key 已同步 `en.go` 英文翻译
