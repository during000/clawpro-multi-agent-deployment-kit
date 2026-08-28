package controller

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestSMHInternalDomainTransport_InjectsOnMultipartPath 验证：当 ctx 标记了
// "使用内网域名" 且请求是 MultipartUploadFile 时，transport 会在 query 上追加
// internal_domain=1。
func TestSMHInternalDomainTransport_InjectsOnMultipartPath(t *testing.T) {
	var gotURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	client := &http.Client{
		Transport: &smhInternalDomainTransport{base: http.DefaultTransport},
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/file/lib/space/foo?multipart=1", nil)
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	req = req.WithContext(WithSMHInternalDomain(context.Background(), true))

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	resp.Body.Close()

	if gotURL == nil {
		t.Fatal("服务端未收到请求")
	}
	if got := gotURL.Query().Get("internal_domain"); got != "1" {
		t.Errorf("应注入 internal_domain=1，实际 query=%q", gotURL.RawQuery)
	}
}

// TestSMHInternalDomainTransport_NoInjectWithoutCtxMark 验证：未标记内网时
// 即使路径匹配也不应注入参数。
func TestSMHInternalDomainTransport_NoInjectWithoutCtxMark(t *testing.T) {
	var gotURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	client := &http.Client{
		Transport: &smhInternalDomainTransport{base: http.DefaultTransport},
	}

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/file/lib/space/foo?multipart=1", nil)
	// 不调用 WithSMHInternalDomain
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	resp.Body.Close()

	if gotURL == nil {
		t.Fatal("服务端未收到请求")
	}
	if _, ok := gotURL.Query()["internal_domain"]; ok {
		t.Errorf("未标记内网时不应注入 internal_domain，实际 query=%q", gotURL.RawQuery)
	}
}

// TestSMHInternalDomainTransport_NoInjectOnUnrelatedPath 验证：与 SMH 上传/续期
// 无关的路径（如 Token 获取、目录管理）即使 ctx 标记了内网也不应被注入。
func TestSMHInternalDomainTransport_NoInjectOnUnrelatedPath(t *testing.T) {
	var gotURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{
		Transport: &smhInternalDomainTransport{base: http.DefaultTransport},
	}

	// 走 /api/v1/token，与上传无关
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/token/foo", nil)
	req = req.WithContext(WithSMHInternalDomain(context.Background(), true))
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	resp.Body.Close()

	if _, ok := gotURL.Query()["internal_domain"]; ok {
		t.Errorf("非上传路径不应注入 internal_domain，实际 query=%q", gotURL.RawQuery)
	}
}

// TestSMHInternalDomainTransport_InjectsOnRenewPath 验证：续期分块上传 API
// (/api/v1/file/.../?renew=1) 同样会被识别并注入。
func TestSMHInternalDomainTransport_InjectsOnRenewPath(t *testing.T) {
	var gotURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{
		Transport: &smhInternalDomainTransport{base: http.DefaultTransport},
	}

	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/file/lib/space/ckey?renew=1", nil)
	req = req.WithContext(WithSMHInternalDomain(context.Background(), true))
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	resp.Body.Close()

	if got := gotURL.Query().Get("internal_domain"); got != "1" {
		t.Errorf("renew 路径应注入 internal_domain=1，实际 query=%q", gotURL.RawQuery)
	}
}

// TestProbeSMHInternalReachable_OK 验证：探测一个能正常监听的本地端口时应返回 true。
func TestProbeSMHInternalReachable_OK(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	defer ln.Close()

	url := "https://" + ln.Addr().String() + "/foo?partNumber=1"
	if !ProbeSMHInternalReachable(context.Background(), url) {
		t.Error("可达端口应探测成功")
	}
}

// TestProbeSMHInternalReachable_Fail 验证：探测一个不存在的端口时应快速返回 false。
func TestProbeSMHInternalReachable_Fail(t *testing.T) {
	// 选一个大概率没在监听的高位端口
	urlStr := "https://127.0.0.1:1/foo"
	start := time.Now()
	if ProbeSMHInternalReachable(context.Background(), urlStr) {
		t.Error("不可达端口应探测失败")
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("探测耗时过长：%v，应在 smhInternalProbeTimeout(%v) 量级", d, smhInternalProbeTimeout)
	}
}

// TestProbeSMHInternalReachable_BadInput 验证：空串或非法 URL 直接返回 false。
func TestProbeSMHInternalReachable_BadInput(t *testing.T) {
	if ProbeSMHInternalReachable(context.Background(), "") {
		t.Error("空 URL 应返回 false")
	}
	if ProbeSMHInternalReachable(context.Background(), "://not a url") {
		t.Error("非法 URL 应返回 false")
	}
}

// TestWithSMHInternalDomain_ToggleOff 验证：on=false 时 smhInternalDomainEnabled 返回 false。
func TestWithSMHInternalDomain_ToggleOff(t *testing.T) {
	ctx := WithSMHInternalDomain(context.Background(), true)
	if !smhInternalDomainEnabled(ctx) {
		t.Fatal("on=true 时应启用")
	}
	ctx = WithSMHInternalDomain(ctx, false)
	if smhInternalDomainEnabled(ctx) {
		t.Error("on=false 时应禁用")
	}
}

// TestShouldInjectInternalDomain_RequiresMultipartOrRenewFlag 验证：只有带
// multipart 或 renew query flag 的请求才会被注入，避免误伤其它 /api/v1/file 类操作。
func TestShouldInjectInternalDomain_RequiresMultipartOrRenewFlag(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://x.example.com/api/v1/file/lib/space/foo", nil)
	req = req.WithContext(WithSMHInternalDomain(context.Background(), true))
	if shouldInjectInternalDomain(req) {
		t.Error("无 multipart/renew 标记时不应注入")
	}

	req2, _ := http.NewRequest(http.MethodPost, "https://x.example.com/api/v1/file/lib/space/foo?multipart=1", nil)
	req2 = req2.WithContext(WithSMHInternalDomain(context.Background(), true))
	if !shouldInjectInternalDomain(req2) {
		t.Error("multipart=1 时应注入")
	}
}

// 静态检查：strings 包确实被用到，防止 lint 在没引用时报错（保留以便未来扩展）。
var _ = strings.Contains
