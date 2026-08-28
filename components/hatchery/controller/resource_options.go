package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	cbs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs/v20170312"
	sdkcommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// ──────────────────────────────────────────────
// 缓存（带 TTL）
// ──────────────────────────────────────────────

var resourceOptionsCache sync.Map

type optionsCachePayload struct {
	Data        json.RawMessage `json:"data"`
	RefreshedAt string          `json:"refreshed_at"`
}

type optionsCacheEntry struct {
	Payload   *optionsCachePayload
	ExpiresAt time.Time
}

const optionsCacheTTL = 5 * time.Minute

func optionsCacheGet(key string) (*optionsCachePayload, bool) {
	v, ok := resourceOptionsCache.Load(key)
	if !ok {
		return nil, false
	}
	entry := v.(*optionsCacheEntry)
	if time.Now().After(entry.ExpiresAt) {
		resourceOptionsCache.Delete(key)
		return nil, false
	}
	return entry.Payload, true
}

func optionsCacheSet(key string, payload *optionsCachePayload) {
	resourceOptionsCache.Store(key, &optionsCacheEntry{
		Payload:   payload,
		ExpiresAt: time.Now().Add(optionsCacheTTL),
	})
}

// ──────────────────────────────────────────────
// Inflight 去重
// ──────────────────────────────────────────────

var inflightMap sync.Map

func inflightDedup(key string) (bool, func()) {
	ch := make(chan struct{})
	if actual, loaded := inflightMap.LoadOrStore(key, ch); loaded {
		<-actual.(chan struct{})
		return false, nil
	}
	done := func() {
		close(ch)
		inflightMap.CompareAndDelete(key, ch)
	}
	return true, done
}

// ──────────────────────────────────────────────
// 共享 cache key helper
// ──────────────────────────────────────────────

// resourceOptionsCacheKey 构建租户隔离的缓存键。
// 包含 tenant identifier、region、endpoint 和所有 scope 参数。
func resourceOptionsCacheKey(ctx context.Context, endpoint string, scope ...string) string {
	parts := []string{endpoint, model.CurrentIdentifier(ctx), CVMRegion}
	parts = append(parts, scope...)
	return strings.Join(parts, ":")
}

// ──────────────────────────────────────────────
// charge type / instance type 校验
// ──────────────────────────────────────────────

var validInstanceChargeTypes = map[string]bool{
	"PREPAID":          true,
	"POSTPAID_BY_HOUR": true,
}

func isValidInstanceChargeType(ct string) bool {
	return validInstanceChargeTypes[ct]
}

func isValidInstanceType(it string) bool {
	return slices.Contains(model.AllowedInstanceTypes, it)
}

// ──────────────────────────────────────────────
// SDK client interfaces（依赖注入）
// ──────────────────────────────────────────────

type cvmOptionsClient interface {
	DescribeZoneInstanceConfigInfos(request *cvm.DescribeZoneInstanceConfigInfosRequest) (*cvm.DescribeZoneInstanceConfigInfosResponse, error)
}

type cbsOptionsClient interface {
	DescribeDiskConfigQuota(request *cbs.DescribeDiskConfigQuotaRequest) (*cbs.DescribeDiskConfigQuotaResponse, error)
}

var getCBSOptionsClientFn = func(ctx context.Context) (cbsOptionsClient, error) {
	return GetCBSClient(ctx)
}

// ──────────────────────────────────────────────
// 响应构建
// ──────────────────────────────────────────────

func buildInstanceTypesResponse(source string, payload *optionsCachePayload) map[string]interface{} {
	var items []instanceTypeItem
	if payload != nil && payload.Data != nil {
		json.Unmarshal(payload.Data, &items)
	}
	refreshedAt := ""
	if payload != nil {
		refreshedAt = payload.RefreshedAt
	}
	return map[string]interface{}{
		"ok":             true,
		"source":         source,
		"refreshed_at":   refreshedAt,
		"instance_types": items,
	}
}

func buildSystemDisksResponse(source string, payload *optionsCachePayload) map[string]interface{} {
	var items []diskOption
	if payload != nil && payload.Data != nil {
		json.Unmarshal(payload.Data, &items)
	}
	refreshedAt := ""
	if payload != nil {
		refreshedAt = payload.RefreshedAt
	}
	return map[string]interface{}{
		"ok":                  true,
		"source":              source,
		"refreshed_at":        refreshedAt,
		"system_disk_options": items,
	}
}

// ──────────────────────────────────────────────
// 共享 cache/inflight 辅助逻辑
// ──────────────────────────────────────────────

// cacheOrInflight 封装 cache hit / inflight waiter 逻辑。
// 返回 (proceed, cleanup)。proceed 为 true 表示当前请求为 winner，应执行云调用；
// cleanup 必须在云调用完成后调用（defer cleanup()），用于释放 inflight 锁。
func cacheOrInflight(w http.ResponseWriter, r *http.Request, cacheKey string, refresh bool, responseBuilder func(string, *optionsCachePayload) map[string]interface{}) (bool, func()) {
	if !refresh {
		if payload, ok := optionsCacheGet(cacheKey); ok {
			jsonOK(w, responseBuilder("cache", payload))
			return false, nil
		}
	}

	winner, done := inflightDedup(cacheKey)
	if !winner {
		if payload, ok := optionsCacheGet(cacheKey); ok {
			jsonOK(w, responseBuilder("cache", payload))
		} else {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryCloudInstanceTypesFailed))
		}
		return false, nil
	}
	return true, done
}

// ──────────────────────────────────────────────
// GET /admin/resource-policies/options/instance-types?zone=X&instance_charge_type=Y&refresh=1
// ──────────────────────────────────────────────

type instanceTypeItem struct {
	InstanceType string  `json:"instance_type"`
	CPU          int64   `json:"cpu"`
	Memory       int64   `json:"memory"`
	UnitPrice    float64 `json:"unit_price,omitempty"`
}

func HandleResourceOptionsInstanceTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	zone := r.URL.Query().Get("zone")
	chargeType := r.URL.Query().Get("instance_charge_type")
	refresh := r.URL.Query().Get("refresh") == "1"

	if zone == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "zone"))
		return
	}
	if chargeType != "" && !isValidInstanceChargeType(chargeType) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "instance_charge_type"))
		return
	}

	handleResourceOptionsInstanceTypes(r.Context(), w, r, nil, zone, chargeType, refresh)
}

func handleResourceOptionsInstanceTypes(ctx context.Context, w http.ResponseWriter, r *http.Request, cvmClient cvmOptionsClient, zone, chargeType string, refresh bool) {
	if zone == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "zone"))
		return
	}
	if chargeType != "" && !isValidInstanceChargeType(chargeType) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "instance_charge_type"))
		return
	}

	cacheKey := resourceOptionsCacheKey(ctx, "instance-types", zone, chargeType)

	proceed, cleanup := cacheOrInflight(w, r, cacheKey, refresh, buildInstanceTypesResponse)
	if !proceed {
		return
	}
	defer cleanup()
	if cvmClient == nil {
		var err error
		cvmClient, err = getCVMOptionsClientFn(ctx)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryCloudInstanceTypesCVMFailed))
			return
		}
	}

	req := cvm.NewDescribeZoneInstanceConfigInfosRequest()
	filters := []*cvm.Filter{
		{Name: sdkcommon.StringPtr("zone"), Values: sdkcommon.StringPtrs([]string{zone})},
	}
	if chargeType != "" {
		filters = append(filters, &cvm.Filter{
			Name: sdkcommon.StringPtr("instance-charge-type"), Values: sdkcommon.StringPtrs([]string{chargeType}),
		})
	}
	req.Filters = filters

	resp, err := CallSDKAPITyped(ctx, SDKComponentCVM, req, cvmClient.DescribeZoneInstanceConfigInfos)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCloudInstanceTypesFailed))
		return
	}

	items := make([]instanceTypeItem, 0)
	if resp.Response != nil {
		for _, info := range resp.Response.InstanceTypeQuotaSet {
			if info.Status == nil || *info.Status != "SELL" {
				continue
			}
			if info.InstanceType == nil || !isValidInstanceType(*info.InstanceType) {
				continue
			}
			item := instanceTypeItem{}
			if info.InstanceType != nil {
				item.InstanceType = *info.InstanceType
			}
			if info.Cpu != nil {
				item.CPU = *info.Cpu
			}
			if info.Memory != nil {
				item.Memory = *info.Memory
			}
			if info.Price != nil && info.Price.UnitPrice != nil {
				item.UnitPrice = *info.Price.UnitPrice
			}
			items = append(items, item)
		}
	}

	data, _ := json.Marshal(items)
	payload := &optionsCachePayload{
		Data:        data,
		RefreshedAt: time.Now().UTC().Format(time.RFC3339),
	}
	optionsCacheSet(cacheKey, payload)
	jsonOK(w, buildInstanceTypesResponse("tencent_cloud", payload))
}

// ──────────────────────────────────────────────
// GET /admin/resource-policies/options/system-disks?zone=X&instance_charge_type=Y&instance_type=Z&refresh=1
// ──────────────────────────────────────────────

type diskOption struct {
	DiskType    string `json:"disk_type"`
	MinDiskSize int64  `json:"min_disk_size"`
	MaxDiskSize int64  `json:"max_disk_size"`
	StepSize    int64  `json:"step_size"`
}

func HandleResourceOptionsSystemDisks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	zone := r.URL.Query().Get("zone")
	chargeType := r.URL.Query().Get("instance_charge_type")
	instanceType := r.URL.Query().Get("instance_type")
	refresh := r.URL.Query().Get("refresh") == "1"

	if zone == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "zone"))
		return
	}
	if instanceType == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "instance_type"))
		return
	}
	if chargeType != "" && !isValidInstanceChargeType(chargeType) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "instance_charge_type"))
		return
	}
	if !isValidInstanceType(instanceType) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "instance_type"))
		return
	}

	handleResourceOptionsSystemDisks(r.Context(), w, r, nil, nil, zone, chargeType, instanceType, refresh)
}
func handleResourceOptionsSystemDisks(ctx context.Context, w http.ResponseWriter, r *http.Request, cvmClient cvmOptionsClient, cbsClient cbsOptionsClient, zone, chargeType, instanceType string, refresh bool) {
	if zone == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "zone"))
		return
	}
	if instanceType == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "instance_type"))
		return
	}
	if chargeType != "" && !isValidInstanceChargeType(chargeType) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "instance_charge_type"))
		return
	}
	if !isValidInstanceType(instanceType) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "instance_type"))
		return
	}

	cacheKey := resourceOptionsCacheKey(ctx, "system-disks", zone, chargeType, instanceType)

	proceed, cleanup := cacheOrInflight(w, r, cacheKey, refresh, buildSystemDisksResponse)
	if !proceed {
		return
	}
	defer cleanup()
	if cvmClient == nil {
		var err error
		cvmClient, err = getCVMOptionsClientFn(ctx)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryCloudInstanceTypesCVMFailed))
			return
		}
	}

	// Step 1: 查询 instance family / cpu / memory
	cvmReq := cvm.NewDescribeZoneInstanceConfigInfosRequest()
	cvmFilters := []*cvm.Filter{
		{Name: sdkcommon.StringPtr("zone"), Values: sdkcommon.StringPtrs([]string{zone})},
		{Name: sdkcommon.StringPtr("instance-type"), Values: sdkcommon.StringPtrs([]string{instanceType})},
	}
	if chargeType != "" {
		cvmFilters = append(cvmFilters, &cvm.Filter{
			Name: sdkcommon.StringPtr("instance-charge-type"), Values: sdkcommon.StringPtrs([]string{chargeType}),
		})
	}
	cvmReq.Filters = cvmFilters

	cvmResp, err := CallSDKAPITyped(ctx, SDKComponentCVM, cvmReq, cvmClient.DescribeZoneInstanceConfigInfos)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCloudInstanceTypesFailed))
		return
	}

	var instanceFamily string
	var cpu, memory int64
	if cvmResp.Response != nil {
		for _, info := range cvmResp.Response.InstanceTypeQuotaSet {
			if info == nil ||
				info.Status == nil || *info.Status != "SELL" ||
				info.InstanceType == nil || *info.InstanceType != instanceType {
				continue
			}
			if info.InstanceFamily != nil {
				instanceFamily = *info.InstanceFamily
			}
			if info.Cpu != nil {
				cpu = *info.Cpu
			}
			if info.Memory != nil {
				memory = *info.Memory
			}
			break
		}
	}

	if instanceFamily == "" {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInstanceTypeNotAvailable, instanceType, ""))
		return
	}

	// Step 2: 仅 PREPAID / POSTPAID_BY_HOUR 调用 DescribeDiskConfigQuota
	if chargeType != "" && chargeType != "PREPAID" && chargeType != "POSTPAID_BY_HOUR" {
		payload := &optionsCachePayload{
			Data:        json.RawMessage("[]"),
			RefreshedAt: time.Now().UTC().Format(time.RFC3339),
		}
		optionsCacheSet(cacheKey, payload)
		jsonOK(w, buildSystemDisksResponse("tencent_cloud", payload))
		return
	}

	diskChargeType := "POSTPAID_BY_HOUR"
	if chargeType == "PREPAID" {
		diskChargeType = "PREPAID"
	}
	if cbsClient == nil {
		var err error
		cbsClient, err = getCBSOptionsClientFn(ctx)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryCloudInstanceTypesFailed))
			return
		}
	}

	cbsReq := cbs.NewDescribeDiskConfigQuotaRequest()
	cbsReq.InquiryType = sdkcommon.StringPtr("INQUIRY_CVM_CONFIG")
	cbsReq.DiskChargeType = sdkcommon.StringPtr(diskChargeType)
	cbsReq.Zones = sdkcommon.StringPtrs([]string{zone})
	cbsReq.DiskUsage = sdkcommon.StringPtr("SYSTEM_DISK")
	cbsReq.InstanceFamilies = sdkcommon.StringPtrs([]string{instanceFamily})
	cbsReq.CPU = sdkcommon.Uint64Ptr(uint64(cpu))
	cbsReq.Memory = sdkcommon.Uint64Ptr(uint64(memory))

	cbsResp, err := CallSDKAPITyped(ctx, SDKComponentCBS, cbsReq, cbsClient.DescribeDiskConfigQuota)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCloudInstanceTypesFailed))
		return
	}

	diskOptions := make([]diskOption, 0)
	if cbsResp.Response != nil {
		for _, config := range cbsResp.Response.DiskConfigSet {
			if config.Available == nil || !*config.Available {
				continue
			}
			if config.DiskType == nil {
				continue
			}
			if err := model.ValidateDiskType(*config.DiskType); err != nil {
				continue
			}
			opt := diskOption{DiskType: *config.DiskType}
			if config.MinDiskSize != nil {
				opt.MinDiskSize = int64(*config.MinDiskSize)
			}
			if config.MaxDiskSize != nil {
				opt.MaxDiskSize = int64(*config.MaxDiskSize)
			}
			if opt.MaxDiskSize > 0 && opt.MaxDiskSize < opt.MinDiskSize {
				continue
			}
			if config.StepSize != nil {
				opt.StepSize = int64(*config.StepSize)
			}
			diskOptions = append(diskOptions, opt)
		}
	}

	data, _ := json.Marshal(diskOptions)
	payload := &optionsCachePayload{
		Data:        data,
		RefreshedAt: time.Now().UTC().Format(time.RFC3339),
	}
	optionsCacheSet(cacheKey, payload)
	jsonOK(w, buildSystemDisksResponse("tencent_cloud", payload))
}
