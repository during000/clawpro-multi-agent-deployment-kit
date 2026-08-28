# 02. Plan — 方案设计

---

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `task/doctor_cleanup.go` | 修改 | `cleanupDoctorSessions` 的 mtime 获取失败分支改走 `startProbeChecker`；新增 `startProbeChecker`/`runProbeChecker` 后台验证协程 |
| `task/doctor_cleanup_test.go` | 修改 | 新增单测：mtime 持续失败+超 12h 强制结束 / DoctorInstanceID 为 nil+超 12h / 实例缺失+超 12h / 未超 12h 探测失败不误伤 / 协程 context 取消退出 / 获取失败启动验证协程 |

## 调用链 / 数据流

```
定时任务 doctor-cleanup（5min）
  → cleanupDoctorSessions(ctx)
    → 查询 status=active 的 DoctorSession 列表
    → for each session:
        DoctorInstanceID / Instance / 目标实例 检查（原有逻辑不变）
        → GetDoctorSessionMtimeFn 探测 mtime
            → 失败（result.Err != nil）：startProbeChecker(ctx, s, cvmID, log) → continue（不再本轮直接结束）
            → 成功：按 mtime 判断是否超 12h → 超时则 endDoctorSession
  → startProbeChecker：分布式锁 doctor:probe:{sessionID} 保证单协程
    → goroutine 内 runProbeChecker（每 5min 重试，最多 10 次）
        → 探测恢复成功 → 退出（交给主循环）
        → 连续 10 次失败 + CreatedAt > 12h → endDoctorSession(ctx, &s) → status=ending
        → 连续 10 次失败但 CreatedAt <= 12h → 退出，等下轮 doctor-cleanup 重启
        → ctx 取消 / DB 持续失败 → 退出
  → endDoctorSession 将 status 置为 ending
定时任务 doctor-ending（1min）
  → processDoctorEnding(ctx) → CleanupDoctorSession(ctx, &session) → 销毁 CVM → status=ended
```

## 数据库变更

无。

## 测试用例设计（自然语言描述）

> 先于实现编写，Implement 阶段据此编码。

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | mtime 探测持续失败但会话已超 12h | active 会话 `CreatedAt` = now-13h，`GetDoctorSessionMtimeFn` mock 返回 `Err != nil` | 会话被强制置为 `ending`（不应因探测失败被 continue 跳过） | P0 |
| 2 | DoctorInstanceID 为 nil 但会话已超 12h | active 会话 `CreatedAt` = now-13h，`DoctorInstanceID = nil` | 会话被强制置为 `ending` | P0 |
| 3 | 关联 Instance 记录已不存在但会话已超 12h | active 会话 `CreatedAt` = now-13h，`DoctorInstanceID` 指向已删除/不存在的 Instance | 会话被强制置为 `ending` | P0 |
| 4 | 会话未超 12h，mtime 探测失败 | active 会话 `CreatedAt` = now-1h，`GetDoctorSessionMtimeFn` mock 返回 `Err != nil` | 会话保持 `active`（不应被误伤提前结束） | P0 |
| 5 | 会话未超 12h，mtime 探测正常且未空闲超时（回归） | active 会话 `CreatedAt` = now-1h，mtime 正常 | 会话保持 `active`，原有逻辑不受影响 | P1（回归） |

### 集成测试（IT）

无新增（本次改动为纯后台定时任务内部逻辑修复，不涉及 API 契约变更，现有 IT 用例已覆盖 `/openclaw/doctor/*` 接口契约，无需新增）。

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| 1 | 兜底分支误伤探测正常但业务仍在使用的会话 | 低 | 中 | 阈值复用既有 12h 常量，与现有 mtime 判断超时阈值完全一致，不缩短用户可用时长 |
| 2 | 新分支破坏现有 for 循环内其他判断的执行顺序 | 低 | 中 | 新分支放在循环最前面并 `continue`，不修改后续任何既有分支的代码和顺序 |

## 与最初方案差异（实现阶段重大偏离，已回写 03-implement）

最初方案为「`cleanupDoctorSessions` for 循环最前面加 8 行前置兜底分支：
`time.Since(s.CreatedAt) > timeout` 即 `endDoctorSession`」。

实际实现改为「mtime 探测失败 → 启动 `startProbeChecker` 后台验证协程，
每 5min 重试，连续 10 次失败**且** `CreatedAt > 12h` 才兜底结束」。

偏离原因：前置分支方案会在 TAT 偶发抖动（非持续故障）场景下，把实际仍活跃、只是探测失败的
会话在满 12h 时强制 `ending`，缩短用户可用时长、可能误杀。后台协程机制给远端恢复机会，
只有确认真死（持续失败 + 超 12h）才兜底，同时仍保证不会永久卡在 active。

> ⚠️ 此偏离是在实现阶段发现并调整的，未在 Plan 阶段预判。后续同类任务应在 Plan 阶段
> 即评估「偶发故障 vs 持续故障」的兜底策略差异（见 07-review 的流程改进建议）。
