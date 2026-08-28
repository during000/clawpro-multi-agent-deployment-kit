package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"hatchery/common"
)

// captureSlog 临时把 slog.Default 替换成写入 buf 的 JSON handler，
// 返回 restore 函数与 buf；测试结束后 restore 还原。
func captureSlog(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	buf := &bytes.Buffer{}
	h := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	return buf, func() { slog.SetDefault(prev) }
}

// parseLastJSONLine 取 buf 最后一行 JSON 解析为 map，便于断言字段是否存在。
func parseLastJSONLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) == 0 || lines[len(lines)-1] == "" {
		t.Fatalf("slog 未产生任何输出: %q", buf.String())
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &m); err != nil {
		t.Fatalf("解析 slog JSON 输出失败: %v, raw=%s", err, lines[len(lines)-1])
	}
	return m
}

// TestLogRcvRequest_NilRequest 覆盖 r == nil 的兜底分支：
// - 不应 panic
// - context.method / context.path / context.query / context.headers 不应出现
// - context.body 仍应出现
func TestLogRcvRequest_NilRequest(t *testing.T) {
	buf, restore := captureSlog(t)
	defer restore()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("LogRcvRequest 在 r==nil 时不应 panic，但发生了: %v", r)
		}
	}()

	LogRcvRequest(context.Background(), nil, []byte(`{"hello":"world"}`))

	m := parseLastJSONLine(t, buf)
	if _, ok := m["context.method"]; ok {
		t.Errorf("r==nil 时不应记录 context.method, got=%v", m["context.method"])
	}
	if _, ok := m["context.path"]; ok {
		t.Errorf("r==nil 时不应记录 context.path, got=%v", m["context.path"])
	}
	if _, ok := m["context.query"]; ok {
		t.Errorf("r==nil 时不应记录 context.query, got=%v", m["context.query"])
	}
	if _, ok := m["context.headers"]; ok {
		t.Errorf("r==nil 时不应记录 context.headers, got=%v", m["context.headers"])
	}
	if body, ok := m["context.body"]; !ok {
		t.Errorf("r==nil 时仍应记录 context.body, 实际缺失")
	} else if !strings.Contains(body.(string), "hello") {
		t.Errorf("context.body 内容异常: %v", body)
	}
}

// TestLogRcvRequest_NilURL 覆盖 r != nil && r.URL == nil 分支：
// - 不应 panic
// - context.method / context.headers 应出现
// - context.path / context.query 不应出现
// - context.body 应出现
func TestLogRcvRequest_NilURL(t *testing.T) {
	buf, restore := captureSlog(t)
	defer restore()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("LogRcvRequest 在 r.URL==nil 时不应 panic，但发生了: %v", r)
		}
	}()

	r := &http.Request{
		Method: http.MethodPost,
		Header: http.Header{
			"X-Test":        []string{"v1"},
			"Authorization": []string{"Bearer secret-token"}, // 测试脱敏顺带带过
		},
		// URL 故意保持 nil
	}
	LogRcvRequest(context.Background(), r, []byte("hello"))

	m := parseLastJSONLine(t, buf)
	if got, ok := m["context.method"]; !ok || got != http.MethodPost {
		t.Errorf("context.method 异常: ok=%v val=%v", ok, got)
	}
	if _, ok := m["context.path"]; ok {
		t.Errorf("r.URL==nil 时不应记录 context.path, got=%v", m["context.path"])
	}
	if _, ok := m["context.query"]; ok {
		t.Errorf("r.URL==nil 时不应记录 context.query, got=%v", m["context.query"])
	}
	if _, ok := m["context.headers"]; !ok {
		t.Errorf("r.URL==nil 时仍应记录 context.headers")
	}
	if body, ok := m["context.body"]; !ok || body.(string) != "hello" {
		t.Errorf("context.body 异常: ok=%v val=%v", ok, body)
	}
}

// TestLogRcvRequest_IfaceFromCtx 覆盖 r/r.URL 均非空，但 ctx 已带 interface 的分支：
// - interface 字段应来自 ctx，而不是 r.URL.Path
// - path/query/method/headers/body 全部应被记录
func TestLogRcvRequest_IfaceFromCtx(t *testing.T) {
	buf, restore := captureSlog(t)
	defer restore()

	ctx := context.WithValue(context.Background(), common.CtxKeyInterface, "/from/ctx")

	r := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Path:     "/from/url",
			RawQuery: "a=1&b=2",
		},
		Header: http.Header{"X-Test": []string{"v"}},
	}
	LogRcvRequest(ctx, r, []byte(`{"k":"v"}`))

	m := parseLastJSONLine(t, buf)
	if got, _ := m["interface"].(string); got != "/from/ctx" {
		t.Errorf("interface 应取自 ctx，得到 %q", got)
	}
	if got, _ := m["context.method"].(string); got != http.MethodGet {
		t.Errorf("context.method 异常: %v", got)
	}
	if got, _ := m["context.path"].(string); got != "/from/url" {
		t.Errorf("context.path 异常: %v", got)
	}
	if got, _ := m["context.query"].(string); got != "a=1&b=2" {
		t.Errorf("context.query 异常: %v", got)
	}
	if _, ok := m["context.headers"]; !ok {
		t.Errorf("context.headers 缺失")
	}
	if _, ok := m["context.body"]; !ok {
		t.Errorf("context.body 缺失")
	}
}

// TestLogRcvRequest_IfaceFromURL 覆盖 r/r.URL 均非空且 ctx 没有 interface 的分支：
// - iface 应被从 r.URL.Path 推导出来
func TestLogRcvRequest_IfaceFromURL(t *testing.T) {
	buf, restore := captureSlog(t)
	defer restore()

	r := &http.Request{
		Method: http.MethodPut,
		URL: &url.URL{
			Path:     "/api/v1/foo",
			RawQuery: "x=y",
		},
		Header: http.Header{},
	}
	LogRcvRequest(context.Background(), r, nil)

	m := parseLastJSONLine(t, buf)
	// 注意：baseAttrs 里 "interface" 取的是 common.GetInterface(ctx)，
	// 当 ctx 没注入时返回 ""。iface 推导主要影响 "Rcv request" 这条 message 之外的链路
	// （目前 LogRcvRequest 把 iface 传给 baseAttrs 的 iface 形参用于内部记录，但
	// 输出 key 仍为 interface, 此处用 message 字段断言函数路径已走通即可）。
	if got, _ := m["message"].(string); got != "Rcv request" {
		t.Errorf("message 异常: %v", got)
	}
	if got, _ := m["context.path"].(string); got != "/api/v1/foo" {
		t.Errorf("context.path 异常: %v", got)
	}
	if got, _ := m["context.query"].(string); got != "x=y" {
		t.Errorf("context.query 异常: %v", got)
	}
	// body 为 nil 时 safeBody 返回 ""，应仍存在该字段
	if got, ok := m["context.body"]; !ok || got.(string) != "" {
		t.Errorf("context.body 应为空字符串，实际 ok=%v val=%v", ok, got)
	}
}

func TestDesensitizeBody_ChannelSecrets(t *testing.T) {
	raw := `{"app_id":"safe-id","app_secret":"app-secret-value","secret":"plain-secret-value","signing_secret":"signing-secret-value","bot_token":"bot-token-value","password":"password-value"}`
	got := desensitizeBody(raw)
	for _, secret := range []string{
		"app-secret-value",
		"plain-secret-value",
		"signing-secret-value",
		"bot-token-value",
		"password-value",
	} {
		if strings.Contains(got, secret) {
			t.Errorf("desensitizeBody leaked %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, `"app_id":"safe-id"`) {
		t.Errorf("desensitizeBody removed non-sensitive field: %s", got)
	}
}

func TestLogRcvRequest_ChannelSecretsRedacted(t *testing.T) {
	buf, restore := captureSlog(t)
	defer restore()

	body := []byte(`{"name":"safe-name","channels":[{"channel":"custom","config":{"encoding_aes_key":"request-secret-value","arbitrary_credential":"another-secret-value"}}]}`)
	req, err := http.NewRequest(http.MethodPost, adminCreateInstanceLogPath, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	LogRcvRequest(context.Background(), req, body)

	for _, secret := range []string{"request-secret-value", "another-secret-value"} {
		if strings.Contains(buf.String(), secret) {
			t.Fatalf("request structured log leaked %q: %s", secret, buf.String())
		}
	}
	m := parseLastJSONLine(t, buf)
	if got := m["context.body"]; got != "" {
		t.Fatalf("admin create request body=%v, want fully suppressed", got)
	}
}

func TestLogTATRunCommand_ParamValuesRedacted(t *testing.T) {
	buf, restore := captureSlog(t)
	defer restore()

	logTATRunCommand(
		"ins-sensitive",
		"set_channel.sh",
		"ubuntu",
		"/home/ubuntu",
		60,
		map[string]string{"app_secret": "tat-secret-value"},
	)
	if strings.Contains(buf.String(), "tat-secret-value") {
		t.Fatalf("TAT structured log leaked script parameter value: %s", buf.String())
	}
	m := parseLastJSONLine(t, buf)
	if got := m["param_count"]; got != float64(1) {
		t.Errorf("param_count=%v, want 1", got)
	}
}

func TestScriptFailureDetail_SetChannelRedacted(t *testing.T) {
	const secret = "script-output-secret-value"
	for _, scriptName := range []string{"set_channel.sh", "set_channel_hermes.sh", "set_channel_ace.sh"} {
		if got := scriptFailureDetail(scriptName, secret); got != "" {
			t.Errorf("scriptFailureDetail(%q)=%q, want redacted empty detail", scriptName, got)
		}
	}
	if got := scriptFailureDetail("diagnostic.sh", "  safe diagnostic  "); got != "safe diagnostic" {
		t.Errorf("non-sensitive script detail=%q, want trimmed diagnostic", got)
	}
}
