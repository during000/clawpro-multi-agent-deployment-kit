# 07. Review — 代码审查

---

## 审查方式

- [x] AI 自动 Review
- [ ] 人工 Review

## 发现的问题

| # | 文件 | 行 | 问题 | 严重度 | 状态 |
|---|------|---|------|-------|------|
| 1 | `controller/openclaw_create_service.go` | 44-95 | 原持久化通道预设 worker 在多副本下没有原子 claim，可能重复下发 | 高 | 已修：删除预设表、恢复 worker 和状态接口；改为当前进程内等待 ready 后单次下发 |
| 2 | `controller/openclaw_model.go` | 1465-1541 | 原方案预先批量写模型绑定，下发失败时不符合用户 `add-model` 的回滚语义 | 高 | 已修：统一调用 `applyBuiltinModel`，逐个写入；TAT 失败删除本次绑定并停止后续模型 |
| 3 | `controller/openclaw_channel.go` | 101-237 | 创建期通道自建执行逻辑会与用户 `set-channel` 的能力、可见性、参数和错误边界分叉 | 中 | 已修：创建期与用户端共用 `validateManualChannelConfig` / `applyManualChannelConfig` |
| 4 | `controller/openclaw_skill.go` | 443-504 | 创建期企业技能可能绕过用户 `add-skill` 的 source、可见性和安装语义 | 高 | 已修：共用 public/enterprise helper；空 source 默认 public；追加技能不创建安装任务 |
| 5 | `controller/access_log.go`；`controller/audit.go`；`controller/tat.go` | 175-181；80；316-331 | 请求体、审计详情、TAT 参数或脚本失败输出可能泄露通道凭据 | 高 | 已修：跳过管理员创建 body，审计不采集 body，日志仅记录参数数量，set-channel 失败输出不记录 |
| 6 | `controller/admin_instance_create.go` | 61-74 | 宽松 JSON 解码可能静默接受未知字段、多个对象或超大请求体 | 中 | 已修：限制 1 MiB、启用 `DisallowUnknownFields`，并要求第二次 decode 返回 EOF |

## 审查通过确认

- [x] 无高严重度未修复问题
- [x] 代码风格一致
- [x] 安全基线检查通过
