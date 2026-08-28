# ClawPro 多人协作分支开发规范

> 本文是面向 CodeBuddy、ClawPro 项目协作者和人工 Git 操作者的完整执行规范。CodeBuddy 日常开发由轻量常驻 Rule 和分支快速索引执行，无需每轮读取全文；遇到首次配置、异常处理、MR 模板或规则争议时再读取本文。

| 项目 | 内容 |
| --- | --- |
| 规范版本 | v1.14（未提交修改基线版） |
| 状态 | 生效中 |
| 适用仓库 | `git@git.woa.com:cvm-openclaw/openclaw-enterprise.git` |
| 默认主干 | `main` |
| 适用工具 | CodeBuddy、ClawPro、本地 Git |
| 主干及公共基建负责人 | `qitinghuang` |
| 设计规范接口人 | `addietang`、`miekoyychen` |
| 人工职责数据源 | `.codebuddy/docs/clawpro-branch-ownership.tsv` |
| 实时分支状态来源 | 每个独立新需求通过脚本查询工蜂 `origin` |
| 工蜂实时索引 | `<git-dir>/clawpro-branch-map.live.tsv`（自动生成，不提交） |
| 同步命令 | `python3 .codebuddy/scripts/sync-clawpro-branches.py --origin origin` |
| 路径约定 | 使用 CodeBuddy 的 `CODEBUDDY_PROJECT_DIR` 定位仓库，不依赖个人电脑目录 |

## 1. 规范级别

本文使用以下关键词：

- **必须**：不满足时不得提交或合入。
- **禁止**：任何情况下不得自行执行；获得对应负责人明确授权后，按授权范围操作。
- **建议**：默认采用；如需偏离，应在 MR 中说明原因。

## 2. 核心原则

1. **禁止直接在 `main` 开发或直接向 `main` 推送代码。**所有需求必须在对应 `feature/*` 分支完成，并通过 MR 合入 `main`。
2. **用户只需描述需求，不需要预先选择分支。**CodeBuddy 必须先同步工蜂实时分支，再用分支登记表匹配职责并在安全时切换；告知匹配结果后等待用户确认，确认前不得开始开发。
3. **一个页面或一个职责边界对应一个特性分支。**只修改当前分支负责的页面和文件。
4. **一个分支建议对应一个 CodeBuddy 项目。**这里的“项目”是独立的开发上下文，不等同于必须使用 Git `worktree`；本规范不要求协作者自行配置 `worktree`。
5. **禁止自行修改公共基建。**未经 `qitinghuang` 明确确认，不得修改 `App.tsx`、`AdminLayout.tsx`、用户端或管控端导航，不得新增子页面或子 Tab。
6. **每次开工和提交前都要同步工蜂。**执行 `git fetch origin --prune`，先处理当前远端 feature 分支更新，再将当前分支 rebase 到 `origin/main`。
7. **设计实现必须读取仓库内现行规范。**以 `main` 中的 `.codebuddy/rules/` 和 `.codebuddy/skills/` 为准，不使用个人保存的旧版设计包覆盖仓库版本。
8. **禁止依赖 AI 对话记录恢复代码。**代码恢复必须依赖 Git 提交、`reflog`、备份分支或明确保存的补丁。
9. **任何敏感信息不得进入对话、日志或仓库。**禁止上传 SSH 私钥、Token、Cookie、账号密码和生产凭据。

## 3. 角色与职责

### 3.1 主干与公共基建负责人

- 维护 `main` 主干。
- 维护路由、导航和公共基建，包括但不限于 `App.tsx`、`AdminLayout.tsx`。
- 确认新增页面、子页面、子 Tab 和导航调整的归属。
- 协调跨分支冲突和最终合入。

当前接口人：`qitinghuang`。

### 3.2 特性分支开发者

- 只需描述需求，由 CodeBuddy 从分支登记表匹配 `feature/*` 分支；核对并确认 CodeBuddy 汇报的分支和负责范围。
- 仅修改当前页面或模块范围内的代码。
- 在开工前、提交前同步 `origin/main`。
- 完成本地检查、预览、MR 信息和问题说明。

操作者：各产品经理、前端开发者及对应模块协作者。

### 3.3 全局样式与公共组件负责人

- 维护 ClawPro 全局样式、公共组件、图标库和设计规范 Skill。
- 处理标准组件缺失、Token 缺失和设计规范冲突。

当前接口人：`addietang`、`miekoyychen`。

## 4. 分支职责与工蜂实时索引

分支数据分为两层：

- `.codebuddy/docs/clawpro-branch-ownership.tsv` 是随仓库提交的人工职责数据源，维护“需求 → 页面/模块 → 负责分支”。页面归属不能仅凭分支名可靠推断，因此新增或变更职责仍需人工确认并通过 MR 更新。
- `<git-dir>/clawpro-branch-map.live.tsv` 是每个独立新需求自动生成的工蜂实时索引。同步脚本读取人工职责数据并调用 `git ls-remote --heads origin`，合并生成当前全部远端 `feature/*`、远端 HEAD、登记状态和存在状态。该文件位于 Git 内部目录，不进入提交，也不会弄脏工作区。

下表是供人阅读的职责说明；“远端状态”仅为原始手册的 2026-07-17 历史快照，不作为当前分支是否存在的依据。实际路由必须以本轮生成的实时 TSV 为准。远端新增但未登记的分支会自动进入实时 TSV，标记为 `unregistered`；已登记但远端消失的分支会保留并标记为 `missing`，不得据此自行重建。

| 序号 | 分支 | 模块 | 页面或 Tab | 团队 | 远端状态 |
| ---: | --- | --- | --- | --- | --- |
| 1 | `feature/landing-page` | 用户端 | 落地页 | 计算 | 已存在 |
| 2 | `feature/tenant-my-openclaw` | 用户端 | 我的 Agent | 计算 | 已存在 |
| 3 | `feature/tenant-openclaw-detail-basic` | 用户端 | Agent 详细配置：基础配置 Tab；模型、通道、技能；页面 Header | 计算 | 已存在 |
| 4 | `feature/tenant-openclaw-detail-tools` | 用户端 | Agent 详细配置：工具管理 Tab | 计算 | **待确认** |
| 5 | `feature/tenant-openclaw-detail-memory` | 用户端 | Agent 详细配置：记忆管理 Tab | 数据库 | 已存在 |
| 6 | `feature/tenant-openclaw-detail-files` | 用户端 | Agent 详细配置：文件管理 Tab | 云盘 | **待确认** |
| 7 | `feature/tenant-openclaw-detail-backup` | 用户端 | Agent 详细配置：数据备份 Tab | 计算 | 已存在 |
| 8 | `feature/tenant-openclaw-detail-doctor` | 用户端 | Agent 详细配置：龙虾医生 Tab | 计算 | 已存在 |
| 9 | `feature/tenant-skill-square` | 用户端 | 技能广场 | 计算 | 已存在 |
| 10 | `feature/tenant-model-quota` | 用户端 | 模型配额 | 计算 | 已存在 |
| 11 | `feature/tenant-help-docs` | 用户端 | 帮助文档 | 计算 | **待确认** |
| 12 | `feature/tenant-reset-password` | 用户端 | 重置密码 | 计算 | **待确认** |
| 13 | `feature/admin-basic-info` | 管控端 | 基础信息配置 | 计算 | 已存在 |
| 14 | `feature/admin-api-docs` | 管控端 | API 文档；入口在基础信息配置，但作为独立页面 | 计算 | 已存在 |
| 15 | `feature/admin-platform-policy` | 管控端 | 平台策略：用户配额、模型配额、功能权限开关 | 计算 | 已存在 |
| 16 | `feature/admin-member-management` | 管控端 | 用户管理 | 计算 | 已存在 |
| 17 | `feature/admin-model-config` | 管控端 | 模型配置 | 计算 | 已存在 |
| 18 | `feature/admin-channel-config` | 管控端 | 通道配置 | 计算 | 已存在 |
| 19 | `feature/admin-memory-management` | 管控端 | 记忆管理 | 数据库 | 已存在 |
| 20 | `feature/admin-file-management` | 管控端 | 文件管理 | 云盘 | 已存在 |
| 21 | `feature/admin-knowledge-management` | 管控端 | 知识管理 | 计算 | 已存在 |
| 22 | `feature/admin-resource-management` | 管控端 | 资源管理 | 计算 | 已存在 |
| 23 | `feature/admin-image-management` | 管控端 | Agent 类型，原镜像管理 | 计算 | 已存在 |
| 24 | `feature/admin-security-group` | 管控端 | 网络管理：安全组、私有网络和子网、公网、敬请期待，共 4 个 Tab | 计算 | 已存在 |
| 25 | `feature/admin-cloud-dev` | 管控端 | 云开发管理 | 云开发 | 已存在 |
| 26 | `feature/admin-security-management` | 管控端 | AI Agent 安全 | 安全 | 已存在 |
| 27 | `feature/admin-audit-log` | 管控端 | 操作记录、审计日志 | 计算 | **待确认** |
| 28 | `feature/admin-skill-preset` | 管控端·技能配置 | 技能初始包 Tab | 计算 | 已存在 |
| 29 | `feature/admin-skill-roles` | 管控端·技能配置 | 角色设定 Tab | 计算 | 已存在 |
| 30 | `feature/admin-skill-source` | 管控端·技能配置 | 技能安装来源 Tab | 计算 | **待确认** |
| 31 | `feature/admin-tool-public` | 管控端·Agent 工具库 | 公共技能库 Tab | 计算 | 已存在 |
| 32 | `feature/admin-tool-enterprise` | 管控端·Agent 工具库 | 企业技能库 Tab | 计算 | 已存在 |
| 33 | `feature/admin-tool-plugin` | 管控端·Agent 工具库 | 企业插件库 Tab | 计算 | 已存在 |
| 34 | `feature/admin-tool-mcp` | 管控端·Agent 工具库 | 企业 MCP 库 Tab | 计算 | 已存在 |
| 35 | `feature/admin-knowledge-base` | 管控端·Agent 工具库 | 企业知识库 Tab | 计算 | 已存在 |
| 36 | `feature/admin-standards-library` | 管控端·Agent 工具库 | 企业规范库 Tab | 计算 | 已存在 |
| 37 | `feature/admin-settings-library` | 管控端·Agent 工具库 | 企业设定库 Tab | 计算 | 已存在 |
| 38 | `feature/admin-asset-management` | 管控端 | 资产管理 | 计算 | 已存在 |
| 39 | `feature/credential-management` | 管控端 | 凭据管理 | 计算 | 已存在 |
| 40 | `feature/admin-cls-monitoring` | 管控端·联动 | 会话管理、会话详情、运维观测、Tokens 监控；强联动 | CLS、计算 | 已存在 |
| 41 | `feature/admin-openclaw-monitor` | 管控端 | Agent 列表 | 计算 | 已存在 |
| 42 | `feature/admin-agent-migration` | 管控端 | 智能体迁移；入口在 Agent 列表，但作为独立页面 | 计算 | 已存在 |
| 43 | `feature/design-integration-2026` | 全局 | 设计侧刷新样式专用 | 设计 | 已存在 |

### 4.1 需求自动匹配分支

用户可以只发送自然语言需求，不必自行查表或写分支名。CodeBuddy 按以下顺序处理：

1. 在 origin 正确的仓库根目录执行 `python3 .codebuddy/scripts/sync-clawpro-branches.py --origin origin`。每个独立新需求执行一次；脚本输出的 `LIVE_TSV=<path>` 是本轮唯一实时分支索引。同一需求的补充调整、修复、验证、预览、提交、推送或 MR 属于连续任务，不重复路由。
2. 从需求中识别端侧、页面、Tab、模块、功能名称和可能涉及的文件；先检索同步命令输出的 `LIVE_TSV` 路径中的职责字段，再在无法唯一命中时只读搜索代码或查阅本节完整表格。
3. `registration_status=registered` 且 `remote_status=active` 的唯一高置信度结果可进入正常确认流程。`remote_status=missing` 时禁止按旧表创建分支；应从工蜂新增分支、最新提交和相对 `origin/main` 的变更文件中查找候选，无法唯一判断时请用户选择。
4. `registration_status=unregistered` 表示工蜂已新增但人工职责表尚未登记。它可以成为候选，但必须明确提示“工蜂新增未登记”，不能仅凭名称自动断言页面归属。
5. 唯一高置信度匹配后，在选中仓库执行 `git fetch origin --prune`，记录 `origin/<branch>` 完整 commit SHA，再安全选择或切换 tracking branch；此时不 rebase、不修改文件。
6. 首次进入目标仓库时记录 `git status --porcelain=v1` 作为当前对话的未提交修改基线，并只说明一次。需要切换分支或 rebase 时，基线不为空则停止，禁止自动 switch、stash、reset、覆盖或删除；当前分支已是目标分支且 HEAD 已包含最新 `origin/main` 时，可保护基线并继续。后续本次开发产生的修改属于正常状态，不重复告警；只有基线之外的未知修改、职责范围外文件或无法安全区分的重叠才停止。同步脚本的实时 TSV 位于 Git 内部目录，不计为工作区修改。
7. 当前对话没有已确认上下文，或匹配结果将切换仓库、分支或职责模块时，分支决策必须以固定的连续三行确认区块收尾。已确认且仍在同一仓库、分支和职责范围内的连续任务直接继续，不重复询问。CodeBuddy UI 在工具调用之间自动展示的定位、检索和核查过程属于产品级过程消息，不作为 Rule 是否加载的验收标准。
8. 用户确认后、首次写操作前再次 fetch。确认在当前 CodeBuddy 对话、仓库和分支内持续有效；同一页面的补充修改、验证、预览、提交、推送和 MR 均复用该确认。需求明显转到其他仓库、分支或职责模块，实际分支被外部切换，或 CodeBuddy 输出新的确认区块时，旧确认失效。用户确认的是分支而非 commit SHA；同一分支 HEAD 更新后同步即可继续，分支被删除、改名或异常分叉时才停止说明。
9. 本地 `PreToolUse` Hook 复核“当前对话中最新确认卡片、其后的用户确认、实际 Git 仓库与分支”。最新确认在后续普通消息中保持有效；出现新的确认卡片后，必须等待用户再次确认。同一会话、仓库、分支和 tracking HEAD 的普通连续写操作可复用 5 分钟工蜂校验缓存；`rebase`、`commit`、`push`、`merge` 等关键 Git 操作始终实时查询工蜂。
10. 首次启用或外部更新 Hook 后，用户需在 CodeBuddy `/hooks` 面板批准“ClawPro 分支确认与单仓写入门禁”；不得绕过 Hook 继续开发。

固定确认格式：

```text
【分支匹配待确认】
分支：<branch>
请确认是否在该分支继续。
```

实际确认区块不得放进代码块。Hook 从回复中识别上述连续三行区块；CodeBuddy 自动展示的过程消息不会使确认失效。确认区块输出后必须停止，用户确认前不修改代码、不 rebase、不提交、不推送。确认后，只要仍是同一对话、仓库和分支，后续补充要求不需要再次回复“确认”。本次任务产生的未提交修改不是异常；只有“工蜂新增未登记”、多候选、职责冲突、需要切换或 rebase、出现基线外未知修改或其他风险时，才追加最少必要提示。

## 5. 首次使用

### 5.1 确认工蜂权限

开始前必须确认：

- 使用自己的工蜂账号，不共用账号。
- 已加入 `openclaw-enterprise` 仓库并具备 Developer 权限。
- 权限检查地址：<https://git.woa.com/cvm-openclaw/openclaw-enterprise/-/project_members>。

没有权限时，先联系 `qitinghuang`，不要借用他人账号或凭据。

### 5.2 配置 SSH Key

在本地终端执行：

```bash
ssh-keygen -t ed25519 -C "你的企业邮箱"
```

生成结果中：

- `~/.ssh/id_ed25519` 是私钥，禁止发送给任何人或提交到仓库。
- `~/.ssh/id_ed25519.pub` 是公钥，可添加到工蜂。
- 是否设置私钥口令应遵循公司安全策略；本文不替代安全要求。

复制公钥：

```bash
cat ~/.ssh/id_ed25519.pub
```

打开 <https://git.woa.com>，进入“个人设置 → SSH Keys → Add new key”，粘贴完整公钥并保存。

测试连接：

```bash
ssh -T git@git.woa.com
```

首次连接如出现主机确认提示，应先核对域名为 `git.woa.com`，再按公司安全要求确认。看到欢迎信息后，SSH 配置完成。

### 5.3 打开 CodeBuddy 工作空间

日常使用直接在 CodeBuddy 打开工蜂仓库 clone 的根目录。常驻 Rule 会在每个独立新需求开始时生成工蜂实时 TSV，并结合仓库内人工职责索引匹配分支；同一开发上下文则复用已确认分支。因此不需要用户预先选择分支、手动更新远端状态、读取完整手册、重复粘贴开发 Prompt 或为每个小调整重复确认。团队版文件全部使用仓库相对路径，仓库位于任意本地目录都应生效。

新建对话后直接发送需求，例如：

```text
在企业规范库页面优化搜索框提示文案，完成后只本地预览，不提交、不推送。
```

CodeBuddy 必须先自动匹配或切换分支，并输出 `【分支匹配待确认】`。用户确认后，它才可以同步 `origin/main`、修改代码和执行检查；同一对话中继续调整该页面时直接描述即可，不再重复确认分支。

### 5.4 远端不存在目标分支

先运行同步脚本并检查实时 TSV 中的 `remote_status`。确认目标分支为 `missing` 时：

1. 停止开发。
2. 联系 `qitinghuang` 确认分支应当新建、改名还是已停用。
3. 仅在获得明确确认后，从最新 `origin/main` 创建分支。

建议让 CodeBuddy 执行以下意图，不要从当前未知分支直接复制历史：

```text
目标分支 <feature-branch> 在远端不存在，并且我已经获得负责人确认可以新建。
请先执行 git fetch origin --prune，确认工作区干净，再基于最新 origin/main 创建
<feature-branch>，推送到远端并设置 upstream。完成后汇报基准提交和远端分支地址。
```

### 5.5 验证设计规范

当前 `main` 已包含以下入口：

- `.codebuddy/rules/clawpro-design.md`
- `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md`
- `.codebuddy/skills/clawpro-walkthrough/SKILL.md`

仓库内入口存在时，无需导入外部压缩包。若缺失：

1. 先确认当前分支已同步最新 `origin/main`。
2. 再确认是否检出完整仓库内容。
3. 仍然缺失时，联系 `addietang`，不要自行安装来源不明或旧版本 Skill。

## 6. 每次开发的标准流程

每个需求或独立功能都必须完整执行本章。即使昨天在同一项目中开发，今天开始新工作时也要重新从步骤 1 开始。

### 步骤 1：开工前同步主干

复制并替换 `<feature-branch>`：

```text
开始本次开发前，请先检查 git status 和当前分支，执行 git fetch origin --prune；
同步 origin/<feature-branch> 后，再将 <feature-branch> rebase 到最新 origin/main。

要求：
- 首次记录未提交修改基线；只有需要 switch/rebase、出现基线外未知修改或覆盖风险时才停止，不要自动 stash、覆盖或删除。本次任务产生的修改不重复告警；
- 当前分支仅落后 origin/<feature-branch> 时只做 fast-forward；发生分叉时停止说明；
- 出现冲突时先停止，逐项说明冲突文件、双方改动意图和建议处理方式；
- 不要使用 --force，不要修改 main；
- 完成后汇报当前 HEAD、origin/main、ahead/behind 状态和变更文件列表。
```

### 步骤 2：描述需求与边界

使用以下模板，避免只说“做一个页面”：

```text
请严格遵循仓库中的 ClawPro 设计规范和当前分支边界完成开发。

- 目标分支：<feature-branch>
- 页面或模块：<page-or-module>
- 使用者：<user-role>
- 业务场景：<scenario>
- 需要完成：<requirements>
- 验收标准：<acceptance-criteria>
- 不在本次范围：<non-goals>
- 账号模式：<普通 / OneID 专用 / 统一 / 三种都适用>

开始编码前请先说明计划修改的文件。禁止修改 App.tsx、AdminLayout.tsx、
导航、路由、新子页面或新 Tab；如需求必须涉及这些内容，先停止并说明原因。
```

### 步骤 3：遵循设计规范

开发页面时必须：

- 使用标准组件和 Token。
- 不自行发明圆角、颜色、字号或间距。
- 不用普通 `div` 拼装已有标准组件能够表达的卡片。
- 不使用 emoji 或手写 SVG 代替图标库。
- 危险操作使用规范指定的确认交互，不用普通弹窗代替。
- 不直接使用来源不明的网络图片。
- 规范库缺少的能力使用 `needs-design-confirmation` 标记，并联系设计接口人。

设计自查提示词：

```text
请读取当前仓库的 ClawPro 设计规范，对本次页面从头到尾自查。
列出所有问题并按“严重 / 中等 / 轻微”分级；每项说明文件位置、问题原因和
修改方案。修复后再执行一次自检，确认无遗留。

重点检查标准组件、Token、圆角、颜色、字号、间距、图标、危险操作、空状态、
加载状态、错误状态和响应式表现。规范缺失项标记 needs-design-confirmation，
不要自行创造设计规则。
```

### 步骤 4：覆盖三种账号模式

开发以下页面时，应特别检查普通模式、OneID 专用模式和统一模式：

- 管控端基础信息配置。
- 用户管理。
- Agent 列表。
- Tokens 监控。

如需求方没有特别说明，默认本次改动应对三种模式同时生效。提示词：

```text
这个页面有普通模式、OneID 专用模式和统一模式。除非我明确说明例外，
本次改动默认对三种模式都生效。开发前请定位三个模式的条件分支；开发后
分别验证并汇报结果，避免只修改当前预览模式。
```

### 步骤 5：开发中持续控制范围

每完成一个可独立验证的小单元，检查：

- `git status` 中是否只有本需求文件。
- 是否误改公共基建、锁文件、生成文件或无关格式。
- 是否引入硬编码颜色、间距、文案或模拟数据。
- 是否处理空状态、加载状态、错误状态和权限差异。
- 是否在三个账号模式中产生行为差异。

发现跨分支依赖时，不要把其他页面一起改入当前分支；在 MR 中记录依赖，并联系对应分支负责人协同。

### 步骤 6：提交前质量检查

让 CodeBuddy 执行：

```text
本次开发完成。请先不要提交或推送，执行提交前检查：
1. 汇总 git diff，并确认所有变更都属于本需求；
2. 检查是否误改 App.tsx、AdminLayout.tsx、导航、路由、新子页面或新 Tab；
3. 读取 package.json，运行现有的类型检查和构建命令；缺少 Node、pnpm、测试浏览器或其他运行时环境时，先说明安装位置和影响并征得用户确认，不得自行安装全局工具或修改用户环境；
4. 运行与本次改动相关的现有检查，不要虚构不存在的测试脚本；
5. UI 改动必须读取当前启动脚本和路由配置，启动或复用本地开发/预览服务，从终端输出取得实际端口，验证目标页面可访问，并返回 `http://localhost:<实际端口>/<目标路由>` 形式的可点击直达链接；不得写死默认端口、只返回首页或仅声称“已预览”；用户说“只做本地修改”只代表不提交、不推送、不创建 MR，不代表跳过预览；
6. 在本地预览中验证主路径、空状态、错误状态和本次涉及的账号模式；
7. 最终回复前重新执行 `git status --short`，一次性区分开始前已有基线修改和本次任务修改；只要存在本地修改，就不得写“工作区干净”，但不得把本次任务正常改动描述为异常；
8. 列出未通过项、未验证项和剩余风险。

任何检查失败都先修复或说明，不要跳过后直接提交。本次引入的问题必须修复；分支既有且与本次无关的问题不得顺手修改，但提交或推送前必须列出证据并获得用户“仍然继续”的明确确认。
```

当前仓库至少提供以下基础检查：

```bash
pnpm check
pnpm build
```

是否执行其他命令，以当前 `package.json` 和改动范围为准。

UI 需求的交付回复必须包含可点击的目标页面预览链接。若默认端口被占用，以开发服务器终端实际输出为准；若服务无法启动，必须说明阻塞原因和用户可复现的启动命令，不得省略预览状态。

完成开发后的交付回复固定包含以下字段：

```text
已完成：<本次改动>
分支：<feature/*>
验证：<实际执行的检查及结果>
预览：<可点击的目标页面直达链接；非 UI 改动写“不适用”；失败写原因和复现命令>
工作区：<根据最新 git status --short 得出的未提交文件数和本次文件>
提交状态：<未提交/已提交>、<未推送/已推送>、<未创建 MR/MR 链接>
```

### 步骤 7：提交前再次同步主干

```text
质量检查通过后，请再次执行 git fetch origin --prune，先核对当前远端功能分支，
再将当前功能分支 rebase 到最新 origin/main。不要丢弃本次开发内容，也不要使用“全部采用 ours/theirs”之类的
批量冲突处理。

出现冲突时立即停止，按文件说明主干改动、本分支改动、建议保留内容和风险；
涉及设计规范、公共组件、路由或导航时，优先联系对应负责人确认。
```

### 步骤 8：提交、推送并创建或准备 MR

```text
请根据本次需求整理提交：
1. 再次确认当前分支是 <feature-branch>，目标分支是 main；
2. 生成清晰的提交信息和 MR 标题；
3. MR 描述包含背景、改动范围、验证结果、截图或预览、风险、未完成项；
4. 推送到远端 <feature-branch>；
5. 已安装且已认证工蜂 API/CLI 时创建合入 main 的 MR；不可用时返回工蜂预填创建链接，并明确写“MR 未创建，等待人工打开链接确认”，不得声称已创建；
6. 返回 MR 链接，但不要替我点击合入，也不要为创建 MR 自行安装 CLI 或索取、输出访问 Token。

不要为了“压缩提交”擅自重写其他协作者的远端历史。若 rebase 后确实需要更新
远端历史，先说明原因和影响；只有确认该远端分支没有其他人在共同使用后，
才可使用 --force-with-lease，禁止使用 --force。
```

### 步骤 9：人工检查并合入

打开 CodeBuddy 返回的 MR 链接，逐项确认：

- Source branch 是当前 `feature/*` 分支。
- Target branch 是 `main`。
- Diff 中没有无关文件和敏感信息。
- 没有未授权修改 `App.tsx`、`AdminLayout.tsx`、导航、路由、新页面或新 Tab。
- MR 标题、描述、验证结果和风险说明完整。
- 必要的检查和 CI 已通过。
- 审批人、合入权限和合入方式符合仓库当前设置。
- 页面效果已由需求方或对应负责人确认。

检查完成后，再由有权限的人在工蜂执行合入。

## 7. 冲突处理规范

### 7.1 基本规则

- 禁止批量选择“全部保留本分支”或“全部保留主干”。
- 按文件和业务意图逐项解决冲突。
- 主干中的新版设计规范、公共组件契约和公共基建通常是当前基线，但不得因此静默删除本需求的业务逻辑。
- 涉及其他特性分支负责范围时，联系相应负责人共同确认。
- 解决后必须重新运行类型检查、构建和页面验证。

### 7.2 无法确认时

停止 rebase，并向负责人提供：

- 冲突文件。
- `origin/main` 的改动意图。
- 当前分支的改动意图。
- 可选处理方案和各自影响。
- 是否会影响其他页面、账号模式或公共组件。

### 7.3 恢复错误操作

不得依赖 AI 对话历史恢复。优先使用：

- `git status` 判断当前操作状态。
- `git rebase --abort` 回到 rebase 前状态。
- `git reflog` 查找丢失提交。
- 已提交的本地或远端分支、备份分支和补丁。

执行重置、删除分支、覆盖文件或强推前，必须先说明目标提交和影响范围，并获得明确确认。

## 8. MR 描述模板

```markdown
## 背景

<!-- 为什么要改，关联需求是什么。 -->

## 改动范围

- 页面或模块：
- 主要改动：
- 明确未改：

## 验证结果

- [ ] `pnpm check` 通过
- [ ] `pnpm build` 通过
- [ ] 本地主路径通过
- [ ] 空状态、加载状态、错误状态已检查
- [ ] 普通模式已检查
- [ ] OneID 专用模式已检查
- [ ] 统一模式已检查

## 预览与截图

<!-- 填本地预览说明或截图。不要上传敏感数据。 -->

## 风险与回滚

- 风险：
- 回滚方式：

## 待确认项

- 无；或列出尚未确认的事实和负责人。
```

不涉及三种账号模式的页面，可将对应项标记为“不适用”，并说明原因，不要直接删除检查项。

## 9. 完成标准

只有同时满足以下条件，任务才算完成：

- [ ] 使用正确的特性分支，未直接修改 `main`。
- [ ] 开工前和提交前均已同步最新 `origin/main`。
- [ ] 变更仅覆盖当前需求和分支负责范围。
- [ ] 未经授权不修改公共基建、导航、路由、新页面或新 Tab。
- [ ] 已读取并遵循仓库内最新设计规范。
- [ ] 相关账号模式已验证。
- [ ] 类型检查和构建通过，或失败项已在 MR 明确说明并获得认可。
- [ ] 本地预览和关键状态已验证。
- [ ] MR Source、Target、Diff、描述、风险和验证结果均正确。
- [ ] 工蜂 CI 通过并完成必要审批。
- [ ] 合入 `main` 后，Stream CI/CD 构建和 demo 部署成功。
- [ ] demo 环境已完成冒烟验证。

demo 地址：<http://clawprodemo.woa.com/>。

## 10. 常见问题

### 10.1 `Permission denied (publickey)`

依次检查：

1. 是否使用自己的工蜂账号。
2. 是否具备仓库 Developer 权限。
3. `~/.ssh/id_ed25519.pub` 是否已添加到工蜂。
4. `ssh -T git@git.woa.com` 是否成功。
5. clone 地址是否为 SSH 地址。

禁止上传私钥让 AI 或他人代查。

### 10.2 远端没有目标分支

不要让 AI 从当前分支直接创建。先检查拼写，再联系 `qitinghuang` 确认分支状态；确认新建后，基于最新 `origin/main` 创建。

### 10.3 本地页面看起来不是最新 demo

先检查是否已 `fetch origin` 并 rebase 到最新 `origin/main`，再确认本地服务使用的是当前目标分支。不要通过复制 demo 产物覆盖源码。

### 10.4 设计 Skill 没有生效

确认已同步最新 `main`，并检查第 5.5 节的三个入口。仍有问题时联系 `addietang`，不要优先导入旧压缩包。

### 10.5 rebase 后代码不见了

立即停止后续操作。使用 `git reflog` 定位 rebase 前提交，必要时先创建恢复分支。不要继续提交、强推或依赖 AI 对话记录重写代码。

### 10.6 MR 无法合入

检查目标分支、冲突、CI、审批和仓库权限。不要绕过保护规则直接向 `main` 推送。

### 10.7 demo 没有更新

确认 MR 已合入 `main`、Stream CI 已触发且构建和部署均成功。失败时保留流水线链接和日志摘要，交给对应负责人处理。

## 11. 待确认项

以下信息没有在原手册中形成可验证的最终规则，应由负责人确认后再更新本文：

1. 远端未发现的 6 个登记分支是否应新建、改名或停用：
   - `feature/tenant-openclaw-detail-tools`
   - `feature/tenant-openclaw-detail-files`
   - `feature/tenant-help-docs`
   - `feature/tenant-reset-password`
   - `feature/admin-audit-log`
   - `feature/admin-skill-source`
2. 工蜂 MR 的必选审批人、最少审批数、允许合入人员和默认合入方式。
3. SSH 私钥口令的统一安全要求。
4. 三种账号模式检查是否已扩展到本规范列出的 4 个页面之外。
5. 团队、页面归属和接口人变更后的人工职责表维护责任人和更新频率。远端分支存在状态已由脚本在每个独立新需求中自动同步。

## 12. 规范维护

- 分支在工蜂新增、删除或 HEAD 更新时，无需人工修改实时状态；每个独立新需求会重新生成 `<git-dir>/clawpro-branch-map.live.tsv`。
- 页面归属、模块或团队变化时，通过 MR 更新第 4 节和 `.codebuddy/docs/clawpro-branch-ownership.tsv`。工蜂新增的 `unregistered` 分支确认职责后，应补录到人工职责表。
- 公共基建、设计 Skill、账号模式或 CI/CD 流程变化时，同步更新对应章节。
- 本规范与仓库保护规则、公司安全策略冲突时，以更严格且更新的规则为准，并及时修订本文。
