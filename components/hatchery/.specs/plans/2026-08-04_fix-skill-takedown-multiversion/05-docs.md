# 05. Docs — 文档更新清单

> 记录本次 bugfix 涉及的文档更新。

---

## 一、已更新文档

| 文件 | 改动 | 说明 |
|------|------|------|
| `docs/API.md` | 修改 | 明确 takedown / approve / `pending_review` 的整 slug 语义与按 slug 关联规则；顺带去掉 contributions 列表段一处多余的 \`\`\` |

---

## 二、变更要点

### `POST /openclaw/skills/takedown`

- 写明：审核通过后整 slug 所有 published → offline
- `resource_id` 绑最新 published
- 归属：存在本人 published 版本即可

### `POST /admin/contributions/approve`

- `takedown`：按 slug 批量 offline（兼容旧 resource_id）

### `GET /admin/skills` 字段扩展

- `pending_review` 按 **slug** 关联 pending，而非 `skill.id == resource_id`

---

## 三、不需要更新的文档

| 文件 | 原因 |
|------|------|
| `sql/*` | 无 schema 变更 |
| `docs/INDEX.md` | 无新模块 |
| OpenAPI 路径 | 无新增/删除接口，仅语义澄清 |

---

## 四、格式合规

- [x] 未改参数表结构（无新增请求参数）
- [x] 语义补充写在说明段落，不破坏 OpenAPI 参数解析表头
- [x] 表格前后空行保持
