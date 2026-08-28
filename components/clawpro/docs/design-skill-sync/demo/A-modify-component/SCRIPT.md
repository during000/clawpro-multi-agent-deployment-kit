# 场景 A · 修改现有组件（给 StatusTag 加 `banner` 形态 + 配套 token）

> 日常触发语：「给 StatusTag 加个新形态 banner，再配一个 banner 底色 token。」
> 亮点：P1 **双通道**（代码枚举 + CSS 变量）同时命中 → `sync-tokens` 消 CSS 差集 → 三层对齐 → 退出码 **1→0** 闭环。

## 1. 改代码（真相源，只动代码不碰文档）
- `client/src/components/ui/status-tag.tsx` 第 167 行 union 加 `"banner"`：
  ```
  type StatusTagMode = "text" | "dot" | "fill" | "soft" | "banner";
  ```
- `client/src/index.css` 的 `:root{}` 加一行新 token：
  ```
  --status-tag-banner-bg: #EEF2FF;
  ```

## 1.5 人工触发（真实入口）📸 `A0-trigger.png`
改完代码后，在对话框对 AI 说口令：**`对齐 skill 规范`**。
AI 读取 `docs/design-skill-sync/SOP.md` A 段（增量流程）→ 按 Stage0/1/2 执行下述流水线。截图框住「口令 + AI 回复开头读 SOP」。

## 2. 触发增量差集
```
node .codebuddy/skills/clawpro-portable-design-skill/scripts/diff-code-vs-docs.mjs --since HEAD ; echo "exit=$?"
```
**实测证据（exit=1）**——一次命中三条候选：
- `CODE_ENUM_NOT_IN_DOC` · status-tag union `StatusTagMode = "banner"`（本次新增）
- `CODE_ENUM_NOT_IN_DOC` · status-tag union `StatusTagPreset = "role-user"`（**顺带揪出的历史漏声明**——碰到该组件就把它所有枚举都体检了）
- `CSS_VAR_NOT_IN_DOC` · `--status-tag-banner-bg`

## 3. 消 CSS 差集（sync-tokens）
```
node .codebuddy/skills/clawpro-portable-design-skill/scripts/sync-tokens.mjs
```
**实测**：从 index.css 抽取 244 个 token 写入 `tokens/design-tokens.json`，新 token `status-tag-banner-bg` 被纳入文档池。

## 4. 三层对齐（reconcile）
- **Stage1（Tier2 主戏）**：`component-specs/status-tag.md` §3.4 形态表补一行 `banner`，并补齐 `role-admin`/`role-user` 字面量。
- **Stage0（Tier1）**：口播说明沉淀落点（SKILL-GLOBAL-COMPONENTS.md 无独立 StatusTag 章，本次不实际编辑，按 Q3=b）。
- **Stage2（Tier3）**：
  ```
  node .codebuddy/skills/clawpro-walkthrough/scripts/walkthrough.mjs refresh-fixtures
  ```
  **实测**：6 个 fixtures 全部重抽成功，exit 0。

## 5. 复跑至闭环
```
node .codebuddy/skills/clawpro-portable-design-skill/scripts/diff-code-vs-docs.mjs --since HEAD ; echo "exit=$?"
```
**实测证据**：`✅ 无 P1 差集` + **exit=0**（与第 2 步的 exit=1 形成 1→0 对照）。

## 6. 亮点话术（PPT）
- **以代码为准**：这正是历史上 status-tag「文档声称只剩 text」事故的防线——文档漏声明会被机械揪出。
- **双通道**：枚举漂移 + token 漂移一次抓齐。
- **彻底性**：碰到组件就体检它全部枚举，连历史漏的 `role-user` 一并补齐。

## 7. 录制取帧（截图存本文件夹）
| 文件名 | 截什么 |
|---|---|
| `A2-diff-exit1.png` | 「对不上清单」双通道三条 finding + `exit=1` |
| `A3-sync-tokens.png` | sync-tokens 抽取 244 token、写入 design-tokens.json |
| `A6-diff-exit0.png` | `✅ 无 P1 差集` + `exit=0` |
| `A-recording.mp4` | 本场景全程录屏 |
