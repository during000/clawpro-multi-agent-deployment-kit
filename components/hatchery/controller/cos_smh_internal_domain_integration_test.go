package controller

// 集成测试：SMH 上传"内网优先 + 不可达降级外网"完整链路
//
// 与 cos_smh_upload_test.go 中的单元用例不同，本文件用 httptest.Server 作为假 SMH，
// 同时把 hatchery 内部 RoundTripper、ctx 标记、ProbeSMHInternalReachable 串起来跑，
// 检验"端到端"行为，避免任何一处实现回归（例如 SDK 升级、参数名改写、域名格式变化）。

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// captureSMHHandler 是一个能记录所有进入请求 query 的 SMH mock handler。
// 在 SMH 假服务器和被测代码之间承担"探针"角色，便于断言每次 API 调用是否带上了
// internal_domain=1。
type captureSMHHandler struct {
	mu sync.Mutex
	// reqs 记录每次到达的请求摘要：path + query 平铺
	reqs []capturedReq
}

type capturedReq struct {
	Path           string
	Query          string
	HasMultipart   bool
	HasRenew       bool
	InternalDomain string // 实际收到的 internal_domain 值，"" 表示未传
}

func (h *captureSMHHandler) snapshot() []capturedReq {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]capturedReq, len(h.reqs))
	copy(out, h.reqs)
	return out
}

// uploadSuccessHandler 构造一个能完整跑通 PrepareSMHCommonUpload 的 SMH 假服务器：
//   - GET .../usage/...      → 200 容量充足
//   - POST .../directory/... → 201 目录创建成功
//   - POST .../file/...?multipart=1 → 201 分块上传凭证
//   - PUT  .../multipart/...?renew=1 → 200 凭证续期成功
//
// 通过 capture 把每次请求摘要写入，方便测试断言 internal_domain 注入情况。
func uploadSuccessHandler(capture *captureSMHHandler, partDomain string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 记录请求
		q := r.URL.Query()
		_, hasMulti := q["multipart"]
		_, hasRenew := q["renew"]
		capture.mu.Lock()
		capture.reqs = append(capture.reqs, capturedReq{
			Path:           r.URL.Path,
			Query:          r.URL.RawQuery,
			HasMultipart:   hasMulti,
			HasRenew:       hasRenew,
			InternalDomain: q.Get("internal_domain"),
		})
		capture.mu.Unlock()

		switch {
		case strings.Contains(r.URL.Path, "/usage/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"spaceId":"sp-common","size":"0"}]`))

		case strings.Contains(r.URL.Path, "/directory/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"path":["backups","ins-1"]}`))

		case strings.Contains(r.URL.Path, "/file/") && hasRenew:
			// 续期接口（SMH SDK 实际路径：/api/v1/file/{lib}/{space}/{confirmKey}?renew=1）
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"domain": "` + partDomain + `",
				"path": "/backups/ins-1/state.tgz",
				"uploadId": "upload-id-renew",
				"headers": {"Authorization": "renew-sign"},
				"expiration": "2099-01-01T00:00:00Z"
			}`))

		case strings.Contains(r.URL.Path, "/file/") && hasMulti:
			// 开始分块上传：返回 partDomain 作为接入域名
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{
				"domain": "` + partDomain + `",
				"path": "/backups/ins-1/state.tgz",
				"uploadId": "upload-id-001",
				"confirmKey": "confirm-key-001",
				"headers": {"Authorization": "fake-sign"},
				"expiration": "2099-01-01T00:00:00Z"
			}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

// withProbeFnReplaced 暂时把 hatchery 用的内网可达性探测函数替换为 fn，
// 测试结束后通过 t.Cleanup 自动还原。隔离全局状态污染。
func withProbeFnReplaced(t *testing.T, fn func(context.Context, string) bool) {
	t.Helper()
	orig := smhProbeInternalReachableFn
	smhProbeInternalReachableFn = fn
	t.Cleanup(func() { smhProbeInternalReachableFn = orig })
}

// ─── C1：内网可达 → 走内网 ───────────────────────────────────────────────

// TestSMHUpload_InternalDomainReachable_UsesInternal 验证：
//  1. prepareSMHUploadWithFallback 触发 Prepare 时，HTTP 请求 query 上注入 internal_domain=1
//  2. 返回的 cred.UsedInternalDomain=true
//  3. 后续 RenewSMHCommonUpload 也带 internal_domain=1（保持链路一致）
//  4. usage/directory 等非上传接口不会被误注入 internal_domain
func TestSMHUpload_InternalDomainReachable_UsesInternal(t *testing.T) {
	capture := &captureSMHHandler{}
	const partDomain = "smhcp-internal.data-tencentsmh.com"
	_, cleanup := setupSMHHTTPTestEnv(t, uploadSuccessHandler(capture, partDomain))
	defer cleanup()

	// 内网视为可达
	withProbeFnReplaced(t, func(_ context.Context, _ string) bool { return true })

	cred, err := prepareSMHUploadWithFallback(context.Background(), "ins-1", "/tmp/state.tgz", 1024)
	if err != nil {
		t.Fatalf("prepareSMHUploadWithFallback 不应失败：%v", err)
	}
	if cred == nil || cred.PartURLTemplate == "" {
		t.Fatalf("应返回非空 cred 和 PartURLTemplate，实际 cred=%+v", cred)
	}
	if !cred.UsedInternalDomain {
		t.Errorf("内网可达时 cred.UsedInternalDomain 应为 true")
	}

	// 断言：上传接口请求带 internal_domain=1，非上传接口不带
	var sawMultiInternal, sawUsageNoInternal bool
	for _, req := range capture.snapshot() {
		switch {
		case req.HasMultipart:
			if req.InternalDomain != "1" {
				t.Errorf("multipart 请求应注入 internal_domain=1，实际=%q (query=%s)", req.InternalDomain, req.Query)
			}
			sawMultiInternal = true
		case strings.Contains(req.Path, "/usage/") || strings.Contains(req.Path, "/directory/"):
			if req.InternalDomain != "" {
				t.Errorf("非上传接口 %s 不应注入 internal_domain，实际=%q", req.Path, req.InternalDomain)
			}
			sawUsageNoInternal = true
		}
	}
	if !sawMultiInternal {
		t.Error("未观察到 multipart 上传请求")
	}
	if !sawUsageNoInternal {
		t.Error("未观察到 usage/directory 请求（mock 路径覆盖不全？）")
	}

	// 再走一次 Renew，验证续期同样带 internal_domain=1
	if err := RenewSMHCommonUpload(context.Background(), cred); err != nil {
		t.Fatalf("Renew 不应失败：%v", err)
	}
	sawRenewInternal := false
	for _, req := range capture.snapshot() {
		if req.HasRenew {
			if req.InternalDomain != "1" {
				t.Errorf("renew 请求应注入 internal_domain=1，实际=%q (query=%s)", req.InternalDomain, req.Query)
			}
			sawRenewInternal = true
		}
	}
	if !sawRenewInternal {
		t.Error("未观察到 renew 请求")
	}
}

// ─── C2：内网不可达 → 降级外网 ───────────────────────────────────────────

// TestSMHUpload_InternalDomainUnreachable_FallsBackToExternal 验证：
//  1. 首次 Prepare 用 internal_domain=1 拿凭证
//  2. 探测不可达后，再次调用 Prepare（不带 internal_domain）拿外网凭证
//  3. 返回的 cred.UsedInternalDomain=false
//  4. Renew 也不带 internal_domain=1
func TestSMHUpload_InternalDomainUnreachable_FallsBackToExternal(t *testing.T) {
	capture := &captureSMHHandler{}
	const partDomain = "smh-external.cos.tencentsmh.cn"
	_, cleanup := setupSMHHTTPTestEnv(t, uploadSuccessHandler(capture, partDomain))
	defer cleanup()

	// 内网视为不可达
	var probeCalls int32
	withProbeFnReplaced(t, func(_ context.Context, _ string) bool {
		atomic.AddInt32(&probeCalls, 1)
		return false
	})

	cred, err := prepareSMHUploadWithFallback(context.Background(), "ins-1", "/tmp/state.tgz", 1024)
	if err != nil {
		t.Fatalf("prepareSMHUploadWithFallback 不应失败：%v", err)
	}
	if cred == nil || cred.PartURLTemplate == "" {
		t.Fatalf("应返回非空 cred，实际 cred=%+v", cred)
	}
	if cred.UsedInternalDomain {
		t.Errorf("内网不可达后应回退为外网，cred.UsedInternalDomain 应为 false")
	}
	if atomic.LoadInt32(&probeCalls) != 1 {
		t.Errorf("探测应被调用 1 次，实际 %d 次", probeCalls)
	}

	// 统计 multipart 请求的 internal_domain 注入情况：
	// 期望恰好 2 次 multipart 请求（首次内网 + 降级外网），第一次=1，第二次为空
	var multiReqs []capturedReq
	for _, req := range capture.snapshot() {
		if req.HasMultipart {
			multiReqs = append(multiReqs, req)
		}
	}
	if len(multiReqs) != 2 {
		t.Fatalf("应有 2 次 multipart 请求（内网+外网），实际 %d 次", len(multiReqs))
	}
	if multiReqs[0].InternalDomain != "1" {
		t.Errorf("第 1 次 multipart 应带 internal_domain=1，实际=%q", multiReqs[0].InternalDomain)
	}
	if multiReqs[1].InternalDomain != "" {
		t.Errorf("第 2 次（降级）multipart 不应带 internal_domain，实际=%q", multiReqs[1].InternalDomain)
	}

	// Renew 也应保持外网，不带 internal_domain
	if err := RenewSMHCommonUpload(context.Background(), cred); err != nil {
		t.Fatalf("Renew 不应失败：%v", err)
	}
	for _, req := range capture.snapshot() {
		if req.HasRenew {
			if req.InternalDomain != "" {
				t.Errorf("外网 cred 续期不应注入 internal_domain，实际=%q", req.InternalDomain)
			}
		}
	}
}

// ─── C3：外网 Prepare 也失败 → 回退用内网 cred 兜底 ─────────────────────────

// TestSMHUpload_ExternalPrepareFails_FallbackToInternalCred 验证：
// 当内网不可达 + 外网 Prepare 也失败时，函数不应直接报错让升级流程中断，
// 而是回退使用首次拿到的内网 cred，让 CVM 走原有路径继续尝试。
func TestSMHUpload_ExternalPrepareFails_FallbackToInternalCred(t *testing.T) {
	capture := &captureSMHHandler{}
	const partDomain = "smhcp-internal.data-tencentsmh.com"

	// 自定义 handler：仅"第二次 multipart 请求"返回 500，其它都成功
	var multiCount int32
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		_, hasMulti := q["multipart"]
		_, hasRenew := q["renew"]
		capture.mu.Lock()
		capture.reqs = append(capture.reqs, capturedReq{
			Path:           r.URL.Path,
			Query:          r.URL.RawQuery,
			HasMultipart:   hasMulti,
			HasRenew:       hasRenew,
			InternalDomain: q.Get("internal_domain"),
		})
		capture.mu.Unlock()

		switch {
		case strings.Contains(r.URL.Path, "/usage/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"spaceId":"sp-common","size":"0"}]`))
		case strings.Contains(r.URL.Path, "/directory/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"path":["backups","ins-1"]}`))
		case strings.Contains(r.URL.Path, "/file/") && hasMulti:
			n := atomic.AddInt32(&multiCount, 1)
			if n == 2 {
				// 第二次（外网降级）失败
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"code":"InternalError","message":"boom"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{
				"domain": "` + partDomain + `",
				"path": "/backups/ins-1/state.tgz",
				"uploadId": "upload-id-001",
				"confirmKey": "confirm-key-001",
				"headers": {"Authorization": "fake-sign"},
				"expiration": "2099-01-01T00:00:00Z"
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	withProbeFnReplaced(t, func(_ context.Context, _ string) bool { return false })

	cred, err := prepareSMHUploadWithFallback(context.Background(), "ins-1", "/tmp/state.tgz", 1024)
	if err != nil {
		t.Fatalf("外网失败时应回退内网 cred，不应直接返回 error，实际 err=%v", err)
	}
	if cred == nil || cred.PartURLTemplate == "" {
		t.Fatal("应回退使用内网 cred，cred 不应为空")
	}
	if !cred.UsedInternalDomain {
		t.Errorf("外网降级失败后应保留内网 cred，UsedInternalDomain 应为 true，实际=%v", cred.UsedInternalDomain)
	}
	if got := atomic.LoadInt32(&multiCount); got != 2 {
		t.Errorf("应有 2 次 multipart 请求（内网成功 + 外网失败），实际 %d 次", got)
	}
}

// ─── C4：秒传命中 → 不需要探测，直接返回 ────────────────────────────────

// TestSMHUpload_InstantUpload200_SkipsProbe 验证：当 SMH 返回 200 表示秒传成功时，
// PartURLTemplate 为空，prepareSMHUploadWithFallback 应短路返回，不调用探测函数，
// 也不进行外网降级。
func TestSMHUpload_InstantUpload200_SkipsProbe(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		_, hasMulti := q["multipart"]
		switch {
		case strings.Contains(r.URL.Path, "/usage/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"spaceId":"sp-common","size":"0"}]`))
		case strings.Contains(r.URL.Path, "/directory/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"path":["backups","ins-1"]}`))
		case strings.Contains(r.URL.Path, "/file/") && hasMulti:
			// 秒传：返回 200 + 文件信息
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"path":["backups","ins-1","state.tgz"],
				"name":"state.tgz",
				"type":"file",
				"size":"1024"
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	var probeCalls int32
	withProbeFnReplaced(t, func(_ context.Context, _ string) bool {
		atomic.AddInt32(&probeCalls, 1)
		return true
	})

	cred, err := prepareSMHUploadWithFallback(context.Background(), "ins-1", "/tmp/state.tgz", 1024)
	if err != nil {
		t.Fatalf("秒传不应返回错误：%v", err)
	}
	if cred == nil || cred.PartURLTemplate != "" {
		t.Errorf("秒传时 PartURLTemplate 应为空，实际 cred=%+v", cred)
	}
	if cred == nil || cred.FileKey == "" {
		t.Error("秒传时应返回 FileKey")
	}
	if probeCalls != 0 {
		t.Errorf("秒传时不应调用探测，实际调用 %d 次", probeCalls)
	}
}

// ─── C5：Renew 显式双向覆盖（防上游 ctx 污染）─────────────────────────────

// TestSMHUpload_RenewExternalCred_IgnoresUpstreamInternalCtx 验证：
// 即使调用方传入的 ctx 已带 WithSMHInternalDomain(true)，对一个外网 cred
// 续期时也应以 cred.UsedInternalDomain 为准（即不注入 internal_domain）。
// 这是修复"单向覆盖"隐患的回归用例。
func TestSMHUpload_RenewExternalCred_IgnoresUpstreamInternalCtx(t *testing.T) {
	capture := &captureSMHHandler{}
	_, cleanup := setupSMHHTTPTestEnv(t, uploadSuccessHandler(capture, "smh-external.cos.tencentsmh.cn"))
	defer cleanup()

	// 手工构造一个"外网"凭证
	cred := &SMHUploadCredential{
		ConfirmKey:         "confirm-key-ext",
		FileKey:            "backups/ins-1/state.tgz",
		UsedInternalDomain: false,
	}

	// 故意污染：上游 ctx 带 true
	ctx := WithSMHInternalDomain(context.Background(), true)
	if err := RenewSMHCommonUpload(ctx, cred); err != nil {
		t.Fatalf("Renew 不应失败：%v", err)
	}

	for _, req := range capture.snapshot() {
		if req.HasRenew && req.InternalDomain != "" {
			t.Errorf("外网 cred 续期时不应注入 internal_domain（即使上游 ctx 误标 true），实际=%q (query=%s)",
				req.InternalDomain, req.Query)
		}
	}
}
