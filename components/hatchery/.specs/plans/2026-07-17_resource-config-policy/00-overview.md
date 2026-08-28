# [2026-07-17] Resource Config Policy

> **本文件是本任务的单一真相源（Single Source of Truth）**：任务元信息、进度、当前步骤、关键决策全部在这里。
> 会话恢复时，先读本文件定位当前步骤，再按需加载对应阶段文件。
>
> ⚠️ Meta 中的 `分支` 字段是上下文恢复时定位任务的唯一依据，必须与 `git branch --show-current` 的输出完全一致。

---

## Meta

| 项 | 值 |
|----|----|
| 分支 | `feat/resource-config-policy` |
| 摘要 | 为站点、用户组和实例创建提供可配置、可继承、可校验的云资源策略 |
| 状态 | 已完成（2026-07-22 资源策略实体化重设计） |
| 创建日期 | 2026-07-17 |
| 负责人 | AI / 用户共同确认 |
| 预期完成 | 2026-07-19 |

---

## Progress

- [x] 01. Clarify    → [01-clarify.md](./01-clarify.md) (仅调整镜像与系统盘容量冲突行为，其余保持首版)
- [x] 02. Plan       → [02-plan.md](./02-plan.md) (追加独立策略、可索引 GroupConfigBinding 应用范围、默认策略懒创建和 4 API 重设计)
- [x] 03. Implement  → [03-implement.md](./03-implement.md) (独立 ResourcePolicy、可索引通用分组绑定、lazy default、4 API、resolver 和干净切换完成)
- [x] 04. UT         → [04-ut.md](./04-ut.md) (重设计 R-U01–R-U12 全通过；默认策略保护缺口已修复；全量、竞态、覆盖率和静态检查通过)
- [x] 05. Docs       → [05-docs.md](./05-docs.md) (六个资源策略 operation、默认策略 409 保护、OpenAPI、README 和 IT 脚本契约已同步)
- [x] 06. IT         → [06-it.md](./06-it.md) (I01–I12、3/3 脚本、116/116 HTTP、3 台真实 CVM、任务 API 6/6 + 参数 2/2 及清理全部通过)
- [x] 07. Review     → [07-review.md](./07-review.md) (高 2/中 1/低 2 全修；补齐默认名称读取时 i18n，普通策略名称保持原值)
- [x] 08. Commit     → [08-commit.md](./08-commit.md)（按用户要求覆盖此前功能提交）

---

## 当前步骤

- **步骤**：✅ 08. Commit（执行覆盖提交）
- **文件**：[08-commit.md](./08-commit.md)
- **上次更新**：2026-07-22 09:37

---

## 时间记录

| 步骤 | 开始时间 | 结束时间 | 耗时 | 备注 |
|------|---------|---------|------|------|
| 01. Clarify | 2026-07-17 10:00:06 | 2026-07-17 10:15:36 | 00:15:30 | 仅新增镜像容量快速失败及 ImageSize=0 跳过规则 |
| 02. Plan | 2026-07-17 10:19:48 | 2026-07-17 10:32:11 | 00:12:23 | 完整覆盖全功能、合规差距、64 UT、12 IT、Docs/风险/DoD |
| 03. Implement | 2026-07-17 10:56:25 | 2026-07-17 11:48:23 | 00:51:58 | 首版差距定向修复；按用户调整保持未知字段兼容，focused smoke/vet 通过 |
| 04. UT | 2026-07-17 12:00:41 | 2026-07-17 12:28:43 | 00:28:02 | U01–U64/P0/P1 全通过；全量 70.5%、增量 80.2%；全项目 `-race` 有非本功能异常 |
| 05. Docs | 2026-07-17 14:26:19 | 2026-07-17 14:36:21 | 00:10:02 | API/模块/索引/IT 契约完成；current/base OpenAPI 生成及参数断言通过 |
| 06. IT | 2026-07-19 15:34:50 | 2026-07-19 16:40:36 | 01:05:46 | I01–I12、3/3 脚本、真实 CVM 三容量分支、增量 API 5/5 + 6/6、隔离资源清理均通过 |
| 07. Review | 2026-07-19 16:43:00 | 2026-07-19 17:13:10 | 00:30:10 | 6 路专项 + 主审；高 2/中 10/低 4 全修；root 全量、focused race、vet、nested integration 均通过 |
| 08. Commit | 2026-07-19 17:16:18 | 2026-07-19 17:27:20 | 00:11:02 | 完成主提交/push；远端 MySQL schema gate 发现 COMMENT 差异并追加 `fix(sql)` 修正，拒绝 force-push |

### 2026-07-21 重设计时间记录

| 步骤 | 开始时间 | 结束时间 | 耗时 | 备注 |
|------|---------|---------|------|------|
| Clarify/Plan 修订 | 2026-07-21 22:44:53 | 2026-07-21 22:49:22 | 00:04:29 | TAPD 原型复核；后续按用户确认修正为独立策略 + 可索引通用资源绑定 + lazy default |
| Implement | 2026-07-21 22:49:22 | 2026-07-22 00:05:51 | 01:16:29 | 独立策略 + GroupConfigBinding 可索引范围；旧双真相源清除；全量/竞态/文档/脚本验证完成 |
| UT | 2026-07-22 07:39:08 | 2026-07-22 08:09:02 | 00:29:54 | R-U01–R-U12 全通过；修复默认策略保护；核心新增文件覆盖率 81.2%，全量 70.6%、分支增量 77.1%、定向 race 和 vet 通过 |
| Docs | 2026-07-22 08:11:34 | 2026-07-22 08:14:33 | 00:02:59 | 修正默认策略保护和 options 数量文档；OpenAPI 383 paths/393 ops；README、IT 脚本及旧路径检查通过 |
| IT | 2026-07-22 08:15:00 | 2026-07-22 08:26:49 | 00:11:49 | 隔离 EKS 3/3 脚本、116/116 HTTP、3 台真实 CVM、任务 API 6/6 + 参数 2/2、Pod restart 0 和清理全部通过 |
| Review | 2026-07-22 08:28:00 | 2026-07-22 09:31:25 | 01:03:25 | 高 2/中 1/低 2 全修；补默认名称 read-time i18n；资源定向 race、nested tests/vet、affected vet、OpenAPI 和 diagnostics 通过 |
| Commit | 2026-07-22 09:37:34 | 2026-07-22 09:37:34 | 00:00:00 | 用户确认覆盖此前功能提交；以 `--force-with-lease` 压缩为单一最终提交 |
| Commit 兼容调整 | 2026-07-22 09:43:00 | 2026-07-22 09:52:03 | 00:09:03 | group config-overview 恢复旧版 `meta.value` ResourceConfig 响应契约；handler race、vet、diagnostics 与脚本语法检查通过 |

---

## 关键决策备忘

- 本任务目录是在首版代码已经实现、rebase 并 push 后按新 SOP 追溯建立；已有代码仅代表“当前状态”，不自动视为变更后需求的最终契约。
- Clarify 从用户提出需求变化的当前时间开始真实记录；不补造此前步骤的开始/结束时间或确认记录。
- 当前代码基线为 `Release/2026_07_28`，资源配置与实例调整提交在联合 feature 分支完成。
- 只有用户明确确认的 Clarify 结论才能进入 Plan；未确认项不得在实现阶段自行推断。
- 已确认：最终所选镜像的 `ImageSize` 大于最终系统盘容量时必须直接创建失败；禁止后端静默扩大磁盘，相等时允许创建。
- 已确认：`AIImage.ImageSize == 0` 时跳过镜像容量比较，不改变磁盘容量，继续调用 CVM API 并由其判断兼容性。
- 已确认：除上述镜像容量行为外，用户组解析、用户覆盖、allowlist、fail-closed、CVMTemplate fallback、API 契约和后端交付范围均保持首版行为。
- 2026-07-21 重设计：资源策略改为独立 `ResourcePolicy`，策略内容只存 `ConfigJSON`；不修改 `user_groups`，也不新增专用关联表。
- 2026-07-21 重设计修正：应用范围复用 `GroupConfigBinding` 的资源绑定语义：`config_type=resource_policy`、`config_key=<policy ID>`、`value_json={}`，支持按策略和按组双向索引查询。
- 2026-07-21 重设计：一条策略可绑定多个组，一个组最多直接绑定一条；生效顺序为本组、最近祖先、企业默认。
- 2026-07-21 重设计：默认策略固定名“企业默认资源策略”，只允许编辑配置，首次需要时并发安全懒创建，不启动扫描全租户。
- 2026-07-21 重设计：删除未发布的 SiteConfig/分组内嵌完整资源配置双真相源和静态 basic options；四个独立管理 API 为最终契约。
- 2026-07-22 Implement：策略配置只存 `ResourcePolicy.ConfigJSON`；`GroupConfigBinding.ValueJSON={}` 仅为通用关系表空占位，`config_key=<policy ID>` 支持策略→分组索引反查。
- 2026-07-22 Implement：四个管理 API、两个新 options 路径、用户组树直接策略元数据、创建和总览统一 resolver 已落地；旧站点/分组内嵌配置路径无残留。
- 2026-07-22 Implement：资源策略最终接口契约维护在 `docs/API.md`；`docs/basic/platform-policy.md` 与 `docs/INDEX.md` 不属于本功能范围，保持基线不变。
- 2026-07-22 Implement 验证：`go test ./...` 曾全量 9 包通过；后续复跑唯一非本功能 LLM quota 测试偶发失败并单独复验通过；资源策略定向 `-race`、受影响包 `go vet`、OpenAPI 和三脚本 py_compile 通过。
- 2026-07-22 UT：R-U01–R-U12 全通过；默认策略更新现在只允许配置变更，改名或绑定分组显式返回冲突，拒绝请求不覆盖已编辑配置。
- 2026-07-22 UT 验证：全项目非竞态测试 9 包通过；资源策略模型/controller 定向 `-race` 通过；核心新增模型与 handler 覆盖率 81.2% 达到 80% 目标；全量覆盖率 70.6%、分支增量 77.1% 达到 CI 60% 门禁。
- 2026-07-22 Docs：默认策略改名或提交非空 `group_ids` 的 409 契约已同步到 `docs/API.md`、OpenAPI、`test/scripts/README.md` 和集成脚本；无关基础文档保持基线。
- 2026-07-22 Docs 验证：OpenAPI current 383 paths/393 operations、base 375/385；六个资源策略 operation、新 tree 参数及 create/update/delete schema 断言通过，三个集成脚本 py_compile 通过。
- 2026-07-22 IT：镜像 `it-resource-policy-redesign-202607220815` 在隔离 EKS run `6fe86982` 中通过 I01–I12；默认策略 409 保护、独立策略继承、真实 options 和三条 CVM 创建分支均验证成功。
- 2026-07-22 IT 清理：3 台真实 CVM 均销毁，测试 SG、StatefulSet、Service、策略/分组/用户和 fixture 清理或恢复；Pod restart 0，远端 TCR logout 和临时目录删除完成。
- 2026-07-22 IT API 覆盖：资源策略任务范围 6/6 operations、2/2 added params 全覆盖；全分支另有两个非本任务 operation 和六个参数未在 scoped run 覆盖，未计入本任务通过数字。
- 2026-07-22 Review：修复普通策略抢占默认保留名、GORM/SQL 复合索引漂移、integration token 明文日志、默认保护 i18n 和文档/注释问题；高 2/中 1/低 2 全部关闭。
- 2026-07-22 Review 验证：资源策略模型/controller 定向 `-race`、nested integration tests/vet、受影响包 vet、OpenAPI 和 gopls 通过；全仓 controller 超时与 task doctor cleanup goroutine panic 为非本功能测试基础设施问题，未虚报通过。
- 2026-07-22 Review i18n 调整：默认策略 canonical 名称稳定存储，列表/总览/树读取时仅对 `IsDefault` 本地化；英文为 `Enterprise Default Resource Policy`，普通策略名称原样返回，本地化名称可 round-trip update。
- Plan 决策：删除创建路径的磁盘静默扩容；新增无副作用镜像容量校验，并放在 VPC/SG/CVM 前。
- Plan 决策（已按用户调整）：资源配置入口校验单个 object、必填 wrapper 和预期字段，未知字段兼容忽略；options 补 tenant cache key、cache source、参数校验、i18n 和 SDK 调用包装。
- Plan 决策（已按用户调整）：本分支资源配置与实例调整增量 migration 合并为 `0728-resource-management.sql`，不新增重复 DDL。
- Implement 结论：生产主体首版已经存在，本步骤只修复镜像容量、JSON object/预期字段校验、错误透传、tenant cache、SDK/i18n 和 migration 前缀等确认差距。
- Implement 结论：创建路径不再静默扩盘；镜像容量失败早于 VPC/SG/CVM，`ImageSize==0` 保持原盘继续。
- Implement 结论：Plan U01–U64 测试代码已落地；controller/usergroup focused smoke、winner-failure 20 次压力回归和 `go vet ./...` 通过。
- Implement 说明：schema CI 只读取已提交 HEAD，将在 Commit 产生包含 `0715` migration 的新 HEAD 后执行。
- Implement 调整：用户明确不需要严格 JSON；未知字段保持兼容忽略，只要求单个 object、必填 wrapper 和预期字段解析/业务校验正确。
- Implement 调整验证：站点、组、resolver 和用户配置均忽略未知字段，已知字段继续规范化/校验；兼容专项测试、全量 focused smoke、`go vet ./...` 和 gopls diagnostics 通过。
- UT 修复：U35 补齐 handler 冲突测试；U63/U64 改为直接验证创建预检，并修复 options 缓存数组被错误按 wrapper 解码、导致命中后仍回源 CVM 的问题。
- UT 结论：Plan U01–U64 全部通过，P0 48/48、P1 16/16；覆盖率脚本内全项目非竞态测试通过，增量覆盖率 80.2%（528/658）通过。
- UT 限制：`go test ./... -v -race -count=1` 的 `controller` / `task` 仍受非本功能全局测试 DB 竞争及既有断言失败影响；功能定向竞态测试无 FAIL、SKIP 或 DATA RACE。
- Docs 结论：`docs/API.md` 成功生成 390 operations，较 `Release/2026_07_15` 精确新增 5 个 resource operations；query/body 参数断言全部通过。
- Docs 边界：普通 `/openclaw/create` 支持本次 `resource_config` form 覆盖；`/admin/instances/create` 当前不接受直接 `resource_config` JSON 字段，只复用站点/组策略及共享校验，文档按实际实现记录。
- Docs/IT 衔接：`test/scripts/README.md` 已写明 I01–I12、隔离 fixture 和清理契约；三个目标脚本尚未创建，并明确标记为下一步 IT 产物，不虚构当前可运行状态。
- IT 产物：三个目标 Python 脚本已落地；Python 3.9 annotations 兼容修复及 cloud proxy bootstrap-token 子进程注入已由 nested integration module 测试覆盖。
- IT 结论：最终隔离 EKS run `ad3b06c0` 中 3/3 脚本全通过、138 coverage frames、Pod restart 0；I01–I12 全部 PASS。
- IT 云验证：真实创建并销毁 3 台 CVM，DescribeInstances 反查用户最终机型/系统盘；ImageSize=100/DiskSize=50 在任何 cloud/SG 调用前失败且无占位/SG 池变化。
- IT 覆盖率：相对 `origin/Release/2026_07_15` 的 5 个新增 operation 全覆盖、6 个既有 operation 新参数全覆盖；资源 options 全部 query 参数及 `resource_config` 请求参数无缺失。
- IT 清理：最终 StatefulSet/Service/CVM/SG/用户/组/站点配置/镜像 fixture 全部删除或恢复；临时 TCR 凭证通过 stdin 使用并在 push 后 logout，未进入仓库与报告。
- Review 结论：确认高 2、中 10、低 4 共 16 项问题并全部修复；无高严重度未修问题，API/schema/安全基线通过。
- Review fail-closed：持久化站点/组配置应用后的有效请求异常返回 500；用户局部覆盖与继承 discriminator 合并后无效返回 400，均早于 SG/CVM。
- Review fallback：新租户不再物化 ResourceConfig 副本，空值继续实时回退当前 CVMTemplate；组 wrapper `value` 强制 object，未知字段仍兼容忽略。
- Review options：三个 handler 严格 GET-only；cache hit/inflight waiter 先于 client 构造；system-disks 只接受匹配且 SELL 的 quota。
- Review IT 加固：真实创建脚本 runner 独占；拿到 `cvm_id` 立即登记清理，DB id 反查失败时直接 TerminateInstances；ImageSize=0 云失败断言最终磁盘日志。
- Review 边界：不新增创建时 CBS 磁盘 availability 预检；明确保持文档既有“最终机型 SELL 预检”契约，修正误导注释。
- Review 验证：root `go test ./...` 8 packages PASS；定向 `-race`、root/nested `go vet`、nested integration 全量、gopls diagnostics、3 脚本 py_compile 均通过。
- Commit 决策（最终）：用户明确要求覆盖此前提交；保留 `3c27393e^`，将本任务已有 3 个提交与重设计 worktree 压缩为单一最终提交，并使用 `--force-with-lease` 安全更新远端。
- Commit 兼容调整：`GroupConfigBinding` 继续只存策略 ID；group config-overview 响应时解析 `ResourcePolicy.ConfigJSON`，以旧版同格式 ResourceConfig object 返回 `meta.value`，并保留 `meta.resource_config` 显式别名。

---

## 风险速览

| # | 风险 | 严重度 | 缓解 |
|---|------|-------|------|
| 1 | 新需求与已实现的数据结构、API 或继承语义不一致 | 高 | Clarify 明确最终契约，Plan 阶段列出迁移和兼容策略 |
| 2 | 已推送提交缺少 SOP 任务产物 | 中 | 本分支按 01–08 阶段补齐真实产物并追加符合规范的提交 |
| 3 | 腾讯云可售性和磁盘配额依赖真实云 API | 中 | UT 使用注入/Mock；IT 在已部署环境验证并记录结果 |
| 4 | 仓库全量 `-race` 门禁存在非本功能数据竞争/断言失败 | 中 | UT 已记录完整证据并以功能定向 `-race` 隔离验证；作为独立测试基础设施治理项处理 |

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
