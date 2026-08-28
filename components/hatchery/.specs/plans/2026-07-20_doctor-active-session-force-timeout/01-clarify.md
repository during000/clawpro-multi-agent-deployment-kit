# 01. Clarify — 需求澄清

> AI 以产品经理角色进行 Discovery + Challenge，确保需求清晰、边界明确。

---

## 背景

TAPD bug 单：https://tapd.woa.com/tapd_fe/20422209/bug/detail/1020422209160782882
【clawpro bug 单】龙虾医生无法使用一直报错已有进行中的诊断会话

用户 user_id=272 于 `2026-07-13 03:22:05` 创建的龙虾医生诊断会话（针对目标实例 `ins-1njivqck`），
诊断完毕之后一直没有销毁，直到 `2026-07-15 06:07:22` 才被销毁（卡住近 51 小时，远超系统承诺的
12 小时兜底 SLA）。期间 `doctor_sessions.status` 一直是 `active`，触发了 `HandleDoctorStart` 中
按 `user_id` 维度的互斥锁（同一用户同时只能有一个进行中的诊断会话），导致该用户后续无法再创建
新的诊断会话，报错 `active_session_exists`（"已有进行中的诊断会话"）。

**代码级根因**：`task/doctor_cleanup.go` 的 `cleanupDoctorSessions`（每 5 分钟执行）对 `active`
会话的 12 小时超时判断，完全依赖远端探测链路全部成功：
1. `session.DoctorInstanceID` 非 nil
2. `model.Instance` 记录能查到且 `InstanceId` 非空
3. `controller.GetDoctorSessionMtimeFn`（通过 TAT 在龙虾医生 CVM 上跑脚本读取 session 文件 mtime）成功返回

以上任一环节失败，代码逻辑均为 `continue`（跳过本次、等下一轮重试），**没有任何不依赖远端探测的
绝对兜底出口**。如果远端探测持续失败（TAT Agent 掉线、CVM 状态异常等——诊断完毕后 CVM 完全可能进入
这类状态），该会话就会永久卡在 `active`，与 TAPD 单描述的"兜底销毁逻辑有缺陷"完全一致。

`controller/doctor.go` 的 `CleanupDoctorSession`（ending 阶段清理逻辑，由 `doctor_ending` 定时任务
调用）已经保证：即使 CVM 销毁失败，也会无条件把 `status` 更新为 `ended`（TAPD 单优化建议第 2 条
"销毁必须联动终态更新"，现状已满足）。真正的缺口是 TAPD 单优化建议第 1 条——"确保 12 小时内会将
新创建的龙虾医生 CVM 销毁掉"，即 active → ending 这一步没有绝对时限兜底。

## 目标

- [x] 定位并确认根因：`cleanupDoctorSessions` 中 active 会话超时判断依赖远端探测成功，无绝对兜底出口
- [ ] 修复：新增不依赖远端调用的绝对超时兜底，确保任一环节异常都不会导致会话永久卡在 `active`
- [ ] 验收标准：`time.Since(session.CreatedAt) > 12h` 的 active 会话，无论 CVM/TAT/mtime 探测结果如何，
      都会被强制推进到 `ending`（进而由 `doctor_ending` 任务清理为 `ended`），释放用户维度互斥锁

## 范围

| 包含 | 不包含 |
|------|--------|
| `task/doctor_cleanup.go`：`cleanupDoctorSessions` 新增绝对兜底超时分支 | 不改 `HandleDoctorStart` 的互斥锁语义 |
| 对应单元测试补充 | 不改 `doctor_ending.go` / `CleanupDoctorSession` 现有清理逻辑（已验证无缺陷） |
| | 不改 `doctor_activate.go`（creating 状态超时兜底已存在且独立正常） |
| | 不做告警/监控层面的增强（超出本次 bugfix 范围） |

## 待确认问题

| # | 问题 | 状态 | 结论 |
|---|------|------|------|
| 1 | 分支命名策略（feature/ 还是 bugfix/fix） | 已确认 | 用户明确要求：这是缺陷，分支用 `bugfix/` 或 `fix/` 前缀，已创建 `bugfix/doctor-active-session-force-timeout` |
| 2 | 方案范围是否需要扩大（如增加告警、扩大兜底到 creating/ending 其他状态） | 已确认 | 用户已审阅方案并明确表示"其他的我看了下，没有什么问题"，按原方案（仅 active 状态新增兜底分支）执行，不扩大范围 |

## 约束与依赖

- 遵循 CODEBUDDY.md 红线：不使用裸 SQL、写接口无新增（本次无 API 变更，不涉及审计规则）、不涉及 GORM model 变更
- 复用现有 `timeout` 常量（12h）与现有 `endDoctorSession` 函数，不引入新依赖
- 不涉及数据库 schema 变更、不涉及 i18n 新增文案
