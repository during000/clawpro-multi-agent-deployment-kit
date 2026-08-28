# 03. Implement — 实现记录

---

## 关键实现细节

### 接口、参数与持久状态

- 新增 `POST /admin/instances/adjust-config/validate` 与 `POST /admin/instances/adjust-config`。请求采用严格 JSON 解码，拒绝未知字段、尾随 JSON、重复/空实例 ID，并把规格、磁盘容量与停机模式规范化为单一 operation。
- validate 与 submit 共用完整实时校验链；submit 按实例返回接受、同目标幂等或稳定拒绝原因。活动调整通过 instances 单行 CAS 状态机持久化，不新增任务历史表。
- instances 新增资源缓存、调整目标、原始运行状态、RequestId、阶段、稳定错误码及独立资源更新时间字段；同步更新 `model/instance.go`、`sql/init.sql` 和 `sql/0728-resource-management.sql`。

### 云侧准入与调用约束

- 新增统一 CVM/CBS client 工厂和 adjustment cloud gateway。规格升级按 AI2 等级比较，系统盘事实以 CBS `DescribeDisks` 为准；实时检查覆盖实例/磁盘状态、SELL 可售、配额、denied action，规格分支额外执行 InquiryPrice。
- denied-action CommonRequest 保持 best-effort；不可用或失败时，规格分支仍强制执行 InquiryPrice，系统盘分支继续按实时事实、配额和模式规则判定。
- 规格 Inquiry 与 write 从同一规范化 operation 构造；系统盘不调用 InquiryPriceResizeInstanceDisks，write 直接使用规范化 operation。调用前按 `UIN+region+action` 获取 MySQL 分布式锁，并把锁持有到最小调用间隔结束，实现跨进程、无 burst 的 action 级 gate。
- 所有腾讯云错误先映射到稳定内部 reason code，对外响应和列表展示再通过 `i18n.T()` 实时翻译；新增 CBS 访问日志组件，未记录密钥或完整请求体。

### 异步执行、崩溃恢复与互斥

- 新增每 5 秒运行、按租户并发上限 5 的持久 worker。流程为 queued → submitting → polling → restoring；首次云写前再次执行完整 JIT 校验。
- 云写 RequestId 落库前崩溃时，worker 先检查目标资源与最新操作痕迹；连续 3 次、间隔至少 5 秒均无痕迹才允许重放，避免重复变配。
- 需要停机时记录原始状态和计费停机模式，执行完成后恢复；JIT、执行、轮询或恢复失败均写入稳定终态，成功后清理活动调整字段并保留结果状态。
- 共享操作 guard、delete、批量命令、技能/插件/MCP/角色分发及 detect-install 等绕过入口统一拒绝活动调整实例。调整期间 `/admin/instances` 和 status 接口展示目标、阶段、资源值与稳定错误信息，并屏蔽普通 lifecycle 副作用。
- 资源缓存使用独立 `resource_synced_at` 做时间戳保护，避免旧 reconcile 结果覆盖较新的规格或磁盘容量。

### 实现阶段验证

- `gofmt` 已覆盖全部本次改动 Go 文件；`go mod tidy` 通过。
- `go vet ./...` 通过。
- `go test ./controller ./model ./task ./i18n -run 'Test(OperationTimeouts|SetOperation|CanOperate|ClearOperation|OperationTransitStatus|BuildAdminInstanceFromCache|BuildAdminInstanceWithStatus|UpdateInstanceCachedStatus)'` 通过。
- 新增接口契约测试覆盖严格解析、规格/磁盘校验、云侧 common gates、规格 Inquiry fallback、余额/配额、部分接受、幂等、delete lock 与 actions。
- 新增 worker 契约测试覆盖 JIT、规格 Inquiry/write 目标一致性、系统盘无询价且 operation/write 参数一致、JIT 失败、成功清理及 RequestId 崩溃窗口 3 次观察；两组新增测试的合并定向命令通过。完整用例、race 与覆盖率在下一步 UT 执行并记录。

## 与 Plan 差异

1. CVM SDK 的 `InquiryPriceResetInstancesTypeRequest` 只有 `InstanceIds` 与 `InstanceType`，没有 `ForceStop` 字段；因此规格询价与写请求复用 SDK 共同支持的目标字段，`ForceStop` 仅在 `ResetInstancesType` 执行请求中设置。系统盘不调用价格询价接口，直接以规范化 operation 的 `DiskId`、`DiskSize`、`ForceStop`、`ResizeOnline` 构造真实写请求。
2. 升级 CVM 至 `v1.3.130` 时，Go module 解析要求腾讯云 common 模块同步从 `v1.3.73` 升至 `v1.3.130`；CBS 固定为 `v1.3.115`，未升级其他腾讯云 service 模块。
3. 现有批量 command dispatch 响应模型不支持提交前逐目标返回冲突项，因此只要批次中存在活动调整实例就保守拒绝整批；其他本身支持逐项结果的批量接口按 Plan 执行安全部分跳过。

## 检查项

- [x] `gofmt` 格式化通过
- [x] `go vet ./...` 无错误
- [x] 写接口已添加审计日志
- [x] 数据库变更已同步 `sql/init.sql` + migration SQL
- [x] 使用 `model.DB(r.Context())` 而非 `model.DB`
- [x] 无硬编码密钥/配置
- [x] 用户可见文案使用 `i18n.T()`，新增 Key 已同步 `en.go` 英文翻译

## 2026-07-22 Implement 修订

系统盘校验已移除价格询价。`instanceAdjustmentCloudGateway` 将询价方法收窄为 `InquiryInstanceType`；系统盘在事实、配额、DeniedAction 和模式检查通过后直接标记可提交。worker 执行 `ResizeInstanceDisks` 时保留在线拒绝的稳定错误映射。规格升配逻辑不变。
