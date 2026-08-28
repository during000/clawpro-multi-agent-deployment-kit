# MR：引入 brennali 用户端最新设计，管理端保持 2026 版本

- **源分支**：`feature/design-brennali2`
- **目标分支**：`feature/design-refresh-2026`
- **Commit**：`7c6ef487`
- **改动规模**：29 files changed, +408 / -248

## 背景

将 brennali 个人分支上完成的「用户端（tenant）最新设计走查成果」合入设计主干 `feature/design-refresh-2026`，同时保持管理端（admin）继续沿用 2026 版本，不在本次 MR 中调整。

## 主要改动

### 1. 用户端组件视觉与交互更新（client/src/components）
- `agent/AgentCard.tsx`：Agent 卡片底部操作栏改用 `mt-auto` 固定贴底，避免不同卡片高度导致的对齐错位。
- `agent/QuickStartGuide.tsx`：快速开始引导按新 Figma 重写结构与样式（-88 行精简）。
- `agent/StatusBadge.tsx`：状态徽标整体重写（248 行 ↔ 248 行），统一新设计稿的色板与排版。
- `agent/ViewModeSegmented.tsx`：视图切换分段控件细节微调。
- `topnav/CenterTabs.tsx`、`topnav/HelpPanel.tsx`、`topnav/NotificationPanel.tsx`：顶部导航中心 Tab、帮助面板、消息面板的文案/样式微调。
- `TenantLayout.tsx`：用户端整体布局微调。

### 2. 用户端页面更新（client/src/pages/tenant）
- `AgentChat.tsx`：Agent 对话页样式与结构对齐新设计。
- `MyOpenClaw.tsx`：「我的 OpenClaw」页面细节微调。

### 3. 全局样式
- `client/src/index.css`：新增 +183 行全局样式（含字体、设计 token 等）。
- `client/src/pages/landing/landing.css`：Landing 页样式微调。

### 4. 静态资源
- 新增 quickstart 引导图：`close-hover.png`、`close-normal.png`、`dynamic-bg.png`、`steps-bg.png`，并更新 `step-1/2/3.svg`。
- 新增腾讯云数字字体 `TCloudNumber.ttf`、`TCloudNumber-Bold.ttf`。
- 新增图标：`Agent总数.png`、`产品动态.svg`、`其他.png`、`定位箭头.svg`、`已关机.png`、`运行中数量.png`。
- 新增/替换租户背景：`tenant_bg.jpg`（新增）、`tenant_bg.png`（替换为高清版）。

## 影响范围

- ✅ 用户端（tenant）UI：视觉/交互按 brennali 走查结果对齐。
- ⛔ 管理端（admin）：本次未改动，保持 `feature/design-refresh-2026` 现状。

## 自测点

- [ ] Agent 列表卡片（不同状态、不同高度）底部操作栏对齐一致。
- [ ] StatusBadge 各状态色与字号符合最新设计稿。
- [ ] 顶部导航中心 Tab、帮助面板、消息面板正常打开/切换。
- [ ] 快速开始引导（QuickStartGuide）三步动效与图片正常加载。
- [ ] AgentChat 对话页布局、滚动、输入区无回归。
- [ ] Landing 页与 tenant 背景图加载正常，无破图/锯齿。

## 备注

本次为**单 commit**（已 squash），如需 cherry-pick 或回滚，可直接基于 `7c6ef487`。
