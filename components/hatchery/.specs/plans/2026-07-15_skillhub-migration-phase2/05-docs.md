# 05. Docs — 文档更新

---

## 更新清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `docs/API.md` | 新增 | `/admin/skillhub-status` 接口文档（请求/响应/字段说明） |
| `PHASE3-INTERFACE-MIGRATION-GUIDE.md` | 新增 | Phase 3 接口迁移指南（项目根目录） |

---

## 检查项

- [x] `docs/API.md` 已更新（新增 `/admin/skillhub-status`）
- [x] 参数表格式符合 CLAUDE.md 要求（4 列、无反引号包裹参数名）
- [x] Phase 3 迁移指南已输出到项目根目录

---

## API.md 新增内容摘要

### GET /admin/skillhub-status

查询当前租户的 SkillHub 灰度状态。

**响应**:
```json
{
  "ok": true,
  "enabled": false,
  "skillhub_url": "https://skillhub.cn"
}
```

- `enabled`: 是否启用 SkillHub 灰度（对应 `site_configs.skill_hub_enabled`）
- `skillhub_url`: 前端跳转地址（由 `skill_hub_api_url` 去掉 `api.` 前缀推导）
