# 05. Docs — 文档更新

---

## 更新清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `docs/API.md` | 修改 | 新增 `POST /admin/instances/create` 的权限、Content-Type、1 MiB 上限、严格 JSON、请求参数、响应、错误码和异步语义 |
| `.specs/docs/API.md` | 同步 | 与 `docs/API.md` 为同一 inode，API 修改已同步可见 |
| `.specs/plans/2026-07-09_clawpro-agent-admin-create-assign/` | 新增/修改 | 按阶段模板记录 Clarify、Plan、Implement、UT、Docs、IT、Review 和 Commit 产物 |

API 文档明确以下契约：

- HTTP 200 只表示 CVM 创建成功，且非空 presets 已在当前进程中安排等待 Agent ready；不表示 model/channel/skill 已完成。
- 指定 models 时覆盖站点默认模型；单模型 runtime 只接受一个模型，OpenClaw 兼容 runtime 才支持 fallbacks，3.28.x 不支持 fallback。
- channel 和追加 skill 不新增持久化、状态查询、恢复或重试；角色技能和技能包技能保持原任务流程。
- 管理员响应只返回 `ok` 和与用户创建接口一致的 CVM `instance_id`，不返回数据库主键、虚假的 queued 状态或凭据。
- 请求体、通道凭据、TAT 参数值和 set-channel 脚本失败输出不得进入日志。

## 检查项

- [x] `docs/API.md` 已更新（涉及新增 API）
- [x] `.specs/docs/` 相关文档已同步
- [x] 参数表格式符合 CLAUDE.md 要求（4 列、无反引号包裹参数名）
