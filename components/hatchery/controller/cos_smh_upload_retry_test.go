package controller

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	client "cnb.cool/tencent/cloud/smh/smh-go-sdk"
	"cnb.cool/tencent/cloud/smh/smh-go-sdk/transfer"

	"hatchery/model"
)

// ============================================================================
// isRateLimitError 单元测试
// ============================================================================

func TestIsRateLimitError_NilError(t *testing.T) {
	if isRateLimitError(nil) {
		t.Error("nil error 应返回 false")
	}
}

func TestIsRateLimitError_Contains429(t *testing.T) {
	err := errors.New("something 429 Too Many Requests")
	if !isRateLimitError(err) {
		t.Error("包含 429 的错误应返回 true")
	}
}

func TestIsRateLimitError_ContainsTooManyRequests(t *testing.T) {
	err := errors.New("failed to confirm upload: Too Many Requests")
	if isRateLimitError(err) {
		t.Error("不含 429 的错误不应被当作限流错误")
	}
}

func TestIsRateLimitError_OtherError(t *testing.T) {
	err := errors.New("network timeout")
	if isRateLimitError(err) {
		t.Error("普通错误应返回 false")
	}
}

func TestIsRateLimitError_NotFoundError(t *testing.T) {
	err := errors.New("404 Not Found")
	if isRateLimitError(err) {
		t.Error("404 错误不应被当作限流错误")
	}
}

func TestIsRateLimitError_Wrapped429Error(t *testing.T) {
	inner := errors.New("HTTP 429 Too Many Requests")
	wrapped := fmt.Errorf("upload failed: %w", inner)
	if !isRateLimitError(wrapped) {
		t.Error("包装后的 429 错误应返回 true")
	}
}

// ============================================================================
// smhClient.Upload 指数退避重试测试（mock transferUpload）
// ============================================================================

func setupUploadRetryTestDB(t *testing.T) func() {
	t.Helper()
	cleanup := setupCosCommonTestDB(t)
	seedSMHFullyConfigured(t)
	expiredAt := time.Now().Add(24 * time.Hour).Unix()
	model.UpdateSMHSpaceToken(context.Background(), "skillhub", true, "fake-token", expiredAt)
	return cleanup
}

// newRetryMockUpload 返回一个按次数控制行为的 mock upload 函数。
// errorList[i] 为第 i+1 次调用的返回值。若 calls > len(errorList)，返回最后一次定义的 error。
func newRetryMockUpload(errorList []error) (mockFn func() error, callCount *int) {
	callCount = new(int)
	mockFn = func() error {
		*callCount++
		idx := *callCount - 1
		if idx < len(errorList) {
			return errorList[idx]
		}
		return errorList[len(errorList)-1]
	}
	return
}

func TestUpload_FirstAttemptSuccess(t *testing.T) {
	cleanup := setupUploadRetryTestDB(t)
	defer cleanup()

	mockFn, calls := newRetryMockUpload([]error{nil})
	origUpload := transferUpload
	transferUpload = func(ctx context.Context, file transfer.ReaderFileOptions, opts *transfer.UploadOptions, cfg *client.Configuration) (*transfer.UploadResult, error) {
		return nil, mockFn()
	}
	defer func() { transferUpload = origUpload }()

	sc, err := getStorageClient(context.Background())
	if err != nil {
		t.Fatalf("getStorageClient: %v", err)
	}
	if err := sc.Upload("test.zip", []byte("hello"), "application/zip"); err != nil {
		t.Fatalf("首次上传应成功: %v", err)
	}
	if *calls != 1 {
		t.Errorf("应调用 1 次，实际=%d", *calls)
	}
}

func TestUpload_RetryAfter429ThenSuccess(t *testing.T) {
	cleanup := setupUploadRetryTestDB(t)
	defer cleanup()

	rateLimitErr := errors.New("[NetworkFailure] failed to confirm upload: 429 Too Many Requests")
	mockFn, calls := newRetryMockUpload([]error{rateLimitErr, rateLimitErr, nil})
	origUpload := transferUpload
	transferUpload = func(ctx context.Context, file transfer.ReaderFileOptions, opts *transfer.UploadOptions, cfg *client.Configuration) (*transfer.UploadResult, error) {
		return nil, mockFn()
	}
	defer func() { transferUpload = origUpload }()

	sc, err := getStorageClient(context.Background())
	if err != nil {
		t.Fatalf("getStorageClient: %v", err)
	}
	start := time.Now()
	if err := sc.Upload("test.zip", []byte("hello"), "application/zip"); err != nil {
		t.Fatalf("重试后应成功: %v", err)
	}
	elapsed := time.Since(start)
	if *calls != 3 {
		t.Errorf("应调用 3 次（2 失败+1 成功），实际=%d", *calls)
	}
	if elapsed < 2*time.Second {
		t.Errorf("退避耗时至少 2s，实际=%v", elapsed)
	}
}

func TestUpload_429Exhausted(t *testing.T) {
	cleanup := setupUploadRetryTestDB(t)
	defer cleanup()

	rateLimitErr := errors.New("[NetworkFailure] failed to confirm upload: 429 Too Many Requests")
	mockFn, calls := newRetryMockUpload([]error{
		rateLimitErr, rateLimitErr, rateLimitErr, rateLimitErr, rateLimitErr, rateLimitErr, rateLimitErr,
	})
	origUpload := transferUpload
	transferUpload = func(ctx context.Context, file transfer.ReaderFileOptions, opts *transfer.UploadOptions, cfg *client.Configuration) (*transfer.UploadResult, error) {
		return nil, mockFn()
	}
	defer func() { transferUpload = origUpload }()

	sc, err := getStorageClient(context.Background())
	if err != nil {
		t.Fatalf("getStorageClient: %v", err)
	}
	err = sc.Upload("test.zip", []byte("hello"), "application/zip")
	if err == nil {
		t.Fatal("重试耗尽应返回 error")
	}
	if !isRateLimitError(err) {
		t.Errorf("应为限流错误，实际=%v", err)
	}
	if *calls != 6 {
		t.Errorf("应调用 6 次（首次+5 重试），实际=%d", *calls)
	}
}

func TestUpload_Non429ErrorNoRetry(t *testing.T) {
	cleanup := setupUploadRetryTestDB(t)
	defer cleanup()

	nonRateLimitErr := errors.New("[NetworkFailure] failed to call simple upload API: 500 Internal Server Error")
	mockFn, calls := newRetryMockUpload([]error{nonRateLimitErr})
	origUpload := transferUpload
	transferUpload = func(ctx context.Context, file transfer.ReaderFileOptions, opts *transfer.UploadOptions, cfg *client.Configuration) (*transfer.UploadResult, error) {
		return nil, mockFn()
	}
	defer func() { transferUpload = origUpload }()

	sc, err := getStorageClient(context.Background())
	if err != nil {
		t.Fatalf("getStorageClient: %v", err)
	}
	err = sc.Upload("test.zip", []byte("hello"), "application/zip")
	if err == nil {
		t.Fatal("非 429 错误应返回 error")
	}
	if isRateLimitError(err) {
		t.Error("非 429 错误不应被当作限流错误")
	}
	if *calls != 1 {
		t.Errorf("非 429 不应重试，实际=%d", *calls)
	}
}

func TestUpload_ContextCancelledDuringRetry(t *testing.T) {
	cleanup := setupUploadRetryTestDB(t)
	defer cleanup()

	rateLimitErr := errors.New("[NetworkFailure] failed to confirm upload: 429 Too Many Requests")
	mockFn, _ := newRetryMockUpload([]error{
		rateLimitErr, rateLimitErr, rateLimitErr, rateLimitErr, rateLimitErr, rateLimitErr, rateLimitErr,
	})
	origUpload := transferUpload
	transferUpload = func(ctx context.Context, file transfer.ReaderFileOptions, opts *transfer.UploadOptions, cfg *client.Configuration) (*transfer.UploadResult, error) {
		return nil, mockFn()
	}
	defer func() { transferUpload = origUpload }()

	sc, err := getStorageClient(context.Background())
	if err != nil {
		t.Fatalf("getStorageClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	client := sc.(*smhClient)
	orig := client.ctx
	client.ctx = ctx

	start := time.Now()
	err = client.Upload("test.zip", []byte("hello"), "application/zip")
	client.ctx = orig

	if err == nil {
		t.Fatal("ctx 取消应返回 error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("应为 DeadlineExceeded，实际=%v", err)
	}
	if time.Since(start) > 1500*time.Millisecond {
		t.Errorf("ctx 取消后不应等满退避时间，耗时=%v", time.Since(start))
	}
}

// ============================================================================
// UploadWithContext 辅助：允许测试注入自定义 ctx
// ============================================================================

// UploadWithContext is a test helper that allows passing a custom context.
// In production, smhClient.ctx is used directly.
func (s *smhClient) UploadWithContext(ctx context.Context, key string, data []byte, contentType string) error {
	// 临时替换 ctx 用于测试
	orig := s.ctx
	s.ctx = ctx
	defer func() { s.ctx = orig }()
	return s.Upload(key, data, contentType)
}
