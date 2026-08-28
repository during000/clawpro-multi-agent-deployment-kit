---
name: purchase-guide-writer
description: 根据购买指南大纲和官网文档规范生成购买指南初稿。
---

# purchase-guide-writer

Source: ClawPro enterprise Skill library.
Version: v0.1.0
Owner: 倩茹

## Instructions

生成购买指南时，先读取腾讯云官网文档规范和购买指南大纲。不得编造价格、购买入口、权限、地域、限制条件。未知信息必须进入待确认信息清单。

## Required Specs

- 文档写作规范 (doc-writing v1.0.0): 对外文档必须使用准确、克制、可验证的表达。不得编造未提供的价格、入口、权限、地域、限制条件。输出必须包含未确认信息清单。
- 腾讯云官网文档规范 (tencent-cloud-doc-style v2.0.0): 标题不超过 30 个字符，不以动词开头，不使用句号。中文和英文之间需要空格。数字和单位之间不加空格。面向用户时使用“您”。描述性句子以句号结束。
- 购买指南大纲 (purchase-guide-outline v0.1.0): 购买指南包含：功能概述、适用场景、购买前准备、购买步骤、后续操作、常见问题。缺失信息必须列入待确认项。

## Required MCP

- wecom-doc-reader (wecom-doc-reader v1.0.0): 读取腾讯文档、KM、Sheet 的采集内容。

## Acceptance

- Read the required specs before producing output.
- Do not invent unknown price, purchase entry, permission, region, or quota information.
- List unresolved facts under a clear pending-confirmation section.
