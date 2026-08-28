# 05. Docs — 文档更新清单

> 记录本次需求涉及的文档更新。

---

## 一、已更新文档

| 文件 | 改动 | 说明 |
|------|------|------|
| `docs/API.md` | 新增 | 在"技能广场"和"用户端 MCP 管理"之间新增「技能共建审核」章节，包含 8 个接口的完整文档 |

---

## 二、API.md 新增内容

### 新增章节：技能共建审核

位置：`docs/API.md`，在 `GET /admin/skills/download` 之后、`用户端 MCP 管理` 之前。

包含 8 个接口：

| # | 接口 | 方法 | 权限 | 说明 |
|---|------|------|------|------|
| 1 | `/openclaw/skills/contribute` | POST | 登录用户 | 提交技能/新版本（multipart，参数同 `/admin/skills/create`） |
| 2 | `/openclaw/skills/takedown` | POST | 登录用户 | 申请下架（slug + reason） |
| 3 | `/openclaw/skills/contributions` | GET | 登录用户 | 我的申请列表（支持 status/action_type 筛选） |
| 4 | `/openclaw/skills/contributions/detail` | GET | 登录用户 | 申请详情（申请人或管理员可查看） |
| 5 | `/admin/contributions` | GET | 管理员 | 所有申请列表（支持 resource_type/action_type/status 筛选） |
| 6 | `/admin/contributions/detail` | GET | 管理员 | 申请详情（含 Skill 信息） |
| 7 | `/admin/contributions/approve` | POST | 管理员 | 审核通过 |
| 8 | `/admin/contributions/reject` | POST | 管理员 | 审核拒绝（含 review_comment） |

### 格式合规检查

- [x] 参数表头使用 `| 参数 | 类型 | 必填 | 说明 |`（4 列）
- [x] 参数名不加 backtick（如 `slug` 而非 `` `slug` ``）
- [x] "必填"列值：`是` / `否`
- [x] POST 接口含 Content-Type 声明
- [x] 响应字段表使用 `| 字段 | 类型 | 说明 |`（3 列）
- [x] 表格前后有空行
- [x] 错误码表独立列出

---

## 三、不需要更新的文档

| 文件 | 原因 |
|------|------|
| `docs/INDEX.md` | 仅文件目录索引，不列功能模块 |
| `docs/i18n.md` | i18n 方案不变，仅新增 Key |
| `docs/testing.md` | 测试规范不变 |
| `docs/agent-config/skill-visibility-api.md` | 可见范围接口不变 |
