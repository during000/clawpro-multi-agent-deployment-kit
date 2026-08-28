# 05. Docs — 文档更新

---

## 更新清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `docs/API.md` | 修改 | 所有实时技能返回稳定 slug 和 `can_uninstall`；仅 Admin 下发技能补充版本与更新字段 |
| `docs/API.md` | 新增 | `POST /openclaw/update-skill` 的表单参数、准入、同步响应、幂等与错误码 |
| `docs/API.md` | 新增 | `POST /openclaw/uninstall-skill` 的 Admin 下发/其他运行时技能物理卸载、响应差异、幂等与错误码 |
| `docs/API.md` | 修改 | 接口总览新增更新、卸载路由 |

## 验证

- 三个接口路径与 `main.go` 路由一致。
- 参数表均为「参数 / 类型 / 必填 / 说明」四列。
- JSON 示例覆盖 `name != slug`、Admin 下发/其他运行时/内建技能列表、`can_uninstall`、精简后的更新响应，以及两类卸载响应。
- 本任务没有独立 `.specs/docs/` 功能文档；模块契约统一维护在 `docs/API.md`。

## 检查项

- [x] `docs/API.md` 已更新
- [x] 无相关 `.specs/docs/` 文档需要同步
- [x] 参数表格式符合 CLAUDE.md 要求（4 列、参数名无反引号）
