package controller

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
)

// instanceWriter 包装 http.ResponseWriter，携带实例 ID。
// writeError 会自动从中提取 instance_id，无需通过 request context 传递。
type instanceWriter struct {
	http.ResponseWriter
	instanceId string
}

// Flush 实现 http.Flusher 接口，确保 SSE 流式输出不受影响。
func (w *instanceWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap 返回底层 ResponseWriter，兼容 http.ResponseController 等标准库机制。
func (w *instanceWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// Hijack 透传底层 ResponseWriter 的 Hijack 接口，支持 WebSocket 连接升级。
// 修复 HandleBrowserVNCProxy 中 getInstanceByID 将 w 包装为 instanceWriter 后，
// w.(http.Hijacker) 断言失败导致 500 "服务器不支持连接升级" 的问题。
func (w *instanceWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, hcommon.I18nError(i18n.MsgHijackNotSupported)
}

// WrapInstanceId 将实例 ID 绑定到 ResponseWriter 上，返回包装后的 writer。
// 在获取到 instance 后调用，writeError 会自动从 writer 中提取 instance_id。
func WrapInstanceId(w http.ResponseWriter, instanceId string) http.ResponseWriter {
	return &instanceWriter{ResponseWriter: w, instanceId: instanceId}
}

// instanceIdFromWriter 从 ResponseWriter 中提取实例 ID，不存在则返回空。
func instanceIdFromWriter(w http.ResponseWriter) string {
	if iw, ok := w.(*instanceWriter); ok {
		return iw.instanceId
	}
	return ""
}

// jsonOK 返回成功 JSON
func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// jsonAPI 标记当前 handler 为纯 JSON API，确保 writeError 输出 JSON 而非 HTML。
// 在 handler 入口处调用。
func jsonAPI(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func requireJSONContentType(w http.ResponseWriter, r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, r, http.StatusUnsupportedMediaType, hcommon.I18nError(i18n.MsgContentTypeApplicationJSONRequired))
		return false
	}
	return true
}

var defaultErrorMsg = map[int]*hcommon.RichError{
	http.StatusBadRequest:          hcommon.I18nError(i18n.MsgBadRequest),
	http.StatusUnauthorized:        hcommon.I18nError(i18n.MsgUnauthorized),
	http.StatusForbidden:           hcommon.I18nError(i18n.MsgForbidden),
	http.StatusNotFound:            hcommon.I18nError(i18n.MsgNotFound),
	http.StatusInternalServerError: hcommon.I18nError(i18n.MsgInternalError),
}

func defaultError(status int) *hcommon.RichError {
	if err, ok := defaultErrorMsg[status]; ok {
		return err
	}
	return hcommon.I18nError(i18n.MsgUnknownError)
}

// writeError 统一错误返回
func writeError(w http.ResponseWriter, r *http.Request, status int, err *hcommon.RichError) {
	if err == nil {
		err = defaultError(status)
	}

	msg := err.ErrorMessage(r.Context())
	detail := hcommon.ErrorDetailWithCtx(r.Context(), err)
	reqId := hcommon.ErrorRequestId(err)
	// 业务请求 ID：优先从 RichError 中获取，其次从 request context 中获取
	bizReqId := hcommon.ErrorBizRequestId(err)
	if bizReqId == "" {
		bizReqId = hcommon.GetRequestID(r.Context())
	}
	// 优先从 error 中提取 instanceId（RichError），其次从 ResponseWriter 包装中提取
	instanceId := hcommon.ErrorInstanceId(err)
	if instanceId == "" {
		instanceId = instanceIdFromWriter(w)
	}
	// 记录错误日志：包含调用方的文件:行号，便于排查 writeError 的外部触发点
	logWriteError(r, status, err, msg, detail, reqId, bizReqId, instanceId)
	w.Header().Set("X-Audit-Failed", "1")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := err.CustomData
	if resp == nil {
		resp = make(map[string]any)
	}

	resp["error"] = msg

	if detail != "" {
		resp["detail"] = detail
	}
	if reqId != "" {
		resp["request_id"] = reqId
	}
	if bizReqId != "" {
		resp["biz_request_id"] = bizReqId
	}
	if instanceId != "" {
		resp["instance_id"] = instanceId
	}

	json.NewEncoder(w).Encode(resp)
}

// jsonRedirect 返回重定向 JSON
func jsonRedirect(w http.ResponseWriter, url string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "redirect": url})
}

// parsePagination 从请求中解析 page 和 page_size 参数。
// page 默认为 1，page_size 默认为 20，最大默认 100。
// 可选参数: params[0] = maxPageSize, params[1] = defaultPageSize。
func parsePagination(r *http.Request, params ...int) (page, pageSize int) {
	maxPS := 100
	defaultPS := 20
	if len(params) > 0 && params[0] > 0 {
		maxPS = params[0]
	}
	if len(params) > 1 && params[1] > 0 {
		defaultPS = params[1]
	}
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 {
		pageSize = defaultPS
	}
	if pageSize > maxPS {
		pageSize = maxPS
	}
	return
}

// logWriteError 记录 writeError 触发的错误日志，包含外部调用方的文件:行号。
// 通过 runtime.Caller(2) 跳过本函数与 writeError 自身，定位到真正调用 writeError 的业务代码位置。
func logWriteError(r *http.Request, status int, err error, msg, detail, reqId, bizReqId, instanceId string) {
	caller := "unknown"
	if _, file, line, ok := runtime.Caller(2); ok {
		// 截短为相对项目的短路径，便于阅读：保留最后两段（pkg/file.go）
		short := file
		if idx := strings.LastIndex(file, "/hatchery/"); idx >= 0 {
			short = file[idx+len("/hatchery/"):]
		} else if idx := strings.LastIndex(file, "/"); idx >= 0 {
			if idx2 := strings.LastIndex(file[:idx], "/"); idx2 >= 0 {
				short = file[idx2+1:]
			} else {
				short = file[idx+1:]
			}
		}
		caller = fmt.Sprintf("%s:%d", short, line)
	}

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	var ctx = r.Context()
	log := Logger(ctx)
	attrs := []any{
		"caller", caller,
		"status", status,
		"error", errStr,
		"message", msg,
		"detail", detail,
		"path", r.URL.Path,
		"method", r.Method,
	}
	if reqId != "" {
		attrs = append(attrs, "upstream_request_id", reqId)
	}
	if bizReqId != "" {
		attrs = append(attrs, "biz_request_id", bizReqId)
	}
	if instanceId != "" {
		attrs = append(attrs, "instance_id", instanceId)
	}

	// 5xx → ERROR，其余（含 4xx）→ WARN，与 LogSendResponse 的级别推导一致
	if status >= 500 {
		log.LogAttrs(ctx, slog.LevelError, "[writeError] 接口错误响应", toLogAttrs(attrs)...)
	} else {
		log.LogAttrs(ctx, slog.LevelWarn, "[writeError] 接口错误响应", toLogAttrs(attrs)...)
	}
}

// toLogAttrs 将 key/value 交替的 []any 转换为 []slog.Attr。
func toLogAttrs(kv []any) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		k, _ := kv[i].(string)
		attrs = append(attrs, slog.Any(k, kv[i+1]))
	}
	return attrs
}
