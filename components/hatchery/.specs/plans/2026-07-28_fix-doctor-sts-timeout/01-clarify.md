# 01. Clarify — 需求澄清

> AI 以产品经理角色进行 Discovery + Challenge，确保需求清晰、边界明确。

---

## 背景

线上 `doctor_sessions.id=74` 对应的龙虾医生实例 `ins-mhpri21y` 在用户未开始对话的情况下持续存活 34 小时以上，未触发既有的 12 小时自动结束策略。

当前清理链路每 5 分钟执行一次：

1. `task/doctor_cleanup.go` 先调用 `RefreshDoctorSTS`；
2. STS 凭证约每 2 小时刷新一次；
3. `RefreshDoctorSTS` 使用 GORM `Update("sts_expired_at", ...)`，同时自动写入 `doctor_sessions.updated_at = now()`；
4. 无 `.jsonl` 对话文件时，清理逻辑使用 `DoctorSession.UpdatedAt` 作为“激活时间”计算 12 小时超时；
5. STS 刷新持续重置该时间窗口，导致 NoFiles 会话永远无法超时。

该问题只影响“会话已进入 active，但用户始终未开始对话”的分支。有对话文件时仍以远端 session 文件 mtime 判断空闲时长。

仓库中已有一个 2026-07-20 完成的任务 `doctor-active-session-force-timeout`，处理的是 CVM/TAT/mtime 探测持续失败导致 active 会话无法进入 ending 的问题。本次问题的直接根因是 STS 刷新污染生命周期时间，两者相关但不等同；当前 `origin/master` 也未包含旧任务记录的绝对 CreatedAt 前置兜底。

## 目标

- [ ] STS 周期刷新不得改变诊断会话的激活时间或无对话超时窗口。
- [ ] active 且始终无对话文件的会话，在激活满 12 小时后的下一轮 cleanup 扫描中进入 `ending`。
- [ ] `doctor-ending` 继续按现有流程销毁龙虾医生 CVM、删除实例记录并将会话推进到 `ended`。
- [ ] 已被 STS 刷新污染的存量 active 会话在升级后能够自动恢复清理，不要求人工逐条处理。
- [ ] 有对话文件的会话继续按文件 mtime 计算 12 小时空闲超时，现有语义不变。
- [ ] 通过回归测试覆盖“STS 刷新与 NoFiles 超时发生在同一轮 cleanup”这一真实调用组合。

调度精度说明：cleanup 每 5 分钟扫描一次，ending 每 1 分钟处理一次，因此“12 小时自动结束”指阈值达到后在一个 cleanup 周期内进入 ending，并在后续 ending 周期执行资源清理。

## 范围

| 包含 | 不包含 |
|------|--------|
| 修复 STS 刷新对会话生命周期时间的污染 | 修改 12 小时超时阈值 |
| 为 NoFiles 分支建立稳定、不可被普通字段更新污染的超时基准 | 改造有对话文件时基于 mtime 的空闲判断 |
| 兼容并自动收敛缺少新时间字段的存量会话 | 重构整个 Doctor cleanup/ending 调度框架 |
| 增加 controller/task 层回归测试 | 新增或修改对外 API |
| 如引入新字段，同步 GORM、`sql/init.sql` 和增量 migration | 修改 STS 有效期或刷新频率 |

## 待确认问题

| # | 问题 | 状态 | 结论 |
|---|------|------|------|
| 1 | NoFiles 的 12 小时应从“会话进入 active”还是“创建 session”开始计算？ | 待确认 | 建议从 active 开始，保持当前日志和业务语义；为此增加显式 `activated_at` |
| 2 | 是否接受数据库增加 `doctor_sessions.activated_at` 字段？ | 待确认 | 建议接受；仅把 STS 改为 `UpdateColumn` 虽能止血，但未来其他字段更新仍可能再次污染 `updated_at` |
| 3 | 存量 active 会话没有准确激活时间时如何兼容？ | 待确认 | 建议回退使用 `created_at`，避免继续使用已被污染的 `updated_at`，并使 34 小时等存量异常会话在下一轮扫描自动收敛 |

## 约束与依赖

- HTTP/API 契约保持不变，不新增用户可见字段。
- 所有数据库访问继续使用带租户上下文的 `model.DB(ctx)`。
- 若新增 GORM 字段，必须同步维护 `model/doctor_session.go`、`sql/init.sql` 和目标 Release 对应的增量 migration。
- SQLite 依赖 AutoMigrate；MySQL 不执行 AutoMigrate，必须先应用 migration。
- STS 刷新仍需继续执行，只调整其数据库写入对时间戳的副作用。
- 回归测试不能将 `RefreshDoctorSTSFn` mock 为空操作，否则无法覆盖本次真实根因。

## Challenge 结论

仅将 `Update` 改成 `UpdateColumn` 是必要的线上止血，但不是完整的生命周期建模：`updated_at` 表示“记录最后一次业务更新”，不能可靠承担“会话激活时间”的职责。

本任务建议采用：

1. STS 过期时间更新使用不刷新 `updated_at` 的写法；
2. 新增显式 `activated_at`，在状态首次切换到 active 时写入；
3. NoFiles 超时只读取 `activated_at`，存量 NULL 记录回退到 `created_at`；
4. 不改变有对话文件时的 mtime 判断。
