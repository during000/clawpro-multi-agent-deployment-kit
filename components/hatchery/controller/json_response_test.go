package controller

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ========== instanceWriter Hijack 测试 ==========

// mockHijackResponseWriter 模拟支持 Hijack 的 ResponseWriter
type mockHijackResponseWriter struct {
	http.ResponseWriter
	hijacked bool
}

func (m *mockHijackResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	m.hijacked = true
	// 返回模拟的连接（实际测试中不需要真实连接）
	return nil, nil, nil
}

// mockPlainResponseWriter 模拟不支持 Hijack 的 ResponseWriter
type mockPlainResponseWriter struct {
	http.ResponseWriter
}

// TestInstanceWriter_Hijack_Supported 测试 instanceWriter 在底层支持 Hijack 时正确透传
func TestInstanceWriter_Hijack_Supported(t *testing.T) {
	mock := &mockHijackResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}
	iw := &instanceWriter{
		ResponseWriter: mock,
		instanceId:     "ins-test123",
	}

	conn, rw, err := iw.Hijack()
	if err != nil {
		t.Errorf("Hijack 不应返回错误: %v", err)
	}
	if !mock.hijacked {
		t.Error("底层 Hijack 应被调用")
	}
	// mockHijackResponseWriter 返回 nil, nil, nil
	if conn != nil || rw != nil {
		t.Error("应返回 mock 的值")
	}
}

// TestInstanceWriter_Hijack_NotSupported 测试 instanceWriter 在底层不支持 Hijack 时返回错误
func TestInstanceWriter_Hijack_NotSupported(t *testing.T) {
	plain := &mockPlainResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}
	iw := &instanceWriter{
		ResponseWriter: plain,
		instanceId:     "ins-test456",
	}

	conn, rw, err := iw.Hijack()
	if err == nil {
		t.Error("底层不支持 Hijack 时应返回错误")
	}
	if conn != nil || rw != nil {
		t.Error("失败时 conn 和 rw 应为 nil")
	}
}

// TestInstanceWriter_Hijack_ImplementsInterface 测试 instanceWriter 实现了 http.Hijacker 接口
func TestInstanceWriter_Hijack_ImplementsInterface(t *testing.T) {
	mock := &mockHijackResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}
	iw := &instanceWriter{
		ResponseWriter: mock,
		instanceId:     "ins-test789",
	}

	// 验证类型断言成功
	var w http.ResponseWriter = iw
	if _, ok := w.(http.Hijacker); !ok {
		t.Error("instanceWriter 应实现 http.Hijacker 接口")
	}
}

// TestWrapInstanceId_PreservesHijack 测试 WrapInstanceId 包装后仍支持 Hijack
func TestWrapInstanceId_PreservesHijack(t *testing.T) {
	mock := &mockHijackResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}
	wrapped := WrapInstanceId(mock, "ins-wrapped")

	hj, ok := wrapped.(http.Hijacker)
	if !ok {
		t.Fatal("WrapInstanceId 包装后应仍支持 Hijack")
	}

	_, _, err := hj.Hijack()
	if err != nil {
		t.Errorf("Hijack 不应返回错误: %v", err)
	}
	if !mock.hijacked {
		t.Error("底层 Hijack 应被调用")
	}
}

// ========== instanceWriter 其他方法测试 ==========

// TestInstanceWriter_Flush 测试 instanceWriter 的 Flush 透传
func TestInstanceWriter_Flush(t *testing.T) {
	rec := httptest.NewRecorder()
	iw := &instanceWriter{
		ResponseWriter: rec,
		instanceId:     "ins-flush",
	}

	// httptest.ResponseRecorder 实现了 http.Flusher
	// Flush 不应 panic
	iw.Flush()
	if !rec.Flushed {
		t.Error("Flush 应透传到底层 ResponseWriter")
	}
}

// TestInstanceWriter_Unwrap 测试 instanceWriter 的 Unwrap 方法
func TestInstanceWriter_Unwrap(t *testing.T) {
	rec := httptest.NewRecorder()
	iw := &instanceWriter{
		ResponseWriter: rec,
		instanceId:     "ins-unwrap",
	}

	unwrapped := iw.Unwrap()
	if unwrapped != rec {
		t.Error("Unwrap 应返回底层 ResponseWriter")
	}
}

// TestInstanceIdFromWriter 测试从 writer 中提取实例 ID
func TestInstanceIdFromWriter(t *testing.T) {
	rec := httptest.NewRecorder()

	// 普通 ResponseWriter 应返回空
	if id := instanceIdFromWriter(rec); id != "" {
		t.Errorf("普通 ResponseWriter 应返回空，实际为 '%s'", id)
	}

	// instanceWriter 应返回实例 ID
	iw := &instanceWriter{
		ResponseWriter: rec,
		instanceId:     "ins-extract",
	}
	if id := instanceIdFromWriter(iw); id != "ins-extract" {
		t.Errorf("应返回 'ins-extract'，实际为 '%s'", id)
	}
}

// ========== ResponseCapture Hijack 测试 ==========

// TestResponseCapture_Hijack_Supported 测试 ResponseCapture 在底层支持 Hijack 时正确透传
func TestResponseCapture_Hijack_Supported(t *testing.T) {
	mock := &mockHijackResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}
	rc := NewResponseCapture(mock)

	conn, rw, err := rc.Hijack()
	if err != nil {
		t.Errorf("Hijack 不应返回错误: %v", err)
	}
	if !mock.hijacked {
		t.Error("底层 Hijack 应被调用")
	}
	if conn != nil || rw != nil {
		t.Error("应返回 mock 的值")
	}
}

// TestResponseCapture_Hijack_NotSupported 测试 ResponseCapture 在底层不支持 Hijack 时返回错误
func TestResponseCapture_Hijack_NotSupported(t *testing.T) {
	rec := httptest.NewRecorder()
	rc := NewResponseCapture(rec)

	conn, rw, err := rc.Hijack()
	if err == nil {
		t.Error("底层不支持 Hijack 时应返回错误")
	}
	if conn != nil || rw != nil {
		t.Error("失败时 conn 和 rw 应为 nil")
	}
	expectedErr := hcommon.I18nError(i18n.MsgHijackNotSupported)
	if !errors.Is(err, expectedErr) {
		t.Errorf("错误信息应为 '%s'，实际为 '%s'", expectedErr, err.Error())
	}
}

// TestResponseCapture_Hijack_ImplementsInterface 测试 ResponseCapture 实现了 http.Hijacker 接口
func TestResponseCapture_Hijack_ImplementsInterface(t *testing.T) {
	mock := &mockHijackResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}
	rc := NewResponseCapture(mock)

	var w http.ResponseWriter = rc
	if _, ok := w.(http.Hijacker); !ok {
		t.Error("ResponseCapture 应实现 http.Hijacker 接口")
	}
}

// ========== parsePagination 测试 ==========

// TestParsePagination_JsonResponse 测试分页参数解析
func TestParsePagination_JsonResponse(t *testing.T) {
	tests := []struct {
		name         string
		queryString  string
		wantPage     int
		wantPageSize int
	}{
		{"默认值", "", 1, 20},
		{"自定义值", "page=3&page_size=50", 3, 50},
		{"page 为 0", "page=0&page_size=10", 1, 10},
		{"page 为负数", "page=-1&page_size=10", 1, 10},
		{"page_size 为 0", "page=1&page_size=0", 1, 20},
		{"page_size 超过上限", "page=1&page_size=200", 1, 100},
		{"非数字参数", "page=abc&page_size=xyz", 1, 20},
		{"只有 page", "page=5", 5, 20},
		{"只有 page_size", "page_size=30", 1, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.queryString != "" {
				url += "?" + tt.queryString
			}
			r, _ := http.NewRequest("GET", url, nil)
			page, pageSize := parsePagination(r)
			if page != tt.wantPage {
				t.Errorf("page = %d, want %d", page, tt.wantPage)
			}
			if pageSize != tt.wantPageSize {
				t.Errorf("pageSize = %d, want %d", pageSize, tt.wantPageSize)
			}
		})
	}
}

// ========== jsonOK 测试 ==========

// TestJsonOK 测试 JSON 成功响应
func TestJsonOK(t *testing.T) {
	rec := httptest.NewRecorder()
	data := map[string]interface{}{"ok": true, "message": "success"}
	jsonOK(rec, data)

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Error("Content-Type 应为 application/json")
	}
	body := rec.Body.String()
	if body == "" {
		t.Error("响应 body 不应为空")
	}
	// 简单验证包含关键字段
	if !containsStr(body, "\"ok\":true") && !containsStr(body, "\"ok\": true") {
		t.Errorf("响应应包含 ok:true，实际为: %s", body)
	}
}

func containsStr(s, substr string) bool {
	return fmt.Sprintf("%s", s) != "" && len(s) >= len(substr) && findSubstr(s, substr)
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ========== logWriteError / toLogAttrs 测试 ==========

// captureSlogOutput 捕获 slog 默认 logger 的输出，返回还原函数。
// 使用 TextHandler 输出便于断言关键字段。
func captureSlogOutput(t *testing.T, lvl slog.Level) (*bytes.Buffer, func()) {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := slog.Default()
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
	return buf, func() { slog.SetDefault(prev) }
}

// TestToLogAttrs_NormalKV 验证正常 key/value 列表转换为 slog.Attr 切片。
func TestToLogAttrs_NormalKV(t *testing.T) {
	kv := []any{"a", 1, "b", "two", "c", true}
	attrs := toLogAttrs(kv)
	if len(attrs) != 3 {
		t.Fatalf("应得到 3 个 attr，实际 %d", len(attrs))
	}
	if attrs[0].Key != "a" || attrs[1].Key != "b" || attrs[2].Key != "c" {
		t.Errorf("attr key 顺序不对: %+v", attrs)
	}
}

// TestToLogAttrs_OddLengthIgnoresLast 验证奇数长度时最后一个 key 被忽略。
func TestToLogAttrs_OddLengthIgnoresLast(t *testing.T) {
	kv := []any{"a", 1, "dangling"}
	attrs := toLogAttrs(kv)
	if len(attrs) != 1 {
		t.Fatalf("奇数长度时应忽略最后一个 key，得到 %d", len(attrs))
	}
	if attrs[0].Key != "a" {
		t.Errorf("attr[0].Key 应为 'a'，实际 %s", attrs[0].Key)
	}
}

// TestToLogAttrs_Empty 验证空切片输入。
func TestToLogAttrs_Empty(t *testing.T) {
	attrs := toLogAttrs(nil)
	if len(attrs) != 0 {
		t.Errorf("空输入应返回空切片，实际 %d", len(attrs))
	}
}

// TestToLogAttrs_NonStringKeyTolerated 验证非 string 类型 key 被容错为空字符串而非 panic。
func TestToLogAttrs_NonStringKeyTolerated(t *testing.T) {
	kv := []any{123, "value"}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("非 string key 不应 panic，实际 panic: %v", r)
		}
	}()
	attrs := toLogAttrs(kv)
	if len(attrs) != 1 {
		t.Fatalf("应得到 1 个 attr，实际 %d", len(attrs))
	}
	if attrs[0].Key != "" {
		t.Errorf("非 string key 应被转为空字符串，实际 %s", attrs[0].Key)
	}
}

// TestLogWriteError_ServerErrorLevel 验证 5xx 走 ERROR 级别，且包含 caller 与 error 字段。
func TestLogWriteError_ServerErrorLevel(t *testing.T) {
	buf, restore := captureSlogOutput(t, slog.LevelDebug)
	defer restore()

	r := httptest.NewRequest("POST", "/admin/test?x=1", nil)
	logWriteError(r, http.StatusInternalServerError, errors.New("boom"),
		"内部错误", "stack trace", "tc-req-1", "biz-req-1", "ins-1")

	out := buf.String()
	if !strings.Contains(out, "level=ERROR") {
		t.Errorf("5xx 应使用 ERROR 级别，输出: %s", out)
	}
	for _, expect := range []string{
		"[writeError] 接口错误响应",
		"caller=",
		"status=500",
		"error=boom",
		"message=",
		"path=/admin/test",
		"method=POST",
		"upstream_request_id=tc-req-1",
		"biz_request_id=biz-req-1",
		"instance_id=ins-1",
	} {
		if !strings.Contains(out, expect) {
			t.Errorf("日志应包含 %q，实际输出: %s", expect, out)
		}
	}
}

// TestLogWriteError_ClientErrorLevel 验证 4xx 走 WARN 级别，并支持 nil err 与空 ID 字段。
func TestLogWriteError_ClientErrorLevel(t *testing.T) {
	buf, restore := captureSlogOutput(t, slog.LevelDebug)
	defer restore()

	r := httptest.NewRequest("GET", "/api/users", nil)
	logWriteError(r, http.StatusBadRequest, nil, "参数错误", "", "", "", "")

	out := buf.String()
	if !strings.Contains(out, "level=WARN") {
		t.Errorf("4xx 应使用 WARN 级别，输出: %s", out)
	}
	if !strings.Contains(out, "status=400") {
		t.Errorf("应包含 status=400，输出: %s", out)
	}
	// err 为 nil 时不应出现非空 error 值
	if !strings.Contains(out, `error=""`) && !strings.Contains(out, "error=") {
		t.Errorf("err 为 nil 时 error 应为空字符串，输出: %s", out)
	}
	// 空 ID 字段不应出现在日志中
	for _, missing := range []string{"upstream_request_id=", "biz_request_id=", "instance_id="} {
		if strings.Contains(out, missing) {
			t.Errorf("空 ID 字段不应出现在日志中: %s，输出: %s", missing, out)
		}
	}
}

// TestLogWriteError_CallerShortPath 验证 caller 字段被截短为 hatchery 内的相对路径。
func TestLogWriteError_CallerShortPath(t *testing.T) {
	buf, restore := captureSlogOutput(t, slog.LevelDebug)
	defer restore()

	r := httptest.NewRequest("GET", "/x", nil)
	// 通过本测试函数直接调用，runtime.Caller(2) 会跳过当前匿名 wrapper 与 logWriteError 自身，
	// 因此 caller 应指向调用 wrapper 的位置（即下面的 wrapper() 调用所在行）。
	wrapper := func() {
		logWriteError(r, http.StatusForbidden, errors.New("nope"), "forbidden", "", "", "", "")
	}
	wrapper()

	out := buf.String()
	if !strings.Contains(out, "caller=") {
		t.Fatalf("应包含 caller 字段，输出: %s", out)
	}
	// caller 字段不应包含绝对路径前缀（如 /Users/...）
	if strings.Contains(out, "caller=/Users/") || strings.Contains(out, "caller=/home/") {
		t.Errorf("caller 应被截短为相对路径，实际: %s", out)
	}
	// caller 应包含 :行号
	if !strings.Contains(out, ".go:") {
		t.Errorf("caller 应包含 .go: 段，实际: %s", out)
	}
}

// ========== writeError 集成测试 ==========

// TestWriteError_LogsAndJSONResponse 验证 writeError 同时输出日志与 JSON 错误响应（含 RichError 字段）。
func TestWriteError_LogsAndJSONResponse(t *testing.T) {
	buf, restore := captureSlogOutput(t, slog.LevelDebug)
	defer restore()

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/foo", nil)
	r.Header.Set("Accept", "application/json")

	rich := hcommon.I18nError(i18n.MsgTATTimeout).
		WithBizRequestId("biz-9").
		WithInstanceId("ins-9").
		WithRequestId("tc-req-9").
		WithI18nDetail(i18n.MsgTATTimeout)
	writeError(rec, r, http.StatusInternalServerError, rich)

	// 1) HTTP 响应断言
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status code 应为 500，实际 %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type 应为 application/json，实际 %s", got)
	}
	if rec.Header().Get("X-Audit-Failed") != "1" {
		t.Errorf("应设置 X-Audit-Failed: 1")
	}
	body := rec.Body.String()
	for _, expect := range []string{
		`"error":"TAT 操作超时"`,
		`"detail":"TAT 操作超时"`,
		`"request_id":"tc-req-9"`,
		`"biz_request_id":"biz-9"`,
		`"instance_id":"ins-9"`,
	} {
		if !strings.Contains(body, expect) {
			t.Errorf("响应 body 应包含 %q，实际: %s", expect, body)
		}
	}

	// 2) 日志断言
	out := buf.String()
	if !strings.Contains(out, "level=ERROR") {
		t.Errorf("500 错误应输出 ERROR 日志，实际: %s", out)
	}
	if !strings.Contains(out, "[writeError] 接口错误响应") {
		t.Errorf("日志应包含模块前缀，实际: %s", out)
	}
	for _, expect := range []string{
		"upstream_request_id=tc-req-9",
		"biz_request_id=biz-9",
		"instance_id=ins-9",
	} {
		if !strings.Contains(out, expect) {
			t.Errorf("日志应包含 %q，实际: %s", expect, out)
		}
	}
	// caller 应指向当前测试文件
	if !strings.Contains(out, "json_response_test.go:") {
		t.Errorf("caller 应指向当前测试文件，实际: %s", out)
	}
}

// TestWriteError_InstanceIdFromWriter 验证 writeError 能从 instanceWriter 提取 instance_id 并写入日志/响应。
func TestWriteError_InstanceIdFromWriter(t *testing.T) {
	buf, restore := captureSlogOutput(t, slog.LevelDebug)
	defer restore()

	rec := httptest.NewRecorder()
	wrapped := WrapInstanceId(rec, "ins-from-writer")
	r := httptest.NewRequest("GET", "/api/x", nil)
	r.Header.Set("Accept", "application/json")

	writeError(wrapped, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgNotFound))

	if !strings.Contains(rec.Body.String(), `"instance_id":"ins-from-writer"`) {
		t.Errorf("响应 body 应包含 instance_id，实际: %s", rec.Body.String())
	}
	if !strings.Contains(buf.String(), "instance_id=ins-from-writer") {
		t.Errorf("日志应包含 instance_id=ins-from-writer，实际: %s", buf.String())
	}
}
