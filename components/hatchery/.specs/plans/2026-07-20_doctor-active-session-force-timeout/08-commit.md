# 08. Commit — 提交

---

## Commit Message

```
fix(task): 龙虾医生诊断会话新增绝对超时兜底防止永久卡死

修复 TAPD bug 1020422209160782882：cleanupDoctorSessions 对 active
会话的 12h 超时判断完全依赖远端探测（GetDoctorSessionMtimeFn/CVM/
TAT）成功，探测持续失败或 DoctorInstanceID/Instance 记录异常时会
被 continue 永久跳过，导致会话卡死在 active，触发用户维度创建互
斥锁，报错 active_session_exists。

新增不依赖任何远端调用的绝对兜底判断：会话自创建起超过 12h 后强
制调用 endDoctorSession 推进到 ending，交由 doctor_ending 任务清
理。仅新增前置分支，不改动任何既有分支逻辑和顺序。

task/doctor_cleanup.go: 新增兜底超时分支
task/doctor_cleanup_test.go: 新增 4 个用例（探测失败超时/
DoctorInstanceID为空超时/实例不存在超时/未超时不误伤回归）
```

## 提交前检查清单

- [x] `go build ./...` 通过
- [x] `go test ./task/... ./controller/...` 全量通过，无回归
- [x] 无裸 SQL / 无硬编码密钥 / 无 API 变更 / 无 DB schema 变更
- [x] `.specs/plans/2026-07-20_doctor-active-session-force-timeout/` 下 00~08 全部产物已就位
- [x] 分支 `bugfix/doctor-active-session-force-timeout` 已通过 gongfeng MCP `create_branch` 创建

## 提交方式

遵循项目规则，通过 gongfeng MCP `batch_modify_files` 提交到远端分支
`bugfix/doctor-active-session-force-timeout`，涉及文件：
- `task/doctor_cleanup.go`
- `task/doctor_cleanup_test.go`
- `.specs/plans/2026-07-20_doctor-active-session-force-timeout/*.md`（9 个产物文件）
