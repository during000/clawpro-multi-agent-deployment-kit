package controller

import (
	"context"
	"sync/atomic"
	"testing"

	tcerr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
)

// withCountingFakeVpcClient 替换 newVpcClientForSGFn 返回 fake，并统计工厂被调用次数（= 云 API 调用次数）。
func withCountingFakeVpcClient(t *testing.T, fake *fakeSGPoolVpcClient) (calls func() int32, restore func()) {
	t.Helper()
	orig := newVpcClientForSGFn
	var n int32
	newVpcClientForSGFn = func(ctx context.Context) (sgVpcClient, error) {
		atomic.AddInt32(&n, 1)
		return fake, nil
	}
	return func() int32 { return atomic.LoadInt32(&n) }, func() { newVpcClientForSGFn = orig }
}

const retrySampleRulesJSON = `[{"direction":"INGRESS","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`

func TestCreateCloudSGWithRetry(t *testing.T) {
	t.Run("success first attempt", func(t *testing.T) {
		withFastRetryParams(t)
		fake := &fakeSGPoolVpcClient{}
		calls, restore := withCountingFakeVpcClient(t, fake)
		defer restore()
		sgID, err := createCloudSGWithRetry(context.Background(), "name", "desc")
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if sgID != "sg-created" {
			t.Fatalf("expected sg-created, got %q", sgID)
		}
		if got := calls(); got != 1 {
			t.Fatalf("expected 1 call, got %d", got)
		}
	})
	t.Run("permanent error no retry", func(t *testing.T) {
		withFastRetryParams(t)
		fake := &fakeSGPoolVpcClient{createErr: &tcerr.TencentCloudSDKError{Code: "InvalidParameter", Message: "bad"}}
		calls, restore := withCountingFakeVpcClient(t, fake)
		defer restore()
		_, err := createCloudSGWithRetry(context.Background(), "name", "desc")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got := calls(); got != 1 {
			t.Fatalf("permanent error should not retry, expected 1 call, got %d", got)
		}
	})
	t.Run("retryable error exhausts", func(t *testing.T) {
		withFastRetryParams(t)
		fake := &fakeSGPoolVpcClient{createErr: &tcerr.TencentCloudSDKError{Code: "InternalError", Message: "boom"}}
		calls, restore := withCountingFakeVpcClient(t, fake)
		defer restore()
		_, err := createCloudSGWithRetry(context.Background(), "name", "desc")
		if err == nil {
			t.Fatal("expected error after exhaustion, got nil")
		}
		if got := calls(); got != 3 {
			t.Fatalf("expected 3 calls (max attempts), got %d", got)
		}
	})
}

func TestDescribeSGPoliciesWithRetry(t *testing.T) {
	t.Run("success first attempt", func(t *testing.T) {
		withFastRetryParams(t)
		fake := &fakeSGPoolVpcClient{}
		calls, restore := withCountingFakeVpcClient(t, fake)
		defer restore()
		_, err := DescribeSGPoliciesWithRetry(context.Background(), "sg-x")
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if got := calls(); got != 1 {
			t.Fatalf("expected 1 call, got %d", got)
		}
	})
	t.Run("permanent error no retry", func(t *testing.T) {
		withFastRetryParams(t)
		fake := &fakeSGPoolVpcClient{describePoliciesErr: &tcerr.TencentCloudSDKError{Code: "InvalidParameter", Message: "bad"}}
		calls, restore := withCountingFakeVpcClient(t, fake)
		defer restore()
		_, err := DescribeSGPoliciesWithRetry(context.Background(), "sg-x")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got := calls(); got != 1 {
			t.Fatalf("permanent error should not retry, expected 1 call, got %d", got)
		}
	})
	t.Run("retryable error exhausts", func(t *testing.T) {
		withFastRetryParams(t)
		fake := &fakeSGPoolVpcClient{describePoliciesErr: &tcerr.TencentCloudSDKError{Code: "InternalError", Message: "boom"}}
		calls, restore := withCountingFakeVpcClient(t, fake)
		defer restore()
		_, err := DescribeSGPoliciesWithRetry(context.Background(), "sg-x")
		if err == nil {
			t.Fatal("expected error after exhaustion, got nil")
		}
		if got := calls(); got != 3 {
			t.Fatalf("expected 3 calls, got %d", got)
		}
	})
}

func TestApplyRulesToCloudSGWithRetry(t *testing.T) {
	t.Run("success first attempt", func(t *testing.T) {
		withFastRetryParams(t)
		fake := &fakeSGPoolVpcClient{}
		calls, restore := withCountingFakeVpcClient(t, fake)
		defer restore()
		err := applyRulesToCloudSGWithRetry(context.Background(), "sg-x", retrySampleRulesJSON)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if got := calls(); got != 1 {
			t.Fatalf("expected 1 call, got %d", got)
		}
	})
	t.Run("permanent error no retry", func(t *testing.T) {
		withFastRetryParams(t)
		fake := &fakeSGPoolVpcClient{modifyErr: &tcerr.TencentCloudSDKError{Code: "InvalidParameter", Message: "bad"}}
		calls, restore := withCountingFakeVpcClient(t, fake)
		defer restore()
		err := applyRulesToCloudSGWithRetry(context.Background(), "sg-x", retrySampleRulesJSON)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got := calls(); got != 1 {
			t.Fatalf("permanent error should not retry, expected 1 call, got %d", got)
		}
	})
	t.Run("retryable error exhausts", func(t *testing.T) {
		withFastRetryParams(t)
		fake := &fakeSGPoolVpcClient{modifyErr: &tcerr.TencentCloudSDKError{Code: "InternalError", Message: "boom"}}
		calls, restore := withCountingFakeVpcClient(t, fake)
		defer restore()
		err := applyRulesToCloudSGWithRetry(context.Background(), "sg-x", retrySampleRulesJSON)
		if err == nil {
			t.Fatal("expected error after exhaustion, got nil")
		}
		if got := calls(); got != 3 {
			t.Fatalf("expected 3 calls, got %d", got)
		}
	})
}
