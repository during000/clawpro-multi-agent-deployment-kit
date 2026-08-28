# 05. Docs — 文档更新

> 三期需求前端依赖 API.md，本期先补全协议文档（功能代码 F1-F6 实现时如有字段偏差再回来修正）。

---

## 更新清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `docs/API.md` | 修改 | 目录表新增 `/local-agent/remove`、`/admin/local-agent/remove`；sync 响应新增 `cmds` 统一字段列表（双列表兼容）；ack type 枚举扩展 + 新增 `uninstall_teamai`/`install_hook_rule`/`uninstall_hook_rule` 语义与软删说明；新增两个移除接口章节（用户端+管控端，含幂等/错误响应）；`POST /admin/rules/create` 补充 `type=hook` 分支（event/cmd 表单字段） |

## 本次文档覆盖的需求点

- 移除本地 Agent（双端接口 + 执行链路 + 软删/重试语义）
- sync 协议演进（commands + cmds 双写，handle_type 统一字段，uninstall_teamai 命令格式）
- Hook 资源创建（EnterpriseRule type=hook，event + cmd 表单）
- ack 路由（hook 走 rule 管道、uninstall_teamai 走本地任务管道 + 软删激活）

## 检查项

- [x] `docs/API.md` 已更新（API 变更全部覆盖）
- [x] 目录表与章节同步
- [x] 参数表格式符合 4 列要求
- [ ] 功能代码 F1-F6 实现后回查字段是否一致（如 cmd 落表字段名、local_agent_tasks 字段、hook Event 字段名）

## 待实现阶段对齐

文档按 iWiki 方案定稿编写，字段命名以下方为准（代码实现时严格对齐）：

- `local_agent_tasks`：`id` / `identifier` / `instance_id` / `instance_c_id` / `type=uninstall_teamai` / `cmd` / `status`(pending|success|failed|cancelled) / `error` / `operator_id`
- remove 响应：`{ ok, task_id, status }`
- sync cmds item：`id` / `type` / `slug` / `version` / `handle_type` / `event` / `cmd` / `download_url` / `scope` / `workspace_path` / `project_id`
- hook 创建：`POST /admin/rules/create`，`type=hook` + `event` + `cmd` 表单字段
