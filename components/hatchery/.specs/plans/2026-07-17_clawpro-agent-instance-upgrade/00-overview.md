# [2026-07-17] 【clawpro】存量 Agent 实例规格升配与系统盘容量扩容 - M1

> **本文件是本任务的单一真相源（Single Source of Truth）**：任务元信息、进度、当前步骤、关键决策全部在这里。
> 会话恢复时，先读本文件定位当前步骤，再按需加载对应阶段文件。
>
> ⚠️ Meta 中的 `分支` 字段是上下文恢复时定位任务的唯一依据，**必须**与 `git branch --show-current` 的输出完全一致。

---

## Meta

| 项 | 值 |
|----|----|
| 分支 | `feature/clawpro-agent-instance-upgrade` |
| 摘要 | 支持管理员对存量云端 Agent 单个或批量执行 AI2 规格升配与系统盘扩容 |
| 状态 | 已完成 |
| 创建日期 | 2026-07-17 |
| 负责人 | yutaoguo |
| 预期完成 | 2026-07-31 |

---

## Progress

<!--
Progress 更新规则（AI 必读）：
1. 步骤完成时，**原地替换** `- [ ] N. <step>` 为 `- [x] N. <step> (<结果摘要>)`
2. 禁止插入新行或重复编号，每个编号 01-08 有且仅有一行
3. 结果摘要示例：`(15/15 passed)`、`(覆盖率 92%)`、`(全量通过)`
4. 同步更新下方「当前步骤」章节
-->

- [x] 01. Clarify (产品、代码与 CVM/CBS API 均已核验，无剩余疑点) → [01-clarify.md](./01-clarify.md)
- [x] 02. Plan (CVM/CBS 常见阻断、三次完整复验、action 级限流及 41 UT/16 IT 已设计并独立复核 PASS) → [02-plan.md](./02-plan.md)
- [x] 03. Implement (双接口、CVM/CBS 三次复验、持久 worker、操作互斥与资源展示已落地；定向测试及 go vet 通过) → [03-implement.md](./03-implement.md)
- [x] 04. UT (U01-U41 41/41 PASS；新增核心代码覆盖率 82.7%；focused race 与涉及包全量非 race 通过) → [04-ut.md](./04-ut.md)
- [x] 05. Docs (新接口、list/status、34 个稳定码及严格 OpenAPI schema 已同步；生成与独立复核 PASS) → [05-docs.md](./05-docs.md)
- [x] 06. IT (I01-I16 完成；真实云规格/离线及停止态盘扩容通过；增量 API 2/2 路由、2/2 参数；资源已清理) → [06-it.md](./06-it.md)
- [x] 07. Review (2 高、5 中问题全部修复；独立复核无剩余 High/Medium；controller 全包、focused race、vet、OpenAPI 与确定性真实云 E2E 通过) → [07-review.md](./07-review.md)
- [x] 08. Commit (提交信息与全量检查已就绪；按强制顺序执行 add/commit/push) → [08-commit.md](./08-commit.md)

---

## 当前步骤

> 恢复会话时，优先读取此处指向的阶段文件。

- **步骤**：✅ 08. Commit（完成）
- **文件**：[08-commit.md](./08-commit.md)
- **上次更新**：2026-07-20 10:41:10

---

## 时间记录

| 步骤 | 开始时间 | 结束时间 | 耗时 | 备注 |
|------|---------|---------|------|------|
| 01. Clarify | 2026-07-17 11:30:12 | 2026-07-17 14:59:18 | 3小时29分6秒 | TAPD、项目代码、CVM/CBS API 核验完成；无剩余 blocker |
| 02. Plan | 2026-07-19 15:35:49 | 2026-07-19 16:52:03 | 1小时16分14秒 | 官方 CVM/CBS 规则、DescribeDisks 确定性门禁、首写 JIT、SDK 版本及 action 级限流已核验；独立复核 PASS |
| 03. Implement | 2026-07-19 16:54:29 | 2026-07-19 17:58:29 | 1小时4分0秒 | 双接口、完整实时复验、持久 worker、崩溃恢复及全入口互斥已实现；定向测试与 `go vet ./...` 通过 |
| 04. UT | 2026-07-19 18:18:22 | 2026-07-19 19:09:50 | 51分28秒 | U01-U41 41/41 PASS；新增核心代码覆盖率 82.7%；focused race、涉及包全量非 race 与 `go vet ./...` 通过；全量 race 存量竞态已记录 |
| 05. Docs | 2026-07-19 20:28:43 | 2026-07-19 20:48:00 | 19分17秒 | API 新接口、list/status、34 个稳定码与严格 OpenAPI schema 已同步；生成及独立 contract review PASS |
| 06. IT | 2026-07-19 20:51:55 | 2026-07-20 00:24:31 | 3小时32分36秒 | I01-I16 完成；真实云规格/离线与停止态盘扩容、失败持久化和锁冲突通过；增量 API 2/2 路由、2/2 参数；专用资源清理完成 |
| 07. Review | 2026-07-20 09:43:11 | 2026-07-20 10:32:35 | 49分24秒 | 2 个高、5 个中问题全部修复；独立复核无剩余 High/Medium；controller 全包、focused race、vet、OpenAPI 及确定性真实云 E2E 通过 |
| 08. Commit | 2026-07-20 10:37:57 | 2026-07-20 10:41:10 | 3分13秒 | Conventional Commit 信息、全阶段产物、测试与 Review 结论已就绪；`AGENTS.md` 按用户要求保持本地未跟踪且不加入 gitignore |

---

## 关键决策备忘

> **跨阶段共享的关键上下文**。仅记录影响后续步骤的决策，避免恢复时还要翻阅历史阶段文件。

- 仅支持存量云端 Agent 的 AI2 族内升配与系统盘扩容，不支持降配、跨族变配或磁盘类型变更。
- 调整采用“校验 → 提交复验 → 按实例异步执行 → 列表展示结果”的主流程。
- Q1-Q9、现有调用链、腾讯云官方规则和实际 Go SDK 均已核验；无 Plan blocker，真实账号/可用区差异统一在 IT 中保守验证。
- Plan 固定采用 instances 单行持久状态机、不新增任务历史表；`/admin/instances/status` 作为现有单实例详情/status 接口扩展资源字段。
- RequestId 落库前崩溃采用 3 次、间隔至少 5 秒的无痕迹观察阈值；对外失败文案按稳定错误码实时 i18n 渲染。
- submit 先判同目标幂等/异目标冲突，再对无活动操作实例实时复验；运行规格升配自动停机，系统盘严格按 `resize_mode` 决定在线或停机执行。
- validate、submit 与 worker 首次云写前 JIT 均执行完整实时检查；规格升配必选 InquiryPrice，系统盘扩容按实时磁盘事实、配额与模式规则判定，现有 CommonRequest denied-action 仅作 best-effort 补充。
- 系统盘事实以 CBS DescribeDisks 为准，规格/容量/盘状态/配额/售卖状态失败均逐台短路；系统盘不调用 InquiryPriceResizeInstanceDisks，校验结果与 write 由同一规范化 operation 构造。
- CVM SDK 固定升级至 `v1.3.130`，CBS SDK 固定 `v1.3.115`；分布式云调用 gate 按 `UIN+region+action` 隔离且无 burst。
- 腾讯云没有 DryRun 或资源预留；检查通过只承诺已排除可观测常见阻断，不承诺 100% 执行成功，JIT 后的库存/余额/状态 TOCTOU 作为显式残余风险。
- CVM 规格询价请求不支持 `ForceStop` 字段，询价/写请求复用 SDK 共同支持的规格目标，强停参数仅用于实际 Reset；系统盘直接按规范化 operation 写入并以真实执行结果为准。
- CVM 升级要求 common 模块同步至 `v1.3.130`；批量 command dispatch 因响应契约无法逐项报冲突，遇活动调整实例时保守拒绝整批。
- UT 修复两处契约缺口：CVM/CBS 系统盘容量漂移在 quota 前保守拒绝；RequestId 崩溃窗口读取失败不再消费三次成功观察计数。
- 新增核心文件 focused statement coverage 为 82.7%；全量 race 暴露既有全局 DB/hook 测试竞态，adjustment focused race 与涉及包非 race 全量均通过，官方增量脚本需在 commit 后重跑。
- `docs/API.md` 继续作为唯一 OpenAPI 源；严格请求约束由通用 schema overlay 生成，ID 数组“规范化后 1～100 项”用 `x-normalization` 表达，避免错误套用原始数组 min/max。
- IT 真实账号的 `Ai2.MEDIUM4 + CLOUD_BSSD` 在线系统盘扩容 `50→51 GiB` 成功且实例全程 RUNNING；离线 `51→52 GiB` 与停止态 `52→53 GiB` 云写也成功。
- IT 修复四处真实环境缺口：admin stop/restart-gateway 活动调整冲突由 500 统一为 409；`UnsupportedOperation` 不再被宽泛误判为 `cvm_operation_in_progress`；Python 3.9 E2E annotations 已兼容；I13 排除上一场景残留的同类型 LatestOperation RequestId。
- 最终验证镜像 digest 为 `sha256:2eaf8ae1a3f0b1b2f7863ae370ebce05b4c3464ce571cae56bb36a1636259d8e`；持久环境 Pod Ready、重启数 0、`/admin/config` HTTP 200，专用 CVM/安全组/K8s 资源均已清理。
- Review 修复 RequestId 崩溃窗口 FAILED 重放、restore 阶段无界超时和 `adjustment_status=processing/current_operation=''` 覆盖三项状态机/锁缺口；对应 race contract 全部通过。
- role apply 与 MCP add/update/delete/toggle 在事务内重新读取 adjustment lock；OpenAPI 新增机器可读 400/405/500 JSON error responses；I14 移除腾讯云 Reboot 抢跑并改为确定性审计/成功终态验证。
- Review 初审 2 高、5 中问题均已修复，独立复核无剩余 High/Medium；真实云确定性 E2E 使用已发布 itfix3 验证脚本及正常云链路，Review 新增源码修复尚未另行发布镜像。
- Commit 明确排除本地 `CODEBUDDY.md` 识别软链接 `AGENTS.md`；该文件保持未跟踪，不提交远端，也不加入 `.gitignore`。

---

## 风险速览

| # | 风险 | 严重度 | 缓解 |
|---|------|-------|------|
| R1 | 真实账号/可用区的规格售卖与询价、盘配额、系统盘实际执行能力及 RequestId 回传时序存在差异 | 中 | 规格未知保守拒绝；系统盘按可观测规则校验、写失败稳定映射；在 IT 用真实账号验证 |
| R2 | 云写 API 已受理但进程在 RequestId 落库前退出 | 高 | 持久化实例级目标与原状态，worker 先按目标值/最新操作收敛，不盲目重放 |
| R3 | 批量调用触发 CVM/CBS 频率限制或重复操作 | 高 | 单批 100、每租户并发 5、按 `UIN+region+action` 无 burst gate、逐台 JIT 复验和实例级幂等锁 |
| R4 | 调整期间其他实例变更破坏状态 | 高 | 所有共享 guard 和绕过 guard 的批量入口统一拒绝变更，delete 无覆盖例外 |

---

## 文件索引

| 文件 | 产物 |
|------|------|
| [00-overview.md](./00-overview.md) | 任务总览（本文件） |
| [01-clarify.md](./01-clarify.md) | 需求澄清：背景、目标、范围、待确认问题 |
| [02-plan.md](./02-plan.md) | 方案设计：改动文件、调用链、测试用例、风险 |
| [03-implement.md](./03-implement.md) | 实现：关键细节、与 Plan 差异 |
| [04-ut.md](./04-ut.md) | 单元测试：用例、覆盖率、未覆盖行 |
| [05-docs.md](./05-docs.md) | 文档更新清单 |
| [06-it.md](./06-it.md) | 集成测试：构建部署、端到端验证、增量覆盖率 |
| [07-review.md](./07-review.md) | Code Review：问题与修复 |
| [08-commit.md](./08-commit.md) | Commit message 与提交前检查 |

## 2026-07-22 系统盘校验修订

- `InquiryPriceResizeInstanceDisks` 是价格接口，不再作为系统盘扩容或在线能力门禁。
- 系统盘 validate、submit 和 worker JIT 统一根据实时实例状态、`DescribeDisks` 盘事实、`DescribeDiskConfigQuota` 容量区间/步长、DeniedAction 和 `resize_mode` 判定。
- 运行中 `online` 构造 `ResizeOnline=true/ForceStop=false`；运行中 `offline` 构造 `false/true`；已停止实例构造 `false/false`。
- 实际 `ResizeInstanceDisks` 仍是最终权威；若云端拒绝在线扩容，任务以 `online_resize_not_supported` 失败，不静默降级。
- 规格升配仍保留 `InquiryPriceResetInstancesType`；本修订只移除系统盘价格询价。
