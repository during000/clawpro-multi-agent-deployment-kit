package controller

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ==================== 纯函数测试 ====================

func TestCalcVPCQuotaLevel(t *testing.T) {
	tests := []struct {
		name  string
		total uint64
		used  uint64
		want  string
	}{
		{"正常有余量", 20, 15, "info"},
		{"刚好耗尽", 20, 20, "critical"},
		{"超出上限", 20, 25, "critical"},
		{"上限为 0", 0, 0, "critical"},
		{"仅剩 1 个", 20, 19, "info"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcVPCQuotaLevel(tt.total, tt.used)
			if got != tt.want {
				t.Errorf("calcVPCQuotaLevel(%d, %d) = %q, want %q", tt.total, tt.used, got, tt.want)
			}
		})
	}
}

func TestCalcSubnetIPLevel(t *testing.T) {
	tests := []struct {
		name    string
		subnets []subnetIPItem
		want    string
	}{
		{
			"无子网",
			[]subnetIPItem{},
			"info",
		},
		{
			"全部有余量",
			[]subnetIPItem{
				{SubnetID: "subnet-1", AvailableIPCount: 100},
				{SubnetID: "subnet-2", AvailableIPCount: 200},
			},
			"info",
		},
		{
			"部分耗尽",
			[]subnetIPItem{
				{SubnetID: "subnet-1", AvailableIPCount: 0},
				{SubnetID: "subnet-2", AvailableIPCount: 100},
			},
			"info",
		},
		{
			"全部耗尽",
			[]subnetIPItem{
				{SubnetID: "subnet-1", AvailableIPCount: 0},
				{SubnetID: "subnet-2", AvailableIPCount: 0},
			},
			"critical",
		},
		{
			"单个子网耗尽",
			[]subnetIPItem{
				{SubnetID: "subnet-1", AvailableIPCount: 0},
			},
			"critical",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcSubnetIPLevel(tt.subnets)
			if got != tt.want {
				t.Errorf("calcSubnetIPLevel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCalcSGQuotaLevel(t *testing.T) {
	tests := []struct {
		name     string
		cvmCount uint64
		limit    uint64
		want     string
	}{
		{"远低于上限", 500, 2000, "info"},
		{"刚好达到上限", 2000, 2000, "critical"},
		{"超出上限", 2500, 2000, "critical"},
		{"仅差 1 个", 1999, 2000, "info"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcSGQuotaLevel(tt.cvmCount, tt.limit)
			if got != tt.want {
				t.Errorf("calcSGQuotaLevel(%d, %d) = %q, want %q", tt.cvmCount, tt.limit, got, tt.want)
			}
		})
	}
}

func TestCalcAccountArrearsLevel(t *testing.T) {
	tests := []struct {
		name         string
		isOverCredit bool
		want         string
	}{
		{"正常账号", false, "info"},
		{"超额", true, "critical"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcAccountArrearsLevel(tt.isOverCredit)
			if got != tt.want {
				t.Errorf("calcAccountArrearsLevel(%v) = %q, want %q",
					tt.isOverCredit, got, tt.want)
			}
		})
	}
}

// ==================== CheckAccountBalance 响应解析测试 ====================

func TestParseCheckAccountBalanceResp_Success(t *testing.T) {
	body := []byte(`{"Response":{"IsOverCredit":false,"IsOwed":false,"Uin":600000561284,"RequestId":"fbb81c94"}}`)
	got, err := parseCheckAccountBalanceResp(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.IsOwed != false || got.IsOverCredit != false || got.Uin != 600000561284 {
		t.Errorf("parsed result mismatch: %+v", got)
	}
}

func TestParseCheckAccountBalanceResp_Owed(t *testing.T) {
	body := []byte(`{"Response":{"IsOverCredit":true,"IsOwed":true,"Uin":123,"RequestId":"r"}}`)
	got, err := parseCheckAccountBalanceResp(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.IsOwed || !got.IsOverCredit || got.Uin != 123 {
		t.Errorf("parsed result mismatch: %+v", got)
	}
}

func TestParseCheckAccountBalanceResp_APIError(t *testing.T) {
	body := []byte(`{"Response":{"Error":{"Code":"AuthFailure","Message":"签名错误"},"RequestId":"r"}}`)
	_, err := parseCheckAccountBalanceResp(body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "AuthFailure") || !contains(err.Error(), "签名错误") {
		t.Errorf("error message should include API code/message, got: %v", err)
	}
}

func TestParseCheckAccountBalanceResp_InvalidJSON(t *testing.T) {
	_, err := parseCheckAccountBalanceResp([]byte("not a json"))
	if err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ==================== checkAccountArrears 集成测试（mock fetcher） ====================

func TestCheckAccountArrears_Normal(t *testing.T) {
	orig := accountBalanceFetcher
	defer func() { accountBalanceFetcher = orig }()

	accountBalanceFetcher = func(_ context.Context) ([]byte, error) {
		return []byte(`{"Response":{"IsOverCredit":false,"IsOwed":false,"Uin":1,"RequestId":"r"}}`), nil
	}
	item := checkAccountArrears(context.Background())
	if item == nil {
		t.Fatal("expected non-nil QuotaAlertItem")
	}
	if item.Level != alertLevelInfo {
		t.Errorf("expected info level, got %q", item.Level)
	}
	if item.Message != "" {
		t.Errorf("info level should have empty message, got %q", item.Message)
	}
	detail, ok := item.Detail.(accountArrearsDetail)
	if !ok {
		t.Fatalf("detail type mismatch: %T", item.Detail)
	}
	if detail.IsOwed || detail.IsOverCredit {
		t.Errorf("detail flags should be false: %+v", detail)
	}
}

func TestCheckAccountArrears_Owed(t *testing.T) {
	orig := accountBalanceFetcher
	defer func() { accountBalanceFetcher = orig }()

	// IsOwed=true 但 IsOverCredit=false，不应触发告警
	accountBalanceFetcher = func(_ context.Context) ([]byte, error) {
		return []byte(`{"Response":{"IsOverCredit":false,"IsOwed":true,"Uin":1,"RequestId":"r"}}`), nil
	}
	item := checkAccountArrears(context.Background())
	if item == nil {
		t.Fatal("expected non-nil QuotaAlertItem")
	}
	if item.Level != alertLevelInfo {
		t.Errorf("expected info level (IsOwed alone should not trigger), got %q", item.Level)
	}
}

func TestCheckAccountArrears_OverCredit(t *testing.T) {
	orig := accountBalanceFetcher
	defer func() { accountBalanceFetcher = orig }()

	accountBalanceFetcher = func(_ context.Context) ([]byte, error) {
		return []byte(`{"Response":{"IsOverCredit":true,"IsOwed":false,"Uin":1,"RequestId":"r"}}`), nil
	}
	item := checkAccountArrears(context.Background())
	if item == nil || item.Level != alertLevelCritical {
		t.Fatalf("expected critical alert, got %+v", item)
	}
	if !contains(item.Message, "欠费") {
		t.Errorf("message should mention 欠费, got %q", item.Message)
	}
}

func TestCheckAccountArrears_OwedAndOverCredit(t *testing.T) {
	orig := accountBalanceFetcher
	defer func() { accountBalanceFetcher = orig }()

	accountBalanceFetcher = func(_ context.Context) ([]byte, error) {
		return []byte(`{"Response":{"IsOverCredit":true,"IsOwed":true,"Uin":1,"RequestId":"r"}}`), nil
	}
	item := checkAccountArrears(context.Background())
	if item == nil || item.Level != alertLevelCritical {
		t.Fatalf("expected critical alert, got %+v", item)
	}
	if !contains(item.Message, "欠费") {
		t.Errorf("message should mention 欠费, got %q", item.Message)
	}
}

func TestCheckAccountArrears_FetcherError(t *testing.T) {
	orig := accountBalanceFetcher
	defer func() { accountBalanceFetcher = orig }()

	accountBalanceFetcher = func(_ context.Context) ([]byte, error) {
		return nil, errors.New("network timeout")
	}
	if item := checkAccountArrears(context.Background()); item != nil {
		t.Errorf("fetcher error should return nil, got %+v", item)
	}
}

func TestCheckAccountArrears_ParseError(t *testing.T) {
	orig := accountBalanceFetcher
	defer func() { accountBalanceFetcher = orig }()

	accountBalanceFetcher = func(_ context.Context) ([]byte, error) {
		return []byte(`{"Response":{"Error":{"Code":"AuthFailure","Message":"bad"}}}`), nil
	}
	if item := checkAccountArrears(context.Background()); item != nil {
		t.Errorf("parse error should return nil, got %+v", item)
	}
}

// ==================== 缓存测试 ====================

func TestQuotaAlertCacheExpiry(t *testing.T) {
	cache := &quotaAlertCache{}
	if got := cache.get(); got != nil {
		t.Error("空缓存 get() 应返回 nil")
	}

	cache.set([]QuotaAlertItem{{ID: "test", Level: "info"}})
	if got := cache.get(); got == nil {
		t.Error("刚写入的缓存 get() 不应返回 nil")
	}

	// 模拟过期
	cache.mu.Lock()
	cache.expires = time.Now().Add(-1 * time.Second)
	cache.mu.Unlock()
	if got := cache.get(); got != nil {
		t.Error("过期缓存 get() 应返回 nil")
	}
}

func TestQuotaAlertCacheCopy(t *testing.T) {
	cache := &quotaAlertCache{}
	alerts := []QuotaAlertItem{
		{ID: "vpc", Level: "critical"},
		{ID: "subnet", Level: "info"},
	}
	cache.set(alerts)

	got := cache.get()
	if len(got) != 2 || got[0].ID != "vpc" || got[1].ID != "subnet" {
		t.Errorf("缓存内容不匹配: %+v", got)
	}

	// 修改返回值不应影响缓存
	got[0].Level = "modified"
	got2 := cache.get()
	if got2[0].Level != "critical" {
		t.Error("修改 get() 返回值不应影响缓存内容")
	}
}
