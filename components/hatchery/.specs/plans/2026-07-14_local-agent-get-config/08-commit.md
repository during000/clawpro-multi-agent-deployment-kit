# 08. Commit — 提交与推送

> 单 commit 提交全部功能 + 测试代码，推送至远端 `feature/local-agent-get-config` 分支。

---

## 一、提交信息

```
feat(local-agent): 新增 GET /local-agent/get-config 接口（CLS 公网配置拉取）

本地 agent 不在 VPC 内、无法用 CAM Role，需主动拉取 CLS 公网接入配置
（endpoint + topic_id 实时查 + 按租户隔离的 AK/SK）。

- 新增 LocalAgentCLSCredential model + 迁移（init.sql / 0708 增量 / MigrateFromSQLite）
- 新增 HandleLocalAgentGetConfig，复用两层白名单 + CheckCLSClawServiceOpened
- 唯一索引 (identifier, config_type) 保证租户隔离
- i18n key MsgLocalGetConfigCredentialNotReady + en.go 翻译
- API.md 文档 + 单测（controller 9 + model 2）
- install_cmd / update_cmd 先留空串常量，值到位后替换即可

Co-Authored-By: OpenClaw (内网版) <noreply@tencent.com>
```

---

## 二、提交范围

| 类别 | 文件 | 数量 |
|------|------|------|
| 功能代码 | `controller/local_agent.go`、`main.go`、`model/db.go`、`model/migrate.go`、`model/local_agent_cls_credential.go`、`i18n/en.go`、`i18n/keys.go`、`sql/init.sql`、`sql/0715-add-local-agent-cls-credential.sql`、`docs/API.md` | 10 |
| 单元测试 | `controller/local_agent_get_config_test.go`、`model/local_agent_cls_credential_test.go`、`controller/local_agent_report_test.go`（共享 helper 迁移 +1 行） | 3 |
| **合计** | | **13 files changed** |

> 排除项：`.specs/plans/2026-07-14_local-agent-get-config/` 是 SOP AI 协作过程文件，不纳入代码仓库。

---

## 三、推送结果

```
[feature/local-agent-get-config ec39ab79] feat(local-agent): ...
 15 files changed, 688 insertions(+)
 * [new branch] feature/local-agent-get-config -> feature/local-agent-get-config
```

远端已创建分支。MR 创建链接：
https://git.woa.com/cvm-openclaw/hatchery/-/merge_requests/new?merge_request%5Bsource_branch%5D=feature%2Flocal-agent-get-config

---

## 四、后续动作（需用户/人工）

1. **创建 MR**：点击上方链接，source=`feature/local-agent-get-config` → target=`master`（或目标 Release 分支）。
2. **CI 验证**：MR 触发后，确认 `go build` / `go vet` / 单测全绿。
3. **install_cmd / update_cmd 填值**：待用户提供具体字符串后，替换 `localAgentCLSInstallCmd` / `localAgentCLSUpdateCmd` 常量，可补一个非空断言用例（可选），再提一个 follow-up commit。
