---
task_id: hermes-agent-upgrade
stage: 01-clarify
author: AI（反向推导）
date: 2026-07-20
run_mode: manual
---

# Clarify（反向补建）

> 本文档基于分支 `feature/hermes_agent_upgrade` 相对 `master` 的 38 个提交、代码现状反推整理，非开发前的原始澄清记录。

## 背景

Hatchery 已支持对 OpenClaw 类型实例的"一键升级"（备份用户数据 → 上传 SMH → 重装系统 → 恢复数据）。随着 Hermes Agent 类型实例数量增加，需要把同样的一键升级能力开放给 Hermes 实例，而不是让用户手动重装。

## 目标

1. Hermes 类型实例可以通过 `/openclaw/upgrade`、`/openclaw/upgrade/retry` 接口触发与 OpenClaw 一致的备份→重装→恢复流程。
2. 升级过程中正确处理 Hermes 官方镜像运行用户可能随版本变化的问题（如 `agentuser` → `ubuntu`），避免恢复脚本以错误用户身份运行导致权限问题。
3. 不破坏现有 OpenClaw 升级链路的行为和兼容性。

## 约束 / 待确认问题（反推）

1. **RuntimeUser 假设失效**：原实现假设升级前后运行用户不变，直接复用旧值。Hermes 官方镜像存在跨版本变更运行用户的情况，重装后必须重新探测。
2. **备份/恢复策略差异**：OpenClaw 使用增量备份脚本，Hermes 因目录结构不同（`~/.hermes` 等）需要专属的 backup/restore 脚本。
3. **插件/Skill 数据在重装后如何保留**：需要一份"重装后必须保留的路径清单"（后落地为 `config/agent_plugin_preserve_paths.json` + 脚本内 `PRESERVE_PATHS`），本次 Review 发现这份清单在 Go 侧和 Shell 侧存在双写风险（见 07-review.md）。
4. **是否需要为 Hermes 补充升级后置动作**：OpenClaw 升级后有 5 项 post-hook（同步网关端口、修复插件 node_modules、跑兼容脚本、清理临时文件、审批设备），Hermes 是否需要对应动作、需要哪些 —— 最终采用表驱动 `upgradePostHookTable[runtimeType]`，Hermes 走精简版（二次 ready 探测 + 通道兼容 + 清理）。

## 结论

按"能力位化 + 表驱动分派"的方向实现（见 02-plan.md），已完成开发并合并到当前分支，等待走完补建的 SOP 收尾流程后提 MR。
