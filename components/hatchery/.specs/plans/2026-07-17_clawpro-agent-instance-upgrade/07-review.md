# 07. Review — 代码审查

## 审查结论

- [x] AI 自动 Review
- [ ] 人工 Review
- [x] 无高严重度未修复问题
- [x] 无中严重度未修复问题
- [x] 代码风格一致
- [x] 安全基线检查通过
- [x] API、数据库、i18n、审计与测试契约一致

结论：**PASS**。初审共发现 2 个高严重度、5 个中严重度问题，均已修复并补充行为回归；独立复核确认无剩余 High/Medium finding。

- Review 时间：`2026-07-20 09:43:11` ～ `2026-07-20 10:32:35`，耗时 `49分24秒`。

## 审查方式

1. 主审按 `.specs/review.md` 检查正确性、安全性、规范性、可维护性和性能。
2. 四路独立 specialist review：
   - 持久状态机、崩溃窗口、并发与恢复；
   - HTTP/API、权限、审计、i18n、数据库迁移和所有共享写入口；
   - CVM/CBS SDK 请求字段、规格 Inquiry/write 同源、系统盘本地规则与规范化写请求、配额、错误映射和 action gate；
   - 单测、真实云 E2E、文档与 OpenAPI 生成契约。
3. 修复后再次独立复核 worker、submit CAS、role/MCP 事务、OpenAPI 和 E2E；最终复核无剩余 High/Medium 问题。
4. 使用 focused race、controller 全包、`go vet ./...`、Python unittest/compile、OpenAPI 生成和确定性真实云 E2E 提供行为证据。

## 发现与处理

| # | 文件 | 问题 | 严重度 | 修复 | 回归证据 | 状态 |
|---|---|---|---|---|---|---|
| R01 | `controller/instance_adjustment_worker.go` | RequestId 落库前崩溃后，若腾讯云相关 LatestOperation 已为 FAILED，原逻辑会把它当“无痕迹”并再次执行付费/停机云写 | 高 | ambiguous reconcile 识别相关 FAILED，映射稳定原因并直接进入 restore_failure，绝不重放 | `U26/failed_request_is_recorded_without_replay`；focused race PASS | 已修 |
| R02 | `controller/admin_instance_adjustment.go` | `adjustment_status=processing` 但 `current_operation=''` 时，submit preflight、validator 和最终 CAS 可覆盖在途任务 | 高 | 三层门禁同时把 processing adjustment 视为活动锁；最终 CAS 增加 `adjustment_status <> processing` | `U05FirstFailurePriority`、`U13ProcessingStatusWithoutCurrentOperationRejectsSubmit`；race PASS | 已修 |
| R03 | `controller/instance_adjustment_worker.go` | restore_success/restore_failure 被排除在 15 分钟总超时外，瞬时态或读错可永久持有实例锁 | 中 | restore 阶段超过固定总时限后读取一次实时云状态，写入 truthful cloud cache，并以 `adjustment_restore_failed` 终态清锁 | `U28/expired_restore_terminates_with_truthful_failure`；race PASS | 已修 |
| R04 | `controller/openclaw_role_apply.go` | role precheck 与事务写之间存在 TOCTOU，调整可在其间获得实例，随后 role 字段/技能任务仍被写入 | 中 | role 字段更新改为带 `current_operation='' AND adjustment_status<>processing` 的条件写；RowsAffected=0 返回冲突，调用层加入 skipped result 且不启动异步任务 | `TestWriteRoleFieldsTx_RejectsConcurrentAdjustment`；role focused race PASS | 已修 |
| R05 | `controller/openclaw_mcp.go` | MCP add/update/delete/toggle 只检查旧实例快照，调整在 precheck 后受理时仍可能写安装记录或执行 TAT | 中 | 所有 MCP mutation 通过 transaction-local `SELECT ... FOR UPDATE` 重新读取 adjustment lock；add/update/toggle 的 DB 写和 delete 的 TAT 准入均使用同一 guard，冲突返回 operation-in-progress | `TestSaveMCPInstallationUnlessAdjusting_*`、`TestUpdateMCPInstallationUnlessAdjusting_*` 及全部 MCP mutation focused race PASS；独立复核无残余 finding | 已修 |
| R07 | `test/scripts/openclaw_instance/test_admin_instance_adjust_config.py` | I14 通过真实 Reboot 与 Resize 抢跑制造失败，腾讯云任一合法调度顺序都可能令用例随机失败 | 中 | 删除 Reboot/I14A 和 allow-failure 分支；I06 独立验证离线目标、确认拒绝、受理、成功收敛及终态清错；I14 只验证审计和确定性成功终态，失败持久化由 fake-backed worker/surface contract 覆盖 | Python compile PASS；Review 真实云 E2E `All 1 test scripts passed` | 已修 |

## 正确性复核

- **持久状态机**：queued/submitting/polling/restore/terminal 的 DB 真相源完整；FAILED ambiguous cloud operation 不重放；RequestId 已持久化后不重放；读失败不消费无痕迹观察计数。
- **超时与恢复**：15 分钟边界覆盖 polling 与 restore；restore 超时以实时 `last_cvm_state` 落库，不伪造原稳定态。
- **并发锁**：submit preflight、实时 validator、最终 CAS、生命周期共享 guard、role 与 MCP mutation transaction 均识别 `current_operation` 和 `adjustment_status=processing`。
- **云契约**：独立 cloud reviewer 对 CVM/CBS 提取、规格询价/写目标同源、系统盘本地规则与规范化写请求、CBS quota、denied-action、action gate、RequestId、`UnsupportedOperation` 在线扩容映射和 SDK factory 未发现问题。
- **批量语义**：目标去重、顺序保留、逐项 accepted/rejected/already_processing 和部分成功契约保持不变。

## 安全与项目红线

- [x] 所有 handler 使用登录/管理员权限门禁；新 submit 路由使用 `WithAudit(WithOpenAPI(...))`。
- [x] 未使用 `db.Exec`、`db.Raw`、`db.Table`、`db.Row` 或 `db.Rows` 裸 SQL。
- [x] 所有请求/worker DB 访问使用 tenant context 的 `model.DB(ctx)`；新增事务保留 tenant callback。
- [x] CVM/CBS client 通过 `GetCVMClient(ctx)` / `GetCBSClient(ctx)` 工厂创建。
- [x] 用户可见调整错误通过 `i18n.T()` 渲染；全部新增 Key 有英文翻译。
- [x] GORM 字段、`sql/init.sql` 与 `sql/0728-resource-management.sql` 一致。
- [x] API 文档、生成 OpenAPI、新路由与新增 list 参数均已覆盖。
- [x] 未新增硬编码密钥；含 AK/SK/admin token 的临时 E2E 日志已删除。

## 验证结果

| 验证 | 结果 |
|---|---|
| Review 新增 7 组回归（worker replay/restore timeout、submit lock/CAS、role/MCP TOCTOU、OpenAPI、E2E determinism） | PASS；所有 Go focused regression 使用 `-race` |
| `go test ./controller -run TestAdjustment -count=1 -race` | PASS |
| role apply focused race | PASS |
| MCP add/update/delete/toggle focused race | PASS |
| `go test ./task -run InstanceAdjustment -count=1 -race` | PASS |
| `go test ./model -run Adjustment -count=1 -race` | PASS |
| `go test ./controller -count=1` | PASS |
| `go vet ./...` | PASS |
| `python3 -m unittest test.test_api_md_to_openapi` | PASS |
| `python3 -m py_compile`（E2E、OpenAPI generator/test） | PASS |
| `make openapi BASE_BRANCH=origin/master` | PASS；377 paths / 387 operations |
| 确定性 Review 真实云 E2E（`ADJUSTMENT_IT_SKIP_SPEC=1`） | PASS；I01/I02/I05/I06/I07/I10～I16，122 coverage frames，测试 Pod 无重启 |
| Review E2E 清理 | PASS；CVM `ins-1d4vdm02` 删除后 TotalCount=0，安全组及 `hatchery-b258e8d4` StatefulSet/Service 已删除 |

Review 真实云复验使用 IT 已发布的 `itfix3` 镜像验证确定性脚本和既有正常云链路；Review 新增的 CAS/崩溃恢复/事务竞态修复由可控 race contract 覆盖，尚未另行发布新镜像。生成的临时报告、base spec 和敏感日志已在提取证据后删除。

## 剩余风险

- 当前真实账号的 `Ai2.MEDIUM4 + CLOUD_BSSD` 仍不支持在线扩容；服务稳定返回 `online_resize_not_supported` 且不自动降级，离线和停止态扩容真实云写均通过。
- CLOUD_BASIC 条件分支未在专用实例出现，仍依赖 deterministic cloud contract tests 与实时 CBS quota 的保守门禁。
- 腾讯云检查与写入间的产品级 TOCTOU 无法由 DryRun/预留消除；JIT validator、稳定失败码、15 分钟边界和审计为既定缓解。

## 2026-07-22 Review 修订

原 Review 对“磁盘询价/写同源”的通过结论被产品决策替代。新实现不再把价格接口错误当作在线能力判断，避免所有运行中系统盘因询价接口返回 `UnsupportedOperation.InstanceStateStopped` 被统一拒绝。保留事实/配额校验和执行阶段 fail-closed。
