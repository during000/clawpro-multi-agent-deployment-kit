# [2026-07-15] SkillHub 迁移 Phase 2 — 技能列表灰度代理

> **本文件是本任务的单一真相源（Single Source of Truth）**：任务元信息、进度、当前步骤、关键决策全部在这里。
> 会话恢复时，先读本文件定位当前步骤，再按需加载对应阶段文件。

> ⚠️ Meta 中的 `分支` 字段是上下文恢复时定位任务的唯一依据，**必须**与 `git branch --show-current` 的输出完全一致。

---

## Meta

| 项 | 值 |
|----|----|
| 分支 | `feat/skill-migration-phase2` |
| 摘要 | Phase 2：技能列表 API 灰度代理，通过 Gateway 获取 OneID access_token 转发 SkillHub，装饰器模式无侵入切换 |
| 状态 | 已完成 |
| 创建日期 | 2026-07-15 |
| 负责人 | grassylcao |
| 预期完成 | 2026-07-15 |

---

## Progress

<!--
Progress 更新规则（AI 必读）：
1. 步骤完成时，**原地替换** `- [ ] N. <step>` 为 `- [x] N. <step> (<结果摘要>)`
2. 禁止插入新行或重复编号，每个编号 01-08 有且仅有一行
3. 结果摘要示例：`(15/15 passed)`、`(覆盖率 92%)`、`(全量通过)`
4. 同步更新下方「当前步骤」章节
-->

- [x] 01. Clarify    → [01-clarify.md](./01-clarify.md) (灰度开关+Gateway代理token+装饰器模式，5项待确认已全部解决)
- [x] 02. Plan       → [02-plan.md](./02-plan.md) (13文件改动清单+调用链+15单测设计)
- [x] 03. Implement  → [03-implement.md](./03-implement.md) (13文件落地，build/vet通过)
- [x] 04. UT         → [04-ut.md](./04-ut.md) (3包34单测全绿，覆盖率>85%)
- [x] 05. Docs       → [05-docs.md](./05-docs.md) (API.md新增skillhub-status端点)
- [x] 06. IT         → [06-it.md](./06-it.md) (6用例8请求全PASS)
- [x] 07. Review     → [07-review.md](./07-review.md) (AI review 4项已修，人工CR已通过)
- [x] 08. Commit     → [08-commit.md](./08-commit.md) (commit b65a0c5d，已推送)

---

## 当前步骤

> 恢复会话时，优先读取此处指向的阶段文件。

- **步骤**：✅ Done
- **文件**：[08-commit.md](./08-commit.md)
- **上次更新**：2026-07-15 18:00

---

## 时间记录

| 步骤 | 开始时间 | 结束时间 | 耗时 | 备注 |
|------|---------|---------|------|------|
| 01. Clarify | 2026-07-15 10:00:00 | 2026-07-15 11:00:00 | 60m | 方案多轮演进，5项待确认全部解决 |
| 02. Plan | 2026-07-15 11:00:00 | 2026-07-15 12:00:00 | 60m | 装饰器模式+Gateway代理方案确定 |
| 03. Implement | 2026-07-15 13:00:00 | 2026-07-15 16:00:00 | 180m | 13文件实现+CI/CD schema修复 |
| 04. UT | 2026-07-15 16:00:00 | 2026-07-15 17:30:00 | 90m | 3包34单测 |
| 05. Docs | 2026-07-15 17:30:00 | 2026-07-15 17:45:00 | 15m | API.md更新 |
| 06. IT | 2026-07-15 17:45:00 | 2026-07-15 18:30:00 | 45m | 8用例全PASS |
| 07. Review | 2026-07-15 17:50:00 | 2026-07-16 10:00:00 | 16h | AI review+人工CR，9项全部修复 |
| 08. Commit | 2026-07-16 10:00:00 | 2026-07-16 10:15:00 | 15m | commit b65a0c5d |

---

## 关键决策备忘

> **跨阶段共享的关键上下文**。仅记录影响后续步骤的决策，避免恢复时还要翻阅历史阶段文件。

- **灰度控制**：`site_configs.skill_hub_enabled`（bool）控制是否走 SkillHub 代理，与 `skill_hub` 字段分离（后者用户可改，前者管理员控制）
- **token 获取方式**：Hatchery 无 OneID 私钥，通过 Gateway `POST /api/access-token` 转发获取用户级 access_token（非直接调 OneID）
- **装饰器模式**：`WithSkillHubProxy(localHandler, skillHubHandler)` 无侵入路由切换，不修改 `admin_skills.go` 原有逻辑
- **缓存模式**：access_token 缓存 key=`"identifier:sub:tid"`，OrgInfo 缓存 key=`"identifier"`，均使用 `sync.Mutex + map`（与 `getOneIDAppToken` 同模式）
- **前端 URL 推导**：`skill_hub_api_url` 去掉 `api.` 前缀得到前端 URL，如 `https://api.skillhub.cn` → `https://skillhub.cn`
- **API URL 分离**：`skill_hub_api_url`（API 地址，admin 控制）与 `skill_hub`（前端展示 URL，用户可改）分开存储
- **错误处理**：SkillHub API 调用失败返回 502，不静默降级到本地
- **Gateway 端点**：`POST /api/access-token`，请求体 `{"sub":"...", "tid":"..."}`，`aud_app_type` 在 Gateway 写死为 `"skillhub"`，Hatchery 不传
- **OrgID 发现**：通过 SkillHub `GET /api/v1/auth/me` 获取，按租户缓存（一个 OneID 租户在 SkillHub 的 orgId 唯一）
- **Phase 3 指南**：根目录 `PHASE3-INTERFACE-MIGRATION-GUIDE.md` 指导后续接口迁移

---

## 风险速览

| # | 风险 | 严重度 | 缓解 |
|---|------|-------|------|
| 1 | 多 pod 副本下 token 缓存不共享 | 中 | 缓存提前 60s 刷新，过期后各 pod 独立获取新 token，可接受 |
| 2 | SkillHub API 不可用时全部 502 | 中 | 灰度控制可快速关闭 `skill_hub_enabled` 回退本地 |
| 3 | Gateway 不可用时无法获取 token | 中 | Gateway 与 Hatchery 同集群部署，同生命周期 |

---

## 文件索引

| 文件 | 产物 |
|------|------|
| [00-overview.md](./00-overview.md) | 任务总览（本文件） |
| [01-clarify.md](./01-clarify.md) | 需求澄清：背景、目标、范围、待确认问题 |
| [02-plan.md](./02-plan.md) | 方案设计：改动文件、调用链、测试用例、风险 |
| [03-implement.md](./03-implement.md) | 实现：关键细节、与 Plan 差异 |
| [04-ut.md](./04-ut.md) | 单元测试：用例、覆盖率、未覆盖行 |
| [05-docs.md](./05-docs.md) | 文档更新清单 |
| [06-it.md](./06-it.md) | 集成测试：构建部署、端到端验证、增量覆盖率 |
| [07-review.md](./07-review.md) | Code Review：问题与修复 |
| [08-commit.md](./08-commit.md) | Commit message 与提交前检查 |
