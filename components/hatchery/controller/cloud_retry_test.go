package controller

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	tcerr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
)

// withFastRetryParams 临时将重试退避缩短到 1ms，加速测试。
func withFastRetryParams(t *testing.T) {
	t.Helper()
	origMax := cloudQueryRetryMaxAttempts
	origBackoff := cloudQueryRetryBaseBackoff
	cloudQueryRetryMaxAttempts = 3
	cloudQueryRetryBaseBackoff = 1 * time.Millisecond
	t.Cleanup(func() {
		cloudQueryRetryMaxAttempts = origMax
		cloudQueryRetryBaseBackoff = origBackoff
	})
}

func TestRetryCloudCall_SuccessFirstAttempt(t *testing.T) {
	withFastRetryParams(t)
	var calls int32
	err := RetryCloudCall(context.Background(), func() error {
		atomic.AddInt32(&calls, 1)
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected 1 call, got %d", got)
	}
}

func TestRetryCloudCall_SuccessAfterRetry(t *testing.T) {
	withFastRetryParams(t)
	var calls int32
	err := RetryCloudCall(context.Background(), func() error {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return errors.New("net/http: TLS handshake timeout")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error after retry, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 calls, got %d", got)
	}
}

func TestRetryCloudCall_NonRetryableErrorImmediate(t *testing.T) {
	withFastRetryParams(t)
	var calls int32
	permErr := &tcerr.TencentCloudSDKError{
		Code:    "InvalidParameter",
		Message: "bad param",
	}
	err := RetryCloudCall(context.Background(), func() error {
		atomic.AddInt32(&calls, 1)
		return permErr
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("non-retryable error should not retry, expected 1 call, got %d", got)
	}
}

func TestRetryCloudCall_MaxAttemptsExhausted(t *testing.T) {
	withFastRetryParams(t)
	var calls int32
	err := RetryCloudCall(context.Background(), func() error {
		atomic.AddInt32(&calls, 1)
		return errors.New("net/http: TLS handshake timeout")
	})
	if err == nil {
		t.Fatal("expected error after max retries, got nil")
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 calls (max attempts), got %d", got)
	}
}

func TestRetryCloudCall_ContextCancelledDuringBackoff(t *testing.T) {
	// 用较长退避确保 context 在等待期间取消
	origMax := cloudQueryRetryMaxAttempts
	origBackoff := cloudQueryRetryBaseBackoff
	cloudQueryRetryMaxAttempts = 10
	cloudQueryRetryBaseBackoff = 5 * time.Second
	t.Cleanup(func() {
		cloudQueryRetryMaxAttempts = origMax
		cloudQueryRetryBaseBackoff = origBackoff
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var calls int32
	start := time.Now()
	err := RetryCloudCall(ctx, func() error {
		atomic.AddInt32(&calls, 1)
		return errors.New("net/http: TLS handshake timeout")
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
	// 应在 ~50ms 后返回，而不是等满 5s 退避
	if elapsed > 2*time.Second {
		t.Fatalf("should return shortly after ctx cancel, took %v", elapsed)
	}
}

func TestRetryCloudCall_AlreadyCancelledContext(t *testing.T) {
	withFastRetryParams(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var calls int32
	err := RetryCloudCall(ctx, func() error {
		atomic.AddInt32(&calls, 1)
		return errors.New("net/http: TLS handshake timeout")
	})
	// 首次调用会执行（fn 不检查 ctx），但退避时 ctx 已取消 → 返回 ctx.Err()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// 至少调用 1 次（首次 fn 执行），但不应达到 maxAttempts
	if got := atomic.LoadInt32(&calls); got >= 3 {
		t.Fatalf("should not retry after context cancel, got %d calls", got)
	}
}

func TestRetryCloudCall_RetryableTCErrorCodes(t *testing.T) {
	withFastRetryParams(t)
	retryableCodes := []string{
		"RequestLimitExceeded",
		"RequestRateLimitExceeded",
		"InternalError",
		"InternalServerError",
		"ClientError.NetworkError",
	}
	for _, code := range retryableCodes {
		t.Run(code, func(t *testing.T) {
			var calls int32
			tcErr := &tcerr.TencentCloudSDKError{
				Code:    code,
				Message: "transient",
			}
			err := RetryCloudCall(context.Background(), func() error {
				n := atomic.AddInt32(&calls, 1)
				if n < 2 {
					return tcErr
				}
				return nil
			})
			if err != nil {
				t.Fatalf("code %s should be retryable, got error: %v", code, err)
			}
			if got := atomic.LoadInt32(&calls); got < 2 {
				t.Fatalf("code %s should trigger retry, got %d calls", code, got)
			}
		})
	}
}

func TestRetryCloudCall_ConnectionRefusedRetried(t *testing.T) {
	withFastRetryParams(t)
	var calls int32
	err := RetryCloudCall(context.Background(), func() error {
		n := atomic.AddInt32(&calls, 1)
		if n < 2 {
			return errors.New("dial tcp 10.0.0.1:443: connect: connection refused")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 calls, got %d", got)
	}
}

// isRetryableError 是从 ruleset_helpers.go 前移到 cloud_retry.go 的判定函数，
// 其分类测试随之迁移到本文件，与函数定义同处。

func TestIsRetryableError_Nil(t *testing.T) {
	if isRetryableError(nil) {
		t.Error("nil err should not be retryable")
	}
}

func TestIsRetryableError_DeadlineExceeded(t *testing.T) {
	if !isRetryableError(context.DeadlineExceeded) {
		t.Error("deadline exceeded should be retryable")
	}
}

func TestIsRetryableError_NetworkStrings(t *testing.T) {
	if !isRetryableError(errors.New("dial tcp: connection refused")) {
		t.Error("connection refused should be retryable")
	}
	if !isRetryableError(errors.New("i/o timeout")) {
		t.Error("timeout should be retryable")
	}
}

func TestIsRetryableError_TencentCloudCodes(t *testing.T) {
	retryable := []string{
		"RequestLimitExceeded", "RequestRateLimitExceeded",
		"InternalError", "InternalServerError", "ClientError.NetworkError",
	}
	for _, code := range retryable {
		sdkErr := &tcerr.TencentCloudSDKError{Code: code, Message: "boom"}
		if !isRetryableError(sdkErr) {
			t.Errorf("code %q should be retryable", code)
		}
	}

	notRetryable := &tcerr.TencentCloudSDKError{
		Code:    "InvalidParameter",
		Message: "bad argument",
	}
	if isRetryableError(notRetryable) {
		t.Error("InvalidParameter should not be retryable")
	}
}

func TestIsRetryableError_RandomErrorNotRetryable(t *testing.T) {
	if isRetryableError(errors.New("some random error")) {
		t.Error("random error shouldn't be retryable")
	}
}

// TestRetryWithBackoff_ZeroMaxAttempts 覆盖 retryWithBackoff 的前置守卫：
// maxAttempts<=0 时直接返回 nil，且不执行 fn（对应 cloud_retry.go:49-51）。
func TestRetryWithBackoff_ZeroMaxAttempts(t *testing.T) {
	var calls int32
	err := retryWithBackoff(context.Background(), 0, time.Millisecond, func() error {
		atomic.AddInt32(&calls, 1)
		return errors.New("should never be called")
	})
	if err != nil {
		t.Fatalf("expected nil for maxAttempts<=0, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("fn must not be called when maxAttempts<=0, got %d calls", got)
	}
}

// 负值同样应走守卫分支。
func TestRetryWithBackoff_NegativeMaxAttempts(t *testing.T) {
	var calls int32
	err := retryWithBackoff(context.Background(), -1, time.Millisecond, func() error {
		atomic.AddInt32(&calls, 1)
		return errors.New("should never be called")
	})
	if err != nil {
		t.Fatalf("expected nil for negative maxAttempts, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("fn must not be called when maxAttempts<0, got %d calls", got)
	}
}
