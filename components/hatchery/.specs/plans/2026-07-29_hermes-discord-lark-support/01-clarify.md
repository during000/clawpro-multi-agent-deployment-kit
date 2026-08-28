# 01. Clarify — 需求澄清

> 事后补录。本任务的代码实现先于 SOP 文档，此处根据实际改动补全。

---

## 背景

Hermes agent 类型已支持 feishu / wecom / weixin / qqbot / ddingtalk / slack / msteams 通道。两个独立需求分别要求新增 Discord 和 Lark（飞书国际版）通道支持，分别在 `feat/hermes-discord-support` 和 `feat/hermes-lark-support` 分支上开发，需要合并到一个分支。

## 目标

- [x] Hermes agent 类型支持 Discord 通道配置
- [x] Hermes agent 类型支持 Lark 通道配置
- [x] 配置/删除通道脚本（set_channel_hermes.sh / del_channel_hermes.sh）支持 discord 和 lark
- [x] openclaw 的 set_channel.sh 同步更新 discord 的 user_id 参数
- [x] API 文档更新

## 范围

| 包含 | 不包含 |
|------|--------|
| Hermes 通道白名单新增 discord / lark | openclaw agent 类型的通道白名单（已支持 discord） |
| set_channel_hermes.sh 新增 discord / lark case 分支 | 其他通道的逻辑变更 |
| del_channel_hermes.sh 新增 discord / lark 删除逻辑 | 数据库 schema 变更 |
| set_channel.sh discord 分支新增 user_id 参数 | 前端代码变更 |
| model/ai_channel.go discord 新增 user_id 字段 | |
| 单元测试更新 | |

## 待确认问题

| # | 问题 | 状态 | 结论 |
|---|------|------|------|
| 1 | Discord 是否支持通配符匹配用户？ | 已确认 | 不支持，必须指定 user_id |
| 2 | Lark 是否需要独立配置字段？ | 已确认 | 复用 feishu 字段，FEISHU_DOMAIN 设为 lark |

## 约束与依赖

- 基于分支 `Release/2026_07_28`（commit 31f4f314）
- 两个 feature 分支需 rebase 到同一 base 后再合并
