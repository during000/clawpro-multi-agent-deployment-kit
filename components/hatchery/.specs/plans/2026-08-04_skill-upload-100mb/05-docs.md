# 05. Docs — 文档更新

---

## 更新清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `docs/API.md` | 修改 | `POST /admin/skills/create`：`file` 说明「最大 50MB」→「最大 100MB」 |
| `docs/API.md` | 修改 | `POST /openclaw/skills/contribute`：`file` 说明「最大 50MB」→「最大 100MB」 |

**明确不做（用户确认）：** 不在 API 文档中备注反向代理 / 网关 `client_max_body_size` 限制。

**无需更新：**

- `.specs/docs/` — 无对应模块功能文档需改
- `docs/INDEX.md` — 无索引变更
- 其他 `docs/**` — 全仓检索仅上述两处「Skill 上传 50MB」表述

## 检查项

- [x] `docs/API.md` 已更新（两处参数说明）
- [x] `.specs/docs/` 相关文档已同步（无适用项）
- [x] 参数表格式符合 CLAUDE.md 要求（4 列、无反引号包裹参数名；仅改说明列文案）
