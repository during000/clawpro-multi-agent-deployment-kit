# PRD：本地 Agent 资源管理 & 接入本地Agent

> 版本：v1.2  
> 日期：2026-07-03  
> 分支：feature/tenant-openclaw-detail-basic  
> 涉及页面：OpenClaw 详情页（`/openclaw/:id`）、我的 Agent 页（`/my-openclaw`）

---

## 一、背景

用户通过 ClawPro 接入本地 AI 编码工具（CodeBuddy / WorkBuddy / Claude Code）后，需要在用户端查看和管理本地 Agent 的资源配置。当前需要：

1. 在 OpenClaw 详情页展示「用户级资源」和「项目级资源」，让用户了解组织和 workspace 下发的技能与规范
2. 在「我的 Agent」页提供「接入本地Agent」入口，引导用户完成本地工具的企业接入

---

## 二、功能概览

### 2.1 OpenClaw 详情页 - 资源管理模块

路由：`/openclaw/:id`（本地 Agent 类型）  
组件：`OpenClawDetailGuide.tsx` → `LocalAgentSettingsPanel`

页面分为两个区块：

| 区块 | 作用域 | 说明 |
|------|--------|------|
| 用户级资源 | 当前用户本地 Agent 所有工作空间（workspace） | 跟随用户账号，跨 workspace 生效 |
| 项目级资源 | 单个 workspace | 每个 workspace 独立绑定 |

### 2.2 我的 Agent 页 - 接入本地Agent弹窗

路由：`/my-openclaw`  
组件：`MyOpenClaw.tsx` → 接入本地Agent Dialog

用户选择操作系统后，复制通用安装 Prompt 粘贴到本地 AI 工具中执行，完成 ClawPro 插件安装。Prompt 不区分 Agent 类型，适用于所有已支持的本地 AI 工具。

---

## 三、详细需求 - 用户级资源

### 3.1 区块结构

```
┌─────────────────────────────────────────────────┐
│ 用户级资源                                       │
│ 作用于当前用户本地 Agent 所有工作空间（workspace）│
│                                                 │
│ 组织：CVM团队（主组织）  ✏️                       │
│                                                 │
│ 技能（6）                                        │
│ [code-interpreter 1.2.0] [image-recognition]   │
│ [text-to-speech] [pdf-parser] [excel-reader]   │
│                                                 │
│ 规范（6）❓                                      │
│ [CODEBUDDY.md CodeBuddy] [分支规范 CodeBuddy]   │
│ [代码评审 CodeBuddy] [WORKBUDDY.md WorkBuddy]   │
└─────────────────────────────────────────────────┘
```

### 3.2 组织选择器

**展示规则：**
- 默认展示用户所属的「主组织」，标注"（主组织）"
- 组织名称右侧有编辑图标（铅笔），点击进入编辑态
- 编辑态下组织名称变为 Select 下拉，可切换到其他组织（单选）
- 下拉选项中主组织标注"（主组织）"
- 编辑态有取消按钮（X 图标），点击恢复原值

**组织切换确认弹窗：**

选择新组织后，不立即生效，先弹出确认对话框：

> **切换用户级组织**
>
> 切换至「新组织名」后：
> - 新建工作空间、以及已有工作空间中新建会话，将使用新组织下发的技能和规范
> - 已有会话的工具配置不会改变
>
> [取消] [确认切换]

- 用户确认 → 应用切换，Toast 提示成功
- 用户取消 → 恢复原组织，不切换
- 选择同一组织不弹确认（无变化）

**无组织功能时：**
- 组织显示"—"
- 编辑按钮 disabled（opacity 40%，cursor not-allowed）

**切换组织后的行为规则（后端逻辑，前端仅展示确认弹窗和 Toast）：**

| 场景 | 行为 |
|------|------|
| 切换用户级组织 | 新 session 补充新组织下发的 skill，不删除已安装的 skill |
| 切换前已有的 session | 工具配置不变（不动态更新） |
| 用户组织信息变更 | 已有 workspace（项目级）的 skill 等工具不更新 |
| 用户手动选了非主组织 | 主组织变更后，若当前组织仍有效，用户级组织不自动切换 |
| 当前组织权限失效 | 用户级组织自动回退到主组织（优先选择主组织） |

### 3.3 技能展示

**数据模型：**

```typescript
type LocalAgentSkill = {
  id: string;
  name: string;        // 技能名称，如 "code-interpreter"
  version: string;     // 版本号，如 "1.2.0"
  description?: string; // 技能描述
  distributeStatus: "distributing" | "distributed" | "failed";
};
```

**展示方式：**
- Chip 标签式布局（flex-wrap），高信息密度
- 每个 Chip 显示：技能名称 + 版本号（灰色小字）
- hover 时 Chip 边框变为主题色，鼠标显示 tooltip（技能描述）
- 区块标题显示"技能（N）"，N 为数量
- 空数据显示"暂无技能"

**下发状态展示：**

| 状态 | 含义 | Chip 展示 | hover 说明 |
|------|------|-----------|-----------|
| `distributing` | 下发中 | spinner + "下发中"（蓝色） | 管理员已下发，等待本地 Agent 完成安装 |
| `distributed` | 已下发 | 正常 Chip（无额外标识） | — |
| `failed` | 下发失败 | 红色边框 + 警告图标 + "下发失败" | 本地 Agent 安装失败，请检查插件状态后重试 |

**版本更新场景：**
- skill 处于"下发中"时管理员更新版本，后端取消旧下发、发起新版本下发
- 前端始终展示最新的"下发中"状态 + 最新版本号

**不支持的操作：**
- 不支持用户级技能安装/卸载（只读展示）
- 不支持搜索/筛选

### 3.4 规范展示

**数据模型：**

```typescript
type SpecItem = {
  id: string;
  name: string;        // 显示名称，如 "CODEBUDDY.md" 或 "分支规范"
  path: string;        // 文件路径，如 ".codebuddy/CODEBUDDY.md"
  ideType: "codebuddy" | "workbuddy";
  type: "system-prompt" | "rule";
  status: "已生效" | "未生效";
  updatedAt: string;   // 更新时间
  distributeStatus: "distributing" | "distributed" | "failed";
};
```

**展示方式：**
- 与技能相同的 Chip 布局
- 每个 Chip 显示：规范名称 + IDE 标签（CodeBuddy / WorkBuddy）
- 下发状态展示方式与技能一致（distributing / distributed / failed）
- 区块标题"规范（N）"右侧有 ❓ 图标

**规范说明 Tooltip（❓ hover 展示）：**

> **CodeBuddy**
> • System Prompt：`.codebuddy/CODEBUDDY.md`（或项目根目录 `CODEBUDDY.md`），初始化时自动创建，帮助 AI 快速了解项目上下文
> • Rules：`.codebuddy/rules/{slug}.md`，项目级编码规范，受版本控制管理，可团队共享
>
> **WorkBuddy**
> • 在 workbuddy 目录下新建 rules 文件夹存放规范文件（WorkBuddy 本身不内置规则系统）

**规范文件来源说明：**

| IDE 类型 | System Prompt | Rules |
|----------|---------------|-------|
| CodeBuddy | `.codebuddy/CODEBUDDY.md` 或项目根目录 `CODEBUDDY.md` | `.codebuddy/rules/{slug}.md` |
| WorkBuddy | `.workbuddy/WORKBUDDY.md` | `.workbuddy/rules/{slug}.md`（workbuddy 目录下新建 rules 文件夹） |

---

## 四、详细需求 - 项目级资源

### 4.1 区块结构

```
┌──────────────────────────────────────────────────┐
│ 项目级资源                                        │
│ 管理各工作空间（workspace）绑定的技能和规范，      │
│ 点击展开查看详情。                                 │
│                                                  │
│ ▸ clawpro项目      组织：CVM团队                  │
│   /Users/petzhou/CodeBuddy/clawpro  CodeBuddy     │
│   技能 6  规范 4                                   │
│                                                  │
│ ▾ 接管本地虾-用户端  组织：CVM团队                 │
│   /Users/petzhou/CodeBuddy/接管本地虾-用户端       │
│   技能 4  规范 3                                   │
│ ┌──────────────────────────────────────────────┐ │
│ │ 技能（4）                                      │ │
│ │ [code-explorer 1.0.0] [refactor-helper 0.8.2]│ │
│ │ [test-generator 1.1.0 下发中] [doc-writer]    │ │
│ │                                               │ │
│ │ 规范（3）❓                                    │ │
│ │ [CODEBUDDY.md CodeBuddy] [前端编码规范]       │ │
│ │ [组件使用约束 CodeBuddy]                       │ │
│ └──────────────────────────────────────────────┘ │
│                                                  │
│ ▸ 异构项目  组织：异构团队（已被移除组织）          │
│   /Users/petzhou/CodeBuddy/hetero  WorkBuddy      │
│   技能 4  规范 3                                   │
└──────────────────────────────────────────────────┘
```

### 4.2 Workspace 列表

**数据模型：**

```typescript
type Workspace = {
  id: string;
  name: string;           // workspace 名称
  path: string;           // 本地路径
  groupId: string;        // 所属组织 ID
  groupName: string;      // 所属组织名称
  groupActive: boolean;   // 用户是否仍在该组织内
  ideType: "codebuddy" | "workbuddy";
  skills: LocalAgentSkill[];
  specs: SpecItem[];
};
```

**列表行展示：**
- 展开/收起箭头（ChevronDown，收起时旋转 -90°）
- Workspace 名称（font-medium）
- 组织标签（GroupTag 组件）：
  - 正常：蓝色背景，显示"组织：xxx"
  - 已失效：橙色背景 + 警告图标，显示"组织：xxx（已被移除组织）"
- 路径（等宽字体，灰色）
- IDE 类型标签（CodeBuddy / WorkBuddy）
- 右侧汇总：技能数量、规范数量

**展开行内容：**
- 灰色背景区域，上下排列"技能"和"规范"两个区块
- 技能/规范使用与用户级相同的 Chip 布局（含下发状态）
- 规范标题同样带 ❓ 说明 tooltip

**交互规则：**
- 点击 workspace 行展开/收起（accordion 模式，不跳转页面）
- 多个 workspace 可同时展开（非互斥）
- 不支持 workspace 的组织修改（只读）
- 不支持技能安装/卸载（只读展示）

### 4.3 组织失效场景

当用户被移出某个组织后，该组织下的 workspace 仍展示在列表中，但：
- 组织标签变为橙色警告样式："组织：xxx（已被移除组织）"
- Workspace 的技能和规范仍可查看（已下发的资源不会被回收）
- 不支持在该 workspace 上做任何修改操作

---

## 五、详细需求 - 接入本地Agent弹窗

### 5.1 入口

「我的 Agent」页（`/my-openclaw`）顶部"接入本地Agent"按钮，点击打开弹窗。

### 5.2 弹窗结构

```
┌──────────────────────────────────────────────────────┐
│ 接入本地Agent                                         │
│ 将本地 Agent 接入企业管理后，组织下发的技能和规范      │
│ 会自动同步到你的 AI 工具（如 CodeBuddy / WorkBuddy）   │
│                                                      │
│ ℹ️ 复制安装 Prompt，粘贴到本地 AI 工具的对话框中       │
│    执行，即可完成 ClawPro 插件安装和接入。             │
│                                                      │
│ 操作系统                                              │
│ ┌──────────────┐  ┌──────────────┐                  │
│ │ 🍎 macOS     │  │ 🖥️ Windows   │                  │
│ │ 适用于...    │  │ 适用于...    │                  │
│ └──────────────┘  └──────────────┘                  │
│                                                      │
│ 支持的 Agent                                          │
│ [💻 CodeBuddy] [💻 WorkBuddy] [💻 Claude Code]       │
│                                                      │
│                    [关闭]  [复制 Prompt]              │
└──────────────────────────────────────────────────────┘
```

### 5.3 交互规则

**选择操作系统：**
- 两个选项卡片（macOS / Windows），单选
- 选中后边框高亮 + 绿色勾选图标
- 选项包含：图标、名称、描述

**支持的 Agent（只读展示）：**
- 以标签形式展示当前支持的 Agent：CodeBuddy / WorkBuddy / Claude Code
- 不可选择，仅告知用户支持哪些本地 AI 工具
- Prompt 通用，不区分 Agent 类型

**复制 Prompt 按钮：**
- 选择操作系统后按钮可用
- 未选操作系统时按钮 disabled，hover 显示 tooltip（即时响应，无延迟）："请先选择操作系统"
- 点击后调用 `handleCopyLocalClientInstallPrompt` 复制到剪贴板

**Prompt 内容：**
根据选择的操作系统生成通用安装指令文本，不区分 Agent 类型，包含：
- 操作系统标识
- ClawPro 插件安装步骤（适用于所有支持的本地 AI 工具）

---

## 六、边界与异常处理

### 6.1 用户级资源

| 场景 | 处理 |
|------|------|
| 无组织功能 | 组织显示"—"，编辑按钮禁用 |
| 组织下无技能 | 显示"暂无技能" |
| 组织下无规范 | 显示"暂无规范" |
| 用户级组织切换 | 弹确认弹窗，确认后 Toast 提示，不刷新已有 session 的工具配置 |
| 组织权限失效 | 自动回退到主组织 |
| 技能/规范下发中 | Chip 显示 spinner + "下发中"（蓝色），hover 显示说明 |
| 技能/规范下发失败 | Chip 红色边框 + "下发失败"，hover 显示说明 |
| 下发中版本更新 | 展示最新版本 + "下发中"状态，旧版本丢弃 |

### 6.2 项目级资源

| 场景 | 处理 |
|------|------|
| 无 workspace | 显示空态 |
| Workspace 组织失效 | 标签变为橙色"已被移除组织"，仍可展开查看 |
| Workspace 无技能 | 展开区显示"暂无技能" |
| Workspace 无规范 | 展开区显示"暂无规范" |

### 6.3 接入本地Agent

| 场景 | 处理 |
|------|------|
| 未选操作系统 | "复制 Prompt"按钮禁用，hover 提示"请先选择操作系统" |
| 复制失败 | Toast 错误提示 |

---

## 七、不涉及的功能（本期不做）

- ❌ 用户级/项目级技能安装与卸载
- ❌ Workspace 组织修改
- ❌ 插件、MCP、知识库展示（仅保留技能和规范）
- ❌ 规范文件内容编辑
- ❌ Workspace 新增/删除

---

## 八、涉及文件

| 文件 | 改动内容 |
|------|---------|
| `client/src/pages/tenant/OpenClawDetailGuide.tsx` | 用户级资源 + 项目级资源模块重构，下发状态展示，组织切换确认弹窗 |
| `client/src/pages/tenant/MyOpenClaw.tsx` | 接入本地Agent弹窗：单页面布局、操作系统选择、支持Agent只读展示、通用Prompt、禁用按钮 tooltip |
| `client/src/App.tsx` | 未修改（路由不变） |
| `client/src/components/AdminLayout.tsx` | 未修改 |
