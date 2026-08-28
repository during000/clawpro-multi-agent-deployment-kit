# 03. Implement — 实现

---

## 关键实现细节

### `task/doctor_cleanup.go`

> **注意**：本实现与 02-plan 最初设计的「循环最前面 8 行前置兜底分支」有**重大偏离**，
> 已在 02-plan「与最初方案差异」章节及本文件末尾「与 Plan 的差异」中如实记录。
> 偏离原因：前置分支方案会把「TAT 偶发抖动但会话实际仍活跃」的会话在满 12h 时直接 `ending` 误杀，
> 故改为后台验证协程机制（见下）。

#### 1. `cleanupDoctorSessions` 的 mtime 获取失败分支改走后台验证协程

原 `result.Err != nil` 分支由 `continue`（等下轮）改为启动 `startProbeChecker`：

```go
result := controller.GetDoctorSessionMtimeFn(ctx, doctorInst.InstanceId)

// 获取失败：启动后台验证协程持续探测，
// 只有持续失败超过 12 小时才兜底结束，避免 TAT 偶发故障导致误杀
if result.Err != nil {
    sessionLog.Error("[DoctorCleanup] 无法获取 session mtime，启动验证协程",
        "error", result.Err)
    startProbeChecker(ctx, s, doctorInst.InstanceId, log)
    continue
}
```

#### 2. 新增 `startProbeChecker`：为单 session 启动后台验证协程

- 用分布式锁 `doctor:probe:{sessionID}` 保证同一 session 全局只有一个协程在跑（多实例部署安全）。
- 用 `common.DetachContext(ctx)` + `context.WithTimeout(55min)` 包裹，确保协程不绑死原 ctx 且自然到期退出。
- 获取锁失败（已有协程在跑）则直接跳过，不重复启动。

#### 3. 新增 `runProbeChecker`：验证协程核心探测循环

- 每 `interval`（5min）重试一次 `GetDoctorSessionMtimeFn` 探测，最多 `maxAttempts`（10）次。
- `probeFails`（探测失败计数）与 `dbFails`（DB 查询失败计数）**分计**：
  - 探测成功 → 退出（会话恢复，交给原定时任务路径）。
  - `dbFails >= maxAttempts`（DB 持续不可用）→ 直接退出，避免使用零值 `s` 误杀。
  - `probeFails >= maxAttempts`：仅当 `time.Since(s.CreatedAt) > probeTimeout(12h)` 才 `endDoctorSession`；
    未超时则退出，等下轮 `doctor-cleanup` 重新启动协程。
- 循环等待用 `select { case <-ctx.Done(): / case <-time.After(interval): }`，响应 context 取消，避免 goroutine 泄漏。
- session 中途被删除或非 active → 立即退出。

- 复用既有 `timeout`（12h）常量与 `endDoctorSession`，语义不变（status=ending → doctor_ending 清理）。
- 正常探测成功路径行为完全不变（仍由 `cleanupDoctorSessions` 主循环按 mtime 判断）。

## 与 Plan 的差异

**重大偏离（必须记录）**：

| 维度 | 02-plan 最初方案 | 最终实现 |
|------|------------------|----------|
| 触发位置 | `cleanupDoctorSessions` for 循环最前面，无条件 | mtime 探测 `result.Err != nil` 分支内启动协程 |
| 兜底逻辑 | `CreatedAt > 12h` 立即 `endDoctorSession` | 后台协程每 5min 重试，连续 10 次失败**且** `CreatedAt > 12h` 才结束 |
| 代码量 | +约 8 行单分支 | +`startProbeChecker`/`runProbeChecker` 两函数（约 90 行）+ 分布式锁 |
| 对「TAT 偶发失败」 | 满 12h 直接杀（可能误杀活跃会话） | 持续重试给远端恢复机会，确认真死才杀 |

**偏离决策依据**：原方案会在 TAT 偶发抖动（非持续故障）场景下，把实际仍活跃、只是探测失败的会话在满 12h 时强制 `ending`，影响用户可用时长。改为后台验证协程后，只有「探测持续失败且真实超 12h」才兜底结束，避免了偶发故障误杀，同时仍保证不会永久卡在 active。

**回归验证**：`go build ./...` 通过；`go test ./task/... ./controller/...` 全部通过（含 4 个新增用例，本次实际新增的 `TestCleanupDoctorSessions_获取失败则启动验证协程` 与 `TestRunProbeChecker_Context取消则退出` 覆盖了协程启动与 context 取消退出路径）。

## 验证

- `go build ./...`：通过
- `go test ./task/...`：全部通过（含新增 4 个用例）
- `go test ./controller/...`：全部通过（回归无影响，本次未改动 controller 包）
