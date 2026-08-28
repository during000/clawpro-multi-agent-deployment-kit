# 01. Clarify — 需求澄清

> 本任务的需求澄清在 2026-07-27 与用户（alexwhwang）多轮对话中完成，结论已固化到 iWiki 技术方案：
> https://iwiki.woa.com/p/4027911568

---

## 背景

TAPD 需求【clawpro】接管本地agent三期（1020422209135500434）。hatchery 已具备本地 Agent 一/二期能力（report/sync/ack、skill/rule 下发、scope 绑定）。三期核心：支持移除本地 Agent、sync 协议统一化、Hook 资源纳管、codex 纳管。

## 目标

- 用户端/管控端可移除本地 Agent（卸载 clawpro-teamai 插件 + 解绑，不动本地 AI 工具与已装资源）
- sync 返回新增 cmds 统一字段列表，兼容新老 reporter
- 企业规范库支持 Hook 资源（表单创建、下发、卸载）
- 不活跃阈值 7 天、Agent 列表云端/本地筛选、codex 纳管

## 范围

| 包含 | 不包含 |
|------|--------|
| local_agent_tasks 通用任务表 | 野鹤白名单（用户明确跳过） |
| 双端移除接口 + sync/ack 扩展 | 重复提交去重/已跳过（已有） |
| sync cmds 双列表 | workspace 组织绑定反问（插件侧） |
| Hook（复用 EnterpriseRule） | 临时会话保存为工作空间（CB/WB 原生接口依赖） |
| 7 天阈值 / codex / 列表筛选 | 接入 Prompt 前端拼接逻辑（后端仅供参数） |

## 已确认问题（全部拍板）

| # | 问题 | 状态 | 结论 |
|---|------|------|------|
| 1 | 卸载任务承载 | ✅ | 独立通用表 local_agent_tasks，type=uninstall_teamai |
| 2 | ack 成功后实例处理 | ✅ | 仅软删 instances 行；重新 report 激活（Unscoped 恢复） |
| 3 | 不活跃第三态？ | ✅ | 无第三态，stopped，阈值 7 天 |
| 4 | status 枚举 | ✅ | 对标 rule_distribution_records |
| 5 | cmd 生成时机 | ✅ | 创建任务时落表 |
| 6 | 移除接口入参 | ✅ | 双端统一 instance_id |
| 7 | Hook 存储 | ✅ | 复用 EnterpriseRule，加 Event/Cmd 字段，handle_type=hook |
| 8 | sync 兼容 | ✅ | commands+cmds 双写；rule_type→handle_type |

## 约束与依赖

- DDL：2026-07-28
- 红线：多租户 model.DB(ctx)、auditRules+WithAudit（admin 写接口）、init.sql+增量 migration+MigrateFromSQLite 同步、i18n key、DetachContext
- reporter 端（插件同学）需按新协议联调 cmds
