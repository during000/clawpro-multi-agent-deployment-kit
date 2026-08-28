# 05. Docs — 文档更新核对

> API.md 在 Implement 阶段已更新（红线 9），本步骤核对无遗漏项。

---

## 一、已更新文档（Implement 阶段完成）

### `docs/API.md`
- **目录表**（71 行）：新增 `| 本地 agent | /local-agent/get-config | GET |`
- **接口文档块**（9421 行起）：`### GET /local-agent/get-config 🆕`
  - 接口背景（公网 CLS endpoint + 永久 AK/SK，区别于 CVM 内网域名）
  - 权限、Content-Type、Query 参数表（config_type）
  - 响应示例 + 字段表（含 `endpoint` 公网区分说明、`secret_id/secret_key` 敏感标记、`install_cmd/update_cmd` 固定值）
  - 错误响应全覆盖：`401` / `403` / `400(config_type 无效)` / `400(CLS 服务未开启)` / `500(CLS 凭据未配置)` / `405`
  - 安全约束：`secret_key` 仅响应出现、禁止落日志；凭据按租户隔离

### `i18n/keys.go` + `i18n/en.go`（Implement 阶段）
- 新增 `MsgGetConfigCredentialNotReady` key + 英文翻译（红线 7：新增 i18n.Key 必须加 en.go）
- 复用既有 `MsgInvalidConfigType` / `MsgCLSServiceNotEnabled`，英文翻译已存在

---

## 二、经核对无需更新的文档

| 文档 | 原因 |
|------|------|
| `docs/INDEX.md` | 仅索引大文档模块，不列每个接口；本地 agent 文档入口在 iWiki 技术方案（p/4022150701），不在 `docs/` 下 |
| `docs/i18n.md` | 国际化方案指引，不维护逐 key 清单 |
| `docs/testing.md` | 单测编写指引，本次未引入新的通用测试模式（seam / 内存 sqlite / CookieStore 均为既有惯例） |
| `docs/openapi.json` | 由 CI 通过 `test/api_md_to_openapi.py` 从 API.md 自动生成（`.ci/integration-test.yml` + `Makefile openapi`），无需手动编辑；API.md 已正确，openapi 会在 CI 自动再生 |

---

## 三、红线核对

| 红线 | 状态 |
|------|------|
| 红线 6（文案硬编码 → i18n） | ✅ handler 全部走 `i18n.MsgXxx`，无硬编码 |
| 红线 7（新增 i18n.Key 加 en.go） | ✅ `MsgGetConfigCredentialNotReady` 已加英文翻译 |
| 红线 9（改 API 必须更新 docs/API.md） | ✅ 接口文档 + 目录表已更新 |
| 红线 13（新增接口必须有集成测试） | ⏭ 集成测试在 06. IT 步骤完成 |

---

## 四、待办

- 06. IT 步骤需补端到端集成测试（`test/` 下），覆盖 config_type 参数传入 + 正常/错误路径（红线 13）。
- `install_cmd` / `update_cmd` 具体字符串值待用户提供；给出后需回填 `handler` 常量 + 可补一个非空断言用例（可选）。
