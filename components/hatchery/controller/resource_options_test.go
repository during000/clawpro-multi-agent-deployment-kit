package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/model"

	cbs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs/v20170312"
	sdkcommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// ──────────────────────────────────────────────
// 测试辅助
// ──────────────────────────────────────────────

func setupResourceOptionsTest(t *testing.T) {
	t.Helper()
	resourceOptionsCache = sync.Map{}
	inflightMap = sync.Map{}
}

func teardownResourceOptionsTest(t *testing.T) {
	t.Helper()
	resourceOptionsCache = sync.Map{}
	inflightMap = sync.Map{}
}

func mockCVMClientWith(fn func(*cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error)) *mockCVMClientOpt {
	return &mockCVMClientOpt{fn: fn}
}

type mockCVMClientOpt struct {
	fn func(*cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error)
}

func (m *mockCVMClientOpt) DescribeZoneInstanceConfigInfos(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
	return m.fn(req)
}

func mockCBSClientWith(fn func(*cbs.DescribeDiskConfigQuotaRequest) (*cbs.DescribeDiskConfigQuotaResponse, error)) *mockCBSClientOpt {
	return &mockCBSClientOpt{fn: fn}
}

type mockCBSClientOpt struct {
	fn func(*cbs.DescribeDiskConfigQuotaRequest) (*cbs.DescribeDiskConfigQuotaResponse, error)
}

func (m *mockCBSClientOpt) DescribeDiskConfigQuota(req *cbs.DescribeDiskConfigQuotaRequest) (*cbs.DescribeDiskConfigQuotaResponse, error) {
	return m.fn(req)
}

// ──────────────────────────────────────────────
// basic Chinese response
// ──────────────────────────────────────────────

// ──────────────────────────────────────────────
// basic English response
// ──────────────────────────────────────────────

// ──────────────────────────────────────────────
// instance-types parameters reject missing or invalid values
// ──────────────────────────────────────────────

func TestResourceOptionsInstanceTypes_BadParams(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantSubstr string
	}{
		{"missing zone", "/admin/resource-policies/options/instance-types", http.StatusBadRequest, "zone"},
		{"invalid charge type", "/admin/resource-policies/options/instance-types?zone=ap-guangzhou-1&instance_charge_type=INVALID", http.StatusBadRequest, "instance_charge_type"},
		{"valid charge type PREPAID", "/admin/resource-policies/options/instance-types?zone=ap-guangzhou-1&instance_charge_type=PREPAID", 0, ""}, // 0 = don't assert status
		{"valid charge type POSTPAID", "/admin/resource-policies/options/instance-types?zone=ap-guangzhou-1&instance_charge_type=POSTPAID_BY_HOUR", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use unexported helper directly with mock client to skip auth
			mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
				return &cvm.DescribeZoneInstanceConfigInfosResponse{
					Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{},
				}, nil
			})

			req := httptest.NewRequest(http.MethodGet, tt.query, nil)
			zone := req.URL.Query().Get("zone")
			chargeType := req.URL.Query().Get("instance_charge_type")
			refresh := req.URL.Query().Get("refresh") == "1"
			rr := httptest.NewRecorder()

			handleResourceOptionsInstanceTypes(context.Background(), rr, req, mockCVM, zone, chargeType, refresh)

			if tt.wantStatus > 0 && rr.Code != tt.wantStatus {
				t.Errorf("status=%d, want %d", rr.Code, tt.wantStatus)
			}
			if tt.wantSubstr != "" && !strings.Contains(rr.Body.String(), tt.wantSubstr) {
				t.Errorf("body missing %q: %s", tt.wantSubstr, rr.Body.String())
			}
		})
	}
}

// ──────────────────────────────────────────────
// instance-types cloud results apply SELL and allowlist filters
// ──────────────────────────────────────────────

func TestResourceOptionsInstanceTypes_SellAndAllowlistFilter(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	callCount := 0
	mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		callCount++
		return &cvm.DescribeZoneInstanceConfigInfosResponse{
			Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
				InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{
					{Status: sdkcommon.StringPtr("SELL"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"), Cpu: sdkcommon.Int64Ptr(2), Memory: sdkcommon.Int64Ptr(4)},
					{Status: sdkcommon.StringPtr("SELL"), InstanceType: sdkcommon.StringPtr("S5.MEDIUM4"), Cpu: sdkcommon.Int64Ptr(2), Memory: sdkcommon.Int64Ptr(4)}, // non-allowlist
					{Status: sdkcommon.StringPtr("OUT_OF_STOCK"), InstanceType: sdkcommon.StringPtr("Ai2.LARGE8"), Cpu: sdkcommon.Int64Ptr(4), Memory: sdkcommon.Int64Ptr(8)},
					{Status: sdkcommon.StringPtr("SELL"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM4"), Cpu: sdkcommon.Int64Ptr(4), Memory: sdkcommon.Int64Ptr(8)},
				},
			},
		}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone=ap-guangzhou-1", nil)
	rr := httptest.NewRecorder()

	handleResourceOptionsInstanceTypes(context.Background(), rr, req, mockCVM, "ap-guangzhou-1", "", false)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	items := resp["instance_types"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("items count=%d, want 2 (only Ai2.MEDIUM2 and Ai2.MEDIUM4)", len(items))
	}

	types := make(map[string]bool)
	for _, item := range items {
		types[item.(map[string]interface{})["instance_type"].(string)] = true
	}
	if !types["Ai2.MEDIUM2"] {
		t.Error("missing Ai2.MEDIUM2")
	}
	if !types["Ai2.MEDIUM4"] {
		t.Error("missing Ai2.MEDIUM4")
	}
	if types["S5.MEDIUM4"] || types["Ai2.LARGE8"] {
		t.Error("should have filtered non-allowlist and non-SELL")
	}
	if callCount != 1 {
		t.Errorf("callCount=%d, want 1", callCount)
	}
}

// ──────────────────────────────────────────────
// system-disks parameters reject missing or invalid values
// ──────────────────────────────────────────────

func TestResourceOptionsSystemDisks_BadParams(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	callCount := 0
	mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		callCount++
		return &cvm.DescribeZoneInstanceConfigInfosResponse{
			Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{},
		}, nil
	})
	mockCBS := mockCBSClientWith(func(req *cbs.DescribeDiskConfigQuotaRequest) (*cbs.DescribeDiskConfigQuotaResponse, error) {
		callCount++
		return &cbs.DescribeDiskConfigQuotaResponse{}, nil
	})

	tests := []struct {
		name         string
		zone         string
		chargeType   string
		instanceType string
		wantStatus   int
		wantSubstr   string
	}{
		{"missing zone", "", "", "Ai2.MEDIUM2", http.StatusBadRequest, "zone"},
		{"missing instance_type", "ap-guangzhou-1", "", "", http.StatusBadRequest, "instance_type"},
		{"invalid charge type", "ap-guangzhou-1", "INVALID", "Ai2.MEDIUM2", http.StatusBadRequest, "instance_charge_type"},
		{"non-allowlist instance_type", "ap-guangzhou-1", "", "S5.LARGE8", http.StatusBadRequest, "instance_type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://test/options/system-disks?zone=%s&instance_charge_type=%s&instance_type=%s", tt.zone, tt.chargeType, tt.instanceType), nil)
			rr := httptest.NewRecorder()

			handleResourceOptionsSystemDisks(context.Background(), rr, req, mockCVM, mockCBS, tt.zone, tt.chargeType, tt.instanceType, false)

			if rr.Code != tt.wantStatus {
				t.Errorf("status=%d, want %d", rr.Code, tt.wantStatus)
			}
			if tt.wantSubstr != "" && !strings.Contains(rr.Body.String(), tt.wantSubstr) {
				t.Errorf("body missing %q: %s", tt.wantSubstr, rr.Body.String())
			}
		})
	}
}

// ──────────────────────────────────────────────
// system-disks cloud results filter by availability, usage, allowlist, and minimum size
// ──────────────────────────────────────────────

func TestResourceOptionsSystemDisks_CloudFilter(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		return &cvm.DescribeZoneInstanceConfigInfosResponse{
			Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
				InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{
					{Status: sdkcommon.StringPtr("SELL"), InstanceFamily: sdkcommon.StringPtr("Ai2"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"), Cpu: sdkcommon.Int64Ptr(2), Memory: sdkcommon.Int64Ptr(4)},
				},
			},
		}, nil
	})

	mockCBS := mockCBSClientWith(func(req *cbs.DescribeDiskConfigQuotaRequest) (*cbs.DescribeDiskConfigQuotaResponse, error) {
		return &cbs.DescribeDiskConfigQuotaResponse{
			Response: &cbs.DescribeDiskConfigQuotaResponseParams{
				DiskConfigSet: []*cbs.DiskConfig{
					// Good: available, SYSTEM_DISK, allowlist type, valid range
					{Available: sdkcommon.BoolPtr(true), DiskUsage: sdkcommon.StringPtr("SYSTEM_DISK"), DiskType: sdkcommon.StringPtr("CLOUD_SSD"), MinDiskSize: sdkcommon.Uint64Ptr(20), MaxDiskSize: sdkcommon.Uint64Ptr(500), StepSize: sdkcommon.Uint64Ptr(10)},
					// Not available
					{Available: sdkcommon.BoolPtr(false), DiskUsage: sdkcommon.StringPtr("SYSTEM_DISK"), DiskType: sdkcommon.StringPtr("CLOUD_PREMIUM"), MinDiskSize: sdkcommon.Uint64Ptr(20), MaxDiskSize: sdkcommon.Uint64Ptr(500)},
					// Non-allowlist type
					{Available: sdkcommon.BoolPtr(true), DiskUsage: sdkcommon.StringPtr("SYSTEM_DISK"), DiskType: sdkcommon.StringPtr("CLOUD_UNKNOWN"), MinDiskSize: sdkcommon.Uint64Ptr(20), MaxDiskSize: sdkcommon.Uint64Ptr(500)},
					// Cloud minimum is returned unchanged.
					{Available: sdkcommon.BoolPtr(true), DiskUsage: sdkcommon.StringPtr("SYSTEM_DISK"), DiskType: sdkcommon.StringPtr("CLOUD_HSSD"), MinDiskSize: sdkcommon.Uint64Ptr(10), MaxDiskSize: sdkcommon.Uint64Ptr(200)},
				},
			},
		}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "http://test/options/system-disks?zone=ap-guangzhou-1&instance_type=Ai2.MEDIUM2", nil)
	rr := httptest.NewRecorder()

	handleResourceOptionsSystemDisks(context.Background(), rr, req, mockCVM, mockCBS, "ap-guangzhou-1", "", "Ai2.MEDIUM2", false)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200 body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	items := resp["system_disk_options"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("items count=%d, want 2 (CLOUD_SSD + CLOUD_HSSD)", len(items))
	}

	diskTypes := make(map[string]int64)
	for _, item := range items {
		m := item.(map[string]interface{})
		dt := m["disk_type"].(string)
		minSize := int64(m["min_disk_size"].(float64))
		diskTypes[dt] = minSize
	}

	if diskTypes["CLOUD_SSD"] != 20 {
		t.Errorf("CLOUD_SSD min=%d, want cloud minimum 20", diskTypes["CLOUD_SSD"])
	}
	if diskTypes["CLOUD_HSSD"] != 10 {
		t.Errorf("CLOUD_HSSD min=%d, want cloud minimum 10", diskTypes["CLOUD_HSSD"])
	}
}

// ──────────────────────────────────────────────
// cache miss reports source=tencent_cloud
// ──────────────────────────────────────────────

func TestResourceOptions_FreshCacheSource(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		return &cvm.DescribeZoneInstanceConfigInfosResponse{
			Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
				InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{
					{Status: sdkcommon.StringPtr("SELL"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"), Cpu: sdkcommon.Int64Ptr(2), Memory: sdkcommon.Int64Ptr(4)},
				},
			},
		}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone=ap-guangzhou-1", nil)
	rr := httptest.NewRecorder()

	handleResourceOptionsInstanceTypes(context.Background(), rr, req, mockCVM, "ap-guangzhou-1", "", false)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["source"] != "tencent_cloud" {
		t.Errorf("fresh source=%v, want tencent_cloud", resp["source"])
	}
	if resp["refreshed_at"] == nil {
		t.Error("fresh must have refreshed_at")
	}
}

// ──────────────────────────────────────────────
// cache hit reports source=cache
// ──────────────────────────────────────────────

func TestResourceOptions_CacheHitSource(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	callCount := 0
	mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		callCount++
		return &cvm.DescribeZoneInstanceConfigInfosResponse{
			Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
				InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{
					{Status: sdkcommon.StringPtr("SELL"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"), Cpu: sdkcommon.Int64Ptr(2), Memory: sdkcommon.Int64Ptr(4)},
				},
			},
		}, nil
	})

	// First request — populates cache
	req1 := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone=ap-guangzhou-1", nil)
	rr1 := httptest.NewRecorder()
	handleResourceOptionsInstanceTypes(context.Background(), rr1, req1, mockCVM, "ap-guangzhou-1", "", false)

	// Second request — should hit cache
	req2 := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone=ap-guangzhou-1", nil)
	rr2 := httptest.NewRecorder()
	handleResourceOptionsInstanceTypes(context.Background(), rr2, req2, mockCVM, "ap-guangzhou-1", "", false)

	var resp map[string]interface{}
	json.Unmarshal(rr2.Body.Bytes(), &resp)

	if resp["source"] != "cache" {
		t.Errorf("cache hit source=%v, want cache", resp["source"])
	}
	if callCount != 1 {
		t.Errorf("callCount=%d, want 1 (second should be cache hit)", callCount)
	}
}

// ──────────────────────────────────────────────
// refresh bypasses cache
// ──────────────────────────────────────────────

func TestResourceOptions_RefreshBypassCache(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	callCount := 0
	mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		callCount++
		return &cvm.DescribeZoneInstanceConfigInfosResponse{
			Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
				InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{
					{Status: sdkcommon.StringPtr("SELL"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"), Cpu: sdkcommon.Int64Ptr(2), Memory: sdkcommon.Int64Ptr(4)},
				},
			},
		}, nil
	})

	// Populate cache
	req := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone=ap-guangzhou-1", nil)
	rr := httptest.NewRecorder()
	handleResourceOptionsInstanceTypes(context.Background(), rr, req, mockCVM, "ap-guangzhou-1", "", false)

	resetCache := func() {
		// Don't actually reset — we want to test refresh bypassing cache
	}

	_ = resetCache
	firstCalls := callCount

	// Refresh request
	req2 := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone=ap-guangzhou-1&refresh=1", nil)
	rr2 := httptest.NewRecorder()
	handleResourceOptionsInstanceTypes(context.Background(), rr2, req2, mockCVM, "ap-guangzhou-1", "", true)

	var resp map[string]interface{}
	json.Unmarshal(rr2.Body.Bytes(), &resp)

	if resp["source"] != "tencent_cloud" {
		t.Errorf("refresh source=%v, want tencent_cloud (re-fetched)", resp["source"])
	}
	if callCount <= firstCalls {
		t.Errorf("callCount=%d, should have made a new SDK call for refresh", callCount)
	}
}

// ──────────────────────────────────────────────
// cache keys are tenant-isolated
// ──────────────────────────────────────────────

func TestResourceOptions_TenantIsolation(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	callsForA := 0
	callsForB := 0

	mockCVMA := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		callsForA++
		return &cvm.DescribeZoneInstanceConfigInfosResponse{
			Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
				InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{
					{Status: sdkcommon.StringPtr("SELL"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"), Cpu: sdkcommon.Int64Ptr(2), Memory: sdkcommon.Int64Ptr(4)},
				},
			},
		}, nil
	})
	mockCVMB := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		callsForB++
		return &cvm.DescribeZoneInstanceConfigInfosResponse{
			Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
				InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{
					{Status: sdkcommon.StringPtr("SELL"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM4"), Cpu: sdkcommon.Int64Ptr(4), Memory: sdkcommon.Int64Ptr(8)},
				},
			},
		}, nil
	})

	zone := "ap-guangzhou-1"

	// Tenant A request
	ctxA := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Identifier: "tenant-a"})
	reqA := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone="+zone, nil)
	rrA := httptest.NewRecorder()
	handleResourceOptionsInstanceTypes(ctxA, rrA, reqA, mockCVMA, zone, "", false)

	// Tenant A second request — cache hit
	reqA2 := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone="+zone, nil)
	rrA2 := httptest.NewRecorder()
	handleResourceOptionsInstanceTypes(ctxA, rrA2, reqA2, mockCVMA, zone, "", false)

	var respA2 map[string]interface{}
	json.Unmarshal(rrA2.Body.Bytes(), &respA2)
	if respA2["source"] != "cache" {
		t.Errorf("tenant A second request source=%v, want cache", respA2["source"])
	}

	// Tenant B request — must NOT share cache with Tenant A
	ctxB := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Identifier: "tenant-b"})
	reqB := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone="+zone, nil)
	rrB := httptest.NewRecorder()
	handleResourceOptionsInstanceTypes(ctxB, rrB, reqB, mockCVMB, zone, "", false)

	var respB map[string]interface{}
	json.Unmarshal(rrB.Body.Bytes(), &respB)
	if respB["source"] != "tencent_cloud" {
		t.Errorf("tenant B first request source=%v, want tencent_cloud (should miss tenant A's cache)", respB["source"])
	}
	if callsForA != 1 {
		t.Errorf("tenant A calls=%d, want 1", callsForA)
	}
	if callsForB != 1 {
		t.Errorf("tenant B calls=%d, want 1 (separate from A)", callsForB)
	}
}

// ──────────────────────────────────────────────
// expired entries miss the cache
// ──────────────────────────────────────────────

func TestResourceOptions_TTLExpiry(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	callCount := 0
	mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		callCount++
		return &cvm.DescribeZoneInstanceConfigInfosResponse{
			Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
				InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{
					{Status: sdkcommon.StringPtr("SELL"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"), Cpu: sdkcommon.Int64Ptr(2), Memory: sdkcommon.Int64Ptr(4)},
				},
			},
		}, nil
	})

	zone := "ap-guangzhou-1"
	cacheKey := resourceOptionsCacheKey(context.Background(), "instance-types", zone, "")

	// Populate cache with stale entry
	resourceOptionsCache.Store(cacheKey, &optionsCacheEntry{
		Payload: &optionsCachePayload{
			Data:        json.RawMessage(`[]`),
			RefreshedAt: time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339),
		},
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	})

	req := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone="+zone, nil)
	rr := httptest.NewRecorder()
	handleResourceOptionsInstanceTypes(context.Background(), rr, req, mockCVM, zone, "", false)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["source"] != "tencent_cloud" {
		t.Errorf("expired TTL source=%v, want tencent_cloud (miss)", resp["source"])
	}
	if callCount != 1 {
		t.Errorf("callCount=%d, want 1 (expired entry should miss)", callCount)
	}

	if _, ok := resourceOptionsCache.Load(cacheKey); !ok {
		t.Error("expired cache entry was not replaced with a fresh entry")
	}
}

// ──────────────────────────────────────────────
// concurrent requests for one key share inflight work
// ──────────────────────────────────────────────

func TestResourceOptions_InflightDedup(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	var mu sync.Mutex
	callCount := 0

	mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		return &cvm.DescribeZoneInstanceConfigInfosResponse{
			Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
				InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{
					{Status: sdkcommon.StringPtr("SELL"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"), Cpu: sdkcommon.Int64Ptr(2), Memory: sdkcommon.Int64Ptr(4)},
				},
			},
		}, nil
	})

	zone := "ap-guangzhou-1"
	errCh := make(chan error, 3)

	// Launch 3 concurrent requests
	for i := 0; i < 3; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone="+zone, nil)
			rr := httptest.NewRecorder()
			handleResourceOptionsInstanceTypes(context.Background(), rr, req, mockCVM, zone, "", false)
			if rr.Code != http.StatusOK {
				errCh <- fmt.Errorf("status=%d", rr.Code)
				return
			}
			var resp map[string]interface{}
			json.Unmarshal(rr.Body.Bytes(), &resp)
			errCh <- nil
		}()
	}

	for i := 0; i < 3; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("concurrent request %d: %v", i, err)
		}
	}

	if callCount != 1 {
		t.Errorf("callCount=%d, want 1 (only winner should call SDK)", callCount)
	}

	// Verify no leak: inflightMap should be empty after winner completes
	inflightMap.Range(func(key, value interface{}) bool {
		t.Errorf("inflightMap leaked key=%v", key)
		return false
	})
}

// ──────────────────────────────────────────────
// failed winners are not cached and propagate errors to waiters
// ──────────────────────────────────────────────

func TestResourceOptions_WinnerFailure(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	zone := "ap-guangzhou-1"
	cacheKey := resourceOptionsCacheKey(context.Background(), "instance-types", zone, "")
	sdkEntered := make(chan struct{})
	releaseSDK := make(chan struct{})
	var enteredOnce sync.Once
	var callMu sync.Mutex
	callCount := 0
	mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		callMu.Lock()
		callCount++
		callMu.Unlock()
		enteredOnce.Do(func() {
			close(sdkEntered)
			<-releaseSDK
		})
		return nil, fmt.Errorf("SDK error")
	})

	start := make(chan struct{})
	errCh := make(chan int, 2)
	for range 2 {
		go func() {
			<-start
			req := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone="+zone, nil)
			rr := httptest.NewRecorder()
			handleResourceOptionsInstanceTypes(context.Background(), rr, req, mockCVM, zone, "", false)
			errCh <- rr.Code
		}()
	}
	close(start)
	<-sdkEntered
	time.Sleep(20 * time.Millisecond)
	close(releaseSDK)

	codes := []int{<-errCh, <-errCh}
	for _, code := range codes {
		if code != http.StatusInternalServerError {
			t.Fatalf("winner failure should return controlled 500 to both requests, got codes=%v", codes)
		}
	}

	callMu.Lock()
	gotCalls := callCount
	callMu.Unlock()
	if gotCalls != 1 {
		t.Errorf("callCount=%d, want 1 (only winner attempts)", gotCalls)
	}
	if _, ok := resourceOptionsCache.Load(cacheKey); ok {
		t.Error("cache was populated despite winner failure")
	}
}

// ──────────────────────────────────────────────
// SDK wrapper handles CVM success and failure
// ──────────────────────────────────────────────

func TestResourceOptions_SDKWrapper_Success(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		return &cvm.DescribeZoneInstanceConfigInfosResponse{
			Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
				InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{
					{Status: sdkcommon.StringPtr("SELL"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"), Cpu: sdkcommon.Int64Ptr(2), Memory: sdkcommon.Int64Ptr(4)},
				},
			},
		}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone=ap-guangzhou-1", nil)
	rr := httptest.NewRecorder()
	handleResourceOptionsInstanceTypes(context.Background(), rr, req, mockCVM, "ap-guangzhou-1", "", false)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}
}

func TestResourceOptions_SDKWrapper_Failure(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		return nil, fmt.Errorf("simulated CVM error")
	})

	req := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone=ap-guangzhou-1", nil)
	rr := httptest.NewRecorder()
	handleResourceOptionsInstanceTypes(context.Background(), rr, req, mockCVM, "ap-guangzhou-1", "", false)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status=%d, want 500", rr.Code)
	}
}

func TestResourceOptions_SDKWrapper_CBSSuccess(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		return &cvm.DescribeZoneInstanceConfigInfosResponse{
			Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
				InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{
					{Status: sdkcommon.StringPtr("SELL"), InstanceFamily: sdkcommon.StringPtr("Ai2"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"), Cpu: sdkcommon.Int64Ptr(2), Memory: sdkcommon.Int64Ptr(4)},
				},
			},
		}, nil
	})

	mockCBS := mockCBSClientWith(func(req *cbs.DescribeDiskConfigQuotaRequest) (*cbs.DescribeDiskConfigQuotaResponse, error) {
		return &cbs.DescribeDiskConfigQuotaResponse{
			Response: &cbs.DescribeDiskConfigQuotaResponseParams{
				DiskConfigSet: []*cbs.DiskConfig{
					{Available: sdkcommon.BoolPtr(true), DiskUsage: sdkcommon.StringPtr("SYSTEM_DISK"), DiskType: sdkcommon.StringPtr("CLOUD_SSD"), MinDiskSize: sdkcommon.Uint64Ptr(50), MaxDiskSize: sdkcommon.Uint64Ptr(500)},
				},
			},
		}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "http://test/options/system-disks?zone=ap-guangzhou-1&instance_type=Ai2.MEDIUM2", nil)
	rr := httptest.NewRecorder()
	handleResourceOptionsSystemDisks(context.Background(), rr, req, mockCVM, mockCBS, "ap-guangzhou-1", "", "Ai2.MEDIUM2", false)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200 body=%s", rr.Code, rr.Body.String())
	}
}

func TestResourceOptions_SDKWrapper_CBSFailure(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		return &cvm.DescribeZoneInstanceConfigInfosResponse{
			Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
				InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{
					{Status: sdkcommon.StringPtr("SELL"), InstanceFamily: sdkcommon.StringPtr("Ai2"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"), Cpu: sdkcommon.Int64Ptr(2), Memory: sdkcommon.Int64Ptr(4)},
				},
			},
		}, nil
	})

	mockCBS := mockCBSClientWith(func(req *cbs.DescribeDiskConfigQuotaRequest) (*cbs.DescribeDiskConfigQuotaResponse, error) {
		return nil, fmt.Errorf("simulated CBS error")
	})

	req := httptest.NewRequest(http.MethodGet, "http://test/options/system-disks?zone=ap-guangzhou-1&instance_type=Ai2.MEDIUM2", nil)
	rr := httptest.NewRecorder()
	handleResourceOptionsSystemDisks(context.Background(), rr, req, mockCVM, mockCBS, "ap-guangzhou-1", "", "Ai2.MEDIUM2", false)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status=%d, want 500", rr.Code)
	}
}

// ──────────────────────────────────────────────
// Options endpoint cache seam regression.
// ──────────────────────────────────────────────

func TestResourceOptions_EndpointCacheSeam(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	// Populate cache with instance types data
	zone := "ap-guangzhou-1"
	cacheKey := resourceOptionsCacheKey(context.Background(), "instance-types", zone, "")
	data, _ := json.Marshal([]instanceTypeItem{
		{InstanceType: "Ai2.MEDIUM2", CPU: 2, Memory: 4},
	})
	resourceOptionsCache.Store(cacheKey, &optionsCacheEntry{
		Payload: &optionsCachePayload{
			Data:        data,
			RefreshedAt: time.Now().UTC().Format(time.RFC3339),
		},
		ExpiresAt: time.Now().Add(optionsCacheTTL),
	})

	// Verify cache hit returns source=cache
	req := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone="+zone, nil)
	rr := httptest.NewRecorder()

	callCount := 0
	mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		callCount++
		return &cvm.DescribeZoneInstanceConfigInfosResponse{}, nil
	})

	handleResourceOptionsInstanceTypes(context.Background(), rr, req, mockCVM, zone, "", false)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["source"] != "cache" {
		t.Errorf("cached preflight source=%v, want cache", resp["source"])
	}
	if callCount != 0 {
		t.Errorf("should not have called SDK on cache hit")
	}
}

// ──────────────────────────────────────────────
// Options endpoint cache-miss regression.
// ──────────────────────────────────────────────

func TestResourceOptions_EndpointCacheMiss(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	zone := "ap-guangzhou-1"

	t.Run("seller available", func(t *testing.T) {
		setupResourceOptionsTest(t)
		defer teardownResourceOptionsTest(t)

		mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
			return &cvm.DescribeZoneInstanceConfigInfosResponse{
				Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
					InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{
						{Status: sdkcommon.StringPtr("SELL"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"), Cpu: sdkcommon.Int64Ptr(2), Memory: sdkcommon.Int64Ptr(4)},
					},
				},
			}, nil
		})

		req := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone="+zone, nil)
		rr := httptest.NewRecorder()
		handleResourceOptionsInstanceTypes(context.Background(), rr, req, mockCVM, zone, "", false)

		if rr.Code != http.StatusOK {
			t.Errorf("SELL hit: status=%d, want 200", rr.Code)
		}
	})

	t.Run("non-sell returns empty", func(t *testing.T) {
		setupResourceOptionsTest(t)
		defer teardownResourceOptionsTest(t)

		mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
			return &cvm.DescribeZoneInstanceConfigInfosResponse{
				Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
					InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{
						{Status: sdkcommon.StringPtr("OUT_OF_STOCK"), InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"), Cpu: sdkcommon.Int64Ptr(2), Memory: sdkcommon.Int64Ptr(4)},
					},
				},
			}, nil
		})

		req := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone="+zone, nil)
		rr := httptest.NewRecorder()
		handleResourceOptionsInstanceTypes(context.Background(), rr, req, mockCVM, zone, "", false)

		if rr.Code != http.StatusOK {
			t.Errorf("non-SELL: status=%d, want 200 (empty list)", rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		items := resp["instance_types"].([]interface{})
		if len(items) != 0 {
			t.Errorf("non-SELL should return empty list, got %d items", len(items))
		}
	})

	t.Run("sdk error returns 500", func(t *testing.T) {
		setupResourceOptionsTest(t)
		defer teardownResourceOptionsTest(t)

		mockCVM := mockCVMClientWith(func(req *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
			return nil, fmt.Errorf("cloud error")
		})

		req := httptest.NewRequest(http.MethodGet, "http://test/options/instance-types?zone="+zone, nil)
		rr := httptest.NewRecorder()
		handleResourceOptionsInstanceTypes(context.Background(), rr, req, mockCVM, zone, "", false)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("SDK error: status=%d, want 500", rr.Code)
		}
	})
}

// Create preflight uses cached instance options without calling CVM.
func TestValidateCreateResourceConfig_UsesCachedOptions(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	ctx := context.Background()
	zone := "ap-guangzhou-1"
	data, err := json.Marshal([]instanceTypeItem{{InstanceType: "Ai2.MEDIUM2", CPU: 2, Memory: 4}})
	if err != nil {
		t.Fatalf("marshal cached instance types: %v", err)
	}
	optionsCacheSet(resourceOptionsCacheKey(ctx, "instance-types", zone, ""), &optionsCachePayload{
		Data:        data,
		RefreshedAt: time.Now().UTC().Format(time.RFC3339),
	})

	originalGetClient := getCVMOptionsClientFn
	clientCalls := 0
	getCVMOptionsClientFn = func(context.Context) (cvmOptionsClient, error) {
		clientCalls++
		return nil, fmt.Errorf("CVM client must not be requested on cache hit")
	}
	defer func() { getCVMOptionsClientFn = originalGetClient }()

	if err := validateCreateResourceConfig(ctx, zone, "", "Ai2.MEDIUM2"); err != nil {
		t.Fatalf("cached available instance type should pass: %v", err)
	}
	if err := validateCreateResourceConfig(ctx, zone, "", "Ai2.LARGE8"); err == nil {
		t.Fatal("instance type absent from a valid cache entry should be rejected")
	}
	if clientCalls != 0 {
		t.Fatalf("cache hit requested CVM client %d times, want 0", clientCalls)
	}
}

// Create preflight validates cloud options on cache miss and fails closed.
func TestValidateCreateResourceConfig_ValidatesCloudOptionsOnCacheMiss(t *testing.T) {
	originalGetClient := getCVMOptionsClientFn
	defer func() { getCVMOptionsClientFn = originalGetClient }()

	tests := []struct {
		name      string
		status    string
		sdkErr    error
		wantError bool
	}{
		{name: "SELL passes", status: "SELL"},
		{name: "non-SELL fails closed", status: "OUT_OF_STOCK", wantError: true},
		{name: "SDK error fails closed", sdkErr: fmt.Errorf("cloud error"), wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupResourceOptionsTest(t)
			defer teardownResourceOptionsTest(t)

			clientCalls := 0
			getCVMOptionsClientFn = func(context.Context) (cvmOptionsClient, error) {
				clientCalls++
				return mockCVMClientWith(func(*cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
					if tt.sdkErr != nil {
						return nil, tt.sdkErr
					}
					return &cvm.DescribeZoneInstanceConfigInfosResponse{
						Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
							InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{{
								Status:       sdkcommon.StringPtr(tt.status),
								InstanceType: sdkcommon.StringPtr("Ai2.MEDIUM2"),
							}},
						},
					}, nil
				}), nil
			}

			err := validateCreateResourceConfig(context.Background(), "ap-guangzhou-1", "POSTPAID_BY_HOUR", "Ai2.MEDIUM2")
			if (err != nil) != tt.wantError {
				t.Fatalf("validateCreateResourceConfig() error = %v, wantError %v", err, tt.wantError)
			}
			if clientCalls != 1 {
				t.Fatalf("CVM client requested %d times, want 1", clientCalls)
			}
		})
	}
}

func TestValidateCreateResourceConfig_EmptyInstanceTypeSkips(t *testing.T) {
	originalGetClient := getCVMOptionsClientFn
	clientCalled := false
	getCVMOptionsClientFn = func(context.Context) (cvmOptionsClient, error) {
		clientCalled = true
		return nil, fmt.Errorf("CVM client must not be requested for an empty instance type")
	}
	defer func() { getCVMOptionsClientFn = originalGetClient }()

	if err := validateCreateResourceConfig(context.Background(), "ap-guangzhou-1", "", ""); err != nil {
		t.Fatalf("empty instance type should skip preflight: %v", err)
	}
	if clientCalled {
		t.Fatal("empty instance type requested a CVM client")
	}
}

// ──────────────────────────────────────────────
// 缓存 key 辅助函数测试
// ──────────────────────────────────────────────

func TestResourceOptionsCacheKey(t *testing.T) {
	ctx := context.Background()

	key1 := resourceOptionsCacheKey(ctx, "instance-types", "ap-guangzhou-1", "PREPAID")
	key2 := resourceOptionsCacheKey(ctx, "instance-types", "ap-guangzhou-1", "POSTPAID_BY_HOUR")

	if key1 == key2 {
		t.Errorf("different charge types should produce different keys: %q", key1)
	}

	key3 := resourceOptionsCacheKey(ctx, "instance-types", "ap-guangzhou-1", "PREPAID")
	if key1 != key3 {
		t.Errorf("same params should produce same key: %q vs %q", key1, key3)
	}

	// System disks has more scope parts
	key4 := resourceOptionsCacheKey(ctx, "system-disks", "ap-guangzhou-1", "PREPAID", "Ai2.MEDIUM2")
	if key1 == key4 {
		t.Errorf("different endpoints should produce different keys")
	}

	// Verify keys contain expected parts
	if !strings.Contains(key1, "instance-types") {
		t.Errorf("key1 missing endpoint: %q", key1)
	}
	if !strings.Contains(key1, CVMRegion) {
		t.Errorf("key1 missing CVMRegion: %q", key1)
	}
}

// ──────────────────────────────────────────────
// charge type 校验
// ──────────────────────────────────────────────

func TestIsValidInstanceChargeType(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"PREPAID", true},
		{"POSTPAID_BY_HOUR", true},
		{"", false},
		{"INVALID", false},
		{"prepaid", false},
	}

	for _, tt := range tests {
		got := isValidInstanceChargeType(tt.ct)
		if got != tt.want {
			t.Errorf("isValidInstanceChargeType(%q)=%v, want %v", tt.ct, got, tt.want)
		}
	}
}

// ──────────────────────────────────────────────
// instance type 校验
// ──────────────────────────────────────────────

func TestIsValidInstanceType(t *testing.T) {
	validTypes := []string{"Ai2.MEDIUM2", "Ai2.MEDIUM4", "Ai2.LARGE8"}
	for _, it := range validTypes {
		if !isValidInstanceType(it) {
			t.Errorf("isValidInstanceType(%q)=false, want true", it)
		}
	}

	invalidTypes := []string{"", "S5.LARGE8", "Ai2.small"}
	for _, it := range invalidTypes {
		if isValidInstanceType(it) {
			t.Errorf("isValidInstanceType(%q)=true, want false", it)
		}
	}
}

// ──────────────────────────────────────────────

// ──────────────────────────────────────────────
// SDK component CBS constant
// ──────────────────────────────────────────────

func TestSDKComponentCBSConstant(t *testing.T) {
	if SDKComponentCBS != "cbs" {
		t.Errorf("SDKComponentCBS=%q, want cbs", SDKComponentCBS)
	}
}

// ──────────────────────────────────────────────
// 确保 model.AllowedInstanceTypes 满足测试假设
// ──────────────────────────────────────────────

func TestAllowedInstanceTypesContract(t *testing.T) {
	if len(model.AllowedInstanceTypes) == 0 {
		t.Fatal("model.AllowedInstanceTypes is empty — tests assume non-empty")
	}
	for _, it := range model.AllowedInstanceTypes {
		if !isValidInstanceType(it) {
			t.Errorf("AllowedInstanceType %q should be valid", it)
		}
	}
}

func TestResourceOptionsHandlersRejectNonGET(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		handler http.HandlerFunc
	}{
		{name: "instance types", path: "/admin/resource-policies/options/instance-types", handler: HandleResourceOptionsInstanceTypes},
		{name: "system disks", path: "/admin/resource-policies/options/system-disks", handler: HandleResourceOptionsSystemDisks},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			rr := httptest.NewRecorder()
			tt.handler(rr, req)
			if rr.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status=%d, want 405; body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestResourceOptionsHandlersServeCacheBeforeConstructingClients(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	originalToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = originalToken }()

	originalGetCVM := getCVMOptionsClientFn
	originalGetCBS := getCBSOptionsClientFn
	cvmCalls, cbsCalls := 0, 0
	getCVMOptionsClientFn = func(context.Context) (cvmOptionsClient, error) {
		cvmCalls++
		return nil, fmt.Errorf("client must not be constructed on cache hit")
	}
	getCBSOptionsClientFn = func(context.Context) (cbsOptionsClient, error) {
		cbsCalls++
		return nil, fmt.Errorf("client must not be constructed on cache hit")
	}
	defer func() {
		getCVMOptionsClientFn = originalGetCVM
		getCBSOptionsClientFn = originalGetCBS
	}()

	ctx := context.Background()
	instanceData, _ := json.Marshal([]instanceTypeItem{{InstanceType: "Ai2.MEDIUM2"}})
	optionsCacheSet(resourceOptionsCacheKey(ctx, "instance-types", "ap-guangzhou-1", ""), &optionsCachePayload{
		Data:        instanceData,
		RefreshedAt: time.Now().UTC().Format(time.RFC3339),
	})
	diskData, _ := json.Marshal([]diskOption{{DiskType: "CLOUD_SSD", MinDiskSize: 50}})
	optionsCacheSet(resourceOptionsCacheKey(ctx, "system-disks", "ap-guangzhou-1", "", "Ai2.MEDIUM2"), &optionsCachePayload{
		Data:        diskData,
		RefreshedAt: time.Now().UTC().Format(time.RFC3339),
	})

	requests := []struct {
		path    string
		handler http.HandlerFunc
	}{
		{
			path:    "/admin/resource-policies/options/instance-types?zone=ap-guangzhou-1",
			handler: HandleResourceOptionsInstanceTypes,
		},
		{
			path:    "/admin/resource-policies/options/system-disks?zone=ap-guangzhou-1&instance_type=Ai2.MEDIUM2",
			handler: HandleResourceOptionsSystemDisks,
		},
	}
	for _, tt := range requests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		req.Header.Set("Authorization", "Bearer test-admin-token")
		rr := httptest.NewRecorder()
		tt.handler(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status=%d, want 200; body=%s", tt.path, rr.Code, rr.Body.String())
		}
	}
	if cvmCalls != 0 || cbsCalls != 0 {
		t.Fatalf("cache hits constructed clients: cvm=%d cbs=%d", cvmCalls, cbsCalls)
	}
}

func TestResourceOptionsSystemDisksRequiresSELLInstance(t *testing.T) {
	setupResourceOptionsTest(t)
	defer teardownResourceOptionsTest(t)

	mockCVM := mockCVMClientWith(func(*cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error) {
		return &cvm.DescribeZoneInstanceConfigInfosResponse{
			Response: &cvm.DescribeZoneInstanceConfigInfosResponseParams{
				InstanceTypeQuotaSet: []*cvm.InstanceTypeQuotaItem{{
					Status:         sdkcommon.StringPtr("OUT_OF_STOCK"),
					InstanceType:   sdkcommon.StringPtr("Ai2.MEDIUM2"),
					InstanceFamily: sdkcommon.StringPtr("Ai2"),
					Cpu:            sdkcommon.Int64Ptr(2),
					Memory:         sdkcommon.Int64Ptr(4),
				}},
			},
		}, nil
	})
	cbsCalls := 0
	mockCBS := mockCBSClientWith(func(*cbs.DescribeDiskConfigQuotaRequest) (*cbs.DescribeDiskConfigQuotaResponse, error) {
		cbsCalls++
		return &cbs.DescribeDiskConfigQuotaResponse{}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/options/system-disks?zone=ap-guangzhou-1&instance_type=Ai2.MEDIUM2", nil)
	rr := httptest.NewRecorder()
	handleResourceOptionsSystemDisks(context.Background(), rr, req, mockCVM, mockCBS,
		"ap-guangzhou-1", "", "Ai2.MEDIUM2", false)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500; body=%s", rr.Code, rr.Body.String())
	}
	if cbsCalls != 0 {
		t.Fatalf("CBS called %d times for a non-SELL instance, want 0", cbsCalls)
	}
}
