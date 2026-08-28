package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	hcommon "hatchery/common"
)

// withCloudUinFetcher 临时替换 cloudUinFetcher，并在测试结束时还原。
// 同时清空 CAM UIN 缓存，避免不同用例间互相污染。
func withCloudUinFetcher(t *testing.T, fn func(ctx context.Context) (string, error)) {
	t.Helper()
	ResetCAMUinCacheForTest()
	orig := cloudUinFetcher
	cloudUinFetcher = fn
	t.Cleanup(func() {
		cloudUinFetcher = orig
		ResetCAMUinCacheForTest()
	})
}

// ctxWithUin 构造一个注入了指定腾讯云 UIN 的 ctx。
func ctxWithUin(uin string) context.Context {
	return hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Uin: uin})
}

// withFakeCAMServer 启动一个 httptest 服务器扮演 CAM，并把 endpoint/AKSK provider 指过去。
// handler 收到的请求是 SDK 真实序列化后的 CAM 请求；返回值应是 CAM 协议的 JSON 字符串。
func withFakeCAMServer(t *testing.T, handler func(r *http.Request) (status int, body string)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status, body := handler(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse fake server url: %v", err)
	}

	origEndpoint := camEndpointOverride
	origProvider := camAKSKProvider
	camEndpointOverride = u.Host
	camAKSKProvider = func(ctx context.Context) (string, string) {
		return "fake-id", "fake-key"
	}
	ResetCAMUinCacheForTest()
	t.Cleanup(func() {
		srv.Close()
		camEndpointOverride = origEndpoint
		camAKSKProvider = origProvider
		ResetCAMUinCacheForTest()
	})
	return srv
}

// withCAMAKSKProvider 仅替换 AKSK provider，不动 endpoint。
func withCAMAKSKProvider(t *testing.T, fn func(ctx context.Context) (string, string)) {
	t.Helper()
	orig := camAKSKProvider
	camAKSKProvider = fn
	ResetCAMUinCacheForTest()
	t.Cleanup(func() {
		camAKSKProvider = orig
		ResetCAMUinCacheForTest()
	})
}

// ─── IsInternalAccount / ResolveCloudUin ─────────────────────────────────────

func TestIsInternalAccount_FromCtx_Hit(t *testing.T) {
	withCloudUinFetcher(t, func(ctx context.Context) (string, error) {
		t.Fatal("ctx 已能拿到 UIN，不应触发 CAM 调用")
		return "", nil
	})

	got, err := IsInternalAccount(ctxWithUin("3205597606"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatalf("want internal=true, got false")
	}
}

func TestIsInternalAccount_FromCtx_Miss(t *testing.T) {
	withCloudUinFetcher(t, func(ctx context.Context) (string, error) {
		t.Fatal("ctx 已能拿到 UIN，不应触发 CAM 调用")
		return "", nil
	})

	got, err := IsInternalAccount(ctxWithUin("9999999999"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatalf("want internal=false, got true")
	}
}

func TestIsInternalAccount_FallbackToCAM_Hit(t *testing.T) {
	called := 0
	withCloudUinFetcher(t, func(ctx context.Context) (string, error) {
		called++
		return "3205597606", nil
	})

	got, err := IsInternalAccount(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatalf("want internal=true via CAM, got false")
	}
	if called != 1 {
		t.Fatalf("CAM fetcher should be called exactly once, got %d", called)
	}
}

func TestIsInternalAccount_FallbackToCAM_Miss(t *testing.T) {
	withCloudUinFetcher(t, func(ctx context.Context) (string, error) {
		return "1234567890", nil
	})

	got, err := IsInternalAccount(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatalf("want internal=false, got true")
	}
}

func TestIsInternalAccount_FallbackToCAM_Empty(t *testing.T) {
	withCloudUinFetcher(t, func(ctx context.Context) (string, error) {
		return "", nil
	})

	got, err := IsInternalAccount(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatalf("want internal=false on empty uin, got true")
	}
}

func TestIsInternalAccount_FallbackToCAM_Error(t *testing.T) {
	wantErr := errors.New("cam down")
	withCloudUinFetcher(t, func(ctx context.Context) (string, error) {
		return "", wantErr
	})

	got, err := IsInternalAccount(context.Background())
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("want wrapped error %v, got %v", wantErr, err)
	}
	if got {
		t.Fatalf("error case must return false")
	}
}

// ctx 中已有 UIN 但其值就是内部白名单时，也不应触发 CAM。
func TestIsInternalAccount_CtxTrumpsFetcher(t *testing.T) {
	withCloudUinFetcher(t, func(ctx context.Context) (string, error) {
		t.Fatal("ctx 优先级最高，不应回落到 CAM")
		return "9999999999", nil
	})
	got, err := IsInternalAccount(ctxWithUin("3205597606"))
	if err != nil || !got {
		t.Fatalf("ctx hit white list should win, got=%v err=%v", got, err)
	}
}

// nil ctx 应被安全处理（IsInternalAccount → ResolveCloudUin → CVMUinFromCtx 均需兜底）。
func TestIsInternalAccount_NilCtx(t *testing.T) {
	withCloudUinFetcher(t, func(ctx context.Context) (string, error) {
		return "3205597606", nil
	})
	//nolint:staticcheck // 故意传 nil ctx 验证健壮性
	got, err := IsInternalAccount(nil)
	if err != nil {
		t.Fatalf("nil ctx should not error: %v", err)
	}
	if !got {
		t.Fatalf("want internal=true under nil ctx fallback")
	}
}

func TestResolveCloudUin_NoFetcher(t *testing.T) {
	ResetCAMUinCacheForTest()
	orig := cloudUinFetcher
	cloudUinFetcher = nil
	t.Cleanup(func() {
		cloudUinFetcher = orig
		ResetCAMUinCacheForTest()
	})

	if _, err := ResolveCloudUin(context.Background()); err == nil {
		t.Fatal("want error when fetcher is nil and ctx has no uin")
	}
}

func TestResolveCloudUin_CtxFirst(t *testing.T) {
	withCloudUinFetcher(t, func(ctx context.Context) (string, error) {
		t.Fatal("ctx 已有 UIN，不应触发 fetcher")
		return "", nil
	})
	uin, err := ResolveCloudUin(ctxWithUin("123"))
	if err != nil || uin != "123" {
		t.Fatalf("want 123/<nil>, got %q/%v", uin, err)
	}
}

// ─── fetchCloudUinViaCAM 真实覆盖（替换底层 doCallCAMGetUserAppId 走的 endpoint/AKSK） ───

// 成功路径：首次调用穿透到 CAM，后续命中进程内缓存，handler 仅被触发一次。
func TestFetchCloudUinViaCAM_CachesSuccess(t *testing.T) {
	var calls int32
	withFakeCAMServer(t, func(r *http.Request) (int, string) {
		atomic.AddInt32(&calls, 1)
		return http.StatusOK, `{"Response":{"OwnerUin":"3205597606","Uin":"3205597606","AppId":1300000000,"RequestId":"req-ok"}}`
	})

	for i := 0; i < 5; i++ {
		uin, err := fetchCloudUinViaCAM(context.Background())
		if err != nil {
			t.Fatalf("iter %d unexpected error: %v", i, err)
		}
		if uin != "3205597606" {
			t.Fatalf("want 3205597606, got %q", uin)
		}
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("CAM handler should be called exactly once due to cache, got %d", got)
	}
}

// 失败路径：CAM 返回业务错误，once 应被重置以允许重试；缓存不能污染。
func TestFetchCloudUinViaCAM_RetryOnError(t *testing.T) {
	var calls int32
	withFakeCAMServer(t, func(r *http.Request) (int, string) {
		atomic.AddInt32(&calls, 1)
		return http.StatusOK, `{"Response":{"Error":{"Code":"AuthFailure","Message":"bad sig"},"RequestId":"req-err"}}`
	})

	for i := 0; i < 3; i++ {
		if _, err := fetchCloudUinViaCAM(context.Background()); err == nil {
			t.Fatalf("iter %d want error, got nil", i)
		}
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("failure should not be cached: want 3 calls, got %d", got)
	}
	if camUinCache != "" {
		t.Fatalf("failure must not pollute cache, got %q", camUinCache)
	}
}

// 失败后再成功：先失败、once 重置；下一次成功结果应被缓存。
func TestFetchCloudUinViaCAM_FailThenSuccessCached(t *testing.T) {
	var calls int32
	withFakeCAMServer(t, func(r *http.Request) (int, string) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			return http.StatusOK, `{"Response":{"Error":{"Code":"InternalError","Message":"oops"},"RequestId":"r1"}}`
		}
		return http.StatusOK, `{"Response":{"OwnerUin":"3205597606","RequestId":"r2"}}`
	})

	if _, err := fetchCloudUinViaCAM(context.Background()); err == nil {
		t.Fatal("first call should fail")
	}
	for i := 0; i < 3; i++ {
		uin, err := fetchCloudUinViaCAM(context.Background())
		if err != nil || uin != "3205597606" {
			t.Fatalf("after recovery iter %d want 3205597606/nil, got %q/%v", i, uin, err)
		}
	}
	// 1 次失败 + 1 次成功 = 2 次握手，第三次起命中缓存。
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("want 2 upstream calls, got %d", got)
	}
}

// 并发路径：多 goroutine 同时首次调用，CAM handler 仅被命中一次（once 语义）。
func TestFetchCloudUinViaCAM_ConcurrentOnceFires(t *testing.T) {
	var calls int32
	withFakeCAMServer(t, func(r *http.Request) (int, string) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(20 * time.Millisecond) // 模拟慢网络，让并发 goroutine 撞到 once.Do 门上
		return http.StatusOK, `{"Response":{"OwnerUin":"3205597606","RequestId":"r"}}`
	})

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			uin, err := fetchCloudUinViaCAM(context.Background())
			if err != nil || uin != "3205597606" {
				t.Errorf("got uin=%q err=%v", uin, err)
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("CAM handler should be called exactly once across goroutines, got %d", got)
	}
}

// ─── doCallCAMGetUserAppId 各响应形态覆盖 ─────────────────────────────────────

func TestDoCallCAM_OwnerUin(t *testing.T) {
	withFakeCAMServer(t, func(r *http.Request) (int, string) {
		return http.StatusOK, `{"Response":{"OwnerUin":"3205597606","Uin":"99","RequestId":"r"}}`
	})
	uin, err := doCallCAMGetUserAppId(context.Background())
	if err != nil || uin != "3205597606" {
		t.Fatalf("want 3205597606, got %q err=%v", uin, err)
	}
}

func TestDoCallCAM_FallbackToUin(t *testing.T) {
	withFakeCAMServer(t, func(r *http.Request) (int, string) {
		return http.StatusOK, `{"Response":{"OwnerUin":"","Uin":"7777","RequestId":"r"}}`
	})
	uin, err := doCallCAMGetUserAppId(context.Background())
	if err != nil || uin != "7777" {
		t.Fatalf("want 7777, got %q err=%v", uin, err)
	}
}

func TestDoCallCAM_BothMissing(t *testing.T) {
	withFakeCAMServer(t, func(r *http.Request) (int, string) {
		return http.StatusOK, `{"Response":{"RequestId":"r"}}`
	})
	if _, err := doCallCAMGetUserAppId(context.Background()); err == nil ||
		!strings.Contains(err.Error(), "OwnerUin/Uin") {
		t.Fatalf("want OwnerUin/Uin missing error, got %v", err)
	}
}

func TestDoCallCAM_BusinessError(t *testing.T) {
	withFakeCAMServer(t, func(r *http.Request) (int, string) {
		return http.StatusOK, `{"Response":{"Error":{"Code":"AuthFailure.SignatureFailure","Message":"sign error"},"RequestId":"r"}}`
	})
	_, err := doCallCAMGetUserAppId(context.Background())
	if err == nil {
		t.Fatal("want CAM business error")
	}
	if !strings.Contains(err.Error(), "AuthFailure") {
		t.Fatalf("error should carry CAM code, got %v", err)
	}
}

func TestDoCallCAM_HTTP500(t *testing.T) {
	withFakeCAMServer(t, func(r *http.Request) (int, string) {
		return http.StatusInternalServerError, `{"Response":{"Error":{"Code":"InternalError","Message":"boom"}}}`
	})
	if _, err := doCallCAMGetUserAppId(context.Background()); err == nil {
		t.Fatal("want error on http 500")
	}
}

func TestDoCallCAM_MalformedJSON(t *testing.T) {
	withFakeCAMServer(t, func(r *http.Request) (int, string) {
		return http.StatusOK, `not-json-at-all`
	})
	if _, err := doCallCAMGetUserAppId(context.Background()); err == nil {
		t.Fatal("want error on malformed json")
	}
}

// AKSK 未配置时，应直接报错且不发起 HTTP 请求。
func TestDoCallCAM_MissingAKSK(t *testing.T) {
	var hit int32
	withFakeCAMServer(t, func(r *http.Request) (int, string) {
		atomic.AddInt32(&hit, 1)
		return http.StatusOK, `{"Response":{"OwnerUin":"x"}}`
	})
	// 覆盖 fake server 注入的 provider，把 AKSK 清空。
	withCAMAKSKProvider(t, func(ctx context.Context) (string, string) { return "", "" })

	_, err := doCallCAMGetUserAppId(context.Background())
	if err == nil || !strings.Contains(err.Error(), "AKSK") {
		t.Fatalf("want AKSK missing error, got %v", err)
	}
	if atomic.LoadInt32(&hit) != 0 {
		t.Fatalf("should short-circuit before HTTP, got hits=%d", hit)
	}
}

// SecretId 单独缺失也应短路。
func TestDoCallCAM_PartialAKSK(t *testing.T) {
	withFakeCAMServer(t, func(r *http.Request) (int, string) {
		t.Fatal("should not be reached")
		return 0, ""
	})
	withCAMAKSKProvider(t, func(ctx context.Context) (string, string) { return "", "key-only" })

	if _, err := doCallCAMGetUserAppId(context.Background()); err == nil {
		t.Fatal("secretID empty should error")
	}

	withCAMAKSKProvider(t, func(ctx context.Context) (string, string) { return "id-only", "" })
	if _, err := doCallCAMGetUserAppId(context.Background()); err == nil {
		t.Fatal("secretKey empty should error")
	}
}

// nil ctx 走兜底分支 ctx = context.Background()，不应 panic。
func TestDoCallCAM_NilCtx(t *testing.T) {
	withFakeCAMServer(t, func(r *http.Request) (int, string) {
		return http.StatusOK, `{"Response":{"OwnerUin":"888","RequestId":"r"}}`
	})
	//nolint:staticcheck // 故意传 nil ctx 验证健壮性
	uin, err := doCallCAMGetUserAppId(nil)
	if err != nil || uin != "888" {
		t.Fatalf("want 888/<nil>, got %q/%v", uin, err)
	}
}

// ─── ResetCAMUinCacheForTest ─────────────────────────────────────────────────

func TestResetCAMUinCacheForTest_ClearsState(t *testing.T) {
	camUinMu.Lock()
	camUinCache = "stale"
	camUinOnce.Do(func() {}) // 模拟 once 已被消费
	camUinMu.Unlock()

	ResetCAMUinCacheForTest()

	camUinMu.Lock()
	got := camUinCache
	camUinMu.Unlock()
	if got != "" {
		t.Fatalf("cache not cleared, got %q", got)
	}

	// 重置后，再次 Do 仍应执行一次（验证 once 是新实例）。
	var ran bool
	camUinMu.Lock()
	camUinOnce.Do(func() { ran = true })
	camUinMu.Unlock()
	if !ran {
		t.Fatal("camUinOnce should be a fresh instance after reset")
	}
	// 收尾
	ResetCAMUinCacheForTest()
}

// ─── 白名单覆盖 ──────────────────────────────────────────────────────────────

func TestInternalAccountUins_ContainsKnownUins(t *testing.T) {
	for _, uin := range []string{"3205597606", "100049049642"} {
		if !isInternalAccountUin(uin) {
			t.Fatalf("白名单应包含已知内部 UIN %s", uin)
		}
	}
}
