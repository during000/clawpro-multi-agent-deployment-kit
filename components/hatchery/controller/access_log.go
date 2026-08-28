package controller

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"hatchery/common"
	"hatchery/controller/provider"
	"hatchery/i18n"

	"github.com/google/uuid"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"gorm.io/gorm"
)

// ---- Level Number Mapping ----

const (
	LevelNoDebug    = 100
	LevelNoInfo     = 200
	LevelNoWarn     = 300
	LevelNoError    = 400
	LevelNoFatal    = 500
	LevelNoCritical = 600
)

// SDK component 常量，用于 CallSDKAPITyped / CallSDKAPI 的 component 参数。
const (
	SDKComponentCVM = "cvm"
	SDKComponentCBS = "cbs"
	SDKComponentTAT = "tat"
	SDKComponentCLS = "cls"
	SDKComponentSTS = "sts"
)

func levelNo(l slog.Level) int {
	switch {
	case l >= slog.LevelError:
		return LevelNoError
	case l >= slog.LevelWarn:
		return LevelNoWarn
	case l >= slog.LevelInfo:
		return LevelNoInfo
	default:
		return LevelNoDebug
	}
}

// ---- Context Helpers ----

// NewRequestContext 生成 request_id / trace_id，注入 context 并返回。
func NewRequestContext(r *http.Request) *http.Request {
	requestID := uuid.New().String()
	traceID := r.Header.Get("X-Request-ID")
	if traceID == "" {
		traceID = r.Header.Get("X-Trace-ID")
	}
	if traceID == "" {
		traceID = requestID
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, common.CtxKeyRequestID, requestID)
	ctx = context.WithValue(ctx, common.CtxKeyTraceID, traceID)
	ctx = context.WithValue(ctx, common.CtxKeyInterface, r.URL.Path)
	ctx = context.WithValue(ctx, common.CtxKeyUin, resolveUinFromCtx(ctx))
	return r.WithContext(ctx)
}

// resolveUinFromCtx 返回当前租户的腾讯云 UIN。
func resolveUinFromCtx(ctx context.Context) string {
	if uin := common.CVMUinFromCtx(ctx); uin != "" {
		return uin
	}
	return ""
}

// InjectSubUin 将系统用户 ID 注入 context（在认证完成后调用）。
func InjectSubUin(r *http.Request, subUin uint) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), common.CtxKeySubUin, subUin))
}

// ---- IP Helpers ----

// ExtractClientIP 从请求中提取客户端 IP。
// 仅当直接连接方（RemoteAddr）属于可信代理网段时，才信任 X-Forwarded-For / X-Real-IP，
// 否则直接返回 RemoteAddr，防止外部伪造。
func ExtractClientIP(r *http.Request) string {
	callerIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		callerIP = r.RemoteAddr
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	return callerIP
}

// ExtractCallerIP 返回 TCP 对端 IP（RemoteAddr 的 IP 部分）。
func ExtractCallerIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// ---- Body Helpers ----

const maxBodyLog = 4096 // 超过此长度截断

// safeBody 读取 body 并截断，二进制内容返回 "--- binary ---"。
func safeBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	// 简单判断是否为二进制：含 NUL 字节
	for _, b := range body {
		if b == 0 {
			return "--- binary ---"
		}
	}
	s := string(body)
	if len(s) > maxBodyLog {
		return s[:maxBodyLog] + "...[truncated]"
	}
	return s
}

// ---- Sensitive Data Desensitization ----

var sensitiveKeys = regexp.MustCompile(
	`(?i)(secret(?:[_-]?key)?|api[_-]?key|password|token|authorization|credential|cookie)`)

var sensitiveBodyKeys = regexp.MustCompile(
	`(?i)("(?:[^"]*(?:secret|token|password|authorization|credential|cookie|api[_-]?key)[^"]*)"\s*:\s*)"[^"]*"`)

// desensitizeHeaders 对敏感 header 值进行脱敏。
func desensitizeHeaders(headers http.Header) []string {
	result := make([]string, 0, len(headers))
	for k, vs := range headers {
		if sensitiveKeys.MatchString(k) {
			result = append(result, fmt.Sprintf("%s: ***", k))
		} else {
			result = append(result, fmt.Sprintf("%s: %s", k, strings.Join(vs, "; ")))
		}
	}
	return result
}

// desensitizeBody 对 body 中的敏感字段进行脱敏（正则替换）。
func desensitizeBody(body string) string {
	return sensitiveBodyKeys.ReplaceAllString(body, `$1"***"`)
}

// 管理员创建实例时，channels[].config 的凭据键由通道定义决定，可能是
// encoding_aes_key 或任意自定义名称，不能依赖 sensitiveBodyKeys 穷举。
// 因此该接口完全不记录请求 body，只保留路径、方法、请求 ID 等元数据。
const adminCreateInstanceLogPath = "/admin/instances/create"

func requestBodyForLog(r *http.Request, body []byte) string {
	if r != nil && r.URL != nil && r.URL.Path == adminCreateInstanceLogPath {
		return ""
	}
	return desensitizeBody(safeBody(body))
}

// ---- Base Log Attrs ----

// baseAttrs 返回所有日志共用的基础字段。
func baseAttrs(ctx context.Context, r *http.Request, lvl slog.Level, iface, message string) []any {
	clientIP := ""
	callerIP := ""
	if r != nil {
		clientIP = ExtractClientIP(r)
		callerIP = ExtractCallerIP(r)
	}
	return []any{
		"client_ip", clientIP,
		"caller_ip", callerIP,
		"level_no", levelNo(lvl),
		"request_id", common.GetRequestID(ctx),
		"trace_id", common.GetTraceID(ctx),
		"uin", common.CVMUinFromCtx(ctx),
		"subuin", common.GetSubUin(ctx),
		"interface", common.GetInterface(ctx),
		"message", message,
	}
}

// ---- Log Functions ----

// LogRcvRequest 记录"接收请求"日志（Rcv request）。
func LogRcvRequest(ctx context.Context, r *http.Request, body []byte) {
	iface := common.GetInterface(ctx)
	if iface == "" && r != nil && r.URL != nil {
		iface = r.URL.Path
	}
	attrs := baseAttrs(ctx, r, slog.LevelInfo, iface, "Rcv request")
	if r != nil {
		attrs = append(attrs, "context.method", r.Method)
		if r.URL != nil {
			attrs = append(attrs,
				"context.path", r.URL.Path,
				"context.query", r.URL.RawQuery,
			)
		}
		attrs = append(attrs, "context.headers", desensitizeHeaders(r.Header))
	}
	attrs = append(attrs, "context.body", requestBodyForLog(r, body))
	slog.InfoContext(ctx, "Rcv request", attrs...)
}

// LogSendResponse 记录"响应"日志（Send response）。
// success 由内部根据 statusCode 自动推导（< 400 为成功），无需调用方传入。
func LogSendResponse(ctx context.Context, r *http.Request, statusCode int, headers http.Header, body []byte, cost time.Duration) {
	iface := common.GetInterface(ctx)
	if iface == "" && r != nil {
		iface = r.URL.Path
	}
	// 根据状态码动态选择日志级别：5xx → ERROR，4xx → WARN，其余 → INFO
	var lvl slog.Level
	switch {
	case statusCode >= 500:
		lvl = slog.LevelError
	case statusCode >= 400:
		lvl = slog.LevelWarn
	default:
		lvl = slog.LevelInfo
	}
	success := statusCode < 400
	attrs := baseAttrs(ctx, r, lvl, iface, "Send response")
	attrs = append(attrs,
		"context.status_code", statusCode,
		"context.headers", desensitizeHeaders(headers),
		"context.body", desensitizeBody(safeBody(body)),
		"context.cost", float64(cost.Milliseconds()),
		"context.status", success,
	)
	slog.Log(ctx, lvl, "Send response", attrs...)
}

// LogCallHTTPAPI 记录"http请求调用"日志（Call http api）。
func LogCallHTTPAPI(ctx context.Context, r *http.Request,
	reqHost, reqIP, reqMethod, reqURI string,
	reqHeaders http.Header, reqBody []byte,
	reqInterface, reqRegion string,
	respStatusCode int, respHeaders http.Header, respBody []byte,
	cost time.Duration, component string, success bool,
) {
	iface := common.GetInterface(ctx)
	if iface == "" && r != nil {
		iface = r.URL.Path
	}
	// 根据 success 选择日志级别，baseAttrs 与实际输出级别保持一致
	lvl := slog.LevelInfo
	if !success {
		lvl = slog.LevelWarn
	}
	attrs := baseAttrs(ctx, r, lvl, iface, "Call http api")
	attrs = append(attrs,
		"context.request.host", reqHost,
		"context.request.ip", reqIP,
		"context.request.method", reqMethod,
		"context.request.uri", reqURI,
		"context.request.headers", desensitizeHeaders(reqHeaders),
		"context.request.body", desensitizeBody(safeBody(reqBody)),
		"context.request.interface", reqInterface,
		"context.request.region", reqRegion,
		"context.response.status_code", respStatusCode,
		"context.response.headers", desensitizeHeaders(respHeaders),
		"context.response.body", desensitizeBody(safeBody(respBody)),
		"context.cost", float64(cost.Milliseconds()),
		"context.component", component,
		"context.status", success,
	)
	if success {
		slog.InfoContext(ctx, "Call http api", attrs...)
	} else {
		// 失败时升级为 WARN，便于 CLS 告警
		slog.WarnContext(ctx, "Call http api", attrs...)
	}
}

// LogUncaughtException 记录"异常兜底"日志（Uncaught exception）。
func LogUncaughtException(ctx context.Context, r *http.Request, class string, code int, message, trace string) {
	iface := common.GetInterface(ctx)
	if iface == "" && r != nil {
		iface = r.URL.Path
	}
	attrs := baseAttrs(ctx, r, slog.LevelError, iface, "Uncaught exception")
	attrs = append(attrs,
		"context.class", class,
		"context.code", code,
		"context.message", message,
		"context.trace", trace,
	)
	slog.ErrorContext(ctx, "Uncaught exception", attrs...)
}

// LogCli 记录"定时任务调用"日志（Cli）。
func LogCli(ctx context.Context, method, query string, duration time.Duration) {
	attrs := baseAttrs(ctx, nil, slog.LevelInfo, method, "Cli")
	attrs = append(attrs,
		"context.method", method,
		"context.query", query,
		"context.cost", float64(duration.Milliseconds()),
	)
	slog.InfoContext(ctx, "Cli", attrs...)
}

// ---- Response Capture Writer ----

// ResponseCapture 包装 http.ResponseWriter，捕获状态码和响应 body。
type ResponseCapture struct {
	http.ResponseWriter
	StatusCode  int
	Body        []byte
	IsStreaming bool // 是否为流式响应（SSE），由 Flush 调用触发标记
}

func NewResponseCapture(w http.ResponseWriter) *ResponseCapture {
	return &ResponseCapture{ResponseWriter: w, StatusCode: http.StatusOK}
}

func (rc *ResponseCapture) WriteHeader(code int) {
	rc.StatusCode = code
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *ResponseCapture) Write(b []byte) (int, error) {
	if len(rc.Body) < maxBodyLog {
		remaining := maxBodyLog - len(rc.Body)
		if len(b) <= remaining {
			rc.Body = append(rc.Body, b...)
		} else {
			rc.Body = append(rc.Body, b[:remaining]...)
		}
	}
	return rc.ResponseWriter.Write(b)
}

func (rc *ResponseCapture) Flush() {
	if f, ok := rc.ResponseWriter.(http.Flusher); ok {
		rc.IsStreaming = true
		f.Flush()
	}
}

// Hijack 透传底层 ResponseWriter 的 Hijack 接口，支持 WebSocket 连接升级。
// 仅在 WebSocket 代理（如 VNC WebSocket 代理）中被调用，不影响普通 HTTP 请求和 SSE 流式响应。
func (rc *ResponseCapture) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rc.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, common.I18nError(i18n.MsgHijackNotSupported)
}

// LogLLMStream 记录流式 LLM 响应完成后的结构化日志。
// 由于 SSE 流式响应的 body 是原始 SSE 帧，LogSendResponse 中只记录 [streaming] 占位符，
// 本函数在流式传输完成后补充记录 token 用量、延迟、状态码等关键信息，便于排查问题和统计。
func LogLLMStream(ctx context.Context, statusCode int, usage *provider.Usage, cost time.Duration, err error) {
	var success bool
	var errMsg string
	if err != nil {
		success = false
		errMsg = err.Error()
	} else {
		success = true
	}

	promptTokens, completionTokens, totalTokens := 0, 0, 0
	if usage != nil {
		promptTokens = usage.PromptTokens
		completionTokens = usage.CompletionTokens
		totalTokens = usage.TotalTokens
	}

	iface := common.GetInterface(ctx)
	var lvl slog.Level
	switch {
	case statusCode >= 500:
		lvl = slog.LevelError
	case statusCode >= 400:
		lvl = slog.LevelWarn
	default:
		lvl = slog.LevelInfo
	}
	attrs := baseAttrs(ctx, nil, lvl, iface, "LLM stream done")
	attrs = append(attrs,
		"context.status_code", statusCode,
		"context.prompt_tokens", promptTokens,
		"context.completion_tokens", completionTokens,
		"context.total_tokens", totalTokens,
		"context.cost", float64(cost.Milliseconds()),
		"context.status", success,
		"context.error", errMsg,
	)
	slog.Log(ctx, lvl, "LLM stream done", attrs...)
}

// logCallSDKAPI 记录腾讯云 SDK 调用日志（Call sdk api，component 为服务名如 cvm/tat/cls/sts）。
// action 为接口名（如 RunInstances），params 为请求参数，result 为响应摘要，success 为是否成功。
func logCallSDKAPI(ctx context.Context, component, action string, params interface{}, result interface{}, cost time.Duration, success bool, callErr error) {
	iface := common.GetInterface(ctx)
	// 根据 success 选择日志级别，baseAttrs 与实际输出级别保持一致
	lvl := slog.LevelInfo
	if !success {
		lvl = slog.LevelWarn
	}
	attrs := baseAttrs(ctx, nil, lvl, iface, "Call sdk api")

	paramsStr := ""
	if params != nil {
		if b, err := json.Marshal(params); err == nil {
			paramsStr = desensitizeBody(string(b))
		} else {
			paramsStr = fmt.Sprintf("%v", params)
		}
	}
	resultStr := ""
	if result != nil {
		if b, err := json.Marshal(result); err == nil {
			resultStr = desensitizeBody(string(b))
		} else {
			resultStr = fmt.Sprintf("%v", result)
		}
	}
	errMsg := ""
	if callErr != nil {
		errMsg = callErr.Error()
		resultStr = errMsg
	}

	attrs = append(attrs,
		"context.request.method", "POST",
		"context.request.uri", "/",
		"context.request.headers", []string{},
		"context.request.body", paramsStr,
		"context.request.interface", action,
		"context.request.region", CVMRegion,
		"context.response.status_code", 0,
		"context.response.headers", []string{},
		"context.response.body", resultStr,
		"context.cost", float64(cost.Milliseconds()),
		"context.component", component,
		"context.status", success,
	)

	if success {
		slog.InfoContext(ctx, "Call sdk api", attrs...)
	} else {
		// 失败时升级为 WARN，便于 CLS 告警
		slog.WarnContext(ctx, "Call sdk api", attrs...)
	}
}

// CallSDKAPITyped 是类型安全的 SDK 调用包装，无需手写 action 字符串，也无需类型断言。
// action 自动从返回类型名推导：*cvm.DescribeInstancesResponse → "DescribeInstances"。
// req 作为请求参数自动记录到日志，fn 直接传方法引用即可。
//
// 用法示例：
//
//	resp, err := CallSDKAPITyped(ctx, SDKComponentCVM, req, client.DescribeInstances)
//	// resp 直接是 *cvm.DescribeInstancesResponse，无需类型断言，req 自动打印到日志
func CallSDKAPITyped[Req any, Resp any](ctx context.Context, component string, req *Req, fn func(*Req) (*Resp, error)) (*Resp, error) {
	start := time.Now()
	result, err := fn(req)
	// 从返回类型名自动推导 action：去掉 "Response" 后缀
	typeName := reflect.TypeOf((*Resp)(nil)).Elem().Name()
	action := strings.TrimSuffix(typeName, "Response")
	logCallSDKAPI(ctx, component, action, req, result, time.Since(start), err == nil, err)
	return result, err
}

// CallSDKAPICommon 是针对 CommonClient（无 SDK 封装接口）的调用包装，对齐 CallSDKAPITyped 风格。
// 自动创建 CommonResponse、计时、执行 Send、记录日志，调用侧无需手写闭包。
// action 自动从 req.GetAction() 获取，无需手动传入。
//
// 用法示例：
//
//	req := tchttp.NewCommonRequest("cls", "2020-10-16", "GetClsService")
//	req.SetActionParameters("{}")
//	resp, err := CallSDKAPICommon(ctx, SDKComponentCLS, req, client.Send)
//	// resp 直接是 *tchttp.CommonResponse，可直接调用 resp.GetBody()
func CallSDKAPICommon(ctx context.Context, component string, req *tchttp.CommonRequest, send func(tchttp.Request, tchttp.Response) error) (*tchttp.CommonResponse, error) {
	resp := tchttp.NewCommonResponse()
	start := time.Now()
	err := send(req, resp)
	var result interface{}
	if err == nil {
		result = map[string]interface{}{"body": string(resp.GetBody())}
	}
	logCallSDKAPI(ctx, component, req.GetAction(), req, result, time.Since(start), err == nil, err)
	return resp, err
}

// logCallDBAPI 记录数据库操作日志（Call db api），由 RegisterDBLogger 的回调自动调用。
func logCallDBAPI(ctx context.Context, operation, table, query string, vars []interface{}, rowsAffected int64, cost time.Duration, success bool, callErr error) {
	iface := common.GetInterface(ctx)
	// 根据 success 选择日志级别，baseAttrs 与实际输出级别保持一致
	lvl := slog.LevelInfo
	if !success {
		lvl = slog.LevelWarn
	}
	attrs := baseAttrs(ctx, nil, lvl, iface, "Call db api")

	errMsg := ""
	if callErr != nil {
		errMsg = callErr.Error()
	}

	attrs = append(attrs,
		"context.request.operation", operation,
		"context.request.table", table,
		"context.request.query", desensitizeBody(query),
		"context.request.vars", vars,
		"context.response.rows_affected", rowsAffected,
		"context.response.body", errMsg,
		"context.cost", float64(cost.Milliseconds()),
		"context.component", "sqlite",
		"context.status", success,
	)

	if success {
		slog.InfoContext(ctx, "Call db api", attrs...)
	} else {
		// 失败时升级为 WARN
		slog.WarnContext(ctx, "Call db api", attrs...)
	}
}

// dbStartTimeKey 是存储在 GORM Statement.Settings 中的计时起点 key。
const dbStartTimeKey = "_log_start_time"

// Logger 返回一个预置了当前请求链路字段（request_id、trace_id、uin、subuin、interface）的 *slog.Logger。
// 业务代码直接调用 Logger(ctx).Info("xxx", "key", val) 即可串联 trace 信息，无需手写 slog.InfoContext。
func Logger(ctx context.Context) *slog.Logger {
	l := slog.Default().With(
		"request_id", common.GetRequestID(ctx),
		"trace_id", common.GetTraceID(ctx),
		"uin", common.CVMUinFromCtx(ctx),
		"subuin", common.GetSubUin(ctx),
		"interface", common.GetInterface(ctx),
	)
	if common.IsTask(ctx) {
		l = l.With("crontask", true)
	}
	return l
}

// RegisterDBLogger 向 db 注册 Before/After 回调，自动记录所有数据库操作日志。
// 在 main.go 中 model.InitDB 之后调用一次即可，业务代码无需任何改动。
// 业务代码通过 db.WithContext(ctx) 传递 request_id 等链路信息。
func RegisterDBLogger(db *gorm.DB) {
	before := func(tx *gorm.DB) {
		tx.Statement.Settings.Store(dbStartTimeKey, time.Now())
	}
	after := func(opName string) func(*gorm.DB) {
		return func(tx *gorm.DB) {
			var cost time.Duration
			if v, ok := tx.Statement.Settings.Load(dbStartTimeKey); ok {
				cost = time.Since(v.(time.Time))
				tx.Statement.Settings.Delete(dbStartTimeKey)
			}
			ctx := tx.Statement.Context
			logCallDBAPI(ctx, opName, tx.Statement.Table, tx.Statement.SQL.String(), tx.Statement.Vars, tx.RowsAffected, cost, tx.Error == nil, tx.Error)
		}
	}

	cb := db.Callback()
	cb.Query().Before("gorm:query").Register("access_log:before_query", before)
	cb.Query().After("gorm:query").Register("access_log:after_query", after("SELECT"))
	cb.Create().Before("gorm:create").Register("access_log:before_create", before)
	cb.Create().After("gorm:create").Register("access_log:after_create", after("INSERT"))
	cb.Update().Before("gorm:update").Register("access_log:before_update", before)
	cb.Update().After("gorm:update").Register("access_log:after_update", after("UPDATE"))
	cb.Delete().Before("gorm:delete").Register("access_log:before_delete", before)
	cb.Delete().After("gorm:delete").Register("access_log:after_delete", after("DELETE"))
	cb.Row().Before("gorm:row").Register("access_log:before_row", before)
	cb.Row().After("gorm:row").Register("access_log:after_row", after("ROW"))
	cb.Raw().Before("gorm:raw").Register("access_log:before_raw", before)
	cb.Raw().After("gorm:raw").Register("access_log:after_raw", after("RAW"))
}
