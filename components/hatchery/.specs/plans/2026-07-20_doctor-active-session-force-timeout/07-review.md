# 07. Review — 代码审查结果

---

## 审查维度自查

| 维度 | 结论 |
|------|------|
| 裸 SQL（`db.Exec`/`db.Raw`/`db.Table`） | 未使用，全部走 GORM 接口 |
| 多租户隔离（`model.DB(ctx)` vs `model.DB`） | 新增代码复用循环内已有的 `ctx`，`endDoctorSession(ctx, &s)` 正确传递 tenant 上下文 |
| 审计日志 | `endDoctorSession` 内部已调用 `model.LogAudit`（既有逻辑，未改动），本次新增分支复用该函数，审计链路完整 |
| 数据库/model 变更 | 无 |
| API 兼容性 | 无 API 改动 |
| 硬编码密钥/配置 | 无 |
| goroutine context | **新增** `startProbeChecker` 内 goroutine，已用 `common.DetachContext(ctx)` + `context.WithTimeout(55min)` 包裹，且循环等待用 `select/case <-ctx.Done()` 响应取消，无泄漏 |
| 越权/权限检查 | 定时任务内部逻辑，不涉及用户请求鉴权 |
| i18n | 无新增用户可见文案 |

## 逆向检查：改动是否可能引入新问题

- **幂等性**：`endDoctorSession` 只把 `status` Update 为 `ending`，`runProbeChecker` 在进入兜底结束前先校验
  `s.Status == model.DoctorStatusActive`，非 active 直接退出，不会误触发。
- **并发安全**：`cleanupDoctorSessions` 由 `model.TryLock(ctx, "doctor:cleanup")` 保护；`startProbeChecker`
  额外用 `doctor:probe:{sessionID}` 分布式锁保证同一 session 全局只有一个验证协程，多实例部署安全。
- **DB 持续故障时误杀**：`runProbeChecker` 将 `probeFails`（探测失败）与 `dbFails`（DB 查询失败）**分计**，
  `dbFails >= maxAttempts` 时直接退出（用零值 `s` 兜底判断被规避），避免 DB 故障导致零值 session 被误 `ending`。
- **是否会跳过应有的清理步骤**：`endDoctorSession` 仅推进到 `ending`，CVM 销毁/快照清理仍由 `doctor_ending`
  的 `CleanupDoctorSession` 完成，链路兼容。
- **是否可能误杀正常会话（与原方案差异点）**：原 Plan 方案为「`CreatedAt > 12h` 立即 `ending`」，会在 TAT
  偶发抖动时误杀活跃会话；本实现改为「后台协程每 5min 重试，连续 10 次失败**且** `CreatedAt > 12h` 才结束」，
  降低偶发故障误杀概率，仍保证不会永久卡 active。

## 结论

未发现问题，符合 CODEBUDDY.md 红线要求。但**本实现与 02-plan 最初方案存在重大偏离**（见 02-plan / 03-implement），
已在文档中如实回写。

## 流程改进建议（防止方案-实现严重偏离）

本次偏离是在实现阶段才发现并调整的，Plan 阶段未预判「偶发故障 vs 持续故障」的兜底策略差异。建议在
工作流中增加**实现-方案一致性 gate**（已落实于 `.codebuddy/agents/developer.md` Step 4.3 与
`.codebuddy/agents/solo-developer.md` Step 5，并在 `devflow.defaults.yaml` 的 CODE-REVIEW 前置校验中
明确包含一致性检查）：

1. **developer 编码后**：对比实际 `git diff` 与 `execution-plan` 的 `parallel_tasks` / `files_whitelist`，
   发现方案未涵盖的公开函数/文件/行为逻辑变化，必须**回写** `tech-design.md` + `execution-plan.md`，
   并在 `change-report.md` 标注 `deviation`，禁止静默写「与 Plan 无差异」。
2. **solo-developer 编码后**：`solo-report.md` 的「简化设计」与最终 diff 对比，有偏离必须回写。
3. **code-reviewer 前置校验**：除越界 + acceptance 外，新增**方案一致性**校验——若 `change-report` /
   `solo-report` 声称「无差异」但实际 diff 偏离上游方案，打回 developer 补写偏离说明。
4. **Plan 阶段强制预判兜底策略**：对「超时/失败兜底」类需求，方案阶段须显式区分「偶发故障」
   与「持续故障」两种假设并给出对应策略，避免实现时才发现策略分歧。
