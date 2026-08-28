---
type: always
---
你是一个专业的 Go 开发助手，专注于 hatchery 项目开发。所有代码必须严格遵循以下编码规范。

## 基本原则

- 【必须】使用 `gofmt` 格式化代码，禁止提交未格式化代码
- 【必须】通过 `go vet` 和 `staticcheck` 检查，无错误方可提交
- 【必须】遵循 Go 官方 Effective Go 和 Code Review Comments 指南
- 【必须】保持代码简洁，优先使用标准库

## 项目架构约束

### 分层结构

| 层 | 目录 | 职责 |
|----|------|------|
| 路由 & 中间件 | `main.go` | 路由注册、中间件链、服务启动 |
| Controller | `controller/` | HTTP handler、请求解析、响应构造 |
| Model | `model/` | GORM 模型定义、数据库操作 |
| Task | `task/` | 后台任务 |
| Config | `config/` | 配置管理 |
| Common | `common/` | 通用工具函数 |
| Scripts | `scripts/` | 远程执行脚本 |

### 路由注册

```go
// main.go 中注册路由
mux.HandleFunc("POST /api/resource", controller.WithAudit(controller.HandleResource))
```

- 使用 Go 1.22+ 的路由语法 `METHOD /path`
- 写接口（POST/PUT/DELETE）必须用 `WithAudit()` 包装

## 编码规范

### 命名

- 【必须】包名小写、短、无下划线：`controller`、`model`
- 【必须】导出函数/类型用 PascalCase：`HandleLogin`、`UserInstance`
- 【必须】非导出函数/变量用 camelCase：`getUserFromToken`、`parseRequest`
- 【必须】常量用 PascalCase 或 UPPER_SNAKE（视可见性）
- 【必须】接口名不加 `I` 前缀；单方法接口用方法名 + `er` 后缀
- 【推荐】避免包名与导出名重复：`model.User` 而非 `model.ModelUser`

### 错误处理

- 【必须】不要忽略返回的 error，必须处理或显式 `_ =` 标记
- 【必须】内部 error wrap 信息小写开头、不以标点结尾：`fmt.Errorf("open file: %w", err)`
- 【必须】面向用户的错误使用 `i18n.T()` 而非硬编码字符串
- 【必须】使用 `%w` 包装错误以保留错误链
- 【必须】HTTP handler 中使用 `writeError(w, r, code, msg)` 返回错误
- 【推荐】优先使用 sentinel error 或自定义类型做错误判断

### 函数

- 【推荐】函数体不超过 80 行，超过考虑拆分
- 【必须】多返回值最后一个为 error
- 【推荐】避免超过 5 个参数，使用 struct 传递
- 【推荐】`context.Context` 作为第一个参数

### 并发

- 【必须】goroutine 中使用 `common.DetachContext(r.Context())` 而非直接传 `r.Context()`
- 【必须】避免 goroutine 泄漏，确保有退出条件
- 【推荐】优先使用 channel 通信而非共享内存
- 【推荐】使用 `sync.WaitGroup` 等待 goroutine 完成

## 数据库操作规范

### GORM 使用

```go
// 正确 — 走 GORM 回调链，多租户自动注入
db := model.DB(r.Context())
db.Where("id = ?", id).First(&user)
db.Create(&instance)
db.Model(&User{}).Where("id = ?", id).Update("name", name)

// 错误 — 绕过回调，多租户隔离失效
db.Exec("UPDATE users SET name = ? WHERE id = ?", name, id)
db.Raw("SELECT * FROM users WHERE id = ?", id).Scan(&user)
db.Table("users").Where("id = ?", id).Update("name", name)
```

**禁止的接口**：`db.Exec()`、`db.Raw()`、`db.Table()`、`db.Row()`、`db.Rows()`、`db.ScanRows()`、`db.Connection()`

### DB 访问模式

| 场景 | 写法 |
|------|------|
| HTTP handler 内 | `model.DB(r.Context())` |
| 全局表（无 Identifier 字段） | `model.DBGlobal(ctx)` |
| 后台任务/goroutine | `model.DB(common.DetachContext(r.Context()))` |

### Schema 变更

修改 GORM model 时必须同步：
1. `model/*.go` 中的 struct（SQLite 通过 AutoMigrate 自动生效）
2. 若新增模型，加入 `model/db.go` 的 `allModels` 切片
3. `sql/init.sql` 中对应的 `CREATE TABLE`（新 MySQL 部署用）
4. `sql/<MMDD>-<描述>.sql` 增量迁移文件（现有 MySQL 升级用）
5. 若为新表，在 `model/migrate.go` 的 `MigrateFromSQLite` 函数中添加该表的迁移逻辑

**迁移文件命名**：`<MMDD>-<kebab-case-描述>.sql`
- `MMDD` 取目标 Release 分支日期（如 `Release/2026_06_18` → `0618`）
- 示例：`0618-add-user-group-quota.sql`

**SQLite→MySQL 迁移覆盖校验**：`checkMigrationCoverage()` 会遍历 `allModels` 检查每张表是否在 `MigrateFromSQLite` 中被处理。新增表但未添加迁移逻辑会导致校验报 missing。

## 多租户 Context 规范

```go
// HTTP handler — 直接用 r.Context()
func HandleFoo(w http.ResponseWriter, r *http.Request) {
    db := model.DB(r.Context())
    // ...
}

// 异步 goroutine — 必须 DetachContext
go func(ctx context.Context) {
    model.DB(ctx).Update(...)
}(common.DetachContext(r.Context()))
```

## 腾讯云 SDK 规范

```go
// 正确 — 使用统一工厂函数
cli, err := controller.GetCVMClient(ctx)
cli, err := controller.GetVPCClient(ctx)
cli, err := controller.GetSTSClient(ctx)

// 错误 — 直接 New
cli, err := cvm.NewClient(credential, region, cpf)
```

## HTTP Handler 标准写法

```go
func HandleCreateFoo(w http.ResponseWriter, r *http.Request) {
    user := getLoginUser(r)
    if user == nil {
        writeError(w, r, http.StatusUnauthorized, i18n.T(r.Context(), i18n.MsgUnauthorized))
        return
    }

    var req struct {
        Name string `json:"name"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, r, http.StatusBadRequest, i18n.T(r.Context(), i18n.MsgBadRequest))
        return
    }

    db := model.DB(r.Context())
    // 业务逻辑...

    jsonOK(w, r, result)
}
```

## 国际化（i18n）规范

项目使用 `golang.org/x/text/message` 实现国际化，核心文件位于 `i18n/` 包。

### 架构

```
i18n/
├── i18n.go       # Middleware、Printer(ctx)、T(ctx, key, args...) 核心函数
├── keys.go       # 所有 Key 定义（中文原文作为 key）
└── en.go         # 英文翻译注册（message.SetString）
```

### 使用方式

```go
// 在 handler 中翻译消息
msg := i18n.T(r.Context(), i18n.MsgInstanceNotFound)

// 带格式化参数
msg := i18n.T(r.Context(), i18n.MsgQuotaExceeded, used, limit)

// 返回错误时
writeError(w, r, http.StatusNotFound, i18n.T(r.Context(), i18n.MsgInstanceNotFound))
```

### 新增文案规则

1. **在 `i18n/keys.go` 中定义 Key**：按模块分组，中文原文作为 `string` 值
   ```go
   var MsgXxxFailed = Key{string: "操作失败"}
   ```

2. **在 `i18n/en.go` 中注册英文翻译**：
   ```go
   message.SetString(language.English, MsgXxxFailed.string, "Operation failed")
   ```

3. **在代码中使用 `i18n.T(ctx, key, args...)`**，禁止硬编码用户可见的中文字符串

### 强制规则

- 【必须】所有面向用户的错误信息、提示文案必须使用 `i18n.T()` 或已定义的 `i18n.Key`
- 【必须】禁止在 handler/controller 中硬编码中文字符串作为响应内容
- 【必须】新增 Key 时必须同时在 `en.go` 中添加英文翻译
- 【必须】Key 的中文原文即为默认语言输出，保持简洁明确
- 【必须】带参数的 Key 使用 `%s`/`%d`/`%v` 占位符，与 `fmt.Sprintf` 语法一致
- 【必须】异步 goroutine 中若需 i18n，使用 `i18n.WithPrinter(parent, src)` 传递语言偏好
- 【推荐】Key 变量命名格式：`Msg<模块><描述>`，如 `MsgInstanceNotFound`、`MsgModelQuotaExceeded`

### 语言检测优先级

1. URL 查询参数 `?lang=en`
2. `Accept-Language` 请求头
3. 默认语言（通过 `SetDefaultLang` 设置，默认中文）

## 日志规范

详见 `docs/ops/logging.md`，以下为强制要求：

- 【必须】HTTP handler 中需要链路追踪时使用 `controller.Logger(ctx)`（自动携带 request_id 等）
- 【必须】后台任务/启动逻辑等无需链路追踪的场景，直接使用 `slog.Info/Warn/Error` 即可
- 【必须】腾讯云 SDK 调用通过 `CallSDKAPITyped` / `CallSDKAPICommon` 包装，禁止直接调 SDK 方法
- 【必须】定时任务使用 `common.WithTaskTrace(ctx, "task_name")` 创建链路 ctx
- 【必须】日志消息使用 `[模块名]` 前缀，结构化字段用 key-value 对
- 【禁止】将错误信息拼接进消息字符串（`log.Error("[X] 失败: " + err.Error())`）

## 安全基线

- 【必须】所有写接口有权限检查（`requireLogin` / `requireAdmin`）
- 【必须】SQL 通过 GORM ORM，禁止字符串拼接
- 【必须】禁止硬编码密钥/Token/密码
- 【必须】外部输入必须校验
- 【必须】API 响应不含敏感信息
- 【必须】写接口必须有审计日志

## 禁止的做法

- 【必须】禁止使用 `panic` 做业务错误处理（仅用于不可恢复的程序错误）
- 【必须】禁止忽略 error 返回值
- 【必须】禁止在 init() 中做复杂逻辑
- 【必须】禁止使用全局可变状态（package-level var）做请求间共享
- 【必须】禁止在 handler 中直接 `os.Exit`
- 【必须】禁止使用 `unsafe` 包（除非有充分理由并注释）
