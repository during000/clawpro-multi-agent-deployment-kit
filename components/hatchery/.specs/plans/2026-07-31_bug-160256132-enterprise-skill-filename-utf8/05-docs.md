# 05. Docs — 文档更新（TAPD 160256132）

## 更新清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `docs/API.md` | 修改 | 记录企业技能 ZIP 文件名严格 UTF-8 校验和 400 响应 |
| `i18n/keys.go` | 修改 | 新增非 UTF-8 文件名的中文用户提示 |
| `i18n/en.go` | 修改 | 同步英文翻译 |
| `.specs/plans/2026-07-31_bug-160256132-enterprise-skill-filename-utf8/` | 新增 | 本任务完整 SOP 记录 |

## 检查项

- [x] `docs/API.md` 已更新
- [x] `.specs/docs/` 无本模块对应文档，本任务计划文档已补齐
- [x] 未修改参数表；既有表格仍符合 CLAUDE.md 的四列格式
- [x] 新增 i18n Key 已在 `en.go` 注册
