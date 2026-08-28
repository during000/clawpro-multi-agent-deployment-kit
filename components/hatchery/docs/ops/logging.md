# 日志使用指引

本文档描述 Hatchery 项目的日志规范，包括日志体系架构、各类日志函数的使用方式，以及业务代码中的最佳实践。

---

## 目录

- [整体架构](#整体架构)
- [日志级别规范](#日志级别规范)
- [Context 链路追踪](#context-链路追踪)
- [业务代码日志：Logger(ctx)](#业务代码日志loggercx)
- [结构化日志函数](#结构化日志函数)
- [腾讯云 SDK 调用日志](#腾讯云-sdk-调用日志)
- [数据库操作日志](#数据库操作日志)
- [定时任务日志](#定时任务日志)
- [敏感信息脱敏](#敏感信息脱敏)
- [常见错误用法](#常见错误用法)

---

## 整体架构

日志体系基于 Go 标准库 `log/slog`，在 `controller/access_log.go` 中封装了一套结构化日志规范。

```
HTTP 请求
  └── logMiddleware（main.go）
        ├── NewRequestContext()     → 注入 request_id / trace_id / interface / uin
        ├── InjectSubUin()          → 注入 subuin（用户 ID）
        ├── LogRcvRequest()         → 记录"接收请求"日志
        ├── handler 执行（业务逻辑）
        │     └── Logger(ctx).Info/Warn/Error()  → 业务日志（自动携带链路字段）
        │     └── CallSDKAPITyped()              → 自动记录 SDK 调用日志
        │     └── db.WithContext(ctx).xxx()      → 自动记录 DB 操作日志
        └── LogSendResponse()       → 记录"响应"日志
```

**日志初始化**（`main.go`）：

```go
// 启动时根据 --log-json 参数选择文本或 JSON 格式，并附加 hostname 字段
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}).
    WithAttrs([]slog.Attr{slog.String("hostname", hostname)})
slog.SetDefault(slog.New(handler))

// 注册 DB 日志回调（在 model.InitDB 之后调用一次）
controller.RegisterDBLogger(model.DB(context.Background()))
```

---

## 日志级别规范

| 级别    | level_no | 使用场景                                      |
|---------|----------|-----------------------------------------------|
| DEBUG   | 100      | 调试信息，生产环境默认不输出                  |
| INFO    | 200      | 正常流程节点、操作成功                        |
| WARN    | 300      | 可恢复的异常、外部调用失败、4xx 响应          |
| ERROR   | 400      | 不可恢复的错误、5xx 响应、panic               |
| FATAL   | 500      | 致命错误（仅启动阶段使用 `os.Exit`）          |
| CRITICAL| 600      | 保留，暂未使用                                |

**级别自动推导规则**：

- HTTP 响应：`5xx → ERROR`，`4xx → WARN`，其余 → `INFO`
- 外部调用（SDK/HTTP）：失败 → `WARN`，成功 → `INFO`
- DB 操作：失败 → `WARN`，成功 → `INFO`

---

## Context 链路追踪

所有日志通过 `context.Context` 串联链路信息，确保同一请求的所有日志可以通过 `request_id` 关联。

### 公共链路字段

每条日志都会自动携带以下字段：

| 字段         | 说明                                              |
|--------------|---------------------------------------------------|
| `request_id` | 每次请求唯一 ID（UUID），由 `NewRequestContext` 生成 |
| `trace_id`   | 优先取请求头 `X-Request-ID` / `X-Trace-ID`，否则同 `request_id` |
| `uin`        | 腾讯云账号 UIN                                    |
| `subuin`     | 系统用户 ID（认证后注入）                         |
| `interface`  | 当前接口路径（如 `/admin/instances`）             |
| `level_no`   | 日志级别数值（便于 CLS 告警规则配置）             |

### HTTP 请求 Context

`logMiddleware` 在每个请求进入时自动注入，业务代码无需手动处理：

```go
// main.go 中间件自动完成，业务代码直接使用 r.Context() 即可
r = controller.NewRequestContext(r)
r = controller.InjectSubUin(r, user.ID)
```

### 定时任务 Context

定时任务需要手动创建 context，以便串联任务内所有日志：

```go
func myTask() {
    ctx := common.WithTaskTrace(context.Background(), "my_task_name")
    log := controller.Logger(ctx)
    log.Info("[MyTask] 任务开始")
    // ... 后续所有操作传入 ctx
}
```

---

## 业务代码日志：Logger(ctx)

**在 HTTP handler 中需要链路追踪时使用 `Logger(ctx)`**，自动携带 `request_id`、`trace_id` 等链路字段。后台任务、启动逻辑等无需链路追踪的场景，直接使用 `slog.Info/Warn/Error` 即可。

```go
// ✅ HTTP handler 中需要链路追踪：使用 Logger(ctx)
log := controller.Logger(ctx)
log.Info("[Admin] 开机", "admin", adminUser, "instanceId", instance.ID)
log.Warn("[CLS] 查询失败，跳过本轮", "error", err)
log.Error("[AdminDelete] 创建通知失败", "id", inst.ID, "error", err)

// ✅ 后台任务 / 启动逻辑 / 无需链路追踪：直接用 slog
slog.Info("[Init] 镜像初始化完成", "count", len(images))
slog.Warn("[STS] 凭证刷新失败，将重试", "error", err)
```

### 日志消息命名规范

- 使用 `[模块名]` 前缀标识来源，便于过滤，例如：`[AdminList]`、`[CLS Agent]`、`[LLM Proxy]`
- 消息描述操作结果，而非操作意图，例如：`"全量 CVM 查询完成"` 而非 `"开始查询 CVM"`

### 结构化字段规范

```go
// ✅ 正确：key-value 对形式，便于 CLS 检索
log.Info("[AdminList] 全量 CVM 查询完成",
    "count", len(allCvmIds),
    "duration", time.Since(cvmStart),
)

// ❌ 错误：将信息拼接进消息字符串，无法结构化检索
log.Info(fmt.Sprintf("[AdminList] 全量 CVM 查询完成，共 %d 个", len(allCvmIds)))
```

---

## 结构化日志函数

以下函数由框架自动调用（通过中间件），**业务代码通常不需要直接调用**。

### LogRcvRequest — 接收请求

```go
// 由 logMiddleware 自动调用，记录每个 HTTP 请求的入参
controller.LogRcvRequest(ctx, r, reqBody)
```

输出字段：`context.method`、`context.path`、`context.query`、`context.headers`、`context.body`

### LogSendResponse — 发送响应

```go
// 由 logMiddleware 自动调用，记录每个 HTTP 响应
controller.LogSendResponse(ctx, r, statusCode, headers, body, cost)
```

输出字段：`context.status_code`、`context.headers`、`context.body`、`context.cost`、`context.status`

### LogUncaughtException — 未捕获异常

```go
// 由 logMiddleware 的 panic recover 自动调用
controller.LogUncaughtException(ctx, r, "panic", 500, msg, stack)
```

输出字段：`context.class`、`context.code`、`context.message`、`context.trace`

### LogLLMStream — LLM 流式响应

```go
// 在 LLM 流式传输完成后调用，补充记录 token 用量
controller.LogLLMStream(ctx, statusCode, usage, cost, err)
```

输出字段：`context.status_code`、`context.prompt_tokens`、`context.completion_tokens`、`context.total_tokens`、`context.cost`、`context.status`、`context.error`

---

## 腾讯云 SDK 调用日志

**所有腾讯云 SDK 调用必须通过 `CallSDKAPITyped` 或 `CallSDKAPICommon` 包装**，禁止直接调用 SDK 方法，以确保调用日志自动记录。

### CallSDKAPITyped — 类型安全调用（推荐）

适用于有 SDK 封装的接口（CVM、TAT、STS 等）：

```go
// action 自动从返回类型名推导（去掉 "Response" 后缀）
// 例如：*cvm.DescribeInstancesResponse → action = "DescribeInstances"
resp, err := controller.CallSDKAPITyped(ctx, controller.SDKComponentCVM, req, client.DescribeInstances)
if err != nil {
    // 错误处理
}
// resp 直接是 *cvm.DescribeInstancesResponse，无需类型断言
```

SDK component 常量：

| 常量                        | 值      | 对应服务         |
|-----------------------------|---------|------------------|
| `controller.SDKComponentCVM` | `"cvm"` | 云服务器         |
| `controller.SDKComponentTAT` | `"tat"` | 自动化助手       |
| `controller.SDKComponentCLS` | `"cls"` | 日志服务         |
| `controller.SDKComponentSTS` | `"sts"` | 安全令牌服务     |

### CallSDKAPICommon — 通用客户端调用

适用于无 SDK 封装的接口（使用 `CommonClient`）：

```go
req := tchttp.NewCommonRequest("cls", "2020-10-16", "GetClsService")
req.SetActionParameters("{}")
resp, err := controller.CallSDKAPICommon(ctx, controller.SDKComponentCLS, req, client.Send)
if err != nil {
    // 错误处理
}
// resp 是 *tchttp.CommonResponse，调用 resp.GetBody() 获取响应体
```

### 自动记录的日志字段

SDK 调用日志复用 `Call http api` 规范，自动记录：

| 字段                          | 说明                   |
|-------------------------------|------------------------|
| `context.request.body`        | 请求参数（已脱敏）     |
| `context.request.interface`   | 接口名（如 `RunInstances`） |
| `context.request.region`      | 请求 Region            |
| `context.response.body`       | 响应体或错误信息       |
| `context.cost`                | 调用耗时（毫秒）       |
| `context.component`           | 服务名（如 `cvm`）     |
| `context.status`              | 是否成功               |

---

## 数据库操作日志

**DB 操作日志完全自动化**，业务代码只需在查询时传入 `ctx` 即可，无需任何额外操作。

```go
// ✅ 正确：传入 ctx，DB 操作日志自动记录
model.DB(ctx).First(&instance, id)
model.DB(ctx).Where("id IN ?", ids).Find(&instances)

// ❌ 错误：不传 ctx，DB 操作日志无法关联链路信息
model.DB(context.Background()).First(&instance, id)
```

`RegisterDBLogger` 在启动时注册了 GORM 的 Before/After 回调，自动拦截所有 SELECT / INSERT / UPDATE / DELETE / ROW / RAW 操作并记录日志。

自动记录的日志字段：

| 字段                           | 说明                   |
|--------------------------------|------------------------|
| `context.request.operation`    | 操作类型（SELECT 等）  |
| `context.request.table`        | 表名                   |
| `context.request.query`        | SQL 语句（已脱敏）     |
| `context.request.vars`         | 绑定参数               |
| `context.response.rows_affected` | 影响行数             |
| `context.cost`                 | 执行耗时（毫秒）       |
| `context.component`            | 固定为 `"sqlite"`      |
| `context.status`               | 是否成功               |

---

## 定时任务日志

定时任务需要手动创建 context，并在任务结束时调用 `LogCli` 记录任务执行摘要：

```go
func runMyTask() {
    ctx := common.WithTaskTrace(context.Background(), "my_task")
    start := time.Now()
    log := controller.Logger(ctx)

    log.Info("[MyTask] 任务开始")

    // 执行任务逻辑，所有子调用传入 ctx
    if err := doSomething(ctx); err != nil {
        log.Error("[MyTask] 执行失败", "error", err)
        return
    }

    log.Info("[MyTask] 任务完成")

    // 记录任务执行摘要（可选）
    controller.LogCli(ctx, "my_task", "query=xxx", time.Since(start))
}
```

`LogCli` 输出字段：`context.method`（任务名）、`context.query`（任务参数摘要）、`context.cost`（耗时）

---

## 敏感信息脱敏

框架自动对以下内容进行脱敏，业务代码无需手动处理：

**Header 脱敏**（`desensitizeHeaders`）：

匹配以下关键词的 Header 值替换为 `***`：
`secret_key`、`api_key`、`password`、`token`、`authorization`、`credential`、`cookie`

**Body 脱敏**（`desensitizeBody`）：

JSON Body 中匹配以下字段名的值替换为 `"***"`：
`secret_key`、`api_key`、`password`、`token`、`authorization`、`credential`

**Body 截断**：超过 4096 字节的 body 自动截断并追加 `...[truncated]`；二进制内容替换为 `--- binary ---`。

> ⚠️ **注意**：业务代码中如需手动记录可能含敏感信息的字符串，应避免直接将密钥、密码等字段作为日志字段值输出。

---

## 常见错误用法

### ❌ 不传 ctx 给 DB 操作

```go
// 错误：DB 日志无法关联 request_id
model.DB(context.Background()).First(&instance, id)

// 正确
model.DB(r.Context()).First(&instance, id)
```

### ❌ 直接调用 SDK 方法

```go
// 错误：SDK 调用不会被日志记录
resp, err := client.DescribeInstances(req)

// 正确
resp, err := controller.CallSDKAPITyped(ctx, controller.SDKComponentCVM, req, client.DescribeInstances)
```

### ❌ 在 HTTP handler 中不用 Logger(ctx)

```go
// 错误：handler 中直接用 slog，丢失链路字段（无法通过 request_id 关联日志）
func HandleFoo(w http.ResponseWriter, r *http.Request) {
    slog.Info("[Foo] 操作完成", "id", id)
}

// 正确：handler 中用 Logger(ctx)
func HandleFoo(w http.ResponseWriter, r *http.Request) {
    log := controller.Logger(r.Context())
    log.Info("[Foo] 操作完成", "id", id)
}
```

> 注：后台任务、启动逻辑等无需链路追踪的场景，直接用 `slog` 没问题。

### ❌ 将错误信息拼入消息字符串

```go
// 错误：无法结构化检索
log.Error("[Admin] 操作失败: " + err.Error())

// 正确：使用 key-value 对
log.Error("[Admin] 操作失败", "error", err)
```

### ❌ 在 goroutine 中使用外部 ctx（可能已取消）

```go
// 错误：HTTP 请求结束后 ctx 会被取消，goroutine 中的 DB/SDK 调用会失败
go func() {
    model.DB(ctx).Delete(&instance)  // ctx 可能已取消
}()

// 正确：使用 common.DetachContext 保留链路信息但脱离请求生命周期
go func(ctx context.Context) {
    model.DB(ctx).Delete(&instance)
}(common.DetachContext(r.Context()))
```
