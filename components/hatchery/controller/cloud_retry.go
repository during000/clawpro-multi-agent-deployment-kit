package controller

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	tcerr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
)

// 云 API 查询重试参数（包级变量，便于测试覆盖）。
var (
	cloudQueryRetryMaxAttempts = 3                      // 最大尝试次数（含首次）
	cloudQueryRetryBaseBackoff = 200 * time.Millisecond // 首次退避基数，后续指数增长
)

// RetryCloudCall 对只读云 API 调用做 transient 错误重试。
//
// 调用方在 fn 闭包中执行 SDK 查询调用（Describe*/List*/Get*/AssumeRole 等），
// 将响应捕获到外部变量，fn 仅返回 error。
//
// 行为：
//   - 成功（fn 返回 nil）：立即返回 nil
//   - 不可重试错误（InvalidParameter 等）：立即返回，不重试
//   - 可重试错误（timeout / connection refused / RequestLimitExceeded 等）：
//     指数退避重试，最多 cloudQueryRetryMaxAttempts 次
//   - 退避等待期间 context 取消：立即返回 ctx.Err()
//
// 示例：
//
//	var resp *vpc.DescribeVpcsResponse
//	err := RetryCloudCall(ctx, func() error {
//	    var callErr error
//	    resp, callErr = vpcClient.DescribeVpcs(req)
//	    return callErr
//	})
func RetryCloudCall(ctx context.Context, fn func() error) error {
	return retryWithBackoff(ctx, cloudQueryRetryMaxAttempts, cloudQueryRetryBaseBackoff, fn)
}

// retryWithBackoff 执行 fn，transient 错误指数退避重试，不可重试错误立即返回。
//
// 退避策略：baseBackoff << attempt + 随机 jitter（0~100ms），
// 与 applyRulesToSGWithRetry 保持一致。
func retryWithBackoff(ctx context.Context, maxAttempts int, baseBackoff time.Duration, fn func() error) error {
	if maxAttempts <= 0 {
		return nil
	}
	for attempt := 0; ; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		if !isRetryableError(err) {
			return err
		}
		if attempt == maxAttempts-1 {
			return err
		}
		backoff := baseBackoff << attempt
		jitter := time.Duration(rand.Int63n(int64(100 * time.Millisecond)))
		slog.Debug("cloud API query retry",
			"attempt", attempt+1,
			"max_attempts", maxAttempts,
			"backoff_ms", (backoff + jitter).Milliseconds(),
			"error", err)
		select {
		case <-time.After(backoff + jitter):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// isRetryableError 判断腾讯云错误是否属于 transient 可重试类别。
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	msg := err.Error()
	if strings.Contains(msg, "connection refused") || strings.Contains(msg, "timeout") {
		return true
	}
	var tce *tcerr.TencentCloudSDKError
	if errors.As(err, &tce) {
		switch tce.Code {
		case "RequestLimitExceeded",
			"RequestRateLimitExceeded",
			"InternalError",
			"InternalServerError",
			"ClientError.NetworkError":
			return true
		}
	}
	return false
}
