# CODEBUDDY.md — AI 协作入口

> 本文件是 AI 协作的统一入口，CodeBuddy / Claude Code / 其它 Agent 都从这里读取项目约定与 SOP。
> 功能文档维护在 `docs/`（按模块分类），任务产物维护在 `.specs/plans/<任务目录>/`，本文件只做索引 + 恢复协议 + 项目特定红线。

---

## 一、快速索引

| 想了解什么 | 文档位置 |
|-----------|---------|
| 项目整体架构、模块划分、关键流程 | `CLAUDE.md` |
| 功能文档索引（按模块分类） | `docs/INDEX.md` |
| 接口文档（完整版） | `docs/API.md` |
| 国际化方案 | `docs/i18n.md` |
| 单测编写指引 | `docs/testing.md` |
| 数据库 schema | `sql/init.sql`、`model/*.go` |
| Go 编码规范 | `.codebuddy/rules/code.md` |
| 单元测试 & 集成测试规范 | `.codebuddy/rules/unittest.md` |
| AI Code Review 指导 | `.specs/review.md` |
| 所有任务清单 | `ls .specs/plans/`（每个任务一个目录） |
| 单个任务的总览（**单一真相源**） | `.specs/plans/YYYY-MM-DD_<title>/00-overview.md` |

---

## 二、SOP（标准开发流程）

> **触发方式**：当用户描述需求、提出功能/修复/重构任务时，AI 自动进入 SOP 模式，从 Step 1 Clarify 开始逐步推进。
> **若需求意图模糊**：AI 应主动询问「这个需求是否需要进入开发流程（SOP）？」，而不是默认跳过。
> 也可以直接说：**「开始 SOP」**、**「按 SOP 处理」**、**「新建任务：<描述>」**。

> **SOP 入口判定（进入步骤前必做）**：
> 在按「SOP 启动前置：分支与任务判断」建立/定位任务目录后，AI **必须检查** `.specs/plans/<任务目录>/01-clarify.md` 是否已有有效内容（文件存在且非模板占位、包含明确的背景/目标/决策）：
> - **若 `01-clarify.md` 已是用户预先写好的有效澄清文档**：**跳过 Step 1 Clarify**，直接从 **Step 2 Plan** 开始。跳过前须：
>   1. 向用户确认：「检测到已有 Clarify 文档，将跳过 Clarify 步骤直接进入 Plan，是否确认？」
>   2. 在 `00-overview.md` 的 Progress 中把 `- [ ] 01. Clarify` 标记为 `- [x] 01. Clarify (用户预置，跳过)`
>   3. 在 `00-overview.md` 的「当前步骤」章节更新为 `⏳ 02. Plan`
>   4. 在 `00-overview.md` 的「时间记录」表中，把 `01. Clarify` 行的开始时间 / 结束时间填同一时间戳（当前时间），耗时填 `0`，备注列写「跳过：用户预置」
> - **若 `01-clarify.md` 不存在或只有模板占位**：按正常流程从 Step 1 Clarify 开始。

> 每步完成后：① 将产物写入 `.specs/plans/YYYY-MM-DD_<title>/0N-<步骤名>.md` 对应文件；② 更新本任务的 `00-overview.md`（Progress 勾选 + 当前步骤指针 + 必要的关键决策备忘）。**不需要**同步写任何全局任务总览文件。

### 上下文恢复（必须）

> **每次会话开始、clear 或 compact 之后，必须立即执行以下恢复流程，再继续任何工作：**
>
> 1. **读取本文件** `CODEBUDDY.md`，回顾项目规范和 SOP 流程。
> 2. **获取当前分支名**（`git branch --show-current`）。
> 3. **定位当前任务目录**：在 `.specs/plans/` 下遍历各 `00-overview.md`，匹配 Meta 中 `分支` 字段等于当前分支名的那个目录即为当前任务。
>    - 若匹配不到：说明是新需求，按「SOP 启动前置：分支与任务判断」处理。
> 4. **读取当前任务的 `00-overview.md`**，查看 Progress、当前步骤、关键决策备忘——这是本任务**唯一的**进度真相源。
> 5. **Lazy-load**：只读当前阶段对应的 `0N-<step>.md`，严禁一次性加载所有阶段文件。
> 6. **向用户汇报**当前任务进展，并询问是否继续推进。

### 步骤定义

#### SOP 启动前置：分支与任务判断

> 进入 **Clarify** 步骤之前，必须先执行以下操作：
>
> 1. 运行 `git branch --show-current` 获取当前分支名。
> 2. 在 `.specs/plans/` 下查找是否已有匹配当前分支的任务目录（遍历各 `00-overview.md` 的 Meta `分支` 字段）：
>    - **有对应任务**：说明是之前做过的需求，直接根据该任务的 `00-overview.md` 进度继续推进（参照「上下文恢复」流程）。
>    - **无对应任务且当前在 `master`/`main` 或其他无关分支上**：说明是全新需求，执行以下操作：
>      1. 基于最新主干创建新分支：`git fetch origin && git checkout -b feature/<任务短名> origin/master`
>      2. 从模板复制任务目录：`cp -r .specs/plans/_template .specs/plans/YYYY-MM-DD_<short-title>`
>      3. 编辑 `00-overview.md` 填入 Meta（分支、摘要、创建日期等）——这一步完成后，**本任务的单一真相源就建立好了**。
>      4. 然后进入上文「SOP 入口判定」（检查 `01-clarify.md` 是否已预置，决定从 Clarify 还是 Plan 开始）。
>    - **无对应任务且当前在非 `master`/`main` 且非 `feature/` 的分支上**：询问用户以当前分支还是 `master` 作为 base 创建 feature 分支，按用户选择执行后进入「SOP 入口判定」。
>    - **无对应任务但已在某个 `feature/` 分支上**：确认分支意图后按标准流程创建任务目录，进入「SOP 入口判定」。
>
> **禁止**在 `master`/`main` 分支上直接进行任何 SOP 步骤的工作。
> **禁止**在多个任务目录里同时填同一个分支名——一个分支只允许对应一个任务目录。

| # | 步骤 | 产物文件 | 说明 |
|---|------|---------|------|
| 1 | **Clarify** | `01-clarify.md` | AI 切换为产品经理角色做 Discovery + Challenge，产出背景、目标、待确认问题；同步更新 `00-overview.md` 的关键决策备忘 |
| 2 | **Plan** | `02-plan.md` | 改动文件、调用链、**测试用例设计（自然语言描述，先于实现）**、风险评估 |
| 3 | **Implement** | `03-implement.md` | 关键实现细节、特殊处理说明；遵循 Plan 中测试用例描述 |
| 4 | **UT** | `04-ut.md` | 用例执行结果、覆盖率、未覆盖行；规则参考 `.codebuddy/rules/unittest.md` |
| 5 | **Docs** | `05-docs.md` | 增量更新 `docs/API.md`、`docs/<模块>/` 等（IT 依赖文档生成 OpenAPI spec） |
| 6 | **IT** | `06-it.md` | 集成测试：构建镜像、K8s 部署、端到端验证、增量覆盖率；详见 `test/README.md` |
| 7 | **Review** | `07-review.md` | 代码审查结果 |
| 8 | **Commit** | `08-commit.md` | **执行顺序：① 写好 `08-commit.md`（含 commit message）→ ② 更新 `00-overview.md` Meta 状态为已完成 + Progress 勾选 08 → ③ 确认 `.specs/plans/<任务目录>/` 下所有 `0N-*.md` 均已就绪 → ④ `git add` 所有变更 → ⑤ `git commit` → ⑥ `git push`** |

> **每完成一步**必须（且只需要）：
> 1. 在对应 `0N-<step>.md` 写入产物
> 2. 在本任务 `00-overview.md` 的 Progress 将 `- [ ] 0N.` 原地替换为 `- [x] 0N. xxx (<摘要>)`
> 3. 在本任务 `00-overview.md` 的「当前步骤」章节更新为下一步
> 4. 在本任务 `00-overview.md` 的「时间记录」表中，把对应步骤行的「结束时间」填入当前时间（格式 `YYYY-MM-DD HH:MM:SS`，**精确到秒**），并补充「耗时」

### 步骤执行规则

> **每个步骤必须遵循「开始确认 → 记录开始时间 → 执行 → 记录结束时间 → 结束确认（含扭转交互）」五段式：**
>
> 1. **开始确认**：进入下一步骤前，必须向用户确认「是否开始 \<步骤名\>？」，得到明确同意后方可执行。
> 2. **记录开始时间**：用户同意开始的当下、动手执行之前，先在 `00-overview.md` 的「时间记录」表中把对应步骤行的「开始时间」填入当前本机时间（格式 `YYYY-MM-DD HH:MM:SS`，**精确到秒**）。
> 3. **执行**：按步骤要求完成工作，产出对应产物。
> 4. **记录结束时间**：步骤产物落地、向用户发起结束确认**之前**，把「时间记录」表中对应步骤行的「结束时间」填入当前时间，并补全「耗时」列。
> 5. **结束确认（步骤扭转交互）**：步骤执行完毕后，**必须**向用户主动发起扭转确认，格式固定为：
>    ```
>    当前步骤「<当前步骤名>」已完成
>    产物文件：.specs/plans/<任务目录>/0N-<step>.md
>    本步骤核心结论：<一句话摘要>
>    耗时：<开始时间> ~ <结束时间>（<耗时>）
>
>    下一步将进入「<下一步骤名>」：<1-2 句话概览>
>    是否同意推进？（同意 / 暂停 / 调整）
>    ```
>    - 必须等待用户**明确回复同意**后方可推进
>    - 若用户回复「调整」：留在当前步骤，按用户意见修正产物后**再次**发起结束确认
>    - 若用户回复「暂停」：停留在当前步骤，等待下次指令恢复
>
> **禁止**未经结束确认就自动跳入下一步骤。

> **进度格式**（写入本任务 `00-overview.md` 的 Progress / 当前步骤章节）：
> `Clarify → Plan → Implement → UT → Docs → IT → Review → Commit`

### Commit 规范

**格式（Conventional Commits）**：
```
<type>(<scope>): <subject>

<body>（可选）

type:  feat | fix | refactor | test | docs | chore | perf | ci
scope: 模块名（controller / model / scripts 等）
```

**示例**：
```
feat(controller): add user batch import API

feat(model): support MySQL distributed lock
fix(llm_proxy): handle nil pointer in streaming response
docs(api): update instance management endpoints
```

---

## 三、开发准则

> 详细规范见 `.codebuddy/rules/code.md`；下方为快速参考索引。

### 语言 / 框架版本约束

| 项目 | 要求 |
|------|------|
| Go 版本 | 1.21+（使用 go.mod 管理） |
| HTTP 框架 | `net/http`（标准库，无第三方框架） |
| ORM | GORM v2 |
| 前端 | Vue.js（SPA） |
| 构建 | `make build`（开发）/ `make release`（生产） |

### 代码检测

```bash
# 格式化（强制）
gofmt -w .
goimports -w .

# 静态检查
go vet ./...

# 单元测试
go test ./... -v -race
```

### 新增 API Handler 步骤

1. 在 `controller/` 目录创建或修改对应文件
2. 实现 handler 函数 `func(w http.ResponseWriter, r *http.Request)`
3. 在 `main.go` 注册路由
4. 若为写接口：在 `controller/audit.go` 的 `auditRules` 中添加审计规则，路由注册时用 `WithAudit()` 包装
5. 所有用户可见的错误/提示文案使用 `i18n.T(r.Context(), i18n.MsgXxx)`
6. 更新 `docs/API.md` 接口文档
7. 若涉及数据库变更：更新 GORM model + `sql/init.sql` + 新增增量 migration SQL（命名规则见下方）

### 数据库变更规范

修改 GORM model 时必须同步维护：
1. `model/*.go` — struct 定义（SQLite 通过 AutoMigrate 自动生效）
2. 若为新模型，加入 `model/db.go` 的 `allModels` 切片
3. `sql/init.sql` — 对应的 `CREATE TABLE`（新 MySQL 部署用）
4. `sql/<MMDD>-<描述>.sql` — 增量迁移文件（现有 MySQL 升级用）
5. 若为新表，在 `model/migrate.go` 的 `MigrateFromSQLite` 中添加该表的迁移逻辑（`checkMigrationCoverage` 会校验所有 `allModels` 表均被覆盖）

**增量迁移文件命名规范**：`<MMDD>-<kebab-case-描述>.sql`
- `MMDD` 为目标 Release 分支的日期，例如合并到 `Release/2026_06_18` 则以 `0618` 开头
- 描述用英文 kebab-case，简明概括变更内容
- 示例：`0618-add-user-group-quota.sql`、`0618-instance-add-proxy-port.sql`

### 国际化（i18n）

项目使用 `i18n/` 包实现多语言支持（中/英），基于 `golang.org/x/text/message`：

- **新增文案**：在 `i18n/keys.go` 定义 `Key`（中文原文）→ 在 `i18n/en.go` 注册英文翻译
- **使用翻译**：`i18n.T(r.Context(), i18n.MsgXxx, args...)`
- **语言检测**：URL `?lang=en` > `Accept-Language` 头 > 默认语言
- **异步场景**：goroutine 中用 `i18n.WithPrinter(detachedCtx, r.Context())` 传递语言偏好
- **禁止**：在 handler 中硬编码中文字符串作为用户响应

### 安全基线

1. SQL 必须使用 GORM ORM 接口，禁止 `db.Exec()`/`db.Raw()` 等裸 SQL。
2. 禁止硬编码密钥、Token、密码。
3. 所有外部输入必须校验。
4. HTTP handler 必须检查用户权限（`requireLogin` / `requireAdmin`）。
5. API 响应不得包含敏感信息（密码、密钥等）。
6. 腾讯云 SDK Client 使用统一工厂函数 `controller.GetXxxClient(ctx)`，禁止自行 `New`。

### 测试

- Plan 阶段先用**自然语言描述**测试用例（场景、输入、预期输出）
- Implement 阶段根据描述编写测试代码

**单元测试**：`go test ./... -v -race`
**覆盖率**：新增代码 >= 80%，P0 用例 100% 通过

**增量覆盖率检查**：
```bash
# 本地运行 CI 覆盖率检查脚本（全量 + 增量）
BASE_BRANCH=origin/master bash .ci/ci-check-coverage.sh
# 报告输出：coverage-report/index.html
```

**集成测试**：`make test IMAGE=<镜像> TEST_ARGS="--ak xxx --sk xxx"`
**API 覆盖率要求**：
- 新增 API 接口必须有集成测试用例覆盖
- 新增请求参数必须在集成测试中被传入覆盖
- 通过增量覆盖率报告验证（需指定 `BASE_BRANCH`）

**获取增量覆盖率**：
```bash
# 生成当前分支 + 基线分支的 OpenAPI spec
make openapi BASE_BRANCH=origin/master

# 运行集成测试并生成覆盖率报告
make test IMAGE=<镜像> BASE_BRANCH=origin/master TEST_ARGS="--report-dir ./test-report"

# 报告输出：test-report/coverage.html（含增量覆盖率章节）
```

详见 `.codebuddy/rules/unittest.md` 和 `test/README.md`。

---

## 四、禁止红线

| # | 红线 | 后果 |
|---|------|------|
| 1 | 使用裸 SQL 接口（`db.Exec`/`db.Raw`/`db.Table`/`db.Row`/`db.Rows`） | 多租户隔离失效，禁止合并 |
| 2 | 写接口缺少审计日志（未在 `auditRules` 注册 + `WithAudit` 包装） | 审计缺失，禁止合并 |
| 3 | 修改 GORM model 但未同步更新 `sql/init.sql`、增量 migration SQL 或 `MigrateFromSQLite` | 数据库不一致，禁止合并 |
| 4 | 破坏公共 API 兼容性（删除字段、修改语义） | 接口契约违反，禁止合并 |
| 5 | 在 handler 中使用 `model.DB` 而非 `model.DB(r.Context())` | 多租户数据越权，禁止合并 |
| 6 | 直接 `New` 腾讯云 SDK Client 而非使用 `controller.GetXxxClient(ctx)` | 配置不统一，禁止合并 |
| 7 | 硬编码配置信息（IP、端口、密钥等） | 安全漏洞，禁止合并 |
| 8 | 异步 goroutine 直接使用 `r.Context()` 而非 `common.DetachContext(r.Context())` | context cancel 导致异常，禁止合并 |
| 9 | 在 `master`/`main` 分支直接进行 SOP 开发步骤 | 流程违规，应切 feature 分支 |
| 10 | 修改 API 但未更新 `docs/API.md` | 文档缺失，禁止合并 |
| 11 | 面向用户的文案硬编码中文而非使用 `i18n.T()` | 国际化缺失，禁止合并 |
| 12 | 新增 `i18n.Key` 但未在 `en.go` 中添加英文翻译 | 翻译缺失，禁止合并 |
| 13 | 新增 API 接口或参数但集成测试未覆盖 | 增量覆盖率不达标，禁止合并 |
